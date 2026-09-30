package triggergraph

import "testing"

func TestDiffSameGraph(t *testing.T) {
	graph := testGraph("same", []map[string]any{
		{
			"name":       "T0",
			"enabled":    1,
			"looping":    0,
			"effects":    []map[string]any{{"type": 4, "message": "hello"}},
			"conditions": []map[string]any{{"type": 10, "fields": []int{10, 33, 5}}},
		},
	})
	report := Diff(graph, graph, DiffOptions{})
	if !report.Same {
		t.Fatalf("Same = false, changes=%v", report.Changes)
	}
	if report.Summary.Total != 0 || len(report.Changes) != 0 {
		t.Fatalf("changes = %d/%d, want 0", report.Summary.Total, len(report.Changes))
	}
}

func TestDiffEffectAndConditionChanges(t *testing.T) {
	before := testGraph("before", []map[string]any{
		{
			"name":       "T0",
			"enabled":    1,
			"looping":    0,
			"effects":    []map[string]any{{"type": 4, "message": "old"}},
			"conditions": []map[string]any{{"type": 10, "fields": []int{10, 33, 5}}},
		},
	})
	after := testGraph("after", []map[string]any{
		{
			"name":       "T0",
			"enabled":    uint32(0),
			"looping":    int8(0),
			"effects":    []map[string]any{{"type": float64(4), "message": "new"}},
			"conditions": []map[string]any{{"type": float64(10), "fields": []any{float64(10), float64(33), float64(7)}}},
		},
	})
	report := Diff(before, after, DiffOptions{})
	if report.Same {
		t.Fatalf("Same = true, want false")
	}
	if report.Summary.Modified == 0 {
		t.Fatalf("modified = 0, changes=%v", report.Changes)
	}
	wantFields := map[string]bool{"enabled": false, "effect_0.message": false, "condition_0.quantity": false}
	for _, change := range report.Changes {
		if _, ok := wantFields[change.Field]; ok {
			wantFields[change.Field] = true
		}
	}
	for field, seen := range wantFields {
		if !seen {
			t.Fatalf("missing field change %s in %#v", field, report.Changes)
		}
	}
}

func TestDiffLimitPreservesTotal(t *testing.T) {
	before := testGraph("before", []map[string]any{{"name": "A"}, {"name": "B"}, {"name": "C"}})
	after := testGraph("after", []map[string]any{{"name": "X"}, {"name": "Y"}, {"name": "Z"}})
	report := Diff(before, after, DiffOptions{Limit: 2})
	if report.Summary.Total != 3 {
		t.Fatalf("total = %d, want 3", report.Summary.Total)
	}
	if report.Summary.Shown != 2 || len(report.Changes) != 2 {
		t.Fatalf("shown = %d len=%d, want 2", report.Summary.Shown, len(report.Changes))
	}
}

func TestDiffTriggerMetadataAndOrder(t *testing.T) {
	before := testGraph("before", []map[string]any{{
		"name":            "T0",
		"description":     "old",
		"effect_order":    []int{0, 1},
		"condition_order": []int{0},
	}})
	after := testGraph("after", []map[string]any{{
		"name":            "T0",
		"description":     "new",
		"effect_order":    []int{1, 0},
		"condition_order": []int{},
	}})
	report := Diff(before, after, DiffOptions{})
	want := map[string]bool{"description": false, "effect_order": false, "condition_order": false}
	for _, change := range report.Changes {
		if _, ok := want[change.Field]; ok {
			want[change.Field] = true
		}
	}
	for field, seen := range want {
		if !seen {
			t.Fatalf("missing %s change in %#v", field, report.Changes)
		}
	}
}

func TestDiffTriggerHeaderProperties(t *testing.T) {
	before := testGraph("before", []map[string]any{{
		"name":                   "T0",
		"description_stid":       -1,
		"display_as_objective":   0,
		"description_order":      0,
		"make_header":            0,
		"short_description_stid": -1,
	}})
	after := testGraph("after", []map[string]any{{
		"name":                   "T0",
		"description_stid":       0,
		"display_as_objective":   1,
		"description_order":      0,
		"make_header":            1,
		"short_description_stid": 0,
	}})
	report := Diff(before, after, DiffOptions{})
	want := map[string]bool{
		"description_stid":       false,
		"display_as_objective":   false,
		"make_header":            false,
		"short_description_stid": false,
	}
	for _, change := range report.Changes {
		if _, ok := want[change.Field]; ok {
			want[change.Field] = true
		}
	}
	for field, seen := range want {
		if !seen {
			t.Fatalf("missing %s change in %#v", field, report.Changes)
		}
	}
}

func testGraph(hash string, triggers []map[string]any) *Graph {
	graph, err := FromTriggers(triggers, 0, 0, nil)
	if err != nil {
		panic(err)
	}
	graph.SHA256 = hash
	return graph
}
