package rpgshop

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"aoe2kit/pkg/scenario"
)

func TestBuildHeroShopDemoWritesFreshScenarioAndArtifacts(t *testing.T) {
	dir := t.TempDir()
	report, err := BuildHeroShopDemo(DemoOptions{OutputDir: dir, Timestamp: 1788780904})
	if err != nil {
		t.Fatalf("BuildHeroShopDemo: %v", err)
	}
	for _, path := range []string{report.ScenarioPath, report.BaseScenarioPath, report.RecipePath, report.XSPath, report.InstructionsPath, report.ManifestPath} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("missing artifact %s: %v", path, err)
		}
	}
	if report.BlankReport.UnitCountAfter != 3 {
		t.Fatalf("blank starter units = %d, want 3", report.BlankReport.UnitCountAfter)
	}
	if report.PatchReport.TriggerCountAfter != 17 {
		t.Fatalf("trigger count = %d, want 17", report.PatchReport.TriggerCountAfter)
	}
	if len(report.Variables) != 27 {
		t.Fatalf("variable count = %d, want 27", len(report.Variables))
	}
	if len(report.ShopItems) != 4 {
		t.Fatalf("shop items = %d, want 4", len(report.ShopItems))
	}
	if report.Earning.GoldPerKill != 1 || report.Earning.XPPerKill != 25 || report.Earning.Signal != "xsPlayerAttribute(player, 300+victim_player)" {
		t.Fatalf("earning report = %+v", report.Earning)
	}
	scen, err := scenario.Open(report.ScenarioPath)
	if err != nil {
		t.Fatalf("open demo scenario: %v", err)
	}
	if err := scen.VerifyRebuild(); err != nil {
		t.Fatalf("VerifyRebuild: %v", err)
	}
	census, err := scen.XSCensus(scenario.XSCensusOptions{})
	if err != nil {
		t.Fatalf("XSCensus: %v", err)
	}
	if census.ScriptContentEmbedded || census.Counts.ScriptContentAttachments != 0 {
		t.Fatalf("script-content XS unexpectedly present in runtime-inline demo: %+v", census.Counts)
	}
	if census.Counts.EmbeddedCarriers != 1 {
		t.Fatalf("embedded carrier count = %d, want 1", census.Counts.EmbeddedCarriers)
	}
	if len(census.EmbeddedCarriers) != 1 || census.EmbeddedCarriers[0].TriggerIndex != 0 || census.EmbeddedCarriers[0].Title != "XS string" {
		t.Fatalf("runtime XS carrier = %+v", census.EmbeddedCarriers)
	}
	if scen.Triggers.Triggers[0].Enabled != 1 {
		t.Fatalf("runtime XS carrier trigger enabled=%d, want 1", scen.Triggers.Triggers[0].Enabled)
	}
	bootTrigger := findTriggerSummary(scen.Triggers.Triggers, "A2KRPG 002 Boot")
	if bootTrigger == nil {
		t.Fatalf("missing parsed boot trigger summary")
	}
	addTrainEffect := findEffectSummary(bootTrigger.EffectData, "add_train_location", unitMilitia)
	if addTrainEffect == nil || knownInt(addTrainEffect.KnownFields, "train_location_unit_const") != unitBarracks || knownInt(addTrainEffect.KnownFields, "button_location") != 1 || knownInt(addTrainEffect.KnownFields, "train_time") != 0 {
		t.Fatalf("add_train_location effect summary = %+v", addTrainEffect)
	}
	costEffect := findEffectSummary(bootTrigger.EffectData, "change_object_cost", unitMilitia)
	if costEffect == nil || knownInt(costEffect.KnownFields, "resource_3") != resourceGold || knownInt(costEffect.KnownFields, "resource_3_quantity") != 3 {
		t.Fatalf("change_object_cost effect summary = %+v", costEffect)
	}
	hpName := findEffectText(bootTrigger.EffectData, "change_object_name", unitMilitia, "+HP")
	if hpName == nil {
		t.Fatalf("missing +HP change_object_name effect")
	}
	hpDescription := findEffectText(bootTrigger.EffectData, "change_object_description", unitMilitia, "+60 HP to your units (3 gold)")
	if hpDescription == nil {
		t.Fatalf("missing +HP change_object_description effect")
	}
	attackName := findEffectText(bootTrigger.EffectData, "change_object_name", unitSpearman, "+Attack")
	if attackName == nil {
		t.Fatalf("missing +Attack change_object_name effect")
	}
	attackDescription := findEffectText(bootTrigger.EffectData, "change_object_description", unitSpearman, "+4 Attack to your units (5 gold)")
	if attackDescription == nil {
		t.Fatalf("missing +Attack change_object_description effect")
	}
	purchaseTrigger := findTriggerSummary(scen.Triggers.Triggers, "A2KRPG P1H1 Buy hero_hp")
	if purchaseTrigger == nil {
		t.Fatalf("missing parsed purchase trigger summary")
	}
	if purchaseTrigger.Conditions != 2 || len(purchaseTrigger.ConditionData) != 2 {
		t.Fatalf("purchase condition summary count=%d data=%d, want 2/2", purchaseTrigger.Conditions, len(purchaseTrigger.ConditionData))
	}
	ownObjects := purchaseTrigger.ConditionData[0]
	if ownObjects.TypeName != "own_objects" || knownInt(ownObjects.KnownFields, "source_player") != 1 || knownInt(ownObjects.KnownFields, "object_list") != unitMilitia || knownInt(ownObjects.KnownFields, "quantity") != 1 {
		t.Fatalf("own_objects condition summary = %+v", ownObjects)
	}
	variableValue := purchaseTrigger.ConditionData[1]
	if variableValue.TypeName != "variable_value" || knownInt(variableValue.KnownFields, "variable") != varP1H1Coin || knownInt(variableValue.KnownFields, "comparison") != cmpGE || knownInt(variableValue.KnownFields, "quantity") != 3 {
		t.Fatalf("variable_value condition summary = %+v", variableValue)
	}
	attachment := scen.XSAttachment()
	if attachment.ScriptName != "" || attachment.ScriptFilePath != "" || attachment.ScriptFileContentBytes != 0 {
		t.Fatalf("external XS attachment unexpectedly present: %+v", attachment)
	}
	stringsReport := scen.Strings()
	if !hasVariableRef(stringsReport, "Variable 132") {
		t.Fatalf("missing objective variable substitution refs: %#v", stringsReport.MarkupSummary.VariableRefs)
	}
	if hasVariableRef(stringsReport, "A2K HERO P1 H1 COINS") {
		t.Fatalf("declared variable name was misclassified as substitution ref: %#v", stringsReport.MarkupSummary.VariableRefs)
	}
	for _, removed := range []string{"--- SHOP (Barracks) ---", "Militia = +HP (3 gold)", "Spearman = +Attack (5 gold)"} {
		if stringReportContains(stringsReport, removed) {
			t.Fatalf("obsolete objective shop legend line still present %q", removed)
		}
	}
	for _, want := range []string{"+HP", "+60 HP to your units (3 gold)", "+Attack", "+4 Attack to your units (5 gold)"} {
		if !stringReportContains(stringsReport, want) {
			t.Fatalf("missing renamed shop button text %q", want)
		}
	}
	xs, err := os.ReadFile(report.XSPath)
	if err != nil {
		t.Fatalf("read xs: %v", err)
	}
	for _, want := range []string{"A2KRPG_Buy", "A2KRPG_SCOPE_PARTY", "A2KRPG_SCOPE_PLAYER", "A2KRPG_SCOPE_HERO", "xsTriggerVariable", "xsSetTriggerVariable", "int killAttribute = A2KRPG_ATTR_PLAYER_KILLS_BASE + victim;", "xsPlayerAttribute(player, killAttribute)", "AddPartyXP", "A2KRPG_PollKills"} {
		if !strings.Contains(string(xs), want) {
			t.Fatalf("xs module missing %q", want)
		}
	}
	for _, variable := range report.Variables {
		if variable.ID >= 256 {
			t.Fatalf("demo variable %s id=%d, want <256 for live objective substitution", variable.Name, variable.ID)
		}
	}
}

