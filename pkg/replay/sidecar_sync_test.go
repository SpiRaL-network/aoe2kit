package replay

import (
	"testing"

	xsauthor "aoe2kit/pkg/xs"
)

func TestParseAndCorrelateStateVectorSidecar(t *testing.T) {
	values := []xsauthor.DataValue{
		xsString("A2K_RTV10_STATE_VECTOR"),
		xsInt(10),
		xsString("sync_matrix_sidecar_calibration_v10"),
	}
	values = append(values, xsPhaseValues(0, 5, 881000, 0, 0, 0, 1, 0, 0, 0)...)
	values = append(values, xsPhaseValues(10, 27, 881010, 0, 0, 0, 1, 1, 0, 0)...)
	values = append(values, xsPhaseValues(40, 93, 881040, 0, 101, 0, 1, 1, 1, 1)...)
	values = append(values, xsString("end"), xsInt(3), xsString("append_probe"), xsInt(810010))

	sidecar, warnings := parseStateVectorSidecar(values)
	if len(warnings) != 0 {
		t.Fatalf("warnings = %#v", warnings)
	}
	if sidecar.Magic != "A2K_RTV10_STATE_VECTOR" || sidecar.Version != 10 || sidecar.RowsWritten != 3 || sidecar.AppendProbe != 810010 {
		t.Fatalf("sidecar header = %+v", sidecar)
	}
	if len(sidecar.Phases) != 3 {
		t.Fatalf("phases = %d, want 3", len(sidecar.Phases))
	}
	if got := sidecar.Phases[1].Expected.UnitTypeSum; got != 16 {
		t.Fatalf("phase 10 unit type sum = %d, want 16", got)
	}
	if got := sidecar.Phases[1].Expected.ObjectCount; got != 2 {
		t.Fatalf("phase 10 object count = %d, want 2", got)
	}

	events := []SyncEvent{
		checksumEvent(5200, []uint32{0, 881000, 12, 2, 0, 0, 1, 100, 1, 0, 9000}),
		checksumEvent(27200, []uint32{0, 881010, 16, 4, 0, 0, 2, 110, 1, 10, 9004}),
		checksumEvent(93600, []uint32{0, 881040, 223, 8, 101, 0, 4, 120, 1, 20, 9010}),
	}
	rows := correlateSidecarPhases(sidecar.Phases, events, 1, 30000)
	if len(rows) != 3 {
		t.Fatalf("correlations = %d, want 3", len(rows))
	}
	for i, row := range rows {
		if row.Sample == nil {
			t.Fatalf("row %d has no sample", i)
		}
		if row.Verdict != "matched" {
			t.Fatalf("row %d verdict = %q, want matched; notes=%v", i, row.Verdict, row.Notes)
		}
		if len(row.KnownWordChecks) != 3 {
			t.Fatalf("row %d known word checks = %d, want 3", i, len(row.KnownWordChecks))
		}
		if row.CarrierMatch == nil || !*row.CarrierMatch {
			t.Fatalf("row %d carrier match = %v", i, row.CarrierMatch)
		}
	}
	if rows[1].UnitTypeMatch == nil || !*rows[1].UnitTypeMatch {
		t.Fatalf("phase 10 unit type match = %v", rows[1].UnitTypeMatch)
	}
	if rows[1].ObjectCountMatch == nil || !*rows[1].ObjectCountMatch {
		t.Fatalf("phase 10 object count match = %v", rows[1].ObjectCountMatch)
	}
	if len(rows[2].AttributeMirrors) != 1 || rows[2].AttributeMirrors[0].Attribute != "attr33" || rows[2].AttributeMirrors[0].WordIndex != 4 {
		t.Fatalf("phase 40 attribute mirrors = %#v, want attr33 at word4", rows[2].AttributeMirrors)
	}
}

