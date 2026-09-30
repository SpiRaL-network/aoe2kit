package cba

import (
	"aoe2kit/pkg/testfixtures"
	"os"
	"testing"
)

func TestLatestCBAProgressionGolden(t *testing.T) {
	path := testfixtures.Path(t, "save-analysis/latest_cba.aoe2record")
	if _, err := os.Stat(path); err != nil {
		t.Skipf("golden replay not present: %v", err)
	}
	report, err := BuildProgression(path)
	if err != nil {
		t.Fatal(err)
	}
	if report.Summary.ProductionEvents != 4163 || report.Summary.ResearchEvents != 233 {
		t.Fatalf("progression summary = production %d research %d, want 4163/233", report.Summary.ProductionEvents, report.Summary.ResearchEvents)
	}
	if report.Summary.ImperialProxySignals != 1 {
		t.Fatalf("imperial proxy signals = %d, want 1", report.Summary.ImperialProxySignals)
	}
	p4 := progressionPlayerByID(report.Players, 4)
	if p4 == nil {
		t.Fatalf("missing P4 progression")
	}
	if p4.CivName != "Chinese" {
		t.Fatalf("P4 civ name = %q, want Chinese", p4.CivName)
	}
	if p4.FirstImperialProxyAt != "00:37:36.054" || p4.FirstImperialProxy != "1901/Fire Lancer" {
		t.Fatalf("P4 imperial proxy = %s %s, want 00:37:36.054 1901/Fire Lancer", p4.FirstImperialProxyAt, p4.FirstImperialProxy)
	}
}

func TestLatestCBARazeSetGolden(t *testing.T) {
	path := testfixtures.Path(t, "save-analysis/latest_cba.aoe2record")
	if _, err := os.Stat(path); err != nil {
		t.Skipf("golden replay not present: %v", err)
	}
	report, err := BuildRazes(path)
	if err != nil {
		t.Fatal(err)
	}
	if report.Summary.RazeCandidates != 7 || report.Summary.SetPlayCandidates != 2 {
		t.Fatalf("raze summary = candidates %d sets %d, want 7/2", report.Summary.RazeCandidates, report.Summary.SetPlayCandidates)
	}
	p6 := razePlayerByID(report.Players, 6)
	if p6 == nil {
		t.Fatalf("missing P6 raze summary")
	}
	if p6.CivName != "Ethiopians" || p6.RazeArchetype != "raze_first_expected" {
		t.Fatalf("P6 civ/archetype = %q/%q, want Ethiopians/raze_first_expected", p6.CivName, p6.RazeArchetype)
	}
	if p6.FirstPressureTime != "00:02:58.274" || p6.PressureOrder != 1 || p6.PressureArchetypeFit != "fits_early_raze_expectation" {
		t.Fatalf("P6 pressure = time %s order %d fit %s, want 00:02:58.274/1/fits_early_raze_expectation", p6.FirstPressureTime, p6.PressureOrder, p6.PressureArchetypeFit)
	}
	set := setPlayByTargetFinisher(report.SetPlays, 5252, 2)
	if set == nil {
		t.Fatalf("missing P1->P2 set play on target 5252")
	}
	if set.Time != "00:05:59.834" || len(set.WeakenerSequence) != 2 {
		t.Fatalf("set play = time %s sequence %d, want 00:05:59.834/2", set.Time, len(set.WeakenerSequence))
	}
}

