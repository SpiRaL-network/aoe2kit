package scenario

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"testing"

	"aoe2kit/pkg/felt"
)

func motifFixture() *Artwork {
	a := &Artwork{Version: "1.36", Width: 5, Height: 4, Units: UnitInfo{Sections: []PlayerUnitsInfo{{Player: 0, Units: []UnitSummary{
		{UnitConst: 1062, X: 1.25, Y: 2.25, Z: .5, Rotation: 0, Status: 2, InitialAnimationFrame: 3, ReferenceID: 99, CaptionString: "fence"},
		{UnitConst: 1062, X: 4, Y: 2, Rotation: 0, Status: 2},
	}}}}}
	for y := 0; y < 4; y++ {
		for x := 0; x < 5; x++ {
			a.Tiles = append(a.Tiles, TerrainTile{X: x, Y: y, TerrainID: y*5 + x, Layer: -1, Elevation: y})
		}
	}
	return a
}

func TestExtractAndTranslateMotif(t *testing.T) {
	a := motifFixture()
	opts := ExtractOptions{Crop: MotifCrop{1, 1, 4, 3}, Terrain: true, Relative: true}
	r, err := ExtractRecipe(a, filepath.Join("source", "sample.aoe2scenario"), opts)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Units) != 1 || *r.Units[0].X != .25 || *r.Units[0].Y != 1.25 || *r.Units[0].Z != .5 || r.Units[0].ReferenceID != nil || r.Units[0].Snap != "fractional" || *r.Units[0].InitialAnimationFrame != 3 || r.Source.Scenario != "sample.aoe2scenario" {
		t.Fatalf("%+v", r)
	}
	if !reflect.DeepEqual(r.Map[0].TerrainIDGrid, []int{6, 7, 8, 11, 12, 13}) {
		t.Fatal(r.Map)
	}
	b, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var decoded Recipe
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatal(err)
	}
	placed, err := TransformMotif(decoded, MotifTransform{At: &PointRecipe{10, 20}})
	if err != nil {
		t.Fatal(err)
	}
	if *placed.Units[0].X != 10.25 || *placed.Units[0].Y != 21.25 || placed.Map[0].X1 != 10 || placed.Map[0].Y1 != 20 || *placed.Map[0].Width != 3 {
		t.Fatalf("%+v", placed)
	}
	if *decoded.Units[0].X != .25 || decoded.Source.Placement != nil {
		t.Fatal("source motif mutated")
	}
	// Absolute extraction normalizes through the source frame to the same target.
	opts.Relative = false
	absolute, err := ExtractRecipe(a, "sample", opts)
	if err != nil {
		t.Fatal(err)
	}
	p, err := TransformMotif(absolute, MotifTransform{At: &PointRecipe{10, 20}})
	if err != nil || *p.Units[0].X != 10.25 || *p.Units[0].Y != 21.25 {
		t.Fatalf("absolute placement %+v %v", p, err)
	}
}

func TestMotifQuarterTurnsAndGrammarRejection(t *testing.T) {
	r, err := ExtractRecipe(motifFixture(), "sample", ExtractOptions{Crop: MotifCrop{1, 1, 4, 3}, Terrain: true, Relative: true})
	if err != nil {
		t.Fatal(err)
	}
	c, err := felt.Load("")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		degrees       int
		x, y, variant float64
		grid          []int
		w, h          int
	}{
		{90, .75, .25, 1, []int{11, 6, 12, 7, 13, 8}, 2, 3},
		{180, 2.75, .75, 0, []int{13, 12, 11, 8, 7, 6}, 3, 2},
		{270, 1.25, 2.75, 1, []int{8, 13, 7, 12, 6, 11}, 2, 3},
	} {
		p, e := TransformMotif(r, MotifTransform{Rotate: tc.degrees, Grammar: c})
		if e != nil {
			t.Fatal(e)
		}
		u := p.Units[0]
		if *u.X != tc.x || *u.Y != tc.y || *u.Rotation != tc.variant || *p.Map[0].Width != tc.w || *p.Map[0].Height != tc.h || !reflect.DeepEqual(p.Map[0].TerrainIDGrid, tc.grid) {
			t.Fatalf("turn%d: %+v %+v", tc.degrees, u, p.Map)
		}
		if len(p.Map[0].LayerGrid) != 6 || len(p.Map[0].ElevationGrid) != 6 {
			t.Fatal("terrain channels lost")
		}
	}
	if _, e := TransformMotif(r, MotifTransform{Rotate: 45, Grammar: c}); e == nil {
		t.Fatal("arbitrary angle accepted")
	}
	if _, e := TransformMotif(r, MotifTransform{At: &PointRecipe{.5, 1}, Grammar: c}); e == nil {
		t.Fatal("fractional terrain origin accepted")
	}
	r.Units[0].UnitConst = 2043
	if _, e := TransformMotif(r, MotifTransform{Rotate: 90, Grammar: c}); e == nil {
		t.Fatal("unsupported bridge rotation inferred")
	}
	r.Units[0].UnitConst = 1062
	r.Units[0].Rotation = motifPtr(1.570796)
	if _, e := TransformMotif(r, MotifTransform{Rotate: 90, Grammar: c}); e == nil {
		t.Fatal("legacy angle guessed as a variant")
	}
}

