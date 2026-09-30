package datcodec

import (
	"strings"
	"testing"

	"aoe2kit/pkg/datfile"
)

func TestEffectCommandMatrixSummarizesTypedReferences(t *testing.T) {
	units := make([]datfile.UnitSummary, 84)
	units[74] = datfile.UnitSummary{Index: 74, ID: 74, Present: true, Name: "Militia"}
	units[75] = datfile.UnitSummary{Index: 75, ID: 75, Present: true, Name: "Man-at-Arms"}
	units[83] = datfile.UnitSummary{Index: 83, ID: 83, Present: true, Name: "Villager"}
	techs := make([]datfile.Tech, 245)
	techs[244] = datfile.Tech{Index: 244, Name: "Loom"}
	idx := &datfile.Index{
		Civs: []datfile.Civ{
			{Index: 0, Units: units},
		},
		Techs: techs,
		Effects: []datfile.Effect{
			{
				Index: 0,
				Name:  "test upgrades",
				Commands: []datfile.EffectCommand{
					semanticCommand(datfile.EffectCommand{Index: 0, Type: 3, A: 74, B: 75, C: -1, D: 0}),
					semanticCommand(datfile.EffectCommand{Index: 1, Type: 102, A: -1, B: -1, C: -1, D: 244}),
					semanticCommand(datfile.EffectCommand{Index: 2, Type: 4, A: 83, B: -1, C: 9, D: 1027}),
				},
			},
			{
				Index: 1,
				Name:  "test hp",
				Commands: []datfile.EffectCommand{
					semanticCommand(datfile.EffectCommand{Index: 0, Type: 4, A: 83, B: -1, C: 0, D: 50}),
				},
			},
		},
	}

	report := EffectCommandMatrix(idx, 2)
	if report.CommandCount != 4 || report.TypeCount != 3 {
		t.Fatalf("matrix summary = %+v", report)
	}
	upgrade := commandTypeProfile(t, report, 3)
	if upgrade.TypeName != "upgrade_unit" || upgrade.CommandCount != 1 || upgrade.EffectCount != 1 {
		t.Fatalf("upgrade profile = %+v", upgrade)
	}
	if !profileHasReference(upgrade, "unit", "source_unit", 1) || !profileHasReference(upgrade, "unit", "target_unit", 1) {
		t.Fatalf("upgrade refs = %+v", upgrade.TypedReferences)
	}
	if !profileHasReferenceName(upgrade, "unit", "source_unit", "Militia") || !profileHasReferenceName(upgrade, "unit", "target_unit", "Man-at-Arms") {
		t.Fatalf("upgrade refs missing names = %+v", upgrade.TypedReferences)
	}
	if len(upgrade.Examples) != 1 || len(upgrade.Examples[0].References) != 2 {
		t.Fatalf("upgrade examples = %+v", upgrade.Examples)
	}
	disable := commandTypeProfile(t, report, 102)
	if !profileHasReference(disable, "tech", "amount", 1) {
		t.Fatalf("disable refs = %+v", disable.TypedReferences)
	}
	if !profileHasReferenceName(disable, "tech", "amount", "Loom") {
		t.Fatalf("disable refs missing names = %+v", disable.TypedReferences)
	}
	add := commandTypeProfile(t, report, 4)
	if len(add.Attributes) == 0 || add.Attributes[0].Value != 0 || add.Attributes[0].Name != "hit_points" {
		t.Fatalf("attribute profile = %+v", add.Attributes)
	}
	if len(add.PackedTypeIDs) != 1 || add.PackedTypeIDs[0].Value != 4 ||
		len(add.PackedAmounts) != 1 || add.PackedAmounts[0].Value != 3 {
		t.Fatalf("packed attack profile types=%+v amounts=%+v", add.PackedTypeIDs, add.PackedAmounts)
	}
}