func TestLatestCBAPerformanceTeamTempoGolden(t *testing.T) {
	path := testfixtures.Path(t, "save-analysis/latest_cba.aoe2record")
	if _, err := os.Stat(path); err != nil {
		t.Skipf("golden replay not present: %v", err)
	}
	report, err := BuildPerformance(path)
	if err != nil {
		t.Fatal(err)
	}
	if report.Summary.Players != 8 || report.Summary.MetricEvents != 4386 {
		t.Fatalf("perf summary players/events = %d/%d, want 8/4386", report.Summary.Players, report.Summary.MetricEvents)
	}
	p1 := performanceBySlot(report.Rows, 1)
	if p1 == nil {
		t.Fatalf("missing P1 performance")
	}
	if intValue(p1.OwnFirstVillS) != 666 || intValue(p1.OwnVills) != 1 || intValue(p1.OwnRazes) != 1 {
		t.Fatalf("P1 own vill/raze tempo = first_vill %v vills %v razes %v, want 666/1/1", p1.OwnFirstVillS, p1.OwnVills, p1.OwnRazes)
	}
	if intValue(p1.TeamFirstVillS) != 359 || intValue(p1.TeamVills) != 4 || intValue(p1.TeamRazes) != 4 {
		t.Fatalf("P1 team vill/raze tempo = first_vill %v vills %v razes %v, want 359/4/4", p1.TeamFirstVillS, p1.TeamVills, p1.TeamRazes)
	}
	if len(report.Events) == 0 || report.Events[0].Kind != "raze_candidate" || report.Events[0].PlayerID != 2 || report.Events[0].TimeMS != 359834 {
		t.Fatalf("first metric event = %+v, want P2 raze_candidate at 359834", report.Events[0])
	}
}

func TestCBA2TeamOutcomeDoesNotCrownLastStandingLoser(t *testing.T) {
	path := testfixtures.Path(t, "save-analysis/diagnostics/cba2.aoe2record")
	if _, err := os.Stat(path); err != nil {
		t.Skipf("golden replay not present: %v", err)
	}
	report, err := BuildPerformance(path)
	if err != nil {
		t.Fatal(err)
	}
	if !report.Summary.WinnerKnown {
		t.Fatalf("winner_known = false, want CBA domain team outcome")
	}
	for _, slot := range []int{1, 2, 3, 4} {
		row := performanceBySlot(report.Rows, slot)
		if row == nil || intValue(row.Won) != 1 {
			t.Fatalf("slot %d won = %+v, want team A winner", slot, row)
		}
	}
	for _, slot := range []int{5, 6, 7, 8} {
		row := performanceBySlot(report.Rows, slot)
		if row == nil || intValue(row.Won) != 0 {
			t.Fatalf("slot %d won = %+v, want team B loser", slot, row)
		}
	}
	p6 := performanceBySlot(report.Rows, 6)
	if intValue(p6.Resigned) != 0 {
		t.Fatalf("P6 resigned = %v, want 0: last-standing loser inherited team loss", p6.Resigned)
	}
}

func TestLatestCBADoctrineGolden(t *testing.T) {
	path := testfixtures.Path(t, "save-analysis/latest_cba.aoe2record")
	if _, err := os.Stat(path); err != nil {
		t.Skipf("golden replay not present: %v", err)
	}
	report, err := BuildDoctrine(path)
	if err != nil {
		t.Fatal(err)
	}
	if report.Summary.Players != 8 || report.Summary.RazeCandidates != 7 || report.Summary.SetPlayCandidates != 2 {
		t.Fatalf("doctrine summary = players %d razes %d sets %d, want 8/7/2",
			report.Summary.Players, report.Summary.RazeCandidates, report.Summary.SetPlayCandidates)
	}
	if report.Summary.DirectKillAttribution != "not_available_from_replay_parser" {
		t.Fatalf("direct kill attribution = %q", report.Summary.DirectKillAttribution)
	}
	p1 := doctrineBySlot(report.Players, 1)
	if p1 == nil {
		t.Fatalf("missing P1 doctrine row")
	}
	if intValue(p1.OwnFirstVillS) != 666 || intValue(p1.TeamFirstVillS) != 359 {
		t.Fatalf("P1 phase boundary = own %v team %v, want 666/359", p1.OwnFirstVillS, p1.TeamFirstVillS)
	}
	if p1.Duty == "" || p1.DutyConfidence == "" {
		t.Fatalf("P1 duty not populated: %+v", p1)
	}
}

