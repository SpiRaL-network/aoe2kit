package replay

import (
	"fmt"

	"aoe2kit/pkg/datfile"
)

type EffectiveDataModCheckOptions struct {
	DatPath        string
	BaselineReplay string
	KnownTemplate  string
	TailOffset     int
	Count          int
	Limit          int
}

type EffectiveDataModCheckReport struct {
	Path              string                         `json:"path"`
	BaselineReplay    string                         `json:"baseline_replay,omitempty"`
	DatPath           string                         `json:"dat_path,omitempty"`
	Method            string                         `json:"method"`
	Verification      string                         `json:"verification"`
	Verdict           string                         `json:"verdict"`
	DataSet           DataSetIdentity                `json:"data_set_identity"`
	TemplateCheck     EffectiveTemplateCheck         `json:"template_check"`
	TargetTemplate    EffectiveDataTemplate          `json:"target_template"`
	BaselineTemplate  *EffectiveDataTemplate         `json:"baseline_template,omitempty"`
	TailComparison    EffectiveTailComparisonSummary `json:"tail_comparison,omitempty"`
	AvailabilityArray EffectiveArraySummary          `json:"availability_array"`
	AvailabilityDiff  EffectiveAvailabilityDiff      `json:"availability_diff,omitempty"`
	Warnings          []string                       `json:"warnings,omitempty"`
}

type EffectiveTailComparisonSummary struct {
	ComparedPlayers  int                       `json:"compared_players"`
	EqualPlayers     int                       `json:"equal_players"`
	DifferentPlayers int                       `json:"different_players"`
	MissingPlayers   int                       `json:"missing_players"`
	ComparedBytes    int                       `json:"compared_bytes"`
	ChangedBytes     int                       `json:"changed_bytes"`
	DiffRuns         int                       `json:"diff_runs"`
	Players          []EffectiveTailPlayerDiff `json:"players,omitempty"`
}

type EffectiveTailPlayerDiff struct {
	PlayerIndex    int     `json:"player_index"`
	Label          string  `json:"label,omitempty"`
	TargetBytes    int     `json:"target_bytes"`
	BaselineBytes  int     `json:"baseline_bytes"`
	SHA256         string  `json:"sha256"`
	BaselineSHA256 string  `json:"baseline_sha256"`
	Equal          bool    `json:"equal"`
	ComparedBytes  int     `json:"compared_bytes"`
	ChangedBytes   int     `json:"changed_bytes"`
	IdenticalRatio float64 `json:"identical_ratio"`
	DiffRuns       int     `json:"diff_runs"`
}

type EffectiveAvailabilityDiff struct {
	Compared          bool                           `json:"compared"`
	ReferencePlayer   int                            `json:"reference_player,omitempty"`
	BaselinePlayer    int                            `json:"baseline_player,omitempty"`
	ComparedEntries   int                            `json:"compared_entries,omitempty"`
	ChangedEntries    int                            `json:"changed_entries,omitempty"`
	AvailabilityFlips int                            `json:"availability_flips,omitempty"`
	TargetAvailable   int                            `json:"target_available,omitempty"`
	BaselineAvailable int                            `json:"baseline_available,omitempty"`
	Shown             int                            `json:"shown,omitempty"`
	Entries           []EffectiveAvailabilityDiffRow `json:"entries,omitempty"`
}

type EffectiveAvailabilityDiffRow struct {
	Index            int    `json:"index"`
	TailOffset       int    `json:"tail_offset"`
	TargetValue      uint16 `json:"target_value"`
	BaselineValue    uint16 `json:"baseline_value"`
	TargetMeaning    string `json:"target_meaning"`
	BaselineMeaning  string `json:"baseline_meaning"`
	AvailabilityFlip bool   `json:"availability_flip"`
	UnitSlot         int    `json:"unit_slot"`
	UnitID           int16  `json:"unit_id,omitempty"`
	UnitName         string `json:"unit_name,omitempty"`
	DatPresent       bool   `json:"dat_present,omitempty"`
	Confidence       string `json:"confidence"`
}

type effectiveDataContext struct {
	rec   *File
	tails []effectiveTailBytes
	ref   effectiveTailBytes
}

