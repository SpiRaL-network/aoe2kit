package cba

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"aoe2kit/pkg/datfile"
	"aoe2kit/pkg/replay"
)

type TriggerRazeReport struct {
	Path         string                `json:"path,omitempty"`
	DatPath      string                `json:"dat_path,omitempty"`
	Method       string                `json:"method"`
	Verification string                `json:"verification"`
	Summary      TriggerRazeSummary    `json:"summary"`
	Rows         []TriggerRazeRow      `json:"rows,omitempty"`
	Buckets      []TriggerRazeBucket   `json:"buckets,omitempty"`
	Mechanics    TriggerRazeMechanics  `json:"mechanics"`
	Conflicts    []TriggerRazeConflict `json:"conflicts,omitempty"`
	Warnings     []string              `json:"warnings,omitempty"`
}

type TriggerRazeSummary struct {
	TriggerCount           int `json:"trigger_count"`
	RewardRungTriggers     int `json:"reward_rung_triggers"`
	ActivatorTriggers      int `json:"activator_triggers"`
	Rows                   int `json:"rows"`
	ResolvedCivs           int `json:"resolved_civs"`
	UniqueTechs            int `json:"unique_techs"`
	Buckets                int `json:"buckets"`
	Conflicts              int `json:"conflicts"`
	UnresolvedTechs        int `json:"unresolved_techs"`
	MapucheRazesToVillager int `json:"mapuche_razes_to_villager,omitempty"`
	KhmerRazesToVillager   int `json:"khmer_razes_to_villager,omitempty"`
}

type TriggerRazeRow struct {
	CivID           int    `json:"civ_id,omitempty"`
	CivName         string `json:"civ_name,omitempty"`
	TechID          int    `json:"technology_id"`
	TechName        string `json:"technology_name,omitempty"`
	RazesToVillager int    `json:"razes_to_villager"`
	TriggerID       int    `json:"trigger_id"`
	TargetTriggerID int    `json:"target_trigger_id"`
	SourcePlayers   []int  `json:"source_players,omitempty"`
	Message         string `json:"message,omitempty"`
	Confidence      string `json:"confidence"`
}

type TriggerRazeBucket struct {
	RazesToVillager int      `json:"razes_to_villager"`
	TechIDs         []int    `json:"technology_ids"`
	CivIDs          []int    `json:"civ_ids,omitempty"`
	CivNames        []string `json:"civ_names,omitempty"`
	Count           int      `json:"count"`
}

type TriggerRazeMechanics struct {
	RazeAttribute       int    `json:"raze_attribute"`
	RazeAttributeName   string `json:"raze_attribute_name"`
	ConditionType       int    `json:"condition_type"`
	ConditionName       string `json:"condition_name"`
	RewardEffectType    int    `json:"reward_effect_type"`
	RewardEffectName    string `json:"reward_effect_name"`
	ActivatorEffectType int    `json:"activator_effect_type"`
	ActivatorEffectName string `json:"activator_effect_name"`
	Notes               string `json:"notes"`
}

type TriggerRazeConflict struct {
	TechID int   `json:"technology_id"`
	Values []int `json:"values"`
}

type triggerRazeRung struct {
	TriggerID       int
	Player          int
	RazesToVillager int
	Message         string
}

type triggerRazeCandidate struct {
	TechID          int
	RazesToVillager int
	TriggerID       int
	TargetTriggerID int
	Message         string
	SourcePlayers   map[int]bool
}

