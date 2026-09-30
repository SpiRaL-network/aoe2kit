package scenario

import (
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
)

type EffectSummary struct {
	TriggerIndex      int            `json:"trigger_index,omitempty"`
	TriggerName       string         `json:"trigger_name,omitempty"`
	EffectIndex       int            `json:"effect_index,omitempty"`
	Type              int            `json:"type"`
	TypeName          string         `json:"type_name"`
	UnitConst         int            `json:"unit_const,omitempty"`
	SourcePlayer      int            `json:"source_player,omitempty"`
	TargetPlayer      int            `json:"target_player,omitempty"`
	Location          []int          `json:"location,omitempty"`
	Area              []int          `json:"area,omitempty"`
	TargetTrigger     int            `json:"target_trigger,omitempty"`
	StringID          int            `json:"string_id,omitempty"`
	Text              string         `json:"text,omitempty"`
	DisplayTime       int            `json:"display_time,omitempty"`
	TimeUnit          int            `json:"time_unit,omitempty"`
	TimerID           int            `json:"timer_id,omitempty"`
	ResetTimer        int            `json:"reset_timer,omitempty"`
	Sound             string         `json:"sound,omitempty"`
	Variable          int            `json:"variable,omitempty"`
	Variable2         int            `json:"variable2,omitempty"`
	Operation         int            `json:"operation,omitempty"`
	Quantity          int            `json:"quantity,omitempty"`
	QuantityFloat     float64        `json:"quantity_float,omitempty"`
	Technology        int            `json:"technology,omitempty"`
	Diplomacy         int            `json:"diplomacy,omitempty"`
	Resource          int            `json:"resource,omitempty"`
	ResourceQuantity  int            `json:"resource_quantity,omitempty"`
	ObjectAttribute   int            `json:"object_attribute,omitempty"`
	SelectedObjectIDs []int          `json:"selected_object_ids,omitempty"`
	RawFields         []int          `json:"raw_fields,omitempty"`
	KnownFields       map[string]any `json:"known_fields,omitempty"`
}

type EffectsReport struct {
	Path         string             `json:"path,omitempty"`
	Version      string             `json:"version"`
	Verification string             `json:"verification"`
	Total        int                `json:"total"`
	Types        []EffectTypeBucket `json:"types"`
}

type EffectTypeBucket struct {
	Type     int             `json:"type"`
	TypeName string          `json:"type_name"`
	Count    int             `json:"count"`
	Sample   []EffectSummary `json:"sample,omitempty"`
}

type EffectsOptions struct {
	IncludeRawFields bool
}

type EffectsCensusOptions struct {
	MaxInflatedBytes int
}

type EffectsCensusReport struct {
	Path                 string             `json:"path,omitempty"`
	Version              string             `json:"version"`
	Verification         string             `json:"verification"`
	HeaderBytes          int                `json:"header_bytes"`
	CompressedBytes      int                `json:"compressed_body_bytes"`
	InflatedBytes        int                `json:"inflated_body_bytes"`
	ParsedThroughSection string             `json:"parsed_through_section"`
	TriggerCount         int                `json:"trigger_count"`
	EffectCount          int                `json:"effect_count"`
	Types                []EffectTypeBucket `json:"types"`
}

func (f *File) Effects() EffectsReport {
	return f.EffectsWithOptions(EffectsOptions{})
}

func EffectsCensusFile(path string) (EffectsCensusReport, error) {
	return EffectsCensusFileWithOptions(path, EffectsCensusOptions{})
}

func EffectsCensusFileWithOptions(path string, opts EffectsCensusOptions) (EffectsCensusReport, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return EffectsCensusReport{}, err
	}
	report, err := EffectsCensusWithOptions(data, opts)
	if err != nil {
		return EffectsCensusReport{}, err
	}
	report.Path = path
	return report, nil
}

