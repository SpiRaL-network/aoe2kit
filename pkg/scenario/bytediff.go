package scenario

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"unicode"
)

type ByteDiffOptions struct {
	IncludeNoise bool `json:"include_noise"`
}

type ByteDiffReport struct {
	BeforePath             string           `json:"before_path,omitempty"`
	AfterPath              string           `json:"after_path,omitempty"`
	BeforeBytes            int              `json:"before_inflated_bytes"`
	AfterBytes             int              `json:"after_inflated_bytes"`
	Mode                   string           `json:"mode"`
	LengthDelta            int              `json:"length_delta"`
	FirstDivergence        int              `json:"first_divergence"`
	FirstDivergenceSection string           `json:"first_divergence_section,omitempty"`
	Warning                string           `json:"warning,omitempty"`
	NoiseSuppressed        []SuppressedRun  `json:"noise_suppressed,omitempty"`
	Changes                []ByteDiffRun    `json:"changes,omitempty"`
	Verification           string           `json:"verification"`
	Notes                  []string         `json:"notes,omitempty"`
	ComparedByteSpan       ComparedByteSpan `json:"compared_byte_span"`
}

type ComparedByteSpan struct {
	Start int `json:"start"`
	End   int `json:"end"`
}

type SuppressedRun struct {
	Kind   string `json:"kind"`
	Start  int    `json:"start"`
	End    int    `json:"end"`
	Detail string `json:"detail,omitempty"`
}

type ByteDiffRun struct {
	Start       int              `json:"start"`
	StartHex    string           `json:"start_hex"`
	End         int              `json:"end"`
	Length      int              `json:"length"`
	Section     string           `json:"section,omitempty"`
	BeforeHex   string           `json:"before_hex,omitempty"`
	AfterHex    string           `json:"after_hex,omitempty"`
	BeforeText  string           `json:"before_ascii,omitempty"`
	AfterText   string           `json:"after_ascii,omitempty"`
	Decodes     ByteDiffDecodes  `json:"decodes"`
	ByteChanges []ByteChangeInfo `json:"byte_changes,omitempty"`
}

type ByteChangeInfo struct {
	Offset      int    `json:"offset"`
	OffsetHex   string `json:"offset_hex"`
	BeforeUint8 uint8  `json:"before_uint8"`
	AfterUint8  uint8  `json:"after_uint8"`
	BeforeASCII string `json:"before_ascii"`
	AfterASCII  string `json:"after_ascii"`
}

type ByteDiffDecodes struct {
	Int16LE *BeforeAfter[int16]  `json:"int16_le,omitempty"`
	Int32LE *BeforeAfter[int32]  `json:"int32_le,omitempty"`
	Uint8   *BeforeAfter[uint8]  `json:"uint8,omitempty"`
	ASCII   *BeforeAfter[string] `json:"ascii,omitempty"`
	FieldLE *BeforeAfter[int32]  `json:"field_int32_le,omitempty"`
}

type byteSpan struct {
	start int
	end   int
}

type knownNoiseField struct {
	span byteSpan
	raw  []byte
}

type BeforeAfter[T any] struct {
	Before T `json:"before"`
	After  T `json:"after"`
}

func ByteDiffFiles(beforePath, afterPath string, opts ByteDiffOptions) (ByteDiffReport, error) {
	before, beforeProtected, beforeNoise, beforeSections, err := inflatedBodyAndSpans(beforePath)
	if err != nil {
		return ByteDiffReport{}, err
	}
	after, afterProtected, afterNoise, afterSections, err := inflatedBodyAndSpans(afterPath)
	if err != nil {
		return ByteDiffReport{}, err
	}
	report := byteDiffBodies(before, after, opts, append(beforeProtected, afterProtected...), beforeNoise, afterNoise, beforeSections, afterSections)
	report.BeforePath = beforePath
	report.AfterPath = afterPath
	return report, nil
}

func ByteDiffBodies(before, after []byte, opts ByteDiffOptions) ByteDiffReport {
	return byteDiffBodies(before, after, opts, nil, nil, nil, nil, nil)
}

