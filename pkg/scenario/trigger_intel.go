package scenario

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

type triggerRow struct {
	Index            int
	Name             string
	Description      string
	ShortDescription string
	Enabled          uint32
	Looping          int8
	Effects          []EffectSummary
	EffectNodes      []*parsedNode
	Conditions       []map[string]any
	ConditionNodes   []*parsedNode
}

type TriggerNeighborhoodOptions struct {
	TriggerIndex     *int `json:"trigger_index,omitempty"`
	UnitRef          *int `json:"unit_ref,omitempty"`
	Variable         *int `json:"variable,omitempty"`
	Depth            int  `json:"depth,omitempty"`
	Limit            int  `json:"limit,omitempty"`
	MaxInflatedBytes int  `json:"max_inflated_bytes,omitempty"`
}

type TriggerNeighborhoodReport struct {
	Path         string                     `json:"path,omitempty"`
	Version      string                     `json:"version"`
	Verification string                     `json:"verification"`
	Query        TriggerNeighborhoodOptions `json:"query"`
	RootTriggers []int                      `json:"root_triggers,omitempty"`
	Nodes        []TriggerNeighborhoodNode  `json:"nodes,omitempty"`
	Edges        []TriggerNeighborhoodEdge  `json:"edges,omitempty"`
	Warnings     []string                   `json:"warnings,omitempty"`
}

type TriggerNeighborhoodNode struct {
	Index      int      `json:"index"`
	Name       string   `json:"name,omitempty"`
	Depth      int      `json:"depth"`
	Enabled    uint32   `json:"enabled"`
	Looping    int8     `json:"looping"`
	Effects    int      `json:"effects"`
	Conditions int      `json:"conditions"`
	Reasons    []string `json:"reasons,omitempty"`
}

type TriggerNeighborhoodEdge struct {
	From        int    `json:"from"`
	To          int    `json:"to"`
	Kind        string `json:"kind"`
	SourceChild string `json:"source_child,omitempty"`
}

type PlayerCoverageOptions struct {
	Players          []int `json:"players,omitempty"`
	MaxInflatedBytes int   `json:"max_inflated_bytes,omitempty"`
}

type PlayerCoverageReport struct {
	Path         string                 `json:"path,omitempty"`
	Version      string                 `json:"version"`
	Verification string                 `json:"verification"`
	Players      []int                  `json:"players"`
	Summary      PlayerCoverageSummary  `json:"summary"`
	Families     []PlayerCoverageFamily `json:"families,omitempty"`
	Issues       []PlayerCoverageIssue  `json:"issues,omitempty"`
}

type PlayerCoverageSummary struct {
	TriggerFamilies int `json:"trigger_families"`
	Issues          int `json:"issues"`
}

type PlayerCoverageFamily struct {
	Key      string         `json:"key"`
	Players  []int          `json:"players"`
	Triggers []TriggerBrief `json:"triggers"`
	Missing  []int          `json:"missing,omitempty"`
}

type PlayerCoverageIssue struct {
	Severity string `json:"severity"`
	Code     string `json:"code"`
	Message  string `json:"message"`
	Family   string `json:"family,omitempty"`
}

type TriggerBrief struct {
	Index   int    `json:"index"`
	Name    string `json:"name,omitempty"`
	Enabled uint32 `json:"enabled"`
	Looping int8   `json:"looping"`
}

type MechanicOptions struct {
	Kind             string `json:"kind"`
	Grep             string `json:"grep,omitempty"`
	MaxInflatedBytes int    `json:"max_inflated_bytes,omitempty"`
}

type MechanicReport struct {
	Path         string            `json:"path,omitempty"`
	Version      string            `json:"version"`
	Verification string            `json:"verification"`
	Kind         string            `json:"kind"`
	Summary      MechanicSummary   `json:"summary"`
	Triggers     []MechanicTrigger `json:"triggers,omitempty"`
	Warnings     []string          `json:"warnings,omitempty"`
}

type MechanicSummary struct {
	TriggerCount   int            `json:"trigger_count"`
	EffectTypes    map[string]int `json:"effect_types,omitempty"`
	ConditionTypes map[string]int `json:"condition_types,omitempty"`
	Players        []int          `json:"players,omitempty"`
}

type MechanicTrigger struct {
	Index      int      `json:"index"`
	Name       string   `json:"name,omitempty"`
	Role       string   `json:"role"`
	Players    []int    `json:"players,omitempty"`
	Effects    []string `json:"effects,omitempty"`
	Conditions []string `json:"conditions,omitempty"`
	Notes      []string `json:"notes,omitempty"`
}

type TriggerFlowOptions struct {
	Grep             string `json:"grep,omitempty"`
	UnitRef          *int   `json:"unit_ref,omitempty"`
	Variable         *int   `json:"variable,omitempty"`
	Limit            int    `json:"limit,omitempty"`
	MaxInflatedBytes int    `json:"max_inflated_bytes,omitempty"`
}

