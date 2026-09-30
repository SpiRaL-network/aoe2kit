package kit

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestVerifyParsesDatArtifacts(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "README.md"), "readme")
	mustWrite(t, filepath.Join(root, "START_HERE.md"), "start")
	mustWrite(t, filepath.Join(root, "AI_GUIDE.md"), "guide")
	mustWrite(t, filepath.Join(root, "docs", "GO_AOE2KIT.md"), "go guide")
	mustWrite(t, filepath.Join(root, "docs", "DAT_ENGINE.md"), "dat")
	mustWrite(t, filepath.Join(root, "docs", "DAT_RECIPE_EXAMPLE.json"), "{}")
	mustWrite(t, filepath.Join(root, "KIT_MANIFEST.json"), string(mustManifestJSON(t)))
	mustWrite(t, filepath.Join(root, "broken.dat"), "not deflate")

	report := Verify(root, false)
	if report.ArtifactsOK {
		t.Fatal("expected artifact parse failure")
	}
	found := false
	for _, err := range report.Errors {
		if strings.Contains(err, "dat parse") && strings.Contains(err, "broken.dat") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected dat parse error, got %#v", report.Errors)
	}
}
