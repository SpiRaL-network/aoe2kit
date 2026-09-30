package main

import (
	"fmt"
	"os"
	"strings"

	"aoe2kit/pkg/cba"
	"aoe2kit/pkg/jsonout"
	"aoe2kit/pkg/replay"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "balance":
		runBalance(os.Args[2:])
	case "trigger-razes":
		runTriggerRazes(os.Args[2:])
	case "trigger-spawns":
		runTriggerSpawns(os.Args[2:])
	case "sidechannels":
		runSideChannels(os.Args[2:])
	case "phase-facts":
		runPhaseFacts(os.Args[2:])
	case "replay":
		runReplay(os.Args[2:])
	default:
		usage()
		os.Exit(2)
	}
}

func runSideChannels(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: cba sidechannels <file.aoe2record|replay.zip> [--text]")
		os.Exit(2)
	}
	path := args[0]
	textOut := false
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--text":
			textOut = true
		case "--json":
			textOut = false
		default:
			die("cba sidechannels", fmt.Errorf("unknown option %q", args[i]))
		}
	}
	report, err := cba.BuildSideChannels(path)
	if err != nil {
		die("cba sidechannels", err)
	}
	if textOut {
		printSideChannels(report)
	} else {
		printJSON(report)
	}
}

func runPhaseFacts(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: cba phase-facts <file.aoe2record|replay.zip> --dat <empires2_x2_p1.dat> [--text]")
		os.Exit(2)
	}
	path := args[0]
	datPath := ""
	textOut := false
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--dat":
			i++
			if i >= len(args) {
				die("cba phase-facts", fmt.Errorf("--dat requires a DAT path"))
			}
			datPath = args[i]
		case "--text":
			textOut = true
		case "--json":
			textOut = false
		default:
			die("cba phase-facts", fmt.Errorf("unknown option %q", args[i]))
		}
	}
	report, err := cba.BuildPhaseFacts(path, datPath)
	if err != nil {
		die("cba phase-facts", err)
	}
	if textOut {
		printPhaseFacts(report)
	} else {
		printJSON(report)
	}
}

func runTriggerRazes(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: cba trigger-razes <file.aoe2record|replay.zip> [--dat <empires2_x2_p1.dat>] [--text]")
		os.Exit(2)
	}
	path := args[0]
	datPath := ""
	textOut := false
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--dat":
			i++
			if i >= len(args) {
				die("cba trigger-razes", fmt.Errorf("--dat requires a DAT path"))
			}
			datPath = args[i]
		case "--text":
			textOut = true
		case "--json":
			textOut = false
		default:
			die("cba trigger-razes", fmt.Errorf("unknown option %q", args[i]))
		}
	}
	report, err := cba.BuildTriggerRazes(path, datPath)
	if err != nil {
		die("cba trigger-razes", err)
	}
	if textOut {
		printTriggerRazes(report)
	} else {
		printJSON(report)
	}
}

func runTriggerSpawns(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: cba trigger-spawns <file.aoe2record|replay.zip> [--dat <empires2_x2_p1.dat>] [--text]")
		os.Exit(2)
	}
	path := args[0]
	datPath := ""
	textOut := false
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--dat":
			i++
			if i >= len(args) {
				die("cba trigger-spawns", fmt.Errorf("--dat requires a DAT path"))
			}
			datPath = args[i]
		case "--text":
			textOut = true
		case "--json":
			textOut = false
		default:
			die("cba trigger-spawns", fmt.Errorf("unknown option %q", args[i]))
		}
	}
	report, err := cba.BuildTriggerSpawns(path, datPath)
	if err != nil {
		die("cba trigger-spawns", err)
	}
	if textOut {
		printTriggerSpawns(report)
	} else {
		printJSON(report)
	}
}

func runBalance(args []string) {
	textOut := false
	for _, arg := range args {
		switch arg {
		case "--text":
			textOut = true
		case "--json":
			textOut = false
		default:
			die("cba balance", fmt.Errorf("unknown option %q", arg))
		}
	}
	report := cba.BuildBalance()
	if textOut {
		printBalance(report)
	} else {
		printJSON(report)
	}
}

