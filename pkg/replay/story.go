package replay

import (
	"fmt"
	"sort"
	"strings"
)

type StoryOptions struct {
	ContextPath string
}

type StoryReport struct {
	Path             string               `json:"path,omitempty"`
	ContextName      string               `json:"context_name,omitempty"`
	Identity         StoryIdentity        `json:"identity"`
	DataSet          DataSetIdentity      `json:"data_set_identity"`
	Players          []FeedbackPlayer     `json:"players,omitempty"`
	PlayerSummary    []PlayerStorySummary `json:"player_summary,omitempty"`
	Brief            []string             `json:"brief"`
	Timeline         []StoryLine          `json:"timeline,omitempty"`
	Feedback         []StoryLine          `json:"feedback,omitempty"`
	Telemetry        []StoryLine          `json:"telemetry,omitempty"`
	Chapters         []StoryChapter       `json:"chapters,omitempty"`
	Moments          []StoryMoment        `json:"moments,omitempty"`
	Regions          []RegionSummary      `json:"regions,omitempty"`
	FeedbackClusters []FeedbackCluster    `json:"feedback_clusters,omitempty"`
	EventCounts      EventCounts          `json:"event_counts"`
	Result           MatchResult          `json:"result"`
	Verification     string               `json:"verification"`
	Claims           []StoryClaim         `json:"claims,omitempty"`
	Missing          []string             `json:"missing,omitempty"`
	Warnings         []string             `json:"warnings,omitempty"`
}

type StoryIdentity struct {
	Tier           string `json:"tier,omitempty"`
	SHA256         string `json:"sha256,omitempty"`
	TriggerGraphOK bool   `json:"trigger_graph_ok"`
	Method         string `json:"method,omitempty"`
}

type StoryLine struct {
	Time       string `json:"time,omitempty"`
	TimeMS     int    `json:"time_ms,omitempty"`
	PlayerID   int    `json:"player_id,omitempty"`
	PlayerName string `json:"player_name,omitempty"`
	Region     string `json:"region,omitempty"`
	Type       string `json:"type"`
	Text       string `json:"text"`
}

type StoryChapter struct {
	Name       string        `json:"name"`
	StartMS    int           `json:"start_ms"`
	EndMS      int           `json:"end_ms"`
	Start      string        `json:"start"`
	End        string        `json:"end"`
	Counts     ChapterCounts `json:"counts"`
	Highlights []StoryLine   `json:"highlights,omitempty"`
}

type ChapterCounts struct {
	Events    int `json:"events"`
	Actions   int `json:"actions"`
	Chat      int `json:"chat"`
	Taunts    int `json:"taunts"`
	Flares    int `json:"flares"`
	Telemetry int `json:"telemetry"`
	Resigns   int `json:"resigns"`
	Viewlocks int `json:"viewlocks"`
}

type StoryMoment struct {
	Kind       string `json:"kind"`
	Time       string `json:"time,omitempty"`
	TimeMS     int    `json:"time_ms,omitempty"`
	PlayerID   int    `json:"player_id,omitempty"`
	PlayerName string `json:"player_name,omitempty"`
	Detail     string `json:"detail"`
	Source     string `json:"source"`
	Confidence string `json:"confidence"`
}

type StoryClaim struct {
	Name       string `json:"name"`
	Status     string `json:"status"`
	Source     string `json:"source"`
	Confidence string `json:"confidence"`
	Detail     string `json:"detail,omitempty"`
}

type PlayerStorySummary struct {
	PlayerID   int    `json:"player_id"`
	PlayerName string `json:"player_name,omitempty"`
	Kind       string `json:"kind,omitempty"`
	FirstTime  string `json:"first_time,omitempty"`
	LastTime   string `json:"last_time,omitempty"`
	Chat       int    `json:"chat"`
	Taunts     int    `json:"taunts"`
	Flares     int    `json:"flares"`
	Telemetry  int    `json:"telemetry"`
}

type RegionSummary struct {
	Name   string `json:"name"`
	Events int    `json:"events"`
	Flares int    `json:"flares"`
}

type FeedbackCluster struct {
	Key        string `json:"key"`
	Region     string `json:"region,omitempty"`
	PlayerID   int    `json:"player_id,omitempty"`
	PlayerName string `json:"player_name,omitempty"`
	Count      int    `json:"count"`
	FirstTime  string `json:"first_time,omitempty"`
	LastTime   string `json:"last_time,omitempty"`
}

