package replay

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
)

type EventOptions struct {
	IncludeSystemEvents  bool
	IncludeUntypedAction bool
	IncludeRaw           bool
	IncludeObjectIndex   bool
	TelemetryPrefixes    []string
	// PriorPaths supplies recordings whose early chat is evidence for DE's
	// carry-over backlog. It is intentionally opt-in.
	PriorPaths []string
}

type EventReport struct {
	Path     string           `json:"path,omitempty"`
	Method   string           `json:"method"`
	DataSet  DataSetIdentity  `json:"data_set_identity"`
	Players  []FeedbackPlayer `json:"players,omitempty"`
	Events   []ReplayEvent    `json:"events"`
	Counts   EventCounts      `json:"counts"`
	Result   MatchResult      `json:"result"`
	Warnings []string         `json:"warnings,omitempty"`
}

type EventCounts struct {
	Total          int            `json:"total"`
	DurationMS     int            `json:"duration_ms,omitempty"`
	Duration       string         `json:"duration,omitempty"`
	Chat           int            `json:"chat"`
	BacklogChat    int            `json:"backlog_chat,omitempty"`
	Taunts         int            `json:"taunts"`
	Flares         int            `json:"flares"`
	Telemetry      int            `json:"telemetry"`
	Syncs          int            `json:"syncs"`
	Starts         int            `json:"starts"`
	Ends           int            `json:"ends"`
	Postgames      int            `json:"postgames"`
	Saves          int            `json:"saves"`
	EmbeddedTails  int            `json:"embedded_tails"`
	Viewlocks      int            `json:"viewlocks"`
	Actions        int            `json:"actions"`
	UntypedActions int            `json:"untyped_actions"`
	SpatialActions int            `json:"spatial_actions"`
	Resigns        int            `json:"resigns"`
	UnknownOps     int            `json:"unknown_ops"`
	ActionIDs      map[int]int    `json:"action_ids,omitempty"`
	EventTypes     map[string]int `json:"event_types,omitempty"`
}

type ReplayEvent struct {
	Index          int               `json:"index"`
	Type           string            `json:"type"`
	TimeMS         int               `json:"time_ms,omitempty"`
	Time           string            `json:"time,omitempty"`
	PlayerID       int               `json:"player_id,omitempty"`
	PlayerName     string            `json:"player_name,omitempty"`
	Channel        int               `json:"channel,omitempty"`
	ChannelName    string            `json:"channel_name,omitempty"`
	Text           string            `json:"text,omitempty"`
	Phase          string            `json:"phase,omitempty"`
	TauntNumber    int               `json:"taunt_number,omitempty"`
	TauntText      string            `json:"taunt_text,omitempty"`
	DestinationMap int               `json:"destination_map,omitempty"`
	MessageAGP     string            `json:"message_agp,omitempty"`
	X              float64           `json:"x,omitempty"`
	Y              float64           `json:"y,omitempty"`
	XEnd           float64           `json:"x_end,omitempty"`
	YEnd           float64           `json:"y_end,omitempty"`
	Region         string            `json:"region,omitempty"`
	Targets        []int             `json:"targets,omitempty"`
	OperationID    int               `json:"operation_id,omitempty"`
	ReplayAction   bool              `json:"replay_action,omitempty"`
	ActionID       int               `json:"action_id,omitempty"`
	ActionName     string            `json:"action_name,omitempty"`
	SourceOffset   int               `json:"source_offset,omitempty"`
	PayloadBytes   int               `json:"payload_bytes,omitempty"`
	RawHex         string            `json:"raw_hex,omitempty"`
	RawText        string            `json:"raw_text,omitempty"`
	Source         string            `json:"source"`
	Confidence     string            `json:"confidence"`
	Telemetry      *TelemetryEvent   `json:"telemetry,omitempty"`
	ObjectIDs      []int             `json:"object_ids,omitempty"`
	TargetID       int               `json:"target_id,omitempty"`
	CommandID      int               `json:"command_id,omitempty"`
	UnitID         int               `json:"unit_id,omitempty"`
	TechnologyID   int               `json:"technology_id,omitempty"`
	Amount         int               `json:"amount,omitempty"`
	BuildingID     int               `json:"building_id,omitempty"`
	TargetObject   *ObjectReference  `json:"target_object,omitempty"`
	BuildingObject *ObjectReference  `json:"building_object,omitempty"`
	ObjectRefs     []ObjectReference `json:"object_refs,omitempty"`
	ModeID         int               `json:"mode_id,omitempty"`
	StanceID       int               `json:"stance_id,omitempty"`
	FormationID    int               `json:"formation_id,omitempty"`
	OrderID        int               `json:"order_id,omitempty"`
	SlotID         int               `json:"slot_id,omitempty"`
	Sequence       int               `json:"sequence,omitempty"`
}

