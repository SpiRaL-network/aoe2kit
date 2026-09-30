// Package diagnosticfixtures contains small, one-actor engine probes.
package diagnosticfixtures

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"

	"aoe2kit/pkg/scenario"
	"aoe2kit/pkg/xs"
)

const (
	MapSize      = 120
	FireUnit     = 939
	FlowerUnit   = 1366
	BuildingUnit = 82
	ProbeTech    = 22 // Loom: valid stock tech, used only as a local-tech carrier.
)

type Report struct {
	ScenarioPath string `json:"scenario_path"`
	XSPath       string `json:"xs_path"`
	TypesPath    string `json:"types_path"`
	RecipePath   string `json:"recipe_path"`
	ScenarioSHA  string `json:"scenario_sha256"`
	XSSHA        string `json:"xs_sha256"`
}

// BuildHalo creates a fresh, one-active-player visual/retirement probe.
func BuildHalo(dir string, timestamp int) (*Report, error) {
	return build(dir, timestamp, "A2K Halo Timer", "halo", haloXS(), haloTypes(), haloRecipe)
}

// BuildLocalTechnology creates two identical P1 buildings and applies one local
// technology to exactly one of them. The XS sidecar records both unit IDs and
// the engine-visible attributes selected by the run; it does not infer success.
func BuildLocalTechnology(dir string, timestamp int) (*Report, error) {
	return build(dir, timestamp, "A2K Local Technology Probe", "local-tech", localTechXS(), localTechTypes(), localTechRecipe)
}

type recipeFunc func(xsPath string, timestamp int) scenario.Recipe

func build(dir string, timestamp int, name, stem, source, types string, makeRecipe recipeFunc) (*Report, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	base := filepath.Join(dir, "A2K_"+stem+"_BLANK_BASE.aoe2scenario")
	scen := filepath.Join(dir, name+".aoe2scenario")
	xsPath := filepath.Join(dir, "A2K_"+stem+".xs")
	typesPath := filepath.Join(dir, "A2K_"+stem+".types")
	recipePath := filepath.Join(dir, "A2K_"+stem+".recipe.json")
	if _, err := scenario.WriteBlankScenarioFile(base, scenario.BlankOptions{
		PlayerCount: 2, HumanSlots: 1, MapWidth: MapSize, MapHeight: MapSize,
		Timestamp: timestamp, ClearTriggers: true, DummyStarters: true, DummyUnit: 12,
		StartingResources: true, StartingFood: 100000, StartingWood: 100000,
		StartingGold: 100000, StartingStone: 100000, StartingTradeGoods: 100000,
	}); err != nil {
		return nil, err
	}
	if err := os.WriteFile(xsPath, []byte(source), 0644); err != nil {
		return nil, err
	}
	if err := xs.WriteTypesDocument(typesPath, types); err != nil {
		return nil, err
	}
	recipe := makeRecipe(xsPath, timestamp)
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
	scenHash, err := hashFile(scen)
	if err != nil {
		return nil, err
	}
	xsHash, err := hashFile(xsPath)
	if err != nil {
		return nil, err
	}
	return &Report{ScenarioPath: scen, XSPath: xsPath, TypesPath: typesPath, RecipePath: recipePath, ScenarioSHA: scenHash, XSSHA: xsHash}, nil
}

func haloRecipe(xsPath string, timestamp int) scenario.Recipe {
	return scenario.Recipe{
		Scenario: &scenario.ScenarioRecipe{PlayerCount: intPtr(2), TimestampOfLastSave: &timestamp, StartingAge: intPtr(6)},
		Players:  []scenario.PlayerRecipe{{Player: 2, Active: boolPtr(false), Human: boolPtr(false)}},
		XS:       &scenario.XSRecipe{Mode: "inline_runtime", Name: filepath.Base(xsPath), ContentFile: xsPath, CarrierTitle: "XS string", CarrierTriggerName: "A2K Halo Timer XS Runtime Source"},
		Triggers: []scenario.TriggerRecipe{{Op: "add_trigger", Name: "A2K Halo Timer", Enabled: boolPtr(true), Looping: boolPtr(true), Conditions: []scenario.ConditionRecipe{{Op: "timer", Timer: intPtr(1)}}, Effects: []scenario.EffectRecipe{{Op: "script_call", SourcePlayer: intPtr(1), Message: "A2K_HaloTick();"}}}},
	}
}

