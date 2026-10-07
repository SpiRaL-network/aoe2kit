package rpgshop

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"aoe2kit/pkg/scenario"
)

const (
	DemoScenarioName = "A2K RPG Shop Demo"
	xsFileName       = "A2KRPG_SHOP_ENGINE.xs"

	unitBarracks = 12
	unitHouse    = 70
	unitKnight   = 38
	unitArcher   = 4
	unitMilitia  = 74
	unitSpearman = 93

	scopeParty  = "party"
	scopePlayer = "player"
	scopeHero   = "hero"

	itemHP  = "hero_hp"
	itemATK = "hero_attack"

	resourceFood   = 0
	resourceWood   = 1
	resourceGold   = 3
	resourcePopCap = 4
	resourceKills  = 20

	opSet          = 1
	opAdd          = 2
	opAttributeSet = 1

	cmpGE = 4
)

const (
	varPartyLevel = 100
	varPartyXP    = 101
	varPartyCoin  = 102
	varP1Level    = 110
	varP1XP       = 111
	varP1Coin     = 112
	varP2Level    = 120
	varP2XP       = 121
	varP2Coin     = 122
	varP1H1Level  = 130
	varP1H1XP     = 131
	varP1H1Coin   = 132
	varP1H2Level  = 140
	varP1H2XP     = 141
	varP1H2Coin   = 142
	varP2H1Level  = 150
	varP2H1XP     = 151
	varP2H1Coin   = 152
	varP2H2Level  = 160
	varP2H2XP     = 161
	varP2H2Coin   = 162
	varLastScope  = 170
	varLastPlayer = 171
	varLastHero   = 172
	varLastItem   = 173
	varLastCost   = 174
	varLastResult = 175
)

type DemoOptions struct {
	OutputDir    string
	ScenarioName string
	Timestamp    int
}

type DemoReport struct {
	OutputDir          string               `json:"output_dir"`
	ScenarioPath       string               `json:"scenario_path"`
	BaseScenarioPath   string               `json:"base_scenario_path"`
	RecipePath         string               `json:"recipe_path"`
	XSPath             string               `json:"xs_path"`
	InstructionsPath   string               `json:"instructions_path"`
	ManifestPath       string               `json:"manifest_path"`
	ScenarioSHA256     string               `json:"scenario_sha256"`
	XSSHA256           string               `json:"xs_sha256"`
	BlankReport        scenario.BlankReport `json:"blank_report"`
	PatchReport        scenario.PatchReport `json:"patch_report"`
	Variables          []DemoVariable       `json:"variables"`
	ShopItems          []DemoShopItem       `json:"shop_items"`
	Earning            DemoEarning          `json:"earning"`
	VerificationClaims []string             `json:"verification_claims"`
	KnownGaps          []string             `json:"known_gaps"`
	SemanticLint       scenario.LintReport  `json:"semantic_lint"`
	Files              map[string]string    `json:"files"`
}

type DemoVariable struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Scope string `json:"scope"`
}

type DemoShopItem struct {
	Scope      string `json:"scope"`
	Item       string `json:"item"`
	Player     int    `json:"player"`
	Hero       int    `json:"hero"`
	TokenUnit  int    `json:"token_unit"`
	HeroUnit   int    `json:"hero_unit"`
	Cost       int    `json:"cost"`
	UnlockNote string `json:"unlock_note"`
}

type DemoEarning struct {
	Signal           string `json:"signal"`
	Players          []int  `json:"players"`
	EnemyPlayers     []int  `json:"enemy_players"`
	GoldPerKill      int    `json:"gold_per_kill"`
	XPPerKill        int    `json:"xp_per_kill"`
	LevelCurve       string `json:"level_curve"`
	HeroAttribution  string `json:"hero_attribution"`
	PollTriggerName  string `json:"poll_trigger_name"`
	PollIntervalNote string `json:"poll_interval_note"`
}

func BuildHeroShopDemo(opts DemoOptions) (*DemoReport, error) {
	opts = normalizeDemoOptions(opts)
	if err := os.MkdirAll(opts.OutputDir, 0755); err != nil {
		return nil, err
	}

	xsPath := filepath.Join(opts.OutputDir, xsFileName)
	recipePath := filepath.Join(opts.OutputDir, "A2K_RPG_SHOP_DEMO_RECIPE.json")
	instructionsPath := filepath.Join(opts.OutputDir, "A2K_RPG_SHOP_DEMO_README.txt")
	manifestPath := filepath.Join(opts.OutputDir, "A2K_RPG_SHOP_DEMO_MANIFEST.json")
	basePath := filepath.Join(opts.OutputDir, "A2K_RPG_SHOP_DEMO_BLANK_BASE.aoe2scenario")
	scenarioPath := filepath.Join(opts.OutputDir, opts.ScenarioName+".aoe2scenario")

	xs := XSModule()
	if err := os.WriteFile(xsPath, []byte(xs), 0644); err != nil {
		return nil, err
	}
	blank, err := scenario.WriteBlankScenarioFile(basePath, scenario.BlankOptions{
		PlayerCount:   3,
		HumanSlots:    1,
		MapWidth:      120,
		MapHeight:     120,
		Timestamp:     opts.Timestamp,
		DummyStarters: true,
		DummyUnit:     unitBarracks,
		DummyStartX:   5.5,
		DummyStartY:   5.5,
		DummySpacing:  8,
	})
	if err != nil {
		return nil, err
	}
	recipe := DemoRecipe(xsPath, opts.Timestamp)
	if err := writeJSONFile(recipePath, recipe); err != nil {
		return nil, err
	}
	patch, err := scenario.PatchRecipeFile(basePath, scenarioPath, recipe)
	if err != nil {
		return nil, err
	}
	semanticLint, err := scenario.LintFileWithOptions(scenarioPath, scenario.LintOptions{SemanticInvariants: true, Generated: true})
	if err != nil {
		return nil, fmt.Errorf("semantic lint generated RPG shop: %w", err)
	}
	if !semanticLint.OK {
		return nil, fmt.Errorf("generated RPG shop failed semantic lint: %s", semanticLintSummary(semanticLint))
	}
	scenSHA, err := fileSHA256(scenarioPath)
	if err != nil {
		return nil, err
	}
	xsSHA, err := fileSHA256(xsPath)
	if err != nil {
		return nil, err
	}

	report := &DemoReport{
		OutputDir:        opts.OutputDir,
		ScenarioPath:     scenarioPath,
		BaseScenarioPath: basePath,
		RecipePath:       recipePath,
		XSPath:           xsPath,
		InstructionsPath: instructionsPath,
		ManifestPath:     manifestPath,
		ScenarioSHA256:   scenSHA,
		XSSHA256:         xsSHA,
		BlankReport:      blank,
		PatchReport:      patch,
		Variables:        demoVariables(),
		ShopItems:        demoShopItems(),
		Earning:          demoEarning(),
		VerificationClaims: []string{
			"structure_verified: generated from a fresh Kit blank scenario; patched file reparses and scenario trigger invariants pass",
			"source_embedded: XS source is stored in an enabled trigger-0 script_call source carrier titled XS string; scenario script filename/path fields are intentionally empty",
			"engine_unverified: renamed train-menu buttons, XS transactions, directed kill-counter polling, and live objective substitution still require a DE run",
		},
		KnownGaps: []string{
			"earning polls xsPlayerAttribute(player, 300+victim_player), the directed engine kill ledger decoded from SpiRaL constants; this exact shop demo still needs engine verification",
			"hero XP and hero coins are credited to both heroes owned by the credited player because the directed kill counter is per-player, not per-hero",
			"shop cost wiring uses change_object_cost plus XS attribute writes; starting resources remain generous as a fallback while this engine behavior is being verified",
			"objective variables are intentionally kept under 256 because current engine evidence shows live <Variable N> substitution works in that range",
			"XS exposes party/player/hero XP scopes and the demo demonstrates hero-scope purchases first",
			"hero upgrades are applied by unit type and player; production RPGs should use dedicated hero unit ids or selected-object targeting if a player can own duplicate hero base units",
		},
		SemanticLint: semanticLint,
		Files: map[string]string{
			"scenario":     scenarioPath,
			"blank_base":   basePath,
			"recipe":       recipePath,
			"xs":           xsPath,
			"instructions": instructionsPath,
			"manifest":     manifestPath,
		},
	}
	if err := os.WriteFile(instructionsPath, []byte(demoInstructions(report)), 0644); err != nil {
		return nil, err
	}
	if err := writeJSONFile(manifestPath, report); err != nil {
		return nil, err
	}
	return report, nil
}

