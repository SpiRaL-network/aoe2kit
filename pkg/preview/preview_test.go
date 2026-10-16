package preview

import (
	"encoding/json"
	"errors"
	"image"
	"image/color"
	"math"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"aoe2kit/pkg/datfile"
	"aoe2kit/pkg/replay"
	"aoe2kit/pkg/scenario"
)

func testSnapshot() Snapshot {
	layer := 5
	return Snapshot{Schema: SnapshotSchema, Stage: "test", Width: 2, Height: 2, Terrain: []Tile{{X: 0, Y: 0, TerrainID: 1, Layer: &layer}, {X: 1, Y: 0, TerrainID: 2, Elevation: 2}, {X: 0, Y: 1, TerrainID: 3}, {X: 1, Y: 1, TerrainID: 4}}, Objects: []Object{{UnitID: 349, X: .5, Y: .5, Player: 0, Class: "tree", FootprintX: .5, FootprintY: .5, Blocking: true}, {UnitID: 83, X: 1.5, Y: 1.5, Player: 1, Class: "unit"}}}
}

func TestArtworkSnapshotRetainsReadOnlyProjection(t *testing.T) {
	if _, err := ArtworkSnapshot(nil, "test", nil); err == nil {
		t.Fatal("nil artwork accepted")
	}
	art := &scenario.Artwork{
		Version: "1.36", Width: 2, Height: 1,
		PlayerColors: map[int]int{1: 7},
		Tiles:        []scenario.TerrainTile{{X: 0, Y: 0, TerrainID: 1, Elevation: 2, Layer: 5}, {X: 1, Y: 0, TerrainID: 2, Layer: -1}},
		Units:        scenario.UnitInfo{Total: 1, Sections: []scenario.PlayerUnitsInfo{{Player: 1, Count: 1, Units: []scenario.UnitSummary{{UnitConst: 83, X: 1.25, Y: .5, Z: .75, Rotation: 2, Status: 3, InitialAnimationFrame: 4}}}}},
	}
	snap, err := ArtworkSnapshot(art, "campaign", nil)
	if err != nil {
		t.Fatal(err)
	}
	if snap.Stage != "campaign" || snap.Width != 2 || snap.Height != 1 || snap.PlayerColors[1] != 7 || len(snap.SourceWarnings) != 1 {
		t.Fatalf("projection metadata: %+v", snap)
	}
	if len(snap.Terrain) != 2 || snap.Terrain[0].Layer == nil || *snap.Terrain[0].Layer != 5 || snap.Terrain[0].Elevation != 2 || snap.Terrain[1].Layer != nil {
		t.Fatalf("projection terrain: %+v", snap.Terrain)
	}
	if len(snap.Objects) != 1 {
		t.Fatalf("projection objects: %+v", snap.Objects)
	}
	o := snap.Objects[0]
	if o.UnitID != 83 || o.Player != 1 || o.X != 1.25 || o.Y != .5 || o.Z != .75 || o.Rotation != 2 || o.Status != 3 || o.InitialAnimationFrame != 4 {
		t.Fatalf("projection unit: %+v", o)
	}
	_, report, err := RenderMinimap(snap, nil, Options{})
	if err != nil || !slices.Contains(report.Fallbacks, snap.SourceWarnings[0]) {
		t.Fatalf("read-only limitation missing from render report: %+v, %v", report, err)
	}
}

func TestSourceWarningsSurviveFlatAndSpriteReports(t *testing.T) {
	root, idx := spriteFixture(t)
	s := spriteSnapshot()
	s.SourceWarnings = []string{"read-only artwork: gameplay not decoded"}
	_, flat, err := RenderFlat(s, Options{})
	if err != nil || !slices.Contains(flat.Fallbacks, s.SourceWarnings[0]) {
		t.Fatalf("flat report lost source warning: %+v, %v", flat, err)
	}
	_, sprites, err := RenderSprites(s, Options{}, idx, root)
	if err != nil || !slices.Contains(sprites.Fallbacks, s.SourceWarnings[0]) {
		t.Fatalf("sprite report lost source warning: %+v, %v", sprites, err)
	}
}