func byteDiffBodies(before, after []byte, opts ByteDiffOptions, protectedSpans []byteSpan, beforeNoise, afterNoise map[string]knownNoiseField, beforeSections, afterSections []SectionInfo) ByteDiffReport {
	report := ByteDiffReport{
		BeforeBytes:  len(before),
		AfterBytes:   len(after),
		LengthDelta:  len(after) - len(before),
		Verification: "structure_verified_not_engine_verified",
		Notes:        []string{"comparison covers inflated scenario body; file-header fields such as timestamp_of_last_save are outside this byte span"},
	}
	limit := len(before)
	if len(after) < limit {
		limit = len(after)
	}
	comparedBefore := append([]byte(nil), before...)
	comparedAfter := append([]byte(nil), after...)
	if !opts.IncludeNoise {
		maskKnownNoiseFields(comparedBefore, beforeNoise)
		maskKnownNoiseFields(comparedAfter, afterNoise)
		report.NoiseSuppressed = changedKnownNoiseFields(beforeNoise, afterNoise)
	}
	report.FirstDivergence = firstDivergence(comparedBefore, comparedAfter)
	report.ComparedByteSpan = ComparedByteSpan{Start: 0, End: limit}
	if report.FirstDivergence >= 0 {
		report.FirstDivergenceSection = sectionAt(report.FirstDivergence, beforeSections)
		if report.FirstDivergenceSection == "" {
			report.FirstDivergenceSection = sectionAt(report.FirstDivergence, afterSections)
		}
	}
	if len(before) == len(after) {
		report.Mode = "in-place"
	} else {
		report.Mode = "insert/delete"
		report.Warning = fmt.Sprintf("raw byte offsets at and after divergence %d are shift-noise; only changes before that offset are unshifted", report.FirstDivergence)
		limit = report.FirstDivergence
		report.ComparedByteSpan.End = limit
	}

	runs := changedRuns(comparedBefore, comparedAfter, limit)
	if !opts.IncludeNoise {
		var kept []byteRun
		for _, run := range runs {
			if likelyDerivedNonce(run, len(before), len(after)) && !runOverlapsSpans(run, protectedSpans) {
				report.NoiseSuppressed = append(report.NoiseSuppressed, SuppressedRun{
					Kind:   "likely_per_player_nonce",
					Start:  run.start,
					End:    run.end,
					Detail: "late short high-entropy run near editor/player marker state; suppressed as derived save noise",
				})
				continue
			}
			kept = append(kept, run)
		}
		runs = kept
		report.Notes = append(report.Notes, "suppressed likely per-player derived nonce runs")
	}
	for _, run := range runs {
		change := decodeByteRun(before, after, run)
		change.Section = sectionAt(run.start, beforeSections)
		if change.Section == "" {
			change.Section = sectionAt(run.start, afterSections)
		}
		report.Changes = append(report.Changes, change)
	}
	return report
}

func inflatedBodyAndSpans(path string) ([]byte, []byteSpan, map[string]knownNoiseField, []SectionInfo, error) {
	file, err := Open(path)
	if err == nil {
		return file.body, protectedByteSpans(file), knownSaveNoiseFields(file), file.Sections, nil
	}
	body, bodyErr := InflatedBodyFile(path)
	if bodyErr != nil {
		return nil, nil, nil, nil, bodyErr
	}
	return body, nil, nil, nil, nil
}

func knownSaveNoiseFields(file *File) map[string]knownNoiseField {
	if file == nil || file.root == nil {
		return nil
	}
	fields := collectFileFields(file)
	wanted := []string{"DataHeader.next_unit_id_to_place"}
	for i := 0; i < 16; i++ {
		for _, axis := range []string{"x", "y"} {
			wanted = append(wanted, fmt.Sprintf("Units.player_data_3[%d].editor_camera_%s", i, axis))
		}
	}
	noise := make(map[string]knownNoiseField)
	for _, path := range wanted {
		field, ok := fields[path]
		if !ok || len(field.Raw) == 0 {
			continue
		}
		noise[path] = knownNoiseField{span: byteSpan{start: field.Start, end: field.Start + len(field.Raw)}, raw: append([]byte(nil), field.Raw...)}
	}
	return noise
}

func maskKnownNoiseFields(data []byte, fields map[string]knownNoiseField) {
	for _, field := range fields {
		span := field.span
		start, end := maxInt(0, span.start), minInt(len(data), span.end)
		for i := start; i < end; i++ {
			data[i] = 0
		}
	}
}

func changedKnownNoiseFields(before, after map[string]knownNoiseField) []SuppressedRun {
	var out []SuppressedRun
	for path, old := range before {
		current, ok := after[path]
		if ok && bytes.Equal(old.raw, current.raw) {
			continue
		}
		out = append(out, SuppressedRun{Kind: "known_save_field", Start: old.span.start, End: old.span.end, Detail: path})
	}
	for path, current := range after {
		if _, ok := before[path]; ok {
			continue
		}
		out = append(out, SuppressedRun{Kind: "known_save_field", Start: current.span.start, End: current.span.end, Detail: path})
	}
	return out
}

func sectionAt(offset int, sections []SectionInfo) string {
	for _, section := range sections {
		if offset >= section.Start && offset < section.End {
			return section.Name
		}
	}
	return ""
}

func protectedByteSpans(file *File) []byteSpan {
	if file == nil || file.root == nil {
		return nil
	}
	triggers := file.root.section("Triggers")
	if triggers == nil {
		return nil
	}
	return []byteSpan{{start: triggers.Start, end: triggers.End}}
}

