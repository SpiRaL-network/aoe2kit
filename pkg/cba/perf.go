package cba

import (
	"sort"

	"aoe2kit/pkg/replay"
)

var cbaProductionBuildings = map[int]string{
	12:  "barracks",
	87:  "range",
	101: "stable",
	49:  "siege",
	82:  "castle",
}

type PerformanceReport struct {
	Path         string             `json:"path,omitempty"`
	Method       string             `json:"method"`
	Verification string             `json:"verification"`
	Summary      PerformanceSummary `json:"summary"`
	Rows         []PerformanceRow   `json:"performances"`
	Events       []PerformanceEvent `json:"events,omitempty"`
	Warnings     []string           `json:"warnings,omitempty"`
}

type PerformanceSummary struct {
	Players            int    `json:"players"`
	DurationMS         int    `json:"duration_ms,omitempty"`
	Duration           string `json:"duration,omitempty"`
	WinnerKnown        bool   `json:"winner_known"`
	RowsWithProfileID  int    `json:"rows_with_profile_id"`
	RazeCandidates     int    `json:"raze_candidates"`
	SetPlayCandidates  int    `json:"set_play_candidates"`
	ChecksumSeriesRows int    `json:"checksum_series_rows"`
	MetricEvents       int    `json:"metric_events"`
}

type PerformanceRow struct {
	GameID              string   `json:"game_id,omitempty"`
	ProfileID           int      `json:"profile_id,omitempty"`
	Slot                int      `json:"slot"`
	Team                string   `json:"team,omitempty"`
	Civ                 string   `json:"civ,omitempty"`
	Won                 *int     `json:"won"`
	APM                 *float64 `json:"apm"`
	ControlGroups       *int     `json:"control_groups"`
	CGSwitches          *int     `json:"cg_switches"`
	Spatial             *int     `json:"spatial"`
	Chat                *int     `json:"chat"`
	Taunts              *int     `json:"taunts"`
	Razes               *int     `json:"razes"`
	VillagersGained     *int     `json:"villagers_gained"`
	OwnFirstVillS       *int     `json:"own_first_vill_s"`
	OwnVills            *int     `json:"own_vills"`
	OwnRazes            *int     `json:"own_razes"`
	OwnFirstRazeS       *int     `json:"own_first_raze_s"`
	TeamFirstVillS      *int     `json:"team_first_vill_s"`
	TeamVills           *int     `json:"team_vills"`
	TeamRazes           *int     `json:"team_razes"`
	TeamFirstRazeS      *int     `json:"team_first_raze_s"`
	ProdBuildings       *int     `json:"prod_buildings"`
	ProdBarracks        *int     `json:"prod_barracks"`
	ProdRange           *int     `json:"prod_range"`
	ProdStable          *int     `json:"prod_stable"`
	ProdSiege           *int     `json:"prod_siege"`
	ProdCastle          *int     `json:"prod_castle"`
	FirstProdBuildS     *int     `json:"first_prod_build_s"`
	DefensiveBuilds     *int     `json:"defensive_builds"`
	ViewlockEvents      *int     `json:"viewlock_events"`
	ViewlockDist        *int     `json:"viewlock_dist"`
	ViewlockJumps       *int     `json:"viewlock_jumps"`
	VocabMove           *float64 `json:"vocab_move"`
	VocabPatrol         *float64 `json:"vocab_patrol"`
	VocabStance         *float64 `json:"vocab_stance"`
	VocabQueue          *float64 `json:"vocab_queue"`
	Resigned            *int     `json:"resigned"`
	ResignTimeS         *int     `json:"resign_time_s"`
	UnitsProduced       *int     `json:"units_produced"`
	UnitsLost           *int     `json:"units_lost"`
	UnitsKilled         *int     `json:"units_killed"`
	MetricConfidence    string   `json:"metric_confidence"`
	AttributionWarnings []string `json:"attribution_warnings,omitempty"`
}

