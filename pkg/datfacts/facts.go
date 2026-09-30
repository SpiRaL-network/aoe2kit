package datfacts

import (
	"fmt"
	"sort"

	"aoe2kit/pkg/aoe2"
	"aoe2kit/pkg/datfile"
)

const Version = "datfacts.v1"

type CatalogReport struct {
	Version      string                 `json:"version"`
	Domain       string                 `json:"domain"`
	Summary      CatalogSummary         `json:"summary"`
	Verification aoe2.VerificationClaim `json:"verification"`
	Rows         []FactRow              `json:"rows"`
}

type CatalogSummary struct {
	Rows               int      `json:"rows"`
	EngineVerified     int      `json:"engine_verified"`
	StrongHypothesis   int      `json:"strong_hypothesis"`
	SourceCorrelated   int      `json:"source_correlated"`
	KnownDisagreements int      `json:"known_disagreements"`
	Sources            []string `json:"sources,omitempty"`
}

type FactRow struct {
	ID            int      `json:"id"`
	Name          string   `json:"name"`
	Meaning       string   `json:"meaning"`
	Family        string   `json:"family,omitempty"`
	Scope         string   `json:"scope,omitempty"`
	Confidence    string   `json:"confidence"`
	Sources       []string `json:"sources,omitempty"`
	Disagreements []string `json:"disagreements,omitempty"`
	SafeUse       string   `json:"safe_use,omitempty"`
	Notes         []string `json:"notes,omitempty"`
}

func Catalog(domain string) (CatalogReport, error) {
	switch domain {
	case "effect-types", "effect-type", "effect-commands", "effect-command":
		return effectTypeCatalog(), nil
	case "attributes", "attribute":
		return simpleCatalog("attributes", attributeRows()), nil
	case "resources", "resource":
		return simpleCatalog("resources", resourceRows()), nil
	case "task-types", "tasks", "task":
		return simpleCatalog("task-types", taskRows()), nil
	case "scenario-unit-classes", "scenario-unit-class", "unit-classes", "unit-class":
		return simpleCatalog("scenario-unit-classes", scenarioUnitClassRows()), nil
	case "scenario-object-types", "scenario-object-type", "object-types", "object-type":
		return simpleCatalog("scenario-object-types", scenarioObjectTypeRows()), nil
	case "scenario-unit-ai-actions", "scenario-unit-ai-action", "unit-ai-actions", "unit-ai-action":
		return simpleCatalog("scenario-unit-ai-actions", scenarioUnitAIActionRows()), nil
	case "tech-modifiers", "tech-attributes", "tech-attrs":
		return simpleCatalog("tech-modifiers", techModifierRows()), nil
	default:
		return CatalogReport{}, fmt.Errorf("unknown DAT facts domain %q", domain)
	}
}

func ScenarioUnitClassName(id int) string {
	if id < 0 {
		return ""
	}
	return datfile.EffectUnitClassName(int16(id + 900))
}

func ScenarioObjectTypeName(id int) string {
	return scenarioObjectTypeNames()[id]
}

func ScenarioUnitAIActionName(id int) string {
	return scenarioUnitAIActionNames()[id]
}

func ExplainEffectCommandType(id int) (FactRow, error) {
	report := effectTypeCatalog()
	for _, row := range report.Rows {
		if row.ID == id {
			return row, nil
		}
	}
	name := datfile.EffectCommandTypeName(uint8(id))
	if name == "unknown" {
		return FactRow{}, fmt.Errorf("unknown effect command type %d", id)
	}
	return FactRow{
		ID:         id,
		Name:       name,
		Meaning:    "recognized by AoE2Kit's command-name table, but not yet promoted into the detailed facts catalog",
		Confidence: "typed_by_kit_table",
		Sources:    []string{"AoE2Kit datfile.EffectCommandTypeName"},
		SafeUse:    "inspect this command in a real DAT with kit dat command-matrix before authoring new rows",
	}, nil
}

