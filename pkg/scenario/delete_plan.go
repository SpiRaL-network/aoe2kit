package scenario

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

type DeletePlanRequest struct {
	Kind          string   `json:"kind"`
	ID            int      `json:"id"`
	TriggerID     *int     `json:"trigger_id,omitempty"`
	ChildIndex    *int     `json:"child_index,omitempty"`
	TargetName    string   `json:"target_name,omitempty"`
	TargetPrefix  string   `json:"target_prefix,omitempty"`
	TargetCaption string   `json:"target_caption,omitempty"`
	TargetText    string   `json:"target_text,omitempty"`
	AreaX1        *float64 `json:"area_x1,omitempty"`
	AreaY1        *float64 `json:"area_y1,omitempty"`
	AreaX2        *float64 `json:"area_x2,omitempty"`
	AreaY2        *float64 `json:"area_y2,omitempty"`
	Player        *int     `json:"player,omitempty"`
	UnitConst     *int     `json:"unit_const,omitempty"`
}

type DeletePlanReport struct {
	Path              string              `json:"path,omitempty"`
	Version           string              `json:"version"`
	Verification      string              `json:"verification"`
	Request           DeletePlanRequest   `json:"request"`
	CanDelete         bool                `json:"can_delete"`
	CanTombstone      bool                `json:"can_tombstone,omitempty"`
	Strategy          string              `json:"strategy"`
	ReferenceSummary  ReferenceSummary    `json:"reference_summary"`
	BlockingRefs      []ScenarioReference `json:"blocking_refs,omitempty"`
	ResolvedIDs       []int               `json:"resolved_ids,omitempty"`
	ResolvedSets      map[string][]int    `json:"resolved_sets,omitempty"`
	SuggestedRecipe   map[string]any      `json:"suggested_recipe,omitempty"`
	CleanupCommand    string              `json:"cleanup_command,omitempty"`
	CleanupRecipe     map[string]any      `json:"cleanup_recipe,omitempty"`
	StructuralCaveats []string            `json:"structural_caveats,omitempty"`
}

func DeletePlanFile(path string, request DeletePlanRequest) (DeletePlanReport, error) {
	file, err := Open(path)
	if err != nil {
		return DeletePlanReport{}, err
	}
	report, err := file.DeletePlan(request)
	if err != nil {
		return DeletePlanReport{}, err
	}
	report.Path = path
	return report, nil
}

