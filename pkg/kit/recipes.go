package kit

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"aoe2kit/pkg/datcodec"
	"aoe2kit/pkg/datfile"
	"aoe2kit/pkg/scenario"
)

type RecipeTemplate struct {
	Name           string           `json:"name"`
	Domain         string           `json:"domain"`
	Summary        string           `json:"summary"`
	AppliesTo      string           `json:"applies_to"`
	Command        string           `json:"command"`
	Verification   string           `json:"verification"`
	Notes          []string         `json:"notes,omitempty"`
	ScenarioRecipe *scenario.Recipe `json:"scenario_recipe,omitempty"`
	DatRecipe      *datcodec.Recipe `json:"dat_recipe,omitempty"`
}

type RecipeTemplateSummary struct {
	Name         string   `json:"name"`
	Domain       string   `json:"domain"`
	Summary      string   `json:"summary"`
	AppliesTo    string   `json:"applies_to"`
	Command      string   `json:"command"`
	Verification string   `json:"verification"`
	Notes        []string `json:"notes,omitempty"`
}

type RecipeTemplateListReport struct {
	Verification string                  `json:"verification"`
	Domain       string                  `json:"domain,omitempty"`
	Count        int                     `json:"count"`
	Templates    []RecipeTemplateSummary `json:"templates"`
}

type RecipeTemplateExportReport struct {
	Verification string                     `json:"verification"`
	OutputDir    string                     `json:"output_dir"`
	Domain       string                     `json:"domain,omitempty"`
	Count        int                        `json:"count"`
	Files        []RecipeTemplateExportFile `json:"files"`
}

type RecipeTemplateExportFile struct {
	Name        string `json:"name"`
	Domain      string `json:"domain"`
	Path        string `json:"path"`
	Bytes       int    `json:"bytes"`
	Overwritten bool   `json:"overwritten,omitempty"`
}