type PerformanceEvent struct {
	Kind             string `json:"kind"`
	TimeMS           int    `json:"time_ms"`
	Time             string `json:"time"`
	PlayerID         int    `json:"player_id"`
	PlayerLabel      string `json:"player_label,omitempty"`
	Team             string `json:"team,omitempty"`
	Count            int    `json:"count,omitempty"`
	UnitID           int    `json:"unit_id,omitempty"`
	UnitName         string `json:"unit_name,omitempty"`
	BuildingID       int    `json:"building_id,omitempty"`
	BuildingName     string `json:"building_name,omitempty"`
	TargetID         int    `json:"target_id,omitempty"`
	TargetClass      string `json:"target_class,omitempty"`
	TargetOwnerID    int    `json:"target_owner_id,omitempty"`
	TargetOwnerLabel string `json:"target_owner_label,omitempty"`
	TargetUnitID     int    `json:"target_unit_id,omitempty"`
	TargetUnitName   string `json:"target_unit_name,omitempty"`
	Source           string `json:"source"`
	Confidence       string `json:"confidence"`
}

func BuildPerformance(path string) (*PerformanceReport, error) {
	rec, err := replay.Open(path)
	if err != nil {
		return nil, err
	}
	profile, err := replay.BuildPlayerProfile(path, replay.PlayerProfileOptions{})
	if err != nil {
		return nil, err
	}
	events, err := replay.ExtractEvents(path, replay.EventOptions{IncludeUntypedAction: true})
	if err != nil {
		return nil, err
	}
	camera, err := replay.BuildCamera(path, replay.CameraOptions{Tail: -1})
	if err != nil {
		return nil, err
	}
	series, err := replay.BuildPlayerSeries(path, replay.PlayerSeriesOptions{ChangesOnly: true})
	if err != nil {
		return nil, err
	}
	razes, err := BuildRazes(path)
	if err != nil {
		return nil, err
	}

	report := &PerformanceReport{
		Path:         path,
		Method:       "cba_multi_axis_row_extractor_from_replay_primitives",
		Verification: "mixed: direct_action_metrics_plus_checksum_object_series_plus_raze_candidates_no_kill_attribution",
		Summary: PerformanceSummary{
			DurationMS:         profile.EventCounts.DurationMS,
			Duration:           profile.EventCounts.Duration,
			WinnerKnown:        profile.Result.WinnerKnown,
			RazeCandidates:     razes.Summary.RazeCandidates,
			SetPlayCandidates:  razes.Summary.SetPlayCandidates,
			ChecksumSeriesRows: len(series.Players),
		},
		Warnings: append([]string{}, profile.Warnings...),
	}
	report.Warnings = append(report.Warnings, events.Warnings...)
	report.Warnings = append(report.Warnings, camera.Warnings...)
	report.Warnings = append(report.Warnings, series.Warnings...)
	report.Warnings = append(report.Warnings, razes.Warnings...)
	report.Warnings = append(report.Warnings,
		"razes and villagers_gained are candidate/proxy metrics from CBA target-pressure correlation, not direct engine postgame stats",
		"units_produced and units_lost are own-side checksum object-count changes; units_lost is not engine-confirmed deaths and units_killed is not attributed",
		"small checksum object-removal samples can be materially noisy; the Combat axis ignores rows below the production floor",
	)

	rows := map[int]*PerformanceRow{}
	for _, slot := range rec.Players {
		if !slot.Active || slot.Number <= 0 {
			continue
		}
		row := &PerformanceRow{
			ProfileID:        slot.ProfileID,
			Slot:             slot.Number,
			Team:             cbaTeam(slot.Number),
			Civ:              replay.CivDisplayName(slot.Civ),
			MetricConfidence: "structure_verified_not_engine_verified",
		}
		if profile.Result.WinnerKnown {
			won := 0
			if intIn(profile.Result.Winners, slot.Number) {
				won = 1
			}
			row.Won = &won
		}
		rows[slot.Number] = row
		if slot.ProfileID > 0 {
			report.Summary.RowsWithProfileID++
		}
	}
	applyProfileRows(rows, profile)
	metricEvents := applyActionRows(rows, events.Events)
	applyCameraRows(rows, camera)
	applySeriesRows(rows, series)
	metricEvents = append(metricEvents, razeMetricEvents(razes)...)
	applyTempoRows(rows, metricEvents)
	applyResignRows(rows, profile.Result)
	if applyCBATeamOutcome(rows, profile.Result) {
		report.Summary.WinnerKnown = true
		report.Warnings = append(report.Warnings, "CBA team outcome inferred by domain teams after resign stream; unrecorded last-standing losing teammate inherits team loss because DE may stop recording at game end before final defeat op")
	}
	sortPerformanceEvents(metricEvents)
	report.Events = metricEvents
	report.Summary.MetricEvents = len(metricEvents)

	keys := make([]int, 0, len(rows))
	for playerID := range rows {
		keys = append(keys, playerID)
	}
	sort.Ints(keys)
	for _, playerID := range keys {
		report.Rows = append(report.Rows, *rows[playerID])
	}
	report.Summary.Players = len(report.Rows)
	return report, nil
}

