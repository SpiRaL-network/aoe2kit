package scenario

import (
	"os"
	"path/filepath"
	"testing"

	"aoe2kit/pkg/testfixtures"
)

func TestEffectWhereFrontTowersGolden(t *testing.T) {
	path := testfixtures.Path(t, "save-analysis/CB_FRONT_TOWERS_V247.aoe2scenario")
	if _, err := os.Stat(path); err != nil {
		t.Skipf("golden scenario not present: %v", err)
	}
	report, err := EffectsWhereFile(path, EffectWhereOptions{Query: "create_object", Limit: 3})
	if err != nil {
		t.Fatal(err)
	}
	if report.TotalMatches == 0 || len(report.Matches) == 0 {
		t.Fatalf("create_object matches = total %d sample %d, want nonzero", report.TotalMatches, len(report.Matches))
	}
	if report.Matches[0].TypeName != "create_object" {
		t.Fatalf("first match type = %q, want create_object", report.Matches[0].TypeName)
	}
}

func TestTriggerSearchFrontTowersGolden(t *testing.T) {
	path := testfixtures.Path(t, "save-analysis/CB_FRONT_TOWERS_V247.aoe2scenario")
	if _, err := os.Stat(path); err != nil {
		t.Skipf("golden scenario not present: %v", err)
	}
	report, err := TriggerSearchFile(path, TriggerSearchOptions{EffectQuery: "display_instructions", Limit: 5})
	if err != nil {
		t.Fatal(err)
	}
	if report.TotalMatches == 0 || len(report.Matches) == 0 {
		t.Fatalf("display_instructions matches = total %d sample %d, want nonzero", report.TotalMatches, len(report.Matches))
	}
	if report.Matches[0].EffectTypeName != "display_instructions" {
		t.Fatalf("first match effect = %q, want display_instructions", report.Matches[0].EffectTypeName)
	}
}

