package replay

import (
	"fmt"
	"sort"
	"strings"
)

const PlaytestReportVersion = "aoe2kit.replay.playtest.v1"

type PlaytestOptions struct {
	// Window is the number of checksum samples before and after a feedback
	// moment to include. Values <= 0 use the default.
	Window           int  `json:"window,omitempty"`
	IncludeComputers bool `json:"include_computers,omitempty"`
}

type PlaytestReport struct {
	Version      string           `json:"version"`
	Path         string           `json:"path,omitempty"`
	Method       string           `json:"method"`
	Verification string           `json:"verification"`
	Session      PlaytestSession  `json:"session"`
	Summary      PlaytestSummary  `json:"summary"`
	Moments      []PlaytestMoment `json:"moments,omitempty"`
	IssueCards   []IssueCard      `json:"issue_cards,omitempty"`
	Honesty      []string         `json:"honesty"`
	Warnings     []string         `json:"warnings,omitempty"`
}

type PlaytestSession struct {
	ScenarioName       string           `json:"scenario_name,omitempty"`
	DurationMS         int              `json:"duration_ms,omitempty"`
	Duration           string           `json:"duration,omitempty"`
	TriggerGraphSHA256 string           `json:"trigger_graph_sha256,omitempty"`
	TriggerGraphOK     bool             `json:"trigger_graph_ok"`
	TriggerGraphMethod string           `json:"trigger_graph_method,omitempty"`
	DataSet            DataSetIdentity  `json:"data_set_identity"`
	HumanCount         int              `json:"human_count"`
	ComputerCount      int              `json:"computer_count"`
	Players            []PlaytestPlayer `json:"players,omitempty"`
}

type PlaytestPlayer struct {
	PlayerID int    `json:"player_id"`
	Label    string `json:"label"`
	Name     string `json:"name,omitempty"`
	Kind     string `json:"kind,omitempty"`
	Human    bool   `json:"human"`
}

type PlaytestSummary struct {
	FeedbackMoments      int `json:"feedback_moments"`
	RawFeedbackRows      int `json:"raw_feedback_rows,omitempty"`
	BacklogRowsExcluded  int `json:"backlog_rows_excluded,omitempty"`
	ChecksumSamples      int `json:"checksum_samples"`
	StateDeltas          int `json:"state_deltas"`
	WindowSamplesEachWay int `json:"window_samples_each_way"`
}

type PlaytestMoment struct {
	Index            int                 `json:"index"`
	RecordedCount    int                 `json:"recorded_count,omitempty"`
	TimeMS           int                 `json:"time_ms"`
	Time             string              `json:"time"`
	LastTimeMS       int                 `json:"last_time_ms,omitempty"`
	LastTime         string              `json:"last_time,omitempty"`
	Type             string              `json:"type"`
	PlayerID         int                 `json:"player_id,omitempty"`
	PlayerName       string              `json:"player_name,omitempty"`
	Text             string              `json:"text,omitempty"`
	TauntNumber      int                 `json:"taunt_number,omitempty"`
	X                float64             `json:"x,omitempty"`
	Y                float64             `json:"y,omitempty"`
	Region           string              `json:"region,omitempty"`
	Source           string              `json:"source"`
	Confidence       string              `json:"confidence"`
	ChangedPlayers   []string            `json:"changed_players,omitempty"`
	UnchangedPlayers []string            `json:"unchanged_players,omitempty"`
	Window           PlaytestStateWindow `json:"state_window"`
}

type PlaytestStateWindow struct {
	SamplesBefore      int                     `json:"samples_before"`
	SamplesAfter       int                     `json:"samples_after"`
	FocusIntervalIndex int                     `json:"focus_interval_index,omitempty"`
	StartTimeMS        int                     `json:"start_time_ms,omitempty"`
	StartTime          string                  `json:"start_time,omitempty"`
	EndTimeMS          int                     `json:"end_time_ms,omitempty"`
	EndTime            string                  `json:"end_time,omitempty"`
	Intervals          []PlaytestStateInterval `json:"intervals,omitempty"`
}