func BuildStory(path string, opts StoryOptions) (*StoryReport, error) {
	context, err := LoadContext(opts.ContextPath)
	if err != nil {
		return nil, err
	}
	eventOpts := EventOptions{IncludeSystemEvents: true, IncludeUntypedAction: true, TelemetryPrefixes: context.TelemetryPrefixes}
	events, err := ExtractEvents(path, eventOpts)
	if err != nil {
		return nil, err
	}
	AnnotateRegions(events.Events, context)
	counts := events.Counts
	rec, err := Open(path)
	if err != nil {
		return nil, err
	}
	report := &StoryReport{
		Path:             path,
		ContextName:      context.Name,
		Identity:         storyIdentity(rec),
		DataSet:          rec.DataSet,
		Players:          events.Players,
		PlayerSummary:    summarizePlayers(events.Events, events.Players),
		Timeline:         storyTimeline(events.Events),
		Feedback:         feedbackStoryLines(events.Events),
		Telemetry:        telemetryStoryLines(events.Events),
		Chapters:         buildStoryChapters(events.Events, counts.DurationMS),
		Moments:          buildStoryMoments(events.Events, counts.DurationMS),
		Regions:          summarizeRegions(events.Events),
		FeedbackClusters: summarizeFeedbackClusters(events.Events),
		EventCounts:      counts,
		Result:           events.Result,
		Verification:     "structure_verified_not_engine_verified",
		Claims:           storyClaims(events, counts, rec),
		Missing:          missingStoryCapabilities(counts),
		Warnings:         append([]string{}, events.Warnings...),
	}
	report.Brief = storyBrief(report)
	return report, nil
}

func storyIdentity(rec *File) StoryIdentity {
	if rec.TriggerGraphOK && rec.TriggerGraph != nil {
		return StoryIdentity{
			Tier:           "trigger_graph",
			SHA256:         rec.TriggerGraph.SHA256,
			TriggerGraphOK: true,
			Method:         "replay embedded trigger graph",
		}
	}
	if rec.FallbackOK && rec.Fallback != nil {
		return StoryIdentity{
			Tier:           rec.Fallback.Tier,
			SHA256:         rec.Fallback.SHA256,
			TriggerGraphOK: false,
			Method:         rec.Fallback.Method,
		}
	}
	return StoryIdentity{}
}

func storyBrief(report *StoryReport) []string {
	var lines []string
	if report.Identity.SHA256 != "" {
		lines = append(lines, fmt.Sprintf("Replay identity: %s %s.", report.Identity.Tier, shortStoryHash(report.Identity.SHA256)))
	} else {
		lines = append(lines, "Replay identity could not be established by current AoE2Kit parsers.")
	}
	if len(report.Players) > 0 {
		lines = append(lines, fmt.Sprintf("Players observed: %s.", summarizeStoryPlayers(report.Players)))
	}
	lines = append(lines, summarizeDataSetIdentity(report.DataSet))
	chatSummary := fmt.Sprintf("%d chat", report.EventCounts.Chat)
	if report.EventCounts.BacklogChat > 0 {
		chatSummary += fmt.Sprintf(" (%d backlog)", report.EventCounts.BacklogChat)
	}
	lines = append(lines, fmt.Sprintf("Parsed replay-visible events: %s, %d flare, %d telemetry marker.", chatSummary, report.EventCounts.Flares, report.EventCounts.Telemetry))
	if report.EventCounts.Duration != "" {
		lines = append(lines, "Parsed action-stream duration: "+report.EventCounts.Duration+".")
	}
	if report.Result.WinnerKnown {
		lines = append(lines, fmt.Sprintf("Result by %s: winners=%v losers=%v.", report.Result.Method, report.Result.Winners, report.Result.Losers))
	} else if report.Result.Completed {
		lines = append(lines, "Replay completion was observed, but winners are not decoded yet.")
	} else {
		lines = append(lines, "Replay result is unknown to current AoE2Kit parsers.")
	}
	if len(report.Regions) > 0 {
		lines = append(lines, "Most marked regions: "+summarizeTopRegions(report.Regions)+".")
	}
	if len(report.Feedback) > 0 {
		lines = append(lines, fmt.Sprintf("Feedback surface contains %d replay-visible note(s).", len(report.Feedback)))
	}
	if len(report.Telemetry) > 0 {
		lines = append(lines, fmt.Sprintf("Telemetry surface contains %d replay-visible marker(s).", len(report.Telemetry)))
	}
	if len(report.FeedbackClusters) > 0 {
		lines = append(lines, "Largest feedback cluster: "+summarizeTopFeedbackCluster(report.FeedbackClusters)+".")
	}
	if report.EventCounts.UntypedActions > 0 {
		lines = append(lines, fmt.Sprintf("Low-level parser counted %d untyped action command(s); these are preserved as counts/raw events but not gameplay semantics yet.", report.EventCounts.UntypedActions))
	}
	lines = append(lines, "Verification: structural replay parsing only; no in-engine replay playback oracle was run.")
	return lines
}

