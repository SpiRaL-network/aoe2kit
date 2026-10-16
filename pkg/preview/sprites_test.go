package preview

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"

	"image"
	"image/color"
	"os"
	"path/filepath"

	"testing"

	"aoe2kit/pkg/datfile"
)

// A single opaque-red BC1 block on a 4x4 canvas with hotspot (2,3).
func spriteFixture(t *testing.T) (string, *datfile.Index) {
	t.Helper()
	b := []byte("SLDX")
	for _, v := range []uint16{4, 1, 0, 16} {
		b = binary.LittleEndian.AppendUint16(b, v)
	}
	b = binary.LittleEndian.AppendUint32(b, 0xff000000)
	for _, v := range []uint16{4, 4, 2, 3} {
		b = binary.LittleEndian.AppendUint16(b, v)
	}
	b = append(b, 1, 1, 0, 0)
	body := []byte{}
	for _, v := range []uint16{0, 0, 4, 4} {
		body = binary.LittleEndian.AppendUint16(body, v)
	}
	body = append(body, 0, 1, 1, 0, 0, 1)
	body = binary.LittleEndian.AppendUint16(body, 0xf800)
	body = binary.LittleEndian.AppendUint16(body, 0)
	body = binary.LittleEndian.AppendUint32(body, 0)
	b = binary.LittleEndian.AppendUint32(b, uint32(len(body)+4))
	b = append(b, body...)
	for len(b)%4 != 0 {
		b = append(b, 0)
	}
	root := t.TempDir()
	dir := filepath.Join(root, "resources", "_common", "drs", "graphics")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "synthetic.sld"), b, 0600); err != nil {
		t.Fatal(err)
	}
	idx := &datfile.Index{Graphics: []datfile.Graphic{{Present: true, FileName: datfile.DebugString{Value: "synthetic"}}}, Civs: []datfile.Civ{{Units: []datfile.UnitSummary{{Index: 7, ID: 7, Present: true, Name: "Synthetic", StandingGraphic1: 0}}}}}
	return root, idx
}

func spriteSnapshot() Snapshot {
	s := Snapshot{Schema: SnapshotSchema, Width: 10, Height: 10}
	for y := 0; y < 10; y++ {
		for x := 0; x < 10; x++ {
			s.Terrain = append(s.Terrain, Tile{X: x, Y: y})
		}
	}
	s.Objects = []Object{{UnitID: 7, X: 5, Y: 5, Class: "tree"}}
	return s
}

func TestSpriteCropAnchorScaleAndHash(t *testing.T) {
	root, idx := spriteFixture(t)
	s := spriteSnapshot()
	for _, native := range []int{4, 8} {
		opts := Options{Scale: 4, SpriteTilePixels: native, Crop: &Crop{X1: 4, Y1: 4, X2: 8, Y2: 8}, Layers: map[string]bool{"objects": true}}
		img, r, err := RenderSprites(s, opts, idx, root)
		if err != nil {
			t.Fatal(err)
		}
		factor := 4.0 / float64(native)
		// Source (5,5) becomes crop-relative (1,1), projected to (4,8).
		x, y := 4-int(2*factor), 8-int(3*factor+0.5)
		if c := img.RGBAAt(x, y); c.R < 248 || c.G != 0 || c.B != 0 {
			t.Fatalf("hotspot/scale mismatch at (%d,%d): %v", x, y, c)
		}
		if r.Drawn != 1 {
			t.Fatalf("report=%+v", r)
		}
		h := sha256.Sum256(img.Pix)
		if r.PixelSHA256 != hex.EncodeToString(h[:]) {
			t.Fatal("report hash must include the sprite")
		}
		again, rr, err := RenderSprites(s, opts, idx, root)
		if err != nil || rr.PixelSHA256 != r.PixelSHA256 || again.Bounds() != img.Bounds() {
			t.Fatal("non-deterministic sprite render")
		}
	}
}

