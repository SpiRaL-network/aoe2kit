package preview

import (
	"math"
	"testing"
)

func TestStoredHeightObservedSaddleAndPit(t *testing.T) {
	// Editor-saved observations. The saddle's black marker was independently
	// registered in a screenshot: saved Z=0, observed lift within .05 px.
	for _, tt := range []struct {
		name     string
		grid     []float64
		x, y, z  float64
		inferred int
	}{
		{"fractional saddle", []float64{0, 0, 1, 0, 0, 0, 1, 0, 0}, 1.25, 1.75, 0, 11},
		{"ring pit oak", []float64{1, 1, 1, 1, 0, 1, 1, 1, 1}, 1.5, 1.5, .5, 18},
	} {
		t.Run(tt.name, func(t *testing.T) {
			s := Snapshot{Schema: SnapshotSchema, Width: 3, Height: 3}
			for i, h := range tt.grid {
				s.Terrain = append(s.Terrain, Tile{X: i % 3, Y: i / 3, Elevation: h})
			}
			o := Object{X: tt.x, Y: tt.y, Z: tt.z}
			tiles := heightTiles(s, nil)
			if got := objectGroundLift(s, tiles, o, 72, "diamond"); got != tt.inferred {
				t.Fatalf("default attachment changed: %d", got)
			}
			if got := objectGroundLift(s, tiles, o, 72, "diamond", true); got != int(math.Round(tt.z*18)) {
				t.Fatalf("stored attachment: %d", got)
			}
			if got := objectGroundLift(s, tiles, o, 72, "topdown", true); got != 0 {
				t.Fatalf("topdown lifted: %d", got)
			}
		})
	}
}

func TestStoredHeightFrameAndWindow(t *testing.T) {
	s := heightSnapshot(9)
	s.Objects = []Object{{UnitID: 306, X: 4, Y: 4, Z: 7}}
	opts := Options{Scale: 16, Elevation: true, StoredObjectHeights: true}
	full, r, e := RenderFlat(s, opts)
	if e != nil {
		t.Fatal(e)
	}
	ext, e := SnapshotImageExtent(s, 16, "diamond", true, true)
	if e != nil {
		t.Fatal(e)
	}
	if r.ElevationPadding != 28 || full.Bounds().Dy() != ext.Height {
		t.Fatalf("inconsistent stored-height frame: %+v %+v", r, ext)
	}
	opts.Window = &PixelWindow{X: 10, Y: 15, Width: 50, Height: 50}
	part, _, e := RenderFlat(s, opts)
	if e != nil {
		t.Fatal(e)
	}
	rect, _ := opts.Window.rectangle()
	assertWindowPixels(t, full, part, rect)
}

func TestStoredHeightRejectsInvalidZ(t *testing.T) {
	for _, z := range []float64{math.NaN(), math.Inf(1), -1, 256} {
		s := heightSnapshot(3)
		s.Objects = []Object{{X: 1, Y: 1, Z: z}}
		if _, _, e := RenderFlat(s, Options{Elevation: true, StoredObjectHeights: true}); e == nil {
			t.Fatalf("accepted %v", z)
		}
		if _, e := SnapshotImageExtent(s, 16, "diamond", true, true); e == nil {
			t.Fatalf("extent accepted %v", z)
		}
	}
}