func storyTimeline(events []ReplayEvent) []StoryLine {
	lines := make([]StoryLine, 0, len(events))
	for _, event := range events {
		switch event.Type {
		case "chat", "flare", "unknown_operation":
			if event.Type == "chat" && isBacklogChatEvent(event) {
				continue
			}
			lines = append(lines, storyLineForEvent(event))
		}
	}
	return lines
}

func feedbackStoryLines(events []ReplayEvent) []StoryLine {
	var lines []StoryLine
	for _, event := range events {
		if isLiveFeedbackEvent(event) {
			lines = append(lines, storyLineForEvent(event))
		}
	}
	return lines
}

func telemetryStoryLines(events []ReplayEvent) []StoryLine {
	var lines []StoryLine
	for _, event := range events {
		if event.Telemetry != nil {
			lines = append(lines, storyLineForEvent(event))
		}
	}
	return lines
}

func buildStoryChapters(events []ReplayEvent, durationMS int) []StoryChapter {
	if durationMS <= 0 {
		durationMS = maxEventTime(events)
	}
	if durationMS <= 0 {
		return nil
	}
	bounds := storyChapterBounds(durationMS)
	chapters := make([]StoryChapter, 0, len(bounds))
	for _, bound := range bounds {
		chapter := StoryChapter{Name: bound.name, StartMS: bound.start, EndMS: bound.end, Start: FormatTime(bound.start), End: FormatTime(bound.end)}
		for _, event := range events {
			if event.TimeMS < bound.start || event.TimeMS > bound.end {
				continue
			}
			chapter.Counts.Events++
			if event.ReplayAction {
				chapter.Counts.Actions++
			}
			switch event.Type {
			case "chat":
				if isBacklogChatEvent(event) {
					continue
				}
				chapter.Counts.Chat++
				if event.TauntNumber > 0 {
					chapter.Counts.Taunts++
				}
				if event.Telemetry != nil {
					chapter.Counts.Telemetry++
				}
				if len(chapter.Highlights) < 5 {
					chapter.Highlights = append(chapter.Highlights, storyLineForEvent(event))
				}
			case "flare":
				chapter.Counts.Flares++
				if len(chapter.Highlights) < 5 {
					chapter.Highlights = append(chapter.Highlights, storyLineForEvent(event))
				}
			case "resign":
				chapter.Counts.Resigns++
				if len(chapter.Highlights) < 5 {
					chapter.Highlights = append(chapter.Highlights, storyLineForEvent(event))
				}
			case "viewlock":
				chapter.Counts.Viewlocks++
			}
		}
		chapters = append(chapters, chapter)
	}
	return chapters
}

type storyChapterBound struct {
	name  string
	start int
	end   int
}

func storyChapterBounds(durationMS int) []storyChapterBound {
	preEnd := minInt(durationMS, 1000)
	openEnd := minInt(durationMS, 300000)
	endStart := durationMS - 120000
	if endStart < openEnd {
		endStart = openEnd
	}
	var bounds []storyChapterBound
	bounds = append(bounds, storyChapterBound{name: "pre_game", start: 0, end: preEnd})
	if durationMS > preEnd {
		bounds = append(bounds, storyChapterBound{name: "opening", start: preEnd + 1, end: openEnd})
	}
	if durationMS > openEnd && endStart > openEnd {
		bounds = append(bounds, storyChapterBound{name: "midgame", start: openEnd + 1, end: endStart - 1})
	}
	if durationMS > openEnd {
		bounds = append(bounds, storyChapterBound{name: "endgame", start: endStart, end: durationMS})
	}
	return bounds
}

