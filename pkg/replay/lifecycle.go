package replay

import (
	"fmt"
	"math"
	"sort"
)

const (
	defaultLifecycleSpawnRadius    = 20.0
	defaultLifecycleSmallSelection = 2
	defaultLifecycleBuildSelection = 4
	lifecycleMoveRadius            = 12.0
	lifecycleBuildRadius           = 6.0
)

type LifecycleOptions struct {
	SpawnRadius       float64
	MaxMoveSelection  int
	MaxBuildSelection int
}

type LifecycleReport struct {
	Path         string                     `json:"path,omitempty"`
	Method       string                     `json:"method"`
	Verification string                     `json:"verification"`
	Summary      LifecycleSummary           `json:"summary"`
	Players      []LifecyclePlayerSummary   `json:"players,omitempty"`
	FirstSeen    []LifecycleFirstSeen       `json:"first_seen,omitempty"`
	Candidates   []LifecycleObjectCandidate `json:"candidates,omitempty"`
	Warnings     []string                   `json:"warnings,omitempty"`
}

type LifecycleSummary struct {
	FirstSeenObjects              int `json:"first_seen_objects"`
	SpawnRecipes                  int `json:"spawn_recipes"`
	SpawnMatchedCandidates        int `json:"spawn_matched_candidates"`
	VillagerCandidates            int `json:"villager_candidates"`
	PlayersWithVillagerCandidate  int `json:"players_with_villager_candidate"`
	OvermatchedNearVillagerEvents int `json:"overmatched_near_villager_events"`
}

type LifecyclePlayerSummary struct {
	PlayerID             int    `json:"player_id"`
	Label                string `json:"label"`
	FirstVillagerTimeMS  int    `json:"first_villager_time_ms,omitempty"`
	FirstVillagerTime    string `json:"first_villager_time,omitempty"`
	VillagerCandidates   int    `json:"villager_candidates"`
	SpawnMatchedObjects  int    `json:"spawn_matched_objects"`
	OvermatchedNearSpawn int    `json:"overmatched_near_spawn"`
}

type LifecycleFirstSeen struct {
	ObjectID      int     `json:"object_id"`
	PlayerID      int     `json:"player_id"`
	PlayerLabel   string  `json:"player_label,omitempty"`
	TimeMS        int     `json:"time_ms"`
	Time          string  `json:"time"`
	EventIndex    int     `json:"event_index"`
	Type          string  `json:"type"`
	ActionID      int     `json:"action_id,omitempty"`
	ActionName    string  `json:"action_name,omitempty"`
	X             float64 `json:"x,omitempty"`
	Y             float64 `json:"y,omitempty"`
	SelectedCount int     `json:"selected_count"`
}

type LifecycleObjectCandidate struct {
	ObjectID          int      `json:"object_id"`
	PlayerID          int      `json:"player_id"`
	PlayerLabel       string   `json:"player_label,omitempty"`
	TimeMS            int      `json:"time_ms"`
	Time              string   `json:"time"`
	EventIndex        int      `json:"event_index"`
	Type              string   `json:"type"`
	ActionID          int      `json:"action_id,omitempty"`
	ActionName        string   `json:"action_name,omitempty"`
	X                 float64  `json:"x,omitempty"`
	Y                 float64  `json:"y,omitempty"`
	SelectedCount     int      `json:"selected_count"`
	SpawnX            int      `json:"spawn_x"`
	SpawnY            int      `json:"spawn_y"`
	Distance          float64  `json:"distance"`
	UnitID            int      `json:"unit_id,omitempty"`
	UnitName          string   `json:"unit_name,omitempty"`
	CandidateUnitIDs  []int    `json:"candidate_unit_ids,omitempty"`
	CandidateUnitName []string `json:"candidate_unit_names,omitempty"`
	AmbiguousUnits    int      `json:"ambiguous_units,omitempty"`
	Kind              string   `json:"kind"`
	Confidence        string   `json:"confidence"`
	Reason            string   `json:"reason"`
}

type spawnPoint struct {
	PlayerID int
	X        int
	Y        int
	Recipes  []SpawnRecipe
}

