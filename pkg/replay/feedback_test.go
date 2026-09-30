package replay

import (
	"bytes"
	"encoding/binary"
	"math"
	"strings"
	"testing"
)

func floatNear(a, b, tolerance float64) bool {
	return math.Abs(a-b) <= tolerance
}

func TestExtractFeedbackEventsDedupesChatJSON(t *testing.T) {
	body := []byte(`noise {"player":1,"channel":0,"message":"hello","tauntNumber":0,"messageAGP":"@#1 <ALL> chrae: hello","destinationMap":1020}` +
		` echo {"player":1,"channel":0,"message":"hello","tauntNumber":0,"destinationMap":1020}` +
		` next {"player":2,"channel":1,"message":"14","tauntNumber":14,"destinationMap":1020}`)
	events, names, warnings := ExtractFeedbackEvents(body)
	if len(warnings) != 0 {
		t.Fatalf("warnings = %v", warnings)
	}
	if len(events) != 2 {
		t.Fatalf("events len = %d, want 2", len(events))
	}
	if events[0].PlayerID != 1 || events[0].Text != "hello" || events[0].ChannelName != "all" {
		t.Fatalf("first event = %+v", events[0])
	}
	if got := names[1]; got != "chrae" {
		t.Fatalf("name from messageAGP = %q, want chrae", got)
	}
	if events[1].PlayerID != 2 || events[1].ChannelName != "team" || events[1].TauntNumber != 14 {
		t.Fatalf("second event = %+v", events[1])
	}
}

func TestExtractActionStreamFeedbackChatAndFlare(t *testing.T) {
	var body bytes.Buffer
	writeU32(&body, 5)
	writeU32(&body, 500)
	writeU32(&body, 1)
	writeU32(&body, 14)
	writeU32(&body, 1)
	writeU32(&body, 2)
	writeU32(&body, 0)
	writeU32(&body, 0)
	writeU32(&body, 2)
	writeU32(&body, 1000)
	rawChat := []byte(`{"player":1,"channel":0,"message":"hello","tauntNumber":14,"destinationMap":1020}`)
	writeU32(&body, 4)
	writeU32(&body, 0)
	writeU32(&body, uint32(len(rawChat)))
	body.Write(rawChat)
	flare := make([]byte, 82)
	binary.LittleEndian.PutUint32(flare[4:8], math.Float32bits(12.5))
	binary.LittleEndian.PutUint32(flare[44:48], math.Float32bits(34.5))
	writeU32(&body, 1)
	writeU32(&body, uint32(len(flare)+1))
	body.WriteByte(115)
	body.Write(flare)
	writeU32(&body, 7)
	writeU32(&body, 6)
	events, _, warnings := ExtractActionStreamFeedback(body.Bytes())
	if len(warnings) != 0 {
		t.Fatalf("warnings = %v", warnings)
	}
	if len(events) != 2 {
		t.Fatalf("events len = %d, want 2: %+v", len(events), events)
	}
	if events[0].Kind != "CHAT" || events[0].TimeMS != 1000 || events[0].TauntNumber != 14 {
		t.Fatalf("chat event = %+v", events[0])
	}
	if events[1].Kind != "FLARE" || events[1].X != 12.5 || events[1].Y != 34.5 {
		t.Fatalf("flare event = %+v", events[1])
	}
}

