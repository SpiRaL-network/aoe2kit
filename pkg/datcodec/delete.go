package datcodec

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"aoe2kit/pkg/aoe2"
	"aoe2kit/pkg/datfile"
)

type DeletePlanRequest struct {
	Section      string `json:"section"`
	ID           int    `json:"id"`
	EffectID     *int   `json:"effect_id,omitempty"`
	CommandIndex *int   `json:"command_index,omitempty"`
	SoundID      *int   `json:"sound_id,omitempty"`
	ItemIndex    *int   `json:"item_index,omitempty"`
	GraphicID    *int   `json:"graphic_id,omitempty"`
	UnitHeaderID *int   `json:"unit_header_id,omitempty"`
	UnitID       *int   `json:"unit_id,omitempty"`
	RowIndex     *int   `json:"row_index,omitempty"`
	CivID        *int   `json:"civ_id,omitempty"`
	AllCivs      bool   `json:"all_civs,omitempty"`
}

type DeletePlanReport struct {
	Path           string                     `json:"path,omitempty"`
	Version        string                     `json:"version"`
	Request        DeletePlanRequest          `json:"request"`
	Supported      bool                       `json:"supported"`
	Mutates        bool                       `json:"mutates"`
	Strategy       string                     `json:"strategy"`
	RecipeHint     json.RawMessage            `json:"recipe_hint,omitempty"`
	CleanupCommand string                     `json:"cleanup_command,omitempty"`
	CleanupRecipe  json.RawMessage            `json:"cleanup_recipe,omitempty"`
	References     []DeleteReference          `json:"references,omitempty"`
	TechMigration  *TechDeleteMigrationReport `json:"tech_migration,omitempty"`
	Warnings       []string                   `json:"warnings,omitempty"`
	Reason         string                     `json:"reason,omitempty"`
	Verification   aoe2.VerificationClaim     `json:"verification"`
}

type DeleteReference struct {
	Section    string `json:"section"`
	ID         int    `json:"id"`
	Field      string `json:"field"`
	Confidence string `json:"confidence,omitempty"`
}

type ReferenceReport struct {
	Version      string                 `json:"version"`
	Request      DeletePlanRequest      `json:"request"`
	Filters      ReferenceFilters       `json:"filters,omitempty"`
	References   []ClassifiedReference  `json:"references,omitempty"`
	Summary      ReferenceSummary       `json:"summary"`
	Verification aoe2.VerificationClaim `json:"verification"`
}

type ReferenceFilters struct {
	Class         string `json:"class,omitempty"`
	Confidence    string `json:"confidence,omitempty"`
	SourceSection string `json:"source_section,omitempty"`
	Limit         int    `json:"limit,omitempty"`
}

type ClassifiedReference struct {
	DeleteReference
	Class  string `json:"class"`
	Reason string `json:"reason,omitempty"`
}

type ReferenceSummary struct {
	Total            int `json:"total"`
	Returned         int `json:"returned"`
	RewriteSupported int `json:"rewrite_supported"`
	KnownReadonly    int `json:"known_readonly"`
	PossibleOperand  int `json:"possible_operand"`
	CandidateOperand int `json:"candidate_operand"`
	Unsupported      int `json:"unsupported"`
}

type TechDeleteMigrationReport struct {
	DeleteID              int  `json:"delete_id"`
	TailID                int  `json:"tail_id"`
	CheckedTechIDs        int  `json:"checked_tech_ids"`
	Ready                 bool `json:"ready"`
	CoveredReferences     int  `json:"covered_references"`
	UnsupportedReferences int  `json:"unsupported_references"`
	IDsWithUnsupported    int  `json:"ids_with_unsupported"`
	IDsWithReferences     int  `json:"ids_with_references"`
}

func DeletePlanFile(path string, request DeletePlanRequest) (DeletePlanReport, error) {
	compressed, err := os.ReadFile(path)
	if err != nil {
		return DeletePlanReport{}, err
	}
	payload, err := datfile.Inflate(compressed)
	if err != nil {
		return DeletePlanReport{}, err
	}
	idx, err := datfile.Parse(payload)
	if err != nil {
		return DeletePlanReport{}, fmt.Errorf("parse input: %w", err)
	}
	report := DeletePlan(idx, request)
	report.Path = path
	refreshDeletePlanCleanupCommand(&report)
	return report, nil
}

func ReferencesFile(path string, request DeletePlanRequest) (ReferenceReport, error) {
	compressed, err := os.ReadFile(path)
	if err != nil {
		return ReferenceReport{}, err
	}
	payload, err := datfile.Inflate(compressed)
	if err != nil {
		return ReferenceReport{}, err
	}
	idx, err := datfile.Parse(payload)
	if err != nil {
		return ReferenceReport{}, fmt.Errorf("parse input: %w", err)
	}
	return References(idx, request), nil
}

func References(idx *datfile.Index, request DeletePlanRequest) ReferenceReport {
	report := ReferenceReport{
		Version:      Version,
		Request:      request,
		Verification: aoe2.StructureVerification(true),
	}
	report.Verification.Note = "refs is a structural DAT reference explorer; it does not prove in-engine behavior."
	refs := referencesForRequest(idx, request)
	report.References = make([]ClassifiedReference, 0, len(refs))
	for _, ref := range refs {
		classified := classifyReference(request.Section, ref)
		report.References = append(report.References, classified)
	}
	report.Summary = summarizeClassifiedReferences(report.References)
	return report
}

func FilterReferenceReport(report ReferenceReport, filters ReferenceFilters) ReferenceReport {
	report.Filters = filters
	if filters.Class == "" && filters.Confidence == "" && filters.SourceSection == "" && filters.Limit <= 0 {
		return report
	}
	matches := make([]ClassifiedReference, 0, len(report.References))
	for _, ref := range report.References {
		if filters.Class != "" {
			if filters.Class == "candidate_operand" {
				if ref.Confidence != "candidate_effect_command_operand" {
					continue
				}
			} else if ref.Class != filters.Class {
				continue
			}
		}
		if filters.Confidence != "" && ref.Confidence != filters.Confidence {
			continue
		}
		if filters.SourceSection != "" && ref.Section != filters.SourceSection {
			continue
		}
		matches = append(matches, ref)
	}
	report.Summary = summarizeClassifiedReferences(matches)
	if filters.Limit > 0 && len(matches) > filters.Limit {
		report.References = matches[:filters.Limit]
		report.Summary.Returned = filters.Limit
		return report
	}
	report.References = matches
	return report
}

func summarizeClassifiedReferences(refs []ClassifiedReference) ReferenceSummary {
	var summary ReferenceSummary
	summary.Returned = len(refs)
	for _, classified := range refs {
		summary.Total++
		switch classified.Class {
		case "rewrite_supported":
			summary.RewriteSupported++
		case "known_readonly":
			summary.KnownReadonly++
		case "possible_operand":
			summary.PossibleOperand++
			if classified.Confidence == "candidate_effect_command_operand" {
				summary.CandidateOperand++
			}
		default:
			summary.Unsupported++
		}
	}
	return summary
}