func runReplay(args []string) {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: cba replay <progression|razes|raze-pressure|phases|perf|doctrine> <file.aoe2record|replay.zip> [--dat <empires2_x2_p1.dat>] [--text]")
		os.Exit(2)
	}
	textOut := false
	datPath := ""
	for i := 2; i < len(args); i++ {
		switch args[i] {
		case "--dat":
			i++
			if i >= len(args) {
				die("cba replay", fmt.Errorf("--dat requires a DAT path"))
			}
			datPath = args[i]
		case "--text":
			textOut = true
		case "--json":
			textOut = false
		default:
			die("cba replay", fmt.Errorf("unknown option %q", args[i]))
		}
	}
	switch args[0] {
	case "progression":
		report, err := cba.BuildProgression(args[1])
		if err != nil {
			die("cba replay progression", err)
		}
		if textOut {
			printProgression(report)
		} else {
			printJSON(report)
		}
	case "razes":
		report, err := cba.BuildRazes(args[1])
		if err != nil {
			die("cba replay razes", err)
		}
		if textOut {
			printRazes(report)
		} else {
			printJSON(report)
		}
	case "raze-pressure":
		report, err := cba.BuildRazePressure(args[1])
		if err != nil {
			die("cba replay raze-pressure", err)
		}
		if textOut {
			printRazePressure(report)
		} else {
			printJSON(report)
		}
	case "phases":
		report, err := cba.BuildPhaseTimeline(args[1], datPath)
		if err != nil {
			die("cba replay phases", err)
		}
		if textOut {
			printPhaseTimeline(report)
		} else {
			printJSON(report)
		}
	case "perf":
		report, err := cba.BuildPerformance(args[1])
		if err != nil {
			die("cba replay perf", err)
		}
		if textOut {
			printPerformance(report)
		} else {
			printJSON(report)
		}
	case "doctrine":
		report, err := cba.BuildDoctrine(args[1])
		if err != nil {
			die("cba replay doctrine", err)
		}
		if textOut {
			printDoctrine(report)
		} else {
			printJSON(report)
		}
	default:
		fmt.Fprintln(os.Stderr, "usage: cba replay <progression|razes|raze-pressure|phases|perf|doctrine> <file.aoe2record|replay.zip> [--dat <empires2_x2_p1.dat>] [--text]")
		os.Exit(2)
	}
}

func printBalance(report *cba.BalanceReport) {
	fmt.Printf("method: %s\n", report.Method)
	fmt.Printf("verification: %s\n", report.Verification)
	fmt.Printf("summary: rows=%d civ_rows=%d versions=%d trigger_evidence=%d missing=%d\n",
		report.Summary.Rows, report.Summary.CivRows, report.Summary.VersionRows, report.Summary.TriggerEvidence, report.Summary.MissingRows)
	if len(report.Rows) > 0 {
		fmt.Println("rows:")
		for _, row := range report.Rows {
			fmt.Printf("- version=%s civ=%d/%s razes_to_vill=%s castle_kills=%s imp_kills=%s units=%s spawn_s=%s stable_elephants=%s battle_elephants=%s coverage=%s sources=%s",
				row.Version, row.CivID, row.CivName, intText(row.RazesToVillager), intText(row.CastleKills),
				intText(row.ImperialKills), intText(row.UnitCount), intText(row.SpawnSeconds),
				intText(row.StableElephants), intText(row.BattleElephants), row.Coverage, strings.Join(row.Sources, ","))
			if row.PreferredForV292 {
				fmt.Printf(" preferred_v292=true")
			}
			if row.Notes != "" {
				fmt.Printf(" note=%q", row.Notes)
			}
			fmt.Println()
		}
	}
	if len(report.Missing) > 0 {
		fmt.Println("missing:")
		for _, missing := range report.Missing {
			civ := "all_unmapped"
			if missing.CivName != "" {
				civ = fmt.Sprintf("%d/%s", missing.CivID, missing.CivName)
			}
			fmt.Printf("- version=%s civ=%s reason=%q\n", missing.Version, civ, missing.Reason)
		}
	}
	if len(report.Evidence) > 0 {
		fmt.Println("evidence:")
		for _, ev := range report.Evidence {
			fmt.Printf("- version=%s kind=%s attr=%d amounts=%v counts=%v confidence=%s source=%q",
				ev.Version, ev.Kind, ev.Attribute, ev.Amounts, ev.Counts, ev.Confidence, ev.Source)
			if ev.Notes != "" {
				fmt.Printf(" note=%q", ev.Notes)
			}
			fmt.Println()
		}
	}
	printWarnings(report.Warnings)
}