func TestExtractActionStreamEventsTelemetryAndCounts(t *testing.T) {
	var body bytes.Buffer
	writeU32(&body, 5)
	writeU32(&body, 500)
	writeU32(&body, 1)
	writeU32(&body, 14)
	writeU32(&body, 1)
	writeU32(&body, 2)
	writeU32(&body, 0)
	writeU32(&body, 0)
	writeU32(&body, 2)
	writeU32(&body, 2500)
	rawChat := []byte(`{"player":1,"channel":0,"message":"SDS EVT trial_start id=oracle","tauntNumber":0,"destinationMap":1020}`)
	writeU32(&body, 4)
	writeU32(&body, 0)
	writeU32(&body, uint32(len(rawChat)))
	body.Write(rawChat)
	action := []byte{1, 2, 3}
	writeU32(&body, 1)
	writeU32(&body, uint32(len(action)+1))
	body.WriteByte(77)
	body.Write(action)
	writeU32(&body, 9)
	writeU32(&body, 6)

	events, _, counts, warnings := ExtractActionStreamEvents(body.Bytes(), EventOptions{})
	if len(warnings) != 0 {
		t.Fatalf("warnings = %v", warnings)
	}
	if len(events) != 1 {
		t.Fatalf("events len = %d, want 1: %+v", len(events), events)
	}
	if events[0].Telemetry == nil || events[0].Telemetry.Name != "trial_start" || events[0].Telemetry.Fields["id"] != "oracle" {
		t.Fatalf("telemetry = %+v", events[0].Telemetry)
	}
	if counts.Actions != 1 || counts.UntypedActions != 1 || counts.ActionIDs[77] != 1 {
		t.Fatalf("counts did not preserve untyped action: %+v", counts)
	}
}

func TestExtractActionStreamEventsPreservesNamedRawAction(t *testing.T) {
	var body bytes.Buffer
	writeReplayMetaFixture(&body)
	writeU32(&body, 2)
	writeU32(&body, 1000)
	payload := []byte{1, 0, 0}
	writeU32(&body, 1)
	writeU32(&body, uint32(len(payload)+1))
	body.WriteByte(byte(actionDEAutoscout))
	body.Write(payload)
	writeU32(&body, 6)

	events, _, counts, warnings := ExtractActionStreamEvents(body.Bytes(), EventOptions{IncludeUntypedAction: true})
	if len(warnings) != 0 {
		t.Fatalf("warnings = %v", warnings)
	}
	if counts.Actions != 1 || counts.UntypedActions != 0 || counts.ActionIDs[actionDEAutoscout] != 1 {
		t.Fatalf("counts did not preserve named raw action: %+v", counts)
	}
	if len(events) != 1 {
		t.Fatalf("events len = %d, want 1: %+v", len(events), events)
	}
	if events[0].ActionName != "DE_AUTOSCOUT" || events[0].Confidence != "raw_preserved_named_action" {
		t.Fatalf("event = %+v, want named raw DE_AUTOSCOUT", events[0])
	}
}

func TestExtractActionStreamEventsSkipsSaveChapter(t *testing.T) {
	var body bytes.Buffer
	writeReplayMetaFixture(&body)
	writeU32(&body, 2)
	writeU32(&body, 1000)
	saveStart := body.Len()
	writeU32(&body, uint32(saveStart+11))
	writeU32(&body, 123)
	body.Write([]byte{1, 2, 3})
	writeU32(&body, 6)

	events, _, counts, warnings := ExtractActionStreamEvents(body.Bytes(), EventOptions{IncludeSystemEvents: true})
	if len(warnings) != 0 {
		t.Fatalf("warnings = %v", warnings)
	}
	if counts.Saves != 1 || counts.Postgames != 1 || counts.DurationMS != 1000 {
		t.Fatalf("counts = %+v", counts)
	}
	if got := events[len(events)-1].Type; got != "postgame" {
		t.Fatalf("last event = %q, want postgame", got)
	}
}

func TestExtractActionStreamEventsConsumesDESyncPayload(t *testing.T) {
	var body bytes.Buffer
	writeReplayMetaFixture(&body)
	writeU32(&body, 2)
	writeU32(&body, 1000)
	writeU32(&body, 0)
	probe := make([]byte, 16)
	binary.LittleEndian.PutUint32(probe[12:16], 1)
	body.Write(probe)
	body.Write(make([]byte, 8*11*4-16))
	writeU32(&body, 1000)
	writeU32(&body, 6)

	events, _, counts, warnings := ExtractActionStreamEvents(body.Bytes(), EventOptions{IncludeSystemEvents: true})
	if len(warnings) != 0 {
		t.Fatalf("warnings = %v", warnings)
	}
	if counts.Syncs != 1 || counts.Postgames != 1 || counts.UnknownOps != 0 || counts.DurationMS != 1000 {
		t.Fatalf("counts = %+v", counts)
	}
	if len(events) != 2 || events[0].Type != "sync" || events[1].Type != "postgame" {
		t.Fatalf("events = %+v", events)
	}
}

