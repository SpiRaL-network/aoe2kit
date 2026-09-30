package scenario

import (
	"fmt"
	"sort"
)

func TriggerDisconnectRecipeFile(path string, targetID int) (ReferenceReport, Recipe, error) {
	file, err := Open(path)
	if err != nil {
		return ReferenceReport{}, Recipe{}, err
	}
	return file.TriggerDisconnectRecipe(targetID)
}

func (f *File) TriggerDisconnectRecipe(targetID int) (ReferenceReport, Recipe, error) {
	if targetID < 0 {
		return ReferenceReport{}, Recipe{}, fmt.Errorf("trigger id must be non-negative")
	}
	refs, err := f.References(ReferenceOptions{Kind: "trigger", TargetID: &targetID})
	if err != nil {
		return ReferenceReport{}, Recipe{}, err
	}
	recipe := triggerRowDisconnectRecipe(refs.References)
	return refs, recipe, nil
}

func TriggerNameDisconnectRecipeFile(path string, name string) (int, ReferenceReport, Recipe, error) {
	file, err := Open(path)
	if err != nil {
		return 0, ReferenceReport{}, Recipe{}, err
	}
	return file.TriggerNameDisconnectRecipe(name)
}

func (f *File) TriggerNameDisconnectRecipe(name string) (int, ReferenceReport, Recipe, error) {
	targetID, err := f.findUniqueTriggerIndexByName(name)
	if err != nil {
		return 0, ReferenceReport{}, Recipe{}, err
	}
	refs, recipe, err := f.TriggerDisconnectRecipe(targetID)
	if err != nil {
		return 0, ReferenceReport{}, Recipe{}, err
	}
	return targetID, refs, recipe, nil
}

func TriggerPrefixDisconnectRecipeFile(path string, prefix string) ([]int, ReferenceReport, Recipe, error) {
	file, err := Open(path)
	if err != nil {
		return nil, ReferenceReport{}, Recipe{}, err
	}
	return file.TriggerPrefixDisconnectRecipe(prefix)
}

func (f *File) TriggerPrefixDisconnectRecipe(prefix string) ([]int, ReferenceReport, Recipe, error) {
	indexes, err := f.findTriggerIndexesByNamePrefix(prefix)
	if err != nil {
		return nil, ReferenceReport{}, Recipe{}, err
	}
	refs, err := f.referencesForTriggerIndexes(indexes)
	if err != nil {
		return nil, ReferenceReport{}, Recipe{}, err
	}
	recipe := triggerRowDisconnectRecipe(refs.References)
	return indexes, refs, recipe, nil
}

func TriggerContainsDisconnectRecipeFile(path string, text string) ([]int, ReferenceReport, Recipe, error) {
	file, err := Open(path)
	if err != nil {
		return nil, ReferenceReport{}, Recipe{}, err
	}
	return file.TriggerContainsDisconnectRecipe(text)
}

func (f *File) TriggerContainsDisconnectRecipe(text string) ([]int, ReferenceReport, Recipe, error) {
	indexes, err := f.findTriggerIndexesContainingText(text)
	if err != nil {
		return nil, ReferenceReport{}, Recipe{}, err
	}
	refs, err := f.referencesForTriggerIndexes(indexes)
	if err != nil {
		return nil, ReferenceReport{}, Recipe{}, err
	}
	recipe := triggerRowDisconnectRecipe(refs.References)
	return indexes, refs, recipe, nil
}

func UnitDisconnectRecipeFile(path string, referenceID int) (ReferenceReport, Recipe, error) {
	file, err := Open(path)
	if err != nil {
		return ReferenceReport{}, Recipe{}, err
	}
	return file.UnitDisconnectRecipe(referenceID)
}