func applyProfileRows(rows map[int]*PerformanceRow, profile *replay.PlayerProfileReport) {
	for _, player := range profile.Players {
		row := rows[player.PlayerID]
		if row == nil {
			continue
		}
		row.APM = floatPtr(player.APM)
		row.ControlGroups = intPtr(player.Spatial.CellsVisited)
		row.CGSwitches = intPtr(player.Spatial.CentroidSwitches)
		row.Spatial = intPtr(player.Spatial.Commands)
		row.Chat = intPtr(player.Feedback.Chat)
		row.Taunts = intPtr(player.Feedback.Taunts)
		for _, vocab := range player.Vocabulary {
			switch vocab.ActionName {
			case "MOVE":
				row.VocabMove = floatPtr(vocab.SharePercent)
			case "PATROL":
				row.VocabPatrol = floatPtr(vocab.SharePercent)
			case "STANCE":
				row.VocabStance = floatPtr(vocab.SharePercent)
			case "DE_QUEUE":
				row.VocabQueue = floatPtr(vocab.SharePercent)
			}
		}
	}
}

func applyActionRows(rows map[int]*PerformanceRow, events []replay.ReplayEvent) []PerformanceEvent {
	prodCounts := map[int]map[string]int{}
	defensive := map[int]int{}
	firstProd := map[int]int{}
	var metricEvents []PerformanceEvent
	for _, event := range events {
		if event.PlayerID <= 0 {
			continue
		}
		switch event.Type {
		case "de_queue":
			metricEvents = append(metricEvents, PerformanceEvent{
				Kind:        "unit_queued",
				TimeMS:      event.TimeMS,
				Time:        event.Time,
				PlayerID:    event.PlayerID,
				PlayerLabel: playerLabel(event.PlayerID),
				Team:        cbaTeam(event.PlayerID),
				Count:       event.Amount,
				UnitID:      event.UnitID,
				UnitName:    replay.UnitDisplayName(event.UnitID),
				BuildingID:  event.BuildingID,
				Source:      "action_stream_de_queue",
				Confidence:  "parsed_action_queue_not_spawn_confirmed",
			})
		case "build":
			if _, ok := cbaProductionBuildings[event.TargetID]; ok {
				if prodCounts[event.PlayerID] == nil {
					prodCounts[event.PlayerID] = map[string]int{}
				}
				name := cbaProductionBuildings[event.TargetID]
				prodCounts[event.PlayerID][name]++
				if _, ok := firstProd[event.PlayerID]; !ok {
					firstProd[event.PlayerID] = event.TimeMS / 1000
				}
				metricEvents = append(metricEvents, PerformanceEvent{
					Kind:         "production_building_started",
					TimeMS:       event.TimeMS,
					Time:         event.Time,
					PlayerID:     event.PlayerID,
					PlayerLabel:  playerLabel(event.PlayerID),
					Team:         cbaTeam(event.PlayerID),
					BuildingID:   event.TargetID,
					BuildingName: name,
					Source:       "action_stream_build",
					Confidence:   "parsed_build_command_started_not_completion_confirmed",
				})
			} else {
				defensive[event.PlayerID]++
			}
		case "wall", "gate":
			defensive[event.PlayerID]++
		}
	}
	for playerID, row := range rows {
		counts := prodCounts[playerID]
		total := 0
		for _, value := range counts {
			total += value
		}
		row.ProdBuildings = intPtr(total)
		row.ProdBarracks = intPtr(counts["barracks"])
		row.ProdRange = intPtr(counts["range"])
		row.ProdStable = intPtr(counts["stable"])
		row.ProdSiege = intPtr(counts["siege"])
		row.ProdCastle = intPtr(counts["castle"])
		if value, ok := firstProd[playerID]; ok {
			row.FirstProdBuildS = intPtr(value)
		}
		row.DefensiveBuilds = intPtr(defensive[playerID])
	}
	return metricEvents
}