func effectTypeCatalog() CatalogReport {
	rows := []FactRow{
		effectRow(0, "set_attribute", "set an object attribute", "attribute", "", "source_correlated", "safe for ordinary scalar attributes; attack/armor packed values need class-aware helpers"),
		effectRow(1, "resource_modifier", "set/add a player resource/stat value", "resource", "", "source_correlated", "safe when resource id and operation id are explicit"),
		effectRow(2, "enable_unit", "enable or disable a unit for the target player scope", "unit", "", "source_correlated", "safe for availability toggles; validate civ/team scope separately"),
		effectRow(3, "upgrade_unit", "upgrade/replace one unit id with another", "unit", "", "source_correlated", "treat as authored-data only until the intended runtime path is engine-verified"),
		effectRow(4, "add_attribute", "add to an object attribute", "attribute", "", "source_correlated", "safe for ordinary scalar attributes; use packed helpers for attack/armor class deltas"),
		effectRow(5, "attribute_multiplier", "multiply an object attribute", "attribute", "", "source_correlated", "safe for scalar attributes when multiplier semantics are intended"),
		effectRow(6, "resource_multiplier", "multiply a player resource/stat value", "resource", "", "source_correlated", "use cautiously for score/stat resources; verify the target resource is multiplicative"),
		effectRow(7, "spawn_unit", "spawn units when the effect is applied", "unit", "", "source_correlated", "runtime behavior depends on application path; verify before relying on it"),
		effectRow(8, "modify_tech", "modify a technology field such as cost, icon, button, effect, state, or time", "tech", "", "source_correlated", "safe for known tech modifiers; prefer named tech-modifier constants"),
		effectRow(9, "set_player_data", "set player data/civilization-name style metadata", "player-data", "", "source_correlated", "rare; inspect examples before use"),
	}
	for _, family := range []struct {
		offset int
		scope  string
	}{
		{10, "team"},
		{20, "enemy"},
		{30, "neutral"},
		{40, "gaia"},
	} {
		baseNames := []string{"set_attribute", "resource_modifier", "enable_unit", "upgrade_unit", "add_attribute", "attribute_multiplier", "resource_multiplier", "spawn_unit", "modify_tech", "set_player_data"}
		for i, base := range baseNames {
			id := family.offset + i
			if family.offset == 40 && id > 48 {
				continue
			}
			name := datfile.EffectCommandTypeName(uint8(id))
			if name == "unknown" {
				name = family.scope + "_" + base
			}
			rows = append(rows, effectRow(id, name, fmt.Sprintf("%s-scoped %s", family.scope, base), scopedFamily(base), family.scope, "source_correlated", "same operand shape as the base command, with scoped runtime application"))
		}
	}
	rows = append(rows,
		effectRow(100, "set_tech_cost", "legacy/AGE-shaped tech cost setter", "tech", "", "source_correlated", "prefer command 101 unless a real DAT row demonstrates 100 is required"),
		effectRow(101, "tech_cost_modifier", "modify a technology resource cost", "tech", "", "source_correlated", "safe with named resource ids and operation semantics"),
		effectRow(102, "disable_tech", "disable a technology", "tech", "", "source_correlated", "safe when target tech id is explicit; check UI availability after write"),
		effectRow(103, "tech_time_modifier", "modify a technology research time", "tech", "", "source_correlated", "safe with operation id set/add semantics"),
		FactRow{
			ID:         200,
			Name:       "set_local_building_attribute",
			Meaning:    "set an attribute on the local building instance that researched a building-specific technology",
			Family:     "local-building",
			Scope:      "local_building",
			Confidence: "source_verified_for_200_201",
			Sources:    defaultSources("official DE update notes define 200 as local-building set"),
			SafeUse:    "safe as local-building set when paired with type-32 building-specific technologies",
		},
		FactRow{
			ID:         201,
			Name:       "add_local_building_attribute",
			Meaning:    "add/subtract an attribute on the local building instance that researched a building-specific technology",
			Family:     "local-building",
			Scope:      "local_building",
			Confidence: "source_verified_for_200_201",
			Sources:    defaultSources("official DE update notes define 201 as local-building add/subtract"),
			SafeUse:    "safe as local-building add/subtract when paired with type-32 building-specific technologies",
		},
		FactRow{
			ID:         202,
			Name:       "multiply_local_building_attribute",
			Meaning:    "multiply a local-building attribute; observed rows fit the 200/201 local-building family extension",
			Family:     "local-building",
			Scope:      "local_building",
			Confidence: "strong_hypothesis",
			Sources:    defaultSources("observed DE DAT rows; GoKu labels this own-master-objects multiply"),
			Disagreements: []string{
				"GoKu aoe2-genie-tooling labels 202 own_master_objects_attribute_modifier_multiply",
				"AoE2Kit currently treats 202 as local-building multiply from observed DE rows",
			},
			SafeUse: "use for local-building multiply only with explicit readback/engine verification",
		},
		FactRow{
			ID:         203,
			Name:       "selected_unit_attribute_modifier_set",
			Meaning:    "GoKu labels this selected-unit set; AoE2Kit has not verified current-DE rows",
			Family:     "selected-unit",
			Scope:      "selected_unit",
			Confidence: "external_oracle_unverified",
			Sources:    []string{"GoKu aoe2-genie-tooling EffectCommandBuilder"},
			Disagreements: []string{
				"not currently promoted in AoE2Kit's command interpreter",
				"needs dedicated engine fixture before safe authoring",
			},
			SafeUse: "do not author by default; create a semantics fixture first",
		},
		FactRow{
			ID:         204,
			Name:       "add_local_building_attribute_advanced",
			Meaning:    "AoE2Kit observed this as local-building additive/packed attack-armor rows; GoKu labels it selected-unit add",
			Family:     "local-building/selected-unit-disputed",
			Scope:      "local_building",
			Confidence: "strong_hypothesis_with_known_disagreement",
			Sources:    defaultSources("observed DE DAT rows; GoKu labels this selected-unit add"),
			Disagreements: []string{
				"GoKu aoe2-genie-tooling labels 204 selected_unit_attribute_modifier_add",
				"AoE2Kit currently treats 204 as local-building advanced add based on observed official rows",
			},
			SafeUse: "use only through named helpers such as add_local_building_attack/armor, and keep the plan/readback warning visible",
		},
		FactRow{
			ID:         205,
			Name:       "selected_unit_attribute_modifier_multiply",
			Meaning:    "GoKu labels this selected-unit multiply; AoE2Kit has not verified current-DE rows",
			Family:     "selected-unit",
			Scope:      "selected_unit",
			Confidence: "external_oracle_unverified",
			Sources:    []string{"GoKu aoe2-genie-tooling EffectCommandBuilder"},
			SafeUse:    "do not author by default; create a semantics fixture first",
		},
		FactRow{
			ID:         206,
			Name:       "transform_selected_unit",
			Meaning:    "GoKu labels this selected-unit transform; AoE2Kit has not verified current-DE rows",
			Family:     "selected-unit",
			Scope:      "selected_unit",
			Confidence: "external_oracle_unverified",
			Sources:    []string{"GoKu aoe2-genie-tooling EffectCommandBuilder"},
			SafeUse:    "do not author by default; create a semantics fixture first",
		},
	)
	sort.Slice(rows, func(i, j int) bool { return rows[i].ID < rows[j].ID })
	return simpleCatalog("effect-types", rows)
}

