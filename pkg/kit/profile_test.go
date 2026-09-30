package kit

import (
	"os"
	"path/filepath"
	"testing"
)

// A redistribution gate must not let "I cannot tell what this archive claims to
// be" pass as "this archive is fine". These tests pin that behavior.

func TestReadProfileMarkerAbsentMeansFull(t *testing.T) {
	dir := t.TempDir()
	marker, err := ReadProfileMarker(dir)
	if err != nil {
		t.Fatalf("absent marker should not error: %v", err)
	}
	if marker.Profile != string(ProfileFull) {
		t.Errorf("absent marker = %q, want full (every existing checkout has no marker)", marker.Profile)
	}
}

func TestReadProfileMarkerUnknownProfileIsRejected(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, `{"profile":"minimal","generation":3}`)
	marker, err := ReadProfileMarker(dir)
	if err == nil {
		t.Fatal("unknown profile must be reported, not silently treated as full")
	}
	if marker.Profile != "minimal" {
		t.Errorf("marker profile = %q, want the declared value preserved for the report", marker.Profile)
	}
}

func TestReadProfileMarkerMalformedIsRejected(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, `{"profile":`)
	if _, err := ReadProfileMarker(dir); err == nil {
		t.Fatal("malformed marker must be reported")
	}
	write(t, dir, `{"generation":2}`)
	if _, err := ReadProfileMarker(dir); err == nil {
		t.Fatal("marker with no profile must be reported")
	}
}

func TestVerifyFailsOnUnknownProfileMarker(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, `{"profile":"totally-made-up"}`)
	report := Verify(dir, false)
	found := false
	for _, e := range report.Errors {
		if len(e) >= 21 && e[:21] == "unknown KIT_PROFILE p" {
			found = true
		}
	}
	if !found {
		t.Errorf("Verify must reject an unknown profile marker; errors = %v", report.Errors)
	}
}

// RequiredDocs is the promise a profile makes. It must be derived from the same
// exclude rules that drive packing, or the two can disagree.
func TestRequiredDocsShrinksForReducedProfiles(t *testing.T) {
	full := RequiredDocs(ProfileFull)
	handoff := RequiredDocs(ProfileHandoff)
	if len(handoff) >= len(full) {
		t.Errorf("handoff requires %d docs, full requires %d; handoff must require fewer", len(handoff), len(full))
	}
	for _, doc := range handoff {
		if len(doc) >= 5 && doc[:5] == "docs/" {
			t.Errorf("handoff must not require %q: the profile excludes docs/", doc)
		}
	}
}

func TestRequiredDocsForPublicProfileKeepsPublicDocs(t *testing.T) {
	public, err := RequiredDocsIn(filepath.Join("..", ".."), ProfilePublic)
	if err != nil {
		t.Fatal(err)
	}
	have := map[string]bool{}
	for _, doc := range public {
		have[doc] = true
		if doc == "AUDITS.md" || doc == "BRIEFING.md" || doc == "KIT_PARITY_ROADMAP.md" {
			t.Errorf("public profile must not require internal handoff doc %q", doc)
		}
		if len(doc) >= len("docs/REPLAY_TRANSPARENCY_") && doc[:len("docs/REPLAY_TRANSPARENCY_")] == "docs/REPLAY_TRANSPARENCY_" {
			t.Errorf("public profile must not require diagnostic doc %q", doc)
		}
	}
	for _, doc := range []string{"README.md", "NOTICE.md", "CONTRIBUTING.md", "SECURITY.md", "docs/GO_AOE2KIT.md", "data/credits.json"} {
		if !have[doc] {
			t.Errorf("public profile missing public artifact %q", doc)
		}
	}
}

func write(t *testing.T, dir, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, ProfileMarkerName), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