func semanticLintSummary(report scenario.LintReport) string {
	if len(report.Issues) == 0 {
		return "unknown error"
	}
	parts := make([]string, 0, len(report.Issues))
	for _, issue := range report.Issues {
		if issue.Severity == "error" {
			parts = append(parts, issue.Code+": "+issue.Message)
		}
	}
	if len(parts) == 0 {
		return report.Issues[0].Code + ": " + report.Issues[0].Message
	}
	return fmt.Sprintf("%v", parts)
}

func demoInstructions(report *DemoReport) string {
	return fmt.Sprintf(`A2K RPG Shop Demo

Purpose
This is a structure-verified, engine-unverified RPG shop fixture generated by AoE2Kit from a fresh blank scenario. It demonstrates trigger-variable currency ledgers, three-tier RPG XP ledgers, an XS transaction brain, objective-panel live variable substitution, directed kill-counter earning, and renamed train-to-buy shop tokens.

Install
Copy the .aoe2scenario into the active scenario folder. The runtime XS source is embedded in trigger 0; no external .xs file should be required.

Run
1. Start the scenario as Player 1.
2. At 5 seconds, P1 Hero 1 starts with 5 hero coins.
3. At 12 seconds, red dummy units spawn near the P1 and P2 heroes.
4. Kill red dummies. The XS earn loop polls xsPlayerAttribute(player, 300+victim_player), grants 1 native gold and 25 XP per kill, then syncs party/player/hero ledgers back to the Objectives panel.
5. Select the P1 hero shop Barracks and train the renamed token items. +HP costs 3 hero coins; +Attack costs 5 hero coins.
6. Watch the Objectives panel for live variable substitution.

Important Claims
- Structural verification passed: %t
- Trigger count: %d
- Unit count: %d
- Scenario SHA256: %s
- XS SHA256: %s

Honesty
The file is structurally verified by AoE2Kit. It is not engine-verified until it loads and runs in AoE2DE.
`, report.PatchReport.RebuildOK && report.PatchReport.InvariantOK, report.PatchReport.TriggerCountAfter, report.PatchReport.UnitCountAfter, report.ScenarioSHA256, report.XSSHA256)
}

