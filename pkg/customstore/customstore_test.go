package customstore

import (
	"os"
	"strings"
	"testing"

	"aoe2kit/pkg/scenario"
	"aoe2kit/pkg/xs"
)

func TestRecipeIsParameterizedAndKeepsConstantTriggerShape(t *testing.T) {
	recipe := Recipe("customstore.xs", 1780000000, 236, 440, 8)
	if len(recipe.Units) != DefaultInstances {
		t.Fatalf("object units=%d want %d", len(recipe.Units), DefaultInstances)
	}
	if len(recipe.Triggers) != AddedTriggerCount {
		t.Fatalf("triggers=%d want %d", len(recipe.Triggers), TriggerCount)
	}
	seen := map[int]bool{}
	for _, unit := range recipe.Units {
		if seen[*unit.ReferenceID] {
			t.Fatalf("duplicate reference id %d", *unit.ReferenceID)
		}
		seen[*unit.ReferenceID] = true
	}
	if len(seen) != DefaultInstances {
		t.Fatalf("unique refs=%d want %d", len(seen), DefaultInstances)
	}
	module := XSModule(236, 440, 8)
	if !strings.Contains(module, "A2K_CAPACITY = 440") || !strings.Contains(module, "CustomStore_Consume") || !strings.Contains(module, "CustomStore_FireRing") || !strings.Contains(module, "CustomStore_FlowerRing") {
		t.Fatal("XS module missing bounded store API")
	}
	if !strings.Contains(module, "xsCreateFile(true)") || !strings.Contains(module, "xsCloseFile();") {
		t.Fatal("XS module does not close append writes for durable sidecar rows")
	}
	if !strings.Contains(module, "A2K_FLOWER_UNIT = 1366") || !strings.Contains(module, "xsSetUnitPosition") {
		t.Fatal("XS module missing non-HP Gaia-art retirement path")
	}
	if len(recipe.Variables) != 1 || recipe.Variables[0].Name != "A2K CustomStore Gaia sweep" {
		t.Fatalf("sweep variable=%+v", recipe.Variables)
	}
	if len(recipe.Triggers) != 3+KillSweepSeconds || recipe.Triggers[3].Effects[0].ObjectListUnitID == nil || *recipe.Triggers[3].Effects[0].ObjectListUnitID != FlowerUnitConst {
		t.Fatalf("kill sweep recipe shape is incorrect: triggers=%d", len(recipe.Triggers))
	}
	setup := recipe.Triggers[0]
	costs := map[int][2]int{}
	for _, effect := range setup.Effects {
		if effect.Op != "change_object_cost" || effect.ObjectListUnitID == nil || effect.Resource1 == nil || effect.Resource1Quantity == nil {
			continue
		}
		if effect.Resource2Quantity == nil || effect.Resource3Quantity == nil {
			t.Fatalf("cost effect missing complete resource slots: %+v", effect)
		}
		costs[*effect.ObjectListUnitID] = [2]int{*effect.Resource2Quantity, *effect.Resource3Quantity}
	}
	if costs[TokenUpgradeUnit] != [2]int{0, 100} || costs[TokenInspectUnit] != [2]int{100, 0} {
		t.Fatalf("token costs=%v, want upgrade gold=100 and inspect wood=100", costs)
	}
	for _, effect := range setup.Effects {
		if effect.Op == "enable_disable_object" && effect.ObjectListUnitID != nil && *effect.ObjectListUnitID == TokenUpgradeUnit {
			if effect.Enabled == nil || *effect.Enabled != 1 {
				t.Fatalf("upgrade token is not explicitly enabled: %+v", effect)
			}
		}
	}
	for _, effect := range setup.Effects {
		if effect.Op == "modify_attribute" && effect.ObjectListUnitID != nil && *effect.ObjectListUnitID == TokenUpgradeUnit {
			if effect.ObjectAttributes == nil || *effect.ObjectAttributes != 110 || effect.Operation == nil || *effect.Operation != 1 || effect.Quantity == nil || *effect.Quantity != 0 {
				t.Fatalf("upgrade token population cost is not zeroed: %+v", effect)
			}
		}
	}
	for _, effect := range setup.Effects {
		if effect.Op != "change_object_cost" {
			continue
		}
		if effect.Resource1 == nil || effect.Resource2 == nil || effect.Resource3 == nil ||
			*effect.Resource1 != 0 || *effect.Resource2 != 1 || *effect.Resource3 != 3 {
			t.Fatalf("cost effect uses invalid resource slots: %+v", effect)
		}
	}
	if findings := xs.LintSource("A2K_CUSTOM_STORE.xs", module); len(findings) != 0 {
		t.Fatalf("XS lint findings: %+v", findings)
	}
	if len(Recipe("customstore.xs", 1780000000, 82, 12, 4).Units) != 12 {
		t.Fatal("parameterized object count was not honored")
	}
}

