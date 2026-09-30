package scenario

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"sort"

	"aoe2kit/pkg/datfile"
	"aoe2kit/pkg/testfixtures"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestLoadCurrentDESpec(t *testing.T) {
	spec, err := LoadCurrentDESpec()
	if err != nil {
		t.Fatalf("LoadCurrentDESpec: %v", err)
	}
	if len(spec.Sections) == 0 {
		t.Fatal("expected sections")
	}
	if spec.Sections[0].Name != "DataHeader" {
		t.Fatalf("first body section = %q, want DataHeader", spec.Sections[0].Name)
	}
	if _, ok := spec.section("FileHeader"); ok {
		t.Fatal("FileHeader must be owned by the typed codec, not the body descriptor")
	}
	if _, ok := spec.section("Triggers"); !ok {
		t.Fatal("missing Triggers section")
	}
}

func TestScenarioVersionSupport(t *testing.T) {
	for _, version := range []string{"1.58", "1.59"} {
		if !SupportsReadVersion(version) {
			t.Fatalf("SupportsReadVersion(%q) = false", version)
		}
	}
	for _, version := range []string{"1.55", "1.56", "1.57"} {
		if SupportsReadVersion(version) {
			t.Fatalf("SupportsReadVersion(%q) = true", version)
		}
	}
	if SupportsReadVersion("1.54") {
		t.Fatal("SupportsReadVersion(1.54) = true")
	}
	if !SupportsWriteVersion("1.58") {
		t.Fatal("SupportsWriteVersion(1.58) = false")
	}
	if !SupportsWriteVersion("1.57") {
		t.Fatal("SupportsWriteVersion(1.57) = false")
	}
	if !SupportsWriteVersion("1.59") {
		t.Fatal("SupportsWriteVersion(1.59) = false")
	}
}

func TestUnsupported157HeaderFailsBeforeBodyParse(t *testing.T) {
	data, err := BlankScenarioSeedBytes()
	if err != nil {
		t.Fatalf("BlankScenarioSeedBytes: %v", err)
	}
	data = bytes.Replace(data, []byte("1.58"), []byte("1.57"), 1)
	if _, err := Parse(data); err == nil || err.Error() != "unsupported scenario version 1.57 (supported: 1.58, 1.59)" {
		t.Fatalf("Parse(1.57) error = %v", err)
	}
}

func TestQueryFieldCorpusReportsUnsupportedAndContinues(t *testing.T) {
	data, err := BlankScenarioSeedBytes()
	if err != nil {
		t.Fatalf("BlankScenarioSeedBytes: %v", err)
	}
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "supported.aoe2scenario"), data, 0o600); err != nil {
		t.Fatalf("write supported fixture: %v", err)
	}
	legacy := bytes.Replace(data, []byte("1.58"), []byte("1.57"), 1)
	if err := os.WriteFile(filepath.Join(root, "legacy.aoe2scenario"), legacy, 0o600); err != nil {
		t.Fatalf("write legacy fixture: %v", err)
	}
	results, err := QueryFieldWithOptions("Map.water_definition", root, FieldQueryOptions{})
	if err != nil {
		t.Fatalf("QueryFieldWithOptions: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("results = %d, want 2", len(results))
	}
	if results[0].Status != "unsupported" || results[0].Version != "1.57" {
		t.Fatalf("legacy result = %#v", results[0])
	}
	if results[1].Status != "ok" || results[1].Version != "1.58" {
		t.Fatalf("supported result = %#v", results[1])
	}
	if _, err := QueryFieldWithOptions("Map.water_definition", root, FieldQueryOptions{Strict: true}); err == nil {
		t.Fatal("strict corpus query unexpectedly succeeded")
	}
}

func TestLoadDESpecForVersion159UsesReadFallback(t *testing.T) {
	spec, err := LoadDESpecForVersion("1.59")
	if err != nil {
		t.Fatalf("LoadDESpecForVersion(1.59): %v", err)
	}
	if _, ok := spec.section("Triggers"); !ok {
		t.Fatal("1.59 fallback spec missing Triggers")
	}
}

func TestExtractPrintableStrings(t *testing.T) {
	got := extractPrintableStrings([]byte("\x00alpha\x00alpha\x01beta text\x00"))
	want := []string{"alpha", "beta text"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("extractPrintableStrings = %#v, want %#v", got, want)
	}
}

func TestBlankScenarioSeedIntegrity(t *testing.T) {
	data, err := BlankScenarioSeedBytes()
	if err != nil {
		t.Fatalf("BlankScenarioSeedBytes: %v", err)
	}
	if len(data) != 763 {
		t.Fatalf("blank seed length = %d, want 763", len(data))
	}
	file, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse blank seed: %v", err)
	}
	if file.Version != "1.58" {
		t.Fatalf("blank seed version = %q, want 1.58", file.Version)
	}
	if file.Triggers == nil || file.Triggers.Count != 0 {
		t.Fatalf("blank seed trigger count = %#v, want 0 editor blank triggers", file.Triggers)
	}
}

func TestWriteBlankScenarioFileMatchesEditorBlankPlayerInit(t *testing.T) {
	editorPath := filepath.Join("..", "..", "testdata", "editor-refs", "Create scenario, do nothing, click save.aoe2scenario")
	editor, err := Open(editorPath)
	if err != nil {
		t.Fatalf("Open editor blank fixture: %v", err)
	}
	out := filepath.Join(t.TempDir(), "KitEditorParityBlank.aoe2scenario")
	if _, err := WriteBlankScenarioFile(out, BlankOptions{Timestamp: 1800000000}); err != nil {
		t.Fatalf("WriteBlankScenarioFile: %v", err)
	}
	kit, err := Open(out)
	if err != nil {
		t.Fatalf("Open kit blank output: %v", err)
	}
	if kit.PlayerCount != editor.PlayerCount {
		t.Fatalf("player_count = %d, want editor %d", kit.PlayerCount, editor.PlayerCount)
	}
	for i, want := range editor.Players {
		got := kit.Players[i]
		if got.Active != want.Active || got.Human != want.Human {
			t.Fatalf("P%d active/human = %t/%t, want editor %t/%t", i, got.Active, got.Human, want.Active, want.Human)
		}
	}
}

func TestWriteBlankScenarioFileInflatedBodyMatchesEditorBlank(t *testing.T) {
	editorPath := filepath.Join("..", "..", "testdata", "editor-refs", "Create scenario, do nothing, click save.aoe2scenario")
	editor, err := Open(editorPath)
	if err != nil {
		t.Fatalf("Open editor blank fixture: %v", err)
	}
	out := filepath.Join(t.TempDir(), "KitEditorBodyParityBlank.aoe2scenario")
	if _, err := WriteBlankScenarioFile(out, BlankOptions{Timestamp: 1800000000, NoStarters: true}); err != nil {
		t.Fatalf("WriteBlankScenarioFile: %v", err)
	}
	kit, err := Open(out)
	if err != nil {
		t.Fatalf("Open kit blank output: %v", err)
	}
	if !bytes.Equal(kit.body, editor.body) {
		for i := 0; i < len(kit.body) && i < len(editor.body); i++ {
			if kit.body[i] != editor.body[i] {
				t.Fatalf("inflated body differs at offset %d: kit=0x%02x editor=0x%02x", i, kit.body[i], editor.body[i])
			}
		}
		t.Fatalf("inflated body length differs: kit=%d editor=%d", len(kit.body), len(editor.body))
	}
}

func TestEditorReferenceFixturesParseAndRebuild(t *testing.T) {
	dir := filepath.Join("..", "..", "testdata", "editor-refs")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir editor refs: %v", err)
	}
	checked := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".aoe2scenario") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		file, err := Open(path)
		if err != nil {
			t.Fatalf("Open %s: %v", entry.Name(), err)
		}
		if err := file.VerifyRebuild(); err != nil {
			t.Fatalf("VerifyRebuild %s: %v", entry.Name(), err)
		}
		if file.Version != "1.58" {
			t.Fatalf("%s version = %q, want 1.58", entry.Name(), file.Version)
		}
		checked++
	}
	if checked != 9 {
		t.Fatalf("checked %d editor refs, want 9", checked)
	}
}

func TestEditorReferenceObjectRotations(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "editor-refs", "I added the same object, 2 rotations.aoe2scenario")
	file, err := Open(path)
	if err != nil {
		t.Fatalf("Open object fixture: %v", err)
	}
	var rotations []float64
	for _, section := range file.Units.Sections {
		for _, unit := range section.Units {
			if unit.UnitConst == 2411 {
				rotations = append(rotations, unit.Rotation)
			}
		}
	}
	sort.Float64s(rotations)
	if !reflect.DeepEqual(rotations, []float64{0, 3}) {
		t.Fatalf("unit 2411 rotations = %v, want [0 3]", rotations)
	}
}

func TestDiffSurfacesUnitsNumberOfPlayers(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "editor-refs", "Create scenario, do nothing, click save.aoe2scenario")
	before, err := Open(path)
	if err != nil {
		t.Fatalf("Open before: %v", err)
	}
	after, err := Open(path)
	if err != nil {
		t.Fatalf("Open after: %v", err)
	}
	after.Units.NumberOfPlayers = 3
	report := Diff(before, after)
	if report.Same {
		t.Fatalf("diff same=true, want number_of_players change")
	}
	for _, change := range report.Changes {
		if change.Kind == "units" && change.Field == "number_of_players" && change.Before == 9 && change.After == 3 {
			return
		}
	}
	t.Fatalf("diff changes = %+v, want units.number_of_players 9 -> 3", report.Changes)
}

func TestWriteBlankScenarioFileClearsSeedAndAddsBarracksStarters(t *testing.T) {
	out := filepath.Join(t.TempDir(), "BlankRoot.aoe2scenario")
	report, err := WriteBlankScenarioFile(out, BlankOptions{
		PlayerCount:   4,
		HumanSlots:    1,
		Timestamp:     1800000000,
		DummyStarters: true,
	})
	if err != nil {
		t.Fatalf("WriteBlankScenarioFile: %v", err)
	}
	if report.TriggerCountBefore != 0 || report.TriggerCountAfter != 0 {
		t.Fatalf("trigger counts = %d -> %d, want 0 -> 0", report.TriggerCountBefore, report.TriggerCountAfter)
	}
	if report.UnitCountBefore != 0 || report.UnitCountAfter != 4 {
		t.Fatalf("unit counts = %d -> %d, want 0 -> 4", report.UnitCountBefore, report.UnitCountAfter)
	}
	if report.DummyUnit != defaultBlankDummyUnit {
		t.Fatalf("dummy unit = %d, want %d", report.DummyUnit, defaultBlankDummyUnit)
	}
	if report.EditorParityOK || !strings.Contains(report.EditorParityNote, "human=false") {
		t.Fatalf("editor parity = %t %q, want explicit active AI slots to diverge from editor-style blank metadata", report.EditorParityOK, report.EditorParityNote)
	}
	file, err := Open(out)
	if err != nil {
		t.Fatalf("Open blank output: %v", err)
	}
	if file.PlayerCount != 4 {
		t.Fatalf("player_count = %d, want 4", file.PlayerCount)
	}
	if file.Triggers == nil || file.Triggers.Count != 0 || !file.Triggers.InvariantOK {
		t.Fatalf("trigger invariant/count = %#v, want zero invariant-ok triggers", file.Triggers)
	}
	if file.Units == nil || file.Units.Total != 4 {
		t.Fatalf("unit total = %#v, want 4", file.Units)
	}
	for player := 1; player <= 4; player++ {
		section := file.Units.Sections[player]
		if section.Count != 1 {
			t.Fatalf("P%d unit count = %d, want 1", player, section.Count)
		}
		unit := section.Units[0]
		if unit.UnitConst != defaultBlankDummyUnit {
			t.Fatalf("P%d dummy unit = %d, want %d", player, unit.UnitConst, defaultBlankDummyUnit)
		}
		if unit.CaptionString != fmt.Sprintf("A2K_BLANK_DUMMY_P%d", player) {
			t.Fatalf("P%d caption = %q", player, unit.CaptionString)
		}
	}
	settings := file.Settings()
	if settings.Players[0].Active != true || settings.Players[0].Human != true {
		t.Fatalf("P1 settings = %+v, want active human", settings.Players[0])
	}
	if settings.Players[0].Active != true || settings.Players[0].Human != true {
		t.Fatalf("editor slot 0 = %+v, want active human P1 metadata", settings.Players[0])
	}
	if settings.Players[1].Active != true || settings.Players[1].Human != false {
		t.Fatalf("P2 settings = %+v, want active computer from explicit --human-slots behavior", settings.Players[1])
	}
	if settings.Players[5].Active != false {
		t.Fatalf("P5 active = true, want inactive")
	}
	if settings.Players[9].Active != false || settings.Players[9].Human != true {
		t.Fatalf("P9 settings = %+v, want inactive human metadata", settings.Players[9])
	}
	if file.Units.NumberOfUnitSections != 9 || file.Units.NumberOfPlayers != 9 {
		t.Fatalf("units section counts = %d/%d, want 9/9", file.Units.NumberOfUnitSections, file.Units.NumberOfPlayers)
	}
	checkIntField(t, file.root.section("Units"), "number_of_players", 9)
	checkIntField(t, file.headerRoot, "timestamp_of_last_save", 1800000000)
}

func TestWriteBlankScenarioFileSupportsDiagnosticBlankFlags(t *testing.T) {
	out := filepath.Join(t.TempDir(), "DiagnosticBlank.aoe2scenario")
	report, err := WriteBlankScenarioFile(out, BlankOptions{
		PlayerCount:   4,
		HumanSlots:    1,
		Timestamp:     1800000000,
		DummyStarters: true,
		NoConquest:    true,
	})
	if err != nil {
		t.Fatalf("WriteBlankScenarioFile: %v", err)
	}
	if !report.GaiaActive {
		t.Fatal("report GaiaActive=false, want editor-style active Gaia")
	}
	if !report.NoConquest {
		t.Fatal("report NoConquest=false, want true")
	}
	file, err := Open(out)
	if err != nil {
		t.Fatalf("Open blank output: %v", err)
	}
	settings := file.Settings()
	if !settings.Players[0].Active {
		t.Fatalf("editor slot 0 = %+v, want active P1", settings.Players[0])
	}
	for player := 0; player < 4; player++ {
		if !settings.Players[player].Active {
			t.Fatalf("P%d inactive, want active", player)
		}
	}
	if settings.Players[5].Active {
		t.Fatalf("P5 active, want inactive")
	}
	if got := settings.Victory["conquest_required"]; got != 0 {
		t.Fatalf("conquest_required = %d, want 0", got)
	}
	checkIntField(t, file.root.section("Units"), "number_of_players", 9)
}

func TestWriteBlankScenarioFileCanAddTimerCloseout(t *testing.T) {
	out := filepath.Join(t.TempDir(), "DiagnosticCloseout.aoe2scenario")
	report, err := WriteBlankScenarioFile(out, BlankOptions{
		PlayerCount:     4,
		HumanSlots:      1,
		Timestamp:       1800000000,
		DummyStarters:   true,
		NoConquest:      true,
		CloseoutSeconds: 120,
	})
	if err != nil {
		t.Fatalf("WriteBlankScenarioFile: %v", err)
	}
	if report.CloseoutSeconds != 120 || report.CloseoutPlayer != 1 {
		t.Fatalf("closeout report = %d/P%d, want 120/P1", report.CloseoutSeconds, report.CloseoutPlayer)
	}
	if report.TriggerCountAfter != 1 {
		t.Fatalf("trigger count after = %d, want 1", report.TriggerCountAfter)
	}
	file, err := Open(out)
	if err != nil {
		t.Fatalf("Open blank output: %v", err)
	}
	triggers := file.root.section("Triggers").list("trigger_data")
	if len(triggers) != 1 {
		t.Fatalf("triggers = %d, want 1", len(triggers))
	}
	trigger := triggers[0]
	if name, _ := trigger.stringValue("trigger_name"); name != "A2K Blank Closeout - declare victory" {
		t.Fatalf("trigger name = %q", name)
	}
	conditions := trigger.list("condition_data")
	if len(conditions) != 1 {
		t.Fatalf("conditions = %d, want 1", len(conditions))
	}
	checkIntField(t, conditions[0], "condition_type", 10)
	checkIntField(t, conditions[0], "timer", 120)
	effects := trigger.list("effect_data")
	if len(effects) != 1 {
		t.Fatalf("effects = %d, want 1", len(effects))
	}
	checkIntField(t, effects[0], "effect_type", 13)
	checkIntField(t, effects[0], "source_player", 1)
}

func TestWriteBlankScenarioFileRejectsInvalidCloseout(t *testing.T) {
	out := filepath.Join(t.TempDir(), "BadCloseout.aoe2scenario")
	if _, err := WriteBlankScenarioFile(out, BlankOptions{PlayerCount: 2, CloseoutSeconds: 30, CloseoutPlayer: 3}); err == nil {
		t.Fatal("expected invalid closeout player error")
	}
	if _, err := WriteBlankScenarioFile(out, BlankOptions{PlayerCount: 2, CloseoutSeconds: -1}); err == nil {
		t.Fatal("expected invalid closeout seconds error")
	}
}

func TestApplyRecipeBulkAddUnitsAssignsSequentialReferences(t *testing.T) {
	out := filepath.Join(t.TempDir(), "BulkUnits.aoe2scenario")
	if _, err := WriteBlankScenarioFile(out, BlankOptions{Timestamp: 1800000000, NoStarters: true}); err != nil {
		t.Fatalf("WriteBlankScenarioFile: %v", err)
	}
	file, err := Open(out)
	if err != nil {
		t.Fatalf("Open blank output: %v", err)
	}
	x := 10.5
	y := 20.5
	explicit := 50
	recipe := Recipe{Units: []UnitRecipe{
		{Op: "add_unit", Player: 1, UnitConst: 83, X: &x, Y: &y},
		{Op: "add_unit", Player: 1, UnitConst: 83, X: &x, Y: &y},
		{Op: "add_unit", Player: 1, UnitConst: 83, X: &x, Y: &y, ReferenceID: &explicit},
		{Op: "add_unit", Player: 1, UnitConst: 83, X: &x, Y: &y},
	}}
	if err := file.ApplyRecipe(recipe); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	if file.Units.Total != 4 {
		t.Fatalf("unit total = %d, want 4", file.Units.Total)
	}
	gotRefs := make([]int, 0, 4)
	for _, unit := range file.Units.Sections[1].Units {
		gotRefs = append(gotRefs, unit.ReferenceID)
	}
	wantRefs := []int{1, 2, 50, 51}
	if !reflect.DeepEqual(gotRefs, wantRefs) {
		t.Fatalf("refs = %v, want %v", gotRefs, wantRefs)
	}
	checkIntField(t, file.root.section("Units"), "number_of_players", 9)
}

func TestWriteBlankScenarioFileResizesMap(t *testing.T) {
	out := filepath.Join(t.TempDir(), "Blank220.aoe2scenario")
	report, err := WriteBlankScenarioFile(out, BlankOptions{
		PlayerCount:   2,
		HumanSlots:    1,
		MapWidth:      220,
		MapHeight:     220,
		Timestamp:     1800000000,
		GaiaActive:    true,
		DummyStarters: true,
	})
	if err != nil {
		t.Fatalf("WriteBlankScenarioFile: %v", err)
	}
	if report.MapWidth != 220 || report.MapHeight != 220 || report.TileCount != 220*220 {
		t.Fatalf("blank report map = %dx%d tiles=%d, want 220x220 tiles=%d", report.MapWidth, report.MapHeight, report.TileCount, 220*220)
	}
	file, err := Open(out)
	if err != nil {
		t.Fatalf("Open blank output: %v", err)
	}
	if file.Map == nil || file.Map.Width != 220 || file.Map.Height != 220 || file.Map.TileCount != 220*220 {
		t.Fatalf("file map = %#v, want 220x220", file.Map)
	}
	tiles, width, height, err := file.mapTiles()
	if err != nil {
		t.Fatalf("mapTiles: %v", err)
	}
	if width != 220 || height != 220 || len(tiles) != 220*220 {
		t.Fatalf("mapTiles = %dx%d len=%d, want 220x220 len=%d", width, height, len(tiles), 220*220)
	}
	terrain, err := file.Terrain(TerrainOptions{})
	if err != nil {
		t.Fatalf("Terrain: %v", err)
	}
	if terrain.Width != 220 || terrain.Height != 220 || terrain.TileCount != 220*220 {
		t.Fatalf("terrain report = %dx%d tiles=%d", terrain.Width, terrain.Height, terrain.TileCount)
	}
	if len(terrain.Aggregates) != 1 || terrain.Aggregates[0].Count != 220*220 {
		t.Fatalf("terrain aggregates = %#v, want one full-map terrain", terrain.Aggregates)
	}
}

func TestScenarioMapPresetsMatchGameDataStringIDs(t *testing.T) {
	want := []ScenarioMapPreset{
		{Name: "Mini", Size: 80, StringID: 25080, StringKey: "MAPSIZE_MINI"},
		{Name: "Tiny", Size: 120, StringID: 25120, StringKey: "MAPSIZE_TINY"},
		{Name: "Small", Size: 144, StringID: 25144, StringKey: "MAPSIZE_SMALL"},
		{Name: "Medium", Size: 168, StringID: 25168, StringKey: "MAPSIZE_MEDIUM"},
		{Name: "Normal", Size: 200, StringID: 25200, StringKey: "MAPSIZE_NORMAL"},
		{Name: "Large", Size: 220, StringID: 25220, StringKey: "MAPSIZE_LARGE"},
		{Name: "Huge", Size: 240, StringID: 25240, StringKey: "MAPSIZE_HUGE"},
		{Name: "Giant", Size: 252, StringID: 25252, StringKey: "MAPSIZE_GIANT"},
		{Name: "Massive", Size: 276, StringID: 25276, StringKey: "MAPSIZE_MASSIVE"},
		{Name: "Enormous", Size: 300, StringID: 25300, StringKey: "MAPSIZE_ENORMOUS"},
		{Name: "Colossal", Size: 320, StringID: 25320, StringKey: "MAPSIZE_COLOSSAL"},
		{Name: "Incredible", Size: 360, StringID: 25360, StringKey: "MAPSIZE_INCREDIBLE"},
		{Name: "Monstrous", Size: 400, StringID: 25400, StringKey: "MAPSIZE_MONSTROUS"},
		{Name: "Ludicrous", Size: 480, StringID: 25480, StringKey: "MAPSIZE_LUDICROUS"},
	}
	if !reflect.DeepEqual(ScenarioMapPresets(), want) {
		t.Fatalf("ScenarioMapPresets mismatch\ngot:  %#v\nwant: %#v", ScenarioMapPresets(), want)
	}
	for _, preset := range want {
		if got, ok := ScenarioMapPresetBySize(preset.Size); !ok || got != preset {
			t.Fatalf("preset lookup %d = %#v,%t want %#v,true", preset.Size, got, ok, preset)
		}
	}
	if _, ok := ScenarioMapPresetBySize(301); ok {
		t.Fatal("off-preset map size 301 accepted")
	}
}

func TestWriteBlankScenarioFileRejectsBadMapSizeAndDummyPlacement(t *testing.T) {
	out := filepath.Join(t.TempDir(), "bad.aoe2scenario")
	if _, err := WriteBlankScenarioFile(out, BlankOptions{MapWidth: 7, MapHeight: 7}); err == nil {
		t.Fatal("accepted map width below canonical presets")
	}
	if _, err := WriteBlankScenarioFile(out, BlankOptions{MapWidth: 481, MapHeight: 481}); err == nil {
		t.Fatal("accepted map height above canonical presets")
	}
	if _, err := WriteBlankScenarioFile(out, BlankOptions{MapWidth: 300, MapHeight: 300}); err != nil {
		t.Fatalf("canonical map size 300 rejected: %v", err)
	}
	if _, err := WriteBlankScenarioFile(out, BlankOptions{MapWidth: 301, MapHeight: 301}); err == nil {
		t.Fatal("accepted off-preset square map size")
	}
	if _, err := WriteBlankScenarioFile(out, BlankOptions{MapWidth: 220, MapHeight: 240}); err == nil {
		t.Fatal("accepted non-square map size")
	}
	_, err := WriteBlankScenarioFile(out, BlankOptions{
		PlayerCount:   2,
		HumanSlots:    1,
		MapWidth:      80,
		MapHeight:     80,
		DummyStartX:   79.5,
		DummyStartY:   79.5,
		DummySpacing:  6,
		DummyStarters: true,
	})
	if err == nil || !strings.Contains(err.Error(), "outside map") {
		t.Fatalf("dummy placement err = %v, want outside map", err)
	}
}

func TestMaxInflatedScenarioBytes(t *testing.T) {
	t.Setenv("AOE2KIT_ALLOW_HUGE_SCENARIO", "")
	t.Setenv("AOE2KIT_MAX_SCENARIO_MB", "")
	if got := maxInflatedScenarioBytes(); got != defaultMaxInflatedScenarioBytes {
		t.Fatalf("default max = %d, want %d", got, defaultMaxInflatedScenarioBytes)
	}
	t.Setenv("AOE2KIT_MAX_SCENARIO_MB", "2")
	if got := maxInflatedScenarioBytes(); got != 2*1024*1024 {
		t.Fatalf("env max = %d, want 2 MiB", got)
	}
	t.Setenv("AOE2KIT_ALLOW_HUGE_SCENARIO", "1")
	if got := maxInflatedScenarioBytes(); got != 0 {
		t.Fatalf("allow huge max = %d, want 0", got)
	}
}

func TestInflateRawLimitedStopsAboveLimit(t *testing.T) {
	compressed, err := DeflateRaw([]byte(strings.Repeat("x", 128)))
	if err != nil {
		t.Fatalf("DeflateRaw: %v", err)
	}
	if _, err := inflateRawLimited(compressed, 64); err == nil || !strings.Contains(err.Error(), "safety limit") {
		t.Fatalf("inflateRawLimited error = %v, want safety limit", err)
	}
	body, err := inflateRawLimited(compressed, 128)
	if err != nil {
		t.Fatalf("inflateRawLimited exact limit: %v", err)
	}
	if len(body) != 128 {
		t.Fatalf("body len = %d, want 128", len(body))
	}
}

func TestDE155SpecUsesLegacyPlayerDataShape(t *testing.T) {
	spec, err := LoadDESpecForVersion("1.55")
	if err != nil {
		t.Fatalf("LoadDESpecForVersion: %v", err)
	}
	dataHeader, ok := spec.section("DataHeader")
	if !ok {
		t.Fatal("missing DataHeader")
	}
	playerData := dataHeader.Structs["PlayerDataOneStruct"]
	var fields []string
	for _, field := range playerData.Fields {
		fields = append(fields, field.Name+":"+field.Type)
	}
	got := strings.Join(fields, ",")
	want := "active:u32,human:u32,civilization:u32,architecture_set:u32,cty_mode:u32"
	if got != want {
		t.Fatalf("PlayerDataOneStruct = %s, want %s", got, want)
	}
}

func TestOlderDEEffectSpecsRemoveNewerFields(t *testing.T) {
	tests := []struct {
		version string
		absent  []string
	}{
		{
			version: "1.55",
			absent: []string{
				"issue_group_command",
				"queue_action",
				"mutual_diplomacy",
				"building_list",
				"wall_x1",
				"wall_y1",
				"wall_x2",
				"wall_y2",
				"object_filter",
				"use_tag_color_for_icon",
			},
		},
		{
			version: "1.56",
			absent: []string{
				"mutual_diplomacy",
				"building_list",
				"wall_x1",
				"wall_y1",
				"wall_x2",
				"wall_y2",
				"object_filter",
				"use_tag_color_for_icon",
			},
		},
		{
			version: "1.57",
			absent: []string{
				"object_filter",
				"use_tag_color_for_icon",
			},
		},
	}
	for _, test := range tests {
		spec, err := LoadDESpecForVersion(test.version)
		if err != nil {
			t.Fatalf("LoadDESpecForVersion(%s): %v", test.version, err)
		}
		triggers, ok := spec.section("Triggers")
		if !ok {
			t.Fatal("missing Triggers section")
		}
		effects := triggers.Structs["TriggerStruct"].Structs["EffectStruct"]
		fields := map[string]bool{}
		for _, field := range effects.Fields {
			fields[field.Name] = true
		}
		for _, name := range test.absent {
			if fields[name] {
				t.Fatalf("version %s still has EffectStruct.%s", test.version, name)
			}
		}
		if !fields["message"] {
			t.Fatalf("version %s missing EffectStruct.message", test.version)
		}
	}
}

func TestWriteRejectsUnsupportedScenarioVersions(t *testing.T) {
	out := filepath.Join(t.TempDir(), "readonly.aoe2scenario")
	err := (&File{Version: "1.54"}).Write(out)
	if err == nil || !strings.Contains(err.Error(), "read-only") {
		t.Fatalf("Write error = %v, want read-only version error", err)
	}
}

func TestIntListNormalizesScalarPrimitive(t *testing.T) {
	node := &parsedNode{
		Fields: []*parsedNode{
			{Name: "selected_object_ids", Value: int32(12345)},
		},
	}
	got := node.intList("selected_object_ids")
	if len(got) != 1 || got[0] != 12345 {
		t.Fatalf("intList = %#v, want [12345]", got)
	}
}

func TestValidateOptionalOrder(t *testing.T) {
	if ok, note := validateOptionalOrder(nil, 3, "effect", 4); !ok || note != "" {
		t.Fatalf("empty optional order = %v %q, want ok", ok, note)
	}
	if ok, note := validateOptionalOrder([]int{2, 0, 1}, 3, "effect", 4); !ok || note != "" {
		t.Fatalf("valid optional order = %v %q, want ok", ok, note)
	}
	if ok, _ := validateOptionalOrder([]int{0, 0}, 2, "effect", 4); ok {
		t.Fatal("duplicate optional order accepted")
	}
	if ok, _ := validateOptionalOrder([]int{0, 2}, 2, "effect", 4); ok {
		t.Fatal("out-of-range optional order accepted")
	}
}

func TestBuildDisplayInstructionTriggerRaw(t *testing.T) {
	spec, err := LoadCurrentDESpec()
	if err != nil {
		t.Fatalf("LoadCurrentDESpec: %v", err)
	}
	raw, err := buildDisplayInstructionTrigger(spec, TriggerRecipe{
		Name:    "Smoke",
		Message: "hello",
	})
	if err != nil {
		t.Fatalf("buildDisplayInstructionTrigger: %v", err)
	}
	triggerSpec, _, err := triggerEffectSpecs(spec)
	if err != nil {
		t.Fatalf("triggerEffectSpecs: %v", err)
	}
	p := parser{data: raw, sections: map[string]*parsedSection{}}
	node, err := p.parseNode("TriggerStruct", triggerSpec, nil)
	if err != nil {
		t.Fatalf("parse trigger raw: %v", err)
	}
	if p.off != len(raw) {
		t.Fatalf("parse consumed %d bytes, want %d", p.off, len(raw))
	}
	name, _ := node.stringValue("trigger_name")
	if name != "Smoke" {
		t.Fatalf("trigger_name = %q, want Smoke", name)
	}
	effects := node.list("effect_data")
	if len(effects) != 1 {
		t.Fatalf("effects = %d, want 1", len(effects))
	}
	effectType, _ := effects[0].intValue("effect_type")
	if effectType != 20 {
		t.Fatalf("effect_type = %d, want 20", effectType)
	}
	message, _ := effects[0].stringValue("message")
	if message != "hello" {
		t.Fatalf("message = %q, want hello", message)
	}
	assertLengthPrefixedStringRaw(t, node.field("trigger_name").Raw, "Smoke")
	assertLengthPrefixedStringRaw(t, effects[0].field("message").Raw, "hello")
}

func TestBuildCreateObjectTriggerRaw(t *testing.T) {
	spec, err := LoadCurrentDESpec()
	if err != nil {
		t.Fatalf("LoadCurrentDESpec: %v", err)
	}
	unitID := 83
	sourcePlayer := 0
	x := 12
	y := 34
	raw, err := buildCreateObjectTrigger(spec, TriggerRecipe{
		Name:             "Create Smoke",
		ObjectListUnitID: &unitID,
		SourcePlayer:     &sourcePlayer,
		LocationX:        &x,
		LocationY:        &y,
	})
	if err != nil {
		t.Fatalf("buildCreateObjectTrigger: %v", err)
	}
	triggerSpec, _, err := triggerEffectSpecs(spec)
	if err != nil {
		t.Fatalf("triggerEffectSpecs: %v", err)
	}
	p := parser{data: raw, sections: map[string]*parsedSection{}}
	node, err := p.parseNode("TriggerStruct", triggerSpec, nil)
	if err != nil {
		t.Fatalf("parse trigger raw: %v", err)
	}
	effects := node.list("effect_data")
	if len(effects) != 1 {
		t.Fatalf("effects = %d, want 1", len(effects))
	}
	effect := effects[0]
	checkIntField(t, effect, "effect_type", 11)
	checkIntField(t, effect, "object_list_unit_id", unitID)
	checkIntField(t, effect, "source_player", sourcePlayer)
	checkIntField(t, effect, "location_x", x)
	checkIntField(t, effect, "location_y", y)
}

func checkIntField(t *testing.T, node *parsedNode, name string, want int) {
	t.Helper()
	got, ok := node.intValue(name)
	if !ok {
		t.Fatalf("%s missing", name)
	}
	if got != want {
		t.Fatalf("%s = %d, want %d", name, got, want)
	}
}

func summaryIntField(t *testing.T, fields map[string]any, name string) int {
	t.Helper()
	value, ok := fields[name]
	if !ok {
		t.Fatalf("summary field %s missing", name)
	}
	intValue, ok := value.(int)
	if !ok {
		t.Fatalf("summary field %s = %T(%v), want int", name, value, value)
	}
	return intValue
}

func checkFloatField(t *testing.T, node *parsedNode, name string, want float64) {
	t.Helper()
	got, ok := node.floatValue(name)
	if !ok {
		t.Fatalf("%s missing", name)
	}
	if math.Abs(got-want) > 0.001 {
		t.Fatalf("%s = %.4f, want %.4f", name, got, want)
	}
}

func assertLengthPrefixedStringRaw(t *testing.T, raw []byte, want string) {
	t.Helper()
	wantPayload := want + "\x00"
	for _, prefixSize := range []int{2, 4} {
		if len(raw) < prefixSize {
			continue
		}
		if signedInt(raw[:prefixSize]) == len([]byte(wantPayload)) && string(raw[prefixSize:]) == wantPayload {
			return
		}
	}
	t.Fatalf("string raw = % x, want length=%d payload=%q", raw, len([]byte(wantPayload)), wantPayload)
}

func assertZeroLengthStringRaw(t *testing.T, node *parsedNode, name string, wantPrefixBytes int) {
	t.Helper()
	field := node.field(name)
	if field == nil {
		t.Fatalf("%s missing", name)
	}
	if len(field.Raw) != wantPrefixBytes {
		t.Fatalf("%s raw len = %d, want %d raw=% x", name, len(field.Raw), wantPrefixBytes, field.Raw)
	}
	if signedInt(field.Raw) != 0 {
		t.Fatalf("%s raw = % x, want zero-length string prefix", name, field.Raw)
	}
}

func TestBuildGeneralTriggerWithConditionRaw(t *testing.T) {
	spec, err := LoadCurrentDESpec()
	if err != nil {
		t.Fatalf("LoadCurrentDESpec: %v", err)
	}
	timer := 5
	triggerID := 7
	enabled := true
	looping := true
	raw, err := buildTriggerFromRecipe(spec, TriggerRecipe{
		Op:      "add_trigger",
		Name:    "Timer Activate",
		Enabled: &enabled,
		Looping: &looping,
		Conditions: []ConditionRecipe{
			{Op: "timer", Timer: &timer},
		},
		Effects: []EffectRecipe{
			{Op: "activate_trigger", TriggerID: &triggerID},
		},
	})
	if err != nil {
		t.Fatalf("buildTriggerFromRecipe: %v", err)
	}
	triggerSpec, _, err := triggerEffectSpecs(spec)
	if err != nil {
		t.Fatalf("triggerEffectSpecs: %v", err)
	}
	p := parser{data: raw, sections: map[string]*parsedSection{}}
	node, err := p.parseNode("TriggerStruct", triggerSpec, nil)
	if err != nil {
		t.Fatalf("parse trigger raw: %v", err)
	}
	checkIntField(t, node, "enabled", 1)
	checkIntField(t, node, "looping", 1)
	conditions := node.list("condition_data")
	if len(conditions) != 1 {
		t.Fatalf("conditions = %d, want 1", len(conditions))
	}
	checkIntField(t, conditions[0], "condition_type", 10)
	checkIntField(t, conditions[0], "timer", timer)
	effects := node.list("effect_data")
	if len(effects) != 1 {
		t.Fatalf("effects = %d, want 1", len(effects))
	}
	checkIntField(t, effects[0], "effect_type", 8)
	checkIntField(t, effects[0], "trigger_id", triggerID)
}

func TestBuildOwnershipConditionsRaw(t *testing.T) {
	spec, err := LoadCurrentDESpec()
	if err != nil {
		t.Fatalf("LoadCurrentDESpec: %v", err)
	}
	quantity := 3
	unitConst := 600
	player := 2
	areaX1 := 10
	areaY1 := 11
	areaX2 := 12
	areaY2 := 13
	objectState := 2
	raw, err := buildTriggerFromRecipe(spec, TriggerRecipe{
		Op:   "add_trigger",
		Name: "Ownership Conditions",
		Conditions: []ConditionRecipe{
			{Op: "own_objects", Quantity: &quantity, ObjectList: &unitConst, SourcePlayer: &player},
			{Op: "own_fewer_objects", Quantity: &quantity, ObjectList: &unitConst, SourcePlayer: &player, AreaX1: &areaX1, AreaY1: &areaY1, AreaX2: &areaX2, AreaY2: &areaY2, ObjectState: &objectState},
		},
	})
	if err != nil {
		t.Fatalf("buildTriggerFromRecipe: %v", err)
	}
	triggerSpec, _, err := triggerEffectSpecs(spec)
	if err != nil {
		t.Fatalf("triggerEffectSpecs: %v", err)
	}
	p := parser{data: raw, sections: map[string]*parsedSection{}}
	node, err := p.parseNode("TriggerStruct", triggerSpec, nil)
	if err != nil {
		t.Fatalf("parse trigger raw: %v", err)
	}
	conditions := node.list("condition_data")
	if len(conditions) != 2 {
		t.Fatalf("conditions = %d, want 2", len(conditions))
	}
	checkIntField(t, conditions[0], "condition_type", 3)
	checkIntField(t, conditions[0], "quantity", quantity)
	checkIntField(t, conditions[0], "object_list", unitConst)
	checkIntField(t, conditions[0], "source_player", player)
	checkIntField(t, conditions[1], "condition_type", 4)
	checkIntField(t, conditions[1], "quantity", quantity)
	checkIntField(t, conditions[1], "object_list", unitConst)
	checkIntField(t, conditions[1], "source_player", player)
	checkIntField(t, conditions[1], "area_x1", areaX1)
	checkIntField(t, conditions[1], "area_y1", areaY1)
	checkIntField(t, conditions[1], "area_x2", areaX2)
	checkIntField(t, conditions[1], "area_y2", areaY2)
	checkIntField(t, conditions[1], "object_state", objectState)
}

func TestBuildDeclareVictoryTriggerRaw(t *testing.T) {
	spec, err := LoadCurrentDESpec()
	if err != nil {
		t.Fatalf("LoadCurrentDESpec: %v", err)
	}
	timer := 120
	player := 1
	enabled := 1
	raw, err := buildTriggerFromRecipe(spec, TriggerRecipe{
		Op:   "add_trigger",
		Name: "Declare Victory",
		Conditions: []ConditionRecipe{
			{Op: "timer", Timer: &timer},
		},
		Effects: []EffectRecipe{
			{Op: "declare_victory", SourcePlayer: &player, Enabled: &enabled},
		},
	})
	if err != nil {
		t.Fatalf("buildTriggerFromRecipe: %v", err)
	}
	triggerSpec, _, err := triggerEffectSpecs(spec)
	if err != nil {
		t.Fatalf("triggerEffectSpecs: %v", err)
	}
	p := parser{data: raw, sections: map[string]*parsedSection{}}
	node, err := p.parseNode("TriggerStruct", triggerSpec, nil)
	if err != nil {
		t.Fatalf("parse trigger raw: %v", err)
	}
	conditions := node.list("condition_data")
	if len(conditions) != 1 {
		t.Fatalf("conditions = %d, want 1", len(conditions))
	}
	checkIntField(t, conditions[0], "condition_type", 10)
	checkIntField(t, conditions[0], "timer", timer)
	effects := node.list("effect_data")
	if len(effects) != 1 {
		t.Fatalf("effects = %d, want 1", len(effects))
	}
	checkIntField(t, effects[0], "effect_type", 13)
	checkIntField(t, effects[0], "source_player", player)
	checkIntField(t, effects[0], "enabled", enabled)
}

func TestBuildChatAndTimerEffectTriggersRaw(t *testing.T) {
	spec, err := LoadCurrentDESpec()
	if err != nil {
		t.Fatalf("LoadCurrentDESpec: %v", err)
	}
	sourcePlayer := 2
	displayTime := 45
	timeUnit := 1
	timerID := 7
	resetTimer := 1
	raw, err := buildTriggerFromRecipe(spec, TriggerRecipe{
		Op:   "add_trigger",
		Name: "Chat and Timer",
		Effects: []EffectRecipe{
			{Op: "send_chat", SourcePlayer: &sourcePlayer, Message: "<GREEN>Hello"},
			{
				Op:           "display_timer",
				SourcePlayer: &sourcePlayer,
				DisplayTime:  &displayTime,
				TimeUnit:     &timeUnit,
				TimerID:      &timerID,
				ResetTimer:   &resetTimer,
				Message:      "Timer label",
			},
		},
	})
	if err != nil {
		t.Fatalf("buildTriggerFromRecipe: %v", err)
	}
	triggerSpec, _, err := triggerEffectSpecs(spec)
	if err != nil {
		t.Fatalf("triggerEffectSpecs: %v", err)
	}
	p := parser{data: raw, sections: map[string]*parsedSection{}}
	node, err := p.parseNode("TriggerStruct", triggerSpec, nil)
	if err != nil {
		t.Fatalf("parse trigger raw: %v", err)
	}
	effects := node.list("effect_data")
	if len(effects) != 2 {
		t.Fatalf("effects = %d, want 2", len(effects))
	}
	checkIntField(t, effects[0], "effect_type", 3)
	checkIntField(t, effects[0], "source_player", sourcePlayer)
	message, _ := effects[0].stringValue("message")
	if message != "<GREEN>Hello" {
		t.Fatalf("chat message = %q, want <GREEN>Hello", message)
	}
	assertLengthPrefixedStringRaw(t, effects[0].field("message").Raw, "<GREEN>Hello")
	checkIntField(t, effects[1], "effect_type", 37)
	checkIntField(t, effects[1], "source_player", sourcePlayer)
	checkIntField(t, effects[1], "display_time", displayTime)
	checkIntField(t, effects[1], "time_unit", timeUnit)
	checkIntField(t, effects[1], "timer", timerID)
	checkIntField(t, effects[1], "reset_timer", resetTimer)
	message, _ = effects[1].stringValue("message")
	if message != "Timer label" {
		t.Fatalf("timer message = %q, want Timer label", message)
	}
	assertLengthPrefixedStringRaw(t, node.field("trigger_name").Raw, "Chat and Timer")
	assertLengthPrefixedStringRaw(t, effects[1].field("message").Raw, "Timer label")
	timerSummary := summarizeEffect(effects[1], EffectsOptions{})
	if timerSummary.TypeName != "display_timer" {
		t.Fatalf("timer type_name = %q, want display_timer", timerSummary.TypeName)
	}
	if len(timerSummary.RawFields) != 0 {
		t.Fatalf("default timer raw_fields len = %d, want 0", len(timerSummary.RawFields))
	}
	for key, want := range map[string]any{
		"source_player": sourcePlayer,
		"display_time":  displayTime,
		"time_unit":     timeUnit,
		"timer_id":      timerID,
		"reset_timer":   resetTimer,
		"message":       "Timer label",
	} {
		if got := timerSummary.KnownFields[key]; got != want {
			t.Fatalf("timer known_fields[%s] = %#v, want %#v", key, got, want)
		}
	}
	rawSummary := summarizeEffect(effects[1], EffectsOptions{IncludeRawFields: true})
	if len(rawSummary.RawFields) == 0 {
		t.Fatal("raw timer summary has empty raw_fields")
	}
}

func TestBuildVariableAndClearTimerEffectTriggersRaw(t *testing.T) {
	spec, err := LoadCurrentDESpec()
	if err != nil {
		t.Fatalf("LoadCurrentDESpec: %v", err)
	}
	variable := 9
	variable2 := 10
	quantity := 77
	operation := 1
	timerID := 3
	raw, err := buildTriggerFromRecipe(spec, TriggerRecipe{
		Op:   "add_trigger",
		Name: "Variable and Timer Effects",
		Effects: []EffectRecipe{
			{Op: "modify_variable", Variable: &variable, Quantity: &quantity, Operation: &operation, Message: "set var"},
			{Op: "modify_variable_by_variable", Variable: &variable, Variable2: &variable2, Operation: &operation},
			{Op: "clear_timer", TimerID: &timerID},
		},
	})
	if err != nil {
		t.Fatalf("buildTriggerFromRecipe: %v", err)
	}
	triggerSpec, _, err := triggerEffectSpecs(spec)
	if err != nil {
		t.Fatalf("triggerEffectSpecs: %v", err)
	}
	p := parser{data: raw, sections: map[string]*parsedSection{}}
	node, err := p.parseNode("TriggerStruct", triggerSpec, nil)
	if err != nil {
		t.Fatalf("parse trigger raw: %v", err)
	}
	effects := node.list("effect_data")
	if len(effects) != 3 {
		t.Fatalf("effects = %d, want 3", len(effects))
	}
	checkIntField(t, effects[0], "effect_type", 56)
	checkIntField(t, effects[0], "variable", variable)
	checkIntField(t, effects[0], "quantity", quantity)
	checkIntField(t, effects[1], "effect_type", 100)
	checkIntField(t, effects[1], "variable", variable)
	checkIntField(t, effects[1], "variable2", variable2)
	checkIntField(t, effects[2], "effect_type", 57)
	checkIntField(t, effects[2], "timer", timerID)
}

func TestReadPrimitiveTruncatesStringsAtFirstNUL(t *testing.T) {
	payload := []byte("Visible\x00hidden")
	raw := make([]byte, 4+len(payload))
	writeSignedBytes(raw[:4], int64(len(payload)))
	copy(raw[4:], payload)
	p := parser{data: raw}
	got, gotRaw, err := p.readPrimitive("str32")
	if err != nil {
		t.Fatalf("read str32: %v", err)
	}
	if got != "Visible" {
		t.Fatalf("str32 = %q, want Visible", got)
	}
	if len(gotRaw) != len(raw) {
		t.Fatalf("raw len = %d, want %d", len(gotRaw), len(raw))
	}

	p = parser{data: []byte("Fixed\x00XX")}
	got, _, err = p.readPrimitive("c8")
	if err != nil {
		t.Fatalf("read c8: %v", err)
	}
	if got != "Fixed" {
		t.Fatalf("c8 = %q, want Fixed", got)
	}
}

func TestReadPrimitiveTreatsNegativeOneStringLengthAsEmptySentinel(t *testing.T) {
	raw := []byte{0xff, 0xff, 0xff, 0xff}
	p := parser{data: raw}
	got, gotRaw, err := p.readPrimitive("str32")
	if err != nil {
		t.Fatalf("read str32 sentinel: %v", err)
	}
	if got != "" {
		t.Fatalf("str32 sentinel = %q, want empty", got)
	}
	if !bytes.Equal(gotRaw, raw) {
		t.Fatalf("raw sentinel = %x, want %x", gotRaw, raw)
	}
}

func TestEncodePrimitiveStringsUseInclusiveNULTerminator(t *testing.T) {
	raw, err := encodePrimitive("str32", "SPAWN ROSTER BOARD")
	if err != nil {
		t.Fatalf("encodePrimitive: %v", err)
	}
	assertLengthPrefixedStringRaw(t, raw, "SPAWN ROSTER BOARD")

	raw, err = encodePrimitive("str16", "Clean\x00garbage")
	if err != nil {
		t.Fatalf("encodePrimitive embedded NUL: %v", err)
	}
	assertLengthPrefixedStringRaw(t, raw, "Clean")
}

func TestBuildDisplayInstructionTriggerPreservesUTF8MarkupRaw(t *testing.T) {
	spec, err := LoadCurrentDESpec()
	if err != nil {
		t.Fatalf("LoadCurrentDESpec: %v", err)
	}
	message := "<AQUA>~ WELCOME \u2605 ~<PURPLE>\u00c6ther caf\u00e9"
	raw, err := buildDisplayInstructionTrigger(spec, TriggerRecipe{
		Name:    "UTF8 Markup",
		Message: message,
	})
	if err != nil {
		t.Fatalf("buildDisplayInstructionTrigger: %v", err)
	}
	triggerSpec, _, err := triggerEffectSpecs(spec)
	if err != nil {
		t.Fatalf("triggerEffectSpecs: %v", err)
	}
	p := parser{data: raw, sections: map[string]*parsedSection{}}
	node, err := p.parseNode("TriggerStruct", triggerSpec, nil)
	if err != nil {
		t.Fatalf("parse trigger raw: %v", err)
	}
	if p.off != len(raw) {
		t.Fatalf("parse consumed %d bytes, want %d", p.off, len(raw))
	}
	assertLengthPrefixedStringRaw(t, node.field("trigger_name").Raw, "UTF8 Markup")
	effects := node.list("effect_data")
	if len(effects) != 1 {
		t.Fatalf("effect count = %d, want 1", len(effects))
	}
	got, _ := effects[0].stringValue("message")
	if got != message {
		t.Fatalf("message = %q, want %q", got, message)
	}
	assertLengthPrefixedStringRaw(t, effects[0].field("message").Raw, message)
}

func TestLintCorruptStringsFlagsInvalidUTF8AndEmbeddedNUL(t *testing.T) {
	raw := make([]byte, 4+len("Visible\x00hidden"))
	writeSignedBytes(raw[:4], int64(len("Visible\x00hidden")))
	copy(raw[4:], "Visible\x00hidden")
	report := LintReport{}
	lintCorruptStrings(&report, &parsedNode{
		Name:  "effect.message",
		Start: 100,
		Raw:   raw,
		Value: "Visible",
	}, &parsedNode{
		Name:  "bad.utf8",
		Start: 200,
		Raw:   []byte{0xff, 0xfe},
		Value: string([]byte{0xff, 0xfe}),
	})
	if !hasIssueCode(report.Issues, "corrupt_string_embedded_nul") {
		t.Fatalf("missing embedded NUL issue: %+v", report.Issues)
	}
	if !hasIssueCode(report.Issues, "corrupt_string_invalid_utf8") {
		t.Fatalf("missing invalid UTF-8 issue: %+v", report.Issues)
	}
}

func TestBuildGeneralTriggerPrimitiveSetRaw(t *testing.T) {
	spec, err := LoadCurrentDESpec()
	if err != nil {
		t.Fatalf("LoadCurrentDESpec: %v", err)
	}
	unitObject := 1234
	triggerID := 3
	actionType := 1
	selectedA := 44
	selectedB := 45
	raw, err := buildTriggerFromRecipe(spec, TriggerRecipe{
		Op:   "add_trigger",
		Name: "Primitive Set",
		Conditions: []ConditionRecipe{
			{Op: "object_selected", UnitObject: &unitObject},
		},
		Effects: []EffectRecipe{
			{Op: "kill_object", SelectedObjectIDs: []int{selectedA, selectedB}},
			{Op: "task_object", ActionType: &actionType, SelectedObjectIDs: []int{selectedA}},
			{Op: "deactivate_trigger", TriggerID: &triggerID},
		},
	})
	if err != nil {
		t.Fatalf("buildTriggerFromRecipe: %v", err)
	}
	triggerSpec, _, err := triggerEffectSpecs(spec)
	if err != nil {
		t.Fatalf("triggerEffectSpecs: %v", err)
	}
	p := parser{data: raw, sections: map[string]*parsedSection{}}
	node, err := p.parseNode("TriggerStruct", triggerSpec, nil)
	if err != nil {
		t.Fatalf("parse trigger raw: %v", err)
	}
	conditions := node.list("condition_data")
	if len(conditions) != 1 {
		t.Fatalf("conditions = %d, want 1", len(conditions))
	}
	checkIntField(t, conditions[0], "condition_type", 11)
	checkIntField(t, conditions[0], "unit_object", unitObject)
	effects := node.list("effect_data")
	if len(effects) != 3 {
		t.Fatalf("effects = %d, want 3", len(effects))
	}
	checkIntField(t, effects[0], "effect_type", 14)
	checkIntField(t, effects[0], "number_of_units_selected", 2)
	checkIntField(t, effects[1], "effect_type", 12)
	checkIntField(t, effects[1], "action_type", actionType)
	checkIntField(t, effects[1], "number_of_units_selected", 1)
	checkIntField(t, effects[2], "effect_type", 9)
	checkIntField(t, effects[2], "trigger_id", triggerID)
}

func TestSmokeRecipeShape(t *testing.T) {
	recipe := SmokeRecipe(SmokeRecipeOptions{
		Player:       2,
		PlayerSet:    true,
		UnitConst:    74,
		UnitConstSet: true,
		X:            20,
		XSet:         true,
		Y:            30,
		YSet:         true,
		TerrainID:    5,
		TerrainIDSet: true,
		Elevation:    1,
		ElevationSet: true,
		Layer:        -1,
		LayerSet:     true,
	})
	if len(recipe.Triggers) != 3 {
		t.Fatalf("smoke triggers = %d, want 3", len(recipe.Triggers))
	}
	for i, trigger := range recipe.Triggers {
		if trigger.Enabled == nil || *trigger.Enabled {
			t.Fatalf("smoke trigger %d enabled = %v, want disabled", i, trigger.Enabled)
		}
	}
	if len(recipe.Units) != 1 || recipe.Units[0].Player != 2 || recipe.Units[0].UnitConst != 74 {
		t.Fatalf("smoke units = %+v", recipe.Units)
	}
	if recipe.Units[0].X == nil || *recipe.Units[0].X != 21.5 || recipe.Units[0].Y == nil || *recipe.Units[0].Y != 31.5 {
		t.Fatalf("smoke unit location = x %v y %v", recipe.Units[0].X, recipe.Units[0].Y)
	}
	if len(recipe.Map) != 1 || recipe.Map[0].X1 != 20 || recipe.Map[0].Y1 != 30 || recipe.Map[0].X2 != 22 || recipe.Map[0].Y2 != 32 {
		t.Fatalf("smoke map = %+v", recipe.Map)
	}
	if recipe.Map[0].TerrainID == nil || *recipe.Map[0].TerrainID != 5 || recipe.Map[0].Elevation == nil || *recipe.Map[0].Elevation != 1 || recipe.Map[0].Layer == nil || *recipe.Map[0].Layer != -1 {
		t.Fatalf("smoke map fields = %+v", recipe.Map[0])
	}
}

func TestPlanOperationEnabledIsOnlySerializedWhenApplicable(t *testing.T) {
	disabled := false
	plan := Plan{Operations: []PlanOp{
		{Op: "add_trigger", Name: "disabled trigger", Enabled: &disabled},
		{Op: "set_resources", Name: "P1"},
	}}
	data, err := json.Marshal(plan)
	if err != nil {
		t.Fatalf("marshal plan: %v", err)
	}
	text := string(data)
	if !strings.Contains(text, `"op":"add_trigger"`) || !strings.Contains(text, `"enabled":false`) {
		t.Fatalf("trigger enabled field was not preserved: %s", text)
	}
	if strings.Contains(text, `"op":"set_resources","name":"P1","enabled"`) {
		t.Fatalf("non-trigger operation serialized misleading enabled field: %s", text)
	}
}

func hasIssueCode(issues []LintIssue, code string) bool {
	for _, issue := range issues {
		if issue.Code == code {
			return true
		}
	}
	return false
}

func TestBuildAuthoringFrontierTriggerRaw(t *testing.T) {
	spec, err := LoadCurrentDESpec()
	if err != nil {
		t.Fatalf("LoadCurrentDESpec: %v", err)
	}
	tech := 408
	vis := 0
	unitConst := 83
	player := 1
	targetPlayer := 2
	x := 20
	y := 21
	attr := 0
	resource := 3
	quantity := 50
	operation := 1
	stringID := 1234
	unitObject := 12345
	areaX1 := 10
	areaY1 := 11
	areaX2 := 12
	areaY2 := 13
	variable := 7
	comparison := 4
	raw, err := buildTriggerFromRecipe(spec, TriggerRecipe{
		Op:   "add_trigger",
		Name: "Authoring Frontier",
		Conditions: []ConditionRecipe{
			{Op: "object_in_area", SourcePlayer: &player, ObjectList: &unitConst, AreaX1: &areaX1, AreaY1: &areaY1, AreaX2: &areaX2, AreaY2: &areaY2},
			{Op: "accumulate_attribute", SourcePlayer: &player, Attribute: &resource, Quantity: &quantity},
			{Op: "object_visible", UnitObject: &unitObject},
			{Op: "variable_value", Variable: &variable, Comparison: &comparison, Quantity: &quantity},
		},
		Effects: []EffectRecipe{
			{Op: "research_technology", SourcePlayer: &player, Technology: &tech},
			{Op: "reveal_map", SourcePlayer: &player, TargetPlayer: &targetPlayer, VisibilityState: &vis},
			{Op: "modify_attribute", SourcePlayer: &player, ObjectListUnitID: &unitConst, ObjectAttributes: &attr, Quantity: &quantity, Operation: &operation},
			{Op: "modify_resource", SourcePlayer: &player, Resource: &resource, Quantity: &quantity, Operation: &operation},
			{Op: "script_call", SourcePlayer: &player, Message: "main();"},
			{Op: "change_ownership", SourcePlayer: &player, TargetPlayer: &targetPlayer, SelectedObjectIDs: []int{unitObject}},
			{Op: "change_object_name", SourcePlayer: &player, ObjectListUnitID: &unitConst, Message: "Renamed", SelectedObjectIDs: []int{unitObject}},
			{Op: "change_object_description", SourcePlayer: &player, ObjectListUnitID: &unitConst, StringID: &stringID, Message: "Described", SelectedObjectIDs: []int{unitObject}},
			{Op: "change_object_hp", SourcePlayer: &player, Quantity: &quantity, Operation: &operation, SelectedObjectIDs: []int{unitObject}},
			{Op: "teleport_object", SourcePlayer: &player, LocationX: &x, LocationY: &y, SelectedObjectIDs: []int{unitObject}},
			{Op: "remove_object", SourcePlayer: &player, SelectedObjectIDs: []int{unitObject}},
		},
	})
	if err != nil {
		t.Fatalf("buildTriggerFromRecipe: %v", err)
	}
	triggerSpec, _, err := triggerEffectSpecs(spec)
	if err != nil {
		t.Fatalf("triggerEffectSpecs: %v", err)
	}
	p := parser{data: raw, sections: map[string]*parsedSection{}}
	node, err := p.parseNode("TriggerStruct", triggerSpec, nil)
	if err != nil {
		t.Fatalf("parse trigger raw: %v", err)
	}
	conditions := node.list("condition_data")
	if len(conditions) != 4 {
		t.Fatalf("conditions = %d, want 4", len(conditions))
	}
	checkIntField(t, conditions[0], "condition_type", 5)
	checkIntField(t, conditions[0], "object_list", unitConst)
	checkIntField(t, conditions[1], "condition_type", 8)
	checkIntField(t, conditions[1], "attribute", resource)
	checkIntField(t, conditions[2], "condition_type", 15)
	checkIntField(t, conditions[2], "unit_object", unitObject)
	checkIntField(t, conditions[3], "condition_type", 22)
	checkIntField(t, conditions[3], "variable", variable)
	checkIntField(t, conditions[3], "comparison", comparison)
	effects := node.list("effect_data")
	if len(effects) != 11 {
		t.Fatalf("effects = %d, want 11", len(effects))
	}
	checkIntField(t, effects[0], "effect_type", 2)
	checkIntField(t, effects[0], "technology", tech)
	checkIntField(t, effects[1], "effect_type", 41)
	checkIntField(t, effects[1], "visibility_state", vis)
	checkIntField(t, effects[2], "effect_type", 51)
	checkIntField(t, effects[2], "object_attributes", attr)
	checkIntField(t, effects[3], "effect_type", 52)
	checkIntField(t, effects[3], "tribute_list", resource)
	checkIntField(t, effects[4], "effect_type", 55)
	if got, _ := effects[4].stringValue("message"); got != "main();" {
		t.Fatalf("effects[4].message = %q, want main();", got)
	}
	checkIntField(t, effects[5], "effect_type", 18)
	checkIntField(t, effects[5], "target_player", targetPlayer)
	checkIntField(t, effects[6], "effect_type", 26)
	checkIntField(t, effects[6], "object_list_unit_id", unitConst)
	if got, _ := effects[6].stringValue("message"); got != "Renamed" {
		t.Fatalf("effects[6].message = %q, want Renamed", got)
	}
	checkIntField(t, effects[7], "effect_type", 44)
	checkIntField(t, effects[7], "object_list_unit_id", unitConst)
	checkIntField(t, effects[7], "string_id", stringID)
	if got, _ := effects[7].stringValue("message"); got != "Described" {
		t.Fatalf("effects[7].message = %q, want Described", got)
	}
	checkIntField(t, effects[8], "effect_type", 27)
	checkIntField(t, effects[8], "quantity", quantity)
	checkIntField(t, effects[9], "effect_type", 35)
	checkIntField(t, effects[9], "location_x", x)
	checkIntField(t, effects[10], "effect_type", 15)
	checkIntField(t, effects[10], "number_of_units_selected", 1)
	summary := summarizeEffect(effects[10], EffectsOptions{})
	if len(summary.SelectedObjectIDs) != 1 || summary.SelectedObjectIDs[0] != unitObject {
		t.Fatalf("selected_object_ids summary = %#v, want [%d]", summary.SelectedObjectIDs, unitObject)
	}
}

func TestBuildExpandedAuthoringEffectsRaw(t *testing.T) {
	spec, err := LoadCurrentDESpec()
	if err != nil {
		t.Fatalf("LoadCurrentDESpec: %v", err)
	}
	player := 1
	targetPlayer := 2
	diplomacy := 3
	mutualDiplomacy := 1
	unitConst := 83
	tech := 408
	x := 20
	y := 21
	scroll := 1
	enabled := 0
	quantity := 25
	operation := 1
	unitObject := 12345
	localTech := 1
	stringID := 2345
	raw, err := buildTriggerFromRecipe(spec, TriggerRecipe{
		Op:   "add_trigger",
		Name: "Expanded Authoring Effects",
		Effects: []EffectRecipe{
			{Op: "change_diplomacy", SourcePlayer: &player, TargetPlayer: &targetPlayer, Diplomacy: &diplomacy, MutualDiplomacy: &mutualDiplomacy},
			{Op: "play_sound", SourcePlayer: &player, SoundName: "ui\\default"},
			{Op: "change_view", SourcePlayer: &player, LocationX: &x, LocationY: &y, Scroll: &scroll},
			{Op: "place_foundation", SourcePlayer: &player, ObjectListUnitID: &unitConst, LocationX: &x, LocationY: &y},
			{Op: "build_object", SourcePlayer: &player, ObjectListUnitID: &unitConst, LocationX: &x, LocationY: &y},
			{Op: "damage_object", SourcePlayer: &player, Quantity: &quantity, Operation: &operation, SelectedObjectIDs: []int{unitObject}},
			{Op: "heal_object", SourcePlayer: &player, Quantity: &quantity, Operation: &operation, SelectedObjectIDs: []int{unitObject}},
			{Op: "change_object_attack", SourcePlayer: &player, Quantity: &quantity, Operation: &operation, SelectedObjectIDs: []int{unitObject}},
			{Op: "change_object_armor", SourcePlayer: &player, Quantity: &quantity, Operation: &operation, SelectedObjectIDs: []int{unitObject}},
			{Op: "change_object_range", SourcePlayer: &player, Quantity: &quantity, Operation: &operation, SelectedObjectIDs: []int{unitObject}},
			{Op: "change_object_speed", SourcePlayer: &player, Quantity: &quantity, Operation: &operation, SelectedObjectIDs: []int{unitObject}},
			{Op: "change_object_caption", SourcePlayer: &player, StringID: &stringID, Message: "Caption", SelectedObjectIDs: []int{unitObject}},
			{Op: "enable_disable_object", SourcePlayer: &player, Enabled: &enabled, SelectedObjectIDs: []int{unitObject}},
			{Op: "enable_disable_technology", SourcePlayer: &player, Technology: &tech, Enabled: &enabled},
			{Op: "train_unit", SourcePlayer: &player, ObjectListUnitID: &unitConst, SelectedObjectIDs: []int{unitObject}},
			{Op: "initiate_research", SourcePlayer: &player, Technology: &tech, SelectedObjectIDs: []int{unitObject}},
			{Op: "research_local_technology", SourcePlayer: &player, Technology: &tech, LocalTechnology: &localTech, SelectedObjectIDs: []int{unitObject}},
		},
	})
	if err != nil {
		t.Fatalf("buildTriggerFromRecipe: %v", err)
	}
	triggerSpec, _, err := triggerEffectSpecs(spec)
	if err != nil {
		t.Fatalf("triggerEffectSpecs: %v", err)
	}
	p := parser{data: raw, sections: map[string]*parsedSection{}}
	node, err := p.parseNode("TriggerStruct", triggerSpec, nil)
	if err != nil {
		t.Fatalf("parse trigger raw: %v", err)
	}
	effects := node.list("effect_data")
	wantTypes := []int{1, 4, 16, 25, 108, 24, 34, 28, 31, 32, 33, 88, 38, 39, 75, 76, 103}
	if len(effects) != len(wantTypes) {
		t.Fatalf("effects = %d, want %d", len(effects), len(wantTypes))
	}
	for i, want := range wantTypes {
		checkIntField(t, effects[i], "effect_type", want)
	}
	checkIntField(t, effects[0], "diplomacy", diplomacy)
	checkIntField(t, effects[0], "mutual_diplomacy", mutualDiplomacy)
	if got, _ := effects[1].stringValue("sound_name"); got != "ui\\default" {
		t.Fatalf("sound_name = %q, want ui\\default", got)
	}
	checkIntField(t, effects[2], "scroll", scroll)
	checkIntField(t, effects[3], "object_list_unit_id", unitConst)
	checkIntField(t, effects[4], "object_list_unit_id", unitConst)
	checkIntField(t, effects[5], "quantity", quantity)
	checkIntField(t, effects[11], "number_of_units_selected", 1)
	checkIntField(t, effects[11], "string_id", stringID)
	if got, _ := effects[11].stringValue("message"); got != "Caption" {
		t.Fatalf("caption message = %q, want Caption", got)
	}
	checkIntField(t, effects[12], "enabled", enabled)
	checkIntField(t, effects[13], "technology", tech)
	checkIntField(t, effects[13], "enabled", enabled)
	checkIntField(t, effects[14], "object_list_unit_id", unitConst)
	checkIntField(t, effects[15], "technology", tech)
	checkIntField(t, effects[16], "local_technology", localTech)
}

func TestBuildShopTechAndControlEffectsRaw(t *testing.T) {
	spec, err := LoadCurrentDESpec()
	if err != nil {
		t.Fatalf("LoadCurrentDESpec: %v", err)
	}
	player := 1
	unitConst := 83
	buildingConst := 12
	tech := 408
	button := 3
	hotkey := 16121
	trainTime := 7
	enabled := 0
	color := 4
	unitObject := 12345
	food := 0
	foodCost := 11
	gold := 3
	goldCost := 22
	icon := 77
	raw, err := buildTriggerFromRecipe(spec, TriggerRecipe{
		Op:   "add_trigger",
		Name: "Shop Tech And Control Effects",
		Effects: []EffectRecipe{
			{Op: "change_train_location", SourcePlayer: &player, ObjectListUnitID: &unitConst, ObjectListUnitID2: &buildingConst, ButtonLocation: &button, Hotkey: &hotkey, TrainTime: &trainTime},
			{Op: "add_train_location", SourcePlayer: &player, ObjectListUnitID: &unitConst, ObjectListUnitID2: &buildingConst, ButtonLocation: &button},
			{Op: "change_technology_location", SourcePlayer: &player, Technology: &tech, ObjectListUnitID: &buildingConst, ButtonLocation: &button},
			{Op: "change_technology_cost", SourcePlayer: &player, Technology: &tech, Resource1: &food, Resource1Quantity: &foodCost, Resource2: &gold, Resource2Quantity: &goldCost},
			{Op: "change_technology_research_time", SourcePlayer: &player, Technology: &tech, TrainTime: &trainTime},
			{Op: "change_technology_name", SourcePlayer: &player, Technology: &tech, Message: "Tech Name"},
			{Op: "change_technology_description", SourcePlayer: &player, Technology: &tech, Message: "Tech Description"},
			{Op: "change_technology_icon", SourcePlayer: &player, Technology: &tech, Quantity: &icon},
			{Op: "change_technology_hotkey", SourcePlayer: &player, Technology: &tech, Hotkey: &hotkey},
			{Op: "change_player_name", SourcePlayer: &player, Message: "Player Name"},
			{Op: "change_civilization_name", SourcePlayer: &player, Message: "Civilization Name"},
			{Op: "change_player_color", SourcePlayer: &player, PlayerColor: &color},
			{Op: "freeze_unit", SourcePlayer: &player, SelectedObjectIDs: []int{unitObject}},
			{Op: "stop_unit", SourcePlayer: &player, SelectedObjectIDs: []int{unitObject}},
			{Op: "disable_unit_targeting", SourcePlayer: &player, SelectedObjectIDs: []int{unitObject}},
			{Op: "enable_unit_targeting", SourcePlayer: &player, SelectedObjectIDs: []int{unitObject}},
			{Op: "disable_object_selection", SourcePlayer: &player, SelectedObjectIDs: []int{unitObject}},
			{Op: "enable_object_selection", SourcePlayer: &player, SelectedObjectIDs: []int{unitObject}},
			{Op: "enable_object_deletion", SourcePlayer: &player, SelectedObjectIDs: []int{unitObject}},
			{Op: "disable_object_deletion", SourcePlayer: &player, SelectedObjectIDs: []int{unitObject}},
			{Op: "disable_unit_attackable", SourcePlayer: &player, SelectedObjectIDs: []int{unitObject}},
			{Op: "enable_unit_attackable", SourcePlayer: &player, SelectedObjectIDs: []int{unitObject}},
			{Op: "enable_disable_technology", SourcePlayer: &player, Technology: &tech, Enabled: &enabled},
		},
	})
	if err != nil {
		t.Fatalf("buildTriggerFromRecipe: %v", err)
	}
	triggerSpec, _, err := triggerEffectSpecs(spec)
	if err != nil {
		t.Fatalf("triggerEffectSpecs: %v", err)
	}
	p := parser{data: raw, sections: map[string]*parsedSection{}}
	node, err := p.parseNode("TriggerStruct", triggerSpec, nil)
	if err != nil {
		t.Fatalf("parse trigger raw: %v", err)
	}
	effects := node.list("effect_data")
	wantTypes := []int{46, 102, 47, 63, 64, 65, 66, 84, 85, 45, 48, 89, 22, 29, 61, 62, 70, 71, 73, 74, 98, 99, 39}
	if len(effects) != len(wantTypes) {
		t.Fatalf("effects = %d, want %d", len(effects), len(wantTypes))
	}
	for i, want := range wantTypes {
		checkIntField(t, effects[i], "effect_type", want)
	}
	checkIntField(t, effects[0], "object_list_unit_id", unitConst)
	checkIntField(t, effects[0], "object_list_unit_id_2", buildingConst)
	checkIntField(t, effects[0], "button_location", button)
	checkIntField(t, effects[0], "hotkey", hotkey)
	checkIntField(t, effects[0], "train_time", trainTime)
	checkIntField(t, effects[2], "technology", tech)
	checkIntField(t, effects[2], "button_location", button)
	checkIntField(t, effects[3], "resource_1", food)
	checkIntField(t, effects[3], "resource_1_quantity", foodCost)
	checkIntField(t, effects[3], "resource_2", gold)
	checkIntField(t, effects[3], "resource_2_quantity", goldCost)
	checkIntField(t, effects[4], "quantity", trainTime)
	locationSummary := summarizeEffect(effects[2], EffectsOptions{})
	if locationSummary.SourcePlayer != player || locationSummary.Technology != tech || locationSummary.UnitConst != buildingConst || summaryIntField(t, locationSummary.KnownFields, "button_location") != button {
		t.Fatalf("change_technology_location summary = %+v", locationSummary)
	}
	costSummary := summarizeEffect(effects[3], EffectsOptions{})
	if costSummary.SourcePlayer != player || costSummary.Technology != tech ||
		summaryIntField(t, costSummary.KnownFields, "resource_1") != food ||
		summaryIntField(t, costSummary.KnownFields, "resource_1_quantity") != foodCost ||
		summaryIntField(t, costSummary.KnownFields, "resource_2") != gold ||
		summaryIntField(t, costSummary.KnownFields, "resource_2_quantity") != goldCost {
		t.Fatalf("change_technology_cost summary = %+v", costSummary)
	}
	researchTimeSummary := summarizeEffect(effects[4], EffectsOptions{})
	if researchTimeSummary.Quantity != trainTime || summaryIntField(t, researchTimeSummary.KnownFields, "research_time") != trainTime {
		t.Fatalf("change_technology_research_time summary = %+v", researchTimeSummary)
	}
	if got, _ := effects[5].stringValue("message"); got != "Tech Name" {
		t.Fatalf("tech name message = %q, want Tech Name", got)
	}
	checkIntField(t, effects[7], "quantity", icon)
	iconSummary := summarizeEffect(effects[7], EffectsOptions{})
	if iconSummary.Quantity != icon || summaryIntField(t, iconSummary.KnownFields, "icon") != icon {
		t.Fatalf("change_technology_icon summary = %+v", iconSummary)
	}
	checkIntField(t, effects[8], "hotkey", hotkey)
	if got, _ := effects[9].stringValue("message"); got != "Player Name" {
		t.Fatalf("player name message = %q, want Player Name", got)
	}
	checkIntField(t, effects[11], "player_color", color)
	checkIntField(t, effects[12], "number_of_units_selected", 1)
	checkIntField(t, effects[22], "enabled", enabled)
	enableTechSummary := summarizeEffect(effects[22], EffectsOptions{})
	if enableTechSummary.SourcePlayer != player || enableTechSummary.Technology != tech || summaryIntField(t, enableTechSummary.KnownFields, "enabled") != enabled {
		t.Fatalf("enable_disable_technology summary = %+v", enableTechSummary)
	}
}

func TestBuildTriggerObjectiveFieldsRaw(t *testing.T) {
	spec, err := LoadCurrentDESpec()
	if err != nil {
		t.Fatalf("LoadCurrentDESpec: %v", err)
	}
	enabled := true
	displayAsObjective := true
	displayOnScreen := true
	makeHeader := true
	muteObjectives := true
	executeOnLoad := true
	order := 17
	descID := -1
	shortDescID := -1
	raw, err := buildTriggerFromRecipe(spec, TriggerRecipe{
		Op:                        "add_trigger",
		Name:                      "Objective HUD",
		Description:               "Long objective description",
		ShortDescription:          "Coins <Variable 332>",
		Enabled:                   &enabled,
		DisplayAsObjective:        &displayAsObjective,
		DisplayOnScreen:           &displayOnScreen,
		MakeHeader:                &makeHeader,
		MuteObjectives:            &muteObjectives,
		ExecuteOnLoad:             &executeOnLoad,
		ObjectiveDescriptionOrder: &order,
		DescriptionStringID:       &descID,
		ShortDescriptionStringID:  &shortDescID,
	})
	if err != nil {
		t.Fatalf("buildTriggerFromRecipe: %v", err)
	}
	triggerSpec, _, err := triggerEffectSpecs(spec)
	if err != nil {
		t.Fatalf("triggerEffectSpecs: %v", err)
	}
	p := parser{data: raw, sections: map[string]*parsedSection{}}
	node, err := p.parseNode("TriggerStruct", triggerSpec, nil)
	if err != nil {
		t.Fatalf("parse trigger raw: %v", err)
	}
	checkIntField(t, node, "enabled", 1)
	checkIntField(t, node, "execute_on_load", 1)
	checkIntField(t, node, "display_as_objective", 1)
	checkIntField(t, node, "display_on_screen", 1)
	checkIntField(t, node, "make_header", 1)
	checkIntField(t, node, "mute_objectives", 1)
	checkIntField(t, node, "objective_description_order", order)
	checkIntField(t, node, "description_string_table_id", descID)
	checkIntField(t, node, "short_description_string_table_id", shortDescID)
	if got, _ := node.stringValue("trigger_description"); got != "Long objective description" {
		t.Fatalf("trigger_description = %q", got)
	}
	if got, _ := node.stringValue("short_description"); got != "Coins <Variable 332>" {
		t.Fatalf("short_description = %q", got)
	}
}

func TestPatchRecipeScenarioSettingsAndXS(t *testing.T) {
	input := testfixtures.Path(t, "save-analysis/diagnostics/Replay_Transparency_Diagnostic_v2b_flagged.aoe2scenario")
	if _, err := os.Stat(input); err != nil {
		t.Skipf("diagnostic fixture unavailable: %v", err)
	}
	out := filepath.Join(t.TempDir(), "xs_probe.aoe2scenario")
	playerCount := 4
	timestamp := 1788336000
	active := true
	human := false
	food := 888888
	recipe := Recipe{
		Scenario: &ScenarioRecipe{PlayerCount: &playerCount, TimestampOfLastSave: &timestamp},
		XS:       &XSRecipe{Name: "TestProbe", Content: "rule TestProbe active minInterval 5 { xsChatData(\"SDSTELEM\"); }\n"},
		Players: []PlayerRecipe{
			{Player: 2, Active: &active, Human: &human},
		},
		Diplomacy: []DiplomacyRecipe{
			{From: 1, To: 2, Stance: 3},
		},
		Resources: []ResourceRecipe{
			{Player: 1, Food: &food},
		},
	}
	if _, err := PatchRecipeFile(input, out, recipe); err != nil {
		t.Fatalf("PatchRecipeFile: %v", err)
	}
	scen, err := Open(out)
	if err != nil {
		t.Fatalf("Open patched: %v", err)
	}
	mapSection := scen.root.section("Map")
	if got, _ := mapSection.stringValue("script_name"); got != "TestProbe.xs" {
		t.Fatalf("script_name = %q, want TestProbe.xs", got)
	}
	files := scen.root.section("Files")
	if got, _ := files.stringValue("script_file_path"); got != "TestProbe.xs" {
		t.Fatalf("script_file_path = %q, want TestProbe.xs", got)
	}
	if got, _ := files.stringValue("script_file_content"); got == "" {
		t.Fatal("script_file_content is empty")
	}
	checkIntField(t, scen.headerRoot, "player_count", playerCount)
	checkIntField(t, scen.headerRoot, "timestamp_of_last_save", timestamp)
	players := scen.root.section("DataHeader").list("player_data_1")
	checkIntField(t, players[2], "active", 1)
	checkIntField(t, players[2], "human", 0)
	resources := scen.root.section("PlayerDataTwo").list("resources")
	checkIntField(t, resources[1], "food", food)
	rows := scen.root.section("Diplomacy").list("per_player_diplomacy")
	stances := rows[1].uint32List("stance_with_each_player")
	if stances[2] != 3 {
		t.Fatalf("P1->P2 stance = %d, want 3", stances[2])
	}
}

func TestPatchRecipeFileRefreshesTimestampWhenRecipeDoesNotSpecifyOne(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "old.aoe2scenario")
	if _, err := WriteBlankScenarioFile(input, BlankOptions{PlayerCount: 2, HumanSlots: 1, Timestamp: 1, DummyStarters: true}); err != nil {
		t.Fatalf("WriteBlankScenarioFile: %v", err)
	}
	out := filepath.Join(dir, "patched.aoe2scenario")
	before := int(time.Now().Unix()) - 1
	playerCount := 3
	report, err := PatchRecipeFile(input, out, Recipe{Scenario: &ScenarioRecipe{PlayerCount: &playerCount}})
	if err != nil {
		t.Fatalf("PatchRecipeFile: %v", err)
	}
	if report.TimestampOfLastSave < before {
		t.Fatalf("report timestamp = %d, want >= %d", report.TimestampOfLastSave, before)
	}
	scen, err := Open(out)
	if err != nil {
		t.Fatalf("Open patched: %v", err)
	}
	got, ok := scen.headerRoot.intValue("timestamp_of_last_save")
	if !ok {
		t.Fatal("timestamp_of_last_save missing")
	}
	if got != report.TimestampOfLastSave {
		t.Fatalf("written timestamp = %d, report = %d", got, report.TimestampOfLastSave)
	}
	if got < before {
		t.Fatalf("written timestamp = %d, want >= %d", got, before)
	}
}

func TestPatchRecipeXSCarrierMode(t *testing.T) {
	input := testfixtures.Path(t, "save-analysis/diagnostics/Replay_Transparency_Diagnostic_v2b_flagged.aoe2scenario")
	if _, err := os.Stat(input); err != nil {
		t.Skipf("diagnostic fixture unavailable: %v", err)
	}
	out := filepath.Join(t.TempDir(), "xs_carrier.aoe2scenario")
	enabled := true
	recipe := Recipe{
		XS: &XSRecipe{
			Mode:                "carrier",
			CarrierTitle:        "XS string",
			CarrierTriggerIndex: intPtr(0),
			Content:             "const int PROBE = 22;\nvoid BootProbe() { xsChatData(\"A2K\"); }\n",
		},
		Triggers: []TriggerRecipe{{
			Op:      "add_trigger",
			Name:    "Boot Probe",
			Enabled: &enabled,
			Effects: []EffectRecipe{{
				Op:      "script_call",
				Message: "BootProbe();",
			}},
		}},
	}
	if _, err := PatchRecipeFile(input, out, recipe); err != nil {
		t.Fatalf("PatchRecipeFile: %v", err)
	}
	scen, err := Open(out)
	if err != nil {
		t.Fatalf("Open patched: %v", err)
	}
	mapSection := scen.root.section("Map")
	if got, _ := mapSection.stringValue("script_name"); got != "" {
		t.Fatalf("script_name = %q, want empty carrier mode", got)
	}
	assertZeroLengthStringRaw(t, mapSection, "script_name", 2)
	files := scen.root.section("Files")
	if got, _ := files.stringValue("script_file_path"); got != "" {
		t.Fatalf("script_file_path = %q, want empty carrier mode", got)
	}
	assertZeroLengthStringRaw(t, files, "script_file_path", 2)
	if got, _ := files.stringValue("script_file_content"); got != "" {
		t.Fatalf("script_file_content = %q, want empty carrier mode", got)
	}
	assertZeroLengthStringRaw(t, files, "script_file_content", 4)
	report, err := scen.XSCensus(XSCensusOptions{})
	if err != nil {
		t.Fatalf("XSCensus: %v", err)
	}
	if report.Counts.EmbeddedCarriers != 1 || report.Counts.ScriptCalls != 1 || report.Counts.Functions != 1 {
		t.Fatalf("counts = %+v", report.Counts)
	}
	if len(report.EmbeddedCarriers) != 1 || report.EmbeddedCarriers[0].Title != "XS string" {
		t.Fatalf("carriers = %+v", report.EmbeddedCarriers)
	}
	if report.EmbeddedCarriers[0].TriggerIndex != 0 {
		t.Fatalf("carrier trigger index = %d, want 0", report.EmbeddedCarriers[0].TriggerIndex)
	}
	if report.EmbeddedCarriers[0].Content != "const int PROBE = 22;\nvoid BootProbe() { xsChatData(\"A2K\"); }\n" {
		t.Fatalf("carrier content = %q", report.EmbeddedCarriers[0].Content)
	}
	if len(report.ScriptCalls) != 1 || report.ScriptCalls[0].Message != "BootProbe();" {
		t.Fatalf("script calls = %+v", report.ScriptCalls)
	}
}

func TestPatchRecipeXSInlineRuntimeInsertsEnabledTriggerZero(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "blank.aoe2scenario")
	if _, err := WriteBlankScenarioFile(input, BlankOptions{
		PlayerCount:   2,
		HumanSlots:    1,
		MapWidth:      80,
		MapHeight:     80,
		DummyStarters: true,
		DummyUnit:     598,
	}); err != nil {
		t.Fatalf("WriteBlankScenarioFile: %v", err)
	}
	out := filepath.Join(dir, "xs_inline.aoe2scenario")
	enabled := true
	targetZero := 0
	source := "const int PROBE = 22;\nvoid BootProbe() { xsChatData(\"A2K\"); }\n"
	recipe := Recipe{
		XS: &XSRecipe{
			Mode:               "inline_runtime",
			Name:               "Entry.xs",
			CarrierTriggerName: "XS Runtime Source",
			Content:            source,
		},
		Triggers: []TriggerRecipe{
			{Op: "clear_triggers"},
			{
				Op:      "add_trigger",
				Name:    "Boot Probe",
				Enabled: &enabled,
				Effects: []EffectRecipe{{
					Op:      "script_call",
					Message: "BootProbe();",
				}},
			},
			{
				Op:      "add_trigger",
				Name:    "Activate Boot Probe",
				Enabled: &enabled,
				Effects: []EffectRecipe{{
					Op:        "activate_trigger",
					TriggerID: &targetZero,
				}},
			},
		},
	}
	if _, err := PatchRecipeFile(input, out, recipe); err != nil {
		t.Fatalf("PatchRecipeFile: %v", err)
	}
	scen, err := Open(out)
	if err != nil {
		t.Fatalf("Open patched: %v", err)
	}
	mapSection := scen.root.section("Map")
	if got, _ := mapSection.stringValue("script_name"); got != "" {
		t.Fatalf("script_name = %q, want empty inline runtime mode", got)
	}
	assertZeroLengthStringRaw(t, mapSection, "script_name", 2)
	files := scen.root.section("Files")
	if got, _ := files.stringValue("script_file_path"); got != "" {
		t.Fatalf("script_file_path = %q, want empty inline runtime mode", got)
	}
	assertZeroLengthStringRaw(t, files, "script_file_path", 2)
	if got, _ := files.stringValue("script_file_content"); got != "" {
		t.Fatalf("script_file_content = %q, want empty inline runtime mode", got)
	}
	assertZeroLengthStringRaw(t, files, "script_file_content", 4)
	report, err := scen.XSCensus(XSCensusOptions{})
	if err != nil {
		t.Fatalf("XSCensus: %v", err)
	}
	if report.Counts.ScriptContentAttachments != 0 || report.Counts.EmbeddedCarriers != 1 || report.Counts.ScriptCalls != 1 || report.Counts.Functions != 1 {
		t.Fatalf("counts = %+v", report.Counts)
	}
	carrier := report.EmbeddedCarriers[0]
	if carrier.TriggerIndex != 0 || carrier.TriggerName != "XS Runtime Source" || carrier.Title != "XS string" || carrier.Content != source {
		t.Fatalf("carrier = %+v", carrier)
	}
	if scen.Triggers.Triggers[0].Enabled != 1 {
		t.Fatalf("carrier enabled = %d, want 1", scen.Triggers.Triggers[0].Enabled)
	}
	if report.ScriptCalls[0].TriggerIndex != 1 || report.ScriptCalls[0].Message != "BootProbe();" {
		t.Fatalf("script calls = %+v", report.ScriptCalls)
	}
	triggerNodes := scen.root.section("Triggers").list("trigger_data")
	if len(triggerNodes) != 3 {
		t.Fatalf("trigger count = %d, want 3", len(triggerNodes))
	}
	activateEffects := triggerNodes[2].list("effect_data")
	if got, _ := activateEffects[0].intValue("trigger_id"); got != 1 {
		t.Fatalf("shifted activate trigger_id = %d, want 1", got)
	}
}

func TestPatchRecipeXSAttachmentAndCarrierSurvivesClearTriggers(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "blank.aoe2scenario")
	if _, err := WriteBlankScenarioFile(input, BlankOptions{
		PlayerCount:   2,
		HumanSlots:    1,
		MapWidth:      80,
		MapHeight:     80,
		DummyStarters: true,
		DummyUnit:     598,
	}); err != nil {
		t.Fatalf("WriteBlankScenarioFile: %v", err)
	}
	out := filepath.Join(dir, "xs_both.aoe2scenario")
	enabled := true
	source := "const int PROBE = 22;\nvoid BootProbe() { xsChatData(\"A2K\"); }\n"
	recipe := Recipe{
		XS: &XSRecipe{
			Mode:               "attachment_and_carrier",
			Name:               "Entry.xs",
			CarrierTitle:       "Entry.xs",
			CarrierTriggerName: "XS Source Carrier",
			Content:            source,
		},
		Triggers: []TriggerRecipe{
			{Op: "clear_triggers"},
			{
				Op:      "add_trigger",
				Name:    "Boot Probe",
				Enabled: &enabled,
				Effects: []EffectRecipe{{
					Op:      "script_call",
					Message: "BootProbe();",
				}},
			},
		},
	}
	if _, err := PatchRecipeFile(input, out, recipe); err != nil {
		t.Fatalf("PatchRecipeFile: %v", err)
	}
	scen, err := Open(out)
	if err != nil {
		t.Fatalf("Open patched: %v", err)
	}
	report, err := scen.XSCensus(XSCensusOptions{})
	if err != nil {
		t.Fatalf("XSCensus: %v", err)
	}
	if !report.ScriptContentEmbedded || report.Counts.ScriptContentAttachments != 1 {
		t.Fatalf("script content embedded = %t counts=%+v", report.ScriptContentEmbedded, report.Counts)
	}
	if len(report.EmbeddedCarriers) != 1 {
		t.Fatalf("embedded carriers = %+v", report.EmbeddedCarriers)
	}
	carrier := report.EmbeddedCarriers[0]
	if carrier.TriggerName != "XS Source Carrier" || carrier.Title != "Entry.xs" || carrier.Content != source {
		t.Fatalf("carrier = %+v", carrier)
	}
	if report.Counts.ScriptCalls != 1 || len(report.ScriptCalls) != 1 || report.ScriptCalls[0].Message != "BootProbe();" {
		t.Fatalf("script calls = %+v counts=%+v", report.ScriptCalls, report.Counts)
	}
	if scen.Triggers == nil || scen.Triggers.Count != 2 {
		t.Fatalf("trigger count = %#v, want boot + carrier", scen.Triggers)
	}
}

func TestXSCensusClassifiesParserStyleCarrier(t *testing.T) {
	carrier := xsCarrierMessage("XS string", "const int PROBE = 22;\nvoid BootProbe() {}\n")
	scen := scenarioWithXSFields("", "", "")
	scen.Version = "1.58"
	scen.Triggers = &TriggerInfo{Triggers: []TriggerSummary{{
		Index:   0,
		Name:    "XS SCRIPT",
		Enabled: 0,
		Looping: 0,
		EffectData: []EffectSummary{{
			EffectIndex: 0,
			Type:        55,
			TypeName:    "script_call",
			Text:        carrier,
		}},
	}, {
		Index: 1,
		Name:  "Boot",
		EffectData: []EffectSummary{{
			EffectIndex: 0,
			Type:        55,
			TypeName:    "script_call",
			Text:        "BootProbe();",
		}},
	}}}
	report, err := scen.XSCensus(XSCensusOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if report.Counts.EmbeddedCarriers != 1 || report.Counts.ScriptCalls != 1 || report.Counts.Functions != 1 {
		t.Fatalf("counts = %+v", report.Counts)
	}
	if report.EmbeddedCarriers[0].SourceBytes != len([]byte(carrier)) || report.EmbeddedCarriers[0].SourceSHA256 == "" {
		t.Fatalf("carrier summary = %+v", report.EmbeddedCarriers[0])
	}
	if report.EmbeddedCarriers[0].Content != "const int PROBE = 22;\nvoid BootProbe() {}\n" || report.EmbeddedCarriers[0].ContentSHA256 == "" {
		t.Fatalf("carrier content summary = %+v", report.EmbeddedCarriers[0])
	}
	if report.ScriptCalls[0].Message != "BootProbe();" {
		t.Fatalf("ordinary call not preserved: %+v", report.ScriptCalls)
	}
}

func TestTerrainFileAggregatesAndFilteredTiles(t *testing.T) {
	input := testfixtures.Path(t, "save-analysis/diagnostics/Replay_Transparency_Diagnostic_v2b_flagged.aoe2scenario")
	if _, err := os.Stat(input); err != nil {
		t.Skipf("diagnostic fixture unavailable: %v", err)
	}
	report, err := TerrainFile(input, TerrainOptions{})
	if err != nil {
		t.Fatalf("TerrainFile: %v", err)
	}
	if report.Width <= 0 || report.Height <= 0 || report.TileCount != report.Width*report.Height {
		t.Fatalf("bad terrain dimensions: %+v", report)
	}
	if len(report.Tiles) != 0 {
		t.Fatalf("unfiltered terrain report returned %d tiles without --tiles", len(report.Tiles))
	}
	if len(report.Aggregates) == 0 {
		t.Fatal("terrain aggregates are empty")
	}
	id := report.Aggregates[0].TerrainID
	filtered, err := TerrainFile(input, TerrainOptions{ID: &id})
	if err != nil {
		t.Fatalf("TerrainFile filtered: %v", err)
	}
	if filtered.Returned != filtered.Aggregates[0].Count || len(filtered.Tiles) != filtered.Returned {
		t.Fatalf("filtered returned/count mismatch: returned=%d tiles=%d aggregate=%d", filtered.Returned, len(filtered.Tiles), filtered.Aggregates[0].Count)
	}
}

func TestScenarioUnitsReadRotationFromFixture(t *testing.T) {
	input := testfixtures.Path(t, "save-analysis/diagnostics/Mandala Foundry.aoe2scenario")
	file, err := Open(input)
	if err != nil {
		t.Fatalf("Open Mandala: %v", err)
	}
	found := false
	for _, section := range file.Units.Sections {
		for _, unit := range section.Units {
			if unit.UnitConst == 1777 && unit.Rotation == 5 {
				found = true
			}
		}
	}
	if !found {
		t.Fatal("expected Mandala fixture to contain IndianStatues with rotation index 5")
	}
}

func TestPaletteUsageReportsUnusedAddressableIndices(t *testing.T) {
	file := &File{
		Units: &UnitInfo{
			Sections: []PlayerUnitsInfo{
				{
					Player: 0,
					Units: []UnitSummary{
						{UnitConst: 1777, Rotation: 0},
						{UnitConst: 1777, Rotation: 5},
						{UnitConst: 1777, Rotation: 11},
					},
				},
			},
		},
	}
	variantCount := 16
	report := file.PaletteUsage(datfile.PaletteReport{Rows: []datfile.PaletteRow{
		{
			UnitID:           1777,
			UnitName:         "INDIANSTATUES",
			StandingGraphic1: 12530,
			GraphicName:      "IndianStatues",
			AngleCount:       16,
			FrameCount:       1,
			VariantCount:     &variantCount,
			Classification:   "multi_variant",
			Confidence:       "engine_measured_one_case",
		},
	}})
	if len(report.Rows) != 1 {
		t.Fatalf("rows = %d, want 1", len(report.Rows))
	}
	row := report.Rows[0]
	if got, want := row.UsedIndices, []int{0, 5, 11}; !sameInts(got, want) {
		t.Fatalf("used indices = %v, want %v", got, want)
	}
	if len(row.UnusedIndices) != 13 || row.UnusedIndices[0] != 1 || row.UnusedIndices[len(row.UnusedIndices)-1] != 15 {
		t.Fatalf("unused indices = %v, want 13 missing including 1..15", row.UnusedIndices)
	}
	if report.Summary.MultiVariantTypeCount != 1 || report.Summary.ArtworksAvailable != 16 || report.Summary.ArtworksUsed != 3 {
		t.Fatalf("summary = %+v, want 1 type / 16 available / 3 used", report.Summary)
	}
	if len(report.Summary.BiggestUntapped) != 1 || report.Summary.BiggestUntapped[0].UnusedVariantCount != 13 {
		t.Fatalf("biggest untapped = %+v, want one row with 13 unused", report.Summary.BiggestUntapped)
	}
}

func TestPaletteUsageCarriesRotationEncoding(t *testing.T) {
	file := &File{
		Units: &UnitInfo{
			Sections: []PlayerUnitsInfo{
				{
					Player: 0,
					Units: []UnitSummary{
						{UnitConst: 2411, Rotation: 0},
						{UnitConst: 2411, Rotation: 3},
					},
				},
			},
		},
	}
	variantCount := 4
	report := file.PaletteUsage(datfile.PaletteReport{Rows: []datfile.PaletteRow{
		{
			UnitID:           2411,
			UnitName:         "Rock Limestone Hover",
			UnitType:         10,
			UnitClass:        14,
			UnitClassName:    "Eye Candy",
			StandingGraphic1: 15798,
			GraphicName:      "Rock Limestone Hover",
			AngleCount:       4,
			FrameCount:       1,
			SequenceType:     6,
			VariantCount:     &variantCount,
			Classification:   "multi_variant",
			Confidence:       "editor_fixture",
			RotationEncoding: "integer_artwork_index",
		},
	}})
	if len(report.Rows) != 1 {
		t.Fatalf("rows = %d, want 1", len(report.Rows))
	}
	row := report.Rows[0]
	if row.UnitType != 10 || row.UnitClass != 14 || row.UnitClassName != "Eye Candy" {
		t.Fatalf("unit type/class = %d/%d/%q, want 10/14/Eye Candy", row.UnitType, row.UnitClass, row.UnitClassName)
	}
	if row.RotationEncoding != "integer_artwork_index" {
		t.Fatalf("rotation_encoding = %q, want integer_artwork_index", row.RotationEncoding)
	}
	if got, want := row.UsedIndices, []int{0, 3}; !sameInts(got, want) {
		t.Fatalf("used indices = %v, want %v", got, want)
	}
}

func TestPaletteUsageAnimatedSharedPhaseIsNeutral(t *testing.T) {
	file := &File{
		Units: &UnitInfo{
			Sections: []PlayerUnitsInfo{
				{
					Player: 0,
					Units: []UnitSummary{
						{UnitConst: 455, Rotation: 7},
						{UnitConst: 455, Rotation: 7},
					},
				},
			},
		},
	}
	report := file.PaletteUsage(datfile.PaletteReport{Rows: []datfile.PaletteRow{
		{
			UnitID:         455,
			UnitName:       "FISH1",
			GraphicName:    "Fish Dorado",
			AngleCount:     15,
			FrameCount:     1,
			SequenceType:   6,
			Classification: "animated",
			Confidence:     "author_confirmed",
		},
	}})
	if len(report.Rows) != 1 {
		t.Fatalf("rows = %d, want 1", len(report.Rows))
	}
	row := report.Rows[0]
	if row.AvailableVariantCount != nil || len(row.UsedIndices) != 0 || len(row.UnusedIndices) != 0 {
		t.Fatalf("animated row should not be treated as variant artwork: %+v", row)
	}
	if !strings.Contains(row.Note, "phase distribution: 2 placements across 1 distinct raw rotation value") {
		t.Fatalf("note = %q, want neutral phase distribution", row.Note)
	}
	if strings.Contains(row.Note, "varying rotation") || strings.Contains(row.Note, "under-use") {
		t.Fatalf("note = %q, should not recommend variation or imply under-use", row.Note)
	}
}

func sameInts(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestPatchRecipeGlobalVictory(t *testing.T) {
	input := testfixtures.Path(t, "save-analysis/diagnostics/Replay_Transparency_Diagnostic_v2b_flagged.aoe2scenario")
	if _, err := os.Stat(input); err != nil {
		t.Skipf("diagnostic fixture unavailable: %v", err)
	}
	out := filepath.Join(t.TempDir(), "global_victory.aoe2scenario")
	conquest := 0
	allCustom := 1
	score := 0
	timer := 0
	recipe := Recipe{
		Victory: &VictoryRecipe{
			ConquestRequired:               &conquest,
			AllCustomConditionsRequired:    &allCustom,
			RequiredScoreForScoreVictory:   &score,
			TimeForTimedGameIn10thsOfAYear: &timer,
		},
	}
	if _, err := PatchRecipeFile(input, out, recipe); err != nil {
		t.Fatalf("PatchRecipeFile: %v", err)
	}
	scen, err := Open(out)
	if err != nil {
		t.Fatalf("Open patched: %v", err)
	}
	victory := scen.root.section("GlobalVictory")
	if victory == nil {
		t.Fatal("missing GlobalVictory")
	}
	checkIntField(t, victory, "conquest_required", conquest)
	checkIntField(t, victory, "all_custom_conditions_required", allCustom)
	checkIntField(t, victory, "required_score_for_score_victory", score)
	checkIntField(t, victory, "time_for_timed_game_in_10ths_of_a_year", timer)
}

func TestPatchRecipeClearTriggers(t *testing.T) {
	input := testfixtures.Path(t, "save-analysis/diagnostics/Replay_Transparency_Diagnostic_v2b_flagged.aoe2scenario")
	if _, err := os.Stat(input); err != nil {
		t.Skipf("diagnostic fixture unavailable: %v", err)
	}
	out := filepath.Join(t.TempDir(), "clear_triggers.aoe2scenario")
	timer := 3
	enabled := true
	recipe := Recipe{
		Triggers: []TriggerRecipe{
			{Op: "clear_triggers"},
			{
				Op:      "add_trigger",
				Name:    "Only New Trigger",
				Enabled: &enabled,
				Conditions: []ConditionRecipe{
					{Op: "timer", Timer: &timer},
				},
				Effects: []EffectRecipe{
					{Op: "display_instructions", Message: "new only"},
				},
			},
		},
	}
	if _, err := PatchRecipeFile(input, out, recipe); err != nil {
		t.Fatalf("PatchRecipeFile: %v", err)
	}
	scen, err := Open(out)
	if err != nil {
		t.Fatalf("Open patched: %v", err)
	}
	if scen.Triggers.Count != 1 {
		t.Fatalf("trigger count = %d, want 1", scen.Triggers.Count)
	}
	if len(scen.Triggers.DisplayOrder) != 1 || scen.Triggers.DisplayOrder[0] != 0 {
		t.Fatalf("display order = %#v, want [0]", scen.Triggers.DisplayOrder)
	}
	triggers := scen.root.section("Triggers")
	nodes := triggers.list("trigger_data")
	if len(nodes) != 1 {
		t.Fatalf("trigger nodes = %d, want 1", len(nodes))
	}
	if got, _ := nodes[0].stringValue("trigger_name"); got != "Only New Trigger" {
		t.Fatalf("trigger name = %q, want Only New Trigger", got)
	}
	if !scen.Triggers.InvariantOK {
		t.Fatalf("trigger invariant failed: %s", scen.Triggers.InvariantNote)
	}
}

func TestRemoveTriggerRewritesSurvivingReferences(t *testing.T) {
	input := testfixtures.Path(t, "save-analysis/diagnostics/Replay_Transparency_Diagnostic_v2b_flagged.aoe2scenario")
	if _, err := os.Stat(input); err != nil {
		t.Skipf("diagnostic fixture unavailable: %v", err)
	}
	out := filepath.Join(t.TempDir(), "remove_rewrite_refs.aoe2scenario")
	enabled := false
	targetTwo := 2
	recipe := Recipe{
		Triggers: []TriggerRecipe{
			{Op: "clear_triggers"},
			{
				Op:      "add_trigger",
				Name:    "Refers To T2",
				Enabled: &enabled,
				Effects: []EffectRecipe{
					{Op: "activate_trigger", TriggerID: &targetTwo},
				},
			},
			{Op: "add_trigger", Name: "Remove Me", Enabled: &enabled},
			{Op: "add_trigger", Name: "Survives", Enabled: &enabled},
			{Op: "remove_trigger", TargetName: "Remove Me"},
		},
	}
	if _, err := PatchRecipeFile(input, out, recipe); err != nil {
		t.Fatalf("PatchRecipeFile: %v", err)
	}
	scen, err := Open(out)
	if err != nil {
		t.Fatalf("Open patched: %v", err)
	}
	if scen.Triggers.Count != 2 {
		t.Fatalf("trigger count = %d, want 2", scen.Triggers.Count)
	}
	triggers := scen.root.section("Triggers").list("trigger_data")
	effects := triggers[0].list("effect_data")
	checkIntField(t, effects[0], "trigger_id", 1)
	name, _ := triggers[1].stringValue("trigger_name")
	if name != "Survives" {
		t.Fatalf("trigger 1 name = %q, want Survives", name)
	}
	if !scen.Triggers.InvariantOK {
		t.Fatalf("trigger invariant failed: %s", scen.Triggers.InvariantNote)
	}
}

func TestRemoveTriggersAllowsInternalRefsAndRewritesSurvivors(t *testing.T) {
	input := testfixtures.Path(t, "save-analysis/diagnostics/Replay_Transparency_Diagnostic_v2b_flagged.aoe2scenario")
	if _, err := os.Stat(input); err != nil {
		t.Skipf("diagnostic fixture unavailable: %v", err)
	}
	out := filepath.Join(t.TempDir(), "remove_batch_refs.aoe2scenario")
	enabled := false
	internalTarget := 2
	survivorTarget := 4
	recipe := Recipe{
		Triggers: []TriggerRecipe{
			{Op: "clear_triggers"},
			{
				Op:      "add_trigger",
				Name:    "Keeper",
				Enabled: &enabled,
				Effects: []EffectRecipe{
					{Op: "activate_trigger", TriggerID: &survivorTarget},
				},
			},
			{
				Op:      "add_trigger",
				Name:    "Batch Source",
				Enabled: &enabled,
				Effects: []EffectRecipe{
					{Op: "activate_trigger", TriggerID: &internalTarget},
				},
			},
			{Op: "add_trigger", Name: "Batch Target", Enabled: &enabled},
			{Op: "add_trigger", Name: "Batch Spare", Enabled: &enabled},
			{Op: "add_trigger", Name: "Survives", Enabled: &enabled},
			{Op: "remove_triggers", TargetIndexes: []int{3, 1, 2}},
		},
	}
	if _, err := PatchRecipeFile(input, out, recipe); err != nil {
		t.Fatalf("PatchRecipeFile: %v", err)
	}
	scen, err := Open(out)
	if err != nil {
		t.Fatalf("Open patched: %v", err)
	}
	if scen.Triggers.Count != 2 {
		t.Fatalf("trigger count = %d, want 2", scen.Triggers.Count)
	}
	triggers := scen.root.section("Triggers").list("trigger_data")
	effects := triggers[0].list("effect_data")
	checkIntField(t, effects[0], "trigger_id", 1)
	name, _ := triggers[1].stringValue("trigger_name")
	if name != "Survives" {
		t.Fatalf("trigger 1 name = %q, want Survives", name)
	}
	if !scen.Triggers.InvariantOK {
		t.Fatalf("trigger invariant failed: %s", scen.Triggers.InvariantNote)
	}
}

func TestRemoveTriggersRefusesExternalReferences(t *testing.T) {
	input := testfixtures.Path(t, "save-analysis/diagnostics/Replay_Transparency_Diagnostic_v2b_flagged.aoe2scenario")
	if _, err := os.Stat(input); err != nil {
		t.Skipf("diagnostic fixture unavailable: %v", err)
	}
	out := filepath.Join(t.TempDir(), "remove_batch_external_ref.aoe2scenario")
	enabled := false
	targetOne := 1
	recipe := Recipe{
		Triggers: []TriggerRecipe{
			{Op: "clear_triggers"},
			{
				Op:      "add_trigger",
				Name:    "External Ref",
				Enabled: &enabled,
				Effects: []EffectRecipe{
					{Op: "activate_trigger", TriggerID: &targetOne},
				},
			},
			{Op: "add_trigger", Name: "Batch Target", Enabled: &enabled},
			{Op: "remove_triggers", TargetIndexes: []int{1}},
		},
	}
	err := func() error {
		_, err := PatchRecipeFile(input, out, recipe)
		return err
	}()
	if err == nil || !strings.Contains(err.Error(), "references removed trigger 1") {
		t.Fatalf("remove_triggers external ref error = %v, want refusal", err)
	}
}

func TestRemoveTriggerRefusesReferencedTarget(t *testing.T) {
	input := testfixtures.Path(t, "save-analysis/diagnostics/Replay_Transparency_Diagnostic_v2b_flagged.aoe2scenario")
	if _, err := os.Stat(input); err != nil {
		t.Skipf("diagnostic fixture unavailable: %v", err)
	}
	out := filepath.Join(t.TempDir(), "remove_referenced_target.aoe2scenario")
	enabled := false
	targetOne := 1
	recipe := Recipe{
		Triggers: []TriggerRecipe{
			{Op: "clear_triggers"},
			{
				Op:      "add_trigger",
				Name:    "Refers To Removed",
				Enabled: &enabled,
				Effects: []EffectRecipe{
					{Op: "activate_trigger", TriggerID: &targetOne},
				},
			},
			{Op: "add_trigger", Name: "Referenced Target", Enabled: &enabled},
			{Op: "remove_trigger", TargetName: "Referenced Target"},
		},
	}
	err := func() error {
		_, err := PatchRecipeFile(input, out, recipe)
		return err
	}()
	if err == nil || !strings.Contains(err.Error(), "refuses to remove trigger 1") {
		t.Fatalf("remove referenced trigger error = %v, want refuse message", err)
	}
}

func TestTombstoneTriggerPreservesReferencedIndex(t *testing.T) {
	input := testfixtures.Path(t, "save-analysis/diagnostics/Replay_Transparency_Diagnostic_v2b_flagged.aoe2scenario")
	if _, err := os.Stat(input); err != nil {
		t.Skipf("diagnostic fixture unavailable: %v", err)
	}
	out := filepath.Join(t.TempDir(), "tombstone_referenced_target.aoe2scenario")
	enabled := true
	targetOne := 1
	timer := 5
	recipe := Recipe{
		Triggers: []TriggerRecipe{
			{Op: "clear_triggers"},
			{
				Op:      "add_trigger",
				Name:    "Activator",
				Enabled: &enabled,
				Effects: []EffectRecipe{
					{Op: "activate_trigger", TriggerID: &targetOne},
				},
			},
			{
				Op:      "add_trigger",
				Name:    "Referenced Behavior",
				Enabled: &enabled,
				Conditions: []ConditionRecipe{
					{Op: "timer", Timer: &timer},
				},
				Effects: []EffectRecipe{
					{Op: "send_chat", SourcePlayer: &targetOne, Message: "should be cleared"},
				},
			},
			{Op: "tombstone_trigger", TargetName: "Referenced Behavior"},
		},
	}
	if _, err := PatchRecipeFile(input, out, recipe); err != nil {
		t.Fatalf("PatchRecipeFile: %v", err)
	}
	scen, err := Open(out)
	if err != nil {
		t.Fatalf("Open patched: %v", err)
	}
	if scen.Triggers.Count != 2 {
		t.Fatalf("trigger count = %d, want 2", scen.Triggers.Count)
	}
	triggers := scen.root.section("Triggers").list("trigger_data")
	effects := triggers[0].list("effect_data")
	checkIntField(t, effects[0], "trigger_id", 1)
	checkIntField(t, triggers[1], "enabled", 0)
	checkIntField(t, triggers[1], "number_of_effects", 0)
	checkIntField(t, triggers[1], "number_of_conditions", 0)
	name, _ := triggers[1].stringValue("trigger_name")
	if !strings.Contains(name, "TOMBSTONED") {
		t.Fatalf("tombstone trigger name = %q, want marker", name)
	}
	if !scen.Triggers.InvariantOK {
		t.Fatalf("trigger invariant failed: %s", scen.Triggers.InvariantNote)
	}
}

func TestPatchRecipeEditTriggerRemoveChildren(t *testing.T) {
	input := testfixtures.Path(t, "save-analysis/diagnostics/Replay_Transparency_Diagnostic_v2b_flagged.aoe2scenario")
	if _, err := os.Stat(input); err != nil {
		t.Skipf("diagnostic fixture unavailable: %v", err)
	}
	out := filepath.Join(t.TempDir(), "edit_trigger_children.aoe2scenario")
	enabled := false
	timerA := 3
	timerB := 7
	triggerZero := 0
	removeEffects := []int{0}
	removeConditions := []int{1}
	recipe := Recipe{
		Triggers: []TriggerRecipe{
			{Op: "clear_triggers"},
			{
				Op:      "add_trigger",
				Name:    "Child Surgery",
				Enabled: &enabled,
				Conditions: []ConditionRecipe{
					{Op: "timer", Timer: &timerA},
					{Op: "timer", Timer: &timerB},
				},
				Effects: []EffectRecipe{
					{Op: "display_instructions", Message: "remove me"},
					{Op: "send_chat", Message: "keep me"},
				},
			},
			{
				Op:               "edit_trigger",
				TargetName:       "Child Surgery",
				RemoveEffects:    removeEffects,
				RemoveConditions: removeConditions,
				Effects: []EffectRecipe{
					{Op: "activate_trigger", TriggerID: &triggerZero},
				},
				Conditions: []ConditionRecipe{
					{Op: "timer", Timer: &timerB},
				},
			},
		},
	}
	if _, err := PatchRecipeFile(input, out, recipe); err != nil {
		t.Fatalf("PatchRecipeFile: %v", err)
	}
	scen, err := Open(out)
	if err != nil {
		t.Fatalf("Open patched: %v", err)
	}
	if !scen.Triggers.InvariantOK {
		t.Fatalf("trigger invariant failed: %s", scen.Triggers.InvariantNote)
	}
	trigger := scen.root.section("Triggers").list("trigger_data")[0]
	effects := trigger.list("effect_data")
	if len(effects) != 2 {
		t.Fatalf("effects = %d, want 2", len(effects))
	}
	checkIntField(t, effects[0], "effect_type", 3)
	checkIntField(t, effects[1], "effect_type", 8)
	if got := trigger.intList("effect_display_order_array"); len(got) != 2 || got[0] != 0 || got[1] != 1 {
		t.Fatalf("effect order = %#v, want [0 1]", got)
	}
	conditions := trigger.list("condition_data")
	if len(conditions) != 2 {
		t.Fatalf("conditions = %d, want 2", len(conditions))
	}
	checkIntField(t, conditions[0], "timer", timerA)
	checkIntField(t, conditions[1], "timer", timerB)
	if got := trigger.intList("condition_display_order_array"); len(got) != 2 || got[0] != 0 || got[1] != 1 {
		t.Fatalf("condition order = %#v, want [0 1]", got)
	}
}

func TestPatchRecipeCopyTriggerCanRenameAndEditClone(t *testing.T) {
	input := testfixtures.Path(t, "save-analysis/diagnostics/Replay_Transparency_Diagnostic_v2b_flagged.aoe2scenario")
	if _, err := os.Stat(input); err != nil {
		t.Skipf("diagnostic fixture unavailable: %v", err)
	}
	out := filepath.Join(t.TempDir(), "copy_trigger.aoe2scenario")
	enabled := false
	timerA := 3
	timerB := 7
	removeEffects := []int{0}
	newName := "Copied Child Surgery"
	recipe := Recipe{
		Triggers: []TriggerRecipe{
			{Op: "clear_triggers"},
			{
				Op:      "add_trigger",
				Name:    "Child Surgery",
				Enabled: &enabled,
				Conditions: []ConditionRecipe{
					{Op: "timer", Timer: &timerA},
				},
				Effects: []EffectRecipe{
					{Op: "display_instructions", Message: "remove me"},
					{Op: "send_chat", Message: "keep me"},
				},
			},
			{
				Op:            "copy_trigger",
				TargetName:    "Child Surgery",
				SetName:       &newName,
				RemoveEffects: removeEffects,
				Conditions: []ConditionRecipe{
					{Op: "timer", Timer: &timerB},
				},
			},
		},
	}
	if _, err := PatchRecipeFile(input, out, recipe); err != nil {
		t.Fatalf("PatchRecipeFile: %v", err)
	}
	scen, err := Open(out)
	if err != nil {
		t.Fatalf("Open patched: %v", err)
	}
	if !scen.Triggers.InvariantOK {
		t.Fatalf("trigger invariant failed: %s", scen.Triggers.InvariantNote)
	}
	triggers := scen.root.section("Triggers").list("trigger_data")
	if len(triggers) != 2 {
		t.Fatalf("triggers = %d, want 2", len(triggers))
	}
	clone := triggers[1]
	if got, _ := clone.stringValue("trigger_name"); got != newName {
		t.Fatalf("clone trigger_name = %q, want %q", got, newName)
	}
	checkIntField(t, clone, "enabled", 0)
	if got := len(clone.list("effect_data")); got != 1 {
		t.Fatalf("clone effects = %d, want 1", got)
	}
	checkIntField(t, clone.list("effect_data")[0], "effect_type", 3)
	if got := len(clone.list("condition_data")); got != 2 {
		t.Fatalf("clone conditions = %d, want 2", got)
	}
	checkIntField(t, clone.list("condition_data")[0], "timer", timerA)
	checkIntField(t, clone.list("condition_data")[1], "timer", timerB)
	if got := clone.intList("effect_display_order_array"); len(got) != 1 || got[0] != 0 {
		t.Fatalf("clone effect order = %#v, want [0]", got)
	}
	if got := clone.intList("condition_display_order_array"); len(got) != 2 || got[0] != 0 || got[1] != 1 {
		t.Fatalf("clone condition order = %#v, want [0 1]", got)
	}
}

func TestPatchRecipeEditTriggerClearChildren(t *testing.T) {
	input := testfixtures.Path(t, "save-analysis/diagnostics/Replay_Transparency_Diagnostic_v2b_flagged.aoe2scenario")
	if _, err := os.Stat(input); err != nil {
		t.Skipf("diagnostic fixture unavailable: %v", err)
	}
	out := filepath.Join(t.TempDir(), "clear_trigger_children.aoe2scenario")
	enabled := false
	clear := true
	timer := 3
	recipe := Recipe{
		Triggers: []TriggerRecipe{
			{Op: "clear_triggers"},
			{
				Op:      "add_trigger",
				Name:    "Clear Child Lists",
				Enabled: &enabled,
				Conditions: []ConditionRecipe{
					{Op: "timer", Timer: &timer},
				},
				Effects: []EffectRecipe{
					{Op: "display_instructions", Message: "clear me"},
				},
			},
			{
				Op:              "edit_trigger",
				TargetName:      "Clear Child Lists",
				ClearEffects:    &clear,
				ClearConditions: &clear,
			},
		},
	}
	if _, err := PatchRecipeFile(input, out, recipe); err != nil {
		t.Fatalf("PatchRecipeFile: %v", err)
	}
	scen, err := Open(out)
	if err != nil {
		t.Fatalf("Open patched: %v", err)
	}
	trigger := scen.root.section("Triggers").list("trigger_data")[0]
	checkIntField(t, trigger, "number_of_effects", 0)
	checkIntField(t, trigger, "number_of_conditions", 0)
	if got := trigger.intList("effect_display_order_array"); len(got) != 0 {
		t.Fatalf("effect order = %#v, want empty", got)
	}
	if got := trigger.intList("condition_display_order_array"); len(got) != 0 {
		t.Fatalf("condition order = %#v, want empty", got)
	}
	if !scen.Triggers.InvariantOK {
		t.Fatalf("trigger invariant failed: %s", scen.Triggers.InvariantNote)
	}
}

func TestPatchRecipeEditTriggerReplaceChildren(t *testing.T) {
	input := testfixtures.Path(t, "save-analysis/diagnostics/Replay_Transparency_Diagnostic_v2b_flagged.aoe2scenario")
	if _, err := os.Stat(input); err != nil {
		t.Skipf("diagnostic fixture unavailable: %v", err)
	}
	out := filepath.Join(t.TempDir(), "replace_trigger_children.aoe2scenario")
	enabled := false
	timer := 3
	newTimer := 11
	triggerZero := 0
	recipe := Recipe{
		Triggers: []TriggerRecipe{
			{Op: "clear_triggers"},
			{
				Op:      "add_trigger",
				Name:    "Replace Child Lists",
				Enabled: &enabled,
				Conditions: []ConditionRecipe{
					{Op: "timer", Timer: &timer},
				},
				Effects: []EffectRecipe{
					{Op: "display_instructions", Message: "replace me"},
				},
			},
			{
				Op:         "edit_trigger",
				TargetName: "Replace Child Lists",
				ReplaceEffects: []EffectRecipe{
					{Op: "send_chat", Message: "replacement chat"},
					{Op: "activate_trigger", TriggerID: &triggerZero},
				},
				ReplaceConditions: []ConditionRecipe{
					{Op: "timer", Timer: &newTimer},
				},
			},
		},
	}
	if _, err := PatchRecipeFile(input, out, recipe); err != nil {
		t.Fatalf("PatchRecipeFile: %v", err)
	}
	scen, err := Open(out)
	if err != nil {
		t.Fatalf("Open patched: %v", err)
	}
	trigger := scen.root.section("Triggers").list("trigger_data")[0]
	effects := trigger.list("effect_data")
	if len(effects) != 2 {
		t.Fatalf("effects = %d, want 2", len(effects))
	}
	checkIntField(t, effects[0], "effect_type", 3)
	checkIntField(t, effects[1], "effect_type", 8)
	conditions := trigger.list("condition_data")
	if len(conditions) != 1 {
		t.Fatalf("conditions = %d, want 1", len(conditions))
	}
	checkIntField(t, conditions[0], "timer", newTimer)
	if !scen.Triggers.InvariantOK {
		t.Fatalf("trigger invariant failed: %s", scen.Triggers.InvariantNote)
	}
}

func TestEditTriggerReplaceChildrenRejectsMixedModes(t *testing.T) {
	spec, err := LoadCurrentDESpec()
	if err != nil {
		t.Fatalf("LoadCurrentDESpec: %v", err)
	}
	enabled := false
	raw, err := buildTriggerFromRecipe(spec, TriggerRecipe{
		Op:      "add_trigger",
		Name:    "Mixed Modes",
		Enabled: &enabled,
		Effects: []EffectRecipe{
			{Op: "display_instructions", Message: "old"},
		},
	})
	if err != nil {
		t.Fatalf("buildTriggerFromRecipe: %v", err)
	}
	triggerSpec, _, err := triggerEffectSpecs(spec)
	if err != nil {
		t.Fatalf("triggerEffectSpecs: %v", err)
	}
	p := parser{data: raw, sections: map[string]*parsedSection{}}
	node, err := p.parseNode("TriggerStruct", triggerSpec, nil)
	if err != nil {
		t.Fatalf("parse trigger raw: %v", err)
	}
	if err := editTriggerChildLists(spec, node, TriggerRecipe{
		ReplaceEffects: []EffectRecipe{{Op: "send_chat", Message: "new"}},
		RemoveEffects:  []int{0},
	}); err == nil || !strings.Contains(err.Error(), "replace_effects cannot be combined") {
		t.Fatalf("mixed replace/remove error = %v, want replace_effects guard", err)
	}
}

func TestLintTriggerRuntimeHazardsFlagsLoopingDisplay(t *testing.T) {
	spec, err := LoadCurrentDESpec()
	if err != nil {
		t.Fatalf("LoadCurrentDESpec: %v", err)
	}
	enabled := true
	looping := true
	raw, err := buildTriggerFromRecipe(spec, TriggerRecipe{
		Op:      "add_trigger",
		Name:    "Looping Display",
		Enabled: &enabled,
		Looping: &looping,
		Effects: []EffectRecipe{
			{Op: "display_instructions", Message: "unsafe"},
		},
	})
	if err != nil {
		t.Fatalf("buildTriggerFromRecipe: %v", err)
	}
	triggerSpec, _, err := triggerEffectSpecs(spec)
	if err != nil {
		t.Fatalf("triggerEffectSpecs: %v", err)
	}
	p := parser{data: raw, sections: map[string]*parsedSection{}}
	node, err := p.parseNode("TriggerStruct", triggerSpec, nil)
	if err != nil {
		t.Fatalf("parse trigger raw: %v", err)
	}
	report := LintReport{}
	lintTriggerRuntimeHazards(&report, []*parsedNode{node}, LintOptions{})
	if len(report.Issues) != 1 {
		t.Fatalf("issues = %d, want 1: %+v", len(report.Issues), report.Issues)
	}
	if report.Issues[0].Severity != "error" || report.Issues[0].Code != "looping_display_effect" {
		t.Fatalf("issue = %+v, want looping_display_effect error", report.Issues[0])
	}
	if report.Issues[0].FactID != "scenario.looping_display_effect_spams" || report.Issues[0].FactTier != "engine_verified" {
		t.Fatalf("issue fact = %+v, want engine fact citation", report.Issues[0])
	}
}

func TestLintTriggerModifyAttributeRejectsZeroOperation(t *testing.T) {
	spec, err := LoadCurrentDESpec()
	if err != nil {
		t.Fatalf("LoadCurrentDESpec: %v", err)
	}
	enabled := true
	looping := false
	unitID, player, attr, quantity, operation := 12, 1, 3, 1, 1
	raw, err := buildTriggerFromRecipe(spec, TriggerRecipe{
		Op: "add_trigger", Name: "Invalid modify operation", Enabled: &enabled, Looping: &looping,
		Effects: []EffectRecipe{{Op: "modify_attribute", SourcePlayer: &player, ObjectListUnitID: &unitID, ObjectAttributes: &attr, Quantity: &quantity, Operation: &operation}},
	})
	if err != nil {
		t.Fatalf("buildTriggerFromRecipe: %v", err)
	}
	triggerSpec, _, err := triggerEffectSpecs(spec)
	if err != nil {
		t.Fatalf("triggerEffectSpecs: %v", err)
	}
	node, err := (&parser{data: raw, sections: map[string]*parsedSection{}}).parseNode("TriggerStruct", triggerSpec, nil)
	if err != nil {
		t.Fatalf("parse trigger raw: %v", err)
	}
	node.list("effect_data")[0].field("operation").Value = 0
	report := LintReport{}
	lintTriggerRuntimeHazards(&report, []*parsedNode{node}, LintOptions{})
	if !hasIssueCode(report.Issues, "modify_attribute_invalid_operation") {
		t.Fatalf("missing invalid operation issue: %+v", report.Issues)
	}
	if report.Issues[0].Severity != "error" {
		t.Fatalf("issue severity = %q, want error", report.Issues[0].Severity)
	}
}

func TestLintTriggerTextMarkupCitesEngineFacts(t *testing.T) {
	spec, err := LoadCurrentDESpec()
	if err != nil {
		t.Fatalf("LoadCurrentDESpec: %v", err)
	}
	enabled := true
	looping := false
	raw, err := buildTriggerFromRecipe(spec, TriggerRecipe{
		Op:      "add_trigger",
		Name:    "Bad Color Markup",
		Enabled: &enabled,
		Looping: &looping,
		Effects: []EffectRecipe{
			{Op: "display_instructions", Message: "<GREEN>first <RED>second"},
			{Op: "send_chat", Message: "<WHITE>bad"},
		},
	})
	if err != nil {
		t.Fatalf("buildTriggerFromRecipe: %v", err)
	}
	triggerSpec, _, err := triggerEffectSpecs(spec)
	if err != nil {
		t.Fatalf("triggerEffectSpecs: %v", err)
	}
	p := parser{data: raw, sections: map[string]*parsedSection{}}
	node, err := p.parseNode("TriggerStruct", triggerSpec, nil)
	if err != nil {
		t.Fatalf("parse trigger raw: %v", err)
	}
	report := LintReport{}
	lintTriggerRuntimeHazards(&report, []*parsedNode{node}, LintOptions{})
	if !hasIssueCode(report.Issues, "trigger_color_tag_not_leading_singleton") {
		t.Fatalf("missing multiple-color-tag issue: %+v", report.Issues)
	}
	if !hasIssueCode(report.Issues, "unsupported_white_color_tag") {
		t.Fatalf("missing white-tag issue: %+v", report.Issues)
	}
	for _, issue := range report.Issues {
		if issue.FactID == "" || issue.FactTier != "engine_verified" {
			t.Fatalf("issue missing engine fact citation: %+v", issue)
		}
	}
}

func TestLintTriggerRuntimeHazardsAllowsLoopingNonDisplay(t *testing.T) {
	spec, err := LoadCurrentDESpec()
	if err != nil {
		t.Fatalf("LoadCurrentDESpec: %v", err)
	}
	enabled := true
	looping := true
	unitID := 83
	x := 14
	y := 14
	raw, err := buildTriggerFromRecipe(spec, TriggerRecipe{
		Op:      "add_trigger",
		Name:    "Looping Create",
		Enabled: &enabled,
		Looping: &looping,
		Effects: []EffectRecipe{
			{Op: "create_object", ObjectListUnitID: &unitID, SourcePlayer: intPtr(1), LocationX: &x, LocationY: &y},
		},
	})
	if err != nil {
		t.Fatalf("buildTriggerFromRecipe: %v", err)
	}
	triggerSpec, _, err := triggerEffectSpecs(spec)
	if err != nil {
		t.Fatalf("triggerEffectSpecs: %v", err)
	}
	p := parser{data: raw, sections: map[string]*parsedSection{}}
	node, err := p.parseNode("TriggerStruct", triggerSpec, nil)
	if err != nil {
		t.Fatalf("parse trigger raw: %v", err)
	}
	report := LintReport{}
	lintTriggerRuntimeHazards(&report, []*parsedNode{node}, LintOptions{})
	if len(report.Issues) != 0 {
		t.Fatalf("issues = %+v, want none", report.Issues)
	}
}

func TestLintActiveComputerWithoutAuthoredUnits(t *testing.T) {
	input := testfixtures.Path(t, "save-analysis/diagnostics/Replay Transparency Diagnostic v14.1 Castle Kill Calibration.aoe2scenario")
	report, err := LintFile(input)
	if err != nil {
		t.Fatalf("LintFile: %v", err)
	}
	if !hasIssueCode(report.Issues, "active_computer_without_authored_units") {
		t.Fatalf("missing active computer starter-injection warning: %+v", report.Issues)
	}
	if !report.OK {
		t.Fatalf("report should remain OK for warning-only starter-injection lint: %+v", report.Issues)
	}
}

func TestLintLoadSafetyPlayerCountAndLockedCiv(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "base.aoe2scenario")
	if _, err := WriteBlankScenarioFile(input, BlankOptions{PlayerCount: 4, HumanSlots: 4, Timestamp: 1800000000}); err != nil {
		t.Fatalf("WriteBlankScenarioFile: %v", err)
	}
	output := filepath.Join(dir, "bad_players.aoe2scenario")
	playerCount := 3
	locked := true
	if _, err := PatchRecipeFile(input, output, Recipe{
		Scenario: &ScenarioRecipe{PlayerCount: &playerCount},
		Players:  []PlayerRecipe{{Player: 2, LockCivilization: &locked}},
	}); err != nil {
		t.Fatalf("PatchRecipeFile: %v", err)
	}
	report, err := LintFileWithOptions(output, LintOptions{LoadSafety: true})
	if err != nil {
		t.Fatalf("LintFileWithOptions: %v", err)
	}
	if !hasIssueCode(report.Issues, "load_safety_player_count_mismatch") {
		t.Fatalf("missing player-count mismatch issue: %+v", report.Issues)
	}
	if !hasIssueCode(report.Issues, "load_safety_locked_civ_launch_risk") {
		t.Fatalf("missing locked-civ launch warning: %+v", report.Issues)
	}
	if report.OK {
		t.Fatalf("load-safety error should make lint fail: %+v", report.Issues)
	}
}

func TestLintLoadSafetyRejectsLegacyEightPlayerPreviewLayout(t *testing.T) {
	path := filepath.Join(t.TempDir(), "legacy-eight.aoe2scenario")
	if _, err := WriteBlankScenarioFile(path, BlankOptions{PlayerCount: 8, HumanSlots: 8, Timestamp: 1800000000}); err != nil {
		t.Fatalf("WriteBlankScenarioFile: %v", err)
	}
	report, err := LintFileWithOptions(path, LintOptions{LoadSafety: true})
	if err != nil {
		t.Fatalf("LintFileWithOptions: %v", err)
	}
	if !hasIssueCode(report.Issues, "load_safety_legacy_eight_player_preview_layout") {
		t.Fatalf("missing legacy eight-player preview issue: %+v", report.Issues)
	}
	if report.OK {
		t.Fatalf("legacy eight-player preview layout should fail load-safety: %+v", report.Issues)
	}
}

func TestLintLoadSafetyTrainButtonCollisionAndOverflow(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "base.aoe2scenario")
	if _, err := WriteBlankScenarioFile(input, BlankOptions{PlayerCount: 1, HumanSlots: 1, Timestamp: 1800000000}); err != nil {
		t.Fatalf("WriteBlankScenarioFile: %v", err)
	}
	output := filepath.Join(dir, "bad_shop.aoe2scenario")
	player := 1
	button := 1
	overflowButton := 17
	building := 1097
	unitA := 74
	unitB := 93
	unitC := 83
	if _, err := PatchRecipeFile(input, output, Recipe{
		Triggers: []TriggerRecipe{{
			Op:   "add_trigger",
			Name: "Bad Shop Buttons",
			Effects: []EffectRecipe{
				{Op: "add_train_location", SourcePlayer: &player, ObjectListUnitID: &unitA, ObjectListUnitID2: &building, ButtonLocation: &button},
				{Op: "add_train_location", SourcePlayer: &player, ObjectListUnitID: &unitB, ObjectListUnitID2: &building, ButtonLocation: &button},
				{Op: "add_train_location", SourcePlayer: &player, ObjectListUnitID: &unitC, ObjectListUnitID2: &building, ButtonLocation: &overflowButton},
			},
		}},
	}); err != nil {
		t.Fatalf("PatchRecipeFile: %v", err)
	}
	report, err := LintFileWithOptions(output, LintOptions{LoadSafety: true})
	if err != nil {
		t.Fatalf("LintFileWithOptions: %v", err)
	}
	if !hasIssueCode(report.Issues, "load_safety_train_button_slot_collision") {
		t.Fatalf("missing train-button collision issue: %+v", report.Issues)
	}
	if !hasIssueCode(report.Issues, "load_safety_train_button_overflow") {
		t.Fatalf("missing train-button overflow warning: %+v", report.Issues)
	}
	if report.OK {
		t.Fatalf("load-safety collision should make lint fail: %+v", report.Issues)
	}
}

func TestLintLoadSafetyAllowsEmptyActivePlayers(t *testing.T) {
	path := filepath.Join(t.TempDir(), "empty_active_ok.aoe2scenario")
	if _, err := WriteBlankScenarioFile(path, BlankOptions{PlayerCount: 4, HumanSlots: 4, Timestamp: 1800000000}); err != nil {
		t.Fatalf("WriteBlankScenarioFile: %v", err)
	}
	report, err := LintFileWithOptions(path, LintOptions{LoadSafety: true})
	if err != nil {
		t.Fatalf("LintFileWithOptions: %v", err)
	}
	if hasIssueCode(report.Issues, "load_safety_empty_active_player") {
		t.Fatalf("empty active player should not be a load-safety issue: %+v", report.Issues)
	}
	if !report.OK {
		t.Fatalf("editor-style empty active human slots should pass load-safety: %+v", report.Issues)
	}
}

func intPtr(value int) *int {
	return &value
}

func TestDeployCheckCatchesXSNameFailures(t *testing.T) {
	dir := t.TempDir()
	xsDir := filepath.Join(dir, "resources", "_common", "xs")
	if err := os.MkdirAll(xsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(xsDir, "Probe.xs"), []byte("include \"constants.xs\";\nvoid Boot() { xsSetPlayerAttribute(1, BAD_CONST, 1); }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(xsDir, "constants.xs"), []byte("const int BAD_CONST = 22;\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	scen := scenarioWithXSFields("Probe", "Probe.xs", "embedded")
	report, err := scen.DeployCheck(dir)
	if err != nil {
		t.Fatal(err)
	}
	if report.OK {
		t.Fatalf("report unexpectedly OK: %+v", report)
	}
	if !hasDeployFinding(report, "does not match") || !hasDeployFinding(report, "has no .xs extension") || !hasDeployFinding(report, "does not resolve") {
		t.Fatalf("missing expected findings: %+v", report.Findings)
	}
}

func TestDeployCheckCatchesCrossFileNonExtern(t *testing.T) {
	dir := t.TempDir()
	xsDir := filepath.Join(dir, "resources", "_common", "xs")
	if err := os.MkdirAll(xsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(xsDir, "Probe.xs"), []byte("include \"constants.xs\";\nvoid Boot() { xsSetPlayerAttribute(1, BAD_CONST, 1); }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(xsDir, "constants.xs"), []byte("const int BAD_CONST = 22;\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	scen := scenarioWithXSFields("Probe.xs", "Probe.xs", "embedded")
	report, err := scen.DeployCheck(dir)
	if err != nil {
		t.Fatal(err)
	}
	if report.OK {
		t.Fatalf("report unexpectedly OK: %+v", report)
	}
	if !hasDeployFinding(report, "declared without extern") {
		t.Fatalf("missing expected finding: %+v", report.Findings)
	}
	foundFact := false
	for _, finding := range report.Findings {
		if finding.FactID == "xs.cross_file_symbols_require_extern" && finding.FactTier == "engine_verified" {
			foundFact = true
		}
	}
	if !foundFact {
		t.Fatalf("missing extern fact citation: %+v", report.Findings)
	}
}

func TestDeployCheckPassesResolvedExternXS(t *testing.T) {
	dir := t.TempDir()
	xsDir := filepath.Join(dir, "resources", "_common", "xs")
	if err := os.MkdirAll(xsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(xsDir, "Probe.xs"), []byte("include \"constants.xs\";\nvoid Boot() { xsSetPlayerAttribute(1, GOOD_CONST, 1); }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(xsDir, "constants.xs"), []byte("extern const int GOOD_CONST = 22;\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	scen := scenarioWithXSFields("Probe.xs", "Probe.xs", "embedded")
	report, err := scen.DeployCheck(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !report.OK || report.XS.ResolvedRelativePath != "Probe.xs" || report.Analysis == nil {
		t.Fatalf("report = %+v", report)
	}
}

func TestXSCensusInventoriesEmbeddedXS(t *testing.T) {
	scen := scenarioWithXSFields("Entry.xs", "Entry.xs", "include \"missing.xs\";\nconst int LOCAL = 1;\nvoid Boot() {}\n")
	scen.Version = "1.58"
	scen.Triggers = &TriggerInfo{Triggers: []TriggerSummary{{
		Index: 7,
		Name:  "Call Boot",
		EffectData: []EffectSummary{{
			EffectIndex: 3,
			Type:        55,
			TypeName:    "script_call",
			Text:        "Boot(); Other(); Boot();",
		}},
	}}}
	report, err := scen.XSCensus(XSCensusOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if !report.OK {
		t.Fatalf("report unexpectedly failed: %+v", report)
	}
	if report.Counts.ScriptCalls != 1 || report.Counts.Functions != 1 || report.Counts.UniqueCalledFunctions != 2 {
		t.Fatalf("counts = %+v", report.Counts)
	}
	if len(report.ScriptCalls) != 1 || report.ScriptCalls[0].TriggerIndex != 7 || len(report.ScriptCalls[0].Calls) != 2 {
		t.Fatalf("script calls = %+v", report.ScriptCalls)
	}
	if !hasDeployFinding(xsCensusFindings(report), "embedded XS include cannot be resolved") {
		t.Fatalf("missing embedded include finding: %+v", report.Findings)
	}
}

func TestXSCensusUsesDeployTree(t *testing.T) {
	dir := t.TempDir()
	xsDir := filepath.Join(dir, "resources", "_common", "xs")
	if err := os.MkdirAll(xsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(xsDir, "Entry.xs"), []byte("include \"shared.xs\";\nvoid Boot() { xsSetPlayerAttribute(1, SHARED, 1); }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(xsDir, "shared.xs"), []byte("extern const int SHARED = 22;\nvoid Helper() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	scen := scenarioWithXSFields("Entry.xs", "Entry.xs", "embedded")
	report, err := scen.XSCensus(XSCensusOptions{DeployTree: dir})
	if err != nil {
		t.Fatal(err)
	}
	if !report.OK || report.Counts.ResolvedIncludes != 1 || report.Counts.Functions != 2 || report.XS.ResolvedRelativePath != "Entry.xs" {
		t.Fatalf("report = %+v", report)
	}
}

func scenarioWithXSFields(scriptName, scriptPath, content string) *File {
	return &File{root: &parsedRoot{Sections: []*parsedSection{
		{
			Name: "Map",
			Fields: []*parsedNode{
				{Name: "script_name", Value: scriptName},
			},
		},
		{
			Name: "Files",
			Fields: []*parsedNode{
				{Name: "script_file_path", Value: scriptPath},
				{Name: "script_file_content", Value: content},
			},
		},
	}}}
}

func xsCensusFindings(report XSCensusReport) DeployCheckReport {
	return DeployCheckReport{Findings: report.Findings}
}

func hasDeployFinding(report DeployCheckReport, needle string) bool {
	for _, finding := range report.Findings {
		if strings.Contains(finding.What, needle) {
			return true
		}
	}
	return false
}

func TestMapPatchPointsCircleAndLine(t *testing.T) {
	radius := 1
	circle, err := mapPatchPoints(MapRecipe{Op: "set_terrain_circle", X1: 10, Y1: 10, Radius: &radius}, 20, 20)
	if err != nil {
		t.Fatalf("circle points: %v", err)
	}
	if len(circle) != 5 {
		t.Fatalf("circle points len=%d want 5: %+v", len(circle), circle)
	}
	line, err := mapPatchPoints(MapRecipe{Op: "set_terrain_line", X1: 1, Y1: 1, X2: 3, Y2: 3}, 20, 20)
	if err != nil {
		t.Fatalf("line points: %v", err)
	}
	if len(line) != 3 || line[0] != (mapPoint{X: 1, Y: 1}) || line[2] != (mapPoint{X: 3, Y: 3}) {
		t.Fatalf("line points = %+v", line)
	}
}

func TestMapPatchPointsBorder(t *testing.T) {
	thickness := 1
	points, err := mapPatchPoints(MapRecipe{Op: "set_terrain_border", X1: 2, Y1: 3, X2: 5, Y2: 6, Thickness: &thickness}, 10, 10)
	if err != nil {
		t.Fatalf("border points: %v", err)
	}
	if len(points) != 12 {
		t.Fatalf("border points len=%d want 12: %+v", len(points), points)
	}
	if !hasMapPoint(points, mapPoint{X: 2, Y: 3}) || !hasMapPoint(points, mapPoint{X: 5, Y: 6}) {
		t.Fatalf("border missing corner points: %+v", points)
	}
	if hasMapPoint(points, mapPoint{X: 3, Y: 4}) || hasMapPoint(points, mapPoint{X: 4, Y: 5}) {
		t.Fatalf("border included inner points: %+v", points)
	}

	thick := 2
	thickPoints, err := mapPatchPoints(MapRecipe{Op: "set_terrain_border", X1: 0, Y1: 0, X2: 4, Y2: 4, Thickness: &thick}, 10, 10)
	if err != nil {
		t.Fatalf("thick border points: %v", err)
	}
	if len(thickPoints) != 24 {
		t.Fatalf("thick border points len=%d want 24: %+v", len(thickPoints), thickPoints)
	}
	if hasMapPoint(thickPoints, mapPoint{X: 2, Y: 2}) {
		t.Fatalf("thick border included center: %+v", thickPoints)
	}

	zero := 0
	if _, err := mapPatchPoints(MapRecipe{Op: "set_terrain_border", X1: 0, Y1: 0, X2: 4, Y2: 4, Thickness: &zero}, 10, 10); err == nil {
		t.Fatalf("zero-thickness border succeeded")
	}
}

func TestMapPatchPointsCopyTerrainArea(t *testing.T) {
	targetX := 6
	targetY := 7
	points, err := mapPatchPoints(MapRecipe{Op: "copy_terrain_area", X1: 1, Y1: 2, X2: 3, Y2: 4, TargetX: &targetX, TargetY: &targetY}, 10, 10)
	if err != nil {
		t.Fatalf("copy terrain points: %v", err)
	}
	if len(points) != 9 {
		t.Fatalf("copy terrain points len=%d want 9: %+v", len(points), points)
	}
	if points[0] != (mapPoint{X: 1, Y: 2}) || points[len(points)-1] != (mapPoint{X: 3, Y: 4}) {
		t.Fatalf("copy terrain source points = %+v", points)
	}
	if _, err := mapPatchPoints(MapRecipe{Op: "copy_terrain_area", X1: 1, Y1: 2, X2: 3, Y2: 4}, 10, 10); err == nil {
		t.Fatalf("copy terrain without target succeeded")
	}
	targetX = 8
	targetY = 8
	if _, err := mapPatchPoints(MapRecipe{Op: "copy_terrain_area", X1: 1, Y1: 2, X2: 3, Y2: 4, TargetX: &targetX, TargetY: &targetY}, 10, 10); err == nil {
		t.Fatalf("copy terrain accepted out-of-bounds destination")
	}
}

func hasMapPoint(points []mapPoint, needle mapPoint) bool {
	for _, point := range points {
		if point == needle {
			return true
		}
	}
	return false
}

func TestCopyTerrainArea(t *testing.T) {
	input := testfixtures.Path(t, "save-analysis/diagnostics/Replay_Transparency_Diagnostic_v2b_flagged.aoe2scenario")
	if _, err := os.Stat(input); err != nil {
		t.Skipf("diagnostic fixture unavailable: %v", err)
	}
	file, err := Open(input)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	terrain := 222
	elevation := 7
	layer := 3
	targetX := 8
	targetY := 9
	recipe := Recipe{Map: []MapRecipe{
		{Op: "set_terrain_rect", X1: 2, Y1: 3, X2: 3, Y2: 5, TerrainID: &terrain, Elevation: &elevation, Layer: &layer},
		{Op: "copy_terrain_area", X1: 2, Y1: 3, X2: 3, Y2: 5, TargetX: &targetX, TargetY: &targetY},
	}}
	plan, err := file.Plan(recipe)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if plan.MapTilesChanged != 12 {
		t.Fatalf("plan map tiles changed=%d want 12", plan.MapTilesChanged)
	}
	if err := file.ApplyRecipe(recipe); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	tiles, width, _, err := file.mapTiles()
	if err != nil {
		t.Fatalf("mapTiles: %v", err)
	}
	for dy := 0; dy < 3; dy++ {
		for dx := 0; dx < 2; dx++ {
			src := tiles[(3+dy)*width+2+dx]
			dst := tiles[(targetY+dy)*width+targetX+dx]
			for _, field := range []string{"terrain_id", "elevation", "layer"} {
				want, _ := src.intValue(field)
				got, _ := dst.intValue(field)
				if got != want {
					t.Fatalf("copied %s at delta (%d,%d) = %d want %d", field, dx, dy, got, want)
				}
			}
		}
	}
}

func TestCopyTerrainAreaOverlappingUsesSourceSnapshot(t *testing.T) {
	input := testfixtures.Path(t, "save-analysis/diagnostics/Replay_Transparency_Diagnostic_v2b_flagged.aoe2scenario")
	if _, err := os.Stat(input); err != nil {
		t.Skipf("diagnostic fixture unavailable: %v", err)
	}
	file, err := Open(input)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	targetX := 1
	targetY := 0
	values := []int{201, 202, 203}
	recipe := Recipe{
		Map: []MapRecipe{
			{Op: "set_terrain_rect", X1: 0, Y1: 0, X2: 0, Y2: 0, TerrainID: &values[0]},
			{Op: "set_terrain_rect", X1: 1, Y1: 0, X2: 1, Y2: 0, TerrainID: &values[1]},
			{Op: "set_terrain_rect", X1: 2, Y1: 0, X2: 2, Y2: 0, TerrainID: &values[2]},
			{Op: "copy_terrain_area", X1: 0, Y1: 0, X2: 2, Y2: 0, TargetX: &targetX, TargetY: &targetY},
		},
	}
	if err := file.ApplyRecipe(recipe); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	tiles, width, _, err := file.mapTiles()
	if err != nil {
		t.Fatalf("mapTiles: %v", err)
	}
	want := []int{201, 201, 202, 203}
	for x, wantID := range want {
		got, _ := tiles[x].intValue("terrain_id")
		if got != wantID {
			t.Fatalf("terrain at x=%d = %d want %d; width=%d", x, got, wantID, width)
		}
	}
}

func TestNoiseFillTerrainDeterministic(t *testing.T) {
	input := testfixtures.Path(t, "save-analysis/diagnostics/Replay_Transparency_Diagnostic_v2b_flagged.aoe2scenario")
	if _, err := os.Stat(input); err != nil {
		t.Skipf("diagnostic fixture unavailable: %v", err)
	}
	a, err := Open(input)
	if err != nil {
		t.Fatalf("Open A: %v", err)
	}
	b, err := Open(input)
	if err != nil {
		t.Fatalf("Open B: %v", err)
	}
	terrainA := 53
	terrainB := 58
	seed := 77
	scale := 3.5
	threshold := 0.5
	recipe := Recipe{Map: []MapRecipe{
		{Op: "noise_fill", X1: 2, Y1: 2, X2: 12, Y2: 12, TerrainID: &terrainA, TerrainID2: &terrainB, Seed: &seed, Scale: &scale, Threshold: &threshold},
	}}
	plan, err := a.Plan(recipe)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if plan.MapTilesChanged != 121 {
		t.Fatalf("plan map tiles changed=%d want 121", plan.MapTilesChanged)
	}
	if err := a.ApplyRecipe(recipe); err != nil {
		t.Fatalf("ApplyRecipe A: %v", err)
	}
	if err := b.ApplyRecipe(recipe); err != nil {
		t.Fatalf("ApplyRecipe B: %v", err)
	}
	aTiles, width, _, err := a.mapTiles()
	if err != nil {
		t.Fatalf("mapTiles A: %v", err)
	}
	bTiles, _, _, err := b.mapTiles()
	if err != nil {
		t.Fatalf("mapTiles B: %v", err)
	}
	countA := 0
	countB := 0
	for y := 2; y <= 12; y++ {
		for x := 2; x <= 12; x++ {
			gotA, _ := aTiles[y*width+x].intValue("terrain_id")
			gotB, _ := bTiles[y*width+x].intValue("terrain_id")
			if gotA != gotB {
				t.Fatalf("noise_fill not deterministic at (%d,%d): %d != %d", x, y, gotA, gotB)
			}
			switch gotA {
			case terrainA:
				countA++
			case terrainB:
				countB++
			default:
				t.Fatalf("noise_fill terrain at (%d,%d) = %d, want %d or %d", x, y, gotA, terrainA, terrainB)
			}
		}
	}
	if countA == 0 || countB == 0 {
		t.Fatalf("noise_fill did not use both terrains: terrainA=%d terrainB=%d", countA, countB)
	}
}

func TestErodeTerrainSmoothsSingleton(t *testing.T) {
	input := testfixtures.Path(t, "save-analysis/diagnostics/Replay_Transparency_Diagnostic_v2b_flagged.aoe2scenario")
	if _, err := os.Stat(input); err != nil {
		t.Skipf("diagnostic fixture unavailable: %v", err)
	}
	file, err := Open(input)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	base := 22
	speck := 57
	iterations := 1
	recipe := Recipe{Map: []MapRecipe{
		{Op: "set_terrain_rect", X1: 20, Y1: 20, X2: 22, Y2: 22, TerrainID: &base},
		{Op: "set_terrain_rect", X1: 21, Y1: 21, X2: 21, Y2: 21, TerrainID: &speck},
		{Op: "erode", X1: 20, Y1: 20, X2: 22, Y2: 22, Iterations: &iterations},
	}}
	plan, err := file.Plan(recipe)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if plan.MapTilesChanged != 19 {
		t.Fatalf("plan map tiles changed=%d want 19", plan.MapTilesChanged)
	}
	if err := file.ApplyRecipe(recipe); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	tiles, width, _, err := file.mapTiles()
	if err != nil {
		t.Fatalf("mapTiles: %v", err)
	}
	center, _ := tiles[21*width+21].intValue("terrain_id")
	if center != base {
		t.Fatalf("erode center terrain=%d want %d", center, base)
	}
}

func TestMaskAwareNoiseFill(t *testing.T) {
	input := testfixtures.Path(t, "save-analysis/diagnostics/Replay_Transparency_Diagnostic_v2b_flagged.aoe2scenario")
	if _, err := os.Stat(input); err != nil {
		t.Skipf("diagnostic fixture unavailable: %v", err)
	}
	file, err := Open(input)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	base := 22
	a := 53
	b := 58
	radius := 2
	seed := 9
	scale := 2.5
	recipe := Recipe{
		Masks: []MaskRecipe{{Name: "round", Op: "circle", X1: 10, Y1: 10, Radius: &radius}},
		Map: []MapRecipe{
			{Op: "set_terrain_rect", X1: 6, Y1: 6, X2: 14, Y2: 14, TerrainID: &base},
			{Op: "noise_fill", Mask: "round", TerrainID: &a, TerrainID2: &b, Seed: &seed, Scale: &scale},
		},
	}
	plan, err := file.Plan(recipe)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if plan.MapTilesChanged != 94 {
		t.Fatalf("plan map tiles changed=%d want 94", plan.MapTilesChanged)
	}
	if err := file.ApplyRecipe(recipe); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	tiles, width, _, err := file.mapTiles()
	if err != nil {
		t.Fatalf("mapTiles: %v", err)
	}
	outside, _ := tiles[6*width+6].intValue("terrain_id")
	if outside != base {
		t.Fatalf("outside mask terrain=%d want base %d", outside, base)
	}
	insideA, insideB := 0, 0
	for y := 8; y <= 12; y++ {
		for x := 8; x <= 12; x++ {
			if (x-10)*(x-10)+(y-10)*(y-10) > radius*radius {
				continue
			}
			got, _ := tiles[y*width+x].intValue("terrain_id")
			switch got {
			case a:
				insideA++
			case b:
				insideB++
			default:
				t.Fatalf("inside mask terrain at (%d,%d)=%d want %d or %d", x, y, got, a, b)
			}
		}
	}
	if insideA == 0 || insideB == 0 {
		t.Fatalf("masked noise_fill did not use both terrains: a=%d b=%d", insideA, insideB)
	}
}

func TestLayeredCrossfadeSetsBottomTerrainAndTopLayer(t *testing.T) {
	input := testfixtures.Path(t, "save-analysis/diagnostics/Replay_Transparency_Diagnostic_v2b_flagged.aoe2scenario")
	if _, err := os.Stat(input); err != nil {
		t.Skipf("diagnostic fixture unavailable: %v", err)
	}
	file, err := Open(input)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	bottom := 1
	top := 2
	recipe := Recipe{Map: []MapRecipe{
		{Op: "layered_crossfade", X1: 12, Y1: 13, X2: 14, Y2: 13, TerrainID: &bottom, TerrainID2: &top},
	}}
	plan, err := file.Plan(recipe)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if plan.MapTilesChanged != 3 {
		t.Fatalf("plan map tiles changed=%d want 3", plan.MapTilesChanged)
	}
	if err := file.ApplyRecipe(recipe); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	tiles, width, _, err := file.mapTiles()
	if err != nil {
		t.Fatalf("mapTiles: %v", err)
	}
	for x := 12; x <= 14; x++ {
		tile := tiles[13*width+x]
		gotBottom, _ := tile.intValue("terrain_id")
		gotTop, _ := tile.intValue("layer")
		if gotBottom != bottom || gotTop != top {
			t.Fatalf("tile (%d,13) bottom/top = %d/%d want %d/%d", x, gotBottom, gotTop, bottom, top)
		}
	}
}

func TestSemanticErodePreservesOutOfClassTiles(t *testing.T) {
	input := testfixtures.Path(t, "save-analysis/diagnostics/Replay_Transparency_Diagnostic_v2b_flagged.aoe2scenario")
	if _, err := os.Stat(input); err != nil {
		t.Skipf("diagnostic fixture unavailable: %v", err)
	}
	file, err := Open(input)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	water := 22
	land := 0
	iterations := 1
	recipe := Recipe{
		Masks: []MaskRecipe{{Name: "patch", Op: "rect", X1: 30, Y1: 30, X2: 32, Y2: 32}},
		Map: []MapRecipe{
			{Op: "set_terrain_rect", X1: 30, Y1: 30, X2: 32, Y2: 32, TerrainID: &water},
			{Op: "set_terrain_rect", X1: 31, Y1: 31, X2: 31, Y2: 31, TerrainID: &land},
			{Op: "semantic_erode", Mask: "patch", Iterations: &iterations, TerrainIDs: []int{water}},
		},
	}
	if err := file.ApplyRecipe(recipe); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	tiles, width, _, err := file.mapTiles()
	if err != nil {
		t.Fatalf("mapTiles: %v", err)
	}
	center, _ := tiles[31*width+31].intValue("terrain_id")
	if center != land {
		t.Fatalf("semantic_erode changed out-of-class center=%d want %d", center, land)
	}
}

func TestBlobMaskClipsAtMapEdge(t *testing.T) {
	input := testfixtures.Path(t, "save-analysis/diagnostics/Replay_Transparency_Diagnostic_v2b_flagged.aoe2scenario")
	if _, err := os.Stat(input); err != nil {
		t.Skipf("diagnostic fixture unavailable: %v", err)
	}
	file, err := Open(input)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	terrain := 53
	radius := 6
	seed := 4
	recipe := Recipe{
		Masks: []MaskRecipe{{Name: "edge_blob", Op: "blob", X1: 0, Y1: 0, Radius: &radius, Seed: &seed}},
		Map:   []MapRecipe{{Op: "set_terrain_mask", Mask: "edge_blob", TerrainID: &terrain}},
	}
	plan, err := file.Plan(recipe)
	if err != nil {
		t.Fatalf("Plan clipped blob: %v", err)
	}
	if plan.MapTilesChanged == 0 {
		t.Fatalf("clipped blob plan selected no tiles")
	}
	if err := file.ApplyRecipe(recipe); err != nil {
		t.Fatalf("ApplyRecipe clipped blob: %v", err)
	}
	tiles, _, _, err := file.mapTiles()
	if err != nil {
		t.Fatalf("mapTiles: %v", err)
	}
	got, _ := tiles[0].intValue("terrain_id")
	if got != terrain {
		t.Fatalf("corner terrain=%d want %d", got, terrain)
	}
}

func TestClusterScatterPlacesExactCountInHabitat(t *testing.T) {
	input := testfixtures.Path(t, "save-analysis/diagnostics/Replay_Transparency_Diagnostic_v2b_flagged.aoe2scenario")
	if _, err := os.Stat(input); err != nil {
		t.Skipf("diagnostic fixture unavailable: %v", err)
	}
	file, err := Open(input)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	habitat := 53
	count := 5
	spread := 2.0
	minDist := 0.5
	seed := 123
	x1, y1, x2, y2 := 40.0, 40.0, 48.0, 48.0
	recipe := Recipe{
		Map: []MapRecipe{{Op: "set_terrain_rect", X1: 40, Y1: 40, X2: 48, Y2: 48, TerrainID: &habitat}},
		Units: []UnitRecipe{{
			Op:                "cluster_scatter",
			Player:            0,
			UnitConst:         351,
			Count:             &count,
			Centers:           []PointRecipe{{X: 44.5, Y: 44.5}},
			Spread:            &spread,
			MinDistance:       &minDist,
			AllowedTerrainIDs: []int{habitat},
			TargetAreaX1:      &x1,
			TargetAreaY1:      &y1,
			TargetAreaX2:      &x2,
			TargetAreaY2:      &y2,
			Seed:              &seed,
		}},
	}
	plan, err := file.Plan(recipe)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if plan.UnitCountAfter-plan.UnitCountBefore != count {
		t.Fatalf("unit delta=%d want %d", plan.UnitCountAfter-plan.UnitCountBefore, count)
	}
	if err := file.ApplyRecipe(recipe); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	matches, err := file.findUnitsInArea(UnitRecipe{TargetAreaX1: &x1, TargetAreaY1: &y1, TargetAreaX2: &x2, TargetAreaY2: &y2, TargetUnitConst: intPtr(351)})
	if err != nil {
		t.Fatalf("findUnitsInArea: %v", err)
	}
	if len(matches) != count {
		t.Fatalf("cluster scatter placed %d unit(s), want %d", len(matches), count)
	}
	tiles, width, _, err := file.mapTiles()
	if err != nil {
		t.Fatalf("mapTiles: %v", err)
	}
	for _, match := range matches {
		x, _ := match.Unit.floatValue("x")
		y, _ := match.Unit.floatValue("y")
		got, _ := tiles[int(math.Floor(y))*width+int(math.Floor(x))].intValue("terrain_id")
		if got != habitat {
			t.Fatalf("cluster unit at %.2f,%.2f on terrain %d want %d", x, y, got, habitat)
		}
	}
}

func TestClusterScatterPackedShapeStaysContiguous(t *testing.T) {
	input := testfixtures.Path(t, "save-analysis/diagnostics/Replay_Transparency_Diagnostic_v2b_flagged.aoe2scenario")
	if _, err := os.Stat(input); err != nil {
		t.Skipf("diagnostic fixture unavailable: %v", err)
	}
	file, err := Open(input)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	habitat := 53
	count := 18
	spread := 3.0
	minDist := 0.45
	seed := 456
	x1, y1, x2, y2 := 70.0, 70.0, 80.0, 80.0
	recipe := Recipe{
		Map: []MapRecipe{{Op: "set_terrain_rect", X1: 70, Y1: 70, X2: 80, Y2: 80, TerrainID: &habitat}},
		Units: []UnitRecipe{{
			Op:                "cluster_scatter",
			Player:            0,
			UnitConst:         351,
			Count:             &count,
			Centers:           []PointRecipe{{X: 75.5, Y: 75.5}},
			ClusterShape:      "packed",
			Spread:            &spread,
			MinDistance:       &minDist,
			AllowedTerrainIDs: []int{habitat},
			TargetAreaX1:      &x1,
			TargetAreaY1:      &y1,
			TargetAreaX2:      &x2,
			TargetAreaY2:      &y2,
			Seed:              &seed,
		}},
	}
	if err := file.ApplyRecipe(recipe); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	matches, err := file.findUnitsInArea(UnitRecipe{TargetAreaX1: &x1, TargetAreaY1: &y1, TargetAreaX2: &x2, TargetAreaY2: &y2, TargetUnitConst: intPtr(351)})
	if err != nil {
		t.Fatalf("findUnitsInArea: %v", err)
	}
	if len(matches) != count {
		t.Fatalf("cluster scatter placed %d unit(s), want %d", len(matches), count)
	}
	isolated := 0
	for i, match := range matches {
		x, _ := match.Unit.floatValue("x")
		y, _ := match.Unit.floatValue("y")
		hasNeighbor := false
		for j, other := range matches {
			if i == j {
				continue
			}
			ox, _ := other.Unit.floatValue("x")
			oy, _ := other.Unit.floatValue("y")
			if math.Hypot(x-ox, y-oy) <= 1.6 {
				hasNeighbor = true
				break
			}
		}
		if !hasNeighbor {
			isolated++
		}
	}
	if isolated > 1 {
		t.Fatalf("packed cluster left %d isolated unit(s), want <= 1", isolated)
	}
}

func TestClusterScatterGridContiguousStaysOnTileCenters(t *testing.T) {
	file := openClusterScatterFixture(t)
	count := 14
	spread := 3.0
	minDist := 0.45
	seed := 789
	adds, err := file.ClusterScatterUnits(UnitRecipe{
		Op:           "cluster_scatter",
		Player:       0,
		UnitConst:    349,
		Count:        &count,
		Centers:      []PointRecipe{{X: 95.5, Y: 95.5}},
		Snap:         "grid",
		Distribution: "contiguous",
		Spread:       &spread,
		MinDistance:  &minDist,
		Seed:         &seed,
	})
	if err != nil {
		t.Fatalf("ClusterScatterUnits: %v", err)
	}
	if len(adds) != count {
		t.Fatalf("cluster scatter returned %d add(s), want %d", len(adds), count)
	}
	assertAllTileCenters(t, adds)
	assertMostlyContiguous(t, adds, 1)
}

func TestClusterScatterFractionalDispersedJitterUsesSubTileCoords(t *testing.T) {
	file := openClusterScatterFixture(t)
	count := 18
	spread := 2.5
	jitter := 0.2
	seed := 321
	adds, err := file.ClusterScatterUnits(UnitRecipe{
		Op:           "cluster_scatter",
		Player:       0,
		UnitConst:    1366,
		Count:        &count,
		Centers:      []PointRecipe{{X: 105.5, Y: 105.5}},
		Snap:         "fractional",
		Distribution: "dispersed",
		Jitter:       &jitter,
		Spread:       &spread,
		Seed:         &seed,
	})
	if err != nil {
		t.Fatalf("ClusterScatterUnits: %v", err)
	}
	if len(adds) != count {
		t.Fatalf("cluster scatter returned %d add(s), want %d", len(adds), count)
	}
	if countTileCenters(adds) == len(adds) {
		t.Fatalf("fractional dispersed scatter produced only tile-center coordinates")
	}
	if scatterMaxPairDistance(adds) < 2.0 {
		t.Fatalf("fractional dispersed scatter did not spread around center enough")
	}
}

func TestFootprintBlocksIgnoresFlatCollisionBoxes(t *testing.T) {
	cases := []struct {
		name    string
		x, y, z float32
		want    bool
	}{
		{"flower 1366: no footprint", 0, 0, 0, false},
		{"tree 349: footprint + height", 0.5, 0.5, 2, true},
		{"statue 1279: footprint + height", 0.5, 0.5, 3, true},
		{"cactus 709: footprint, zero height", 0.5, 0.5, 0, false},
		{"rugs 711: footprint, zero height", 0.5, 0.5, 0, false},
		{"flag: tiny footprint, zero height", 0.1, 0.1, 0, false},
		{"dummy: height without footprint", 0, 0, 2, false},
	}
	for _, c := range cases {
		if got := footprintBlocks(c.x, c.y, c.z); got != c.want {
			t.Errorf("%s: footprintBlocks(%v,%v,%v)=%v want %v", c.name, c.x, c.y, c.z, got, c.want)
		}
	}
}

func TestResolveUnitPlacementPolicy(t *testing.T) {
	x, y := 10.25, 20.75
	base := UnitRecipe{Op: "add_unit", UnitConst: 1366, X: &x, Y: &y}

	legacy, err := resolveUnitPlacement(base, nil)
	if err != nil {
		t.Fatalf("default placement: %v", err)
	}
	if *legacy.X != 10.5 || *legacy.Y != 20.5 {
		t.Fatalf("default placement = (%v,%v), want (10.5,20.5)", *legacy.X, *legacy.Y)
	}

	grid := base
	grid.Snap = "grid"
	grid, err = resolveUnitPlacement(grid, nil)
	if err != nil {
		t.Fatalf("grid placement: %v", err)
	}
	if *grid.X != 10.5 || *grid.Y != 20.5 {
		t.Fatalf("grid snap = (%v,%v), want (10.5,20.5)", *grid.X, *grid.Y)
	}

	fractional := base
	fractional.Snap = "fractional"
	fractional, err = resolveUnitPlacement(fractional, nil)
	if err != nil {
		t.Fatalf("fractional placement: %v", err)
	}
	if *fractional.X != x || *fractional.Y != y {
		t.Fatalf("fractional snap changed coordinates: (%v,%v)", *fractional.X, *fractional.Y)
	}

	auto := base
	auto.Snap = "auto"
	auto, err = resolveUnitPlacement(auto, nil)
	if err != nil {
		t.Fatalf("auto placement without DAT: %v", err)
	}
	if *auto.X != 10.5 || *auto.Y != 20.5 {
		t.Fatalf("auto without DAT = (%v,%v), want conservative grid", *auto.X, *auto.Y)
	}

	bad := base
	bad.Snap = "diagonal"
	if _, err := resolveUnitPlacement(bad, nil); err == nil {
		t.Fatal("invalid snap policy unexpectedly accepted")
	}
}

func TestLintWarnsOffGridBlockingPlacement(t *testing.T) {
	report := LintReport{}
	lintUnitPlacement(&report, 1, UnitSummary{Index: 3, UnitConst: 12, X: 10.25, Y: 20.5})
	if !hasIssueCode(report.Issues, "unit_off_grid_blocking_placement") {
		t.Fatalf("missing off-grid blocking warning: %+v", report.Issues)
	}
	decorative := LintReport{}
	lintUnitPlacement(&decorative, 0, UnitSummary{Index: 4, UnitConst: 1366, X: 10.25, Y: 20.25})
	if hasIssueCode(decorative.Issues, "unit_off_grid_blocking_placement") {
		t.Fatalf("decorative Gaia art was flagged: %+v", decorative.Issues)
	}
}

func TestClusterScatterAutoSnapUsesFootprintResolver(t *testing.T) {
	file := openClusterScatterFixture(t)
	count := 8
	spread := 2.0
	seed := 654
	resolver := &clusterFootprintResolver{memo: map[int]bool{
		349:  false,
		1366: true,
	}}
	blocking, err := file.clusterScatterUnits(UnitRecipe{
		Op:           "cluster_scatter",
		Player:       0,
		UnitConst:    349,
		Count:        &count,
		Centers:      []PointRecipe{{X: 115.5, Y: 115.5}},
		Snap:         "auto",
		Distribution: "dispersed",
		Spread:       &spread,
		Seed:         &seed,
	}, resolver)
	if err != nil {
		t.Fatalf("blocking clusterScatterUnits: %v", err)
	}
	assertAllTileCenters(t, blocking)

	decorative, err := file.clusterScatterUnits(UnitRecipe{
		Op:           "cluster_scatter",
		Player:       0,
		UnitConst:    1366,
		Count:        &count,
		Centers:      []PointRecipe{{X: 125.5, Y: 125.5}},
		Snap:         "auto",
		Distribution: "dispersed",
		Spread:       &spread,
		Seed:         &seed,
	}, resolver)
	if err != nil {
		t.Fatalf("decorative clusterScatterUnits: %v", err)
	}
	if countTileCenters(decorative) == len(decorative) {
		t.Fatalf("auto snap decorative unit produced only tile-center coordinates")
	}
}

func TestClusterScatterBackCompatPackedMapsToAutoContiguousNoJitter(t *testing.T) {
	file := openClusterScatterFixture(t)
	count := 10
	spread := 3.0
	seed := 987
	resolver := &clusterFootprintResolver{memo: map[int]bool{349: false}}
	adds, err := file.clusterScatterUnits(UnitRecipe{
		Op:           "cluster_scatter",
		Player:       0,
		UnitConst:    349,
		Count:        &count,
		Centers:      []PointRecipe{{X: 135.5, Y: 135.5}},
		ClusterShape: "packed",
		Spread:       &spread,
		Seed:         &seed,
	}, resolver)
	if err != nil {
		t.Fatalf("clusterScatterUnits: %v", err)
	}
	assertAllTileCenters(t, adds)
	assertMostlyContiguous(t, adds, 1)
}

func TestClusterScatterExactFailure(t *testing.T) {
	input := testfixtures.Path(t, "save-analysis/diagnostics/Replay_Transparency_Diagnostic_v2b_flagged.aoe2scenario")
	if _, err := os.Stat(input); err != nil {
		t.Skipf("diagnostic fixture unavailable: %v", err)
	}
	file, err := Open(input)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	habitat := 53
	count := 3
	minDist := 10.0
	x1, y1, x2, y2 := 60.0, 60.0, 60.9, 60.9
	recipe := Recipe{
		Map: []MapRecipe{{Op: "set_terrain_rect", X1: 60, Y1: 60, X2: 60, Y2: 60, TerrainID: &habitat}},
		Units: []UnitRecipe{{
			Op:                "cluster_scatter",
			Player:            0,
			UnitConst:         351,
			Count:             &count,
			Centers:           []PointRecipe{{X: 60.5, Y: 60.5}},
			MinDistance:       &minDist,
			AllowedTerrainIDs: []int{habitat},
			TargetAreaX1:      &x1,
			TargetAreaY1:      &y1,
			TargetAreaX2:      &x2,
			TargetAreaY2:      &y2,
		}},
	}
	if err := file.ApplyRecipe(recipe); err == nil {
		t.Fatalf("cluster_scatter exact placement unexpectedly succeeded")
	}
}

func openClusterScatterFixture(t *testing.T) *File {
	t.Helper()
	input := testfixtures.Path(t, "save-analysis/diagnostics/Replay_Transparency_Diagnostic_v2b_flagged.aoe2scenario")
	if _, err := os.Stat(input); err != nil {
		t.Skipf("diagnostic fixture unavailable: %v", err)
	}
	file, err := Open(input)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	return file
}

func assertAllTileCenters(t *testing.T, adds []UnitRecipe) {
	t.Helper()
	for _, add := range adds {
		if add.X == nil || add.Y == nil {
			t.Fatalf("add unit missing coordinates: %+v", add)
		}
		if !isTileCenter(*add.X) || !isTileCenter(*add.Y) {
			t.Fatalf("coord %.4f,%.4f is not tile-centered", *add.X, *add.Y)
		}
	}
}

func countTileCenters(adds []UnitRecipe) int {
	count := 0
	for _, add := range adds {
		if add.X != nil && add.Y != nil && isTileCenter(*add.X) && isTileCenter(*add.Y) {
			count++
		}
	}
	return count
}

func isTileCenter(v float64) bool {
	return math.Abs((v-math.Floor(v))-0.5) < 1e-9
}

func assertMostlyContiguous(t *testing.T, adds []UnitRecipe, maxIsolated int) {
	t.Helper()
	isolated := 0
	for i, add := range adds {
		hasNeighbor := false
		for j, other := range adds {
			if i == j || add.X == nil || add.Y == nil || other.X == nil || other.Y == nil {
				continue
			}
			if math.Hypot(*add.X-*other.X, *add.Y-*other.Y) <= 1.6 {
				hasNeighbor = true
				break
			}
		}
		if !hasNeighbor {
			isolated++
		}
	}
	if isolated > maxIsolated {
		t.Fatalf("cluster left %d isolated unit(s), want <= %d", isolated, maxIsolated)
	}
}

func scatterMaxPairDistance(adds []UnitRecipe) float64 {
	maxDist := 0.0
	for i, add := range adds {
		for j, other := range adds {
			if j <= i || add.X == nil || add.Y == nil || other.X == nil || other.Y == nil {
				continue
			}
			maxDist = math.Max(maxDist, math.Hypot(*add.X-*other.X, *add.Y-*other.Y))
		}
	}
	return maxDist
}

func TestEditExistingTriggerRaw(t *testing.T) {
	spec, err := LoadCurrentDESpec()
	if err != nil {
		t.Fatalf("LoadCurrentDESpec: %v", err)
	}
	raw, err := buildTriggerFromRecipe(spec, TriggerRecipe{
		Op:   "add_trigger",
		Name: "Original",
	})
	if err != nil {
		t.Fatalf("buildTriggerFromRecipe: %v", err)
	}
	triggerSpec, _, err := triggerEffectSpecs(spec)
	if err != nil {
		t.Fatalf("triggerEffectSpecs: %v", err)
	}
	p := parser{data: raw, sections: map[string]*parsedSection{}}
	node, err := p.parseNode("TriggerStruct", triggerSpec, nil)
	if err != nil {
		t.Fatalf("parse trigger raw: %v", err)
	}
	newName := "Edited"
	if err := setStringField(node, "trigger_name", "str32", newName); err != nil {
		t.Fatalf("setStringField: %v", err)
	}
	if err := setIntField(node, "enabled", "u32", 1); err != nil {
		t.Fatalf("setIntField enabled: %v", err)
	}
	triggerID := 2
	if err := appendEffectsToTrigger(spec, node, []EffectRecipe{{Op: "deactivate_trigger", TriggerID: &triggerID}}); err != nil {
		t.Fatalf("appendEffectsToTrigger: %v", err)
	}
	timer := 9
	if err := appendConditionsToTrigger(spec, node, []ConditionRecipe{{Op: "timer", Timer: &timer}}); err != nil {
		t.Fatalf("appendConditionsToTrigger: %v", err)
	}
	roundTrip := parser{data: node.raw(), sections: map[string]*parsedSection{}}
	edited, err := roundTrip.parseNode("TriggerStruct", triggerSpec, nil)
	if err != nil {
		t.Fatalf("parse edited trigger raw: %v", err)
	}
	name, _ := edited.stringValue("trigger_name")
	if name != newName {
		t.Fatalf("trigger_name = %q, want %q", name, newName)
	}
	checkIntField(t, edited, "enabled", 1)
	effects := edited.list("effect_data")
	if len(effects) != 1 {
		t.Fatalf("effects = %d, want 1", len(effects))
	}
	checkIntField(t, effects[0], "effect_type", 9)
	checkIntField(t, effects[0], "trigger_id", triggerID)
	conditions := edited.list("condition_data")
	if len(conditions) != 1 {
		t.Fatalf("conditions = %d, want 1", len(conditions))
	}
	checkIntField(t, conditions[0], "condition_type", 10)
	checkIntField(t, conditions[0], "timer", timer)
}

func TestBuildUnitFromRecipeRaw(t *testing.T) {
	spec, err := LoadCurrentDESpec()
	if err != nil {
		t.Fatalf("LoadCurrentDESpec: %v", err)
	}
	x := 10.5
	y := 12.5
	refID := 999
	caption := "Click This Marker"
	raw, err := buildUnitFromRecipe(spec, UnitRecipe{
		Op:            "add_unit",
		Player:        1,
		UnitConst:     83,
		X:             &x,
		Y:             &y,
		ReferenceID:   &refID,
		CaptionString: caption,
	}, 1)
	if err != nil {
		t.Fatalf("buildUnitFromRecipe: %v", err)
	}
	unitSpec, err := unitStructSpec(spec)
	if err != nil {
		t.Fatalf("unitStructSpec: %v", err)
	}
	p := parser{data: raw, sections: map[string]*parsedSection{}}
	unit, err := p.parseNode("UnitStruct", unitSpec, nil)
	if err != nil {
		t.Fatalf("parse unit raw: %v", err)
	}
	checkIntField(t, unit, "reference_id", refID)
	checkIntField(t, unit, "unit_const", 83)
	checkIntField(t, unit, "status", 2)
	gotX, _ := unit.floatValue("x")
	gotY, _ := unit.floatValue("y")
	if gotX != x || gotY != y {
		t.Fatalf("coords = %v,%v want %v,%v", gotX, gotY, x, y)
	}
	gotCaption, _ := unit.stringValue("caption_string")
	if gotCaption != caption {
		t.Fatalf("caption_string = %q, want %q", gotCaption, caption)
	}
	assertLengthPrefixedStringRaw(t, unit.field("caption_string").Raw, caption)
}

func TestEditAndRemoveUnitSelectors(t *testing.T) {
	input := testfixtures.Path(t, "save-analysis/diagnostics/Replay_Transparency_Diagnostic_v2b_flagged.aoe2scenario")
	if _, err := os.Stat(input); err != nil {
		t.Skipf("diagnostic fixture unavailable: %v", err)
	}
	file, err := Open(input)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	player := 1
	section := file.root.section("Units").list("players_units")[player]
	unitField := section.field("units")
	index := len(unitField.Elements)
	x := 12.5
	y := 13.5
	caption := "selector-index-target"
	if err := file.AddUnit(UnitRecipe{Op: "add_unit", Player: player, UnitConst: 83, X: &x, Y: &y, CaptionString: caption}); err != nil {
		t.Fatalf("AddUnit: %v", err)
	}
	newX := 44.5
	newCaption := "selector-index-edited"
	if err := file.EditUnit(UnitRecipe{Op: "edit_unit", TargetPlayer: &player, TargetIndex: &index, X: &newX, CaptionString: newCaption}); err != nil {
		t.Fatalf("EditUnit by target_index: %v", err)
	}
	unit, _, _, err := file.findUnitByCaption(newCaption, &player)
	if err != nil {
		t.Fatalf("find edited unit: %v", err)
	}
	gotX, _ := unit.floatValue("x")
	if gotX != newX {
		t.Fatalf("edited x = %v, want %v", gotX, newX)
	}
	destPlayer := 2
	if err := file.EditUnit(UnitRecipe{Op: "edit_unit", TargetCaption: newCaption, TargetPlayer: &player, SetPlayer: &destPlayer}); err != nil {
		t.Fatalf("EditUnit set_player by caption: %v", err)
	}
	if _, _, _, err := file.findUnitByCaption(newCaption, &destPlayer); err != nil {
		t.Fatalf("moved unit not found on destination player: %v", err)
	}
	if err := file.RemoveUnit(UnitRecipe{Op: "remove_unit", TargetCaption: newCaption, TargetPlayer: &destPlayer}); err != nil {
		t.Fatalf("RemoveUnit by caption: %v", err)
	}
	if _, _, _, err := file.findUnitByCaption(newCaption, &destPlayer); err == nil {
		t.Fatal("removed unit still found by caption")
	}
}

func TestUnitCaptionSelectorRejectsAmbiguousMatches(t *testing.T) {
	input := testfixtures.Path(t, "save-analysis/diagnostics/Replay_Transparency_Diagnostic_v2b_flagged.aoe2scenario")
	if _, err := os.Stat(input); err != nil {
		t.Skipf("diagnostic fixture unavailable: %v", err)
	}
	file, err := Open(input)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	caption := "selector-ambiguous"
	x := 10.5
	y := 10.5
	if err := file.AddUnit(UnitRecipe{Op: "add_unit", Player: 1, UnitConst: 83, X: &x, Y: &y, CaptionString: caption}); err != nil {
		t.Fatalf("AddUnit P1: %v", err)
	}
	if err := file.AddUnit(UnitRecipe{Op: "add_unit", Player: 2, UnitConst: 83, X: &x, Y: &y, CaptionString: caption}); err != nil {
		t.Fatalf("AddUnit P2: %v", err)
	}
	if err := file.EditUnit(UnitRecipe{Op: "edit_unit", TargetCaption: caption, X: &x}); err == nil || !strings.Contains(err.Error(), "matched 2 units") {
		t.Fatalf("ambiguous caption error = %v, want matched 2 units", err)
	}
	targetPlayer := 2
	if err := file.EditUnit(UnitRecipe{Op: "edit_unit", TargetCaption: caption, TargetPlayer: &targetPlayer, X: &x}); err != nil {
		t.Fatalf("player-scoped caption edit: %v", err)
	}
}

func TestRemoveUnitRejectsSelectedObjectReferences(t *testing.T) {
	input := testfixtures.Path(t, "save-analysis/diagnostics/Replay_Transparency_Diagnostic_v2b_flagged.aoe2scenario")
	if _, err := os.Stat(input); err != nil {
		t.Skipf("diagnostic fixture unavailable: %v", err)
	}
	file, err := Open(input)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	refID := 990501
	x := 15.5
	y := 15.5
	if err := file.AddUnit(UnitRecipe{Op: "add_unit", Player: 1, UnitConst: 83, X: &x, Y: &y, ReferenceID: &refID}); err != nil {
		t.Fatalf("AddUnit: %v", err)
	}
	if err := file.AddTrigger(TriggerRecipe{
		Op:   "add_trigger",
		Name: "references selected unit",
		Effects: []EffectRecipe{
			{Op: "kill_object", SourcePlayer: intPtr(1), SelectedObjectIDs: []int{refID}},
		},
	}); err != nil {
		t.Fatalf("AddTrigger: %v", err)
	}
	err = file.RemoveUnit(UnitRecipe{Op: "remove_unit", ReferenceID: &refID})
	if err == nil || !strings.Contains(err.Error(), "selected_object_ids[0] references it") {
		t.Fatalf("RemoveUnit error = %v, want selected_object_ids reference refusal", err)
	}
}

func TestRemoveUnitRejectsConditionAndLocationReferences(t *testing.T) {
	input := testfixtures.Path(t, "save-analysis/diagnostics/Replay_Transparency_Diagnostic_v2b_flagged.aoe2scenario")
	if _, err := os.Stat(input); err != nil {
		t.Skipf("diagnostic fixture unavailable: %v", err)
	}
	file, err := Open(input)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	refID := 990502
	x := 16.5
	y := 16.5
	timer := 1
	locationX := 20
	locationY := 20
	if err := file.AddUnit(UnitRecipe{Op: "add_unit", Player: 1, UnitConst: 83, X: &x, Y: &y, ReferenceID: &refID}); err != nil {
		t.Fatalf("AddUnit: %v", err)
	}
	if err := file.AddTrigger(TriggerRecipe{
		Op:   "add_trigger",
		Name: "references location unit",
		Conditions: []ConditionRecipe{
			{Op: "object_selected", UnitObject: &refID},
			{Op: "timer", Timer: &timer},
		},
		Effects: []EffectRecipe{
			{Op: "task_object", SourcePlayer: intPtr(1), LocationX: &locationX, LocationY: &locationY, LocationObjectReference: &refID},
		},
	}); err != nil {
		t.Fatalf("AddTrigger: %v", err)
	}
	err = file.RemoveUnit(UnitRecipe{Op: "remove_unit", ReferenceID: &refID})
	if err == nil || !strings.Contains(err.Error(), "condition 0 field unit_object references it") {
		t.Fatalf("RemoveUnit error = %v, want unit_object reference refusal", err)
	}

	file, err = Open(input)
	if err != nil {
		t.Fatalf("Open second file: %v", err)
	}
	refID = 990503
	if err := file.AddUnit(UnitRecipe{Op: "add_unit", Player: 1, UnitConst: 83, X: &x, Y: &y, ReferenceID: &refID}); err != nil {
		t.Fatalf("AddUnit location target: %v", err)
	}
	if err := file.AddTrigger(TriggerRecipe{
		Op:   "add_trigger",
		Name: "references location unit only",
		Effects: []EffectRecipe{
			{Op: "task_object", SourcePlayer: intPtr(1), LocationX: &locationX, LocationY: &locationY, LocationObjectReference: &refID},
		},
	}); err != nil {
		t.Fatalf("AddTrigger location-only: %v", err)
	}
	err = file.RemoveUnit(UnitRecipe{Op: "remove_unit", ReferenceID: &refID})
	if err == nil || !strings.Contains(err.Error(), "effect 0 field location_object_reference references it") {
		t.Fatalf("RemoveUnit location error = %v, want location_object_reference refusal", err)
	}
}

func TestRemoveUnitsInArea(t *testing.T) {
	input := testfixtures.Path(t, "save-analysis/diagnostics/Replay_Transparency_Diagnostic_v2b_flagged.aoe2scenario")
	if _, err := os.Stat(input); err != nil {
		t.Skipf("diagnostic fixture unavailable: %v", err)
	}
	file, err := Open(input)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	before := file.Units.Total
	player := 1
	unitConst := 83
	x1, y1 := 30.5, 30.5
	x2, y2 := 31.5, 31.5
	x3, y3 := 45.5, 45.5
	ref1 := 991001
	ref2 := 991002
	ref3 := 991003
	if err := file.AddUnit(UnitRecipe{Op: "add_unit", Player: player, UnitConst: unitConst, X: &x1, Y: &y1, ReferenceID: &ref1}); err != nil {
		t.Fatalf("AddUnit ref1: %v", err)
	}
	if err := file.AddUnit(UnitRecipe{Op: "add_unit", Player: player, UnitConst: unitConst, X: &x2, Y: &y2, ReferenceID: &ref2}); err != nil {
		t.Fatalf("AddUnit ref2: %v", err)
	}
	if err := file.AddUnit(UnitRecipe{Op: "add_unit", Player: player, UnitConst: unitConst, X: &x3, Y: &y3, ReferenceID: &ref3}); err != nil {
		t.Fatalf("AddUnit ref3: %v", err)
	}
	ax1, ay1, ax2, ay2 := 30.0, 30.0, 32.0, 32.0
	plan, err := file.Plan(Recipe{Units: []UnitRecipe{{
		Op:              "remove_units_in_area",
		TargetPlayer:    &player,
		TargetUnitConst: &unitConst,
		TargetAreaX1:    &ax1,
		TargetAreaY1:    &ay1,
		TargetAreaX2:    &ax2,
		TargetAreaY2:    &ay2,
	}}})
	if err != nil {
		t.Fatalf("Plan remove_units_in_area: %v", err)
	}
	if plan.UnitCountAfter != before+1 {
		t.Fatalf("plan unit_count_after = %d, want %d", plan.UnitCountAfter, before+1)
	}
	removed, err := file.RemoveUnitsInArea(UnitRecipe{
		Op:              "remove_units_in_area",
		TargetPlayer:    &player,
		TargetUnitConst: &unitConst,
		TargetAreaX1:    &ax1,
		TargetAreaY1:    &ay1,
		TargetAreaX2:    &ax2,
		TargetAreaY2:    &ay2,
	})
	if err != nil {
		t.Fatalf("RemoveUnitsInArea: %v", err)
	}
	if removed != 2 {
		t.Fatalf("removed = %d, want 2", removed)
	}
	if file.Units.Total != before+1 {
		t.Fatalf("unit total = %d, want %d", file.Units.Total, before+1)
	}
	if _, _, _, err := file.findUnitByReferenceID(ref1); err == nil {
		t.Fatal("ref1 survived area removal")
	}
	if _, _, _, err := file.findUnitByReferenceID(ref2); err == nil {
		t.Fatal("ref2 survived area removal")
	}
	if _, _, _, err := file.findUnitByReferenceID(ref3); err != nil {
		t.Fatalf("outside unit missing after area removal: %v", err)
	}
}

func TestRemoveUnitsInAreaRejectsReferencesBeforeMutation(t *testing.T) {
	input := testfixtures.Path(t, "save-analysis/diagnostics/Replay_Transparency_Diagnostic_v2b_flagged.aoe2scenario")
	if _, err := os.Stat(input); err != nil {
		t.Skipf("diagnostic fixture unavailable: %v", err)
	}
	file, err := Open(input)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	before := file.Units.Total
	player := 1
	unitConst := 83
	x1, y1 := 34.5, 34.5
	x2, y2 := 35.5, 35.5
	ref1 := 991101
	ref2 := 991102
	if err := file.AddUnit(UnitRecipe{Op: "add_unit", Player: player, UnitConst: unitConst, X: &x1, Y: &y1, ReferenceID: &ref1}); err != nil {
		t.Fatalf("AddUnit ref1: %v", err)
	}
	if err := file.AddUnit(UnitRecipe{Op: "add_unit", Player: player, UnitConst: unitConst, X: &x2, Y: &y2, ReferenceID: &ref2}); err != nil {
		t.Fatalf("AddUnit ref2: %v", err)
	}
	if err := file.AddTrigger(TriggerRecipe{
		Op:   "add_trigger",
		Name: "area delete blocker",
		Effects: []EffectRecipe{
			{Op: "kill_object", SourcePlayer: intPtr(1), SelectedObjectIDs: []int{ref2}},
		},
	}); err != nil {
		t.Fatalf("AddTrigger: %v", err)
	}
	ax1, ay1, ax2, ay2 := 34.0, 34.0, 36.0, 36.0
	recipe := UnitRecipe{
		Op:              "remove_units_in_area",
		TargetPlayer:    &player,
		TargetUnitConst: &unitConst,
		TargetAreaX1:    &ax1,
		TargetAreaY1:    &ay1,
		TargetAreaX2:    &ax2,
		TargetAreaY2:    &ay2,
	}
	if _, err := file.Plan(Recipe{Units: []UnitRecipe{recipe}}); err == nil || !strings.Contains(err.Error(), "selected_object_ids[0] references it") {
		t.Fatalf("Plan remove_units_in_area error = %v, want selected-object reference refusal", err)
	}
	if _, err := file.RemoveUnitsInArea(recipe); err == nil || !strings.Contains(err.Error(), "selected_object_ids[0] references it") {
		t.Fatalf("RemoveUnitsInArea error = %v, want selected-object reference refusal", err)
	}
	if file.Units.Total != before+2 {
		t.Fatalf("unit total changed after rejected area removal: %d, want %d", file.Units.Total, before+2)
	}
	if _, _, _, err := file.findUnitByReferenceID(ref1); err != nil {
		t.Fatalf("unreferenced matched unit was removed before blocker: %v", err)
	}
	if _, _, _, err := file.findUnitByReferenceID(ref2); err != nil {
		t.Fatalf("referenced matched unit missing after rejected removal: %v", err)
	}
}

func TestMoveUnitsInArea(t *testing.T) {
	input := testfixtures.Path(t, "save-analysis/diagnostics/Replay_Transparency_Diagnostic_v2b_flagged.aoe2scenario")
	if _, err := os.Stat(input); err != nil {
		t.Skipf("diagnostic fixture unavailable: %v", err)
	}
	file, err := Open(input)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	before := file.Units.Total
	player := 1
	unitConst := 83
	x1, y1 := 20.5, 20.5
	x2, y2 := 21.5, 21.5
	x3, y3 := 30.5, 30.5
	ref1 := 991401
	ref2 := 991402
	ref3 := 991403
	if err := file.AddUnit(UnitRecipe{Op: "add_unit", Player: player, UnitConst: unitConst, X: &x1, Y: &y1, ReferenceID: &ref1}); err != nil {
		t.Fatalf("AddUnit ref1: %v", err)
	}
	if err := file.AddUnit(UnitRecipe{Op: "add_unit", Player: player, UnitConst: unitConst, X: &x2, Y: &y2, ReferenceID: &ref2}); err != nil {
		t.Fatalf("AddUnit ref2: %v", err)
	}
	if err := file.AddUnit(UnitRecipe{Op: "add_unit", Player: player, UnitConst: unitConst, X: &x3, Y: &y3, ReferenceID: &ref3}); err != nil {
		t.Fatalf("AddUnit ref3: %v", err)
	}
	ax1, ay1, ax2, ay2 := 20.0, 20.0, 22.0, 22.0
	offsetX, offsetY := 4.0, 5.0
	recipe := UnitRecipe{
		Op:              "move_units_in_area",
		TargetPlayer:    &player,
		TargetUnitConst: &unitConst,
		TargetAreaX1:    &ax1,
		TargetAreaY1:    &ay1,
		TargetAreaX2:    &ax2,
		TargetAreaY2:    &ay2,
		OffsetX:         &offsetX,
		OffsetY:         &offsetY,
	}
	plan, err := file.Plan(Recipe{Units: []UnitRecipe{recipe}})
	if err != nil {
		t.Fatalf("Plan move_units_in_area: %v", err)
	}
	if plan.UnitCountAfter != before+3 {
		t.Fatalf("plan unit_count_after = %d, want %d", plan.UnitCountAfter, before+3)
	}
	moved, err := file.MoveUnitsInArea(recipe)
	if err != nil {
		t.Fatalf("MoveUnitsInArea: %v", err)
	}
	if moved != 2 {
		t.Fatalf("moved = %d, want 2", moved)
	}
	if file.Units.Total != before+3 {
		t.Fatalf("unit total = %d, want %d", file.Units.Total, before+3)
	}
	unit1, _, _, err := file.findUnitByReferenceID(ref1)
	if err != nil {
		t.Fatalf("moved ref1 missing: %v", err)
	}
	checkFloatField(t, unit1, "x", x1+offsetX)
	checkFloatField(t, unit1, "y", y1+offsetY)
	unit2, _, _, err := file.findUnitByReferenceID(ref2)
	if err != nil {
		t.Fatalf("moved ref2 missing: %v", err)
	}
	checkFloatField(t, unit2, "x", x2+offsetX)
	checkFloatField(t, unit2, "y", y2+offsetY)
	unit3, _, _, err := file.findUnitByReferenceID(ref3)
	if err != nil {
		t.Fatalf("outside ref3 missing: %v", err)
	}
	checkFloatField(t, unit3, "x", x3)
	checkFloatField(t, unit3, "y", y3)
}

func TestMoveUnitsInAreaTargetAnchor(t *testing.T) {
	input := testfixtures.Path(t, "save-analysis/diagnostics/Replay_Transparency_Diagnostic_v2b_flagged.aoe2scenario")
	if _, err := os.Stat(input); err != nil {
		t.Skipf("diagnostic fixture unavailable: %v", err)
	}
	file, err := Open(input)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	player := 1
	unitConst := 83
	x, y := 10.5, 11.5
	refID := 991801
	if err := file.AddUnit(UnitRecipe{Op: "add_unit", Player: player, UnitConst: unitConst, X: &x, Y: &y, ReferenceID: &refID}); err != nil {
		t.Fatalf("AddUnit: %v", err)
	}
	ax1, ay1, ax2, ay2 := 10.0, 11.0, 12.0, 13.0
	targetX, targetY := 30.0, 31.0
	recipe := UnitRecipe{
		Op:              "move_units_in_area",
		TargetPlayer:    &player,
		TargetUnitConst: &unitConst,
		TargetAreaX1:    &ax1,
		TargetAreaY1:    &ay1,
		TargetAreaX2:    &ax2,
		TargetAreaY2:    &ay2,
		TargetX:         &targetX,
		TargetY:         &targetY,
	}
	if _, err := file.MoveUnitsInArea(recipe); err != nil {
		t.Fatalf("MoveUnitsInArea target anchor: %v", err)
	}
	unit, _, _, err := file.findUnitByReferenceID(refID)
	if err != nil {
		t.Fatalf("unit missing after target move: %v", err)
	}
	checkFloatField(t, unit, "x", 30.5)
	checkFloatField(t, unit, "y", 31.5)
}

func TestMoveUnitsInAreaRejectsMissingOffsetAndOutOfBounds(t *testing.T) {
	input := testfixtures.Path(t, "save-analysis/diagnostics/Replay_Transparency_Diagnostic_v2b_flagged.aoe2scenario")
	if _, err := os.Stat(input); err != nil {
		t.Skipf("diagnostic fixture unavailable: %v", err)
	}
	file, err := Open(input)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	player := 1
	unitConst := 83
	x, y := 1.5, 1.5
	refID := 991501
	if err := file.AddUnit(UnitRecipe{Op: "add_unit", Player: player, UnitConst: unitConst, X: &x, Y: &y, ReferenceID: &refID}); err != nil {
		t.Fatalf("AddUnit: %v", err)
	}
	ax1, ay1, ax2, ay2 := 1.0, 1.0, 2.0, 2.0
	recipe := UnitRecipe{
		Op:              "move_units_in_area",
		TargetPlayer:    &player,
		TargetUnitConst: &unitConst,
		TargetAreaX1:    &ax1,
		TargetAreaY1:    &ay1,
		TargetAreaX2:    &ax2,
		TargetAreaY2:    &ay2,
	}
	if _, err := file.MoveUnitsInArea(recipe); err == nil || !strings.Contains(err.Error(), "requires offset_x/offset_y or target_x/target_y") {
		t.Fatalf("MoveUnitsInArea missing offset error = %v", err)
	}
	offsetX := -3.0
	recipe.OffsetX = &offsetX
	if _, err := file.MoveUnitsInArea(recipe); err == nil || !strings.Contains(err.Error(), "outside map") {
		t.Fatalf("MoveUnitsInArea out-of-bounds error = %v", err)
	}
	targetX := 10.0
	targetY := 10.0
	recipe.TargetX = &targetX
	recipe.TargetY = &targetY
	if _, err := file.Plan(Recipe{Units: []UnitRecipe{recipe}}); err == nil || !strings.Contains(err.Error(), "cannot combine offset_x/offset_y with target_x/target_y") {
		t.Fatalf("Plan move_units_in_area mixed target/offset error = %v", err)
	}
	unit, _, _, err := file.findUnitByReferenceID(refID)
	if err != nil {
		t.Fatalf("unit missing after rejected move: %v", err)
	}
	checkFloatField(t, unit, "x", x)
	checkFloatField(t, unit, "y", y)
}

func TestEditUnitsInArea(t *testing.T) {
	input := testfixtures.Path(t, "save-analysis/diagnostics/Replay_Transparency_Diagnostic_v2b_flagged.aoe2scenario")
	if _, err := os.Stat(input); err != nil {
		t.Skipf("diagnostic fixture unavailable: %v", err)
	}
	file, err := Open(input)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	before := file.Units.Total
	player := 1
	targetPlayer := 2
	unitConst := 83
	newUnitConst := 74
	status := 4
	rotation := 1.25
	caption := "area edited"
	x1, y1 := 23.5, 23.5
	x2, y2 := 24.5, 24.5
	x3, y3 := 32.5, 32.5
	ref1 := 991601
	ref2 := 991602
	ref3 := 991603
	if err := file.AddUnit(UnitRecipe{Op: "add_unit", Player: player, UnitConst: unitConst, X: &x1, Y: &y1, ReferenceID: &ref1}); err != nil {
		t.Fatalf("AddUnit ref1: %v", err)
	}
	if err := file.AddUnit(UnitRecipe{Op: "add_unit", Player: player, UnitConst: unitConst, X: &x2, Y: &y2, ReferenceID: &ref2}); err != nil {
		t.Fatalf("AddUnit ref2: %v", err)
	}
	if err := file.AddUnit(UnitRecipe{Op: "add_unit", Player: player, UnitConst: unitConst, X: &x3, Y: &y3, ReferenceID: &ref3}); err != nil {
		t.Fatalf("AddUnit ref3: %v", err)
	}
	ax1, ay1, ax2, ay2 := 23.0, 23.0, 25.0, 25.0
	recipe := UnitRecipe{
		Op:              "edit_units_in_area",
		TargetPlayer:    &player,
		TargetUnitConst: &unitConst,
		TargetAreaX1:    &ax1,
		TargetAreaY1:    &ay1,
		TargetAreaX2:    &ax2,
		TargetAreaY2:    &ay2,
		UnitConst:       newUnitConst,
		Status:          &status,
		Rotation:        &rotation,
		CaptionString:   caption,
		SetPlayer:       &targetPlayer,
	}
	plan, err := file.Plan(Recipe{Units: []UnitRecipe{recipe}})
	if err != nil {
		t.Fatalf("Plan edit_units_in_area: %v", err)
	}
	if plan.UnitCountAfter != before+3 {
		t.Fatalf("plan unit_count_after = %d, want %d", plan.UnitCountAfter, before+3)
	}
	edited, err := file.EditUnitsInArea(recipe)
	if err != nil {
		t.Fatalf("EditUnitsInArea: %v", err)
	}
	if edited != 2 {
		t.Fatalf("edited = %d, want 2", edited)
	}
	if file.Units.Total != before+3 {
		t.Fatalf("unit total = %d, want %d", file.Units.Total, before+3)
	}
	playerSections := file.root.section("Units").list("players_units")
	for _, refID := range []int{ref1, ref2} {
		unit, section, _, err := file.findUnitByReferenceID(refID)
		if err != nil {
			t.Fatalf("edited unit %d missing: %v", refID, err)
		}
		if section != playerSections[targetPlayer] {
			t.Fatalf("edited unit %d not moved to player %d", refID, targetPlayer)
		}
		checkIntField(t, unit, "unit_const", newUnitConst)
		checkIntField(t, unit, "status", status)
		checkFloatField(t, unit, "rotation", rotation)
		if got, _ := unit.stringValue("caption_string"); got != caption {
			t.Fatalf("caption_string = %q, want %q", got, caption)
		}
	}
	outside, section, _, err := file.findUnitByReferenceID(ref3)
	if err != nil {
		t.Fatalf("outside unit missing: %v", err)
	}
	if section != playerSections[player] {
		t.Fatalf("outside unit moved unexpectedly")
	}
	checkIntField(t, outside, "unit_const", unitConst)
}

func TestEditUnitsInAreaRejectsNoFieldsAndPositionFields(t *testing.T) {
	input := testfixtures.Path(t, "save-analysis/diagnostics/Replay_Transparency_Diagnostic_v2b_flagged.aoe2scenario")
	if _, err := os.Stat(input); err != nil {
		t.Skipf("diagnostic fixture unavailable: %v", err)
	}
	file, err := Open(input)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	player := 1
	unitConst := 83
	x, y := 26.5, 26.5
	refID := 991701
	if err := file.AddUnit(UnitRecipe{Op: "add_unit", Player: player, UnitConst: unitConst, X: &x, Y: &y, ReferenceID: &refID}); err != nil {
		t.Fatalf("AddUnit: %v", err)
	}
	ax1, ay1, ax2, ay2 := 26.0, 26.0, 27.0, 27.0
	recipe := UnitRecipe{
		Op:              "edit_units_in_area",
		TargetPlayer:    &player,
		TargetUnitConst: &unitConst,
		TargetAreaX1:    &ax1,
		TargetAreaY1:    &ay1,
		TargetAreaX2:    &ax2,
		TargetAreaY2:    &ay2,
	}
	if _, err := file.EditUnitsInArea(recipe); err == nil || !strings.Contains(err.Error(), "requires unit_const") {
		t.Fatalf("EditUnitsInArea no-fields error = %v", err)
	}
	recipe.X = &x
	if _, err := file.EditUnitsInArea(recipe); err == nil || !strings.Contains(err.Error(), "does not edit positions") {
		t.Fatalf("EditUnitsInArea position error = %v", err)
	}
	unit, _, _, err := file.findUnitByReferenceID(refID)
	if err != nil {
		t.Fatalf("unit missing after rejected edit: %v", err)
	}
	checkFloatField(t, unit, "x", x)
	checkFloatField(t, unit, "y", y)
	checkIntField(t, unit, "unit_const", unitConst)
}

func TestCopyUnitsInArea(t *testing.T) {
	input := testfixtures.Path(t, "save-analysis/diagnostics/Replay_Transparency_Diagnostic_v2b_flagged.aoe2scenario")
	if _, err := os.Stat(input); err != nil {
		t.Skipf("diagnostic fixture unavailable: %v", err)
	}
	file, err := Open(input)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	before := file.Units.Total
	player := 1
	targetPlayer := 2
	unitConst := 83
	x1, y1 := 40.5, 40.5
	x2, y2 := 41.5, 41.5
	ref1 := 991201
	ref2 := 991202
	if err := file.AddUnit(UnitRecipe{Op: "add_unit", Player: player, UnitConst: unitConst, X: &x1, Y: &y1, ReferenceID: &ref1, CaptionString: "CopyMeA"}); err != nil {
		t.Fatalf("AddUnit ref1: %v", err)
	}
	if err := file.AddUnit(UnitRecipe{Op: "add_unit", Player: player, UnitConst: unitConst, X: &x2, Y: &y2, ReferenceID: &ref2, GarrisonedInID: &ref1, CaptionString: "CopyMeB"}); err != nil {
		t.Fatalf("AddUnit ref2: %v", err)
	}
	ax1, ay1, ax2, ay2 := 40.0, 40.0, 42.0, 42.0
	offsetX, offsetY := 10.0, 5.0
	base := 992200
	recipe := UnitRecipe{
		Op:              "copy_units_in_area",
		TargetPlayer:    &player,
		TargetUnitConst: &unitConst,
		TargetAreaX1:    &ax1,
		TargetAreaY1:    &ay1,
		TargetAreaX2:    &ax2,
		TargetAreaY2:    &ay2,
		OffsetX:         &offsetX,
		OffsetY:         &offsetY,
		ReferenceIDBase: &base,
		SetPlayer:       &targetPlayer,
		CaptionSuffix:   " copied",
	}
	plan, err := file.Plan(Recipe{Units: []UnitRecipe{recipe}})
	if err != nil {
		t.Fatalf("Plan copy_units_in_area: %v", err)
	}
	if plan.UnitCountAfter != before+4 {
		t.Fatalf("plan unit_count_after = %d, want %d", plan.UnitCountAfter, before+4)
	}
	copied, err := file.CopyUnitsInArea(recipe)
	if err != nil {
		t.Fatalf("CopyUnitsInArea: %v", err)
	}
	if copied != 2 {
		t.Fatalf("copied = %d, want 2", copied)
	}
	if file.Units.Total != before+4 {
		t.Fatalf("unit total = %d, want %d", file.Units.Total, before+4)
	}
	copyA, _, _, err := file.findUnitByReferenceID(base)
	if err != nil {
		t.Fatalf("copy A missing: %v", err)
	}
	copyB, _, _, err := file.findUnitByReferenceID(base + 1)
	if err != nil {
		t.Fatalf("copy B missing: %v", err)
	}
	gotX, _ := copyA.floatValue("x")
	gotY, _ := copyA.floatValue("y")
	if gotX != x1+offsetX || gotY != y1+offsetY {
		t.Fatalf("copy A position = %.1f,%.1f want %.1f,%.1f", gotX, gotY, x1+offsetX, y1+offsetY)
	}
	caption, _ := copyB.stringValue("caption_string")
	if caption != "CopyMeB copied" {
		t.Fatalf("copy B caption = %q", caption)
	}
	garrisonedIn, _ := copyB.intValue("garrisoned_in_id")
	if garrisonedIn != base {
		t.Fatalf("copy B garrisoned_in_id = %d, want copied container %d", garrisonedIn, base)
	}
	if _, _, _, err := file.findUnitByReferenceID(ref1); err != nil {
		t.Fatalf("original ref1 missing after copy: %v", err)
	}
	if _, _, _, err := file.findUnitByReferenceID(ref2); err != nil {
		t.Fatalf("original ref2 missing after copy: %v", err)
	}
}

func TestCopyUnitsInAreaTargetAnchor(t *testing.T) {
	input := testfixtures.Path(t, "save-analysis/diagnostics/Replay_Transparency_Diagnostic_v2b_flagged.aoe2scenario")
	if _, err := os.Stat(input); err != nil {
		t.Skipf("diagnostic fixture unavailable: %v", err)
	}
	file, err := Open(input)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	player := 1
	unitConst := 83
	x, y := 12.5, 13.5
	refID := 991901
	if err := file.AddUnit(UnitRecipe{Op: "add_unit", Player: player, UnitConst: unitConst, X: &x, Y: &y, ReferenceID: &refID}); err != nil {
		t.Fatalf("AddUnit: %v", err)
	}
	ax1, ay1, ax2, ay2 := 12.0, 13.0, 13.0, 14.0
	targetX, targetY := 30.0, 31.0
	base := 992900
	recipe := UnitRecipe{
		Op:              "copy_units_in_area",
		TargetPlayer:    &player,
		TargetUnitConst: &unitConst,
		TargetAreaX1:    &ax1,
		TargetAreaY1:    &ay1,
		TargetAreaX2:    &ax2,
		TargetAreaY2:    &ay2,
		TargetX:         &targetX,
		TargetY:         &targetY,
		ReferenceIDBase: &base,
	}
	if _, err := file.CopyUnitsInArea(recipe); err != nil {
		t.Fatalf("CopyUnitsInArea target anchor: %v", err)
	}
	copy, _, _, err := file.findUnitByReferenceID(base)
	if err != nil {
		t.Fatalf("copy missing: %v", err)
	}
	checkFloatField(t, copy, "x", 30.5)
	checkFloatField(t, copy, "y", 31.5)
	offsetX := 1.0
	recipe.OffsetX = &offsetX
	if _, err := file.Plan(Recipe{Units: []UnitRecipe{recipe}}); err == nil || !strings.Contains(err.Error(), "cannot combine offset_x/offset_y with target_x/target_y") {
		t.Fatalf("Plan copy_units_in_area mixed target/offset error = %v", err)
	}
}

func TestCopyUnitsInAreaRejectsDuplicateReferenceBase(t *testing.T) {
	input := testfixtures.Path(t, "save-analysis/diagnostics/Replay_Transparency_Diagnostic_v2b_flagged.aoe2scenario")
	if _, err := os.Stat(input); err != nil {
		t.Skipf("diagnostic fixture unavailable: %v", err)
	}
	file, err := Open(input)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	player := 1
	unitConst := 83
	x, y := 46.5, 46.5
	ref := 991301
	if err := file.AddUnit(UnitRecipe{Op: "add_unit", Player: player, UnitConst: unitConst, X: &x, Y: &y, ReferenceID: &ref}); err != nil {
		t.Fatalf("AddUnit: %v", err)
	}
	ax1, ay1, ax2, ay2 := 46.0, 46.0, 47.0, 47.0
	recipe := UnitRecipe{
		Op:              "copy_units_in_area",
		TargetPlayer:    &player,
		TargetUnitConst: &unitConst,
		TargetAreaX1:    &ax1,
		TargetAreaY1:    &ay1,
		TargetAreaX2:    &ax2,
		TargetAreaY2:    &ay2,
		ReferenceIDBase: &ref,
	}
	if _, err := file.Plan(Recipe{Units: []UnitRecipe{recipe}}); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("Plan duplicate copy ref error = %v, want already exists", err)
	}
	if _, err := file.CopyUnitsInArea(recipe); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("CopyUnitsInArea duplicate ref error = %v, want already exists", err)
	}
}

func TestScenarioReferencesReportsKnownDependencyClasses(t *testing.T) {
	input := testfixtures.Path(t, "save-analysis/diagnostics/Replay_Transparency_Diagnostic_v2b_flagged.aoe2scenario")
	if _, err := os.Stat(input); err != nil {
		t.Skipf("diagnostic fixture unavailable: %v", err)
	}
	file, err := Open(input)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	unitRef := 990601
	x := 17.5
	y := 17.5
	if err := file.AddUnit(UnitRecipe{Op: "add_unit", Player: 1, UnitConst: 83, X: &x, Y: &y, ReferenceID: &unitRef}); err != nil {
		t.Fatalf("AddUnit: %v", err)
	}
	targetTrigger := 0
	variable := 0
	stringID := 0
	if err := file.AddTrigger(TriggerRecipe{
		Op:   "add_trigger",
		Name: "references many things",
		Conditions: []ConditionRecipe{
			{Op: "object_selected", UnitObject: &unitRef},
			{Op: "variable_value", Variable: &variable, Comparison: intPtr(0)},
		},
		Effects: []EffectRecipe{
			{Op: "activate_trigger", TriggerID: &targetTrigger},
			{Op: "display_instructions", Message: "uses inline text"},
		},
	}); err != nil {
		t.Fatalf("AddTrigger: %v", err)
	}
	trigger := file.root.section("Triggers").list("trigger_data")
	added := trigger[len(trigger)-1]
	if err := setIntField(added.list("effect_data")[1], "string_id", "s32", stringID); err != nil {
		t.Fatalf("set string_id: %v", err)
	}
	report, err := file.References(ReferenceOptions{})
	if err != nil {
		t.Fatalf("References: %v", err)
	}
	if report.Summary.ByKind["unit"] == 0 || report.Summary.ByKind["trigger"] == 0 || report.Summary.ByKind["variable"] == 0 || report.Summary.ByKind["string"] == 0 {
		t.Fatalf("reference summary missing kinds: %+v", report.Summary.ByKind)
	}
	unitOnly, err := file.References(ReferenceOptions{Kind: "unit", TargetID: &unitRef})
	if err != nil {
		t.Fatalf("References filtered: %v", err)
	}
	if len(unitOnly.References) != 1 || unitOnly.References[0].Field != "unit_object" {
		t.Fatalf("filtered unit refs = %+v, want unit_object only", unitOnly.References)
	}
}

func TestRemoveUnitRejectsGarrisonReferences(t *testing.T) {
	input := testfixtures.Path(t, "save-analysis/diagnostics/Replay_Transparency_Diagnostic_v2b_flagged.aoe2scenario")
	if _, err := os.Stat(input); err != nil {
		t.Skipf("diagnostic fixture unavailable: %v", err)
	}
	file, err := Open(input)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	containerRef := 990701
	garrisonRef := 990702
	x := 18.5
	y := 18.5
	if err := file.AddUnit(UnitRecipe{Op: "add_unit", Player: 1, UnitConst: 82, X: &x, Y: &y, ReferenceID: &containerRef}); err != nil {
		t.Fatalf("AddUnit container: %v", err)
	}
	if err := file.AddUnit(UnitRecipe{Op: "add_unit", Player: 1, UnitConst: 83, X: &x, Y: &y, ReferenceID: &garrisonRef, GarrisonedInID: &containerRef}); err != nil {
		t.Fatalf("AddUnit garrisoned: %v", err)
	}
	err = file.RemoveUnit(UnitRecipe{Op: "remove_unit", ReferenceID: &containerRef})
	if err == nil || !strings.Contains(err.Error(), "garrisoned_in_id references it") {
		t.Fatalf("RemoveUnit error = %v, want garrisoned_in_id refusal", err)
	}
}

func TestScenarioDeletePlanUnitBlockedAndClear(t *testing.T) {
	input := testfixtures.Path(t, "save-analysis/diagnostics/Replay_Transparency_Diagnostic_v2b_flagged.aoe2scenario")
	if _, err := os.Stat(input); err != nil {
		t.Skipf("diagnostic fixture unavailable: %v", err)
	}
	file, err := Open(input)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	blockedRef := 990801
	clearRef := 990802
	blockedCaption := "delete-plan-caption-blocked"
	clearCaption := "delete-plan-caption-clear"
	x := 19.5
	y := 19.5
	if err := file.AddUnit(UnitRecipe{Op: "add_unit", Player: 1, UnitConst: 83, X: &x, Y: &y, ReferenceID: &blockedRef, CaptionString: blockedCaption}); err != nil {
		t.Fatalf("AddUnit blocked: %v", err)
	}
	if err := file.AddUnit(UnitRecipe{Op: "add_unit", Player: 1, UnitConst: 83, X: &x, Y: &y, ReferenceID: &clearRef, CaptionString: clearCaption}); err != nil {
		t.Fatalf("AddUnit clear: %v", err)
	}
	if err := file.AddTrigger(TriggerRecipe{
		Op:   "add_trigger",
		Name: "blocks unit delete",
		Effects: []EffectRecipe{
			{Op: "kill_object", SourcePlayer: intPtr(1), SelectedObjectIDs: []int{blockedRef}},
		},
	}); err != nil {
		t.Fatalf("AddTrigger: %v", err)
	}
	blocked, err := file.DeletePlan(DeletePlanRequest{Kind: "unit", ID: blockedRef})
	if err != nil {
		t.Fatalf("DeletePlan blocked: %v", err)
	}
	if blocked.CanDelete || len(blocked.BlockingRefs) != 1 || !strings.Contains(blocked.Strategy, "blocked") {
		t.Fatalf("blocked delete plan = %+v, want one blocking ref", blocked)
	}
	if blocked.CleanupRecipe == nil || blocked.CleanupCommand == "" {
		t.Fatalf("blocked delete plan = %+v, want unit cleanup recipe/command", blocked)
	}
	blockedByCaption, err := file.DeletePlan(DeletePlanRequest{Kind: "unit-caption", TargetCaption: blockedCaption, Player: intPtr(1)})
	if err != nil {
		t.Fatalf("DeletePlan blocked by caption: %v", err)
	}
	if blockedByCaption.CanDelete || len(blockedByCaption.BlockingRefs) != 1 || blockedByCaption.CleanupRecipe == nil || !strings.Contains(blockedByCaption.CleanupCommand, fmt.Sprintf("unit %d", blockedRef)) {
		t.Fatalf("blocked caption delete plan = %+v, want reference-id cleanup", blockedByCaption)
	}
	_, disconnectRecipe, err := file.UnitDisconnectRecipe(blockedRef)
	if err != nil {
		t.Fatalf("UnitDisconnectRecipe: %v", err)
	}
	if len(disconnectRecipe.Triggers) != 1 {
		t.Fatalf("unit disconnect trigger edits = %+v, want one edit", disconnectRecipe.Triggers)
	}
	clear, err := file.DeletePlan(DeletePlanRequest{Kind: "unit", ID: clearRef})
	if err != nil {
		t.Fatalf("DeletePlan clear: %v", err)
	}
	if !clear.CanDelete || clear.SuggestedRecipe == nil {
		t.Fatalf("clear delete plan = %+v, want suggested recipe", clear)
	}
	clearByCaption, err := file.DeletePlan(DeletePlanRequest{Kind: "unit_caption", TargetCaption: clearCaption, Player: intPtr(1)})
	if err != nil {
		t.Fatalf("DeletePlan clear by caption: %v", err)
	}
	if !clearByCaption.CanDelete || clearByCaption.SuggestedRecipe == nil || !strings.Contains(clearByCaption.Strategy, "caption selector") {
		t.Fatalf("clear caption delete plan = %+v, want caption recipe", clearByCaption)
	}
	captionData, err := json.Marshal(clearByCaption.SuggestedRecipe)
	if err != nil {
		t.Fatalf("marshal caption delete recipe: %v", err)
	}
	if !strings.Contains(string(captionData), "target_caption") || !strings.Contains(string(captionData), clearCaption) {
		t.Fatalf("caption suggested recipe = %s, want target_caption", captionData)
	}
	prefixBlockedRef := 990805
	prefixClearRef1 := 990806
	prefixClearRef2 := 990807
	prefixX1, prefixY1 := 24.5, 24.5
	prefixX2, prefixY2 := 25.5, 25.5
	prefixBlockedCaption := "delete-plan-prefix-blocked alpha"
	prefixClearCaption := "delete-plan-prefix-clear "
	if err := file.AddUnit(UnitRecipe{Op: "add_unit", Player: 1, UnitConst: 83, X: &prefixX1, Y: &prefixY1, ReferenceID: &prefixBlockedRef, CaptionString: prefixBlockedCaption}); err != nil {
		t.Fatalf("AddUnit prefix blocked: %v", err)
	}
	if err := file.AddUnit(UnitRecipe{Op: "add_unit", Player: 1, UnitConst: 83, X: &prefixX1, Y: &prefixY1, ReferenceID: &prefixClearRef1, CaptionString: prefixClearCaption + "alpha"}); err != nil {
		t.Fatalf("AddUnit prefix clear 1: %v", err)
	}
	if err := file.AddUnit(UnitRecipe{Op: "add_unit", Player: 1, UnitConst: 83, X: &prefixX2, Y: &prefixY2, ReferenceID: &prefixClearRef2, CaptionString: prefixClearCaption + "beta"}); err != nil {
		t.Fatalf("AddUnit prefix clear 2: %v", err)
	}
	if err := file.AddTrigger(TriggerRecipe{
		Op:   "add_trigger",
		Name: "blocks caption prefix unit delete",
		Effects: []EffectRecipe{
			{Op: "kill_object", SourcePlayer: intPtr(1), SelectedObjectIDs: []int{prefixBlockedRef}},
		},
	}); err != nil {
		t.Fatalf("AddTrigger prefix blocker: %v", err)
	}
	prefixBlocked, err := file.DeletePlan(DeletePlanRequest{Kind: "unit-caption-prefix", TargetPrefix: "delete-plan-prefix-blocked", Player: intPtr(1)})
	if err != nil {
		t.Fatalf("DeletePlan blocked unit-caption-prefix: %v", err)
	}
	if prefixBlocked.CanDelete || len(prefixBlocked.ResolvedIDs) != 1 || prefixBlocked.ResolvedIDs[0] != prefixBlockedRef ||
		len(prefixBlocked.BlockingRefs) != 1 || prefixBlocked.CleanupRecipe == nil ||
		!strings.Contains(prefixBlocked.CleanupCommand, "unit-caption-prefix") {
		t.Fatalf("blocked unit-caption-prefix plan = %+v, want resolved id cleanup", prefixBlocked)
	}
	resolvedPrefixRefs, prefixRefs, prefixRecipe, err := file.UnitCaptionPrefixDisconnectRecipe("delete-plan-prefix-blocked", intPtr(1))
	if err != nil {
		t.Fatalf("UnitCaptionPrefixDisconnectRecipe: %v", err)
	}
	if len(resolvedPrefixRefs) != 1 || resolvedPrefixRefs[0] != prefixBlockedRef ||
		prefixRefs.Summary.Total != 1 || len(prefixRecipe.Triggers) != 1 {
		t.Fatalf("unit-caption-prefix disconnect resolved=%v refs=%+v recipe=%+v, want one trigger cleanup", resolvedPrefixRefs, prefixRefs.Summary, prefixRecipe)
	}
	prefixClear, err := file.DeletePlan(DeletePlanRequest{Kind: "unit-caption-prefix", TargetPrefix: prefixClearCaption, Player: intPtr(1)})
	if err != nil {
		t.Fatalf("DeletePlan clear unit-caption-prefix: %v", err)
	}
	if !prefixClear.CanDelete || prefixClear.SuggestedRecipe == nil || len(prefixClear.ResolvedIDs) != 2 ||
		prefixClear.ResolvedIDs[0] != prefixClearRef1 || prefixClear.ResolvedIDs[1] != prefixClearRef2 {
		t.Fatalf("clear unit-caption-prefix plan = %+v, want two resolved reference-id removes", prefixClear)
	}
	prefixData, err := json.Marshal(prefixClear.SuggestedRecipe)
	if err != nil {
		t.Fatalf("marshal prefix delete recipe: %v", err)
	}
	if !strings.Contains(string(prefixData), `"reference_id":`+strconv.Itoa(prefixClearRef1)) ||
		!strings.Contains(string(prefixData), `"reference_id":`+strconv.Itoa(prefixClearRef2)) {
		t.Fatalf("unit-caption-prefix suggested recipe = %s, want reference_id removes", prefixData)
	}
	containsBlockedRef := 990811
	containsClearRef1 := 990812
	containsClearRef2 := 990813
	containsX1, containsY1 := 26.5, 24.5
	containsX2, containsY2 := 26.5, 25.5
	if err := file.AddUnit(UnitRecipe{Op: "add_unit", Player: 1, UnitConst: 83, X: &containsX1, Y: &containsY1, ReferenceID: &containsBlockedRef, CaptionString: "blocked middle CAPMARK alpha"}); err != nil {
		t.Fatalf("AddUnit contains blocked: %v", err)
	}
	if err := file.AddUnit(UnitRecipe{Op: "add_unit", Player: 1, UnitConst: 83, X: &containsX1, Y: &containsY1, ReferenceID: &containsClearRef1, CaptionString: "clear middle CAPMARK alpha"}); err != nil {
		t.Fatalf("AddUnit contains clear 1: %v", err)
	}
	if err := file.AddUnit(UnitRecipe{Op: "add_unit", Player: 1, UnitConst: 83, X: &containsX2, Y: &containsY2, ReferenceID: &containsClearRef2, CaptionString: "clear middle CAPMARK beta"}); err != nil {
		t.Fatalf("AddUnit contains clear 2: %v", err)
	}
	if err := file.AddTrigger(TriggerRecipe{
		Op:   "add_trigger",
		Name: "blocks caption contains unit delete",
		Effects: []EffectRecipe{
			{Op: "kill_object", SourcePlayer: intPtr(1), SelectedObjectIDs: []int{containsBlockedRef}},
		},
	}); err != nil {
		t.Fatalf("AddTrigger contains blocker: %v", err)
	}
	containsBlocked, err := file.DeletePlan(DeletePlanRequest{Kind: "unit-caption-contains", TargetText: "blocked middle CAPMARK", Player: intPtr(1)})
	if err != nil {
		t.Fatalf("DeletePlan blocked unit-caption-contains: %v", err)
	}
	if containsBlocked.CanDelete || len(containsBlocked.ResolvedIDs) != 1 || containsBlocked.ResolvedIDs[0] != containsBlockedRef ||
		len(containsBlocked.BlockingRefs) != 1 || containsBlocked.CleanupRecipe == nil ||
		!strings.Contains(containsBlocked.CleanupCommand, "unit-caption-contains") {
		t.Fatalf("blocked unit-caption-contains plan = %+v, want resolved id cleanup", containsBlocked)
	}
	resolvedContainsRefs, containsRefs, containsRecipe, err := file.UnitCaptionContainsDisconnectRecipe("blocked middle CAPMARK", intPtr(1))
	if err != nil {
		t.Fatalf("UnitCaptionContainsDisconnectRecipe: %v", err)
	}
	if len(resolvedContainsRefs) != 1 || resolvedContainsRefs[0] != containsBlockedRef ||
		containsRefs.Summary.Total != 1 || len(containsRecipe.Triggers) != 1 {
		t.Fatalf("unit-caption-contains disconnect resolved=%v refs=%+v recipe=%+v, want one trigger cleanup", resolvedContainsRefs, containsRefs.Summary, containsRecipe)
	}
	containsClear, err := file.DeletePlan(DeletePlanRequest{Kind: "unit-caption-contains", TargetText: "clear middle CAPMARK", Player: intPtr(1)})
	if err != nil {
		t.Fatalf("DeletePlan clear unit-caption-contains: %v", err)
	}
	if !containsClear.CanDelete || containsClear.SuggestedRecipe == nil || len(containsClear.ResolvedIDs) != 2 ||
		containsClear.ResolvedIDs[0] != containsClearRef1 || containsClear.ResolvedIDs[1] != containsClearRef2 {
		t.Fatalf("clear unit-caption-contains plan = %+v, want two resolved reference-id removes", containsClear)
	}
	containsData, err := json.Marshal(containsClear.SuggestedRecipe)
	if err != nil {
		t.Fatalf("marshal contains delete recipe: %v", err)
	}
	if !strings.Contains(string(containsData), `"reference_id":`+strconv.Itoa(containsClearRef1)) ||
		!strings.Contains(string(containsData), `"reference_id":`+strconv.Itoa(containsClearRef2)) {
		t.Fatalf("unit-caption-contains suggested recipe = %s, want reference_id removes", containsData)
	}
	typeBlockedRef := 990808
	typeClearRef1 := 990809
	typeClearRef2 := 990810
	typeBlockedUnitConst := 997
	typeClearUnitConst := 998
	typeX1, typeY1 := 27.5, 27.5
	typeX2, typeY2 := 28.5, 28.5
	if err := file.AddUnit(UnitRecipe{Op: "add_unit", Player: 1, UnitConst: typeBlockedUnitConst, X: &typeX1, Y: &typeY1, ReferenceID: &typeBlockedRef}); err != nil {
		t.Fatalf("AddUnit type blocked: %v", err)
	}
	if err := file.AddUnit(UnitRecipe{Op: "add_unit", Player: 1, UnitConst: typeClearUnitConst, X: &typeX1, Y: &typeY1, ReferenceID: &typeClearRef1}); err != nil {
		t.Fatalf("AddUnit type clear 1: %v", err)
	}
	if err := file.AddUnit(UnitRecipe{Op: "add_unit", Player: 1, UnitConst: typeClearUnitConst, X: &typeX2, Y: &typeY2, ReferenceID: &typeClearRef2}); err != nil {
		t.Fatalf("AddUnit type clear 2: %v", err)
	}
	if err := file.AddTrigger(TriggerRecipe{
		Op:   "add_trigger",
		Name: "blocks unit-type delete",
		Effects: []EffectRecipe{
			{Op: "kill_object", SourcePlayer: intPtr(1), SelectedObjectIDs: []int{typeBlockedRef}},
		},
	}); err != nil {
		t.Fatalf("AddTrigger type blocker: %v", err)
	}
	typeBlocked, err := file.DeletePlan(DeletePlanRequest{Kind: "unit-type", UnitConst: &typeBlockedUnitConst, Player: intPtr(1)})
	if err != nil {
		t.Fatalf("DeletePlan blocked unit-type: %v", err)
	}
	if typeBlocked.CanDelete || len(typeBlocked.ResolvedIDs) != 1 || typeBlocked.ResolvedIDs[0] != typeBlockedRef ||
		len(typeBlocked.BlockingRefs) != 1 || typeBlocked.CleanupRecipe == nil ||
		!strings.Contains(typeBlocked.CleanupCommand, "unit-type") {
		t.Fatalf("blocked unit-type plan = %+v, want resolved id cleanup", typeBlocked)
	}
	resolvedTypeRefs, typeRefs, typeRecipe, err := file.UnitTypeDisconnectRecipe(typeBlockedUnitConst, intPtr(1))
	if err != nil {
		t.Fatalf("UnitTypeDisconnectRecipe: %v", err)
	}
	if len(resolvedTypeRefs) != 1 || resolvedTypeRefs[0] != typeBlockedRef ||
		typeRefs.Summary.Total != 1 || len(typeRecipe.Triggers) != 1 {
		t.Fatalf("unit-type disconnect resolved=%v refs=%+v recipe=%+v, want one trigger cleanup", resolvedTypeRefs, typeRefs.Summary, typeRecipe)
	}
	typeClear, err := file.DeletePlan(DeletePlanRequest{Kind: "unit-type", UnitConst: &typeClearUnitConst, Player: intPtr(1)})
	if err != nil {
		t.Fatalf("DeletePlan clear unit-type: %v", err)
	}
	if !typeClear.CanDelete || typeClear.SuggestedRecipe == nil || len(typeClear.ResolvedIDs) != 2 ||
		typeClear.ResolvedIDs[0] != typeClearRef1 || typeClear.ResolvedIDs[1] != typeClearRef2 {
		t.Fatalf("clear unit-type plan = %+v, want two resolved reference-id removes", typeClear)
	}
	typeData, err := json.Marshal(typeClear.SuggestedRecipe)
	if err != nil {
		t.Fatalf("marshal unit-type delete recipe: %v", err)
	}
	if !strings.Contains(string(typeData), `"reference_id":`+strconv.Itoa(typeClearRef1)) ||
		!strings.Contains(string(typeData), `"reference_id":`+strconv.Itoa(typeClearRef2)) {
		t.Fatalf("unit-type suggested recipe = %s, want reference_id removes", typeData)
	}
	playerSections, err := file.unitPlayerSections()
	if err != nil {
		t.Fatalf("unitPlayerSections: %v", err)
	}
	var emptyPlayers []int
	for player, section := range playerSections {
		if player == 0 {
			continue
		}
		if len(section.list("units")) == 0 {
			emptyPlayers = append(emptyPlayers, player)
		}
	}
	if len(emptyPlayers) < 2 {
		t.Skipf("need two empty player slots for units-player test, found %v", emptyPlayers)
	}
	playerBlocked := emptyPlayers[0]
	playerClear := emptyPlayers[1]
	playerBlockedRef := 990831
	playerClearRef1 := 990832
	playerClearRef2 := 990833
	playerUnitConst := 999
	playerX1, playerY1 := 30.5, 30.5
	playerX2, playerY2 := 31.5, 31.5
	if err := file.AddUnit(UnitRecipe{Op: "add_unit", Player: playerBlocked, UnitConst: playerUnitConst, X: &playerX1, Y: &playerY1, ReferenceID: &playerBlockedRef}); err != nil {
		t.Fatalf("AddUnit player blocked: %v", err)
	}
	if err := file.AddUnit(UnitRecipe{Op: "add_unit", Player: playerClear, UnitConst: playerUnitConst, X: &playerX1, Y: &playerY1, ReferenceID: &playerClearRef1}); err != nil {
		t.Fatalf("AddUnit player clear 1: %v", err)
	}
	if err := file.AddUnit(UnitRecipe{Op: "add_unit", Player: playerClear, UnitConst: playerUnitConst, X: &playerX2, Y: &playerY2, ReferenceID: &playerClearRef2}); err != nil {
		t.Fatalf("AddUnit player clear 2: %v", err)
	}
	if err := file.AddTrigger(TriggerRecipe{
		Op:   "add_trigger",
		Name: "blocks units-player delete",
		Effects: []EffectRecipe{
			{Op: "kill_object", SourcePlayer: intPtr(1), SelectedObjectIDs: []int{playerBlockedRef}},
		},
	}); err != nil {
		t.Fatalf("AddTrigger units-player blocker: %v", err)
	}
	playerBlockedPlan, err := file.DeletePlan(DeletePlanRequest{Kind: "units-player", Player: &playerBlocked})
	if err != nil {
		t.Fatalf("DeletePlan blocked units-player: %v", err)
	}
	if playerBlockedPlan.CanDelete || len(playerBlockedPlan.ResolvedIDs) != 1 || playerBlockedPlan.ResolvedIDs[0] != playerBlockedRef ||
		len(playerBlockedPlan.BlockingRefs) != 1 || playerBlockedPlan.CleanupRecipe == nil ||
		!strings.Contains(playerBlockedPlan.CleanupCommand, "units-player") {
		t.Fatalf("blocked units-player plan = %+v, want resolved id cleanup", playerBlockedPlan)
	}
	resolvedPlayerRefs, playerRefs, playerRecipe, err := file.UnitsPlayerDisconnectRecipe(playerBlocked)
	if err != nil {
		t.Fatalf("UnitsPlayerDisconnectRecipe: %v", err)
	}
	if len(resolvedPlayerRefs) != 1 || resolvedPlayerRefs[0] != playerBlockedRef ||
		playerRefs.Summary.Total != 1 || len(playerRecipe.Triggers) != 1 {
		t.Fatalf("units-player disconnect resolved=%v refs=%+v recipe=%+v, want one trigger cleanup", resolvedPlayerRefs, playerRefs.Summary, playerRecipe)
	}
	playerClearPlan, err := file.DeletePlan(DeletePlanRequest{Kind: "units-player", Player: &playerClear})
	if err != nil {
		t.Fatalf("DeletePlan clear units-player: %v", err)
	}
	if !playerClearPlan.CanDelete || playerClearPlan.SuggestedRecipe == nil || len(playerClearPlan.ResolvedIDs) != 2 ||
		playerClearPlan.ResolvedIDs[0] != playerClearRef1 || playerClearPlan.ResolvedIDs[1] != playerClearRef2 {
		t.Fatalf("clear units-player plan = %+v, want two resolved reference ids", playerClearPlan)
	}
	playerData, err := json.Marshal(playerClearPlan.SuggestedRecipe)
	if err != nil {
		t.Fatalf("marshal units-player delete recipe: %v", err)
	}
	if !strings.Contains(string(playerData), `"remove_units_for_player"`) ||
		!strings.Contains(string(playerData), `"target_player":`+strconv.Itoa(playerClear)) {
		t.Fatalf("units-player suggested recipe = %s, want remove_units_for_player", playerData)
	}
	removed, err := file.RemoveUnitsForPlayer(UnitRecipe{Op: "remove_units_for_player", TargetPlayer: &playerClear})
	if err != nil {
		t.Fatalf("RemoveUnitsForPlayer: %v", err)
	}
	if removed != 2 {
		t.Fatalf("RemoveUnitsForPlayer removed %d, want 2", removed)
	}
	if matches, err := file.findUnitsByPlayer(&playerClear); err == nil {
		t.Fatalf("player %d still has units after RemoveUnitsForPlayer: %+v", playerClear, matches)
	}
	if _, err := file.DeletePlan(DeletePlanRequest{Kind: "unit", ID: 999999}); err == nil {
		t.Fatal("DeletePlan accepted nonexistent unit")
	}
	areaBlockedX1, areaBlockedY1, areaBlockedX2, areaBlockedY2 := 19.0, 19.0, 20.0, 20.0
	areaBlocked, err := file.DeletePlan(DeletePlanRequest{
		Kind:      "units-area",
		AreaX1:    &areaBlockedX1,
		AreaY1:    &areaBlockedY1,
		AreaX2:    &areaBlockedX2,
		AreaY2:    &areaBlockedY2,
		Player:    intPtr(1),
		UnitConst: intPtr(83),
	})
	if err != nil {
		t.Fatalf("DeletePlan blocked area: %v", err)
	}
	if areaBlocked.CanDelete || len(areaBlocked.BlockingRefs) != 1 || !strings.Contains(areaBlocked.Strategy, "area selector matched 2 placed unit") {
		t.Fatalf("blocked area delete plan = %+v, want one blocking ref across two matches", areaBlocked)
	}
	if areaBlocked.CleanupRecipe == nil || areaBlocked.CleanupCommand == "" || !strings.Contains(areaBlocked.CleanupCommand, "units-area") {
		t.Fatalf("blocked area cleanup command=%q recipe=%+v, want units-area cleanup", areaBlocked.CleanupCommand, areaBlocked.CleanupRecipe)
	}
	areaRef1 := 990803
	areaRef2 := 990804
	areaX1, areaY1 := 21.5, 21.5
	areaX2, areaY2 := 22.5, 22.5
	if err := file.AddUnit(UnitRecipe{Op: "add_unit", Player: 1, UnitConst: 83, X: &areaX1, Y: &areaY1, ReferenceID: &areaRef1}); err != nil {
		t.Fatalf("AddUnit areaRef1: %v", err)
	}
	if err := file.AddUnit(UnitRecipe{Op: "add_unit", Player: 1, UnitConst: 83, X: &areaX2, Y: &areaY2, ReferenceID: &areaRef2}); err != nil {
		t.Fatalf("AddUnit areaRef2: %v", err)
	}
	areaClearX1, areaClearY1, areaClearX2, areaClearY2 := 21.0, 21.0, 23.0, 23.0
	areaClear, err := file.DeletePlan(DeletePlanRequest{
		Kind:      "units_area",
		AreaX1:    &areaClearX1,
		AreaY1:    &areaClearY1,
		AreaX2:    &areaClearX2,
		AreaY2:    &areaClearY2,
		Player:    intPtr(1),
		UnitConst: intPtr(83),
	})
	if err != nil {
		t.Fatalf("DeletePlan clear area: %v", err)
	}
	if !areaClear.CanDelete || areaClear.SuggestedRecipe == nil || !strings.Contains(areaClear.Strategy, "remove 2 placed unit") {
		t.Fatalf("clear area delete plan = %+v, want area recipe", areaClear)
	}
	data, err := json.Marshal(areaClear.SuggestedRecipe)
	if err != nil {
		t.Fatalf("marshal area delete recipe: %v", err)
	}
	if !strings.Contains(string(data), "remove_units_in_area") || !strings.Contains(string(data), "target_area_x1") {
		t.Fatalf("area suggested recipe = %s, want remove_units_in_area bounds", data)
	}
}

func TestScenarioUnitDisconnectClearsTriggerAndGarrisonReferences(t *testing.T) {
	input := testfixtures.Path(t, "save-analysis/diagnostics/Replay_Transparency_Diagnostic_v2b_flagged.aoe2scenario")
	if _, err := os.Stat(input); err != nil {
		t.Skipf("diagnostic fixture unavailable: %v", err)
	}
	file, err := Open(input)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	containerRef := 990811
	garrisonRef := 990812
	containerCaption := "unit-disconnect-caption-container"
	x := 19.5
	y := 19.5
	if err := file.AddUnit(UnitRecipe{Op: "add_unit", Player: 1, UnitConst: 82, X: &x, Y: &y, ReferenceID: &containerRef, CaptionString: containerCaption}); err != nil {
		t.Fatalf("AddUnit container: %v", err)
	}
	if err := file.AddUnit(UnitRecipe{Op: "add_unit", Player: 1, UnitConst: 83, X: &x, Y: &y, ReferenceID: &garrisonRef, GarrisonedInID: &containerRef}); err != nil {
		t.Fatalf("AddUnit garrisoned: %v", err)
	}
	if err := file.AddTrigger(TriggerRecipe{
		Op:   "add_trigger",
		Name: "unit disconnect source",
		Effects: []EffectRecipe{
			{Op: "kill_object", SourcePlayer: intPtr(1), SelectedObjectIDs: []int{containerRef}},
		},
		Conditions: []ConditionRecipe{
			{Op: "object_selected", UnitObject: &containerRef, SourcePlayer: intPtr(1)},
		},
	}); err != nil {
		t.Fatalf("AddTrigger blocker: %v", err)
	}
	refs, recipe, err := file.UnitDisconnectRecipe(containerRef)
	if err != nil {
		t.Fatalf("UnitDisconnectRecipe: %v", err)
	}
	if refs.Summary.Total != 3 || len(recipe.Triggers) != 1 || len(recipe.Units) != 1 {
		t.Fatalf("disconnect refs=%+v recipe=%+v, want trigger edit plus garrison clear", refs.Summary, recipe)
	}
	resolvedRef, captionRefs, captionRecipe, err := file.UnitCaptionDisconnectRecipe(containerCaption, intPtr(1))
	if err != nil {
		t.Fatalf("UnitCaptionDisconnectRecipe: %v", err)
	}
	if resolvedRef != containerRef || captionRefs.Summary.Total != refs.Summary.Total || len(captionRecipe.Triggers) != 1 || len(captionRecipe.Units) != 1 {
		t.Fatalf("caption disconnect ref=%d refs=%+v recipe=%+v, want same unit cleanup", resolvedRef, captionRefs.Summary, captionRecipe)
	}
	if got := recipe.Triggers[0].RemoveEffects; len(got) != 1 || got[0] != 0 {
		t.Fatalf("remove effects = %v, want [0]", got)
	}
	if got := recipe.Triggers[0].RemoveConditions; len(got) != 1 || got[0] != 0 {
		t.Fatalf("remove conditions = %v, want [0]", got)
	}
	if recipe.Units[0].ReferenceID == nil || *recipe.Units[0].ReferenceID != garrisonRef || recipe.Units[0].GarrisonedInID == nil || *recipe.Units[0].GarrisonedInID != -1 {
		t.Fatalf("unit cleanup = %+v, want garrison clear by source reference id", recipe.Units[0])
	}
	if err := file.EditTrigger(recipe.Triggers[0]); err != nil {
		t.Fatalf("EditTrigger cleanup: %v", err)
	}
	if err := file.EditUnit(recipe.Units[0]); err != nil {
		t.Fatalf("EditUnit cleanup: %v", err)
	}
	clear, err := file.DeletePlan(DeletePlanRequest{Kind: "unit", ID: containerRef})
	if err != nil {
		t.Fatalf("DeletePlan after cleanup: %v", err)
	}
	if !clear.CanDelete || clear.ReferenceSummary.Total != 0 {
		t.Fatalf("delete plan after cleanup = %+v, want clean physical delete", clear)
	}
}

func TestScenarioDeletePlanTriggerAndUnsupportedKinds(t *testing.T) {
	input := testfixtures.Path(t, "save-analysis/diagnostics/Replay_Transparency_Diagnostic_v2b_flagged.aoe2scenario")
	if _, err := os.Stat(input); err != nil {
		t.Skipf("diagnostic fixture unavailable: %v", err)
	}
	file, err := Open(input)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	targetTrigger := len(file.root.section("Triggers").list("trigger_data"))
	if err := file.AddTrigger(TriggerRecipe{Op: "add_trigger", Name: "delete-plan trigger target"}); err != nil {
		t.Fatalf("AddTrigger target: %v", err)
	}
	if err := file.AddTrigger(TriggerRecipe{
		Op:   "add_trigger",
		Name: "blocks trigger delete",
		Effects: []EffectRecipe{
			{Op: "activate_trigger", TriggerID: &targetTrigger},
		},
	}); err != nil {
		t.Fatalf("AddTrigger blocker: %v", err)
	}
	blocked, err := file.DeletePlan(DeletePlanRequest{Kind: "trigger", ID: targetTrigger})
	if err != nil {
		t.Fatalf("DeletePlan trigger blocked: %v", err)
	}
	if blocked.CanDelete || !blocked.CanTombstone || len(blocked.BlockingRefs) != 1 || blocked.SuggestedRecipe == nil || blocked.CleanupRecipe == nil || blocked.CleanupCommand == "" {
		t.Fatalf("trigger blocked plan = %+v, want physical block plus tombstone and cleanup recipes", blocked)
	}
	data, err := json.Marshal(blocked.SuggestedRecipe)
	if err != nil {
		t.Fatalf("marshal tombstone recipe: %v", err)
	}
	if !strings.Contains(string(data), "tombstone_trigger") {
		t.Fatalf("suggested recipe = %s, want tombstone_trigger", data)
	}
	cleanupData, err := json.Marshal(blocked.CleanupRecipe)
	if err != nil {
		t.Fatalf("marshal cleanup recipe: %v", err)
	}
	if !strings.Contains(string(cleanupData), "edit_trigger") || !strings.Contains(string(cleanupData), "remove_effects") || !strings.Contains(blocked.CleanupCommand, "scen disconnect") {
		t.Fatalf("cleanup command=%q recipe=%s, want trigger disconnect cleanup", blocked.CleanupCommand, cleanupData)
	}
	blockedByName, err := file.DeletePlan(DeletePlanRequest{Kind: "trigger-name", TargetName: "delete-plan trigger target"})
	if err != nil {
		t.Fatalf("DeletePlan trigger-name blocked: %v", err)
	}
	if blockedByName.CanDelete || !blockedByName.CanTombstone || blockedByName.CleanupRecipe == nil || !strings.Contains(blockedByName.Strategy, fmt.Sprintf("index %d", targetTrigger)) {
		t.Fatalf("trigger-name blocked plan = %+v, want resolved-index tombstone", blockedByName)
	}
	resolvedTrigger, nameRefs, nameRecipe, err := file.TriggerNameDisconnectRecipe("delete-plan trigger target")
	if err != nil {
		t.Fatalf("TriggerNameDisconnectRecipe: %v", err)
	}
	if resolvedTrigger != targetTrigger || nameRefs.Summary.Total != blocked.ReferenceSummary.Total || len(nameRecipe.Triggers) != 1 {
		t.Fatalf("trigger-name disconnect resolved=%d refs=%+v recipe=%+v, want same trigger cleanup", resolvedTrigger, nameRefs.Summary, nameRecipe)
	}
	clear, err := file.DeletePlan(DeletePlanRequest{Kind: "trigger", ID: targetTrigger + 1})
	if err != nil {
		t.Fatalf("DeletePlan trigger clear: %v", err)
	}
	if !clear.CanDelete || clear.SuggestedRecipe == nil {
		t.Fatalf("trigger clear plan = %+v, want recipe", clear)
	}
	clearByName, err := file.DeletePlan(DeletePlanRequest{Kind: "trigger_name", TargetName: "blocks trigger delete"})
	if err != nil {
		t.Fatalf("DeletePlan trigger-name clear: %v", err)
	}
	if !clearByName.CanDelete || clearByName.SuggestedRecipe == nil || !strings.Contains(clearByName.Strategy, "unique name selector") {
		t.Fatalf("trigger-name clear plan = %+v, want name recipe", clearByName)
	}
	scriptCallTrigger := len(file.root.section("Triggers").list("trigger_data"))
	if err := file.AddTrigger(TriggerRecipe{
		Op:   "add_trigger",
		Name: "A2K Script Call Delete",
		Effects: []EffectRecipe{{
			Op:      "script_call",
			Message: "A2K_DeleteProbe();",
		}},
	}); err != nil {
		t.Fatalf("AddTrigger script_call: %v", err)
	}
	scriptCallPlan, err := file.DeletePlan(DeletePlanRequest{Kind: "trigger", ID: scriptCallTrigger})
	if err != nil {
		t.Fatalf("DeletePlan script_call trigger: %v", err)
	}
	if !scriptCallPlan.CanDelete || !strings.Contains(strings.Join(scriptCallPlan.StructuralCaveats, "\n"), "script_call") {
		t.Fatalf("script_call trigger plan = %+v, want deletable plan with semantic caveat", scriptCallPlan)
	}
	duplicateName := "delete-plan duplicate trigger name"
	if err := file.AddTrigger(TriggerRecipe{Op: "add_trigger", Name: duplicateName}); err != nil {
		t.Fatalf("AddTrigger duplicate A: %v", err)
	}
	if err := file.AddTrigger(TriggerRecipe{Op: "add_trigger", Name: duplicateName}); err != nil {
		t.Fatalf("AddTrigger duplicate B: %v", err)
	}
	if _, err := file.DeletePlan(DeletePlanRequest{Kind: "trigger-name", TargetName: duplicateName}); err == nil {
		t.Fatal("DeletePlan trigger-name accepted duplicate trigger names")
	}
	prefixStart := len(file.root.section("Triggers").list("trigger_data"))
	if err := file.AddTrigger(TriggerRecipe{Op: "add_trigger", Name: "A2K Prefix Delete: alpha"}); err != nil {
		t.Fatalf("AddTrigger prefix A: %v", err)
	}
	if err := file.AddTrigger(TriggerRecipe{Op: "add_trigger", Name: "A2K Prefix Delete: beta"}); err != nil {
		t.Fatalf("AddTrigger prefix B: %v", err)
	}
	prefixPlan, err := file.DeletePlan(DeletePlanRequest{Kind: "trigger-prefix", TargetPrefix: "A2K Prefix Delete:"})
	if err != nil {
		t.Fatalf("DeletePlan trigger-prefix clear: %v", err)
	}
	if !prefixPlan.CanDelete || prefixPlan.SuggestedRecipe == nil || len(prefixPlan.ResolvedIDs) != 2 ||
		prefixPlan.ResolvedIDs[0] != prefixStart || prefixPlan.ResolvedIDs[1] != prefixStart+1 {
		t.Fatalf("trigger-prefix clear plan = %+v, want two resolved physical removes", prefixPlan)
	}
	data, err = json.Marshal(prefixPlan.SuggestedRecipe)
	if err != nil {
		t.Fatalf("marshal trigger-prefix remove recipe: %v", err)
	}
	if !strings.Contains(string(data), `"remove_triggers"`) ||
		!strings.Contains(string(data), strconv.Itoa(prefixStart)) ||
		!strings.Contains(string(data), strconv.Itoa(prefixStart+1)) {
		t.Fatalf("trigger-prefix remove recipe = %s, want remove_triggers target indexes", data)
	}
	internalPrefixStart := len(file.root.section("Triggers").list("trigger_data"))
	internalPrefixTarget := internalPrefixStart + 1
	if err := file.AddTrigger(TriggerRecipe{
		Op:   "add_trigger",
		Name: "A2K Prefix Internal Delete: source",
		Effects: []EffectRecipe{
			{Op: "activate_trigger", TriggerID: &internalPrefixTarget},
		},
	}); err != nil {
		t.Fatalf("AddTrigger internal prefix source: %v", err)
	}
	if err := file.AddTrigger(TriggerRecipe{Op: "add_trigger", Name: "A2K Prefix Internal Delete: target"}); err != nil {
		t.Fatalf("AddTrigger internal prefix target: %v", err)
	}
	internalPrefixPlan, err := file.DeletePlan(DeletePlanRequest{Kind: "trigger-prefix", TargetPrefix: "A2K Prefix Internal Delete:"})
	if err != nil {
		t.Fatalf("DeletePlan trigger-prefix internal refs: %v", err)
	}
	if !internalPrefixPlan.CanDelete || internalPrefixPlan.ReferenceSummary.Total != 0 ||
		len(internalPrefixPlan.ResolvedIDs) != 2 ||
		!strings.Contains(strings.Join(internalPrefixPlan.StructuralCaveats, "\n"), "internal trigger-control reference") {
		t.Fatalf("trigger-prefix internal plan = %+v, want physical delete with internal-ref caveat only", internalPrefixPlan)
	}
	referencedPrefixTarget := len(file.root.section("Triggers").list("trigger_data"))
	if err := file.AddTrigger(TriggerRecipe{Op: "add_trigger", Name: "A2K Prefix Tombstone: target"}); err != nil {
		t.Fatalf("AddTrigger referenced prefix target: %v", err)
	}
	if err := file.AddTrigger(TriggerRecipe{
		Op:   "add_trigger",
		Name: "blocks trigger-prefix delete",
		Effects: []EffectRecipe{
			{Op: "activate_trigger", TriggerID: &referencedPrefixTarget},
		},
	}); err != nil {
		t.Fatalf("AddTrigger prefix blocker: %v", err)
	}
	prefixBlocked, err := file.DeletePlan(DeletePlanRequest{Kind: "trigger-prefix", TargetPrefix: "A2K Prefix Tombstone:"})
	if err != nil {
		t.Fatalf("DeletePlan trigger-prefix blocked: %v", err)
	}
	if prefixBlocked.CanDelete || !prefixBlocked.CanTombstone || len(prefixBlocked.ResolvedIDs) != 1 ||
		prefixBlocked.CleanupRecipe == nil || prefixBlocked.CleanupCommand == "" ||
		!strings.Contains(prefixBlocked.Strategy, "semantic tombstones") {
		t.Fatalf("trigger-prefix blocked plan = %+v, want tombstone plus cleanup", prefixBlocked)
	}
	resolvedPrefix, prefixRefs, prefixCleanup, err := file.TriggerPrefixDisconnectRecipe("A2K Prefix Tombstone:")
	if err != nil {
		t.Fatalf("TriggerPrefixDisconnectRecipe: %v", err)
	}
	if len(resolvedPrefix) != 1 || resolvedPrefix[0] != referencedPrefixTarget ||
		prefixRefs.Summary.Total != 1 || len(prefixCleanup.Triggers) != 1 {
		t.Fatalf("trigger-prefix disconnect resolved=%v refs=%+v cleanup=%+v, want one trigger cleanup", resolvedPrefix, prefixRefs.Summary, prefixCleanup)
	}
	if _, err := file.DeletePlan(DeletePlanRequest{Kind: "trigger", ID: targetTrigger + 20}); err == nil {
		t.Fatal("DeletePlan accepted nonexistent trigger")
	}
	containsA := len(file.root.section("Triggers").list("trigger_data"))
	if err := file.AddTrigger(TriggerRecipe{
		Op:   "add_trigger",
		Name: "contains delete target by name MARKER",
		Effects: []EffectRecipe{
			{Op: "display_instructions", Message: "no marker here"},
		},
	}); err != nil {
		t.Fatalf("AddTrigger contains A: %v", err)
	}
	containsB := containsA + 1
	if err := file.AddTrigger(TriggerRecipe{
		Op:   "add_trigger",
		Name: "contains delete target by effect",
		Effects: []EffectRecipe{
			{Op: "display_instructions", Message: "MARKER in effect text"},
			{Op: "activate_trigger", TriggerID: &containsA},
		},
	}); err != nil {
		t.Fatalf("AddTrigger contains B: %v", err)
	}
	containsPlan, err := file.DeletePlan(DeletePlanRequest{Kind: "trigger-contains", TargetText: "MARKER"})
	if err != nil {
		t.Fatalf("DeletePlan trigger-contains: %v", err)
	}
	if !containsPlan.CanDelete || containsPlan.ReferenceSummary.Total != 0 || len(containsPlan.ResolvedIDs) != 2 {
		t.Fatalf("trigger-contains plan = %+v, want clean two-trigger batch", containsPlan)
	}
	var containsRecipe Recipe
	containsData, err := json.Marshal(containsPlan.SuggestedRecipe)
	if err != nil {
		t.Fatalf("marshal trigger-contains recipe: %v", err)
	}
	if err := json.Unmarshal(containsData, &containsRecipe); err != nil {
		t.Fatalf("unmarshal trigger-contains recipe: %v", err)
	}
	if err := file.ApplyRecipe(containsRecipe); err != nil {
		t.Fatalf("ApplyRecipe trigger-contains: %v", err)
	}
	if _, err := file.findUniqueTriggerIndexByName("contains delete target by name MARKER"); err == nil {
		t.Fatal("trigger-contains left name-matched trigger behind")
	}
	if _, err := file.findUniqueTriggerIndexByName("contains delete target by effect"); err == nil {
		t.Fatalf("trigger-contains left effect-matched trigger behind at original index %d", containsB)
	}
	blockedContains := len(file.root.section("Triggers").list("trigger_data"))
	if err := file.AddTrigger(TriggerRecipe{Op: "add_trigger", Name: "blocked contains MARKER2"}); err != nil {
		t.Fatalf("AddTrigger blocked contains: %v", err)
	}
	if err := file.AddTrigger(TriggerRecipe{
		Op:   "add_trigger",
		Name: "outside contains blocker",
		Effects: []EffectRecipe{
			{Op: "activate_trigger", TriggerID: &blockedContains},
		},
	}); err != nil {
		t.Fatalf("AddTrigger outside contains blocker: %v", err)
	}
	blockedContainsPlan, err := file.DeletePlan(DeletePlanRequest{Kind: "trigger-contains", TargetText: "MARKER2"})
	if err != nil {
		t.Fatalf("DeletePlan blocked trigger-contains: %v", err)
	}
	if blockedContainsPlan.CanDelete || !blockedContainsPlan.CanTombstone || blockedContainsPlan.ReferenceSummary.Total != 1 || blockedContainsPlan.CleanupRecipe == nil {
		t.Fatalf("blocked trigger-contains plan = %+v, want tombstone plus cleanup", blockedContainsPlan)
	}
	variableID := 9903
	if err := file.AddVariable(VariableRecipe{Op: "add_variable", ID: &variableID, Name: "DeletePlanVar"}); err != nil {
		t.Fatalf("AddVariable: %v", err)
	}
	variable, err := file.DeletePlan(DeletePlanRequest{Kind: "variable", ID: variableID})
	if err != nil {
		t.Fatalf("DeletePlan variable: %v", err)
	}
	if !variable.CanDelete || variable.SuggestedRecipe == nil || !strings.Contains(variable.Strategy, "remove unreferenced variable record") {
		t.Fatalf("variable plan = %+v, want remove recipe", variable)
	}
	variableByName, err := file.DeletePlan(DeletePlanRequest{Kind: "variable-name", TargetName: "DeletePlanVar"})
	if err != nil {
		t.Fatalf("DeletePlan variable-name: %v", err)
	}
	if !variableByName.CanDelete || variableByName.SuggestedRecipe == nil || !strings.Contains(variableByName.Strategy, fmt.Sprintf("id %d", variableID)) {
		t.Fatalf("variable-name plan = %+v, want resolved variable recipe", variableByName)
	}
	prefixVarA := 99041
	prefixVarB := 99042
	if err := file.AddVariable(VariableRecipe{Op: "add_variable", ID: &prefixVarA, Name: "A2K Prefix Var: alpha"}); err != nil {
		t.Fatalf("AddVariable prefix A: %v", err)
	}
	if err := file.AddVariable(VariableRecipe{Op: "add_variable", ID: &prefixVarB, Name: "A2K Prefix Var: beta"}); err != nil {
		t.Fatalf("AddVariable prefix B: %v", err)
	}
	variablePrefix, err := file.DeletePlan(DeletePlanRequest{Kind: "variable-prefix", TargetPrefix: "A2K Prefix Var:"})
	if err != nil {
		t.Fatalf("DeletePlan variable-prefix: %v", err)
	}
	if !variablePrefix.CanDelete || variablePrefix.SuggestedRecipe == nil || len(variablePrefix.ResolvedIDs) != 2 ||
		variablePrefix.ResolvedIDs[0] != prefixVarA || variablePrefix.ResolvedIDs[1] != prefixVarB {
		t.Fatalf("variable-prefix plan = %+v, want two resolved variable deletes", variablePrefix)
	}
	data, err = json.Marshal(variablePrefix.SuggestedRecipe)
	if err != nil {
		t.Fatalf("marshal variable-prefix remove recipe: %v", err)
	}
	if !strings.Contains(string(data), `"target_id":`+strconv.Itoa(prefixVarA)) ||
		!strings.Contains(string(data), `"target_id":`+strconv.Itoa(prefixVarB)) {
		t.Fatalf("variable-prefix remove recipe = %s, want target ids", data)
	}
	referencedPrefixVar := 99043
	if err := file.AddVariable(VariableRecipe{Op: "add_variable", ID: &referencedPrefixVar, Name: "A2K Referenced Prefix Var: target"}); err != nil {
		t.Fatalf("AddVariable referenced prefix: %v", err)
	}
	if err := file.AddTrigger(TriggerRecipe{
		Op:   "add_trigger",
		Name: "blocks variable-prefix delete",
		Conditions: []ConditionRecipe{
			{Op: "variable_value", Variable: &referencedPrefixVar, Comparison: intPtr(0), Quantity: intPtr(1)},
		},
	}); err != nil {
		t.Fatalf("AddTrigger variable prefix blocker: %v", err)
	}
	variablePrefixBlocked, err := file.DeletePlan(DeletePlanRequest{Kind: "variable-prefix", TargetPrefix: "A2K Referenced Prefix Var:"})
	if err != nil {
		t.Fatalf("DeletePlan blocked variable-prefix: %v", err)
	}
	if variablePrefixBlocked.CanDelete || variablePrefixBlocked.CleanupRecipe == nil || variablePrefixBlocked.CleanupCommand == "" ||
		len(variablePrefixBlocked.ResolvedIDs) != 1 || variablePrefixBlocked.ResolvedIDs[0] != referencedPrefixVar {
		t.Fatalf("blocked variable-prefix plan = %+v, want blocked cleanup by resolved id", variablePrefixBlocked)
	}
	resolvedVarPrefix, varPrefixRefs, varPrefixCleanup, err := file.VariablePrefixDisconnectRecipe("A2K Referenced Prefix Var:")
	if err != nil {
		t.Fatalf("VariablePrefixDisconnectRecipe: %v", err)
	}
	if len(resolvedVarPrefix) != 1 || resolvedVarPrefix[0] != referencedPrefixVar ||
		varPrefixRefs.Summary.Total != 1 || len(varPrefixCleanup.Triggers) != 1 {
		t.Fatalf("variable-prefix disconnect resolved=%v refs=%+v cleanup=%+v, want one trigger cleanup", resolvedVarPrefix, varPrefixRefs.Summary, varPrefixCleanup)
	}
	containsVarA := 99044
	containsVarB := 99045
	if err := file.AddVariable(VariableRecipe{Op: "add_variable", ID: &containsVarA, Name: "alpha A2K Contains Var target"}); err != nil {
		t.Fatalf("AddVariable contains A: %v", err)
	}
	if err := file.AddVariable(VariableRecipe{Op: "add_variable", ID: &containsVarB, Name: "beta A2K Contains Var target"}); err != nil {
		t.Fatalf("AddVariable contains B: %v", err)
	}
	variableContains, err := file.DeletePlan(DeletePlanRequest{Kind: "variable-contains", TargetText: "A2K Contains Var"})
	if err != nil {
		t.Fatalf("DeletePlan variable-contains: %v", err)
	}
	if !variableContains.CanDelete || variableContains.SuggestedRecipe == nil || len(variableContains.ResolvedIDs) != 2 ||
		variableContains.ResolvedIDs[0] != containsVarA || variableContains.ResolvedIDs[1] != containsVarB {
		t.Fatalf("variable-contains plan = %+v, want two resolved variable deletes", variableContains)
	}
	data, err = json.Marshal(variableContains.SuggestedRecipe)
	if err != nil {
		t.Fatalf("marshal variable-contains remove recipe: %v", err)
	}
	if !strings.Contains(string(data), `"target_id":`+strconv.Itoa(containsVarA)) ||
		!strings.Contains(string(data), `"target_id":`+strconv.Itoa(containsVarB)) {
		t.Fatalf("variable-contains remove recipe = %s, want target ids", data)
	}
	referencedContainsVar := 99046
	if err := file.AddVariable(VariableRecipe{Op: "add_variable", ID: &referencedContainsVar, Name: "prefix A2K Referenced Contains Var suffix"}); err != nil {
		t.Fatalf("AddVariable referenced contains: %v", err)
	}
	if err := file.AddTrigger(TriggerRecipe{
		Op:   "add_trigger",
		Name: "blocks variable-contains delete",
		Conditions: []ConditionRecipe{
			{Op: "variable_value", Variable: &referencedContainsVar, Comparison: intPtr(0), Quantity: intPtr(1)},
		},
	}); err != nil {
		t.Fatalf("AddTrigger variable contains blocker: %v", err)
	}
	variableContainsBlocked, err := file.DeletePlan(DeletePlanRequest{Kind: "variable-contains", TargetText: "A2K Referenced Contains Var"})
	if err != nil {
		t.Fatalf("DeletePlan blocked variable-contains: %v", err)
	}
	if variableContainsBlocked.CanDelete || variableContainsBlocked.CleanupRecipe == nil || variableContainsBlocked.CleanupCommand == "" ||
		len(variableContainsBlocked.ResolvedIDs) != 1 || variableContainsBlocked.ResolvedIDs[0] != referencedContainsVar {
		t.Fatalf("blocked variable-contains plan = %+v, want blocked cleanup by resolved id", variableContainsBlocked)
	}
	resolvedVarContains, varContainsRefs, varContainsCleanup, err := file.VariableContainsDisconnectRecipe("A2K Referenced Contains Var")
	if err != nil {
		t.Fatalf("VariableContainsDisconnectRecipe: %v", err)
	}
	if len(resolvedVarContains) != 1 || resolvedVarContains[0] != referencedContainsVar ||
		varContainsRefs.Summary.Total != 1 || len(varContainsCleanup.Triggers) != 1 {
		t.Fatalf("variable-contains disconnect resolved=%v refs=%+v cleanup=%+v, want one trigger cleanup", resolvedVarContains, varContainsRefs.Summary, varContainsCleanup)
	}
	duplicateVariableName := "DeletePlanDuplicateVar"
	dupA := 99031
	dupB := 99032
	if err := file.AddVariable(VariableRecipe{Op: "add_variable", ID: &dupA, Name: duplicateVariableName}); err != nil {
		t.Fatalf("AddVariable duplicate A: %v", err)
	}
	if err := file.AddVariable(VariableRecipe{Op: "add_variable", ID: &dupB, Name: duplicateVariableName}); err != nil {
		t.Fatalf("AddVariable duplicate B: %v", err)
	}
	if _, err := file.DeletePlan(DeletePlanRequest{Kind: "variable-name", TargetName: duplicateVariableName}); err == nil {
		t.Fatal("DeletePlan variable-name accepted duplicate variable names")
	}
	stringID, err := file.AddString(StringRecipe{Op: "add_string", Text: "DeletePlanString"})
	if err != nil {
		t.Fatalf("AddString: %v", err)
	}
	stringPlan, err := file.DeletePlan(DeletePlanRequest{Kind: "string", ID: stringID})
	if err != nil {
		t.Fatalf("DeletePlan string: %v", err)
	}
	if !stringPlan.CanDelete || stringPlan.SuggestedRecipe == nil || !strings.Contains(stringPlan.Strategy, "clear fixed string-table slot") {
		t.Fatalf("string plan = %+v, want clear recipe", stringPlan)
	}
	stringByText, err := file.DeletePlan(DeletePlanRequest{Kind: "string-text", TargetText: "DeletePlanString"})
	if err != nil {
		t.Fatalf("DeletePlan string-text: %v", err)
	}
	if !stringByText.CanDelete || stringByText.SuggestedRecipe == nil || !strings.Contains(stringByText.Strategy, fmt.Sprintf("slot %d", stringID)) {
		t.Fatalf("string-text plan = %+v, want clear recipe resolved to string id", stringByText)
	}
	data, err = json.Marshal(stringByText.SuggestedRecipe)
	if err != nil {
		t.Fatalf("marshal string-text clear recipe: %v", err)
	}
	if !strings.Contains(string(data), "clear_string") || !strings.Contains(string(data), "DeletePlanString") {
		t.Fatalf("string-text suggested recipe = %s, want old_text clear recipe", data)
	}
	if _, err := file.AddString(StringRecipe{Op: "add_string", Text: "DeletePlanString"}); err != nil {
		t.Fatalf("AddString duplicate text: %v", err)
	}
	if _, err := file.DeletePlan(DeletePlanRequest{Kind: "string-text", TargetText: "DeletePlanString"}); err == nil {
		t.Fatal("DeletePlan string-text accepted duplicate exact text")
	}
	uniqueReferencedText := "DeletePlanStringRefsByText"
	referencedStringID, err := file.AddString(StringRecipe{Op: "add_string", Text: uniqueReferencedText})
	if err != nil {
		t.Fatalf("AddString referenced text: %v", err)
	}
	refID := 997701
	x := 10.5
	y := 11.5
	z := 0.0
	status := 2
	if err := file.AddUnit(UnitRecipe{
		Op:              "add_unit",
		Player:          1,
		UnitConst:       600,
		X:               &x,
		Y:               &y,
		Z:               &z,
		Status:          &status,
		ReferenceID:     &refID,
		CaptionStringID: &referencedStringID,
	}); err != nil {
		t.Fatalf("AddUnit caption ref: %v", err)
	}
	referencedString, err := file.DeletePlan(DeletePlanRequest{Kind: "string", ID: referencedStringID})
	if err != nil {
		t.Fatalf("DeletePlan referenced string: %v", err)
	}
	if referencedString.CanDelete || !referencedString.CanTombstone || len(referencedString.BlockingRefs) != 1 || referencedString.SuggestedRecipe == nil {
		t.Fatalf("referenced string plan = %+v, want physical block plus tombstone recipe", referencedString)
	}
	if referencedString.CleanupRecipe == nil || referencedString.CleanupCommand == "" {
		t.Fatalf("referenced string plan = %+v, want cleanup recipe/command", referencedString)
	}
	data, err = json.Marshal(referencedString.SuggestedRecipe)
	if err != nil {
		t.Fatalf("marshal string tombstone recipe: %v", err)
	}
	if !strings.Contains(string(data), "TOMBSTONED string") {
		t.Fatalf("referenced string suggested recipe = %s, want tombstone text", data)
	}
	referencedStringByText, err := file.DeletePlan(DeletePlanRequest{Kind: "string-text", TargetText: uniqueReferencedText})
	if err != nil {
		t.Fatalf("DeletePlan referenced string-text: %v", err)
	}
	if referencedStringByText.CanDelete || !referencedStringByText.CanTombstone ||
		referencedStringByText.CleanupRecipe == nil || referencedStringByText.CleanupCommand == "" ||
		!strings.Contains(referencedStringByText.Strategy, fmt.Sprintf("id %d", referencedStringID)) {
		t.Fatalf("referenced string-text plan = %+v, want resolved tombstone plus cleanup", referencedStringByText)
	}
	prefixStringA, err := file.AddString(StringRecipe{Op: "add_string", Text: "A2K Prefix String: alpha"})
	if err != nil {
		t.Fatalf("AddString prefix A: %v", err)
	}
	prefixStringB, err := file.AddString(StringRecipe{Op: "add_string", Text: "A2K Prefix String: beta"})
	if err != nil {
		t.Fatalf("AddString prefix B: %v", err)
	}
	stringPrefixPlan, err := file.DeletePlan(DeletePlanRequest{Kind: "string-prefix", TargetPrefix: "A2K Prefix String:"})
	if err != nil {
		t.Fatalf("DeletePlan string-prefix: %v", err)
	}
	if !stringPrefixPlan.CanDelete || stringPrefixPlan.SuggestedRecipe == nil || len(stringPrefixPlan.ResolvedIDs) != 2 ||
		stringPrefixPlan.ResolvedIDs[0] != prefixStringA || stringPrefixPlan.ResolvedIDs[1] != prefixStringB {
		t.Fatalf("string-prefix plan = %+v, want two resolved clear_string recipes", stringPrefixPlan)
	}
	data, err = json.Marshal(stringPrefixPlan.SuggestedRecipe)
	if err != nil {
		t.Fatalf("marshal string-prefix clear recipe: %v", err)
	}
	if !strings.Contains(string(data), `"id":`+strconv.Itoa(prefixStringA)) ||
		!strings.Contains(string(data), `"id":`+strconv.Itoa(prefixStringB)) {
		t.Fatalf("string-prefix clear recipe = %s, want target ids", data)
	}
	referencedPrefixString, err := file.AddString(StringRecipe{Op: "add_string", Text: "A2K Referenced Prefix String: target"})
	if err != nil {
		t.Fatalf("AddString referenced prefix: %v", err)
	}
	refID2 := 997702
	if err := file.AddUnit(UnitRecipe{
		Op:              "add_unit",
		Player:          1,
		UnitConst:       600,
		X:               &x,
		Y:               &y,
		Z:               &z,
		Status:          &status,
		ReferenceID:     &refID2,
		CaptionStringID: &referencedPrefixString,
	}); err != nil {
		t.Fatalf("AddUnit prefix string caption ref: %v", err)
	}
	stringPrefixBlocked, err := file.DeletePlan(DeletePlanRequest{Kind: "string-prefix", TargetPrefix: "A2K Referenced Prefix String:"})
	if err != nil {
		t.Fatalf("DeletePlan referenced string-prefix: %v", err)
	}
	if stringPrefixBlocked.CanDelete || !stringPrefixBlocked.CanTombstone || stringPrefixBlocked.CleanupRecipe == nil ||
		stringPrefixBlocked.CleanupCommand == "" || len(stringPrefixBlocked.ResolvedIDs) != 1 ||
		stringPrefixBlocked.ResolvedIDs[0] != referencedPrefixString {
		t.Fatalf("referenced string-prefix plan = %+v, want tombstone plus cleanup by resolved id", stringPrefixBlocked)
	}
	resolvedStringPrefix, stringPrefixRefs, stringPrefixCleanup, err := file.StringPrefixDisconnectRecipe("A2K Referenced Prefix String:")
	if err != nil {
		t.Fatalf("StringPrefixDisconnectRecipe: %v", err)
	}
	if len(resolvedStringPrefix) != 1 || resolvedStringPrefix[0] != referencedPrefixString ||
		stringPrefixRefs.Summary.Total != 1 || len(stringPrefixCleanup.Units) != 1 {
		t.Fatalf("string-prefix disconnect resolved=%v refs=%+v cleanup=%+v, want one unit cleanup", resolvedStringPrefix, stringPrefixRefs.Summary, stringPrefixCleanup)
	}
	containsStringA, err := file.AddString(StringRecipe{Op: "add_string", Text: "alpha A2K Contains String target"})
	if err != nil {
		t.Fatalf("AddString contains A: %v", err)
	}
	containsStringB, err := file.AddString(StringRecipe{Op: "add_string", Text: "beta A2K Contains String target"})
	if err != nil {
		t.Fatalf("AddString contains B: %v", err)
	}
	stringContainsPlan, err := file.DeletePlan(DeletePlanRequest{Kind: "string-contains", TargetText: "A2K Contains String"})
	if err != nil {
		t.Fatalf("DeletePlan string-contains: %v", err)
	}
	if !stringContainsPlan.CanDelete || stringContainsPlan.SuggestedRecipe == nil || len(stringContainsPlan.ResolvedIDs) != 2 ||
		stringContainsPlan.ResolvedIDs[0] != containsStringA || stringContainsPlan.ResolvedIDs[1] != containsStringB {
		t.Fatalf("string-contains plan = %+v, want two resolved clear_string recipes", stringContainsPlan)
	}
	data, err = json.Marshal(stringContainsPlan.SuggestedRecipe)
	if err != nil {
		t.Fatalf("marshal string-contains clear recipe: %v", err)
	}
	if !strings.Contains(string(data), `"id":`+strconv.Itoa(containsStringA)) ||
		!strings.Contains(string(data), `"id":`+strconv.Itoa(containsStringB)) {
		t.Fatalf("string-contains clear recipe = %s, want target ids", data)
	}
	referencedContainsString, err := file.AddString(StringRecipe{Op: "add_string", Text: "prefix A2K Referenced Contains String suffix"})
	if err != nil {
		t.Fatalf("AddString referenced contains: %v", err)
	}
	refID3 := 997703
	if err := file.AddUnit(UnitRecipe{
		Op:              "add_unit",
		Player:          1,
		UnitConst:       600,
		X:               &x,
		Y:               &y,
		Z:               &z,
		Status:          &status,
		ReferenceID:     &refID3,
		CaptionStringID: &referencedContainsString,
	}); err != nil {
		t.Fatalf("AddUnit contains string caption ref: %v", err)
	}
	stringContainsBlocked, err := file.DeletePlan(DeletePlanRequest{Kind: "string-contains", TargetText: "A2K Referenced Contains String"})
	if err != nil {
		t.Fatalf("DeletePlan referenced string-contains: %v", err)
	}
	if stringContainsBlocked.CanDelete || !stringContainsBlocked.CanTombstone || stringContainsBlocked.CleanupRecipe == nil ||
		stringContainsBlocked.CleanupCommand == "" || len(stringContainsBlocked.ResolvedIDs) != 1 ||
		stringContainsBlocked.ResolvedIDs[0] != referencedContainsString {
		t.Fatalf("referenced string-contains plan = %+v, want tombstone plus cleanup by resolved id", stringContainsBlocked)
	}
	resolvedStringContains, stringContainsRefs, stringContainsCleanup, err := file.StringContainsDisconnectRecipe("A2K Referenced Contains String")
	if err != nil {
		t.Fatalf("StringContainsDisconnectRecipe: %v", err)
	}
	if len(resolvedStringContains) != 1 || resolvedStringContains[0] != referencedContainsString ||
		stringContainsRefs.Summary.Total != 1 || len(stringContainsCleanup.Units) != 1 {
		t.Fatalf("string-contains disconnect resolved=%v refs=%+v cleanup=%+v, want one unit cleanup", resolvedStringContains, stringContainsRefs.Summary, stringContainsCleanup)
	}
}

func TestScenarioStringDisconnectClearsDirectReferences(t *testing.T) {
	input := testfixtures.Path(t, "save-analysis/diagnostics/Replay_Transparency_Diagnostic_v2b_flagged.aoe2scenario")
	if _, err := os.Stat(input); err != nil {
		t.Skipf("diagnostic fixture unavailable: %v", err)
	}
	file, err := Open(input)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	stringID, err := file.AddString(StringRecipe{Op: "add_string", Text: "DeletePlanStringRefs"})
	if err != nil {
		t.Fatalf("AddString: %v", err)
	}
	refID := 990805
	x := 10.5
	y := 11.5
	if err := file.AddUnit(UnitRecipe{Op: "add_unit", Player: 1, UnitConst: 600, X: &x, Y: &y, ReferenceID: &refID, CaptionStringID: &stringID}); err != nil {
		t.Fatalf("AddUnit caption ref: %v", err)
	}
	triggerIndex := len(file.root.section("Triggers").list("trigger_data"))
	if err := file.AddTrigger(TriggerRecipe{
		Op:   "add_trigger",
		Name: "string disconnect source",
		Effects: []EffectRecipe{
			{Op: "display_instructions", Message: "uses string id"},
		},
	}); err != nil {
		t.Fatalf("AddTrigger: %v", err)
	}
	trigger := file.root.section("Triggers").list("trigger_data")[triggerIndex]
	if err := setIntField(trigger, "description_string_table_id", "s32", stringID); err != nil {
		t.Fatalf("set description_string_table_id: %v", err)
	}
	if err := setIntField(trigger, "short_description_string_table_id", "s32", stringID); err != nil {
		t.Fatalf("set short_description_string_table_id: %v", err)
	}
	if err := setIntField(trigger.list("effect_data")[0], "string_id", "s32", stringID); err != nil {
		t.Fatalf("set effect string_id: %v", err)
	}
	refs, cleanup, err := file.StringDisconnectRecipe(stringID)
	if err != nil {
		t.Fatalf("StringDisconnectRecipe: %v", err)
	}
	if refs.Summary.Total != 4 || len(cleanup.Triggers) != 1 || len(cleanup.Units) != 1 {
		t.Fatalf("string disconnect refs=%+v cleanup=%+v, want one trigger edit plus one unit edit", refs.Summary, cleanup)
	}
	resolvedStringID, textRefs, textCleanup, err := file.StringTextDisconnectRecipe("DeletePlanStringRefs")
	if err != nil {
		t.Fatalf("StringTextDisconnectRecipe: %v", err)
	}
	if resolvedStringID != stringID || textRefs.Summary.Total != refs.Summary.Total ||
		len(textCleanup.Triggers) != len(cleanup.Triggers) || len(textCleanup.Units) != len(cleanup.Units) {
		t.Fatalf("string-text disconnect resolved=%d refs=%+v cleanup=%+v, want same string cleanup", resolvedStringID, textRefs.Summary, textCleanup)
	}
	triggerCleanup := cleanup.Triggers[0]
	if triggerCleanup.DescriptionStringID == nil || *triggerCleanup.DescriptionStringID != -1 ||
		triggerCleanup.ShortDescriptionStringID == nil || *triggerCleanup.ShortDescriptionStringID != -1 {
		t.Fatalf("trigger cleanup = %+v, want both description ids cleared", triggerCleanup)
	}
	if len(triggerCleanup.RemoveEffects) != 1 || triggerCleanup.RemoveEffects[0] != 0 {
		t.Fatalf("remove effects = %v, want [0]", triggerCleanup.RemoveEffects)
	}
	if cleanup.Units[0].ReferenceID == nil || *cleanup.Units[0].ReferenceID != refID ||
		cleanup.Units[0].CaptionStringID == nil || *cleanup.Units[0].CaptionStringID != -1 {
		t.Fatalf("unit cleanup = %+v, want caption string clear", cleanup.Units[0])
	}
	if err := file.EditTrigger(triggerCleanup); err != nil {
		t.Fatalf("EditTrigger string cleanup: %v", err)
	}
	if err := file.EditUnit(cleanup.Units[0]); err != nil {
		t.Fatalf("EditUnit string cleanup: %v", err)
	}
	clearPlan, err := file.DeletePlan(DeletePlanRequest{Kind: "string", ID: stringID})
	if err != nil {
		t.Fatalf("DeletePlan string after cleanup: %v", err)
	}
	if !clearPlan.CanDelete || clearPlan.ReferenceSummary.Total != 0 {
		t.Fatalf("string plan after cleanup = %+v, want clean clear", clearPlan)
	}
}

func TestAddEditTombstoneAndRemoveVariableRecipes(t *testing.T) {
	input := testfixtures.Path(t, "save-analysis/diagnostics/Replay_Transparency_Diagnostic_v2b_flagged.aoe2scenario")
	if _, err := os.Stat(input); err != nil {
		t.Skipf("diagnostic fixture unavailable: %v", err)
	}
	file, err := Open(input)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	before := file.Triggers.Variables
	id := 9901
	if err := file.AddVariable(VariableRecipe{Op: "add_variable", ID: &id, Name: "DexTemp"}); err != nil {
		t.Fatalf("AddVariable: %v", err)
	}
	if file.Triggers.Variables != before+1 {
		t.Fatalf("variable count = %d, want %d", file.Triggers.Variables, before+1)
	}
	setName := "DexRenamed"
	if err := file.EditVariable(VariableRecipe{Op: "edit_variable", TargetID: &id, SetName: &setName}); err != nil {
		t.Fatalf("EditVariable: %v", err)
	}
	variable, err := file.findVariableByID(id)
	if err != nil {
		t.Fatalf("findVariableByID: %v", err)
	}
	name, _ := variable.stringValue("variable_name")
	if name != setName {
		t.Fatalf("variable name = %q, want %q", name, setName)
	}
	if err := file.TombstoneVariable(VariableRecipe{Op: "tombstone_variable", TargetName: setName}); err != nil {
		t.Fatalf("TombstoneVariable: %v", err)
	}
	variable, err = file.findVariableByID(id)
	if err != nil {
		t.Fatalf("find tombstoned variable: %v", err)
	}
	name, _ = variable.stringValue("variable_name")
	if name != "_DeletedVariable9901" {
		t.Fatalf("tombstone name = %q, want _DeletedVariable9901", name)
	}
	if file.Triggers.Variables != before+1 {
		t.Fatalf("tombstone changed variable count to %d", file.Triggers.Variables)
	}
	id2 := 9904
	if err := file.AddVariable(VariableRecipe{Op: "add_variable", ID: &id2, Name: "DexRemoveMe"}); err != nil {
		t.Fatalf("AddVariable remove target: %v", err)
	}
	if err := file.RemoveVariable(VariableRecipe{Op: "remove_variable", TargetID: &id2}); err != nil {
		t.Fatalf("RemoveVariable: %v", err)
	}
	if _, err := file.findVariableByID(id2); err == nil {
		t.Fatalf("removed variable id %d is still present", id2)
	}
	if file.Triggers.Variables != before+1 {
		t.Fatalf("remove changed variable count to %d, want %d", file.Triggers.Variables, before+1)
	}
	out := filepath.Join(t.TempDir(), "removed_variable.aoe2scenario")
	if err := file.Write(out); err != nil {
		t.Fatalf("Write: %v", err)
	}
	reopened, err := Open(out)
	if err != nil {
		t.Fatalf("Open written scenario: %v", err)
	}
	if err := reopened.VerifyRebuild(); err != nil {
		t.Fatalf("VerifyRebuild: %v", err)
	}
	if _, err := reopened.findVariableByID(id); err != nil {
		t.Fatalf("surviving tombstoned variable id %d missing after reopen: %v", id, err)
	}
	if _, err := reopened.findVariableByID(id2); err == nil {
		t.Fatalf("removed variable id %d returned after reopen", id2)
	}
}

func TestPlanAndPatchRejectReferencedVariableTombstone(t *testing.T) {
	input := testfixtures.Path(t, "save-analysis/diagnostics/Replay_Transparency_Diagnostic_v2b_flagged.aoe2scenario")
	if _, err := os.Stat(input); err != nil {
		t.Skipf("diagnostic fixture unavailable: %v", err)
	}
	file, err := Open(input)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	id := 9902
	name := "DexReferenced"
	if err := file.AddVariable(VariableRecipe{Op: "add_variable", ID: &id, Name: name}); err != nil {
		t.Fatalf("AddVariable: %v", err)
	}
	if err := file.AddTrigger(TriggerRecipe{
		Op:   "add_trigger",
		Name: "variable blocker",
		Conditions: []ConditionRecipe{
			{Op: "variable_value", Variable: &id, Comparison: intPtr(0), Quantity: intPtr(7)},
		},
	}); err != nil {
		t.Fatalf("AddTrigger: %v", err)
	}
	recipe := Recipe{Variables: []VariableRecipe{{Op: "tombstone_variable", TargetID: &id}}}
	if _, err := file.Plan(recipe); err == nil || !strings.Contains(err.Error(), "referenced") {
		t.Fatalf("Plan tombstone error = %v, want referenced refusal", err)
	}
	if err := file.TombstoneVariable(VariableRecipe{Op: "tombstone_variable", TargetID: &id}); err == nil || !strings.Contains(err.Error(), "referenced") {
		t.Fatalf("TombstoneVariable error = %v, want referenced refusal", err)
	}
	removeRecipe := Recipe{Variables: []VariableRecipe{{Op: "remove_variable", TargetID: &id}}}
	if _, err := file.Plan(removeRecipe); err == nil || !strings.Contains(err.Error(), "referenced") {
		t.Fatalf("Plan remove error = %v, want referenced refusal", err)
	}
	if err := file.RemoveVariable(VariableRecipe{Op: "remove_variable", TargetID: &id}); err == nil || !strings.Contains(err.Error(), "referenced") {
		t.Fatalf("RemoveVariable error = %v, want referenced refusal", err)
	}
	plan, err := file.DeletePlan(DeletePlanRequest{Kind: "variable", ID: id})
	if err != nil {
		t.Fatalf("DeletePlan referenced variable: %v", err)
	}
	if plan.CanDelete || plan.CleanupRecipe == nil || plan.CleanupCommand == "" {
		t.Fatalf("referenced variable plan = %+v, want cleanup recipe/command and blocked delete", plan)
	}
	planByName, err := file.DeletePlan(DeletePlanRequest{Kind: "variable-name", TargetName: name})
	if err != nil {
		t.Fatalf("DeletePlan referenced variable-name: %v", err)
	}
	if planByName.CanDelete || planByName.CleanupRecipe == nil || !strings.Contains(planByName.CleanupCommand, fmt.Sprintf("variable %d", id)) {
		t.Fatalf("referenced variable-name plan = %+v, want cleanup by resolved id", planByName)
	}
	refs, cleanup, err := file.VariableDisconnectRecipe(id)
	if err != nil {
		t.Fatalf("VariableDisconnectRecipe: %v", err)
	}
	resolvedID, nameRefs, nameCleanup, err := file.VariableNameDisconnectRecipe(name)
	if err != nil {
		t.Fatalf("VariableNameDisconnectRecipe: %v", err)
	}
	if resolvedID != id || nameRefs.Summary.Total != refs.Summary.Total || len(nameCleanup.Triggers) != 1 {
		t.Fatalf("variable-name disconnect id=%d refs=%+v cleanup=%+v, want same variable cleanup", resolvedID, nameRefs.Summary, nameCleanup)
	}
	if refs.Summary.Total != 1 || len(cleanup.Triggers) != 1 || len(cleanup.Triggers[0].RemoveConditions) != 1 {
		t.Fatalf("variable disconnect refs=%+v cleanup=%+v, want one removed condition", refs.Summary, cleanup)
	}
	if err := file.EditTrigger(cleanup.Triggers[0]); err != nil {
		t.Fatalf("EditTrigger variable cleanup: %v", err)
	}
	cleanPlan, err := file.DeletePlan(DeletePlanRequest{Kind: "variable", ID: id})
	if err != nil {
		t.Fatalf("DeletePlan after variable cleanup: %v", err)
	}
	if !cleanPlan.CanDelete || cleanPlan.ReferenceSummary.Total != 0 {
		t.Fatalf("variable plan after cleanup = %+v, want clean physical delete", cleanPlan)
	}
}

func TestScenarioDeletePlanEffectAndConditionRows(t *testing.T) {
	input := testfixtures.Path(t, "save-analysis/diagnostics/Replay_Transparency_Diagnostic_v2b_flagged.aoe2scenario")
	if _, err := os.Stat(input); err != nil {
		t.Skipf("diagnostic fixture unavailable: %v", err)
	}
	file, err := Open(input)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	triggerIndex := len(file.root.section("Triggers").list("trigger_data"))
	if err := file.AddTrigger(TriggerRecipe{
		Op:   "add_trigger",
		Name: "child delete target",
		Effects: []EffectRecipe{
			{Op: "display_instructions", Message: "keep"},
			{Op: "display_instructions", Message: "delete"},
		},
		Conditions: []ConditionRecipe{
			{Op: "timer", Timer: intPtr(3)},
		},
	}); err != nil {
		t.Fatalf("AddTrigger: %v", err)
	}
	effectIndex := 1
	effectPlan, err := file.DeletePlan(DeletePlanRequest{Kind: "effect", TriggerID: &triggerIndex, ChildIndex: &effectIndex})
	if err != nil {
		t.Fatalf("DeletePlan effect: %v", err)
	}
	if !effectPlan.CanDelete || effectPlan.SuggestedRecipe == nil || !strings.Contains(effectPlan.Strategy, "effect row 1") {
		t.Fatalf("effect plan = %+v, want child-row removal recipe", effectPlan)
	}
	conditionIndex := 0
	conditionPlan, err := file.DeletePlan(DeletePlanRequest{Kind: "condition", TriggerID: &triggerIndex, ChildIndex: &conditionIndex})
	if err != nil {
		t.Fatalf("DeletePlan condition: %v", err)
	}
	if !conditionPlan.CanDelete || conditionPlan.SuggestedRecipe == nil || !strings.Contains(conditionPlan.Strategy, "condition row 0") {
		t.Fatalf("condition plan = %+v, want child-row removal recipe", conditionPlan)
	}
	if err := file.EditTrigger(TriggerRecipe{Op: "edit_trigger", TargetIndex: &triggerIndex, RemoveEffects: []int{effectIndex}, RemoveConditions: []int{conditionIndex}}); err != nil {
		t.Fatalf("EditTrigger remove children: %v", err)
	}
	trigger, err := file.triggerByIndex(triggerIndex)
	if err != nil {
		t.Fatalf("triggerByIndex: %v", err)
	}
	if got := len(trigger.list("effect_data")); got != 1 {
		t.Fatalf("effect count = %d, want 1", got)
	}
	if got := len(trigger.list("condition_data")); got != 0 {
		t.Fatalf("condition count = %d, want 0", got)
	}
	if _, err := file.DeletePlan(DeletePlanRequest{Kind: "effect", TriggerID: &triggerIndex, ChildIndex: &effectIndex}); err == nil {
		t.Fatal("DeletePlan accepted removed effect index")
	}

	typeTriggerA := len(file.root.section("Triggers").list("trigger_data"))
	if err := file.AddTrigger(TriggerRecipe{
		Op:   "add_trigger",
		Name: "type delete target A",
		Effects: []EffectRecipe{
			{Op: "display_instructions", Message: "delete display A1"},
			{Op: "send_chat", Message: "keep chat A"},
			{Op: "display_instructions", Message: "delete display A2"},
		},
		Conditions: []ConditionRecipe{
			{Op: "timer", Timer: intPtr(4)},
			{Op: "object_selected", UnitObject: intPtr(12345)},
		},
	}); err != nil {
		t.Fatalf("AddTrigger type A: %v", err)
	}
	typeTriggerB := typeTriggerA + 1
	if err := file.AddTrigger(TriggerRecipe{
		Op:   "add_trigger",
		Name: "type delete target B",
		Effects: []EffectRecipe{
			{Op: "display_instructions", Message: "delete display B"},
		},
		Conditions: []ConditionRecipe{
			{Op: "timer", Timer: intPtr(5)},
		},
	}); err != nil {
		t.Fatalf("AddTrigger type B: %v", err)
	}
	effectTypePlan, err := file.DeletePlan(DeletePlanRequest{Kind: "effect-type", ID: 20, TargetName: "display_instructions", TargetPrefix: "type delete target"})
	if err != nil {
		t.Fatalf("DeletePlan effect-type: %v", err)
	}
	if !effectTypePlan.CanDelete || len(effectTypePlan.ResolvedIDs) != 6 || !strings.Contains(effectTypePlan.Strategy, "3 effect row(s)") {
		t.Fatalf("effect-type plan = %+v, want three matched effect rows", effectTypePlan)
	}
	var effectTypeRecipe Recipe
	effectTypeData, err := json.Marshal(effectTypePlan.SuggestedRecipe)
	if err != nil {
		t.Fatalf("marshal effect-type recipe: %v", err)
	}
	if err := json.Unmarshal(effectTypeData, &effectTypeRecipe); err != nil {
		t.Fatalf("unmarshal effect-type recipe: %v", err)
	}
	if err := file.ApplyRecipe(effectTypeRecipe); err != nil {
		t.Fatalf("ApplyRecipe effect-type: %v", err)
	}
	triggerA, err := file.triggerByIndex(typeTriggerA)
	if err != nil {
		t.Fatalf("triggerByIndex A: %v", err)
	}
	triggerB, err := file.triggerByIndex(typeTriggerB)
	if err != nil {
		t.Fatalf("triggerByIndex B: %v", err)
	}
	if got := len(triggerA.list("effect_data")); got != 1 {
		t.Fatalf("type trigger A effect count = %d, want 1", got)
	}
	effectType, ok := triggerA.list("effect_data")[0].intValue("effect_type")
	if !ok {
		t.Fatal("type trigger A remaining effect missing effect_type")
	}
	if got := EffectTypeName(effectType); got != "send_chat" {
		t.Fatalf("type trigger A remaining effect = %q, want send_chat", got)
	}
	if got := len(triggerB.list("effect_data")); got != 0 {
		t.Fatalf("type trigger B effect count = %d, want 0", got)
	}
	if got := triggerA.intList("effect_display_order_array"); len(got) != 1 || got[0] != 0 {
		t.Fatalf("type trigger A effect order = %#v, want [0]", got)
	}

	conditionTypePlan, err := file.DeletePlan(DeletePlanRequest{Kind: "condition-type", ID: 10, TargetName: "timer", TriggerID: &typeTriggerA})
	if err != nil {
		t.Fatalf("DeletePlan condition-type: %v", err)
	}
	if !conditionTypePlan.CanDelete || len(conditionTypePlan.ResolvedIDs) != 2 || !strings.Contains(conditionTypePlan.Strategy, "1 condition row(s)") {
		t.Fatalf("condition-type plan = %+v, want one scoped condition row", conditionTypePlan)
	}
	var conditionTypeRecipe Recipe
	conditionTypeData, err := json.Marshal(conditionTypePlan.SuggestedRecipe)
	if err != nil {
		t.Fatalf("marshal condition-type recipe: %v", err)
	}
	if err := json.Unmarshal(conditionTypeData, &conditionTypeRecipe); err != nil {
		t.Fatalf("unmarshal condition-type recipe: %v", err)
	}
	if err := file.ApplyRecipe(conditionTypeRecipe); err != nil {
		t.Fatalf("ApplyRecipe condition-type: %v", err)
	}
	if got := len(triggerA.list("condition_data")); got != 1 {
		t.Fatalf("type trigger A condition count = %d, want 1", got)
	}
	conditionType, ok := triggerA.list("condition_data")[0].intValue("condition_type")
	if !ok {
		t.Fatal("type trigger A remaining condition missing condition_type")
	}
	if got := ConditionTypeName(conditionType); got != "object_selected" {
		t.Fatalf("type trigger A remaining condition = %q, want object_selected", got)
	}
	if got := len(triggerB.list("condition_data")); got != 1 {
		t.Fatalf("type trigger B condition count = %d, want 1", got)
	}
	if _, err := file.DeletePlan(DeletePlanRequest{Kind: "effect-type", ID: 20, TriggerID: &typeTriggerA}); err == nil {
		t.Fatal("DeletePlan effect-type accepted scope with no remaining matching rows")
	}

	textTrigger := len(file.root.section("Triggers").list("trigger_data"))
	if err := file.AddTrigger(TriggerRecipe{
		Op:   "add_trigger",
		Name: "text prefix delete target",
		Effects: []EffectRecipe{
			{Op: "display_instructions", Message: "A2KTXT delete display"},
			{Op: "send_chat", Message: "keep chat"},
			{Op: "script_call", Message: "A2KTXT_delete_script();"},
		},
	}); err != nil {
		t.Fatalf("AddTrigger text prefix: %v", err)
	}
	textPrefixPlan, err := file.DeletePlan(DeletePlanRequest{Kind: "effect-text-prefix", TargetText: "A2KTXT", TriggerID: &textTrigger})
	if err != nil {
		t.Fatalf("DeletePlan effect-text-prefix: %v", err)
	}
	if !textPrefixPlan.CanDelete || len(textPrefixPlan.ResolvedIDs) != 4 || !strings.Contains(textPrefixPlan.Strategy, "2 effect row(s)") {
		t.Fatalf("effect-text-prefix plan = %+v, want two scoped effect rows", textPrefixPlan)
	}
	var textPrefixRecipe Recipe
	textPrefixData, err := json.Marshal(textPrefixPlan.SuggestedRecipe)
	if err != nil {
		t.Fatalf("marshal effect-text-prefix recipe: %v", err)
	}
	if err := json.Unmarshal(textPrefixData, &textPrefixRecipe); err != nil {
		t.Fatalf("unmarshal effect-text-prefix recipe: %v", err)
	}
	if err := file.ApplyRecipe(textPrefixRecipe); err != nil {
		t.Fatalf("ApplyRecipe effect-text-prefix: %v", err)
	}
	textNode, err := file.triggerByIndex(textTrigger)
	if err != nil {
		t.Fatalf("triggerByIndex text prefix: %v", err)
	}
	if got := len(textNode.list("effect_data")); got != 1 {
		t.Fatalf("text-prefix trigger effect count = %d, want 1", got)
	}
	message, _ := textNode.list("effect_data")[0].stringValue("message")
	if message != "keep chat" {
		t.Fatalf("text-prefix remaining message = %q, want keep chat", message)
	}
	if _, err := file.DeletePlan(DeletePlanRequest{Kind: "effect-text-prefix", TargetText: "A2KTXT", TriggerID: &textTrigger}); err == nil {
		t.Fatal("DeletePlan effect-text-prefix accepted scope with no matching rows")
	}
	containsTextTrigger := len(file.root.section("Triggers").list("trigger_data"))
	if err := file.AddTrigger(TriggerRecipe{
		Op:   "add_trigger",
		Name: "text contains delete target",
		Effects: []EffectRecipe{
			{Op: "display_instructions", Message: "keep before"},
			{Op: "display_instructions", Message: "display with MIDMARK in middle"},
			{Op: "send_chat", Message: "chat with MIDMARK suffix"},
			{Op: "script_call", Message: "keep_after();"},
		},
	}); err != nil {
		t.Fatalf("AddTrigger text contains: %v", err)
	}
	textContainsPlan, err := file.DeletePlan(DeletePlanRequest{Kind: "effect-text-contains", TargetText: "MIDMARK", TriggerID: &containsTextTrigger})
	if err != nil {
		t.Fatalf("DeletePlan effect-text-contains: %v", err)
	}
	if !textContainsPlan.CanDelete || len(textContainsPlan.ResolvedIDs) != 4 || !strings.Contains(textContainsPlan.Strategy, "2 effect row(s)") {
		t.Fatalf("effect-text-contains plan = %+v, want two scoped effect rows", textContainsPlan)
	}
	var textContainsRecipe Recipe
	textContainsData, err := json.Marshal(textContainsPlan.SuggestedRecipe)
	if err != nil {
		t.Fatalf("marshal effect-text-contains recipe: %v", err)
	}
	if err := json.Unmarshal(textContainsData, &textContainsRecipe); err != nil {
		t.Fatalf("unmarshal effect-text-contains recipe: %v", err)
	}
	if err := file.ApplyRecipe(textContainsRecipe); err != nil {
		t.Fatalf("ApplyRecipe effect-text-contains: %v", err)
	}
	containsTextNode, err := file.triggerByIndex(containsTextTrigger)
	if err != nil {
		t.Fatalf("triggerByIndex text contains: %v", err)
	}
	remainingMessages := []string{}
	for _, effect := range containsTextNode.list("effect_data") {
		message, _ := effect.stringValue("message")
		remainingMessages = append(remainingMessages, message)
	}
	if !reflect.DeepEqual(remainingMessages, []string{"keep before", "keep_after();"}) {
		t.Fatalf("text-contains remaining messages = %#v, want keep rows only", remainingMessages)
	}
	if _, err := file.DeletePlan(DeletePlanRequest{Kind: "effect-text-contains", TargetText: "MIDMARK", TriggerID: &containsTextTrigger}); err == nil {
		t.Fatal("DeletePlan effect-text-contains accepted scope with no matching rows")
	}
}

func TestScenarioDeletePlanSystemPrefix(t *testing.T) {
	input := testfixtures.Path(t, "save-analysis/diagnostics/Replay_Transparency_Diagnostic_v2b_flagged.aoe2scenario")
	if _, err := os.Stat(input); err != nil {
		t.Skipf("diagnostic fixture unavailable: %v", err)
	}
	file, err := Open(input)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	prefix := "A2K System Clean:"
	stringID, err := file.AddString(StringRecipe{Op: "add_string", Text: prefix + " string"})
	if err != nil {
		t.Fatalf("AddString: %v", err)
	}
	variableID := 880101
	if err := file.AddVariable(VariableRecipe{Op: "add_variable", ID: &variableID, Name: prefix + " variable"}); err != nil {
		t.Fatalf("AddVariable: %v", err)
	}
	unitX, unitY := 24.0, 24.0
	if err := file.AddUnit(UnitRecipe{Op: "add_unit", Player: 1, UnitConst: 600, X: &unitX, Y: &unitY, CaptionString: prefix + " unit"}); err != nil {
		t.Fatalf("AddUnit: %v", err)
	}
	triggerB := len(file.root.section("Triggers").list("trigger_data"))
	if err := file.AddTrigger(TriggerRecipe{Op: "add_trigger", Name: prefix + " B"}); err != nil {
		t.Fatalf("AddTrigger B: %v", err)
	}
	triggerA := triggerB + 1
	if err := file.AddTrigger(TriggerRecipe{
		Op:   "add_trigger",
		Name: prefix + " A",
		Effects: []EffectRecipe{
			{Op: "activate_trigger", TriggerID: &triggerB},
			{Op: "change_variable", Variable: &variableID, Quantity: intPtr(1)},
		},
	}); err != nil {
		t.Fatalf("AddTrigger A: %v", err)
	}
	plan, err := file.DeletePlan(DeletePlanRequest{Kind: "system-prefix", TargetPrefix: prefix})
	if err != nil {
		t.Fatalf("DeletePlan system-prefix: %v", err)
	}
	if !plan.CanDelete || plan.ReferenceSummary.Total != 0 || plan.SuggestedRecipe == nil {
		t.Fatalf("system-prefix plan = %+v, want clean delete", plan)
	}
	if len(plan.ResolvedSets["triggers"]) != 2 || len(plan.ResolvedSets["variables"]) != 1 || len(plan.ResolvedSets["strings"]) != 1 || len(plan.ResolvedSets["units"]) != 1 {
		t.Fatalf("system-prefix resolved sets = %#v, want trigger/variable/string/unit matches", plan.ResolvedSets)
	}
	var recipe Recipe
	data, err := json.Marshal(plan.SuggestedRecipe)
	if err != nil {
		t.Fatalf("marshal system-prefix recipe: %v", err)
	}
	if err := json.Unmarshal(data, &recipe); err != nil {
		t.Fatalf("unmarshal system-prefix recipe: %v", err)
	}
	if err := file.ApplyRecipe(recipe); err != nil {
		t.Fatalf("ApplyRecipe system-prefix: %v", err)
	}
	if _, err := file.findVariableByID(variableID); err == nil {
		t.Fatal("system-prefix left matched variable behind")
	}
	values, err := file.scenarioStringValues()
	if err != nil {
		t.Fatalf("scenarioStringValues: %v", err)
	}
	if values[stringID] != "" {
		t.Fatalf("system-prefix string id %d = %q, want cleared", stringID, values[stringID])
	}
	if _, err := file.findUniqueTriggerIndexByName(prefix + " A"); err == nil {
		t.Fatalf("system-prefix left trigger A behind at original index %d", triggerA)
	}
	if _, err := file.findUniqueTriggerIndexByName(prefix + " B"); err == nil {
		t.Fatalf("system-prefix left trigger B behind")
	}
	if _, err := file.findUnitsByCaptionPrefix(prefix, nil); err == nil {
		t.Fatal("system-prefix left captioned unit behind")
	}

	blockedPrefix := "A2K System Blocked:"
	blockedTrigger := len(file.root.section("Triggers").list("trigger_data"))
	if err := file.AddTrigger(TriggerRecipe{Op: "add_trigger", Name: blockedPrefix + " target"}); err != nil {
		t.Fatalf("AddTrigger blocked target: %v", err)
	}
	if err := file.AddTrigger(TriggerRecipe{
		Op:   "add_trigger",
		Name: "outside system blocker",
		Effects: []EffectRecipe{
			{Op: "activate_trigger", TriggerID: &blockedTrigger},
		},
	}); err != nil {
		t.Fatalf("AddTrigger external blocker: %v", err)
	}
	blockedPlan, err := file.DeletePlan(DeletePlanRequest{Kind: "system-prefix", TargetPrefix: blockedPrefix})
	if err != nil {
		t.Fatalf("DeletePlan blocked system-prefix: %v", err)
	}
	if blockedPlan.CanDelete || blockedPlan.ReferenceSummary.Total != 1 || len(blockedPlan.BlockingRefs) != 1 {
		t.Fatalf("blocked system-prefix plan = %+v, want one external blocker", blockedPlan)
	}
}

func TestAddSetTombstoneAndClearStringRecipes(t *testing.T) {
	input := testfixtures.Path(t, "save-analysis/diagnostics/Replay_Transparency_Diagnostic_v2b_flagged.aoe2scenario")
	if _, err := os.Stat(input); err != nil {
		t.Skipf("diagnostic fixture unavailable: %v", err)
	}
	file, err := Open(input)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	id, err := file.AddString(StringRecipe{Op: "add_string", Text: "DexStringSmoke"})
	if err != nil {
		t.Fatalf("AddString: %v", err)
	}
	values, err := file.scenarioStringValues()
	if err != nil {
		t.Fatalf("scenarioStringValues: %v", err)
	}
	if values[id] != "DexStringSmoke" {
		t.Fatalf("string %d = %q, want DexStringSmoke", id, values[id])
	}
	oldText := "DexStringSmoke"
	setText := "DexStringRenamed"
	if err := file.SetString(StringRecipe{Op: "set_string", ID: &id, OldText: oldText, SetText: &setText}); err != nil {
		t.Fatalf("SetString: %v", err)
	}
	values, _ = file.scenarioStringValues()
	if values[id] != setText {
		t.Fatalf("string %d after set = %q, want %q", id, values[id], setText)
	}
	if err := file.TombstoneString(StringRecipe{Op: "tombstone_string", ID: &id, OldText: setText}); err != nil {
		t.Fatalf("TombstoneString: %v", err)
	}
	values, _ = file.scenarioStringValues()
	if values[id] != fmt.Sprintf("_DeletedString%d", id) {
		t.Fatalf("string %d after tombstone = %q", id, values[id])
	}
	clearText := fmt.Sprintf("_DeletedString%d", id)
	if err := file.ClearString(StringRecipe{Op: "clear_string", OldText: clearText}); err != nil {
		t.Fatalf("ClearString: %v", err)
	}
	values, _ = file.scenarioStringValues()
	if values[id] != "" {
		t.Fatalf("string %d after clear = %q, want empty", id, values[id])
	}
	reused, err := file.AddString(StringRecipe{Op: "add_string", Text: "DexStringReuse"})
	if err != nil {
		t.Fatalf("AddString after clear: %v", err)
	}
	if reused != id {
		t.Fatalf("reused string id = %d, want cleared id %d", reused, id)
	}
}

func TestPlanAndPatchRejectReferencedStringTombstone(t *testing.T) {
	input := testfixtures.Path(t, "save-analysis/diagnostics/Replay_Transparency_Diagnostic_v2b_flagged.aoe2scenario")
	if _, err := os.Stat(input); err != nil {
		t.Skipf("diagnostic fixture unavailable: %v", err)
	}
	file, err := Open(input)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	id := 7
	if err := file.SetString(StringRecipe{Op: "set_string", ID: &id, Text: "DexReferencedString"}); err != nil {
		t.Fatalf("SetString: %v", err)
	}
	if err := file.AddTrigger(TriggerRecipe{
		Op:   "add_trigger",
		Name: "string blocker",
		Effects: []EffectRecipe{
			{Op: "display_instructions", Message: "uses string id"},
		},
	}); err != nil {
		t.Fatalf("AddTrigger: %v", err)
	}
	trigger := file.root.section("Triggers").list("trigger_data")
	added := trigger[len(trigger)-1]
	if err := setIntField(added.list("effect_data")[0], "string_id", "s32", id); err != nil {
		t.Fatalf("set string_id: %v", err)
	}
	recipe := Recipe{Strings: []StringRecipe{{Op: "tombstone_string", ID: &id}}}
	if _, err := file.Plan(recipe); err == nil || !strings.Contains(err.Error(), "referenced") {
		t.Fatalf("Plan tombstone string error = %v, want referenced refusal", err)
	}
	if err := file.TombstoneString(StringRecipe{Op: "tombstone_string", ID: &id}); err == nil || !strings.Contains(err.Error(), "referenced") {
		t.Fatalf("TombstoneString error = %v, want referenced refusal", err)
	}
	clearRecipe := Recipe{Strings: []StringRecipe{{Op: "clear_string", ID: &id}}}
	if _, err := file.Plan(clearRecipe); err == nil || !strings.Contains(err.Error(), "referenced") {
		t.Fatalf("Plan clear string error = %v, want referenced refusal", err)
	}
	if err := file.ClearString(StringRecipe{Op: "clear_string", ID: &id}); err == nil || !strings.Contains(err.Error(), "referenced") {
		t.Fatalf("ClearString error = %v, want referenced refusal", err)
	}
}

func TestScenarioSettingsReport(t *testing.T) {
	input := testfixtures.Path(t, "save-analysis/diagnostics/Replay_Transparency_Diagnostic_v2b_flagged.aoe2scenario")
	if _, err := os.Stat(input); err != nil {
		t.Skipf("diagnostic fixture unavailable: %v", err)
	}
	file, err := Open(input)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	report := file.Settings()
	if report.PlayerCount == 0 || len(report.Players) == 0 {
		t.Fatalf("empty settings players: %+v", report)
	}
	if len(report.Resources) == 0 || report.Players[0].Resources.Player != 0 {
		t.Fatalf("settings resources not wired into players: %+v", report.Players[0])
	}
	if len(report.Diplomacy.Matrix) == 0 || len(report.Players[0].Diplomacy) == 0 {
		t.Fatalf("settings diplomacy missing: %+v", report.Diplomacy)
	}
	if _, ok := report.Victory["conquest_required"]; !ok {
		t.Fatalf("settings victory missing conquest_required: %+v", report.Victory)
	}
	if report.Verification != "structure_verified_not_engine_verified" {
		t.Fatalf("verification = %q", report.Verification)
	}
}

func TestScenarioSettingsExposeStartingAgeAndColor(t *testing.T) {
	input := filepath.Join("..", "..", "testdata", "editor-refs", "Create scenario, do nothing, click save.aoe2scenario")
	file, err := Open(input)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	ensureRepeatedIntField(t, file.root.section("Options"), "per_player_starting_age", 1, 4)
	players := file.root.section("DataHeader").list("player_data_1")
	if len(players) <= 1 {
		t.Fatalf("player_data_1 len=%d, want P1 row", len(players))
	}
	setStringFieldForTest(t, players[1], "architecture_set", "MAGYAR-CIV")
	setRepeatedIntField(t, file.root.section("DataHeader"), "string_table_player_names", 1, 44)
	ensureRepeatedIntField(t, file.root.section("Options"), "per_player_base_priority", 1, 6)
	setRepeatedBigEndianU32Raw(t, file.root.section("Map"), "per_player_population_cap", 1, 180)
	resources := file.root.section("PlayerDataTwo").list("resources")
	if len(resources) <= 1 {
		t.Fatalf("resources len=%d, want P1 row", len(resources))
	}
	setIntFieldForTest(t, resources[1], "player_color", 7)
	file.Players = file.root.playerInfo()

	settings := file.Settings()
	if settings.Players[1].StartingAge != 4 || settings.Players[1].StartingAgeName != "Castle" {
		t.Fatalf("P1 starting age = %d/%q", settings.Players[1].StartingAge, settings.Players[1].StartingAgeName)
	}
	if settings.Players[1].Architecture != "MAGYAR-CIV" {
		t.Fatalf("P1 architecture = %q", settings.Players[1].Architecture)
	}
	if settings.Players[1].NameStringID != 44 {
		t.Fatalf("P1 name string id = %d", settings.Players[1].NameStringID)
	}
	if settings.Players[1].Color != 7 || settings.Players[1].ColorName != "Orange" {
		t.Fatalf("P1 color = %d/%q", settings.Players[1].Color, settings.Players[1].ColorName)
	}
	if settings.Players[1].BasePriority != 6 {
		t.Fatalf("P1 base priority = %d", settings.Players[1].BasePriority)
	}
}

func TestScenarioSettingsCalibrationFixtures(t *testing.T) {
	dir := scenarioProjectPath("Mandala", "felt-compendium", "kit_settings_calibration")
	baseline := filepath.Join(dir, "rof_p1_postimperial.aoe2scenario")
	castle := filepath.Join(dir, "rof_p1_castle.aoe2scenario")
	architecture := filepath.Join(dir, "rof_p2_magyar_architecture.aoe2scenario")
	basePriority6 := filepath.Join(dir, "rof_p3_basepriority_6.aoe2scenario")
	basePriority3 := filepath.Join(dir, "rof_p3_basepriority_3.aoe2scenario")
	popNameUnset := filepath.Join(dir, "rof_p4_pop121_nameid_unset.aoe2scenario")
	popNameSet := filepath.Join(dir, "rof_p4_pop180_nameid44.aoe2scenario")
	for _, path := range []string{baseline, castle, architecture, basePriority6, basePriority3, popNameUnset, popNameSet} {
		if _, err := os.Stat(path); err != nil {
			t.Skipf("settings calibration fixture unavailable: %s: %v", path, err)
		}
	}
	postImpFile, err := Open(baseline)
	if err != nil {
		t.Fatalf("Open post-imperial: %v", err)
	}
	postImp := postImpFile.Settings()
	if got := postImp.Players[0].StartingAge; got != 6 {
		t.Fatalf("baseline P0 starting_age=%d, want 6", got)
	}
	if got := postImp.Players[0].StartingAgeName; got != "Post-Imperial" {
		t.Fatalf("baseline P0 starting_age_name=%q, want Post-Imperial", got)
	}

	castleFile, err := Open(castle)
	if err != nil {
		t.Fatalf("Open castle: %v", err)
	}
	castleSettings := castleFile.Settings()
	if got := castleSettings.Players[0].StartingAge; got != 4 {
		t.Fatalf("castle P0 starting_age=%d, want 4", got)
	}
	if got := castleSettings.Players[0].StartingAgeName; got != "Castle" {
		t.Fatalf("castle P0 starting_age_name=%q, want Castle", got)
	}

	report := Diff(postImpFile, castleFile)
	if !diffHasField(report, "player_0.starting_age") {
		t.Fatalf("diff missing player_0.starting_age: %+v", report.Changes)
	}

	architectureFile, err := Open(architecture)
	if err != nil {
		t.Fatalf("Open architecture: %v", err)
	}
	architectureSettings := architectureFile.Settings()
	if got := architectureSettings.Players[1].Architecture; got != "MAGYAR-CIV" {
		t.Fatalf("architecture fixture P1 architecture=%q, want MAGYAR-CIV", got)
	}
	report = Diff(postImpFile, architectureFile)
	if !diffHasField(report, "player_1.architecture") {
		t.Fatalf("diff missing player_1.architecture: %+v", report.Changes)
	}

	priority6File, err := Open(basePriority6)
	if err != nil {
		t.Fatalf("Open base priority 6: %v", err)
	}
	priority6 := priority6File.Settings()
	if got := priority6.Players[2].BasePriority; got != 6 {
		t.Fatalf("base-priority fixture P2 base_priority=%d, want 6", got)
	}
	priority3File, err := Open(basePriority3)
	if err != nil {
		t.Fatalf("Open base priority 3: %v", err)
	}
	priority3 := priority3File.Settings()
	if got := priority3.Players[2].BasePriority; got != 3 {
		t.Fatalf("base-priority fixture P2 base_priority=%d, want 3", got)
	}
	report = Diff(priority6File, priority3File)
	if !diffHasField(report, "player_2.base_priority") {
		t.Fatalf("diff missing player_2.base_priority: %+v", report.Changes)
	}

	popUnsetFile, err := Open(popNameUnset)
	if err != nil {
		t.Fatalf("Open pop/name unset: %v", err)
	}
	popUnset := popUnsetFile.Settings()
	if got := popUnset.Players[3].PopulationLimit; got != 121 {
		t.Fatalf("pop unset fixture P3 population_limit=%d, want 121", got)
	}
	if got := popUnset.Players[3].NameStringID; got != -2 {
		t.Fatalf("pop unset fixture P3 name_string_id=%d, want -2", got)
	}
	popSetFile, err := Open(popNameSet)
	if err != nil {
		t.Fatalf("Open pop/name set: %v", err)
	}
	popSet := popSetFile.Settings()
	if got := popSet.Players[3].PopulationLimit; got != 180 {
		t.Fatalf("pop set fixture P3 population_limit=%d, want 180", got)
	}
	if got := popSet.Players[3].NameStringID; got != 44 {
		t.Fatalf("pop set fixture P3 name_string_id=%d, want 44", got)
	}
	if got := popSet.Players[0].PopulationLimit; got != 200 {
		t.Fatalf("pop set fixture P0 population_limit=%d, want default 200", got)
	}
	report = Diff(popUnsetFile, popSetFile)
	for _, field := range []string{"player_3.population_limit", "player_3.name_string_id"} {
		if !diffHasField(report, field) {
			t.Fatalf("diff missing %s: %+v", field, report.Changes)
		}
	}
}

func TestScenarioSettingsOptionsDiplomacyFixtures(t *testing.T) {
	dir := scenarioProjectPath("Mandala", "felt-compendium", "kit_settings_calibration")
	path := func(name string) string {
		return filepath.Join(dir, name+".aoe2scenario")
	}
	required := []string{
		"md_baseline",
		"md_t1_victory",
		"md_t2_lock_coop_alliances",
		"md_t3_villager_force_drop",
		"md_t4_collide_correcting",
		"md_t5_full_tech_tree",
		"md_t7_p1_disable_4buildings",
		"md_t8_p1_disable_3units",
		"md_t9_p1_disable_6techs",
		"md_t10_camera_110_9",
		"md_t11_p2_disable_3techs",
		"md_t12_execorder_newbehavior",
		"md_t13_fresh_blank_baseline",
		"md_t15_p1_allied_victory",
		"md_t16_players_choose_teams_off",
		"md_t17_random_start_points",
		"md_t18_lock_teams",
		"md_t20_max_teams_3",
	}
	for _, name := range required {
		if _, err := os.Stat(path(name)); err != nil {
			t.Skipf("settings options/diplomacy fixture unavailable: %s: %v", name, err)
		}
	}
	open := func(name string) *File {
		t.Helper()
		file, err := Open(path(name))
		if err != nil {
			t.Fatalf("Open %s: %v", name, err)
		}
		return file
	}

	victory := open("md_t1_victory")
	if got := victory.Settings().Victory["mode"]; got != 1 {
		t.Fatalf("victory mode=%d, want 1", got)
	}
	if got := victory.Settings().Victory["required_score_for_score_victory"]; got != 14000 {
		t.Fatalf("victory required score=%d, want 14000", got)
	}
	report := Diff(open("md_baseline"), victory)
	for _, field := range []string{"victory.mode", "victory.required_score_for_score_victory"} {
		if !diffHasField(report, field) {
			t.Fatalf("diff missing %s: %+v", field, report.Changes)
		}
	}

	if !open("md_t2_lock_coop_alliances").Settings().Options.LockCoopAlliances {
		t.Fatal("lock_coop_alliances not surfaced")
	}
	if !open("md_t3_villager_force_drop").Settings().Options.VillagerForceDrop {
		t.Fatal("villager_force_drop not surfaced")
	}
	if !open("md_t4_collide_correcting").Settings().Options.CollideAndCorrecting {
		t.Fatal("collide_and_correcting not surfaced")
	}
	if !open("md_t5_full_tech_tree").Settings().Options.FullTechTree {
		t.Fatal("full_tech_tree not surfaced")
	}
	for _, tc := range []struct {
		before string
		after  string
		field  string
	}{
		{"md_t1_victory", "md_t2_lock_coop_alliances", "options.lock_coop_alliances"},
		{"md_t2_lock_coop_alliances", "md_t3_villager_force_drop", "options.villager_force_drop"},
		{"md_t3_villager_force_drop", "md_t4_collide_correcting", "options.collide_and_correcting"},
		{"md_t4_collide_correcting", "md_t5_full_tech_tree", "options.full_tech_tree"},
	} {
		report := Diff(open(tc.before), open(tc.after))
		if !diffHasField(report, tc.field) {
			t.Fatalf("diff %s -> %s missing %s: %+v", tc.before, tc.after, tc.field, report.Changes)
		}
	}

	disabledBuildings := open("md_t7_p1_disable_4buildings").Settings()
	if got := disabledBuildings.Players[0].DisabledBuildings; !intSlicesEqual(got, []int{103, 45, 234, 104}) {
		t.Fatalf("P0 disabled buildings=%v", got)
	}
	disabledUnits := open("md_t8_p1_disable_3units").Settings()
	if got := disabledUnits.Players[0].DisabledUnits; !intSlicesEqual(got, []int{40, 283, 567}) {
		t.Fatalf("P0 disabled units=%v", got)
	}
	disabledTechs := open("md_t9_p1_disable_6techs").Settings()
	if got := disabledTechs.Players[0].DisabledTechs; !intSlicesEqual(got, []int{754, 19, 7, 514, 905, 513}) {
		t.Fatalf("P0 disabled techs=%v", got)
	}
	p2Techs := open("md_t11_p2_disable_3techs").Settings()
	if got := p2Techs.Players[1].DisabledTechs; !intSlicesEqual(got, []int{1202, 982, 517}) {
		t.Fatalf("P1 disabled techs=%v", got)
	}
	for _, tc := range []struct {
		before string
		after  string
		field  string
	}{
		{"md_t6_testdifficulty_noop", "md_t7_p1_disable_4buildings", "player_0.disabled_buildings"},
		{"md_t7_p1_disable_4buildings", "md_t8_p1_disable_3units", "player_0.disabled_units"},
		{"md_t8_p1_disable_3units", "md_t9_p1_disable_6techs", "player_0.disabled_techs"},
		{"md_t10_camera_110_9", "md_t11_p2_disable_3techs", "player_1.disabled_techs"},
	} {
		report := Diff(open(tc.before), open(tc.after))
		if !diffHasField(report, tc.field) {
			t.Fatalf("diff %s -> %s missing %s: %+v", tc.before, tc.after, tc.field, report.Changes)
		}
	}

	camera := open("md_t10_camera_110_9").Settings()
	if len(camera.Options.InitialPlayerViews) < 2 {
		t.Fatalf("camera views len=%d", len(camera.Options.InitialPlayerViews))
	}
	if got := camera.Options.InitialPlayerViews[1]; got.X != 110 || got.Y != 9 || !got.Set {
		t.Fatalf("P1 camera=%+v, want 110,9 set", got)
	}
	report = Diff(open("md_t9_p1_disable_6techs"), open("md_t10_camera_110_9"))
	if !diffHasField(report, "options.initial_player_views") {
		t.Fatalf("diff missing options.initial_player_views: %+v", report.Changes)
	}

	execOrder := open("md_t12_execorder_newbehavior").Settings()
	if execOrder.Options.TriggerExecutionOrder != 1 || execOrder.Options.TriggerExecutionOrderName != "New Behaviour" {
		t.Fatalf("trigger execution order=%d/%q", execOrder.Options.TriggerExecutionOrder, execOrder.Options.TriggerExecutionOrderName)
	}
	report = Diff(open("md_t11_p2_disable_3techs"), open("md_t12_execorder_newbehavior"))
	if !diffHasField(report, "options.trigger_execution_order") {
		t.Fatalf("diff missing options.trigger_execution_order: %+v", report.Changes)
	}

	if open("md_t16_players_choose_teams_off").Settings().Diplomacy.AllowPlayersChooseTeams {
		t.Fatal("allow_players_choose_teams should be false")
	}
	if !open("md_t17_random_start_points").Settings().Diplomacy.RandomStartPoints {
		t.Fatal("random_start_points not surfaced")
	}
	if !open("md_t18_lock_teams").Settings().Diplomacy.LockTeams {
		t.Fatal("lock_teams not surfaced")
	}
	if got := open("md_t20_max_teams_3").Settings().Diplomacy.MaxNumberOfTeams; got != 3 {
		t.Fatalf("max_number_of_teams=%d, want 3", got)
	}
	for _, tc := range []struct {
		before string
		after  string
		field  string
	}{
		{"md_t15_p1_allied_victory", "md_t16_players_choose_teams_off", "diplomacy.allow_players_choose_teams"},
		{"md_t16_players_choose_teams_off", "md_t17_random_start_points", "diplomacy.random_start_points"},
		{"md_t17_random_start_points", "md_t18_lock_teams", "diplomacy.lock_teams"},
		{"md_t13_fresh_blank_baseline", "md_t20_max_teams_3", "diplomacy.max_number_of_teams"},
	} {
		report := Diff(open(tc.before), open(tc.after))
		if !diffHasField(report, tc.field) {
			t.Fatalf("diff %s -> %s missing %s: %+v", tc.before, tc.after, tc.field, report.Changes)
		}
	}
}

func TestScenarioSettingsExposeWaterDefinition(t *testing.T) {
	dir := scenarioAoe2DEPath("runs", "cal159", "round3")
	before := filepath.Join(dir, "Clean v159.11.aoe2scenario")
	after := filepath.Join(dir, "Clean v159.12.aoe2scenario")
	for _, path := range []string{before, after} {
		if _, err := os.Stat(path); err != nil {
			t.Skipf("water definition calibration fixture unavailable: %s: %v", path, err)
		}
	}
	beforeReport, err := SettingsFile(before)
	if err != nil {
		t.Fatalf("SettingsFile before: %v", err)
	}
	afterReport, err := SettingsFile(after)
	if err != nil {
		t.Fatalf("SettingsFile after: %v", err)
	}
	if beforeReport.Map.WaterDefinition != "" {
		t.Fatalf("before water_definition=%q, want empty", beforeReport.Map.WaterDefinition)
	}
	if afterReport.Map.WaterDefinition != "Default" {
		t.Fatalf("after water_definition=%q, want Default", afterReport.Map.WaterDefinition)
	}
}

func TestSecondaryGameModesAreNamedAndDiffed(t *testing.T) {
	dir := scenarioAoe2DEPath("runs", "cal159", "round3")
	beforePath := filepath.Join(dir, "Clean v159.4.aoe2scenario")
	afterPath := filepath.Join(dir, "Clean v159.5.aoe2scenario")
	if _, err := os.Stat(beforePath); err != nil {
		t.Skipf("victory calibration fixture unavailable: %v", err)
	}
	if _, err := os.Stat(afterPath); err != nil {
		t.Skipf("secondary-mode calibration fixture unavailable: %v", err)
	}
	before, err := Open(beforePath)
	if err != nil {
		t.Fatal(err)
	}
	after, err := Open(afterPath)
	if err != nil {
		t.Fatal(err)
	}
	settings := after.Settings()
	if settings.VictoryModeName != "Secondary Game Mode" {
		t.Fatalf("victory mode name=%q, want Secondary Game Mode", settings.VictoryModeName)
	}
	if settings.SecondaryGameModes.Bits != 4 || len(settings.SecondaryGameModes.Names) != 1 || settings.SecondaryGameModes.Names[0] != "Regicide" {
		t.Fatalf("secondary modes=%+v, want Regicide bit 4", settings.SecondaryGameModes)
	}
	if !diffHasField(Diff(before, after), "secondary_game_modes") {
		t.Fatal("diff missing map.secondary_game_modes")
	}
	if got := Diff(before, after); len(got.Changes) == 0 || got.Changes[0].Detail != "bitfield" {
		t.Fatalf("secondary mode diff detail=%+v, want bitfield", got.Changes)
	}
}

func TestScenarioSettingsMessagesCinematicsFixtures(t *testing.T) {
	dir := scenarioProjectPath("Mandala", "felt-compendium", "kit_settings_calibration")
	path := func(name string) string {
		return filepath.Join(dir, name+".aoe2scenario")
	}
	required := []string{"md_t38_message_slots", "md_t39_all_messages", "md_t40_cinematic_pregame"}
	for _, name := range required {
		if _, err := os.Stat(path(name)); err != nil {
			t.Skipf("messages/cinematics fixture unavailable: %s: %v", name, err)
		}
	}
	open := func(name string) *File {
		t.Helper()
		file, err := Open(path(name))
		if err != nil {
			t.Fatalf("Open %s: %v", name, err)
		}
		return file
	}

	allMessages := open("md_t39_all_messages").Settings()
	if got := allMessages.Messages.Instructions.Literal; !strings.Contains(got, "Scenario Instructions are here") {
		t.Fatalf("instructions literal = %q", got)
	}
	for name, slot := range map[string]MessageSlotSettings{
		"hints":   allMessages.Messages.Hints,
		"victory": allMessages.Messages.Victory,
		"loss":    allMessages.Messages.Loss,
		"history": allMessages.Messages.History,
		"scouts":  allMessages.Messages.Scouts,
	} {
		if slot.Literal == "" {
			t.Fatalf("%s literal empty: %+v", name, slot)
		}
		if slot.StringTableID != -2 || slot.UsesStringTableID || slot.EffectiveSource != "literal" {
			t.Fatalf("%s slot = %+v, want literal with -2 sentinel", name, slot)
		}
	}
	report := Diff(open("md_t38_message_slots"), open("md_t39_all_messages"))
	for _, field := range []string{
		"messages.hints.literal",
		"messages.victory.literal",
		"messages.loss.literal",
		"messages.history.literal",
		"messages.scouts.literal",
	} {
		if !diffHasField(report, field) {
			t.Fatalf("diff missing %s: %+v", field, report.Changes)
		}
	}

	cinematic := open("md_t40_cinematic_pregame").Settings()
	if cinematic.Cinematics.Pregame != "aoeiide_titlevideo.wmv" {
		t.Fatalf("pregame cinematic=%q", cinematic.Cinematics.Pregame)
	}
	report = Diff(open("md_t39_all_messages"), open("md_t40_cinematic_pregame"))
	if !diffHasField(report, "cinematics.pregame") {
		t.Fatalf("diff missing cinematics.pregame: %+v", report.Changes)
	}
}

func TestScenarioDiffPerUnitReferenceChanges(t *testing.T) {
	input := filepath.Join("..", "..", "testdata", "editor-refs", "Regenerate with Grass 1 as terrain seed.aoe2scenario")
	file, err := Open(input)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	sectionIndex, player, unitIndex, unit := firstUnitWithReference(t, file.Units)

	tiny := cloneFileForUnitDiff(file)
	tiny.Units.Sections[sectionIndex].Units[unitIndex].X += unitFloatDiffEpsilon / 2
	report := Diff(file, tiny)
	if diffHasPrefix(report, fmt.Sprintf("units/player_%d.unit_%d.", player, unit.ReferenceID)) {
		t.Fatalf("tiny float jitter produced unit diff: %+v", report.Changes)
	}

	moved := cloneFileForUnitDiff(file)
	moved.Units.Sections[sectionIndex].Units[unitIndex].X += 0.5
	moved.Units.Sections[sectionIndex].Units[unitIndex].Rotation += 0.25
	report = Diff(file, moved)
	for _, field := range []string{
		fmt.Sprintf("units/player_%d.unit_%d.x", player, unit.ReferenceID),
		fmt.Sprintf("units/player_%d.unit_%d.rotation", player, unit.ReferenceID),
	} {
		if !diffHasField(report, field) {
			t.Fatalf("diff missing %s: %+v", field, report.Changes)
		}
	}

	retyped := cloneFileForUnitDiff(file)
	retyped.Units.Sections[sectionIndex].Units[unitIndex].UnitConst++
	retyped.Units.Sections[sectionIndex].Units[unitIndex].Status++
	retyped.Units.Sections[sectionIndex].Units[unitIndex].CaptionStringID = 1234
	report = Diff(file, retyped)
	for _, field := range []string{
		fmt.Sprintf("units/player_%d.unit_%d.unit_const", player, unit.ReferenceID),
		fmt.Sprintf("units/player_%d.unit_%d.status", player, unit.ReferenceID),
		fmt.Sprintf("units/player_%d.unit_%d.caption_string_id", player, unit.ReferenceID),
	} {
		if !diffHasField(report, field) {
			t.Fatalf("diff missing %s: %+v", field, report.Changes)
		}
	}

	added := cloneFileForUnitDiff(file)
	addedUnit := unit
	addedUnit.ReferenceID = maxUnitReference(added.Units) + 1
	addedUnit.X += 1
	added.Units.Sections[sectionIndex].Units = append(added.Units.Sections[sectionIndex].Units, addedUnit)
	added.Units.Sections[sectionIndex].Count++
	added.Units.Total++
	report = Diff(file, added)
	if !diffHasField(report, fmt.Sprintf("units/player_%d.added", player)) {
		t.Fatalf("diff missing added unit: %+v", report.Changes)
	}

	removed := cloneFileForUnitDiff(file)
	removed.Units.Sections[sectionIndex].Units = append(removed.Units.Sections[sectionIndex].Units[:unitIndex], removed.Units.Sections[sectionIndex].Units[unitIndex+1:]...)
	removed.Units.Sections[sectionIndex].Count--
	removed.Units.Total--
	report = Diff(file, removed)
	if !diffHasField(report, fmt.Sprintf("units/player_%d.removed", player)) {
		t.Fatalf("diff missing removed unit: %+v", report.Changes)
	}
}

func TestInflatedBodyFile(t *testing.T) {
	input := filepath.Join("..", "..", "testdata", "editor-refs", "Create scenario, do nothing, click save.aoe2scenario")
	file, err := Open(input)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	body, err := InflatedBodyFile(input)
	if err != nil {
		t.Fatalf("InflatedBodyFile: %v", err)
	}
	if !bytes.Equal(body, file.body) {
		t.Fatalf("inflated body mismatch: got %d bytes, want %d", len(body), len(file.body))
	}
	if len(body) == 0 {
		t.Fatal("inflated body is empty")
	}
	body[0] ^= 0xff
	if bytes.Equal(body, file.body) {
		t.Fatal("InflatedBodyFile returned aliased body")
	}
}

func TestScenarioByteDiffCalibrationFixtures(t *testing.T) {
	dir := scenarioProjectPath("Mandala", "felt-compendium", "kit_settings_calibration")
	postImp := filepath.Join(dir, "rof_p1_postimperial.aoe2scenario")
	castle := filepath.Join(dir, "rof_p1_castle.aoe2scenario")
	for _, path := range []string{postImp, castle} {
		if _, err := os.Stat(path); err != nil {
			t.Skipf("settings calibration fixture unavailable: %s: %v", path, err)
		}
	}
	report, err := ByteDiffFiles(postImp, castle, ByteDiffOptions{})
	if err != nil {
		t.Fatalf("ByteDiffFiles: %v", err)
	}
	if report.Mode != "in-place" {
		t.Fatalf("mode=%q, want in-place", report.Mode)
	}
	if len(report.Changes) != 2 {
		t.Fatalf("changes=%+v, want the cursor and authored changes; suppressed=%+v", report.Changes, report.NoiseSuppressed)
	}
	if report.Changes[0].Start != 0 || report.Changes[0].Length != 1 || report.Changes[0].Decodes.Uint8 == nil || report.Changes[0].Decodes.Uint8.Before != 172 || report.Changes[0].Decodes.Uint8.After != 174 {
		t.Fatalf("cursor change=%+v, want body offset 0 172->174", report.Changes[0])
	}
	change := report.Changes[1]
	if change.Start != 0x5046 || change.Length != 1 {
		t.Fatalf("change=%+v, want one byte at 0x5046", change)
	}
	if change.Decodes.Uint8 == nil || change.Decodes.Uint8.Before != 6 || change.Decodes.Uint8.After != 4 {
		t.Fatalf("uint8 decode=%+v, want 6->4", change.Decodes.Uint8)
	}
	if change.Decodes.FieldLE == nil || change.Decodes.FieldLE.Before != 6 || change.Decodes.FieldLE.After != 4 {
		t.Fatalf("field int32 decode=%+v, want 6->4", change.Decodes.FieldLE)
	}
	if !suppressedKind(report.NoiseSuppressed, "likely_per_player_nonce") {
		t.Fatalf("missing nonce suppression: %+v", report.NoiseSuppressed)
	}
}

func TestScenarioByteDiffDoesNotSuppressTriggerSectionChanges(t *testing.T) {
	dir := scenarioProjectPath("Mandala", "felt-compendium", "kit_settings_calibration")
	before := filepath.Join(dir, "md_t21_trigger0_added.aoe2scenario")
	after := filepath.Join(dir, "md_t22_trigger_make_header.aoe2scenario")
	for _, path := range []string{before, after} {
		if _, err := os.Stat(path); err != nil {
			t.Skipf("trigger calibration fixture unavailable: %s: %v", path, err)
		}
	}
	report, err := ByteDiffFiles(before, after, ByteDiffOptions{})
	if err != nil {
		t.Fatalf("ByteDiffFiles: %v", err)
	}
	if len(report.Changes) == 0 {
		t.Fatalf("trigger changes were suppressed: suppressed=%+v", report.NoiseSuppressed)
	}
	for _, suppressed := range report.NoiseSuppressed {
		if suppressed.Kind == "likely_per_player_nonce" && suppressed.Start >= 121000 {
			t.Fatalf("trigger-region nonce suppression survived: %+v", report.NoiseSuppressed)
		}
	}
}

func TestScenarioAllFieldsDiffAlignsRepeatedVictoryBlocks(t *testing.T) {
	dir := scenarioAoe2DEPath("runs", "cal159", "round3")
	before := filepath.Join(dir, "Clean v159.7.aoe2scenario")
	after := filepath.Join(dir, "Clean v159.8.aoe2scenario")
	for _, path := range []string{before, after} {
		if _, err := os.Stat(path); err != nil {
			t.Skipf("all-fields calibration fixture unavailable: %s: %v", path, err)
		}
	}
	report, err := DiffFilesWithOptions(before, after, DiffOptions{AllFields: true})
	if err != nil {
		t.Fatalf("DiffFilesWithOptions: %v", err)
	}
	paths := make(map[string]StructureFieldDiff, len(report.Fields))
	for _, field := range report.Fields {
		paths[field.Path] = field
	}
	for _, path := range []string{
		"DataHeader.unknown",
		"GlobalVictory.all_custom_conditions_required",
		"Units.player_data_3[0].unknown_structure_3[0]",
		"Units.player_data_3[0].unknown_structure_3[1]",
		"Units.player_data_3[7].unknown_structure_3[1]",
	} {
		if _, ok := paths[path]; !ok {
			t.Fatalf("missing field path %q in %+v", path, report.Fields)
		}
	}
	if got := len(paths["Units.player_data_3[0].unknown_structure_3[0]"].AfterHex) + len(paths["Units.player_data_3[0].unknown_structure_3[1]"].AfterHex); got != 88*2 {
		t.Fatalf("victory block hex length = %d, want %d", got, 88*2)
	}
	for path := range paths {
		if path == "DataHeader.next_unit_id_to_place" {
			t.Fatalf("save cursor leaked through suppression: %q", path)
		}
	}
}

func suppressedKind(runs []SuppressedRun, kind string) bool {
	for _, run := range runs {
		if run.Kind == kind {
			return true
		}
	}
	return false
}

func TestDiffSurfacesPlayerSettingsFields(t *testing.T) {
	input := filepath.Join("..", "..", "testdata", "editor-refs", "Create scenario, do nothing, click save.aoe2scenario")
	before, err := Open(input)
	if err != nil {
		t.Fatalf("Open before: %v", err)
	}
	after, err := Open(input)
	if err != nil {
		t.Fatalf("Open after: %v", err)
	}
	dataHeader := after.root.section("DataHeader")
	ensureRepeatedIntField(t, dataHeader, "per_player_lock_personality", 1, 1)
	setRepeatedIntField(t, dataHeader, "string_table_player_names", 1, 44)
	ensureRepeatedIntField(t, after.root.section("Options"), "per_player_starting_age", 1, 4)
	ensureRepeatedIntField(t, after.root.section("Options"), "per_player_base_priority", 1, 6)
	setRepeatedBigEndianU32Raw(t, after.root.section("Map"), "per_player_population_cap", 1, 180)
	players := dataHeader.list("player_data_1")
	setStringFieldForTest(t, players[1], "architecture_set", "MAGYAR-CIV")
	resources := after.root.section("PlayerDataTwo").list("resources")
	setIntFieldForTest(t, resources[1], "player_color", 7)
	setIntFieldForTest(t, resources[1], "gold", 1234)
	diplomacy := after.root.section("Diplomacy")
	alliedVictory := diplomacy.field("per_player_allied_victory")
	setRepeatedNodeValue(t, alliedVictory, 1, 1)
	rows := diplomacy.list("per_player_diplomacy")
	stances := rows[1].intList("stance_with_each_player")
	if len(stances) <= 2 {
		t.Fatalf("P1 diplomacy stance list too short: %v", stances)
	}
	setRepeatedIntField(t, rows[1], "stance_with_each_player", 2, stances[2]+1)
	after.Players = after.root.playerInfo()

	report := Diff(before, after)
	for _, field := range []string{
		"player_1.lock_personality",
		"player_1.name_string_id",
		"player_1.architecture",
		"player_1.starting_age",
		"player_1.color",
		"player_1.base_priority",
		"player_1.population_limit",
		"player_1.resources.gold",
		"player_1.allied_victory",
		"player_1.diplomacy",
	} {
		if !diffHasField(report, field) {
			t.Fatalf("diff missing %s: %+v", field, report.Changes)
		}
	}
}

func TestDiffLabelsOneEditorPlayerAcrossSectionBases(t *testing.T) {
	dir := scenarioAoe2DEPath("runs", "cal159", "round2")
	beforePath := filepath.Join(dir, "Clean v159.4.aoe2scenario")
	afterPath := filepath.Join(dir, "Clean v159.5.aoe2scenario")
	if _, err := os.Stat(beforePath); err != nil {
		t.Skipf("round-2 calibration fixture unavailable: %v", err)
	}
	if _, err := os.Stat(afterPath); err != nil {
		t.Skipf("round-2 calibration fixture unavailable: %v", err)
	}
	before, err := Open(beforePath)
	if err != nil {
		t.Fatalf("Open before: %v", err)
	}
	after, err := Open(afterPath)
	if err != nil {
		t.Fatalf("Open after: %v", err)
	}

	var aiNameRefs []PlayerReference
	for _, change := range Diff(before, after).Changes {
		if change.Kind != "players" {
			continue
		}
		if change.Player != "P1" {
			t.Fatalf("player change mislabeled as %q: %+v", change.Player, change)
		}
		if strings.HasSuffix(change.Field, ".ai_name") {
			if change.SectionIndex == nil {
				t.Fatalf("AI-name change missing section index: %+v", change)
			}
			aiNameRefs = append(aiNameRefs, PlayerReference{
				Player:       change.Player,
				SectionIndex: *change.SectionIndex,
				IndexBase:    change.IndexBase,
			})
		}
	}
	if len(aiNameRefs) != 2 {
		t.Fatalf("got %d P1 AI-name changes, want two raw observations: %+v", len(aiNameRefs), aiNameRefs)
	}
	if aiNameRefs[0].SectionIndex != 0 || aiNameRefs[1].SectionIndex != 1 {
		t.Fatalf("P1 AI-name raw indexes = %+v, want 0 and 1", aiNameRefs)
	}
	if aiNameRefs[0].IndexBase == aiNameRefs[1].IndexBase {
		t.Fatalf("P1 AI-name references lost index-base distinction: %+v", aiNameRefs)
	}
}

func TestDiffSurfacesDiplomacyMirrors(t *testing.T) {
	dir := scenarioAoe2DEPath("runs", "cal159", "round3")
	beforePath := filepath.Join(dir, "Clean v159.2.aoe2scenario")
	afterPath := filepath.Join(dir, "Clean v159.3.aoe2scenario")
	if _, err := os.Stat(beforePath); err != nil {
		t.Skipf("round-3 calibration fixture unavailable: %v", err)
	}
	if _, err := os.Stat(afterPath); err != nil {
		t.Skipf("round-3 calibration fixture unavailable: %v", err)
	}
	before, err := Open(beforePath)
	if err != nil {
		t.Fatalf("Open before: %v", err)
	}
	after, err := Open(afterPath)
	if err != nil {
		t.Fatalf("Open after: %v", err)
	}
	report := Diff(before, after)
	want := map[string]bool{
		"units.player.P1.diplomacy_for_interaction.P2": true,
		"units.player.P1.diplomacy_for_ai_system.P2":   true,
	}
	for _, change := range report.Changes {
		delete(want, change.Field)
	}
	if len(want) != 0 {
		t.Fatalf("missing diplomacy mirror changes: %v; all changes=%+v", want, report.Changes)
	}
}

func TestSetDiplomacyUpdatesAllMirrors(t *testing.T) {
	input := filepath.Join(scenarioAoe2DEPath("runs", "cal159", "round3"), "Clean v159.2.aoe2scenario")
	wantPath := filepath.Join(scenarioAoe2DEPath("runs", "cal159", "round3"), "Clean v159.3.aoe2scenario")
	wantFinalPath := filepath.Join(scenarioAoe2DEPath("runs", "cal159", "round3"), "Clean v159.4.aoe2scenario")
	if _, err := os.Stat(input); err != nil {
		t.Skipf("round-3 calibration fixture unavailable: %v", err)
	}
	if _, err := os.Stat(wantPath); err != nil {
		t.Skipf("round-3 calibration fixture unavailable: %v", err)
	}
	if _, err := os.Stat(wantFinalPath); err != nil {
		t.Skipf("round-3 calibration fixture unavailable: %v", err)
	}
	file, err := Open(input)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if err := file.SetDiplomacy(DiplomacyRecipe{From: 0, To: 1, Stance: 0}); err != nil {
		t.Fatalf("SetDiplomacy: %v", err)
	}
	mainRow := file.root.section("Diplomacy").list("per_player_diplomacy")[0].intList("stance_with_each_player")
	mirror := file.root.section("Units").list("player_data_3")[0]
	interaction := mirror.intList("diplomacy_for_interaction")
	aiSystem := mirror.intList("diplomacy_for_ai_system")
	if mainRow[1] != 0 || interaction[2] != 0 || aiSystem[2] != 2 {
		t.Fatalf("ally mirrors = main[%d] interaction[%d] ai[%d], want 0/0/2", mainRow[1], interaction[2], aiSystem[2])
	}
	want, err := Open(wantPath)
	if err != nil {
		t.Fatalf("Open expected editor save: %v", err)
	}
	for _, section := range []string{"Diplomacy", "Units"} {
		gotSection := file.rawSection(section)
		wantSection := want.rawSection(section)
		if !bytes.Equal(gotSection, wantSection) {
			t.Fatalf("ally writer %s section differs from editor save: got %d bytes want %d", section, len(gotSection), len(wantSection))
		}
	}
	if err := file.SetDiplomacy(DiplomacyRecipe{From: 0, To: 1, Stance: 1}); err != nil {
		t.Fatalf("SetDiplomacy neutral: %v", err)
	}
	interaction = mirror.intList("diplomacy_for_interaction")
	aiSystem = mirror.intList("diplomacy_for_ai_system")
	mainRow = file.root.section("Diplomacy").list("per_player_diplomacy")[0].intList("stance_with_each_player")
	if mainRow[1] != 1 || interaction[2] != 1 || aiSystem[2] != 3 {
		t.Fatalf("neutral mirrors = main[%d] interaction[%d] ai[%d], want 1/1/3", mainRow[1], interaction[2], aiSystem[2])
	}
	lockTeams := true
	allowChoose := false
	randomStarts := true
	maxTeams := 2
	if err := file.SetDiplomacyOptions(DiplomacyOptionsRecipe{
		LockTeams:               &lockTeams,
		AllowPlayersChooseTeams: &allowChoose,
		RandomStartPoints:       &randomStarts,
		MaxNumberOfTeams:        &maxTeams,
		AlliedVictory:           []AlliedVictoryRecipe{{Player: 0, Enabled: true}},
	}); err != nil {
		t.Fatalf("SetDiplomacyOptions allied victory: %v", err)
	}
	if got := file.root.section("Diplomacy").intList("per_player_allied_victory")[0]; got != 1 {
		t.Fatalf("primary allied victory=%d, want 1", got)
	}
	if got, _ := mirror.intValue("aok_allied_victory"); got != 1 {
		t.Fatalf("mirror allied victory=%d, want 1", got)
	}
	wantFinal, err := Open(wantFinalPath)
	if err != nil {
		t.Fatalf("Open final editor save: %v", err)
	}
	for _, section := range []string{"Diplomacy", "Units"} {
		gotSection := file.rawSection(section)
		wantSection := wantFinal.rawSection(section)
		if !bytes.Equal(gotSection, wantSection) {
			t.Fatalf("neutral/options writer %s section differs from editor save: got %d bytes want %d", section, len(gotSection), len(wantSection))
		}
	}
}

func TestLintDetectsDiplomacyMirrorMismatch(t *testing.T) {
	input := filepath.Join(scenarioAoe2DEPath("runs", "cal159", "round3"), "Clean v159.2.aoe2scenario")
	if _, err := os.Stat(input); err != nil {
		t.Skipf("round-3 calibration fixture unavailable: %v", err)
	}
	file, err := Open(input)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	row := file.root.section("Units").list("player_data_3")[0]
	interaction := row.intList("diplomacy_for_interaction")
	interaction[2] = 0
	if err := setIntListField(row, "diplomacy_for_interaction", "u8", interaction); err != nil {
		t.Fatalf("mutate interaction mirror: %v", err)
	}
	if !hasLintCode(file.Lint().Issues, "diplomacy_mirror_mismatch") {
		t.Fatalf("missing diplomacy_mirror_mismatch: %+v", file.Lint().Issues)
	}
}

func setRepeatedIntField(t *testing.T, node *parsedNode, field string, index int, value int) {
	t.Helper()
	if node == nil {
		t.Fatalf("nil node for field %s", field)
	}
	setRepeatedNodeValue(t, node.field(field), index, value)
}

func setRepeatedBigEndianU32Raw(t *testing.T, node *parsedNode, field string, index int, value int) {
	t.Helper()
	if node == nil {
		t.Fatalf("nil node for field %s", field)
	}
	child := node.field(field)
	if child == nil {
		t.Fatalf("missing field %s", field)
	}
	need := (index + 1) * 4
	if len(child.Raw) < need {
		t.Fatalf("%s raw len %d too short for index %d", field, len(child.Raw), index)
	}
	binary.BigEndian.PutUint32(child.Raw[index*4:], uint32(value))
	values, ok := child.Value.([]any)
	if ok && index < len(values) {
		values[index] = uint32(value)
		child.Value = values
	}
}

func ensureRepeatedIntField(t *testing.T, node *parsedNode, field string, index int, value int) {
	t.Helper()
	if node == nil {
		t.Fatalf("nil node for field %s", field)
	}
	child := node.field(field)
	if child == nil {
		values := make([]any, index+1)
		for i := range values {
			values[i] = int32(0)
		}
		child = &parsedNode{Name: field, Value: values}
		node.Fields = append(node.Fields, child)
	}
	values, ok := child.Value.([]any)
	if !ok {
		t.Fatalf("%s value type = %T, want []any", child.Name, child.Value)
	}
	for len(values) <= index {
		values = append(values, int32(0))
	}
	child.Value = values
	setRepeatedNodeValue(t, child, index, value)
}

func setRepeatedNodeValue(t *testing.T, field *parsedNode, index int, value int) {
	t.Helper()
	if field == nil {
		t.Fatal("nil repeated field")
	}
	values, ok := field.Value.([]any)
	if !ok {
		t.Fatalf("%s value type = %T, want []any", field.Name, field.Value)
	}
	if index < 0 || index >= len(values) {
		t.Fatalf("%s index %d out of range 0..%d", field.Name, index, len(values)-1)
	}
	values[index] = sameNumericType(values[index], value)
	field.Value = values
	field.Raw = nil
}

func setIntFieldForTest(t *testing.T, node *parsedNode, field string, value int) {
	t.Helper()
	if node == nil {
		t.Fatalf("nil node for field %s", field)
	}
	child := node.field(field)
	if child == nil {
		t.Fatalf("missing field %s", field)
	}
	child.Value = sameNumericType(child.Value, value)
	child.Raw = nil
}

func setStringFieldForTest(t *testing.T, node *parsedNode, field string, value string) {
	t.Helper()
	if node == nil {
		t.Fatalf("nil node for field %s", field)
	}
	child := node.field(field)
	if child == nil {
		t.Fatalf("missing field %s", field)
	}
	child.Value = value
	child.Raw = nil
}

func sameNumericType(old any, value int) any {
	switch old.(type) {
	case int8:
		return int8(value)
	case int16:
		return int16(value)
	case int32:
		return int32(value)
	case uint8:
		return uint8(value)
	case uint16:
		return uint16(value)
	case uint32:
		return uint32(value)
	default:
		return value
	}
}

func diffHasField(report DiffReport, field string) bool {
	for _, change := range report.Changes {
		if change.Field == field {
			return true
		}
		if legacyPlayerFieldMatches(change.Field, field) {
			return true
		}
	}
	return false
}

func legacyPlayerFieldMatches(got, want string) bool {
	parts := strings.SplitN(strings.TrimPrefix(got, "player."), ".", 2)
	if len(parts) != 2 {
		return false
	}
	label, field := parts[0], parts[1]
	var number int
	if _, err := fmt.Sscanf(label, "P%d", &number); err != nil {
		return false
	}
	return want == fmt.Sprintf("player_%d.%s", number-1, field)
}

func diffHasPrefix(report DiffReport, prefix string) bool {
	for _, change := range report.Changes {
		if strings.HasPrefix(change.Field, prefix) {
			return true
		}
	}
	return false
}

func cloneFileForUnitDiff(file *File) *File {
	clone := *file
	clone.Units = cloneUnitInfo(file.Units)
	return &clone
}

func cloneUnitInfo(info *UnitInfo) *UnitInfo {
	if info == nil {
		return nil
	}
	clone := *info
	clone.Sections = make([]PlayerUnitsInfo, len(info.Sections))
	for i, section := range info.Sections {
		clone.Sections[i] = section
		clone.Sections[i].Units = append([]UnitSummary(nil), section.Units...)
	}
	return &clone
}

func firstUnitWithReference(t *testing.T, info *UnitInfo) (int, int, int, UnitSummary) {
	t.Helper()
	if info == nil {
		t.Fatal("nil units")
	}
	for sectionIndex, section := range info.Sections {
		for unitIndex, unit := range section.Units {
			if unit.ReferenceID != 0 {
				return sectionIndex, section.Player, unitIndex, unit
			}
		}
	}
	t.Fatal("fixture has no unit with non-zero reference_id")
	return 0, 0, 0, UnitSummary{}
}

func maxUnitReference(info *UnitInfo) int {
	maxRef := 0
	if info == nil {
		return maxRef
	}
	for _, section := range info.Sections {
		for _, unit := range section.Units {
			if unit.ReferenceID > maxRef {
				maxRef = unit.ReferenceID
			}
		}
	}
	return maxRef
}

func TestSetDiplomacyOptionsRecipe(t *testing.T) {
	input := testfixtures.Path(t, "save-analysis/diagnostics/Replay_Transparency_Diagnostic_v2b_flagged.aoe2scenario")
	if _, err := os.Stat(input); err != nil {
		t.Skipf("diagnostic fixture unavailable: %v", err)
	}
	file, err := Open(input)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	lockTeams := true
	allowChoose := false
	randomStarts := true
	maxTeams := 2
	recipe := DiplomacyOptionsRecipe{
		LockTeams:               &lockTeams,
		AllowPlayersChooseTeams: &allowChoose,
		RandomStartPoints:       &randomStarts,
		MaxNumberOfTeams:        &maxTeams,
		AlliedVictory: []AlliedVictoryRecipe{
			{Player: 1, Enabled: true},
			{Player: 2, Enabled: false},
		},
	}
	if _, err := file.Plan(Recipe{DiplomacyOptions: &recipe}); err != nil {
		t.Fatalf("Plan diplomacy options: %v", err)
	}
	if err := file.SetDiplomacyOptions(recipe); err != nil {
		t.Fatalf("SetDiplomacyOptions: %v", err)
	}
	settings := file.Settings()
	if !settings.Diplomacy.LockTeams || settings.Diplomacy.AllowPlayersChooseTeams || !settings.Diplomacy.RandomStartPoints || settings.Diplomacy.MaxNumberOfTeams != maxTeams {
		t.Fatalf("diplomacy options readback = %+v", settings.Diplomacy)
	}
	if !settings.Diplomacy.AlliedVictory[1] || settings.Diplomacy.AlliedVictory[2] {
		t.Fatalf("allied victory readback = %+v", settings.Diplomacy.AlliedVictory[:4])
	}
	badMax := 17
	if err := file.SetDiplomacyOptions(DiplomacyOptionsRecipe{MaxNumberOfTeams: &badMax}); err == nil {
		t.Fatal("SetDiplomacyOptions accepted max_number_of_teams above 16")
	}
}

func TestSetPlayerRecipeExtendedFields(t *testing.T) {
	input := testfixtures.Path(t, "save-analysis/diagnostics/Replay_Transparency_Diagnostic_v2b_flagged.aoe2scenario")
	if _, err := os.Stat(input); err != nil {
		t.Skipf("diagnostic fixture unavailable: %v", err)
	}
	file, err := Open(input)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	active := true
	human := false
	tribe := "Dex Tribe"
	civ := "MAYANS"
	lockCiv := true
	lockPersonality := true
	aiName := "PlaygroundHelper"
	aiType := 2
	recipe := PlayerRecipe{
		Player:           2,
		Active:           &active,
		Human:            &human,
		TribeName:        &tribe,
		Civilization:     &civ,
		LockCivilization: &lockCiv,
		LockPersonality:  &lockPersonality,
		AIName:           &aiName,
		AIType:           &aiType,
	}
	if _, err := file.Plan(Recipe{Players: []PlayerRecipe{recipe}}); err != nil {
		t.Fatalf("Plan player: %v", err)
	}
	if err := file.SetPlayer(recipe); err != nil {
		t.Fatalf("SetPlayer: %v", err)
	}
	settings := file.Settings()
	player := settings.Players[2]
	if !player.Active || player.Human || player.TribeName != tribe || player.Civilization != civ ||
		!player.LockCivilization || !player.LockPersonality || player.AIName != aiName || player.AIType != aiType {
		t.Fatalf("player settings readback = %+v", player)
	}
	out := filepath.Join(t.TempDir(), "player_fields.aoe2scenario")
	if err := file.Write(out); err != nil {
		t.Fatalf("Write: %v", err)
	}
	reopened, err := Open(out)
	if err != nil {
		t.Fatalf("Open written player fields: %v", err)
	}
	reopenedPlayer := reopened.Settings().Players[2]
	if reopenedPlayer.TribeName != tribe || reopenedPlayer.Civilization != civ || !reopenedPlayer.LockCivilization ||
		!reopenedPlayer.LockPersonality || reopenedPlayer.AIName != aiName || reopenedPlayer.AIType != aiType {
		t.Fatalf("reopened player settings = %+v", reopenedPlayer)
	}
}

func TestParserRejectsImpossibleRepeatsBeforeAllocation(t *testing.T) {
	p := parser{data: make([]byte, 4)}
	_, err := p.parseNode("Huge", SectionSpec{Fields: []FieldSpec{
		{Name: "huge", Type: "u32", Repeat: maxScenarioFieldRepeat + 1},
	}}, nil)
	if err == nil || !strings.Contains(err.Error(), "exceeds parser safety limit") {
		t.Fatalf("huge repeat error = %v, want safety limit", err)
	}

	p = parser{data: make([]byte, 4)}
	_, err = p.parseNode("Overrun", SectionSpec{Fields: []FieldSpec{
		{Name: "too_many", Type: "u32", Repeat: 2},
	}}, nil)
	if err == nil || !strings.Contains(err.Error(), "exceeds remaining bytes") {
		t.Fatalf("overrun repeat error = %v, want remaining-bytes guard", err)
	}
}

func TestPlanRecipeValidatesRemoveUnitSafety(t *testing.T) {
	input := testfixtures.Path(t, "save-analysis/diagnostics/Replay_Transparency_Diagnostic_v2b_flagged.aoe2scenario")
	if _, err := os.Stat(input); err != nil {
		t.Skipf("diagnostic fixture unavailable: %v", err)
	}
	file, err := Open(input)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	refID := 990901
	x := 20.5
	y := 20.5
	if err := file.AddUnit(UnitRecipe{Op: "add_unit", Player: 1, UnitConst: 83, X: &x, Y: &y, ReferenceID: &refID}); err != nil {
		t.Fatalf("AddUnit: %v", err)
	}
	if err := file.AddTrigger(TriggerRecipe{
		Op:   "add_trigger",
		Name: "plan blocker",
		Effects: []EffectRecipe{
			{Op: "kill_object", SourcePlayer: intPtr(1), SelectedObjectIDs: []int{refID}},
		},
	}); err != nil {
		t.Fatalf("AddTrigger: %v", err)
	}
	_, err = file.Plan(Recipe{Units: []UnitRecipe{{Op: "remove_unit", ReferenceID: &refID}}})
	if err == nil || !strings.Contains(err.Error(), "selected_object_ids[0] references it") {
		t.Fatalf("Plan remove_unit error = %v, want selected-object reference refusal", err)
	}
	missing := 999999
	_, err = file.Plan(Recipe{Units: []UnitRecipe{{Op: "remove_unit", ReferenceID: &missing}}})
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("Plan missing remove_unit error = %v, want not found", err)
	}
}

func TestPlanRecipeValidatesRemoveTriggerSafety(t *testing.T) {
	input := testfixtures.Path(t, "save-analysis/diagnostics/Replay_Transparency_Diagnostic_v2b_flagged.aoe2scenario")
	if _, err := os.Stat(input); err != nil {
		t.Skipf("diagnostic fixture unavailable: %v", err)
	}
	file, err := Open(input)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	targetTrigger := len(file.root.section("Triggers").list("trigger_data"))
	if err := file.AddTrigger(TriggerRecipe{Op: "add_trigger", Name: "plan trigger target"}); err != nil {
		t.Fatalf("AddTrigger target: %v", err)
	}
	if err := file.AddTrigger(TriggerRecipe{
		Op:   "add_trigger",
		Name: "plan trigger blocker",
		Effects: []EffectRecipe{
			{Op: "activate_trigger", TriggerID: &targetTrigger},
		},
	}); err != nil {
		t.Fatalf("AddTrigger blocker: %v", err)
	}
	_, err = file.Plan(Recipe{Triggers: []TriggerRecipe{{Op: "remove_trigger", TargetIndex: &targetTrigger}}})
	if err == nil || !strings.Contains(err.Error(), "blocked by 1 reference") {
		t.Fatalf("Plan remove_trigger error = %v, want blocker", err)
	}
}

func TestValidateMapRect(t *testing.T) {
	ok := MapRecipe{Op: "set_terrain_rect", X1: 1, Y1: 2, X2: 3, Y2: 4}
	if err := validateMapRect(ok, 10, 10); err != nil {
		t.Fatalf("validateMapRect valid: %v", err)
	}
	reversed := MapRecipe{Op: "set_terrain_rect", X1: 3, Y1: 2, X2: 1, Y2: 4}
	if err := validateMapRect(reversed, 10, 10); err == nil {
		t.Fatal("validateMapRect accepted reversed rectangle")
	}
	outside := MapRecipe{Op: "set_terrain_rect", X1: 8, Y1: 8, X2: 10, Y2: 9}
	if err := validateMapRect(outside, 10, 10); err == nil {
		t.Fatal("validateMapRect accepted out-of-bounds rectangle")
	}
}

func TestTerrainGridRecipeInlineAndGridFile(t *testing.T) {
	data, err := BlankScenarioSeedBytes()
	if err != nil {
		t.Fatalf("BlankScenarioSeedBytes: %v", err)
	}
	file, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse blank seed: %v", err)
	}
	inlineJSON := []byte(`{
		"map": [{
			"op": "terrain_grid",
			"x1": 3,
			"y1": 4,
			"width": 2,
			"height": 2,
			"terrain_id": [7, 8, 9, 10],
			"layer": [-1, 11, 12, 13],
			"elevation": [1, 2, 3, 4]
		}]
	}`)
	var inline Recipe
	if err := json.Unmarshal(inlineJSON, &inline); err != nil {
		t.Fatalf("Unmarshal inline terrain_grid: %v", err)
	}
	if got := len(inline.Map[0].TerrainIDGrid); got != 4 {
		t.Fatalf("inline TerrainIDGrid length = %d, want 4", got)
	}
	marshaled, err := json.Marshal(inline)
	if err != nil {
		t.Fatalf("Marshal inline terrain_grid: %v", err)
	}
	if !strings.Contains(string(marshaled), `"terrain_id":[7,8,9,10]`) || !strings.Contains(string(marshaled), `"layer":[-1,11,12,13]`) {
		t.Fatalf("marshaled terrain_grid lost grid arrays: %s", marshaled)
	}
	plan, err := file.Plan(inline)
	if err != nil {
		t.Fatalf("Plan inline terrain_grid: %v", err)
	}
	if plan.MapTilesChanged != 4 {
		t.Fatalf("inline plan MapTilesChanged = %d, want 4", plan.MapTilesChanged)
	}
	if err := file.ApplyRecipe(inline); err != nil {
		t.Fatalf("ApplyRecipe inline terrain_grid: %v", err)
	}

	tmp := t.TempDir()
	gridPath := filepath.Join(tmp, "grid.json")
	if err := os.WriteFile(gridPath, []byte(`{"width":2,"height":1,"terrain_id":[50,51],"layer":[-1,52]}`), 0644); err != nil {
		t.Fatalf("write grid file: %v", err)
	}
	recipePath := filepath.Join(tmp, "recipe.json")
	if err := os.WriteFile(recipePath, []byte(`{"map":[{"op":"terrain_grid","x1":5,"y1":6,"grid_file":"grid.json"}]}`), 0644); err != nil {
		t.Fatalf("write recipe file: %v", err)
	}
	fromFile, err := LoadRecipe(recipePath)
	if err != nil {
		t.Fatalf("LoadRecipe grid_file: %v", err)
	}
	plan, err = file.Plan(fromFile)
	if err != nil {
		t.Fatalf("Plan grid_file terrain_grid: %v", err)
	}
	if plan.MapTilesChanged != 2 {
		t.Fatalf("grid_file plan MapTilesChanged = %d, want 2", plan.MapTilesChanged)
	}
	if err := file.ApplyRecipe(fromFile); err != nil {
		t.Fatalf("ApplyRecipe grid_file terrain_grid: %v", err)
	}

	report, err := file.Terrain(TerrainOptions{IncludeTiles: true})
	if err != nil {
		t.Fatalf("Terrain readback: %v", err)
	}
	tiles := map[[2]int]TerrainTile{}
	for _, tile := range report.Tiles {
		tiles[[2]int{tile.X, tile.Y}] = tile
	}
	assertTile := func(x, y, terrainID, layer, elevation int) {
		t.Helper()
		tile, ok := tiles[[2]int{x, y}]
		if !ok {
			t.Fatalf("missing tile (%d,%d)", x, y)
		}
		if tile.TerrainID != terrainID || tile.Layer != layer || tile.Elevation != elevation {
			t.Fatalf("tile (%d,%d) = terrain %d layer %d elevation %d, want terrain %d layer %d elevation %d",
				x, y, tile.TerrainID, tile.Layer, tile.Elevation, terrainID, layer, elevation)
		}
	}
	assertTile(3, 4, 7, -1, 1)
	assertTile(4, 4, 8, 11, 2)
	assertTile(3, 5, 9, 12, 3)
	assertTile(4, 5, 10, 13, 4)
	assertTile(5, 6, 50, -1, 0)
	assertTile(6, 6, 51, 52, 0)
}