func effectRow(id int, name, meaning, family, scope, confidence, safeUse string) FactRow {
	return FactRow{
		ID:         id,
		Name:       name,
		Meaning:    meaning,
		Family:     family,
		Scope:      scope,
		Confidence: confidence,
		Sources:    defaultSources("AGE/Genie command family naming and AoE2Kit decoded DAT rows"),
		SafeUse:    safeUse,
	}
}

func scopedFamily(base string) string {
	switch base {
	case "set_attribute", "add_attribute", "attribute_multiplier":
		return "attribute"
	case "resource_modifier", "resource_multiplier":
		return "resource"
	case "enable_unit", "upgrade_unit", "spawn_unit":
		return "unit"
	case "modify_tech":
		return "tech"
	default:
		return "player-data"
	}
}

func defaultSources(extra string) []string {
	return []string{
		"AoE2Kit decoded current-DE DAT rows",
		"Advanced Genie Editor / Genie command terminology",
		"GoKuModder/aoe2-genie-tooling LGPL-3.0 reference catalog",
		extra,
	}
}

func simpleCatalog(domain string, rows []FactRow) CatalogReport {
	sort.Slice(rows, func(i, j int) bool { return rows[i].ID < rows[j].ID })
	summary := CatalogSummary{Rows: len(rows)}
	sourceSet := map[string]bool{}
	for _, row := range rows {
		switch row.Confidence {
		case "engine_verified":
			summary.EngineVerified++
		case "strong_hypothesis", "strong_hypothesis_with_known_disagreement":
			summary.StrongHypothesis++
		default:
			summary.SourceCorrelated++
		}
		if len(row.Disagreements) > 0 {
			summary.KnownDisagreements++
		}
		for _, source := range row.Sources {
			sourceSet[source] = true
		}
	}
	for source := range sourceSet {
		summary.Sources = append(summary.Sources, source)
	}
	sort.Strings(summary.Sources)
	claim := aoe2.StructureVerification(true)
	claim.Note = "DAT facts are an evidence-tiered authoring catalog assembled from Kit decodes, upstream references, and external oracles. They are not automatically engine proof."
	return CatalogReport{
		Version:      Version,
		Domain:       domain,
		Summary:      summary,
		Verification: claim,
		Rows:         rows,
	}
}

