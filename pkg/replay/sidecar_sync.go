package replay

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"

	xsauthor "aoe2kit/pkg/xs"
)

type SidecarSyncOptions struct {
	SidecarPath string
	LedgerPath  string
	Schema      string
	PlayerID    int
	WindowMS    int
}

type SidecarSyncReport struct {
	Path          string                           `json:"path,omitempty"`
	SidecarPath   string                           `json:"sidecar_path,omitempty"`
	LedgerPath    string                           `json:"ledger_path,omitempty"`
	Schema        string                           `json:"schema,omitempty"`
	Method        string                           `json:"method"`
	Verification  string                           `json:"verification"`
	Summary       SidecarSyncSummary               `json:"summary"`
	Sidecar       SidecarStateVector               `json:"sidecar"`
	Correlations  []SidecarSyncCorrelation         `json:"correlations,omitempty"`
	Warnings      []string                         `json:"warnings,omitempty"`
	Decode        *xsauthor.DataDecodeReport       `json:"xsdat_decode,omitempty"`
	SchemaDecode  *xsauthor.DataSchemaDecodeReport `json:"xsdat_schema_decode,omitempty"`
	WordSemantics map[string]string                `json:"checksum_word_semantics,omitempty"`
}

type SidecarSyncSummary struct {
	PhaseCount          int  `json:"phase_count"`
	ChecksumSamples     int  `json:"checksum_samples"`
	SidecarDecodeOK     bool `json:"sidecar_decode_ok"`
	LedgerAssertions    int  `json:"ledger_assertions"`
	LedgerPassed        int  `json:"ledger_passed"`
	LedgerFailed        int  `json:"ledger_failed"`
	LedgerUnknown       int  `json:"ledger_unknown"`
	MatchedPhases       int  `json:"matched_phases"`
	MissingSamples      int  `json:"missing_samples"`
	KnownWordChecks     int  `json:"known_word_checks"`
	KnownWordMatches    int  `json:"known_word_matches"`
	KnownWordMismatches int  `json:"known_word_mismatches"`
	CarrierMatches      int  `json:"carrier_matches"`
	UnitTypeMatches     int  `json:"unit_type_sum_matches"`
	ObjectCountMatches  int  `json:"object_count_matches"`
	AttrMirrorHits      int  `json:"attribute_mirror_hits"`
}

type SidecarStateVector struct {
	Magic       string              `json:"magic,omitempty"`
	Version     int                 `json:"version,omitempty"`
	Label       string              `json:"label,omitempty"`
	RowsWritten int                 `json:"rows_written,omitempty"`
	AppendProbe int                 `json:"append_probe,omitempty"`
	Phases      []SidecarStatePhase `json:"phases,omitempty"`
}

type SidecarStatePhase struct {
	Index    int                `json:"index"`
	PhaseID  int                `json:"phase_id"`
	Label    string             `json:"label,omitempty"`
	TimerS   int                `json:"ledger_timer_s,omitempty"`
	XSTimeS  int                `json:"xs_time_s"`
	TargetMS int                `json:"target_ms"`
	Food     float64            `json:"food"`
	Wood     float64            `json:"wood"`
	Gold     float64            `json:"gold"`
	Stone    float64            `json:"stone"`
	Attr20   float64            `json:"attr20"`
	Attr33   float64            `json:"attr33"`
	Attr220  float64            `json:"attr220"`
	Counts   map[string]int     `json:"counts"`
	Expected SidecarExpectation `json:"expected"`
}

type SidecarExpectation struct {
	ResourceStockpile      int  `json:"resource_stockpile"`
	UnitTypeSum            int  `json:"unit_type_sum"`
	ObjectCount            int  `json:"object_count"`
	ResourceStockpileKnown bool `json:"resource_stockpile_known"`
	UnitTypeSumKnown       bool `json:"unit_type_sum_known"`
	ObjectCountKnown       bool `json:"object_count_known"`
}

