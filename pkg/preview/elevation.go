package preview

import (
	"image/color"
	"math"
)

// Central differences approximate a directional terrain light. Equal-height
// plateaus remain unchanged; this deliberately makes no geometry claim.
func slopeShade(c color.RGBA, tiles []Tile, width, height, x, y int) color.RGBA {
	e := func(x, y int) float64 {
		v := tiles[max(0, min(height-1, y))*width+max(0, min(width-1, x))].Elevation
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return 0
		}
		return v
	}
	dx, dy := (e(x+1, y)-e(x-1, y))*.5, (e(x, y+1)-e(x, y-1))*.5
	factor := math.Max(.55, math.Min(1.35, 1+(dx-dy)*.12))
	channel := func(v uint8) uint8 { return uint8(math.Min(255, math.Round(float64(v)*factor))) }
	return color.RGBA{channel(c.R), channel(c.G), channel(c.B), c.A}
}