func RecipeTemplates() []RecipeTemplate {
	enabled := true
	disabled := false
	timer5 := 5
	timer120 := 120
	displayTime := 8
	sourcePlayer := 1
	unitConst := 83
	x := 10
	y := 10
	terrainID := 2
	borderThickness := 1
	unitX := 12.5
	unitY := 12.5
	unitStatus := 2
	unitRotation := 0.0
	unitReference := 900001
	touchTimestamp := 1800000000
	allCivs := true
	graphicName := "A2K Example Graphic Clone"
	particleName := "example_particle_effect"
	soundProbability := int16(0)
	colourBase := int32(0x00FFFFFF)
	exampleEffectName := "A2K Example Effect"
	exampleTechName := "A2K Example Ability"
	exampleResourceAmount := float32(100)
	exampleAttributeAmount := float32(5)
	exampleMultiplierAmount := float32(1.25)
	exampleTrainTime := int16(5)
	exampleTrainButton := uint8(4)
	exampleTrainHotkey := int32(16123)
	exampleTechID := 101
	exampleFromEffect := 0
	exampleResearchTime := int16(0)
	exampleResearchButton := uint8(5)
	exampleResearchHotkey := int32(16124)

	templates := []RecipeTemplate{
		{
			Name:         "scen.xs-attachment",
			Domain:       "scenario",
			Summary:      "Attach an XS script as the executable scenario source DE opens at runtime.",
			AppliesTo:    "Current DE .aoe2scenario files that need in-scenario XS without external source drift.",
			Command:      "kit scen patch input.aoe2scenario output.aoe2scenario --recipe xs-attachment.json",
			Verification: "structure_verified_not_engine_verified",
			Notes: []string{
				"Set content_file to the XS source beside the recipe before applying.",
				"Run kit scen xs deploy --check afterward to materialize and verify the runtime resources/_common/xs copy.",
			},
			ScenarioRecipe: &scenario.Recipe{
				XS: &scenario.XSRecipe{
					Mode:        "attachment",
					Name:        "script.xs",
					ContentFile: "script.xs",
				},
			},
		},
		{
			Name:         "scen.xs-attachment-and-carrier",
			Domain:       "scenario",
			Summary:      "Attach executable XS and also carry a disabled source-escrow trigger for inspection.",
			AppliesTo:    "Diagnostics and handoff scenarios where the runtime script and readable embedded source should travel together.",
			Command:      "kit scen patch input.aoe2scenario output.aoe2scenario --recipe xs-attachment-and-carrier.json",
			Verification: "structure_verified_not_engine_verified",
			Notes: []string{
				"Attachment fields are the engine source of truth; the carrier is redundant escrow for tools and humans.",
				"Use kit scen xs compare to prove the scenario-carried source still matches the local source file.",
			},
			ScenarioRecipe: &scenario.Recipe{
				XS: &scenario.XSRecipe{
					Mode:               "attachment_and_carrier",
					Name:               "script.xs",
					CarrierTitle:       "script.xs",
					CarrierTriggerName: "A2K XS Source Escrow",
					ContentFile:        "script.xs",
				},
			},
		},
		{
			Name:         "scen.xs-carrier",
			Domain:       "scenario",
			Summary:      "Embed an XS script as parser-style source escrow in a disabled carrier trigger.",
			AppliesTo:    "Current DE .aoe2scenario files with trigger support.",
			Command:      "kit scen patch input.aoe2scenario output.aoe2scenario --recipe xs-carrier.json",
			Verification: "structure_verified_not_engine_verified",
			Notes: []string{
				"Set content_file to the XS source beside the recipe before applying.",
				"Carrier-only mode is not the engine runtime source; use xs-attachment when DE should execute the XS.",
				"Run kit scen xs compare afterward to prove the embedded payload matches the source file.",
			},
			ScenarioRecipe: &scenario.Recipe{
				XS: &scenario.XSRecipe{
					Mode:               "carrier",
					CarrierTitle:       "script.xs",
					CarrierTriggerName: "A2K XS Carrier",
					ContentFile:        "script.xs",
				},
			},
		},
		{
			Name:         "scen.timestamp-touch",
			Domain:       "scenario",
			Summary:      "Set the scenario internal save timestamp explicitly.",
			AppliesTo:    "Deterministic fixtures or generated scenarios that need a pinned browser date.",
			Command:      "kit scen patch input.aoe2scenario output.aoe2scenario --recipe timestamp-touch.json",
			Verification: "structure_verified_not_engine_verified",
			Notes:        []string{"Most patches refresh the timestamp automatically; use this only when a pinned Unix timestamp is intentional."},
			ScenarioRecipe: &scenario.Recipe{
				Scenario: &scenario.ScenarioRecipe{
					TimestampOfLastSave: &touchTimestamp,
				},
			},
		},
		{
			Name:         "scen.timer-display-chat",
			Domain:       "scenario",
			Summary:      "Add one timer-gated trigger that displays panel text and sends a chat line.",
			AppliesTo:    "Smoke tests, diagnostics, and first writable-trigger checks.",
			Command:      "kit scen patch input.aoe2scenario output.aoe2scenario --recipe timer-display-chat.json",
			Verification: "structure_verified_not_engine_verified",
			Notes:        []string{"Trigger is enabled and non-looping; adjust text and timer before using in a real design."},
			ScenarioRecipe: &scenario.Recipe{
				Triggers: []scenario.TriggerRecipe{{
					Op:      "add_trigger",
					Name:    "A2K Template - timer display and chat",
					Enabled: &enabled,
					Looping: &disabled,
					Conditions: []scenario.ConditionRecipe{
						{Op: "timer", Timer: &timer5},
					},
					Effects: []scenario.EffectRecipe{
						{Op: "display_instructions", SourcePlayer: &sourcePlayer, Message: "A2K TEMPLATE: display instruction.", DisplayTime: &displayTime},
						{Op: "send_chat", SourcePlayer: &sourcePlayer, Message: "A2K TEMPLATE: chat line."},
					},
				}},
			},
		},
		{
			Name:         "scen.timer-declare-victory",
			Domain:       "scenario",
			Summary:      "Add one timer-gated trigger that declares victory for a chosen player.",
			AppliesTo:    "Diagnostic scenarios that should close themselves after a fixed readback window.",
			Command:      "kit scen patch input.aoe2scenario output.aoe2scenario --recipe timer-declare-victory.json",
			Verification: "structure_verified_not_engine_verified",
			Notes: []string{
				"Template fires at 120 seconds for player 1; adjust timer and source_player before a real run.",
				"Pair with scen.no-conquest-victory for diagnostics where ordinary conquest should not decide the ending first.",
			},
			ScenarioRecipe: &scenario.Recipe{
				Triggers: []scenario.TriggerRecipe{{
					Op:      "add_trigger",
					Name:    "A2K Template - timer declare victory",
					Enabled: &enabled,
					Looping: &disabled,
					Conditions: []scenario.ConditionRecipe{
						{Op: "timer", Timer: &timer120},
					},
					Effects: []scenario.EffectRecipe{{
						Op:           "declare_victory",
						SourcePlayer: &sourcePlayer,
					}},
				}},
			},
		},
		{
			Name:         "scen.no-conquest-victory",
			Domain:       "scenario",
			Summary:      "Disable ordinary conquest as the scenario victory requirement.",
			AppliesTo:    "Diagnostics, sandboxes, and authored tests whose ending is controlled by triggers.",
			Command:      "kit scen patch input.aoe2scenario output.aoe2scenario --recipe no-conquest-victory.json",
			Verification: "structure_verified_not_engine_verified",
			Notes: []string{
				"This edits the scenario GlobalVictory conquest field only; it is not a lobby setting.",
				"Use this when active inert players, Gaia-owned dummies, or controlled kill rigs would otherwise end early.",
			},
			ScenarioRecipe: &scenario.Recipe{
				Victory: &scenario.VictoryRecipe{
					ConquestRequired: recipeIntPtr(0),
				},
			},
		},
		{
			Name:         "scen.create-object-trigger",
			Domain:       "scenario",
			Summary:      "Add a trigger that creates one unit at a known tile.",
			AppliesTo:    "Spawn diagnostics and scenario trigger authoring smoke tests.",
			Command:      "kit scen patch input.aoe2scenario output.aoe2scenario --recipe create-object-trigger.json",
			Verification: "structure_verified_not_engine_verified",
			Notes:        []string{"Unit id 83 is an example; confirm the desired unit id against the active data set."},
			ScenarioRecipe: &scenario.Recipe{
				Triggers: []scenario.TriggerRecipe{{
					Op:      "add_trigger",
					Name:    "A2K Template - create object",
					Enabled: &enabled,
					Looping: &disabled,
					Effects: []scenario.EffectRecipe{{
						Op:               "create_object",
						SourcePlayer:     &sourcePlayer,
						ObjectListUnitID: &unitConst,
						LocationX:        &x,
						LocationY:        &y,
					}},
				}},
			},
		},
		{
			Name:         "scen.caption-marker-unit",
			Domain:       "scenario",
			Summary:      "Place one captioned marker unit with a stable reference id.",
			AppliesTo:    "Authoring anchors, diagnostic labels, and delete-by-caption workflows.",
			Command:      "kit scen patch input.aoe2scenario output.aoe2scenario --recipe caption-marker-unit.json",
			Verification: "structure_verified_not_engine_verified",
			Notes:        []string{"Unit id 598 is an outpost marker convention; inspect the target data set if a mod changes unit ids."},
			ScenarioRecipe: &scenario.Recipe{
				Units: []scenario.UnitRecipe{{
					Op:              "add_unit",
					Player:          1,
					UnitConst:       598,
					X:               &unitX,
					Y:               &unitY,
					ReferenceID:     &unitReference,
					Status:          &unitStatus,
					Rotation:        &unitRotation,
					CaptionString:   "A2K TEMPLATE MARKER",
					CaptionStringID: recipeIntPtr(-1),
				}},
			},
		},
		{
			Name:         "scen.terrain-border",
			Domain:       "scenario",
			Summary:      "Stamp a rectangular terrain border around a test area.",
			AppliesTo:    "Readable diagnostic arenas and visual region boundaries.",
			Command:      "kit scen patch input.aoe2scenario output.aoe2scenario --recipe terrain-border.json",
			Verification: "structure_verified_not_engine_verified",
			Notes:        []string{"Terrain id 2 is only an example; pick terrain deliberately for the target map."},
			ScenarioRecipe: &scenario.Recipe{
				Map: []scenario.MapRecipe{{
					Op:        "set_terrain_border",
					X1:        8,
					Y1:        8,
					X2:        20,
					Y2:        20,
					Thickness: &borderThickness,
					TerrainID: &terrainID,
				}},
			},
		},
		{
			Name:         "dat.ability-create-command",
			Domain:       "dat",
			Summary:      "Create a paired Tech+Effect ability from a template tech and explicit command row.",
			AppliesTo:    "Custom command-button abilities, spell buttons, and train-to-cast systems.",
			Command:      "kit dat codec-patch empires2_x2_p1.dat mod.dat --recipe ability-create-command.json",
			Verification: "structure_verified_not_engine_verified",
			Notes: []string{
				"Template tech/effect ids are placeholders; inspect abilities/effects before choosing the source rows.",
				"Research locations control where the button appears; pair this with unit train-location/button patches when needed.",
			},
			DatRecipe: &datcodec.Recipe{
				CreateAbility: &datcodec.AbilityCreateRecipe{
					Name:       exampleTechName,
					EffectName: exampleEffectName,
					FromTech:   exampleTechID,
					FromEffect: &exampleFromEffect,
					Commands: []datcodec.EffectCommandRecipe{{
						Kind:         "resource_modifier",
						ResourceName: "gold",
						Operation:    "add",
						Amount:       &exampleResourceAmount,
					}},
					ResearchLocations: []datfile.ResearchLocation{{
						LocationID:   12,
						ResearchTime: exampleResearchTime,
						ButtonID:     exampleResearchButton,
						HotKeyID:     exampleResearchHotkey,
					}},
				},
			},
		},
		{
			Name:         "dat.ability-disable-semantic",
			Domain:       "dat",
			Summary:      "Disable an ability by clearing its paired effect commands while preserving stable IDs.",
			AppliesTo:    "Removing a command-button behavior without renumbering tech/effect tables.",
			Command:      "kit dat codec-patch empires2_x2_p1.dat mod.dat --recipe ability-disable-semantic.json",
			Verification: "structure_verified_not_engine_verified",
			Notes:        []string{"This is semantic deletion: menus/prerequisites may still mention the tech until separately patched."},
			DatRecipe: &datcodec.Recipe{
				DisableAbilities: []int{exampleTechID},
			},
		},
		{
			Name:         "dat.effect-disable-semantic",
			Domain:       "dat",
			Summary:      "Disable one effect by clearing its command list while preserving the effect ID.",
			AppliesTo:    "Safe removal of effect behavior when techs or abilities still reference the effect row.",
			Command:      "kit dat codec-patch empires2_x2_p1.dat mod.dat --recipe effect-disable-semantic.json",
			Verification: "structure_verified_not_engine_verified",
			Notes:        []string{"This mirrors kit dat effect-disable; inspect references if the effect is part of a larger ability chain."},
			DatRecipe: &datcodec.Recipe{
				DisableEffects: []int{0},
			},
		},
		{
			Name:         "dat.effect-resource-grant",
			Domain:       "dat",
			Summary:      "Create an effect command that adds a resource amount.",
			AppliesTo:    "Reward buttons, scenario techs, shop purchases, and diagnostics that need simple resource deltas.",
			Command:      "kit dat codec-patch empires2_x2_p1.dat mod.dat --recipe effect-resource-grant.json",
			Verification: "structure_verified_not_engine_verified",
			Notes:        []string{"Uses named resource/operation operands so the recipe stays readable; change gold/add/100 to the intended values."},
			DatRecipe: &datcodec.Recipe{
				CreateEffect: &datcodec.EffectCreateRecipe{
					Name: exampleEffectName,
					Commands: []datcodec.EffectCommandRecipe{{
						Kind:         "resource_modifier",
						ResourceName: "gold",
						Operation:    "add",
						Amount:       &exampleResourceAmount,
					}},
				},
			},
		},
		{
			Name:         "dat.effect-tech-cost-time-adjust",
			Domain:       "dat",
			Summary:      "Create effect commands that set a tech cost and modify its research time.",
			AppliesTo:    "Custom upgrades, shop techs, and balance experiments that tune affordability and timing together.",
			Command:      "kit dat codec-patch empires2_x2_p1.dat mod.dat --recipe effect-tech-cost-time-adjust.json",
			Verification: "structure_verified_not_engine_verified",
			Notes:        []string{"Template targets tech id 101; inspect tech ids and cost resources before use."},
			DatRecipe: &datcodec.Recipe{
				CreateEffect: &datcodec.EffectCreateRecipe{
					Name: exampleEffectName,
					Commands: []datcodec.EffectCommandRecipe{
						{
							Kind:         "set_tech_cost",
							TechID:       &exampleTechID,
							ResourceName: "gold",
							Amount:       &exampleResourceAmount,
						},
						{
							Kind:      "tech_time_modifier",
							TechID:    &exampleTechID,
							Operation: "set",
							Amount:    &exampleResourceAmount,
						},
					},
				},
			},
		},
		{
			Name:         "dat.effect-unit-attribute-buff",
			Domain:       "dat",
			Summary:      "Create an effect command that modifies one unit attribute.",
			AppliesTo:    "Spell effects, civilization bonuses, shop upgrades, and command-button buffs.",
			Command:      "kit dat codec-patch empires2_x2_p1.dat mod.dat --recipe effect-unit-attribute-buff.json",
			Verification: "structure_verified_not_engine_verified",
			Notes:        []string{"Template adds attack to unit id 83; inspect unit ids and attribute names before applying to a real mod."},
			DatRecipe: &datcodec.Recipe{
				CreateEffect: &datcodec.EffectCreateRecipe{
					Name: exampleEffectName,
					Commands: []datcodec.EffectCommandRecipe{{
						Kind:          "add_unit_attribute",
						UnitID:        int16Ptr(83),
						AttributeName: "attack",
						Amount:        &exampleAttributeAmount,
					}},
				},
			},
		},
		{
			Name:         "dat.effect-unit-speed-multiplier",
			Domain:       "dat",
			Summary:      "Create an effect command that multiplies one unit attribute.",
			AppliesTo:    "Temporary-feeling abilities, upgrades, movement tweaks, and balance experiments.",
			Command:      "kit dat codec-patch empires2_x2_p1.dat mod.dat --recipe effect-unit-speed-multiplier.json",
			Verification: "structure_verified_not_engine_verified",
			Notes:        []string{"Template multiplies movement_speed on unit id 83 by 1.25; engine behavior still needs in-game verification."},
			DatRecipe: &datcodec.Recipe{
				CreateEffect: &datcodec.EffectCreateRecipe{
					Name: exampleEffectName,
					Commands: []datcodec.EffectCommandRecipe{{
						Kind:          "multiply_unit_attribute",
						UnitID:        int16Ptr(83),
						AttributeName: "movement_speed",
						Amount:        &exampleMultiplierAmount,
					}},
				},
			},
		},
		{
			Name:         "dat.graphic-clone-particle-bind",
			Domain:       "dat",
			Summary:      "Clone a graphic row and bind a particle_effect_name.",
			AppliesTo:    "Particle experiments and custom visual-effect rows.",
			Command:      "kit dat codec-patch empires2_x2_p1.dat mod.dat --recipe graphic-clone-particle-bind.json",
			Verification: "structure_verified_not_engine_verified",
			Notes:        []string{"The source graphic id is an example; select a real template row before applying."},
			DatRecipe: &datcodec.Recipe{
				CreateGraphic: &datfile.GraphicCreateRecipe{
					From:               0,
					Name:               &graphicName,
					ParticleEffectName: &particleName,
				},
			},
		},
		{
			Name:         "dat.unit-clone-all-civs",
			Domain:       "dat",
			Summary:      "Clone one unit row across every civilization.",
			AppliesTo:    "New generic command units, tokens, shop buttons, and editor-visible prototypes.",
			Command:      "kit dat codec-patch empires2_x2_p1.dat mod.dat --recipe unit-clone-all-civs.json",
			Verification: "structure_verified_not_engine_verified",
			Notes:        []string{"Template uses civ 1 unit 83 as placeholders; choose a stable template unit before applying."},
			DatRecipe: &datcodec.Recipe{
				CreateUnit: &datfile.UnitCreateRecipe{
					FromCivID:  1,
					FromUnitID: 83,
					AllCivs:    &allCivs,
				},
			},
		},
		{
			Name:         "dat.unit-disable-all-civs",
			Domain:       "dat",
			Summary:      "Semantically disable a unit id across all civilizations without compacting unit tables.",
			AppliesTo:    "Retiring a unit from availability while preserving stable unit IDs for scenarios and references.",
			Command:      "kit dat codec-patch empires2_x2_p1.dat mod.dat --recipe unit-disable-all-civs.json",
			Verification: "structure_verified_not_engine_verified",
			Notes:        []string{"Use kit dat delete-plan unit <id> --all-civs first when you need reference cleanup advice too."},
			DatRecipe: &datcodec.Recipe{
				DeleteUnits: []datcodec.UnitDeleteRecipe{{UnitID: 83, AllCivs: true}},
			},
		},
		{
			Name:         "dat.unit-enable-existing-civ",
			Domain:       "dat",
			Summary:      "Enable one existing unit id for one civilization.",
			AppliesTo:    "Small-civ balance tweaks and focused diagnostics where all-civ changes are too broad.",
			Command:      "kit dat codec-patch empires2_x2_p1.dat mod.dat --recipe unit-enable-existing-civ.json",
			Verification: "structure_verified_not_engine_verified",
			Notes:        []string{"This flips DAT availability only; production button, train location, costs, and prerequisites are separate fields."},
			DatRecipe: &datcodec.Recipe{
				UnitAvailability: []datcodec.UnitAvailabilityRecipe{{CivID: intPtr(1), UnitID: 83, Enabled: true}},
			},
		},
		{
			Name:         "dat.unit-availability-all-civs",
			Domain:       "dat",
			Summary:      "Enable one existing unit id for every civilization.",
			AppliesTo:    "Making custom or hidden units trainable/available under all civs.",
			Command:      "kit dat codec-patch empires2_x2_p1.dat mod.dat --recipe unit-availability-all-civs.json",
			Verification: "structure_verified_not_engine_verified",
			Notes:        []string{"This changes availability, not button location, costs, train location, or prerequisites."},
			DatRecipe: &datcodec.Recipe{
				UnitAvailability: []datcodec.UnitAvailabilityRecipe{{UnitID: 83, AllCivs: true, Enabled: true}},
			},
		},
		{
			Name:         "dat.unit-train-button",
			Domain:       "dat",
			Summary:      "Patch the first train-location row for a unit, including time, trained unit, button, and hotkey.",
			AppliesTo:    "Custom production buildings, shop buildings, and train-to-buy command surfaces.",
			Command:      "kit dat codec-patch empires2_x2_p1.dat mod.dat --recipe unit-train-button.json",
			Verification: "structure_verified_not_engine_verified",
			Notes: []string{
				"Template patches civ 1 unit 12 row 0; inspect the target unit's creatable train_locations before applying.",
				"Pair with attribute 43/Button Location or other UI fields when the production button must appear in a specific slot.",
			},
			DatRecipe: &datcodec.Recipe{
				Units: []datfile.UnitRecipePatch{{
					CivID:  1,
					UnitID: 12,
					TrainLocations: []datfile.TrainLocationPatch{{
						Index:       0,
						TrainTime:   &exampleTrainTime,
						TrainUnitID: int16Ptr(83),
						TrainButton: &exampleTrainButton,
						TrainHotkey: &exampleTrainHotkey,
					}},
				}},
			},
		},
		{
			Name:         "dat.sound-item-remove",
			Domain:       "dat",
			Summary:      "Remove one item row from a sound record.",
			AppliesTo:    "Audio cleanup where one file/variant should disappear but the sound ID remains.",
			Command:      "kit dat codec-patch empires2_x2_p1.dat mod.dat --recipe sound-item-remove.json",
			Verification: "structure_verified_not_engine_verified",
			Notes:        []string{"Row indices shift after removal; inspect kit dat sound before and after applying."},
			DatRecipe: &datcodec.Recipe{
				Sounds: []datcodec.SoundPatchRecipe{{ID: 1, RemoveItems: []int{0}}},
			},
		},
		{
			Name:         "dat.sound-mute",
			Domain:       "dat",
			Summary:      "Mute one sound row by setting total probability to zero.",
			AppliesTo:    "Silencing unwanted stock audio without deleting the row.",
			Command:      "kit dat codec-patch empires2_x2_p1.dat mod.dat --recipe sound-mute.json",
			Verification: "structure_verified_not_engine_verified",
			Notes:        []string{"Sound id 0 is a placeholder; inspect sounds and choose the exact target row."},
			DatRecipe: &datcodec.Recipe{
				Sounds: []datcodec.SoundPatchRecipe{{ID: 0, TotalProbability: &soundProbability}},
			},
		},
		{
			Name:         "dat.player-colour-clone",
			Domain:       "dat",
			Summary:      "Clone a player-colour row with an example base color override.",
			AppliesTo:    "Palette and player-color experiments.",
			Command:      "kit dat codec-patch empires2_x2_p1.dat mod.dat --recipe player-colour-clone.json",
			Verification: "structure_verified_not_engine_verified",
			Notes:        []string{"Color integers are DAT-format values; verify in engine because render semantics are external."},
			DatRecipe: &datcodec.Recipe{
				CreatePlayerColour: &datcodec.PlayerColourCreateRecipe{From: 0, Base: &colourBase},
			},
		},
	}
	sort.Slice(templates, func(i, j int) bool {
		return templates[i].Name < templates[j].Name
	})
	return templates
}

