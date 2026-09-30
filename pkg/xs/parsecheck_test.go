package xs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeChecker(t *testing.T, output string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "checker.sh")
	script := "#!/bin/sh\nprintf '%s\\n' '" + strings.ReplaceAll(output, "'", "'\\''") + "'\n"
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestCheckFileTreatsDiagnosticOutputAsFailure(t *testing.T) {
	checker := writeChecker(t, "Error: invalid expression")
	path := filepath.Join(t.TempDir(), "runtime.xs")
	if err := os.WriteFile(path, []byte("void main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	report, err := CheckFile(path, checker)
	if err == nil || report.Status != "failed" {
		t.Fatalf("report=%+v err=%v, want diagnostic failure", report, err)
	}
}

func TestCheckFilePassesCleanOutput(t *testing.T) {
	checker := writeChecker(t, "Finished analysing file")
	path := filepath.Join(t.TempDir(), "runtime.xs")
	if err := os.WriteFile(path, []byte("void main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	report, err := CheckFile(path, checker)
	if err != nil || report.Status != "passed" {
		t.Fatalf("report=%+v err=%v, want pass", report, err)
	}
}

func TestCheckASCIIFileRejectsNonASCIIBytes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "unicode.xs")
	if err := os.WriteFile(path, []byte("void main() { xsChatData(\"coin ™\"); }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := CheckASCIIFile(path); err == nil || !strings.Contains(err.Error(), "non-ASCII byte") {
		t.Fatalf("CheckASCIIFile error = %v, want non-ASCII diagnostic", err)
	}
}

func TestCheckFileRejectsNonASCIIBeforeExternalChecker(t *testing.T) {
	checker := writeChecker(t, "this checker should not run")
	path := filepath.Join(t.TempDir(), "unicode.xs")
	if err := os.WriteFile(path, []byte("// café\nvoid main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	report, err := CheckFile(path, checker)
	if err == nil || report.Status != "failed" || !strings.Contains(report.Output, "non-ASCII byte") {
		t.Fatalf("report=%+v err=%v, want preflight failure", report, err)
	}
}

func TestExternalFindingsAndDifferential(t *testing.T) {
	report := ParseCheckReport{Status: "passed", Output: "[109] Warning: NoNumPromo\n  /tmp/runtime.xs:7:4\n"}
	external := ExternalFindings(report)
	if len(external) != 1 || external[0].Code != "NoNumPromo" || external[0].Line != 7 {
		t.Fatalf("external findings=%+v", external)
	}
	kit := []SourceFinding{{Code: "xs_numeric_promotion", Severity: "warning", File: "/tmp/runtime.xs", Line: 7, Message: "same line"}}
	diff := CompareFindings(kit, external)
	if len(diff) != 1 || diff[0].Class != "BOTH" {
		t.Fatalf("differential=%+v", diff)
	}
	kit = append(kit, SourceFinding{Code: "kit_only", Severity: "error", File: "/tmp/runtime.xs", Line: 9, Message: "kit-only"})
	diff = CompareFindings(kit, external)
	if len(diff) != 2 || diff[1].Class != "KIT_ONLY" {
		t.Fatalf("kit-only differential=%+v", diff)
	}
}
