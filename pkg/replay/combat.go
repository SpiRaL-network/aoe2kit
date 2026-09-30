package replay

import (
	"sort"
)

const (
	defaultCombatWindowMS     = 5000
	defaultCombatLimit        = 40
	defaultCombatMinNetLoss   = 1
	defaultPressurePerPulse   = 8
	combatVerification        = "structure_verified_death_replacement_and_net_lifecycle_loss_not_kill_attribution"
	combatCandidateConfidence = "sync_net_object_loss_plus_nearby_target_command_candidate"
)

type CombatOptions struct {
	WindowMS   int
	Limit      int
	MinNetLoss int
	Sort       string
}

type CombatReport struct {
	Path         string                `json:"path,omitempty"`
	Method       string                `json:"method"`
	Verification string                `json:"verification"`
	Summary      CombatSummary         `json:"summary"`
	Players      []CombatPlayerSummary `json:"players,omitempty"`
	Targets      []CombatTargetSummary `json:"targets,omitempty"`
	DeathEvents  []CombatDeathEvent    `json:"death_events,omitempty"`
	LossPulses   []CombatLossPulse     `json:"loss_pulses,omitempty"`
	Warnings     []string              `json:"warnings,omitempty"`
}

type CombatSummary struct {
	ChecksumSamples        int    `json:"checksum_samples"`
	StateDeltas            int    `json:"state_deltas"`
	LossPulses             int    `json:"loss_pulses"`
	ShownLossPulses        int    `json:"shown_loss_pulses"`
	DeathReplacements      int    `json:"death_replacements"`
	ShownDeathReplacements int    `json:"shown_death_replacements"`
	NetObjectsLost         int    `json:"net_objects_lost"`
	SingleObjectRemovals   int    `json:"single_object_removals"`
	FilteredArtifacts      int    `json:"filtered_checksum_artifacts"`
	PressureLinks          int    `json:"pressure_links"`
	PlayersWithLossPulses  int    `json:"players_with_loss_pulses"`
	Targets                int    `json:"targets"`
	WindowMS               int    `json:"pressure_window_ms"`
	Sort                   string `json:"sort"`
}

type CombatPlayerSummary struct {
	PlayerID             int                       `json:"player_id"`
	PlayerName           string                    `json:"player_name,omitempty"`
	LossPulses           int                       `json:"loss_pulses"`
	NetObjectsLost       int                       `json:"net_objects_lost"`
	SingleObjectRemovals int                       `json:"single_object_removals"`
	LargestNetLoss       int                       `json:"largest_net_loss"`
	FirstLossTimeMS      int                       `json:"first_loss_time_ms,omitempty"`
	FirstLossTime        string                    `json:"first_loss_time,omitempty"`
	FirstDeathTimeMS     int                       `json:"first_death_time_ms,omitempty"`
	FirstDeathTime       string                    `json:"first_death_time,omitempty"`
	PressureLinksAgainst int                       `json:"pressure_links_against"`
	DeathReplacements    int                       `json:"death_replacements"`
	PressureByAttacker   []CombatPressureAggressor `json:"pressure_by_attacker,omitempty"`
}

type CombatPressureAggressor struct {
	PlayerID   int    `json:"player_id"`
	PlayerName string `json:"player_name,omitempty"`
	Events     int    `json:"events"`
}

type CombatTargetSummary struct {
	TargetObjectID              int                       `json:"target_object_id"`
	TargetOwnerID               int                       `json:"target_owner_id"`
	TargetOwnerName             string                    `json:"target_owner_name,omitempty"`
	TargetUnitID                int                       `json:"target_unit_id,omitempty"`
	TargetUnitName              string                    `json:"target_unit_name,omitempty"`
	TargetClass                 string                    `json:"target_class,omitempty"`
	TargetX                     float64                   `json:"target_x,omitempty"`
	TargetY                     float64                   `json:"target_y,omitempty"`
	PressureEvents              int                       `json:"pressure_events"`
	FirstPressureTimeMS         int                       `json:"first_pressure_time_ms,omitempty"`
	FirstPressureTime           string                    `json:"first_pressure_time,omitempty"`
	LastPressureTimeMS          int                       `json:"last_pressure_time_ms,omitempty"`
	LastPressureTime            string                    `json:"last_pressure_time,omitempty"`
	NearbyLossPulses            int                       `json:"nearby_loss_pulses"`
	NetObjectsLostNearby        int                       `json:"net_objects_lost_nearby"`
	PressureByAttacker          []CombatPressureAggressor `json:"pressure_by_attacker,omitempty"`
	NearbyLossPulseVictimOnly   bool                      `json:"nearby_loss_pulse_victim_only"`
	NearbyLossNotObjectSpecific bool                      `json:"nearby_loss_not_object_specific"`
	Confidence                  string                    `json:"confidence"`
}