func TestScenarioGlossaryFrontTowersGolden(t *testing.T) {
	path := testfixtures.Path(t, "save-analysis/CB_FRONT_TOWERS_V247.aoe2scenario")
	if _, err := os.Stat(path); err != nil {
		t.Skipf("golden scenario not present: %v", err)
	}
	report, err := ScenarioGlossaryFile(path, ScenarioGlossaryOptions{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if report.TriggerCount == 0 || report.EffectCount == 0 {
		t.Fatalf("glossary counts = triggers %d effects %d, want nonzero", report.TriggerCount, report.EffectCount)
	}
	if bucketByName(report.EffectTypes, "create_object") == nil {
		t.Fatalf("glossary missing create_object bucket")
	}
}

func TestSummarizeConditionPreservesMeaningfulZeroes(t *testing.T) {
	cond := &parsedNode{Fields: []*parsedNode{
		{Name: "condition_type", Value: int32(22)},
		{Name: "comparison", Value: int32(0)},
		{Name: "inverted", Value: int32(0)},
		{Name: "quantity", Value: int32(-1)},
		{Name: "variable", Value: int32(7)},
	}}
	summary := summarizeCondition(cond)
	if summary["type_name"] != "variable_value" {
		t.Fatalf("type_name = %v, want variable_value", summary["type_name"])
	}
	if got, ok := summary["comparison"]; !ok || got != 0 {
		t.Fatalf("comparison = %v ok=%t, want preserved zero", got, ok)
	}
	if got, ok := summary["inverted"]; !ok || got != 0 {
		t.Fatalf("inverted = %v ok=%t, want preserved zero", got, ok)
	}
	if _, ok := summary["quantity"]; ok {
		t.Fatalf("quantity sentinel should be omitted: %#v", summary)
	}
}

func TestEffectWhereAndDiffExposePayloadAndTriggerScope(t *testing.T) {
	before := effectControlFixture(t, 100)
	after := effectControlFixture(t, 250)

	all, err := EffectsWhereFile(before, EffectWhereOptions{
		Query: "change_object_cost", Limit: 10, IncludeRawFields: true,
	})
	if err != nil {
		t.Fatalf("EffectsWhereFile: %v", err)
	}
	if all.TotalMatches != 1 || len(all.Matches) != 1 {
		t.Fatalf("all matches = total %d sample %d, want 1/1", all.TotalMatches, len(all.Matches))
	}
	if got := all.Matches[0].KnownFields["resource_1_quantity"]; got != 100 {
		t.Fatalf("known resource_1_quantity = %#v, want 100", got)
	}
	if len(all.Matches[0].RawFields) == 0 {
		t.Fatal("raw fields empty, want encoded effect payload")
	}

	trigger := all.Matches[0].TriggerIndex
	scoped, err := EffectsWhereFile(before, EffectWhereOptions{
		Query: "change_object_cost", TriggerIndex: &trigger, Limit: 10,
	})
	if err != nil {
		t.Fatalf("scoped EffectsWhereFile: %v", err)
	}
	if scoped.TotalMatches != 1 || len(scoped.Matches) != 1 {
		t.Fatalf("scoped matches = total %d sample %d, want 1/1", scoped.TotalMatches, len(scoped.Matches))
	}
	missingTrigger := trigger + 1
	empty, err := EffectsWhereFile(before, EffectWhereOptions{
		Query: "change_object_cost", TriggerIndex: &missingTrigger, Limit: 10,
	})
	if err != nil {
		t.Fatalf("empty scoped EffectsWhereFile: %v", err)
	}
	if empty.TotalMatches != 0 || len(empty.Matches) != 0 {
		t.Fatalf("missing trigger matches = total %d sample %d, want 0/0", empty.TotalMatches, len(empty.Matches))
	}

	diff, err := DiffEffectsFile(before, after, EffectWhereOptions{Query: "change_object_cost"})
	if err != nil {
		t.Fatalf("DiffEffectsFile: %v", err)
	}
	if diff.BeforeMatches != 1 || diff.AfterMatches != 1 || diff.Compared != 1 || diff.Modified != 1 {
		t.Fatalf("diff counts = %+v, want 1/1/1/1", diff)
	}
	delta, ok := diff.Changes[0].Fields["resource_1_quantity"]
	if !ok || delta.Before != 100 || delta.After != 250 {
		t.Fatalf("quantity delta = %#v, want 100 -> 250", delta)
	}
}

func effectControlFixture(t *testing.T, quantity int) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "control.aoe2scenario")
	if _, err := WriteBlankScenarioFile(path, BlankOptions{
		PlayerCount: 2, HumanSlots: 2, NoStarters: true,
	}); err != nil {
		t.Fatalf("WriteBlankScenarioFile: %v", err)
	}
	file, err := Open(path)
	if err != nil {
		t.Fatalf("Open blank: %v", err)
	}
	unit, player, resource, amount := 629, 1, 0, quantity
	if err := file.AddTrigger(TriggerRecipe{
		Op:   "add_trigger",
		Name: "Control cost",
		Effects: []EffectRecipe{{
			Op: "change_object_cost", ObjectListUnitID: &unit, SourcePlayer: &player,
			Resource1: &resource, Resource1Quantity: &amount,
		}},
	}); err != nil {
		t.Fatalf("AddTrigger: %v", err)
	}
	if err := file.Write(path); err != nil {
		t.Fatalf("Write fixture: %v", err)
	}
	return path
}

