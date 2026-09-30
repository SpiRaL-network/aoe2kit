package replay

import (
	"fmt"
	"sort"
)

type SpawnCatalogReport struct {
	Path         string               `json:"path,omitempty"`
	Method       string               `json:"method"`
	Verification string               `json:"verification"`
	Summary      SpawnCatalogSummary  `json:"summary"`
	Players      []SpawnPlayerSummary `json:"players,omitempty"`
	Spawns       []SpawnRecipe        `json:"spawns,omitempty"`
	Warnings     []string             `json:"warnings,omitempty"`
}

type SpawnCatalogSummary struct {
	CreateObjectEffects int `json:"create_object_effects"`
	PlausibleSpawns     int `json:"plausible_spawns"`
	Players             int `json:"players"`
	UnitTypes           int `json:"unit_types"`
}

type SpawnPlayerSummary struct {
	PlayerID int         `json:"player_id"`
	Label    string      `json:"label"`
	Spawns   int         `json:"spawns"`
	Units    map[int]int `json:"units,omitempty"`
}

type SpawnRecipe struct {
	TriggerIndex int    `json:"trigger_index"`
	TriggerName  string `json:"trigger_name,omitempty"`
	EffectIndex  int    `json:"effect_index"`
	UnitID       int    `json:"unit_id"`
	UnitName     string `json:"unit_name,omitempty"`
	TargetPlayer int    `json:"target_player"`
	X            int    `json:"x"`
	Y            int    `json:"y"`
	Confidence   string `json:"confidence"`
}

func BuildSpawnCatalog(path string) (*SpawnCatalogReport, error) {
	rec, err := Open(path)
	if err != nil {
		return nil, err
	}
	if rec.TriggerGraph == nil || len(rec.TriggerGraph.Triggers) == 0 {
		return nil, fmt.Errorf("trigger graph with trigger bodies is unavailable")
	}
	var spawns []SpawnRecipe
	createEffects := 0
	unitTypes := map[int]bool{}
	for triggerIndex, trigger := range rec.TriggerGraph.Triggers {
		triggerName, _ := trigger["name"].(string)
		effects, _ := trigger["effects"].([]map[string]any)
		for effectIndex, effect := range effects {
			effectType, _ := effect["type"].(int)
			if effectType != 11 {
				continue
			}
			createEffects++
			fields, ok := intSliceFromAny(effect["fields_prefix"])
			if !ok || len(fields) < 33 {
				continue
			}
			recipe := SpawnRecipe{
				TriggerIndex: triggerIndex,
				TriggerName:  triggerName,
				EffectIndex:  effectIndex,
				UnitID:       fields[8],
				UnitName:     UnitDisplayName(fields[8]),
				TargetPlayer: fields[9],
				X:            fields[16],
				Y:            fields[17],
				Confidence:   "effect_type_11_prefix_observed",
			}
			if !plausibleSpawnRecipe(recipe) {
				continue
			}
			spawns = append(spawns, recipe)
			unitTypes[recipe.UnitID] = true
		}
	}
	players := summarizeSpawnPlayers(spawns)
	return &SpawnCatalogReport{
		Path:         path,
		Method:       "trigger_graph_effect_type_11_prefix",
		Verification: "structure_verified_spawn_recipe_catalog_not_runtime_object_ids",
		Summary: SpawnCatalogSummary{
			CreateObjectEffects: createEffects,
			PlausibleSpawns:     len(spawns),
			Players:             len(players),
			UnitTypes:           len(unitTypes),
		},
		Players: players,
		Spawns:  spawns,
	}, nil
}

func intSliceFromAny(value any) ([]int, bool) {
	switch typed := value.(type) {
	case []int:
		return typed, true
	case []any:
		out := make([]int, 0, len(typed))
		for _, item := range typed {
			switch v := item.(type) {
			case int:
				out = append(out, v)
			case float64:
				out = append(out, int(v))
			default:
				return nil, false
			}
		}
		return out, true
	default:
		return nil, false
	}
}

func plausibleSpawnRecipe(recipe SpawnRecipe) bool {
	return recipe.UnitID >= 0 && recipe.UnitID <= 5000 &&
		recipe.TargetPlayer >= 0 && recipe.TargetPlayer <= 8 &&
		recipe.X >= 0 && recipe.X <= 512 &&
		recipe.Y >= 0 && recipe.Y <= 512
}

func summarizeSpawnPlayers(spawns []SpawnRecipe) []SpawnPlayerSummary {
	byPlayer := map[int]*SpawnPlayerSummary{}
	for _, spawn := range spawns {
		summary := byPlayer[spawn.TargetPlayer]
		if summary == nil {
			summary = &SpawnPlayerSummary{
				PlayerID: spawn.TargetPlayer,
				Label:    initialPlayerLabel(spawn.TargetPlayer),
				Units:    map[int]int{},
			}
			byPlayer[spawn.TargetPlayer] = summary
		}
		summary.Spawns++
		summary.Units[spawn.UnitID]++
	}
	out := make([]SpawnPlayerSummary, 0, len(byPlayer))
	for _, summary := range byPlayer {
		out = append(out, *summary)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].PlayerID < out[j].PlayerID })
	return out
}