func TestDecodeActionEventPreservesOrderActionZero(t *testing.T) {
	actionBody := make([]byte, 24)
	binary.LittleEndian.PutUint32(actionBody[0:4], 999)
	binary.LittleEndian.PutUint32(actionBody[4:8], math.Float32bits(12.5))
	binary.LittleEndian.PutUint32(actionBody[8:12], math.Float32bits(34.5))
	binary.LittleEndian.PutUint16(actionBody[12:14], 1)
	binary.LittleEndian.PutUint32(actionBody[20:24], 456)
	payload := append([]byte{2, byte(len(actionBody)), 0}, actionBody...)

	event, ok := DecodeActionEvent(actionOrder, payload, 500, 42, false)
	if !ok {
		t.Fatal("DecodeActionEvent returned !ok")
	}
	if !event.ReplayAction || event.ActionID != 0 || event.ActionName != "ORDER" {
		t.Fatalf("action identity = %+v", event)
	}
	if event.Type != "order" || event.PlayerID != 2 || event.TargetID != 999 || len(event.ObjectIDs) != 1 || event.ObjectIDs[0] != 456 {
		t.Fatalf("decoded event = %+v", event)
	}
	counts := CountReplayEvents([]ReplayEvent{event})
	if counts.ActionIDs[0] != 1 || counts.SpatialActions != 1 {
		t.Fatalf("counts = %+v", counts)
	}
}

func TestBacklogChatCountsAndFeedbackFiltering(t *testing.T) {
	events := []ReplayEvent{
		{Index: 1, Type: "chat", TimeMS: 3054, Text: "old 1", Source: "action_stream", Confidence: "parsed"},
		{Index: 2, Type: "chat", TimeMS: 3054, Text: "old 2", Source: "action_stream", Confidence: "parsed"},
		{Index: 3, Type: "chat", TimeMS: 3054, Text: "old 3", Source: "action_stream", Confidence: "parsed"},
		{Index: 4, Type: "chat", TimeMS: 3054, Text: "old 4", Source: "action_stream", Confidence: "parsed"},
		{Index: 5, Type: "chat", TimeMS: 3054, Text: "old 5", Source: "action_stream", Confidence: "parsed"},
		{Index: 6, Type: "chat", TimeMS: 20000, Text: "live", Source: "action_stream", Confidence: "parsed"},
		{Index: 7, Type: "flare", TimeMS: 22000, X: 1, Y: 2, Source: "action_stream", Confidence: "parsed"},
	}
	if got := tagInitialChatBacklogEvents(events); got != 5 {
		t.Fatalf("tagged = %d, want 5", got)
	}
	counts := CountReplayEvents(events)
	if counts.Total != 7 || counts.BacklogChat != 5 || counts.Chat != 1 || counts.Flares != 1 {
		t.Fatalf("counts = %+v", counts)
	}
	feedback := eventsToFeedback(events)
	if len(feedback) != 2 || feedback[0].Text != "live" || feedback[1].Kind != "FLARE" {
		t.Fatalf("feedback = %+v", feedback)
	}
}