func (f *File) UnitDisconnectRecipe(referenceID int) (ReferenceReport, Recipe, error) {
	if referenceID < 0 {
		return ReferenceReport{}, Recipe{}, fmt.Errorf("unit reference_id must be non-negative")
	}
	refs, err := f.References(ReferenceOptions{Kind: "unit", TargetID: &referenceID})
	if err != nil {
		return ReferenceReport{}, Recipe{}, err
	}
	recipe := unitDisconnectRecipeFromRefs(refs.References)
	return refs, recipe, nil
}

func UnitCaptionDisconnectRecipeFile(path string, caption string, targetPlayer *int) (int, ReferenceReport, Recipe, error) {
	file, err := Open(path)
	if err != nil {
		return 0, ReferenceReport{}, Recipe{}, err
	}
	return file.UnitCaptionDisconnectRecipe(caption, targetPlayer)
}

func (f *File) UnitCaptionDisconnectRecipe(caption string, targetPlayer *int) (int, ReferenceReport, Recipe, error) {
	unit, _, _, err := f.findUnitByCaption(caption, targetPlayer)
	if err != nil {
		return 0, ReferenceReport{}, Recipe{}, err
	}
	referenceID, ok := unit.intValue("reference_id")
	if !ok || referenceID < 0 {
		return 0, ReferenceReport{}, Recipe{}, fmt.Errorf("unit-caption %q resolved to a unit without a valid reference_id", caption)
	}
	refs, recipe, err := f.UnitDisconnectRecipe(referenceID)
	if err != nil {
		return 0, ReferenceReport{}, Recipe{}, err
	}
	return referenceID, refs, recipe, nil
}

func UnitCaptionPrefixDisconnectRecipeFile(path string, prefix string, targetPlayer *int) ([]int, ReferenceReport, Recipe, error) {
	file, err := Open(path)
	if err != nil {
		return nil, ReferenceReport{}, Recipe{}, err
	}
	return file.UnitCaptionPrefixDisconnectRecipe(prefix, targetPlayer)
}

func (f *File) UnitCaptionPrefixDisconnectRecipe(prefix string, targetPlayer *int) ([]int, ReferenceReport, Recipe, error) {
	targets, err := f.findUnitsByCaptionText(prefix, targetPlayer, true)
	if err != nil {
		return nil, ReferenceReport{}, Recipe{}, err
	}
	referenceIDs, err := unitReferenceIDsFromMatches(prefix, targets)
	if err != nil {
		return nil, ReferenceReport{}, Recipe{}, err
	}
	refs, err := f.referencesForUnitAreaTargets(targets)
	if err != nil {
		return nil, ReferenceReport{}, Recipe{}, err
	}
	recipe := unitDisconnectRecipeFromRefs(refs.References)
	return referenceIDs, refs, recipe, nil
}

func UnitCaptionContainsDisconnectRecipeFile(path string, text string, targetPlayer *int) ([]int, ReferenceReport, Recipe, error) {
	file, err := Open(path)
	if err != nil {
		return nil, ReferenceReport{}, Recipe{}, err
	}
	return file.UnitCaptionContainsDisconnectRecipe(text, targetPlayer)
}

func (f *File) UnitCaptionContainsDisconnectRecipe(text string, targetPlayer *int) ([]int, ReferenceReport, Recipe, error) {
	targets, err := f.findUnitsByCaptionText(text, targetPlayer, false)
	if err != nil {
		return nil, ReferenceReport{}, Recipe{}, err
	}
	referenceIDs, err := unitReferenceIDsFromMatches(text, targets)
	if err != nil {
		return nil, ReferenceReport{}, Recipe{}, err
	}
	refs, err := f.referencesForUnitAreaTargets(targets)
	if err != nil {
		return nil, ReferenceReport{}, Recipe{}, err
	}
	recipe := unitDisconnectRecipeFromRefs(refs.References)
	return referenceIDs, refs, recipe, nil
}