func DemoRecipe(xsPath string, timestamp int) scenario.Recipe {
	active := true
	inactive := false
	human := true
	computer := false
	lock := true
	playerCount := 3
	ally := 0
	neutral := 1
	enemy := 3
	conquestRequired := 0
	allCustomRequired := 1
	requiredScore := 0
	timedGame := 0
	mute := true
	show := true
	noLoop := false
	loop := true
	source := 1
	pollTimer := 1
	longTimer := 999999

	triggers := []scenario.TriggerRecipe{
		{Op: "clear_triggers"},
		objectiveHeader(0, "A2K RPG Shop Demo"),
		objectiveLine(1, fmt.Sprintf("Party L %s  XP %s  Coins %s", objectiveVar(varPartyLevel), objectiveVar(varPartyXP), objectiveVar(varPartyCoin))),
		objectiveLine(2, fmt.Sprintf("P1 Knight L %s  XP %s  Coins %s", objectiveVar(varP1H1Level), objectiveVar(varP1H1XP), objectiveVar(varP1H1Coin))),
		objectiveLine(3, fmt.Sprintf("P1 Archer L %s  XP %s  Coins %s", objectiveVar(varP1H2Level), objectiveVar(varP1H2XP), objectiveVar(varP1H2Coin))),
		objectiveLine(4, fmt.Sprintf("P2 Knight L %s  XP %s  Coins %s", objectiveVar(varP2H1Level), objectiveVar(varP2H1XP), objectiveVar(varP2H1Coin))),
		objectiveLine(5, fmt.Sprintf("P2 Archer L %s  XP %s  Coins %s", objectiveVar(varP2H2Level), objectiveVar(varP2H2XP), objectiveVar(varP2H2Coin))),
		timedTrigger(2, "A2KRPG 002 Boot", []scenario.EffectRecipe{
			view(18, 18),
			displayMsg("A2K RPG shop demo. Kill red dummies for hero coins, then train instant shop tokens at the Barracks."),
			setVar(varPartyLevel, 1),
			setVar(varP1Level, 1),
			setVar(varP2Level, 1),
			setVar(varP1H1Level, 1),
			setVar(varP1H2Level, 1),
			setVar(varP2H1Level, 1),
			setVar(varP2H2Level, 1),
			enableObject(1, unitMilitia, 1),
			enableObject(1, unitSpearman, 1),
			enableObject(2, unitMilitia, 1),
			enableObject(2, unitSpearman, 1),
			addTrain(1, unitMilitia, unitBarracks, 1, 0, 65),
			addTrain(1, unitSpearman, unitBarracks, 2, 0, 83),
			addTrain(2, unitMilitia, unitBarracks, 1, 0, 65),
			addTrain(2, unitSpearman, unitBarracks, 2, 0, 83),
			renameTokenName(1, unitMilitia, "+HP"),
			renameTokenDescription(1, unitMilitia, "+60 HP to your units (3 gold)"),
			renameTokenName(1, unitSpearman, "+Attack"),
			renameTokenDescription(1, unitSpearman, "+4 Attack to your units (5 gold)"),
			renameTokenName(2, unitMilitia, "+HP"),
			renameTokenDescription(2, unitMilitia, "+60 HP to your units (3 gold)"),
			renameTokenName(2, unitSpearman, "+Attack"),
			renameTokenDescription(2, unitSpearman, "+4 Attack to your units (5 gold)"),
			shopCost(1, unitMilitia, 3),
			shopCost(1, unitSpearman, 5),
			shopCost(2, unitMilitia, 3),
			shopCost(2, unitSpearman, 5),
			setTokenAttr(1, unitMilitia, 103, 0),
			setTokenAttr(1, unitMilitia, 104, 0),
			setTokenAttr(1, unitMilitia, 105, 3),
			setTokenAttr(1, unitMilitia, 106, 0),
			setTokenAttr(1, unitMilitia, 101, 0),
			setTokenAttr(1, unitMilitia, 110, 0),
			setTokenAttr(1, unitSpearman, 103, 0),
			setTokenAttr(1, unitSpearman, 104, 0),
			setTokenAttr(1, unitSpearman, 105, 5),
			setTokenAttr(1, unitSpearman, 106, 0),
			setTokenAttr(1, unitSpearman, 101, 0),
			setTokenAttr(1, unitSpearman, 110, 0),
			setTokenAttr(2, unitMilitia, 103, 0),
			setTokenAttr(2, unitMilitia, 104, 0),
			setTokenAttr(2, unitMilitia, 105, 3),
			setTokenAttr(2, unitMilitia, 106, 0),
			setTokenAttr(2, unitMilitia, 101, 0),
			setTokenAttr(2, unitMilitia, 110, 0),
			setTokenAttr(2, unitSpearman, 103, 0),
			setTokenAttr(2, unitSpearman, 104, 0),
			setTokenAttr(2, unitSpearman, 105, 5),
			setTokenAttr(2, unitSpearman, 106, 0),
			setTokenAttr(2, unitSpearman, 101, 0),
			setTokenAttr(2, unitSpearman, 110, 0),
			addResource(1, resourcePopCap, 20),
			addResource(2, resourcePopCap, 20),
			call("A2KRPG_Boot"),
		}),
		timedTrigger(5, "A2KRPG 005 Seed Coins", []scenario.EffectRecipe{
			addVar(varPartyCoin, 5),
			addVar(varP1Coin, 5),
			addVar(varP1H1Coin, 5),
			addResource(1, resourceGold, 10),
			call("A2KRPG_SyncFromTriggers"),
			displayMsg("P1 Knight seeded with 5 coins so the shop can be tested even before kills."),
		}),
		timedTrigger(12, "A2KRPG 012 Spawn Directed Kill Wave", dummyWaveEffects()),
		{
			Op:      "add_trigger",
			Name:    "A2KRPG 020 Poll Directed Kill XP",
			Enabled: &active,
			Looping: &loop,
			Conditions: []scenario.ConditionRecipe{
				{Op: "timer", Timer: &pollTimer},
			},
			Effects: []scenario.EffectRecipe{
				call("A2KRPG_PollKills"),
			},
		},
		purchaseTrigger(1, 1, itemHP, unitMilitia, varP1H1Coin, 3, "A2KRPG_P1H1BuyHP"),
		purchaseTrigger(1, 1, itemATK, unitSpearman, varP1H1Coin, 5, "A2KRPG_P1H1BuyATK"),
		purchaseTrigger(2, 1, itemHP, unitMilitia, varP2H1Coin, 3, "A2KRPG_P2H1BuyHP"),
		purchaseTrigger(2, 1, itemATK, unitSpearman, varP2H1Coin, 5, "A2KRPG_P2H1BuyATK"),
		{
			Op:      "add_trigger",
			Name:    "A2KRPG 900 Keep Objective HUD Alive",
			Enabled: &active,
			Looping: &loop,
			Conditions: []scenario.ConditionRecipe{
				{Op: "timer", Timer: &longTimer},
			},
			Effects: []scenario.EffectRecipe{},
		},
		timedTrigger(180, "A2KRPG 999 Declare Victory", []scenario.EffectRecipe{
			{Op: "declare_victory", SourcePlayer: &source, Enabled: intPtr(1)},
		}),
	}
	for i := range triggers {
		if triggers[i].Enabled == nil {
			triggers[i].Enabled = &active
		}
		if triggers[i].Looping == nil {
			triggers[i].Looping = &noLoop
		}
		_ = mute
		_ = show
	}

	return scenario.Recipe{
		Scenario: &scenario.ScenarioRecipe{PlayerCount: &playerCount, TimestampOfLastSave: &timestamp},
		XS: &scenario.XSRecipe{
			Mode:               "inline_runtime",
			Name:               xsFileName,
			ContentFile:        xsPath,
			CarrierTitle:       "XS string",
			CarrierTriggerName: "A2KRPG XS Runtime Source",
		},
		Victory: &scenario.VictoryRecipe{
			ConquestRequired:               &conquestRequired,
			AllCustomConditionsRequired:    &allCustomRequired,
			RequiredScoreForScoreVictory:   &requiredScore,
			TimeForTimedGameIn10thsOfAYear: &timedGame,
		},
		Players: []scenario.PlayerRecipe{
			{Player: 1, Active: &active, Human: &human, TribeName: strPtr("Hero Tester"), LockCivilization: &lock},
			{Player: 2, Active: &active, Human: &computer, TribeName: strPtr("Ally Hero Lane"), LockCivilization: &lock},
			{Player: 3, Active: &active, Human: &computer, TribeName: strPtr("Dummy Enemies"), LockCivilization: &lock},
			{Player: 4, Active: &inactive, Human: &computer},
			{Player: 5, Active: &inactive, Human: &computer},
			{Player: 6, Active: &inactive, Human: &computer},
			{Player: 7, Active: &inactive, Human: &computer},
			{Player: 8, Active: &inactive, Human: &computer},
		},
		Diplomacy: []scenario.DiplomacyRecipe{
			{From: 1, To: 2, Stance: neutral}, {From: 2, To: 1, Stance: neutral},
			{From: 1, To: 3, Stance: neutral}, {From: 3, To: 1, Stance: neutral},
			{From: 1, To: 4, Stance: neutral}, {From: 4, To: 1, Stance: neutral},
			{From: 2, To: 3, Stance: ally}, {From: 3, To: 2, Stance: ally},
			{From: 2, To: 4, Stance: enemy}, {From: 4, To: 2, Stance: enemy},
			{From: 3, To: 4, Stance: enemy}, {From: 4, To: 3, Stance: enemy},
		},
		DiplomacyOptions: &scenario.DiplomacyOptionsRecipe{
			LockTeams:               boolPtr(true),
			AllowPlayersChooseTeams: boolPtr(false),
			RandomStartPoints:       boolPtr(false),
			MaxNumberOfTeams:        intPtr(3),
			AlliedVictory: []scenario.AlliedVictoryRecipe{
				{Player: 1, Enabled: true},
				{Player: 2, Enabled: true},
			},
		},
		Resources: []scenario.ResourceRecipe{
			{Player: 1, Food: intPtr(200), Wood: intPtr(200), Gold: intPtr(20), Stone: intPtr(0)},
			{Player: 2, Food: intPtr(200), Wood: intPtr(200), Gold: intPtr(20), Stone: intPtr(0)},
			{Player: 3, Food: intPtr(1), Wood: intPtr(0), Gold: intPtr(0), Stone: intPtr(0)},
		},
		Variables: variableRecipes(),
		Triggers:  triggers,
		Units: []scenario.UnitRecipe{
			unit(1, unitBarracks, 18.5, 18.5, 910101, "P1 HERO SHOP: train Militia=+HP, Spearman=+Attack."),
			unit(1, unitHouse, 16.5, 18.5, 910102, "P1 shop pop headroom."),
			unit(1, unitHouse, 16.5, 20.5, 910103, "P1 shop pop headroom."),
			unit(1, unitHouse, 16.5, 22.5, 910104, "P1 shop pop headroom."),
			unit(2, unitBarracks, 30.5, 18.5, 910201, "P2 HERO SHOP: same wiring for ally lane."),
			unit(2, unitHouse, 28.5, 18.5, 910202, "P2 shop pop headroom."),
			unit(2, unitHouse, 28.5, 20.5, 910203, "P2 shop pop headroom."),
			unit(2, unitHouse, 28.5, 22.5, 910204, "P2 shop pop headroom."),
			unit(1, unitKnight, 18.5, 22.5, 910111, "P1 Hero 1 Knight. Kill red dummies for coins."),
			unit(1, unitArcher, 20.5, 22.5, 910112, "P1 Hero 2 Archer. Scope plumbing present."),
			unit(2, unitKnight, 30.5, 22.5, 910211, "P2 Hero 1 Knight. Ally clone lane."),
			unit(2, unitArcher, 32.5, 22.5, 910212, "P2 Hero 2 Archer. Scope plumbing present."),
			unit(3, unitBarracks, 45.5, 18.5, 910301, "P3 dummy barracks. Keeps defeat detection alive from scenario start."),
		},
		Map: []scenario.MapRecipe{
			{Op: "set_terrain_rect", X1: 10, Y1: 10, X2: 52, Y2: 30, TerrainID: intPtr(0), Layer: intPtr(-1)},
		},
	}
}

