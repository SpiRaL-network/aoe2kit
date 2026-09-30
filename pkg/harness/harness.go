// Package harness generates unattended XS engine-fact probes.
package harness

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
	BatchSafe      = "safe"
	BatchSixArm    = "six-arm"
	BatchTrigLong  = "trig-long"
	BatchAttrSweep = "attr-sweep"
	BatchLongV2    = "long-v2"
	BatchLongV3    = "long-v3"
	MapSize        = 120
)

type Fact struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	Category    string `json:"category"`
	Setup       string `json:"setup"`
	Operation   string `json:"operation"`
	Observed    *int   `json:"observed,omitempty"`
	Expected    *int   `json:"expected,omitempty"`
	Risk        string `json:"risk"`
	Evidence    string `json:"evidence"`
}

type Options struct {
	OutputDir           string
	ScenarioName        string
	Timestamp           int
	Batch               string
	XSCheckPath         string
	RequireXSCheck      bool
	AllowMissingXSCheck bool
}

type Report struct {
	OutputDir             string                   `json:"output_dir"`
	ScenarioPath          string                   `json:"scenario_path"`
	XSPath                string                   `json:"xs_path"`
	TypesPath             string                   `json:"types_path"`
	ManifestPath          string                   `json:"manifest_path"`
	RecipePath            string                   `json:"recipe_path"`
	ScenarioSHA           string                   `json:"scenario_sha256"`
	XSSHA                 string                   `json:"xs_sha256"`
	Batch                 string                   `json:"batch"`
	FactCount             int                      `json:"fact_count"`
	ExpectedRecords       int                      `json:"expected_records,omitempty"`
	EstimatedSidecarBytes int                      `json:"estimated_sidecar_bytes,omitempty"`
	EstimatedDurationS    int                      `json:"estimated_duration_s,omitempty"`
	SidecarByteCap        int                      `json:"sidecar_byte_cap,omitempty"`
	ExecutionCarrier      string                   `json:"execution_carrier"`
	XSParseCheck          string                   `json:"xs_parse_check"`
	XSParseChecker        string                   `json:"xs_parse_checker,omitempty"`
	XSParseOutput         string                   `json:"xs_parse_output,omitempty"`
	KitLintFindings       []xs.SourceFinding       `json:"kit_lint_findings,omitempty"`
	XSCheckFindings       []xs.ExternalFinding     `json:"xscheck_findings,omitempty"`
	Controversies         []xs.DifferentialFinding `json:"controversies,omitempty"`
	DifferentialStatus    string                   `json:"differential_status"`
}

const (
	executionCarrierAttribute = 0
	executionCarrierBase      = 100000
)

var safeFacts = []Fact{
	{ID: "string_case_insensitive_equal", Description: `"A" == "a"`, Category: "strings", Setup: "No engine objects; compare two ASCII literals.", Operation: "Evaluate the equality expression.", Expected: intPtr(1), Risk: "safe", Evidence: "observed in A2 atoms probe; rerun in this harness"},
	{ID: "array_int_round_trip", Description: "create, set, and read an integer array cell", Category: "arrays", Setup: "Create a four-cell integer array.", Operation: "Set index 2 to 37, then read index 2.", Expected: intPtr(37), Risk: "safe", Evidence: "structure and runtime behavior to be measured"},
	{ID: "vector_x_round_trip", Description: "create a vector and read its X component", Category: "vectors", Setup: "Create vector (20.5, 1.0, 0.0).", Operation: "Read X and compare it with 20.5.", Expected: intPtr(1), Risk: "safe", Evidence: "structure and runtime behavior to be measured"},
	{ID: "unit_position_round_trip", Description: "create and move footprintless Gaia art in bounds", Category: "placement", Setup: "Create Gaia Flower 1366 at an in-bounds fractional position.", Operation: "Move it to (22.5, 22.5) and compare the reported X position.", Expected: intPtr(1), Risk: "safe", Evidence: "fractional Flower placement is engine-verified; this round-trip is a harness probe"},
	{ID: "unit_exists_before_retire", Description: "new Gaia art exists before retirement", Category: "lifecycle", Setup: "Create Gaia Flower 1366.", Operation: "Read xsDoesUnitExist before any retirement operation.", Expected: intPtr(1), Risk: "safe", Evidence: "structure and runtime behavior to be measured"},
	{ID: "unit_exists_after_hp_zero", Description: "FLAME1 existence after HP zero", Category: "retirement", Setup: "Create Gaia Flame1 939.", Operation: "Set hit points to zero, then read xsDoesUnitExist.", Expected: intPtr(0), Risk: "safe", Evidence: "Flame retirement is engine-verified; rerun in this harness"},
	{ID: "bounded_loop_count", Description: "for loop executes exactly ten iterations", Category: "cadence", Setup: "Initialize a local counter to zero.", Operation: "Run a ten-iteration XS for loop and return the counter.", Expected: intPtr(10), Risk: "safe", Evidence: "structure and runtime behavior to be measured"},
}

