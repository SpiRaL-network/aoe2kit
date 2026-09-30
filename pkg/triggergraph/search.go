package triggergraph

import (
	"fmt"
	"strconv"
	"strings"
)

type SearchOptions struct {
	Grep         string `json:"grep,omitempty"`
	Effect       string `json:"effect,omitempty"`
	Condition    string `json:"condition,omitempty"`
	Player       *int   `json:"player,omitempty"`
	Variable     *int   `json:"variable,omitempty"`
	UnitConst    *int   `json:"unit_const,omitempty"`
	TriggerLimit int    `json:"trigger_limit,omitempty"`
	MatchLimit   int    `json:"match_limit,omitempty"`
}

type SearchReport struct {
	Verification   string        `json:"verification"`
	GraphSHA256    string        `json:"graph_sha256"`
	TriggerCount   int           `json:"trigger_count"`
	EffectCount    int           `json:"effect_count"`
	ConditionCount int           `json:"condition_count"`
	Query          SearchOptions `json:"query"`
	TriggersShown  int           `json:"triggers_shown"`
	TotalMatches   int           `json:"total_matches"`
	Matches        []SearchMatch `json:"matches,omitempty"`
	Warnings       []string      `json:"warnings,omitempty"`
}

type SearchMatch struct {
	TriggerIndex     int            `json:"trigger_index"`
	TriggerName      string         `json:"trigger_name,omitempty"`
	Description      string         `json:"description,omitempty"`
	ShortDescription string         `json:"short_description,omitempty"`
	Enabled          int            `json:"enabled,omitempty"`
	Looping          int            `json:"looping,omitempty"`
	EffectIndex      *int           `json:"effect_index,omitempty"`
	EffectType       int            `json:"effect_type,omitempty"`
	EffectTypeName   string         `json:"effect_type_name,omitempty"`
	ConditionIndex   *int           `json:"condition_index,omitempty"`
	ConditionType    int            `json:"condition_type,omitempty"`
	ConditionName    string         `json:"condition_name,omitempty"`
	Message          string         `json:"message,omitempty"`
	KnownFields      map[string]any `json:"known_fields,omitempty"`
	MatchedFields    map[string]any `json:"matched_fields,omitempty"`
}

func Search(graph *Graph, opts SearchOptions) (*SearchReport, error) {
	if graph == nil {
		return nil, fmt.Errorf("trigger graph is nil")
	}
	report := &SearchReport{
		Verification:   "structure_verified_embedded_trigger_graph_not_engine_runtime_verified",
		GraphSHA256:    graph.SHA256,
		TriggerCount:   graph.TriggerCount,
		EffectCount:    graph.EffectCount,
		ConditionCount: graph.ConditionCount,
		Query:          opts,
		Warnings:       graph.Warnings,
	}
	triggerLimit := opts.TriggerLimit
	if triggerLimit <= 0 || triggerLimit > len(graph.Triggers) {
		triggerLimit = len(graph.Triggers)
	}
	for triggerIndex := 0; triggerIndex < triggerLimit; triggerIndex++ {
		trigger := graph.Triggers[triggerIndex]
		matches := searchTrigger(triggerIndex, trigger, opts)
		for _, match := range matches {
			report.TotalMatches++
			if opts.MatchLimit <= 0 || len(report.Matches) < opts.MatchLimit {
				report.Matches = append(report.Matches, match)
			}
		}
	}
	report.TriggersShown = triggerLimit
	return report, nil
}

func hasSearchCriteria(opts SearchOptions) bool {
	return opts.Grep != "" || opts.Effect != "" || opts.Condition != "" || opts.Player != nil || opts.Variable != nil || opts.UnitConst != nil
}