func BuildEffectiveDataModCheck(path string, opts EffectiveDataModCheckOptions) (*EffectiveDataModCheckReport, error) {
	if opts.TailOffset == 0 {
		opts.TailOffset = DefaultEffectiveAvailabilityOffset
	}
	if opts.Count == 0 {
		opts.Count = DefaultEffectiveAvailabilityCount
	}
	if opts.Limit == 0 {
		opts.Limit = 40
	}
	if opts.TailOffset < 0 || opts.Count < 0 || opts.Limit < 0 {
		return nil, fmt.Errorf("tail offset, count, and limit must be non-negative")
	}
	target, err := loadEffectiveDataContext(path)
	if err != nil {
		return nil, err
	}
	if len(target.tails) == 0 {
		return nil, fmt.Errorf("no target effective data tails")
	}
	var dat *datfile.Index
	if opts.DatPath != "" {
		dat, err = datfile.Open(opts.DatPath)
		if err != nil {
			return nil, err
		}
	}
	report := &EffectiveDataModCheckReport{
		Path:         path,
		DatPath:      opts.DatPath,
		Method:       "v68_effective_gamedata_template_and_baseline_diff",
		Verification: "structure_verified_effective_gamedata_tail_diff_not_engine_verified",
		DataSet:      target.rec.DataSet,
		TargetTemplate: EffectiveDataTemplate{
			ReferencePlayer: target.ref.playerIndex,
			ReferenceLabel:  target.ref.label,
			TailStart:       target.ref.start,
			TailEnd:         target.ref.end,
			TailBytes:       len(target.ref.bytes),
			SHA256:          sha256Hex(target.ref.bytes),
		},
		AvailabilityArray: EffectiveArraySummary{
			TailOffset: opts.TailOffset,
			Count:      opts.Count,
			Bytes:      opts.Count * 2,
			AbsStart:   target.ref.start + opts.TailOffset,
			AbsEnd:     target.ref.start + opts.TailOffset + opts.Count*2,
			Kind:       "u16_availability_array",
			Confidence: "offset_and_length_measured_from_replay_tail_diff_analysis",
		},
	}
	report.TemplateCheck = buildEffectiveTemplateCheck(report.TargetTemplate.SHA256, opts.KnownTemplate)
	report.Verdict = verdictFromTemplateCheck(report.TemplateCheck)
	if opts.BaselineReplay == "" {
		report.Warnings = append(report.Warnings, "no --baseline-replay supplied; verdict is a build-sensitive template-anchor check, not a definitive data-mod/no-data-mod proof")
		return report, nil
	}
	baseline, err := loadEffectiveDataContext(opts.BaselineReplay)
	if err != nil {
		return nil, fmt.Errorf("baseline replay: %w", err)
	}
	report.BaselineReplay = opts.BaselineReplay
	baselineTemplate := EffectiveDataTemplate{
		ReferencePlayer: baseline.ref.playerIndex,
		ReferenceLabel:  baseline.ref.label,
		TailStart:       baseline.ref.start,
		TailEnd:         baseline.ref.end,
		TailBytes:       len(baseline.ref.bytes),
		SHA256:          sha256Hex(baseline.ref.bytes),
	}
	report.BaselineTemplate = &baselineTemplate
	report.TailComparison = compareEffectiveTails(target.tails, baseline.tails)
	report.AvailabilityDiff = compareEffectiveAvailability(target, baseline, dat, opts)
	if report.TailComparison.DifferentPlayers == 0 && report.TailComparison.MissingPlayers == 0 && report.AvailabilityDiff.ChangedEntries == 0 {
		report.Verdict = "matches_baseline_replay_effective_data"
	} else {
		report.Verdict = "differs_from_baseline_replay_effective_data"
	}
	return report, nil
}

func loadEffectiveDataContext(path string) (effectiveDataContext, error) {
	data, err := ReadRecordBytes(path)
	if err != nil {
		return effectiveDataContext{}, err
	}
	rec, err := Parse(data)
	if err != nil {
		return effectiveDataContext{}, err
	}
	coverage, err := BuildCoverage(path)
	if err != nil {
		return effectiveDataContext{}, err
	}
	if coverage.HeaderSpine == nil {
		return effectiveDataContext{}, fmt.Errorf("header spine unavailable")
	}
	tails := effectiveDataTailsFromSpine(rec.HeaderBytes(), coverage.HeaderSpine)
	if len(tails) == 0 {
		return effectiveDataContext{}, fmt.Errorf("no non-Gaia effective game-data tails found")
	}
	return effectiveDataContext{rec: rec, tails: tails, ref: tails[0]}, nil
}

func verdictFromTemplateCheck(check EffectiveTemplateCheck) string {
	switch check.Status {
	case "matches_known_vanilla_template":
		return "matches_observed_anchor_template"
	case "matches_user_template":
		return "matches_user_template"
	case "differs_from_user_template":
		return "differs_from_user_template_baseline_needed"
	default:
		return "differs_from_observed_anchor_template_baseline_needed"
	}
}

func compareEffectiveTails(target []effectiveTailBytes, baseline []effectiveTailBytes) EffectiveTailComparisonSummary {
	byPlayer := map[int]effectiveTailBytes{}
	for _, tail := range baseline {
		byPlayer[tail.playerIndex] = tail
	}
	var summary EffectiveTailComparisonSummary
	for _, tail := range target {
		base, ok := byPlayer[tail.playerIndex]
		if !ok {
			summary.MissingPlayers++
			summary.Players = append(summary.Players, EffectiveTailPlayerDiff{
				PlayerIndex: tail.playerIndex,
				Label:       tail.label,
				TargetBytes: len(tail.bytes),
				SHA256:      sha256Hex(tail.bytes),
			})
			continue
		}
		diff := compareEffectiveTailPlayer(tail, base)
		summary.ComparedPlayers++
		summary.ComparedBytes += diff.ComparedBytes
		summary.ChangedBytes += diff.ChangedBytes
		summary.DiffRuns += diff.DiffRuns
		if diff.Equal {
			summary.EqualPlayers++
		} else {
			summary.DifferentPlayers++
		}
		summary.Players = append(summary.Players, diff)
	}
	return summary
}

