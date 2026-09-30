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

type TriggerSpawnReport struct {
	Path         string                 `json:"path,omitempty"`
	DatPath      string                 `json:"dat_path,omitempty"`
	Method       string                 `json:"method"`
	Verification string                 `json:"verification"`
	Summary      TriggerSpawnSummary    `json:"summary"`
	Rows         []TriggerSpawnRow      `json:"rows,omitempty"`
	Buckets      []TriggerSpawnBucket   `json:"buckets,omitempty"`
	Mechanics    TriggerSpawnMechanics  `json:"mechanics"`
	Conflicts    []TriggerSpawnConflict `json:"conflicts,omitempty"`
	Warnings     []string               `json:"warnings,omitempty"`
}

type TriggerSpawnSummary struct {
	TriggerCount           int `json:"trigger_count"`
	SpawnerSecondTriggers  int `json:"spawner_second_triggers"`
	SpawnCountTriggers     int `json:"spawn_count_triggers"`
	Rows                   int `json:"rows"`
	CompleteRows           int `json:"complete_rows"`
	ResolvedCivs           int `json:"resolved_civs"`
	UniqueTechs            int `json:"unique_techs"`
	Buckets                int `json:"buckets"`
	Conflicts              int `json:"conflicts"`
	UnresolvedTechs        int `json:"unresolved_techs"`
	MapucheSpawnSeconds    int `json:"mapuche_spawn_seconds,omitempty"`
	MapucheSpawnUnitCount  int `json:"mapuche_spawn_unit_count,omitempty"`
	KhmerSpawnSeconds      int `json:"khmer_spawn_seconds,omitempty"`
	KhmerSpawnUnitCount    int `json:"khmer_spawn_unit_count,omitempty"`
	PersiansSpawnSeconds   int `json:"persians_spawn_seconds,omitempty"`
	PersiansSpawnUnitCount int `json:"persians_spawn_unit_count,omitempty"`
}

type TriggerSpawnRow struct {
	CivID          int    `json:"civ_id,omitempty"`
	CivName        string `json:"civ_name,omitempty"`
	TechID         int    `json:"technology_id"`
	TechName       string `json:"technology_name,omitempty"`
	SpawnSeconds   *int   `json:"spawn_seconds,omitempty"`
	SpawnUnitCount *int   `json:"spawn_unit_count,omitempty"`
	SecondsTrigger int    `json:"seconds_trigger_id,omitempty"`
	CountTrigger   int    `json:"count_trigger_id,omitempty"`
	SourcePlayers  []int  `json:"source_players,omitempty"`
	Message        string `json:"message,omitempty"`
	Confidence     string `json:"confidence"`
}

type TriggerSpawnBucket struct {
	SpawnSeconds   *int     `json:"spawn_seconds,omitempty"`
	SpawnUnitCount *int     `json:"spawn_unit_count,omitempty"`
	TechIDs        []int    `json:"technology_ids"`
	CivIDs         []int    `json:"civ_ids,omitempty"`
	CivNames       []string `json:"civ_names,omitempty"`
	Count          int      `json:"count"`
}

type TriggerSpawnMechanics struct {
	SecondsSource       string `json:"seconds_source"`
	CountConditionType  int    `json:"count_condition_type"`
	CountConditionName  string `json:"count_condition_name"`
	SpawnEffectTypes    []int  `json:"spawn_effect_types"`
	SpawnEffectNames    string `json:"spawn_effect_names"`
	TechnologyCondition int    `json:"technology_condition"`
	Notes               string `json:"notes"`
}

type TriggerSpawnConflict struct {
	TechID int    `json:"technology_id"`
	Field  string `json:"field"`
	Values []int  `json:"values"`
}

type triggerSpawnCandidate struct {
	TechID        int
	Seconds       map[int]bool
	Counts        map[int]bool
	SecondsTrig   map[int]bool
	CountTrig     map[int]bool
	Messages      []string
	SourcePlayers map[int]bool
}

