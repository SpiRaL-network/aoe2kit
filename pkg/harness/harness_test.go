package harness

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"aoe2kit/pkg/xs"
)

func TestFactsKeepHazardousComparatorsOutOfSafeBatch(t *testing.T) {
	facts := Facts(BatchSafe)
	if len(facts) != 7 {
		t.Fatalf("safe facts=%d, want 7", len(facts))
	}
	for _, fact := range facts {
		if fact.ID == "string_case_insensitive_less_equal" {
			t.Fatal("known hazardous <= probe leaked into safe batch")
		}
	}
}

func TestBuildSixArmHarnessHasIndependentRules(t *testing.T) {
	report, err := Build(Options{OutputDir: t.TempDir(), Timestamp: 123, Batch: BatchSixArm, AllowMissingXSCheck: true})
	if err != nil {
		t.Fatal(err)
	}
	if report.FactCount != 6 {
		t.Fatalf("fact count=%d, want 6", report.FactCount)
	}
	source, err := os.ReadFile(report.XSPath)
	if err != nil {
		t.Fatal(err)
	}
	text := string(source)
	for _, rule := range []string{"a2kArmA", "a2kArmB", "a2kArmC", "a2kArmD", "a2kArmE", "a2kArmF"} {
		if !strings.Contains(text, "rule "+rule) {
			t.Fatalf("missing rule %s", rule)
		}
	}
	if strings.Count(text, "\nrule ") != 6 || !strings.Contains(text, "A2K_ArmFTick") {
		t.Fatalf("six-arm structure missing: %s", text)
	}
}

func TestBuildTrigLongHarnessIsBoundedAndCrashFreeByConstruction(t *testing.T) {
	triggers := trigLongTriggers()
	if len(triggers) != len(trigLongFacts)+1 || !*triggers[0].Enabled {
		t.Fatalf("trigger count/driver=%d/%v", len(triggers), triggers[0].Enabled)
	}
	for i := 1; i < len(triggers); i++ {
		if *triggers[i].Enabled || *triggers[i].Looping {
			t.Fatalf("fact trigger %d is not inactive/non-looping", i)
		}
		effects := triggers[i].Effects
		if len(effects) < 2 || effects[len(effects)-2].Op != "script_call" || !strings.HasPrefix(effects[len(effects)-2].Message, "A2K_Write_") || effects[len(effects)-1].Op != "deactivate_trigger" {
			t.Fatalf("fact trigger %d lacks final writer: %+v", i, effects)
		}
		for _, effect := range effects {
			if effect.Op == "script_call" && strings.Contains(effect.Message, "(") && !strings.HasSuffix(effect.Message, "();") {
				t.Fatalf("parameterized script call leaked into fact trigger %d: %q", i, effect.Message)
			}
		}
	}
	if len(triggers[26].Effects) < 2 || triggers[26].Effects[0].Op != "modify_resource" || len(triggers[32].Effects) < 2 || triggers[32].Effects[0].Op != "create_object" || len(triggers[34].Effects) < 2 || triggers[34].Effects[0].Op != "remove_object" {
		t.Fatal("native trigger effects missing from resource/render/removal facts")
	}
	report, err := Build(Options{OutputDir: t.TempDir(), Timestamp: 123, Batch: BatchTrigLong, AllowMissingXSCheck: true})
	if err != nil {
		t.Fatal(err)
	}
	if report.FactCount != len(trigLongFacts) || report.FactCount < 30 || report.ExpectedRecords != 263 || report.SidecarByteCap != (1<<20)-1 {
		t.Fatalf("fact count=%d, want %d and a long probe", report.FactCount, len(trigLongFacts))
	}
	source, err := os.ReadFile(report.XSPath)
	if err != nil {
		t.Fatal(err)
	}
	text := string(source)
	if !strings.Contains(text, "A2K_TRIG_LONG") || strings.Contains(text, "script_call") {
		t.Fatalf("long probe identity or safety contract missing")
	}
	if got := xs.LintSource(report.XSPath, text); len(got) != 0 {
		t.Fatalf("generated long probe lint findings=%+v", got)
	}
	types, err := os.ReadFile(report.TypesPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(types), "value = string,string,string,float,int") {
		t.Fatalf("long probe did not select float value schema: %s", types)
	}
}

