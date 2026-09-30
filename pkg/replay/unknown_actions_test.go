package replay

import (
	"aoe2kit/pkg/testfixtures"
	"os"
	"testing"
)

func TestUnknownActionsRatchetLatestCBA(t *testing.T) {
	path := testfixtures.Path(t, "save-analysis/latest_cba.aoe2record")
	if _, err := os.Stat(path); err != nil {
		t.Skipf("golden replay not present: %v", err)
	}
	report, err := BuildUnknownActions(path, 2)
	if err != nil {
		t.Fatalf("BuildUnknownActions failed: %v", err)
	}
	if report.TotalActions != 11510 {
		t.Fatalf("expected 11510 total actions, got %d", report.TotalActions)
	}
	// After the RATHA_ABILITY decode + mgz name adds, only id 42 and 140... (140 named)
	// remain: 30x id-42 + 3x id-140 named -> unknown = 33 - 3 named 140 = 30.
	for _, g := range report.Groups {
		if g.ActionID == 43 || g.ActionID == 19 || g.ActionID == 140 {
			t.Fatalf("action id %d should no longer be unknown", g.ActionID)
		}
	}
	if len(report.Groups) != 1 || report.Groups[0].ActionID != 42 {
		t.Fatalf("expected only action id 42 to remain unknown, got %+v", report.Groups)
	}
}

func TestRathaAbilityDecodeLatestCBA(t *testing.T) {
	path := testfixtures.Path(t, "save-analysis/latest_cba.aoe2record")
	if _, err := os.Stat(path); err != nil {
		t.Skipf("golden replay not present: %v", err)
	}
	report, err := ExtractEvents(path, EventOptions{IncludeUntypedAction: true})
	if err != nil {
		t.Fatalf("ExtractEvents failed: %v", err)
	}
	seen := 0
	for _, event := range report.Events {
		if event.Type != "ratha_ability" {
			continue
		}
		seen++
		if event.PlayerID != 1 {
			t.Fatalf("expected only P1 (Bengalis) ratha abilities, got player %d", event.PlayerID)
		}
		if event.ModeID != 153 && event.ModeID != 154 {
			t.Fatalf("expected mode 153 or 154, got %d", event.ModeID)
		}
		if len(event.ObjectIDs) == 0 {
			t.Fatalf("expected object ids on ratha ability event")
		}
	}
	if seen != 10 {
		t.Fatalf("expected 10 decoded ratha abilities, got %d", seen)
	}
}
