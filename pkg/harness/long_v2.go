package harness

import (
	"fmt"
	"strings"

	"aoe2kit/pkg/scenario"
)

const longV2AttributeCount = 102

func longV2AttributeIDs() []int {
	ids := make([]int, 0, longV2AttributeCount)
	for attr := 19; attr < 222; attr += 2 {
		ids = append(ids, attr)
	}
	return ids
}

func longV2FactsFor() []Fact {
	var facts []Fact
	for _, attr := range longV2AttributeIDs() {
		facts = append(facts, Fact{
			ID:          fmt.Sprintf("unit_attribute_%d", attr),
			Description: fmt.Sprintf("write/read unit and type attribute %d", attr),
			Category:    "unit_attributes",
			Setup:       "Create fresh villager and control villager subjects.",
			Operation:   "Read defaults, write unit values, then write the type through a trigger and read both subjects.",
			Risk:        "safe; hit-point extremes are skipped",
			Evidence:    "long write-then-read engine probe; verification pending",
		})
	}
	facts = append(facts,
		Fact{ID: "local_tech_villager_base", Description: "villager base hit points before local Loom", Category: "local_technology", Operation: "Read the researched and control villager before Loom.", Risk: "safe", Evidence: "local technology instance-vs-type probe"},
		Fact{ID: "local_tech_villager_researched", Description: "villager hit points after local Loom", Category: "local_technology", Operation: "Research Loom on one villager and read both villagers.", Risk: "safe", Evidence: "local technology instance-vs-type probe"},
		Fact{ID: "local_tech_control", Description: "control villager after local Loom", Category: "local_technology", Operation: "Read the unresearched villager after the first villager researches Loom.", Risk: "safe", Evidence: "local technology instance-vs-type probe"},
		Fact{ID: "xs_logical_short_circuit", Description: "whether XS short-circuits && and || operands", Category: "syntax_runtime", Setup: "Initialize a side-effect counter before any file-dependent probe.", Operation: "Evaluate false && side_effect() and true || side_effect(), then record the counter.", Risk: "safe; no file access", Evidence: "direct runtime probe; verification pending"},
	)
	for _, fact := range trigLongFacts {
		fact.ID = "recheck_" + fact.ID
		fact.Category = "trig_recheck"
		fact.Evidence = "trig-long fixed-encoder recheck"
		facts = append(facts, fact)
	}
	return facts
}

func longV2ExpectedRecords() int {
	// Header/footer, attribute blocks, local-tech block, the complete
	// trig-long record budget, eight sparse milestone rows, one short-circuit
	// value row, and one rate row
	// per planned game second. Timer conditions are relative to activation, so
	// every phase uses one second and the duration is linear.
	return 1 + longV2AttributeCount*12 + 10 + (1 + len(trigLongFacts)*2 + 79 + 1) + 8 + 1 + 1 + longV2EstimatedDurationS()
}

func longV2EstimatedDurationS() int {
	return longV2AttributeCount + 1 + len(trigLongFacts)
}

