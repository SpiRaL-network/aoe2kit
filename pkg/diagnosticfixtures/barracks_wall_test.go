package diagnosticfixtures

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"aoe2kit/pkg/scenario"
)

func TestBuildBarracksWall(t *testing.T) {
	dir := t.TempDir()
	report, err := BuildBarracksWall(dir, 20260923)
	if err != nil {
		t.Fatalf("BuildBarracksWall: %v", err)
	}
	if report.ScenarioSHA == "" {
		t.Fatal("scenario SHA is empty")
	}
	file, err := scenario.Open(report.ScenarioPath)
	if err != nil {
		t.Fatalf("open fixture: %v", err)
	}
	if file.PlayerCount != 6 {
		t.Fatalf("player_count = %d, want 6", file.PlayerCount)
	}
	for player := 0; player < 6; player++ {
		if !file.Players[player].Active {
			t.Fatalf("slot %d is inactive", player)
		}
	}
	if file.Units == nil || file.Units.Total != 36 {
		t.Fatalf("units total = %v, want 36", file.Units)
	}
	lint := file.LintWithOptions(scenario.LintOptions{LoadSafety: true})
	for _, issue := range lint.Issues {
		if issue.Severity == "error" {
			t.Fatalf("load-safety error: %s", issue.Message)
		}
	}
	for _, section := range file.Units.Sections {
		if section.Player >= 2 && section.Player <= 6 && len(section.Units) != 5 {
			t.Fatalf("row owner slot %d has %d units, want 5", section.Player, len(section.Units))
		}
	}
}

func TestBarracksWallTreatmentCheckDistinguishesUnknownAttributes(t *testing.T) {
	// The source-level check is covered through the shared XS string builder in
	// the v2 fixture; this test keeps the three-state messages from regressing.
	data := barracksWallV2XS()
	for _, want := range []string{"collision_x APPLIED", "collision_x UNKNOWN", "clearance_x UNKNOWN", "NOT APPLIED"} {
		if !strings.Contains(data, want) {
			t.Fatalf("treatment XS missing %q", want)
		}
	}
}

func TestBuildRyanScratchAB2IsFiveWallFixture(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "RyanScratch.aoe2scenario")
	if _, err := scenario.WriteBlankScenarioFile(base, scenario.BlankOptions{
		PlayerCount: 5, HumanSlots: 1, MapWidth: 168, MapHeight: 168,
		Timestamp: 20260923, ClearTriggers: true, NoStarters: true,
		StartingResources: true, StartingFood: 1000, StartingWood: 1000,
		StartingGold: 1000, StartingStone: 1000, StartingTradeGoods: 1000,
	}); err != nil {
		t.Fatalf("write base: %v", err)
	}
	seed := 1.0
	seedY := 1.0
	seedRecipe := scenario.Recipe{Units: []scenario.UnitRecipe{{Op: "add_unit", Player: 2, UnitConst: barracksWallUnit, X: &seed, Y: &seedY, Status: intPtr(2)}}}
	for _, attribute := range []int{3, 4, 200, 201} {
		timer := 0
		seedRecipe.Triggers = append(seedRecipe.Triggers, scenario.TriggerRecipe{Op: "add_trigger", Name: fmt.Sprintf("Scratch P1 barracks attr %d = 1.0", attribute), Enabled: boolPtr(true), Looping: boolPtr(false), Conditions: []scenario.ConditionRecipe{{Op: "timer", Timer: &timer}}})
	}
	seedPath := filepath.Join(dir, "RyanScratch seeded.aoe2scenario")
	if _, err := scenario.PatchRecipeFile(base, seedPath, seedRecipe); err != nil {
		t.Fatalf("seed base: %v", err)
	}
	report, err := BuildRyanScratchAB2(seedPath, dir, 20260923)
	if err != nil {
		t.Fatalf("BuildRyanScratchAB2: %v", err)
	}
	if report.ScenarioPath == base {
		t.Fatal("AB2 overwrote the base path")
	}
	if _, err := os.Stat(filepath.Join(dir, "RyanScratch AB.aoe2scenario")); !os.IsNotExist(err) {
		t.Fatalf("unexpected RyanScratch AB artifact: %v", err)
	}
	file, err := scenario.Open(report.ScenarioPath)
	if err != nil {
		t.Fatalf("open AB2: %v", err)
	}
	if file.PlayerCount != 5 {
		t.Fatalf("player_count = %d, want 5", file.PlayerCount)
	}
	if file.Units == nil || file.Units.Total != 9 {
		t.Fatalf("units total = %v, want five P1 villagers and four AI keep-alives", file.Units)
	}
	data, err := os.ReadFile(report.XSPath)
	if err != nil {
		t.Fatalf("read XS: %v", err)
	}
	for _, want := range []string{"xsEffectAmount(0, 12, 200, 1.0, 3)", "A2K_RYAN_SCRATCH_AB2", "P3 clearance APPLIED"} {
		if !strings.Contains(string(data), want) {
			t.Fatalf("XS missing %q", want)
		}
	}
}