type SidecarSyncCorrelation struct {
	Phase             SidecarStatePhase          `json:"phase"`
	Verdict           string                     `json:"verdict"`
	Confidence        string                     `json:"confidence"`
	Sample            *SidecarSyncSample         `json:"sample,omitempty"`
	DeltaFromPrevious []int64                    `json:"delta_words_from_previous_sample,omitempty"`
	KnownWordChecks   []SidecarKnownWordCheck    `json:"known_word_checks,omitempty"`
	CarrierMatch      *bool                      `json:"carrier_word_1_match,omitempty"`
	UnitTypeMatch     *bool                      `json:"unit_type_word_2_match,omitempty"`
	ObjectCountMatch  *bool                      `json:"object_count_word_6_match,omitempty"`
	AttributeMirrors  []AttributeMirrorCandidate `json:"attribute_mirrors,omitempty"`
	Notes             []string                   `json:"notes,omitempty"`
}

type SidecarSyncSample struct {
	TimeMS int      `json:"time_ms"`
	Time   string   `json:"time"`
	LagMS  int      `json:"lag_ms"`
	Words  []uint32 `json:"words_u32"`
}

type SidecarKnownWordCheck struct {
	WordIndex int    `json:"word_index"`
	Name      string `json:"name"`
	Expected  int    `json:"expected"`
	Actual    int    `json:"actual"`
	Match     bool   `json:"match"`
}

type AttributeMirrorCandidate struct {
	Attribute string `json:"attribute"`
	Value     int    `json:"value"`
	WordIndex int    `json:"word_index"`
}

type sidecarPhaseMetadata struct {
	Label  string
	TimerS int
}

func BuildSidecarSyncReport(path string, opts SidecarSyncOptions) (*SidecarSyncReport, error) {
	if strings.TrimSpace(opts.SidecarPath) == "" {
		return nil, fmt.Errorf("--xsdat is required")
	}
	if strings.TrimSpace(opts.LedgerPath) == "" && strings.TrimSpace(opts.Schema) == "" {
		return nil, fmt.Errorf("--ledger or --schema is required for typed sidecar-sync decode")
	}
	if opts.PlayerID == 0 {
		opts.PlayerID = 1
	}
	if opts.PlayerID < 1 || opts.PlayerID > 8 {
		return nil, fmt.Errorf("--player must be 1..8")
	}
	if opts.WindowMS <= 0 {
		opts.WindowMS = 20000
	}
	var decode *xsauthor.DataDecodeReport
	var schemaDecode *xsauthor.DataSchemaDecodeReport
	var sidecar SidecarStateVector
	var warnings []string
	if strings.TrimSpace(opts.Schema) != "" {
		report, err := xsauthor.DecodeDataFileSchema(opts.SidecarPath, xsauthor.DataSchemaDecodeOptions{Schema: opts.Schema})
		if err != nil {
			return nil, err
		}
		schemaDecode = &report
		sidecar, warnings = parseSchemaSidecar(report, opts.PlayerID)
	} else {
		report, err := xsauthor.DecodeDataFile(opts.SidecarPath, xsauthor.DataDecodeOptions{Ledger: opts.LedgerPath})
		if err != nil {
			return nil, err
		}
		decode = &report
		sidecar, warnings = parseStateVectorSidecar(report.Decode.Values)
		metadata, err := loadSidecarPhaseMetadata(opts.LedgerPath)
		if err != nil {
			warnings = append(warnings, "phase metadata parse failed: "+err.Error())
		}
		applySidecarPhaseMetadata(&sidecar, metadata)
	}
	sync, err := BuildSyncStream(path, SyncOptions{ChecksumsOnly: true})
	if err != nil {
		return nil, err
	}
	report := &SidecarSyncReport{
		Path:          path,
		SidecarPath:   opts.SidecarPath,
		LedgerPath:    opts.LedgerPath,
		Schema:        strings.TrimSpace(opts.Schema),
		Method:        "xsdat_state_vector_rows_correlated_to_nearest_de_sync_matrix_samples",
		Verification:  "structure_verified_sidecar_and_sync_correlation_not_engine_generalization",
		Sidecar:       sidecar,
		Warnings:      append(warnings, sync.Warnings...),
		Decode:        decode,
		SchemaDecode:  schemaDecode,
		WordSemantics: syncWordSemantics(),
	}
	report.Summary.PhaseCount = len(sidecar.Phases)
	report.Summary.ChecksumSamples = sync.Summary.ChecksumDE
	if decode != nil {
		applySidecarDecodeSummary(report, *decode)
		if !decode.OK {
			report.Warnings = append(report.Warnings, "sidecar ledger assertions did not pass completely")
		}
	}
	if schemaDecode != nil {
		report.Summary.SidecarDecodeOK = schemaDecode.OK
		if !schemaDecode.OK {
			report.Warnings = append(report.Warnings, "sidecar schema decode did not pass completely")
		}
	}
	report.Correlations = correlateSidecarPhases(sidecar.Phases, sync.Events, opts.PlayerID, opts.WindowMS)
	for _, row := range report.Correlations {
		if row.Sample == nil {
			report.Summary.MissingSamples++
			continue
		}
		report.Summary.MatchedPhases++
		for _, check := range row.KnownWordChecks {
			report.Summary.KnownWordChecks++
			if check.Match {
				report.Summary.KnownWordMatches++
			} else {
				report.Summary.KnownWordMismatches++
			}
		}
		if row.CarrierMatch != nil && *row.CarrierMatch {
			report.Summary.CarrierMatches++
		}
		if row.UnitTypeMatch != nil && *row.UnitTypeMatch {
			report.Summary.UnitTypeMatches++
		}
		if row.ObjectCountMatch != nil && *row.ObjectCountMatch {
			report.Summary.ObjectCountMatches++
		}
		report.Summary.AttrMirrorHits += len(row.AttributeMirrors)
	}
	return report, nil
}

