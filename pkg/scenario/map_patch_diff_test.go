package scenario

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMapPatchDiffRecipeRoundTripsMapArt(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "base.aoe2scenario")
	first := filepath.Join(dir, "first.aoe2scenario")
	after := filepath.Join(dir, "after.aoe2scenario")
	patched := filepath.Join(dir, "patched.aoe2scenario")
	if _, err := WriteBlankScenarioFile(base, BlankOptions{PlayerCount: 2, NoStarters: true, MapWidth: 80, MapHeight: 80}); err != nil {
		t.Fatal(err)
	}
	x1, y1 := 10.5, 11.5
	x2, y2 := 20.5, 21.5
	baseRecipe := Recipe{Map: []MapRecipe{{Op: "set_terrain_rect", X1: 2, Y1: 3, X2: 3, Y2: 3, TerrainID: mapPatchIntPtr(5), Elevation: mapPatchIntPtr(2), Layer: mapPatchIntPtr(-1)}},
		Units: []UnitRecipe{{Op: "add_unit", Player: 1, UnitConst: 83, X: &x1, Y: &y1}, {Op: "add_unit", Player: 2, UnitConst: 83, X: &x2, Y: &y2}}}
	if _, err := PatchRecipeFile(base, first, baseRecipe); err != nil {
		t.Fatalf("create first scenario: %v", err)
	}
	firstFile, err := Open(first)
	if err != nil {
		t.Fatal(err)
	}
	if firstFile.Units == nil || firstFile.Units.Total != 2 {
		t.Fatalf("first units = %#v, want 2", firstFile.Units)
	}
	refEdit := firstFile.Units.Sections[1].Units[0].ReferenceID
	refRemove := firstFile.Units.Sections[2].Units[0].ReferenceID
	xMoved, yMoved := 14.25, 15.75
	afterRecipe := Recipe{Map: []MapRecipe{{Op: "set_terrain_rect", X1: 3, Y1: 3, X2: 3, Y2: 3, TerrainID: mapPatchIntPtr(6), Elevation: mapPatchIntPtr(2), Layer: mapPatchIntPtr(-1)}},
		Units: []UnitRecipe{{Op: "remove_unit", ReferenceID: mapPatchIntPtr(refRemove)}, {Op: "edit_unit", ReferenceID: mapPatchIntPtr(refEdit), X: &xMoved, Y: &yMoved}}}
	if _, err := PatchRecipeFile(first, after, afterRecipe); err != nil {
		t.Fatalf("create after scenario: %v", err)
	}

	report, err := MapPatchDiffFiles(first, after)
	if err != nil {
		t.Fatalf("MapPatchDiffFiles: %v", err)
	}
	if !report.Patchable || len(report.UnsupportedChanges) != 0 {
		t.Fatalf("diff not patchable: %+v", report)
	}
	if report.ChangedCells != 1 || report.RemovedUnits != 1 || report.EditedUnits != 1 || report.AddedUnits != 0 {
		t.Fatalf("diff counts = cells %d units +%d/-%d/~%d", report.ChangedCells, report.AddedUnits, report.RemovedUnits, report.EditedUnits)
	}
	if len(report.Recipe.Map) != 1 || report.Recipe.Map[0].X1 != 3 || report.Recipe.Map[0].Y1 != 3 || report.Recipe.Map[0].X2 != 3 || report.Recipe.Map[0].Y2 != 3 {
		t.Fatalf("terrain recipe = %+v", report.Recipe.Map)
	}
	if _, err := PatchRecipeFile(first, patched, report.Recipe); err != nil {
		t.Fatalf("apply generated recipe: %v", err)
	}
	check, err := MapPatchDiffFiles(patched, after)
	if err != nil {
		t.Fatal(err)
	}
	if !check.Patchable || check.ChangedCells != 0 || check.AddedUnits+check.RemovedUnits+check.EditedUnits != 0 {
		t.Fatalf("generated recipe did not round-trip map art: %+v", check)
	}
}

func TestMapPatchDiffRefusesNonMapChanges(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "base.aoe2scenario")
	after := filepath.Join(dir, "after.aoe2scenario")
	if _, err := WriteBlankScenarioFile(base, BlankOptions{PlayerCount: 2, NoStarters: true, MapWidth: 80, MapHeight: 80}); err != nil {
		t.Fatal(err)
	}
	changed := 1
	if _, err := PatchRecipeFile(base, after, Recipe{Players: []PlayerRecipe{{Player: 1, Color: &changed}}}); err != nil {
		t.Fatalf("create settings edit: %v", err)
	}
	report, err := MapPatchDiffFiles(base, after)
	if err != nil {
		t.Fatal(err)
	}
	if report.Patchable || len(report.UnsupportedChanges) == 0 {
		t.Fatalf("non-map edit was accepted: %+v", report)
	}
	if err := WriteMapPatchRecipe(filepath.Join(dir, "must-not-write.json"), report); err == nil {
		t.Fatal("wrote recipe for non-map change")
	}
}