func TestRecipeForPlayersRestrictsStoreSetup(t *testing.T) {
	recipe := RecipeForPlayers("customstore.xs", 1780000000, 236, 440, 8, []int{1})
	setup := recipe.Triggers[0]
	for _, effect := range setup.Effects {
		if effect.SourcePlayer == nil {
			continue
		}
		if *effect.SourcePlayer != 1 {
			t.Fatalf("non-participant store effect targets P%d: %+v", *effect.SourcePlayer, effect)
		}
	}
	module := XSModuleForPlayers(236, 440, 8, []int{1})
	if !strings.Contains(module, "CustomStore_Consume(1, A2K_TOKEN_UPGRADE, 1)") {
		t.Fatal("participant P1 is not consumed")
	}
	if strings.Contains(module, "CustomStore_Consume(2, A2K_TOKEN_UPGRADE, 1)") {
		t.Fatal("non-participant P2 is still consumed")
	}
	if !strings.Contains(module, "for (p = 1; <= A2K_PLAYERS)") {
		t.Fatal("object discovery no longer covers all scale-test players")
	}
}

func TestBuildReportsActiveStorePlayers(t *testing.T) {
	report, err := Build(Options{OutputDir: t.TempDir(), Timestamp: 1780000000, ActivePlayers: []int{1}})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.ActivePlayers) != 1 || report.ActivePlayers[0] != 1 {
		t.Fatalf("active players=%v, want [1]", report.ActivePlayers)
	}
	xsSource, err := os.ReadFile(report.XSPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(xsSource), "CustomStore_Consume(2, A2K_TOKEN_UPGRADE, 1)") {
		t.Fatal("isolated build still scans P2")
	}
}

func TestBuildWritesStructureVerifiedFixture(t *testing.T) {
	report, err := Build(Options{OutputDir: t.TempDir(), Timestamp: 1780000000})
	if err != nil {
		t.Fatal(err)
	}
	if report.InstanceCount != 440 || report.PlayerCount != 8 || report.TriggerCount != TriggerCount {
		t.Fatalf("report=%+v", report)
	}
	for _, path := range []string{report.ScenarioPath, report.BaseScenarioPath, report.RecipePath, report.XSPath, report.TypesPath} {
		if _, err := os.Stat(path); err != nil {
			t.Fatal(err)
		}
	}
	scen, err := scenario.Open(report.ScenarioPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := scen.VerifyRebuild(); err != nil {
		t.Fatal(err)
	}
	if scen.Units.Total != 448 {
		t.Fatalf("unit total=%d want 448 including 8 barracks starters", scen.Units.Total)
	}
	if scen.Triggers.Count != TriggerCount {
		t.Fatalf("trigger total=%d want %d", scen.Triggers.Count, TriggerCount)
	}
	if got := scen.Settings().Players[0].StartingAge; got != 6 {
		t.Fatalf("starting age=%d want post-imperial 6", got)
	}
	settings := scen.Settings()
	if len(settings.Resources) < DefaultPlayerCount+1 {
		t.Fatalf("resource rows=%d want at least %d including Gaia", len(settings.Resources), DefaultPlayerCount+1)
	}
	for player := 0; player < DefaultPlayerCount; player++ {
		resources := settings.Resources[player]
		if resources.Player != player+1 {
			t.Fatalf("resource row %d identifies slot %d", player, resources.Player)
		}
		for name, value := range map[string]int{
			"food": resources.Food, "wood": resources.Wood, "gold": resources.Gold,
			"stone": resources.Stone, "trade_goods": resources.TradeGoods,
		} {
			if value != DefaultStartingResourceAmount {
				t.Fatalf("P%d %s=%d want %d", player, name, value, DefaultStartingResourceAmount)
			}
		}
	}
	types, err := os.ReadFile(report.TypesPath)
	if err != nil || !strings.Contains(string(types), "purchase = string,int,int,int") || !strings.Contains(string(types), "footer_integrity = 424242") {
		t.Fatalf("sidecar types missing protocol: %v", err)
	}
}