func applySidecarDecodeSummary(report *SidecarSyncReport, decode xsauthor.DataDecodeReport) {
	report.Summary.SidecarDecodeOK = decode.OK
	report.Summary.LedgerAssertions = decode.Summary.Total
	report.Summary.LedgerPassed = decode.Summary.Passed
	report.Summary.LedgerFailed = decode.Summary.Failed
	report.Summary.LedgerUnknown = decode.Summary.Unknown
}

func loadSidecarPhaseMetadata(path string) (map[int]sidecarPhaseMetadata, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var ledger struct {
		RowsSummary []struct {
			PhaseID int    `json:"phase_id"`
			Timer   int    `json:"timer"`
			Label   string `json:"label"`
		} `json:"rows_summary"`
	}
	if err := json.Unmarshal(data, &ledger); err != nil {
		return nil, err
	}
	out := map[int]sidecarPhaseMetadata{}
	for _, row := range ledger.RowsSummary {
		out[row.PhaseID] = sidecarPhaseMetadata{Label: row.Label, TimerS: row.Timer}
	}
	return out, nil
}

func applySidecarPhaseMetadata(sidecar *SidecarStateVector, metadata map[int]sidecarPhaseMetadata) {
	for i := range sidecar.Phases {
		meta, ok := metadata[sidecar.Phases[i].PhaseID]
		if !ok {
			continue
		}
		sidecar.Phases[i].Label = meta.Label
		sidecar.Phases[i].TimerS = meta.TimerS
	}
}

