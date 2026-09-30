package datcodec

import (
	"fmt"
	"math"
	"strings"

	"aoe2kit/pkg/aoe2"
	"aoe2kit/pkg/datfacts"
	"aoe2kit/pkg/datfile"
)

type EffectExplainReport struct {
	Version      string                     `json:"version"`
	EffectID     int                        `json:"effect_id"`
	EffectName   string                     `json:"effect_name"`
	Summary      string                     `json:"summary"`
	CommandCount int                        `json:"command_count"`
	Commands     []EffectCommandExplanation `json:"commands"`
	Warnings     []string                   `json:"warnings,omitempty"`
	Effect       datfile.Effect             `json:"effect"`
	Verification aoe2.VerificationClaim     `json:"verification"`
}

type TechExplainReport struct {
	Version      string                 `json:"version"`
	TechID       int                    `json:"tech_id"`
	TechName     string                 `json:"tech_name"`
	EffectID     int16                  `json:"effect_id"`
	EffectName   string                 `json:"effect_name,omitempty"`
	Summary      string                 `json:"summary"`
	TechLines    []string               `json:"tech_lines,omitempty"`
	Effect       *EffectExplainReport   `json:"effect,omitempty"`
	Warnings     []string               `json:"warnings,omitempty"`
	Tech         datfile.Tech           `json:"tech"`
	Verification aoe2.VerificationClaim `json:"verification"`
}

type EffectCommandExplanation struct {
	Index      int                              `json:"index"`
	Type       uint8                            `json:"type"`
	TypeName   string                           `json:"type_name"`
	Scope      string                           `json:"scope,omitempty"`
	Raw        datfile.EffectCommand            `json:"raw"`
	Summary    string                           `json:"summary"`
	Details    []string                         `json:"details,omitempty"`
	Warnings   []string                         `json:"warnings,omitempty"`
	References []datfile.EffectCommandReference `json:"references,omitempty"`
	Recipe     EffectCommandRecipe              `json:"recipe"`
	Confidence string                           `json:"confidence,omitempty"`
	Source     string                           `json:"source,omitempty"`
}

func ExplainEffect(idx *datfile.Index, effectID int) (EffectExplainReport, error) {
	effect, ok := idx.Effect(effectID)
	if !ok {
		return EffectExplainReport{}, fmt.Errorf("effect %d is outside effects table", effectID)
	}
	report := EffectExplainReport{
		Version:      Version,
		EffectID:     effect.Index,
		EffectName:   effect.Name,
		CommandCount: len(effect.Commands),
		Effect:       effect,
		Verification: datExplainVerification(),
	}
	for _, command := range effect.Commands {
		item := explainEffectCommand(idx, command)
		report.Commands = append(report.Commands, item)
		report.Warnings = append(report.Warnings, item.Warnings...)
	}
	report.Summary = summarizeEffectExplain(report)
	return report, nil
}

func ExplainTech(idx *datfile.Index, techID int) (TechExplainReport, error) {
	tech, ok := idx.Tech(techID)
	if !ok {
		return TechExplainReport{}, fmt.Errorf("tech %d is outside tech table", techID)
	}
	report := TechExplainReport{
		Version:      Version,
		TechID:       tech.Index,
		TechName:     tech.Name,
		EffectID:     tech.EffectID,
		Tech:         tech,
		Verification: datExplainVerification(),
	}
	report.TechLines = explainTechFields(tech)
	if tech.EffectID >= 0 {
		effect, ok := idx.Effect(int(tech.EffectID))
		if ok {
			effectReport, err := ExplainEffect(idx, int(tech.EffectID))
			if err != nil {
				return TechExplainReport{}, err
			}
			report.EffectName = effect.Name
			report.Effect = &effectReport
			report.Warnings = append(report.Warnings, effectReport.Warnings...)
		} else {
			report.Warnings = append(report.Warnings, fmt.Sprintf("tech references missing effect_id %d", tech.EffectID))
		}
	}
	report.Summary = summarizeTechExplain(report)
	return report, nil
}

