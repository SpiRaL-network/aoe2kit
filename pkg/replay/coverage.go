package replay

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	"aoe2kit/pkg/triggergraph"
)

type CoverageReport struct {
	Path        string          `json:"path,omitempty"`
	FileBytes   int             `json:"file_bytes"`
	Summary     CoverageSummary `json:"summary"`
	HeaderSpine *HeaderSpine    `json:"header_spine,omitempty"`
	Regions     []ReplayRegion  `json:"regions"`
	BodyOps     map[int]int     `json:"body_operations,omitempty"`
	Samples     []ReplayRegion  `json:"samples,omitempty"`
	Warnings    []string        `json:"warnings,omitempty"`
}

type CoverageSummary struct {
	CompressedHeaderBytes int     `json:"compressed_header_bytes"`
	InflatedHeaderBytes   int     `json:"inflated_header_bytes"`
	BodyBytes             int     `json:"body_bytes"`
	HeaderDecodedBytes    int     `json:"header_decoded_bytes"`
	HeaderOpaqueBytes     int     `json:"header_opaque_bytes"`
	HeaderDecodedPercent  float64 `json:"header_decoded_percent"`
	BodyDecodedBytes      int     `json:"body_decoded_bytes"`
	BodyOpaqueBytes       int     `json:"body_opaque_bytes"`
	BodyDecodedPercent    float64 `json:"body_decoded_percent"`
}

type ReplayRegion struct {
	Space      string         `json:"space"`
	Name       string         `json:"name"`
	Start      int            `json:"start"`
	End        int            `json:"end"`
	Bytes      int            `json:"bytes"`
	Status     string         `json:"status"`
	Confidence string         `json:"confidence"`
	Details    map[string]any `json:"details,omitempty"`
}

type effectiveGameDataTail struct {
	playerIndex int
	region      ReplayRegion
	bytes       []byte
}

type effectiveDataRun struct {
	start int
	end   int
}

func BuildCoverage(path string) (*CoverageReport, error) {
	data, err := ReadRecordBytes(path)
	if err != nil {
		return nil, err
	}
	if len(data) < 12 {
		return nil, fmt.Errorf("record too short")
	}
	rec, err := Parse(data)
	if err != nil {
		return nil, err
	}
	headerLength := rec.HeaderLength
	if headerLength > len(data) {
		return nil, fmt.Errorf("record header length %d exceeds file length %d", headerLength, len(data))
	}
	body := data[headerLength:]
	events, _, _, eventWarnings := ExtractActionStreamEvents(body, EventOptions{IncludeUntypedAction: true})
	refs := collectRefsFromEvents(events)
	report := &CoverageReport{
		Path:      path,
		FileBytes: len(data),
		BodyOps:   map[int]int{},
	}
	for _, warning := range eventWarnings {
		report.Warnings = append(report.Warnings, "body event refs: "+warning)
	}
	report.add("file", "record.header_length", 0, 4, "decoded", "parsed", nil)
	report.add("file", "record.compression_header", 4, 8, "decoded", "parsed", nil)
	report.add("file", "record.compressed_header", 8, headerLength, "decoded", "inflated", nil)
	report.add("file", "record.body", headerLength, len(data), "decoded", "parsed_in_body_space", nil)
	report.coverHeader(rec, refs)
	report.coverHeaderTriggerGraph(rec)
	if rec.TriggerGraph == nil {
		report.coverHeaderDEStrings(rec.HeaderBytes())
		report.coverHeaderScenarioStrings(rec.HeaderBytes())
	}
	report.coverEffectiveGameDataTails(rec.HeaderBytes())
	report.coverHeaderBoundedOpaqueDEStringIslands(rec.HeaderBytes())
	report.coverHeaderBoundedOpaqueScenarioStringIslands(rec.HeaderBytes())
	report.coverHeaderBoundedOpaquePlainLengthStringIslands(rec.HeaderBytes())
	report.coverHeaderBoundedOpaqueAILoadDirectives(rec.HeaderBytes())
	report.coverHeaderBoundedOpaqueXSIncludeDirectives(rec.HeaderBytes())
	report.coverHeaderBoundedOpaquePromiDENameTables(rec.HeaderBytes())
	report.coverHeaderBoundedOpaqueAIScriptBlockSeparators(rec.HeaderBytes())
	report.coverHeaderBoundedOpaqueAIScriptTextFragments(rec.HeaderBytes())
	report.coverHeaderBoundedOpaqueEffectiveTemplateDuplicates(rec.HeaderBytes(), 256)
	report.fillGaps("inflated_header", len(rec.HeaderBytes()))
	report.coverHeaderBoundedOpaqueCaptionRecords(rec.HeaderBytes())
	report.coverHeaderBoundedOpaquePlayerResourceBlocks(rec.HeaderBytes())
	report.coverBody(body)
	report.fillGaps("body", len(body))
	report.Summary = summarizeCoverage(report.Regions, headerLength-8, rec.InflatedBytes, len(body))
	if len(report.BodyOps) == 0 {
		report.BodyOps = nil
	}
	return report, nil
}

type headerStringIsland struct {
	start int
	end   int
	text  string
}

type aiLoadDirectiveIsland struct {
	start      int
	end        int
	text       string
	modulePath string
	moduleName string
}

type xsIncludeDirectiveIsland struct {
	start int
	end   int
	text  string
	path  string
	name  string
}

type promideNameTableIsland struct {
	start int
	end   int
	count int
}

func (r *CoverageReport) coverHeaderBoundedOpaqueDEStringIslands(header []byte) {
	var out []ReplayRegion
	for _, region := range r.Regions {
		if region.Space != "inflated_header" || region.Status != "bounded_opaque" {
			out = append(out, region)
			continue
		}
		islands := deStringIslands(header, region.Start, region.End)
		if len(islands) == 0 {
			out = append(out, region)
			continue
		}
		cursor := region.Start
		for i, island := range islands {
			if cursor < island.start {
				out = append(out, replayRegion("inflated_header", fmt.Sprintf("%s_gap_%04d", region.Name, i), cursor, island.start, region.Status, region.Confidence, map[string]any{
					"source_region": region.Name,
				}))
			}
			text := island.text
			if len(text) > 120 {
				text = text[:120]
			}
			out = append(out, replayRegion("inflated_header", fmt.Sprintf("%s_de_string_%04d", region.Name, i), island.start, island.end, "decoded", "parsed_de_string_record_inside_bounded_opaque_header_region", map[string]any{
				"text":          text,
				"source_region": region.Name,
				"note":          "decoded DE string island carved out of a bounded opaque header region; surrounding structure remains opaque",
			}))
			cursor = island.end
		}
		if cursor < region.End {
			out = append(out, replayRegion("inflated_header", fmt.Sprintf("%s_gap_%04d", region.Name, len(islands)), cursor, region.End, region.Status, region.Confidence, map[string]any{
				"source_region": region.Name,
			}))
		}
	}
	r.Regions = out
}

