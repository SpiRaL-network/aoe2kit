package scenario

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
)

type EffectWhereOptions struct {
	Query            string `json:"query,omitempty"`
	Limit            int    `json:"limit,omitempty"`
	TriggerIndex     *int   `json:"trigger_index,omitempty"`
	IncludeRawFields bool   `json:"include_raw_fields,omitempty"`
	MaxInflatedBytes int    `json:"max_inflated_bytes,omitempty"`
}

type EffectWhereReport struct {
	Path            string          `json:"path,omitempty"`
	Version         string          `json:"version"`
	Verification    string          `json:"verification"`
	HeaderBytes     int             `json:"header_bytes"`
	CompressedBytes int             `json:"compressed_body_bytes"`
	InflatedBytes   int             `json:"inflated_body_bytes"`
	Query           string          `json:"query"`
	Limit           int             `json:"limit"`
	TotalMatches    int             `json:"total_matches"`
	Matches         []EffectSummary `json:"matches,omitempty"`
}

type EffectFieldDelta struct {
	Before any `json:"before,omitempty"`
	After  any `json:"after,omitempty"`
}

type EffectDiffItem struct {
	Ordinal       int                         `json:"ordinal"`
	BeforeTrigger int                         `json:"before_trigger"`
	AfterTrigger  int                         `json:"after_trigger"`
	BeforeEffect  int                         `json:"before_effect"`
	AfterEffect   int                         `json:"after_effect"`
	Fields        map[string]EffectFieldDelta `json:"fields,omitempty"`
	Before        *EffectSummary              `json:"before,omitempty"`
	After         *EffectSummary              `json:"after,omitempty"`
}

type EffectDiffReport struct {
	Before        string           `json:"before"`
	After         string           `json:"after"`
	Verification  string           `json:"verification"`
	Query         string           `json:"query"`
	TriggerIndex  *int             `json:"trigger_index,omitempty"`
	BeforeMatches int              `json:"before_matches"`
	AfterMatches  int              `json:"after_matches"`
	Compared      int              `json:"compared"`
	Modified      int              `json:"modified"`
	Added         int              `json:"added"`
	Removed       int              `json:"removed"`
	Changes       []EffectDiffItem `json:"changes,omitempty"`
}

type TriggerSearchOptions struct {
	Grep             string `json:"grep,omitempty"`
	EffectQuery      string `json:"effect_query,omitempty"`
	Message          string `json:"message,omitempty"`
	ConditionQuery   string `json:"condition_query,omitempty"`
	UnitRef          *int   `json:"unit_ref,omitempty"`
	UnitType         *int   `json:"unit_type,omitempty"`
	Player           *int   `json:"player,omitempty"`
	Variable         *int   `json:"variable,omitempty"`
	Area             []int  `json:"area,omitempty"`
	Limit            int    `json:"limit,omitempty"`
	MaxInflatedBytes int    `json:"max_inflated_bytes,omitempty"`
}

type TriggerSearchReport struct {
	Path            string               `json:"path,omitempty"`
	Version         string               `json:"version"`
	Verification    string               `json:"verification"`
	HeaderBytes     int                  `json:"header_bytes"`
	CompressedBytes int                  `json:"compressed_body_bytes"`
	InflatedBytes   int                  `json:"inflated_body_bytes"`
	Limit           int                  `json:"limit"`
	Query           TriggerSearchOptions `json:"query"`
	TotalMatches    int                  `json:"total_matches"`
	Matches         []TriggerSearchMatch `json:"matches,omitempty"`
}

type TriggerSearchMatch struct {
	TriggerIndex     int               `json:"trigger_index"`
	TriggerName      string            `json:"trigger_name,omitempty"`
	Description      string            `json:"description,omitempty"`
	ShortDescription string            `json:"short_description,omitempty"`
	Enabled          uint32            `json:"enabled"`
	Looping          int8              `json:"looping"`
	EffectCount      int               `json:"effect_count"`
	ConditionCount   int               `json:"condition_count"`
	EffectIndex      *int              `json:"effect_index,omitempty"`
	EffectType       int               `json:"effect_type,omitempty"`
	EffectTypeName   string            `json:"effect_type_name,omitempty"`
	ConditionIndex   *int              `json:"condition_index,omitempty"`
	ConditionType    int               `json:"condition_type,omitempty"`
	ConditionName    string            `json:"condition_name,omitempty"`
	MatchedFields    map[string]string `json:"matched_fields"`
}

