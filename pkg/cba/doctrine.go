package cba

import (
	"fmt"
	"sort"

	"aoe2kit/pkg/replay"
)

type DoctrineReport struct {
	Path         string              `json:"path,omitempty"`
	Method       string              `json:"method"`
	Verification string              `json:"verification"`
	Summary      DoctrineSummary     `json:"summary"`
	Players      []DoctrinePlayerRow `json:"players,omitempty"`
	Warnings     []string            `json:"warnings,omitempty"`
}

type DoctrineSummary struct {
	Players                  int    `json:"players"`
	Duration                 string `json:"duration,omitempty"`
	WinnerKnown              bool   `json:"winner_known"`
	RazeCandidates           int    `json:"raze_candidates"`
	SetPlayCandidates        int    `json:"set_play_candidates"`
	PlayersWithOwnVill       int    `json:"players_with_own_vill"`
	PlayersWithTeamVill      int    `json:"players_with_team_vill"`
	PlayersWithDuty          int    `json:"players_with_duty"`
	DirectKillAttribution    string `json:"direct_kill_attribution"`
	DirectRazeAttribution    string `json:"direct_raze_attribution"`
	ReplayPhaseBoundaryBasis string `json:"replay_phase_boundary_basis"`
}

type DoctrinePlayerRow struct {
	Slot                int      `json:"slot"`
	ProfileID           int      `json:"profile_id,omitempty"`
	Name                string   `json:"name,omitempty"`
	Team                string   `json:"team,omitempty"`
	CivID               int      `json:"civ_id,omitempty"`
	Civ                 string   `json:"civ,omitempty"`
	Won                 *int     `json:"won,omitempty"`
	Duty                string   `json:"duty"`
	DutyConfidence      string   `json:"duty_confidence"`
	PhaseRead           string   `json:"phase_read"`
	OwnFirstVillS       *int     `json:"own_first_vill_s,omitempty"`
	TeamFirstVillS      *int     `json:"team_first_vill_s,omitempty"`
	OwnRazes            *int     `json:"own_razes,omitempty"`
	TeamRazes           *int     `json:"team_razes,omitempty"`
	FirstPressureTime   string   `json:"first_pressure_time,omitempty"`
	FirstRazeCandidate  string   `json:"first_raze_candidate,omitempty"`
	SetPlayFinishes     int      `json:"set_play_finishes,omitempty"`
	FirstProdBuildS     *int     `json:"first_prod_build_s,omitempty"`
	ProdBuildings       *int     `json:"prod_buildings,omitempty"`
	UnitsProduced       *int     `json:"units_produced,omitempty"`
	UnitsLost           *int     `json:"units_lost,omitempty"`
	ProgressionSignal   string   `json:"progression_signal,omitempty"`
	CombatProxy         string   `json:"combat_proxy,omitempty"`
	MetricConfidence    string   `json:"metric_confidence"`
	AttributionWarnings []string `json:"attribution_warnings,omitempty"`
}

