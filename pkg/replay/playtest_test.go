package replay

import (
	"reflect"
	"testing"

	"aoe2kit/pkg/testfixtures"
)

func TestBuildPlaytestStateWindowShowsChangedAndUnchangedPlayers(t *testing.T) {
	players := []PlaytestPlayer{
		{PlayerID: 1, Label: "P1", Name: "One", Human: true},
		{PlayerID: 2, Label: "P2", Name: "Two", Human: true},
		{PlayerID: 3, Label: "P3", Name: "Three", Human: true},
	}
	samples := []SyncEvent{
		{TimeMS: 1000, Time: FormatTime(1000), Form: "checksum_de", Matrix: make([][]uint32, 8)},
		{TimeMS: 2000, Time: FormatTime(2000), Form: "checksum_de", Matrix: make([][]uint32, 8)},
		{TimeMS: 3000, Time: FormatTime(3000), Form: "checksum_de", Matrix: make([][]uint32, 8)},
		{TimeMS: 4000, Time: FormatTime(4000), Form: "checksum_de", Matrix: make([][]uint32, 8)},
	}
	deltas := []SyncStateDelta{
		{PlayerID: 2, FromTimeMS: 2000, ToTimeMS: 3000, Kind: "object_replaced_or_transformed", UnitTypeSumDelta: 99, Confidence: "checksum_delta_observed"},
		{PlayerID: 3, FromTimeMS: 3000, ToTimeMS: 4000, Kind: "object_added", ObjectCountDelta: 1, UnitTypeSumDelta: 83, UnitID: 83, UnitName: "VILLAGER_MALE", Confidence: "fixture"},
	}

	window := buildPlaytestStateWindow(3500, 1, samples, deltas, players)
	if window.StartTimeMS != 2000 || window.EndTimeMS != 4000 {
		t.Fatalf("window range = %d..%d, want 2000..4000", window.StartTimeMS, window.EndTimeMS)
	}
	if len(window.Intervals) != 2 {
		t.Fatalf("intervals = %d, want 2", len(window.Intervals))
	}
	focus := window.Intervals[window.FocusIntervalIndex]
	if focus.FromTimeMS != 2000 || focus.ToTimeMS != 3000 {
		t.Fatalf("focus interval = %d..%d, want 2000..3000", focus.FromTimeMS, focus.ToTimeMS)
	}
	if focus.Players[0].Kind != "no_change" || focus.Players[0].Changed {
		t.Fatalf("P1 change = %+v, want no_change", focus.Players[0])
	}
	if focus.Players[1].Kind != "object_replaced_or_transformed" || !focus.Players[1].Changed {
		t.Fatalf("P2 change = %+v, want replacement", focus.Players[1])
	}
	changed, unchanged := summarizePlaytestSplit(window, players)
	if len(changed) != 1 || changed[0] != "P2 Two" {
		t.Fatalf("changed = %#v, want P2 Two", changed)
	}
	if len(unchanged) != 2 || unchanged[0] != "P1 One" || unchanged[1] != "P3 Three" {
		t.Fatalf("unchanged = %#v, want P1/P3", unchanged)
	}
}

func TestPlaytestReportRPGFourPlayerFixtureFocusesAuthorSignal(t *testing.T) {
	path := testfixtures.Path(t, "save-analysis/diagnostics/latest_0819.aoe2record")
	report, err := BuildPlaytestReport(path, PlaytestOptions{Window: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Moments) != 1 {
		t.Fatalf("moments = %d, want 1 collapsed feedback moment", len(report.Moments))
	}
	moment := report.Moments[0]
	if moment.RecordedCount != 2 {
		t.Fatalf("recorded_count = %d, want 2", moment.RecordedCount)
	}
	wantChanged := []string{"P3 chrae", "P4 Sekiro"}
	if !reflect.DeepEqual(moment.ChangedPlayers, wantChanged) {
		t.Fatalf("changed = %#v, want %#v", moment.ChangedPlayers, wantChanged)
	}
	wantUnchanged := []string{"P1 SDS | krmyth9", "P2 SunWooKooong"}
	if !reflect.DeepEqual(moment.UnchangedPlayers, wantUnchanged) {
		t.Fatalf("unchanged = %#v, want %#v", moment.UnchangedPlayers, wantUnchanged)
	}
	if len(moment.Window.Intervals) <= moment.Window.FocusIntervalIndex {
		t.Fatalf("focus interval index %d outside %d intervals", moment.Window.FocusIntervalIndex, len(moment.Window.Intervals))
	}
	focus := moment.Window.Intervals[moment.Window.FocusIntervalIndex]
	if focus.FromTime != "00:18:03.290" || focus.ToTime != "00:18:34.290" {
		t.Fatalf("focus interval = %s -> %s, want 00:18:03.290 -> 00:18:34.290", focus.FromTime, focus.ToTime)
	}
}
