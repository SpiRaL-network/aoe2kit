package scenario

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestPlayableBlankStarterDefaults(t *testing.T) {
	for _, players := range []int{1, 4, 8} {
		path := filepath.Join(t.TempDir(), "blank.aoe2scenario")
		r, e := WriteBlankScenarioFile(path, BlankOptions{PlayerCount: players})
		if e != nil {
			t.Fatal(e)
		}
		f, e := Open(path)
		if e != nil {
			t.Fatal(e)
		}
		active := activeNonGaiaPlayers(f.Players)
		if r.PlayerCount != len(active) || f.Units.Total != len(active) {
			t.Fatal(r, active)
		}
		for _, p := range active {
			u := f.Units.Sections[p].Units
			if len(u) != 1 || u[0].UnitConst != 12 {
				t.Fatalf("P%d: %+v", p, u)
			}
		}
		if hasIssueCode(f.LintWithOptions(LintOptions{LoadSafety: true}).Issues, "load_safety_missing_starting_building") {
			t.Fatal("default blank has missing starter")
		}
	}
}

func TestLoadSafetyStarterConventionAndEmptyOptOut(t *testing.T) {
	path := filepath.Join(t.TempDir(), "blank.aoe2scenario")
	if _, e := WriteBlankScenarioFile(path, BlankOptions{NoStarters: true}); e != nil {
		t.Fatal(e)
	}
	f, e := Open(path)
	if e != nil {
		t.Fatal(e)
	}
	if f.Units.Total != 0 {
		t.Fatal("empty opt-out added units")
	}
	for _, kind := range []int{598, 83, 12} {
		f.Units.Sections[1].Units = []UnitSummary{{UnitConst: kind}}
		f.Units.Sections[2].Units = []UnitSummary{{UnitConst: 12}}
		// Gaia's barracks cannot satisfy P1's obligation.
		f.Units.Sections[0].Units = []UnitSummary{{UnitConst: 12}}
		r := LintReport{OK: true}
		lintLoadSafety(&r, f, LintOptions{LoadSafety: true})
		found := false
		for _, i := range r.Issues {
			if i.Code == "load_safety_missing_starting_building" {
				found = true
				if !strings.Contains(i.Message, "P1 ") || !strings.Contains(i.Fix, "unit 12") {
					t.Fatal(i)
				}
			}
		}
		if found != (kind != 12) {
			t.Fatalf("unit %d warning=%t", kind, found)
		}
	}
}

func TestLoadSafetyReportsLaunchCheckpoint(t *testing.T) {
	path := filepath.Join(t.TempDir(), "blank.aoe2scenario")
	if _, err := WriteBlankScenarioFile(path, BlankOptions{PlayerCount: 1, NoStarters: true}); err != nil {
		t.Fatal(err)
	}
	f, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	report := f.LintWithOptions(LintOptions{LoadSafety: true})
	for _, issue := range report.Issues {
		if issue.Code == "load_safety_missing_starting_building" && issue.Checkpoint != "C4" {
			t.Fatalf("missing starter checkpoint = %q, want C4", issue.Checkpoint)
		}
	}
	if !hasIssueCode(report.Issues, "load_safety_missing_starting_building") {
		t.Fatal("expected missing-starter finding")
	}
}
