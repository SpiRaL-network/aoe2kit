package preview

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"aoe2kit/pkg/datfile"
)

func terrainTestInstall(t *testing.T) (string, *datfile.Index) {
	t.Helper()
	root := t.TempDir()
	write := func(group, name string, b []byte) {
		t.Helper()
		p := filepath.Join(root, "resources", "_common", "terrain", group, name)
		if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, b, 0600); err != nil {
			t.Fatal(err)
		}
	}
	dds := func(c uint16) []byte {
		b := make([]byte, 136)
		copy(b, "DDS ")
		for off, v := range map[int]uint32{4: 124, 12: 4, 16: 4, 76: 32, 80: 4} {
			binary.LittleEndian.PutUint32(b[off:], v)
		}
		copy(b[84:], "DXT1")
		binary.LittleEndian.PutUint16(b[128:], c)
		return b
	}
	write("textures/2x", "base.dds", dds(0xf800))
	write("textures/2x", "top.dds", dds(0x1f))
	mask := image.NewGray(image.Rect(0, 0, 8, 8))
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			if x >= 4 {
				mask.SetGray(x, y, color.Gray{255})
			}
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, mask); err != nil {
		t.Fatal(err)
	}
	write("masks", "layer.png", buf.Bytes())
	write("masks", "default_strong.png", buf.Bytes())
	write("blends", "roadland.png", buf.Bytes())
	write("blends", "landland.png", buf.Bytes())
	d := &datfile.Index{Terrains: []datfile.Terrain{
		{Index: 0, Name2: datfile.DebugString{Value: "base"}, BlendPriority: 0},
		{Index: 1, Name2: datfile.DebugString{Value: "top"}, OverlayMaskName: datfile.DebugString{Value: "layer.png"}, BlendPriority: 10, BlendType: 5},
	}}
	var blend bytes.Buffer
	for _, v := range []uint32{9, 31} {
		binary.Write(&blend, binary.LittleEndian, v)
	}
	for mode := 0; mode < 9; mode++ {
		binary.Write(&blend, binary.LittleEndian, uint32(2353))
		blend.Write(make([]byte, 31+4*2353))
		blend.Write(make([]byte, 31*2353))
	}
	dir := filepath.Join(root, "resources", "_common", "dat")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "blendomatic_x1.dat"), blend.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	return root, d
}

func TestTexturedLayerCropAndReport(t *testing.T) {
	root, d := terrainTestInstall(t)
	layer := 1
	s := Snapshot{Schema: SnapshotSchema, Width: 16, Height: 12}
	for y := 0; y < s.Height; y++ {
		for x := 0; x < s.Width; x++ {
			s.Terrain = append(s.Terrain, Tile{X: x, Y: y, Layer: &layer})
		}
	}
	o := Options{Projection: "topdown", Scale: 8, Dat: d, Textures: &TerrainTextureOptions{GameDir: root, Repeat: 8}, Layers: map[string]bool{"terrain": true}}
	full, r, err := RenderFlat(s, o)
	if err != nil {
		t.Fatal(err)
	}
	if full.RGBAAt(8, 8).R != 255 || full.RGBAAt(40, 8).B != 255 {
		t.Fatal("mask polarity or world UV", full.RGBAAt(8, 8), full.RGBAAt(40, 8))
	}
	if r.Textures == nil || len(r.Textures.Assets) != 2 || r.Textures.Assets[1].BlendFamily != "roadland" {
		t.Fatalf("report %+v", r)
	}
	crop := Crop{3, 2, 13, 10}
	o.Crop = &crop
	part, r2, err := RenderFlat(s, o)
	if err != nil {
		t.Fatal(err)
	}
	for y := 0; y < part.Bounds().Dy(); y++ {
		for x := 0; x < part.Bounds().Dx(); x++ {
			if part.RGBAAt(x, y) != full.RGBAAt(x+24, y+16) {
				t.Fatal("crop reset world UV")
			}
		}
	}
	_, r3, err := RenderFlat(s, o)
	if err != nil || r2.PixelSHA256 != r3.PixelSHA256 {
		t.Fatal("non-deterministic")
	}
	// Input order cannot alter the renderer.
	for i, j := 0, len(s.Terrain)-1; i < j; i, j = i+1, j-1 {
		s.Terrain[i], s.Terrain[j] = s.Terrain[j], s.Terrain[i]
	}
	_, r4, err := RenderFlat(s, o)
	if err != nil || r4.PixelSHA256 != r3.PixelSHA256 {
		t.Fatal("input ordering changed output")
	}
	d.Terrains[1].OverlayMaskName.Value = ""
	inv, r, err := RenderFlat(s, o)
	if err != nil {
		t.Fatal(err)
	}
	if inv.RGBAAt(0, 0) == part.RGBAAt(0, 0) || len(r.Textures.Assets[1].Issues) == 0 {
		t.Fatal("empty-mask candidate not identified")
	}
	d.Terrains[1].OverlayMaskName.Value = "missing.png"
	fallback, r, err := RenderFlat(s, o)
	if err != nil {
		t.Fatal(err)
	}
	if fallback.RGBAAt(40, 8).R != 255 {
		t.Fatal("missing mask should retain base")
	}
	b, _ := json.Marshal(r)
	if !strings.Contains(string(b), "base retained") {
		t.Fatal("silent fallback")
	}
}

