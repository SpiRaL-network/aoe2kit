package kit

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBuildArtifactLineage(t *testing.T) {
	root := t.TempDir()
	input := filepath.Join(root, "input.aoe2scenario")
	output := filepath.Join(root, "output.aoe2scenario")
	mustWritePortable(t, input, "before\n")
	mustWritePortable(t, output, "after\n")

	report, err := BuildArtifactLineage([]string{input}, []string{output}, ArtifactLineageOptions{
		Tool:    "kit scen patch",
		Command: []string{"kit", "scen", "patch", "input.aoe2scenario", "output.aoe2scenario", "--recipe", "recipe.json"},
		Notes:   []string{"test note"},
	})
	if err != nil {
		t.Fatalf("lineage: %v", err)
	}
	if !report.OK || report.Verification != "structure_verified_not_engine_verified" {
		t.Fatalf("report = %+v", report)
	}
	if len(report.Inputs) != 1 || len(report.Outputs) != 1 {
		t.Fatalf("input/output counts = %d/%d", len(report.Inputs), len(report.Outputs))
	}
	if report.Inputs[0].SHA256 == report.Outputs[0].SHA256 {
		t.Fatalf("expected different hashes, got %s", report.Inputs[0].SHA256)
	}
}

func TestWriteArtifactLineage(t *testing.T) {
	root := t.TempDir()
	input := filepath.Join(root, "input.txt")
	output := filepath.Join(root, "output.txt")
	manifest := filepath.Join(root, "lineage.json")
	mustWritePortable(t, input, "before\n")
	mustWritePortable(t, output, "after\n")

	report, err := BuildArtifactLineage([]string{input}, []string{output}, ArtifactLineageOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if err := WriteArtifactLineage(manifest, report); err != nil {
		t.Fatalf("write lineage: %v", err)
	}
	if _, err := os.Stat(manifest); err != nil {
		t.Fatalf("stat manifest: %v", err)
	}
}