type PlaytestStateInterval struct {
	FromTimeMS int                   `json:"from_time_ms"`
	FromTime   string                `json:"from_time"`
	ToTimeMS   int                   `json:"to_time_ms"`
	ToTime     string                `json:"to_time"`
	Players    []PlaytestStateChange `json:"players"`
}

type PlaytestStateChange struct {
	PlayerID         int    `json:"player_id"`
	PlayerLabel      string `json:"player_label"`
	PlayerName       string `json:"player_name,omitempty"`
	Changed          bool   `json:"changed"`
	Kind             string `json:"kind"`
	ObjectCountDelta int64  `json:"object_count_delta,omitempty"`
	UnitTypeSumDelta int64  `json:"unit_type_sum_delta,omitempty"`
	UnitID           int    `json:"unit_id,omitempty"`
	UnitName         string `json:"unit_name,omitempty"`
	Confidence       string `json:"confidence,omitempty"`
}

func BuildPlaytestReport(path string, opts PlaytestOptions) (*PlaytestReport, error) {
	if opts.Window <= 0 {
		opts.Window = 2
	}
	summary, err := BuildSummary(path)
	if err != nil {
		return nil, err
	}
	rec, err := Open(path)
	if err != nil {
		return nil, err
	}
	eventReport, err := ExtractEvents(path, EventOptions{})
	if err != nil {
		return nil, err
	}
	sync, err := BuildSyncStream(path, SyncOptions{ChecksumsOnly: true})
	if err != nil {
		return nil, err
	}

	players := playtestPlayers(summary.Players)
	statePlayers := playtestStatePlayers(players, opts.IncludeComputers)
	identity := StoryIdentity{Tier: "unknown"}
	if rec.TriggerGraphOK && rec.TriggerGraph != nil {
		identity = StoryIdentity{
			Tier:           "trigger_graph",
			SHA256:         rec.TriggerGraph.SHA256,
			TriggerGraphOK: true,
			Method:         "embedded_trigger_graph",
		}
	}
	cards := BuildIssueCards(path, identity, eventReport.Events, eventReport.Counts.DurationMS)
	report := &PlaytestReport{
		Version:      PlaytestReportVersion,
		Path:         path,
		Method:       "feedback_moments_plus_checksum_state_windows",
		Verification: "replay_visible_feedback_plus_structure_verified_checksum_state; no trigger-name or screen-render claim",
		Session: PlaytestSession{
			ScenarioName:   summary.ScenarioName,
			DurationMS:     summary.DurationMS,
			Duration:       summary.Duration,
			TriggerGraphOK: rec.TriggerGraphOK && rec.TriggerGraph != nil,
			DataSet:        summary.DataSet,
			Players:        players,
		},
		Summary: PlaytestSummary{
			ChecksumSamples:      sync.Summary.ChecksumDE,
			StateDeltas:          len(sync.StateDeltas),
			WindowSamplesEachWay: opts.Window,
		},
		IssueCards: cards,
		Honesty: []string{
			"Replay feedback is visible when it appears in the recorded chat, taunt, or flare stream; backlog chat injected by DE is excluded.",
			"The replay checksum stream stores player state snapshots, not trigger names, UI text, or the author's intended logic.",
			"A state change near a feedback moment is correlation only. It may be the broken mechanic, a working mechanic, or unrelated game state.",
		},
		Warnings: uniquePlaytestWarnings(append([]string{}, summary.Warnings...)),
	}
	if rec.TriggerGraphOK && rec.TriggerGraph != nil {
		report.Session.TriggerGraphSHA256 = rec.TriggerGraph.SHA256
		report.Session.TriggerGraphMethod = "embedded_trigger_graph"
	}
	for _, player := range players {
		if player.Human {
			report.Session.HumanCount++
		} else if player.Kind != "" {
			report.Session.ComputerCount++
		}
	}
	report.Warnings = uniquePlaytestWarnings(append(report.Warnings, eventReport.Warnings...))
	report.Warnings = uniquePlaytestWarnings(append(report.Warnings, sync.Warnings...))

	checksums := checksumEvents(sync.Events)
	backlogExcluded := 0
	for _, event := range eventReport.Events {
		if isBacklogChatEvent(event) {
			backlogExcluded++
			continue
		}
		if !isLiveFeedbackEvent(event) {
			continue
		}
		report.Summary.RawFeedbackRows++
		moment := playtestMomentFromEvent(len(report.Moments)+1, event)
		moment.Window = buildPlaytestStateWindow(event.TimeMS, opts.Window, checksums, sync.StateDeltas, statePlayers)
		moment.ChangedPlayers, moment.UnchangedPlayers = summarizePlaytestSplit(moment.Window, statePlayers)
		report.Moments = appendOrCollapsePlaytestMoment(report.Moments, moment)
	}
	for i := range report.Moments {
		report.Moments[i].Index = i + 1
	}
	report.Summary.FeedbackMoments = len(report.Moments)
	report.Summary.BacklogRowsExcluded = backlogExcluded
	return report, nil
}

