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
// rule matches registered grid vertices in single-tile, pyramid, plateau,
// terrace and pit captures. Coverage is still limited to these grass fixtures.
func cornerHeight(tiles []Tile, width, height, x, y int) float64 {
	h := 0.0
	for yy := max(0, y-1); yy <= min(height-1, y); yy++ {
		for xx := max(0, x-1); xx <= min(width-1, x); xx++ {
			h = math.Max(h, tiles[yy*width+xx].Elevation)
		}
	}
	return h
}

func elevationPadding(s Snapshot, crop Crop, scale int, projection string, enabled bool, stored ...bool) int {
	if !enabled || defaultProjection(projection) != "diamond" {
		return 0
	}
	h := 0.0
	for _, t := range s.Terrain {
		if t.X >= max(0, crop.X1-1) && t.X <= min(s.Width-1, crop.X2) && t.Y >= max(0, crop.Y1-1) && t.Y <= min(s.Height-1, crop.Y2) {
			h = math.Max(h, t.Elevation)
		}
	}
	if len(stored) > 0 && stored[0] {
		for _, o := range s.Objects {
			if o.X >= float64(crop.X1) && o.X < float64(crop.X2) && o.Y >= float64(crop.Y1) && o.Y < float64(crop.Y2) {
				h = math.Max(h, o.Z)
			}
		}
	}
	return int(math.Ceil(h * float64(scale) / 4))
}

func validateStoredObjectHeights(s Snapshot) error {
	for _, o := range s.Objects {
		if math.IsNaN(o.Z) || math.IsInf(o.Z, 0) || o.Z < 0 || o.Z > 255 {
			return fmt.Errorf("stored object height must be finite and between 0 and 255")
		}
	}
	return nil
}

// SnapshotImageExtent adds the height-field headroom to the flat image frame.
// Optional stored=true also reserves headroom for recorded object Z values.
func SnapshotImageExtent(s Snapshot, scale int, projection string, elevation bool, stored ...bool) (ImageExtent, error) {
	if err := validateSnapshot(s); err != nil {
		return ImageExtent{}, err
	}
	if elevation {
		if len(stored) > 0 && stored[0] {
			if err := validateStoredObjectHeights(s); err != nil {
				return ImageExtent{}, err
			}
		}
		if err := validateHeightfield(s); err != nil {
			return ImageExtent{}, err
		}
	}
	e, err := FullImageExtent(s.Width, s.Height, scale, projection)
	if err != nil {
		return e, err
	}
	pad := elevationPadding(s, Crop{X2: s.Width, Y2: s.Height}, e.Scale, projection, elevation, stored...)
	e.Height += pad
	e.MapOriginY += pad
	return e, nil
}

type heightVertex struct{ x, y, h, px, py float64 }

func heightVertexAt(tiles []Tile, s Snapshot, crop Crop, scale, pad, x, y int) heightVertex {
	t := diamondTransform(crop.X2-crop.X1, crop.Y2-crop.Y1, float64(scale), float64(scale), "")
	h := cornerHeight(tiles, s.Width, s.Height, x, y)
	px, py := t.point(float64(x-crop.X1), float64(y-crop.Y1))
	return heightVertex{float64(x), float64(y), h, px, py + float64(pad) - h*float64(scale)/4}
}

// Lighting fitted to three grass faces in one editor capture, not engine maths.
// Least-squares fit to flat=1 and axial slopes +x=.68, -x=1.08,
// +y=.79, -y=1.21. Opposing-pair means differ in the capture (.88/1),
// so one Lambert-style term cannot match both exactly: axial residuals .06.
func heightLight(dx, dy float64) float64 {
	n := math.Sqrt(1 + dx*dx + dy*dy)
	return math.Max(.4, math.Min(1.5, .795147+(0.204853-.282843*dx-.296985*dy)/n))
}

func heightShade(c color.RGBA, factor float64) color.RGBA {
	channel := func(v uint8) uint8 { return uint8(math.Min(255, math.Round(float64(v)*factor))) }
	return color.RGBA{channel(c.R), channel(c.G), channel(c.B), c.A}
}

func heightFoldBD(a, b, c, d float64) bool {
	oddAC := (b == c && c == d && a != b) || (a == b && b == d && c != a)
	oddBD := (a == c && c == d && b != a) || (a == b && b == c && d != a)
	return oddBD || (!oddAC && math.Abs(a-c) > math.Abs(b-d))
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
		worldDet := (b.x-a.x)*(c.y-a.y) - (b.y-a.y)*(c.x-a.x)
		dx := ((b.h-a.h)*(c.y-a.y) - (c.h-a.h)*(b.y-a.y)) / worldDet
		dy := ((b.x-a.x)*(c.h-a.h) - (c.x-a.x)*(b.h-a.h)) / worldDet
		light := heightLight(dx, dy)
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
			// Fold through a lone raised/lowered corner, so diagonal ridges
			// stay continuous rather than alternating flat and sloped teeth.
			// Ties retain A-C; other ambiguous cases prefer the level diagonal.
			if heightFoldBD(a.h, b.h, c.h, d.h) {
				triangle(a, b, d)
				triangle(b, c, d)
			} else {
				triangle(a, b, c)
				triangle(a, c, d)
			}
		}
	}
}

func heightTiles(s Snapshot, session *renderSession) []Tile {
	if session != nil {
		return session.terrain
	}
	tiles := make([]Tile, len(s.Terrain))
	for _, t := range s.Terrain {
		tiles[t.Y*s.Width+t.X] = t
	}
	return tiles
}

// Bilinear attachment is provisional away from flat terraces: the object
// capture verifies height-two plateau attachment, not slope interpolation.
func objectGroundLift(s Snapshot, tiles []Tile, o Object, scale int, projection string, stored ...bool) int {
	if tiles == nil || defaultProjection(projection) != "diamond" || math.IsNaN(o.X) || math.IsNaN(o.Y) || o.X < 0 || o.Y < 0 || o.X >= float64(s.Width) || o.Y >= float64(s.Height) {
		return 0
	}
	if len(stored) > 0 && stored[0] {
		return int(math.Round(o.Z * float64(scale) / 4))
	}
	x, y := int(o.X), int(o.Y)
	u, v := o.X-float64(x), o.Y-float64(y)
	a := cornerHeight(tiles, s.Width, s.Height, x, y)
	b := cornerHeight(tiles, s.Width, s.Height, x+1, y)
	c := cornerHeight(tiles, s.Width, s.Height, x+1, y+1)
	d := cornerHeight(tiles, s.Width, s.Height, x, y+1)
	h := (1-v)*((1-u)*a+u*b) + v*((1-u)*d+u*c)
	return int(math.Round(h * float64(scale) / 4))
}

// groundCanvas translates object drawing into its padded, ground-attached frame.
func groundCanvas(img *image.RGBA, pad int) *image.RGBA {
	view := *img
	view.Rect = view.Rect.Sub(image.Pt(0, pad))
	return &view
}