func TestDemoRecipeCarriesShopAndCurrencyPrimitives(t *testing.T) {
	recipe := DemoRecipe(filepath.Join("tmp", xsFileName), 1788780904)
	if recipe.XS == nil || recipe.XS.Mode != "inline_runtime" {
		t.Fatalf("xs recipe = %#v", recipe.XS)
	}
	var objectiveCount, addTrainCount, costCount, renameCount, descriptionCount, tokenAttrCount, pollCount, purchaseCount int
	var nonLandTerrainCount int
	for _, trigger := range recipe.Triggers {
		if trigger.DisplayAsObjective != nil && *trigger.DisplayAsObjective {
			objectiveCount++
		}
		if trigger.Name == "A2KRPG 020 Poll Directed Kill XP" {
			pollCount++
		}
		if strings.Contains(trigger.Name, "Buy ") {
			purchaseCount++
		}
		for _, effect := range trigger.Effects {
			if effect.Op == "add_train_location" {
				addTrainCount++
				if effect.ObjectListUnitID == nil || (*effect.ObjectListUnitID != unitMilitia && *effect.ObjectListUnitID != unitSpearman) {
					t.Fatalf("shop train token = %#v, want militia or spearman token", effect.ObjectListUnitID)
				}
				if effect.ObjectListUnitID2 == nil || *effect.ObjectListUnitID2 != unitBarracks {
					t.Fatalf("shop train location = %#v, want barracks", effect.ObjectListUnitID2)
				}
				if effect.TrainTime == nil || *effect.TrainTime != 0 {
					t.Fatalf("shop train time = %#v, want instant 0", effect.TrainTime)
				}
			}
			if effect.Op == "change_object_cost" {
				costCount++
				if effect.ObjectListUnitID == nil || (*effect.ObjectListUnitID != unitMilitia && *effect.ObjectListUnitID != unitSpearman) {
					t.Fatalf("cost token = %#v, want militia or spearman token", effect.ObjectListUnitID)
				}
				if effect.Resource1 == nil || *effect.Resource1 != resourceFood || effect.Resource1Quantity == nil || *effect.Resource1Quantity != 0 {
					t.Fatalf("food cost wiring = resource %#v quantity %#v", effect.Resource1, effect.Resource1Quantity)
				}
				if effect.Resource2 == nil || *effect.Resource2 != resourceWood || effect.Resource2Quantity == nil || *effect.Resource2Quantity != 0 {
					t.Fatalf("wood cost wiring = resource %#v quantity %#v", effect.Resource2, effect.Resource2Quantity)
				}
				if effect.Resource3 == nil || *effect.Resource3 != resourceGold || effect.Resource3Quantity == nil || (*effect.Resource3Quantity != 3 && *effect.Resource3Quantity != 5) {
					t.Fatalf("gold cost wiring = resource %#v quantity %#v", effect.Resource3, effect.Resource3Quantity)
				}
			}
			if effect.Op == "change_object_name" {
				renameCount++
			}
			if effect.Op == "change_object_description" {
				descriptionCount++
			}
			if effect.Op == "modify_attribute" && effect.ObjectListUnitID != nil && (*effect.ObjectListUnitID == unitMilitia || *effect.ObjectListUnitID == unitSpearman) {
				tokenAttrCount++
			}
		}
	}
	for _, patch := range recipe.Map {
		if patch.TerrainID != nil && *patch.TerrainID != 0 {
			nonLandTerrainCount++
		}
	}
	if objectiveCount != 6 {
		t.Fatalf("objective triggers = %d, want 6", objectiveCount)
	}
	if nonLandTerrainCount != 0 {
		t.Fatalf("non-land terrain patches = %d, want 0", nonLandTerrainCount)
	}
	if addTrainCount != 4 {
		t.Fatalf("add_train_location effects = %d, want 4", addTrainCount)
	}
	if costCount != 4 {
		t.Fatalf("change_object_cost effects = %d, want 4", costCount)
	}
	if renameCount != 4 {
		t.Fatalf("change_object_name effects = %d, want 4", renameCount)
	}
	if descriptionCount != 4 {
		t.Fatalf("change_object_description effects = %d, want 4", descriptionCount)
	}
	if tokenAttrCount != 24 {
		t.Fatalf("token modify_attribute effects = %d, want 24", tokenAttrCount)
	}
	if pollCount != 1 {
		t.Fatalf("poll triggers = %d, want 1", pollCount)
	}
	if purchaseCount != 4 {
		t.Fatalf("purchase triggers = %d, want 4", purchaseCount)
	}
}

