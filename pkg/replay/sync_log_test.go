package replay

import (
	"aoe2kit/pkg/testfixtures"
	"os"
	"testing"
)

func TestSyncLogDesyncFixture(t *testing.T) {
	logPath := testfixtures.Path(t, "save-analysis/diagnostics/desync_500473166_p0-sync.txt")
	replayPath := testfixtures.Path(t, "save-analysis/diagnostics/MP Replay v101.103.48987.0 @2026.08.18 215323 (1).aoe2record")
	if _, err := os.Stat(logPath); err != nil {
		t.Skipf("sync log fixture not present: %v", err)
	}
	if _, err := os.Stat(replayPath); err != nil {
		t.Skipf("replay fixture not present: %v", err)
	}
	report, err := BuildSyncLogReport(logPath, SyncLogOptions{ReplayPath: replayPath, Limit: 10})
	if err != nil {
		t.Fatalf("BuildSyncLogReport failed: %v", err)
	}
	if report.Summary.Turns != 10 {
		t.Fatalf("turns = %d, want 10", report.Summary.Turns)
	}
	if report.Summary.FirstWorldTimeMS != 1126191 || report.Summary.LastWorldTimeMS != 1128115 {
		t.Fatalf("world window = %d..%d, want 1126191..1128115", report.Summary.FirstWorldTimeMS, report.Summary.LastWorldTimeMS)
	}
	if report.Summary.ObjectRows != 6091 || report.Summary.DeclaredObjects != report.Summary.ObjectRows {
		t.Fatalf("object rows/declared = %d/%d, want 6091/6091", report.Summary.ObjectRows, report.Summary.DeclaredObjects)
	}
	if report.Summary.AttributeBlocks != 20 || report.Summary.AttributeValues != 80 {
		t.Fatalf("attribute blocks/values = %d/%d, want 20/80", report.Summary.AttributeBlocks, report.Summary.AttributeValues)
	}
	if got := report.Summary.AttributeIndices; len(got) != 4 || got[0] != 0 || got[1] != 1 || got[2] != 2 || got[3] != 3 {
		t.Fatalf("attribute indices = %v, want [0 1 2 3]", got)
	}
	if report.Summary.RNGEvents != 2617 || report.Summary.RNGSetSeeds != 738 || report.Summary.RNGRands != 1879 {
		t.Fatalf("rng events/setseed/rand = %d/%d/%d, want 2617/738/1879", report.Summary.RNGEvents, report.Summary.RNGSetSeeds, report.Summary.RNGRands)
	}
	if len(report.Turns) == 0 || len(report.Turns[0].Players) < 2 {
		t.Fatalf("expected first turn to have at least two player object snapshots")
	}
	p1 := report.Turns[0].Players[0]
	if p1.PlayerID != 1 || p1.Checksum.ObjectCount != 316 || p1.Checksum.UnitTypeSum != 61316 || p1.Checksum.ObjectIDSum != 1471765 {
		t.Fatalf("first P1 checksum candidates = player %d count %d type %d id %d, want P1 316/61316/1471765",
			p1.PlayerID, p1.Checksum.ObjectCount, p1.Checksum.UnitTypeSum, p1.Checksum.ObjectIDSum)
	}
	if p1.DeclaredAttrs != 601 || len(p1.Attributes) != 4 {
		t.Fatalf("first P1 attributes = declared %d printed %d, want 601/4", p1.DeclaredAttrs, len(p1.Attributes))
	}
	if p1.Checksum.ResourceStockpileSum < 1090.696 || p1.Checksum.ResourceStockpileSum > 1090.698 {
		t.Fatalf("first P1 resource sum = %.3f, want about 1090.697", p1.Checksum.ResourceStockpileSum)
	}
	if len(report.RNGStreams) != 8 {
		t.Fatalf("rng streams = %d, want 8", len(report.RNGStreams))
	}
	if len(report.ObjectCensus) == 0 || report.ObjectCensus[0].Name != "WALL2" || report.ObjectCensus[0].Count != 2890 {
		t.Fatalf("top census = %+v, want WALL2 count 2890", report.ObjectCensus)
	}
	if stateRows(report, 2) != 5672 || stateRows(report, 3) != 318 || stateRows(report, 5) != 70 {
		t.Fatalf("state census = %+v, want state 2/3/5 rows 5672/318/70", report.StateCensus)
	}
	state3 := stateByID(report, 3)
	if state3 == nil || len(state3.Names) != 1 || state3.Names[0] != "FLARE" || state3.HPZeroRows != 318 {
		t.Fatalf("state 3 census = %+v, want FLARE hp-zero rows", state3)
	}
	if len(report.CorpseCarry) == 0 {
		t.Fatalf("expected corpse carry tracks")
	}
	var kipchak *SyncLogCorpseCarry
	for i := range report.CorpseCarry {
		row := &report.CorpseCarry[i]
		if row.PlayerID == 2 && row.ObjectID == 6032 && row.UnitID == 1232 && row.UnitName == "KIPCHAK_D" {
			kipchak = row
			break
		}
	}
	if kipchak == nil {
		t.Fatalf("missing P2 KIPCHAK_D[6032] corpse carry track: %+v", report.CorpseCarry)
	}
	if kipchak.Samples != 10 || kipchak.FirstCarry != 297 || kipchak.LastCarry != 295 || kipchak.CarryDrop != 2 || kipchak.ObservedRatePS < 1.03 || kipchak.ObservedRatePS > 1.05 {
		t.Fatalf("bad KIPCHAK_D carry decay: %+v", kipchak)
	}
	if len(report.CorpseDecay) == 0 {
		t.Fatalf("expected corpse decay baselines")
	}
	kipchakBaseline := corpseDecayByUnit(report, 1232)
	if kipchakBaseline == nil {
		t.Fatalf("missing KIPCHAK_D corpse decay baseline: %+v", report.CorpseDecay)
	}
	if kipchakBaseline.UnitName != "KIPCHAK_D" || kipchakBaseline.MovingTracks == 0 || kipchakBaseline.CarryDropTotal < 2 || kipchakBaseline.ObservedRateMinPS < 0.76 || kipchakBaseline.ObservedRateMaxPS > 1.05 {
		t.Fatalf("bad KIPCHAK_D decay baseline: %+v", kipchakBaseline)
	}
	if kipchakBaseline.EstimatedLifetimeMS < 320000 || kipchakBaseline.EstimatedLifetimeMS > 322000 {
		t.Fatalf("KIPCHAK_D estimated lifetime = %d, want about 5 minutes from observed carry/rate", kipchakBaseline.EstimatedLifetimeMS)
	}
	if report.Comparison == nil {
		t.Fatalf("expected replay comparison")
	}
	if report.Comparison.Overlap || report.Comparison.OverlappingSamples != 0 {
		t.Fatalf("expected no overlap, got overlap=%t samples=%d", report.Comparison.Overlap, report.Comparison.OverlappingSamples)
	}
	if report.Comparison.NearestBeforeLogStartMS != 1117039 {
		t.Fatalf("nearest before log start = %d, want 1117039", report.Comparison.NearestBeforeLogStartMS)
	}
	if report.Comparison.NearestBeforeToFirstLog == nil {
		t.Fatalf("expected rough nearest-before comparison")
	}
	rough := report.Comparison.NearestBeforeToFirstLog
	if rough.GapMS != 9152 {
		t.Fatalf("rough gap = %d, want 9152", rough.GapMS)
	}
	if len(rough.Players) == 0 {
		t.Fatalf("expected rough player comparisons")
	}
	if p1 := rough.Players[0]; p1.PlayerID != 1 || p1.ReplayWord1 != 1538 || p1.LogResourceSum < 1090.696 || p1.LogResourceSum > 1090.698 || p1.ReplayWord6 != 305 || p1.LogObjectCount != 316 || p1.ReplayWord2 != 58803 || p1.LogUnitTypeSum != 61316 || p1.ReplayWord3 != 623 || p1.LogStateSum != 658 || p1.ReplayWord4 != 1935 || syncLogTestWord4Candidate(p1, "retarget_timer_sum") != 303693 || syncLogTestWord4Candidate(p1, "action_state_sum") != 486 || syncLogTestWord4Candidate(p1, "carry_sum") != 1972 || p1.ReplayWord10 != 1405464 || p1.LogObjectIDSum != 1471765 {
		t.Fatalf("rough P1 comparison = %+v, want replay/log word6 305/316 word2 58803/61316 word3 623/658 word4 1935/303693 word10 1405464/1471765", p1)
	}
	if p1 := rough.Players[0]; p1.Word4PerObject < 6.12 || p1.Word4PerObject > 6.13 {
		t.Fatalf("P1 word4 per object = %.4f, want about 6.12", p1.Word4PerObject)
	}
	if len(rough.Players) < 2 {
		t.Fatalf("expected P2 rough comparison")
	}
	if p2 := rough.Players[1]; p2.Word4PerObject < 1.88 || p2.Word4PerObject > 1.89 {
		t.Fatalf("P2 word4 per object = %.4f, want about 1.89", p2.Word4PerObject)
	}
}

