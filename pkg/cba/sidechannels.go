package cba

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"aoe2kit/pkg/replay"
)

type SideChannelReport struct {
	Path          string                 `json:"path,omitempty"`
	Method        string                 `json:"method"`
	Verification  string                 `json:"verification"`
	GraphSHA256   string                 `json:"graph_sha256,omitempty"`
	ScenarioToken string                 `json:"scenario_token,omitempty"`
	Scenario      ScenarioIdentity       `json:"scenario_identity"`
	DataSet       replay.DataSetIdentity `json:"data_set_identity"`
	Summary       SideChannelSummary     `json:"summary"`
	Scoreboard    []ScoreboardChannelRow `json:"scoreboard,omitempty"`
	EngineAttrs   []EngineAttributeRow   `json:"engine_attributes,omitempty"`
	Resources     []SideChannelResource  `json:"resources,omitempty"`
	RegisterMap   []RegisterMapRow       `json:"register_map,omitempty"`
	Thresholds    []SideChannelThreshold `json:"thresholds,omitempty"`
	Templates     []string               `json:"scoreboard_templates,omitempty"`
	Notes         []SideChannelNote      `json:"notes,omitempty"`
	Warnings      []string               `json:"warnings,omitempty"`
}

type SideChannelSummary struct {
	SideChannelVariant    string `json:"sidechannel_variant"`
	TriggerCount          int    `json:"trigger_count"`
	EffectCount           int    `json:"effect_count"`
	ConditionCount        int    `json:"condition_count"`
	MessageCount          int    `json:"message_count"`
	ModifyResourceEffects int    `json:"modify_resource_effects"`
	ChangeVariableEffects int    `json:"change_variable_effects"`
	AccumAttributeConds   int    `json:"accumulate_attribute_conditions"`
	ScoreboardRows        int    `json:"scoreboard_rows"`
	EngineAttrChannels    int    `json:"engine_attr_channels"`
	ResourcesWritten      int    `json:"resources_written"`
	ThresholdChannels     int    `json:"threshold_channels"`
}

type ScoreboardChannelRow struct {
	Label          string `json:"label"`
	Players        []int  `json:"players"`
	KillResourceA  int    `json:"kill_resource_a"`
	KillResourceB  int    `json:"kill_resource_b"`
	DeathResourceA int    `json:"death_resource_a"`
	DeathResourceB int    `json:"death_resource_b"`
	RazeResourceA  int    `json:"raze_resource_a"`
	RazeResourceB  int    `json:"raze_resource_b"`
	TemplateLine   string `json:"template_line,omitempty"`
	Confidence     string `json:"confidence"`
}

type EngineAttributeRow struct {
	Attribute      int    `json:"attribute"`
	Name           string `json:"name"`
	Semantic       string `json:"semantic"`
	ConditionType  int    `json:"condition_type"`
	ConditionName  string `json:"condition_name"`
	Quantities     []int  `json:"quantities"`
	Players        []int  `json:"players"`
	ConditionCount int    `json:"condition_count"`
	Notes          string `json:"notes,omitempty"`
}

type SideChannelResource struct {
	Resource int    `json:"resource"`
	Family   string `json:"family"`
	Count    int    `json:"write_count"`
	Amounts  []int  `json:"amounts"`
	Players  []int  `json:"players"`
}

type RegisterMapRow struct {
	Lane       int                   `json:"lane"`
	Label      string                `json:"label"`
	Players    []int                 `json:"players"`
	Metric     string                `json:"metric"`
	Layer      string                `json:"layer"`
	Resources  []RegisterMapResource `json:"resources"`
	Confidence string                `json:"confidence"`
	Notes      string                `json:"notes,omitempty"`
}

type RegisterMapResource struct {
	Resource   int                `json:"resource"`
	Role       string             `json:"role"`
	WriteCount int                `json:"write_count"`
	Amounts    []int              `json:"amounts,omitempty"`
	Players    []int              `json:"players,omitempty"`
	Evidence   []RegisterEvidence `json:"evidence,omitempty"`
}

type RegisterEvidence struct {
	TriggerIndex int `json:"trigger_index"`
	EffectIndex  int `json:"effect_index"`
	Player       int `json:"player,omitempty"`
	Operation    int `json:"operation,omitempty"`
	Amount       int `json:"amount,omitempty"`
}