func TestBuildAttrSweepHarnessIsCompleteAndSequenced(t *testing.T) {
	if len(attrSweepFacts) != 222 {
		t.Fatalf("attribute facts=%d, want 222", len(attrSweepFacts))
	}
	triggers := attrSweepTriggers()
	if len(triggers) != 223 || !*triggers[0].Enabled || *triggers[0].Looping {
		t.Fatalf("trigger count/driver=%d/%v/%v", len(triggers), triggers[0].Enabled, triggers[0].Looping)
	}
	for i := 1; i < len(triggers); i++ {
		trigger := triggers[i]
		if *trigger.Enabled || *trigger.Looping {
			t.Fatalf("attribute trigger %d is not inactive/non-looping", i-1)
		}
		if len(trigger.Effects) < 4 || trigger.Effects[0].Op != "modify_attribute" {
			t.Fatalf("attribute trigger %d lacks modify effect: %+v", i-1, trigger.Effects)
		}
		effect := trigger.Effects[0]
		if effect.ObjectAttributes == nil || *effect.ObjectAttributes != i-1 || effect.Quantity == nil || *effect.Quantity != 123 || effect.OperationName != "set" || effect.Operation != nil {
			t.Fatalf("attribute trigger %d has wrong modify payload: %+v", i-1, effect)
		}
		if trigger.Effects[len(trigger.Effects)-1].Op != "deactivate_trigger" {
			t.Fatalf("attribute trigger %d lacks deactivation", i-1)
		}
		for _, effect := range trigger.Effects {
			if effect.Op == "script_call" && !strings.HasSuffix(effect.Message, "();") {
				t.Fatalf("parameterized script call leaked into attribute trigger %d: %q", i-1, effect.Message)
			}
		}
	}
	report, err := Build(Options{OutputDir: t.TempDir(), Timestamp: 123, Batch: BatchAttrSweep, AllowMissingXSCheck: true})
	if err != nil {
		t.Fatal(err)
	}
	if report.FactCount != 222 || report.ExpectedRecords != 668 || report.SidecarByteCap != (1<<20)-1 {
		t.Fatalf("report=%+v", report)
	}
	source, err := os.ReadFile(report.XSPath)
	if err != nil {
		t.Fatal(err)
	}
	if got := xs.LintSource(report.XSPath, string(source)); len(got) != 0 {
		t.Fatalf("generated attribute sweep lint findings=%+v", got)
	}
	types, err := os.ReadFile(report.TypesPath)
	if err != nil {
		t.Fatal(err)
	}
	if schema, ok, err := xs.SchemaFromTypesDocument(string(types)); err != nil || !ok || schema != "a2k-trig-long" {
		t.Fatalf("attribute sweep schema=%q ok=%v err=%v", schema, ok, err)
	}
}

