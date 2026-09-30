package replay

import (
	"aoe2kit/pkg/testfixtures"
	"os"
	"testing"
)

func TestSyncStreamGoldenLatestCBA(t *testing.T) {
	path := testfixtures.Path(t, "save-analysis/latest_cba.aoe2record")
	if _, err := os.Stat(path); err != nil {
		t.Skipf("golden replay not present: %v", err)
	}
	report, err := BuildSyncStream(path, SyncOptions{Limit: 3, ChecksumsOnly: true})
	if err != nil {
		t.Fatalf("BuildSyncStream failed: %v", err)
	}
	if report.Summary.SyncCount != 26878 {
		t.Fatalf("expected 26878 syncs, got %d", report.Summary.SyncCount)
	}
	if report.Summary.ChecksumDE != 53 {
		t.Fatalf("expected 53 DE checksum syncs, got %d", report.Summary.ChecksumDE)
	}
	if report.Summary.Duration != "00:42:06.162" {
		t.Fatalf("unexpected duration %s", report.Summary.Duration)
	}
	if len(report.Events) != 3 {
		t.Fatalf("expected 3 emitted checksum events, got %d", len(report.Events))
	}
	first := report.Events[0]
	if first.Form != "checksum_de" || len(first.Matrix) != 8 || len(first.Matrix[0]) != 11 {
		t.Fatalf("expected checksum_de with 8x11 matrix, got form=%s matrix=%dx", first.Form, len(first.Matrix))
	}
	// Decoded semantics: the DE checksum trailer equals current world time in ms.
	if first.TrailerU32 == nil || int(*first.TrailerU32) != first.TimeMS {
		t.Fatalf("expected trailer to equal time_ms (world-time), got trailer=%v time=%d", first.TrailerU32, first.TimeMS)
	}
	// Decoded semantics: word 8 of each row equals the row's player number.
	for p := 0; p < 8; p++ {
		if int(first.Matrix[p][8]) != p+1 {
			t.Fatalf("expected matrix row %d word 8 to equal player number %d, got %d", p, p+1, first.Matrix[p][8])
		}
	}
	// Strong hypothesis anchor: CBA uniform starting stockpile in word 1.
	if first.Matrix[0][1] != 610000 {
		t.Fatalf("expected P1 word 1 starting stockpile 610000, got %d", first.Matrix[0][1])
	}
}

func TestSyncRawWordsLatestCBA(t *testing.T) {
	path := testfixtures.Path(t, "save-analysis/latest_cba.aoe2record")
	if _, err := os.Stat(path); err != nil {
		t.Skipf("golden replay not present: %v", err)
	}
	report, err := BuildSyncStream(path, SyncOptions{ChecksumsOnly: true, RawWords: true})
	if err != nil {
		t.Fatalf("BuildSyncStream failed: %v", err)
	}
	if len(report.RawWords) != report.Summary.ChecksumDE*8 {
		t.Fatalf("raw rows = %d, want checksum samples %d * 8", len(report.RawWords), report.Summary.ChecksumDE)
	}
	first := report.RawWords[0]
	if first.SampleIndex != 1 || first.PlayerID != 1 || len(first.Words) != 11 {
		t.Fatalf("first raw row = sample %d player %d words %d, want 1/P1/11", first.SampleIndex, first.PlayerID, len(first.Words))
	}
	if first.Words[1] != 610000 || first.Words[8] != 1 {
		t.Fatalf("first raw row word1/word8 = %d/%d, want 610000/1", first.Words[1], first.Words[8])
	}
}

func TestChecksumPhaseAllHasTopLevelSummary(t *testing.T) {
	path := testfixtures.Path(t, "save-analysis/latest_cba.aoe2record")
	if _, err := os.Stat(path); err != nil {
		t.Skipf("golden replay not present: %v", err)
	}
	report, err := BuildChecksumPhaseAll(path, ChecksumPhaseOptions{})
	if err != nil {
		t.Fatalf("BuildChecksumPhaseAll failed: %v", err)
	}
	if report.Summary.Words != 11 {
		t.Fatalf("words = %d, want 11", report.Summary.Words)
	}
	if report.Summary.ChecksumSamples != 53 || report.Summary.Players != 8 {
		t.Fatalf("summary = %+v, want 53 samples and 8 players", report.Summary)
	}
}

