package triggergraph

import "testing"

func TestNeighborhoodFollowsTriggerControlEdges(t *testing.T) {
	graph := testGraph("neighborhood", []map[string]any{
		{
			"name":    "Arm",
			"effects": []map[string]any{triggerControlEffect(8, 1)},
		},
		{
			"name":    "Use",
			"effects": []map[string]any{triggerControlEffect(9, 2)},
		},
		{
			"name":    "Cleanup",
			"effects": []map[string]any{},
		},
	})
	trigger := 1
	report, err := Neighborhood(graph, NeighborhoodOptions{TriggerIndex: &trigger, Depth: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Nodes) != 3 {
		t.Fatalf("nodes = %d, want 3: %+v", len(report.Nodes), report.Nodes)
	}
	if len(report.Edges) != 2 {
		t.Fatalf("edges = %d, want 2: %+v", len(report.Edges), report.Edges)
	}
}

func TestNeighborhoodNeedsValidTrigger(t *testing.T) {
	_, err := Neighborhood(testGraph("empty", nil), NeighborhoodOptions{})
	if err == nil {
		t.Fatalf("Neighborhood without trigger succeeded")
	}
	trigger := 4
	_, err = Neighborhood(testGraph("empty", []map[string]any{{"name": "only"}}), NeighborhoodOptions{TriggerIndex: &trigger})
	if err == nil {
		t.Fatalf("Neighborhood with out-of-range trigger succeeded")
	}
}

func triggerControlEffect(effectType, target int) map[string]any {
	return map[string]any{
		"type":         effectType,
		"type_name":    effectTypeName(effectType),
		"known_fields": map[string]any{"effect_type": effectType, "trigger_id": target},
	}
}
