package cba

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"aoe2kit/pkg/replay"
)

type PhaseTimelineReport struct {
	Path         string                `json:"path,omitempty"`
	DatPath      string                `json:"dat_path,omitempty"`
	Method       string                `json:"method"`
	Verification string                `json:"verification"`
	Summary      PhaseTimelineSummary  `json:"summary"`
	Players      []PhaseTimelinePlayer `json:"players,omitempty"`
	Warnings     []string              `json:"warnings,omitempty"`
}

type PhaseTimelineSummary struct {
	Players          int    `json:"players"`
	CastleObserved   int    `json:"castle_observed"`
	ImperialObserved int    `json:"imperial_observed"`
	PhaseFactsCivs   int    `json:"phase_facts_civs"`
	SyncDeltas       int    `json:"sync_deltas"`
	DurationMS       int    `json:"duration_ms"`
	Duration         string `json:"duration"`
	Confidence       string `json:"confidence"`
}

type PhaseTimelinePlayer struct {
	PlayerID               int          `json:"player_id"`
	PlayerLabel            string       `json:"player_label"`
	PlayerName             string       `json:"player_name,omitempty"`
	CivID                  int          `json:"civ_id,omitempty"`
	CivName                string       `json:"civ_name,omitempty"`
	CastleThresholdKills   int          `json:"castle_threshold_kills,omitempty"`
	ImperialThresholdKills int          `json:"imperial_threshold_kills,omitempty"`
	RazesToVillager        int          `json:"razes_to_villager,omitempty"`
	Castle                 *PhaseAnchor `json:"castle,omitempty"`
	Imperial               *PhaseAnchor `json:"imperial,omitempty"`
	EarlySpawnUnitID       int          `json:"early_spawn_unit_id,omitempty"`
	EarlySpawnUnitName     string       `json:"early_spawn_unit_name,omitempty"`
	EarlySpawnConfidence   string       `json:"early_spawn_confidence,omitempty"`
	DetectionWarnings      []string     `json:"detection_warnings,omitempty"`
}

type PhaseAnchor struct {
	Phase              string  `json:"phase"`
	TimeMS             int     `json:"time_ms"`
	Time               string  `json:"time"`
	KillAnchor         int     `json:"kill_anchor"`
	Signal             string  `json:"signal"`
	UnitID             int     `json:"unit_id,omitempty"`
	UnitName           string  `json:"unit_name,omitempty"`
	ObjectCountDelta   int64   `json:"object_count_delta,omitempty"`
	UnitTypeSumDelta   int64   `json:"unit_type_sum_delta,omitempty"`
	AverageAddedTypeID float64 `json:"average_added_type_id,omitempty"`
	Confidence         string  `json:"confidence"`
	Evidence           string  `json:"evidence"`
}

func BuildPhaseTimeline(path, datPath string) (*PhaseTimelineReport, error) {
	if datPath == "" {
		return nil, fmt.Errorf("phase timeline requires --dat <empires2_x2_p1.dat> for civ threshold resolution")
	}
	rec, err := replay.Open(path)
	if err != nil {
		return nil, err
	}
	sync, err := replay.BuildSyncStream(path, replay.SyncOptions{ChecksumsOnly: true})
	if err != nil {
		return nil, err
	}
	facts, err := BuildPhaseFacts(path, datPath)
	if err != nil {
		return nil, err
	}
	factsByCiv := map[int]PhaseCivFact{}
	for _, fact := range facts.Civs {
		factsByCiv[fact.CivID] = fact
	}
	deltasByPlayer := map[int][]replay.SyncStateDelta{}
	for _, delta := range sync.StateDeltas {
		if delta.PlayerID > 0 {
			deltasByPlayer[delta.PlayerID] = append(deltasByPlayer[delta.PlayerID], delta)
		}
	}
	report := &PhaseTimelineReport{
		Path:         path,
		DatPath:      datPath,
		Method:       "cba_requiem_checksum_phase_shadow_detector",
		Verification: "behavior_inferred_from_checksum_object_type_sums_plus_structure_verified_phase_thresholds_not_direct_age_state",
		Summary: PhaseTimelineSummary{
			PhaseFactsCivs: len(facts.Civs),
			SyncDeltas:     len(sync.StateDeltas),
			DurationMS:     sync.Summary.DurationMS,
			Duration:       sync.Summary.Duration,
			Confidence:     "phase_anchors_are_inferred_shadows_with_per_player_confidence",
		},
		Warnings: append([]string{}, facts.Warnings...),
	}
	for _, player := range rec.Players {
		if !player.Active || player.Number <= 0 || player.Number > 8 {
			continue
		}
		row := PhaseTimelinePlayer{
			PlayerID:    player.Number,
			PlayerLabel: playerLabel(player.Number),
			PlayerName:  player.Name,
			CivID:       player.Civ,
			CivName:     replay.CivDisplayName(player.Civ),
		}
		fact, hasFact := factsByCiv[player.Civ]
		if hasFact {
			row.CastleThresholdKills = fact.CastleThreshold
			row.ImperialThresholdKills = fact.ImperialThreshold
			row.RazesToVillager = fact.RazesToVillager
		} else {
			row.DetectionWarnings = append(row.DetectionWarnings, "no phase-facts row for player civilization")
		}
		deltas := deltasByPlayer[player.Number]
		baseID, baseConfidence := detectEarlySpawnUnit(deltas)
		row.EarlySpawnUnitID = baseID
		row.EarlySpawnUnitName = replay.UnitDisplayName(baseID)
		row.EarlySpawnConfidence = baseConfidence
		if hasFact && fact.CastleThreshold > 0 {
			row.Castle = detectCastleAnchor(deltas, fact.CastleThreshold)
			if row.Castle != nil {
				report.Summary.CastleObserved++
			}
		}
		if hasFact && fact.ImperialThreshold > 0 && baseID > 0 {
			afterMS := 0
			if row.Castle != nil {
				afterMS = row.Castle.TimeMS
			}
			row.Imperial = detectImperialAnchor(deltas, fact.ImperialThreshold, baseID, afterMS)
			if row.Imperial != nil {
				report.Summary.ImperialObserved++
			}
		}
		if row.Castle == nil && hasFact && fact.CastleThreshold > 0 {
			row.DetectionWarnings = append(row.DetectionWarnings, "Castle threshold known but no checksum eco/villager shadow was detected")
		}
		if row.Imperial == nil && hasFact && fact.ImperialThreshold > 0 {
			row.DetectionWarnings = append(row.DetectionWarnings, "Imperial threshold known but no clean base-to-elite spawn-type switch was detected")
		}
		report.Players = append(report.Players, row)
	}
	sort.Slice(report.Players, func(i, j int) bool { return report.Players[i].PlayerID < report.Players[j].PlayerID })
	report.Summary.Players = len(report.Players)
	return report, nil
}