func (r *CoverageReport) coverHeaderBoundedOpaqueEffectiveTemplateDuplicates(header []byte, minBytes int) {
	if minBytes <= 0 {
		minBytes = 256
	}
	sources := frontierTemplateSources(r.Regions, header)
	if len(sources) == 0 {
		return
	}
	var out []ReplayRegion
	for _, region := range r.Regions {
		if region.Space != "inflated_header" || region.Status != "bounded_opaque" || region.Bytes < minBytes {
			out = append(out, region)
			continue
		}
		if strings.Contains(region.Name, "_effective_gamedata_diff_run_") {
			out = append(out, region)
			continue
		}
		if region.Start < 0 || region.End > len(header) || region.Start >= region.End {
			out = append(out, region)
			continue
		}
		payload := header[region.Start:region.End]
		matched := false
		for _, source := range sources {
			rel := bytes.Index(source.data, payload)
			if rel < 0 {
				continue
			}
			details := map[string]any{
				"source_region":          source.region.Name,
				"source_start":           source.region.Start + rel,
				"source_end":             source.region.Start + rel + len(payload),
				"source_relative_offset": rel,
				"matched_payload_sha256": sha256Hex(payload),
				"note":                   "whole opaque span is byte-identical to a subrange of an already framed effective game-data template/common region; field semantics remain inherited from that template island, not newly decoded here",
			}
			out = append(out, replayRegion(region.Space, region.Name, region.Start, region.End, "decoded", "duplicate_of_effective_gamedata_template_subrange", details))
			matched = true
			break
		}
		if !matched {
			out = append(out, region)
		}
	}
	r.Regions = out
}

func deStringIslands(header []byte, start int, end int) []headerStringIsland {
	if start < 0 {
		start = 0
	}
	if end > len(header) {
		end = len(header)
	}
	var islands []headerStringIsland
	for off := start; off+4 <= end; off++ {
		if header[off] != 0x60 || header[off+1] != 0x0a {
			continue
		}
		length := int(int16(binary.LittleEndian.Uint16(header[off+2:])))
		if length < 4 || off+4+length > end {
			continue
		}
		text := decodeCString(header[off+4 : off+4+length])
		if !plausibleHeaderString(text) {
			continue
		}
		islands = append(islands, headerStringIsland{start: off, end: off + 4 + length, text: text})
		off += 3 + length
	}
	return islands
}

func (r *CoverageReport) coverHeaderBoundedOpaqueScenarioStringIslands(header []byte) {
	var out []ReplayRegion
	for _, region := range r.Regions {
		if region.Space != "inflated_header" || region.Status != "bounded_opaque" {
			out = append(out, region)
			continue
		}
		if region.Start < 0 || region.End > len(header) || region.Start >= region.End {
			out = append(out, region)
			continue
		}
		items := triggergraph.ScenarioStrings(header[region.Start:region.End])
		if len(items) == 0 {
			out = append(out, region)
			continue
		}
		cursor := region.Start
		emitted := 0
		for _, item := range items {
			start := region.Start + item.MarkerOffset
			end := region.Start + item.EndOffset
			if start < cursor || end > region.End || start >= end {
				continue
			}
			if cursor < start {
				out = append(out, replayRegion("inflated_header", fmt.Sprintf("%s_gap_%04d", region.Name, emitted), cursor, start, region.Status, region.Confidence, map[string]any{
					"source_region": region.Name,
				}))
			}
			text := item.Text
			if len(text) > 120 {
				text = text[:120]
			}
			out = append(out, replayRegion("inflated_header", fmt.Sprintf("%s_scenario_string_%04d", region.Name, emitted), start, end, "decoded", "parsed_scenario_string_record_inside_bounded_opaque_header_region", map[string]any{
				"text":          text,
				"source_region": region.Name,
				"note":          "decoded scenario string record carved out of a bounded opaque header region; surrounding structure remains opaque",
			}))
			cursor = end
			emitted++
		}
		if emitted == 0 {
			out = append(out, region)
			continue
		}
		if cursor < region.End {
			out = append(out, replayRegion("inflated_header", fmt.Sprintf("%s_gap_%04d", region.Name, emitted), cursor, region.End, region.Status, region.Confidence, map[string]any{
				"source_region": region.Name,
			}))
		}
	}
	r.Regions = out
}

func (r *CoverageReport) coverHeaderBoundedOpaquePlainLengthStringIslands(header []byte) {
	var out []ReplayRegion
	for _, region := range r.Regions {
		if region.Space != "inflated_header" || region.Status != "bounded_opaque" {
			out = append(out, region)
			continue
		}
		islands := plainLengthStringIslands(header, region.Start, region.End)
		if len(islands) == 0 {
			out = append(out, region)
			continue
		}
		cursor := region.Start
		for i, island := range islands {
			if cursor < island.start {
				out = append(out, replayRegion("inflated_header", fmt.Sprintf("%s_gap_%04d", region.Name, i), cursor, island.start, region.Status, region.Confidence, map[string]any{
					"source_region": region.Name,
				}))
			}
			text := island.text
			if len(text) > 120 {
				text = text[:120]
			}
			out = append(out, replayRegion("inflated_header", fmt.Sprintf("%s_plain_string_%04d", region.Name, i), island.start, island.end, "decoded", "parsed_u16_length_ascii_string_inside_bounded_opaque_header_region", map[string]any{
				"text":          text,
				"source_region": region.Name,
				"note":          "decoded u16-length ASCII string carved out of a bounded opaque header region; surrounding structure remains opaque",
			}))
			cursor = island.end
		}
		if cursor < region.End {
			out = append(out, replayRegion("inflated_header", fmt.Sprintf("%s_gap_%04d", region.Name, len(islands)), cursor, region.End, region.Status, region.Confidence, map[string]any{
				"source_region": region.Name,
			}))
		}
	}
	r.Regions = out
}

func (r *CoverageReport) coverHeaderBoundedOpaqueAILoadDirectives(header []byte) {
	var out []ReplayRegion
	for _, region := range r.Regions {
		if region.Space != "inflated_header" || region.Status != "bounded_opaque" {
			out = append(out, region)
			continue
		}
		islands := aiLoadDirectiveIslands(header, region.Start, region.End)
		if len(islands) == 0 {
			out = append(out, region)
			continue
		}
		cursor := region.Start
		for i, island := range islands {
			if cursor < island.start {
				out = append(out, replayRegion("inflated_header", fmt.Sprintf("%s_gap_%04d", region.Name, i), cursor, island.start, region.Status, region.Confidence, map[string]any{
					"source_region": region.Name,
				}))
			}
			out = append(out, replayRegion("inflated_header", fmt.Sprintf("%s_ai_load_%04d", region.Name, i), island.start, island.end, "decoded", "parsed_ai_load_directive_inside_bounded_opaque_header_region", map[string]any{
				"text":          island.text,
				"module_path":   island.modulePath,
				"module_name":   island.moduleName,
				"script_family": "PromiDE/Promisory",
				"note":          "decoded Promisory load directive inside a bounded opaque AI script/module loadout container; binary wrapper remains opaque",
				"source_region": region.Name,
			}))
			cursor = island.end
		}
		if cursor < region.End {
			out = append(out, replayRegion("inflated_header", fmt.Sprintf("%s_gap_%04d", region.Name, len(islands)), cursor, region.End, region.Status, region.Confidence, map[string]any{
				"source_region": region.Name,
			}))
		}
	}
	r.Regions = out
}