func EffectsCensus(data []byte) (EffectsCensusReport, error) {
	return EffectsCensusWithOptions(data, EffectsCensusOptions{})
}

func EffectsCensusWithOptions(data []byte, opts EffectsCensusOptions) (EffectsCensusReport, error) {
	if len(data) < 8 {
		return EffectsCensusReport{}, errors.New("scenario too short")
	}
	headerModel, _, headerLen, err := DecodeScenarioHeaderNode(data)
	if err != nil {
		return EffectsCensusReport{}, fmt.Errorf("decode FileHeader: %w", err)
	}
	version := headerModel.Version
	if !SupportsReadVersion(version) {
		return EffectsCensusReport{}, unsupportedReadVersionError(version)
	}
	bodySpec, err := LoadDESpecForVersion(version)
	if err != nil {
		return EffectsCensusReport{}, err
	}
	maxBytes := opts.MaxInflatedBytes
	if maxBytes == 0 {
		maxBytes = maxInflatedScenarioBytes()
	}
	body, err := inflateRawLimited(data[headerLen:], maxBytes)
	if err != nil {
		return EffectsCensusReport{}, fmt.Errorf("inflate scenario body: %w", err)
	}
	parser := parser{
		data:     body,
		sections: map[string]*parsedSection{},
		version:  version,
	}
	report := EffectsCensusReport{
		Version:              version,
		Verification:         "structure_verified_not_engine_verified",
		HeaderBytes:          headerLen,
		CompressedBytes:      len(data) - headerLen,
		InflatedBytes:        len(body),
		ParsedThroughSection: "Triggers",
	}
	for _, sectionSpec := range bodySpec.Sections {
		if sectionSpec.Name == "FileHeader" {
			continue
		}
		if sectionSpec.Name == "Triggers" {
			if err := parser.parseTriggersCensus(sectionSpec.Spec, &report); err != nil {
				return EffectsCensusReport{}, fmt.Errorf("section Triggers: %w", err)
			}
			break
		}
		if _, err := parser.parseNodeDiscard(sectionSpec.Name, sectionSpec.Spec); err != nil {
			return EffectsCensusReport{}, fmt.Errorf("section %s: %w", sectionSpec.Name, err)
		}
	}
	if report.ParsedThroughSection == "" {
		return EffectsCensusReport{}, errors.New("missing Triggers section")
	}
	return report, nil
}

func (p *parser) parseTriggersCensus(sectionSpec SectionSpec, report *EffectsCensusReport) error {
	node := &parsedNode{Name: "Triggers", Start: p.off}
	buckets := map[int]*EffectTypeBucket{}
	for _, fieldSpec := range sectionSpec.Fields {
		repeat, err := p.constructRepeat(fieldSpec, node)
		if err != nil {
			return fmt.Errorf("%s: %w", fieldSpec.Name, err)
		}
		if repeat < 0 {
			repeat = 0
		}
		field := &parsedNode{Name: fieldSpec.Name, Start: p.off}
		switch field.Name {
		case "trigger_data":
			for i := 0; i < repeat; i++ {
				trigger, consumed, err := DecodeTriggerNode(p.data[p.off:])
				if err != nil {
					return fmt.Errorf("TriggerStruct[%d]: %w", i, err)
				}
				p.off += consumed
				for _, effect := range trigger.list("effect_data") {
					effectType, _ := effect.intValue("effect_type")
					bucket := buckets[effectType]
					if bucket == nil {
						bucket = &EffectTypeBucket{Type: effectType, TypeName: EffectTypeName(effectType)}
						buckets[effectType] = bucket
					}
					bucket.Count++
					report.EffectCount++
				}
			}
			field.Value = make([]any, repeat)
			field.End = p.off
		case "variable_data":
			for i := 0; i < repeat; i++ {
				_, consumed, err := DecodeVariableNode(p.data[p.off:])
				if err != nil {
					return fmt.Errorf("VariableStruct[%d]: %w", i, err)
				}
				p.off += consumed
			}
			field.Value = make([]any, repeat)
			field.End = p.off
		default:
			if strings.HasPrefix(fieldSpec.Type, "struct:") {
				return fmt.Errorf("unexpected nested trigger census field %s", fieldSpec.Type)
			}
			if err := p.parsePrimitiveFieldDiscard(field, fieldSpec, repeat); err != nil {
				return err
			}
		}
		if fieldSpec.Name == "number_of_triggers" {
			report.TriggerCount = asInt(field.Value)
		}
		node.Fields = append(node.Fields, field)
	}
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
	for _, key := range keys {
		report.Types = append(report.Types, *buckets[key])
	}
	report.ParsedThroughSection = "Triggers"
	return nil
}

