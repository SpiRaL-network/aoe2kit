package scenario

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"aoe2kit/pkg/aoe2"
	"aoe2kit/pkg/datcache"
	"aoe2kit/pkg/enginefacts"
)

type DiffReport struct {
	Before  string               `json:"before"`
	After   string               `json:"after"`
	Same    bool                 `json:"same"`
	Changes []DiffChange         `json:"changes"`
	Fields  []StructureFieldDiff `json:"fields,omitempty"`
}

type DiffChange struct {
	Kind         string `json:"kind"`
	Field        string `json:"field"`
	Player       string `json:"player,omitempty"`
	SectionIndex *int   `json:"section_index,omitempty"`
	IndexBase    string `json:"index_base,omitempty"`
	Before       any    `json:"before,omitempty"`
	After        any    `json:"after,omitempty"`
	Detail       string `json:"detail,omitempty"`
}

type UnitDiffSummary struct {
	ReferenceID int     `json:"reference_id"`
	UnitConst   int     `json:"unit_const"`
	X           float64 `json:"x"`
	Y           float64 `json:"y"`
}

type LintReport struct {
	Path         string                 `json:"path,omitempty"`
	OK           bool                   `json:"ok"`
	Verification aoe2.VerificationClaim `json:"verification"`
	Issues       []LintIssue            `json:"issues,omitempty"`
	Summary      LintSummary            `json:"summary"`
}

type LintIssue struct {
	Severity     string `json:"severity"`
	Code         string `json:"code"`
	Checkpoint   string `json:"checkpoint,omitempty"`
	Message      string `json:"message"`
	Why          string `json:"why,omitempty"`
	Fix          string `json:"fix,omitempty"`
	FactID       string `json:"fact_id,omitempty"`
	FactTier     string `json:"fact_tier,omitempty"`
	VerifiedDate string `json:"verified_date,omitempty"`
	FixtureRef   string `json:"fixture_ref,omitempty"`
}

type LintSummary struct {
	Triggers int `json:"triggers"`
	Units    int `json:"units"`
	Players  int `json:"players"`
	AIFiles  int `json:"ai_files"`
}

type LintOptions struct {
	IncludeProvisional bool
	LoadSafety         bool
	SemanticInvariants bool
	// DatPath enables DAT-backed unit availability checks. Without it the
	// scenario linter reports that this rule was skipped rather than guessing.
	DatPath string
	// Generated marks output produced by Kit. Generated artifacts are held to
	// stricter semantic checks than foreign, hand-authored scenarios.
	Generated bool
}

func DiffFiles(beforePath, afterPath string) (DiffReport, error) {
	before, err := Open(beforePath)
	if err != nil {
		return DiffReport{}, fmt.Errorf("open before: %w", err)
	}
	after, err := Open(afterPath)
	if err != nil {
		return DiffReport{}, fmt.Errorf("open after: %w", err)
	}
	return Diff(before, after), nil
}

func Diff(before, after *File) DiffReport {
	report := DiffReport{Before: before.Path, After: after.Path}
	addChange := func(kind, field string, b, a any, detail string) {
		report.Changes = append(report.Changes, DiffChange{Kind: kind, Field: field, Before: b, After: a, Detail: detail})
	}
	if before.Version != after.Version {
		addChange("scenario", "version", before.Version, after.Version, "")
	}
	if before.InflatedBytes != after.InflatedBytes {
		addChange("scenario", "inflated_body_bytes", before.InflatedBytes, after.InflatedBytes, "")
	}
	if before.Map != nil && after.Map != nil {
		if before.Map.Width != after.Map.Width || before.Map.Height != after.Map.Height {
			addChange("map", "dimensions", fmt.Sprintf("%dx%d", before.Map.Width, before.Map.Height), fmt.Sprintf("%dx%d", after.Map.Width, after.Map.Height), "")
		}
		for _, terrainID := range sortedTerrainIDs(before.Map.TerrainCounts, after.Map.TerrainCounts) {
			b := before.Map.TerrainCounts[terrainID]
			a := after.Map.TerrainCounts[terrainID]
			if b != a {
				addChange("map", fmt.Sprintf("terrain_%d_count", terrainID), b, a, "")
			}
		}
		if !secondaryGameModesEqual(before.Map.SecondaryGameModes, after.Map.SecondaryGameModes) {
			addChange("map", "secondary_game_modes", before.Map.SecondaryGameModes, after.Map.SecondaryGameModes, "bitfield")
		}
	}
	if before.Triggers != nil && after.Triggers != nil {
		if before.Triggers.Count != after.Triggers.Count {
			addChange("triggers", "count", before.Triggers.Count, after.Triggers.Count, "")
		}
		if before.Triggers.GraphSHA256 != after.Triggers.GraphSHA256 {
			addChange("triggers", "graph_sha256", before.Triggers.GraphSHA256, after.Triggers.GraphSHA256, "")
		}
		diffTriggerSummaries(before.Triggers.Triggers, after.Triggers.Triggers, addChange)
	}
	if before.Units != nil && after.Units != nil {
		if before.Units.NumberOfUnitSections != after.Units.NumberOfUnitSections {
			addChange("units", "number_of_unit_sections", before.Units.NumberOfUnitSections, after.Units.NumberOfUnitSections, "")
		}
		if before.Units.NumberOfPlayers != after.Units.NumberOfPlayers {
			addChange("units", "number_of_players", before.Units.NumberOfPlayers, after.Units.NumberOfPlayers, "")
		}
		if before.Units.Total != after.Units.Total {
			addChange("units", "total", before.Units.Total, after.Units.Total, "")
		}
		beforeCounts := unitCountsByPlayer(before.Units)
		afterCounts := unitCountsByPlayer(after.Units)
		for _, player := range sortedIntKeys(beforeCounts, afterCounts) {
			if beforeCounts[player] != afterCounts[player] {
				addChange("units", fmt.Sprintf("player_%d_count", player), beforeCounts[player], afterCounts[player], "")
			}
		}
		diffUnitsByReference(before.Units, after.Units, addChange)
	}
	beforeSettings := before.Settings()
	afterSettings := after.Settings()
	diffOptions(beforeSettings.Options, afterSettings.Options, addChange)
	diffDiplomacy(beforeSettings.Diplomacy, afterSettings.Diplomacy, addChange)
	diffMessages(beforeSettings.Messages, afterSettings.Messages, addChange)
	diffCinematics(beforeSettings.Cinematics, afterSettings.Cinematics, addChange)
	diffVictory(beforeSettings.Victory, afterSettings.Victory, addChange)
	diffPlayers(beforeSettings.Players, afterSettings.Players, addChange)
	decoratePlayerChanges(&report, beforeSettings.Players)
	diffDiplomacyMirrors(beforeSettings.DiplomacyMirrors, afterSettings.DiplomacyMirrors, addChange)
	decorateDiplomacyMirrorChanges(&report)
	diffAI(before.AI, after.AI, addChange)
	report.Same = len(report.Changes) == 0
	return report
}

func secondaryGameModesEqual(before, after SecondaryGameModesInfo) bool {
	if before.Bits != after.Bits || len(before.Names) != len(after.Names) {
		return false
	}
	for i := range before.Names {
		if before.Names[i] != after.Names[i] {
			return false
		}
	}
	return true
}

