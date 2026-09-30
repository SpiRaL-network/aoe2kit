package cba

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"aoe2kit/pkg/datfile"
	"aoe2kit/pkg/replay"
)

type PhaseFactsReport struct {
	Path         string            `json:"path,omitempty"`
	DatPath      string            `json:"dat_path,omitempty"`
	Method       string            `json:"method"`
	Verification string            `json:"verification"`
	Summary      PhaseFactsSummary `json:"summary"`
	Rungs        []PhaseRungRow    `json:"rungs,omitempty"`
	Civs         []PhaseCivFact    `json:"civs,omitempty"`
	Warnings     []string          `json:"warnings,omitempty"`
}

type PhaseFactsSummary struct {
	TriggerCount      int  `json:"trigger_count"`
	CastleRungs       int  `json:"castle_rungs"`
	ImperialRungs     int  `json:"imperial_rungs"`
	ActivatorTriggers int  `json:"activator_triggers"`
	ConditionGroups   int  `json:"condition_groups"`
	CivFacts          int  `json:"civ_facts"`
	DefaultInferred   int  `json:"default_inferred"`
	RazeFactsJoined   int  `json:"raze_facts_joined"`
	LaneSymmetryOK    bool `json:"lane_symmetry_ok"`
	UnresolvedGroups  int  `json:"unresolved_groups"`
	MultiCivGroups    int  `json:"multi_civ_groups"`
}

type PhaseRungRow struct {
	Age             string                `json:"age"`
	ThresholdKills  int                   `json:"threshold_kills"`
	Player          int                   `json:"player"`
	RungTriggerID   int                   `json:"rung_trigger_id"`
	ActivatorIDs    []int                 `json:"activator_trigger_ids,omitempty"`
	ConditionGroups []PhaseConditionGroup `json:"condition_groups,omitempty"`
	Confidence      string                `json:"confidence"`
}

type PhaseConditionGroup struct {
	ActivatorTriggerID int             `json:"activator_trigger_id"`
	Terms              []PhaseTechTerm `json:"terms"`
	ResolvedCivIDs     []int           `json:"resolved_civ_ids,omitempty"`
	ResolvedCivNames   []string        `json:"resolved_civ_names,omitempty"`
	Confidence         string          `json:"confidence"`
}

type PhaseTechTerm struct {
	TechnologyID   int    `json:"technology_id"`
	TechnologyName string `json:"technology_name,omitempty"`
	CivID          int    `json:"civ_id,omitempty"`
	CivName        string `json:"civ_name,omitempty"`
	Inverted       bool   `json:"inverted,omitempty"`
}

type PhaseCivFact struct {
	CivID             int    `json:"civ_id"`
	CivName           string `json:"civ_name"`
	CastleThreshold   int    `json:"castle_threshold_kills"`
	ImperialThreshold int    `json:"imperial_threshold_kills"`
	RazesToVillager   int    `json:"razes_to_villager,omitempty"`
	CastleEvidence    string `json:"castle_evidence"`
	ImperialEvidence  string `json:"imperial_evidence"`
	RazeEvidence      string `json:"raze_evidence,omitempty"`
	Confidence        string `json:"confidence"`
}

type phaseRung struct {
	Age       string
	Threshold int
	Player    int
	TriggerID int
}