type TriggerFlowReport struct {
	Path         string             `json:"path,omitempty"`
	Version      string             `json:"version"`
	Verification string             `json:"verification"`
	Query        TriggerFlowOptions `json:"query"`
	Stages       []TriggerFlowStage `json:"stages,omitempty"`
	Warnings     []string           `json:"warnings,omitempty"`
}

type TriggerFlowStage struct {
	Stage    string            `json:"stage"`
	Triggers []MechanicTrigger `json:"triggers,omitempty"`
}

func collectTriggerRows(f *File) []triggerRow {
	if f == nil || f.root == nil {
		return nil
	}
	section := f.root.section("Triggers")
	if section == nil {
		return nil
	}
	nodes := section.list("trigger_data")
	out := make([]triggerRow, 0, len(nodes))
	for i, trigger := range nodes {
		name, _ := trigger.stringValue("trigger_name")
		description, _ := trigger.stringValue("trigger_description")
		shortDescription, _ := trigger.stringValue("short_description")
		enabled, _ := trigger.uint32Value("enabled")
		looping, _ := trigger.int8Value("looping")
		row := triggerRow{
			Index:            i,
			Name:             name,
			Description:      description,
			ShortDescription: shortDescription,
			Enabled:          enabled,
			Looping:          looping,
		}
		for j, effect := range trigger.list("effect_data") {
			summary := summarizeEffect(effect, EffectsOptions{})
			summary.TriggerIndex = i
			summary.TriggerName = name
			summary.EffectIndex = j
			row.Effects = append(row.Effects, summary)
			row.EffectNodes = append(row.EffectNodes, effect)
		}
		for _, condition := range trigger.list("condition_data") {
			row.Conditions = append(row.Conditions, summarizeCondition(condition))
			row.ConditionNodes = append(row.ConditionNodes, condition)
		}
		out = append(out, row)
	}
	return out
}

func (row triggerRow) searchMatches(opts TriggerSearchOptions) []TriggerSearchMatch {
	var matches []TriggerSearchMatch
	if !hasTriggerSearchCriteria(opts) {
		return []TriggerSearchMatch{{
			TriggerIndex:     row.Index,
			TriggerName:      row.Name,
			Description:      row.Description,
			ShortDescription: row.ShortDescription,
			Enabled:          row.Enabled,
			Looping:          row.Looping,
			EffectCount:      len(row.Effects),
			ConditionCount:   len(row.Conditions),
		}}
	}
	metaFields := row.metaMatchFields(strings.ToLower(opts.Grep))
	if len(metaFields) > 0 && !opts.hasChildSpecificCriteria() {
		matches = append(matches, TriggerSearchMatch{
			TriggerIndex:   row.Index,
			TriggerName:    row.Name,
			Enabled:        row.Enabled,
			Looping:        row.Looping,
			EffectCount:    len(row.Effects),
			ConditionCount: len(row.Conditions),
			MatchedFields:  metaFields,
		})
	}
	for i, effect := range row.Effects {
		fields, ok := row.matchEffect(i, effect, opts, metaFields)
		if !ok {
			continue
		}
		idx := i
		matches = append(matches, TriggerSearchMatch{
			TriggerIndex:   row.Index,
			TriggerName:    row.Name,
			Enabled:        row.Enabled,
			Looping:        row.Looping,
			EffectCount:    len(row.Effects),
			ConditionCount: len(row.Conditions),
			EffectIndex:    &idx,
			EffectType:     effect.Type,
			EffectTypeName: effect.TypeName,
			MatchedFields:  fields,
		})
	}
	for i, condition := range row.Conditions {
		fields, ok := row.matchCondition(i, condition, opts, metaFields)
		if !ok {
			continue
		}
		idx := i
		condType := intFromAny(condition["type"])
		matches = append(matches, TriggerSearchMatch{
			TriggerIndex:   row.Index,
			TriggerName:    row.Name,
			Enabled:        row.Enabled,
			Looping:        row.Looping,
			EffectCount:    len(row.Effects),
			ConditionCount: len(row.Conditions),
			ConditionIndex: &idx,
			ConditionType:  condType,
			ConditionName:  ConditionTypeName(condType),
			MatchedFields:  fields,
		})
	}
	return matches
}

func (opts TriggerSearchOptions) hasChildSpecificCriteria() bool {
	return strings.TrimSpace(opts.EffectQuery) != "" ||
		strings.TrimSpace(opts.ConditionQuery) != "" ||
		strings.TrimSpace(opts.Message) != "" ||
		opts.UnitRef != nil ||
		opts.UnitType != nil ||
		opts.Player != nil ||
		opts.Variable != nil ||
		len(opts.Area) == 4
}