func applyCameraRows(rows map[int]*PerformanceRow, camera *replay.CameraReport) {
	for _, stream := range camera.Streams {
		row := rows[stream.PlayerNumberIfTail]
		if row == nil {
			continue
		}
		row.ViewlockEvents = intPtr(stream.Events)
		row.ViewlockDist = intPtr(int(stream.DistanceTraveled))
		row.ViewlockJumps = intPtr(stream.LargeJumps)
	}
}

func applySeriesRows(rows map[int]*PerformanceRow, series *replay.PlayerSeriesReport) {
	for _, player := range series.Players {
		row := rows[player.PlayerID]
		if row == nil {
			continue
		}
		row.UnitsProduced = intPtr(player.ProducedEstimate)
		row.UnitsLost = intPtr(player.LostEstimate)
		row.AttributionWarnings = append(row.AttributionWarnings, "units_lost is checksum object removals, not engine-confirmed deaths or enemy kill attribution")
		if player.ProducedEstimate < MinProdForCombat {
			row.AttributionWarnings = append(row.AttributionWarnings, "units_lost sample is below the Combat-axis production floor and may be noisy")
		}
	}
}

func razeMetricEvents(razes *RazeReport) []PerformanceEvent {
	var out []PerformanceEvent
	for _, event := range razes.Events {
		base := PerformanceEvent{
			TimeMS:           event.TimeMS,
			Time:             event.Time,
			PlayerID:         event.PlayerID,
			PlayerLabel:      event.PlayerLabel,
			Team:             cbaTeam(event.PlayerID),
			Count:            1,
			TargetID:         event.TargetID,
			TargetClass:      event.TargetClass,
			TargetOwnerID:    event.TargetOwnerID,
			TargetOwnerLabel: event.TargetOwnerLabel,
			TargetUnitID:     event.TargetUnitID,
			TargetUnitName:   event.TargetUnitName,
			Source:           "cba_raze_candidate_correlation",
			Confidence:       event.Confidence,
		}
		raze := base
		raze.Kind = "raze_candidate"
		out = append(out, raze)
		vill := base
		vill.Kind = "villager_gain_candidate"
		vill.UnitID = event.VillagerUnitID
		vill.UnitName = event.VillagerUnitName
		out = append(out, vill)
	}
	return out
}

func applyTempoRows(rows map[int]*PerformanceRow, events []PerformanceEvent) {
	type tempo struct {
		razes      int
		vills      int
		firstRazeS *int
		firstVillS *int
	}
	own := map[int]*tempo{}
	team := map[string]*tempo{}
	for playerID, row := range rows {
		own[playerID] = &tempo{}
		if row.Team != "" {
			if team[row.Team] == nil {
				team[row.Team] = &tempo{}
			}
		}
	}
	for _, event := range events {
		o := own[event.PlayerID]
		t := team[event.Team]
		if o == nil {
			continue
		}
		seconds := event.TimeMS / 1000
		switch event.Kind {
		case "raze_candidate":
			o.razes += event.Count
			setFirst(&o.firstRazeS, seconds)
			if t != nil {
				t.razes += event.Count
				setFirst(&t.firstRazeS, seconds)
			}
		case "villager_gain_candidate":
			o.vills += event.Count
			setFirst(&o.firstVillS, seconds)
			if t != nil {
				t.vills += event.Count
				setFirst(&t.firstVillS, seconds)
			}
		}
	}
	for playerID, row := range rows {
		o := own[playerID]
		t := team[row.Team]
		if o == nil {
			continue
		}
		row.Razes = intPtr(o.razes)
		row.VillagersGained = intPtr(o.vills)
		row.OwnRazes = intPtr(o.razes)
		row.OwnVills = intPtr(o.vills)
		row.OwnFirstRazeS = cloneIntPtr(o.firstRazeS)
		row.OwnFirstVillS = cloneIntPtr(o.firstVillS)
		if t != nil {
			row.TeamRazes = intPtr(t.razes)
			row.TeamVills = intPtr(t.vills)
			row.TeamFirstRazeS = cloneIntPtr(t.firstRazeS)
			row.TeamFirstVillS = cloneIntPtr(t.firstVillS)
		}
		if o.razes > 0 || o.vills > 0 {
			row.AttributionWarnings = append(row.AttributionWarnings, "own razes/villagers are candidate counts from CBA raze detector")
		}
		if t != nil && (t.razes > 0 || t.vills > 0) {
			row.AttributionWarnings = append(row.AttributionWarnings, "team razes/villagers include teammate candidate counts for enablement/team-tempo analysis")
		}
	}
}