type TelemetryEvent struct {
	Prefix string            `json:"prefix"`
	Name   string            `json:"name,omitempty"`
	Fields map[string]string `json:"fields,omitempty"`
	Raw    string            `json:"raw"`
}

func ExtractEvents(path string, opts EventOptions) (*EventReport, error) {
	cache, err := OpenReplayCache(path)
	if err != nil {
		return nil, err
	}
	rec, err := cache.Metadata()
	if err != nil {
		return nil, err
	}
	events, names, counts, warnings, err := extractCachedEvents(cache, opts)
	if err != nil {
		return nil, err
	}
	if names == nil {
		names = map[int]string{}
	}
	method := "action_stream"
	if backlog := tagInitialChatBacklogEvents(events); backlog > 0 {
		warnings = append(warnings, fmt.Sprintf("tagged %d same-timestamp/repeated-text body chat lines as source=backlog; DE can inject prior session chat into a new recording", backlog))
	}
	if len(opts.PriorPaths) > 0 {
		priorTexts := map[string]bool{}
		for _, priorPath := range opts.PriorPaths {
			prior, err := ExtractEvents(priorPath, EventOptions{})
			if err != nil {
				warnings = append(warnings, fmt.Sprintf("prior replay %q unavailable: %v", priorPath, err))
				continue
			}
			for _, event := range prior.Events {
				if event.Type == "chat" {
					if text := normalizedBacklogText(event.Text); text != "" {
						priorTexts[text] = true
					}
				}
			}
		}
		if tagged := tagPriorRecordBacklogEvents(events, priorTexts); tagged > 0 {
			warnings = append(warnings, fmt.Sprintf("tagged %d early chat lines matching --prior recordings as source=backlog", tagged))
		}
	}
	players := feedbackPlayers(rec.Players, names, eventsToFeedback(events))
	AnnotateFeedbackPhases(events)
	fillEventPlayerNames(events, players)
	if opts.IncludeObjectIndex {
		objectIndex, err := BuildObjectIndexFromEvents(path, events, ObjectIndexOptions{ReferencedOnly: true})
		if err != nil {
			warnings = append(warnings, "object index enrichment unavailable: "+err.Error())
		} else {
			AnnotateEventObjects(events, objectIndex)
		}
	}
	counts = finalizeEventCounts(counts, events)
	result := InferResult(players, events, counts, warnings)
	if method == "body_json_scan" {
		warnings = append(warnings, "flare extraction requires typed replay action-stream parsing; body_json_scan reports chat/taunt JSON only")
	}
	return &EventReport{
		Path:     path,
		Method:   method,
		DataSet:  rec.DataSet,
		Players:  players,
		Events:   events,
		Counts:   counts,
		Result:   result,
		Warnings: warnings,
	}, nil
}

