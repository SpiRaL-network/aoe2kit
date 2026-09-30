package replay

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type CorpusOptions struct {
	Recursive      bool
	IncludeZip     bool
	UnknownSamples int
}

type ReplayCorpusReport struct {
	Folder       string               `json:"folder"`
	Method       string               `json:"method"`
	Verification string               `json:"verification"`
	Summary      ReplayCorpusSummary  `json:"summary"`
	Rows         []ReplayCorpusRow    `json:"rows"`
	Unknowns     []CorpusUnknownGroup `json:"unknown_actions,omitempty"`
	Warnings     []string             `json:"warnings,omitempty"`
}

type ReplayCorpusSummary struct {
	Scanned                 int            `json:"scanned"`
	Failures                int            `json:"failures"`
	ScenarioReplays         int            `json:"scenario_replays"`
	NonScenarioReplays      int            `json:"non_scenario_replays"`
	TriggerGraphs           int            `json:"trigger_graphs"`
	FallbackFingerprints    int            `json:"fallback_fingerprints"`
	Players                 int            `json:"players"`
	Humans                  int            `json:"humans"`
	AIs                     int            `json:"ais"`
	TotalActions            int            `json:"total_actions"`
	UntypedActions          int            `json:"untyped_actions"`
	UnknownOps              int            `json:"unknown_ops"`
	Chat                    int            `json:"chat"`
	Taunts                  int            `json:"taunts"`
	Flares                  int            `json:"flares"`
	Resigns                 int            `json:"resigns"`
	Postgames               int            `json:"postgames"`
	BodyOps                 map[int]int    `json:"body_ops,omitempty"`
	ActionIDs               map[int]int    `json:"action_ids,omitempty"`
	HeaderBytes             int            `json:"header_bytes"`
	BodyBytes               int            `json:"body_bytes"`
	HeaderOpaqueBytes       int            `json:"header_opaque_bytes"`
	BodyOpaqueBytes         int            `json:"body_opaque_bytes"`
	HeaderDecodedPercentMin float64        `json:"header_decoded_percent_min,omitempty"`
	HeaderDecodedPercentAvg float64        `json:"header_decoded_percent_avg,omitempty"`
	BodyDecodedPercentMin   float64        `json:"body_decoded_percent_min,omitempty"`
	BodyDecodedPercentAvg   float64        `json:"body_decoded_percent_avg,omitempty"`
	PostgameShapes          map[string]int `json:"postgame_shapes,omitempty"`
}

type ReplayCorpusRow struct {
	Path                 string      `json:"path"`
	OK                   bool        `json:"ok"`
	Error                string      `json:"error,omitempty"`
	FileBytes            int         `json:"file_bytes,omitempty"`
	GameVersion          string      `json:"game_version,omitempty"`
	SaveVersion          float64     `json:"save_version,omitempty"`
	LogVersion           uint32      `json:"log_version,omitempty"`
	ReplayKind           string      `json:"replay_kind,omitempty"`
	ScenarioIdentity     string      `json:"scenario_identity,omitempty"`
	ScenarioIdentityTier string      `json:"scenario_identity_tier,omitempty"`
	TriggerCount         int         `json:"trigger_count,omitempty"`
	EffectCount          int         `json:"effect_count,omitempty"`
	ConditionCount       int         `json:"condition_count,omitempty"`
	MessageCount         int         `json:"message_count,omitempty"`
	MapWidth             int         `json:"map_width,omitempty"`
	MapHeight            int         `json:"map_height,omitempty"`
	DataSetStatus        string      `json:"data_set_status,omitempty"`
	ActiveDataSet        string      `json:"active_data_set,omitempty"`
	PlayerCount          int         `json:"player_count,omitempty"`
	HumanCount           int         `json:"human_count,omitempty"`
	AICount              int         `json:"ai_count,omitempty"`
	PlayerNames          []string    `json:"player_names,omitempty"`
	DurationMS           int         `json:"duration_ms,omitempty"`
	Duration             string      `json:"duration,omitempty"`
	Actions              int         `json:"actions,omitempty"`
	UntypedActions       int         `json:"untyped_actions,omitempty"`
	UnknownOps           int         `json:"unknown_ops,omitempty"`
	Chat                 int         `json:"chat,omitempty"`
	Taunts               int         `json:"taunts,omitempty"`
	Flares               int         `json:"flares,omitempty"`
	Resigns              int         `json:"resigns,omitempty"`
	Postgames            int         `json:"postgames,omitempty"`
	HeaderBytes          int         `json:"header_bytes,omitempty"`
	BodyBytes            int         `json:"body_bytes,omitempty"`
	HeaderOpaqueBytes    int         `json:"header_opaque_bytes,omitempty"`
	BodyOpaqueBytes      int         `json:"body_opaque_bytes,omitempty"`
	HeaderDecodedPercent float64     `json:"header_decoded_percent,omitempty"`
	BodyDecodedPercent   float64     `json:"body_decoded_percent,omitempty"`
	BodyOps              map[int]int `json:"body_ops,omitempty"`
	ActionIDs            map[int]int `json:"action_ids,omitempty"`
	UnknownActionIDs     map[int]int `json:"unknown_action_ids,omitempty"`
	PostgameShape        string      `json:"postgame_shape,omitempty"`
	PostgameVerification string      `json:"postgame_verification,omitempty"`
	Warnings             []string    `json:"warnings,omitempty"`
}

