package harness

import (
	"fmt"
	"strings"

	"aoe2kit/pkg/scenario"
)

var trigLongFacts = []Fact{
	{ID: "sin_zero", Description: "sin(0)", Category: "trig", Operation: "Record sin(0).", Risk: "safe", Evidence: "guide-sourced; engine verification pending"},
	{ID: "cos_zero", Description: "cos(0)", Category: "trig", Operation: "Record cos(0).", Risk: "safe", Evidence: "guide-sourced; engine verification pending"},
	{ID: "sin_ninety", Description: "sin(radians(90))", Category: "trig", Operation: "Record sin(radians(90)).", Risk: "safe", Evidence: "guide-sourced; engine verification pending"},
	{ID: "cos_ninety", Description: "cos(radians(90))", Category: "trig", Operation: "Record cos(radians(90)).", Risk: "safe", Evidence: "guide-sourced; engine verification pending"},
	{ID: "tan_fortyfive", Description: "tan(radians(45))", Category: "trig", Operation: "Record tan(radians(45)).", Risk: "safe", Evidence: "guide-sourced; engine verification pending"},
	{ID: "atan2_quarter_turn", Description: "atan2(1,0)", Category: "trig", Operation: "Record atan2(1,0).", Risk: "safe", Evidence: "guide-sourced; engine verification pending"},
	{ID: "sqrt_sixteen", Description: "sqrt(16)", Category: "trig", Operation: "Record sqrt(16).", Risk: "safe", Evidence: "guide-sourced; engine verification pending"},
	{ID: "pow_two_ten", Description: "pow(2,10)", Category: "trig", Operation: "Record pow(2,10).", Risk: "safe", Evidence: "guide-sourced; engine verification pending"},
	{ID: "degrees_round_trip", Description: "degrees(radians(180))", Category: "trig", Operation: "Record the degree/radian round trip.", Risk: "safe", Evidence: "guide-sourced; engine verification pending"},
	{ID: "dist_three_four_five", Description: "dist((0,0,0),(3,4,0))", Category: "trig", Operation: "Record a 3-4-5 vector distance.", Risk: "safe", Evidence: "guide-sourced; engine verification pending"},
	{ID: "str_len", Description: "strLen(Hello)", Category: "strings", Operation: "Record strLen(Hello).", Risk: "safe", Evidence: "guide-sourced; engine verification pending"},
	{ID: "ord_upper", Description: "ord(A)", Category: "strings", Operation: "Record ord(A).", Risk: "safe", Evidence: "guide-sourced; engine verification pending"},
	{ID: "ord_lower", Description: "ord(a)", Category: "strings", Operation: "Record ord(a).", Risk: "safe", Evidence: "guide-sourced; engine verification pending"},
	{ID: "chr_upper", Description: "chr(65)", Category: "strings", Operation: "Record chr(65).", Risk: "safe", Evidence: "guide-sourced; engine verification pending"},
	{ID: "str_char_at", Description: "strCharAt(abc,1)", Category: "strings", Operation: "Record strCharAt indexing semantics.", Risk: "safe", Evidence: "guide-sourced; engine verification pending"},
	{ID: "str_substring", Description: "strSubstring(abcdef,1,3)", Category: "strings", Operation: "Record substring bounds semantics.", Risk: "safe", Evidence: "guide-sourced; engine verification pending"},
	{ID: "str_contains", Description: "strContains(abc,b)", Category: "strings", Operation: "Record the string contains predicate.", Risk: "safe", Evidence: "guide-sourced; engine verification pending"},
	{ID: "str_case_conversion", Description: "upper/lower conversion", Category: "strings", Operation: "Record ASCII case conversion.", Risk: "safe", Evidence: "guide-sourced; engine verification pending"},
	{ID: "str_parse_int", Description: "strToInt(42,10,-1)", Category: "strings", Operation: "Record decimal string parsing.", Risk: "safe", Evidence: "guide-sourced; engine verification pending"},
	{ID: "hex_parse", Description: "hex(ff,-1)", Category: "strings", Operation: "Record hexadecimal string parsing.", Risk: "safe", Evidence: "guide-sourced; engine verification pending"},
	{ID: "bin_parse", Description: "bin(101,-1)", Category: "strings", Operation: "Record binary string parsing.", Risk: "safe", Evidence: "guide-sourced; engine verification pending"},
	{ID: "split_join", Description: "strSplit/strJoin round trip", Category: "strings", Operation: "Record split and join output.", Risk: "safe", Evidence: "guide-sourced; engine verification pending"},
	{ID: "fstr_value", Description: "fstr interpolation", Category: "strings", Operation: "Record fstr output with one variable.", Risk: "safe", Evidence: "guide-sourced; engine verification pending"},
	{ID: "game_time", Description: "xsGetGameTime cadence", Category: "reads", Operation: "Record the rule execution time.", Risk: "safe", Evidence: "guide-sourced; engine verification pending"},
	{ID: "resource_reads", Description: "food/wood/gold reads", Category: "reads", Operation: "Record three player resource values.", Risk: "safe", Evidence: "resource reads are used elsewhere; this run measures them together"},
	{ID: "resource_ceiling_100k", Description: "resource write/read 100000", Category: "reads", Operation: "Write and read food at 100000.", Risk: "safe", Evidence: "engine ceiling calibration"},
	{ID: "resource_ceiling_500k", Description: "resource write/read 500000", Category: "reads", Operation: "Write and read food at 500000.", Risk: "safe", Evidence: "engine ceiling calibration"},
	{ID: "resource_ceiling_1m", Description: "resource write/read 1000000", Category: "reads", Operation: "Write and read food at 1000000.", Risk: "safe", Evidence: "engine ceiling calibration"},
	{ID: "unit_position_round_trip", Description: "unit position read after move", Category: "reads", Operation: "Move a footprintless Flower and read its position.", Risk: "safe", Evidence: "fractional placement is engine-verified; readback pending"},
	{ID: "unit_attribute_167", Description: "read unit attribute 167", Category: "reads", Operation: "Read a new-range unit attribute.", Risk: "safe", Evidence: "DLC field semantics pending"},
	{ID: "unit_attribute_200", Description: "read unit attribute 200", Category: "reads", Operation: "Read a new-range unit attribute.", Risk: "safe", Evidence: "DLC field semantics pending"},
	{ID: "arc_render", Description: "runtime trig arc", Category: "writes", Operation: "Create a Flower arc with runtime sin/cos.", Risk: "safe", Evidence: "new runtime geometry; visual verification pending"},
	{ID: "rotate_render", Description: "runtime trig rotation", Category: "writes", Operation: "Create a rotated five-point stroke with runtime sin/cos.", Risk: "safe", Evidence: "new runtime geometry; visual verification pending"},
	{ID: "remove_flower", Description: "Flower HP-zero lifecycle", Category: "writes", Operation: "Create Flower, zero hitpoints, read existence.", Risk: "safe", Evidence: "lifecycle behavior pending in this harness"},
	{ID: "remove_flame", Description: "Flame HP-zero lifecycle", Category: "writes", Operation: "Create Flame1, zero hitpoints, read existence.", Risk: "safe", Evidence: "flame lifecycle has prior evidence; control readback pending"},
	{ID: "local_technology", Description: "local technology isolation", Category: "writes", Operation: "Research local technology 22 at one Castle and compare two buildings.", Risk: "safe", Evidence: "shop foundation; engine verification pending"},
}