func TestCBABalanceRecentCoverage(t *testing.T) {
	report := BuildBalance()
	if report.Summary.Rows < 20 {
		t.Fatalf("balance rows = %d, want recent coverage rows", report.Summary.Rows)
	}
	mapuche := balanceRow(report.Rows, "V292", 58)
	if mapuche == nil {
		t.Fatalf("missing V292 Mapuche carry-forward row")
	}
	if intValue(mapuche.RazesToVillager) != 1 || intValue(mapuche.CastleKills) != 100 || intValue(mapuche.ImperialKills) != 400 || !mapuche.PreferredForV292 {
		t.Fatalf("V292 Mapuche = razes %v castle %v imp %v preferred %t, want 1/100/400/true",
			mapuche.RazesToVillager, mapuche.CastleKills, mapuche.ImperialKills, mapuche.PreferredForV292)
	}
	japanese := balanceRow(report.Rows, "V293", 5)
	if japanese == nil || intValue(japanese.RazesToVillager) != 2 {
		t.Fatalf("V293 Japanese row = %+v, want razes_to_villager 2", japanese)
	}
	for _, missing := range report.Missing {
		if missing.Version == "V293" && missing.CivID == 5 {
			t.Fatalf("V293 Japanese should no longer be missing: %+v", missing)
		}
		if missing.Version == "V286" && missing.CivName == "Mapuche" {
			t.Fatalf("V286 Mapuche should no longer be missing: %+v", missing)
		}
	}
}

func doctrineBySlot(rows []DoctrinePlayerRow, slot int) *DoctrinePlayerRow {
	for i := range rows {
		if rows[i].Slot == slot {
			return &rows[i]
		}
	}
	return nil
}

func TestLatestCBATriggerRazesGolden(t *testing.T) {
	path := testfixtures.Path(t, "save-analysis/latest_cba.aoe2record")
	datPath := testfixtures.Path(t, "reference_aoe2_dump/dat/empires2_x2_p1.dat")
	if _, err := os.Stat(path); err != nil {
		t.Skipf("golden replay not present: %v", err)
	}
	if _, err := os.Stat(datPath); err != nil {
		t.Skipf("golden DAT not present: %v", err)
	}
	report, err := BuildTriggerRazes(path, datPath)
	if err != nil {
		t.Fatal(err)
	}
	if report.Summary.TriggerCount != 3218 || report.Summary.RewardRungTriggers != 40 || report.Summary.ActivatorTriggers != 40 {
		t.Fatalf("trigger raze summary = triggers %d rungs %d activators %d, want 3218/40/40",
			report.Summary.TriggerCount, report.Summary.RewardRungTriggers, report.Summary.ActivatorTriggers)
	}
	if report.Summary.Conflicts != 0 || report.Summary.Buckets != 5 || report.Summary.MapucheRazesToVillager != 1 {
		t.Fatalf("trigger raze buckets/conflicts/mapuche = %d/%d/%d, want 5/0/1",
			report.Summary.Buckets, report.Summary.Conflicts, report.Summary.MapucheRazesToVillager)
	}
	for _, tc := range []struct {
		civ   string
		razes int
	}{
		{"Mapuche", 1},
		{"Persians", 4},
		{"Huns", 5},
	} {
		row := triggerRazeRowByCiv(report.Rows, tc.civ)
		if row == nil {
			t.Fatalf("missing trigger-raze row for %s", tc.civ)
		}
		if row.RazesToVillager != tc.razes {
			t.Fatalf("%s razes_to_villager = %d, want %d", tc.civ, row.RazesToVillager, tc.razes)
		}
	}
}

