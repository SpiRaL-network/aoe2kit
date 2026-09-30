package replay

import (
	"fmt"
	"sort"

	"aoe2kit/pkg/aoe2"
)

type DiffStateOptions struct {
	Focus string
	Limit int
}

type DiffStateReport struct {
	Before       string                 `json:"before"`
	After        string                 `json:"after"`
	Verification aoe2.VerificationClaim `json:"verification"`
	Summary      DiffStateSummary       `json:"summary"`
	Spans        []DiffStateSpan        `json:"spans"`
	Sync         *DiffSyncComparison    `json:"sync,omitempty"`
	Warnings     []string               `json:"warnings,omitempty"`
}

type DiffStateSummary struct {
	Focus                 string `json:"focus"`
	HeaderBytesBefore     int    `json:"header_bytes_before"`
	HeaderBytesAfter      int    `json:"header_bytes_after"`
	BodyBytesBefore       int    `json:"body_bytes_before"`
	BodyBytesAfter        int    `json:"body_bytes_after"`
	HeaderChangedBytes    int    `json:"header_changed_bytes"`
	BodyChangedBytes      int    `json:"body_changed_bytes"`
	ChangedSpanCount      int    `json:"changed_span_count"`
	Shown                 int    `json:"shown"`
	SameTriggerGraph      bool   `json:"same_trigger_graph"`
	TriggerGraphCheck     string `json:"trigger_graph_check"`
	ChangedOpaqueBytes    int    `json:"changed_opaque_bytes,omitempty"`
	ChangedNonOpaqueBytes int    `json:"changed_non_opaque_bytes,omitempty"`
}

type DiffStateSpan struct {
	Space         string    `json:"space"`
	Start         int       `json:"start"`
	End           int       `json:"end"`
	Bytes         int       `json:"bytes"`
	BeforeRegion  string    `json:"before_region,omitempty"`
	BeforeStatus  string    `json:"before_status,omitempty"`
	AfterRegion   string    `json:"after_region,omitempty"`
	AfterStatus   string    `json:"after_status,omitempty"`
	BeforeHex     string    `json:"before_hex,omitempty"`
	AfterHex      string    `json:"after_hex,omitempty"`
	BeforeProfile SpanStats `json:"before_profile"`
	AfterProfile  SpanStats `json:"after_profile"`
	ShapeHints    []string  `json:"shape_hints,omitempty"`
}

type DiffSyncComparison struct {
	Verification          string                `json:"verification"`
	BeforeChecksumSamples int                   `json:"before_checksum_samples"`
	AfterChecksumSamples  int                   `json:"after_checksum_samples"`
	BeforeDurationMS      int                   `json:"before_duration_ms"`
	BeforeDuration        string                `json:"before_duration"`
	AfterDurationMS       int                   `json:"after_duration_ms"`
	AfterDuration         string                `json:"after_duration"`
	ComparablePlayers     int                   `json:"comparable_players"`
	ChangedPlayers        int                   `json:"changed_players"`
	ChangedWords          int                   `json:"changed_words"`
	PlayerDeltas          []DiffSyncPlayerDelta `json:"player_deltas,omitempty"`
	WordDeltas            []DiffSyncWordDelta   `json:"word_deltas,omitempty"`
	Warnings              []string              `json:"warnings,omitempty"`
}

type DiffSyncPlayerDelta struct {
	PlayerID               int    `json:"player_id"`
	PlayerLabel            string `json:"player_label"`
	ChangedWords           int    `json:"changed_words"`
	ObjectCountDelta       int64  `json:"object_count_delta,omitempty"`
	UnitTypeSumDelta       int64  `json:"unit_type_sum_delta,omitempty"`
	ObjectIDSumDelta       int64  `json:"object_id_sum_delta,omitempty"`
	PositionSumDelta       int64  `json:"position_sum_delta,omitempty"`
	Word3Delta             int64  `json:"word_3_delta,omitempty"`
	Word4Delta             int64  `json:"word_4_delta,omitempty"`
	ScoreCandidateDelta    int64  `json:"word_9_score_candidate_delta,omitempty"`
	ResourceStockpileDelta int64  `json:"resource_stockpile_delta,omitempty"`
}

type DiffSyncWordDelta struct {
	PlayerID    int    `json:"player_id"`
	PlayerLabel string `json:"player_label"`
	WordIndex   int    `json:"word_index"`
	WordName    string `json:"word_name"`
	Semantic    string `json:"semantic,omitempty"`
	Before      uint32 `json:"before"`
	After       uint32 `json:"after"`
	Delta       int64  `json:"delta"`
	Confidence  string `json:"confidence"`
}