func RecipeTemplateSummaries(domain string) (RecipeTemplateListReport, error) {
	domain = strings.ToLower(strings.TrimSpace(domain))
	if domain == "all" {
		domain = ""
	}
	report := RecipeTemplateListReport{
		Verification: "structure_verified_not_engine_verified",
		Domain:       domain,
	}
	for _, template := range RecipeTemplates() {
		if domain != "" && template.Domain != domain {
			continue
		}
		report.Templates = append(report.Templates, RecipeTemplateSummary{
			Name:         template.Name,
			Domain:       template.Domain,
			Summary:      template.Summary,
			AppliesTo:    template.AppliesTo,
			Command:      template.Command,
			Verification: template.Verification,
			Notes:        append([]string(nil), template.Notes...),
		})
	}
	if len(report.Templates) == 0 && domain != "" {
		return RecipeTemplateListReport{}, fmt.Errorf("no recipe templates for domain %q", domain)
	}
	report.Count = len(report.Templates)
	return report, nil
}

func RecipeTemplateByName(name string) (RecipeTemplate, bool) {
	name = strings.TrimSpace(name)
	var suffixMatches []RecipeTemplate
	for _, template := range RecipeTemplates() {
		if template.Name == name {
			return template, true
		}
		if !strings.Contains(name, ".") && strings.HasSuffix(template.Name, "."+name) {
			suffixMatches = append(suffixMatches, template)
		}
	}
	if len(suffixMatches) == 1 {
		return suffixMatches[0], true
	}
	return RecipeTemplate{}, false
}

