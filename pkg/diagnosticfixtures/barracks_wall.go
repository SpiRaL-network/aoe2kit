package diagnosticfixtures

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"aoe2kit/pkg/scenario"
	"aoe2kit/pkg/xs"
)

const (
	barracksWallMapSize  = 168
	barracksWallUnit     = 12
	barracksWallVillager = 83
	barracksWallInfantry = 74
	barracksWallOutpost  = 598
)

// BuildBarracksWall creates a human-pathing A/B fixture. Each row belongs to a
// different allied player so type-level attribute changes cannot contaminate
// another row. Rows B-E have pre-placed and post-modification halves.
func BuildBarracksWall(dir string, timestamp int) (*Report, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	base := filepath.Join(dir, "A2K_Barracks_Wall_BLANK_BASE.aoe2scenario")
	scen := filepath.Join(dir, "A2K Barracks Wall AB.aoe2scenario")
	recipePath := filepath.Join(dir, "A2K_Barracks_Wall_AB.recipe.json")
	if _, err := scenario.WriteBlankScenarioFile(base, scenario.BlankOptions{
		PlayerCount: 6, HumanSlots: 1, MapWidth: barracksWallMapSize, MapHeight: barracksWallMapSize,
		Timestamp: timestamp, ClearTriggers: true, NoStarters: true,
		StartingResources: true, StartingFood: 100000, StartingWood: 100000,
		StartingGold: 100000, StartingStone: 100000, StartingTradeGoods: 100000,
	}); err != nil {
		return nil, err
	}
	recipe := barracksWallRecipe(timestamp)
	data, err := json.MarshalIndent(recipe, "", "  ")
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(recipePath, append(data, '\n'), 0644); err != nil {
		return nil, err
	}
	if _, err := scenario.PatchRecipeFile(base, scen, recipe); err != nil {
		return nil, err
	}
	return &Report{
		ScenarioPath: scen,
		RecipePath:   recipePath,
		ScenarioSHA:  hashBarracksWallFile(scen),
	}, nil
}

// BuildBarracksWallV2 adds an unchanged spawned control to the wall and embeds
// a small XS telemetry probe for the engine-visible unit state.
func BuildBarracksWallV2(dir string, timestamp int) (*Report, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	base := filepath.Join(dir, "A2K_Barracks_Wall_V2_BLANK_BASE.aoe2scenario")
	scen := filepath.Join(dir, "A2K Barracks Wall AB v2.aoe2scenario")
	xsPath := filepath.Join(dir, "A2K_Barracks_Wall_AB_v2.xs")
	typesPath := filepath.Join(dir, "A2K_BARRACKS_WALL_V2.types")
	if _, err := scenario.WriteBlankScenarioFile(base, scenario.BlankOptions{
		PlayerCount: 6, HumanSlots: 1, MapWidth: barracksWallMapSize, MapHeight: barracksWallMapSize,
		Timestamp: timestamp, ClearTriggers: true, NoStarters: true,
		StartingResources: true, StartingFood: 100000, StartingWood: 100000,
		StartingGold: 100000, StartingStone: 100000, StartingTradeGoods: 100000,
	}); err != nil {
		return nil, err
	}
	if err := os.WriteFile(xsPath, []byte(barracksWallV2XS()), 0644); err != nil {
		return nil, err
	}
	if err := xs.WriteTypesDocument(typesPath, barracksWallV2Types()); err != nil {
		return nil, err
	}
	recipe := barracksWallRecipe(timestamp)
	recipe.XS = &scenario.XSRecipe{
		Mode: "inline_runtime", Name: filepath.Base(xsPath), ContentFile: xsPath,
		CarrierTitle: "XS string", CarrierTriggerName: "A2K Barracks Wall v2 Telemetry",
	}
	// Row A is the unchanged spawned control. B-E remain the modified spawned
	// rows from v1, so every right-hand row shares the same creation mechanism.
	active := true
	timer := 2
	y := 24
	effects := make([]scenario.EffectRecipe, 0, 5)
	for i := 0; i < 5; i++ {
		x := 68 + i*4
		effects = append(effects, scenario.EffectRecipe{Op: "create_object", SourcePlayer: intPtr(2), ObjectListUnitID: intPtr(barracksWallUnit), LocationX: intPtr(x), LocationY: &y})
	}
	recipe.Triggers = append(recipe.Triggers, scenario.TriggerRecipe{
		Op: "add_trigger", Name: "A2K Wall A unchanged spawned control", Enabled: &active, Looping: boolPtr(false),
		Conditions: []scenario.ConditionRecipe{{Op: "timer", Timer: &timer}}, Effects: effects,
	})
	recipePath := filepath.Join(dir, "A2K_Barracks_Wall_AB_v2.recipe.json")
	data, err := json.MarshalIndent(recipe, "", "  ")
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(recipePath, append(data, '\n'), 0644); err != nil {
		return nil, err
	}
	if _, err := scenario.PatchRecipeFile(base, scen, recipe); err != nil {
		return nil, err
	}
	return &Report{
		ScenarioPath: scen, XSPath: xsPath, TypesPath: typesPath, RecipePath: recipePath,
		ScenarioSHA: hashBarracksWallFile(scen), XSSHA: hashBarracksWallFile(xsPath),
	}, nil
}