func (row triggerRow) metaMatchFields(grep string) map[string]string {
	fields := map[string]string{}
	if grep == "" {
		return fields
	}
	addContains(fields, "trigger_name", row.Name, grep)
	addContains(fields, "trigger_description", row.Description, grep)
	addContains(fields, "short_description", row.ShortDescription, grep)
	return fields
}

func (row triggerRow) matchEffect(index int, effect EffectSummary, opts TriggerSearchOptions, metaFields map[string]string) (map[string]string, bool) {
	fields := copyStringMap(metaFields)
	if opts.ConditionQuery != "" {
		return nil, false
	}
	grep := strings.ToLower(opts.Grep)
	if grep != "" {
		addContains(fields, "effect_type", effect.TypeName, grep)
		addContains(fields, "effect_message", effect.Text, grep)
		for key, value := range effect.KnownFields {
			addContains(fields, "effect_"+key, fmt.Sprint(value), grep)
		}
		if !hasAnyPrefixed(fields, "trigger_", "effect_") {
			return nil, false
		}
	}
	if opts.EffectQuery != "" {
		if !effectTypeMatchesQuery(effect.Type, opts.EffectQuery) {
			return nil, false
		}
		fields["effect_type"] = effect.TypeName
	}
	if opts.Message != "" {
		if !strings.Contains(strings.ToLower(effect.Text), strings.ToLower(opts.Message)) {
			return nil, false
		}
		fields["effect_message"] = effect.Text
	}
	if opts.UnitRef != nil {
		if !effectTouchesUnitRef(row.EffectNodes[index], effect, *opts.UnitRef) {
			return nil, false
		}
		fields["unit_ref"] = strconv.Itoa(*opts.UnitRef)
	}
	if opts.UnitType != nil {
		if effect.UnitConst != *opts.UnitType {
			return nil, false
		}
		fields["unit_type"] = strconv.Itoa(*opts.UnitType)
	}
	if opts.Player != nil {
		if !effectTouchesPlayer(row.EffectNodes[index], effect, *opts.Player) {
			return nil, false
		}
		fields["player"] = strconv.Itoa(*opts.Player)
	}
	if opts.Variable != nil {
		if effect.Variable != *opts.Variable && effect.Variable2 != *opts.Variable {
			return nil, false
		}
		fields["variable"] = strconv.Itoa(*opts.Variable)
	}
	if len(opts.Area) == 4 {
		if !effectTouchesArea(effect, opts.Area) {
			return nil, false
		}
		fields["area"] = fmt.Sprint(opts.Area)
	}
	return fields, len(fields) > 0
}

func (row triggerRow) matchCondition(index int, condition map[string]any, opts TriggerSearchOptions, metaFields map[string]string) (map[string]string, bool) {
	fields := copyStringMap(metaFields)
	if opts.EffectQuery != "" || opts.Message != "" {
		return nil, false
	}
	condType := intFromAny(condition["type"])
	condName := ConditionTypeName(condType)
	grep := strings.ToLower(opts.Grep)
	if grep != "" {
		addContains(fields, "condition_type", condName, grep)
		for key, value := range condition {
			if key == "type" || key == "type_name" {
				continue
			}
			addContains(fields, "condition_"+key, fmt.Sprint(value), grep)
		}
		if !hasAnyPrefixed(fields, "trigger_", "condition_") {
			return nil, false
		}
	}
	if opts.ConditionQuery != "" {
		if !conditionTypeMatchesQuery(condType, opts.ConditionQuery) {
			return nil, false
		}
		fields["condition_type"] = condName
	}
	if opts.UnitRef != nil {
		if intFromAny(condition["unit_object"]) != *opts.UnitRef && intFromAny(condition["next_object"]) != *opts.UnitRef {
			return nil, false
		}
		fields["unit_ref"] = strconv.Itoa(*opts.UnitRef)
	}
	if opts.UnitType != nil {
		if intFromAny(condition["object_list"]) != *opts.UnitType {
			return nil, false
		}
		fields["unit_type"] = strconv.Itoa(*opts.UnitType)
	}
	if opts.Player != nil {
		if intFromAny(condition["source_player"]) != *opts.Player && intFromAny(condition["target_player"]) != *opts.Player {
			return nil, false
		}
		fields["player"] = strconv.Itoa(*opts.Player)
	}
	if opts.Variable != nil {
		if intFromAny(condition["variable"]) != *opts.Variable && intFromAny(condition["variable2"]) != *opts.Variable {
			return nil, false
		}
		fields["variable"] = strconv.Itoa(*opts.Variable)
	}
	if len(opts.Area) == 4 {
		if !conditionTouchesArea(condition, opts.Area) {
			return nil, false
		}
		fields["area"] = fmt.Sprint(opts.Area)
	}
	return fields, len(fields) > 0
}

