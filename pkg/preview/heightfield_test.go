package preview

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"path/filepath"
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
		flat, _ := FullImageExtent(9, 9, scale, "diamond")
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
	for _, c := range []struct{ dx, dy, want float64 }{{0, 0, 1}, {1, 0, .72}, {-1, 0, 1.25}, {0, -1, 1.13}} {
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

func TestHeightfieldSpriteAnchorsStayInGroundFrame(t *testing.T) {
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
			if flat.RGBAAt(x, y) != raised.RGBAAt(x, y+r.ElevationPadding) {
				t.Fatal("object was lifted instead of just padded", x, y)
			}
		}
	}
	opts.Elevation = true
	opts.Window = &PixelWindow{X: 0, Y: 4, Width: 64, Height: 64}
	window, _, err := RenderSprites(s, opts, idx, game)
	if err != nil {
		t.Fatal(err)
	}
	rect, _ := opts.Window.rectangle()
	assertWindowPixels(t, raised, window, rect)
}

func TestHeightfieldSpriteTilePNGs(t *testing.T) {
	game, idx := spriteFixture(t)
	s := spriteSnapshot()
	for i := range s.Terrain {
		s.Terrain[i].Elevation = float64(s.Terrain[i].X % 3)
	}
	for _, budget := range []int64{1 << 20, 32 * 32 * 8} {
		opts := Options{Scale: 8, SpriteTilePixels: 1, Elevation: true, Shadows: &ShadowOptions{}}
		dir := t.TempDir()
		r, err := RenderTiles(s, opts, idx, game, TileBatchOptions{Dir: dir, TileSize: 32, Scales: []int{8}, CanvasBudgetBytes: budget, Sprites: true})
		if err != nil {
			t.Fatal(err)
		}
		level := r.Levels[0]
		for y := 0; y < level.Height; y += 32 {
			for x := 0; x < level.Width; x += 32 {
				opts.Window = &PixelWindow{X: x, Y: y, Width: 32, Height: 32}
				window, _, err := RenderSprites(s, opts, idx, game)
				if err != nil {
					t.Fatal(err)
				}
				assertTileFileMatchesWindow(t, filepath.Join(dir, "0", fmt.Sprintf("%d_%d.png", x/32, y/32)), window)
			}
		}
	}
}

func TestHeightfieldTextureWindowsAndTilePNGs(t *testing.T) {
	game, idx := terrainTestInstall(t)
	s := heightSnapshot(9)
	top := 1
	for i := range s.Terrain {
		if i%3 == 0 {
			s.Terrain[i].Layer = &top
		}
	}
	for _, edges := range []bool{false, true} {
		opts := Options{Elevation: true, Scale: 16, Dat: idx, Textures: &TerrainTextureOptions{GameDir: game, Edges: edges}, Layers: map[string]bool{"terrain": true}}
		whole, _, err := RenderFlat(s, opts)
		if err != nil {
			t.Fatal(err)
		}
		for y := -3; y < whole.Rect.Max.Y+8; y += 19 {
			for x := -5; x < whole.Rect.Max.X+8; x += 23 {
				w := PixelWindow{X: x, Y: y, Width: 23, Height: 19}
				opts.Window = &w
				window, _, err := RenderFlat(s, opts)
				if err != nil {
					t.Fatal(err)
				}
				r, _ := w.rectangle()
				assertWindowPixels(t, whole, window, r)
			}
		}
		opts.Window = nil
		for _, budget := range []int64{1 << 20, 16 * 16 * 8} {
			dir := t.TempDir()
			batch, err := RenderTiles(s, opts, idx, game, TileBatchOptions{Dir: dir, TileSize: 16, Scales: []int{8, 16}, CanvasBudgetBytes: budget})
			if err != nil {
				t.Fatal(err)
			}
			for _, level := range batch.Levels {
				for y := 0; y < level.Height; y += 16 {
					for x := 0; x < level.Width; x += 16 {
						opts.Scale = level.Scale
						opts.Window = &PixelWindow{X: x, Y: y, Width: 16, Height: 16}
						window, _, err := RenderFlat(s, opts)
						if err != nil {
							t.Fatal(err)
						}
						assertTileFileMatchesWindow(t, filepath.Join(dir, fmt.Sprint(level.Z), fmt.Sprintf("%d_%d.png", x/16, y/16)), window)
					}
				}
			}
			opts.Window = nil
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
