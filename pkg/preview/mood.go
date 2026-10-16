package preview

import (
	"encoding/json"
	"fmt"
	"image"
	"math"
	"os"
	"path/filepath"
	"strings"
)

// Mood is one colour mood ("Color Mood" in the editor) from the game's
// terrain/colorcorrection_json/colorcorrection.json. Only the fields the
// preview grade uses are decoded.
type Mood struct {
	ShaderFields    bool       `json:"shader_fields,omitempty"`
	Name            string     `json:"name"`
	Saturation      float64    `json:"saturation"`
	Contrast        float64    `json:"contrast"`
	Brightness      float64    `json:"brightness"`
	TintFactor      float64    `json:"tint_factor"`
	TintColor       [3]float64 `json:"tint_color"`
	TerrainTint     [3]float64 `json:"terrain_tint_color"`
	BloomThreshold  float64    `json:"bloom_threshold"`
	BloomIntensity  float64    `json:"bloom_intensity"`
	BloomTint       [3]float64 `json:"bloom_tint_color"`
	ShadowStrength  float64    `json:"shadow_strength"`
	CloudStrength   float64    `json:"cloud_strength"`
	DistanceFog     float64    `json:"distance_fog_visible_intensity"`
	FogColor        [3]float64 `json:"distance_fog_color"`
	BloomSaturation float64    `json:"bloom_saturation"`
	BloomScene      [3]float64 `json:"bloom_scene_intensity"`
	BloomTerrain    float64    `json:"bloom_terrain_correction"`
	BloomSigma      float64    `json:"bloom_sigma"`
	ShadowColor     [3]float64 `json:"shadow_color"`
	VignetteBorder  [2]float64 `json:"vignette_border"`
	VignetteRadius  float64    `json:"vignette_radius"`
	VignetteAmount  float64    `json:"vignette_intensity"`
	VignetteColor   [3]float64 `json:"vignette_color"`
}

// ShaderPostGrade follows the recovered combine-pass arithmetic. Gamma here is
// the engine option exponent, NOT the mood JSON's unused gamma field. Every
// saturation point matters; brightness does not precede contrast or saturation.
// This is a stage equation, not a claim to reproduce the complete renderer.
func ShaderPostGrade(c [3]float64, m Mood, correction bool, gamma float64) [3]float64 {
	if correction {
		l := moodLuma(c[0], c[1], c[2])
		for k := range c {
			c[k] = clampMood(l + m.Saturation*(c[k]-l))
			c[k] = clampMood((c[k]-.5)*m.Contrast + .5)
			c[k] = clampMood(c[k] * m.Brightness)
			c[k] = clampMood(m.TintColor[k] + (1-m.TintFactor)*(c[k]-m.TintColor[k]))
		}
	}
	for k := range c {
		c[k] = math.Min(1, math.Pow(math.Abs(c[k]), gamma))
	}
	return c
}

// ShaderTerrainGrade is the terrain branch before sprite composition and post
// grading. light = diffuse*attributeTexture.b + ambient + base; the caller must
// supply these engine values. Modulation RGB is not used by this stage.
func ShaderTerrainGrade(c, light [3]float64, modulationAlpha float64, m Mood, correction, bloom bool, bloomIntensity float64) [3]float64 {
	shadow := 2 * math.Max(0, math.Min(.375, modulationAlpha))
	for k := range c {
		if correction {
			c[k] *= m.TerrainTint[k]
		}
		if bloom {
			c[k] = clampMood(c[k] * (1 - (1-m.BloomTerrain)*bloomIntensity))
		}
		c[k] *= light[k]
		if correction {
			c[k] *= 1 + clampMood(shadow*m.ShadowStrength)*(m.ShadowColor[k]-1)
		} else {
			c[k] *= 1 - shadow
		}
	}
	return c
}

// ShaderTerrainAttributes packs the visibility and lightmap samples. These are
// not terrain IDs or a material palette index.
func ShaderTerrainAttributes(visibility, lightmap [2]float64, fogBorder bool) [4]float64 {
	a := [4]float64{0, 0, lightmap[0], lightmap[1]}
	if fogBorder {
		a[0] = clampMood((visibility[0]-.5)*1.5 + .5)
		a[1] = clampMood((visibility[1]-.5)*2 + .5)
	} else {
		a[0] = clampMood((visibility[0]-.5)*2 + .5)
		a[1] = clampMood((visibility[1]-.5)*5 + .5)
	}
	return a
}