func TestEffectCommandMatrixFiltersByReferenceAndUnknown(t *testing.T) {
	idx := &datfile.Index{
		Techs: []datfile.Tech{{Index: 0, Name: "probe tech"}},
		Effects: []datfile.Effect{
			{
				Index: 0,
				Name:  "mixed",
				Commands: []datfile.EffectCommand{
					semanticCommand(datfile.EffectCommand{Index: 0, Type: 102, A: -1, B: -1, C: -1, D: 0}),
					semanticCommand(datfile.EffectCommand{Index: 1, Type: 250, A: 10, B: 20, C: 30, D: 40}),
					semanticCommand(datfile.EffectCommand{Index: 2, Type: 4, A: 83, B: -1, C: 0, D: 50}),
				},
			},
		},
	}

	techOnly := EffectCommandMatrixWithOptions(idx, EffectCommandMatrixOptions{ReferenceKind: "tech"})
	if techOnly.CommandCount != 1 || techOnly.TypeCount != 1 || techOnly.CommandTypes[0].Type != 102 {
		t.Fatalf("tech filter = %+v", techOnly)
	}
	if techOnly.Filters.ReferenceKind != "tech" {
		t.Fatalf("filters not echoed = %+v", techOnly.Filters)
	}

	unknownOnly := EffectCommandMatrixWithOptions(idx, EffectCommandMatrixOptions{UnknownOnly: true})
	if unknownOnly.CommandCount != 1 || unknownOnly.TypeCount != 1 || unknownOnly.CommandTypes[0].Type != 250 {
		t.Fatalf("unknown filter = %+v", unknownOnly)
	}
	operandOnly := EffectCommandMatrixWithOptions(idx, EffectCommandMatrixOptions{Operand: intPtr(83)})
	if operandOnly.CommandCount != 1 || operandOnly.TypeCount != 1 || operandOnly.CommandTypes[0].Type != 4 {
		t.Fatalf("operand filter = %+v", operandOnly)
	}
	if operandOnly.Filters.Operand == nil || *operandOnly.Filters.Operand != 83 {
		t.Fatalf("operand filter not echoed = %+v", operandOnly.Filters)
	}

	typedUnits := EffectCommandMatrixWithOptions(idx, EffectCommandMatrixOptions{TypedOnly: true, ReferenceKind: "unit"})
	if typedUnits.CommandCount != 1 || typedUnits.TypeCount != 1 || typedUnits.CommandTypes[0].Type != 4 {
		t.Fatalf("typed unit filter = %+v", typedUnits)
	}
}

func TestEffectCommandMatrixSeparatesCandidateReferences(t *testing.T) {
	units := make([]datfile.UnitSummary, 2085)
	units[82] = datfile.UnitSummary{Index: 82, ID: 82, Present: true, Name: "Castle"}
	units[2084] = datfile.UnitSummary{Index: 2084, ID: 2084, Present: true, Name: "WCHICKENA"}
	idx := &datfile.Index{
		Civs: []datfile.Civ{{Index: 0, Units: units}},
		Effects: []datfile.Effect{{
			Index: 0,
			Name:  "candidate split",
			Commands: []datfile.EffectCommand{
				semanticCommand(datfile.EffectCommand{Index: 0, Type: 40, A: 2084, B: -1, C: 50, D: 21220}),
				semanticCommand(datfile.EffectCommand{Index: 1, Type: 201, A: 82, B: -1, C: 1, D: 4}),
				semanticCommand(datfile.EffectCommand{Index: 2, Type: 202, A: 82, B: -1, C: 0, D: 1.1}),
				semanticCommand(datfile.EffectCommand{Index: 3, Type: 250, A: 82, B: -1, C: 1, D: 4}),
			},
		}},
	}

	report := EffectCommandMatrix(idx, 0)
	rename := commandTypeProfile(t, report, 40)
	if !profileHasReference(rename, "unit", "target_unit", 1) || len(rename.CandidateRefs) != 0 {
		t.Fatalf("rename profile refs=%+v candidates=%+v", rename.TypedReferences, rename.CandidateRefs)
	}
	localBuilding := commandTypeProfile(t, report, 201)
	if localBuilding.TypeName != "add_local_building_attribute" ||
		!profileHasReferenceName(localBuilding, "unit", "local_building_unit", "Castle") ||
		len(localBuilding.CandidateRefs) != 0 {
		t.Fatalf("local-building profile refs=%+v candidates=%+v", localBuilding.TypedReferences, localBuilding.CandidateRefs)
	}
	arbitrary := commandTypeProfile(t, report, 250)
	if len(arbitrary.CandidateRefs) != 0 {
		t.Fatalf("arbitrary profile candidates=%+v", arbitrary.CandidateRefs)
	}
	localMultiply := commandTypeProfile(t, report, 202)
	if localMultiply.TypeName != "multiply_local_building_attribute" ||
		!profileHasReferenceName(localMultiply, "unit", "local_building_unit", "Castle") ||
		len(localMultiply.CandidateRefs) != 0 {
		t.Fatalf("local multiply profile refs=%+v candidates=%+v", localMultiply.TypedReferences, localMultiply.CandidateRefs)
	}

	candidateOnly := EffectCommandMatrixWithOptions(idx, EffectCommandMatrixOptions{CandidateOnly: true, ReferenceKind: "unit"})
	if candidateOnly.CommandCount != 0 || candidateOnly.TypeCount != 0 {
		t.Fatalf("candidate-only filter = %+v", candidateOnly)
	}
}