func printTriggerRazes(report *cba.TriggerRazeReport) {
	fmt.Printf("path: %s\n", report.Path)
	if report.DatPath != "" {
		fmt.Printf("dat: %s\n", report.DatPath)
	}
	fmt.Printf("method: %s\n", report.Method)
	fmt.Printf("verification: %s\n", report.Verification)
	fmt.Printf("summary: triggers=%d reward_rungs=%d activators=%d rows=%d resolved_civs=%d buckets=%d conflicts=%d unresolved_techs=%d mapuche=%d khmer=%d\n",
		report.Summary.TriggerCount, report.Summary.RewardRungTriggers, report.Summary.ActivatorTriggers,
		report.Summary.Rows, report.Summary.ResolvedCivs, report.Summary.Buckets, report.Summary.Conflicts,
		report.Summary.UnresolvedTechs, report.Summary.MapucheRazesToVillager, report.Summary.KhmerRazesToVillager)
	for _, bucket := range report.Buckets {
		names := strings.Join(bucket.CivNames, ", ")
		if names == "" {
			names = fmt.Sprintf("techs=%v", bucket.TechIDs)
		}
		fmt.Printf("- %d raze(s): %d tech refs, %d civs: %s\n",
			bucket.RazesToVillager, bucket.Count, len(bucket.CivIDs), names)
	}
	if len(report.Conflicts) > 0 {
		fmt.Println("conflicts:")
		for _, conflict := range report.Conflicts {
			fmt.Printf("- tech=%d values=%v\n", conflict.TechID, conflict.Values)
		}
	}
	printWarnings(report.Warnings)
}

func printTriggerSpawns(report *cba.TriggerSpawnReport) {
	fmt.Printf("path: %s\n", report.Path)
	if report.DatPath != "" {
		fmt.Printf("dat: %s\n", report.DatPath)
	}
	fmt.Printf("method: %s\n", report.Method)
	fmt.Printf("verification: %s\n", report.Verification)
	fmt.Printf("summary: triggers=%d seconds_triggers=%d count_triggers=%d rows=%d complete=%d resolved_civs=%d buckets=%d conflicts=%d unresolved_techs=%d mapuche=%d/%d khmer=%d/%d persians=%d/%d\n",
		report.Summary.TriggerCount, report.Summary.SpawnerSecondTriggers, report.Summary.SpawnCountTriggers,
		report.Summary.Rows, report.Summary.CompleteRows, report.Summary.ResolvedCivs, report.Summary.Buckets,
		report.Summary.Conflicts, report.Summary.UnresolvedTechs, report.Summary.MapucheSpawnUnitCount,
		report.Summary.MapucheSpawnSeconds, report.Summary.KhmerSpawnUnitCount, report.Summary.KhmerSpawnSeconds,
		report.Summary.PersiansSpawnUnitCount, report.Summary.PersiansSpawnSeconds)
	for _, bucket := range report.Buckets {
		names := strings.Join(bucket.CivNames, ", ")
		if names == "" {
			names = fmt.Sprintf("techs=%v", bucket.TechIDs)
		}
		fmt.Printf("- units=%s seconds=%s: %d tech refs, %d civs: %s\n",
			intText(bucket.SpawnUnitCount), intText(bucket.SpawnSeconds), bucket.Count, len(bucket.CivIDs), names)
	}
	if len(report.Conflicts) > 0 {
		fmt.Println("conflicts:")
		for _, conflict := range report.Conflicts {
			fmt.Printf("- tech=%d field=%s values=%v\n", conflict.TechID, conflict.Field, conflict.Values)
		}
	}
	printWarnings(report.Warnings)
}

func printPhaseFacts(report *cba.PhaseFactsReport) {
	fmt.Printf("path: %s\n", report.Path)
	if report.DatPath != "" {
		fmt.Printf("dat: %s\n", report.DatPath)
	}
	fmt.Printf("method: %s\n", report.Method)
	fmt.Printf("verification: %s\n", report.Verification)
	fmt.Printf("summary: triggers=%d castle_rungs=%d imperial_rungs=%d activators=%d groups=%d civs=%d defaults=%d raze_joined=%d lane_symmetry=%t unresolved_groups=%d multi_civ_groups=%d\n",
		report.Summary.TriggerCount, report.Summary.CastleRungs, report.Summary.ImperialRungs,
		report.Summary.ActivatorTriggers, report.Summary.ConditionGroups, report.Summary.CivFacts,
		report.Summary.DefaultInferred, report.Summary.RazeFactsJoined, report.Summary.LaneSymmetryOK,
		report.Summary.UnresolvedGroups, report.Summary.MultiCivGroups)
	for _, row := range report.Civs {
		fmt.Printf("- civ=%d/%s castle=%d imperial=%d razes=%d confidence=%s\n",
			row.CivID, row.CivName, row.CastleThreshold, row.ImperialThreshold, row.RazesToVillager, row.Confidence)
	}
	printWarnings(report.Warnings)
}