func BuildLifecycle(path string, opts LifecycleOptions) (*LifecycleReport, error) {
	opts = normalizeLifecycleOptions(opts)
	events, spawns, err := lifecycleInputs(path)
	if err != nil {
		return nil, err
	}
	firstSeen := collectLifecycleFirstSeen(events)
	spawnPoints := groupSpawnPoints(spawns.Spawns)
	sort.Slice(firstSeen, func(i, j int) bool {
		if firstSeen[i].TimeMS != firstSeen[j].TimeMS {
			return firstSeen[i].TimeMS < firstSeen[j].TimeMS
		}
		return firstSeen[i].ObjectID < firstSeen[j].ObjectID
	})
	candidates, overmatched := matchLifecycleCandidates(firstSeen, spawnPoints, opts)
	players := summarizeLifecyclePlayers(candidates, overmatched)
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].TimeMS != candidates[j].TimeMS {
			return candidates[i].TimeMS < candidates[j].TimeMS
		}
		return candidates[i].ObjectID < candidates[j].ObjectID
	})
	villagers := 0
	playersWithVillagers := map[int]bool{}
	for _, candidate := range candidates {
		if candidate.Kind == "villager_spawn_candidate" {
			villagers++
			playersWithVillagers[candidate.PlayerID] = true
		}
	}
	report := &LifecycleReport{
		Path:         path,
		Method:       "event_first_seen_joined_to_trigger_spawn_catalog",
		Verification: "structure_verified_lifecycle_candidate_not_runtime_unit_type_proof",
		Summary: LifecycleSummary{
			FirstSeenObjects:              len(firstSeen),
			SpawnRecipes:                  spawns.Summary.PlausibleSpawns,
			SpawnMatchedCandidates:        len(candidates),
			VillagerCandidates:            villagers,
			PlayersWithVillagerCandidate:  len(playersWithVillagers),
			OvermatchedNearVillagerEvents: overmatchedVillagerEvents(overmatched),
		},
		Players:    players,
		FirstSeen:  firstSeen,
		Candidates: candidates,
	}
	if overmatchedVillagerEvents(overmatched) > 0 {
		report.Warnings = append(report.Warnings, fmt.Sprintf("%d near-villager-spawn first-seen events were overmatched by large selections and not typed as villagers", overmatchedVillagerEvents(overmatched)))
	}
	return report, nil
}

func normalizeLifecycleOptions(opts LifecycleOptions) LifecycleOptions {
	if opts.SpawnRadius <= 0 {
		opts.SpawnRadius = defaultLifecycleSpawnRadius
	}
	if opts.MaxMoveSelection <= 0 {
		opts.MaxMoveSelection = defaultLifecycleSmallSelection
	}
	if opts.MaxBuildSelection <= 0 {
		opts.MaxBuildSelection = defaultLifecycleBuildSelection
	}
	return opts
}

func lifecycleInputs(path string) ([]ReplayEvent, *SpawnCatalogReport, error) {
	eventReport, err := ExtractEvents(path, EventOptions{IncludeUntypedAction: true})
	if err != nil {
		return nil, nil, err
	}
	spawns, err := BuildSpawnCatalog(path)
	if err != nil {
		return nil, nil, err
	}
	return eventReport.Events, spawns, nil
}

func collectLifecycleFirstSeen(events []ReplayEvent) []LifecycleFirstSeen {
	seen := map[int]LifecycleFirstSeen{}
	for _, event := range events {
		if event.PlayerID <= 0 || len(event.ObjectIDs) == 0 || !plausibleXY(event.X, event.Y) {
			continue
		}
		selected := len(event.ObjectIDs)
		for _, objectID := range event.ObjectIDs {
			if objectID <= 0 {
				continue
			}
			if _, ok := seen[objectID]; ok {
				continue
			}
			seen[objectID] = LifecycleFirstSeen{
				ObjectID:      objectID,
				PlayerID:      event.PlayerID,
				PlayerLabel:   initialPlayerLabel(event.PlayerID),
				TimeMS:        event.TimeMS,
				Time:          FormatTime(event.TimeMS),
				EventIndex:    event.Index,
				Type:          event.Type,
				ActionID:      event.ActionID,
				ActionName:    event.ActionName,
				X:             event.X,
				Y:             event.Y,
				SelectedCount: selected,
			}
		}
	}
	out := make([]LifecycleFirstSeen, 0, len(seen))
	for _, first := range seen {
		out = append(out, first)
	}
	return out
}

