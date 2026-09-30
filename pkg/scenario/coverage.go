package scenario

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
)

// CoverageReport accounts for parsed scenario bytes without treating
// successful structural parsing as semantic understanding.
type CoverageReport struct {
	Path              string         `json:"path,omitempty"`
	Version           string         `json:"version"`
	Verification      string         `json:"verification"`
	FileBytes         int            `json:"file_bytes"`
	HeaderBytes       int            `json:"header_bytes"`
	InflatedBodyBytes int            `json:"inflated_body_bytes"`
	TriggerCount      int            `json:"trigger_count"`
	ConditionCount    int            `json:"condition_count"`
	AccountedBytes    int            `json:"accounted_bytes"`
	NamedBytes        int            `json:"named_bytes"`
	DarkBytes         int            `json:"dark_bytes"`
	GapBytes          int            `json:"gap_bytes"`
	Sections          []ByteCoverage `json:"sections"`
	Notes             []string       `json:"notes,omitempty"`
}

type ByteCoverage struct {
	Space     string     `json:"space"`
	Name      string     `json:"name"`
	Start     int        `json:"start"`
	End       int        `json:"end"`
	Bytes     int        `json:"bytes"`
	Accounted int        `json:"accounted_bytes"`
	Named     int        `json:"named_bytes"`
	Dark      int        `json:"dark_bytes"`
	Gaps      int        `json:"gap_bytes"`
	Status    string     `json:"status"`
	DarkSpans []DarkSpan `json:"dark_spans,omitempty"`
}

// DarkSpan identifies the parser field responsible for a conservative dark
// byte range. Repeated structures may produce the same field path at different
// offsets; offsets are the identity.
type DarkSpan struct {
	Path         string `json:"path"`
	Start        int    `json:"start"`
	End          int    `json:"end"`
	Bytes        int    `json:"bytes"`
	SHA256       string `json:"sha256,omitempty"`
	NonZeroBytes int    `json:"nonzero_bytes,omitempty"`
	HexPrefix    string `json:"hex_prefix,omitempty"`
}

type coverageSpan struct {
	start int
	end   int
}

// CoverageFile reports structural accounting plus anonymous/unknown spans.
// A complete parse can have zero gaps while retaining semantic dark bytes.
func CoverageFile(path string) (CoverageReport, error) {
	f, err := Open(path)
	if err != nil {
		return CoverageReport{}, err
	}
	report := CoverageReport{
		Path:              path,
		Version:           f.Version,
		Verification:      "structure_verified_not_semantics_verified",
		FileBytes:         len(f.original),
		HeaderBytes:       f.HeaderBytes,
		InflatedBodyBytes: f.InflatedBytes,
		TriggerCount:      triggerCount(f.Triggers),
		ConditionCount:    conditionCount(f.Triggers),
		Notes: []string{
			"accounted bytes are structurally consumed by the parser; dark bytes are anonymous/unknown field spans",
			"compressed-body bytes are not counted as semantic scenario bytes; the inflated body is the decoded format space",
		},
	}

	header := coverageForNode("file", "FileHeader", f.headerRoot, f.HeaderBytes)
	report.Sections = append(report.Sections, header)
	for _, section := range f.root.Sections {
		report.Sections = append(report.Sections, coverageForNode("inflated_body", section.Name, section, section.End-section.Start))
	}
	report.AccountedBytes = header.Accounted
	report.NamedBytes = header.Named
	report.DarkBytes = header.Dark
	for _, section := range report.Sections[1:] {
		report.AccountedBytes += section.Accounted
		report.NamedBytes += section.Named
		report.DarkBytes += section.Dark
	}
	annotateDarkSpans(report.Sections, f.header, f.body)
	report.GapBytes = bodyGapBytes(f.Sections, f.InflatedBytes)
	return report, nil
}

func triggerCount(info *TriggerInfo) int {
	if info == nil {
		return 0
	}
	return info.Count
}

func conditionCount(info *TriggerInfo) int {
	if info == nil {
		return 0
	}
	count := 0
	for _, trigger := range info.Triggers {
		count += trigger.Conditions
	}
	return count
}

