package preview

import (
	"image"
	"image/color"
	"math"
	"testing"
)

func heightSnapshot(size int) Snapshot {
	s := Snapshot{Schema: SnapshotSchema, Width: size, Height: size}
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			distance := math.Max(math.Abs(float64(x-size/2)), math.Abs(float64(y-size/2)))
			s.Terrain = append(s.Terrain, Tile{X: x, Y: y, Elevation: math.Max(0, 3-distance)})
		}
	}
	return s
}

func TestHeightCornersLiftAndExtent(t *testing.T) {
	s := heightSnapshot(9)
	for _, scale := range []int{8, 13, 16, 32, 64} {
		e, err := SnapshotImageExtent(s, scale, "diamond", true)
		flat, _ := SnapshotImageExtent(heightSnapshot(9), scale, "diamond", false)
		pad := int(math.Ceil(3 * float64(scale) / 4))
		if err != nil || e.Height-flat.Height != pad || e.MapOriginY-flat.MapOriginY != pad || e.Width != flat.Width {
			t.Fatal(e, flat, err)
		}
	}
	// An isolated level-one tile has a level-one flat top and raises all
	// four shared corners; the surrounding eight tiles carry its slopes.
	for i := range s.Terrain {
		s.Terrain[i].Elevation = 0
	}
	s.Terrain[4*9+4].Elevation = 1
	for y := 0; y <= 9; y++ {
		for x := 0; x <= 9; x++ {
			want := 0.0
			if x >= 4 && x <= 5 && y >= 4 && y <= 5 {
				want = 1
			}
			if got := cornerHeight(s.Terrain, 9, 9, x, y); got != want {
				t.Fatal(x, y, got, want)
			}
		}
	}
	// A one-level-per-tile back slope cancels the projected y increment.
	tf := diamondTransform(9, 9, 72, 72, "")
	_, y0 := tf.point(4, 4)
	_, y1 := tf.point(4, 5)
	if y1-72.0/4 != y0 {
		t.Fatal("back-face edge is not horizontal", y0, y1)
	}
	for _, c := range []struct{ dx, dy, want float64 }{{0, 0, 1}, {1, 0, .74}, {-1, 0, 1.14}, {0, 1, .73}, {0, -1, 1.15}} {
		if math.Abs(heightLight(c.dx, c.dy)-c.want) > .002 {
			t.Fatal("lighting fit", c, heightLight(c.dx, c.dy))
		}
	}
	// Exercise the actual mesh projection on both back faces of a pyramid.
	pyramid := heightSnapshot(9)
	crop := Crop{X2: 9, Y2: 9}
	for y := 2; y < 4; y++ {
		a := heightVertexAt(pyramid.Terrain, pyramid, crop, 72, 54, 4, y)
		b := heightVertexAt(pyramid.Terrain, pyramid, crop, 72, 54, 4, y+1)
		if b.h != a.h+1 || a.py != b.py {
			t.Fatal("pyramid back-face edge must be horizontal", a, b)
		}
	}
	for x := 6; x < 7; x++ {
		a := heightVertexAt(pyramid.Terrain, pyramid, crop, 72, 54, x, 4)
		b := heightVertexAt(pyramid.Terrain, pyramid, crop, 72, 54, x+1, 4)
		if b.h != a.h-1 || a.py != b.py {
			t.Fatal("opposite pyramid back-face edge must be horizontal", a, b)
		}
	}
}

func TestHeightfieldRejectsInvalidLevels(t *testing.T) {
	for _, h := range []float64{math.NaN(), math.Inf(1), -1, 256} {
		s := heightSnapshot(3)
		s.Terrain[0].Elevation = h
		if _, err := SnapshotImageExtent(s, 16, "diamond", true); err == nil {
			t.Fatal("extent accepted invalid elevation", h)
		}
		if _, _, err := RenderFlat(s, Options{Elevation: true}); err == nil {
			t.Fatal("renderer accepted invalid elevation", h)
		}
	}
}

func TestHeightfieldFoldsThroughOddCorner(t *testing.T) {
	for _, odd := range []float64{0, 2} {
		for i := 0; i < 4; i++ {
			h := [4]float64{1, 1, 1, 1}
			h[i] = odd
			if got, want := heightFoldBD(h[0], h[1], h[2], h[3]), i%2 == 1; got != want {
				t.Fatal("fold cut off the odd corner", h, got, want)
			}
		}
	}
	if heightFoldBD(0, 0, 1, 1) || !heightFoldBD(0, 1, 2, 1) {
		t.Fatal("ambiguous cases should prefer level diagonal, then A-C")
	}
	// A diagonal pyramid ridge: the old B-D fold created one flat triangle
	// and one two-axis slope. A-C creates two axial slopes sharing the ridge.
	s := heightSnapshot(9)
	crop := Crop{X2: 9, Y2: 9}
	a := heightVertexAt(s.Terrain, s, crop, 72, 54, 2, 2)
	b := heightVertexAt(s.Terrain, s, crop, 72, 54, 3, 2)
	c := heightVertexAt(s.Terrain, s, crop, 72, 54, 3, 3)
	d := heightVertexAt(s.Terrain, s, crop, 72, 54, 2, 3)
	if a.h != b.h || a.h != d.h || c.h != a.h+1 || heightFoldBD(a.h, b.h, c.h, d.h) {
		t.Fatal("pyramid ridge was not folded through its peak", a, b, c, d)
	}
}

