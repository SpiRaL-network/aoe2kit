package scenario

import (
	"fmt"
	"strings"
)

// lintSemanticInvariants checks authored interaction fixtures. These checks are
// opt-in because ordinary editor scenarios can intentionally leave slots empty
// or make a shop unaffordable during progression.
func lintSemanticInvariants(report *LintReport, f *File, opts LintOptions) {
	if f == nil || f.root == nil {
		return
	}
	settings := f.Settings()
	active := map[int]bool{}
	for _, player := range f.Players {
		if player.Active && player.Player >= 0 && player.Player < 8 {
			active[player.Player+1] = true
		}
	}
	for player := range active {
		if !playerHasStartingBuilding(f, player) {
			report.addIssueDetail("error", "semantic_interactive_fixture_missing_starting_building",
				fmt.Sprintf("active P%d has no authored starting keep-alive building", player),
				"DE defeat detection checks authored buildings at scenario start; trigger-spawned units and outposts do not provide a reliable diagnostic keep-alive.",
				"Place a recognized production building, typically a Barracks, for every active diagnostic player before the scenario starts.")
		}
	}
	lintInteractiveKeepAliveUnits(report, f, active, opts)
	// Generated producers may intentionally use enemy/neutral slots (shops and
	// multiplayer fixtures are examples). Their own builders own diplomacy;
	// apply this diagnostic-only heuristic to inspected authored fixtures.
	if !opts.Generated {
		lintInteractiveDiplomacy(report, settings, active, opts)
	}
	for i, player := range settings.Players {
		playerNumber := i + 1
		if active[playerNumber] {
			continue
		}
		if player.Resources.Food != 0 || player.Resources.Wood != 0 || player.Resources.Gold != 0 || player.Resources.Stone != 0 || player.Resources.TradeGoods != 0 || (player.StartingAge != 0 && player.StartingAge != 2) || len(player.DisabledTechs) != 0 || len(player.DisabledUnits) != 0 || len(player.DisabledBuildings) != 0 {
			report.addIssueDetail("error", "semantic_inactive_player_nondefault_data", fmt.Sprintf("inactive P%d carries non-default per-player settings", playerNumber), "Player-indexed data on an inactive slot is a common symptom of a one-based/zero-based list alignment error.", "Align per-player lists to the active player slots and clear inactive-slot values.")
		}
	}
	populatedActive := []int{}
	defaultActive := []int{}
	for i, player := range settings.Players {
		playerNumber := i + 1
		if !active[playerNumber] {
			continue
		}
		if hasNonDefaultPlayerData(player) {
			populatedActive = append(populatedActive, playerNumber)
		} else {
			defaultActive = append(defaultActive, playerNumber)
		}
	}
	if len(populatedActive) > 0 && len(defaultActive) > 0 {
		report.addIssueDetail("error", "semantic_active_player_data_alignment",
			fmt.Sprintf("active player data is split between populated slots %v and default slots %v", populatedActive, defaultActive),
			"A shifted or truncated per-player list can leave one active slot at its default while later active slots carry authored values.",
			"Write per-player values from the same zero-based active-slot list used by the scenario settings parser.")
	}

	section := f.root.section("Triggers")
	if section == nil {
		return
	}
	tokens := map[int]trainTokenSemantics{}
	for triggerIndex, trigger := range section.list("trigger_data") {
		for effectIndex, effect := range trigger.list("effect_data") {
			effectType, _ := effect.intValue("effect_type")
			switch effectType {
			case 102:
				unit, ok := effect.intValue("object_list_unit_id")
				if !ok {
					continue
				}
				player, _ := effect.intValue("source_player")
				token := tokens[unit]
				if token.players == nil {
					token.players = map[int]bool{}
				}
				token.players[player] = true
				token.buttons++
				token.firstTrigger, token.firstEffect = triggerIndex, effectIndex
				tokens[unit] = token
			case 40:
				unit, ok := effect.intValue("object_list_unit_id")
				if !ok {
					continue
				}
				token := tokens[unit]
				token.hasCost = true
				token.cost = append(token.cost, costsFromEffect(effect)...)
				tokens[unit] = token
			case 51:
				unit, ok := effect.intValue("object_list_unit_id")
				attribute, attrOK := effect.intValue("object_attributes")
				if !ok || !attrOK || attribute != 110 {
					continue
				}
				token := tokens[unit]
				quantity, _ := effect.intValue("quantity")
				if quantity == 0 {
					token.popFree = true
				}
				tokens[unit] = token
			}
		}
	}
	for unit, token := range tokens {
		if token.buttons == 0 {
			continue
		}
		if !token.hasCost {
			report.addIssueDetail("error", "semantic_train_token_missing_cost", fmt.Sprintf("train token unit %d has no change_object_cost effect", unit), "The engine can render a token button but cannot make its purchase affordable or explain its price when cost data is absent.", "Add a change_object_cost effect with explicit zeroes for unused resource slots.")
		}
		if !token.popFree {
			report.addIssueDetail("error", "semantic_train_token_not_pop_free", fmt.Sprintf("train token unit %d is not explicitly population-free", unit), "A renamed train-token shop consumes population unless attribute 110 is explicitly set to zero.", "Add modify_attribute object_attributes=110 quantity=0 for the token unit.")
		}
		for _, cost := range token.cost {
			if cost.resource < 0 || cost.quantity < 0 {
				report.addIssueDetail("error", "semantic_invalid_token_cost", fmt.Sprintf("train token unit %d has an undefined cost slot resource=%d quantity=%d", unit, cost.resource, cost.quantity), "DE treats -1 cost fields as undefined rather than as a free purchase.", "Write unused cost slots as resource 0 and quantity 0.")
			}
			if cost.resource == 4 {
				report.addIssueDetail("error", "semantic_token_population_cost", fmt.Sprintf("train token unit %d uses population as a purchase cost", unit), "Population cost defeats the token-shop pattern and can make a button fail with a pop-space message.", "Set attribute 110 to zero and use food, wood, gold, or stone for the purchase cost.")
			}
		}
	}
}