func syncLogTestWord4Candidate(player SyncLogReplayPlayerCompare, name string) int64 {
	for _, candidate := range player.Word4Candidates {
		if candidate.Name == name {
			return candidate.LogValue
		}
	}
	return 0
}

func stateByID(report *SyncLogReport, state int) *SyncLogStateCensus {
	for i := range report.StateCensus {
		if report.StateCensus[i].State == state {
			return &report.StateCensus[i]
		}
	}
	return nil
}

func stateRows(report *SyncLogReport, state int) int {
	row := stateByID(report, state)
	if row == nil {
		return 0
	}
	return row.Rows
}

func corpseDecayByUnit(report *SyncLogReport, unitID int) *SyncLogCorpseDecay {
	for i := range report.CorpseDecay {
		if report.CorpseDecay[i].UnitID == unitID {
			return &report.CorpseDecay[i]
		}
	}
	return nil
}

func TestSyncLogCorpseDecayListAggregatesMovingTracks(t *testing.T) {
	tracks := map[string]*syncLogCorpseTrack{
		"a": {
			PlayerID:   1,
			ObjectID:   100,
			UnitID:     28,
			UnitName:   "CHUKN_D",
			Samples:    3,
			FirstMS:    1000,
			LastMS:     3000,
			FirstCarry: 285,
			LastCarry:  283,
		},
		"b": {
			PlayerID:   2,
			ObjectID:   101,
			UnitID:     28,
			UnitName:   "CHUKN_D",
			Samples:    2,
			FirstMS:    2000,
			LastMS:     3000,
			FirstCarry: 284,
			LastCarry:  283,
		},
		"static": {
			PlayerID:   1,
			ObjectID:   102,
			UnitID:     3,
			UnitName:   "ARCHR_D",
			Samples:    1,
			FirstMS:    2000,
			LastMS:     2000,
			FirstCarry: 0,
			LastCarry:  0,
		},
	}
	rows := syncLogCorpseDecayList(tracks, 0)
	if len(rows) != 2 {
		t.Fatalf("decay rows = %+v, want two unit baselines", rows)
	}
	row := rows[0]
	if row.UnitID != 28 || row.UnitName != "CHUKN_D" || row.Tracks != 2 || row.MovingTracks != 2 || row.Samples != 5 {
		t.Fatalf("first decay row = %+v, want CHUKN_D aggregate", row)
	}
	if row.FirstCarryMin != 284 || row.FirstCarryMax != 285 || row.LastCarryMin != 283 || row.LastCarryMax != 283 {
		t.Fatalf("carry ranges = %+v, want 284..285 -> 283..283", row)
	}
	if row.CarryDropTotal != 3 || row.DurationMSTotal != 3000 || row.ObservedRatePS != 1 || row.ObservedRateMinPS != 1 || row.ObservedRateMaxPS != 1 {
		t.Fatalf("rate aggregate = %+v, want 3 carry over 3s", row)
	}
	if row.EstimatedLifetimeMS != 285000 || row.EstimatedLifetime != "00:04:45.000" {
		t.Fatalf("estimated lifetime = %d/%s, want 285000/00:04:45.000", row.EstimatedLifetimeMS, row.EstimatedLifetime)
	}
	static := rows[1]
	if static.UnitID != 3 || static.Confidence != "observed_static_or_single_sample_no_decay_rate" || static.ObservedRatePS != 0 {
		t.Fatalf("static row = %+v, want no decay-rate claim", static)
	}
}
