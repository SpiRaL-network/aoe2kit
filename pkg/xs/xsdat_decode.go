package xs

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
)

type DataDecodeOptions struct {
	Types  []string
	Ledger string
}

type DataDecodeReport struct {
	Path         string              `json:"path,omitempty"`
	OK           bool                `json:"ok"`
	Verification VerificationClaim   `json:"verification"`
	Decode       DataInspectReport   `json:"decode"`
	Ledger       string              `json:"ledger,omitempty"`
	Assertions   []DataAssertion     `json:"assertions,omitempty"`
	Summary      DataAssertionCounts `json:"summary"`
	Errors       []string            `json:"errors,omitempty"`
}

type DataAssertionCounts struct {
	Total   int `json:"total"`
	Passed  int `json:"passed"`
	Failed  int `json:"failed"`
	Unknown int `json:"unknown"`
}

type DataAssertion struct {
	Index    int    `json:"index"`
	Name     string `json:"name,omitempty"`
	Expected string `json:"expected"`
	Actual   string `json:"actual,omitempty"`
	Status   string `json:"status"`
	Message  string `json:"message,omitempty"`
}

type expectedDataValue struct {
	Type  string
	Value string
	Any   bool
}

func DecodeDataFile(path string, opts DataDecodeOptions) (DataDecodeReport, error) {
	expected := []expectedDataValue{}
	var err error
	if opts.Ledger != "" {
		expected, err = LoadExpectedDataLedger(opts.Ledger)
		if err != nil {
			return DataDecodeReport{}, err
		}
		if len(opts.Types) == 0 {
			opts.Types = expectedDataTypes(expected)
		}
	}
	decode, err := InspectDataFile(path, DataInspectOptions{Types: opts.Types})
	if err != nil {
		return DataDecodeReport{}, err
	}
	report := DataDecodeReport{
		Path:   path,
		Decode: decode,
		Ledger: opts.Ledger,
		Verification: StructureVerifiedClaim(
			"XS sidecar decode uses documented .xsdat byte encodings and optional ledger assertions; engine behavior is verified only by the scenario run that produced the sidecar.",
		),
	}
	if opts.Ledger != "" {
		report.Assertions = compareDataAssertions(expected, decode.Values)
		report.Summary = summarizeDataAssertions(report.Assertions)
	} else {
		report.Summary = DataAssertionCounts{}
	}
	if len(decode.Errors) > 0 {
		report.Errors = append(report.Errors, decode.Errors...)
	}
	report.OK = decode.OK && (opts.Ledger == "" || report.Summary.Failed == 0 && report.Summary.Unknown == 0)
	return report, nil
}

func LoadExpectedDataLedger(path string) ([]expectedDataValue, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var root any
	if err := json.Unmarshal(data, &root); err != nil {
		return nil, err
	}
	if expected, ok, err := findStructuredLedgerPayload(root); ok || err != nil {
		return expected, err
	}
	payload, ok := findLedgerPayload(root)
	if !ok {
		return nil, fmt.Errorf("ledger has no payload array; supported paths are payload or writer.payload; supported structured ledgers include v8 rows")
	}
	out := make([]expectedDataValue, 0, len(payload))
	for i, item := range payload {
		switch v := item.(type) {
		case string:
			expected, err := parseExpectedDataString(v)
			if err != nil {
				return nil, fmt.Errorf("payload[%d]: %w", i, err)
			}
			out = append(out, expected)
		case map[string]any:
			expected, err := parseExpectedDataObject(v)
			if err != nil {
				return nil, fmt.Errorf("payload[%d]: %w", i, err)
			}
			out = append(out, expected)
		default:
			return nil, fmt.Errorf("payload[%d]: unsupported expected value %T", i, item)
		}
	}
	return out, nil
}

func findLedgerPayload(root any) ([]any, bool) {
	if arr, ok := root.([]any); ok {
		return arr, true
	}
	m, ok := root.(map[string]any)
	if !ok {
		return nil, false
	}
	if arr, ok := m["payload"].([]any); ok {
		return arr, true
	}
	if writer, ok := m["writer"].(map[string]any); ok {
		if arr, ok := writer["payload"].([]any); ok {
			return arr, true
		}
	}
	return nil, false
}