func diffDiplomacyMirrors(before, after []DiplomacyMirrorSettings, add func(kind, field string, b, a any, detail string)) {
	limit := len(before)
	if len(after) < limit {
		limit = len(after)
	}
	for i := 0; i < limit; i++ {
		if before[i].AlliedVictory != after[i].AlliedVictory {
			add("units", fmt.Sprintf("mirror_player_%d.aok_allied_victory", i), before[i].AlliedVictory, after[i].AlliedVictory, "")
		}
		for _, mirror := range []struct {
			name string
			b    []int
			a    []int
		}{
			{name: "diplomacy_for_interaction", b: before[i].Interaction, a: after[i].Interaction},
			{name: "diplomacy_for_ai_system", b: before[i].AISystem, a: after[i].AISystem},
		} {
			itemLimit := len(mirror.b)
			if len(mirror.a) < itemLimit {
				itemLimit = len(mirror.a)
			}
			for j := 0; j < itemLimit; j++ {
				if mirror.b[j] != mirror.a[j] {
					add("units", fmt.Sprintf("mirror_player_%d.%s[%d]", i, mirror.name, j), mirror.b[j], mirror.a[j], "")
				}
			}
		}
	}
}

func decorateDiplomacyMirrorChanges(report *DiffReport) {
	for i := range report.Changes {
		change := &report.Changes[i]
		if change.Kind != "units" || !strings.HasPrefix(change.Field, "mirror_player_") {
			continue
		}
		dot := strings.IndexByte(change.Field, '.')
		open := strings.LastIndexByte(change.Field, '[')
		close := strings.LastIndexByte(change.Field, ']')
		if dot < 0 {
			continue
		}
		source, err := strconv.Atoi(strings.TrimPrefix(change.Field[:dot], "mirror_player_"))
		if err != nil {
			continue
		}
		mirror := change.Field[dot+1:]
		change.Player = fmt.Sprintf("P%d", source+1)
		if mirror == "aok_allied_victory" {
			change.Field = fmt.Sprintf("units.player.%s.%s", change.Player, mirror)
		} else {
			if open < dot || close < open {
				continue
			}
			target, err := strconv.Atoi(change.Field[open+1 : close])
			if err != nil {
				continue
			}
			mirror = change.Field[dot+1 : open]
			targetLabel := "Gaia"
			if target > 0 {
				targetLabel = fmt.Sprintf("P%d", target)
			}
			change.Field = fmt.Sprintf("units.player.%s.%s.%s", change.Player, mirror, targetLabel)
		}
		sectionIndex := source
		change.SectionIndex = &sectionIndex
		change.IndexBase = "players_from_1"
	}
}

func decoratePlayerChanges(report *DiffReport, players []PlayerSettings) {
	for i := range report.Changes {
		change := &report.Changes[i]
		if change.Kind != "players" || !strings.HasPrefix(change.Field, "player_") {
			continue
		}
		var rawIndex int
		var field string
		if _, err := fmt.Sscanf(change.Field, "player_%d.%s", &rawIndex, &field); err != nil || rawIndex < 0 || rawIndex >= len(players) {
			continue
		}
		ref, ok := players[rawIndex].FieldReferences[field]
		if !ok {
			ref = PlayerReference{
				Player:       players[rawIndex].PlayerLabel,
				SectionIndex: players[rawIndex].SectionIndex,
				IndexBase:    players[rawIndex].IndexBase,
			}
		}
		change.Field = "player." + ref.Player + "." + field
		change.Player = ref.Player
		sectionIndex := ref.SectionIndex
		change.SectionIndex = &sectionIndex
		change.IndexBase = ref.IndexBase
	}
}

func LintFile(path string) (LintReport, error) {
	return LintFileWithOptions(path, LintOptions{})
}

func LintFileWithOptions(path string, opts LintOptions) (LintReport, error) {
	file, err := Open(path)
	if err != nil {
		if version, body, partialErr := rawScenarioBody(path); partialErr == nil && version == "1.59" {
			return lintPartial159(path, body, err, opts), nil
		}
		return LintReport{}, err
	}
	return file.LintWithOptions(opts), nil
}

func lintPartial159(path string, body []byte, parseErr error, opts LintOptions) LintReport {
	report := LintReport{Path: path, Verification: aoe2.StructureVerification(false)}
	report.addIssueDetail("warning", "format_partial_structure",
		fmt.Sprintf("DE 1.59 structural parse stopped before semantic lint: %v", parseErr),
		"The 1.59 Units layout is not yet byte-mapped, so trigger and effect indices cannot be trusted.",
		"Use the partial string findings as leads only; keep the scenario read-only until the 1.59 structure is mapped.")
	lintPartial159ScriptCallText(&report, extractPrintableStrings(body), opts)
	report.OK = !hasLintSeverity(report.Issues, "error")
	return report
}

func lintPartial159ScriptCallText(report *LintReport, texts []string, opts LintOptions) {
	severity := "warning"
	if opts.Generated {
		severity = "error"
	}
	for _, text := range texts {
		lower := strings.ToLower(text)
		if !strings.Contains(lower, "script call") || !strings.Contains(lower, "effect") || !strings.Contains(lower, "function") {
			continue
		}
		report.addIssueDetail(severity, "semantic_script_call_target_undefined",
			"1.59 partial read found prose describing a script-call effect, but the trigger/effect structure is not decoded; the target may be comment text rather than a function",
			"This is a conservative crash-prevention heuristic for the known DE access-violation signature; partial strings cannot prove which effect owns the text.",
			"Open the scenario in the editor, replace the prose with an exact-case parameterless XS function call, and rerun lint after 1.59 structural support is available.")
	}
}

func (f *File) Lint() LintReport {
	return f.LintWithOptions(LintOptions{})
}