func TestExtractedMotifPatchesTwiceWithFreshReferences(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "base.aoe2scenario")
	if _, err := WriteBlankScenarioFile(base, BlankOptions{MapWidth: 80, MapHeight: 80, NoStarters: true}); err != nil {
		t.Fatal(err)
	}
	f, err := Open(base)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.AddUnit(UnitRecipe{Op: "add_unit", Player: 0, UnitConst: 1062, X: motifPtr(70.5), Y: motifPtr(70.5), ReferenceID: motifPtr(99), Snap: "fractional"}); err != nil {
		t.Fatal(err)
	}
	if err := f.Write(base); err != nil {
		t.Fatal(err)
	}
	r, err := ExtractRecipe(motifFixture(), "synthetic.aoe2scenario", ExtractOptions{Crop: MotifCrop{1, 1, 4, 3}, Terrain: true, Relative: true})
	if err != nil {
		t.Fatal(err)
	}
	c, err := felt.Load("")
	if err != nil {
		t.Fatal(err)
	}
	p, err := TransformMotif(r, MotifTransform{At: &PointRecipe{10, 20}, Rotate: 90, Grammar: c})
	if err != nil {
		t.Fatal(err)
	}
	first, second := filepath.Join(dir, "first.aoe2scenario"), filepath.Join(dir, "second.aoe2scenario")
	if _, err := PatchRecipeFile(base, first, p); err != nil {
		t.Fatal(err)
	}
	if _, err := PatchRecipeFile(first, second, p); err != nil {
		t.Fatal(err)
	}
	a, err := OpenArtwork(second)
	if err != nil {
		t.Fatal(err)
	}
	if a.Units.Total != 3 {
		t.Fatalf("total %d", a.Units.Total)
	}
	seen := map[int]bool{}
	placed := 0
	for _, s := range a.Units.Sections {
		for _, u := range s.Units {
			if seen[u.ReferenceID] {
				t.Fatal("duplicate pasted reference")
			}
			seen[u.ReferenceID] = true
			if u.ReferenceID == 99 {
				continue
			}
			placed++
			if u.X != 10.75 || u.Y != 20.25 || u.Z != .5 || u.Rotation != 1 || u.Status != 2 || u.CaptionString != "fence" {
				t.Fatalf("pasted unit %+v", u)
			}
		}
	}
	if placed != 2 || !seen[99] {
		t.Fatal("lost original or pasted units")
	}
	tile := a.Tiles[20*a.Width+10]
	if tile.TerrainID != 11 || tile.Elevation != 2 || tile.Layer != -1 {
		t.Fatalf("pasted terrain %+v", tile)
	}
	// Placement metadata makes repeated transforms use the new rectangle.
	turn := r
	for i := 0; i < 4; i++ {
		turn, err = TransformMotif(turn, MotifTransform{Rotate: 90, Grammar: c})
		if err != nil {
			t.Fatal(err)
		}
	}
	if *turn.Units[0].X != *r.Units[0].X || *turn.Units[0].Y != *r.Units[0].Y || !reflect.DeepEqual(turn.Map[0].TerrainIDGrid, r.Map[0].TerrainIDGrid) {
		t.Fatal("four quarter turns did not restore motif")
	}
}
