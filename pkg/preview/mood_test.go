package preview

import (
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func neutralTestMood(name string) Mood {
	return Mood{Name: name, Brightness: 1, Contrast: 1, Saturation: 1, TerrainTint: [3]float64{1, 1, 1}, BloomTint: [3]float64{1, 1, 1}}
}

func TestMoodInverseAndHighlight(t *testing.T) {
	m := neutralTestMood("Default")
	m.Brightness = .95
	m.Contrast = 1.01
	m.Saturation = .99
	m.BloomIntensity = .6
	m.BloomThreshold = .25
	p := CurrentMoodModel()
	for _, c := range [][3]float64{{.1, .2, .3}, {.7, .6, .2}, {.6, .7, .8}} {
		back := p.ungrade(p.grade(c, m), m)
		for k := range c {
			if math.Abs(back[k]-c[k]) > 0.001 {
				t.Fatalf("inverse: %v -> %v", c, back)
			}
		}
	}
	p = legacyMoodModel()
	p.Highlight = 1
	a := p.grade([3]float64{1, 1, 1}, neutralTestMood("test"))
	for _, v := range a {
		if v != 1 {
			t.Fatal("highlight changed without bloom")
		}
	}
}

func TestMoodModelApplicationAndValidation(t *testing.T) {
	im := image.NewRGBA(image.Rect(0, 0, 5, 5))
	im.SetRGBA(2, 2, color.RGBA{100, 120, 140, 255})
	sub := im.SubImage(image.Rect(1, 1, 4, 4)).(*image.RGBA)
	m := neutralTestMood("test")
	m.TintFactor = .2
	m.TintColor = [3]float64{.8, .1, .2}
	p := legacyMoodModel()
	p.Tint = .5
	r, err := ApplyMoodModel(sub, m, "test", p)
	if err != nil {
		t.Fatal(err)
	}
	want := p.grade([3]float64{100.0 / 255, 120.0 / 255, 140.0 / 255}, m)
	got := im.RGBAAt(2, 2)
	if got.R != uint8(math.Round(want[0]*255)) || got.G != uint8(math.Round(want[1]*255)) || got.B != uint8(math.Round(want[2]*255)) || r.Rule == "" {
		t.Fatal("selected model was not applied")
	}
	p.Bloom = math.Inf(1)
	before := append([]byte(nil), im.Pix...)
	if _, err = ApplyMoodModel(sub, m, "test", p); err == nil || !reflect.DeepEqual(before, im.Pix) {
		t.Fatal("invalid model mutated image")
	}
	path := filepath.Join(t.TempDir(), "model.json")
	b, err := json.Marshal(map[string]any{"model": legacyMoodModel()})
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(path, b, 0600); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadMoodModel(path)
	if err != nil || loaded != legacyMoodModel() {
		t.Fatal(loaded, err)
	}
}

func TestMoodFitHoldoutDoesNotAffectSelection(t *testing.T) {
	m := neutralTestMood("Train")
	m.TintFactor = .15
	m.TintColor = [3]float64{.2, .3, .5}
	m.BloomIntensity = .6
	m.BloomThreshold = .25
	c := MoodFitCase{Name: "training", Mood: m}
	p := legacyMoodModel()
	p.Tint = .8
	for i := 1; i <= 8; i++ {
		x := [3]float64{float64(i) / 12, .3, float64(9-i) / 12}
		c.Samples = append(c.Samples, MoodSample{Input: x, Capture: p.grade(x, m)})
	}
	h := c
	h.Name = "held"
	h.Mood.Name = "Held"
	h.Samples = append([]MoodSample(nil), c.Samples...)
	a, as, err := FitMoodModel([]MoodFitCase{c, h}, "HELD")
	if err != nil {
		t.Fatal(err)
	}
	for i := range h.Samples {
		h.Samples[i].Capture = [3]float64{1, 0, 1}
	}
	b, bs, err := FitMoodModel([]MoodFitCase{c, h}, "Held")
	if err != nil {
		t.Fatal(err)
	}
	if a != b || !reflect.DeepEqual(as, bs) {
		t.Fatal("held-out captures affected fit")
	}
	if moodError(a, c) >= moodError(legacyMoodModel(), c) {
		t.Fatal("fit did not improve synthetic observations")
	}
	if _, _, err = FitMoodModel([]MoodFitCase{c}, "absent"); err == nil {
		t.Fatal("missing held-out mood accepted")
	}
	c.Samples[0].Input[0] = math.NaN()
	if err = ValidateMoodFitCases([]MoodFitCase{c}); err == nil {
		t.Fatal("NaN accepted")
	}
}

func TestMoodAlignment(t *testing.T) {
	a := image.NewRGBA(image.Rect(0, 0, 192, 144))
	b := image.NewRGBA(a.Bounds())
	// Non-periodic block texture makes coarse alignment identifiable, even
	// when the target is darker and has a different colour balance.
	for y := 0; y < 144; y++ {
		for x := 0; x < 192; x++ {
			v := uint8(40 + ((x/9*71 + y/7*131 + (x/9)*(y/7)*17) % 160))
			a.SetRGBA(x, y, color.RGBA{v, v, v, 255})
			if x+13 < 192 && y+7 < 144 {
				b.SetRGBA(x+13, y+7, color.RGBA{v/2 + 10, v/2 + 20, v/2 + 30, 255})
			}
		}
	}
	shift, c, err := AlignMoodImages(a, b)
	if err != nil || shift != [2]int{13, 7} || c < .99 {
		t.Fatalf("alignment %v %.4f %v", shift, c, err)
	}
}

func TestMoodCalibrationRegionsAndTexture(t *testing.T) {
	root, _ := terrainTestInstall(t)
	m := neutralTestMood("Default")
	dir := filepath.Join(root, "resources", "_common", "terrain", "colorcorrection_json")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(map[string]any{"correction_types": []Mood{m}})
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(dir, "colorcorrection.json"), b, 0600); err != nil {
		t.Fatal(err)
	}
	im := image.NewRGBA(image.Rect(0, 0, 4, 4))
	im.SetRGBA(1, 1, color.RGBA{90, 100, 110, 255})
	im.SetRGBA(2, 2, color.RGBA{120, 130, 140, 255})
	path := filepath.Join(root, "sample.png")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err = png.Encode(f, im); err != nil {
		t.Fatal(err)
	}
	if err = f.Close(); err != nil {
		t.Fatal(err)
	}
	manifest := MoodCalibrationManifest{Cases: []MoodCalibrationCase{{Name: "patch", Preview: "sample.png", Capture: "sample.png", Mood: "Default", Offset: &[2]int{}, Regions: []MoodCalibrationRegion{{Name: "red", Points: [][2]int{{1, 1}, {2, 2}}, Texture: "resources/_common/terrain/textures/2x/base.dds"}}}}}
	write := func() {
		t.Helper()
		b, err := json.Marshal(manifest)
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(filepath.Join(root, "manifest.json"), b, 0600); err != nil {
			t.Fatal(err)
		}
	}
	write()
	cases, align, err := PrepareMoodCalibration(filepath.Join(root, "manifest.json"), root)
	if err != nil {
		t.Fatal(err)
	}
	if len(cases) != 1 || cases[0].Samples[0].Input[0] != 120.0/255 || align[0].CaptureSHA256 == "" {
		t.Fatalf("bad samples: %+v", cases)
	}
	stats := cases[0].Samples[0].Texture
	if stats == nil || stats.Pixels != 16 || stats.Median != [3]float64{1, 0, 0} {
		t.Fatalf("DDS stats: %+v", stats)
	}
	manifest.Cases[0].TextureInput = true
	write()
	cases, _, err = PrepareMoodCalibration(filepath.Join(root, "manifest.json"), root)
	if err != nil {
		t.Fatal(err)
	}
	if cases[0].Samples[0].Input != [3]float64{1, 0, 0} {
		t.Fatal("raw texture input not used")
	}
	manifest.Cases[0].Regions[0].Exclude = true
	write()
	if _, _, err = PrepareMoodCalibration(filepath.Join(root, "manifest.json"), root); err == nil {
		t.Fatal("empty training data accepted")
	}
}