func parseStateVectorSidecar(values []xsauthor.DataValue) (SidecarStateVector, []string) {
	var out SidecarStateVector
	var warnings []string
	if len(values) >= 3 {
		out.Magic, _ = dataString(values[0])
		out.Version, _ = dataInt(values[1])
		out.Label, _ = dataString(values[2])
	}
	for i := 3; i < len(values); {
		tag, ok := dataString(values[i])
		if !ok {
			warnings = append(warnings, fmt.Sprintf("value %d is not a sidecar string tag", i))
			i++
			continue
		}
		switch tag {
		case "phase":
			if i+13 >= len(values) {
				warnings = append(warnings, fmt.Sprintf("phase at value %d is truncated", i))
				return out, warnings
			}
			phaseID, _ := dataInt(values[i+1])
			xsTime, _ := dataInt(values[i+2])
			phase := SidecarStatePhase{
				Index:    len(out.Phases),
				PhaseID:  phaseID,
				XSTimeS:  xsTime,
				TargetMS: xsTime * 1000,
				Counts:   map[string]int{},
			}
			phase.Food, _ = dataFloat(values[i+3])
			phase.Wood, _ = dataFloat(values[i+4])
			phase.Gold, _ = dataFloat(values[i+5])
			phase.Stone, _ = dataFloat(values[i+6])
			phase.Attr20, _ = dataFloat(values[i+7])
			phase.Attr33, _ = dataFloat(values[i+8])
			phase.Attr220, _ = dataFloat(values[i+9])
			phase.Counts["barracks_12"], _ = dataInt(values[i+10])
			phase.Counts["archer_4"], _ = dataInt(values[i+11])
			phase.Counts["monk_125"], _ = dataInt(values[i+12])
			phase.Counts["castle_82"], _ = dataInt(values[i+13])
			phase.Expected = sidecarExpectation(phase)
			out.Phases = append(out.Phases, phase)
			i += 14
		case "end":
			if i+1 < len(values) {
				out.RowsWritten, _ = dataInt(values[i+1])
			}
			i += 2
		case "append_probe":
			if i+1 < len(values) {
				out.AppendProbe, _ = dataInt(values[i+1])
			}
			i += 2
		default:
			warnings = append(warnings, fmt.Sprintf("unknown sidecar tag %q at value %d", tag, i))
			i++
		}
	}
	return out, warnings
}

func parseSchemaSidecar(report xsauthor.DataSchemaDecodeReport, playerID int) (SidecarStateVector, []string) {
	var out SidecarStateVector
	var warnings []string
	if report.Header != nil {
		out.Magic, _ = report.Header["magic"].(string)
		out.Version = anyInt(report.Header["version"])
		out.Label, _ = report.Header["fixture"].(string)
	}
	if report.Footer != nil {
		out.RowsWritten = anyInt(report.Footer["rows_written"])
		out.AppendProbe = anyInt(report.Footer["append_probe"])
	}
	for _, row := range report.Rows {
		phase := SidecarStatePhase{
			Index:    len(out.Phases),
			PhaseID:  row.PhaseID,
			XSTimeS:  row.TimeS,
			TargetMS: row.TimeS * 1000,
			Label:    schemaPhaseLabel(row),
			Counts:   map[string]int{},
		}
		phase.Food = anyFloat(firstPresent(row.Fields, "p1_resource_sum_carrier_food_attr0", "p1_food_attr0"))
		phase.Wood = anyFloat(firstPresent(row.Fields, "p1_wood_family_attr1", "p1_wood_case_player_attr1", "p1_wood_attr1"))
		phase.Gold = anyFloat(firstPresent(row.Fields, "p1_gold_outcome_attr3", "p1_gold_case_kind_attr3", "p1_gold_attr3"))
		phase.Stone = anyFloat(firstPresent(row.Fields, "p1_stone_kind_attr2", "p1_stone_row_index_attr2", "p1_stone_expected_p1_kills_attr2", "p1_stone_attr2"))
		phase.Attr20 = anyFloat(row.Fields[fmt.Sprintf("p%d_kills_attr20", playerID)])
		phase.Counts = schemaCountsForPlayer(row.Fields, playerID)
		phase.Expected = sidecarExpectation(phase)
		if report.Schema == "a2ksem2-dat-command-semantics" {
			phase.Expected.UnitTypeSumKnown = false
			phase.Expected.ObjectCountKnown = false
		}
		out.Phases = append(out.Phases, phase)
	}
	if !report.OK {
		warnings = append(warnings, "schema sidecar decode has errors")
	}
	return out, warnings
}

func schemaPhaseLabel(row xsauthor.DataSchemaRow) string {
	casePlayer := anyInt(row.Fields["case_player"])
	caseKind := anyInt(row.Fields["case_kind"])
	caseUnit := anyInt(row.Fields["case_unit"])
	if casePlayer <= 0 || caseKind <= 0 {
		if casePlayer > 0 && caseUnit > 0 {
			return fmt.Sprintf("p%d_unit_%d", casePlayer, caseUnit)
		}
		return ""
	}
	kind := "case"
	switch caseKind {
	case 1:
		kind = "unit_kill"
	case 2:
		kind = "building_raze"
	case 9:
		kind = "close"
	}
	return fmt.Sprintf("p%d_%s", casePlayer, kind)
}