type CombatLossPulse struct {
	PlayerID             int                   `json:"player_id"`
	PlayerName           string                `json:"player_name,omitempty"`
	FromTimeMS           int                   `json:"from_time_ms"`
	FromTime             string                `json:"from_time"`
	ToTimeMS             int                   `json:"to_time_ms"`
	ToTime               string                `json:"to_time"`
	NetObjectsLost       int                   `json:"net_objects_lost"`
	ObjectCountDelta     int64                 `json:"object_count_delta"`
	UnitTypeSumDelta     int64                 `json:"unit_type_sum_delta"`
	ObjectIDSumDelta     int64                 `json:"object_id_sum_delta"`
	PositionSumDelta     int64                 `json:"position_sum_delta,omitempty"`
	Word3Delta           int64                 `json:"word_3_delta,omitempty"`
	SingleObjectRemoval  bool                  `json:"single_object_removal,omitempty"`
	RemovedUnitID        int                   `json:"removed_unit_id,omitempty"`
	RemovedUnitName      string                `json:"removed_unit_name,omitempty"`
	RemovedObjectID      int64                 `json:"removed_object_id,omitempty"`
	CandidatePressure    []CombatPressureEvent `json:"candidate_pressure,omitempty"`
	CandidatePressureNum int                   `json:"candidate_pressure_events"`
	Confidence           string                `json:"confidence"`
}

type CombatDeathEvent struct {
	PlayerID         int    `json:"player_id"`
	PlayerName       string `json:"player_name,omitempty"`
	FromTimeMS       int    `json:"from_time_ms"`
	FromTime         string `json:"from_time"`
	ToTimeMS         int    `json:"to_time_ms"`
	ToTime           string `json:"to_time"`
	LiveUnitID       int    `json:"live_unit_id,omitempty"`
	LiveUnitName     string `json:"live_unit_name,omitempty"`
	CorpseUnitID     int    `json:"corpse_unit_id,omitempty"`
	CorpseUnitName   string `json:"corpse_unit_name,omitempty"`
	ObjectCountDelta int64  `json:"object_count_delta"`
	UnitTypeSumDelta int64  `json:"unit_type_sum_delta"`
	ObjectIDSumDelta int64  `json:"object_id_sum_delta"`
	Word3Delta       int64  `json:"word_3_delta,omitempty"`
	Word4Delta       int64  `json:"word_4_delta,omitempty"`
	Confidence       string `json:"confidence"`
}

type CombatPressureEvent struct {
	TimeMS          int     `json:"time_ms"`
	Time            string  `json:"time"`
	PlayerID        int     `json:"player_id"`
	PlayerName      string  `json:"player_name,omitempty"`
	Type            string  `json:"type"`
	ActionID        int     `json:"action_id,omitempty"`
	ActionName      string  `json:"action_name,omitempty"`
	TargetObjectID  int     `json:"target_object_id,omitempty"`
	TargetOwnerID   int     `json:"target_owner_id,omitempty"`
	TargetUnitID    int     `json:"target_unit_id,omitempty"`
	TargetUnitName  string  `json:"target_unit_name,omitempty"`
	TargetClass     string  `json:"target_class,omitempty"`
	TargetX         float64 `json:"target_x,omitempty"`
	TargetY         float64 `json:"target_y,omitempty"`
	SelectedObjects int     `json:"selected_objects,omitempty"`
	Confidence      string  `json:"confidence"`
}

