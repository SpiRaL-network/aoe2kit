package cba

import (
	"sort"

	"aoe2kit/pkg/replay"
)

const (
	razePressureMethod       = "cba_gate_castle_pressure_joined_to_checksum_loss_pulses"
	razePressureVerification = "behavior_inferred_target_pressure_plus_victim_net_loss_not_engine_confirmed_raze"
)

type RazePressureReport struct {
	Path         string               `json:"path,omitempty"`
	Method       string               `json:"method"`
	Verification string               `json:"verification"`
	Summary      RazePressureSummary  `json:"summary"`
	Players      []RazePressurePlayer `json:"players,omitempty"`
	Targets      []RazePressureTarget `json:"targets,omitempty"`
	Warnings     []string             `json:"warnings,omitempty"`
}

type RazePressureSummary struct {
	Targets              int `json:"targets"`
	PrimaryTargets       int `json:"primary_targets"`
	GateTargets          int `json:"gate_targets"`
	CastleTargets        int `json:"castle_targets"`
	WallTargets          int `json:"wall_targets"`
	OtherBuildingTargets int `json:"other_building_targets"`
	PressureEvents       int `json:"pressure_events"`
	NearbyLossPulses     int `json:"nearby_loss_pulses"`
	NetObjectsLostNearby int `json:"net_objects_lost_nearby"`
	PlayersWithPressure  int `json:"players_with_pressure"`
}

type RazePressurePlayer struct {
	PlayerID              int    `json:"player_id"`
	PlayerLabel           string `json:"player_label"`
	PlayerName            string `json:"player_name,omitempty"`
	PressureEvents        int    `json:"pressure_events"`
	PrimaryPressureEvents int    `json:"primary_pressure_events"`
	Targets               int    `json:"targets"`
	PrimaryTargets        int    `json:"primary_targets"`
	NearbyLossTargets     int    `json:"nearby_loss_targets"`
	NetObjectsLostNearby  int    `json:"net_objects_lost_nearby"`
}

type RazePressureTarget struct {
	TargetID             int                    `json:"target_id"`
	OwnerID              int                    `json:"owner_id,omitempty"`
	OwnerLabel           string                 `json:"owner_label,omitempty"`
	OwnerName            string                 `json:"owner_name,omitempty"`
	UnitID               int                    `json:"unit_id,omitempty"`
	UnitName             string                 `json:"unit_name,omitempty"`
	Kind                 string                 `json:"kind"`
	Primary              bool                   `json:"primary"`
	X                    float64                `json:"x,omitempty"`
	Y                    float64                `json:"y,omitempty"`
	FirstPressureTimeMS  int                    `json:"first_pressure_time_ms,omitempty"`
	FirstPressureTime    string                 `json:"first_pressure_time,omitempty"`
	LastPressureTimeMS   int                    `json:"last_pressure_time_ms,omitempty"`
	LastPressureTime     string                 `json:"last_pressure_time,omitempty"`
	PressureEvents       int                    `json:"pressure_events"`
	NearbyLossPulses     int                    `json:"nearby_loss_pulses"`
	NetObjectsLostNearby int                    `json:"net_objects_lost_nearby"`
	Attackers            []RazePressureAttacker `json:"attackers,omitempty"`
	Confidence           string                 `json:"confidence"`
}

type RazePressureAttacker struct {
	PlayerID   int    `json:"player_id"`
	PlayerName string `json:"player_name,omitempty"`
	Events     int    `json:"events"`
}