func TestExplainEffectLocalBuildingPackedRows(t *testing.T) {
	units := make([]datfile.UnitSummary, 83)
	units[82] = datfile.UnitSummary{Index: 82, ID: 82, Present: true, Name: "Castle"}
	idx := &datfile.Index{
		Civs:    []datfile.Civ{{Index: 0, Units: units}},
		Effects: make([]datfile.Effect, 1267),
	}
	idx.Effects[1266] = datfile.Effect{
		Index: 1266,
		Name:  "local building probe",
		Commands: []datfile.EffectCommand{
			semanticCommand(datfile.EffectCommand{Index: 0, Type: 202, A: 82, B: -1, C: 108, D: 6}),
			semanticCommand(datfile.EffectCommand{Index: 1, Type: 204, A: 82, B: -1, C: 9, D: 771}),
		},
	}

	report, err := ExplainEffect(idx, 1266)
	if err != nil {
		t.Fatal(err)
	}
	if report.CommandCount != 2 || !strings.Contains(report.Summary, "local-building") {
		t.Fatalf("effect explain summary = %+v", report)
	}
	if !strings.Contains(report.Commands[0].Summary, "multiply local 82/Castle garrison_heal_rate by 6") {
		t.Fatalf("local multiply summary = %q", report.Commands[0].Summary)
	}
	if !strings.Contains(report.Commands[1].Summary, "attack[pierce +3]") {
		t.Fatalf("packed local attack summary = %q details=%+v", report.Commands[1].Summary, report.Commands[1].Details)
	}
	if len(report.Warnings) == 0 || !strings.Contains(strings.Join(report.Warnings, "\n"), "strong_hypothesis") {
		t.Fatalf("local-building warnings = %+v", report.Warnings)
	}
	if !report.Verification.StructureVerified || report.Verification.EngineVerified {
		t.Fatalf("verification claim = %+v", report.Verification)
	}
}

func TestRecipeAuthoringWarnings(t *testing.T) {
	warnings := RecipeAuthoringWarnings(Recipe{
		CreateEffect: &EffectCreateRecipe{
			Name: "warning probe",
			Commands: []EffectCommandRecipe{
				{Kind: "spawn_unit", UnitID: int16Ptr(83), BuildingID: int16Ptr(109), Amount: recipeFloat32Ptr(1)},
				{Kind: "multiply_local_building_attribute", UnitID: int16Ptr(82), AttributeName: "garrison_heal_rate", Amount: recipeFloat32Ptr(6)},
				{Type: 203, A: 82, B: -1, C: 0, D: 7},
			},
		},
	})
	joined := strings.Join(warnings, "\n")
	if len(warnings) < 3 || !strings.Contains(joined, "spawn_unit") || !strings.Contains(joined, "202/204") || !strings.Contains(joined, "external_oracle_unverified") || !strings.Contains(joined, "unknown effect command type") {
		t.Fatalf("warnings = %+v", warnings)
	}
}

func TestRecipeAuthoringWarningsDoNotWarnForSourceCorrelatedCommand(t *testing.T) {
	warnings := RecipeAuthoringWarnings(Recipe{
		CreateEffect: &EffectCreateRecipe{
			Name: "quiet probe",
			Commands: []EffectCommandRecipe{
				{Kind: "set_unit_attribute", UnitID: int16Ptr(74), AttributeName: "hit_points", Amount: recipeFloat32Ptr(100)},
			},
		},
	})
	if len(warnings) != 0 {
		t.Fatalf("warnings = %+v", warnings)
	}
}

func semanticCommand(command datfile.EffectCommand) datfile.EffectCommand {
	command.Semantic = datfile.InterpretEffectCommand(command)
	return command
}

func commandTypeProfile(t *testing.T, report EffectCommandMatrixReport, typ uint8) EffectCommandTypeProfile {
	t.Helper()
	for _, profile := range report.CommandTypes {
		if profile.Type == typ {
			return profile
		}
	}
	t.Fatalf("missing command type %d in %+v", typ, report.CommandTypes)
	return EffectCommandTypeProfile{}
}

func profileHasReference(profile EffectCommandTypeProfile, kind, field string, count int) bool {
	for _, ref := range profile.TypedReferences {
		if ref.Kind == kind && ref.Field == field && ref.Count == count {
			return true
		}
	}
	return false
}

func profileHasReferenceName(profile EffectCommandTypeProfile, kind, field, name string) bool {
	for _, ref := range profile.TypedReferences {
		if ref.Kind != kind || ref.Field != field {
			continue
		}
		for _, sample := range ref.SampleIDs {
			if sample.Name == name {
				return true
			}
		}
	}
	return false
}

func profileHasCandidateReferenceName(profile EffectCommandTypeProfile, kind, field, name string) bool {
	for _, ref := range profile.CandidateRefs {
		if ref.Kind != kind || ref.Field != field {
			continue
		}
		for _, sample := range ref.SampleIDs {
			if sample.Name == name {
				return true
			}
		}
	}
	return false
}