func TriggerNeighborhoodFile(path string, opts TriggerNeighborhoodOptions) (TriggerNeighborhoodReport, error) {
	if opts.Depth <= 0 {
		opts.Depth = 2
	}
	if opts.Limit <= 0 {
		opts.Limit = 200
	}
	if opts.TriggerIndex == nil && opts.UnitRef == nil && opts.Variable == nil {
		return TriggerNeighborhoodReport{}, fmt.Errorf("one of --trigger, --unit-ref, or --variable is required")
	}
	file, err := OpenWithOptions(path, ParseOptions{MaxInflatedBytes: opts.MaxInflatedBytes})
	if err != nil {
		return TriggerNeighborhoodReport{}, err
	}
	rows := collectTriggerRows(file)
	report := TriggerNeighborhoodReport{
		Path:         path,
		Version:      file.Version,
		Verification: "structure_verified_not_engine_verified",
		Query:        opts,
	}
	rootReasons := map[int][]string{}
	if opts.TriggerIndex != nil {
		if *opts.TriggerIndex < 0 || *opts.TriggerIndex >= len(rows) {
			return TriggerNeighborhoodReport{}, fmt.Errorf("trigger index %d outside 0..%d", *opts.TriggerIndex, len(rows)-1)
		}
		rootReasons[*opts.TriggerIndex] = append(rootReasons[*opts.TriggerIndex], "requested trigger")
	}
	for _, row := range rows {
		if opts.UnitRef != nil && rowTouchesUnitRef(row, *opts.UnitRef) {
			rootReasons[row.Index] = append(rootReasons[row.Index], fmt.Sprintf("touches unit_ref %d", *opts.UnitRef))
		}
		if opts.Variable != nil && rowTouchesVariable(row, *opts.Variable) {
			rootReasons[row.Index] = append(rootReasons[row.Index], fmt.Sprintf("touches variable %d", *opts.Variable))
		}
	}
	if len(rootReasons) == 0 {
		report.Warnings = append(report.Warnings, "no root triggers matched")
		return report, nil
	}
	graph := triggerEdgeMap(rows)
	seen := map[int]int{}
	queue := make([]int, 0, len(rootReasons))
	for _, root := range sortedIntMapKeys(rootReasons) {
		queue = append(queue, root)
		seen[root] = 0
		report.RootTriggers = append(report.RootTriggers, root)
	}
	for len(queue) > 0 && len(seen) <= opts.Limit {
		cur := queue[0]
		queue = queue[1:]
		depth := seen[cur]
		if depth >= opts.Depth {
			continue
		}
		for _, edge := range graph[cur] {
			report.Edges = append(report.Edges, edge)
			if _, ok := seen[edge.To]; ok {
				continue
			}
			if len(seen) >= opts.Limit {
				report.Warnings = append(report.Warnings, fmt.Sprintf("node limit %d reached", opts.Limit))
				break
			}
			seen[edge.To] = depth + 1
			queue = append(queue, edge.To)
		}
	}
	for _, index := range sortedSeenByDepth(seen) {
		row := rows[index]
		report.Nodes = append(report.Nodes, TriggerNeighborhoodNode{
			Index:      row.Index,
			Name:       row.Name,
			Depth:      seen[index],
			Enabled:    row.Enabled,
			Looping:    row.Looping,
			Effects:    len(row.Effects),
			Conditions: len(row.Conditions),
			Reasons:    rootReasons[index],
		})
	}
	return report, nil
}

func PlayerCoverageFile(path string, opts PlayerCoverageOptions) (PlayerCoverageReport, error) {
	file, err := OpenWithOptions(path, ParseOptions{MaxInflatedBytes: opts.MaxInflatedBytes})
	if err != nil {
		return PlayerCoverageReport{}, err
	}
	rows := collectTriggerRows(file)
	players := opts.Players
	if len(players) == 0 {
		for _, player := range file.Players {
			if player.Player > 0 && player.Active {
				players = append(players, player.Player)
			}
		}
		if len(players) == 0 {
			players = []int{1, 2, 3, 4}
		}
	}
	sort.Ints(players)
	report := PlayerCoverageReport{
		Path:         path,
		Version:      file.Version,
		Verification: "structure_verified_not_engine_verified",
		Players:      players,
	}
	families := map[string]*PlayerCoverageFamily{}
	for _, row := range rows {
		key, namedPlayer := playerFamilyKey(row.Name)
		touched := rowTouchedPlayers(row)
		if namedPlayer > 0 {
			touched[namedPlayer] = true
		}
		if key == "" || len(touched) == 0 {
			continue
		}
		family := families[key]
		if family == nil {
			family = &PlayerCoverageFamily{Key: key}
			families[key] = family
		}
		for player := range touched {
			if containsInt(players, player) {
				family.Players = appendUniqueInt(family.Players, player)
			}
		}
		family.Triggers = append(family.Triggers, TriggerBrief{Index: row.Index, Name: row.Name, Enabled: row.Enabled, Looping: row.Looping})
	}
	for _, key := range sortedCoverageKeys(families) {
		family := families[key]
		sort.Ints(family.Players)
		for _, player := range players {
			if !containsInt(family.Players, player) && len(family.Players) > 0 {
				family.Missing = append(family.Missing, player)
			}
		}
		if len(family.Missing) > 0 && len(family.Players) > 0 {
			report.Issues = append(report.Issues, PlayerCoverageIssue{
				Severity: "warning",
				Code:     "player_family_asymmetry",
				Family:   family.Key,
				Message:  fmt.Sprintf("trigger family %q touches players %v but is missing %v", family.Key, family.Players, family.Missing),
			})
		}
		report.Families = append(report.Families, *family)
	}
	report.Summary.TriggerFamilies = len(report.Families)
	report.Summary.Issues = len(report.Issues)
	return report, nil
}

