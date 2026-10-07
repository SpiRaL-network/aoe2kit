package replay

import "testing"

func TestPostUpdateCivNames(t *testing.T) {
	for id, want := range map[int]string{60: "Saxons", 61: "Varangians", 62: "Danes"} {
		if got := CivDisplayName(id); got != want {
			t.Fatalf("civ %d=%q, want %q", id, got, want)
		}
	}
}