func XSModule() string {
	return `// A2KRPG_SHOP_ENGINE.xs
// Generated by AoE2Kit. Project-neutral RPG shop core: trigger-owned transactions,
// XS-owned ledgers, stock wiring, and hero/player/party state helpers.

const int A2KRPG_SCOPE_PARTY = 1;
const int A2KRPG_SCOPE_PLAYER = 2;
const int A2KRPG_SCOPE_HERO = 3;

const int A2KRPG_ITEM_HP = 1;
const int A2KRPG_ITEM_ATTACK = 2;

const int A2KRPG_UNIT_SHOP = 12;
const int A2KRPG_UNIT_KNIGHT = 38;
const int A2KRPG_UNIT_ARCHER = 4;
const int A2KRPG_TOKEN_HP = 74;
const int A2KRPG_TOKEN_ATTACK = 93;

const int A2KRPG_ATTR_HITPOINTS = 0;
const int A2KRPG_ATTR_ATTACK = 9;
const int A2KRPG_ATTR_TRAIN_LOC = 42;
const int A2KRPG_ATTR_TRAIN_BUTTON = 43;
const int A2KRPG_ATTR_HOTKEY = 58;
const int A2KRPG_ATTR_TRAIN_TIME = 101;
const int A2KRPG_ATTR_FOOD_COST = 103;
const int A2KRPG_ATTR_WOOD_COST = 104;
const int A2KRPG_ATTR_GOLD_COST = 105;
const int A2KRPG_ATTR_STONE_COST = 106;
const int A2KRPG_ATTR_PLAYER_KILLS_BASE = 300;
const int A2KRPG_RES_GOLD = 3;

const int A2KRPG_NUM_PLAYERS = 2;
const int A2KRPG_HEROES_PER_PLAYER = 2;
const int A2KRPG_NUM_HEROES = 4;
const int A2KRPG_XP_PER_KILL = 25;
const int A2KRPG_GOLD_PER_KILL = 1;

const int VAR_PARTY_LEVEL = 100;
const int VAR_PARTY_XP = 101;
const int VAR_PARTY_COINS = 102;
const int VAR_P1_LEVEL = 110;
const int VAR_P1_XP = 111;
const int VAR_P1_COINS = 112;
const int VAR_P2_LEVEL = 120;
const int VAR_P2_XP = 121;
const int VAR_P2_COINS = 122;
const int VAR_P1_H1_LEVEL = 130;
const int VAR_P1_H1_XP = 131;
const int VAR_P1_H1_COINS = 132;
const int VAR_P1_H2_LEVEL = 140;
const int VAR_P1_H2_XP = 141;
const int VAR_P1_H2_COINS = 142;
const int VAR_P2_H1_LEVEL = 150;
const int VAR_P2_H1_XP = 151;
const int VAR_P2_H1_COINS = 152;
const int VAR_P2_H2_LEVEL = 160;
const int VAR_P2_H2_XP = 161;
const int VAR_P2_H2_COINS = 162;
const int VAR_LAST_SCOPE = 170;
const int VAR_LAST_PLAYER = 171;
const int VAR_LAST_HERO = 172;
const int VAR_LAST_ITEM = 173;
const int VAR_LAST_COST = 174;
const int VAR_LAST_RESULT = 175;

int gA2KRPGInitialized = 0;
int gHeroXP = -1;
int gHeroLevel = -1;
int gHeroLine = -1;
int gHeroUnit = -1;
int gPlayerXP = -1;
int gPlayerLevel = -1;
int gKillLast = -1;
int gPartyXP = 0;
int gPartyLevel = 1;

void A2KRPG_setVar(int id=0, int value=0) {
    xsSetTriggerVariable(id, value);
}

int A2KRPG_var(int id=0) {
    return(xsTriggerVariable(id));
}

int A2KRPG_heroLevelVar(int player=1, int hero=1) {
    if (player == 1) { if (hero == 1) { return(VAR_P1_H1_LEVEL); } return(VAR_P1_H2_LEVEL); }
    if (hero == 1) { return(VAR_P2_H1_LEVEL); }
    return(VAR_P2_H2_LEVEL);
}

int A2KRPG_heroXPVar(int player=1, int hero=1) {
    if (player == 1) { if (hero == 1) { return(VAR_P1_H1_XP); } return(VAR_P1_H2_XP); }
    if (hero == 1) { return(VAR_P2_H1_XP); }
    return(VAR_P2_H2_XP);
}

int A2KRPG_heroCoinVar(int player=1, int hero=1) {
    if (player == 1) { if (hero == 1) { return(VAR_P1_H1_COINS); } return(VAR_P1_H2_COINS); }
    if (hero == 1) { return(VAR_P2_H1_COINS); }
    return(VAR_P2_H2_COINS);
}

int A2KRPG_playerCoinVar(int player=1) {
    if (player == 1) { return(VAR_P1_COINS); }
    return(VAR_P2_COINS);
}

int A2KRPG_playerXPVar(int player=1) {
    if (player == 1) { return(VAR_P1_XP); }
    return(VAR_P2_XP);
}

int A2KRPG_playerLevelVar(int player=1) {
    if (player == 1) { return(VAR_P1_LEVEL); }
    return(VAR_P2_LEVEL);
}

int A2KRPG_heroIndex(int player=1, int heroSlot=0) {
    return((player - 1) * A2KRPG_HEROES_PER_PLAYER + heroSlot);
}

int A2KRPG_killIndex(int player=1, int victim=3) {
    return((player - 1) * 9 + victim);
}

int A2KRPG_heroUnit(int player=1, int hero=1) {
    if (hero == 1) { return(A2KRPG_UNIT_KNIGHT); }
    return(A2KRPG_UNIT_ARCHER);
}

void A2KRPG_InitLedgers() {
    if (gA2KRPGInitialized == 1) { return; }
    gHeroXP = xsArrayCreateInt(A2KRPG_NUM_HEROES, 0, "a2krpg hero xp");
    gHeroLevel = xsArrayCreateInt(A2KRPG_NUM_HEROES, 1, "a2krpg hero level");
    gHeroLine = xsArrayCreateInt(A2KRPG_NUM_HEROES, 0, "a2krpg hero line");
    gHeroUnit = xsArrayCreateInt(A2KRPG_NUM_HEROES, -1, "a2krpg hero unit");
    gPlayerXP = xsArrayCreateInt(A2KRPG_NUM_PLAYERS + 1, 0, "a2krpg player xp");
    gPlayerLevel = xsArrayCreateInt(A2KRPG_NUM_PLAYERS + 1, 1, "a2krpg player level");
    gKillLast = xsArrayCreateInt(27, 0, "a2krpg last directed kills");
    gPartyXP = 0;
    gPartyLevel = 1;
    gA2KRPGInitialized = 1;
}

int GetHeroXP(int hi=0) { A2KRPG_InitLedgers(); return(xsArrayGetInt(gHeroXP, hi)); }
int GetHeroLevel(int hi=0) { A2KRPG_InitLedgers(); return(xsArrayGetInt(gHeroLevel, hi)); }
int GetHeroLine(int hi=0) { A2KRPG_InitLedgers(); return(xsArrayGetInt(gHeroLine, hi)); }
int GetHeroUnit(int hi=0) { A2KRPG_InitLedgers(); return(xsArrayGetInt(gHeroUnit, hi)); }
int GetPlayerXP(int p=1) { A2KRPG_InitLedgers(); return(xsArrayGetInt(gPlayerXP, p)); }
int GetPlayerLevel(int p=1) { A2KRPG_InitLedgers(); return(xsArrayGetInt(gPlayerLevel, p)); }
int GetPartyXP() { A2KRPG_InitLedgers(); return(gPartyXP); }
int GetPartyLevel() { A2KRPG_InitLedgers(); return(gPartyLevel); }

void A2KRPG_SyncXPVars() {
    A2KRPG_setVar(VAR_PARTY_XP, gPartyXP);
    A2KRPG_setVar(VAR_PARTY_LEVEL, gPartyLevel);
    A2KRPG_setVar(VAR_P1_XP, xsArrayGetInt(gPlayerXP, 1));
    A2KRPG_setVar(VAR_P1_LEVEL, xsArrayGetInt(gPlayerLevel, 1));
    A2KRPG_setVar(VAR_P2_XP, xsArrayGetInt(gPlayerXP, 2));
    A2KRPG_setVar(VAR_P2_LEVEL, xsArrayGetInt(gPlayerLevel, 2));
    A2KRPG_setVar(VAR_P1_H1_XP, xsArrayGetInt(gHeroXP, A2KRPG_heroIndex(1, 0)));
    A2KRPG_setVar(VAR_P1_H1_LEVEL, xsArrayGetInt(gHeroLevel, A2KRPG_heroIndex(1, 0)));
    A2KRPG_setVar(VAR_P1_H2_XP, xsArrayGetInt(gHeroXP, A2KRPG_heroIndex(1, 1)));
    A2KRPG_setVar(VAR_P1_H2_LEVEL, xsArrayGetInt(gHeroLevel, A2KRPG_heroIndex(1, 1)));
    A2KRPG_setVar(VAR_P2_H1_XP, xsArrayGetInt(gHeroXP, A2KRPG_heroIndex(2, 0)));
    A2KRPG_setVar(VAR_P2_H1_LEVEL, xsArrayGetInt(gHeroLevel, A2KRPG_heroIndex(2, 0)));
    A2KRPG_setVar(VAR_P2_H2_XP, xsArrayGetInt(gHeroXP, A2KRPG_heroIndex(2, 1)));
    A2KRPG_setVar(VAR_P2_H2_LEVEL, xsArrayGetInt(gHeroLevel, A2KRPG_heroIndex(2, 1)));
}

void A2KRPG_LevelHero(int hi=0) {
    int xp = xsArrayGetInt(gHeroXP, hi);
    int level = xsArrayGetInt(gHeroLevel, hi);
    int threshold = level * 100;
    while (xp >= threshold) {
        xp = xp - threshold;
        level = level + 1;
        threshold = level * 100;
    }
    xsArraySetInt(gHeroXP, hi, xp);
    xsArraySetInt(gHeroLevel, hi, level);
}

void A2KRPG_LevelPlayer(int player=1) {
    int xp = xsArrayGetInt(gPlayerXP, player);
    int level = xsArrayGetInt(gPlayerLevel, player);
    int threshold = level * 100;
    while (xp >= threshold) {
        xp = xp - threshold;
        level = level + 1;
        threshold = level * 100;
    }
    xsArraySetInt(gPlayerXP, player, xp);
    xsArraySetInt(gPlayerLevel, player, level);
}

void A2KRPG_LevelParty() {
    int threshold = gPartyLevel * 100;
    while (gPartyXP >= threshold) {
        gPartyXP = gPartyXP - threshold;
        gPartyLevel = gPartyLevel + 1;
        threshold = gPartyLevel * 100;
    }
}

void AddHeroXP(int hi=0, int amt=0) {
    A2KRPG_InitLedgers();
    xsArraySetInt(gHeroXP, hi, xsArrayGetInt(gHeroXP, hi) + amt);
    A2KRPG_LevelHero(hi);
}

void AddPlayerXP(int p=1, int amt=0) {
    A2KRPG_InitLedgers();
    xsArraySetInt(gPlayerXP, p, xsArrayGetInt(gPlayerXP, p) + amt);
    A2KRPG_LevelPlayer(p);
}

void AddPartyXP(int amt=0) {
    A2KRPG_InitLedgers();
    gPartyXP = gPartyXP + amt;
    A2KRPG_LevelParty();
}

void SetHeroLine(int hi=0, int line=0) { A2KRPG_InitLedgers(); xsArraySetInt(gHeroLine, hi, line); }
void SetHeroUnit(int hi=0, int uref=0) { A2KRPG_InitLedgers(); xsArraySetInt(gHeroUnit, hi, uref); }

void A2KRPG_wireItem(int player=1, int token=0, int button=1, int cost=0) {
    xsEffectAmount(cSetAttribute, token, A2KRPG_ATTR_TRAIN_LOC, A2KRPG_UNIT_SHOP, player);
    xsEffectAmount(cSetAttribute, token, A2KRPG_ATTR_TRAIN_BUTTON, button, player);
    xsEffectAmount(cSetAttribute, token, A2KRPG_ATTR_HOTKEY, button, player);
    xsEffectAmount(cSetAttribute, token, A2KRPG_ATTR_TRAIN_TIME, 0, player);
    xsEffectAmount(cSetAttribute, token, A2KRPG_ATTR_FOOD_COST, 0, player);
    xsEffectAmount(cSetAttribute, token, A2KRPG_ATTR_WOOD_COST, 0, player);
    xsEffectAmount(cSetAttribute, token, A2KRPG_ATTR_GOLD_COST, cost, player);
    xsEffectAmount(cSetAttribute, token, A2KRPG_ATTR_STONE_COST, 0, player);
}

void A2KRPG_updateStockForPlayer(int player=1) {
    A2KRPG_wireItem(player, A2KRPG_TOKEN_HP, 1, 3);
    A2KRPG_wireItem(player, A2KRPG_TOKEN_ATTACK, 2, 5);
}

void A2KRPG_SyncFromTriggers() {
    A2KRPG_InitLedgers();
    A2KRPG_SyncXPVars();
    A2KRPG_updateStockForPlayer(1);
    A2KRPG_updateStockForPlayer(2);
}

void A2KRPG_AddCoinsForKill(int player=1) {
    A2KRPG_setVar(VAR_PARTY_COINS, A2KRPG_var(VAR_PARTY_COINS) + A2KRPG_GOLD_PER_KILL);
    A2KRPG_setVar(A2KRPG_playerCoinVar(player), A2KRPG_var(A2KRPG_playerCoinVar(player)) + A2KRPG_GOLD_PER_KILL);
    A2KRPG_setVar(A2KRPG_heroCoinVar(player, 1), A2KRPG_var(A2KRPG_heroCoinVar(player, 1)) + A2KRPG_GOLD_PER_KILL);
    A2KRPG_setVar(A2KRPG_heroCoinVar(player, 2), A2KRPG_var(A2KRPG_heroCoinVar(player, 2)) + A2KRPG_GOLD_PER_KILL);
}

void A2KRPG_GrantKill(int player=1) {
    A2KRPG_InitLedgers();
    float gold = A2KRPG_GOLD_PER_KILL;
    xsSetPlayerAttribute(player, A2KRPG_RES_GOLD, xsPlayerAttribute(player, A2KRPG_RES_GOLD) + gold);
    A2KRPG_AddCoinsForKill(player);
    AddPartyXP(A2KRPG_XP_PER_KILL);
    AddPlayerXP(player, A2KRPG_XP_PER_KILL);
    AddHeroXP(A2KRPG_heroIndex(player, 0), A2KRPG_XP_PER_KILL);
    AddHeroXP(A2KRPG_heroIndex(player, 1), A2KRPG_XP_PER_KILL);
    A2KRPG_SyncFromTriggers();
}

void A2KRPG_PollVictim(int player=1, int victim=3) {
    A2KRPG_InitLedgers();
    int idx = A2KRPG_killIndex(player, victim);
    int last = xsArrayGetInt(gKillLast, idx);
    int killAttribute = A2KRPG_ATTR_PLAYER_KILLS_BASE + victim;
    while (xsPlayerAttribute(player, killAttribute) > last) {
        last = last + 1;
        xsArraySetInt(gKillLast, idx, last);
        A2KRPG_GrantKill(player);
    }
}

void A2KRPG_PollPlayerKills(int player=1) {
    A2KRPG_PollVictim(player, 3);
    A2KRPG_PollVictim(player, 4);
    A2KRPG_PollVictim(player, 5);
    A2KRPG_PollVictim(player, 6);
    A2KRPG_PollVictim(player, 7);
    A2KRPG_PollVictim(player, 8);
}

void A2KRPG_PollKills() {
    A2KRPG_PollPlayerKills(1);
    A2KRPG_PollPlayerKills(2);
}

int A2KRPG_Buy(int scope=3, int player=1, int hero=1, int item=1, int cost=1) {
    int coinVar = A2KRPG_heroCoinVar(player, hero);
    if (scope == A2KRPG_SCOPE_PLAYER) { coinVar = A2KRPG_playerCoinVar(player); }
    if (scope == A2KRPG_SCOPE_PARTY) { coinVar = VAR_PARTY_COINS; }
    int coins = A2KRPG_var(coinVar);
    A2KRPG_setVar(VAR_LAST_SCOPE, scope);
    A2KRPG_setVar(VAR_LAST_PLAYER, player);
    A2KRPG_setVar(VAR_LAST_HERO, hero);
    A2KRPG_setVar(VAR_LAST_ITEM, item);
    A2KRPG_setVar(VAR_LAST_COST, cost);
    if (coins < cost) {
        A2KRPG_setVar(VAR_LAST_RESULT, 0);
        return(0);
    }
    A2KRPG_setVar(coinVar, coins - cost);
    if (item == A2KRPG_ITEM_HP) {
        xsEffectAmount(cAddAttribute, A2KRPG_heroUnit(player, hero), A2KRPG_ATTR_HITPOINTS, 60, player);
    }
    if (item == A2KRPG_ITEM_ATTACK) {
        xsEffectAmount(cAddAttribute, A2KRPG_heroUnit(player, hero), A2KRPG_ATTR_ATTACK, 1028, player);
    }
    A2KRPG_setVar(VAR_LAST_RESULT, 1);
    A2KRPG_SyncFromTriggers();
    return(1);
}

string A2KRPG_PanelLineP1H1() { return("P1 Knight L <Variable 130> XP <Variable 131> Coins <Variable 132>"); }
string A2KRPG_PanelLineP1H2() { return("P1 Archer L <Variable 140> XP <Variable 141> Coins <Variable 142>"); }
string A2KRPG_PanelLineP2H1() { return("P2 Knight L <Variable 150> XP <Variable 151> Coins <Variable 152>"); }
string A2KRPG_PanelLineP2H2() { return("P2 Archer L <Variable 160> XP <Variable 161> Coins <Variable 162>"); }

void A2KRPG_Boot() {
    A2KRPG_InitLedgers();
    A2KRPG_SyncXPVars();
    A2KRPG_SyncFromTriggers();
}

void A2KRPG_P1H1BuyHP() { A2KRPG_Buy(A2KRPG_SCOPE_HERO, 1, 1, A2KRPG_ITEM_HP, 3); }
void A2KRPG_P1H1BuyATK() { A2KRPG_Buy(A2KRPG_SCOPE_HERO, 1, 1, A2KRPG_ITEM_ATTACK, 5); }
void A2KRPG_P2H1BuyHP() { A2KRPG_Buy(A2KRPG_SCOPE_HERO, 2, 1, A2KRPG_ITEM_HP, 3); }
void A2KRPG_P2H1BuyATK() { A2KRPG_Buy(A2KRPG_SCOPE_HERO, 2, 1, A2KRPG_ITEM_ATTACK, 5); }

void main() {
    A2KRPG_Boot();
}
`
}