func DeletePlan(idx *datfile.Index, request DeletePlanRequest) DeletePlanReport {
	report := DeletePlanReport{
		Version:      Version,
		Request:      request,
		Verification: aoe2.StructureVerification(true),
	}
	report.Verification.Note = "delete-plan is a structural DAT codec advisory; it does not prove in-engine behavior."
	switch request.Section {
	case "ability", "abilities":
		return abilityDeletePlan(idx, report, request.ID)
	case "effect_command", "effect-command", "command":
		return effectCommandDeletePlan(idx, report, request)
	case "effect", "effects":
		return effectDeletePlan(idx, report, request.ID)
	case "tech", "research", "researches":
		return techDeletePlan(idx, report, request.ID)
	case "unit", "units":
		return unitDeletePlan(idx, report, request)
	case "unit_header_task", "unit-header-task":
		return unitHeaderTaskDeletePlan(idx, report, request)
	case "unit_damage_graphic", "unit-damage-graphic", "unit_damage", "unit-damage":
		return unitChildRowDeletePlan(idx, report, request, "damage_graphic")
	case "unit_attack", "unit-attack":
		return unitChildRowDeletePlan(idx, report, request, "attack")
	case "unit_armour", "unit-armour", "unit_armor", "unit-armor":
		return unitChildRowDeletePlan(idx, report, request, "armour")
	case "unit_train_location", "unit-train-location":
		return unitChildRowDeletePlan(idx, report, request, "train_location")
	case "unit_drop_site", "unit-drop-site":
		return unitChildRowDeletePlan(idx, report, request, "drop_site")
	case "unit_task", "unit-task":
		return unitChildRowDeletePlan(idx, report, request, "task")
	case "graphic", "graphics":
		return graphicDeletePlan(idx, report, request.ID)
	case "graphic_delta", "graphic-delta":
		return graphicChildRowDeletePlan(idx, report, request, "delta")
	case "graphic_angle_sound", "graphic-angle-sound":
		return graphicChildRowDeletePlan(idx, report, request, "angle_sound")
	case "civ", "civilization", "civilizations":
		report.Strategy = "unsupported"
		report.References = civIDReferences(idx, request.ID)
		report.Reason = "civilization rows are stable ID-addressed and referenced by scenario/replay/player metadata; no verified semantic tombstone is implemented."
		if len(report.References) > 0 {
			report.Warnings = append(report.Warnings, "References list DAT rows that currently point at this civ ID; patch those rows before designing any semantic civ tombstone.")
		}
	case "terrain_restriction", "terrain_restrictions":
		report.Strategy = "unsupported"
		report.Reason = "terrain restriction rows are indexed by terrain passability systems; no verified semantic tombstone is implemented."
	case "tech_tree_building_connection", "tech_tree_building_connections", "building_connection", "building_connections":
		return techTreeConnectionDeletePlan(idx, report, "building", request.ID)
	case "tech_tree_unit_connection", "tech_tree_unit_connections", "unit_connection", "unit_connections":
		return techTreeConnectionDeletePlan(idx, report, "unit", request.ID)
	case "tech_tree_research_connection", "tech_tree_research_connections", "research_connection", "research_connections":
		return techTreeConnectionDeletePlan(idx, report, "research", request.ID)
	case "terrain", "terrains":
		report.Strategy = "unsupported"
		report.References = terrainIDReferences(idx, request.ID)
		report.Reason = "terrain rows are fixed engine terrain IDs; AoE2Kit can frame terrain records but does not support deleting terrain IDs."
		if len(report.References) > 0 {
			report.Warnings = append(report.Warnings, "References list DAT task rows that currently point at this terrain ID; patch those rows instead of physically deleting the terrain slot.")
		}
	case "sound", "sounds":
		return soundDeletePlan(idx, report, request.ID)
	case "sound_item", "sound-item":
		return soundItemDeletePlan(idx, report, request)
	case "player_colour", "player_colours", "color", "colors":
		return playerColourDeletePlan(idx, report, request.ID)
	case "random_map", "random_maps":
		report.Strategy = "unsupported"
		report.Reason = "random-map rows are framed structurally, but the current DE fixture has no records to validate write/delete behavior."
	default:
		report.Strategy = "unsupported"
		report.Reason = fmt.Sprintf("unknown DAT section %q", request.Section)
	}
	return report
}

func setDeletePlanCleanup(report *DeletePlanReport, section string, id int, recipe Recipe) {
	cleanupRecipe := mustRecipeHint(recipe)
	report.RecipeHint = cleanupRecipe
	report.CleanupRecipe = cleanupRecipe
	report.CleanupCommand = datDisconnectCommand(report.Path, section, id)
}

func setDeletePlanOptionalCleanup(report *DeletePlanReport, section string, id int, recipe Recipe) {
	report.CleanupRecipe = mustRecipeHint(recipe)
	report.CleanupCommand = datDisconnectCommand(report.Path, section, id)
}

func refreshDeletePlanCleanupCommand(report *DeletePlanReport) {
	if len(report.CleanupRecipe) == 0 {
		return
	}
	switch {
	case strings.Contains(string(report.CleanupRecipe), `"disconnect_techs"`):
		report.CleanupCommand = datDisconnectCommand(report.Path, "tech", report.Request.ID)
	case strings.Contains(string(report.CleanupRecipe), `"disconnect_units"`):
		report.CleanupCommand = datDisconnectCommand(report.Path, "unit", report.Request.ID)
	}
}

func datDisconnectCommand(path, section string, id int) string {
	inPath := strings.TrimSpace(path)
	if inPath == "" {
		inPath = "<in.dat>"
	}
	return fmt.Sprintf("kit dat disconnect %s <out.dat> %s %d", inPath, section, id)
}

func techTreeConnectionDeletePlan(idx *datfile.Index, report DeletePlanReport, family string, id int) DeletePlanReport {
	report.Strategy = "physical_connection_row_delete"
	var count int
	recipe := TechTreePatchRecipe{}
	switch family {
	case "building":
		count = len(idx.TechTree.BuildingConnections)
		recipe.DeleteBuildingConnections = []int{id}
	case "unit":
		count = len(idx.TechTree.UnitConnections)
		recipe.DeleteUnitConnections = []int{id}
	case "research":
		count = len(idx.TechTree.ResearchConnections)
		recipe.DeleteResearchConnections = []int{id}
	default:
		report.Strategy = "unsupported"
		report.Reason = fmt.Sprintf("unknown tech-tree connection family %q", family)
		return report
	}
	if id < 0 || id >= count {
		report.Reason = fmt.Sprintf("%s connection index=%d outside table length %d", family, id, count)
		return report
	}
	report.Supported = true
	report.Mutates = true
	report.RecipeHint = mustRecipeHint(Recipe{TechTree: &recipe})
	report.Warnings = append(report.Warnings, fmt.Sprintf("Tech-tree %s connection delete removes one counted relationship row by current table index; later rows shift down by one, and this does not delete any Unit, Tech, Civ, Graphic, or Sound records.", family))
	return report
}

func referencesForRequest(idx *datfile.Index, request DeletePlanRequest) []DeleteReference {
	switch request.Section {
	case "ability", "abilities":
		return abilityIDReferences(idx, request.ID)
	case "effect", "effects":
		var refs []DeleteReference
		for _, tech := range idx.Techs {
			if int(tech.EffectID) == request.ID {
				refs = append(refs, DeleteReference{Section: "tech", ID: tech.Index, Field: "effect_id"})
			}
		}
		sortReferences(refs)
		return refs
	case "tech", "research", "researches":
		return techIDReferences(idx, request.ID)
	case "unit", "units":
		return unitIDReferences(idx, request.ID)
	case "graphic", "graphics":
		return graphicIDReferences(idx, request.ID)
	case "civ", "civilization", "civilizations":
		return civIDReferences(idx, request.ID)
	case "terrain", "terrains":
		return terrainIDReferences(idx, request.ID)
	case "sound", "sounds":
		return soundIDReferences(idx, request.ID)
	case "player_colour", "player_colours", "color", "colors":
		return playerColourIDReferences(idx, request.ID)
	default:
		return nil
	}
}

func classifyReference(section string, ref DeleteReference) ClassifiedReference {
	classified := ClassifiedReference{DeleteReference: ref}
	if ref.Confidence == "candidate_effect_command_operand" {
		classified.Class = "possible_operand"
		classified.Reason = "command-shape candidate field; stronger than arbitrary numeric coincidence, but not promoted to rewrite support"
		return classified
	}
	if ref.Confidence == "possible_effect_command_operand" {
		classified.Class = "possible_operand"
		classified.Reason = "numeric effect-command operand sighting; command semantics not promoted"
		return classified
	}
	switch section {
	case "tech", "research", "researches":
		if techReferenceRewriteCovers(ref) {
			classified.Class = "rewrite_supported"
			classified.Reason = "covered by reference_rewrites target=tech"
			return classified
		}
	case "unit", "units":
		if unitReferenceRewriteCovers(ref) {
			classified.Class = "rewrite_supported"
			classified.Reason = "covered by reference_rewrites target=unit or existing unit patch surfaces"
			return classified
		}
	case "effect", "effects":
		if ref.Section == "tech" && ref.Field == "effect_id" {
			classified.Class = "rewrite_supported"
			classified.Reason = "covered by effect physical-delete effect_id rewrites"
			return classified
		}
	case "sound", "sounds":
		if strings.HasPrefix(ref.Section, "graphic") {
			classified.Class = "known_readonly"
			classified.Reason = "sound references are known; sound delete uses semantic mute, not ID rewrite"
			return classified
		}
	}
	if strings.HasPrefix(ref.Section, "tech_tree_") {
		classified.Class = "known_readonly"
		classified.Reason = "known tech-tree reference, but no verified rewrite surface for this requested section"
		return classified
	}
	classified.Class = "unsupported"
	classified.Reason = "known reference, but no verified rewrite strategy for this requested section"
	return classified
}