func schemaCountsForPlayer(fields map[string]any, playerID int) map[string]int {
	out := map[string]int{}
	prefix := fmt.Sprintf("p%d_", playerID)
	for key, value := range fields {
		if !strings.HasPrefix(key, prefix) || !strings.HasSuffix(key, "_count") {
			continue
		}
		label := strings.TrimPrefix(key, prefix)
		out[label] = anyInt(value)
	}
	return out
}

func sidecarExpectation(phase SidecarStatePhase) SidecarExpectation {
	unitTypeSum := 0
	objectCount := 0
	for name, count := range phase.Counts {
		unitID, ok := countUnitID(name)
		if !ok {
			continue
		}
		unitTypeSum += unitID * count
		objectCount += count
	}
	return SidecarExpectation{
		ResourceStockpile:      int(math.Round(phase.Food + phase.Wood + phase.Gold + phase.Stone)),
		UnitTypeSum:            unitTypeSum,
		ObjectCount:            objectCount,
		ResourceStockpileKnown: true,
		UnitTypeSumKnown:       true,
		ObjectCountKnown:       true,
	}
}

func countUnitID(name string) (int, bool) {
	name = strings.TrimSuffix(name, "_count")
	parts := strings.Split(name, "_")
	if len(parts) < 2 {
		return 0, false
	}
	unitID, err := strconv.Atoi(parts[len(parts)-1])
	if err != nil {
		return 0, false
	}
	return unitID, true
}

func correlateSidecarPhases(phases []SidecarStatePhase, events []SyncEvent, playerID, windowMS int) []SidecarSyncCorrelation {
	out := make([]SidecarSyncCorrelation, 0, len(phases))
	var previous *SidecarSyncSample
	for _, phase := range phases {
		row := SidecarSyncCorrelation{
			Phase:      phase,
			Verdict:    "unsampled",
			Confidence: "no_checksum_sample_in_phase_window",
		}
		sample := firstSyncSampleAtOrAfter(events, playerID, phase.TargetMS, windowMS)
		if sample == nil {
			row.Notes = append(row.Notes, "no checksum sample found in phase window")
			out = append(out, row)
			continue
		}
		row.Sample = sample
		if previous != nil {
			row.DeltaFromPrevious = wordDeltas(previous.Words, sample.Words)
		}
		row.KnownWordChecks = sidecarKnownWordChecks(sample.Words, phase)
		row.CarrierMatch = checkMatchPointer(row.KnownWordChecks, 1)
		row.UnitTypeMatch = checkMatchPointer(row.KnownWordChecks, 2)
		row.ObjectCountMatch = checkMatchPointer(row.KnownWordChecks, 6)
		row.Verdict = "matched"
		row.Confidence = "sidecar_phase_nearby_checksum_known_words_match"
		for _, check := range row.KnownWordChecks {
			if check.Match {
				continue
			}
			row.Verdict = "known_word_mismatch"
			row.Confidence = "sidecar_phase_sampled_known_checksum_word_mismatch"
			row.Notes = append(row.Notes, fmt.Sprintf("word_%d %s expected %d actual %d", check.WordIndex, check.Name, check.Expected, check.Actual))
		}
		row.AttributeMirrors = append(row.AttributeMirrors, attributeMirrors(sample.Words, "attr20", phase.Attr20)...)
		row.AttributeMirrors = append(row.AttributeMirrors, attributeMirrors(sample.Words, "attr33", phase.Attr33)...)
		row.AttributeMirrors = append(row.AttributeMirrors, attributeMirrors(sample.Words, "attr220", phase.Attr220)...)
		previous = sample
		out = append(out, row)
	}
	return out
}