func normalizeDemoOptions(opts DemoOptions) DemoOptions {
	if opts.OutputDir == "" {
		opts.OutputDir = filepath.Join("build", "rpgshop-demo")
	}
	if opts.ScenarioName == "" {
		opts.ScenarioName = DemoScenarioName
	}
	if opts.Timestamp == 0 {
		opts.Timestamp = int(time.Now().Unix())
	}
	return opts
}

func variableRecipes() []scenario.VariableRecipe {
	vars := demoVariables()
	out := make([]scenario.VariableRecipe, 0, len(vars))
	for _, v := range vars {
		id := v.ID
		out = append(out, scenario.VariableRecipe{Op: "add_variable", ID: &id, Name: v.Name})
	}
	return out
}

func objectiveVar(id int) string {
	return fmt.Sprintf("<Variable %d>", id)
}

func demoVariables() []DemoVariable {
	vars := []DemoVariable{
		{varPartyLevel, "A2K PARTY LEVEL", scopeParty},
		{varPartyXP, "A2K PARTY XP", scopeParty},
		{varPartyCoin, "A2K PARTY COINS", scopeParty},
		{varP1Level, "A2K PLAYER P1 LEVEL", scopePlayer},
		{varP1XP, "A2K PLAYER P1 XP", scopePlayer},
		{varP1Coin, "A2K PLAYER P1 COINS", scopePlayer},
		{varP2Level, "A2K PLAYER P2 LEVEL", scopePlayer},
		{varP2XP, "A2K PLAYER P2 XP", scopePlayer},
		{varP2Coin, "A2K PLAYER P2 COINS", scopePlayer},
		{varP1H1Level, "A2K HERO P1 H1 LEVEL", scopeHero},
		{varP1H1XP, "A2K HERO P1 H1 XP", scopeHero},
		{varP1H1Coin, "A2K HERO P1 H1 COINS", scopeHero},
		{varP1H2Level, "A2K HERO P1 H2 LEVEL", scopeHero},
		{varP1H2XP, "A2K HERO P1 H2 XP", scopeHero},
		{varP1H2Coin, "A2K HERO P1 H2 COINS", scopeHero},
		{varP2H1Level, "A2K HERO P2 H1 LEVEL", scopeHero},
		{varP2H1XP, "A2K HERO P2 H1 XP", scopeHero},
		{varP2H1Coin, "A2K HERO P2 H1 COINS", scopeHero},
		{varP2H2Level, "A2K HERO P2 H2 LEVEL", scopeHero},
		{varP2H2XP, "A2K HERO P2 H2 XP", scopeHero},
		{varP2H2Coin, "A2K HERO P2 H2 COINS", scopeHero},
		{varLastScope, "A2K SHOP LAST SCOPE", "transaction"},
		{varLastPlayer, "A2K SHOP LAST PLAYER", "transaction"},
		{varLastHero, "A2K SHOP LAST HERO", "transaction"},
		{varLastItem, "A2K SHOP LAST ITEM", "transaction"},
		{varLastCost, "A2K SHOP LAST COST", "transaction"},
		{varLastResult, "A2K SHOP LAST RESULT", "transaction"},
	}
	sort.Slice(vars, func(i, j int) bool { return vars[i].ID < vars[j].ID })
	return vars
}

