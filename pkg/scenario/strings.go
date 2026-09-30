package scenario

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"
)

var scenarioMarkupTagRE = regexp.MustCompile(`<([^<>]+)>`)

type StringReport struct {
	Path          string             `json:"path,omitempty"`
	Version       string             `json:"version"`
	Verification  string             `json:"verification"`
	Messages      map[string]string  `json:"messages"`
	StringTable   []StringTableEntry `json:"string_table,omitempty"`
	Variables     []VariableEntry    `json:"variables,omitempty"`
	TriggerText   []TriggerTextEntry `json:"trigger_text,omitempty"`
	EffectText    []EffectTextEntry  `json:"effect_text,omitempty"`
	MarkupSummary MarkupSummary      `json:"markup_summary"`
	Warnings      []string           `json:"warnings,omitempty"`
}

type StringTableEntry struct {
	ID     int    `json:"id"`
	Source string `json:"source"`
	Text   string `json:"text"`
}

type VariableEntry struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type TriggerTextEntry struct {
	TriggerIndex int    `json:"trigger_index"`
	TriggerName  string `json:"trigger_name,omitempty"`
	Field        string `json:"field"`
	Text         string `json:"text"`
}

type EffectTextEntry struct {
	TriggerIndex int    `json:"trigger_index"`
	TriggerName  string `json:"trigger_name,omitempty"`
	EffectIndex  int    `json:"effect_index"`
	Type         int    `json:"type"`
	StringID     int    `json:"string_id,omitempty"`
	Text         string `json:"text"`
}

type MarkupSummary struct {
	ColorTags         map[string]int `json:"color_tags"`
	VariableRefs      []string       `json:"variable_refs"`
	VariableRefCounts map[string]int `json:"variable_ref_counts"`
}

func (f *File) Strings() StringReport {
	report := StringReport{
		Path:         f.Path,
		Version:      f.Version,
		Verification: "structure_verified_not_engine_verified",
		Messages:     map[string]string{},
	}
	if f.headerRoot == nil || f.root == nil {
		report.Warnings = append(report.Warnings, "scenario parse tree unavailable")
		return report
	}
	if text, ok := f.headerRoot.stringValue("scenario_instructions"); ok && text != "" {
		report.Messages["scenario_instructions"] = text
	}
	report.addMessageSection(f.root.section("Messages"))
	report.addMessageSection(f.root.section("Cinematics"))
	report.StringTable = scenarioStringTable(f.root)
	report.Variables = scenarioVariables(f.root)
	report.TriggerText = scenarioTriggerText(f.root)
	report.EffectText = scenarioEffectText(f.root)
	report.MarkupSummary = summarizeScenarioMarkup(report.allText())
	return report
}

// StringsFile returns the structured string report when the scenario format is
// fully decoded. For a known-but-not-yet-mapped format, it falls back to a
// deliberately partial printable-string extraction. This keeps text recovery
// useful without presenting a guessed section layout as a successful parse.
func StringsFile(path string) (StringReport, error) {
	f, err := Open(path)
	if err == nil {
		return f.Strings(), nil
	}
	version, body, bodyErr := rawScenarioBody(path)
	if bodyErr != nil || version != "1.59" {
		return StringReport{}, err
	}
	report := StringReport{
		Path:         path,
		Version:      version,
		Verification: "format_partial_read_not_structure_verified",
		Messages:     map[string]string{},
		MarkupSummary: MarkupSummary{
			ColorTags:         map[string]int{},
			VariableRefCounts: map[string]int{},
		},
		Warnings: []string{
			"DE 1.59 structure is not fully byte-mapped; extracted printable strings only",
			"full scenario parse failed: " + err.Error(),
		},
	}
	for i, text := range extractPrintableStrings(body) {
		report.Messages[fmt.Sprintf("raw_string_%d", i)] = text
	}
	report.MarkupSummary = summarizeScenarioMarkup(report.allText())
	return report, nil
}

func rawScenarioBody(path string) (string, []byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", nil, err
	}
	header, _, headerLen, err := DecodeScenarioHeaderNode(data)
	if err != nil {
		return "", nil, fmt.Errorf("decode FileHeader: %w", err)
	}
	body, err := inflateRawLimited(data[headerLen:], maxPartialStringBodyBytes)
	if err != nil {
		return "", nil, fmt.Errorf("inflate scenario body: %w", err)
	}
	return header.Version, body, nil
}

const maxPartialStringBodyBytes = 32 * 1024 * 1024

