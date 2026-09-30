package replay

import (
	"fmt"
	"math"
	"sort"
)

type PlayerProfileOptions struct {
	ContextPath   string
	WindowSec     int
	DeadGapSec    int
	CellSize      float64
	IncludeEvents bool
}

type PlayerProfileReport struct {
	Path         string                  `json:"path,omitempty"`
	Method       string                  `json:"method"`
	ContextName  string                  `json:"context_name,omitempty"`
	Identity     StoryIdentity           `json:"identity"`
	DataSet      DataSetIdentity         `json:"data_set_identity"`
	Players      []PlayerProfile         `json:"players"`
	Camera       []CameraObservation     `json:"camera_observed,omitempty"`
	EventCounts  EventCounts             `json:"event_counts"`
	Coverage     ProfileCoverage         `json:"coverage"`
	Result       MatchResult             `json:"result"`
	Verification string                  `json:"verification"`
	Claims       []StoryClaim            `json:"claims,omitempty"`
	Missing      []string                `json:"missing,omitempty"`
	Warnings     []string                `json:"warnings,omitempty"`
	Options      PlayerProfileRunOptions `json:"options"`
}

type PlayerProfileRunOptions struct {
	WindowSec     int     `json:"window_sec"`
	DeadGapSec    int     `json:"dead_gap_sec"`
	CellSize      float64 `json:"cell_size"`
	IncludeEvents bool    `json:"include_events"`
}

type PlayerProfile struct {
	PlayerID       int                  `json:"player_id"`
	PlayerName     string               `json:"player_name,omitempty"`
	Kind           string               `json:"kind,omitempty"`
	Actions        int                  `json:"actions"`
	DecodedActions int                  `json:"decoded_actions"`
	Undecoded      int                  `json:"undecoded_actions"`
	APM            float64              `json:"apm"`
	APMWindows     []APMWindow          `json:"apm_windows,omitempty"`
	DeadGaps       []DeadGap            `json:"dead_gaps,omitempty"`
	Vocabulary     []CommandVocabulary  `json:"vocabulary,omitempty"`
	Spatial        SpatialProfile       `json:"spatial"`
	Feedback       ProfileFeedbackCount `json:"feedback"`
	Notables       []ProfileNotable     `json:"notables,omitempty"`
	Claims         []StoryClaim         `json:"claims,omitempty"`
	Events         []ProfileEventRow    `json:"events,omitempty"`
}

type ProfileCoverage struct {
	ReplayActions      int     `json:"replay_actions"`
	DecodedActions     int     `json:"decoded_actions"`
	UndecodedActions   int     `json:"undecoded_actions"`
	DecodedPercent     float64 `json:"decoded_percent"`
	CoordinateActions  int     `json:"coordinate_actions"`
	ViewlockEvents     int     `json:"viewlock_events"`
	FeedbackEvents     int     `json:"feedback_events"`
	PhaseTagged        int     `json:"phase_tagged_feedback"`
	PhaseHeuristicNote string  `json:"phase_heuristic_note,omitempty"`
}

type APMWindow struct {
	StartMS int     `json:"start_ms"`
	EndMS   int     `json:"end_ms"`
	Start   string  `json:"start"`
	End     string  `json:"end"`
	Actions int     `json:"actions"`
	APM     float64 `json:"apm"`
}

type DeadGap struct {
	StartMS    int    `json:"start_ms"`
	EndMS      int    `json:"end_ms"`
	Start      string `json:"start"`
	End        string `json:"end"`
	DurationMS int    `json:"duration_ms"`
	Duration   string `json:"duration"`
}

type CommandVocabulary struct {
	Key          string  `json:"key"`
	ActionID     int     `json:"action_id,omitempty"`
	ActionName   string  `json:"action_name,omitempty"`
	Type         string  `json:"type,omitempty"`
	Count        int     `json:"count"`
	SharePercent float64 `json:"share_percent"`
	Decoded      bool    `json:"decoded"`
}