func init() {
	// Keep the new-build attribute window contiguous in the manifest. The two
	// anchor rows above remain named for readability; this fills the gaps.
	for attr := 168; attr <= 199; attr++ {
		trigLongFacts = append(trigLongFacts, Fact{
			ID: fmt.Sprintf("unit_attribute_%d", attr), Description: fmt.Sprintf("read unit attribute %d", attr),
			Category: "reads", Operation: "Read a new-range unit attribute.", Risk: "safe", Evidence: "DLC field semantics pending",
		})
	}
	for attr := 201; attr <= 221; attr++ {
		trigLongFacts = append(trigLongFacts, Fact{
			ID: fmt.Sprintf("unit_attribute_%d", attr), Description: fmt.Sprintf("read unit attribute %d", attr),
			Category: "reads", Operation: "Read a new-range unit attribute.", Risk: "safe", Evidence: "DLC field semantics pending",
		})
	}
	trigLongFacts = append(trigLongFacts,
		Fact{ID: "str_starts_with", Description: "strStartsWith(abc,a)", Category: "strings", Operation: "Record the string starts-with predicate.", Risk: "safe", Evidence: "guide-sourced; engine verification pending"},
		Fact{ID: "str_ends_with", Description: "strEndsWith(abc,c)", Category: "strings", Operation: "Record the string ends-with predicate.", Risk: "safe", Evidence: "guide-sourced; engine verification pending"},
	)
}