func BuildCombatStory(path string, opts CombatOptions) (*CombatReport, error) {
	opts = normalizeCombatOptions(opts)
	sync, err := BuildSyncStream(path, SyncOptions{ChecksumsOnly: true})
	if err != nil {
		return nil, err
	}
	events, err := ExtractEvents(path, EventOptions{IncludeUntypedAction: true, IncludeObjectIndex: true})
	if err != nil {
		return nil, err
	}
	nameByPlayer := combatPlayerNames(events.Players)
	pressure := combatPressureEvents(events.Events, nameByPlayer)
	report := &CombatReport{
		Path:         path,
		Method:       "sync_checksum_death_replacement_and_net_loss_joined_to_target_command_pressure",
		Verification: combatVerification,
		Summary: CombatSummary{
			ChecksumSamples: sync.Summary.ChecksumDE,
			StateDeltas:     len(sync.StateDeltas),
			WindowMS:        opts.WindowMS,
			Sort:            opts.Sort,
		},
		Warnings: append(append([]string{}, sync.Warnings...), events.Warnings...),
	}
	playerRows := map[int]*CombatPlayerSummary{}
	for _, delta := range sync.StateDeltas {
		if delta.Kind == "live_unit_replaced_by_corpse" {
			event := CombatDeathEvent{
				PlayerID:         delta.PlayerID,
				PlayerName:       nameByPlayer[delta.PlayerID],
				FromTimeMS:       delta.FromTimeMS,
				FromTime:         delta.FromTime,
				ToTimeMS:         delta.ToTimeMS,
				ToTime:           delta.ToTime,
				LiveUnitID:       delta.UnitID,
				LiveUnitName:     delta.UnitName,
				CorpseUnitID:     delta.ReplacementUnitID,
				CorpseUnitName:   delta.ReplacementUnitName,
				ObjectCountDelta: delta.ObjectCountDelta,
				UnitTypeSumDelta: delta.UnitTypeSumDelta,
				ObjectIDSumDelta: delta.ObjectIDSumDelta,
				Word3Delta:       delta.Word3Delta,
				Word4Delta:       delta.Word4Delta,
				Confidence:       "checksum_live_unit_to_corpse_observed_not_kill_attribution",
			}
			report.DeathEvents = append(report.DeathEvents, event)
			report.Summary.DeathReplacements++
			row := combatPlayerRow(playerRows, delta.PlayerID, nameByPlayer[delta.PlayerID])
			row.DeathReplacements++
			if row.FirstDeathTimeMS == 0 || event.ToTimeMS < row.FirstDeathTimeMS {
				row.FirstDeathTimeMS = event.ToTimeMS
				row.FirstDeathTime = event.ToTime
			}
			continue
		}
		if delta.ObjectCountDelta >= 0 {
			continue
		}
		if checksumDeltaIsKnownNonLossArtifact(delta) {
			report.Summary.FilteredArtifacts++
			continue
		}
		lost := int(-delta.ObjectCountDelta)
		if lost < opts.MinNetLoss {
			continue
		}
		pulse := CombatLossPulse{
			PlayerID:         delta.PlayerID,
			PlayerName:       nameByPlayer[delta.PlayerID],
			FromTimeMS:       delta.FromTimeMS,
			FromTime:         delta.FromTime,
			ToTimeMS:         delta.ToTimeMS,
			ToTime:           delta.ToTime,
			NetObjectsLost:   lost,
			ObjectCountDelta: delta.ObjectCountDelta,
			UnitTypeSumDelta: delta.UnitTypeSumDelta,
			ObjectIDSumDelta: delta.ObjectIDSumDelta,
			PositionSumDelta: delta.PositionSumDelta,
			Word3Delta:       delta.Word3Delta,
			Confidence:       "checksum_net_object_count_loss_observed_not_kill_attribution",
		}
		if delta.Kind == "object_removed" {
			pulse.SingleObjectRemoval = true
			pulse.RemovedUnitID = delta.UnitID
			pulse.RemovedUnitName = delta.UnitName
			pulse.RemovedObjectID = delta.ObjectID
			report.Summary.SingleObjectRemovals++
		}
		pulse.CandidatePressure = combatPressureForPulse(pulse, pressure, opts.WindowMS)
		pulse.CandidatePressureNum = len(pulse.CandidatePressure)
		if pulse.CandidatePressureNum > 0 {
			pulse.Confidence = combatCandidateConfidence
			report.Summary.PressureLinks += pulse.CandidatePressureNum
		}
		report.LossPulses = append(report.LossPulses, pulse)
		row := combatPlayerRow(playerRows, delta.PlayerID, nameByPlayer[delta.PlayerID])
		row.LossPulses++
		row.NetObjectsLost += lost
		if pulse.SingleObjectRemoval {
			row.SingleObjectRemovals++
		}
		if lost > row.LargestNetLoss {
			row.LargestNetLoss = lost
		}
		if row.FirstLossTimeMS == 0 || pulse.ToTimeMS < row.FirstLossTimeMS {
			row.FirstLossTimeMS = pulse.ToTimeMS
			row.FirstLossTime = pulse.ToTime
		}
		row.PressureLinksAgainst += pulse.CandidatePressureNum
		for _, event := range pulse.CandidatePressure {
			combatAddPressureAggressor(row, event.PlayerID, event.PlayerName)
		}
		report.Summary.LossPulses++
		report.Summary.NetObjectsLost += lost
	}
	sortCombatPulses(report.LossPulses, opts.Sort)
	report.Players = combatPlayerRows(playerRows)
	report.Summary.PlayersWithLossPulses = len(report.Players)
	report.Targets = combatTargetRows(pressure, report.LossPulses, nameByPlayer, opts.WindowMS)
	report.Summary.Targets = len(report.Targets)
	if opts.Limit > 0 && len(report.DeathEvents) > opts.Limit {
		report.DeathEvents = report.DeathEvents[:opts.Limit]
	}
	report.Summary.ShownDeathReplacements = len(report.DeathEvents)
	if opts.Limit > 0 && len(report.LossPulses) > opts.Limit {
		report.LossPulses = report.LossPulses[:opts.Limit]
	}
	report.Summary.ShownLossPulses = len(report.LossPulses)
	if len(report.Warnings) == 0 {
		report.Warnings = nil
	}
	return report, nil
}