type ScenarioGlossaryOptions struct {
	Limit            int `json:"limit,omitempty"`
	MaxInflatedBytes int `json:"max_inflated_bytes,omitempty"`
}

type ScenarioGlossaryReport struct {
	Path            string             `json:"path,omitempty"`
	Version         string             `json:"version"`
	Verification    string             `json:"verification"`
	HeaderBytes     int                `json:"header_bytes"`
	CompressedBytes int                `json:"compressed_body_bytes"`
	InflatedBytes   int                `json:"inflated_body_bytes"`
	Limit           int                `json:"limit"`
	TriggerCount    int                `json:"trigger_count"`
	EffectCount     int                `json:"effect_count"`
	VariableCount   int                `json:"variable_count"`
	EffectTypes     []EffectTypeBucket `json:"effect_types"`
	TriggerPrefixes []StringCount      `json:"trigger_prefixes,omitempty"`
	Variables       []VariableEntry    `json:"variables,omitempty"`
	XSCalls         []StringCount      `json:"xs_calls,omitempty"`
	TextSamples     []StringCount      `json:"text_samples,omitempty"`
	UnitIDs         []IntCount         `json:"unit_ids,omitempty"`
	TechnologyIDs   []IntCount         `json:"technology_ids,omitempty"`
	AttributeIDs    []IntCount         `json:"attribute_ids,omitempty"`
}

func EffectsWhereFile(path string, opts EffectWhereOptions) (EffectWhereReport, error) {
	return effectsWhereFile(path, opts, false)
}

func effectsWhereFile(path string, opts EffectWhereOptions, unlimited bool) (EffectWhereReport, error) {
	if strings.TrimSpace(opts.Query) == "" {
		return EffectWhereReport{}, fmt.Errorf("effect --where query is required")
	}
	if opts.Limit <= 0 && !unlimited {
		opts.Limit = 25
	}
	report := EffectWhereReport{
		Path:         path,
		Verification: "structure_verified_not_engine_verified",
		Query:        opts.Query,
		Limit:        opts.Limit,
	}
	header, err := scanScenarioFile(path, ScenarioScanOptions{MaxInflatedBytes: opts.MaxInflatedBytes}, triggerScanCallbacks{
		onEffect: func(meta ScenarioScanMeta, effectIndex int, effect *parsedNode) error {
			effectType, _ := effect.intValue("effect_type")
			if !effectTypeMatchesQuery(effectType, opts.Query) {
				return nil
			}
			if opts.TriggerIndex != nil && meta.Index != *opts.TriggerIndex {
				return nil
			}
			report.TotalMatches++
			if opts.Limit > 0 && len(report.Matches) >= opts.Limit {
				return nil
			}
			summary := summarizeEffect(effect, EffectsOptions{IncludeRawFields: opts.IncludeRawFields})
			summary.TriggerIndex = meta.Index
			summary.TriggerName = meta.Name
			summary.EffectIndex = effectIndex
			report.Matches = append(report.Matches, summary)
			return nil
		},
	})
	if err != nil {
		return EffectWhereReport{}, err
	}
	report.Version = header.version
	report.HeaderBytes = header.headerBytes
	report.CompressedBytes = header.compressedBytes
	report.InflatedBytes = header.inflatedBytes
	return report, nil
}

// DiffEffectsFile compares matching effects in occurrence order. It is a
// control-oriented diagnostic: trigger/effect indexes are reported, but field
// drift is the primary result because two authors may organize triggers
// differently while expressing the same mechanic.
func DiffEffectsFile(beforePath, afterPath string, opts EffectWhereOptions) (EffectDiffReport, error) {
	// A control comparison should be complete by default. An explicit positive
	// limit remains useful for a bounded probe on very large scenarios.
	unlimited := opts.Limit <= 0
	before, err := effectsWhereFile(beforePath, opts, unlimited)
	if err != nil {
		return EffectDiffReport{}, err
	}
	after, err := effectsWhereFile(afterPath, opts, unlimited)
	if err != nil {
		return EffectDiffReport{}, err
	}
	report := EffectDiffReport{
		Before: beforePath, After: afterPath, Verification: "structure_verified_not_engine_verified",
		Query: opts.Query, TriggerIndex: opts.TriggerIndex,
		BeforeMatches: before.TotalMatches, AfterMatches: after.TotalMatches,
	}
	limit := len(before.Matches)
	if len(after.Matches) < limit {
		limit = len(after.Matches)
	}
	report.Compared = limit
	for i := 0; i < limit; i++ {
		left, right := before.Matches[i], after.Matches[i]
		fields := effectFieldDeltas(left, right)
		if len(fields) == 0 {
			continue
		}
		report.Modified++
		report.Changes = append(report.Changes, EffectDiffItem{
			Ordinal: i, BeforeTrigger: left.TriggerIndex, AfterTrigger: right.TriggerIndex,
			BeforeEffect: left.EffectIndex, AfterEffect: right.EffectIndex,
			Fields: fields, Before: &left, After: &right,
		})
	}
	if after.TotalMatches > before.TotalMatches {
		report.Added = after.TotalMatches - before.TotalMatches
	} else if before.TotalMatches > after.TotalMatches {
		report.Removed = before.TotalMatches - after.TotalMatches
	}
	return report, nil
}