func ExtractActionStreamEvents(body []byte, opts EventOptions) ([]ReplayEvent, map[int]string, EventCounts, []string) {
	reader := bytes.NewReader(body)
	var warnings []string
	if err := readReplayMeta(reader); err != nil {
		return nil, nil, EventCounts{}, []string{"action-stream meta parse failed: " + err.Error()}
	}
	var events []ReplayEvent
	names := map[int]string{}
	seenChat := map[string]bool{}
	counts := EventCounts{ActionIDs: map[int]int{}}
	timeMS := 0
	for {
		opOffset := len(body) - reader.Len()
		op, err := readU32(reader)
		if err != nil {
			if err != io.EOF {
				warnings = append(warnings, "action-stream parse stopped: "+err.Error())
			}
			break
		}
		switch op {
		case 1:
			actionID, payload, sequence, err := readReplayAction(reader)
			if err != nil {
				warnings = append(warnings, "action parse stopped: "+err.Error())
				return finishReplayEvents(events, counts), names, counts, warnings
			}
			counts.Actions++
			counts.ActionIDs[actionID]++
			if event, ok := DecodeActionEvent(actionID, payload, timeMS, opOffset, opts.IncludeRaw); ok {
				event.Sequence = sequence
				switch event.Type {
				case "flare":
					counts.Flares++
					events = append(events, event)
				case "resign":
					counts.Resigns++
					events = append(events, event)
				default:
					if event.X != 0 || event.Y != 0 || event.TargetID != 0 {
						counts.SpatialActions++
					}
					if opts.IncludeUntypedAction {
						events = append(events, event)
					}
				}
				continue
			}
			counts.UntypedActions++
			name := actionName(actionID)
			if name != "" {
				counts.UntypedActions--
			}
			if opts.IncludeUntypedAction {
				event := ReplayEvent{
					Type:         "action",
					TimeMS:       timeMS,
					Time:         FormatTime(timeMS),
					OperationID:  int(op),
					ReplayAction: true,
					ActionID:     actionID,
					ActionName:   name,
					SourceOffset: opOffset,
					PayloadBytes: len(payload),
					Source:       "action_stream",
					Confidence:   "raw_preserved",
					Sequence:     sequence,
				}
				if name != "" {
					event.Confidence = "raw_preserved_named_action"
				}
				if opts.IncludeRaw {
					event.RawHex = hex.EncodeToString(payload)
				}
				events = append(events, event)
			}
		case 2:
			delta, err := readReplaySync(reader)
			if err != nil {
				warnings = append(warnings, "sync parse stopped: "+err.Error())
				return finishReplayEvents(events, counts), names, counts, warnings
			}
			timeMS += int(delta)
			counts.DurationMS = timeMS
			counts.Syncs++
			if opts.IncludeSystemEvents {
				events = append(events, ReplayEvent{
					Type:         "sync",
					TimeMS:       timeMS,
					Time:         FormatTime(timeMS),
					OperationID:  int(op),
					SourceOffset: opOffset,
					Source:       "action_stream",
					Confidence:   "parsed",
				})
			}
		case 3:
			raw := make([]byte, 12)
			if _, err := io.ReadFull(reader, raw); err != nil {
				warnings = append(warnings, "viewlock parse stopped: "+err.Error())
				return finishReplayEvents(events, counts), names, counts, warnings
			}
			counts.Viewlocks++
			if opts.IncludeSystemEvents {
				events = append(events, ReplayEvent{Type: "viewlock", TimeMS: timeMS, Time: FormatTime(timeMS), OperationID: int(op), SourceOffset: opOffset, X: f32(raw[0:4]), Y: f32(raw[4:8]), Source: "action_stream", Confidence: "camera_observed_no_player"})
			}
		case 4:
			if err := skipN(reader, 4); err != nil {
				warnings = append(warnings, "chat prefix parse stopped: "+err.Error())
				return finishReplayEvents(events, counts), names, counts, warnings
			}
			length, err := readU32(reader)
			if err != nil {
				warnings = append(warnings, "chat length parse stopped: "+err.Error())
				return finishReplayEvents(events, counts), names, counts, warnings
			}
			if length > uint32(reader.Len()) {
				warnings = append(warnings, fmt.Sprintf("chat length %d exceeds remaining body %d", length, reader.Len()))
				return finishReplayEvents(events, counts), names, counts, warnings
			}
			raw := make([]byte, int(length))
			if _, err := io.ReadFull(reader, raw); err != nil {
				warnings = append(warnings, "chat payload parse stopped: "+err.Error())
				return finishReplayEvents(events, counts), names, counts, warnings
			}
			feedback, ok, name := decodeActionChat(raw)
			if !ok {
				continue
			}
			key := fmt.Sprintf("%d\x00%d\x00%s\x00%d", feedback.PlayerID, feedback.Channel, feedback.Text, timeMS)
			if seenChat[key] {
				continue
			}
			seenChat[key] = true
			event := eventFromFeedback(feedback, "chat", timeMS, opOffset, "action_stream", "parsed")
			event.Telemetry = ParseTelemetry(event.Text, opts.TelemetryPrefixes)
			if opts.IncludeRaw {
				event.RawHex = hex.EncodeToString(raw)
			}
			if name != "" {
				names[event.PlayerID] = name
			}
			events = append(events, event)
			counts.Chat++
			if event.TauntNumber > 0 {
				counts.Taunts++
			}
			if event.Telemetry != nil {
				counts.Telemetry++
			}
		case 5:
			result, err := skipStart(reader)
			if err != nil {
				warnings = append(warnings, "start op parse stopped: "+err.Error())
				return finishReplayEvents(events, counts), names, counts, warnings
			}
			if result.Terminal {
				counts.EmbeddedTails++
				if opts.IncludeSystemEvents {
					events = append(events, ReplayEvent{Type: "embedded_tail", TimeMS: timeMS, Time: FormatTime(timeMS), OperationID: int(op), SourceOffset: opOffset, PayloadBytes: result.SkippedBytes, Source: "action_stream", Confidence: "raw_preserved"})
				}
				return finishReplayEvents(events, counts), names, counts, warnings
			}
			counts.Starts++
			if opts.IncludeSystemEvents {
				events = append(events, ReplayEvent{Type: "start", TimeMS: timeMS, Time: FormatTime(timeMS), OperationID: int(op), SourceOffset: opOffset, Source: "action_stream", Confidence: "parsed"})
			}
		case 6:
			counts.Ends++
			counts.Postgames++
			if opts.IncludeSystemEvents {
				events = append(events, ReplayEvent{Type: "postgame", TimeMS: timeMS, Time: FormatTime(timeMS), OperationID: int(op), SourceOffset: opOffset, Source: "action_stream", Confidence: "parsed"})
			}
			return finishReplayEvents(events, counts), names, counts, warnings
		default:
			if err := skipReplaySaveChapter(reader, len(body)); err != nil {
				counts.UnknownOps++
				events = append(events, ReplayEvent{
					Type:         "unknown_operation",
					TimeMS:       timeMS,
					Time:         FormatTime(timeMS),
					OperationID:  int(op),
					SourceOffset: opOffset,
					Source:       "action_stream",
					Confidence:   "raw_preserved",
				})
				warnings = append(warnings, fmt.Sprintf("unknown operation id %d at body offset %d", op, opOffset))
				warnings = append(warnings, fmt.Sprintf("save chapter skip failed: %v", err))
				return finishReplayEvents(events, counts), names, counts, warnings
			}
			counts.Saves++
			if opts.IncludeSystemEvents {
				events = append(events, ReplayEvent{Type: "save", TimeMS: timeMS, Time: FormatTime(timeMS), OperationID: int(op), SourceOffset: opOffset, Source: "action_stream", Confidence: "skipped"})
			}
		}
	}
	return finishReplayEvents(events, counts), names, counts, warnings
}