func hasVariableRef(report scenario.StringReport, name string) bool {
	for _, ref := range report.MarkupSummary.VariableRefs {
		if ref == name {
			return true
		}
	}
	return false
}

func stringReportContains(report scenario.StringReport, text string) bool {
	for _, value := range report.Messages {
		if strings.Contains(value, text) {
			return true
		}
	}
	for _, entry := range report.StringTable {
		if strings.Contains(entry.Text, text) {
			return true
		}
	}
	for _, entry := range report.TriggerText {
		if strings.Contains(entry.Text, text) {
			return true
		}
	}
	for _, entry := range report.EffectText {
		if strings.Contains(entry.Text, text) {
			return true
		}
	}
	return false
}

func findTriggerSummary(triggers []scenario.TriggerSummary, name string) *scenario.TriggerSummary {
	for i := range triggers {
		if triggers[i].Name == name {
			return &triggers[i]
		}
	}
	return nil
}

func findEffectSummary(effects []scenario.EffectSummary, typeName string, unitConst int) *scenario.EffectSummary {
	for i := range effects {
		if effects[i].TypeName == typeName && effects[i].UnitConst == unitConst {
			return &effects[i]
		}
	}
	return nil
}

func findEffectText(effects []scenario.EffectSummary, typeName string, unitConst int, text string) *scenario.EffectSummary {
	for i := range effects {
		if effects[i].TypeName == typeName && effects[i].UnitConst == unitConst && effects[i].Text == text {
			return &effects[i]
		}
	}
	return nil
}

func knownInt(fields map[string]any, name string) int {
	value, _ := fields[name].(int)
	return value
}
