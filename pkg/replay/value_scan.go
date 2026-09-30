package replay

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"aoe2kit/pkg/aoe2"
)

type ValueScanOptions struct {
	Values        []string
	Hex           []string
	Text          []string
	NumericWidths []int
	Space         string
	Limit         int
}

type ValueScanReport struct {
	Path         string                 `json:"path,omitempty"`
	Verification aoe2.VerificationClaim `json:"verification"`
	Summary      ValueScanSummary       `json:"summary"`
	Queries      []ValueQuery           `json:"queries"`
	Hits         []ValueHit             `json:"hits"`
	Warnings     []string               `json:"warnings,omitempty"`
}

type ValueScanSummary struct {
	InflatedHeaderBytes int `json:"inflated_header_bytes"`
	BodyBytes           int `json:"body_bytes"`
	QueryCount          int `json:"query_count"`
	HitCount            int `json:"hit_count"`
	Shown               int `json:"shown"`
}

type ValueQuery struct {
	Label   string `json:"label"`
	Kind    string `json:"kind"`
	Pattern string `json:"pattern"`
	Bytes   int    `json:"bytes"`
}

type ValueHit struct {
	QueryLabel string `json:"query_label"`
	QueryKind  string `json:"query_kind"`
	Space      string `json:"space"`
	Offset     int    `json:"offset"`
	End        int    `json:"end"`
	Bytes      int    `json:"bytes"`
	Region     string `json:"region,omitempty"`
	Status     string `json:"status,omitempty"`
	ContextHex string `json:"context_hex,omitempty"`
}

type compiledValueQuery struct {
	ValueQuery
	bytes []byte
}

func BuildValueScan(path string, opts ValueScanOptions) (*ValueScanReport, error) {
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
	queries, warnings := compileValueQueries(opts)
	header := rec.HeaderBytes()
	body := replayBody(data, rec.HeaderLength)
	report := &ValueScanReport{
		Path: path,
		Verification: aoe2.VerificationClaim{
			Label:             "structure_verified_not_engine_verified",
			StructureVerified: true,
			EngineVerified:    false,
			Note:              "Exact byte-pattern search over inflated replay header and body; hits are evidence candidates, not semantic decode.",
		},
		Summary: ValueScanSummary{
			InflatedHeaderBytes: len(header),
			BodyBytes:           len(body),
			QueryCount:          len(queries),
		},
		Warnings: append(append([]string(nil), coverage.Warnings...), warnings...),
	}
	index := indexRegions(coverage.Regions)
	for _, q := range queries {
		report.Queries = append(report.Queries, q.ValueQuery)
		if opts.Space == "" || opts.Space == "inflated_header" {
			report.Hits = append(report.Hits, scanValueSpace(q, "inflated_header", header, index)...)
		}
		if opts.Space == "" || opts.Space == "body" {
			report.Hits = append(report.Hits, scanValueSpace(q, "body", body, index)...)
		}
	}
	sort.Slice(report.Hits, func(i, j int) bool {
		if report.Hits[i].Space == report.Hits[j].Space {
			if report.Hits[i].Offset == report.Hits[j].Offset {
				return report.Hits[i].QueryLabel < report.Hits[j].QueryLabel
			}
			return report.Hits[i].Offset < report.Hits[j].Offset
		}
		return report.Hits[i].Space < report.Hits[j].Space
	})
	report.Summary.HitCount = len(report.Hits)
	if opts.Limit > 0 && len(report.Hits) > opts.Limit {
		report.Hits = report.Hits[:opts.Limit]
	}
	report.Summary.Shown = len(report.Hits)
	return report, nil
}