func ShaderTerrainLight(attributeBlue float64, diffuse, ambient, base [3]float64) [3]float64 {
	var light [3]float64
	for k := range light {
		light[k] = diffuse[k]*attributeBlue + ambient[k] + base[k]
	}
	return light
}

// ShaderSpriteCombine composites straight sprite RGB, not premultiplied RGB.
// receivesShadow is the sampled metadata's low flag bit. Terrain has already
// been shadowed separately by ShaderTerrainGrade.
func ShaderSpriteCombine(terrain, sprite [3]float64, alpha, modulationAlpha float64, receivesShadow bool, m Mood, correction bool) [3]float64 {
	if receivesShadow {
		shadow := 2 * math.Max(0, math.Min(.375, modulationAlpha))
		for k := range sprite {
			if correction {
				// Unlike the terrain path, sprite shadow strength is not saturated.
				sprite[k] *= 1 + shadow*m.ShadowStrength*(m.ShadowColor[k]-1)
			} else {
				sprite[k] *= 1 - shadow
			}
		}
	}
	for k := range terrain {
		terrain[k] += alpha * (sprite[k] - terrain[k])
	}
	return terrain
}

// Distance fog is additive in this combine pass, not a lerp toward fog colour.
// screenY is the full pass's normalized coordinate, not a crop-relative value.
func ShaderDistanceFog(c [3]float64, screenY, optionEnable, optionIntensity float64, m Mood) [3]float64 {
	for k := range c {
		c[k] += (1 - screenY) * m.FogColor[k] * optionEnable * optionIntensity * m.DistanceFog
	}
	return c
}

// ShaderVignette uses four rectangular edge ramps; smoothness is unused in the
// recovered pass. It operates before the post colour correction.
func ShaderVignette(c [3]float64, uv [2]float64, explored, optionIntensity float64, m Mood) [3]float64 {
	var edge float64
	for axis, p := range uv {
		border := (1 - m.VignetteRadius) * m.VignetteBorder[axis]
		edge += math.Max(0, border-math.Max(0, p))
		edge += math.Max(0, border-math.Max(0, 1-p))
	}
	factor := edge * math.Min(1, math.Max(.5, explored)) * m.VignetteAmount * optionIntensity
	for k := range c {
		c[k] += factor * (optionIntensity*m.VignetteColor[k] - c[k])
	}
	return c
}

// ShaderBloomExtract averages the four bilinear fetch results BEFORE thresholding
// each channel and applying bloom saturation. It does not clip the output to 1.
func ShaderBloomExtract(samples [4][3]float64, m Mood, intensity float64) [3]float64 {
	var c [3]float64
	for _, s := range samples {
		for k := range c {
			c[k] += .25 * s[k]
		}
	}
	for k := range c {
		c[k] = math.Max(0, c[k]*intensity*m.BloomScene[k]-m.BloomThreshold)
	}
	l := moodLuma(c[0], c[1], c[2])
	for k := range c {
		c[k] = l + m.BloomSaturation*(c[k]-l)
	}
	return c
}

// ShaderBloomFilter is one eight-fetch blur pass. Weights and offsets are CPU
// supplied; the shader does not normalize weights or compute them from sigma.
// Callers supply already-sampled colours and apply BloomTint after the vertical
// pass only. No assumed Gaussian kernel is hidden here.
func ShaderBloomFilter(samples [8][3]float64, weights [8]float64) [3]float64 {
	var c [3]float64
	for i, s := range samples {
		for k := range c {
			c[k] += weights[i] * s[k]
		}
	}
	return c
}

func ShaderBloomTint(c [3]float64, m Mood) [3]float64 {
	for k := range c {
		c[k] *= m.BloomTint[k]
	}
	return c
}

// ShaderBloomCompose is the copy pass. Render-target quantization/clipping, and
// which texture was bound as its input, are outside this equation.
func ShaderBloomCompose(scene, bloom [3]float64, enabled bool) [3]float64 {
	if enabled {
		for k := range scene {
			scene[k] += bloom[k]
		}
	}
	return scene
}

type MoodReport struct {
	Name        string   `json:"name"`
	Source      string   `json:"source"`
	Rule        string   `json:"rule"`
	NotModelled []string `json:"not_modelled"`
}

