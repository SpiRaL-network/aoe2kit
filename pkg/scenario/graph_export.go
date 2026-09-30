package scenario

import "errors"

// ConditionTypeName maps trigger condition type ids to names (DE editor order).
func ConditionTypeName(t int) string {
	if n, ok := conditionTypeNames[t]; ok {
		return n
	}
	return "unknown"
}

func ConditionTypeForName(name string) (int, bool) {
	for id, candidate := range conditionTypeNames {
		if candidate == name {
			return id, true
		}
	}
	return 0, false
}

var conditionTypeNames = map[int]string{
	0:  "none",
	1:  "bring_object_to_area",
	2:  "bring_object_to_object",
	3:  "own_objects",
	4:  "own_fewer_objects",
	5:  "objects_in_area",
	6:  "destroy_object",
	7:  "capture_object",
	8:  "accumulate_attribute",
	9:  "research_technology",
	10: "timer",
	11: "object_selected",
	12: "ai_signal",
	13: "player_defeated",
	14: "object_has_target",
	15: "object_visible",
	16: "object_not_visible",
	17: "researching_technology",
	18: "units_garrisoned",
	19: "difficulty_level",
	20: "chance",
	21: "technology_state",
	22: "variable_value",
	23: "object_hp",
	24: "diplomacy_state",
	25: "script_call",
	26: "object_selected_multiplayer",
	27: "object_visible_multiplayer",
	28: "object_has_action",
	29: "or",
	30: "building_is_trading",
}

// TriggerConditionDump is a full per-trigger view including decoded conditions,
// which TriggerInfo summarizes away.
type TriggerConditionDump struct {
	Index      int              `json:"index"`
	Name       string           `json:"name"`
	Enabled    uint32           `json:"enabled"`
	Looping    int8             `json:"looping"`
	Conditions []map[string]any `json:"conditions"`
	Effects    []EffectSummary  `json:"effects"`
}

type ConditionSummary struct {
	TriggerIndex   int            `json:"trigger_index,omitempty"`
	TriggerName    string         `json:"trigger_name,omitempty"`
	ConditionIndex int            `json:"condition_index,omitempty"`
	Type           int            `json:"type"`
	TypeName       string         `json:"type_name"`
	KnownFields    map[string]any `json:"known_fields,omitempty"`
}

type TriggerConditionsOptions struct {
	Limit            int `json:"limit,omitempty"`
	MaxInflatedBytes int `json:"max_inflated_bytes,omitempty"`
}

type TriggerConditionsReport struct {
	Path            string                 `json:"path,omitempty"`
	Version         string                 `json:"version"`
	Verification    string                 `json:"verification"`
	HeaderBytes     int                    `json:"header_bytes"`
	CompressedBytes int                    `json:"compressed_body_bytes"`
	InflatedBytes   int                    `json:"inflated_body_bytes"`
	Limit           int                    `json:"limit,omitempty"`
	TotalTriggers   int                    `json:"total_triggers"`
	Triggers        []TriggerConditionDump `json:"triggers,omitempty"`
}

var conditionFieldNames = []string{
	"quantity", "attribute", "unit_object", "next_object", "object_list",
	"source_player", "technology", "timer", "trigger_id",
	"area_x1", "area_y1", "area_x2", "area_y2",
	"object_group", "object_type", "ai_signal", "inverted",
	"variable", "comparison", "target_player", "unit_ai_action",
	"object_state", "timer_id", "victory_timer_type", "decision_id", "decision_option",
}

func TriggerConditionsFile(path string, opts TriggerConditionsOptions) (TriggerConditionsReport, error) {
	file, err := OpenWithOptions(path, ParseOptions{MaxInflatedBytes: opts.MaxInflatedBytes})
	if err != nil {
		return TriggerConditionsReport{}, err
	}
	triggers, err := file.DumpTriggersWithConditions()
	if err != nil {
		return TriggerConditionsReport{}, err
	}
	total := len(triggers)
	if opts.Limit > 0 && len(triggers) > opts.Limit {
		triggers = triggers[:opts.Limit]
	}
	return TriggerConditionsReport{
		Path:            path,
		Version:         file.Version,
		Verification:    "structure_verified_not_engine_verified",
		HeaderBytes:     file.HeaderBytes,
		CompressedBytes: file.CompressedBytes,
		InflatedBytes:   file.InflatedBytes,
		Limit:           opts.Limit,
		TotalTriggers:   total,
		Triggers:        triggers,
	}, nil
}

// DumpTriggersWithConditions walks trigger_data and decodes every condition's
// known fields alongside the existing effect summaries.
func (f *File) DumpTriggersWithConditions() ([]TriggerConditionDump, error) {
	if f.root == nil {
		return nil, errors.New("scenario body not parsed")
	}
	section := f.root.section("Triggers")
	if section == nil {
		return nil, errors.New("missing Triggers section")
	}
	triggerNodes := section.list("trigger_data")
	out := make([]TriggerConditionDump, 0, len(triggerNodes))
	for i, trigger := range triggerNodes {
		name, _ := trigger.stringValue("trigger_name")
		enabled, _ := trigger.uint32Value("enabled")
		looping, _ := trigger.int8Value("looping")
		dump := TriggerConditionDump{Index: i, Name: name, Enabled: enabled, Looping: looping}
		for _, cond := range trigger.list("condition_data") {
			dump.Conditions = append(dump.Conditions, summarizeCondition(cond))
		}
		for j, effect := range trigger.list("effect_data") {
			summary := summarizeEffect(effect, EffectsOptions{})
			if op, ok := effect.intValue("operation"); ok && op != -1 {
				if summary.KnownFields == nil {
					summary.KnownFields = map[string]any{}
				}
				summary.KnownFields["operation"] = op
			}
			summary.TriggerIndex = i
			summary.TriggerName = name
			summary.EffectIndex = j
			dump.Effects = append(dump.Effects, summary)
		}
		out = append(out, dump)
	}
	return out, nil
}

func summarizeCondition(cond *parsedNode) map[string]any {
	condType, _ := cond.intValue("condition_type")
	entry := map[string]any{
		"type":      condType,
		"type_name": ConditionTypeName(condType),
	}
	for _, field := range conditionFieldNames {
		if v, ok := cond.intValue(field); ok && v != -1 {
			entry[field] = v
		}
	}
	if xsFunction, ok := cond.stringValue("xs_function"); ok && xsFunction != "" {
		entry["xs_function"] = xsFunction
	}
	return entry
}

func summarizeConditionData(cond *parsedNode) ConditionSummary {
	condType, _ := cond.intValue("condition_type")
	summary := ConditionSummary{
		Type:     condType,
		TypeName: ConditionTypeName(condType),
	}
	for _, field := range conditionFieldNames {
		if v, ok := cond.intValue(field); ok && v != -1 {
			if summary.KnownFields == nil {
				summary.KnownFields = map[string]any{}
			}
			summary.KnownFields[field] = v
		}
	}
	if xsFunction, ok := cond.stringValue("xs_function"); ok && xsFunction != "" {
		if summary.KnownFields == nil {
			summary.KnownFields = map[string]any{}
		}
		summary.KnownFields["xs_function"] = xsFunction
	}
	return summary
}
