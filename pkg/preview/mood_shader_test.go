package preview

import (
	"math"
	"testing"
)

func closeShaderRGB(t *testing.T, got, want [3]float64) {
	t.Helper()
	for k := range got {
		if math.Abs(got[k]-want[k]) > 1e-10 {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

func TestShaderPostStageOrderAndClipping(t *testing.T) {
	m := neutralTestMood("synthetic")
	m.Saturation = 0
	m.Contrast = 2
	m.Brightness = .5
	m.TintFactor = .25
	m.TintColor = [3]float64{1, 0, .5}
	// Red becomes luma .2126; contrast clips it to 0 before brightness.
	closeShaderRGB(t, ShaderPostGrade([3]float64{1, 0, 0}, m, true, 1), [3]float64{.25, 0, .125})
	m.Saturation = 1
	// Contrast clips 1.5 to 1 before brightness: the result is .5, not .75.
	closeShaderRGB(t, ShaderPostGrade([3]float64{1, 1, 1}, m, true, 1), [3]float64{.625, .375, .5})
	// Gamma remains active when colour correction is off; it uses abs.
	closeShaderRGB(t, ShaderPostGrade([3]float64{-.5, .25, 2}, m, false, 2), [3]float64{.25, .0625, 1})
}

func TestShaderTerrainAndSpriteStages(t *testing.T) {
	m := neutralTestMood("synthetic")
	m.BloomTerrain = .5
	m.ShadowStrength = 2
	m.ShadowColor = [3]float64{.25, .5, .75}
	// Modulation is capped at .375, then doubled; terrain shadow is saturated.
	light := ShaderTerrainLight(.5, [3]float64{2, 2, 2}, [3]float64{.5, .5, .5}, [3]float64{.5, .5, .5})
	c := ShaderTerrainGrade([3]float64{1, 1, 1}, light, 1, m, true, true, 1)
	closeShaderRGB(t, c, [3]float64{.25, .5, .75})
	// Sprite shadow, unlike terrain, does not saturate strength*coverage.
	closeShaderRGB(t, ShaderSpriteCombine([3]float64{}, [3]float64{1, 1, 1}, 1, 1, true, m, true), [3]float64{-.125, .25, .625})
	closeShaderRGB(t, ShaderSpriteCombine(c, [3]float64{1, 1, 1}, .5, 1, false, m, true), [3]float64{.625, .75, .875})
	a := ShaderTerrainAttributes([2]float64{.6, .6}, [2]float64{.4, .7}, true)
	b := ShaderTerrainAttributes([2]float64{.6, .6}, [2]float64{.4, .7}, false)
	for i, w := range [4]float64{.65, .7, .4, .7} {
		if math.Abs(a[i]-w) > 1e-10 {
			t.Fatal(a)
		}
	}
	for i, w := range [4]float64{.7, 1, .4, .7} {
		if math.Abs(b[i]-w) > 1e-10 {
			t.Fatal(b)
		}
	}
}

func TestShaderBloomStages(t *testing.T) {
	m := neutralTestMood("synthetic")
	m.BloomScene = [3]float64{1, 2, 1}
	m.BloomThreshold = .25
	m.BloomSaturation = 1
	s := [4][3]float64{{1, 0, 0}, {0, 1, 0}, {0, 0, 1}, {1, 1, 1}}
	// Average first, then intensity*scene, then component thresholds.
	b := ShaderBloomExtract(s, m, 2)
	closeShaderRGB(t, b, [3]float64{.75, 1.75, .75})
	m.BloomSaturation = 0
	y := moodLuma(.75, 1.75, .75)
	closeShaderRGB(t, ShaderBloomExtract(s, m, 2), [3]float64{y, y, y})
	m.BloomTint = [3]float64{2, 1, .5}
	var taps [8][3]float64
	var weights [8]float64
	for i := range taps {
		taps[i] = b
		weights[i] = .25
	}
	// Sum 2 is not silently normalized to 1.
	filtered := ShaderBloomFilter(taps, weights)
	closeShaderRGB(t, ShaderBloomTint(filtered, m), [3]float64{3, 3.5, .75})
	closeShaderRGB(t, ShaderBloomCompose([3]float64{1, 1, 1}, b, true), [3]float64{1.75, 2.75, 1.75})
}

func TestShaderFogAndVignette(t *testing.T) {
	m := neutralTestMood("synthetic")
	m.FogColor = [3]float64{.2, .4, .6}
	m.DistanceFog = .5
	closeShaderRGB(t, ShaderDistanceFog([3]float64{.5, .5, .5}, .25, 1, 2, m), [3]float64{.65, .8, .95})
	m.VignetteBorder = [2]float64{1, 1}
	m.VignetteRadius = .5
	m.VignetteAmount = 1
	m.VignetteColor = [3]float64{1, 1, 1}
	closeShaderRGB(t, ShaderVignette([3]float64{.2, .2, .2}, [2]float64{.5, .5}, 1, 1, m), [3]float64{.2, .2, .2})
	closeShaderRGB(t, ShaderVignette([3]float64{.2, .2, .2}, [2]float64{0, 0}, 1, .5, m), [3]float64{.35, .35, .35})
}

func TestShaderAuditIsExplicitAndDoesNotReplaceDefault(t *testing.T) {
	o := ShaderMoodOptions{ColorCorrection: true, Gamma: 1, InputLight: [3]float64{1, 1, 1}}
	m := neutralTestMood("synthetic")
	m.ShaderFields = true
	c := MoodFitCase{Name: "test", Mood: m, Samples: []MoodSample{{Input: [3]float64{.3, .4, .5}, Capture: [3]float64{.3, .4, .5}}}}
	a, err := ShaderMoodResiduals([]MoodFitCase{c}, o, "synthetic")
	if err != nil || a.Results[0].After > 1e-10 || !a.Results[0].HeldOut {
		t.Fatal(a, err)
	}
	if CurrentMoodModel() != legacyMoodModel() {
		t.Fatal("shader audit promoted default")
	}
	c.Mood.ShaderFields = false
	if _, err = ShaderMoodResiduals([]MoodFitCase{c}, o, ""); err == nil {
		t.Fatal("accepted old sample report without shader fields")
	}
	o.Gamma = math.NaN()
	if o.Validate() == nil {
		t.Fatal("accepted NaN")
	}
}
