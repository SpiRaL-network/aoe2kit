package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"aoe2kit/pkg/datcodec"
	"aoe2kit/pkg/datfile"
	"aoe2kit/pkg/scenario"
	"aoe2kit/pkg/testfixtures"
)

func TestParseDatGraphicsOptions(t *testing.T) {
	opts, err := parseDatGraphicsOptions([]string{"--limit", "0", "--particle", "--name-contains", "Smoke", "--spans"}, "kit dat graphics")
	if err != nil {
		t.Fatalf("parseDatGraphicsOptions: %v", err)
	}
	if !opts.All || opts.Limit != 0 || !opts.ParticleOnly || opts.NameContains != "Smoke" || !opts.Spans {
		t.Fatalf("options = %+v", opts)
	}
}

func TestScenarioXSCarrierCLIWorkflow(t *testing.T) {
	input := testfixtures.Path(t, "save-analysis/diagnostics/Replay_Transparency_Diagnostic_v2b_flagged.aoe2scenario")
	if _, err := os.Stat(input); err != nil {
		t.Skipf("diagnostic fixture unavailable: %v", err)
	}
	dir := t.TempDir()
	source := "const int PROBE = 22;\nvoid BootProbe() { xsChatData(\"A2K\"); }\n"
	xsPath := filepath.Join(dir, "probe.xs")
	if err := os.WriteFile(xsPath, []byte(source), 0644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "embedded.aoe2scenario")
	runScenarioXSEmbed([]string{input, out, "--xs", xsPath, "--title", "probe.xs", "--replace-trigger", "0", "--json"})
	runScenarioXSCompare([]string{out, xsPath, "--json"})
	extracted := filepath.Join(dir, "extracted.xs")
	runScenarioXSExtract([]string{out, "--out", extracted, "--json"})
	got, err := os.ReadFile(extracted)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != source {
		t.Fatalf("extracted XS mismatch\ngot:\n%s\nwant:\n%s", got, source)
	}
}

func TestScenarioXSAttachDeployCLIWorkflow(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "blank.aoe2scenario")
	if _, err := scenario.WriteBlankScenarioFile(input, scenario.BlankOptions{PlayerCount: 2, HumanSlots: 1, DummyStarters: true}); err != nil {
		t.Fatalf("WriteBlankScenarioFile: %v", err)
	}
	source := "include \"shared.xs\";\nvoid BootProbe() { xsSetPlayerAttribute(1, 20, 1); }\n"
	xsPath := filepath.Join(dir, "Entry.xs")
	if err := os.WriteFile(xsPath, []byte(source), 0644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "attached.aoe2scenario")
	runScenarioXSAttach([]string{input, out, "--xs", xsPath, "--name", "Entry.xs", "--text"})
	runScenarioXSCompare([]string{out, xsPath, "--text"})
	extracted := filepath.Join(dir, "entry_extracted.xs")
	runScenarioXSExtract([]string{out, "--out", extracted, "--text"})
	extractedData, err := os.ReadFile(extracted)
	if err != nil {
		t.Fatal(err)
	}
	if string(extractedData) != source {
		t.Fatalf("extracted attachment mismatch\ngot:\n%s\nwant:\n%s", extractedData, source)
	}
	deployRoot := filepath.Join(dir, "mod")
	runScenarioXSDeploy([]string{out, deployRoot, "--force", "--text"})
	deployed := filepath.Join(deployRoot, "resources", "_common", "xs", "Entry.xs")
	got, err := os.ReadFile(deployed)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != source {
		t.Fatalf("deployed XS mismatch\ngot:\n%s\nwant:\n%s", got, source)
	}
}

func TestParseDatGraphicsOptionsRejectsNegativeLimit(t *testing.T) {
	if _, err := parseDatGraphicsOptions([]string{"--limit", "-1"}, "kit dat graphics"); err == nil {
		t.Fatal("negative limit accepted")
	}
}

func TestParseDatGraphicCreateOptions(t *testing.T) {
	recipe, err := parseDatGraphicCreateOptions([]string{
		"--from", "1711",
		"--name", "AoE2Kit Graphic",
		"--file-name", "aoe2kit_graphic",
		"--particle-effect-name", "aoe2kit_particle",
		"--slp", "971",
		"--frame-count", "8",
		"--coordinates", "1,2,3,4",
	})
	if err != nil {
		t.Fatalf("parse graphic-create: %v", err)
	}
	if recipe.From != 1711 || recipe.Name == nil || *recipe.Name != "AoE2Kit Graphic" || recipe.FileName == nil || *recipe.FileName != "aoe2kit_graphic" {
		t.Fatalf("basic graphic create fields = %+v", recipe)
	}
	if recipe.ParticleEffectName == nil || *recipe.ParticleEffectName != "aoe2kit_particle" || recipe.SLP == nil || *recipe.SLP != 971 || recipe.FrameCount == nil || *recipe.FrameCount != 8 {
		t.Fatalf("graphic create scalar fields = %+v", recipe)
	}
	if len(recipe.Coordinates) != 4 || recipe.Coordinates[0] != 1 || recipe.Coordinates[3] != 4 {
		t.Fatalf("graphic coordinates = %+v", recipe.Coordinates)
	}
}

func TestParseDatGraphicCreateOptionsRejectsMissingSource(t *testing.T) {
	if _, err := parseDatGraphicCreateOptions([]string{"--name", "Missing Source"}); err == nil {
		t.Fatal("graphic-create accepted missing --from")
	}
}

func TestDatEffectListOptionsFilterCommands(t *testing.T) {
	opts, err := parseDatEffectListOptions([]string{"--command-type", "102", "--operand", "244", "--limit", "5"}, "kit dat effects")
	if err != nil {
		t.Fatalf("parseDatEffectListOptions: %v", err)
	}
	if opts.CommandType == nil || *opts.CommandType != 102 || opts.Operand == nil || *opts.Operand != 244 || opts.Limit != 5 {
		t.Fatalf("options = %+v", opts)
	}
	effects := compactEffects([]datfile.Effect{
		{
			Index: 1,
			Name:  "match",
			Commands: []datfile.EffectCommand{
				{Index: 0, Type: 102, A: -1, B: -1, C: -1, D: 244},
				{Index: 1, Type: 102, A: -1, B: -1, C: -1, D: 100},
			},
		},
		{
			Index:    2,
			Name:     "miss",
			Commands: []datfile.EffectCommand{{Index: 0, Type: 8, A: 244, B: -1, C: -1, D: 1}},
		},
	}, opts)
	if len(effects) != 1 || effects[0].Index != 1 || len(effects[0].MatchingCommands) != 1 || effects[0].MatchingCommands[0].Index != 0 {
		t.Fatalf("filtered effects = %+v", effects)
	}
}

func TestParseEffectCommandRecipeSpecRawAndJSON(t *testing.T) {
	raw, err := parseEffectCommandRecipeSpec("5,1,-1,3,7.5", "--command")
	if err != nil {
		t.Fatalf("parse raw command: %v", err)
	}
	if raw.Type != 5 || raw.A != 1 || raw.B != -1 || raw.C != 3 || raw.D != 7.5 {
		t.Fatalf("raw command = %+v", raw)
	}

	fromJSON, err := parseEffectCommandRecipeSpec(`{"kind":"enable_unit","unit_id":83}`, "--command")
	if err != nil {
		t.Fatalf("parse JSON command: %v", err)
	}
	if fromJSON.Kind != "enable_unit" || fromJSON.UnitID == nil || *fromJSON.UnitID != 83 {
		t.Fatalf("JSON command = %+v", fromJSON)
	}
	if _, err := parseEffectCommandRecipeSpec("1,2,3", "--command"); err == nil {
		t.Fatal("bad raw command accepted")
	}
}

func TestParseDatEffectCreateOptions(t *testing.T) {
	recipe, err := parseDatEffectCreateOptions([]string{
		"--name", "Test Effect",
		"--from-effect", "91",
		"--remove-command", "0",
		"--command", "2,83,-1,-1,0",
		"--append-command", `{"kind":"disable_tech","tech_id":244}`,
	})
	if err != nil {
		t.Fatalf("parse effect-create: %v", err)
	}
	if recipe.Name != "Test Effect" || recipe.FromEffect == nil || *recipe.FromEffect != 91 {
		t.Fatalf("basic effect create fields = %+v", recipe)
	}
	if len(recipe.RemoveCommands) != 1 || recipe.RemoveCommands[0] != 0 || len(recipe.Commands) != 1 || recipe.Commands[0].Type != 2 || len(recipe.AppendCommands) != 1 || recipe.AppendCommands[0].Kind != "disable_tech" {
		t.Fatalf("effect create commands = %+v remove=%+v append=%+v", recipe.Commands, recipe.RemoveCommands, recipe.AppendCommands)
	}
}