func TestReadMinimapPalette(t *testing.T) {
	var entries []map[string]int
	for i := 0; i < 256; i++ {
		entries = append(entries, map[string]int{"r": i, "g": 255 - i, "b": i / 2, "a": 0})
	}
	path := filepath.Join(t.TempDir(), "palette.pal")
	data, err := json.Marshal(map[string]any{"Palette": entries})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	palette, err := ReadMinimapPalette(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(palette) != 256 || palette[19] != (color.RGBA{R: 19, G: 236, B: 9, A: 255}) {
		t.Fatalf("palette[19]=%v (len %d)", palette[19], len(palette))
	}
}

func TestReplaySnapshotFromEmbeddedTerrain(t *testing.T) {
	mapInfo := &replay.MapInfo{
		Width: 2, Height: 1, TileCount: 2,
		Tiles: []replay.MapTile{{X: 0, Y: 0, Terrain: 7, Elevation: 1}, {X: 1, Y: 0, Terrain: 12}},
	}
	objects := []replay.ObjectStateCard{{UnitID: 83, OwnerID: 2, Class: "unit", X: 1.5, Y: 0.5}}
	snap, err := replaySnapshotFromData(mapInfo, []replay.SummaryPlayer{{PlayerID: 2, Color: 4}}, objects, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if snap.Width != 2 || snap.Height != 1 || len(snap.Terrain) != 2 || snap.Terrain[0].TerrainID != 7 || snap.Terrain[0].Elevation != 1 {
		t.Fatalf("replay terrain snapshot = %+v", snap)
	}
	if len(snap.Objects) != 1 || snap.Objects[0].Player != 2 || snap.PlayerColors[2] != 4 {
		t.Fatalf("replay objects/player colors = %+v / %+v", snap.Objects, snap.PlayerColors)
	}
	snap, err = replaySnapshotFromData(mapInfo, nil, nil, errors.New("spine unavailable"), nil)
	if err != nil || len(snap.SourceWarnings) != 1 {
		t.Fatalf("missing replay objects should be reported, snapshot=%+v err=%v", snap, err)
	}
}

func TestRenderMinimapUsesDATColorAndMinimapVisibility(t *testing.T) {
	s := testSnapshot()
	s.Terrain = []Tile{{X: 0, Y: 0, TerrainID: 0}, {X: 1, Y: 0, TerrainID: 0}, {X: 0, Y: 1, TerrainID: 0}, {X: 1, Y: 1, TerrainID: 0}}
	s.Objects = []Object{
		{UnitID: 349, X: .5, Y: .5, Player: 0, Class: "tree", HasMapFields: true, MapDrawLevel: 0},
		{UnitID: 83, X: 1.5, Y: 1.5, Player: 1, Class: "unit", HasMapFields: true, MapDrawLevel: 1},
		{UnitID: 1, X: .5, Y: 1.5, Player: 1, Class: "unit", HasMapFields: true, MapDrawLevel: 1, HasMinimapMode: true, MinimapMode: 0},
	}
	palette := make([]color.RGBA, 256)
	palette[19] = color.RGBA{R: 10, G: 32, B: 48, A: 255}
	dat := &datfile.Index{Terrains: []datfile.Terrain{{Index: 0, MinimapColorMedium: 19}}}
	img, report, err := RenderMinimap(s, dat, Options{Palette: palette})
	if err != nil {
		t.Fatal(err)
	}
	if report.Drawn != 1 {
		t.Fatalf("drawn=%d, want only the non-Gaia visible unit", report.Drawn)
	}
	if report.TerrainColors != "base terrain; DAT minimap medium-color index resolved through 256-color palette" {
		t.Fatalf("terrain color source = %q", report.TerrainColors)
	}
	if !imageHasColor(img, palette[19]) {
		t.Fatalf("render does not use DAT palette color %v", palette[19])
	}
}

func TestRenderMinimapLayeredInteriorUsesBaseAndReportsBoundaryLimitation(t *testing.T) {
	s := testSnapshot()
	s.Objects = nil
	layer := 1
	for i := range s.Terrain {
		s.Terrain[i].TerrainID = 0
		s.Terrain[i].Layer = &layer
	}
	palette := make([]color.RGBA, 256)
	palette[19] = color.RGBA{R: 10, G: 32, B: 48, A: 255}
	palette[20] = color.RGBA{R: 120, G: 220, B: 40, A: 255}
	dat := &datfile.Index{Terrains: []datfile.Terrain{
		{Index: 0, MinimapColorMedium: 19},
		{Index: 1, MinimapColorMedium: 20},
	}}
	img, report, err := RenderMinimap(s, dat, Options{Palette: palette})
	if err != nil {
		t.Fatal(err)
	}
	if !imageHasColor(img, palette[19]) || imageHasColor(img, palette[20]) {
		t.Fatal("layer metadata must not override the measured base interior color")
	}
	if !slices.Contains(report.Fallbacks, "layered terrain uses base interior colors; engine boundary blending is not modeled") {
		t.Fatalf("missing boundary-blending limitation: %+v", report)
	}
}

func TestMinimapDefaultAxesWithAsymmetricTerrain(t *testing.T) {
	s := Snapshot{Schema: SnapshotSchema, Width: 8, Height: 6}
	for y := 0; y < s.Height; y++ {
		for x := 0; x < s.Width; x++ {
			s.Terrain = append(s.Terrain, Tile{X: x, Y: y, TerrainID: y*s.Width + x})
		}
	}
	palette := make([]color.RGBA, 256)
	dat := &datfile.Index{}
	for id := range s.Terrain {
		palette[id] = color.RGBA{R: uint8(id + 50), G: 100, B: 200, A: 255}
		dat.Terrains = append(dat.Terrains, datfile.Terrain{Index: id, MinimapColorMedium: uint8(id)})
	}
	img, report, err := RenderMinimap(s, dat, Options{Palette: palette, MinimapScale: 2})
	if err != nil {
		t.Fatal(err)
	}
	if report.MinimapOrientation != "rotate270" {
		t.Fatalf("default orientation: %q", report.MinimapOrientation)
	}
	// Measured axes: increasing source x goes up-right; y goes down-right.
	// At scale 2 a tile center projects to (2*(x+y+1), width+y-x).
	for _, p := range []image.Point{{0, 0}, {7, 0}, {0, 5}, {7, 5}, {2, 4}} {
		px, py := 2*(p.X+p.Y+1), s.Width+p.Y-p.X
		if got, want := img.RGBAAt(px, py), palette[p.Y*s.Width+p.X]; got != want {
			t.Fatalf("source %v at pixel (%d,%d): %v, want %v", p, px, py, got, want)
		}
	}
}

func TestScenarioSnapshotRetainsTopTerrainLayer(t *testing.T) {
	path := filepath.Join(t.TempDir(), "layered.aoe2scenario")
	if _, err := scenario.WriteBlankScenarioFile(path, scenario.BlankOptions{
		PlayerCount: 2, HumanSlots: 1, NoStarters: true, MapWidth: 80, MapHeight: 80,
	}); err != nil {
		t.Fatal(err)
	}
	file, err := scenario.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	bottom, top := 1, 22
	if err := file.ApplyRecipe(scenario.Recipe{Map: []scenario.MapRecipe{{
		Op: "layered_crossfade", X1: 12, Y1: 13, X2: 12, Y2: 13,
		TerrainID: &bottom, TerrainID2: &top,
	}}}); err != nil {
		t.Fatal(err)
	}
	snap, err := ScenarioSnapshot(file, "terrain", nil)
	if err != nil {
		t.Fatal(err)
	}
	tile := snap.Terrain[13*snap.Width+12]
	if tile.TerrainID != bottom || tile.Layer == nil || *tile.Layer != top {
		t.Fatalf("snapshot layered tile = %+v, want bottom=%d top=%d", tile, bottom, top)
	}
}

func TestMinimapOrientationCoordinates(t *testing.T) {
	for _, orientation := range []string{"normal", "transpose", "flipx", "flipy", "rotate90", "rotate180", "rotate270"} {
		w, h := 3, 2
		transform := diamondTransform(w, h, 8, 3, orientation)
		for sy := 0; sy < h; sy++ {
			for sx := 0; sx < w; sx++ {
				x, y := float64(sx)+0.5, float64(sy)+0.5
				px, py := transform.point(x, y)
				backX, backY := transform.inverse(px, py)
				if math.Abs(backX-x) > 1e-9 || math.Abs(backY-y) > 1e-9 {
					t.Fatalf("%s round trip (%v,%v) -> (%v,%v)", orientation, x, y, backX, backY)
				}
			}
		}
	}
}

func imageHasColor(img *image.RGBA, want color.RGBA) bool {
	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
			if img.RGBAAt(x, y) == want {
				return true
			}
		}
	}
	return false
}