func MechanicFile(path string, opts MechanicOptions) (MechanicReport, error) {
	if opts.Kind == "" {
		return MechanicReport{}, fmt.Errorf("--kind is required")
	}
	file, err := OpenWithOptions(path, ParseOptions{MaxInflatedBytes: opts.MaxInflatedBytes})
	if err != nil {
		return MechanicReport{}, err
	}
	kind := normalizeMechanicKind(opts.Kind)
	if kind == "" {
		return MechanicReport{}, fmt.Errorf("unknown mechanic kind %q; expected garrison-token, transform-toggle, teleport-transition, or refresh-cycle", opts.Kind)
	}
	report := MechanicReport{
		Path:         path,
		Version:      file.Version,
		Verification: "structure_verified_not_engine_verified",
		Kind:         kind,
		Summary: MechanicSummary{
			EffectTypes:    map[string]int{},
			ConditionTypes: map[string]int{},
		},
	}
	grep := strings.ToLower(opts.Grep)
	playerSet := map[int]bool{}
	for _, row := range collectTriggerRows(file) {
		if grep != "" && !rowTextContains(row, grep) {
			continue
		}
		role := mechanicRole(row, kind)
		if role == "" {
			continue
		}
		mt := mechanicTrigger(row, role)
		report.Triggers = append(report.Triggers, mt)
		for _, player := range mt.Players {
			playerSet[player] = true
		}
		for _, name := range mt.Effects {
			report.Summary.EffectTypes[name]++
		}
		for _, name := range mt.Conditions {
			report.Summary.ConditionTypes[name]++
		}
	}
	report.Summary.TriggerCount = len(report.Triggers)
	report.Summary.Players = sortedBoolKeys(playerSet)
	if len(report.Triggers) == 0 {
		report.Warnings = append(report.Warnings, "no matching mechanic triggers found")
	}
	if kind == "garrison-token" {
		if len(report.Summary.Players) > 0 {
			report.Warnings = append(report.Warnings, "audit per-player refresh symmetry with kit scen audit-player-coverage when a token works for some slots but not others")
		}
	}
	return report, nil
}

func TriggerFlowFile(path string, opts TriggerFlowOptions) (TriggerFlowReport, error) {
	if opts.Limit <= 0 {
		opts.Limit = 100
	}
	if opts.Grep == "" && opts.UnitRef == nil && opts.Variable == nil {
		return TriggerFlowReport{}, fmt.Errorf("one of --grep, --unit-ref, or --variable is required")
	}
	file, err := OpenWithOptions(path, ParseOptions{MaxInflatedBytes: opts.MaxInflatedBytes})
	if err != nil {
		return TriggerFlowReport{}, err
	}
	report := TriggerFlowReport{
		Path:         path,
		Version:      file.Version,
		Verification: "structure_verified_not_engine_verified",
		Query:        opts,
	}
	stageOrder := []string{"initial_grant", "use_detection", "cooldown_or_state", "cleanup", "refresh_or_regrant", "transition", "other"}
	stages := map[string][]MechanicTrigger{}
	grep := strings.ToLower(opts.Grep)
	for _, row := range collectTriggerRows(file) {
		if grep != "" && !rowTextContains(row, grep) {
			continue
		}
		if opts.UnitRef != nil && !rowTouchesUnitRef(row, *opts.UnitRef) {
			continue
		}
		if opts.Variable != nil && !rowTouchesVariable(row, *opts.Variable) {
			continue
		}
		stage := flowStage(row)
		stages[stage] = append(stages[stage], mechanicTrigger(row, stage))
	}
	total := 0
	for _, stage := range stageOrder {
		items := stages[stage]
		if len(items) == 0 {
			continue
		}
		if total+len(items) > opts.Limit {
			remaining := opts.Limit - total
			if remaining < 0 {
				remaining = 0
			}
			items = items[:remaining]
			report.Warnings = append(report.Warnings, fmt.Sprintf("limit %d reached", opts.Limit))
		}
		report.Stages = append(report.Stages, TriggerFlowStage{Stage: stage, Triggers: items})
		total += len(items)
		if total >= opts.Limit {
			break
		}
	}
	if len(report.Stages) == 0 {
		report.Warnings = append(report.Warnings, "no matching triggers found")
	}
	return report, nil
}

