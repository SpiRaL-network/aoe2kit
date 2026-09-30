package replay

import (
	"math"
	"sort"
	"strings"

	"aoe2kit/pkg/aoe2"
)

type OpaqueSpanOptions struct {
	Space    string
	MinBytes int
	Limit    int
}

type OpaqueSpanReport struct {
	Path         string                 `json:"path,omitempty"`
	Verification aoe2.VerificationClaim `json:"verification"`
	Summary      OpaqueSpanSummary      `json:"summary"`
	Spans        []OpaqueSpan           `json:"spans"`
	Warnings     []string               `json:"warnings,omitempty"`
}

type OpaqueSpanSummary struct {
	InflatedHeaderBytes int `json:"inflated_header_bytes"`
	BodyBytes           int `json:"body_bytes"`
	HeaderOpaqueBytes   int `json:"header_opaque_bytes"`
	BodyOpaqueBytes     int `json:"body_opaque_bytes"`
	SpanCount           int `json:"span_count"`
	Shown               int `json:"shown"`
}

type OpaqueSpan struct {
	Space          string   `json:"space"`
	Name           string   `json:"name"`
	Start          int      `json:"start"`
	End            int      `json:"end"`
	Bytes          int      `json:"bytes"`
	Status         string   `json:"status"`
	Confidence     string   `json:"confidence"`
	Entropy        float64  `json:"entropy"`
	DistinctBytes  int      `json:"distinct_bytes"`
	ZeroBytes      int      `json:"zero_bytes"`
	FFBytes        int      `json:"ff_bytes"`
	PrintableBytes int      `json:"printable_bytes"`
	LongestZeroRun int      `json:"longest_zero_run"`
	LongestFFRun   int      `json:"longest_ff_run"`
	HexSample      string   `json:"hex_sample,omitempty"`
	TextSamples    []string `json:"text_samples,omitempty"`
	PrevRegion     string   `json:"prev_region,omitempty"`
	NextRegion     string   `json:"next_region,omitempty"`
	ShapeHints     []string `json:"shape_hints,omitempty"`
}