func Facts(batch string) []Fact {
	if batch == BatchTrigLong {
		return append([]Fact(nil), trigLongFacts...)
	}
	if batch == BatchAttrSweep {
		return append([]Fact(nil), attrSweepFacts...)
	}
	if batch == BatchLongV2 {
		return longV2FactsFor()
	}
	if batch == BatchLongV3 {
		return longV3FactsFor()
	}
	if batch == BatchSixArm {
		return []Fact{
			{ID: "arm_a_single_append", Description: "counter plus one append per tick", Category: "six_arm", Setup: "Independent rule A increments its own counter.", Operation: "Append one record per tick.", Risk: "safe", Evidence: "control arm; engine measurement required"},
			{ID: "arm_b_local_float", Description: "counter plus local float declaration", Category: "six_arm", Setup: "Independent rule B declares a local float.", Operation: "Append one record per tick after the declaration.", Risk: "safe", Evidence: "syntax/runtime arm; engine measurement required"},
			{ID: "arm_c_player_attribute", Description: "counter plus player resource write", Category: "six_arm", Setup: "Independent rule C writes player food.", Operation: "Write the carrier attribute and append one record per tick.", Risk: "safe", Evidence: "carrier arm; engine measurement required"},
			{ID: "arm_d_double_append", Description: "counter plus two appends per tick", Category: "six_arm", Setup: "Independent rule D performs two file appends.", Operation: "Append two records per tick.", Risk: "safe", Evidence: "append multiplicity arm; engine measurement required"},
			{ID: "arm_e_call_argument", Description: "counter plus function-call argument", Category: "six_arm", Setup: "Independent rule E passes a function result to the writer.", Operation: "Append one record with a call-result argument per tick.", Risk: "safe", Evidence: "argument evaluation arm; engine measurement required"},
			{ID: "arm_f_no_file_control", Description: "counter with no file write", Category: "six_arm", Setup: "Independent rule F increments only.", Operation: "Publish the counter through the food carrier and never touch the file.", Risk: "safe", Evidence: "execution control arm; engine measurement required"},
		}
	}
	if batch == "" || batch == BatchSafe {
		return append([]Fact(nil), safeFacts...)
	}
	return nil
}

