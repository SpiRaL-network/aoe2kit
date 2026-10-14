package scenario

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAINameReferenceUsesOneBasedPlayerLabels(t *testing.T) {
	for _, test := range []struct {
		index int
		want  string
	}{{0, "P1"}, {1, "P2"}, {2, "P3"}, {7, "P8"}} {
		got := aiNameReference(test.index)
		if got.Player != test.want || got.SectionIndex != test.index || got.IndexBase != "players_from_1" {
			t.Fatalf("aiNameReference(%d)=%+v, want player %s", test.index, got, test.want)
		}
	}
}

func TestSetPlayerWritesEditorPlayerMirrors(t *testing.T) {
	path := filepath.Join(t.TempDir(), "player.aoe2scenario")
	if _, err := WriteBlankScenarioFile(path, BlankOptions{PlayerCount: 2, NoStarters: true, MapWidth: 120, MapHeight: 120}); err != nil {
		t.Fatal(err)
	}
	f, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	architecture := "SICILIANS-CIV"
	age, population, color := 6, 55, 5
	if err := f.SetPlayer(PlayerRecipe{Player: 1, Architecture: &architecture, StartingAge: &age, PopulationLimit: &population, Color: &color}); err != nil {
		t.Fatal(err)
	}
	p := f.Settings().Players[1]
	if p.Architecture != architecture || p.StartingAge != age || p.PopulationLimit != population || p.Color != color {
		t.Fatalf("player settings = %+v", p)
	}
	if got := testIntValue(t, f.root.section("Units").list("player_data_3")[1], "color"); got != color {
		t.Fatalf("player_data_3 color=%d, want %d", got, color)
	}
	if got, _ := f.root.section("Units").list("player_data_4")[1].floatValue("population_limit"); got != float64(population) {
		t.Fatalf("player_data_4 population_limit=%v, want %d", got, population)
	}
	if err := f.Write(path); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	p = reopened.Settings().Players[1]
	if p.Architecture != architecture || p.StartingAge != age || p.PopulationLimit != population || p.Color != color {
		t.Fatalf("reopened player settings = %+v", p)
	}
}

func TestScenarioPlayerCountSynchronizesActiveSlots(t *testing.T) {
	path := filepath.Join(t.TempDir(), "players.aoe2scenario")
	if _, err := WriteBlankScenarioFile(path, BlankOptions{PlayerCount: 2, NoStarters: true}); err != nil {
		t.Fatal(err)
	}
	f, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	count := 3
	if err := f.SetScenario(ScenarioRecipe{PlayerCount: &count}); err != nil {
		t.Fatal(err)
	}
	if got := testIntValue(t, f.headerRoot, "player_count"); got != count {
		t.Fatalf("header player_count=%d, want %d", got, count)
	}
	players := f.root.section("DataHeader").list("player_data_1")
	for i := 0; i < 8; i++ {
		if got, want := testIntValue(t, players[i], "active"), boolInt(i < count); got != want {
			t.Fatalf("player %d active=%d, want %d", i+1, got, want)
		}
	}
	if len(f.Players) < 3 || !f.Players[2].Active {
		t.Fatalf("refreshed P3 = %+v", f.Players)
	}
	if f.PlayerCount != count {
		t.Fatalf("in-memory player count=%d, want %d", f.PlayerCount, count)
	}
}

func TestTypedPlayerRecordOffsetsAreAbsolute(t *testing.T) {
	path := filepath.Join(t.TempDir(), "offsets.aoe2scenario")
	if _, err := WriteBlankScenarioFile(path, BlankOptions{PlayerCount: 2, NoStarters: true}); err != nil {
		t.Fatal(err)
	}
	f, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	players := f.root.section("Units").list("player_data_3")
	first := players[0].field("editor_camera_x")
	second := players[1].field("editor_camera_x")
	unitsStart := -1
	for _, section := range f.Sections {
		if section.Name == "Units" {
			unitsStart = section.Start
			break
		}
	}
	if unitsStart < 0 || first.Start < unitsStart || second.Start <= first.Start {
		t.Fatalf("player camera offsets first=%d second=%d sections=%+v", first.Start, second.Start, f.Sections)
	}
	for _, field := range []*parsedNode{first, second} {
		if field.Start < 0 || field.Start+len(field.Raw) > len(f.body) || string(f.body[field.Start:field.Start+len(field.Raw)]) != string(field.Raw) {
			t.Fatalf("field %s offset %d does not address its raw bytes", field.Name, field.Start)
		}
	}
}