// The default retains the nine-capture fit. The later meadow + 16-terrain
// swatch calibration (2026-10-16, beach/road excluded) improved Autumn with
// mixed tint and coloured bloom but regressed Savannah and held-out Steppes.
// Such fits are opt-in through ApplyMoodModel, not silently promoted. Residuals
// also include registration, texture, water, lighting and shadow differences;
// they do not identify the engine's shader formula.

// LoadMood reads mood name (case-insensitive; "" means Default) from the
// game install.
func LoadMood(gameDir, name string) (Mood, error) {
	path := filepath.Join(gameDir, "resources", "_common", "terrain", "colorcorrection_json", "colorcorrection.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		return Mood{}, fmt.Errorf("colour moods: %w", err)
	}
	var cc struct {
		Types []Mood `json:"correction_types"`
	}
	if err := json.Unmarshal(raw, &cc); err != nil {
		return Mood{}, fmt.Errorf("colour moods: %w", err)
	}
	if name == "" {
		name = "Default"
	}
	var names []string
	for _, m := range cc.Types {
		if strings.EqualFold(m.Name, name) {
			m.ShaderFields = true
			return m, nil
		}
		names = append(names, m.Name)
	}
	return Mood{}, fmt.Errorf("unknown colour mood %q; known: %s", name, strings.Join(names, ", "))
}

func moodLuma(r, g, b float64) float64 { return 0.2126*r + 0.7152*g + 0.0722*b }

// gradePixel applies m to a scene colour (0-1): terrain tint and brightness,
// contrast about a mid pivot, saturation about luma, a blend toward the tint
// colour by the tint factor, and bloom added above the threshold.
func gradePixel(c [3]float64, m Mood) [3]float64 {
	return CurrentMoodModel().grade(c, m)
}

// ApplyMood grades img in place. The pixel at the image origin is the
// background (the diamond projection never covers the corners) and is left
// as it is.
func ApplyMood(img *image.RGBA, m Mood, source string) MoodReport {
	return applyMoodModel(img, m, source, CurrentMoodModel())
}

// ApplyMoodModel applies an explicitly selected calibration, including its
// assumptions in the render report. Invalid models do not mutate the image.
func ApplyMoodModel(img *image.RGBA, m Mood, source string, p MoodModel) (MoodReport, error) {
	if err := p.Validate(); err != nil {
		return MoodReport{}, err
	}
	return applyMoodModel(img, m, source, p), nil
}

func applyMoodModel(img *image.RGBA, m Mood, source string, p MoodModel) MoodReport {
	b := img.Bounds()
	if b.Empty() {
		return MoodReport{Name: m.Name, Source: source, Rule: "empty image"}
	}
	o := img.PixOffset(b.Min.X, b.Min.Y)
	bg := [3]uint8{img.Pix[o], img.Pix[o+1], img.Pix[o+2]}
	clamp := func(v float64) uint8 { return uint8(math.Round(255 * math.Max(0, math.Min(1, v)))) }
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			i := img.PixOffset(x, y)
			pixel := img.Pix[i : i+3 : i+3]
			if pixel[0] == bg[0] && pixel[1] == bg[1] && pixel[2] == bg[2] {
				continue
			}
			g := p.grade([3]float64{float64(pixel[0]) / 255, float64(pixel[1]) / 255, float64(pixel[2]) / 255}, m)
			pixel[0], pixel[1], pixel[2] = clamp(g[0]), clamp(g[1]), clamp(g[2])
		}
	}
	return MoodReport{
		Name:   m.Name,
		Source: source,
		Rule:   fmt.Sprintf("empirical display-RGB grade, not engine shader: terrain tint x brightness; contrast pivot %.3f; saturation; %s tint (add %.3f, multiply %.3f, attenuation %.3f); bloom %.3f with chroma %.3f and highlight suppression %.3f; distance-fog colour veil %.3f. Joint meadow/swatch fits require held-out review; no universal accuracy guarantee", p.Pivot, p.TintShape, p.Tint, p.Multiply, p.Attenuate, p.Bloom, p.BloomChroma, p.Highlight, p.Fog),
		NotModelled: []string{
			fmt.Sprintf("cloud shadows (cloud_strength %.2f)", m.CloudStrength),
			fmt.Sprintf("sprite shadow strength (shadow_strength %.2f); sprite shadows are not drawn yet", m.ShadowStrength),
			"spatial bloom blur/sigma, bloom terrain correction and scene intensity, vignette, sharpening, depth-dependent fog and fog of war",
		},
	}
}