type SpanStats struct {
	Entropy        float64 `json:"entropy"`
	DistinctBytes  int     `json:"distinct_bytes"`
	ZeroBytes      int     `json:"zero_bytes"`
	FFBytes        int     `json:"ff_bytes"`
	PrintableBytes int     `json:"printable_bytes"`
}

type byteRun struct {
	space string
	start int
	end   int
}

func BuildDiffState(beforePath string, afterPath string, opts DiffStateOptions) (*DiffStateReport, error) {
	beforeData, err := ReadRecordBytes(beforePath)
	if err != nil {
		return nil, err
	}
	afterData, err := ReadRecordBytes(afterPath)
	if err != nil {
		return nil, err
	}
	beforeRec, err := Parse(beforeData)
	if err != nil {
		return nil, err
	}
	afterRec, err := Parse(afterData)
	if err != nil {
		return nil, err
	}
	beforeCoverage, err := BuildCoverage(beforePath)
	if err != nil {
		return nil, err
	}
	afterCoverage, err := BuildCoverage(afterPath)
	if err != nil {
		return nil, err
	}
	beforeHeader := beforeRec.HeaderBytes()
	afterHeader := afterRec.HeaderBytes()
	beforeBody := replayBody(beforeData, beforeRec.HeaderLength)
	afterBody := replayBody(afterData, afterRec.HeaderLength)
	focus := opts.Focus
	if focus == "" {
		focus = "all"
	}
	report := &DiffStateReport{
		Before: beforePath,
		After:  afterPath,
		Verification: aoe2.VerificationClaim{
			Label:             "structure_verified_not_engine_verified",
			StructureVerified: true,
			EngineVerified:    false,
			Note:              "Changed bytes are framed against current coverage regions; semantic meaning is not claimed.",
		},
		Summary: DiffStateSummary{
			Focus:             focus,
			HeaderBytesBefore: len(beforeHeader),
			HeaderBytesAfter:  len(afterHeader),
			BodyBytesBefore:   len(beforeBody),
			BodyBytesAfter:    len(afterBody),
		},
		Warnings: append(append([]string(nil), beforeCoverage.Warnings...), afterCoverage.Warnings...),
	}
	report.Summary.SameTriggerGraph, report.Summary.TriggerGraphCheck = sameTriggerGraph(beforeRec, afterRec)
	runs := append(diffRuns("inflated_header", beforeHeader, afterHeader), diffRuns("body", beforeBody, afterBody)...)
	beforeWidth, beforeHeight := mapDimensionsFromCoverage(beforeCoverage)
	triggerCount := 0
	if beforeRec.TriggerGraph != nil {
		triggerCount = beforeRec.TriggerGraph.TriggerCount
	}
	beforeIndex := indexRegions(beforeCoverage.Regions)
	afterIndex := indexRegions(afterCoverage.Regions)
	for _, run := range runs {
		beforeRegion := beforeIndex.regionForRange(run.space, run.start, run.end)
		afterRegion := afterIndex.regionForRange(run.space, run.start, run.end)
		opaque := isOpaqueRegion(beforeRegion) || isOpaqueRegion(afterRegion)
		if opaque {
			report.Summary.ChangedOpaqueBytes += run.end - run.start
		} else {
			report.Summary.ChangedNonOpaqueBytes += run.end - run.start
		}
		if focus == "opaque" && !opaque {
			continue
		}
		beforeBytes := rangeBytes(run.space, run.start, run.end, beforeHeader, beforeBody)
		afterBytes := rangeBytes(run.space, run.start, run.end, afterHeader, afterBody)
		span := DiffStateSpan{
			Space:         run.space,
			Start:         run.start,
			End:           run.end,
			Bytes:         run.end - run.start,
			BeforeRegion:  replayRegionName(beforeRegion),
			BeforeStatus:  replayRegionStatus(beforeRegion),
			AfterRegion:   replayRegionName(afterRegion),
			AfterStatus:   replayRegionStatus(afterRegion),
			BeforeHex:     HexSample(beforeBytes, 96),
			AfterHex:      HexSample(afterBytes, 96),
			BeforeProfile: publicSpanStats(byteProfile(beforeBytes)),
			AfterProfile:  publicSpanStats(byteProfile(afterBytes)),
			ShapeHints:    shapeHints(run.end-run.start, beforeWidth, beforeHeight, triggerCount),
		}
		report.Spans = append(report.Spans, span)
	}
	sort.Slice(report.Spans, func(i, j int) bool {
		if report.Spans[i].Bytes == report.Spans[j].Bytes {
			if report.Spans[i].Space == report.Spans[j].Space {
				return report.Spans[i].Start < report.Spans[j].Start
			}
			return report.Spans[i].Space < report.Spans[j].Space
		}
		return report.Spans[i].Bytes > report.Spans[j].Bytes
	})
	report.Summary.HeaderChangedBytes = changedByteCount("inflated_header", runs)
	report.Summary.BodyChangedBytes = changedByteCount("body", runs)
	report.Summary.ChangedSpanCount = len(report.Spans)
	if opts.Limit > 0 && len(report.Spans) > opts.Limit {
		report.Spans = report.Spans[:opts.Limit]
	}
	report.Summary.Shown = len(report.Spans)
	syncComparison, err := buildDiffSyncComparison(beforePath, afterPath)
	if err != nil {
		report.Warnings = append(report.Warnings, "sync comparison unavailable: "+err.Error())
	} else {
		report.Sync = syncComparison
		report.Warnings = append(report.Warnings, syncComparison.Warnings...)
	}
	return report, nil
}