func (f *File) LintWithOptions(opts LintOptions) LintReport {
	report := LintReport{
		Path: f.Path,
		Summary: LintSummary{
			Players: len(f.Players),
			AIFiles: len(f.AI),
		},
	}
	if f.Triggers != nil {
		report.Summary.Triggers = f.Triggers.Count
		if !f.Triggers.InvariantOK {
			report.addIssue("error", "trigger_invariant", f.Triggers.InvariantNote)
		}
		seenNames := map[string]int{}
		for _, trigger := range f.Triggers.Triggers {
			if trigger.Name != "" {
				seenNames[trigger.Name]++
			}
			if trigger.Enabled != 0 && trigger.Effects == 0 && trigger.Conditions == 0 {
				report.addIssue("warning", "empty_enabled_trigger", fmt.Sprintf("trigger %d %q is enabled with no effects or conditions", trigger.Index, trigger.Name))
			}
		}
		if f.root != nil {
			if section := f.root.section("Triggers"); section != nil {
				lintTriggerRuntimeHazards(&report, section.list("trigger_data"), opts)
			}
			lintTriggerPlayerCoverage(&report, f, opts)
		}
		lintCorruptStrings(&report, f.headerRoot)
		if f.root != nil {
			for _, section := range f.root.Sections {
				lintCorruptStrings(&report, section)
			}
		}
		for name, count := range seenNames {
			if count > 1 {
				report.addIssue("warning", "duplicate_trigger_name", fmt.Sprintf("%q appears %d times", name, count))
			}
		}
	} else {
		report.addIssue("error", "missing_triggers", "scenario has no parsed trigger section")
	}
	lintDiplomacyMirrors(&report, f)
	if opts.LoadSafety {
		lintLoadSafety(&report, f, opts)
	}
	if opts.SemanticInvariants {
		lintSemanticInvariants(&report, f, opts)
		lintScriptCallTargets(&report, f, opts)
	}
	if f.Map != nil {
		if f.Map.Width*f.Map.Height != f.Map.TileCount {
			report.addIssue("error", "map_tile_count", fmt.Sprintf("map dimensions %dx%d imply %d tiles but parsed %d", f.Map.Width, f.Map.Height, f.Map.Width*f.Map.Height, f.Map.TileCount))
		}
		if err := ValidateScenarioMapSize(f.Map.Width, f.Map.Height); err != nil {
			report.addIssueFact("error", "map_size_preset", err.Error(), "scenario.map_size_editor_presets_are_discrete", opts)
		}
	} else {
		report.addIssue("error", "missing_map", "scenario has no parsed map section")
	}
	if f.Units != nil {
		report.Summary.Units = f.Units.Total
		if f.Units.NumberOfUnitSections != len(f.Units.Sections) {
			report.addIssue("error", "unit_section_count_mismatch", fmt.Sprintf("Units number_of_unit_sections=%d but parsed %d player unit sections", f.Units.NumberOfUnitSections, len(f.Units.Sections)))
		}
		if f.Units.NumberOfPlayers != len(f.Units.Sections) {
			report.addIssue("error", "unit_owner_count_mismatch", fmt.Sprintf("Units number_of_players=%d but parsed %d player unit sections; current DE editor-authored scenarios use all Gaia+8 unit-owner sections", f.Units.NumberOfPlayers, len(f.Units.Sections)))
		}
		seenRefs := map[int]string{}
		unitCountsByPlayer := map[int]int{}
		for _, player := range f.Units.Sections {
			unitCountsByPlayer[player.Player] = len(player.Units)
			if player.Count != len(player.Units) {
				report.addIssue("error", "unit_count_mismatch", fmt.Sprintf("player %d unit_count=%d but parsed %d units", player.Player, player.Count, len(player.Units)))
			}
			for _, unit := range player.Units {
				lintUnitPlacement(&report, player.Player, unit)
				if unit.ReferenceID < 0 {
					report.addIssue("warning", "negative_unit_reference", fmt.Sprintf("player %d unit %d has reference_id %d", player.Player, unit.Index, unit.ReferenceID))
				}
				if unit.ReferenceID == 0 {
					continue
				}
				where := fmt.Sprintf("P%d[%d]", player.Player, unit.Index)
				if prev, ok := seenRefs[unit.ReferenceID]; ok {
					report.addIssue("error", "duplicate_unit_reference", fmt.Sprintf("reference_id %d appears at %s and %s", unit.ReferenceID, prev, where))
				} else {
					seenRefs[unit.ReferenceID] = where
				}
			}
		}
		lintActiveComputerStarterInjection(&report, f.Players, unitCountsByPlayer, opts)
		lintGaiaOnlyUnits(&report, f, opts)
		if dataHeader := f.root.section("DataHeader"); dataHeader != nil {
			if nextID, ok := dataHeader.intValue("next_unit_id_to_place"); ok {
				maxReferenceID := 0
				for _, player := range f.Units.Sections {
					for _, unit := range player.Units {
						if unit.ReferenceID > maxReferenceID {
							maxReferenceID = unit.ReferenceID
						}
					}
				}
				if nextID <= maxReferenceID {
					report.addIssue("error", "next_unit_id_collision", fmt.Sprintf("DataHeader.next_unit_id_to_place=%d is not above maximum unit reference_id=%d", nextID, maxReferenceID))
				}
			}
		}
	} else {
		report.addIssue("error", "missing_units", "scenario has no parsed unit section")
	}
	for _, player := range f.Players {
		if player.Active && !player.Human && player.AIName == "" && player.AIType != 0 {
			report.addIssue("warning", "active_ai_without_name", fmt.Sprintf("%s is active non-human with ai_type=%d and no AI name", playerLabel(player.Player), player.AIType))
		}
	}
	for _, ai := range f.AI {
		if ai.Name == "" {
			report.addIssue("warning", "embedded_ai_missing_name", fmt.Sprintf("embedded AI index %d has no name", ai.Index))
		}
		if ai.Name != "" && ai.ContentBytes == 0 {
			report.addIssue("warning", "embedded_ai_empty_content", fmt.Sprintf("embedded AI %q has no content", ai.Name))
		}
	}
	report.OK = !hasLintSeverity(report.Issues, "error")
	report.Verification = aoe2.StructureVerification(report.OK)
	return report
}

func lintDiplomacyMirrors(report *LintReport, f *File) {
	settings := f.Settings()
	if len(settings.DiplomacyMirrors) == 0 || len(settings.Diplomacy.Matrix) == 0 {
		return
	}
	for _, mirror := range settings.DiplomacyMirrors {
		primaryIndex := mirror.Player
		if primaryIndex < 0 || primaryIndex >= len(settings.Diplomacy.Matrix) {
			continue
		}
		primary := settings.Diplomacy.Matrix[primaryIndex]
		for target, stance := range primary {
			if target == primaryIndex {
				continue // the engine uses special self-stances in the mirror arrays.
			}
			interaction, aiSystem, err := diplomacyMirrorValues(stance)
			mirrorTarget := target + 1 // mirror slot 0 is Gaia.
			if err != nil || mirrorTarget >= len(mirror.Interaction) || mirrorTarget >= len(mirror.AISystem) {
				continue
			}
			if mirror.Interaction[mirrorTarget] != interaction || mirror.AISystem[mirrorTarget] != aiSystem {
				report.addIssueDetail("error", "diplomacy_mirror_mismatch",
					fmt.Sprintf("%s toward P%d differs across Diplomacy and Units mirrors: primary=%d interaction=%d ai_system=%d", mirror.PlayerLabel, target+1, stance, mirror.Interaction[mirrorTarget], mirror.AISystem[mirrorTarget]),
					"DE stores diplomacy in the main Diplomacy table and two Units player_data_3 mirrors; the mirrors use distinct verified enums.",
					"Write diplomacy through Kit so all three representations are updated together.")
			}
		}
		if primaryIndex < len(settings.Diplomacy.AlliedVictory) && mirror.AlliedVictory != boolInt(settings.Diplomacy.AlliedVictory[primaryIndex]) {
			report.addIssueDetail("error", "allied_victory_mirror_mismatch",
				fmt.Sprintf("%s allied victory differs across Diplomacy and Units mirrors: primary=%t mirror=%d", mirror.PlayerLabel, settings.Diplomacy.AlliedVictory[primaryIndex], mirror.AlliedVictory),
				"DE stores allied victory in both the Diplomacy section and Units player_data_3.",
				"Write allied victory through Kit so both representations are updated together.")
		}
	}
}

func lintUnitPlacement(report *LintReport, player int, unit UnitSummary) {
	if !likelyBlockingUnit(unit.UnitConst) || (!offGridCoordinate(unit.X) && !offGridCoordinate(unit.Y)) {
		return
	}
	report.addIssueDetail("warning", "unit_off_grid_blocking_placement",
		fmt.Sprintf("P%d unit %d (unit_const=%d) is placed at (%.3f, %.3f), outside the tile-center grid", player, unit.Index, unit.UnitConst, unit.X, unit.Y),
		"Blocking buildings are sensitive to fractional placement and can create seams or pathing surprises when their authored footprint is not tile aligned.",
		"Use the writer default grid placement, or specify snap=grid explicitly. Use snap=fractional only for footprint-less decorative Gaia art.")
}