func TestBacklogChatKeepsDiagnosticMarkersInDelayedBurst(t *testing.T) {
	events := []ReplayEvent{
		{Index: 1, Type: "chat", TimeMS: 92768, Text: "old 1", Source: "action_stream", Confidence: "parsed"},
		{Index: 2, Type: "chat", TimeMS: 92768, Text: "old 2", Source: "action_stream", Confidence: "parsed"},
		{Index: 3, Type: "chat", TimeMS: 92768, Text: "old 3", Source: "action_stream", Confidence: "parsed"},
		{Index: 4, Type: "chat", TimeMS: 92768, Text: "old 4", Source: "action_stream", Confidence: "parsed"},
		{Index: 5, Type: "chat", TimeMS: 92768, Text: "old 5", Source: "action_stream", Confidence: "parsed"},
		{Index: 6, Type: "chat", TimeMS: 92768, Text: "040 start", Source: "action_stream", Confidence: "parsed"},
		{Index: 7, Type: "chat", TimeMS: 110488, Text: "040 done", Source: "action_stream", Confidence: "parsed"},
	}
	if got := tagInitialChatBacklogEvents(events); got != 5 {
		t.Fatalf("tagged = %d, want 5", got)
	}
	counts := CountReplayEvents(events)
	if counts.BacklogChat != 5 || counts.Chat != 2 {
		t.Fatalf("counts = %+v", counts)
	}
	feedback := eventsToFeedback(events)
	if len(feedback) != 2 || feedback[0].Text != "040 start" || feedback[1].Text != "040 done" {
		t.Fatalf("feedback = %+v", feedback)
	}
}

func TestBacklogChatTagsEarlyExactBacklogRepeats(t *testing.T) {
	events := []ReplayEvent{
		{Index: 1, Type: "chat", TimeMS: 124, Text: "old 1", Source: "action_stream", Confidence: "parsed"},
		{Index: 2, Type: "chat", TimeMS: 124, Text: "old 2", Source: "action_stream", Confidence: "parsed"},
		{Index: 3, Type: "chat", TimeMS: 124, Text: "and p5 never had a starter barracks and therefore has a TC", Source: "action_stream", Confidence: "parsed"},
		{Index: 4, Type: "chat", TimeMS: 124, Text: "so at least the \"place one item\" trick is confirmed", Source: "action_stream", Confidence: "parsed"},
		{Index: 5, Type: "chat", TimeMS: 124, Text: "old 5", Source: "action_stream", Confidence: "parsed"},
		{Index: 6, Type: "chat", TimeMS: 30084, Text: "and p5 never had a starter barracks and therefore has a TC", Source: "action_stream", Confidence: "parsed"},
		{Index: 7, Type: "chat", TimeMS: 30084, Text: "so at least the \"place one item\" trick is confirmed", Source: "action_stream", Confidence: "parsed"},
		{Index: 8, Type: "chat", TimeMS: 55358, Text: "who wants to swap with me", Source: "action_stream", Confidence: "parsed"},
		{Index: 9, Type: "chat", TimeMS: 756288, Text: "set", Source: "action_stream", Confidence: "parsed"},
	}
	if got := tagInitialChatBacklogEvents(events); got != 7 {
		t.Fatalf("tagged = %d, want 7", got)
	}
	counts := CountReplayEvents(events)
	if counts.BacklogChat != 7 || counts.Chat != 2 {
		t.Fatalf("counts = %+v, want backlog 7 game chat 2", counts)
	}
	for _, idx := range []int{5, 6} {
		if events[idx].Source != "backlog" || events[idx].Confidence != "parsed+exact_backlog_text_repeat_heuristic" {
			t.Fatalf("repeat event %d = %+v, want backlog exact-repeat confidence", idx, events[idx])
		}
	}
	feedback := eventsToFeedback(events)
	if len(feedback) != 2 || feedback[0].Text != "who wants to swap with me" || feedback[1].Text != "set" {
		t.Fatalf("feedback = %+v, want only live chat", feedback)
	}
}

func TestBacklogChatDoesNotKeepZeroTimeDiagnosticMarkers(t *testing.T) {
	events := []ReplayEvent{
		{Index: 1, Type: "chat", TimeMS: 0, Text: "old 1", Source: "action_stream", Confidence: "parsed"},
		{Index: 2, Type: "chat", TimeMS: 0, Text: "old 2", Source: "action_stream", Confidence: "parsed"},
		{Index: 3, Type: "chat", TimeMS: 0, Text: "old 3", Source: "action_stream", Confidence: "parsed"},
		{Index: 4, Type: "chat", TimeMS: 0, Text: "040 start", Source: "action_stream", Confidence: "parsed"},
		{Index: 5, Type: "chat", TimeMS: 0, Text: "040 done", Source: "action_stream", Confidence: "parsed"},
	}
	if got := tagInitialChatBacklogEvents(events); got != 5 {
		t.Fatalf("tagged = %d, want 5", got)
	}
	counts := CountReplayEvents(events)
	if counts.BacklogChat != 5 || counts.Chat != 0 {
		t.Fatalf("counts = %+v", counts)
	}
	if feedback := eventsToFeedback(events); len(feedback) != 0 {
		t.Fatalf("feedback = %+v", feedback)
	}
}

