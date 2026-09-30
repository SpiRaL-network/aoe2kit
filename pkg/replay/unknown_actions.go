package replay

import (
	"sort"
)

// UnknownActions aggregates untyped (no ActionName) replay actions so the next
// opcode decoder can be chosen from corpus evidence instead of guessing.
// Honesty: everything here is raw_preserved structural observation; no
// semantics are claimed for any unknown id.

type UnknownActionsReport struct {
	Path         string              `json:"path,omitempty"`
	Method       string              `json:"method"`
	Verification string              `json:"verification"`
	TotalActions int                 `json:"total_actions"`
	UnknownTotal int                 `json:"unknown_actions_total"`
	Groups       []UnknownActionInfo `json:"groups,omitempty"`
	Warnings     []string            `json:"warnings,omitempty"`
}

type UnknownActionInfo struct {
	ActionID      int          `json:"action_id"`
	Count         int          `json:"count"`
	Players       []int        `json:"players_seen,omitempty"`
	FirstTimeMS   int          `json:"first_time_ms"`
	FirstTime     string       `json:"first_time"`
	LastTimeMS    int          `json:"last_time_ms"`
	LastTime      string       `json:"last_time"`
	PayloadShapes []DeltaShape `json:"payload_length_shapes,omitempty"`
	SampleHex     []string     `json:"sample_hex,omitempty"`
}

func BuildUnknownActions(path string, samples int) (*UnknownActionsReport, error) {
	report, err := ExtractEvents(path, EventOptions{IncludeUntypedAction: true, IncludeRaw: true})
	if err != nil {
		return nil, err
	}
	out := &UnknownActionsReport{
		Path:         path,
		Method:       "action_stream_untyped_aggregation",
		Verification: "raw_preserved_no_semantics_claimed",
	}
	type agg struct {
		info    UnknownActionInfo
		players map[int]bool
		lengths map[int]int
	}
	groups := map[int]*agg{}
	for _, event := range report.Events {
		if !event.ReplayAction {
			continue
		}
		out.TotalActions++
		if event.ActionName != "" || actionName(event.ActionID) != "" {
			continue
		}
		out.UnknownTotal++
		g, ok := groups[event.ActionID]
		if !ok {
			g = &agg{
				info: UnknownActionInfo{
					ActionID:    event.ActionID,
					FirstTimeMS: event.TimeMS,
					FirstTime:   event.Time,
				},
				players: map[int]bool{},
				lengths: map[int]int{},
			}
			groups[event.ActionID] = g
		}
		g.info.Count++
		g.info.LastTimeMS = event.TimeMS
		g.info.LastTime = event.Time
		if event.PlayerID != 0 {
			g.players[event.PlayerID] = true
		}
		g.lengths[event.PayloadBytes]++
		if samples > 0 && len(g.info.SampleHex) < samples && event.RawHex != "" {
			g.info.SampleHex = append(g.info.SampleHex, event.RawHex)
		}
	}
	for _, g := range groups {
		for p := range g.players {
			g.info.Players = append(g.info.Players, p)
		}
		sort.Ints(g.info.Players)
		for length, count := range g.lengths {
			g.info.PayloadShapes = append(g.info.PayloadShapes, DeltaShape{DeltaMS: length, Count: count})
		}
		sort.Slice(g.info.PayloadShapes, func(i, j int) bool { return g.info.PayloadShapes[i].Count > g.info.PayloadShapes[j].Count })
		out.Groups = append(out.Groups, g.info)
	}
	sort.Slice(out.Groups, func(i, j int) bool { return out.Groups[i].Count > out.Groups[j].Count })
	return out, nil
}
