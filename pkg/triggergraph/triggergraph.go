package triggergraph

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"unicode/utf8"
)

type Graph struct {
	SHA256         string           `json:"sha256"`
	TriggerCount   int              `json:"trigger_count"`
	EffectCount    int              `json:"effect_count"`
	ConditionCount int              `json:"condition_count"`
	MessageCount   int              `json:"message_count"`
	Start          int              `json:"start"`
	End            int              `json:"end"`
	Warnings       []string         `json:"warnings,omitempty"`
	Triggers       []map[string]any `json:"triggers,omitempty"`
}

type Region struct {
	Start       int      `json:"start"`
	End         int      `json:"end"`
	StringCount int      `json:"string_count"`
	FirstString string   `json:"first_string,omitempty"`
	LastString  string   `json:"last_string,omitempty"`
	Warnings    []string `json:"warnings,omitempty"`
}

type ScenarioString struct {
	MarkerOffset int    `json:"marker_offset"`
	EndOffset    int    `json:"end_offset"`
	Text         string `json:"text"`
}

type ParseOptions struct {
	Start int
	End   int
	Count int
}

func Parse(blob []byte, opts ParseOptions) (*Graph, error) {
	if opts.Start < 0 || opts.Start > len(blob) {
		return nil, fmt.Errorf("trigger graph start %d outside blob length %d", opts.Start, len(blob))
	}
	end := opts.End
	if end == 0 {
		end = len(blob)
	}
	if end < opts.Start || end > len(blob) {
		return nil, fmt.Errorf("trigger graph end %d outside blob length %d", end, len(blob))
	}
	count := opts.Count
	if count == 0 && opts.Start >= 4 {
		if candidate := readI32(blob, opts.Start-4); candidate > 0 && candidate < 20000 {
			count = int(candidate)
		}
	}
	graph, err := parseTriggerGraph(blob, opts.Start, end, count)
	if err != nil {
		return nil, err
	}
	return graph, nil
}

func FromTriggers(triggers []map[string]any, start, end int, warnings []string) (*Graph, error) {
	triggers = enrichTriggersForGraph(triggers)
	effects := 0
	conditions := 0
	messages := 0
	for _, trigger := range triggers {
		effectList, _ := trigger["effects"].([]map[string]any)
		conditionList, _ := trigger["conditions"].([]map[string]any)
		effects += len(effectList)
		conditions += len(conditionList)
		for _, effect := range effectList {
			if msg, _ := effect["message"].(string); msg != "" {
				messages++
			}
		}
	}
	canonical, err := canonicalJSON(identityTriggers(triggers))
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(canonical)
	return &Graph{
		SHA256:         hex.EncodeToString(sum[:]),
		Triggers:       triggers,
		TriggerCount:   len(triggers),
		EffectCount:    effects,
		ConditionCount: conditions,
		MessageCount:   messages,
		Start:          start,
		End:            end,
		Warnings:       warnings,
	}, nil
}

func enrichTriggersForGraph(triggers []map[string]any) []map[string]any {
	out := make([]map[string]any, 0, len(triggers))
	for _, trigger := range triggers {
		copyTrigger := cloneMap(trigger)
		if effects, ok := copyTrigger["effects"].([]map[string]any); ok {
			enriched := make([]map[string]any, 0, len(effects))
			for _, effect := range effects {
				copyEffect := cloneMap(effect)
				if values, ok := intList(copyEffect["fields_prefix"]); ok {
					enrichEffect(copyEffect, values, nil)
				}
				enriched = append(enriched, copyEffect)
			}
			copyTrigger["effects"] = enriched
		}
		if conditions, ok := copyTrigger["conditions"].([]map[string]any); ok {
			enriched := make([]map[string]any, 0, len(conditions))
			for _, condition := range conditions {
				copyCondition := cloneMap(condition)
				if values, ok := intList(copyCondition["fields"]); ok {
					enrichCondition(copyCondition, values)
				}
				enriched = append(enriched, copyCondition)
			}
			copyTrigger["conditions"] = enriched
		}
		out = append(out, copyTrigger)
	}
	return out
}

func cloneMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func Scan(blob []byte) (*Graph, error) {
	var candidates []*Graph
	limit := len(blob) - 64
	for countOffset := 0; countOffset < limit; countOffset++ {
		count := int(readI32(blob, countOffset))
		if count <= 0 || count >= 20000 {
			continue
		}
		start := countOffset + 4
		if !looksLikeTriggerStart(blob, start) {
			continue
		}
		graph, err := parseTriggerGraph(blob, start, len(blob), count)
		if err != nil {
			continue
		}
		if graph.TriggerCount != count {
			continue
		}
		if graph.EffectCount == 0 && graph.ConditionCount == 0 {
			continue
		}
		candidates = append(candidates, graph)
	}
	if len(candidates) == 0 {
		return nil, fmt.Errorf("could not locate trigger graph by binary structure scan")
	}
	sort.Slice(candidates, func(i, j int) bool {
		scoreI := candidates[i].EffectCount + candidates[i].ConditionCount
		scoreJ := candidates[j].EffectCount + candidates[j].ConditionCount
		if scoreI != scoreJ {
			return scoreI > scoreJ
		}
		if candidates[i].TriggerCount != candidates[j].TriggerCount {
			return candidates[i].TriggerCount > candidates[j].TriggerCount
		}
		return candidates[i].Start < candidates[j].Start
	})
	best := candidates[0]
	best.Warnings = append(best.Warnings, fmt.Sprintf("trigger graph located by binary structure scan among %d candidate(s)", len(candidates)))
	return best, nil
}

func ParseLocated(blob []byte) (*Graph, *Region, error) {
	region, err := FindRegion(blob)
	if err != nil {
		graph, scanErr := Scan(blob)
		if scanErr != nil {
			return nil, nil, fmt.Errorf("%v; fallback scan failed: %w", err, scanErr)
		}
		graph.Warnings = append([]string{fmt.Sprintf("primary trigger region locator failed: %v", err)}, graph.Warnings...)
		return graph, nil, nil
	}
	start, count, err := findTriggerGraphStart(blob, region.Start)
	if err != nil {
		graph, scanErr := Scan(blob)
		if scanErr != nil {
			return nil, region, fmt.Errorf("%v; fallback scan failed: %w", err, scanErr)
		}
		graph.Warnings = append([]string{fmt.Sprintf("primary trigger graph start locator failed: %v", err)}, graph.Warnings...)
		return graph, region, nil
	}
	graph, err := parseTriggerGraph(blob, start, region.End, count)
	if err != nil {
		graph, scanErr := Scan(blob)
		if scanErr != nil {
			return nil, region, fmt.Errorf("%v; fallback scan failed: %w", err, scanErr)
		}
		graph.Warnings = append([]string{fmt.Sprintf("primary trigger region parse failed: %v", err)}, graph.Warnings...)
		return graph, region, nil
	}
	graph.Warnings = append(append([]string{}, region.Warnings...), graph.Warnings...)
	graph.Warnings = append(graph.Warnings, "trigger region located by dense-default/string-cluster heuristic; graph start uses explicit trigger_count")
	return graph, region, nil
}