type SideChannelThreshold struct {
	Kind      string   `json:"kind"`
	Attribute int      `json:"attribute"`
	Players   []int    `json:"players"`
	Values    []int    `json:"values"`
	Messages  []string `json:"messages,omitempty"`
}

type SideChannelNote struct {
	Name       string `json:"name"`
	Confidence string `json:"confidence"`
	Detail     string `json:"detail"`
}

type attrAgg struct {
	Attr       int
	Count      int
	Quantities map[int]bool
	Players    map[int]bool
}

type resAgg struct {
	Res     int
	Count   int
	Amounts map[int]bool
	Players map[int]bool
	Samples []RegisterEvidence
}

type thresholdAgg struct {
	Kind     string
	Attr     int
	Players  map[int]bool
	Values   map[int]bool
	Messages map[string]bool
}

func BuildSideChannels(path string) (*SideChannelReport, error) {
	rec, err := replay.Open(path)
	if err != nil {
		return nil, err
	}
	if rec.TriggerGraph == nil {
		return nil, fmt.Errorf("trigger graph unavailable: %s", rec.TriggerGraphErr)
	}
	report := &SideChannelReport{
		Path:          path,
		Method:        "cba_requiem_v292_trigger_graph_scoreboard_sidechannel_decode",
		Verification:  "structure_verified_from_replay_embedded_trigger_graph_not_runtime_resource_visibility_verified",
		GraphSHA256:   rec.TriggerGraph.SHA256,
		ScenarioToken: scenarioToken(rec.HeaderBytes()),
		DataSet:       rec.DataSet,
	}
	report.Scenario = ScenarioIdentity{
		Token:        report.ScenarioToken,
		TriggerCount: rec.TriggerGraph.TriggerCount,
		Ladder:       report.ScenarioToken == LadderToken && rec.TriggerGraph.TriggerCount >= RequiemTriggerMin && rec.TriggerGraph.TriggerCount <= RequiemTriggerMax,
	}
	if report.Scenario.Token == "" {
		report.Scenario.Reason = "no CBA Requiem token in header"
	} else if report.Scenario.Token != LadderToken {
		report.Scenario.Reason = "non-ladder variant: " + report.Scenario.Token
	} else if report.Scenario.Ladder {
		report.Scenario.Reason = "official V292"
	} else {
		report.Scenario.Reason = fmt.Sprintf("trigger count %d outside V292 band %d..%d",
			rec.TriggerGraph.TriggerCount, RequiemTriggerMin, RequiemTriggerMax)
	}
	report.Summary.TriggerCount = rec.TriggerGraph.TriggerCount
	report.Summary.EffectCount = rec.TriggerGraph.EffectCount
	report.Summary.ConditionCount = rec.TriggerGraph.ConditionCount
	report.Summary.MessageCount = rec.TriggerGraph.MessageCount
	report.Scoreboard = scoreboardRows(rec.TriggerGraph.Triggers)
	report.Templates = scoreboardTemplates(rec.TriggerGraph.Triggers)

	attrAggs := map[int]*attrAgg{}
	resAggs := map[int]*resAgg{}
	thresholds := map[string]*thresholdAgg{}

	for triggerID, trigger := range rec.TriggerGraph.Triggers {
		messages := triggerMessages(trigger)
		for _, condition := range mapSlice(trigger, "conditions") {
			fields := intSlice(condition, "fields")
			if len(fields) == 0 {
				continue
			}
			if fields[0] == 8 && len(fields) > 7 {
				report.Summary.AccumAttributeConds++
				attr := fields[3]
				agg := attrAggs[attr]
				if agg == nil {
					agg = &attrAgg{Attr: attr, Quantities: map[int]bool{}, Players: map[int]bool{}}
					attrAggs[attr] = agg
				}
				agg.Count++
				agg.Quantities[fields[2]] = true
				if fields[7] >= 0 {
					agg.Players[fields[7]] = true
				}
				if kind := thresholdKind(attr); kind != "" {
					key := fmt.Sprintf("%s:%d", kind, attr)
					th := thresholds[key]
					if th == nil {
						th = &thresholdAgg{Kind: kind, Attr: attr, Players: map[int]bool{}, Values: map[int]bool{}, Messages: map[string]bool{}}
						thresholds[key] = th
					}
					if fields[7] >= 0 {
						th.Players[fields[7]] = true
					}
					th.Values[fields[2]] = true
					for _, msg := range messages {
						if sideChannelMessageForKind(kind, msg) {
							th.Messages[cleanCBAWhitespace(msg)] = true
						}
					}
				}
			}
		}
		for effectID, effect := range mapSlice(trigger, "effects") {
			fields := intSlice(effect, "fields_prefix")
			if len(fields) == 0 {
				continue
			}
			switch fields[0] {
			case 52:
				report.Summary.ModifyResourceEffects++
				if len(fields) > 9 {
					res := fields[4]
					agg := resAggs[res]
					if agg == nil {
						agg = &resAgg{Res: res, Amounts: map[int]bool{}, Players: map[int]bool{}}
						resAggs[res] = agg
					}
					agg.Count++
					agg.Amounts[fields[3]] = true
					if fields[9] >= 0 {
						agg.Players[fields[9]] = true
					}
					if len(agg.Samples) < 6 {
						operation := 0
						if fields[2] >= 0 {
							operation = fields[2]
						}
						agg.Samples = append(agg.Samples, RegisterEvidence{
							TriggerIndex: triggerID,
							EffectIndex:  effectID,
							Player:       fields[9],
							Operation:    operation,
							Amount:       fields[3],
						})
					}
				}
			case 56:
				report.Summary.ChangeVariableEffects++
			}
		}
	}

	for _, attr := range sortedAttrAggs(attrAggs) {
		if row, ok := engineAttributeRow(attr); ok {
			report.EngineAttrs = append(report.EngineAttrs, row)
		}
	}
	for _, res := range sortedResAggs(resAggs) {
		report.Resources = append(report.Resources, SideChannelResource{
			Resource: res.Res,
			Family:   resourceFamily(res.Res),
			Count:    res.Count,
			Amounts:  sortedBoolKeys(res.Amounts),
			Players:  sortedBoolKeys(res.Players),
		})
	}
	report.RegisterMap = buildRegisterMap(resAggs)
	for _, th := range sortedThresholdAggs(thresholds) {
		report.Thresholds = append(report.Thresholds, SideChannelThreshold{
			Kind:      th.Kind,
			Attribute: th.Attr,
			Players:   sortedBoolKeys(th.Players),
			Values:    sortedBoolKeys(th.Values),
			Messages:  limitedSortedStrings(th.Messages, 16),
		})
	}

	report.Summary.ScoreboardRows = len(report.Scoreboard)
	report.Summary.EngineAttrChannels = len(report.EngineAttrs)
	report.Summary.ResourcesWritten = len(report.Resources)
	report.Summary.ThresholdChannels = len(report.Thresholds)
	report.Summary.SideChannelVariant = classifySideChannelVariant(report)
	report.Notes = []SideChannelNote{
		{
			Name:       "scoreboard_is_quantized",
			Confidence: "structure_verified",
			Detail:     "CBA consumes engine kill/death attributes in 10-count buckets for the live objective display; this is not exact per-death telemetry.",
		},
		{
			Name:       "trigger_graph_not_runtime_state",
			Confidence: "honesty_boundary",
			Detail:     "This report decodes authored CBA accounting paths. It does not prove the replay stream exposes every resource write or trigger firing at runtime.",
		},
		{
			Name:       "runtime_probe_target",
			Confidence: "recommended_next_step",
			Detail:     "Probe resources 571-574, 391-398, 60, 198-200, 361-364, 111-118, and 395-398 with unique values to determine whether replay sync/state blocks carry them.",
		},
	}
	if rec.TriggerGraph.TriggerCount != 3218 {
		report.Warnings = append(report.Warnings, fmt.Sprintf("trigger count is %d, not the known CBA Requiem V292 embedded count 3218; decode may still be useful but should not be labeled V292 without identity verification", rec.TriggerGraph.TriggerCount))
	}
	if report.Summary.SideChannelVariant != "cba_requiem_v292_full_kill_death_raze_bridge" {
		report.Warnings = append(report.Warnings, "side-channel variant is not the full V292 kill/death/raze bridge; only the channels present in engine_attributes/resources should be claimed")
	}
	if len(report.Templates) == 0 {
		report.Warnings = append(report.Warnings, "no CBA scoreboard template with Unused Resource substitutions was found")
	}
	report.Warnings = append(report.Warnings, rec.TriggerGraph.Warnings...)
	return report, nil
}