// lintGaiaOnlyUnits is deliberately a thin consumer of the DAT cache. The
// cache is the source of truth for civ/unit presence and enabled state; this
// linter must not grow a second hand-maintained Gaia-unit table.
func lintGaiaOnlyUnits(report *LintReport, f *File, opts LintOptions) {
	if opts.DatPath == "" {
		report.addIssue("warning", "gaia_only_unit_rule_skipped", "DAT-backed Gaia-only unit checks skipped: no --dat was supplied")
		return
	}
	cache, err := datcache.Open(opts.DatPath)
	if err != nil {
		report.addIssue("warning", "gaia_only_unit_rule_skipped", fmt.Sprintf("DAT-backed Gaia-only unit checks skipped: open DAT: %v", err))
		return
	}
	defer cache.Close()

	check := func(unitID, owner int, where string) {
		if owner == 0 || unitID <= 0 {
			return
		}
		rows, _, queryErr := cache.QueryUnits(datcache.UnitQuery{ID: &unitID, All: true})
		if queryErr != nil {
			report.addIssue("warning", "gaia_only_unit_rule_skipped", fmt.Sprintf("DAT-backed Gaia-only check for unit %d at %s failed: %v", unitID, where, queryErr))
			return
		}
		if len(rows) == 0 {
			report.addIssue("warning", "gaia_unit_unknown", fmt.Sprintf("unit %d at %s is absent from the DAT cache; cannot verify its civ availability", unitID, where))
			return
		}
		gaia, nonGaia, name, disabled, enabledNonGaia := false, false, "", false, false
		for _, row := range rows {
			if name == "" {
				name = row.Name
			}
			if row.CivIndex == 0 {
				gaia = true
			} else {
				nonGaia = true
				if row.Enabled == 0 {
					disabled = true
				} else {
					enabledNonGaia = true
				}
			}
		}
		if gaia && !nonGaia {
			report.addIssueDetail("error", "gaia_only_unit_for_player",
				fmt.Sprintf("Gaia-only unit %d %s is created for player %d at %s", unitID, name, owner, where),
				"The DAT contains this unit only in civ 0 (Gaia); DE may reject or fail to render it when created under a player owner.",
				"Use owner 0 for Gaia eyecandy, or use a unit present in the target player's civ table.")
			return
		}
		if disabled && !enabledNonGaia {
			report.addIssueDetail("warning", "disabled_unit_for_player",
				fmt.Sprintf("unit %d %s has disabled rows for a player civ and is created for player %d at %s", unitID, name, owner, where),
				"The unit exists in the DAT but is disabled for at least one non-Gaia civ; availability and creation behavior can differ.",
				"Check the assigned civ or enable the unit in the DAT before relying on player-owned creation.")
		}
	}

	if f.Units != nil {
		for _, player := range f.Units.Sections {
			for _, unit := range player.Units {
				check(unit.UnitConst, player.Player, fmt.Sprintf("scenario P%d unit[%d]", player.Player, unit.Index))
			}
		}
	}
	if f.root == nil {
		return
	}
	if section := f.root.section("Triggers"); section != nil {
		for triggerIndex, trigger := range section.list("trigger_data") {
			for effectIndex, effect := range trigger.list("effect_data") {
				effectType, typeOK := effect.intValue("effect_type")
				if !typeOK || effectType != 11 { // create_object
					continue
				}
				unitID, unitOK := effect.intValue("object_list_unit_id")
				owner, ownerOK := effect.intValue("source_player")
				if unitOK && ownerOK {
					check(unitID, owner, fmt.Sprintf("trigger %d effect %d", triggerIndex, effectIndex))
				}
			}
		}
	}
}

func offGridCoordinate(value float64) bool {
	frac := value - math.Floor(value)
	return math.Abs(frac-0.5) > 0.0001
}

func likelyBlockingUnit(unitConst int) bool {
	switch unitConst {
	case 12, 49, 82, 87, 101, 103, 104, 109, 598:
		return true
	default:
		return false
	}
}

type trainButtonKey struct {
	player   int
	building int
	slot     int
}

type trainButtonState struct {
	count       int
	trigger     int
	effect      int
	sourceUnits []int
}

func lintLoadSafety(report *LintReport, f *File, opts LintOptions) {
	if f == nil {
		return
	}
	activeNonGaia := activeNonGaiaPlayers(f.Players)
	for _, player := range activeNonGaia {
		if !playerHasStartingBuilding(f, player) {
			report.addCheckpointIssueDetail("C4", "warning", "load_safety_missing_starting_building",
				fmt.Sprintf("P%d is active without a recognized starting production building", player),
				"Diagnostic runs with outpost-only starters have repeatedly required repair. The exact engine cause is not established; this stock-ID check cannot classify custom DAT buildings.",
				"Place a Barracks Age1 (unit 12) for this player away from the test area. An outpost (598) does not satisfy this diagnostic convention; verify custom-building alternatives in engine.")
		}
	}
	if f.PlayerCount > 8 {
		report.addCheckpointIssueFactDetail(
			"C0",
			"error",
			"load_safety_too_many_players",
			fmt.Sprintf("scenario player_count=%d exceeds the DE hard cap of 8 non-Gaia players", f.PlayerCount),
			"DE supports at most 8 non-Gaia player slots plus Gaia.",
			"Set scenario player_count to 1..8 and keep only player slots 1..8 active.",
			"scenario.de_max_8_non_gaia_players",
			opts,
		)
	}
	if len(activeNonGaia) > 8 {
		report.addCheckpointIssueFactDetail(
			"C0",
			"error",
			"load_safety_too_many_active_players",
			fmt.Sprintf("scenario has %d active non-Gaia players: %v", len(activeNonGaia), activeNonGaia),
			"DE supports at most 8 non-Gaia player slots plus Gaia.",
			"Deactivate non-Gaia slots above the intended playable player count.",
			"scenario.de_max_8_non_gaia_players",
			opts,
		)
	}
	if f.PlayerCount != len(activeNonGaia) {
		report.addCheckpointIssueFactDetail(
			"C0",
			"error",
			"load_safety_player_count_mismatch",
			fmt.Sprintf("scenario player_count=%d but active non-Gaia player count=%d (%v)", f.PlayerCount, len(activeNonGaia), activeNonGaia),
			"DE can reject scenario loading when header player_count disagrees with active playable slots.",
			"Set scenario player_count to the active non-Gaia player count.",
			"scenario.player_count_must_match_active_non_gaia_count",
			opts,
		)
	}
	if f.Version == "1.58" && f.PlayerCount == 8 && len(activeNonGaia) == 8 {
		report.addCheckpointIssueFactDetail(
			"C0",
			"error",
			"load_safety_legacy_eight_player_preview_layout",
			"scenario uses the legacy 1.58 preview layout with all 8 player slots active",
			"A real DE 101.103.54800.0 C0 load failure was reproduced for this combination before the editor-authored 1.59 eight-player base was used. The failure occurs during scenario-list preview, before lobby or XS compilation.",
			"Re-save an eight-player editor-authored base in the current scenario format before patching it, or use a current 1.59 base.",
			"scenario.c0_preview_requires_current_eight_player_base",
			opts,
		)
	}
	for _, player := range f.Players {
		if player.Player < 0 || player.Player >= 8 || !player.Active || !player.Human || !player.LockCivilization {
			continue
		}
		report.addCheckpointIssueFactDetail(
			"C1",
			"warning",
			"load_safety_locked_civ_launch_risk",
			fmt.Sprintf("P%d is active+human with lock_civilization=true", player.Player+1),
			"DE multiplayer lobbies can refuse to launch when open human slots with locked civilizations are unclaimed.",
			"Use lock_civilization=false for open human slots, or make the slot non-human/closed if it must stay locked.",
			"scenario.locked_civ_open_human_slot_blocks_mp_launch",
			opts,
		)
	}
	if f.root != nil {
		if section := f.root.section("Triggers"); section != nil {
			lintTrainButtonLoadSafety(report, section.list("trigger_data"), opts)
		}
	}
}