func TestParseDatEffectPatchOptions(t *testing.T) {
	recipe, err := parseDatEffectPatchOptions(12, []string{
		"--name", "Patched Effect",
		"--clear-commands",
		"--command", "5,1,-1,3,7.5",
		"--append-command", `{"kind":"enable_unit","unit_id":83}`,
		"--remove-command", "0",
	})
	if err != nil {
		t.Fatalf("parse effect-patch: %v", err)
	}
	if recipe.ID != 12 || recipe.Name == nil || *recipe.Name != "Patched Effect" {
		t.Fatalf("basic effect patch fields = %+v", recipe)
	}
	if recipe.Commands == nil || len(*recipe.Commands) != 1 || (*recipe.Commands)[0].Type != 5 {
		t.Fatalf("set commands = %+v", recipe.Commands)
	}
	if len(recipe.AppendCommands) != 1 || recipe.AppendCommands[0].Kind != "enable_unit" || len(recipe.RemoveCommands) != 1 || recipe.RemoveCommands[0] != 0 {
		t.Fatalf("patch commands = %+v remove=%+v", recipe.AppendCommands, recipe.RemoveCommands)
	}
}

func TestParseDatEffectOptionsRejectRequiredAndNoop(t *testing.T) {
	if _, err := parseDatEffectCreateOptions([]string{"--command", "2,83,-1,-1,0"}); err == nil {
		t.Fatal("effect-create accepted missing --name")
	}
	if _, err := parseDatEffectPatchOptions(1, nil); err == nil {
		t.Fatal("effect-patch accepted no operations")
	}
	if _, err := parseDatEffectPatchOptions(-1, []string{"--name", "Bad"}); err == nil {
		t.Fatal("effect-patch accepted negative id")
	}
}

func TestParseDatAbilityCreateOptions(t *testing.T) {
	recipe, err := parseDatAbilityCreateOptions([]string{
		"--from-tech", "22",
		"--name", "Test Ability",
		"--effect-name", "Test Ability Effect",
		"--from-effect", "91",
		"--command", "2,83,-1,-1,0",
		"--append-command", `{"kind":"disable_tech","tech_id":244}`,
		"--required-techs", "-1,-1,-1,-1,-1,-1",
		"--required-tech-count", "0",
		"--civ", "-1",
		"--icon-id", "7",
		"--repeatable", "1",
	})
	if err != nil {
		t.Fatalf("parse ability-create: %v", err)
	}
	if recipe.FromTech != 22 || recipe.Name != "Test Ability" || recipe.EffectName != "Test Ability Effect" {
		t.Fatalf("basic recipe fields = %+v", recipe)
	}
	if recipe.FromEffect == nil || *recipe.FromEffect != 91 {
		t.Fatalf("from effect = %+v", recipe.FromEffect)
	}
	if len(recipe.Commands) != 1 || recipe.Commands[0].Type != 2 || len(recipe.AppendCommands) != 1 || recipe.AppendCommands[0].Kind != "disable_tech" {
		t.Fatalf("commands = %+v append=%+v", recipe.Commands, recipe.AppendCommands)
	}
	if len(recipe.RequiredTechs) != 6 || recipe.RequiredTechCount == nil || *recipe.RequiredTechCount != 0 || recipe.Civ == nil || *recipe.Civ != -1 || recipe.IconID == nil || *recipe.IconID != 7 || recipe.Repeatable == nil || *recipe.Repeatable != 1 {
		t.Fatalf("tech fields = %+v", recipe)
	}
}

func TestParseDatAbilityCreateOptionsRejectsMissingRequired(t *testing.T) {
	if _, err := parseDatAbilityCreateOptions([]string{"--from-tech", "22"}); err == nil {
		t.Fatal("ability-create accepted missing name")
	}
	if _, err := parseDatAbilityCreateOptions([]string{"--name", "Missing Source"}); err == nil {
		t.Fatal("ability-create accepted missing from-tech")
	}
}

func TestParseDatTechCreateOptions(t *testing.T) {
	recipe, err := parseDatTechCreateOptions([]string{
		"--from", "22",
		"--name", "Test Tech",
		"--required-techs", "-1,-1,-1,-1,-1,-1",
		"--required-tech-count", "0",
		"--civ", "-1",
		"--full-tech-mode", "0",
		"--dll-name", "123",
		"--dll-description", "124",
		"--effect-id", "-1",
		"--type", "2",
		"--icon-id", "7",
		"--dll-help", "125",
		"--dll-tech-tree", "126",
		"--repeatable", "1",
	})
	if err != nil {
		t.Fatalf("parse tech-create: %v", err)
	}
	if recipe.From != 22 || recipe.Name == nil || *recipe.Name != "Test Tech" || len(recipe.RequiredTechs) != 6 {
		t.Fatalf("basic tech create fields = %+v", recipe)
	}
	if recipe.RequiredTechCount == nil || *recipe.RequiredTechCount != 0 || recipe.Civ == nil || *recipe.Civ != -1 || recipe.FullTechMode == nil || *recipe.FullTechMode != 0 {
		t.Fatalf("requirement fields = %+v", recipe)
	}
	if recipe.LanguageDLLName == nil || *recipe.LanguageDLLName != 123 || recipe.LanguageDLLDescription == nil || *recipe.LanguageDLLDescription != 124 || recipe.EffectID == nil || *recipe.EffectID != -1 {
		t.Fatalf("dll/effect fields = %+v", recipe)
	}
	if recipe.Type == nil || *recipe.Type != 2 || recipe.IconID == nil || *recipe.IconID != 7 || recipe.LanguageDLLHelp == nil || *recipe.LanguageDLLHelp != 125 || recipe.LanguageDLLTechTree == nil || *recipe.LanguageDLLTechTree != 126 || recipe.Repeatable == nil || *recipe.Repeatable != 1 {
		t.Fatalf("ui fields = %+v", recipe)
	}
}

func TestParseDatTechPatchOptions(t *testing.T) {
	recipe, err := parseDatTechPatchOptions(44, []string{
		"--name", "Patched Tech",
		"--effect-id", "-1",
		"--icon-id", "9",
		"--repeatable", "0",
	})
	if err != nil {
		t.Fatalf("parse tech-patch: %v", err)
	}
	if recipe.ID != 44 || recipe.Name == nil || *recipe.Name != "Patched Tech" || recipe.EffectID == nil || *recipe.EffectID != -1 || recipe.IconID == nil || *recipe.IconID != 9 || recipe.Repeatable == nil || *recipe.Repeatable != 0 {
		t.Fatalf("tech patch = %+v", recipe)
	}
}

func TestParseDatTechOptionsRejectRequiredAndNoop(t *testing.T) {
	if _, err := parseDatTechCreateOptions([]string{"--name", "Missing Source"}); err == nil {
		t.Fatal("tech-create accepted missing --from")
	}
	if _, err := parseDatTechCreateOptions([]string{"--from", "22", "--required-techs", "-1,-1"}); err == nil {
		t.Fatal("tech-create accepted malformed required-techs")
	}
	if _, err := parseDatTechPatchOptions(1, nil); err == nil {
		t.Fatal("tech-patch accepted no operations")
	}
	if _, err := parseDatTechPatchOptions(-1, []string{"--name", "Bad"}); err == nil {
		t.Fatal("tech-patch accepted negative id")
	}
}

func TestParseDatPlayerColourCreateOptions(t *testing.T) {
	recipe, err := parseDatPlayerColourCreateOptions([]string{
		"--from", "1",
		"--colour-id", "16",
		"--base", "201",
		"--outline", "202",
		"--selection-1", "203",
		"--selection-2", "204",
		"--minimap-1", "205",
		"--minimap-2", "206",
		"--minimap-3", "207",
		"--stats", "208",
	})
	if err != nil {
		t.Fatalf("parse player-colour-create: %v", err)
	}
	if recipe.From != 1 || recipe.ColourID == nil || *recipe.ColourID != 16 || recipe.Base == nil || *recipe.Base != 201 || recipe.UnitOutlineColour == nil || *recipe.UnitOutlineColour != 202 {
		t.Fatalf("basic colour create fields = %+v", recipe)
	}
	if recipe.SelectionColour1 == nil || *recipe.SelectionColour1 != 203 || recipe.SelectionColour2 == nil || *recipe.SelectionColour2 != 204 {
		t.Fatalf("selection fields = %+v", recipe)
	}
	if recipe.MinimapColour1 == nil || *recipe.MinimapColour1 != 205 || recipe.MinimapColour2 == nil || *recipe.MinimapColour2 != 206 || recipe.MinimapColour3 == nil || *recipe.MinimapColour3 != 207 {
		t.Fatalf("minimap fields = %+v", recipe)
	}
	if recipe.StatisticsTextColor == nil || *recipe.StatisticsTextColor != 208 {
		t.Fatalf("stats colour = %+v", recipe)
	}
}

func TestParseDatPlayerColourOptionsRejectRequiredAndNoop(t *testing.T) {
	if _, err := parseDatPlayerColourCreateOptions([]string{"--base", "201"}); err == nil {
		t.Fatal("player-colour-create accepted missing --from")
	}
	if _, err := parseDatPlayerColourPatchOptions(1, nil); err == nil {
		t.Fatal("player-colour-patch accepted no operations")
	}
	if _, err := parseDatPlayerColourPatchOptions(-1, []string{"--base", "201"}); err == nil {
		t.Fatal("player-colour-patch accepted negative id")
	}
}