func playtestStatePlayers(players []PlaytestPlayer, includeComputers bool) []PlaytestPlayer {
	if includeComputers {
		return players
	}
	var humans []PlaytestPlayer
	for _, player := range players {
		if player.Human {
			humans = append(humans, player)
		}
	}
	if len(humans) == 0 {
		return players
	}
	return humans
}

func playtestPlayers(summaryPlayers []SummaryPlayer) []PlaytestPlayer {
	out := make([]PlaytestPlayer, 0, len(summaryPlayers))
	for _, player := range summaryPlayers {
		if player.PlayerID <= 0 {
			continue
		}
		out = append(out, PlaytestPlayer{
			PlayerID: player.PlayerID,
			Label:    fmt.Sprintf("P%d", player.PlayerID),
			Name:     player.Name,
			Kind:     player.Kind,
			Human:    player.Human,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].PlayerID < out[j].PlayerID })
	return out
}

func checksumEvents(events []SyncEvent) []SyncEvent {
	out := make([]SyncEvent, 0, len(events))
	for _, event := range events {
		if event.Form == "checksum_de" && len(event.Matrix) == 8 {
			out = append(out, event)
		}
	}
	return out
}

func playtestMomentFromEvent(index int, event ReplayEvent) PlaytestMoment {
	text := issueText(event)
	kind := event.Type
	if event.Type == "chat" && event.TauntNumber > 0 {
		kind = "taunt"
	}
	return PlaytestMoment{
		Index:         index,
		RecordedCount: 1,
		TimeMS:        event.TimeMS,
		Time:          event.Time,
		LastTimeMS:    event.TimeMS,
		LastTime:      event.Time,
		Type:          kind,
		PlayerID:      event.PlayerID,
		PlayerName:    event.PlayerName,
		Text:          text,
		TauntNumber:   event.TauntNumber,
		X:             event.X,
		Y:             event.Y,
		Region:        event.Region,
		Source:        event.Source,
		Confidence:    event.Confidence,
	}
}

func appendOrCollapsePlaytestMoment(moments []PlaytestMoment, next PlaytestMoment) []PlaytestMoment {
	const nearDuplicateMS = 3000
	for i := len(moments) - 1; i >= 0; i-- {
		prev := moments[i]
		if next.TimeMS-prev.LastTimeMS > nearDuplicateMS {
			break
		}
		if !samePlaytestFeedback(prev, next) {
			continue
		}
		moments[i].RecordedCount += next.RecordedCount
		moments[i].LastTimeMS = next.LastTimeMS
		moments[i].LastTime = next.LastTime
		return moments
	}
	return append(moments, next)
}

func samePlaytestFeedback(a, b PlaytestMoment) bool {
	if a.Type != b.Type || a.PlayerID != b.PlayerID || normalizeIssueText(a.Text) != normalizeIssueText(b.Text) {
		return false
	}
	if a.Type != "flare" {
		return true
	}
	return int(a.X*10) == int(b.X*10) && int(a.Y*10) == int(b.Y*10)
}