func TestCBATriggerSpawnsGolden(t *testing.T) {
	path := testfixtures.Path(t, "save-analysis/archive/078__MP Replay v101.103.48987.0 @2026.07.19 224323 (5).aoe2record")
	datPath := testfixtures.Path(t, "reference_aoe2_dump/dat/empires2_x2_p1.dat")
	if _, err := os.Stat(path); err != nil {
		t.Skipf("golden replay not present: %v", err)
	}
	if _, err := os.Stat(datPath); err != nil {
		t.Skipf("golden DAT not present: %v", err)
	}
	report, err := BuildTriggerSpawns(path, datPath)
	if err != nil {
		t.Fatal(err)
	}
	if report.Summary.TriggerCount != 3218 || report.Summary.SpawnerSecondTriggers != 72 || report.Summary.SpawnCountTriggers != 976 {
		t.Fatalf("trigger spawn summary = triggers %d seconds %d counts %d, want 3218/72/976",
			report.Summary.TriggerCount, report.Summary.SpawnerSecondTriggers, report.Summary.SpawnCountTriggers)
	}
	if report.Summary.CompleteRows < 55 || report.Summary.MapucheSpawnUnitCount != 80 || report.Summary.MapucheSpawnSeconds != 10 {
		t.Fatalf("trigger spawn decoded rows/mapuche = complete %d mapuche %d/%d, want >=55 and 80/10",
			report.Summary.CompleteRows, report.Summary.MapucheSpawnUnitCount, report.Summary.MapucheSpawnSeconds)
	}
	for _, tc := range []struct {
		civ     string
		units   int
		seconds int
	}{
		{"Mapuche", 80, 10},
		{"Khmer", 30, 15},
		{"Persians", 60, 13},
		{"Malay", 120, 6},
	} {
		row := triggerSpawnRowByCiv(report.Rows, tc.civ)
		if row == nil {
			t.Fatalf("missing trigger-spawn row for %s", tc.civ)
		}
		if row.SpawnUnitCount == nil || *row.SpawnUnitCount != tc.units || row.SpawnSeconds == nil || *row.SpawnSeconds != tc.seconds {
			t.Fatalf("%s spawn = units %v seconds %v, want %d/%d",
				tc.civ, row.SpawnUnitCount, row.SpawnSeconds, tc.units, tc.seconds)
		}
	}
}

func TestLatestCBASideChannelsGolden(t *testing.T) {
	path := testfixtures.Path(t, "save-analysis/latest_cba.aoe2record")
	if _, err := os.Stat(path); err != nil {
		t.Skipf("golden replay not present: %v", err)
	}
	report, err := BuildSideChannels(path)
	if err != nil {
		t.Fatal(err)
	}
	if report.Summary.TriggerCount != 3218 || report.Summary.ModifyResourceEffects != 1328 || report.Summary.ChangeVariableEffects != 1600 {
		t.Fatalf("sidechannel summary = triggers %d modify_resource %d change_variable %d, want 3218/1328/1600",
			report.Summary.TriggerCount, report.Summary.ModifyResourceEffects, report.Summary.ChangeVariableEffects)
	}
	if report.GraphSHA256 != "172fa04c719615257ecc7991a2605c8f7ac45c05bdd72a8758c759687f2dd28c" || report.ScenarioToken != LadderToken || !report.Scenario.Ladder {
		t.Fatalf("sidechannel identity = graph %s token %q ladder %t, want registered V292 host-copy B",
			report.GraphSHA256, report.ScenarioToken, report.Scenario.Ladder)
	}
	if report.Summary.SideChannelVariant != "cba_requiem_v292_full_kill_death_raze_bridge" {
		t.Fatalf("sidechannel variant = %q, want full bridge", report.Summary.SideChannelVariant)
	}
	if report.Summary.AccumAttributeConds != 480 || report.Summary.ScoreboardRows != 4 || report.Summary.EngineAttrChannels != 3 {
		t.Fatalf("sidechannel channels = accumulate %d scoreboard %d engine %d, want 480/4/3",
			report.Summary.AccumAttributeConds, report.Summary.ScoreboardRows, report.Summary.EngineAttrChannels)
	}
	kills := sideChannelEngineAttr(report.EngineAttrs, 20)
	if kills == nil || kills.Semantic != "engine_kills_consumed_by_cba_scoreboard" || !intSliceEqual(kills.Quantities, []int{5, 10}) {
		t.Fatalf("kill attr row = %+v, want attr20 quantities 5/10", kills)
	}
	deaths := sideChannelEngineAttr(report.EngineAttrs, 154)
	if deaths == nil || deaths.Semantic != "engine_deaths_consumed_by_cba_scoreboard" || !intSliceEqual(deaths.Quantities, []int{10}) {
		t.Fatalf("death attr row = %+v, want attr154 quantity 10", deaths)
	}
	razes := sideChannelEngineAttr(report.EngineAttrs, 43)
	if razes == nil || razes.Semantic != "engine_razes_consumed_by_cba_villager_reward_and_scoreboard" || !intSliceEqual(razes.Quantities, []int{1}) {
		t.Fatalf("raze attr row = %+v, want attr43 quantity 1", razes)
	}
	row := sideChannelScoreboardRow(report.Scoreboard, "P1-5")
	if row == nil || row.KillResourceA != 571 || row.KillResourceB != 391 || row.DeathResourceA != 60 || row.DeathResourceB != 361 || row.RazeResourceA != 111 || row.RazeResourceB != 395 {
		t.Fatalf("P1-5 scoreboard row = %+v", row)
	}
	res391 := sideChannelResource(report.Resources, 391)
	if res391 == nil || res391.Family != "kill_scoreboard_resource_partner_or_gaia_mirror" || res391.Count != 145 {
		t.Fatalf("resource 391 = %+v, want kill mirror writes=145", res391)
	}
	res395 := sideChannelResource(report.Resources, 395)
	if res395 == nil || res395.Family != "raze_scoreboard_resource_partner_or_gaia_mirror" || !intSliceEqual(res395.Amounts, []int{1, 2, 3, 4, 5}) {
		t.Fatalf("resource 395 = %+v, want raze mirror amounts 1..5", res395)
	}
}