func BuildPhaseFacts(path, datPath string) (*PhaseFactsReport, error) {
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
	report := &PhaseFactsReport{
		Path:         path,
		DatPath:      datPath,
		Method:       "cba_requiem_v292_civ_phase_threshold_fact_decode",
		Verification: "structure_verified_trigger_graph_plus_dat_tech_name_cross_reference_not_engine_runtime_verified",
	}
	report.Summary.TriggerCount = rec.TriggerGraph.TriggerCount
	rungs := map[int]phaseRung{}
	for triggerID, trigger := range rec.TriggerGraph.Triggers {
		if rung, ok := parsePhaseRung(triggerID, trigger); ok {
			rungs[triggerID] = rung
			if rung.Age == "castle" {
				report.Summary.CastleRungs++
			} else if rung.Age == "imperial" {
				report.Summary.ImperialRungs++
			}
		}
	}

	byRung := map[int][]PhaseConditionGroup{}
	activators := map[int]bool{}
	for triggerID, trigger := range rec.TriggerGraph.Triggers {
		for _, target := range activateTargets(trigger) {
			if _, ok := rungs[target]; !ok {
				continue
			}
			groups := phaseConditionGroups(triggerID, trigger, dat)
			if len(groups) == 0 {
				continue
			}
			activators[triggerID] = true
			byRung[target] = append(byRung[target], groups...)
			report.Summary.ConditionGroups += len(groups)
			for _, group := range groups {
				switch group.Confidence {
				case "resolved_single_civ_marker":
				case "resolved_multi_civ_or_combo_marker":
					report.Summary.MultiCivGroups++
				default:
					report.Summary.UnresolvedGroups++
				}
			}
		}
	}
	report.Summary.ActivatorTriggers = len(activators)

	castleByCiv := map[int]PhaseCivFact{}
	imperialByCiv := map[int]PhaseCivFact{}
	explicitCastle := map[int]bool{}
	explicitImperial := map[int]bool{}
	civUniverse := map[int]string{}

	for triggerID, rung := range rungs {
		row := PhaseRungRow{
			Age:             rung.Age,
			ThresholdKills:  rung.Threshold,
			Player:          rung.Player,
			RungTriggerID:   triggerID,
			ConditionGroups: byRung[triggerID],
			Confidence:      "trigger_researches_age_tech_at_authored_kill_threshold",
		}
		for _, group := range row.ConditionGroups {
			row.ActivatorIDs = append(row.ActivatorIDs, group.ActivatorTriggerID)
			for _, civID := range group.ResolvedCivIDs {
				name := replay.CivDisplayName(civID)
				if name == "" {
					name = fmt.Sprintf("civ_%d", civID)
				}
				civUniverse[civID] = name
				fact := PhaseCivFact{CivID: civID, CivName: name, Confidence: group.Confidence}
				if rung.Age == "castle" {
					fact.CastleThreshold = rung.Threshold
					fact.CastleEvidence = fmt.Sprintf("rung %d activated by %d", triggerID, group.ActivatorTriggerID)
					castleByCiv[civID] = fact
					explicitCastle[civID] = true
				} else if rung.Age == "imperial" {
					fact.ImperialThreshold = rung.Threshold
					fact.ImperialEvidence = fmt.Sprintf("rung %d activated by %d", triggerID, group.ActivatorTriggerID)
					imperialByCiv[civID] = fact
					explicitImperial[civID] = true
				}
			}
		}
		row.ActivatorIDs = dedupeIntsSorted(row.ActivatorIDs)
		report.Rungs = append(report.Rungs, row)
	}

	razeByCiv := map[int]TriggerRazeRow{}
	if datPath != "" {
		razeReport, err := BuildTriggerRazes(path, datPath)
		if err == nil {
			for _, row := range razeReport.Rows {
				if row.CivID > 0 {
					razeByCiv[row.CivID] = row
					civUniverse[row.CivID] = row.CivName
				}
			}
		} else {
			report.Warnings = append(report.Warnings, "raze requirement join failed: "+err.Error())
		}
	}
	if dat == nil {
		report.Warnings = append(report.Warnings, "no --dat supplied; phase rung tech ids are decoded but civ names and default complements cannot be fully resolved")
	}

	defaultCastle, defaultImperial := defaultPhaseThresholds(rungs)
	for civID, name := range civUniverse {
		fact := PhaseCivFact{CivID: civID, CivName: name, Confidence: "structure_verified"}
		if row, ok := castleByCiv[civID]; ok {
			fact.CastleThreshold = row.CastleThreshold
			fact.CastleEvidence = row.CastleEvidence
		} else if defaultCastle > 0 {
			fact.CastleThreshold = defaultCastle
			fact.CastleEvidence = "inferred default: no civ activator for another Castle rung"
			report.Summary.DefaultInferred++
		}
		if row, ok := imperialByCiv[civID]; ok {
			fact.ImperialThreshold = row.ImperialThreshold
			fact.ImperialEvidence = row.ImperialEvidence
		} else if defaultImperial > 0 {
			fact.ImperialThreshold = defaultImperial
			fact.ImperialEvidence = "inferred default: no civ activator for another Imperial rung"
			report.Summary.DefaultInferred++
		}
		if raze, ok := razeByCiv[civID]; ok {
			fact.RazesToVillager = raze.RazesToVillager
			fact.RazeEvidence = fmt.Sprintf("rung %d activated by %d", raze.TargetTriggerID, raze.TriggerID)
			report.Summary.RazeFactsJoined++
		}
		report.Civs = append(report.Civs, fact)
	}

	sort.Slice(report.Rungs, func(i, j int) bool {
		if report.Rungs[i].Player != report.Rungs[j].Player {
			return report.Rungs[i].Player < report.Rungs[j].Player
		}
		if report.Rungs[i].Age != report.Rungs[j].Age {
			return report.Rungs[i].Age < report.Rungs[j].Age
		}
		return report.Rungs[i].ThresholdKills < report.Rungs[j].ThresholdKills
	})
	sort.Slice(report.Civs, func(i, j int) bool {
		if report.Civs[i].CivName != report.Civs[j].CivName {
			return report.Civs[i].CivName < report.Civs[j].CivName
		}
		return report.Civs[i].CivID < report.Civs[j].CivID
	})
	report.Summary.CivFacts = len(report.Civs)
	report.Summary.LaneSymmetryOK = phaseLaneSymmetryOK(rungs)
	if report.Summary.UnresolvedGroups > 0 {
		report.Warnings = append(report.Warnings, fmt.Sprintf("%d condition groups could not be resolved to a civ from positive civ-marker techs; they are retained as grouped evidence and not used for single-civ facts", report.Summary.UnresolvedGroups))
	}
	if report.Summary.MultiCivGroups > 0 {
		report.Warnings = append(report.Warnings, fmt.Sprintf("%d condition groups resolved to multiple civs; grouped evidence is retained because these may be combo guards", report.Summary.MultiCivGroups))
	}
	if report.Summary.RazeFactsJoined < report.Summary.CivFacts {
		report.Warnings = append(report.Warnings, fmt.Sprintf("%d civ fact rows have no joined raze requirement from the decoded V292 raze ladder", report.Summary.CivFacts-report.Summary.RazeFactsJoined))
	}
	return report, nil
}