func TestParseDatPlayerColourPatchOptions(t *testing.T) {
	recipe, err := parseDatPlayerColourPatchOptions(2, []string{
		"--color-id", "22",
		"--base", "301",
		"--unit-outline-color", "302",
		"--statistics-text-color", "308",
	})
	if err != nil {
		t.Fatalf("parse player-colour-patch: %v", err)
	}
	if recipe.ID != 2 || recipe.ColourID == nil || *recipe.ColourID != 22 || recipe.Base == nil || *recipe.Base != 301 || recipe.UnitOutlineColour == nil || *recipe.UnitOutlineColour != 302 || recipe.StatisticsTextColor == nil || *recipe.StatisticsTextColor != 308 {
		t.Fatalf("player-colour patch = %+v", recipe)
	}
}

func TestParseDatSoundCreateOptions(t *testing.T) {
	recipe, err := parseDatSoundCreateOptions([]string{
		"--from", "12",
		"--sound-id", "901",
		"--play-delay", "2",
		"--cache-time", "30",
		"--total-probability", "100",
		"--item", "aoe2kit_test.wav,78,90,1,2",
	})
	if err != nil {
		t.Fatalf("parse sound-create: %v", err)
	}
	if recipe.From != 12 || recipe.SoundID == nil || *recipe.SoundID != 901 || recipe.PlayDelay == nil || *recipe.PlayDelay != 2 || recipe.CacheTime == nil || *recipe.CacheTime != 30 || recipe.TotalProbability == nil || *recipe.TotalProbability != 100 {
		t.Fatalf("sound create fields = %+v", recipe)
	}
	if recipe.Items == nil || len(*recipe.Items) != 1 {
		t.Fatalf("sound create items = %+v", recipe.Items)
	}
	item := (*recipe.Items)[0]
	if item.FileName != "aoe2kit_test.wav" || item.ResourceID != 78 || item.Probability != 90 || item.Civ != 1 || item.IconSet != 2 {
		t.Fatalf("sound create item = %+v", item)
	}
}

func TestParseDatSoundPatchOptions(t *testing.T) {
	recipe, err := parseDatSoundPatchOptions(22, []string{
		"--play-delay", "3",
		"--probability", "44",
		"--item", "1,file=patched.wav,resource=79,prob=80,civ=2,icon=3",
		"--remove-item", "0",
	})
	if err != nil {
		t.Fatalf("parse sound-patch: %v", err)
	}
	if recipe.ID != 22 || recipe.PlayDelay == nil || *recipe.PlayDelay != 3 || recipe.TotalProbability == nil || *recipe.TotalProbability != 44 {
		t.Fatalf("sound patch fields = %+v", recipe)
	}
	if len(recipe.Items) != 1 || recipe.Items[0].Index != 1 || recipe.Items[0].FileName == nil || *recipe.Items[0].FileName != "patched.wav" || recipe.Items[0].ResourceID == nil || *recipe.Items[0].ResourceID != 79 || recipe.Items[0].Probability == nil || *recipe.Items[0].Probability != 80 || recipe.Items[0].Civ == nil || *recipe.Items[0].Civ != 2 || recipe.Items[0].IconSet == nil || *recipe.Items[0].IconSet != 3 {
		t.Fatalf("sound patch items = %+v", recipe.Items)
	}
	if len(recipe.RemoveItems) != 1 || recipe.RemoveItems[0] != 0 {
		t.Fatalf("sound remove items = %+v", recipe.RemoveItems)
	}
}

func TestParseDatSoundOptionsRejectRequiredAndNoop(t *testing.T) {
	if _, err := parseDatSoundCreateOptions([]string{"--play-delay", "1"}); err == nil {
		t.Fatal("sound-create accepted missing --from")
	}
	if _, err := parseDatSoundPatchOptions(1, nil); err == nil {
		t.Fatal("sound-patch accepted no operations")
	}
	if _, err := parseDatSoundPatchOptions(-1, []string{"--play-delay", "1"}); err == nil {
		t.Fatal("sound-patch accepted negative id")
	}
	if _, err := parseDatSoundCreateOptions([]string{"--from", "1", "--item", "missing-fields"}); err == nil {
		t.Fatal("sound-create accepted malformed item")
	}
}

func TestParseDatUnitCreateOptions(t *testing.T) {
	recipe, err := parseDatUnitCreateOptions([]string{
		"--from-civ", "0",
		"--from-unit", "83",
		"--civ", "0,1",
		"--hit-points", "77",
		"--enabled", "1",
		"--attribute", "1,amount=3",
	})
	if err != nil {
		t.Fatalf("parse unit-create: %v", err)
	}
	if recipe.FromCivID != 0 || recipe.FromUnitID != 83 || len(recipe.CivIDs) != 2 || recipe.CivIDs[0] != 0 || recipe.CivIDs[1] != 1 {
		t.Fatalf("basic unit create fields = %+v", recipe)
	}
	if recipe.HitPoints == nil || *recipe.HitPoints != 77 || recipe.Enabled == nil || *recipe.Enabled != 1 {
		t.Fatalf("scalar unit create fields = %+v", recipe)
	}
	if len(recipe.Attributes) != 1 || recipe.Attributes[0].Index != 1 || recipe.Attributes[0].Amount == nil || *recipe.Attributes[0].Amount != 3 {
		t.Fatalf("attribute patches = %+v", recipe.Attributes)
	}
}

func TestParseDatUnitDeleteRequest(t *testing.T) {
	request, err := parseDatUnitDeleteRequest(83, []string{"--all-civs"})
	if err != nil {
		t.Fatalf("parse unit-delete: %v", err)
	}
	if request.Section != "unit" || request.ID != 83 || !request.AllCivs {
		t.Fatalf("all-civs request = %+v", request)
	}
	request, err = parseDatUnitDeleteRequest(83, []string{"--civ", "2"})
	if err != nil {
		t.Fatalf("parse unit-delete civ: %v", err)
	}
	if request.CivID == nil || *request.CivID != 2 || request.AllCivs {
		t.Fatalf("civ request = %+v", request)
	}
}

func TestParseDatUnitCreateDeleteRejectsBadScope(t *testing.T) {
	if _, err := parseDatUnitCreateOptions([]string{"--from-unit", "83", "--civ", "0"}); err == nil {
		t.Fatal("unit-create accepted missing --from-civ")
	}
	if _, err := parseDatUnitCreateOptions([]string{"--from-civ", "0", "--civ", "0"}); err == nil {
		t.Fatal("unit-create accepted missing --from-unit")
	}
	if _, err := parseDatUnitCreateOptions([]string{"--from-civ", "0", "--from-unit", "83"}); err == nil {
		t.Fatal("unit-create accepted missing civ scope")
	}
	if _, err := parseDatUnitCreateOptions([]string{"--from-civ", "0", "--from-unit", "83", "--civ", "0", "--all-civs"}); err == nil {
		t.Fatal("unit-create accepted conflicting civ scope")
	}
	if _, err := parseDatUnitDeleteRequest(83, nil); err == nil {
		t.Fatal("unit-delete accepted missing civ scope")
	}
	if _, err := parseDatUnitDeleteRequest(83, []string{"--civ", "0", "--all-civs"}); err == nil {
		t.Fatal("unit-delete accepted conflicting civ scope")
	}
}

func TestParseDatCivPatchOptions(t *testing.T) {
	recipe, err := parseDatCivPatchOptions(2, []string{
		"--name", "Test Civ",
		"--tech-tree-id", "3",
		"--team-bonus-id", "4",
		"--icon-set", "5",
		"--resource", "0,123.5",
	})
	if err != nil {
		t.Fatalf("parse civ-patch: %v", err)
	}
	if recipe.ID != 2 || recipe.Name == nil || *recipe.Name != "Test Civ" || recipe.TechTreeID == nil || *recipe.TechTreeID != 3 || recipe.TeamBonusID == nil || *recipe.TeamBonusID != 4 || recipe.IconSet == nil || *recipe.IconSet != 5 {
		t.Fatalf("civ patch fields = %+v", recipe)
	}
	if len(recipe.Resources) != 1 || recipe.Resources[0].Index != 0 || recipe.Resources[0].Value != 123.5 {
		t.Fatalf("civ resources = %+v", recipe.Resources)
	}
}

func TestParseDatTerrainPatchOptions(t *testing.T) {
	recipe, err := parseDatTerrainPatchOptions(3, []string{
		"--name", "Grass Test",
		"--name-2", "Grass Test 2",
		"--overlay-mask-name", "MASK_TEST",
	})
	if err != nil {
		t.Fatalf("parse terrain-patch: %v", err)
	}
	if recipe.ID != 3 || recipe.Name == nil || *recipe.Name != "Grass Test" || recipe.Name2 == nil || *recipe.Name2 != "Grass Test 2" || recipe.OverlayMaskName == nil || *recipe.OverlayMaskName != "MASK_TEST" {
		t.Fatalf("terrain patch fields = %+v", recipe)
	}
}

func TestParseDatTerrainRestrictionPatchOptions(t *testing.T) {
	recipe, err := parseDatTerrainRestrictionPatchOptions(1, []string{"--terrain", "2,1.5"})
	if err != nil {
		t.Fatalf("parse terrain-restriction-patch: %v", err)
	}
	if recipe.ID != 1 || len(recipe.Rows) != 1 || recipe.Rows[0].TerrainID != 2 || recipe.Rows[0].Passability != 1.5 {
		t.Fatalf("terrain restriction patch = %+v", recipe)
	}
}