func TestCorrelateStateVectorSidecarKnownWordMismatch(t *testing.T) {
	values := []xsauthor.DataValue{
		xsString("A2K_RTV10_STATE_VECTOR"),
		xsInt(10),
		xsString("sync_matrix_sidecar_calibration_v10"),
	}
	values = append(values, xsPhaseValues(10, 27, 881010, 0, 0, 0, 1, 1, 0, 0)...)
	sidecar, warnings := parseStateVectorSidecar(values)
	if len(warnings) != 0 {
		t.Fatalf("warnings = %#v", warnings)
	}

	events := []SyncEvent{
		checksumEvent(27200, []uint32{0, 881010, 999, 4, 0, 0, 2, 110, 1, 10, 9004}),
	}
	rows := correlateSidecarPhases(sidecar.Phases, events, 1, 30000)
	if len(rows) != 1 {
		t.Fatalf("correlations = %d, want 1", len(rows))
	}
	if rows[0].Verdict != "known_word_mismatch" {
		t.Fatalf("verdict = %q, want known_word_mismatch", rows[0].Verdict)
	}
	if rows[0].UnitTypeMatch == nil || *rows[0].UnitTypeMatch {
		t.Fatalf("unit type match = %v, want false", rows[0].UnitTypeMatch)
	}
	if len(rows[0].Notes) == 0 {
		t.Fatal("expected mismatch note")
	}
}

func TestBuildSidecarSyncReportSummarizesSidecarDecodeFailure(t *testing.T) {
	report := &SidecarSyncReport{}
	decode := xsauthor.DataDecodeReport{}
	decode.OK = false
	decode.Summary.Total = 4
	decode.Summary.Passed = 3
	decode.Summary.Failed = 1
	applySidecarDecodeSummary(report, decode)
	if report.Summary.SidecarDecodeOK {
		t.Fatal("sidecar decode ok = true, want false")
	}
	if report.Summary.LedgerAssertions != 4 || report.Summary.LedgerPassed != 3 || report.Summary.LedgerFailed != 1 {
		t.Fatalf("ledger summary = %+v", report.Summary)
	}
}

func TestParseSchemaSidecarBuildsGenericExpectation(t *testing.T) {
	report := xsauthor.DataSchemaDecodeReport{
		OK:     true,
		Schema: "rtv13-directed-attribution",
		Header: map[string]any{
			"magic":   "A2K_RTV13_DIRECTED_ATTRIBUTION",
			"version": 13,
			"fixture": "directed_kill_raze_matrix_v13",
		},
		Footer: map[string]any{
			"rows_written": 1,
			"append_probe": 810013,
		},
		Rows: []xsauthor.DataSchemaRow{
			{
				Index:   0,
				PhaseID: 101,
				TimeS:   205,
				Fields: map[string]any{
					"case_player":                        4,
					"case_kind":                          1,
					"p1_resource_sum_carrier_food_attr0": float32(884101),
					"p1_wood_case_player_attr1":          float32(4),
					"p1_gold_case_kind_attr3":            float32(1),
					"p1_stone_row_index_attr2":           float32(7),
					"p1_kills_attr20":                    float32(3),
					"p1_militia_74_count":                1,
					"p1_barracks_12_count":               1,
					"p4_militia_74_count":                0,
					"p4_barracks_12_count":               1,
				},
			},
		},
	}
	sidecar, warnings := parseSchemaSidecar(report, 1)
	if len(warnings) != 0 {
		t.Fatalf("warnings = %#v", warnings)
	}
	if sidecar.Magic != "A2K_RTV13_DIRECTED_ATTRIBUTION" || sidecar.Version != 13 || sidecar.RowsWritten != 1 || sidecar.AppendProbe != 810013 {
		t.Fatalf("sidecar header = %+v", sidecar)
	}
	if len(sidecar.Phases) != 1 {
		t.Fatalf("phases = %d, want 1", len(sidecar.Phases))
	}
	phase := sidecar.Phases[0]
	if phase.Label != "p4_unit_kill" {
		t.Fatalf("label = %q, want p4_unit_kill", phase.Label)
	}
	if phase.Expected.ResourceStockpile != 884113 {
		t.Fatalf("resource stockpile = %d, want 884113", phase.Expected.ResourceStockpile)
	}
	if phase.Expected.UnitTypeSum != 86 {
		t.Fatalf("unit type sum = %d, want 86", phase.Expected.UnitTypeSum)
	}
	if phase.Expected.ObjectCount != 2 {
		t.Fatalf("object count = %d, want 2", phase.Expected.ObjectCount)
	}
	if phase.Attr20 != 3 {
		t.Fatalf("attr20 = %v, want 3", phase.Attr20)
	}
}