func FindRegion(blob []byte) (*Region, error) {
	var warnings []string
	if !bytes.HasPrefix(blob, []byte("VER ")) {
		warnings = append(warnings, "decompressed header does not start with VER")
	}
	strings := iterScenarioStrings(blob)
	var meaningful []scenarioString
	for _, s := range strings {
		// Current DE scenario exports can use short trigger strings (for
		// example "Gaia" and "Closed"). They are still structurally useful
		// when the surrounding trigger records provide the bounded graph.
		if len(s.Text) >= 4 {
			meaningful = append(meaningful, s)
		}
	}
	if len(meaningful) == 0 {
		return nil, fmt.Errorf("could not locate VER scenario strings")
	}
	first := meaningful[0]
	for idx, candidate := range meaningful {
		nearby := 0
		for _, s := range meaningful[idx:] {
			if s.MarkerOffset-candidate.MarkerOffset <= 256000 {
				nearby++
			}
			if nearby >= 8 {
				first = candidate
				break
			}
		}
		if nearby >= 8 {
			break
		}
	}
	denseStart := lastDenseFFRunStart(blob, maxInt(0, first.MarkerOffset-512000), first.MarkerOffset)
	start := first.MarkerOffset
	if denseStart < 0 {
		warnings = append(warnings, "trigger start fallback: using first encoded trigger string")
	} else {
		start = denseStart
		for start < first.MarkerOffset && blob[start] == 0 {
			start++
		}
		if start >= first.MarkerOffset {
			start = denseStart
			warnings = append(warnings, "trigger start fallback: dense run had no nonzero lead byte")
		}
	}
	end := len(blob)
	if len(strings) > 0 {
		if chatStart := findReplayChatStart(blob, strings[len(strings)-1].EndOffset); chatStart >= 0 {
			end = chatStart
		} else {
			warnings = append(warnings, "replay chat boundary not found; trigger region extends to header end")
		}
	}
	if start < 0 || start >= end || end > len(blob) {
		return nil, fmt.Errorf("invalid trigger region bounds: %d..%d of %d", start, end, len(blob))
	}
	var regionStrings []scenarioString
	for _, s := range strings {
		if start <= s.MarkerOffset && s.MarkerOffset < end {
			regionStrings = append(regionStrings, s)
		}
	}
	region := &Region{Start: start, End: end, StringCount: len(regionStrings), Warnings: warnings}
	if len(regionStrings) > 0 {
		region.FirstString = regionStrings[0].Text
		region.LastString = regionStrings[len(regionStrings)-1].Text
	}
	return region, nil
}

func ScenarioStrings(blob []byte) []ScenarioString {
	raw := iterScenarioStrings(blob)
	out := make([]ScenarioString, 0, len(raw))
	for _, s := range raw {
		out = append(out, ScenarioString{
			MarkerOffset: s.MarkerOffset,
			EndOffset:    s.EndOffset,
			Text:         s.Text,
		})
	}
	return out
}

func parseTriggerGraph(blob []byte, start, end, expectedCount int) (*Graph, error) {
	offset := start
	var warnings []string
	triggers := make([]map[string]any, 0)
	for offset < end && len(triggers) < 20000 {
		if expectedCount > 0 && len(triggers) >= expectedCount {
			break
		}
		trigger, next, err := parseTrigger(blob, offset)
		if err != nil {
			if len(triggers) == 0 {
				return nil, err
			}
			if looksLikeTriggerDisplayOrder(blob, offset, len(triggers), end) {
				warnings = append(warnings, fmt.Sprintf("trigger parse stopped cleanly before %d-entry trigger display-order array at %d", len(triggers), offset))
			} else {
				warnings = append(warnings, fmt.Sprintf("stopped trigger parse at %d: %v", offset, err))
			}
			break
		}
		triggers = append(triggers, trigger)
		offset = next
	}
	if expectedCount > 0 && len(triggers) != expectedCount {
		return nil, fmt.Errorf("parsed %d triggers, expected %d", len(triggers), expectedCount)
	}
	graph, err := FromTriggers(triggers, start, offset, warnings)
	if err != nil {
		return nil, err
	}
	return graph, nil
}

