package cba

import (
	"sort"

	"aoe2kit/pkg/replay"
)

const defaultRazeCorrelationWindowMS = 5 * 60 * 1000

type RazeReport struct {
	Path         string              `json:"path,omitempty"`
	Method       string              `json:"method"`
	Verification string              `json:"verification"`
	Summary      RazeSummary         `json:"summary"`
	Players      []RazePlayerSummary `json:"players,omitempty"`
	Targets      []TargetPressure    `json:"targets,omitempty"`
	Events       []RazeCandidate     `json:"events,omitempty"`
	SetPlays     []SetPlayCandidate  `json:"set_plays,omitempty"`
	Warnings     []string            `json:"warnings,omitempty"`
}

type RazeSummary struct {
	BuildingTargetEvents int `json:"building_target_events"`
	PressuredTargets     int `json:"pressured_targets"`
	MultiPlayerTargets   int `json:"multi_player_targets"`
	RazeCandidates       int `json:"raze_candidates"`
	SetPlayCandidates    int `json:"set_play_candidates"`
}

type RazePlayerSummary struct {
	PlayerID             int    `json:"player_id"`
	Label                string `json:"label"`
	Name                 string `json:"name,omitempty"`
	CivID                int    `json:"civ_id,omitempty"`
	CivName              string `json:"civ_name,omitempty"`
	RazeArchetype        string `json:"raze_archetype"`
	FirstPressureTime    string `json:"first_pressure_time,omitempty"`
	FirstPressureTarget  int    `json:"first_pressure_target,omitempty"`
	PressureOrder        int    `json:"pressure_order,omitempty"`
	PressureArchetypeFit string `json:"pressure_archetype_fit,omitempty"`
	FirstRazeCandidate   string `json:"first_raze_candidate,omitempty"`
	FirstRazeTarget      int    `json:"first_raze_target,omitempty"`
	SetPlayFinishes      int    `json:"set_play_finishes"`
	RazeOrder            int    `json:"raze_order,omitempty"`
	ArchetypeFit         string `json:"archetype_fit,omitempty"`
	Confidence           string `json:"confidence"`
}

type TargetPressure struct {
	TargetID        int                    `json:"target_id"`
	OwnerID         int                    `json:"owner_id,omitempty"`
	OwnerLabel      string                 `json:"owner_label,omitempty"`
	Class           string                 `json:"class,omitempty"`
	UnitID          int                    `json:"unit_id,omitempty"`
	UnitName        string                 `json:"unit_name,omitempty"`
	X               float64                `json:"x,omitempty"`
	Y               float64                `json:"y,omitempty"`
	FirstTimeMS     int                    `json:"first_time_ms"`
	FirstTime       string                 `json:"first_time"`
	LastTimeMS      int                    `json:"last_time_ms"`
	LastTime        string                 `json:"last_time"`
	Events          int                    `json:"events"`
	DistinctPlayers int                    `json:"distinct_players"`
	Players         []TargetPlayerPressure `json:"players,omitempty"`
}

type TargetPlayerPressure struct {
	PlayerID    int    `json:"player_id"`
	PlayerLabel string `json:"player_label,omitempty"`
	FirstTimeMS int    `json:"first_time_ms"`
	FirstTime   string `json:"first_time"`
	LastTimeMS  int    `json:"last_time_ms"`
	LastTime    string `json:"last_time"`
	Events      int    `json:"events"`
}