func abilityDeletePlan(idx *datfile.Index, report DeletePlanReport, techID int) DeletePlanReport {
	report.Strategy = "paired_tech_then_effect_delete"
	if techID < 0 || techID >= len(idx.Techs) {
		report.Reason = fmt.Sprintf("ability tech id=%d outside tech table length %d", techID, len(idx.Techs))
		return report
	}
	tech := idx.Techs[techID]
	if tech.EffectID < 0 || int(tech.EffectID) >= len(idx.Effects) {
		report.Strategy = "unsupported"
		report.Reason = fmt.Sprintf("ability tech id=%d does not reference a valid effect_id: %d", techID, tech.EffectID)
		return report
	}
	effectID := int(tech.EffectID)
	report.References = abilityIDReferences(idx, techID)
	techPlan := techDeletePlan(idx, DeletePlanReport{
		Version:      report.Version,
		Request:      DeletePlanRequest{Section: "tech", ID: techID},
		Verification: report.Verification,
	}, techID)
	report.TechMigration = techPlan.TechMigration
	if !techPlan.Supported {
		if canSemanticallyClearAbilityEffect(idx, techID, effectID) {
			report.Strategy = "semantic_clear_ability_effect"
			report.Supported = true
			report.Mutates = true
			report.RecipeHint = mustRecipeHint(Recipe{DisableAbilities: []int{techID}})
			if techPlan.Reason != "" {
				report.Reason = fmt.Sprintf("ability tech id=%d cannot be physically deleted: %s", techID, techPlan.Reason)
			} else {
				report.Reason = fmt.Sprintf("ability tech id=%d cannot be physically deleted by strategy %s", techID, techPlan.Strategy)
			}
			report.Warnings = append(report.Warnings, "Semantic ability delete preserves the tech row and effect ID, then clears the paired effect commands. The ability may still appear in menus, prerequisites, and tech-tree references, but activating it becomes structurally inert.")
			return report
		}
		report.Strategy = "unsupported_ability_tech_delete"
		if techPlan.Reason != "" {
			report.Reason = fmt.Sprintf("ability tech id=%d cannot be deleted: %s", techID, techPlan.Reason)
		} else {
			report.Reason = fmt.Sprintf("ability tech id=%d cannot be deleted by strategy %s", techID, techPlan.Strategy)
		}
		return report
	}
	for _, ref := range report.References {
		if ref.Section == "tech" && ref.ID == techID && ref.Field == "effect_id" {
			continue
		}
		report.Strategy = "unsupported_referenced_ability_delete"
		report.Reason = "ability is still referenced outside its own tech->effect edge; inspect references before deleting the paired tech/effect."
		return report
	}
	report.Supported = true
	report.Mutates = true
	report.RecipeHint = mustRecipeHint(Recipe{DeleteTechs: []int{techID}, DeleteEffects: []int{effectID}})
	report.Warnings = append(report.Warnings, "Ability delete is a paired structural delete: the tech row is removed first using the current tech delete strategy, then the now-unreferenced effect is removed. Effects above the deleted effect ID are renumbered and tech.effect_id values are rewritten by the codec.")
	return report
}

func canSemanticallyClearAbilityEffect(idx *datfile.Index, techID int, effectID int) bool {
	for _, ref := range effectIDReferences(idx, effectID) {
		if ref.Section == "tech" && ref.ID == techID && ref.Field == "effect_id" {
			continue
		}
		return false
	}
	return true
}

func abilityIDReferences(idx *datfile.Index, techID int) []DeleteReference {
	if techID < 0 || techID >= len(idx.Techs) {
		return nil
	}
	refs := techIDReferences(idx, techID)
	tech := idx.Techs[techID]
	if tech.EffectID >= 0 && int(tech.EffectID) < len(idx.Effects) {
		for _, ref := range effectIDReferences(idx, int(tech.EffectID)) {
			refs = append(refs, ref)
		}
	}
	sortReferences(refs)
	return refs
}

func graphicDeletePlan(idx *datfile.Index, report DeletePlanReport, id int) DeletePlanReport {
	report.References = graphicIDReferences(idx, id)
	if id < 0 || id >= idx.GraphicsSize {
		report.Strategy = "unsupported_out_of_range"
		report.Reason = fmt.Sprintf("graphic %d is outside graphics table size %d", id, idx.GraphicsSize)
		return report
	}
	if _, ok := idx.Graphic(id); !ok {
		report.Strategy = "already_absent"
		report.Reason = "graphic slot is already absent; no mutation is needed."
		return report
	}
	if len(report.References) > 0 {
		report.Strategy = "unsupported_referenced_stable_slot"
		report.Reason = "graphic slot is referenced by decoded DAT rows; stable-ID delete would leave dangling art references."
		report.Warnings = append(report.Warnings, "Patch or delete referencing rows first, then re-run delete-plan before deleting the graphic slot.")
		return report
	}
	report.Supported = true
	report.Mutates = true
	report.Strategy = "stable_id_absent_slot"
	report.RecipeHint = mustRecipeHint(Recipe{DeleteGraphics: []int{id}})
	report.Warnings = append(report.Warnings, "Stable-ID graphic delete is structure-verified only: it zeroes the pointer slot and removes the graphic record while preserving graphics_size and downstream graphic IDs.")
	return report
}

func soundDeletePlan(idx *datfile.Index, report DeletePlanReport, id int) DeletePlanReport {
	report.Strategy = "semantic_mute"
	if id < 0 || id >= len(idx.Sounds) {
		report.Reason = fmt.Sprintf("sound id=%d outside sound table length %d", id, len(idx.Sounds))
		return report
	}
	report.References = soundIDReferences(idx, id)
	report.Supported = true
	report.Mutates = true
	report.RecipeHint = mustRecipeHint(Recipe{DeleteSounds: []int{id}})
	report.Warnings = append(report.Warnings, "Semantic sound delete preserves the sound ID and sets total/item probabilities to 0; in-engine silence still needs live verification for each use case.")
	return report
}

func soundItemDeletePlan(idx *datfile.Index, report DeletePlanReport, request DeletePlanRequest) DeletePlanReport {
	report.Strategy = "physical_sound_item_row_delete"
	if request.SoundID == nil || request.ItemIndex == nil {
		report.Reason = "sound-item delete needs target formatted as <sound_id>:<item_index>"
		return report
	}
	soundID := *request.SoundID
	itemIndex := *request.ItemIndex
	report.Request.ID = soundID
	if soundID < 0 || soundID >= len(idx.Sounds) {
		report.Reason = fmt.Sprintf("sound id=%d outside sound table length %d", soundID, len(idx.Sounds))
		return report
	}
	sound := idx.Sounds[soundID]
	if itemIndex < 0 || itemIndex >= len(sound.Items) {
		report.Reason = fmt.Sprintf("sound id=%d item_index=%d outside item list length %d", soundID, itemIndex, len(sound.Items))
		return report
	}
	report.References = soundIDReferences(idx, soundID)
	report.Supported = true
	report.Mutates = true
	report.RecipeHint = mustRecipeHint(Recipe{Sounds: []SoundPatchRecipe{{ID: soundID, RemoveItems: []int{itemIndex}}}})
	report.Warnings = append(report.Warnings, "Sound-item row delete removes one counted item from one sound record. Later item indices in that sound shift down by one; references to the sound ID itself are preserved.")
	return report
}

func playerColourDeletePlan(idx *datfile.Index, report DeletePlanReport, id int) DeletePlanReport {
	report.Strategy = "physical_tail_delete"
	if id < 0 || id >= len(idx.PlayerColours) {
		report.Reason = fmt.Sprintf("player colour id=%d outside player colour table length %d", id, len(idx.PlayerColours))
		return report
	}
	report.References = playerColourIDReferences(idx, id)
	if id != len(idx.PlayerColours)-1 {
		report.Strategy = "unsupported_non_tail_palette_slot_delete"
		report.Reason = fmt.Sprintf("player colour id=%d is not the tail id=%d; physical removal would renumber stable palette slots", id, len(idx.PlayerColours)-1)
		if len(report.References) > 0 {
			report.Warnings = append(report.Warnings, "References list graphics that currently point at this player-colour slot; patch those graphics instead of deleting the palette row.")
		}
		return report
	}
	if len(report.References) > 0 {
		report.Strategy = "unsupported_referenced_tail_palette_slot_delete"
		report.Reason = "tail player-colour slot is still referenced by graphics; patch those graphics first."
		report.Warnings = append(report.Warnings, "References list graphics that currently point at this player-colour slot; patch those graphics before retrying the tail delete.")
		return report
	}
	report.Supported = true
	report.Mutates = true
	report.RecipeHint = mustRecipeHint(Recipe{DeletePlayerColours: []int{id}})
	report.Warnings = append(report.Warnings, "Tail-only player-colour delete preserves all earlier palette IDs. Non-tail palette compaction remains unsupported; patch colour fields instead.")
	return report
}

func effectDeletePlan(idx *datfile.Index, report DeletePlanReport, id int) DeletePlanReport {
	report.Strategy = "physical_delete_with_effect_id_renumber"
	if id < 0 || id >= len(idx.Effects) {
		report.Reason = fmt.Sprintf("effect id=%d outside effects table length %d", id, len(idx.Effects))
		return report
	}
	report.References = effectIDReferences(idx, id)
	if len(report.References) > 0 {
		report.Strategy = "semantic_clear_commands"
		report.Supported = true
		report.Mutates = true
		report.RecipeHint = mustRecipeHint(Recipe{DisableEffects: []int{id}})
		report.Warnings = append(report.Warnings, "Effect is directly referenced, so physical delete would break stable effect IDs. Semantic effect delete preserves the effect row and clears every command; all techs that reference this effect become inert unless retargeted first.")
		return report
	}
	report.Supported = true
	report.Mutates = true
	report.RecipeHint = mustRecipeHint(Recipe{DeleteEffects: []int{id}})
	report.Warnings = append(report.Warnings, "Effects above the deleted ID are renumbered; tech.effect_id values above the deleted ID are rewritten by the codec.")
	return report
}

