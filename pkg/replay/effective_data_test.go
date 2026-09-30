package replay

import (
	"aoe2kit/pkg/testfixtures"
	"encoding/json"
	"os"
	"testing"
)

func TestLatestCBAEffectiveDataAvailabilityXrefGolden(t *testing.T) {
	replayPath := testfixtures.Path(t, "save-analysis/latest_cba.aoe2record")
	datPath := testfixtures.Path(t, "reference_aoe2_dump/dat/empires2_x2_p1.dat")
	if _, err := os.Stat(replayPath); err != nil {
		t.Skipf("golden replay not present: %v", err)
	}
	if _, err := os.Stat(datPath); err != nil {
		t.Skipf("golden dat not present: %v", err)
	}
	report, err := BuildEffectiveData(replayPath, EffectiveDataOptions{
		DatPath: datPath,
		Player:  2,
		Limit:   12,
	})
	if err != nil {
		t.Fatal(err)
	}
	if report.Template.SHA256 != "cba7550a169f61ba9bafd751df2fec8a7b02fabef5acaa4245e07f2fed86b9d0" {
		t.Fatalf("template sha = %s", report.Template.SHA256)
	}
	if report.TemplateCheck.Status != "matches_known_vanilla_template" || report.TemplateCheck.ExpectedSHA256 != ObservedVanillaEffectiveTemplateSHA256 {
		t.Fatalf("template check = %+v", report.TemplateCheck)
	}
	if report.AvailabilityArray.AbsStart != 2191677 || report.AvailabilityArray.Count != 864 {
		t.Fatalf("availability array = %+v, want abs 2191677 count 864", report.AvailabilityArray)
	}
	if len(report.Players) != 1 {
		t.Fatalf("players = %d, want 1", len(report.Players))
	}
	player := report.Players[0]
	if player.CivID != 31 || player.CivName != "Vietnamese" {
		t.Fatalf("player civ = %d/%q, want 31/Vietnamese", player.CivID, player.CivName)
	}
	if player.DiffRuns < 1000 || player.IdenticalRatio < 99.0 {
		t.Fatalf("player diff summary = runs %d ratio %.2f", player.DiffRuns, player.IdenticalRatio)
	}
	if player.EntryCount != 864 || player.ChangedEntries < 100 || player.Available == 0 || player.Unavailable == 0 {
		t.Fatalf("player availability summary = entries %d changed %d available %d unavailable %d", player.EntryCount, player.ChangedEntries, player.Available, player.Unavailable)
	}
	if len(player.Entries) == 0 {
		t.Fatalf("expected availability entries")
	}
	first := player.Entries[0]
	if first.Index != 0 || first.Value != 2 || first.ReferenceValue != 1 || !first.Changed || !first.ReplayAvailable || first.UnitID != 0 || first.UnitName != "OLD-ACADEMY" {
		t.Fatalf("first entry = %+v", first)
	}
}