func parseTrigger(blob []byte, offset int) (map[string]any, int, error) {
	start := offset
	enabled, err := readU32At(blob, offset)
	if err != nil {
		return nil, offset, err
	}
	offset += 4
	looping, err := readByteAt(blob, offset)
	if err != nil {
		return nil, offset, err
	}
	offset++
	executeOnLoad, err := readByteAt(blob, offset)
	if err != nil {
		return nil, offset, err
	}
	offset++
	descriptionSTID, err := readI32At(blob, offset)
	if err != nil {
		return nil, offset, err
	}
	offset += 4
	displayAsObjective, err := readByteAt(blob, offset)
	if err != nil {
		return nil, offset, err
	}
	offset++
	descriptionOrder, err := readU32At(blob, offset)
	if err != nil {
		return nil, offset, err
	}
	offset += 4
	makeHeader, err := readByteAt(blob, offset)
	if err != nil {
		return nil, offset, err
	}
	offset++
	shortDescriptionSTID, err := readI32At(blob, offset)
	if err != nil {
		return nil, offset, err
	}
	offset += 4
	displayOnScreen, err := readByteAt(blob, offset)
	if err != nil {
		return nil, offset, err
	}
	offset++
	offset += 5
	muteObjectives, err := readByteAt(blob, offset)
	if err != nil {
		return nil, offset, err
	}
	offset++
	description, next, err := readTriggerString(blob, offset)
	if err != nil {
		return nil, start, err
	}
	offset = next
	name, next, err := readTriggerString(blob, offset)
	if err != nil {
		return nil, start, err
	}
	offset = next
	shortDescription, next, err := readTriggerString(blob, offset)
	if err != nil {
		return nil, start, err
	}
	offset = next
	effectCountRaw, err := readI32At(blob, offset)
	if err != nil {
		return nil, start, err
	}
	offset += 4
	effectCount := int(effectCountRaw)
	if effectCount < 0 || effectCount >= 10000 {
		return nil, start, fmt.Errorf("unreasonable effect count at %d: %d", start, effectCount)
	}
	effects := make([]map[string]any, 0, effectCount)
	for i := 0; i < effectCount; i++ {
		effect, next, err := parseEffect(blob, offset, i, effectCount)
		if err != nil {
			return nil, start, err
		}
		effects = append(effects, effect)
		offset = next
	}
	effectOrder, err := readI32List(blob, offset, effectCount)
	if err != nil {
		return nil, start, err
	}
	offset += 4 * effectCount
	if !isPermutation(effectOrder, effectCount) {
		return nil, start, fmt.Errorf("effect order mismatch at trigger %d", start)
	}
	conditionCountRaw, err := readI32At(blob, offset)
	if err != nil {
		return nil, start, err
	}
	offset += 4
	conditionCount := int(conditionCountRaw)
	if conditionCount < 0 || conditionCount >= 10000 {
		return nil, start, fmt.Errorf("unreasonable condition count at %d: %d", start, conditionCount)
	}
	conditions := make([]map[string]any, 0, conditionCount)
	for i := 0; i < conditionCount; i++ {
		condition, next, err := parseCondition(blob, offset)
		if err != nil {
			return nil, start, err
		}
		conditions = append(conditions, condition)
		offset = next
	}
	conditionOrder, err := readI32List(blob, offset, conditionCount)
	if err != nil {
		return nil, start, err
	}
	offset += 4 * conditionCount
	if !isPermutation(conditionOrder, conditionCount) {
		return nil, start, fmt.Errorf("condition order mismatch at trigger %d", start)
	}
	return map[string]any{
		"conditions":             conditions,
		"condition_order":        conditionOrder,
		"description":            description,
		"description_order":      int(descriptionOrder),
		"description_stid":       descriptionSTID,
		"display_as_objective":   int(displayAsObjective),
		"display_on_screen":      int(displayOnScreen),
		"effect_order":           effectOrder,
		"effects":                effects,
		"enabled":                int(enabled),
		"execute_on_load":        int(executeOnLoad),
		"looping":                int(looping),
		"make_header":            int(makeHeader),
		"mute_objectives":        int(muteObjectives),
		"name":                   name,
		"short_description":      shortDescription,
		"short_description_stid": shortDescriptionSTID,
	}, offset, nil
}

