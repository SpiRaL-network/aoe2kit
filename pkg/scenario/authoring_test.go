package scenario

import (
	"bytes"
	"math"
	"path/filepath"
	"testing"
)

func TestFieldLevelAuthoringForSekiroTriggerTypes(t *testing.T) {
	quantity, object, action, replacement := 25, 17, 3, 18
	conditions := []ConditionRecipe{
		{Op: "chance", Quantity: &quantity},
		{Op: "destroy_object", UnitObject: &object},
		{Op: "object_has_action", UnitObject: &object, UnitAIAction: &action},
	}
	wantTypes := []int{20, 6, 28}
	for i, recipe := range conditions {
		overrides, err := conditionOverrides(recipe)
		if err != nil {
			t.Fatalf("condition %q: %v", recipe.Op, err)
		}
		if got := overrides["condition_type"]; got != wantTypes[i] {
			t.Fatalf("condition %q type = %v, want %d", recipe.Op, got, wantTypes[i])
		}
	}
	effect, err := effectOverrides(EffectRecipe{Op: "replace_object", SelectedObjectIDs: []int{0}, ObjectListUnitID2: &replacement})
	if err != nil {
		t.Fatalf("replace_object: %v", err)
	}
	if got := effect["effect_type"]; got != 43 {
		t.Fatalf("replace_object type = %v, want 43", got)
	}
	if got := effect["object_list_unit_id_2"]; got != replacement {
		t.Fatalf("replace_object target = %v, want %d", got, replacement)
	}
	if got := effect["object_list_unit_id"]; got != -1 {
		t.Fatalf("replace_object source selector = %v, want -1", got)
	}
	if got := effect["target_player"]; got != 2 {
		t.Fatalf("replace_object target player = %v, want 2", got)
	}
	if got, ok := effect["quantity_float"].(float64); !ok || !math.IsNaN(got) || math.Float32bits(float32(got)) != 0xffffffff {
		t.Fatalf("replace_object quantity_float = %v, want NaN", effect["quantity_float"])
	}
}

func TestFieldLevelAuthoringRejectsMissingRequiredValues(t *testing.T) {
	if _, err := conditionOverrides(ConditionRecipe{Op: "chance"}); err == nil {
		t.Fatal("chance without quantity unexpectedly succeeded")
	}
	if _, err := conditionOverrides(ConditionRecipe{Op: "object_has_action"}); err == nil {
		t.Fatal("object_has_action without fields unexpectedly succeeded")
	}
	if _, err := effectOverrides(EffectRecipe{Op: "replace_object"}); err == nil {
		t.Fatal("replace_object without object lists unexpectedly succeeded")
	}
	replacement := 18
	if _, err := effectOverrides(EffectRecipe{Op: "replace_object", ObjectListUnitID2: &replacement}); err == nil {
		t.Fatal("replace_object without a selector unexpectedly succeeded")
	}
}

func TestReplaceObjectMatchesEditorFixtureWhenConfigured(t *testing.T) {
	path := filepath.Join("testdata", "replace_object_selected.aoe2scenario")
	file, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	triggers := file.root.section("Triggers")
	if triggers == nil || len(triggers.list("trigger_data")) == 0 || len(triggers.list("trigger_data")[0].list("effect_data")) == 0 {
		t.Fatal("fixture has no trigger effect")
	}
	effects, err := buildEffectNodes(mustWriteSpec(t, file), []EffectRecipe{{
		Op: "replace_object", SelectedObjectIDs: []int{0}, ObjectListUnitID2: intPtr(2552),
	}})
	if err != nil {
		t.Fatal(err)
	}
	want := triggers.list("trigger_data")[0].list("effect_data")[0].raw()
	if !bytes.Equal(effects[0].raw(), want) {
		t.Fatalf("recipe Replace Object bytes differ from editor fixture: got %d bytes, want %d", len(effects[0].raw()), len(want))
	}
}

func TestReplaceObjectMatchesEditorAreaFixtureWhenConfigured(t *testing.T) {
	path := filepath.Join("testdata", "replace_object_area.aoe2scenario")
	file, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	triggers := file.root.section("Triggers")
	if triggers == nil || len(triggers.list("trigger_data")) == 0 || len(triggers.list("trigger_data")[0].list("effect_data")) == 0 {
		t.Fatal("fixture has no trigger effect")
	}
	source, target, group, objectType, facet, targetPlayer := 1559, 1089, 8, 3, 4, 0
	x1, y1, x2, y2 := 39, 52, 45, 58
	effects, err := buildEffectNodes(mustWriteSpec(t, file), []EffectRecipe{{
		Op: "replace_object", SelectedObjectIDs: []int{0},
		ObjectListUnitID: &source, ObjectListUnitID2: &target, TargetPlayer: &targetPlayer,
		AreaX1: &x1, AreaY1: &y1, AreaX2: &x2, AreaY2: &y2,
		ObjectGroup: &group, ObjectType: &objectType, Facet2: &facet,
	}})
	if err != nil {
		t.Fatal(err)
	}
	want := triggers.list("trigger_data")[0].list("effect_data")[0].raw()
	if !bytes.Equal(effects[0].raw(), want) {
		t.Fatalf("recipe area Replace Object bytes differ from editor fixture: got %d bytes, want %d", len(effects[0].raw()), len(want))
	}
}

func mustWriteSpec(t *testing.T, file *File) *Spec {
	t.Helper()
	spec, err := file.writeSpec()
	if err != nil {
		t.Fatal(err)
	}
	return spec
}