func normalizeCombatOptions(opts CombatOptions) CombatOptions {
	if opts.WindowMS < 0 {
		opts.WindowMS = 0
	}
	if opts.Limit < 0 {
		opts.Limit = 0
	}
	if opts.MinNetLoss <= 0 {
		opts.MinNetLoss = defaultCombatMinNetLoss
	}
	if opts.Sort == "" {
		opts.Sort = "loss"
	}
	if opts.Sort != "loss" && opts.Sort != "time" {
		opts.Sort = "loss"
	}
	return opts
}

func sortCombatPulses(pulses []CombatLossPulse, mode string) {
	sort.Slice(pulses, func(i, j int) bool {
		if mode == "time" {
			if pulses[i].ToTimeMS != pulses[j].ToTimeMS {
				return pulses[i].ToTimeMS < pulses[j].ToTimeMS
			}
			if pulses[i].PlayerID != pulses[j].PlayerID {
				return pulses[i].PlayerID < pulses[j].PlayerID
			}
			return pulses[i].NetObjectsLost > pulses[j].NetObjectsLost
		}
		if pulses[i].NetObjectsLost != pulses[j].NetObjectsLost {
			return pulses[i].NetObjectsLost > pulses[j].NetObjectsLost
		}
		if pulses[i].ToTimeMS != pulses[j].ToTimeMS {
			return pulses[i].ToTimeMS < pulses[j].ToTimeMS
		}
		return pulses[i].PlayerID < pulses[j].PlayerID
	})
}

func combatPlayerNames(players []FeedbackPlayer) map[int]string {
	out := map[int]string{}
	for _, player := range players {
		if player.PlayerID > 0 && player.Name != "" {
			out[player.PlayerID] = player.Name
		}
	}
	return out
}