func (p *parser) parseNodeDiscard(name string, spec SectionSpec) (*parsedNode, error) {
	if name == "DataHeader" {
		_, node, consumed, err := DecodeDataHeaderNode(p.data[p.off:])
		if err != nil {
			return nil, err
		}
		node.Start = p.off
		node.End = p.off + consumed
		p.off += consumed
		return node, nil
	}
	if name == "Units" {
		return p.parseUnitsNode(spec)
	}
	if name == "Map" {
		_, node, consumed, err := DecodeMapLayoutNode(p.data[p.off:])
		if err != nil {
			return nil, err
		}
		node.Start = p.off
		node.End = p.off + consumed
		p.off += consumed
		return node, nil
	}
	if name == "Diplomacy" {
		_, node, consumed, err := DecodeDiplomacyNode(p.data[p.off:])
		if err != nil {
			return nil, err
		}
		node.Start = p.off
		node.End = p.off + consumed
		p.off += consumed
		return node, nil
	}
	if name == "GlobalVictory" {
		_, node, consumed, err := DecodeGlobalVictoryNode(p.data[p.off:])
		if err != nil {
			return nil, err
		}
		node.Start = p.off
		node.End = p.off + consumed
		p.off += consumed
		return node, nil
	}
	if name == "Messages" {
		node, consumed, err := DecodeMessagesNode(p.data[p.off:])
		if err != nil {
			return nil, err
		}
		node.Start = p.off
		node.End = p.off + consumed
		p.off += consumed
		return node, nil
	}
	if name == "Cinematics" {
		node, consumed, err := DecodeCinematicsNode(p.data[p.off:])
		if err != nil {
			return nil, err
		}
		node.Start = p.off
		node.End = p.off + consumed
		p.off += consumed
		return node, nil
	}
	if name == "Options" {
		node, consumed, err := DecodeOptionsNode(p.data[p.off:])
		if err != nil {
			return nil, err
		}
		node.Start = p.off
		node.End = p.off + consumed
		p.off += consumed
		return node, nil
	}
	if name == "BackgroundImage" {
		node, consumed, err := DecodeBackgroundImageNode(p.data[p.off:])
		if err != nil {
			return nil, err
		}
		node.Start = p.off
		node.End = p.off + consumed
		p.off += consumed
		return node, nil
	}
	if name == "PlayerDataTwo" {
		node, consumed, err := DecodePlayerDataTwoNode(p.data[p.off:])
		if err != nil {
			return nil, err
		}
		node.Start = p.off
		node.End = p.off + consumed
		p.off += consumed
		return node, nil
	}
	if name == "Files" {
		node, consumed, err := DecodeFilesNode(p.data[p.off:])
		if err != nil {
			return nil, err
		}
		node.Start = p.off
		node.End = p.off + consumed
		p.off += consumed
		return node, nil
	}
	return nil, fmt.Errorf("unowned scenario discard node %q", name)
}

