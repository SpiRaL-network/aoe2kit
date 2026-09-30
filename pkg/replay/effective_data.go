package replay

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"sort"

	"aoe2kit/pkg/datfile"
)

const (
	DefaultEffectiveAvailabilityOffset = 879618
	DefaultEffectiveAvailabilityCount  = 864

	ObservedVanillaEffectiveTemplateName   = "observed_de_v68_vanilla_effective_template_2026_07_21"
	ObservedVanillaEffectiveTemplateSHA256 = "cba7550a169f61ba9bafd751df2fec8a7b02fabef5acaa4245e07f2fed86b9d0"
)

type EffectiveDataOptions struct {
	DatPath       string
	Player        int
	TailOffset    int
	Count         int
	Limit         int
	IncludeAll    bool
	KnownTemplate string
}

type EffectiveDataReport struct {
	Path              string                 `json:"path"`
	DatPath           string                 `json:"dat_path,omitempty"`
	Method            string                 `json:"method"`
	Verification      string                 `json:"verification"`
	Template          EffectiveDataTemplate  `json:"template"`
	TemplateCheck     EffectiveTemplateCheck `json:"template_check"`
	AvailabilityArray EffectiveArraySummary  `json:"availability_array"`
	Players           []EffectivePlayerTable `json:"players,omitempty"`
	Warnings          []string               `json:"warnings,omitempty"`
}

type EffectiveDataTemplate struct {
	ReferencePlayer       int    `json:"reference_player"`
	ReferenceLabel        string `json:"reference_label,omitempty"`
	TailStart             int    `json:"tail_start"`
	TailEnd               int    `json:"tail_end"`
	TailBytes             int    `json:"tail_bytes"`
	SHA256                string `json:"sha256"`
	KnownTemplateStatus   string `json:"known_template_status,omitempty"`
	KnownTemplateExpected string `json:"known_template_expected,omitempty"`
}

type EffectiveTemplateCheck struct {
	Status         string `json:"status"`
	BaselineName   string `json:"baseline_name"`
	ExpectedSHA256 string `json:"expected_sha256"`
	ObservedSHA256 string `json:"observed_sha256"`
	Confidence     string `json:"confidence"`
	Note           string `json:"note,omitempty"`
}

type EffectiveArraySummary struct {
	TailOffset int    `json:"tail_offset"`
	Count      int    `json:"count"`
	Bytes      int    `json:"bytes"`
	AbsStart   int    `json:"abs_start"`
	AbsEnd     int    `json:"abs_end"`
	Kind       string `json:"kind"`
	Confidence string `json:"confidence"`
}

type EffectivePlayerTable struct {
	PlayerIndex    int                   `json:"player_index"`
	PlayerNumber   int                   `json:"player_number,omitempty"`
	PlayerName     string                `json:"player_name,omitempty"`
	CivID          int                   `json:"civ_id,omitempty"`
	CivName        string                `json:"civ_name,omitempty"`
	TailStart      int                   `json:"tail_start"`
	TailEnd        int                   `json:"tail_end"`
	TailBytes      int                   `json:"tail_bytes"`
	IdenticalRatio float64               `json:"identical_ratio_vs_template"`
	DiffRuns       int                   `json:"diff_runs_vs_template"`
	EntryCount     int                   `json:"entry_count"`
	ChangedEntries int                   `json:"changed_entries_vs_template"`
	Available      int                   `json:"available_entries"`
	Unavailable    int                   `json:"unavailable_entries"`
	Level0         int                   `json:"level_0_entries"`
	Level1         int                   `json:"level_1_entries"`
	Level2         int                   `json:"level_2_entries"`
	Level3         int                   `json:"level_3_entries"`
	OtherValues    int                   `json:"other_value_entries"`
	DatNamed       int                   `json:"dat_named_entries,omitempty"`
	Shown          int                   `json:"shown"`
	Entries        []EffectiveArrayEntry `json:"entries,omitempty"`
	Warnings       []string              `json:"warnings,omitempty"`
}

type EffectiveArrayEntry struct {
	Index           int    `json:"index"`
	TailOffset      int    `json:"tail_offset"`
	AbsOffset       int    `json:"abs_offset"`
	Value           uint16 `json:"value"`
	ReferenceValue  uint16 `json:"reference_value"`
	Changed         bool   `json:"changed_from_reference"`
	ReplayAvailable bool   `json:"replay_available"`
	Meaning         string `json:"meaning"`
	UnitSlot        int    `json:"unit_slot"`
	UnitID          int16  `json:"unit_id,omitempty"`
	UnitName        string `json:"unit_name,omitempty"`
	DatPresent      bool   `json:"dat_present"`
	DatEnabled      *uint8 `json:"dat_enabled,omitempty"`
	Confidence      string `json:"confidence"`
}