func combatPressureEvents(events []ReplayEvent, names map[int]string) []CombatPressureEvent {
	var out []CombatPressureEvent
	for _, event := range events {
		if event.PlayerID <= 0 || event.TargetObject == nil || event.TargetObject.OwnerID <= 0 || event.TargetObject.OwnerID == event.PlayerID {
			continue
		}
		switch event.Type {
		case "order", "special", "attack_ground", "attack_move":
		default:
			continue
		}
		pressure := CombatPressureEvent{
			TimeMS:          event.TimeMS,
			Time:            event.Time,
			PlayerID:        event.PlayerID,
			PlayerName:      event.PlayerName,
			Type:            event.Type,
			ActionID:        event.ActionID,
			ActionName:      event.ActionName,
			TargetObjectID:  event.TargetObject.ObjectID,
			TargetOwnerID:   event.TargetObject.OwnerID,
			TargetUnitID:    event.TargetObject.UnitID,
			TargetUnitName:  UnitDisplayName(event.TargetObject.UnitID),
			TargetClass:     event.TargetObject.Class,
			TargetX:         event.TargetObject.X,
			TargetY:         event.TargetObject.Y,
			SelectedObjects: len(event.ObjectIDs),
			Confidence:      "target_command_to_replay_indexed_enemy_object",
		}
		if pressure.PlayerName == "" {
			pressure.PlayerName = names[event.PlayerID]
		}
		out = append(out, pressure)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].TimeMS != out[j].TimeMS {
			return out[i].TimeMS < out[j].TimeMS
		}
		return out[i].PlayerID < out[j].PlayerID
	})
	return out
}

func combatPressureForPulse(pulse CombatLossPulse, pressure []CombatPressureEvent, windowMS int) []CombatPressureEvent {
	start := pulse.FromTimeMS - windowMS
	end := pulse.ToTimeMS + windowMS
	var out []CombatPressureEvent
	for _, event := range pressure {
		if event.TimeMS < start {
			continue
		}
		if event.TimeMS > end {
			break
		}
		if event.TargetOwnerID != pulse.PlayerID {
			continue
		}
		out = append(out, event)
		if len(out) >= defaultPressurePerPulse {
			break
		}
	}
	return out
}

func combatPlayerRow(rows map[int]*CombatPlayerSummary, id int, name string) *CombatPlayerSummary {
	row := rows[id]
	if row == nil {
		row = &CombatPlayerSummary{PlayerID: id, PlayerName: name}
		rows[id] = row
	}
	return row
}

func combatAddPressureAggressor(row *CombatPlayerSummary, id int, name string) {
	if id <= 0 {
		return
	}
	for i := range row.PressureByAttacker {
		if row.PressureByAttacker[i].PlayerID == id {
			row.PressureByAttacker[i].Events++
			if row.PressureByAttacker[i].PlayerName == "" {
				row.PressureByAttacker[i].PlayerName = name
			}
			return
		}
	}
	row.PressureByAttacker = append(row.PressureByAttacker, CombatPressureAggressor{PlayerID: id, PlayerName: name, Events: 1})
}

func combatPlayerRows(rows map[int]*CombatPlayerSummary) []CombatPlayerSummary {
	out := make([]CombatPlayerSummary, 0, len(rows))
	for _, row := range rows {
		sort.Slice(row.PressureByAttacker, func(i, j int) bool {
			if row.PressureByAttacker[i].Events != row.PressureByAttacker[j].Events {
				return row.PressureByAttacker[i].Events > row.PressureByAttacker[j].Events
			}
			return row.PressureByAttacker[i].PlayerID < row.PressureByAttacker[j].PlayerID
		})
		out = append(out, *row)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].NetObjectsLost != out[j].NetObjectsLost {
			return out[i].NetObjectsLost > out[j].NetObjectsLost
		}
		return out[i].PlayerID < out[j].PlayerID
	})
	return out
}