func effectCommandDeletePlan(idx *datfile.Index, report DeletePlanReport, request DeletePlanRequest) DeletePlanReport {
	report.Strategy = "physical_effect_command_row_delete"
	if request.EffectID == nil || request.CommandIndex == nil {
		report.Reason = "effect-command delete needs target formatted as <effect_id>:<command_index>"
		return report
	}
	effectID := *request.EffectID
	commandIndex := *request.CommandIndex
	report.Request.ID = effectID
	if effectID < 0 || effectID >= len(idx.Effects) {
		report.Reason = fmt.Sprintf("effect id=%d outside effects table length %d", effectID, len(idx.Effects))
		return report
	}
	effect := idx.Effects[effectID]
	if commandIndex < 0 || commandIndex >= len(effect.Commands) {
		report.Reason = fmt.Sprintf("effect id=%d command_index=%d outside command list length %d", effectID, commandIndex, len(effect.Commands))
		return report
	}
	report.Supported = true
	report.Mutates = true
	report.RecipeHint = mustRecipeHint(Recipe{Effects: []EffectPatchRecipe{{ID: effectID, RemoveCommands: []int{commandIndex}}}})
	report.Warnings = append(report.Warnings, "Effect-command row delete removes one counted command from one effect record. Later command indices in that effect shift down by one; references to the effect ID itself are preserved.")
	return report
}

func effectIDReferences(idx *datfile.Index, id int) []DeleteReference {
	var refs []DeleteReference
	for _, tech := range idx.Techs {
		if int(tech.EffectID) == id {
			refs = append(refs, DeleteReference{Section: "tech", ID: tech.Index, Field: "effect_id"})
		}
	}
	sortReferences(refs)
	return refs
}

func techDeletePlan(idx *datfile.Index, report DeletePlanReport, id int) DeletePlanReport {
	report.Strategy = "physical_tail_delete"
	if id < 0 || id >= len(idx.Techs) {
		report.Reason = fmt.Sprintf("tech id=%d outside tech table length %d", id, len(idx.Techs))
		return report
	}
	report.References = techIDReferences(idx, id)
	if hasTypedEffectCommandRefs(report.References) {
		report.Warnings = append(report.Warnings, "Typed effect-command references are decoded from the upstream genieutils effect-command layout plus known command types; retarget or remove those effects before deleting the tech.")
	}
	if hasPossibleEffectCommandRefs(report.References) {
		report.Warnings = append(report.Warnings, "Candidate/possible effect-command operand references are not rewrite-supported typed command semantics; inspect the listed effects before retargeting or deleting the tech.")
	}
	if id != len(idx.Techs)-1 {
		report.Strategy = "physical_non_tail_tail_swap"
		report.TechMigration = techDeleteMigrationReport(idx, id)
		if len(report.References) > 0 {
			covered, unsupported := techReferenceRewriteCoverage(report.References)
			report.Strategy = "unsupported_referenced_non_tail_delete"
			report.Reason = fmt.Sprintf("tech id=%d is still referenced by %d decoded refs; remove or retarget those references before physical tail-swap delete.", id, len(report.References))
			if covered > 0 {
				setDeletePlanCleanup(&report, "tech", id, Recipe{DisconnectTechs: []int{id}})
				report.Warnings = append(report.Warnings, fmt.Sprintf("Semantic disconnect recipe can remove %d currently decoded tech references; %d references remain outside the verified rewrite surface. Re-run delete-plan after applying it before attempting physical delete.", covered, unsupported))
			}
			return report
		}
		if report.TechMigration != nil && !report.TechMigration.Ready {
			report.Strategy = "unsupported_tail_migration_refs"
			report.Reason = fmt.Sprintf("tech id=%d is unreferenced, but tail tech id=%d has %d references outside the verified rewrite surface; moving the tail into the hole would orphan those refs.", id, report.TechMigration.TailID, report.TechMigration.UnsupportedReferences)
			return report
		}
		report.Supported = true
		report.Mutates = true
		report.RecipeHint = mustRecipeHint(Recipe{DeleteTechs: []int{id}})
		report.Warnings = append(report.Warnings, "Non-tail tech delete uses tail-swap, not renumber-all: the current tail tech row is moved into the deleted slot, verified tail-tech references are rewritten to that slot, then the tail row is removed.")
		return report
	}
	if len(report.References) > 0 {
		covered, unsupported := techReferenceRewriteCoverage(report.References)
		if covered > 0 {
			setDeletePlanCleanup(&report, "tech", id, Recipe{DisconnectTechs: []int{id}})
			report.Warnings = append(report.Warnings, fmt.Sprintf("Semantic disconnect recipe can remove %d currently decoded tech references; %d references remain outside the verified rewrite surface. Re-run delete-plan after applying it before attempting physical delete.", covered, unsupported))
		}
		report.Reason = "tail tech is still referenced; delete or retarget those references before removing the row."
		return report
	}
	report.Supported = true
	report.Mutates = true
	report.RecipeHint = mustRecipeHint(Recipe{DeleteTechs: []int{id}})
	report.Warnings = append(report.Warnings, "Tail tech deletion preserves all earlier tech IDs; non-tail deletion is intentionally blocked.")
	return report
}

func unitDeletePlan(idx *datfile.Index, report DeletePlanReport, request DeletePlanRequest) DeletePlanReport {
	report.Strategy = "semantic_disable"
	if request.ID < 0 {
		report.Reason = fmt.Sprintf("negative unit id=%d", request.ID)
		return report
	}
	if request.AllCivs && request.CivID != nil {
		report.Reason = "set either all_civs or civ_id, not both"
		return report
	}
	if request.AllCivs {
		present := 0
		for _, civ := range idx.Civs {
			if request.ID < len(civ.Units) && civ.Units[request.ID].Present {
				present++
			}
		}
		if present == 0 {
			report.Reason = fmt.Sprintf("unit id=%d is not present in any civ table", request.ID)
			return report
		}
		report.References = unitIDReferences(idx, request.ID)
		report.Supported = true
		report.Mutates = true
		report.RecipeHint = mustRecipeHint(Recipe{DisconnectUnits: []int{request.ID}, DeleteUnits: []UnitDeleteRecipe{{UnitID: request.ID, AllCivs: true}}})
		setUnitDeletePlanCleanup(&report, request.ID)
		report.Warnings = append(report.Warnings, "Semantic unit delete sets enabled=0, preserves stable unit IDs, and the recipe hint also disconnects known DAT references through verified rewrite surfaces; scenario triggers that create the exact unit ID may still need scenario-side edits.")
		if hasTypedEffectCommandRefs(report.References) {
			report.Warnings = append(report.Warnings, "Typed effect-command references are decoded from the upstream genieutils effect-command layout plus known command types; inspect the listed effects before assuming enabled=0 covers every way this unit can appear.")
		}
		if hasPossibleEffectCommandRefs(report.References) {
			report.Warnings = append(report.Warnings, "Candidate/possible effect-command operand references are not rewrite-supported typed command semantics; inspect the listed effects before assuming enabled=0 covers every way this unit can appear.")
		}
		return report
	}
	if request.CivID == nil {
		report.Reason = "unit delete needs civ_id or all_civs=true"
		return report
	}
	if *request.CivID < 0 || *request.CivID >= len(idx.Civs) {
		report.Reason = fmt.Sprintf("civ_id=%d outside civ table length %d", *request.CivID, len(idx.Civs))
		return report
	}
	civ := idx.Civs[*request.CivID]
	if request.ID >= len(civ.Units) || !civ.Units[request.ID].Present {
		report.Reason = fmt.Sprintf("unit id=%d is not present for civ %d", request.ID, *request.CivID)
		return report
	}
	report.References = unitIDReferences(idx, request.ID)
	report.Supported = true
	report.Mutates = true
	report.RecipeHint = mustRecipeHint(Recipe{DisconnectUnits: []int{request.ID}, DeleteUnits: []UnitDeleteRecipe{{CivID: request.CivID, UnitID: request.ID}}})
	setUnitDeletePlanCleanup(&report, request.ID)
	report.Warnings = append(report.Warnings, "Semantic unit delete sets enabled=0, preserves stable unit IDs, and the recipe hint also disconnects known DAT references through verified rewrite surfaces; scenario triggers that create the exact unit ID may still need scenario-side edits.")
	if hasTypedEffectCommandRefs(report.References) {
		report.Warnings = append(report.Warnings, "Typed effect-command references are decoded from the upstream genieutils effect-command layout plus known command types; inspect the listed effects before assuming enabled=0 covers every way this unit can appear.")
	}
	if hasPossibleEffectCommandRefs(report.References) {
		report.Warnings = append(report.Warnings, "Candidate/possible effect-command operand references are not rewrite-supported typed command semantics; inspect the listed effects before assuming enabled=0 covers every way this unit can appear.")
	}
	return report
}