func UnitTypeDisconnectRecipeFile(path string, unitConst int, targetPlayer *int) ([]int, ReferenceReport, Recipe, error) {
	file, err := Open(path)
	if err != nil {
		return nil, ReferenceReport{}, Recipe{}, err
	}
	return file.UnitTypeDisconnectRecipe(unitConst, targetPlayer)
}

func (f *File) UnitTypeDisconnectRecipe(unitConst int, targetPlayer *int) ([]int, ReferenceReport, Recipe, error) {
	targets, err := f.findUnitsByUnitConst(unitConst, targetPlayer)
	if err != nil {
		return nil, ReferenceReport{}, Recipe{}, err
	}
	referenceIDs, err := unitReferenceIDsFromMatches(fmt.Sprintf("unit_const %d", unitConst), targets)
	if err != nil {
		return nil, ReferenceReport{}, Recipe{}, err
	}
	refs, err := f.referencesForUnitAreaTargets(targets)
	if err != nil {
		return nil, ReferenceReport{}, Recipe{}, err
	}
	recipe := unitDisconnectRecipeFromRefs(refs.References)
	return referenceIDs, refs, recipe, nil
}

func UnitsPlayerDisconnectRecipeFile(path string, player int) ([]int, ReferenceReport, Recipe, error) {
	file, err := Open(path)
	if err != nil {
		return nil, ReferenceReport{}, Recipe{}, err
	}
	return file.UnitsPlayerDisconnectRecipe(player)
}

func (f *File) UnitsPlayerDisconnectRecipe(player int) ([]int, ReferenceReport, Recipe, error) {
	targets, err := f.findUnitsByPlayer(&player)
	if err != nil {
		return nil, ReferenceReport{}, Recipe{}, err
	}
	referenceIDs, err := unitReferenceIDsFromMatches(fmt.Sprintf("player %d", player), targets)
	if err != nil {
		return nil, ReferenceReport{}, Recipe{}, err
	}
	refs, err := f.referencesForUnitAreaTargets(targets)
	if err != nil {
		return nil, ReferenceReport{}, Recipe{}, err
	}
	recipe := unitDisconnectRecipeFromRefs(refs.References)
	return referenceIDs, refs, recipe, nil
}

func (f *File) UnitsAreaDisconnectRecipe(request DeletePlanRequest) (ReferenceReport, Recipe, error) {
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
		return ReferenceReport{}, Recipe{}, err
	}
	refs, err := f.referencesForUnitAreaTargets(targets)
	if err != nil {
		return ReferenceReport{}, Recipe{}, err
	}
	return refs, unitDisconnectRecipeFromRefs(refs.References), nil
}

func VariableDisconnectRecipeFile(path string, variableID int) (ReferenceReport, Recipe, error) {
	file, err := Open(path)
	if err != nil {
		return ReferenceReport{}, Recipe{}, err
	}
	return file.VariableDisconnectRecipe(variableID)
}

func (f *File) VariableDisconnectRecipe(variableID int) (ReferenceReport, Recipe, error) {
	if variableID < 0 {
		return ReferenceReport{}, Recipe{}, fmt.Errorf("variable id must be non-negative")
	}
	refs, err := f.References(ReferenceOptions{Kind: "variable", TargetID: &variableID})
	if err != nil {
		return ReferenceReport{}, Recipe{}, err
	}
	recipe := triggerRowDisconnectRecipe(refs.References)
	return refs, recipe, nil
}

func VariableNameDisconnectRecipeFile(path string, name string) (int, ReferenceReport, Recipe, error) {
	file, err := Open(path)
	if err != nil {
		return 0, ReferenceReport{}, Recipe{}, err
	}
	return file.VariableNameDisconnectRecipe(name)
}

func (f *File) VariableNameDisconnectRecipe(name string) (int, ReferenceReport, Recipe, error) {
	variableID, err := f.findUniqueVariableIDByName(name)
	if err != nil {
		return 0, ReferenceReport{}, Recipe{}, err
	}
	refs, recipe, err := f.VariableDisconnectRecipe(variableID)
	if err != nil {
		return 0, ReferenceReport{}, Recipe{}, err
	}
	return variableID, refs, recipe, nil
}