func TestResizeMapRecipeCentersTerrainAndUnits(t *testing.T) {
	path := filepath.Join(t.TempDir(), "resize.aoe2scenario")
	if _, err := WriteBlankScenarioFile(path, BlankOptions{PlayerCount: 1, NoStarters: true, MapWidth: 80, MapHeight: 80}); err != nil {
		t.Fatal(err)
	}
	f, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	x, y, id, terrain := 10.5, 11.5, 45, 3
	if err := f.AddUnits([]UnitRecipe{{Op: "add_unit", Player: 1, UnitConst: 83, X: &x, Y: &y, ReferenceID: &id}}); err != nil {
		t.Fatal(err)
	}
	if err := f.SetTerrainWithMasks(MapRecipe{Op: "set_terrain_rect", X1: 0, Y1: 0, X2: 79, Y2: 79, TerrainID: &terrain}, nil); err != nil {
		t.Fatal(err)
	}
	if err := f.ApplyRecipe(Recipe{Map: []MapRecipe{{Op: "resize_map", Preset: "Tiny"}}}); err != nil {
		t.Fatal(err)
	}
	if f.Map.Width != 120 || f.Map.Height != 120 {
		t.Fatalf("map size=%dx%d, want 120x120", f.Map.Width, f.Map.Height)
	}
	if f.Units.Sections[1].Units[0].X != 30.5 || f.Units.Sections[1].Units[0].Y != 31.5 {
		t.Fatalf("translated unit position=(%v,%v)", f.Units.Sections[1].Units[0].X, f.Units.Sections[1].Units[0].Y)
	}
	tiles := f.root.section("Map").list("terrain_data")
	if got := testIntValue(t, tiles[20*120+20], "terrain_id"); got != terrain {
		t.Fatalf("translated terrain id=%d, want %d", got, terrain)
	}
	if got := testIntValue(t, tiles[0], "terrain_id"); got != 0 {
		t.Fatalf("new border terrain id=%d, want 0", got)
	}
	if err := f.VerifyRebuild(); err != nil {
		t.Fatalf("resize rebuild: %v", err)
	}
}

func TestResizeMapRejectsReferencedUnitsBeforeMutation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "shrink.aoe2scenario")
	if _, err := WriteBlankScenarioFile(path, BlankOptions{PlayerCount: 1, NoStarters: true, MapWidth: 200, MapHeight: 200}); err != nil {
		t.Fatal(err)
	}
	f, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	x, y, id := 190.5, 190.5, 91
	if err := f.AddUnits([]UnitRecipe{{Op: "add_unit", Player: 1, UnitConst: 83, X: &x, Y: &y, ReferenceID: &id}}); err != nil {
		t.Fatal(err)
	}
	triggerData := f.root.section("Triggers").field("trigger_data")
	condition := &parsedNode{Name: "Condition", Fields: []*parsedNode{{Name: "unit_object", Value: int32(id), Raw: []byte{byte(id), 0, 0, 0}}}}
	trigger := &parsedNode{Name: "Trigger", Fields: []*parsedNode{{Name: "condition_data", Elements: []*parsedNode{condition}}}}
	triggerData.Elements = append(triggerData.Elements, trigger)
	if err := f.ResizeMapWithOffset(120, 120, -40, -40, 0); err == nil {
		t.Fatal("resize removed a unit referenced by a trigger")
	}
	if f.Map.Width != 200 || f.Units.Total != 1 || f.Units.Sections[1].Units[0].ReferenceID != id {
		t.Fatalf("failed resize mutated scenario: map=%dx%d units=%+v", f.Map.Width, f.Map.Height, f.Units)
	}
}