// BuildRyanScratchAB preserves the editor-authored P1 wall and adds two
// aligned spawned walls to the 1.59 scratch. P2 receives the four attribute
// changes before spawning; P3 is the unchanged spawned control.
func BuildRyanScratchAB(basePath, dir string, timestamp int) (*Report, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	scen := filepath.Join(dir, "RyanScratch AB.aoe2scenario")
	xsPath := filepath.Join(dir, "A2K_RyanScratch_AB.xs")
	typesPath := filepath.Join(dir, "A2K_RYAN_SCRATCH_AB.types")
	recipePath := filepath.Join(dir, "A2K_RyanScratch_AB.recipe.json")
	if err := os.WriteFile(xsPath, []byte(barracksWallV2XS()), 0644); err != nil {
		return nil, err
	}
	if err := xs.WriteTypesDocument(typesPath, barracksWallV2Types()); err != nil {
		return nil, err
	}
	active := true
	human := false
	players := []scenario.PlayerRecipe{
		{Player: 1, Active: &active, Human: &active},
		{Player: 2, Active: &active, Human: &human, AIType: intPtr(0)},
		{Player: 3, Active: &active, Human: &human, AIType: intPtr(0)},
	}
	recipe := scenario.Recipe{
		Scenario: &scenario.ScenarioRecipe{TimestampOfLastSave: &timestamp},
		Players:  players,
		XS: &scenario.XSRecipe{Mode: "inline_runtime", Name: filepath.Base(xsPath), ContentFile: xsPath,
			CarrierTitle: "XS string", CarrierTriggerName: "A2K RyanScratch AB telemetry"},
		Units: []scenario.UnitRecipe{{Op: "remove_units_for_player", TargetPlayer: intPtr(2)}},
	}
	// The editor-authored source contains four P1 calibration triggers. Keep
	// the source scenario intact, but repair their trigger operation enum in the
	// generated A/B copy so the control is a real SET rather than operation 0.
	for _, attribute := range []int{3, 4, 200, 201} {
		quantity := 1.0
		recipe.Triggers = append(recipe.Triggers, scenario.TriggerRecipe{
			Op: "edit_trigger", TargetName: fmt.Sprintf("Scratch P1 barracks attr %d = 1.0", attribute),
			ReplaceEffects: []scenario.EffectRecipe{{Op: "modify_attribute", SourcePlayer: intPtr(1), ObjectListUnitID: intPtr(barracksWallUnit), ObjectAttributes: &attribute, QuantityFloat: &quantity, OperationName: "set"}},
		})
	}
	for _, player := range []int{2, 3} {
		recipe.Diplomacy = append(recipe.Diplomacy,
			scenario.DiplomacyRecipe{From: 1, To: player, Stance: 3},
			scenario.DiplomacyRecipe{From: player, To: 1, Stance: 3})
	}
	for _, attribute := range []int{3, 4, 200, 201} {
		recipe.Triggers = append(recipe.Triggers, barracksWallModifyTrigger(2, attribute, 1.0))
	}
	for _, player := range []int{2, 3} {
		recipe.Triggers = append(recipe.Triggers, alignedBarracksWallSpawnTrigger(player, 2, 40.5+float64(player-2)*24.0))
	}
	for i, x := range []float64{16.5, 48.5, 80.5} {
		y := 8.5
		recipe.Units = append(recipe.Units, scenario.UnitRecipe{Op: "add_unit", Player: 1, UnitConst: barracksWallVillager, X: &x, Y: &y, Status: intPtr(2), CaptionString: fmt.Sprintf("RyanScratch AB walker %d", i+1)})
	}
	data, err := json.MarshalIndent(recipe, "", "  ")
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(recipePath, append(data, '\n'), 0644); err != nil {
		return nil, err
	}
	if _, err := scenario.PatchRecipeFile(basePath, scen, recipe); err != nil {
		return nil, err
	}
	return &Report{ScenarioPath: scen, XSPath: xsPath, TypesPath: typesPath, RecipePath: recipePath,
		ScenarioSHA: hashBarracksWallFile(scen), XSSHA: hashBarracksWallFile(xsPath)}, nil
}