func combatTargetRows(pressure []CombatPressureEvent, pulses []CombatLossPulse, names map[int]string, windowMS int) []CombatTargetSummary {
	type pulseKey struct {
		fromMS   int
		toMS     int
		playerID int
	}
	type targetRow struct {
		CombatTargetSummary
		seenPulses map[pulseKey]bool
	}
	rows := map[int]*targetRow{}
	for _, event := range pressure {
		if event.TargetObjectID <= 0 || event.TargetOwnerID <= 0 {
			continue
		}
		row := rows[event.TargetObjectID]
		if row == nil {
			row = &targetRow{
				CombatTargetSummary: CombatTargetSummary{
					TargetObjectID:              event.TargetObjectID,
					TargetOwnerID:               event.TargetOwnerID,
					TargetOwnerName:             names[event.TargetOwnerID],
					TargetUnitID:                event.TargetUnitID,
					TargetUnitName:              event.TargetUnitName,
					TargetClass:                 event.TargetClass,
					TargetX:                     event.TargetX,
					TargetY:                     event.TargetY,
					NearbyLossPulseVictimOnly:   true,
					NearbyLossNotObjectSpecific: true,
					Confidence:                  "target_command_group_with_nearby_victim_net_loss_not_object_specific",
				},
				seenPulses: map[pulseKey]bool{},
			}
			rows[event.TargetObjectID] = row
		}
		row.PressureEvents++
		if row.FirstPressureTimeMS == 0 || event.TimeMS < row.FirstPressureTimeMS {
			row.FirstPressureTimeMS = event.TimeMS
			row.FirstPressureTime = event.Time
		}
		if event.TimeMS > row.LastPressureTimeMS {
			row.LastPressureTimeMS = event.TimeMS
			row.LastPressureTime = event.Time
		}
		combatAddTargetAggressor(&row.CombatTargetSummary, event.PlayerID, event.PlayerName)
		for _, pulse := range pulses {
			if pulse.PlayerID != event.TargetOwnerID {
				continue
			}
			if event.TimeMS < pulse.FromTimeMS-windowMS || event.TimeMS > pulse.ToTimeMS+windowMS {
				continue
			}
			key := pulseKey{fromMS: pulse.FromTimeMS, toMS: pulse.ToTimeMS, playerID: pulse.PlayerID}
			if row.seenPulses[key] {
				continue
			}
			row.seenPulses[key] = true
			row.NearbyLossPulses++
			row.NetObjectsLostNearby += pulse.NetObjectsLost
		}
	}
	out := make([]CombatTargetSummary, 0, len(rows))
	for _, row := range rows {
		sort.Slice(row.PressureByAttacker, func(i, j int) bool {
			if row.PressureByAttacker[i].Events != row.PressureByAttacker[j].Events {
				return row.PressureByAttacker[i].Events > row.PressureByAttacker[j].Events
			}
			return row.PressureByAttacker[i].PlayerID < row.PressureByAttacker[j].PlayerID
		})
		out = append(out, row.CombatTargetSummary)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].NetObjectsLostNearby != out[j].NetObjectsLostNearby {
			return out[i].NetObjectsLostNearby > out[j].NetObjectsLostNearby
		}
		if out[i].PressureEvents != out[j].PressureEvents {
			return out[i].PressureEvents > out[j].PressureEvents
		}
		if out[i].FirstPressureTimeMS != out[j].FirstPressureTimeMS {
			return out[i].FirstPressureTimeMS < out[j].FirstPressureTimeMS
		}
		return out[i].TargetObjectID < out[j].TargetObjectID
	})
	return out
}

func combatAddTargetAggressor(row *CombatTargetSummary, id int, name string) {
	if id <= 0 {
		return
	}
	for i := range row.PressureByAttacker {
		if row.PressureByAttacker[i].PlayerID == id {
			row.PressureByAttacker[i].Events++
			if row.PressureByAttacker[i].PlayerName == "" {
				row.PressureByAttacker[i].PlayerName = name
			}
			return
		}
	}
	row.PressureByAttacker = append(row.PressureByAttacker, CombatPressureAggressor{PlayerID: id, PlayerName: name, Events: 1})
}
