package replay

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"

	"aoe2kit/pkg/aoe2"
)

type FrontierOptions struct {
	MinDuplicateBytes int
	Limit             int
}

type FrontierReport struct {
	Path         string                 `json:"path,omitempty"`
	Verification aoe2.VerificationClaim `json:"verification"`
	Summary      FrontierSummary        `json:"summary"`
	Buckets      []FrontierBucket       `json:"buckets"`
	Duplicates   []FrontierDuplicate    `json:"duplicates,omitempty"`
	TemplateHits []FrontierTemplateHit  `json:"template_hits,omitempty"`
	Warnings     []string               `json:"warnings,omitempty"`
}

type FrontierSummary struct {
	InflatedHeaderBytes int     `json:"inflated_header_bytes"`
	HeaderDecodedBytes  int     `json:"header_decoded_bytes"`
	HeaderOpaqueBytes   int     `json:"header_opaque_bytes"`
	HeaderDecodedPct    float64 `json:"header_decoded_percent"`
	BodyBytes           int     `json:"body_bytes"`
	BodyOpaqueBytes     int     `json:"body_opaque_bytes"`
	OpaqueSpans         int     `json:"opaque_spans"`
	DuplicateGroups     int     `json:"duplicate_groups"`
	ShownDuplicates     int     `json:"shown_duplicates"`
	DuplicateBytes      int     `json:"duplicate_repeated_bytes"`
	TemplateHitCount    int     `json:"template_hit_count"`
	ShownTemplateHits   int     `json:"shown_template_hits"`
	TemplateHitBytes    int     `json:"template_hit_bytes"`
}

type FrontierBucket struct {
	Name             string       `json:"name"`
	Bytes            int          `json:"bytes"`
	Spans            int          `json:"spans"`
	Percent          float64      `json:"percent_of_header_opaque"`
	TemplateHitBytes int          `json:"template_hit_bytes,omitempty"`
	ResidualBytes    int          `json:"residual_bytes"`
	Confidence       string       `json:"confidence"`
	Hypothesis       string       `json:"hypothesis"`
	NextMove         string       `json:"next_move"`
	Largest          []OpaqueSpan `json:"largest,omitempty"`
	TextSamples      []string     `json:"text_samples,omitempty"`
}

type FrontierDuplicate struct {
	SHA256        string                  `json:"sha256"`
	Bytes         int                     `json:"bytes"`
	Count         int                     `json:"count"`
	RepeatedBytes int                     `json:"repeated_bytes"`
	Buckets       []string                `json:"buckets,omitempty"`
	Spans         []FrontierDuplicateSpan `json:"spans"`
	HexSample     string                  `json:"hex_sample,omitempty"`
	TextSamples   []string                `json:"text_samples,omitempty"`
}

type FrontierDuplicateSpan struct {
	Space  string `json:"space"`
	Name   string `json:"name"`
	Start  int    `json:"start"`
	End    int    `json:"end"`
	Bucket string `json:"bucket"`
}

type FrontierTemplateHit struct {
	Space              string `json:"space"`
	Name               string `json:"name"`
	Start              int    `json:"start"`
	End                int    `json:"end"`
	Bytes              int    `json:"bytes"`
	Bucket             string `json:"bucket"`
	SourceRegion       string `json:"source_region"`
	SourceStart        int    `json:"source_start"`
	SourceEnd          int    `json:"source_end"`
	SourceRelative     int    `json:"source_relative_offset"`
	Confidence         string `json:"confidence"`
	MatchedPayloadSHA  string `json:"matched_payload_sha256"`
	MatchedPayloadHead string `json:"matched_payload_hex_sample,omitempty"`
}

type frontierSpan struct {
	span   OpaqueSpan
	bucket string
	data   []byte
}

type frontierTemplateSource struct {
	region ReplayRegion
	data   []byte
}