func TestBuildLongV2HarnessIsCompleteAndNamespaced(t *testing.T) {
	facts := longV2FactsFor()
	if len(facts) != 197 {
		t.Fatalf("long-v2 facts=%d, want 197", len(facts))
	}
	triggers := longV2Triggers()
	wantTriggers := 1 + 1 + len(trigLongFacts)
	if len(triggers) != wantTriggers || !*triggers[0].Enabled || *triggers[0].Looping {
		t.Fatalf("trigger count/driver=%d/%v/%v, want %d", len(triggers), triggers[0].Enabled, triggers[0].Looping, wantTriggers)
	}
	for i := 1; i < len(triggers); i++ {
		if i != 1 && *triggers[i].Enabled {
			t.Fatalf("trigger %d is unexpectedly enabled", i)
		}
		if *triggers[i].Looping {
			t.Fatalf("trigger %d is not inactive/non-looping", i)
		}
		for _, effect := range triggers[i].Effects {
			if effect.Op == "script_call" && !strings.HasSuffix(effect.Message, "();") {
				t.Fatalf("parameterized script call leaked into trigger %d: %q", i, effect.Message)
			}
		}
	}
	if triggers[1].Effects[0].Op != "modify_attribute" || triggers[1].Effects[0].ObjectAttributes == nil || *triggers[1].Effects[0].ObjectAttributes != 19 {
		t.Fatalf("local trigger has wrong native effect: %+v", triggers[1].Effects[0])
	}
	local := triggers[1]
	if local.Effects[3].Op != "research_local_technology" || local.Effects[3].SelectedObjectIDs[0] != 9001 {
		t.Fatalf("local technology trigger missing selected object: %+v", local.Effects)
	}

	report, err := Build(Options{OutputDir: t.TempDir(), Timestamp: 123, Batch: BatchLongV2, AllowMissingXSCheck: true})
	if err != nil {
		t.Fatal(err)
	}
	if report.FactCount != len(facts) || report.ExpectedRecords != 1702 || report.EstimatedDurationS != 194 || report.SidecarByteCap != (1<<20)-1 {
		t.Fatalf("report=%+v", report)
	}
	source, err := os.ReadFile(report.XSPath)
	if err != nil {
		t.Fatal(err)
	}
	if got := xs.LintSource(report.XSPath, string(source)); len(got) != 0 {
		t.Fatalf("generated long-v2 lint findings=%+v", got)
	}
	text := string(source)
	for _, marker := range []string{"A2K_ATTR_TECH_TRIG_V2", "a2kV2TrigBefore", "A2K_V2Trig_Before_0", "A2K_V2Trig_Write_90", "A2K_Attr_Begin_221", "A2K_LocalTech", "a2kRecordsWritten"} {
		if !strings.Contains(text, marker) {
			t.Fatalf("generated long-v2 source missing %q", marker)
		}
	}
	types, err := os.ReadFile(report.TypesPath)
	if err != nil {
		t.Fatal(err)
	}
	if schema, ok, err := xs.SchemaFromTypesDocument(string(types)); err != nil || !ok || schema != "a2k-trig-long" {
		t.Fatalf("long-v2 schema=%q ok=%v err=%v", schema, ok, err)
	}
}

func TestBuildLongV3HarnessIsPacedAndDifferentialCleanByConstruction(t *testing.T) {
	facts := longV3FactsFor()
	if len(longV3UnitTypes()) < 100 || len(facts) < 1700 {
		t.Fatalf("long-v3 catalog/facts=%d/%d, want a broad catalog", len(longV3UnitTypes()), len(facts))
	}
	triggers := longV3Triggers()
	if len(triggers) != 1 || !*triggers[0].Enabled || !*triggers[0].Looping || len(triggers[0].Effects) != 3 {
		t.Fatalf("long-v3 trigger block=%+v", triggers)
	}
	for _, effect := range triggers[0].Effects {
		if effect.Op == "modify_attribute" && (effect.OperationName != "set" || effect.Operation != nil) {
			t.Fatalf("long-v3 trigger uses unnamed operation: %+v", effect)
		}
		if effect.Op == "script_call" && !strings.HasSuffix(effect.Message, "();") {
			t.Fatalf("long-v3 parameterized script call leaked: %q", effect.Message)
		}
	}
	report, err := Build(Options{OutputDir: t.TempDir(), Timestamp: 123, Batch: BatchLongV3, AllowMissingXSCheck: true})
	if err != nil {
		t.Fatal(err)
	}
	if report.FactCount != len(facts) || report.ExpectedRecords != longV3ExpectedRecords() || report.EstimatedDurationS != longV3EstimatedDurationS() || report.SidecarByteCap != (1<<20)-1 {
		t.Fatalf("long-v3 report=%+v", report)
	}
	source, err := os.ReadFile(report.XSPath)
	if err != nil {
		t.Fatal(err)
	}
	text := string(source)
	if got := xs.LintSource(report.XSPath, text); len(got) != 0 {
		t.Fatalf("generated long-v3 lint findings=%+v", got)
	}
	for _, marker := range []string{"A2K_ENGINE_HARNESS_LONG_V3", "xsTaskUnits", "A2K_V3_TriggerReadback", "a2kV3Ended", "xsGetWorldTime", "block A done", "trails done", "halo done", "Z look done", "REAL records"} {
		if !strings.Contains(text, marker) {
			t.Fatalf("generated long-v3 source missing %q", marker)
		}
	}
	types, err := os.ReadFile(report.TypesPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(types), "treatment = string,int,int,int,float,float,int") || !strings.Contains(string(types), "footer_integrity = 424242") {
		t.Fatalf("long-v3 types missing treatment/footer schema: %s", types)
	}
}

