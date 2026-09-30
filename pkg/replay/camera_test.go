package replay

import (
	"aoe2kit/pkg/testfixtures"
	"os"
	"testing"
)

func TestCameraGoldenLatestCBA(t *testing.T) {
	path := testfixtures.Path(t, "save-analysis/latest_cba.aoe2record")
	if _, err := os.Stat(path); err != nil {
		t.Skipf("golden replay not present: %v", err)
	}
	report, err := BuildCamera(path, CameraOptions{Limit: 5, Tail: -1})
	if err != nil {
		t.Fatalf("BuildCamera failed: %v", err)
	}
	if report.Summary.Events != 26878 {
		t.Fatalf("expected 26878 viewlock events, got %d", report.Summary.Events)
	}
	if report.Summary.DistinctTails != 1 {
		t.Fatalf("expected a single distinct tail value, got %d", report.Summary.DistinctTails)
	}
	if len(report.Streams) != 1 || report.Streams[0].TailU32 != 4 {
		t.Fatalf("expected single stream tail=4 (recording player chrae/P4), got %+v", report.Streams)
	}
	if !report.Summary.TailsMatchRoster {
		t.Fatalf("expected tail to coincide with roster player number")
	}
	if report.Streams[0].PlayerNameIfTail != "chrae" {
		t.Fatalf("expected inferred player name chrae, got %q", report.Streams[0].PlayerNameIfTail)
	}
	if report.Streams[0].DistanceTraveled <= 0 {
		t.Fatalf("expected nonzero camera distance for a real game")
	}
	if len(report.Events) != 5 {
		t.Fatalf("expected limit=5 emitted events, got %d", len(report.Events))
	}
	if report.Verification != "structure_verified_xy_decoded_tail_ownership_unverified" {
		t.Fatalf("unexpected verification label: %s", report.Verification)
	}
}