func ExtractJSONScanEvents(body []byte, opts EventOptions) ([]ReplayEvent, map[int]string, []string) {
	feedback, names, warnings := ExtractFeedbackEvents(body)
	events := make([]ReplayEvent, 0, len(feedback))
	for _, event := range feedback {
		replayEvent := eventFromFeedback(event, "chat", event.TimeMS, 0, "body_json_scan", "json_scan")
		replayEvent.Telemetry = ParseTelemetry(replayEvent.Text, opts.TelemetryPrefixes)
		events = append(events, replayEvent)
	}
	return finishReplayEvents(events, EventCounts{}), names, warnings
}

func CountReplayEvents(events []ReplayEvent) EventCounts {
	counts := EventCounts{ActionIDs: map[int]int{}, EventTypes: map[string]int{}}
	for _, event := range events {
		counts.Total++
		counts.EventTypes[event.Type]++
		if event.ReplayAction {
			counts.ActionIDs[event.ActionID]++
		}
		switch event.Type {
		case "chat":
			if isBacklogChatEvent(event) {
				counts.BacklogChat++
			} else {
				counts.Chat++
				if event.TauntNumber > 0 {
					counts.Taunts++
				}
				if event.Telemetry != nil {
					counts.Telemetry++
				}
			}
		case "flare":
			counts.Flares++
		case "sync":
			counts.Syncs++
		case "start":
			counts.Starts++
		case "end":
			counts.Ends++
		case "postgame":
			counts.Postgames++
			counts.Ends++
		case "save":
			counts.Saves++
		case "viewlock":
			counts.Viewlocks++
		case "action":
			counts.Actions++
			counts.UntypedActions++
		case "resign":
			counts.Resigns++
		case "move", "order", "patrol", "attack_move", "attack_ground", "build", "wall", "gather_point", "de_multi_gatherpoint", "special", "ungarrison", "ai_order":
			counts.SpatialActions++
		case "unknown_operation":
			counts.UnknownOps++
		case "embedded_tail":
			counts.EmbeddedTails++
		}
	}
	if len(counts.ActionIDs) == 0 {
		counts.ActionIDs = nil
	}
	if len(counts.EventTypes) == 0 {
		counts.EventTypes = nil
	}
	return counts
}