func parseExpectedDataString(s string) (expectedDataValue, error) {
	typ, rest, ok := strings.Cut(strings.TrimSpace(s), " ")
	if !ok {
		return expectedDataValue{}, fmt.Errorf("expected value must start with a type")
	}
	normalized, err := normalizeDataType(typ)
	if err != nil {
		return expectedDataValue{}, err
	}
	if normalized == "uint" {
		normalized = "int"
	}
	value := strings.TrimSpace(rest)
	return expectedDataValue{Type: normalized, Value: value, Any: isAnyExpectedValue(value)}, nil
}

func parseExpectedDataObject(m map[string]any) (expectedDataValue, error) {
	typeRaw, ok := m["type"].(string)
	if !ok || strings.TrimSpace(typeRaw) == "" {
		return expectedDataValue{}, fmt.Errorf("object expected value needs string field type")
	}
	typ, err := normalizeDataType(typeRaw)
	if err != nil {
		return expectedDataValue{}, err
	}
	value, ok := m["value"]
	if !ok {
		return expectedDataValue{}, fmt.Errorf("object expected value needs field value")
	}
	valueText := fmt.Sprint(value)
	return expectedDataValue{Type: typ, Value: valueText, Any: isAnyExpectedValue(valueText)}, nil
}

func expectedDataTypes(expected []expectedDataValue) []string {
	out := make([]string, 0, len(expected))
	for _, item := range expected {
		out = append(out, item.Type)
	}
	return out
}

func compareDataAssertions(expected []expectedDataValue, actual []DataValue) []DataAssertion {
	out := make([]DataAssertion, 0, maxDataInt(len(expected), len(actual)))
	n := maxDataInt(len(expected), len(actual))
	for i := 0; i < n; i++ {
		assertion := DataAssertion{Index: i, Status: "unknown"}
		if i >= len(expected) {
			assertion.Expected = "<none>"
			assertion.Actual = dataValueString(actual[i])
			assertion.Status = "fail"
			assertion.Message = "unexpected extra decoded value"
			out = append(out, assertion)
			continue
		}
		assertion.Expected = expected[i].Type + " " + expected[i].Value
		if i >= len(actual) {
			assertion.Status = "fail"
			assertion.Message = "expected value missing from decoded file"
			out = append(out, assertion)
			continue
		}
		assertion.Actual = dataValueString(actual[i])
		if dataValueMatches(expected[i], actual[i]) {
			assertion.Status = "pass"
		} else {
			assertion.Status = "fail"
			assertion.Message = "decoded value did not match expected ledger value"
		}
		out = append(out, assertion)
	}
	return out
}

