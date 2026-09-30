// Package glyph converts TrueType outlines into the canonical Gaia-art cell
// representation used by Kit's text and placement tooling.
package glyph

import (
	"fmt"
	"math"
	"os"
	"sort"

	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
)

type Cell struct {
	X                 int     `json:"x"`
	Y                 int     `json:"y"`
	MaterialUnitConst int     `json:"material_unit_const"`
	Rotation          float64 `json:"rotation"`
}

type Glyph struct {
	Char   string `json:"char"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
	Cells  []Cell `json:"cells"`
}

type Library struct {
	Format      string  `json:"format"`
	Source      string  `json:"source"`
	Material    int     `json:"material_unit_const"`
	Rotation    float64 `json:"rotation"`
	CellScale   float64 `json:"cell_scale"`
	SampleSpace float64 `json:"sample_spacing"`
	Glyphs      []Glyph `json:"glyphs"`
}

type Options struct {
	Chars            []rune
	Material         int
	Rotation         float64
	CellScale        float64
	SampleSpacing    float64
	MaxCellsPerGlyph int
}

type point struct{ x, y float64 }

func DefaultOptions() Options {
	return Options{Chars: []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789 .,!?-"), Material: 1366, Rotation: 0, CellScale: 8, SampleSpacing: 0.6, MaxCellsPerGlyph: 4096}
}

func FromTTF(path string, opts Options) (Library, error) {
	if opts.CellScale <= 0 || opts.SampleSpacing <= 0 {
		return Library{}, fmt.Errorf("cell scale and sample spacing must be positive")
	}
	if opts.MaxCellsPerGlyph <= 0 {
		opts.MaxCellsPerGlyph = 4096
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return Library{}, err
	}
	font, err := sfnt.Parse(data)
	if err != nil {
		return Library{}, fmt.Errorf("parse TrueType %q: %w", path, err)
	}
	if len(opts.Chars) == 0 {
		opts.Chars = DefaultOptions().Chars
	}
	seen := map[rune]bool{}
	result := Library{Format: "aoe2kit-gaia-glyphs-v1", Source: path, Material: opts.Material, Rotation: opts.Rotation, CellScale: opts.CellScale, SampleSpace: opts.SampleSpacing}
	for _, r := range opts.Chars {
		if seen[r] {
			continue
		}
		seen[r] = true
		g, err := rasterGlyph(font, r, opts)
		if err != nil {
			return Library{}, err
		}
		result.Glyphs = append(result.Glyphs, g)
	}
	sort.SliceStable(result.Glyphs, func(i, j int) bool { return result.Glyphs[i].Char < result.Glyphs[j].Char })
	return result, nil
}

func rasterGlyph(font *sfnt.Font, r rune, opts Options) (Glyph, error) {
	var buf sfnt.Buffer
	index, err := font.GlyphIndex(&buf, r)
	if err != nil {
		return Glyph{}, err
	}
	ppem := fixed.Int26_6(font.UnitsPerEm() << 6)
	segments, err := font.LoadGlyph(&buf, index, ppem, nil)
	if err != nil {
		return Glyph{}, err
	}
	metrics, err := font.Metrics(&buf, ppem, 0)
	if err != nil {
		return Glyph{}, err
	}
	pointFn := func(p fixed.Point26_6) point {
		return point{x: float64(p.X) / 64 / float64(font.UnitsPerEm()) * opts.CellScale, y: (float64(metrics.Ascent)/64 + float64(p.Y)/64) / float64(font.UnitsPerEm()) * opts.CellScale}
	}
	var contours [][]point
	var contour []point
	var current point
	flush := func() {
		if len(contour) > 1 {
			if contour[0] != contour[len(contour)-1] {
				contour = append(contour, contour[0])
			}
			contours = append(contours, contour)
		}
		contour = nil
	}
	for _, segment := range segments {
		switch segment.Op {
		case sfnt.SegmentOpMoveTo:
			flush()
			current = pointFn(segment.Args[0])
			contour = append(contour, current)
		case sfnt.SegmentOpLineTo:
			current = pointFn(segment.Args[0])
			contour = append(contour, current)
		case sfnt.SegmentOpQuadTo:
			control, end := pointFn(segment.Args[0]), pointFn(segment.Args[1])
			for i := 1; i <= 12; i++ {
				contour = append(contour, quadratic(current, control, end, float64(i)/12))
			}
			current = end
		case sfnt.SegmentOpCubeTo:
			c1, c2, end := pointFn(segment.Args[0]), pointFn(segment.Args[1]), pointFn(segment.Args[2])
			for i := 1; i <= 16; i++ {
				contour = append(contour, cubic(current, c1, c2, end, float64(i)/16))
			}
			current = end
		}
	}
	flush()
	var samples []point
	for _, c := range contours {
		samples = append(samples, sample(c, opts.SampleSpacing)...)
	}
	if len(samples) == 0 {
		return Glyph{Char: string(r), Width: 1, Height: 1}, nil
	}
	minX, minY, maxX, maxY := samples[0].x, samples[0].y, samples[0].x, samples[0].y
	for _, p := range samples {
		minX, minY = math.Min(minX, p.x), math.Min(minY, p.y)
		maxX, maxY = math.Max(maxX, p.x), math.Max(maxY, p.y)
	}
	occupied := map[[2]int]bool{}
	for _, p := range samples {
		x := int(math.Round(p.x - minX))
		y := int(math.Round(maxY - p.y))
		occupied[[2]int{x, y}] = true
	}
	if len(occupied) > opts.MaxCellsPerGlyph {
		return Glyph{}, fmt.Errorf("glyph %q has %d cells, limit %d", r, len(occupied), opts.MaxCellsPerGlyph)
	}
	cells := make([]Cell, 0, len(occupied))
	for key := range occupied {
		cells = append(cells, Cell{X: key[0], Y: key[1], MaterialUnitConst: opts.Material, Rotation: opts.Rotation})
	}
	sort.Slice(cells, func(i, j int) bool {
		if cells[i].Y == cells[j].Y {
			return cells[i].X < cells[j].X
		}
		return cells[i].Y < cells[j].Y
	})
	return Glyph{Char: string(r), Width: int(math.Round(maxX-minX)) + 1, Height: int(math.Round(maxY-minY)) + 1, Cells: cells}, nil
}

func quadratic(a, b, c point, t float64) point {
	u := 1 - t
	return point{u*u*a.x + 2*u*t*b.x + t*t*c.x, u*u*a.y + 2*u*t*b.y + t*t*c.y}
}
func cubic(a, b, c, d point, t float64) point {
	u := 1 - t
	return point{u*u*u*a.x + 3*u*u*t*b.x + 3*u*t*t*c.x + t*t*t*d.x, u*u*u*a.y + 3*u*u*t*b.y + 3*u*t*t*c.y + t*t*t*d.y}
}
func sample(poly []point, spacing float64) []point {
	if len(poly) < 2 {
		return nil
	}
	out := []point{poly[0]}
	next := spacing
	for i := 1; i < len(poly); i++ {
		a, b := poly[i-1], poly[i]
		length := math.Hypot(b.x-a.x, b.y-a.y)
		for length >= next && length > 0 {
			ratio := next / length
			p := point{a.x + (b.x-a.x)*ratio, a.y + (b.y-a.y)*ratio}
			out = append(out, p)
			a = p
			length = math.Hypot(b.x-a.x, b.y-a.y)
			next = spacing
		}
		next -= length
	}
	return out
}