func longV2XS() string {
	facts := longV2FactsFor()
	var attrCases strings.Builder
	var attrSteps strings.Builder
	for ordinal, attr := range longV2AttributeIDs() {
		fmt.Fprintf(&attrCases, "void A2K_Attr_Begin_%d() { A2K_AttrBegin(%d, %d); }\n", attr, attr, ordinal)
		fmt.Fprintf(&attrCases, "void A2K_Attr_Write_%d() { A2K_AttrWrite(%d, %d); }\n", attr, attr, ordinal)
		fmt.Fprintf(&attrCases, "void A2K_Attr_End_%d() { A2K_AttrEnd(%d, %d); }\n", attr, attr, ordinal)
		fmt.Fprintf(&attrSteps, "  if (index == %d) { if (phase == 0) { A2K_Attr_Begin_%d(); } else if (phase == 1) { A2K_Attr_Write_%d(); } else { A2K_Attr_End_%d(); a2kWriteResult(\"unit_attribute_%d\",%d); } }\n", ordinal, attr, attr, attr, attr, attr)
	}
	fmt.Fprintf(&attrCases, "void A2K_Local_Write_0() { a2kWriteResult(\"local_tech_villager_base\",%d); }\n", longV2AttributeCount)
	fmt.Fprintf(&attrCases, "void A2K_Local_Write_1() { a2kWriteResult(\"local_tech_villager_researched\",%d); }\n", longV2AttributeCount+1)
	fmt.Fprintf(&attrCases, "void A2K_Local_Write_2() { a2kWriteResult(\"local_tech_control\",%d); }\n", longV2AttributeCount+2)

	trigSource := strings.TrimSpace(trigLongXS())
	if main := strings.Index(trigSource, "void main()"); main >= 0 {
		// trigLongXS places its generated cases after main. Remove only the
		// standalone main block; truncating at its start silently discarded
		// every trig case from long-v2.
		if end := strings.Index(trigSource[main:], "}\n"); end >= 0 {
			trigSource = trigSource[:main] + trigSource[main+end+2:]
		}
	}
	trigSource = strings.ReplaceAll(trigSource, "a2k", "a2kV2Trig")
	trigSource = strings.ReplaceAll(trigSource, "A2K_Test", "A2K_V2Trig_Test")
	trigSource = strings.ReplaceAll(trigSource, "A2K_Before", "A2K_V2Trig_Before")
	trigSource = strings.ReplaceAll(trigSource, "A2K_Write", "A2K_V2Trig_Write")
	trigSource = strings.ReplaceAll(trigSource, "A2K_FACTS", "A2K_V2_TRIG_FACTS")
	for _, fact := range trigLongFacts {
		trigSource = strings.ReplaceAll(trigSource, "\""+fact.ID+"\"", "\"recheck_"+fact.ID+"\"")
	}
	trigSource = strings.ReplaceAll(trigSource, "xsCloseFile();", "xsCloseFile(); a2kRecordsWritten = a2kRecordsWritten + 1;")

	return fmt.Sprintf(`// A2K long probe v2: attributes, local technology, and trig recheck.
// No deliberate crash tests. All fact triggers are inactive and non-looping.
const int A2K_FACTS = %d;
int a2kAttrUnit = -1;
int a2kAttrControl = -1;
int a2kLocalTechUnit = -1;
int a2kLocalTechControl = -1;
int a2kAttrTypeProbe = -1;
float a2kAttrTypeDefault = -1.0;
int a2kRateLastGame = -1;
int a2kRateCalls = 0;
int a2kAttrIndex = 0;
int a2kAttrPhase = 0;
bool a2kAttrSweepDone = false;
bool a2kStarted = false;
bool a2kLocalTechPending = false;
bool a2kLocalTechRecorded = false;
int a2kLocalTechStartGame = -1;
int a2kRecordsWritten = 0;
int a2kShortCircuitCounter = 0;
bool a2kShortCircuitDone = false;
bool a2kShortCircuitRecorded = false;
float a2kShortCircuitCountFloat = 0.0;

bool A2K_ShortCircuitProbeSideEffect() {
  a2kShortCircuitCounter = a2kShortCircuitCounter + 1;
  return (true);
}
void A2K_ShortCircuitProbe() {
  if (false && A2K_ShortCircuitProbeSideEffect()) { }
  if (true || A2K_ShortCircuitProbeSideEffect()) { }
  a2kShortCircuitDone = true;
}

void a2kValueFloat(string id = "", float value = 0.0) {
  if (xsCreateFile(true)) {
    xsWriteString("value"); xsWriteString(id); xsWriteString("float");
    xsWriteFloat(value); xsWriteInt(xsGetGameTime()); xsCloseFile(); a2kRecordsWritten = a2kRecordsWritten + 1;
  }
}
void a2kValueText(string id = "", string value = "") {
  if (xsCreateFile(true)) {
    xsWriteString("value"); xsWriteString(id); xsWriteString(value);
    xsWriteInt(xsGetGameTime()); xsCloseFile(); a2kRecordsWritten = a2kRecordsWritten + 1;
  }
}
void a2kBefore(string id = "", int ordinal = 0) {
  if (xsCreateFile(true)) {
    xsWriteString("before"); xsWriteString(id); xsWriteInt(ordinal);
    xsWriteInt(xsGetGameTime()); xsCloseFile(); a2kRecordsWritten = a2kRecordsWritten + 1;
  }
}
void a2kWriteResult(string id = "", int ordinal = 0) {
  if (xsCreateFile(true)) {
    xsWriteString("result"); xsWriteString(id); xsWriteInt(ordinal);
    xsWriteInt(0); xsWriteInt(0); xsWriteInt(2);
    xsWriteInt(xsGetGameTime()); xsCloseFile(); a2kRecordsWritten = a2kRecordsWritten + 1;
  }
}
void a2kMilestone(string message = "", int ordinal = 0) {
  xsChatData(message);
  a2kBefore("milestone." + message, ordinal);
}
void A2K_DriverStart() {
  if (a2kStarted == false) {
    a2kStarted = true;
    a2kMilestone("A2K probe: started (%d facts)", 0);
  }
}

void A2K_RateTick() {
  int now = xsGetGameTime();
  if (now != a2kRateLastGame) {
    if (a2kRateLastGame >= 0) {
      if (xsCreateFile(true)) {
        xsWriteString("rate"); xsWriteInt(a2kRateLastGame); xsWriteInt(a2kRateCalls);
        xsWriteInt(xsGetTurn()); xsWriteInt(xsGetWorldTime()); xsCloseFile(); a2kRecordsWritten = a2kRecordsWritten + 1;
      }
    }
    a2kRateLastGame = now;
    a2kRateCalls = 0;
  }
  a2kRateCalls = a2kRateCalls + 1;
}

void A2K_AttrBegin(int attr = 0, int ordinal = 0) {
	if (ordinal == 25) { a2kMilestone("A2K probe: attributes 25%%", ordinal); }
	if (ordinal == 51) { a2kMilestone("A2K probe: attributes 50%%", ordinal); }
	if (ordinal == 76) { a2kMilestone("A2K probe: attributes 75%%", ordinal); }
  a2kAttrUnit = xsCreateUnit(83, 1, xsVectorSet(20.5, 20.5, 0.0), false, false, false);
  a2kAttrControl = xsCreateUnit(83, 1, xsVectorSet(22.0, 20.5, 0.0), false, false, false);
  a2kAttrTypeProbe = xsCreateUnit(1366, 1, xsVectorSet(24.5, 20.5, 0.0), false, false, false);
  a2kValueFloat("unit_attribute." + attr + ".unit.default", xsGetUnitAttribute(a2kAttrUnit, attr, -1));
  a2kAttrTypeDefault = xsGetUnitAttribute(a2kAttrTypeProbe, attr, -1);
  a2kValueFloat("unit_attribute." + attr + ".control.default", a2kAttrTypeDefault);
}
void A2K_AttrWrite(int attr = 0, int ordinal = 0) {
  xsEffectAmount(10, a2kAttrUnit, attr, 123.0, 1);
  a2kValueFloat("unit_attribute." + attr + ".unit.after_unit_write", xsGetUnitAttribute(a2kAttrUnit, attr, -1));
  a2kValueFloat("unit_attribute." + attr + ".control.after_unit_write", xsGetUnitAttribute(a2kAttrControl, attr, -1));
  if (attr == 0) {
    a2kValueText("unit_attribute.0.extreme_positive", "skipped_hitpoints");
    a2kValueText("unit_attribute.0.extreme_negative", "skipped_hitpoints");
    a2kValueText("unit_attribute.0.extreme_zero", "skipped_hitpoints");
    a2kValueText("unit_attribute.0.extreme_fractional", "skipped_hitpoints");
  } else {
    xsEffectAmount(10, a2kAttrUnit, attr, 1000000.0, 1);
    a2kValueFloat("unit_attribute." + attr + ".extreme_positive", xsGetUnitAttribute(a2kAttrUnit, attr, -1));
    xsEffectAmount(10, a2kAttrUnit, attr, -1000000.0, 1);
    a2kValueFloat("unit_attribute." + attr + ".extreme_negative", xsGetUnitAttribute(a2kAttrUnit, attr, -1));
    xsEffectAmount(10, a2kAttrUnit, attr, 0.0, 1);
    a2kValueFloat("unit_attribute." + attr + ".extreme_zero", xsGetUnitAttribute(a2kAttrUnit, attr, -1));
    xsEffectAmount(10, a2kAttrUnit, attr, 0.5, 1);
    a2kValueFloat("unit_attribute." + attr + ".extreme_fractional", xsGetUnitAttribute(a2kAttrUnit, attr, -1));
  }
  if (a2kAttrTypeDefault >= 0.0) { xsEffectAmount(0, 1366, attr, a2kAttrTypeDefault, 1); }
}
void A2K_AttrEnd(int attr = 0, int ordinal = 0) {
  a2kValueFloat("unit_attribute." + attr + ".unit.after_type_write", xsGetUnitAttribute(a2kAttrUnit, attr, -1));
  a2kValueFloat("unit_attribute." + attr + ".control.after_type_write", xsGetUnitAttribute(a2kAttrControl, attr, -1));
  xsRemoveUnit(a2kAttrUnit);
  xsRemoveUnit(a2kAttrControl);
  xsRemoveUnit(a2kAttrTypeProbe);
  if (ordinal == %d) { a2kMilestone("A2K probe: attributes done, local tech next", %d); }
}

%s

void A2K_AttrStep(int index = 0, int phase = 0) {
%s
}

void A2K_AttrTick() {
  int nowGame = xsGetGameTime();
  int localTechDeadline = a2kLocalTechStartGame + 5;
  if (a2kShortCircuitRecorded == false) {
    a2kShortCircuitCountFloat = a2kShortCircuitCounter;
    a2kValueFloat("xs_logical_short_circuit.counter", a2kShortCircuitCountFloat);
    a2kShortCircuitRecorded = true;
  }
  A2K_DriverStart();
  if (a2kLocalTechPending) {
    if (a2kLocalTechRecorded == false) {
      if (nowGame >= localTechDeadline) {
        a2kValueFloat("local_tech.villager_after", xsGetUnitAttribute(a2kLocalTechUnit, 0, -1));
        a2kValueFloat("local_tech.control_after", xsGetUnitAttribute(a2kLocalTechControl, 0, -1));
        float a2kResearchedFlag = xsHasResearchedLocalTechnology(a2kLocalTechUnit, 22);
        a2kValueFloat("local_tech.researched_flag", a2kResearchedFlag);
        xsRemoveUnit(a2kLocalTechUnit);
        xsRemoveUnit(a2kLocalTechControl);
        a2kLocalTechRecorded = true;
        a2kLocalTechPending = false;
        xsSetTriggerVariable(0, 2);
        a2kMilestone("A2K probe: trig recheck next", 225);
      }
    }
  }
  if (a2kAttrIndex >= %d) {
    if (a2kAttrSweepDone == false) {
      a2kAttrSweepDone = true;
      xsSetTriggerVariable(0, 1);
    }
    return;
  }
  A2K_AttrStep(a2kAttrIndex, a2kAttrPhase);
  if (a2kAttrPhase == 2) {
    a2kAttrPhase = 0;
    a2kAttrIndex = a2kAttrIndex + 1;
  } else {
    a2kAttrPhase = a2kAttrPhase + 1;
  }
}

void A2K_LocalTech() {
  int researched = xsCreateUnit(83, 1, xsVectorSet(80.5, 80.5, 0.0), false, false, false);
  int control = xsCreateUnit(83, 1, xsVectorSet(84.5, 80.5, 0.0), false, false, false);
  a2kLocalTechUnit = researched;
  a2kLocalTechControl = control;
  float a2kVillagerBase = xsGetUnitAttribute(researched, 0, -1);
  a2kValueFloat("local_tech.villager_base", a2kVillagerBase);
  a2kValueFloat("local_tech.control_base", xsGetUnitAttribute(control, 0, -1));
  if (a2kVillagerBase == 40.0) {
    a2kValueText("local_tech.status", "null_experiment");
    xsRemoveUnit(researched);
    xsRemoveUnit(control);
    return;
  }
  xsResearchLocalTechnology(researched, 22);
  a2kValueText("local_tech.status", "active_probe");
  a2kLocalTechStartGame = xsGetGameTime();
  a2kLocalTechPending = true;
}
void A2K_TrigStart() { a2kBefore("milestone.trig_recheck", 225); }
void A2K_Done() { a2kMilestone("A2K probe: DONE, %%d records written", a2kRecordsWritten); }

rule a2kRateLogger active highFrequency { A2K_RateTick(); A2K_AttrTick(); }

%s
void main() {
  A2K_ShortCircuitProbe();
  if (xsCreateFile(false)) {
    xsWriteString("A2K_ATTR_TECH_TRIG_V2"); xsWriteInt(1);
    xsWriteString("attributes-local-tech-trig-recheck"); xsWriteInt(A2K_FACTS);
    xsCloseFile();
  }
}
`, len(facts), len(facts), longV2AttributeCount-1, longV2AttributeCount, attrCases.String(), attrSteps.String(), longV2AttributeCount, trigSource)
}

