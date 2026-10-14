package scenario

import (
	"path/filepath"
	"testing"
)

func TestRemoveUnitsBatchPreservesAndRejectsReferences(t *testing.T) {
	path := filepath.Join(t.TempDir(), "batch.aoe2scenario")
	if _, err := WriteBlankScenarioFile(path, BlankOptions{PlayerCount: 2, HumanSlots: 1, NoStarters: true, MapWidth: 80, MapHeight: 80}); err != nil {
		t.Fatal(err)
	}
	f, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	x, y := 10.5, 10.5
	a, b, c := 100, 101, 102
	if err = f.AddUnits([]UnitRecipe{
		{Op: "add_unit", Player: 1, UnitConst: 12, X: &x, Y: &y, ReferenceID: &a},
		{Op: "add_unit", Player: 1, UnitConst: 83, X: &x, Y: &y, ReferenceID: &b, GarrisonedInID: &a},
		{Op: "add_unit", Player: 0, UnitConst: 349, X: &x, Y: &y, ReferenceID: &c},
	}); err != nil {
		t.Fatal(err)
	}
	if err = f.RemoveUnits([]int{a, c}); err == nil {
		t.Fatal("removed referenced building")
	}
	if f.Units.Total != 3 {
		t.Fatal("failed batch mutated units")
	}
	if err = f.RemoveUnits([]int{c, 9999}); err == nil {
		t.Fatal("accepted missing ID")
	}
	if f.Units.Total != 3 {
		t.Fatal("missing-ID batch mutated units")
	}
	if err = f.RemoveUnits([]int{b, c, c}); err != nil {
		t.Fatal(err)
	}
	if f.Units.Total != 1 {
		t.Fatalf("retained %d", f.Units.Total)
	}
	if err = f.Write(path); err != nil {
		t.Fatal(err)
	}
	f, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if err = f.VerifyRebuild(); err != nil {
		t.Fatal(err)
	}
	if f.Units.Total != 1 {
		t.Fatalf("written total %d", f.Units.Total)
	}
}
