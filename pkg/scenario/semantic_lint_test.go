package scenario

import (
	"path/filepath"
	"testing"
)

func TestSemanticLintRequiresAuthoredStarterForActivePlayers(t *testing.T) {
	path := filepath.Join(t.TempDir(), "empty.aoe2scenario")
	if _, err := WriteBlankScenarioFile(path, BlankOptions{PlayerCount: 2, HumanSlots: 2, NoStarters: true}); err != nil {
		t.Fatal(err)
	}
	report, err := LintFileWithOptions(path, LintOptions{SemanticInvariants: true})
	if err != nil {
		t.Fatal(err)
	}
	if !hasLintCode(report.Issues, "semantic_interactive_fixture_missing_starting_building") {
		t.Fatalf("missing semantic starter finding: %+v", report.Issues)
	}
}

func TestSemanticLintAcceptsAuthoredKeepAliveStarters(t *testing.T) {
	path := filepath.Join(t.TempDir(), "starters.aoe2scenario")
	if _, err := WriteBlankScenarioFile(path, BlankOptions{PlayerCount: 2, HumanSlots: 2, DummyStarters: true}); err != nil {
		t.Fatal(err)
	}
	report, err := LintFileWithOptions(path, LintOptions{SemanticInvariants: true})
	if err != nil {
		t.Fatal(err)
	}
	if hasLintCode(report.Issues, "semantic_interactive_fixture_missing_starting_building") {
		t.Fatalf("starter finding on authored fixture: %+v", report.Issues)
	}
}

func TestSemanticLintRejectsOutpostOnlyKeepAlive(t *testing.T) {
	path := filepath.Join(t.TempDir(), "outpost-only.aoe2scenario")
	if _, err := WriteBlankScenarioFile(path, BlankOptions{PlayerCount: 1, HumanSlots: 1, NoStarters: true}); err != nil {
		t.Fatal(err)
	}
	f, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	x, y := 12.5, 12.5
	if err := f.AddUnits([]UnitRecipe{{Op: "add_unit", Player: 1, UnitConst: 598, X: &x, Y: &y}}); err != nil {
		t.Fatal(err)
	}
	if err := f.Write(path); err != nil {
		t.Fatal(err)
	}
	report, err := LintFileWithOptions(path, LintOptions{SemanticInvariants: true})
	if err != nil {
		t.Fatal(err)
	}
	if !hasLintCode(report.Issues, "semantic_interactive_fixture_missing_starting_building") {
		t.Fatalf("outpost-only fixture was accepted: %+v", report.Issues)
	}
}

func TestSemanticLintRejectsAttackingKeepAliveAndUnconfiguredDiplomacy(t *testing.T) {
	path := filepath.Join(t.TempDir(), "attacking-keep-alive.aoe2scenario")
	if _, err := WriteBlankScenarioFile(path, BlankOptions{PlayerCount: 2, HumanSlots: 2, NoStarters: true}); err != nil {
		t.Fatal(err)
	}
	f, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	x, y := 12.5, 12.5
	if err := f.AddUnits([]UnitRecipe{{Op: "add_unit", Player: 1, UnitConst: 79, X: &x, Y: &y}}); err != nil {
		t.Fatal(err)
	}
	if err := f.Write(path); err != nil {
		t.Fatal(err)
	}
	report, err := LintFileWithOptions(path, LintOptions{SemanticInvariants: true})
	if err != nil {
		t.Fatal(err)
	}
	if !hasLintCode(report.Issues, "semantic_interactive_fixture_attacking_keep_alive") {
		t.Fatalf("missing attacking keep-alive finding: %+v", report.Issues)
	}
	if !hasLintCode(report.Issues, "semantic_interactive_fixture_diplomacy_unconfigured") {
		t.Fatalf("missing diplomacy finding: %+v", report.Issues)
	}
}

func TestSemanticLintRejectsUndefinedAndParameterizedScriptCalls(t *testing.T) {
	path := filepath.Join(t.TempDir(), "script-call.aoe2scenario")
	if _, err := WriteBlankScenarioFile(path, BlankOptions{PlayerCount: 1, HumanSlots: 1, DummyStarters: true}); err != nil {
		t.Fatal(err)
	}
	known := "void Known() {}\nvoid NeedsData(int value) {}\n"
	enabled := true
	out := filepath.Join(t.TempDir(), "patched.aoe2scenario")
	recipe := Recipe{
		XS: &XSRecipe{Name: "ScriptCallLint", Content: known},
		Triggers: []TriggerRecipe{{
			Op: "add_trigger", Name: "script-call lint", Enabled: &enabled,
			Effects: []EffectRecipe{
				{Op: "script_call", Message: "Missing();"},
				{Op: "script_call", Message: "Known(1);"},
				{Op: "script_call", Message: "NeedsData();"},
			},
		}},
	}
	if _, err := PatchRecipeFile(path, out, recipe); err != nil {
		t.Fatal(err)
	}
	report, err := LintFileWithOptions(out, LintOptions{SemanticInvariants: true, Generated: true})
	if err != nil {
		t.Fatal(err)
	}
	if !hasLintCode(report.Issues, "semantic_script_call_target_undefined") {
		t.Fatalf("missing undefined-target finding: %+v", report.Issues)
	}
	if !hasLintCode(report.Issues, "semantic_script_call_parameters") {
		t.Fatalf("missing parameter finding: %+v", report.Issues)
	}
}

func TestPartial159LintFlagsKnownScriptCallProseSignature(t *testing.T) {
	crashing := LintReport{}
	lintPartial159ScriptCallText(&crashing, []string{"write function for sidecar should be a script call effect, last in the list."}, LintOptions{})
	if !hasLintCode(crashing.Issues, "semantic_script_call_target_undefined") {
		t.Fatalf("partial crash signature was not flagged: %+v", crashing.Issues)
	}
	fixed := LintReport{}
	lintPartial159ScriptCallText(&fixed, []string{"Sup Yall", "Scenario Editor Phantom"}, LintOptions{})
	if hasLintCode(fixed.Issues, "semantic_script_call_target_undefined") {
		t.Fatalf("clean partial specimen was flagged: %+v", fixed.Issues)
	}
}

func hasLintCode(issues []LintIssue, code string) bool {
	for _, issue := range issues {
		if issue.Code == code {
			return true
		}
	}
	return false
}