func TestParseDatListOptionsAcceptsExplicitJSON(t *testing.T) {
	limit, all, err := parseDatListOptions([]string{"--limit", "3", "--json"}, "kit dat terrains")
	if err != nil {
		t.Fatalf("parseDatListOptions: %v", err)
	}
	if limit != 3 || all {
		t.Fatalf("parseDatListOptions limit/all = %d/%v", limit, all)
	}

	limit, all, err = parseDatListOptions([]string{"--all", "--json"}, "kit dat terrains")
	if err != nil {
		t.Fatalf("parseDatListOptions all: %v", err)
	}
	if limit != 200 || !all {
		t.Fatalf("parseDatListOptions all limit/all = %d/%v", limit, all)
	}
}

func TestParseDatAvailabilitySetOptions(t *testing.T) {
	recipe, err := parseDatAvailabilitySetOptions(83, []string{"--civ", "0,1", "--enabled", "0"})
	if err != nil {
		t.Fatalf("parse availability-set: %v", err)
	}
	if recipe.UnitID != 83 || recipe.Enabled || len(recipe.CivIDs) != 2 || recipe.CivIDs[0] != 0 || recipe.CivIDs[1] != 1 {
		t.Fatalf("availability recipe = %+v", recipe)
	}
	recipe, err = parseDatAvailabilitySetOptions(83, []string{"--civ", "0", "--enabled", "true"})
	if err != nil {
		t.Fatalf("parse availability-set one civ: %v", err)
	}
	if recipe.CivID == nil || *recipe.CivID != 0 || len(recipe.CivIDs) != 0 || !recipe.Enabled {
		t.Fatalf("single civ availability recipe = %+v", recipe)
	}
}

func TestParseDatFixedTableOptionsRejectsBadInput(t *testing.T) {
	if _, err := parseDatCivPatchOptions(1, nil); err == nil {
		t.Fatal("civ-patch accepted no operations")
	}
	if _, err := parseDatTerrainPatchOptions(1, nil); err == nil {
		t.Fatal("terrain-patch accepted no operations")
	}
	if _, err := parseDatTerrainRestrictionPatchOptions(1, nil); err == nil {
		t.Fatal("terrain-restriction-patch accepted no rows")
	}
	if _, err := parseDatAvailabilitySetOptions(83, []string{"--civ", "0"}); err == nil {
		t.Fatal("availability-set accepted missing --enabled")
	}
	if _, err := parseDatAvailabilitySetOptions(83, []string{"--all-civs", "--civ", "0", "--enabled", "1"}); err == nil {
		t.Fatal("availability-set accepted conflicting scope")
	}
}

func TestParseDatTechTreeConnectionCreateOptions(t *testing.T) {
	recipe, request, err := parseDatTechTreeConnectionCreateOptions("building", 2, []string{
		"--id", "1200",
		"--status", "2",
		"--buildings", "12,103",
		"--units", "83",
		"--techs", "101",
		"--unit-research", "-1,-1,-1,-1,-1,-1,-1,-1,-1,-1",
		"--mode", "0,0,0,0,0,0,0,0,0,0",
		"--location-in-age", "1",
		"--units-techs-total", "1,2,3,4,5",
		"--units-techs-first", "5,4,3,2,1",
		"--line-mode", "7",
		"--enabling-research", "8",
	})
	if err != nil {
		t.Fatalf("parse building connection create: %v", err)
	}
	if len(recipe.CreateBuildingConnections) != 1 {
		t.Fatalf("create recipe = %+v", recipe)
	}
	create := recipe.CreateBuildingConnections[0]
	if create.From != 2 || create.ID == nil || *create.ID != 1200 || create.Status == nil || *create.Status != 2 || create.LineMode == nil || *create.LineMode != 7 || create.EnablingResearch == nil || *create.EnablingResearch != 8 {
		t.Fatalf("create fields = %+v request=%+v", create, request)
	}
	if create.Buildings == nil || len(*create.Buildings) != 2 || (*create.Buildings)[1] != 103 {
		t.Fatalf("create buildings = %+v", create.Buildings)
	}
	if create.UnitResearch == nil || len(*create.UnitResearch) != 10 {
		t.Fatalf("unit research = %+v", create.UnitResearch)
	}
	if create.UnitsTechsTotal == nil || len(*create.UnitsTechsTotal) != 5 || (*create.UnitsTechsTotal)[4] != 5 {
		t.Fatalf("units techs total = %+v", create.UnitsTechsTotal)
	}
}

func TestParseDatTechTreeConnectionPatchOptions(t *testing.T) {
	recipe, request, err := parseDatTechTreeConnectionPatchOptions("unit", 3, []string{
		"--id", "83",
		"--status", "2",
		"--upper-building", "12",
		"--units", "83,422",
		"--location-in-age", "1",
		"--required-research", "101",
		"--enabling-research", "102",
	})
	if err != nil {
		t.Fatalf("parse unit connection patch: %v", err)
	}
	if len(recipe.UnitConnections) != 1 {
		t.Fatalf("patch recipe = %+v", recipe)
	}
	patch := recipe.UnitConnections[0]
	if patch.Index != 3 || patch.ID == nil || *patch.ID != 83 || patch.UpperBuilding == nil || *patch.UpperBuilding != 12 || patch.RequiredResearch == nil || *patch.RequiredResearch != 101 || patch.EnablingResearch == nil || *patch.EnablingResearch != 102 {
		t.Fatalf("unit connection patch = %+v request=%+v", patch, request)
	}
	if patch.Units == nil || len(*patch.Units) != 2 || (*patch.Units)[1] != 422 {
		t.Fatalf("unit connection units = %+v", patch.Units)
	}

	recipe, _, err = parseDatTechTreeConnectionPatchOptions("research", 4, []string{"--buildings", "12", "--techs", "101,102"})
	if err != nil {
		t.Fatalf("parse research connection patch: %v", err)
	}
	if len(recipe.ResearchConnections) != 1 || recipe.ResearchConnections[0].Index != 4 || recipe.ResearchConnections[0].Buildings == nil || recipe.ResearchConnections[0].Techs == nil {
		t.Fatalf("research connection patch = %+v", recipe)
	}
}

func TestDatTechTreeConnectionDeleteRecipe(t *testing.T) {
	recipe, err := datTechTreeConnectionDeleteRecipe("unit", 5)
	if err != nil {
		t.Fatalf("delete unit connection recipe: %v", err)
	}
	if len(recipe.DeleteUnitConnections) != 1 || recipe.DeleteUnitConnections[0] != 5 {
		t.Fatalf("delete unit recipe = %+v", recipe)
	}
	recipe, err = datTechTreeConnectionDeleteRecipe("building", 6)
	if err != nil {
		t.Fatalf("delete building connection recipe: %v", err)
	}
	if len(recipe.DeleteBuildingConnections) != 1 || recipe.DeleteBuildingConnections[0] != 6 {
		t.Fatalf("delete building recipe = %+v", recipe)
	}
	recipe, err = datTechTreeConnectionDeleteRecipe("research", 7)
	if err != nil {
		t.Fatalf("delete research connection recipe: %v", err)
	}
	if len(recipe.DeleteResearchConnections) != 1 || recipe.DeleteResearchConnections[0] != 7 {
		t.Fatalf("delete research recipe = %+v", recipe)
	}
}

func TestParseDatTechTreeConnectionRejectsBadInput(t *testing.T) {
	if _, err := normalizeTechTreeConnectionFamily("age"); err == nil {
		t.Fatal("accepted unknown tech-tree connection family")
	}
	if _, _, err := parseDatTechTreeConnectionPatchOptions("unit", 0, nil); err == nil {
		t.Fatal("accepted no-op unit connection patch")
	}
	if _, err := datTechTreeConnectionDeleteRecipe("unit", -1); err == nil {
		t.Fatal("accepted negative connection delete")
	}
	if _, _, err := parseDatTechTreeConnectionPatchOptions("building", 0, []string{"--units-techs-total", "1,2,3,4,300"}); err == nil {
		t.Fatal("accepted uint8 list overflow")
	}
}

func TestDatUnitChildDeleteSection(t *testing.T) {
	cases := map[string]string{
		"damage-graphic": "unit_damage_graphic",
		"attack":         "unit_attack",
		"armor":          "unit_armour",
		"armour":         "unit_armour",
		"train-location": "unit_train_location",
		"drop-site":      "unit_drop_site",
		"task":           "unit_task",
	}
	for raw, want := range cases {
		got, err := datUnitChildDeleteSection(raw)
		if err != nil {
			t.Fatalf("datUnitChildDeleteSection(%q): %v", raw, err)
		}
		if got != want {
			t.Fatalf("datUnitChildDeleteSection(%q)=%q want %q", raw, got, want)
		}
	}
	if _, err := datUnitChildDeleteSection("unknown"); err == nil {
		t.Fatal("accepted unknown unit child delete section")
	}
}

