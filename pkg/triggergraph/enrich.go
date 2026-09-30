package triggergraph

import (
	"encoding/binary"
	"math"

	"aoe2kit/pkg/datfacts"
)

var effectFieldNames = []string{
	"effect_type", "static_value_83", "ai_script_goal", "quantity", "tribute_list",
	"diplomacy", "number_of_units_selected", "legacy_location_object_reference",
	"object_list_unit_id", "source_player", "target_player", "technology",
	"string_id", "unknown_2", "display_time", "trigger_id", "location_x",
	"location_y", "area_x1", "area_y1", "area_x2", "area_y2", "object_group",
	"object_type", "instruction_panel_position", "attack_stance", "time_unit",
	"enabled", "food", "wood", "stone", "gold", "item_id", "flash_object",
	"force_research_technology", "visibility_state", "scroll", "operation",
	"object_list_unit_id_2", "button_location", "ai_signal_value", "unknown_3",
	"object_attributes", "variable", "timer", "facet", "location_object_reference",
	"play_sound", "player_color", "unknown_4", "color_mood", "reset_timer",
	"object_state", "action_type", "resource_1", "resource_1_quantity",
	"resource_2", "resource_2_quantity", "resource_3", "resource_3_quantity",
	"decision_id", "string_id_option1", "string_id_option2", "variable2",
	"max_units_affected", "disable_garrison_unload_sound", "hotkey", "train_time",
	"local_technology", "disable_sound", "object_group2", "object_type2",
	"quantity_float", "facet2", "global_sound", "issue_group_command",
	"queue_action", "mutual_diplomacy", "building_list", "wall_x1", "wall_y1",
	"wall_x2", "wall_y2", "object_filter", "use_tag_color_for_icon",
}

var conditionFieldNames = []string{
	"condition_type", "static_value_33", "quantity", "attribute", "unit_object",
	"next_object", "object_list", "source_player", "technology", "timer",
	"trigger_id", "area_x1", "area_y1", "area_x2", "area_y2", "object_group",
	"object_type", "ai_signal", "inverted", "unknown_2", "variable",
	"comparison", "target_player", "unit_ai_action", "unknown_4", "object_state",
	"timer_id", "victory_timer_type", "include_changeable_weapon_objects",
	"decision_id", "decision_option", "variable2", "local_technology",
	"object_group2", "object_type2",
}

const legacyEffectIdentityFieldPrefixCount = 84

func enrichEffect(effect map[string]any, values []int, raw []byte) {
	effectType := 0
	if len(values) > 0 {
		effectType = values[0]
	}
	effect["type_name"] = effectTypeName(effectType)
	known := namedIntFields(values, effectFieldNames)
	if len(raw) >= 73*4 {
		quantityFloat := math.Float32frombits(binary.LittleEndian.Uint32(raw[72*4:]))
		if !math.IsNaN(float64(quantityFloat)) && quantityFloat != 0 {
			known["quantity_float"] = quantityFloat
		}
	}
	if msg, _ := effect["message"].(string); msg != "" {
		known["message"] = msg
	}
	if len(known) > 0 {
		effect["known_fields"] = known
	}
}

func enrichCondition(condition map[string]any, values []int) {
	conditionType := 0
	if len(values) > 0 {
		conditionType = values[0]
	}
	condition["type_name"] = conditionTypeName(conditionType)
	known := namedIntFields(values, conditionFieldNames)
	if xs, _ := condition["xs_function"].(string); xs != "" {
		known["xs_function"] = xs
	}
	if len(known) > 0 {
		condition["known_fields"] = known
	}
}

func namedIntFields(values []int, names []string) map[string]any {
	out := map[string]any{}
	for i, value := range values {
		if i >= len(names) {
			break
		}
		if value == -1 {
			continue
		}
		name := names[i]
		if name == "" || name == "static_value_83" || name == "static_value_33" {
			continue
		}
		out[name] = value
		if enumName := triggerEnumValueName(name, value); enumName != "" {
			out[name+"_name"] = enumName
		}
	}
	return out
}