func printPhaseTimeline(report *cba.PhaseTimelineReport) {
	fmt.Printf("path: %s\n", report.Path)
	if report.DatPath != "" {
		fmt.Printf("dat: %s\n", report.DatPath)
	}
	fmt.Printf("method: %s\n", report.Method)
	fmt.Printf("verification: %s\n", report.Verification)
	fmt.Printf("summary: players=%d castle_observed=%d imperial_observed=%d phase_facts_civs=%d sync_deltas=%d duration=%s confidence=%s\n",
		report.Summary.Players, report.Summary.CastleObserved, report.Summary.ImperialObserved,
		report.Summary.PhaseFactsCivs, report.Summary.SyncDeltas, report.Summary.Duration, report.Summary.Confidence)
	if len(report.Players) > 0 {
		fmt.Println("players:")
		for _, row := range report.Players {
			fmt.Printf("- %s name=%q civ=%d/%s castle_kills=%d imperial_kills=%d razes_to_vill=%d early_spawn=%s confidence=%s\n",
				row.PlayerLabel, row.PlayerName, row.CivID, row.CivName, row.CastleThresholdKills,
				row.ImperialThresholdKills, row.RazesToVillager, unitText(row.EarlySpawnUnitID, row.EarlySpawnUnitName),
				row.EarlySpawnConfidence)
			if row.Castle != nil {
				printPhaseAnchor("  castle", row.Castle)
			}
			if row.Imperial != nil {
				printPhaseAnchor("  imperial", row.Imperial)
			}
			for _, warning := range row.DetectionWarnings {
				fmt.Printf("  note: %s\n", warning)
			}
		}
	}
	printWarnings(report.Warnings)
}

func printPhaseAnchor(prefix string, anchor *cba.PhaseAnchor) {
	if anchor == nil {
		return
	}
	unit := ""
	if anchor.UnitID != 0 {
		unit = fmt.Sprintf(" unit=%s", unitText(anchor.UnitID, anchor.UnitName))
	}
	fmt.Printf("%s: time=%s kill_anchor=%d signal=%s confidence=%s objects_delta=%d type_sum_delta=%d avg_type=%.2f%s evidence=%q\n",
		prefix, anchor.Time, anchor.KillAnchor, anchor.Signal, anchor.Confidence,
		anchor.ObjectCountDelta, anchor.UnitTypeSumDelta, anchor.AverageAddedTypeID, unit, anchor.Evidence)
}