func (p *parser) parsePrimitiveFieldDiscard(node *parsedNode, fieldSpec FieldSpec, repeat int) error {
	values := make([]any, 0, repeat)
	var raw []byte
	for i := 0; i < repeat; i++ {
		value, chunk, err := p.readPrimitive(fieldSpec.Type)
		if err != nil {
			return fmt.Errorf("%s item %d: %w", fieldSpec.Name, i, err)
		}
		values = append(values, value)
		raw = append(raw, chunk...)
	}
	node.Raw = raw
	node.End = p.off
	if fieldSpec.IsList != nil && !*fieldSpec.IsList && len(values) > 0 {
		node.Value = values[0]
	} else if repeat == 1 && fieldSpec.IsList == nil {
		node.Value = values[0]
	} else {
		node.Value = values
	}
	return nil
}

func (f *File) EffectsWithOptions(opts EffectsOptions) EffectsReport {
	report := EffectsReport{
		Path:         f.Path,
		Version:      f.Version,
		Verification: "structure_verified_not_engine_verified",
	}
	if f.root == nil {
		return report
	}
	sampleLimit := 1
	if opts.IncludeRawFields {
		sampleLimit = 3
	}
	section := f.root.section("Triggers")
	if section == nil {
		return report
	}
	buckets := map[int]*EffectTypeBucket{}
	for triggerIndex, trigger := range section.list("trigger_data") {
		triggerName, _ := trigger.stringValue("trigger_name")
		for effectIndex, effect := range trigger.list("effect_data") {
			summary := summarizeEffect(effect, opts)
			summary.TriggerIndex = triggerIndex
			summary.TriggerName = triggerName
			summary.EffectIndex = effectIndex
			report.Total++
			bucket := buckets[summary.Type]
			if bucket == nil {
				bucket = &EffectTypeBucket{Type: summary.Type, TypeName: summary.TypeName}
				buckets[summary.Type] = bucket
			}
			bucket.Count++
			if len(bucket.Sample) < sampleLimit {
				bucket.Sample = append(bucket.Sample, summary)
			}
		}
	}
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
	for _, key := range keys {
		report.Types = append(report.Types, *buckets[key])
	}
	return report
}

