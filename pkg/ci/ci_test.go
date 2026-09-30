package ci

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInitCreatesPortableSkeleton(t *testing.T) {
	dir := t.TempDir()
	report, err := Init(dir, "example.aoe2scenario", false)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Created) != 3 {
		t.Fatalf("created = %d, want 3 (%v)", len(report.Created), report.Created)
	}
	for _, path := range []string{
		filepath.Join(dir, DefaultConfigName),
		filepath.Join(dir, "ci", "verify-run.contract.json"),
		filepath.Join(dir, "xs", "a2k_debug.xs"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected %s: %v", path, err)
		}
	}
}

func TestCheckUnknownType(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, DefaultConfigName)
	if err := os.WriteFile(path, []byte(`{
  "schema_version": 1,
  "name": "test",
  "checks": [
    {"name": "mystery", "type": "not_real"}
  ]
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	report, err := CheckFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if report.OK {
		t.Fatal("report OK = true, want false")
	}
	if report.Summary.Unknown != 1 || len(report.Checks) != 1 || report.Checks[0].Status != "unknown" {
		t.Fatalf("unexpected report: %+v", report)
	}
}