func dataValueMatches(expected expectedDataValue, actual DataValue) bool {
	if expected.Any {
		return actual.Type == expected.Type
	}
	switch expected.Type {
	case "string":
		return actual.Type == "string" && actual.String == expected.Value
	case "int":
		if actual.Int == nil {
			return false
		}
		want, err := strconv.ParseInt(expected.Value, 10, 32)
		return err == nil && *actual.Int == int32(want)
	case "uint":
		if actual.UInt == nil {
			return false
		}
		want, err := strconv.ParseUint(expected.Value, 10, 32)
		return err == nil && *actual.UInt == uint32(want)
	case "float":
		if actual.Float == nil {
			return false
		}
		want, err := strconv.ParseFloat(expected.Value, 32)
		return err == nil && math.Abs(float64(*actual.Float)-want) < 0.0001
	case "vector":
		parts := strings.Split(expected.Value, ",")
		if len(parts) != 3 || len(actual.Vector) != 3 {
			return false
		}
		for i, part := range parts {
			want, err := strconv.ParseFloat(strings.TrimSpace(part), 32)
			if err != nil || math.Abs(float64(actual.Vector[i])-want) >= 0.0001 {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func isAnyExpectedValue(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	return value == "*" || value == "any"
}

func findStructuredLedgerPayload(root any) ([]expectedDataValue, bool, error) {
	m, ok := root.(map[string]any)
	if !ok {
		return nil, false, nil
	}
	rows, ok := m["rows"].([]any)
	if !ok {
		return nil, false, nil
	}
	fixture, _ := m["fixture"].(string)
	if !strings.Contains(strings.ToLower(fixture), "v8") {
		return nil, false, nil
	}
	expected, err := v8AssertionLedgerPayload(m, rows)
	return expected, true, err
}

func v8AssertionLedgerPayload(root map[string]any, rows []any) ([]expectedDataValue, error) {
	out := []expectedDataValue{
		{Type: "string", Value: "A2K_RTV8_ASSERTION"},
		{Type: "int", Value: "82"},
		{Type: "string", Value: "open_once_close_once_plus_append_probe"},
	}
	for i, raw := range rows {
		row, ok := raw.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("rows[%d]: expected object", i)
		}
		phase, ok := jsonNumberAsIntString(row["phase_id"])
		if !ok {
			return nil, fmt.Errorf("rows[%d]: missing numeric phase_id", i)
		}
		timer := "*"
		if v, ok := jsonNumberAsIntString(row["timer"]); ok {
			timer = v
		}
		food := "*"
		if v, ok := jsonNumberAsNumberString(row["expected_food"]); ok {
			food = v
		}
		counts := map[string]string{}
		if rawCounts, ok := row["expected_counts"].(map[string]any); ok {
			for k, v := range rawCounts {
				if text, ok := jsonNumberAsIntString(v); ok {
					counts[k] = text
				}
			}
		}
		out = append(out,
			expectedDataValue{Type: "string", Value: "phase"},
			expectedDataValue{Type: "int", Value: phase},
			expectedDataValue{Type: "int", Value: timer, Any: isAnyExpectedValue(timer)},
			expectedDataValue{Type: "float", Value: food, Any: isAnyExpectedValue(food)},
			expectedDataValue{Type: "int", Value: "*", Any: true},
			expectedDataValue{Type: "int", Value: "*", Any: true},
			expectedDataValue{Type: "int", Value: "*", Any: true},
			expectedDataValue{Type: "int", Value: "*", Any: true},
			expectedDataValue{Type: "int", Value: "*", Any: true},
			expectedDataValue{Type: "int", Value: "*", Any: true},
			expectedIntMaybeAny(counts, "p2_scout_448"),
			expectedDataValue{Type: "float", Value: "*", Any: true},
		)
	}
	if appendProbe, ok := root["append_probe"].(map[string]any); ok {
		_ = appendProbe
		out = append(out,
			expectedDataValue{Type: "string", Value: "end"},
			expectedDataValue{Type: "int", Value: strconv.Itoa(len(rows))},
			expectedDataValue{Type: "string", Value: "append_probe"},
			expectedDataValue{Type: "int", Value: "812382"},
		)
	}
	return out, nil
}

func expectedIntMaybeAny(values map[string]string, key string) expectedDataValue {
	if value, ok := values[key]; ok {
		return expectedDataValue{Type: "int", Value: value}
	}
	return expectedDataValue{Type: "int", Value: "*", Any: true}
}

func jsonNumberAsIntString(value any) (string, bool) {
	switch v := value.(type) {
	case float64:
		if math.Trunc(v) != v {
			return "", false
		}
		return strconv.FormatInt(int64(v), 10), true
	case int:
		return strconv.Itoa(v), true
	case json.Number:
		i, err := v.Int64()
		if err != nil {
			return "", false
		}
		return strconv.FormatInt(i, 10), true
	default:
		return "", false
	}
}

func jsonNumberAsNumberString(value any) (string, bool) {
	switch v := value.(type) {
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64), true
	case int:
		return strconv.Itoa(v), true
	case json.Number:
		return v.String(), true
	default:
		return "", false
	}
}

func dataValueString(value DataValue) string {
	switch value.Type {
	case "string":
		return "string " + value.String
	case "int":
		if value.Int != nil {
			return "int " + strconv.FormatInt(int64(*value.Int), 10)
		}
	case "uint":
		if value.UInt != nil {
			return "uint " + strconv.FormatUint(uint64(*value.UInt), 10)
		}
	case "float":
		if value.Float != nil {
			return "float " + strconv.FormatFloat(float64(*value.Float), 'g', -1, 32)
		}
	case "vector":
		parts := make([]string, 0, len(value.Vector))
		for _, component := range value.Vector {
			parts = append(parts, strconv.FormatFloat(float64(component), 'g', -1, 32))
		}
		return "vector " + strings.Join(parts, ",")
	case "int_or_float":
		if value.Int != nil {
			return "int_or_float " + strconv.FormatInt(int64(*value.Int), 10)
		}
	}
	return value.Type
}

func summarizeDataAssertions(assertions []DataAssertion) DataAssertionCounts {
	counts := DataAssertionCounts{Total: len(assertions)}
	for _, assertion := range assertions {
		switch assertion.Status {
		case "pass":
			counts.Passed++
		case "fail":
			counts.Failed++
		default:
			counts.Unknown++
		}
	}
	return counts
}

func maxDataInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
