package scenario

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// ImportTriggersOptions controls a raw-preserving trigger import. Selection
// entries are trigger indexes, inclusive ranges, or exact trigger names.
type ImportTriggersOptions struct {
	Select []string `json:"select,omitempty"`
	Prefix string   `json:"prefix,omitempty"`
	Out    string   `json:"-"`
}

type TriggerImportMapping struct {
	SourceID int    `json:"source_id"`
	TargetID int    `json:"target_id"`
	Name     string `json:"name,omitempty"`
}

type VariableImportMapping struct {
	SourceID int    `json:"source_id"`
	TargetID int    `json:"target_id"`
	Name     string `json:"name"`
	Action   string `json:"action"` // reused or added
}

type ObjectImportMapping struct {
	SourceID int    `json:"source_id"`
	TargetID int    `json:"target_id"`
	Label    string `json:"label,omitempty"`
}

type TriggerImportReport struct {
	Source                 string                  `json:"source"`
	Destination            string                  `json:"destination"`
	Output                 string                  `json:"output,omitempty"`
	Copied                 []TriggerImportMapping  `json:"copied,omitempty"`
	Triggers               []TriggerImportMapping  `json:"trigger_id_map,omitempty"`
	Variables              []VariableImportMapping `json:"variable_id_map,omitempty"`
	Objects                []ObjectImportMapping   `json:"object_id_map,omitempty"`
	Unresolved             []string                `json:"unresolved,omitempty"`
	Verification           string                  `json:"verification"`
	DestinationCountBefore int                     `json:"destination_trigger_count_before"`
	DestinationCountAfter  int                     `json:"destination_trigger_count_after"`
}

// ImportTriggersFile copies selected typed trigger records from source into
// destination. The source bytes are parsed and changed only for references and
// the optional name prefix; destination records and all unrelated sections are
// otherwise left to the normal preservation writer.
func ImportTriggersFile(sourcePath, destinationPath string, opts ImportTriggersOptions) (TriggerImportReport, error) {
	report := TriggerImportReport{Source: sourcePath, Destination: destinationPath, Verification: "structure_verified_not_engine_verified"}
	source, err := Open(sourcePath)
	if err != nil {
		return report, fmt.Errorf("open source: %w", err)
	}
	destination, err := Open(destinationPath)
	if err != nil {
		return report, fmt.Errorf("open destination: %w", err)
	}
	sourceTriggers := source.root.section("Triggers")
	destinationTriggers := destination.root.section("Triggers")
	if sourceTriggers == nil || destinationTriggers == nil {
		return report, fmt.Errorf("source and destination must have typed Triggers sections")
	}
	sourceNodes := sourceTriggers.list("trigger_data")
	destinationNodes := destinationTriggers.list("trigger_data")
	report.DestinationCountBefore = len(destinationNodes)
	selected, err := selectTriggerIndexes(sourceNodes, opts.Select)
	if err != nil {
		return report, err
	}
	triggerMap := make(map[int]int, len(selected))
	for i, sourceID := range selected {
		triggerMap[sourceID] = len(destinationNodes) + i
	}

	variableMap, err := importVariables(source, destination, selected, sourceNodes, reportPtrVariables(&report))
	if err != nil {
		return report, err
	}
	objectMap, unresolvedObjects := buildObjectMap(source, destination, selected, sourceNodes)
	for _, mapping := range objectMap {
		report.Objects = append(report.Objects, mapping)
	}
	sort.Slice(report.Objects, func(i, j int) bool { return report.Objects[i].SourceID < report.Objects[j].SourceID })
	report.Unresolved = append(report.Unresolved, unresolvedObjects...)

	for _, sourceID := range selected {
		raw := sourceNodes[sourceID].raw()
		spec, err := destination.writeSpec()
		if err != nil {
			return report, err
		}
		triggerSpec, _, err := triggerEffectSpecs(spec)
		if err != nil {
			return report, err
		}
		p := parser{data: raw, sections: map[string]*parsedSection{}, version: destination.Version}
		node, err := p.parseNode("TriggerStruct", triggerSpec, nil)
		if err != nil || p.off != len(raw) {
			return report, fmt.Errorf("source trigger %d is not compatible with destination trigger layout", sourceID)
		}
		if err := remapImportedTrigger(node, sourceID, triggerMap, variableMap, objectMap, &report); err != nil {
			return report, err
		}
		if opts.Prefix != "" {
			name, _ := node.stringValue("trigger_name")
			if err := setStringField(node, "trigger_name", "str32", opts.Prefix+name); err != nil {
				return report, err
			}
		}
		if err := destination.addRawTrigger(node.raw()); err != nil {
			return report, fmt.Errorf("append source trigger %d: %w", sourceID, err)
		}
		name, _ := node.stringValue("trigger_name")
		mapping := TriggerImportMapping{SourceID: sourceID, TargetID: triggerMap[sourceID], Name: name}
		report.Copied = append(report.Copied, mapping)
	}
	report.Triggers = append([]TriggerImportMapping(nil), report.Copied...)
	report.DestinationCountAfter = len(destination.root.section("Triggers").list("trigger_data"))
	if opts.Out == "" {
		return report, fmt.Errorf("import requires an output path (use --out)")
	}
	if err := destination.Write(opts.Out); err != nil {
		return report, fmt.Errorf("write output: %w", err)
	}
	report.Output = opts.Out
	return report, nil
}

