package enginefacts

import (
	"os"
	"path/filepath"
	"testing"
)

// The facts ledger is private workshop data and does not ship in the public
// release, so this test runs only where the ledger is present.
func TestLoadDefaultFactsLedger(t *testing.T) {
	if _, err := os.Stat(filepath.Join("..", "..", "data", "engine_facts.json")); err != nil {
		t.Skip("engine facts ledger not present (private data)")
	}
	ledger, err := LoadDefault()
	if err != nil {
		t.Fatal(err)
	}
	if len(ledger.Facts) < 10 {
		t.Fatalf("facts = %d, want seeded ledger", len(ledger.Facts))
	}
	fact, ok := ledger.ByID("text.trigger_color_first_tag_only")
	if !ok {
		t.Fatal("missing text.trigger_color_first_tag_only")
	}
	if fact.Tier != EngineVerified || fact.VerifiedDate == "" || fact.FixtureRef == "" {
		t.Fatalf("fact citation incomplete: %+v", fact)
	}
	if len(ledger.Filter(false, "")) >= len(ledger.Facts) {
		t.Fatalf("engine-verified filter did not hide provisional facts")
	}
	if len(ledger.Filter(true, "xs")) == 0 {
		t.Fatalf("domain filter xs returned no facts")
	}
}