func TestSyncStateDeltasUseSignedChecksumWordDelta(t *testing.T) {
	prev := SyncEvent{
		TimeMS: 1000,
		Time:   "00:00:01.000",
		Matrix: emptyChecksumMatrix(),
	}
	curr := SyncEvent{
		TimeMS: 2000,
		Time:   "00:00:02.000",
		Matrix: emptyChecksumMatrix(),
	}
	prev.Matrix[0][8] = 1
	curr.Matrix[0][8] = 1
	prev.Matrix[0][9] = 4294967000
	curr.Matrix[0][9] = 120

	deltas := syncStateDeltas(prev, curr)
	if len(deltas) != 1 {
		t.Fatalf("deltas = %d, want 1: %+v", len(deltas), deltas)
	}
	if deltas[0].ScoreDelta != 416 {
		t.Fatalf("score delta = %d, want signed wrap delta 416", deltas[0].ScoreDelta)
	}

	prev.Matrix[0][9] = 120
	curr.Matrix[0][9] = 4294967000
	deltas = syncStateDeltas(prev, curr)
	if len(deltas) != 1 {
		t.Fatalf("reverse deltas = %d, want 1: %+v", len(deltas), deltas)
	}
	if deltas[0].ScoreDelta != -416 {
		t.Fatalf("reverse score delta = %d, want signed wrap delta -416", deltas[0].ScoreDelta)
	}
}

func emptyChecksumMatrix() [][]uint32 {
	matrix := make([][]uint32, 8)
	for i := range matrix {
		matrix[i] = make([]uint32, 11)
	}
	return matrix
}

func TestSyncMatrixIdenticalAcrossMultiplayerPOVs(t *testing.T) {
	leftPath := testfixtures.Path(t, "save-analysis/insights_rm_pov5725572.zip")
	rightPath := testfixtures.Path(t, "save-analysis/insights_rm_team_493687875.zip")
	left, err := BuildSyncStream(leftPath, SyncOptions{ChecksumsOnly: true, RawWords: true})
	if err != nil {
		t.Fatalf("left BuildSyncStream failed: %v", err)
	}
	right, err := BuildSyncStream(rightPath, SyncOptions{ChecksumsOnly: true, RawWords: true})
	if err != nil {
		t.Fatalf("right BuildSyncStream failed: %v", err)
	}
	if left.Summary.ChecksumDE != right.Summary.ChecksumDE || left.Summary.DurationMS != right.Summary.DurationMS {
		t.Fatalf("summary mismatch: left checksums/duration=%d/%d right=%d/%d",
			left.Summary.ChecksumDE, left.Summary.DurationMS, right.Summary.ChecksumDE, right.Summary.DurationMS)
	}
	if len(left.RawWords) != len(right.RawWords) {
		t.Fatalf("raw row count mismatch: left=%d right=%d", len(left.RawWords), len(right.RawWords))
	}
	for i := range left.RawWords {
		l := left.RawWords[i]
		r := right.RawWords[i]
		if l.SampleIndex != r.SampleIndex || l.TimeMS != r.TimeMS || l.PlayerID != r.PlayerID {
			t.Fatalf("raw row %d metadata mismatch: left=%+v right=%+v", i, l, r)
		}
		if len(l.Words) != len(r.Words) {
			t.Fatalf("raw row %d word count mismatch: left=%d right=%d", i, len(l.Words), len(r.Words))
		}
		for word := range l.Words {
			if l.Words[word] != r.Words[word] {
				t.Fatalf("raw row %d sample=%d player=P%d word=%d mismatch: left=%d right=%d",
					i, l.SampleIndex, l.PlayerID, word, l.Words[word], r.Words[word])
			}
		}
	}
}

func TestSyncMatrixCollapseOnWipeLatestCBA(t *testing.T) {
	path := testfixtures.Path(t, "save-analysis/latest_cba.aoe2record")
	if _, err := os.Stat(path); err != nil {
		t.Skipf("golden replay not present: %v", err)
	}
	report, err := BuildSyncStream(path, SyncOptions{ChecksumsOnly: true})
	if err != nil {
		t.Fatalf("BuildSyncStream failed: %v", err)
	}
	if len(report.Events) == 0 {
		t.Fatalf("expected checksum events")
	}
	last := report.Events[len(report.Events)-1]
	// P2 was vote-kicked (35:14) and P6 wiped (~33:24): their word-3 metric collapses.
	// P5 resigned with units standing: metric persists. Ground truth from the replay's
	// chat/resign record; this pins the wipe-vs-resign distinction.
	if last.Matrix[1][3] >= 100 {
		t.Fatalf("expected kicked P2 word 3 to collapse below 100, got %d", last.Matrix[1][3])
	}
	if last.Matrix[5][3] >= 100 {
		t.Fatalf("expected wiped P6 word 3 to collapse below 100, got %d", last.Matrix[5][3])
	}
	if last.Matrix[4][3] < 100 {
		t.Fatalf("expected resigned P5 word 3 to persist above 100, got %d", last.Matrix[4][3])
	}
}

