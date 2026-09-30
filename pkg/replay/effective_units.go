package replay

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"aoe2kit/pkg/datfile"
)

const effectiveUnitHitPointsNameDelta = -120

type EffectiveUnitStatsOptions struct {
	DatPath               string
	Player                int
	Limit                 int
	IncludeUnchanged      bool
	IncludeAllTypes       bool
	IncludeDuplicateNames bool
}

type EffectiveUnitStatsReport struct {
	Path           string                     `json:"path"`
	DatPath        string                     `json:"dat_path,omitempty"`
	Method         string                     `json:"method"`
	Verification   string                     `json:"verification"`
	TemplateCheck  EffectiveTemplateCheck     `json:"template_check"`
	TargetTemplate EffectiveDataTemplate      `json:"target_template"`
	DecodedFields  []string                   `json:"decoded_fields"`
	NotDecoded     []string                   `json:"not_decoded"`
	Summary        EffectiveUnitStatsSummary  `json:"summary"`
	Players        []EffectiveUnitStatsPlayer `json:"players,omitempty"`
	Rows           []EffectiveUnitStatDiffRow `json:"rows,omitempty"`
	Warnings       []string                   `json:"warnings,omitempty"`
}

type EffectiveUnitStatsSummary struct {
	Players              int `json:"players"`
	UnitsConsidered      int `json:"units_considered"`
	Rows                 int `json:"rows"`
	ChangedHitPoints     int `json:"changed_hit_points"`
	ModConfirmedMultiCiv int `json:"mod_confirmed_multi_civ"`
	UnresolvedSingleCiv  int `json:"unresolved_single_civ"`
	DuplicateNameAnchors int `json:"duplicate_name_anchors"`
	MissingNameAnchors   int `json:"missing_name_anchors"`
	InvalidHitPointReads int `json:"invalid_hit_point_reads"`
}

type EffectiveUnitStatsPlayer struct {
	PlayerIndex          int     `json:"player_index"`
	PlayerName           string  `json:"player_name,omitempty"`
	CivID                int     `json:"civ_id,omitempty"`
	CivName              string  `json:"civ_name,omitempty"`
	TailBytes            int     `json:"tail_bytes"`
	IdenticalRatioVsRef  float64 `json:"identical_ratio_vs_reference"`
	DiffRunsVsRef        int     `json:"diff_runs_vs_reference"`
	UnitsConsidered      int     `json:"units_considered"`
	Rows                 int     `json:"rows"`
	ChangedHitPoints     int     `json:"changed_hit_points"`
	DuplicateNameAnchors int     `json:"duplicate_name_anchors"`
	MissingNameAnchors   int     `json:"missing_name_anchors"`
	InvalidHitPointReads int     `json:"invalid_hit_point_reads"`
}

type EffectiveUnitStatDiffRow struct {
	PlayerIndex        int    `json:"player_index"`
	PlayerName         string `json:"player_name,omitempty"`
	CivID              int    `json:"civ_id"`
	CivName            string `json:"civ_name,omitempty"`
	UnitSlot           int    `json:"unit_slot"`
	UnitID             int16  `json:"unit_id"`
	UnitName           string `json:"unit_name"`
	UnitType           int    `json:"unit_type"`
	NameTailOffset     int    `json:"name_tail_offset"`
	NameOccurrences    int    `json:"name_occurrences"`
	HitPointOffset     int    `json:"hit_point_tail_offset"`
	VanillaHitPoints   int16  `json:"vanilla_hit_points"`
	EffectiveHitPoints int16  `json:"effective_hit_points"`
	HitPointDelta      int    `json:"hit_point_delta"`
	Changed            bool   `json:"changed"`
	CivSpan            int    `json:"civ_span,omitempty"`
	Classification     string `json:"classification,omitempty"`
	Confidence         string `json:"confidence"`
}