func conditionTypeMatchesQuery(conditionType int, query string) bool {
	query = strings.TrimSpace(strings.ToLower(query))
	if query == "" {
		return false
	}
	if n, err := strconv.Atoi(query); err == nil {
		return conditionType == n
	}
	name := strings.ToLower(ConditionTypeName(conditionType))
	return name == query || strings.Contains(name, query)
}

func effectTouchesUnitRef(node *parsedNode, effect EffectSummary, unitRef int) bool {
	for _, id := range effect.SelectedObjectIDs {
		if id == unitRef {
			return true
		}
	}
	for _, field := range []string{"legacy_location_object_reference", "location_object_reference"} {
		if value, ok := node.intValue(field); ok && value == unitRef {
			return true
		}
	}
	return false
}

func effectTouchesPlayer(node *parsedNode, effect EffectSummary, player int) bool {
	if effect.SourcePlayer == player || effect.TargetPlayer == player {
		return true
	}
	for _, field := range []string{"source_player", "target_player"} {
		if value, ok := node.intValue(field); ok && value == player {
			return true
		}
	}
	return false
}

func effectTouchesArea(effect EffectSummary, area []int) bool {
	if len(effect.Location) == 2 {
		return pointInArea(effect.Location[0], effect.Location[1], area)
	}
	if len(effect.Area) == 4 {
		return areasOverlap(effect.Area, area)
	}
	return false
}

func conditionTouchesArea(condition map[string]any, area []int) bool {
	condArea := []int{
		intFromAny(condition["area_x1"]),
		intFromAny(condition["area_y1"]),
		intFromAny(condition["area_x2"]),
		intFromAny(condition["area_y2"]),
	}
	if condArea[0] == -1 && condArea[1] == -1 && condArea[2] == -1 && condArea[3] == -1 {
		return false
	}
	return areasOverlap(condArea, area)
}

func pointInArea(x, y int, area []int) bool {
	if len(area) != 4 {
		return false
	}
	x1, x2 := minMax(area[0], area[2])
	y1, y2 := minMax(area[1], area[3])
	return x >= x1 && x <= x2 && y >= y1 && y <= y2
}

func areasOverlap(a, b []int) bool {
	if len(a) != 4 || len(b) != 4 {
		return false
	}
	ax1, ax2 := minMax(a[0], a[2])
	ay1, ay2 := minMax(a[1], a[3])
	bx1, bx2 := minMax(b[0], b[2])
	by1, by2 := minMax(b[1], b[3])
	return ax1 <= bx2 && bx1 <= ax2 && ay1 <= by2 && by1 <= ay2
}

func minMax(a, b int) (int, int) {
	if a <= b {
		return a, b
	}
	return b, a
}

func rowTouchesUnitRef(row triggerRow, unitRef int) bool {
	for i, effect := range row.Effects {
		if effectTouchesUnitRef(row.EffectNodes[i], effect, unitRef) {
			return true
		}
	}
	for _, condition := range row.Conditions {
		if intFromAny(condition["unit_object"]) == unitRef || intFromAny(condition["next_object"]) == unitRef {
			return true
		}
	}
	return false
}

func rowTouchesVariable(row triggerRow, variable int) bool {
	for _, effect := range row.Effects {
		if effect.Variable == variable || effect.Variable2 == variable {
			return true
		}
	}
	for _, condition := range row.Conditions {
		if intFromAny(condition["variable"]) == variable || intFromAny(condition["variable2"]) == variable {
			return true
		}
	}
	return false
}

func rowTouchedPlayers(row triggerRow) map[int]bool {
	players := map[int]bool{}
	for i, effect := range row.Effects {
		for _, field := range []string{"source_player", "target_player"} {
			if value, ok := row.EffectNodes[i].intValue(field); ok && value > 0 {
				players[value] = true
			}
		}
		if effect.SourcePlayer > 0 {
			players[effect.SourcePlayer] = true
		}
		if effect.TargetPlayer > 0 {
			players[effect.TargetPlayer] = true
		}
	}
	for _, condition := range row.Conditions {
		for _, field := range []string{"source_player", "target_player"} {
			if value := intFromAny(condition[field]); value > 0 {
				players[value] = true
			}
		}
	}
	return players
}

