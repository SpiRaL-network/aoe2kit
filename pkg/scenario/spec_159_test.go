package scenario

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestScenario159AddsAnonymousUnitByteBeforeCaptionID(t *testing.T) {
	spec, err := LoadDESpecForVersion("1.59")
	if err != nil {
		t.Fatalf("LoadDESpecForVersion: %v", err)
	}
	units, ok := spec.section("Units")
	if !ok {
		t.Fatal("missing Units section")
	}
	unit := units.Structs["PlayerUnitsStruct"].Structs["UnitStruct"]
	for i, field := range unit.Fields {
		if field.Name != "unknown_1_59_before_caption_string_id" {
			continue
		}
		if i == 0 || unit.Fields[i-1].Name != "garrisoned_in_id" || unit.Fields[i+1].Name != "caption_string_id" {
			t.Fatalf("anonymous field order is wrong at index %d", i)
		}
		return
	}
	t.Fatal("missing anonymous 1.59 unit field")
}

func TestScenario159ByteRoundTripExternalFixtures(t *testing.T) {
	fixtures := []string{
		scenarioProjectPath("Mandala", "ring_work", "ryanscratch", "RyanScratch.original.aoe2scenario"),
		scenarioProjectPath("Mandala", "ring_work", "gate_footprint", "gate_footprint.aoe2scenario"),
		// Editor save of RyanScratch AB2: +74 inflated bytes versus the
		// kit-authored copy, including one opaque s32 per non-empty condition.
		scenarioProjectPath("Mandala", "ring_work", "ryanscratch", "AB2_as_run.aoe2scenario"),
	}
	for _, input := range fixtures {
		original, err := os.ReadFile(input)
		if os.IsNotExist(err) {
			t.Skipf("external 1.59 fixture unavailable: %s", input)
		}
		if err != nil {
			t.Fatalf("read %s: %v", input, err)
		}
		file, err := Open(input)
		if err != nil {
			t.Fatalf("open %s: %v", input, err)
		}
		rebuiltBody, err := file.RebuildBody()
		if err != nil {
			t.Fatalf("rebuild body %s: %v", input, err)
		}
		if !bytes.Equal(file.originalBody, rebuiltBody) {
			i := firstByteDifference(file.originalBody, rebuiltBody)
			t.Logf("body differs before compression: first byte %d (original=%d rebuilt=%d; lengths %d/%d)", i, byteAt(file.originalBody, i), byteAt(rebuiltBody, i), len(file.originalBody), len(rebuiltBody))
		}
		output := filepath.Join(t.TempDir(), filepath.Base(input))
		if err := file.WriteReencoded(output); err != nil {
			t.Fatalf("reencoded write %s: %v", input, err)
		}
		got, err := os.ReadFile(output)
		if err != nil {
			t.Fatalf("read round-trip %s: %v", input, err)
		}
		roundTripped, err := Open(output)
		if err != nil {
			t.Fatalf("open reencoded %s: %v", input, err)
		}
		if !bytes.Equal(file.header, roundTripped.header) {
			i := firstByteDifference(file.header, roundTripped.header)
			t.Fatalf("%s header changed after re-encode at byte %d (original=%d output=%d)", input, i, byteAt(file.header, i), byteAt(roundTripped.header, i))
		}
		if !bytes.Equal(file.body, roundTripped.body) {
			i := firstByteDifference(file.body, roundTripped.body)
			t.Fatalf("%s inflated body changed after re-encode at byte %d (original=%d output=%d; lengths %d/%d)", input, i, byteAt(file.body, i), byteAt(roundTripped.body, i), len(file.body), len(roundTripped.body))
		}
		if !bytes.Equal(original, got) {
			t.Logf("%s compressed bytes differ only by encoder representation (original=%d output=%d); header and inflated body are exact", input, len(original), len(got))
		}
	}
}

