package replay

import (
	"os"
	"testing"

	"aoe2kit/pkg/testfixtures"
)

func TestChecksumPhaseWord9FrozenMPFixture(t *testing.T) {
	path := testfixtures.Path(t, "save-analysis/diagnostics/MP Replay v101.103.48987.0 @2026.08.18 180004 (1).aoe2record")
	if _, err := os.Stat(path); err != nil {
		t.Skipf("golden replay not present: %v", err)
	}
	report, err := BuildChecksumPhase(path, ChecksumPhaseOptions{WordIndex: 9, PlayerID: 1})
	if err != nil {
		t.Fatalf("BuildChecksumPhase failed: %v", err)
	}
	if report.Summary.WordIndex != 9 || len(report.Players) != 1 {
		t.Fatalf("summary word/players = %d/%d, want 9/1", report.Summary.WordIndex, len(report.Players))
	}
	player := report.Players[0]
	if player.DistinctValues != 5 {
		t.Fatalf("distinct word_9 values = %d, want 5", player.DistinctValues)
	}
	if player.KnownStateChanges != 0 {
		t.Fatalf("known state changes = %d, want 0", player.KnownStateChanges)
	}
	if player.Transitions != 51 {
		t.Fatalf("transitions = %d, want 51", player.Transitions)
	}
	if player.DominantTransitions != 48 {
		t.Fatalf("dominant transitions = %d, want 48", player.DominantTransitions)
	}
	if player.RepeatTransitions != 0 {
		t.Fatalf("repeat transitions = %d, want 0", player.RepeatTransitions)
	}
	if player.Ring == nil || !player.Ring.Detected || player.Ring.Length != 5 || player.Ring.ForwardSkips != 3 {
		t.Fatalf("ring = %+v, want detected length 5 with 3 forward skips", player.Ring)
	}
}

func TestChecksumPhaseAllWordsFrozenMPFixture(t *testing.T) {
	path := testfixtures.Path(t, "save-analysis/diagnostics/MP Replay v101.103.48987.0 @2026.08.18 180004 (1).aoe2record")
	if _, err := os.Stat(path); err != nil {
		t.Skipf("golden replay not present: %v", err)
	}
	report, err := BuildChecksumPhaseAll(path, ChecksumPhaseOptions{PlayerID: 1})
	if err != nil {
		t.Fatalf("BuildChecksumPhaseAll failed: %v", err)
	}
	if len(report.Words) != 11 {
		t.Fatalf("word reports = %d, want 11", len(report.Words))
	}
	for _, wordReport := range report.Words {
		if len(wordReport.Players) != 1 {
			t.Fatalf("word %d players = %d, want 1", wordReport.Summary.WordIndex, len(wordReport.Players))
		}
		player := wordReport.Players[0]
		if wordReport.Summary.WordIndex == 9 {
			if player.DistinctValues != 5 || player.RepeatTransitions != 0 {
				t.Fatalf("word 9 distinct/repeats = %d/%d, want 5/0", player.DistinctValues, player.RepeatTransitions)
			}
			continue
		}
		if player.DistinctValues != 1 || player.RepeatTransitions != 51 || player.DominantTransitionShare != 1 {
			t.Fatalf("word %d distinct/repeats/share = %d/%d/%f, want 1/51/1",
				wordReport.Summary.WordIndex, player.DistinctValues, player.RepeatTransitions, player.DominantTransitionShare)
		}
	}
}