type RazeCandidate struct {
	TimeMS            int     `json:"time_ms"`
	Time              string  `json:"time"`
	PlayerID          int     `json:"player_id"`
	PlayerLabel       string  `json:"player_label,omitempty"`
	VillagerObjectID  int     `json:"villager_object_id"`
	VillagerUnitID    int     `json:"villager_unit_id,omitempty"`
	VillagerUnitName  string  `json:"villager_unit_name,omitempty"`
	TargetID          int     `json:"target_id"`
	TargetClass       string  `json:"target_class,omitempty"`
	TargetOwnerID     int     `json:"target_owner_id,omitempty"`
	TargetOwnerLabel  string  `json:"target_owner_label,omitempty"`
	TargetUnitID      int     `json:"target_unit_id,omitempty"`
	TargetUnitName    string  `json:"target_unit_name,omitempty"`
	TargetX           float64 `json:"target_x,omitempty"`
	TargetY           float64 `json:"target_y,omitempty"`
	LastAttackTimeMS  int     `json:"last_attack_time_ms"`
	LastAttackTime    string  `json:"last_attack_time"`
	DeltaMS           int     `json:"delta_ms"`
	Delta             string  `json:"delta"`
	DistinctAttackers int     `json:"distinct_attackers"`
	Confidence        string  `json:"confidence"`
	Reason            string  `json:"reason"`
}

type SetPlayCandidate struct {
	TimeMS           int                    `json:"time_ms"`
	Time             string                 `json:"time"`
	TargetID         int                    `json:"target_id"`
	TargetClass      string                 `json:"target_class,omitempty"`
	TargetOwnerID    int                    `json:"target_owner_id,omitempty"`
	TargetOwnerLabel string                 `json:"target_owner_label,omitempty"`
	WeakenerSequence []TargetPlayerPressure `json:"weakener_sequence,omitempty"`
	FinisherID       int                    `json:"finisher_id"`
	FinisherLabel    string                 `json:"finisher_label,omitempty"`
	VillagerObjectID int                    `json:"villager_object_id"`
	Confidence       string                 `json:"confidence"`
	Reason           string                 `json:"reason"`
}

type targetPressureBuilder struct {
	ref     replay.ObjectReference
	events  int
	players map[int]*TargetPlayerPressure
	firstMS int
	lastMS  int
}

type playerPressureFirst struct {
	PlayerID int
	TargetID int
	TimeMS   int
}

func BuildRazes(path string) (*RazeReport, error) {
	rec, err := replay.Open(path)
	if err != nil {
		return nil, err
	}
	eventReport, err := replay.ExtractEvents(path, replay.EventOptions{IncludeUntypedAction: true, IncludeObjectIndex: true})
	if err != nil {
		return nil, err
	}
	lifecycle, err := replay.BuildLifecycle(path, replay.LifecycleOptions{})
	if err != nil {
		return nil, err
	}
	targets, targetEvents := collectTargetPressure(eventReport.Events)
	razeEvents, setPlays := correlateRazeCandidates(lifecycle.Candidates, targets)
	players := summarizeRazePlayers(rec.Players, razeEvents, setPlays, targets)
	report := &RazeReport{
		Path:         path,
		Method:       "building_target_pressure_plus_villager_lifecycle_payoff",
		Verification: "behavior_inferred_raze_candidate_not_engine_raze_event",
		Summary: RazeSummary{
			BuildingTargetEvents: targetEvents,
			PressuredTargets:     len(targets),
			RazeCandidates:       len(razeEvents),
			SetPlayCandidates:    len(setPlays),
		},
		Players:  players,
		Targets:  targetPressures(targets),
		Events:   razeEvents,
		SetPlays: setPlays,
		Warnings: []string{
			"raze candidates are inferred from target pressure followed by villager lifecycle payoff; no direct raze packet has been decoded",
		},
	}
	for _, target := range report.Targets {
		if target.DistinctPlayers >= 2 {
			report.Summary.MultiPlayerTargets++
		}
	}
	return report, nil
}