func VariablePrefixDisconnectRecipeFile(path string, prefix string) ([]int, ReferenceReport, Recipe, error) {
	file, err := Open(path)
	if err != nil {
		return nil, ReferenceReport{}, Recipe{}, err
	}
	return file.VariablePrefixDisconnectRecipe(prefix)
}

func (f *File) VariablePrefixDisconnectRecipe(prefix string) ([]int, ReferenceReport, Recipe, error) {
	ids, err := f.findVariableIDsByNamePrefix(prefix)
	if err != nil {
		return nil, ReferenceReport{}, Recipe{}, err
	}
	refs, err := f.referencesForVariableIDs(ids)
	if err != nil {
		return nil, ReferenceReport{}, Recipe{}, err
	}
	recipe := triggerRowDisconnectRecipe(refs.References)
	return ids, refs, recipe, nil
}

func VariableContainsDisconnectRecipeFile(path string, text string) ([]int, ReferenceReport, Recipe, error) {
	file, err := Open(path)
	if err != nil {
		return nil, ReferenceReport{}, Recipe{}, err
	}
	return file.VariableContainsDisconnectRecipe(text)
}

func (f *File) VariableContainsDisconnectRecipe(text string) ([]int, ReferenceReport, Recipe, error) {
	ids, err := f.findVariableIDsByNameContains(text)
	if err != nil {
		return nil, ReferenceReport{}, Recipe{}, err
	}
	refs, err := f.referencesForVariableIDs(ids)
	if err != nil {
		return nil, ReferenceReport{}, Recipe{}, err
	}
	recipe := triggerRowDisconnectRecipe(refs.References)
	return ids, refs, recipe, nil
}

func unitDisconnectRecipeFromRefs(refs []ScenarioReference) Recipe {
	recipe := triggerRowDisconnectRecipe(refs)
	clearGarrison := -1
	seenUnits := map[int]bool{}
	for _, ref := range refs {
		if ref.SourceKind != "unit" || ref.Field != "garrisoned_in_id" || ref.UnitReferenceID < 0 {
			continue
		}
		if seenUnits[ref.UnitReferenceID] {
			continue
		}
		seenUnits[ref.UnitReferenceID] = true
		sourceRef := ref.UnitReferenceID
		recipe.Units = append(recipe.Units, UnitRecipe{
			Op:             "edit_unit",
			ReferenceID:    &sourceRef,
			GarrisonedInID: &clearGarrison,
		})
	}
	return recipe
}

func StringDisconnectRecipeFile(path string, stringID int) (ReferenceReport, Recipe, error) {
	file, err := Open(path)
	if err != nil {
		return ReferenceReport{}, Recipe{}, err
	}
	return file.StringDisconnectRecipe(stringID)
}

func (f *File) StringDisconnectRecipe(stringID int) (ReferenceReport, Recipe, error) {
	if stringID < 0 {
		return ReferenceReport{}, Recipe{}, fmt.Errorf("string id must be non-negative")
	}
	refs, err := f.References(ReferenceOptions{Kind: "string", TargetID: &stringID})
	if err != nil {
		return ReferenceReport{}, Recipe{}, err
	}
	recipe := triggerRowDisconnectRecipe(refs.References)
	clearStringID := -1
	seenUnits := map[int]bool{}
	for _, ref := range refs.References {
		switch {
		case ref.SourceKind == "trigger" && ref.ChildIndex < 0 && ref.Field == "description_string_table_id":
			edit := upsertTriggerEdit(&recipe, ref.TriggerIndex)
			edit.DescriptionStringID = &clearStringID
		case ref.SourceKind == "trigger" && ref.ChildIndex < 0 && ref.Field == "short_description_string_table_id":
			edit := upsertTriggerEdit(&recipe, ref.TriggerIndex)
			edit.ShortDescriptionStringID = &clearStringID
		case ref.SourceKind == "unit" && ref.Field == "caption_string_id" && ref.UnitReferenceID >= 0:
			if seenUnits[ref.UnitReferenceID] {
				continue
			}
			seenUnits[ref.UnitReferenceID] = true
			sourceRef := ref.UnitReferenceID
			recipe.Units = append(recipe.Units, UnitRecipe{
				Op:              "edit_unit",
				ReferenceID:     &sourceRef,
				CaptionStringID: &clearStringID,
			})
		}
	}
	return refs, recipe, nil
}