var phaseKillRE = regexp.MustCompile(`(?i)([0-9]+)\s+kills?\s+\((castle|imperial)\s+age\)`)

func parsePhaseRung(triggerID int, trigger map[string]any) (phaseRung, bool) {
	age := ""
	for _, effect := range mapSlice(trigger, "effects") {
		fields := intSlice(effect, "fields_prefix")
		if len(fields) > 11 && fields[0] == 2 {
			switch fields[11] {
			case 102:
				age = "castle"
			case 103:
				age = "imperial"
			}
		}
	}
	if age == "" {
		return phaseRung{}, false
	}
	for _, msg := range triggerMessages(trigger) {
		m := phaseKillRE.FindStringSubmatch(msg)
		if len(m) != 3 {
			continue
		}
		threshold := atoiDefault(m[1])
		msgAge := strings.ToLower(m[2])
		if threshold > 0 && msgAge == age {
			return phaseRung{Age: age, Threshold: threshold, Player: playerFromColorTag(msg), TriggerID: triggerID}, true
		}
	}
	return phaseRung{}, false
}

func phaseConditionGroups(triggerID int, trigger map[string]any, dat *datfile.Index) []PhaseConditionGroup {
	var groups []PhaseConditionGroup
	current := PhaseConditionGroup{ActivatorTriggerID: triggerID}
	flush := func() {
		if len(current.Terms) == 0 {
			return
		}
		resolvePhaseGroup(&current)
		groups = append(groups, current)
		current = PhaseConditionGroup{ActivatorTriggerID: triggerID}
	}
	for _, condition := range mapSlice(trigger, "conditions") {
		fields := intSlice(condition, "fields")
		if len(fields) == 0 {
			continue
		}
		switch fields[0] {
		case 29:
			flush()
		case 57:
			continue
		case 9:
			if len(fields) <= 18 || fields[8] < 0 {
				continue
			}
			term := PhaseTechTerm{
				TechnologyID: fields[8],
				Inverted:     fields[18] == 1,
			}
			if dat != nil {
				if tech, ok := dat.Tech(term.TechnologyID); ok {
					term.TechnologyName = tech.Name
					term.CivID = int(tech.Civ)
					if term.CivID > 0 {
						term.CivName = replay.CivDisplayName(term.CivID)
					}
				}
			}
			current.Terms = append(current.Terms, term)
		}
	}
	flush()
	return groups
}

