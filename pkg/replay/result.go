package replay

import "sort"

type MatchResult struct {
	WinnerKnown  bool          `json:"winner_known"`
	Completed    bool          `json:"completed"`
	Method       string        `json:"method"`
	Winners      []int         `json:"winners,omitempty"`
	Losers       []int         `json:"losers,omitempty"`
	Resigned     []int         `json:"resigned,omitempty"`
	DurationMS   int           `json:"duration_ms,omitempty"`
	Duration     string        `json:"duration,omitempty"`
	ResignEvents []ResignEvent `json:"resign_events,omitempty"`
	Warnings     []string      `json:"warnings,omitempty"`
}

type ResignEvent struct {
	TimeMS   int    `json:"time_ms"`
	Time     string `json:"time"`
	PlayerID int    `json:"player_id"`
	Sequence int    `json:"sequence,omitempty"`
}

func InferResult(players []FeedbackPlayer, events []ReplayEvent, counts EventCounts, parseWarnings []string) MatchResult {
	result := MatchResult{
		Method:     "unknown",
		DurationMS: counts.DurationMS,
		Duration:   counts.Duration,
	}
	playerIDs := activePlayerIDs(players)
	resigned := map[int]bool{}
	for _, event := range events {
		if event.Type != "resign" || event.PlayerID <= 0 {
			continue
		}
		resigned[event.PlayerID] = true
		result.ResignEvents = append(result.ResignEvents, ResignEvent{
			TimeMS:   event.TimeMS,
			Time:     event.Time,
			PlayerID: event.PlayerID,
			Sequence: event.Sequence,
		})
	}
	if len(result.ResignEvents) > 0 {
		for playerID := range resigned {
			if containsInt(playerIDs, playerID) {
				result.Resigned = append(result.Resigned, playerID)
			}
		}
		sort.Ints(result.Resigned)
		resignedKnown := map[int]bool{}
		for _, playerID := range result.Resigned {
			resignedKnown[playerID] = true
		}
		result.Completed = true
		for _, playerID := range playerIDs {
			if resignedKnown[playerID] {
				result.Losers = append(result.Losers, playerID)
			}
		}
		survivors := make([]int, 0, len(playerIDs)-len(result.Losers))
		for _, playerID := range playerIDs {
			if !resignedKnown[playerID] {
				survivors = append(survivors, playerID)
			}
		}
		if len(survivors) == 1 {
			result.Winners = append(result.Winners, survivors[0])
			result.WinnerKnown = true
			result.Method = "action_stream_resign_last_single_survivor"
		} else {
			result.WinnerKnown = false
			result.Method = "action_stream_resign_partial_without_team_outcome"
			result.Warnings = append(result.Warnings, "multiple players lacked recorded resign/defeat events; replay may stop at game end before recording the final losing player's defeat, so absence of a defeat is not winner evidence without team/outcome corroboration")
		}
		result.Warnings = append(result.Warnings, parseWarnings...)
		return result
	}
	if counts.Postgames > 0 {
		result.Completed = true
		result.Method = "postgame_seen_without_decoded_winner"
	}
	if counts.UnknownOps > 0 {
		result.Warnings = append(result.Warnings, "action stream stopped before a decoded result signal; outcome may exist later in the replay")
	}
	result.Warnings = append(result.Warnings, parseWarnings...)
	return result
}

func activePlayerIDs(players []FeedbackPlayer) []int {
	var ids []int
	for _, player := range players {
		if player.PlayerID <= 0 {
			continue
		}
		switch player.Kind {
		case "absent", "closed", "spectator":
			continue
		}
		ids = append(ids, player.PlayerID)
	}
	sort.Ints(ids)
	out := ids[:0]
	prev := -1
	for _, id := range ids {
		if id == prev {
			continue
		}
		out = append(out, id)
		prev = id
	}
	return out
}

func containsInt(values []int, value int) bool {
	for _, existing := range values {
		if existing == value {
			return true
		}
	}
	return false
}