func TestSyncMatrixDiagnosticV2ObjectSumDeltas(t *testing.T) {
	path := testfixtures.Path(t, "save-analysis/diagnostics/diag_v2_run_20260721.aoe2record")
	if _, err := os.Stat(path); err != nil {
		t.Skipf("golden replay not present: %v", err)
	}
	report, err := BuildSyncStream(path, SyncOptions{ChecksumsOnly: true})
	if err != nil {
		t.Fatalf("BuildSyncStream failed: %v", err)
	}
	var additions []SyncStateDelta
	for _, delta := range report.StateDeltas {
		if delta.PlayerID == 1 && delta.Kind == "object_added" {
			additions = append(additions, delta)
		}
	}
	if len(additions) != 6 {
		t.Fatalf("object additions = %d, want 6: %+v", len(additions), additions)
	}
	wantUnits := []int{83, 448, 83, 83, 448, 83}
	wantObjects := []int64{4, 5, 6, 7, 8, 9}
	for i, delta := range additions {
		if delta.ObjectCountDelta != 1 {
			t.Fatalf("addition %d object count delta = %d, want 1", i, delta.ObjectCountDelta)
		}
		if delta.UnitID != wantUnits[i] {
			t.Fatalf("addition %d unit = %d, want %d: %+v", i, delta.UnitID, wantUnits[i], delta)
		}
		if delta.ObjectID != wantObjects[i] {
			t.Fatalf("addition %d object id = %d, want %d: %+v", i, delta.ObjectID, wantObjects[i], delta)
		}
	}
	// Word 7 is movement-sensitive: after object creation stops, autoscout movement
	// continues changing the position-sum while count/type/id sums stay fixed.
	movementOnly := false
	for _, delta := range report.StateDeltas {
		if delta.PlayerID == 1 && delta.Kind == "position_sum_changed" && delta.ObjectCountDelta == 0 && delta.UnitTypeSumDelta == 0 && delta.ObjectIDSumDelta == 0 {
			movementOnly = true
			break
		}
	}
	if !movementOnly {
		t.Fatalf("expected at least one movement-only position-sum delta")
	}
}

func TestKillFactoryFogMarkersAreClassified(t *testing.T) {
	path := testfixtures.Path(t, "save-analysis/diagnostics/SP Replay v101.103.48987.0 @2026.08.18 035259.aoe2record")
	report, err := BuildSyncStream(path, SyncOptions{ChecksumsOnly: true})
	if err != nil {
		t.Fatalf("BuildSyncStream failed: %v", err)
	}
	var p1FogRemoved, p2FogRemoved bool
	for _, delta := range report.StateDeltas {
		if delta.Kind != "fog_reveal_markers_removed" {
			continue
		}
		if delta.UnitID != 112 || delta.UnitName != "Fog Reveal Marker" || delta.ObjectCountDelta != -3 || delta.UnitTypeSumDelta != -336 {
			t.Fatalf("bad fog marker delta: %+v", delta)
		}
		if delta.PlayerID == 1 {
			p1FogRemoved = true
		}
		if delta.PlayerID == 2 {
			p2FogRemoved = true
		}
	}
	if !p1FogRemoved || !p2FogRemoved {
		t.Fatalf("expected fog marker removals for P1 and P2, got p1=%t p2=%t deltas=%+v", p1FogRemoved, p2FogRemoved, report.StateDeltas)
	}
}

