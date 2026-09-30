package replay

import (
	"aoe2kit/pkg/testfixtures"
	"os"
	"path/filepath"
	"testing"
)

func TestReplayCorpusSmallAnchors(t *testing.T) {
	dir := corpusFixtureDir(t,
		testfixtures.Path(t, "save-analysis/latest_test_20260720_233538.aoe2record"),
		testfixtures.Path(t, "save-analysis/latest_20260721_012221.aoe2record"),
	)
	report, err := BuildReplayCorpus(dir, CorpusOptions{Recursive: true, IncludeZip: true, UnknownSamples: 1})
	if err != nil {
		t.Fatalf("BuildReplayCorpus: %v", err)
	}
	if report.Summary.Scanned != 2 || report.Summary.Failures != 0 {
		t.Fatalf("summary scanned/failures = %d/%d, want 2/0", report.Summary.Scanned, report.Summary.Failures)
	}
	if report.Summary.HeaderBytes == 0 || report.Summary.BodyBytes == 0 {
		t.Fatalf("summary missing byte totals: %+v", report.Summary)
	}
	if report.Summary.HeaderDecodedPercentAvg == 0 || report.Summary.BodyDecodedPercentAvg == 0 {
		t.Fatalf("summary missing decode percentages: %+v", report.Summary)
	}
	if len(report.Rows) != 2 {
		t.Fatalf("rows = %d, want 2", len(report.Rows))
	}
	kinds := map[string]int{}
	for _, row := range report.Rows {
		if !row.OK {
			t.Fatalf("row failed: %+v", row)
		}
		if row.PostgameShape == "" {
			t.Fatalf("row missing postgame shape: %+v", row)
		}
		kinds[row.ReplayKind]++
	}
	if kinds["scenario"] == 0 || kinds["non_scenario"] == 0 {
		t.Fatalf("corpus should include scenario and non-scenario anchors, got %v", kinds)
	}
}

func TestOpaqueClustersSmallAnchors(t *testing.T) {
	dir := corpusFixtureDir(t,
		testfixtures.Path(t, "save-analysis/latest_test_20260720_233538.aoe2record"),
		testfixtures.Path(t, "save-analysis/latest_20260721_012221.aoe2record"),
	)
	report, err := BuildOpaqueClusters(dir, OpaqueClusterOptions{
		Recursive:   true,
		IncludeZip:  true,
		Space:       "inflated_header",
		MinBytes:    64,
		Limit:       10,
		SampleLimit: 1,
	})
	if err != nil {
		t.Fatalf("BuildOpaqueClusters: %v", err)
	}
	if report.Summary.Files != 2 || report.Summary.Failures != 0 {
		t.Fatalf("summary files/failures = %d/%d, want 2/0", report.Summary.Files, report.Summary.Failures)
	}
	if report.Summary.Spans == 0 || report.Summary.Clusters == 0 || len(report.Clusters) == 0 {
		t.Fatalf("expected opaque clusters, got %+v", report.Summary)
	}
	for _, cluster := range report.Clusters {
		if cluster.Key == "" || cluster.Space != "inflated_header" || cluster.Count == 0 || cluster.TotalBytes == 0 {
			t.Fatalf("bad cluster: %+v", cluster)
		}
	}
}

func corpusFixtureDir(t *testing.T, paths ...string) string {
	t.Helper()
	dir := t.TempDir()
	for i, path := range paths {
		if _, err := os.Stat(path); err != nil {
			t.Skipf("corpus fixture unavailable: %s: %v", path, err)
		}
		link := filepath.Join(dir, filepath.Base(path))
		if i > 0 && link == filepath.Join(dir, filepath.Base(paths[0])) {
			link = filepath.Join(dir, "fixture_"+filepath.Base(path))
		}
		if err := os.Symlink(path, link); err != nil {
			t.Fatalf("symlink fixture: %v", err)
		}
	}
	return dir
}