type SpatialProfile struct {
	Commands           int             `json:"commands"`
	CellsVisited       int             `json:"cells_visited"`
	CellSize           float64         `json:"cell_size"`
	CentroidSwitches   int             `json:"centroid_switches"`
	TopCells           []SpatialCell   `json:"top_cells,omitempty"`
	TopRegions         []RegionSummary `json:"top_regions,omitempty"`
	CoordinateCoverage float64         `json:"coordinate_coverage_percent,omitempty"`
}

type SpatialCell struct {
	Cell  string  `json:"cell"`
	MinX  float64 `json:"min_x"`
	MinY  float64 `json:"min_y"`
	Count int     `json:"count"`
}

type ProfileFeedbackCount struct {
	Chat         int            `json:"chat"`
	Taunts       int            `json:"taunts"`
	Flares       int            `json:"flares"`
	Telemetry    int            `json:"telemetry"`
	ByPhase      map[string]int `json:"by_phase,omitempty"`
	ChatByPhase  map[string]int `json:"chat_by_phase,omitempty"`
	TauntByPhase map[string]int `json:"taunt_by_phase,omitempty"`
}

type ProfileNotable struct {
	Name   string `json:"name"`
	Detail string `json:"detail"`
}

type CameraObservation struct {
	TimeMS int     `json:"time_ms"`
	Time   string  `json:"time"`
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Region string  `json:"region,omitempty"`
}

type ProfileEventRow struct {
	Index       int     `json:"index"`
	Type        string  `json:"type"`
	TimeMS      int     `json:"time_ms"`
	Time        string  `json:"time"`
	Phase       string  `json:"phase,omitempty"`
	PlayerID    int     `json:"player_id,omitempty"`
	PlayerName  string  `json:"player_name,omitempty"`
	ActionID    int     `json:"action_id,omitempty"`
	ActionName  string  `json:"action_name,omitempty"`
	X           float64 `json:"x,omitempty"`
	Y           float64 `json:"y,omitempty"`
	Region      string  `json:"region,omitempty"`
	Text        string  `json:"text,omitempty"`
	TauntNumber int     `json:"taunt_number,omitempty"`
	Confidence  string  `json:"confidence"`
}

func BuildPlayerProfile(path string, opts PlayerProfileOptions) (*PlayerProfileReport, error) {
	opts = normalizeProfileOptions(opts)
	context, err := LoadContext(opts.ContextPath)
	if err != nil {
		return nil, err
	}
	eventOpts := EventOptions{
		IncludeSystemEvents:  true,
		IncludeUntypedAction: true,
		TelemetryPrefixes:    context.TelemetryPrefixes,
	}
	events, err := ExtractEvents(path, eventOpts)
	if err != nil {
		return nil, err
	}
	AnnotateRegions(events.Events, context)
	AnnotateFeedbackPhases(events.Events)
	rec, err := Open(path)
	if err != nil {
		return nil, err
	}
	durationMS := events.Counts.DurationMS
	if durationMS == 0 {
		durationMS = maxEventTime(events.Events)
	}
	report := &PlayerProfileReport{
		Path:         path,
		Method:       events.Method,
		ContextName:  context.Name,
		Identity:     storyIdentity(rec),
		DataSet:      rec.DataSet,
		EventCounts:  events.Counts,
		Result:       events.Result,
		Verification: "structure_verified_not_engine_verified",
		Options: PlayerProfileRunOptions{
			WindowSec:     opts.WindowSec,
			DeadGapSec:    opts.DeadGapSec,
			CellSize:      opts.CellSize,
			IncludeEvents: opts.IncludeEvents,
		},
		Warnings: append([]string{}, events.Warnings...),
	}
	report.Players = buildPlayerProfiles(events, durationMS, opts)
	report.Camera = cameraObservations(events.Events)
	report.Coverage = profileCoverage(events.Events)
	report.Claims = profileClaims(report)
	report.Missing = profileMissing(report)
	return report, nil
}

func normalizeProfileOptions(opts PlayerProfileOptions) PlayerProfileOptions {
	if opts.WindowSec <= 0 {
		opts.WindowSec = 60
	}
	if opts.DeadGapSec <= 0 {
		opts.DeadGapSec = 10
	}
	if opts.CellSize <= 0 {
		opts.CellSize = 20
	}
	return opts
}

