package preview

import (
	"image/color"
	"math"
	"testing"
)

func TestCropShortcuts(t *testing.T) {
	s := Snapshot{Width: 20, Height: 30, Objects: []Object{{UnitID: 1, X: .5, Y: .5}, {UnitID: 1, X: 10, Y: 15}, {UnitID: 2, X: 19, Y: 29}}}
	c, err := FitUnitCrop(s, []int{1})
	if err != nil || *c != (Crop{0, 0, 11, 16}) {
		t.Fatalf("%+v: %v", c, err)
	}
	if _, err := FitUnitCrop(s, []int{3}); err == nil {
		t.Fatal("accepted empty selection")
	}
	c, err = AroundCrop(s, 1, 2, 4)
	if err != nil || *c != (Crop{0, 0, 5, 6}) {
		t.Fatalf("%+v: %v", c, err)
	}
	for _, radius := range []float64{0, -1, math.NaN(), math.Inf(1)} {
		if _, err := AroundCrop(s, 1, 2, radius); err == nil {
			t.Fatal("accepted invalid radius")
		}
	}
	c, err = AroundCrop(s, 1, 2, math.MaxFloat64)
	if err != nil || *c != (Crop{0, 0, 20, 30}) {
		t.Fatalf("huge radius: %+v %v", c, err)
	}
}

func TestExperimentalSlopeShade(t *testing.T) {
	base := color.RGBA{100, 100, 100, 255}
	tiles := make([]Tile, 9)
	if got := slopeShade(base, tiles, 3, 3, 1, 1); got != base {
		t.Fatal(got)
	}
	tiles[5].Elevation = 4
	if got := slopeShade(base, tiles, 3, 3, 1, 1); got.R <= base.R {
		t.Fatal("slope did not lighten", got)
	}
	tiles[3].Elevation, tiles[5].Elevation = 4, 0
	if got := slopeShade(base, tiles, 3, 3, 1, 1); got.R >= base.R {
		t.Fatal("opposite slope did not darken", got)
	}
}
