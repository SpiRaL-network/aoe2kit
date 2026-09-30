package datcodec

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"aoe2kit/pkg/datfile"
)

func TestRoundTripFixtureDAT(t *testing.T) {
	root := os.Getenv("AOE2KIT_FIXTURE_ROOT")
	if root == "" {
		t.Skip("AOE2KIT_FIXTURE_ROOT not set")
	}
	path := filepath.Join(root, "reference_aoe2_dump", "dat", "empires2_x2_p1.dat")
	if _, err := os.Stat(path); err != nil {
		t.Skipf("fixture dat missing: %v", err)
	}
	report, err := RoundTripFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !report.OK || !report.PayloadIdentical {
		t.Fatalf("roundtrip failed: %+v", report)
	}
	if report.TypedTargets.Graphics.Count == 0 ||
		report.TypedTargets.Graphics.Present == 0 ||
		report.TypedTargets.Graphics.Deltas == 0 ||
		len(report.TypedTargets.Graphics.DecodedFields) == 0 {
		t.Fatalf("graphics target not populated: %+v", report.TypedTargets.Graphics)
	}
	if report.TypedTargets.Effects.Count == 0 || report.TypedTargets.Effects.Commands == 0 {
		t.Fatalf("effects target not populated: %+v", report.TypedTargets.Effects)
	}
	if report.TypedTargets.UnitHeaders.Count == 0 ||
		report.TypedTargets.UnitHeaders.Present == 0 ||
		report.TypedTargets.UnitHeaders.Tasks == 0 ||
		len(report.TypedTargets.UnitHeaders.DecodedFields) == 0 {
		t.Fatalf("unit headers target not populated: %+v", report.TypedTargets.UnitHeaders)
	}
	if report.TypedTargets.Techs.Count == 0 {
		t.Fatalf("tech target not populated: %+v", report.TypedTargets.Techs)
	}
	if report.TypedTargets.Civs.Count == 0 || report.TypedTargets.Civs.Resources == 0 || report.TypedTargets.Civs.PresentUnits == 0 {
		t.Fatalf("civs target not populated: %+v", report.TypedTargets.Civs)
	}
	if report.TypedTargets.Units.Count == 0 ||
		report.TypedTargets.Units.Tasks == 0 ||
		report.TypedTargets.Units.TaskRawTailBytes == 0 ||
		report.TypedTargets.Units.Attacks == 0 ||
		report.TypedTargets.Units.Armours == 0 ||
		report.TypedTargets.Units.Costs == 0 {
		t.Fatalf("units target not populated: %+v", report.TypedTargets.Units)
	}
	if report.TypedTargets.TerrainRestrictions.Count == 0 || report.TypedTargets.TerrainRestrictions.TerrainRows == 0 {
		t.Fatalf("terrain restrictions target not populated: %+v", report.TypedTargets.TerrainRestrictions)
	}
	if report.TypedTargets.Terrains.Count == 0 {
		t.Fatalf("terrains target not populated: %+v", report.TypedTargets.Terrains)
	}
	if report.TypedTargets.PlayerColours.Count == 0 {
		t.Fatalf("player colours target not populated: %+v", report.TypedTargets.PlayerColours)
	}
	if report.TypedTargets.Sounds.Count == 0 || report.TypedTargets.Sounds.Items == 0 {
		t.Fatalf("sounds target not populated: %+v", report.TypedTargets.Sounds)
	}
}

