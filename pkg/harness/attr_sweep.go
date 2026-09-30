package harness

import (
	"fmt"
	"strings"

	"aoe2kit/pkg/scenario"
)

var attrSweepFacts = make([]Fact, 222)

func init() {
	for attr := range attrSweepFacts {
		attrSweepFacts[attr] = Fact{
			ID: fmt.Sprintf("unit_attribute_%d", attr), Description: fmt.Sprintf("write/read Flower attribute %d", attr),
			Category: "unit_attributes", Setup: "Create one Gaia Flower and set the attribute through a Modify Attribute trigger effect.",
			Operation: "Read the attribute after the trigger write.", Risk: "safe", Evidence: "write-then-read engine probe; verification pending",
		}
	}
}

func attrSweepXS() string {
	var cases strings.Builder
	for attr := range attrSweepFacts {
		fmt.Fprintf(&cases, "void A2K_Test_%d() { a2kFact = %d; a2kObserveScaled(\"unit_attribute_%d\", xsGetUnitAttribute(a2kFlower,%d,-1)); }\n", attr, attr, attr, attr)
		fmt.Fprintf(&cases, "void A2K_Before_%d() { a2kBefore(\"unit_attribute_%d\",%d); }\n", attr, attr, attr)
		fmt.Fprintf(&cases, "void A2K_Write_%d() { a2kWriteResult(\"unit_attribute_%d\"); }\n", attr, attr)
	}
	return fmt.Sprintf(`// A2K write-then-read unit attribute sweep. No deliberate crash tests.
const int A2K_FACTS = %d;
int a2kFact = 0; int a2kFlower = -1; int a2kExpected = 0; int a2kObserved = 0; int a2kStatus = 1;
void a2kObserveScaled(string id = "", float value = 0.0) { a2kObserved = 0; a2kExpected = 0; a2kStatus = 2; if (xsCreateFile(true)) { xsWriteString("value"); xsWriteString(id); xsWriteString("float"); xsWriteFloat(value); xsWriteInt(xsGetGameTime()); xsCloseFile(); } }
void a2kBefore(string id = "", int ordinal = 0) { if (xsCreateFile(true)) { xsWriteString("before"); xsWriteString(id); xsWriteInt(ordinal); xsWriteInt(xsGetGameTime()); xsCloseFile(); } }
void a2kWriteResult(string id = "unknown") { if (xsCreateFile(true)) { xsWriteString("result"); xsWriteString(id); xsWriteInt(a2kFact); xsWriteInt(a2kExpected); xsWriteInt(a2kObserved); xsWriteInt(a2kStatus); xsWriteInt(xsGetGameTime()); xsCloseFile(); } }
void main() { a2kFlower = xsCreateUnit(1366,0,xsVectorSet(20.5,20.5,0.0),false,false,false); if (xsCreateFile(false)) { xsWriteString("A2K_ATTR_SWEEP"); xsWriteInt(1); xsWriteString("write-then-read-0-221"); xsWriteInt(A2K_FACTS); xsCloseFile(); } }
%s`, len(attrSweepFacts), cases.String())
}

func attrSweepTriggers() []scenario.TriggerRecipe {
	driver := scenario.TriggerRecipe{
		Op: "add_trigger", Name: "A2K Attribute Sweep Driver", Enabled: boolPtr(true), Looping: boolPtr(false),
		Conditions: []scenario.ConditionRecipe{{Op: "timer", Timer: intPtr(1)}},
		Effects:    []scenario.EffectRecipe{{Op: "script_call", SourcePlayer: intPtr(1), Message: "A2K_Before_0();"}, {Op: "activate_trigger", TriggerID: intPtr(1)}},
	}
	triggers := []scenario.TriggerRecipe{driver}
	for attr := range attrSweepFacts {
		source, unit, quantity := 0, 1366, 123
		attribute := attr
		effects := []scenario.EffectRecipe{{
			Op: "modify_attribute", SourcePlayer: &source, ObjectListUnitID: &unit,
			ObjectAttributes: &attribute, Quantity: &quantity, OperationName: "set",
		}, {Op: "script_call", SourcePlayer: intPtr(1), Message: fmt.Sprintf("A2K_Test_%d();", attr)}}
		if attr+1 < len(attrSweepFacts) {
			effects = append(effects,
				scenario.EffectRecipe{Op: "script_call", SourcePlayer: intPtr(1), Message: fmt.Sprintf("A2K_Before_%d();", attr+1)},
				scenario.EffectRecipe{Op: "activate_trigger", TriggerID: intPtr(attr + 2)},
			)
		}
		effects = append(effects,
			scenario.EffectRecipe{Op: "script_call", SourcePlayer: intPtr(1), Message: fmt.Sprintf("A2K_Write_%d();", attr)},
			scenario.EffectRecipe{Op: "deactivate_trigger", TriggerID: intPtr(attr + 2)},
		)
		triggers = append(triggers, scenario.TriggerRecipe{Op: "add_trigger", Name: fmt.Sprintf("A2K Attribute %03d", attr), Enabled: boolPtr(false), Looping: boolPtr(false), Effects: effects})
	}
	return triggers
}

func attrSweepUnits() []scenario.UnitRecipe {
	x, y := 20.5, 20.5
	return []scenario.UnitRecipe{{Op: "add_unit", Player: 0, UnitConst: 1366, X: &x, Y: &y}}
}