type CorpusUnknownGroup struct {
	ActionID int            `json:"action_id"`
	Count    int            `json:"count"`
	Files    int            `json:"files"`
	Players  []int          `json:"players_seen,omitempty"`
	Shapes   []DeltaShape   `json:"payload_length_shapes,omitempty"`
	Samples  []CorpusSample `json:"samples,omitempty"`
	ByFile   map[string]int `json:"by_file,omitempty"`
}

type CorpusSample struct {
	Path string `json:"path"`
	Hex  string `json:"hex"`
}

type OpaqueClusterOptions struct {
	Recursive   bool
	IncludeZip  bool
	Space       string
	MinBytes    int
	Limit       int
	SampleLimit int
}

type OpaqueClusterReport struct {
	Folder       string               `json:"folder"`
	Method       string               `json:"method"`
	Verification string               `json:"verification"`
	Summary      OpaqueClusterSummary `json:"summary"`
	Clusters     []OpaqueCluster      `json:"clusters"`
	Warnings     []string             `json:"warnings,omitempty"`
}

type OpaqueClusterSummary struct {
	Files             int `json:"files"`
	Failures          int `json:"failures"`
	Spans             int `json:"spans"`
	Clusters          int `json:"clusters"`
	Shown             int `json:"shown"`
	TotalOpaqueBytes  int `json:"total_opaque_bytes"`
	HeaderOpaqueBytes int `json:"header_opaque_bytes"`
	BodyOpaqueBytes   int `json:"body_opaque_bytes"`
}

type OpaqueCluster struct {
	Key                 string          `json:"key"`
	Space               string          `json:"space"`
	NamePattern         string          `json:"name_pattern"`
	PrevPattern         string          `json:"prev_pattern,omitempty"`
	NextPattern         string          `json:"next_pattern,omitempty"`
	ByteBucket          string          `json:"byte_bucket"`
	Count               int             `json:"count"`
	Files               int             `json:"files"`
	TotalBytes          int             `json:"total_bytes"`
	MinBytes            int             `json:"min_bytes"`
	MaxBytes            int             `json:"max_bytes"`
	AvgBytes            float64         `json:"avg_bytes"`
	AvgEntropy          float64         `json:"avg_entropy"`
	AvgZeroPercent      float64         `json:"avg_zero_percent"`
	AvgFFPercent        float64         `json:"avg_ff_percent"`
	AvgPrintablePercent float64         `json:"avg_printable_percent"`
	ShapeHints          []string        `json:"shape_hints,omitempty"`
	Samples             []ClusterSample `json:"samples,omitempty"`
}

type ClusterSample struct {
	Path      string `json:"path"`
	SpanName  string `json:"span_name"`
	Start     int    `json:"start"`
	End       int    `json:"end"`
	Bytes     int    `json:"bytes"`
	HexSample string `json:"hex_sample,omitempty"`
}