func TestLatestCBADatamodCheckSelfBaselineGolden(t *testing.T) {
	replayPath := testfixtures.Path(t, "save-analysis/latest_cba.aoe2record")
	datPath := testfixtures.Path(t, "reference_aoe2_dump/dat/empires2_x2_p1.dat")
	if _, err := os.Stat(replayPath); err != nil {
		t.Skipf("golden replay not present: %v", err)
	}
	if _, err := os.Stat(datPath); err != nil {
		t.Skipf("golden dat not present: %v", err)
	}
	report, err := BuildEffectiveDataModCheck(replayPath, EffectiveDataModCheckOptions{
		BaselineReplay: replayPath,
		DatPath:        datPath,
		Limit:          5,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := json.Marshal(report); err != nil {
		t.Fatalf("datamod-check report must marshal as JSON: %v", err)
	}
	if report.Verdict != "matches_baseline_replay_effective_data" {
		t.Fatalf("verdict = %q, want matches_baseline_replay_effective_data", report.Verdict)
	}
	if report.TailComparison.ComparedPlayers != 8 || report.TailComparison.DifferentPlayers != 0 || report.TailComparison.ChangedBytes != 0 {
		t.Fatalf("tail comparison = %+v, want 8 compared, 0 different, 0 changed bytes", report.TailComparison)
	}
	if !report.AvailabilityDiff.Compared || report.AvailabilityDiff.ComparedEntries != 864 || report.AvailabilityDiff.ChangedEntries != 0 {
		t.Fatalf("availability diff = %+v, want 864 compared and 0 changed", report.AvailabilityDiff)
	}
	if report.TargetTemplate.SHA256 != ObservedVanillaEffectiveTemplateSHA256 {
		t.Fatalf("target template sha = %s", report.TargetTemplate.SHA256)
	}
}

func TestCBAEffectiveUnitHitPointDecodeGolden(t *testing.T) {
	replayPath := testfixtures.Path(t, "save-analysis/archive/078__MP Replay v101.103.48987.0 @2026.07.19 224323 (5).aoe2record")
	datPath := testfixtures.Path(t, "reference_aoe2_dump/dat/empires2_x2_p1.dat")
	if _, err := os.Stat(replayPath); err != nil {
		t.Skipf("golden replay not present: %v", err)
	}
	if _, err := os.Stat(datPath); err != nil {
		t.Skipf("golden dat not present: %v", err)
	}
	report, err := BuildEffectiveUnitStats(replayPath, EffectiveUnitStatsOptions{
		DatPath: datPath,
		Limit:   12,
	})
	if err != nil {
		t.Fatal(err)
	}
	if report.TemplateCheck.Status != "differs_from_known_vanilla_template" {
		t.Fatalf("template check = %s, want differs_from_known_vanilla_template", report.TemplateCheck.Status)
	}
	if report.Summary.Players != 8 || report.Summary.ChangedHitPoints < 100 || report.Summary.Rows < 40 {
		t.Fatalf("summary = %+v, want 8 players and decoded changed HP rows", report.Summary)
	}
	if report.Summary.ModConfirmedMultiCiv == 0 || report.Summary.UnresolvedSingleCiv == 0 {
		t.Fatalf("classification summary = %+v, want both multi-civ confirmed and unresolved single-civ rows", report.Summary)
	}
	row := effectiveUnitRow(report.Rows, 2, 21)
	if row == nil {
		t.Fatalf("missing P2 unit 21 GALLY row")
	}
	if row.CivName != "Portuguese" || row.UnitName != "GALLY" || row.VanillaHitPoints != 125 || row.EffectiveHitPoints != 138 {
		t.Fatalf("P2 GALLY row = %+v, want Portuguese GALLY 125->138", *row)
	}
	if row.Classification != "unresolved_single_civ" || row.CivSpan != 1 {
		t.Fatalf("P2 GALLY classification = %q span %d, want unresolved_single_civ/1", row.Classification, row.CivSpan)
	}
	row = effectiveUnitRow(report.Rows, 2, 156)
	if row == nil {
		t.Fatalf("missing P2 unit 156 VMREP row")
	}
	if row.Classification != "mod_confirmed_multi_civ" || row.CivSpan < 2 {
		t.Fatalf("P2 VMREP classification = %q span %d, want mod_confirmed_multi_civ and span >=2", row.Classification, row.CivSpan)
	}
	row = effectiveUnitRow(report.Rows, 3, 160)
	if row == nil {
		t.Fatalf("missing P3 unit 160 HRLION row")
	}
	if row.CivName != "French" || row.UnitName != "HRLION" || row.VanillaHitPoints != 220 || row.EffectiveHitPoints != 264 {
		t.Fatalf("P3 HRLION row = %+v, want French HRLION 220->264", *row)
	}
}

func effectiveUnitRow(rows []EffectiveUnitStatDiffRow, playerIndex int, unitID int16) *EffectiveUnitStatDiffRow {
	for i := range rows {
		if rows[i].PlayerIndex == playerIndex && rows[i].UnitID == unitID {
			return &rows[i]
		}
	}
	return nil
}
