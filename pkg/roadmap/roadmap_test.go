package roadmap

import (
	"strings"
	"testing"
)

func TestRWDMatrixFiltersByDomain(t *testing.T) {
	report, err := RWDMatrix("dat")
	if err != nil {
		t.Fatalf("RWDMatrix returned error: %v", err)
	}
	if report.Domain != "dat" {
		t.Fatalf("domain = %q, want dat", report.Domain)
	}
	if len(report.Rows) == 0 {
		t.Fatal("expected dat rows")
	}
	for _, row := range report.Rows {
		if row.Domain != "dat" {
			t.Fatalf("unexpected row domain %q", row.Domain)
		}
	}
	if report.Summary.Rows != len(report.Rows) {
		t.Fatalf("summary rows = %d, want %d", report.Summary.Rows, len(report.Rows))
	}
	if report.Summary.ReadFull == 0 {
		t.Fatal("expected at least one full read surface")
	}
}

func TestCRUDMatrixNamesUpdateAndDeleteModes(t *testing.T) {
	report, err := CRUDMatrix("dat")
	if err != nil {
		t.Fatalf("CRUDMatrix returned error: %v", err)
	}
	if report.Domain != "dat" {
		t.Fatalf("domain = %q, want dat", report.Domain)
	}
	if report.Summary.Rows != len(report.Rows) {
		t.Fatalf("summary rows = %d, want %d", report.Summary.Rows, len(report.Rows))
	}
	if report.Summary.ReadFull == 0 || report.Summary.CreateFull == 0 {
		t.Fatalf("expected read/create coverage in summary: %+v", report.Summary)
	}
	if report.Summary.PartialCapabilities == 0 {
		t.Fatalf("expected partial capabilities in summary: %+v", report.Summary)
	}

	var foundEffects bool
	for _, row := range report.Rows {
		if row.Domain != "dat" {
			t.Fatalf("unexpected row domain %q", row.Domain)
		}
		if row.Section == "effects and commands" {
			foundEffects = true
			if row.Update.Status != "partial" {
				t.Fatalf("effects update status = %q, want partial", row.Update.Status)
			}
			if row.Delete.Mode != "guarded_physical_or_semantic" {
				t.Fatalf("effects delete mode = %q, want guarded_physical_or_semantic", row.Delete.Mode)
			}
		}
	}
	if !foundEffects {
		t.Fatal("missing effects and commands row")
	}
}

func TestScenarioRoadmapDoesNotAdvertiseGaiaInactiveAuthoring(t *testing.T) {
	report, err := RWDMatrix("scenario")
	if err != nil {
		t.Fatalf("RWDMatrix returned error: %v", err)
	}
	var found bool
	for _, row := range report.Rows {
		if row.Section != "players, diplomacy, resources, victory" {
			continue
		}
		found = true
		text := row.Write + " " + row.Next
		if strings.Contains(text, "explicit Gaia active-state control") {
			t.Fatalf("roadmap still advertises explicit Gaia active-state control: %+v", row)
		}
		if !strings.Contains(text, "Gaia-active blank baseline") {
			t.Fatalf("roadmap should describe Gaia as a baseline, not a disable target: %+v", row)
		}
		if !strings.Contains(text, "do not advertise Gaia-inactive authoring") {
			t.Fatalf("roadmap should preserve the Gaia-inactive caveat: %+v", row)
		}
	}
	if !found {
		t.Fatal("missing scenario players/diplomacy/resources/victory row")
	}
}

func TestDarkBytesIncludesReplayPriorityOne(t *testing.T) {
	report, err := DarkBytes("replay")
	if err != nil {
		t.Fatalf("DarkBytes returned error: %v", err)
	}
	if len(report.Frontiers) == 0 {
		t.Fatal("expected replay frontiers")
	}
	if report.Summary.Priority1 == 0 {
		t.Fatal("expected at least one priority 1 replay frontier")
	}
	for _, frontier := range report.Frontiers {
		if frontier.Domain != "replay" {
			t.Fatalf("unexpected frontier domain %q", frontier.Domain)
		}
	}
}

func TestDarkBytesTracksCurrentBacklog(t *testing.T) {
	report, err := DarkBytes("all")
	if err != nil {
		t.Fatalf("DarkBytes returned error: %v", err)
	}
	if report.Version != "2026-09-27" {
		t.Fatalf("roadmap version = %q, want 2026-09-27", report.Version)
	}
	want := []struct {
		domain string
		region string
	}{
		{"scenario", "placement snap and footprint intent"},
		{"dat", "full unit decode and SQL export"},
		{"replay", "sidecar schema pipeline"},
		{"scenario", "XS safety and differential lint"},
		{"scenario", "load-safety checkpoints"},
	}
	for _, item := range want {
		found := false
		for _, frontier := range report.Frontiers {
			if frontier.Domain == item.domain && frontier.Region == item.region {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("missing current backlog frontier %s/%s", item.domain, item.region)
		}
	}
}

func TestDarkBytesSyncMatrixTracksPromotedWordSemantics(t *testing.T) {
	report, err := DarkBytes("replay")
	if err != nil {
		t.Fatalf("DarkBytes returned error: %v", err)
	}
	var syncMatrix *DarkFrontier
	for i := range report.Frontiers {
		if report.Frontiers[i].Region == "sync matrix words" {
			syncMatrix = &report.Frontiers[i]
			break
		}
	}
	if syncMatrix == nil {
		t.Fatal("missing sync matrix frontier")
	}
	for _, want := range []string{
		"word_4 is current object carry sum",
		"word_7 is checksum-counted object position sum",
		"word_3 is a strong object-state-sum hypothesis",
		"word_0/5/9 unresolved",
	} {
		if !strings.Contains(syncMatrix.Known+" "+syncMatrix.Unknown+" "+syncMatrix.Verification, want) {
			t.Fatalf("sync matrix frontier missing %q:\n%+v", want, *syncMatrix)
		}
	}
}

func TestUnknownDomainFails(t *testing.T) {
	if _, err := RWDMatrix("worms"); err == nil {
		t.Fatal("expected unknown domain error")
	}
	if _, err := CRUDMatrix("worms"); err == nil {
		t.Fatal("expected unknown domain error")
	}
	if _, err := DarkBytes("worms"); err == nil {
		t.Fatal("expected unknown domain error")
	}
}