func TestKillFactoryCorpseAndFlareAreClassified(t *testing.T) {
	path := testfixtures.Path(t, "save-analysis/diagnostics/SP Replay v101.103.48987.0 @2026.08.18 132058.aoe2record")
	report, err := BuildSyncStream(path, SyncOptions{ChecksumsOnly: true})
	if err != nil {
		t.Fatalf("BuildSyncStream failed: %v", err)
	}
	if len(report.Events) == 0 || len(report.Events[0].Matrix) < 4 {
		t.Fatalf("missing first checksum event: %+v", report.Events)
	}
	initial := report.Events[0].Matrix
	if initial[0][7] != 1200 || initial[1][7] != 12000 || initial[2][7] != 12300 || initial[3][7] != 12600 {
		t.Fatalf("bad initial word_7 position sums: P1=%d P2=%d P3=%d P4=%d", initial[0][7], initial[1][7], initial[2][7], initial[3][7])
	}
	var replacement, carryDecay, flareAdded bool
	var spawnPosition, flarePosition bool
	for _, delta := range report.StateDeltas {
		switch delta.Kind {
		case "object_added":
			if delta.PlayerID == 2 && delta.UnitID == 73 && delta.PositionSumDelta == 1700 {
				spawnPosition = true
			}
		case "live_unit_replaced_by_corpse":
			if delta.PlayerID == 2 && delta.UnitTypeSumDelta == -45 && delta.UnitID == 73 {
				replacement = true
			}
		case "carry_sum_decreased":
			if delta.PlayerID == 2 && delta.Word4Delta < 0 {
				carryDecay = true
			}
		case "player_flare_ping_added":
			if delta.UnitID == 274 && delta.UnitName == "Flare2" {
				flareAdded = true
			}
			if delta.UnitID == 274 && delta.PositionSumDelta == 1647 {
				flarePosition = true
			}
		}
	}
	if !replacement || !carryDecay || !flareAdded || !spawnPosition || !flarePosition {
		t.Fatalf("expected replacement=%t carryDecay=%t flareAdded=%t spawnPosition=%t flarePosition=%t in deltas=%+v", replacement, carryDecay, flareAdded, spawnPosition, flarePosition, report.StateDeltas)
	}
}

func TestKillFactoryDeathReportNamesKnownCorpseReplacement(t *testing.T) {
	path := testfixtures.Path(t, "save-analysis/diagnostics/SP Replay v101.103.48987.0 @2026.08.18 132058.aoe2record")
	report, err := BuildDeathReport(path, DeathReportOptions{Limit: 0})
	if err != nil {
		t.Fatalf("BuildDeathReport failed: %v", err)
	}
	if report.Verification != "structure_verified_replacement_events_not_kill_attribution" {
		t.Fatalf("unexpected verification label: %s", report.Verification)
	}
	if report.Summary.ReplacementEvents != 1 || len(report.Events) != 1 {
		t.Fatalf("replacement events = summary %d emitted %d, want 1: %+v", report.Summary.ReplacementEvents, len(report.Events), report.Events)
	}
	if report.Summary.AmbiguousFlatTransforms != 0 || len(report.Ambiguous) != 0 {
		t.Fatalf("ambiguous transforms = summary %d emitted %d, want 0: %+v", report.Summary.AmbiguousFlatTransforms, len(report.Ambiguous), report.Ambiguous)
	}
	event := report.Events[0]
	if event.PlayerID != 2 || event.LiveUnitID != 73 || event.LiveUnitName != "CHUKN" || event.CorpseUnitID != 28 || event.CorpseUnitName != "CHUKN_D" || event.UnitTypeSumDelta != -45 {
		t.Fatalf("bad death report event: %+v", event)
	}
	if event.Confidence != "corpse_table_engine_verified_replacement_not_kill_attribution" {
		t.Fatalf("bad confidence: %s", event.Confidence)
	}
	if report.CorpseTable.EngineVerified < 3 {
		t.Fatalf("corpse table engine-verified rows = %d, want at least 3", report.CorpseTable.EngineVerified)
	}
}

func TestKillFactoryCombatFiltersKnownArtifacts(t *testing.T) {
	fogPath := testfixtures.Path(t, "save-analysis/diagnostics/SP Replay v101.103.48987.0 @2026.08.18 035259.aoe2record")
	fogReport, err := BuildCombatStory(fogPath, CombatOptions{Limit: 0})
	if err != nil {
		t.Fatalf("BuildCombatStory fog failed: %v", err)
	}
	if fogReport.Summary.LossPulses != 0 || fogReport.Summary.NetObjectsLost != 0 {
		t.Fatalf("fog markers should not be combat losses: summary=%+v pulses=%+v", fogReport.Summary, fogReport.LossPulses)
	}
	if fogReport.Summary.FilteredArtifacts != 2 {
		t.Fatalf("fog filtered artifacts = %d, want 2", fogReport.Summary.FilteredArtifacts)
	}

	visiblePath := testfixtures.Path(t, "save-analysis/diagnostics/SP Replay v101.103.48987.0 @2026.08.18 132058.aoe2record")
	visibleReport, err := BuildCombatStory(visiblePath, CombatOptions{Limit: 0})
	if err != nil {
		t.Fatalf("BuildCombatStory visible failed: %v", err)
	}
	if visibleReport.Summary.LossPulses != 0 || visibleReport.Summary.NetObjectsLost != 0 {
		t.Fatalf("flare removals should not be combat losses: summary=%+v pulses=%+v", visibleReport.Summary, visibleReport.LossPulses)
	}
	if visibleReport.Summary.DeathReplacements != 1 || len(visibleReport.DeathEvents) != 1 {
		t.Fatalf("visible death replacements = summary %d events %d, want 1: %+v", visibleReport.Summary.DeathReplacements, len(visibleReport.DeathEvents), visibleReport.DeathEvents)
	}
	death := visibleReport.DeathEvents[0]
	if death.PlayerID != 2 || death.LiveUnitID != 73 || death.CorpseUnitID != 28 || death.UnitTypeSumDelta != -45 {
		t.Fatalf("bad death replacement event: %+v", death)
	}
	if visibleReport.Summary.FilteredArtifacts != 4 {
		t.Fatalf("visible filtered artifacts = %d, want 4", visibleReport.Summary.FilteredArtifacts)
	}
}