func (f *File) DeletePlan(request DeletePlanRequest) (DeletePlanReport, error) {
	kind, err := normalizeDeletePlanKind(request.Kind)
	if err != nil {
		return DeletePlanReport{}, err
	}
	if kind != "system_prefix" && kind != "effect_text_prefix" && kind != "effect_text_contains" && kind != "trigger_contains" && kind != "units_area" && kind != "units_player" && kind != "unit_caption" && kind != "unit_caption_prefix" && kind != "unit_caption_contains" && kind != "unit_type" && kind != "trigger_name" && kind != "trigger_prefix" && kind != "variable_name" && kind != "variable_prefix" && kind != "variable_contains" && kind != "string_text" && kind != "string_prefix" && kind != "string_contains" && request.ID < 0 {
		return DeletePlanReport{}, fmt.Errorf("delete-plan id must be non-negative")
	}
	refs := ReferenceReport{}
	if kind != "effect" && kind != "condition" && kind != "effect_type" && kind != "condition_type" && kind != "effect_text_prefix" && kind != "effect_text_contains" && kind != "trigger_contains" && kind != "system_prefix" && kind != "units_area" && kind != "units_player" && kind != "unit_caption" && kind != "unit_caption_prefix" && kind != "unit_caption_contains" && kind != "unit_type" && kind != "trigger_name" && kind != "trigger_prefix" && kind != "variable_name" && kind != "variable_prefix" && kind != "variable_contains" && kind != "string_text" && kind != "string_prefix" && kind != "string_contains" {
		var err error
		refs, err = f.References(ReferenceOptions{Kind: kind, TargetID: &request.ID})
		if err != nil {
			return DeletePlanReport{}, err
		}
	}
	report := DeletePlanReport{
		Path:              f.Path,
		Version:           f.Version,
		Verification:      "structure_verified_not_engine_verified",
		Request:           normalizedDeletePlanRequest(kind, request),
		ReferenceSummary:  refs.Summary,
		BlockingRefs:      refs.References,
		StructuralCaveats: []string{"this is a structural authoring plan; it is not an in-engine load/play verification"},
	}
	if report.ReferenceSummary.ByKind == nil {
		report.ReferenceSummary.ByKind = map[string]int{}
	}
	switch kind {
	case "system_prefix":
		if strings.TrimSpace(request.TargetPrefix) == "" {
			return DeletePlanReport{}, fmt.Errorf("system-prefix delete-plan needs a non-empty target prefix")
		}
		matches, err := f.matchSystemPrefix(request.TargetPrefix)
		if err != nil {
			return DeletePlanReport{}, err
		}
		report.ResolvedSets = matches.resolvedSets()
		f.addScriptCallDeleteCaveats(&report, matches.TriggerIndexes)
		externalRefs, internalRefs, err := f.systemPrefixReferences(matches)
		if err != nil {
			return DeletePlanReport{}, err
		}
		report.ReferenceSummary = externalRefs.Summary
		report.BlockingRefs = externalRefs.References
		if report.ReferenceSummary.ByKind == nil {
			report.ReferenceSummary.ByKind = map[string]int{}
		}
		if internalRefs.Summary.Total > 0 {
			report.StructuralCaveats = append(report.StructuralCaveats, fmt.Sprintf("%d internal reference(s) inside the matched prefix are allowed because remove_system_prefix deletes triggers, units, variables, and strings as one ordered operation", internalRefs.Summary.Total))
		}
		if externalRefs.Summary.Total == 0 {
			report.CanDelete = true
			report.Strategy = fmt.Sprintf("remove generated system prefix %q: %d trigger(s), %d variable(s), %d string slot(s), %d captioned unit(s)", request.TargetPrefix, len(matches.TriggerIndexes), len(matches.VariableIDs), len(matches.StringIDs), len(matches.UnitReferenceIDs))
			report.SuggestedRecipe = map[string]any{"triggers": []map[string]any{{
				"op":            "remove_system_prefix",
				"target_prefix": request.TargetPrefix,
			}}}
			report.StructuralCaveats = append(report.StructuralCaveats, "system-prefix deletion matches trigger names, variable names, string-table text, and unit inline captions by the same prefix")
			report.StructuralCaveats = append(report.StructuralCaveats, "the writer re-resolves the prefix at mutation time and removes matched triggers first so internal child-row/string/unit references disappear before fixed slots are cleared")
		} else {
			report.Strategy = fmt.Sprintf("blocked: generated system prefix %q has %d external reference(s) from outside the matched bundle", request.TargetPrefix, externalRefs.Summary.Total)
			report.StructuralCaveats = append(report.StructuralCaveats, "external references must be disconnected or retargeted before system-prefix can physically remove the generated bundle")
		}
	case "unit":
		if _, _, _, err := f.findUnitByReferenceID(request.ID); err != nil {
			return DeletePlanReport{}, err
		}
		if refs.Summary.Total == 0 {
			report.CanDelete = true
			report.Strategy = "remove placed unit by reference_id"
			report.SuggestedRecipe = map[string]any{
				"units": []map[string]any{{"op": "remove_unit", "reference_id": request.ID}},
			}
		} else {
			_, cleanupRecipe, err := f.UnitDisconnectRecipe(request.ID)
			if err != nil {
				return DeletePlanReport{}, err
			}
			if len(cleanupRecipe.Triggers) > 0 || len(cleanupRecipe.Units) > 0 {
				cleanupMap, err := recipeToMap(cleanupRecipe)
				if err != nil {
					return DeletePlanReport{}, err
				}
				report.CleanupRecipe = cleanupMap
				report.CleanupCommand = scenarioDisconnectCommand(f.Path, "unit", request.ID)
			}
			report.Strategy = "blocked: remove or retarget scenario references before deleting this placed unit"
			if report.CleanupRecipe != nil {
				report.StructuralCaveats = append(report.StructuralCaveats, "cleanup_recipe removes direct placed-unit references first; re-run delete-plan after cleanup before attempting a physical unit delete")
			}
		}
	case "unit_caption":
		if strings.TrimSpace(request.TargetCaption) == "" {
			return DeletePlanReport{}, fmt.Errorf("unit-caption delete-plan needs a non-empty target caption")
		}
		unit, _, _, err := f.findUnitByCaption(request.TargetCaption, request.Player)
		if err != nil {
			return DeletePlanReport{}, err
		}
		referenceID, ok := unit.intValue("reference_id")
		if !ok || referenceID < 0 {
			return DeletePlanReport{}, fmt.Errorf("unit-caption %q resolved to a unit without a valid reference_id", request.TargetCaption)
		}
		refs, err := f.References(ReferenceOptions{Kind: "unit", TargetID: &referenceID})
		if err != nil {
			return DeletePlanReport{}, err
		}
		report.ReferenceSummary = refs.Summary
		report.BlockingRefs = refs.References
		if report.ReferenceSummary.ByKind == nil {
			report.ReferenceSummary.ByKind = map[string]int{}
		}
		if refs.Summary.Total == 0 {
			report.CanDelete = true
			report.Strategy = "remove placed unit by unique caption selector; writer re-resolves the caption before mutation"
			item := map[string]any{
				"op":             "remove_unit",
				"target_caption": request.TargetCaption,
			}
			if request.Player != nil {
				item["target_player"] = *request.Player
			}
			report.SuggestedRecipe = map[string]any{"units": []map[string]any{item}}
		} else {
			_, cleanupRecipe, err := f.UnitDisconnectRecipe(referenceID)
			if err != nil {
				return DeletePlanReport{}, err
			}
			if len(cleanupRecipe.Triggers) > 0 || len(cleanupRecipe.Units) > 0 {
				cleanupMap, err := recipeToMap(cleanupRecipe)
				if err != nil {
					return DeletePlanReport{}, err
				}
				report.CleanupRecipe = cleanupMap
				report.CleanupCommand = scenarioDisconnectCommand(f.Path, "unit", referenceID)
			}
			report.Strategy = fmt.Sprintf("blocked: unit-caption resolved to reference_id %d, which is still referenced by scenario logic", referenceID)
			report.StructuralCaveats = append(report.StructuralCaveats, "unit-caption selectors are re-resolved at mutation time; if captions change between plan and delete, re-run delete-plan")
			if report.CleanupRecipe != nil {
				report.StructuralCaveats = append(report.StructuralCaveats, "cleanup_recipe removes direct placed-unit references first; re-run delete-plan after cleanup before attempting a caption-selected physical unit delete")
			}
		}
	case "unit_caption_prefix":
		if strings.TrimSpace(request.TargetPrefix) == "" {
			return DeletePlanReport{}, fmt.Errorf("unit-caption-prefix delete-plan needs a non-empty target prefix")
		}
		targets, err := f.findUnitsByCaptionText(request.TargetPrefix, request.Player, true)
		if err != nil {
			return DeletePlanReport{}, err
		}
		resolvedIDs, err := unitReferenceIDsFromMatches(request.TargetPrefix, targets)
		if err != nil {
			return DeletePlanReport{}, err
		}
		report.ResolvedIDs = resolvedIDs
		refs, err := f.referencesForUnitAreaTargets(targets)
		if err != nil {
			return DeletePlanReport{}, err
		}
		report.ReferenceSummary = refs.Summary
		report.BlockingRefs = refs.References
		if report.ReferenceSummary.ByKind == nil {
			report.ReferenceSummary.ByKind = map[string]int{}
		}
		if refs.Summary.Total == 0 {
			report.CanDelete = true
			report.Strategy = fmt.Sprintf("remove %d placed unit(s) by caption prefix %q; resolved reference_ids are used for mutation stability", len(targets), request.TargetPrefix)
			items := make([]map[string]any, 0, len(resolvedIDs))
			for _, id := range resolvedIDs {
				items = append(items, map[string]any{"op": "remove_unit", "reference_id": id})
			}
			report.SuggestedRecipe = map[string]any{"units": items}
		} else {
			_, _, cleanupRecipe, err := f.UnitCaptionPrefixDisconnectRecipe(request.TargetPrefix, request.Player)
			if err != nil {
				return DeletePlanReport{}, err
			}
			if len(cleanupRecipe.Triggers) > 0 || len(cleanupRecipe.Units) > 0 {
				cleanupMap, err := recipeToMap(cleanupRecipe)
				if err != nil {
					return DeletePlanReport{}, err
				}
				report.CleanupRecipe = cleanupMap
				report.CleanupCommand = scenarioUnitCaptionPrefixDisconnectCommand(f.Path, request.TargetPrefix, request.Player)
			}
			report.Strategy = fmt.Sprintf("blocked: caption-prefix selector matched %d placed unit(s), but at least one matched reference_id is used by scenario logic", len(targets))
			report.StructuralCaveats = append(report.StructuralCaveats, "unit-caption-prefix selectors are re-resolved during cleanup planning, but delete recipes use resolved reference_ids to avoid duplicate-caption ambiguity at mutation time")
			report.StructuralCaveats = append(report.StructuralCaveats, "caption-prefix deletion is all-or-nothing; split the prefix or retarget references if only some matched units should be removed")
			if report.CleanupRecipe != nil {
				report.StructuralCaveats = append(report.StructuralCaveats, "cleanup_recipe removes direct references to units matched by the caption prefix; re-run delete-plan after cleanup before attempting the physical delete")
			}
		}
	case "unit_caption_contains":
		if strings.TrimSpace(request.TargetText) == "" {
			return DeletePlanReport{}, fmt.Errorf("unit-caption-contains delete-plan needs a non-empty target text")
		}
		targets, err := f.findUnitsByCaptionText(request.TargetText, request.Player, false)
		if err != nil {
			return DeletePlanReport{}, err
		}
		resolvedIDs, err := unitReferenceIDsFromMatches(request.TargetText, targets)
		if err != nil {
			return DeletePlanReport{}, err
		}
		report.ResolvedIDs = resolvedIDs
		refs, err := f.referencesForUnitAreaTargets(targets)
		if err != nil {
			return DeletePlanReport{}, err
		}
		report.ReferenceSummary = refs.Summary
		report.BlockingRefs = refs.References
		if report.ReferenceSummary.ByKind == nil {
			report.ReferenceSummary.ByKind = map[string]int{}
		}
		if refs.Summary.Total == 0 {
			report.CanDelete = true
			report.Strategy = fmt.Sprintf("remove %d placed unit(s) by caption text marker %q; resolved reference_ids are used for mutation stability", len(targets), request.TargetText)
			items := make([]map[string]any, 0, len(resolvedIDs))
			for _, id := range resolvedIDs {
				items = append(items, map[string]any{"op": "remove_unit", "reference_id": id})
			}
			report.SuggestedRecipe = map[string]any{"units": items}
		} else {
			_, _, cleanupRecipe, err := f.UnitCaptionContainsDisconnectRecipe(request.TargetText, request.Player)
			if err != nil {
				return DeletePlanReport{}, err
			}
			if len(cleanupRecipe.Triggers) > 0 || len(cleanupRecipe.Units) > 0 {
				cleanupMap, err := recipeToMap(cleanupRecipe)
				if err != nil {
					return DeletePlanReport{}, err
				}
				report.CleanupRecipe = cleanupMap
				report.CleanupCommand = scenarioUnitCaptionContainsDisconnectCommand(f.Path, request.TargetText, request.Player)
			}
			report.Strategy = fmt.Sprintf("blocked: caption-contains selector matched %d placed unit(s), but at least one matched reference_id is used by scenario logic", len(targets))
			report.StructuralCaveats = append(report.StructuralCaveats, "unit-caption-contains selectors are re-resolved during cleanup planning, but delete recipes use resolved reference_ids to avoid duplicate-caption ambiguity at mutation time")
			report.StructuralCaveats = append(report.StructuralCaveats, "caption-contains deletion is all-or-nothing; use unit-caption, unit-caption-prefix, unit-type, units-player, or reference_id selectors if only some matched units should be removed")
			if report.CleanupRecipe != nil {
				report.StructuralCaveats = append(report.StructuralCaveats, "cleanup_recipe removes direct references to units matched by the caption text marker; re-run delete-plan after cleanup before attempting the physical delete")
			}
		}
	case "unit_type":
		if request.UnitConst == nil || *request.UnitConst < 0 {
			return DeletePlanReport{}, fmt.Errorf("unit-type delete-plan needs a non-negative unit const")
		}
		targets, err := f.findUnitsByUnitConst(*request.UnitConst, request.Player)
		if err != nil {
			return DeletePlanReport{}, err
		}
		resolvedIDs, err := unitReferenceIDsFromMatches(fmt.Sprintf("unit_const %d", *request.UnitConst), targets)
		if err != nil {
			return DeletePlanReport{}, err
		}
		report.ResolvedIDs = resolvedIDs
		refs, err := f.referencesForUnitAreaTargets(targets)
		if err != nil {
			return DeletePlanReport{}, err
		}
		report.ReferenceSummary = refs.Summary
		report.BlockingRefs = refs.References
		if report.ReferenceSummary.ByKind == nil {
			report.ReferenceSummary.ByKind = map[string]int{}
		}
		if refs.Summary.Total == 0 {
			report.CanDelete = true
			report.Strategy = fmt.Sprintf("remove %d placed unit(s) with unit_const %d; resolved reference_ids are used for mutation stability", len(targets), *request.UnitConst)
			items := make([]map[string]any, 0, len(resolvedIDs))
			for _, id := range resolvedIDs {
				items = append(items, map[string]any{"op": "remove_unit", "reference_id": id})
			}
			report.SuggestedRecipe = map[string]any{"units": items}
		} else {
			_, _, cleanupRecipe, err := f.UnitTypeDisconnectRecipe(*request.UnitConst, request.Player)
			if err != nil {
				return DeletePlanReport{}, err
			}
			if len(cleanupRecipe.Triggers) > 0 || len(cleanupRecipe.Units) > 0 {
				cleanupMap, err := recipeToMap(cleanupRecipe)
				if err != nil {
					return DeletePlanReport{}, err
				}
				report.CleanupRecipe = cleanupMap
				report.CleanupCommand = scenarioUnitTypeDisconnectCommand(f.Path, *request.UnitConst, request.Player)
			}
			report.Strategy = fmt.Sprintf("blocked: unit-type selector matched %d placed unit(s), but at least one matched reference_id is used by scenario logic", len(targets))
			report.StructuralCaveats = append(report.StructuralCaveats, "unit-type deletion is all-or-nothing; add --player, use units-area, or retarget references if only some matched units should be removed")
			if report.CleanupRecipe != nil {
				report.StructuralCaveats = append(report.StructuralCaveats, "cleanup_recipe removes direct references to units matched by the unit type; re-run delete-plan after cleanup before attempting the physical delete")
			}
		}
	case "units_player":
		if request.Player == nil || *request.Player < 0 {
			return DeletePlanReport{}, fmt.Errorf("units-player delete-plan needs a non-negative player")
		}
		targets, err := f.findUnitsByPlayer(request.Player)
		if err != nil {
			return DeletePlanReport{}, err
		}
		resolvedIDs, err := unitReferenceIDsFromMatches(fmt.Sprintf("player %d", *request.Player), targets)
		if err != nil {
			return DeletePlanReport{}, err
		}
		report.ResolvedIDs = resolvedIDs
		refs, err := f.referencesForUnitAreaTargets(targets)
		if err != nil {
			return DeletePlanReport{}, err
		}
		report.ReferenceSummary = refs.Summary
		report.BlockingRefs = refs.References
		if report.ReferenceSummary.ByKind == nil {
			report.ReferenceSummary.ByKind = map[string]int{}
		}
		if refs.Summary.Total == 0 {
			report.CanDelete = true
			report.Strategy = fmt.Sprintf("remove all %d placed unit(s) for player %d", len(targets), *request.Player)
			report.SuggestedRecipe = map[string]any{"units": []map[string]any{{
				"op":            "remove_units_for_player",
				"target_player": *request.Player,
			}}}
		} else {
			_, _, cleanupRecipe, err := f.UnitsPlayerDisconnectRecipe(*request.Player)
			if err != nil {
				return DeletePlanReport{}, err
			}
			if len(cleanupRecipe.Triggers) > 0 || len(cleanupRecipe.Units) > 0 {
				cleanupMap, err := recipeToMap(cleanupRecipe)
				if err != nil {
					return DeletePlanReport{}, err
				}
				report.CleanupRecipe = cleanupMap
				report.CleanupCommand = scenarioUnitsPlayerDisconnectCommand(f.Path, *request.Player)
			}
			report.Strategy = fmt.Sprintf("blocked: player %d has %d placed unit(s), but at least one reference_id is used by scenario logic", *request.Player, len(targets))
			report.StructuralCaveats = append(report.StructuralCaveats, "units-player deletion is all-or-nothing; use unit-type, unit-caption-prefix, or units-area if only some player units should be removed")
			if report.CleanupRecipe != nil {
				report.StructuralCaveats = append(report.StructuralCaveats, "cleanup_recipe removes direct references to units owned by this player; re-run delete-plan after cleanup before attempting the physical delete")
			}
		}
	case "units_area":
		recipe := UnitRecipe{
			Op:              "remove_units_in_area",
			TargetPlayer:    request.Player,
			TargetUnitConst: request.UnitConst,
			TargetAreaX1:    request.AreaX1,
			TargetAreaY1:    request.AreaY1,
			TargetAreaX2:    request.AreaX2,
			TargetAreaY2:    request.AreaY2,
		}
		targets, err := f.findUnitsInArea(recipe)
		if err != nil {
			return DeletePlanReport{}, err
		}
		refs, err := f.referencesForUnitAreaTargets(targets)
		if err != nil {
			return DeletePlanReport{}, err
		}
		report.ReferenceSummary = refs.Summary
		report.BlockingRefs = refs.References
		if report.ReferenceSummary.ByKind == nil {
			report.ReferenceSummary.ByKind = map[string]int{}
		}
		if refs.Summary.Total == 0 {
			report.CanDelete = true
			report.Strategy = fmt.Sprintf("remove %d placed unit(s) by area selector; writer rechecks direct scenario references before mutation", len(targets))
			item := map[string]any{
				"op":             "remove_units_in_area",
				"target_area_x1": *request.AreaX1,
				"target_area_y1": *request.AreaY1,
				"target_area_x2": *request.AreaX2,
				"target_area_y2": *request.AreaY2,
			}
			if request.Player != nil {
				item["target_player"] = *request.Player
			}
			if request.UnitConst != nil {
				item["target_unit_const"] = *request.UnitConst
			}
			report.SuggestedRecipe = map[string]any{"units": []map[string]any{item}}
		} else {
			_, cleanupRecipe, err := f.UnitsAreaDisconnectRecipe(request)
			if err != nil {
				return DeletePlanReport{}, err
			}
			if len(cleanupRecipe.Triggers) > 0 || len(cleanupRecipe.Units) > 0 {
				cleanupMap, err := recipeToMap(cleanupRecipe)
				if err != nil {
					return DeletePlanReport{}, err
				}
				report.CleanupRecipe = cleanupMap
				report.CleanupCommand = scenarioAreaDisconnectCommand(f.Path, request)
			}
			report.Strategy = fmt.Sprintf("blocked: area selector matched %d placed unit(s), but at least one matched reference_id is used by scenario logic", len(targets))
			report.StructuralCaveats = append(report.StructuralCaveats, "area deletion is all-or-nothing; split the rectangle or retarget references if only some matched units should be removed")
			if report.CleanupRecipe != nil {
				report.StructuralCaveats = append(report.StructuralCaveats, "cleanup_recipe removes direct references to units matched by the area selector; re-run delete-plan after cleanup before attempting the area delete")
			}
		}
	case "trigger":
		if err := f.validateTriggerIndex(request.ID); err != nil {
			return DeletePlanReport{}, err
		}
		f.addScriptCallDeleteCaveats(&report, []int{request.ID})
		if refs.Summary.Total == 0 {
			report.CanDelete = true
			report.Strategy = "remove trigger by target_index; writer rewrites trigger-control ids above the deleted index"
			report.SuggestedRecipe = map[string]any{
				"triggers": []map[string]any{{"op": "remove_trigger", "target_index": request.ID}},
			}
		} else {
			_, cleanupRecipe, err := f.TriggerDisconnectRecipe(request.ID)
			if err != nil {
				return DeletePlanReport{}, err
			}
			if len(cleanupRecipe.Triggers) > 0 {
				cleanupMap, err := recipeToMap(cleanupRecipe)
				if err != nil {
					return DeletePlanReport{}, err
				}
				report.CleanupRecipe = cleanupMap
				report.CleanupCommand = scenarioDisconnectCommand(f.Path, "trigger", request.ID)
			}
			tombstoneName := fmt.Sprintf("TOMBSTONED trigger %d", request.ID)
			report.CanTombstone = true
			report.Strategy = "physical remove blocked: trigger-control references point at this index; semantic tombstone preserves the index while deleting behavior"
			report.SuggestedRecipe = map[string]any{
				"triggers": []map[string]any{{
					"op":               "tombstone_trigger",
					"target_index":     request.ID,
					"set_name":         tombstoneName,
					"clear_effects":    true,
					"clear_conditions": true,
				}},
			}
			report.StructuralCaveats = append(report.StructuralCaveats, "tombstoning keeps the trigger row so existing activate/deactivate references remain structurally valid, but they now target an inert disabled trigger")
			if report.CleanupRecipe != nil {
				report.StructuralCaveats = append(report.StructuralCaveats, "cleanup_recipe removes direct trigger-control rows first; re-run delete-plan after cleanup before attempting a physical trigger delete")
			}
		}
	case "trigger_name":
		if strings.TrimSpace(request.TargetName) == "" {
			return DeletePlanReport{}, fmt.Errorf("trigger-name delete-plan needs a non-empty target name")
		}
		triggerIndex, err := f.findUniqueTriggerIndexByName(request.TargetName)
		if err != nil {
			return DeletePlanReport{}, err
		}
		f.addScriptCallDeleteCaveats(&report, []int{triggerIndex})
		refs, err := f.References(ReferenceOptions{Kind: "trigger", TargetID: &triggerIndex})
		if err != nil {
			return DeletePlanReport{}, err
		}
		report.ReferenceSummary = refs.Summary
		report.BlockingRefs = refs.References
		if report.ReferenceSummary.ByKind == nil {
			report.ReferenceSummary.ByKind = map[string]int{}
		}
		if refs.Summary.Total == 0 {
			report.CanDelete = true
			report.Strategy = fmt.Sprintf("remove trigger %d by unique name selector; writer re-resolves the name before mutation", triggerIndex)
			report.SuggestedRecipe = map[string]any{
				"triggers": []map[string]any{{"op": "remove_trigger", "target_name": request.TargetName}},
			}
		} else {
			_, cleanupRecipe, err := f.TriggerDisconnectRecipe(triggerIndex)
			if err != nil {
				return DeletePlanReport{}, err
			}
			if len(cleanupRecipe.Triggers) > 0 {
				cleanupMap, err := recipeToMap(cleanupRecipe)
				if err != nil {
					return DeletePlanReport{}, err
				}
				report.CleanupRecipe = cleanupMap
				report.CleanupCommand = scenarioDisconnectCommand(f.Path, "trigger", triggerIndex)
			}
			tombstoneName := fmt.Sprintf("TOMBSTONED trigger %d", triggerIndex)
			report.CanTombstone = true
			report.Strategy = fmt.Sprintf("physical remove blocked: trigger-name resolved to index %d, and trigger-control references point at this index; semantic tombstone preserves the row while deleting behavior", triggerIndex)
			report.SuggestedRecipe = map[string]any{
				"triggers": []map[string]any{{
					"op":               "tombstone_trigger",
					"target_name":      request.TargetName,
					"set_name":         tombstoneName,
					"clear_effects":    true,
					"clear_conditions": true,
				}},
			}
			report.StructuralCaveats = append(report.StructuralCaveats, "trigger-name selectors must be unique and are re-resolved at mutation time; if names change between plan and delete, re-run delete-plan")
			if report.CleanupRecipe != nil {
				report.StructuralCaveats = append(report.StructuralCaveats, "cleanup_recipe removes direct trigger-control rows first; re-run delete-plan after cleanup before attempting a name-selected physical trigger delete")
			}
		}
	case "trigger_prefix":
		if strings.TrimSpace(request.TargetPrefix) == "" {
			return DeletePlanReport{}, fmt.Errorf("trigger-prefix delete-plan needs a non-empty target prefix")
		}
		indexes, err := f.findTriggerIndexesByNamePrefix(request.TargetPrefix)
		if err != nil {
			return DeletePlanReport{}, err
		}
		report.ResolvedIDs = indexes
		f.addScriptCallDeleteCaveats(&report, indexes)
		refs, err := f.referencesForTriggerIndexes(indexes)
		if err != nil {
			return DeletePlanReport{}, err
		}
		externalRefs, internalRefs := splitTriggerBatchReferences(indexes, refs)
		report.ReferenceSummary = externalRefs.Summary
		report.BlockingRefs = externalRefs.References
		if report.ReferenceSummary.ByKind == nil {
			report.ReferenceSummary.ByKind = map[string]int{}
		}
		if internalRefs.Summary.Total > 0 {
			report.StructuralCaveats = append(report.StructuralCaveats, fmt.Sprintf("%d internal trigger-control reference(s) among matched triggers are allowed because remove_triggers deletes the batch atomically", internalRefs.Summary.Total))
		}
		if externalRefs.Summary.Total == 0 {
			report.CanDelete = true
			report.Strategy = fmt.Sprintf("remove %d trigger(s) by trigger-name prefix %q as one batch; writer rewrites surviving trigger-control ids after atomic deletion", len(indexes), request.TargetPrefix)
			report.SuggestedRecipe = map[string]any{"triggers": []map[string]any{{
				"op":             "remove_triggers",
				"target_indexes": indexes,
			}}}
		} else {
			cleanupRecipe := triggerRowDisconnectRecipe(externalRefs.References)
			if len(cleanupRecipe.Triggers) > 0 {
				cleanupMap, err := recipeToMap(cleanupRecipe)
				if err != nil {
					return DeletePlanReport{}, err
				}
				report.CleanupRecipe = cleanupMap
				report.CleanupCommand = scenarioTriggerPrefixDisconnectCommand(f.Path, request.TargetPrefix)
			}
			report.CanTombstone = true
			report.Strategy = fmt.Sprintf("physical remove blocked: %d trigger(s) matched prefix %q, but external trigger-control references point at one or more matched indexes; semantic tombstones preserve indexes", len(indexes), request.TargetPrefix)
			items := make([]map[string]any, 0, len(indexes))
			for _, index := range indexes {
				items = append(items, map[string]any{
					"op":               "tombstone_trigger",
					"target_index":     index,
					"set_name":         fmt.Sprintf("TOMBSTONED trigger %d", index),
					"clear_effects":    true,
					"clear_conditions": true,
				})
			}
			report.SuggestedRecipe = map[string]any{"triggers": items}
			report.StructuralCaveats = append(report.StructuralCaveats, "trigger-prefix selectors are re-resolved at mutation time; if trigger names change between plan and delete, re-run delete-plan")
			report.StructuralCaveats = append(report.StructuralCaveats, "trigger-control references from outside the matched prefix remain blockers; internal references among matched triggers can be physically removed only by the atomic remove_triggers recipe")
			if report.CleanupRecipe != nil {
				report.StructuralCaveats = append(report.StructuralCaveats, "cleanup_recipe removes direct trigger-control rows first; re-run delete-plan after cleanup before attempting physical trigger-prefix deletion")
			}
		}
	case "trigger_contains":
		if strings.TrimSpace(request.TargetText) == "" {
			return DeletePlanReport{}, fmt.Errorf("trigger-contains delete-plan needs a non-empty target text")
		}
		indexes, err := f.findTriggerIndexesContainingText(request.TargetText)
		if err != nil {
			return DeletePlanReport{}, err
		}
		report.ResolvedIDs = indexes
		f.addScriptCallDeleteCaveats(&report, indexes)
		refs, err := f.referencesForTriggerIndexes(indexes)
		if err != nil {
			return DeletePlanReport{}, err
		}
		externalRefs, internalRefs := splitTriggerBatchReferences(indexes, refs)
		report.ReferenceSummary = externalRefs.Summary
		report.BlockingRefs = externalRefs.References
		if report.ReferenceSummary.ByKind == nil {
			report.ReferenceSummary.ByKind = map[string]int{}
		}
		if internalRefs.Summary.Total > 0 {
			report.StructuralCaveats = append(report.StructuralCaveats, fmt.Sprintf("%d internal trigger-control reference(s) among matched triggers are allowed because remove_triggers deletes the batch atomically", internalRefs.Summary.Total))
		}
		if externalRefs.Summary.Total == 0 {
			report.CanDelete = true
			report.Strategy = fmt.Sprintf("remove %d trigger(s) whose name or raw effect message contains %q as one batch; writer rewrites surviving trigger-control ids after atomic deletion", len(indexes), request.TargetText)
			report.SuggestedRecipe = map[string]any{"triggers": []map[string]any{{
				"op":             "remove_triggers",
				"target_indexes": indexes,
			}}}
		} else {
			cleanupRecipe := triggerRowDisconnectRecipe(externalRefs.References)
			if len(cleanupRecipe.Triggers) > 0 {
				cleanupMap, err := recipeToMap(cleanupRecipe)
				if err != nil {
					return DeletePlanReport{}, err
				}
				report.CleanupRecipe = cleanupMap
				report.CleanupCommand = scenarioTriggerContainsDisconnectCommand(f.Path, request.TargetText)
			}
			report.CanTombstone = true
			report.Strategy = fmt.Sprintf("physical remove blocked: %d trigger(s) contained %q, but external trigger-control references point at one or more matched indexes; semantic tombstones preserve indexes", len(indexes), request.TargetText)
			items := make([]map[string]any, 0, len(indexes))
			for _, index := range indexes {
				items = append(items, map[string]any{
					"op":               "tombstone_trigger",
					"target_index":     index,
					"set_name":         fmt.Sprintf("TOMBSTONED trigger %d", index),
					"clear_effects":    true,
					"clear_conditions": true,
				})
			}
			report.SuggestedRecipe = map[string]any{"triggers": items}
			report.StructuralCaveats = append(report.StructuralCaveats, "trigger-contains matches trigger names and raw effect message fields only; inspect resolved_ids before applying a mutating delete")
			report.StructuralCaveats = append(report.StructuralCaveats, "trigger-control references from outside the matched set remain blockers; internal references among matched triggers can be physically removed only by the atomic remove_triggers recipe")
		}
	case "variable":
		if _, err := f.findVariableByID(request.ID); err != nil {
			return DeletePlanReport{}, err
		}
		if refs.Summary.Total == 0 {
			report.CanDelete = true
			report.Strategy = "remove unreferenced variable record; surviving variable ids are explicit and are not renumbered"
			report.SuggestedRecipe = map[string]any{
				"variables": []map[string]any{{"op": "remove_variable", "target_id": request.ID}},
			}
		} else {
			_, cleanupRecipe, err := f.VariableDisconnectRecipe(request.ID)
			if err != nil {
				return DeletePlanReport{}, err
			}
			if len(cleanupRecipe.Triggers) > 0 {
				cleanupMap, err := recipeToMap(cleanupRecipe)
				if err != nil {
					return DeletePlanReport{}, err
				}
				report.CleanupRecipe = cleanupMap
				report.CleanupCommand = scenarioDisconnectCommand(f.Path, "variable", request.ID)
			}
			report.Strategy = "blocked: remove or retarget trigger variable references before deleting this variable"
			if report.CleanupRecipe != nil {
				report.StructuralCaveats = append(report.StructuralCaveats, "cleanup_recipe removes direct trigger rows that reference the variable; re-run delete-plan after cleanup before attempting a physical variable delete")
			}
		}
	case "variable_name":
		if strings.TrimSpace(request.TargetName) == "" {
			return DeletePlanReport{}, fmt.Errorf("variable-name delete-plan needs a non-empty target name")
		}
		variableID, err := f.findUniqueVariableIDByName(request.TargetName)
		if err != nil {
			return DeletePlanReport{}, err
		}
		refs, err := f.References(ReferenceOptions{Kind: "variable", TargetID: &variableID})
		if err != nil {
			return DeletePlanReport{}, err
		}
		report.ReferenceSummary = refs.Summary
		report.BlockingRefs = refs.References
		if report.ReferenceSummary.ByKind == nil {
			report.ReferenceSummary.ByKind = map[string]int{}
		}
		if refs.Summary.Total == 0 {
			report.CanDelete = true
			report.Strategy = fmt.Sprintf("remove unreferenced variable id %d by unique name selector; surviving variable ids are explicit and are not renumbered", variableID)
			report.SuggestedRecipe = map[string]any{
				"variables": []map[string]any{{"op": "remove_variable", "target_name": request.TargetName}},
			}
		} else {
			_, cleanupRecipe, err := f.VariableDisconnectRecipe(variableID)
			if err != nil {
				return DeletePlanReport{}, err
			}
			if len(cleanupRecipe.Triggers) > 0 {
				cleanupMap, err := recipeToMap(cleanupRecipe)
				if err != nil {
					return DeletePlanReport{}, err
				}
				report.CleanupRecipe = cleanupMap
				report.CleanupCommand = scenarioDisconnectCommand(f.Path, "variable", variableID)
			}
			report.Strategy = fmt.Sprintf("blocked: variable-name resolved to id %d, which is still referenced by trigger logic", variableID)
			report.StructuralCaveats = append(report.StructuralCaveats, "variable-name selectors must be unique and are re-resolved at mutation time; if variable names change between plan and delete, re-run delete-plan")
			if report.CleanupRecipe != nil {
				report.StructuralCaveats = append(report.StructuralCaveats, "cleanup_recipe removes direct trigger rows that reference the variable; re-run delete-plan after cleanup before attempting a name-selected physical variable delete")
			}
		}
	case "variable_prefix":
		if strings.TrimSpace(request.TargetPrefix) == "" {
			return DeletePlanReport{}, fmt.Errorf("variable-prefix delete-plan needs a non-empty target prefix")
		}
		ids, err := f.findVariableIDsByNamePrefix(request.TargetPrefix)
		if err != nil {
			return DeletePlanReport{}, err
		}
		report.ResolvedIDs = ids
		refs, err := f.referencesForVariableIDs(ids)
		if err != nil {
			return DeletePlanReport{}, err
		}
		report.ReferenceSummary = refs.Summary
		report.BlockingRefs = refs.References
		if report.ReferenceSummary.ByKind == nil {
			report.ReferenceSummary.ByKind = map[string]int{}
		}
		if refs.Summary.Total == 0 {
			report.CanDelete = true
			report.Strategy = fmt.Sprintf("remove %d unreferenced variable(s) by name prefix %q; variables are addressed by stable ids and surviving ids are not renumbered", len(ids), request.TargetPrefix)
			items := make([]map[string]any, 0, len(ids))
			for _, id := range ids {
				items = append(items, map[string]any{"op": "remove_variable", "target_id": id})
			}
			report.SuggestedRecipe = map[string]any{"variables": items}
		} else {
			_, _, cleanupRecipe, err := f.VariablePrefixDisconnectRecipe(request.TargetPrefix)
			if err != nil {
				return DeletePlanReport{}, err
			}
			if len(cleanupRecipe.Triggers) > 0 {
				cleanupMap, err := recipeToMap(cleanupRecipe)
				if err != nil {
					return DeletePlanReport{}, err
				}
				report.CleanupRecipe = cleanupMap
				report.CleanupCommand = scenarioVariablePrefixDisconnectCommand(f.Path, request.TargetPrefix)
			}
			report.Strategy = fmt.Sprintf("blocked: %d variable(s) matched prefix %q, but at least one matched id is still referenced by trigger logic", len(ids), request.TargetPrefix)
			report.StructuralCaveats = append(report.StructuralCaveats, "variable-prefix deletion is all-or-nothing for physical removal; disconnect direct references first or split the prefix")
			if report.CleanupRecipe != nil {
				report.StructuralCaveats = append(report.StructuralCaveats, "cleanup_recipe removes direct trigger rows that reference matched variables; re-run delete-plan after cleanup before attempting physical variable-prefix deletion")
			}
		}
	case "variable_contains":
		if strings.TrimSpace(request.TargetText) == "" {
			return DeletePlanReport{}, fmt.Errorf("variable-contains delete-plan needs a non-empty target text")
		}
		ids, err := f.findVariableIDsByNameContains(request.TargetText)
		if err != nil {
			return DeletePlanReport{}, err
		}
		report.ResolvedIDs = ids
		refs, err := f.referencesForVariableIDs(ids)
		if err != nil {
			return DeletePlanReport{}, err
		}
		report.ReferenceSummary = refs.Summary
		report.BlockingRefs = refs.References
		if report.ReferenceSummary.ByKind == nil {
			report.ReferenceSummary.ByKind = map[string]int{}
		}
		if refs.Summary.Total == 0 {
			report.CanDelete = true
			report.Strategy = fmt.Sprintf("remove %d unreferenced variable(s) whose name contains %q; variables are addressed by stable ids and surviving ids are not renumbered", len(ids), request.TargetText)
			items := make([]map[string]any, 0, len(ids))
			for _, id := range ids {
				items = append(items, map[string]any{"op": "remove_variable", "target_id": id})
			}
			report.SuggestedRecipe = map[string]any{"variables": items}
		} else {
			_, _, cleanupRecipe, err := f.VariableContainsDisconnectRecipe(request.TargetText)
			if err != nil {
				return DeletePlanReport{}, err
			}
			if len(cleanupRecipe.Triggers) > 0 {
				cleanupMap, err := recipeToMap(cleanupRecipe)
				if err != nil {
					return DeletePlanReport{}, err
				}
				report.CleanupRecipe = cleanupMap
				report.CleanupCommand = scenarioVariableContainsDisconnectCommand(f.Path, request.TargetText)
			}
			report.Strategy = fmt.Sprintf("blocked: %d variable(s) contained %q, but at least one matched id is still referenced by trigger logic", len(ids), request.TargetText)
			report.StructuralCaveats = append(report.StructuralCaveats, "variable-contains deletion is all-or-nothing for physical removal; disconnect direct references first or split the selector")
			if report.CleanupRecipe != nil {
				report.StructuralCaveats = append(report.StructuralCaveats, "cleanup_recipe removes direct trigger rows that reference matched variables; re-run delete-plan after cleanup before attempting physical variable-contains deletion")
			}
		}
	case "string":
		if _, err := f.planStringSlot(StringRecipe{Op: "clear_string", ID: &request.ID}, false); err != nil {
			return DeletePlanReport{}, err
		}
		if refs.Summary.Total == 0 {
			report.CanDelete = true
			report.Strategy = "clear fixed string-table slot; physical string compaction is intentionally unsupported because it would renumber later string ids"
			report.SuggestedRecipe = map[string]any{
				"strings": []map[string]any{{"op": "clear_string", "id": request.ID}},
			}
		} else {
			_, cleanupRecipe, err := f.StringDisconnectRecipe(request.ID)
			if err != nil {
				return DeletePlanReport{}, err
			}
			if len(cleanupRecipe.Triggers) > 0 || len(cleanupRecipe.Units) > 0 {
				cleanupMap, err := recipeToMap(cleanupRecipe)
				if err != nil {
					return DeletePlanReport{}, err
				}
				report.CleanupRecipe = cleanupMap
				report.CleanupCommand = scenarioDisconnectCommand(f.Path, "string", request.ID)
			}
			report.CanTombstone = true
			report.Strategy = "physical clear blocked: string_id references point at this fixed slot; semantic tombstone preserves the id while replacing the text"
			report.SuggestedRecipe = map[string]any{
				"strings": []map[string]any{{
					"op":       "set_string",
					"id":       request.ID,
					"set_text": fmt.Sprintf("TOMBSTONED string %d", request.ID),
				}},
			}
			report.StructuralCaveats = append(report.StructuralCaveats, "tombstoning keeps the string-table slot so existing references remain structurally valid; visible UI text will show the tombstone marker until a designer replaces it with final wording or an intentional blank")
			if report.CleanupRecipe != nil {
				report.StructuralCaveats = append(report.StructuralCaveats, "cleanup_recipe removes or clears direct string references first; re-run delete-plan after cleanup before attempting a physical string clear")
			}
		}
	case "string_text":
		if strings.TrimSpace(request.TargetText) == "" {
			return DeletePlanReport{}, fmt.Errorf("string-text delete-plan needs a non-empty exact text")
		}
		stringID, err := f.findUniqueStringIDByText(request.TargetText)
		if err != nil {
			return DeletePlanReport{}, err
		}
		refs, err := f.References(ReferenceOptions{Kind: "string", TargetID: &stringID})
		if err != nil {
			return DeletePlanReport{}, err
		}
		report.ReferenceSummary = refs.Summary
		report.BlockingRefs = refs.References
		if report.ReferenceSummary.ByKind == nil {
			report.ReferenceSummary.ByKind = map[string]int{}
		}
		if refs.Summary.Total == 0 {
			report.CanDelete = true
			report.Strategy = fmt.Sprintf("clear fixed string-table slot %d by unique exact-text selector; physical string compaction remains unsupported", stringID)
			report.SuggestedRecipe = map[string]any{
				"strings": []map[string]any{{"op": "clear_string", "old_text": request.TargetText}},
			}
		} else {
			_, cleanupRecipe, err := f.StringDisconnectRecipe(stringID)
			if err != nil {
				return DeletePlanReport{}, err
			}
			if len(cleanupRecipe.Triggers) > 0 || len(cleanupRecipe.Units) > 0 {
				cleanupMap, err := recipeToMap(cleanupRecipe)
				if err != nil {
					return DeletePlanReport{}, err
				}
				report.CleanupRecipe = cleanupMap
				report.CleanupCommand = scenarioDisconnectCommand(f.Path, "string", stringID)
			}
			report.CanTombstone = true
			report.Strategy = fmt.Sprintf("physical clear blocked: string-text resolved to id %d, and string_id references point at this fixed slot; semantic tombstone preserves the id while replacing the text", stringID)
			report.SuggestedRecipe = map[string]any{
				"strings": []map[string]any{{
					"op":       "set_string",
					"id":       stringID,
					"set_text": fmt.Sprintf("TOMBSTONED string %d", stringID),
				}},
			}
			report.StructuralCaveats = append(report.StructuralCaveats, "string-text selectors must be unique and are re-resolved at mutation time; if string text changes between plan and delete, re-run delete-plan")
			if report.CleanupRecipe != nil {
				report.StructuralCaveats = append(report.StructuralCaveats, "cleanup_recipe removes or clears direct string references first; re-run delete-plan after cleanup before attempting a text-selected physical string clear")
			}
		}
	case "string_prefix":
		if strings.TrimSpace(request.TargetPrefix) == "" {
			return DeletePlanReport{}, fmt.Errorf("string-prefix delete-plan needs a non-empty target prefix")
		}
		ids, err := f.findStringIDsByTextPrefix(request.TargetPrefix)
		if err != nil {
			return DeletePlanReport{}, err
		}
		report.ResolvedIDs = ids
		refs, err := f.referencesForStringIDs(ids)
		if err != nil {
			return DeletePlanReport{}, err
		}
		report.ReferenceSummary = refs.Summary
		report.BlockingRefs = refs.References
		if report.ReferenceSummary.ByKind == nil {
			report.ReferenceSummary.ByKind = map[string]int{}
		}
		if refs.Summary.Total == 0 {
			report.CanDelete = true
			report.Strategy = fmt.Sprintf("clear %d fixed string-table slot(s) by text prefix %q; physical string compaction remains unsupported", len(ids), request.TargetPrefix)
			items := make([]map[string]any, 0, len(ids))
			for _, id := range ids {
				items = append(items, map[string]any{"op": "clear_string", "id": id})
			}
			report.SuggestedRecipe = map[string]any{"strings": items}
		} else {
			_, _, cleanupRecipe, err := f.StringPrefixDisconnectRecipe(request.TargetPrefix)
			if err != nil {
				return DeletePlanReport{}, err
			}
			if len(cleanupRecipe.Triggers) > 0 || len(cleanupRecipe.Units) > 0 {
				cleanupMap, err := recipeToMap(cleanupRecipe)
				if err != nil {
					return DeletePlanReport{}, err
				}
				report.CleanupRecipe = cleanupMap
				report.CleanupCommand = scenarioStringPrefixDisconnectCommand(f.Path, request.TargetPrefix)
			}
			report.CanTombstone = true
			report.Strategy = fmt.Sprintf("physical clear blocked: %d string-table slot(s) matched prefix %q, but direct references point at one or more matched ids; semantic tombstones preserve fixed ids", len(ids), request.TargetPrefix)
			items := make([]map[string]any, 0, len(ids))
			for _, id := range ids {
				items = append(items, map[string]any{
					"op":       "set_string",
					"id":       id,
					"set_text": fmt.Sprintf("TOMBSTONED string %d", id),
				})
			}
			report.SuggestedRecipe = map[string]any{"strings": items}
			report.StructuralCaveats = append(report.StructuralCaveats, "string-prefix selectors are re-resolved at mutation time; if string text changes between plan and delete, re-run delete-plan")
			report.StructuralCaveats = append(report.StructuralCaveats, "string-prefix physical clear is all-or-nothing; referenced matched strings are tombstoned unless direct references are disconnected first")
			if report.CleanupRecipe != nil {
				report.StructuralCaveats = append(report.StructuralCaveats, "cleanup_recipe removes or clears direct string references first; re-run delete-plan after cleanup before attempting physical string-prefix clear")
			}
		}
	case "string_contains":
		if strings.TrimSpace(request.TargetText) == "" {
			return DeletePlanReport{}, fmt.Errorf("string-contains delete-plan needs a non-empty target text")
		}
		ids, err := f.findStringIDsByTextContains(request.TargetText)
		if err != nil {
			return DeletePlanReport{}, err
		}
		report.ResolvedIDs = ids
		refs, err := f.referencesForStringIDs(ids)
		if err != nil {
			return DeletePlanReport{}, err
		}
		report.ReferenceSummary = refs.Summary
		report.BlockingRefs = refs.References
		if report.ReferenceSummary.ByKind == nil {
			report.ReferenceSummary.ByKind = map[string]int{}
		}
		if refs.Summary.Total == 0 {
			report.CanDelete = true
			report.Strategy = fmt.Sprintf("clear %d fixed string-table slot(s) whose text contains %q; physical string compaction remains unsupported", len(ids), request.TargetText)
			items := make([]map[string]any, 0, len(ids))
			for _, id := range ids {
				items = append(items, map[string]any{"op": "clear_string", "id": id})
			}
			report.SuggestedRecipe = map[string]any{"strings": items}
		} else {
			_, _, cleanupRecipe, err := f.StringContainsDisconnectRecipe(request.TargetText)
			if err != nil {
				return DeletePlanReport{}, err
			}
			if len(cleanupRecipe.Triggers) > 0 || len(cleanupRecipe.Units) > 0 {
				cleanupMap, err := recipeToMap(cleanupRecipe)
				if err != nil {
					return DeletePlanReport{}, err
				}
				report.CleanupRecipe = cleanupMap
				report.CleanupCommand = scenarioStringContainsDisconnectCommand(f.Path, request.TargetText)
			}
			report.CanTombstone = true
			report.Strategy = fmt.Sprintf("physical clear blocked: %d string-table slot(s) contained %q, but direct references point at one or more matched ids; semantic tombstones preserve fixed ids", len(ids), request.TargetText)
			items := make([]map[string]any, 0, len(ids))
			for _, id := range ids {
				items = append(items, map[string]any{
					"op":       "set_string",
					"id":       id,
					"set_text": fmt.Sprintf("TOMBSTONED string %d", id),
				})
			}
			report.SuggestedRecipe = map[string]any{"strings": items}
			report.StructuralCaveats = append(report.StructuralCaveats, "string-contains selectors are re-resolved during cleanup planning, but delete recipes use resolved fixed string ids")
			report.StructuralCaveats = append(report.StructuralCaveats, "string-contains physical clear is all-or-nothing; referenced matched strings are tombstoned unless direct references are disconnected first")
			if report.CleanupRecipe != nil {
				report.StructuralCaveats = append(report.StructuralCaveats, "cleanup_recipe removes or clears direct string references first; re-run delete-plan after cleanup before attempting physical string-contains clear")
			}
		}
	case "effect", "condition":
		triggerIndex, childIndex := deletePlanChildTarget(request)
		trigger, err := f.triggerByIndex(triggerIndex)
		if err != nil {
			return DeletePlanReport{}, err
		}
		childKind := kind
		dataField := "effect_data"
		recipeKey := "remove_effects"
		if kind == "condition" {
			dataField = "condition_data"
			recipeKey = "remove_conditions"
		}
		children := trigger.list(dataField)
		if childIndex < 0 || childIndex >= len(children) {
			return DeletePlanReport{}, fmt.Errorf("%s index %d out of range 0..%d for trigger %d", childKind, childIndex, len(children)-1, triggerIndex)
		}
		report.CanDelete = true
		report.Strategy = fmt.Sprintf("remove trigger %d %s row %d; later sibling rows shift down by one", triggerIndex, childKind, childIndex)
		report.SuggestedRecipe = map[string]any{
			"triggers": []map[string]any{{
				"op":           "edit_trigger",
				"target_index": triggerIndex,
				recipeKey:      []int{childIndex},
			}},
		}
		report.StructuralCaveats = append(report.StructuralCaveats, "effect and condition rows are addressed by trigger-local index, not stable ids; inspect the trigger again before applying an old plan")
	case "effect_type", "condition_type":
		if request.ID < 0 {
			return DeletePlanReport{}, fmt.Errorf("%s delete-plan needs a non-negative type id", kind)
		}
		rows, err := f.findTriggerChildRowsByType(kind, request.ID, request.TriggerID, request.TargetPrefix)
		if err != nil {
			return DeletePlanReport{}, err
		}
		report.ResolvedIDs = flattenTriggerChildRows(rows)
		childKind := strings.TrimSuffix(kind, "_type")
		recipeKey := "remove_effects"
		typeName := EffectTypeName(request.ID)
		if kind == "condition_type" {
			recipeKey = "remove_conditions"
			typeName = ConditionTypeName(request.ID)
		}
		items := make([]map[string]any, 0, len(rows))
		totalRows := 0
		for _, row := range rows {
			totalRows += len(row.ChildIndexes)
			items = append(items, map[string]any{
				"op":           "edit_trigger",
				"target_index": row.TriggerIndex,
				recipeKey:      row.ChildIndexes,
			})
		}
		report.CanDelete = true
		report.Strategy = fmt.Sprintf("remove %d %s row(s) with type %d (%s) across %d trigger(s)", totalRows, childKind, request.ID, typeName, len(rows))
		report.SuggestedRecipe = map[string]any{"triggers": items}
		report.StructuralCaveats = append(report.StructuralCaveats, fmt.Sprintf("%s-type deletion removes every currently matching child row in scope; it does not prove the gameplay semantics of removing those rows", childKind))
		report.StructuralCaveats = append(report.StructuralCaveats, "effect and condition rows are trigger-local ordered rows, not stable ids; re-run delete-plan if any trigger children changed")
	case "effect_text_prefix":
		if strings.TrimSpace(request.TargetText) == "" {
			return DeletePlanReport{}, fmt.Errorf("effect-text-prefix delete-plan needs a non-empty target text prefix")
		}
		rows, err := f.findEffectRowsByText(request.TargetText, request.TriggerID, request.TargetPrefix, true)
		if err != nil {
			return DeletePlanReport{}, err
		}
		report.ResolvedIDs = flattenTriggerChildRows(rows)
		items := make([]map[string]any, 0, len(rows))
		totalRows := 0
		for _, row := range rows {
			totalRows += len(row.ChildIndexes)
			items = append(items, map[string]any{
				"op":             "edit_trigger",
				"target_index":   row.TriggerIndex,
				"remove_effects": row.ChildIndexes,
			})
		}
		report.CanDelete = true
		report.Strategy = fmt.Sprintf("remove %d effect row(s) whose message starts with %q across %d trigger(s)", totalRows, request.TargetText, len(rows))
		report.SuggestedRecipe = map[string]any{"triggers": items}
		report.StructuralCaveats = append(report.StructuralCaveats, "effect-text-prefix deletion matches the raw effect message field only; it does not inspect string-table ids or resolved display text")
		report.StructuralCaveats = append(report.StructuralCaveats, "effect rows are trigger-local ordered rows, not stable ids; re-run delete-plan if any trigger children changed")
	case "effect_text_contains":
		if strings.TrimSpace(request.TargetText) == "" {
			return DeletePlanReport{}, fmt.Errorf("effect-text-contains delete-plan needs a non-empty target text")
		}
		rows, err := f.findEffectRowsByText(request.TargetText, request.TriggerID, request.TargetPrefix, false)
		if err != nil {
			return DeletePlanReport{}, err
		}
		report.ResolvedIDs = flattenTriggerChildRows(rows)
		items := make([]map[string]any, 0, len(rows))
		totalRows := 0
		for _, row := range rows {
			totalRows += len(row.ChildIndexes)
			items = append(items, map[string]any{
				"op":             "edit_trigger",
				"target_index":   row.TriggerIndex,
				"remove_effects": row.ChildIndexes,
			})
		}
		report.CanDelete = true
		report.Strategy = fmt.Sprintf("remove %d effect row(s) whose message contains %q across %d trigger(s)", totalRows, request.TargetText, len(rows))
		report.SuggestedRecipe = map[string]any{"triggers": items}
		report.StructuralCaveats = append(report.StructuralCaveats, "effect-text-contains deletion matches the raw effect message field only; it does not inspect string-table ids or resolved display text")
		report.StructuralCaveats = append(report.StructuralCaveats, "effect rows are trigger-local ordered rows, not stable ids; re-run delete-plan if any trigger children changed")
	}
	return report, nil
}