func TestRenderMinimapWholeMapAndFallbackReport(t *testing.T) {
	s := testSnapshot()
	s.PlayerColors = map[int]int{1: 3}
	img, report, err := RenderMinimap(s, nil, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if img.Bounds().Dx() != 4 || img.Bounds().Dy() != 2 {
		t.Fatalf("minimap bounds = %v, want 4x2", img.Bounds())
	}
	if !report.Minimap || report.Projection != "minimap-diamond-2-to-1" || report.Drawn != 1 || report.MinimapOrientation != "rotate270" {
		t.Fatalf("minimap report = %+v", report)
	}
	if len(report.Fallbacks) == 0 || report.ElevationMode != "disabled; engine behavior has not been verified" {
		t.Fatalf("minimap fallback/elevation report = %+v", report)
	}
	if img.At(2, 0) == (color.RGBA{}) {
		t.Fatal("minimap terrain/object pixels were not rendered")
	}
}

func TestMinimapPlayerColourRequiresScenarioMapping(t *testing.T) {
	if got, want := minimapObjectColor("tree", 0, false), classColor("tree"); got != want {
		t.Fatalf("unmapped object colour=%v, want class fallback %v", got, want)
	}
	if got := minimapObjectColor("tree", 0, true); got != minimapPlayerColors[0] {
		t.Fatalf("scenario color index 0 = %v, want %v", got, minimapPlayerColors[0])
	}
}

func TestRenderMinimapMaximumMapStaysCompact(t *testing.T) {
	const side = 480
	s := Snapshot{Schema: SnapshotSchema, Stage: "large", Width: side, Height: side, Terrain: make([]Tile, side*side)}
	for y := 0; y < side; y++ {
		for x := 0; x < side; x++ {
			s.Terrain[y*side+x] = Tile{X: x, Y: y, TerrainID: (x + y) % 16}
		}
	}
	img, report, err := RenderMinimap(s, nil, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if img.Bounds().Dx() != 2*side || img.Bounds().Dy() != side {
		t.Fatalf("480 map minimap bounds=%v, want 960x480", img.Bounds())
	}
	if got := len(img.Pix); got != 960*480*4 {
		t.Fatalf("RGBA buffer is %d bytes, want %d", got, 960*480*4)
	}
	if report.ImageWidth != 960 || report.ImageHeight != 480 {
		t.Fatalf("report dimensions=%dx%d", report.ImageWidth, report.ImageHeight)
	}
}

func TestRenderFlatPixelHashes(t *testing.T) {
	s := testSnapshot()
	for _, tc := range []struct{ name, projection, want string }{{"topdown", "topdown", "6a297afa1a46ba0a19b71a4e2bd8bf1a3901be69712df18969c372b157ce7d02"}, {"diamond", "diamond", "dc8b66a4fa23ca8d37aed8bd9a3f6bf0497612928519387687af23fe5c4352f3"}} {
		img, report, err := RenderFlat(s, Options{Projection: tc.projection, Scale: 4, Layers: map[string]bool{"terrain": true, "elevation": true, "objects": true, "starts": true}})
		if err != nil {
			t.Fatalf("%s render: %v", tc.name, err)
		}
		if img.Bounds().Dx() != report.ImageWidth || img.Bounds().Dy() != report.ImageHeight {
			t.Fatalf("%s dimensions disagree", tc.name)
		}
		if report.Drawn != 2 || report.CountsByClass["tree"] != 1 || report.CountsByClass["unit"] != 1 {
			t.Fatalf("%s report = %+v", tc.name, report)
		}
		if report.PixelSHA256 != tc.want {
			t.Errorf("%s pixel SHA256=%s want %s", tc.name, report.PixelSHA256, tc.want)
		}
		t.Logf("%s pixel SHA256=%s", tc.name, report.PixelSHA256)
	}
}

func TestScenarioPlayerColorsUseUnitOwnerNumbers(t *testing.T) {
	path := filepath.Join(t.TempDir(), "colors.aoe2scenario")
	if _, err := scenario.WriteBlankScenarioFile(path, scenario.BlankOptions{PlayerCount: 2, HumanSlots: 1, NoStarters: true, MapWidth: 80, MapHeight: 80}); err != nil {
		t.Fatal(err)
	}
	f, err := scenario.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	f.Players[0].Color = 3
	f.Players[1].Color = 5
	s, err := ScenarioSnapshot(f, "test", nil)
	if err != nil {
		t.Fatal(err)
	}
	if s.PlayerColors[1] != 3 || s.PlayerColors[2] != 5 {
		t.Fatalf("owner colors=%v", s.PlayerColors)
	}
	if _, ok := s.PlayerColors[0]; ok {
		t.Fatal("player settings must not assign P1 color to Gaia")
	}
}

func TestMinimapMarkerSizesAndDATPlayerColors(t *testing.T) {
	s := Snapshot{Schema: SnapshotSchema, Width: 240, Height: 240, PlayerColors: map[int]int{1: 0, 2: 1}}
	for y := 0; y < 240; y++ {
		for x := 0; x < 240; x++ {
			s.Terrain = append(s.Terrain, Tile{X: x, Y: y})
		}
	}
	s.Objects = []Object{{UnitID: 83, X: 100, Y: 100, Player: 1}, {UnitID: 109, X: 200, Y: 50, Player: 2, Class: "building"}}
	palette := make([]color.RGBA, 256)
	palette[1] = color.RGBA{R: 40, G: 40, B: 40, A: 255}
	palette[79] = color.RGBA{B: 255, A: 255}
	palette[80] = color.RGBA{R: 255, A: 255}
	dat := &datfile.Index{Terrains: []datfile.Terrain{{MinimapColorMedium: 1}}, PlayerColours: []datfile.PlayerColour{{MinimapColour1: 79}, {MinimapColour1: 80}}}
	im, r, err := RenderMinimap(s, dat, Options{Palette: palette})
	if err != nil {
		t.Fatal(err)
	}
	counts := map[color.RGBA]int{}
	for y := 0; y < im.Bounds().Dy(); y++ {
		for x := 0; x < im.Bounds().Dx(); x++ {
			counts[im.RGBAAt(x, y)]++
		}
	}
	if counts[palette[79]] != 4 || counts[palette[80]] != 12 {
		t.Fatalf("unit=%d building=%d", counts[palette[79]], counts[palette[80]])
	}
	if im.RGBAAt(200, 120) != palette[79] || im.RGBAAt(250, 45) != palette[80] || r.Drawn != 2 {
		t.Fatalf("wrong marker positions/report: %+v", r)
	}
}

func TestDiamondCropGroundUsesLocalCoordinates(t *testing.T) {
	s := spriteSnapshot()
	opts := Options{Scale: 16, Projection: "diamond", Crop: &Crop{X1: 4, Y1: 4, X2: 8, Y2: 8}, Layers: map[string]bool{"terrain": true}}
	im, _, err := RenderFlat(s, opts)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range []image.Point{{0, 0}, {1, 2}, {3, 3}} {
		x, y := mapPoint(float64(p.X)+.5, float64(p.Y)+.5, *opts.Crop, opts.Scale, "diamond")
		if got := im.RGBAAt(x, y); got != terrainColor(0) {
			t.Fatalf("crop tile %v center (%d,%d) = %v", p, x, y, got)
		}
	}
	if im.Bounds().Dx() != 65 || im.Bounds().Dy() != 65 {
		t.Fatalf("diamond bounds=%v", im.Bounds())
	}
}

func TestSnapshotRoundTripIsDeterministic(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.json")
	b := filepath.Join(dir, "b.json")
	if err := WriteSnapshot(a, testSnapshot()); err != nil {
		t.Fatal(err)
	}
	got, err := ReadSnapshot(a)
	if err != nil {
		t.Fatal(err)
	}
	if got.Terrain[0].Layer == nil || *got.Terrain[0].Layer != 5 {
		t.Fatalf("snapshot layer = %v, want 5", got.Terrain[0].Layer)
	}
	if err := WriteSnapshot(b, got); err != nil {
		t.Fatal(err)
	}
	left, err := os.ReadFile(a)
	if err != nil {
		t.Fatal(err)
	}
	right, err := os.ReadFile(b)
	if err != nil {
		t.Fatal(err)
	}
	if string(left) != string(right) {
		t.Fatal("snapshot JSON changed after a read/write round trip")
	}
}

func TestRenderFlatCropAndBounds(t *testing.T) {
	s := testSnapshot()
	img, report, err := RenderFlat(s, Options{Projection: "topdown", Scale: 3, Crop: &Crop{X1: 1, Y1: 0, X2: 2, Y2: 2}})
	if err != nil {
		t.Fatal(err)
	}
	if img.Bounds().Dx() != 3 || img.Bounds().Dy() != 6 {
		t.Fatalf("crop image = %v", img.Bounds())
	}
	if report.Crop == nil || *report.Crop != (Crop{X1: 1, Y1: 0, X2: 2, Y2: 2}) || report.Scale != 3 {
		t.Fatalf("missing render framing: %+v", report)
	}
	if _, _, err := RenderFlat(s, Options{Crop: &Crop{X1: 0, Y1: 0, X2: 3, Y2: 2}}); err == nil {
		t.Fatal("out-of-bounds crop accepted")
	}
}
