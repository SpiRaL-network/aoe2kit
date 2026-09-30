package kit

import (
	"bufio"
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestNoAbsolutePaths is deliberately a Go test rather than a release-only
// script: a newly tracked source/data file fails during ordinary go test.
func TestNoAbsolutePaths(t *testing.T) {
	cmd := exec.Command("git", "ls-files", "-z")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git ls-files: %v", err)
	}
	for _, raw := range bytes.Split(out, []byte{0}) {
		if len(raw) == 0 {
			continue
		}
		rel := filepath.ToSlash(string(raw))
		if absolutePathAllowlisted(rel) {
			continue
		}
		data, err := readTrackedText(rel)
		if err != nil {
			t.Fatalf("read tracked file %s: %v", rel, err)
		}
		if data == nil {
			continue
		}
		scanner := bufio.NewScanner(bytes.NewReader(data))
		line := 0
		for scanner.Scan() {
			line++
			if needle := absolutePathNeedle(scanner.Bytes()); needle != "" {
				t.Errorf("%s:%d: machine-specific path %q; use testdata or an opt-in corpus root", rel, line, needle)
			}
		}
		if err := scanner.Err(); err != nil {
			t.Fatalf("scan tracked file %s: %v", rel, err)
		}
	}
}

func readTrackedText(rel string) ([]byte, error) {
	data, err := os.ReadFile(rel)
	if err != nil {
		return nil, err
	}
	if looksBinary(data) {
		return nil, nil
	}
	return data, nil
}

func absolutePathNeedle(line []byte) string {
	text := string(line)
	needles := []string{
		"/" + "home/",
		"/" + "Users/",
		"~" + "/",
		"/" + "tmp/",
		"C:" + "\\Users\\",
		"C:" + string([]byte{47}) + "Users/",
	}
	for _, needle := range needles {
		if strings.Contains(text, needle) {
			return needle
		}
	}
	return ""
}

func absolutePathAllowlisted(rel string) bool {
	if strings.HasPrefix(rel, "docs/") || strings.HasPrefix(rel, "tools/") || strings.HasPrefix(rel, "DEX_TASK_") {
		return true
	}
	switch rel {
	case "AI_GUIDE.md", "project.go", "pack_test.go", "portable_test.go", "coverage_test.go", "scenario_test.go", "spec_159_test.go", "parsecheck_test.go", "pkg/kit/project.go", "pkg/kit/pack_test.go", "pkg/kit/portable_test.go", "pkg/replay/coverage_test.go", "pkg/scenario/scenario_test.go", "pkg/scenario/spec_159_test.go", "pkg/xs/parsecheck_test.go":
		return true
	default:
		return false
	}
}

func TestAbsolutePathNeedleReportsMachineRoots(t *testing.T) {
	for _, line := range []string{"x /" + "home/example", "x ~" + "/fixture", "x " + string([]byte{67, 58, 92}) + "Users\\example"} {
		if got := absolutePathNeedle([]byte(line)); got == "" {
			t.Fatalf("absolutePathNeedle(%q) returned empty", line)
		}
	}
	if absolutePathAllowlisted("pkg/scenario/layout_test.go") {
		t.Fatal("new ordinary test unexpectedly allowlisted")
	}
}