func rowTextContains(row triggerRow, query string) bool {
	if query == "" {
		return true
	}
	for _, value := range []string{row.Name, row.Description, row.ShortDescription} {
		if strings.Contains(strings.ToLower(value), query) {
			return true
		}
	}
	for _, effect := range row.Effects {
		if strings.Contains(strings.ToLower(effect.TypeName), query) || strings.Contains(strings.ToLower(effect.Text), query) {
			return true
		}
	}
	for _, condition := range row.Conditions {
		if strings.Contains(strings.ToLower(ConditionTypeName(intFromAny(condition["type"]))), query) {
			return true
		}
	}
	return false
}

func triggerEdgeMap(rows []triggerRow) map[int][]TriggerNeighborhoodEdge {
	out := map[int][]TriggerNeighborhoodEdge{}
	for _, row := range rows {
		for _, effect := range row.Effects {
			if effect.TargetTrigger > 0 && effect.TargetTrigger < len(rows) {
				out[row.Index] = append(out[row.Index], TriggerNeighborhoodEdge{
					From:        row.Index,
					To:          effect.TargetTrigger,
					Kind:        effect.TypeName,
					SourceChild: fmt.Sprintf("effect[%d]", effect.EffectIndex),
				})
				out[effect.TargetTrigger] = append(out[effect.TargetTrigger], TriggerNeighborhoodEdge{
					From:        effect.TargetTrigger,
					To:          row.Index,
					Kind:        "referenced_by_" + effect.TypeName,
					SourceChild: fmt.Sprintf("trigger[%d].effect[%d]", row.Index, effect.EffectIndex),
				})
			}
		}
		for i, condition := range row.Conditions {
			target := intFromAny(condition["trigger_id"])
			if target > 0 && target < len(rows) {
				out[row.Index] = append(out[row.Index], TriggerNeighborhoodEdge{
					From:        row.Index,
					To:          target,
					Kind:        "condition_trigger_ref",
					SourceChild: fmt.Sprintf("condition[%d]", i),
				})
				out[target] = append(out[target], TriggerNeighborhoodEdge{
					From:        target,
					To:          row.Index,
					Kind:        "referenced_by_condition",
					SourceChild: fmt.Sprintf("trigger[%d].condition[%d]", row.Index, i),
				})
			}
		}
	}
	return out
}

func playerFamilyKey(name string) (string, int) {
	lower := strings.ToLower(strings.TrimSpace(name))
	if lower == "" {
		return "", 0
	}
	replacer := strings.NewReplacer("player ", "p", "player_", "p", "player-", "p")
	lower = replacer.Replace(lower)
	for player := 1; player <= 8; player++ {
		for _, token := range []string{fmt.Sprintf("p%d", player), fmt.Sprintf("p %d", player)} {
			if strings.Contains(lower, token) {
				key := strings.ReplaceAll(lower, token, "p#")
				key = strings.Join(strings.Fields(key), " ")
				return key, player
			}
		}
	}
	return "", 0
}

func normalizeMechanicKind(kind string) string {
	kind = strings.ToLower(strings.ReplaceAll(strings.TrimSpace(kind), "_", "-"))
	switch kind {
	case "garrison-token", "garrison", "token", "heal-token":
		return "garrison-token"
	case "transform-toggle", "toggle", "ratha-toggle", "replace-toggle":
		return "transform-toggle"
	case "teleport-transition", "teleport", "transition":
		return "teleport-transition"
	case "refresh-cycle", "refresh", "regrant", "respawn":
		return "refresh-cycle"
	default:
		return ""
	}
}

func mechanicRole(row triggerRow, kind string) string {
	effects := row.effectTypeSet()
	conditions := row.conditionTypeSet()
	name := strings.ToLower(row.Name)
	switch kind {
	case "garrison-token":
		switch {
		case effects["create_garrisoned_object"]:
			return "grant_or_regarrison"
		case conditions["units_garrisoned"]:
			return "use_or_presence_detection"
		case effects["unload"]:
			return "ungarrison_or_cast"
		case effects["heal_object"] || effects["change_object_hp"]:
			return "heal_resolution"
		case effects["kill_object"] || effects["remove_object"]:
			return "cleanup"
		case strings.Contains(name, "heal") || strings.Contains(name, "sheep") || strings.Contains(name, "chicken"):
			return "named_candidate"
		}
	case "transform-toggle":
		switch {
		case effects["replace_object"]:
			return "replace_form"
		case conditions["object_selected"] || conditions["object_selected_multiplayer"]:
			return "selection_detection"
		case effects["disable_object_selection"] || effects["enable_object_selection"]:
			return "selection_guard"
		case strings.Contains(name, "ratha") || strings.Contains(name, "toggle") || strings.Contains(name, "form"):
			return "named_candidate"
		}
	case "teleport-transition":
		switch {
		case effects["teleport_object"]:
			return "teleport"
		case effects["change_view"]:
			return "camera_or_view"
		case strings.Contains(name, "trial") || strings.Contains(name, "transition") || strings.Contains(name, "teleport"):
			return "named_candidate"
		}
	case "refresh-cycle":
		switch {
		case effects["activate_trigger"] || effects["deactivate_trigger"]:
			return "relay"
		case effects["create_object"] || effects["create_garrisoned_object"]:
			return "create_or_regrant"
		case effects["kill_object"] || effects["remove_object"]:
			return "cleanup"
		case effects["change_variable"] || effects["modify_variable_by_variable"] || conditions["variable_value"]:
			return "state_variable"
		case strings.Contains(name, "refresh") || strings.Contains(name, "reset") || strings.Contains(name, "respawn"):
			return "named_candidate"
		}
	}
	return ""
}