func AnnotateFeedbackPhases(events []ReplayEvent) {
	for i := range events {
		if events[i].Type != "chat" {
			continue
		}
		if events[i].TimeMS <= 1000 {
			events[i].Phase = "pre_game"
		} else {
			events[i].Phase = "in_game"
		}
	}
}

func buildPlayerProfiles(events *EventReport, durationMS int, opts PlayerProfileOptions) []PlayerProfile {
	players := map[int]*PlayerProfile{}
	for _, player := range events.Players {
		if player.PlayerID <= 0 {
			continue
		}
		players[player.PlayerID] = &PlayerProfile{
			PlayerID:   player.PlayerID,
			PlayerName: player.Name,
			Kind:       player.Kind,
			Feedback: ProfileFeedbackCount{
				ByPhase:      map[string]int{},
				ChatByPhase:  map[string]int{},
				TauntByPhase: map[string]int{},
			},
		}
	}
	actionTimes := map[int][]int{}
	vocab := map[int]map[string]*CommandVocabulary{}
	cellCounts := map[int]map[string]*SpatialCell{}
	regionCounts := map[int]map[string]*RegionSummary{}
	prevCell := map[int]string{}
	for _, event := range events.Events {
		playerID := event.PlayerID
		if playerID <= 0 && event.Type != "viewlock" {
			continue
		}
		if playerID > 0 && players[playerID] == nil {
			players[playerID] = &PlayerProfile{
				PlayerID: playerID,
				Feedback: ProfileFeedbackCount{
					ByPhase:      map[string]int{},
					ChatByPhase:  map[string]int{},
					TauntByPhase: map[string]int{},
				},
			}
		}
		if event.ReplayAction && playerID > 0 {
			profile := players[playerID]
			profile.Actions++
			actionTimes[playerID] = append(actionTimes[playerID], event.TimeMS)
			if event.Type == "action" {
				profile.Undecoded++
			} else {
				profile.DecodedActions++
			}
			if event.Type == "flare" {
				profile.Feedback.Flares++
			}
			key := profileVocabularyKey(event)
			if vocab[playerID] == nil {
				vocab[playerID] = map[string]*CommandVocabulary{}
			}
			item := vocab[playerID][key]
			if item == nil {
				item = &CommandVocabulary{
					Key:        key,
					ActionID:   event.ActionID,
					ActionName: event.ActionName,
					Type:       event.Type,
					Decoded:    event.Type != "action",
				}
				vocab[playerID][key] = item
			}
			item.Count++
			if hasCoordinates(event) {
				profile.Spatial.Commands++
				profile.Spatial.CellSize = opts.CellSize
				cellKey, minX, minY := spatialCell(event.X, event.Y, opts.CellSize)
				if cellCounts[playerID] == nil {
					cellCounts[playerID] = map[string]*SpatialCell{}
				}
				cell := cellCounts[playerID][cellKey]
				if cell == nil {
					cell = &SpatialCell{Cell: cellKey, MinX: minX, MinY: minY}
					cellCounts[playerID][cellKey] = cell
				}
				cell.Count++
				if prevCell[playerID] != "" && prevCell[playerID] != cellKey {
					profile.Spatial.CentroidSwitches++
				}
				prevCell[playerID] = cellKey
				if event.Region != "" {
					if regionCounts[playerID] == nil {
						regionCounts[playerID] = map[string]*RegionSummary{}
					}
					region := regionCounts[playerID][event.Region]
					if region == nil {
						region = &RegionSummary{Name: event.Region}
						regionCounts[playerID][event.Region] = region
					}
					region.Events++
					if event.Type == "flare" {
						region.Flares++
					}
				}
			}
			if opts.IncludeEvents {
				profile.Events = append(profile.Events, profileEventRow(event))
			}
			continue
		}
		if playerID > 0 && isLiveFeedbackEvent(event) {
			profile := players[playerID]
			if event.Type == "chat" {
				profile.Feedback.Chat++
				if event.Telemetry != nil {
					profile.Feedback.Telemetry++
				}
				phase := event.Phase
				if phase == "" {
					phase = "unknown"
				}
				profile.Feedback.ByPhase[phase]++
				profile.Feedback.ChatByPhase[phase]++
				if event.TauntNumber > 0 {
					profile.Feedback.Taunts++
					profile.Feedback.TauntByPhase[phase]++
				}
			} else {
				profile.Feedback.Flares++
			}
			if opts.IncludeEvents {
				profile.Events = append(profile.Events, profileEventRow(event))
			}
		}
	}
	out := make([]PlayerProfile, 0, len(players))
	for _, profile := range players {
		profile.APM = round2(apm(profile.Actions, durationMS))
		profile.APMWindows = buildAPMWindows(actionTimes[profile.PlayerID], durationMS, opts.WindowSec)
		profile.DeadGaps = buildDeadGaps(actionTimes[profile.PlayerID], opts.DeadGapSec)
		profile.Vocabulary = sortedVocabulary(vocab[profile.PlayerID], profile.Actions, 10)
		profile.Spatial.CellsVisited = len(cellCounts[profile.PlayerID])
		profile.Spatial.TopCells = sortedSpatialCells(cellCounts[profile.PlayerID], 8)
		profile.Spatial.TopRegions = sortedProfileRegions(regionCounts[profile.PlayerID], 8)
		if profile.Actions > 0 {
			profile.Spatial.CoordinateCoverage = round2(float64(profile.Spatial.Commands) * 100 / float64(profile.Actions))
		}
		profile.Notables = profileNotables(profile)
		profile.Claims = playerProfileClaims(profile)
		out = append(out, *profile)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].PlayerID < out[j].PlayerID })
	return out
}