func BuildTriggerRazes(path, datPath string) (*TriggerRazeReport, error) {
	rec, err := replay.Open(path)
	if err != nil {
		return nil, err
	}
	if rec.TriggerGraph == nil {
		return nil, fmt.Errorf("trigger graph unavailable: %s", rec.TriggerGraphErr)
	}
	var dat *datfile.Index
	if datPath != "" {
		dat, err = datfile.Open(datPath)
		if err != nil {
			return nil, err
		}
	}
	rungs := map[int]triggerRazeRung{}
	for triggerID, trigger := range rec.TriggerGraph.Triggers {
		rung, ok := parseTriggerRazeRung(triggerID, trigger)
		if ok {
			rungs[triggerID] = rung
		}
	}
	candidates := map[int]map[int]*triggerRazeCandidate{}
	activatorTriggers := map[int]bool{}
	for triggerID, trigger := range rec.TriggerGraph.Triggers {
		if _, isRung := rungs[triggerID]; isRung {
			continue
		}
		for _, targetID := range activateTargets(trigger) {
			rung, ok := rungs[targetID]
			if !ok {
				continue
			}
			techIDs := researchTechConditions(trigger)
			if len(techIDs) == 0 {
				continue
			}
			activatorTriggers[triggerID] = true
			for _, techID := range techIDs {
				if _, ok := candidates[techID]; !ok {
					candidates[techID] = map[int]*triggerRazeCandidate{}
				}
				c := candidates[techID][rung.RazesToVillager]
				if c == nil {
					c = &triggerRazeCandidate{
						TechID:          techID,
						RazesToVillager: rung.RazesToVillager,
						TriggerID:       triggerID,
						TargetTriggerID: targetID,
						Message:         firstTriggerMessage(trigger),
						SourcePlayers:   map[int]bool{},
					}
					candidates[techID][rung.RazesToVillager] = c
				}
				if rung.Player > 0 {
					c.SourcePlayers[rung.Player] = true
				}
			}
		}
	}
	report := &TriggerRazeReport{
		Path:         path,
		DatPath:      datPath,
		Method:       "cba_requiem_trigger_graph_raze_rung_activation_decode",
		Verification: "structure_verified_trigger_graph_plus_dat_tech_name_cross_reference_not_engine_runtime_verified",
		Mechanics: TriggerRazeMechanics{
			RazeAttribute:       43,
			RazeAttributeName:   "cAttributeRazings",
			ConditionType:       8,
			ConditionName:       "ACCUMULATE_ATTRIBUTE",
			RewardEffectType:    52,
			RewardEffectName:    "MODIFY_RESOURCE",
			ActivatorEffectType: 8,
			ActivatorEffectName: "ACTIVATE_TRIGGER",
			Notes:               "Per-player reward rung triggers consume cAttributeRazings one at a time; civ-start activators select the initial rung, which determines razes_to_villager.",
		},
	}
	report.Summary.TriggerCount = rec.TriggerGraph.TriggerCount
	report.Summary.RewardRungTriggers = len(rungs)
	report.Summary.ActivatorTriggers = len(activatorTriggers)
	seenCivs := map[int]bool{}
	seenBuckets := map[int]*TriggerRazeBucket{}
	for techID, byValue := range candidates {
		values := make([]int, 0, len(byValue))
		for value := range byValue {
			values = append(values, value)
		}
		sort.Ints(values)
		if len(values) > 1 {
			report.Conflicts = append(report.Conflicts, TriggerRazeConflict{TechID: techID, Values: values})
			continue
		}
		c := byValue[values[0]]
		row := TriggerRazeRow{
			TechID:          techID,
			RazesToVillager: c.RazesToVillager,
			TriggerID:       c.TriggerID,
			TargetTriggerID: c.TargetTriggerID,
			Message:         c.Message,
			SourcePlayers:   sortedPlayerKeys(c.SourcePlayers),
			Confidence:      "trigger_graph_civ_tech_activates_raze_reward_rung",
		}
		if dat != nil {
			if tech, ok := dat.Tech(techID); ok {
				row.TechName = tech.Name
				row.CivID = int(tech.Civ)
				if row.CivID >= 0 {
					row.CivName = replay.CivDisplayName(row.CivID)
					if row.CivName == "" {
						row.CivName = tech.Name
					}
				}
			} else {
				report.Summary.UnresolvedTechs++
			}
		}
		if row.CivID > 0 {
			seenCivs[row.CivID] = true
			switch row.CivName {
			case "Mapuche":
				report.Summary.MapucheRazesToVillager = row.RazesToVillager
			case "Khmer":
				report.Summary.KhmerRazesToVillager = row.RazesToVillager
			}
		}
		report.Rows = append(report.Rows, row)
		bucket := seenBuckets[row.RazesToVillager]
		if bucket == nil {
			bucket = &TriggerRazeBucket{RazesToVillager: row.RazesToVillager}
			seenBuckets[row.RazesToVillager] = bucket
		}
		bucket.TechIDs = append(bucket.TechIDs, techID)
		if row.CivID > 0 {
			bucket.CivIDs = append(bucket.CivIDs, row.CivID)
			bucket.CivNames = append(bucket.CivNames, row.CivName)
		}
	}
	sort.Slice(report.Rows, func(i, j int) bool {
		if report.Rows[i].RazesToVillager != report.Rows[j].RazesToVillager {
			return report.Rows[i].RazesToVillager < report.Rows[j].RazesToVillager
		}
		if report.Rows[i].CivID != report.Rows[j].CivID {
			return report.Rows[i].CivID < report.Rows[j].CivID
		}
		return report.Rows[i].TechID < report.Rows[j].TechID
	})
	for _, bucket := range seenBuckets {
		sort.Ints(bucket.TechIDs)
		sort.Ints(bucket.CivIDs)
		bucket.CivIDs = dedupeInts(bucket.CivIDs)
		sort.Strings(bucket.CivNames)
		bucket.CivNames = dedupeStrings(bucket.CivNames)
		bucket.Count = len(bucket.TechIDs)
		report.Buckets = append(report.Buckets, *bucket)
	}
	sort.Slice(report.Buckets, func(i, j int) bool {
		return report.Buckets[i].RazesToVillager < report.Buckets[j].RazesToVillager
	})
	sort.Slice(report.Conflicts, func(i, j int) bool {
		return report.Conflicts[i].TechID < report.Conflicts[j].TechID
	})
	report.Summary.Rows = len(report.Rows)
	report.Summary.UniqueTechs = len(candidates)
	report.Summary.ResolvedCivs = len(seenCivs)
	report.Summary.Buckets = len(report.Buckets)
	report.Summary.Conflicts = len(report.Conflicts)
	if dat == nil {
		report.Warnings = append(report.Warnings, "no --dat supplied; technology ids are decoded but civ names cannot be cross-referenced")
	}
	if len(report.Conflicts) > 0 {
		report.Warnings = append(report.Warnings, "one or more technology ids map to multiple raze counts; rows with conflicts are withheld")
	}
	return report, nil
}

