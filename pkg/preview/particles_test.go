package preview

import (
	"encoding/binary"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"aoe2kit/pkg/datfile"
)

func particleFixture(t *testing.T, root, layer string) *particleLoader {
	t.Helper()
	l := newParticleLoader(root)
	if err := os.MkdirAll(filepath.Join(l.root, "textures"), 0700); err != nil {
		t.Fatal(err)
	}
	img := image.NewRGBA(image.Rect(0, 0, 8, 2))
	colors := []color.RGBA{{G: 255, A: 255}, {R: 255, G: 255, A: 255}, {B: 255, A: 255}, {R: 255, B: 255, A: 255}}
	var atlas particleAtlas
	for i, c := range colors {
		for y := 0; y < 2; y++ {
			for x := 2 * i; x < 2*i+2; x++ {
				img.SetRGBA(x, y, c)
			}
		}
		atlas.Frames = append(atlas.Frames, particleAtlasFrame{Filename: "synthetic", Frame: particleRect{X: 2 * i, W: 2, H: 2}, SpriteSourceSize: particleRect{W: 2, H: 2}, SourceSize: particleSize{2, 2}, Pivot: &particlePivot{.5, .5}})
	}
	f, err := os.Create(filepath.Join(l.root, "textures", "atlas.png"))
	if err != nil {
		t.Fatal(err)
	}
	err = png.Encode(f, img)
	f.Close()
	if err != nil {
		t.Fatal(err)
	}
	writeParticleFixtureJSON(t, filepath.Join(l.root, "textures", "atlas.json"), atlas)
	writeParticleFixtureJSON(t, filepath.Join(l.root, "test.json"), particleDefinition{AtlasFile: "textures\\atlas.png", Type: "Loop", ImageCount: 4, ImageAngles: 2, Scale: 1, Layer: layer})
	return l
}

func writeParticleFixtureJSON(t *testing.T, path string, v any) {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(path, b, 0600); err != nil {
		t.Fatal(err)
	}
}

func TestParticleAngleAndTrimmedPivot(t *testing.T) {
	l := particleFixture(t, t.TempDir(), "")
	d, err := l.definition("test")
	if err != nil {
		t.Fatal(err)
	}
	f, err := l.frame(d, 1)
	if err != nil {
		t.Fatal(err)
	}
	if f.ordinal != 2 || f.image.RGBAAt(0, 0) != (color.RGBA{B: 255, A: 255}) {
		t.Fatalf("angle frame: %+v", f)
	}
	a := l.atlases["textures\\atlas.json"]
	a.Frames[2].SourceSize = particleSize{W: 8, H: 6}
	a.Frames[2].SpriteSourceSize.X = 3
	a.Frames[2].SpriteSourceSize.Y = 1
	l.atlases["textures\\atlas.json"] = a
	clear(l.frames)
	f, err = l.frame(d, 1)
	if err != nil {
		t.Fatal(err)
	}
	if f.hotspotX != 1 || f.hotspotY != 2 {
		t.Fatalf("trimmed hotspot: %v,%v", f.hotspotX, f.hotspotY)
	}
	if _, err = l.frame(d, 2); err == nil {
		t.Fatal("wrapped invalid angle")
	}
	bad := d
	bad.ImageCount = 8
	bad.ImageAngles = 4 // selects the already cached ordinal 2
	if _, err = l.frame(bad, 1); err == nil {
		t.Fatal("cached frame bypassed animation range validation")
	}
	d.ImageFirst = 1
	if _, err = l.frame(d, 0); err == nil {
		t.Fatal("accepted out-of-range animation extent")
	}
}

func TestParticleRejectsInvalidAssets(t *testing.T) {
	l := particleFixture(t, t.TempDir(), "")
	for _, path := range []string{"../atlas.png", "/atlas.png", "C:\\atlas.png", "textures/../../atlas.png"} {
		if _, err := particleAssetPath(l.root, path); err == nil {
			t.Fatalf("accepted %q", path)
		}
	}
	outside := filepath.Join(t.TempDir(), "outside.json")
	writeParticleFixtureJSON(t, outside, map[string]int{})
	if err := os.Symlink(outside, filepath.Join(l.root, "escape.json")); err == nil {
		if _, err = particleAssetPath(l.root, "escape.json"); err == nil {
			t.Fatal("accepted escaping symlink")
		}
	}
	d, _ := l.definition("test")
	for _, change := range []func(*particleDefinition){func(d *particleDefinition) { d.Type = "Emitter" }, func(d *particleDefinition) { d.ImageAngles = 3 }, func(d *particleDefinition) { d.Scale = 0 }, func(d *particleDefinition) { d.Layer = "Unrecognized" }} {
		bad := d
		change(&bad)
		writeParticleFixtureJSON(t, filepath.Join(l.root, "bad.json"), bad)
		if _, err := l.definition("bad"); err == nil {
			t.Fatal("accepted invalid definition")
		}
	}
	for _, change := range []func(*particleAtlasFrame){func(f *particleAtlasFrame) { f.Pivot = nil }, func(f *particleAtlasFrame) { f.Frame.X = 9 }, func(f *particleAtlasFrame) { f.SpriteSourceSize.X = 3 }} {
		var a particleAtlas
		path := filepath.Join(l.root, "textures", "atlas.json")
		if err := readParticleJSON(path, &a); err != nil {
			t.Fatal(err)
		}
		change(&a.Frames[0])
		l.atlases["textures\\atlas.json"] = a
		clear(l.frames)
		if _, err := l.frame(d, 0); err == nil {
			t.Fatal("accepted invalid frame metadata")
		}
	}
}

