package replay

import "testing"

func TestExtractHeaderLobbyChatAGP(t *testing.T) {
	header := []byte("noise\x00@#2 <platform_icon_world> Alice: ready?\x00more @#9 <x> Bad<Name>: nope\x00")
	lines := extractHeaderLobbyChat(header, map[string]int{"alice": 4})
	if len(lines) != 1 {
		t.Fatalf("lines = %d, want 1: %#v", len(lines), lines)
	}
	line := lines[0]
	if line.Source != "lobby" || line.PlayerID != 4 || line.PlayerName != "Alice" || line.Text != "ready?" || line.PlatformIcon != "platform_icon_world" {
		t.Fatalf("line = %+v", line)
	}
}

func TestChatDedupeKeyNormalizesWhitespace(t *testing.T) {
	a := chatDedupeKey(ChatLine{PlayerName: "Alice", Text: "Need   help"})
	b := chatDedupeKey(ChatLine{PlayerName: " alice ", Text: " need help "})
	if a != b {
		t.Fatalf("keys differ: %q != %q", a, b)
	}
}

func TestTagInitialChatBacklog(t *testing.T) {
	lines := []ChatLine{
		{Source: "game", TimeMS: 3054, Text: "old 1", Confidence: "parsed"},
		{Source: "game", TimeMS: 3054, Text: "old 2", Confidence: "parsed"},
		{Source: "game", TimeMS: 3054, Text: "old 3", Confidence: "parsed"},
		{Source: "game", TimeMS: 3054, Text: "old 4", Confidence: "parsed"},
		{Source: "game", TimeMS: 3054, Text: "old 5", Confidence: "parsed"},
		{Source: "game", TimeMS: 20000, Text: "live", Confidence: "parsed"},
	}
	if got := tagInitialChatBacklog(lines); got != 5 {
		t.Fatalf("tagged = %d, want 5", got)
	}
	for i := 0; i < 5; i++ {
		if lines[i].Source != "backlog" || lines[i].Confidence != "parsed+same_timestamp_backlog_heuristic" {
			t.Fatalf("line %d = %+v", i, lines[i])
		}
	}
	if lines[5].Source != "game" {
		t.Fatalf("later line source = %q", lines[5].Source)
	}
}

func TestTagInitialChatBacklogHandlesDelayedBurstAndKeepsDiagnosticMarker(t *testing.T) {
	lines := []ChatLine{
		{Source: "game", TimeMS: 92768, Text: "old 1", Confidence: "parsed"},
		{Source: "game", TimeMS: 92768, Text: "old 2", Confidence: "parsed"},
		{Source: "game", TimeMS: 92768, Text: "old 3", Confidence: "parsed"},
		{Source: "game", TimeMS: 92768, Text: "old 4", Confidence: "parsed"},
		{Source: "game", TimeMS: 92768, Text: "old 5", Confidence: "parsed"},
		{Source: "game", TimeMS: 92768, Text: "040 start", Confidence: "parsed"},
		{Source: "game", TimeMS: 110488, Text: "040 done", Confidence: "parsed"},
	}
	if got := tagInitialChatBacklog(lines); got != 5 {
		t.Fatalf("tagged = %d, want 5", got)
	}
	for i := 0; i < 5; i++ {
		if lines[i].Source != "backlog" {
			t.Fatalf("line %d source = %q", i, lines[i].Source)
		}
	}
	if lines[5].Source != "game" || lines[6].Source != "game" {
		t.Fatalf("diagnostic markers were tagged: %+v %+v", lines[5], lines[6])
	}
}

func TestTagInitialChatBacklogDoesNotPreserveZeroTimeDiagnosticMarkers(t *testing.T) {
	lines := []ChatLine{
		{Source: "game", TimeMS: 0, Text: "old 1", Confidence: "parsed"},
		{Source: "game", TimeMS: 0, Text: "old 2", Confidence: "parsed"},
		{Source: "game", TimeMS: 0, Text: "old 3", Confidence: "parsed"},
		{Source: "game", TimeMS: 0, Text: "040 start", Confidence: "parsed"},
		{Source: "game", TimeMS: 0, Text: "040 done", Confidence: "parsed"},
	}
	if got := tagInitialChatBacklog(lines); got != 5 {
		t.Fatalf("tagged = %d, want 5", got)
	}
	for i, line := range lines {
		if line.Source != "backlog" {
			t.Fatalf("line %d source = %q", i, line.Source)
		}
	}
}

func TestTagInitialChatBacklogIgnoresSmallOpeningChat(t *testing.T) {
	lines := []ChatLine{
		{Source: "game", TimeMS: 3054, Text: "gl"},
		{Source: "game", TimeMS: 3054, Text: "hf"},
		{Source: "game", TimeMS: 3054, Text: "ready"},
	}
	if got := tagInitialChatBacklog(lines); got != 0 {
		t.Fatalf("tagged = %d, want 0", got)
	}
	for i, line := range lines {
		if line.Source != "game" {
			t.Fatalf("line %d source = %q", i, line.Source)
		}
	}
}

func TestTagInitialChatBacklogTagsEarlyExactBacklogRepeats(t *testing.T) {
	lines := []ChatLine{
		{Source: "game", TimeMS: 124, Text: "old 1", Confidence: "parsed"},
		{Source: "game", TimeMS: 124, Text: "old 2", Confidence: "parsed"},
		{Source: "game", TimeMS: 124, Text: "and p5 never had a starter barracks and therefore has a TC", Confidence: "parsed"},
		{Source: "game", TimeMS: 124, Text: "so at least the \"place one item\" trick is confirmed", Confidence: "parsed"},
		{Source: "game", TimeMS: 124, Text: "old 5", Confidence: "parsed"},
		{Source: "game", TimeMS: 30084, Text: "and p5 never had a starter barracks and therefore has a TC", Confidence: "parsed"},
		{Source: "game", TimeMS: 30084, Text: "so at least the \"place one item\" trick is confirmed", Confidence: "parsed"},
		{Source: "game", TimeMS: 55358, Text: "who wants to swap with me", Confidence: "parsed"},
		{Source: "game", TimeMS: 756288, Text: "set", Confidence: "parsed"},
	}
	if got := tagInitialChatBacklog(lines); got != 7 {
		t.Fatalf("tagged = %d, want 7", got)
	}
	for _, idx := range []int{5, 6} {
		if lines[idx].Source != "backlog" || lines[idx].Confidence != "parsed+exact_backlog_text_repeat_heuristic" {
			t.Fatalf("repeat line %d = %+v, want backlog exact-repeat confidence", idx, lines[idx])
		}
	}
	for _, idx := range []int{7, 8} {
		if lines[idx].Source != "game" {
			t.Fatalf("live line %d = %+v, want game", idx, lines[idx])
		}
	}
}
