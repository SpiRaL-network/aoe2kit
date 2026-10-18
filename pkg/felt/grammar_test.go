package felt

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestReviewedConnectorsAndUnmappedVariants(t *testing.T) {
	c, err := Load("")
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []int{1062, 1888, 1885, 1886, 2274, 2285, 2037, 2039, 2041, 2043, 2081, 2651, 264, 2748} {
		if _, ok := c.Find(id); !ok {
			t.Fatalf("missing seed %d", id)
		}
	}
	if got, err := c.Rotate(1062, 0, 90); err != nil || got != 1 {
		t.Fatalf("fence rotation %d %v", got, err)
	}
	if _, err := c.Rotate(2043, 0, 90); err == nil {
		t.Fatal("inferred unsupported bridge rotation")
	}
	e, _ := c.Find(2651)
	e, err = Skeleton(e, 25)
	if err != nil {
		t.Fatal(err)
	}
	if len(e.Grammar) != 25 || e.Grammar[0].Evidence != "unmapped" || e.Grammar[16].Role != "sloped_end" {
		t.Fatalf("skeleton %+v", e)
	}
	path := filepath.Join(t.TempDir(), "review.json")
	b, _ := json.Marshal(Catalog{Schema: "aoe2kit.felt.connectors.v1", Entries: []Entry{e}})
	if err := os.WriteFile(path, b, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err != nil {
		t.Fatal(err)
	}
	if _, err := Skeleton(e, 3); err == nil {
		t.Fatal("accepted variants outside DAT")
	}
}

func TestGrammarRejectsInvalidMappings(t *testing.T) {
	for _, v := range []Variant{
		{Index: 0, Role: "run", Axis: "z", Evidence: "preview_reviewed"},
		{Index: 0, Role: "run", Evidence: "unmapped", RotatesTo: map[int]int{90: 0}},
		{Index: 0, Role: "run", Evidence: "preview_reviewed", RotatesTo: map[int]int{90: 1}},
		{Index: 0, Role: "run", Evidence: "preview_reviewed", RotatesTo: map[int]int{45: 0}},
	} {
		if err := (Catalog{Entries: []Entry{{UnitID: 1, Grammar: []Variant{v}}}}).Validate(); err == nil {
			t.Fatalf("accepted %+v", v)
		}
	}
}
