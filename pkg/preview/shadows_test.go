package preview

import (
	"image"
	"image/color"
	"math"
	"testing"
)

func TestShadowOverlapAndClipping(t *testing.T) {
	src := image.NewAlpha(image.Rect(0, 0, 4, 4))
	for i := range src.Pix {
		src.Pix[i] = 80
	}
	for _, mode := range []string{"max", "sum"} {
		dst := image.NewAlpha(image.Rect(0, 0, 5, 5))
		for n := 0; n < 2; n++ {
			accumulateShadow(dst, src, image.Rect(-2, -2, 2, 2), mode)
		}
		want := uint8(80)
		if mode == "sum" {
			want = 160
		}
		if dst.AlphaAt(0, 0).A != want || dst.AlphaAt(2, 2).A != 0 {
			t.Fatalf("%s clip: %v", mode, dst.Pix)
		}
	}
	if _, err := shadowOverlap(ShadowOptions{Overlap: "over"}); err == nil {
		t.Fatal("invalid overlap accepted")
	}
}

func TestTerrainShadowMatchesShader(t *testing.T) {
	for _, m := range []*Mood{nil, {ShadowStrength: 1, ShadowColor: [3]float64{.1, .2, .3}, TerrainTint: [3]float64{1, 1, 1}}, {ShadowStrength: 3, ShadowColor: [3]float64{-.2, 0, .1}, TerrainTint: [3]float64{1, 1, 1}}} {
		for _, a := range []uint8{0, 40, 80, 96, 255} {
			im := image.NewRGBA(image.Rect(0, 0, 2, 1))
			im.SetRGBA(0, 0, color.RGBA{200, 180, 160, 255})
			coverage := image.NewAlpha(im.Bounds())
			coverage.SetAlpha(0, 0, color.Alpha{a})
			coverage.SetAlpha(1, 0, color.Alpha{a})
			applyShadows(im, coverage, m)
			mood := Mood{}
			if m != nil {
				mood = *m
			}
			want := ShaderTerrainGrade([3]float64{200. / 255, 180. / 255, 160. / 255}, [3]float64{1, 1, 1}, float64(a)/255, mood, m != nil, false, 0)
			for k, v := range want {
				expected := uint8(math.Round(clampMood(v) * 255))
				if im.Pix[k] != expected {
					t.Fatalf("alpha=%d channel=%d got=%d want=%d", a, k, im.Pix[k], expected)
				}
			}
			if im.RGBAAt(1, 0) != (color.RGBA{}) {
				t.Fatal("changed transparent terrain")
			}
		}
	}
}