func BuildFrontier(path string, opts FrontierOptions) (*FrontierReport, error) {
	if opts.MinDuplicateBytes <= 0 {
		opts.MinDuplicateBytes = 256
	}
	if opts.Limit < 0 {
		opts.Limit = 0
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
	header := rec.HeaderBytes()
	body := replayBody(data, rec.HeaderLength)
	width, height := mapDimensionsFromCoverage(coverage)
	triggerCount := 0
	if rec.TriggerGraph != nil {
		triggerCount = rec.TriggerGraph.TriggerCount
	}
	neighbors := frontierRegionNeighbors(coverage.Regions)
	templateSources := frontierTemplateSources(coverage.Regions, header)
	report := &FrontierReport{
		Path: path,
		Verification: aoe2.VerificationClaim{
			Label:             "structure_verified_not_engine_verified",
			StructureVerified: true,
			EngineVerified:    false,
			Note:              "Frontier buckets are derived from decoded coverage regions and byte profiles; bucket names are hypotheses until a field parser claims the bytes.",
		},
		Summary: FrontierSummary{
			InflatedHeaderBytes: coverage.Summary.InflatedHeaderBytes,
			HeaderDecodedBytes:  coverage.Summary.HeaderDecodedBytes,
			HeaderOpaqueBytes:   coverage.Summary.HeaderOpaqueBytes,
			HeaderDecodedPct:    coverage.Summary.HeaderDecodedPercent,
			BodyBytes:           coverage.Summary.BodyBytes,
			BodyOpaqueBytes:     coverage.Summary.BodyOpaqueBytes,
		},
		Warnings: append([]string(nil), coverage.Warnings...),
	}
	buckets := map[string]*FrontierBucket{}
	var spans []frontierSpan
	for _, region := range coverage.Regions {
		if region.Status == "decoded" {
			continue
		}
		payload := bytesForRegion(region, header, body)
		neighbor := neighbors[frontierRegionKey(region)]
		span := profileFrontierSpan(region, payload, neighbor.prev, neighbor.next, width, height, triggerCount)
		bucketName := frontierBucketFor(span)
		meta := frontierBucketMeta(bucketName)
		bucket := buckets[bucketName]
		if bucket == nil {
			bucket = &FrontierBucket{
				Name:       bucketName,
				Confidence: meta.confidence,
				Hypothesis: meta.hypothesis,
				NextMove:   meta.nextMove,
			}
			buckets[bucketName] = bucket
		}
		bucket.Bytes += span.Bytes
		bucket.Spans++
		bucket.TextSamples = mergeTextSamples(bucket.TextSamples, span.TextSamples, 6)
		bucket.Largest = append(bucket.Largest, span)
		spans = append(spans, frontierSpan{span: span, bucket: bucketName, data: payload})
	}
	for _, bucket := range buckets {
		sort.Slice(bucket.Largest, func(i, j int) bool {
			if bucket.Largest[i].Bytes == bucket.Largest[j].Bytes {
				return bucket.Largest[i].Start < bucket.Largest[j].Start
			}
			return bucket.Largest[i].Bytes > bucket.Largest[j].Bytes
		})
		if opts.Limit > 0 && len(bucket.Largest) > opts.Limit {
			bucket.Largest = bucket.Largest[:opts.Limit]
		}
		if report.Summary.HeaderOpaqueBytes > 0 {
			bucket.Percent = round2(float64(bucket.Bytes) * 100 / float64(report.Summary.HeaderOpaqueBytes))
		}
		report.Buckets = append(report.Buckets, *bucket)
	}
	sort.Slice(report.Buckets, func(i, j int) bool {
		if report.Buckets[i].Bytes == report.Buckets[j].Bytes {
			return report.Buckets[i].Name < report.Buckets[j].Name
		}
		return report.Buckets[i].Bytes > report.Buckets[j].Bytes
	})
	report.Summary.OpaqueSpans = len(spans)
	duplicates := frontierDuplicates(spans, opts.MinDuplicateBytes, 0)
	report.Summary.DuplicateGroups = len(duplicates)
	for _, group := range duplicates {
		report.Summary.DuplicateBytes += group.RepeatedBytes
	}
	report.Duplicates = limitFrontierDuplicates(duplicates, opts.Limit)
	report.Summary.ShownDuplicates = len(report.Duplicates)
	templateHits := frontierTemplateHits(spans, templateSources, opts.MinDuplicateBytes, 0)
	report.Summary.TemplateHitCount = len(templateHits)
	templateBytesByBucket := map[string]int{}
	for _, hit := range templateHits {
		report.Summary.TemplateHitBytes += hit.Bytes
		templateBytesByBucket[hit.Bucket] += hit.Bytes
	}
	for i := range report.Buckets {
		report.Buckets[i].TemplateHitBytes = templateBytesByBucket[report.Buckets[i].Name]
		report.Buckets[i].ResidualBytes = report.Buckets[i].Bytes - report.Buckets[i].TemplateHitBytes
	}
	report.TemplateHits = limitFrontierTemplateHits(templateHits, opts.Limit)
	report.Summary.ShownTemplateHits = len(report.TemplateHits)
	return report, nil
}

type frontierNeighbor struct {
	prev string
	next string
}

func frontierRegionNeighbors(regions []ReplayRegion) map[string]frontierNeighbor {
	out := map[string]frontierNeighbor{}
	bySpace := map[string][]ReplayRegion{}
	for _, region := range regions {
		bySpace[region.Space] = append(bySpace[region.Space], region)
	}
	for _, list := range bySpace {
		sort.Slice(list, func(i, j int) bool {
			if list[i].Start == list[j].Start {
				return list[i].End < list[j].End
			}
			return list[i].Start < list[j].Start
		})
		for i, region := range list {
			neighbor := frontierNeighbor{}
			if i > 0 {
				neighbor.prev = regionLabel(&list[i-1])
			}
			if i+1 < len(list) {
				neighbor.next = regionLabel(&list[i+1])
			}
			out[frontierRegionKey(region)] = neighbor
		}
	}
	return out
}

func frontierRegionKey(region ReplayRegion) string {
	return region.Space + ":" + region.Name + ":" + itoaReplay(region.Start) + ":" + itoaReplay(region.End)
}

func profileFrontierSpan(region ReplayRegion, data []byte, prev string, next string, mapWidth int, mapHeight int, triggerCount int) OpaqueSpan {
	profile := byteProfile(data)
	return OpaqueSpan{
		Space:          region.Space,
		Name:           region.Name,
		Start:          region.Start,
		End:            region.End,
		Bytes:          region.Bytes,
		Status:         region.Status,
		Confidence:     region.Confidence,
		Entropy:        profile.entropy,
		DistinctBytes:  profile.distinct,
		ZeroBytes:      profile.zeros,
		FFBytes:        profile.ffs,
		PrintableBytes: profile.printable,
		LongestZeroRun: profile.longestZeroRun,
		LongestFFRun:   profile.longestFFRun,
		HexSample:      HexSample(data, 96),
		TextSamples:    textSamples(data, 8),
		PrevRegion:     prev,
		NextRegion:     next,
		ShapeHints:     shapeHints(region.Bytes, mapWidth, mapHeight, triggerCount),
	}
}

type frontierBucketInfo struct {
	confidence string
	hypothesis string
	nextMove   string
}

func frontierBucketFor(span OpaqueSpan) string {
	name := span.Name
	switch {
	case span.Space == "body":
		return "body_untyped_or_gap"
	case strings.Contains(name, "_object_candidate_body_"):
		return "referenced_object_body_remainder"
	case strings.Contains(name, "_object_tail_before_candidate_prefixes"):
		return "player_object_tail_before_candidates"
	case strings.Contains(name, "_object_tail_after_candidate_prefixes"):
		return "player_object_tail_after_candidates"
	case strings.Contains(name, "_effective_gamedata_diff_run_"):
		return "effective_gamedata_civ_diff"
	case strings.Contains(name, "v68_tail_before_trigger_graph"):
		if opaqueSpanHasText(span, "(include ") || opaqueSpanHasText(span, ".xs") || neighborNamesScriptContainer(span) {
			return "ai_or_xs_script_container_gap"
		}
		return "v68_pre_trigger_tail"
	case strings.Contains(name, "trigger_graph"):
		return "trigger_graph_remainder"
	case strings.Contains(name, "inflated_header_gap_unaccounted"):
		return "unframed_header_gap"
	default:
		return "other_header_save_state"
	}
}

func frontierBucketMeta(name string) frontierBucketInfo {
	switch name {
	case "referenced_object_body_remainder":
		return frontierBucketInfo{"high_boundary_low_semantics", "coverage-visible gaps inside replay-referenced object records, often between decoded DE string islands and partially parsed v68 object tails", "align duplicate object-body gaps against v68 pre-trigger twins, then promote repeated subrecords into the object-body parser"}
	case "player_object_tail_before_candidates":
		return frontierBucketInfo{"framed_not_semantic", "object/save-state payload before the first conservative object prefix in each player span", "use diagnostic object fixtures and duplicate matching to find earlier object prefixes or count-prefixed sublists"}
	case "player_object_tail_after_candidates":
		return frontierBucketInfo{"framed_not_semantic", "object/save-state payload between or after candidate object records, often repeated with v68 tail spans", "mine exact duplicates and align them to known unit/building object shapes"}
	case "effective_gamedata_civ_diff":
		return frontierBucketInfo{"bounded_diff_semantics_partial", "per-player effective data differs from the shared template, likely civ/unit/tech availability and attributes", "extend dat cross-reference beyond unit availability into tech/research/effect arrays"}
	case "v68_pre_trigger_tail":
		return frontierBucketInfo{"bounded_not_semantic", "global v68 save-state tail immediately before the trigger graph; contains repeated object/default-state blocks and DE strings", "parse by duplicate alignment against object-tail spans and string-island boundaries"}
	case "ai_or_xs_script_container_gap":
		return frontierBucketInfo{"text_anchor_semantics_partial", "script/loadout container bytes around parsed AI/XS text anchors", "split the wrapper around include/load text and name the count/length fields"}
	case "trigger_graph_remainder":
		return frontierBucketInfo{"unexpected", "bytes adjacent to decoded trigger graph that were not claimed", "verify trigger parser section bounds"}
	case "unframed_header_gap":
		return frontierBucketInfo{"low_boundary", "header bytes not yet assigned to a named section", "add section framing before field decoding"}
	case "body_untyped_or_gap":
		return frontierBucketInfo{"body_gap", "action-stream bytes not covered by typed body parser", "decode the command op shape or add a bounded opaque action parser"}
	default:
		return frontierBucketInfo{"unknown", "header save-state bytes outside current named buckets", "profile with controlled replay diffs and exact duplicate matching"}
	}
}

func opaqueSpanHasText(span OpaqueSpan, text string) bool {
	for _, sample := range span.TextSamples {
		if strings.Contains(sample, text) {
			return true
		}
	}
	return false
}

func neighborNamesScriptContainer(span OpaqueSpan) bool {
	return strings.Contains(span.PrevRegion, "_ai_load_") ||
		strings.Contains(span.NextRegion, "_ai_load_") ||
		strings.Contains(span.PrevRegion, "_xs_include_") ||
		strings.Contains(span.NextRegion, "_xs_include_")
}

func mergeTextSamples(existing []string, incoming []string, limit int) []string {
	if limit <= 0 {
		return nil
	}
	seen := map[string]bool{}
	out := make([]string, 0, limit)
	for _, sample := range existing {
		if sample == "" || seen[sample] {
			continue
		}
		out = append(out, sample)
		seen[sample] = true
		if len(out) >= limit {
			return out
		}
	}
	for _, sample := range incoming {
		if sample == "" || seen[sample] {
			continue
		}
		out = append(out, sample)
		seen[sample] = true
		if len(out) >= limit {
			return out
		}
	}
	return out
}

func frontierDuplicates(spans []frontierSpan, minBytes int, limit int) []FrontierDuplicate {
	type bucket struct {
		hash  string
		data  []byte
		spans []frontierSpan
	}
	byHash := map[string]*bucket{}
	for _, span := range spans {
		if len(span.data) < minBytes {
			continue
		}
		sum := sha256.Sum256(span.data)
		hash := hex.EncodeToString(sum[:])
		entry := byHash[hash]
		if entry == nil {
			entry = &bucket{hash: hash, data: span.data}
			byHash[hash] = entry
		}
		entry.spans = append(entry.spans, span)
	}
	var out []FrontierDuplicate
	for _, entry := range byHash {
		if len(entry.spans) < 2 {
			continue
		}
		bucketNames := map[string]bool{}
		group := FrontierDuplicate{
			SHA256:        entry.hash,
			Bytes:         len(entry.data),
			Count:         len(entry.spans),
			RepeatedBytes: len(entry.data) * (len(entry.spans) - 1),
			HexSample:     HexSample(entry.data, 64),
			TextSamples:   textSamples(entry.data, 4),
		}
		for _, span := range entry.spans {
			bucketNames[span.bucket] = true
			group.Spans = append(group.Spans, FrontierDuplicateSpan{
				Space:  span.span.Space,
				Name:   span.span.Name,
				Start:  span.span.Start,
				End:    span.span.End,
				Bucket: span.bucket,
			})
		}
		for name := range bucketNames {
			group.Buckets = append(group.Buckets, name)
		}
		sort.Strings(group.Buckets)
		sort.Slice(group.Spans, func(i, j int) bool {
			if group.Spans[i].Space == group.Spans[j].Space {
				return group.Spans[i].Start < group.Spans[j].Start
			}
			return group.Spans[i].Space < group.Spans[j].Space
		})
		out = append(out, group)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].RepeatedBytes == out[j].RepeatedBytes {
			return out[i].SHA256 < out[j].SHA256
		}
		return out[i].RepeatedBytes > out[j].RepeatedBytes
	})
	if limit > 0 && len(out) > limit {
		return out[:limit]
	}
	return out
}

