package replay

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var (
	syncLogTurnRe       = regexp.MustCompile(`BEGIN TURN\s+([0-9]+)`)
	syncLogWorldTimeRe  = regexp.MustCompile(`WORLD TIME\s+([0-9]+)`)
	syncLogPlayerRe     = regexp.MustCompile(`^Player\s+([0-9]+)\(([^)]*)\):`)
	syncLogAttrsRe      = regexp.MustCompile(`^\s+Attributes:\s+([0-9]+)\s*$`)
	syncLogAttrRe       = regexp.MustCompile(`^\s+\[([0-9]+)\]=([-0-9.]+),?\s*$`)
	syncLogObjectsRe    = regexp.MustCompile(`^\s+Objects:\s+([0-9]+)\s*$`)
	syncLogObjectRe     = regexp.MustCompile(`^\s+(.+)\[(-?[0-9]+)\], dbid=(-?[0-9]+), hp=([-0-9.]+), pos=([-0-9.]+),([-0-9.]+),([-0-9.]+), state=(-?[0-9]+), carry=(-?[0-9]+), actionType=(-?[0-9]+), actionState=(-?[0-9]+), retargetTimer=(-?[0-9]+)\s*$`)
	syncLogRNGHeaderRe  = regexp.MustCompile(`^\s*yes -\s+([A-Za-z]+) RNG - ([^@]+) @([0-9]+)\s*$`)
	syncLogSetSeedRe    = regexp.MustCompile(`^SetSeed: seed = ([0-9]+)\s*$`)
	syncLogRandRe       = regexp.MustCompile(`^Rand: #=([0-9]+)\s*$`)
	syncLogPturnRe      = regexp.MustCompile(`^\(PlayerSync\) pturn=([0-9]+)\s*$`)
	syncLogUpdatePlayRe = regexp.MustCompile(`^\(PlayerSync\) UpdatePlayer=([0-9]+)\s*$`)
)

type SyncLogOptions struct {
	ReplayPath string
	Limit      int
}

type SyncLogReport struct {
	Path         string                `json:"path,omitempty"`
	Method       string                `json:"method"`
	Verification string                `json:"verification"`
	Summary      SyncLogSummary        `json:"summary"`
	Turns        []SyncLogTurn         `json:"turns,omitempty"`
	ObjectCensus []SyncLogObjectCensus `json:"object_census,omitempty"`
	StateCensus  []SyncLogStateCensus  `json:"state_census,omitempty"`
	CorpseCarry  []SyncLogCorpseCarry  `json:"corpse_carry,omitempty"`
	CorpseDecay  []SyncLogCorpseDecay  `json:"corpse_decay_baselines,omitempty"`
	RNGStreams   []SyncLogRNGStream    `json:"rng_streams,omitempty"`
	Comparison   *SyncLogReplayCompare `json:"replay_comparison,omitempty"`
	Warnings     []string              `json:"warnings,omitempty"`
}

type SyncLogSummary struct {
	Lines             int    `json:"lines"`
	Turns             int    `json:"turns"`
	FirstTurn         int    `json:"first_turn,omitempty"`
	LastTurn          int    `json:"last_turn,omitempty"`
	FirstWorldTimeMS  int    `json:"first_world_time_ms,omitempty"`
	LastWorldTimeMS   int    `json:"last_world_time_ms,omitempty"`
	DurationMS        int    `json:"duration_ms,omitempty"`
	Duration          string `json:"duration,omitempty"`
	PlayerBlocks      int    `json:"player_blocks"`
	AttributeBlocks   int    `json:"attribute_blocks"`
	AttributeValues   int    `json:"attribute_values"`
	AttributeIndices  []int  `json:"attribute_indices,omitempty"`
	ObjectRows        int    `json:"object_rows"`
	DeclaredObjects   int    `json:"declared_objects"`
	RNGEvents         int    `json:"rng_events"`
	RNGSetSeeds       int    `json:"rng_set_seeds"`
	RNGRands          int    `json:"rng_rands"`
	UpdatePlayerLines int    `json:"update_player_lines"`
	PturnLines        int    `json:"pturn_lines"`
}

type SyncLogTurn struct {
	Turn          int                     `json:"turn"`
	WorldTimeMS   int                     `json:"world_time_ms"`
	WorldTime     string                  `json:"world_time"`
	Pturns        []int                   `json:"pturns,omitempty"`
	UpdatePlayers []int                   `json:"update_players,omitempty"`
	Players       []SyncLogPlayerSnapshot `json:"players,omitempty"`
	RNGStreams    []SyncLogRNGStream      `json:"rng_streams,omitempty"`
	ObjectCensus  []SyncLogObjectCensus   `json:"object_census,omitempty"`
}

type SyncLogPlayerSnapshot struct {
	PlayerID        int                     `json:"player_id"`
	PlayerLabel     string                  `json:"player_label"`
	Kind            string                  `json:"kind,omitempty"`
	DeclaredAttrs   int                     `json:"declared_attributes,omitempty"`
	Attributes      []SyncLogPlayerAttr     `json:"attributes,omitempty"`
	DeclaredObjects int                     `json:"declared_objects,omitempty"`
	ObjectRows      int                     `json:"object_rows"`
	Checksum        SyncLogChecksumEstimate `json:"checksum_estimate"`
	objects         []syncLogObject
}