func TestParseDirectDatChildDeleteTargets(t *testing.T) {
	targets := map[string]string{
		"effect-command":      "10:2",
		"sound-item":          "20:1",
		"graphic-delta":       "30:4",
		"graphic-angle-sound": "31:5",
		"unit-header-task":    "40:6",
	}
	for section, target := range targets {
		request, err := parseDatDeleteTarget(section, target)
		if err != nil {
			t.Fatalf("parseDatDeleteTarget(%s,%s): %v", section, target, err)
		}
		if request.ID < 0 {
			t.Fatalf("request id should be non-negative: %+v", request)
		}
	}
	request, err := parseDatDeleteTarget("unit_attack", "1:83:0")
	if err != nil {
		t.Fatalf("unit child delete target: %v", err)
	}
	if request.CivID == nil || *request.CivID != 1 || request.UnitID == nil || *request.UnitID != 83 || request.RowIndex == nil || *request.RowIndex != 0 {
		t.Fatalf("unit child delete request = %+v", request)
	}
}

func TestAbilityPatchNoopRecipeEmpty(t *testing.T) {
	recipe := datcodec.Recipe{
		Techs:   []datcodec.TechPatchRecipe{},
		Effects: []datcodec.EffectPatchRecipe{},
	}
	if !recipe.Empty() {
		t.Fatalf("empty ability patch recipe should be empty: %+v", recipe)
	}
}

func TestParseDatCommandMatrixOptionsFilters(t *testing.T) {
	opts, err := parseDatCommandMatrixOptions([]string{
		"--command-type", "102",
		"--reference-kind", "tech",
		"--reference-field", "amount",
		"--typed-only",
		"--candidate-only",
		"--limit", "0",
		"--text",
	})
	if err != nil {
		t.Fatalf("parseDatCommandMatrixOptions: %v", err)
	}
	if opts.CommandType == nil || *opts.CommandType != 102 ||
		opts.ReferenceKind != "tech" || opts.ReferenceField != "amount" ||
		!opts.TypedOnly || !opts.CandidateOnly || opts.UnknownOnly || opts.ExampleLimit != 0 || !opts.Text {
		t.Fatalf("options = %+v", opts)
	}
	if _, err := parseDatCommandMatrixOptions([]string{"--reference-kind", "sound"}); err == nil {
		t.Fatal("bad reference kind accepted")
	}
	if _, err := parseDatCommandMatrixOptions([]string{"--command-type", "256"}); err == nil {
		t.Fatal("out-of-range command type accepted")
	}
	aliasOpts, err := parseDatCommandMatrixOptions([]string{"--typed", "--unknown", "--candidate"})
	if err != nil {
		t.Fatalf("parseDatCommandMatrixOptions aliases: %v", err)
	}
	if !aliasOpts.TypedOnly || !aliasOpts.UnknownOnly || !aliasOpts.CandidateOnly {
		t.Fatalf("alias options = %+v", aliasOpts)
	}
}

func TestParseScenarioSmokeOptions(t *testing.T) {
	opts, err := parseScenarioSmokeOptions([]string{
		"--x", "20",
		"--y", "30",
		"--player", "2",
		"--unit", "74",
		"--terrain", "5",
		"--elevation", "1",
		"--layer", "-1",
	})
	if err != nil {
		t.Fatalf("parseScenarioSmokeOptions: %v", err)
	}
	if opts.X != 20 || !opts.XSet ||
		opts.Y != 30 || !opts.YSet ||
		opts.Player != 2 || !opts.PlayerSet ||
		opts.UnitConst != 74 || !opts.UnitConstSet ||
		opts.TerrainID != 5 || !opts.TerrainIDSet ||
		opts.Elevation != 1 || !opts.ElevationSet ||
		opts.Layer != -1 || !opts.LayerSet {
		t.Fatalf("options = %+v", opts)
	}
	if _, err := parseScenarioSmokeOptions([]string{"--bad", "1"}); err == nil {
		t.Fatal("unknown smoke option accepted")
	}
}

func TestParseScenarioDeleteAreaOptions(t *testing.T) {
	request, err := parseScenarioDeletePlanRequest("units-area", "1,2,3.5,4.25")
	if err != nil {
		t.Fatalf("parseScenarioDeletePlanRequest: %v", err)
	}
	if request.Kind != "units-area" ||
		request.AreaX1 == nil || *request.AreaX1 != 1 ||
		request.AreaY1 == nil || *request.AreaY1 != 2 ||
		request.AreaX2 == nil || *request.AreaX2 != 3.5 ||
		request.AreaY2 == nil || *request.AreaY2 != 4.25 {
		t.Fatalf("area request = %+v", request)
	}
	text, err := parseScenarioDeleteOptions(&request, []string{"--player", "2", "--unit", "83", "--text"}, true)
	if err != nil {
		t.Fatalf("parseScenarioDeleteOptions: %v", err)
	}
	if !text || request.Player == nil || *request.Player != 2 || request.UnitConst == nil || *request.UnitConst != 83 {
		t.Fatalf("area options text=%v request=%+v", text, request)
	}
}

func TestParseScenarioDeleteEffectAddress(t *testing.T) {
	request, err := parseScenarioDeletePlanRequest("effect", "7:3")
	if err != nil {
		t.Fatalf("parseScenarioDeletePlanRequest: %v", err)
	}
	if request.Kind != "effect" || request.ID != 3 ||
		request.TriggerID == nil || *request.TriggerID != 7 ||
		request.ChildIndex == nil || *request.ChildIndex != 3 {
		t.Fatalf("effect request = %+v", request)
	}
	if _, err := parseScenarioDeletePlanRequest("condition", "7"); err == nil {
		t.Fatal("condition target without child index accepted")
	}
	if _, err := parseScenarioDeletePlanRequest("effect", "-1:0"); err == nil {
		t.Fatal("negative trigger index accepted")
	}

	effectType, err := parseScenarioDeletePlanRequest("effect-type", "display_instructions")
	if err != nil {
		t.Fatalf("parseScenarioDeletePlanRequest effect-type: %v", err)
	}
	if effectType.Kind != "effect_type" || effectType.ID != 20 || effectType.TargetName != "display_instructions" {
		t.Fatalf("effect-type request = %+v", effectType)
	}
	if _, err := parseScenarioDeleteOptions(&effectType, []string{"--trigger", "4", "--text"}, true); err != nil {
		t.Fatalf("effect-type options rejected trigger/text: %v", err)
	}
	if effectType.TriggerID == nil || *effectType.TriggerID != 4 {
		t.Fatalf("effect-type trigger option = %+v", effectType)
	}

	conditionType, err := parseScenarioDeletePlanRequest("condition-type", "10")
	if err != nil {
		t.Fatalf("parseScenarioDeletePlanRequest condition-type: %v", err)
	}
	if conditionType.Kind != "condition_type" || conditionType.ID != 10 || conditionType.TargetName != "timer" {
		t.Fatalf("condition-type request = %+v", conditionType)
	}
	if _, err := parseScenarioDeleteOptions(&conditionType, []string{"--trigger-prefix", "A2K"}, true); err != nil {
		t.Fatalf("condition-type options rejected trigger-prefix: %v", err)
	}
	if conditionType.TargetPrefix != "A2K" {
		t.Fatalf("condition-type trigger-prefix option = %+v", conditionType)
	}
	if _, err := parseScenarioDeleteOptions(&conditionType, []string{"--trigger", "1", "--trigger-prefix", "A2K"}, true); err == nil {
		t.Fatal("condition-type accepted trigger and trigger-prefix together")
	}
	if _, err := parseScenarioDeleteOptions(&request, []string{"--trigger", "1"}, true); err == nil {
		t.Fatal("single child-row delete accepted --trigger scope")
	}
	if _, err := parseScenarioDeletePlanRequest("effect-type", "does_not_exist"); err == nil {
		t.Fatal("unknown effect-type name accepted")
	}
	if _, err := parseScenarioDeletePlanRequest("condition-type", "does_not_exist"); err == nil {
		t.Fatal("unknown condition-type name accepted")
	}

	effectTextPrefix, err := parseScenarioDeletePlanRequest("effect-text-prefix", "SDSDBG ")
	if err != nil {
		t.Fatalf("parseScenarioDeletePlanRequest effect-text-prefix: %v", err)
	}
	if effectTextPrefix.Kind != "effect_text_prefix" || effectTextPrefix.TargetText != "SDSDBG " {
		t.Fatalf("effect-text-prefix request = %+v", effectTextPrefix)
	}
	if _, err := parseScenarioDeleteOptions(&effectTextPrefix, []string{"--trigger-prefix", "Debug"}, true); err != nil {
		t.Fatalf("effect-text-prefix options rejected trigger-prefix: %v", err)
	}
	if effectTextPrefix.TargetPrefix != "Debug" {
		t.Fatalf("effect-text-prefix trigger-prefix option = %+v", effectTextPrefix)
	}
	if _, err := parseScenarioDeletePlanRequest("effect-text-prefix", " "); err == nil {
		t.Fatal("effect-text-prefix accepted empty prefix")
	}
	effectTextContains, err := parseScenarioDeletePlanRequest("effect-text-contains", "DBG_MARK")
	if err != nil {
		t.Fatalf("parseScenarioDeletePlanRequest effect-text-contains: %v", err)
	}
	if effectTextContains.Kind != "effect_text_contains" || effectTextContains.TargetText != "DBG_MARK" {
		t.Fatalf("effect-text-contains request = %+v", effectTextContains)
	}
	if _, err := parseScenarioDeleteOptions(&effectTextContains, []string{"--trigger", "2"}, true); err != nil {
		t.Fatalf("effect-text-contains options rejected trigger: %v", err)
	}
	if effectTextContains.TriggerID == nil || *effectTextContains.TriggerID != 2 {
		t.Fatalf("effect-text-contains trigger option = %+v", effectTextContains)
	}
	if _, err := parseScenarioDeletePlanRequest("effect-text-contains", " "); err == nil {
		t.Fatal("effect-text-contains accepted empty text")
	}
}