func TestStoryIgnoresBacklogChatAsFeedback(t *testing.T) {
	events := []ReplayEvent{
		{Index: 1, Type: "chat", TimeMS: 3054, Time: FormatTime(3054), PlayerID: 1, Text: "old 1", Source: "backlog", Confidence: "parsed+same_timestamp_backlog_heuristic"},
		{Index: 2, Type: "chat", TimeMS: 3054, Time: FormatTime(3054), PlayerID: 1, Text: "old 2", Source: "backlog", Confidence: "parsed+same_timestamp_backlog_heuristic"},
		{Index: 3, Type: "chat", TimeMS: 20000, Time: FormatTime(20000), PlayerID: 1, Text: "live", Source: "action_stream", Confidence: "parsed"},
	}
	if feedback := feedbackStoryLines(events); len(feedback) != 1 || !strings.Contains(feedback[0].Text, "live") {
		t.Fatalf("feedback = %+v", feedback)
	}
	moments := buildStoryMoments(events, 60000)
	found := false
	for _, moment := range moments {
		if moment.Kind == "first_chat" {
			found = true
			if !strings.Contains(moment.Detail, "live") {
				t.Fatalf("first_chat moment = %+v", moment)
			}
		}
	}
	if !found {
		t.Fatalf("first_chat not found: %+v", moments)
	}
}

func TestExtractActionStreamEventsViewlockCoordinates(t *testing.T) {
	var body bytes.Buffer
	writeReplayMetaFixture(&body)
	writeU32(&body, 2)
	writeU32(&body, 1000)
	writeU32(&body, 3)
	var raw [12]byte
	binary.LittleEndian.PutUint32(raw[0:4], math.Float32bits(44.25))
	binary.LittleEndian.PutUint32(raw[4:8], math.Float32bits(55.5))
	body.Write(raw[:])
	writeU32(&body, 6)

	events, _, counts, warnings := ExtractActionStreamEvents(body.Bytes(), EventOptions{IncludeSystemEvents: true})
	if len(warnings) != 0 {
		t.Fatalf("warnings = %v", warnings)
	}
	if counts.Viewlocks != 1 {
		t.Fatalf("viewlocks = %d, want 1", counts.Viewlocks)
	}
	if len(events) != 3 || events[1].Type != "viewlock" {
		t.Fatalf("events = %+v", events)
	}
	if events[1].X != 44.25 || events[1].Y != 55.5 || events[1].Confidence != "camera_observed_no_player" {
		t.Fatalf("viewlock event = %+v", events[1])
	}
}

func TestExtractActionStreamEventsStartOpFourIntShape(t *testing.T) {
	var body bytes.Buffer
	writeReplayMetaFixture(&body)
	writeU32(&body, 2)
	writeU32(&body, 1000)
	writeU32(&body, 5)
	writeU32(&body, 1)
	writeU32(&body, 3835)
	writeU32(&body, 2)
	writeU32(&body, 50)
	writeU32(&body, 3)
	var raw [12]byte
	binary.LittleEndian.PutUint32(raw[0:4], math.Float32bits(169))
	binary.LittleEndian.PutUint32(raw[4:8], math.Float32bits(182))
	binary.LittleEndian.PutUint32(raw[8:12], 6)
	body.Write(raw[:])
	writeU32(&body, 6)

	events, _, counts, warnings := ExtractActionStreamEvents(body.Bytes(), EventOptions{IncludeSystemEvents: true})
	if len(warnings) != 0 {
		t.Fatalf("warnings = %v", warnings)
	}
	if counts.Starts != 1 || counts.Viewlocks != 1 || counts.UnknownOps != 0 {
		t.Fatalf("counts = %+v", counts)
	}
	if len(events) != 4 || events[1].Type != "start" || events[2].Type != "viewlock" {
		t.Fatalf("events = %+v", events)
	}
	if events[2].X != 169 || events[2].Y != 182 {
		t.Fatalf("viewlock = %+v", events[2])
	}
}