func runOverlapsSpans(run byteRun, spans []byteSpan) bool {
	for _, span := range spans {
		if run.start < span.end && run.end > span.start {
			return true
		}
	}
	return false
}

type byteRun struct {
	start int
	end   int
}

func firstDivergence(before, after []byte) int {
	limit := len(before)
	if len(after) < limit {
		limit = len(after)
	}
	for i := 0; i < limit; i++ {
		if before[i] != after[i] {
			return i
		}
	}
	if len(before) != len(after) {
		return limit
	}
	return -1
}

func changedRuns(before, after []byte, limit int) []byteRun {
	var runs []byteRun
	for i := 0; i < limit; {
		if before[i] == after[i] {
			i++
			continue
		}
		start := i
		for i < limit && before[i] != after[i] {
			i++
		}
		runs = append(runs, byteRun{start: start, end: i})
	}
	return runs
}

func likelyDerivedNonce(run byteRun, beforeLen, afterLen int) bool {
	if run.start < beforeLen*3/4 || run.start < afterLen*3/4 {
		return false
	}
	length := run.end - run.start
	return length >= 4 && length <= 12
}

func decodeByteRun(before, after []byte, run byteRun) ByteDiffRun {
	beforeChunk := append([]byte(nil), before[run.start:run.end]...)
	afterChunk := append([]byte(nil), after[run.start:run.end]...)
	out := ByteDiffRun{
		Start:      run.start,
		StartHex:   fmt.Sprintf("0x%x", run.start),
		End:        run.end,
		Length:     run.end - run.start,
		BeforeHex:  fmt.Sprintf("% x", beforeChunk),
		AfterHex:   fmt.Sprintf("% x", afterChunk),
		BeforeText: asciiString(beforeChunk),
		AfterText:  asciiString(afterChunk),
	}
	for off := run.start; off < run.end; off++ {
		out.ByteChanges = append(out.ByteChanges, ByteChangeInfo{
			Offset:      off,
			OffsetHex:   fmt.Sprintf("0x%x", off),
			BeforeUint8: before[off],
			AfterUint8:  after[off],
			BeforeASCII: asciiByte(before[off]),
			AfterASCII:  asciiByte(after[off]),
		})
	}
	if len(beforeChunk) == 1 {
		out.Decodes.Uint8 = &BeforeAfter[uint8]{Before: beforeChunk[0], After: afterChunk[0]}
		out.Decodes.ASCII = &BeforeAfter[string]{Before: asciiByte(beforeChunk[0]), After: asciiByte(afterChunk[0])}
	}
	if len(beforeChunk) >= 2 {
		out.Decodes.Int16LE = &BeforeAfter[int16]{
			Before: int16(binary.LittleEndian.Uint16(beforeChunk[:2])),
			After:  int16(binary.LittleEndian.Uint16(afterChunk[:2])),
		}
	}
	if len(beforeChunk) >= 4 {
		out.Decodes.Int32LE = &BeforeAfter[int32]{
			Before: int32(binary.LittleEndian.Uint32(beforeChunk[:4])),
			After:  int32(binary.LittleEndian.Uint32(afterChunk[:4])),
		}
	}
	if field := nearbyInt32FieldDecode(before, after, run); field != nil {
		out.Decodes.FieldLE = field
	}
	return out
}

func nearbyInt32FieldDecode(before, after []byte, run byteRun) *BeforeAfter[int32] {
	for _, start := range []int{run.start, run.start - 1, run.start - 2, run.start - 3} {
		if start < 0 || start+4 > len(before) || start+4 > len(after) {
			continue
		}
		b := int32(binary.LittleEndian.Uint32(before[start : start+4]))
		a := int32(binary.LittleEndian.Uint32(after[start : start+4]))
		if b >= -1000000 && b <= 1000000 && a >= -1000000 && a <= 1000000 {
			return &BeforeAfter[int32]{Before: b, After: a}
		}
	}
	if run.start+4 <= len(before) && run.start+4 <= len(after) {
		return &BeforeAfter[int32]{
			Before: int32(binary.LittleEndian.Uint32(before[run.start : run.start+4])),
			After:  int32(binary.LittleEndian.Uint32(after[run.start : run.start+4])),
		}
	}
	return nil
}

func asciiString(data []byte) string {
	out := make([]rune, 0, len(data))
	for _, b := range data {
		if unicode.IsPrint(rune(b)) && b < 0x7f {
			out = append(out, rune(b))
		} else {
			out = append(out, '.')
		}
	}
	return string(out)
}

func asciiByte(b byte) string {
	if unicode.IsPrint(rune(b)) && b < 0x7f {
		return string(rune(b))
	}
	return "."
}