func compareEffectiveTailPlayer(target effectiveTailBytes, baseline effectiveTailBytes) EffectiveTailPlayerDiff {
	compared := len(target.bytes)
	if len(baseline.bytes) < compared {
		compared = len(baseline.bytes)
	}
	changed := countByteDiff(target.bytes, baseline.bytes, compared)
	if len(target.bytes) != len(baseline.bytes) {
		if len(target.bytes) > len(baseline.bytes) {
			changed += len(target.bytes) - len(baseline.bytes)
		} else {
			changed += len(baseline.bytes) - len(target.bytes)
		}
	}
	ratio := 0.0
	if compared > 0 {
		ratio = 100 * float64(compared-countByteDiff(target.bytes, baseline.bytes, compared)) / float64(compared)
	}
	diffRuns := effectiveGameDataDiffRuns(baseline.bytes, target.bytes)
	return EffectiveTailPlayerDiff{
		PlayerIndex:    target.playerIndex,
		Label:          target.label,
		TargetBytes:    len(target.bytes),
		BaselineBytes:  len(baseline.bytes),
		SHA256:         sha256Hex(target.bytes),
		BaselineSHA256: sha256Hex(baseline.bytes),
		Equal:          len(target.bytes) == len(baseline.bytes) && changed == 0,
		ComparedBytes:  compared,
		ChangedBytes:   changed,
		IdenticalRatio: round2(ratio),
		DiffRuns:       len(diffRuns),
	}
}

func countByteDiff(a []byte, b []byte, limit int) int {
	changed := 0
	for i := 0; i < limit; i++ {
		if a[i] != b[i] {
			changed++
		}
	}
	return changed
}

func compareEffectiveAvailability(target effectiveDataContext, baseline effectiveDataContext, dat *datfile.Index, opts EffectiveDataModCheckOptions) EffectiveAvailabilityDiff {
	diff := EffectiveAvailabilityDiff{
		Compared:        true,
		ReferencePlayer: target.ref.playerIndex,
		BaselinePlayer:  baseline.ref.playerIndex,
	}
	if opts.TailOffset+opts.Count*2 > len(target.ref.bytes) || opts.TailOffset+opts.Count*2 > len(baseline.ref.bytes) {
		diff.Compared = false
		return diff
	}
	targetValues := readU16Array(target.ref.bytes, opts.TailOffset, opts.Count)
	baselineValues := readU16Array(baseline.ref.bytes, opts.TailOffset, opts.Count)
	limit := len(targetValues)
	if len(baselineValues) < limit {
		limit = len(baselineValues)
	}
	targetSlot := playerSlotByNumber(target.rec.Players, target.ref.playerIndex)
	for i := 0; i < limit; i++ {
		targetAvailable := effectiveValueAvailable(targetValues[i])
		baselineAvailable := effectiveValueAvailable(baselineValues[i])
		diff.ComparedEntries++
		if targetAvailable {
			diff.TargetAvailable++
		}
		if baselineAvailable {
			diff.BaselineAvailable++
		}
		if targetValues[i] == baselineValues[i] {
			continue
		}
		diff.ChangedEntries++
		flip := targetAvailable != baselineAvailable
		if flip {
			diff.AvailabilityFlips++
		}
		if opts.Limit > 0 && len(diff.Entries) >= opts.Limit {
			continue
		}
		row := EffectiveAvailabilityDiffRow{
			Index:            i,
			TailOffset:       opts.TailOffset + i*2,
			TargetValue:      targetValues[i],
			BaselineValue:    baselineValues[i],
			TargetMeaning:    availabilityMeaning(targetValues[i]),
			BaselineMeaning:  availabilityMeaning(baselineValues[i]),
			AvailabilityFlip: flip,
			UnitSlot:         i,
			Confidence:       "baseline_replay_effective_availability_array_diff",
		}
		if dat != nil && targetSlot != nil && targetSlot.Civ >= 0 && targetSlot.Civ < len(dat.Civs) {
			civ := dat.Civs[targetSlot.Civ]
			if i < len(civ.Units) {
				unit := civ.Units[i]
				row.UnitID = unit.ID
				row.UnitName = unit.Name
				row.DatPresent = unit.Present
			}
		}
		diff.Entries = append(diff.Entries, row)
	}
	diff.Shown = len(diff.Entries)
	return diff
}

func playerSlotByNumber(players []PlayerSlot, number int) *PlayerSlot {
	for i := range players {
		if players[i].Number == number {
			return &players[i]
		}
	}
	return nil
}
