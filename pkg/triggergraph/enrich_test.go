package triggergraph

import "testing"

func TestEnrichedTriggerRowsPreserveIdentityHash(t *testing.T) {
	legacy := []map[string]any{{
		"name":    "T0",
		"enabled": 1,
		"effects": []map[string]any{{
			"type":          11,
			"message":       "",
			"raw_sha256":    "abc",
			"fields_prefix": []int{11, 83, -1, -1, -1, -1, -1, -1, 83, 2},
		}},
		"conditions": []map[string]any{{
			"type":        10,
			"fields":      []int{10, 33, 30},
			"xs_function": "",
		}},
	}}
	enriched := []map[string]any{{
		"name":    "T0",
		"enabled": 1,
		"effects": []map[string]any{{
			"type":          11,
			"type_name":     "create_object",
			"message":       "",
			"raw_sha256":    "abc",
			"fields_prefix": []int{11, 83, -1, -1, -1, -1, -1, -1, 83, 2},
			"known_fields":  map[string]any{"effect_type": 11, "object_list_unit_id": 83, "source_player": 2},
		}},
		"conditions": []map[string]any{{
			"type":         10,
			"type_name":    "timer",
			"fields":       []int{10, 33, 30},
			"xs_function":  "",
			"known_fields": map[string]any{"condition_type": 10, "quantity": 30},
		}},
	}}
	legacyGraph, err := FromTriggers(legacy, 0, 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	enrichedGraph, err := FromTriggers(enriched, 0, 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	if legacyGraph.SHA256 != enrichedGraph.SHA256 {
		t.Fatalf("enrichment changed identity hash: legacy=%s enriched=%s", legacyGraph.SHA256, enrichedGraph.SHA256)
	}
}

func TestEnrichEffectAndCondition(t *testing.T) {
	values := make([]int, 84)
	for i := range values {
		values[i] = -1
	}
	values[0] = 11
	values[1] = 83
	values[8] = 83
	values[9] = 2
	effect := map[string]any{"type": 11, "message": "spawn"}
	enrichEffect(effect, values, make([]byte, 84*4))
	if effect["type_name"] != "create_object" {
		t.Fatalf("effect type_name = %v, want create_object", effect["type_name"])
	}
	known, _ := effect["known_fields"].(map[string]any)
	if known["object_list_unit_id"] != 83 || known["source_player"] != 2 || known["message"] != "spawn" {
		t.Fatalf("known effect fields = %#v", known)
	}

	conditionValues := make([]int, 35)
	for i := range conditionValues {
		conditionValues[i] = -1
	}
	conditionValues[0] = 8
	conditionValues[1] = 33
	conditionValues[2] = 10
	conditionValues[3] = 20
	condition := map[string]any{"type": 8}
	enrichCondition(condition, conditionValues)
	if condition["type_name"] != "accumulate_attribute" {
		t.Fatalf("condition type_name = %v, want accumulate_attribute", condition["type_name"])
	}
	known, _ = condition["known_fields"].(map[string]any)
	if known["quantity"] != 10 || known["attribute"] != 20 {
		t.Fatalf("known condition fields = %#v", known)
	}
}