func TestCBA2SideChannelsGolden(t *testing.T) {
	path := testfixtures.Path(t, "save-analysis/diagnostics/cba2.aoe2record")
	if _, err := os.Stat(path); err != nil {
		t.Skipf("golden replay not present: %v", err)
	}
	report, err := BuildSideChannels(path)
	if err != nil {
		t.Fatal(err)
	}
	if report.GraphSHA256 != "172fa04c719615257ecc7991a2605c8f7ac45c05bdd72a8758c759687f2dd28c" {
		t.Fatalf("graph = %s, want cba2 registered V292 host-copy B", report.GraphSHA256)
	}
	if report.Summary.SideChannelVariant != "cba_requiem_v292_full_kill_death_raze_bridge" {
		t.Fatalf("sidechannel variant = %q, want full bridge", report.Summary.SideChannelVariant)
	}
	if sideChannelEngineAttr(report.EngineAttrs, 43) == nil {
		t.Fatalf("missing attr43 raze bridge in cba2")
	}
}

func TestCBA2PhaseTimelineGolden(t *testing.T) {
	path := testfixtures.Path(t, "save-analysis/diagnostics/cba2.aoe2record")
	datPath := testfixtures.Path(t, "reference_aoe2_dump/dat/empires2_x2_p1.dat")
	if _, err := os.Stat(path); err != nil {
		t.Skipf("golden replay not present: %v", err)
	}
	if _, err := os.Stat(datPath); err != nil {
		t.Skipf("golden DAT not present: %v", err)
	}
	report, err := BuildPhaseTimeline(path, datPath)
	if err != nil {
		t.Fatal(err)
	}
	if report.Summary.Players != 8 || report.Summary.CastleObserved != 8 || report.Summary.ImperialObserved != 1 {
		t.Fatalf("phase summary = players %d castle %d imperial %d, want 8/8/1",
			report.Summary.Players, report.Summary.CastleObserved, report.Summary.ImperialObserved)
	}
	p3 := phaseTimelinePlayerByID(report.Players, 3)
	if p3 == nil {
		t.Fatalf("missing P3 phase row")
	}
	if p3.CivName != "Burmese" || p3.CastleThresholdKills != 300 || p3.ImperialThresholdKills != 600 || p3.RazesToVillager != 3 {
		t.Fatalf("P3 phase facts = civ %s castle %d imperial %d razes %d, want Burmese 300/600/3",
			p3.CivName, p3.CastleThresholdKills, p3.ImperialThresholdKills, p3.RazesToVillager)
	}
	if p3.Castle == nil || p3.Castle.Signal != "low_type_id_positive_object_delta" || p3.Castle.Confidence != "medium_confidence_mixed_delta_type_id_shadow" {
		t.Fatalf("P3 castle shadow = %+v, want medium mixed checksum shadow", p3.Castle)
	}
	if p3.Imperial != nil {
		t.Fatalf("P3 imperial shadow = %+v, want no overconfident low-ID object noise", p3.Imperial)
	}
	p1 := phaseTimelinePlayerByID(report.Players, 1)
	if p1 == nil || p1.Imperial == nil || p1.Imperial.UnitID != 1334 {
		t.Fatalf("P1 imperial shadow = %+v, want clean switch to unit 1334", p1)
	}
}

