package xs

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteTypesDocument(t *testing.T) {
	path := filepath.Join(t.TempDir(), "run.types")
	document := "# A2K_CUSTOM_STORE.xsdat\nheader = string,int\n"
	if err := WriteTypesDocument(path, document); err != nil {
		t.Fatal(err)
	}
	if got, err := os.ReadFile(path); err != nil || string(got) != document {
		t.Fatalf("types document = %q, err=%v", got, err)
	}
	if err := WriteTypesDocument(filepath.Join(t.TempDir(), "empty.types"), " "); err == nil {
		t.Fatal("empty types document unexpectedly accepted")
	}
	if err := WriteTypesDocument(filepath.Join(t.TempDir(), "bad.types"), "# A2K_CUSTOM_STORE.xsdat\n"); err == nil {
		t.Fatal("types document without header unexpectedly accepted")
	}
}

func TestDiscoverSiblingTypesFileRejectsAmbiguity(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "run.xsdat")
	if err := os.WriteFile(input, nil, 0644); err != nil {
		t.Fatal(err)
	}
	for name, identity := range map[string]string{
		"custom-store.types": "A2K_CUSTOM_STORE",
		"second.types":       "A2K_BARRACKS_WALL_V2",
	} {
		document := "# " + identity + ".xsdat\nheader = string,int\n"
		if err := os.WriteFile(filepath.Join(dir, name), []byte(document), 0644); err != nil {
			t.Fatal(err)
		}
	}
	if got := DiscoverSiblingTypesFile(input); got != "" {
		t.Fatalf("ambiguous discovery = %q, want empty", got)
	}

	matching := filepath.Join(dir, "custom-store-run.xsdat")
	if err := os.WriteFile(matching, nil, 0644); err != nil {
		t.Fatal(err)
	}
	if got := DiscoverSiblingTypesFile(matching); got != filepath.Join(dir, "custom-store.types") {
		t.Fatalf("matching discovery = %q, want custom-store.types", got)
	}
}