func parseEffect(blob []byte, offset, effectIndex, effectCount int) (map[string]any, int, error) {
	start := offset
	fieldCount := len(effectFieldNames)
	values, err := readI32List(blob, offset, fieldCount)
	if err != nil {
		return nil, offset, err
	}
	if values[1] != 83 {
		return nil, offset, fmt.Errorf("bad effect static value at %d: %d", offset, values[1])
	}
	offset += fieldCount * 4
	message, next, err := readTriggerString(blob, offset)
	if err != nil {
		return nil, start, fmt.Errorf("effect message: %w", err)
	}
	offset = next
	soundName, next, err := readTriggerString(blob, offset)
	if err != nil {
		return nil, start, fmt.Errorf("effect sound name: %w", err)
	}
	offset = next
	selectedCount := values[6]
	selectedObjectIDs := []int{}
	if selectedCount > 0 {
		if selectedCount > 10000 {
			return nil, start, fmt.Errorf("unreasonable selected object count at %d: %d", start, selectedCount)
		}
		selectedObjectIDs, err = readI32List(blob, offset, selectedCount)
		if err != nil {
			return nil, start, fmt.Errorf("selected object ids: %w", err)
		}
		offset += 4 * selectedCount
	} else if selectedCount < -1 {
		return nil, start, fmt.Errorf("bad selected object count at %d: %d", start, selectedCount)
	}
	messageOption1, next, err := readTriggerString(blob, offset)
	if err != nil {
		return nil, start, fmt.Errorf("effect message option1: %w", err)
	}
	offset = next
	messageOption2, next, err := readTriggerString(blob, offset)
	if err != nil {
		return nil, start, fmt.Errorf("effect message option2: %w", err)
	}
	offset = next
	raw := blob[start:offset]
	sum := sha256.Sum256(raw)
	effect := map[string]any{
		"fields_prefix": values,
		"message":       message,
		"raw_sha256":    hex.EncodeToString(sum[:]),
		"type":          values[0],
	}
	if soundName != "" {
		effect["sound_name"] = soundName
	}
	if selectedCount >= 0 {
		effect["selected_object_ids"] = selectedObjectIDs
	}
	if messageOption1 != "" {
		effect["message_option1"] = messageOption1
	}
	if messageOption2 != "" {
		effect["message_option2"] = messageOption2
	}
	enrichEffect(effect, values, raw)
	return effect, offset, nil
}

func parseCondition(blob []byte, offset int) (map[string]any, int, error) {
	values, err := readI32List(blob, offset, 35)
	if err != nil {
		return nil, offset, err
	}
	// DE 1.59-era replay embeddings use condition marker 34. Scenario files
	// still commonly carry the established 33 marker, and both share the
	// same 35-field condition layout.
	if values[1] != 33 && values[1] != 34 {
		return nil, offset, fmt.Errorf("bad condition static value at %d: %d", offset, values[1])
	}
	xsFunction, next, err := readTriggerString(blob, offset+35*4)
	if err != nil {
		return nil, offset, err
	}
	condition := map[string]any{
		"fields":      values,
		"type":        values[0],
		"xs_function": xsFunction,
	}
	enrichCondition(condition, values)
	return condition, next, nil
}

func canonicalJSON(value any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(value); err != nil {
		return nil, err
	}
	return bytes.TrimSuffix(buf.Bytes(), []byte("\n")), nil
}

type scenarioString struct {
	MarkerOffset int
	EndOffset    int
	Text         string
}

func iterScenarioStrings(blob []byte) []scenarioString {
	var out []scenarioString
	marker := []byte{1, 0, 0, 0}
	pos := 0
	for {
		idx := bytes.Index(blob[pos:], marker)
		if idx < 0 {
			break
		}
		markerOffset := pos + idx
		pos = markerOffset + 1
		if markerOffset+8 > len(blob) {
			continue
		}
		length := int(binary.LittleEndian.Uint32(blob[markerOffset+4:]))
		end := markerOffset + 8 + length
		if length < 1 || length > 4096 || end > len(blob) {
			continue
		}
		if blob[end-1] != 0 {
			continue
		}
		raw := blob[markerOffset+8 : end-1]
		if !isPrintableScenarioText(raw) {
			continue
		}
		out = append(out, scenarioString{MarkerOffset: markerOffset, EndOffset: end, Text: string(raw)})
	}
	return out
}

