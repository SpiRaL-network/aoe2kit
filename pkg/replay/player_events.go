package replay

import (
	"fmt"
	"sort"
	"strings"
)

type PlayerEventsOptions struct {
	ContextPath    string
	PlayerID       int
	Type           string
	ActionID       int
	ActionIDSet    bool
	Phase          string
	FromMS         int
	ToMS           int
	Limit          int
	IncludeRaw     bool
	IncludeObjects bool
}

type PlayerEventsReport struct {
	Path         string              `json:"path,omitempty"`
	Method       string              `json:"method"`
	ContextName  string              `json:"context_name,omitempty"`
	Players      []FeedbackPlayer    `json:"players,omitempty"`
	Counts       EventCounts         `json:"counts"`
	Filters      PlayerEventsFilters `json:"filters"`
	Shown        int                 `json:"shown"`
	Events       []ReplayEvent       `json:"events"`
	Verification string              `json:"verification"`
	Claims       []StoryClaim        `json:"claims,omitempty"`
	Warnings     []string            `json:"warnings,omitempty"`
}

type PlayerEventsFilters struct {
	PlayerID int    `json:"player_id,omitempty"`
	Type     string `json:"type,omitempty"`
	ActionID *int   `json:"action_id,omitempty"`
	Phase    string `json:"phase,omitempty"`
	FromMS   int    `json:"from_ms,omitempty"`
	ToMS     int    `json:"to_ms,omitempty"`
	Limit    int    `json:"limit,omitempty"`
}

func BuildPlayerEvents(path string, opts PlayerEventsOptions) (*PlayerEventsReport, error) {
	context, err := LoadContext(opts.ContextPath)
	if err != nil {
		return nil, err
	}
	eventOpts := EventOptions{
		IncludeSystemEvents:  true,
		IncludeUntypedAction: true,
		IncludeRaw:           opts.IncludeRaw,
		IncludeObjectIndex:   opts.IncludeObjects,
		TelemetryPrefixes:    context.TelemetryPrefixes,
	}
	eventReport, err := ExtractEvents(path, eventOpts)
	if err != nil {
		return nil, err
	}
	AnnotateRegions(eventReport.Events, context)
	filtered := filterPlayerEvents(eventReport.Events, opts)
	report := &PlayerEventsReport{
		Path:        path,
		Method:      eventReport.Method,
		ContextName: context.Name,
		Players:     eventReport.Players,
		Counts:      eventReport.Counts,
		Filters: PlayerEventsFilters{
			PlayerID: opts.PlayerID,
			Type:     opts.Type,
			ActionID: filterActionIDPointer(opts),
			Phase:    opts.Phase,
			FromMS:   opts.FromMS,
			ToMS:     opts.ToMS,
			Limit:    opts.Limit,
		},
		Shown:        len(filtered),
		Events:       filtered,
		Verification: "structure_verified_not_engine_verified",
		Claims:       playerEventsClaims(eventReport.Counts, filtered),
		Warnings:     append([]string{}, eventReport.Warnings...),
	}
	return report, nil
}

func filterPlayerEvents(events []ReplayEvent, opts PlayerEventsOptions) []ReplayEvent {
	var out []ReplayEvent
	for _, event := range events {
		if opts.PlayerID > 0 && event.PlayerID != opts.PlayerID {
			continue
		}
		if opts.Type != "" && event.Type != opts.Type {
			continue
		}
		if opts.ActionIDSet && (!event.ReplayAction || event.ActionID != opts.ActionID) {
			continue
		}
		if opts.Phase != "" && event.Phase != opts.Phase {
			continue
		}
		if opts.FromMS > 0 && event.TimeMS < opts.FromMS {
			continue
		}
		if opts.ToMS > 0 && event.TimeMS > opts.ToMS {
			continue
		}
		out = append(out, event)
		if opts.Limit > 0 && len(out) >= opts.Limit {
			break
		}
	}
	return out
}

func filterActionIDPointer(opts PlayerEventsOptions) *int {
	if !opts.ActionIDSet {
		return nil
	}
	value := opts.ActionID
	return &value
}

func playerEventsClaims(counts EventCounts, events []ReplayEvent) []StoryClaim {
	return []StoryClaim{
		{Name: "event_table", Status: "available", Source: "replay action stream", Confidence: "replay_action_verified", Detail: fmt.Sprintf("%d filtered rows from %d visible rows", len(events), counts.Total)},
		{Name: "raw_actions", Status: rawActionStatus(counts), Source: "replay action stream", Confidence: "raw_preserved", Detail: fmt.Sprintf("%d untyped action command(s)", counts.UntypedActions)},
		{Name: "behavior_interpretation", Status: "not_claimed", Source: "none", Confidence: "not_claimed", Detail: "player-events is an event table; it does not infer skill, intent, attention, or psychology"},
	}
}

func rawActionStatus(counts EventCounts) string {
	if counts.UntypedActions > 0 {
		return "partial"
	}
	return "available"
}

func ParseClockMS(value string) (int, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, nil
	}
	var h, m, s, ms int
	if strings.Count(value, ":") == 2 {
		if _, err := fmt.Sscanf(value, "%d:%d:%d.%d", &h, &m, &s, &ms); err == nil {
			return ((h*60+m)*60+s)*1000 + ms, nil
		}
		if _, err := fmt.Sscanf(value, "%d:%d:%d", &h, &m, &s); err == nil {
			return ((h*60+m)*60 + s) * 1000, nil
		}
	}
	if strings.Count(value, ":") == 1 {
		if _, err := fmt.Sscanf(value, "%d:%d.%d", &m, &s, &ms); err == nil {
			return (m*60+s)*1000 + ms, nil
		}
		if _, err := fmt.Sscanf(value, "%d:%d", &m, &s); err == nil {
			return (m*60 + s) * 1000, nil
		}
	}
	var seconds float64
	if _, err := fmt.Sscanf(value, "%f", &seconds); err == nil {
		return int(seconds * 1000), nil
	}
	return 0, fmt.Errorf("invalid time %q; use seconds, MM:SS, or HH:MM:SS.mmm", value)
}

func SortReplayEvents(events []ReplayEvent) {
	sort.Slice(events, func(i, j int) bool {
		if events[i].TimeMS != events[j].TimeMS {
			return events[i].TimeMS < events[j].TimeMS
		}
		return events[i].Index < events[j].Index
	})
}
