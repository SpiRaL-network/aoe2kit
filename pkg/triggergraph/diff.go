package triggergraph

import (
	"fmt"
	"reflect"
	"sort"

	"aoe2kit/pkg/datfacts"
)

type DiffOptions struct {
	Limit int `json:"limit,omitempty"`
}

type DiffCounts struct {
	Triggers   int `json:"triggers"`
	Effects    int `json:"effects"`
	Conditions int `json:"conditions"`
	Messages   int `json:"messages"`
}

type DiffSummary struct {
	Same        bool `json:"same"`
	Total       int  `json:"total_changes"`
	Shown       int  `json:"shown_changes"`
	Limit       int  `json:"limit,omitempty"`
	Added       int  `json:"added"`
	Removed     int  `json:"removed"`
	Modified    int  `json:"modified"`
	CountFields int  `json:"count_fields"`
}

type DiffReport struct {
	Before       string       `json:"before,omitempty"`
	After        string       `json:"after,omitempty"`
	Verification string       `json:"verification"`
	Same         bool         `json:"same"`
	BeforeSHA256 string       `json:"before_sha256,omitempty"`
	AfterSHA256  string       `json:"after_sha256,omitempty"`
	BeforeCounts DiffCounts   `json:"before_counts"`
	AfterCounts  DiffCounts   `json:"after_counts"`
	Summary      DiffSummary  `json:"summary"`
	Changes      []DiffChange `json:"changes,omitempty"`
	Warnings     []string     `json:"warnings,omitempty"`
}

type DiffChange struct {
	Kind         string `json:"kind"`
	Field        string `json:"field,omitempty"`
	TriggerIndex int    `json:"trigger_index,omitempty"`
	TriggerName  string `json:"trigger_name,omitempty"`
	Before       any    `json:"before,omitempty"`
	After        any    `json:"after,omitempty"`
	Detail       string `json:"detail,omitempty"`
}

func Diff(before, after *Graph, opts DiffOptions) *DiffReport {
	report := &DiffReport{
		Verification: "structure_verified_trigger_graph_not_engine_runtime_verified",
		BeforeCounts: countsFor(before),
		AfterCounts:  countsFor(after),
	}
	if before != nil {
		report.BeforeSHA256 = before.SHA256
		report.Warnings = appendWarnings(report.Warnings, "before", before.Warnings)
	}
	if after != nil {
		report.AfterSHA256 = after.SHA256
		report.Warnings = appendWarnings(report.Warnings, "after", after.Warnings)
	}
	add := func(change DiffChange) {
		report.Changes = append(report.Changes, change)
		switch change.Kind {
		case "trigger_added":
			report.Summary.Added++
		case "trigger_removed":
			report.Summary.Removed++
		case "trigger_modified":
			report.Summary.Modified++
		case "count_changed":
			report.Summary.CountFields++
		}
	}
	if before == nil || after == nil {
		add(DiffChange{Kind: "graph_unavailable", Detail: "one or both trigger graphs are unavailable"})
		return finalizeDiff(report, opts)
	}
	if before.TriggerCount != after.TriggerCount {
		add(DiffChange{Kind: "count_changed", Field: "trigger_count", Before: before.TriggerCount, After: after.TriggerCount})
	}
	if before.EffectCount != after.EffectCount {
		add(DiffChange{Kind: "count_changed", Field: "effect_count", Before: before.EffectCount, After: after.EffectCount})
	}
	if before.ConditionCount != after.ConditionCount {
		add(DiffChange{Kind: "count_changed", Field: "condition_count", Before: before.ConditionCount, After: after.ConditionCount})
	}
	if before.MessageCount != after.MessageCount {
		add(DiffChange{Kind: "count_changed", Field: "message_count", Before: before.MessageCount, After: after.MessageCount})
	}
	maxTriggers := len(before.Triggers)
	if len(after.Triggers) > maxTriggers {
		maxTriggers = len(after.Triggers)
	}
	for i := 0; i < maxTriggers; i++ {
		if i >= len(before.Triggers) {
			add(DiffChange{Kind: "trigger_added", TriggerIndex: i, TriggerName: triggerName(after.Triggers[i]), After: triggerDigest(after.Triggers[i])})
			continue
		}
		if i >= len(after.Triggers) {
			add(DiffChange{Kind: "trigger_removed", TriggerIndex: i, TriggerName: triggerName(before.Triggers[i]), Before: triggerDigest(before.Triggers[i])})
			continue
		}
		for _, change := range diffTrigger(i, before.Triggers[i], after.Triggers[i]) {
			add(change)
		}
	}
	return finalizeDiff(report, opts)
}

