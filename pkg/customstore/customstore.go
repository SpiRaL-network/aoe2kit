// Package customstore provides a generic, scale-tested per-instance store
// recipe. It deliberately contains no scenario-specific item effects.
package customstore

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"aoe2kit/pkg/scenario"
	"aoe2kit/pkg/xs"
)

const (
	DefaultObjectUnitConst        = 236 // WCTW4 / Tower WEST Bombard, demo only
	DefaultPlayerCount            = 8
	DefaultInstances              = 440
	InstancesPerPlayer            = DefaultInstances / DefaultPlayerCount
	MapWidth                      = 120
	MapHeight                     = 120
	ReferenceIDBase               = 920000
	KeepAliveUnitConst            = 12 // Barracks; prevents DE from injecting a TC
	DefaultStartingResourceAmount = 100000
	FireUnitConst                 = 939  // FLAME1; HP-retired example purchase effect
	FlowerUnitConst               = 1366 // footprintless class-14 Gaia art; kill-zone retired
	FireCapacity                  = 96
	FireLifetimeSeconds           = 12
	KillSweepSeconds              = 120
	TokenUpgradeUnit              = 74 // placeholder token; content is intentionally not defined here
	TokenInspectUnit              = 93 // placeholder token; content is intentionally not defined here
	// The blank scenario retains one harmless seed trigger; the store adds setup,
	// polling, telemetry, and one bounded Gaia kill-zone sweep per second.
	TriggerCount        = 1 + AddedTriggerCount
	AddedTriggerCount   = 3 + KillSweepSeconds
	TriggerVariableHigh = 6
)

type Options struct {
	OutputDir       string
	ScenarioName    string
	Timestamp       int
	ObjectUnitConst int
	InstanceCount   int
	PlayerCount     int
	// ActivePlayers identifies the players allowed to interact with the store.
	// Empty means every configured player, preserving the scale-test behavior.
	ActivePlayers  []int
	XSCheckPath    string
	RequireXSCheck bool
}

type Report struct {
	OutputDir          string              `json:"output_dir"`
	ScenarioPath       string              `json:"scenario_path"`
	BaseScenarioPath   string              `json:"base_scenario_path"`
	RecipePath         string              `json:"recipe_path"`
	XSPath             string              `json:"xs_path"`
	TypesPath          string              `json:"types_path"`
	ScenarioSHA256     string              `json:"scenario_sha256"`
	XSSHA256           string              `json:"xs_sha256"`
	ObjectUnitConst    int                 `json:"object_unit_const"`
	InstanceCount      int                 `json:"instance_count"`
	InstancesPerPlayer int                 `json:"instances_per_player"`
	PlayerCount        int                 `json:"player_count"`
	ActivePlayers      []int               `json:"active_players"`
	TriggerCount       int                 `json:"trigger_count"`
	UnitCount          int                 `json:"unit_count"`
	Verification       string              `json:"verification"`
	KnownGaps          []string            `json:"known_gaps"`
	XSParseCheck       string              `json:"xs_parse_check"`
	XSParseChecker     string              `json:"xs_parse_checker,omitempty"`
	SemanticLint       scenario.LintReport `json:"semantic_lint"`
}