func profileVocabularyKey(event ReplayEvent) string {
	name := event.ActionName
	if name == "" {
		name = fmt.Sprintf("ACTION_%d", event.ActionID)
	}
	if event.Type == "action" {
		return name + ":raw"
	}
	return name + ":" + event.Type
}

func hasCoordinates(event ReplayEvent) bool {
	return event.X != 0 || event.Y != 0
}

func spatialCell(x, y, size float64) (string, float64, float64) {
	cx := math.Floor(x / size)
	cy := math.Floor(y / size)
	minX := cx * size
	minY := cy * size
	return fmt.Sprintf("%.0f,%.0f", cx, cy), minX, minY
}

func apm(actions int, durationMS int) float64 {
	if actions == 0 || durationMS <= 0 {
		return 0
	}
	return float64(actions) / (float64(durationMS) / 60000)
}

func buildAPMWindows(times []int, durationMS int, windowSec int) []APMWindow {
	if durationMS <= 0 || windowSec <= 0 {
		return nil
	}
	windowMS := windowSec * 1000
	windows := int(math.Ceil(float64(durationMS) / float64(windowMS)))
	if windows <= 0 {
		windows = 1
	}
	counts := make([]int, windows)
	for _, timeMS := range times {
		idx := timeMS / windowMS
		if idx >= windows {
			idx = windows - 1
		}
		if idx >= 0 {
			counts[idx]++
		}
	}
	out := make([]APMWindow, 0, windows)
	for i, count := range counts {
		start := i * windowMS
		end := start + windowMS
		if end > durationMS {
			end = durationMS
		}
		minutes := float64(end-start) / 60000
		windowAPM := 0.0
		if minutes > 0 {
			windowAPM = float64(count) / minutes
		}
		out = append(out, APMWindow{StartMS: start, EndMS: end, Start: FormatTime(start), End: FormatTime(end), Actions: count, APM: round2(windowAPM)})
	}
	return out
}

func buildDeadGaps(times []int, deadGapSec int) []DeadGap {
	if len(times) < 2 || deadGapSec <= 0 {
		return nil
	}
	sort.Ints(times)
	threshold := deadGapSec * 1000
	var out []DeadGap
	for i := 1; i < len(times); i++ {
		delta := times[i] - times[i-1]
		if delta < threshold {
			continue
		}
		out = append(out, DeadGap{
			StartMS:    times[i-1],
			EndMS:      times[i],
			Start:      FormatTime(times[i-1]),
			End:        FormatTime(times[i]),
			DurationMS: delta,
			Duration:   FormatTime(delta),
		})
	}
	return out
}