func groupSpawnPoints(spawns []SpawnRecipe) []spawnPoint {
	byKey := map[string]*spawnPoint{}
	for _, recipe := range spawns {
		key := fmt.Sprintf("%d:%d:%d", recipe.TargetPlayer, recipe.X, recipe.Y)
		point := byKey[key]
		if point == nil {
			point = &spawnPoint{PlayerID: recipe.TargetPlayer, X: recipe.X, Y: recipe.Y}
			byKey[key] = point
		}
		point.Recipes = append(point.Recipes, recipe)
	}
	points := make([]spawnPoint, 0, len(byKey))
	for _, point := range byKey {
		sort.Slice(point.Recipes, func(i, j int) bool {
			return point.Recipes[i].UnitID < point.Recipes[j].UnitID
		})
		points = append(points, *point)
	}
	sort.Slice(points, func(i, j int) bool {
		if points[i].PlayerID != points[j].PlayerID {
			return points[i].PlayerID < points[j].PlayerID
		}
		if points[i].X != points[j].X {
			return points[i].X < points[j].X
		}
		return points[i].Y < points[j].Y
	})
	return points
}

func matchLifecycleCandidates(firstSeen []LifecycleFirstSeen, points []spawnPoint, opts LifecycleOptions) ([]LifecycleObjectCandidate, map[int]int) {
	var candidates []LifecycleObjectCandidate
	overmatched := map[int]int{}
	for _, first := range firstSeen {
		point, distance, ok := nearestSpawnPoint(first, points, lifecycleMatchRadius(first, opts))
		if !ok {
			continue
		}
		uniqueUnits, unitNames := uniqueSpawnUnits(point.Recipes)
		isVillagerPoint := len(uniqueUnits) == 1 && isVillagerUnit(uniqueUnits[0])
		if !smallLifecycleSelection(first, opts) {
			if isVillagerPoint {
				overmatched[first.PlayerID]++
			}
			continue
		}
		candidate := LifecycleObjectCandidate{
			ObjectID:          first.ObjectID,
			PlayerID:          first.PlayerID,
			PlayerLabel:       first.PlayerLabel,
			TimeMS:            first.TimeMS,
			Time:              first.Time,
			EventIndex:        first.EventIndex,
			Type:              first.Type,
			ActionID:          first.ActionID,
			ActionName:        first.ActionName,
			X:                 first.X,
			Y:                 first.Y,
			SelectedCount:     first.SelectedCount,
			SpawnX:            point.X,
			SpawnY:            point.Y,
			Distance:          round2(distance),
			CandidateUnitIDs:  uniqueUnits,
			CandidateUnitName: unitNames,
			AmbiguousUnits:    len(uniqueUnits),
			Kind:              "spawn_region_candidate",
			Confidence:        "first_seen_near_spawn_small_selection",
			Reason:            "first selected action appeared near a trigger create-object spawn point",
		}
		if len(uniqueUnits) == 1 {
			candidate.UnitID = uniqueUnits[0]
			candidate.UnitName = UnitDisplayName(candidate.UnitID)
			candidate.AmbiguousUnits = 0
			candidate.CandidateUnitIDs = nil
			candidate.CandidateUnitName = nil
			candidate.Confidence = "first_seen_near_unique_spawn_small_selection"
		}
		if isVillagerPoint {
			candidate.Kind = "villager_spawn_candidate"
			if first.Type == "build" {
				candidate.Reason = "first selected build action appeared near unique Villager spawn point"
			} else {
				candidate.Reason = "first selected move/order action appeared near unique Villager spawn point"
			}
		}
		candidates = append(candidates, candidate)
	}
	return candidates, overmatched
}

