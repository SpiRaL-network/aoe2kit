package preview

import (
	"fmt"
	"image"
	"image/color"
	"math"
)

func validateHeightfield(s Snapshot) error {
	for _, t := range s.Terrain {
		if math.IsNaN(t.Elevation) || math.IsInf(t.Elevation, 0) || t.Elevation < 0 || t.Elevation > 255 {
			return fmt.Errorf("elevation must be finite and between 0 and 255")
		}
	}
	return nil
}

// Stored levels lift screen y by one quarter of a tile width. The shared-corner
// rule is provisional: it matches the single-tile and pyramid captures, but
// still needs independent pair/pit calibration.
func cornerHeight(tiles []Tile, width, height, x, y int) float64 {
	h := 0.0
	for yy := max(0, y-1); yy <= min(height-1, y); yy++ {
		for xx := max(0, x-1); xx <= min(width-1, x); xx++ {
			h = math.Max(h, tiles[yy*width+xx].Elevation)
		}
	}
	return h
}

func elevationPadding(s Snapshot, crop Crop, scale int, projection string, enabled bool) int {
	if !enabled || defaultProjection(projection) != "diamond" {
		return 0
	}
	h := 0.0
	for _, t := range s.Terrain {
		if t.X >= max(0, crop.X1-1) && t.X <= min(s.Width-1, crop.X2) && t.Y >= max(0, crop.Y1-1) && t.Y <= min(s.Height-1, crop.Y2) {
			h = math.Max(h, t.Elevation)
		}
	}
	return int(math.Ceil(h * float64(scale) / 4))
}

// SnapshotImageExtent adds the height-field headroom to the flat image frame.
func SnapshotImageExtent(s Snapshot, scale int, projection string, elevation bool) (ImageExtent, error) {
	if err := validateSnapshot(s); err != nil {
		return ImageExtent{}, err
	}
	if elevation {
		if err := validateHeightfield(s); err != nil {
			return ImageExtent{}, err
		}
	}
	e, err := FullImageExtent(s.Width, s.Height, scale, projection)
	if err != nil {
		return e, err
	}
	pad := elevationPadding(s, Crop{X2: s.Width, Y2: s.Height}, e.Scale, projection, elevation)
	e.Height += pad
	e.MapOriginY += pad
	return e, nil
}

type heightVertex struct{ x, y, h, px, py, light float64 }

func heightVertexAt(tiles []Tile, s Snapshot, crop Crop, scale, pad, x, y int) heightVertex {
	t := diamondTransform(crop.X2-crop.X1, crop.Y2-crop.Y1, float64(scale), float64(scale), "")
	h := cornerHeight(tiles, s.Width, s.Height, x, y)
	px, py := t.point(float64(x-crop.X1), float64(y-crop.Y1))
	x0, x1 := max(0, x-1), min(s.Width, x+1)
	y0, y1 := max(0, y-1), min(s.Height, y+1)
	dx := (cornerHeight(tiles, s.Width, s.Height, x1, y) - cornerHeight(tiles, s.Width, s.Height, x0, y)) / float64(x1-x0)
	dy := (cornerHeight(tiles, s.Width, s.Height, x, y1) - cornerHeight(tiles, s.Width, s.Height, x, y0)) / float64(y1-y0)
	return heightVertex{float64(x), float64(y), h, px, py + float64(pad) - h*float64(scale)/4, heightLight(dx, dy)}
}

// Lighting fitted to three grass faces in one editor capture, not engine maths.
// Opposite x slopes: .72/1.25; the brighter y slope: 1.13; flat: 1.
func heightLight(dx, dy float64) float64 {
	n := math.Sqrt(1 + dx*dx + dy*dy)
	return math.Max(.4, math.Min(1.5, .949+(0.051-.375*dx-.205*dy)/n))
}

func heightShade(c color.RGBA, factor float64) color.RGBA {
	channel := func(v uint8) uint8 { return uint8(math.Min(255, math.Round(float64(v)*factor))) }
	return color.RGBA{channel(c.R), channel(c.G), channel(c.B), c.A}
}

// Rasterize in back-to-front tile order. Texture/blend coordinates are the
// barycentrically interpolated original world positions, never screen pixels.
func renderHeightfield(img *image.RGBA, tiles []Tile, s Snapshot, crop Crop, scale, pad int, sample func(float64, float64) color.RGBA) {
	vertex := func(x, y int) heightVertex {
		return heightVertexAt(tiles, s, crop, scale, pad, x, y)
	}
	triangle := func(a, b, c heightVertex) {
		det := (b.px-a.px)*(c.py-a.py) - (b.py-a.py)*(c.px-a.px)
		if math.Abs(det) < 1e-9 {
			return
		}
		x0 := max(img.Rect.Min.X, int(math.Floor(math.Min(a.px, math.Min(b.px, c.px)))))
		x1 := min(img.Rect.Max.X, int(math.Ceil(math.Max(a.px, math.Max(b.px, c.px)))))
		y0 := max(img.Rect.Min.Y, int(math.Floor(math.Min(a.py, math.Min(b.py, c.py)))))
		y1 := min(img.Rect.Max.Y, int(math.Ceil(math.Max(a.py, math.Max(b.py, c.py)))))
		for py := y0; py < y1; py++ {
			for px := x0; px < x1; px++ {
				x, y := float64(px)+.5-a.px, float64(py)+.5-a.py
				u, v := (x*(c.py-a.py)-y*(c.px-a.px))/det, ((b.px-a.px)*y-(b.py-a.py)*x)/det
				if u < -1e-9 || v < -1e-9 || u+v > 1+1e-9 {
					continue
				}
				wx, wy := a.x+u*(b.x-a.x)+v*(c.x-a.x), a.y+u*(b.y-a.y)+v*(c.y-a.y)
				wx = math.Min(math.Nextafter(float64(crop.X2), 0), math.Max(float64(crop.X1), wx))
				wy = math.Min(math.Nextafter(float64(crop.Y2), 0), math.Max(float64(crop.Y1), wy))
				// Interpolate vertex lighting to avoid artificial flat-shaded
				// teeth at slope/plateau junctions. This smoothing is fitted,
				// not a claim about the engine's vertex-normal calculation.
				light := a.light + u*(b.light-a.light) + v*(c.light-a.light)
				img.SetRGBA(px, py, heightShade(sample(wx, wy), light))
			}
		}
	}
	for depth := crop.Y1 - (crop.X2 - 1); depth <= crop.Y2-1-crop.X1; depth++ {
		for x := crop.X1; x < crop.X2; x++ {
			y := x + depth
			if y < crop.Y1 || y >= crop.Y2 {
				continue
			}
			a, b, c, d := vertex(x, y), vertex(x+1, y), vertex(x+1, y+1), vertex(x, y+1)
			// Prefer the level diagonal when a lone corner rises or falls.
			// This triangulation remains part of the provisional corner model.
			if math.Abs(a.h-c.h) > math.Abs(b.h-d.h) {
				triangle(a, b, d)
				triangle(b, c, d)
			} else {
				triangle(a, b, c)
				triangle(a, c, d)
			}
		}
	}
}

// groundCanvas translates existing object drawing into the padded ground frame;
// it deliberately does not lift objects onto the terrain yet.
func groundCanvas(img *image.RGBA, pad int) *image.RGBA {
	view := *img
	view.Rect = view.Rect.Sub(image.Pt(0, pad))
	return &view
}