func BuildReplayCorpus(folder string, opts CorpusOptions) (*ReplayCorpusReport, error) {
	paths, warnings, err := collectReplayPaths(folder, opts.Recursive, opts.IncludeZip)
	if err != nil {
		return nil, err
	}
	report := &ReplayCorpusReport{
		Folder:       folder,
		Method:       "folder_replay_corpus_coverage_events_postgame",
		Verification: "structure_verified_not_engine_verified",
		Summary: ReplayCorpusSummary{
			BodyOps:        map[int]int{},
			ActionIDs:      map[int]int{},
			PostgameShapes: map[string]int{},
		},
		Warnings: warnings,
	}
	unknowns := map[int]*corpusUnknownAgg{}
	var headerPctSum float64
	var bodyPctSum float64
	for _, path := range paths {
		row := buildReplayCorpusRow(path, opts, unknowns)
		report.Rows = append(report.Rows, row)
		report.Summary.Scanned++
		if !row.OK {
			report.Summary.Failures++
			continue
		}
		if row.ReplayKind == "scenario" {
			report.Summary.ScenarioReplays++
		} else {
			report.Summary.NonScenarioReplays++
		}
		if row.ScenarioIdentityTier == "trigger_graph" {
			report.Summary.TriggerGraphs++
		} else if row.ScenarioIdentityTier != "" {
			report.Summary.FallbackFingerprints++
		}
		report.Summary.Players += row.PlayerCount
		report.Summary.Humans += row.HumanCount
		report.Summary.AIs += row.AICount
		report.Summary.TotalActions += row.Actions
		report.Summary.UntypedActions += row.UntypedActions
		report.Summary.UnknownOps += row.UnknownOps
		report.Summary.Chat += row.Chat
		report.Summary.Taunts += row.Taunts
		report.Summary.Flares += row.Flares
		report.Summary.Resigns += row.Resigns
		report.Summary.Postgames += row.Postgames
		report.Summary.HeaderBytes += row.HeaderBytes
		report.Summary.BodyBytes += row.BodyBytes
		report.Summary.HeaderOpaqueBytes += row.HeaderOpaqueBytes
		report.Summary.BodyOpaqueBytes += row.BodyOpaqueBytes
		headerPctSum += row.HeaderDecodedPercent
		bodyPctSum += row.BodyDecodedPercent
		if report.Summary.HeaderDecodedPercentMin == 0 || row.HeaderDecodedPercent < report.Summary.HeaderDecodedPercentMin {
			report.Summary.HeaderDecodedPercentMin = row.HeaderDecodedPercent
		}
		if report.Summary.BodyDecodedPercentMin == 0 || row.BodyDecodedPercent < report.Summary.BodyDecodedPercentMin {
			report.Summary.BodyDecodedPercentMin = row.BodyDecodedPercent
		}
		for op, count := range row.BodyOps {
			report.Summary.BodyOps[op] += count
		}
		for id, count := range row.ActionIDs {
			report.Summary.ActionIDs[id] += count
		}
		if row.PostgameShape != "" {
			report.Summary.PostgameShapes[row.PostgameShape]++
		}
	}
	okRows := report.Summary.Scanned - report.Summary.Failures
	if okRows > 0 {
		report.Summary.HeaderDecodedPercentAvg = round2(headerPctSum / float64(okRows))
		report.Summary.BodyDecodedPercentAvg = round2(bodyPctSum / float64(okRows))
	}
	if len(report.Summary.BodyOps) == 0 {
		report.Summary.BodyOps = nil
	}
	if len(report.Summary.ActionIDs) == 0 {
		report.Summary.ActionIDs = nil
	}
	if len(report.Summary.PostgameShapes) == 0 {
		report.Summary.PostgameShapes = nil
	}
	report.Unknowns = flattenCorpusUnknowns(unknowns)
	return report, nil
}