func mechanicTrigger(row triggerRow, role string) MechanicTrigger {
	playerSet := rowTouchedPlayers(row)
	effects := row.effectTypeNames()
	conditions := row.conditionTypeNames()
	notes := []string{}
	if row.Enabled == 0 {
		notes = append(notes, "disabled at scenario start")
	}
	if row.Looping != 0 {
		notes = append(notes, "looping")
	}
	return MechanicTrigger{
		Index:      row.Index,
		Name:       row.Name,
		Role:       role,
		Players:    sortedBoolKeys(playerSet),
		Effects:    effects,
		Conditions: conditions,
		Notes:      notes,
	}
}

func flowStage(row triggerRow) string {
	name := strings.ToLower(row.Name)
	effects := row.effectTypeSet()
	conditions := row.conditionTypeSet()
	switch {
	case strings.Contains(name, "init") || strings.Contains(name, "start") || strings.Contains(name, "grant"):
		return "initial_grant"
	case conditions["object_selected"] || conditions["object_selected_multiplayer"] || conditions["units_garrisoned"]:
		return "use_detection"
	case effects["change_variable"] || effects["modify_variable_by_variable"] || conditions["variable_value"]:
		return "cooldown_or_state"
	case effects["kill_object"] || effects["remove_object"]:
		return "cleanup"
	case strings.Contains(name, "refresh") || strings.Contains(name, "respawn") || strings.Contains(name, "regrant") || effects["create_garrisoned_object"]:
		return "refresh_or_regrant"
	case strings.Contains(name, "trial") || strings.Contains(name, "transition") || effects["teleport_object"]:
		return "transition"
	default:
		return "other"
	}
}

func (row triggerRow) effectTypeSet() map[string]bool {
	out := map[string]bool{}
	for _, effect := range row.Effects {
		out[effect.TypeName] = true
	}
	return out
}

func (row triggerRow) conditionTypeSet() map[string]bool {
	out := map[string]bool{}
	for _, condition := range row.Conditions {
		out[ConditionTypeName(intFromAny(condition["type"]))] = true
	}
	return out
}

func (row triggerRow) effectTypeNames() []string {
	set := row.effectTypeSet()
	out := make([]string, 0, len(set))
	for name := range set {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

func (row triggerRow) conditionTypeNames() []string {
	set := row.conditionTypeSet()
	out := make([]string, 0, len(set))
	for name := range set {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

func copyStringMap(in map[string]string) map[string]string {
	out := map[string]string{}
	for k, v := range in {
		out[k] = v
	}
	return out
}

func hasAnyPrefixed(fields map[string]string, prefixes ...string) bool {
	for key := range fields {
		for _, prefix := range prefixes {
			if strings.HasPrefix(key, prefix) {
				return true
			}
		}
	}
	return false
}

func intFromAny(value any) int {
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
	case uint32:
		return int(v)
	case float64:
		return int(v)
	default:
		return -1
	}
}

func sortedIntMapKeys[V any](m map[int]V) []int {
	out := make([]int, 0, len(m))
	for key := range m {
		out = append(out, key)
	}
	sort.Ints(out)
	return out
}

func sortedSeenByDepth(seen map[int]int) []int {
	out := sortedIntMapKeys(seen)
	sort.Slice(out, func(i, j int) bool {
		if seen[out[i]] == seen[out[j]] {
			return out[i] < out[j]
		}
		return seen[out[i]] < seen[out[j]]
	})
	return out
}

func sortedBoolKeys(m map[int]bool) []int {
	out := make([]int, 0, len(m))
	for key := range m {
		out = append(out, key)
	}
	sort.Ints(out)
	return out
}

func sortedCoverageKeys(m map[string]*PlayerCoverageFamily) []string {
	out := make([]string, 0, len(m))
	for key := range m {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

func appendUniqueInt(values []int, value int) []int {
	if containsInt(values, value) {
		return values
	}
	return append(values, value)
}
