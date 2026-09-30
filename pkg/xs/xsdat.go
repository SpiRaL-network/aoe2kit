package xs

import (
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"unicode/utf8"
)

type DataInspectOptions struct {
	Types         []string
	MaxAutoString int
}

type DataInspectReport struct {
	Path              string            `json:"path"`
	SizeBytes         int               `json:"size_bytes"`
	Mode              string            `json:"mode"`
	OK                bool              `json:"ok"`
	Verification      VerificationClaim `json:"verification"`
	Values            []DataValue       `json:"values"`
	RemainingBytes    int               `json:"remaining_bytes,omitempty"`
	RemainingOffset   int               `json:"remaining_offset,omitempty"`
	RemainingHex      string            `json:"remaining_hex,omitempty"`
	Errors            []string          `json:"errors,omitempty"`
	HeuristicWarnings []string          `json:"heuristic_warnings,omitempty"`
}

type DataValue struct {
	Index          int       `json:"index"`
	Offset         int       `json:"offset"`
	Type           string    `json:"type"`
	Size           int       `json:"size"`
	String         string    `json:"string,omitempty"`
	Int            *int32    `json:"int,omitempty"`
	UInt           *uint32   `json:"uint,omitempty"`
	Float          *float32  `json:"float,omitempty"`
	Vector         []float32 `json:"vector,omitempty"`
	RawHex         string    `json:"raw_hex,omitempty"`
	Heuristic      bool      `json:"heuristic,omitempty"`
	FloatCandidate *float32  `json:"float_candidate,omitempty"`
	Note           string    `json:"note,omitempty"`
}

// MarshalJSON keeps ordinary float fields numeric, but represents NaN and
// infinities as strings so one non-finite engine result cannot invalidate the
// entire diagnostic report. This is an output-only policy; the decoded Float
// pointer still contains the original IEEE value for callers that need it.
func (value DataValue) MarshalJSON() ([]byte, error) {
	type wire struct {
		Index          int     `json:"index"`
		Offset         int     `json:"offset"`
		Type           string  `json:"type"`
		Size           int     `json:"size"`
		String         string  `json:"string,omitempty"`
		Int            *int32  `json:"int,omitempty"`
		UInt           *uint32 `json:"uint,omitempty"`
		Float          any     `json:"float,omitempty"`
		Vector         []any   `json:"vector,omitempty"`
		RawHex         string  `json:"raw_hex,omitempty"`
		Heuristic      bool    `json:"heuristic,omitempty"`
		FloatCandidate any     `json:"float_candidate,omitempty"`
		Note           string  `json:"note,omitempty"`
	}
	out := wire{
		Index: value.Index, Offset: value.Offset, Type: value.Type, Size: value.Size,
		String: value.String, Int: value.Int, UInt: value.UInt, RawHex: value.RawHex,
		Heuristic: value.Heuristic, Note: value.Note,
	}
	if value.Float != nil {
		out.Float = jsonSafeFloat32(*value.Float)
	}
	if len(value.Vector) > 0 {
		out.Vector = make([]any, len(value.Vector))
		for i, component := range value.Vector {
			out.Vector[i] = jsonSafeFloat32(component)
		}
	}
	if value.FloatCandidate != nil {
		out.FloatCandidate = jsonSafeFloat32(*value.FloatCandidate)
	}
	return json.Marshal(out)
}

func jsonSafeFloat32(value float32) any {
	switch {
	case math.IsNaN(float64(value)):
		return "NaN"
	case math.IsInf(float64(value), 1):
		return "+Inf"
	case math.IsInf(float64(value), -1):
		return "-Inf"
	default:
		return value
	}
}

func InspectDataFile(path string, opts DataInspectOptions) (DataInspectReport, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return DataInspectReport{}, err
	}
	report := InspectDataBytes(data, opts)
	report.Path = path
	return report, nil
}