func lintInteractiveKeepAliveUnits(report *LintReport, f *File, active map[int]bool, opts LintOptions) {
	if f.Units == nil {
		return
	}
	severity := "warning"
	if opts.Generated {
		severity = "error"
	}
	for _, section := range f.Units.Sections {
		if !active[section.Player] {
			continue
		}
		for _, unit := range section.Units {
			if unit.UnitConst == 79 {
				report.addIssueDetail(severity, "semantic_interactive_fixture_attacking_keep_alive",
					fmt.Sprintf("active P%d uses Watch Tower 79 as a keep-alive building", section.Player),
					"A Watch Tower can attack nearby units and introduces tower damage into an interaction fixture that is meant to measure its own mechanic.",
					"Use a recognized non-attacking production building, typically Barracks 12, in a quiet corner pocket.")
			}
		}
	}
}

func lintInteractiveDiplomacy(report *LintReport, settings SettingsReport, active map[int]bool, opts LintOptions) {
	if len(active) < 2 {
		return
	}
	severity := "warning"
	if opts.Generated {
		severity = "error"
	}
	for from := range active {
		for to := range active {
			if from == to || from >= len(settings.Diplomacy.Matrix) || to >= len(settings.Diplomacy.Matrix[from]) {
				continue
			}
			if settings.Diplomacy.Matrix[from][to] != 0 {
				report.addIssueDetail(severity, "semantic_interactive_fixture_diplomacy_unconfigured",
					fmt.Sprintf("active P%d and P%d are not explicitly allied (stance %d)", from, to, settings.Diplomacy.Matrix[from][to]),
					"Unconfigured diplomacy can add player combat, targeting, and line-of-sight behavior to an interaction fixture.",
					"Set every active-player pair to allied stance 0 and keep shared LOS disabled unless the fixture needs it.")
				return
			}
		}
	}
}