func reportPtrVariables(report *TriggerImportReport) *[]VariableImportMapping {
	return &report.Variables
}

func selectTriggerIndexes(nodes []*parsedNode, selectors []string) ([]int, error) {
	if len(selectors) == 0 {
		out := make([]int, len(nodes))
		for i := range nodes {
			out[i] = i
		}
		return out, nil
	}
	byName := map[string]int{}
	for i, node := range nodes {
		name, _ := node.stringValue("trigger_name")
		byName[name] = i
	}
	seen := map[int]bool{}
	for _, selector := range selectors {
		for _, part := range strings.Split(selector, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			if strings.Contains(part, "-") {
				bounds := strings.Split(part, "-")
				if len(bounds) != 2 {
					return nil, fmt.Errorf("invalid trigger range %q", part)
				}
				a, e1 := strconv.Atoi(strings.TrimSpace(bounds[0]))
				b, e2 := strconv.Atoi(strings.TrimSpace(bounds[1]))
				if e1 != nil || e2 != nil || a < 0 || b < a || b >= len(nodes) {
					return nil, fmt.Errorf("invalid trigger range %q", part)
				}
				for i := a; i <= b; i++ {
					seen[i] = true
				}
				continue
			}
			if i, err := strconv.Atoi(part); err == nil {
				if i < 0 || i >= len(nodes) {
					return nil, fmt.Errorf("trigger index %d out of range", i)
				}
				seen[i] = true
				continue
			}
			i, ok := byName[part]
			if !ok {
				return nil, fmt.Errorf("trigger name %q not found", part)
			}
			seen[i] = true
		}
	}
	out := make([]int, 0, len(seen))
	for i := range seen {
		out = append(out, i)
	}
	sort.Ints(out)
	return out, nil
}

func importVariables(source, destination *File, selected []int, nodes []*parsedNode, report *[]VariableImportMapping) (map[int]int, error) {
	ids := map[int]bool{}
	for _, i := range selected {
		walkTriggerReferenceInts(nodes[i], func(kind, field string, value int) {
			if kind == "variable" && value >= 0 {
				ids[value] = true
			}
		})
	}
	out := map[int]int{}
	for id := range ids {
		node, err := source.findVariableByID(id)
		if err != nil {
			return nil, fmt.Errorf("source variable %d is referenced but not defined", id)
		}
		name, _ := node.stringValue("variable_name")
		if existing, err := destination.findVariableByName(name); err == nil {
			target, _ := existing.intValue("variable_id")
			out[id] = target
			*report = append(*report, VariableImportMapping{SourceID: id, TargetID: target, Name: name, Action: "reused"})
			continue
		}
		if err := destination.AddVariable(VariableRecipe{Name: name}); err != nil {
			return nil, err
		}
		existing, err := destination.findVariableByName(name)
		if err != nil {
			return nil, err
		}
		target, _ := existing.intValue("variable_id")
		out[id] = target
		*report = append(*report, VariableImportMapping{SourceID: id, TargetID: target, Name: name, Action: "added"})
	}
	return out, nil
}