func hasReplayFeedbackEvents(events []ReplayEvent) bool {
	for _, event := range events {
		if event.Type == "chat" || event.Type == "flare" {
			return true
		}
	}
	return false
}

func finalizeEventCounts(counts EventCounts, events []ReplayEvent) EventCounts {
	visible := CountReplayEvents(events)
	counts.Total = visible.Total
	counts.Chat = visible.Chat
	counts.BacklogChat = visible.BacklogChat
	counts.Taunts = visible.Taunts
	counts.Flares = visible.Flares
	counts.Telemetry = visible.Telemetry
	counts.Resigns = visible.Resigns
	if counts.DurationMS < maxEventTime(events) {
		counts.DurationMS = maxEventTime(events)
	}
	if counts.DurationMS > 0 {
		counts.Duration = FormatTime(counts.DurationMS)
	}
	counts.EventTypes = visible.EventTypes
	return counts
}

func tagInitialChatBacklogEvents(events []ReplayEvent) int {
	const (
		backlogWindowMS       = 120000
		backlogRepeatWindowMS = 60000
		minBacklogBurst       = 5
	)
	bursts := map[int]int{}
	for _, event := range events {
		if event.Type != "chat" || event.TimeMS < 0 || event.TimeMS > backlogWindowMS {
			continue
		}
		bursts[event.TimeMS]++
	}
	tagged := 0
	for i := range events {
		if events[i].Type != "chat" || events[i].TimeMS < 0 || events[i].TimeMS > backlogWindowMS {
			continue
		}
		if bursts[events[i].TimeMS] < minBacklogBurst || shouldPreserveDiagnosticChatMarker(events[i].Text, events[i].TimeMS) {
			continue
		}
		events[i].Source = "backlog"
		events[i].Confidence = appendConfidence(events[i].Confidence, "same_timestamp_backlog_heuristic")
		tagged++
	}
	backlogTexts := map[string]bool{}
	for _, event := range events {
		if event.Type != "chat" || event.Source != "backlog" {
			continue
		}
		if text := normalizedBacklogText(event.Text); text != "" {
			backlogTexts[text] = true
		}
	}
	for i := range events {
		if events[i].Type != "chat" || events[i].Source == "backlog" || events[i].TimeMS < 0 || events[i].TimeMS > backlogRepeatWindowMS {
			continue
		}
		if shouldPreserveDiagnosticChatMarker(events[i].Text, events[i].TimeMS) {
			continue
		}
		if !backlogTexts[normalizedBacklogText(events[i].Text)] {
			continue
		}
		events[i].Source = "backlog"
		events[i].Confidence = appendConfidence(events[i].Confidence, "exact_backlog_text_repeat_heuristic")
		tagged++
	}
	return tagged
}

func tagPriorRecordBacklogEvents(events []ReplayEvent, priorTexts map[string]bool) int {
	const backlogWindowMS = 120000
	tagged := 0
	for i := range events {
		if events[i].Type != "chat" || events[i].Source == "backlog" || events[i].TimeMS < 0 || events[i].TimeMS > backlogWindowMS {
			continue
		}
		if shouldPreserveDiagnosticChatMarker(events[i].Text, events[i].TimeMS) || !priorTexts[normalizedBacklogText(events[i].Text)] {
			continue
		}
		events[i].Source = "backlog"
		events[i].Confidence = appendConfidence(events[i].Confidence, "cross_record_backlog_heuristic")
		tagged++
	}
	return tagged
}

func isBacklogChatEvent(event ReplayEvent) bool {
	return event.Type == "chat" && event.Source == "backlog"
}

func isLiveFeedbackEvent(event ReplayEvent) bool {
	return event.Type == "flare" || event.Type == "chat" && !isBacklogChatEvent(event)
}

func maxEventTime(events []ReplayEvent) int {
	out := 0
	for _, event := range events {
		if event.TimeMS > out {
			out = event.TimeMS
		}
	}
	return out
}