func scenarioDisconnectCommand(path, kind string, id int) string {
	inPath := path
	if strings.TrimSpace(inPath) == "" {
		inPath = "<in.aoe2scenario>"
	}
	return fmt.Sprintf("kit scen disconnect %s <out.aoe2scenario> %s %d", inPath, kind, id)
}

func scenarioAreaDisconnectCommand(path string, request DeletePlanRequest) string {
	inPath := path
	if strings.TrimSpace(inPath) == "" {
		inPath = "<in.aoe2scenario>"
	}
	command := fmt.Sprintf("kit scen disconnect %s <out.aoe2scenario> units-area %.2f,%.2f,%.2f,%.2f",
		inPath,
		deletePlanFloatPtrValue(request.AreaX1),
		deletePlanFloatPtrValue(request.AreaY1),
		deletePlanFloatPtrValue(request.AreaX2),
		deletePlanFloatPtrValue(request.AreaY2))
	if request.Player != nil {
		command += fmt.Sprintf(" --player %d", *request.Player)
	}
	if request.UnitConst != nil {
		command += fmt.Sprintf(" --unit %d", *request.UnitConst)
	}
	return command
}

func scenarioTriggerPrefixDisconnectCommand(path, prefix string) string {
	inPath := path
	if strings.TrimSpace(inPath) == "" {
		inPath = "<in.aoe2scenario>"
	}
	return fmt.Sprintf("kit scen disconnect %s <out.aoe2scenario> trigger-prefix %q", inPath, prefix)
}