func parseTriggerRazeRung(triggerID int, trigger map[string]any) (triggerRazeRung, bool) {
	if !hasRazeCondition(trigger) || !hasRazeConsumeEffect(trigger) {
		return triggerRazeRung{}, false
	}
	message := firstTriggerMessage(trigger)
	if message == "" {
		return triggerRazeRung{}, false
	}
	razes, ok := razesFromRungMessage(message)
	if !ok {
		return triggerRazeRung{}, false
	}
	return triggerRazeRung{
		TriggerID:       triggerID,
		Player:          playerFromRazeMessage(message),
		RazesToVillager: razes,
		Message:         message,
	}, true
}

func hasRazeCondition(trigger map[string]any) bool {
	for _, condition := range mapSlice(trigger, "conditions") {
		fields := intSlice(condition, "fields")
		if len(fields) > 7 && fields[0] == 8 && fields[2] == 1 && fields[3] == 43 && fields[7] >= 1 && fields[7] <= 8 {
			return true
		}
	}
	return false
}

func hasRazeConsumeEffect(trigger map[string]any) bool {
	for _, effect := range mapSlice(trigger, "effects") {
		fields := intSlice(effect, "fields_prefix")
		if len(fields) > 9 && fields[0] == 52 && fields[3] == 1 && fields[4] == 43 && fields[9] >= 1 && fields[9] <= 8 {
			return true
		}
	}
	return false
}

func activateTargets(trigger map[string]any) []int {
	seen := map[int]bool{}
	for _, effect := range mapSlice(trigger, "effects") {
		fields := intSlice(effect, "fields_prefix")
		if len(fields) > 15 && fields[0] == 8 && fields[15] >= 0 {
			seen[fields[15]] = true
		}
	}
	out := make([]int, 0, len(seen))
	for id := range seen {
		out = append(out, id)
	}
	sort.Ints(out)
	return out
}

func researchTechConditions(trigger map[string]any) []int {
	seen := map[int]bool{}
	for _, condition := range mapSlice(trigger, "conditions") {
		fields := intSlice(condition, "fields")
		if len(fields) > 8 && fields[0] == 9 && fields[8] >= 0 {
			seen[fields[8]] = true
		}
	}
	out := make([]int, 0, len(seen))
	for id := range seen {
		out = append(out, id)
	}
	sort.Ints(out)
	return out
}

var razeRemainingRE = regexp.MustCompile(`([0-9]+) Raze[s]? Remaining`)
var playerRazeRE = regexp.MustCompile(`P([0-9]+) - `)

func razesFromRungMessage(message string) (int, bool) {
	if strings.Contains(message, "Raze (Villager)") {
		return 1, true
	}
	m := razeRemainingRE.FindStringSubmatch(message)
	if len(m) != 2 {
		return 0, false
	}
	remaining, err := strconv.Atoi(m[1])
	if err != nil {
		return 0, false
	}
	return remaining + 1, true
}

func playerFromRazeMessage(message string) int {
	m := playerRazeRE.FindStringSubmatch(message)
	if len(m) != 2 {
		return 0
	}
	player, _ := strconv.Atoi(m[1])
	return player
}

func firstTriggerMessage(trigger map[string]any) string {
	for _, effect := range mapSlice(trigger, "effects") {
		if msg, ok := effect["message"].(string); ok && msg != "" {
			return msg
		}
	}
	return ""
}

func mapSlice(value map[string]any, key string) []map[string]any {
	raw, _ := value[key].([]map[string]any)
	return raw
}

func intSlice(value map[string]any, key string) []int {
	raw, _ := value[key].([]int)
	return raw
}

func sortedPlayerKeys(players map[int]bool) []int {
	out := make([]int, 0, len(players))
	for player := range players {
		out = append(out, player)
	}
	sort.Ints(out)
	return out
}

func dedupeStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := values[:0]
	prev := ""
	for i, value := range values {
		if i == 0 || value != prev {
			out = append(out, value)
			prev = value
		}
	}
	return out
}

func dedupeInts(values []int) []int {
	if len(values) == 0 {
		return nil
	}
	out := values[:0]
	prev := 0
	for i, value := range values {
		if i == 0 || value != prev {
			out = append(out, value)
			prev = value
		}
	}
	return out
}