func datExplainVerification() aoe2.VerificationClaim {
	claim := aoe2.StructureVerification(true)
	claim.Note = "DAT explain output translates decoded tech/effect rows into authoring language; structural interpretation is not by itself an in-engine proof of runtime behavior."
	return claim
}

func explainEffectCommand(idx *datfile.Index, command datfile.EffectCommand) EffectCommandExplanation {
	semantic := command.Semantic
	if semantic == nil {
		semantic = datfile.InterpretEffectCommand(command)
		command.Semantic = semantic
	}
	out := EffectCommandExplanation{
		Index:      command.Index,
		Type:       command.Type,
		TypeName:   semantic.TypeName,
		Scope:      semantic.Scope,
		Raw:        command,
		References: semantic.References,
		Recipe:     effectCommandRecipeFromCommand(command),
		Confidence: semantic.Confidence,
		Source:     semantic.Source,
	}
	out.Summary = explainCommandSummary(idx, command, semantic)
	out.Details = explainCommandDetails(idx, command, semantic)
	out.Warnings = effectCommandWarnings(command, semantic)
	return out
}

func explainCommandSummary(idx *datfile.Index, command datfile.EffectCommand, semantic *datfile.EffectCommandSemantic) string {
	scope := effectScopeText(semantic.Scope)
	unit := namedUnit(idx, int(command.A))
	switch semantic.TypeName {
	case "set_attribute", "team_set_attribute", "enemy_set_attribute", "neutral_set_attribute", "gaia_set_attribute":
		return fmt.Sprintf("set %s%s on %s to %s", scope, attributeText(command, semantic), unit, amountText(command.D))
	case "add_attribute", "team_add_attribute", "enemy_add_attribute", "neutral_add_attribute", "gaia_add_attribute":
		return fmt.Sprintf("add %s to %s%s on %s", amountText(command.D), scope, attributeText(command, semantic), unit)
	case "attribute_multiplier", "team_attribute_multiplier", "enemy_attribute_multiplier", "neutral_attribute_multiplier", "gaia_attribute_multiplier":
		return fmt.Sprintf("multiply %s%s on %s by %s", scope, attributeText(command, semantic), unit, amountText(command.D))
	case "set_local_building_attribute":
		return fmt.Sprintf("set local %s %s to %s", unit, attributeText(command, semantic), amountText(command.D))
	case "add_local_building_attribute", "add_local_building_attribute_advanced":
		return fmt.Sprintf("add %s to local %s %s", amountText(command.D), unit, attributeText(command, semantic))
	case "multiply_local_building_attribute":
		return fmt.Sprintf("multiply local %s %s by %s", unit, attributeText(command, semantic), amountText(command.D))
	case "resource_modifier", "team_resource_modifier", "enemy_resource_modifier", "neutral_resource_modifier", "gaia_resource_modifier":
		return fmt.Sprintf("%sresource %s: %s %s", scope, resourceText(semantic), operationText(semantic), amountText(command.D))
	case "resource_multiplier", "team_resource_multiplier", "enemy_resource_multiplier", "neutral_resource_multiplier", "gaia_resource_multiplier":
		return fmt.Sprintf("multiply %sresource %s by %s", scope, resourceText(semantic), amountText(command.D))
	case "enable_unit", "team_enable_unit", "enemy_enable_unit", "neutral_enable_unit", "gaia_enable_unit":
		return fmt.Sprintf("enable %sunit %s", scope, unit)
	case "upgrade_unit", "team_upgrade_unit", "enemy_upgrade_unit", "neutral_upgrade_unit", "gaia_upgrade_unit":
		return fmt.Sprintf("upgrade %s%s to %s", scope, unit, namedUnit(idx, int(command.B)))
	case "spawn_unit", "team_spawn_unit", "enemy_spawn_unit", "neutral_spawn_unit", "gaia_spawn_unit":
		return fmt.Sprintf("spawn %s%s at %s amount %s", scope, unit, namedUnit(idx, int(command.B)), amountText(command.D))
	case "modify_tech":
		return fmt.Sprintf("modify tech %s: %s = %s", namedTech(idx, int(command.A)), techAttributeText(semantic), amountText(command.D))
	case "team_modify_tech", "enemy_modify_tech", "neutral_modify_tech", "gaia_modify_tech":
		return fmt.Sprintf("modify %stech %s: %s = %s", scope, namedTech(idx, int(command.A)), techAttributeText(semantic), amountText(command.D))
	case "set_tech_cost", "tech_cost_modifier":
		return fmt.Sprintf("%s tech %s cost resource %s by %s", strings.TrimSuffix(strings.TrimPrefix(semantic.TypeName, "tech_"), "_modifier"), namedTech(idx, int(command.A)), resourceText(semantic), amountText(command.D))
	case "disable_tech":
		return fmt.Sprintf("disable tech %s", namedTech(idx, int(command.D)))
	case "tech_time_modifier":
		return fmt.Sprintf("modify tech %s time: %s %s", namedTech(idx, int(command.A)), operationText(semantic), amountText(command.D))
	default:
		return fmt.Sprintf("raw command type %d/%s a=%d b=%d c=%d d=%s", command.Type, semantic.TypeName, command.A, command.B, command.C, amountText(command.D))
	}
}