func TestParseScenarioDeleteCaptionSelector(t *testing.T) {
	systemRequest, err := parseScenarioDeletePlanRequest("system-prefix", "A2K System:")
	if err != nil {
		t.Fatalf("parseScenarioDeletePlanRequest system-prefix: %v", err)
	}
	if systemRequest.Kind != "system_prefix" || systemRequest.TargetPrefix != "A2K System:" {
		t.Fatalf("system-prefix request = %+v", systemRequest)
	}
	if _, err := parseScenarioDeleteOptions(&systemRequest, []string{"--player", "1"}, true); err == nil {
		t.Fatal("system-prefix scenario delete accepted --player")
	}
	if _, err := parseScenarioDeletePlanRequest("system-prefix", " "); err == nil {
		t.Fatal("system-prefix scenario delete accepted empty prefix")
	}

	request, err := parseScenarioDeletePlanRequest("unit-caption", "DELETE ME")
	if err != nil {
		t.Fatalf("parseScenarioDeletePlanRequest unit-caption: %v", err)
	}
	if request.Kind != "unit_caption" || request.TargetCaption != "DELETE ME" {
		t.Fatalf("unit-caption request = %+v", request)
	}
	if _, err := parseScenarioDeleteOptions(&request, []string{"--player", "1"}, true); err != nil {
		t.Fatalf("unit-caption scenario delete rejected --player: %v", err)
	}
	if request.Player == nil || *request.Player != 1 {
		t.Fatalf("unit-caption player option = %+v", request)
	}
	if _, err := parseScenarioDeleteOptions(&request, []string{"--unit", "83"}, true); err == nil {
		t.Fatal("unit-caption scenario delete accepted --unit")
	}
	captionPrefixRequest, err := parseScenarioDeletePlanRequest("unit-caption-prefix", "A2K Cap:")
	if err != nil {
		t.Fatalf("parseScenarioDeletePlanRequest unit-caption-prefix: %v", err)
	}
	if captionPrefixRequest.Kind != "unit_caption_prefix" || captionPrefixRequest.TargetPrefix != "A2K Cap:" {
		t.Fatalf("unit-caption-prefix request = %+v", captionPrefixRequest)
	}
	if _, err := parseScenarioDeleteOptions(&captionPrefixRequest, []string{"--player", "1"}, true); err != nil {
		t.Fatalf("unit-caption-prefix scenario delete rejected --player: %v", err)
	}
	if captionPrefixRequest.Player == nil || *captionPrefixRequest.Player != 1 {
		t.Fatalf("unit-caption-prefix player option = %+v", captionPrefixRequest)
	}
	if _, err := parseScenarioDeleteOptions(&captionPrefixRequest, []string{"--unit", "83"}, true); err == nil {
		t.Fatal("unit-caption-prefix scenario delete accepted --unit")
	}
	if _, err := parseScenarioDeletePlanRequest("unit-caption-prefix", " "); err == nil {
		t.Fatal("unit-caption-prefix scenario delete accepted empty prefix")
	}
	captionContainsRequest, err := parseScenarioDeletePlanRequest("unit-caption-contains", "CapMark")
	if err != nil {
		t.Fatalf("parseScenarioDeletePlanRequest unit-caption-contains: %v", err)
	}
	if captionContainsRequest.Kind != "unit_caption_contains" || captionContainsRequest.TargetText != "CapMark" {
		t.Fatalf("unit-caption-contains request = %+v", captionContainsRequest)
	}
	if _, err := parseScenarioDeleteOptions(&captionContainsRequest, []string{"--player", "2"}, true); err != nil {
		t.Fatalf("unit-caption-contains scenario delete rejected --player: %v", err)
	}
	if captionContainsRequest.Player == nil || *captionContainsRequest.Player != 2 {
		t.Fatalf("unit-caption-contains player option = %+v", captionContainsRequest)
	}
	if _, err := parseScenarioDeleteOptions(&captionContainsRequest, []string{"--unit", "83"}, true); err == nil {
		t.Fatal("unit-caption-contains scenario delete accepted --unit")
	}
	if _, err := parseScenarioDeletePlanRequest("unit-caption-contains", " "); err == nil {
		t.Fatal("unit-caption-contains scenario delete accepted empty text")
	}
	unitTypeRequest, err := parseScenarioDeletePlanRequest("unit-type", "600")
	if err != nil {
		t.Fatalf("parseScenarioDeletePlanRequest unit-type: %v", err)
	}
	if unitTypeRequest.Kind != "unit_type" || unitTypeRequest.UnitConst == nil || *unitTypeRequest.UnitConst != 600 {
		t.Fatalf("unit-type request = %+v", unitTypeRequest)
	}
	if _, err := parseScenarioDeleteOptions(&unitTypeRequest, []string{"--player", "1"}, true); err != nil {
		t.Fatalf("unit-type scenario delete rejected --player: %v", err)
	}
	if unitTypeRequest.Player == nil || *unitTypeRequest.Player != 1 {
		t.Fatalf("unit-type player option = %+v", unitTypeRequest)
	}
	if _, err := parseScenarioDeleteOptions(&unitTypeRequest, []string{"--unit", "83"}, true); err == nil {
		t.Fatal("unit-type scenario delete accepted --unit")
	}
	if _, err := parseScenarioDeletePlanRequest("unit-type", "-1"); err == nil {
		t.Fatal("unit-type scenario delete accepted negative unit const")
	}
	unitsPlayerRequest, err := parseScenarioDeletePlanRequest("units-player", "2")
	if err != nil {
		t.Fatalf("parseScenarioDeletePlanRequest units-player: %v", err)
	}
	if unitsPlayerRequest.Kind != "units_player" || unitsPlayerRequest.Player == nil || *unitsPlayerRequest.Player != 2 {
		t.Fatalf("units-player request = %+v", unitsPlayerRequest)
	}
	if _, err := parseScenarioDeleteOptions(&unitsPlayerRequest, nil, true); err != nil {
		t.Fatalf("units-player scenario delete rejected without options: %v", err)
	}
	if _, err := parseScenarioDeleteOptions(&unitsPlayerRequest, []string{"--player", "1"}, true); err == nil {
		t.Fatal("units-player scenario delete accepted --player")
	}
	if _, err := parseScenarioDeletePlanRequest("units-player", "-1"); err == nil {
		t.Fatal("units-player scenario delete accepted negative player")
	}
	triggerRequest, err := parseScenarioDeletePlanRequest("trigger-name", "Open Gate")
	if err != nil {
		t.Fatalf("parseScenarioDeletePlanRequest trigger-name: %v", err)
	}
	if triggerRequest.Kind != "trigger_name" || triggerRequest.TargetName != "Open Gate" {
		t.Fatalf("trigger-name request = %+v", triggerRequest)
	}
	triggerPrefixRequest, err := parseScenarioDeletePlanRequest("trigger-prefix", "A2K Generated:")
	if err != nil {
		t.Fatalf("parseScenarioDeletePlanRequest trigger-prefix: %v", err)
	}
	if triggerPrefixRequest.Kind != "trigger_prefix" || triggerPrefixRequest.TargetPrefix != "A2K Generated:" {
		t.Fatalf("trigger-prefix request = %+v", triggerPrefixRequest)
	}
	if _, err := parseScenarioDeletePlanRequest("trigger-prefix", " "); err == nil {
		t.Fatal("trigger-prefix scenario delete accepted empty prefix")
	}
	triggerContainsRequest, err := parseScenarioDeletePlanRequest("trigger-contains", "Shared Marker")
	if err != nil {
		t.Fatalf("parseScenarioDeletePlanRequest trigger-contains: %v", err)
	}
	if triggerContainsRequest.Kind != "trigger_contains" || triggerContainsRequest.TargetText != "Shared Marker" {
		t.Fatalf("trigger-contains request = %+v", triggerContainsRequest)
	}
	if _, err := parseScenarioDeletePlanRequest("trigger-contains", " "); err == nil {
		t.Fatal("trigger-contains scenario delete accepted empty text")
	}
	variableRequest, err := parseScenarioDeletePlanRequest("variable-name", "ScoreGate")
	if err != nil {
		t.Fatalf("parseScenarioDeletePlanRequest variable-name: %v", err)
	}
	if variableRequest.Kind != "variable_name" || variableRequest.TargetName != "ScoreGate" {
		t.Fatalf("variable-name request = %+v", variableRequest)
	}
	variablePrefixRequest, err := parseScenarioDeletePlanRequest("variable-prefix", "A2KVar:")
	if err != nil {
		t.Fatalf("parseScenarioDeletePlanRequest variable-prefix: %v", err)
	}
	if variablePrefixRequest.Kind != "variable_prefix" || variablePrefixRequest.TargetPrefix != "A2KVar:" {
		t.Fatalf("variable-prefix request = %+v", variablePrefixRequest)
	}
	if _, err := parseScenarioDeletePlanRequest("variable-prefix", " "); err == nil {
		t.Fatal("variable-prefix scenario delete accepted empty prefix")
	}
	variableContainsRequest, err := parseScenarioDeletePlanRequest("variable-contains", "ScoreMarker")
	if err != nil {
		t.Fatalf("parseScenarioDeletePlanRequest variable-contains: %v", err)
	}
	if variableContainsRequest.Kind != "variable_contains" || variableContainsRequest.TargetText != "ScoreMarker" {
		t.Fatalf("variable-contains request = %+v", variableContainsRequest)
	}
	if _, err := parseScenarioDeletePlanRequest("variable-contains", " "); err == nil {
		t.Fatal("variable-contains scenario delete accepted empty text")
	}
	stringRequest, err := parseScenarioDeletePlanRequest("string-text", "Exact display text")
	if err != nil {
		t.Fatalf("parseScenarioDeletePlanRequest string-text: %v", err)
	}
	if stringRequest.Kind != "string_text" || stringRequest.TargetText != "Exact display text" {
		t.Fatalf("string-text request = %+v", stringRequest)
	}
	if _, err := parseScenarioDeletePlanRequest("string-text", " "); err == nil {
		t.Fatal("string-text scenario delete accepted empty text")
	}
	stringPrefixRequest, err := parseScenarioDeletePlanRequest("string-prefix", "A2K String:")
	if err != nil {
		t.Fatalf("parseScenarioDeletePlanRequest string-prefix: %v", err)
	}
	if stringPrefixRequest.Kind != "string_prefix" || stringPrefixRequest.TargetPrefix != "A2K String:" {
		t.Fatalf("string-prefix request = %+v", stringPrefixRequest)
	}
	if _, err := parseScenarioDeletePlanRequest("string-prefix", " "); err == nil {
		t.Fatal("string-prefix scenario delete accepted empty prefix")
	}
	stringContainsRequest, err := parseScenarioDeletePlanRequest("string-contains", "StringMarker")
	if err != nil {
		t.Fatalf("parseScenarioDeletePlanRequest string-contains: %v", err)
	}
	if stringContainsRequest.Kind != "string_contains" || stringContainsRequest.TargetText != "StringMarker" {
		t.Fatalf("string-contains request = %+v", stringContainsRequest)
	}
	if _, err := parseScenarioDeletePlanRequest("string-contains", " "); err == nil {
		t.Fatal("string-contains scenario delete accepted empty text")
	}
}