func demoShopItems() []DemoShopItem {
	return []DemoShopItem{
		{Scope: scopeHero, Item: itemHP, Player: 1, Hero: 1, TokenUnit: unitMilitia, HeroUnit: unitKnight, Cost: 3, UnlockNote: "available at boot"},
		{Scope: scopeHero, Item: itemATK, Player: 1, Hero: 1, TokenUnit: unitSpearman, HeroUnit: unitKnight, Cost: 5, UnlockNote: "available at boot"},
		{Scope: scopeHero, Item: itemHP, Player: 2, Hero: 1, TokenUnit: unitMilitia, HeroUnit: unitKnight, Cost: 3, UnlockNote: "available at boot"},
		{Scope: scopeHero, Item: itemATK, Player: 2, Hero: 1, TokenUnit: unitSpearman, HeroUnit: unitKnight, Cost: 5, UnlockNote: "available at boot"},
	}
}

func demoEarning() DemoEarning {
	return DemoEarning{
		Signal:           "xsPlayerAttribute(player, 300+victim_player)",
		Players:          []int{1, 2},
		EnemyPlayers:     []int{3, 4, 5, 6, 7, 8},
		GoldPerKill:      1,
		XPPerKill:        25,
		LevelCurve:       "level N requires N*100 XP; XP is reduced by the crossed threshold",
		HeroAttribution:  "both heroes owned by the credited player receive full kill XP and hero coins because the engine signal is per-player, not per-hero",
		PollTriggerName:  "A2KRPG 020 Poll Directed Kill XP",
		PollIntervalNote: "looping trigger with a 1-second timer calls A2KRPG_PollKills",
	}
}