func TestKillFactoryPlayerSeriesSeparatesDeathReplacement(t *testing.T) {
	path := testfixtures.Path(t, "save-analysis/diagnostics/SP Replay v101.103.48987.0 @2026.08.18 132058.aoe2record")
	report, err := BuildPlayerSeries(path, PlayerSeriesOptions{PlayerID: 2})
	if err != nil {
		t.Fatalf("BuildPlayerSeries failed: %v", err)
	}
	if len(report.Players) != 1 {
		t.Fatalf("players = %d, want 1", len(report.Players))
	}
	player := report.Players[0]
	if player.DeathReplacements != 1 {
		t.Fatalf("death replacements = %d, want 1: %+v", player.DeathReplacements, player)
	}
	if player.LostEstimate != 0 || player.ObjectRemoves != 0 {
		t.Fatalf("death/corpse/flare artifacts should not become loss estimate: lost=%d removes=%d player=%+v", player.LostEstimate, player.ObjectRemoves, player)
	}
	if player.FilteredArtifactAdds != 1 || player.FilteredArtifactRemoves != 1 {
		t.Fatalf("filtered artifacts = %d/%d, want 1/1: %+v", player.FilteredArtifactAdds, player.FilteredArtifactRemoves, player)
	}
	if len(player.DeathReplacementUnits) != 1 || player.DeathReplacementUnits[0].UnitID != 73 || player.DeathReplacementUnits[0].Count != 1 {
		t.Fatalf("death replacement units = %+v, want one Chu Ko Nu", player.DeathReplacementUnits)
	}
}

func TestPlayerSeriesDiagnosticV2ObjectAdds(t *testing.T) {
	path := testfixtures.Path(t, "save-analysis/diagnostics/diag_v2_run_20260721.aoe2record")
	if _, err := os.Stat(path); err != nil {
		t.Skipf("golden replay not present: %v", err)
	}
	report, err := BuildPlayerSeries(path, PlayerSeriesOptions{PlayerID: 1, ChangesOnly: true, Limit: 4})
	if err != nil {
		t.Fatalf("BuildPlayerSeries failed: %v", err)
	}
	if report.Summary.Players != 1 {
		t.Fatalf("players = %d, want 1", report.Summary.Players)
	}
	if len(report.Players) != 1 {
		t.Fatalf("player rows = %d, want 1", len(report.Players))
	}
	player := report.Players[0]
	if player.ProducedEstimate != 6 || player.SingleObjectAdds != 6 {
		t.Fatalf("produced estimate/single adds = %d/%d, want 6/6: %+v", player.ProducedEstimate, player.SingleObjectAdds, player)
	}
	if player.LostEstimate != 0 {
		t.Fatalf("lost estimate = %d, want 0", player.LostEstimate)
	}
	wantAdded := map[int]int{83: 4, 448: 2}
	gotAdded := map[int]int{}
	for _, entry := range player.AddedUnitTypes {
		gotAdded[entry.UnitID] = entry.Count
	}
	for unitID, want := range wantAdded {
		if gotAdded[unitID] != want {
			t.Fatalf("added unit %d count = %d, want %d; all=%v", unitID, gotAdded[unitID], want, gotAdded)
		}
	}
	if len(player.SamplesOut) != 4 {
		t.Fatalf("emitted samples = %d, want limit 4", len(player.SamplesOut))
	}
	if report.Summary.EmittedDeltas == 0 {
		t.Fatalf("expected emitted deltas")
	}
}