func effectFieldDeltas(before, after EffectSummary) map[string]EffectFieldDelta {
	fields := map[string]EffectFieldDelta{}
	keys := map[string]bool{}
	for key := range before.KnownFields {
		keys[key] = true
	}
	for key := range after.KnownFields {
		keys[key] = true
	}
	for key := range keys {
		left, leftOK := before.KnownFields[key]
		right, rightOK := after.KnownFields[key]
		if !leftOK {
			left = nil
		}
		if !rightOK {
			right = nil
		}
		if !reflect.DeepEqual(left, right) {
			fields[key] = EffectFieldDelta{Before: left, After: right}
		}
	}
	if !reflect.DeepEqual(before.RawFields, after.RawFields) {
		fields["raw_fields"] = EffectFieldDelta{Before: before.RawFields, After: after.RawFields}
	}
	return fields
}

func TriggerSearchFile(path string, opts TriggerSearchOptions) (TriggerSearchReport, error) {
	if opts.Limit <= 0 {
		opts.Limit = 50
	}
	file, err := OpenWithOptions(path, ParseOptions{MaxInflatedBytes: opts.MaxInflatedBytes})
	if err != nil {
		return TriggerSearchReport{}, err
	}
	report := TriggerSearchReport{
		Path:            path,
		Version:         file.Version,
		Verification:    "structure_verified_not_engine_verified",
		HeaderBytes:     file.HeaderBytes,
		CompressedBytes: file.CompressedBytes,
		InflatedBytes:   file.InflatedBytes,
		Limit:           opts.Limit,
		Query:           opts,
	}
	add := func(match TriggerSearchMatch) {
		report.TotalMatches++
		if len(report.Matches) < opts.Limit {
			report.Matches = append(report.Matches, match)
		}
	}
	for _, row := range collectTriggerRows(file) {
		for _, match := range row.searchMatches(opts) {
			add(match)
		}
	}
	return report, nil
}

func hasTriggerSearchCriteria(opts TriggerSearchOptions) bool {
	return strings.TrimSpace(opts.Grep) != "" ||
		strings.TrimSpace(opts.EffectQuery) != "" ||
		strings.TrimSpace(opts.ConditionQuery) != "" ||
		strings.TrimSpace(opts.Message) != "" ||
		opts.UnitRef != nil ||
		opts.UnitType != nil ||
		opts.Player != nil ||
		opts.Variable != nil ||
		len(opts.Area) == 4
}