// lintScriptCallTargets prevents the DE sentinel crash caused by a script_call
// effect whose message names prose, a missing function, or a parameterized
// invocation. Foreign scenarios get warnings because their embedded/deployed
// XS may be incomplete locally; generated output gets blocking errors.
func lintScriptCallTargets(report *LintReport, f *File, opts LintOptions) {
	if f == nil || f.root == nil {
		return
	}
	section := f.root.section("Triggers")
	if section == nil {
		return
	}
	severity := "warning"
	if opts.Generated {
		severity = "error"
	}
	census, err := f.XSCensus(XSCensusOptions{})
	if err != nil {
		report.addIssueDetail(severity, "semantic_script_call_symbol_table_unavailable",
			fmt.Sprintf("cannot verify script_call targets: %v", err),
			"DE resolves script_call names at runtime; an unavailable XS symbol table leaves the target unchecked.",
			"Embed or deploy the scenario XS and make the target a parameterless function before running it.")
		return
	}
	functions := make(map[string]string, len(census.Functions))
	for _, function := range census.Functions {
		functions[function.Name] = strings.TrimSpace(function.Params)
	}
	for triggerIndex, trigger := range section.list("trigger_data") {
		triggerName, _ := trigger.stringValue("trigger_name")
		for effectIndex, effect := range trigger.list("effect_data") {
			effectType, _ := effect.intValue("effect_type")
			if effectType != 55 {
				continue
			}
			message, _ := effect.stringValue("message")
			if isXSCarrierMessage(message) {
				continue
			}
			targets := extractXSCallNames(message)
			if len(targets) == 0 {
				targets = []string{strings.TrimSpace(message)}
			}
			for _, target := range targets {
				target = strings.TrimSpace(target)
				if target == "" {
					continue
				}
				params, found := functions[target]
				if !found {
					report.addIssueDetail(severity, "semantic_script_call_target_undefined",
						fmt.Sprintf("trigger %d %q effect %d script_call target %q is not defined in the scenario XS", triggerIndex, triggerName, effectIndex, target),
						"DE can resolve a missing script_call target to an invalid sentinel and crash when the effect fires.",
						"Use the exact case-sensitive name of an embedded/deployed parameterless XS function, and keep prose in a comment or message effect.")
					continue
				}
				if params != "" || scriptCallHasArguments(message, target) {
					report.addIssueDetail(severity, "semantic_script_call_parameters",
						fmt.Sprintf("trigger %d %q effect %d script_call target %q is not a parameterless call", triggerIndex, triggerName, effectIndex, target),
						"DE script_call is a no-argument function bridge; passing data or targeting a function with parameters is unsafe.",
						"Call a parameterless wrapper function and pass data through scenario state or trigger variables.")
				}
			}
		}
	}
}

func scriptCallHasArguments(message, target string) bool {
	start := strings.Index(message, target)
	if start < 0 {
		return false
	}
	rest := strings.TrimSpace(message[start+len(target):])
	if !strings.HasPrefix(rest, "(") {
		return true
	}
	end := strings.IndexByte(rest, ')')
	if end < 0 {
		return true
	}
	return strings.TrimSpace(rest[1:end]) != ""
}

func hasNonDefaultPlayerData(player PlayerSettings) bool {
	return player.Resources.Food != 0 || player.Resources.Wood != 0 || player.Resources.Gold != 0 || player.Resources.Stone != 0 || player.Resources.TradeGoods != 0 || (player.StartingAge != 0 && player.StartingAge != 2) || len(player.DisabledTechs) != 0 || len(player.DisabledUnits) != 0 || len(player.DisabledBuildings) != 0
}

type tokenCost struct{ resource, quantity int }
type trainTokenSemantics struct {
	players               map[int]bool
	buttons, firstTrigger int
	firstEffect           int
	hasCost, popFree      bool
	cost                  []tokenCost
}

func costsFromEffect(effect *parsedNode) []tokenCost {
	var out []tokenCost
	for i := 1; i <= 3; i++ {
		resource, resourceOK := effect.intValue(fmt.Sprintf("resource_%d", i))
		quantity, quantityOK := effect.intValue(fmt.Sprintf("resource_%d_quantity", i))
		if resourceOK || quantityOK {
			out = append(out, tokenCost{resource: resource, quantity: quantity})
		}
	}
	return out
}