func Build(opts Options) (*Report, error) {
	if opts.OutputDir == "" {
		return nil, fmt.Errorf("harness output directory is required")
	}
	if opts.Batch == "" {
		opts.Batch = BatchSafe
	}
	facts := Facts(opts.Batch)
	if len(facts) == 0 {
		return nil, fmt.Errorf("unknown harness batch %q", opts.Batch)
	}
	if opts.ScenarioName == "" {
		opts.ScenarioName = fmt.Sprintf("A2K Engine Harness %s", opts.Batch)
	}
	if err := os.MkdirAll(opts.OutputDir, 0755); err != nil {
		return nil, err
	}
	base := filepath.Join(opts.OutputDir, "A2K_ENGINE_HARNESS_BLANK_BASE.aoe2scenario")
	scen := filepath.Join(opts.OutputDir, opts.ScenarioName+".aoe2scenario")
	xsPath := filepath.Join(opts.OutputDir, "A2K_ENGINE_HARNESS_"+opts.Batch+".xs")
	typesPath := filepath.Join(opts.OutputDir, "A2K_ENGINE_HARNESS_"+opts.Batch+".types")
	manifestPath := filepath.Join(opts.OutputDir, "A2K_ENGINE_HARNESS_"+opts.Batch+".manifest.json")
	recipePath := filepath.Join(opts.OutputDir, "A2K_ENGINE_HARNESS_"+opts.Batch+".recipe.json")
	if _, err := scenario.WriteBlankScenarioFile(base, scenario.BlankOptions{
		PlayerCount: 2, HumanSlots: 1, MapWidth: MapSize, MapHeight: MapSize,
		Timestamp: opts.Timestamp, ClearTriggers: true, DummyStarters: true,
		DummyUnit: 12, StartingResources: true, StartingFood: 100000, StartingWood: 100000,
		StartingGold: 100000, StartingStone: 100000, StartingTradeGoods: 100000,
	}); err != nil {
		return nil, err
	}
	module := xsModule(facts)
	if opts.Batch == BatchSixArm {
		module = sixArmXS()
	}
	if opts.Batch == BatchTrigLong {
		module = trigLongXS()
	}
	if opts.Batch == BatchAttrSweep {
		module = attrSweepXS()
	}
	if opts.Batch == BatchLongV2 || opts.Batch == BatchLongV3 {
		module = longV2XS()
	}
	if opts.Batch == BatchLongV3 {
		module = longV3XS()
	}
	if err := os.WriteFile(xsPath, []byte(module), 0644); err != nil {
		return nil, err
	}
	requireXSCheck := opts.RequireXSCheck || !opts.AllowMissingXSCheck
	kitFindings := xs.LintSource(xsPath, module)
	// The carrier wraps the generated module in three XS lines before the
	// engine compiles it. Keep the source line for local editing and expose the
	// corresponding game line for screenshots and load-error reports.
	for i := range kitFindings {
		kitFindings[i].GameLine = kitFindings[i].Line + 3
	}
	parseReport, err := checkGeneratedXS(xsPath, opts.XSCheckPath, requireXSCheck)
	if err != nil {
		return nil, err
	}
	externalFindings := xs.ExternalFindings(parseReport)
	controversies := []xs.DifferentialFinding(nil)
	differentialStatus := "unavailable"
	if parseReport.Status == "passed" {
		differentialStatus = "clean"
		for _, finding := range xs.CompareFindings(kitFindings, externalFindings) {
			if finding.Class != "BOTH" {
				controversies = append(controversies, finding)
			}
		}
		if len(controversies) > 0 {
			differentialStatus = "controversy"
		}
	}
	if err := xs.WriteTypesDocument(typesPath, sidecarTypesFor(opts.Batch)); err != nil {
		return nil, err
	}
	manifest := map[string]any{"schema": "a2k-engine-harness-v1", "batch": opts.Batch, "facts": facts, "deliberate_crash_tests": false}
	manifestData, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(manifestPath, append(manifestData, '\n'), 0644); err != nil {
		return nil, err
	}
	startAge := 6
	if opts.Batch == BatchLongV2 || opts.Batch == BatchLongV3 {
		// Loom is already active post-Imperial. Dark Age makes the local-tech
		// arm a real before/after experiment instead of a null result.
		startAge = 0
	}
	recipe := scenario.Recipe{
		Scenario: &scenario.ScenarioRecipe{PlayerCount: intPtr(2), TimestampOfLastSave: &opts.Timestamp, StartingAge: &startAge},
		// Keep the two-slot structural baseline, but deactivate P2. The
		// unattended harness must have one actor; an AI would pollute action and
		// resource observations without adding evidence.
		Players:  []scenario.PlayerRecipe{{Player: 2, Active: boolPtr(false), Human: boolPtr(false)}},
		XS:       &scenario.XSRecipe{Mode: "inline_runtime", Name: filepath.Base(xsPath), ContentFile: xsPath, CarrierTitle: "XS string", CarrierTriggerName: "A2K Engine Harness XS Runtime Source"},
		Triggers: []scenario.TriggerRecipe{{Op: "add_trigger", Name: "A2K Engine Harness Poll", Enabled: boolPtr(true), Looping: boolPtr(true), Conditions: []scenario.ConditionRecipe{{Op: "timer", Timer: intPtr(1)}}, Effects: []scenario.EffectRecipe{{Op: "script_call", SourcePlayer: intPtr(1), Message: "A2K_Harness_Tick();"}}}},
	}
	if opts.Batch == BatchTrigLong {
		recipe.Triggers = trigLongTriggers()
		recipe.Units = trigLongUnits()
	}
	if opts.Batch == BatchAttrSweep {
		recipe.Triggers = attrSweepTriggers()
		recipe.Units = attrSweepUnits()
	}
	if opts.Batch == BatchLongV2 {
		recipe.Triggers = longV2Triggers()
		recipe.Units = longV2Units()
	}
	if opts.Batch == BatchLongV3 {
		recipe.Triggers = longV3Triggers()
		recipe.Units = longV3Units()
		recipe.Players = []scenario.PlayerRecipe{
			{Player: 1, Active: boolPtr(true), Human: boolPtr(true)},
			{Player: 2, Active: boolPtr(false), Human: boolPtr(false)},
		}
	}
	recipeData, err := json.MarshalIndent(recipe, "", "  ")
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(recipePath, append(recipeData, '\n'), 0644); err != nil {
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
	expectedRecords, sidecarByteCap := 0, 0
	if opts.Batch == BatchTrigLong {
		// Header + one before/result pair per fact + 79 self-describing value
		// rows (raw-float and string probes) + end footer. The engine cap is
		// 2^20-1.
		expectedRecords = 1 + len(facts)*2 + 79 + 1
		sidecarByteCap = (1 << 20) - 1
	}
	if opts.Batch == BatchAttrSweep {
		// Header + before/value/result per attribute + end footer.
		expectedRecords = 1 + len(facts)*3 + 1
		sidecarByteCap = (1 << 20) - 1
	}
	if opts.Batch == BatchLongV2 {
		expectedRecords = longV2ExpectedRecords()
		sidecarByteCap = longV3ByteCap
	}
	if opts.Batch == BatchLongV3 {
		expectedRecords = longV3ExpectedRecords()
		sidecarByteCap = longV3ByteCap
	}
	estimatedDuration := 0
	estimatedSidecarBytes := 0
	if opts.Batch == BatchLongV2 {
		estimatedDuration = longV2EstimatedDurationS()
	}
	if opts.Batch == BatchLongV3 {
		estimatedDuration = longV3EstimatedDurationS()
		estimatedSidecarBytes = longV3ExpectedRecords() * 64
	}
	report := &Report{OutputDir: opts.OutputDir, ScenarioPath: scen, XSPath: xsPath, TypesPath: typesPath, ManifestPath: manifestPath, RecipePath: recipePath, ScenarioSHA: scenHash, XSSHA: xsHash, Batch: opts.Batch, FactCount: len(facts), ExpectedRecords: expectedRecords, EstimatedSidecarBytes: estimatedSidecarBytes, EstimatedDurationS: estimatedDuration, SidecarByteCap: sidecarByteCap, ExecutionCarrier: fmt.Sprintf("P1 resource %d base %d", executionCarrierAttribute, executionCarrierBase), XSParseCheck: parseReport.Status, XSParseChecker: parseReport.Checker, XSParseOutput: parseReport.Output, KitLintFindings: kitFindings, XSCheckFindings: externalFindings, Controversies: controversies, DifferentialStatus: differentialStatus}
	if len(controversies) > 0 && !opts.AllowMissingXSCheck {
		return report, fmt.Errorf("XS checker controversy: %d disagreement(s)", len(controversies))
	}
	return report, nil
}

func checkGeneratedXS(path, explicit string, required bool) (xs.ParseCheckReport, error) {
	checker, err := xs.ResolveParser(explicit)
	if err != nil {
		return xs.ParseCheckReport{Checker: checker, Status: "failed"}, err
	}
	if checker == "" {
		if required {
			checker, err = xs.RequireParser(explicit)
			return xs.ParseCheckReport{Checker: checker, Status: "unavailable"}, err
		}
		return xs.ParseCheckReport{Status: "unavailable"}, nil
	}
	report, err := xs.CheckFile(path, checker)
	if err != nil {
		return report, err
	}
	return report, nil
}

func xsModule(facts []Fact) string {
	var cases string
	var names string
	for i, fact := range facts {
		names += fmt.Sprintf("  if (a2kFact == %d) { return (\"%s\"); }\n", i, fact.ID)
		if fact.Expected != nil {
			cases += fmt.Sprintf("  if (a2kFact == %d) { a2kExpected = %d; a2kObserved = a2kObserve%d(); }\n", i, *fact.Expected, i)
		}
	}
	return fmt.Sprintf(`// A2K unattended engine-fact harness: safe batch.
const int A2K_FACTS = %d;
int a2kFact = 0; int a2kArray = -1; int a2kArt = -1; int a2kFlame = -1; int a2kLoop = 0; int a2kLastTick = -1;

int a2kObserve0() { return ("A" == "a"); }
int a2kObserve1() { if (a2kArray < 0) { a2kArray = xsArrayCreateInt(4, 0, "harness_array"); xsArraySetInt(a2kArray, 2, 37); } return (xsArrayGetInt(a2kArray, 2)); }
int a2kObserve2() { vector v = xsVectorSet(20.5, 1.0, 0.0); return (xsVectorGetX(v) == 20.5); }
int a2kObserve3() { if (a2kArt < 0) { a2kArt = xsCreateUnit(1366, 0, xsVectorSet(22.5, 22.5, 0.0), false, false, false); } xsSetUnitPosition(a2kArt, xsVectorSet(22.5, 22.5, 0.0), false); return (xsVectorGetX(xsGetUnitPosition(a2kArt)) == 22.5); }
int a2kObserve4() { if (a2kArt < 0) { a2kArt = xsCreateUnit(1366, 0, xsVectorSet(24.5, 24.5, 0.0), false, false, false); } return (xsDoesUnitExist(a2kArt)); }
int a2kObserve5() { if (a2kFlame < 0) { a2kFlame = xsCreateUnit(939, 0, xsVectorSet(26.5, 26.5, 0.0), false, false, false); xsSetUnitHitpoints(a2kFlame, 0.0); } return (xsDoesUnitExist(a2kFlame)); }
int a2kObserve6() { int n = 0; for (i = 0; < 10) { n = n + 1; } return (n); }

string a2kFactName() {
%s  return ("");
}
void a2kAppendMarker(string tag = "", string id = "", int ordinal = 0, int expected = 0, int observed = 0, int pass = 0) {
  if (xsCreateFile(true)) { xsWriteString(tag); xsWriteString("safe"); xsWriteString(id); xsWriteInt(ordinal); xsWriteInt(expected); xsWriteInt(observed); xsWriteInt(pass); xsWriteInt(xsGetGameTime()); xsCloseFile(); }
}
void A2K_Harness_Tick() {
  if (a2kFact >= A2K_FACTS) { return; }
  if (xsGetGameTime() == a2kLastTick) { return; }
	 a2kLastTick = xsGetGameTime();
	 a2kLoop = a2kLoop + 1;
	 float a2kCarrier = %d.0;
	 a2kCarrier = a2kCarrier + a2kLoop;
	 xsSetPlayerAttribute(1, %d, a2kCarrier);
  int a2kExpected = 0; int a2kObserved = 0;
  a2kAppendMarker("before", a2kFactName(), a2kFact, 0, 0, 0);
%s
  a2kAppendMarker("result", a2kFactName(), a2kFact, a2kExpected, a2kObserved, a2kObserved == a2kExpected);
  a2kFact = a2kFact + 1;
  if (a2kFact >= A2K_FACTS) { if (xsCreateFile(true)) { xsWriteString("end"); xsWriteInt(A2K_FACTS); xsWriteInt(424242); xsCloseFile(); } }
}
void main() { if (xsCreateFile(false)) { xsWriteString("A2K_ENGINE_HARNESS"); xsWriteInt(1); xsWriteString("safe"); xsWriteInt(A2K_FACTS); xsCloseFile(); } }
rule a2kHarnessTick
 active
 minInterval 1
 maxInterval 1
{ A2K_Harness_Tick(); }
`, len(facts), names, executionCarrierBase, executionCarrierAttribute, cases)
}

func sidecarTypes() string {
	return sidecarTypesFor(BatchSafe)
}

func sidecarTypesFor(batch string) string {
	if batch == BatchTrigLong {
		return "# A2K_ENGINE_HARNESS_trig-long.xsdat\nheader = string,int,string,int\nbefore = string,string,int,int\nresult = string,string,int,int,int,int,int\nvalue = string,string,string,float,int\nend = string,int,int\nfooter_integrity = 424242\n"
	}
	if batch == BatchAttrSweep {
		return "# A2K_ENGINE_HARNESS_attr-sweep.xsdat\nheader = string,int,string,int\nbefore = string,string,int,int\nresult = string,string,int,int,int,int,int\nvalue = string,string,string,float,int\nend = string,int,int\nfooter_integrity = 424242\n"
	}
	if batch == BatchLongV2 {
		return "# A2K_ENGINE_HARNESS_long-v2.xsdat\nheader = string,int,string,int\nbefore = string,string,int,int\nresult = string,string,int,int,int,int,int\nvalue = string,string,string,float,int\nrate = string,int,int,int,int\nend = string,int,int\nfooter_integrity = 424242\n"
	}
	if batch == BatchLongV3 {
		return "# A2K_ENGINE_HARNESS_long-v3.xsdat\nheader = string,int,string,int,int\nunit = string,int,int,int,float,int\ntreatment = string,int,int,int,float,float,int\ntrail = string,int,int,int,float,float,int\ncombat = string,int,int,float,float,int,int,int\npassenger = string,int,int,float,float,int,int\nz = string,int,int,int,float,float,int,int\nrate = string,int,int,int,int\nmilestone = string,int\nend = string,int,int,int\nfooter_integrity = 424242\n"
	}
	return "# A2K_ENGINE_HARNESS.xsdat\nheader = string,int,string,int\nmarker = string,string,string,int,int,int,int,int\nend = string,int,int\nfooter_integrity = 424242\n"
}

func sixArmXS() string {
	return `// A2K six-arm unattended harness. Each rule is intentionally independent.
const int A2K_ARM_COUNT = 6;
int a2kA = 0; int a2kB = 0; int a2kC = 0; int a2kD = 0; int a2kE = 0; int a2kF = 0;
bool a2kAClosed = false; bool a2kBClosed = false; bool a2kCClosed = false; bool a2kDClosed = false; bool a2kEClosed = false; bool a2kFClosed = false;

void A2K_ArmATick() {
  int now = xsGetGameTime(); a2kA = a2kA + 1;
  if (xsCreateFile(true)) { xsWriteString("tick"); xsWriteString("A"); xsWriteInt(now); xsWriteInt(a2kA); xsCloseFile(); }
  if (now >= 60 && a2kAClosed == false) { if (xsCreateFile(true)) { xsWriteString("end"); xsWriteString("A"); xsWriteInt(a2kA); xsWriteInt(424242); xsCloseFile(); } a2kAClosed = true; }
}
void A2K_ArmBTick() {
  int now = xsGetGameTime(); float localValue = 1.0; localValue = localValue + now; a2kB = a2kB + 1;
  if (xsCreateFile(true)) { xsWriteString("tick"); xsWriteString("B"); xsWriteInt(now); xsWriteInt(a2kB); xsCloseFile(); }
  if (now >= 60 && a2kBClosed == false) { a2kBClosed = true; }
}
void A2K_ArmCTick() {
  int now = xsGetGameTime(); a2kC = a2kC + 1; float carrier = 100000.0 + a2kC;
  xsSetPlayerAttribute(1, 0, carrier);
  if (xsCreateFile(true)) { xsWriteString("tick"); xsWriteString("C"); xsWriteInt(now); xsWriteInt(a2kC); xsCloseFile(); }
  if (now >= 60 && a2kCClosed == false) { a2kCClosed = true; }
}
void A2K_ArmDTick() {
  int now = xsGetGameTime(); a2kD = a2kD + 1;
  if (xsCreateFile(true)) { xsWriteString("tick"); xsWriteString("D1"); xsWriteInt(now); xsWriteInt(a2kD); xsCloseFile(); }
  if (xsCreateFile(true)) { xsWriteString("tick"); xsWriteString("D2"); xsWriteInt(now); xsWriteInt(a2kD); xsCloseFile(); }
  if (now >= 60 && a2kDClosed == false) { a2kDClosed = true; }
}
void A2K_ArmETick() {
  int now = xsGetGameTime(); a2kE = a2kE + 1;
  if (xsCreateFile(true)) { xsWriteString("tick"); xsWriteString("E"); xsWriteInt(xsGetGameTime()); xsWriteInt(a2kE); xsCloseFile(); }
  if (now >= 60 && a2kEClosed == false) { a2kEClosed = true; }
}
void A2K_ArmFTick() {
  int now = xsGetGameTime(); a2kF = a2kF + 1; float carrier = 100000.0 + a2kF;
  xsSetPlayerAttribute(1, 0, carrier);
  if (now >= 60 && a2kFClosed == false) { a2kFClosed = true; }
}
void main() { if (xsCreateFile(false)) { xsWriteString("A2K_ENGINE_HARNESS"); xsWriteInt(1); xsWriteString("six-arm"); xsWriteInt(A2K_ARM_COUNT); xsCloseFile(); } }
rule a2kArmA active minInterval 1 maxInterval 1 { A2K_ArmATick(); }
rule a2kArmB active minInterval 1 maxInterval 1 { A2K_ArmBTick(); }
rule a2kArmC active minInterval 1 maxInterval 1 { A2K_ArmCTick(); }
rule a2kArmD active minInterval 1 maxInterval 1 { A2K_ArmDTick(); }
rule a2kArmE active minInterval 1 maxInterval 1 { A2K_ArmETick(); }
rule a2kArmF active minInterval 1 maxInterval 1 { A2K_ArmFTick(); }
`
}
func intPtr(v int) *int    { return &v }
func boolPtr(v bool) *bool { return &v }
func hashFile(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}
