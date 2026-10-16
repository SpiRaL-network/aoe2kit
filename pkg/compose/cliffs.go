package compose

import (
	"fmt"
	"math"
)

type Axis string

const (
	AxisX Axis = "x"
	AxisY Axis = "y"
)

// CliffFacets follows the observed camera-facing convention: +x slabs face +y,
// +y slabs face -x. High ground on the near side uses lips, not reverse faces.
// Evidence: cliff facet preview comparisons; not a terrain-height operation.
func CliffFacets(axis Axis, highSide int) ([]int, error) {
	if highSide != -1 && highSide != 1 {
		return nil, fmt.Errorf("high side must be -1 or +1")
	}
	switch axis {
	case AxisX:
		if highSide < 0 {
			return []int{4, 5}, nil
		}
		return []int{20, 21, 22, 23}, nil
	case AxisY:
		if highSide > 0 {
			return []int{1, 2}, nil
		}
		return []int{7, 8, 9, 10, 11}, nil
	default:
		return nil, fmt.Errorf("axis must be x or y")
	}
}

type WallOptions struct {
	UnitID   int
	Axis     Axis
	HighSide int
	// Mortar palettes are explicit so rock species match the wall material.
	Mortar []Brush
}

// CliffWall lays 1-2 slabs at 3-tile pitch separated by 1.5-2.5-tile joints.
// Position may bend the run; excessive offsets cannot make rigid slabs curve.
// Evidence: preview-reviewed bricks-and-mortar walls; joints remain art recipes.
func (s *Scene) CliffWall(from, to float64, position func(float64) Point, o WallOptions) {
	if s.err != nil {
		return
	}
	facets, err := CliffFacets(o.Axis, o.HighSide)
	if err != nil {
		s.fail(err.Error())
		return
	}
	if position == nil || !finite(from, to) || to < from || to-from > 4096 || len(o.Mortar) == 0 {
		s.fail("invalid wall range, position or mortar palette")
		return
	}
	for t := from; t <= to && s.err == nil; {
		for n := 1 + s.Random.Intn(2); n > 0 && t <= to; n-- {
			p := position(t)
			s.stamp(Brush{o.UnitID, facets}, p)
			s.Claim(p, 2.2)
			t += 3
		}
		if t > to {
			break
		}
		gap := 1.5 + s.Random.Float64()
		p := position(t - 1.5 + gap/2)
		s.stamp(o.Mortar[s.Random.Intn(len(o.Mortar))], p)
		for k, n := 0, 3+s.Random.Intn(3); k < n; k++ {
			s.stamp(o.Mortar[s.Random.Intn(len(o.Mortar))], Point{p.X + s.Random.Jitter(1.1), p.Y + s.Random.Jitter(.8)})
		}
		s.Claim(p, 1.8)
		t += gap
	}
}

type StairOptions struct {
	UnitID     int
	Steps      int
	GapStart   int // -1 disables the two-step gully
	GapTerrain int
	FlatChance float64
	EndCaps    bool
}

// LatticeStair uses the observed York corner lattice: v15/v3 on a three-tile
// (1,1) staircase, optional v4 flats, v12/v14 caps and a two-step gully.
// Evidence: campaign placement census and preview, not a walkability guarantee.
func (s *Scene) LatticeStair(start Point, o StairOptions) []Point {
	if s.err != nil {
		return nil
	}
	if o.Steps < 1 || o.Steps > 10000 || o.GapStart < -1 || o.GapStart >= o.Steps || !finite(o.FlatChance) || o.FlatChance < 0 || o.FlatChance > 1 {
		s.fail("invalid stair options")
		return nil
	}
	x, y := start.X, start.Y
	foot := []Point{}
	for k := 0; k < o.Steps && s.err == nil; k++ {
		if o.GapStart >= 0 && (k == o.GapStart || k == o.GapStart+1) {
			if k == o.GapStart {
				s.Gaia(o.UnitID, Point{x - 1.5, y + 1.5}, 12)
			} else {
				s.Gaia(o.UnitID, Point{x + 1.5, y + 1.5}, 14)
			}
			for gx := -1; gx <= 3; gx++ {
				for gy := 0; gy <= 4; gy++ {
					s.SetTerrain(int(x)+gx, int(y)+gy, o.GapTerrain)
				}
			}
			s.Claim(Point{x + 1, y + 2}, 3)
			x += 3
			y += 3
			continue
		}
		s.Gaia(o.UnitID, Point{x, y}, 15)
		s.Claim(Point{x, y}, 2.8)
		s.Gaia(o.UnitID, Point{x, y + 3}, 3)
		s.Claim(Point{x, y + 3}, 2.8)
		foot = append(foot, Point{x, y + 4.5})
		if s.Random.Float64() < o.FlatChance {
			x += 3
			s.Gaia(o.UnitID, Point{x, y + 3}, 4)
			s.Claim(Point{x, y + 3}, 2)
			foot = append(foot, Point{x, y + 4.5})
		}
		x += 3
		y += 3
	}
	if o.EndCaps && len(foot) > 0 {
		last := foot[len(foot)-1]
		for _, p := range []Point{{start.X - 3, start.Y + 3}, {last.X + 3, last.Y - 4.5}} {
			s.Gaia(o.UnitID, p, 12)
			s.Gaia(o.UnitID, Point{p.X + 3, p.Y}, 14)
			s.Claim(Point{p.X + 1.5, p.Y}, 2.5)
		}
	}
	return foot
}

type TerraceOptions struct {
	UnitID   int
	Coverups [][]Brush
}

// Terrace lays straight runs of 2-3 v4/v5 pieces at rounded whole-tile levels.
// Coverup groups rotate at clean cuts and sit downhill (+y) of the face.
// Evidence: preview-reviewed built terraces; gradual offset slabs made stairs,
// so direction/height changes are discrete cuts disguised with matching art.
func (s *Scene) Terrace(from, to float64, level func(float64) float64, o TerraceOptions) []Point {
	if s.err != nil {
		return nil
	}
	if !finite(from, to) || to <= from || to-from > 4096 || level == nil || len(o.Coverups) == 0 {
		s.fail("invalid terrace options")
		return nil
	}
	for _, group := range o.Coverups {
		if len(group) == 0 {
			s.fail("empty terrace coverup group")
			return nil
		}
	}
	cuts := []Point{}
	for x := from + 1.5; x <= to-1.5 && s.err == nil; {
		n := 2 + s.Random.Intn(2)
		y := math.Round(level(math.Min(x+1.5*float64(n-1), to-1.5)))
		for k := 0; k < n && x <= to-1.5; k++ {
			s.Gaia(o.UnitID, Point{x, y}, s.Random.Pick(4, 5))
			s.Claim(Point{x, y}, 1.6)
			x += 3
		}
		if x <= to-1.5 {
			p := Point{x - 1.5, y}
			group := o.Coverups[len(cuts)%len(o.Coverups)]
			for _, b := range group {
				s.stamp(b, Point{p.X + s.Random.Jitter(.8), p.Y + .6 + s.Random.Float64()*1.2})
			}
			s.Claim(p, 1.5)
			cuts = append(cuts, p)
		}
	}
	return cuts
}