func localTechRecipe(xsPath string, timestamp int) scenario.Recipe {
	left, right := 930001, 930002
	return scenario.Recipe{
		Scenario: &scenario.ScenarioRecipe{PlayerCount: intPtr(2), TimestampOfLastSave: &timestamp, StartingAge: intPtr(6)},
		Players:  []scenario.PlayerRecipe{{Player: 2, Active: boolPtr(false), Human: boolPtr(false)}},
		XS:       &scenario.XSRecipe{Mode: "inline_runtime", Name: filepath.Base(xsPath), ContentFile: xsPath, CarrierTitle: "XS string", CarrierTriggerName: "A2K Local Technology Probe XS Runtime Source"},
		Units: []scenario.UnitRecipe{
			{Op: "add_unit", Player: 1, UnitConst: BuildingUnit, X: floatPtr(40.5), Y: floatPtr(40.5), Status: intPtr(2), ReferenceID: &left, CaptionString: "A2K LOCAL TECH TARGET"},
			{Op: "add_unit", Player: 1, UnitConst: BuildingUnit, X: floatPtr(60.5), Y: floatPtr(40.5), Status: intPtr(2), ReferenceID: &right, CaptionString: "A2K LOCAL TECH CONTROL"},
		},
		Triggers: []scenario.TriggerRecipe{
			{Op: "add_trigger", Name: "A2K Apply Local Technology", Enabled: boolPtr(true), Looping: boolPtr(false), Conditions: []scenario.ConditionRecipe{{Op: "timer", Timer: intPtr(3)}}, Effects: []scenario.EffectRecipe{{Op: "research_local_technology", SourcePlayer: intPtr(1), Technology: intPtr(ProbeTech), LocalTechnology: intPtr(1), SelectedObjectIDs: []int{left}}}},
			{Op: "add_trigger", Name: "A2K Local Technology Telemetry", Enabled: boolPtr(true), Looping: boolPtr(true), Conditions: []scenario.ConditionRecipe{{Op: "timer", Timer: intPtr(1)}}, Effects: []scenario.EffectRecipe{{Op: "script_call", SourcePlayer: intPtr(1), Message: "A2K_LocalTechTick();"}}},
		},
	}
}

func haloXS() string {
	return `const int A2K_FIRE = 939;
const int A2K_FLOWER = 1366;
int a2kFire = -1; int a2kFlower = -1; int a2kBorn = -1; bool a2kOpen = false; bool a2kDone = false;
void A2K_Write(string tag = "", int fire = 0, int flower = 0, int moved = 0, int fireAlive = 0, int flowerAlive = 0) {
  if (a2kOpen) { if (xsCreateFile(true)) { xsWriteString(tag); xsWriteInt(xsGetGameTime()); xsWriteInt(fire); xsWriteInt(flower); xsWriteInt(moved); xsWriteInt(fireAlive); xsWriteInt(flowerAlive); xsCloseFile(); } }
}
void A2K_HaloTick() {
  int now = xsGetGameTime();
  if (a2kBorn < 0) {
    a2kOpen = xsCreateFile(false); if (a2kOpen) { xsWriteString("A2K_HALO_TIMER"); xsWriteInt(1); xsWriteInt(12); xsWriteInt(12); xsCloseFile(); }
    a2kFire = xsCreateUnit(A2K_FIRE, 0, xsVectorSet(45.5, 55.5, 0.0), false, false, false);
    a2kFlower = xsCreateUnit(A2K_FLOWER, 0, xsVectorSet(75.5, 55.5, 0.0), false, false, false);
    a2kBorn = now; A2K_Write("before", 1, 1, 0, xsDoesUnitExist(a2kFire), xsDoesUnitExist(a2kFlower));
  } else {
    int retireAt = a2kBorn + 15;
    if (a2kDone == false && now >= retireAt) {
      int fireRetired = xsSetUnitHitpoints(a2kFire, 0.0);
      int moved = xsSetUnitPosition(a2kFlower, xsVectorSet(112.5, 112.5, 0.0), false);
      A2K_Write("after", 0, 0, moved, xsDoesUnitExist(a2kFire), xsDoesUnitExist(a2kFlower));
      if (a2kOpen) { if (xsCreateFile(true)) { xsWriteString("end"); xsWriteInt(2); xsWriteInt(fireRetired); xsWriteInt(moved); xsWriteInt(424242); xsCloseFile(); } }
      a2kDone = true;
    }
  }
}
void main() { }
rule a2kHaloRule active minInterval 1 maxInterval 1 { A2K_HaloTick(); }
`
}

func localTechXS() string {
	return `bool a2kOpen = false; bool a2kDone = false; int a2kLeft = -1; int a2kRight = -1;
void A2K_LocalTechTick() {
  if (a2kDone == false) {
    if (a2kOpen == false) { a2kOpen = xsCreateFile(false); if (a2kOpen) { xsWriteString("A2K_LOCAL_TECH"); xsWriteInt(1); xsWriteInt(930001); xsWriteInt(930002); xsWriteInt(22); xsWriteInt(1); xsCloseFile(); } }
    int ids = xsGetPlayerUnitIds(1, 82, -1); int n = xsArrayGetSize(ids);
    if (n >= 2) { a2kLeft = xsArrayGetInt(ids, 0); a2kRight = xsArrayGetInt(ids, 1); }
    if (a2kLeft != -1 && a2kOpen) { if (xsCreateFile(true)) { xsWriteString("state"); xsWriteInt(xsGetGameTime()); xsWriteInt(a2kLeft); xsWriteInt(a2kRight); xsWriteFloat(xsGetUnitAttribute(a2kLeft, 0, -1)); xsWriteFloat(xsGetUnitAttribute(a2kRight, 0, -1)); xsCloseFile(); a2kDone = true; } }
  }
}
void main() { }
rule a2kLocalTechRule active minInterval 1 maxInterval 1 { A2K_LocalTechTick(); }
`
}

func haloTypes() string {
	return "header = string,int,int,int\nstate = string,int,int,int,int,int,int\nend = string,int,int,int,int\nfooter_integrity = 424242\n"
}
func localTechTypes() string {
	return "header = string,int,int,int,int,int\nstate = string,int,int,int,float,float\n"
}
func intPtr(v int) *int           { return &v }
func boolPtr(v bool) *bool        { return &v }
func floatPtr(v float64) *float64 { return &v }
func hashFile(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:]), nil
}
