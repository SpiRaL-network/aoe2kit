package compose

import "math"

type RiverPoint struct {
	Point
	HalfWidth float64
}
type Pool struct {
	Point
	Radius float64
}
type RiverOptions struct {
	Core, Shallow, Bank, Deep int
	BankWidth                 float64
	Pools                     []Pool
}

func validTerrain(ids ...int) bool {
	for _, id := range ids {
		if id < 0 || id > 255 {
			return false
		}
	}
	return true
}

func segmentProjection(p, a, b Point) (float64, float64) {
	dx, dy := b.X-a.X, b.Y-a.Y
	d := dx*dx + dy*dy
	t := 0.0
	if d > 0 {
		t = math.Max(0, math.Min(1, ((p.X-a.X)*dx+(p.Y-a.Y)*dy)/d))
	}
	return math.Hypot(p.X-a.X-t*dx, p.Y-a.Y-t*dy), t
}

// River paints a variable-width polyline with core (55%), shallow and bank,
// then round plunge pools (60% deep core). Rasterization uses tile centres and
// strongest water coverage at crossings, replacing the generators' overlapping
// half-tile stamps. Evidence: preview-derived narrow streams spread at blended
// borders; this controls source terrain, not the engine's visible water width.
func (s *Scene) River(points []RiverPoint, o RiverOptions) {
	if s.err != nil {
		return
	}
	if len(points) < 2 || len(points) > 10000 || !validTerrain(o.Core, o.Shallow, o.Bank, o.Deep) || !finite(o.BankWidth) || o.BankWidth < 0 || o.BankWidth > 480 {
		s.fail("invalid river options")
		return
	}
	for _, p := range points {
		if !s.inside(p.X, p.Y) || !finite(p.HalfWidth) || p.HalfWidth <= 0 || p.HalfWidth > 480 {
			s.fail("invalid river point")
			return
		}
	}
	for _, p := range o.Pools {
		if !s.inside(p.X, p.Y) || !finite(p.Radius) || p.Radius <= 0 || p.Radius > 480 {
			s.fail("invalid plunge pool")
			return
		}
	}
	coverage := make([]byte, len(s.terrain))
	for i := 0; i+1 < len(points); i++ {
		a, b := points[i], points[i+1]
		r := math.Max(a.HalfWidth, b.HalfWidth) + o.BankWidth
		x0, x1 := max(0, int(math.Floor(math.Min(a.X, b.X)-r))), min(s.width, int(math.Ceil(math.Max(a.X, b.X)+r)))
		y0, y1 := max(0, int(math.Floor(math.Min(a.Y, b.Y)-r))), min(s.height, int(math.Ceil(math.Max(a.Y, b.Y)+r)))
		for y := y0; y < y1; y++ {
			for x := x0; x < x1; x++ {
				d, t := segmentProjection(Point{float64(x) + .5, float64(y) + .5}, a.Point, b.Point)
				hw := a.HalfWidth + (b.HalfWidth-a.HalfWidth)*t
				v := byte(0)
				switch {
				case d < hw*.55:
					v = 3
				case d < hw:
					v = 2
				case d < hw+o.BankWidth:
					v = 1
				}
				j := y*s.width + x
				coverage[j] = max(coverage[j], v)
			}
		}
	}
	for j, v := range coverage {
		switch v {
		case 1:
			s.terrain[j] = o.Bank
		case 2:
			s.terrain[j] = o.Shallow
		case 3:
			s.terrain[j] = o.Core
		}
		if v > 0 {
			s.Claim(Point{float64(j%s.width) + .5, float64(j/s.width) + .5}, .71)
		}
	}
	for _, p := range o.Pools {
		for y := max(0, int(math.Floor(p.Y-p.Radius))); y < min(s.height, int(math.Ceil(p.Y+p.Radius))); y++ {
			for x := max(0, int(math.Floor(p.X-p.Radius))); x < min(s.width, int(math.Ceil(p.X+p.Radius))); x++ {
				d := math.Hypot(float64(x)+.5-p.X, float64(y)+.5-p.Y)
				if d < p.Radius*.6 {
					s.SetTerrain(x, y, o.Deep)
				} else if d < p.Radius {
					s.SetTerrain(x, y, o.Core)
				}
			}
		}
		s.Claim(p.Point, p.Radius+1)
	}
}