func sortedVocabulary(values map[string]*CommandVocabulary, actions int, limit int) []CommandVocabulary {
	out := make([]CommandVocabulary, 0, len(values))
	for _, item := range values {
		copied := *item
		if actions > 0 {
			copied.SharePercent = round2(float64(copied.Count) * 100 / float64(actions))
		}
		out = append(out, copied)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].Key < out[j].Key
	})
	if limit > 0 && len(out) > limit {
		return out[:limit]
	}
	return out
}

func sortedSpatialCells(values map[string]*SpatialCell, limit int) []SpatialCell {
	out := make([]SpatialCell, 0, len(values))
	for _, item := range values {
		out = append(out, *item)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].Cell < out[j].Cell
	})
	if limit > 0 && len(out) > limit {
		return out[:limit]
	}
	return out
}

func sortedProfileRegions(values map[string]*RegionSummary, limit int) []RegionSummary {
	out := make([]RegionSummary, 0, len(values))
	for _, item := range values {
		out = append(out, *item)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Events != out[j].Events {
			return out[i].Events > out[j].Events
		}
		return out[i].Name < out[j].Name
	})
	if limit > 0 && len(out) > limit {
		return out[:limit]
	}
	return out
}

func profileNotables(profile *PlayerProfile) []ProfileNotable {
	var out []ProfileNotable
	if len(profile.Vocabulary) > 0 && profile.Vocabulary[0].SharePercent >= 50 {
		out = append(out, ProfileNotable{Name: "dominant_command", Detail: fmt.Sprintf("%s accounts for %.2f%% of replay actions", profile.Vocabulary[0].Key, profile.Vocabulary[0].SharePercent)})
	}
	if len(profile.DeadGaps) > 0 {
		longest := profile.DeadGaps[0]
		for _, gap := range profile.DeadGaps[1:] {
			if gap.DurationMS > longest.DurationMS {
				longest = gap
			}
		}
		out = append(out, ProfileNotable{Name: "longest_dead_gap", Detail: fmt.Sprintf("%s from %s to %s", longest.Duration, longest.Start, longest.End)})
	}
	if len(profile.APMWindows) > 0 {
		peak := profile.APMWindows[0]
		for _, window := range profile.APMWindows[1:] {
			if window.APM > peak.APM {
				peak = window
			}
		}
		out = append(out, ProfileNotable{Name: "peak_apm_window", Detail: fmt.Sprintf("%.2f APM from %s to %s", peak.APM, peak.Start, peak.End)})
	}
	if profile.Spatial.Commands > 0 {
		out = append(out, ProfileNotable{Name: "spatial_footprint", Detail: fmt.Sprintf("%d coordinate commands across %d grid cells", profile.Spatial.Commands, profile.Spatial.CellsVisited)})
	}
	return out
}

func playerProfileClaims(profile *PlayerProfile) []StoryClaim {
	return []StoryClaim{
		{Name: "action_count", Status: "available", Source: "replay action stream", Confidence: "replay_action_verified", Detail: fmt.Sprintf("%d actions for player %d", profile.Actions, profile.PlayerID)},
		{Name: "apm_windows", Status: "available", Source: "replay action stream", Confidence: "replay_action_verified", Detail: "derived from replay-visible action timestamps"},
		{Name: "dead_gaps", Status: "available", Source: "replay action stream", Confidence: "replay_action_verified", Detail: "gaps between replay-visible actions; causes are not inferred"},
		{Name: "command_vocabulary", Status: "partial", Source: "replay action decoder", Confidence: "replay_action_verified", Detail: fmt.Sprintf("%d decoded, %d raw-preserved undecoded", profile.DecodedActions, profile.Undecoded)},
		{Name: "spatial_footprint", Status: "partial", Source: "decoded coordinate actions", Confidence: "replay_action_verified", Detail: fmt.Sprintf("%d of %d actions carry decoded coordinates", profile.Spatial.Commands, profile.Actions)},
		{Name: "chat_phase_counts", Status: "heuristic", Source: "replay chat timestamps", Confidence: "phase_heuristic", Detail: "chat at or before 1.000s is tagged pre_game; later chat is tagged in_game"},
	}
}