func buildPlaytestStateWindow(momentMS, window int, samples []SyncEvent, deltas []SyncStateDelta, players []PlaytestPlayer) PlaytestStateWindow {
	if len(samples) == 0 || len(players) == 0 {
		return PlaytestStateWindow{}
	}
	anchor := sort.Search(len(samples), func(i int) bool { return samples[i].TimeMS > momentMS }) - 1
	if anchor < 0 {
		anchor = 0
	} else if anchor >= len(samples) {
		anchor = len(samples) - 1
	}
	start := anchor - window
	if start < 0 {
		start = 0
	}
	end := anchor + window
	if end >= len(samples) {
		end = len(samples) - 1
	}
	index := map[string]SyncStateDelta{}
	for _, delta := range deltas {
		key := playtestDeltaKey(delta.FromTimeMS, delta.ToTimeMS, delta.PlayerID)
		index[key] = delta
	}
	stateWindow := PlaytestStateWindow{
		SamplesBefore: anchor - start,
		SamplesAfter:  end - anchor,
		StartTimeMS:   samples[start].TimeMS,
		StartTime:     samples[start].Time,
		EndTimeMS:     samples[end].TimeMS,
		EndTime:       samples[end].Time,
	}
	for i := start; i < end; i++ {
		from, to := samples[i], samples[i+1]
		interval := PlaytestStateInterval{
			FromTimeMS: from.TimeMS,
			FromTime:   from.Time,
			ToTimeMS:   to.TimeMS,
			ToTime:     to.Time,
		}
		for _, player := range players {
			change := PlaytestStateChange{
				PlayerID:    player.PlayerID,
				PlayerLabel: player.Label,
				PlayerName:  player.Name,
				Kind:        "no_change",
			}
			if delta, ok := index[playtestDeltaKey(from.TimeMS, to.TimeMS, player.PlayerID)]; ok {
				change.Kind = delta.Kind
				change.ObjectCountDelta = delta.ObjectCountDelta
				change.UnitTypeSumDelta = delta.UnitTypeSumDelta
				change.UnitID = delta.UnitID
				change.UnitName = delta.UnitName
				change.Confidence = delta.Confidence
				change.Changed = playtestDeltaIsAuthorRelevant(delta)
			}
			interval.Players = append(interval.Players, change)
		}
		stateWindow.Intervals = append(stateWindow.Intervals, interval)
		if i+1 == anchor {
			stateWindow.FocusIntervalIndex = len(stateWindow.Intervals) - 1
		}
	}
	return stateWindow
}

func playtestDeltaIsAuthorRelevant(delta SyncStateDelta) bool {
	return delta.ObjectCountDelta != 0 || delta.UnitTypeSumDelta != 0
}

func summarizePlaytestSplit(window PlaytestStateWindow, players []PlaytestPlayer) ([]string, []string) {
	changed := map[int]bool{}
	if len(window.Intervals) == 0 {
		return nil, nil
	}
	focus := window.FocusIntervalIndex
	if focus < 0 || focus >= len(window.Intervals) {
		focus = 0
	}
	for _, interval := range window.Intervals[focus : focus+1] {
		for _, change := range interval.Players {
			if change.Changed {
				changed[change.PlayerID] = true
			}
		}
	}
	var changedLabels, unchangedLabels []string
	for _, player := range players {
		label := playtestPlayerLabel(player)
		if changed[player.PlayerID] {
			changedLabels = append(changedLabels, label)
		} else {
			unchangedLabels = append(unchangedLabels, label)
		}
	}
	return changedLabels, unchangedLabels
}

func playtestDeltaKey(fromMS, toMS, playerID int) string {
	return fmt.Sprintf("%d:%d:%d", fromMS, toMS, playerID)
}

func playtestPlayerLabel(player PlaytestPlayer) string {
	if player.Name == "" {
		return player.Label
	}
	return strings.TrimSpace(player.Label + " " + player.Name)
}

func uniquePlaytestWarnings(warnings []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, warning := range warnings {
		if warning == "" || seen[warning] {
			continue
		}
		seen[warning] = true
		out = append(out, warning)
	}
	return out
}
