package diagnostics

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"aoe2kit/pkg/datcodec"
	"aoe2kit/pkg/scenario"
)

func TestDATCommandSemanticsPackFixture(t *testing.T) {
	datPath := fixtureDAT(t)
	pack, err := BuildDATCommandSemanticsPack(datPath)
	if err != nil {
		t.Fatalf("BuildDATCommandSemanticsPack: %v", err)
	}
	if pack.Version != SemanticsPackVersion || pack.Status == "" {
		t.Fatalf("bad pack header: %+v", pack)
	}
	if got := len(pack.DatRecipe.CreateEffects); got != 18 {
		t.Fatalf("create effects = %d, want 18", got)
	}
	if got := len(pack.DatRecipe.CreateTechs); got != 18 {
		t.Fatalf("create techs = %d, want 18", got)
	}
	if got := len(pack.ScenarioRecipe.Triggers); got < 25 {
		t.Fatalf("scenario triggers = %d, want packed probe", got)
	}
	if got := len(pack.Expected.Lanes); got != 8 {
		t.Fatalf("expected lanes = %d, want 8", got)
	}
	report, err := datcodec.PlanRecipeFile(datPath, pack.DatRecipe)
	if err != nil {
		t.Fatalf("PlanRecipeFile generated recipe: %v", err)
	}
	if !report.Verified || !report.ReadbackOK || !report.PayloadRoundTripOK {
		t.Fatalf("generated DAT recipe not verified: %+v", report)
	}
	if report.AfterEffectCount != pack.GeneratedFrom.BaseEffectCount+18 {
		t.Fatalf("after effects = %d, want %d", report.AfterEffectCount, pack.GeneratedFrom.BaseEffectCount+18)
	}
	if report.AfterTechCount != pack.GeneratedFrom.BaseTechCount+18 {
		t.Fatalf("after techs = %d, want %d", report.AfterTechCount, pack.GeneratedFrom.BaseTechCount+18)
	}
}

func TestWriteDATCommandSemanticsPackFixture(t *testing.T) {
	datPath := fixtureDAT(t)
	out := t.TempDir()
	pack, err := WriteDATCommandSemanticsPack(datPath, out)
	if err != nil {
		t.Fatalf("WriteDATCommandSemanticsPack: %v", err)
	}
	var datRecipe datcodec.Recipe
	readJSON(t, filepath.Join(out, "DAT_COMMAND_SEMANTICS_RECIPE.json"), &datRecipe)
	if len(datRecipe.CreateEffects) != len(pack.DatRecipe.CreateEffects) {
		t.Fatalf("written DAT recipe effects = %d, want %d", len(datRecipe.CreateEffects), len(pack.DatRecipe.CreateEffects))
	}
	var scenRecipe scenario.Recipe
	readJSON(t, filepath.Join(out, "DAT_COMMAND_SEMANTICS_SCEN_RECIPE.json"), &scenRecipe)
	if len(scenRecipe.Triggers) != len(pack.ScenarioRecipe.Triggers) {
		t.Fatalf("written scenario recipe triggers = %d, want %d", len(scenRecipe.Triggers), len(pack.ScenarioRecipe.Triggers))
	}
	var expected ExpectedLedger
	readJSON(t, filepath.Join(out, "DAT_COMMAND_SEMANTICS_EXPECTED.json"), &expected)
	if expected.Version != SemanticsPackVersion {
		t.Fatalf("written expected version = %q", expected.Version)
	}
}

func TestWriteDATCommandSemanticsLocalBuildingFeaturePackFixture(t *testing.T) {
	datPath := fixtureDAT(t)
	out := t.TempDir()
	_, files, err := WriteDATCommandSemanticsPackWithOptions(datPath, out, SemanticsPackOptions{Feature: "local-building-effects"})
	if err != nil {
		t.Fatalf("WriteDATCommandSemanticsPackWithOptions: %v", err)
	}
	if len(files) != 5 {
		t.Fatalf("written files = %d, want base pack plus local-building files: %+v", len(files), files)
	}
	var report LocalBuildingEffectsReport
	readJSON(t, filepath.Join(out, "LOCAL_BUILDING_EFFECTS.json"), &report)
	if report.Feature != "local-building-effects" || len(report.Rows) == 0 {
		t.Fatalf("local-building report = %+v", report)
	}
	if _, err := os.Stat(filepath.Join(out, "LOCAL_BUILDING_EFFECTS.md")); err != nil {
		t.Fatalf("local-building markdown missing: %v", err)
	}
}

func TestWriteDATCommandSemanticsClassScopeFeaturePackFixture(t *testing.T) {
	datPath := fixtureDAT(t)
	out := t.TempDir()
	_, files, err := WriteDATCommandSemanticsPackWithOptions(datPath, out, SemanticsPackOptions{Feature: "class-scope-effects"})
	if err != nil {
		t.Fatalf("WriteDATCommandSemanticsPackWithOptions: %v", err)
	}
	if len(files) != 5 {
		t.Fatalf("written files = %d, want base pack plus class/scope files: %+v", len(files), files)
	}
	var report ClassScopeEffectsReport
	readJSON(t, filepath.Join(out, "CLASS_SCOPE_EFFECTS.json"), &report)
	if report.Feature != "class-scope-effects" || len(report.Rows) == 0 {
		t.Fatalf("class/scope report = %+v", report)
	}
	foundClassTarget := false
	for _, row := range report.Rows {
		if row.UnitID < 0 && row.UnitClassID >= 0 {
			foundClassTarget = true
			break
		}
	}
	if !foundClassTarget {
		t.Fatalf("class/scope report did not include a class-targeted row: %+v", report.Rows[:min(3, len(report.Rows))])
	}
	if _, err := os.Stat(filepath.Join(out, "CLASS_SCOPE_EFFECTS.md")); err != nil {
		t.Fatalf("class/scope markdown missing: %v", err)
	}
}

func fixtureDAT(t *testing.T) string {
	t.Helper()
	root := os.Getenv("AOE2KIT_FIXTURE_ROOT")
	if root == "" {
		t.Skip("AOE2KIT_FIXTURE_ROOT not set")
	}
	path := filepath.Join(root, "reference_aoe2_dump", "dat", "empires2_x2_p1.dat")
	if _, err := os.Stat(path); err != nil {
		t.Skipf("fixture dat missing: %v", err)
	}
	return path
}

func readJSON(t *testing.T, path string, out any) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if err := json.Unmarshal(data, out); err != nil {
		t.Fatalf("unmarshal %s: %v", path, err)
	}
}