func BuildEffectiveUnitStats(path string, opts EffectiveUnitStatsOptions) (*EffectiveUnitStatsReport, error) {
	if opts.Limit == 0 {
		opts.Limit = 80
	}
	if opts.Limit < 0 {
		return nil, fmt.Errorf("limit must be non-negative")
	}
	if opts.DatPath == "" {
		return nil, fmt.Errorf("--dat is required for effective unit stat decoding")
	}
	dat, err := datfile.Open(opts.DatPath)
	if err != nil {
		return nil, err
	}
	ctx, err := loadEffectiveDataContext(path)
	if err != nil {
		return nil, err
	}
	report := &EffectiveUnitStatsReport{
		Path:         path,
		DatPath:      opts.DatPath,
		Method:       "v68_effective_gamedata_unit_name_anchor_hitpoint_decode_vs_dat",
		Verification: "structure_verified_name_anchor_hitpoint_decode_not_engine_verified_not_full_dat_diff",
		TargetTemplate: EffectiveDataTemplate{
			ReferencePlayer: ctx.ref.playerIndex,
			ReferenceLabel:  ctx.ref.label,
			TailStart:       ctx.ref.start,
			TailEnd:         ctx.ref.end,
			TailBytes:       len(ctx.ref.bytes),
			SHA256:          sha256Hex(ctx.ref.bytes),
		},
		DecodedFields: []string{"hit_points"},
		NotDecoded: []string{
			"attack",
			"armor",
			"resource_cost",
			"train_time",
			"movement_speed",
			"spawn_wave_unit_count",
			"spawn_wave_seconds",
		},
		Warnings: []string{
			"effective replay values include civ-effective bonuses/technologies as well as data-mod changes; this report compares against the supplied base DAT and does not isolate those causes yet",
			"unit rows are anchored by unit debug-name strings inside the effective game-data tail; duplicate debug names are skipped by default because they are lower-confidence anchors",
			"spawn wave unit counts/seconds were not found in this decoded unit-stat slice; they remain likely trigger-side or in an undecoded effective-data subregion",
		},
	}
	report.TemplateCheck = buildEffectiveTemplateCheck(report.TargetTemplate.SHA256, "")
	slots := map[int]PlayerSlot{}
	for _, player := range ctx.rec.Players {
		if player.Number > 0 {
			slots[player.Number] = player
		}
	}
	for _, tail := range ctx.tails {
		if opts.Player > 0 && tail.playerIndex != opts.Player {
			continue
		}
		slot := slots[tail.playerIndex]
		player := buildEffectiveUnitStatsPlayer(ctx.ref, tail, slot, dat, opts, report)
		report.Players = append(report.Players, player)
	}
	classifyEffectiveUnitRows(report)
	report.Summary.Rows = len(report.Rows)
	report.Summary.Players = len(report.Players)
	return report, nil
}

