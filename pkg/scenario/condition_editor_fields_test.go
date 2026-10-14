package scenario

import "testing"

func TestConditionEditorSchemasCoverObservedTypes(t *testing.T) {
	rows := ConditionEditorSchemas()
	if got, want := len(rows), 40; got != want {
		t.Fatalf("schema rows = %d, want %d", got, want)
	}
	for _, conditionType := range []int{1, 20, 30, 54, 75, 80} {
		schema, ok := ConditionEditorSchemaForType(conditionType)
		if !ok || schema.Type != conditionType || schema.Name == "" {
			t.Fatalf("schema %d = %#v, present=%t", conditionType, schema, ok)
		}
	}
	separator, _ := ConditionEditorSchemaForType(29)
	if len(separator.VisibleFields) != 0 {
		t.Fatalf("separator fields = %#v, want empty", separator.VisibleFields)
	}
}

func TestConditionSummaryIncludesEditorFields(t *testing.T) {
	cond := &parsedNode{Fields: []*parsedNode{
		{Name: "condition_type", Value: int32(20)},
		{Name: "quantity", Value: int32(44)},
	}}
	summary := summarizeConditionData(cond)
	if len(summary.EditorFields) != 1 || summary.EditorFields[0] != "Quantity" {
		t.Fatalf("chance editor_fields = %#v", summary.EditorFields)
	}
	if got := summary.KnownFields["quantity"]; got != 44 {
		t.Fatalf("chance quantity = %#v, want 44", got)
	}
	flat := summarizeCondition(cond)
	if got, ok := flat["editor_fields"].([]string); !ok || len(got) != 1 || got[0] != "Quantity" {
		t.Fatalf("flat chance editor_fields = %#v", flat["editor_fields"])
	}
}

func TestTriggerSearchMatchCarriesConditionEditorFields(t *testing.T) {
	row := triggerRow{
		Index:         0,
		Conditions:    []map[string]any{{"type": int32(20), "quantity": int32(44)}},
		ConditionData: []ConditionSummary{{ConditionIndex: 0, Type: 20, TypeName: "Chance", KnownFields: map[string]any{"quantity": int32(44)}, EditorFields: []string{"Quantity"}}},
	}
	matches := row.searchMatches(TriggerSearchOptions{ConditionQuery: "chance"})
	if len(matches) != 1 || len(matches[0].ConditionData) != 1 {
		t.Fatalf("condition search matches=%+v", matches)
	}
	got := matches[0].ConditionData[0]
	if got.TypeName != "Chance" || got.KnownFields["quantity"] != int32(44) || len(got.EditorFields) != 1 || got.EditorFields[0] != "Quantity" {
		t.Fatalf("condition data=%+v", got)
	}
}