func ParseTelemetry(text string, prefixes []string) *TelemetryEvent {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	if len(prefixes) == 0 {
		prefixes = []string{"SDS EVT", "SDSDBG"}
	}
	for _, prefix := range prefixes {
		prefix = strings.TrimSpace(prefix)
		if prefix == "" {
			continue
		}
		if text == prefix || strings.HasPrefix(text, prefix+" ") {
			rest := strings.TrimSpace(strings.TrimPrefix(text, prefix))
			telemetry := &TelemetryEvent{Prefix: prefix, Raw: text}
			if rest == "" {
				return telemetry
			}
			parts := strings.Fields(rest)
			if len(parts) > 0 && !strings.Contains(parts[0], "=") {
				telemetry.Name = parts[0]
				parts = parts[1:]
			}
			fields := map[string]string{}
			for _, part := range parts {
				key, value, ok := strings.Cut(part, "=")
				if !ok || key == "" {
					continue
				}
				fields[key] = strings.Trim(value, `"`)
			}
			if len(fields) > 0 {
				telemetry.Fields = fields
			}
			return telemetry
		}
	}
	return nil
}

func eventFromFeedback(event FeedbackEvent, eventType string, timeMS int, offset int, source string, confidence string) ReplayEvent {
	out := ReplayEvent{
		Type:           eventType,
		TimeMS:         timeMS,
		Time:           FormatTime(timeMS),
		PlayerID:       event.PlayerID,
		PlayerName:     event.PlayerName,
		Channel:        event.Channel,
		ChannelName:    event.ChannelName,
		Text:           event.Text,
		TauntNumber:    event.TauntNumber,
		TauntText:      event.TauntText,
		DestinationMap: event.DestinationMap,
		MessageAGP:     event.MessageAGP,
		X:              event.X,
		Y:              event.Y,
		Targets:        append([]int(nil), event.Targets...),
		SourceOffset:   offset,
		Source:         source,
		Confidence:     confidence,
		RawText:        event.Raw,
	}
	if eventType == "flare" {
		out.ActionID = 115
		out.OperationID = 1
	}
	if out.TauntNumber > 0 && out.TauntText == "" {
		out.TauntText = TauntText(out.TauntNumber)
	}
	return out
}

func eventsToFeedback(events []ReplayEvent) []FeedbackEvent {
	out := make([]FeedbackEvent, 0, len(events))
	for _, event := range events {
		switch event.Type {
		case "chat", "flare":
			if !isLiveFeedbackEvent(event) {
				continue
			}
			out = append(out, FeedbackEvent{
				Index:          event.Index,
				Kind:           strings.ToUpper(event.Type),
				TimeMS:         event.TimeMS,
				Time:           event.Time,
				Phase:          event.Phase,
				PlayerID:       event.PlayerID,
				PlayerName:     event.PlayerName,
				Channel:        event.Channel,
				ChannelName:    event.ChannelName,
				Text:           event.Text,
				Raw:            event.RawText,
				TauntNumber:    event.TauntNumber,
				TauntText:      event.TauntText,
				DestinationMap: event.DestinationMap,
				MessageAGP:     event.MessageAGP,
				X:              event.X,
				Y:              event.Y,
				Targets:        append([]int(nil), event.Targets...),
			})
		}
	}
	return out
}

func fillEventPlayerNames(events []ReplayEvent, players []FeedbackPlayer) {
	nameByID := map[int]string{}
	for _, player := range players {
		nameByID[player.PlayerID] = player.Name
	}
	for i := range events {
		if name := nameByID[events[i].PlayerID]; name != "" {
			events[i].PlayerName = name
		}
	}
}

func finishReplayEvents(events []ReplayEvent, counts EventCounts) []ReplayEvent {
	for i := range events {
		events[i].Index = i + 1
		if events[i].Time == "" && events[i].TimeMS > 0 {
			events[i].Time = FormatTime(events[i].TimeMS)
		}
		if events[i].Confidence == "" {
			events[i].Confidence = "parsed"
		}
		if events[i].Source == "" {
			events[i].Source = "unknown"
		}
	}
	_ = counts
	return events
}

func sortedEventTypeCounts(counts map[string]int) []string {
	keys := make([]string, 0, len(counts))
	for key := range counts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]string, 0, len(keys))
	for _, key := range keys {
		out = append(out, key+"="+strconv.Itoa(counts[key]))
	}
	return out
}