func explainCommandDetails(idx *datfile.Index, command datfile.EffectCommand, semantic *datfile.EffectCommandSemantic) []string {
	var out []string
	if semantic.Scope == "local_building" {
		out = append(out, "targets the specific local building instance that researched the containing type-32 technology")
	}
	if semantic.UnitClassName != "" && command.B != -1 && strings.Contains(semantic.TypeName, "attribute") {
		out = append(out, fmt.Sprintf("unit class filter: %d/%s", command.B, semantic.UnitClassName))
	}
	if semantic.PackedTypeID != nil && semantic.PackedAmount != nil {
		name := semantic.PackedTypeName
		if name == "" {
			name = fmt.Sprintf("class_%d", *semantic.PackedTypeID)
		}
		out = append(out, fmt.Sprintf("packed attack/armor delta: class=%d/%s amount=%+d raw=%s", *semantic.PackedTypeID, name, *semantic.PackedAmount, amountText(command.D)))
	}
	for _, ref := range semantic.References {
		switch ref.Kind {
		case "unit":
			out = append(out, fmt.Sprintf("unit reference %s: %s", ref.Field, namedUnit(idx, ref.ID)))
		case "tech":
			out = append(out, fmt.Sprintf("tech reference %s: %s", ref.Field, namedTech(idx, ref.ID)))
		}
	}
	return out
}

func effectCommandWarnings(command datfile.EffectCommand, semantic *datfile.EffectCommandSemantic) []string {
	var warnings []string
	if factWarning := effectCommandFactWarning(command.Type); factWarning != "" {
		warnings = append(warnings, factWarning)
	}
	switch command.Type {
	case 3, 13, 23, 33, 43:
		warnings = append(warnings, "engine fact: upgrade_unit did not transform units in the forced scenario-research fixture; verify UI or runtime path before relying on it")
	case 7, 17, 27, 37, 47:
		warnings = append(warnings, "engine fact: spawn_unit did not add units in the forced scenario-research fixture; verify runtime path before relying on it")
	}
	if semantic.TypeName == "unknown" {
		warnings = append(warnings, "unknown effect command type; Kit preserves raw operands but cannot explain runtime behavior")
	}
	return warnings
}