func StringTextDisconnectRecipeFile(path string, text string) (int, ReferenceReport, Recipe, error) {
	file, err := Open(path)
	if err != nil {
		return 0, ReferenceReport{}, Recipe{}, err
	}
	return file.StringTextDisconnectRecipe(text)
}

func (f *File) StringTextDisconnectRecipe(text string) (int, ReferenceReport, Recipe, error) {
	stringID, err := f.findUniqueStringIDByText(text)
	if err != nil {
		return 0, ReferenceReport{}, Recipe{}, err
	}
	refs, recipe, err := f.StringDisconnectRecipe(stringID)
	if err != nil {
		return 0, ReferenceReport{}, Recipe{}, err
	}
	return stringID, refs, recipe, nil
}

func StringPrefixDisconnectRecipeFile(path string, prefix string) ([]int, ReferenceReport, Recipe, error) {
	file, err := Open(path)
	if err != nil {
		return nil, ReferenceReport{}, Recipe{}, err
	}
	return file.StringPrefixDisconnectRecipe(prefix)
}

func (f *File) StringPrefixDisconnectRecipe(prefix string) ([]int, ReferenceReport, Recipe, error) {
	ids, err := f.findStringIDsByTextPrefix(prefix)
	if err != nil {
		return nil, ReferenceReport{}, Recipe{}, err
	}
	refs, err := f.referencesForStringIDs(ids)
	if err != nil {
		return nil, ReferenceReport{}, Recipe{}, err
	}
	recipe := triggerRowDisconnectRecipe(refs.References)
	clearStringID := -1
	seenUnits := map[int]bool{}
	for _, ref := range refs.References {
		switch {
		case ref.SourceKind == "trigger" && ref.ChildIndex < 0 && ref.Field == "description_string_table_id":
			edit := upsertTriggerEdit(&recipe, ref.TriggerIndex)
			edit.DescriptionStringID = &clearStringID
		case ref.SourceKind == "trigger" && ref.ChildIndex < 0 && ref.Field == "short_description_string_table_id":
			edit := upsertTriggerEdit(&recipe, ref.TriggerIndex)
			edit.ShortDescriptionStringID = &clearStringID
		case ref.SourceKind == "unit" && ref.Field == "caption_string_id" && ref.UnitReferenceID >= 0:
			if seenUnits[ref.UnitReferenceID] {
				continue
			}
			seenUnits[ref.UnitReferenceID] = true
			sourceRef := ref.UnitReferenceID
			recipe.Units = append(recipe.Units, UnitRecipe{
				Op:              "edit_unit",
				ReferenceID:     &sourceRef,
				CaptionStringID: &clearStringID,
			})
		}
	}
	return ids, refs, recipe, nil
}

func StringContainsDisconnectRecipeFile(path string, text string) ([]int, ReferenceReport, Recipe, error) {
	file, err := Open(path)
	if err != nil {
		return nil, ReferenceReport{}, Recipe{}, err
	}
	return file.StringContainsDisconnectRecipe(text)
}

