package preview

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strings"
)

// MoodModel is a shared empirical grade, not a reconstruction of the engine shader.
// TintShape selects lerp, multiply, or mixed. All coefficients are shared by moods.
type MoodModel struct {
	TintShape   string  `json:"tint_shape"`
	Pivot       float64 `json:"pivot"`
	Tint        float64 `json:"tint"`
	Multiply    float64 `json:"multiply"`
	Attenuate   float64 `json:"attenuate"`
	Bloom       float64 `json:"bloom"`
	Highlight   float64 `json:"highlight"`
	Fog         float64 `json:"fog"`
	BloomChroma float64 `json:"bloom_chroma"`
}

func legacyMoodModel() MoodModel {
	return MoodModel{TintShape: "lerp", Pivot: 0.525, Tint: 1.095, Bloom: 0.29}
}

func CurrentMoodModel() MoodModel { return legacyMoodModel() }

func (p MoodModel) Validate() error {
	if p.TintShape != "lerp" && p.TintShape != "multiply" && p.TintShape != "mixed" {
		return fmt.Errorf("unknown tint shape %q", p.TintShape)
	}
	for i, v := range []float64{p.Pivot, p.Tint, p.Multiply, p.Attenuate, p.Bloom, p.Highlight, p.Fog, p.BloomChroma} {
		upper := []float64{1, 4, 6, 4, 2, 1, 1, 1}
		if math.IsNaN(v) || math.IsInf(v, 0) || v < 0 || v > upper[i] {
			return fmt.Errorf("invalid mood model coefficient %d", i)
		}
	}
	return nil
}

// LoadMoodModel reads the model from a calibrate-mood JSON report.
func LoadMoodModel(path string) (MoodModel, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return MoodModel{}, err
	}
	var report struct {
		Model MoodModel `json:"model"`
	}
	if err = json.Unmarshal(b, &report); err != nil {
		return MoodModel{}, err
	}
	return report.Model, report.Model.Validate()
}

func (p MoodModel) grade(c [3]float64, m Mood) [3]float64 {
	raw := c
	scene := moodLuma(c[0], c[1], c[2])
	for k := range c {
		c[k] = (c[k]*m.TerrainTint[k]*m.Brightness-p.Pivot)*m.Contrast + p.Pivot
	}
	l := moodLuma(c[0], c[1], c[2])
	bloom := math.Max(0, scene-m.BloomThreshold) * m.BloomIntensity * p.Bloom
	fog := math.Min(0.8, p.Fog*m.DistanceFog)
	for k := range c {
		c[k] = l + m.Saturation*(c[k]-l)
		switch p.TintShape {
		case "lerp":
			t := math.Min(1, m.TintFactor*p.Tint)
			c[k] = c[k]*(1-t) + m.TintColor[k]*t
		case "multiply":
			c[k] *= 1 + p.Multiply*m.TintFactor*m.TintColor[k]
		case "mixed":
			c[k] = c[k]*(1-p.Attenuate*m.TintFactor) + m.TintFactor*m.TintColor[k]*(p.Tint+p.Multiply*c[k])
		}
		coloredBloom := bloom + p.BloomChroma*m.BloomSaturation*m.BloomIntensity*p.Bloom*(math.Max(0, raw[k]-m.BloomThreshold)-math.Max(0, scene-m.BloomThreshold))
		c[k] += coloredBloom * m.BloomTint[k] * (1 - p.Highlight*clampMood(c[k]))
		c[k] = c[k]*(1-fog) + m.FogColor[k]*fog
	}
	return c
}

func clampMood(v float64) float64 { return math.Max(0, math.Min(1, v)) }

// Inversion is only used for already-graded calibration inputs, never rendering.
// Saturated source pixels cannot be recovered and are excluded during sampling.
func (p MoodModel) ungrade(c [3]float64, m Mood) [3]float64 {
	x := c
	for it := 0; it < 12; it++ {
		y := p.grade(x, m)
		for k := range x {
			slope := math.Max(0.2, m.TerrainTint[k]*m.Brightness*m.Contrast)
			x[k] = clampMood(x[k] + 0.7*(c[k]-y[k])/slope)
		}
	}
	return x
}