func classifySideChannelVariant(report *SideChannelReport) string {
	hasScoreboard := len(report.Scoreboard) == 4
	hasKills := hasEngineAttribute(report.EngineAttrs, 20)
	hasDeaths := hasEngineAttribute(report.EngineAttrs, 154)
	hasRazes := hasEngineAttribute(report.EngineAttrs, 43)
	switch {
	case report.GraphSHA256 == "172fa04c719615257ecc7991a2605c8f7ac45c05bdd72a8758c759687f2dd28c" && hasScoreboard && hasKills && hasDeaths && hasRazes:
		return "cba_requiem_v292_full_kill_death_raze_bridge"
	case report.Scenario.Ladder && hasScoreboard && hasKills && hasDeaths && hasRazes:
		return "cba_requiem_v292_kill_death_raze_bridge_other_host_copy"
	case report.Scenario.Ladder && hasScoreboard && hasKills && hasDeaths:
		return "cba_requiem_v292_kill_death_bridge_raze_absent_or_truncated"
	case report.Scenario.Ladder && hasScoreboard:
		return "cba_requiem_v292_scoreboard_template_partial"
	case hasScoreboard:
		return "cba_family_scoreboard_template_unverified_variant"
	default:
		return "unknown_or_no_cba_scoreboard_sidechannel"
	}
}