func trigLongXS() string {
	var cases strings.Builder
	// Each case is independent. A case may leave a value record in addition to
	// its numeric result; the pre-marker remains the choke diagnostic.
	caseText := []string{
		"a2kResultScaled(sin(0.0));", "a2kResultScaled(cos(0.0));", "a2kResultScaled(sin(radians(90.0)));", "a2kResultScaled(cos(radians(90.0)));", "a2kResultScaled(tan(radians(45.0)));", "a2kResultScaled(atan2(1.0,0.0));", "a2kResultScaled(sqrt(16.0));", "a2kResultScaled(pow(2.0,10.0));", "a2kResultScaled(degrees(radians(180.0)));", "a2kResultScaled(dist(xsVectorSet(0.0,0.0,0.0),xsVectorSet(3.0,4.0,0.0)));",
		"a2kResult(strLen(\"Hello\"),5,1);", "a2kValue(\"ord_upper\",chr(ord(\"A\"))); a2kResult(ord(\"A\"),65,1);", "a2kResult(ord(\"a\"),97,1);", "a2kValue(\"chr_upper\",chr(65)); a2kResult(strLen(chr(65)),1,1);", "a2kValue(\"str_char_at\",strCharAt(\"abc\",1)); a2kResult(ord(strCharAt(\"abc\",1)),98,1);", "a2kValue(\"str_substring\",strSubstring(\"abcdef\",1,3)); a2kResult(strLen(strSubstring(\"abcdef\",1,3)),2,1);", "a2kResult(strContains(\"abc\",\"b\"),1,2);", "a2kValue(\"str_case_conversion\",strToUpper(\"a\")); a2kResult(strLen(strToLower(\"A\")),1,1);", "a2kResult(strToInt(\"42\",10,-1),42,1);", "a2kResult(hex(\"ff\",-1),255,1);", "a2kResult(bin(\"101\",-1),5,1);", "int a = strSplit(\"a,b,c\",\",\"); a2kValue(\"split_join\",strJoin(a,\",\")); a2kResult(strLen(strJoin(a,\",\")),5,1);", "int fstrValue = 42; a2kValue(\"fstr_value\",fstr(\"value {fstrValue}\")); a2kResult(1,1,2);",
		"a2kResult(xsGetGameTime(),0,2);", "a2kValueFloat(\"resource_reads\",xsPlayerAttribute(1,0)); a2kValueFloat(\"resource_reads\",xsPlayerAttribute(1,1)); a2kValueFloat(\"resource_reads\",xsPlayerAttribute(1,3)); a2kResult(1,1,2);", "xsSetPlayerAttribute(1,0,100000.0); a2kResultScaled(xsPlayerAttribute(1,0));", "xsSetPlayerAttribute(1,0,500000.0); a2kResultScaled(xsPlayerAttribute(1,0));", "xsSetPlayerAttribute(1,0,1000000.0); a2kResultScaled(xsPlayerAttribute(1,0));", "xsSetUnitPosition(a2kFlower,xsVectorSet(22.5,22.5,0.0),false); a2kResultScaled(xsVectorGetX(xsGetUnitPosition(a2kFlower)));", "a2kResultScaled(xsGetUnitAttribute(a2kFlower,167,-1));", "a2kResultScaled(xsGetUnitAttribute(a2kFlower,200,-1));",
		"for (i = 0; < 16) { float arcIndex = i; float arcAngle = radians(360.0*arcIndex/16.0); xsCreateUnit(1366,0,xsVectorSet(62.0+6.0*cos(arcAngle),62.0+6.0*sin(arcAngle),0.0),false,false,false); } a2kResult(16,16,1);", "for (i = 0; < 5) { float rotateAngle = radians(45.0); float rotateX = i; float rotateY = 0.0; float rotateRX = rotateX*cos(rotateAngle)-rotateY*sin(rotateAngle); float rotateRY = rotateX*sin(rotateAngle)+rotateY*cos(rotateAngle); xsCreateUnit(1366,0,xsVectorSet(72.0+rotateRX,72.0+rotateRY,0.0),false,false,false); } a2kResult(5,5,1);", "int flowerToRemove = xsCreateUnit(1366,0,xsVectorSet(28.5,28.5,0.0),false,false,false); xsSetUnitHitpoints(flowerToRemove,0.0); a2kResult(xsDoesUnitExist(flowerToRemove),0,1);", "int flameToRemove = xsCreateUnit(939,0,xsVectorSet(30.5,30.5,0.0),false,false,false); xsSetUnitHitpoints(flameToRemove,0.0); a2kResult(xsDoesUnitExist(flameToRemove),0,1);", "if (a2kCastleA < 0) a2kCastleA = xsCreateUnit(82,1,xsVectorSet(40.5,40.5,0.0),false,false,false); if (a2kCastleB < 0) a2kCastleB = xsCreateUnit(82,1,xsVectorSet(50.5,40.5,0.0),false,false,false); xsResearchLocalTechnology(a2kCastleA,22); a2kResult(xsHasResearchedLocalTechnology(a2kCastleA,22),1,2);",
	}
	for attr := 168; attr <= 199; attr++ {
		caseText = append(caseText, fmt.Sprintf("a2kResultScaled(xsGetUnitAttribute(a2kFlower,%d,-1));", attr))
	}
	for attr := 201; attr <= 221; attr++ {
		caseText = append(caseText, fmt.Sprintf("a2kResultScaled(xsGetUnitAttribute(a2kFlower,%d,-1));", attr))
	}
	caseText = append(caseText, "a2kResult(strStartsWith(\"abc\",\"a\"),1,2);", "a2kResult(strEndsWith(\"abc\",\"c\"),1,2);")
	for i, body := range caseText {
		body = strings.ReplaceAll(body, "a2kResultScaled(", fmt.Sprintf("a2kObserveScaled(\"%s\", ", trigLongFacts[i].ID))
		body = strings.ReplaceAll(body, "a2kResult(", "a2kObserve(")
		fmt.Fprintf(&cases, "void A2K_Test_%d() { a2kFact = %d; a2kExpected = 0; a2kObserved = 0; a2kStatus = 1; %s }\n", i, i, body)
		fmt.Fprintf(&cases, "void A2K_Before_%d() { a2kBefore(\"%s\",%d); }\n", i, trigLongFacts[i].ID, i)
		fmt.Fprintf(&cases, "void A2K_Write_%d() { a2kWriteResult(\"%s\"); }\n", i, trigLongFacts[i].ID)
	}
	return fmt.Sprintf(`// A2K long unattended capability probe. No deliberate crash tests.
const int A2K_FACTS = %d;
int a2kFact = 0; int a2kFlower = -1; int a2kCastleA = -1; int a2kCastleB = -1;
	int a2kExpected = 0; int a2kObserved = 0; int a2kStatus = 1;
void a2kObserve(int observed = 0, int expected = 0, int status = 1) { a2kObserved = observed; a2kExpected = expected; a2kStatus = status; }
void a2kObserveScaled(string id = "", float value = 0.0) { a2kObserved = 0; a2kExpected = 0; a2kStatus = 2; if (xsCreateFile(true)) { xsWriteString("value"); xsWriteString(id); xsWriteString("float"); xsWriteFloat(value); xsWriteInt(xsGetGameTime()); xsCloseFile(); } }
void a2kValue(string id = "", string value = "") { if (xsCreateFile(true)) { xsWriteString("value"); xsWriteString(id); xsWriteString(value); xsWriteInt(xsGetGameTime()); xsCloseFile(); } }
void a2kValueFloat(string id = "", float value = 0.0) { if (xsCreateFile(true)) { xsWriteString("value"); xsWriteString(id); xsWriteString("float"); xsWriteFloat(value); xsWriteInt(xsGetGameTime()); xsCloseFile(); } }
void a2kBefore(string id = "", int ordinal = 0) { if (xsCreateFile(true)) { xsWriteString("before"); xsWriteString(id); xsWriteInt(ordinal); xsWriteInt(xsGetGameTime()); xsCloseFile(); } }
void a2kWriteResult(string id = "unknown") { if (xsCreateFile(true)) { xsWriteString("result"); xsWriteString(id); xsWriteInt(a2kFact); xsWriteInt(a2kExpected); xsWriteInt(a2kObserved); xsWriteInt(a2kStatus); xsWriteInt(xsGetGameTime()); xsCloseFile(); } }
void main() { a2kFlower = xsCreateUnit(1366,0,xsVectorSet(20.5,20.5,0.0),false,false,false); if (xsCreateFile(false)) { xsWriteString("A2K_TRIG_LONG"); xsWriteInt(1); xsWriteString("trig-strings-reads-writes"); xsWriteInt(A2K_FACTS); xsCloseFile(); } }
%s`, len(trigLongFacts), cases.String())
}

