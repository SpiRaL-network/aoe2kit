package geotrace

import (
	"aoe2kit/pkg/scenario"
	"reflect"
	"testing"
)

func startTestReport(n int) Report {
	r := Report{GridSize: n, EcologyProfile: defaultEcologyProfile(), Recipe: &scenario.Recipe{Scenario: &scenario.ScenarioRecipe{}}}
	r.EcologyProfile.PlayerStarts = &PlayerStartProfile{}
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			v := true
			r.Cells = append(r.Cells, Cell{X: x, Y: y, Land: &v})
		}
	}
	return r
}

func TestStartsAndNeutralResourcesRequireOptIn(t *testing.T) {
	r := startTestReport(160)
	r.EcologyProfile.PlayerStarts = nil
	if err := applyPlayerStarts(&r); err != nil {
		t.Fatal(err)
	}
	if err := applyNeutralResources(&r); err != nil {
		t.Fatal(err)
	}
	if r.PlayerStarts != nil || len(r.NeutralResources) > 0 || len(r.Recipe.Units) > 0 {
		t.Fatal("default generation added game starts or neutral mines")
	}
}
func TestPlayerStartsEqualDeterministicAndClear(t *testing.T) {
	a, b := startTestReport(160), startTestReport(160)
	a.Recipe.Units = []scenario.UnitRecipe{startUnit(0, 349, 79.5, 79.5)}
	b.Recipe.Units = append([]scenario.UnitRecipe(nil), a.Recipe.Units...)
	if err := applyPlayerStarts(&a); err != nil {
		t.Fatal(err)
	}
	if err := applyPlayerStarts(&b); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(a.Recipe, b.Recipe) {
		t.Fatal("nondeterministic starts")
	}
	if len(a.PlayerStarts.Positions) != 2 || !reflect.DeepEqual(a.PlayerStarts.Positions[0].Inventory, a.PlayerStarts.Positions[1].Inventory) {
		t.Fatal("unequal inventory")
	}
	if a.PlayerStarts.RemovedAmbient != 1 {
		t.Fatal("core tree retained")
	}
	for _, p := range a.PlayerStarts.Positions {
		if p.Inventory[109] != 1 || p.Inventory[83] != 3 || p.Inventory[448] != 1 || p.Inventory[594] != 8 || p.Inventory[66] != 15 || p.Inventory[102] != 9 {
			t.Fatal(p.Inventory)
		}
	}
	occupied := map[[2]float64]bool{}
	for _, u := range a.Recipe.Units {
		k := [2]float64{*u.X, *u.Y}
		if occupied[k] {
			t.Fatal("overlap", k)
		}
		occupied[k] = true
		if u.UnitConst == 12 || u.UnitConst == 600 {
			t.Fatal("test-only object")
		}
	}
}
func TestPlayerStartModesAndInvalidLand(t *testing.T) {
	for _, mode := range []string{"engine", "none", "calibration"} {
		r := startTestReport(80)
		r.EcologyProfile.PlayerStarts = &PlayerStartProfile{Mode: mode, Count: 3}
		if err := applyPlayerStarts(&r); err != nil {
			t.Fatal(err)
		}
		want := 0
		if mode == "calibration" {
			want = 3
		}
		if len(r.Recipe.Units) != want || *r.Recipe.Scenario.PlayerCount != 3 {
			t.Fatal(mode)
		}
	}
	r := startTestReport(80)
	for i := range r.Cells {
		v := false
		r.Cells[i].Land = &v
	}
	if err := applyPlayerStarts(&r); err == nil {
		t.Fatal("accepted water starts")
	}
	if _, err := normalizedStarts(&PlayerStartProfile{Count: 9}); err == nil {
		t.Fatal("accepted9players")
	}
}

func TestPlayerStartsTwoOneVillagersAndSmallClearing(t *testing.T) {
	r := startTestReport(160)
	r.Recipe.Units = []scenario.UnitRecipe{startUnit(0, 349, 100.5, 100.5)}
	if err := applyPlayerStarts(&r); err != nil {
		t.Fatal(err)
	}
	for _, s := range r.PlayerStarts.Positions {
		v := map[[2]float64]bool{}
		for _, u := range s.Objects {
			if u.Unit == 83 {
				v[[2]float64{u.X - s.X, u.Y - s.Y}] = true
			}
		}
		for _, off := range [][2]float64{{1, 5}, {2, 5}, {-5, 0}} {
			if !v[off] {
				t.Fatalf("P%d villager offsets %v", s.Player, v)
			}
		}
		for _, u := range s.Objects {
			if u.Unit == 350 || u.Unit == 2567 {
				if dx, dy := u.X-s.X, u.Y-s.Y; dx <= 8 && dx >= -8 && dy <= 8 && dy >= -8 {
					t.Fatalf("tree in flora exclusion: %+v", u)
				}
			}
		}
	}
	found := false
	for _, u := range r.Recipe.Units {
		if u.UnitConst == 349 && *u.X == 100.5 && *u.Y == 100.5 {
			found = true
		}
	}
	if !found {
		t.Fatalf("cleared vegetation beyond the TC exclusion and resource approaches: starts=%+v removed=%d", r.PlayerStarts.Positions, r.PlayerStarts.RemovedAmbient)
	}
	if r.PlayerStarts.Config.FloraClearance != 6 {
		t.Fatal("wrong footprint-edge clearance")
	}
}