func hasEngineAttribute(rows []EngineAttributeRow, attr int) bool {
	for _, row := range rows {
		if row.Attribute == attr {
			return true
		}
	}
	return false
}

func scoreboardRows(triggers []map[string]any) []ScoreboardChannelRow {
	templates := scoreboardTemplates(triggers)
	byLabel := map[string]string{}
	for _, line := range templates {
		if strings.HasPrefix(line, "P1-5") {
			byLabel["P1-5"] = line
		} else if strings.HasPrefix(line, "P2-6") {
			byLabel["P2-6"] = line
		} else if strings.HasPrefix(line, "P3-7") {
			byLabel["P3-7"] = line
		} else if strings.HasPrefix(line, "P4-8") {
			byLabel["P4-8"] = line
		}
	}
	rows := []ScoreboardChannelRow{
		{Label: "P1-5", Players: []int{1, 5}, KillResourceA: 571, KillResourceB: 391, DeathResourceA: 60, DeathResourceB: 361, RazeResourceA: 111, RazeResourceB: 395},
		{Label: "P2-6", Players: []int{2, 6}, KillResourceA: 572, KillResourceB: 392, DeathResourceA: 198, DeathResourceB: 362, RazeResourceA: 112, RazeResourceB: 396},
		{Label: "P3-7", Players: []int{3, 7}, KillResourceA: 573, KillResourceB: 393, DeathResourceA: 199, DeathResourceB: 363, RazeResourceA: 113, RazeResourceB: 397},
		{Label: "P4-8", Players: []int{4, 8}, KillResourceA: 574, KillResourceB: 394, DeathResourceA: 200, DeathResourceB: 364, RazeResourceA: 114, RazeResourceB: 398},
	}
	for i := range rows {
		rows[i].TemplateLine = byLabel[rows[i].Label]
		rows[i].Confidence = "decoded_from_objective_unused_resource_template"
	}
	return rows
}

func scoreboardTemplates(triggers []map[string]any) []string {
	seen := map[string]bool{}
	for _, trigger := range triggers {
		for _, msg := range triggerMessages(trigger) {
			if !strings.Contains(msg, "<Unused Resource") && !strings.Contains(msg, "<Units Killed>") && !strings.Contains(msg, "<Killed by Others>") {
				continue
			}
			for _, line := range strings.Split(cleanCBAWhitespace(msg), "\n") {
				line = strings.TrimSpace(line)
				if line == "" || seen[line] {
					continue
				}
				seen[line] = true
			}
		}
	}
	return sortedStringSet(seen)
}

func triggerMessages(trigger map[string]any) []string {
	var out []string
	for _, key := range []string{"name", "short_description", "description"} {
		if msg, _ := trigger[key].(string); msg != "" {
			out = append(out, msg)
		}
	}
	for _, effect := range mapSlice(trigger, "effects") {
		if msg, _ := effect["message"].(string); msg != "" {
			out = append(out, msg)
		}
	}
	return out
}

var sideChannelSpaceRE = regexp.MustCompile(`[ \t]+`)

func cleanCBAWhitespace(value string) string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")
	lines := strings.Split(value, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimSpace(sideChannelSpaceRE.ReplaceAllString(line, " "))
	}
	return strings.Join(lines, "\n")
}