func Build(opts Options) (*Report, error) {
	if opts.OutputDir == "" {
		return nil, fmt.Errorf("custom store output directory is required")
	}
	if opts.ObjectUnitConst == 0 {
		opts.ObjectUnitConst = DefaultObjectUnitConst
	}
	if opts.InstanceCount == 0 {
		opts.InstanceCount = DefaultInstances
	}
	if opts.PlayerCount == 0 {
		opts.PlayerCount = DefaultPlayerCount
	}
	if opts.PlayerCount > DefaultPlayerCount {
		return nil, fmt.Errorf("player count %d exceeds sidecar limit %d", opts.PlayerCount, DefaultPlayerCount)
	}
	activePlayers, err := normalizeActivePlayers(opts.PlayerCount, opts.ActivePlayers)
	if err != nil {
		return nil, err
	}
	if opts.InstanceCount < opts.PlayerCount {
		return nil, fmt.Errorf("instance count %d must be at least player count %d", opts.InstanceCount, opts.PlayerCount)
	}
	if opts.InstanceCount%opts.PlayerCount != 0 {
		return nil, fmt.Errorf("instance count %d must divide evenly across %d players", opts.InstanceCount, opts.PlayerCount)
	}
	if opts.ScenarioName == "" {
		opts.ScenarioName = fmt.Sprintf("A2K Custom Store %d", opts.InstanceCount)
	}
	if err := os.MkdirAll(opts.OutputDir, 0755); err != nil {
		return nil, err
	}
	base := filepath.Join(opts.OutputDir, "A2K_CUSTOM_STORE_BLANK_BASE.aoe2scenario")
	scen := filepath.Join(opts.OutputDir, opts.ScenarioName+".aoe2scenario")
	recipePath := filepath.Join(opts.OutputDir, "A2K_CUSTOM_STORE_RECIPE.json")
	xsPath := filepath.Join(opts.OutputDir, "A2K_CUSTOM_STORE.xs")
	typesPath := filepath.Join(opts.OutputDir, "A2K_CUSTOM_STORE.types")

	if _, err := scenario.WriteBlankScenarioFile(base, scenario.BlankOptions{
		PlayerCount: opts.PlayerCount, HumanSlots: 1, MapWidth: MapWidth, MapHeight: MapHeight,
		Timestamp: opts.Timestamp, ClearTriggers: true, DummyStarters: true, DummyUnit: KeepAliveUnitConst,
		StartingResources: true, StartingFood: DefaultStartingResourceAmount, StartingWood: DefaultStartingResourceAmount, StartingGold: DefaultStartingResourceAmount, StartingStone: DefaultStartingResourceAmount, StartingTradeGoods: DefaultStartingResourceAmount,
		DummyStartX: 3.5, DummyStartY: 3.5, DummySpacing: 2,
	}); err != nil {
		return nil, err
	}
	if err := os.WriteFile(xsPath, []byte(XSModuleForPlayers(opts.ObjectUnitConst, opts.InstanceCount, opts.PlayerCount, activePlayers)), 0644); err != nil {
		return nil, err
	}
	parseStatus, parseChecker, err := checkGeneratedXS(xsPath, opts.XSCheckPath, opts.RequireXSCheck)
	if err != nil {
		return nil, err
	}
	if err := xs.WriteTypesDocument(typesPath, SidecarTypes()); err != nil {
		return nil, err
	}
	recipe := RecipeForPlayers(xsPath, opts.Timestamp, opts.ObjectUnitConst, opts.InstanceCount, opts.PlayerCount, activePlayers)
	data, err := json.MarshalIndent(recipe, "", "  ")
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(recipePath, append(data, '\n'), 0644); err != nil {
		return nil, err
	}
	patch, err := scenario.PatchRecipeFile(base, scen, recipe)
	if err != nil {
		return nil, err
	}
	semanticLint, err := scenario.LintFileWithOptions(scen, scenario.LintOptions{SemanticInvariants: true, Generated: true})
	if err != nil {
		return nil, fmt.Errorf("semantic lint generated custom store: %w", err)
	}
	if !semanticLint.OK {
		return nil, fmt.Errorf("generated custom store failed semantic lint: %s", semanticLintSummary(semanticLint))
	}
	scenHash, err := sha256File(scen)
	if err != nil {
		return nil, err
	}
	xsHash, err := sha256File(xsPath)
	if err != nil {
		return nil, err
	}
	return &Report{
		OutputDir: opts.OutputDir, ScenarioPath: scen, BaseScenarioPath: base,
		RecipePath: recipePath, XSPath: xsPath, TypesPath: typesPath, ScenarioSHA256: scenHash, XSSHA256: xsHash,
		ObjectUnitConst: opts.ObjectUnitConst, InstanceCount: opts.InstanceCount, InstancesPerPlayer: opts.InstanceCount / opts.PlayerCount, PlayerCount: opts.PlayerCount, ActivePlayers: activePlayers,
		TriggerCount: patch.TriggerCountAfter, UnitCount: patch.UnitCountAfter,
		Verification: fmt.Sprintf("structure_verified: fresh blank scenario, %d addressable object instances, store participants %v, one keep-alive barracks per player, one shared setup trigger, one polling trigger, one telemetry trigger; engine-unverified", opts.InstanceCount, activePlayers),
		XSParseCheck: parseStatus, XSParseChecker: parseChecker, SemanticLint: semanticLint,
		KnownGaps: []string{
			"The DE run must confirm that a train token appears at the selected object position and is mapped back to that instance; the XS sidecar records each mapping attempt.",
			"Token labels and gameplay effects are placeholders by design; demo prices are concrete and the example fire-ring effect proves the purchase path.",
			"object_selected_multiplayer remains a UI/trigger bridge for optional direct demonstrations; the scalable path is token position plus XS state.",
		},
	}, nil
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

func checkGeneratedXS(path, explicitChecker string, required bool) (status, checker string, err error) {
	checker, err = xs.ResolveParser(explicitChecker)
	if err != nil {
		return "unavailable", "", err
	}
	if checker == "" {
		if required {
			checker, err = xs.RequireParser(explicitChecker)
			return "unavailable", checker, err
		}
		return "unavailable", "", nil
	}
	if _, err := xs.CheckFile(path, checker); err != nil {
		return "failed", checker, err
	}
	return "passed", checker, nil
}

func Recipe(xsPath string, timestamp int, objectUnitConst, instanceCount, playerCount int) scenario.Recipe {
	return RecipeForPlayers(xsPath, timestamp, objectUnitConst, instanceCount, playerCount, allPlayers(playerCount))
}

// RecipeForPlayers keeps the full object/player scale while isolating store
// interaction to the supplied participants. Non-participants retain their
// objects for scale and their normal scenario slots, but receive no store UI.
func RecipeForPlayers(xsPath string, timestamp int, objectUnitConst, instanceCount, playerCount int, activePlayers []int) scenario.Recipe {
	active, looping, noLoop := true, true, false
	zero := 0
	activePlayers, _ = normalizeActivePlayers(playerCount, activePlayers)
	activeSet := make(map[int]bool, len(activePlayers))
	for _, player := range activePlayers {
		activeSet[player] = true
	}
	resources := make([]scenario.ResourceRecipe, 0, playerCount-1)
	for player := 1; player <= playerCount; player++ {
		if !activeSet[player] {
			zero := 0
			resources = append(resources, scenario.ResourceRecipe{Player: player, Gold: &zero, Wood: &zero, Food: &zero, Stone: &zero, TradeGoods: &zero})
		}
	}
	setup := scenario.TriggerRecipe{Op: "add_trigger", Name: fmt.Sprintf("A2K CustomStore Setup %d", instanceCount), Enabled: &active, Looping: &noLoop, Conditions: []scenario.ConditionRecipe{{Op: "timer", Timer: &zero}}}
	for _, player := range activePlayers {
		ids := objectIDs(player, instanceCount, playerCount)
		for _, token := range []int{TokenUpgradeUnit, TokenInspectUnit} {
			button := 1
			label := "+ Placeholder upgrade"
			description := "Placeholder store item; purchase draws a fire ring"
			foodCost, woodCost, goldCost := 0, 0, 0
			if token == TokenInspectUnit {
				button = 2
				label = "Inspect object state"
				woodCost = 100
			} else {
				goldCost = 100
			}
			populationAttribute, setOperation, freePopulation := 110, 1, 0
			setup.Effects = append(setup.Effects,
				scenario.EffectRecipe{Op: "add_train_location", SourcePlayer: intPtr(player), ObjectListUnitID: intPtr(token), ObjectListUnitID2: intPtr(objectUnitConst), ButtonLocation: intPtr(button), TrainTime: intPtr(0), SelectedObjectIDs: ids},
				scenario.EffectRecipe{Op: "enable_disable_object", SourcePlayer: intPtr(player), ObjectListUnitID: intPtr(token), Enabled: intPtr(1)},
				scenario.EffectRecipe{Op: "modify_attribute", SourcePlayer: intPtr(player), ObjectListUnitID: intPtr(token), ObjectAttributes: &populationAttribute, Operation: &setOperation, Quantity: &freePopulation},
				scenario.EffectRecipe{Op: "change_object_name", SourcePlayer: intPtr(player), ObjectListUnitID: intPtr(token), Message: label},
				scenario.EffectRecipe{Op: "change_object_description", SourcePlayer: intPtr(player), ObjectListUnitID: intPtr(token), Message: description},
				// Keep every cost slot valid. The engine hides the train button when
				// unused slots are encoded with -1; zero-cost slots are the working
				// convention used by the RPG shop demo.
				scenario.EffectRecipe{Op: "change_object_cost", SourcePlayer: intPtr(player), ObjectListUnitID: intPtr(token), Resource1: intPtr(0), Resource1Quantity: intPtr(foodCost), Resource2: intPtr(1), Resource2Quantity: intPtr(woodCost), Resource3: intPtr(3), Resource3Quantity: intPtr(goldCost)},
			)
		}
	}
	pollTimer := 1
	scriptPlayer := activePlayers[0]
	poll := scenario.TriggerRecipe{Op: "add_trigger", Name: "A2K CustomStore Poll Tokens", Enabled: &active, Looping: &looping, Conditions: []scenario.ConditionRecipe{{Op: "timer", Timer: &pollTimer}}, Effects: []scenario.EffectRecipe{{Op: "script_call", SourcePlayer: intPtr(scriptPlayer), Message: "CustomStore_Tick();"}}}
	telemetry := scenario.TriggerRecipe{Op: "add_trigger", Name: "A2K CustomStore Telemetry", Enabled: &active, Looping: &looping, Conditions: []scenario.ConditionRecipe{{Op: "timer", Timer: &pollTimer}}, Effects: []scenario.EffectRecipe{{Op: "script_call", SourcePlayer: intPtr(scriptPlayer), Message: "CustomStore_Report();"}}}
	return scenario.Recipe{
		Scenario:  &scenario.ScenarioRecipe{PlayerCount: intPtr(playerCount), TimestampOfLastSave: &timestamp, StartingAge: intPtr(6)},
		Resources: resources,
		XS:        &scenario.XSRecipe{Mode: "inline_runtime", Name: "A2K_CUSTOM_STORE.xs", ContentFile: xsPath, CarrierTitle: "XS string", CarrierTriggerName: "A2K CustomStore XS Runtime Source"},
		Variables: []scenario.VariableRecipe{{Op: "add_variable", ID: intPtr(6), Name: "A2K CustomStore Gaia sweep"}},
		Triggers:  append([]scenario.TriggerRecipe{setup, poll, telemetry}, killSweepTriggers()...),
		Units:     objectUnits(objectUnitConst, instanceCount, playerCount),
	}
}

func allPlayers(playerCount int) []int {
	players := make([]int, playerCount)
	for i := range players {
		players[i] = i + 1
	}
	return players
}

func normalizeActivePlayers(playerCount int, requested []int) ([]int, error) {
	if playerCount < 1 {
		return nil, fmt.Errorf("player count %d must be positive", playerCount)
	}
	if len(requested) == 0 {
		return allPlayers(playerCount), nil
	}
	seen := make(map[int]bool, len(requested))
	players := make([]int, 0, len(requested))
	for _, player := range requested {
		if player < 1 || player > playerCount {
			return nil, fmt.Errorf("active player %d out of range 1..%d", player, playerCount)
		}
		if !seen[player] {
			seen[player] = true
			players = append(players, player)
		}
	}
	sort.Ints(players)
	return players, nil
}

func killSweepTriggers() []scenario.TriggerRecipe {
	active, looping := true, false
	variable, comparison := 6, 0
	zoneX1, zoneY1, zoneX2, zoneY2 := 112, 112, 116, 116
	triggers := make([]scenario.TriggerRecipe, 0, KillSweepSeconds)
	for second := 1; second <= KillSweepSeconds; second++ {
		quantity := second
		triggers = append(triggers, scenario.TriggerRecipe{
			Op: "add_trigger", Name: fmt.Sprintf("A2K CustomStore Gaia sweep %d", second), Enabled: &active, Looping: &looping,
			Conditions: []scenario.ConditionRecipe{{Op: "variable_value", Variable: &variable, Comparison: &comparison, Quantity: &quantity}},
			Effects:    []scenario.EffectRecipe{{Op: "remove_object", SourcePlayer: intPtr(0), ObjectListUnitID: intPtr(FlowerUnitConst), AreaX1: &zoneX1, AreaY1: &zoneY1, AreaX2: &zoneX2, AreaY2: &zoneY2}},
		})
	}
	return triggers
}

func objectUnits(objectUnitConst, instanceCount, playerCount int) []scenario.UnitRecipe {
	units := make([]scenario.UnitRecipe, 0, instanceCount)
	for player := 1; player <= playerCount; player++ {
		for i, ref := range objectIDs(player, instanceCount, playerCount) {
			col, row := i%7, i/7
			x := 8.5 + float64((player-1)%4)*28 + float64(col)*2.25
			y := 8.5 + float64((player-1)/4)*50 + float64(row)*4.25
			units = append(units, scenario.UnitRecipe{Op: "add_unit", Player: player, UnitConst: objectUnitConst, X: &x, Y: &y, Status: intPtr(2), ReferenceID: &ref, CaptionString: fmt.Sprintf("Store object %d / P%d", i+1, player)})
		}
	}
	return units
}

func objectIDs(player, instanceCount, playerCount int) []int {
	perPlayer := instanceCount / playerCount
	ids := make([]int, perPlayer)
	for i := range ids {
		ids[i] = ReferenceIDBase + (player-1)*(perPlayer+1) + i
	}
	return ids
}

func XSModule(objectUnitConst, instanceCount, playerCount int) string {
	return XSModuleForPlayers(objectUnitConst, instanceCount, playerCount, allPlayers(playerCount))
}

// XSModuleForPlayers emits the same bounded runtime for the full object scale,
// but consumes tokens only for the selected store participants.
func XSModuleForPlayers(objectUnitConst, instanceCount, playerCount int, activePlayers []int) string {
	activePlayers, _ = normalizeActivePlayers(playerCount, activePlayers)
	consumeCalls := ""
	for _, player := range activePlayers {
		consumeCalls += fmt.Sprintf("  CustomStore_Consume(%d, A2K_TOKEN_UPGRADE, 1); CustomStore_Consume(%d, A2K_TOKEN_INSPECT, 0);\n", player, player)
	}
	return fmt.Sprintf(`// A2K generic custom store: %d addressable instances, no per-instance triggers.
// The .xsdat sidecar is a bounded, typed export of the store's live state.
const int A2K_OBJECT_UNIT = %d;
const int A2K_TOKEN_UPGRADE = 74;
const int A2K_TOKEN_INSPECT = 93;
const int A2K_PLAYERS = %d;
const int A2K_CAPACITY = %d;
const int A2K_FIRE_UNIT = %d;
const int A2K_FLOWER_UNIT = %d;
const int A2K_FIRE_CAPACITY = %d;
const int A2K_FIRE_LIFETIME = %d;
const int A2K_CLOSE_SECONDS = 60;
const int A2K_SWEEP_VARIABLE = 6;
const int A2K_SWEEP_SECONDS = %d;
const float A2K_KILL_ZONE_X = 112.5;
const float A2K_KILL_ZONE_Y = 112.5;

int a2kObjectIDs = -1;
int a2kObjectState = -1;
int a2kScan = -1;
int a2kPlayerCounts = -1;
int a2kCount = 0;
int a2kHighWater = 0;
int a2kRows = 0;
int a2kLastObject = -1;
int a2kLastBefore = 0;
int a2kLastAfter = 0;
int a2kFireIDs = -1;
int a2kFireBorn = -1;
int a2kHaloMaterial = -1;
int a2kFireCursor = 0;
int a2kSweepLastTime = -1;
int a2kSweepCount = 0;
bool a2kFileOpen = false;
bool a2kClosed = false;

int objectIndex(int unitID = -1) {
  for (i = 0; < a2kCount) { if (xsArrayGetInt(a2kObjectIDs, i) == unitID) return (i); }
  return (-1);
}

int nearestObject(int player = 1, vector pos = vector(0,0,0)) {
  int ids = xsGetPlayerUnitIds(player, A2K_OBJECT_UNIT, a2kScan); int n = xsArrayGetSize(ids); int best = -1; float bestD = 999999.0;
  for (j = 0; < n) {
    int id = xsArrayGetInt(ids, j); vector q = xsGetUnitPosition(id);
    float dx = xsVectorGetX(q) - xsVectorGetX(pos); float dy = xsVectorGetY(q) - xsVectorGetY(pos); float d = dx * dx + dy * dy;
    if (d < bestD) { bestD = d; best = id; }
  }
  return (best);
}

void CustomStore_Init() {
  a2kObjectIDs = xsArrayCreateInt(A2K_CAPACITY, -1, "object_ids");
  a2kObjectState = xsArrayCreateInt(A2K_CAPACITY, 0, "object_state");
  a2kScan = xsArrayCreateInt(A2K_CAPACITY, -1, "scan");
	a2kPlayerCounts = xsArrayCreateInt(9, 0, "player_counts");
	a2kFireIDs = xsArrayCreateInt(A2K_FIRE_CAPACITY, -1, "halo_ids");
	a2kFireBorn = xsArrayCreateInt(A2K_FIRE_CAPACITY, -1, "halo_born");
	a2kHaloMaterial = xsArrayCreateInt(A2K_FIRE_CAPACITY, -1, "halo_material");
  a2kCount = 0;

  for (p = 1; <= A2K_PLAYERS) {
    int ids = xsGetPlayerUnitIds(p, A2K_OBJECT_UNIT, a2kScan); int n = xsArrayGetSize(ids);
    xsArraySetInt(a2kPlayerCounts, p, n);
    for (j = 0; < n) { if (a2kCount < A2K_CAPACITY) { xsArraySetInt(a2kObjectIDs, a2kCount, xsArrayGetInt(ids, j)); a2kCount = a2kCount + 1; } }
  }
  a2kHighWater = a2kCount;
  a2kFileOpen = xsCreateFile(false);
  if (a2kFileOpen) {
    xsWriteString("A2K_CUSTOM_STORE"); xsWriteInt(1); xsWriteString("custom_store_%d");
    xsWriteInt(a2kCount); for (reportPlayer = 1; <= 8) xsWriteInt(xsArrayGetInt(a2kPlayerCounts, reportPlayer)); xsWriteInt(a2kHighWater);
    xsCloseFile();
  }
  if (a2kFileOpen) {
    if (xsCreateFile(true)) {
      xsWriteString("resource"); xsWriteInt(xsGetGameTime());
      xsWriteInt(xsPlayerAttribute(1, 0)); xsWriteInt(xsPlayerAttribute(1, 1)); xsWriteInt(xsPlayerAttribute(1, 3)); xsCloseFile();
    }
  }
  xsSetTriggerVariable(0, a2kCount); xsSetTriggerVariable(5, a2kHighWater);

  xsSetTriggerVariable(A2K_SWEEP_VARIABLE, 0);
  xsChatData("CUSTOMSTORE init objects=" + a2kCount + " capacity=" + A2K_CAPACITY + " triggers=3");
}

void CustomStore_AgeFire() {
  int now = xsGetGameTime();
  for (i = 0; < A2K_FIRE_CAPACITY) {
    int born = xsArrayGetInt(a2kFireBorn, i);
    int age = now - born;
    if (born >= 0 && age >= A2K_FIRE_LIFETIME) {
      int old = xsArrayGetInt(a2kFireIDs, i); int material = xsArrayGetInt(a2kHaloMaterial, i);
      if (old != -1) {
        if (material == A2K_FLOWER_UNIT) {
          float zx = A2K_KILL_ZONE_X + (i %% 16) * 0.20; float zy = A2K_KILL_ZONE_Y + (i / 16) * 0.20;
          if (xsSetUnitPosition(old, xsVectorSet(zx, zy, -1.0), false)) { xsArraySetInt(a2kFireIDs, i, -1); }
        } else { xsSetUnitHitpoints(old, 0.0); xsArraySetInt(a2kFireIDs, i, -1); }
      } else { xsArraySetInt(a2kFireIDs, i, -1); }
      if (xsArrayGetInt(a2kFireIDs, i) == -1) { xsArraySetInt(a2kFireBorn, i, -1); xsArraySetInt(a2kHaloMaterial, i, -1); }
    }
  }
}

void CustomStore_DrawOne(vector center = vector(0,0,0), float dx = 0.0, float dy = 0.0, int material = 939) {
  float px = xsVectorGetX(center) + dx; float py = xsVectorGetY(center) + dy;
  if ((px >= 0.5 && px < 119.5) && (py >= 0.5 && py < 119.5)) {
    int old = xsArrayGetInt(a2kFireIDs, a2kFireCursor); int oldMaterial = xsArrayGetInt(a2kHaloMaterial, a2kFireCursor);
    if (old != -1) {
      if (oldMaterial == A2K_FLOWER_UNIT) {
        float zx = A2K_KILL_ZONE_X + (a2kFireCursor %% 16) * 0.20; float zy = A2K_KILL_ZONE_Y + (a2kFireCursor / 16) * 0.20;
        xsSetUnitPosition(old, xsVectorSet(zx, zy, -1.0), false);
      } else { xsSetUnitHitpoints(old, 0.0); }
    }
    xsArraySetInt(a2kFireIDs, a2kFireCursor, xsCreateUnit(material, 0, xsVectorSet(px, py, -1.0), false, false, false));
    xsArraySetInt(a2kFireBorn, a2kFireCursor, xsGetGameTime());
    xsArraySetInt(a2kHaloMaterial, a2kFireCursor, material);
    a2kFireCursor = a2kFireCursor + 1;
    if (a2kFireCursor >= A2K_FIRE_CAPACITY) a2kFireCursor = 0;
  }
}

void CustomStore_FireOne(vector center = vector(0,0,0), float dx = 0.0, float dy = 0.0) { CustomStore_DrawOne(center, dx, dy, A2K_FIRE_UNIT); }

// Screen-space circle converted through the verified 2:1 isometric basis.
void CustomStore_FireRing(vector center = vector(0,0,0)) {
  CustomStore_FireOne(center, 2.000, 2.000); CustomStore_FireOne(center, -0.268, 3.732);
  CustomStore_FireOne(center, -2.464, 4.464); CustomStore_FireOne(center, -4.000, 4.000);
  CustomStore_FireOne(center, -4.464, 2.464); CustomStore_FireOne(center, -3.732, 0.268);
  CustomStore_FireOne(center, -2.000, -2.000); CustomStore_FireOne(center, 0.268, -3.732);
  CustomStore_FireOne(center, 2.464, -4.464); CustomStore_FireOne(center, 4.000, -4.000);
  CustomStore_FireOne(center, 4.464, -2.464); CustomStore_FireOne(center, 3.732, -0.268);
}

void CustomStore_FlowerRing(vector center = vector(0,0,0)) {
  CustomStore_DrawOne(center, 2.000, 2.000, A2K_FLOWER_UNIT); CustomStore_DrawOne(center, -0.268, 3.732, A2K_FLOWER_UNIT);
  CustomStore_DrawOne(center, -2.464, 4.464, A2K_FLOWER_UNIT); CustomStore_DrawOne(center, -4.000, 4.000, A2K_FLOWER_UNIT);
  CustomStore_DrawOne(center, -4.464, 2.464, A2K_FLOWER_UNIT); CustomStore_DrawOne(center, -3.732, 0.268, A2K_FLOWER_UNIT);
  CustomStore_DrawOne(center, -2.000, -2.000, A2K_FLOWER_UNIT); CustomStore_DrawOne(center, 0.268, -3.732, A2K_FLOWER_UNIT);
  CustomStore_DrawOne(center, 2.464, -4.464, A2K_FLOWER_UNIT); CustomStore_DrawOne(center, 4.000, -4.000, A2K_FLOWER_UNIT);
  CustomStore_DrawOne(center, 4.464, -2.464, A2K_FLOWER_UNIT); CustomStore_DrawOne(center, 3.732, -0.268, A2K_FLOWER_UNIT);
}

void CustomStore_WritePurchase(int time = 0, int player = 1, int tokenType = 74, int token = -1, vector pos = vector(0,0,0), int object = -1, int before = 0, int after = 0, int neighbor = -1, int neighborBefore = 0, int neighborAfter = 0) {
  if (a2kFileOpen == false) return;
  if (xsCreateFile(true) == false) return;
  xsWriteString("purchase"); xsWriteInt(time); xsWriteInt(a2kCount); xsWriteInt(a2kHighWater);
  xsWriteInt(player); xsWriteInt(tokenType); xsWriteInt(token); xsWriteFloat(xsVectorGetX(pos)); xsWriteFloat(xsVectorGetY(pos));
  xsWriteInt(object); xsWriteInt(neighbor); xsWriteInt(before); xsWriteInt(after); xsWriteInt(neighborBefore); xsWriteInt(neighborAfter);
  a2kRows = a2kRows + 1; xsCloseFile();
}

void CustomStore_Close() {
  if (a2kFileOpen == false || a2kClosed) return;
  if (xsCreateFile(true)) { xsWriteString("end"); xsWriteInt(a2kRows); xsWriteInt(a2kHighWater); xsWriteInt(424242); xsCloseFile(); }
  a2kFileOpen = false; a2kClosed = true;
}

void CustomStore_Consume(int player = 1, int tokenType = 74, int delta = 1) {
  int tokens = xsGetPlayerUnitIds(player, tokenType, a2kScan); int n = xsArrayGetSize(tokens);
  for (i = 0; < n) {
    int token = xsArrayGetInt(tokens, i); vector pos = xsGetUnitPosition(token); int object = nearestObject(player, pos); int ix = objectIndex(object); int neighbor = -1; int neighborBefore = 0; int neighborAfter = 0;
    a2kLastBefore = 0; a2kLastAfter = 0;
    if (ix >= 0) {
      a2kLastObject = object; a2kLastBefore = xsArrayGetInt(a2kObjectState, ix); a2kLastAfter = a2kLastBefore + delta; xsArraySetInt(a2kObjectState, ix, a2kLastAfter);
      neighbor = xsArrayGetInt(a2kObjectIDs, (ix + 1) %% a2kCount); neighborBefore = xsArrayGetInt(a2kObjectState, (ix + 1) %% a2kCount); neighborAfter = neighborBefore;
      xsSetTriggerVariable(1, a2kLastObject); xsSetTriggerVariable(2, a2kLastBefore); xsSetTriggerVariable(3, a2kLastAfter); xsSetTriggerVariable(4, neighborAfter);
      if (tokenType == A2K_TOKEN_INSPECT) CustomStore_FlowerRing(pos); else CustomStore_FireRing(pos);
      xsChatData("CUSTOMSTORE purchase object=" + object + " before=" + a2kLastBefore + " after=" + a2kLastAfter + " neighbor=" + neighbor + " neighbor_state=" + neighborAfter);
    }
    CustomStore_WritePurchase(xsGetGameTime(), player, tokenType, token, pos, object, a2kLastBefore, a2kLastAfter, neighbor, neighborBefore, neighborAfter);
    xsRemoveUnit(token);
  }
}

void CustomStore_Tick() {
  if (a2kCount == 0) CustomStore_Init();
  CustomStore_AgeFire();
  int now = xsGetGameTime();
  if (now != a2kSweepLastTime) { a2kSweepLastTime = now; if (a2kSweepCount < A2K_SWEEP_SECONDS) { a2kSweepCount = a2kSweepCount + 1; xsSetTriggerVariable(A2K_SWEEP_VARIABLE, a2kSweepCount); } }
%s  if (xsGetGameTime() >= A2K_CLOSE_SECONDS) CustomStore_Close();
}

void CustomStore_Report() { xsSetTriggerVariable(0, a2kCount); xsSetTriggerVariable(5, a2kHighWater); }

void main() { CustomStore_Init(); }

rule customStoreTick
  active
  minInterval 1
  maxInterval 1
{ CustomStore_Tick(); }

rule customStoreReport
  active
  minInterval 5
  maxInterval 5
{ CustomStore_Report(); }
`, instanceCount, objectUnitConst, playerCount, instanceCount, FireUnitConst, FlowerUnitConst, FireCapacity, FireLifetimeSeconds, KillSweepSeconds, instanceCount, consumeCalls)
}

// SidecarTypes documents the exact xsWrite order emitted by XSModule. It is
// intentionally plain text so external scenario tooling can consume the
// fixture without importing AoE2Kit.
func SidecarTypes() string {
	return `# A2K_CUSTOM_STORE.xsdat
# Typed stream: header, zero or more purchase rows, end footer.
# Header: magic string, version int, fixture string, total_objects int,
#         player_1_objects..player_8_objects int, array_high_water int
header = string,int,string,int,int,int,int,int,int,int,int,int,int
# Startup engine probe: tag string, game_time int, P1 food, wood, and gold
# read through xsPlayerAttribute (attributes 0, 1, and 3).
resource = string,int,int,int,int
# Row: tag string, game_time int, total_objects int, array_high_water int,
#      player int, token_type int, token_id int, token_x float, token_y float,
#      resolved_object_id int, neighbor_object_id int, state_before int,
#      state_after int, neighbor_state_before int, neighbor_state_after int
purchase = string,int,int,int,int,int,int,float,float,int,int,int,int,int,int
# Footer: tag string, rows_written int, array_high_water int, integrity int
end = string,int,int,int
footer_integrity = 424242
`
}

func intPtr(v int) *int { return &v }

func sha256File(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}