func TestCodecRecipeCreateAndDeleteEffectTech(t *testing.T) {
	root := os.Getenv("AOE2KIT_FIXTURE_ROOT")
	if root == "" {
		t.Skip("AOE2KIT_FIXTURE_ROOT not set")
	}
	path := filepath.Join(root, "reference_aoe2_dump", "dat", "empires2_x2_p1.dat")
	compressed, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("fixture dat missing: %v", err)
	}
	beforePayload, err := datfile.Inflate(compressed)
	if err != nil {
		t.Fatal(err)
	}
	before, err := datfile.Parse(beforePayload)
	if err != nil {
		t.Fatal(err)
	}
	newEffectID := int16(len(before.Effects))
	newTechID := len(before.Techs)
	name := "AoE2Kit Test Tech"
	patched, report, err := PatchRecipe(compressed, Recipe{
		CreateEffect: &EffectCreateRecipe{
			Name: "AoE2Kit Test Effect",
			Commands: []EffectCommandRecipe{
				{Type: 0, A: 9, B: 21, C: 3, D: 7},
				{Kind: "disable_tech", TechID: intPtr(244)},
				{Kind: "add_attribute", UnitID: int16Ptr(9), AttributeID: int16Ptr(0), Amount: float32Ptr(50)},
				{Kind: "upgrade_unit", UnitID: int16Ptr(74), ToUnitID: int16Ptr(75)},
			},
		},
		CreateTech: &TechCreateRecipe{
			From:     22,
			Name:     &name,
			EffectID: &newEffectID,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !report.Verified || !report.PayloadRoundTripOK || !report.ReadbackOK {
		t.Fatalf("patch report not verified: %+v", report)
	}
	if report.AfterEffectCount != len(before.Effects)+1 || report.AfterTechCount != len(before.Techs)+1 {
		t.Fatalf("counts after create = effects %d techs %d", report.AfterEffectCount, report.AfterTechCount)
	}
	createdPayload, err := datfile.Inflate(patched)
	if err != nil {
		t.Fatal(err)
	}
	created, err := datfile.Parse(createdPayload)
	if err != nil {
		t.Fatal(err)
	}
	effect := created.Effects[newEffectID]
	if effect.Name != "AoE2Kit Test Effect" || len(effect.Commands) != 4 {
		t.Fatalf("created effect = %+v", effect)
	}
	command := effect.Commands[0]
	if command.Type != 0 || command.A != 9 || command.B != 21 || command.C != 3 || command.D != 7 {
		t.Fatalf("created command = %+v", command)
	}
	disableTech := effect.Commands[1]
	if disableTech.Type != 102 || disableTech.A != -1 || disableTech.B != -1 || disableTech.C != -1 || disableTech.D != 244 || disableTech.Semantic == nil || disableTech.Semantic.ReferenceKind != "tech" {
		t.Fatalf("created disable_tech command = %+v", disableTech)
	}
	addHP := effect.Commands[2]
	if addHP.Type != 4 || addHP.A != 9 || addHP.B != -1 || addHP.C != 0 || addHP.D != 50 || addHP.Semantic == nil || addHP.Semantic.AttributeName != "hit_points" {
		t.Fatalf("created add_attribute command = %+v", addHP)
	}
	upgradeUnit := effect.Commands[3]
	if upgradeUnit.Type != 3 || upgradeUnit.A != 74 || upgradeUnit.B != 75 || upgradeUnit.C != -1 || upgradeUnit.D != 0 ||
		upgradeUnit.Semantic == nil ||
		!semanticHasRef(upgradeUnit.Semantic, "unit", "source_unit", 74) ||
		!semanticHasRef(upgradeUnit.Semantic, "unit", "target_unit", 75) {
		t.Fatalf("created upgrade_unit command = %+v", upgradeUnit)
	}
	tech := created.Techs[newTechID]
	if tech.Name != name || tech.EffectID != newEffectID {
		t.Fatalf("created tech = %+v", tech)
	}
	removed, deleteReport, err := PatchRecipe(patched, Recipe{
		DeleteTechs:   []int{newTechID},
		DeleteEffects: []int{int(newEffectID)},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !deleteReport.Verified || deleteReport.AfterEffectCount != len(before.Effects) || deleteReport.AfterTechCount != len(before.Techs) {
		t.Fatalf("delete report = %+v", deleteReport)
	}
	removedPayload, err := datfile.Inflate(removed)
	if err != nil {
		t.Fatal(err)
	}
	removedIdx, err := datfile.Parse(removedPayload)
	if err != nil {
		t.Fatal(err)
	}
	if len(removedIdx.Effects) != len(before.Effects) || len(removedIdx.Techs) != len(before.Techs) {
		t.Fatalf("counts after delete = effects %d/%d techs %d/%d", len(removedIdx.Effects), len(before.Effects), len(removedIdx.Techs), len(before.Techs))
	}
}

func TestCodecRecipeDeleteNonTailTechTailSwap(t *testing.T) {
	root := os.Getenv("AOE2KIT_FIXTURE_ROOT")
	if root == "" {
		t.Skip("AOE2KIT_FIXTURE_ROOT not set")
	}
	path := filepath.Join(root, "reference_aoe2_dump", "dat", "empires2_x2_p1.dat")
	compressed, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("fixture dat missing: %v", err)
	}
	firstName := "Tail Swap Delete Me"
	secondName := "Tail Swap Survivor"
	patched, createReport, err := PatchRecipe(compressed, Recipe{
		CreateTechs: []TechCreateRecipe{
			{From: 0, Name: &firstName},
			{From: 0, Name: &secondName},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(createReport.CreatedTechs) != 2 {
		t.Fatalf("created techs = %+v", createReport.CreatedTechs)
	}
	deleteID := createReport.CreatedTechs[0]
	tailID := createReport.CreatedTechs[1]
	planPayload, err := datfile.Inflate(patched)
	if err != nil {
		t.Fatal(err)
	}
	planIdx, err := datfile.Parse(planPayload)
	if err != nil {
		t.Fatal(err)
	}
	plan := DeletePlan(planIdx, DeletePlanRequest{Section: "tech", ID: deleteID})
	if !plan.Supported || plan.Strategy != "physical_non_tail_tail_swap" || len(plan.RecipeHint) == 0 {
		t.Fatalf("non-tail tech delete plan = %+v", plan)
	}
	deletedPatched, deleteReport, err := PatchRecipe(patched, Recipe{DeleteTechs: []int{deleteID}})
	if err != nil {
		t.Fatal(err)
	}
	if !deleteReport.Verified || len(deleteReport.DeletedTechDetails) != 1 {
		t.Fatalf("non-tail delete report = %+v", deleteReport)
	}
	detail := deleteReport.DeletedTechDetails[0]
	if detail.TechID != deleteID || detail.Strategy != "physical_non_tail_tail_swap" || detail.MovedTailTechID == nil || *detail.MovedTailTechID != tailID {
		t.Fatalf("non-tail delete detail = %+v", detail)
	}
	deletedPayload, err := datfile.Inflate(deletedPatched)
	if err != nil {
		t.Fatal(err)
	}
	deletedIdx, err := datfile.Parse(deletedPayload)
	if err != nil {
		t.Fatal(err)
	}
	if len(deletedIdx.Techs) != len(planIdx.Techs)-1 {
		t.Fatalf("tech count after non-tail delete = %d want %d", len(deletedIdx.Techs), len(planIdx.Techs)-1)
	}
	if deletedIdx.Techs[deleteID].Name != secondName || deletedIdx.Techs[deleteID].Index != deleteID {
		t.Fatalf("tail tech did not move into deleted slot: %+v", deletedIdx.Techs[deleteID])
	}
}

func TestCodecRecipeEffectCommandHelpersValidate(t *testing.T) {
	root := os.Getenv("AOE2KIT_FIXTURE_ROOT")
	if root == "" {
		t.Skip("AOE2KIT_FIXTURE_ROOT not set")
	}
	path := filepath.Join(root, "reference_aoe2_dump", "dat", "empires2_x2_p1.dat")
	compressed, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("fixture dat missing: %v", err)
	}
	_, _, err = PatchRecipe(compressed, Recipe{
		CreateEffect: &EffectCreateRecipe{
			Name:     "Bad Effect",
			Commands: []EffectCommandRecipe{{Kind: "disable_tech"}},
		},
	})
	if err == nil {
		t.Fatal("missing tech_id helper recipe unexpectedly succeeded")
	}
}

func TestEffectCommandHelpersBuildXSConstants(t *testing.T) {
	amount := float32(10)
	cases := []struct {
		name  string
		input EffectCommandRecipe
		want  datfile.EffectCommand
	}{
		{
			name:  "resource_modifier",
			input: EffectCommandRecipe{Kind: "resource_modifier", ResourceID: int16Ptr(0), OperationID: int16Ptr(1), Amount: &amount},
			want:  datfile.EffectCommand{Type: 1, A: 0, B: 1, C: -1, D: amount},
		},
		{
			name:  "resource_modifier_names",
			input: EffectCommandRecipe{Kind: "resource_modifier", Resource: "cAttributeKills", Operation: "cAttributeAdd", Amount: &amount},
			want:  datfile.EffectCommand{Type: 1, A: 20, B: 1, C: -1, D: amount},
		},
		{
			name:  "resource_multiplier",
			input: EffectCommandRecipe{Kind: "resource_multiplier", ResourceID: int16Ptr(0), Amount: &amount},
			want:  datfile.EffectCommand{Type: 6, A: 0, B: 0, C: -1, D: amount},
		},
		{
			name:  "add_attribute_unit_class_name",
			input: EffectCommandRecipe{Kind: "add_attribute", UnitID: int16Ptr(83), UnitClass: "cCavalryClass", AttributeName: "cAttack", Amount: &amount},
			want:  datfile.EffectCommand{Type: 4, A: 83, B: 912, C: 9, D: amount},
		},
		{
			name:  "spawn_unit",
			input: EffectCommandRecipe{Kind: "spawn_unit", UnitID: int16Ptr(83), BuildingID: int16Ptr(109), Amount: &amount},
			want:  datfile.EffectCommand{Type: 7, A: 83, B: 109, C: -1, D: amount},
		},
		{
			name:  "modify_tech",
			input: EffectCommandRecipe{Kind: "modify_tech", TechID: intPtr(22), TechAttrID: int16Ptr(0), Amount: &amount},
			want:  datfile.EffectCommand{Type: 8, A: 22, B: 0, C: -1, D: amount},
		},
		{
			name:  "modify_tech_name",
			input: EffectCommandRecipe{Kind: "modify_tech", TechID: intPtr(22), TechAttributeName: "cAttrSetGoldCost", Amount: &amount},
			want:  datfile.EffectCommand{Type: 8, A: 22, B: 3, C: -1, D: amount},
		},
		{
			name:  "set_unit_attribute",
			input: EffectCommandRecipe{Kind: "set_unit_attribute", UnitID: int16Ptr(74), AttributeID: int16Ptr(0), Amount: &amount},
			want:  datfile.EffectCommand{Type: 0, A: 74, B: -1, C: 0, D: amount},
		},
		{
			name:  "set_unit_attribute_name",
			input: EffectCommandRecipe{Kind: "set_unit_attribute", UnitID: int16Ptr(74), AttributeName: "cAttack", Amount: &amount},
			want:  datfile.EffectCommand{Type: 0, A: 74, B: -1, C: 9, D: amount},
		},
		{
			name:  "add_tech_cost",
			input: EffectCommandRecipe{Kind: "add_tech_cost", TechID: intPtr(22), ResourceID: int16Ptr(0), Amount: &amount},
			want:  datfile.EffectCommand{Type: 101, A: 22, B: 0, C: -1, D: amount},
		},
		{
			name:  "tech_time_modifier",
			input: EffectCommandRecipe{Kind: "tech_time_modifier", TechID: intPtr(22), OperationID: int16Ptr(1), Amount: &amount},
			want:  datfile.EffectCommand{Type: 103, A: 22, B: 1, C: -1, D: amount},
		},
		{
			name:  "enemy_resource_multiplier",
			input: EffectCommandRecipe{Kind: "enemy_resource_multiplier", Resource: "cAttributeRelics", Amount: recipeFloat32Ptr(0.5)},
			want:  datfile.EffectCommand{Type: 26, A: 7, B: 0, C: -1, D: 0.5},
		},
		{
			name:  "team_modify_tech",
			input: EffectCommandRecipe{Kind: "team_modify_tech", TechID: intPtr(22), TechAttributeName: "cAttrSetButton", Amount: recipeFloat32Ptr(3)},
			want:  datfile.EffectCommand{Type: 18, A: 22, B: 5, C: -1, D: 3},
		},
		{
			name:  "local_building_multiply_attribute",
			input: EffectCommandRecipe{Kind: "multiply_local_building_attribute", UnitID: int16Ptr(82), AttributeName: "garrison_heal_rate", Amount: recipeFloat32Ptr(6)},
			want:  datfile.EffectCommand{Type: 202, A: 82, B: -1, C: 108, D: 6},
		},
		{
			name:  "local_building_add_armor",
			input: EffectCommandRecipe{Kind: "add_local_building_armor", UnitID: int16Ptr(82), ArmorClass: "pierce", PackedAmount: recipeIntPtr(2)},
			want:  datfile.EffectCommand{Type: 204, A: 82, B: -1, C: 8, D: 770},
		},
		{
			name:  "add_unit_attack",
			input: EffectCommandRecipe{Kind: "add_unit_attack", UnitID: int16Ptr(83), AttackClass: "pierce", Amount: recipeFloat32Ptr(3)},
			want:  datfile.EffectCommand{Type: 4, A: 83, B: -1, C: 9, D: 771},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := commandFromRecipe(tc.input)
			if err != nil {
				t.Fatal(err)
			}
			if got.Type != tc.want.Type || got.A != tc.want.A || got.B != tc.want.B || got.C != tc.want.C || got.D != tc.want.D {
				t.Fatalf("command = %+v want %+v", got, tc.want)
			}
		})
	}
}

func TestEffectUnitClassNames(t *testing.T) {
	if got := datfile.EffectUnitClassName(912); got != "cavalry" {
		t.Fatalf("class 912 name = %q", got)
	}
	if got := datfile.EffectUnitClassName(-1); got != "all_units" {
		t.Fatalf("class -1 name = %q", got)
	}
	cases := []struct {
		name string
		want int16
	}{
		{name: "cCavalryClass", want: 912},
		{name: "cavalry", want: 912},
		{name: "all-units-class", want: -1},
	}
	for _, tc := range cases {
		got, ok := datfile.EffectUnitClassID(tc.name)
		if !ok || got != tc.want {
			t.Fatalf("class %q = %d/%t, want %d/true", tc.name, got, ok, tc.want)
		}
	}
}

func TestEffectCommandHelpersRejectConflictingNames(t *testing.T) {
	amount := float32(10)
	_, err := commandFromRecipe(EffectCommandRecipe{
		Kind:         "resource_modifier",
		ResourceID:   int16Ptr(0),
		ResourceName: "gold",
		Operation:    "add",
		Amount:       &amount,
	})
	if err == nil {
		t.Fatal("conflicting resource id/name unexpectedly succeeded")
	}
	_, err = commandFromRecipe(EffectCommandRecipe{
		Kind:              "modify_tech",
		TechID:            intPtr(22),
		TechAttrID:        int16Ptr(0),
		TechAttributeName: "set_gold_cost",
		Amount:            &amount,
	})
	if err == nil {
		t.Fatal("conflicting tech attribute id/name unexpectedly succeeded")
	}
	_, err = commandFromRecipe(EffectCommandRecipe{
		Kind:          "set_unit_attribute",
		UnitID:        int16Ptr(74),
		AttributeID:   int16Ptr(0),
		AttributeName: "attack",
		Amount:        &amount,
	})
	if err == nil {
		t.Fatal("conflicting unit attribute id/name unexpectedly succeeded")
	}
	_, err = commandFromRecipe(EffectCommandRecipe{
		Kind:          "add_attribute",
		UnitID:        int16Ptr(74),
		UnitClassID:   int16Ptr(903),
		UnitClassName: "cCavalryClass",
		AttributeName: "attack",
		Amount:        &amount,
	})
	if err == nil {
		t.Fatal("conflicting unit class id/name unexpectedly succeeded")
	}
}

func TestEffectCommandRecipesFromCommandsUsesHelpers(t *testing.T) {
	commands := []datfile.EffectCommand{
		{Type: 1, A: 20, B: 1, C: -1, D: 5},
		{Type: 8, A: 22, B: 3, C: -1, D: 10},
		{Type: 4, A: 83, B: 912, C: 9, D: 3},
		{Type: 250, A: 1, B: 2, C: 3, D: 4},
	}
	recipes := effectCommandRecipesFromCommands(commands)
	if len(recipes) != len(commands) {
		t.Fatalf("recipes len = %d, want %d", len(recipes), len(commands))
	}
	if recipes[0].Kind != "resource_modifier" ||
		recipes[0].ResourceID == nil || *recipes[0].ResourceID != 20 ||
		recipes[0].ResourceName != "kills" ||
		recipes[0].OperationID == nil || *recipes[0].OperationID != 1 ||
		recipes[0].OperationName != "add" {
		t.Fatalf("resource recipe = %+v", recipes[0])
	}
	if recipes[1].Kind != "modify_tech" ||
		recipes[1].TechID == nil || *recipes[1].TechID != 22 ||
		recipes[1].TechAttrID == nil || *recipes[1].TechAttrID != 3 ||
		recipes[1].TechAttributeName != "set_gold_cost" {
		t.Fatalf("modify tech recipe = %+v", recipes[1])
	}
	if recipes[2].Kind != "add_attribute" ||
		recipes[2].UnitClassID == nil || *recipes[2].UnitClassID != 912 ||
		recipes[2].UnitClassName != "cavalry" ||
		recipes[2].AttributeID == nil || *recipes[2].AttributeID != 9 ||
		recipes[2].AttributeName != "attack" {
		t.Fatalf("add attribute recipe = %+v", recipes[2])
	}
	if recipes[3].Kind != "" || recipes[3].Type != commands[3].Type || recipes[3].A != commands[3].A || recipes[3].B != commands[3].B || recipes[3].C != commands[3].C || recipes[3].D != commands[3].D {
		t.Fatalf("raw fallback recipe = %+v", recipes[3])
	}
	roundTrip, err := commandsFromRecipe(recipes)
	if err != nil {
		t.Fatal(err)
	}
	for i := range commands {
		if roundTrip[i].Type != commands[i].Type || roundTrip[i].A != commands[i].A || roundTrip[i].B != commands[i].B || roundTrip[i].C != commands[i].C || roundTrip[i].D != commands[i].D {
			t.Fatalf("roundtrip[%d] = %+v want %+v", i, roundTrip[i], commands[i])
		}
	}
}

func TestCodecRecipeRemoveEffectCommands(t *testing.T) {
	root := os.Getenv("AOE2KIT_FIXTURE_ROOT")
	if root == "" {
		t.Skip("AOE2KIT_FIXTURE_ROOT not set")
	}
	path := filepath.Join(root, "reference_aoe2_dump", "dat", "empires2_x2_p1.dat")
	compressed, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("fixture dat missing: %v", err)
	}
	payload, err := datfile.Inflate(compressed)
	if err != nil {
		t.Fatal(err)
	}
	before, err := datfile.Parse(payload)
	if err != nil {
		t.Fatal(err)
	}
	newEffectID := len(before.Effects)
	created, _, err := PatchRecipe(compressed, Recipe{
		CreateEffect: &EffectCreateRecipe{
			Name: "AoE2Kit Remove Command Test",
			Commands: []EffectCommandRecipe{
				{Type: 0, A: 9, B: 21, C: 3, D: 7},
				{Kind: "disable_tech", TechID: intPtr(244)},
				{Kind: "add_attribute", UnitID: int16Ptr(9), AttributeID: int16Ptr(0), Amount: float32Ptr(50)},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	patched, report, err := PatchRecipe(created, Recipe{
		Effects: []EffectPatchRecipe{{
			ID:             newEffectID,
			RemoveCommands: []int{1},
			AppendCommands: []EffectCommandRecipe{
				{Kind: "enable_unit", UnitID: int16Ptr(83)},
			},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !report.Verified || len(report.PatchedEffects) != 1 || report.PatchedEffects[0] != newEffectID {
		t.Fatalf("remove command report = %+v", report)
	}
	afterPayload, err := datfile.Inflate(patched)
	if err != nil {
		t.Fatal(err)
	}
	after, err := datfile.Parse(afterPayload)
	if err != nil {
		t.Fatal(err)
	}
	effect := after.Effects[newEffectID]
	if effect.CommandSize != 3 || len(effect.Commands) != 3 {
		t.Fatalf("effect command count = %d/%d", effect.CommandSize, len(effect.Commands))
	}
	if effect.Commands[0].Index != 0 || effect.Commands[0].Type != 0 {
		t.Fatalf("first command = %+v", effect.Commands[0])
	}
	if effect.Commands[1].Index != 1 || effect.Commands[1].Type != 4 {
		t.Fatalf("surviving command = %+v", effect.Commands[1])
	}
	if effect.Commands[2].Index != 2 || effect.Commands[2].Type != 2 || effect.Commands[2].A != 83 {
		t.Fatalf("appended command = %+v", effect.Commands[2])
	}
	_, _, err = PatchRecipe(created, Recipe{
		Effects: []EffectPatchRecipe{{
			ID:             newEffectID,
			RemoveCommands: []int{99},
		}},
	})
	if err == nil {
		t.Fatal("out-of-range remove_commands unexpectedly succeeded")
	}
}

func TestCodecRecipeCloneEffect(t *testing.T) {
	root := os.Getenv("AOE2KIT_FIXTURE_ROOT")
	if root == "" {
		t.Skip("AOE2KIT_FIXTURE_ROOT not set")
	}
	path := filepath.Join(root, "reference_aoe2_dump", "dat", "empires2_x2_p1.dat")
	compressed, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("fixture dat missing: %v", err)
	}
	payload, err := datfile.Inflate(compressed)
	if err != nil {
		t.Fatal(err)
	}
	before, err := datfile.Parse(payload)
	if err != nil {
		t.Fatal(err)
	}
	sourceTechID := firstTechWithCommandedEffect(t, before)
	sourceEffectID := int(before.Techs[sourceTechID].EffectID)
	sourceEffect := before.Effects[sourceEffectID]
	if len(sourceEffect.Commands) == 0 {
		t.Fatal("source effect has no commands")
	}
	newEffectID := len(before.Effects)
	patched, report, err := PatchRecipe(compressed, Recipe{
		CreateEffect: &EffectCreateRecipe{
			Name:           "AoE2Kit Cloned Effect",
			FromEffect:     &sourceEffectID,
			RemoveCommands: []int{0},
			AppendCommands: []EffectCommandRecipe{
				{Kind: "enable_unit", UnitID: int16Ptr(83)},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !report.Verified || len(report.CreatedEffects) != 1 || report.CreatedEffects[0] != newEffectID {
		t.Fatalf("clone effect report = %+v", report)
	}
	afterPayload, err := datfile.Inflate(patched)
	if err != nil {
		t.Fatal(err)
	}
	after, err := datfile.Parse(afterPayload)
	if err != nil {
		t.Fatal(err)
	}
	effect := after.Effects[newEffectID]
	wantCommands := len(sourceEffect.Commands)
	if len(effect.Commands) != wantCommands {
		t.Fatalf("cloned effect command count=%d want %d", len(effect.Commands), wantCommands)
	}
	if len(sourceEffect.Commands) > 1 {
		first := effect.Commands[0]
		wantFirst := sourceEffect.Commands[1]
		if first.Type != wantFirst.Type || first.A != wantFirst.A || first.B != wantFirst.B || first.C != wantFirst.C || first.D != wantFirst.D {
			t.Fatalf("first surviving command = %+v want %+v", first, wantFirst)
		}
	}
	appended := effect.Commands[len(effect.Commands)-1]
	if appended.Index != len(effect.Commands)-1 || appended.Type != 2 || appended.A != 83 {
		t.Fatalf("appended command = %+v", appended)
	}
	disabledPatched, disableReport, err := PatchRecipe(patched, Recipe{DisableEffect: &newEffectID})
	if err != nil {
		t.Fatal(err)
	}
	if !disableReport.Verified || len(disableReport.DisabledEffects) != 1 || disableReport.DisabledEffects[0] != newEffectID || len(disableReport.PatchedEffects) != 1 {
		t.Fatalf("disable effect report = %+v", disableReport)
	}
	disabledPayload, err := datfile.Inflate(disabledPatched)
	if err != nil {
		t.Fatal(err)
	}
	disabledIdx, err := datfile.Parse(disabledPayload)
	if err != nil {
		t.Fatal(err)
	}
	disabledEffect := disabledIdx.Effects[newEffectID]
	if disabledEffect.CommandSize != 0 || len(disabledEffect.Commands) != 0 {
		t.Fatalf("disabled effect command count = %d/%d, want 0/0", disabledEffect.CommandSize, len(disabledEffect.Commands))
	}
}

func TestPatchEffectCanSemanticallyClearCommands(t *testing.T) {
	commands := []EffectCommandRecipe{}
	effect, err := patchEffect(datfile.Effect{
		Index:       12,
		Name:        "Clear Me",
		CommandSize: 2,
		Commands: []datfile.EffectCommand{
			{Index: 0, Type: 0, A: 83, B: -1, C: 0, D: 50},
			{Index: 1, Type: 102, A: -1, B: -1, C: -1, D: 244},
		},
	}, EffectPatchRecipe{ID: 12, Commands: &commands})
	if err != nil {
		t.Fatalf("patchEffect: %v", err)
	}
	if effect.CommandSize != 0 || len(effect.Commands) != 0 {
		t.Fatalf("cleared effect command count = %d/%d, want 0/0", effect.CommandSize, len(effect.Commands))
	}
}

func TestCodecRecipeCreateAbility(t *testing.T) {
	root := os.Getenv("AOE2KIT_FIXTURE_ROOT")
	if root == "" {
		t.Skip("AOE2KIT_FIXTURE_ROOT not set")
	}
	path := filepath.Join(root, "reference_aoe2_dump", "dat", "empires2_x2_p1.dat")
	compressed, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("fixture dat missing: %v", err)
	}
	payload, err := datfile.Inflate(compressed)
	if err != nil {
		t.Fatal(err)
	}
	before, err := datfile.Parse(payload)
	if err != nil {
		t.Fatal(err)
	}
	amount := float32(25)
	patched, report, err := PatchRecipe(compressed, Recipe{
		CreateAbility: &AbilityCreateRecipe{
			Name:       "AoE2Kit Test Ability",
			EffectName: "AoE2Kit Test Ability Effect",
			FromTech:   22,
			Commands: []EffectCommandRecipe{
				{Kind: "add_attribute", UnitID: int16Ptr(83), AttributeID: int16Ptr(0), Amount: &amount},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !report.Verified || !report.PayloadRoundTripOK || !report.ReadbackOK {
		t.Fatalf("ability report not verified: %+v", report)
	}
	if len(report.CreatedAbilities) != 1 || report.AfterEffectCount != len(before.Effects)+1 || report.AfterTechCount != len(before.Techs)+1 {
		t.Fatalf("ability create counts/report = %+v", report)
	}
	ability := report.CreatedAbilities[0]
	if ability.TechID != len(before.Techs) || ability.EffectID != len(before.Effects) {
		t.Fatalf("ability ids = %+v, want tech %d effect %d", ability, len(before.Techs), len(before.Effects))
	}
	afterPayload, err := datfile.Inflate(patched)
	if err != nil {
		t.Fatal(err)
	}
	after, err := datfile.Parse(afterPayload)
	if err != nil {
		t.Fatal(err)
	}
	tech := after.Techs[ability.TechID]
	if tech.Name != "AoE2Kit Test Ability" || int(tech.EffectID) != ability.EffectID {
		t.Fatalf("ability tech = %+v", tech)
	}
	effect := after.Effects[ability.EffectID]
	if effect.Name != "AoE2Kit Test Ability Effect" || len(effect.Commands) != 1 {
		t.Fatalf("ability effect = %+v", effect)
	}
	command := effect.Commands[0]
	if command.Type != 4 || command.A != 83 || command.B != -1 || command.C != 0 || command.D != 25 ||
		command.Semantic == nil || command.Semantic.TypeName != "add_attribute" || command.Semantic.AttributeName != "hit_points" {
		t.Fatalf("ability command semantic = %+v", command)
	}
	disabledPatched, disableReport, err := PatchRecipe(patched, Recipe{DisableAbility: &ability.TechID})
	if err != nil {
		t.Fatal(err)
	}
	if !disableReport.Verified || len(disableReport.DisabledAbilities) != 1 || len(disableReport.PatchedEffects) != 1 {
		t.Fatalf("ability disable report = %+v", disableReport)
	}
	disabled := disableReport.DisabledAbilities[0]
	if disabled.TechID != ability.TechID || disabled.EffectID != ability.EffectID || disabled.Strategy != "semantic_clear_ability_effect" {
		t.Fatalf("disabled ability = %+v, created ability = %+v", disabled, ability)
	}
	disabledPayload, err := datfile.Inflate(disabledPatched)
	if err != nil {
		t.Fatal(err)
	}
	disabledIdx, err := datfile.Parse(disabledPayload)
	if err != nil {
		t.Fatal(err)
	}
	if len(disabledIdx.Techs) != len(before.Techs)+1 || len(disabledIdx.Effects) != len(before.Effects)+1 {
		t.Fatalf("disable ability changed counts effects=%d/%d techs=%d/%d", len(disabledIdx.Effects), len(before.Effects)+1, len(disabledIdx.Techs), len(before.Techs)+1)
	}
	disabledEffect := disabledIdx.Effects[ability.EffectID]
	if disabledEffect.CommandSize != 0 || len(disabledEffect.Commands) != 0 {
		t.Fatalf("disabled ability effect commands = %d/%d, want 0/0", disabledEffect.CommandSize, len(disabledEffect.Commands))
	}
	deletedPatched, deleteReport, err := PatchRecipe(patched, Recipe{DeleteAbility: &ability.TechID})
	if err != nil {
		t.Fatal(err)
	}
	if !deleteReport.Verified || len(deleteReport.DeletedAbilities) != 1 {
		t.Fatalf("ability delete report = %+v", deleteReport)
	}
	deleted := deleteReport.DeletedAbilities[0]
	if deleted.TechID != ability.TechID || deleted.EffectID != ability.EffectID || deleted.Strategy != "paired_tech_then_effect_delete" {
		t.Fatalf("deleted ability = %+v, created ability = %+v", deleted, ability)
	}
	deletedPayload, err := datfile.Inflate(deletedPatched)
	if err != nil {
		t.Fatal(err)
	}
	deletedIdx, err := datfile.Parse(deletedPayload)
	if err != nil {
		t.Fatal(err)
	}
	if len(deletedIdx.Techs) != len(before.Techs) || len(deletedIdx.Effects) != len(before.Effects) {
		t.Fatalf("delete ability counts effects=%d/%d techs=%d/%d", len(deletedIdx.Effects), len(before.Effects), len(deletedIdx.Techs), len(before.Techs))
	}
}

func TestCodecRecipeDeleteNonTailAbilityTailSwap(t *testing.T) {
	root := os.Getenv("AOE2KIT_FIXTURE_ROOT")
	if root == "" {
		t.Skip("AOE2KIT_FIXTURE_ROOT not set")
	}
	path := filepath.Join(root, "reference_aoe2_dump", "dat", "empires2_x2_p1.dat")
	compressed, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("fixture dat missing: %v", err)
	}
	payload, err := datfile.Inflate(compressed)
	if err != nil {
		t.Fatal(err)
	}
	before, err := datfile.Parse(payload)
	if err != nil {
		t.Fatal(err)
	}
	amount := float32(25)
	patched, createReport, err := PatchRecipe(compressed, Recipe{
		CreateAbilities: []AbilityCreateRecipe{
			{
				Name:       "Delete First Ability",
				EffectName: "Delete First Ability Effect",
				FromTech:   22,
				Commands:   []EffectCommandRecipe{{Kind: "add_attribute", UnitID: int16Ptr(83), AttributeID: int16Ptr(0), Amount: &amount}},
			},
			{
				Name:       "Surviving Ability",
				EffectName: "Surviving Ability Effect",
				FromTech:   22,
				Commands:   []EffectCommandRecipe{{Kind: "add_attribute", UnitID: int16Ptr(83), AttributeID: int16Ptr(0), Amount: &amount}},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(createReport.CreatedAbilities) != 2 {
		t.Fatalf("created abilities = %+v", createReport.CreatedAbilities)
	}
	deleteAbility := createReport.CreatedAbilities[0]
	survivor := createReport.CreatedAbilities[1]
	planPayload, err := datfile.Inflate(patched)
	if err != nil {
		t.Fatal(err)
	}
	planIdx, err := datfile.Parse(planPayload)
	if err != nil {
		t.Fatal(err)
	}
	plan := DeletePlan(planIdx, DeletePlanRequest{Section: "ability", ID: deleteAbility.TechID})
	if !plan.Supported || plan.Strategy != "paired_tech_then_effect_delete" || len(plan.RecipeHint) == 0 {
		t.Fatalf("non-tail ability delete plan = %+v", plan)
	}
	deletedPatched, deleteReport, err := PatchRecipe(patched, Recipe{DeleteAbility: &deleteAbility.TechID})
	if err != nil {
		t.Fatal(err)
	}
	if !deleteReport.Verified || len(deleteReport.DeletedAbilities) != 1 || len(deleteReport.DeletedTechDetails) != 1 {
		t.Fatalf("non-tail ability delete report = %+v", deleteReport)
	}
	detail := deleteReport.DeletedTechDetails[0]
	if detail.TechID != deleteAbility.TechID || detail.Strategy != "physical_non_tail_tail_swap" || detail.MovedTailTechID == nil || *detail.MovedTailTechID != survivor.TechID {
		t.Fatalf("non-tail ability tech delete detail = %+v", detail)
	}
	deletedPayload, err := datfile.Inflate(deletedPatched)
	if err != nil {
		t.Fatal(err)
	}
	deletedIdx, err := datfile.Parse(deletedPayload)
	if err != nil {
		t.Fatal(err)
	}
	if len(deletedIdx.Techs) != len(before.Techs)+1 || len(deletedIdx.Effects) != len(before.Effects)+1 {
		t.Fatalf("non-tail ability delete counts effects=%d/%d techs=%d/%d", len(deletedIdx.Effects), len(before.Effects)+1, len(deletedIdx.Techs), len(before.Techs)+1)
	}
	movedTech := deletedIdx.Techs[deleteAbility.TechID]
	if movedTech.Name != "Surviving Ability" || int(movedTech.EffectID) != deleteAbility.EffectID {
		t.Fatalf("surviving ability did not move/renumber correctly: %+v", movedTech)
	}
	movedEffect := deletedIdx.Effects[deleteAbility.EffectID]
	if movedEffect.Name != "Surviving Ability Effect" {
		t.Fatalf("surviving effect did not renumber correctly: %+v", movedEffect)
	}
}

func TestCodecRecipeCloneAbilityEffect(t *testing.T) {
	root := os.Getenv("AOE2KIT_FIXTURE_ROOT")
	if root == "" {
		t.Skip("AOE2KIT_FIXTURE_ROOT not set")
	}
	path := filepath.Join(root, "reference_aoe2_dump", "dat", "empires2_x2_p1.dat")
	compressed, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("fixture dat missing: %v", err)
	}
	payload, err := datfile.Inflate(compressed)
	if err != nil {
		t.Fatal(err)
	}
	before, err := datfile.Parse(payload)
	if err != nil {
		t.Fatal(err)
	}
	sourceTechID := firstTechWithCommandedEffect(t, before)
	sourceTech := before.Techs[sourceTechID]
	sourceEffect := before.Effects[sourceTech.EffectID]
	name := "AoE2Kit Cloned Ability"
	patched, report, err := PatchRecipe(compressed, Recipe{
		CreateAbility: &AbilityCreateRecipe{
			Name:     name,
			FromTech: sourceTechID,
			AppendCommands: []EffectCommandRecipe{
				{Kind: "disable_tech", TechID: intPtr(244)},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !report.Verified || len(report.CreatedAbilities) != 1 {
		t.Fatalf("clone ability report = %+v", report)
	}
	ability := report.CreatedAbilities[0]
	afterPayload, err := datfile.Inflate(patched)
	if err != nil {
		t.Fatal(err)
	}
	after, err := datfile.Parse(afterPayload)
	if err != nil {
		t.Fatal(err)
	}
	tech := after.Techs[ability.TechID]
	effect := after.Effects[ability.EffectID]
	if tech.Name != name || tech.EffectID != int16(ability.EffectID) {
		t.Fatalf("cloned tech = %+v", tech)
	}
	if len(effect.Commands) != len(sourceEffect.Commands)+1 {
		t.Fatalf("cloned effect command count=%d want %d", len(effect.Commands), len(sourceEffect.Commands)+1)
	}
	first := effect.Commands[0]
	wantFirst := sourceEffect.Commands[0]
	if first.Type != wantFirst.Type || first.A != wantFirst.A || first.B != wantFirst.B || first.C != wantFirst.C || first.D != wantFirst.D {
		t.Fatalf("first cloned command = %+v want %+v", first, wantFirst)
	}
	appended := effect.Commands[len(effect.Commands)-1]
	if appended.Type != 102 || appended.D != 244 || appended.Semantic == nil || appended.Semantic.ReferenceKind != "tech" {
		t.Fatalf("appended command = %+v", appended)
	}
}

func TestCodecRecipeCreateGraphicAndUnitBridge(t *testing.T) {
	root := os.Getenv("AOE2KIT_FIXTURE_ROOT")
	if root == "" {
		t.Skip("AOE2KIT_FIXTURE_ROOT not set")
	}
	path := filepath.Join(root, "reference_aoe2_dump", "dat", "empires2_x2_p1.dat")
	compressed, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("fixture dat missing: %v", err)
	}
	payload, err := datfile.Inflate(compressed)
	if err != nil {
		t.Fatal(err)
	}
	before, err := datfile.Parse(payload)
	if err != nil {
		t.Fatal(err)
	}
	name := "AoE2Kit Codec Create Graphic"
	particle := "aoe2kit_codec_create"
	slp := int32(971)
	hp := int16(77)
	enabled := uint8(1)
	patched, report, err := PatchRecipe(compressed, Recipe{
		CreateGraphic: &datfile.GraphicCreateRecipe{
			From:               1711,
			Name:               &name,
			ParticleEffectName: &particle,
			SLP:                &slp,
		},
		CreateUnit: &datfile.UnitCreateRecipe{
			FromCivID:  0,
			FromUnitID: 83,
			CivIDs:     []int{0, 1},
			HitPoints:  &hp,
			Enabled:    &enabled,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !report.Verified || !report.PayloadRoundTripOK || !report.ReadbackOK {
		t.Fatalf("create bridge report not verified: %+v", report)
	}
	if report.BeforeGraphicCount != before.GraphicsSize || report.AfterGraphicCount != before.GraphicsSize+1 || len(report.CreatedGraphics) != 1 || len(report.CreatedUnits) != 1 {
		t.Fatalf("create bridge counts/report = before_gfx %d after_gfx %d created_gfx %d created_units %d",
			report.BeforeGraphicCount, report.AfterGraphicCount, len(report.CreatedGraphics), len(report.CreatedUnits))
	}
	afterPayload, err := datfile.Inflate(patched)
	if err != nil {
		t.Fatal(err)
	}
	after, err := datfile.Parse(afterPayload)
	if err != nil {
		t.Fatal(err)
	}
	createdGraphicID := report.CreatedGraphics[0].NewGraphicID
	graphic, ok := after.Graphic(createdGraphicID)
	if !ok {
		t.Fatalf("created graphic %d missing", createdGraphicID)
	}
	if graphic.Name.Value != name || graphic.ParticleEffectName.Value != particle || graphic.SLP != slp {
		t.Fatalf("created graphic readback = %+v", graphic.Summary())
	}
	createdUnitID := report.CreatedUnits[0].NewUnitID
	for _, civID := range []int{0, 1} {
		unit := after.Civs[civID].Units[createdUnitID]
		if !unit.Present || unit.ID != int16(createdUnitID) || unit.HitPoints != hp || unit.Enabled != enabled {
			t.Fatalf("created civ %d unit %d = %+v", civID, createdUnitID, unit)
		}
	}
}

func TestTechTreeEncodeIdentityAndPatch(t *testing.T) {
	root := os.Getenv("AOE2KIT_FIXTURE_ROOT")
	if root == "" {
		t.Skip("AOE2KIT_FIXTURE_ROOT not set")
	}
	path := filepath.Join(root, "reference_aoe2_dump", "dat", "empires2_x2_p1.dat")
	compressed, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("fixture dat missing: %v", err)
	}
	payload, err := datfile.Inflate(compressed)
	if err != nil {
		t.Fatal(err)
	}
	idx, err := datfile.Parse(payload)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := encodeTechTreeSection(idx.TechTree)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(encoded, payload[idx.TechTree.Span.Start:idx.TechTree.Span.End]) {
		t.Fatalf("tech-tree encode is not byte-identical: got %d bytes want %d", len(encoded), idx.TechTree.Span.End-idx.TechTree.Span.Start)
	}
	if len(idx.TechTree.UnitConnections) == 0 {
		t.Fatal("fixture has no tech-tree unit connections")
	}
	old := idx.TechTree.UnitConnections[0].RequiredResearch
	next := int32(-1)
	if old == -1 {
		next = 0
	}
	units := append([]int32(nil), idx.TechTree.UnitConnections[0].Units...)
	units = append(units, 9999)
	patched, report, err := PatchRecipe(compressed, Recipe{
		TechTree: &TechTreePatchRecipe{
			UnitConnections: []TechTreeUnitConnectionPatch{
				{Index: 0, RequiredResearch: int32Ptr(next), Units: int32SlicePtr(units)},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !report.Verified || !report.PatchedTechTree {
		t.Fatalf("tech-tree patch report = %+v", report)
	}
	patchedPayload, err := datfile.Inflate(patched)
	if err != nil {
		t.Fatal(err)
	}
	patchedIdx, err := datfile.Parse(patchedPayload)
	if err != nil {
		t.Fatal(err)
	}
	if got := patchedIdx.TechTree.UnitConnections[0].RequiredResearch; got != next {
		t.Fatalf("patched required_research got %d want %d", got, next)
	}
	if got := patchedIdx.TechTree.UnitConnections[0].Units; !int32SlicesEqual(got, units) {
		t.Fatalf("patched unit connection units got %+v want %+v", got, units)
	}
}

func TestCodecRecipeCreateTechTreeUnitConnection(t *testing.T) {
	root := os.Getenv("AOE2KIT_FIXTURE_ROOT")
	if root == "" {
		t.Skip("AOE2KIT_FIXTURE_ROOT not set")
	}
	path := filepath.Join(root, "reference_aoe2_dump", "dat", "empires2_x2_p1.dat")
	compressed, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("fixture dat missing: %v", err)
	}
	payload, err := datfile.Inflate(compressed)
	if err != nil {
		t.Fatal(err)
	}
	before, err := datfile.Parse(payload)
	if err != nil {
		t.Fatal(err)
	}
	if len(before.TechTree.UnitConnections) == 0 {
		t.Fatal("fixture has no tech-tree unit connections")
	}
	id := int32(83)
	status := uint8(1)
	upperBuilding := int32(109)
	verticalLine := int32(3)
	units := []int32{id}
	locationInAge := int32(2)
	requiredResearch := int32(-1)
	lineMode := int32(1)
	enablingResearch := int32(-1)
	patched, report, err := PatchRecipe(compressed, Recipe{
		TechTree: &TechTreePatchRecipe{
			CreateUnitConnections: []TechTreeUnitConnectionCreate{
				{
					From:             0,
					ID:               &id,
					Status:           &status,
					UpperBuilding:    &upperBuilding,
					VerticalLine:     &verticalLine,
					Units:            &units,
					LocationInAge:    &locationInAge,
					RequiredResearch: &requiredResearch,
					LineMode:         &lineMode,
					EnablingResearch: &enablingResearch,
				},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !report.Verified || !report.PatchedTechTree {
		t.Fatalf("create unit connection report = %+v", report)
	}
	if len(report.CreatedTechTreeUnitConnections) != 1 || report.CreatedTechTreeUnitConnections[0] != len(before.TechTree.UnitConnections) {
		t.Fatalf("created unit connections = %+v, before count %d", report.CreatedTechTreeUnitConnections, len(before.TechTree.UnitConnections))
	}
	patchedPayload, err := datfile.Inflate(patched)
	if err != nil {
		t.Fatal(err)
	}
	after, err := datfile.Parse(patchedPayload)
	if err != nil {
		t.Fatal(err)
	}
	if len(after.TechTree.UnitConnections) != len(before.TechTree.UnitConnections)+1 {
		t.Fatalf("unit connection count got %d want %d", len(after.TechTree.UnitConnections), len(before.TechTree.UnitConnections)+1)
	}
	created := after.TechTree.UnitConnections[len(before.TechTree.UnitConnections)]
	if created.Index != len(before.TechTree.UnitConnections) ||
		created.ID != id ||
		created.Status != status ||
		created.UpperBuilding != upperBuilding ||
		created.VerticalLine != verticalLine ||
		!int32SlicesEqual(created.Units, units) ||
		created.LocationInAge != locationInAge ||
		created.RequiredResearch != requiredResearch ||
		created.LineMode != lineMode ||
		created.EnablingResearch != enablingResearch {
		t.Fatalf("created unit connection = %+v", created)
	}
}

func TestCodecRecipeCreateTechTreeBuildingAndResearchConnections(t *testing.T) {
	root := os.Getenv("AOE2KIT_FIXTURE_ROOT")
	if root == "" {
		t.Skip("AOE2KIT_FIXTURE_ROOT not set")
	}
	path := filepath.Join(root, "reference_aoe2_dump", "dat", "empires2_x2_p1.dat")
	compressed, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("fixture dat missing: %v", err)
	}
	payload, err := datfile.Inflate(compressed)
	if err != nil {
		t.Fatal(err)
	}
	before, err := datfile.Parse(payload)
	if err != nil {
		t.Fatal(err)
	}
	if len(before.TechTree.BuildingConnections) == 0 || len(before.TechTree.ResearchConnections) == 0 {
		t.Fatalf("fixture tech-tree connections missing: buildings=%d researches=%d", len(before.TechTree.BuildingConnections), len(before.TechTree.ResearchConnections))
	}
	buildingID := int32(109)
	researchID := int32(101)
	status := uint8(1)
	buildingUnits := []int32{83}
	buildingTechs := []int32{researchID}
	locationByte := uint8(2)
	total := []uint8{1, 2, 3, 4, 5}
	first := []uint8{5, 4, 3, 2, 1}
	lineMode := int32(1)
	enablingResearch := int32(-1)
	upperBuilding := int32(109)
	researchBuildings := []int32{109}
	researchUnits := []int32{83}
	researchTechs := []int32{researchID}
	verticalLine := int32(4)
	locationInt := int32(2)
	patched, report, err := PatchRecipe(compressed, Recipe{
		TechTree: &TechTreePatchRecipe{
			CreateBuildingConnections: []TechTreeBuildingConnectionCreate{
				{
					From:             0,
					ID:               &buildingID,
					Status:           &status,
					Units:            &buildingUnits,
					Techs:            &buildingTechs,
					LocationInAge:    &locationByte,
					UnitsTechsTotal:  &total,
					UnitsTechsFirst:  &first,
					LineMode:         &lineMode,
					EnablingResearch: &enablingResearch,
				},
			},
			CreateResearchConnections: []TechTreeResearchConnectionCreate{
				{
					From:          0,
					ID:            &researchID,
					Status:        &status,
					UpperBuilding: &upperBuilding,
					Buildings:     &researchBuildings,
					Units:         &researchUnits,
					Techs:         &researchTechs,
					VerticalLine:  &verticalLine,
					LocationInAge: &locationInt,
					LineMode:      &lineMode,
				},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !report.Verified || !report.PatchedTechTree {
		t.Fatalf("create building/research connection report = %+v", report)
	}
	if len(report.CreatedTechTreeBuildingConnections) != 1 || report.CreatedTechTreeBuildingConnections[0] != len(before.TechTree.BuildingConnections) {
		t.Fatalf("created building connections = %+v, before count %d", report.CreatedTechTreeBuildingConnections, len(before.TechTree.BuildingConnections))
	}
	if len(report.CreatedTechTreeResearchConnections) != 1 || report.CreatedTechTreeResearchConnections[0] != len(before.TechTree.ResearchConnections) {
		t.Fatalf("created research connections = %+v, before count %d", report.CreatedTechTreeResearchConnections, len(before.TechTree.ResearchConnections))
	}
	patchedPayload, err := datfile.Inflate(patched)
	if err != nil {
		t.Fatal(err)
	}
	after, err := datfile.Parse(patchedPayload)
	if err != nil {
		t.Fatal(err)
	}
	if len(after.TechTree.BuildingConnections) != len(before.TechTree.BuildingConnections)+1 {
		t.Fatalf("building connection count got %d want %d", len(after.TechTree.BuildingConnections), len(before.TechTree.BuildingConnections)+1)
	}
	if len(after.TechTree.ResearchConnections) != len(before.TechTree.ResearchConnections)+1 {
		t.Fatalf("research connection count got %d want %d", len(after.TechTree.ResearchConnections), len(before.TechTree.ResearchConnections)+1)
	}
	createdBuilding := after.TechTree.BuildingConnections[len(before.TechTree.BuildingConnections)]
	if createdBuilding.Index != len(before.TechTree.BuildingConnections) ||
		createdBuilding.ID != buildingID ||
		createdBuilding.Status != status ||
		!int32SlicesEqual(createdBuilding.Units, buildingUnits) ||
		!int32SlicesEqual(createdBuilding.Techs, buildingTechs) ||
		createdBuilding.LocationInAge != locationByte ||
		!uint8SlicesEqual(createdBuilding.UnitsTechsTotal, total) ||
		!uint8SlicesEqual(createdBuilding.UnitsTechsFirst, first) ||
		createdBuilding.LineMode != lineMode ||
		createdBuilding.EnablingResearch != enablingResearch {
		t.Fatalf("created building connection = %+v", createdBuilding)
	}
	createdResearch := after.TechTree.ResearchConnections[len(before.TechTree.ResearchConnections)]
	if createdResearch.Index != len(before.TechTree.ResearchConnections) ||
		createdResearch.ID != researchID ||
		createdResearch.Status != status ||
		createdResearch.UpperBuilding != upperBuilding ||
		!int32SlicesEqual(createdResearch.Buildings, researchBuildings) ||
		!int32SlicesEqual(createdResearch.Units, researchUnits) ||
		!int32SlicesEqual(createdResearch.Techs, researchTechs) ||
		createdResearch.VerticalLine != verticalLine ||
		createdResearch.LocationInAge != locationInt ||
		createdResearch.LineMode != lineMode {
		t.Fatalf("created research connection = %+v", createdResearch)
	}
}

func TestCodecRecipeDeleteTechTreeConnections(t *testing.T) {
	root := os.Getenv("AOE2KIT_FIXTURE_ROOT")
	if root == "" {
		t.Skip("AOE2KIT_FIXTURE_ROOT not set")
	}
	path := filepath.Join(root, "reference_aoe2_dump", "dat", "empires2_x2_p1.dat")
	compressed, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("fixture dat missing: %v", err)
	}
	payload, err := datfile.Inflate(compressed)
	if err != nil {
		t.Fatal(err)
	}
	before, err := datfile.Parse(payload)
	if err != nil {
		t.Fatal(err)
	}
	if len(before.TechTree.BuildingConnections) < 3 || len(before.TechTree.UnitConnections) < 3 || len(before.TechTree.ResearchConnections) < 3 {
		t.Fatalf("fixture needs at least 3 tech-tree rows per family: buildings=%d units=%d researches=%d",
			len(before.TechTree.BuildingConnections), len(before.TechTree.UnitConnections), len(before.TechTree.ResearchConnections))
	}
	patched, report, err := PatchRecipe(compressed, Recipe{
		TechTree: &TechTreePatchRecipe{
			DeleteBuildingConnections: []int{1},
			DeleteUnitConnections:     []int{1},
			DeleteResearchConnections: []int{1},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !report.Verified || !report.PatchedTechTree {
		t.Fatalf("delete connection report = %+v", report)
	}
	if len(report.DeletedTechTreeBuildingConnections) != 1 || report.DeletedTechTreeBuildingConnections[0] != 1 {
		t.Fatalf("deleted building connections = %+v", report.DeletedTechTreeBuildingConnections)
	}
	if len(report.DeletedTechTreeUnitConnections) != 1 || report.DeletedTechTreeUnitConnections[0] != 1 {
		t.Fatalf("deleted unit connections = %+v", report.DeletedTechTreeUnitConnections)
	}
	if len(report.DeletedTechTreeResearchConnections) != 1 || report.DeletedTechTreeResearchConnections[0] != 1 {
		t.Fatalf("deleted research connections = %+v", report.DeletedTechTreeResearchConnections)
	}
	patchedPayload, err := datfile.Inflate(patched)
	if err != nil {
		t.Fatal(err)
	}
	after, err := datfile.Parse(patchedPayload)
	if err != nil {
		t.Fatal(err)
	}
	if len(after.TechTree.BuildingConnections) != len(before.TechTree.BuildingConnections)-1 {
		t.Fatalf("building connection count got %d want %d", len(after.TechTree.BuildingConnections), len(before.TechTree.BuildingConnections)-1)
	}
	if len(after.TechTree.UnitConnections) != len(before.TechTree.UnitConnections)-1 {
		t.Fatalf("unit connection count got %d want %d", len(after.TechTree.UnitConnections), len(before.TechTree.UnitConnections)-1)
	}
	if len(after.TechTree.ResearchConnections) != len(before.TechTree.ResearchConnections)-1 {
		t.Fatalf("research connection count got %d want %d", len(after.TechTree.ResearchConnections), len(before.TechTree.ResearchConnections)-1)
	}
	if after.TechTree.BuildingConnections[1].ID != before.TechTree.BuildingConnections[2].ID ||
		after.TechTree.UnitConnections[1].ID != before.TechTree.UnitConnections[2].ID ||
		after.TechTree.ResearchConnections[1].ID != before.TechTree.ResearchConnections[2].ID {
		t.Fatal("expected row after deleted index to slide into the deleted slot")
	}
}

func TestTechTreeSemanticRewrites(t *testing.T) {
	root := os.Getenv("AOE2KIT_FIXTURE_ROOT")
	if root == "" {
		t.Skip("AOE2KIT_FIXTURE_ROOT not set")
	}
	path := filepath.Join(root, "reference_aoe2_dump", "dat", "empires2_x2_p1.dat")
	compressed, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("fixture dat missing: %v", err)
	}
	payload, err := datfile.Inflate(compressed)
	if err != nil {
		t.Fatal(err)
	}
	idx, err := datfile.Parse(payload)
	if err != nil {
		t.Fatal(err)
	}
	unitFrom := firstTechTreeUnitRef(t, idx)
	unitTo := int32(9999)
	beforeUnitRefs, err := countTechTreeRewriteTarget(idx.TechTree, "unit", unitFrom)
	if err != nil {
		t.Fatal(err)
	}
	techFrom := firstTechTreeTechRef(t, idx)
	beforeTechRefs, err := countTechTreeRewriteTarget(idx.TechTree, "tech", techFrom)
	if err != nil {
		t.Fatal(err)
	}
	patched, report, err := PatchRecipe(compressed, Recipe{
		TechTree: &TechTreePatchRecipe{
			Rewrites: []TechTreeIDRewriteRecipe{
				{Target: "unit", From: unitFrom, To: int32Ptr(unitTo)},
				{Target: "tech", From: techFrom, Remove: true},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !report.Verified || !report.PatchedTechTree || len(report.TechTreeRewrites) != 2 {
		t.Fatalf("tech-tree rewrite report = %+v", report)
	}
	if report.TechTreeRewrites[0].ReplacedScalars+report.TechTreeRewrites[0].ReplacedListValues != beforeUnitRefs {
		t.Fatalf("unit rewrite report = %+v beforeRefs=%d", report.TechTreeRewrites[0], beforeUnitRefs)
	}
	if report.TechTreeRewrites[1].ReplacedScalars+report.TechTreeRewrites[1].ReplacedListValues+report.TechTreeRewrites[1].RemovedListValues != beforeTechRefs {
		t.Fatalf("tech rewrite report = %+v beforeRefs=%d", report.TechTreeRewrites[1], beforeTechRefs)
	}
	patchedPayload, err := datfile.Inflate(patched)
	if err != nil {
		t.Fatal(err)
	}
	patchedIdx, err := datfile.Parse(patchedPayload)
	if err != nil {
		t.Fatal(err)
	}
	if remaining, err := countTechTreeRewriteTarget(patchedIdx.TechTree, "unit", unitFrom); err != nil || remaining != 0 {
		t.Fatalf("unit from=%d remaining=%d err=%v", unitFrom, remaining, err)
	}
	if remaining, err := countTechTreeRewriteTarget(patchedIdx.TechTree, "tech", techFrom); err != nil || remaining != 0 {
		t.Fatalf("tech from=%d remaining=%d err=%v", techFrom, remaining, err)
	}
}

func TestReferenceRewrites(t *testing.T) {
	root := os.Getenv("AOE2KIT_FIXTURE_ROOT")
	if root == "" {
		t.Skip("AOE2KIT_FIXTURE_ROOT not set")
	}
	path := filepath.Join(root, "reference_aoe2_dump", "dat", "empires2_x2_p1.dat")
	compressed, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("fixture dat missing: %v", err)
	}
	payload, err := datfile.Inflate(compressed)
	if err != nil {
		t.Fatal(err)
	}
	idx, err := datfile.Parse(payload)
	if err != nil {
		t.Fatal(err)
	}
	techFrom := int32(244)
	beforeEffectRefs := countTypedEffectCommandRefs(idx.Effects, "tech", techFrom)
	if beforeEffectRefs == 0 {
		t.Fatalf("fixture has no typed effect-command refs for tech %d", techFrom)
	}
	unitFrom := firstTechTreeUnitRef(t, idx)
	unitTo := int32(9999)
	patched, report, err := PatchRecipe(compressed, Recipe{
		ReferenceRewrites: []TechTreeIDRewriteRecipe{
			{Target: "tech", From: techFrom, Remove: true},
			{Target: "unit", From: unitFrom, To: int32Ptr(unitTo)},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !report.Verified || !report.PatchedTechTree || len(report.ReferenceRewrites) != 2 {
		t.Fatalf("reference rewrite report = %+v", report)
	}
	if report.ReferenceRewrites[0].EffectCommandsRemoved != beforeEffectRefs {
		t.Fatalf("tech rewrite removed effect commands=%d want %d report=%+v", report.ReferenceRewrites[0].EffectCommandsRemoved, beforeEffectRefs, report.ReferenceRewrites[0])
	}
	patchedPayload, err := datfile.Inflate(patched)
	if err != nil {
		t.Fatal(err)
	}
	patchedIdx, err := datfile.Parse(patchedPayload)
	if err != nil {
		t.Fatal(err)
	}
	if remaining := countTypedEffectCommandRefs(patchedIdx.Effects, "tech", techFrom); remaining != 0 {
		t.Fatalf("tech %d still has %d typed effect-command refs", techFrom, remaining)
	}
	if remaining := countTechRequiredRefs(patchedIdx.Techs, techFrom); remaining != 0 {
		t.Fatalf("tech %d still has %d required-tech refs", techFrom, remaining)
	}
	if remaining, err := countTechTreeRewriteTarget(patchedIdx.TechTree, "tech", techFrom); err != nil || remaining != 0 {
		t.Fatalf("tech %d remaining tech-tree refs=%d err=%v", techFrom, remaining, err)
	}
	if remaining, err := countTechTreeRewriteTarget(patchedIdx.TechTree, "unit", unitFrom); err != nil || remaining != 0 {
		t.Fatalf("unit %d remaining tech-tree refs=%d err=%v", unitFrom, remaining, err)
	}
}

func TestReferenceRewriteUpgradeUnitTarget(t *testing.T) {
	root := os.Getenv("AOE2KIT_FIXTURE_ROOT")
	if root == "" {
		t.Skip("AOE2KIT_FIXTURE_ROOT not set")
	}
	path := filepath.Join(root, "reference_aoe2_dump", "dat", "empires2_x2_p1.dat")
	compressed, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("fixture dat missing: %v", err)
	}
	payload, err := datfile.Inflate(compressed)
	if err != nil {
		t.Fatal(err)
	}
	before, err := datfile.Parse(payload)
	if err != nil {
		t.Fatal(err)
	}
	newEffectID := int16(len(before.Effects))
	patched, report, err := PatchRecipe(compressed, Recipe{
		CreateEffect: &EffectCreateRecipe{
			Name: "AoE2Kit Upgrade Unit Rewrite Effect",
			Commands: []EffectCommandRecipe{
				{Kind: "upgrade_unit", UnitID: int16Ptr(74), ToUnitID: int16Ptr(75)},
			},
		},
		ReferenceRewrites: []TechTreeIDRewriteRecipe{
			{Target: "unit", From: 75, To: int32Ptr(77)},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !report.Verified || len(report.ReferenceRewrites) != 1 || report.ReferenceRewrites[0].EffectCommandsRewritten == 0 {
		t.Fatalf("upgrade target rewrite report = %+v", report)
	}
	patchedPayload, err := datfile.Inflate(patched)
	if err != nil {
		t.Fatal(err)
	}
	after, err := datfile.Parse(patchedPayload)
	if err != nil {
		t.Fatal(err)
	}
	command := after.Effects[newEffectID].Commands[0]
	if command.Type != 3 || command.A != 74 || command.B != 77 ||
		command.Semantic == nil ||
		!semanticHasRef(command.Semantic, "unit", "source_unit", 74) ||
		!semanticHasRef(command.Semantic, "unit", "target_unit", 77) {
		t.Fatalf("rewritten upgrade_unit command = %+v", command)
	}
}

func TestEffectCommandSemanticsPromoteXSConstants(t *testing.T) {
	resource := datfile.InterpretEffectCommand(datfile.EffectCommand{Type: 1, A: 0, B: 1, C: -1, D: 50})
	if resource.TypeName != "resource_modifier" ||
		resource.ResourceID == nil || *resource.ResourceID != 0 || resource.ResourceName != "food" ||
		resource.OperationID == nil || *resource.OperationID != 1 || resource.OperationName != "add" {
		t.Fatalf("resource modifier semantics = %+v", resource)
	}
	resourceMultiplier := datfile.InterpretEffectCommand(datfile.EffectCommand{Type: 6, A: 0, B: 0, C: -1, D: 1.5})
	if resourceMultiplier.TypeName != "resource_multiplier" ||
		resourceMultiplier.ResourceID == nil || *resourceMultiplier.ResourceID != 0 || resourceMultiplier.ResourceName != "food" {
		t.Fatalf("resource multiplier semantics = %+v", resourceMultiplier)
	}
	spawn := datfile.InterpretEffectCommand(datfile.EffectCommand{Type: 7, A: 83, B: 109, C: -1, D: 5})
	if spawn.TypeName != "spawn_unit" ||
		!semanticHasRef(spawn, "unit", "spawn_unit", 83) ||
		!semanticHasRef(spawn, "unit", "spawn_building", 109) {
		t.Fatalf("spawn semantics = %+v", spawn)
	}
	modifyTech := datfile.InterpretEffectCommand(datfile.EffectCommand{Type: 8, A: 22, B: 0, C: -1, D: 10})
	if modifyTech.TypeName != "modify_tech" ||
		modifyTech.TechAttributeID == nil || *modifyTech.TechAttributeID != 0 || modifyTech.TechAttributeName != "set_food_cost" ||
		!semanticHasRef(modifyTech, "tech", "target_tech", 22) {
		t.Fatalf("modify tech semantics = %+v", modifyTech)
	}
	setUnitAttribute := datfile.InterpretEffectCommand(datfile.EffectCommand{Type: 0, A: 74, B: -1, C: 0, D: 40})
	if setUnitAttribute.TypeName != "set_attribute" || setUnitAttribute.AttributeID != 0 || setUnitAttribute.AttributeName != "hit_points" || !semanticHasRef(setUnitAttribute, "unit", "target_unit", 74) {
		t.Fatalf("set unit attribute semantics = %+v", setUnitAttribute)
	}
	addAttribute := datfile.InterpretEffectCommand(datfile.EffectCommand{Type: 4, A: 83, B: 912, C: 9, D: 3})
	if addAttribute.TypeName != "add_attribute" ||
		addAttribute.UnitClassID != 912 || addAttribute.UnitClassName != "cavalry" ||
		addAttribute.AttributeID != 9 || addAttribute.AttributeName != "attack" ||
		!semanticHasRef(addAttribute, "unit", "target_unit", 83) {
		t.Fatalf("add attribute semantics = %+v", addAttribute)
	}
	teamMultiply := datfile.InterpretEffectCommand(datfile.EffectCommand{Type: 15, A: 83, B: -1, C: 9, D: 3})
	if teamMultiply.TypeName != "team_attribute_multiplier" || teamMultiply.Scope != "team" ||
		teamMultiply.AttributeID != 9 || teamMultiply.AttributeName != "attack" ||
		!semanticHasRef(teamMultiply, "unit", "target_unit", 83) {
		t.Fatalf("team multiply semantics = %+v", teamMultiply)
	}
	enemyResource := datfile.InterpretEffectCommand(datfile.EffectCommand{Type: 26, A: 7, B: 0, C: -1, D: 0.5})
	if enemyResource.TypeName != "enemy_resource_multiplier" || enemyResource.Scope != "enemy" ||
		enemyResource.ResourceID == nil || *enemyResource.ResourceID != 7 || enemyResource.ResourceName != "relics" {
		t.Fatalf("enemy resource semantics = %+v", enemyResource)
	}
	teamModifyTech := datfile.InterpretEffectCommand(datfile.EffectCommand{Type: 18, A: 22, B: 5, C: -1, D: 3})
	if teamModifyTech.TypeName != "team_modify_tech" || teamModifyTech.Scope != "team" ||
		teamModifyTech.TechAttributeID == nil || *teamModifyTech.TechAttributeID != 5 || teamModifyTech.TechAttributeName != "set_button" ||
		!semanticHasRef(teamModifyTech, "tech", "target_tech", 22) {
		t.Fatalf("team modify tech semantics = %+v", teamModifyTech)
	}
	packedAttack := datfile.InterpretEffectCommand(datfile.EffectCommand{Type: 4, A: 83, B: -1, C: 9, D: 1027})
	if packedAttack.PackedTypeID == nil || *packedAttack.PackedTypeID != 4 ||
		packedAttack.PackedAmount == nil || *packedAttack.PackedAmount != 3 {
		t.Fatalf("packed attack semantics = %+v", packedAttack)
	}
	localPackedArmor := datfile.InterpretEffectCommand(datfile.EffectCommand{Type: 201, A: 82, B: -1, C: 8, D: 770})
	if localPackedArmor.TypeName != "add_local_building_attribute" ||
		localPackedArmor.Scope != "local_building" ||
		localPackedArmor.PackedTypeID == nil || *localPackedArmor.PackedTypeID != 3 ||
		localPackedArmor.PackedAmount == nil || *localPackedArmor.PackedAmount != 2 ||
		!semanticHasRef(localPackedArmor, "unit", "local_building_unit", 82) {
		t.Fatalf("local packed armor semantics = %+v", localPackedArmor)
	}
	localMultiply := datfile.InterpretEffectCommand(datfile.EffectCommand{Type: 202, A: 82, B: -1, C: 108, D: 6})
	if localMultiply.TypeName != "multiply_local_building_attribute" ||
		localMultiply.Scope != "local_building" ||
		localMultiply.AttributeID != 108 || localMultiply.AttributeName != "garrison_heal_rate" ||
		!semanticHasRef(localMultiply, "unit", "local_building_unit", 82) {
		t.Fatalf("local multiply semantics = %+v", localMultiply)
	}
	localAdvancedPackedAttack := datfile.InterpretEffectCommand(datfile.EffectCommand{Type: 204, A: 82, B: -1, C: 9, D: 771})
	if localAdvancedPackedAttack.TypeName != "add_local_building_attribute_advanced" ||
		localAdvancedPackedAttack.Scope != "local_building" ||
		localAdvancedPackedAttack.PackedTypeID == nil || *localAdvancedPackedAttack.PackedTypeID != 3 ||
		localAdvancedPackedAttack.PackedAmount == nil || *localAdvancedPackedAttack.PackedAmount != 3 ||
		!semanticHasRef(localAdvancedPackedAttack, "unit", "local_building_unit", 82) {
		t.Fatalf("local advanced packed attack semantics = %+v", localAdvancedPackedAttack)
	}
	addTechCost := datfile.InterpretEffectCommand(datfile.EffectCommand{Type: 101, A: 22, B: 0, C: -1, D: 10})
	if addTechCost.TypeName != "tech_cost_modifier" ||
		addTechCost.ResourceID == nil || *addTechCost.ResourceID != 0 || addTechCost.ResourceName != "food" ||
		!semanticHasRef(addTechCost, "tech", "target_tech", 22) {
		t.Fatalf("add tech cost semantics = %+v", addTechCost)
	}
	modTechTime := datfile.InterpretEffectCommand(datfile.EffectCommand{Type: 103, A: 22, B: 0, C: -1, D: 10})
	if modTechTime.TypeName != "tech_time_modifier" ||
		modTechTime.OperationID == nil || *modTechTime.OperationID != 0 || modTechTime.OperationName != "set" ||
		!semanticHasRef(modTechTime, "tech", "target_tech", 22) {
		t.Fatalf("mod tech time semantics = %+v", modTechTime)
	}
	renameUnit := datfile.InterpretEffectCommand(datfile.EffectCommand{Type: 40, A: 2084, B: -1, C: 50, D: 21220})
	if renameUnit.TypeName != "gaia_set_attribute" ||
		renameUnit.AttributeID != 50 || renameUnit.AttributeName != "name_id" ||
		renameUnit.StringID == nil || *renameUnit.StringID != 21220 ||
		!semanticHasRef(renameUnit, "unit", "target_unit", 2084) {
		t.Fatalf("rename unit semantics = %+v", renameUnit)
	}
}

func TestEffectCommandMatrixPromotedOperandProfiles(t *testing.T) {
	resource := datfile.EffectCommand{Index: 0, Type: 1, A: 0, B: 1, C: -1, D: 50}
	resource.Semantic = datfile.InterpretEffectCommand(resource)
	modifyTech := datfile.EffectCommand{Index: 1, Type: 8, A: 22, B: 3, C: -1, D: 10}
	modifyTech.Semantic = datfile.InterpretEffectCommand(modifyTech)
	idx := &datfile.Index{
		Effects: []datfile.Effect{
			{Index: 0, Name: "matrix probe", Commands: []datfile.EffectCommand{resource, modifyTech}},
		},
		Techs: []datfile.Tech{{Index: 22, Name: "probe tech"}},
	}
	report := EffectCommandMatrix(idx, 0)
	if report.CommandCount != 2 || report.TypeCount != 2 {
		t.Fatalf("matrix summary = %+v", report)
	}
	byType := map[uint8]EffectCommandTypeProfile{}
	for _, profile := range report.CommandTypes {
		byType[profile.Type] = profile
	}
	resourceProfile := byType[1]
	if len(resourceProfile.ResourceIDs) != 1 || resourceProfile.ResourceIDs[0].Value != 0 || resourceProfile.ResourceIDs[0].Name != "food" ||
		len(resourceProfile.OperationIDs) != 1 || resourceProfile.OperationIDs[0].Value != 1 || resourceProfile.OperationIDs[0].Name != "add" {
		t.Fatalf("resource profile = %+v", resourceProfile)
	}
	modifyTechProfile := byType[8]
	if len(modifyTechProfile.TechAttributeIDs) != 1 ||
		modifyTechProfile.TechAttributeIDs[0].Value != 3 || modifyTechProfile.TechAttributeIDs[0].Name != "set_gold_cost" ||
		!referenceCountsContain(modifyTechProfile.TypedReferences, "tech", "target_tech", 22) {
		t.Fatalf("modify tech profile = %+v", modifyTechProfile)
	}
}

func TestReferenceRewriteModifyTechTarget(t *testing.T) {
	root := os.Getenv("AOE2KIT_FIXTURE_ROOT")
	if root == "" {
		t.Skip("AOE2KIT_FIXTURE_ROOT not set")
	}
	path := filepath.Join(root, "reference_aoe2_dump", "dat", "empires2_x2_p1.dat")
	compressed, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("fixture dat missing: %v", err)
	}
	payload, err := datfile.Inflate(compressed)
	if err != nil {
		t.Fatal(err)
	}
	before, err := datfile.Parse(payload)
	if err != nil {
		t.Fatal(err)
	}
	newEffectID := int16(len(before.Effects))
	fromTech := int32(244)
	toTech := int32(245)
	patched, report, err := PatchRecipe(compressed, Recipe{
		CreateEffect: &EffectCreateRecipe{
			Name: "AoE2Kit Modify Tech Rewrite Effect",
			Commands: []EffectCommandRecipe{
				{Type: 8, A: int16(fromTech), B: 0, C: -1, D: 10},
			},
		},
		ReferenceRewrites: []TechTreeIDRewriteRecipe{
			{Target: "tech", From: fromTech, To: &toTech},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !report.Verified || len(report.ReferenceRewrites) != 1 || report.ReferenceRewrites[0].EffectCommandsRewritten == 0 {
		t.Fatalf("modify tech rewrite report = %+v", report)
	}
	afterPayload, err := datfile.Inflate(patched)
	if err != nil {
		t.Fatal(err)
	}
	after, err := datfile.Parse(afterPayload)
	if err != nil {
		t.Fatal(err)
	}
	command := after.Effects[newEffectID].Commands[0]
	if command.Type != 8 || command.A != int16(toTech) ||
		command.Semantic == nil ||
		!semanticHasRef(command.Semantic, "tech", "target_tech", int(toTech)) {
		t.Fatalf("rewritten modify_tech command = %+v", command)
	}
}

func TestDisconnectTechIntent(t *testing.T) {
	root := os.Getenv("AOE2KIT_FIXTURE_ROOT")
	if root == "" {
		t.Skip("AOE2KIT_FIXTURE_ROOT not set")
	}
	path := filepath.Join(root, "reference_aoe2_dump", "dat", "empires2_x2_p1.dat")
	compressed, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("fixture dat missing: %v", err)
	}
	payload, err := datfile.Inflate(compressed)
	if err != nil {
		t.Fatal(err)
	}
	idx, err := datfile.Parse(payload)
	if err != nil {
		t.Fatal(err)
	}
	techID := 244
	beforeRefs := References(idx, DeletePlanRequest{Section: "tech", ID: techID})
	if beforeRefs.Summary.RewriteSupported == 0 {
		t.Fatalf("fixture tech %d has no rewrite-supported refs: %+v", techID, beforeRefs.Summary)
	}

	patched, report, err := PatchRecipe(compressed, Recipe{DisconnectTechs: []int{techID}})
	if err != nil {
		t.Fatal(err)
	}
	if !report.Verified || !report.PatchedTechTree || len(report.ReferenceRewrites) != 1 {
		t.Fatalf("disconnect report = %+v", report)
	}
	if len(report.DisconnectedTechs) != 1 {
		t.Fatalf("disconnect report missing tech summary: %+v", report)
	}
	if report.DisconnectedTechs[0].TechID != techID || report.DisconnectedTechs[0].ResidualSummary.RewriteSupported != 0 {
		t.Fatalf("unexpected residual rewrite-supported refs: %+v", report.DisconnectedTechs[0])
	}

	patchedPayload, err := datfile.Inflate(patched)
	if err != nil {
		t.Fatal(err)
	}
	patchedIdx, err := datfile.Parse(patchedPayload)
	if err != nil {
		t.Fatal(err)
	}
	if remaining := countTypedEffectCommandRefs(patchedIdx.Effects, "tech", int32(techID)); remaining != 0 {
		t.Fatalf("tech %d still has %d typed effect-command refs", techID, remaining)
	}
	if remaining := countTechRequiredRefs(patchedIdx.Techs, int32(techID)); remaining != 0 {
		t.Fatalf("tech %d still has %d required-tech refs", techID, remaining)
	}
	if remaining, err := countTechTreeRewriteTarget(patchedIdx.TechTree, "tech", int32(techID)); err != nil || remaining != 0 {
		t.Fatalf("tech %d remaining tech-tree refs=%d err=%v", techID, remaining, err)
	}
}

func TestDisconnectUnitIntent(t *testing.T) {
	root := os.Getenv("AOE2KIT_FIXTURE_ROOT")
	if root == "" {
		t.Skip("AOE2KIT_FIXTURE_ROOT not set")
	}
	path := filepath.Join(root, "reference_aoe2_dump", "dat", "empires2_x2_p1.dat")
	compressed, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("fixture dat missing: %v", err)
	}
	payload, err := datfile.Inflate(compressed)
	if err != nil {
		t.Fatal(err)
	}
	idx, err := datfile.Parse(payload)
	if err != nil {
		t.Fatal(err)
	}
	unitID := int(firstTechTreeUnitRef(t, idx))
	beforeTreeRefs, err := countTechTreeRewriteTarget(idx.TechTree, "unit", int32(unitID))
	if err != nil {
		t.Fatal(err)
	}
	if beforeTreeRefs == 0 {
		t.Fatalf("fixture unit %d has no tech-tree refs", unitID)
	}

	patched, report, err := PatchRecipe(compressed, Recipe{DisconnectUnits: []int{unitID}})
	if err != nil {
		t.Fatal(err)
	}
	if !report.Verified || !report.PatchedTechTree || len(report.ReferenceRewrites) != 1 {
		t.Fatalf("disconnect unit report = %+v", report)
	}
	if len(report.DisconnectedUnits) != 1 || report.DisconnectedUnits[0].UnitID != unitID {
		t.Fatalf("disconnect unit summary = %+v", report.DisconnectedUnits)
	}

	patchedPayload, err := datfile.Inflate(patched)
	if err != nil {
		t.Fatal(err)
	}
	patchedIdx, err := datfile.Parse(patchedPayload)
	if err != nil {
		t.Fatal(err)
	}
	if remaining, err := countTechTreeRewriteTarget(patchedIdx.TechTree, "unit", int32(unitID)); err != nil || remaining != 0 {
		t.Fatalf("unit %d remaining tech-tree refs=%d err=%v", unitID, remaining, err)
	}
	if remaining := countTypedEffectCommandRefs(patchedIdx.Effects, "unit", int32(unitID)); remaining != 0 {
		t.Fatalf("unit %d still has %d typed effect-command refs", unitID, remaining)
	}
}

func TestDisconnectUnitIntentRewritesUnitHeaderTasks(t *testing.T) {
	root := os.Getenv("AOE2KIT_FIXTURE_ROOT")
	if root == "" {
		t.Skip("AOE2KIT_FIXTURE_ROOT not set")
	}
	path := filepath.Join(root, "reference_aoe2_dump", "dat", "empires2_x2_p1.dat")
	compressed, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("fixture dat missing: %v", err)
	}
	payload, err := datfile.Inflate(compressed)
	if err != nil {
		t.Fatal(err)
	}
	idx, err := datfile.Parse(payload)
	if err != nil {
		t.Fatal(err)
	}
	headerID, taskID, unitID := firstUnitHeaderTaskObjectRef(t, idx)

	patched, report, err := PatchRecipe(compressed, Recipe{DisconnectUnits: []int{unitID}})
	if err != nil {
		t.Fatal(err)
	}
	if !report.Verified || !intSliceContains(report.PatchedUnitHeaders, headerID) {
		t.Fatalf("disconnect unit-header report = %+v", report)
	}

	patchedPayload, err := datfile.Inflate(patched)
	if err != nil {
		t.Fatal(err)
	}
	patchedIdx, err := datfile.Parse(patchedPayload)
	if err != nil {
		t.Fatal(err)
	}
	if got := patchedIdx.UnitHeaders[headerID].Tasks[taskID].ObjectID; got != -1 {
		t.Fatalf("unit header %d task %d object_id=%d want -1", headerID, taskID, got)
	}
}

func TestDisconnectUnitIntentRewritesUnitRecordRefs(t *testing.T) {
	root := os.Getenv("AOE2KIT_FIXTURE_ROOT")
	if root == "" {
		t.Skip("AOE2KIT_FIXTURE_ROOT not set")
	}
	path := filepath.Join(root, "reference_aoe2_dump", "dat", "empires2_x2_p1.dat")
	compressed, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("fixture dat missing: %v", err)
	}
	payload, err := datfile.Inflate(compressed)
	if err != nil {
		t.Fatal(err)
	}
	idx, err := datfile.Parse(payload)
	if err != nil {
		t.Fatal(err)
	}
	unitID := firstDropSiteReferencedUnit(t, idx)
	beforeRefs := unitIDReferences(idx, unitID)
	if !referencesFieldContaining(beforeRefs, ".drop_site[") {
		t.Fatalf("fixture unit %d has refs but no drop-site refs: %+v", unitID, beforeRefs)
	}

	patched, report, err := PatchRecipe(compressed, Recipe{DisconnectUnits: []int{unitID}})
	if err != nil {
		t.Fatal(err)
	}
	if !report.Verified {
		t.Fatalf("disconnect unit-record report = %+v", report)
	}

	patchedPayload, err := datfile.Inflate(patched)
	if err != nil {
		t.Fatal(err)
	}
	patchedIdx, err := datfile.Parse(patchedPayload)
	if err != nil {
		t.Fatal(err)
	}
	afterRefs := unitIDReferences(patchedIdx, unitID)
	if referencesFieldContaining(afterRefs, ".drop_site[") {
		t.Fatalf("unit %d still has drop-site refs after disconnect: %+v", unitID, afterRefs)
	}
	if remaining := countSupportedUnitRefs(patchedIdx, unitID); remaining != 0 {
		t.Fatalf("unit %d still has %d rewrite-supported refs after disconnect: %+v", unitID, remaining, afterRefs)
	}
}

func TestDisconnectUnitIntentRewritesBloodUnitRefs(t *testing.T) {
	root := os.Getenv("AOE2KIT_FIXTURE_ROOT")
	if root == "" {
		t.Skip("AOE2KIT_FIXTURE_ROOT not set")
	}
	path := filepath.Join(root, "reference_aoe2_dump", "dat", "empires2_x2_p1.dat")
	compressed, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("fixture dat missing: %v", err)
	}
	payload, err := datfile.Inflate(compressed)
	if err != nil {
		t.Fatal(err)
	}
	idx, err := datfile.Parse(payload)
	if err != nil {
		t.Fatal(err)
	}
	unitID, ok := firstBloodReferencedUnit(idx)
	if !ok {
		t.Skip("fixture has no blood-unit references")
	}

	patched, report, err := PatchRecipe(compressed, Recipe{DisconnectUnits: []int{unitID}})
	if err != nil {
		t.Fatal(err)
	}
	if !report.Verified {
		t.Fatalf("disconnect blood-unit report = %+v", report)
	}
	patchedPayload, err := datfile.Inflate(patched)
	if err != nil {
		t.Fatal(err)
	}
	patchedIdx, err := datfile.Parse(patchedPayload)
	if err != nil {
		t.Fatal(err)
	}
	if referencesFieldContaining(unitIDReferences(patchedIdx, unitID), ".blood_unit_id") {
		t.Fatalf("unit %d still has blood-unit refs after disconnect", unitID)
	}
}

func TestDisconnectUnitIntentRewritesLinkedBuildingRefs(t *testing.T) {
	root := os.Getenv("AOE2KIT_FIXTURE_ROOT")
	if root == "" {
		t.Skip("AOE2KIT_FIXTURE_ROOT not set")
	}
	path := filepath.Join(root, "reference_aoe2_dump", "dat", "empires2_x2_p1.dat")
	compressed, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("fixture dat missing: %v", err)
	}
	payload, err := datfile.Inflate(compressed)
	if err != nil {
		t.Fatal(err)
	}
	idx, err := datfile.Parse(payload)
	if err != nil {
		t.Fatal(err)
	}
	unitID, ok := firstLinkedBuildingReferencedUnit(idx)
	if !ok {
		t.Skip("fixture has no linked-building references")
	}

	patched, report, err := PatchRecipe(compressed, Recipe{DisconnectUnits: []int{unitID}})
	if err != nil {
		t.Fatal(err)
	}
	if !report.Verified {
		t.Fatalf("disconnect linked-building report = %+v", report)
	}
	patchedPayload, err := datfile.Inflate(patched)
	if err != nil {
		t.Fatal(err)
	}
	patchedIdx, err := datfile.Parse(patchedPayload)
	if err != nil {
		t.Fatal(err)
	}
	if referencesFieldContaining(unitIDReferences(patchedIdx, unitID), ".linked_building[") {
		t.Fatalf("unit %d still has linked-building refs after disconnect", unitID)
	}
}

func TestCodecRecipePatchUnitHeaderTask(t *testing.T) {
	root := os.Getenv("AOE2KIT_FIXTURE_ROOT")
	if root == "" {
		t.Skip("AOE2KIT_FIXTURE_ROOT not set")
	}
	path := filepath.Join(root, "reference_aoe2_dump", "dat", "empires2_x2_p1.dat")
	compressed, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("fixture dat missing: %v", err)
	}
	payload, err := datfile.Inflate(compressed)
	if err != nil {
		t.Fatal(err)
	}
	before, err := datfile.Parse(payload)
	if err != nil {
		t.Fatal(err)
	}
	headerID := -1
	for _, header := range before.UnitHeaders {
		if len(header.Tasks) > 1 {
			headerID = header.Index
			break
		}
	}
	if headerID < 0 {
		t.Fatal("fixture has no unit header task rows")
	}
	task := before.UnitHeaders[headerID].Tasks[0]
	actionType := task.ActionType + 1
	workRange := task.WorkRange + 1.25
	enableTargeting := uint8(1)
	if task.EnableTargeting == 1 {
		enableTargeting = 0
	}
	combatLevel := task.CombatLevel + 1
	patched, report, err := PatchRecipe(compressed, Recipe{
		UnitHeaders: []UnitHeaderPatchRecipe{
			{
				ID: headerID,
				Tasks: []datfile.TaskPatch{
					{
						Index:           0,
						ActionType:      &actionType,
						WorkRange:       &workRange,
						EnableTargeting: &enableTargeting,
						CombatLevel:     &combatLevel,
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !report.Verified || !report.PayloadRoundTripOK || !report.ReadbackOK {
		t.Fatalf("unit header task patch report not verified: %+v", report)
	}
	if len(report.PatchedUnitHeaders) != 1 || report.PatchedUnitHeaders[0] != headerID {
		t.Fatalf("patched unit headers = %+v, want [%d]", report.PatchedUnitHeaders, headerID)
	}
	afterPayload, err := datfile.Inflate(patched)
	if err != nil {
		t.Fatal(err)
	}
	after, err := datfile.Parse(afterPayload)
	if err != nil {
		t.Fatal(err)
	}
	got := after.UnitHeaders[headerID].Tasks[0]
	if got.ActionType != actionType || got.WorkRange != workRange || got.EnableTargeting != enableTargeting || got.CombatLevel != combatLevel {
		t.Fatalf("patched task = %+v, want action_type=%d work_range=%g enable_targeting=%d combat_level=%d", got, actionType, workRange, enableTargeting, combatLevel)
	}

	setTasks := []datfile.TaskRow{
		taskRowFromSummary(before.UnitHeaders[headerID].Tasks[0]),
		taskRowFromSummary(before.UnitHeaders[headerID].Tasks[1]),
		taskRowFromSummary(before.UnitHeaders[headerID].Tasks[0]),
	}
	setTasks[2].ActionType += 3
	setTasks[2].WorkRange += 2
	patched, report, err = PatchRecipe(compressed, Recipe{
		UnitHeaders: []UnitHeaderPatchRecipe{
			{
				ID:       headerID,
				SetTasks: &setTasks,
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !report.Verified || !report.PayloadRoundTripOK || !report.ReadbackOK {
		t.Fatalf("unit header set_tasks report not verified: %+v", report)
	}
	afterPayload, err = datfile.Inflate(patched)
	if err != nil {
		t.Fatal(err)
	}
	after, err = datfile.Parse(afterPayload)
	if err != nil {
		t.Fatal(err)
	}
	gotHeader := after.UnitHeaders[headerID]
	if len(gotHeader.Tasks) != len(setTasks) ||
		gotHeader.Tasks[2].ActionType != setTasks[2].ActionType ||
		gotHeader.Tasks[2].WorkRange != setTasks[2].WorkRange {
		t.Fatalf("set_tasks readback = %+v want %+v", gotHeader.Tasks, setTasks)
	}
}

func TestCodecRecipePatchCivHeader(t *testing.T) {
	root := os.Getenv("AOE2KIT_FIXTURE_ROOT")
	if root == "" {
		t.Skip("AOE2KIT_FIXTURE_ROOT not set")
	}
	path := filepath.Join(root, "reference_aoe2_dump", "dat", "empires2_x2_p1.dat")
	compressed, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("fixture dat missing: %v", err)
	}
	name := "AoE2Kit Test Civ"
	techTreeID := int16(7)
	teamBonusID := int16(23)
	iconSet := uint8(2)
	patched, report, err := PatchRecipe(compressed, Recipe{
		Civs: []CivPatchRecipe{
			{
				ID:          1,
				Name:        &name,
				TechTreeID:  &techTreeID,
				TeamBonusID: &teamBonusID,
				IconSet:     &iconSet,
				Resources: []CivResourcePatch{
					{Index: 0, Value: 1234.5},
				},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !report.Verified || !report.PayloadRoundTripOK || !report.ReadbackOK {
		t.Fatalf("civ patch report not verified: %+v", report)
	}
	if report.AfterCivCount != report.BeforeCivCount || len(report.PatchedCivs) != 1 || report.PatchedCivs[0] != 1 {
		t.Fatalf("civ patch report = %+v", report)
	}
	payload, err := datfile.Inflate(patched)
	if err != nil {
		t.Fatal(err)
	}
	idx, err := datfile.Parse(payload)
	if err != nil {
		t.Fatal(err)
	}
	civ := idx.Civs[1]
	if civ.Name != name || civ.TechTreeID != techTreeID || civ.TeamBonusID != teamBonusID || civ.IconSet != iconSet {
		t.Fatalf("patched civ = %+v", civ)
	}
	if len(civ.Resources) == 0 || civ.Resources[0] != 1234.5 {
		t.Fatalf("patched civ resource[0] = %+v", civ.Resources)
	}
}

func TestCodecRecipePatchPlayerColour(t *testing.T) {
	root := os.Getenv("AOE2KIT_FIXTURE_ROOT")
	if root == "" {
		t.Skip("AOE2KIT_FIXTURE_ROOT not set")
	}
	path := filepath.Join(root, "reference_aoe2_dump", "dat", "empires2_x2_p1.dat")
	compressed, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("fixture dat missing: %v", err)
	}
	base := int32(101)
	outline := int32(102)
	selection1 := int32(103)
	selection2 := int32(104)
	minimap1 := int32(105)
	minimap2 := int32(106)
	minimap3 := int32(107)
	stats := int32(108)
	patched, report, err := PatchRecipe(compressed, Recipe{
		PlayerColours: []PlayerColourPatchRecipe{
			{
				ID:                  1,
				Base:                &base,
				UnitOutlineColour:   &outline,
				SelectionColour1:    &selection1,
				SelectionColour2:    &selection2,
				MinimapColour1:      &minimap1,
				MinimapColour2:      &minimap2,
				MinimapColour3:      &minimap3,
				StatisticsTextColor: &stats,
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !report.Verified || !report.PayloadRoundTripOK || !report.ReadbackOK {
		t.Fatalf("player colour patch report not verified: %+v", report)
	}
	if len(report.PatchedPlayerColours) != 1 || report.PatchedPlayerColours[0] != 1 {
		t.Fatalf("patched player colours = %+v", report.PatchedPlayerColours)
	}
	payload, err := datfile.Inflate(patched)
	if err != nil {
		t.Fatal(err)
	}
	idx, err := datfile.Parse(payload)
	if err != nil {
		t.Fatal(err)
	}
	colour := idx.PlayerColours[1]
	if colour.Base != base ||
		colour.UnitOutlineColour != outline ||
		colour.SelectionColour1 != selection1 ||
		colour.SelectionColour2 != selection2 ||
		colour.MinimapColour1 != minimap1 ||
		colour.MinimapColour2 != minimap2 ||
		colour.MinimapColour3 != minimap3 ||
		colour.StatisticsTextColor != stats {
		t.Fatalf("patched player colour = %+v", colour)
	}
}

func TestCodecRecipeCreatePlayerColour(t *testing.T) {
	root := os.Getenv("AOE2KIT_FIXTURE_ROOT")
	if root == "" {
		t.Skip("AOE2KIT_FIXTURE_ROOT not set")
	}
	path := filepath.Join(root, "reference_aoe2_dump", "dat", "empires2_x2_p1.dat")
	compressed, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("fixture dat missing: %v", err)
	}
	payload, err := datfile.Inflate(compressed)
	if err != nil {
		t.Fatal(err)
	}
	before, err := datfile.Parse(payload)
	if err != nil {
		t.Fatal(err)
	}
	base := int32(201)
	outline := int32(202)
	stats := int32(208)
	patched, report, err := PatchRecipe(compressed, Recipe{
		CreatePlayerColour: &PlayerColourCreateRecipe{
			From:                1,
			Base:                &base,
			UnitOutlineColour:   &outline,
			StatisticsTextColor: &stats,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !report.Verified || !report.PayloadRoundTripOK || !report.ReadbackOK {
		t.Fatalf("player colour create report not verified: %+v", report)
	}
	if len(report.CreatedPlayerColours) != 1 || report.CreatedPlayerColours[0] != len(before.PlayerColours) {
		t.Fatalf("created player colours = %+v, before count %d", report.CreatedPlayerColours, len(before.PlayerColours))
	}
	if report.BeforePlayerColourCount != len(before.PlayerColours) || report.AfterPlayerColourCount != len(before.PlayerColours)+1 {
		t.Fatalf("player colour counts = before %d after %d, want %d/%d", report.BeforePlayerColourCount, report.AfterPlayerColourCount, len(before.PlayerColours), len(before.PlayerColours)+1)
	}
	afterPayload, err := datfile.Inflate(patched)
	if err != nil {
		t.Fatal(err)
	}
	after, err := datfile.Parse(afterPayload)
	if err != nil {
		t.Fatal(err)
	}
	created := after.PlayerColours[len(before.PlayerColours)]
	if created.ID != int32(len(before.PlayerColours)) ||
		created.Base != base ||
		created.UnitOutlineColour != outline ||
		created.StatisticsTextColor != stats {
		t.Fatalf("created player colour = %+v", created)
	}
}

func TestCodecRecipeDeletePlayerColourTail(t *testing.T) {
	root := os.Getenv("AOE2KIT_FIXTURE_ROOT")
	if root == "" {
		t.Skip("AOE2KIT_FIXTURE_ROOT not set")
	}
	path := filepath.Join(root, "reference_aoe2_dump", "dat", "empires2_x2_p1.dat")
	compressed, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("fixture dat missing: %v", err)
	}
	payload, err := datfile.Inflate(compressed)
	if err != nil {
		t.Fatal(err)
	}
	before, err := datfile.Parse(payload)
	if err != nil {
		t.Fatal(err)
	}
	base := int32(211)
	createdDat, createReport, err := PatchRecipe(compressed, Recipe{
		CreatePlayerColour: &PlayerColourCreateRecipe{
			From: 1,
			Base: &base,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(createReport.CreatedPlayerColours) != 1 {
		t.Fatalf("created player colours = %+v", createReport.CreatedPlayerColours)
	}
	createdID := createReport.CreatedPlayerColours[0]
	deletedDat, deleteReport, err := PatchRecipe(createdDat, Recipe{
		DeletePlayerColour: &createdID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !deleteReport.Verified || !deleteReport.PayloadRoundTripOK || !deleteReport.ReadbackOK {
		t.Fatalf("player colour delete report not verified: %+v", deleteReport)
	}
	if len(deleteReport.DeletedPlayerColours) != 1 {
		t.Fatalf("deleted player colours = %+v", deleteReport.DeletedPlayerColours)
	}
	deleted := deleteReport.DeletedPlayerColours[0]
	if deleted.ID != createdID || deleted.BeforeCount != len(before.PlayerColours)+1 || deleted.AfterCount != len(before.PlayerColours) || deleted.Strategy != "physical_tail_delete" {
		t.Fatalf("deleted player colour report = %+v", deleted)
	}
	afterPayload, err := datfile.Inflate(deletedDat)
	if err != nil {
		t.Fatal(err)
	}
	after, err := datfile.Parse(afterPayload)
	if err != nil {
		t.Fatal(err)
	}
	if len(after.PlayerColours) != len(before.PlayerColours) {
		t.Fatalf("player colour count after delete = %d, want %d", len(after.PlayerColours), len(before.PlayerColours))
	}
}

func TestCodecRecipePatchTerrainAndSound(t *testing.T) {
	root := os.Getenv("AOE2KIT_FIXTURE_ROOT")
	if root == "" {
		t.Skip("AOE2KIT_FIXTURE_ROOT not set")
	}
	path := filepath.Join(root, "reference_aoe2_dump", "dat", "empires2_x2_p1.dat")
	compressed, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("fixture dat missing: %v", err)
	}
	payload, err := datfile.Inflate(compressed)
	if err != nil {
		t.Fatal(err)
	}
	before, err := datfile.Parse(payload)
	if err != nil {
		t.Fatal(err)
	}
	soundID, itemID := firstSoundWithItem(t, before)
	terrainName := "AoE2Kit Terrain Test"
	terrainName2 := "AoE2Kit Terrain Test 2"
	overlay := "AoE2Kit Overlay"
	passability := float32(0.25)
	playDelay := int16(12)
	cacheTime := int32(34)
	totalProbability := int16(56)
	fileName := "aoe2kit_test_sound.wav"
	resourceID := int32(78)
	probability := int16(90)
	civ := int16(1)
	iconSet := int16(2)
	patched, report, err := PatchRecipe(compressed, Recipe{
		TerrainRestrictions: []TerrainRestrictionPatchRecipe{
			{ID: 0, Rows: []TerrainPassabilityPatch{{TerrainID: 0, Passability: passability}}},
		},
		Terrains: []TerrainPatchRecipe{
			{ID: 0, Name: &terrainName, Name2: &terrainName2, OverlayMaskName: &overlay},
		},
		Sounds: []SoundPatchRecipe{
			{
				ID:               soundID,
				PlayDelay:        &playDelay,
				CacheTime:        &cacheTime,
				TotalProbability: &totalProbability,
				Items: []SoundItemPatchRecipe{
					{
						Index:       itemID,
						FileName:    &fileName,
						ResourceID:  &resourceID,
						Probability: &probability,
						Civ:         &civ,
						IconSet:     &iconSet,
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !report.Verified || !report.PayloadRoundTripOK || !report.ReadbackOK {
		t.Fatalf("terrain/sound patch report not verified: %+v", report)
	}
	if len(report.PatchedTerrainRestrictions) != 1 || len(report.PatchedTerrains) != 1 || len(report.PatchedSounds) != 1 {
		t.Fatalf("terrain/sound patch report = %+v", report)
	}
	afterPayload, err := datfile.Inflate(patched)
	if err != nil {
		t.Fatal(err)
	}
	after, err := datfile.Parse(afterPayload)
	if err != nil {
		t.Fatal(err)
	}
	if after.TerrainRestrictions[0].TerrainRows[0].Passability != passability {
		t.Fatalf("passability readback got %g want %g", after.TerrainRestrictions[0].TerrainRows[0].Passability, passability)
	}
	terrain := after.Terrains[0]
	if terrain.Name.Value != terrainName || terrain.Name2.Value != terrainName2 || terrain.OverlayMaskName.Value != overlay {
		t.Fatalf("terrain readback = %+v", terrain)
	}
	sound := after.Sounds[soundID]
	item := sound.Items[itemID]
	if sound.PlayDelay != playDelay || sound.CacheTime != cacheTime || sound.TotalProbability != totalProbability {
		t.Fatalf("sound readback = %+v", sound)
	}
	if item.FileName.Value != fileName || item.ResourceID != resourceID || item.Probability != probability || item.Civ != civ || item.IconSet != iconSet {
		t.Fatalf("sound item readback = %+v", item)
	}
}

func TestCodecRecipeSemanticDeleteSound(t *testing.T) {
	root := os.Getenv("AOE2KIT_FIXTURE_ROOT")
	if root == "" {
		t.Skip("AOE2KIT_FIXTURE_ROOT not set")
	}
	path := filepath.Join(root, "reference_aoe2_dump", "dat", "empires2_x2_p1.dat")
	compressed, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("fixture dat missing: %v", err)
	}
	payload, err := datfile.Inflate(compressed)
	if err != nil {
		t.Fatal(err)
	}
	before, err := datfile.Parse(payload)
	if err != nil {
		t.Fatal(err)
	}
	soundID, _ := firstSoundWithItem(t, before)
	patched, report, err := PatchRecipe(compressed, Recipe{DeleteSounds: []int{soundID}})
	if err != nil {
		t.Fatal(err)
	}
	if !report.Verified || len(report.DeletedSounds) != 1 {
		t.Fatalf("delete sound report = %+v", report)
	}
	if report.DeletedSounds[0].Strategy != "semantic_mute" || report.DeletedSounds[0].AfterTotalProbability != 0 {
		t.Fatalf("deleted sound row = %+v", report.DeletedSounds[0])
	}
	afterPayload, err := datfile.Inflate(patched)
	if err != nil {
		t.Fatal(err)
	}
	after, err := datfile.Parse(afterPayload)
	if err != nil {
		t.Fatal(err)
	}
	sound := after.Sounds[soundID]
	if sound.TotalProbability != 0 {
		t.Fatalf("muted sound total_probability=%d, want 0", sound.TotalProbability)
	}
	for i, item := range sound.Items {
		if item.Probability != 0 {
			t.Fatalf("muted sound item[%d] probability=%d, want 0", i, item.Probability)
		}
	}
}

func TestCodecRecipeRemoveSoundItem(t *testing.T) {
	root := os.Getenv("AOE2KIT_FIXTURE_ROOT")
	if root == "" {
		t.Skip("AOE2KIT_FIXTURE_ROOT not set")
	}
	path := filepath.Join(root, "reference_aoe2_dump", "dat", "empires2_x2_p1.dat")
	compressed, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("fixture dat missing: %v", err)
	}
	payload, err := datfile.Inflate(compressed)
	if err != nil {
		t.Fatal(err)
	}
	before, err := datfile.Parse(payload)
	if err != nil {
		t.Fatal(err)
	}
	soundID, itemID := firstSoundWithItem(t, before)
	beforeCount := len(before.Sounds[soundID].Items)
	patched, report, err := PatchRecipe(compressed, Recipe{
		Sounds: []SoundPatchRecipe{{ID: soundID, RemoveItems: []int{itemID}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !report.Verified || !report.PayloadRoundTripOK || !report.ReadbackOK || len(report.PatchedSounds) != 1 {
		t.Fatalf("remove sound item report = %+v", report)
	}
	afterPayload, err := datfile.Inflate(patched)
	if err != nil {
		t.Fatal(err)
	}
	after, err := datfile.Parse(afterPayload)
	if err != nil {
		t.Fatal(err)
	}
	afterItems := after.Sounds[soundID].Items
	if len(afterItems) != beforeCount-1 {
		t.Fatalf("sound item count=%d, want %d", len(afterItems), beforeCount-1)
	}
	for i, item := range afterItems {
		if item.Index != i {
			t.Fatalf("sound item[%d] index=%d, want %d", i, item.Index, i)
		}
	}
}

func TestDeletePlanUnitChildRowsProducePatchRecipes(t *testing.T) {
	root := os.Getenv("AOE2KIT_FIXTURE_ROOT")
	if root == "" {
		t.Skip("AOE2KIT_FIXTURE_ROOT not set")
	}
	path := filepath.Join(root, "reference_aoe2_dump", "dat", "empires2_x2_p1.dat")
	compressed, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("fixture dat missing: %v", err)
	}
	payload, err := datfile.Inflate(compressed)
	if err != nil {
		t.Fatal(err)
	}
	idx, err := datfile.Parse(payload)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		section string
		count   func(datfile.UnitSummary) int
	}{
		{section: "unit_attack", count: func(unit datfile.UnitSummary) int {
			if unit.Type50 == nil {
				return 0
			}
			return len(unit.Type50.Attacks)
		}},
		{section: "unit_armour", count: func(unit datfile.UnitSummary) int {
			if unit.Type50 == nil {
				return 0
			}
			return len(unit.Type50.Armours)
		}},
		{section: "unit_train_location", count: func(unit datfile.UnitSummary) int {
			if unit.Creatable == nil {
				return 0
			}
			return len(unit.Creatable.TrainLocations)
		}},
		{section: "unit_task", count: func(unit datfile.UnitSummary) int {
			if unit.Action == nil {
				return 0
			}
			return len(unit.Action.Tasks)
		}},
		{section: "unit_drop_site", count: func(unit datfile.UnitSummary) int {
			if unit.Action == nil {
				return 0
			}
			return len(unit.Action.DropSites)
		}},
		{section: "unit_damage_graphic", count: func(unit datfile.UnitSummary) int {
			return len(unit.DamageGraphics)
		}},
	} {
		civID, unitID, rowIndex := firstUnitWithDeletableRows(t, idx, tc.count)
		before := idx.Civs[civID].Units[unitID]
		beforeCount := tc.count(before)
		plan := DeletePlan(idx, DeletePlanRequest{Section: tc.section, CivID: &civID, UnitID: &unitID, RowIndex: &rowIndex})
		if !plan.Supported || !plan.Mutates || plan.Strategy != "physical_unit_child_row_delete" || !bytes.Contains(plan.RecipeHint, []byte("units")) {
			t.Fatalf("%s plan = %+v", tc.section, plan)
		}
		var recipe Recipe
		if err := json.Unmarshal(plan.RecipeHint, &recipe); err != nil {
			t.Fatalf("%s recipe hint: %v", tc.section, err)
		}
		patched, report, err := PatchRecipe(compressed, recipe)
		if err != nil {
			t.Fatalf("%s PatchRecipe: %v", tc.section, err)
		}
		if !report.Verified || !report.PayloadRoundTripOK || !report.ReadbackOK {
			t.Fatalf("%s report = %+v", tc.section, report)
		}
		afterPayload, err := datfile.Inflate(patched)
		if err != nil {
			t.Fatalf("%s inflate patched: %v", tc.section, err)
		}
		after, err := datfile.Parse(afterPayload)
		if err != nil {
			t.Fatalf("%s parse patched: %v", tc.section, err)
		}
		got := tc.count(after.Civs[civID].Units[unitID])
		if got != beforeCount-1 {
			t.Fatalf("%s row count=%d, want %d", tc.section, got, beforeCount-1)
		}
	}
}

func TestDeletePlanUnitHeaderTaskProducesPatchRecipe(t *testing.T) {
	root := os.Getenv("AOE2KIT_FIXTURE_ROOT")
	if root == "" {
		t.Skip("AOE2KIT_FIXTURE_ROOT not set")
	}
	path := filepath.Join(root, "reference_aoe2_dump", "dat", "empires2_x2_p1.dat")
	compressed, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("fixture dat missing: %v", err)
	}
	payload, err := datfile.Inflate(compressed)
	if err != nil {
		t.Fatal(err)
	}
	idx, err := datfile.Parse(payload)
	if err != nil {
		t.Fatal(err)
	}
	headerID, rowIndex := firstUnitHeaderWithTasks(t, idx)
	beforeCount := len(idx.UnitHeaders[headerID].Tasks)
	plan := DeletePlan(idx, DeletePlanRequest{Section: "unit_header_task", UnitHeaderID: &headerID, RowIndex: &rowIndex})
	if !plan.Supported || !plan.Mutates || plan.Strategy != "physical_unit_header_task_row_delete" || !bytes.Contains(plan.RecipeHint, []byte("unit_headers")) {
		t.Fatalf("unit-header-task plan = %+v", plan)
	}
	var recipe Recipe
	if err := json.Unmarshal(plan.RecipeHint, &recipe); err != nil {
		t.Fatalf("unit-header-task recipe hint: %v", err)
	}
	patched, report, err := PatchRecipe(compressed, recipe)
	if err != nil {
		t.Fatalf("unit-header-task PatchRecipe: %v", err)
	}
	if !report.Verified || !report.PayloadRoundTripOK || !report.ReadbackOK {
		t.Fatalf("unit-header-task report = %+v", report)
	}
	afterPayload, err := datfile.Inflate(patched)
	if err != nil {
		t.Fatal(err)
	}
	after, err := datfile.Parse(afterPayload)
	if err != nil {
		t.Fatal(err)
	}
	got := len(after.UnitHeaders[headerID].Tasks)
	if got != beforeCount-1 {
		t.Fatalf("unit-header-task row count=%d, want %d", got, beforeCount-1)
	}
}

func TestDeletePlanGraphicChildRowsProducePatchRecipes(t *testing.T) {
	root := os.Getenv("AOE2KIT_FIXTURE_ROOT")
	if root == "" {
		t.Skip("AOE2KIT_FIXTURE_ROOT not set")
	}
	path := filepath.Join(root, "reference_aoe2_dump", "dat", "empires2_x2_p1.dat")
	compressed, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("fixture dat missing: %v", err)
	}
	payload, err := datfile.Inflate(compressed)
	if err != nil {
		t.Fatal(err)
	}
	idx, err := datfile.Parse(payload)
	if err != nil {
		t.Fatal(err)
	}
	graphicID, rowIndex := firstGraphicWithDeltas(t, idx)
	beforeDeltaCount := len(idx.Graphics[graphicID].Deltas)
	plan := DeletePlan(idx, DeletePlanRequest{Section: "graphic_delta", GraphicID: &graphicID, RowIndex: &rowIndex})
	if !plan.Supported || !plan.Mutates || plan.Strategy != "physical_graphic_child_row_delete" || !bytes.Contains(plan.RecipeHint, []byte("graphics")) {
		t.Fatalf("graphic-delta plan = %+v", plan)
	}
	var recipe Recipe
	if err := json.Unmarshal(plan.RecipeHint, &recipe); err != nil {
		t.Fatalf("graphic-delta recipe hint: %v", err)
	}
	patched, report, err := PatchRecipe(compressed, recipe)
	if err != nil {
		t.Fatalf("graphic-delta PatchRecipe: %v", err)
	}
	if !report.Verified || !report.PayloadRoundTripOK || !report.ReadbackOK || len(report.GraphicReports) != 1 {
		t.Fatalf("graphic-delta report = %+v", report)
	}
	afterPayload, err := datfile.Inflate(patched)
	if err != nil {
		t.Fatal(err)
	}
	after, err := datfile.Parse(afterPayload)
	if err != nil {
		t.Fatal(err)
	}
	if got := len(after.Graphics[graphicID].Deltas); got != beforeDeltaCount-1 {
		t.Fatalf("graphic-delta count=%d, want %d", got, beforeDeltaCount-1)
	}

	angleGraphicID, angleRowIndex, canDelete := firstGraphicWithAngleSounds(t, idx)
	if !canDelete {
		t.Skip("fixture has no graphic angle-sound rows")
	}
	beforeAngleCount := len(idx.Graphics[angleGraphicID].AngleSounds)
	anglePlan := DeletePlan(idx, DeletePlanRequest{Section: "graphic_angle_sound", GraphicID: &angleGraphicID, RowIndex: &angleRowIndex})
	if beforeAngleCount == 1 {
		if !anglePlan.Supported || !anglePlan.Mutates || !bytes.Contains(anglePlan.RecipeHint, []byte("set_angle_sounds")) {
			t.Fatalf("graphic-angle-sound single-row plan = %+v", anglePlan)
		}
		var angleRecipe Recipe
		if err := json.Unmarshal(anglePlan.RecipeHint, &angleRecipe); err != nil {
			t.Fatalf("graphic-angle-sound recipe hint: %v", err)
		}
		anglePatched, angleReport, err := PatchRecipe(compressed, angleRecipe)
		if err != nil {
			t.Fatalf("graphic-angle-sound PatchRecipe: %v", err)
		}
		if !angleReport.Verified || !angleReport.PayloadRoundTripOK || !angleReport.ReadbackOK || len(angleReport.GraphicReports) != 1 {
			t.Fatalf("graphic-angle-sound report = %+v", angleReport)
		}
		anglePayload, err := datfile.Inflate(anglePatched)
		if err != nil {
			t.Fatal(err)
		}
		angleAfter, err := datfile.Parse(anglePayload)
		if err != nil {
			t.Fatal(err)
		}
		if got := len(angleAfter.Graphics[angleGraphicID].AngleSounds); got != 0 {
			t.Fatalf("graphic-angle-sound count=%d, want 0", got)
		}
	} else if anglePlan.Supported || anglePlan.Reason == "" {
		t.Fatalf("graphic-angle-sound partial delete should be guarded: %+v", anglePlan)
	}
}

func TestCodecRecipeCreateSound(t *testing.T) {
	root := os.Getenv("AOE2KIT_FIXTURE_ROOT")
	if root == "" {
		t.Skip("AOE2KIT_FIXTURE_ROOT not set")
	}
	path := filepath.Join(root, "reference_aoe2_dump", "dat", "empires2_x2_p1.dat")
	compressed, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("fixture dat missing: %v", err)
	}
	payload, err := datfile.Inflate(compressed)
	if err != nil {
		t.Fatal(err)
	}
	before, err := datfile.Parse(payload)
	if err != nil {
		t.Fatal(err)
	}
	fromSoundID, _ := firstSoundWithItem(t, before)
	playDelay := int16(17)
	cacheTime := int32(29)
	totalProbability := int16(100)
	item := SoundItemRecipe{
		FileName:    "aoe2kit_created_sound.wav",
		ResourceID:  424242,
		Probability: 100,
		Civ:         -1,
		IconSet:     0,
	}
	patched, report, err := PatchRecipe(compressed, Recipe{
		CreateSound: &SoundCreateRecipe{
			From:             fromSoundID,
			PlayDelay:        &playDelay,
			CacheTime:        &cacheTime,
			TotalProbability: &totalProbability,
			Items:            &[]SoundItemRecipe{item},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !report.Verified || !report.PayloadRoundTripOK || !report.ReadbackOK {
		t.Fatalf("create sound report not verified: %+v", report)
	}
	if report.BeforeSoundCount != len(before.Sounds) || report.AfterSoundCount != len(before.Sounds)+1 || len(report.CreatedSounds) != 1 {
		t.Fatalf("create sound counts/report = %+v", report)
	}
	newSoundID := report.CreatedSounds[0]
	if newSoundID != len(before.Sounds) {
		t.Fatalf("created sound id=%d, want %d", newSoundID, len(before.Sounds))
	}
	afterPayload, err := datfile.Inflate(patched)
	if err != nil {
		t.Fatal(err)
	}
	after, err := datfile.Parse(afterPayload)
	if err != nil {
		t.Fatal(err)
	}
	if len(after.Sounds) != len(before.Sounds)+1 {
		t.Fatalf("sound count after create=%d, want %d", len(after.Sounds), len(before.Sounds)+1)
	}
	sound := after.Sounds[newSoundID]
	if sound.Index != newSoundID || sound.ID != int16(newSoundID) || sound.PlayDelay != playDelay || sound.CacheTime != cacheTime || sound.TotalProbability != totalProbability {
		t.Fatalf("created sound = %+v", sound)
	}
	if len(sound.Items) != 1 {
		t.Fatalf("created sound items=%d, want 1", len(sound.Items))
	}
	gotItem := sound.Items[0]
	if gotItem.FileName.Value != item.FileName || gotItem.ResourceID != item.ResourceID || gotItem.Probability != item.Probability || gotItem.Civ != item.Civ || gotItem.IconSet != item.IconSet {
		t.Fatalf("created sound item = %+v", gotItem)
	}
	if after.Sounds[fromSoundID].ID != before.Sounds[fromSoundID].ID || len(after.Sounds[fromSoundID].Items) != len(before.Sounds[fromSoundID].Items) {
		t.Fatalf("template neighbor changed: before=%+v after=%+v", before.Sounds[fromSoundID], after.Sounds[fromSoundID])
	}
}

func TestCodecRecipeSemanticDeleteUnit(t *testing.T) {
	root := os.Getenv("AOE2KIT_FIXTURE_ROOT")
	if root == "" {
		t.Skip("AOE2KIT_FIXTURE_ROOT not set")
	}
	path := filepath.Join(root, "reference_aoe2_dump", "dat", "empires2_x2_p1.dat")
	compressed, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("fixture dat missing: %v", err)
	}
	payload, err := datfile.Inflate(compressed)
	if err != nil {
		t.Fatal(err)
	}
	before, err := datfile.Parse(payload)
	if err != nil {
		t.Fatal(err)
	}
	civID, unitID := firstEnabledUnit(t, before)
	patched, report, err := PatchRecipe(compressed, Recipe{
		DeleteUnits: []UnitDeleteRecipe{
			{CivID: &civID, UnitID: unitID},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !report.Verified || len(report.DeletedUnits) != 1 {
		t.Fatalf("delete unit report = %+v", report)
	}
	deleted := report.DeletedUnits[0]
	if deleted.CivID != civID || deleted.UnitID != unitID || deleted.Strategy != "semantic_disable" || deleted.AfterEnabled != 0 {
		t.Fatalf("deleted unit report row = %+v", deleted)
	}
	afterPayload, err := datfile.Inflate(patched)
	if err != nil {
		t.Fatal(err)
	}
	after, err := datfile.Parse(afterPayload)
	if err != nil {
		t.Fatal(err)
	}
	unit := after.Civs[civID].Units[unitID]
	if unit.Enabled != 0 {
		t.Fatalf("deleted unit enabled=%d, want 0", unit.Enabled)
	}
	if len(after.Civs) != len(before.Civs) || len(after.Civs[civID].Units) != len(before.Civs[civID].Units) {
		t.Fatalf("semantic delete changed counts: civs %d/%d units %d/%d", len(after.Civs), len(before.Civs), len(after.Civs[civID].Units), len(before.Civs[civID].Units))
	}
}

func TestCodecRecipeSemanticDeleteCreatedUnitWithoutHeader(t *testing.T) {
	root := os.Getenv("AOE2KIT_FIXTURE_ROOT")
	if root == "" {
		t.Skip("AOE2KIT_FIXTURE_ROOT not set")
	}
	path := filepath.Join(root, "reference_aoe2_dump", "dat", "empires2_x2_p1.dat")
	compressed, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("fixture dat missing: %v", err)
	}
	enabled := uint8(1)
	created, createReport, err := PatchRecipe(compressed, Recipe{
		CreateUnit: &datfile.UnitCreateRecipe{
			FromCivID:  0,
			FromUnitID: 83,
			CivIDs:     []int{0},
			Enabled:    &enabled,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(createReport.CreatedUnits) != 1 {
		t.Fatalf("created units = %+v", createReport.CreatedUnits)
	}
	unitID := createReport.CreatedUnits[0].NewUnitID
	patched, report, err := PatchRecipe(created, Recipe{
		DisconnectUnits: []int{unitID},
		DeleteUnits:     []UnitDeleteRecipe{{UnitID: unitID, CivID: intPtr(0)}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !report.Verified || len(report.DeletedUnits) != 1 || report.DeletedUnits[0].AfterEnabled != 0 {
		t.Fatalf("delete created unit report = %+v", report)
	}
	payload, err := datfile.Inflate(patched)
	if err != nil {
		t.Fatal(err)
	}
	after, err := datfile.Parse(payload)
	if err != nil {
		t.Fatal(err)
	}
	if unitID >= len(after.Civs[0].Units) || after.Civs[0].Units[unitID].Enabled != 0 {
		t.Fatalf("created unit %d was not semantically disabled", unitID)
	}
}

func TestUnitAvailabilityRecipe(t *testing.T) {
	root := os.Getenv("AOE2KIT_FIXTURE_ROOT")
	if root == "" {
		t.Skip("AOE2KIT_FIXTURE_ROOT not set")
	}
	path := filepath.Join(root, "reference_aoe2_dump", "dat", "empires2_x2_p1.dat")
	compressed, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("fixture dat missing: %v", err)
	}
	payload, err := datfile.Inflate(compressed)
	if err != nil {
		t.Fatal(err)
	}
	before, err := datfile.Parse(payload)
	if err != nil {
		t.Fatal(err)
	}
	civID, unitID := firstEnabledUnit(t, before)
	patched, report, err := PatchRecipe(compressed, Recipe{
		UnitAvailability: []UnitAvailabilityRecipe{
			{CivIDs: []int{civID}, UnitID: unitID, Enabled: false},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !report.Verified || len(report.UnitAvailability) != 1 {
		t.Fatalf("unit availability report = %+v", report)
	}
	row := report.UnitAvailability[0]
	if row.CivID != civID || row.UnitID != unitID || !row.Present || row.BeforeEnabled == 0 || row.AfterEnabled != 0 {
		t.Fatalf("unit availability row = %+v", row)
	}
	afterPayload, err := datfile.Inflate(patched)
	if err != nil {
		t.Fatal(err)
	}
	after, err := datfile.Parse(afterPayload)
	if err != nil {
		t.Fatal(err)
	}
	unit := after.Civs[civID].Units[unitID]
	if !unit.Present || unit.Enabled != 0 {
		t.Fatalf("availability patch present=%t enabled=%d, want present enabled=0", unit.Present, unit.Enabled)
	}
	if len(after.Civs) != len(before.Civs) || len(after.Civs[civID].Units) != len(before.Civs[civID].Units) {
		t.Fatalf("availability patch changed counts: civs %d/%d units %d/%d", len(after.Civs), len(before.Civs), len(after.Civs[civID].Units), len(before.Civs[civID].Units))
	}
	cardReport, err := UnitAvailability(after, UnitAvailabilityRequest{CivID: &civID, UnitID: unitID})
	if err != nil {
		t.Fatal(err)
	}
	if len(cardReport.Cards) != 1 || cardReport.Cards[0].Status != "disabled_unit_record" || len(cardReport.Cards[0].RecipeHint) == 0 {
		t.Fatalf("disabled availability card missing recipe hint: %+v", cardReport)
	}
}

func TestUnitAvailabilityCards(t *testing.T) {
	root := os.Getenv("AOE2KIT_FIXTURE_ROOT")
	if root == "" {
		t.Skip("AOE2KIT_FIXTURE_ROOT not set")
	}
	path := filepath.Join(root, "reference_aoe2_dump", "dat", "empires2_x2_p1.dat")
	idx, err := datfile.Open(path)
	if err != nil {
		t.Skipf("fixture dat missing: %v", err)
	}
	civID, unitID := firstEnabledUnit(t, idx)
	report, err := UnitAvailability(idx, UnitAvailabilityRequest{CivID: &civID, UnitID: unitID})
	if err != nil {
		t.Fatal(err)
	}
	if !report.Verification.StructureVerified || report.Summary.CivCount != 1 || len(report.Cards) != 1 {
		t.Fatalf("availability report = %+v", report)
	}
	card := report.Cards[0]
	if card.CivID != civID || card.UnitID != unitID || !card.Present || !card.Enabled || card.Status == "" {
		t.Fatalf("availability card = %+v", card)
	}
	if report.Summary.Present != 1 || report.Summary.Enabled != 1 {
		t.Fatalf("availability summary = %+v", report.Summary)
	}
}

func TestDeletePlanUnitAndUnsupportedGraphic(t *testing.T) {
	root := os.Getenv("AOE2KIT_FIXTURE_ROOT")
	if root == "" {
		t.Skip("AOE2KIT_FIXTURE_ROOT not set")
	}
	path := filepath.Join(root, "reference_aoe2_dump", "dat", "empires2_x2_p1.dat")
	idx, err := datfile.Open(path)
	if err != nil {
		t.Skipf("fixture dat missing: %v", err)
	}
	civID, unitID := 1, 74
	unitPlan := DeletePlan(idx, DeletePlanRequest{Section: "unit", ID: unitID, CivID: &civID})
	if !unitPlan.Supported || !unitPlan.Mutates || unitPlan.Strategy != "semantic_disable" || len(unitPlan.RecipeHint) == 0 || !bytes.Contains(unitPlan.RecipeHint, []byte("disconnect_units")) {
		t.Fatalf("unit delete plan = %+v", unitPlan)
	}
	if len(unitPlan.References) == 0 {
		t.Fatalf("unit delete plan did not report DAT references: %+v", unitPlan)
	}
	if anyUnitRewriteSupported(unitPlan.References) && (len(unitPlan.CleanupRecipe) == 0 || !bytes.Contains(unitPlan.CleanupRecipe, []byte("disconnect_units")) || !strings.Contains(unitPlan.CleanupCommand, "kit dat disconnect <in.dat> <out.dat> unit")) {
		t.Fatalf("unit delete plan missing optional cleanup metadata: %+v", unitPlan)
	}
	dropSiteUnitID := firstDropSiteReferencedUnit(t, idx)
	dropSitePlan := DeletePlan(idx, DeletePlanRequest{Section: "unit", ID: dropSiteUnitID, CivID: &civID})
	if !referencesFieldContaining(dropSitePlan.References, ".drop_site[") {
		t.Fatalf("drop-site references missing for unit %d: %+v", dropSiteUnitID, dropSitePlan.References)
	}
	if len(dropSitePlan.CleanupRecipe) == 0 || !bytes.Contains(dropSitePlan.CleanupRecipe, []byte("disconnect_units")) {
		t.Fatalf("drop-site unit delete plan missing optional cleanup metadata: %+v", dropSitePlan)
	}
	graphicPlan := DeletePlan(idx, DeletePlanRequest{Section: "graphic", ID: 339})
	if graphicPlan.Supported || graphicPlan.Strategy != "unsupported_referenced_stable_slot" || graphicPlan.Reason == "" || len(graphicPlan.References) == 0 {
		t.Fatalf("graphic delete plan = %+v", graphicPlan)
	}
	referencedCivID := firstReferencedCiv(t, idx)
	civPlan := DeletePlan(idx, DeletePlanRequest{Section: "civ", ID: referencedCivID})
	if civPlan.Supported || civPlan.Strategy != "unsupported" || civPlan.Reason == "" || len(civPlan.References) == 0 {
		t.Fatalf("civ delete plan = %+v", civPlan)
	}
	terrainID := firstReferencedTerrain(t, idx)
	terrainPlan := DeletePlan(idx, DeletePlanRequest{Section: "terrain", ID: terrainID})
	if terrainPlan.Supported || terrainPlan.Strategy != "unsupported" || terrainPlan.Reason == "" || len(terrainPlan.References) == 0 {
		t.Fatalf("terrain delete plan = %+v", terrainPlan)
	}
	playerColourID := firstReferencedPlayerColour(t, idx)
	playerColourPlan := DeletePlan(idx, DeletePlanRequest{Section: "player_colour", ID: playerColourID})
	if playerColourPlan.Supported || !strings.HasPrefix(playerColourPlan.Strategy, "unsupported") || playerColourPlan.Reason == "" || len(playerColourPlan.References) == 0 {
		t.Fatalf("player colour delete plan = %+v", playerColourPlan)
	}
	compressed, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("fixture dat missing: %v", err)
	}
	createdDat, createReport, err := PatchRecipe(compressed, Recipe{
		CreatePlayerColour: &PlayerColourCreateRecipe{From: 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	createdPayload, err := datfile.Inflate(createdDat)
	if err != nil {
		t.Fatal(err)
	}
	createdIdx, err := datfile.Parse(createdPayload)
	if err != nil {
		t.Fatal(err)
	}
	createdColourID := createReport.CreatedPlayerColours[0]
	tailColourPlan := DeletePlan(createdIdx, DeletePlanRequest{Section: "player_colour", ID: createdColourID})
	if !tailColourPlan.Supported || !tailColourPlan.Mutates || tailColourPlan.Strategy != "physical_tail_delete" || !bytes.Contains(tailColourPlan.RecipeHint, []byte("delete_player_colours")) {
		t.Fatalf("tail player colour delete plan = %+v", tailColourPlan)
	}
	createdGraphicDat, graphicCreateReport, err := PatchRecipe(compressed, Recipe{
		CreateGraphic: &datfile.GraphicCreateRecipe{From: 1711},
	})
	if err != nil {
		t.Fatal(err)
	}
	createdGraphicPayload, err := datfile.Inflate(createdGraphicDat)
	if err != nil {
		t.Fatal(err)
	}
	createdGraphicIdx, err := datfile.Parse(createdGraphicPayload)
	if err != nil {
		t.Fatal(err)
	}
	createdGraphicID := graphicCreateReport.CreatedGraphics[0].NewGraphicID
	tailGraphicPlan := DeletePlan(createdGraphicIdx, DeletePlanRequest{Section: "graphic", ID: createdGraphicID})
	if !tailGraphicPlan.Supported || !tailGraphicPlan.Mutates || tailGraphicPlan.Strategy != "stable_id_absent_slot" || !bytes.Contains(tailGraphicPlan.RecipeHint, []byte("delete_graphics")) {
		t.Fatalf("tail graphic delete plan = %+v", tailGraphicPlan)
	}
	var graphicDeleteRecipe Recipe
	if err := json.Unmarshal(tailGraphicPlan.RecipeHint, &graphicDeleteRecipe); err != nil {
		t.Fatal(err)
	}
	deletedGraphicDat, graphicDeleteReport, err := PatchRecipe(createdGraphicDat, graphicDeleteRecipe)
	if err != nil {
		t.Fatal(err)
	}
	if len(graphicDeleteReport.DeletedGraphics) != 1 || !graphicDeleteReport.Verified || !graphicDeleteReport.ReadbackOK {
		t.Fatalf("graphic delete report = %+v", graphicDeleteReport)
	}
	deletedGraphicPayload, err := datfile.Inflate(deletedGraphicDat)
	if err != nil {
		t.Fatal(err)
	}
	deletedGraphicIdx, err := datfile.Parse(deletedGraphicPayload)
	if err != nil {
		t.Fatal(err)
	}
	if deletedGraphicIdx.GraphicsSize != createdGraphicIdx.GraphicsSize {
		t.Fatalf("graphics size changed: before=%d after=%d", createdGraphicIdx.GraphicsSize, deletedGraphicIdx.GraphicsSize)
	}
	if _, ok := deletedGraphicIdx.Graphic(createdGraphicID); ok {
		t.Fatalf("deleted graphic %d still present", createdGraphicID)
	}
	buildingConnectionPlan := DeletePlan(idx, DeletePlanRequest{Section: "tech_tree_building_connection", ID: 1})
	if !buildingConnectionPlan.Supported || !buildingConnectionPlan.Mutates || buildingConnectionPlan.Strategy != "physical_connection_row_delete" || !bytes.Contains(buildingConnectionPlan.RecipeHint, []byte("delete_building_connections")) {
		t.Fatalf("building connection delete plan = %+v", buildingConnectionPlan)
	}
	unitConnectionPlan := DeletePlan(idx, DeletePlanRequest{Section: "unit_connection", ID: 1})
	if !unitConnectionPlan.Supported || !unitConnectionPlan.Mutates || unitConnectionPlan.Strategy != "physical_connection_row_delete" || !bytes.Contains(unitConnectionPlan.RecipeHint, []byte("delete_unit_connections")) {
		t.Fatalf("unit connection delete plan = %+v", unitConnectionPlan)
	}
	researchConnectionPlan := DeletePlan(idx, DeletePlanRequest{Section: "research_connection", ID: 1})
	if !researchConnectionPlan.Supported || !researchConnectionPlan.Mutates || researchConnectionPlan.Strategy != "physical_connection_row_delete" || !bytes.Contains(researchConnectionPlan.RecipeHint, []byte("delete_research_connections")) {
		t.Fatalf("research connection delete plan = %+v", researchConnectionPlan)
	}
	outOfRangeConnectionPlan := DeletePlan(idx, DeletePlanRequest{Section: "building_connection", ID: len(idx.TechTree.BuildingConnections)})
	if outOfRangeConnectionPlan.Supported || outOfRangeConnectionPlan.Strategy != "physical_connection_row_delete" || outOfRangeConnectionPlan.Reason == "" {
		t.Fatalf("out-of-range building connection plan = %+v", outOfRangeConnectionPlan)
	}
}

func TestDeletePlanEffectAndTechRules(t *testing.T) {
	root := os.Getenv("AOE2KIT_FIXTURE_ROOT")
	if root == "" {
		t.Skip("AOE2KIT_FIXTURE_ROOT not set")
	}
	path := filepath.Join(root, "reference_aoe2_dump", "dat", "empires2_x2_p1.dat")
	idx, err := datfile.Open(path)
	if err != nil {
		t.Skipf("fixture dat missing: %v", err)
	}
	referencedEffect := firstReferencedEffect(t, idx)
	effectPlan := DeletePlan(idx, DeletePlanRequest{Section: "effect", ID: referencedEffect})
	if !effectPlan.Supported || !effectPlan.Mutates || effectPlan.Strategy != "semantic_clear_commands" || len(effectPlan.References) == 0 || !bytes.Contains(effectPlan.RecipeHint, []byte("disable_effects")) {
		t.Fatalf("referenced effect delete plan = %+v", effectPlan)
	}
	effectCommandPlan := DeletePlan(idx, DeletePlanRequest{Section: "effect_command", EffectID: &referencedEffect, CommandIndex: intPtr(0)})
	if !effectCommandPlan.Supported || !effectCommandPlan.Mutates || effectCommandPlan.Strategy != "physical_effect_command_row_delete" || !bytes.Contains(effectCommandPlan.RecipeHint, []byte("remove_commands")) {
		t.Fatalf("effect-command delete plan = %+v", effectCommandPlan)
	}
	badCommandIndex := len(idx.Effects[referencedEffect].Commands)
	badEffectCommandPlan := DeletePlan(idx, DeletePlanRequest{Section: "effect_command", EffectID: &referencedEffect, CommandIndex: &badCommandIndex})
	if badEffectCommandPlan.Supported || badEffectCommandPlan.Reason == "" {
		t.Fatalf("bad effect-command delete plan = %+v", badEffectCommandPlan)
	}
	referencedTech := firstReferencedTech(t, idx)
	nonTailTechPlan := DeletePlan(idx, DeletePlanRequest{Section: "tech", ID: referencedTech})
	if len(idx.Techs) > 1 && (nonTailTechPlan.Supported || len(nonTailTechPlan.References) == 0 || nonTailTechPlan.Reason == "") {
		t.Fatalf("non-tail tech delete plan = %+v", nonTailTechPlan)
	}
	effectCommandTech := firstTechReferencedByEffectCommand(t, idx)
	effectCommandTechPlan := DeletePlan(idx, DeletePlanRequest{Section: "tech", ID: effectCommandTech})
	if !referencesWithConfidence(effectCommandTechPlan.References, "typed_effect_command_reference") {
		t.Fatalf("tech %d delete plan missing typed effect-command refs: %+v", effectCommandTech, effectCommandTechPlan.References)
	}
	if effectCommandTechPlan.Supported || len(effectCommandTechPlan.RecipeHint) == 0 || !bytes.Contains(effectCommandTechPlan.RecipeHint, []byte("disconnect_techs")) {
		t.Fatalf("tech %d delete plan missing staged disconnect_techs hint: %+v", effectCommandTech, effectCommandTechPlan)
	}
	if len(effectCommandTechPlan.CleanupRecipe) == 0 || !bytes.Contains(effectCommandTechPlan.CleanupRecipe, []byte("disconnect_techs")) || !strings.Contains(effectCommandTechPlan.CleanupCommand, "kit dat disconnect <in.dat> <out.dat> tech") {
		t.Fatalf("tech %d delete plan missing explicit cleanup metadata: %+v", effectCommandTech, effectCommandTechPlan)
	}
	if effectCommandTechPlan.TechMigration == nil || effectCommandTechPlan.TechMigration.CheckedTechIDs == 0 || effectCommandTechPlan.TechMigration.CoveredReferences == 0 {
		t.Fatalf("tech %d delete plan missing migration advisory: %+v", effectCommandTech, effectCommandTechPlan.TechMigration)
	}
	filePlan, err := DeletePlanFile(path, DeletePlanRequest{Section: "tech", ID: effectCommandTech})
	if err != nil {
		t.Fatalf("DeletePlanFile: %v", err)
	}
	if filePlan.Path != path || !strings.Contains(filePlan.CleanupCommand, path) {
		t.Fatalf("DeletePlanFile cleanup command=%q path=%q, want hydrated input path %q", filePlan.CleanupCommand, filePlan.Path, path)
	}
	refsReport := References(idx, DeletePlanRequest{Section: "tech", ID: effectCommandTech})
	if refsReport.Summary.Total == 0 || refsReport.Summary.RewriteSupported == 0 || refsReport.Summary.PossibleOperand == 0 {
		t.Fatalf("tech %d refs report summary = %+v", effectCommandTech, refsReport.Summary)
	}
	filteredRefs := FilterReferenceReport(refsReport, ReferenceFilters{Class: "rewrite_supported", SourceSection: "effect", Limit: 1})
	if filteredRefs.Summary.Total < 1 || filteredRefs.Summary.Returned != 1 || filteredRefs.Summary.RewriteSupported < 1 || len(filteredRefs.References) != 1 {
		t.Fatalf("filtered refs summary=%+v refs=%+v", filteredRefs.Summary, filteredRefs.References)
	}
	if filteredRefs.Filters.Class != "rewrite_supported" || filteredRefs.Filters.SourceSection != "effect" || filteredRefs.Filters.Limit != 1 {
		t.Fatalf("filtered refs did not echo filters: %+v", filteredRefs.Filters)
	}
	tailTechPlan := DeletePlan(idx, DeletePlanRequest{Section: "tech", ID: len(idx.Techs) - 1})
	if !tailTechPlan.Supported || tailTechPlan.Strategy != "physical_tail_delete" || len(tailTechPlan.RecipeHint) == 0 {
		t.Fatalf("tail tech delete plan = %+v", tailTechPlan)
	}
	soundID, _ := firstSoundWithItem(t, idx)
	soundPlan := DeletePlan(idx, DeletePlanRequest{Section: "sound", ID: soundID})
	if !soundPlan.Supported || soundPlan.Strategy != "semantic_mute" || len(soundPlan.RecipeHint) == 0 {
		t.Fatalf("sound delete plan = %+v", soundPlan)
	}
	itemIndex := 0
	soundItemPlan := DeletePlan(idx, DeletePlanRequest{Section: "sound_item", SoundID: &soundID, ItemIndex: &itemIndex})
	if !soundItemPlan.Supported || !soundItemPlan.Mutates || soundItemPlan.Strategy != "physical_sound_item_row_delete" || !bytes.Contains(soundItemPlan.RecipeHint, []byte("remove_items")) {
		t.Fatalf("sound-item delete plan = %+v", soundItemPlan)
	}
	badSoundItemIndex := len(idx.Sounds[soundID].Items)
	badSoundItemPlan := DeletePlan(idx, DeletePlanRequest{Section: "sound_item", SoundID: &soundID, ItemIndex: &badSoundItemIndex})
	if badSoundItemPlan.Supported || badSoundItemPlan.Reason == "" {
		t.Fatalf("bad sound-item delete plan = %+v", badSoundItemPlan)
	}
}

func TestUnitReferencesSeparateCandidateAndPossibleEffectOperands(t *testing.T) {
	units := make([]datfile.UnitSummary, 84)
	units[83] = datfile.UnitSummary{Index: 83, ID: 83, Present: true, Name: "Villager"}
	idx := &datfile.Index{
		Civs: []datfile.Civ{{
			Index: 0,
			Units: units,
		}},
		Effects: []datfile.Effect{{
			Index: 0,
			Name:  "candidate and possible refs",
			Commands: []datfile.EffectCommand{
				semanticCommand(datfile.EffectCommand{Index: 0, Type: 202, A: 83, B: -1, C: 0, D: 1.1}),
				semanticCommand(datfile.EffectCommand{Index: 1, Type: 204, A: 83, B: -1, C: 8, D: 770}),
				{Index: 2, Type: 250, A: 1, B: 2, C: 83, D: 4},
			},
		}},
	}

	refs := unitIDReferences(idx, 83)
	if !referencesWithConfidence(refs, "typed_effect_command_reference") {
		t.Fatalf("typed effect-command reference missing: %+v", refs)
	}
	if !referencesFieldContaining(refs, "command[0].local_building_unit_unit_id") || !referencesFieldContaining(refs, "command[1].local_building_unit_unit_id") {
		t.Fatalf("local-building typed refs missing: %+v", refs)
	}
	if !referencesWithConfidence(refs, "possible_effect_command_operand") {
		t.Fatalf("possible effect-command operand missing: %+v", refs)
	}
	report := References(idx, DeletePlanRequest{Section: "unit", ID: 83})
	if report.Summary.PossibleOperand != 1 || report.Summary.CandidateOperand != 0 || report.Summary.RewriteSupported != 2 {
		t.Fatalf("reference summary = %+v refs=%+v", report.Summary, report.References)
	}
	civID := 0
	plan := DeletePlan(idx, DeletePlanRequest{Section: "unit", ID: 83, CivID: &civID})
	if !plan.Supported || len(plan.References) == 0 || !bytes.Contains(plan.CleanupRecipe, []byte("disconnect_units")) {
		t.Fatalf("delete plan missed local-building references: %+v", plan)
	}
}

func TestKnownEffectCommandsDoNotEmitPossibleOperandNoise(t *testing.T) {
	idx := &datfile.Index{
		Techs: make([]datfile.Tech, 128),
		Effects: []datfile.Effect{{
			Index: 0,
			Name:  "known multiply command",
			Commands: []datfile.EffectCommand{
				semanticCommand(datfile.EffectCommand{Index: 0, Type: 5, A: 83, B: -1, C: 101, D: 1.25}),
				semanticCommand(datfile.EffectCommand{Index: 1, Type: 250, A: 0, B: 0, C: 101, D: 0}),
			},
		}},
	}

	refs := techIDReferences(idx, 101)
	possibleCount := 0
	for _, ref := range refs {
		if ref.Confidence != "possible_effect_command_operand" {
			continue
		}
		possibleCount++
		if !strings.Contains(ref.Field, "command[1].c_possible_tech_id") {
			t.Fatalf("unexpected possible operand ref from known command: %+v all=%+v", ref, refs)
		}
	}
	if possibleCount != 1 {
		t.Fatalf("possible operand refs = %d, want only the unknown command; refs=%+v", possibleCount, refs)
	}
	report := References(idx, DeletePlanRequest{Section: "tech", ID: 101})
	if report.Summary.PossibleOperand != 1 {
		t.Fatalf("possible operand summary = %+v refs=%+v", report.Summary, report.References)
	}
}

func TestFilterReferenceReportCandidateOperand(t *testing.T) {
	report := ReferenceReport{
		References: []ClassifiedReference{
			{DeleteReference: DeleteReference{Section: "effect", ID: 1, Confidence: "possible_effect_command_operand"}, Class: "possible_operand"},
			{DeleteReference: DeleteReference{Section: "effect", ID: 2, Confidence: "candidate_effect_command_operand"}, Class: "possible_operand"},
			{DeleteReference: DeleteReference{Section: "tech", ID: 3, Confidence: "direct"}, Class: "rewrite_supported"},
		},
	}

	filtered := FilterReferenceReport(report, ReferenceFilters{Class: "candidate_operand"})
	if filtered.Summary.Total != 1 || filtered.Summary.PossibleOperand != 1 || filtered.Summary.CandidateOperand != 1 {
		t.Fatalf("candidate filter summary = %+v refs=%+v", filtered.Summary, filtered.References)
	}
	if len(filtered.References) != 1 || filtered.References[0].ID != 2 {
		t.Fatalf("candidate filter refs = %+v", filtered.References)
	}
}

func TestDeletePlanAbilityRules(t *testing.T) {
	idx := &datfile.Index{
		Effects: []datfile.Effect{
			{Index: 0, Name: "base effect"},
			{Index: 1, Name: "tail ability effect"},
		},
		Techs: []datfile.Tech{
			{Index: 0, Name: "base tech", EffectID: 0},
			{Index: 1, Name: "tail ability", EffectID: 1},
		},
	}
	plan := DeletePlan(idx, DeletePlanRequest{Section: "ability", ID: 1})
	if !plan.Supported || !plan.Mutates || plan.Strategy != "paired_tech_then_effect_delete" || len(plan.RecipeHint) == 0 {
		t.Fatalf("tail ability delete plan = %+v", plan)
	}
	if !bytes.Contains(plan.RecipeHint, []byte("delete_techs")) || !bytes.Contains(plan.RecipeHint, []byte("delete_effects")) {
		t.Fatalf("tail ability recipe hint = %s", plan.RecipeHint)
	}
	reports, techDeletes, effectDeletes, err := planAbilityRecipeDeletes(idx, []int{1})
	if err != nil {
		t.Fatalf("ability recipe delete plan failed: %v", err)
	}
	if len(reports) != 1 || reports[0].TechID != 1 || reports[0].EffectID != 1 || len(techDeletes) != 1 || techDeletes[0] != 1 || len(effectDeletes) != 1 || effectDeletes[0] != 1 {
		t.Fatalf("ability recipe delete expansion reports=%+v tech=%v effect=%v", reports, techDeletes, effectDeletes)
	}
	if _, _, _, err := planAbilityRecipeDeletes(idx, []int{1, 1}); err == nil {
		t.Fatal("duplicate ability recipe delete unexpectedly succeeded")
	}
	nonTail := DeletePlan(idx, DeletePlanRequest{Section: "ability", ID: 0})
	if !nonTail.Supported || nonTail.Strategy != "paired_tech_then_effect_delete" || len(nonTail.RecipeHint) == 0 {
		t.Fatalf("non-tail ability delete plan = %+v", nonTail)
	}
	reports, techDeletes, effectDeletes, err = planAbilityRecipeDeletes(idx, []int{0})
	if err != nil {
		t.Fatalf("non-tail ability recipe delete plan failed: %v", err)
	}
	if len(reports) != 1 || reports[0].TechID != 0 || reports[0].EffectID != 0 || len(techDeletes) != 1 || techDeletes[0] != 0 || len(effectDeletes) != 1 || effectDeletes[0] != 0 {
		t.Fatalf("non-tail ability recipe expansion reports=%+v tech=%v effect=%v", reports, techDeletes, effectDeletes)
	}
	referenced := &datfile.Index{
		Effects: []datfile.Effect{
			{Index: 0, Name: "base effect"},
			{Index: 1, Name: "referenced ability effect", Commands: []datfile.EffectCommand{{Type: 0, A: 1}}, CommandSize: 1},
		},
		Techs: []datfile.Tech{
			{Index: 0, Name: "requires ability", EffectID: 0, RequiredTechs: []int16{1}},
			{Index: 1, Name: "referenced tail ability", EffectID: 1},
		},
	}
	referencedPlan := DeletePlan(referenced, DeletePlanRequest{Section: "ability", ID: 1})
	if !referencedPlan.Supported || !referencedPlan.Mutates || referencedPlan.Strategy != "semantic_clear_ability_effect" || len(referencedPlan.References) == 0 || !bytes.Contains(referencedPlan.RecipeHint, []byte("disable_abilities")) {
		t.Fatalf("referenced ability semantic delete plan = %+v", referencedPlan)
	}
	if _, _, _, err := planAbilityRecipeDeletes(referenced, []int{1}); err == nil {
		t.Fatal("delete_abilities unexpectedly accepted semantic ability plan as a physical paired delete")
	}
	cleared, err := patchEffect(referenced.Effects[1], EffectPatchRecipe{ID: 1, Commands: &[]EffectCommandRecipe{}})
	if err != nil {
		t.Fatalf("semantic ability effect clear failed: %v", err)
	}
	if cleared.CommandSize != 0 || len(cleared.Commands) != 0 {
		t.Fatalf("semantic ability effect clear left commands: %+v", cleared)
	}
	shared := &datfile.Index{
		Effects: []datfile.Effect{
			{Index: 0, Name: "shared effect"},
			{Index: 1, Name: "tail effect"},
		},
		Techs: []datfile.Tech{
			{Index: 0, Name: "other user", EffectID: 1},
			{Index: 1, Name: "tail ability", EffectID: 1},
		},
	}
	sharedPlan := DeletePlan(shared, DeletePlanRequest{Section: "ability", ID: 1})
	if sharedPlan.Supported || sharedPlan.Strategy != "unsupported_referenced_ability_delete" || len(sharedPlan.References) < 2 {
		t.Fatalf("shared-effect ability delete plan = %+v", sharedPlan)
	}
}

func firstReferencedEffect(t *testing.T, idx *datfile.Index) int {
	t.Helper()
	for _, tech := range idx.Techs {
		if tech.EffectID >= 0 {
			return int(tech.EffectID)
		}
	}
	t.Fatal("fixture has no referenced effect")
	return 0
}

func taskRowFromSummary(task datfile.TaskSummary) datfile.TaskRow {
	return datfile.TaskRow{
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
	}
}

func firstReferencedTech(t *testing.T, idx *datfile.Index) int {
	t.Helper()
	for id := range idx.Techs {
		if id == len(idx.Techs)-1 {
			continue
		}
		if len(techIDReferences(idx, id)) > 0 {
			return id
		}
	}
	t.Fatal("fixture has no referenced non-tail tech")
	return 0
}

func firstTechWithCommandedEffect(t *testing.T, idx *datfile.Index) int {
	t.Helper()
	for _, tech := range idx.Techs {
		if tech.EffectID < 0 || int(tech.EffectID) >= len(idx.Effects) {
			continue
		}
		if len(idx.Effects[tech.EffectID].Commands) > 0 {
			return tech.Index
		}
	}
	t.Fatal("fixture has no tech with a commanded effect")
	return 0
}

func firstTechReferencedByEffectCommand(t *testing.T, idx *datfile.Index) int {
	t.Helper()
	for _, effect := range idx.Effects {
		for _, command := range effect.Commands {
			values := []int{int(command.A), int(command.B), int(command.C)}
			d := int(command.D)
			if command.D == float32(d) {
				values = append(values, d)
			}
			for _, value := range values {
				if value >= 0 && value < len(idx.Techs) && referencesWithConfidence(techIDReferences(idx, value), "typed_effect_command_reference") {
					return value
				}
			}
		}
	}
	t.Fatal("fixture has no tech id sighting in effect commands")
	return 0
}

func firstTechTreeUnitRef(t *testing.T, idx *datfile.Index) int32 {
	t.Helper()
	for _, connection := range idx.TechTree.UnitConnections {
		if connection.ID >= 0 {
			return connection.ID
		}
		for _, value := range connection.Units {
			if value >= 0 {
				return value
			}
		}
	}
	for _, connection := range idx.TechTree.BuildingConnections {
		if connection.ID >= 0 {
			return connection.ID
		}
		for _, value := range connection.Units {
			if value >= 0 {
				return value
			}
		}
		for _, value := range connection.Buildings {
			if value >= 0 {
				return value
			}
		}
	}
	t.Fatal("fixture has no tech-tree unit reference")
	return 0
}

func firstUnitHeaderTaskObjectRef(t *testing.T, idx *datfile.Index) (headerID int, taskID int, unitID int) {
	t.Helper()
	for _, header := range idx.UnitHeaders {
		if !header.Exists {
			continue
		}
		for _, task := range header.Tasks {
			if task.ObjectID >= 0 && int(task.ObjectID) < len(idx.UnitHeaders) {
				return header.Index, task.Index, int(task.ObjectID)
			}
		}
	}
	t.Fatal("fixture has no unit-header task object_id reference")
	return 0, 0, 0
}

func intSliceContains(values []int, target int) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func firstTechTreeTechRef(t *testing.T, idx *datfile.Index) int32 {
	t.Helper()
	for _, connection := range idx.TechTree.UnitConnections {
		if connection.RequiredResearch >= 0 {
			return connection.RequiredResearch
		}
		if connection.EnablingResearch >= 0 {
			return connection.EnablingResearch
		}
	}
	for _, connection := range idx.TechTree.BuildingConnections {
		if connection.EnablingResearch >= 0 {
			return connection.EnablingResearch
		}
		for _, value := range connection.Techs {
			if value >= 0 {
				return value
			}
		}
	}
	for _, connection := range idx.TechTree.ResearchConnections {
		if connection.ID >= 0 {
			return connection.ID
		}
		for _, value := range connection.Techs {
			if value >= 0 {
				return value
			}
		}
	}
	t.Fatal("fixture has no tech-tree tech reference")
	return 0
}

func firstReferencedCiv(t *testing.T, idx *datfile.Index) int {
	t.Helper()
	for id := range idx.Civs {
		if len(civIDReferences(idx, id)) > 0 {
			return id
		}
	}
	t.Fatal("fixture has no referenced civ")
	return 0
}

func firstReferencedTerrain(t *testing.T, idx *datfile.Index) int {
	t.Helper()
	for id := range idx.Terrains {
		if len(terrainIDReferences(idx, id)) > 0 {
			return id
		}
	}
	t.Fatal("fixture has no referenced terrain")
	return 0
}

func firstReferencedPlayerColour(t *testing.T, idx *datfile.Index) int {
	t.Helper()
	for id := range idx.PlayerColours {
		if len(playerColourIDReferences(idx, id)) > 0 {
			return id
		}
	}
	t.Fatal("fixture has no referenced player colour")
	return 0
}

func firstDropSiteReferencedUnit(t *testing.T, idx *datfile.Index) int {
	t.Helper()
	for _, civ := range idx.Civs {
		for _, unit := range civ.Units {
			if !unit.Present || unit.Action == nil {
				continue
			}
			for _, id := range unit.Action.DropSites {
				if id >= 0 {
					return int(id)
				}
			}
		}
	}
	t.Fatal("fixture has no action drop-site references")
	return 0
}

func firstBloodReferencedUnit(idx *datfile.Index) (int, bool) {
	for _, civ := range idx.Civs {
		for _, unit := range civ.Units {
			if !unit.Present || unit.BloodUnitID < 0 {
				continue
			}
			return int(unit.BloodUnitID), true
		}
	}
	return 0, false
}

func firstLinkedBuildingReferencedUnit(idx *datfile.Index) (int, bool) {
	for _, civ := range idx.Civs {
		for _, unit := range civ.Units {
			if !unit.Present || unit.Building == nil {
				continue
			}
			for _, row := range unit.Building.LinkedBuildings {
				if !row.Active {
					continue
				}
				return int(row.UnitID), true
			}
		}
	}
	return 0, false
}

func referencesFieldContaining(refs []DeleteReference, needle string) bool {
	for _, ref := range refs {
		if strings.Contains(ref.Field, needle) {
			return true
		}
	}
	return false
}

func referencesWithConfidence(refs []DeleteReference, confidence string) bool {
	for _, ref := range refs {
		if ref.Confidence == confidence {
			return true
		}
	}
	return false
}

func anyUnitRewriteSupported(refs []DeleteReference) bool {
	for _, ref := range refs {
		if unitReferenceRewriteCovers(ref) {
			return true
		}
	}
	return false
}

func semanticHasRef(semantic *datfile.EffectCommandSemantic, kind, field string, id int) bool {
	for _, ref := range semantic.References {
		if ref.Kind == kind && ref.Field == field && ref.ID == id {
			return true
		}
	}
	if semantic.ReferenceID == nil {
		return false
	}
	return semantic.ReferenceKind == kind && semantic.ReferenceField == field && *semantic.ReferenceID == id
}

func referenceCountsContain(refs []EffectCommandReferenceCount, kind, field string, id int) bool {
	for _, ref := range refs {
		if ref.Kind != kind || ref.Field != field {
			continue
		}
		for _, sample := range ref.SampleIDs {
			if sample.Value == id {
				return true
			}
		}
	}
	return false
}

func firstSoundWithItem(t *testing.T, idx *datfile.Index) (int, int) {
	t.Helper()
	for _, sound := range idx.Sounds {
		if len(sound.Items) > 0 {
			return sound.Index, sound.Items[0].Index
		}
	}
	t.Fatal("fixture has no sound with items")
	return 0, 0
}

func firstEnabledUnit(t *testing.T, idx *datfile.Index) (int, int) {
	t.Helper()
	for _, civ := range idx.Civs {
		for _, unit := range civ.Units {
			if unit.Present && unit.Enabled != 0 {
				return civ.Index, unit.Index
			}
		}
	}
	t.Fatal("fixture has no enabled unit")
	return 0, 0
}

func firstUnitWithDeletableRows(t *testing.T, idx *datfile.Index, count func(datfile.UnitSummary) int) (int, int, int) {
	t.Helper()
	for _, civ := range idx.Civs {
		for _, unit := range civ.Units {
			if !unit.Present {
				continue
			}
			if count(unit) > 0 {
				return civ.Index, unit.Index, 0
			}
		}
	}
	t.Fatal("fixture has no matching unit rows")
	return 0, 0, 0
}

func firstUnitHeaderWithTasks(t *testing.T, idx *datfile.Index) (int, int) {
	t.Helper()
	for _, header := range idx.UnitHeaders {
		if header.Exists && len(header.Tasks) > 0 {
			return header.Index, 0
		}
	}
	t.Fatal("fixture has no unit header tasks")
	return 0, 0
}

func firstGraphicWithDeltas(t *testing.T, idx *datfile.Index) (int, int) {
	t.Helper()
	for _, graphic := range idx.Graphics {
		if graphic.Present && len(graphic.Deltas) > 0 {
			return graphic.Index, 0
		}
	}
	t.Fatal("fixture has no graphic deltas")
	return 0, 0
}

func firstGraphicWithAngleSounds(t *testing.T, idx *datfile.Index) (int, int, bool) {
	t.Helper()
	for _, graphic := range idx.Graphics {
		if graphic.Present && len(graphic.AngleSounds) > 0 {
			return graphic.Index, 0, true
		}
	}
	return 0, 0, false
}

func intPtr(value int) *int {
	return &value
}

func int16Ptr(value int16) *int16 {
	return &value
}

func int32Ptr(value int32) *int32 {
	return &value
}

func int32SlicePtr(values []int32) *[]int32 {
	return &values
}

func float32Ptr(value float32) *float32 {
	return &value
}

func int32SlicesEqual(a, b []int32) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func uint8SlicesEqual(a, b []uint8) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