func attributeRows() []FactRow {
	var rows []FactRow
	for id := -1; id <= 160; id++ {
		name := datfile.EffectAttributeName(int16(id))
		if name == "" {
			continue
		}
		rows = append(rows, FactRow{
			ID:         id,
			Name:       name,
			Meaning:    "Genie object attribute usable by effect commands and unit patch recipes",
			Family:     "attribute",
			Confidence: "source_correlated",
			Sources:    defaultSources("AoE2Kit EffectAttributeName table"),
			SafeUse:    "prefer named attributes in recipes; validate packed attack/armor class deltas separately",
		})
	}
	return rows
}

func resourceRows() []FactRow {
	var rows []FactRow
	for id := int16(0); id <= 500; id++ {
		name := datfile.EffectResourceName(id)
		if name == "" {
			continue
		}
		rows = append(rows, FactRow{
			ID:         int(id),
			Name:       name,
			Meaning:    "player resource/stat slot usable by resource effect commands and replay-side state comparisons",
			Family:     "resource",
			Confidence: "source_correlated",
			Sources:    defaultSources("AoE2Kit EffectResourceName table and replay side-channel work"),
			SafeUse:    "safe for explain/lint; gameplay meaning may still depend on mode and engine path",
		})
	}
	return rows
}

func techModifierRows() []FactRow {
	ids := []int16{-3, -2, -1, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 16384, 16385, 16386, 16387}
	rows := make([]FactRow, 0, len(ids))
	for _, id := range ids {
		rows = append(rows, FactRow{
			ID:         int(id),
			Name:       datfile.EffectTechAttributeName(id),
			Meaning:    "technology field selector used by modify_tech and tech cost/time commands",
			Family:     "tech-modifier",
			Confidence: "source_correlated",
			Sources:    defaultSources("AoE2Kit EffectTechAttributeName table and GoKu tech_modifiers catalog"),
			SafeUse:    "safe when target tech id and command type are explicit",
		})
	}
	return rows
}

