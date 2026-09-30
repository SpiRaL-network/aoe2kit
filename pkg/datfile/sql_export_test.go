package datfile

import "testing"

func TestAttributeMapUsesCanonicalDecodedFields(t *testing.T) {
	tests := map[int]string{
		0:   "hit_points",
		66:  "blood_unit_id",
		71:  "standing_graphic_1",
		145: "trailing_unit",
		185: "fly_mode",
		186: "can_be_gathered",
		187: "hill_mode",
	}
	if field, confidence := UGCAttributeDATField(125); field != "" || confidence != "unmapped" {
		t.Fatalf("attribute 125 = %q/%q, want unmapped until its byte location is proven", field, confidence)
	}
	if field, _ := UGCAttributeDATField(145); field == "" {
		t.Fatal("attribute 145 lost its proven trailing-unit mapping")
	}
	for id, want := range tests {
		got, confidence := UGCAttributeDATField(id)
		if got != want || confidence != "layout_correlated" {
			t.Errorf("attribute %d = %q/%q, want %q/layout_correlated", id, got, confidence, want)
		}
	}
}

func TestAttributeExportCoverageMarksOpaqueFields(t *testing.T) {
	coverage := exportAttributeCoverage()
	if coverage.Mapped == 0 || coverage.Exported == 0 || len(coverage.Unexported) == 0 {
		t.Fatalf("coverage = %+v, want mapped/exported/opaque values", coverage)
	}
	fields := exportedDATFields()
	for _, field := range []string{"trailing_unit", "trailing_options", "trailing_spacing", "blast_attack_level", "area_effect_specials", "blast_damage", "fly_mode"} {
		if !exportFieldAvailable(fields, field) {
			t.Errorf("newly decoded field %q was not marked exported", field)
		}
	}
	if !exportFieldAvailable(fields, "death_spawn") {
		t.Error("decoded death_spawn was not marked exported")
	}
	for _, field := range []string{"secondary_projectile_unit", "charge_target"} {
		if exportFieldAvailable(fields, field) {
			t.Errorf("opaque field %q was incorrectly marked exported", field)
		}
	}
	for _, field := range []string{"spawning_graphic", "upgrade_graphic", "stack_unit", "head_unit", "transform_unit", "pile_unit", "annex_unit_1_offset_x", "annex_unit_4_offset_y"} {
		if !exportFieldAvailable(fields, field) {
			t.Errorf("decoded building field %q was not marked exported", field)
		}
	}
	if !exportFieldAvailable(fields, "attacks.value") || !exportFieldAvailable(fields, "armours.value") {
		t.Fatal("child weapon fields were not recognized as exported")
	}
}