func unitChildRowDeletePlan(idx *datfile.Index, report DeletePlanReport, request DeletePlanRequest, kind string) DeletePlanReport {
	report.Strategy = "physical_unit_child_row_delete"
	if request.CivID == nil || request.UnitID == nil || request.RowIndex == nil {
		report.Reason = fmt.Sprintf("unit-%s delete needs target formatted as <civ_id>:<unit_id>:<row_index>", strings.ReplaceAll(kind, "_", "-"))
		return report
	}
	civID := *request.CivID
	unitID := *request.UnitID
	rowIndex := *request.RowIndex
	report.Request.ID = unitID
	if civID < 0 || civID >= len(idx.Civs) {
		report.Reason = fmt.Sprintf("civ_id=%d outside civ table length %d", civID, len(idx.Civs))
		return report
	}
	civ := idx.Civs[civID]
	if unitID < 0 || unitID >= len(civ.Units) || !civ.Units[unitID].Present {
		report.Reason = fmt.Sprintf("unit id=%d is not present for civ %d", unitID, civID)
		return report
	}
	unit := civ.Units[unitID]
	patch := datfile.UnitRecipePatch{CivID: civID, UnitID: unitID}
	rowCount := 0
	switch kind {
	case "damage_graphic":
		rowCount = len(unit.DamageGraphics)
		if rowIndex < 0 || rowIndex >= rowCount {
			report.Reason = fmt.Sprintf("civ=%d unit=%d damage_graphic row_index=%d outside row list length %d", civID, unitID, rowIndex, rowCount)
			return report
		}
		rows := removeDamageGraphicSummaryRows(unit.DamageGraphics, rowIndex)
		patch.SetDamageGraphics = &rows
	case "attack":
		if unit.Type50 == nil {
			report.Reason = fmt.Sprintf("civ=%d unit=%d has no type50 attack/armour subrecord", civID, unitID)
			return report
		}
		rowCount = len(unit.Type50.Attacks)
		if rowIndex < 0 || rowIndex >= rowCount {
			report.Reason = fmt.Sprintf("civ=%d unit=%d attack row_index=%d outside row list length %d", civID, unitID, rowIndex, rowCount)
			return report
		}
		rows := removeWeaponInfoSummaryRows(unit.Type50.Attacks, rowIndex)
		patch.SetType50Attacks = &rows
	case "armour":
		if unit.Type50 == nil {
			report.Reason = fmt.Sprintf("civ=%d unit=%d has no type50 attack/armour subrecord", civID, unitID)
			return report
		}
		rowCount = len(unit.Type50.Armours)
		if rowIndex < 0 || rowIndex >= rowCount {
			report.Reason = fmt.Sprintf("civ=%d unit=%d armour row_index=%d outside row list length %d", civID, unitID, rowIndex, rowCount)
			return report
		}
		rows := removeWeaponInfoSummaryRows(unit.Type50.Armours, rowIndex)
		patch.SetType50Armours = &rows
	case "train_location":
		if unit.Creatable == nil {
			report.Reason = fmt.Sprintf("civ=%d unit=%d has no creatable subrecord", civID, unitID)
			return report
		}
		rowCount = len(unit.Creatable.TrainLocations)
		if rowIndex < 0 || rowIndex >= rowCount {
			report.Reason = fmt.Sprintf("civ=%d unit=%d train_location row_index=%d outside row list length %d", civID, unitID, rowIndex, rowCount)
			return report
		}
		rows := removeTrainLocationSummaryRows(unit.Creatable.TrainLocations, rowIndex)
		patch.SetTrainLocations = &rows
	case "drop_site":
		if unit.Action == nil {
			report.Reason = fmt.Sprintf("civ=%d unit=%d has no action subrecord", civID, unitID)
			return report
		}
		rowCount = len(unit.Action.DropSites)
		if rowIndex < 0 || rowIndex >= rowCount {
			report.Reason = fmt.Sprintf("civ=%d unit=%d drop_site row_index=%d outside row list length %d", civID, unitID, rowIndex, rowCount)
			return report
		}
		rows := removeInt16Rows(unit.Action.DropSites, rowIndex)
		patch.SetDropSites = &rows
	case "task":
		if unit.Action == nil {
			report.Reason = fmt.Sprintf("civ=%d unit=%d has no action subrecord", civID, unitID)
			return report
		}
		rowCount = len(unit.Action.Tasks)
		if rowIndex < 0 || rowIndex >= rowCount {
			report.Reason = fmt.Sprintf("civ=%d unit=%d task row_index=%d outside row list length %d", civID, unitID, rowIndex, rowCount)
			return report
		}
		rows := removeTaskSummaryRows(unit.Action.Tasks, rowIndex)
		patch.SetTasks = &rows
	default:
		report.Strategy = "unsupported"
		report.Reason = fmt.Sprintf("unknown unit child row kind %q", kind)
		return report
	}
	report.Supported = true
	report.Mutates = true
	report.RecipeHint = mustRecipeHint(Recipe{Units: []datfile.UnitRecipePatch{patch}})
	report.Warnings = append(report.Warnings, fmt.Sprintf("Unit %s row delete rebuilds one count-prefixed list for civ=%d unit=%d. Later row indices in that list shift down by one; in-engine behavior remains unverified until the patched DAT is tested.", strings.ReplaceAll(kind, "_", "-"), civID, unitID))
	return report
}

func unitHeaderTaskDeletePlan(idx *datfile.Index, report DeletePlanReport, request DeletePlanRequest) DeletePlanReport {
	report.Strategy = "physical_unit_header_task_row_delete"
	if request.UnitHeaderID == nil || request.RowIndex == nil {
		report.Reason = "unit-header-task delete needs target formatted as <unit_header_id>:<row_index>"
		return report
	}
	headerID := *request.UnitHeaderID
	rowIndex := *request.RowIndex
	report.Request.ID = headerID
	if headerID < 0 || headerID >= len(idx.UnitHeaders) {
		report.Reason = fmt.Sprintf("unit_header_id=%d outside unit header table length %d", headerID, len(idx.UnitHeaders))
		return report
	}
	header := idx.UnitHeaders[headerID]
	if !header.Exists {
		report.Reason = fmt.Sprintf("unit_header_id=%d is absent", headerID)
		return report
	}
	if rowIndex < 0 || rowIndex >= len(header.Tasks) {
		report.Reason = fmt.Sprintf("unit_header_id=%d row_index=%d outside task list length %d", headerID, rowIndex, len(header.Tasks))
		return report
	}
	rows := removeTaskSummaryRows(header.Tasks, rowIndex)
	report.Supported = true
	report.Mutates = true
	report.RecipeHint = mustRecipeHint(Recipe{UnitHeaders: []UnitHeaderPatchRecipe{{ID: headerID, SetTasks: &rows}}})
	report.Warnings = append(report.Warnings, fmt.Sprintf("Unit-header task row delete rebuilds the task list for unit_header_id=%d. Later task indices in that header shift down by one; in-engine behavior remains unverified until the patched DAT is tested.", headerID))
	return report
}