func buildReplayCorpusRow(path string, opts CorpusOptions, unknowns map[int]*corpusUnknownAgg) ReplayCorpusRow {
	row := ReplayCorpusRow{Path: path}
	cache, err := OpenReplayCache(path)
	if err != nil {
		row.Error = err.Error()
		return row
	}
	row.FileBytes = int(cache.Manifest.RecordBytes)
	rec, err := cache.Metadata()
	if err != nil {
		row.Error = err.Error()
		return row
	}
	row.OK = true
	row.GameVersion = rec.GameVersion
	row.SaveVersion = rec.SaveVersion
	row.LogVersion = rec.LogVersion
	row.DataSetStatus = rec.DataSet.Status
	row.ActiveDataSet = rec.DataSet.ActiveDataSet
	row.ReplayKind = "non_scenario"
	if rec.TriggerGraph != nil {
		row.ReplayKind = "scenario"
		row.ScenarioIdentity = rec.TriggerGraph.SHA256
		row.ScenarioIdentityTier = "trigger_graph"
		if rec.TriggerGraphTier == "triggergraph_bounded_candidate" {
			row.ScenarioIdentityTier = rec.TriggerGraphTier
			row.Warnings = append(row.Warnings, "bounded trigger candidate; complete scenario identity not established")
		}
		row.TriggerCount = rec.TriggerGraph.TriggerCount
		row.EffectCount = rec.TriggerGraph.EffectCount
		row.ConditionCount = rec.TriggerGraph.ConditionCount
		row.MessageCount = rec.TriggerGraph.MessageCount
	} else if rec.Fallback != nil {
		row.ScenarioIdentity = rec.Fallback.SHA256
		row.ScenarioIdentityTier = rec.Fallback.Tier
		if rec.Fallback.Map != nil {
			row.MapWidth = rec.Fallback.Map.Width
			row.MapHeight = rec.Fallback.Map.Height
		}
	}
	for _, player := range rec.Players {
		if !player.Active || player.Number <= 0 {
			continue
		}
		row.PlayerCount++
		if player.Human {
			row.HumanCount++
		} else {
			row.AICount++
		}
		if player.Name != "" {
			row.PlayerNames = append(row.PlayerNames, player.Name)
		}
	}
	{
		events, _, counts, warnings, eventErr := extractCachedEvents(cache, EventOptions{})
		if eventErr != nil {
			row.Error = eventErr.Error()
			row.OK = false
			return row
		}
		counts = finalizeEventCounts(counts, events)
		row.DurationMS = counts.DurationMS
		row.Duration = counts.Duration
		row.Actions = counts.Actions
		row.UntypedActions = counts.UntypedActions
		row.UnknownOps = counts.UnknownOps
		row.Chat = counts.Chat
		row.Taunts = counts.Taunts
		row.Flares = counts.Flares
		row.Resigns = counts.Resigns
		row.Postgames = counts.Postgames
		row.ActionIDs = cloneIntMap(counts.ActionIDs)
		row.Warnings = append(row.Warnings, warnings...)
		ids := map[int]int{}
		err = cache.WalkOps(func(op CachedOp) error {
			if op.ID != 1 || actionName(op.ActionID) != "" {
				return nil
			}
			ev, ok := DecodeActionEvent(op.ActionID, op.Payload, op.TimeMS, op.Offset, false)
			if !ok {
				ev = ReplayEvent{ReplayAction: true, ActionID: op.ActionID, PayloadBytes: len(op.Payload)}
			}
			if opts.UnknownSamples > 0 {
				ev.RawHex = hex.EncodeToString(op.Payload[:min(len(op.Payload), 4096)])
			}
			for id, n := range collectCorpusUnknownActions(path, []ReplayEvent{ev}, unknowns, min(opts.UnknownSamples, 100)) {
				ids[id] += n
			}
			return nil
		})
		if err != nil {
			row.Error = err.Error()
			row.OK = false
			return row
		}
		if len(ids) > 0 {
			row.UnknownActionIDs = ids
		}
	}
	var coverage *CoverageReport
	if cache.Manifest.HeaderBytes > headerMetadataWindow || cache.Manifest.RecordBytes > 32<<20 {
		err = fmt.Errorf("legacy detailed coverage deferred for large recording; cached body statistics remain available")
		row.HeaderBytes = int(cache.Manifest.HeaderBytes)
		row.BodyBytes = int(cache.Manifest.RecordBytes - cache.Manifest.HeaderLength)
	} else {
		coverage, err = BuildCoverage(path)
	}
	if err != nil {
		row.Warnings = append(row.Warnings, "coverage unavailable: "+err.Error())
	} else {
		row.HeaderBytes = coverage.Summary.InflatedHeaderBytes
		row.BodyBytes = coverage.Summary.BodyBytes
		row.HeaderOpaqueBytes = coverage.Summary.HeaderOpaqueBytes
		row.BodyOpaqueBytes = coverage.Summary.BodyOpaqueBytes
		row.HeaderDecodedPercent = coverage.Summary.HeaderDecodedPercent
		row.BodyDecodedPercent = coverage.Summary.BodyDecodedPercent
		row.BodyOps = cloneIntMap(coverage.BodyOps)
		row.Warnings = append(row.Warnings, coverage.Warnings...)
	}
	postgame, err := BuildPostgame(path)
	if err != nil {
		row.Warnings = append(row.Warnings, "postgame unavailable: "+err.Error())
	} else {
		row.PostgameShape = postgameShape(postgame)
		row.PostgameVerification = postgame.Verification
	}
	return row
}

