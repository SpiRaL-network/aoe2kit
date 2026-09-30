package replay

import "testing"

func TestDiffSyncReportsFinalMatrixDeltas(t *testing.T) {
	before := checksumReportWithFinalMatrix(1000, [][]uint32{
		syncRow(100, 200, 300, 400, 500),
		nil,
	})
	after := checksumReportWithFinalMatrix(2000, [][]uint32{
		syncRow(103, 283, 301, 407, 509),
		nil,
	})
	report := diffSyncReports(before, after)
	if report.BeforeChecksumSamples != 1 || report.AfterChecksumSamples != 1 {
		t.Fatalf("checksum samples = %d/%d, want 1/1", report.BeforeChecksumSamples, report.AfterChecksumSamples)
	}
	if report.ComparablePlayers != 1 || report.ChangedPlayers != 1 {
		t.Fatalf("players comparable/changed = %d/%d, want 1/1", report.ComparablePlayers, report.ChangedPlayers)
	}
	if report.ChangedWords != 5 {
		t.Fatalf("changed words = %d, want 5: %+v", report.ChangedWords, report.WordDeltas)
	}
	player := report.PlayerDeltas[0]
	if player.PlayerID != 1 || player.ObjectCountDelta != 3 || player.UnitTypeSumDelta != 83 || player.ObjectIDSumDelta != 9 ||
		player.PositionSumDelta != 7 || player.Word3Delta != 1 {
		t.Fatalf("player delta = %+v", player)
	}
	want := map[int]int64{2: 83, 3: 1, 6: 3, 7: 7, 10: 9}
	for _, delta := range report.WordDeltas {
		if delta.WordName != syncWordName(delta.WordIndex) {
			t.Fatalf("word name = %q, want %q", delta.WordName, syncWordName(delta.WordIndex))
		}
		if want[delta.WordIndex] != delta.Delta {
			t.Fatalf("word %d delta = %d, want %d", delta.WordIndex, delta.Delta, want[delta.WordIndex])
		}
		if delta.WordIndex == 10 && delta.WordName != "word_10" {
			t.Fatalf("word_10 name = %q", delta.WordName)
		}
	}
}

func TestDiffSyncReportsNoChecksumMatrix(t *testing.T) {
	report := diffSyncReports(&SyncReport{}, &SyncReport{})
	if len(report.Warnings) == 0 {
		t.Fatal("expected missing checksum warning")
	}
	if report.ChangedWords != 0 || len(report.WordDeltas) != 0 {
		t.Fatalf("unexpected deltas without checksum matrices: %+v", report)
	}
}

func TestChecksumWordDeltaUsesSignedWrap(t *testing.T) {
	if got := checksumWordDelta(4294967000, 200); got != 496 {
		t.Fatalf("wrapped positive delta = %d, want 496", got)
	}
	if got := checksumWordDelta(200, 4294967000); got != -496 {
		t.Fatalf("wrapped negative delta = %d, want -496", got)
	}
}

func checksumReportWithFinalMatrix(durationMS int, rows [][]uint32) *SyncReport {
	matrix := make([][]uint32, 8)
	for i := 0; i < 8; i++ {
		matrix[i] = make([]uint32, 11)
		if i < len(rows) && rows[i] != nil {
			copy(matrix[i], rows[i])
		}
	}
	return &SyncReport{
		Summary: SyncSummary{
			ChecksumDE: 1,
			DurationMS: durationMS,
			Duration:   FormatTime(durationMS),
		},
		Events: []SyncEvent{{TimeMS: durationMS, Time: FormatTime(durationMS), Form: "checksum_de", Matrix: matrix}},
	}
}

func syncRow(objectCount, unitTypeSum, word3, positionSum, objectIDSum uint32) []uint32 {
	row := make([]uint32, 11)
	row[2] = unitTypeSum
	row[3] = word3
	row[6] = objectCount
	row[7] = positionSum
	row[8] = 1
	row[10] = objectIDSum
	return row
}