type SyncLogChecksumEstimate struct {
	ResourceStockpileSum float64 `json:"resource_stockpile_word_1_candidate,omitempty"`
	ObjectCount          int     `json:"object_count_word_6_candidate"`
	UnitTypeSum          int64   `json:"unit_type_sum_word_2_candidate"`
	ObjectIDSum          int64   `json:"object_id_sum_word_10_candidate"`
	PositiveObjectIDs    int64   `json:"positive_object_id_sum_candidate"`
	PositionX100         int64   `json:"position_x_100_sum_candidate"`
	PositionY100         int64   `json:"position_y_100_sum_candidate"`
	PositionXY100        int64   `json:"position_xy_100_sum_candidate"`
	PositionXYZ100       int64   `json:"position_xyz_100_sum_candidate"`
	RetargetTimerSum     int64   `json:"retarget_timer_sum_candidate"`
	StateSum             int64   `json:"state_sum_candidate"`
	CarrySum             int64   `json:"carry_sum_candidate"`
	ActionTypeSum        int64   `json:"action_type_sum_candidate"`
	ActionTypeNonNegSum  int64   `json:"action_type_non_negative_sum_candidate"`
	ActionStateSum       int64   `json:"action_state_sum_candidate"`
	HP100Sum             int64   `json:"hp_100_sum_candidate"`
}

type SyncLogPlayerAttr struct {
	Index int     `json:"index"`
	Value float64 `json:"value"`
}

type SyncLogObjectCensus struct {
	Name  string `json:"name"`
	DBID  int    `json:"dbid"`
	Count int    `json:"count"`
}

type SyncLogStateCensus struct {
	State      int      `json:"state"`
	Rows       int      `json:"rows"`
	HPZeroRows int      `json:"hp_zero_rows,omitempty"`
	CarrySum   int64    `json:"carry_sum"`
	Names      []string `json:"sample_names,omitempty"`
}

type SyncLogCorpseCarry struct {
	PlayerID       int     `json:"player_id"`
	ObjectID       int     `json:"object_id"`
	UnitID         int     `json:"unit_id"`
	UnitName       string  `json:"unit_name"`
	Samples        int     `json:"samples"`
	FirstTimeMS    int     `json:"first_time_ms"`
	FirstTime      string  `json:"first_time"`
	LastTimeMS     int     `json:"last_time_ms"`
	LastTime       string  `json:"last_time"`
	FirstCarry     int     `json:"first_carry"`
	LastCarry      int     `json:"last_carry"`
	CarryDrop      int     `json:"carry_drop"`
	ObservedRatePS float64 `json:"observed_rate_per_second,omitempty"`
}

type SyncLogCorpseDecay struct {
	UnitID              int     `json:"unit_id"`
	UnitName            string  `json:"unit_name"`
	Tracks              int     `json:"tracks"`
	MovingTracks        int     `json:"moving_tracks"`
	Samples             int     `json:"samples"`
	FirstCarryMin       int     `json:"first_carry_min,omitempty"`
	FirstCarryMax       int     `json:"first_carry_max,omitempty"`
	LastCarryMin        int     `json:"last_carry_min,omitempty"`
	LastCarryMax        int     `json:"last_carry_max,omitempty"`
	CarryDropTotal      int     `json:"carry_drop_total"`
	DurationMSTotal     int     `json:"duration_ms_total"`
	ObservedRatePS      float64 `json:"observed_rate_per_second,omitempty"`
	ObservedRateMinPS   float64 `json:"observed_rate_min_per_second,omitempty"`
	ObservedRateMaxPS   float64 `json:"observed_rate_max_per_second,omitempty"`
	EstimatedLifetimeMS int     `json:"estimated_lifetime_ms,omitempty"`
	EstimatedLifetime   string  `json:"estimated_lifetime,omitempty"`
	Confidence          string  `json:"confidence"`
}

type SyncLogRNGStream struct {
	Stream   string           `json:"stream"`
	SetSeeds int              `json:"set_seeds"`
	Rands    int              `json:"rands"`
	Events   int              `json:"events"`
	Sites    []SyncLogRNGSite `json:"sites,omitempty"`
}

type SyncLogRNGSite struct {
	Operation string `json:"operation"`
	Line      int    `json:"line"`
	Count     int    `json:"count"`
}

type SyncLogReplayCompare struct {
	ReplayPath              string               `json:"replay_path"`
	ChecksumSamples         int                  `json:"checksum_samples"`
	ReplayFirstTimeMS       int                  `json:"replay_first_time_ms,omitempty"`
	ReplayLastTimeMS        int                  `json:"replay_last_time_ms,omitempty"`
	LogFirstWorldTimeMS     int                  `json:"log_first_world_time_ms,omitempty"`
	LogLastWorldTimeMS      int                  `json:"log_last_world_time_ms,omitempty"`
	OverlappingSamples      int                  `json:"overlapping_samples"`
	Overlap                 bool                 `json:"overlap"`
	ExactWorldTimeMatches   []SyncLogReplayMatch `json:"exact_world_time_matches,omitempty"`
	NearestBeforeLogStartMS int                  `json:"nearest_before_log_start_ms,omitempty"`
	NearestBeforeLogStart   string               `json:"nearest_before_log_start,omitempty"`
	NearestAfterLogEndMS    int                  `json:"nearest_after_log_end_ms,omitempty"`
	NearestAfterLogEnd      string               `json:"nearest_after_log_end,omitempty"`
	NearestBeforeToFirstLog *SyncLogReplayApprox `json:"nearest_before_to_first_log,omitempty"`
	Warning                 string               `json:"warning,omitempty"`
}

type SyncLogReplayMatch struct {
	TimeMS  int                          `json:"time_ms"`
	Time    string                       `json:"time"`
	Players []SyncLogReplayPlayerCompare `json:"players,omitempty"`
}