func printSideChannels(report *cba.SideChannelReport) {
	fmt.Printf("path: %s\n", report.Path)
	fmt.Printf("method: %s\n", report.Method)
	fmt.Printf("verification: %s\n", report.Verification)
	fmt.Printf("graph: sha256=%s token=%q ladder=%t reason=%q\n",
		report.GraphSHA256, report.ScenarioToken, report.Scenario.Ladder, report.Scenario.Reason)
	if report.DataSet.Status != "" {
		fmt.Printf("data_set: status=%s active=%q source=%s confidence=%s\n",
			report.DataSet.Status, report.DataSet.ActiveDataSet, report.DataSet.Source, report.DataSet.Confidence)
	}
	fmt.Printf("summary: variant=%s triggers=%d effects=%d conditions=%d messages=%d modify_resource=%d change_variable=%d accumulate_attribute=%d scoreboard_rows=%d engine_attrs=%d resources=%d thresholds=%d\n",
		report.Summary.SideChannelVariant,
		report.Summary.TriggerCount, report.Summary.EffectCount, report.Summary.ConditionCount,
		report.Summary.MessageCount, report.Summary.ModifyResourceEffects, report.Summary.ChangeVariableEffects,
		report.Summary.AccumAttributeConds, report.Summary.ScoreboardRows, report.Summary.EngineAttrChannels,
		report.Summary.ResourcesWritten, report.Summary.ThresholdChannels)
	if len(report.Scoreboard) > 0 {
		fmt.Println("scoreboard_rows:")
		for _, row := range report.Scoreboard {
			fmt.Printf("- %s players=%v kills=%d/%d deaths=%d/%d razes=%d/%d confidence=%s\n",
				row.Label, row.Players, row.KillResourceA, row.KillResourceB,
				row.DeathResourceA, row.DeathResourceB, row.RazeResourceA,
				row.RazeResourceB, row.Confidence)
		}
	}
	if len(report.EngineAttrs) > 0 {
		fmt.Println("engine_attributes:")
		for _, row := range report.EngineAttrs {
			fmt.Printf("- attr=%d %s semantic=%s quantities=%v players=%v conditions=%d\n",
				row.Attribute, row.Name, row.Semantic, row.Quantities, row.Players, row.ConditionCount)
		}
	}
	if len(report.Resources) > 0 {
		fmt.Println("resources:")
		for _, row := range report.Resources {
			fmt.Printf("- res=%d family=%s writes=%d amounts=%v players=%v\n",
				row.Resource, row.Family, row.Count, row.Amounts, row.Players)
		}
	}
	if len(report.Thresholds) > 0 {
		fmt.Println("thresholds:")
		for _, row := range report.Thresholds {
			fmt.Printf("- kind=%s attr=%d players=%v values=%v", row.Kind, row.Attribute, row.Players, row.Values)
			if len(row.Messages) > 0 {
				fmt.Printf(" messages=%d", len(row.Messages))
			}
			fmt.Println()
		}
	}
	if len(report.Templates) > 0 {
		fmt.Println("templates:")
		for _, line := range report.Templates {
			fmt.Printf("- %s\n", strings.ReplaceAll(line, "\n", " / "))
		}
	}
	if len(report.Notes) > 0 {
		fmt.Println("notes:")
		for _, note := range report.Notes {
			fmt.Printf("- %s confidence=%s detail=%q\n", note.Name, note.Confidence, note.Detail)
		}
	}
	printWarnings(report.Warnings)
}

func printProgression(report *cba.ProgressionReport) {
	fmt.Printf("path: %s\n", report.Path)
	fmt.Printf("method: %s\n", report.Method)
	fmt.Printf("verification: %s\n", report.Verification)
	fmt.Printf("summary: production_events=%d research_events=%d players_with_production=%d players_with_research=%d feudal_flags=%d imperial_proxy_signals=%d\n",
		report.Summary.ProductionEvents, report.Summary.ResearchEvents, report.Summary.PlayersWithProduction, report.Summary.PlayersWithResearch, report.Summary.FeudalProductionFlags, report.Summary.ImperialProxySignals)
	if len(report.Players) > 0 {
		fmt.Println("players:")
		for _, player := range report.Players {
			firstProduction := "none"
			if player.FirstProductionTime != "" {
				firstProduction = fmt.Sprintf("%s %s", player.FirstProductionTime, player.FirstProductionUnit)
			}
			imperial := "none"
			if player.FirstImperialProxyAt != "" {
				imperial = fmt.Sprintf("%s %s", player.FirstImperialProxyAt, player.FirstImperialProxy)
			}
			feudal := "none"
			if player.FeudalProductionAt != "" {
				feudal = fmt.Sprintf("%s %s", player.FeudalProductionAt, player.FeudalProductionUnit)
			}
			fmt.Printf("- %s name=%q civ=%d/%s first_production=%s imperial_proxy=%s feudal_flag=%s unit_types=%d research=%d confidence=%s\n",
				player.Label, player.Name, player.CivID, player.CivName, firstProduction, imperial, feudal, player.ProductionUnitTypes, player.ResearchTechnologies, player.Confidence)
		}
	}
	if len(report.Production) > 0 {
		fmt.Println("production_first_seen:")
		limit := len(report.Production)
		if limit > 80 {
			limit = 80
		}
		for _, item := range report.Production[:limit] {
			signal := item.Signal
			if signal == "" {
				signal = "none"
			}
			fmt.Printf("- %s %s unit=%s amount=%d building=%d signal=%s confidence=%s\n",
				item.Time, item.PlayerLabel, unitText(item.UnitID, item.UnitName), item.Amount, item.BuildingID, signal, item.Confidence)
		}
		if len(report.Production) > limit {
			fmt.Printf("- ... %d more\n", len(report.Production)-limit)
		}
	}
	printWarnings(report.Warnings)
}