func TestResizeMapPlanReportsOutOfBoundsUnitRemoval(t *testing.T) {
	path := filepath.Join(t.TempDir(), "resize-report.aoe2scenario")
	if _, err := WriteBlankScenarioFile(path, BlankOptions{PlayerCount: 1, NoStarters: true, MapWidth: 200, MapHeight: 200}); err != nil {
		t.Fatal(err)
	}
	f, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	x, y, id := 190.5, 190.5, 91
	if err := f.AddUnits([]UnitRecipe{{Op: "add_unit", Player: 1, UnitConst: 83, X: &x, Y: &y, ReferenceID: &id}}); err != nil {
		t.Fatal(err)
	}
	plan, err := f.Plan(Recipe{Map: []MapRecipe{{Op: "resize_map", Preset: "Tiny"}}})
	if err != nil {
		t.Fatal(err)
	}
	if plan.UnitCountBefore != 1 || plan.UnitCountAfter != 0 {
		t.Fatalf("resize plan unit counts=%d -> %d, want 1 -> 0", plan.UnitCountBefore, plan.UnitCountAfter)
	}
}

func TestRegenerateMapResetsTerrainAndClearsAllUnits(t *testing.T) {
	path := filepath.Join(t.TempDir(), "regenerate.aoe2scenario")
	if _, err := WriteBlankScenarioFile(path, BlankOptions{PlayerCount: 1, NoStarters: true, MapWidth: 120, MapHeight: 120}); err != nil {
		t.Fatal(err)
	}
	f, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	x, y, id := 20.5, 30.5, 91
	if err := f.AddUnits([]UnitRecipe{{Op: "add_unit", Player: 1, UnitConst: 83, X: &x, Y: &y, ReferenceID: &id}}); err != nil {
		t.Fatal(err)
	}
	terrain := 5
	recipe := Recipe{Map: []MapRecipe{{Op: "regenerate_map", Preset: "Small", FillTerrain: &terrain}}}
	plan, err := f.Plan(recipe)
	if err != nil {
		t.Fatal(err)
	}
	if plan.UnitCountBefore != 1 || plan.UnitCountAfter != 0 {
		t.Fatalf("regenerate plan unit counts=%d -> %d, want 1 -> 0", plan.UnitCountBefore, plan.UnitCountAfter)
	}
	if err := f.ApplyRecipe(recipe); err != nil {
		t.Fatal(err)
	}
	if f.Map.Width != 144 || f.Units.Total != 0 {
		t.Fatalf("regenerated map=%dx%d units=%d", f.Map.Width, f.Map.Height, f.Units.Total)
	}
	for index, tile := range f.root.section("Map").list("terrain_data") {
		if got := testIntValue(t, tile, "terrain_id"); got != terrain {
			t.Fatalf("tile %d terrain=%d, want %d", index, got, terrain)
		}
	}
	if err := f.VerifyRebuild(); err != nil {
		t.Fatalf("regenerate rebuild: %v", err)
	}
}

func testIntValue(t *testing.T, node *parsedNode, name string) int {
	t.Helper()
	value, ok := node.intValue(name)
	if !ok {
		t.Fatalf("missing integer field %s", name)
	}
	return value
}

func TestSetMapColorMoodAndRecipeRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mood.aoe2scenario")
	if _, err := WriteBlankScenarioFile(path, BlankOptions{PlayerCount: 1, NoStarters: true}); err != nil {
		t.Fatal(err)
	}
	f, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	mood := "Swamp"
	recipe := Recipe{Map: []MapRecipe{{Op: "set_map_color_mood", MapColorMood: &mood}}}
	if _, err := f.Plan(recipe); err != nil {
		t.Fatal(err)
	}
	if err := f.ApplyRecipe(recipe); err != nil {
		t.Fatal(err)
	}
	if got := f.Settings().Map.ColorMood; got != mood {
		t.Fatalf("map color mood=%q, want %q", got, mood)
	}
	if err := f.VerifyRebuild(); err != nil {
		t.Fatal(err)
	}
}

func TestScenarioDiffReportsMapColorMood(t *testing.T) {
	dir := t.TempDir()
	beforePath := filepath.Join(dir, "before.aoe2scenario")
	afterPath := filepath.Join(dir, "after.aoe2scenario")
	if _, err := WriteBlankScenarioFile(beforePath, BlankOptions{PlayerCount: 1, NoStarters: true}); err != nil {
		t.Fatal(err)
	}
	before, err := Open(beforePath)
	if err != nil {
		t.Fatal(err)
	}
	after, err := Open(beforePath)
	if err != nil {
		t.Fatal(err)
	}
	if err := after.SetMapColorMood("Autumn"); err != nil {
		t.Fatal(err)
	}
	if err := after.Write(afterPath); err != nil {
		t.Fatal(err)
	}
	after, err = Open(afterPath)
	if err != nil {
		t.Fatal(err)
	}
	report := Diff(before, after)
	for _, change := range report.Changes {
		if change.Kind == "map" && change.Field == "color_mood" && change.Before == "" && change.After == "Autumn" {
			return
		}
	}
	t.Fatalf("diff changes=%+v, missing map.color_mood", report.Changes)
}