func isPrintableScenarioText(raw []byte) bool {
	if len(raw) < 4 || !utf8.Valid(raw) {
		return false
	}
	printable := 0
	for _, b := range raw {
		if b == 9 || b == 10 || b == 13 || (b >= 32 && b < 127) {
			printable++
		}
	}
	return float64(printable)/float64(len(raw)) >= 0.90
}

func readTriggerString(blob []byte, offset int) (string, int, error) {
	markerOrLength, err := readU32At(blob, offset)
	if err != nil {
		return "", offset, err
	}
	if markerOrLength == 0 {
		return "", offset + 4, nil
	}
	if markerOrLength == 0xffffffff {
		return "", offset + 4, nil
	}
	if markerOrLength == 1 {
		if offset+5 > len(blob) {
			return "", offset, fmt.Errorf("bad short trigger string at %d", offset)
		}
		if blob[offset+4] == 0 {
			return "", offset + 5, nil
		}
		length, err := readU32At(blob, offset+4)
		if err != nil {
			return "", offset, err
		}
		end := offset + 8 + int(length)
		if length < 1 || length > 4096 || end > len(blob) || blob[end-1] != 0 {
			return "", offset, fmt.Errorf("bad trigger string length at %d: %d", offset, length)
		}
		return string(blob[offset+8 : end-1]), end, nil
	}
	length := int(markerOrLength)
	end := offset + 4 + length
	if length < 1 || length > 4096 || end > len(blob) || blob[end-1] != 0 {
		return "", offset, fmt.Errorf("bad trigger string marker at %d", offset)
	}
	return string(blob[offset+4 : end-1]), end, nil
}

func candidateEffectStart(blob []byte, offset int) bool {
	if offset+8 > len(blob) {
		return false
	}
	effectType := readI32(blob, offset)
	static := readI32(blob, offset+4)
	return effectType >= 0 && effectType < 256 && static == 83
}

func findNextEffectStart(blob []byte, offset, limit int) int {
	end := offset + limit
	if end > len(blob)-8 {
		end = len(blob) - 8
	}
	for candidate := offset; candidate < end; candidate++ {
		if candidateEffectStart(blob, candidate) {
			return candidate
		}
	}
	return -1
}

func looksLikeTriggerStart(blob []byte, offset int) bool {
	pos := offset + 27
	for i := 0; i < 3; i++ {
		_, next, err := readTriggerString(blob, pos)
		if err != nil {
			return false
		}
		pos = next
	}
	effectCount, err := readI32At(blob, pos)
	if err != nil {
		return false
	}
	return effectCount >= 0 && effectCount < 10000 && (effectCount == 0 || candidateEffectStart(blob, pos+4))
}

func findOrderArray(blob []byte, offset, count, limit int) int {
	probeCount := count
	if probeCount > 16 {
		probeCount = 16
	}
	end := offset + limit
	if end > len(blob)-4*probeCount {
		end = len(blob) - 4*probeCount
	}
	for candidate := offset; candidate < end; candidate++ {
		ok := true
		for i := 0; i < probeCount; i++ {
			if readI32(blob, candidate+4*i) != int32(i) {
				ok = false
				break
			}
		}
		if !ok {
			continue
		}
		if count == 1 {
			conditionCountOffset := candidate + 4
			if conditionCountOffset+8 > len(blob) {
				continue
			}
			conditionCount := readI32(blob, conditionCountOffset)
			if conditionCount < 0 || conditionCount >= 10000 {
				continue
			}
			if conditionCount > 0 && readI32(blob, conditionCountOffset+8) != 33 {
				continue
			}
			if conditionCount == 0 && !looksLikeTriggerStart(blob, conditionCountOffset+4) {
				continue
			}
		}
		return candidate
	}
	return -1
}

func looksLikeTriggerDisplayOrder(blob []byte, offset, count, limit int) bool {
	if count <= 0 || count >= 20000 || offset < 0 || offset+4*count > len(blob) || offset+4*count > limit {
		return false
	}
	values := make([]int, count)
	for i := 0; i < count; i++ {
		value := readI32(blob, offset+4*i)
		if value < 0 || int(value) >= count {
			return false
		}
		values[i] = int(value)
	}
	return isPermutation(values, count)
}