func buildDiffSyncComparison(beforePath, afterPath string) (*DiffSyncComparison, error) {
	beforeSync, err := BuildSyncStream(beforePath, SyncOptions{ChecksumsOnly: true})
	if err != nil {
		return nil, err
	}
	afterSync, err := BuildSyncStream(afterPath, SyncOptions{ChecksumsOnly: true})
	if err != nil {
		return nil, err
	}
	return diffSyncReports(beforeSync, afterSync), nil
}

func diffSyncReports(beforeSync, afterSync *SyncReport) *DiffSyncComparison {
	report := &DiffSyncComparison{
		Verification:          "structure_verified_final_checksum_matrix_delta_only",
		BeforeChecksumSamples: beforeSync.Summary.ChecksumDE,
		AfterChecksumSamples:  afterSync.Summary.ChecksumDE,
		BeforeDurationMS:      beforeSync.Summary.DurationMS,
		BeforeDuration:        beforeSync.Summary.Duration,
		AfterDurationMS:       afterSync.Summary.DurationMS,
		AfterDuration:         afterSync.Summary.Duration,
		Warnings:              append(append([]string(nil), beforeSync.Warnings...), afterSync.Warnings...),
	}
	beforeLast := lastChecksumMatrix(beforeSync)
	afterLast := lastChecksumMatrix(afterSync)
	if beforeLast == nil || afterLast == nil {
		report.Warnings = append(report.Warnings, "one or both replays contain no DE checksum matrix; final sync delta comparison skipped")
		return report
	}
	semantics := syncWordSemantics()
	for player := 0; player < len(beforeLast) && player < len(afterLast); player++ {
		if len(beforeLast[player]) < 11 || len(afterLast[player]) < 11 {
			continue
		}
		if allZeroU32(beforeLast[player]) && allZeroU32(afterLast[player]) {
			continue
		}
		report.ComparablePlayers++
		playerDelta := DiffSyncPlayerDelta{
			PlayerID:    player + 1,
			PlayerLabel: fmt.Sprintf("P%d", player+1),
		}
		for word := 0; word < 11; word++ {
			before := beforeLast[player][word]
			after := afterLast[player][word]
			if before == after {
				continue
			}
			delta := checksumWordDelta(before, after)
			playerDelta.ChangedWords++
			report.ChangedWords++
			switch word {
			case 1:
				playerDelta.ResourceStockpileDelta = delta
			case 2:
				playerDelta.UnitTypeSumDelta = delta
			case 3:
				playerDelta.Word3Delta = delta
			case 4:
				playerDelta.Word4Delta = delta
			case 6:
				playerDelta.ObjectCountDelta = delta
			case 7:
				playerDelta.PositionSumDelta = delta
			case 9:
				playerDelta.ScoreCandidateDelta = delta
			case 10:
				playerDelta.ObjectIDSumDelta = delta
			}
			wordName := syncWordName(word)
			report.WordDeltas = append(report.WordDeltas, DiffSyncWordDelta{
				PlayerID:    player + 1,
				PlayerLabel: playerDelta.PlayerLabel,
				WordIndex:   word,
				WordName:    wordName,
				Semantic:    semantics[wordName],
				Before:      before,
				After:       after,
				Delta:       delta,
				Confidence:  syncWordDiffConfidence(word),
			})
		}
		if playerDelta.ChangedWords > 0 {
			report.ChangedPlayers++
			report.PlayerDeltas = append(report.PlayerDeltas, playerDelta)
		}
	}
	return report
}

func lastChecksumMatrix(report *SyncReport) [][]uint32 {
	for i := len(report.Events) - 1; i >= 0; i-- {
		if len(report.Events[i].Matrix) == 8 {
			return report.Events[i].Matrix
		}
	}
	return nil
}

func syncWordName(word int) string {
	return fmt.Sprintf("word_%d", word)
}

func syncWordDiffConfidence(word int) string {
	switch word {
	case 1, 2, 6, 8, 10:
		return "decoded_final_matrix_delta"
	case 3, 4, 7, 9:
		return "provisional_final_matrix_delta"
	default:
		return "observed_final_matrix_delta"
	}
}