func finalizeDiff(report *DiffReport, opts DiffOptions) *DiffReport {
	report.Summary.Total = len(report.Changes)
	report.Same = len(report.Changes) == 0 && report.BeforeSHA256 == report.AfterSHA256
	report.Summary.Same = report.Same
	report.Summary.Limit = opts.Limit
	if opts.Limit > 0 && len(report.Changes) > opts.Limit {
		report.Changes = report.Changes[:opts.Limit]
	}
	report.Summary.Shown = len(report.Changes)
	return report
}

func countsFor(graph *Graph) DiffCounts {
	if graph == nil {
		return DiffCounts{}
	}
	return DiffCounts{
		Triggers:   graph.TriggerCount,
		Effects:    graph.EffectCount,
		Conditions: graph.ConditionCount,
		Messages:   graph.MessageCount,
	}
}

func appendWarnings(dst []string, prefix string, warnings []string) []string {
	for _, warning := range warnings {
		dst = append(dst, prefix+": "+warning)
	}
	return dst
}

func diffTrigger(index int, before, after map[string]any) []DiffChange {
	var changes []DiffChange
	name := triggerName(before)
	if name == "" {
		name = triggerName(after)
	}
	for _, field := range triggerHeaderDiffFields {
		b, bOK := comparableValue(before[field])
		a, aOK := comparableValue(after[field])
		if bOK != aOK || !reflect.DeepEqual(b, a) {
			changes = append(changes, DiffChange{Kind: "trigger_modified", Field: field, TriggerIndex: index, TriggerName: name, Before: b, After: a})
		}
	}
	beforeEffects := listOfMaps(before["effects"])
	afterEffects := listOfMaps(after["effects"])
	changes = append(changes, diffChildList(index, name, "effect", beforeEffects, afterEffects)...)
	beforeConditions := listOfMaps(before["conditions"])
	afterConditions := listOfMaps(after["conditions"])
	changes = append(changes, diffChildList(index, name, "condition", beforeConditions, afterConditions)...)
	return changes
}

var triggerHeaderDiffFields = []string{
	"name",
	"description",
	"short_description",
	"enabled",
	"looping",
	"execute_on_load",
	"description_stid",
	"display_as_objective",
	"description_order",
	"make_header",
	"short_description_stid",
	"display_on_screen",
	"mute_objectives",
	"effect_order",
	"condition_order",
}

func diffChildList(triggerIndex int, triggerName, child string, before, after []map[string]any) []DiffChange {
	var changes []DiffChange
	countField := child + "_count"
	if len(before) != len(after) {
		changes = append(changes, DiffChange{
			Kind:         "trigger_modified",
			Field:        countField,
			TriggerIndex: triggerIndex,
			TriggerName:  triggerName,
			Before:       len(before),
			After:        len(after),
		})
	}
	maxChildren := len(before)
	if len(after) > maxChildren {
		maxChildren = len(after)
	}
	for i := 0; i < maxChildren; i++ {
		field := fmt.Sprintf("%s_%d", child, i)
		if i >= len(before) {
			changes = append(changes, DiffChange{Kind: "trigger_modified", Field: field, TriggerIndex: triggerIndex, TriggerName: triggerName, After: childDigest(after[i]), Detail: child + " added"})
			continue
		}
		if i >= len(after) {
			changes = append(changes, DiffChange{Kind: "trigger_modified", Field: field, TriggerIndex: triggerIndex, TriggerName: triggerName, Before: childDigest(before[i]), Detail: child + " removed"})
			continue
		}
		b := childDigest(before[i])
		a := childDigest(after[i])
		if !reflect.DeepEqual(b, a) {
			fieldChanges := diffChildFields(triggerIndex, triggerName, child, i, before[i], after[i])
			if len(fieldChanges) > 0 {
				changes = append(changes, fieldChanges...)
			} else {
				changes = append(changes, DiffChange{Kind: "trigger_modified", Field: field, TriggerIndex: triggerIndex, TriggerName: triggerName, Before: b, After: a, Detail: child + " changed"})
			}
		}
	}
	return changes
}

func diffChildFields(triggerIndex int, triggerName, child string, childIndex int, before, after map[string]any) []DiffChange {
	var changes []DiffChange
	base := fmt.Sprintf("%s_%d", child, childIndex)
	add := func(field string, b, a any) {
		changes = append(changes, DiffChange{
			Kind:         "trigger_modified",
			Field:        base + "." + field,
			TriggerIndex: triggerIndex,
			TriggerName:  triggerName,
			Before:       b,
			After:        a,
			Detail:       child + " field changed",
		})
	}
	for _, field := range []string{"type", "type_name", "message", "xs_function", "selected_object_ids"} {
		b, bOK := comparableValue(before[field])
		a, aOK := comparableValue(after[field])
		if bOK != aOK || !reflect.DeepEqual(b, a) {
			add(field, b, a)
		}
	}
	beforeFields, beforeOK := childFieldValues(child, before)
	afterFields, afterOK := childFieldValues(child, after)
	if beforeOK || afterOK {
		names := childFieldNames(child)
		maxFields := len(beforeFields)
		if len(afterFields) > maxFields {
			maxFields = len(afterFields)
		}
		for i := 0; i < maxFields; i++ {
			b, bOK := fieldValueAt(beforeFields, i)
			a, aOK := fieldValueAt(afterFields, i)
			if bOK != aOK || b != a {
				fieldName := childSlotName(names, i)
				add(fieldName, namedSlotValue(fieldName, b, bOK), namedSlotValue(fieldName, a, aOK))
			}
		}
	}
	return changes
}