func TestCBALiveVariantSideChannelsGolden(t *testing.T) {
	path := testfixtures.Path(t, "save-analysis/diagnostics/MP Replay v101.103.48987.0 @2026.08.14 183139 (6).aoe2record")
	if _, err := os.Stat(path); err != nil {
		t.Skipf("golden replay not present: %v", err)
	}
	report, err := BuildSideChannels(path)
	if err != nil {
		t.Fatal(err)
	}
	if report.GraphSHA256 != "32d93dffc7ff551221b71b631240962609086807d186e1b5039bcd7794885f87" {
		t.Fatalf("graph = %s, want live stress variant", report.GraphSHA256)
	}
	if report.ScenarioToken != LadderToken || !report.Scenario.Ladder {
		t.Fatalf("scenario identity = token %q ladder %t, want admitted CBA Requiem V292", report.ScenarioToken, report.Scenario.Ladder)
	}
	if report.Summary.TriggerCount != 2815 || report.Summary.MessageCount != 1077 {
		t.Fatalf("summary = triggers %d messages %d, want 2815/1077", report.Summary.TriggerCount, report.Summary.MessageCount)
	}
	if report.Summary.SideChannelVariant != "cba_requiem_v292_kill_death_bridge_raze_absent_or_truncated" {
		t.Fatalf("sidechannel variant = %q, want kill/death bridge with absent raze", report.Summary.SideChannelVariant)
	}
	if sideChannelEngineAttr(report.EngineAttrs, 20) == nil || sideChannelEngineAttr(report.EngineAttrs, 154) == nil {
		t.Fatalf("missing kill/death attrs: %+v", report.EngineAttrs)
	}
	if sideChannelEngineAttr(report.EngineAttrs, 43) != nil {
		t.Fatalf("unexpected attr43 raze bridge in live variant: %+v", sideChannelEngineAttr(report.EngineAttrs, 43))
	}
}

func TestCBA2RazePressureGolden(t *testing.T) {
	path := testfixtures.Path(t, "save-analysis/diagnostics/cba2.aoe2record")
	if _, err := os.Stat(path); err != nil {
		t.Skipf("golden replay not present: %v", err)
	}
	report, err := BuildRazePressure(path)
	if err != nil {
		t.Fatal(err)
	}
	if report.Summary.Targets < 40 || report.Summary.PrimaryTargets < 20 || report.Summary.GateTargets < 15 || report.Summary.CastleTargets < 3 {
		t.Fatalf("raze pressure summary = targets %d primary %d gates %d castles %d, want live CBA gate/castle coverage",
			report.Summary.Targets, report.Summary.PrimaryTargets, report.Summary.GateTargets, report.Summary.CastleTargets)
	}
	target := razePressureTarget(report.Targets, 58912)
	if target == nil {
		t.Fatalf("missing P6 gate target 58912")
	}
	if target.Kind != "gate" || target.OwnerID != 6 || target.NetObjectsLostNearby < 200 {
		t.Fatalf("target 58912 = %+v, want P6 gate with large nearby loss", target)
	}
	if !razePressureTargetHasAttacker(target, 4) {
		t.Fatalf("target 58912 attackers = %+v, want P4 pressure", target.Attackers)
	}
}