func graphicChildRowDeletePlan(idx *datfile.Index, report DeletePlanReport, request DeletePlanRequest, kind string) DeletePlanReport {
	report.Strategy = "physical_graphic_child_row_delete"
	if request.GraphicID == nil || request.RowIndex == nil {
		report.Reason = fmt.Sprintf("graphic-%s delete needs target formatted as <graphic_id>:<row_index>", strings.ReplaceAll(kind, "_", "-"))
		return report
	}
	graphicID := *request.GraphicID
	rowIndex := *request.RowIndex
	report.Request.ID = graphicID
	if graphicID < 0 || graphicID >= len(idx.Graphics) {
		report.Reason = fmt.Sprintf("graphic_id=%d outside graphics table length %d", graphicID, len(idx.Graphics))
		return report
	}
	graphic := idx.Graphics[graphicID]
	if !graphic.Present {
		report.Reason = fmt.Sprintf("graphic_id=%d is absent", graphicID)
		return report
	}
	patch := datfile.GraphicRecipePatch{ID: graphicID}
	switch kind {
	case "delta":
		if rowIndex < 0 || rowIndex >= len(graphic.Deltas) {
			report.Reason = fmt.Sprintf("graphic_id=%d delta row_index=%d outside row list length %d", graphicID, rowIndex, len(graphic.Deltas))
			return report
		}
		rows := removeGraphicDeltaRows(graphic.Deltas, rowIndex)
		patch.SetDeltas = &rows
	case "angle_sound":
		if rowIndex < 0 || rowIndex >= len(graphic.AngleSounds) {
			report.Reason = fmt.Sprintf("graphic_id=%d angle_sound row_index=%d outside row list length %d", graphicID, rowIndex, len(graphic.AngleSounds))
			return report
		}
		rows := removeGraphicAngleSoundRows(graphic.AngleSounds, rowIndex)
		if len(rows) != 0 && len(rows) != int(graphic.AngleCount) {
			report.Strategy = "unsupported"
			report.Reason = fmt.Sprintf("graphic_id=%d angle_sound partial delete would leave %d rows, but angle_count is %d; use a full set_angle_sounds replacement instead", graphicID, len(rows), graphic.AngleCount)
			return report
		}
		patch.SetAngleSounds = &rows
	default:
		report.Strategy = "unsupported"
		report.Reason = fmt.Sprintf("unknown graphic child row kind %q", kind)
		return report
	}
	report.References = graphicIDReferences(idx, graphicID)
	report.Supported = true
	report.Mutates = true
	report.RecipeHint = mustRecipeHint(Recipe{Graphics: []datfile.GraphicRecipePatch{patch}})
	report.Warnings = append(report.Warnings, fmt.Sprintf("Graphic %s row delete rebuilds one count-prefixed list for graphic_id=%d. Later row indices in that list shift down by one; in-engine render behavior remains unverified until the patched DAT is tested.", strings.ReplaceAll(kind, "_", "-"), graphicID))
	return report
}

func removeDamageGraphicSummaryRows(in []datfile.DamageGraphic, removeIndex int) []datfile.DamageGraphicRow {
	out := make([]datfile.DamageGraphicRow, 0, len(in)-1)
	for i, row := range in {
		if i == removeIndex {
			continue
		}
		out = append(out, datfile.DamageGraphicRow{
			GraphicID:     row.GraphicID,
			DamagePercent: row.DamagePercent,
			Flag:          row.Flag,
		})
	}
	return out
}

func removeGraphicDeltaRows(in []datfile.GraphicDelta, removeIndex int) []datfile.GraphicDeltaRow {
	out := make([]datfile.GraphicDeltaRow, 0, len(in)-1)
	for i, row := range in {
		if i == removeIndex {
			continue
		}
		out = append(out, datfile.GraphicDeltaRow{
			GraphicID:    row.GraphicID,
			Padding1:     row.Padding1,
			SpritePtr:    row.SpritePtr,
			OffsetX:      row.OffsetX,
			OffsetY:      row.OffsetY,
			DisplayAngle: row.DisplayAngle,
			Padding2:     row.Padding2,
		})
	}
	return out
}

func removeGraphicAngleSoundRows(in []datfile.GraphicAngleSound, removeIndex int) []datfile.GraphicAngleSoundRow {
	out := make([]datfile.GraphicAngleSoundRow, 0, len(in)-1)
	for i, row := range in {
		if i == removeIndex {
			continue
		}
		out = append(out, datfile.GraphicAngleSoundRow{
			FrameNum:      row.FrameNum,
			SoundID:       row.SoundID,
			WwiseSoundID:  row.WwiseSoundID,
			FrameNum2:     row.FrameNum2,
			WwiseSoundID2: row.WwiseSoundID2,
			SoundID2:      row.SoundID2,
			FrameNum3:     row.FrameNum3,
			WwiseSoundID3: row.WwiseSoundID3,
			SoundID3:      row.SoundID3,
		})
	}
	return out
}

func removeWeaponInfoSummaryRows(in []datfile.WeaponInfo, removeIndex int) []datfile.WeaponInfoRow {
	out := make([]datfile.WeaponInfoRow, 0, len(in)-1)
	for i, row := range in {
		if i == removeIndex {
			continue
		}
		out = append(out, datfile.WeaponInfoRow{
			Class: row.Class,
			Value: row.Value,
		})
	}
	return out
}

func removeTrainLocationSummaryRows(in []datfile.TrainLocation, removeIndex int) []datfile.TrainLocationRow {
	out := make([]datfile.TrainLocationRow, 0, len(in)-1)
	for i, row := range in {
		if i == removeIndex {
			continue
		}
		out = append(out, datfile.TrainLocationRow{
			TrainTime:   row.TrainTime,
			TrainUnitID: row.TrainUnitID,
			TrainButton: row.TrainButton,
			TrainHotkey: row.TrainHotkey,
		})
	}
	return out
}

func removeInt16Rows(in []int16, removeIndex int) []int16 {
	out := make([]int16, 0, len(in)-1)
	for i, row := range in {
		if i == removeIndex {
			continue
		}
		out = append(out, row)
	}
	return out
}

func removeTaskSummaryRows(in []datfile.TaskSummary, removeIndex int) []datfile.TaskRow {
	out := make([]datfile.TaskRow, 0, len(in)-1)
	for i, task := range in {
		if i == removeIndex {
			continue
		}
		out = append(out, datfile.TaskRow{
			RecordType:        task.RecordType,
			ID:                task.ID,
			IsDefault:         task.IsDefault,
			ActionType:        task.ActionType,
			ObjectClass:       task.ObjectClass,
			ObjectID:          task.ObjectID,
			TerrainID:         task.TerrainID,
			AttributeTypes:    append([]int16(nil), task.AttributeTypes...),
			WorkValue1:        task.WorkValue1,
			WorkValue2:        task.WorkValue2,
			WorkRange:         task.WorkRange,
			AutoSearchTargets: task.AutoSearchTargets,
			SearchWaitTime:    task.SearchWaitTime,
			EnableTargeting:   task.EnableTargeting,
			CombatLevel:       task.CombatLevel,
			RawTail:           append([]byte(nil), task.RawTail...),
		})
	}
	return out
}

func setUnitDeletePlanCleanup(report *DeletePlanReport, unitID int) {
	for _, ref := range report.References {
		if unitReferenceRewriteCovers(ref) {
			setDeletePlanOptionalCleanup(report, "unit", unitID, Recipe{DisconnectUnits: []int{unitID}})
			return
		}
	}
}

func unitIDReferences(idx *datfile.Index, id int) []DeleteReference {
	var refs []DeleteReference
	for _, header := range idx.UnitHeaders {
		if !header.Exists {
			continue
		}
		for _, task := range header.Tasks {
			if int(task.ObjectID) == id {
				refs = append(refs, DeleteReference{Section: "unit_header", ID: header.Index, Field: fmt.Sprintf("task[%d].object_id", task.Index)})
			}
		}
	}
	for _, civ := range idx.Civs {
		for _, unit := range civ.Units {
			if !unit.Present {
				continue
			}
			if int(unit.BloodUnitID) == id {
				refs = append(refs, DeleteReference{Section: "unit", ID: unit.Index, Field: fmt.Sprintf("civ[%d].blood_unit_id", civ.Index)})
			}
			if unit.Type50 != nil && int(unit.Type50.ProjectileUnitID) == id {
				refs = append(refs, DeleteReference{Section: "unit", ID: unit.Index, Field: fmt.Sprintf("civ[%d].type50_projectile_unit_id", civ.Index)})
			}
			if unit.Action != nil {
				for i, dropSiteID := range unit.Action.DropSites {
					if int(dropSiteID) == id {
						refs = append(refs, DeleteReference{Section: "unit", ID: unit.Index, Field: fmt.Sprintf("civ[%d].drop_site[%d]", civ.Index, i)})
					}
				}
				for _, task := range unit.Action.Tasks {
					if int(task.ObjectID) == id {
						refs = append(refs, DeleteReference{Section: "unit", ID: unit.Index, Field: fmt.Sprintf("civ[%d].task[%d].object_id", civ.Index, task.Index)})
					}
				}
			}
			if unit.Creatable != nil {
				for _, row := range unit.Creatable.TrainLocations {
					if int(row.TrainUnitID) == id {
						refs = append(refs, DeleteReference{Section: "unit", ID: unit.Index, Field: fmt.Sprintf("civ[%d].train_location[%d].train_unit_id", civ.Index, row.Index)})
					}
				}
			}
			if unit.Building != nil {
				for _, row := range unit.Building.LinkedBuildings {
					if int(row.UnitID) == id {
						refs = append(refs, DeleteReference{Section: "unit", ID: unit.Index, Field: fmt.Sprintf("civ[%d].linked_building[%d].unit_id", civ.Index, row.Index)})
					}
				}
			}
		}
	}
	for _, age := range idx.TechTree.Ages {
		appendInt32Refs(&refs, "tech_tree_age", age.Index, "buildings", age.Buildings, id)
		appendInt32Refs(&refs, "tech_tree_age", age.Index, "units", age.Units, id)
	}
	for _, connection := range idx.TechTree.BuildingConnections {
		if int(connection.ID) == id {
			refs = append(refs, DeleteReference{Section: "tech_tree_building_connection", ID: connection.Index, Field: "id"})
		}
		appendInt32Refs(&refs, "tech_tree_building_connection", connection.Index, "buildings", connection.Buildings, id)
		appendInt32Refs(&refs, "tech_tree_building_connection", connection.Index, "units", connection.Units, id)
	}
	for _, connection := range idx.TechTree.UnitConnections {
		if int(connection.ID) == id {
			refs = append(refs, DeleteReference{Section: "tech_tree_unit_connection", ID: connection.Index, Field: "id"})
		}
		if int(connection.UpperBuilding) == id {
			refs = append(refs, DeleteReference{Section: "tech_tree_unit_connection", ID: connection.Index, Field: "upper_building"})
		}
		appendInt32Refs(&refs, "tech_tree_unit_connection", connection.Index, "units", connection.Units, id)
	}
	for _, connection := range idx.TechTree.ResearchConnections {
		if int(connection.UpperBuilding) == id {
			refs = append(refs, DeleteReference{Section: "tech_tree_research_connection", ID: connection.Index, Field: "upper_building"})
		}
		appendInt32Refs(&refs, "tech_tree_research_connection", connection.Index, "buildings", connection.Buildings, id)
		appendInt32Refs(&refs, "tech_tree_research_connection", connection.Index, "units", connection.Units, id)
	}
	refs = append(refs, typedEffectCommandReferences(idx, id, "unit")...)
	refs = append(refs, candidateEffectCommandReferences(idx, id, "unit")...)
	refs = append(refs, possibleEffectCommandReferences(idx, id, "unit")...)
	sortReferences(refs)
	return refs
}

