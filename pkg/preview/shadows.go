package preview

import (
	"fmt"
	"image"
	"image/draw"
	"math"

	xdraw "golang.org/x/image/draw"
)

// ShadowOptions enables the experimental SLD coverage pass. Nil Mood means
// color correction is disabled, not that a default mood should be guessed.
type ShadowOptions struct {
	Mood    *Mood
	Overlap string // max (default) or sum; blend state is not yet engine-verified.
}

type ShadowReport struct {
	Overlap       string    `json:"overlap"`
	Mood          string    `json:"mood"`
	Drawn         int       `json:"drawn"`
	Missing       int       `json:"missing_layers"`
	CoveredPixels int       `json:"covered_pixels"`
	Skipped       []Skipped `json:"skipped,omitempty"`
	Limitations   []string  `json:"limitations"`
}

func shadowOverlap(opts ShadowOptions) (string, error) {
	if opts.Overlap == "" || opts.Overlap == "max" {
		return "max", nil
	}
	if opts.Overlap == "sum" {
		return "sum", nil
	}
	return "", fmt.Errorf("shadow overlap must be max or sum, got %q", opts.Overlap)
}

// Resample only the visible rectangle; large source canvases and cropped
// destinations must not allocate a full output-sized scratch per object.
func accumulateShadow(dst *image.Alpha, src image.Image, rect image.Rectangle, overlap string) {
	clip := rect.Intersect(dst.Bounds())
	if clip.Empty() {
		return
	}
	scaled := image.NewAlpha(clip)
	xdraw.CatmullRom.Scale(scaled, rect, src, src.Bounds(), draw.Src, nil)
	for y := clip.Min.Y; y < clip.Max.Y; y++ {
		for x := clip.Min.X; x < clip.Max.X; x++ {
			i, j := dst.PixOffset(x, y), scaled.PixOffset(x, y)
			a := scaled.Pix[j]
			if overlap == "sum" {
				dst.Pix[i] = uint8(min(255, int(dst.Pix[i])+int(a)))
			} else if a > dst.Pix[i] {
				dst.Pix[i] = a
			}
		}
	}
}

func applyShadows(dst *image.RGBA, coverage *image.Alpha, mood *Mood) int {
	count := 0
	for y := dst.Bounds().Min.Y; y < dst.Bounds().Max.Y; y++ {
		for x := dst.Bounds().Min.X; x < dst.Bounds().Max.X; x++ {
			a := coverage.AlphaAt(x, y).A
			if a == 0 {
				continue
			}
			i := dst.PixOffset(x, y)
			if dst.Pix[i+3] == 0 {
				continue
			}
			count++
			shadow := 2 * math.Min(.375, float64(a)/255)
			for k := 0; k < 3; k++ {
				factor := 1 - shadow
				if mood != nil {
					factor = 1 + clampMood(shadow*mood.ShadowStrength)*(mood.ShadowColor[k]-1)
				}
				dst.Pix[i+k] = uint8(math.Round(math.Max(0, math.Min(float64(dst.Pix[i+3]), float64(dst.Pix[i+k])*factor))))
			}
		}
	}
	return count
}
