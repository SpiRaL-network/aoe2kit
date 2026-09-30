package diagnostics

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"aoe2kit/pkg/datcodec"
	"aoe2kit/pkg/datfile"
	"aoe2kit/pkg/scenario"
)

const SemanticsPackVersion = "dat-command-semantics-v4"

type SemanticsPack struct {
	Version        string          `json:"version"`
	Status         string          `json:"status"`
	GeneratedFrom  GeneratedFrom   `json:"generated_from"`
	Constants      PackConstants   `json:"constants"`
	DatRecipe      datcodec.Recipe `json:"dat_recipe"`
	ScenarioRecipe scenario.Recipe `json:"scenario_recipe"`
	Expected       ExpectedLedger  `json:"expected"`
}

type SemanticsPackOptions struct {
	Feature string
}

type LocalBuildingEffectsReport struct {
	Version      string                             `json:"version"`
	Feature      string                             `json:"feature"`
	Status       string                             `json:"status"`
	Honesty      []string                           `json:"honesty"`
	CommandTypes []uint8                            `json:"command_types"`
	Matrix       datcodec.EffectCommandMatrixReport `json:"matrix"`
	Rows         []LocalBuildingEffectRow           `json:"rows"`
}

type LocalBuildingEffectRow struct {
	EffectID    int      `json:"effect_id"`
	EffectName  string   `json:"effect_name"`
	Command     int      `json:"command"`
	Type        uint8    `json:"type"`
	TypeName    string   `json:"type_name"`
	UnitID      int16    `json:"unit_id"`
	AttributeID int16    `json:"attribute_id"`
	Amount      float32  `json:"amount"`
	Summary     string   `json:"summary"`
	Details     []string `json:"details,omitempty"`
	Warnings    []string `json:"warnings,omitempty"`
}

type ClassScopeEffectsReport struct {
	Version string                             `json:"version"`
	Feature string                             `json:"feature"`
	Status  string                             `json:"status"`
	Honesty []string                           `json:"honesty"`
	Matrix  datcodec.EffectCommandMatrixReport `json:"matrix"`
	Rows    []ClassScopeEffectRow              `json:"rows"`
}

type ClassScopeEffectRow struct {
	EffectID      int      `json:"effect_id"`
	EffectName    string   `json:"effect_name"`
	Command       int      `json:"command"`
	Type          uint8    `json:"type"`
	TypeName      string   `json:"type_name"`
	Scope         string   `json:"scope,omitempty"`
	UnitID        int16    `json:"unit_id"`
	UnitClassID   int16    `json:"unit_class_id"`
	UnitClassName string   `json:"unit_class_name,omitempty"`
	AttributeID   int16    `json:"attribute_id"`
	AttributeName string   `json:"attribute_name,omitempty"`
	Amount        float32  `json:"amount"`
	Summary       string   `json:"summary"`
	Details       []string `json:"details,omitempty"`
	Warnings      []string `json:"warnings,omitempty"`
}

type GeneratedFrom struct {
	DatPath         string `json:"dat_path"`
	BaseEffectCount int    `json:"base_effect_count"`
	BaseTechCount   int    `json:"base_tech_count"`
	TemplateTechID  int    `json:"template_tech_id"`
}

type PackConstants struct {
	PlayerCivID      int `json:"player_civ_id"`
	VillagerUnitID   int `json:"villager_unit_id"`
	MilitiaUnitID    int `json:"militia_unit_id"`
	ManAtArmsUnitID  int `json:"man_at_arms_unit_id"`
	ScoutUnitID      int `json:"scout_unit_id"`
	SentinelUnitID   int `json:"sentinel_unit_id"`
	TownCenterUnitID int `json:"town_center_unit_id"`
	BarracksUnitID   int `json:"barracks_unit_id"`
	StableUnitID     int `json:"stable_unit_id"`
	CastleUnitID     int `json:"castle_unit_id"`
	HitPointsAttrID  int `json:"hit_points_attr_id"`
	FoodResourceID   int `json:"food_resource_id"`
	WoodResourceID   int `json:"wood_resource_id"`
	StoneResourceID  int `json:"stone_resource_id"`
	GoldResourceID   int `json:"gold_resource_id"`
}

type ExpectedLedger struct {
	Version       string         `json:"version"`
	Status        string         `json:"status"`
	Honesty       []string       `json:"honesty"`
	Lanes         []ExpectedLane `json:"lanes"`
	PendingLanes  []ExpectedLane `json:"pending_lanes"`
	ReadbackSteps []string       `json:"readback_steps"`
}

type ExpectedLane struct {
	ID                  string         `json:"id"`
	Label               string         `json:"label"`
	Tier                string         `json:"tier"`
	DatEffectID         *int           `json:"dat_effect_id,omitempty"`
	DatEffectIDs        []int          `json:"dat_effect_ids,omitempty"`
	DatTechID           *int           `json:"dat_tech_id,omitempty"`
	DatTechIDs          []int          `json:"dat_tech_ids,omitempty"`
	ScenarioTriggerName string         `json:"scenario_trigger_name,omitempty"`
	Expected            []string       `json:"expected"`
	Evidence            []string       `json:"evidence"`
	Commands            []CommandClaim `json:"commands,omitempty"`
}

type CommandClaim struct {
	Kind        string   `json:"kind,omitempty"`
	RawType     *uint8   `json:"raw_type,omitempty"`
	UnitID      *int16   `json:"unit_id,omitempty"`
	AttributeID *int16   `json:"attribute_id,omitempty"`
	TechID      *int     `json:"tech_id,omitempty"`
	Amount      *float32 `json:"amount,omitempty"`
}

func BuildDATCommandSemanticsPack(datPath string) (SemanticsPack, error) {
	payload, err := readDatPayload(datPath)
	if err != nil {
		return SemanticsPack{}, err
	}
	idx, err := datfile.Parse(payload)
	if err != nil {
		return SemanticsPack{}, fmt.Errorf("parse dat: %w", err)
	}
	if len(idx.Effects) == 0 {
		return SemanticsPack{}, fmt.Errorf("dat has no effects table")
	}
	if len(idx.Techs) <= 22 {
		return SemanticsPack{}, fmt.Errorf("dat has %d techs, need at least template tech 22", len(idx.Techs))
	}
	constants := PackConstants{
		PlayerCivID:      1,
		VillagerUnitID:   83,
		MilitiaUnitID:    74,
		ManAtArmsUnitID:  75,
		ScoutUnitID:      448,
		SentinelUnitID:   74,
		TownCenterUnitID: 109,
		BarracksUnitID:   12,
		StableUnitID:     101,
		CastleUnitID:     82,
		HitPointsAttrID:  0,
		FoodResourceID:   0,
		WoodResourceID:   1,
		StoneResourceID:  2,
		GoldResourceID:   3,
	}
	if len(idx.UnitHeaders) <= constants.ScoutUnitID {
		return SemanticsPack{}, fmt.Errorf("dat has %d unit headers, need unit %d", len(idx.UnitHeaders), constants.ScoutUnitID)
	}
	baseEffectID := len(idx.Effects)
	baseTechID := len(idx.Techs)
	datRecipe := semanticsDatRecipe(baseEffectID, baseTechID, constants)
	scenarioRecipe := semanticsScenarioRecipe(baseTechID, constants)
	return SemanticsPack{
		Version: SemanticsPackVersion,
		Status:  "structure_generated_engine_pending",
		GeneratedFrom: GeneratedFrom{
			DatPath:         datPath,
			BaseEffectCount: baseEffectID,
			BaseTechCount:   baseTechID,
			TemplateTechID:  22,
		},
		Constants:      constants,
		DatRecipe:      datRecipe,
		ScenarioRecipe: scenarioRecipe,
		Expected:       semanticsExpected(baseEffectID, baseTechID, constants),
	}, nil
}

func WriteDATCommandSemanticsPack(datPath, outDir string) (SemanticsPack, error) {
	pack, _, err := WriteDATCommandSemanticsPackWithOptions(datPath, outDir, SemanticsPackOptions{})
	return pack, err
}