func scenarioTriggerContainsDisconnectCommand(path, text string) string {
	inPath := path
	if strings.TrimSpace(inPath) == "" {
		inPath = "<in.aoe2scenario>"
	}
	return fmt.Sprintf("kit scen disconnect %s <out.aoe2scenario> trigger-contains %q", inPath, text)
}

func scenarioVariablePrefixDisconnectCommand(path, prefix string) string {
	inPath := path
	if strings.TrimSpace(inPath) == "" {
		inPath = "<in.aoe2scenario>"
	}
	return fmt.Sprintf("kit scen disconnect %s <out.aoe2scenario> variable-prefix %q", inPath, prefix)
}

func scenarioVariableContainsDisconnectCommand(path, text string) string {
	inPath := path
	if strings.TrimSpace(inPath) == "" {
		inPath = "<in.aoe2scenario>"
	}
	return fmt.Sprintf("kit scen disconnect %s <out.aoe2scenario> variable-contains %q", inPath, text)
}

func scenarioUnitCaptionPrefixDisconnectCommand(path, prefix string, player *int) string {
	inPath := path
	if strings.TrimSpace(inPath) == "" {
		inPath = "<in.aoe2scenario>"
	}
	if player != nil {
		return fmt.Sprintf("kit scen disconnect %s <out.aoe2scenario> unit-caption-prefix %q --player %d", inPath, prefix, *player)
	}
	return fmt.Sprintf("kit scen disconnect %s <out.aoe2scenario> unit-caption-prefix %q", inPath, prefix)
}