func TestParseA2KSEM2SchemaSidecarBuildsResourceExpectation(t *testing.T) {
	report := xsauthor.DataSchemaDecodeReport{
		OK:     true,
		Schema: "a2ksem2-dat-command-semantics",
		Header: map[string]any{
			"magic":   "A2K_DAT_COMMAND_SEMANTICS",
			"version": 2,
			"fixture": "run1_resource_object_count_contract",
		},
		Footer: map[string]any{"rows_written": 1},
		Rows: []xsauthor.DataSchemaRow{
			{
				Index:   0,
				PhaseID: 25,
				TimeS:   25,
				Fields: map[string]any{
					"p1_food_attr0":            float32(1111),
					"p1_wood_attr1":            float32(22),
					"p1_stone_attr2":           float32(333),
					"p1_gold_attr3":            float32(222),
					"p1_militia_74_count":      1,
					"p1_man_at_arms_75_count":  0,
					"p1_villager_83_count":     1,
					"p1_scout_448_count":       2,
					"p1_town_center_109_count": 1,
					"p1_barracks_12_count":     1,
					"p1_stable_101_count":      1,
					"p1_research_count_attr21": float32(2),
					"p1_population_attr11":     float32(4),
				},
			},
		},
	}
	sidecar, warnings := parseSchemaSidecar(report, 1)
	if len(warnings) != 0 {
		t.Fatalf("warnings = %#v", warnings)
	}
	if len(sidecar.Phases) != 1 {
		t.Fatalf("phases = %d, want 1", len(sidecar.Phases))
	}
	phase := sidecar.Phases[0]
	if phase.Expected.ResourceStockpile != 1688 {
		t.Fatalf("resource stockpile = %d, want 1688", phase.Expected.ResourceStockpile)
	}
	if phase.Expected.UnitTypeSum != 1275 {
		t.Fatalf("unit type sum = %d, want 1275", phase.Expected.UnitTypeSum)
	}
	if phase.Expected.ObjectCount != 7 {
		t.Fatalf("object count = %d, want 7", phase.Expected.ObjectCount)
	}
	if !phase.Expected.ResourceStockpileKnown {
		t.Fatal("resource stockpile known = false, want true")
	}
	if phase.Expected.UnitTypeSumKnown || phase.Expected.ObjectCountKnown {
		t.Fatalf("a2ksem2 generic unit/object checksum expectations should be disabled: %+v", phase.Expected)
	}
}

func xsPhaseValues(phaseID, xsTime int, food, attr20, attr33, attr220 float32, barracks, archer, monk, castle int) []xsauthor.DataValue {
	return []xsauthor.DataValue{
		xsString("phase"),
		xsInt(phaseID),
		xsInt(xsTime),
		xsFloat(food),
		xsFloat(0),
		xsFloat(0),
		xsFloat(0),
		xsFloat(attr20),
		xsFloat(attr33),
		xsFloat(attr220),
		xsInt(barracks),
		xsInt(archer),
		xsInt(monk),
		xsInt(castle),
	}
}

func checksumEvent(timeMS int, words []uint32) SyncEvent {
	matrix := make([][]uint32, 8)
	matrix[0] = words
	for i := 1; i < 8; i++ {
		matrix[i] = make([]uint32, 11)
	}
	return SyncEvent{TimeMS: timeMS, Time: FormatTime(timeMS), Form: "checksum_de", Matrix: matrix}
}

func xsString(value string) xsauthor.DataValue {
	return xsauthor.DataValue{Type: "string", String: value}
}

func xsInt(value int) xsauthor.DataValue {
	v := int32(value)
	return xsauthor.DataValue{Type: "int", Int: &v}
}

func xsFloat(value float32) xsauthor.DataValue {
	return xsauthor.DataValue{Type: "float", Float: &value}
}