func (r *CoverageReport) coverHeaderBoundedOpaqueAIScriptTextFragments(header []byte) {
	var out []ReplayRegion
	for i, region := range r.Regions {
		if region.Space != "inflated_header" || region.Status != "bounded_opaque" {
			out = append(out, region)
			continue
		}
		if !adjacentToAILoadDirective(r.Regions, i) || !aiScriptTextFragment(header, region.Start, region.End) {
			out = append(out, region)
			continue
		}
		text := string(header[region.Start:region.End])
		if len(text) > 120 {
			text = text[:120]
		}
		out = append(out, replayRegion(region.Space, region.Name, region.Start, region.End, "decoded", "parsed_ai_script_text_fragment_inside_bounded_opaque_header_region", map[string]any{
			"text":          text,
			"source_region": detailString(region.Details, "source_region"),
			"script_family": "PromiDE/Promisory",
			"note":          "decoded ASCII AI-script text adjacent to Promisory load directives; binary wrapper remains opaque",
		}))
	}
	r.Regions = out
}

func adjacentToAILoadDirective(regions []ReplayRegion, i int) bool {
	if i > 0 && regions[i-1].Space == regions[i].Space && aiScriptDirectiveConfidence(regions[i-1].Confidence) {
		return true
	}
	if i+1 < len(regions) && regions[i+1].Space == regions[i].Space && aiScriptDirectiveConfidence(regions[i+1].Confidence) {
		return true
	}
	return false
}

func aiScriptDirectiveConfidence(confidence string) bool {
	return confidence == "parsed_ai_load_directive_inside_bounded_opaque_header_region" ||
		confidence == "parsed_xs_include_directive_inside_bounded_opaque_header_region"
}

func aiScriptTextFragment(header []byte, start int, end int) bool {
	if start < 0 || end > len(header) || start >= end || end-start > 512 {
		return false
	}
	hasLineByte := false
	hasNonWhitespace := false
	for _, b := range header[start:end] {
		switch {
		case b == '\r' || b == '\n':
			hasLineByte = true
		case b == '\t' || b == ' ':
		case b >= 0x21 && b <= 0x7e:
			hasNonWhitespace = true
		default:
			return false
		}
	}
	return hasLineByte || hasNonWhitespace
}

func (r *CoverageReport) coverHeaderBoundedOpaqueXSIncludeDirectives(header []byte) {
	var out []ReplayRegion
	for _, region := range r.Regions {
		if region.Space != "inflated_header" || region.Status != "bounded_opaque" {
			out = append(out, region)
			continue
		}
		islands := xsIncludeDirectiveIslands(header, region.Start, region.End)
		if len(islands) == 0 {
			out = append(out, region)
			continue
		}
		cursor := region.Start
		for i, island := range islands {
			if cursor < island.start {
				out = append(out, replayRegion("inflated_header", fmt.Sprintf("%s_gap_%04d", region.Name, i), cursor, island.start, region.Status, region.Confidence, map[string]any{
					"source_region": region.Name,
				}))
			}
			out = append(out, replayRegion("inflated_header", fmt.Sprintf("%s_xs_include_%04d", region.Name, i), island.start, island.end, "decoded", "parsed_xs_include_directive_inside_bounded_opaque_header_region", map[string]any{
				"text":          island.text,
				"include_path":  island.path,
				"include_name":  island.name,
				"script_family": "XS",
				"note":          "decoded XS include directive inside a bounded opaque script/loadout container; binary wrapper remains opaque",
				"source_region": region.Name,
			}))
			cursor = island.end
		}
		if cursor < region.End {
			out = append(out, replayRegion("inflated_header", fmt.Sprintf("%s_gap_%04d", region.Name, len(islands)), cursor, region.End, region.Status, region.Confidence, map[string]any{
				"source_region": region.Name,
			}))
		}
	}
	r.Regions = out
}

func xsIncludeDirectiveIslands(header []byte, start int, end int) []xsIncludeDirectiveIsland {
	if start < 0 {
		start = 0
	}
	if end > len(header) {
		end = len(header)
	}
	const prefix = `(include "`
	var islands []xsIncludeDirectiveIsland
	for off := start; off+len(prefix) < end; off++ {
		if string(header[off:off+len(prefix)]) != prefix {
			continue
		}
		quoteStart := off + len(prefix)
		quoteEnd := quoteStart
		for quoteEnd < end && quoteEnd-quoteStart <= 160 && header[quoteEnd] != '"' {
			quoteEnd++
		}
		if quoteEnd >= end || quoteEnd-quoteStart > 160 || quoteEnd+2 > end || header[quoteEnd+1] != ')' {
			continue
		}
		includePath := string(header[quoteStart:quoteEnd])
		includeName, ok := xsIncludeName(includePath)
		if !ok {
			continue
		}
		text := string(header[off : quoteEnd+2])
		islands = append(islands, xsIncludeDirectiveIsland{
			start: off,
			end:   quoteEnd + 2,
			text:  text,
			path:  includePath,
			name:  includeName,
		})
		off = quoteEnd + 1
	}
	return islands
}

