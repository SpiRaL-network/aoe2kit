package preview

import (
	"math"
	"testing"
)

func TestHeightVertexLightPreservesPlanarSlope(t *testing.T) {
	s := Snapshot{Schema: SnapshotSchema, Width: 7, Height: 7}
	for y := 0; y < 7; y++ {
		for x := 0; x < 7; x++ {
			s.Terrain = append(s.Terrain, Tile{X: x, Y: y, Elevation: float64(x)})
		}
	}
	tiles := heightTiles(s, nil)
	for y := 2; y <= 5; y++ {
		for x := 2; x <= 5; x++ {
			got := heightVertexLight(tiles, 7, 7, x, y)
			if math.Abs(got-heightLight(1, 0)) > 1e-12 {
				t.Fatalf("planar slope changed at %d,%d: %v", x, y, got)
			}
		}
	}
	for y := 0; y <= 7; y++ {
		for x := 0; x <= 7; x++ {
			v := heightVertexLight(tiles, 7, 7, x, y)
			if math.IsNaN(v) || math.IsInf(v, 0) {
				t.Fatalf("invalid boundary light %d,%d", x, y)
			}
		}
	}
}
