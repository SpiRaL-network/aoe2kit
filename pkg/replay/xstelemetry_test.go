package replay

import (
	"aoe2kit/pkg/testfixtures"
	"os"
	"testing"
)

func TestDecodeXSTelemetryWord1FoodCarrier(t *testing.T) {
	sample, ok := decodeXSTelemetryWord1(910203)
	if !ok {
		t.Fatal("food carrier did not decode")
	}
	if sample.Carrier != "food_attr_0" || sample.Attr20Value != 1 || sample.Attr154Value != 2 || sample.Attr43Value != 3 || sample.KillEventsFromAttr154 != 2 {
		t.Fatalf("sample = %+v, want food attr20=1 attr154=2 attr43=3", sample)
	}
}

func TestDecodeXSTelemetryWord1Unused220Carrier(t *testing.T) {
	sample, ok := decodeXSTelemetryWord1(720304)
	if !ok {
		t.Fatal("unused 220 carrier did not decode")
	}
	if sample.Carrier != "unused_attr_220_candidate" || sample.Attr20Value != 2 || sample.Attr154Value != 3 || sample.Attr43Value != 4 || sample.KillEventsFromAttr154 != 3 {
		t.Fatalf("sample = %+v, want unused 220 attr20=2 attr154=3 attr43=4", sample)
	}
}

func TestDecodeXSTelemetryWord1RejectsNonProbeValue(t *testing.T) {
	if sample, ok := decodeXSTelemetryWord1(610000); ok {
		t.Fatalf("decoded non-probe value as %+v", sample)
	}
}

func TestDecodeXSTelemetryWord1ConfiguredHarnessBase(t *testing.T) {
	sample, ok := decodeXSTelemetryWord1WithBases(100003, 100000, 700000)
	if !ok || sample.Carrier != "food_attr_0" || sample.Word1 != 100003 {
		t.Fatalf("sample = %+v, ok=%v", sample, ok)
	}
}

func TestXSTelemetryV3InstrumentedGolden(t *testing.T) {
	path := testfixtures.Path(t, "save-analysis/diagnostics/diag_v3_run.aoe2record")
	if _, err := os.Stat(path); err != nil {
		t.Skipf("v3 diagnostic replay unavailable: %v", err)
	}
	report, err := BuildXSTelemetryProbe(path)
	if err != nil {
		t.Fatalf("BuildXSTelemetryProbe: %v", err)
	}
	if report.Summary.ChecksumSamples != 14 {
		t.Fatalf("checksum samples = %d, want 14", report.Summary.ChecksumSamples)
	}
	if report.Summary.DecodedSamples != 3 || !report.Summary.FoodCarrierMoved || report.Summary.Unused220CarrierMoved {
		t.Fatalf("summary = %+v, want 3 food-carrier samples only", report.Summary)
	}
	if report.Summary.ChatMarkers != 0 {
		t.Fatalf("chat markers = %d, want 0", report.Summary.ChatMarkers)
	}
	wantWords := []uint32{900000, 900100, 900200}
	wantAttr154 := []int{0, 1, 2}
	if len(report.Samples) != len(wantWords) {
		t.Fatalf("samples = %d, want %d: %+v", len(report.Samples), len(wantWords), report.Samples)
	}
	for i, sample := range report.Samples {
		if sample.Word1 != wantWords[i] {
			t.Fatalf("sample %d word1 = %d, want %d", i, sample.Word1, wantWords[i])
		}
		if sample.Attr20Value != 0 || sample.KillEventsFromAttr154 != wantAttr154[i] || sample.Attr43Value != 0 {
			t.Fatalf("sample %d = %+v, want attr20=0 attr154_kill_events=%d attr43=0", i, sample, wantAttr154[i])
		}
	}
}

func TestXSTelemetryV3XSlessControlGolden(t *testing.T) {
	path := testfixtures.Path(t, "save-analysis/diagnostics/diag_v3_xsless_control.aoe2record")
	if _, err := os.Stat(path); err != nil {
		t.Skipf("v3 XS-less control replay unavailable: %v", err)
	}
	report, err := BuildXSTelemetryProbe(path)
	if err != nil {
		t.Fatalf("BuildXSTelemetryProbe: %v", err)
	}
	if report.Summary.DecodedSamples != 0 || report.Summary.FoodCarrierMoved || report.Summary.Unused220CarrierMoved || report.Summary.ChatMarkers != 0 {
		t.Fatalf("summary = %+v, want no telemetry", report.Summary)
	}
}

func TestV3TriggerVictoryPostgameMetadataOnlyGolden(t *testing.T) {
	path := testfixtures.Path(t, "save-analysis/diagnostics/diag_v3_run.aoe2record")
	if _, err := os.Stat(path); err != nil {
		t.Skipf("v3 diagnostic replay unavailable: %v", err)
	}
	report, err := BuildPostgame(path)
	if err != nil {
		t.Fatalf("BuildPostgame: %v", err)
	}
	if !report.Summary.Op6Seen || report.Summary.Op6TailBytes != 62 || report.Summary.Op6Blocks != 2 {
		t.Fatalf("summary = %+v, want op6 62-byte 2-block tail", report.Summary)
	}
	if !report.Summary.HasLeaderboardBlock || !report.Summary.HasWorldTimeBlock || report.Summary.HasPlayerKills {
		t.Fatalf("summary = %+v, want metadata-only postgame", report.Summary)
	}
	if report.DE == nil || report.DE.WorldTimeMS != 120032 {
		t.Fatalf("de postgame = %+v, want world time 120032", report.DE)
	}
}

func TestRankedRMPostgameMetadataOnlyGolden(t *testing.T) {
	path := testfixtures.Path(t, "save-analysis/insights_rm_team_493687875.zip")
	if _, err := os.Stat(path); err != nil {
		t.Skipf("ranked RM replay unavailable: %v", err)
	}
	report, err := BuildPostgame(path)
	if err != nil {
		t.Fatalf("BuildPostgame: %v", err)
	}
	if report.Verification != "structure_verified_scenario_and_ranked_rm_postgame_stats_not_serialized" {
		t.Fatalf("verification = %q", report.Verification)
	}
	if !report.Summary.Op6Seen || report.Summary.Op6TailBytes != 252 || report.Summary.Op6Blocks != 2 {
		t.Fatalf("summary = %+v, want op6 252-byte 2-block tail", report.Summary)
	}
	if !report.Summary.HasLeaderboardBlock || !report.Summary.HasWorldTimeBlock || report.Summary.HasPlayerKills {
		t.Fatalf("summary = %+v, want metadata-only ranked RM postgame", report.Summary)
	}
	if report.DE == nil || report.DE.WorldTimeMS != 2620264 || len(report.DE.Leaderboards) != 2 {
		t.Fatalf("de postgame = %+v, want world time 2620264 and two leaderboards", report.DE)
	}
	foundRankOne := false
	for _, board := range report.DE.Leaderboards {
		for _, player := range board.Players {
			if player.Rank == 1 && player.Rating == 2926 {
				foundRankOne = true
			}
		}
	}
	if !foundRankOne {
		t.Fatalf("leaderboards = %+v, want rank 1 rating 2926 row", report.DE.Leaderboards)
	}
}