func TestMessageRecipeResolvesAndWritesGameString(t *testing.T) {
	gameStrings := filepath.Join(t.TempDir(), "key-value-strings-utf8.txt")
	if err := os.WriteFile(gameStrings, []byte("9002 \"Select Object\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "messages.aoe2scenario")
	if _, err := WriteBlankScenarioFile(path, BlankOptions{PlayerCount: 1, NoStarters: true}); err != nil {
		t.Fatal(err)
	}
	f, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	id := 9002
	recipe := Recipe{GameDir: gameStrings, Messages: &MessagesRecipe{Instructions: &MessageSlotRecipe{StringTableID: &id}}}
	if _, err := f.Plan(recipe); err != nil {
		t.Fatal(err)
	}
	if err := f.ApplyRecipe(recipe); err != nil {
		t.Fatal(err)
	}
	settings := f.Settings().Messages.Instructions
	if settings.StringTableID != id || !settings.UsesStringTableID || settings.Literal != "Select Object" || settings.EffectiveSource != "string_table_id" {
		t.Fatalf("message settings = %+v", settings)
	}
	if err := f.Write(path); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := reopened.Settings().Messages.Instructions; got != settings {
		t.Fatalf("reopened message settings = %+v, want %+v", got, settings)
	}
}

func TestMessageRecipeWithoutGameStringsPreservesIDsAndWarns(t *testing.T) {
	path := filepath.Join(t.TempDir(), "messages.aoe2scenario")
	if _, err := WriteBlankScenarioFile(path, BlankOptions{PlayerCount: 1, NoStarters: true}); err != nil {
		t.Fatal(err)
	}
	f, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	id := 9002
	literal := "local fallback"
	recipe := Recipe{GameDir: filepath.Join(t.TempDir(), "no-game-here"), Messages: &MessagesRecipe{
		Instructions: &MessageSlotRecipe{StringTableID: &id},
		Victory:      &MessageSlotRecipe{StringTableID: &id, Literal: &literal},
	}}
	plan, err := f.Plan(recipe)
	if err != nil {
		t.Fatalf("Plan without game strings: %v", err)
	}
	if len(plan.Warnings) != 1 {
		t.Fatalf("plan warnings=%v, want one unresolved-string warning", plan.Warnings)
	}
	if err := f.ApplyRecipe(recipe); err != nil {
		t.Fatalf("ApplyRecipe without game strings: %v", err)
	}
	if got := f.Settings().Messages.Instructions; got.StringTableID != id || got.Literal != "" {
		t.Fatalf("instructions=%+v, want preserved id and empty literal", got)
	}
	if got := f.Settings().Messages.Victory; got.StringTableID != id || got.Literal != literal {
		t.Fatalf("victory=%+v, want preserved id and supplied literal", got)
	}
}

func TestPatchReportCarriesUnresolvedGameStringWarning(t *testing.T) {
	dir := t.TempDir()
	in := filepath.Join(dir, "in.aoe2scenario")
	out := filepath.Join(dir, "out.aoe2scenario")
	if _, err := WriteBlankScenarioFile(in, BlankOptions{PlayerCount: 1, NoStarters: true}); err != nil {
		t.Fatal(err)
	}
	id := 9002
	report, err := PatchRecipeFile(in, out, Recipe{
		GameDir:  filepath.Join(dir, "missing-install"),
		Messages: &MessagesRecipe{Instructions: &MessageSlotRecipe{StringTableID: &id}},
	})
	if err != nil {
		t.Fatalf("PatchRecipeFile without game strings: %v", err)
	}
	if len(report.Warnings) != 1 {
		t.Fatalf("patch warnings=%v, want one unresolved-string warning", report.Warnings)
	}
	patched, err := Open(out)
	if err != nil {
		t.Fatal(err)
	}
	if got := patched.Settings().Messages.Instructions; got.StringTableID != id || got.Literal != "" {
		t.Fatalf("patched instructions=%+v, want preserved id and empty literal", got)
	}
}