func chapterNameForTime(timeMS int, durationMS int) string {
	if timeMS <= 1000 {
		return "pre_game"
	}
	if timeMS <= 300000 {
		return "opening"
	}
	if durationMS > 0 && timeMS >= durationMS-120000 {
		return "endgame"
	}
	return "midgame"
}

func buildStoryMoments(events []ReplayEvent, durationMS int) []StoryMoment {
	var moments []StoryMoment
	if first, ok := firstEvent(events, func(event ReplayEvent) bool { return event.ReplayAction && event.PlayerID > 0 }); ok {
		moments = append(moments, momentForEvent("first_player_action", first, describeActionMoment(first), "replay action stream", "replay_action_verified"))
	}
	if first, ok := firstEvent(events, func(event ReplayEvent) bool { return event.Type == "chat" && !isBacklogChatEvent(event) }); ok {
		moments = append(moments, momentForEvent("first_chat", first, describeChatMoment(first), "replay chat", "replay_visible"))
	}
	if first, ok := firstEvent(events, func(event ReplayEvent) bool { return event.Type == "flare" }); ok {
		moments = append(moments, momentForEvent("first_flare", first, describeFlareMoment(first), "replay action stream", "replay_action_verified"))
	}
	if first, ok := firstEvent(events, func(event ReplayEvent) bool { return event.Type == "resign" }); ok {
		moments = append(moments, momentForEvent("first_resign", first, fmt.Sprintf("P%d resigned", first.PlayerID), "replay action stream", "replay_action_verified"))
	}
	if peak, ok := peakActionWindow(events, durationMS, 60000); ok {
		moments = append(moments, peak)
	}
	sort.Slice(moments, func(i, j int) bool {
		if moments[i].TimeMS != moments[j].TimeMS {
			return moments[i].TimeMS < moments[j].TimeMS
		}
		return moments[i].Kind < moments[j].Kind
	})
	return moments
}

func firstEvent(events []ReplayEvent, match func(ReplayEvent) bool) (ReplayEvent, bool) {
	var first ReplayEvent
	found := false
	for _, event := range events {
		if !match(event) {
			continue
		}
		if !found || event.TimeMS < first.TimeMS || (event.TimeMS == first.TimeMS && event.Index < first.Index) {
			first = event
			found = true
		}
	}
	return first, found
}

func momentForEvent(kind string, event ReplayEvent, detail string, source string, confidence string) StoryMoment {
	return StoryMoment{
		Kind:       kind,
		Time:       event.Time,
		TimeMS:     event.TimeMS,
		PlayerID:   event.PlayerID,
		PlayerName: event.PlayerName,
		Detail:     detail,
		Source:     source,
		Confidence: confidence,
	}
}

func describeActionMoment(event ReplayEvent) string {
	name := event.ActionName
	if name == "" {
		name = fmt.Sprintf("ACTION_%d", event.ActionID)
	}
	if event.Type != "" && event.Type != "action" {
		return fmt.Sprintf("P%d first replay-visible action: %s/%s", event.PlayerID, name, event.Type)
	}
	return fmt.Sprintf("P%d first replay-visible action: %s", event.PlayerID, name)
}

func describeChatMoment(event ReplayEvent) string {
	if event.TauntNumber > 0 {
		return fmt.Sprintf("P%d first chat: taunt %d", event.PlayerID, event.TauntNumber)
	}
	if event.Text == "" {
		return fmt.Sprintf("P%d first chat", event.PlayerID)
	}
	return fmt.Sprintf("P%d first chat: %q", event.PlayerID, event.Text)
}

func describeFlareMoment(event ReplayEvent) string {
	if event.Region != "" {
		return fmt.Sprintf("P%d first flare at x=%.2f y=%.2f in %s", event.PlayerID, event.X, event.Y, event.Region)
	}
	return fmt.Sprintf("P%d first flare at x=%.2f y=%.2f", event.PlayerID, event.X, event.Y)
}