func techIDReferences(idx *datfile.Index, id int) []DeleteReference {
	var refs []DeleteReference
	for _, tech := range idx.Techs {
		appendInt16Refs(&refs, "tech", tech.Index, "required_techs", tech.RequiredTechs, id)
	}
	for _, age := range idx.TechTree.Ages {
		appendInt32Refs(&refs, "tech_tree_age", age.Index, "techs", age.Techs, id)
		appendTechTreeCommonRefs(&refs, "tech_tree_age", age.Index, age.Common, id)
	}
	for _, connection := range idx.TechTree.BuildingConnections {
		appendInt32Refs(&refs, "tech_tree_building_connection", connection.Index, "techs", connection.Techs, id)
		appendTechTreeCommonRefs(&refs, "tech_tree_building_connection", connection.Index, connection.Common, id)
		if int(connection.EnablingResearch) == id {
			refs = append(refs, DeleteReference{Section: "tech_tree_building_connection", ID: connection.Index, Field: "enabling_research"})
		}
	}
	for _, connection := range idx.TechTree.UnitConnections {
		appendTechTreeCommonRefs(&refs, "tech_tree_unit_connection", connection.Index, connection.Common, id)
		if int(connection.RequiredResearch) == id {
			refs = append(refs, DeleteReference{Section: "tech_tree_unit_connection", ID: connection.Index, Field: "required_research"})
		}
		if int(connection.EnablingResearch) == id {
			refs = append(refs, DeleteReference{Section: "tech_tree_unit_connection", ID: connection.Index, Field: "enabling_research"})
		}
	}
	for _, connection := range idx.TechTree.ResearchConnections {
		if int(connection.ID) == id {
			refs = append(refs, DeleteReference{Section: "tech_tree_research_connection", ID: connection.Index, Field: "id"})
		}
		appendInt32Refs(&refs, "tech_tree_research_connection", connection.Index, "techs", connection.Techs, id)
		appendTechTreeCommonRefs(&refs, "tech_tree_research_connection", connection.Index, connection.Common, id)
	}
	refs = append(refs, typedEffectCommandReferences(idx, id, "tech")...)
	refs = append(refs, possibleEffectCommandReferences(idx, id, "tech")...)
	sortReferences(refs)
	return refs
}

func typedEffectCommandReferences(idx *datfile.Index, id int, target string) []DeleteReference {
	var refs []DeleteReference
	for _, effect := range idx.Effects {
		for _, command := range effect.Commands {
			typeName := datfile.EffectCommandTypeName(command.Type)
			if command.Semantic != nil {
				typeName = command.Semantic.TypeName
			}
			for _, ref := range effectCommandSemanticReferences(command) {
				if ref.Kind != target || ref.ID != id {
					continue
				}
				refs = append(refs, DeleteReference{
					Section:    "effect",
					ID:         effect.Index,
					Field:      fmt.Sprintf("command[%d].%s_%s_id(type=%s)", command.Index, ref.Field, target, typeName),
					Confidence: "typed_effect_command_reference",
				})
			}
		}
	}
	return refs
}

func possibleEffectCommandReferences(idx *datfile.Index, id int, target string) []DeleteReference {
	var refs []DeleteReference
	for _, effect := range idx.Effects {
		for _, command := range effect.Commands {
			if len(effectCommandSemanticReferences(command)) > 0 {
				continue
			}
			if !allowPossibleEffectCommandOperandScan(command) {
				continue
			}
			if commandHasTypedEffectCommandReference(command, id, target) {
				continue
			}
			if commandHasCandidateEffectCommandReference(command, id, target) {
				continue
			}
			refs = appendPossibleEffectCommandRef(refs, effect.Index, command, "a", int(command.A), id, target)
			refs = appendPossibleEffectCommandRef(refs, effect.Index, command, "b", int(command.B), id, target)
			refs = appendPossibleEffectCommandRef(refs, effect.Index, command, "c", int(command.C), id, target)
			d := int(command.D)
			if command.D == float32(d) {
				refs = appendPossibleEffectCommandRef(refs, effect.Index, command, "d", d, id, target)
			}
		}
	}
	return refs
}

func allowPossibleEffectCommandOperandScan(command datfile.EffectCommand) bool {
	semantic := command.Semantic
	if semantic == nil {
		semantic = datfile.InterpretEffectCommand(command)
	}
	if semantic == nil {
		return true
	}
	return semantic.TypeName == "" || semantic.TypeName == "unknown"
}

func candidateEffectCommandReferences(idx *datfile.Index, id int, target string) []DeleteReference {
	if target != "unit" {
		return nil
	}
	var refs []DeleteReference
	for _, effect := range idx.Effects {
		for _, command := range effect.Commands {
			if len(effectCommandSemanticReferences(command)) > 0 {
				continue
			}
			if commandHasTypedEffectCommandReference(command, id, target) {
				continue
			}
			if !isCandidateUnitEffectCommandA(command.Type) || int(command.A) != id {
				continue
			}
			refs = append(refs, DeleteReference{
				Section:    "effect",
				ID:         effect.Index,
				Field:      fmt.Sprintf("command[%d].a_candidate_unit_id(type=%d)", command.Index, command.Type),
				Confidence: "candidate_effect_command_operand",
			})
		}
	}
	return refs
}

func isCandidateUnitEffectCommandA(commandType uint8) bool {
	return false
}

func commandHasTypedEffectCommandReference(command datfile.EffectCommand, id int, target string) bool {
	for _, ref := range effectCommandSemanticReferences(command) {
		if ref.Kind == target && ref.ID == id {
			return true
		}
	}
	return false
}

func commandHasCandidateEffectCommandReference(command datfile.EffectCommand, id int, target string) bool {
	return target == "unit" && isCandidateUnitEffectCommandA(command.Type) && int(command.A) == id
}

func effectCommandSemanticReferences(command datfile.EffectCommand) []datfile.EffectCommandReference {
	semantic := command.Semantic
	if semantic == nil {
		return nil
	}
	if len(semantic.References) > 0 {
		return semantic.References
	}
	if semantic.ReferenceID == nil || semantic.ReferenceKind == "" || semantic.ReferenceField == "" {
		return nil
	}
	return []datfile.EffectCommandReference{
		{
			Kind:       semantic.ReferenceKind,
			Field:      semantic.ReferenceField,
			ID:         *semantic.ReferenceID,
			Confidence: semantic.Confidence,
		},
	}
}

func appendPossibleEffectCommandRef(refs []DeleteReference, effectID int, command datfile.EffectCommand, operand string, value, targetID int, target string) []DeleteReference {
	if value != targetID {
		return refs
	}
	return append(refs, DeleteReference{
		Section:    "effect",
		ID:         effectID,
		Field:      fmt.Sprintf("command[%d].%s_possible_%s_id(type=%d)", command.Index, operand, target, command.Type),
		Confidence: "possible_effect_command_operand",
	})
}

func hasPossibleEffectCommandRefs(refs []DeleteReference) bool {
	for _, ref := range refs {
		if ref.Confidence == "possible_effect_command_operand" || ref.Confidence == "candidate_effect_command_operand" {
			return true
		}
	}
	return false
}