func TestCandidateTerrainEdges(t *testing.T) {
	root, d := terrainTestInstall(t)
	s := Snapshot{Schema: SnapshotSchema, Width: 4, Height: 2}
	for y := 0; y < 2; y++ {
		for x := 0; x < 4; x++ {
			id := 0
			if x == 2 {
				id = 1
			}
			s.Terrain = append(s.Terrain, Tile{X: x, Y: y, TerrainID: id})
		}
	}
	o := Options{Projection: "topdown", Scale: 8, Dat: d, Textures: &TerrainTextureOptions{GameDir: root, Edges: true}, Layers: map[string]bool{"terrain": true}}
	a, r, err := RenderFlat(s, o)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(r.Textures.EdgeRule, "CANDIDATE") || len(r.Textures.EdgeEvidence) == 0 {
		t.Fatal("edge assumption not reported")
	}
	if r.Textures.TextureRepeat != 10 {
		t.Fatalf("default texture scale changed: %g", r.Textures.TextureRepeat)
	}
	if r.Textures.BlendomaticModes != 9 || len(r.Textures.BlendomaticSHA256) != 64 || r.Textures.EdgeIssue != "" {
		t.Fatal("missing installed-mask provenance", r.Textures)
	}
	if a.RGBAAt(15, 4).B != 255 {
		t.Fatal("higher-priority blendomatic neighbor did not extend")
	}
	o.Textures.Edges = false
	b, _, err := RenderFlat(s, o)
	if err != nil || b.RGBAAt(15, 4).R != 255 {
		t.Fatal("edge opt-out failed")
	}
	o.Textures.Edges = true
	o.Crop = &Crop{1, 0, 4, 2}
	c, _, err := RenderFlat(s, o)
	if err != nil {
		t.Fatal(err)
	}
	for y := 0; y < c.Bounds().Dy(); y++ {
		for x := 0; x < c.Bounds().Dx(); x++ {
			if c.RGBAAt(x, y) != a.RGBAAt(x+8, y) {
				t.Fatal("edge crop lost neighbor context")
			}
		}
	}
	for _, bad := range []float64{-.5, math.NaN(), math.Inf(1), 257} {
		o.Textures.Repeat = bad
		if _, _, err := RenderFlat(s, o); err == nil {
			t.Fatal("accepted bad repeat")
		}
	}
	o.Textures.Repeat = 9
	o.Crop = nil
	if err := os.Remove(filepath.Join(root, "resources", "_common", "dat", "blendomatic_x1.dat")); err != nil {
		t.Fatal(err)
	}
	hard, missing, err := RenderFlat(s, o)
	if err != nil || hard.RGBAAt(15, 4).R != 255 || !strings.Contains(missing.Textures.EdgeIssue, "hard borders retained") {
		t.Fatal("missing masks must be explicit", err, missing.Textures)
	}
}

func TestTerrainPremultipliedAlpha(t *testing.T) {
	i := image.NewRGBA(image.Rect(0, 0, 1, 1))
	i.SetRGBA(0, 0, color.RGBA{128, 0, 0, 128})
	m := terrainMaterial{texture: i, fallback: color.RGBA{0, 0, 255, 255}}
	if got := materialPixel(m, 0, 0, 9); got != (color.RGBA{128, 0, 127, 255}) {
		t.Fatal(got)
	}
}

func TestMaskAndTextureRepeatsIndependent(t *testing.T) {
	mask := image.NewGray(image.Rect(0, 0, 2, 1))
	mask.SetGray(0, 0, color.Gray{255})
	texture := image.NewRGBA(image.Rect(0, 0, 2, 1))
	texture.SetRGBA(0, 0, color.RGBA{255, 0, 0, 255})
	texture.SetRGBA(1, 0, color.RGBA{0, 255, 0, 255})
	mats := map[int]terrainMaterial{0: {fallback: color.RGBA{0, 0, 255, 255}}, 1: {texture: texture, mask: mask}}
	layer := 1
	t0 := Tile{Layer: &layer}
	if got := layeredPixel(t0, mats, 3, 0, 8, 4); got.G != 255 {
		t.Fatal(got)
	}
	if got := layeredPixel(t0, mats, 3, 0, 8, 8); got.R != 255 {
		t.Fatal(got)
	}
	if got := layeredPixel(t0, mats, 5, 0, 8, 4); got.B != 255 {
		t.Fatal("texture repeat altered mask", got)
	}
}

func TestTexturedDiamondUsesSharedAxes(t *testing.T) {
	root, d := terrainTestInstall(t)
	s := Snapshot{Schema: SnapshotSchema, Width: 12, Height: 10}
	for y := 0; y < s.Height; y++ {
		for x := 0; x < s.Width; x++ {
			id := 0
			if x >= 6 {
				id = 1
			}
			s.Terrain = append(s.Terrain, Tile{X: x, Y: y, TerrainID: id})
		}
	}
	crop := Crop{3, 2, 10, 9}
	o := Options{Projection: "diamond", Scale: 32, Crop: &crop, Dat: d, Textures: &TerrainTextureOptions{GameDir: root}, Layers: map[string]bool{"terrain": true}}
	img, _, err := RenderFlat(s, o)
	if err != nil {
		t.Fatal(err)
	}
	for _, x := range []int{4, 8} {
		px, py := mapPoint(float64(x-crop.X1)+.5, 3.5, crop, 32, "diamond")
		c := img.RGBAAt(px, py)
		if (x == 4 && c.R != 255) || (x == 8 && c.B != 255) {
			t.Fatal("texture axes disagree with sprites", x, c)
		}
	}
}