// BuildRyanScratchAB2 creates a fresh five-wall clearance fixture from the
// editor-authored 1.59 scratch. It deliberately does not mutate or extend the
// existing RyanScratch or RyanScratch AB artifacts.
func BuildRyanScratchAB2(basePath, dir string, timestamp int) (*Report, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	scen := filepath.Join(dir, "RyanScratch AB2.aoe2scenario")
	xsPath := filepath.Join(dir, "A2K_RyanScratch_AB2.xs")
	typesPath := filepath.Join(dir, "A2K_RYAN_SCRATCH_AB2.types")
	recipePath := filepath.Join(dir, "A2K_RyanScratch_AB2.recipe.json")
	if err := os.WriteFile(xsPath, []byte(barracksWallAB2XS()), 0644); err != nil {
		return nil, err
	}
	if err := xs.WriteTypesDocument(typesPath, barracksWallAB2Types()); err != nil {
		return nil, err
	}
	active := true
	human := false
	inactive := false
	startAge := 6
	wood := 500
	recipe := scenario.Recipe{
		Scenario: &scenario.ScenarioRecipe{PlayerCount: intPtr(5), TimestampOfLastSave: &timestamp, StartingAge: &startAge},
		Players: []scenario.PlayerRecipe{
			{Player: 0, Active: &active, Human: &active},
			{Player: 1, Active: &active, Human: &human, AIType: intPtr(0)},
			{Player: 2, Active: &active, Human: &human, AIType: intPtr(0)},
			{Player: 3, Active: &active, Human: &human, AIType: intPtr(0)},
			{Player: 4, Active: &active, Human: &human, AIType: intPtr(0)},
			{Player: 5, Active: &inactive, Human: &human, AIType: intPtr(0)},
		},
		Resources: []scenario.ResourceRecipe{{Player: 0, Wood: &wood}},
		XS: &scenario.XSRecipe{Mode: "inline_runtime", Name: filepath.Base(xsPath), ContentFile: xsPath,
			CarrierTitle: "XS string", CarrierTriggerName: "A2K RyanScratch AB2 telemetry"},
	}
	for player := 2; player <= 5; player++ {
		from := player - 2
		to := player - 1
		recipe.Diplomacy = append(recipe.Diplomacy,
			scenario.DiplomacyRecipe{From: from, To: to, Stance: 3},
			scenario.DiplomacyRecipe{From: to, To: from, Stance: 3})
	}
	// RyanScratch contains one P2 keep-alive barracks. Remove only that helper;
	// P3-P5 are empty in the source and must not be treated as errors.
	recipe.Units = append(recipe.Units, scenario.UnitRecipe{Op: "remove_units_for_player", TargetPlayer: intPtr(2)})
	for player := 2; player <= 5; player++ {
		x := 4.5 + float64(player-2)*3.0
		y := 150.5
		recipe.Units = append(recipe.Units, scenario.UnitRecipe{Op: "add_unit", Player: player, UnitConst: barracksWallOutpost, X: &x, Y: &y, Status: intPtr(2)})
	}
	// Drop RyanScratch's four calibration triggers in the AB2 copy. They are
	// inherited authoring probes, not part of the baseline wall; retaining them
	// would mutate P1 before the comparison and violate the clean-control gate.
	for _, attribute := range []int{3, 4, 200, 201} {
		recipe.Triggers = append(recipe.Triggers, scenario.TriggerRecipe{
			Op: "remove_trigger", TargetName: fmt.Sprintf("Scratch P1 barracks attr %d = 1.0", attribute),
		})
	}

	// P1 remains exactly the editor-authored baseline. The other four columns
	// are created after their respective pre-spawn treatment paths run.
	recipe.Triggers = append(recipe.Triggers,
		barracksWallModifyTrigger(2, 3, 1.0), barracksWallModifyTrigger(2, 4, 1.0),
		barracksWallModifyTrigger(4, 200, 1.0), barracksWallModifyTrigger(4, 201, 1.0))
	for _, row := range []struct {
		player int
		x      float64
	}{{2, 40.0}, {3, 64.0}, {4, 88.0}, {5, 112.0}} {
		player, x := row.player, row.x
		recipe.Triggers = append(recipe.Triggers, alignedBarracksWallSpawnTrigger(player, 2, x))
	}
	for _, x := range []float64{16.5, 48.5, 72.5, 96.5, 120.5} {
		y := 8.5
		recipe.Units = append(recipe.Units, scenario.UnitRecipe{Op: "add_unit", Player: 1, UnitConst: barracksWallVillager, X: &x, Y: &y, Status: intPtr(2)})
	}
	data, err := json.MarshalIndent(recipe, "", "  ")
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(recipePath, append(data, '\n'), 0644); err != nil {
		return nil, err
	}
	if _, err := scenario.PatchRecipeFile(basePath, scen, recipe); err != nil {
		return nil, err
	}
	return &Report{ScenarioPath: scen, XSPath: xsPath, TypesPath: typesPath, RecipePath: recipePath,
		ScenarioSHA: hashBarracksWallFile(scen), XSSHA: hashBarracksWallFile(xsPath)}, nil
}