func detectCastleAnchor(deltas []replay.SyncStateDelta, threshold int) *PhaseAnchor {
	for _, delta := range deltas {
		avg, ok := positiveAverageType(delta)
		if !ok {
			continue
		}
		if isEcoUnit(delta.UnitID) {
			return phaseAnchor("castle", threshold, delta, avg, "exact_eco_unit_added", "high_confidence_exact_single_object_eco_delta")
		}
		if delta.ObjectCountDelta >= 4 && avg >= 50 && avg <= 350 {
			return phaseAnchor("castle", threshold, delta, avg, "low_type_id_positive_object_delta", "medium_confidence_mixed_delta_type_id_shadow")
		}
	}
	return nil
}

func detectImperialAnchor(deltas []replay.SyncStateDelta, threshold, baseUnitID, afterMS int) *PhaseAnchor {
	for _, delta := range deltas {
		if delta.ToTimeMS <= afterMS {
			continue
		}
		unitID, ok := cleanSpawnUnit(delta)
		if !ok || unitID == baseUnitID || unitID <= 0 || !isLikelyEliteSpawnUnit(unitID) {
			continue
		}
		avg := float64(unitID)
		return phaseAnchor("imperial", threshold, delta, avg, "clean_spawn_type_switch", "medium_confidence_clean_spawn_like_delta")
	}
	return nil
}

func detectEarlySpawnUnit(deltas []replay.SyncStateDelta) (int, string) {
	counts := map[int]int{}
	for _, delta := range deltas {
		unitID, ok := cleanSpawnUnit(delta)
		if !ok || isEcoUnit(unitID) || unitID <= 350 {
			continue
		}
		counts[unitID]++
		if len(counts) >= 4 {
			break
		}
	}
	if len(counts) == 0 {
		return 0, "not_observed"
	}
	bestID, bestCount := 0, 0
	for id, count := range counts {
		if count > bestCount || (count == bestCount && id < bestID) {
			bestID, bestCount = id, count
		}
	}
	if bestCount >= 2 {
		return bestID, "high_confidence_repeated_clean_spawn_delta"
	}
	return bestID, "medium_confidence_single_clean_spawn_delta"
}

func cleanSpawnUnit(delta replay.SyncStateDelta) (int, bool) {
	if delta.ObjectCountDelta < 4 || delta.UnitTypeSumDelta <= 0 {
		return 0, false
	}
	if delta.UnitTypeSumDelta%delta.ObjectCountDelta != 0 {
		return 0, false
	}
	unitID := int(delta.UnitTypeSumDelta / delta.ObjectCountDelta)
	if unitID <= 0 {
		return 0, false
	}
	return unitID, true
}

func positiveAverageType(delta replay.SyncStateDelta) (float64, bool) {
	if delta.ObjectCountDelta <= 0 || delta.UnitTypeSumDelta <= 0 {
		return 0, false
	}
	return float64(delta.UnitTypeSumDelta) / float64(delta.ObjectCountDelta), true
}

func phaseAnchor(phase string, threshold int, delta replay.SyncStateDelta, avg float64, signal, confidence string) *PhaseAnchor {
	unitID := delta.UnitID
	if unitID == 0 && math.Abs(avg-math.Round(avg)) < 0.0001 {
		unitID = int(math.Round(avg))
	}
	return &PhaseAnchor{
		Phase:              phase,
		TimeMS:             delta.ToTimeMS,
		Time:               delta.ToTime,
		KillAnchor:         threshold,
		Signal:             signal,
		UnitID:             unitID,
		UnitName:           replay.UnitDisplayName(unitID),
		ObjectCountDelta:   delta.ObjectCountDelta,
		UnitTypeSumDelta:   delta.UnitTypeSumDelta,
		AverageAddedTypeID: avg,
		Confidence:         confidence,
		Evidence:           fmt.Sprintf("checksum word_2 delta %d over word_6 delta %d", delta.UnitTypeSumDelta, delta.ObjectCountDelta),
	}
}

func isEcoUnit(unitID int) bool {
	switch unitID {
	case 13, 56, 83, 128, 204, 293:
		return true
	default:
		return false
	}
}

func isLikelyEliteSpawnUnit(unitID int) bool {
	if unitID >= 1000 {
		return true
	}
	name := replay.UnitDisplayName(unitID)
	return strings.Contains(strings.ToLower(name), "elite")
}