func collectTargetPressure(events []replay.ReplayEvent) (map[int]*targetPressureBuilder, int) {
	targets := map[int]*targetPressureBuilder{}
	total := 0
	for _, event := range events {
		if event.TargetID <= 0 || event.PlayerID <= 0 || event.TargetObject == nil {
			continue
		}
		if event.Type != "order" && event.Type != "attack_move" && event.Type != "attack_ground" {
			continue
		}
		if event.TargetObject.Class != "gate" && event.TargetObject.Class != "building" {
			continue
		}
		if event.TargetObject.OwnerID == event.PlayerID {
			continue
		}
		total++
		target := targets[event.TargetID]
		if target == nil {
			target = &targetPressureBuilder{ref: *event.TargetObject, players: map[int]*TargetPlayerPressure{}, firstMS: event.TimeMS, lastMS: event.TimeMS}
			targets[event.TargetID] = target
		}
		target.events++
		if event.TimeMS < target.firstMS {
			target.firstMS = event.TimeMS
		}
		if event.TimeMS > target.lastMS {
			target.lastMS = event.TimeMS
		}
		player := target.players[event.PlayerID]
		if player == nil {
			player = &TargetPlayerPressure{PlayerID: event.PlayerID, PlayerLabel: playerLabel(event.PlayerID), FirstTimeMS: event.TimeMS, LastTimeMS: event.TimeMS}
			target.players[event.PlayerID] = player
		}
		player.Events++
		if event.TimeMS < player.FirstTimeMS {
			player.FirstTimeMS = event.TimeMS
		}
		if event.TimeMS > player.LastTimeMS {
			player.LastTimeMS = event.TimeMS
		}
	}
	return targets, total
}

func correlateRazeCandidates(candidates []replay.LifecycleObjectCandidate, targets map[int]*targetPressureBuilder) ([]RazeCandidate, []SetPlayCandidate) {
	var razes []RazeCandidate
	var sets []SetPlayCandidate
	seenRazes := map[struct {
		playerID int
		timeMS   int
		targetID int
	}]bool{}
	seenSets := map[struct {
		finisherID int
		timeMS     int
		targetID   int
	}]bool{}
	for _, candidate := range candidates {
		if candidate.Kind != "villager_spawn_candidate" {
			continue
		}
		target, playerPressure, ok := latestTargetBeforeVillager(candidate, targets)
		if !ok {
			continue
		}
		razeKey := struct {
			playerID int
			timeMS   int
			targetID int
		}{candidate.PlayerID, candidate.TimeMS, target.ref.ObjectID}
		if seenRazes[razeKey] {
			continue
		}
		seenRazes[razeKey] = true
		sequence := sortedTargetPlayersBefore(target, candidate.TimeMS)
		raze := RazeCandidate{
			TimeMS:            candidate.TimeMS,
			Time:              candidate.Time,
			PlayerID:          candidate.PlayerID,
			PlayerLabel:       candidate.PlayerLabel,
			VillagerObjectID:  candidate.ObjectID,
			VillagerUnitID:    candidate.UnitID,
			VillagerUnitName:  candidate.UnitName,
			TargetID:          target.ref.ObjectID,
			TargetClass:       target.ref.Class,
			TargetOwnerID:     target.ref.OwnerID,
			TargetOwnerLabel:  target.ref.OwnerLabel,
			TargetUnitID:      target.ref.UnitID,
			TargetUnitName:    replay.UnitDisplayName(target.ref.UnitID),
			TargetX:           target.ref.X,
			TargetY:           target.ref.Y,
			LastAttackTimeMS:  playerPressure.LastTimeMS,
			LastAttackTime:    replay.FormatTime(playerPressure.LastTimeMS),
			DeltaMS:           candidate.TimeMS - playerPressure.LastTimeMS,
			Delta:             replay.FormatTime(candidate.TimeMS - playerPressure.LastTimeMS),
			DistinctAttackers: len(sequence),
			Confidence:        "behavior_inferred_target_pressure_before_villager",
			Reason:            "villager lifecycle candidate followed this player's recent building/gate target pressure",
		}
		razes = append(razes, raze)
		if len(sequence) >= 2 {
			setKey := struct {
				finisherID int
				timeMS     int
				targetID   int
			}{candidate.PlayerID, candidate.TimeMS, target.ref.ObjectID}
			if seenSets[setKey] {
				continue
			}
			seenSets[setKey] = true
			sets = append(sets, SetPlayCandidate{
				TimeMS:           candidate.TimeMS,
				Time:             candidate.Time,
				TargetID:         target.ref.ObjectID,
				TargetClass:      target.ref.Class,
				TargetOwnerID:    target.ref.OwnerID,
				TargetOwnerLabel: target.ref.OwnerLabel,
				WeakenerSequence: sequence,
				FinisherID:       candidate.PlayerID,
				FinisherLabel:    candidate.PlayerLabel,
				VillagerObjectID: candidate.ObjectID,
				Confidence:       "behavior_inferred_set_play_candidate",
				Reason:           "same target had ordered pressure from two or more players before the finisher's villager payoff",
			})
		}
	}
	sort.Slice(razes, func(i, j int) bool {
		if razes[i].TimeMS != razes[j].TimeMS {
			return razes[i].TimeMS < razes[j].TimeMS
		}
		return razes[i].PlayerID < razes[j].PlayerID
	})
	sort.Slice(sets, func(i, j int) bool {
		if sets[i].TimeMS != sets[j].TimeMS {
			return sets[i].TimeMS < sets[j].TimeMS
		}
		return sets[i].TargetID < sets[j].TargetID
	})
	return razes, sets
}