func profileClaims(report *PlayerProfileReport) []StoryClaim {
	claims := []StoryClaim{
		{Name: "player_event_table", Status: "available", Source: "replay action stream", Confidence: "replay_action_verified", Detail: "facts are grouped by player before behavioral summaries are derived"},
		{Name: "profile_narration", Status: "not_claimed", Source: "none", Confidence: "not_claimed", Detail: "v1 reports facts only; temperament, skill, attention, and intent are not inferred"},
		{Name: "combat_lifecycle", Status: "not_claimed", Source: "none", Confidence: "not_claimed", Detail: "current replay parser does not reconstruct combat outcomes or unit lifecycle"},
		{Name: "viewlock", Status: "available", Source: "operation 3 action stream", Confidence: "camera_observed_no_player", Detail: "camera positions are global observations and are not attributed to a player"},
	}
	if report.Coverage.UndecodedActions > 0 {
		claims = append(claims, StoryClaim{Name: "unknown_actions", Status: "partial", Source: "replay action stream", Confidence: "raw_preserved", Detail: fmt.Sprintf("%d replay actions remain raw-preserved only", report.Coverage.UndecodedActions)})
	}
	return claims
}

func profileMissing(report *PlayerProfileReport) []string {
	missing := []string{
		"combat/unit lifecycle reconstruction",
		"economy/resource state reconstruction",
		"per-player camera attribution for viewlock operations",
		"multi-game baselines for skill or temperament claims",
	}
	if report.Coverage.UndecodedActions > 0 {
		missing = append(missing, "more typed replay action decoders")
	}
	return missing
}

func profileCoverage(events []ReplayEvent) ProfileCoverage {
	var coverage ProfileCoverage
	for _, event := range events {
		switch event.Type {
		case "chat", "flare":
			if !isLiveFeedbackEvent(event) {
				continue
			}
			coverage.FeedbackEvents++
			if event.Type == "chat" && event.Phase != "" {
				coverage.PhaseTagged++
			}
		case "viewlock":
			coverage.ViewlockEvents++
		}
		if event.ReplayAction {
			coverage.ReplayActions++
			if event.Type == "action" {
				coverage.UndecodedActions++
			} else {
				coverage.DecodedActions++
			}
			if hasCoordinates(event) {
				coverage.CoordinateActions++
			}
		}
	}
	if coverage.ReplayActions > 0 {
		coverage.DecodedPercent = round2(float64(coverage.DecodedActions) * 100 / float64(coverage.ReplayActions))
	}
	coverage.PhaseHeuristicNote = "chat at or before 1.000s is tagged pre_game; later chat is tagged in_game"
	return coverage
}

func cameraObservations(events []ReplayEvent) []CameraObservation {
	var out []CameraObservation
	for _, event := range events {
		if event.Type != "viewlock" {
			continue
		}
		out = append(out, CameraObservation{TimeMS: event.TimeMS, Time: event.Time, X: event.X, Y: event.Y, Region: event.Region})
	}
	return out
}

func profileEventRow(event ReplayEvent) ProfileEventRow {
	return ProfileEventRow{
		Index:       event.Index,
		Type:        event.Type,
		TimeMS:      event.TimeMS,
		Time:        event.Time,
		Phase:       event.Phase,
		PlayerID:    event.PlayerID,
		PlayerName:  event.PlayerName,
		ActionID:    event.ActionID,
		ActionName:  event.ActionName,
		X:           event.X,
		Y:           event.Y,
		Region:      event.Region,
		Text:        event.Text,
		TauntNumber: event.TauntNumber,
		Confidence:  event.Confidence,
	}
}

func round2(value float64) float64 {
	return math.Round(value*100) / 100
}