func limitFrontierDuplicates(values []FrontierDuplicate, limit int) []FrontierDuplicate {
	if limit > 0 && len(values) > limit {
		return values[:limit]
	}
	return values
}

func frontierTemplateSources(regions []ReplayRegion, header []byte) []frontierTemplateSource {
	var out []frontierTemplateSource
	for _, region := range regions {
		if region.Space != "inflated_header" || region.Status != "decoded" {
			continue
		}
		if !strings.Contains(region.Name, "_effective_gamedata_template_reference") &&
			!strings.Contains(region.Name, "_effective_gamedata_template_common_") {
			continue
		}
		if region.Start < 0 || region.End > len(header) || region.Start >= region.End {
			continue
		}
		out = append(out, frontierTemplateSource{region: region, data: header[region.Start:region.End]})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].region.Name == out[j].region.Name {
			return out[i].region.Start < out[j].region.Start
		}
		if strings.Contains(out[i].region.Name, "_effective_gamedata_template_reference") != strings.Contains(out[j].region.Name, "_effective_gamedata_template_reference") {
			return strings.Contains(out[i].region.Name, "_effective_gamedata_template_reference")
		}
		return out[i].region.Start < out[j].region.Start
	})
	return out
}

func frontierTemplateHits(spans []frontierSpan, sources []frontierTemplateSource, minBytes int, limit int) []FrontierTemplateHit {
	if len(sources) == 0 {
		return nil
	}
	var out []FrontierTemplateHit
	for _, span := range spans {
		if len(span.data) < minBytes {
			continue
		}
		if span.bucket == "effective_gamedata_civ_diff" {
			continue
		}
		for _, source := range sources {
			rel := bytesIndex(source.data, span.data)
			if rel < 0 {
				continue
			}
			sum := sha256.Sum256(span.data)
			out = append(out, FrontierTemplateHit{
				Space:              span.span.Space,
				Name:               span.span.Name,
				Start:              span.span.Start,
				End:                span.span.End,
				Bytes:              span.span.Bytes,
				Bucket:             span.bucket,
				SourceRegion:       source.region.Name,
				SourceStart:        source.region.Start + rel,
				SourceEnd:          source.region.Start + rel + len(span.data),
				SourceRelative:     rel,
				Confidence:         "byte_identical_to_effective_gamedata_template_subrange",
				MatchedPayloadSHA:  hex.EncodeToString(sum[:]),
				MatchedPayloadHead: HexSample(span.data, 64),
			})
			break
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Bytes == out[j].Bytes {
			return out[i].Start < out[j].Start
		}
		return out[i].Bytes > out[j].Bytes
	})
	if limit > 0 && len(out) > limit {
		return out[:limit]
	}
	return out
}

func limitFrontierTemplateHits(values []FrontierTemplateHit, limit int) []FrontierTemplateHit {
	if limit > 0 && len(values) > limit {
		return values[:limit]
	}
	return values
}

func bytesIndex(data []byte, needle []byte) int {
	if len(needle) == 0 || len(needle) > len(data) {
		return -1
	}
	return bytes.Index(data, needle)
}