func latestTargetBeforeVillager(candidate replay.LifecycleObjectCandidate, targets map[int]*targetPressureBuilder) (*targetPressureBuilder, *TargetPlayerPressure, bool) {
	var bestTarget *targetPressureBuilder
	var bestPressure *TargetPlayerPressure
	bestDelta := defaultRazeCorrelationWindowMS + 1
	for _, target := range targets {
		pressure := target.players[candidate.PlayerID]
		if pressure == nil || pressure.LastTimeMS > candidate.TimeMS {
			continue
		}
		delta := candidate.TimeMS - pressure.LastTimeMS
		if delta <= defaultRazeCorrelationWindowMS && delta < bestDelta {
			bestTarget = target
			bestPressure = pressure
			bestDelta = delta
		}
	}
	return bestTarget, bestPressure, bestTarget != nil
}

func targetPressures(targets map[int]*targetPressureBuilder) []TargetPressure {
	out := make([]TargetPressure, 0, len(targets))
	for _, target := range targets {
		players := sortedTargetPlayers(target)
		out = append(out, TargetPressure{
			TargetID:        target.ref.ObjectID,
			OwnerID:         target.ref.OwnerID,
			OwnerLabel:      target.ref.OwnerLabel,
			Class:           target.ref.Class,
			UnitID:          target.ref.UnitID,
			UnitName:        replay.UnitDisplayName(target.ref.UnitID),
			X:               target.ref.X,
			Y:               target.ref.Y,
			FirstTimeMS:     target.firstMS,
			FirstTime:       replay.FormatTime(target.firstMS),
			LastTimeMS:      target.lastMS,
			LastTime:        replay.FormatTime(target.lastMS),
			Events:          target.events,
			DistinctPlayers: len(players),
			Players:         players,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].FirstTimeMS != out[j].FirstTimeMS {
			return out[i].FirstTimeMS < out[j].FirstTimeMS
		}
		return out[i].TargetID < out[j].TargetID
	})
	return out
}

func sortedTargetPlayers(target *targetPressureBuilder) []TargetPlayerPressure {
	return sortedTargetPlayersBefore(target, 0)
}

func sortedTargetPlayersBefore(target *targetPressureBuilder, cutoffMS int) []TargetPlayerPressure {
	players := make([]TargetPlayerPressure, 0, len(target.players))
	for _, player := range target.players {
		if cutoffMS > 0 && player.FirstTimeMS > cutoffMS {
			continue
		}
		player.FirstTime = replay.FormatTime(player.FirstTimeMS)
		player.LastTime = replay.FormatTime(player.LastTimeMS)
		players = append(players, *player)
	}
	sort.Slice(players, func(i, j int) bool {
		if players[i].FirstTimeMS != players[j].FirstTimeMS {
			return players[i].FirstTimeMS < players[j].FirstTimeMS
		}
		return players[i].PlayerID < players[j].PlayerID
	})
	return players
}