func printRazes(report *cba.RazeReport) {
	fmt.Printf("path: %s\n", report.Path)
	fmt.Printf("method: %s\n", report.Method)
	fmt.Printf("verification: %s\n", report.Verification)
	fmt.Printf("summary: building_target_events=%d pressured_targets=%d multi_player_targets=%d raze_candidates=%d set_play_candidates=%d\n",
		report.Summary.BuildingTargetEvents, report.Summary.PressuredTargets, report.Summary.MultiPlayerTargets, report.Summary.RazeCandidates, report.Summary.SetPlayCandidates)
	if len(report.Players) > 0 {
		fmt.Println("players:")
		for _, player := range report.Players {
			first := "none"
			if player.FirstRazeCandidate != "" {
				first = fmt.Sprintf("%s target=%d order=%d fit=%s", player.FirstRazeCandidate, player.FirstRazeTarget, player.RazeOrder, player.ArchetypeFit)
			}
			pressure := "none"
			if player.FirstPressureTime != "" {
				pressure = fmt.Sprintf("%s target=%d order=%d fit=%s", player.FirstPressureTime, player.FirstPressureTarget, player.PressureOrder, player.PressureArchetypeFit)
			}
			fmt.Printf("- %s name=%q civ=%d/%s archetype=%s first_pressure=%s first_raze_candidate=%s set_finishes=%d confidence=%s\n",
				player.Label, player.Name, player.CivID, player.CivName, player.RazeArchetype, pressure, first, player.SetPlayFinishes, player.Confidence)
		}
	}
	if len(report.Events) > 0 {
		fmt.Println("raze_candidates:")
		limit := len(report.Events)
		if limit > 80 {
			limit = 80
		}
		for _, event := range report.Events[:limit] {
			fmt.Printf("- %s %s villager=%d target=%d/%s owner=%s unit=%s xy=%.2f,%.2f last_pressure=%s delta=%s attackers=%d confidence=%s\n",
				event.Time, event.PlayerLabel, event.VillagerObjectID, event.TargetID, event.TargetClass, event.TargetOwnerLabel, unitText(event.TargetUnitID, event.TargetUnitName), event.TargetX, event.TargetY, event.LastAttackTime, event.Delta, event.DistinctAttackers, event.Confidence)
		}
		if len(report.Events) > limit {
			fmt.Printf("- ... %d more\n", len(report.Events)-limit)
		}
	}
	if len(report.SetPlays) > 0 {
		fmt.Println("set_plays:")
		limit := len(report.SetPlays)
		if limit > 80 {
			limit = 80
		}
		for _, set := range report.SetPlays[:limit] {
			parts := make([]string, 0, len(set.WeakenerSequence))
			for _, pressure := range set.WeakenerSequence {
				parts = append(parts, fmt.Sprintf("%s@%s", pressure.PlayerLabel, pressure.FirstTime))
			}
			fmt.Printf("- %s target=%d/%s owner=%s sequence=%s finisher=%s villager=%d confidence=%s\n",
				set.Time, set.TargetID, set.TargetClass, set.TargetOwnerLabel, strings.Join(parts, " -> "), set.FinisherLabel, set.VillagerObjectID, set.Confidence)
		}
		if len(report.SetPlays) > limit {
			fmt.Printf("- ... %d more\n", len(report.SetPlays)-limit)
		}
	}
	printWarnings(report.Warnings)
}