func WriteDATCommandSemanticsPackWithOptions(datPath, outDir string, opts SemanticsPackOptions) (SemanticsPack, []string, error) {
	pack, err := BuildDATCommandSemanticsPack(datPath)
	if err != nil {
		return SemanticsPack{}, nil, err
	}
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return SemanticsPack{}, nil, err
	}
	files := []struct {
		name  string
		value any
	}{
		{name: "DAT_COMMAND_SEMANTICS_RECIPE.json", value: pack.DatRecipe},
		{name: "DAT_COMMAND_SEMANTICS_SCEN_RECIPE.json", value: pack.ScenarioRecipe},
		{name: "DAT_COMMAND_SEMANTICS_EXPECTED.json", value: pack.Expected},
	}
	written := make([]string, 0, len(files)+2)
	for _, file := range files {
		data, err := json.MarshalIndent(file.value, "", "  ")
		if err != nil {
			return SemanticsPack{}, nil, fmt.Errorf("marshal %s: %w", file.name, err)
		}
		data = append(data, '\n')
		path := filepath.Join(outDir, file.name)
		if err := os.WriteFile(path, data, 0644); err != nil {
			return SemanticsPack{}, nil, fmt.Errorf("write %s: %w", file.name, err)
		}
		written = append(written, path)
	}
	feature := strings.TrimSpace(opts.Feature)
	if feature == "" || feature == "full" {
		return pack, written, nil
	}
	switch feature {
	case "local-building-effects":
		report, err := BuildLocalBuildingEffectsReport(datPath)
		if err != nil {
			return SemanticsPack{}, nil, err
		}
		jsonPath := filepath.Join(outDir, "LOCAL_BUILDING_EFFECTS.json")
		data, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			return SemanticsPack{}, nil, fmt.Errorf("marshal LOCAL_BUILDING_EFFECTS.json: %w", err)
		}
		data = append(data, '\n')
		if err := os.WriteFile(jsonPath, data, 0644); err != nil {
			return SemanticsPack{}, nil, fmt.Errorf("write LOCAL_BUILDING_EFFECTS.json: %w", err)
		}
		written = append(written, jsonPath)
		mdPath := filepath.Join(outDir, "LOCAL_BUILDING_EFFECTS.md")
		if err := os.WriteFile(mdPath, []byte(localBuildingEffectsMarkdown(report)), 0644); err != nil {
			return SemanticsPack{}, nil, fmt.Errorf("write LOCAL_BUILDING_EFFECTS.md: %w", err)
		}
		written = append(written, mdPath)
	case "class-scope-effects":
		report, err := BuildClassScopeEffectsReport(datPath)
		if err != nil {
			return SemanticsPack{}, nil, err
		}
		jsonPath := filepath.Join(outDir, "CLASS_SCOPE_EFFECTS.json")
		data, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			return SemanticsPack{}, nil, fmt.Errorf("marshal CLASS_SCOPE_EFFECTS.json: %w", err)
		}
		data = append(data, '\n')
		if err := os.WriteFile(jsonPath, data, 0644); err != nil {
			return SemanticsPack{}, nil, fmt.Errorf("write CLASS_SCOPE_EFFECTS.json: %w", err)
		}
		written = append(written, jsonPath)
		mdPath := filepath.Join(outDir, "CLASS_SCOPE_EFFECTS.md")
		if err := os.WriteFile(mdPath, []byte(classScopeEffectsMarkdown(report)), 0644); err != nil {
			return SemanticsPack{}, nil, fmt.Errorf("write CLASS_SCOPE_EFFECTS.md: %w", err)
		}
		written = append(written, mdPath)
	default:
		return SemanticsPack{}, nil, fmt.Errorf("unknown semantics-pack feature %q", feature)
	}
	return pack, written, nil
}

func BuildLocalBuildingEffectsReport(datPath string) (LocalBuildingEffectsReport, error) {
	payload, err := readDatPayload(datPath)
	if err != nil {
		return LocalBuildingEffectsReport{}, err
	}
	idx, err := datfile.Parse(payload)
	if err != nil {
		return LocalBuildingEffectsReport{}, fmt.Errorf("parse dat: %w", err)
	}
	matrix := datcodec.EffectCommandMatrixWithOptions(idx, datcodec.EffectCommandMatrixOptions{ExampleLimit: 10})
	report := LocalBuildingEffectsReport{
		Version:      SemanticsPackVersion,
		Feature:      "local-building-effects",
		Status:       "structure_generated_engine_pending",
		CommandTypes: []uint8{200, 201, 202, 204},
		Matrix:       matrix,
		Honesty: []string{
			"200 and 201 are source-verified by official Update 128442 wording.",
			"202 and 204 are strong hypotheses from DAT rows, mechanical parallels, and known gameplay effects; dedicated engine proof is still pending.",
			"This feature pack is an authoring/readback aid, not a substitute for an in-engine local-building fixture.",
		},
	}
	for _, effect := range idx.Effects {
		var explained map[int]datcodec.EffectCommandExplanation
		for _, command := range effect.Commands {
			if command.Type != 200 && command.Type != 201 && command.Type != 202 && command.Type != 204 {
				continue
			}
			if explained == nil {
				report, err := datcodec.ExplainEffect(idx, effect.Index)
				if err != nil {
					return LocalBuildingEffectsReport{}, err
				}
				explained = make(map[int]datcodec.EffectCommandExplanation, len(report.Commands))
				for _, item := range report.Commands {
					explained[item.Index] = item
				}
			}
			item := explained[command.Index]
			report.Rows = append(report.Rows, LocalBuildingEffectRow{
				EffectID:    effect.Index,
				EffectName:  effect.Name,
				Command:     command.Index,
				Type:        command.Type,
				TypeName:    item.TypeName,
				UnitID:      command.A,
				AttributeID: command.C,
				Amount:      command.D,
				Summary:     item.Summary,
				Details:     item.Details,
				Warnings:    item.Warnings,
			})
		}
	}
	return report, nil
}

func localBuildingEffectsMarkdown(report LocalBuildingEffectsReport) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Local Building Effect Commands\n\n")
	fmt.Fprintf(&b, "Status: `%s`\n\n", report.Status)
	for _, line := range report.Honesty {
		fmt.Fprintf(&b, "- %s\n", line)
	}
	fmt.Fprintf(&b, "\n## Command Family\n\n")
	fmt.Fprintf(&b, "- `200` set local building attribute\n")
	fmt.Fprintf(&b, "- `201` add/subtract local building attribute\n")
	fmt.Fprintf(&b, "- `202` multiply local building attribute\n")
	fmt.Fprintf(&b, "- `204` add/subtract local building packed/advanced attribute, including attack and armor class deltas\n\n")
	fmt.Fprintf(&b, "## DAT Rows\n\n")
	for _, row := range report.Rows {
		fmt.Fprintf(&b, "- effect `%d` `%s`, command `%d`, type `%d` `%s`: %s\n", row.EffectID, row.EffectName, row.Command, row.Type, row.TypeName, row.Summary)
	}
	return b.String()
}

