package kit

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRecipeTemplateSummariesFilter(t *testing.T) {
	report, err := RecipeTemplateSummaries("scenario")
	if err != nil {
		t.Fatalf("scenario summaries: %v", err)
	}
	if report.Count == 0 {
		t.Fatal("expected scenario recipe templates")
	}
	for _, template := range report.Templates {
		if template.Domain != "scenario" {
			t.Fatalf("template %q has domain %q", template.Name, template.Domain)
		}
	}
}

func TestRecipeTemplatePayloadsAreSingleDomain(t *testing.T) {
	for _, template := range RecipeTemplates() {
		payload, err := RecipeTemplatePayload(template)
		if err != nil {
			t.Fatalf("%s payload: %v", template.Name, err)
		}
		if payload == nil {
			t.Fatalf("%s has nil payload", template.Name)
		}
		if (template.ScenarioRecipe == nil) == (template.DatRecipe == nil) {
			t.Fatalf("%s should have exactly one recipe payload", template.Name)
		}
	}
}

func TestRecipeTemplateByName(t *testing.T) {
	template, ok := RecipeTemplateByName("scen.xs-carrier")
	if !ok {
		t.Fatal("scen.xs-carrier not found")
	}
	if template.ScenarioRecipe == nil || template.ScenarioRecipe.XS == nil {
		t.Fatalf("scen.xs-carrier payload = %+v", template)
	}
	short, ok := RecipeTemplateByName("caption-marker-unit")
	if !ok {
		t.Fatal("short recipe name caption-marker-unit not found")
	}
	if short.Name != "scen.caption-marker-unit" {
		t.Fatalf("short recipe resolved to %q, want scen.caption-marker-unit", short.Name)
	}
	victory, ok := RecipeTemplateByName("no-conquest-victory")
	if !ok {
		t.Fatal("short recipe name no-conquest-victory not found")
	}
	if victory.ScenarioRecipe == nil || victory.ScenarioRecipe.Victory == nil || victory.ScenarioRecipe.Victory.ConquestRequired == nil {
		t.Fatalf("no-conquest-victory payload = %+v", victory)
	}
	if got := *victory.ScenarioRecipe.Victory.ConquestRequired; got != 0 {
		t.Fatalf("no-conquest-victory conquest_required = %d, want 0", got)
	}
	closeout, ok := RecipeTemplateByName("timer-declare-victory")
	if !ok {
		t.Fatal("short recipe name timer-declare-victory not found")
	}
	if closeout.ScenarioRecipe == nil || len(closeout.ScenarioRecipe.Triggers) != 1 {
		t.Fatalf("timer-declare-victory payload = %+v", closeout)
	}
	trigger := closeout.ScenarioRecipe.Triggers[0]
	if len(trigger.Conditions) != 1 || trigger.Conditions[0].Timer == nil || *trigger.Conditions[0].Timer != 120 {
		t.Fatalf("timer-declare-victory conditions = %+v", trigger.Conditions)
	}
	if len(trigger.Effects) != 1 || trigger.Effects[0].Op != "declare_victory" {
		t.Fatalf("timer-declare-victory effects = %+v", trigger.Effects)
	}
}

func TestDATDesignerRecipeTemplatesExist(t *testing.T) {
	names := []string{
		"dat.ability-create-command",
		"dat.ability-disable-semantic",
		"dat.effect-resource-grant",
		"dat.effect-tech-cost-time-adjust",
		"dat.effect-unit-attribute-buff",
		"dat.sound-item-remove",
		"dat.unit-disable-all-civs",
		"dat.unit-enable-existing-civ",
		"dat.unit-train-button",
	}
	for _, name := range names {
		template, ok := RecipeTemplateByName(name)
		if !ok {
			t.Fatalf("%s not found", name)
		}
		if template.DatRecipe == nil {
			t.Fatalf("%s should be a DAT recipe", name)
		}
	}
}

func TestExportRecipeTemplatesWritesDomainRecipes(t *testing.T) {
	dir := t.TempDir()
	report, err := ExportRecipeTemplates(dir, "dat", false)
	if err != nil {
		t.Fatalf("ExportRecipeTemplates: %v", err)
	}
	if report.Domain != "dat" || report.Count == 0 {
		t.Fatalf("report = %+v", report)
	}
	path := filepath.Join(dir, "effect-resource-grant.json")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected exported recipe %s: %v", path, err)
	}
	if _, err := ExportRecipeTemplates(dir, "dat", false); err == nil {
		t.Fatal("expected overwrite protection error")
	}
	if _, err := ExportRecipeTemplates(dir, "dat", true); err != nil {
		t.Fatalf("ExportRecipeTemplates force: %v", err)
	}
}
