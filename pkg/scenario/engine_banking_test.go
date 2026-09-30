package scenario

import (
	"path/filepath"
	"testing"
)

func TestLoadSafetyEditorPlayerZeroIsBlue(t *testing.T) {
	path := filepath.Join(t.TempDir(), "two.aoe2scenario")
	if _, err := WriteBlankScenarioFile(path, BlankOptions{PlayerCount: 2, HumanSlots: 2}); err != nil {
		t.Fatal(err)
	}
	f, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	// Use editor-native player indices; do not inherit the blank seed's convention.
	for i := range f.Players {
		f.Players[i].Active = i < 2
	}
	f.PlayerCount = 2
	r := f.LintWithOptions(LintOptions{LoadSafety: true})
	if hasIssueCode(r.Issues, "load_safety_player_count_mismatch") {
		t.Fatalf("valid blue/red rejected: %+v", r.Issues)
	}
	f.PlayerCount = 1
	if !hasIssueCode(f.LintWithOptions(LintOptions{LoadSafety: true}).Issues, "load_safety_player_count_mismatch") {
		t.Fatal("real mismatch missed")
	}
}

func TestForeignScenarioDuplicateReferencesAcrossPlayers(t *testing.T) {
	path := filepath.Join(t.TempDir(), "foreign.aoe2scenario")
	if _, err := WriteBlankScenarioFile(path, BlankOptions{}); err != nil {
		t.Fatal(err)
	}
	f, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	x, y := 10.5, 10.5
	if err := f.AddUnits([]UnitRecipe{{Op: "add_unit", Player: 0, UnitConst: 128, X: &x, Y: &y}, {Op: "add_unit", Player: 2, UnitConst: 74, X: &x, Y: &y}}); err != nil {
		t.Fatal(err)
	}
	// Simulate a foreign/hand-edited file bypassing the writer allocator.
	f.Units.Sections[2].Units[0].ReferenceID = f.Units.Sections[0].Units[0].ReferenceID
	r := f.Lint()
	if r.OK || !hasIssueCode(r.Issues, "duplicate_unit_reference") {
		t.Fatalf("collision missed: %+v", r)
	}
}

func TestXSCensusChecksEveryIndependentCarrier(t *testing.T) {
	f := scenarioWithXSFields("", "", "")
	f.Triggers = &TriggerInfo{}
	for i, source := range []string{"void good() {}\n", "void bad() { if (x%2==0) {} }\n"} {
		f.Triggers.Triggers = append(f.Triggers.Triggers, TriggerSummary{Index: i, EffectData: []EffectSummary{{Type: 55, Text: xsCarrierMessage("source", source)}}})
	}
	r, err := f.XSCensus(XSCensusOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if r.OK || len(r.Analysis.Files) != 2 || len(r.Findings) != 1 {
		t.Fatalf("later carrier escaped lint: %+v", r)
	}
}