func peakActionWindow(events []ReplayEvent, durationMS int, windowMS int) (StoryMoment, bool) {
	if durationMS <= 0 {
		durationMS = maxEventTime(events)
	}
	if durationMS <= 0 || windowMS <= 0 {
		return StoryMoment{}, false
	}
	windows := durationMS/windowMS + 1
	counts := make([]int, windows)
	for _, event := range events {
		if !event.ReplayAction {
			continue
		}
		idx := event.TimeMS / windowMS
		if idx >= len(counts) {
			idx = len(counts) - 1
		}
		counts[idx]++
	}
	peakIdx := 0
	for i, count := range counts {
		if count > counts[peakIdx] {
			peakIdx = i
		}
	}
	if counts[peakIdx] == 0 {
		return StoryMoment{}, false
	}
	start := peakIdx * windowMS
	end := minInt(durationMS, start+windowMS)
	return StoryMoment{
		Kind:       "peak_action_window",
		Time:       FormatTime(start),
		TimeMS:     start,
		Detail:     fmt.Sprintf("%d replay-visible actions from %s to %s", counts[peakIdx], FormatTime(start), FormatTime(end)),
		Source:     "replay action stream",
		Confidence: "replay_action_verified",
	}, true
}

func storyLineForEvent(event ReplayEvent) StoryLine {
	text := event.Text
	switch event.Type {
	case "flare":
		text = fmt.Sprintf("flared x=%.2f y=%.2f", event.X, event.Y)
		if event.Region != "" {
			text += " in " + event.Region
		}
	case "chat":
		if event.Telemetry != nil {
			text = "telemetry " + event.Telemetry.Raw
		} else if event.TauntNumber > 0 {
			text = fmt.Sprintf("%q [taunt %d]", event.Text, event.TauntNumber)
		} else {
			text = fmt.Sprintf("%q", event.Text)
		}
	case "unknown_operation":
		text = fmt.Sprintf("unknown replay operation %d at offset %d", event.OperationID, event.SourceOffset)
	case "resign":
		text = "resigned"
	}
	return StoryLine{
		Time:       event.Time,
		TimeMS:     event.TimeMS,
		PlayerID:   event.PlayerID,
		PlayerName: event.PlayerName,
		Region:     event.Region,
		Type:       event.Type,
		Text:       text,
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func summarizeRegions(events []ReplayEvent) []RegionSummary {
	byName := map[string]*RegionSummary{}
	for _, event := range events {
		if event.Region == "" {
			continue
		}
		item := byName[event.Region]
		if item == nil {
			item = &RegionSummary{Name: event.Region}
			byName[event.Region] = item
		}
		item.Events++
		if event.Type == "flare" {
			item.Flares++
		}
	}
	names := make([]string, 0, len(byName))
	for name := range byName {
		names = append(names, name)
	}
	sort.Slice(names, func(i, j int) bool {
		a, b := byName[names[i]], byName[names[j]]
		if a.Events != b.Events {
			return a.Events > b.Events
		}
		return a.Name < b.Name
	})
	out := make([]RegionSummary, 0, len(names))
	for _, name := range names {
		out = append(out, *byName[name])
	}
	return out
}

func summarizePlayers(events []ReplayEvent, players []FeedbackPlayer) []PlayerStorySummary {
	byID := map[int]*PlayerStorySummary{}
	for _, player := range players {
		if player.PlayerID <= 0 {
			continue
		}
		byID[player.PlayerID] = &PlayerStorySummary{PlayerID: player.PlayerID, PlayerName: player.Name, Kind: player.Kind}
	}
	for _, event := range events {
		if event.PlayerID <= 0 {
			continue
		}
		item := byID[event.PlayerID]
		if item == nil {
			item = &PlayerStorySummary{PlayerID: event.PlayerID, PlayerName: event.PlayerName}
			byID[event.PlayerID] = item
		}
		if item.PlayerName == "" {
			item.PlayerName = event.PlayerName
		}
		if item.FirstTime == "" || event.TimeMS < parseStoryTime(item.FirstTime) {
			item.FirstTime = event.Time
		}
		if item.LastTime == "" || event.TimeMS >= parseStoryTime(item.LastTime) {
			item.LastTime = event.Time
		}
		switch event.Type {
		case "chat":
			if isBacklogChatEvent(event) {
				continue
			}
			item.Chat++
			if event.TauntNumber > 0 {
				item.Taunts++
			}
			if event.Telemetry != nil {
				item.Telemetry++
			}
		case "flare":
			item.Flares++
		}
	}
	ids := make([]int, 0, len(byID))
	for id := range byID {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	out := make([]PlayerStorySummary, 0, len(ids))
	for _, id := range ids {
		out = append(out, *byID[id])
	}
	return out
}

func summarizeFeedbackClusters(events []ReplayEvent) []FeedbackCluster {
	byKey := map[string]*FeedbackCluster{}
	for _, event := range events {
		if !isLiveFeedbackEvent(event) {
			continue
		}
		key := ""
		if event.Region != "" {
			key = "region:" + event.Region
		} else if event.PlayerID > 0 {
			key = fmt.Sprintf("player:%d", event.PlayerID)
		} else {
			key = "unknown"
		}
		item := byKey[key]
		if item == nil {
			item = &FeedbackCluster{Key: key, Region: event.Region, PlayerID: event.PlayerID, PlayerName: event.PlayerName}
			byKey[key] = item
		}
		item.Count++
		if item.FirstTime == "" || event.TimeMS < parseStoryTime(item.FirstTime) {
			item.FirstTime = event.Time
		}
		if item.LastTime == "" || event.TimeMS >= parseStoryTime(item.LastTime) {
			item.LastTime = event.Time
		}
	}
	keys := make([]string, 0, len(byKey))
	for key := range byKey {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		a, b := byKey[keys[i]], byKey[keys[j]]
		if a.Count != b.Count {
			return a.Count > b.Count
		}
		return a.Key < b.Key
	})
	out := make([]FeedbackCluster, 0, len(keys))
	for _, key := range keys {
		out = append(out, *byKey[key])
	}
	return out
}

func missingStoryCapabilities(counts EventCounts) []string {
	missing := []string{
		"unit lifecycle and combat are not decoded yet",
		"trigger firings are not replay-native facts; use replay-visible telemetry markers for trigger-level truth",
	}
	if counts.Resigns == 0 && counts.Postgames == 0 {
		missing = append(missing, "outcome is not decoded for this replay because no replay-visible result signal was reached")
	}
	if counts.UntypedActions > 0 {
		missing = append(missing, "some low-level action commands are counted but not semantically named yet")
	}
	return missing
}

func storyClaims(events *EventReport, counts EventCounts, rec *File) []StoryClaim {
	var claims []StoryClaim
	if rec.TriggerGraphOK && rec.TriggerGraph != nil {
		claims = append(claims, StoryClaim{
			Name:       "scenario_identity",
			Status:     "known",
			Source:     "embedded_trigger_graph",
			Confidence: "parsed",
			Detail:     rec.TriggerGraph.SHA256,
		})
	} else if rec.FallbackOK && rec.Fallback != nil {
		claims = append(claims, StoryClaim{
			Name:       "scenario_identity",
			Status:     "fallback",
			Source:     rec.Fallback.Method,
			Confidence: "parsed",
			Detail:     rec.Fallback.SHA256,
		})
	} else {
		claims = append(claims, StoryClaim{Name: "scenario_identity", Status: "unknown", Source: "header", Confidence: "not_available"})
	}
	playerStatus := "unknown"
	if len(events.Players) > 0 {
		playerStatus = "known"
	}
	claims = append(claims, StoryClaim{
		Name:       "players",
		Status:     playerStatus,
		Source:     "replay_header_and_chat_names",
		Confidence: "parsed",
		Detail:     fmt.Sprintf("%d player slot(s)", len(events.Players)),
	})
	dataSetStatus := rec.DataSet.Status
	if dataSetStatus == "" {
		dataSetStatus = "unknown"
	}
	dataSetDetail := "vanilla/base data set"
	if rec.DataSet.Status == "modded" {
		dataSetDetail = strings.Join(rec.DataSet.ActiveDataSets, ", ")
	} else if rec.DataSet.Status == "unknown" && rec.DataSet.Error != "" {
		dataSetDetail = rec.DataSet.Error
	}
	claims = append(claims, StoryClaim{
		Name:       "data_set_identity",
		Status:     dataSetStatus,
		Source:     rec.DataSet.Source,
		Confidence: rec.DataSet.Confidence,
		Detail:     dataSetDetail,
	})
	claims = append(claims, StoryClaim{
		Name:       "feedback",
		Status:     "known",
		Source:     events.Method,
		Confidence: "replay_visible",
		Detail:     fmt.Sprintf("%d chat, %d flare", counts.Chat, counts.Flares),
	})
	telemetryStatus := "absent"
	if counts.Telemetry > 0 {
		telemetryStatus = "known"
	}
	claims = append(claims, StoryClaim{
		Name:       "telemetry",
		Status:     telemetryStatus,
		Source:     "chat_markers",
		Confidence: "scenario_declared",
		Detail:     fmt.Sprintf("%d marker(s)", counts.Telemetry),
	})
	resultStatus := "unknown"
	resultConfidence := "not_available"
	resultDetail := "no decoded result signal"
	if events.Result.WinnerKnown {
		resultStatus = "known"
		resultConfidence = "parsed"
		resultDetail = fmt.Sprintf("winners=%v losers=%v", events.Result.Winners, events.Result.Losers)
	} else if events.Result.Completed {
		resultStatus = "partial"
		resultConfidence = "parsed"
		resultDetail = "completion seen, winner not decoded"
	}
	claims = append(claims, StoryClaim{
		Name:       "result",
		Status:     resultStatus,
		Source:     events.Result.Method,
		Confidence: resultConfidence,
		Detail:     resultDetail,
	})
	claims = append(claims, StoryClaim{
		Name:       "combat_and_unit_lifecycle",
		Status:     "unknown",
		Source:     "not_decoded",
		Confidence: "not_claimed",
		Detail:     "AoE2Kit currently does not infer kills, deaths, ownership changes, or trigger firings from replay state",
	})
	return claims
}

func summarizeDataSetIdentity(identity DataSetIdentity) string {
	switch identity.Status {
	case "vanilla":
		return "Active data set: vanilla/base data set by embedded mod block."
	case "modded":
		detail := strings.Join(identity.ActiveDataSets, ", ")
		if detail == "" {
			detail = identity.ActiveDataSet
		}
		if identity.Checksum != 0 {
			detail += fmt.Sprintf(" checksum=%d", identity.Checksum)
		}
		if identity.WorkshopID != 0 {
			detail += fmt.Sprintf(" workshop_id=%d", identity.WorkshopID)
		}
		return "Active data set: modded " + detail + "."
	default:
		return "Active data set could not be established from the embedded mod block."
	}
}

func summarizeStoryPlayers(players []FeedbackPlayer) string {
	parts := make([]string, 0, len(players))
	for _, player := range players {
		name := player.Name
		if name == "" {
			name = "Unknown"
		}
		parts = append(parts, fmt.Sprintf("P%d %s (%s)", player.PlayerID, name, player.Kind))
	}
	return strings.Join(parts, ", ")
}

func summarizeTopRegions(regions []RegionSummary) string {
	limit := len(regions)
	if limit > 3 {
		limit = 3
	}
	parts := make([]string, 0, limit)
	for i := 0; i < limit; i++ {
		parts = append(parts, fmt.Sprintf("%s=%d", regions[i].Name, regions[i].Events))
	}
	return strings.Join(parts, ", ")
}

func summarizeTopFeedbackCluster(clusters []FeedbackCluster) string {
	if len(clusters) == 0 {
		return ""
	}
	cluster := clusters[0]
	label := cluster.Key
	if cluster.Region != "" {
		label = cluster.Region
	} else if cluster.PlayerID > 0 {
		name := cluster.PlayerName
		if name == "" {
			name = "Unknown"
		}
		label = fmt.Sprintf("P%d %s", cluster.PlayerID, name)
	}
	return fmt.Sprintf("%s=%d", label, cluster.Count)
}

func shortStoryHash(hash string) string {
	if len(hash) <= 12 {
		return hash
	}
	return hash[:12]
}

func parseStoryTime(s string) int {
	var h, m, sec, ms int
	if _, err := fmt.Sscanf(s, "%02d:%02d:%02d.%03d", &h, &m, &sec, &ms); err != nil {
		return 0
	}
	return ((h*60+m)*60+sec)*1000 + ms
}