func (f *File) StringContainsDisconnectRecipe(text string) ([]int, ReferenceReport, Recipe, error) {
	ids, err := f.findStringIDsByTextContains(text)
	if err != nil {
		return nil, ReferenceReport{}, Recipe{}, err
	}
	refs, err := f.referencesForStringIDs(ids)
	if err != nil {
		return nil, ReferenceReport{}, Recipe{}, err
	}
	recipe := triggerRowDisconnectRecipe(refs.References)
	clearStringID := -1
	seenUnits := map[int]bool{}
	for _, ref := range refs.References {
		switch {
		case ref.SourceKind == "trigger" && ref.ChildIndex < 0 && ref.Field == "description_string_table_id":
			edit := upsertTriggerEdit(&recipe, ref.TriggerIndex)
			edit.DescriptionStringID = &clearStringID
		case ref.SourceKind == "trigger" && ref.ChildIndex < 0 && ref.Field == "short_description_string_table_id":
			edit := upsertTriggerEdit(&recipe, ref.TriggerIndex)
			edit.ShortDescriptionStringID = &clearStringID
		case ref.SourceKind == "unit" && ref.Field == "caption_string_id" && ref.UnitReferenceID >= 0:
			if seenUnits[ref.UnitReferenceID] {
				continue
			}
			seenUnits[ref.UnitReferenceID] = true
			sourceRef := ref.UnitReferenceID
			recipe.Units = append(recipe.Units, UnitRecipe{
				Op:              "edit_unit",
				ReferenceID:     &sourceRef,
				CaptionStringID: &clearStringID,
			})
		}
	}
	return ids, refs, recipe, nil
}

func triggerRowDisconnectRecipe(refs []ScenarioReference) Recipe {
	type triggerEdits struct {
		effects    []int
		conditions []int
	}
	byTrigger := map[int]*triggerEdits{}
	for _, ref := range refs {
		if ref.SourceKind != "trigger" || ref.TriggerIndex < 0 || ref.ChildIndex < 0 {
			continue
		}
		item := byTrigger[ref.TriggerIndex]
		if item == nil {
			item = &triggerEdits{}
			byTrigger[ref.TriggerIndex] = item
		}
		switch ref.ChildKind {
		case "effect":
			item.effects = append(item.effects, ref.ChildIndex)
		case "condition":
			item.conditions = append(item.conditions, ref.ChildIndex)
		}
	}
	if len(byTrigger) == 0 {
		return Recipe{}
	}
	triggerIndexes := make([]int, 0, len(byTrigger))
	for triggerIndex := range byTrigger {
		triggerIndexes = append(triggerIndexes, triggerIndex)
	}
	sort.Ints(triggerIndexes)
	recipe := Recipe{}
	for _, triggerIndex := range triggerIndexes {
		item := byTrigger[triggerIndex]
		targetIndex := triggerIndex
		recipe.Triggers = append(recipe.Triggers, TriggerRecipe{
			Op:               "edit_trigger",
			TargetIndex:      &targetIndex,
			RemoveEffects:    sortedUniqueInts(item.effects),
			RemoveConditions: sortedUniqueInts(item.conditions),
		})
	}
	return recipe
}

func upsertTriggerEdit(recipe *Recipe, triggerIndex int) *TriggerRecipe {
	for i := range recipe.Triggers {
		if recipe.Triggers[i].TargetIndex != nil && *recipe.Triggers[i].TargetIndex == triggerIndex {
			return &recipe.Triggers[i]
		}
	}
	targetIndex := triggerIndex
	recipe.Triggers = append(recipe.Triggers, TriggerRecipe{
		Op:          "edit_trigger",
		TargetIndex: &targetIndex,
	})
	return &recipe.Triggers[len(recipe.Triggers)-1]
}

func sortedUniqueInts(values []int) []int {
	if len(values) == 0 {
		return nil
	}
	out := append([]int(nil), values...)
	sort.Ints(out)
	write := 0
	for _, value := range out {
		if write > 0 && out[write-1] == value {
			continue
		}
		out[write] = value
		write++
	}
	return out[:write]
}