func buildEffectiveUnitStatsPlayer(ref effectiveTailBytes, tail effectiveTailBytes, slot PlayerSlot, dat *datfile.Index, opts EffectiveUnitStatsOptions, report *EffectiveUnitStatsReport) EffectiveUnitStatsPlayer {
	player := EffectiveUnitStatsPlayer{
		PlayerIndex:   tail.playerIndex,
		PlayerName:    slot.Name,
		CivID:         slot.Civ,
		TailBytes:     len(tail.bytes),
		DiffRunsVsRef: len(effectiveGameDataDiffRuns(ref.bytes, tail.bytes)),
	}
	if slot.Civ >= 0 && slot.Civ < len(dat.Civs) {
		player.CivName = dat.Civs[slot.Civ].Name
	}
	compared := len(ref.bytes)
	if len(tail.bytes) < compared {
		compared = len(tail.bytes)
	}
	if compared > 0 {
		player.IdenticalRatioVsRef = round2(100 * float64(compared-countByteDiff(ref.bytes, tail.bytes, compared)) / float64(compared))
	}
	if slot.Civ < 0 || slot.Civ >= len(dat.Civs) {
		return player
	}
	civ := dat.Civs[slot.Civ]
	for _, unit := range civ.Units {
		if !unit.Present || unit.Name == "" {
			continue
		}
		if !opts.IncludeAllTypes && (unit.Type < 50 || unit.Type >= 80) {
			continue
		}
		player.UnitsConsidered++
		report.Summary.UnitsConsidered++
		nameBytes := []byte(unit.Name)
		nameOff := bytes.Index(tail.bytes, nameBytes)
		if nameOff < 0 {
			player.MissingNameAnchors++
			report.Summary.MissingNameAnchors++
			continue
		}
		occurrences := bytes.Count(tail.bytes, nameBytes)
		if occurrences > 1 {
			player.DuplicateNameAnchors++
			report.Summary.DuplicateNameAnchors++
			if !opts.IncludeDuplicateNames {
				continue
			}
		}
		hpOff := nameOff + effectiveUnitHitPointsNameDelta
		if hpOff < 0 || hpOff+2 > len(tail.bytes) {
			player.InvalidHitPointReads++
			report.Summary.InvalidHitPointReads++
			continue
		}
		effectiveHP := int16(binary.LittleEndian.Uint16(tail.bytes[hpOff:]))
		if effectiveHP <= 0 || effectiveHP > 20000 {
			player.InvalidHitPointReads++
			report.Summary.InvalidHitPointReads++
			continue
		}
		changed := effectiveHP != unit.HitPoints
		if changed {
			player.ChangedHitPoints++
			report.Summary.ChangedHitPoints++
		}
		if !changed && !opts.IncludeUnchanged {
			continue
		}
		if opts.Limit > 0 && player.Rows >= opts.Limit {
			continue
		}
		confidence := "name_anchor_hitpoint_offset_validated_by_replay_tail_cross_unit_probe"
		if occurrences > 1 {
			confidence = "name_anchor_hitpoint_offset_duplicate_name_lower_confidence"
		}
		report.Rows = append(report.Rows, EffectiveUnitStatDiffRow{
			PlayerIndex:        tail.playerIndex,
			PlayerName:         slot.Name,
			CivID:              slot.Civ,
			CivName:            player.CivName,
			UnitSlot:           unit.Index,
			UnitID:             unit.ID,
			UnitName:           unit.Name,
			UnitType:           unit.Type,
			NameTailOffset:     nameOff,
			NameOccurrences:    occurrences,
			HitPointOffset:     hpOff,
			VanillaHitPoints:   unit.HitPoints,
			EffectiveHitPoints: effectiveHP,
			HitPointDelta:      int(effectiveHP) - int(unit.HitPoints),
			Changed:            changed,
			Confidence:         confidence,
		})
		player.Rows++
	}
	return player
}

func classifyEffectiveUnitRows(report *EffectiveUnitStatsReport) {
	type groupKey struct {
		UnitName  string
		Vanilla   int16
		Effective int16
	}
	groups := map[groupKey]map[int]bool{}
	for _, row := range report.Rows {
		if !row.Changed {
			continue
		}
		key := groupKey{
			UnitName:  row.UnitName,
			Vanilla:   row.VanillaHitPoints,
			Effective: row.EffectiveHitPoints,
		}
		if groups[key] == nil {
			groups[key] = map[int]bool{}
		}
		groups[key][row.CivID] = true
	}
	for i := range report.Rows {
		if !report.Rows[i].Changed {
			continue
		}
		key := groupKey{
			UnitName:  report.Rows[i].UnitName,
			Vanilla:   report.Rows[i].VanillaHitPoints,
			Effective: report.Rows[i].EffectiveHitPoints,
		}
		span := len(groups[key])
		report.Rows[i].CivSpan = span
		if span > 1 {
			report.Rows[i].Classification = "mod_confirmed_multi_civ"
			report.Summary.ModConfirmedMultiCiv++
		} else {
			report.Rows[i].Classification = "unresolved_single_civ"
			report.Summary.UnresolvedSingleCiv++
		}
	}
}
