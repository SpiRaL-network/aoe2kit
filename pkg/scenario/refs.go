package scenario

import (
	"fmt"
	"sort"
	"strings"
)

type ReferenceOptions struct {
	Kind     string `json:"kind,omitempty"`
	TargetID *int   `json:"target_id,omitempty"`
}

type ReferenceReport struct {
	Path         string              `json:"path,omitempty"`
	Version      string              `json:"version"`
	Verification string              `json:"verification"`
	Summary      ReferenceSummary    `json:"summary"`
	References   []ScenarioReference `json:"references,omitempty"`
	Warnings     []string            `json:"warnings,omitempty"`
}

type ReferenceSummary struct {
	Total  int            `json:"total"`
	ByKind map[string]int `json:"by_kind"`
}

type ScenarioReference struct {
	Kind            string `json:"kind"`
	TargetID        int    `json:"target_id"`
	TargetLabel     string `json:"target_label,omitempty"`
	SourceKind      string `json:"source_kind"`
	SourcePath      string `json:"source_path"`
	Field           string `json:"field"`
	TriggerIndex    int    `json:"trigger_index,omitempty"`
	TriggerName     string `json:"trigger_name,omitempty"`
	ChildKind       string `json:"child_kind,omitempty"`
	ChildIndex      int    `json:"child_index,omitempty"`
	UnitPlayer      int    `json:"unit_player,omitempty"`
	UnitIndex       int    `json:"unit_index,omitempty"`
	UnitReferenceID int    `json:"unit_reference_id,omitempty"`
}

func ReferencesFile(path string, opts ReferenceOptions) (ReferenceReport, error) {
	file, err := Open(path)
	if err != nil {
		return ReferenceReport{}, err
	}
	report, err := file.References(opts)
	if err != nil {
		return ReferenceReport{}, err
	}
	report.Path = path
	return report, nil
}

func (f *File) References(opts ReferenceOptions) (ReferenceReport, error) {
	kind := strings.ToLower(strings.TrimSpace(opts.Kind))
	if kind == "" {
		kind = "all"
	}
	switch kind {
	case "all", "unit", "object", "trigger", "variable", "string":
	default:
		return ReferenceReport{}, fmt.Errorf("unknown reference kind %q; expected unit, trigger, variable, string, or all", opts.Kind)
	}
	if kind == "object" {
		kind = "unit"
	}
	labels := f.referenceLabels()
	report := ReferenceReport{
		Path:         f.Path,
		Version:      f.Version,
		Verification: "structure_verified_not_engine_verified",
		Summary:      ReferenceSummary{ByKind: map[string]int{}},
	}
	add := func(ref ScenarioReference) {
		if kind != "all" && ref.Kind != kind {
			return
		}
		if opts.TargetID != nil && ref.TargetID != *opts.TargetID {
			return
		}
		if ref.TargetLabel == "" {
			ref.TargetLabel = labels[ref.Kind][ref.TargetID]
		}
		report.References = append(report.References, ref)
		report.Summary.Total++
		report.Summary.ByKind[ref.Kind]++
	}
	f.addTriggerReferences(add)
	f.addUnitReferences(add)
	sort.Slice(report.References, func(i, j int) bool {
		a, b := report.References[i], report.References[j]
		if a.Kind != b.Kind {
			return a.Kind < b.Kind
		}
		if a.TargetID != b.TargetID {
			return a.TargetID < b.TargetID
		}
		return a.SourcePath < b.SourcePath
	})
	return report, nil
}

func (f *File) addTriggerReferences(add func(ScenarioReference)) {
	triggers := f.root.section("Triggers")
	if triggers == nil {
		return
	}
	for triggerIndex, trigger := range triggers.list("trigger_data") {
		triggerName, _ := trigger.stringValue("trigger_name")
		if id, ok := trigger.intValue("description_string_table_id"); ok && id >= 0 {
			add(triggerReference("string", id, triggerIndex, triggerName, "trigger", -1, "description_string_table_id"))
		}
		if id, ok := trigger.intValue("short_description_string_table_id"); ok && id >= 0 {
			add(triggerReference("string", id, triggerIndex, triggerName, "trigger", -1, "short_description_string_table_id"))
		}
		for conditionIndex, condition := range trigger.list("condition_data") {
			for _, field := range []string{"unit_object", "next_object"} {
				if id, ok := condition.intValue(field); ok && id >= 0 {
					add(triggerReference("unit", id, triggerIndex, triggerName, "condition", conditionIndex, field))
				}
			}
			if id, ok := condition.intValue("trigger_id"); ok && id >= 0 {
				add(triggerReference("trigger", id, triggerIndex, triggerName, "condition", conditionIndex, "trigger_id"))
			}
			if id, ok := condition.intValue("variable"); ok && id >= 0 {
				add(triggerReference("variable", id, triggerIndex, triggerName, "condition", conditionIndex, "variable"))
			}
		}
		for effectIndex, effect := range trigger.list("effect_data") {
			for _, field := range []string{"legacy_location_object_reference", "location_object_reference"} {
				if id, ok := effect.intValue(field); ok && id >= 0 {
					add(triggerReference("unit", id, triggerIndex, triggerName, "effect", effectIndex, field))
				}
			}
			for selectedIndex, id := range effect.intList("selected_object_ids") {
				if id >= 0 {
					ref := triggerReference("unit", id, triggerIndex, triggerName, "effect", effectIndex, fmt.Sprintf("selected_object_ids[%d]", selectedIndex))
					add(ref)
				}
			}
			if id, ok := effect.intValue("trigger_id"); ok && id >= 0 {
				add(triggerReference("trigger", id, triggerIndex, triggerName, "effect", effectIndex, "trigger_id"))
			}
			if id, ok := effect.intValue("variable"); ok && id >= 0 {
				add(triggerReference("variable", id, triggerIndex, triggerName, "effect", effectIndex, "variable"))
			}
			if id, ok := effect.intValue("variable2"); ok && id >= 0 {
				add(triggerReference("variable", id, triggerIndex, triggerName, "effect", effectIndex, "variable2"))
			}
			if id, ok := effect.intValue("string_id"); ok && id >= 0 {
				add(triggerReference("string", id, triggerIndex, triggerName, "effect", effectIndex, "string_id"))
			}
		}
	}
}

