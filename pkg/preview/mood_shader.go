package preview

import (
	"fmt"
	"math"
	"strings"
)

// ShaderMoodOptions records engine state that cannot be recovered from the mood
// JSON alone. This audit models a uniform, fully visible terrain patch, not the
// engine's spatial buffers. Values must be supplied explicitly, never fitted
// into the recovered stage equations.
type ShaderMoodOptions struct {
	ColorCorrection bool       `json:"color_correction"`
	Bloom           bool       `json:"bloom"`
	BloomScale      float64    `json:"bloom_option_scale"`
	HorizontalSum   float64    `json:"horizontal_tap_sum"`
	VerticalSum     float64    `json:"vertical_tap_sum"`
	Gamma           float64    `json:"gamma_exponent"`
	InputLight      [3]float64 `json:"input_lighting"`
	ModulationAlpha float64    `json:"modulation_alpha"`
	DistanceFog     bool       `json:"distance_fog"`
	DistanceScale   float64    `json:"distance_fog_option_scale"`
	ScreenY         float64    `json:"screen_y"`
}

func (o ShaderMoodOptions) Validate() error {
	for _, v := range []float64{o.BloomScale, o.HorizontalSum, o.VerticalSum, o.Gamma, o.DistanceScale, o.ScreenY, o.ModulationAlpha, o.InputLight[0], o.InputLight[1], o.InputLight[2]} {
		if math.IsNaN(v) || math.IsInf(v, 0) || v < 0 || v > 8 {
			return fmt.Errorf("shader audit options must be finite and within 0..8")
		}
	}
	if o.Gamma == 0 || o.InputLight[0] == 0 || o.InputLight[1] == 0 || o.InputLight[2] == 0 || o.ScreenY > 1 || o.ModulationAlpha > 1 {
		return fmt.Errorf("shader audit needs positive gamma/lighting and screen_y/modulation_alpha within 0..1")
	}
	return nil
}

// ShaderUniformPatch evaluates recovered stages with explicitly assumed pass
// bindings: combine -> downsample/blur -> copy-add. On a uniform field, spatial
// taps reduce to their sums. This is NOT a substitute for spatial bloom on
// textured pixels, and the runtime pass bindings remain to be verified.
func ShaderUniformPatch(c [3]float64, m Mood, o ShaderMoodOptions) [3]float64 {
	for k := range c {
		c[k] /= o.InputLight[k]
	}
	intensity := m.BloomIntensity * o.BloomScale
	c = ShaderTerrainGrade(c, o.InputLight, o.ModulationAlpha, m, o.ColorCorrection, o.Bloom, intensity)
	if o.DistanceFog {
		c = ShaderDistanceFog(c, o.ScreenY, 1, o.DistanceScale, m)
	}
	c = ShaderPostGrade(c, m, o.ColorCorrection, o.Gamma)
	if o.Bloom {
		b := ShaderBloomExtract([4][3]float64{c, c, c, c}, m, intensity)
		for k := range b {
			b[k] *= o.HorizontalSum * o.VerticalSum
		}
		c = ShaderBloomCompose(c, ShaderBloomTint(b, m), true)
	}
	for k := range c {
		c[k] = clampMood(c[k])
	}
	return c
}

// Already-graded inputs admit only an approximate inverse; clipping loses data.
// The residual of this inverse is reported separately from prediction error.
func inverseShaderPatch(c [3]float64, m Mood, o ShaderMoodOptions) ([3]float64, float64) {
	x := c
	for i := 0; i < 80; i++ {
		y := ShaderUniformPatch(x, m, o)
		for k := range x {
			x[k] = clampMood(x[k] + .4*(c[k]-y[k]))
		}
	}
	y := ShaderUniformPatch(x, m, o)
	var e float64
	for k := range x {
		e += (y[k] - c[k]) * (y[k] - c[k])
	}
	return x, e / 3
}

type ShaderMoodAudit struct {
	Options     ShaderMoodOptions  `json:"options"`
	Assumptions []string           `json:"assumptions"`
	InverseRMS  map[string]float64 `json:"input_inverse_rms"`
	Results     []MoodFitResult    `json:"results"`
}

// ShaderMoodResiduals compares the existing default with a fixed shader audit.
// Holdout is a reporting label only: this path performs no coefficient fitting.
func ShaderMoodResiduals(cases []MoodFitCase, o ShaderMoodOptions, holdout string) (ShaderMoodAudit, error) {
	r := ShaderMoodAudit{Options: o, InverseRMS: map[string]float64{}, Assumptions: []string{
		"uniform fully visible terrain; no sprites, animated fog, vignette or spatial neighbourhoods",
		"input lighting is already present and is divided out before terrain correction, then reapplied; prior clipping cannot be undone",
		"effective bloom intensity assumed mood bloom_intensity times supplied option scale; CPU mapping not captured",
		"combine output assumed input to bloom extraction and copy-add; runtime bindings and intermediate render-target formats not captured",
		"uniform bloom uses supplied tap sums; sigma-to-tap generation and scene-sized blur not inferred",
		"screen_y is an explicit constant for this audit, not a recovered camera/viewport coordinate",
		"negative/overrange output is clipped as a display target; no assertion about sRGB resource views",
	}}
	if err := o.Validate(); err != nil {
		return r, err
	}
	if err := ValidateMoodFitCases(cases); err != nil {
		return r, err
	}
	for _, c := range cases {
		if !c.Mood.ShaderFields || (c.InputMood != nil && !c.InputMood.ShaderFields) {
			return r, fmt.Errorf("%s: report lacks shader mood fields; regenerate from manifest and installation", c.Name)
		}
		moods := []Mood{c.Mood}
		if c.InputMood != nil {
			moods = append(moods, *c.InputMood)
		}
		for _, m := range moods {
			values := append([]float64{m.BloomTerrain, m.ShadowStrength}, m.BloomScene[:]...)
			values = append(values, m.ShadowColor[:]...)
			for _, v := range values {
				if math.IsNaN(v) || math.IsInf(v, 0) || math.Abs(v) > 16 {
					return r, fmt.Errorf("%s: invalid shader mood fields", c.Name)
				}
			}
		}
		v := MoodFitResult{Name: c.Name, Mood: c.Mood.Name, HeldOut: strings.EqualFold(c.Mood.Name, holdout), Before: 255 * math.Sqrt(moodError(CurrentMoodModel(), c))}
		var total, inv float64
		for _, s := range c.Samples {
			x := s.Input
			var ie float64
			if c.InputMood != nil {
				x, ie = inverseShaderPatch(x, *c.InputMood, o)
			}
			y := ShaderUniformPatch(x, c.Mood, o)
			old := moodPrediction(CurrentMoodModel(), c, s)
			var e, before float64
			for k := range y {
				e += (y[k] - s.Capture[k]) * (y[k] - s.Capture[k]) / 3
				before += (old[k] - s.Capture[k]) * (old[k] - s.Capture[k]) / 3
			}
			v.Residuals = append(v.Residuals, MoodResidual{Name: s.Name, Excluded: s.Excluded, Before: 255 * math.Sqrt(before), After: 255 * math.Sqrt(e), Predicted: y})
			if !s.Excluded {
				total += e
				inv += ie
				v.Samples++
			}
		}
		v.After = 255 * math.Sqrt(total/float64(v.Samples))
		r.InverseRMS[c.Name] = 255 * math.Sqrt(inv/float64(v.Samples))
		r.Results = append(r.Results, v)
	}
	return r, nil
}
