package preview

import (
	"fmt"
	"math"
)

// AroundCrop produces a half-open tile window clipped to the map.
func AroundCrop(s Snapshot, x, y, radius float64) (*Crop, error) {
	if math.IsNaN(x) || math.IsNaN(y) || math.IsNaN(radius) || math.IsInf(x, 0) || math.IsInf(y, 0) || math.IsInf(radius, 0) || radius <= 0 || x < 0 || y < 0 || x >= float64(s.Width) || y >= float64(s.Height) {
		return nil, fmt.Errorf("around needs an in-bounds finite point and positive finite radius")
	}
	return &Crop{int(math.Max(0, math.Floor(x-radius))), int(math.Max(0, math.Floor(y-radius))), int(math.Min(float64(s.Width), math.Ceil(x+radius))), int(math.Min(float64(s.Height), math.Ceil(y+radius)))}, nil
}

// FitUnitCrop bounds all matching objects, including their footprint and a
// one-tile margin. It does not silently choose one of several distant clusters.
func FitUnitCrop(s Snapshot, ids []int) (*Crop, error) {
	selected := map[int]bool{}
	for _, id := range ids {
		selected[id] = true
	}
	x1, y1, x2, y2 := float64(s.Width), float64(s.Height), 0.0, 0.0
	found := false
	for _, o := range s.Objects {
		if !selected[o.UnitID] || o.X < 0 || o.Y < 0 || o.X >= float64(s.Width) || o.Y >= float64(s.Height) {
			continue
		}
		found = true
		x1, y1 = math.Min(x1, o.X-math.Max(0, o.FootprintX)-1), math.Min(y1, o.Y-math.Max(0, o.FootprintY)-1)
		x2, y2 = math.Max(x2, o.X+math.Max(0, o.FootprintX)+1), math.Max(y2, o.Y+math.Max(0, o.FootprintY)+1)
	}
	if !found {
		return nil, fmt.Errorf("fit-units: no matching in-bounds objects")
	}
	return &Crop{max(0, int(math.Floor(x1))), max(0, int(math.Floor(y1))), min(s.Width, int(math.Ceil(x2))), min(s.Height, int(math.Ceil(y2)))}, nil
}
