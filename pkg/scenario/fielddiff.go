package scenario

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// StructureFieldDiff is a path-aligned field change. Offsets are diagnostic
// context only; paths remain stable when a preceding structure grows.
type StructureFieldDiff struct {
	Path         string `json:"path"`
	BeforeHex    string `json:"before_hex,omitempty"`
	AfterHex     string `json:"after_hex,omitempty"`
	BeforeValue  any    `json:"before_value,omitempty"`
	AfterValue   any    `json:"after_value,omitempty"`
	BeforeStart  int    `json:"before_start,omitempty"`
	AfterStart   int    `json:"after_start,omitempty"`
	BeforeBytes  int    `json:"before_bytes"`
	AfterBytes   int    `json:"after_bytes"`
	BeforeExists bool   `json:"before_exists"`
	AfterExists  bool   `json:"after_exists"`
}

type DiffOptions struct {
	AllFields bool
}

type fieldSnapshot struct {
	Path  string
	Start int
	Raw   []byte
	Value any
}

// FieldQueryResult is one stable-path lookup result for scen field.
type FieldQueryResult struct {
	File    string `json:"file"`
	Path    string `json:"path"`
	Version string `json:"version,omitempty"`
	Status  string `json:"status"`
	Error   string `json:"error,omitempty"`
	Value   any    `json:"value,omitempty"`
	Hex     string `json:"hex,omitempty"`
	Exists  bool   `json:"exists"`
}

type FieldQueryOptions struct {
	Strict bool
}

// QueryField searches one scenario file or every scenario below root. Results
// are sorted by file path so corpus queries are deterministic.
func QueryField(fieldPath, input string) ([]FieldQueryResult, error) {
	return QueryFieldWithOptions(fieldPath, input, FieldQueryOptions{})
}