func ScenarioGlossaryFile(path string, opts ScenarioGlossaryOptions) (ScenarioGlossaryReport, error) {
	if opts.Limit <= 0 {
		opts.Limit = 25
	}
	report := ScenarioGlossaryReport{
		Path:         path,
		Verification: "structure_verified_not_engine_verified",
		Limit:        opts.Limit,
	}
	effectBuckets := map[int]*EffectTypeBucket{}
	prefixCounts := map[string]int{}
	xsCounts := map[string]int{}
	textCounts := map[string]int{}
	unitCounts := map[int]int{}
	techCounts := map[int]int{}
	attrCounts := map[int]int{}
	variables := []VariableEntry{}
	header, err := scanScenarioFile(path, ScenarioScanOptions{MaxInflatedBytes: opts.MaxInflatedBytes}, triggerScanCallbacks{
		onMeta: func(meta ScenarioScanMeta) {
			report.TriggerCount++
			if prefix := triggerPrefix(meta.Name); prefix != "" {
				prefixCounts[prefix]++
			}
		},
		onVariable: func(id int, name string) {
			report.VariableCount++
			if name != "" {
				variables = append(variables, VariableEntry{ID: id, Name: name})
			}
		},
		onEffect: func(meta ScenarioScanMeta, effectIndex int, effect *parsedNode) error {
			summary := summarizeEffect(effect, EffectsOptions{})
			bucket := effectBuckets[summary.Type]
			if bucket == nil {
				bucket = &EffectTypeBucket{Type: summary.Type, TypeName: summary.TypeName}
				effectBuckets[summary.Type] = bucket
			}
			bucket.Count++
			report.EffectCount++
			if summary.Text != "" && isUserFacingTextEffect(summary.Type) {
				textCounts[summary.Text]++
			}
			if summary.Type == 55 {
				for _, call := range extractXSCallNames(summary.Text) {
					xsCounts[call]++
				}
			}
			if summary.UnitConst != 0 && isUnitEffect(summary.Type) {
				unitCounts[summary.UnitConst]++
			}
			if summary.Technology != 0 && isTechnologyEffect(summary.Type) {
				techCounts[summary.Technology]++
			}
			if summary.ObjectAttribute != 0 && isAttributeEffect(summary.Type) {
				attrCounts[summary.ObjectAttribute]++
			}
			return nil
		},
	})
	if err != nil {
		return ScenarioGlossaryReport{}, err
	}
	report.Version = header.version
	report.HeaderBytes = header.headerBytes
	report.CompressedBytes = header.compressedBytes
	report.InflatedBytes = header.inflatedBytes
	report.EffectTypes = sortedEffectBuckets(effectBuckets)
	report.TriggerPrefixes = limitStringCounts(sortedStringCounts(prefixCounts), opts.Limit)
	sort.Slice(variables, func(i, j int) bool { return variables[i].ID < variables[j].ID })
	if len(variables) > opts.Limit {
		variables = variables[:opts.Limit]
	}
	report.Variables = variables
	report.XSCalls = limitStringCounts(sortedStringCounts(xsCounts), opts.Limit)
	report.TextSamples = limitStringCounts(sortedStringCounts(textCounts), opts.Limit)
	report.UnitIDs = limitIntCounts(sortedIntCounts(unitCounts), opts.Limit)
	report.TechnologyIDs = limitIntCounts(sortedIntCounts(techCounts), opts.Limit)
	report.AttributeIDs = limitIntCounts(sortedIntCounts(attrCounts), opts.Limit)
	return report, nil
}

func addContains(fields map[string]string, name, value, query string) {
	if value == "" || query == "" {
		return
	}
	if strings.Contains(strings.ToLower(value), query) {
		fields[name] = value
	}
}

func triggerPrefix(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	for _, sep := range []string{":", "-", "_", "#", "["} {
		if idx := strings.Index(name, sep); idx > 0 {
			return strings.TrimSpace(name[:idx])
		}
	}
	fields := strings.Fields(name)
	if len(fields) == 0 {
		return ""
	}
	return fields[0]
}

func isUserFacingTextEffect(effectType int) bool {
	switch effectType {
	case 3, 20, 26, 44, 45, 48, 59, 60, 65, 66, 88, 90:
		return true
	default:
		return false
	}
}

func isUnitEffect(effectType int) bool {
	switch effectType {
	case 11, 12, 14, 15, 17, 18, 19, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 42, 43, 49, 58, 60, 61, 62, 70, 71, 73, 74, 75, 77, 78, 88, 98, 99, 105, 106, 107, 108:
		return true
	default:
		return false
	}
}

func isTechnologyEffect(effectType int) bool {
	switch effectType {
	case 2, 39, 47, 63, 64, 65, 66, 67, 68, 76, 84, 85, 103:
		return true
	default:
		return false
	}
}

func isAttributeEffect(effectType int) bool {
	switch effectType {
	case 51, 79, 87, 104, 105, 106:
		return true
	default:
		return false
	}
}

func sortedEffectBuckets(buckets map[int]*EffectTypeBucket) []EffectTypeBucket {
	keys := make([]int, 0, len(buckets))
	for key := range buckets {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		if buckets[keys[i]].Count == buckets[keys[j]].Count {
			return keys[i] < keys[j]
		}
		return buckets[keys[i]].Count > buckets[keys[j]].Count
	})
	out := make([]EffectTypeBucket, 0, len(keys))
	for _, key := range keys {
		out = append(out, *buckets[key])
	}
	return out
}

func limitIntCounts(values []IntCount, limit int) []IntCount {
	if limit > 0 && len(values) > limit {
		return values[:limit]
	}
	return values
}

func limitStringCounts(values []StringCount, limit int) []StringCount {
	if limit > 0 && len(values) > limit {
		return values[:limit]
	}
	return values
}