type SyncLogReplayApprox struct {
	ReplayTimeMS   int                          `json:"replay_time_ms"`
	ReplayTime     string                       `json:"replay_time"`
	LogWorldTimeMS int                          `json:"log_world_time_ms"`
	LogWorldTime   string                       `json:"log_world_time"`
	GapMS          int                          `json:"gap_ms"`
	Gap            string                       `json:"gap"`
	Players        []SyncLogReplayPlayerCompare `json:"players,omitempty"`
	Confidence     string                       `json:"confidence"`
}

type SyncLogReplayPlayerCompare struct {
	PlayerID          int                     `json:"player_id"`
	LogResourceSum    float64                 `json:"log_resource_stockpile_word_1_candidate,omitempty"`
	ReplayWord1       uint32                  `json:"replay_word_1"`
	ResourceSumDiff   float64                 `json:"resource_stockpile_diff,omitempty"`
	LogObjectCount    int                     `json:"log_object_count_word_6_candidate"`
	ReplayWord6       uint32                  `json:"replay_word_6"`
	ObjectCountDiff   int64                   `json:"object_count_diff"`
	LogUnitTypeSum    int64                   `json:"log_unit_type_sum_word_2_candidate"`
	ReplayWord2       uint32                  `json:"replay_word_2"`
	UnitTypeSumDiff   int64                   `json:"unit_type_sum_diff"`
	LogStateSum       int64                   `json:"log_state_sum_word_3_candidate"`
	ReplayWord3       uint32                  `json:"replay_word_3"`
	StateSumDiff      int64                   `json:"state_sum_diff"`
	ReplayWord4       uint32                  `json:"replay_word_4"`
	Word4PerObject    float64                 `json:"word_4_per_log_object"`
	Word4Candidates   []SyncLogWord4Candidate `json:"word_4_candidates,omitempty"`
	LogPositionXY100  int64                   `json:"log_position_xy100_word_7_candidate"`
	ReplayWord7       uint32                  `json:"replay_word_7"`
	PositionXY100Diff int64                   `json:"position_xy100_diff"`
	LogObjectIDSum    int64                   `json:"log_object_id_sum_word_10_candidate"`
	ReplayWord10      uint32                  `json:"replay_word_10"`
	ObjectIDSumDiff   int64                   `json:"object_id_sum_diff"`
	Checksum          SyncLogChecksumEstimate `json:"log_checksum_estimate"`
}

type SyncLogWord4Candidate struct {
	Name        string `json:"name"`
	LogValue    int64  `json:"log_value"`
	ReplayWord4 uint32 `json:"replay_word_4"`
	Diff        int64  `json:"diff"`
}

type syncLogObject struct {
	Name          string
	InstanceID    int
	DBID          int
	HP            float64
	X             float64
	Y             float64
	Z             float64
	State         int
	Carry         int
	ActionType    int
	ActionState   int
	RetargetTimer int
}

type syncLogStateAccumulator struct {
	State      int
	Rows       int
	HPZeroRows int
	CarrySum   int64
	names      map[string]bool
}

type syncLogCorpseTrack struct {
	PlayerID   int
	ObjectID   int
	UnitID     int
	UnitName   string
	Samples    int
	FirstMS    int
	LastMS     int
	FirstCarry int
	LastCarry  int
}

type syncLogRNGEvent struct {
	Turn      int
	Stream    string
	Operation string
	Line      int
	Kind      string
	Value     uint64
}

