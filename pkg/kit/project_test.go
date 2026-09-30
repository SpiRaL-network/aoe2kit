package kit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestProjectSnapshotDiff(t *testing.T) {
	root := t.TempDir()
	mustWritePortable(t, filepath.Join(root, "docs", "README.md"), "first\n")
	beforePath := filepath.Join(root, "before.json")
	if _, err := WriteProjectSnapshot(root, beforePath); err != nil {
		t.Fatalf("before snapshot: %v", err)
	}

	mustWritePortable(t, filepath.Join(root, "docs", "README.md"), "second\n")
	mustWritePortable(t, filepath.Join(root, "outputs", "result.json"), "{}\n")
	afterPath := filepath.Join(root, "after.json")
	if _, err := WriteProjectSnapshot(root, afterPath); err != nil {
		t.Fatalf("after snapshot: %v", err)
	}

	diff, err := DiffProjectSnapshots(beforePath, afterPath, 0)
	if err != nil {
		t.Fatalf("diff snapshots: %v", err)
	}
	if diff.Same {
		t.Fatal("snapshot diff reported same")
	}
	if diff.Summary.Added != 2 || diff.Summary.Modified != 1 || diff.Summary.Removed != 0 {
		t.Fatalf("summary = %+v", diff.Summary)
	}
	var sawModified, sawAdded bool
	for _, change := range diff.Changes {
		if change.Path == "docs/README.md" && change.Change == "modified" {
			sawModified = true
		}
		if change.Path == "outputs/result.json" && change.Change == "added" {
			sawAdded = true
		}
	}
	if !sawModified || !sawAdded {
		t.Fatalf("changes = %+v", diff.Changes)
	}
}

func TestProjectSnapshotDiffLimit(t *testing.T) {
	root := t.TempDir()
	mustWritePortable(t, filepath.Join(root, "docs", "README.md"), "first\n")
	beforePath := filepath.Join(root, "before.json")
	if _, err := WriteProjectSnapshot(root, beforePath); err != nil {
		t.Fatalf("before snapshot: %v", err)
	}

	mustWritePortable(t, filepath.Join(root, "outputs", "one.json"), "{}\n")
	mustWritePortable(t, filepath.Join(root, "outputs", "two.json"), "{}\n")
	afterPath := filepath.Join(root, "after.json")
	if _, err := WriteProjectSnapshot(root, afterPath); err != nil {
		t.Fatalf("after snapshot: %v", err)
	}
	diff, err := DiffProjectSnapshots(beforePath, afterPath, 1)
	if err != nil {
		t.Fatalf("diff snapshots: %v", err)
	}
	if diff.Summary.Shown != 1 || diff.Summary.Limit != 1 {
		t.Fatalf("limited summary = %+v", diff.Summary)
	}
}

func TestLoadProjectSnapshotAcceptsRawInspectReport(t *testing.T) {
	root := t.TempDir()
	mustWritePortable(t, filepath.Join(root, "docs", "README.md"), "first\n")
	inspect, err := InspectProject(root, ProjectInspectOptions{})
	if err != nil {
		t.Fatalf("inspect: %v", err)
	}
	path := filepath.Join(root, "inspect.json")
	data, err := jsonMarshalForTest(inspect)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	snapshot, err := LoadProjectSnapshot(path)
	if err != nil {
		t.Fatalf("load raw inspect: %v", err)
	}
	if snapshot.Inspect.Root != root {
		t.Fatalf("snapshot root = %q, want %q", snapshot.Inspect.Root, root)
	}
}

func jsonMarshalForTest(v any) ([]byte, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}