func TestParticleRotatedAtlasFrame(t *testing.T) {
	l := particleFixture(t, t.TempDir(), "")
	d, _ := l.definition("test")
	if _, err := l.frame(d, 0); err != nil {
		t.Fatal(err)
	}
	meta := "textures\\atlas.json"
	a := l.atlases[meta]
	a.Frames[0] = particleAtlasFrame{Rotated: true, Frame: particleRect{X: 5, W: 2, H: 3}, SpriteSourceSize: particleRect{X: 1, Y: 2, W: 2, H: 3}, SourceSize: particleSize{4, 6}, Pivot: &particlePivot{.5, .5}}
	l.atlases[meta] = a
	clear(l.frames)
	path, err := particleAssetPath(l.root, d.AtlasFile)
	if err != nil {
		t.Fatal(err)
	}
	img := l.textures[path]
	// Clockwise packed 2x3 image: original rows [1,2], [3,4], [5,6].
	for y, row := range [][]uint8{{5, 3, 1}, {6, 4, 2}} {
		for x, value := range row {
			img.SetRGBA(5+x, y, color.RGBA{R: value, A: 255})
		}
	}
	f, err := l.frame(d, 0)
	if err != nil {
		t.Fatal(err)
	}
	if f.image.Bounds() != image.Rect(0, 0, 2, 3) || f.hotspotX != 1 || f.hotspotY != 1 {
		t.Fatalf("rotated frame dimensions/pivot: %+v", f)
	}
	for y := 0; y < 3; y++ {
		for x := 0; x < 2; x++ {
			if got := f.image.RGBAAt(x, y).R; got != uint8(y*2+x+1) {
				t.Fatalf("pixel (%d,%d): %d", x, y, got)
			}
		}
	}
	a.Frames[0].Frame.X = 6 // unrotated width fits, packed width does not
	l.atlases[meta] = a
	clear(l.frames)
	if _, err := l.frame(d, 0); err == nil {
		t.Fatal("accepted rotated packed footprint outside atlas")
	}
}

func TestParticleDDSAtlas(t *testing.T) {
	l := particleFixture(t, t.TempDir(), "")
	b := make([]byte, 128)
	copy(b, "DDS ")
	binary.LittleEndian.PutUint32(b[4:], 124)
	binary.LittleEndian.PutUint32(b[12:], 4)
	binary.LittleEndian.PutUint32(b[16:], 4)
	binary.LittleEndian.PutUint32(b[76:], 32)
	binary.LittleEndian.PutUint32(b[80:], 4)
	copy(b[84:], "DXT1")
	b = append(b, 0, 248, 0, 0, 0, 0, 0, 0)
	path := filepath.Join(l.root, "test.dds")
	if err := os.WriteFile(path, b, 0600); err != nil {
		t.Fatal(err)
	}
	img, err := l.texture(path)
	if err != nil {
		t.Fatal(err)
	}
	if img.RGBAAt(0, 0).R != 255 {
		t.Fatal("DDS not decoded")
	}
}

func TestParticleTerrainAndSpriteOrder(t *testing.T) {
	for _, layer := range []string{"", "Terrain"} {
		t.Run(layer, func(t *testing.T) {
			root, idx := spriteFixture(t)
			particleFixture(t, root, layer)
			idx.Graphics = append(idx.Graphics, datfile.Graphic{Present: true, ParticleEffectName: datfile.DebugString{Value: "test"}})
			idx.Civs[0].Units = append(idx.Civs[0].Units, datfile.UnitSummary{Present: true, Index: 8, ID: 8, StandingGraphic1: 1})
			s := spriteSnapshot()
			s.Objects = append(s.Objects, Object{UnitID: 8, X: 5, Y: 5, Variant: 1})
			for _, shadows := range []*ShadowOptions{nil, {}} {
				im, r, err := RenderSprites(s, Options{Scale: 4, SpriteTilePixels: 4, Shadows: shadows}, idx, root)
				if err != nil {
					t.Fatal(err)
				}
				x, y := mapPoint(5, 5, Crop{X2: 10, Y2: 10}, 4, "diamond")
				want := color.RGBA{B: 255, A: 255}
				if layer == "Terrain" {
					want = color.RGBA{R: 255, A: 255}
				}
				if im.RGBAAt(x-1, y-1) != want {
					t.Fatalf("layer %s: got %v want %v", layer, im.RGBAAt(x-1, y-1), want)
				}
				if r.Drawn != 2 || len(r.Particles) != 1 || r.Particles[0].Status != "drawn" || r.Particles[0].Frame != 2 {
					t.Fatalf("report %+v", r)
				}
			}
			s.Objects[1].Variant = 7
			_, r, err := RenderSprites(s, Options{}, idx, root)
			if err != nil || r.Particles[0].Status != "skipped" || r.Drawn != 1 {
				t.Fatalf("invalid angle report %+v %v", r, err)
			}
			idx.Graphics[1].FileName = datfile.DebugString{Value: "synthetic"}
			s.Objects[1].Variant = 0
			_, r, err = RenderSprites(s, Options{}, idx, root)
			if err != nil || r.Drawn != 2 || r.Particles[0].Status != "skipped" {
				t.Fatalf("existing hybrid sprite lost: %+v %v", r, err)
			}
		})
	}
}
