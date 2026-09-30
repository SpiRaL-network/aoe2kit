package geom

import (
	"fmt"
	"math"
	"sort"
)

const (
	MapMin          = 0
	MapMax          = 299
	DefaultMaxTiles = 40
)

type Tile struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type Frame []Tile

type PatternFunc func() []Frame

type PatternSpec struct {
	Name     string
	Generate PatternFunc
}

func InBounds(t Tile) bool {
	return MapMin <= t.X && t.X <= MapMax && MapMin <= t.Y && t.Y <= MapMax
}

func PatternNames() []string {
	names := make([]string, 0, len(Patterns))
	for name := range Patterns {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func Generate(name string) ([]Frame, error) {
	spec, ok := Patterns[name]
	if !ok {
		return nil, fmt.Errorf("unknown pattern %q", name)
	}
	return spec.Generate(), nil
}

type Validation struct {
	Frames      int
	MaxTiles    int
	TotalTiles  int
	EmptyFrames int
	Errors      []string
}

func Validate(frames []Frame) Validation {
	v := Validation{Frames: len(frames)}
	if len(frames) < 8 || len(frames) > 48 {
		v.Errors = append(v.Errors, fmt.Sprintf("frame count %d outside 8..48", len(frames)))
	}
	for i, frame := range frames {
		if len(frame) == 0 {
			v.EmptyFrames++
			v.Errors = append(v.Errors, fmt.Sprintf("frame %d is empty", i))
		}
		if len(frame) > v.MaxTiles {
			v.MaxTiles = len(frame)
		}
		if len(frame) > DefaultMaxTiles {
			v.Errors = append(v.Errors, fmt.Sprintf("frame %d has %d tiles, exceeds %d", i, len(frame), DefaultMaxTiles))
		}
		seen := map[Tile]struct{}{}
		for _, tile := range frame {
			v.TotalTiles++
			if !InBounds(tile) {
				v.Errors = append(v.Errors, fmt.Sprintf("frame %d tile out of bounds: %+v", i, tile))
			}
			if _, ok := seen[tile]; ok {
				v.Errors = append(v.Errors, fmt.Sprintf("frame %d duplicate tile: %+v", i, tile))
			}
			seen[tile] = struct{}{}
		}
	}
	return v
}

func tile(x, y float64) Tile {
	return Tile{X: int(math.Round(x)), Y: int(math.Round(y))}
}

func frame(points []Tile, maxTiles int) Frame {
	if maxTiles <= 0 {
		maxTiles = DefaultMaxTiles
	}
	seen := map[Tile]struct{}{}
	out := make(Frame, 0, len(points))
	for _, point := range points {
		if _, ok := seen[point]; ok || !InBounds(point) {
			continue
		}
		seen[point] = struct{}{}
		out = append(out, point)
	}
	return thin(out, maxTiles)
}

func thin(points Frame, maxTiles int) Frame {
	if len(points) <= maxTiles {
		return points
	}
	if maxTiles <= 0 {
		return nil
	}
	if maxTiles == 1 {
		return Frame{points[0]}
	}
	out := make(Frame, 0, maxTiles)
	step := float64(len(points)-1) / float64(maxTiles-1)
	for i := 0; i < maxTiles; i++ {
		out = append(out, points[int(math.Round(float64(i)*step))])
	}
	return out
}

func circle(cx, cy, r float64, samples int, phase float64, maxTiles int) Frame {
	if r <= 0 {
		return frame([]Tile{tile(cx, cy)}, maxTiles)
	}
	if samples < 8 {
		samples = 8
	}
	if samples > maxTiles {
		samples = maxTiles
	}
	points := make([]Tile, 0, samples)
	for i := 0; i < samples; i++ {
		a := phase + 2*math.Pi*float64(i)/float64(samples)
		points = append(points, tile(cx+math.Cos(a)*r, cy+math.Sin(a)*r))
	}
	return frame(points, maxTiles)
}

func radialLine(cx, cy int, angle float64, length int, maxTiles int) Frame {
	points := make([]Tile, 0, length+1)
	for i := 0; i <= length; i++ {
		points = append(points, tile(float64(cx)+math.Cos(angle)*float64(i), float64(cy)+math.Sin(angle)*float64(i)))
	}
	return frame(points, maxTiles)
}

func ExpandingRing(cx, cy, rMax int) []Frame {
	frames := make([]Frame, 0, rMax)
	for r := 1; r <= rMax; r++ {
		frames = append(frames, circle(float64(cx), float64(cy), float64(r), 32, 0, DefaultMaxTiles))
	}
	return frames
}

func CollapsingRing(cx, cy, rMax int) []Frame {
	frames := ExpandingRing(cx, cy, rMax)
	for i, j := 0, len(frames)-1; i < j; i, j = i+1, j-1 {
		frames[i], frames[j] = frames[j], frames[i]
	}
	return frames
}

func DiamondPulse(cx, cy, rMax int) []Frame {
	frames := make([]Frame, 0, rMax)
	for r := 1; r <= rMax; r++ {
		points := make([]Tile, 0, r*4)
		for dx := -r; dx <= r; dx++ {
			dy := r - abs(dx)
			points = append(points, Tile{X: cx + dx, Y: cy + dy})
			if dy != 0 {
				points = append(points, Tile{X: cx + dx, Y: cy - dy})
			}
		}
		frames = append(frames, frame(points, DefaultMaxTiles))
	}
	return frames
}

func DoubleRing(cx, cy, rMax, steps int) []Frame {
	frames := make([]Frame, 0, steps)
	for i := 0; i < steps; i++ {
		r1 := 1 + i%rMax
		r2 := 1 + (i+rMax/2)%rMax
		points := append([]Tile{}, circle(float64(cx), float64(cy), float64(r1), 20, 0, 20)...)
		points = append(points, circle(float64(cx), float64(cy), float64(r2), 20, math.Pi/20, 20)...)
		frames = append(frames, frame(points, DefaultMaxTiles))
	}
	return frames
}

func LineSweep(x0, y0, w, h int, axis string) []Frame {
	var frames []Frame
	if axis == "y" {
		frames = make([]Frame, 0, h)
		for y := y0; y < y0+h; y++ {
			points := make([]Tile, 0, w)
			for x := x0; x < x0+w; x++ {
				points = append(points, Tile{X: x, Y: y})
			}
			frames = append(frames, frame(points, DefaultMaxTiles))
		}
		return frames
	}
	frames = make([]Frame, 0, w)
	for x := x0; x < x0+w; x++ {
		points := make([]Tile, 0, h)
		for y := y0; y < y0+h; y++ {
			points = append(points, Tile{X: x, Y: y})
		}
		frames = append(frames, frame(points, DefaultMaxTiles))
	}
	return frames
}

func RadarSweep(cx, cy, r, steps int) []Frame {
	frames := make([]Frame, 0, steps)
	for i := 0; i < steps; i++ {
		frames = append(frames, radialLine(cx, cy, 2*math.Pi*float64(i)/float64(steps), r, DefaultMaxTiles))
	}
	return frames
}

func CheckerboardSweep(x0, y0, w, h, steps int) []Frame {
	frames := make([]Frame, 0, steps)
	for phase := 0; phase < steps; phase++ {
		points := []Tile{}
		for y := y0; y < y0+h; y++ {
			for x := x0; x < x0+w; x++ {
				if (x+y+phase)%4 == 0 {
					points = append(points, Tile{X: x, Y: y})
				}
			}
		}
		frames = append(frames, frame(points, DefaultMaxTiles))
	}
	return frames
}

func SineScroll(x0, y0, w int, amp, wavelength float64, steps int) []Frame {
	frames := make([]Frame, 0, steps)
	for phase := 0; phase < steps; phase++ {
		points := make([]Tile, 0, w)
		for dx := 0; dx < w; dx++ {
			a := 2 * math.Pi * float64(dx+phase) / wavelength
			points = append(points, tile(float64(x0+dx), float64(y0)+math.Sin(a)*amp))
		}
		frames = append(frames, frame(points, DefaultMaxTiles))
	}
	return frames
}

func Spiral(cx, cy int, turns, r float64, steps, tail int) []Frame {
	frames := make([]Frame, 0, steps)
	history := []Tile{}
	for i := 0; i < steps; i++ {
		t := float64(i) / float64(max(1, steps-1))
		a := 2 * math.Pi * turns * t
		rr := r * t
		history = append(history, tile(float64(cx)+math.Cos(a)*rr, float64(cy)+math.Sin(a)*rr))
		frames = append(frames, frame(last(history, tail), DefaultMaxTiles))
	}
	return frames
}

func Lissajous(cx, cy int, rx, ry float64, a, b, steps, tail int) []Frame {
	frames := make([]Frame, 0, steps)
	history := []Tile{}
	for i := 0; i < steps; i++ {
		t := 2 * math.Pi * float64(i) / float64(steps)
		history = append(history, tile(float64(cx)+rx*math.Sin(float64(a)*t+math.Pi/2), float64(cy)+ry*math.Sin(float64(b)*t)))
		frames = append(frames, frame(last(history, tail), DefaultMaxTiles))
	}
	return frames
}

func DNAHelix(x0, y0, length int, amp, wavelength float64, steps int) []Frame {
	frames := make([]Frame, 0, steps)
	for phase := 0; phase < steps; phase++ {
		points := make([]Tile, 0, length*2)
		for dx := 0; dx < length; dx++ {
			a := 2 * math.Pi * float64(dx+phase) / wavelength
			points = append(points, tile(float64(x0+dx), float64(y0)+math.Sin(a)*amp))
			points = append(points, tile(float64(x0+dx), float64(y0)-math.Sin(a)*amp))
		}
		frames = append(frames, frame(points, DefaultMaxTiles))
	}
	return frames
}

func OrbitingDots(cx, cy int, r float64, n, steps int) []Frame {
	frames := make([]Frame, 0, steps)
	for i := 0; i < steps; i++ {
		points := make([]Tile, 0, n)
		base := 2 * math.Pi * float64(i) / float64(steps)
		for j := 0; j < n; j++ {
			a := base + 2*math.Pi*float64(j)/float64(n)
			points = append(points, tile(float64(cx)+math.Cos(a)*r, float64(cy)+math.Sin(a)*r))
		}
		frames = append(frames, frame(points, DefaultMaxTiles))
	}
	return frames
}

func RotatingPolygon(cx, cy int, r float64, sides, steps int) []Frame {
	frames := make([]Frame, 0, steps)
	sideSamples := max(2, DefaultMaxTiles/max(3, sides))
	for i := 0; i < steps; i++ {
		phase := 2 * math.Pi * float64(i) / float64(steps)
		verts := make([]Tile, 0, sides)
		for j := 0; j < sides; j++ {
			a := phase + 2*math.Pi*float64(j)/float64(sides)
			verts = append(verts, tile(float64(cx)+math.Cos(a)*r, float64(cy)+math.Sin(a)*r))
		}
		points := []Tile{}
		for j, v1 := range verts {
			v2 := verts[(j+1)%len(verts)]
			for k := 0; k < sideSamples; k++ {
				t := float64(k) / float64(sideSamples)
				points = append(points, tile(float64(v1.X)+float64(v2.X-v1.X)*t, float64(v1.Y)+float64(v2.Y-v1.Y)*t))
			}
		}
		frames = append(frames, frame(points, DefaultMaxTiles))
	}
	return frames
}

func PulsingStar(cx, cy int, rMax float64, points, steps int) []Frame {
	frames := make([]Frame, 0, steps)
	for i := 0; i < steps; i++ {
		pulse := 0.55 + 0.45*(0.5+0.5*math.Sin(2*math.Pi*float64(i)/float64(steps)))
		outer := rMax * pulse
		inner := outer * 0.42
		total := points * 2
		pts := make([]Tile, 0, total)
		for j := 0; j < total; j++ {
			rr := inner
			if j%2 == 0 {
				rr = outer
			}
			a := -math.Pi/2 + 2*math.Pi*float64(j)/float64(total)
			pts = append(pts, tile(float64(cx)+math.Cos(a)*rr, float64(cy)+math.Sin(a)*rr))
		}
		frames = append(frames, frame(pts, DefaultMaxTiles))
	}
	return frames
}

func BouncingDot(x0, y0, w, h, steps, trail int) []Frame {
	frames := make([]Frame, 0, steps)
	history := []Tile{}
	x, y := x0, y0
	vx, vy := 1, 1
	for i := 0; i < steps; i++ {
		history = append(history, Tile{X: x, Y: y})
		frames = append(frames, frame(last(history, trail), DefaultMaxTiles))
		x += vx
		y += vy
		if x <= x0 || x >= x0+w-1 {
			vx *= -1
		}
		if y <= y0 || y >= y0+h-1 {
			vy *= -1
		}
	}
	return frames
}

func Snake(x0, y0, w, h, steps, length int) []Frame {
	path := make([]Tile, 0, w*h)
	for row := 0; row < h; row++ {
		if row%2 == 0 {
			for x := x0; x < x0+w; x++ {
				path = append(path, Tile{X: x, Y: y0 + row})
			}
			continue
		}
		for x := x0 + w - 1; x >= x0; x-- {
			path = append(path, Tile{X: x, Y: y0 + row})
		}
	}
	frames := make([]Frame, 0, steps)
	for i := 0; i < steps; i++ {
		points := make([]Tile, 0, length)
		for j := 0; j < length; j++ {
			points = append(points, path[(i+j)%len(path)])
		}
		frames = append(frames, frame(points, DefaultMaxTiles))
	}
	return frames
}

func RippleGrid(x0, y0, w, h, cx, cy, steps int) []Frame {
	frames := make([]Frame, 0, steps)
	for phase := 0; phase < steps; phase++ {
		points := []Tile{}
		for y := y0; y < y0+h; y++ {
			for x := x0; x < x0+w; x++ {
				d := int(math.Round(math.Hypot(float64(x-cx), float64(y-cy))))
				if (d-phase)%steps == 0 {
					points = append(points, Tile{X: x, Y: y})
				}
			}
		}
		frames = append(frames, frame(points, DefaultMaxTiles))
	}
	return frames
}

func Pinwheel(cx, cy, r, arms, steps int) []Frame {
	frames := make([]Frame, 0, steps)
	for i := 0; i < steps; i++ {
		points := []Tile{}
		base := 2 * math.Pi * float64(i) / float64(steps)
		for arm := 0; arm < arms; arm++ {
			a := base + 2*math.Pi*float64(arm)/float64(arms)
			points = append(points, radialLine(cx, cy, a, r, max(1, DefaultMaxTiles/arms))...)
		}
		frames = append(frames, frame(points, DefaultMaxTiles))
	}
	return frames
}

func FallingComets(x0, y0, w, h, comets, steps, trail int) []Frame {
	frames := make([]Frame, 0, steps)
	for i := 0; i < steps; i++ {
		points := []Tile{}
		for c := 0; c < comets; c++ {
			headX := x0 + (c*w/comets+i)%w
			headY := y0 + (i*2+c*5)%h
			for t := 0; t < trail; t++ {
				points = append(points, Tile{X: headX - t, Y: headY - t})
			}
		}
		frames = append(frames, frame(points, DefaultMaxTiles))
	}
	return frames
}

var Patterns = map[string]PatternSpec{
	"bouncing_dot":       {Name: "bouncing_dot", Generate: func() []Frame { return BouncingDot(12, 12, 36, 36, 32, 5) }},
	"checkerboard_sweep": {Name: "checkerboard_sweep", Generate: func() []Frame { return CheckerboardSweep(15, 15, 30, 30, 8) }},
	"collapsing_ring":    {Name: "collapsing_ring", Generate: func() []Frame { return CollapsingRing(30, 30, 14) }},
	"diamond_pulse":      {Name: "diamond_pulse", Generate: func() []Frame { return DiamondPulse(30, 30, 14) }},
	"dna_helix":          {Name: "dna_helix", Generate: func() []Frame { return DNAHelix(12, 30, 36, 8, 14, 24) }},
	"double_ring":        {Name: "double_ring", Generate: func() []Frame { return DoubleRing(30, 30, 14, 24) }},
	"expanding_ring":     {Name: "expanding_ring", Generate: func() []Frame { return ExpandingRing(30, 30, 14) }},
	"falling_comets":     {Name: "falling_comets", Generate: func() []Frame { return FallingComets(12, 12, 36, 36, 5, 24, 5) }},
	"line_sweep_x":       {Name: "line_sweep_x", Generate: func() []Frame { return LineSweep(12, 12, 36, 36, "x") }},
	"line_sweep_y":       {Name: "line_sweep_y", Generate: func() []Frame { return LineSweep(12, 12, 36, 36, "y") }},
	"lissajous":          {Name: "lissajous", Generate: func() []Frame { return Lissajous(30, 30, 18, 12, 3, 2, 32, 8) }},
	"orbiting_dots":      {Name: "orbiting_dots", Generate: func() []Frame { return OrbitingDots(30, 30, 16, 6, 24) }},
	"pinwheel":           {Name: "pinwheel", Generate: func() []Frame { return Pinwheel(30, 30, 18, 4, 24) }},
	"pulsing_star":       {Name: "pulsing_star", Generate: func() []Frame { return PulsingStar(30, 30, 18, 5, 24) }},
	"radar_sweep":        {Name: "radar_sweep", Generate: func() []Frame { return RadarSweep(30, 30, 18, 24) }},
	"ripple_grid":        {Name: "ripple_grid", Generate: func() []Frame { return RippleGrid(12, 12, 36, 36, 30, 30, 16) }},
	"rotating_square":    {Name: "rotating_square", Generate: func() []Frame { return RotatingPolygon(30, 30, 16, 4, 24) }},
	"rotating_triangle":  {Name: "rotating_triangle", Generate: func() []Frame { return RotatingPolygon(30, 30, 17, 3, 24) }},
	"sine_scroll":        {Name: "sine_scroll", Generate: func() []Frame { return SineScroll(12, 30, 36, 9, 18, 24) }},
	"snake":              {Name: "snake", Generate: func() []Frame { return Snake(12, 12, 36, 18, 32, 12) }},
	"spiral":             {Name: "spiral", Generate: func() []Frame { return Spiral(30, 30, 3, 18, 32, 6) }},
}

func last(points []Tile, n int) []Tile {
	if n <= 0 || n >= len(points) {
		return append([]Tile{}, points...)
	}
	return append([]Tile{}, points[len(points)-n:]...)
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
