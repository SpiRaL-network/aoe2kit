package scenario

import (
	"aoe2kit/pkg/testfixtures"
	"os"
	"strings"
	"testing"
)

func TestScenarioAnalyzeFrontTowersGolden(t *testing.T) {
	path := testfixtures.Path(t, "save-analysis/CB_FRONT_TOWERS_V247.aoe2scenario")
	if _, err := os.Stat(path); err != nil {
		t.Skipf("golden scenario not present: %v", err)
	}
	scen, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	report := scen.Analyze()
	if report.DataSet.Status != "vanilla" {
		t.Fatalf("data set status = %q, want vanilla", report.DataSet.Status)
	}
	if report.Scale.MapWidth != 220 || report.Scale.MapHeight != 220 {
		t.Fatalf("map = %dx%d, want 220x220", report.Scale.MapWidth, report.Scale.MapHeight)
	}
	if report.Scale.TriggerCount != 2788 || report.Scale.VariableCount != 100 {
		t.Fatalf("scale triggers/vars = %d/%d, want 2788/100", report.Scale.TriggerCount, report.Scale.VariableCount)
	}
	for _, color := range []string{"RED", "GREEN", "YELLOW", "BLUE", "AQUA", "PURPLE", "ORANGE", "GREY"} {
		if report.DisplayTechniques.ColorMarkup.Counts[color] == 0 {
			t.Fatalf("missing color %s", color)
		}
	}
	if !report.DisplayTechniques.VariableSubstitutionHUD.Detected {
		t.Fatalf("variable substitution HUD not detected")
	}
	if !report.DisplayTechniques.CreateKillDisplayLoops.Detected {
		t.Fatalf("create/kill display loops not detected")
	}
	if len(report.Watermarks) == 0 || !watermarkMentions(report.Watermarks, "spiral") {
		t.Fatalf("SpiRaL watermark not detected: %#v", report.Watermarks)
	}
}

func watermarkMentions(watermarks []WatermarkSignal, term string) bool {
	for _, watermark := range watermarks {
		for _, value := range watermark.Terms {
			if strings.EqualFold(value, term) {
				return true
			}
		}
		for _, value := range watermark.Triggers {
			if strings.Contains(strings.ToLower(value), strings.ToLower(term)) {
				return true
			}
		}
	}
	return false
}