func scenarioUnitCaptionContainsDisconnectCommand(path, text string, player *int) string {
	inPath := path
	if strings.TrimSpace(inPath) == "" {
		inPath = "<in.aoe2scenario>"
	}
	if player != nil {
		return fmt.Sprintf("kit scen disconnect %s <out.aoe2scenario> unit-caption-contains %q --player %d", inPath, text, *player)
	}
	return fmt.Sprintf("kit scen disconnect %s <out.aoe2scenario> unit-caption-contains %q", inPath, text)
}

func scenarioUnitTypeDisconnectCommand(path string, unitConst int, player *int) string {
	inPath := path
	if strings.TrimSpace(inPath) == "" {
		inPath = "<in.aoe2scenario>"
	}
	if player != nil {
		return fmt.Sprintf("kit scen disconnect %s <out.aoe2scenario> unit-type %d --player %d", inPath, unitConst, *player)
	}
	return fmt.Sprintf("kit scen disconnect %s <out.aoe2scenario> unit-type %d", inPath, unitConst)
}

func scenarioUnitsPlayerDisconnectCommand(path string, player int) string {
	inPath := path
	if strings.TrimSpace(inPath) == "" {
		inPath = "<in.aoe2scenario>"
	}
	return fmt.Sprintf("kit scen disconnect %s <out.aoe2scenario> units-player %d", inPath, player)
}