func BuildTriggerSpawns(path, datPath string) (*TriggerSpawnReport, error) {
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
	report := &TriggerSpawnReport{
		Path:         path,
		DatPath:      datPath,
		Method:       "cba_requiem_trigger_graph_spawn_seconds_and_wave_count_decode",
		Verification: "structure_verified_trigger_graph_plus_dat_tech_name_cross_reference_not_engine_runtime_verified",
		Mechanics: TriggerSpawnMechanics{
			SecondsSource:       "display-instruction messages matching `Spawner N sec` on civ-gated triggers",
			CountConditionType:  4,
			CountConditionName:  "CBA spawn-wave quantity guard inferred from civ-gated spawn production triggers",
			SpawnEffectTypes:    []int{11, 49, 56},
			SpawnEffectNames:    "CREATE_OBJECT / CHANGE_OWNERSHIP / AI_SCRIPT_GOAL-style relay marker as parsed by trigger graph",
			TechnologyCondition: 9,
			Notes:               "Intervals are read from authored trigger messages; unit counts are inferred from condition type 4 field[2] on repeated four-lane spawn triggers keyed by the same civ technology.",
		},
	}
	report.Summary.TriggerCount = rec.TriggerGraph.TriggerCount
	candidates := map[int]*triggerSpawnCandidate{}
	for triggerID, trigger := range rec.TriggerGraph.Triggers {
		if seconds, ok := spawnerSeconds(trigger); ok {
			report.Summary.SpawnerSecondTriggers++
			techIDs := researchTechConditions(trigger)
			for _, techID := range techIDs {
				c := spawnCandidate(candidates, techID)
				c.Seconds[seconds] = true
				c.SecondsTrig[triggerID] = true
				if msg := firstTriggerMessage(trigger); msg != "" {
					c.Messages = append(c.Messages, msg)
					if player := playerFromColorTag(msg); player > 0 {
						c.SourcePlayers[player] = true
					}
				}
			}
		}
		if count, ok := spawnUnitCount(trigger); ok {
			report.Summary.SpawnCountTriggers++
			techIDs := researchTechConditions(trigger)
			for _, techID := range techIDs {
				c := spawnCandidate(candidates, techID)
				c.Counts[count] = true
				c.CountTrig[triggerID] = true
			}
		}
	}
	seenCivs := map[int]bool{}
	seenBuckets := map[string]*TriggerSpawnBucket{}
	for techID, c := range candidates {
		secondsValues := sortedIntSet(c.Seconds)
		countValues := sortedIntSet(c.Counts)
		if len(secondsValues) > 1 {
			report.Conflicts = append(report.Conflicts, TriggerSpawnConflict{TechID: techID, Field: "spawn_seconds", Values: secondsValues})
		}
		if len(countValues) > 1 {
			report.Conflicts = append(report.Conflicts, TriggerSpawnConflict{TechID: techID, Field: "spawn_unit_count", Values: countValues})
		}
		if len(secondsValues) > 1 || len(countValues) > 1 {
			continue
		}
		row := TriggerSpawnRow{
			TechID:         techID,
			SecondsTrigger: firstIntKey(c.SecondsTrig),
			CountTrigger:   firstIntKey(c.CountTrig),
			SourcePlayers:  sortedPlayerKeys(c.SourcePlayers),
			Message:        firstNonEmpty(c.Messages),
			Confidence:     "trigger_graph_civ_tech_spawn_parameter_decode",
		}
		if len(secondsValues) == 1 {
			row.SpawnSeconds = spawnIntPtr(secondsValues[0])
		}
		if len(countValues) == 1 {
			row.SpawnUnitCount = spawnIntPtr(countValues[0])
		}
		if row.SpawnSeconds != nil && row.SpawnUnitCount != nil {
			report.Summary.CompleteRows++
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
				if row.SpawnSeconds != nil {
					report.Summary.MapucheSpawnSeconds = *row.SpawnSeconds
				}
				if row.SpawnUnitCount != nil {
					report.Summary.MapucheSpawnUnitCount = *row.SpawnUnitCount
				}
			case "Khmer":
				if row.SpawnSeconds != nil {
					report.Summary.KhmerSpawnSeconds = *row.SpawnSeconds
				}
				if row.SpawnUnitCount != nil {
					report.Summary.KhmerSpawnUnitCount = *row.SpawnUnitCount
				}
			case "Persians":
				if row.SpawnSeconds != nil {
					report.Summary.PersiansSpawnSeconds = *row.SpawnSeconds
				}
				if row.SpawnUnitCount != nil {
					report.Summary.PersiansSpawnUnitCount = *row.SpawnUnitCount
				}
			}
		}
		report.Rows = append(report.Rows, row)
		key := spawnBucketKey(row.SpawnSeconds, row.SpawnUnitCount)
		bucket := seenBuckets[key]
		if bucket == nil {
			bucket = &TriggerSpawnBucket{SpawnSeconds: row.SpawnSeconds, SpawnUnitCount: row.SpawnUnitCount}
			seenBuckets[key] = bucket
		}
		bucket.TechIDs = append(bucket.TechIDs, techID)
		if row.CivID > 0 {
			bucket.CivIDs = append(bucket.CivIDs, row.CivID)
			bucket.CivNames = append(bucket.CivNames, row.CivName)
		}
	}
	sort.Slice(report.Rows, func(i, j int) bool {
		si, sj := ptrValue(report.Rows[i].SpawnSeconds), ptrValue(report.Rows[j].SpawnSeconds)
		if si != sj {
			return si < sj
		}
		ci, cj := ptrValue(report.Rows[i].SpawnUnitCount), ptrValue(report.Rows[j].SpawnUnitCount)
		if ci != cj {
			return ci < cj
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
		si, sj := ptrValue(report.Buckets[i].SpawnSeconds), ptrValue(report.Buckets[j].SpawnSeconds)
		if si != sj {
			return si < sj
		}
		ci, cj := ptrValue(report.Buckets[i].SpawnUnitCount), ptrValue(report.Buckets[j].SpawnUnitCount)
		return ci < cj
	})
	sort.Slice(report.Conflicts, func(i, j int) bool {
		if report.Conflicts[i].TechID != report.Conflicts[j].TechID {
			return report.Conflicts[i].TechID < report.Conflicts[j].TechID
		}
		return report.Conflicts[i].Field < report.Conflicts[j].Field
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
		report.Warnings = append(report.Warnings, "one or more technology ids map to multiple spawn values; rows with conflicts are withheld")
	}
	report.Warnings = append(report.Warnings, "unit counts are inferred from trigger structure and need engine/corpus validation before being treated as final balance truth")
	return report, nil
}

func spawnCandidate(candidates map[int]*triggerSpawnCandidate, techID int) *triggerSpawnCandidate {
	c := candidates[techID]
	if c == nil {
		c = &triggerSpawnCandidate{
			TechID:        techID,
			Seconds:       map[int]bool{},
			Counts:        map[int]bool{},
			SecondsTrig:   map[int]bool{},
			CountTrig:     map[int]bool{},
			SourcePlayers: map[int]bool{},
		}
		candidates[techID] = c
	}
	return c
}

var spawnerSecondsRE = regexp.MustCompile(`Spawner ([0-9]+) sec`)

func spawnerSeconds(trigger map[string]any) (int, bool) {
	for _, effect := range mapSlice(trigger, "effects") {
		msg, _ := effect["message"].(string)
		m := spawnerSecondsRE.FindStringSubmatch(msg)
		if len(m) != 2 {
			continue
		}
		seconds, err := strconv.Atoi(m[1])
		if err == nil {
			return seconds, true
		}
	}
	return 0, false
}

func spawnUnitCount(trigger map[string]any) (int, bool) {
	if len(researchTechConditions(trigger)) == 0 {
		return 0, false
	}
	createOrOwn := 0
	hasRelay := false
	for _, effect := range mapSlice(trigger, "effects") {
		fields := intSlice(effect, "fields_prefix")
		if len(fields) == 0 {
			continue
		}
		switch fields[0] {
		case 11, 49:
			createOrOwn++
		case 56:
			hasRelay = true
		}
	}
	if createOrOwn < 4 || !hasRelay {
		return 0, false
	}
	seen := map[int]bool{}
	for _, condition := range mapSlice(trigger, "conditions") {
		fields := intSlice(condition, "fields")
		if len(fields) > 2 && fields[0] == 4 && fields[2] > 0 {
			seen[fields[2]] = true
		}
	}
	values := sortedIntSet(seen)
	if len(values) != 1 {
		return 0, false
	}
	return values[0], true
}

func playerFromColorTag(message string) int {
	switch {
	case strings.Contains(message, "<BLUE>"):
		return 1
	case strings.Contains(message, "<RED>"):
		return 2
	case strings.Contains(message, "<GREEN>"):
		return 3
	case strings.Contains(message, "<YELLOW>"):
		return 4
	case strings.Contains(message, "<AQUA>"):
		return 5
	case strings.Contains(message, "<PURPLE>"):
		return 6
	case strings.Contains(message, "<GREY>"):
		return 7
	case strings.Contains(message, "<ORANGE>"):
		return 8
	default:
		return 0
	}
}

func sortedIntSet(values map[int]bool) []int {
	out := make([]int, 0, len(values))
	for value := range values {
		out = append(out, value)
	}
	sort.Ints(out)
	return out
}

func firstIntKey(values map[int]bool) int {
	out := sortedIntSet(values)
	if len(out) == 0 {
		return 0
	}
	return out[0]
}

func firstNonEmpty(values []string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func spawnIntPtr(value int) *int {
	v := value
	return &v
}

func ptrValue(value *int) int {
	if value == nil {
		return -1
	}
	return *value
}

func spawnBucketKey(seconds, count *int) string {
	return fmt.Sprintf("%d/%d", ptrValue(seconds), ptrValue(count))
}
