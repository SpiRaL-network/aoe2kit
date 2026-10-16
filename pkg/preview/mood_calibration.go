package preview

import (
	"aoe2kit/pkg/gfx"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
)

// MoodCalibrationManifest paths are relative to the manifest, not the process cwd.
type MoodCalibrationManifest struct {
	Cases []MoodCalibrationCase `json:"cases"`
}
type MoodCalibrationCase struct {
	Name         string                  `json:"name"`
	Preview      string                  `json:"preview"`
	Capture      string                  `json:"capture"`
	Mood         string                  `json:"mood"`
	InputMood    string                  `json:"input_mood,omitempty"` // Empty means raw, not Default.
	Offset       *[2]int                 `json:"offset,omitempty"`     // Capture coordinate = preview coordinate + offset.
	Stride       int                     `json:"stride,omitempty"`
	Regions      []MoodCalibrationRegion `json:"regions,omitempty"`
	TextureInput bool                    `json:"texture_input,omitempty"`
	TextureGain  [3]float64              `json:"texture_gain,omitempty"`
}
type MoodCalibrationRegion struct {
	Name    string   `json:"name"`
	Points  [][2]int `json:"points"` // Preview pixel positions; medians over paired valid points.
	Exclude bool     `json:"exclude,omitempty"`
	Texture string   `json:"texture,omitempty"`
}

type MoodTextureStats struct {
	SHA256 string     `json:"sha256"`
	Mean   [3]float64 `json:"mean_rgb"`
	Median [3]float64 `json:"median_rgb"`
	Pixels int        `json:"opaque_pixels"`
}

// MeasureMoodTexture uses the full original DDS mip, not the preview's resized
// texture. Its global distribution is a diagnostic, not a registered UV sample.
func MeasureMoodTexture(path string) (MoodTextureStats, error) {
	f, err := os.Open(path)
	if err != nil {
		return MoodTextureStats{}, err
	}
	defer f.Close()
	im, err := gfx.DecodeDDS(f)
	if err != nil {
		return MoodTextureStats{}, err
	}
	var s MoodTextureStats
	var hist [3][256]int
	for y := im.Bounds().Min.Y; y < im.Bounds().Max.Y; y++ {
		for x := im.Bounds().Min.X; x < im.Bounds().Max.X; x++ {
			i := im.PixOffset(x, y)
			if im.Pix[i+3] != 255 {
				continue
			}
			s.Pixels++
			for k, v := range im.Pix[i : i+3] {
				hist[k][v]++
				s.Mean[k] += float64(v) / 255
			}
		}
	}
	if s.Pixels == 0 {
		return s, fmt.Errorf("texture has no opaque pixels")
	}
	for k := range s.Mean {
		s.Mean[k] /= float64(s.Pixels)
		n := 0
		for v, c := range hist[k] {
			n += c
			if n > s.Pixels/2 {
				s.Median[k] = float64(v) / 255
				break
			}
		}
	}
	if _, err = f.Seek(0, 0); err != nil {
		return s, err
	}
	h := sha256.New()
	if _, err = io.Copy(h, f); err != nil {
		return s, err
	}
	s.SHA256 = hex.EncodeToString(h.Sum(nil))
	return s, nil
}

type MoodAlignment struct {
	Name          string  `json:"name"`
	Offset        [2]int  `json:"offset"`
	Correlation   float64 `json:"correlation"`
	PreviewSHA256 string  `json:"preview_sha256"`
	CaptureSHA256 string  `json:"capture_sha256"`
}

func calibrationImage(path string) (*image.RGBA, string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, "", err
	}
	defer f.Close()
	cfg, err := png.DecodeConfig(f)
	if err != nil {
		return nil, "", err
	}
	if cfg.Width <= 0 || cfg.Height <= 0 || cfg.Width > 8192 || cfg.Height > 8192 || int64(cfg.Width)*int64(cfg.Height) > maxImagePixels {
		return nil, "", fmt.Errorf("calibration PNG exceeds image limits")
	}
	if _, err = f.Seek(0, 0); err != nil {
		return nil, "", err
	}
	im, err := png.Decode(f)
	if err != nil {
		return nil, "", err
	}
	dst := image.NewRGBA(im.Bounds())
	draw.Draw(dst, dst.Bounds(), im, im.Bounds().Min, draw.Src)
	// Hash decoded pixels: independent of PNG compression and metadata.
	h := sha256.Sum256(dst.Pix)
	// The preview diamond's uniform corner background is not scene evidence.
	bg := dst.RGBAAt(0, 0)
	nonUniform := false
	for i := 0; i < len(dst.Pix); i += 4 {
		if dst.Pix[i] != bg.R || dst.Pix[i+1] != bg.G || dst.Pix[i+2] != bg.B {
			nonUniform = true
			break
		}
	}
	if nonUniform && bg == dst.RGBAAt(cfg.Width-1, 0) && bg == dst.RGBAAt(0, cfg.Height-1) && bg == dst.RGBAAt(cfg.Width-1, cfg.Height-1) {
		for i := 0; i < len(dst.Pix); i += 4 {
			if dst.Pix[i] == bg.R && dst.Pix[i+1] == bg.G && dst.Pix[i+2] == bg.B {
				dst.Pix[i+3] = 0
			}
		}
	}
	return dst, hex.EncodeToString(h[:]), nil
}