func summarizeEffect(effect *parsedNode, opts EffectsOptions) EffectSummary {
	effectType, _ := effect.intValue("effect_type")
	summary := EffectSummary{
		Type:        effectType,
		TypeName:    EffectTypeName(effectType),
		KnownFields: map[string]any{},
	}
	if opts.IncludeRawFields {
		summary.RawFields = rawEffectFields(effect)
	}
	putInt := func(outName, fieldName string) int {
		value, ok := effect.intValue(fieldName)
		if ok && value != -1 {
			summary.KnownFields[outName] = value
		}
		return value
	}
	putText := func(outName, fieldName string) string {
		value, ok := effect.stringValue(fieldName)
		if ok && value != "" {
			summary.KnownFields[outName] = value
		}
		return value
	}
	location := func() []int {
		x, _ := effect.intValue("location_x")
		y, _ := effect.intValue("location_y")
		if x == -1 && y == -1 {
			return nil
		}
		return []int{x, y}
	}
	area := func() []int {
		x1, _ := effect.intValue("area_x1")
		y1, _ := effect.intValue("area_y1")
		x2, _ := effect.intValue("area_x2")
		y2, _ := effect.intValue("area_y2")
		if x1 == -1 && y1 == -1 && x2 == -1 && y2 == -1 {
			return nil
		}
		return []int{x1, y1, x2, y2}
	}
	selectedObjects := func() []int {
		ids := effect.intList("selected_object_ids")
		if len(ids) == 0 {
			return nil
		}
		return ids
	}
	switch effectType {
	case 1:
		summary.SourcePlayer = putInt("source_player", "source_player")
		summary.TargetPlayer = putInt("target_player", "target_player")
		summary.Diplomacy = putInt("diplomacy", "diplomacy")
	case 2:
		summary.SourcePlayer = putInt("source_player", "source_player")
		summary.Technology = putInt("technology", "technology")
	case 3:
		summary.SourcePlayer = putInt("source_player", "source_player")
		summary.Text = putText("text", "message")
	case 5:
		summary.SourcePlayer = putInt("source_player", "source_player")
		summary.TargetPlayer = putInt("target_player", "target_player")
		summary.Resource = putInt("resource", "tribute_list")
		summary.Quantity = putInt("quantity", "quantity")
	case 8, 9:
		summary.TargetTrigger = putInt("target_trigger", "trigger_id")
	case 10:
		summary.SourcePlayer = putInt("source_player", "source_player")
		summary.Quantity = putInt("goal", "ai_script_goal")
	case 11, 25:
		summary.UnitConst = putInt("unit_const", "object_list_unit_id")
		summary.SourcePlayer = putInt("source_player", "source_player")
		summary.Location = location()
		if len(summary.Location) > 0 {
			summary.KnownFields["location"] = summary.Location
		}
	case 12, 16, 19, 30, 35:
		summary.UnitConst = putInt("unit_const", "object_list_unit_id")
		summary.SourcePlayer = putInt("source_player", "source_player")
		summary.Location = location()
		if len(summary.Location) > 0 {
			summary.KnownFields["location"] = summary.Location
		}
	case 14, 15, 18, 24, 26, 27, 28, 29, 31, 32, 33, 34, 42, 43, 44, 49, 58, 59, 60, 61, 62, 70, 71, 73, 74, 77, 78, 88, 98, 99, 105, 106, 107:
		summary.UnitConst = putInt("unit_const", "object_list_unit_id")
		summary.SourcePlayer = putInt("source_player", "source_player")
		summary.TargetPlayer = putInt("target_player", "target_player")
		summary.StringID = putInt("string_id", "string_id")
		summary.ObjectAttribute = putInt("object_attribute", "object_attributes")
		summary.Operation = putInt("operation", "operation")
		summary.Quantity = putInt("quantity", "quantity")
		summary.Area = area()
		if len(summary.Area) > 0 {
			summary.KnownFields["area"] = summary.Area
		}
		summary.Text = putText("text", "message")
		summary.SelectedObjectIDs = selectedObjects()
		if len(summary.SelectedObjectIDs) > 0 {
			summary.KnownFields["selected_object_ids"] = summary.SelectedObjectIDs
		}
	case 38:
		summary.UnitConst = putInt("unit_const", "object_list_unit_id")
		summary.SourcePlayer = putInt("source_player", "source_player")
		putInt("enabled", "enabled")
		summary.Area = area()
		if len(summary.Area) > 0 {
			summary.KnownFields["area"] = summary.Area
		}
		putInt("object_group", "object_group")
		putInt("object_type", "object_type")
		putInt("max_units_affected", "max_units_affected")
		summary.SelectedObjectIDs = selectedObjects()
		if len(summary.SelectedObjectIDs) > 0 {
			summary.KnownFields["selected_object_ids"] = summary.SelectedObjectIDs
		}
	case 40:
		summary.UnitConst = putInt("unit_const", "object_list_unit_id")
		summary.SourcePlayer = putInt("source_player", "source_player")
		putInt("resource_1", "resource_1")
		putInt("resource_1_quantity", "resource_1_quantity")
		putInt("resource_2", "resource_2")
		putInt("resource_2_quantity", "resource_2_quantity")
		putInt("resource_3", "resource_3")
		putInt("resource_3_quantity", "resource_3_quantity")
	case 20:
		summary.SourcePlayer = putInt("source_player", "source_player")
		summary.StringID = putInt("string_id", "string_id")
		summary.DisplayTime = putInt("display_time", "display_time")
		summary.Text = putText("text", "message")
		summary.Sound = putText("sound", "sound_name")
	case 36:
		summary.UnitConst = putInt("unit_const", "object_list_unit_id")
		summary.SourcePlayer = putInt("source_player", "source_player")
		summary.Area = area()
		if len(summary.Area) > 0 {
			summary.KnownFields["area"] = summary.Area
		}
		summary.Operation = putInt("attack_stance", "attack_stance")
	case 37:
		if value, ok := effect.intValue("source_player"); ok {
			summary.SourcePlayer = value
			summary.KnownFields["source_player"] = value
		}
		summary.DisplayTime = putInt("display_time", "display_time")
		summary.TimeUnit = putInt("time_unit", "time_unit")
		summary.TimerID = putInt("timer_id", "timer")
		summary.ResetTimer = putInt("reset_timer", "reset_timer")
		summary.Text = putText("message", "message")
	case 51:
		summary.UnitConst = putInt("unit_const", "object_list_unit_id")
		summary.SourcePlayer = putInt("source_player", "source_player")
		summary.ObjectAttribute = putInt("object_attribute", "object_attributes")
		summary.Operation = putInt("operation", "operation")
		summary.Quantity = putInt("quantity", "quantity")
	case 52:
		summary.SourcePlayer = putInt("source_player", "source_player")
		summary.Resource = putInt("resource", "tribute_list")
		summary.Operation = putInt("operation", "operation")
		summary.Quantity = putInt("quantity", "quantity")
	case 55:
		summary.Text = putText("text", "message")
	case 56, 79, 81, 82, 86, 87, 100, 101:
		summary.Variable = putInt("variable", "variable")
		summary.Variable2 = putInt("variable2", "variable2")
		summary.Operation = putInt("operation", "operation")
		summary.Quantity = putInt("quantity", "quantity")
		summary.Text = putText("text", "message")
	case 57:
		summary.TimerID = putInt("timer_id", "timer")
	case 39:
		summary.SourcePlayer = putInt("source_player", "source_player")
		summary.Technology = putInt("technology", "technology")
		putInt("enabled", "enabled")
	case 47:
		summary.SourcePlayer = putInt("source_player", "source_player")
		summary.Technology = putInt("technology", "technology")
		summary.UnitConst = putInt("unit_const", "object_list_unit_id")
		putInt("button_location", "button_location")
	case 63:
		summary.SourcePlayer = putInt("source_player", "source_player")
		summary.Technology = putInt("technology", "technology")
		putInt("resource_1", "resource_1")
		putInt("resource_1_quantity", "resource_1_quantity")
		putInt("resource_2", "resource_2")
		putInt("resource_2_quantity", "resource_2_quantity")
		putInt("resource_3", "resource_3")
		putInt("resource_3_quantity", "resource_3_quantity")
	case 64, 65, 66, 67, 68, 84, 85:
		summary.SourcePlayer = putInt("source_player", "source_player")
		summary.Technology = putInt("technology", "technology")
		summary.Quantity = putInt("quantity", "quantity")
		if effectType == 64 {
			putInt("research_time", "quantity")
		}
		if effectType == 84 {
			putInt("icon", "quantity")
		}
		summary.Text = putText("text", "message")
	case 75, 108:
		summary.UnitConst = putInt("unit_const", "object_list_unit_id")
		summary.SourcePlayer = putInt("source_player", "source_player")
		summary.Location = location()
		if len(summary.Location) > 0 {
			summary.KnownFields["location"] = summary.Location
		}
		summary.Area = area()
		if len(summary.Area) > 0 {
			summary.KnownFields["area"] = summary.Area
		}
		summary.SelectedObjectIDs = selectedObjects()
		if len(summary.SelectedObjectIDs) > 0 {
			summary.KnownFields["selected_object_ids"] = summary.SelectedObjectIDs
		}
	case 76, 103:
		summary.SourcePlayer = putInt("source_player", "source_player")
		summary.Technology = putInt("technology", "technology")
		summary.SelectedObjectIDs = selectedObjects()
		if len(summary.SelectedObjectIDs) > 0 {
			summary.KnownFields["selected_object_ids"] = summary.SelectedObjectIDs
		}
	case 102:
		summary.UnitConst = putInt("unit_const", "object_list_unit_id")
		summary.SourcePlayer = putInt("source_player", "source_player")
		putInt("train_location_unit_const", "object_list_unit_id_2")
		putInt("button_location", "button_location")
		putInt("hotkey", "hotkey")
		putInt("train_time", "train_time")
		summary.SelectedObjectIDs = selectedObjects()
		if len(summary.SelectedObjectIDs) > 0 {
			summary.KnownFields["selected_object_ids"] = summary.SelectedObjectIDs
		}
	case 104:
		summary.SourcePlayer = putInt("source_player", "source_player")
		summary.ObjectAttribute = putInt("object_attribute", "object_attributes")
		summary.Operation = putInt("operation", "operation")
		summary.Quantity = putInt("quantity", "quantity")
		summary.Text = putText("text", "message")
	}
	if len(summary.KnownFields) == 0 {
		summary.KnownFields = nil
	}
	return summary
}