func InspectDataBytes(data []byte, opts DataInspectOptions) DataInspectReport {
	if opts.MaxAutoString <= 0 {
		opts.MaxAutoString = 1 << 20
	}
	report := DataInspectReport{
		SizeBytes: len(data),
		Verification: StructureVerifiedClaim(
			"XS .xsdat inspection decodes documented xsWriteString/xsWriteInt/xsWriteFloat/xsWriteVector byte encodings; higher-level meaning depends on the scenario's write order.",
		),
	}
	if len(opts.Types) > 0 {
		report.Mode = "typed"
		parseTypedData(data, opts.Types, &report)
	} else {
		report.Mode = "auto_heuristic"
		report.HeuristicWarnings = append(report.HeuristicWarnings, "auto mode has no type tags; it treats sane length-prefixed printable bytes as strings and otherwise decodes 4-byte words as int/float candidates")
		parseAutoData(data, opts, &report)
	}
	report.OK = len(report.Errors) == 0 && report.RemainingBytes == 0
	return report
}

func ParseDataTypes(spec string) ([]string, error) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return nil, nil
	}
	parts := strings.Split(spec, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		t, err := normalizeDataType(part)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, nil
}

func normalizeDataType(t string) (string, error) {
	t = strings.ToLower(strings.TrimSpace(t))
	switch t {
	case "s", "str", "string":
		return "string", nil
	case "i", "int", "int32":
		return "int", nil
	case "u", "uint", "uint32":
		return "uint", nil
	case "f", "float", "float32":
		return "float", nil
	case "v", "vec", "vector":
		return "vector", nil
	case "raw", "bytes":
		return "raw", nil
	default:
		if strings.HasPrefix(t, "raw:") {
			n, err := strconv.Atoi(strings.TrimPrefix(t, "raw:"))
			if err != nil || n < 0 {
				return "", fmt.Errorf("invalid raw byte count %q", t)
			}
			return t, nil
		}
		return "", fmt.Errorf("unsupported xsdat type %q", t)
	}
}

func parseTypedData(data []byte, types []string, report *DataInspectReport) {
	offset := 0
	for i, typ := range types {
		value, next, err := readTypedValue(data, offset, i, typ, false)
		if err != nil {
			report.Errors = append(report.Errors, err.Error())
			break
		}
		report.Values = append(report.Values, value)
		offset = next
	}
	if offset < len(data) {
		report.RemainingOffset = offset
		report.RemainingBytes = len(data) - offset
		report.RemainingHex = hex.EncodeToString(data[offset:])
	}
}

func parseAutoData(data []byte, opts DataInspectOptions, report *DataInspectReport) {
	offset := 0
	for offset < len(data) {
		index := len(report.Values)
		if value, next, ok := tryAutoString(data, offset, index, opts.MaxAutoString); ok {
			report.Values = append(report.Values, value)
			offset = next
			continue
		}
		if len(data)-offset >= 4 {
			raw := binary.LittleEndian.Uint32(data[offset : offset+4])
			i := int32(raw)
			f := math.Float32frombits(raw)
			report.Values = append(report.Values, DataValue{
				Index:          index,
				Offset:         offset,
				Type:           "int_or_float",
				Size:           4,
				Int:            &i,
				UInt:           &raw,
				FloatCandidate: &f,
				RawHex:         hex.EncodeToString(data[offset : offset+4]),
				Heuristic:      true,
			})
			offset += 4
			continue
		}
		report.Values = append(report.Values, DataValue{
			Index:     index,
			Offset:    offset,
			Type:      "raw",
			Size:      len(data) - offset,
			RawHex:    hex.EncodeToString(data[offset:]),
			Heuristic: true,
			Note:      "trailing bytes shorter than an int/float word",
		})
		offset = len(data)
	}
}

func tryAutoString(data []byte, offset, index, maxLen int) (DataValue, int, bool) {
	if len(data)-offset < 4 {
		return DataValue{}, offset, false
	}
	n := binary.LittleEndian.Uint32(data[offset : offset+4])
	if n == 0 || n > uint32(maxLen) || int(n) > len(data)-offset-4 {
		return DataValue{}, offset, false
	}
	raw := data[offset+4 : offset+4+int(n)]
	if !saneXSString(raw) {
		return DataValue{}, offset, false
	}
	return DataValue{
		Index:     index,
		Offset:    offset,
		Type:      "string",
		Size:      4 + int(n),
		String:    string(raw),
		RawHex:    hex.EncodeToString(data[offset : offset+4+int(n)]),
		Heuristic: true,
	}, offset + 4 + int(n), true
}