func TestHeightfieldSpritesAndShadowsRidePlateau(t *testing.T) {
	game, idx := spriteFixture(t)
	s := spriteSnapshot()
	for i := range s.Terrain {
		s.Terrain[i].Elevation = 2
	}
	opts := Options{Scale: 8, SpriteTilePixels: 1, Elevation: true, Shadows: &ShadowOptions{}, Layers: map[string]bool{"objects": true}}
	raised, r, err := RenderSprites(s, opts, idx, game)
	if err != nil {
		t.Fatal(err)
	}
	opts.Elevation = false
	flat, _, err := RenderSprites(s, opts, idx, game)
	if err != nil {
		t.Fatal(err)
	}
	for y := 0; y < flat.Rect.Max.Y; y++ {
		for x := 0; x < flat.Rect.Max.X; x++ {
			// Two levels lift four pixels at scale8, cancelling the four
			// pixels of headroom: both sprite and shadow match the flat pair.
			if flat.RGBAAt(x, y) != raised.RGBAAt(x, y) {
				t.Fatal("sprite/shadow did not follow the plateau", x, y, r.ElevationPadding)
			}
		}
	}
}

func TestHeightfieldBilinearObjectAttachment(t *testing.T) {
	s := Snapshot{Schema: SnapshotSchema, Width: 2, Height: 2, Terrain: []Tile{{X: 0, Y: 0}, {X: 1, Y: 0}, {X: 0, Y: 1}, {X: 1, Y: 1, Elevation: 2}}}
	tiles := heightTiles(s)
	// Tile(0,0) has only its far corner raised to two. At .5,.5 the
	// bilinear candidate has height .5; at scale32 this lifts four pixels.
	if got := objectGroundLift(s, tiles, Object{X: .5, Y: .5}, 32, "diamond"); got != 4 {
		t.Fatal("fractional attachment", got)
	}
	if got := objectGroundLift(s, tiles, Object{X: 1.5, Y: 1.5}, 32, "diamond"); got != 16 {
		t.Fatal("plateau attachment", got)
	}
	for _, o := range []Object{{X: -1}, {X: 2}, {Y: 2}, {X: math.NaN()}} {
		if got := objectGroundLift(s, tiles, o, 32, "diamond"); got != 0 {
			t.Fatal("out of bounds attachment", o, got)
		}
	}
	if objectGroundLift(s, nil, Object{}, 32, "diamond") != 0 || objectGroundLift(s, tiles, Object{}, 32, "topdown") != 0 {
		t.Fatal("disabled or topdown attachment")
	}
}

func TestHeightfieldFaceLightingDoesNotBlendAcrossRidge(t *testing.T) {
	s := heightSnapshot(9)
	crop := Crop{X2: 9, Y2: 9}
	img := image.NewRGBA(image.Rect(0, 0, 577, 400))
	constant := color.RGBA{100, 100, 100, 255}
	renderHeightfield(img, s.Terrain, s, crop, 64, 48, func(_, _ float64) color.RGBA { return constant })
	for _, tc := range []struct {
		x, y   int
		factor float64
	}{{2, 4, .74}, {4, 6, 1.15}} {
		a := heightVertexAt(s.Terrain, s, crop, 64, 48, tc.x, tc.y)
		b := heightVertexAt(s.Terrain, s, crop, 64, 48, tc.x+1, tc.y)
		c := heightVertexAt(s.Terrain, s, crop, 64, 48, tc.x+1, tc.y+1)
		want := heightShade(constant, tc.factor)
		for _, uv := range [][2]float64{{.2, .2}, {.6, .2}, {.2, .6}} {
			x := int(a.px + uv[0]*(b.px-a.px) + uv[1]*(c.px-a.px))
			y := int(a.py + uv[0]*(b.py-a.py) + uv[1]*(c.py-a.py))
			if got := img.RGBAAt(x, y); got != want {
				t.Fatal("face lighting interpolated across a face", tc, x, y, got, want)
			}
		}
	}
}

func TestHeightfieldPlateauKeepsWorldTextureCoordinates(t *testing.T) {
	s := heightSnapshot(5)
	for i := range s.Terrain {
		s.Terrain[i].Elevation = 2
	}
	img := image.NewRGBA(image.Rect(0, 0, 161, 145))
	sample := func(x, y float64) color.RGBA { return color.RGBA{uint8(x * 40), uint8(y * 40), 100, 255} }
	renderHeightfield(img, s.Terrain, s, Crop{X2: 5, Y2: 5}, 32, 16, sample)
	tf := diamondTransform(5, 5, 32, 32, "")
	for y := 45; y < 90; y++ {
		for x := 60; x < 100; x++ {
			wx, wy := tf.inverse(float64(x)+.5, float64(y)+.5)
			if wx > 0 && wy > 0 && wx < 5 && wy < 5 {
				if got, want := img.RGBAAt(x, y), sample(wx, wy); got != want {
					t.Fatal("texture shifted on a plateau", x, y, got, want)
				}
			}
		}
	}
}