func findTriggerGraphStart(blob []byte, regionStart int) (int, int, error) {
	lo := maxInt(0, regionStart-256)
	for start := lo; start <= regionStart; start++ {
		pos := start + 27
		ok := true
		for i := 0; i < 3; i++ {
			_, next, err := readTriggerString(blob, pos)
			if err != nil {
				ok = false
				break
			}
			pos = next
		}
		if !ok {
			continue
		}
		effectCount, err := readI32At(blob, pos)
		if err != nil {
			continue
		}
		if effectCount >= 0 && effectCount < 10000 && candidateEffectStart(blob, pos+4) {
			count := 0
			if start >= 4 {
				candidate := readI32(blob, start-4)
				if candidate > 0 && candidate < 20000 {
					count = int(candidate)
				}
			}
			return start, count, nil
		}
	}
	return 0, 0, fmt.Errorf("could not locate first trigger record")
}

func lastDenseFFRunStart(blob []byte, lo, hi int) int {
	lastStart := -1
	inRun := false
	runStart := lo
	end := hi - 256
	if end < lo {
		end = lo
	}
	for pos := lo; pos < end; pos += 16 {
		dense := bytes.Count(blob[pos:pos+256], []byte{0xff}) >= 180
		if dense && !inRun {
			runStart = pos
			inRun = true
		} else if !dense && inRun {
			lastStart = runStart
			inRun = false
		}
	}
	if inRun {
		lastStart = runStart
	}
	return lastStart
}

func findReplayChatStart(blob []byte, after int) int {
	pos := -1
	for _, needle := range [][]byte{[]byte("@#"), []byte(`{"player"`)} {
		candidate := bytes.Index(blob[after:], needle)
		if candidate < 0 {
			continue
		}
		candidate += after
		if pos < 0 || candidate < pos {
			pos = candidate
		}
	}
	if pos < 0 {
		return -1
	}
	lo := maxInt(after, pos-128)
	if marker := bytes.LastIndex(blob[lo:pos], []byte{1, 0, 0, 0}); marker >= 0 {
		return lo + marker
	}
	return pos
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func isPermutation(values []int, count int) bool {
	if len(values) != count {
		return false
	}
	seen := make([]bool, count)
	for _, value := range values {
		if value < 0 || value >= count || seen[value] {
			return false
		}
		seen[value] = true
	}
	return true
}

func readI32List(blob []byte, offset, count int) ([]int, error) {
	if count < 0 || offset+4*count > len(blob) {
		return nil, fmt.Errorf("need %d int32 values at %d, only %d bytes remain", count, offset, len(blob)-offset)
	}
	out := make([]int, count)
	for i := 0; i < count; i++ {
		out[i] = int(int32(binary.LittleEndian.Uint32(blob[offset+4*i:])))
	}
	return out, nil
}

func readI32At(blob []byte, offset int) (int32, error) {
	if offset+4 > len(blob) {
		return 0, fmt.Errorf("need int32 at %d, only %d bytes remain", offset, len(blob)-offset)
	}
	return int32(binary.LittleEndian.Uint32(blob[offset:])), nil
}

func readU32At(blob []byte, offset int) (uint32, error) {
	if offset+4 > len(blob) {
		return 0, fmt.Errorf("need uint32 at %d, only %d bytes remain", offset, len(blob)-offset)
	}
	return binary.LittleEndian.Uint32(blob[offset:]), nil
}

func readByteAt(blob []byte, offset int) (byte, error) {
	if offset >= len(blob) {
		return 0, fmt.Errorf("need byte at %d, no bytes remain", offset)
	}
	return blob[offset], nil
}

func readI32(blob []byte, offset int) int32 {
	if offset+4 > len(blob) {
		return 0
	}
	return int32(binary.LittleEndian.Uint32(blob[offset:]))
}