func objectiveHeader(order int, text string) scenario.TriggerRecipe {
	return objective(order, text, true)
}

func objectiveLine(order int, text string) scenario.TriggerRecipe {
	return objective(order, text, false)
}

func objective(order int, text string, header bool) scenario.TriggerRecipe {
	enabled := true
	noLoop := false
	show := true
	mute := true
	timer := 999999
	return scenario.TriggerRecipe{
		Op:                        "add_trigger",
		Name:                      fmt.Sprintf("A2KRPG Objective %02d", order),
		Description:               text,
		ShortDescription:          text,
		Enabled:                   &enabled,
		Looping:                   &noLoop,
		DisplayAsObjective:        &show,
		DisplayOnScreen:           &show,
		MakeHeader:                &header,
		MuteObjectives:            &mute,
		ObjectiveDescriptionOrder: &order,
		DescriptionStringID:       intPtr(-1),
		ShortDescriptionStringID:  intPtr(-1),
		Conditions: []scenario.ConditionRecipe{
			{Op: "timer", Timer: &timer},
		},
	}
}

func timedTrigger(seconds int, name string, effects []scenario.EffectRecipe) scenario.TriggerRecipe {
	enabled := true
	noLoop := false
	return scenario.TriggerRecipe{
		Op:      "add_trigger",
		Name:    name,
		Enabled: &enabled,
		Looping: &noLoop,
		Conditions: []scenario.ConditionRecipe{
			{Op: "timer", Timer: intPtr(seconds)},
		},
		Effects: effects,
	}
}