func alignedBarracksWallSpawnTrigger(player, timer int, x float64) scenario.TriggerRecipe {
	y := 8.5
	effects := make([]scenario.EffectRecipe, 0, 10)
	for i := 0; i < 10; i++ {
		yi := int(y + float64(i)*3.0)
		xi := int(x)
		effects = append(effects, scenario.EffectRecipe{Op: "create_object", SourcePlayer: intPtr(player), ObjectListUnitID: intPtr(barracksWallUnit), LocationX: &xi, LocationY: &yi})
	}
	return scenario.TriggerRecipe{Op: "add_trigger", Name: fmt.Sprintf("RyanScratch AB P%d aligned wall", player), Enabled: boolPtr(true), Looping: boolPtr(false), Conditions: []scenario.ConditionRecipe{{Op: "timer", Timer: &timer}}, Effects: effects}
}

func barracksWallV2Types() string {
	return `# A2K_BARRACKS_WALL_V2.xsdat
# Header: magic string, version int, fixture string
header = string,int,string
# Row: tag string, unit id, owner, integer-coordinate floats, and four attributes
unit = string,int,int,float,float,float,float,float,float
# Footer: end tag and integrity marker
end = string,int
footer_integrity = 424242
`
}

func barracksWallAB2Types() string {
	return `# A2K_RYAN_SCRATCH_AB2.xsdat
# Header: magic string, version int, fixture string
header = string,int,string
# Row: tag string, unit id, owner, x, y, collision x/y, clearance x/y
unit = string,int,int,float,float,float,float,float,float
# Footer: end tag and integrity marker
end = string,int
footer_integrity = 424242
`
}