func TestParseScenarioDeleteOptionsRejectsMutatingTextAndWrongFilters(t *testing.T) {
	request, err := parseScenarioDeletePlanRequest("trigger", "4")
	if err != nil {
		t.Fatalf("parseScenarioDeletePlanRequest: %v", err)
	}
	if _, err := parseScenarioDeleteOptions(&request, []string{"--text"}, false); err == nil {
		t.Fatal("mutating scenario delete accepted --text")
	}
	if _, err := parseScenarioDeleteOptions(&request, []string{"--player", "1"}, true); err == nil {
		t.Fatal("non-area scenario delete accepted --player")
	}
	if _, err := parseScenarioDeleteOptions(&request, []string{"--unit", "83"}, true); err == nil {
		t.Fatal("non-area scenario delete accepted --unit")
	}
}

func TestParseDatDeleteRequestOptions(t *testing.T) {
	request, text, err := parseDatDeleteRequest("unit", "74", []string{"--civ", "1", "--text"}, true)
	if err != nil {
		t.Fatalf("parseDatDeleteRequest: %v", err)
	}
	if request.Section != "unit" || request.ID != 74 || request.CivID == nil || *request.CivID != 1 || !text {
		t.Fatalf("dat delete request text=%v request=%+v", text, request)
	}

	request, text, err = parseDatDeleteRequest("unit", "74", []string{"--all-civs"}, false)
	if err != nil {
		t.Fatalf("parseDatDeleteRequest all-civs: %v", err)
	}
	if !request.AllCivs || text {
		t.Fatalf("dat all-civs text=%v request=%+v", text, request)
	}

	request, text, err = parseDatDeleteRequest("effect-command", "12:3", []string{"--text"}, true)
	if err != nil {
		t.Fatalf("parseDatDeleteRequest effect-command: %v", err)
	}
	if request.Section != "effect_command" || request.ID != 12 ||
		request.EffectID == nil || *request.EffectID != 12 ||
		request.CommandIndex == nil || *request.CommandIndex != 3 || !text {
		t.Fatalf("effect-command request text=%v request=%+v", text, request)
	}

	request, text, err = parseDatDeleteRequest("sound-item", "22:4", []string{"--text"}, true)
	if err != nil {
		t.Fatalf("parseDatDeleteRequest sound-item: %v", err)
	}
	if request.Section != "sound_item" || request.ID != 22 ||
		request.SoundID == nil || *request.SoundID != 22 ||
		request.ItemIndex == nil || *request.ItemIndex != 4 || !text {
		t.Fatalf("sound-item request text=%v request=%+v", text, request)
	}

	request, text, err = parseDatDeleteRequest("unit-attack", "1:74:2", []string{"--text"}, true)
	if err != nil {
		t.Fatalf("parseDatDeleteRequest unit-attack: %v", err)
	}
	if request.Section != "unit_attack" || request.ID != 74 ||
		request.CivID == nil || *request.CivID != 1 ||
		request.UnitID == nil || *request.UnitID != 74 ||
		request.RowIndex == nil || *request.RowIndex != 2 || !text {
		t.Fatalf("unit-row request text=%v request=%+v", text, request)
	}

	request, text, err = parseDatDeleteRequest("unit-header-task", "74:1", []string{"--text"}, true)
	if err != nil {
		t.Fatalf("parseDatDeleteRequest unit-header-task: %v", err)
	}
	if request.Section != "unit_header_task" || request.ID != 74 ||
		request.UnitHeaderID == nil || *request.UnitHeaderID != 74 ||
		request.RowIndex == nil || *request.RowIndex != 1 || !text {
		t.Fatalf("unit-header-task request text=%v request=%+v", text, request)
	}

	request, text, err = parseDatDeleteRequest("graphic-delta", "339:0", []string{"--text"}, true)
	if err != nil {
		t.Fatalf("parseDatDeleteRequest graphic-delta: %v", err)
	}
	if request.Section != "graphic_delta" || request.ID != 339 ||
		request.GraphicID == nil || *request.GraphicID != 339 ||
		request.RowIndex == nil || *request.RowIndex != 0 || !text {
		t.Fatalf("graphic-row request text=%v request=%+v", text, request)
	}
}

func TestParseDatDeleteRequestRejectsMutatingText(t *testing.T) {
	if _, _, err := parseDatDeleteRequest("sound", "1", []string{"--text"}, false); err == nil {
		t.Fatal("mutating dat delete accepted --text")
	}
	if _, _, err := parseDatDeleteRequest("sound", "not-an-id", nil, true); err == nil {
		t.Fatal("bad dat delete id accepted")
	}
	if _, _, err := parseDatDeleteRequest("effect-command", "1", nil, true); err == nil {
		t.Fatal("bad effect-command target accepted")
	}
	if _, _, err := parseDatDeleteRequest("effect-command", "1:0", []string{"--all-civs"}, true); err == nil {
		t.Fatal("effect-command accepted all-civs")
	}
	if _, _, err := parseDatDeleteRequest("sound-item", "1", nil, true); err == nil {
		t.Fatal("bad sound-item target accepted")
	}
	if _, _, err := parseDatDeleteRequest("sound-item", "1:0", []string{"--civ", "1"}, true); err == nil {
		t.Fatal("sound-item accepted civ")
	}
	if _, _, err := parseDatDeleteRequest("unit-task", "1:74", nil, true); err == nil {
		t.Fatal("bad unit-row target accepted")
	}
	if _, _, err := parseDatDeleteRequest("unit-task", "1:74:0", []string{"--all-civs"}, true); err == nil {
		t.Fatal("unit-row accepted all-civs")
	}
	if _, _, err := parseDatDeleteRequest("unit-header-task", "74", nil, true); err == nil {
		t.Fatal("bad unit-header-task target accepted")
	}
	if _, _, err := parseDatDeleteRequest("unit-header-task", "74:0", []string{"--civ", "1"}, true); err == nil {
		t.Fatal("unit-header-task accepted civ")
	}
	if _, _, err := parseDatDeleteRequest("graphic-delta", "339", nil, true); err == nil {
		t.Fatal("bad graphic-row target accepted")
	}
	if _, _, err := parseDatDeleteRequest("graphic-angle-sound", "339:0", []string{"--all-civs"}, true); err == nil {
		t.Fatal("graphic-row accepted all-civs")
	}
}

func TestParseDatRefsOptions(t *testing.T) {
	opts, err := parseDatRefsOptions([]string{
		"--class", "candidate_operand",
		"--confidence", "typed_effect_command_reference",
		"--source-section", "effect",
		"--limit", "3",
		"--all",
		"--text",
	})
	if err != nil {
		t.Fatalf("parseDatRefsOptions: %v", err)
	}
	if opts.Filters.Class != "candidate_operand" ||
		opts.Filters.Confidence != "typed_effect_command_reference" ||
		opts.Filters.SourceSection != "effect" ||
		opts.Filters.Limit != 0 || !opts.Text {
		t.Fatalf("refs options = %+v", opts)
	}
	if _, err := parseDatRefsOptions([]string{"--class", "bad"}); err == nil {
		t.Fatal("bad refs class accepted")
	}
	if _, err := parseDatRefsOptions([]string{"--limit", "-1"}); err == nil {
		t.Fatal("negative refs limit accepted")
	}
}