func BuildClassScopeEffectsReport(datPath string) (ClassScopeEffectsReport, error) {
	payload, err := readDatPayload(datPath)
	if err != nil {
		return ClassScopeEffectsReport{}, err
	}
	idx, err := datfile.Parse(payload)
	if err != nil {
		return ClassScopeEffectsReport{}, fmt.Errorf("parse dat: %w", err)
	}
	matrix := datcodec.EffectCommandMatrixWithOptions(idx, datcodec.EffectCommandMatrixOptions{ExampleLimit: 10})
	report := ClassScopeEffectsReport{
		Version: "class-scope-effects-v1",
		Feature: "class-scope-effects",
		Status:  "structure_generated_engine_pending",
		Matrix:  matrix,
		Honesty: []string{
			"This report inventories scoped or class-targeted DAT effect-command rows; it does not prove runtime behavior.",
			"Rows with unit_id=-1 and unit_class_id>=0 are class-targeted candidates.",
			"Rows with non-empty scope apply through the AGE command-family scope labels: team, enemy, neutral, or gaia.",
		},
	}
	for _, effect := range idx.Effects {
		var explained map[int]datcodec.EffectCommandExplanation
		for _, command := range effect.Commands {
			semantic := command.Semantic
			if semantic == nil {
				semantic = datfile.InterpretEffectCommand(command)
			}
			if semantic == nil {
				continue
			}
			classTargeted := command.A < 0 && command.B >= 0
			scoped := semantic.Scope != "" && semantic.Scope != "local_building"
			if !classTargeted && !scoped {
				continue
			}
			if explained == nil {
				report, err := datcodec.ExplainEffect(idx, effect.Index)
				if err != nil {
					return ClassScopeEffectsReport{}, err
				}
				explained = make(map[int]datcodec.EffectCommandExplanation, len(report.Commands))
				for _, item := range report.Commands {
					explained[item.Index] = item
				}
			}
			item := explained[command.Index]
			report.Rows = append(report.Rows, ClassScopeEffectRow{
				EffectID:      effect.Index,
				EffectName:    effect.Name,
				Command:       command.Index,
				Type:          command.Type,
				TypeName:      item.TypeName,
				Scope:         semantic.Scope,
				UnitID:        command.A,
				UnitClassID:   command.B,
				UnitClassName: semantic.UnitClassName,
				AttributeID:   command.C,
				AttributeName: semantic.AttributeName,
				Amount:        command.D,
				Summary:       item.Summary,
				Details:       item.Details,
				Warnings:      item.Warnings,
			})
		}
	}
	return report, nil
}

func classScopeEffectsMarkdown(report ClassScopeEffectsReport) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Class And Scope Effect Commands\n\n")
	fmt.Fprintf(&b, "Status: `%s`\n\n", report.Status)
	for _, line := range report.Honesty {
		fmt.Fprintf(&b, "- %s\n", line)
	}
	fmt.Fprintf(&b, "\n## DAT Rows\n\n")
	for _, row := range report.Rows {
		scope := row.Scope
		if scope == "" {
			scope = "none"
		}
		className := row.UnitClassName
		if className == "" {
			className = "unknown"
		}
		fmt.Fprintf(&b, "- effect `%d` `%s`, command `%d`, type `%d` `%s`, scope `%s`, class `%d` `%s`: %s\n", row.EffectID, row.EffectName, row.Command, row.Type, row.TypeName, scope, row.UnitClassID, className, row.Summary)
	}
	return b.String()
}

func readDatPayload(path string) ([]byte, error) {
	compressed, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	payload, err := datfile.Inflate(compressed)
	if err != nil {
		return nil, fmt.Errorf("inflate dat: %w", err)
	}
	return payload, nil
}

func semanticsDatRecipe(baseEffectID, baseTechID int, c PackConstants) datcodec.Recipe {
	costTargetTech := baseTechID + 9
	disableTargetTech := baseTechID + 11
	costTargetTech2 := baseTechID + 16
	zeroCost := []datfile.ResearchResourceCost{
		{Type: -1, Amount: 0, Flag: 0},
		{Type: -1, Amount: 0, Flag: 0},
		{Type: -1, Amount: 0, Flag: 0},
	}
	lockedFoodCost := []datfile.ResearchResourceCost{
		{Type: int16(c.FoodResourceID), Amount: 9000, Flag: 1},
		{Type: -1, Amount: 0, Flag: 0},
		{Type: -1, Amount: 0, Flag: 0},
	}
	required := []int16{-1, -1, -1, -1, -1, -1}
	requiredCount := int16(0)
	civ := int16(-1)
	fullTechMode := int16(1)
	techType := int16(2)
	repeatable := uint8(1)
	researchLocations := []datfile.ResearchLocation{{LocationID: int16(c.TownCenterUnitID), ResearchTime: 1, ButtonID: 7, HotKeyID: 0}}
	createTech := func(i int, name string) datcodec.TechCreateRecipe {
		effectID := int16(baseEffectID + i)
		return datcodec.TechCreateRecipe{
			From:              22,
			Name:              strPtr(name),
			EffectID:          &effectID,
			RequiredTechs:     required,
			ResourceCosts:     zeroCost,
			RequiredTechCount: &requiredCount,
			Civ:               &civ,
			FullTechMode:      &fullTechMode,
			Type:              &techType,
			Repeatable:        &repeatable,
		}
	}
	createTargetTech := func(i int, name string, effectOffset int, costs []datfile.ResearchResourceCost) datcodec.TechCreateRecipe {
		tech := createTech(i, name)
		effectID := int16(baseEffectID + effectOffset)
		tech.EffectID = &effectID
		tech.ResourceCosts = costs
		tech.ResearchLocations = researchLocations
		return tech
	}
	return datcodec.Recipe{
		CreateEffects: []datcodec.EffectCreateRecipe{
			{
				Name: "A2KSEM_E00_RESOURCE_SET_FOOD_1111",
				Commands: []datcodec.EffectCommandRecipe{
					{Kind: "resource_modifier", ResourceID: int16Ptr(int16(c.FoodResourceID)), OperationID: int16Ptr(0), Amount: float32Ptr(1111)},
				},
			},
			{
				Name: "A2KSEM_E01_RESOURCE_ADD_GOLD_222",
				Commands: []datcodec.EffectCommandRecipe{
					{Kind: "resource_modifier", ResourceID: int16Ptr(int16(c.GoldResourceID)), OperationID: int16Ptr(1), Amount: float32Ptr(222)},
				},
			},
			{
				Name: "A2KSEM_E02_RESOURCE_MULTIPLY_WOOD_3",
				Commands: []datcodec.EffectCommandRecipe{
					{Kind: "resource_multiplier", ResourceID: int16Ptr(int16(c.WoodResourceID)), Amount: float32Ptr(3)},
				},
			},
			{
				Name: "A2KSEM_E03_SET_VILLAGER_HP_1234",
				Commands: []datcodec.EffectCommandRecipe{
					{Kind: "set_attribute", UnitID: int16Ptr(int16(c.VillagerUnitID)), AttributeID: int16Ptr(int16(c.HitPointsAttrID)), Amount: float32Ptr(1234)},
				},
			},
			{
				Name: "A2KSEM_E04_ADD_VILLAGER_HP_222",
				Commands: []datcodec.EffectCommandRecipe{
					{Kind: "add_attribute", UnitID: int16Ptr(int16(c.VillagerUnitID)), AttributeID: int16Ptr(int16(c.HitPointsAttrID)), Amount: float32Ptr(222)},
				},
			},
			{
				Name: "A2KSEM_E05_MULTIPLY_VILLAGER_HP_2",
				Commands: []datcodec.EffectCommandRecipe{
					{Kind: "multiply_attribute", UnitID: int16Ptr(int16(c.VillagerUnitID)), AttributeID: int16Ptr(int16(c.HitPointsAttrID)), Amount: float32Ptr(2)},
				},
			},
			{
				Name: "A2KSEM_E06_ENABLE_SCOUT_UNIT",
				Commands: []datcodec.EffectCommandRecipe{
					{Kind: "enable_unit", UnitID: int16Ptr(int16(c.ScoutUnitID))},
				},
			},
			{
				Name: "A2KSEM_E07_UPGRADE_MILITIA_TO_MAA",
				Commands: []datcodec.EffectCommandRecipe{
					{Kind: "upgrade_unit", UnitID: int16Ptr(int16(c.MilitiaUnitID)), ToUnitID: int16Ptr(int16(c.ManAtArmsUnitID))},
				},
			},
			{
				Name: "A2KSEM_E08_SPAWN_2_VILLAGERS_AT_TC",
				Commands: []datcodec.EffectCommandRecipe{
					{Kind: "spawn_unit", UnitID: int16Ptr(int16(c.VillagerUnitID)), BuildingID: int16Ptr(int16(c.TownCenterUnitID)), Amount: float32Ptr(2)},
				},
			},
			{
				Name: "A2KSEM_E09_TARGET_ADD_STONE_777",
				Commands: []datcodec.EffectCommandRecipe{
					{Kind: "resource_modifier", ResourceID: int16Ptr(int16(c.StoneResourceID)), OperationID: int16Ptr(1), Amount: float32Ptr(777)},
				},
			},
			{
				Name: "A2KSEM_E10_SET_COST_TARGET_FOOD_0",
				Commands: []datcodec.EffectCommandRecipe{
					{Kind: "set_tech_cost", TechID: &costTargetTech, ResourceID: int16Ptr(int16(c.FoodResourceID)), Amount: float32Ptr(0)},
				},
			},
			{
				Name: "A2KSEM_E11_TARGET_ADD_STONE_333",
				Commands: []datcodec.EffectCommandRecipe{
					{Kind: "resource_modifier", ResourceID: int16Ptr(int16(c.StoneResourceID)), OperationID: int16Ptr(1), Amount: float32Ptr(333)},
				},
			},
			{
				Name: "A2KSEM_E12_DISABLE_STONE333_TARGET",
				Commands: []datcodec.EffectCommandRecipe{
					{Kind: "disable_tech", TechID: &disableTargetTech},
				},
			},
			{
				Name: "A2KSEM_E13_RAW_CLASS_ADD_INFANTRY_HP_333",
				Commands: []datcodec.EffectCommandRecipe{
					{Type: 4, A: -1, B: 6, C: int16(c.HitPointsAttrID), D: 333},
				},
			},
			{
				Name: "A2KSEM_E14_RESOURCE_SET_WOOD_100",
				Commands: []datcodec.EffectCommandRecipe{
					{Kind: "resource_modifier", ResourceID: int16Ptr(int16(c.WoodResourceID)), OperationID: int16Ptr(0), Amount: float32Ptr(100)},
				},
			},
			{
				Name: "A2KSEM_E15_TARGET_ADD_STONE_444_CONTROL",
				Commands: []datcodec.EffectCommandRecipe{
					{Kind: "resource_modifier", ResourceID: int16Ptr(int16(c.StoneResourceID)), OperationID: int16Ptr(1), Amount: float32Ptr(444)},
				},
			},
			{
				Name: "A2KSEM_E16_TARGET_ADD_STONE_888_COST_TARGET",
				Commands: []datcodec.EffectCommandRecipe{
					{Kind: "resource_modifier", ResourceID: int16Ptr(int16(c.StoneResourceID)), OperationID: int16Ptr(1), Amount: float32Ptr(888)},
				},
			},
			{
				Name: "A2KSEM_E17_SET_COST_TARGET2_FOOD_0",
				Commands: []datcodec.EffectCommandRecipe{
					{Kind: "set_tech_cost", TechID: &costTargetTech2, ResourceID: int16Ptr(int16(c.FoodResourceID)), Amount: float32Ptr(0)},
				},
			},
		},
		CreateTechs: []datcodec.TechCreateRecipe{
			createTech(0, "A2KSEM_T00_RESOURCE_SET_FOOD"),
			createTech(1, "A2KSEM_T01_RESOURCE_ADD_GOLD"),
			createTech(2, "A2KSEM_T02_RESOURCE_MULTIPLY_WOOD"),
			createTech(3, "A2KSEM_T03_SET_HP"),
			createTech(4, "A2KSEM_T04_ADD_HP"),
			createTech(5, "A2KSEM_T05_MULTIPLY_HP"),
			createTech(6, "A2KSEM_T06_ENABLE_SCOUT"),
			createTech(7, "A2KSEM_T07_UPGRADE_MILITIA_TO_MAA"),
			createTech(8, "A2KSEM_T08_SPAWN_VILLAGERS_AT_TC"),
			createTargetTech(9, "A2KSEM_T09_LOCKED_STONE777_TARGET", 9, lockedFoodCost),
			createTech(10, "A2KSEM_T10_SET_TARGET_COST_ZERO"),
			createTargetTech(11, "A2KSEM_T11_STONE333_DISABLE_TARGET", 11, zeroCost),
			createTech(12, "A2KSEM_T12_DISABLE_STONE333_TARGET"),
			createTech(13, "A2KSEM_T13_RAW_CLASS_ADD_INFANTRY_HP"),
			createTech(14, "A2KSEM_T14_RESOURCE_SET_WOOD"),
			createTargetTech(15, "A2KSEM_T15_STONE444_POSITIVE_CONTROL", 15, zeroCost),
			createTargetTech(16, "A2KSEM_T16_LOCKED_STONE888_COST_TARGET", 16, lockedFoodCost),
			createTech(17, "A2KSEM_T17_SET_TARGET2_COST_ZERO"),
		},
		UnitAvailability: []datcodec.UnitAvailabilityRecipe{
			{CivID: &c.PlayerCivID, UnitID: c.ScoutUnitID, Enabled: false},
			{CivID: &c.PlayerCivID, UnitID: c.VillagerUnitID, Enabled: true},
		},
	}
}