// playerHasStartingBuilding checks authored scenario units only. Trigger-spawned
// buildings cannot satisfy DE's start-of-scenario defeat/keep-alive check.
// This is a conservative stock-ID allowlist, not a DAT classifier.
func playerHasStartingBuilding(f *File, player int) bool {
	if f == nil || f.Units == nil {
		return false
	}
	for _, section := range f.Units.Sections {
		if section.Player != player {
			continue
		}
		for _, unit := range section.Units {
			switch unit.UnitConst {
			case 12, 49, 82, 87, 101, 103, 104, 109:
				return true
			}
		}
	}
	return false
}

func activeNonGaiaPlayers(players []PlayerInfo) []int {
	var out []int
	for _, player := range players {
		// PlayerInfo uses editor slots 0..7 for P1..P8; unlike UnitInfo,
		// slot zero is not Gaia. Expose human-facing player numbers in diagnostics.
		if player.Player >= 0 && player.Player < 8 && player.Active {
			out = append(out, player.Player+1)
		}
	}
	sort.Ints(out)
	return out
}

func lintTrainButtonLoadSafety(report *LintReport, triggerNodes []*parsedNode, opts LintOptions) {
	byKey := map[trainButtonKey]*trainButtonState{}
	perBuilding := map[int]int{}
	for i, trigger := range triggerNodes {
		for j, effect := range trigger.list("effect_data") {
			effectType, _ := effect.intValue("effect_type")
			if effectType != 102 {
				continue
			}
			building, buildingOK := effect.intValue("object_list_unit_id_2")
			slot, slotOK := effect.intValue("button_location")
			sourceUnit, _ := effect.intValue("object_list_unit_id")
			if !buildingOK || !slotOK {
				continue
			}
			if slot > 16 {
				report.addCheckpointIssueFactDetail(
					"C4",
					"warning",
					"load_safety_train_button_overflow",
					fmt.Sprintf("trigger %d effect %d add_train_location targets building %d button_location=%d", i, j, building, slot),
					"DE command cards expose 16 train-button positions; higher locations may not render as intended.",
					"Use button_location 1..16 and split larger shops across multiple buildings.",
					"scenario.train_button_command_card_has_16_slots",
					opts,
				)
			}
			player, _ := effect.intValue("source_player")
			key := trainButtonKey{player: player, building: building, slot: slot}
			state := byKey[key]
			if state == nil {
				state = &trainButtonState{trigger: i, effect: j}
				byKey[key] = state
			}
			state.count++
			state.sourceUnits = append(state.sourceUnits, sourceUnit)
			perBuilding[building]++
		}
	}
	for key, state := range byKey {
		if state.count <= 1 {
			continue
		}
		report.addCheckpointIssueFactDetail(
			"C4",
			"error",
			"load_safety_train_button_slot_collision",
			fmt.Sprintf("%d add_train_location effects share player %d building %d button_location=%d; first seen at trigger %d effect %d; unit buttons=%v", state.count, key.player, key.building, key.slot, state.trigger, state.effect, state.sourceUnits),
			"DE can corrupt the building command card when more than one add_train_location writes the same building/button slot, causing buttons to disappear in-engine.",
			"Use exactly one add_train_location per (source_player, train_location_unit_const, button_location); multi-player shops need separate per-player buildings or distinct slots.",
			"scenario.add_train_location_unique_building_slot",
			opts,
		)
	}
	for building, count := range perBuilding {
		if count <= 16 {
			continue
		}
		report.addCheckpointIssueFactDetail(
			"C4",
			"warning",
			"load_safety_train_button_overflow",
			fmt.Sprintf("building %d has %d add_train_location effects", building, count),
			"DE command cards expose 16 train-button positions per building.",
			"Split shops with more than 16 train buttons across multiple buildings.",
			"scenario.train_button_command_card_has_16_slots",
			opts,
		)
	}
}

func lintTriggerPlayerCoverage(report *LintReport, file *File, _ LintOptions) {
	if file == nil {
		return
	}
	var players []int
	for _, player := range file.Players {
		if player.Player > 0 && player.Active {
			players = append(players, player.Player)
		}
	}
	if len(players) < 2 {
		return
	}
	rows := collectTriggerRows(file)
	type familyState struct {
		players map[int]bool
		samples []TriggerBrief
	}
	families := map[string]*familyState{}
	for _, row := range rows {
		key, namedPlayer := playerFamilyKey(row.Name)
		if key == "" {
			continue
		}
		touched := rowTouchedPlayers(row)
		if namedPlayer > 0 {
			touched[namedPlayer] = true
		}
		if len(touched) == 0 {
			continue
		}
		state := families[key]
		if state == nil {
			state = &familyState{players: map[int]bool{}}
			families[key] = state
		}
		for player := range touched {
			if containsInt(players, player) {
				state.players[player] = true
			}
		}
		if len(state.samples) < 3 {
			state.samples = append(state.samples, TriggerBrief{Index: row.Index, Name: row.Name, Enabled: row.Enabled, Looping: row.Looping})
		}
	}
	for family, state := range families {
		if len(state.players) == 0 || len(state.players) == len(players) {
			continue
		}
		var missing []int
		for _, player := range players {
			if !state.players[player] {
				missing = append(missing, player)
			}
		}
		report.addIssue("warning", "trigger_player_family_asymmetry", fmt.Sprintf("trigger family %q touches players %v but is missing %v; this can indicate a per-slot refresh/transition bug", family, sortedBoolKeys(state.players), missing))
	}
}

func lintActiveComputerStarterInjection(report *LintReport, players []PlayerInfo, unitCountsByPlayer map[int]int, opts LintOptions) {
	for _, player := range players {
		if player.Player < 0 || player.Player >= 8 || !player.Active || player.Human {
			continue
		}
		if unitCountsByPlayer[player.Player+1] == 0 {
			report.addIssueFact("warning", "active_computer_without_authored_units", fmt.Sprintf("P%d is active non-human with no authored units; DE scenario-mode runs may inject starter TC/villager/scout state and contaminate replay diagnostics", player.Player+1), "scenario.active_computer_starter_injection", opts)
		}
	}
}

func lintTriggerRuntimeHazards(report *LintReport, triggerNodes []*parsedNode, opts LintOptions) {
	for i, trigger := range triggerNodes {
		looping, _ := trigger.intValue("looping")
		name, _ := trigger.stringValue("trigger_name")
		for j, effect := range trigger.list("effect_data") {
			effectType, _ := effect.intValue("effect_type")
			if effectType == 51 {
				if operation, ok := effect.intValue("operation"); ok && (operation < 1 || operation > 5) {
					report.addIssueDetail("error", "modify_attribute_invalid_operation",
						fmt.Sprintf("trigger %d %q effect %d has modify-attribute operation %d; valid trigger operations are SET=1, ADD=2, SUBTRACT=3, MULTIPLY=4, DIVIDE=5", i, name, j, operation),
						"The trigger effect enum is different from the XS attribute constants; operation 0 is a no-op here and values outside 1..5 are invalid.",
						"Use operation_name=set|add|subtract|multiply|divide in a Kit recipe, or write the trigger operation as an integer from 1 through 5.")
				}
			}
			if looping != 0 && effectType == 20 {
				report.addIssueFact("error", "looping_display_effect", fmt.Sprintf("trigger %d %q is looping and effect %d is display-instructions; timer conditions become permanently true after elapsed and can spam text every tick", i, name, j), "scenario.looping_display_effect_spams", opts)
			}
			if text, ok := effect.stringValue("message"); ok && text != "" {
				lintTriggerTextMarkup(report, i, name, j, effectType, text, opts)
			}
		}
	}
}

