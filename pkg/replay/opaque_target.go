package replay

import (
	"fmt"
	"sort"
)

// OpaqueTarget picks the next best header region to decode, using cross-player
// identity evidence. The once-anonymous large per-player tail is now treated as
// an effective game-data table: a shared template plus sparse civ/player diffs.
// Honesty: identity ratios are measured facts; individual diff-run semantics
// are separate from the structural tail split.

type OpaqueTargetReport struct {
	Path              string            `json:"path"`
	Method            string            `json:"method"`
	Verification      string            `json:"verification"`
	HeaderOpaqueBytes int               `json:"header_opaque_bytes"`
	PlayerTails       []PlayerTailIdent `json:"player_tails,omitempty"`
	CommonPrefixBytes int               `json:"cross_player_common_prefix_bytes"`
	ReducibleEstimate int               `json:"reducible_bytes_if_template_proven"`
	SelectedTarget    string            `json:"selected_target"`
	Rationale         []string          `json:"rationale,omitempty"`
	NextIncrement     string            `json:"next_parser_increment"`
	Warnings          []string          `json:"warnings,omitempty"`
}

type PlayerTailIdent struct {
	Label            string  `json:"label"`
	Start            int     `json:"start"`
	End              int     `json:"end"`
	Bytes            int     `json:"bytes"`
	PrefixMatchVsRef int     `json:"identical_prefix_bytes_vs_reference"`
	IdenticalRatio   float64 `json:"identical_byte_ratio_vs_reference"`
	Reference        bool    `json:"is_reference,omitempty"`
}

func BuildOpaqueTarget(path string) (*OpaqueTargetReport, error) {
	data, err := ReadRecordBytes(path)
	if err != nil {
		return nil, err
	}
	rec, err := Parse(data)
	if err != nil {
		return nil, err
	}
	coverage, err := BuildCoverage(path)
	if err != nil {
		return nil, err
	}
	header := rec.HeaderBytes()
	report := &OpaqueTargetReport{
		Path:              path,
		Method:            "coverage_opaque_regions_plus_cross_player_identity",
		Verification:      "identity_ratios_measured_type_table_naming_is_hypothesis",
		HeaderOpaqueBytes: coverage.Summary.HeaderOpaqueBytes,
	}

	var tails []effectiveTailBytes
	if coverage.HeaderSpine != nil {
		for _, tail := range effectiveDataTailsFromSpine(header, coverage.HeaderSpine) {
			if len(tail.bytes) < 800000 {
				continue
			}
			tails = append(tails, tail)
		}
	}
	sort.Slice(tails, func(i, j int) bool { return tails[i].start < tails[j].start })
	if len(tails) >= 2 {
		ref := tails[0]
		refIdent := PlayerTailIdent{
			Label: fmt.Sprintf("initial_player_%d_effective_gamedata_tail", ref.playerIndex), Start: ref.start, End: ref.end,
			Bytes: len(ref.bytes), Reference: true, IdenticalRatio: 1, PrefixMatchVsRef: len(ref.bytes),
		}
		report.PlayerTails = append(report.PlayerTails, refIdent)
		minPrefix := len(ref.bytes)
		for _, other := range tails[1:] {
			n := len(ref.bytes)
			if len(other.bytes) < n {
				n = len(other.bytes)
			}
			prefix := 0
			for prefix < n && ref.bytes[prefix] == other.bytes[prefix] {
				prefix++
			}
			identical := 0
			for i := 0; i < n; i++ {
				if ref.bytes[i] == other.bytes[i] {
					identical++
				}
			}
			ratio := 0.0
			if n > 0 {
				ratio = float64(identical) / float64(n)
			}
			report.PlayerTails = append(report.PlayerTails, PlayerTailIdent{
				Label: fmt.Sprintf("initial_player_%d_effective_gamedata_tail", other.playerIndex), Start: other.start, End: other.end,
				Bytes: len(other.bytes), PrefixMatchVsRef: prefix, IdenticalRatio: ratio,
			})
			if prefix < minPrefix {
				minPrefix = prefix
			}
		}
		report.CommonPrefixBytes = minPrefix
		report.ReducibleEstimate = minPrefix * (len(tails) - 1)
	}

	meanRatio := 0.0
	if len(report.PlayerTails) > 1 {
		for _, tail := range report.PlayerTails[1:] {
			meanRatio += tail.IdenticalRatio
		}
		meanRatio /= float64(len(report.PlayerTails) - 1)
	}

	switch {
	case meanRatio > 0.95 && report.CommonPrefixBytes <= 100000:
		report.SelectedTarget = "per_player_tail_sparse_differences"
		refBytes := report.PlayerTails[0].Bytes
		diffEstimate := int(float64(refBytes) * (1 - meanRatio))
		report.Rationale = append(report.Rationale,
			fmt.Sprintf("player tails are %.2f%% byte-identical at aligned offsets vs the reference (mean over %d tails)", meanRatio*100, len(report.PlayerTails)-1),
			fmt.Sprintf("so each ~%d-byte tail carries only ~%d truly player-specific bytes; the rest is a shared template", refBytes, diffEstimate),
			"coverage now emits the measured shared template/common runs plus bounded civ-specific diff runs")
		report.NextIncrement = "map more effective-data arrays and add DE tech-table parsing to cross-reference tech indices, not just unit-slot availability"
	case report.CommonPrefixBytes > 100000:
		report.SelectedTarget = "per_player_object_tail_shared_template"
		report.Rationale = append(report.Rationale,
			fmt.Sprintf("all %d non-Gaia player tails share a byte-identical prefix of %d bytes vs the reference tail", len(tails), report.CommonPrefixBytes),
			"embedded unit-type strings (e.g. graphic names) support a repeated per-player type/definition table hypothesis",
			fmt.Sprintf("proving the template bounds ~%d bytes as one shared structure instead of %d anonymous copies", report.ReducibleEstimate, len(tails)-1))
		report.NextIncrement = "emit coverage subregions splitting each player tail into template_common (byte-identical to reference, measured) vs player_specific_remainder; then parse the template head fields once"
	case len(tails) >= 2:
		report.SelectedTarget = "v68_tail_before_trigger_graph"
		report.Rationale = append(report.Rationale, "player tails do not share a large identical prefix; the bounded pre-trigger tail is the next structurally plausible target")
		report.NextIncrement = "sequential parse of the v68 pre-trigger tail guided by its string islands"
	default:
		report.SelectedTarget = "insufficient_opaque_structure"
		report.Rationale = append(report.Rationale, "fewer than two per-player object tails found; nothing to compare")
		report.NextIncrement = "collect more fixtures"
	}
	return report, nil
}