func calibrationRGB(im *image.RGBA, x, y int) ([3]float64, bool) {
	if !image.Pt(x, y).In(im.Bounds()) {
		return [3]float64{}, false
	}
	i := im.PixOffset(x, y)
	c := im.Pix[i : i+4]
	if c[3] < 255 || int(c[0])+int(c[1])+int(c[2]) < 15 {
		return [3]float64{}, false
	}
	return [3]float64{float64(c[0]) / 255, float64(c[1]) / 255, float64(c[2]) / 255}, true
}

func moodCorrelation(a, b *image.RGBA, dx, dy, stride int) (float64, int) {
	var sx, sy, sxx, syy, sxy float64
	n := 0
	for y := a.Bounds().Min.Y; y < a.Bounds().Max.Y; y += stride {
		for x := a.Bounds().Min.X; x < a.Bounds().Max.X; x += stride {
			u, ok := calibrationRGB(a, x, y)
			v, ok2 := calibrationRGB(b, x+dx, y+dy)
			if !ok || !ok2 {
				continue
			}
			p, q := moodLuma(u[0], u[1], u[2]), moodLuma(v[0], v[1], v[2])
			sx += p
			sy += q
			sxx += p * p
			syy += q * q
			sxy += p * q
			n++
		}
	}
	if n < 16 {
		return -1, n
	}
	d := math.Sqrt(math.Max(0, (sxx-sx*sx/float64(n))*(syy-sy*sy/float64(n))))
	if d < 1e-12 {
		return -1, n
	}
	return (sxy - sx*sy/float64(n)) / d, n
}

// AlignMoodImages searches translations only. It does not invent scale/rotation
// corrections; callers must supply screenshots at matching pixel scale.
func AlignMoodImages(a, b *image.RGBA) ([2]int, float64, error) {
	coarse := func(src *image.RGBA) *image.RGBA {
		dst := image.NewRGBA(image.Rect(0, 0, (src.Bounds().Dx()+7)/8, (src.Bounds().Dy()+7)/8))
		for y := 0; y < dst.Bounds().Dy(); y++ {
			for x := 0; x < dst.Bounds().Dx(); x++ {
				var sum [3]int
				n := 0
				for yy := y * 8; yy < min(y*8+8, src.Bounds().Dy()); yy++ {
					for xx := x * 8; xx < min(x*8+8, src.Bounds().Dx()); xx++ {
						i := src.PixOffset(xx, yy)
						if src.Pix[i+3] != 255 {
							continue
						}
						for k := range sum {
							sum[k] += int(src.Pix[i+k])
						}
						n++
					}
				}
				if n > 0 {
					i := dst.PixOffset(x, y)
					for k := range sum {
						dst.Pix[i+k] = uint8(sum[k] / n)
					}
					dst.Pix[i+3] = 255
				}
			}
		}
		return dst
	}
	ca, cb := coarse(a), coarse(b)
	best := -2.0
	shift := [2]int{}
	minSamples := max(16, min(ca.Bounds().Dx()*ca.Bounds().Dy(), cb.Bounds().Dx()*cb.Bounds().Dy())/(2*2*3))
	for dy := -512; dy <= 512; dy += 8 {
		for dx := -512; dx <= 512; dx += 8 {
			c, n := moodCorrelation(ca, cb, dx/8, dy/8, 2)
			if n >= minSamples && c > best {
				best = c
				shift = [2]int{dx, dy}
			}
		}
	}
	base := shift
	best = -2
	for dy := base[1] - 8; dy <= base[1]+8; dy++ {
		for dx := base[0] - 8; dx <= base[0]+8; dx++ {
			c, n := moodCorrelation(a, b, dx, dy, 6)
			if n >= max(16, min(a.Bounds().Dx()*a.Bounds().Dy(), b.Bounds().Dx()*b.Bounds().Dy())/(6*6*3)) && c > best {
				best = c
				shift = [2]int{dx, dy}
			}
		}
	}
	if best < 0.2 {
		return shift, best, fmt.Errorf("no reliable translation (correlation %.3f); supply registered images or explicit offset", best)
	}
	return shift, best, nil
}