func TestSpriteShadowPassOrderAndOptIn(t *testing.T) {
	root, idx := spriteFixture(t)
	path := filepath.Join(root, "resources", "_common", "drs", "graphics", "synthetic.sld")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	// Enlarge canvas; shadow block at (4,0) is beside the main block.
	binary.LittleEndian.PutUint16(b[16:18], 8)
	b[24] |= 2
	body := []byte{4, 0, 0, 0, 8, 0, 4, 0, 0, 1, 1, 0, 0, 1, 80, 0, 0, 0, 0, 0, 0, 0}
	b = binary.LittleEndian.AppendUint32(b, uint32(len(body)+4))
	b = append(b, body...)
	for len(b)%4 != 0 {
		b = append(b, 0)
	}
	if err := os.WriteFile(path, b, 0600); err != nil {
		t.Fatal(err)
	}
	s := spriteSnapshot()
	opts := Options{Scale: 4, SpriteTilePixels: 4}
	baseline, r, err := RenderSprites(s, opts, idx, root)
	if err != nil {
		t.Fatal(err)
	}
	if r.Shadows != nil {
		t.Fatal("enabled shadows by default")
	}
	opts.Shadows = &ShadowOptions{}
	shadowed, r, err := RenderSprites(s, opts, idx, root)
	if err != nil {
		t.Fatal(err)
	}
	if r.Shadows.Drawn != 1 || r.Shadows.CoveredPixels == 0 || r.Drawn != 1 {
		t.Fatalf("report %+v", r)
	}
	x, y := mapPoint(5, 5, Crop{X2: 10, Y2: 10}, 4, "diamond")
	if baseline.RGBAAt(x-2, y-3) != shadowed.RGBAAt(x-2, y-3) {
		t.Fatal("shadow pass altered sprite")
	}
	if shadowed.RGBAAt(x+3, y-2).G >= baseline.RGBAAt(x+3, y-2).G {
		t.Fatal("terrain not darkened")
	}
	// Another caster whose shadow covers the first sprite still cannot darken
	// that sprite: coverage is applied before all main frames.
	s.Objects = append(s.Objects, Object{UnitID: 7, X: 4, Y: 4, Class: "tree"})
	shadowed, _, err = RenderSprites(s, opts, idx, root)
	if err != nil {
		t.Fatal(err)
	}
	if shadowed.RGBAAt(x-1, y-2).R != 255 {
		t.Fatal("caster order darkened main sprite")
	}
}

func TestEngineQuadrantsAcrossRenderers(t *testing.T) {
	root, idx := spriteFixture(t)
	s := Snapshot{Schema: SnapshotSchema, Width: 14, Height: 10}
	for y := 0; y < s.Height; y++ {
		for x := 0; x < s.Width; x++ {
			s.Terrain = append(s.Terrain, Tile{X: x, Y: y})
		}
	}
	palette := make([]color.RGBA, 256)
	palette[1] = color.RGBA{R: 255, A: 255}
	positions := [][2]float64{{6, 5}, {9, 5}, {6, 7}}
	for _, mode := range []string{"minimap", "diamond", "sprites"} {
		var centers [][2]float64
		for _, pos := range positions {
			s.Objects = []Object{{UnitID: 7, X: pos[0], Y: pos[1], Player: 1, Class: "building", MapColor: 1}}
			opts := Options{Projection: "diamond", Scale: 16, SpriteTilePixels: 4, MinimapScale: 4, Palette: palette, Crop: &Crop{X1: 2, Y1: 1, X2: 12, Y2: 9}, Layers: map[string]bool{"objects": true}}
			var img *image.RGBA
			var err error
			want := palette[1]
			switch mode {
			case "minimap":
				img, _, err = RenderMinimap(s, nil, opts)
			case "diamond":
				img, _, err = RenderFlat(s, opts)
				want = classColor("building")
			case "sprites":
				img, _, err = RenderSprites(s, opts, idx, root)
			}
			if err != nil {
				t.Fatal(err)
			}
			var xsum, ysum, n float64
			for y := 0; y < img.Bounds().Dy(); y++ {
				for x := 0; x < img.Bounds().Dx(); x++ {
					if img.RGBAAt(x, y) == want {
						xsum += float64(x)
						ysum += float64(y)
						n++
					}
				}
			}
			if n == 0 {
				t.Fatalf("%s drew no marker pixels", mode)
			}
			centers = append(centers, [2]float64{xsum / n, ysum / n})
		}
		c, east, south := centers[0], centers[1], centers[2]
		if east[0] <= c[0] || east[1] >= c[1] || south[0] <= c[0] || south[1] <= c[1] {
			t.Fatalf("%s: expected east up-right, south down-right: %v", mode, centers)
		}
	}
}

func TestPainterOrderUsesEngineScreenDepth(t *testing.T) {
	input := []Object{{UnitID: 1, X: 2, Y: 4}, {UnitID: 2, X: 8, Y: 4}, {UnitID: 3, X: 5, Y: 4}}
	ordered := depthSortedObjects(input, Crop{X2: 10, Y2: 8}, 16, "diamond")
	for i, want := range []int{2, 3, 1} {
		if ordered[i].UnitID != want {
			t.Fatalf("screen depth order: %+v", ordered)
		}
	}
	if input[0].UnitID != 1 {
		t.Fatal("sort changed snapshot order")
	}
}

func TestSpriteMissingVariantAndHiddenLayer(t *testing.T) {
	root, idx := spriteFixture(t)
	s := spriteSnapshot()
	s.Objects[0].Variant = 7
	_, r, err := RenderSprites(s, Options{}, idx, root)
	if err != nil || r.Drawn != 0 || len(r.Skipped) != 1 {
		t.Fatalf("variant failure: %+v %v", r, err)
	}
	s.Objects[0].Variant = 0
	_, r, err = RenderSprites(s, Options{Layers: map[string]bool{"terrain": true}}, idx, root)
	if err != nil || r.Drawn != 0 {
		t.Fatalf("hidden objects: %+v %v", r, err)
	}
	if _, _, err = RenderSprites(s, Options{Projection: "topdown"}, idx, root); err == nil {
		t.Fatal("isometric sprite must not claim topdown projection")
	}
}