func resolvePhaseGroup(group *PhaseConditionGroup) {
	civs := map[int]string{}
	for _, term := range group.Terms {
		if term.Inverted || term.CivID <= 0 {
			continue
		}
		name := term.CivName
		if name == "" {
			name = fmt.Sprintf("civ_%d", term.CivID)
		}
		civs[term.CivID] = name
	}
	for id := range civs {
		group.ResolvedCivIDs = append(group.ResolvedCivIDs, id)
	}
	sort.Ints(group.ResolvedCivIDs)
	for _, id := range group.ResolvedCivIDs {
		group.ResolvedCivNames = append(group.ResolvedCivNames, civs[id])
	}
	switch len(group.ResolvedCivIDs) {
	case 0:
		group.Confidence = "unresolved_or_negative_only_combo"
	case 1:
		group.Confidence = "resolved_single_civ_marker"
	default:
		group.Confidence = "resolved_multi_civ_or_combo_marker"
	}
}

func defaultPhaseThresholds(rungs map[int]phaseRung) (castle int, imperial int) {
	for _, rung := range rungs {
		if rung.Player != 1 {
			continue
		}
		switch rung.Age {
		case "castle":
			if castle == 0 || rung.Threshold < castle {
				castle = rung.Threshold
			}
		case "imperial":
			if imperial == 0 || rung.Threshold < imperial {
				imperial = rung.Threshold
			}
		}
	}
	return castle, imperial
}

func phaseLaneSymmetryOK(rungs map[int]phaseRung) bool {
	byPlayer := map[int]map[string]map[int]bool{}
	for _, rung := range rungs {
		if rung.Player <= 0 {
			continue
		}
		if byPlayer[rung.Player] == nil {
			byPlayer[rung.Player] = map[string]map[int]bool{}
		}
		if byPlayer[rung.Player][rung.Age] == nil {
			byPlayer[rung.Player][rung.Age] = map[int]bool{}
		}
		byPlayer[rung.Player][rung.Age][rung.Threshold] = true
	}
	ref := byPlayer[1]
	if len(ref) == 0 {
		return false
	}
	for player := 2; player <= 8; player++ {
		if !samePhaseSet(ref, byPlayer[player]) {
			return false
		}
	}
	return true
}

func samePhaseSet(a, b map[string]map[int]bool) bool {
	for _, age := range []string{"castle", "imperial"} {
		if len(a[age]) != len(b[age]) {
			return false
		}
		for value := range a[age] {
			if !b[age][value] {
				return false
			}
		}
	}
	return true
}

func atoiDefault(value string) int {
	var out int
	for _, ch := range value {
		if ch < '0' || ch > '9' {
			return out
		}
		out = out*10 + int(ch-'0')
	}
	return out
}

func dedupeIntsSorted(values []int) []int {
	sort.Ints(values)
	return dedupeInts(values)
}
