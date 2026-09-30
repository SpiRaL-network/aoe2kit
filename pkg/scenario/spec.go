package scenario

import (
	"encoding/json"
	"fmt"
)

type Spec struct {
	Sections []NamedSection
}

func (s *Spec) section(name string) (SectionSpec, bool) {
	for _, section := range s.Sections {
		if section.Name == name {
			return section.Spec, true
		}
	}
	return SectionSpec{}, false
}

type NamedSection struct {
	Name string
	Spec SectionSpec
}

type SectionSpec struct {
	Fields  []FieldSpec
	Structs map[string]SectionSpec
}

type FieldSpec struct {
	Name       string          `json:"name"`
	Type       string          `json:"type"`
	Repeat     int             `json:"repeat"`
	IsList     *bool           `json:"is_list"`
	Default    json.RawMessage `json:"default"`
	RepeatRule *RepeatRule     `json:"repeat_rule,omitempty"`
}

type RepeatRuleKind uint8

const (
	RepeatFromField RepeatRuleKind = iota + 1
	RepeatFromIndexedField
	RepeatProduct
	RepeatSquareRootLength
	RepeatNonEmpty
	RepeatProductNonEmpty
	RepeatFloatAtLeast
	RepeatRoundedFloatAtLeast
	RepeatTypeListOrValue
)

type FieldRef struct {
	Section string
	Name    string
}

// RepeatRule is Kit-owned codec logic for data-dependent field counts. It is
// deliberately typed: no foreign expression language is interpreted here.
type RepeatRule struct {
	Kind      RepeatRuleKind
	Fields    []FieldRef
	Threshold float64
	Then      int
	Else      int
}

func LoadCurrentDESpec() (*Spec, error) {
	return newOwnedDESpec(), nil
}

func LoadDESpecForVersion(version string) (*Spec, error) {
	spec, err := LoadCurrentDESpec()
	if err != nil {
		return nil, err
	}
	switch version {
	case "1.55":
		patchDE155To157EffectSpec(spec, version)
		patchDE155Spec(spec)
	case "1.56", "1.57":
		patchDE155To157EffectSpec(spec, version)
	case "1.58", "1.59":
		if version == "1.59" {
			patchDE159UnitSpec(spec)
		}
	default:
		return nil, fmt.Errorf("unsupported scenario version %q", version)
	}
	return spec, nil
}

// patchDE159UnitSpec records the 1.59 Units-layout byte currently pinned by
// preserved editor output. The byte is intentionally anonymous: its meaning
// is not established, but retaining it is sufficient for lossless writes.
func patchDE159UnitSpec(spec *Spec) {
	for i := range spec.Sections {
		if spec.Sections[i].Name != "Units" {
			continue
		}
		playerUnits := spec.Sections[i].Spec.Structs["PlayerUnitsStruct"]
		unit := playerUnits.Structs["UnitStruct"]
		patched := make([]FieldSpec, 0, len(unit.Fields)+1)
		for _, field := range unit.Fields {
			patched = append(patched, field)
			if field.Name == "garrisoned_in_id" {
				patched = append(patched, FieldSpec{
					Name:    "unknown_1_59_before_caption_string_id",
					Type:    "u8",
					Repeat:  1,
					Default: json.RawMessage("0"),
				})
			}
		}
		unit.Fields = patched
		playerUnits.Structs["UnitStruct"] = unit
		spec.Sections[i].Spec.Structs["PlayerUnitsStruct"] = playerUnits
		return
	}
}

// patchDE159EditorConditionSpec accounts for the condition record emitted by
// the 1.59 scenario editor. The editor changes the condition marker from 33
// to 34 and adds one opaque s32 immediately before xs_function. The marker is
// the layout discriminator; the extra field is retained for lossless writes.
func patchDE159EditorConditionSpec(spec *Spec) {
	for i := range spec.Sections {
		if spec.Sections[i].Name != "Triggers" {
			continue
		}
		trigger := spec.Sections[i].Spec.Structs["TriggerStruct"]
		condition := trigger.Structs["ConditionStruct"]
		patched := make([]FieldSpec, 0, len(condition.Fields)+1)
		for _, field := range condition.Fields {
			if field.Name == "static_value_33" {
				field.Default = json.RawMessage("34")
			}
			if field.Name == "xs_function" {
				patched = append(patched, FieldSpec{
					Name:    "unknown_1_59_editor_before_xs_function",
					Type:    "s32",
					Repeat:  1,
					Default: json.RawMessage("-1"),
				})
			}
			patched = append(patched, field)
		}
		condition.Fields = patched
		trigger.Structs["ConditionStruct"] = condition
		spec.Sections[i].Spec.Structs["TriggerStruct"] = trigger
		return
	}
}

func patchDE155To157EffectSpec(spec *Spec, version string) {
	remove := map[string]bool{
		"object_filter":          true,
		"use_tag_color_for_icon": true,
	}
	switch version {
	case "1.55":
		remove["issue_group_command"] = true
		remove["queue_action"] = true
		remove["mutual_diplomacy"] = true
		remove["building_list"] = true
		remove["wall_x1"] = true
		remove["wall_y1"] = true
		remove["wall_x2"] = true
		remove["wall_y2"] = true
	case "1.56":
		remove["mutual_diplomacy"] = true
		remove["building_list"] = true
		remove["wall_x1"] = true
		remove["wall_y1"] = true
		remove["wall_x2"] = true
		remove["wall_y2"] = true
	}
	for i := range spec.Sections {
		if spec.Sections[i].Name != "Triggers" {
			continue
		}
		triggerSpec := spec.Sections[i].Spec.Structs["TriggerStruct"]
		effectSpec := triggerSpec.Structs["EffectStruct"]
		filtered := effectSpec.Fields[:0]
		for _, field := range effectSpec.Fields {
			if remove[field.Name] {
				continue
			}
			if field.Name == "static_value_83" {
				switch version {
				case "1.55":
					field.Name = "static_value_80"
				case "1.56":
					field.Name = "static_value_81"
				case "1.57":
					field.Name = "static_value_75"
				}
			}
			filtered = append(filtered, field)
		}
		effectSpec.Fields = filtered
		triggerSpec.Structs["EffectStruct"] = effectSpec
		spec.Sections[i].Spec.Structs["TriggerStruct"] = triggerSpec
		return
	}
}

func patchDE155Spec(spec *Spec) {
	for i := range spec.Sections {
		if spec.Sections[i].Name != "DataHeader" {
			continue
		}
		playerData := spec.Sections[i].Spec.Structs["PlayerDataOneStruct"]
		playerData.Fields = []FieldSpec{
			{Name: "active", Type: "u32", Repeat: 1},
			{Name: "human", Type: "u32", Repeat: 1},
			{Name: "civilization", Type: "u32", Repeat: 1},
			{Name: "architecture_set", Type: "u32", Repeat: 1},
			{Name: "cty_mode", Type: "u32", Repeat: 1},
		}
		spec.Sections[i].Spec.Structs["PlayerDataOneStruct"] = playerData
		return
	}
}