func compileValueQueries(opts ValueScanOptions) ([]compiledValueQuery, []string) {
	var queries []compiledValueQuery
	var warnings []string
	for _, raw := range opts.Values {
		value, err := strconv.ParseInt(raw, 0, 64)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("value %q skipped: %v", raw, err))
			continue
		}
		queries = append(queries, numericValueQueries(raw, value, opts.NumericWidths)...)
	}
	for _, raw := range opts.Hex {
		clean := strings.TrimPrefix(strings.TrimPrefix(strings.TrimSpace(raw), "0x"), "0X")
		if len(clean)%2 != 0 {
			clean = "0" + clean
		}
		bytes, err := hex.DecodeString(clean)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("hex %q skipped: %v", raw, err))
			continue
		}
		if len(bytes) == 0 {
			continue
		}
		queries = append(queries, compiledValueQuery{
			ValueQuery: ValueQuery{Label: raw, Kind: "hex", Pattern: hex.EncodeToString(bytes), Bytes: len(bytes)},
			bytes:      bytes,
		})
	}
	for _, text := range opts.Text {
		if text == "" {
			continue
		}
		bytes := []byte(text)
		queries = append(queries, compiledValueQuery{
			ValueQuery: ValueQuery{Label: text, Kind: "text_ascii", Pattern: hex.EncodeToString(bytes), Bytes: len(bytes)},
			bytes:      bytes,
		})
		queries = append(queries, compiledValueQuery{
			ValueQuery: ValueQuery{Label: text, Kind: "text_utf16le", Pattern: hex.EncodeToString(utf16LEBytes(text)), Bytes: len(text) * 2},
			bytes:      utf16LEBytes(text),
		})
	}
	return queries, warnings
}

func numericValueQueries(label string, value int64, widths []int) []compiledValueQuery {
	var out []compiledValueQuery
	if widthEnabled(widths, 8) && value >= -128 && value <= 255 {
		out = append(out, compiledValueQuery{
			ValueQuery: ValueQuery{Label: label, Kind: "int8", Pattern: fmt.Sprintf("%02x", byte(value)), Bytes: 1},
			bytes:      []byte{byte(value)},
		})
	}
	if widthEnabled(widths, 16) && value >= -32768 && value <= 65535 {
		var buf [2]byte
		binary.LittleEndian.PutUint16(buf[:], uint16(value))
		out = append(out, compiledValueQuery{
			ValueQuery: ValueQuery{Label: label, Kind: "int16le", Pattern: hex.EncodeToString(buf[:]), Bytes: 2},
			bytes:      append([]byte(nil), buf[:]...),
		})
	}
	if widthEnabled(widths, 32) && value >= -2147483648 && value <= 4294967295 {
		var buf [4]byte
		binary.LittleEndian.PutUint32(buf[:], uint32(value))
		out = append(out, compiledValueQuery{
			ValueQuery: ValueQuery{Label: label, Kind: "int32le", Pattern: hex.EncodeToString(buf[:]), Bytes: 4},
			bytes:      append([]byte(nil), buf[:]...),
		})
	}
	if widthEnabled(widths, 64) {
		var buf [8]byte
		binary.LittleEndian.PutUint64(buf[:], uint64(value))
		out = append(out, compiledValueQuery{
			ValueQuery: ValueQuery{Label: label, Kind: "int64le", Pattern: hex.EncodeToString(buf[:]), Bytes: 8},
			bytes:      append([]byte(nil), buf[:]...),
		})
	}
	return out
}

func widthEnabled(widths []int, width int) bool {
	if len(widths) == 0 {
		return true
	}
	for _, candidate := range widths {
		if candidate == width {
			return true
		}
	}
	return false
}

func utf16LEBytes(text string) []byte {
	out := make([]byte, 0, len(text)*2)
	for _, r := range text {
		if r > 0xffff {
			continue
		}
		var buf [2]byte
		binary.LittleEndian.PutUint16(buf[:], uint16(r))
		out = append(out, buf[:]...)
	}
	return out
}

func scanValueSpace(query compiledValueQuery, space string, data []byte, index replayRegionIndex) []ValueHit {
	if len(query.bytes) == 0 {
		return nil
	}
	var hits []ValueHit
	start := 0
	for start <= len(data)-len(query.bytes) {
		pos := bytes.Index(data[start:], query.bytes)
		if pos < 0 {
			break
		}
		off := start + pos
		end := off + len(query.bytes)
		region := index.regionForRange(space, off, end)
		hits = append(hits, ValueHit{
			QueryLabel: query.Label,
			QueryKind:  query.Kind,
			Space:      space,
			Offset:     off,
			End:        end,
			Bytes:      len(query.bytes),
			Region:     replayRegionName(region),
			Status:     replayRegionStatus(region),
			ContextHex: HexSample(contextBytes(data, off, end, 16), 96),
		})
		start = off + 1
	}
	return hits
}

func contextBytes(data []byte, start int, end int, radius int) []byte {
	lo := start - radius
	if lo < 0 {
		lo = 0
	}
	hi := end + radius
	if hi > len(data) {
		hi = len(data)
	}
	return data[lo:hi]
}