func engineAttributeRow(agg *attrAgg) (EngineAttributeRow, bool) {
	row := EngineAttributeRow{
		Attribute:      agg.Attr,
		ConditionType:  8,
		ConditionName:  "ACCUMULATE_ATTRIBUTE",
		Quantities:     sortedBoolKeys(agg.Quantities),
		Players:        sortedBoolKeys(agg.Players),
		ConditionCount: agg.Count,
	}
	switch agg.Attr {
	case 20:
		row.Name = "cAttributeKills"
		row.Semantic = "engine_kills_consumed_by_cba_scoreboard"
		row.Notes = "CBA moves kills into mirrored resources in 10-count buckets; some 5-count checks gate Castle Age threshold messages."
	case 154:
		row.Name = "cAttributeKilledByOthers"
		row.Semantic = "engine_deaths_consumed_by_cba_scoreboard"
		row.Notes = "CBA moves deaths into mirrored resources in 10-count buckets."
	case 43:
		row.Name = "cAttributeRazings"
		row.Semantic = "engine_razes_consumed_by_cba_villager_reward_and_scoreboard"
		row.Notes = "CBA consumes razes one at a time for villager rewards and mirrored raze display."
	default:
		return EngineAttributeRow{}, false
	}
	return row, true
}

func resourceFamily(resource int) string {
	switch {
	case resource >= 571 && resource <= 574:
		return "kill_scoreboard_resource_primary"
	case resource >= 391 && resource <= 394:
		return "kill_scoreboard_resource_partner_or_gaia_mirror"
	case resource == 20:
		return "engine_kill_resource_self_copy"
	case resource == 8 || resource == 10 || resource == 18 || resource == 220:
		return "kill_copy_resource_unlabeled_in_template"
	case resource == 60 || resource == 198 || resource == 199 || resource == 200:
		return "death_scoreboard_resource_primary"
	case resource >= 361 && resource <= 364:
		return "death_scoreboard_resource_partner_or_gaia_mirror"
	case resource == 154:
		return "engine_death_resource_self_copy"
	case resource == 30 || resource == 31 || resource == 51 || resource == 59:
		return "death_copy_resource_unlabeled_in_template"
	case resource >= 111 && resource <= 118:
		return "raze_scoreboard_resource_primary_or_cross_team_copy"
	case resource >= 395 && resource <= 398:
		return "raze_scoreboard_resource_partner_or_gaia_mirror"
	case resource == 43:
		return "engine_raze_resource_self_copy"
	case resource >= 201 && resource <= 204:
		return "late_kill_threshold_resource_unlabeled"
	default:
		return "other_modify_resource"
	}
}

func buildRegisterMap(resources map[int]*resAgg) []RegisterMapRow {
	lanes := []struct {
		lane          int
		label         string
		players       []int
		killPrimary   int
		killPartner   int
		killEnemy     int
		deathPrimary  int
		deathPartner  int
		deathEnemy    int
		razePrimary   int
		razePartner   int
		razeCrossCopy int
	}{
		{1, "P1-5", []int{1, 5}, 571, 391, 220, 60, 361, 30, 111, 395, 115},
		{2, "P2-6", []int{2, 6}, 572, 392, 8, 198, 362, 31, 112, 396, 116},
		{3, "P3-7", []int{3, 7}, 573, 393, 10, 199, 363, 51, 113, 397, 117},
		{4, "P4-8", []int{4, 8}, 574, 394, 18, 200, 364, 59, 114, 398, 118},
	}
	var out []RegisterMapRow
	for _, lane := range lanes {
		out = append(out,
			RegisterMapRow{
				Lane:       lane.lane,
				Label:      lane.label,
				Players:    lane.players,
				Metric:     "kills",
				Layer:      "visible_scoreboard_and_enemy_team_mirror",
				Confidence: "template_verified_plus_modify_resource_evidence",
				Notes:      "Visible objective row uses the primary/partner resources; enemy-team mirror is an unlabeled copy resource from activator fan-outs.",
				Resources: []RegisterMapResource{
					registerResource(resources, lane.killPrimary, "visible_scoreboard_primary"),
					registerResource(resources, lane.killPartner, "visible_scoreboard_partner_or_gaia_mirror"),
					registerResource(resources, lane.killEnemy, "enemy_team_mirror_unlabeled"),
					registerResource(resources, 20, "raw_engine_attr_self_copy"),
				},
			},
			RegisterMapRow{
				Lane:       lane.lane,
				Label:      lane.label,
				Players:    lane.players,
				Metric:     "deaths",
				Layer:      "visible_scoreboard_and_enemy_team_mirror",
				Confidence: "template_verified_plus_modify_resource_evidence",
				Notes:      "Visible objective row uses the primary/partner resources; enemy-team mirror is an unlabeled copy resource from activator fan-outs.",
				Resources: []RegisterMapResource{
					registerResource(resources, lane.deathPrimary, "visible_scoreboard_primary"),
					registerResource(resources, lane.deathPartner, "visible_scoreboard_partner_or_gaia_mirror"),
					registerResource(resources, lane.deathEnemy, "enemy_team_mirror_unlabeled"),
					registerResource(resources, 154, "raw_engine_attr_self_copy"),
				},
			},
			RegisterMapRow{
				Lane:       lane.lane,
				Label:      lane.label,
				Players:    lane.players,
				Metric:     "razes",
				Layer:      "visible_scoreboard_and_cross_team_copy",
				Confidence: "template_verified_plus_modify_resource_evidence",
				Notes:      "Resources 111-114 and 395-398 render in the objective template; 115-118 are separate cross-copy raze registers and are not directly shown in the template.",
				Resources: []RegisterMapResource{
					registerResource(resources, lane.razePrimary, "visible_scoreboard_primary"),
					registerResource(resources, lane.razePartner, "visible_scoreboard_partner_or_gaia_mirror"),
					registerResource(resources, lane.razeCrossCopy, "cross_team_copy_unlabeled"),
					registerResource(resources, 43, "raw_engine_attr_self_copy"),
				},
			},
		)
	}
	return out
}

