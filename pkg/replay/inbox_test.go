package replay

import "testing"

func TestBuildIssueCardsGroupsRepeatedText(t *testing.T) {
	events := []ReplayEvent{
		{Type: "chat", TimeMS: 1500, Time: FormatTime(1500), PlayerID: 1, PlayerName: "One", Text: "Need help", Source: "action_stream"},
		{Type: "chat", TimeMS: 2000, Time: FormatTime(2000), PlayerID: 1, PlayerName: "One", Text: " need   help ", Source: "action_stream"},
		{Type: "flare", TimeMS: 400000, Time: FormatTime(400000), PlayerID: 2, PlayerName: "Two", X: 10, Y: 20, Region: "Arena", Source: "action_stream"},
	}
	cards := BuildIssueCards("replay.zip", StoryIdentity{Tier: "trigger_graph", SHA256: "abc"}, events, 600000)
	if len(cards) != 2 {
		t.Fatalf("cards len = %d, want 2: %+v", len(cards), cards)
	}
	if cards[0].Count != 2 || cards[0].Text != "Need help" || cards[0].Confidence != "replay_visible" {
		t.Fatalf("chat card = %+v", cards[0])
	}
	if cards[1].Type != "flare" || cards[1].Region != "Arena" || cards[1].Confidence != "replay_action_verified" {
		t.Fatalf("flare card = %+v", cards[1])
	}
}

func TestMergeIssueCardsAcrossReplays(t *testing.T) {
	identity := StoryIdentity{Tier: "trigger_graph", SHA256: "abc"}
	cards := []IssueCard{
		{Key: "chat|opening|p1|help", Replay: "a.zip", Scenario: identity, Count: 1, FirstTimeMS: 1000, LastTimeMS: 1000, FirstTime: FormatTime(1000), LastTime: FormatTime(1000), Evidence: []IssueEvidence{{Replay: "a.zip"}}},
		{Key: "chat|opening|p1|help", Replay: "b.zip", Scenario: identity, Count: 2, FirstTimeMS: 500, LastTimeMS: 2000, FirstTime: FormatTime(500), LastTime: FormatTime(2000), Evidence: []IssueEvidence{{Replay: "b.zip"}}},
	}
	merged := mergeIssueCards(cards)
	if len(merged) != 1 {
		t.Fatalf("merged len = %d, want 1: %+v", len(merged), merged)
	}
	if merged[0].Count != 3 || merged[0].Replay != "b.zip" || len(merged[0].Evidence) != 2 {
		t.Fatalf("merged card = %+v", merged[0])
	}
}

func TestSummarizeInbox(t *testing.T) {
	report := &InboxReport{
		ScenarioGroups: []InboxScenarioGroup{{Key: "known"}},
		IssueCards:     []IssueCard{{ID: "issue-0001"}},
		Duplicates:     []Duplicate{{Paths: []string{"a", "b"}}},
		Items: []InboxItem{
			{
				Identity:        StoryIdentity{Tier: "trigger_graph", SHA256: "abc"},
				DataSet:         DataSetIdentity{Status: "vanilla"},
				EventCounts:     EventCounts{Chat: 2, Flares: 1, Actions: 10, UntypedActions: 1},
				ProfileCoverage: ProfileCoverage{ReplayActions: 10, DecodedPercent: 90},
				Result:          MatchResult{Completed: true, WinnerKnown: true},
			},
			{Error: "bad replay"},
		},
	}
	summary := summarizeInbox(report)
	if summary.Parsed != 1 || summary.Errors != 1 || summary.DuplicateItems != 1 || summary.FeedbackEvents != 3 || summary.AverageDecodedPercent != 90 {
		t.Fatalf("summary = %+v", summary)
	}
}