func checksumWordDelta(before, after uint32) int64 {
	return int64(int32(after - before))
}

func replayBody(data []byte, headerLength int) []byte {
	if headerLength < 0 || headerLength > len(data) {
		return nil
	}
	return data[headerLength:]
}

func diffRuns(space string, before []byte, after []byte) []byteRun {
	maxLen := len(before)
	if len(after) > maxLen {
		maxLen = len(after)
	}
	var runs []byteRun
	start := -1
	for i := 0; i < maxLen; i++ {
		var b byte
		var a byte
		bOK := i < len(before)
		aOK := i < len(after)
		if bOK {
			b = before[i]
		}
		if aOK {
			a = after[i]
		}
		changed := bOK != aOK || b != a
		if changed && start < 0 {
			start = i
		}
		if !changed && start >= 0 {
			runs = append(runs, byteRun{space: space, start: start, end: i})
			start = -1
		}
	}
	if start >= 0 {
		runs = append(runs, byteRun{space: space, start: start, end: maxLen})
	}
	return coalesceNearbyRuns(runs, 64)
}

func coalesceNearbyRuns(runs []byteRun, gap int) []byteRun {
	if len(runs) == 0 {
		return nil
	}
	out := []byteRun{runs[0]}
	for _, run := range runs[1:] {
		last := &out[len(out)-1]
		if run.space == last.space && run.start-last.end <= gap {
			last.end = run.end
			continue
		}
		out = append(out, run)
	}
	return out
}

func changedByteCount(space string, runs []byteRun) int {
	total := 0
	for _, run := range runs {
		if run.space == space {
			total += run.end - run.start
		}
	}
	return total
}

func rangeBytes(space string, start int, end int, header []byte, body []byte) []byte {
	var source []byte
	switch space {
	case "inflated_header":
		source = header
	case "body":
		source = body
	default:
		return nil
	}
	if start < 0 || start >= len(source) || end < start {
		return nil
	}
	if end > len(source) {
		end = len(source)
	}
	return source[start:end]
}

func isOpaqueRegion(region *ReplayRegion) bool {
	return region == nil || region.Status != "decoded"
}

func replayRegionName(region *ReplayRegion) string {
	if region == nil {
		return ""
	}
	return region.Name
}

func replayRegionStatus(region *ReplayRegion) string {
	if region == nil {
		return ""
	}
	return region.Status
}

func publicSpanStats(profile spanByteProfile) SpanStats {
	return SpanStats{
		Entropy:        profile.entropy,
		DistinctBytes:  profile.distinct,
		ZeroBytes:      profile.zeros,
		FFBytes:        profile.ffs,
		PrintableBytes: profile.printable,
	}
}

func sameTriggerGraph(before *File, after *File) (bool, string) {
	if before.TriggerGraph == nil || after.TriggerGraph == nil {
		return false, "trigger_graph_unavailable"
	}
	if before.TriggerGraph.SHA256 == after.TriggerGraph.SHA256 {
		return true, "same_sha256"
	}
	return false, "different_sha256"
}

type replayRegionIndex map[string][]ReplayRegion

func indexRegions(regions []ReplayRegion) replayRegionIndex {
	out := replayRegionIndex{}
	for _, region := range regions {
		out[region.Space] = append(out[region.Space], region)
	}
	for space := range out {
		sort.Slice(out[space], func(i, j int) bool {
			if out[space][i].Start == out[space][j].Start {
				return out[space][i].End < out[space][j].End
			}
			return out[space][i].Start < out[space][j].Start
		})
	}
	return out
}

func (idx replayRegionIndex) regionForRange(space string, start int, end int) *ReplayRegion {
	regions := idx[space]
	if len(regions) == 0 {
		return nil
	}
	pos := sort.Search(len(regions), func(i int) bool {
		return regions[i].End > start
	})
	if pos >= len(regions) {
		return nil
	}
	best := regions[pos]
	bestOverlap := replayOverlapBytes(start, end, best.Start, best.End)
	for i := pos + 1; i < len(regions) && regions[i].Start < end; i++ {
		overlap := replayOverlapBytes(start, end, regions[i].Start, regions[i].End)
		if overlap > bestOverlap {
			best = regions[i]
			bestOverlap = overlap
		}
	}
	if bestOverlap <= 0 {
		return nil
	}
	return &best
}

func replayOverlapBytes(aStart int, aEnd int, bStart int, bEnd int) int {
	start := aStart
	if bStart > start {
		start = bStart
	}
	end := aEnd
	if bEnd < end {
		end = bEnd
	}
	if end <= start {
		return 0
	}
	return end - start
}