func BuildDoctrine(path string) (*DoctrineReport, error) {
	perf, err := BuildPerformance(path)
	if err != nil {
		return nil, err
	}
	progression, progressionErr := BuildProgression(path)
	razes, razeErr := BuildRazes(path)

	report := &DoctrineReport{
		Path:         path,
		Method:       "cba_doctrine_overlay_over_existing_replay_primitives",
		Verification: "domain_interpretation_over_mixed_confidence_metrics_no_new_replay_decode_claims",
		Summary: DoctrineSummary{
			Players:                  perf.Summary.Players,
			Duration:                 perf.Summary.Duration,
			WinnerKnown:              perf.Summary.WinnerKnown,
			RazeCandidates:           perf.Summary.RazeCandidates,
			SetPlayCandidates:        perf.Summary.SetPlayCandidates,
			DirectKillAttribution:    "not_available_from_replay_parser",
			DirectRazeAttribution:    "not_available_from_replay_parser",
			ReplayPhaseBoundaryBasis: "villager_gain_candidate_from_cba_raze_correlation_when_available",
		},
		Warnings: append([]string{}, perf.Warnings...),
	}
	if progressionErr != nil {
		report.Warnings = append(report.Warnings, fmt.Sprintf("progression overlay unavailable: %v", progressionErr))
	}
	if razeErr != nil {
		report.Warnings = append(report.Warnings, fmt.Sprintf("raze pressure overlay unavailable: %v", razeErr))
	}
	report.Warnings = append(report.Warnings,
		"doctrine duty is CBA-domain interpretation, not a generic AoE2Kit replay fact",
		"phase boundary currently uses villager-gain candidates when available; direct engine villager-award state is still a parser frontier",
	)

	progByPlayer := map[int]ProgressionPlayerSummary{}
	if progression != nil {
		for _, player := range progression.Players {
			progByPlayer[player.PlayerID] = player
		}
	}
	razeByPlayer := map[int]RazePlayerSummary{}
	if razes != nil {
		for _, player := range razes.Players {
			razeByPlayer[player.PlayerID] = player
		}
		report.Summary.RazeCandidates = razes.Summary.RazeCandidates
		report.Summary.SetPlayCandidates = razes.Summary.SetPlayCandidates
	}

	rows := make([]DoctrinePlayerRow, 0, len(perf.Rows))
	for _, perfRow := range perf.Rows {
		duty, dutyConfidence := civDuty(perfRow.Civ)
		row := DoctrinePlayerRow{
			Slot:                perfRow.Slot,
			ProfileID:           perfRow.ProfileID,
			Team:                perfRow.Team,
			Civ:                 perfRow.Civ,
			Won:                 cloneIntPtr(perfRow.Won),
			Duty:                duty,
			DutyConfidence:      dutyConfidence,
			PhaseRead:           phaseRead(perfRow.OwnFirstVillS, perfRow.TeamFirstVillS),
			OwnFirstVillS:       cloneIntPtr(perfRow.OwnFirstVillS),
			TeamFirstVillS:      cloneIntPtr(perfRow.TeamFirstVillS),
			OwnRazes:            cloneIntPtr(perfRow.OwnRazes),
			TeamRazes:           cloneIntPtr(perfRow.TeamRazes),
			FirstProdBuildS:     cloneIntPtr(perfRow.FirstProdBuildS),
			ProdBuildings:       cloneIntPtr(perfRow.ProdBuildings),
			UnitsProduced:       cloneIntPtr(perfRow.UnitsProduced),
			UnitsLost:           cloneIntPtr(perfRow.UnitsLost),
			MetricConfidence:    perfRow.MetricConfidence,
			AttributionWarnings: append([]string{}, perfRow.AttributionWarnings...),
		}
		if dutyConfidence != "unclassified" {
			report.Summary.PlayersWithDuty++
		}
		if row.OwnFirstVillS != nil {
			report.Summary.PlayersWithOwnVill++
		}
		if row.TeamFirstVillS != nil {
			report.Summary.PlayersWithTeamVill++
		}
		if row.UnitsProduced != nil && row.UnitsLost != nil {
			row.CombatProxy = fmt.Sprintf("%d checksum removals over %d checksum additions", *row.UnitsLost, *row.UnitsProduced)
		}
		if prog, ok := progByPlayer[perfRow.Slot]; ok {
			row.Name = prog.Name
			row.CivID = prog.CivID
			row.ProgressionSignal = progressionSignalText(prog)
		}
		if raze, ok := razeByPlayer[perfRow.Slot]; ok {
			if row.Name == "" {
				row.Name = raze.Name
			}
			if row.CivID == 0 {
				row.CivID = raze.CivID
			}
			row.FirstPressureTime = raze.FirstPressureTime
			row.FirstRazeCandidate = raze.FirstRazeCandidate
			row.SetPlayFinishes = raze.SetPlayFinishes
		}
		rows = append(rows, row)
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].Slot < rows[j].Slot })
	report.Players = rows
	return report, nil
}

func civDuty(civ string) (string, string) {
	switch civ {
	case "Persians":
		return "carry threat: guarded War Elephant packs set gates, threaten castles, and force anti-elephant answers; own gates are a targetable weakness", "case_game_chrae_doctrine"
	case "Georgians":
		return "cavalry/Monaspa pressure; newer-civ read is low-confidence in the current doctrine", "case_game_low_confidence"
	case "Franks":
		return "Throwing Axemen deliver ranged melee damage; expected to output damage and set gates but must avoid feeding", "case_game_chrae_doctrine"
	case "Spanish":
		return "Conquistador hit-and-run output and gate setting; late Paladin transition if post-vill phase opens", "case_game_chrae_doctrine"
	case "Berbers":
		return "anti-cavalry and anti-Conquistador Camel Archer duty; draws focus from elephant/cavalry threats and razes gates moderately", "case_game_chrae_doctrine"
	case "Mongols":
		return "Mangudai mobility/output role; not yet deeply covered by chrae's case-game doctrine", "case_game_low_confidence"
	case "Byzantines":
		return "Cataphract/cavalry responsibility against counter-damage lines; anti-Persia effort can be high-value but low-visibility", "case_game_chrae_doctrine"
	case "Khmer":
		return "carry duty: Ballista Elephant mass is the other elephant power spike and should clear tight groups/areas", "case_game_chrae_doctrine"
	default:
		return "unclassified CBA duty; needs matchup doctrine before evaluation", "unclassified"
	}
}

func phaseRead(own, team *int) string {
	if own != nil {
		return fmt.Sprintf("player_post_vill_from_%s", replay.FormatTime(*own*1000))
	}
	if team != nil {
		return fmt.Sprintf("team_post_vill_from_%s_player_not_confirmed", replay.FormatTime(*team*1000))
	}
	return "phase_boundary_unknown_no_vill_gain_candidate_detected"
}

func progressionSignalText(player ProgressionPlayerSummary) string {
	if player.FirstImperialProxyAt != "" {
		return fmt.Sprintf("imperial_proxy %s at %s", player.FirstImperialProxy, player.FirstImperialProxyAt)
	}
	if player.FeudalProductionAt != "" {
		return fmt.Sprintf("early_or_feudal_production %s at %s", player.FeudalProductionUnit, player.FeudalProductionAt)
	}
	if player.FirstProductionTime != "" {
		return fmt.Sprintf("first_production %s at %s", player.FirstProductionUnit, player.FirstProductionTime)
	}
	return ""
}