func BuildOpaqueClusters(folder string, opts OpaqueClusterOptions) (*OpaqueClusterReport, error) {
	paths, warnings, err := collectReplayPaths(folder, opts.Recursive, opts.IncludeZip)
	if err != nil {
		return nil, err
	}
	report := &OpaqueClusterReport{
		Folder:       folder,
		Method:       "folder_opaque_span_structural_clustering",
		Verification: "structure_verified_opaque_bytes_profiled_not_semantically_decoded",
		Warnings:     warnings,
	}
	clusters := map[string]*opaqueClusterAgg{}
	for _, path := range paths {
		spanReport, err := BuildOpaqueSpans(path, OpaqueSpanOptions{Space: opts.Space, MinBytes: opts.MinBytes})
		if err != nil {
			report.Summary.Failures++
			report.Warnings = append(report.Warnings, fmt.Sprintf("%s: %v", path, err))
			continue
		}
		report.Summary.Files++
		for _, span := range spanReport.Spans {
			report.Summary.Spans++
			report.Summary.TotalOpaqueBytes += span.Bytes
			if span.Space == "inflated_header" {
				report.Summary.HeaderOpaqueBytes += span.Bytes
			} else if span.Space == "body" {
				report.Summary.BodyOpaqueBytes += span.Bytes
			}
			key := opaqueClusterKey(span)
			agg := clusters[key]
			if agg == nil {
				agg = &opaqueClusterAgg{
					cluster: OpaqueCluster{
						Key:         key,
						Space:       span.Space,
						NamePattern: normalizeReplaySpanLabel(span.Name),
						PrevPattern: normalizeReplaySpanLabel(span.PrevRegion),
						NextPattern: normalizeReplaySpanLabel(span.NextRegion),
						ByteBucket:  byteBucket(span.Bytes),
						MinBytes:    span.Bytes,
						MaxBytes:    span.Bytes,
					},
					files: map[string]bool{},
					hints: map[string]bool{},
				}
				clusters[key] = agg
			}
			agg.cluster.Count++
			agg.files[path] = true
			agg.cluster.TotalBytes += span.Bytes
			if span.Bytes < agg.cluster.MinBytes {
				agg.cluster.MinBytes = span.Bytes
			}
			if span.Bytes > agg.cluster.MaxBytes {
				agg.cluster.MaxBytes = span.Bytes
			}
			agg.entropySum += span.Entropy
			if span.Bytes > 0 {
				agg.zeroPctSum += float64(span.ZeroBytes) * 100 / float64(span.Bytes)
				agg.ffPctSum += float64(span.FFBytes) * 100 / float64(span.Bytes)
				agg.printablePctSum += float64(span.PrintableBytes) * 100 / float64(span.Bytes)
			}
			for _, hint := range span.ShapeHints {
				agg.hints[hint] = true
			}
			limit := opts.SampleLimit
			if limit <= 0 {
				limit = 2
			}
			if len(agg.cluster.Samples) < limit {
				agg.cluster.Samples = append(agg.cluster.Samples, ClusterSample{
					Path:      path,
					SpanName:  span.Name,
					Start:     span.Start,
					End:       span.End,
					Bytes:     span.Bytes,
					HexSample: span.HexSample,
				})
			}
		}
	}
	for _, agg := range clusters {
		c := agg.cluster
		c.Files = len(agg.files)
		if c.Count > 0 {
			c.AvgBytes = round2(float64(c.TotalBytes) / float64(c.Count))
			c.AvgEntropy = corpusRound3(agg.entropySum / float64(c.Count))
			c.AvgZeroPercent = round2(agg.zeroPctSum / float64(c.Count))
			c.AvgFFPercent = round2(agg.ffPctSum / float64(c.Count))
			c.AvgPrintablePercent = round2(agg.printablePctSum / float64(c.Count))
		}
		for hint := range agg.hints {
			c.ShapeHints = append(c.ShapeHints, hint)
		}
		sort.Strings(c.ShapeHints)
		report.Clusters = append(report.Clusters, c)
	}
	sort.Slice(report.Clusters, func(i, j int) bool {
		if report.Clusters[i].TotalBytes == report.Clusters[j].TotalBytes {
			if report.Clusters[i].Files == report.Clusters[j].Files {
				return report.Clusters[i].Key < report.Clusters[j].Key
			}
			return report.Clusters[i].Files > report.Clusters[j].Files
		}
		return report.Clusters[i].TotalBytes > report.Clusters[j].TotalBytes
	})
	report.Summary.Clusters = len(report.Clusters)
	if opts.Limit > 0 && len(report.Clusters) > opts.Limit {
		report.Clusters = report.Clusters[:opts.Limit]
	}
	report.Summary.Shown = len(report.Clusters)
	return report, nil
}