func TestBuildAndDecodeSafeHarness(t *testing.T) {
	dir := t.TempDir()
	report, err := Build(Options{OutputDir: dir, Timestamp: 123, Batch: BatchSafe, AllowMissingXSCheck: true})
	if err != nil {
		t.Fatal(err)
	}
	if report.FactCount != 7 || report.ScenarioSHA == "" || report.XSSHA == "" || report.ExecutionCarrier != "P1 resource 0 base 100000" {
		t.Fatalf("report=%+v", report)
	}
	xsBytes, err := os.ReadFile(report.XSPath)
	if err != nil {
		t.Fatal(err)
	}
	if got := xs.LintSource(report.XSPath, string(xsBytes)); len(got) != 0 {
		t.Fatalf("generated XS lint findings=%+v", got)
	}
	if !bytes.Contains(xsBytes, []byte("float a2kCarrier = 100000.0")) || !bytes.Contains(xsBytes, []byte("xsSetPlayerAttribute(1, 0, a2kCarrier)")) {
		t.Fatalf("generated XS is missing execution oracle: %s", xsBytes)
	}
	manifest, err := os.ReadFile(report.ManifestPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(manifest) == 0 {
		t.Fatal("empty manifest")
	}

	data := appendHarnessFixture(nil)
	decoded := xs.DecodeDataBytesSchema(data, xs.DataSchemaDecodeOptions{Schema: "engine-harness"})
	if !decoded.OK || len(decoded.Rows) != 2 || !decoded.Summary.HasFooter {
		t.Fatalf("decoded=%+v", decoded)
	}
	if decoded.Rows[1].Fields["fact_id"] != "array_int_round_trip" || decoded.Rows[1].Fields["pass"] != 1 {
		t.Fatalf("row=%+v", decoded.Rows[1])
	}
	if decoded.Footer["integrity"] != 424242 {
		t.Fatalf("footer=%v", decoded.Footer)
	}
}

func appendHarnessFixture(data []byte) []byte {
	putString := func(s string) {
		var b [4]byte
		binary.LittleEndian.PutUint32(b[:], uint32(len(s)))
		data = append(data, b[:]...)
		data = append(data, s...)
	}
	putInt := func(v int32) {
		var b [4]byte
		binary.LittleEndian.PutUint32(b[:], uint32(v))
		data = append(data, b[:]...)
	}
	putString("A2K_ENGINE_HARNESS")
	putInt(1)
	putString("safe")
	putInt(7)
	putString("before")
	putString("safe")
	putString("array_int_round_trip")
	putInt(1)
	putInt(0)
	putInt(0)
	putInt(0)
	putInt(1)
	putString("result")
	putString("safe")
	putString("array_int_round_trip")
	putInt(1)
	putInt(37)
	putInt(37)
	putInt(1)
	putInt(2)
	putString("end")
	putInt(7)
	putInt(424242)
	return data
}

func TestBuildRejectsUnknownBatch(t *testing.T) {
	if _, err := Build(Options{OutputDir: filepath.Join(t.TempDir(), "out"), Batch: "hazardous", AllowMissingXSCheck: true}); err == nil {
		t.Fatal("unknown batch unexpectedly accepted")
	}
}

func TestExpandRemovalSweepLeavesExpectationsUnknown(t *testing.T) {
	facts := ExpandRemovalSweep([]int{1366, 939, 12})
	if len(facts) != 3 || facts[1].ID != "remove_hp_zero_unit_939" {
		t.Fatalf("facts=%+v", facts)
	}
	for _, fact := range facts {
		if fact.Expected != nil {
			t.Fatalf("removal fact %q unexpectedly has expected=%v", fact.ID, *fact.Expected)
		}
	}
}
