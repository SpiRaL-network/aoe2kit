package replay

import (
	"aoe2kit/pkg/testfixtures"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTauntTextKnownAnchors(t *testing.T) {
	tests := map[int]string{
		1:  "Yes",
		11: "Laugh",
		14: "Start the game",
		62: "Attack player 2!",
		86: "Attack w/ Unique Units",
	}
	for number, want := range tests {
		if got := TauntText(number); got != want {
			t.Fatalf("TauntText(%d) = %q, want %q", number, got, want)
		}
	}
}

func TestLatestCBAPlayerDecodeGolden(t *testing.T) {
	path := testfixtures.Path(t, "save-analysis/latest_cba.aoe2record")
	if _, err := os.Stat(path); err != nil {
		t.Skipf("golden replay not present: %v", err)
	}
	rec, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(rec.Players) != 8 {
		t.Fatalf("players = %d, want 8", len(rec.Players))
	}
	wantCivs := map[int]int{
		4: 6,
		6: 25,
	}
	for playerNumber, wantCiv := range wantCivs {
		player := playerByNumber(rec.Players, playerNumber)
		if player == nil {
			t.Fatalf("missing player %d", playerNumber)
		}
		if player.Civ != wantCiv {
			t.Fatalf("P%d civ = %d, want %d", playerNumber, player.Civ, wantCiv)
		}
	}
	for i, player := range rec.Players {
		if player.Color != i {
			t.Fatalf("slot %d color = %d, want %d", i+1, player.Color, i)
		}
	}
}

func TestRequiemRandomPositionRosterGolden(t *testing.T) {
	path := testfixtures.Path(t, "save-analysis/crawl/e_493919777/AgeIIDE_Replay_493919777.aoe2record")
	if _, err := os.Stat(path); err != nil {
		t.Skipf("golden replay not present: %v", err)
	}
	rec, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(rec.Players), 8; got != want {
		t.Fatalf("players = %d, want %d; ai_parse_error=%q data_set_error=%q", got, want, rec.AIParseErr, rec.DataSet.Error)
	}
	wantColors := []int{0, 1, 7, 4, 2, 6, 3, 5}
	for i, want := range wantColors {
		if rec.Players[i].Color != want {
			t.Fatalf("slot %d color = %d, want %d", i+1, rec.Players[i].Color, want)
		}
	}
	if rec.AIParseErr != "" {
		t.Fatalf("ai parse error = %q", rec.AIParseErr)
	}
	if rec.DataSet.Error != "" {
		t.Fatalf("data-set identity error = %q", rec.DataSet.Error)
	}
}

func TestNoTriggerBlankReplayPartialHeaderGolden(t *testing.T) {
	path := testfixtures.Path(t, "save-analysis/latest_test_20260720_233538.aoe2record")
	if _, err := os.Stat(path); err != nil {
		t.Skipf("golden replay not present: %v", err)
	}
	rec, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if rec.TriggerGraphOK {
		t.Fatalf("blank no-trigger replay unexpectedly has trigger graph")
	}
	if got, want := len(rec.Players), 1; got != want {
		t.Fatalf("players = %d, want %d", got, want)
	}
	if rec.Players[0].Name != "chrae" || rec.Players[0].Number != 1 || !rec.Players[0].Human {
		t.Fatalf("player decode = %#v, want P1 human chrae", rec.Players[0])
	}
	report, err := BuildCoverage(path)
	if err != nil {
		t.Fatal(err)
	}
	foundString := false
	for _, region := range report.Regions {
		if region.Name == "scenario_string_000" {
			foundString = true
			break
		}
	}
	if !foundString {
		t.Fatalf("no-trigger coverage did not expose scenario string islands")
	}
	if report.Summary.BodyOpaqueBytes != 0 {
		t.Fatalf("body opaque bytes = %d, want 0", report.Summary.BodyOpaqueBytes)
	}
	scan, err := BuildValueScan(path, ValueScanOptions{Text: []string{"Blank Test.aoe2scenario"}, Limit: 8})
	if err != nil {
		t.Fatal(err)
	}
	if scan.Summary.HitCount == 0 {
		t.Fatalf("scan-value did not find blank scenario path string")
	}
}

func TestTinyTriggerReplayGraphGolden(t *testing.T) {
	path := testfixtures.Path(t, "save-analysis/latest_20260721_012221.aoe2record")
	if _, err := os.Stat(path); err != nil {
		t.Skipf("golden replay not present: %v", err)
	}
	rec, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if !rec.TriggerGraphOK || rec.TriggerGraph == nil {
		t.Fatalf("trigger graph missing: %s", rec.TriggerGraphErr)
	}
	if rec.TriggerGraph.TriggerCount != 3 || rec.TriggerGraph.EffectCount != 1 || rec.TriggerGraph.ConditionCount != 1 {
		t.Fatalf("trigger graph counts = %d/%d/%d, want 3/1/1", rec.TriggerGraph.TriggerCount, rec.TriggerGraph.EffectCount, rec.TriggerGraph.ConditionCount)
	}
	if rec.TriggerGraph.SHA256 != "6817db40016c4ac1ebebb612f50c69217489b947c4500ea471382af60cddeaf9" {
		t.Fatalf("trigger graph hash = %s", rec.TriggerGraph.SHA256)
	}
	report, err := BuildCoverage(path)
	if err != nil {
		t.Fatal(err)
	}
	foundGraph := false
	for _, region := range report.Regions {
		if region.Name == "trigger_graph" && region.Status == "decoded" {
			foundGraph = true
			break
		}
	}
	if !foundGraph {
		t.Fatalf("coverage did not mark binary-scanned trigger graph decoded")
	}
	scan, err := BuildValueScan(path, ValueScanOptions{Text: []string{"Watermelon"}, Limit: 8})
	if err != nil {
		t.Fatal(err)
	}
	foundWatermelon := false
	for _, hit := range scan.Hits {
		if hit.Space == "inflated_header" {
			foundWatermelon = true
			break
		}
	}
	if !foundWatermelon {
		t.Fatalf("scan-value did not find Watermelon in inflated header")
	}
}

func TestVER94ConsolidatedTriggerGraphGolden(t *testing.T) {
	path := testfixtures.Path(t, "VER9.4_consolidated_triggers_PARSEFAIL.aoe2record")
	if _, err := os.Stat(path); err != nil {
		t.Skipf("golden replay not present: %v", err)
	}
	rec, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if !rec.TriggerGraphOK || rec.TriggerGraph == nil {
		t.Fatalf("trigger graph missing: %s", rec.TriggerGraphErr)
	}
	if rec.TriggerGraph.TriggerCount != 3 || rec.TriggerGraph.EffectCount != 1873 || rec.TriggerGraph.ConditionCount != 8 {
		t.Fatalf("trigger graph counts = %d/%d/%d, want 3/1873/8",
			rec.TriggerGraph.TriggerCount, rec.TriggerGraph.EffectCount, rec.TriggerGraph.ConditionCount)
	}
	if rec.TriggerGraph.SHA256 != "084f05e39075acfbf3d51d00a6b4571d94c8a88973569edbef6430493921e54f" {
		t.Fatalf("trigger graph hash = %s", rec.TriggerGraph.SHA256)
	}
	if len(rec.TriggerGraph.Triggers) == 0 {
		t.Fatalf("trigger graph has no trigger rows")
	}
	effects, _ := rec.TriggerGraph.Triggers[0]["effects"].([]map[string]any)
	if len(effects) < 3 {
		t.Fatalf("first consolidated trigger effects = %d, want at least 3", len(effects))
	}
	if got := effects[0]["type_name"]; got != "change_variable" {
		t.Fatalf("first effect type_name = %v, want change_variable", got)
	}
	if got := effects[1]["type_name"]; got != "script_call" {
		t.Fatalf("second effect type_name = %v, want script_call", got)
	}
	if got := effects[2]["type_name"]; got != "send_chat" {
		t.Fatalf("third effect type_name = %v, want send_chat", got)
	}
	if msg, _ := effects[2]["message"].(string); !strings.Contains(msg, "BASE NUKE purchased") {
		t.Fatalf("third effect message = %q, want BASE NUKE purchase text", msg)
	}
}

func TestCurrent159EmbeddedReplayTriggerGraph(t *testing.T) {
	path := filepath.Join("..", "..", "..", "..", "Mandala", "ring_work", "ryanscratch", "scratch.aoe2record")
	if _, err := os.Stat(path); err != nil {
		t.Skipf("external current 1.59 replay unavailable: %v", err)
	}
	rec, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if !rec.TriggerGraphOK || rec.TriggerGraph == nil {
		t.Fatalf("1.59 embedded trigger graph missing: %s", rec.TriggerGraphErr)
	}
	if rec.TriggerGraph.TriggerCount != 4 || rec.TriggerGraph.EffectCount != 4 || rec.TriggerGraph.ConditionCount != 4 {
		t.Fatalf("1.59 graph counts = %d/%d/%d, want 4/4/4", rec.TriggerGraph.TriggerCount, rec.TriggerGraph.EffectCount, rec.TriggerGraph.ConditionCount)
	}
	for _, name := range []string{
		"Scratch P1 barracks attr 3 = 1.0",
		"Scratch P1 barracks attr 4 = 1.0",
		"Scratch P1 barracks attr 200 = 1.0",
		"Scratch P1 barracks attr 201 = 1.0",
	} {
		if !triggerGraphHasName(rec.TriggerGraph.Triggers, name) {
			t.Fatalf("1.59 graph missing %q", name)
		}
	}
}

func TestKrmythReplayTriggerGraphCompletesPastHealTriggers(t *testing.T) {
	path := testfixtures.Path(t, "save-analysis/diagnostics/latest_krmyth_pull_20260904_055048/latest.aoe2record")
	if _, err := os.Stat(path); err != nil {
		t.Skipf("golden replay not present: %v", err)
	}
	rec, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if !rec.TriggerGraphOK || rec.TriggerGraph == nil {
		t.Fatalf("trigger graph missing: %s", rec.TriggerGraphErr)
	}
	if rec.TriggerGraph.TriggerCount != 2641 || rec.TriggerGraph.EffectCount != 30407 || rec.TriggerGraph.ConditionCount != 5866 {
		t.Fatalf("trigger graph counts = %d/%d/%d, want 2641/30407/5866",
			rec.TriggerGraph.TriggerCount, rec.TriggerGraph.EffectCount, rec.TriggerGraph.ConditionCount)
	}
	if rec.TriggerGraph.SHA256 != "25a73b782f5c71c8ec6ff2aef6774e607d1ef461667e27d61b7650855005e464" {
		t.Fatalf("trigger graph hash = %s", rec.TriggerGraph.SHA256)
	}
	for _, name := range []string{"Heal Spell Ready P2", "E (Expl) WV Begin", "TO DO (NOTE)"} {
		if !triggerGraphHasName(rec.TriggerGraph.Triggers, name) {
			t.Fatalf("trigger graph missing %q", name)
		}
	}
	for _, warning := range rec.TriggerGraph.Warnings {
		if strings.Contains(warning, "could not find effect boundary") || strings.Contains(warning, "fallback scan") {
			t.Fatalf("trigger graph retained truncation warning: %q", warning)
		}
	}
}

func TestNumericValueQueries(t *testing.T) {
	queries, warnings := compileValueQueries(ValueScanOptions{Values: []string{"123456789"}})
	if len(warnings) != 0 {
		t.Fatalf("warnings = %#v", warnings)
	}
	found32 := false
	found64 := false
	for _, query := range queries {
		if query.Kind == "int32le" && query.Pattern == "15cd5b07" {
			found32 = true
		}
		if query.Kind == "int64le" && query.Pattern == "15cd5b0700000000" {
			found64 = true
		}
	}
	if !found32 || !found64 {
		t.Fatalf("queries = %#v, want int32/int64 little-endian forms", queries)
	}
}

func triggerGraphHasName(triggers []map[string]any, name string) bool {
	for _, trigger := range triggers {
		if got, _ := trigger["name"].(string); got == name {
			return true
		}
	}
	return false
}

func TestLatestCBAHeaderSpineGolden(t *testing.T) {
	path := testfixtures.Path(t, "save-analysis/latest_cba.aoe2record")
	if _, err := os.Stat(path); err != nil {
		t.Skipf("golden replay not present: %v", err)
	}
	report, err := BuildCoverage(path)
	if err != nil {
		t.Fatal(err)
	}
	if report.HeaderSpine == nil {
		t.Fatalf("header spine missing")
	}
	if got, want := len(report.HeaderSpine.Players), 9; got != want {
		t.Fatalf("initial players = %d, want %d", got, want)
	}
	if got, want := report.HeaderSpine.InitialStart, 261665; got != want {
		t.Fatalf("initial start = %d, want %d", got, want)
	}
	if got, want := report.HeaderSpine.InitialEnd, 8479940; got != want {
		t.Fatalf("initial end = %d, want %d", got, want)
	}
	if got, want := report.HeaderSpine.TriggerStart, 9242986; got != want {
		t.Fatalf("trigger start = %d, want %d", got, want)
	}
	if report.HeaderSpine.Players[4].Name != "chrae" {
		t.Fatalf("initial player 4 name = %q, want chrae", report.HeaderSpine.Players[4].Name)
	}
	p1 := report.HeaderSpine.Players[1]
	if p1.AttributesBytes != 4848 {
		t.Fatalf("P1 attributes bytes = %d, want 4848", p1.AttributesBytes)
	}
	if p1.CivilizationKey != "BENGALIS-CIV" {
		t.Fatalf("P1 civilization key = %q, want BENGALIS-CIV", p1.CivilizationKey)
	}
	if p1.SpawnX != 4 || p1.SpawnY != 15 {
		t.Fatalf("P1 spawn = %d,%d want 4,15", p1.SpawnX, p1.SpawnY)
	}
	if p1.ObjectSpanStart != 1312057 || p1.ObjectSpanBytes != 989142 {
		t.Fatalf("P1 object span = %d bytes=%d, want 1312057 bytes=989142", p1.ObjectSpanStart, p1.ObjectSpanBytes)
	}
	if p1.ObjectTailMarkerStart != 1312057 || p1.ObjectTailMarkerEnd != 1312059 {
		t.Fatalf("P1 object tail marker = %d..%d, want 1312057..1312059", p1.ObjectTailMarkerStart, p1.ObjectTailMarkerEnd)
	}
	if p1.ObjectCandidateBandStart != 2220265 || p1.ObjectCandidateBandEnd != 2300373 || p1.ObjectCandidateCount != 93 {
		t.Fatalf("P1 object candidate band = %d..%d count=%d, want 2220265..2300373 count=93", p1.ObjectCandidateBandStart, p1.ObjectCandidateBandEnd, p1.ObjectCandidateCount)
	}
	if len(report.Warnings) != 0 {
		t.Fatalf("coverage warnings = %#v", report.Warnings)
	}
	if report.Summary.BodyOpaqueBytes != 0 {
		t.Fatalf("body opaque bytes = %d, want 0", report.Summary.BodyOpaqueBytes)
	}
	if _, err := json.Marshal(report); err != nil {
		t.Fatalf("coverage report must marshal as JSON: %v", err)
	}
	if report.Summary.HeaderDecodedBytes != 16898062 || report.Summary.HeaderOpaqueBytes != 1276870 {
		t.Fatalf("header coverage = decoded %d opaque %d, want 16898062/1276870", report.Summary.HeaderDecodedBytes, report.Summary.HeaderOpaqueBytes)
	}
	if !hasCoverageRegion(report, "initial_player_1_effective_gamedata_template_reference", "decoded", "effective_gamedata_shared_template_reference_measured") {
		t.Fatalf("missing effective game-data template reference region")
	}
	if !hasCoverageRegion(report, "initial_player_2_effective_gamedata_unit_availability_array_0000", "decoded", "effective_gamedata_unit_availability_array_mapped_to_dat_unit_slots") {
		t.Fatalf("missing P2 effective game-data unit availability region")
	}
	if !hasCoverageRegion(report, "initial_player_2_effective_gamedata_diff_run_0000", "decoded", "effective_gamedata_u16_diff_run_measured_vs_reference_semantics_partial") {
		t.Fatalf("missing P2 effective game-data measured diff region")
	}
	if !hasCoverageRegion(report, "initial_player_8_effective_gamedata_template_common_0000", "decoded", "effective_gamedata_template_common_vs_reference_measured") {
		t.Fatalf("missing P8 effective game-data partition region")
	}
	if !hasCoverageRegion(report, "initial_player_0_object_tail_before_candidate_prefixes_gap_0006", "decoded", "duplicate_of_effective_gamedata_template_subrange") {
		t.Fatalf("missing promoted effective-template duplicate region")
	}
	if !hasCoverageRegion(report, "v68_tail_before_trigger_graph_de_string_0000", "decoded", "parsed_de_string_record_inside_bounded_opaque_header_region") {
		t.Fatalf("missing decoded v68 tail DE string island")
	}
	if !hasCoverageRegion(report, "v68_tail_before_trigger_graph_gap_1790_scenario_string_0000", "decoded", "parsed_scenario_string_record_inside_bounded_opaque_header_region") {
		t.Fatalf("missing decoded bounded scenario string island")
	}
	if !hasCoverageRegion(report, "v68_tail_before_trigger_graph_gap_1789_plain_string_0000", "decoded", "parsed_u16_length_ascii_string_inside_bounded_opaque_header_region") {
		t.Fatalf("missing decoded bounded u16 plain string island")
	}
	if !hasCoverageRegion(report, "v68_tail_before_trigger_graph_gap_1789_gap_0005_ai_load_0000", "decoded", "parsed_ai_load_directive_inside_bounded_opaque_header_region") {
		t.Fatalf("missing decoded Promisory AI load directive island")
	}
	if !hasCoverageRegion(report, "v68_tail_before_trigger_graph_gap_1789_gap_0005_gap_0001", "decoded", "parsed_ai_script_text_fragment_inside_bounded_opaque_header_region") {
		t.Fatalf("missing decoded Promisory AI script text fragment")
	}
	if !hasCoverageRegion(report, "v68_tail_before_trigger_graph_gap_1789_gap_0005_gap_0198_xs_include_0000", "decoded", "parsed_xs_include_directive_inside_bounded_opaque_header_region") {
		t.Fatalf("missing decoded XS include directive island")
	}
	if !hasCoverageRegion(report, "v68_tail_before_trigger_graph_gap_1789_gap_0005_gap_0000_promide_name_table_0000", "decoded", "parsed_promide_name_table_inside_bounded_opaque_header_region") {
		t.Fatalf("missing decoded PromiDE name table")
	}
	if !hasCoverageRegion(report, "v68_tail_before_trigger_graph_gap_1789_gap_0005_gap_0022_gap_0001", "decoded", "parsed_ai_script_block_separator_inside_bounded_opaque_header_region") {
		t.Fatalf("missing decoded AI script block separator")
	}
}

func hasCoverageRegion(report *CoverageReport, name string, status string, confidence string) bool {
	for _, region := range report.Regions {
		if region.Name == name && region.Status == status && region.Confidence == confidence {
			return true
		}
	}
	return false
}

func TestLatestCBAOpaqueSpansGolden(t *testing.T) {
	path := testfixtures.Path(t, "save-analysis/latest_cba.aoe2record")
	if _, err := os.Stat(path); err != nil {
		t.Skipf("golden replay not present: %v", err)
	}
	report, err := BuildOpaqueSpans(path, OpaqueSpanOptions{Limit: 8})
	if err != nil {
		t.Fatal(err)
	}
	if report.Summary.HeaderOpaqueBytes != 1276870 || report.Summary.BodyOpaqueBytes != 0 {
		t.Fatalf("opaque summary = header %d body %d, want 1276870/0", report.Summary.HeaderOpaqueBytes, report.Summary.BodyOpaqueBytes)
	}
	if report.Summary.SpanCount == 0 || report.Summary.Shown == 0 {
		t.Fatalf("opaque spans not shown: count=%d shown=%d", report.Summary.SpanCount, report.Summary.Shown)
	}
	if report.Spans[0].Space != "inflated_header" || report.Spans[0].Bytes <= 0 {
		t.Fatalf("top span = %#v, want inflated_header with bytes", report.Spans[0])
	}
	if report.Spans[0].HexSample == "" || report.Spans[0].Entropy <= 0 {
		t.Fatalf("top span profile missing sample/entropy: %#v", report.Spans[0])
	}
}

func TestLatestCBAAIManifestGolden(t *testing.T) {
	path := testfixtures.Path(t, "save-analysis/latest_cba.aoe2record")
	if _, err := os.Stat(path); err != nil {
		t.Skipf("golden replay not present: %v", err)
	}
	report, err := BuildAIManifest(path, AIManifestOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if report.Summary.PromiDEMentions != 9 {
		t.Fatalf("PromiDE mentions = %d, want 9", report.Summary.PromiDEMentions)
	}
	if report.Summary.LoadDirectives != 198 || report.Summary.UniqueModules != 22 {
		t.Fatalf("AI manifest directives/modules = %d/%d, want 198/22", report.Summary.LoadDirectives, report.Summary.UniqueModules)
	}
	if report.Summary.XSIncludes != 9 || !hasAIManifestXSInclude(report, "ailib/Geometry.xs") {
		t.Fatalf("XS includes = count %d rows %#v, want 9 ailib/Geometry.xs references", report.Summary.XSIncludes, report.XSIncludes)
	}
	if report.PromisoryRootOK && report.Summary.ResolvedModules != 22 {
		t.Fatalf("resolved modules = %d, want 22 when reference root exists", report.Summary.ResolvedModules)
	}
	if !hasAIManifestModule(report, "defaultConstants") || !hasAIManifestModule(report, "scoutcontrol") || !hasAIManifestModule(report, "watercontrol") {
		t.Fatalf("AI manifest missing expected Promisory modules: %#v", report.UniqueModules)
	}
}

func hasAIManifestModule(report *AIManifestReport, name string) bool {
	for _, module := range report.UniqueModules {
		if module.Name == name {
			return true
		}
	}
	return false
}

func hasAIManifestXSInclude(report *AIManifestReport, path string) bool {
	for _, include := range report.XSIncludes {
		if include.IncludePath == path {
			return true
		}
	}
	return false
}

func TestDiffRunsCoalescesNearbyChanges(t *testing.T) {
	before := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	after := []byte{1, 2, 8, 4, 5, 6, 7, 9, 9, 10, 11}
	runs := diffRuns("body", before, after)
	if len(runs) != 1 {
		t.Fatalf("runs = %#v, want one coalesced run", runs)
	}
	if runs[0].space != "body" || runs[0].start != 2 || runs[0].end != 11 {
		t.Fatalf("run = %#v, want body 2..11", runs[0])
	}
}

func TestLatestCBAObjectIndexGolden(t *testing.T) {
	path := testfixtures.Path(t, "save-analysis/latest_cba.aoe2record")
	if _, err := os.Stat(path); err != nil {
		t.Skipf("golden replay not present: %v", err)
	}
	report, err := BuildObjectIndex(path, ObjectIndexOptions{ReferencedOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	if report.Summary.ReferencedCandidates < 100 {
		t.Fatalf("referenced candidates = %d, want at least 100", report.Summary.ReferencedCandidates)
	}
	if report.Summary.DecodedPrefixBytes != 9840 || report.Summary.BoundedBodyBytes != 102247 {
		t.Fatalf("object byte accounting = prefix %d body %d, want 9840/102247", report.Summary.DecodedPrefixBytes, report.Summary.BoundedBodyBytes)
	}
	if report.Summary.DecodedBodyPrefixBytes != 102097 {
		t.Fatalf("object decoded body prefix bytes = %d, want 102097", report.Summary.DecodedBodyPrefixBytes)
	}
	if report.Summary.OpaqueBodyBytes != 150 {
		t.Fatalf("object opaque body bytes = %d, want 150", report.Summary.OpaqueBodyBytes)
	}
	building := objectCandidateByID(report.Objects, 906)
	if building == nil {
		t.Fatalf("object 906 not indexed")
	}
	if building.OwnerID != 3 || building.UnitID != 209 || building.Class != "building" {
		t.Fatalf("object 906 = owner %d unit %d class %s, want P3 unit 209 building", building.OwnerID, building.UnitID, building.Class)
	}
	gate := objectCandidateByID(report.Objects, 5252)
	if gate == nil {
		t.Fatalf("gate object 5252 not indexed")
	}
	if gate.OwnerID != 6 || gate.UnitID != 88 || gate.Class != "gate" || gate.TargetRefs == 0 {
		t.Fatalf("object 5252 = owner %d unit %d class %s target_refs %d, want P6 unit 88 gate with target refs", gate.OwnerID, gate.UnitID, gate.Class, gate.TargetRefs)
	}
	if gate.PrefixBytes != 80 || gate.BodyBytes != 778 || gate.EndConfidence != "bounded_by_next_candidate_prefix" {
		t.Fatalf("object 5252 span = prefix %d body %d end %q, want 80/778 bounded_by_next_candidate_prefix", gate.PrefixBytes, gate.BodyBytes, gate.EndConfidence)
	}
}

func TestCBA2ObjectIndexJSONMarshalGolden(t *testing.T) {
	path := testfixtures.Path(t, "save-analysis/diagnostics/cba2.aoe2record")
	if _, err := os.Stat(path); err != nil {
		t.Skipf("golden replay not present: %v", err)
	}
	report, err := BuildObjectIndex(path, ObjectIndexOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := json.Marshal(report); err != nil {
		t.Fatalf("object index must marshal as JSON after non-finite sanitization: %v", err)
	}
	if len(report.Warnings) == 0 {
		t.Fatalf("expected non-finite object-candidate sanitization warning on cba2")
	}
}

func TestLatestCBAObjectShapesGolden(t *testing.T) {
	path := testfixtures.Path(t, "save-analysis/latest_cba.aoe2record")
	if _, err := os.Stat(path); err != nil {
		t.Skipf("golden replay not present: %v", err)
	}
	report, err := BuildObjectShapes(path, ObjectIndexOptions{ReferencedOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	if report.Summary.Candidates != 123 || report.Summary.ShapeCount != 25 {
		t.Fatalf("shape summary = candidates %d shapes %d, want 123/25", report.Summary.Candidates, report.Summary.ShapeCount)
	}
	if report.Summary.MinRecordBytes != 717 || report.Summary.MaxRecordBytes != 1920 || report.Summary.FinalBodyShapeCount != 2 {
		t.Fatalf("shape byte summary = min %d max %d final %d, want 717/1920/2", report.Summary.MinRecordBytes, report.Summary.MaxRecordBytes, report.Summary.FinalBodyShapeCount)
	}
	if report.Summary.DecodedPrefixBytes != 9840 || report.Summary.BoundedBodyBytes != 102247 {
		t.Fatalf("shape byte accounting = prefix %d body %d, want 9840/102247", report.Summary.DecodedPrefixBytes, report.Summary.BoundedBodyBytes)
	}
	if report.Summary.DecodedBodyPrefixBytes != 102097 {
		t.Fatalf("shape decoded body prefix bytes = %d, want 102097", report.Summary.DecodedBodyPrefixBytes)
	}
	if report.Summary.OpaqueBodyBytes != 150 {
		t.Fatalf("shape opaque body bytes = %d, want 150", report.Summary.OpaqueBodyBytes)
	}
	if len(report.Shapes) == 0 {
		t.Fatalf("no shapes")
	}
	top := report.Shapes[0]
	if top.RecordType != 80 || top.UnitID != 82 || top.RecordBytes != 858 || top.Count != 26 || top.Referenced != 26 {
		t.Fatalf("top shape = type %d unit %d bytes %d count %d referenced %d, want type80 unit82 bytes858 count26 referenced26", top.RecordType, top.UnitID, top.RecordBytes, top.Count, top.Referenced)
	}
	if top.DecodedBodyPrefixEach != 778 || top.OpaqueBodyEach != 0 {
		t.Fatalf("top shape body decode = decoded each %d opaque each %d, want 778/0", top.DecodedBodyPrefixEach, top.OpaqueBodyEach)
	}
}

func TestLatestCBAObjectStateGolden(t *testing.T) {
	path := testfixtures.Path(t, "save-analysis/latest_cba.aoe2record")
	if _, err := os.Stat(path); err != nil {
		t.Skipf("golden replay not present: %v", err)
	}
	report, err := BuildObjectState(path, ObjectStateOptions{ReferencedOnly: true, Class: "gate"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := json.Marshal(report); err != nil {
		t.Fatalf("object-state report must marshal as JSON: %v", err)
	}
	if got, want := report.Summary.Gates, 18; got != want {
		t.Fatalf("gates = %d, want %d", got, want)
	}
	if report.Summary.ActionPrefixDecoded != 18 || report.Summary.CombatTailDecoded != 18 || report.Summary.BuildingPrefixDecoded != 18 {
		t.Fatalf("decoded gate prefixes = action %d combat %d building %d, want 18/18/18", report.Summary.ActionPrefixDecoded, report.Summary.CombatTailDecoded, report.Summary.BuildingPrefixDecoded)
	}
	if report.Summary.DecodedBodyPrefixBytes != 102097 || report.Summary.OpaqueBodyBytes != 150 {
		t.Fatalf("state byte summary = decoded %d opaque %d, want 102097/150", report.Summary.DecodedBodyPrefixBytes, report.Summary.OpaqueBodyBytes)
	}
	gate := objectStateByID(report.Objects, 5252)
	if gate == nil {
		t.Fatalf("gate object 5252 not found")
	}
	if gate.OwnerID != 6 || gate.UnitID != 88 || gate.Class != "gate" || gate.TargetRefs != 5 {
		t.Fatalf("gate 5252 = owner %d unit %d class %s target_refs %d, want P6 unit 88 gate with 5 target refs", gate.OwnerID, gate.UnitID, gate.Class, gate.TargetRefs)
	}
	if gate.Building == nil || len(gate.Building.LinkedObjectIDs) != 2 {
		t.Fatalf("gate 5252 building/v68 linkage = %#v, want two linked object ids", gate.Building)
	}
	if gate.Coverage.OpaqueBodyBytes != 0 {
		t.Fatalf("gate 5252 opaque body bytes = %d, want 0", gate.Coverage.OpaqueBodyBytes)
	}
}

func TestLatestCBAEventObjectEnrichmentGolden(t *testing.T) {
	path := testfixtures.Path(t, "save-analysis/latest_cba.aoe2record")
	if _, err := os.Stat(path); err != nil {
		t.Skipf("golden replay not present: %v", err)
	}
	report, err := ExtractEvents(path, EventOptions{IncludeUntypedAction: true, IncludeObjectIndex: true})
	if err != nil {
		t.Fatal(err)
	}
	var gateOrder *ReplayEvent
	for i := range report.Events {
		if report.Events[i].Type == "order" && report.Events[i].TargetID == 5252 {
			gateOrder = &report.Events[i]
			break
		}
	}
	if gateOrder == nil {
		t.Fatalf("no order targeting object 5252")
	}
	if gateOrder.TargetObject == nil {
		t.Fatalf("target object enrichment missing for gate order")
	}
	if gateOrder.TargetObject.OwnerID != 6 || gateOrder.TargetObject.UnitID != 88 || gateOrder.TargetObject.Class != "gate" {
		t.Fatalf("target object = owner %d unit %d class %s, want P6 unit 88 gate", gateOrder.TargetObject.OwnerID, gateOrder.TargetObject.UnitID, gateOrder.TargetObject.Class)
	}
}

func TestLatestCBASpawnCatalogGolden(t *testing.T) {
	path := testfixtures.Path(t, "save-analysis/latest_cba.aoe2record")
	if _, err := os.Stat(path); err != nil {
		t.Skipf("golden replay not present: %v", err)
	}
	report, err := BuildSpawnCatalog(path)
	if err != nil {
		t.Fatal(err)
	}
	if report.Summary.CreateObjectEffects != 2044 || report.Summary.PlausibleSpawns != 2044 {
		t.Fatalf("spawn summary = create %d plausible %d, want 2044/2044", report.Summary.CreateObjectEffects, report.Summary.PlausibleSpawns)
	}
	if report.Summary.UnitTypes != 66 {
		t.Fatalf("unit types = %d, want 66", report.Summary.UnitTypes)
	}
	spawn := spawnRecipeFor(report.Spawns, 1, 83, 28, 11)
	if spawn == nil {
		t.Fatalf("missing P1 unit 83 spawn at 28,11")
	}
	if spawn.UnitName != "Villager Male" {
		t.Fatalf("unit 83 name = %q, want Villager Male", spawn.UnitName)
	}
}

func TestLatestCBALifecycleVillagerGolden(t *testing.T) {
	path := testfixtures.Path(t, "save-analysis/latest_cba.aoe2record")
	if _, err := os.Stat(path); err != nil {
		t.Skipf("golden replay not present: %v", err)
	}
	report, err := BuildLifecycle(path, LifecycleOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if report.Summary.FirstSeenObjects < 1000 {
		t.Fatalf("first seen objects = %d, want at least 1000", report.Summary.FirstSeenObjects)
	}
	if report.Summary.VillagerCandidates < 3 {
		t.Fatalf("villager candidates = %d, want at least 3", report.Summary.VillagerCandidates)
	}
	if report.Summary.OvermatchedNearVillagerEvents == 0 {
		t.Fatalf("overmatched near-villager events = 0, want nonzero ambiguity guard")
	}
	candidate := lifecycleCandidateByID(report.Candidates, 33147)
	if candidate == nil {
		t.Fatalf("missing P1 villager candidate object 33147")
	}
	if candidate.PlayerID != 1 || candidate.UnitID != 83 || candidate.UnitName != "Villager Male" || candidate.Kind != "villager_spawn_candidate" {
		t.Fatalf("candidate 33147 = p%d unit %d name %q kind %q, want P1 unit 83 Villager Male villager_spawn_candidate", candidate.PlayerID, candidate.UnitID, candidate.UnitName, candidate.Kind)
	}
	if candidate.Time != "00:00:58.540" {
		t.Fatalf("candidate 33147 time = %s, want 00:00:58.540", candidate.Time)
	}
}

func TestLatestCBACombatGolden(t *testing.T) {
	path := testfixtures.Path(t, "save-analysis/latest_cba.aoe2record")
	if _, err := os.Stat(path); err != nil {
		t.Skipf("golden replay not present: %v", err)
	}
	report, err := BuildCombatStory(path, CombatOptions{WindowMS: 30000, MinNetLoss: 25})
	if err != nil {
		t.Fatal(err)
	}
	if report.Verification != combatVerification {
		t.Fatalf("verification = %q", report.Verification)
	}
	if report.Summary.ChecksumSamples != 53 || report.Summary.LossPulses != 96 || report.Summary.NetObjectsLost != 10452 {
		t.Fatalf("combat summary = samples %d pulses %d lost %d, want 53/96/10452",
			report.Summary.ChecksumSamples, report.Summary.LossPulses, report.Summary.NetObjectsLost)
	}
	if report.Summary.PressureLinks != 51 || report.Summary.PlayersWithLossPulses != 8 {
		t.Fatalf("combat pressure/players = %d/%d, want 51/8", report.Summary.PressureLinks, report.Summary.PlayersWithLossPulses)
	}
	if report.Summary.Targets != 62 {
		t.Fatalf("combat targets = %d, want 62", report.Summary.Targets)
	}
	if len(report.Targets) == 0 {
		t.Fatalf("missing combat targets")
	}
	target := report.Targets[0]
	if target.TargetObjectID != 7981 || target.TargetOwnerID != 6 || target.TargetUnitID != 82 || target.NetObjectsLostNearby != 785 {
		t.Fatalf("top combat target = object %d owner P%d unit %d nearby lost %d, want object 7981 P6 Castle 785",
			target.TargetObjectID, target.TargetOwnerID, target.TargetUnitID, target.NetObjectsLostNearby)
	}
	if len(report.LossPulses) == 0 {
		t.Fatalf("missing combat loss pulses")
	}
	first := report.LossPulses[0]
	if first.PlayerID != 6 || first.NetObjectsLost != 623 || first.ToTime != "00:34:50.364" {
		t.Fatalf("first combat pulse = P%d lost %d to %s, want P6 lost 623 to 00:34:50.364", first.PlayerID, first.NetObjectsLost, first.ToTime)
	}
	if first.CandidatePressureNum != 3 || len(first.CandidatePressure) != 3 {
		t.Fatalf("first combat pulse pressure = %d/%d, want 3", first.CandidatePressureNum, len(first.CandidatePressure))
	}
	if first.CandidatePressure[1].PlayerID != 3 || first.CandidatePressure[1].TargetUnitID != 82 {
		t.Fatalf("first combat pressure[1] = attacker P%d target unit %d, want P3 Castle", first.CandidatePressure[1].PlayerID, first.CandidatePressure[1].TargetUnitID)
	}
}

func TestSortCombatPulses(t *testing.T) {
	pulses := []CombatLossPulse{
		{PlayerID: 3, ToTimeMS: 3000, NetObjectsLost: 2},
		{PlayerID: 2, ToTimeMS: 2000, NetObjectsLost: 9},
		{PlayerID: 1, ToTimeMS: 1000, NetObjectsLost: 7},
	}
	sortCombatPulses(pulses, "loss")
	if pulses[0].PlayerID != 2 || pulses[1].PlayerID != 1 || pulses[2].PlayerID != 3 {
		t.Fatalf("loss sort order = P%d,P%d,P%d, want P2,P1,P3", pulses[0].PlayerID, pulses[1].PlayerID, pulses[2].PlayerID)
	}
	sortCombatPulses(pulses, "time")
	if pulses[0].PlayerID != 1 || pulses[1].PlayerID != 2 || pulses[2].PlayerID != 3 {
		t.Fatalf("time sort order = P%d,P%d,P%d, want P1,P2,P3", pulses[0].PlayerID, pulses[1].PlayerID, pulses[2].PlayerID)
	}
}

func TestLatestCBAPostgameGolden(t *testing.T) {
	path := testfixtures.Path(t, "save-analysis/latest_cba.aoe2record")
	if _, err := os.Stat(path); err != nil {
		t.Skipf("golden replay not present: %v", err)
	}
	report, err := BuildPostgame(path)
	if err != nil {
		t.Fatal(err)
	}
	if !report.Summary.Op6Seen || report.DE == nil {
		t.Fatalf("DE op=6 postgame not found")
	}
	if report.Summary.Op6TailBytes != 86 || report.Summary.Op6Blocks != 2 {
		t.Fatalf("op6 summary = tail %d blocks %d, want 86/2", report.Summary.Op6TailBytes, report.Summary.Op6Blocks)
	}
	if !report.Summary.HasWorldTimeBlock || !report.Summary.HasLeaderboardBlock {
		t.Fatalf("postgame blocks = world_time %t leaderboard %t, want both", report.Summary.HasWorldTimeBlock, report.Summary.HasLeaderboardBlock)
	}
	if report.DE.WorldTimeMS != 2526162 {
		t.Fatalf("world time = %d, want 2526162", report.DE.WorldTimeMS)
	}
	if report.Summary.HasPlayerKills || report.Summary.Action255Seen {
		t.Fatalf("unexpected per-player postgame stats: kills=%t action255=%t", report.Summary.HasPlayerKills, report.Summary.Action255Seen)
	}
}

func TestBodyCoverageSimpleStream(t *testing.T) {
	var body bytes.Buffer
	writeCoverageU32(&body, 500)
	body.Write(make([]byte, 20))
	writeCoverageU32(&body, 2)
	writeCoverageU32(&body, 1234)
	writeCoverageU32(&body, 6)

	report := &CoverageReport{BodyOps: map[int]int{}}
	report.coverBody(body.Bytes())

	if report.BodyOps[2] != 1 || report.BodyOps[6] != 1 {
		t.Fatalf("body ops = %#v, want sync and postgame", report.BodyOps)
	}
	foundPostgame := false
	for _, region := range report.Regions {
		if region.Name == "op_postgame_marker" && region.Status == "decoded" {
			foundPostgame = true
		}
	}
	if !foundPostgame {
		t.Fatalf("postgame coverage region missing: %#v", report.Regions)
	}
}

func spawnRecipeFor(spawns []SpawnRecipe, playerID int, unitID int, x int, y int) *SpawnRecipe {
	for i := range spawns {
		if spawns[i].TargetPlayer == playerID && spawns[i].UnitID == unitID && spawns[i].X == x && spawns[i].Y == y {
			return &spawns[i]
		}
	}
	return nil
}

func objectCandidateByID(objects []ObjectCandidate, id int) *ObjectCandidate {
	for i := range objects {
		if objects[i].ObjectID == id {
			return &objects[i]
		}
	}
	return nil
}

func objectStateByID(objects []ObjectStateCard, id int) *ObjectStateCard {
	for i := range objects {
		if objects[i].ObjectID == id {
			return &objects[i]
		}
	}
	return nil
}

func lifecycleCandidateByID(candidates []LifecycleObjectCandidate, id int) *LifecycleObjectCandidate {
	for i := range candidates {
		if candidates[i].ObjectID == id {
			return &candidates[i]
		}
	}
	return nil
}

func playerByNumber(players []PlayerSlot, number int) *PlayerSlot {
	for i := range players {
		if players[i].Number == number {
			return &players[i]
		}
	}
	return nil
}

func writeCoverageU32(buf *bytes.Buffer, value uint32) {
	var raw [4]byte
	binary.LittleEndian.PutUint32(raw[:], value)
	buf.Write(raw[:])
}