type MoodSample struct {
	Name     string            `json:"name"`
	Input    [3]float64        `json:"input_rgb"` // 0..1, unpremultiplied display RGB
	Capture  [3]float64        `json:"capture_rgb"`
	Excluded bool              `json:"excluded,omitempty"`
	Texture  *MoodTextureStats `json:"texture,omitempty"`
}

type MoodFitCase struct {
	Name      string       `json:"name"`
	Mood      Mood         `json:"mood"`
	InputMood *Mood        `json:"input_mood,omitempty"`
	Samples   []MoodSample `json:"samples"`
}

type MoodResidual struct {
	Name      string     `json:"name"`
	Excluded  bool       `json:"excluded,omitempty"`
	Before    float64    `json:"before_rms"`
	After     float64    `json:"after_rms"`
	Predicted [3]float64 `json:"predicted_rgb"`
}

type MoodFitResult struct {
	Name      string         `json:"name"`
	Mood      string         `json:"mood"`
	HeldOut   bool           `json:"held_out"`
	Samples   int            `json:"samples"`
	Before    float64        `json:"before_rms"`
	After     float64        `json:"after_rms"`
	Residuals []MoodResidual `json:"residuals"`
}

func moodPrediction(p MoodModel, c MoodFitCase, s MoodSample) [3]float64 {
	x := s.Input
	if c.InputMood != nil {
		x = p.ungrade(x, *c.InputMood)
	}
	y := p.grade(x, c.Mood)
	for k := range y {
		y[k] = clampMood(y[k])
	}
	return y
}

func moodError(p MoodModel, c MoodFitCase) float64 {
	var total float64
	n := 0
	for _, s := range c.Samples {
		if s.Excluded {
			continue
		}
		y := moodPrediction(p, c, s)
		for k := range y {
			d := y[k] - s.Capture[k]
			total += d * d
		}
		n++
	}
	if n == 0 {
		return 0
	}
	return total / float64(3*n)
}

// FitMoodModel uses bounded deterministic coordinate descent. Each case has equal
// weight, so a large meadow image cannot drown out small swatch sample sets.
// Held-out moods are never used for coefficients or tint-shape selection.
func FitMoodModel(cases []MoodFitCase, holdout string) (MoodModel, map[string]float64, error) {
	if err := ValidateMoodFitCases(cases); err != nil {
		return MoodModel{}, nil, err
	}
	train, held := 0, 0
	for _, c := range cases {
		if strings.EqualFold(c.Mood.Name, holdout) {
			held++
		} else {
			train++
		}
	}
	if train == 0 || (holdout != "" && held == 0) {
		return MoodModel{}, nil, fmt.Errorf("need training cases and a matching held-out mood")
	}
	// Bound fitting work; final residuals still evaluate every supplied sample.
	fitCases := append([]MoodFitCase(nil), cases...)
	for i, c := range fitCases {
		if len(c.Samples) > 1024 {
			fitCases[i].Samples = nil
			for j := 0; j < 1024; j++ {
				fitCases[i].Samples = append(fitCases[i].Samples, c.Samples[j*len(c.Samples)/1024])
			}
		}
	}
	loss := func(p MoodModel) float64 {
		var sum float64
		for _, c := range fitCases {
			if !strings.EqualFold(c.Mood.Name, holdout) {
				sum += moodError(p, c)
			}
		}
		return sum / float64(train)
	}
	best := legacyMoodModel()
	bestLoss := math.Inf(1)
	scores := map[string]float64{}
	for _, shape := range []string{"lerp", "multiply", "mixed"} {
		shapeLoss := math.Inf(1)
		for start := 0; start < 2; start++ {
			p := legacyMoodModel()
			p.TintShape = shape
			if shape == "multiply" {
				p.Tint = 0
				p.Multiply = 1
			}
			if shape == "mixed" {
				p.Attenuate = p.Tint
			}
			if start == 1 {
				p.Bloom = 0.8
				p.Highlight = 0.8
				p.BloomChroma = 0.5
			}
			value := loss(p)
			for _, step := range []float64{0.2, 0.08, 0.03, 0.01, 0.003} {
				for pass := 0; pass < 20; pass++ {
					changed := false
					for j := 0; j < 8; j++ {
						if (shape == "lerp" && (j == 2 || j == 3)) || (shape == "multiply" && (j == 1 || j == 3)) {
							continue
						}
						for _, sign := range []float64{-1, 1} {
							q := p
							fields := []*float64{&q.Pivot, &q.Tint, &q.Multiply, &q.Attenuate, &q.Bloom, &q.Highlight, &q.Fog, &q.BloomChroma}
							upper := []float64{1, 4, 6, 4, 2, 1, 1, 1}
							*fields[j] = math.Max(0, math.Min(upper[j], *fields[j]+sign*step))
							if v := loss(q); v < value-1e-12 {
								p, value = q, v
								changed = true
							}
						}
					}
					if !changed {
						break
					}
				}
			}
			if value < shapeLoss {
				shapeLoss = value
			}
			if value < bestLoss {
				best, bestLoss = p, value
			}
		}
		scores[shape] = 255 * math.Sqrt(shapeLoss)
	}
	return best, scores, nil
}