func BuildOpaqueSpans(path string, opts OpaqueSpanOptions) (*OpaqueSpanReport, error) {
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
	body := []byte(nil)
	if rec.HeaderLength <= len(data) {
		body = data[rec.HeaderLength:]
	}
	width, height := mapDimensionsFromCoverage(coverage)
	triggerCount := 0
	if rec.TriggerGraph != nil {
		triggerCount = rec.TriggerGraph.TriggerCount
	}
	header := rec.HeaderBytes()
	report := &OpaqueSpanReport{
		Path: path,
		Verification: aoe2.VerificationClaim{
			Label:             "structure_verified_not_engine_verified",
			StructureVerified: true,
			EngineVerified:    false,
			Note:              "Opaque bytes are framed and profiled from the coverage spine; they are not semantically decoded.",
		},
		Summary: OpaqueSpanSummary{
			InflatedHeaderBytes: coverage.Summary.InflatedHeaderBytes,
			BodyBytes:           coverage.Summary.BodyBytes,
			HeaderOpaqueBytes:   coverage.Summary.HeaderOpaqueBytes,
			BodyOpaqueBytes:     coverage.Summary.BodyOpaqueBytes,
		},
		Warnings: append([]string(nil), coverage.Warnings...),
	}
	for _, region := range coverage.Regions {
		if region.Status == "decoded" {
			continue
		}
		if opts.Space != "" && region.Space != opts.Space {
			continue
		}
		if opts.MinBytes > 0 && region.Bytes < opts.MinBytes {
			continue
		}
		payload := bytesForRegion(region, header, body)
		span := profileOpaqueSpan(region, payload, coverage.Regions, width, height, triggerCount)
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
	report.Summary.SpanCount = len(report.Spans)
	if opts.Limit > 0 && len(report.Spans) > opts.Limit {
		report.Spans = report.Spans[:opts.Limit]
	}
	report.Summary.Shown = len(report.Spans)
	return report, nil
}

func bytesForRegion(region ReplayRegion, header []byte, body []byte) []byte {
	var source []byte
	switch region.Space {
	case "inflated_header":
		source = header
	case "body":
		source = body
	default:
		return nil
	}
	if region.Start < 0 || region.End < region.Start || region.Start > len(source) {
		return nil
	}
	end := region.End
	if end > len(source) {
		end = len(source)
	}
	return source[region.Start:end]
}

func profileOpaqueSpan(region ReplayRegion, data []byte, regions []ReplayRegion, mapWidth int, mapHeight int, triggerCount int) OpaqueSpan {
	profile := byteProfile(data)
	prev, next := neighborRegions(region, regions)
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

type spanByteProfile struct {
	entropy        float64
	distinct       int
	zeros          int
	ffs            int
	printable      int
	longestZeroRun int
	longestFFRun   int
}

func byteProfile(data []byte) spanByteProfile {
	var counts [256]int
	zeroRun := 0
	ffRun := 0
	out := spanByteProfile{}
	for _, b := range data {
		counts[b]++
		if b == 0 {
			out.zeros++
			zeroRun++
			if zeroRun > out.longestZeroRun {
				out.longestZeroRun = zeroRun
			}
		} else {
			zeroRun = 0
		}
		if b == 0xff {
			out.ffs++
			ffRun++
			if ffRun > out.longestFFRun {
				out.longestFFRun = ffRun
			}
		} else {
			ffRun = 0
		}
		if b == 9 || b == 10 || b == 13 || (b >= 32 && b <= 126) {
			out.printable++
		}
	}
	if len(data) == 0 {
		return out
	}
	for _, count := range counts {
		if count == 0 {
			continue
		}
		out.distinct++
		p := float64(count) / float64(len(data))
		out.entropy -= p * math.Log2(p)
	}
	out.entropy = math.Round(out.entropy*1000) / 1000
	return out
}

func neighborRegions(target ReplayRegion, regions []ReplayRegion) (string, string) {
	var prev *ReplayRegion
	var next *ReplayRegion
	for i := range regions {
		region := regions[i]
		if region.Space != target.Space || region.Name == target.Name && region.Start == target.Start && region.End == target.End {
			continue
		}
		if region.End <= target.Start {
			if prev == nil || region.End > prev.End {
				copy := region
				prev = &copy
			}
			continue
		}
		if region.Start >= target.End {
			if next == nil || region.Start < next.Start {
				copy := region
				next = &copy
			}
		}
	}
	return regionLabel(prev), regionLabel(next)
}

func regionLabel(region *ReplayRegion) string {
	if region == nil {
		return ""
	}
	return region.Name + " " + intRange(region.Start, region.End)
}

func intRange(start int, end int) string {
	return strings.Join([]string{itoaReplay(start), itoaReplay(end)}, "..")
}

func shapeHints(bytes int, mapWidth int, mapHeight int, triggerCount int) []string {
	if bytes <= 0 {
		return nil
	}
	var hints []string
	if mapWidth > 0 && mapHeight > 0 {
		tiles := mapWidth * mapHeight
		if bytes == tiles {
			hints = append(hints, "equals_map_tiles")
		}
		for _, unit := range []int{1, 2, 4, 8, 16} {
			if bytes == tiles*unit {
				hints = append(hints, "equals_map_tiles_x"+itoaReplay(unit))
			}
		}
		if tiles > 0 && bytes%tiles == 0 {
			hints = append(hints, "multiple_of_map_tiles_x"+itoaReplay(bytes/tiles))
		}
	}
	if triggerCount > 0 {
		for _, unit := range []int{1, 2, 4, 8, 16, 32} {
			if bytes == triggerCount*unit {
				hints = append(hints, "equals_trigger_count_x"+itoaReplay(unit))
			}
		}
		if bytes%triggerCount == 0 {
			hints = append(hints, "multiple_of_trigger_count_x"+itoaReplay(bytes/triggerCount))
		}
	}
	return hints
}

func textSamples(data []byte, limit int) []string {
	seen := map[string]bool{}
	var out []string
	for _, sample := range asciiTextSamples(data, limit) {
		if !seen[sample] {
			seen[sample] = true
			out = append(out, sample)
			if len(out) >= limit {
				return out
			}
		}
	}
	for _, sample := range utf16LETextSamples(data, limit-len(out)) {
		if !seen[sample] {
			seen[sample] = true
			out = append(out, sample)
			if len(out) >= limit {
				return out
			}
		}
	}
	return out
}

func asciiTextSamples(data []byte, limit int) []string {
	var out []string
	start := -1
	for i, b := range data {
		if b >= 32 && b <= 126 {
			if start < 0 {
				start = i
			}
			continue
		}
		if start >= 0 {
			out = appendTextRun(out, string(data[start:i]), limit)
			if len(out) >= limit {
				return out
			}
			start = -1
		}
	}
	if start >= 0 {
		out = appendTextRun(out, string(data[start:]), limit)
	}
	return out
}

func utf16LETextSamples(data []byte, limit int) []string {
	if limit <= 0 {
		return nil
	}
	var out []string
	var run []byte
	flush := func() {
		if len(run) >= 4 {
			out = appendTextRun(out, string(run), limit)
		}
		run = nil
	}
	for i := 0; i+1 < len(data); i += 2 {
		lo := data[i]
		hi := data[i+1]
		if hi == 0 && lo >= 32 && lo <= 126 {
			run = append(run, lo)
			continue
		}
		flush()
		if len(out) >= limit {
			return out
		}
	}
	flush()
	if len(out) > limit {
		return out[:limit]
	}
	return out
}

func appendTextRun(out []string, text string, limit int) []string {
	if limit <= 0 || len(text) < 4 {
		return out
	}
	if len(text) > 80 {
		text = text[:80]
	}
	return append(out, text)
}

func itoaReplay(value int) string {
	if value == 0 {
		return "0"
	}
	negative := value < 0
	if negative {
		value = -value
	}
	var buf [20]byte
	i := len(buf)
	for value > 0 {
		i--
		buf[i] = byte('0' + value%10)
		value /= 10
	}
	if negative {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