func lintTriggerTextMarkup(report *LintReport, triggerIndex int, triggerName string, effectIndex int, effectType int, text string, opts LintOptions) {
	if !isTriggerTextChannel(effectType) {
		return
	}
	if strings.Contains(strings.ToUpper(text), "<WHITE>") {
		report.addIssueFact("warning", "unsupported_white_color_tag", fmt.Sprintf("trigger %d %q effect %d contains <WHITE>; DE trigger text color markup does not support that tag", triggerIndex, triggerName, effectIndex), "text.white_tag_not_supported", opts)
	}
	colorTags := triggerColorTagPositions(text)
	if len(colorTags) == 0 {
		return
	}
	if len(colorTags) > 1 || colorTags[0] != 0 {
		report.addIssueFact("warning", "trigger_color_tag_not_leading_singleton", fmt.Sprintf("trigger %d %q effect %d has %d color tags and first tag offset %d; DE trigger text applies only one leading color tag", triggerIndex, triggerName, effectIndex, len(colorTags), colorTags[0]), "text.trigger_color_first_tag_only", opts)
	}
}

func isTriggerTextChannel(effectType int) bool {
	switch effectType {
	case 3, 20, 37, 44, 55, 65, 66, 88:
		return true
	default:
		return false
	}
}

func triggerColorTagPositions(text string) []int {
	upper := strings.ToUpper(text)
	var positions []int
	for _, tag := range []string{"<RED>", "<GREEN>", "<YELLOW>", "<BLUE>", "<AQUA>", "<PURPLE>", "<ORANGE>", "<GREY>", "<GRAY>", "<WHITE>"} {
		start := 0
		for {
			idx := strings.Index(upper[start:], tag)
			if idx < 0 {
				break
			}
			positions = append(positions, start+idx)
			start += idx + len(tag)
		}
	}
	sort.Ints(positions)
	return positions
}

func lintCorruptStrings(report *LintReport, roots ...*parsedNode) {
	for _, root := range roots {
		lintCorruptStringNode(report, root, rootName(root))
	}
}

func rootName(node *parsedNode) string {
	if node == nil || node.Name == "" {
		return "root"
	}
	return node.Name
}

func lintCorruptStringNode(report *LintReport, node *parsedNode, path string) {
	if node == nil {
		return
	}
	if text, ok := node.Value.(string); ok {
		lintStringValue(report, path, node, text)
	}
	for _, field := range node.Fields {
		childPath := field.Name
		if path != "" {
			childPath = path + "." + field.Name
		}
		lintCorruptStringNode(report, field, childPath)
	}
	for i, elem := range node.Elements {
		childPath := fmt.Sprintf("%s[%d]", path, i)
		if elem.Name != "" {
			childPath = fmt.Sprintf("%s.%s", childPath, elem.Name)
		}
		lintCorruptStringNode(report, elem, childPath)
	}
}

func lintStringValue(report *LintReport, path string, node *parsedNode, text string) {
	if !utf8.ValidString(text) {
		report.addIssue("error", "corrupt_string_invalid_utf8", fmt.Sprintf("%s contains invalid UTF-8 at scenario byte offset %d", path, node.Start))
	}
	for offset, r := range text {
		if r < 0x20 && r != '\n' && r != '\r' && r != '\t' {
			report.addIssue("error", "corrupt_string_control_byte", fmt.Sprintf("%s contains control byte 0x%02x at string offset %d scenario byte offset %d", path, r, offset, node.Start+stringPayloadPrefixLen(node)+offset))
			break
		}
	}
	payload, payloadOffset, ok := stringPayloadBytes(node)
	if ok {
		for i, b := range payload {
			if b == 0 && i < len(payload)-1 {
				report.addIssue("error", "corrupt_string_embedded_nul", fmt.Sprintf("%s contains NUL before declared string end at payload offset %d scenario byte offset %d", path, i, node.Start+payloadOffset+i))
				break
			}
		}
	}
}

func stringPayloadPrefixLen(node *parsedNode) int {
	_, prefix, ok := stringPayloadBytes(node)
	if !ok {
		return 0
	}
	return prefix
}

func stringPayloadBytes(node *parsedNode) ([]byte, int, bool) {
	raw := node.Raw
	for _, prefixSize := range []int{2, 4} {
		if len(raw) < prefixSize {
			continue
		}
		length := signedInt(raw[:prefixSize])
		if length >= 0 && prefixSize+length == len(raw) {
			return raw[prefixSize:], prefixSize, true
		}
	}
	return nil, 0, false
}

func (r *LintReport) addIssue(severity, code, message string) {
	r.Issues = append(r.Issues, LintIssue{Severity: severity, Code: code, Message: message})
}

func (r *LintReport) addIssueDetail(severity, code, message, why, fix string) {
	r.Issues = append(r.Issues, LintIssue{Severity: severity, Code: code, Message: message, Why: why, Fix: fix})
}

func (r *LintReport) addCheckpointIssueDetail(checkpoint, severity, code, message, why, fix string) {
	r.Issues = append(r.Issues, LintIssue{Severity: severity, Code: code, Checkpoint: checkpoint, Message: message, Why: why, Fix: fix})
}

func (r *LintReport) addIssueFact(severity, code, message, factID string, opts LintOptions) {
	issue := LintIssue{Severity: severity, Code: code, Message: message}
	citation := enginefacts.CitationFor(factID, opts.IncludeProvisional)
	issue.FactID = citation.FactID
	issue.FactTier = citation.Tier
	issue.VerifiedDate = citation.VerifiedDate
	issue.FixtureRef = citation.FixtureRef
	r.Issues = append(r.Issues, issue)
}

func (r *LintReport) addIssueFactDetail(severity, code, message, why, fix, factID string, opts LintOptions) {
	issue := LintIssue{Severity: severity, Code: code, Message: message, Why: why, Fix: fix}
	citation := enginefacts.CitationFor(factID, opts.IncludeProvisional)
	issue.FactID = citation.FactID
	issue.FactTier = citation.Tier
	issue.VerifiedDate = citation.VerifiedDate
	issue.FixtureRef = citation.FixtureRef
	r.Issues = append(r.Issues, issue)
}

func (r *LintReport) addCheckpointIssueFactDetail(checkpoint, severity, code, message, why, fix, factID string, opts LintOptions) {
	issue := LintIssue{Severity: severity, Code: code, Checkpoint: checkpoint, Message: message, Why: why, Fix: fix}
	citation := enginefacts.CitationFor(factID, opts.IncludeProvisional)
	issue.FactID = citation.FactID
	issue.FactTier = citation.Tier
	issue.VerifiedDate = citation.VerifiedDate
	issue.FixtureRef = citation.FixtureRef
	r.Issues = append(r.Issues, issue)
}

func hasLintSeverity(issues []LintIssue, severity string) bool {
	for _, issue := range issues {
		if issue.Severity == severity {
			return true
		}
	}
	return false
}

func sortedTerrainIDs(a, b map[int]int) []int {
	seen := map[int]bool{}
	for k := range a {
		seen[k] = true
	}
	for k := range b {
		seen[k] = true
	}
	out := make([]int, 0, len(seen))
	for k := range seen {
		out = append(out, k)
	}
	sort.Ints(out)
	return out
}

func sortedIntKeys(a, b map[int]int) []int {
	seen := map[int]bool{}
	for k := range a {
		seen[k] = true
	}
	for k := range b {
		seen[k] = true
	}
	out := make([]int, 0, len(seen))
	for k := range seen {
		out = append(out, k)
	}
	sort.Ints(out)
	return out
}