func (f *File) addUnitReferences(add func(ScenarioReference)) {
	units := f.root.section("Units")
	if units == nil {
		return
	}
	for player, playerSection := range units.list("players_units") {
		for unitIndex, unit := range playerSection.list("units") {
			sourcePath := fmt.Sprintf("Units.players_units[%d].units[%d]", player, unitIndex)
			refID, _ := unit.intValue("reference_id")
			if id, ok := unit.intValue("caption_string_id"); ok && id >= 0 {
				add(ScenarioReference{
					Kind:            "string",
					TargetID:        id,
					SourceKind:      "unit",
					SourcePath:      sourcePath,
					Field:           "caption_string_id",
					UnitPlayer:      player,
					UnitIndex:       unitIndex,
					UnitReferenceID: refID,
				})
			}
			if id, ok := unit.intValue("garrisoned_in_id"); ok && id >= 0 {
				add(ScenarioReference{
					Kind:            "unit",
					TargetID:        id,
					SourceKind:      "unit",
					SourcePath:      sourcePath,
					Field:           "garrisoned_in_id",
					UnitPlayer:      player,
					UnitIndex:       unitIndex,
					UnitReferenceID: refID,
				})
			}
		}
	}
}

func triggerReference(kind string, targetID, triggerIndex int, triggerName, childKind string, childIndex int, field string) ScenarioReference {
	ref := ScenarioReference{
		Kind:         kind,
		TargetID:     targetID,
		SourceKind:   "trigger",
		TriggerIndex: triggerIndex,
		TriggerName:  triggerName,
		ChildKind:    childKind,
		ChildIndex:   childIndex,
		Field:        field,
	}
	if childIndex >= 0 {
		ref.SourcePath = fmt.Sprintf("Triggers.trigger_data[%d].%s_data[%d].%s", triggerIndex, childKind, childIndex, field)
	} else {
		ref.SourcePath = fmt.Sprintf("Triggers.trigger_data[%d].%s", triggerIndex, field)
	}
	return ref
}

func (f *File) referenceLabels() map[string]map[int]string {
	return map[string]map[int]string{
		"trigger":  f.triggerReferenceLabels(),
		"unit":     f.unitReferenceLabels(),
		"variable": f.variableReferenceLabels(),
		"string":   f.stringReferenceLabels(),
	}
}

func (f *File) triggerReferenceLabels() map[int]string {
	out := map[int]string{}
	triggers := f.root.section("Triggers")
	if triggers == nil {
		return out
	}
	for i, trigger := range triggers.list("trigger_data") {
		name, _ := trigger.stringValue("trigger_name")
		out[i] = name
	}
	return out
}

func (f *File) unitReferenceLabels() map[int]string {
	out := map[int]string{}
	units := f.root.section("Units")
	if units == nil {
		return out
	}
	for player, playerSection := range units.list("players_units") {
		for unitIndex, unit := range playerSection.list("units") {
			refID, ok := unit.intValue("reference_id")
			if !ok || refID < 0 {
				continue
			}
			unitConst, _ := unit.intValue("unit_const")
			caption, _ := unit.stringValue("caption_string")
			label := fmt.Sprintf("P%d[%d] unit_const=%d", player, unitIndex, unitConst)
			if strings.TrimSpace(caption) != "" {
				label += " " + quoteShort(caption)
			}
			out[refID] = label
		}
	}
	return out
}

func (f *File) variableReferenceLabels() map[int]string {
	out := map[int]string{}
	triggers := f.root.section("Triggers")
	if triggers == nil {
		return out
	}
	for _, variable := range triggers.list("variable_data") {
		id, ok := variable.intValue("variable_id")
		if !ok || id < 0 {
			continue
		}
		name, _ := variable.stringValue("variable_name")
		out[id] = name
	}
	return out
}

func (f *File) stringReferenceLabels() map[int]string {
	out := map[int]string{}
	playerDataTwo := f.root.section("PlayerDataTwo")
	if playerDataTwo == nil {
		return out
	}
	for i, text := range playerDataTwo.stringList("strings") {
		if strings.TrimSpace(text) != "" {
			out[i] = quoteShort(text)
		}
	}
	return out
}

func quoteShort(text string) string {
	text = strings.TrimSpace(text)
	if len(text) > 80 {
		text = text[:77] + "..."
	}
	return fmt.Sprintf("%q", text)
}