func BuildSyncLogReport(path string, opts SyncLogOptions) (*SyncLogReport, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	report := &SyncLogReport{
		Path:         path,
		Method:       "aoe2de_p0_sync_text_log",
		Verification: "engine_log_structured_text_parse_not_replay_stream",
	}

	var turns []SyncLogTurn
	var currentTurn *SyncLogTurn
	var currentPlayer *SyncLogPlayerSnapshot
	var pendingRNG *syncLogRNGEvent
	var rngEvents []syncLogRNGEvent
	objectCensus := map[string]SyncLogObjectCensus{}
	stateCensus := map[int]*syncLogStateAccumulator{}
	corpseTracks := map[string]*syncLogCorpseTrack{}
	attributeIndices := map[int]bool{}

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		report.Summary.Lines++
		line := scanner.Text()

		if m := syncLogTurnRe.FindStringSubmatch(line); m != nil {
			turn, _ := strconv.Atoi(m[1])
			turns = append(turns, SyncLogTurn{Turn: turn})
			currentTurn = &turns[len(turns)-1]
			currentPlayer = nil
			report.Summary.Turns++
			if report.Summary.FirstTurn == 0 {
				report.Summary.FirstTurn = turn
			}
			report.Summary.LastTurn = turn
			continue
		}
		if m := syncLogWorldTimeRe.FindStringSubmatch(line); m != nil && currentTurn != nil {
			worldTime, _ := strconv.Atoi(m[1])
			currentTurn.WorldTimeMS = worldTime
			currentTurn.WorldTime = FormatTime(worldTime)
			if report.Summary.FirstWorldTimeMS == 0 {
				report.Summary.FirstWorldTimeMS = worldTime
			}
			report.Summary.LastWorldTimeMS = worldTime
			continue
		}
		if m := syncLogPturnRe.FindStringSubmatch(line); m != nil && currentTurn != nil {
			pturn, _ := strconv.Atoi(m[1])
			currentTurn.Pturns = append(currentTurn.Pturns, pturn)
			report.Summary.PturnLines++
			continue
		}
		if m := syncLogUpdatePlayRe.FindStringSubmatch(line); m != nil && currentTurn != nil {
			player, _ := strconv.Atoi(m[1])
			currentTurn.UpdatePlayers = append(currentTurn.UpdatePlayers, player)
			report.Summary.UpdatePlayerLines++
			continue
		}
		if m := syncLogRNGHeaderRe.FindStringSubmatch(line); m != nil {
			rngLine, _ := strconv.Atoi(m[3])
			pendingRNG = &syncLogRNGEvent{
				Stream:    strings.TrimSpace(m[1]),
				Operation: strings.TrimSpace(m[2]),
				Line:      rngLine,
			}
			if currentTurn != nil {
				pendingRNG.Turn = currentTurn.Turn
			}
			continue
		}
		if pendingRNG != nil {
			if m := syncLogSetSeedRe.FindStringSubmatch(line); m != nil {
				value, _ := strconv.ParseUint(m[1], 10, 64)
				pendingRNG.Kind = "set_seed"
				pendingRNG.Value = value
				rngEvents = append(rngEvents, *pendingRNG)
				report.Summary.RNGEvents++
				report.Summary.RNGSetSeeds++
				pendingRNG = nil
				continue
			}
			if m := syncLogRandRe.FindStringSubmatch(line); m != nil {
				value, _ := strconv.ParseUint(m[1], 10, 64)
				pendingRNG.Kind = "rand"
				pendingRNG.Value = value
				rngEvents = append(rngEvents, *pendingRNG)
				report.Summary.RNGEvents++
				report.Summary.RNGRands++
				pendingRNG = nil
				continue
			}
		}
		if m := syncLogPlayerRe.FindStringSubmatch(line); m != nil && currentTurn != nil {
			playerID, _ := strconv.Atoi(m[1])
			currentTurn.Players = append(currentTurn.Players, SyncLogPlayerSnapshot{
				PlayerID:    playerID,
				PlayerLabel: fmt.Sprintf("P%d", playerID),
				Kind:        m[2],
			})
			currentPlayer = &currentTurn.Players[len(currentTurn.Players)-1]
			report.Summary.PlayerBlocks++
			continue
		}
		if m := syncLogAttrsRe.FindStringSubmatch(line); m != nil && currentPlayer != nil {
			declared, _ := strconv.Atoi(m[1])
			currentPlayer.DeclaredAttrs = declared
			report.Summary.AttributeBlocks++
			continue
		}
		if m := syncLogAttrRe.FindStringSubmatch(line); m != nil && currentPlayer != nil {
			index, _ := strconv.Atoi(m[1])
			value, err := strconv.ParseFloat(m[2], 64)
			if err != nil {
				report.Warnings = append(report.Warnings, fmt.Sprintf("bad player attribute value index=%d value=%q", index, m[2]))
				continue
			}
			currentPlayer.Attributes = append(currentPlayer.Attributes, SyncLogPlayerAttr{Index: index, Value: value})
			report.Summary.AttributeValues++
			attributeIndices[index] = true
			addSyncLogAttributeEstimate(&currentPlayer.Checksum, index, value)
			continue
		}
		if m := syncLogObjectsRe.FindStringSubmatch(line); m != nil && currentPlayer != nil {
			declared, _ := strconv.Atoi(m[1])
			currentPlayer.DeclaredObjects = declared
			report.Summary.DeclaredObjects += declared
			continue
		}
		if m := syncLogObjectRe.FindStringSubmatch(line); m != nil && currentPlayer != nil {
			obj, err := parseSyncLogObject(m)
			if err != nil {
				report.Warnings = append(report.Warnings, err.Error())
				continue
			}
			currentPlayer.ObjectRows++
			report.Summary.ObjectRows++
			addSyncLogObjectEstimate(&currentPlayer.Checksum, obj)
			currentPlayer.objects = append(currentPlayer.objects, obj)
			addSyncLogStateCensus(stateCensus, obj)
			if isSyncLogCorpseLike(obj) {
				addSyncLogCorpseTrack(corpseTracks, *currentTurn, currentPlayer.PlayerID, obj)
			}
			key := fmt.Sprintf("%s\x00%d", obj.Name, obj.DBID)
			census := objectCensus[key]
			if census.Name == "" {
				census.Name = obj.Name
				census.DBID = obj.DBID
			}
			census.Count++
			objectCensus[key] = census
			continue
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if report.Summary.FirstWorldTimeMS != 0 && report.Summary.LastWorldTimeMS >= report.Summary.FirstWorldTimeMS {
		report.Summary.DurationMS = report.Summary.LastWorldTimeMS - report.Summary.FirstWorldTimeMS
		report.Summary.Duration = FormatTime(report.Summary.DurationMS)
	}
	report.Summary.AttributeIndices = syncLogSortedIndices(attributeIndices)
	for i := range turns {
		turns[i].RNGStreams = summarizeSyncLogRNGStreams(filterSyncLogRNGByTurn(rngEvents, turns[i].Turn), opts.Limit)
		turns[i].ObjectCensus = summarizeSyncLogCensus(turns[i].Players, opts.Limit)
	}
	report.Turns = turns
	report.RNGStreams = summarizeSyncLogRNGStreams(rngEvents, opts.Limit)
	report.ObjectCensus = syncLogObjectCensusList(objectCensus, opts.Limit)
	report.StateCensus = syncLogStateCensusList(stateCensus, opts.Limit)
	report.CorpseCarry = syncLogCorpseCarryList(corpseTracks, opts.Limit)
	report.CorpseDecay = syncLogCorpseDecayList(corpseTracks, opts.Limit)
	if opts.ReplayPath != "" {
		comparison, err := compareSyncLogToReplay(report, opts.ReplayPath)
		if err != nil {
			report.Warnings = append(report.Warnings, "replay comparison unavailable: "+err.Error())
		} else {
			report.Comparison = comparison
		}
	}
	return report, nil
}

func addSyncLogAttributeEstimate(est *SyncLogChecksumEstimate, index int, value float64) {
	if index >= 0 && index <= 3 {
		est.ResourceStockpileSum += value
	}
}

func parseSyncLogObject(match []string) (syncLogObject, error) {
	intAt := func(i int) (int, error) {
		return strconv.Atoi(match[i])
	}
	floatAt := func(i int) (float64, error) {
		return strconv.ParseFloat(match[i], 64)
	}
	instanceID, err := intAt(2)
	if err != nil {
		return syncLogObject{}, fmt.Errorf("bad object instance id %q", match[2])
	}
	dbid, err := intAt(3)
	if err != nil {
		return syncLogObject{}, fmt.Errorf("bad object dbid %q", match[3])
	}
	hp, err := floatAt(4)
	if err != nil {
		return syncLogObject{}, fmt.Errorf("bad object hp %q", match[4])
	}
	x, err := floatAt(5)
	if err != nil {
		return syncLogObject{}, fmt.Errorf("bad object x %q", match[5])
	}
	y, err := floatAt(6)
	if err != nil {
		return syncLogObject{}, fmt.Errorf("bad object y %q", match[6])
	}
	z, err := floatAt(7)
	if err != nil {
		return syncLogObject{}, fmt.Errorf("bad object z %q", match[7])
	}
	state, err := intAt(8)
	if err != nil {
		return syncLogObject{}, fmt.Errorf("bad object state %q", match[8])
	}
	carry, err := intAt(9)
	if err != nil {
		return syncLogObject{}, fmt.Errorf("bad object carry %q", match[9])
	}
	actionType, err := intAt(10)
	if err != nil {
		return syncLogObject{}, fmt.Errorf("bad object actionType %q", match[10])
	}
	actionState, err := intAt(11)
	if err != nil {
		return syncLogObject{}, fmt.Errorf("bad object actionState %q", match[11])
	}
	retargetTimer, err := intAt(12)
	if err != nil {
		return syncLogObject{}, fmt.Errorf("bad object retargetTimer %q", match[12])
	}
	return syncLogObject{
		Name:          strings.TrimSpace(match[1]),
		InstanceID:    instanceID,
		DBID:          dbid,
		HP:            hp,
		X:             x,
		Y:             y,
		Z:             z,
		State:         state,
		Carry:         carry,
		ActionType:    actionType,
		ActionState:   actionState,
		RetargetTimer: retargetTimer,
	}, nil
}

func addSyncLogObjectEstimate(est *SyncLogChecksumEstimate, obj syncLogObject) {
	est.ObjectCount++
	est.UnitTypeSum += int64(obj.DBID)
	est.ObjectIDSum += int64(obj.InstanceID)
	if obj.InstanceID > 0 {
		est.PositiveObjectIDs += int64(obj.InstanceID)
	}
	est.PositionX100 += int64(obj.X * 100)
	est.PositionY100 += int64(obj.Y * 100)
	est.PositionXY100 += int64((obj.X + obj.Y) * 100)
	est.PositionXYZ100 += int64((obj.X + obj.Y + obj.Z) * 100)
	est.RetargetTimerSum += int64(obj.RetargetTimer)
	est.StateSum += int64(obj.State)
	est.CarrySum += int64(obj.Carry)
	est.ActionTypeSum += int64(obj.ActionType)
	if obj.ActionType > 0 {
		est.ActionTypeNonNegSum += int64(obj.ActionType)
	}
	est.ActionStateSum += int64(obj.ActionState)
	est.HP100Sum += int64(obj.HP * 100)
}

func addSyncLogStateCensus(census map[int]*syncLogStateAccumulator, obj syncLogObject) {
	row := census[obj.State]
	if row == nil {
		row = &syncLogStateAccumulator{State: obj.State, names: map[string]bool{}}
		census[obj.State] = row
	}
	row.Rows++
	if obj.HP == 0 {
		row.HPZeroRows++
	}
	row.CarrySum += int64(obj.Carry)
	if len(row.names) < 12 {
		row.names[obj.Name] = true
	}
}

func isSyncLogCorpseLike(obj syncLogObject) bool {
	name := strings.ToLower(obj.Name)
	return obj.HP == 0 && (strings.HasSuffix(name, "_d") || strings.Contains(name, "rubble"))
}

func addSyncLogCorpseTrack(tracks map[string]*syncLogCorpseTrack, turn SyncLogTurn, playerID int, obj syncLogObject) {
	key := fmt.Sprintf("%d\x00%d\x00%d", playerID, obj.InstanceID, obj.DBID)
	track := tracks[key]
	if track == nil {
		track = &syncLogCorpseTrack{
			PlayerID:   playerID,
			ObjectID:   obj.InstanceID,
			UnitID:     obj.DBID,
			UnitName:   obj.Name,
			FirstMS:    turn.WorldTimeMS,
			LastMS:     turn.WorldTimeMS,
			FirstCarry: obj.Carry,
			LastCarry:  obj.Carry,
		}
		tracks[key] = track
	}
	track.Samples++
	if turn.WorldTimeMS < track.FirstMS || track.Samples == 1 {
		track.FirstMS = turn.WorldTimeMS
		track.FirstCarry = obj.Carry
	}
	if turn.WorldTimeMS >= track.LastMS {
		track.LastMS = turn.WorldTimeMS
		track.LastCarry = obj.Carry
	}
}

func syncLogStateCensusList(census map[int]*syncLogStateAccumulator, limit int) []SyncLogStateCensus {
	out := make([]SyncLogStateCensus, 0, len(census))
	for _, row := range census {
		names := make([]string, 0, len(row.names))
		for name := range row.names {
			names = append(names, name)
		}
		sort.Strings(names)
		out = append(out, SyncLogStateCensus{
			State:      row.State,
			Rows:       row.Rows,
			HPZeroRows: row.HPZeroRows,
			CarrySum:   row.CarrySum,
			Names:      names,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Rows != out[j].Rows {
			return out[i].Rows > out[j].Rows
		}
		return out[i].State < out[j].State
	})
	if limit > 0 && len(out) > limit {
		return out[:limit]
	}
	return out
}

func syncLogCorpseCarryList(tracks map[string]*syncLogCorpseTrack, limit int) []SyncLogCorpseCarry {
	out := make([]SyncLogCorpseCarry, 0, len(tracks))
	for _, track := range tracks {
		drop := track.FirstCarry - track.LastCarry
		durationMS := track.LastMS - track.FirstMS
		var rate float64
		if durationMS > 0 {
			rate = float64(drop) * 1000 / float64(durationMS)
		}
		out = append(out, SyncLogCorpseCarry{
			PlayerID:       track.PlayerID,
			ObjectID:       track.ObjectID,
			UnitID:         track.UnitID,
			UnitName:       track.UnitName,
			Samples:        track.Samples,
			FirstTimeMS:    track.FirstMS,
			FirstTime:      FormatTime(track.FirstMS),
			LastTimeMS:     track.LastMS,
			LastTime:       FormatTime(track.LastMS),
			FirstCarry:     track.FirstCarry,
			LastCarry:      track.LastCarry,
			CarryDrop:      drop,
			ObservedRatePS: rate,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Samples != out[j].Samples {
			return out[i].Samples > out[j].Samples
		}
		if out[i].CarryDrop != out[j].CarryDrop {
			return out[i].CarryDrop > out[j].CarryDrop
		}
		if out[i].PlayerID != out[j].PlayerID {
			return out[i].PlayerID < out[j].PlayerID
		}
		return out[i].ObjectID < out[j].ObjectID
	})
	if limit > 0 && len(out) > limit {
		return out[:limit]
	}
	return out
}

type syncLogCorpseDecayAccumulator struct {
	UnitID          int
	UnitName        string
	Tracks          int
	MovingTracks    int
	Samples         int
	FirstCarryMin   int
	FirstCarryMax   int
	LastCarryMin    int
	LastCarryMax    int
	CarryDropTotal  int
	DurationMSTotal int
	RateMin         float64
	RateMax         float64
}

func syncLogCorpseDecayList(tracks map[string]*syncLogCorpseTrack, limit int) []SyncLogCorpseDecay {
	byUnit := map[string]*syncLogCorpseDecayAccumulator{}
	for _, track := range tracks {
		key := fmt.Sprintf("%d\x00%s", track.UnitID, track.UnitName)
		row := byUnit[key]
		if row == nil {
			row = &syncLogCorpseDecayAccumulator{
				UnitID:        track.UnitID,
				UnitName:      track.UnitName,
				FirstCarryMin: track.FirstCarry,
				FirstCarryMax: track.FirstCarry,
				LastCarryMin:  track.LastCarry,
				LastCarryMax:  track.LastCarry,
			}
			byUnit[key] = row
		}
		row.Tracks++
		row.Samples += track.Samples
		if track.FirstCarry < row.FirstCarryMin {
			row.FirstCarryMin = track.FirstCarry
		}
		if track.FirstCarry > row.FirstCarryMax {
			row.FirstCarryMax = track.FirstCarry
		}
		if track.LastCarry < row.LastCarryMin {
			row.LastCarryMin = track.LastCarry
		}
		if track.LastCarry > row.LastCarryMax {
			row.LastCarryMax = track.LastCarry
		}
		drop := track.FirstCarry - track.LastCarry
		durationMS := track.LastMS - track.FirstMS
		if durationMS > 0 && drop > 0 {
			rate := float64(drop) * 1000 / float64(durationMS)
			row.MovingTracks++
			row.CarryDropTotal += drop
			row.DurationMSTotal += durationMS
			if row.RateMin == 0 || rate < row.RateMin {
				row.RateMin = rate
			}
			if rate > row.RateMax {
				row.RateMax = rate
			}
		}
	}
	out := make([]SyncLogCorpseDecay, 0, len(byUnit))
	for _, row := range byUnit {
		var rate float64
		var lifetimeMS int
		lifetime := ""
		confidence := "observed_static_or_single_sample_no_decay_rate"
		if row.DurationMSTotal > 0 && row.CarryDropTotal > 0 {
			rate = float64(row.CarryDropTotal) * 1000 / float64(row.DurationMSTotal)
			if rate > 0 && row.FirstCarryMax > 0 {
				lifetimeMS = int(float64(row.FirstCarryMax) * 1000 / rate)
				lifetime = FormatTime(lifetimeMS)
			}
			confidence = "engine_log_observed_carry_decay_rate"
			if row.MovingTracks > 1 {
				confidence = "engine_log_observed_multi_track_carry_decay_rate"
			}
		}
		out = append(out, SyncLogCorpseDecay{
			UnitID:              row.UnitID,
			UnitName:            row.UnitName,
			Tracks:              row.Tracks,
			MovingTracks:        row.MovingTracks,
			Samples:             row.Samples,
			FirstCarryMin:       row.FirstCarryMin,
			FirstCarryMax:       row.FirstCarryMax,
			LastCarryMin:        row.LastCarryMin,
			LastCarryMax:        row.LastCarryMax,
			CarryDropTotal:      row.CarryDropTotal,
			DurationMSTotal:     row.DurationMSTotal,
			ObservedRatePS:      rate,
			ObservedRateMinPS:   row.RateMin,
			ObservedRateMaxPS:   row.RateMax,
			EstimatedLifetimeMS: lifetimeMS,
			EstimatedLifetime:   lifetime,
			Confidence:          confidence,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].MovingTracks != out[j].MovingTracks {
			return out[i].MovingTracks > out[j].MovingTracks
		}
		if out[i].Tracks != out[j].Tracks {
			return out[i].Tracks > out[j].Tracks
		}
		if out[i].Samples != out[j].Samples {
			return out[i].Samples > out[j].Samples
		}
		if out[i].UnitName != out[j].UnitName {
			return out[i].UnitName < out[j].UnitName
		}
		return out[i].UnitID < out[j].UnitID
	})
	if limit > 0 && len(out) > limit {
		return out[:limit]
	}
	return out
}

func summarizeSyncLogCensus(players []SyncLogPlayerSnapshot, limit int) []SyncLogObjectCensus {
	census := map[string]SyncLogObjectCensus{}
	_ = players
	return syncLogObjectCensusList(census, limit)
}

func syncLogObjectCensusList(census map[string]SyncLogObjectCensus, limit int) []SyncLogObjectCensus {
	out := make([]SyncLogObjectCensus, 0, len(census))
	for _, row := range census {
		out = append(out, row)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		if out[i].Name != out[j].Name {
			return out[i].Name < out[j].Name
		}
		return out[i].DBID < out[j].DBID
	})
	if limit > 0 && len(out) > limit {
		return out[:limit]
	}
	return out
}

func syncLogSortedIndices(values map[int]bool) []int {
	out := make([]int, 0, len(values))
	for value := range values {
		out = append(out, value)
	}
	sort.Ints(out)
	return out
}

func filterSyncLogRNGByTurn(events []syncLogRNGEvent, turn int) []syncLogRNGEvent {
	var out []syncLogRNGEvent
	for _, event := range events {
		if event.Turn == turn {
			out = append(out, event)
		}
	}
	return out
}

func summarizeSyncLogRNGStreams(events []syncLogRNGEvent, limit int) []SyncLogRNGStream {
	byStream := map[string]*SyncLogRNGStream{}
	siteCounts := map[string]map[string]*SyncLogRNGSite{}
	for _, event := range events {
		stream := byStream[event.Stream]
		if stream == nil {
			stream = &SyncLogRNGStream{Stream: event.Stream}
			byStream[event.Stream] = stream
			siteCounts[event.Stream] = map[string]*SyncLogRNGSite{}
		}
		stream.Events++
		if event.Kind == "set_seed" {
			stream.SetSeeds++
		} else if event.Kind == "rand" {
			stream.Rands++
		}
		siteKey := fmt.Sprintf("%s\x00%d", event.Operation, event.Line)
		site := siteCounts[event.Stream][siteKey]
		if site == nil {
			site = &SyncLogRNGSite{Operation: event.Operation, Line: event.Line}
			siteCounts[event.Stream][siteKey] = site
		}
		site.Count++
	}
	streams := make([]SyncLogRNGStream, 0, len(byStream))
	for _, stream := range byStream {
		sites := make([]SyncLogRNGSite, 0, len(siteCounts[stream.Stream]))
		for _, site := range siteCounts[stream.Stream] {
			sites = append(sites, *site)
		}
		sort.Slice(sites, func(i, j int) bool {
			if sites[i].Count != sites[j].Count {
				return sites[i].Count > sites[j].Count
			}
			if sites[i].Operation != sites[j].Operation {
				return sites[i].Operation < sites[j].Operation
			}
			return sites[i].Line < sites[j].Line
		})
		if limit > 0 && len(sites) > limit {
			sites = sites[:limit]
		}
		stream.Sites = sites
		streams = append(streams, *stream)
	}
	sort.Slice(streams, func(i, j int) bool {
		if streams[i].Events != streams[j].Events {
			return streams[i].Events > streams[j].Events
		}
		return streams[i].Stream < streams[j].Stream
	})
	return streams
}

func compareSyncLogToReplay(report *SyncLogReport, replayPath string) (*SyncLogReplayCompare, error) {
	sync, err := BuildSyncStream(replayPath, SyncOptions{ChecksumsOnly: true})
	if err != nil {
		return nil, err
	}
	comparison := &SyncLogReplayCompare{
		ReplayPath:          replayPath,
		ChecksumSamples:     sync.Summary.ChecksumDE,
		LogFirstWorldTimeMS: report.Summary.FirstWorldTimeMS,
		LogLastWorldTimeMS:  report.Summary.LastWorldTimeMS,
	}
	if len(sync.Events) > 0 {
		comparison.ReplayFirstTimeMS = sync.Events[0].TimeMS
		comparison.ReplayLastTimeMS = sync.Events[len(sync.Events)-1].TimeMS
	}
	logTurnsByTime := map[int]SyncLogTurn{}
	for _, turn := range report.Turns {
		if turn.WorldTimeMS != 0 {
			logTurnsByTime[turn.WorldTimeMS] = turn
		}
	}
	var nearestBefore *SyncEvent
	for _, event := range sync.Events {
		if event.Form != "checksum_de" {
			continue
		}
		if event.TimeMS >= report.Summary.FirstWorldTimeMS && event.TimeMS <= report.Summary.LastWorldTimeMS {
			comparison.OverlappingSamples++
			comparison.Overlap = true
		}
		if event.TimeMS < report.Summary.FirstWorldTimeMS {
			comparison.NearestBeforeLogStartMS = event.TimeMS
			comparison.NearestBeforeLogStart = event.Time
			copyEvent := event
			nearestBefore = &copyEvent
		}
		if comparison.NearestAfterLogEndMS == 0 && event.TimeMS > report.Summary.LastWorldTimeMS {
			comparison.NearestAfterLogEndMS = event.TimeMS
			comparison.NearestAfterLogEnd = event.Time
		}
		turn, ok := logTurnsByTime[event.TimeMS]
		if !ok {
			continue
		}
		match := SyncLogReplayMatch{TimeMS: event.TimeMS, Time: event.Time}
		match.Players = compareSyncLogPlayersToEvent(turn.Players, event)
		comparison.ExactWorldTimeMatches = append(comparison.ExactWorldTimeMatches, match)
	}
	if nearestBefore != nil && len(report.Turns) > 0 {
		firstTurn := report.Turns[0]
		if firstTurn.WorldTimeMS != 0 {
			gapMS := firstTurn.WorldTimeMS - nearestBefore.TimeMS
			comparison.NearestBeforeToFirstLog = &SyncLogReplayApprox{
				ReplayTimeMS:   nearestBefore.TimeMS,
				ReplayTime:     nearestBefore.Time,
				LogWorldTimeMS: firstTurn.WorldTimeMS,
				LogWorldTime:   firstTurn.WorldTime,
				GapMS:          gapMS,
				Gap:            FormatTime(gapMS),
				Players:        compareSyncLogPlayersToEvent(firstTurn.Players, *nearestBefore),
				Confidence:     "rough_sanity_check_only_not_exact_overlap",
			}
		}
	}
	if !comparison.Overlap {
		comparison.Warning = "no replay checksum sample overlaps the sync-log world-time window"
	}
	return comparison, nil
}

func compareSyncLogPlayersToEvent(players []SyncLogPlayerSnapshot, event SyncEvent) []SyncLogReplayPlayerCompare {
	var out []SyncLogReplayPlayerCompare
	for _, player := range players {
		if player.PlayerID < 1 || player.PlayerID > len(event.Matrix) || len(event.Matrix[player.PlayerID-1]) < 11 {
			continue
		}
		words := event.Matrix[player.PlayerID-1]
		out = append(out, SyncLogReplayPlayerCompare{
			PlayerID:          player.PlayerID,
			LogResourceSum:    player.Checksum.ResourceStockpileSum,
			ReplayWord1:       words[1],
			ResourceSumDiff:   player.Checksum.ResourceStockpileSum - float64(words[1]),
			LogObjectCount:    player.Checksum.ObjectCount,
			ReplayWord6:       words[6],
			ObjectCountDiff:   int64(player.Checksum.ObjectCount) - int64(words[6]),
			LogUnitTypeSum:    player.Checksum.UnitTypeSum,
			ReplayWord2:       words[2],
			UnitTypeSumDiff:   player.Checksum.UnitTypeSum - int64(words[2]),
			LogStateSum:       player.Checksum.StateSum,
			ReplayWord3:       words[3],
			StateSumDiff:      player.Checksum.StateSum - int64(words[3]),
			ReplayWord4:       words[4],
			Word4PerObject:    syncLogRatio(float64(words[4]), float64(player.Checksum.ObjectCount)),
			Word4Candidates:   syncLogWord4Candidates(player.Checksum, words[4]),
			LogPositionXY100:  player.Checksum.PositionXY100,
			ReplayWord7:       words[7],
			PositionXY100Diff: player.Checksum.PositionXY100 - int64(words[7]),
			LogObjectIDSum:    player.Checksum.ObjectIDSum,
			ReplayWord10:      words[10],
			ObjectIDSumDiff:   player.Checksum.ObjectIDSum - int64(words[10]),
			Checksum:          player.Checksum,
		})
	}
	return out
}

func syncLogRatio(numerator, denominator float64) float64 {
	if denominator == 0 {
		return 0
	}
	return numerator / denominator
}

func syncLogWord4Candidates(checksum SyncLogChecksumEstimate, replayWord4 uint32) []SyncLogWord4Candidate {
	values := []struct {
		name  string
		value int64
	}{
		{name: "state_sum", value: checksum.StateSum},
		{name: "action_state_sum", value: checksum.ActionStateSum},
		{name: "action_type_sum", value: checksum.ActionTypeSum},
		{name: "action_type_non_negative_sum", value: checksum.ActionTypeNonNegSum},
		{name: "carry_sum", value: checksum.CarrySum},
		{name: "retarget_timer_sum", value: checksum.RetargetTimerSum},
		{name: "hp100_sum", value: checksum.HP100Sum},
	}
	out := make([]SyncLogWord4Candidate, 0, len(values))
	for _, candidate := range values {
		out = append(out, SyncLogWord4Candidate{
			Name:        candidate.name,
			LogValue:    candidate.value,
			ReplayWord4: replayWord4,
			Diff:        candidate.value - int64(replayWord4),
		})
	}
	return out
}