func summarizeRazePlayers(players []replay.PlayerSlot, razes []RazeCandidate, sets []SetPlayCandidate, targets map[int]*targetPressureBuilder) []RazePlayerSummary {
	byPlayer := map[int]*RazePlayerSummary{}
	for _, player := range players {
		if !player.Active || player.Number <= 0 {
			continue
		}
		civName := replay.CivDisplayName(player.Civ)
		byPlayer[player.Number] = &RazePlayerSummary{
			PlayerID:      player.Number,
			Label:         playerLabel(player.Number),
			Name:          player.Name,
			CivID:         player.Civ,
			CivName:       civName,
			RazeArchetype: CivRazeArchetype(player.Civ),
			Confidence:    "behavior_inferred_raze_order",
		}
	}
	pressureFirsts := firstPressureByPlayer(targets)
	for i, pressure := range pressureFirsts {
		summary := byPlayer[pressure.PlayerID]
		if summary == nil {
			continue
		}
		summary.FirstPressureTime = replay.FormatTime(pressure.TimeMS)
		summary.FirstPressureTarget = pressure.TargetID
		summary.PressureOrder = i + 1
		summary.PressureArchetypeFit = razeArchetypeFit(summary.RazeArchetype, i+1)
	}
	firsts := map[int]RazeCandidate{}
	for _, raze := range razes {
		if _, ok := firsts[raze.PlayerID]; !ok {
			firsts[raze.PlayerID] = raze
		}
	}
	ordered := make([]RazeCandidate, 0, len(firsts))
	for _, raze := range firsts {
		ordered = append(ordered, raze)
	}
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].TimeMS < ordered[j].TimeMS })
	for i, raze := range ordered {
		summary := byPlayer[raze.PlayerID]
		if summary == nil {
			continue
		}
		summary.FirstRazeCandidate = raze.Time
		summary.FirstRazeTarget = raze.TargetID
		summary.RazeOrder = i + 1
		summary.ArchetypeFit = razeArchetypeFit(summary.RazeArchetype, i+1)
	}
	for _, set := range sets {
		if summary := byPlayer[set.FinisherID]; summary != nil {
			summary.SetPlayFinishes++
		}
	}
	out := make([]RazePlayerSummary, 0, len(byPlayer))
	for _, summary := range byPlayer {
		out = append(out, *summary)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].PlayerID < out[j].PlayerID })
	return out
}

func firstPressureByPlayer(targets map[int]*targetPressureBuilder) []playerPressureFirst {
	firsts := map[int]playerPressureFirst{}
	for _, target := range targets {
		for playerID, pressure := range target.players {
			if existing, ok := firsts[playerID]; !ok || pressure.FirstTimeMS < existing.TimeMS {
				firsts[playerID] = playerPressureFirst{
					PlayerID: playerID,
					TargetID: target.ref.ObjectID,
					TimeMS:   pressure.FirstTimeMS,
				}
			}
		}
	}
	out := make([]playerPressureFirst, 0, len(firsts))
	for _, first := range firsts {
		out = append(out, first)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].TimeMS != out[j].TimeMS {
			return out[i].TimeMS < out[j].TimeMS
		}
		return out[i].PlayerID < out[j].PlayerID
	})
	return out
}

func razeArchetypeFit(archetype string, order int) string {
	switch archetype {
	case "raze_first_expected":
		if order > 0 && order <= 3 {
			return "fits_early_raze_expectation"
		}
		return "late_for_raze_first_archetype"
	case "defer_expected":
		if order == 0 {
			return "no_raze_candidate_detected"
		}
		if order >= 5 {
			return "fits_defer_expectation"
		}
		return "early_for_defer_archetype"
	case "weak_razer_set_help_expected":
		return "needs_set_context"
	default:
		if order == 0 {
			return "no_raze_candidate_detected"
		}
		return "unclassified"
	}
}
