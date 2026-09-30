package scenario

import (
	"aoe2kit/pkg/testfixtures"
	"os"
	"strings"
	"testing"
)

func TestScenarioStringsFrontTowersGolden(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{
			name: "CB Front Towers v247",
			path: testfixtures.Path(t, "save-analysis/CB_FRONT_TOWERS_V247.aoe2scenario"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := os.Stat(tt.path); err != nil {
				t.Skipf("golden scenario not present: %v", err)
			}
			scen, err := Open(tt.path)
			if err != nil {
				t.Fatal(err)
			}
			report := scen.Strings()
			for _, color := range []string{"RED", "GREEN", "YELLOW", "BLUE", "AQUA", "PURPLE", "ORANGE", "GREY"} {
				if report.MarkupSummary.ColorTags[color] == 0 {
					t.Fatalf("color tag %s count = 0", color)
				}
			}
			if !containsString(report.MarkupSummary.VariableRefs, "KillCountP1") {
				t.Fatalf("missing KillCountP1 variable ref in %#v", report.MarkupSummary.VariableRefs)
			}
			if !containsString(report.MarkupSummary.VariableRefs, "Variable 150") {
				t.Fatalf("missing Variable 150 ref in %#v", report.MarkupSummary.VariableRefs)
			}
			if !strings.Contains(report.Messages["ascii_instructions"], "Every 100 kills you will receive +1 Upgrade coin") {
				t.Fatalf("ascii_instructions missing upgrade coin FAQ text")
			}
		})
	}
}

func TestScenarioMarkupDoesNotCountVariableDeclarations(t *testing.T) {
	report := StringReport{
		Variables: []VariableEntry{
			{ID: 332, Name: "A2K HERO P1 H1 COINS"},
			{ID: 150, Name: "<Variable DECLARED ONLY>"},
		},
		EffectText: []EffectTextEntry{
			{Text: "Coins <Variable 332> and kills <KillCountP1>"},
		},
	}
	report.MarkupSummary = summarizeScenarioMarkup(report.allText())
	if !containsString(report.MarkupSummary.VariableRefs, "Variable 332") {
		t.Fatalf("missing numeric variable ref: %#v", report.MarkupSummary.VariableRefs)
	}
	if !containsString(report.MarkupSummary.VariableRefs, "KillCountP1") {
		t.Fatalf("missing killcount ref: %#v", report.MarkupSummary.VariableRefs)
	}
	if containsString(report.MarkupSummary.VariableRefs, "A2K HERO P1 H1 COINS") {
		t.Fatalf("plain variable name was misclassified as markup: %#v", report.MarkupSummary.VariableRefs)
	}
	if containsString(report.MarkupSummary.VariableRefs, "Variable DECLARED ONLY") {
		t.Fatalf("variable declaration text was misclassified as markup: %#v", report.MarkupSummary.VariableRefs)
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