func applyResignRows(rows map[int]*PerformanceRow, result replay.MatchResult) {
	resignTime := map[int]int{}
	for _, event := range result.ResignEvents {
		resignTime[event.PlayerID] = event.TimeMS / 1000
	}
	for playerID, row := range rows {
		resigned := 0
		if intIn(result.Resigned, playerID) {
			resigned = 1
			if value, ok := resignTime[playerID]; ok {
				row.ResignTimeS = intPtr(value)
			}
		}
		row.Resigned = intPtr(resigned)
	}
}

func applyCBATeamOutcome(rows map[int]*PerformanceRow, result replay.MatchResult) bool {
	if !result.Completed || len(result.Resigned) == 0 {
		return false
	}
	type teamState struct {
		players  []int
		resigned int
	}
	teams := map[string]*teamState{}
	for playerID, row := range rows {
		if row.Team == "" {
			continue
		}
		if teams[row.Team] == nil {
			teams[row.Team] = &teamState{}
		}
		teams[row.Team].players = append(teams[row.Team].players, playerID)
		if intIn(result.Resigned, playerID) {
			teams[row.Team].resigned++
		}
	}
	if len(teams) != 2 {
		return false
	}
	losingTeam := ""
	for team, state := range teams {
		if len(state.players) == 0 {
			continue
		}
		// CBA replays can stop when the final member of a losing side is defeated,
		// omitting that last defeat/resign. Treat all-but-one resigned on a side as
		// a team loss, not as a personal win for the last recorded survivor.
		if state.resigned >= len(state.players)-1 && state.resigned > 0 {
			if losingTeam != "" {
				return false
			}
			losingTeam = team
		}
	}
	if losingTeam == "" {
		return false
	}
	for _, row := range rows {
		won := 1
		if row.Team == losingTeam {
			won = 0
		}
		row.Won = &won
	}
	return true
}

func cbaTeam(playerID int) string {
	if playerID >= 1 && playerID <= 4 {
		return "A"
	}
	if playerID >= 5 && playerID <= 8 {
		return "B"
	}
	return ""
}

func intIn(values []int, target int) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func intPtr(value int) *int {
	v := value
	return &v
}

func floatPtr(value float64) *float64 {
	v := value
	return &v
}

func cloneIntPtr(value *int) *int {
	if value == nil {
		return nil
	}
	return intPtr(*value)
}

func setFirst(target **int, value int) {
	if *target == nil || value < **target {
		*target = intPtr(value)
	}
}

func sortPerformanceEvents(events []PerformanceEvent) {
	sort.Slice(events, func(i, j int) bool {
		if events[i].TimeMS != events[j].TimeMS {
			return events[i].TimeMS < events[j].TimeMS
		}
		if events[i].PlayerID != events[j].PlayerID {
			return events[i].PlayerID < events[j].PlayerID
		}
		return events[i].Kind < events[j].Kind
	})
}

func replayTime(ms int, fallback string) string {
	if fallback != "" {
		return fallback
	}
	return replay.FormatTime(ms)
}