func sidecarKnownWordChecks(words []uint32, phase SidecarStatePhase) []SidecarKnownWordCheck {
	specs := []struct {
		index    int
		name     string
		expected int
		known    bool
	}{
		{index: 1, name: "resource_stockpile", expected: phase.Expected.ResourceStockpile, known: phase.Expected.ResourceStockpileKnown},
		{index: 2, name: "unit_type_sum", expected: phase.Expected.UnitTypeSum, known: phase.Expected.UnitTypeSumKnown},
		{index: 6, name: "object_count", expected: phase.Expected.ObjectCount, known: phase.Expected.ObjectCountKnown},
	}
	out := make([]SidecarKnownWordCheck, 0, len(specs))
	for _, spec := range specs {
		if !spec.known {
			continue
		}
		check := SidecarKnownWordCheck{WordIndex: spec.index, Name: spec.name, Expected: spec.expected}
		if len(words) > spec.index {
			check.Actual = int(words[spec.index])
			check.Match = check.Actual == check.Expected
		}
		out = append(out, check)
	}
	return out
}

func checkMatchPointer(checks []SidecarKnownWordCheck, wordIndex int) *bool {
	for _, check := range checks {
		if check.WordIndex != wordIndex {
			continue
		}
		return boolPointer(check.Match)
	}
	return nil
}

func firstSyncSampleAtOrAfter(events []SyncEvent, playerID, targetMS, windowMS int) *SidecarSyncSample {
	for _, ev := range events {
		if ev.Form != "checksum_de" || ev.TimeMS < targetMS || ev.TimeMS-targetMS > windowMS {
			continue
		}
		if len(ev.Matrix) < playerID || len(ev.Matrix[playerID-1]) == 0 {
			continue
		}
		words := append([]uint32(nil), ev.Matrix[playerID-1]...)
		return &SidecarSyncSample{TimeMS: ev.TimeMS, Time: ev.Time, LagMS: ev.TimeMS - targetMS, Words: words}
	}
	return nil
}

func wordDeltas(prev, curr []uint32) []int64 {
	n := len(prev)
	if len(curr) < n {
		n = len(curr)
	}
	out := make([]int64, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, int64(curr[i])-int64(prev[i]))
	}
	return out
}

func attributeMirrors(words []uint32, name string, value float64) []AttributeMirrorCandidate {
	rounded := int(math.Round(value))
	if rounded == 0 || math.Abs(value-float64(rounded)) > 0.0001 {
		return nil
	}
	var out []AttributeMirrorCandidate
	for i, word := range words {
		if int(word) == rounded {
			out = append(out, AttributeMirrorCandidate{Attribute: name, Value: rounded, WordIndex: i})
		}
	}
	return out
}

func dataString(value xsauthor.DataValue) (string, bool) {
	return value.String, value.Type == "string"
}

func dataInt(value xsauthor.DataValue) (int, bool) {
	if value.Int != nil {
		return int(*value.Int), true
	}
	if value.UInt != nil {
		return int(*value.UInt), true
	}
	return 0, false
}

func dataFloat(value xsauthor.DataValue) (float64, bool) {
	if value.Float != nil {
		return float64(*value.Float), true
	}
	if value.Int != nil {
		return float64(*value.Int), true
	}
	if value.UInt != nil {
		return float64(*value.UInt), true
	}
	return 0, false
}

func firstPresent(fields map[string]any, keys ...string) any {
	for _, key := range keys {
		value, ok := fields[key]
		if ok {
			return value
		}
	}
	return nil
}

func anyInt(value any) int {
	switch v := value.(type) {
	case int:
		return v
	case int32:
		return int(v)
	case int64:
		return int(v)
	case uint32:
		return int(v)
	case uint64:
		return int(v)
	case float32:
		return int(math.Round(float64(v)))
	case float64:
		return int(math.Round(v))
	}
	return 0
}

func anyFloat(value any) float64 {
	switch v := value.(type) {
	case float32:
		return float64(v)
	case float64:
		return v
	case int:
		return float64(v)
	case int32:
		return float64(v)
	case int64:
		return float64(v)
	case uint32:
		return float64(v)
	case uint64:
		return float64(v)
	}
	return 0
}

func boolPointer(value bool) *bool {
	return &value
}