type corpusUnknownAgg struct {
	info    CorpusUnknownGroup
	files   map[string]bool
	players map[int]bool
	shapes  map[int]int
}

func collectCorpusUnknownActions(path string, events []ReplayEvent, out map[int]*corpusUnknownAgg, samples int) map[int]int {
	row := map[int]int{}
	for _, event := range events {
		if !event.ReplayAction {
			continue
		}
		if event.ActionName != "" || actionName(event.ActionID) != "" {
			continue
		}
		row[event.ActionID]++
		agg := out[event.ActionID]
		if agg == nil {
			agg = &corpusUnknownAgg{
				info: CorpusUnknownGroup{
					ActionID: event.ActionID,
					ByFile:   map[string]int{},
				},
				files:   map[string]bool{},
				players: map[int]bool{},
				shapes:  map[int]int{},
			}
			out[event.ActionID] = agg
		}
		agg.info.Count++
		agg.info.ByFile[path]++
		agg.files[path] = true
		if event.PlayerID != 0 {
			agg.players[event.PlayerID] = true
		}
		agg.shapes[event.PayloadBytes]++
		if samples > 0 && event.RawHex != "" && len(agg.info.Samples) < samples {
			agg.info.Samples = append(agg.info.Samples, CorpusSample{Path: path, Hex: event.RawHex})
		}
	}
	if len(row) == 0 {
		return nil
	}
	return row
}

func flattenCorpusUnknowns(groups map[int]*corpusUnknownAgg) []CorpusUnknownGroup {
	out := make([]CorpusUnknownGroup, 0, len(groups))
	for _, agg := range groups {
		info := agg.info
		info.Files = len(agg.files)
		for player := range agg.players {
			info.Players = append(info.Players, player)
		}
		sort.Ints(info.Players)
		for length, count := range agg.shapes {
			info.Shapes = append(info.Shapes, DeltaShape{DeltaMS: length, Count: count})
		}
		sort.Slice(info.Shapes, func(i, j int) bool {
			if info.Shapes[i].Count == info.Shapes[j].Count {
				return info.Shapes[i].DeltaMS < info.Shapes[j].DeltaMS
			}
			return info.Shapes[i].Count > info.Shapes[j].Count
		})
		out = append(out, info)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count == out[j].Count {
			return out[i].ActionID < out[j].ActionID
		}
		return out[i].Count > out[j].Count
	})
	return out
}