func semanticsScenarioRecipe(baseTechID int, c PackConstants) scenario.Recipe {
	active := true
	inactive := false
	human := true
	computer := false
	lock := true
	aiName := "A2KNoOp"
	aiType := 0
	ally := 0
	neutral := 1
	enemy := 3
	playerCount := 2
	now := int(time.Now().Unix())
	source := 1
	displayTime := 8
	timerDisplay := 60
	timerID := 0
	timeUnit := 2
	resetTimer := 0
	declareEnabled := 1
	conquestRequired := 0
	allCustomRequired := 1
	requiredScore := 0
	timedGame := 0
	research := func(techOffset int, force int) scenario.EffectRecipe {
		return scenario.EffectRecipe{Op: "research_technology", SourcePlayer: &source, Technology: intPtr(baseTechID + techOffset), ForceResearchTechnology: intPtr(force)}
	}
	call := func(name string) scenario.EffectRecipe {
		return scenario.EffectRecipe{Op: "script_call", SourcePlayer: &source, Message: name + "();"}
	}
	create := func(unitID, x, y int) scenario.EffectRecipe {
		return scenario.EffectRecipe{Op: "create_object", SourcePlayer: &source, ObjectListUnitID: intPtr(unitID), LocationX: intPtr(x), LocationY: intPtr(y)}
	}
	triggers := []scenario.TriggerRecipe{
		{Op: "clear_triggers"},
		timedEffectTrigger(2, "A2KSEM2 002 BASELINE", []scenario.EffectRecipe{
			call("A2KSEM2_phase_002_baseline"),
			{Op: "display_instructions", SourcePlayer: &source, Message: "A2KSEM2 Run 3: hands-off isolated DAT semantics probe. Let it run to victory.", DisplayTime: &displayTime},
		}),
		timedEffectTrigger(8, "A2KSEM2 008 RESOURCE SET WOOD BASELINE", []scenario.EffectRecipe{
			research(14, 1),
		}),
		timedEffectTrigger(12, "A2KSEM2 012 RESOURCE SET FOOD", []scenario.EffectRecipe{
			research(0, 1),
			call("A2KSEM2_phase_012_resource_after_food"),
			{Op: "display_instructions", SourcePlayer: &source, Message: "A2KSEM2 resources: wood baseline 100 and food 1111 fired.", DisplayTime: &displayTime},
		}),
		timedEffectTrigger(20, "A2KSEM2 020 RESOURCE ADD GOLD", []scenario.EffectRecipe{
			research(1, 1),
			call("A2KSEM2_phase_020_resource_after_gold"),
		}),
		timedEffectTrigger(30, "A2KSEM2 030 RESOURCE MULTIPLY WOOD", []scenario.EffectRecipe{
			research(2, 1),
			{Op: "display_instructions", SourcePlayer: &source, Message: "A2KSEM2 resources: multiply wood by 3 fired after setting wood to 100.", DisplayTime: &displayTime},
		}),
		timedEffectTrigger(35, "A2KSEM2 035 RESOURCE AFTER MULTIPLY", []scenario.EffectRecipe{
			call("A2KSEM2_phase_035_resource_after_multiply"),
		}),
		timedEffectTrigger(40, "A2KSEM2 040 HP CONTROL BEFORE", []scenario.EffectRecipe{
			call("A2KSEM2_phase_040_hp_control_before"),
		}),
		timedEffectTrigger(42, "A2KSEM2 042 HP CONTROL VILLAGER", []scenario.EffectRecipe{
			create(c.VillagerUnitID, 38, 16),
		}),
		timedEffectTrigger(58, "A2KSEM2 058 HP CONTROL AFTER", []scenario.EffectRecipe{
			call("A2KSEM2_phase_058_hp_control_after"),
		}),
		timedEffectTrigger(60, "A2KSEM2 060 HP COMMANDS", []scenario.EffectRecipe{
			research(3, 1),
			research(4, 1),
			research(5, 1),
			{Op: "display_instructions", SourcePlayer: &source, Message: "A2KSEM2 HP lane: control villager should die; boosted villager spawns next.", DisplayTime: &displayTime},
		}),
		timedEffectTrigger(64, "A2KSEM2 064 HP TEST VILLAGER", []scenario.EffectRecipe{
			create(c.VillagerUnitID, 40, 16),
		}),
		timedEffectTrigger(68, "A2KSEM2 068 HP TEST SPAWNED", []scenario.EffectRecipe{
			call("A2KSEM2_phase_068_hp_test_spawned"),
		}),
		timedEffectTrigger(92, "A2KSEM2 092 HP TEST AFTER", []scenario.EffectRecipe{
			call("A2KSEM2_phase_092_hp_test_after"),
		}),
		timedEffectTrigger(96, "A2KSEM2 096 SCOUT CREATE BEFORE ENABLE", []scenario.EffectRecipe{
			create(c.ScoutUnitID, 22, 18),
			call("A2KSEM2_phase_096_scout_before_enable"),
			{Op: "display_instructions", SourcePlayer: &source, Message: "A2KSEM2 scout lane: attempted create before DAT enable.", DisplayTime: &displayTime},
		}),
		timedEffectTrigger(100, "A2KSEM2 100 ENABLE SCOUT", []scenario.EffectRecipe{
			research(6, 1),
		}),
		timedEffectTrigger(104, "A2KSEM2 104 SCOUT CREATE AFTER ENABLE", []scenario.EffectRecipe{
			create(c.ScoutUnitID, 24, 18),
			call("A2KSEM2_phase_104_scout_after_enable"),
		}),
		timedEffectTrigger(108, "A2KSEM2 108 CREATE UPGRADE MILITIA", []scenario.EffectRecipe{
			create(c.MilitiaUnitID, 12, 36),
		}),
		timedEffectTrigger(112, "A2KSEM2 112 UPGRADE BEFORE", []scenario.EffectRecipe{
			call("A2KSEM2_phase_112_upgrade_before"),
		}),
		timedEffectTrigger(116, "A2KSEM2 116 UPGRADE MILITIA", []scenario.EffectRecipe{
			research(7, 1),
			{Op: "display_instructions", SourcePlayer: &source, Message: "A2KSEM2 upgrade lane: existing militia checked, then new militia created.", DisplayTime: &displayTime},
		}),
		timedEffectTrigger(124, "A2KSEM2 124 UPGRADE AFTER RESEARCH", []scenario.EffectRecipe{
			call("A2KSEM2_phase_124_upgrade_after_research"),
		}),
		timedEffectTrigger(128, "A2KSEM2 128 CREATE MILITIA AFTER UPGRADE", []scenario.EffectRecipe{
			create(c.MilitiaUnitID, 30, 22),
		}),
		timedEffectTrigger(136, "A2KSEM2 136 UPGRADE AFTER NEW CREATE", []scenario.EffectRecipe{
			call("A2KSEM2_phase_136_upgrade_after_new_create"),
		}),
		timedEffectTrigger(144, "A2KSEM2 144 SPAWN BEFORE", []scenario.EffectRecipe{
			call("A2KSEM2_phase_144_spawn_before"),
		}),
		timedEffectTrigger(148, "A2KSEM2 148 SPAWN VILLAGERS", []scenario.EffectRecipe{
			research(8, 1),
			{Op: "display_instructions", SourcePlayer: &source, Message: "A2KSEM2 spawn lane: DAT spawn 2 villagers at TC fired.", DisplayTime: &displayTime},
		}),
		timedEffectTrigger(160, "A2KSEM2 160 SPAWN AFTER", []scenario.EffectRecipe{
			call("A2KSEM2_phase_160_spawn_after"),
		}),
		timedEffectTrigger(168, "A2KSEM2 168 COST TARGET BEFORE MOD", []scenario.EffectRecipe{
			{Op: "research_technology", SourcePlayer: &source, Technology: intPtr(baseTechID + 9), ForceResearchTechnology: intPtr(0)},
			{Op: "display_instructions", SourcePlayer: &source, Message: "A2KSEM2 cost lane: first 9000-food target attempted without force.", DisplayTime: &displayTime},
		}),
		timedEffectTrigger(178, "A2KSEM2 178 COST BEFORE RESULT", []scenario.EffectRecipe{
			call("A2KSEM2_phase_178_cost_before_result"),
		}),
		timedEffectTrigger(184, "A2KSEM2 184 SET TARGET2 COST ZERO", []scenario.EffectRecipe{
			research(17, 1),
		}),
		timedEffectTrigger(188, "A2KSEM2 188 COST TARGET2 AFTER MOD", []scenario.EffectRecipe{
			{Op: "research_technology", SourcePlayer: &source, Technology: intPtr(baseTechID + 16), ForceResearchTechnology: intPtr(0)},
		}),
		timedEffectTrigger(202, "A2KSEM2 202 COST AFTER RESULT", []scenario.EffectRecipe{
			call("A2KSEM2_phase_202_cost_after_result"),
		}),
		timedEffectTrigger(208, "A2KSEM2 208 DISABLE POSITIVE CONTROL", []scenario.EffectRecipe{
			{Op: "research_technology", SourcePlayer: &source, Technology: intPtr(baseTechID + 15), ForceResearchTechnology: intPtr(0)},
		}),
		timedEffectTrigger(216, "A2KSEM2 216 DISABLE POSITIVE RESULT", []scenario.EffectRecipe{
			call("A2KSEM2_phase_216_disable_positive_control"),
		}),
		timedEffectTrigger(220, "A2KSEM2 220 DISABLE TARGET", []scenario.EffectRecipe{
			research(12, 1),
			{Op: "display_instructions", SourcePlayer: &source, Message: "A2KSEM2 disable lane: target disabled before first attempted use.", DisplayTime: &displayTime},
		}),
		timedEffectTrigger(224, "A2KSEM2 224 DISABLED TARGET ATTEMPT", []scenario.EffectRecipe{
			{Op: "research_technology", SourcePlayer: &source, Technology: intPtr(baseTechID + 11), ForceResearchTechnology: intPtr(0)},
		}),
		timedEffectTrigger(236, "A2KSEM2 236 DISABLE AFTER", []scenario.EffectRecipe{
			call("A2KSEM2_phase_236_disable_after"),
		}),
		timedEffectTrigger(244, "A2KSEM2 244 RAW CLASS BEFORE", []scenario.EffectRecipe{
			call("A2KSEM2_phase_244_raw_class_before"),
		}),
		timedEffectTrigger(248, "A2KSEM2 248 RAW CLASS HP", []scenario.EffectRecipe{
			research(13, 1),
			{Op: "display_instructions", SourcePlayer: &source, Message: "A2KSEM2 raw class lane: raw type4 A=-1 B=6 HP+333 fired. Human/combat oracle only.", DisplayTime: &displayTime},
		}),
		timedEffectTrigger(260, "A2KSEM2 260 RAW CLASS AFTER", []scenario.EffectRecipe{
			call("A2KSEM2_phase_260_raw_class_after"),
		}),
		{
			Op:      "add_trigger",
			Name:    "A2KSEM2 TIMER MARKER",
			Enabled: &active,
			Effects: []scenario.EffectRecipe{
				{Op: "display_timer", SourcePlayer: &source, Message: "A2KSEM2 RUN TIMER", DisplayTime: &timerDisplay, TimeUnit: &timeUnit, TimerID: &timerID, ResetTimer: &resetTimer},
			},
		},
		timedEffectTrigger(274, "A2KSEM2 274 CLOSE SIDECAR", []scenario.EffectRecipe{
			call("A2KSEM2_phase_274_close"),
			{Op: "display_instructions", SourcePlayer: &source, Message: "A2KSEM2 sidecar closed. Victory soon.", DisplayTime: &displayTime},
		}),
		timedEffectTrigger(285, "A2KSEM2 285 DECLARE VICTORY", []scenario.EffectRecipe{
			{Op: "declare_victory", SourcePlayer: &source, Enabled: &declareEnabled},
		}),
	}
	units := []scenario.UnitRecipe{}
	units = append(units,
		labelUnit(600, 1, 10.5, 10.5, "A2KSEM2 RUN 3: isolated lab; sidecar records DAT command semantics."),
		labelUnit(600, 1, 18.5, 16.5, "HP lane: remote P2 castle kills a normal villager, then tests boosted villager survival."),
		labelUnit(600, 1, 22.5, 16.5, "Scout lane: create before and after DAT enable."),
		labelUnit(600, 1, 12.5, 34.5, "Upgrade lane: timed militia spawn before DAT upgrade."),
		labelUnit(600, 1, 16.5, 16.5, "Spawn lane: isolated TC probes DAT spawn command."),
		labelUnit(600, 1, 44.5, 16.5, "Cost/disable lanes: stone totals are the oracle."),
		labelUnit(c.BarracksUnitID, 1, 18.5, 20.5, "P1 barracks anchor."),
		labelUnit(c.StableUnitID, 1, 20.5, 20.5, "P1 stable scout availability anchor."),
		labelUnit(c.TownCenterUnitID, 1, 16.5, 20.5, "P1 isolated Town Center spawn/cost tech anchor."),
		labelUnit(c.CastleUnitID, 2, 38.5, 10.5, "P2 remote castle HP oracle."),
		labelUnit(c.BarracksUnitID, 2, 44.5, 44.5, "P2 dummy starter-suppression barracks."),
	)
	return scenario.Recipe{
		Scenario: &scenario.ScenarioRecipe{PlayerCount: &playerCount, TimestampOfLastSave: &now},
		XS:       &scenario.XSRecipe{Name: "A2KSEM2_DAT_SEMANTICS_ENTRY.xs", Content: semanticsXSContent()},
		Victory: &scenario.VictoryRecipe{
			ConquestRequired:               &conquestRequired,
			AllCustomConditionsRequired:    &allCustomRequired,
			RequiredScoreForScoreVictory:   &requiredScore,
			TimeForTimedGameIn10thsOfAYear: &timedGame,
		},
		Players: []scenario.PlayerRecipe{
			{Player: 1, Active: &active, Human: &human, TribeName: strPtr("A2K Tester"), LockCivilization: &lock},
			{Player: 2, Active: &active, Human: &computer, TribeName: strPtr("A2K Castle Oracle"), AIName: &aiName, AIType: &aiType, LockCivilization: &lock, LockPersonality: &lock},
			{Player: 3, Active: &inactive, Human: &computer},
			{Player: 4, Active: &inactive, Human: &computer},
			{Player: 5, Active: &inactive, Human: &computer},
			{Player: 6, Active: &inactive, Human: &computer},
			{Player: 7, Active: &inactive, Human: &computer},
			{Player: 8, Active: &inactive, Human: &computer},
		},
		Diplomacy: []scenario.DiplomacyRecipe{
			{From: 0, To: 1, Stance: neutral},
			{From: 1, To: 0, Stance: neutral},
			{From: 0, To: 2, Stance: neutral},
			{From: 2, To: 0, Stance: neutral},
			{From: 1, To: 1, Stance: ally},
			{From: 2, To: 2, Stance: ally},
			{From: 1, To: 2, Stance: enemy},
			{From: 2, To: 1, Stance: enemy},
		},
		Resources: []scenario.ResourceRecipe{
			{Player: 1, Food: intPtr(0), Wood: intPtr(0), Gold: intPtr(0), Stone: intPtr(0)},
			{Player: 2, Food: intPtr(0), Wood: intPtr(0), Gold: intPtr(0), Stone: intPtr(0)},
		},
		Triggers: triggers,
		Units:    units,
		Map: []scenario.MapRecipe{
			{Op: "set_terrain_rect", X1: 8, Y1: 8, X2: 47, Y2: 26, TerrainID: intPtr(0), Layer: intPtr(-1)},
			{Op: "set_terrain_rect", X1: 12, Y1: 28, X2: 47, Y2: 32, TerrainID: intPtr(1), Layer: intPtr(-1)},
		},
	}
}