func childFieldValues(child string, m map[string]any) ([]int, bool) {
	if child == "effect" {
		return intList(m["fields_prefix"])
	}
	return intList(m["fields"])
}

func childFieldNames(child string) []string {
	if child == "effect" {
		return effectFieldNames
	}
	return conditionFieldNames
}

func fieldValueAt(values []int, index int) (int, bool) {
	if index < 0 || index >= len(values) {
		return 0, false
	}
	return values[index], true
}

func childSlotName(names []string, index int) string {
	if index >= 0 && index < len(names) && names[index] != "" {
		return names[index]
	}
	return fmt.Sprintf("slot_%d", index)
}

func namedSlotValue(field string, value int, ok bool) any {
	if !ok {
		return nil
	}
	name := triggerDiffEnumName(field, value)
	if name == "" {
		return value
	}
	return map[string]any{"id": value, "name": name}
}

func triggerDiffEnumName(field string, value int) string {
	if value < 0 {
		return ""
	}
	switch field {
	case "object_group", "object_group2":
		return datfacts.ScenarioUnitClassName(value)
	case "object_type", "object_type2":
		return datfacts.ScenarioObjectTypeName(value)
	case "unit_ai_action":
		return datfacts.ScenarioUnitAIActionName(value)
	default:
		return ""
	}
}

func triggerName(trigger map[string]any) string {
	name, _ := trigger["name"].(string)
	return name
}

func triggerDigest(trigger map[string]any) map[string]any {
	out := map[string]any{}
	for _, field := range triggerHeaderDiffFields {
		if v, ok := comparableValue(trigger[field]); ok {
			out[field] = v
		}
	}
	out["effects"] = len(listOfMaps(trigger["effects"]))
	out["conditions"] = len(listOfMaps(trigger["conditions"]))
	return sortedMap(out)
}

func childDigest(child map[string]any) map[string]any {
	out := map[string]any{}
	for _, field := range []string{"type", "message", "raw_sha256", "xs_function"} {
		if v, ok := comparableValue(child[field]); ok {
			out[field] = v
		}
	}
	if v, ok := comparableValue(child["type_name"]); ok {
		out["type_name"] = v
	}
	if fields, ok := intList(child["fields_prefix"]); ok {
		out["fields_prefix"] = fields
	}
	if fields, ok := intList(child["fields"]); ok {
		out["fields"] = fields
	}
	if fields, ok := intList(child["selected_object_ids"]); ok {
		out["selected_object_ids"] = fields
	}
	if known, ok := child["known_fields"].(map[string]any); ok {
		out["known_fields"] = sortedMap(known)
	}
	return sortedMap(out)
}

func comparableValue(v any) (any, bool) {
	switch value := v.(type) {
	case nil:
		return nil, false
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, string, bool:
		return value, true
	case float64:
		if value == float64(int64(value)) {
			return int64(value), true
		}
		return value, true
	case float32:
		return float64(value), true
	default:
		return value, true
	}
}

func listOfMaps(v any) []map[string]any {
	switch list := v.(type) {
	case []map[string]any:
		return list
	case []any:
		out := make([]map[string]any, 0, len(list))
		for _, item := range list {
			if m, ok := item.(map[string]any); ok {
				out = append(out, m)
			}
		}
		return out
	default:
		return nil
	}
}

func intList(v any) ([]int, bool) {
	switch list := v.(type) {
	case []int:
		return append([]int(nil), list...), true
	case []any:
		out := make([]int, 0, len(list))
		for _, item := range list {
			switch n := item.(type) {
			case int:
				out = append(out, n)
			case int32:
				out = append(out, int(n))
			case int64:
				out = append(out, int(n))
			case float64:
				out = append(out, int(n))
			default:
				return nil, false
			}
		}
		return out, true
	default:
		return nil, false
	}
}

func sortedMap(in map[string]any) map[string]any {
	keys := make([]string, 0, len(in))
	for key := range in {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make(map[string]any, len(in))
	for _, key := range keys {
		out[key] = in[key]
	}
	return out
}
