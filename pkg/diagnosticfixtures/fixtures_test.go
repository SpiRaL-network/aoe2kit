package diagnosticfixtures

import (
	"os"
	"strings"
	"testing"

	"aoe2kit/pkg/xs"
)

func TestFixturesGenerateFreshAndLintClean(t *testing.T) {
	for name, build := range map[string]func(string, int) (*Report, error){
		"halo":       BuildHalo,
		"local-tech": BuildLocalTechnology,
	} {
		report, err := build(t.TempDir(), 1780000000)
		if err != nil {
			t.Fatalf("%s build: %v", name, err)
		}
		source, err := os.ReadFile(report.XSPath)
		if err != nil {
			t.Fatal(err)
		}
		if findings := xs.LintSource(report.XSPath, string(source)); len(findings) != 0 {
			t.Fatalf("%s XS lint findings: %+v", name, findings)
		}
		if name == "halo" && !strings.Contains(string(source), "A2K_FIRE = 939") {
			t.Fatal("halo fixture lost fire material")
		}
	}
}