func annotateDarkSpans(sections []ByteCoverage, header, body []byte) {
	for i := range sections {
		for j := range sections[i].DarkSpans {
			span := &sections[i].DarkSpans[j]
			data := body
			if sections[i].Space == "file" {
				data = header
			}
			if span.Start < 0 || span.End < span.Start || span.End > len(data) {
				continue
			}
			chunk := data[span.Start:span.End]
			digest := sha256.Sum256(chunk)
			span.SHA256 = hex.EncodeToString(digest[:])
			span.HexPrefix = hex.EncodeToString(chunk[:minInt(len(chunk), 16)])
			for _, value := range chunk {
				if value != 0 {
					span.NonZeroBytes++
				}
			}
		}
	}
}

func coverageForNode(space, name string, node *parsedNode, bytes int) ByteCoverage {
	report := ByteCoverage{Space: space, Name: name, Bytes: bytes}
	if node == nil {
		report.Status = "unparsed"
		return report
	}
	report.Start = node.Start
	report.End = node.End
	report.Accounted = node.End - node.Start
	report.DarkSpans = darkFieldSpans(node, name)
	for _, span := range mergeCoverageSpans(darkSpans(node)) {
		report.Dark += span.end - span.start
	}
	report.Named = report.Accounted - report.Dark
	if report.Bytes > report.Accounted {
		report.Gaps = report.Bytes - report.Accounted
	}
	switch {
	case report.Gaps > 0:
		report.Status = "parsed_with_gaps"
	case report.Dark > 0:
		report.Status = "parsed_with_dark_bytes"
	default:
		report.Status = "parsed_named"
	}
	return report
}

func darkFieldSpans(node *parsedNode, path string) []DarkSpan {
	if node == nil {
		return nil
	}
	if isDarkFieldName(node.Name) {
		return []DarkSpan{{Path: path, Start: node.Start, End: node.End, Bytes: node.End - node.Start}}
	}
	var spans []DarkSpan
	for _, child := range node.Fields {
		spans = append(spans, darkFieldSpans(child, path+"."+child.Name)...)
	}
	for index, child := range node.Elements {
		spans = append(spans, darkFieldSpans(child, fmt.Sprintf("%s[%d]", path, index))...)
	}
	return spans
}

func darkSpans(node *parsedNode) []coverageSpan {
	if node == nil {
		return nil
	}
	if isDarkFieldName(node.Name) {
		return []coverageSpan{{start: node.Start, end: node.End}}
	}
	var spans []coverageSpan
	for _, child := range node.Fields {
		spans = append(spans, darkSpans(child)...)
	}
	for _, child := range node.Elements {
		spans = append(spans, darkSpans(child)...)
	}
	return spans
}

func isDarkFieldName(name string) bool {
	lower := strings.ToLower(name)
	return lower == "unknown" || strings.HasPrefix(lower, "unknown_") || strings.HasPrefix(lower, "opaque_")
}

func mergeCoverageSpans(spans []coverageSpan) []coverageSpan {
	if len(spans) < 2 {
		return spans
	}
	sort.Slice(spans, func(i, j int) bool {
		if spans[i].start == spans[j].start {
			return spans[i].end < spans[j].end
		}
		return spans[i].start < spans[j].start
	})
	out := []coverageSpan{spans[0]}
	for _, span := range spans[1:] {
		last := &out[len(out)-1]
		if span.start <= last.end {
			if span.end > last.end {
				last.end = span.end
			}
			continue
		}
		out = append(out, span)
	}
	return out
}

func bodyGapBytes(sections []SectionInfo, bodyBytes int) int {
	if len(sections) == 0 {
		return bodyBytes
	}
	ordered := append([]SectionInfo(nil), sections...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].Start < ordered[j].Start })
	gaps, cursor := 0, 0
	for _, section := range ordered {
		if section.Start > cursor {
			gaps += section.Start - cursor
		}
		if section.End > cursor {
			cursor = section.End
		}
	}
	if cursor < bodyBytes {
		gaps += bodyBytes - cursor
	}
	return gaps
}