func TestTriggerIntelOnSyntheticMechanic(t *testing.T) {
	path := triggerIntelFixture(t)
	unitRef := 1234
	player := 1
	area := []int{10, 10, 20, 20}
	search, err := TriggerSearchFile(path, TriggerSearchOptions{
		EffectQuery: "heal_object",
		UnitRef:     &unitRef,
		Player:      &player,
		Area:        area,
		Limit:       10,
	})
	if err != nil {
		t.Fatalf("TriggerSearchFile effect: %v", err)
	}
	if search.TotalMatches == 0 || search.Matches[0].EffectTypeName != "heal_object" {
		t.Fatalf("heal search = %+v", search)
	}
	conditionSearch, err := TriggerSearchFile(path, TriggerSearchOptions{
		ConditionQuery: "object_selected",
		UnitRef:        &unitRef,
		Limit:          10,
	})
	if err != nil {
		t.Fatalf("TriggerSearchFile condition: %v", err)
	}
	if conditionSearch.TotalMatches == 0 || conditionSearch.Matches[0].ConditionName != "object_selected" {
		t.Fatalf("condition search = %+v", conditionSearch)
	}
	coverage, err := PlayerCoverageFile(path, PlayerCoverageOptions{Players: []int{1, 2, 3, 4}})
	if err != nil {
		t.Fatalf("PlayerCoverageFile: %v", err)
	}
	if coverage.Summary.Issues == 0 {
		t.Fatalf("coverage issues = %+v, want asymmetry", coverage)
	}
	mechanic, err := MechanicFile(path, MechanicOptions{Kind: "garrison-token", Grep: "heal"})
	if err != nil {
		t.Fatalf("MechanicFile: %v", err)
	}
	if mechanic.Summary.TriggerCount < 2 {
		t.Fatalf("mechanic triggers = %+v, want heal candidates", mechanic)
	}
	neighbor, err := TriggerNeighborhoodFile(path, TriggerNeighborhoodOptions{TriggerIndex: intPtr(0), Depth: 2})
	if err != nil {
		t.Fatalf("TriggerNeighborhoodFile: %v", err)
	}
	if len(neighbor.Nodes) == 0 {
		t.Fatalf("neighborhood = %+v, want nodes", neighbor)
	}
	flow, err := TriggerFlowFile(path, TriggerFlowOptions{Grep: "heal"})
	if err != nil {
		t.Fatalf("TriggerFlowFile: %v", err)
	}
	if len(flow.Stages) == 0 {
		t.Fatalf("flow = %+v, want stages", flow)
	}
	lint, err := LintFile(path)
	if err != nil {
		t.Fatalf("LintFile: %v", err)
	}
	if !hasIssueCode(lint.Issues, "trigger_player_family_asymmetry") {
		t.Fatalf("lint issues = %+v, want player-family asymmetry", lint.Issues)
	}
}

func triggerIntelFixture(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "trigger-intel.aoe2scenario")
	if _, err := WriteBlankScenarioFile(path, BlankOptions{PlayerCount: 4, HumanSlots: 1, DummyStarters: true}); err != nil {
		t.Fatalf("WriteBlankScenarioFile: %v", err)
	}
	file, err := Open(path)
	if err != nil {
		t.Fatalf("Open blank: %v", err)
	}
	unitRef := 1234
	sheep := 594
	p1 := 1
	p3 := 3
	x1, y1, x2, y2 := 10, 10, 20, 20
	zero := 0
	if err := file.AddTrigger(TriggerRecipe{
		Op:   "add_trigger",
		Name: "P1 Heal Init",
		Effects: []EffectRecipe{
			{Op: "heal_object", ObjectListUnitID: &sheep, SourcePlayer: &p1, AreaX1: &x1, AreaY1: &y1, AreaX2: &x2, AreaY2: &y2, SelectedObjectIDs: []int{unitRef}},
		},
		Conditions: []ConditionRecipe{
			{Op: "object_selected", UnitObject: &unitRef},
		},
	}); err != nil {
		t.Fatalf("AddTrigger P1: %v", err)
	}
	if err := file.AddTrigger(TriggerRecipe{
		Op:   "add_trigger",
		Name: "P3 Heal Init",
		Effects: []EffectRecipe{
			{Op: "heal_object", ObjectListUnitID: &sheep, SourcePlayer: &p3, SelectedObjectIDs: []int{unitRef}},
		},
	}); err != nil {
		t.Fatalf("AddTrigger P3: %v", err)
	}
	if err := file.AddTrigger(TriggerRecipe{
		Op:   "add_trigger",
		Name: "P1 Heal Relay",
		Effects: []EffectRecipe{
			{Op: "activate_trigger", TriggerID: &zero},
		},
	}); err != nil {
		t.Fatalf("AddTrigger relay: %v", err)
	}
	if err := file.Write(path); err != nil {
		t.Fatalf("Write fixture: %v", err)
	}
	return path
}