func searchTrigger(triggerIndex int, trigger map[string]any, opts SearchOptions) []SearchMatch {
	var matches []SearchMatch
	if !hasSearchCriteria(opts) {
		return []SearchMatch{{
			TriggerIndex:     triggerIndex,
			TriggerName:      stringValue(trigger["name"]),
			Description:      stringValue(trigger["description"]),
			ShortDescription: stringValue(trigger["short_description"]),
			Enabled:          intValue(trigger["enabled"]),
			Looping:          intValue(trigger["looping"]),
		}}
	}
	grep := strings.ToLower(opts.Grep)
	triggerName, _ := trigger["name"].(string)
	description := stringValue(trigger["description"])
	shortDescription := stringValue(trigger["short_description"])
	baseMatched := map[string]any{}
	if grep != "" {
		addContainsMatch(baseMatched, "trigger_name", triggerName, grep)
		if description, _ := trigger["description"].(string); description != "" {
			addContainsMatch(baseMatched, "description", description, grep)
		}
		if short, _ := trigger["short_description"].(string); short != "" {
			addContainsMatch(baseMatched, "short_description", short, grep)
		}
	}
	for effectIndex, effect := range listOfMaps(trigger["effects"]) {
		if matchedFields, ok := matchSearchEffect(effect, opts, grep, baseMatched); ok {
			i := effectIndex
			matches = append(matches, SearchMatch{
				TriggerIndex:     triggerIndex,
				TriggerName:      triggerName,
				Description:      description,
				ShortDescription: shortDescription,
				Enabled:          intValue(trigger["enabled"]),
				Looping:          intValue(trigger["looping"]),
				EffectIndex:      &i,
				EffectType:       intValue(effect["type"]),
				EffectTypeName:   stringValue(effect["type_name"]),
				Message:          stringValue(effect["message"]),
				KnownFields:      knownFieldMap(effect),
				MatchedFields:    matchedFields,
			})
		}
	}
	for conditionIndex, condition := range listOfMaps(trigger["conditions"]) {
		if matchedFields, ok := matchSearchCondition(condition, opts, grep, baseMatched); ok {
			i := conditionIndex
			matches = append(matches, SearchMatch{
				TriggerIndex:     triggerIndex,
				TriggerName:      triggerName,
				Description:      description,
				ShortDescription: shortDescription,
				Enabled:          intValue(trigger["enabled"]),
				Looping:          intValue(trigger["looping"]),
				ConditionIndex:   &i,
				ConditionType:    intValue(condition["type"]),
				ConditionName:    stringValue(condition["type_name"]),
				KnownFields:      knownFieldMap(condition),
				MatchedFields:    matchedFields,
			})
		}
	}
	if len(matches) == 0 && len(baseMatched) > 0 && criteriaOnlyGrep(opts) {
		matches = append(matches, SearchMatch{
			TriggerIndex:     triggerIndex,
			TriggerName:      triggerName,
			Description:      description,
			ShortDescription: shortDescription,
			Enabled:          intValue(trigger["enabled"]),
			Looping:          intValue(trigger["looping"]),
			MatchedFields:    baseMatched,
		})
	}
	return matches
}

func criteriaOnlyGrep(opts SearchOptions) bool {
	return opts.Grep != "" && opts.Effect == "" && opts.Condition == "" && opts.Player == nil && opts.Variable == nil && opts.UnitConst == nil
}

func matchSearchEffect(effect map[string]any, opts SearchOptions, grep string, base map[string]any) (map[string]any, bool) {
	if opts.Condition != "" {
		return nil, false
	}
	typeID := intValue(effect["type"])
	typeName := stringValue(effect["type_name"])
	known := knownFieldMap(effect)
	matches := copyAnyMap(base)
	if opts.Effect != "" {
		if !idOrNameMatches(typeID, typeName, opts.Effect) {
			return nil, false
		}
		matches["effect_type"] = typeNameOrID(typeName, typeID)
	}
	if opts.Player != nil && !knownHasAnyInt(known, *opts.Player, "source_player", "target_player") {
		return nil, false
	}
	if opts.Player != nil {
		matches["player"] = *opts.Player
	}
	if opts.Variable != nil && !knownHasAnyInt(known, *opts.Variable, "variable", "variable2") {
		return nil, false
	}
	if opts.Variable != nil {
		matches["variable"] = *opts.Variable
	}
	if opts.UnitConst != nil && !knownHasAnyInt(known, *opts.UnitConst, "object_list_unit_id", "object_list_unit_id_2", "building_list") {
		return nil, false
	}
	if opts.UnitConst != nil {
		matches["unit_const"] = *opts.UnitConst
	}
	if grep != "" {
		before := len(matches)
		addContainsMatch(matches, "effect_type", typeName, grep)
		addContainsMatch(matches, "message", stringValue(effect["message"]), grep)
		for key, value := range known {
			addContainsMatch(matches, key, fmt.Sprint(value), grep)
		}
		if len(matches) == before {
			return nil, false
		}
	}
	return matches, true
}