func BuildEffectiveData(path string, opts EffectiveDataOptions) (*EffectiveDataReport, error) {
	if opts.TailOffset == 0 {
		opts.TailOffset = DefaultEffectiveAvailabilityOffset
	}
	if opts.Count == 0 {
		opts.Count = DefaultEffectiveAvailabilityCount
	}
	if opts.Count < 0 || opts.TailOffset < 0 {
		return nil, fmt.Errorf("tail offset and count must be non-negative")
	}
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
	if coverage.HeaderSpine == nil {
		return nil, fmt.Errorf("header spine unavailable")
	}
	header := rec.HeaderBytes()
	var dat *datfile.Index
	if opts.DatPath != "" {
		dat, err = datfile.Open(opts.DatPath)
		if err != nil {
			return nil, err
		}
	}
	tails := effectiveDataTailsFromSpine(header, coverage.HeaderSpine)
	if len(tails) == 0 {
		return nil, fmt.Errorf("no non-Gaia effective game-data tails found")
	}
	ref := tails[0]
	report := &EffectiveDataReport{
		Path:         path,
		DatPath:      opts.DatPath,
		Method:       "v68_initial_player_effective_gamedata_tail_u16_availability_xref",
		Verification: "structure_verified_array_index_to_dat_unit_slot_crossref_template_hash_build_sensitive",
		Template: EffectiveDataTemplate{
			ReferencePlayer: ref.playerIndex,
			ReferenceLabel:  ref.label,
			TailStart:       ref.start,
			TailEnd:         ref.end,
			TailBytes:       len(ref.bytes),
			SHA256:          sha256Hex(ref.bytes),
		},
		AvailabilityArray: EffectiveArraySummary{
			TailOffset: opts.TailOffset,
			Count:      opts.Count,
			Bytes:      opts.Count * 2,
			AbsStart:   ref.start + opts.TailOffset,
			AbsEnd:     ref.start + opts.TailOffset + opts.Count*2,
			Kind:       "u16_availability_array",
			Confidence: "offset_and_length_measured_from_replay_tail_diff_analysis",
		},
	}
	report.TemplateCheck = buildEffectiveTemplateCheck(report.Template.SHA256, opts.KnownTemplate)
	if opts.KnownTemplate != "" {
		report.Template.KnownTemplateExpected = opts.KnownTemplate
		if report.Template.SHA256 == opts.KnownTemplate {
			report.Template.KnownTemplateStatus = "match"
		} else {
			report.Template.KnownTemplateStatus = "mismatch"
		}
	}
	playerSlots := map[int]PlayerSlot{}
	for _, player := range rec.Players {
		if player.Number > 0 {
			playerSlots[player.Number] = player
		}
	}
	for _, tail := range tails {
		if opts.Player > 0 && tail.playerIndex != opts.Player {
			continue
		}
		table := buildEffectivePlayerTable(ref, tail, playerSlots[tail.playerIndex], dat, opts)
		report.Players = append(report.Players, table)
	}
	if dat == nil {
		report.Warnings = append(report.Warnings, "no --dat supplied; entries include array values but not DAT unit names")
	}
	report.Warnings = append(report.Warnings, "effective tech-tree means replay effective unit-slot availability; replay effective technology/research arrays are not mapped to the DAT tech table yet")
	return report, nil
}

func buildEffectiveTemplateCheck(observed string, expected string) EffectiveTemplateCheck {
	check := EffectiveTemplateCheck{
		Status:         "matches_known_vanilla_template",
		BaselineName:   ObservedVanillaEffectiveTemplateName,
		ExpectedSHA256: ObservedVanillaEffectiveTemplateSHA256,
		ObservedSHA256: observed,
		Confidence:     "observed_v68_build_sensitive_template_hash",
		Note:           "template hash is a data-set detector for the observed DE v68 build/corpus; game patches can change the vanilla baseline",
	}
	if expected != "" {
		check.Status = "matches_user_template"
		check.BaselineName = "user_supplied_template"
		check.ExpectedSHA256 = expected
		check.Confidence = "user_supplied_baseline"
		check.Note = "template hash compared to the caller-supplied baseline"
	}
	if observed != check.ExpectedSHA256 {
		if expected != "" {
			check.Status = "differs_from_user_template"
		} else {
			check.Status = "differs_from_known_vanilla_template"
		}
	}
	return check
}

type effectiveTailBytes struct {
	playerIndex int
	label       string
	start       int
	end         int
	bytes       []byte
}

func effectiveDataTailsFromSpine(header []byte, spine *HeaderSpine) []effectiveTailBytes {
	var tails []effectiveTailBytes
	for _, player := range spine.Players {
		if player.Index == 0 {
			continue
		}
		start := player.ObjectTailMarkerEnd
		if start <= 0 {
			start = player.ObjectSpanStart
		}
		end := player.ObjectCandidateBandStart
		if end <= start {
			end = player.ObjectSpanEnd
		}
		if start <= 0 || end <= start || end > len(header) {
			continue
		}
		tails = append(tails, effectiveTailBytes{
			playerIndex: player.Index,
			label:       player.Label,
			start:       start,
			end:         end,
			bytes:       header[start:end],
		})
	}
	sort.Slice(tails, func(i, j int) bool { return tails[i].start < tails[j].start })
	return tails
}