func printRazePressure(report *cba.RazePressureReport) {
	fmt.Printf("path: %s\n", report.Path)
	fmt.Printf("method: %s\n", report.Method)
	fmt.Printf("verification: %s\n", report.Verification)
	fmt.Printf("summary: targets=%d primary=%d gates=%d castles=%d walls=%d other_buildings=%d pressure_events=%d nearby_loss_pulses=%d net_lost_nearby=%d players=%d\n",
		report.Summary.Targets, report.Summary.PrimaryTargets, report.Summary.GateTargets,
		report.Summary.CastleTargets, report.Summary.WallTargets, report.Summary.OtherBuildingTargets,
		report.Summary.PressureEvents, report.Summary.NearbyLossPulses,
		report.Summary.NetObjectsLostNearby, report.Summary.PlayersWithPressure)
	if len(report.Players) > 0 {
		fmt.Println("players:")
		for _, player := range report.Players {
			fmt.Printf("- %s name=%q pressure=%d primary_pressure=%d targets=%d primary_targets=%d nearby_loss_targets=%d net_lost_nearby=%d\n",
				player.PlayerLabel, player.PlayerName, player.PressureEvents, player.PrimaryPressureEvents,
				player.Targets, player.PrimaryTargets, player.NearbyLossTargets, player.NetObjectsLostNearby)
		}
	}
	if len(report.Targets) > 0 {
		fmt.Println("targets:")
		limit := len(report.Targets)
		if limit > 40 {
			limit = 40
		}
		for _, target := range report.Targets[:limit] {
			fmt.Printf("- target=%d kind=%s primary=%t owner=%s/%s unit=%d/%s xy=%.1f,%.1f pressure=%d first=%s last=%s nearby_loss_pulses=%d net_lost_nearby=%d confidence=%s",
				target.TargetID, target.Kind, target.Primary, target.OwnerLabel, target.OwnerName,
				target.UnitID, target.UnitName, target.X, target.Y, target.PressureEvents,
				target.FirstPressureTime, target.LastPressureTime, target.NearbyLossPulses,
				target.NetObjectsLostNearby, target.Confidence)
			if len(target.Attackers) > 0 {
				fmt.Print(" attackers=")
				for i, attacker := range target.Attackers {
					if i > 0 {
						fmt.Print(",")
					}
					fmt.Printf("P%d/%s:%d", attacker.PlayerID, attacker.PlayerName, attacker.Events)
				}
			}
			fmt.Println()
		}
		if len(report.Targets) > limit {
			fmt.Printf("- ... %d more targets\n", len(report.Targets)-limit)
		}
	}
	printWarnings(report.Warnings)
}

func printPerformance(report *cba.PerformanceReport) {
	fmt.Printf("path: %s\n", report.Path)
	fmt.Printf("method: %s\n", report.Method)
	fmt.Printf("verification: %s\n", report.Verification)
	fmt.Printf("summary: players=%d duration=%s winner_known=%t profile_ids=%d raze_candidates=%d set_plays=%d checksum_series_rows=%d\n",
		report.Summary.Players, report.Summary.Duration, report.Summary.WinnerKnown,
		report.Summary.RowsWithProfileID, report.Summary.RazeCandidates, report.Summary.SetPlayCandidates,
		report.Summary.ChecksumSeriesRows)
	for _, row := range report.Rows {
		fmt.Printf("- slot=%d profile=%d team=%s civ=%s won=%s apm=%s cg=%s switches=%s spatial=%s chat=%s taunts=%s own_razes=%s own_vills=%s own_first_raze=%s own_first_vill=%s team_razes=%s team_vills=%s team_first_raze=%s team_first_vill=%s prod=%s first_prod=%s defensive=%s viewlocks=%s dist=%s jumps=%s units_produced=%s units_lost=%s confidence=%s\n",
			row.Slot, row.ProfileID, row.Team, row.Civ,
			intText(row.Won), floatText(row.APM), intText(row.ControlGroups), intText(row.CGSwitches),
			intText(row.Spatial), intText(row.Chat), intText(row.Taunts), intText(row.OwnRazes),
			intText(row.OwnVills), intText(row.OwnFirstRazeS), intText(row.OwnFirstVillS),
			intText(row.TeamRazes), intText(row.TeamVills), intText(row.TeamFirstRazeS), intText(row.TeamFirstVillS),
			intText(row.ProdBuildings), intText(row.FirstProdBuildS),
			intText(row.DefensiveBuilds), intText(row.ViewlockEvents), intText(row.ViewlockDist),
			intText(row.ViewlockJumps), intText(row.UnitsProduced), intText(row.UnitsLost), row.MetricConfidence)
		if len(row.AttributionWarnings) > 0 {
			for _, warning := range row.AttributionWarnings {
				fmt.Printf("  note: %s\n", warning)
			}
		}
	}
	if len(report.Events) > 0 {
		limit := len(report.Events)
		if limit > 24 {
			limit = 24
		}
		fmt.Printf("events (first %d of %d):\n", limit, len(report.Events))
		for _, event := range report.Events[:limit] {
			fmt.Printf("- %s %s %s count=%d unit=%d/%s building=%d/%s target=%d/%s confidence=%s\n",
				event.Time, event.PlayerLabel, event.Kind, event.Count, event.UnitID, event.UnitName,
				event.BuildingID, event.BuildingName, event.TargetID, event.TargetClass, event.Confidence)
		}
	}
	printWarnings(report.Warnings)
}