func matchSearchCondition(condition map[string]any, opts SearchOptions, grep string, base map[string]any) (map[string]any, bool) {
	if opts.Effect != "" {
		return nil, false
	}
	typeID := intValue(condition["type"])
	typeName := stringValue(condition["type_name"])
	known := knownFieldMap(condition)
	matches := copyAnyMap(base)
	if opts.Condition != "" {
		if !idOrNameMatches(typeID, typeName, opts.Condition) {
			return nil, false
		}
		matches["condition_type"] = typeNameOrID(typeName, typeID)
	}
	if opts.Player != nil && !knownHasAnyInt(known, *opts.Player, "source_player", "target_player") {
		return nil, false
	}
	if opts.Player != nil {
		matches["player"] = *opts.Player
	}
	if opts.Variable != nil && !knownHasAnyInt(known, *opts.Variable, "variable", "variable2") {
		return nil, false
	}
	if opts.Variable != nil {
		matches["variable"] = *opts.Variable
	}
	if opts.UnitConst != nil && !knownHasAnyInt(known, *opts.UnitConst, "object_list", "unit_object", "next_object") {
		return nil, false
	}
	if opts.UnitConst != nil {
		matches["unit_const"] = *opts.UnitConst
	}
	if grep != "" {
		before := len(matches)
		addContainsMatch(matches, "condition_type", typeName, grep)
		addContainsMatch(matches, "xs_function", stringValue(condition["xs_function"]), grep)
		for key, value := range known {
			addContainsMatch(matches, key, fmt.Sprint(value), grep)
		}
		if len(matches) == before {
			return nil, false
		}
	}
	return matches, true
}

func knownFieldMap(row map[string]any) map[string]any {
	known, _ := row["known_fields"].(map[string]any)
	if known == nil {
		return nil
	}
	return known
}

func knownHasAnyInt(known map[string]any, want int, keys ...string) bool {
	for _, key := range keys {
		if intValue(known[key]) == want {
			return true
		}
	}
	return false
}

func addContainsMatch(matches map[string]any, field, value, grep string) {
	if value == "" || grep == "" {
		return
	}
	if strings.Contains(strings.ToLower(value), grep) {
		matches[field] = value
	}
}

func idOrNameMatches(id int, name, query string) bool {
	if query == "" {
		return true
	}
	if n, err := strconv.Atoi(query); err == nil {
		return id == n
	}
	name = strings.ToLower(name)
	query = strings.ToLower(query)
	if name == query {
		return true
	}
	for _, token := range strings.Split(name, "_") {
		if strings.HasPrefix(token, query) {
			return true
		}
	}
	return false
}

func typeNameOrID(name string, id int) any {
	if name != "" {
		return name
	}
	return id
}

func copyAnyMap(in map[string]any) map[string]any {
	out := map[string]any{}
	for key, value := range in {
		out[key] = value
	}
	return out
}

func stringValue(value any) string {
	if s, ok := value.(string); ok {
		return s
	}
	return ""
}

func intValue(value any) int {
	switch v := value.(type) {
	case int:
		return v
	case int8:
		return int(v)
	case int16:
		return int(v)
	case int32:
		return int(v)
	case int64:
		return int(v)
	case uint:
		return int(v)
	case uint8:
		return int(v)
	case uint16:
		return int(v)
	case uint32:
		return int(v)
	case uint64:
		return int(v)
	case float64:
		return int(v)
	case float32:
		return int(v)
	default:
		return 0
	}
}