func semanticsXSContent() string {
	return `extern const int A2KSEM2_ATTR_FOOD = 0;
extern const int A2KSEM2_ATTR_WOOD = 1;
extern const int A2KSEM2_ATTR_STONE = 2;
extern const int A2KSEM2_ATTR_GOLD = 3;
extern const int A2KSEM2_ATTR_POPULATION = 11;
extern const int A2KSEM2_ATTR_RESEARCH_COUNT = 21;

extern const int A2KSEM2_UNIT_MILITIA = 74;
extern const int A2KSEM2_UNIT_MAN_AT_ARMS = 75;
extern const int A2KSEM2_UNIT_VILLAGER = 83;
extern const int A2KSEM2_UNIT_SCOUT = 448;
extern const int A2KSEM2_UNIT_TOWN_CENTER = 109;
extern const int A2KSEM2_UNIT_BARRACKS = 12;
extern const int A2KSEM2_UNIT_STABLE = 101;

bool a2ksem2_file_open = false;
int a2ksem2_rows_written = 0;

bool A2KSEM2_open_file() {
    if (a2ksem2_file_open == false) {
        a2ksem2_file_open = xsCreateFile(false);
        if ((a2ksem2_file_open == true) && (a2ksem2_rows_written == 0)) {
            xsWriteString("A2K_DAT_COMMAND_SEMANTICS");
            xsWriteInt(2);
			xsWriteString("run3_isolated_dummy_structures_resource_hp_upgrade_spawn_cost_disable");
        }
    }
    return(a2ksem2_file_open);
}

void A2KSEM2_write_row(int phase_id = -1, int lane_id = -1) {
    if (A2KSEM2_open_file() == false) {
        return;
    }
    xsWriteString("phase");
    xsWriteInt(phase_id);
    xsWriteInt(lane_id);
    xsWriteInt(xsGetGameTime());
    xsWriteFloat(xsPlayerAttribute(1, A2KSEM2_ATTR_FOOD));
    xsWriteFloat(xsPlayerAttribute(1, A2KSEM2_ATTR_WOOD));
    xsWriteFloat(xsPlayerAttribute(1, A2KSEM2_ATTR_STONE));
    xsWriteFloat(xsPlayerAttribute(1, A2KSEM2_ATTR_GOLD));
    xsWriteFloat(xsPlayerAttribute(1, A2KSEM2_ATTR_POPULATION));
    xsWriteFloat(xsPlayerAttribute(1, A2KSEM2_ATTR_RESEARCH_COUNT));
    xsWriteInt(xsGetObjectCount(1, A2KSEM2_UNIT_MILITIA));
    xsWriteInt(xsGetObjectCount(1, A2KSEM2_UNIT_MAN_AT_ARMS));
    xsWriteInt(xsGetObjectCount(1, A2KSEM2_UNIT_VILLAGER));
    xsWriteInt(xsGetObjectCount(1, A2KSEM2_UNIT_SCOUT));
    xsWriteInt(xsGetObjectCount(1, A2KSEM2_UNIT_TOWN_CENTER));
    xsWriteInt(xsGetObjectCount(1, A2KSEM2_UNIT_BARRACKS));
    xsWriteInt(xsGetObjectCount(1, A2KSEM2_UNIT_STABLE));
    a2ksem2_rows_written = a2ksem2_rows_written + 1;
}

void A2KSEM2_phase_002_baseline() { A2KSEM2_write_row(2, 0); }
void A2KSEM2_phase_012_resource_after_food() { A2KSEM2_write_row(12, 1); }
void A2KSEM2_phase_020_resource_after_gold() { A2KSEM2_write_row(20, 1); }
void A2KSEM2_phase_035_resource_after_multiply() { A2KSEM2_write_row(35, 1); }
void A2KSEM2_phase_040_hp_control_before() { A2KSEM2_write_row(40, 2); }
void A2KSEM2_phase_058_hp_control_after() { A2KSEM2_write_row(58, 2); }
void A2KSEM2_phase_068_hp_test_spawned() { A2KSEM2_write_row(68, 2); }
void A2KSEM2_phase_092_hp_test_after() { A2KSEM2_write_row(92, 2); }
void A2KSEM2_phase_096_scout_before_enable() { A2KSEM2_write_row(96, 3); }
void A2KSEM2_phase_104_scout_after_enable() { A2KSEM2_write_row(104, 3); }
void A2KSEM2_phase_112_upgrade_before() { A2KSEM2_write_row(112, 4); }
void A2KSEM2_phase_124_upgrade_after_research() { A2KSEM2_write_row(124, 4); }
void A2KSEM2_phase_136_upgrade_after_new_create() { A2KSEM2_write_row(136, 4); }
void A2KSEM2_phase_144_spawn_before() { A2KSEM2_write_row(144, 5); }
void A2KSEM2_phase_160_spawn_after() { A2KSEM2_write_row(160, 5); }
void A2KSEM2_phase_178_cost_before_result() { A2KSEM2_write_row(178, 6); }
void A2KSEM2_phase_202_cost_after_result() { A2KSEM2_write_row(202, 6); }
void A2KSEM2_phase_216_disable_positive_control() { A2KSEM2_write_row(216, 7); }
void A2KSEM2_phase_236_disable_after() { A2KSEM2_write_row(236, 7); }
void A2KSEM2_phase_244_raw_class_before() { A2KSEM2_write_row(244, 8); }
void A2KSEM2_phase_260_raw_class_after() { A2KSEM2_write_row(260, 8); }

void A2KSEM2_phase_274_close() {
    A2KSEM2_write_row(274, 9);
    if (A2KSEM2_open_file() == false) {
        return;
    }
    xsWriteString("end");
    xsWriteInt(a2ksem2_rows_written);
    xsCloseFile();
    a2ksem2_file_open = false;
}
`
}