func trigLongTriggers() []scenario.TriggerRecipe {
	triggers := []scenario.TriggerRecipe{{
		Op: "add_trigger", Name: "A2K Long Probe Driver", Enabled: boolPtr(true), Looping: boolPtr(true),
		Conditions: []scenario.ConditionRecipe{{Op: "timer", Timer: intPtr(1)}},
		Effects:    []scenario.EffectRecipe{{Op: "script_call", SourcePlayer: intPtr(1), Message: "A2K_Before_0();"}, {Op: "activate_trigger", TriggerID: intPtr(1)}},
	}}
	for i, fact := range trigLongFacts {
		effects := trigLongNativeEffects(i)
		effects = append(effects, scenario.EffectRecipe{Op: "script_call", SourcePlayer: intPtr(1), Message: fmt.Sprintf("A2K_Test_%d();", i)})
		if i+1 < len(trigLongFacts) {
			effects = append(effects,
				scenario.EffectRecipe{Op: "script_call", SourcePlayer: intPtr(1), Message: fmt.Sprintf("A2K_Before_%d();", i+1)},
				scenario.EffectRecipe{Op: "activate_trigger", TriggerID: intPtr(i + 2)},
			)
		}
		effects = append(effects, scenario.EffectRecipe{Op: "script_call", SourcePlayer: intPtr(1), Message: fmt.Sprintf("A2K_Write_%d();", i)})
		// DE can re-fire an activated trigger despite the serialized looping=0
		// flag. Explicitly retire each fact trigger after its one record.
		factTriggerID := i + 2 // raw scenario trigger 0 is the XS carrier
		effects = append(effects, scenario.EffectRecipe{Op: "deactivate_trigger", TriggerID: &factTriggerID})
		triggers = append(triggers, scenario.TriggerRecipe{
			Op: "add_trigger", Name: fmt.Sprintf("A2K Fact %03d %s", i, fact.ID), Enabled: boolPtr(false), Looping: boolPtr(false), Effects: effects,
		})
	}
	return triggers
}

