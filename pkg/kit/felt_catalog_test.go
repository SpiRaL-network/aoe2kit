package kit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateFeltCatalogsAllowsEntriesWithoutProvenance(t *testing.T) {
	root := t.TempDir()
	writeFeltCatalog(t, root, "art-felt-compendium.json", `{"entries":[{"name":"Plants"}]}`)
	writeFeltCatalog(t, root, "terrain-felt-compendium.json", `{"entries":[]}`)
	if err := validateFeltCatalogs(root); err != nil {
		t.Fatalf("catalogs without update_provenance should remain valid: %v", err)
	}
}

func TestValidateFeltCatalogsAcceptsProvenanceKinds(t *testing.T) {
	root := t.TempDir()
	writeFeltCatalog(t, root, "art-felt-compendium.json", `{"entries":[{"update_provenance":[{"kind":"feltboard_comment","id":"pin-12/comment-34","date":"2026-10-17"},{"kind":"tol_claim","id":"claim.felt.ground.pro_plant_cover","date":"2026-10-17"}]}]}`)
	if err := validateFeltCatalogs(root); err != nil {
		t.Fatalf("valid provenance was rejected: %v", err)
	}
}

func TestValidateFeltCatalogsRejectsInvalidProvenance(t *testing.T) {
	cases := []struct {
		name string
		item string
		want string
	}{
		{"not array", `{"kind":"tol_claim","id":"claim.felt.x","date":"2026-10-17"}`, "must be an array"},
		{"null", `null`, "must be an array"},
		{"invalid kind", `[{"kind":"other","id":"x","date":"2026-10-17"}]`, "kind must be"},
		{"missing id", `[{"kind":"feltboard_comment","date":"2026-10-17"}]`, "id must be a non-empty identifier"},
		{"wrong claim id", `[{"kind":"tol_claim","id":"felt.ground.x","date":"2026-10-17"}]`, "non-empty claim.felt.*"},
		{"invalid calendar date", `[{"kind":"feltboard_comment","id":"comment-1","date":"2026-02-30"}]`, "ISO date"},
		{"extra key", `[{"kind":"feltboard_comment","id":"comment-1","date":"2026-10-17","note":"extra"}]`, "only kind, id, and date"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			writeFeltCatalog(t, root, "art-felt-compendium.json", `{"entries":[{"update_provenance":`+tc.item+`}]}`)
			err := validateFeltCatalogs(root)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("validateFeltCatalogs() error = %v, want substring %q", err, tc.want)
			}
		})
	}
}

func writeFeltCatalog(t *testing.T, root, name, data string) {
	t.Helper()
	path := filepath.Join(root, "compendiums", "felt", "data", name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
}