func xsIncludeName(includePath string) (string, bool) {
	if includePath == "" || !strings.HasSuffix(strings.ToLower(includePath), ".xs") {
		return "", false
	}
	normalized := strings.ReplaceAll(includePath, `\`, "/")
	parts := strings.Split(normalized, "/")
	name := parts[len(parts)-1]
	if name == "" || name == ".xs" {
		return "", false
	}
	for _, r := range normalized {
		if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '-' || r == '.' || r == '/' {
			continue
		}
		return "", false
	}
	return name, true
}

func (r *CoverageReport) coverHeaderBoundedOpaquePromiDENameTables(header []byte) {
	var out []ReplayRegion
	for _, region := range r.Regions {
		if region.Space != "inflated_header" || region.Status != "bounded_opaque" {
			out = append(out, region)
			continue
		}
		islands := promideNameTableIslands(header, region.Start, region.End)
		if len(islands) == 0 {
			out = append(out, region)
			continue
		}
		cursor := region.Start
		for i, island := range islands {
			if cursor < island.start {
				out = append(out, replayRegion("inflated_header", fmt.Sprintf("%s_gap_%04d", region.Name, i), cursor, island.start, region.Status, region.Confidence, map[string]any{
					"source_region": region.Name,
				}))
			}
			out = append(out, replayRegion("inflated_header", fmt.Sprintf("%s_promide_name_table_%04d", region.Name, i), island.start, island.end, "decoded", "parsed_promide_name_table_inside_bounded_opaque_header_region", map[string]any{
				"name":          "PromiDE",
				"count":         island.count,
				"record_shape":  "repeated u16 byte length + ASCII name",
				"script_family": "PromiDE/Promisory",
				"note":          "decoded repeated PromiDE name table inside AI/default-loadout wrapper; surrounding binary wrapper remains opaque",
				"source_region": region.Name,
			}))
			cursor = island.end
		}
		if cursor < region.End {
			out = append(out, replayRegion("inflated_header", fmt.Sprintf("%s_gap_%04d", region.Name, len(islands)), cursor, region.End, region.Status, region.Confidence, map[string]any{
				"source_region": region.Name,
			}))
		}
	}
	r.Regions = out
}

func promideNameTableIslands(header []byte, start int, end int) []promideNameTableIsland {
	if start < 0 {
		start = 0
	}
	if end > len(header) {
		end = len(header)
	}
	needle := []byte("PromiDE")
	var islands []promideNameTableIsland
	for off := start; off+2+len(needle) <= end; off++ {
		if binary.LittleEndian.Uint16(header[off:]) != uint16(len(needle)) || !bytes.Equal(header[off+2:off+2+len(needle)], needle) {
			continue
		}
		cursor := off
		count := 0
		for cursor+2+len(needle) <= end &&
			binary.LittleEndian.Uint16(header[cursor:]) == uint16(len(needle)) &&
			bytes.Equal(header[cursor+2:cursor+2+len(needle)], needle) {
			count++
			cursor += 2 + len(needle)
		}
		if count >= 2 {
			islands = append(islands, promideNameTableIsland{start: off, end: cursor, count: count})
			off = cursor - 1
		}
	}
	return islands
}

func (r *CoverageReport) coverHeaderBoundedOpaqueAIScriptBlockSeparators(header []byte) {
	var out []ReplayRegion
	for _, region := range r.Regions {
		if region.Space != "inflated_header" || region.Status != "bounded_opaque" {
			out = append(out, region)
			continue
		}
		if !aiScriptBlockSeparator(header, region.Start, region.End) {
			out = append(out, region)
			continue
		}
		out = append(out, replayRegion(region.Space, region.Name, region.Start, region.End, "decoded", "parsed_ai_script_block_separator_inside_bounded_opaque_header_region", map[string]any{
			"separator_prefix": "CRLF",
			"zero_bytes":       8,
			"observed_u32":     int(binary.LittleEndian.Uint32(header[region.End-4 : region.End])),
			"script_family":    "PromiDE/Promisory",
			"note":             "decoded repeated AI/XS script block separator; observed numeric field is preserved without assigning gameplay meaning",
			"source_region":    detailString(region.Details, "source_region"),
		}))
	}
	r.Regions = out
}

func aiScriptBlockSeparator(header []byte, start int, end int) bool {
	if start < 0 || end > len(header) || end-start != 14 {
		return false
	}
	if header[start] != '\r' || header[start+1] != '\n' {
		return false
	}
	for _, b := range header[start+2 : start+10] {
		if b != 0 {
			return false
		}
	}
	return binary.LittleEndian.Uint32(header[start+10:end]) != 0
}

func aiLoadDirectiveIslands(header []byte, start int, end int) []aiLoadDirectiveIsland {
	if start < 0 {
		start = 0
	}
	if end > len(header) {
		end = len(header)
	}
	const prefix = `(load "`
	var islands []aiLoadDirectiveIsland
	for off := start; off+len(prefix) < end; off++ {
		if string(header[off:off+len(prefix)]) != prefix {
			continue
		}
		quoteStart := off + len(prefix)
		quoteEnd := quoteStart
		for quoteEnd < end && quoteEnd-quoteStart <= 160 && header[quoteEnd] != '"' {
			quoteEnd++
		}
		if quoteEnd >= end || quoteEnd-quoteStart > 160 || quoteEnd+2 > end || header[quoteEnd+1] != ')' {
			continue
		}
		modulePath := string(header[quoteStart:quoteEnd])
		moduleName, ok := promisoryModuleName(modulePath)
		if !ok {
			continue
		}
		text := string(header[off : quoteEnd+2])
		islands = append(islands, aiLoadDirectiveIsland{
			start:      off,
			end:        quoteEnd + 2,
			text:       text,
			modulePath: modulePath,
			moduleName: moduleName,
		})
		off = quoteEnd + 1
	}
	return islands
}

func promisoryModuleName(modulePath string) (string, bool) {
	const a = "Promisory/"
	const b = `Promisory\`
	name := ""
	switch {
	case strings.HasPrefix(modulePath, a):
		name = modulePath[len(a):]
	case strings.HasPrefix(modulePath, b):
		name = modulePath[len(b):]
	default:
		return "", false
	}
	if name == "" || strings.ContainsAny(name, `/\`) {
		return "", false
	}
	for _, r := range name {
		if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			continue
		}
		return "", false
	}
	return name, true
}

func plainLengthStringIslands(header []byte, start int, end int) []headerStringIsland {
	if start < 0 {
		start = 0
	}
	if end > len(header) {
		end = len(header)
	}
	var islands []headerStringIsland
	for off := start; off+2 <= end; off++ {
		if plainTextByte(header[off]) && plainTextByte(header[off+1]) {
			continue
		}
		length := int(binary.LittleEndian.Uint16(header[off:]))
		if length < 8 || length > 4096 || off+2+length > end {
			continue
		}
		raw := header[off+2 : off+2+length]
		if !plausiblePlainLengthString(raw) {
			continue
		}
		if !plainLengthStringEndBoundary(header, off+2+length, end) {
			continue
		}
		islands = append(islands, headerStringIsland{start: off, end: off + 2 + length, text: string(raw)})
		off += 1 + length
	}
	return islands
}

func plainLengthStringEndBoundary(header []byte, off int, regionEnd int) bool {
	if off >= regionEnd {
		return true
	}
	if off < 0 || off >= len(header) {
		return false
	}
	if off+2 <= regionEnd {
		if plainTextByte(header[off]) && plainTextByte(header[off+1]) {
			return false
		}
		nextLength := int(binary.LittleEndian.Uint16(header[off:]))
		if nextLength >= 8 && nextLength <= 4096 && off+2+nextLength <= regionEnd && plausiblePlainLengthString(header[off+2:off+2+nextLength]) {
			return true
		}
	}
	return !plainTextByte(header[off])
}

func plausiblePlainLengthString(raw []byte) bool {
	if len(raw) < 8 {
		return false
	}
	first := -1
	for i, b := range raw {
		if b == 9 || b == 10 || b == 13 || b == ' ' {
			continue
		}
		first = i
		break
	}
	if first < 0 || raw[first] < 32 || raw[first] > 126 {
		return false
	}
	printable := 0
	letters := 0
	for _, b := range raw {
		if b == 9 || b == 10 || b == 13 || (b >= 32 && b <= 126) {
			printable++
		}
		if (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z') {
			letters++
		}
	}
	return printable*100/len(raw) >= 98 && letters >= 3
}

func plainTextByte(b byte) bool {
	return b == 9 || b == 10 || b == 13 || b == ' ' || (b >= 32 && b <= 126)
}

func (r *CoverageReport) coverEffectiveGameDataTails(header []byte) {
	var tails []effectiveGameDataTail
	var kept []ReplayRegion
	for _, region := range r.Regions {
		playerIndex, ok := effectiveGameDataTailRegion(region)
		if !ok || playerIndex == 0 || region.Start < 0 || region.End > len(header) || region.Start >= region.End {
			kept = append(kept, region)
			continue
		}
		tails = append(tails, effectiveGameDataTail{
			playerIndex: playerIndex,
			region:      region,
			bytes:       header[region.Start:region.End],
		})
	}
	if len(tails) < 2 {
		r.Regions = kept
		return
	}
	sort.Slice(tails, func(i, j int) bool { return tails[i].region.Start < tails[j].region.Start })
	ref := tails[0]
	kept = append(kept, replayRegion("inflated_header", fmt.Sprintf("initial_player_%d_effective_gamedata_template_reference", ref.playerIndex), ref.region.Start, ref.region.End, "decoded", "effective_gamedata_shared_template_reference_measured", map[string]any{
		"source_region": ref.region.Name,
		"tail_bytes":    ref.region.Bytes,
		"label":         regionDetail(ref.region, "label"),
		"note":          "complete effective game-data table for the reference player; semantic field layout not fully decoded",
	}))
	for _, other := range tails[1:] {
		kept = append(kept, partitionEffectiveGameDataTail(ref, other)...)
	}
	r.Regions = kept
}

func effectiveGameDataTailRegion(region ReplayRegion) (int, bool) {
	if region.Space != "inflated_header" || region.Status == "decoded" {
		return 0, false
	}
	const prefix = "initial_player_"
	const suffix = "_object_tail_before_candidate_prefixes"
	if !strings.HasPrefix(region.Name, prefix) || !strings.HasSuffix(region.Name, suffix) {
		const unsegmentedSuffix = "_object_tail_unsegmented"
		if !strings.HasSuffix(region.Name, unsegmentedSuffix) {
			return 0, false
		}
		mid := strings.TrimSuffix(strings.TrimPrefix(region.Name, prefix), unsegmentedSuffix)
		playerIndex, err := strconv.Atoi(mid)
		if err != nil {
			return 0, false
		}
		return playerIndex, true
	}
	mid := strings.TrimSuffix(strings.TrimPrefix(region.Name, prefix), suffix)
	playerIndex, err := strconv.Atoi(mid)
	if err != nil {
		return 0, false
	}
	return playerIndex, true
}

func partitionEffectiveGameDataTail(ref, other effectiveGameDataTail) []ReplayRegion {
	compared := len(ref.bytes)
	if len(other.bytes) < compared {
		compared = len(other.bytes)
	}
	identical := 0
	for i := 0; i < compared; i++ {
		if ref.bytes[i] == other.bytes[i] {
			identical++
		}
	}
	ratio := 0.0
	if compared > 0 {
		ratio = float64(identical) / float64(compared)
	}
	diffRuns := effectiveGameDataDiffRuns(ref.bytes, other.bytes)
	boundaries := []int{0, len(other.bytes)}
	for _, run := range diffRuns {
		boundaries = append(boundaries, clampInt(run.start, 0, len(other.bytes)), clampInt(run.end, 0, len(other.bytes)))
	}
	availabilityStart := DefaultEffectiveAvailabilityOffset
	availabilityEnd := DefaultEffectiveAvailabilityOffset + DefaultEffectiveAvailabilityCount*2
	if availabilityStart < len(other.bytes) && availabilityEnd > 0 {
		boundaries = append(boundaries, clampInt(availabilityStart, 0, len(other.bytes)), clampInt(availabilityEnd, 0, len(other.bytes)))
	}
	sort.Ints(boundaries)
	boundaries = uniqueInts(boundaries)
	var regions []ReplayRegion
	commonIndex := 0
	diffIndex := 0
	availabilityIndex := 0
	for i := 0; i+1 < len(boundaries); i++ {
		start := boundaries[i]
		end := boundaries[i+1]
		if start >= end {
			continue
		}
		if start >= availabilityStart && end <= availabilityEnd {
			regions = append(regions, effectiveGameDataAvailabilityRegion(ref, other, availabilityIndex, start, end, compared, ratio))
			availabilityIndex++
			continue
		}
		if effectiveGameDataRangeEqual(ref.bytes, other.bytes, start, end) {
			regions = append(regions, effectiveGameDataCommonRegion(ref, other, commonIndex, start, end, compared, ratio))
			commonIndex++
			continue
		}
		regions = append(regions, effectiveGameDataDiffRegion(ref, other, diffIndex, effectiveDataRun{start: start, end: end}, compared, ratio))
		diffIndex++
	}
	return regions
}

func effectiveGameDataDiffRuns(ref []byte, other []byte) []effectiveDataRun {
	compared := len(ref)
	if len(other) < compared {
		compared = len(other)
	}
	var runs []effectiveDataRun
	inRun := false
	start := 0
	for i := 0; i < compared; i++ {
		if ref[i] == other[i] {
			if inRun {
				runs = append(runs, effectiveDataRun{start: start, end: i})
				inRun = false
			}
			continue
		}
		if !inRun {
			start = i
			inRun = true
		}
	}
	if inRun {
		runs = append(runs, effectiveDataRun{start: start, end: compared})
	}
	if len(other) > compared {
		runs = append(runs, effectiveDataRun{start: compared, end: len(other)})
	}
	for i := range runs {
		runs[i].start = alignDown2(runs[i].start)
		runs[i].end = alignUp2(runs[i].end, len(other))
	}
	return mergeCloseRuns(runs, 2)
}

func effectiveGameDataCommonRegion(ref, other effectiveGameDataTail, index int, start int, end int, compared int, ratio float64) ReplayRegion {
	return replayRegion("inflated_header", fmt.Sprintf("initial_player_%d_effective_gamedata_template_common_%04d", other.playerIndex, index), other.region.Start+start, other.region.Start+end, "decoded", "effective_gamedata_template_common_vs_reference_measured", map[string]any{
		"source_region":     other.region.Name,
		"reference_region":  ref.region.Name,
		"reference_player":  ref.playerIndex,
		"tail_offset_start": start,
		"tail_offset_end":   end,
		"tail_bytes":        end - start,
		"compared_bytes":    compared,
		"identical_ratio":   ratio,
		"label":             regionDetail(other.region, "label"),
		"note":              "byte-identical to the reference player's effective game-data table at the same aligned tail offsets",
	})
}

func effectiveGameDataDiffRegion(ref, other effectiveGameDataTail, index int, run effectiveDataRun, compared int, ratio float64) ReplayRegion {
	refSample, otherSample := effectiveGameDataU16Samples(ref.bytes, other.bytes, run.start, run.end, 8)
	return replayRegion("inflated_header", fmt.Sprintf("initial_player_%d_effective_gamedata_diff_run_%04d", other.playerIndex, index), other.region.Start+run.start, other.region.Start+run.end, "decoded", "effective_gamedata_u16_diff_run_measured_vs_reference_semantics_partial", map[string]any{
		"source_region":        other.region.Name,
		"reference_region":     ref.region.Name,
		"reference_player":     ref.playerIndex,
		"tail_offset_start":    run.start,
		"tail_offset_end":      run.end,
		"tail_bytes":           run.end - run.start,
		"u16_aligned":          run.start%2 == 0 && run.end%2 == 0,
		"compared_bytes":       compared,
		"identical_ratio":      ratio,
		"reference_u16_sample": refSample,
		"player_u16_sample":    otherSample,
		"label":                regionDetail(other.region, "label"),
		"note":                 "player-specific effective game-data bytes parsed as aligned u16 diff runs against the reference effective table; field/index semantics remain partial and are not fully mapped to DAT ids",
	})
}

func effectiveGameDataAvailabilityRegion(ref, other effectiveGameDataTail, index int, start int, end int, compared int, ratio float64) ReplayRegion {
	refSample, otherSample := effectiveGameDataU16Samples(ref.bytes, other.bytes, start, end, 8)
	return replayRegion("inflated_header", fmt.Sprintf("initial_player_%d_effective_gamedata_unit_availability_array_%04d", other.playerIndex, index), other.region.Start+start, other.region.Start+end, "decoded", "effective_gamedata_unit_availability_array_mapped_to_dat_unit_slots", map[string]any{
		"source_region":        other.region.Name,
		"reference_region":     ref.region.Name,
		"reference_player":     ref.playerIndex,
		"tail_offset_start":    start,
		"tail_offset_end":      end,
		"tail_bytes":           end - start,
		"array_index_start":    (start - DefaultEffectiveAvailabilityOffset) / 2,
		"array_index_end":      (end - DefaultEffectiveAvailabilityOffset) / 2,
		"array_entries":        (end - start) / 2,
		"u16_aligned":          start%2 == 0 && end%2 == 0,
		"compared_bytes":       compared,
		"identical_ratio":      ratio,
		"reference_u16_sample": refSample,
		"player_u16_sample":    otherSample,
		"label":                regionDetail(other.region, "label"),
		"note":                 "known effective unit-slot availability array; indices map to DAT unit slots when paired with kit replay effective-data --dat",
	})
}

func effectiveGameDataU16Samples(ref []byte, other []byte, start int, end int, limit int) ([]uint16, []uint16) {
	if start < 0 {
		start = 0
	}
	if end > len(other) {
		end = len(other)
	}
	var refOut []uint16
	var otherOut []uint16
	for off := start; off+2 <= end && len(otherOut) < limit; off += 2 {
		if off+2 <= len(ref) {
			refOut = append(refOut, binary.LittleEndian.Uint16(ref[off:]))
		}
		otherOut = append(otherOut, binary.LittleEndian.Uint16(other[off:]))
	}
	return refOut, otherOut
}

func effectiveGameDataRangeEqual(ref []byte, other []byte, start int, end int) bool {
	if start < 0 || end < start || end > len(other) || end > len(ref) {
		return false
	}
	return bytes.Equal(ref[start:end], other[start:end])
}

func clampInt(value int, min int, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func uniqueInts(values []int) []int {
	if len(values) == 0 {
		return nil
	}
	out := values[:1]
	for _, value := range values[1:] {
		if value != out[len(out)-1] {
			out = append(out, value)
		}
	}
	return out
}

func alignDown2(value int) int {
	return value - value%2
}

func alignUp2(value int, limit int) int {
	if value%2 != 0 {
		value++
	}
	if value > limit {
		return limit
	}
	return value
}

func mergeCloseRuns(runs []effectiveDataRun, maxGap int) []effectiveDataRun {
	if len(runs) == 0 {
		return nil
	}
	sort.Slice(runs, func(i, j int) bool { return runs[i].start < runs[j].start })
	out := []effectiveDataRun{runs[0]}
	for _, run := range runs[1:] {
		last := &out[len(out)-1]
		if run.start <= last.end+maxGap {
			if run.end > last.end {
				last.end = run.end
			}
			continue
		}
		out = append(out, run)
	}
	return out
}

func regionDetail(region ReplayRegion, key string) any {
	if region.Details == nil {
		return nil
	}
	return region.Details[key]
}

func (r *CoverageReport) add(space, name string, start, end int, status, confidence string, details map[string]any) {
	if end < start {
		r.Warnings = append(r.Warnings, fmt.Sprintf("%s %s has negative span %d..%d", space, name, start, end))
		return
	}
	r.Regions = append(r.Regions, ReplayRegion{
		Space:      space,
		Name:       name,
		Start:      start,
		End:        end,
		Bytes:      end - start,
		Status:     status,
		Confidence: confidence,
		Details:    sanitizeRegionDetails(details),
	})
}

func (r *CoverageReport) coverHeaderTriggerGraph(rec *File) {
	if rec.TriggerGraph == nil {
		return
	}
	start := rec.TriggerGraph.Start
	end := rec.TriggerGraph.End
	if rec.TriggerRegion != nil {
		start = rec.TriggerRegion.Start
		end = rec.TriggerRegion.End
	}
	if start < 0 || end <= start || end > len(rec.HeaderBytes()) {
		return
	}
	if headerSpanDecoded(r.Regions, start, end) {
		return
	}
	r.add("inflated_header", "trigger_graph", start, end, "decoded", "parsed", map[string]any{
		"sha256":     rec.TriggerGraph.SHA256,
		"triggers":   rec.TriggerGraph.TriggerCount,
		"effects":    rec.TriggerGraph.EffectCount,
		"conditions": rec.TriggerGraph.ConditionCount,
		"messages":   rec.TriggerGraph.MessageCount,
	})
}

func (r *CoverageReport) coverHeaderDEStrings(header []byte) {
	seen := map[string]bool{}
	for off := 0; off+4 <= len(header); off++ {
		if header[off] != 0x60 || header[off+1] != 0x0a {
			continue
		}
		length := int(int16(binary.LittleEndian.Uint16(header[off+2:])))
		if length < 4 || off+4+length > len(header) {
			continue
		}
		raw := header[off+4 : off+4+length]
		text := decodeCString(raw)
		if !plausibleHeaderString(text) {
			continue
		}
		if headerSpanDecoded(r.Regions, off, off+4+length) {
			continue
		}
		key := fmt.Sprintf("%d:%d", off, off+4+length)
		if seen[key] {
			continue
		}
		seen[key] = true
		if len(text) > 120 {
			text = text[:120]
		}
		r.add("inflated_header", fmt.Sprintf("de_string_%06d", off), off, off+4+length, "decoded", "parsed_de_string_record", map[string]any{
			"text": text,
		})
		off += 3 + length
	}
}

func (r *CoverageReport) coverHeaderScenarioStrings(header []byte) {
	strings := triggergraph.ScenarioStrings(header)
	for i, item := range strings {
		if headerSpanDecoded(r.Regions, item.MarkerOffset, item.EndOffset) {
			continue
		}
		text := item.Text
		if len(text) > 120 {
			text = text[:120]
		}
		r.add("inflated_header", fmt.Sprintf("scenario_string_%03d", i), item.MarkerOffset, item.EndOffset, "decoded", "parsed_string_record", map[string]any{
			"text": text,
		})
	}
}

func headerSpanDecoded(regions []ReplayRegion, start int, end int) bool {
	for _, region := range regions {
		if region.Space != "inflated_header" || region.Status != "decoded" {
			continue
		}
		if start >= region.Start && end <= region.End {
			return true
		}
	}
	return false
}

func plausibleHeaderString(text string) bool {
	if len(text) < 4 {
		return false
	}
	printable := 0
	for _, r := range text {
		if r == 9 || r == 10 || r == 13 || (r >= 32 && r <= 126) {
			printable++
		}
	}
	return printable*100/len(text) >= 90
}

func (r *CoverageReport) coverHeader(rec *File, refs commandObjectRefs) {
	header := rec.HeaderBytes()
	c, err := newDECursor(header, rec.SaveVersion, "coverage parser")
	if err != nil {
		r.Warnings = append(r.Warnings, "header coverage: "+err.Error())
		if rec.TriggerRegion != nil {
			r.add("inflated_header", "trigger_graph", rec.TriggerRegion.Start, rec.TriggerRegion.End, "decoded", "parsed", nil)
		}
		return
	}
	r.add("inflated_header", "version", 0, c.off, "decoded", "parsed", map[string]any{
		"game_version": rec.GameVersion,
		"save_version": rec.SaveVersion,
	})
	deStart := c.off
	if _, err := readDEPlayers(c, rec.SaveVersion); err != nil {
		r.Warnings = append(r.Warnings, "player coverage: "+err.Error())
		return
	}
	playerEnd := c.off
	r.add("inflated_header", "de_settings_and_players", deStart, playerEnd, "decoded", "parsed", nil)
	if _, err := readDEAIFiles(c, rec.SaveVersion); err != nil {
		r.Warnings = append(r.Warnings, "ai coverage: "+err.Error())
		return
	}
	aiEnd := c.off
	r.add("inflated_header", "de_ai_files", playerEnd, aiEnd, "decoded", "parsed", nil)
	if err := skipDETailAfterAI(c); err != nil {
		r.Warnings = append(r.Warnings, "de tail coverage: "+err.Error())
		return
	}
	deEnd := c.off
	r.add("inflated_header", "de_lobby_tail", aiEnd, deEnd, "decoded", "skipped_by_struct", nil)
	metaStart := c.off
	numPlayers, err := skipMetadata(c, rec.SaveVersion)
	if err != nil {
		r.Warnings = append(r.Warnings, "metadata coverage: "+err.Error())
		return
	}
	metaEnd := c.off
	r.add("inflated_header", "replay_metadata", metaStart, metaEnd, "decoded", "skipped_by_struct", nil)
	mapStart := c.off
	mapInfo, err := readMapInfo(c, rec.SaveVersion)
	if err != nil {
		r.Warnings = append(r.Warnings, "map coverage: "+err.Error())
		return
	}
	mapEnd := c.off
	r.add("inflated_header", "map_info", mapStart, mapEnd, "decoded", "parsed", map[string]any{
		"width":          mapInfo.Width,
		"height":         mapInfo.Height,
		"terrain_sha256": mapInfo.TerrainSHA256,
	})
	knownEnd := mapEnd
	if rec.TriggerRegion != nil {
		spine, spineRegions, spineWarnings := parseHeaderSpine(header, mapEnd, numPlayers, rec.TriggerRegion.Start, rec.SaveVersion, mapInfo.Width, mapInfo.Height, refs)
		if spine != nil {
			r.HeaderSpine = spine
			for _, region := range spineRegions {
				r.Regions = append(r.Regions, region)
				if region.End > knownEnd {
					knownEnd = region.End
				}
			}
		}
		for _, warning := range spineWarnings {
			r.Warnings = append(r.Warnings, "header spine: "+warning)
		}
		if rec.TriggerRegion.Start > knownEnd {
			r.add("inflated_header", "v68_tail_before_trigger_graph", knownEnd, rec.TriggerRegion.Start, "bounded_opaque", "sequential_spine_stops_before_trigger_graph", map[string]any{
				"trigger_boundary_source": "triggergraph_region_locator",
			})
		}
		r.add("inflated_header", "trigger_graph", rec.TriggerRegion.Start, rec.TriggerRegion.End, "decoded", "parsed", map[string]any{
			"sha256":     rec.TriggerGraph.SHA256,
			"triggers":   rec.TriggerGraph.TriggerCount,
			"effects":    rec.TriggerGraph.EffectCount,
			"conditions": rec.TriggerGraph.ConditionCount,
			"messages":   rec.TriggerGraph.MessageCount,
		})
		knownEnd = rec.TriggerRegion.End
	}
	if knownEnd < len(header) {
		r.add("inflated_header", "header_tail_unparsed", knownEnd, len(header), "bounded_opaque", "not_yet_mapped", nil)
	}
}

func (r *CoverageReport) coverBody(body []byte) {
	reader := bytes.NewReader(body)
	metaStart := 0
	if err := readReplayMeta(reader); err != nil {
		r.Warnings = append(r.Warnings, "body meta coverage: "+err.Error())
		r.add("body", "body_unparsed", 0, len(body), "bounded_opaque", "parse_failed", nil)
		return
	}
	r.add("body", "body_replay_meta", metaStart, len(body)-reader.Len(), "decoded", "parsed", nil)
	for {
		opStart := len(body) - reader.Len()
		op, err := readU32(reader)
		if err != nil {
			if err != io.EOF {
				r.Warnings = append(r.Warnings, "body coverage stopped: "+err.Error())
				r.add("body", "body_tail_unparsed", opStart, len(body), "bounded_opaque", "parse_failed", nil)
			}
			return
		}
		r.BodyOps[int(op)]++
		switch op {
		case 1:
			actionID, payload, sequence, err := readReplayAction(reader)
			if err != nil {
				r.Warnings = append(r.Warnings, "body action coverage stopped: "+err.Error())
				r.add("body", "action_unparsed", opStart, len(body), "bounded_opaque", "parse_failed", nil)
				return
			}
			r.sample("body", "op_action", opStart, len(body)-reader.Len(), "decoded", "framed", map[string]any{"action_id": actionID, "payload_bytes": len(payload), "sequence": sequence})
		case 2:
			if _, err := readReplaySync(reader); err != nil {
				r.Warnings = append(r.Warnings, "body sync coverage stopped: "+err.Error())
				r.add("body", "sync_unparsed", opStart, len(body), "bounded_opaque", "parse_failed", nil)
				return
			}
			r.sample("body", "op_sync", opStart, len(body)-reader.Len(), "decoded", "parsed", nil)
		case 3:
			if err := skipN(reader, 12); err != nil {
				r.Warnings = append(r.Warnings, "body viewlock coverage stopped: "+err.Error())
				r.add("body", "viewlock_unparsed", opStart, len(body), "bounded_opaque", "parse_failed", nil)
				return
			}
			r.sample("body", "op_viewlock", opStart, len(body)-reader.Len(), "decoded", "camera_observed_no_player", nil)
		case 4:
			if err := skipN(reader, 4); err != nil {
				r.Warnings = append(r.Warnings, "body chat prefix coverage stopped: "+err.Error())
				r.add("body", "chat_unparsed", opStart, len(body), "bounded_opaque", "parse_failed", nil)
				return
			}
			length, err := readU32(reader)
			if err != nil || length > uint32(reader.Len()) {
				r.Warnings = append(r.Warnings, fmt.Sprintf("body chat coverage stopped: length=%d err=%v", length, err))
				r.add("body", "chat_unparsed", opStart, len(body), "bounded_opaque", "parse_failed", nil)
				return
			}
			if err := skipN(reader, int(length)); err != nil {
				r.Warnings = append(r.Warnings, "body chat payload coverage stopped: "+err.Error())
				r.add("body", "chat_unparsed", opStart, len(body), "bounded_opaque", "parse_failed", nil)
				return
			}
			r.sample("body", "op_chat", opStart, len(body)-reader.Len(), "decoded", "framed_json_payload", map[string]any{"payload_bytes": int(length)})
		case 5:
			result, err := skipStart(reader)
			if err != nil {
				r.Warnings = append(r.Warnings, "body start coverage stopped: "+err.Error())
				r.add("body", "start_unparsed", opStart, len(body), "bounded_opaque", "parse_failed", nil)
				return
			}
			if result.Terminal {
				r.add("body", "embedded_tail", opStart, len(body), "bounded_opaque", "raw_preserved", map[string]any{"payload_bytes": result.SkippedBytes})
				return
			}
			r.sample("body", "op_start", opStart, len(body)-reader.Len(), "decoded", "aligned", map[string]any{"payload_bytes": result.SkippedBytes})
		case 6:
			r.sample("body", "op_postgame_marker", opStart, len(body)-reader.Len(), "decoded", "marker_only", nil)
			if reader.Len() > 0 {
				tailStart := len(body) - reader.Len()
				tail := body[tailStart:]
				postgame := parseDEPostgameTail(opStart, 0, tail)
				status := "bounded_opaque"
				confidence := "not_yet_mapped"
				details := map[string]any{"tail_bytes": len(tail)}
				if postgame != nil && len(postgame.Blocks) > 0 && postgame.Confidence != "raw_tail_preserved" {
					status = "decoded"
					confidence = postgame.Confidence
					details["block_count"] = len(postgame.Blocks)
					details["version"] = postgame.Version
					details["has_world_time"] = postgame.WorldTimeMS > 0
					details["leaderboards"] = len(postgame.Leaderboards)
				}
				r.add("body", "op_postgame_tail", tailStart, len(body), status, confidence, details)
			}
			return
		default:
			if err := skipReplaySaveChapter(reader, len(body)); err != nil {
				r.Warnings = append(r.Warnings, fmt.Sprintf("unknown operation id %d at body offset %d", op, opStart))
				r.Warnings = append(r.Warnings, "save chapter coverage failed: "+err.Error())
				r.add("body", "unknown_operation_tail", opStart, len(body), "bounded_opaque", "raw_preserved", map[string]any{"operation_id": int(op)})
				return
			}
			r.sample("body", "op_save_chapter", opStart, len(body)-reader.Len(), "bounded_opaque", "skipped", map[string]any{"operation_id": int(op)})
		}
	}
}

func (r *CoverageReport) sample(space, name string, start, end int, status, confidence string, details map[string]any) {
	region := ReplayRegion{Space: space, Name: name, Start: start, End: end, Bytes: end - start, Status: status, Confidence: confidence, Details: sanitizeRegionDetails(details)}
	if len(r.Samples) < 64 {
		r.Samples = append(r.Samples, region)
	}
	r.Regions = append(r.Regions, region)
}

func (r *CoverageReport) fillGaps(space string, total int) {
	if total <= 0 {
		return
	}
	spans := compactRegions(r.Regions, space)
	end := 0
	for _, span := range spans {
		if span.Start > end {
			r.add(space, space+"_gap_unaccounted", end, span.Start, "bounded_opaque", "gap_filled_by_coverage", nil)
		}
		if span.End > end {
			end = span.End
		}
	}
	if end < total {
		r.add(space, space+"_gap_unaccounted", end, total, "bounded_opaque", "gap_filled_by_coverage", nil)
	}
}

func summarizeCoverage(regions []ReplayRegion, compressedHeaderBytes int, inflatedHeaderBytes int, bodyBytes int) CoverageSummary {
	headerDecoded, headerOpaque := coveredBytes(regions, "inflated_header")
	bodyDecoded, bodyOpaque := coveredBytes(regions, "body")
	return CoverageSummary{
		CompressedHeaderBytes: compressedHeaderBytes,
		InflatedHeaderBytes:   inflatedHeaderBytes,
		BodyBytes:             bodyBytes,
		HeaderDecodedBytes:    headerDecoded,
		HeaderOpaqueBytes:     headerOpaque,
		HeaderDecodedPercent:  percent(headerDecoded, inflatedHeaderBytes),
		BodyDecodedBytes:      bodyDecoded,
		BodyOpaqueBytes:       bodyOpaque,
		BodyDecodedPercent:    percent(bodyDecoded, bodyBytes),
	}
}

func coveredBytes(regions []ReplayRegion, space string) (decoded int, opaque int) {
	for _, span := range compactRegions(regions, space) {
		switch span.Status {
		case "decoded":
			decoded += span.Bytes
		default:
			opaque += span.Bytes
		}
	}
	return decoded, opaque
}

func compactRegions(regions []ReplayRegion, space string) []ReplayRegion {
	var spans []ReplayRegion
	for _, region := range regions {
		if region.Space == space {
			spans = append(spans, region)
		}
	}
	sort.Slice(spans, func(i, j int) bool {
		if spans[i].Start == spans[j].Start {
			return spans[i].End < spans[j].End
		}
		return spans[i].Start < spans[j].Start
	})
	return spans
}

func percent(part int, total int) float64 {
	if total <= 0 {
		return 0
	}
	return float64(int((float64(part)/float64(total))*10000+0.5)) / 100
}

func HexSample(data []byte, limit int) string {
	if len(data) > limit {
		data = data[:limit]
	}
	return hex.EncodeToString(data)
}
