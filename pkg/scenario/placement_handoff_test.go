package scenario

import (
	"path/filepath"
	"testing"
)

func TestAutoPlacementCarriesResolvedPolicyIntoAddUnits(t *testing.T) {
	x, y := 10.25, 20.75
	resolver := &clusterFootprintResolver{memo: map[int]bool{1366: true, 349: false}}
	for _, id := range []int{1366, 349} {
		r := UnitRecipe{Op: "add_unit", UnitConst: id, Snap: "auto", X: &x, Y: &y}
		placed, err := resolveUnitPlacement(r, resolver)
		if err != nil {
			t.Fatal(err)
		}
		// AddUnits intentionally has no resolver. A second pass must preserve
		// the DAT-backed decision made by ApplyRecipe.
		again, err := resolveUnitPlacement(placed, nil)
		if err != nil {
			t.Fatal(err)
		}
		if id == 1366 {
			if again.Snap != "fractional" || *again.X != x || *again.Y != y {
				t.Fatalf("fractional policy lost: %+v", again)
			}
		} else if again.Snap != "grid" || *again.X != 10.5 || *again.Y != 20.5 {
			t.Fatalf("blocking unit not snapped: %+v", again)
		}
		seed, err := BlankScenarioSeedBytes()
		if err != nil {
			t.Fatal(err)
		}
		f, err := Parse(seed)
		if err != nil {
			t.Fatal(err)
		}
		if err = f.AddUnits([]UnitRecipe{placed}); err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(t.TempDir(), "placement.aoe2scenario")
		if err = f.Write(path); err != nil {
			t.Fatal(err)
		}
		read, err := Open(path)
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, section := range read.Units.Sections {
			for _, u := range section.Units {
				if u.UnitConst == id {
					found = true
					if u.X != *again.X || u.Y != *again.Y {
						t.Fatalf("written coordinates changed: %+v", u)
					}
				}
			}
		}
		if !found {
			t.Fatal("added unit missing after write")
		}
	}
}