func unitCountsByPlayer(info *UnitInfo) map[int]int {
	out := map[int]int{}
	if info == nil {
		return out
	}
	for _, player := range info.Sections {
		out[player.Player] = len(player.Units)
	}
	return out
}

const unitFloatDiffEpsilon = 0.0001

func diffUnitsByReference(before, after *UnitInfo, add func(kind, field string, b, a any, detail string)) {
	beforeByPlayer := unitsByPlayerAndReference(before)
	afterByPlayer := unitsByPlayerAndReference(after)
	for _, player := range sortedUnitPlayers(beforeByPlayer, afterByPlayer) {
		beforeUnits := beforeByPlayer[player]
		afterUnits := afterByPlayer[player]
		prefix := fmt.Sprintf("units/player_%d", player)
		for _, refID := range sortedUnitRefs(beforeUnits, afterUnits) {
			b, bOK := beforeUnits[refID]
			a, aOK := afterUnits[refID]
			switch {
			case !bOK && aOK:
				add("units", prefix+".added", nil, unitDiffSummary(a), fmt.Sprintf("unit reference_id %d added", refID))
			case bOK && !aOK:
				add("units", prefix+".removed", unitDiffSummary(b), nil, fmt.Sprintf("unit reference_id %d removed", refID))
			default:
				diffUnitSummary(prefix, b, a, add)
			}
		}
	}
}

func unitsByPlayerAndReference(info *UnitInfo) map[int]map[int]UnitSummary {
	out := map[int]map[int]UnitSummary{}
	if info == nil {
		return out
	}
	for _, section := range info.Sections {
		byRef := out[section.Player]
		if byRef == nil {
			byRef = map[int]UnitSummary{}
			out[section.Player] = byRef
		}
		for _, unit := range section.Units {
			if unit.ReferenceID == 0 {
				continue
			}
			if _, exists := byRef[unit.ReferenceID]; exists {
				continue
			}
			byRef[unit.ReferenceID] = unit
		}
	}
	return out
}

func sortedUnitPlayers(a, b map[int]map[int]UnitSummary) []int {
	seen := map[int]bool{}
	for player := range a {
		seen[player] = true
	}
	for player := range b {
		seen[player] = true
	}
	out := make([]int, 0, len(seen))
	for player := range seen {
		out = append(out, player)
	}
	sort.Ints(out)
	return out
}

func sortedUnitRefs(a, b map[int]UnitSummary) []int {
	seen := map[int]bool{}
	for refID := range a {
		seen[refID] = true
	}
	for refID := range b {
		seen[refID] = true
	}
	out := make([]int, 0, len(seen))
	for refID := range seen {
		out = append(out, refID)
	}
	sort.Ints(out)
	return out
}

func diffUnitSummary(prefix string, before, after UnitSummary, add func(kind, field string, b, a any, detail string)) {
	unitPrefix := fmt.Sprintf("%s.unit_%d.", prefix, before.ReferenceID)
	compareInt := func(field string, b, a int) {
		if b != a {
			add("units", unitPrefix+field, b, a, "")
		}
	}
	compareFloat := func(field string, b, a float64) {
		if math.Abs(b-a) > unitFloatDiffEpsilon {
			add("units", unitPrefix+field, b, a, "")
		}
	}
	compareInt("unit_const", before.UnitConst, after.UnitConst)
	compareFloat("x", before.X, after.X)
	compareFloat("y", before.Y, after.Y)
	compareFloat("z", before.Z, after.Z)
	compareFloat("rotation", before.Rotation, after.Rotation)
	compareInt("status", before.Status, after.Status)
	compareInt("caption_string_id", before.CaptionStringID, after.CaptionStringID)
}

func unitDiffSummary(unit UnitSummary) UnitDiffSummary {
	return UnitDiffSummary{
		ReferenceID: unit.ReferenceID,
		UnitConst:   unit.UnitConst,
		X:           unit.X,
		Y:           unit.Y,
	}
}

func diffTriggerSummaries(before, after []TriggerSummary, add func(kind, field string, b, a any, detail string)) {
	limit := len(before)
	if len(after) < limit {
		limit = len(after)
	}
	for i := 0; i < limit; i++ {
		prefix := fmt.Sprintf("trigger_%d", i)
		if before[i].Name != after[i].Name {
			add("triggers", prefix+".name", before[i].Name, after[i].Name, "")
		}
		if before[i].Enabled != after[i].Enabled {
			add("triggers", prefix+".enabled", before[i].Enabled, after[i].Enabled, before[i].Name)
		}
		if before[i].Looping != after[i].Looping {
			add("triggers", prefix+".looping", before[i].Looping, after[i].Looping, before[i].Name)
		}
		if before[i].ExecuteOnLoad != after[i].ExecuteOnLoad {
			add("triggers", prefix+".execute_on_load", before[i].ExecuteOnLoad, after[i].ExecuteOnLoad, before[i].Name)
		}
		if before[i].DescriptionStringTableID != after[i].DescriptionStringTableID {
			add("triggers", prefix+".description_string_table_id", before[i].DescriptionStringTableID, after[i].DescriptionStringTableID, before[i].Name)
		}
		if before[i].DisplayAsObjective != after[i].DisplayAsObjective {
			add("triggers", prefix+".display_as_objective", before[i].DisplayAsObjective, after[i].DisplayAsObjective, before[i].Name)
		}
		if before[i].ObjectiveDescriptionOrder != after[i].ObjectiveDescriptionOrder {
			add("triggers", prefix+".objective_description_order", before[i].ObjectiveDescriptionOrder, after[i].ObjectiveDescriptionOrder, before[i].Name)
		}
		if before[i].MakeHeader != after[i].MakeHeader {
			add("triggers", prefix+".make_header", before[i].MakeHeader, after[i].MakeHeader, before[i].Name)
		}
		if before[i].ShortDescriptionStringTableID != after[i].ShortDescriptionStringTableID {
			add("triggers", prefix+".short_description_string_table_id", before[i].ShortDescriptionStringTableID, after[i].ShortDescriptionStringTableID, before[i].Name)
		}
		if before[i].DisplayOnScreen != after[i].DisplayOnScreen {
			add("triggers", prefix+".display_on_screen", before[i].DisplayOnScreen, after[i].DisplayOnScreen, before[i].Name)
		}
		if before[i].MuteObjectives != after[i].MuteObjectives {
			add("triggers", prefix+".mute_objectives", before[i].MuteObjectives, after[i].MuteObjectives, before[i].Name)
		}
		if before[i].Effects != after[i].Effects {
			add("triggers", prefix+".effects", before[i].Effects, after[i].Effects, before[i].Name)
		}
		if before[i].Conditions != after[i].Conditions {
			add("triggers", prefix+".conditions", before[i].Conditions, after[i].Conditions, before[i].Name)
		}
	}
	for i := limit; i < len(before); i++ {
		add("triggers", fmt.Sprintf("trigger_%d", i), before[i].Name, nil, "removed")
	}
	for i := limit; i < len(after); i++ {
		add("triggers", fmt.Sprintf("trigger_%d", i), nil, after[i].Name, "added")
	}
}

func diffPlayers(before, after []PlayerSettings, add func(kind, field string, b, a any, detail string)) {
	limit := len(before)
	if len(after) < limit {
		limit = len(after)
	}
	for i := 0; i < limit; i++ {
		diffPlayer(before[i], after[i], add)
	}
	if len(before) != len(after) {
		add("players", "count", len(before), len(after), "")
	}
}

