package preview

import "testing"

func TestHeightTileFoldSeparatesIsolatedCornersFromRidges(t *testing.T) {
	for _, tt := range []struct {
		name   string
		dx, dy int
		bd     bool
	}{{"NW", -1, -1, true}, {"NE", 1, -1, false}, {"SE", 1, 1, true}, {"SW retained", -1, 1, true}} {
		t.Run(tt.name, func(t *testing.T) {
			for _, base := range []float64{0, 2, 14} {
				s := Snapshot{Schema: SnapshotSchema, Width: 7, Height: 7}
				for y := 0; y < 7; y++ {
					for x := 0; x < 7; x++ {
						h := base
						if x == 3+tt.dx && y == 3+tt.dy {
							h++
						}
						s.Terrain = append(s.Terrain, Tile{X: x, Y: y, Elevation: h})
					}
				}
				tiles := heightTiles(s, nil)
				if got := heightTileFoldBD(tiles, 7, 7, 3, 3); got != tt.bd {
					t.Fatalf("base %g: fold=%v want%v", base, got, tt.bd)
				}
			}
		})
	}
	// This connected pyramid tile has one raised shared corner too. Its
	// surrounding lower terrain distinguishes it from the isolated case.
	s := heightSnapshot(9)
	tiles := heightTiles(s, nil)
	for _, xy := range [][2]int{{1, 1}, {2, 2}, {3, 3}} {
		x, y := xy[0], xy[1]
		a, b, c, d := cornerHeight(tiles, 9, 9, x, y), cornerHeight(tiles, 9, 9, x+1, y), cornerHeight(tiles, 9, 9, x+1, y+1), cornerHeight(tiles, 9, 9, x, y+1)
		if heightTileFoldBD(tiles, 9, 9, x, y) != heightFoldBD(a, b, c, d) {
			t.Fatal("isolated-corner rule cut a connected pyramid ridge", xy)
		}
	}
}