func TestExtractActionStreamEventsStartOpTerminalEmbeddedTail(t *testing.T) {
	var body bytes.Buffer
	writeReplayMetaFixture(&body)
	writeU32(&body, 2)
	writeU32(&body, 1000)
	writeU32(&body, 5)
	writeU32(&body, 0)
	writeU32(&body, 2645)
	writeU32(&body, 2)
	writeU32(&body, 62)
	body.Write(make([]byte, 32))

	events, _, counts, warnings := ExtractActionStreamEvents(body.Bytes(), EventOptions{IncludeSystemEvents: true})
	if len(warnings) != 0 {
		t.Fatalf("warnings = %v", warnings)
	}
	if counts.EmbeddedTails != 1 || counts.UnknownOps != 0 {
		t.Fatalf("counts = %+v", counts)
	}
	if len(events) != 2 || events[1].Type != "embedded_tail" {
		t.Fatalf("events = %+v", events)
	}
	if events[1].PayloadBytes != 48 {
		t.Fatalf("embedded tail bytes = %d, want 48", events[1].PayloadBytes)
	}
}

func TestExtractActionStreamEventsShortFlare(t *testing.T) {
	var body bytes.Buffer
	writeReplayMetaFixture(&body)
	writeU32(&body, 2)
	writeU32(&body, 1000)
	writeU32(&body, 1)
	payload := []byte{1, 16, 0}
	var raw [16]byte
	binary.LittleEndian.PutUint32(raw[0:4], 0xffffffff)
	binary.LittleEndian.PutUint32(raw[4:8], math.Float32bits(104.17))
	binary.LittleEndian.PutUint32(raw[8:12], math.Float32bits(44.15))
	raw[12] = 3
	raw[13] = 0
	raw[14] = 1
	raw[15] = 1
	payload = append(payload, raw[:]...)
	writeU32(&body, uint32(1+len(payload)))
	body.WriteByte(byte(actionFlare))
	body.Write(payload)
	writeU32(&body, 1000)
	writeU32(&body, 6)

	events, _, counts, warnings := ExtractActionStreamEvents(body.Bytes(), EventOptions{})
	if len(warnings) != 0 {
		t.Fatalf("warnings = %v", warnings)
	}
	if counts.Flares != 1 || counts.UntypedActions != 0 || counts.SpatialActions != 0 {
		t.Fatalf("counts = %+v", counts)
	}
	if len(events) != 1 || events[0].Type != "flare" || events[0].PlayerID != 1 {
		t.Fatalf("events = %+v", events)
	}
	if !floatNear(events[0].X, 104.17, 0.01) || !floatNear(events[0].Y, 44.15, 0.01) {
		t.Fatalf("flare coords = %+v", events[0])
	}
	if len(events[0].Targets) != 2 || events[0].Targets[0] != 1 || events[0].Targets[1] != 2 {
		t.Fatalf("targets = %v", events[0].Targets)
	}
}