func QueryFieldWithOptions(fieldPath, input string, opts FieldQueryOptions) ([]FieldQueryResult, error) {
	info, err := os.Stat(input)
	if err != nil {
		return nil, err
	}
	paths := []string{input}
	if info.IsDir() {
		paths = paths[:0]
		err = filepath.WalkDir(input, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() {
				return nil
			}
			if strings.HasSuffix(strings.ToLower(entry.Name()), ".aoe2scenario") {
				paths = append(paths, path)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
		sort.Strings(paths)
	}
	results := make([]FieldQueryResult, 0, len(paths))
	for _, path := range paths {
		result := FieldQueryResult{File: path, Path: fieldPath}
		version, versionErr := ReadScenarioVersionFile(path)
		result.Version = version
		if versionErr != nil {
			result.Status = "error"
			result.Error = versionErr.Error()
			results = append(results, result)
			if !info.IsDir() {
				return results, versionErr
			}
			continue
		}
		if !SupportsReadVersion(version) {
			result.Status = "unsupported"
			result.Error = unsupportedReadVersionError(version).Error()
			results = append(results, result)
			if !info.IsDir() {
				return results, errors.New(result.Error)
			}
			continue
		}
		file, err := Open(path)
		if err != nil {
			result.Status = "error"
			result.Error = fmt.Errorf("open %s: %w", path, err).Error()
			results = append(results, result)
			if !info.IsDir() {
				return results, errors.New(result.Error)
			}
			continue
		}
		fields := collectFileFields(file)
		field, ok := fields[fieldPath]
		result.Status = "ok"
		result.Exists = ok
		if ok {
			result.Value = field.Value
			result.Hex = hex.EncodeToString(field.Raw)
		}
		results = append(results, result)
	}
	if info.IsDir() {
		failures := 0
		for _, result := range results {
			if result.Status != "ok" {
				failures++
			}
		}
		if failures == len(results) && len(results) > 0 {
			return results, fmt.Errorf("all %d scenario files failed", failures)
		}
		if opts.Strict && failures > 0 {
			return results, fmt.Errorf("%d of %d scenario files failed", failures, len(results))
		}
	}
	return results, nil
}

func DiffFilesWithOptions(beforePath, afterPath string, opts DiffOptions) (DiffReport, error) {
	before, err := Open(beforePath)
	if err != nil {
		return DiffReport{}, fmt.Errorf("open before: %w", err)
	}
	after, err := Open(afterPath)
	if err != nil {
		return DiffReport{}, fmt.Errorf("open after: %w", err)
	}
	report := Diff(before, after)
	if opts.AllFields {
		report.Fields = diffStructureFields(before, after)
	}
	return report, nil
}

func diffStructureFields(before, after *File) []StructureFieldDiff {
	b := collectFileFields(before)
	a := collectFileFields(after)
	paths := make([]string, 0, len(b)+len(a))
	seen := make(map[string]bool, len(b)+len(a))
	for path := range b {
		seen[path] = true
		paths = append(paths, path)
	}
	for path := range a {
		if !seen[path] {
			paths = append(paths, path)
		}
	}
	sort.Strings(paths)
	changes := make([]StructureFieldDiff, 0)
	for _, path := range paths {
		if suppressedSaveField(path) {
			continue
		}
		beforeField, beforeOK := b[path]
		afterField, afterOK := a[path]
		if beforeOK && afterOK && string(beforeField.Raw) == string(afterField.Raw) {
			continue
		}
		change := StructureFieldDiff{Path: path, BeforeExists: beforeOK, AfterExists: afterOK}
		if beforeOK {
			change.BeforeHex = hex.EncodeToString(beforeField.Raw)
			change.BeforeValue = beforeField.Value
			change.BeforeStart = beforeField.Start
			change.BeforeBytes = len(beforeField.Raw)
		}
		if afterOK {
			change.AfterHex = hex.EncodeToString(afterField.Raw)
			change.AfterValue = afterField.Value
			change.AfterStart = afterField.Start
			change.AfterBytes = len(afterField.Raw)
		}
		changes = append(changes, change)
	}
	return changes
}

func collectFileFields(file *File) map[string]fieldSnapshot {
	fields := make(map[string]fieldSnapshot)
	if file == nil {
		return fields
	}
	if file.headerRoot != nil {
		collectNodeFields(file.headerRoot, "FileHeader", fields)
	}
	if file.root != nil {
		for _, section := range file.root.Sections {
			collectNodeFields(section, section.Name, fields)
		}
	}
	return fields
}

func collectNodeFields(node *parsedNode, path string, out map[string]fieldSnapshot) {
	if node == nil {
		return
	}
	if len(node.Fields) == 0 && len(node.Elements) > 0 {
		for i, element := range node.Elements {
			collectNodeFields(element, fmt.Sprintf("%s[%d]", path, i), out)
		}
		return
	}
	if len(node.Fields) == 0 {
		if values, ok := node.Value.([]any); ok && len(values) > 0 && len(node.Raw) > 0 && len(node.Raw)%len(values) == 0 {
			out[path] = fieldSnapshot{
				Path:  path,
				Start: node.Start,
				Raw:   node.raw(),
				Value: len(values),
			}
			itemBytes := len(node.Raw) / len(values)
			for i, value := range values {
				start := node.Start + i*itemBytes
				out[fmt.Sprintf("%s[%d]", path, i)] = fieldSnapshot{
					Path:  fmt.Sprintf("%s[%d]", path, i),
					Start: start,
					Raw:   append([]byte(nil), node.Raw[i*itemBytes:(i+1)*itemBytes]...),
					Value: jsonValue(value),
				}
			}
			return
		}
		out[path] = fieldSnapshot{Path: path, Start: node.Start, Raw: node.raw(), Value: jsonValue(node.Value)}
		return
	}
	for _, field := range node.Fields {
		fieldPath := path + "." + field.Name
		if len(field.Elements) > 0 {
			out[fieldPath] = fieldSnapshot{
				Path:  fieldPath,
				Start: field.Start,
				Raw:   field.raw(),
				Value: len(field.Elements),
			}
			for i, element := range field.Elements {
				collectNodeFields(element, fmt.Sprintf("%s[%d]", fieldPath, i), out)
			}
			continue
		}
		collectNodeFields(field, fieldPath, out)
	}
}

func jsonValue(value any) any {
	if value == nil {
		return nil
	}
	// Values are parser primitives or []any. Round-tripping through JSON makes
	// the report stable without exposing parser implementation types.
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Sprint(value)
	}
	var out any
	if json.Unmarshal(data, &out) == nil {
		return out
	}
	return strings.TrimSpace(string(data))
}

func suppressedSaveField(path string) bool {
	return path == "FileHeader.timestamp_of_last_save" ||
		path == "DataHeader.next_unit_id_to_place" ||
		strings.HasSuffix(path, ".editor_camera_x") ||
		strings.HasSuffix(path, ".editor_camera_y")
}