func PrepareMoodCalibration(path, gameDir string) ([]MoodFitCase, []MoodAlignment, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}
	var manifest MoodCalibrationManifest
	if err = json.Unmarshal(raw, &manifest); err != nil {
		return nil, nil, err
	}
	if len(manifest.Cases) == 0 || len(manifest.Cases) > 64 {
		return nil, nil, fmt.Errorf("need 1..64 calibration cases")
	}
	var cases []MoodFitCase
	var alignments []MoodAlignment
	textureStats := map[string]MoodTextureStats{}
	for _, c := range manifest.Cases {
		resolve := func(p string) string {
			if filepath.IsAbs(p) {
				return p
			}
			return filepath.Join(filepath.Dir(path), p)
		}
		a, ah, err := calibrationImage(resolve(c.Preview))
		if err != nil {
			return nil, nil, fmt.Errorf("%s preview: %w", c.Name, err)
		}
		b, bh, err := calibrationImage(resolve(c.Capture))
		if err != nil {
			return nil, nil, fmt.Errorf("%s capture: %w", c.Name, err)
		}
		m, err := LoadMood(gameDir, c.Mood)
		if err != nil {
			return nil, nil, err
		}
		fc := MoodFitCase{Name: c.Name, Mood: m}
		if c.TextureInput && (len(c.Regions) == 0 || c.InputMood != "") {
			return nil, nil, fmt.Errorf("texture_input requires regions and an ungraded input")
		}
		if c.InputMood != "" {
			im, err := LoadMood(gameDir, c.InputMood)
			if err != nil {
				return nil, nil, err
			}
			fc.InputMood = &im
		}
		var offset [2]int
		var corr float64
		if c.Offset == nil {
			offset, corr, err = AlignMoodImages(a, b)
			if err != nil {
				return nil, nil, fmt.Errorf("%s: %w", c.Name, err)
			}
		} else {
			offset = *c.Offset
			corr, _ = moodCorrelation(a, b, offset[0], offset[1], 6)
		}
		alignments = append(alignments, MoodAlignment{c.Name, offset, corr, ah, bh})
		pair := func(x, y int) ([3]float64, [3]float64, bool) {
			u, ok := calibrationRGB(a, x, y)
			v, ok2 := calibrationRGB(b, x+offset[0], y+offset[1])
			if fc.InputMood != nil {
				for _, ch := range u {
					if ch <= 1.0/255 || ch >= 254.0/255 {
						ok = false
					}
				}
			}
			return u, v, ok && ok2
		}
		if len(c.Regions) > 0 {
			for _, r := range c.Regions {
				var ins, outs [3][]float64
				for _, pt := range r.Points {
					u, v, ok := pair(pt[0], pt[1])
					if !ok {
						continue
					}
					for k := range u {
						ins[k] = append(ins[k], u[k])
						outs[k] = append(outs[k], v[k])
					}
				}
				if len(ins[0]) < 1 {
					return nil, nil, fmt.Errorf("%s/%s: no paired samples", c.Name, r.Name)
				}
				s := MoodSample{Name: r.Name, Excluded: r.Exclude}
				for k := range s.Input {
					sort.Float64s(ins[k])
					sort.Float64s(outs[k])
					s.Input[k] = ins[k][len(ins[k])/2]
					s.Capture[k] = outs[k][len(outs[k])/2]
				}
				fc.Samples = append(fc.Samples, s)
				if r.Texture != "" {
					texturePath := resolve(r.Texture)
					stats, ok := textureStats[texturePath]
					if !ok {
						stats, err = MeasureMoodTexture(texturePath)
						if err != nil {
							return nil, nil, err
						}
						textureStats[texturePath] = stats
					}
					s.Texture = &stats
					if c.TextureInput {
						gain := c.TextureGain
						if gain == [3]float64{} {
							gain = [3]float64{1, 1, 1}
						}
						for k := range s.Input {
							if gain[k] <= 0 || math.IsNaN(gain[k]) || math.IsInf(gain[k], 0) {
								return nil, nil, fmt.Errorf("texture gain must be finite and positive")
							}
							s.Input[k] = clampMood(stats.Median[k] * gain[k])
						}
					}
					fc.Samples[len(fc.Samples)-1] = s
				} else if c.TextureInput {
					return nil, nil, fmt.Errorf("%s/%s: texture_input requires a texture", c.Name, r.Name)
				}
			}
		} else {
			stride := c.Stride
			if stride == 0 {
				stride = 12
			}
			if stride < 1 {
				return nil, nil, fmt.Errorf("stride must be positive")
			}
			for y := 0; y < a.Bounds().Dy(); y += stride {
				for x := 0; x < a.Bounds().Dx(); x += stride {
					u, v, ok := pair(x, y)
					if ok {
						fc.Samples = append(fc.Samples, MoodSample{Name: fmt.Sprintf("%d,%d", x, y), Input: u, Capture: v})
					}
					if len(fc.Samples) > 100000 {
						return nil, nil, fmt.Errorf("too many samples; increase stride")
					}
				}
			}
		}
		n := 0
		for _, s := range fc.Samples {
			if !s.Excluded {
				n++
			}
		}
		if n == 0 {
			return nil, nil, fmt.Errorf("%s: no included samples", c.Name)
		}
		cases = append(cases, fc)
	}
	return cases, alignments, nil
}