func effectCommandFactWarning(commandType uint8) string {
	row, err := datfacts.ExplainEffectCommandType(int(commandType))
	if err != nil {
		return ""
	}
	needsWarning := false
	switch row.Confidence {
	case "engine_verified", "source_verified_for_200_201", "source_correlated", "typed_by_kit_table":
		needsWarning = len(row.Disagreements) > 0
	default:
		needsWarning = true
	}
	if !needsWarning {
		return ""
	}
	var parts []string
	parts = append(parts, fmt.Sprintf("DAT fact: effect command %d/%s confidence=%s", row.ID, row.Name, row.Confidence))
	if row.SafeUse != "" {
		parts = append(parts, "safe_use="+row.SafeUse)
	}
	if len(row.Disagreements) > 0 {
		parts = append(parts, "known_disagreement="+strings.Join(row.Disagreements, " | "))
	}
	if commandType == 202 || commandType == 204 {
		parts = append(parts, "local-building 202/204 semantics remain facts-led, not engine_verified")
	}
	return strings.Join(parts, "; ")
}

func RecipeAuthoringWarnings(recipe Recipe) []string {
	seen := map[string]bool{}
	var out []string
	addCommand := func(commandRecipe EffectCommandRecipe) {
		command, err := commandFromRecipe(commandRecipe)
		if err != nil {
			warning := fmt.Sprintf("could not inspect effect command recipe %q before write: %v", commandRecipe.Kind, err)
			if !seen[warning] {
				seen[warning] = true
				out = append(out, warning)
			}
			return
		}
		semantic := command.Semantic
		if semantic == nil {
			semantic = datfile.InterpretEffectCommand(command)
		}
		for _, warning := range effectCommandWarnings(command, semantic) {
			if !seen[warning] {
				seen[warning] = true
				out = append(out, warning)
			}
		}
	}
	addEffectCreate := func(item EffectCreateRecipe) {
		for _, command := range item.Commands {
			addCommand(command)
		}
		for _, command := range item.AppendCommands {
			addCommand(command)
		}
	}
	addEffectPatch := func(item EffectPatchRecipe) {
		if item.Commands != nil {
			for _, command := range *item.Commands {
				addCommand(command)
			}
		}
		for _, command := range item.AppendCommands {
			addCommand(command)
		}
	}
	addAbilityCreate := func(item AbilityCreateRecipe) {
		for _, command := range item.Commands {
			addCommand(command)
		}
		for _, command := range item.AppendCommands {
			addCommand(command)
		}
	}
	if recipe.CreateEffect != nil {
		addEffectCreate(*recipe.CreateEffect)
	}
	for _, item := range recipe.CreateEffects {
		addEffectCreate(item)
	}
	for _, item := range recipe.Effects {
		addEffectPatch(item)
	}
	if recipe.CreateAbility != nil {
		addAbilityCreate(*recipe.CreateAbility)
	}
	for _, item := range recipe.CreateAbilities {
		addAbilityCreate(item)
	}
	return out
}

func explainTechFields(tech datfile.Tech) []string {
	var out []string
	out = append(out, fmt.Sprintf("civ=%d type=%d repeatable=%d icon=%d", tech.Civ, tech.Type, tech.Repeatable, tech.IconID))
	out = append(out, fmt.Sprintf("required_tech_count=%d required_techs=%v", tech.RequiredTechCount, tech.RequiredTechs))
	if len(tech.ResourceCosts) > 0 {
		var parts []string
		for _, cost := range tech.ResourceCosts {
			parts = append(parts, fmt.Sprintf("%s=%d flag=%d", resourceNameOrID(cost.Type), cost.Amount, cost.Flag))
		}
		out = append(out, "resource costs: "+strings.Join(parts, ", "))
	}
	if len(tech.ResearchLocations) > 0 {
		var parts []string
		for _, loc := range tech.ResearchLocations {
			parts = append(parts, fmt.Sprintf("location=%d time=%d button=%d hotkey=%d", loc.LocationID, loc.ResearchTime, loc.ButtonID, loc.HotKeyID))
		}
		out = append(out, "research locations: "+strings.Join(parts, "; "))
	}
	return out
}