func taskRows() []FactRow {
	raw := map[int]string{
		0: "none", 1: "move_to", 2: "follow", 3: "garrison", 4: "explore", 5: "gather_rebuild", 6: "graze", 7: "combat", 8: "shoot", 9: "attack", 10: "fly", 11: "scare_hunt", 12: "unload_boat_like", 13: "guard", 14: "siege_tower_ability", 20: "escape", 21: "make_farm_trap", 101: "build", 102: "make_unit", 103: "make_tech", 104: "convert", 105: "heal", 106: "repair", 107: "get_auto_converted", 108: "discovery_artifact", 110: "hunt", 111: "trade", 120: "generate_wonder_victory", 121: "deselect_when_tasked", 122: "loot_gather", 123: "housing", 124: "pack", 125: "unpack_and_attack", 131: "off_map_trade", 132: "pickup_unit", 133: "speed_charge", 134: "transform_unit", 135: "kidnap_unit", 136: "deposit_unit", 149: "shear", 150: "regeneration", 151: "resource_generation", 152: "movement_damage", 153: "moveable_drop_site", 154: "pillage", 155: "influence_ability",
	}
	rows := make([]FactRow, 0, len(raw))
	for id, name := range raw {
		rows = append(rows, FactRow{
			ID:         id,
			Name:       name,
			Meaning:    "unit task id used in unit headers/tasks",
			Family:     "task",
			Confidence: "source_correlated",
			Sources:    []string{"GoKuModder/aoe2-genie-tooling tasks catalog", "Genie Editor task terminology"},
			SafeUse:    "safe for read/explain; task authoring still needs per-task field validation",
		})
	}
	return rows
}

func scenarioUnitClassRows() []FactRow {
	var rows []FactRow
	for id := 0; id <= 64; id++ {
		name := ScenarioUnitClassName(id)
		if name == "" {
			continue
		}
		confidence := "source_correlated"
		sources := defaultSources("scenario editor object-group ids align with DAT unit class constants minus 900")
		notes := []string{"scenario trigger object_group values use compact editor ids; DAT effect filters use the same class family at 900+id"}
		if id == 40 {
			confidence = "engine_verified"
			sources = append(sources, "DE editor calibration md_t32->md_t34: object_group 40 = Salvage Pile")
			notes = append(notes, "engine-verified anchor for the compact editor enum")
		}
		rows = append(rows, FactRow{
			ID:         id,
			Name:       name,
			Meaning:    "scenario editor Object Group / unit-class filter value",
			Family:     "scenario-trigger-enum",
			Confidence: confidence,
			Sources:    sources,
			SafeUse:    "safe for explaining trigger filters; authoring still needs the matching object_type/source_player constraints",
			Notes:      notes,
		})
	}
	return rows
}

func scenarioObjectTypeRows() []FactRow {
	names := scenarioObjectTypeNames()
	rows := make([]FactRow, 0, len(names))
	for id, name := range names {
		rows = append(rows, FactRow{
			ID:         id,
			Name:       name,
			Meaning:    "scenario editor Object Type dropdown value used by trigger filters",
			Family:     "scenario-trigger-enum",
			Confidence: "engine_verified",
			Sources:    []string{"DE editor calibration md_t32->md_t34: object_type 4 = Military"},
			SafeUse:    "safe for explaining calibrated trigger filters; expand this table only from editor-verified values",
			Notes:      []string{"Do not confuse with XS cObjectType constants, which use a different numeric space."},
		})
	}
	return rows
}

func scenarioObjectTypeNames() map[int]string {
	return map[int]string{
		4: "military",
	}
}

func scenarioUnitAIActionRows() []FactRow {
	names := scenarioUnitAIActionNames()
	rows := make([]FactRow, 0, len(names))
	for id, name := range names {
		rows = append(rows, FactRow{
			ID:         id,
			Name:       name,
			Meaning:    "scenario editor Unit AI Action dropdown value used by object_has_action conditions",
			Family:     "scenario-trigger-enum",
			Confidence: "engine_verified",
			Sources:    []string{"DE editor calibration md_t32->md_t34: unit_ai_action 7 = Stop"},
			SafeUse:    "safe for explaining calibrated object_has_action conditions; not interchangeable with DAT unit task ids",
			Notes:      []string{"DAT task id 7 is combat; this scenario editor enum is a separate table."},
		})
	}
	return rows
}

func scenarioUnitAIActionNames() map[int]string {
	return map[int]string{
		7: "stop",
	}
}