func rawEffectFields(effect *parsedNode) []int {
	raw := effect.raw()
	count := len(raw) / 4
	if count > 84 {
		count = 84
	}
	out := make([]int, count)
	for i := 0; i < count; i++ {
		out[i] = int(int32(binary.LittleEndian.Uint32(raw[i*4:])))
	}
	return out
}

func EffectTypeName(effectType int) string {
	if name, ok := effectTypeNames[effectType]; ok {
		return name
	}
	return fmt.Sprintf("unknown_%d", effectType)
}

func EffectTypeForOp(op string) (int, bool) {
	value, ok := effectTypesByRecipeOp[op]
	return value, ok
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

var effectTypesByRecipeOp = map[string]int{
	"change_diplomacy":                1,
	"display_instructions":            20,
	"display_timer":                   37,
	"send_chat":                       3,
	"play_sound":                      4,
	"research_technology":             2,
	"create_object":                   11,
	"place_foundation":                25,
	"build_object":                    108,
	"kill_object":                     14,
	"remove_object":                   15,
	"replace_object":                  43,
	"task_object":                     12,
	"change_view":                     16,
	"change_ownership":                18,
	"freeze_unit":                     22,
	"change_object_name":              26,
	"change_object_description":       44,
	"change_object_hp":                27,
	"damage_object":                   24,
	"heal_object":                     34,
	"change_object_attack":            28,
	"change_object_armor":             31,
	"change_object_range":             32,
	"change_object_speed":             33,
	"change_object_caption":           88,
	"stop_unit":                       29,
	"teleport_object":                 35,
	"change_object_stance":            36,
	"set_player_visibility":           41,
	"set_visibility":                  41,
	"reveal_map":                      41,
	"enable_disable_object":           38,
	"enable_disable_technology":       39,
	"change_object_cost":              40,
	"modify_attribute":                51,
	"modify_resource":                 52,
	"change_player_name":              45,
	"change_train_location":           46,
	"change_technology_location":      47,
	"change_civilization_name":        48,
	"disable_unit_targeting":          61,
	"enable_unit_targeting":           62,
	"change_technology_cost":          63,
	"change_technology_research_time": 64,
	"change_technology_name":          65,
	"change_technology_description":   66,
	"disable_object_selection":        70,
	"enable_object_selection":         71,
	"enable_object_deletion":          73,
	"disable_object_deletion":         74,
	"train_unit":                      75,
	"initiate_research":               76,
	"change_technology_icon":          84,
	"change_technology_hotkey":        85,
	"change_player_color":             89,
	"disable_unit_attackable":         98,
	"enable_unit_attackable":          99,
	"add_train_location":              102,
	"research_local_technology":       103,
	"script_call":                     55,
	"change_variable":                 56,
	"modify_variable":                 56,
	"clear_timer":                     57,
	"modify_variable_by_variable":     100,
	"activate_trigger":                8,
	"deactivate_trigger":              9,
	"declare_victory":                 13,
}