func TestMapPatchDiffIgnoresSaveTimestampButRejectsNoOp(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "base.aoe2scenario")
	after := filepath.Join(dir, "timestamp.aoe2scenario")
	if _, err := WriteBlankScenarioFile(base, BlankOptions{PlayerCount: 2, NoStarters: true, MapWidth: 80, MapHeight: 80}); err != nil {
		t.Fatal(err)
	}
	timestamp := 123456789
	if _, err := PatchRecipeFile(base, after, Recipe{Scenario: &ScenarioRecipe{TimestampOfLastSave: &timestamp}}); err != nil {
		t.Fatal(err)
	}
	report, err := MapPatchDiffFiles(base, after)
	if err != nil {
		t.Fatal(err)
	}
	if !report.Patchable || report.ChangedCells != 0 || report.AddedUnits+report.RemovedUnits+report.EditedUnits != 0 {
		t.Fatalf("timestamp-only save should classify as a patchable no-op: %+v", report)
	}
	if len(report.OpaqueDifferences) != 0 || len(report.IgnoredSaveMetadata) == 0 || report.IgnoredSaveMetadata[0].Path != "FileHeader.timestamp_of_last_save" {
		t.Fatalf("timestamp classification = ignored:%+v opaque:%+v", report.IgnoredSaveMetadata, report.OpaqueDifferences)
	}
	if len(report.UnsupportedChanges) != 0 {
		t.Fatalf("unsupported changes = %v", report.UnsupportedChanges)
	}
}

// Optional real-map regression pair supplied by the repo maintainer. The map
// stays in private scratch storage and is never copied into this public repo.
// Set AOE2KIT_MCB_SDTEST_DIR to the directory containing empty.aoe2scenario,
// ts.aoe2scenario, and ../mcb-0917.aoe2scenario to run it.
func TestMapPatchDiffPrivateMCBSerializationPairs(t *testing.T) {
	dir := os.Getenv("AOE2KIT_MCB_SDTEST_DIR")
	if dir == "" {
		t.Skip("set AOE2KIT_MCB_SDTEST_DIR for the private MCB serialization regression")
	}
	base := filepath.Clean(filepath.Join(dir, "..", "mcb-0917.aoe2scenario"))
	for _, name := range []string{"empty.aoe2scenario", "ts.aoe2scenario"} {
		report, err := MapPatchDiffFiles(base, filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("diff %s: %v", name, err)
		}
		if !report.Patchable || report.ChangedCells != 0 || report.AddedUnits+report.RemovedUnits+report.EditedUnits != 0 {
			t.Errorf("%s classified as map-art change: patchable=%t cells=%d unit ops=%d/%d/%d", name, report.Patchable, report.ChangedCells, report.AddedUnits, report.RemovedUnits, report.EditedUnits)
		}
		if len(report.OpaqueDifferences) != 0 {
			t.Errorf("%s opaque differences: %+v", name, report.OpaqueDifferences)
		}
		foundTimestamp := false
		for _, field := range report.IgnoredSaveMetadata {
			if strings.Contains(field.Path, "timestamp_of_last_save") {
				foundTimestamp = true
			}
		}
		if !foundTimestamp {
			t.Errorf("%s did not report ignored save timestamp: %+v", name, report.IgnoredSaveMetadata)
		}
	}
}

func TestWriteMapPatchRecipeIsJSONAndAtomicOutput(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "recipe.json")
	report := MapPatchDiffReport{Patchable: true, Recipe: Recipe{Map: []MapRecipe{{Op: "set_terrain_rect", X1: 1, Y1: 2, X2: 1, Y2: 2, TerrainID: mapPatchIntPtr(4), Elevation: mapPatchIntPtr(3), Layer: mapPatchIntPtr(-1)}}}}
	if err := WriteMapPatchRecipe(path, report); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var got Recipe
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("generated recipe is invalid JSON: %v", err)
	}
	if len(got.Map) != 1 || got.Map[0].Elevation == nil || *got.Map[0].Elevation != 3 {
		t.Fatalf("decoded recipe = %+v", got)
	}
}
