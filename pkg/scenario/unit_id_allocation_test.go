package scenario

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestAddUnitsReservesFutureExplicitIDs(t *testing.T) {
	path := filepath.Join(t.TempDir(), "blank.aoe2scenario")
	if _, err := WriteBlankScenarioFile(path, BlankOptions{}); err != nil {
		t.Fatal(err)
	}
	f, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	a, b := 910000, 910010
	x, y := 10.5, 10.5
	units := []UnitRecipe{{Op: "add_unit", Player: 0, UnitConst: 128, X: &x, Y: &y, ReferenceID: &a}}
	for i := 0; i < 24; i++ {
		units = append(units, UnitRecipe{Op: "add_unit", Player: 2, UnitConst: 74, X: &x, Y: &y})
	}
	units = append(units, UnitRecipe{Op: "add_unit", Player: 0, UnitConst: 128, X: &x, Y: &y, ReferenceID: &b})
	if err := f.AddUnits(units); err != nil {
		t.Fatal(err)
	}
	seen := map[int]bool{}
	for _, section := range f.Units.Sections {
		for _, unit := range section.Units {
			if seen[unit.ReferenceID] {
				t.Fatalf("duplicate ID %d", unit.ReferenceID)
			}
			seen[unit.ReferenceID] = true
		}
	}
	if !seen[a] || !seen[b] {
		t.Fatal("explicit carrier IDs lost")
	}
	for _, batch := range [][]UnitRecipe{{units[0]}, {{Op: "add_unit", Player: 0, UnitConst: 128, X: &x, Y: &y, ReferenceID: ipForIDTest(920000)}, {Op: "add_unit", Player: 1, UnitConst: 128, X: &x, Y: &y, ReferenceID: ipForIDTest(920000)}}} {
		before := f.Units.Total
		if err := f.AddUnits(batch); err == nil || !strings.Contains(err.Error(), "duplicate reference_id") {
			t.Fatalf("expected collision rejection, got %v", err)
		}
		if f.Units.Total != before {
			t.Fatal("collision mutated units")
		}
	}
}

func TestAddUnitsAdvancesNextUnitIDWithoutLoweringEditorCursor(t *testing.T) {
	path := filepath.Join(t.TempDir(), "blank.aoe2scenario")
	if _, err := WriteBlankScenarioFile(path, BlankOptions{}); err != nil {
		t.Fatal(err)
	}
	f, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	dataHeader := f.root.section("DataHeader")
	if err := setIntField(dataHeader, "next_unit_id_to_place", "u32", 100); err != nil {
		t.Fatal(err)
	}
	x, y := 10.5, 10.5
	ref := 42
	if err := f.AddUnit(UnitRecipe{Op: "add_unit", Player: 0, UnitConst: 128, X: &x, Y: &y, ReferenceID: &ref}); err != nil {
		t.Fatal(err)
	}
	got, ok := f.root.section("DataHeader").intValue("next_unit_id_to_place")
	if !ok || got != 100 {
		t.Fatalf("next_unit_id_to_place = %d, %v; want preserved editor cursor 100", got, ok)
	}

	f2, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	high := 123
	if err := f2.AddUnit(UnitRecipe{Op: "add_unit", Player: 0, UnitConst: 128, X: &x, Y: &y, ReferenceID: &high}); err != nil {
		t.Fatal(err)
	}
	got, ok = f2.root.section("DataHeader").intValue("next_unit_id_to_place")
	if !ok || got != high+1 {
		t.Fatalf("next_unit_id_to_place = %d, %v; want %d", got, ok, high+1)
	}
}

func TestLintFlagsStaleNextUnitID(t *testing.T) {
	path := filepath.Join(t.TempDir(), "blank.aoe2scenario")
	if _, err := WriteBlankScenarioFile(path, BlankOptions{}); err != nil {
		t.Fatal(err)
	}
	f, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	x, y := 10.5, 10.5
	ref := 42
	if err := f.AddUnit(UnitRecipe{Op: "add_unit", Player: 0, UnitConst: 128, X: &x, Y: &y, ReferenceID: &ref}); err != nil {
		t.Fatal(err)
	}
	if err := setIntField(f.root.section("DataHeader"), "next_unit_id_to_place", "u32", ref); err != nil {
		t.Fatal(err)
	}
	report := f.Lint()
	if !hasIssueCode(report.Issues, "next_unit_id_collision") {
		t.Fatalf("stale next-unit cursor not reported: %+v", report.Issues)
	}
}

func ipForIDTest(v int) *int { return &v }