func barracksWallRecipe(timestamp int) scenario.Recipe {
	active := true
	human := false
	playerCount := 6
	startAge := 6
	recipe := scenario.Recipe{
		Scenario: &scenario.ScenarioRecipe{PlayerCount: &playerCount, TimestampOfLastSave: &timestamp, StartingAge: &startAge},
	}
	recipe.Players = append(recipe.Players, scenario.PlayerRecipe{Player: 0, Active: &active, Human: boolPtr(true), AIType: intPtr(0)})
	for player := 1; player <= 5; player++ {
		recipe.Players = append(recipe.Players, scenario.PlayerRecipe{Player: player, Active: &active, Human: &human, AIType: intPtr(0)})
		recipe.Diplomacy = append(recipe.Diplomacy,
			scenario.DiplomacyRecipe{From: 1, To: player, Stance: 3},
			scenario.DiplomacyRecipe{From: player, To: 1, Stance: 3},
		)
	}

	// The row map is deliberately regular: pitch 4, five buildings per half,
	// and twenty tiles between rows so units can be staged without overlap.
	for row := 0; row < 5; row++ {
		player := row + 2
		y := 24.0 + float64(row)*20.0
		label := []string{"A control", "B collision 1.0", "C clearance 1.0", "D both 1.0", "E both 0.5"}[row]
		for i := 0; i < 5; i++ {
			x := 28.0 + float64(i)*4.0
			caption := ""
			if i == 0 {
				caption = fmt.Sprintf("%s PREPLACED pitch4", label)
			}
			recipe.Units = append(recipe.Units, scenario.UnitRecipe{Op: "add_unit", Player: player, UnitConst: barracksWallUnit, X: &x, Y: &y, Status: intPtr(2), CaptionString: caption})
		}
	}
	// P1 has a keep-alive barracks plus a villager and infantry unit in front of each row. Their different
	// footprints make the human oracle test both unit-size classes.
	keepAliveX, keepAliveY := 10.0, 10.0
	recipe.Units = append(recipe.Units, scenario.UnitRecipe{Op: "add_unit", Player: 1, UnitConst: barracksWallUnit, X: &keepAliveX, Y: &keepAliveY, Status: intPtr(2), CaptionString: "P1 keep-alive barracks"})
	for row := 0; row < 5; row++ {
		y := 24.0 + float64(row)*20.0
		for i, unit := range []int{barracksWallVillager, barracksWallInfantry} {
			x := 18.0 + float64(i)*2.0
			recipe.Units = append(recipe.Units, scenario.UnitRecipe{Op: "add_unit", Player: 1, UnitConst: unit, X: &x, Y: &y, Status: intPtr(2), CaptionString: fmt.Sprintf("P1 row %c walker", 'A'+row)})
		}
	}

	// Apply type-level changes at load. Each row has its own owner so these
	// modifications are isolated to one row's barracks.
	for row := 1; row < 5; row++ {
		player := row + 2
		quantity := 1.0
		if row == 4 {
			quantity = 0.5
		}
		if row == 1 || row == 3 || row == 4 {
			recipe.Triggers = append(recipe.Triggers, barracksWallModifyTrigger(player, 3, quantity), barracksWallModifyTrigger(player, 4, quantity))
		}
		if row == 2 || row == 3 || row == 4 {
			recipe.Triggers = append(recipe.Triggers, barracksWallModifyTrigger(player, 200, quantity), barracksWallModifyTrigger(player, 201, quantity))
		}
	}

	// The right half is created after the t=0 changes. A difference between the
	// left and right halves is evidence about placement-time pathing stamping.
	for row := 1; row < 5; row++ {
		player := row + 2
		y := int(24 + row*20)
		effects := make([]scenario.EffectRecipe, 0, 5)
		for i := 0; i < 5; i++ {
			x := 68 + i*4
			effects = append(effects, scenario.EffectRecipe{Op: "create_object", SourcePlayer: intPtr(player), ObjectListUnitID: intPtr(barracksWallUnit), LocationX: intPtr(x), LocationY: &y})
		}
		timer := 2
		recipe.Triggers = append(recipe.Triggers, scenario.TriggerRecipe{Op: "add_trigger", Name: fmt.Sprintf("A2K Wall %c create after modify", 'A'+row), Enabled: &active, Looping: boolPtr(false), Conditions: []scenario.ConditionRecipe{{Op: "timer", Timer: &timer}}, Effects: effects})
	}
	return recipe
}

