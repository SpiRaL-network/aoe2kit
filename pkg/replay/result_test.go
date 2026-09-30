package replay

import "testing"

func TestInferResultPartialResignDoesNotCrownSurvivors(t *testing.T) {
	players := []FeedbackPlayer{
		{PlayerID: 1, Kind: "human"},
		{PlayerID: 2, Kind: "human"},
		{PlayerID: 3, Kind: "human"},
		{PlayerID: 4, Kind: "human"},
	}
	events := []ReplayEvent{
		{Type: "resign", PlayerID: 3, TimeMS: 1000, Time: "00:00:01.000"},
		{Type: "resign", PlayerID: 4, TimeMS: 2000, Time: "00:00:02.000"},
	}

	result := InferResult(players, events, EventCounts{DurationMS: 2000, Duration: "00:00:02.000"}, nil)

	if !result.Completed {
		t.Fatal("partial resign stream should mark completion observed")
	}
	if result.WinnerKnown {
		t.Fatalf("winner_known = true, want false for multi-survivor partial resign stream: %+v", result)
	}
	if len(result.Winners) != 0 {
		t.Fatalf("winners = %v, want none without team/outcome corroboration", result.Winners)
	}
	if len(result.Losers) != 2 || result.Losers[0] != 3 || result.Losers[1] != 4 {
		t.Fatalf("losers = %v, want resigned players [3 4]", result.Losers)
	}
	if result.Method != "action_stream_resign_partial_without_team_outcome" {
		t.Fatalf("method = %q", result.Method)
	}
	if len(result.Warnings) == 0 {
		t.Fatal("expected retention-boundary warning")
	}
}

func TestInferResultSingleSurvivorStillKnown(t *testing.T) {
	players := []FeedbackPlayer{
		{PlayerID: 1, Kind: "human"},
		{PlayerID: 2, Kind: "human"},
	}
	events := []ReplayEvent{
		{Type: "resign", PlayerID: 2, TimeMS: 1000, Time: "00:00:01.000"},
	}

	result := InferResult(players, events, EventCounts{DurationMS: 1000, Duration: "00:00:01.000"}, nil)

	if !result.WinnerKnown || len(result.Winners) != 1 || result.Winners[0] != 1 {
		t.Fatalf("winner inference = %+v, want P1 single survivor", result)
	}
	if result.Method != "action_stream_resign_last_single_survivor" {
		t.Fatalf("method = %q", result.Method)
	}
}