func purchaseTrigger(player int, hero int, item string, token int, coinVar int, cost int, fn string) scenario.TriggerRecipe {
	enabled := true
	loop := true
	qty := 1
	name := fmt.Sprintf("A2KRPG P%dH%d Buy %s", player, hero, item)
	return scenario.TriggerRecipe{
		Op:      "add_trigger",
		Name:    name,
		Enabled: &enabled,
		Looping: &loop,
		Conditions: []scenario.ConditionRecipe{
			{Op: "own_objects", SourcePlayer: intPtr(player), ObjectList: intPtr(token), Quantity: &qty},
			{Op: "variable_value", Variable: &coinVar, Comparison: intPtr(cmpGE), Quantity: &cost},
		},
		Effects: []scenario.EffectRecipe{
			call(fn),
			remove(player, token, 0, 0, 119, 119, 1),
		},
	}
}

func dummyWaveEffects() []scenario.EffectRecipe {
	return []scenario.EffectRecipe{
		create(3, unitMilitia, 22, 22),
		create(3, unitMilitia, 23, 22),
		create(3, unitMilitia, 24, 22),
		create(3, unitMilitia, 25, 22),
		create(3, unitMilitia, 34, 22),
		create(3, unitMilitia, 35, 22),
		create(3, unitMilitia, 36, 22),
		create(3, unitMilitia, 37, 22),
		stance(3, unitMilitia, 21, 21, 38, 23, 0),
		hp(3, unitMilitia, 21, 21, 38, 23, 12),
		displayMsg("Red dummies spawned near both hero lanes. Kills now feed the XS directed-kill earn loop."),
	}
}

func view(x, y int) scenario.EffectRecipe {
	return scenario.EffectRecipe{Op: "change_view", SourcePlayer: intPtr(1), LocationX: &x, LocationY: &y, Scroll: intPtr(1)}
}

func call(fn string) scenario.EffectRecipe {
	return scenario.EffectRecipe{Op: "script_call", SourcePlayer: intPtr(1), Message: fn + "();"}
}

func displayMsg(msg string) scenario.EffectRecipe {
	return scenario.EffectRecipe{Op: "display_instructions", SourcePlayer: intPtr(1), Message: msg, DisplayTime: intPtr(9), PlaySound: intPtr(0)}
}

func setVar(id, value int) scenario.EffectRecipe {
	return scenario.EffectRecipe{Op: "change_variable", Variable: &id, Quantity: &value, Operation: intPtr(opSet)}
}

func addVar(id, value int) scenario.EffectRecipe {
	return scenario.EffectRecipe{Op: "change_variable", Variable: &id, Quantity: &value, Operation: intPtr(opAdd)}
}

func addResource(player, resource, amount int) scenario.EffectRecipe {
	return scenario.EffectRecipe{Op: "modify_resource", SourcePlayer: &player, Resource: &resource, Quantity: &amount, Operation: intPtr(opAdd)}
}

func renameTokenName(player, token int, name string) scenario.EffectRecipe {
	return scenario.EffectRecipe{Op: "change_object_name", SourcePlayer: &player, ObjectListUnitID: &token, Message: name}
}

func renameTokenDescription(player, token int, description string) scenario.EffectRecipe {
	return scenario.EffectRecipe{Op: "change_object_description", SourcePlayer: &player, ObjectListUnitID: &token, Message: description}
}

func shopCost(player, token, goldCost int) scenario.EffectRecipe {
	return scenario.EffectRecipe{
		Op:                "change_object_cost",
		SourcePlayer:      &player,
		ObjectListUnitID:  &token,
		Resource1:         intPtr(resourceFood),
		Resource1Quantity: intPtr(0),
		Resource2:         intPtr(resourceWood),
		Resource2Quantity: intPtr(0),
		Resource3:         intPtr(resourceGold),
		Resource3Quantity: intPtr(goldCost),
	}
}

func setTokenAttr(player, token, attr, value int) scenario.EffectRecipe {
	return scenario.EffectRecipe{
		Op:               "modify_attribute",
		SourcePlayer:     &player,
		ObjectListUnitID: &token,
		ObjectAttributes: &attr,
		Quantity:         &value,
		Operation:        intPtr(opAttributeSet),
	}
}

func create(player, unitConst, x, y int) scenario.EffectRecipe {
	return scenario.EffectRecipe{Op: "create_object", SourcePlayer: &player, ObjectListUnitID: &unitConst, LocationX: &x, LocationY: &y, DisableSound: intPtr(1)}
}

func remove(player, unitConst, x1, y1, x2, y2, max int) scenario.EffectRecipe {
	return scenario.EffectRecipe{Op: "remove_object", SourcePlayer: &player, ObjectListUnitID: &unitConst, AreaX1: &x1, AreaY1: &y1, AreaX2: &x2, AreaY2: &y2, MaxUnitsAffected: &max}
}

func stance(player, unitConst, x1, y1, x2, y2, attackStance int) scenario.EffectRecipe {
	max := -1
	return scenario.EffectRecipe{Op: "change_object_stance", SourcePlayer: &player, ObjectListUnitID: &unitConst, AreaX1: &x1, AreaY1: &y1, AreaX2: &x2, AreaY2: &y2, AttackStance: &attackStance, MaxUnitsAffected: &max}
}

func hp(player, unitConst, x1, y1, x2, y2, value int) scenario.EffectRecipe {
	return scenario.EffectRecipe{Op: "change_object_hp", SourcePlayer: &player, ObjectListUnitID: &unitConst, AreaX1: &x1, AreaY1: &y1, AreaX2: &x2, AreaY2: &y2, Quantity: &value, Operation: intPtr(0), MaxUnitsAffected: intPtr(-1)}
}

func addTrain(player, itemUnit, shopUnit, button, trainTime, hotkey int) scenario.EffectRecipe {
	return scenario.EffectRecipe{
		Op:                "add_train_location",
		SourcePlayer:      &player,
		ObjectListUnitID:  &itemUnit,
		ObjectListUnitID2: &shopUnit,
		ButtonLocation:    &button,
		TrainTime:         &trainTime,
		Hotkey:            &hotkey,
	}
}

func enableObject(player, unitConst, enabled int) scenario.EffectRecipe {
	return scenario.EffectRecipe{Op: "enable_disable_object", SourcePlayer: &player, ObjectListUnitID: &unitConst, Enabled: &enabled}
}

func unit(player, unitConst int, x, y float64, ref int, caption string) scenario.UnitRecipe {
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
		ReferenceID:   &ref,
		CaptionString: caption,
	}
}

func writeJSONFile(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0644)
}

func fileSHA256(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func boolPtr(v bool) *bool {
	return &v
}

func intPtr(v int) *int {
	return &v
}

func strPtr(v string) *string {
	return &v
}