func scenarioStringPrefixDisconnectCommand(path, prefix string) string {
	inPath := path
	if strings.TrimSpace(inPath) == "" {
		inPath = "<in.aoe2scenario>"
	}
	return fmt.Sprintf("kit scen disconnect %s <out.aoe2scenario> string-prefix %q", inPath, prefix)
}

func scenarioStringContainsDisconnectCommand(path, text string) string {
	inPath := path
	if strings.TrimSpace(inPath) == "" {
		inPath = "<in.aoe2scenario>"
	}
	return fmt.Sprintf("kit scen disconnect %s <out.aoe2scenario> string-contains %q", inPath, text)
}

func deletePlanFloatPtrValue(value *float64) float64 {
	if value == nil {
		return 0
	}
	return *value
}

func recipeToMap(recipe Recipe) (map[string]any, error) {
	data, err := json.Marshal(recipe)
	if err != nil {
		return nil, err
	}
	out := map[string]any{}
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	for key, value := range out {
		switch typed := value.(type) {
		case []any:
			if len(typed) == 0 {
				delete(out, key)
			}
		case map[string]any:
			if len(typed) == 0 {
				delete(out, key)
			}
		case nil:
			delete(out, key)
		}
	}
	return out, nil
}

func normalizedDeletePlanRequest(kind string, request DeletePlanRequest) DeletePlanRequest {
	out := DeletePlanRequest{Kind: kind, ID: request.ID}
	if kind == "effect" || kind == "condition" {
		triggerIndex, childIndex := deletePlanChildTarget(request)
		out.ID = childIndex
		out.TriggerID = &triggerIndex
		out.ChildIndex = &childIndex
	}
	if kind == "effect_type" || kind == "condition_type" {
		out.ID = request.ID
		out.TargetName = request.TargetName
		out.TriggerID = request.TriggerID
		out.TargetPrefix = request.TargetPrefix
	}
	if kind == "effect_text_prefix" || kind == "effect_text_contains" {
		out.TargetText = request.TargetText
		out.TriggerID = request.TriggerID
		out.TargetPrefix = request.TargetPrefix
	}
	if kind == "system_prefix" {
		out.TargetPrefix = request.TargetPrefix
	}
	if kind == "unit_caption" {
		out.TargetCaption = request.TargetCaption
		out.Player = request.Player
	}
	if kind == "unit_caption_prefix" {
		out.TargetPrefix = request.TargetPrefix
		out.Player = request.Player
	}
	if kind == "unit_caption_contains" {
		out.TargetText = request.TargetText
		out.Player = request.Player
	}
	if kind == "unit_type" {
		out.UnitConst = request.UnitConst
		out.Player = request.Player
	}
	if kind == "units_player" {
		out.Player = request.Player
	}
	if kind == "trigger_name" || kind == "variable_name" {
		out.TargetName = request.TargetName
	}
	if kind == "trigger_prefix" {
		out.TargetPrefix = request.TargetPrefix
	}
	if kind == "trigger_contains" {
		out.TargetText = request.TargetText
	}
	if kind == "variable_prefix" {
		out.TargetPrefix = request.TargetPrefix
	}
	if kind == "variable_contains" {
		out.TargetText = request.TargetText
	}
	if kind == "string_prefix" {
		out.TargetPrefix = request.TargetPrefix
	}
	if kind == "string_text" {
		out.TargetText = request.TargetText
	}
	if kind == "string_contains" {
		out.TargetText = request.TargetText
	}
	if kind == "units_area" {
		out.AreaX1 = request.AreaX1
		out.AreaY1 = request.AreaY1
		out.AreaX2 = request.AreaX2
		out.AreaY2 = request.AreaY2
		out.Player = request.Player
		out.UnitConst = request.UnitConst
	}
	return out
}

func deletePlanChildTarget(request DeletePlanRequest) (int, int) {
	triggerIndex := request.ID
	childIndex := request.ID
	if request.TriggerID != nil {
		triggerIndex = *request.TriggerID
	}
	if request.ChildIndex != nil {
		childIndex = *request.ChildIndex
	}
	return triggerIndex, childIndex
}

func (f *File) validateTriggerIndex(index int) error {
	_, err := f.triggerByIndex(index)
	return err
}

