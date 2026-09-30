package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The catalog is only trustworthy if it cannot silently fall behind the CLI.
// These tests are the enforcement: add a command to usage() without declaring
// its traits and the build fails here, which is the whole point of moving the
// classification out of downstream tooling.

func TestCommandCatalogCoversUsage(t *testing.T) {
	declared := map[string]bool{}
	for _, spec := range commandCatalog {
		declared[spec.Name] = true
	}
	for _, name := range usageCommandNames(usageText()) {
		if !declared[name] {
			t.Errorf("command %q appears in usage() but has no entry in commandCatalog "+
				"(add one in commands_catalog.go declaring its trait and input)", name)
		}
	}
}

func TestSiblingTypesDocumentMatchesRunSpecificSidecar(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "ab2run.xsdat")
	if err := os.WriteFile(input, nil, 0644); err != nil {
		t.Fatal(err)
	}
	doc := "# A2K_BARRACKS_WALL_V2.xsdat\nheader = string,int,string\nrow = string,int,int,float,float,float,float,float,float\nfooter_integrity = 424242\n"
	if err := os.WriteFile(filepath.Join(dir, "ab2.types"), []byte(doc), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "unrelated.types"), []byte("# unknown.xsdat\nheader = string,int\n"), 0644); err != nil {
		t.Fatal(err)
	}
	got := siblingTypesDocument(input)
	if got != filepath.Join(dir, "ab2.types") {
		t.Fatalf("types discovery=%q, want ab2.types", got)
	}
}

func TestCommandCatalogHasNoStaleEntries(t *testing.T) {
	documented := map[string]bool{}
	for _, name := range usageCommandNames(usageText()) {
		documented[name] = true
	}
	for _, spec := range commandCatalog {
		if !documented[spec.Name] {
			t.Errorf("commandCatalog declares %q but usage() does not document it", spec.Name)
		}
	}
}

func TestCommandCatalogTraitsAreValid(t *testing.T) {
	valid := map[string]bool{TraitReadOnly: true, TraitWrites: true, TraitNetwork: true}
	for _, spec := range commandCatalog {
		if !valid[spec.Trait] {
			t.Errorf("command %q has invalid trait %q", spec.Name, spec.Trait)
		}
		if spec.Input == "" {
			t.Errorf("command %q has no declared input kind", spec.Name)
		}
	}
}

func TestFormatSchemaFieldsPreservesEveryDecodedField(t *testing.T) {
	text := formatSchemaFields(map[string]any{
		"ball_z":     float32(1),
		"expected_z": float32(2),
		"ruler_z4":   float32(4),
	})
	for _, field := range []string{"ball_z=1", "expected_z=2", "ruler_z4=4"} {
		if !strings.Contains(text, field) {
			t.Fatalf("formatted fields=%q, missing %q", text, field)
		}
	}
}