func printDoctrine(report *cba.DoctrineReport) {
	fmt.Printf("path: %s\n", report.Path)
	fmt.Printf("method: %s\n", report.Method)
	fmt.Printf("verification: %s\n", report.Verification)
	fmt.Printf("summary: players=%d duration=%s winner_known=%t raze_candidates=%d set_plays=%d own_vill=%d team_vill=%d duties=%d kill_attr=%s raze_attr=%s phase_basis=%s\n",
		report.Summary.Players, report.Summary.Duration, report.Summary.WinnerKnown,
		report.Summary.RazeCandidates, report.Summary.SetPlayCandidates,
		report.Summary.PlayersWithOwnVill, report.Summary.PlayersWithTeamVill,
		report.Summary.PlayersWithDuty, report.Summary.DirectKillAttribution,
		report.Summary.DirectRazeAttribution, report.Summary.ReplayPhaseBoundaryBasis)
	for _, row := range report.Players {
		name := row.Name
		if name == "" {
			name = "unknown"
		}
		fmt.Printf("- slot=%d profile=%d name=%q team=%s civ=%s won=%s phase=%s duty=%q duty_confidence=%s own_vill=%s team_vill=%s own_razes=%s team_razes=%s first_pressure=%s first_raze=%s set_finishes=%d prod=%s first_prod=%s combat_proxy=%q confidence=%s\n",
			row.Slot, row.ProfileID, name, row.Team, row.Civ, intText(row.Won),
			row.PhaseRead, row.Duty, row.DutyConfidence, intText(row.OwnFirstVillS),
			intText(row.TeamFirstVillS), intText(row.OwnRazes), intText(row.TeamRazes),
			blankText(row.FirstPressureTime), blankText(row.FirstRazeCandidate), row.SetPlayFinishes,
			intText(row.ProdBuildings), intText(row.FirstProdBuildS), row.CombatProxy, row.MetricConfidence)
		if row.ProgressionSignal != "" {
			fmt.Printf("  progression: %s\n", row.ProgressionSignal)
		}
		for _, warning := range row.AttributionWarnings {
			fmt.Printf("  note: %s\n", warning)
		}
	}
	printWarnings(report.Warnings)
}

func blankText(value string) string {
	if value == "" {
		return "none"
	}
	return value
}

func unitText(unitID int, unitName string) string {
	if unitName == "" {
		unitName = replay.UnitDisplayName(unitID)
	}
	if unitName == "" {
		return fmt.Sprintf("%d", unitID)
	}
	return fmt.Sprintf("%d/%s", unitID, unitName)
}

func printJSON(value any) {
	if err := jsonout.Print(value); err != nil {
		die("json", err)
	}
}

func printWarnings(warnings []string) {
	if len(warnings) == 0 {
		return
	}
	fmt.Println("warnings:")
	for _, warning := range warnings {
		fmt.Printf("- %s\n", warning)
	}
}

func intText(value *int) string {
	if value == nil {
		return "null"
	}
	return fmt.Sprintf("%d", *value)
}

func floatText(value *float64) string {
	if value == nil {
		return "null"
	}
	return fmt.Sprintf("%.2f", *value)
}

func die(prefix string, err error) {
	fmt.Fprintf(os.Stderr, "%s: %v\n", prefix, err)
	os.Exit(1)
}

func usage() {
	fmt.Fprint(os.Stderr, `Usage:
  cba balance [--text]
  cba sidechannels <file.aoe2record|replay.zip> [--text]
  cba phase-facts <file.aoe2record|replay.zip> --dat <empires2_x2_p1.dat> [--text]
  cba trigger-razes <file.aoe2record|replay.zip> [--dat <empires2_x2_p1.dat>] [--text]
  cba trigger-spawns <file.aoe2record|replay.zip> [--dat <empires2_x2_p1.dat>] [--text]
  cba replay progression <file.aoe2record|replay.zip> [--text]
  cba replay razes <file.aoe2record|replay.zip> [--text]
  cba replay raze-pressure <file.aoe2record|replay.zip> [--text]
  cba replay phases <file.aoe2record|replay.zip> --dat <empires2_x2_p1.dat> [--text]
  cba replay perf <file.aoe2record|replay.zip> [--text]
  cba replay doctrine <file.aoe2record|replay.zip> [--text]
`)
}