func summarizeEffectExplain(report EffectExplainReport) string {
	if report.CommandCount == 0 {
		return "effect has no commands"
	}
	local := 0
	unknown := 0
	for _, command := range report.Commands {
		if command.Scope == "local_building" {
			local++
		}
		if command.TypeName == "unknown" {
			unknown++
		}
	}
	parts := []string{fmt.Sprintf("%d command(s)", report.CommandCount)}
	if local > 0 {
		parts = append(parts, fmt.Sprintf("%d local-building command(s)", local))
	}
	if unknown > 0 {
		parts = append(parts, fmt.Sprintf("%d unknown command(s)", unknown))
	}
	return strings.Join(parts, ", ")
}

func summarizeTechExplain(report TechExplainReport) string {
	if report.Effect == nil {
		return fmt.Sprintf("tech references effect_id %d but no decoded effect is available", report.EffectID)
	}
	return fmt.Sprintf("tech %s invokes effect %d/%s: %s", report.TechName, report.EffectID, report.EffectName, report.Effect.Summary)
}

func attributeText(command datfile.EffectCommand, semantic *datfile.EffectCommandSemantic) string {
	name := semantic.AttributeName
	if name == "" {
		name = fmt.Sprintf("attribute_%d", command.C)
	}
	if semantic.PackedTypeID != nil && semantic.PackedAmount != nil {
		packedName := semantic.PackedTypeName
		if packedName == "" {
			packedName = fmt.Sprintf("class_%d", *semantic.PackedTypeID)
		}
		return fmt.Sprintf("%s[%s %+d]", name, packedName, *semantic.PackedAmount)
	}
	return name
}

func effectScopeText(scope string) string {
	if scope == "" {
		return ""
	}
	return strings.ReplaceAll(scope, "_", " ") + " "
}

func resourceText(semantic *datfile.EffectCommandSemantic) string {
	if semantic.ResourceName != "" {
		return semantic.ResourceName
	}
	if semantic.ResourceID != nil {
		return fmt.Sprintf("resource_%d", *semantic.ResourceID)
	}
	return "resource"
}

func operationText(semantic *datfile.EffectCommandSemantic) string {
	if semantic.OperationName != "" {
		return semantic.OperationName
	}
	if semantic.OperationID != nil {
		return fmt.Sprintf("operation_%d", *semantic.OperationID)
	}
	return "operation"
}

func techAttributeText(semantic *datfile.EffectCommandSemantic) string {
	if semantic.TechAttributeName != "" {
		return semantic.TechAttributeName
	}
	if semantic.TechAttributeID != nil {
		return fmt.Sprintf("tech_attribute_%d", *semantic.TechAttributeID)
	}
	return "tech_attribute"
}

func namedUnit(idx *datfile.Index, id int) string {
	if id < 0 {
		return fmt.Sprintf("%d", id)
	}
	name := ""
	for _, civ := range idx.Civs {
		if id >= 0 && id < len(civ.Units) && civ.Units[id].Present {
			name = civ.Units[id].Name
			break
		}
	}
	if name == "" {
		return fmt.Sprintf("%d", id)
	}
	return fmt.Sprintf("%d/%s", id, name)
}

func namedTech(idx *datfile.Index, id int) string {
	if id < 0 || id >= len(idx.Techs) {
		return fmt.Sprintf("%d", id)
	}
	name := idx.Techs[id].Name
	if name == "" {
		return fmt.Sprintf("%d", id)
	}
	return fmt.Sprintf("%d/%s", id, name)
}

func resourceNameOrID(id int16) string {
	name := datfile.EffectResourceName(id)
	if name == "" {
		return fmt.Sprintf("resource_%d", id)
	}
	return name
}

func amountText(amount float32) string {
	if math.Trunc(float64(amount)) == float64(amount) {
		return fmt.Sprintf("%.0f", amount)
	}
	return fmt.Sprintf("%g", amount)
}
