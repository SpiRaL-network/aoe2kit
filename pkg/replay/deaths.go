package replay

import (
	"encoding/json"
	"fmt"
	"path/filepath"
)

type DeathReportOptions struct {
	Limit int
}

type DeathReport struct {
	Path         string             `json:"path,omitempty"`
	Method       string             `json:"method"`
	Verification string             `json:"verification"`
	Summary      DeathReportSummary `json:"summary"`
	Events       []DeathReportEvent `json:"events,omitempty"`
	Ambiguous    []DeathReportEvent `json:"ambiguous_replacements,omitempty"`
	Warnings     []string           `json:"warnings,omitempty"`
	CorpseTable  CorpseTableSummary `json:"corpse_table"`
}

type DeathReportSummary struct {
	ChecksumSamples         int `json:"checksum_samples"`
	ReplacementEvents       int `json:"replacement_events"`
	ShownReplacementEvents  int `json:"shown_replacement_events"`
	AmbiguousFlatTransforms int `json:"ambiguous_flat_transforms"`
	ShownAmbiguous          int `json:"shown_ambiguous"`
}

type CorpseTableSummary struct {
	Rows           int    `json:"rows"`
	EngineVerified int    `json:"engine_verified_rows"`
	HeuristicRows  int    `json:"heuristic_rows"`
	Method         string `json:"method"`
}

type DeathReportEvent struct {
	PlayerID         int    `json:"player_id"`
	PlayerLabel      string `json:"player_label"`
	FromTimeMS       int    `json:"from_time_ms"`
	FromTime         string `json:"from_time"`
	ToTimeMS         int    `json:"to_time_ms"`
	ToTime           string `json:"to_time"`
	LiveUnitID       int    `json:"live_unit_id,omitempty"`
	LiveUnitName     string `json:"live_unit_name,omitempty"`
	CorpseUnitID     int    `json:"corpse_unit_id,omitempty"`
	CorpseUnitName   string `json:"corpse_unit_name,omitempty"`
	UnitTypeSumDelta int64  `json:"unit_type_sum_delta"`
	ObjectIDSumDelta int64  `json:"object_id_sum_delta,omitempty"`
	PositionSumDelta int64  `json:"position_sum_delta,omitempty"`
	Word3Delta       int64  `json:"word_3_delta,omitempty"`
	Word4Delta       int64  `json:"word_4_delta,omitempty"`
	FreshCorpseCarry int    `json:"fresh_corpse_carry,omitempty"`
	EstimatedDeathMS int    `json:"estimated_death_time_ms,omitempty"`
	EstimatedDeath   string `json:"estimated_death_time,omitempty"`
	DeathTimeMethod  string `json:"death_time_method,omitempty"`
	ObjectCountDelta int64  `json:"object_count_delta"`
	Kind             string `json:"kind"`
	Confidence       string `json:"confidence"`
}

func BuildDeathReport(path string, opts DeathReportOptions) (*DeathReport, error) {
	cache, err := OpenReplayCache(path)
	if err != nil {
		return nil, err
	}
	sync, dir, err := cache.syncFiles()
	if err != nil {
		return nil, err
	}
	table, err := LoadDefaultCorpseTable()
	if err != nil {
		return nil, err
	}
	report := &DeathReport{
		Path:         path,
		Method:       "body_op2_checksum_flat_count_type_replacement",
		Verification: "structure_verified_replacement_events_not_kill_attribution",
		Warnings:     append([]string{}, sync.Warnings...),
		CorpseTable:  summarizeCorpseTable(table),
	}
	report.Summary.ChecksumSamples = sync.Summary.ChecksumDE
	err = walkJSONRows(filepath.Join(dir, "lifecycle.jsonl"), func(raw []byte) error {
		var delta SyncStateDelta
		if err := json.Unmarshal(raw, &delta); err != nil {
			return err
		}
		if (len(report.Events)+len(report.Ambiguous))*1024 > maxReportBytes {
			return fmt.Errorf("death report exceeds memory budget; use --limit")
		}
		event := DeathReportEvent{
			PlayerID:         delta.PlayerID,
			PlayerLabel:      delta.PlayerLabel,
			FromTimeMS:       delta.FromTimeMS,
			FromTime:         delta.FromTime,
			ToTimeMS:         delta.ToTimeMS,
			ToTime:           delta.ToTime,
			UnitTypeSumDelta: delta.UnitTypeSumDelta,
			ObjectIDSumDelta: delta.ObjectIDSumDelta,
			PositionSumDelta: delta.PositionSumDelta,
			Word3Delta:       delta.Word3Delta,
			Word4Delta:       delta.Word4Delta,
			ObjectCountDelta: delta.ObjectCountDelta,
			Kind:             delta.Kind,
			Confidence:       delta.Confidence,
		}
		if delta.Kind == "live_unit_replaced_by_corpse" {
			event.LiveUnitID = delta.UnitID
			event.LiveUnitName = delta.UnitName
			event.CorpseUnitID = delta.ReplacementUnitID
			event.CorpseUnitName = delta.ReplacementUnitName
			if row, ok := table.Replacement(event.LiveUnitID, event.CorpseUnitID); ok {
				annotateDeathTiming(&event, row)
			}
			report.Summary.ReplacementEvents++
			if opts.Limit == 0 || len(report.Events) < opts.Limit {
				report.Events = append(report.Events, event)
			}
			return nil
		}
		if delta.ObjectCountDelta == 0 && delta.UnitTypeSumDelta != 0 {
			event.Kind = "flat_count_type_transform_ambiguous"
			event.Confidence = "ambiguous_sum_not_decomposed"
			report.Summary.AmbiguousFlatTransforms++
			if opts.Limit == 0 || len(report.Ambiguous) < opts.Limit {
				report.Ambiguous = append(report.Ambiguous, event)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	report.Summary.ShownReplacementEvents = len(report.Events)
	report.Summary.ShownAmbiguous = len(report.Ambiguous)
	return report, nil
}

func annotateDeathTiming(event *DeathReportEvent, row CorpseTableRow) {
	if row.ObservedFreshCarryDelta == nil {
		return
	}
	fresh := *row.ObservedFreshCarryDelta
	event.FreshCorpseCarry = fresh
	if event.Word4Delta == int64(fresh) {
		event.EstimatedDeathMS = event.ToTimeMS
		event.EstimatedDeath = event.ToTime
		event.DeathTimeMethod = "fresh_corpse_carry_exact_at_checksum_sample_within_interval"
		return
	}
	if row.ObservedDecayPerSecond == nil || *row.ObservedDecayPerSecond <= 0 {
		return
	}
	decayed := float64(fresh) - float64(event.Word4Delta)
	if decayed < 0 {
		return
	}
	lagMS := int(decayed * 1000 / *row.ObservedDecayPerSecond)
	event.EstimatedDeathMS = event.ToTimeMS - lagMS
	if event.EstimatedDeathMS < event.FromTimeMS {
		event.EstimatedDeathMS = event.FromTimeMS
	}
	event.EstimatedDeath = FormatTime(event.EstimatedDeathMS)
	event.DeathTimeMethod = "fresh_corpse_carry_minus_observed_word4_delta_using_decay_rate_rough"
}

func summarizeCorpseTable(table CorpseTable) CorpseTableSummary {
	summary := CorpseTableSummary{Rows: len(table.Rows), Method: table.Method}
	for _, row := range table.Rows {
		if row.Confidence == "engine_verified" {
			summary.EngineVerified++
		} else {
			summary.HeuristicRows++
		}
	}
	return summary
}