func uniqueSortedInts(values []int) []int {
	if len(values) == 0 {
		return nil
	}
	seen := map[int]bool{}
	out := make([]int, 0, len(values))
	for _, value := range values {
		if seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	sort.Ints(out)
	return out
}

func (f *File) addScriptCallDeleteCaveats(report *DeletePlanReport, indexes []int) {
	if f == nil || report == nil || len(indexes) == 0 {
		return
	}
	scriptCallType, ok := EffectTypeForOp("script_call")
	if !ok {
		scriptCallType = 55
	}
	triggerCount := 0
	effectCount := 0
	samples := []string{}
	for _, index := range uniqueSortedInts(indexes) {
		trigger, err := f.triggerByIndex(index)
		if err != nil {
			continue
		}
		triggerHasCall := false
		name, _ := trigger.stringValue("name")
		for _, effect := range trigger.list("effect_data") {
			effectType, ok := effect.intValue("effect_type")
			if !ok || effectType != scriptCallType {
				continue
			}
			effectCount++
			triggerHasCall = true
			if len(samples) < 4 {
				message, _ := effect.stringValue("message")
				message = strings.TrimSpace(message)
				if message == "" {
					message = "<empty script_call message>"
				}
				if len(message) > 80 {
					message = message[:77] + "..."
				}
				if name == "" {
					name = fmt.Sprintf("trigger %d", index)
				}
				samples = append(samples, fmt.Sprintf("%d:%q -> %s", index, name, message))
			}
		}
		if triggerHasCall {
			triggerCount++
		}
	}
	if effectCount == 0 {
		return
	}
	report.StructuralCaveats = append(report.StructuralCaveats,
		fmt.Sprintf("semantic warning: matched trigger deletion/tombstone touches %d script_call effect(s) across %d trigger(s); structure checks cannot prove external XS/module expectations still make sense", effectCount, triggerCount))
	if len(samples) > 0 {
		report.StructuralCaveats = append(report.StructuralCaveats, "script_call samples: "+strings.Join(samples, "; "))
	}
}

func (f *File) triggerByIndex(index int) (*parsedNode, error) {
	triggers := f.root.section("Triggers")
	if triggers == nil {
		return nil, fmt.Errorf("missing Triggers section")
	}
	triggerData := triggers.list("trigger_data")
	if index < 0 || index >= len(triggerData) {
		return nil, fmt.Errorf("trigger index %d out of range 0..%d", index, len(triggerData)-1)
	}
	return triggerData[index], nil
}

func normalizeReferenceKind(kind string) (string, error) {
	kind = strings.ToLower(strings.TrimSpace(kind))
	switch kind {
	case "unit", "object":
		return "unit", nil
	case "trigger", "variable", "string":
		return kind, nil
	default:
		return "", fmt.Errorf("unknown delete-plan kind %q; expected unit, trigger, variable, or string", kind)
	}
}

func normalizeDeletePlanKind(kind string) (string, error) {
	kind = strings.ToLower(strings.TrimSpace(kind))
	switch kind {
	case "unit", "object":
		return "unit", nil
	case "unit_caption", "unit-caption", "object_caption", "object-caption", "caption":
		return "unit_caption", nil
	case "unit_caption_prefix", "unit-caption-prefix", "object_caption_prefix", "object-caption-prefix", "caption_prefix", "caption-prefix":
		return "unit_caption_prefix", nil
	case "unit_caption_contains", "unit-caption-contains", "object_caption_contains", "object-caption-contains", "caption_contains", "caption-contains":
		return "unit_caption_contains", nil
	case "unit_type", "unit-type", "units_type", "units-type", "unit_const", "unit-const", "units_const", "units-const":
		return "unit_type", nil
	case "units_player", "units-player", "player_units", "player-units":
		return "units_player", nil
	case "trigger_name", "trigger-name":
		return "trigger_name", nil
	case "trigger_prefix", "trigger-prefix":
		return "trigger_prefix", nil
	case "trigger_contains", "trigger-contains", "trigger_grep", "trigger-grep":
		return "trigger_contains", nil
	case "variable_name", "variable-name":
		return "variable_name", nil
	case "variable_prefix", "variable-prefix":
		return "variable_prefix", nil
	case "variable_contains", "variable-contains":
		return "variable_contains", nil
	case "string_text", "string-text":
		return "string_text", nil
	case "string_prefix", "string-prefix":
		return "string_prefix", nil
	case "string_contains", "string-contains":
		return "string_contains", nil
	case "units_area", "units-area", "area_units", "area-units":
		return "units_area", nil
	case "effect_type", "effect-type", "effects_type", "effects-type":
		return "effect_type", nil
	case "condition_type", "condition-type", "conditions_type", "conditions-type":
		return "condition_type", nil
	case "effect_text_prefix", "effect-text-prefix", "effects_text_prefix", "effects-text-prefix":
		return "effect_text_prefix", nil
	case "effect_text_contains", "effect-text-contains", "effects_text_contains", "effects-text-contains":
		return "effect_text_contains", nil
	case "system_prefix", "system-prefix":
		return "system_prefix", nil
	case "trigger", "variable", "string", "effect", "condition":
		return kind, nil
	default:
		return "", fmt.Errorf("unknown delete-plan kind %q; expected system-prefix, unit, unit-caption, unit-caption-prefix, unit-caption-contains, unit-type, units-player, units-area, trigger, trigger-name, trigger-prefix, trigger-contains, variable, variable-name, variable-prefix, variable-contains, string, string-text, string-prefix, string-contains, effect, effect-type, effect-text-prefix, effect-text-contains, condition, or condition-type", kind)
	}
}

type systemPrefixMatches struct {
	Prefix           string
	TriggerIndexes   []int
	VariableIDs      []int
	StringIDs        []int
	UnitReferenceIDs []int
	UnitMatches      []unitAreaMatch
}

func (m systemPrefixMatches) resolvedSets() map[string][]int {
	out := map[string][]int{}
	if len(m.TriggerIndexes) > 0 {
		out["triggers"] = append([]int(nil), m.TriggerIndexes...)
	}
	if len(m.VariableIDs) > 0 {
		out["variables"] = append([]int(nil), m.VariableIDs...)
	}
	if len(m.StringIDs) > 0 {
		out["strings"] = append([]int(nil), m.StringIDs...)
	}
	if len(m.UnitReferenceIDs) > 0 {
		out["units"] = append([]int(nil), m.UnitReferenceIDs...)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func (f *File) matchSystemPrefix(prefix string) (systemPrefixMatches, error) {
	if strings.TrimSpace(prefix) == "" {
		return systemPrefixMatches{}, fmt.Errorf("system prefix must be non-empty")
	}
	matches := systemPrefixMatches{Prefix: prefix}
	if indexes, err := f.findOptionalTriggerIndexesByNamePrefix(prefix); err != nil {
		return systemPrefixMatches{}, err
	} else {
		matches.TriggerIndexes = indexes
	}
	if ids, err := f.findOptionalVariableIDsByNamePrefix(prefix); err != nil {
		return systemPrefixMatches{}, err
	} else {
		matches.VariableIDs = ids
	}
	if ids, err := f.findOptionalStringIDsByTextPrefix(prefix); err != nil {
		return systemPrefixMatches{}, err
	} else {
		matches.StringIDs = ids
	}
	if unitMatches, err := f.findOptionalUnitsByCaptionPrefix(prefix); err != nil {
		return systemPrefixMatches{}, err
	} else {
		matches.UnitMatches = unitMatches
	}
	if len(matches.UnitMatches) > 0 {
		ids, err := unitReferenceIDsFromMatches(prefix, matches.UnitMatches)
		if err != nil {
			return systemPrefixMatches{}, err
		}
		matches.UnitReferenceIDs = ids
	}
	if len(matches.TriggerIndexes) == 0 && len(matches.VariableIDs) == 0 && len(matches.StringIDs) == 0 && len(matches.UnitReferenceIDs) == 0 {
		return systemPrefixMatches{}, fmt.Errorf("system prefix %q matched no triggers, variables, strings, or captioned units", prefix)
	}
	return matches, nil
}

func (f *File) systemPrefixReferences(matches systemPrefixMatches) (ReferenceReport, ReferenceReport, error) {
	external := ReferenceReport{Path: f.Path, Version: f.Version, Verification: "structure_verified_not_engine_verified", Summary: ReferenceSummary{ByKind: map[string]int{}}}
	internal := ReferenceReport{Path: f.Path, Version: f.Version, Verification: "structure_verified_not_engine_verified", Summary: ReferenceSummary{ByKind: map[string]int{}}}
	triggerSet := intSet(matches.TriggerIndexes)
	unitSet := intSet(matches.UnitReferenceIDs)
	addRows := func(refs ReferenceReport) {
		for _, ref := range refs.References {
			if systemPrefixReferenceIsInternal(ref, triggerSet, unitSet) {
				addReferenceToReport(&internal, ref)
			} else {
				addReferenceToReport(&external, ref)
			}
		}
	}
	for _, id := range matches.TriggerIndexes {
		refs, err := f.References(ReferenceOptions{Kind: "trigger", TargetID: &id})
		if err != nil {
			return ReferenceReport{}, ReferenceReport{}, err
		}
		addRows(refs)
	}
	for _, id := range matches.VariableIDs {
		refs, err := f.References(ReferenceOptions{Kind: "variable", TargetID: &id})
		if err != nil {
			return ReferenceReport{}, ReferenceReport{}, err
		}
		addRows(refs)
	}
	for _, id := range matches.StringIDs {
		refs, err := f.References(ReferenceOptions{Kind: "string", TargetID: &id})
		if err != nil {
			return ReferenceReport{}, ReferenceReport{}, err
		}
		addRows(refs)
	}
	for _, id := range matches.UnitReferenceIDs {
		refs, err := f.References(ReferenceOptions{Kind: "unit", TargetID: &id})
		if err != nil {
			return ReferenceReport{}, ReferenceReport{}, err
		}
		addRows(refs)
	}
	sortReferenceReport(&external)
	sortReferenceReport(&internal)
	return external, internal, nil
}

func systemPrefixReferenceIsInternal(ref ScenarioReference, triggerSet map[int]bool, unitSet map[int]bool) bool {
	if ref.SourceKind == "trigger" && triggerSet[ref.TriggerIndex] {
		return true
	}
	if ref.SourceKind == "unit" && unitSet[ref.UnitReferenceID] {
		return true
	}
	return false
}

func sortReferenceReport(report *ReferenceReport) {
	sort.Slice(report.References, func(i, j int) bool {
		a, b := report.References[i], report.References[j]
		if a.Kind != b.Kind {
			return a.Kind < b.Kind
		}
		if a.TargetID != b.TargetID {
			return a.TargetID < b.TargetID
		}
		return a.SourcePath < b.SourcePath
	})
}

type triggerChildRows struct {
	TriggerIndex  int
	ChildIndexes  []int
	TriggerName   string
	ChildDataName string
}

func (f *File) findTriggerChildRowsByType(kind string, typeID int, triggerFilter *int, triggerPrefix string) ([]triggerChildRows, error) {
	triggers := f.root.section("Triggers")
	if triggers == nil {
		return nil, fmt.Errorf("missing Triggers section")
	}
	triggerNodes := triggers.list("trigger_data")
	if triggerFilter != nil && (*triggerFilter < 0 || *triggerFilter >= len(triggerNodes)) {
		return nil, fmt.Errorf("trigger index %d out of range 0..%d", *triggerFilter, len(triggerNodes)-1)
	}
	dataField := "effect_data"
	typeField := "effect_type"
	if kind == "condition_type" {
		dataField = "condition_data"
		typeField = "condition_type"
	}
	out := []triggerChildRows{}
	for triggerIndex, trigger := range triggerNodes {
		if triggerFilter != nil && triggerIndex != *triggerFilter {
			continue
		}
		triggerName, _ := trigger.stringValue("trigger_name")
		if strings.TrimSpace(triggerPrefix) != "" && !strings.HasPrefix(triggerName, triggerPrefix) {
			continue
		}
		var childIndexes []int
		for childIndex, child := range trigger.list(dataField) {
			childType, ok := child.intValue(typeField)
			if !ok || childType != typeID {
				continue
			}
			childIndexes = append(childIndexes, childIndex)
		}
		if len(childIndexes) > 0 {
			out = append(out, triggerChildRows{
				TriggerIndex:  triggerIndex,
				ChildIndexes:  childIndexes,
				TriggerName:   triggerName,
				ChildDataName: dataField,
			})
		}
	}
	if len(out) == 0 {
		scope := "scenario"
		if triggerFilter != nil {
			scope = fmt.Sprintf("trigger %d", *triggerFilter)
		}
		if strings.TrimSpace(triggerPrefix) != "" {
			scope = fmt.Sprintf("trigger-prefix %q", triggerPrefix)
		}
		return nil, fmt.Errorf("%s type %d matched no rows in %s", strings.TrimSuffix(kind, "_type"), typeID, scope)
	}
	return out, nil
}

func flattenTriggerChildRows(rows []triggerChildRows) []int {
	out := []int{}
	for _, row := range rows {
		for _, childIndex := range row.ChildIndexes {
			out = append(out, row.TriggerIndex, childIndex)
		}
	}
	return out
}

func (f *File) findEffectRowsByText(text string, triggerFilter *int, triggerPrefix string, prefixOnly bool) ([]triggerChildRows, error) {
	if strings.TrimSpace(text) == "" {
		if prefixOnly {
			return nil, fmt.Errorf("effect text prefix must be non-empty")
		}
		return nil, fmt.Errorf("effect text contains marker must be non-empty")
	}
	triggers := f.root.section("Triggers")
	if triggers == nil {
		return nil, fmt.Errorf("missing Triggers section")
	}
	triggerNodes := triggers.list("trigger_data")
	if triggerFilter != nil && (*triggerFilter < 0 || *triggerFilter >= len(triggerNodes)) {
		return nil, fmt.Errorf("trigger index %d out of range 0..%d", *triggerFilter, len(triggerNodes)-1)
	}
	out := []triggerChildRows{}
	for triggerIndex, trigger := range triggerNodes {
		if triggerFilter != nil && triggerIndex != *triggerFilter {
			continue
		}
		triggerName, _ := trigger.stringValue("trigger_name")
		if strings.TrimSpace(triggerPrefix) != "" && !strings.HasPrefix(triggerName, triggerPrefix) {
			continue
		}
		var childIndexes []int
		for childIndex, effect := range trigger.list("effect_data") {
			message, _ := effect.stringValue("message")
			matches := strings.Contains(message, text)
			if prefixOnly {
				matches = strings.HasPrefix(message, text)
			}
			if matches {
				childIndexes = append(childIndexes, childIndex)
			}
		}
		if len(childIndexes) > 0 {
			out = append(out, triggerChildRows{
				TriggerIndex:  triggerIndex,
				ChildIndexes:  childIndexes,
				TriggerName:   triggerName,
				ChildDataName: "effect_data",
			})
		}
	}
	if len(out) == 0 {
		scope := "scenario"
		if triggerFilter != nil {
			scope = fmt.Sprintf("trigger %d", *triggerFilter)
		}
		if strings.TrimSpace(triggerPrefix) != "" {
			scope = fmt.Sprintf("trigger-prefix %q", triggerPrefix)
		}
		if prefixOnly {
			return nil, fmt.Errorf("effect text prefix %q matched no rows in %s", text, scope)
		}
		return nil, fmt.Errorf("effect text contains marker %q matched no rows in %s", text, scope)
	}
	return out, nil
}

func (f *File) findUniqueTriggerIndexByName(name string) (int, error) {
	triggers := f.root.section("Triggers")
	if triggers == nil {
		return 0, fmt.Errorf("missing Triggers section")
	}
	triggerNodes := triggers.list("trigger_data")
	foundIndex := -1
	matches := 0
	for i, trigger := range triggerNodes {
		triggerName, _ := trigger.stringValue("trigger_name")
		if triggerName != name {
			continue
		}
		foundIndex = i
		matches++
	}
	if matches == 0 {
		return 0, fmt.Errorf("target_name %q not found", name)
	}
	if matches > 1 {
		return 0, fmt.Errorf("target_name %q matched %d triggers; use trigger index", name, matches)
	}
	return foundIndex, nil
}

func (f *File) findTriggerIndexesByNamePrefix(prefix string) ([]int, error) {
	triggers := f.root.section("Triggers")
	if triggers == nil {
		return nil, fmt.Errorf("missing Triggers section")
	}
	triggerNodes := triggers.list("trigger_data")
	var indexes []int
	for i, trigger := range triggerNodes {
		triggerName, _ := trigger.stringValue("trigger_name")
		if strings.HasPrefix(triggerName, prefix) {
			indexes = append(indexes, i)
		}
	}
	if len(indexes) == 0 {
		return nil, fmt.Errorf("target_prefix %q matched no triggers", prefix)
	}
	return indexes, nil
}

func (f *File) findTriggerIndexesContainingText(text string) ([]int, error) {
	if strings.TrimSpace(text) == "" {
		return nil, fmt.Errorf("trigger contains text must be non-empty")
	}
	triggers := f.root.section("Triggers")
	if triggers == nil {
		return nil, fmt.Errorf("missing Triggers section")
	}
	triggerNodes := triggers.list("trigger_data")
	var indexes []int
	for i, trigger := range triggerNodes {
		triggerName, _ := trigger.stringValue("trigger_name")
		if strings.Contains(triggerName, text) {
			indexes = append(indexes, i)
			continue
		}
		for _, effect := range trigger.list("effect_data") {
			message, _ := effect.stringValue("message")
			if strings.Contains(message, text) {
				indexes = append(indexes, i)
				break
			}
		}
	}
	if len(indexes) == 0 {
		return nil, fmt.Errorf("trigger text %q matched no trigger names or raw effect messages", text)
	}
	return indexes, nil
}

func (f *File) findOptionalTriggerIndexesByNamePrefix(prefix string) ([]int, error) {
	triggers := f.root.section("Triggers")
	if triggers == nil {
		return nil, fmt.Errorf("missing Triggers section")
	}
	triggerNodes := triggers.list("trigger_data")
	var indexes []int
	for i, trigger := range triggerNodes {
		triggerName, _ := trigger.stringValue("trigger_name")
		if strings.HasPrefix(triggerName, prefix) {
			indexes = append(indexes, i)
		}
	}
	return indexes, nil
}

func (f *File) referencesForTriggerIndexes(indexes []int) (ReferenceReport, error) {
	out := ReferenceReport{
		Path:         f.Path,
		Version:      f.Version,
		Verification: "structure_verified_not_engine_verified",
		Summary:      ReferenceSummary{ByKind: map[string]int{}},
	}
	for _, index := range indexes {
		refs, err := f.References(ReferenceOptions{Kind: "trigger", TargetID: &index})
		if err != nil {
			return ReferenceReport{}, err
		}
		out.References = append(out.References, refs.References...)
		out.Summary.Total += refs.Summary.Total
		for kind, count := range refs.Summary.ByKind {
			out.Summary.ByKind[kind] += count
		}
	}
	sort.Slice(out.References, func(i, j int) bool {
		a, b := out.References[i], out.References[j]
		if a.Kind != b.Kind {
			return a.Kind < b.Kind
		}
		if a.TargetID != b.TargetID {
			return a.TargetID < b.TargetID
		}
		return a.SourcePath < b.SourcePath
	})
	return out, nil
}

func splitTriggerBatchReferences(indexes []int, refs ReferenceReport) (ReferenceReport, ReferenceReport) {
	matched := intSet(indexes)
	external := emptyReferenceReportLike(refs)
	internal := emptyReferenceReportLike(refs)
	for _, ref := range refs.References {
		if ref.SourceKind == "trigger" && matched[ref.TriggerIndex] {
			addReferenceToReport(&internal, ref)
		} else {
			addReferenceToReport(&external, ref)
		}
	}
	return external, internal
}

func emptyReferenceReportLike(refs ReferenceReport) ReferenceReport {
	return ReferenceReport{
		Path:         refs.Path,
		Version:      refs.Version,
		Verification: refs.Verification,
		Summary:      ReferenceSummary{ByKind: map[string]int{}},
	}
}

func addReferenceToReport(report *ReferenceReport, ref ScenarioReference) {
	report.References = append(report.References, ref)
	report.Summary.Total++
	if report.Summary.ByKind == nil {
		report.Summary.ByKind = map[string]int{}
	}
	report.Summary.ByKind[ref.Kind]++
}

func descendingInts(values []int) []int {
	out := append([]int(nil), values...)
	sort.Sort(sort.Reverse(sort.IntSlice(out)))
	return out
}

func (f *File) findUniqueVariableIDByName(name string) (int, error) {
	triggers := f.root.section("Triggers")
	if triggers == nil {
		return 0, fmt.Errorf("missing Triggers section")
	}
	foundID := 0
	matches := 0
	for _, variable := range triggers.list("variable_data") {
		variableName, _ := variable.stringValue("variable_name")
		if variableName != name {
			continue
		}
		variableID, ok := variable.intValue("variable_id")
		if !ok {
			return 0, fmt.Errorf("variable name %q matched a record without variable_id", name)
		}
		foundID = variableID
		matches++
	}
	if matches == 0 {
		return 0, fmt.Errorf("variable name %q not found", name)
	}
	if matches > 1 {
		return 0, fmt.Errorf("variable name %q matched %d variables; use variable id", name, matches)
	}
	return foundID, nil
}

func (f *File) findVariableIDsByNamePrefix(prefix string) ([]int, error) {
	return f.findVariableIDsByNameText(prefix, true)
}

func (f *File) findVariableIDsByNameContains(text string) ([]int, error) {
	return f.findVariableIDsByNameText(text, false)
}

func (f *File) findVariableIDsByNameText(text string, prefixOnly bool) ([]int, error) {
	if strings.TrimSpace(text) == "" {
		if prefixOnly {
			return nil, fmt.Errorf("variable prefix must be non-empty")
		}
		return nil, fmt.Errorf("variable contains marker must be non-empty")
	}
	triggers := f.root.section("Triggers")
	if triggers == nil {
		return nil, fmt.Errorf("missing Triggers section")
	}
	var ids []int
	for _, variable := range triggers.list("variable_data") {
		variableName, _ := variable.stringValue("variable_name")
		matched := strings.Contains(variableName, text)
		if prefixOnly {
			matched = strings.HasPrefix(variableName, text)
		}
		if !matched {
			continue
		}
		variableID, ok := variable.intValue("variable_id")
		if !ok {
			if prefixOnly {
				return nil, fmt.Errorf("variable prefix %q matched a record without variable_id", text)
			}
			return nil, fmt.Errorf("variable contains marker %q matched a record without variable_id", text)
		}
		ids = append(ids, variableID)
	}
	if len(ids) == 0 {
		if prefixOnly {
			return nil, fmt.Errorf("variable prefix %q matched no variables", text)
		}
		return nil, fmt.Errorf("variable contains marker %q matched no variables", text)
	}
	sort.Ints(ids)
	return ids, nil
}

func (f *File) findOptionalVariableIDsByNamePrefix(prefix string) ([]int, error) {
	triggers := f.root.section("Triggers")
	if triggers == nil {
		return nil, fmt.Errorf("missing Triggers section")
	}
	var ids []int
	for _, variable := range triggers.list("variable_data") {
		variableName, _ := variable.stringValue("variable_name")
		if !strings.HasPrefix(variableName, prefix) {
			continue
		}
		variableID, ok := variable.intValue("variable_id")
		if !ok {
			return nil, fmt.Errorf("variable prefix %q matched a record without variable_id", prefix)
		}
		ids = append(ids, variableID)
	}
	sort.Ints(ids)
	return ids, nil
}

func (f *File) referencesForVariableIDs(ids []int) (ReferenceReport, error) {
	out := ReferenceReport{
		Path:         f.Path,
		Version:      f.Version,
		Verification: "structure_verified_not_engine_verified",
		Summary:      ReferenceSummary{ByKind: map[string]int{}},
	}
	for _, id := range ids {
		refs, err := f.References(ReferenceOptions{Kind: "variable", TargetID: &id})
		if err != nil {
			return ReferenceReport{}, err
		}
		out.References = append(out.References, refs.References...)
		out.Summary.Total += refs.Summary.Total
		for kind, count := range refs.Summary.ByKind {
			out.Summary.ByKind[kind] += count
		}
	}
	sort.Slice(out.References, func(i, j int) bool {
		a, b := out.References[i], out.References[j]
		if a.Kind != b.Kind {
			return a.Kind < b.Kind
		}
		if a.TargetID != b.TargetID {
			return a.TargetID < b.TargetID
		}
		return a.SourcePath < b.SourcePath
	})
	return out, nil
}

func (f *File) findUniqueStringIDByText(text string) (int, error) {
	values, err := f.scenarioStringValues()
	if err != nil {
		return 0, err
	}
	foundID := -1
	matches := 0
	for id, value := range values {
		if value != text {
			continue
		}
		foundID = id
		matches++
	}
	if matches == 0 {
		return 0, fmt.Errorf("string text %q not found", text)
	}
	if matches > 1 {
		return 0, fmt.Errorf("string text %q matched %d string ids; use string id", text, matches)
	}
	return foundID, nil
}

func (f *File) findStringIDsByTextPrefix(prefix string) ([]int, error) {
	return f.findStringIDsByText(prefix, true)
}

func (f *File) findStringIDsByTextContains(text string) ([]int, error) {
	return f.findStringIDsByText(text, false)
}

func (f *File) findStringIDsByText(text string, prefixOnly bool) ([]int, error) {
	if strings.TrimSpace(text) == "" {
		if prefixOnly {
			return nil, fmt.Errorf("string prefix must be non-empty")
		}
		return nil, fmt.Errorf("string contains marker must be non-empty")
	}
	values, err := f.scenarioStringValues()
	if err != nil {
		return nil, err
	}
	var ids []int
	for id, value := range values {
		matched := strings.Contains(value, text)
		if prefixOnly {
			matched = strings.HasPrefix(value, text)
		}
		if matched {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		if prefixOnly {
			return nil, fmt.Errorf("string prefix %q matched no string ids", text)
		}
		return nil, fmt.Errorf("string contains marker %q matched no string ids", text)
	}
	sort.Ints(ids)
	return ids, nil
}

func (f *File) findOptionalStringIDsByTextPrefix(prefix string) ([]int, error) {
	values, err := f.scenarioStringValues()
	if err != nil {
		return nil, err
	}
	var ids []int
	for id, value := range values {
		if strings.HasPrefix(value, prefix) {
			ids = append(ids, id)
		}
	}
	sort.Ints(ids)
	return ids, nil
}

func (f *File) findOptionalUnitsByCaptionPrefix(prefix string) ([]unitAreaMatch, error) {
	if strings.TrimSpace(prefix) == "" {
		return nil, fmt.Errorf("unit caption prefix must be non-empty")
	}
	playerSections, err := f.unitPlayerSections()
	if err != nil {
		return nil, err
	}
	var matches []unitAreaMatch
	for player, playerSection := range playerSections {
		unitField := playerSection.field("units")
		if unitField == nil {
			continue
		}
		for index, unit := range unitField.Elements {
			caption, _ := unit.stringValue("caption_string")
			if strings.HasPrefix(caption, prefix) {
				matches = append(matches, unitAreaMatch{Unit: unit, PlayerSection: playerSection, Player: player, Index: index})
			}
		}
	}
	return matches, nil
}

func (f *File) referencesForStringIDs(ids []int) (ReferenceReport, error) {
	out := ReferenceReport{
		Path:         f.Path,
		Version:      f.Version,
		Verification: "structure_verified_not_engine_verified",
		Summary:      ReferenceSummary{ByKind: map[string]int{}},
	}
	for _, id := range ids {
		refs, err := f.References(ReferenceOptions{Kind: "string", TargetID: &id})
		if err != nil {
			return ReferenceReport{}, err
		}
		out.References = append(out.References, refs.References...)
		out.Summary.Total += refs.Summary.Total
		for kind, count := range refs.Summary.ByKind {
			out.Summary.ByKind[kind] += count
		}
	}
	sort.Slice(out.References, func(i, j int) bool {
		a, b := out.References[i], out.References[j]
		if a.Kind != b.Kind {
			return a.Kind < b.Kind
		}
		if a.TargetID != b.TargetID {
			return a.TargetID < b.TargetID
		}
		return a.SourcePath < b.SourcePath
	})
	return out, nil
}

func unitReferenceIDsFromMatches(selector string, targets []unitAreaMatch) ([]int, error) {
	seen := map[int]bool{}
	ids := make([]int, 0, len(targets))
	for _, target := range targets {
		referenceID, ok := target.Unit.intValue("reference_id")
		if !ok || referenceID < 0 {
			return nil, fmt.Errorf("unit selector %q matched player %d unit index %d without a valid reference_id", selector, target.Player, target.Index)
		}
		if seen[referenceID] {
			continue
		}
		seen[referenceID] = true
		ids = append(ids, referenceID)
	}
	sort.Ints(ids)
	return ids, nil
}

func (f *File) referencesForUnitAreaTargets(targets []unitAreaMatch) (ReferenceReport, error) {
	out := ReferenceReport{
		Path:         f.Path,
		Version:      f.Version,
		Verification: "structure_verified_not_engine_verified",
		Summary:      ReferenceSummary{ByKind: map[string]int{}},
	}
	seen := map[int]bool{}
	for _, target := range targets {
		referenceID, ok := target.Unit.intValue("reference_id")
		if !ok || referenceID < 0 || seen[referenceID] {
			continue
		}
		seen[referenceID] = true
		refs, err := f.References(ReferenceOptions{Kind: "unit", TargetID: &referenceID})
		if err != nil {
			return ReferenceReport{}, err
		}
		for _, ref := range refs.References {
			out.References = append(out.References, ref)
			out.Summary.Total++
			out.Summary.ByKind[ref.Kind]++
		}
	}
	return out, nil
}