func semanticsExpected(baseEffectID, baseTechID int, c PackConstants) ExpectedLedger {
	effect := func(offset int) *int {
		return intPtr(baseEffectID + offset)
	}
	tech := func(offset int) *int {
		return intPtr(baseTechID + offset)
	}
	return ExpectedLedger{
		Version: SemanticsPackVersion,
		Status:  "structure_generated_engine_pending",
		Honesty: []string{
			"DAT recipe generation is structure-verified only after kit dat codec-plan/codec-patch succeeds on the target DAT.",
			"Scenario recipe generation is structure-verified only after kit scen plan/patch succeeds on the target base scenario.",
			"XS sidecar rows prove trigger timing, resource values, and object counts only. Hidden stat effects such as live hit points still require a combat/UI oracle.",
			"Command semantics are not engine-verified until the author runs the patched scenario with the patched data mod and AoE2Kit compares the replay/sidecar evidence to this ledger.",
		},
		Lanes: []ExpectedLane{
			{
				ID:                  "lane_a_resource_commands",
				Label:               "DAT resource set/add/multiply commands should update P1 stockpiles in predictable sidecar rows.",
				Tier:                "engine_pending",
				DatEffectIDs:        []int{baseEffectID + 0, baseEffectID + 1, baseEffectID + 2, baseEffectID + 14},
				DatTechIDs:          []int{baseTechID + 0, baseTechID + 1, baseTechID + 2, baseTechID + 14},
				ScenarioTriggerName: "A2KSEM2 008/012/020/030 RESOURCE LANES",
				Expected: []string{
					"phase 002 starts from food/wood/stone/gold near 0/0/0/0",
					"after forced research T14 and T00, phase 012 reports wood near 100 and food near 1111",
					"after forced research T01, phase 020 reports gold near 222",
					"after forced research T02, phase 035 reports wood near 300 if resource_multiplier amount=3 is multiplicative",
				},
				Evidence: []string{"kit dat effect", "kit dat tech", "kit scen effects", "kit xsdat decode with v2 row order", "replay sidecar-sync"},
				Commands: []CommandClaim{
					{Kind: "resource_modifier", RawType: uint8Ptr(1), Amount: float32Ptr(1111)},
					{Kind: "resource_modifier", RawType: uint8Ptr(1), Amount: float32Ptr(222)},
					{Kind: "resource_multiplier", RawType: uint8Ptr(6), Amount: float32Ptr(3)},
					{Kind: "resource_modifier", RawType: uint8Ptr(1), Amount: float32Ptr(100)},
				},
			},
			{
				ID:                  "lane_b_attribute_hp_commands",
				Label:               "DAT set/add/multiply unit hit point commands are structurally emitted, with a castle-fire survival oracle for runtime behavior.",
				Tier:                "engine_pending",
				DatEffectIDs:        []int{baseEffectID + 3, baseEffectID + 4, baseEffectID + 5},
				DatTechIDs:          []int{baseTechID + 3, baseTechID + 4, baseTechID + 5},
				ScenarioTriggerName: "A2KSEM2 040/042/058/060/064/068/092 HP ORACLE",
				Expected: []string{
					"effect rows have types 0/4/5 targeting villager hit_points",
					"phase 058 captures whether a normal villager spawned near the P2 castle died",
					"phase 092 captures whether a post-HP-command villager survived longer near the same P2 castle",
					"the sidecar still cannot directly read hit points; the survival delta is a runtime oracle, not a raw HP read",
				},
				Evidence: []string{"kit dat effect", "kit dat tech", "kit scen effects", "kit xsdat decode with v2 row order", "human/combat oracle for actual HP"},
				Commands: []CommandClaim{
					{Kind: "set_attribute", RawType: uint8Ptr(0), UnitID: int16Ptr(int16(c.VillagerUnitID)), AttributeID: int16Ptr(int16(c.HitPointsAttrID)), Amount: float32Ptr(1234)},
					{Kind: "add_attribute", RawType: uint8Ptr(4), UnitID: int16Ptr(int16(c.VillagerUnitID)), AttributeID: int16Ptr(int16(c.HitPointsAttrID)), Amount: float32Ptr(222)},
					{Kind: "multiply_attribute", RawType: uint8Ptr(5), UnitID: int16Ptr(int16(c.VillagerUnitID)), AttributeID: int16Ptr(int16(c.HitPointsAttrID)), Amount: float32Ptr(2)},
				},
			},
			{
				ID:                  "lane_c_enable_unit",
				Label:               "DAT enable_unit should change scout availability after the civ table disables scout for P1's civ.",
				Tier:                "engine_pending",
				DatEffectID:         effect(6),
				DatTechID:           tech(6),
				ScenarioTriggerName: "A2KSEM2 060/065/070 SCOUT ENABLE",
				Expected: []string{
					"unit availability recipe sets scout disabled for civ 1 before runtime",
					"effect row has type=2 with A equal to scout unit id",
					"phase 096 captures the object count after a pre-enable scout create attempt",
					"phase 104 captures the object count after forced enable_unit and a second scout create attempt",
					"if both scout creates succeed, scenario create_object bypasses civ availability; if only the second succeeds, enable_unit affected scenario creation",
				},
				Evidence: []string{"kit dat availability", "kit dat effect", "kit scen effects", "kit xsdat decode with v2 row order", "optional Stable UI check"},
				Commands: []CommandClaim{
					{Kind: "enable_unit", RawType: uint8Ptr(2), UnitID: int16Ptr(int16(c.ScoutUnitID))},
				},
			},
			{
				ID:                  "lane_d_upgrade_unit",
				Label:               "DAT upgrade_unit should transform existing militia into man-at-arms.",
				Tier:                "engine_pending",
				DatEffectID:         effect(7),
				DatTechID:           tech(7),
				ScenarioTriggerName: "A2KSEM2 082 UPGRADE MILITIA",
				Expected: []string{
					"phase 112 captures militia/man-at-arms counts before forced research",
					"phase 124 captures whether an existing militia transformed after forced research",
					"phase 136 captures whether a militia created after the upgrade appears as militia or man-at-arms",
				},
				Evidence: []string{"kit dat effect", "kit dat tech", "kit scen effects", "kit xsdat decode with v2 row order"},
				Commands: []CommandClaim{
					{Kind: "upgrade_unit", RawType: uint8Ptr(3), UnitID: int16Ptr(int16(c.MilitiaUnitID))},
				},
			},
			{
				ID:                  "lane_e_spawn_unit",
				Label:               "DAT spawn_unit should add two villagers at or near the Town Center.",
				Tier:                "engine_pending",
				DatEffectID:         effect(8),
				DatTechID:           tech(8),
				ScenarioTriggerName: "A2KSEM2 102 SPAWN VILLAGERS",
				Expected: []string{
					"phase 144 captures villager count before forced research",
					"phase 160 captures villager count after forced research; +2 is the expected direct reading if DAT spawn_unit works in scenario mode",
				},
				Evidence: []string{"kit dat effect", "kit dat tech", "kit scen effects", "kit xsdat decode with v2 row order", "replay objects if available"},
				Commands: []CommandClaim{
					{Kind: "spawn_unit", RawType: uint8Ptr(7), UnitID: int16Ptr(int16(c.VillagerUnitID)), Amount: float32Ptr(2)},
				},
			},
			{
				ID:                  "lane_f_tech_cost_modifier",
				Label:               "DAT set_tech_cost should let a 9000-food target tech fire after its food cost is rewritten to zero.",
				Tier:                "engine_pending",
				DatEffectIDs:        []int{baseEffectID + 9, baseEffectID + 16, baseEffectID + 17},
				DatTechIDs:          []int{baseTechID + 9, baseTechID + 16, baseTechID + 17},
				ScenarioTriggerName: "A2KSEM2 168/178/184/188/202 COST TARGET",
				Expected: []string{
					"phase 178 runs after an unforced T09 attempt with 9000 food cost; stone should remain unchanged if cost is enforced",
					"forced research T17 emits set_tech_cost for separate target T16 food=0",
					"phase 202 runs after an unforced T16 attempt; stone should include +888 if cost mutation affects scenario research dispatch",
				},
				Evidence: []string{"kit dat effect", "kit dat tech", "kit scen effects", "kit xsdat decode with v2 row order"},
				Commands: []CommandClaim{
					{Kind: "resource_modifier", RawType: uint8Ptr(1), Amount: float32Ptr(777)},
					{Kind: "resource_modifier", RawType: uint8Ptr(1), Amount: float32Ptr(888)},
					{Kind: "set_tech_cost", RawType: uint8Ptr(100), TechID: intPtr(baseTechID + 16), Amount: float32Ptr(0)},
				},
			},
			{
				ID:                  "lane_g_disable_tech",
				Label:               "DAT disable_tech should prevent a zero-cost target tech from applying after a separate positive-control target proves the wiring works.",
				Tier:                "engine_pending",
				DatEffectIDs:        []int{baseEffectID + 11, baseEffectID + 12, baseEffectID + 15},
				DatTechIDs:          []int{baseTechID + 11, baseTechID + 12, baseTechID + 15},
				ScenarioTriggerName: "A2KSEM2 208/216/220/224/236 DISABLE TARGET",
				Expected: []string{
					"phase 216 runs after zero-cost positive-control T15; stone should gain +444 if target-style resource techs are researchable",
					"forced research T12 emits disable_tech for target T11 before T11 is ever attempted",
					"phase 236 runs after unforced T11 attempt; no additional +333 means disable_tech blocked first use",
				},
				Evidence: []string{"kit dat effect", "kit dat tech", "kit scen effects", "kit xsdat decode with v2 row order"},
				Commands: []CommandClaim{
					{Kind: "resource_modifier", RawType: uint8Ptr(1), Amount: float32Ptr(333)},
					{Kind: "disable_tech", RawType: uint8Ptr(102), TechID: intPtr(baseTechID + 11)},
					{Kind: "resource_modifier", RawType: uint8Ptr(1), Amount: float32Ptr(444)},
				},
			},
			{
				ID:                  "lane_h_raw_class_target",
				Label:               "Raw class-targeted type 4 row probes A=-1/B=6 infantry-class behavior without promoting semantics from structure alone.",
				Tier:                "engine_pending_human_or_combat_oracle",
				DatEffectID:         effect(13),
				DatTechID:           tech(13),
				ScenarioTriggerName: "A2KSEM2 244/248/260 RAW CLASS HP",
				Expected: []string{
					"effect row has raw type=4, A=-1, B=6, C=hit_points, D=333",
					"phase 260 proves the lane fired, but sidecar cannot see class-wide hit point mutation directly",
				},
				Evidence: []string{"kit dat effect --raw", "kit scen effects", "kit xsdat decode with v2 row order", "future combat/UI oracle"},
				Commands: []CommandClaim{
					{Kind: "raw_type4_class_target_probe", RawType: uint8Ptr(4), AttributeID: int16Ptr(int16(c.HitPointsAttrID)), Amount: float32Ptr(333)},
				},
			},
		},
		PendingLanes: []ExpectedLane{
			{ID: "lane_i_unit_task_raw_tail_semantics", Label: "unit task raw-tail semantics", Tier: "not_implemented_in_this_pack", Expected: []string{"requires a dedicated unit-action diagnostic after DAT command semantics are promoted"}},
			{ID: "lane_j_terrain_pass_graphic_semantics", Label: "terrain/passability/graphic interplay", Tier: "not_implemented_in_this_pack", Expected: []string{"requires a visual/pathing diagnostic, not a technology command diagnostic"}},
			{ID: "lane_k_broad_physical_delete_renumber_all", Label: "broad non-tail physical delete renumbering", Tier: "not_implemented_in_this_pack", Expected: []string{"requires a destructive CRUD fixture separate from effect-command semantics"}},
		},
		ReadbackSteps: []string{
			"kit dat semantics-pack <empires.dat> <out-dir>",
			"kit dat codec-plan <empires.dat> --recipe DAT_COMMAND_SEMANTICS_RECIPE.json",
			"kit dat codec-patch <empires.dat> <out.dat> --recipe DAT_COMMAND_SEMANTICS_RECIPE.json",
			"kit scen plan <base.aoe2scenario> --recipe DAT_COMMAND_SEMANTICS_SCEN_RECIPE.json",
			"kit scen patch <base.aoe2scenario> <out.aoe2scenario> --recipe DAT_COMMAND_SEMANTICS_SCEN_RECIPE.json",
			"kit scen xs <out.aoe2scenario> --deploy-tree <mod-root>",
			"place the patched DAT at <mod-root>/resources/_common/dat/empires2_x2_p1.dat and the scenario at <mod-root>/resources/_common/scenario/",
			"run the scenario with the patched data mod enabled until the scripted victory at 285 seconds",
			"decode the sidecar with types: string,int,string, then repeated rows of string,int,int,int,float,float,float,float,float,float,int,int,int,int,int,int,int, then string,int",
			"compare sidecar resource/object-count rows against this ledger before promoting any command semantics",
		},
	}
}