func RecipeTemplatePayload(template RecipeTemplate) (any, error) {
	if template.ScenarioRecipe != nil && template.DatRecipe != nil {
		return nil, fmt.Errorf("recipe template %q has multiple payloads", template.Name)
	}
	if template.ScenarioRecipe != nil {
		return template.ScenarioRecipe, nil
	}
	if template.DatRecipe != nil {
		return template.DatRecipe, nil
	}
	return nil, fmt.Errorf("recipe template %q has no payload", template.Name)
}

func ExportRecipeTemplates(outDir, domain string, force bool) (RecipeTemplateExportReport, error) {
	outDir = strings.TrimSpace(outDir)
	if outDir == "" {
		return RecipeTemplateExportReport{}, fmt.Errorf("output directory is required")
	}
	domain = strings.ToLower(strings.TrimSpace(domain))
	if domain == "all" {
		domain = ""
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return RecipeTemplateExportReport{}, err
	}
	report := RecipeTemplateExportReport{
		Verification: "structure_verified_not_engine_verified",
		OutputDir:    outDir,
		Domain:       domain,
	}
	seen := map[string]bool{}
	for _, template := range RecipeTemplates() {
		if domain != "" && template.Domain != domain {
			continue
		}
		payload, err := RecipeTemplatePayload(template)
		if err != nil {
			return RecipeTemplateExportReport{}, err
		}
		data, err := json.MarshalIndent(payload, "", "  ")
		if err != nil {
			return RecipeTemplateExportReport{}, fmt.Errorf("%s payload: %w", template.Name, err)
		}
		data = append(data, '\n')
		fileName := recipeTemplateFileName(template)
		if seen[fileName] {
			fileName = strings.ReplaceAll(template.Name, ".", "-") + ".json"
		}
		seen[fileName] = true
		path := filepath.Join(outDir, fileName)
		overwritten := false
		if _, err := os.Stat(path); err == nil {
			if !force {
				return RecipeTemplateExportReport{}, fmt.Errorf("%s already exists; pass --force to overwrite", path)
			}
			overwritten = true
		} else if !os.IsNotExist(err) {
			return RecipeTemplateExportReport{}, err
		}
		if err := os.WriteFile(path, data, 0o644); err != nil {
			return RecipeTemplateExportReport{}, err
		}
		report.Files = append(report.Files, RecipeTemplateExportFile{
			Name:        template.Name,
			Domain:      template.Domain,
			Path:        path,
			Bytes:       len(data),
			Overwritten: overwritten,
		})
	}
	if len(report.Files) == 0 && domain != "" {
		return RecipeTemplateExportReport{}, fmt.Errorf("no recipe templates for domain %q", domain)
	}
	report.Count = len(report.Files)
	return report, nil
}

func recipeTemplateFileName(template RecipeTemplate) string {
	prefix := template.Domain + "."
	name := strings.TrimPrefix(template.Name, prefix)
	name = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
			return r
		}
		return '-'
	}, name)
	name = strings.Trim(name, ".-")
	if name == "" {
		name = strings.ReplaceAll(template.Name, ".", "-")
	}
	return name + ".json"
}

func recipeIntPtr(v int) *int {
	return &v
}

func intPtr(v int) *int {
	return &v
}

func int16Ptr(v int16) *int16 {
	return &v
}
