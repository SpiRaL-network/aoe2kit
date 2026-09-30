package datfacts

import "testing"

func TestEffectCommandFactsExposeDisputed200Series(t *testing.T) {
	row, err := ExplainEffectCommandType(204)
	if err != nil {
		t.Fatal(err)
	}
	if row.Name != "add_local_building_attribute_advanced" {
		t.Fatalf("204 name = %q", row.Name)
	}
	if len(row.Disagreements) == 0 {
		t.Fatal("204 should carry the GoKu/AoE2Kit disagreement")
	}
	if row.Confidence != "strong_hypothesis_with_known_disagreement" {
		t.Fatalf("204 confidence = %q", row.Confidence)
	}
}

func TestCatalogsArePopulated(t *testing.T) {
	for _, domain := range []string{"effect-types", "attributes", "resources", "task-types", "tech-modifiers", "scenario-unit-classes", "scenario-object-types", "scenario-unit-ai-actions"} {
		report, err := Catalog(domain)
		if err != nil {
			t.Fatalf("Catalog(%s): %v", domain, err)
		}
		if report.Summary.Rows == 0 || len(report.Rows) == 0 {
			t.Fatalf("Catalog(%s) empty", domain)
		}
	}
}

func TestScenarioEditorEnumFacts(t *testing.T) {
	if got := ScenarioUnitClassName(40); got != "salvage_pile" {
		t.Fatalf("ScenarioUnitClassName(40) = %q, want salvage_pile", got)
	}
	if got := ScenarioObjectTypeName(4); got != "military" {
		t.Fatalf("ScenarioObjectTypeName(4) = %q, want military", got)
	}
	if got := ScenarioUnitAIActionName(7); got != "stop" {
		t.Fatalf("ScenarioUnitAIActionName(7) = %q, want stop", got)
	}
	report, err := Catalog("scenario-unit-classes")
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range report.Rows {
		if row.ID == 40 && row.Name == "salvage_pile" && row.Confidence == "engine_verified" {
			return
		}
	}
	t.Fatalf("missing engine-verified salvage_pile row: %+v", report.Rows)
}

func TestTaskFactsIncludeInfluenceAbility(t *testing.T) {
	report, err := Catalog("task-types")
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range report.Rows {
		if row.ID == 155 && row.Name == "influence_ability" {
			return
		}
	}
	t.Fatal("task 155 influence_ability missing")
}