func TestScenarioOfficialCorpusOptIn(t *testing.T) {
	root := strings.TrimSpace(os.Getenv("AOE2KIT_OFFICIAL_SCENARIO_CORPUS"))
	if root == "" {
		t.Skip("set AOE2KIT_OFFICIAL_SCENARIO_CORPUS to run the private official-campaign corpus gate")
	}
	paths := []string{}
	if err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !entry.IsDir() && strings.EqualFold(filepath.Ext(entry.Name()), ".aoe2scenario") {
			paths = append(paths, path)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	sort.Strings(paths)
	if len(paths) == 0 {
		t.Fatalf("official corpus %q contains no scenario files", root)
	}
	unsupported := map[string]int{}
	parsed := 0
	for _, path := range paths {
		version, err := ReadScenarioVersionFile(path)
		if err != nil {
			t.Fatalf("read version %s: %v", path, err)
		}
		if !SupportsReadVersion(version) {
			unsupported[version]++
			continue
		}
		file, err := Open(path)
		if err != nil {
			t.Fatalf("official corpus open %s: %v", path, err)
		}
		if err := file.VerifyRebuild(); err != nil {
			t.Fatalf("official corpus rebuild %s: %v", path, err)
		}
		parsed++
	}
	t.Logf("official corpus gate: files=%d parsed_supported=%d unsupported=%s", len(paths), parsed, fmt.Sprint(unsupported))
}

func TestScenario159PlayerVictoryConditionRepeatFixture(t *testing.T) {
	for _, name := range []string{"Clean v159.8.aoe2scenario", "Clean v159.9.aoe2scenario"} {
		path := filepath.Join("testdata", name)
		file, err := Open(path)
		if err != nil {
			t.Fatalf("open %s: %v", path, err)
		}
		rebuilt, err := file.RebuildBody()
		if err != nil {
			t.Fatalf("rebuild %s: %v", path, err)
		}
		if !bytes.Equal(file.originalBody, rebuilt) {
			i := firstByteDifference(file.originalBody, rebuilt)
			t.Fatalf("%s body changed at byte %d", path, i)
		}
	}
}

func TestScenario159ConditionLayoutFixtures(t *testing.T) {
	for _, name := range []string{"RyanScratch AB2.aoe2scenario", "Clean v159.12.aoe2scenario"} {
		path := filepath.Join("testdata", name)
		file, err := Open(path)
		if err != nil {
			t.Fatalf("open %s: %v", path, err)
		}
		rebuilt, err := file.RebuildBody()
		if err != nil {
			t.Fatalf("rebuild %s: %v", path, err)
		}
		if !bytes.Equal(file.originalBody, rebuilt) {
			t.Fatalf("%s body changed at byte %d", path, firstByteDifference(file.originalBody, rebuilt))
		}
	}
}

func firstByteDifference(a, b []byte) int {
	limit := len(a)
	if len(b) < limit {
		limit = len(b)
	}
	for i := 0; i < limit; i++ {
		if a[i] != b[i] {
			return i
		}
	}
	return limit
}

func byteAt(data []byte, index int) byte {
	if index < 0 || index >= len(data) {
		return 0
	}
	return data[index]
}

func TestScenario159PatchAddsUnitsAndTriggers(t *testing.T) {
	input := scenarioProjectPath("Mandala", "ring_work", "ryanscratch", "RyanScratch.original.aoe2scenario")
	if _, err := os.Stat(input); os.IsNotExist(err) {
		t.Skip("external 1.59 fixture unavailable")
	}
	output := filepath.Join(t.TempDir(), "patched.aoe2scenario")
	active, looping := true, false
	timer, unit, player, attr, operation := 0, 12, 1, 3, 1
	quantity := 1.0
	x, y := 8.5, 11.5
	recipe := Recipe{
		Units: []UnitRecipe{{Op: "add_unit", Player: player, UnitConst: unit, X: &x, Y: &y, Status: intPtr(2)}},
		Triggers: []TriggerRecipe{{
			Op: "add_trigger", Name: "1.59 writer test", Enabled: &active, Looping: &looping,
			Conditions: []ConditionRecipe{{Op: "timer", Timer: &timer}},
			Effects:    []EffectRecipe{{Op: "modify_attribute", SourcePlayer: &player, ObjectListUnitID: &unit, ObjectAttributes: &attr, QuantityFloat: &quantity, Operation: &operation}},
		}},
	}
	if _, err := PatchRecipeFile(input, output, recipe); err != nil {
		t.Fatalf("PatchRecipeFile 1.59: %v", err)
	}
	patched, err := Open(output)
	if err != nil {
		t.Fatalf("open patched 1.59: %v", err)
	}
	if patched.Version != "1.59" || patched.Triggers.Count != 1 || patched.Units.Total != 13 {
		t.Fatalf("patched 1.59 summary: version=%q triggers=%d units=%d", patched.Version, patched.Triggers.Count, patched.Units.Total)
	}
	triggerSection := patched.root.section("Triggers")
	if triggerSection == nil {
		t.Fatal("patched 1.59 missing Triggers section")
	}
	triggers := triggerSection.list("trigger_data")
	if len(triggers) != 1 || len(triggers[0].list("condition_data")) != 1 {
		t.Fatalf("patched 1.59 trigger shape: triggers=%d conditions=%d", len(triggers), len(triggers[0].list("condition_data")))
	}
	condition := triggers[0].list("condition_data")[0]
	editorSpec, err := LoadDESpecForVersion("1.59")
	if err != nil {
		t.Fatalf("load editor 1.59 spec: %v", err)
	}
	patchDE159EditorConditionSpec(editorSpec)
	expectedRaw, err := buildTriggerFromRecipe(editorSpec, recipe.Triggers[0])
	if err != nil {
		t.Fatalf("build editor-equivalent trigger: %v", err)
	}
	triggerSpec, _, _, err := triggerChildSpecs(editorSpec)
	if err != nil {
		t.Fatalf("editor trigger spec: %v", err)
	}
	expectedTrigger, err := (&parser{data: expectedRaw, sections: map[string]*parsedSection{}}).parseNode("TriggerStruct", triggerSpec, nil)
	if err != nil {
		t.Fatalf("parse editor-equivalent trigger: %v", err)
	}
	expectedCondition := expectedTrigger.list("condition_data")[0]
	if !bytes.Equal(condition.raw(), expectedCondition.raw()) {
		t.Fatalf("kit-authored 1.59 condition does not byte-match editor-equivalent condition")
	}
	marker, ok := condition.intValue("static_value_33")
	if !ok || marker != 34 {
		t.Fatalf("1.59 writer condition marker=%d present=%t, want 34", marker, ok)
	}
	opaque, ok := condition.intValue("unknown_1_59_editor_before_xs_function")
	if !ok || opaque != -1 {
		t.Fatalf("1.59 writer opaque condition field=%d present=%t, want -1", opaque, ok)
	}
}

func TestTriggerModifyAttributeOperationNames(t *testing.T) {
	spec, err := LoadCurrentDESpec()
	if err != nil {
		t.Fatal(err)
	}
	player, unit, attr := 1, 12, 3
	quantity := 1.0
	raw, err := buildTriggerFromRecipe(spec, TriggerRecipe{Op: "add_trigger", Name: "named operation", Effects: []EffectRecipe{{
		Op: "modify_attribute", SourcePlayer: &player, ObjectListUnitID: &unit, ObjectAttributes: &attr,
		QuantityFloat: &quantity, OperationName: "set",
	}}})
	if err != nil {
		t.Fatalf("named operation: %v", err)
	}
	triggerSpec, _, err := triggerEffectSpecs(spec)
	if err != nil {
		t.Fatal(err)
	}
	node, err := (&parser{data: raw, sections: map[string]*parsedSection{}}).parseNode("TriggerStruct", triggerSpec, nil)
	if err != nil {
		t.Fatal(err)
	}
	operation, ok := node.list("effect_data")[0].intValue("operation")
	if !ok || operation != 1 {
		t.Fatalf("named set operation=%d present=%t, want 1", operation, ok)
	}
	zero := 0
	_, err = buildTriggerFromRecipe(spec, TriggerRecipe{Op: "add_trigger", Name: "invalid operation", Effects: []EffectRecipe{{
		Op: "modify_attribute", SourcePlayer: &player, ObjectListUnitID: &unit, ObjectAttributes: &attr,
		QuantityFloat: &quantity, Operation: &zero,
	}}})
	if err == nil || !strings.Contains(err.Error(), "trigger operations are SET=1") {
		t.Fatalf("zero operation error=%v", err)
	}
}