type opaqueClusterAgg struct {
	cluster         OpaqueCluster
	files           map[string]bool
	hints           map[string]bool
	entropySum      float64
	zeroPctSum      float64
	ffPctSum        float64
	printablePctSum float64
}

func collectReplayPaths(root string, recursive bool, includeZip bool) ([]string, []string, error) {
	info, err := os.Stat(root)
	if err != nil {
		return nil, nil, err
	}
	if !info.IsDir() {
		if isReplayCorpusPath(root, includeZip) {
			return []string{root}, nil, nil
		}
		return nil, nil, fmt.Errorf("%s is not a replay path", root)
	}
	var paths []string
	var warnings []string
	walkFn := func(path string, d os.DirEntry, err error) error {
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("%s: %v", path, err))
			return nil
		}
		if path == root {
			return nil
		}
		if d.IsDir() {
			if !recursive {
				return filepath.SkipDir
			}
			return nil
		}
		if isReplayCorpusPath(path, includeZip) {
			paths = append(paths, path)
		}
		return nil
	}
	if err := filepath.WalkDir(root, walkFn); err != nil {
		return nil, warnings, err
	}
	sort.Strings(paths)
	return paths, warnings, nil
}

func isReplayCorpusPath(path string, includeZip bool) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".aoe2record" || includeZip && ext == ".zip"
}

func postgameShape(report *PostgameReport) string {
	if report == nil {
		return ""
	}
	if report.DE == nil {
		if report.Summary.Action255Seen {
			return fmt.Sprintf("action255_payloads_%d", report.Summary.Action255Payloads)
		}
		return "none"
	}
	parts := []string{fmt.Sprintf("op6_tail_%dB", report.Summary.Op6TailBytes)}
	if report.Summary.HasWorldTimeBlock {
		parts = append(parts, "world_time")
	}
	if report.Summary.HasLeaderboardBlock {
		parts = append(parts, "leaderboard")
	}
	if report.Summary.HasPlayerKills {
		parts = append(parts, "player_kills")
	}
	if len(report.DE.Blocks) > 0 {
		var ids []string
		for _, block := range report.DE.Blocks {
			ids = append(ids, fmt.Sprintf("%d:%d", block.ID, block.Length))
		}
		parts = append(parts, "blocks="+strings.Join(ids, ","))
	}
	return strings.Join(parts, "|")
}

func cloneIntMap(in map[int]int) map[int]int {
	if len(in) == 0 {
		return nil
	}
	out := make(map[int]int, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

var replayDigits = regexp.MustCompile(`\d+`)

func opaqueClusterKey(span OpaqueSpan) string {
	parts := []string{
		span.Space,
		normalizeReplaySpanLabel(span.Name),
		normalizeReplaySpanLabel(span.PrevRegion),
		normalizeReplaySpanLabel(span.NextRegion),
		byteBucket(span.Bytes),
		strings.Join(span.ShapeHints, ","),
	}
	sum := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return hex.EncodeToString(sum[:8])
}

func normalizeReplaySpanLabel(label string) string {
	if label == "" {
		return ""
	}
	label = replayDigits.ReplaceAllString(label, "#")
	label = strings.Join(strings.Fields(label), " ")
	return label
}

func byteBucket(n int) string {
	switch {
	case n < 64:
		return "<64"
	case n < 256:
		return "64-255"
	case n < 1024:
		return "256-1023"
	case n < 4096:
		return "1K-4K"
	case n < 16384:
		return "4K-16K"
	case n < 65536:
		return "16K-64K"
	case n < 262144:
		return "64K-256K"
	case n < 1048576:
		return "256K-1M"
	default:
		return "1M+"
	}
}

func corpusRound3(v float64) float64 {
	if v < 0 {
		return -corpusRound3(-v)
	}
	return float64(int(v*1000+0.5)) / 1000
}
