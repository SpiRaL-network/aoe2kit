package replay

import (
	"aoe2kit/pkg/testfixtures"
	"os"
	"testing"
)

func TestOpaqueTargetGoldenLatestCBA(t *testing.T) {
	path := testfixtures.Path(t, "save-analysis/latest_cba.aoe2record")
	if _, err := os.Stat(path); err != nil {
		t.Skipf("golden replay not present: %v", err)
	}
	report, err := BuildOpaqueTarget(path)
	if err != nil {
		t.Fatalf("BuildOpaqueTarget failed: %v", err)
	}
	if len(report.PlayerTails) != 7 {
		t.Fatalf("expected 7 player tails (P1 ref + P2..P7), got %d", len(report.PlayerTails))
	}
	if report.SelectedTarget != "per_player_tail_sparse_differences" {
		t.Fatalf("expected sparse-difference target, got %s", report.SelectedTarget)
	}
	for _, tail := range report.PlayerTails[1:] {
		if tail.IdenticalRatio < 0.95 {
			t.Fatalf("expected >=95%% aligned identity for %s, got %.4f", tail.Label, tail.IdenticalRatio)
		}
	}
}