func diffPlayer(before, after PlayerSettings, add func(kind, field string, b, a any, detail string)) {
	// Diff's intermediate key is the zero-based section index; decoration
	// converts it to the public P1..P8 label using PlayerLabel.
	prefix := fmt.Sprintf("player_%d.", before.SectionIndex)
	compare := func(field string, b, a any) {
		if b != a {
			add("players", prefix+field, b, a, "")
		}
	}
	compare("active", before.Active, after.Active)
	compare("human", before.Human, after.Human)
	compare("tribe_name", before.TribeName, after.TribeName)
	compare("name_string_id", before.NameStringID, after.NameStringID)
	compare("civilization", before.Civilization, after.Civilization)
	compare("architecture", before.Architecture, after.Architecture)
	compare("lock_civilization", before.LockCivilization, after.LockCivilization)
	compare("lock_personality", before.LockPersonality, after.LockPersonality)
	compare("starting_age", before.StartingAge, after.StartingAge)
	compare("color", before.Color, after.Color)
	compare("base_priority", before.BasePriority, after.BasePriority)
	compare("population_limit", before.PopulationLimit, after.PopulationLimit)
	compare("ai_name", before.AIName, after.AIName)
	compare("ai_type", before.AIType, after.AIType)
	compare("allied_victory", before.AlliedVictory, after.AlliedVictory)
	compare("resources.gold", before.Resources.Gold, after.Resources.Gold)
	compare("resources.wood", before.Resources.Wood, after.Resources.Wood)
	compare("resources.food", before.Resources.Food, after.Resources.Food)
	compare("resources.stone", before.Resources.Stone, after.Resources.Stone)
	compare("resources.trade_goods", before.Resources.TradeGoods, after.Resources.TradeGoods)
	if !intSlicesEqual(before.Diplomacy, after.Diplomacy) {
		add("players", prefix+"diplomacy", before.Diplomacy, after.Diplomacy, "")
	}
	if !intSlicesEqual(before.DisabledTechs, after.DisabledTechs) {
		add("players", prefix+"disabled_techs", before.DisabledTechs, after.DisabledTechs, "")
	}
	if !intSlicesEqual(before.DisabledUnits, after.DisabledUnits) {
		add("players", prefix+"disabled_units", before.DisabledUnits, after.DisabledUnits, "")
	}
	if !intSlicesEqual(before.DisabledBuildings, after.DisabledBuildings) {
		add("players", prefix+"disabled_buildings", before.DisabledBuildings, after.DisabledBuildings, "")
	}
}

func intSlicesEqual(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func diffOptions(before, after OptionsSettings, add func(kind, field string, b, a any, detail string)) {
	compare := func(field string, b, a any) {
		if b != a {
			add("settings", "options."+field, b, a, "")
		}
	}
	compare("full_tech_tree", before.FullTechTree, after.FullTechTree)
	compare("collide_and_correcting", before.CollideAndCorrecting, after.CollideAndCorrecting)
	compare("villager_force_drop", before.VillagerForceDrop, after.VillagerForceDrop)
	compare("lock_coop_alliances", before.LockCoopAlliances, after.LockCoopAlliances)
	compare("trigger_execution_order", before.TriggerExecutionOrder, after.TriggerExecutionOrder)
	if !playerViewsEqual(before.InitialPlayerViews, after.InitialPlayerViews) {
		add("settings", "options.initial_player_views", before.InitialPlayerViews, after.InitialPlayerViews, "")
	}
}

func diffDiplomacy(before, after DiplomacySettings, add func(kind, field string, b, a any, detail string)) {
	compare := func(field string, b, a any) {
		if b != a {
			add("settings", "diplomacy."+field, b, a, "")
		}
	}
	compare("lock_teams", before.LockTeams, after.LockTeams)
	compare("allow_players_choose_teams", before.AllowPlayersChooseTeams, after.AllowPlayersChooseTeams)
	compare("random_start_points", before.RandomStartPoints, after.RandomStartPoints)
	compare("max_number_of_teams", before.MaxNumberOfTeams, after.MaxNumberOfTeams)
}

func diffVictory(before, after map[string]int, add func(kind, field string, b, a any, detail string)) {
	for _, key := range sortedVictoryKeys(before, after) {
		b, bok := before[key]
		a, aok := after[key]
		if !bok || !aok || b != a {
			detail := ""
			if key == "mode" {
				detail = fmt.Sprintf("before=%s; after=%s", victoryModeName(b), victoryModeName(a))
			}
			add("settings", "victory."+key, valueOrNil(b, bok), valueOrNil(a, aok), detail)
		}
	}
}

func diffMessages(before, after MessagesSettings, add func(kind, field string, b, a any, detail string)) {
	diffMessageSlot("instructions", before.Instructions, after.Instructions, add)
	diffMessageSlot("hints", before.Hints, after.Hints, add)
	diffMessageSlot("victory", before.Victory, after.Victory, add)
	diffMessageSlot("loss", before.Loss, after.Loss, add)
	diffMessageSlot("history", before.History, after.History, add)
	diffMessageSlot("scouts", before.Scouts, after.Scouts, add)
}

func diffMessageSlot(name string, before, after MessageSlotSettings, add func(kind, field string, b, a any, detail string)) {
	prefix := "messages." + name + "."
	compare := func(field string, b, a any) {
		if b != a {
			add("settings", prefix+field, b, a, "")
		}
	}
	compare("literal", before.Literal, after.Literal)
	compare("string_table_id", before.StringTableID, after.StringTableID)
	compare("uses_string_table_id", before.UsesStringTableID, after.UsesStringTableID)
	compare("effective_source", before.EffectiveSource, after.EffectiveSource)
}

func diffCinematics(before, after CinematicsSettings, add func(kind, field string, b, a any, detail string)) {
	compare := func(field string, b, a string) {
		if b != a {
			add("settings", "cinematics."+field, b, a, "")
		}
	}
	compare("pregame", before.Pregame, after.Pregame)
	compare("victory", before.Victory, after.Victory)
	compare("loss", before.Loss, after.Loss)
}

func sortedVictoryKeys(a, b map[string]int) []string {
	seen := map[string]bool{}
	for key := range a {
		seen[key] = true
	}
	for key := range b {
		seen[key] = true
	}
	keys := make([]string, 0, len(seen))
	for key := range seen {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func valueOrNil(value int, ok bool) any {
	if !ok {
		return nil
	}
	return value
}

func playerViewsEqual(a, b []PlayerViewSettings) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func diffAI(before, after []AIFileInfo, add func(kind, field string, b, a any, detail string)) {
	beforeByName := map[string]AIFileInfo{}
	afterByName := map[string]AIFileInfo{}
	for _, ai := range before {
		beforeByName[ai.Name] = ai
	}
	for _, ai := range after {
		afterByName[ai.Name] = ai
	}
	names := map[string]bool{}
	for name := range beforeByName {
		names[name] = true
	}
	for name := range afterByName {
		names[name] = true
	}
	sorted := make([]string, 0, len(names))
	for name := range names {
		sorted = append(sorted, name)
	}
	sort.Strings(sorted)
	for _, name := range sorted {
		b, bok := beforeByName[name]
		a, aok := afterByName[name]
		switch {
		case !bok:
			add("ai", name, nil, a.ContentSHA256, "added")
		case !aok:
			add("ai", name, b.ContentSHA256, nil, "removed")
		case b.ContentSHA256 != a.ContentSHA256 || b.ContentBytes != a.ContentBytes:
			add("ai", name+".content", b.ContentSHA256, a.ContentSHA256, "")
		}
	}
}