func saneXSString(raw []byte) bool {
	if !utf8.Valid(raw) {
		return false
	}
	for _, r := range string(raw) {
		switch r {
		case '\t', '\n', '\r':
			continue
		}
		if r < 32 {
			return false
		}
	}
	return true
}

func readTypedValue(data []byte, offset, index int, typ string, heuristic bool) (DataValue, int, error) {
	if strings.HasPrefix(typ, "raw:") {
		n, _ := strconv.Atoi(strings.TrimPrefix(typ, "raw:"))
		if len(data)-offset < n {
			return DataValue{}, offset, fmt.Errorf("value %d raw:%d at offset %d exceeds file size %d", index, n, offset, len(data))
		}
		return DataValue{Index: index, Offset: offset, Type: "raw", Size: n, RawHex: hex.EncodeToString(data[offset : offset+n]), Heuristic: heuristic}, offset + n, nil
	}
	switch typ {
	case "string":
		if len(data)-offset < 4 {
			return DataValue{}, offset, fmt.Errorf("value %d string at offset %d lacks length prefix", index, offset)
		}
		n := binary.LittleEndian.Uint32(data[offset : offset+4])
		if int(n) > len(data)-offset-4 {
			return DataValue{}, offset, fmt.Errorf("value %d string length %d at offset %d exceeds remaining bytes %d", index, n, offset, len(data)-offset-4)
		}
		raw := data[offset+4 : offset+4+int(n)]
		return DataValue{Index: index, Offset: offset, Type: "string", Size: 4 + int(n), String: string(raw), RawHex: hex.EncodeToString(data[offset : offset+4+int(n)]), Heuristic: heuristic}, offset + 4 + int(n), nil
	case "int":
		if len(data)-offset < 4 {
			return DataValue{}, offset, fmt.Errorf("value %d int at offset %d exceeds file size %d", index, offset, len(data))
		}
		raw := binary.LittleEndian.Uint32(data[offset : offset+4])
		i := int32(raw)
		return DataValue{Index: index, Offset: offset, Type: "int", Size: 4, Int: &i, UInt: &raw, RawHex: hex.EncodeToString(data[offset : offset+4]), Heuristic: heuristic}, offset + 4, nil
	case "uint":
		if len(data)-offset < 4 {
			return DataValue{}, offset, fmt.Errorf("value %d uint at offset %d exceeds file size %d", index, offset, len(data))
		}
		raw := binary.LittleEndian.Uint32(data[offset : offset+4])
		return DataValue{Index: index, Offset: offset, Type: "uint", Size: 4, UInt: &raw, RawHex: hex.EncodeToString(data[offset : offset+4]), Heuristic: heuristic}, offset + 4, nil
	case "float":
		if len(data)-offset < 4 {
			return DataValue{}, offset, fmt.Errorf("value %d float at offset %d exceeds file size %d", index, offset, len(data))
		}
		raw := binary.LittleEndian.Uint32(data[offset : offset+4])
		f := math.Float32frombits(raw)
		return DataValue{Index: index, Offset: offset, Type: "float", Size: 4, Float: &f, RawHex: hex.EncodeToString(data[offset : offset+4]), Heuristic: heuristic}, offset + 4, nil
	case "vector":
		if len(data)-offset < 12 {
			return DataValue{}, offset, fmt.Errorf("value %d vector at offset %d exceeds file size %d", index, offset, len(data))
		}
		vec := []float32{
			math.Float32frombits(binary.LittleEndian.Uint32(data[offset : offset+4])),
			math.Float32frombits(binary.LittleEndian.Uint32(data[offset+4 : offset+8])),
			math.Float32frombits(binary.LittleEndian.Uint32(data[offset+8 : offset+12])),
		}
		return DataValue{Index: index, Offset: offset, Type: "vector", Size: 12, Vector: vec, RawHex: hex.EncodeToString(data[offset : offset+12]), Heuristic: heuristic}, offset + 12, nil
	case "raw":
		return DataValue{Index: index, Offset: offset, Type: "raw", Size: len(data) - offset, RawHex: hex.EncodeToString(data[offset:]), Heuristic: heuristic}, len(data), nil
	default:
		return DataValue{}, offset, fmt.Errorf("unsupported xsdat type %q", typ)
	}
}
