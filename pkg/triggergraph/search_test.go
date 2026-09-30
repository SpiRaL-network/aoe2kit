package triggergraph

import "testing"

func TestSearchTriggerGraphEffectAndCondition(t *testing.T) {
	graph := testGraph("search", []map[string]any{{
		"name":    "P2 Spawn Relay",
		"enabled": 1,
		"effects": []map[string]any{{
			"type":          11,
			"type_name":     "create_object",
			"fields_prefix": []int{11, 83, -1, -1, -1, -1, -1, -1, 83, 2},
			"known_fields":  map[string]any{"effect_type": 11, "object_list_unit_id": 83, "source_player": 2},
		}},
		"conditions": []map[string]any{{
			"type":         8,
			"type_name":    "accumulate_attribute",
			"fields":       []int{8, 33, 10, 20, -1, -1, -1, 2},
			"known_fields": map[string]any{"condition_type": 8, "quantity": 10, "attribute": 20, "source_player": 2},
		}},
	}})

	player := 2
	unit := 83
	report, err := Search(graph, SearchOptions{Effect: "create", Player: &player, UnitConst: &unit})
	if err != nil {
		t.Fatal(err)
	}
	if report.TotalMatches != 1 || len(report.Matches) != 1 {
		t.Fatalf("effect matches = total %d shown %d, want 1/1", report.TotalMatches, len(report.Matches))
	}
	if report.Matches[0].EffectTypeName != "create_object" {
		t.Fatalf("effect type = %q, want create_object", report.Matches[0].EffectTypeName)
	}

	report, err = Search(graph, SearchOptions{Condition: "accumulate", Player: &player})
	if err != nil {
		t.Fatal(err)
	}
	if report.TotalMatches != 1 || report.Matches[0].ConditionName != "accumulate_attribute" {
		t.Fatalf("condition match = %+v, want accumulate_attribute", report.Matches)
	}

	report, err = Search(graph, SearchOptions{Grep: "spawn"})
	if err != nil {
		t.Fatal(err)
	}
	if report.TotalMatches == 0 {
		t.Fatalf("grep trigger-name search found no matches")
	}
}

func TestSearchWithoutCriteriaListsTriggers(t *testing.T) {
	graph := testGraph("all", []map[string]any{{
		"name":              "A trigger",
		"description":       "A description",
		"short_description": "A short description",
	}})
	report, err := Search(graph, SearchOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if report.TotalMatches != 1 || len(report.Matches) != 1 {
		t.Fatalf("unfiltered matches = total %d shown %d, want 1/1", report.TotalMatches, len(report.Matches))
	}
	match := report.Matches[0]
	if match.TriggerName != "A trigger" || match.Description != "A description" || match.ShortDescription != "A short description" {
		t.Fatalf("unfiltered trigger = %+v", match)
	}
}

func TestSearchEffectNameDoesNotMatchEmbeddedSubstring(t *testing.T) {
	graph := testGraph("activate", []map[string]any{{
		"name": "Deactivate",
		"effects": []map[string]any{{
			"type":         9,
			"type_name":    "deactivate_trigger",
			"known_fields": map[string]any{"effect_type": 9, "trigger_id": 0},
		}},
	}})
	report, err := Search(graph, SearchOptions{Effect: "activate_trigger"})
	if err != nil {
		t.Fatal(err)
	}
	if report.TotalMatches != 0 {
		t.Fatalf("activate_trigger matched deactivate_trigger: %+v", report.Matches)
	}
}