func ValidateMoodFitCases(cases []MoodFitCase) error {
	if len(cases) == 0 || len(cases) > 64 {
		return fmt.Errorf("need 1..64 fit cases")
	}
	for _, c := range cases {
		if len(c.Samples) > 100000 {
			return fmt.Errorf("%s: exceeds sample limit", c.Name)
		}
		moods := []Mood{c.Mood}
		if c.InputMood != nil {
			moods = append(moods, *c.InputMood)
		}
		for _, m := range moods {
			values := []float64{m.Saturation, m.Contrast, m.Brightness, m.TintFactor, m.BloomThreshold, m.BloomIntensity, m.DistanceFog, m.BloomSaturation}
			values = append(values, m.TerrainTint[:]...)
			values = append(values, m.TintColor[:]...)
			values = append(values, m.BloomTint[:]...)
			values = append(values, m.FogColor[:]...)
			for _, v := range values {
				if math.IsNaN(v) || math.IsInf(v, 0) || v < 0 || v > 16 {
					return fmt.Errorf("%s: invalid mood parameters", c.Name)
				}
			}
			if m.Brightness <= 0 || m.Contrast <= 0 || m.Saturation <= 0 {
				return fmt.Errorf("%s: grade must have positive brightness/contrast/saturation", c.Name)
			}
		}
		n := 0
		for _, s := range c.Samples {
			for k := range s.Input {
				for _, v := range []float64{s.Input[k], s.Capture[k]} {
					if math.IsNaN(v) || math.IsInf(v, 0) || v < 0 || v > 1 {
						return fmt.Errorf("%s: RGB outside 0..1", c.Name)
					}
				}
			}
			if !s.Excluded {
				n++
			}
		}
		if n == 0 {
			return fmt.Errorf("%s: no included samples", c.Name)
		}
	}
	return nil
}

func MoodResiduals(cases []MoodFitCase, model MoodModel, holdout string) []MoodFitResult {
	var out []MoodFitResult
	for _, c := range cases {
		r := MoodFitResult{Name: c.Name, Mood: c.Mood.Name, HeldOut: strings.EqualFold(c.Mood.Name, holdout), Before: 255 * math.Sqrt(moodError(legacyMoodModel(), c)), After: 255 * math.Sqrt(moodError(model, c))}
		for _, s := range c.Samples {
			a, b := moodPrediction(legacyMoodModel(), c, s), moodPrediction(model, c, s)
			var ea, eb float64
			for k := range a {
				ea += (a[k] - s.Capture[k]) * (a[k] - s.Capture[k])
				eb += (b[k] - s.Capture[k]) * (b[k] - s.Capture[k])
			}
			r.Residuals = append(r.Residuals, MoodResidual{Name: s.Name, Excluded: s.Excluded, Before: 255 * math.Sqrt(ea/3), After: 255 * math.Sqrt(eb/3), Predicted: b})
			if !s.Excluded {
				r.Samples++
			}
		}
		out = append(out, r)
	}
	return out
}