func timedDisplay(seconds int, name, message string) scenario.TriggerRecipe {
	return timedEffectTrigger(seconds, name, []scenario.EffectRecipe{
		{Op: "display_instructions", SourcePlayer: intPtr(1), Message: message, DisplayTime: intPtr(8)},
	})
}

func timedEffectTrigger(seconds int, name string, effects []scenario.EffectRecipe) scenario.TriggerRecipe {
	enabled := true
	return scenario.TriggerRecipe{
		Op:      "add_trigger",
		Name:    name,
		Enabled: &enabled,
		Conditions: []scenario.ConditionRecipe{
			{Op: "timer", Timer: intPtr(seconds)},
		},
		Effects: effects,
	}
}

func labelUnit(unitConst, player int, x, y float64, caption string) scenario.UnitRecipe {
	z := 0.0
	status := 2
	rotation := 0.0
	return scenario.UnitRecipe{
		Op:            "add_unit",
		Player:        player,
		UnitConst:     unitConst,
		X:             &x,
		Y:             &y,
		Z:             &z,
		Status:        &status,
		Rotation:      &rotation,
		CaptionString: caption,
	}
}

func strPtr(v string) *string {
	return &v
}

func intPtr(v int) *int {
	return &v
}

func int16Ptr(v int16) *int16 {
	return &v
}

func float32Ptr(v float32) *float32 {
	return &v
}

func uint8Ptr(v uint8) *uint8 {
	return &v
}