func barracksWallModifyTrigger(player, attribute int, quantity float64) scenario.TriggerRecipe {
	timer := 0
	return scenario.TriggerRecipe{
		Op: "add_trigger", Name: fmt.Sprintf("A2K Wall slot %d set attr %d", player, attribute), Enabled: boolPtr(true), Looping: boolPtr(false),
		Conditions: []scenario.ConditionRecipe{{Op: "timer", Timer: &timer}},
		Effects:    []scenario.EffectRecipe{{Op: "modify_attribute", SourcePlayer: intPtr(player), ObjectListUnitID: intPtr(barracksWallUnit), ObjectAttributes: &attribute, QuantityFloat: &quantity, OperationName: "set"}},
	}
}

func barracksWallV2XS() string {
	return `int a2kScan = -1;
bool a2kDone = false;

void A2K_WriteBarracks(int id = -1) {
  if (id < 0) return;
  vector pos = xsGetUnitPosition(id);
  xsWriteString("unit");
  xsWriteInt(id);
  xsWriteInt(xsGetUnitOwner(id));
  xsWriteFloat(xsVectorGetX(pos));
  xsWriteFloat(xsVectorGetY(pos));
  xsWriteFloat(xsGetUnitAttribute(id, 3, -1));
  xsWriteFloat(xsGetUnitAttribute(id, 4, -1));
  xsWriteFloat(xsGetUnitAttribute(id, 200, -1));
  xsWriteFloat(xsGetUnitAttribute(id, 201, -1));
}

int A2K_TreatmentState(int id = -1, int attr = -1) {
  if (id < 0) return (-1);
  float value = xsGetUnitAttribute(id, attr, -1);
  if (value == -1.0) return (-1);
  if (value == 1.0) return (1);
  return (0);
}

int A2K_CombineTreatmentState(int current = 1, int next = 1) {
  if (current == -1 || next == -1) return (-1);
  if (current == 0 || next == 0) return (0);
  return (1);
}

void A2K_ReportTreatment(int attr = -1, int state = -1) {
  if (attr == 3) {
    if (state == 1) xsChatData("collision_x APPLIED");
    if (state == 0) xsChatData("collision_x NOT APPLIED");
    if (state == -1) xsChatData("collision_x UNKNOWN");
  }
  if (attr == 4) {
    if (state == 1) xsChatData("collision_y APPLIED");
    if (state == 0) xsChatData("collision_y NOT APPLIED");
    if (state == -1) xsChatData("collision_y UNKNOWN");
  }
  if (attr == 200) {
    if (state == 1) xsChatData("clearance_x APPLIED");
    if (state == 0) xsChatData("clearance_x NOT APPLIED");
    if (state == -1) xsChatData("clearance_x UNKNOWN");
  }
  if (attr == 201) {
    if (state == 1) xsChatData("clearance_y APPLIED");
    if (state == 0) xsChatData("clearance_y NOT APPLIED");
    if (state == -1) xsChatData("clearance_y UNKNOWN");
  }
}

void A2K_BarracksWallV2Tick() {
  if (a2kDone) return;
  if (xsGetGameTime() < 4) return;
  bool opened = xsCreateFile(false);
  if (opened == false) return;
  xsWriteString("A2K_BARRACKS_WALL_V2");
  xsWriteInt(1);
  xsWriteString("spawned-control");
  a2kScan = xsArrayCreateInt(64, -1, "barracks_scan");
  int p = 1;
  int treatment3 = 1;
  int treatment4 = 1;
  int treatment200 = 1;
  int treatment201 = 1;
  int treatmentCount = 0;
  for (p = 1; <= 6) {
    int ids = xsGetPlayerUnitIds(p, 12, a2kScan);
    int n = xsArrayGetSize(ids);
    int i = 0;
    for (i = 0; < n) {
      int id = xsArrayGetInt(ids, i);
      A2K_WriteBarracks(id);
      if (p == 2) {
        treatmentCount = treatmentCount + 1;
        treatment3 = A2K_CombineTreatmentState(treatment3, A2K_TreatmentState(id, 3));
        treatment4 = A2K_CombineTreatmentState(treatment4, A2K_TreatmentState(id, 4));
        treatment200 = A2K_CombineTreatmentState(treatment200, A2K_TreatmentState(id, 200));
        treatment201 = A2K_CombineTreatmentState(treatment201, A2K_TreatmentState(id, 201));
      }
    }
  }
  if (treatmentCount > 0) {
    A2K_ReportTreatment(3, treatment3);
    A2K_ReportTreatment(4, treatment4);
    A2K_ReportTreatment(200, treatment200);
    A2K_ReportTreatment(201, treatment201);
  }
  xsWriteString("end");
  xsWriteInt(424242);
  xsCloseFile();
  a2kDone = true;
}

rule A2K_BarracksWallV2 active minInterval 1 {
  A2K_BarracksWallV2Tick();
}
`
}