func nearestSpawnPoint(first LifecycleFirstSeen, points []spawnPoint, radius float64) (spawnPoint, float64, bool) {
	var best spawnPoint
	bestDistance := radius
	found := false
	for _, point := range points {
		if point.PlayerID != first.PlayerID {
			continue
		}
		distance := math.Hypot(first.X-float64(point.X), first.Y-float64(point.Y))
		if distance > radius {
			continue
		}
		if !found || distance < bestDistance || (distance == bestDistance && spawnPointLess(point, best)) {
			best = point
			bestDistance = distance
			found = true
		}
	}
	return best, bestDistance, found
}

func spawnPointLess(a spawnPoint, b spawnPoint) bool {
	if a.PlayerID != b.PlayerID {
		return a.PlayerID < b.PlayerID
	}
	if a.X != b.X {
		return a.X < b.X
	}
	return a.Y < b.Y
}

func uniqueSpawnUnits(recipes []SpawnRecipe) ([]int, []string) {
	seen := map[int]bool{}
	var ids []int
	var names []string
	for _, recipe := range recipes {
		if seen[recipe.UnitID] {
			continue
		}
		seen[recipe.UnitID] = true
		ids = append(ids, recipe.UnitID)
		if name := UnitDisplayName(recipe.UnitID); name != "" {
			names = append(names, name)
		}
	}
	sort.Ints(ids)
	sort.Strings(names)
	return ids, names
}

func smallLifecycleSelection(first LifecycleFirstSeen, opts LifecycleOptions) bool {
	switch first.Type {
	case "build", "wall", "repair":
		return first.SelectedCount > 0 && first.SelectedCount <= opts.MaxBuildSelection
	default:
		return first.SelectedCount > 0 && first.SelectedCount <= opts.MaxMoveSelection
	}
}

func lifecycleMatchRadius(first LifecycleFirstSeen, opts LifecycleOptions) float64 {
	radius := lifecycleMoveRadius
	switch first.Type {
	case "build", "wall", "repair":
		radius = lifecycleBuildRadius
	}
	if opts.SpawnRadius > 0 && opts.SpawnRadius < radius {
		return opts.SpawnRadius
	}
	return radius
}

func isVillagerUnit(unitID int) bool {
	switch unitID {
	case 83, 293, 118, 212, 259, 214, 56, 57, 120, 354, 579, 581, 122, 216, 123, 218, 156, 222, 592, 590, 124, 220, 2333, 2334:
		return true
	default:
		return false
	}
}

func summarizeLifecyclePlayers(candidates []LifecycleObjectCandidate, overmatched map[int]int) []LifecyclePlayerSummary {
	byPlayer := map[int]*LifecyclePlayerSummary{}
	ensure := func(playerID int) *LifecyclePlayerSummary {
		summary := byPlayer[playerID]
		if summary == nil {
			summary = &LifecyclePlayerSummary{PlayerID: playerID, Label: initialPlayerLabel(playerID)}
			byPlayer[playerID] = summary
		}
		return summary
	}
	for _, candidate := range candidates {
		summary := ensure(candidate.PlayerID)
		summary.SpawnMatchedObjects++
		if candidate.Kind == "villager_spawn_candidate" {
			summary.VillagerCandidates++
			if summary.FirstVillagerTime == "" || candidate.TimeMS < summary.FirstVillagerTimeMS {
				summary.FirstVillagerTimeMS = candidate.TimeMS
				summary.FirstVillagerTime = candidate.Time
			}
		}
	}
	for playerID, count := range overmatched {
		ensure(playerID).OvermatchedNearSpawn = count
	}
	out := make([]LifecyclePlayerSummary, 0, len(byPlayer))
	for _, summary := range byPlayer {
		out = append(out, *summary)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].PlayerID < out[j].PlayerID })
	return out
}

func overmatchedVillagerEvents(overmatched map[int]int) int {
	total := 0
	for _, count := range overmatched {
		total += count
	}
	return total
}