func hasTypedEffectCommandRefs(refs []DeleteReference) bool {
	for _, ref := range refs {
		if ref.Confidence == "typed_effect_command_reference" {
			return true
		}
	}
	return false
}

func techDeleteMigrationReport(idx *datfile.Index, deleteID int) *TechDeleteMigrationReport {
	report := &TechDeleteMigrationReport{
		DeleteID:       deleteID,
		TailID:         len(idx.Techs) - 1,
		CheckedTechIDs: 2,
	}
	for _, id := range []int{deleteID, report.TailID} {
		refs := techIDReferences(idx, id)
		if len(refs) > 0 {
			report.IDsWithReferences++
		}
		covered, unsupported := techReferenceRewriteCoverage(refs)
		report.CoveredReferences += covered
		report.UnsupportedReferences += unsupported
		if unsupported > 0 {
			report.IDsWithUnsupported++
		}
	}
	report.Ready = report.UnsupportedReferences == 0
	return report
}

func techReferenceRewriteCoverage(refs []DeleteReference) (covered, unsupported int) {
	for _, ref := range refs {
		if techReferenceRewriteCovers(ref) {
			covered++
		} else {
			unsupported++
		}
	}
	return covered, unsupported
}

func techReferenceRewriteCovers(ref DeleteReference) bool {
	if ref.Section == "tech" && strings.HasPrefix(ref.Field, "required_techs[") {
		return true
	}
	if strings.HasPrefix(ref.Section, "tech_tree_") {
		return true
	}
	if ref.Section == "effect" && ref.Confidence == "typed_effect_command_reference" {
		return true
	}
	return false
}

func unitReferenceRewriteCovers(ref DeleteReference) bool {
	if strings.HasPrefix(ref.Section, "tech_tree_") {
		return true
	}
	if ref.Section == "effect" && ref.Confidence == "typed_effect_command_reference" {
		return true
	}
	if ref.Section == "unit_header" && strings.HasPrefix(ref.Field, "task[") && strings.Contains(ref.Field, ".object_id") {
		return true
	}
	if ref.Section == "unit" {
		if strings.Contains(ref.Field, ".blood_unit_id") ||
			strings.Contains(ref.Field, ".drop_site[") ||
			strings.Contains(ref.Field, ".train_location[") ||
			strings.Contains(ref.Field, ".linked_building[") ||
			strings.Contains(ref.Field, ".task[") && strings.Contains(ref.Field, ".object_id") ||
			strings.Contains(ref.Field, ".type50_projectile_unit_id") {
			return true
		}
	}
	return false
}

func civIDReferences(idx *datfile.Index, id int) []DeleteReference {
	var refs []DeleteReference
	for _, tech := range idx.Techs {
		if int(tech.Civ) == id {
			refs = append(refs, DeleteReference{Section: "tech", ID: tech.Index, Field: "civ"})
		}
	}
	for _, sound := range idx.Sounds {
		for _, item := range sound.Items {
			if int(item.Civ) == id {
				refs = append(refs, DeleteReference{Section: "sound", ID: sound.Index, Field: fmt.Sprintf("item[%d].civ", item.Index)})
			}
		}
	}
	sortReferences(refs)
	return refs
}

func terrainIDReferences(idx *datfile.Index, id int) []DeleteReference {
	var refs []DeleteReference
	for _, header := range idx.UnitHeaders {
		if !header.Exists {
			continue
		}
		for _, task := range header.Tasks {
			if int(task.TerrainID) == id {
				refs = append(refs, DeleteReference{Section: "unit_header", ID: header.Index, Field: fmt.Sprintf("task[%d].terrain_id", task.Index)})
			}
		}
	}
	for _, civ := range idx.Civs {
		for _, unit := range civ.Units {
			if !unit.Present || unit.Action == nil {
				continue
			}
			for _, task := range unit.Action.Tasks {
				if int(task.TerrainID) == id {
					refs = append(refs, DeleteReference{Section: "unit", ID: unit.Index, Field: fmt.Sprintf("civ[%d].task[%d].terrain_id", civ.Index, task.Index)})
				}
			}
		}
	}
	sortReferences(refs)
	return refs
}

func graphicIDReferences(idx *datfile.Index, id int) []DeleteReference {
	var refs []DeleteReference
	for _, civ := range idx.Civs {
		for _, unit := range civ.Units {
			if !unit.Present {
				continue
			}
			if int(unit.StandingGraphic1) == id {
				refs = append(refs, DeleteReference{Section: "unit", ID: unit.Index, Field: fmt.Sprintf("civ[%d].standing_graphic_1", civ.Index)})
			}
			if int(unit.StandingGraphic2) == id {
				refs = append(refs, DeleteReference{Section: "unit", ID: unit.Index, Field: fmt.Sprintf("civ[%d].standing_graphic_2", civ.Index)})
			}
			if int(unit.DyingGraphic) == id {
				refs = append(refs, DeleteReference{Section: "unit", ID: unit.Index, Field: fmt.Sprintf("civ[%d].dying_graphic", civ.Index)})
			}
			for _, row := range unit.DamageGraphics {
				if int(row.GraphicID) == id {
					refs = append(refs, DeleteReference{Section: "unit", ID: unit.Index, Field: fmt.Sprintf("civ[%d].damage_graphic[%d].graphic_id", civ.Index, row.Index)})
				}
			}
			if unit.Type50 != nil && int(unit.Type50.AttackGraphic) == id {
				refs = append(refs, DeleteReference{Section: "unit", ID: unit.Index, Field: fmt.Sprintf("civ[%d].type50_attack_graphic", civ.Index)})
			}
		}
	}
	for _, graphic := range idx.Graphics {
		if !graphic.Present {
			continue
		}
		for _, delta := range graphic.Deltas {
			if int(delta.GraphicID) == id {
				refs = append(refs, DeleteReference{Section: "graphic", ID: graphic.Index, Field: fmt.Sprintf("delta[%d].graphic_id", delta.Index)})
			}
		}
	}
	sortReferences(refs)
	return refs
}

func playerColourIDReferences(idx *datfile.Index, id int) []DeleteReference {
	var refs []DeleteReference
	for _, graphic := range idx.Graphics {
		if !graphic.Present {
			continue
		}
		if int(graphic.PlayerColor) == id {
			refs = append(refs, DeleteReference{Section: "graphic", ID: graphic.Index, Field: "player_color"})
		}
	}
	sortReferences(refs)
	return refs
}

func soundIDReferences(idx *datfile.Index, id int) []DeleteReference {
	var refs []DeleteReference
	for _, graphic := range idx.Graphics {
		if !graphic.Present {
			continue
		}
		if int(graphic.SoundID) == id {
			refs = append(refs, DeleteReference{Section: "graphic", ID: graphic.Index, Field: "sound_id"})
		}
		for _, row := range graphic.AngleSounds {
			if int(row.SoundID) == id {
				refs = append(refs, DeleteReference{Section: "graphic", ID: graphic.Index, Field: fmt.Sprintf("angle_sound[%d].sound_id", row.Index)})
			}
			if int(row.SoundID2) == id {
				refs = append(refs, DeleteReference{Section: "graphic", ID: graphic.Index, Field: fmt.Sprintf("angle_sound[%d].sound_id_2", row.Index)})
			}
			if int(row.SoundID3) == id {
				refs = append(refs, DeleteReference{Section: "graphic", ID: graphic.Index, Field: fmt.Sprintf("angle_sound[%d].sound_id_3", row.Index)})
			}
		}
	}
	sortReferences(refs)
	return refs
}

func appendInt16Refs(refs *[]DeleteReference, section string, id int, field string, values []int16, target int) {
	for i, value := range values {
		if int(value) == target {
			*refs = append(*refs, DeleteReference{Section: section, ID: id, Field: fmt.Sprintf("%s[%d]", field, i)})
		}
	}
}

func appendInt32Refs(refs *[]DeleteReference, section string, id int, field string, values []int32, target int) {
	for i, value := range values {
		if int(value) == target {
			*refs = append(*refs, DeleteReference{Section: section, ID: id, Field: fmt.Sprintf("%s[%d]", field, i)})
		}
	}
}

func appendTechTreeCommonRefs(refs *[]DeleteReference, section string, id int, common datfile.TechTreeCommon, target int) {
	appendInt32Refs(refs, section, id, "common.unit_research", common.UnitResearch, target)
}

func sortReferences(refs []DeleteReference) {
	sort.Slice(refs, func(i, j int) bool {
		if refs[i].Section != refs[j].Section {
			return refs[i].Section < refs[j].Section
		}
		if refs[i].ID != refs[j].ID {
			return refs[i].ID < refs[j].ID
		}
		return refs[i].Field < refs[j].Field
	})
}

func mustRecipeHint(recipe Recipe) json.RawMessage {
	data, err := json.Marshal(recipe)
	if err != nil {
		panic(err)
	}
	return data
}