func longV2Triggers() []scenario.TriggerRecipe {
	triggers := []scenario.TriggerRecipe{{
		Op: "add_trigger", Name: "A2K Long V2 Driver", Enabled: boolPtr(true), Looping: boolPtr(false),
		Conditions: []scenario.ConditionRecipe{{Op: "timer", Timer: intPtr(1)}},
		Effects: []scenario.EffectRecipe{
			{Op: "script_call", SourcePlayer: intPtr(1), Message: "A2K_DriverStart();"},
		},
	}}
	localRaw := 1
	localNextRaw := 2
	localEffects := []scenario.EffectRecipe{
		modifyAttributeEffect(1366, 19, 123),
		modifyAttributeEffect(1366, 21, 123),
		modifyAttributeEffect(1366, 23, 123),
		{Op: "research_local_technology", SourcePlayer: intPtr(1), Technology: intPtr(22), LocalTechnology: intPtr(1), SelectedObjectIDs: []int{9001}},
		{Op: "script_call", SourcePlayer: intPtr(1), Message: "A2K_LocalTech();"},
		{Op: "activate_trigger", TriggerID: &localNextRaw},
		{Op: "deactivate_trigger", TriggerID: &localRaw},
	}
	triggers = append(triggers, scenario.TriggerRecipe{
		Op: "add_trigger", Name: "A2K Local Technology Loom",
		Enabled: boolPtr(true), Looping: boolPtr(false),
		Conditions: []scenario.ConditionRecipe{{Op: "variable_value", Variable: intPtr(0), Comparison: intPtr(4), Quantity: intPtr(1)}}, Effects: localEffects,
	})

	trigRawBase := 2
	for i := range trigLongFacts {
		raw := trigRawBase + i
		effects := []scenario.EffectRecipe{}
		if i == 0 {
			effects = append(effects, scenario.EffectRecipe{Op: "script_call", SourcePlayer: intPtr(1), Message: "A2K_TrigStart();"})
		}
		effects = append(effects,
			scenario.EffectRecipe{Op: "script_call", SourcePlayer: intPtr(1), Message: fmt.Sprintf("A2K_V2Trig_Before_%d();", i)},
			scenario.EffectRecipe{Op: "script_call", SourcePlayer: intPtr(1), Message: fmt.Sprintf("A2K_V2Trig_Test_%d();", i)},
			scenario.EffectRecipe{Op: "script_call", SourcePlayer: intPtr(1), Message: fmt.Sprintf("A2K_V2Trig_Write_%d();", i)},
		)
		if i+1 < len(trigLongFacts) {
			next := raw + 1
			effects = append(effects, scenario.EffectRecipe{Op: "activate_trigger", TriggerID: &next})
		} else {
			effects = append(effects, scenario.EffectRecipe{Op: "script_call", SourcePlayer: intPtr(1), Message: "A2K_Done();"})
		}
		effects = append(effects, scenario.EffectRecipe{Op: "deactivate_trigger", TriggerID: &raw})
		condition := scenario.ConditionRecipe{Op: "timer", Timer: intPtr(1)}
		if i == 0 {
			condition = scenario.ConditionRecipe{Op: "variable_value", Variable: intPtr(0), Comparison: intPtr(4), Quantity: intPtr(2)}
		}
		triggers = append(triggers, scenario.TriggerRecipe{
			Op: "add_trigger", Name: fmt.Sprintf("A2K Trig Recheck %03d", i),
			Enabled: boolPtr(false), Looping: boolPtr(false), Conditions: []scenario.ConditionRecipe{condition}, Effects: effects,
		})
	}
	return triggers
}

func longV2Units() []scenario.UnitRecipe {
	x1, y1 := 80.5, 80.5
	x2, y2 := 84.5, 80.5
	return []scenario.UnitRecipe{
		{Op: "add_unit", Player: 1, UnitConst: 83, X: &x1, Y: &y1, ReferenceID: intPtr(9001)},
		{Op: "add_unit", Player: 1, UnitConst: 83, X: &x2, Y: &y2, ReferenceID: intPtr(9002)},
	}
}

func modifyAttributeEffect(unit, attr, quantity int) scenario.EffectRecipe {
	source := 1
	return scenario.EffectRecipe{
		Op: "modify_attribute", SourcePlayer: &source, ObjectListUnitID: &unit,
		ObjectAttributes: &attr, Quantity: &quantity, OperationName: "set",
	}
}