func extractPrintableStrings(data []byte) []string {
	const minimum = 4
	var out []string
	seen := map[string]bool{}
	start := -1
	flush := func(end int) {
		if start < 0 || end-start < minimum {
			start = -1
			return
		}
		text := strings.TrimSpace(string(data[start:end]))
		if text == "" || !utf8.ValidString(text) || seen[text] {
			start = -1
			return
		}
		seen[text] = true
		out = append(out, text)
		start = -1
	}
	for i, b := range data {
		if b >= 0x20 && b <= 0x7e {
			if start < 0 {
				start = i
			}
			continue
		}
		flush(i)
	}
	flush(len(data))
	return out
}

func (r *StringReport) addMessageSection(section *parsedSection) {
	if section == nil {
		return
	}
	for _, field := range section.Fields {
		text, ok := field.Value.(string)
		if !ok || text == "" {
			continue
		}
		r.Messages[field.Name] = text
	}
}

func scenarioStringTable(root *parsedRoot) []StringTableEntry {
	section := root.section("PlayerDataTwo")
	if section == nil {
		return nil
	}
	values := section.stringList("strings")
	out := make([]StringTableEntry, 0, len(values))
	for i, text := range values {
		if text == "" {
			continue
		}
		out = append(out, StringTableEntry{ID: i, Source: "PlayerDataTwo.strings", Text: text})
	}
	return out
}

func scenarioVariables(root *parsedRoot) []VariableEntry {
	section := root.section("Triggers")
	if section == nil {
		return nil
	}
	var out []VariableEntry
	for _, variable := range section.list("variable_data") {
		id, _ := variable.intValue("variable_id")
		name, _ := variable.stringValue("variable_name")
		if name == "" {
			continue
		}
		out = append(out, VariableEntry{ID: id, Name: name})
	}
	return out
}

func scenarioTriggerText(root *parsedRoot) []TriggerTextEntry {
	section := root.section("Triggers")
	if section == nil {
		return nil
	}
	var out []TriggerTextEntry
	for triggerIndex, trigger := range section.list("trigger_data") {
		triggerName, _ := trigger.stringValue("trigger_name")
		for _, field := range []string{"trigger_description", "short_description"} {
			text, _ := trigger.stringValue(field)
			text = strings.TrimSpace(text)
			if text == "" {
				continue
			}
			out = append(out, TriggerTextEntry{
				TriggerIndex: triggerIndex,
				TriggerName:  triggerName,
				Field:        field,
				Text:         text,
			})
		}
	}
	return out
}

func scenarioEffectText(root *parsedRoot) []EffectTextEntry {
	section := root.section("Triggers")
	if section == nil {
		return nil
	}
	var out []EffectTextEntry
	for triggerIndex, trigger := range section.list("trigger_data") {
		triggerName, _ := trigger.stringValue("trigger_name")
		for effectIndex, effect := range trigger.list("effect_data") {
			message, _ := effect.stringValue("message")
			message = strings.TrimSpace(message)
			if message == "" {
				continue
			}
			effectType, _ := effect.intValue("effect_type")
			stringID, _ := effect.intValue("string_id")
			entry := EffectTextEntry{
				TriggerIndex: triggerIndex,
				TriggerName:  triggerName,
				EffectIndex:  effectIndex,
				Type:         effectType,
				StringID:     stringID,
				Text:         message,
			}
			out = append(out, entry)
		}
	}
	return out
}

func (r StringReport) allText() []string {
	var out []string
	for _, text := range r.Messages {
		out = append(out, text)
	}
	for _, entry := range r.StringTable {
		out = append(out, entry.Text)
	}
	for _, entry := range r.TriggerText {
		out = append(out, entry.Text)
	}
	for _, entry := range r.EffectText {
		out = append(out, entry.Text)
	}
	return out
}

func summarizeScenarioMarkup(texts []string) MarkupSummary {
	colors := map[string]int{}
	variables := map[string]int{}
	for _, text := range texts {
		for _, match := range scenarioMarkupTagRE.FindAllStringSubmatch(text, -1) {
			if len(match) < 2 {
				continue
			}
			tag := strings.TrimSpace(match[1])
			if tag == "" {
				continue
			}
			upper := strings.ToUpper(tag)
			if scenarioColorTag(upper) {
				colors[upper]++
				continue
			}
			if strings.HasPrefix(upper, "VARIABLE ") || strings.HasPrefix(upper, "KILLCOUNT") {
				variables[tag]++
			}
		}
	}
	return MarkupSummary{ColorTags: colors, VariableRefs: sortedStringKeys(variables), VariableRefCounts: variables}
}

func scenarioColorTag(tag string) bool {
	switch tag {
	case "RED", "GREEN", "YELLOW", "BLUE", "AQUA", "PURPLE", "ORANGE", "GREY":
		return true
	default:
		return false
	}
}

func sortedStringKeys(values map[string]int) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