type TrackOptions struct {
	Dry, Mud                                    int
	HalfWidth                                   float64
	NoiseSeed                                   int
	WetThreshold, MissingBoards, DryBoardChance float64
	RoadID                                      int
	AvoidTerrains                               []int
}

// MixedTrack uses dirt/mud FBM and sparse square/half-board art. Evidence:
// preview-reviewed mixed-character tracks; the caller supplies water terrain
// exclusions and probabilities rather than inheriting a location-specific list.
func (s *Scene) MixedTrack(points []Point, o TrackOptions) {
	if s.err != nil {
		return
	}
	if len(points) < 2 || len(points) > 10000 || !validTerrain(o.Dry, o.Mud) || !finite(o.HalfWidth, o.WetThreshold, o.MissingBoards, o.DryBoardChance) || o.HalfWidth <= 0 || o.HalfWidth > 480 || o.WetThreshold < 0 || o.WetThreshold > 1 || o.MissingBoards < 0 || o.MissingBoards > 1 || o.DryBoardChance < 0 || o.DryBoardChance > 1 {
		s.fail("invalid mixed track options")
		return
	}
	for _, p := range points {
		if !s.inside(p.X, p.Y) {
			s.fail("invalid track point")
			return
		}
	}
	avoid := map[int]bool{}
	for _, id := range o.AvoidTerrains {
		avoid[id] = true
	}
	paint := make([]bool, len(s.terrain))
	centre := make([]bool, len(s.terrain))
	for i := 0; i+1 < len(points); i++ {
		a, b := points[i], points[i+1]
		r := o.HalfWidth
		for y := max(0, int(math.Floor(math.Min(a.Y, b.Y)-r))); y < min(s.height, int(math.Ceil(math.Max(a.Y, b.Y)+r))); y++ {
			for x := max(0, int(math.Floor(math.Min(a.X, b.X)-r))); x < min(s.width, int(math.Ceil(math.Max(a.X, b.X)+r))); x++ {
				d, _ := segmentProjection(Point{float64(x) + .5, float64(y) + .5}, a, b)
				j := y*s.width + x
				if d < r && !avoid[s.terrain[j]] {
					paint[j] = true
					if d <= .71 {
						centre[j] = true
					}
				}
			}
		}
	}
	for j, yes := range paint {
		if yes {
			if FBM(float64(j%s.width)/4, float64(j/s.width)/4, o.NoiseSeed) > o.WetThreshold {
				s.terrain[j] = o.Mud
			} else {
				s.terrain[j] = o.Dry
			}
		}
	}
	for j, yes := range centre {
		if !yes {
			continue
		}
		p := Point{float64(j%s.width) + .5, float64(j/s.width) + .5}
		if s.terrain[j] == o.Mud {
			if s.Random.Float64() >= o.MissingBoards {
				WoodenRoad{s, o.RoadID}.Square(p)
			}
		} else if s.Random.Float64() < o.DryBoardChance {
			s.Gaia(o.RoadID, Point{p.X + s.Random.Jitter(.3), p.Y + s.Random.Jitter(.3)}, s.Random.Pick(8, 9, 10, 11))
			s.Claim(p, 1.1)
		}
	}
}

// WaterfallPair places background and overlay at one lip. Angle and anchor are
// explicit because the particle placement convention is not engine-calibrated.
func (s *Scene) WaterfallPair(p Point, background, overlay, angle int) {
	s.Gaia(background, p, angle)
	s.Gaia(overlay, p, angle)
}