func TestDecodeActionEventQueueAndResearch(t *testing.T) {
	queueBody := make([]byte, 22)
	binary.LittleEndian.PutUint16(queueBody[0:2], 1)
	binary.LittleEndian.PutUint16(queueBody[6:8], 12)
	binary.LittleEndian.PutUint16(queueBody[8:10], 83)
	binary.LittleEndian.PutUint16(queueBody[10:12], 2)
	binary.LittleEndian.PutUint32(queueBody[18:22], 999)
	queuePayload := append([]byte{2, byte(len(queueBody)), 0}, queueBody...)
	queue, ok := DecodeActionEvent(actionDEQueue, queuePayload, 1500, 42, false)
	if !ok {
		t.Fatal("DE_QUEUE did not decode")
	}
	if queue.Type != "de_queue" || queue.PlayerID != 2 || queue.BuildingID != 12 || queue.UnitID != 83 || queue.Amount != 2 || len(queue.ObjectIDs) != 1 || queue.ObjectIDs[0] != 999 {
		t.Fatalf("queue = %+v", queue)
	}

	researchBody := make([]byte, 17)
	binary.LittleEndian.PutUint32(researchBody[0:4], 111)
	binary.LittleEndian.PutUint16(researchBody[4:6], 1)
	binary.LittleEndian.PutUint16(researchBody[6:8], 22)
	binary.LittleEndian.PutUint32(researchBody[13:17], 222)
	researchPayload := append([]byte{3, byte(len(researchBody)), 0}, researchBody...)
	research, ok := DecodeActionEvent(actionResearch, researchPayload, 2000, 50, false)
	if !ok {
		t.Fatal("RESEARCH did not decode")
	}
	if research.Type != "research" || research.PlayerID != 3 || research.TechnologyID != 22 || len(research.ObjectIDs) != 2 || research.ObjectIDs[0] != 111 || research.ObjectIDs[1] != 222 {
		t.Fatalf("research = %+v", research)
	}
}

func TestBuildPlayerProfilesFactsOnly(t *testing.T) {
	events := &EventReport{
		Players: []FeedbackPlayer{{PlayerID: 1, Name: "P1", Kind: "human"}},
		Events: []ReplayEvent{
			{Index: 1, Type: "move", TimeMS: 0, Time: FormatTime(0), PlayerID: 1, PlayerName: "P1", ReplayAction: true, ActionID: actionMove, ActionName: "MOVE", X: 10, Y: 10, Source: "action_stream", Confidence: "parsed"},
			{Index: 2, Type: "action", TimeMS: 1000, Time: FormatTime(1000), PlayerID: 1, PlayerName: "P1", ReplayAction: true, ActionID: 77, Source: "action_stream", Confidence: "raw_preserved"},
			{Index: 3, Type: "chat", TimeMS: 1000, Time: FormatTime(1000), PlayerID: 1, PlayerName: "P1", Text: "ready", Source: "action_stream", Confidence: "parsed"},
			{Index: 4, Type: "chat", TimeMS: 12000, Time: FormatTime(12000), PlayerID: 1, PlayerName: "P1", Text: "go", TauntNumber: 14, Source: "action_stream", Confidence: "parsed"},
		},
	}
	AnnotateFeedbackPhases(events.Events)
	profiles := buildPlayerProfiles(events, 60000, PlayerProfileOptions{WindowSec: 60, DeadGapSec: 10, CellSize: 20, IncludeEvents: true})
	if len(profiles) != 1 {
		t.Fatalf("profiles len = %d, want 1", len(profiles))
	}
	profile := profiles[0]
	if profile.Actions != 2 || profile.DecodedActions != 1 || profile.Undecoded != 1 || profile.APM != 2 {
		t.Fatalf("profile action facts = %+v", profile)
	}
	if profile.Feedback.Chat != 2 || profile.Feedback.Taunts != 1 || profile.Feedback.ChatByPhase["pre_game"] != 1 || profile.Feedback.ChatByPhase["in_game"] != 1 {
		t.Fatalf("feedback = %+v", profile.Feedback)
	}
	if len(profile.Vocabulary) != 2 || profile.Vocabulary[0].Count != 1 || len(profile.Events) != 4 {
		t.Fatalf("vocabulary/events = %+v events=%+v", profile.Vocabulary, profile.Events)
	}
	if len(profile.Claims) == 0 || profile.Claims[len(profile.Claims)-1].Confidence != "phase_heuristic" {
		t.Fatalf("claims = %+v", profile.Claims)
	}
}