func remapImportedTrigger(node *parsedNode, sourceID int, triggers, variables map[int]int, objects map[int]ObjectImportMapping, report *TriggerImportReport) error {
	remap := func(child *parsedNode, kind, field string, value int) {
		if value < 0 {
			return
		}
		var target int
		var ok bool
		switch kind {
		case "trigger":
			target, ok = triggers[value]
		case "variable":
			target, ok = variables[value]
		case "unit":
			m := objects[value]
			target, ok = m.TargetID, m.TargetID >= 0
		}
		if !ok {
			report.Unresolved = append(report.Unresolved, fmt.Sprintf("trigger %d %s=%d", sourceID, field, value))
			return
		}
		if err := setIntField(child, field, "s32", target); err != nil {
			report.Unresolved = append(report.Unresolved, fmt.Sprintf("trigger %d %s=%d: %v", sourceID, field, value, err))
		}
	}
	for _, c := range node.list("condition_data") {
		for _, field := range []string{"unit_object", "next_object"} {
			if v, ok := c.intValue(field); ok {
				remap(c, "unit", field, v)
			}
		}
		if v, ok := c.intValue("trigger_id"); ok {
			remap(c, "trigger", "trigger_id", v)
		}
		if v, ok := c.intValue("variable"); ok {
			remap(c, "variable", "variable", v)
		}
	}
	for _, e := range node.list("effect_data") {
		for _, field := range []string{"legacy_location_object_reference", "location_object_reference"} {
			if v, ok := e.intValue(field); ok {
				remap(e, "unit", field, v)
			}
		}
		values := e.intList("selected_object_ids")
		if len(values) > 0 {
			mapped := make([]int, len(values))
			for i, v := range values {
				m, ok := objects[v]
				if !ok || v < 0 {
					if v >= 0 {
						report.Unresolved = append(report.Unresolved, fmt.Sprintf("trigger %d selected_object_ids[%d]=%d", sourceID, i, v))
					}
					mapped[i] = v
				} else {
					mapped[i] = m.TargetID
				}
			}
			if err := setIntListField(e, "selected_object_ids", "s32", mapped); err != nil {
				report.Unresolved = append(report.Unresolved, fmt.Sprintf("trigger %d selected_object_ids: %v", sourceID, err))
			}
		}
		if v, ok := e.intValue("trigger_id"); ok {
			remap(e, "trigger", "trigger_id", v)
		}
		for _, field := range []string{"variable", "variable2"} {
			if v, ok := e.intValue(field); ok {
				remap(e, "variable", field, v)
			}
		}
	}
	if len(report.Unresolved) > 0 {
		return fmt.Errorf("import refused: %d unresolved references", len(report.Unresolved))
	}
	return nil
}

func walkTriggerReferenceInts(trigger *parsedNode, fn func(kind, field string, value int)) {
	for _, c := range trigger.list("condition_data") {
		for _, field := range []string{"unit_object", "next_object"} {
			if v, ok := c.intValue(field); ok {
				fn("unit", field, v)
			}
		}
		if v, ok := c.intValue("trigger_id"); ok {
			fn("trigger", "trigger_id", v)
		}
		if v, ok := c.intValue("variable"); ok {
			fn("variable", "variable", v)
		}
	}
	for _, e := range trigger.list("effect_data") {
		for _, field := range []string{"legacy_location_object_reference", "location_object_reference"} {
			if v, ok := e.intValue(field); ok {
				fn("unit", field, v)
			}
		}
		for _, v := range e.intList("selected_object_ids") {
			fn("unit", "selected_object_ids", v)
		}
		if v, ok := e.intValue("trigger_id"); ok {
			fn("trigger", "trigger_id", v)
		}
		for _, field := range []string{"variable", "variable2"} {
			if v, ok := e.intValue(field); ok {
				fn("variable", field, v)
			}
		}
	}
}

func buildObjectMap(source, destination *File, selected []int, nodes []*parsedNode) (map[int]ObjectImportMapping, []string) {
	ids := map[int]bool{}
	for _, i := range selected {
		walkTriggerReferenceInts(nodes[i], func(kind, field string, value int) {
			if kind == "unit" && value >= 0 {
				ids[value] = true
			}
		})
	}
	src := unitSignatures(source)
	dst := unitSignatures(destination)
	out := map[int]ObjectImportMapping{}
	unresolved := []string{}
	for id := range ids {
		sig, ok := src[id]
		if !ok {
			unresolved = append(unresolved, fmt.Sprintf("unit reference %d has no source unit", id))
			continue
		}
		matches := []int{}
		for target, candidate := range dst {
			if candidate == sig {
				matches = append(matches, target)
			}
		}
		if len(matches) != 1 {
			unresolved = append(unresolved, fmt.Sprintf("unit reference %d has %d exact destination matches", id, len(matches)))
			continue
		}
		out[id] = ObjectImportMapping{SourceID: id, TargetID: matches[0], Label: sig}
	}
	return out, unresolved
}

func unitSignatures(f *File) map[int]string {
	out := map[int]string{}
	if f.Units == nil {
		return out
	}
	for _, ps := range f.root.section("Units").list("players_units") {
		for _, u := range ps.list("units") {
			id, ok := u.intValue("reference_id")
			if !ok || id < 0 {
				continue
			}
			p, _ := u.intValue("unit_const")
			x, _ := u.floatValue("x")
			y, _ := u.floatValue("y")
			z, _ := u.floatValue("z")
			player := 0
			_ = player
			out[id] = fmt.Sprintf("%d/%g/%g/%g", p, x, y, z)
		}
	}
	return out
}