func triggerEnumValueName(field string, value int) string {
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

func identityTriggers(triggers []map[string]any) []map[string]any {
	out := make([]map[string]any, 0, len(triggers))
	for _, trigger := range triggers {
		out = append(out, identityTrigger(trigger))
	}
	return out
}

func identityTrigger(trigger map[string]any) map[string]any {
	out := map[string]any{}
	for _, field := range []string{
		"conditions", "condition_order", "description", "description_order",
		"description_stid", "display_as_objective", "display_on_screen",
		"effect_order", "effects", "enabled", "execute_on_load", "looping",
		"make_header", "mute_objectives", "name", "short_description",
		"short_description_stid",
	} {
		value, ok := trigger[field]
		if !ok {
			continue
		}
		switch field {
		case "effects":
			out[field] = identityChildren(value, identityEffect)
		case "conditions":
			out[field] = identityChildren(value, identityCondition)
		default:
			out[field] = value
		}
	}
	return out
}

func identityChildren(value any, fn func(map[string]any) map[string]any) []map[string]any {
	children, _ := value.([]map[string]any)
	out := make([]map[string]any, 0, len(children))
	for _, child := range children {
		out = append(out, fn(child))
	}
	return out
}

func identityEffect(effect map[string]any) map[string]any {
	out := map[string]any{}
	for _, field := range []string{"fields_prefix", "message", "raw_sha256", "selected_object_ids", "type"} {
		if value, ok := effect[field]; ok {
			if field == "fields_prefix" {
				out[field] = legacyEffectIdentityPrefix(value)
				continue
			}
			out[field] = value
		}
	}
	return out
}

func legacyEffectIdentityPrefix(value any) any {
	values, ok := value.([]int)
	if !ok || len(values) <= legacyEffectIdentityFieldPrefixCount {
		return value
	}
	return append([]int(nil), values[:legacyEffectIdentityFieldPrefixCount]...)
}

func identityCondition(condition map[string]any) map[string]any {
	out := map[string]any{}
	for _, field := range []string{"fields", "type", "xs_function"} {
		if value, ok := condition[field]; ok {
			out[field] = value
		}
	}
	return out
}

func effectTypeName(effectType int) string {
	if name, ok := effectTypeNames[effectType]; ok {
		return name
	}
	return "unknown"
}

func conditionTypeName(conditionType int) string {
	if name, ok := conditionTypeNames[conditionType]; ok {
		return name
	}
	return "unknown"
}

var effectTypeNames = map[int]string{
	1:   "change_diplomacy",
	2:   "research_technology",
	3:   "send_chat",
	4:   "play_sound",
	5:   "tribute",
	6:   "unlock_gate",
	7:   "lock_gate",
	8:   "activate_trigger",
	9:   "deactivate_trigger",
	10:  "ai_script_goal",
	11:  "create_object",
	12:  "task_object",
	13:  "declare_victory",
	14:  "kill_object",
	15:  "remove_object",
	16:  "change_view",
	17:  "unload",
	18:  "change_ownership",
	19:  "patrol",
	20:  "display_instructions",
	21:  "clear_instructions",
	22:  "freeze_unit",
	23:  "use_advanced_buttons",
	24:  "damage_object",
	25:  "place_foundation",
	26:  "change_object_name",
	27:  "change_object_hp",
	28:  "change_object_attack",
	29:  "stop_unit",
	30:  "attack_move",
	31:  "change_object_armor",
	32:  "change_object_range",
	33:  "change_object_speed",
	34:  "heal_object",
	35:  "teleport_object",
	36:  "change_object_stance",
	37:  "display_timer",
	38:  "enable_disable_object",
	39:  "enable_disable_technology",
	40:  "change_object_cost",
	41:  "set_player_visibility",
	42:  "change_object_icon",
	43:  "replace_object",
	44:  "change_object_description",
	45:  "change_player_name",
	46:  "change_train_location",
	47:  "change_technology_location",
	48:  "change_civilization_name",
	49:  "create_garrisoned_object",
	50:  "acknowledge_ai_signal",
	51:  "modify_attribute",
	52:  "modify_resource",
	53:  "modify_resource_by_variable",
	54:  "set_building_gather_point",
	55:  "script_call",
	56:  "change_variable",
	57:  "clear_timer",
	58:  "change_object_player_color",
	59:  "change_object_civilization_name",
	60:  "change_object_player_name",
	61:  "disable_unit_targeting",
	62:  "enable_unit_targeting",
	63:  "change_technology_cost",
	64:  "change_technology_research_time",
	65:  "change_technology_name",
	66:  "change_technology_description",
	67:  "enable_technology_stacking",
	68:  "disable_technology_stacking",
	69:  "acknowledge_multiplayer_ai_signal",
	70:  "disable_object_selection",
	71:  "enable_object_selection",
	72:  "change_color_mood",
	73:  "enable_object_deletion",
	74:  "disable_object_deletion",
	75:  "train_unit",
	76:  "initiate_research",
	77:  "create_object_attack",
	78:  "create_object_armor",
	79:  "modify_attribute_by_variable",
	80:  "set_object_cost",
	81:  "load_key_value",
	82:  "store_key_value",
	83:  "delete_key",
	84:  "change_technology_icon",
	85:  "change_technology_hotkey",
	86:  "modify_variable_by_resource",
	87:  "modify_variable_by_attribute",
	88:  "change_object_caption",
	89:  "change_player_color",
	90:  "create_decision",
	98:  "disable_unit_attackable",
	99:  "enable_unit_attackable",
	100: "modify_variable_by_variable",
	101: "count_units_into_variable",
	102: "add_train_location",
	103: "research_local_technology",
	104: "modify_attribute_for_class",
	105: "modify_object_attribute",
	106: "modify_object_attribute_by_variable",
	107: "change_object_visibility",
	108: "build_object",
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