func barracksWallAB2XS() string {
	return `bool a2kEffectDone = false;
bool a2kDone = false;
int a2kScan = -1;

void A2K_ApplyP3Clearance() {
  if (a2kEffectDone) return;
  if (xsGetGameTime() < 1) return;
  xsEffectAmount(0, 12, 200, 1.0, 3);
  xsEffectAmount(0, 12, 201, 1.0, 3);
  a2kEffectDone = true;
}

int A2K_TreatmentState(int id = -1, int attr = -1, float expected = 1.0) {
  if (id < 0) return (-1);
  float value = xsGetUnitAttribute(id, attr, -1);
  if (value == -1.0) return (-1);
  if (value == expected) return (1);
  return (0);
}

int A2K_CombineTreatmentState(int current = 1, int next = 1) {
  if (current == -1) return (-1);
  if (next == -1) return (-1);
  if (current == 0) return (0);
  if (next == 0) return (0);
  return (1);
}

void A2K_WriteBarracks(int id = -1) {
  if (id < 0) return;
  vector pos = xsGetUnitPosition(id);
  xsWriteString("unit");
  xsWriteInt(id);
  xsWriteInt(xsGetUnitOwner(id));
  xsWriteFloat(xsVectorGetX(pos));
  xsWriteFloat(xsVectorGetY(pos));
  xsWriteFloat(xsGetUnitAttribute(id, 3, -1));
  xsWriteFloat(xsGetUnitAttribute(id, 4, -1));
  xsWriteFloat(xsGetUnitAttribute(id, 200, -1));
  xsWriteFloat(xsGetUnitAttribute(id, 201, -1));
}

void A2K_ReportP2(int collision = -1, int clearance = -1) {
  if (collision == 1) {
    if (clearance == -1) xsChatData("P2 collision APPLIED clearance UNKNOWN");
    if (clearance == 0) xsChatData("P2 collision APPLIED clearance NOT APPLIED");
    if (clearance == 1) xsChatData("P2 collision APPLIED clearance APPLIED");
  }
  if (collision == 0) xsChatData("P2 collision NOT APPLIED clearance UNKNOWN");
  if (collision == -1) xsChatData("P2 collision UNKNOWN clearance UNKNOWN");
}

void A2K_ReportP3(int collision = -1, int clearance = -1) {
  if (clearance == 1) {
    if (collision == -1) xsChatData("P3 clearance APPLIED collision UNKNOWN");
    if (collision == 0) xsChatData("P3 clearance APPLIED collision NOT APPLIED");
    if (collision == 1) xsChatData("P3 clearance APPLIED collision APPLIED");
  }
  if (clearance == 0) xsChatData("P3 clearance NOT APPLIED collision UNKNOWN");
  if (clearance == -1) xsChatData("P3 clearance UNKNOWN collision UNKNOWN");
}

void A2K_ReportP4(int clearance = -1) {
  if (clearance == 1) xsChatData("P4 clearance APPLIED collision UNKNOWN");
  if (clearance == 0) xsChatData("P4 clearance NOT APPLIED collision UNKNOWN");
  if (clearance == -1) xsChatData("P4 clearance UNKNOWN collision UNKNOWN");
}

void A2K_ReportWall(int player = -1, int count = 0, int collision = -1, int clearance = -1) {
  if (count <= 0) return;
  if (player == 1) xsChatData("P1 baseline editor wall");
  if (player == 2) A2K_ReportP2(collision, clearance);
  if (player == 3) A2K_ReportP3(collision, clearance);
  if (player == 4) A2K_ReportP4(clearance);
  if (player == 5) xsChatData("P5 unchanged control collision UNKNOWN clearance UNKNOWN");
}

void A2K_RyanScratchAB2Tick() {
  if (a2kDone) return;
  if (xsGetGameTime() < 4) return;
  bool opened = xsCreateFile(false);
  if (opened == false) return;
  xsWriteString("A2K_RYAN_SCRATCH_AB2");
  xsWriteInt(1);
  xsWriteString("clearance-variant");
  a2kScan = xsArrayCreateInt(64, -1, "ryan_scratch_ab2_scan");
  int p = 1;
  for (p = 1; <= 5) {
    int collision = 1;
    int clearance = 1;
    int count = 0;
    int ids = xsGetPlayerUnitIds(p, 12, a2kScan);
    int n = xsArrayGetSize(ids);
    int i = 0;
    for (i = 0; < n) {
      int id = xsArrayGetInt(ids, i);
      A2K_WriteBarracks(id);
      count = count + 1;
      if (p == 2) {
        collision = A2K_CombineTreatmentState(collision, A2K_TreatmentState(id, 3, 1.0));
        collision = A2K_CombineTreatmentState(collision, A2K_TreatmentState(id, 4, 1.0));
      }
      if (p == 3) {
        clearance = A2K_CombineTreatmentState(clearance, A2K_TreatmentState(id, 200, 0.5));
        clearance = A2K_CombineTreatmentState(clearance, A2K_TreatmentState(id, 201, 0.5));
      }
      if (p == 4) {
        clearance = A2K_CombineTreatmentState(clearance, A2K_TreatmentState(id, 200, 1.0));
        clearance = A2K_CombineTreatmentState(clearance, A2K_TreatmentState(id, 201, 1.0));
      }
    }
    A2K_ReportWall(p, count, collision, clearance);
  }
  xsWriteString("end");
  xsWriteInt(424242);
  xsCloseFile();
  a2kDone = true;
}

rule A2K_RyanScratchAB2Effect active minInterval 1 {
  A2K_ApplyP3Clearance();
}

rule A2K_RyanScratchAB2Telemetry active minInterval 1 {
  A2K_RyanScratchAB2Tick();
}
`
}

func hashBarracksWallFile(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