func BuildRazePressure(path string) (*RazePressureReport, error) {
	combat, err := replay.BuildCombatStory(path, replay.CombatOptions{Limit: 0})
	if err != nil {
		return nil, err
	}
	report := &RazePressureReport{
		Path:         path,
		Method:       razePressureMethod,
		Verification: razePressureVerification,
		Warnings: append([]string{
			"targets are command-pressure groups joined to nearby victim checksum loss pulses; target object destruction is not directly observed",
			"nearby net loss is owner-wide object-count loss during the pressure window, not a per-target HP/death packet",
		}, combat.Warnings...),
	}
	players := map[int]*RazePressurePlayer{}
	for _, target := range combat.Targets {
		kind, primary, ok := classifyRazePressureTarget(target.TargetUnitID, target.TargetClass)
		if !ok {
			continue
		}
		row := RazePressureTarget{
			TargetID:             target.TargetObjectID,
			OwnerID:              target.TargetOwnerID,
			OwnerLabel:           playerLabel(target.TargetOwnerID),
			OwnerName:            target.TargetOwnerName,
			UnitID:               target.TargetUnitID,
			UnitName:             target.TargetUnitName,
			Kind:                 kind,
			Primary:              primary,
			X:                    target.TargetX,
			Y:                    target.TargetY,
			FirstPressureTimeMS:  target.FirstPressureTimeMS,
			FirstPressureTime:    target.FirstPressureTime,
			LastPressureTimeMS:   target.LastPressureTimeMS,
			LastPressureTime:     target.LastPressureTime,
			PressureEvents:       target.PressureEvents,
			NearbyLossPulses:     target.NearbyLossPulses,
			NetObjectsLostNearby: target.NetObjectsLostNearby,
			Attackers:            razePressureAttackers(target.PressureByAttacker),
			Confidence:           "target_pressure_only_no_nearby_loss",
		}
		if target.NearbyLossPulses > 0 {
			row.Confidence = "target_pressure_plus_owner_net_loss_nearby_not_object_specific"
		}
		report.Targets = append(report.Targets, row)
		report.Summary.Targets++
		report.Summary.PressureEvents += target.PressureEvents
		report.Summary.NearbyLossPulses += target.NearbyLossPulses
		report.Summary.NetObjectsLostNearby += target.NetObjectsLostNearby
		if primary {
			report.Summary.PrimaryTargets++
		}
		switch kind {
		case "gate":
			report.Summary.GateTargets++
		case "castle":
			report.Summary.CastleTargets++
		case "wall":
			report.Summary.WallTargets++
		default:
			report.Summary.OtherBuildingTargets++
		}
		for _, attacker := range row.Attackers {
			player := players[attacker.PlayerID]
			if player == nil {
				player = &RazePressurePlayer{
					PlayerID:    attacker.PlayerID,
					PlayerLabel: playerLabel(attacker.PlayerID),
					PlayerName:  attacker.PlayerName,
				}
				players[attacker.PlayerID] = player
			}
			if player.PlayerName == "" {
				player.PlayerName = attacker.PlayerName
			}
			player.PressureEvents += attacker.Events
			player.Targets++
			if primary {
				player.PrimaryPressureEvents += attacker.Events
				player.PrimaryTargets++
			}
			if row.NearbyLossPulses > 0 {
				player.NearbyLossTargets++
				player.NetObjectsLostNearby += row.NetObjectsLostNearby
			}
		}
	}
	sort.Slice(report.Targets, func(i, j int) bool {
		if report.Targets[i].Primary != report.Targets[j].Primary {
			return report.Targets[i].Primary
		}
		if report.Targets[i].NetObjectsLostNearby != report.Targets[j].NetObjectsLostNearby {
			return report.Targets[i].NetObjectsLostNearby > report.Targets[j].NetObjectsLostNearby
		}
		if report.Targets[i].PressureEvents != report.Targets[j].PressureEvents {
			return report.Targets[i].PressureEvents > report.Targets[j].PressureEvents
		}
		if report.Targets[i].FirstPressureTimeMS != report.Targets[j].FirstPressureTimeMS {
			return report.Targets[i].FirstPressureTimeMS < report.Targets[j].FirstPressureTimeMS
		}
		return report.Targets[i].TargetID < report.Targets[j].TargetID
	})
	report.Players = razePressurePlayers(players)
	report.Summary.PlayersWithPressure = len(report.Players)
	return report, nil
}

func classifyRazePressureTarget(unitID int, class string) (string, bool, bool) {
	switch unitID {
	case 82:
		return "castle", true, true
	case 64, 88, 95:
		return "gate", true, true
	case 117:
		return "wall", false, true
	}
	switch class {
	case "gate":
		return "gate", true, true
	case "building":
		return "building", false, true
	default:
		return "", false, false
	}
}

func razePressureAttackers(values []replay.CombatPressureAggressor) []RazePressureAttacker {
	out := make([]RazePressureAttacker, 0, len(values))
	for _, value := range values {
		out = append(out, RazePressureAttacker{
			PlayerID:   value.PlayerID,
			PlayerName: value.PlayerName,
			Events:     value.Events,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Events != out[j].Events {
			return out[i].Events > out[j].Events
		}
		return out[i].PlayerID < out[j].PlayerID
	})
	return out
}

func razePressurePlayers(values map[int]*RazePressurePlayer) []RazePressurePlayer {
	out := make([]RazePressurePlayer, 0, len(values))
	for _, value := range values {
		out = append(out, *value)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].PrimaryPressureEvents != out[j].PrimaryPressureEvents {
			return out[i].PrimaryPressureEvents > out[j].PrimaryPressureEvents
		}
		if out[i].NetObjectsLostNearby != out[j].NetObjectsLostNearby {
			return out[i].NetObjectsLostNearby > out[j].NetObjectsLostNearby
		}
		return out[i].PlayerID < out[j].PlayerID
	})
	return out
}