func registerResource(resources map[int]*resAgg, resource int, role string) RegisterMapResource {
	row := RegisterMapResource{Resource: resource, Role: role}
	if agg := resources[resource]; agg != nil {
		row.WriteCount = agg.Count
		row.Amounts = sortedBoolKeys(agg.Amounts)
		row.Players = sortedBoolKeys(agg.Players)
		row.Evidence = append(row.Evidence, agg.Samples...)
	}
	return row
}

func thresholdKind(attr int) string {
	switch {
	case attr >= 571 && attr <= 574:
		return "kill_threshold"
	case attr == 20:
		return "raw_kill_bucket"
	case attr == 154:
		return "raw_death_bucket"
	case attr == 43:
		return "raze_reward_step"
	case attr == 8 || attr == 10 || attr == 18 || attr == 220:
		return "kill_copy_threshold_unlabeled"
	case attr >= 201 && attr <= 204:
		return "late_kill_threshold_unlabeled"
	default:
		return ""
	}
}

func sideChannelMessageForKind(kind string, msg string) bool {
	msg = strings.ToLower(msg)
	switch kind {
	case "kill_threshold", "raw_kill_bucket", "kill_copy_threshold_unlabeled", "late_kill_threshold_unlabeled":
		return strings.Contains(msg, "kill")
	case "raw_death_bucket":
		return strings.Contains(msg, "death") || strings.Contains(msg, "killed by others")
	case "raze_reward_step":
		return strings.Contains(msg, "raze")
	default:
		return false
	}
}

func sortedAttrAggs(values map[int]*attrAgg) []*attrAgg {
	out := make([]*attrAgg, 0, len(values))
	for _, value := range values {
		out = append(out, value)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Attr < out[j].Attr })
	return out
}

func sortedResAggs(values map[int]*resAgg) []*resAgg {
	out := make([]*resAgg, 0, len(values))
	for _, value := range values {
		out = append(out, value)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Res < out[j].Res })
	return out
}

func sortedThresholdAggs(values map[string]*thresholdAgg) []*thresholdAgg {
	out := make([]*thresholdAgg, 0, len(values))
	for _, value := range values {
		out = append(out, value)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Kind != out[j].Kind {
			return out[i].Kind < out[j].Kind
		}
		return out[i].Attr < out[j].Attr
	})
	return out
}

func sortedBoolKeys(values map[int]bool) []int {
	out := make([]int, 0, len(values))
	for value := range values {
		out = append(out, value)
	}
	sort.Ints(out)
	return out
}

func sortedStringSet(values map[string]bool) []string {
	out := make([]string, 0, len(values))
	for value := range values {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func limitedSortedStrings(values map[string]bool, limit int) []string {
	out := sortedStringSet(values)
	if limit > 0 && len(out) > limit {
		return out[:limit]
	}
	return out
}