func buildEffectivePlayerTable(ref effectiveTailBytes, tail effectiveTailBytes, slot PlayerSlot, dat *datfile.Index, opts EffectiveDataOptions) EffectivePlayerTable {
	compared := len(ref.bytes)
	if len(tail.bytes) < compared {
		compared = len(tail.bytes)
	}
	identical := 0
	for i := 0; i < compared; i++ {
		if ref.bytes[i] == tail.bytes[i] {
			identical++
		}
	}
	ratio := 0.0
	if compared > 0 {
		ratio = float64(identical) / float64(compared)
	}
	table := EffectivePlayerTable{
		PlayerIndex:    tail.playerIndex,
		PlayerNumber:   tail.playerIndex,
		PlayerName:     slot.Name,
		CivID:          slot.Civ,
		TailStart:      tail.start,
		TailEnd:        tail.end,
		TailBytes:      len(tail.bytes),
		IdenticalRatio: round2(ratio * 100),
		DiffRuns:       len(effectiveGameDataDiffRuns(ref.bytes, tail.bytes)),
	}
	if dat != nil && slot.Civ >= 0 && slot.Civ < len(dat.Civs) {
		table.CivName = dat.Civs[slot.Civ].Name
	}
	if opts.TailOffset+opts.Count*2 > len(tail.bytes) {
		table.Warnings = append(table.Warnings, fmt.Sprintf("requested array %d..%d exceeds tail bytes %d", opts.TailOffset, opts.TailOffset+opts.Count*2, len(tail.bytes)))
		return table
	}
	refValues := readU16Array(ref.bytes, opts.TailOffset, opts.Count)
	values := readU16Array(tail.bytes, opts.TailOffset, opts.Count)
	for i, value := range values {
		refValue := uint16(0)
		if i < len(refValues) {
			refValue = refValues[i]
		}
		table.EntryCount++
		if value != refValue {
			table.ChangedEntries++
		}
		if effectiveValueAvailable(value) {
			table.Available++
		} else {
			table.Unavailable++
		}
		switch value {
		case 0:
			table.Level0++
		case 1:
			table.Level1++
		case 2:
			table.Level2++
		case 3:
			table.Level3++
		case 0xffff:
		default:
			table.OtherValues++
		}
		if dat != nil && slot.Civ >= 0 && slot.Civ < len(dat.Civs) {
			civ := dat.Civs[slot.Civ]
			if i < len(civ.Units) && civ.Units[i].Name != "" {
				table.DatNamed++
			}
		}
		if !opts.IncludeAll && value == refValue {
			continue
		}
		if opts.Limit > 0 && len(table.Entries) >= opts.Limit {
			continue
		}
		entry := EffectiveArrayEntry{
			Index:           i,
			TailOffset:      opts.TailOffset + i*2,
			AbsOffset:       tail.start + opts.TailOffset + i*2,
			Value:           value,
			ReferenceValue:  refValue,
			Changed:         value != refValue,
			ReplayAvailable: effectiveValueAvailable(value),
			Meaning:         availabilityMeaning(value),
			UnitSlot:        i,
			Confidence:      "array_index_to_dat_unit_slot_crossref",
		}
		if dat != nil && slot.Civ >= 0 && slot.Civ < len(dat.Civs) {
			civ := dat.Civs[slot.Civ]
			if i < len(civ.Units) {
				unit := civ.Units[i]
				entry.DatPresent = unit.Present
				entry.UnitID = unit.ID
				entry.UnitName = unit.Name
				enabled := unit.Enabled
				entry.DatEnabled = &enabled
			}
		}
		table.Entries = append(table.Entries, entry)
	}
	table.Shown = len(table.Entries)
	return table
}

func effectiveValueAvailable(value uint16) bool {
	return value != 0xffff
}

func readU16Array(data []byte, offset int, count int) []uint16 {
	if offset < 0 || offset >= len(data) || count <= 0 {
		return nil
	}
	out := make([]uint16, 0, count)
	for i := 0; i < count; i++ {
		off := offset + i*2
		if off+2 > len(data) {
			break
		}
		out = append(out, binary.LittleEndian.Uint16(data[off:]))
	}
	return out
}

func availabilityMeaning(value uint16) string {
	switch value {
	case 0xffff:
		return "disabled_or_unavailable"
	case 0:
		return "available_or_level_0"
	case 1:
		return "available_or_level_1"
	case 2:
		return "available_or_level_2"
	case 3:
		return "available_or_level_3"
	default:
		return "other"
	}
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
