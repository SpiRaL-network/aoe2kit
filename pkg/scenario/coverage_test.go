package scenario

import (
	"path/filepath"
	"testing"
)

func TestCoverageFileAccountsFixtureAndReportsDarkBytes(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "editor-refs", "Create scenario, do nothing, click save.aoe2scenario")
	report, err := CoverageFile(path)
	if err != nil {
		t.Fatalf("CoverageFile: %v", err)
	}
	if report.InflatedBodyBytes == 0 || report.AccountedBytes != report.HeaderBytes+report.InflatedBodyBytes {
		t.Fatalf("coverage accounting = %+v", report)
	}
	if report.GapBytes != 0 {
		t.Fatalf("gap bytes = %d, want complete structural accounting", report.GapBytes)
	}
	if report.DarkBytes == 0 {
		t.Fatalf("dark bytes = 0, want anonymous fixture fields to remain visible")
	}
	if len(report.Sections) < 2 {
		t.Fatalf("sections = %d, want header plus body sections", len(report.Sections))
	}
	darkSpans := 0
	for _, section := range report.Sections {
		darkSpans += len(section.DarkSpans)
	}
	if darkSpans == 0 {
		t.Fatal("dark spans = 0, want field-level attribution")
	}
}

func TestCanonicalCoveragePathRemovesOnlyNumericIndexes(t *testing.T) {
	got := canonicalCoveragePath("Triggers.trigger_data[12].condition_data[3].unknown_2[abc]")
	want := "Triggers.trigger_data[].condition_data[].unknown_2[abc]"
	if got != want {
		t.Fatalf("canonical path=%q, want %q", got, want)
	}
}