func TestCBALiveVariantRazePressureGolden(t *testing.T) {
	path := testfixtures.Path(t, "save-analysis/diagnostics/MP Replay v101.103.48987.0 @2026.08.14 183139 (6).aoe2record")
	if _, err := os.Stat(path); err != nil {
		t.Skipf("golden replay not present: %v", err)
	}
	report, err := BuildRazePressure(path)
	if err != nil {
		t.Fatal(err)
	}
	if report.Summary.CastleTargets < 4 || report.Summary.GateTargets < 10 {
		t.Fatalf("raze pressure summary = castles %d gates %d, want castle-push/gate pressure coverage",
			report.Summary.CastleTargets, report.Summary.GateTargets)
	}
	target := razePressureTarget(report.Targets, 7969)
	if target == nil {
		t.Fatalf("missing P3 castle target 7969")
	}
	if target.Kind != "castle" || target.OwnerID != 3 {
		t.Fatalf("target 7969 = %+v, want P3 castle", target)
	}
	if !razePressureTargetHasAttacker(target, 6) {
		t.Fatalf("target 7969 attackers = %+v, want P6/chrae pressure", target.Attackers)
	}
	player := razePressurePlayer(report.Players, 6)
	if player == nil || player.PrimaryPressureEvents == 0 {
		t.Fatalf("P6 pressure player = %+v, want primary pressure events", player)
	}
}

func triggerRazeRowByCiv(rows []TriggerRazeRow, civName string) *TriggerRazeRow {
	for i := range rows {
		if rows[i].CivName == civName {
			return &rows[i]
		}
	}
	return nil
}

func sideChannelEngineAttr(rows []EngineAttributeRow, attr int) *EngineAttributeRow {
	for i := range rows {
		if rows[i].Attribute == attr {
			return &rows[i]
		}
	}
	return nil
}

func sideChannelScoreboardRow(rows []ScoreboardChannelRow, label string) *ScoreboardChannelRow {
	for i := range rows {
		if rows[i].Label == label {
			return &rows[i]
		}
	}
	return nil
}

func sideChannelResource(rows []SideChannelResource, resource int) *SideChannelResource {
	for i := range rows {
		if rows[i].Resource == resource {
			return &rows[i]
		}
	}
	return nil
}

func razePressureTarget(rows []RazePressureTarget, targetID int) *RazePressureTarget {
	for i := range rows {
		if rows[i].TargetID == targetID {
			return &rows[i]
		}
	}
	return nil
}

func razePressurePlayer(rows []RazePressurePlayer, playerID int) *RazePressurePlayer {
	for i := range rows {
		if rows[i].PlayerID == playerID {
			return &rows[i]
		}
	}
	return nil
}

func razePressureTargetHasAttacker(target *RazePressureTarget, playerID int) bool {
	if target == nil {
		return false
	}
	for _, attacker := range target.Attackers {
		if attacker.PlayerID == playerID {
			return true
		}
	}
	return false
}

func phaseTimelinePlayerByID(rows []PhaseTimelinePlayer, playerID int) *PhaseTimelinePlayer {
	for i := range rows {
		if rows[i].PlayerID == playerID {
			return &rows[i]
		}
	}
	return nil
}

func triggerSpawnRowByCiv(rows []TriggerSpawnRow, civName string) *TriggerSpawnRow {
	for i := range rows {
		if rows[i].CivName == civName {
			return &rows[i]
		}
	}
	return nil
}

func intSliceEqual(a, b []int) bool {
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

func progressionPlayerByID(players []ProgressionPlayerSummary, playerID int) *ProgressionPlayerSummary {
	for i := range players {
		if players[i].PlayerID == playerID {
			return &players[i]
		}
	}
	return nil
}

func razePlayerByID(players []RazePlayerSummary, playerID int) *RazePlayerSummary {
	for i := range players {
		if players[i].PlayerID == playerID {
			return &players[i]
		}
	}
	return nil
}

func setPlayByTargetFinisher(sets []SetPlayCandidate, targetID int, finisherID int) *SetPlayCandidate {
	for i := range sets {
		if sets[i].TargetID == targetID && sets[i].FinisherID == finisherID {
			return &sets[i]
		}
	}
	return nil
}

func performanceBySlot(rows []PerformanceRow, slot int) *PerformanceRow {
	for i := range rows {
		if rows[i].Slot == slot {
			return &rows[i]
		}
	}
	return nil
}

func balanceRow(rows []BalanceRow, version string, civID int) *BalanceRow {
	for i := range rows {
		if rows[i].Version == version && rows[i].CivID == civID {
			return &rows[i]
		}
	}
	return nil
}

func intValue(value *int) int {
	if value == nil {
		return -1
	}
	return *value
}