func TestFilterPlayerEventsSupportsActionZero(t *testing.T) {
	events := []ReplayEvent{
		{Index: 1, Type: "order", TimeMS: 1000, PlayerID: 1, ReplayAction: true, ActionID: 0},
		{Index: 2, Type: "move", TimeMS: 2000, PlayerID: 1, ReplayAction: true, ActionID: 3},
		{Index: 3, Type: "order", TimeMS: 3000, PlayerID: 2, ReplayAction: true, ActionID: 0},
	}
	filtered := filterPlayerEvents(events, PlayerEventsOptions{PlayerID: 1, ActionID: 0, ActionIDSet: true})
	if len(filtered) != 1 || filtered[0].Index != 1 {
		t.Fatalf("filtered = %+v", filtered)
	}
}

func TestParseClockMS(t *testing.T) {
	tests := map[string]int{
		"12.5":         12500,
		"01:02":        62000,
		"01:02.250":    62250,
		"01:02:03":     3723000,
		"01:02:03.456": 3723456,
	}
	for input, want := range tests {
		got, err := ParseClockMS(input)
		if err != nil {
			t.Fatalf("ParseClockMS(%q): %v", input, err)
		}
		if got != want {
			t.Fatalf("ParseClockMS(%q) = %d, want %d", input, got, want)
		}
	}
}

func TestBuildStoryChaptersAndMoments(t *testing.T) {
	events := []ReplayEvent{
		{Index: 1, Type: "chat", TimeMS: 500, Time: FormatTime(500), PlayerID: 1, Text: "ready", Source: "action_stream", Confidence: "parsed"},
		{Index: 2, Type: "move", TimeMS: 1500, Time: FormatTime(1500), PlayerID: 1, ReplayAction: true, ActionID: actionMove, ActionName: "MOVE", Source: "action_stream", Confidence: "parsed"},
		{Index: 3, Type: "flare", TimeMS: 301000, Time: FormatTime(301000), PlayerID: 2, ReplayAction: true, ActionID: actionFlare, X: 10, Y: 20, Source: "action_stream", Confidence: "parsed"},
		{Index: 4, Type: "resign", TimeMS: 590000, Time: FormatTime(590000), PlayerID: 2, ReplayAction: true, ActionID: actionResign, Source: "action_stream", Confidence: "parsed"},
	}
	chapters := buildStoryChapters(events, 600000)
	if len(chapters) != 4 {
		t.Fatalf("chapters len = %d, want 4: %+v", len(chapters), chapters)
	}
	if chapters[0].Name != "pre_game" || chapters[0].Counts.Chat != 1 {
		t.Fatalf("pre_game chapter = %+v", chapters[0])
	}
	if chapters[2].Name != "midgame" || chapters[2].Counts.Flares != 1 {
		t.Fatalf("midgame chapter = %+v", chapters[2])
	}
	if chapters[3].Name != "endgame" || chapters[3].Counts.Resigns != 1 {
		t.Fatalf("endgame chapter = %+v", chapters[3])
	}
	moments := buildStoryMoments(events, 600000)
	kinds := map[string]bool{}
	for _, moment := range moments {
		kinds[moment.Kind] = true
	}
	for _, want := range []string{"first_chat", "first_player_action", "first_flare", "first_resign", "peak_action_window"} {
		if !kinds[want] {
			t.Fatalf("missing moment %s from %+v", want, moments)
		}
	}
}

func TestAnnotateRegions(t *testing.T) {
	events := []ReplayEvent{{Type: "flare", X: 12.5, Y: 34.5}}
	AnnotateRegions(events, &Context{Regions: []ContextRegion{{Name: "Arena", Box: []float64{10, 30, 20, 40}}}})
	if events[0].Region != "Arena" {
		t.Fatalf("region = %q, want Arena", events[0].Region)
	}
}

func writeU32(buf *bytes.Buffer, value uint32) {
	var raw [4]byte
	binary.LittleEndian.PutUint32(raw[:], value)
	buf.Write(raw[:])
}

func writeReplayMetaFixture(buf *bytes.Buffer) {
	writeU32(buf, 5)
	writeU32(buf, 500)
	writeU32(buf, 1)
	writeU32(buf, 14)
	writeU32(buf, 1)
	writeU32(buf, 2)
	writeU32(buf, 0)
	writeU32(buf, 0)
}