func trigLongNativeEffects(index int) []scenario.EffectRecipe {
	source := 1
	resource := 0
	set := 0
	var effects []scenario.EffectRecipe
	switch index {
	case 25, 26, 27:
		amount := []int{100000, 500000, 1000000}[index-25]
		effects = append(effects, scenario.EffectRecipe{Op: "modify_resource", SourcePlayer: &source, Resource: &resource, Quantity: &amount, Operation: &set})
	case 31:
		for i := 0; i < 16; i++ {
			x, y := 62+i%4*3, 62+i/4*3
			unit, player := 1366, 0
			effects = append(effects, scenario.EffectRecipe{Op: "create_object", SourcePlayer: &player, ObjectListUnitID: &unit, LocationX: &x, LocationY: &y})
		}
	case 32:
		for i := 0; i < 5; i++ {
			x, y := 72+i*2, 72+i
			unit, player := 1366, 0
			effects = append(effects, scenario.EffectRecipe{Op: "create_object", SourcePlayer: &player, ObjectListUnitID: &unit, LocationX: &x, LocationY: &y})
		}
	case 33, 34:
		unit, player, max := 1366, 0, 1
		if index == 34 {
			unit = 939
		}
		x1, y1, x2, y2 := 27, 27, 32, 32
		effects = append(effects, scenario.EffectRecipe{Op: "remove_object", SourcePlayer: &player, ObjectListUnitID: &unit, AreaX1: &x1, AreaY1: &y1, AreaX2: &x2, AreaY2: &y2, MaxUnitsAffected: &max})
	}
	return effects
}

func trigLongUnits() []scenario.UnitRecipe {
	flowerX, flowerY := 28.5, 28.5
	flameX, flameY := 30.5, 30.5
	return []scenario.UnitRecipe{
		{Op: "add_unit", Player: 0, UnitConst: 1366, X: &flowerX, Y: &flowerY},
		{Op: "add_unit", Player: 0, UnitConst: 939, X: &flameX, Y: &flameY},
	}
}