func TestParseDatDisconnectRequest(t *testing.T) {
	request, recipe, err := parseDatDisconnectRequest("research", "244")
	if err != nil {
		t.Fatalf("parseDatDisconnectRequest tech: %v", err)
	}
	if request.Section != "tech" || request.ID != 244 || len(recipe.DisconnectTechs) != 1 || recipe.DisconnectTechs[0] != 244 {
		t.Fatalf("tech disconnect request=%+v recipe=%+v", request, recipe)
	}

	request, recipe, err = parseDatDisconnectRequest("units", "83")
	if err != nil {
		t.Fatalf("parseDatDisconnectRequest unit: %v", err)
	}
	if request.Section != "unit" || request.ID != 83 || len(recipe.DisconnectUnits) != 1 || recipe.DisconnectUnits[0] != 83 {
		t.Fatalf("unit disconnect request=%+v recipe=%+v", request, recipe)
	}
}

func TestParseDatDisconnectRequestRejectsUnsupportedTargets(t *testing.T) {
	if _, _, err := parseDatDisconnectRequest("sound", "1"); err == nil {
		t.Fatal("sound disconnect accepted")
	}
	if _, _, err := parseDatDisconnectRequest("tech", "-1"); err == nil {
		t.Fatal("negative disconnect id accepted")
	}
	if _, _, err := parseDatDisconnectRequest("tech", "not-an-id"); err == nil {
		t.Fatal("bad disconnect id accepted")
	}
}

func TestParseGraphicPatchOptionsScalars(t *testing.T) {
	patch, err := parseGraphicPatchOptions([]string{
		"--name", "Test Graphic",
		"--layer", "25",
		"--player-color", "2",
		"--coordinates", "1,2,3,4",
		"--sound-id", "12",
		"--wwise-sound-id", "34",
		"--frame-duration", "0.125",
		"--sequence-type", "7",
	})
	if err != nil {
		t.Fatalf("parseGraphicPatchOptions: %v", err)
	}
	if patch.Name == nil || *patch.Name != "Test Graphic" {
		t.Fatalf("name = %+v", patch.Name)
	}
	if patch.Layer == nil || *patch.Layer != 25 || patch.PlayerColor == nil || *patch.PlayerColor != 2 {
		t.Fatalf("layer/player_color = %+v/%+v", patch.Layer, patch.PlayerColor)
	}
	if len(patch.Coordinates) != 4 || patch.Coordinates[3] != 4 {
		t.Fatalf("coordinates = %+v", patch.Coordinates)
	}
	if patch.SoundID == nil || *patch.SoundID != 12 || patch.WwiseSoundID == nil || *patch.WwiseSoundID != 34 {
		t.Fatalf("sound ids = %+v/%+v", patch.SoundID, patch.WwiseSoundID)
	}
	if patch.FrameDuration == nil || *patch.FrameDuration != 0.125 {
		t.Fatalf("frame_duration = %+v", patch.FrameDuration)
	}
	if patch.SequenceType == nil || *patch.SequenceType != 7 {
		t.Fatalf("sequence_type = %+v", patch.SequenceType)
	}
}

func TestParseUnitPatchOptionsRows(t *testing.T) {
	patch, err := parseUnitPatchOptions([]string{
		"--attribute", "1,amount=3,flag=2",
		"--damage-graphic", "0,graphic_id=123,damage_percent=50",
		"--type50-attack", "2,class=13,value=9",
		"--type50-armour", "1,value=4",
		"--cost", "0,type=0,amount=55,flag=1",
		"--train-location", "0,time=19,unit=109,button=2,hotkey=16121",
		"--task", "0,action_type=3,attribute_types=-1:-1:-1:-1,work_range=2.25,combat_level=1",
	})
	if err != nil {
		t.Fatalf("parseUnitPatchOptions: %v", err)
	}
	if len(patch.Attributes) != 1 || patch.Attributes[0].Index != 1 || patch.Attributes[0].Amount == nil || *patch.Attributes[0].Amount != 3 {
		t.Fatalf("attributes = %+v", patch.Attributes)
	}
	if len(patch.DamageGraphics) != 1 || patch.DamageGraphics[0].GraphicID == nil || *patch.DamageGraphics[0].GraphicID != 123 {
		t.Fatalf("damage graphics = %+v", patch.DamageGraphics)
	}
	if len(patch.Type50Attacks) != 1 || patch.Type50Attacks[0].Class == nil || *patch.Type50Attacks[0].Class != 13 || patch.Type50Attacks[0].Value == nil || *patch.Type50Attacks[0].Value != 9 {
		t.Fatalf("attacks = %+v", patch.Type50Attacks)
	}
	if len(patch.Type50Armours) != 1 || patch.Type50Armours[0].Value == nil || *patch.Type50Armours[0].Value != 4 {
		t.Fatalf("armours = %+v", patch.Type50Armours)
	}
	if len(patch.Costs) != 1 || patch.Costs[0].AttributeType == nil || *patch.Costs[0].AttributeType != 0 || patch.Costs[0].Amount == nil || *patch.Costs[0].Amount != 55 {
		t.Fatalf("costs = %+v", patch.Costs)
	}
	if len(patch.TrainLocations) != 1 || patch.TrainLocations[0].TrainButton == nil || *patch.TrainLocations[0].TrainButton != 2 {
		t.Fatalf("train locations = %+v", patch.TrainLocations)
	}
	if len(patch.Tasks) != 1 || patch.Tasks[0].ActionType == nil || *patch.Tasks[0].ActionType != 3 || patch.Tasks[0].WorkRange == nil || *patch.Tasks[0].WorkRange != 2.25 || len(patch.Tasks[0].AttributeTypes) != 4 {
		t.Fatalf("tasks = %+v", patch.Tasks)
	}
}

func TestParseUnitPatchOptionsRejectsBadRowSpec(t *testing.T) {
	if _, err := parseUnitPatchOptions([]string{"--task", "0,attribute_types=1:2"}); err == nil {
		t.Fatal("bad task attribute_types accepted")
	}
	if _, err := parseUnitPatchOptions([]string{"--type50-attack", "2,unknown=9"}); err == nil {
		t.Fatal("unknown attack field accepted")
	}
}

func TestCombinedDatRecipeUnmarshal(t *testing.T) {
	var recipe combinedDatRecipe
	err := json.Unmarshal([]byte(`{
		"units": [{"civ_id": 0, "unit_id": 83, "hit_points": 26}],
		"unit_headers": [{"id": 83, "tasks": [{"index": 0, "action_type": 3}]}],
		"create_effect": {"name": "combined effect"}
	}`), &recipe)
	if err != nil {
		t.Fatalf("unmarshal combined recipe: %v", err)
	}
	if recipe.datfileEmpty() || len(recipe.Span.Units) != 1 {
		t.Fatalf("span recipe not populated: %+v", recipe.Span)
	}
	if recipe.datcodecEmpty() || recipe.Codec.CreateEffect == nil || recipe.Codec.CreateEffect.Name != "combined effect" {
		t.Fatalf("codec recipe not populated: %+v", recipe.Codec)
	}
	if len(recipe.Codec.UnitHeaders) != 1 || recipe.Codec.UnitHeaders[0].ID != 83 || len(recipe.Codec.UnitHeaders[0].Tasks) != 1 {
		t.Fatalf("codec unit header recipe not populated: %+v", recipe.Codec.UnitHeaders)
	}
}

func TestCombinedDatRecipeRoutesCreateGraphicUnitToCodecOnly(t *testing.T) {
	var recipe combinedDatRecipe
	err := json.Unmarshal([]byte(`{
		"create_graphic": {"from": 1711, "name": "one graphic"},
		"create_unit": {"from_civ_id": 0, "from_unit_id": 83, "civ_ids": [0, 1]},
		"graphics": [{"id": 1711, "name": "patched graphic"}],
		"units": [{"civ_id": 0, "unit_id": 83, "hit_points": 26}]
	}`), &recipe)
	if err != nil {
		t.Fatalf("unmarshal combined create recipe: %v", err)
	}
	if recipe.Span.CreateGraphic != nil || len(recipe.Span.CreateGraphics) != 0 || recipe.Span.CreateUnit != nil || len(recipe.Span.CreateUnits) != 0 {
		t.Fatalf("overlapping creates must be routed away from span recipe to avoid double-create: %+v", recipe.Span)
	}
	if recipe.Codec.CreateGraphic == nil || recipe.Codec.CreateGraphic.From != 1711 || recipe.Codec.CreateUnit == nil || recipe.Codec.CreateUnit.FromUnitID != 83 {
		t.Fatalf("codec creates not populated: %+v", recipe.Codec)
	}
	if len(recipe.Span.Graphics) != 1 || len(recipe.Span.Units) != 1 {
		t.Fatalf("non-overlapping span patches must remain populated: %+v", recipe.Span)
	}
}
