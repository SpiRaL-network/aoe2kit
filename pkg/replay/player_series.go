package replay

import (
	"fmt"
	"sort"
)

type PlayerSeriesOptions struct {
	PlayerID int
	// Limit caps emitted samples per player (0 = all). Aggregates always cover
	// every checksum sample and every delta.
	Limit int
	// ChangesOnly emits only samples where one of the decoded checksum words
	// changed from that player's previous sample. Aggregates are unaffected.
	ChangesOnly bool
}

type PlayerSeriesReport struct {
	Path          string               `json:"path,omitempty"`
	Method        string               `json:"method"`
	Verification  string               `json:"verification"`
	Summary       PlayerSeriesSummary  `json:"summary"`
	WordSemantics map[string]string    `json:"checksum_word_semantics,omitempty"`
	Players       []PlayerSeriesPlayer `json:"players,omitempty"`
	Deltas        []SyncStateDelta     `json:"deltas,omitempty"`
	Warnings      []string             `json:"warnings,omitempty"`
}

type PlayerSeriesSummary struct {
	ChecksumSamples int    `json:"checksum_samples"`
	Players         int    `json:"players"`
	EmittedSamples  int    `json:"emitted_samples"`
	EmittedDeltas   int    `json:"emitted_deltas"`
	DurationMS      int    `json:"duration_ms"`
	Duration        string `json:"duration"`
}

type PlayerSeriesPlayer struct {
	PlayerID                int                 `json:"player_id"`
	PlayerLabel             string              `json:"player_label"`
	PlayerName              string              `json:"player_name,omitempty"`
	ProfileID               int                 `json:"profile_id,omitempty"`
	Samples                 int                 `json:"samples"`
	EmittedSamples          int                 `json:"emitted_samples"`
	FirstTimeMS             int                 `json:"first_time_ms,omitempty"`
	FirstTime               string              `json:"first_time,omitempty"`
	LastTimeMS              int                 `json:"last_time_ms,omitempty"`
	LastTime                string              `json:"last_time,omitempty"`
	FinalObjectCount        int                 `json:"final_object_count"`
	FinalUnitTypeSum        uint32              `json:"final_unit_type_sum"`
	FinalObjectIDSum        uint32              `json:"final_object_id_sum"`
	FinalResourceStockpile  uint32              `json:"final_resource_stockpile"`
	FinalPositionSum        uint32              `json:"final_position_sum"`
	FinalScoreCandidate     uint32              `json:"final_score_candidate"`
	ObjectAdds              int                 `json:"object_adds"`
	ObjectRemoves           int                 `json:"object_removes"`
	FilteredArtifactAdds    int                 `json:"filtered_artifact_adds,omitempty"`
	FilteredArtifactRemoves int                 `json:"filtered_artifact_removes,omitempty"`
	DeathReplacements       int                 `json:"death_replacements,omitempty"`
	SingleObjectAdds        int                 `json:"single_object_adds"`
	SingleObjectRemoves     int                 `json:"single_object_removes"`
	MultiObjectAddEvents    int                 `json:"multi_object_add_events"`
	MultiObjectRemoveEvents int                 `json:"multi_object_remove_events"`
	AddedUnitTypes          []UnitDeltaCount    `json:"added_unit_types,omitempty"`
	RemovedUnitTypes        []UnitDeltaCount    `json:"removed_unit_types,omitempty"`
	DeathReplacementUnits   []UnitDeltaCount    `json:"death_replacement_units,omitempty"`
	ProducedEstimate        int                 `json:"produced_estimate"`
	LostEstimate            int                 `json:"lost_estimate"`
	ProducedEstimateSource  string              `json:"produced_estimate_source"`
	LostEstimateSource      string              `json:"lost_estimate_source"`
	AttributionConfidence   string              `json:"attribution_confidence"`
	SamplesOut              []PlayerStateSample `json:"samples_out,omitempty"`
}

type PlayerStateSample struct {
	TimeMS            int      `json:"time_ms"`
	Time              string   `json:"time"`
	ResourceStockpile uint32   `json:"resource_stockpile"`
	UnitTypeSum       uint32   `json:"unit_type_sum"`
	ForceMetric       uint32   `json:"word_3_force_metric"`
	Word4             uint32   `json:"word_4"`
	ObjectCount       uint32   `json:"object_count"`
	PositionSum       uint32   `json:"position_sum"`
	PlayerNumber      uint32   `json:"player_number_word_8"`
	ScoreCandidate    uint32   `json:"word_9_score_candidate"`
	ObjectIDSum       uint32   `json:"object_id_sum"`
	RawWords          []uint32 `json:"raw_words_11_u32,omitempty"`
}

type UnitDeltaCount struct {
	UnitID   int    `json:"unit_id"`
	UnitName string `json:"unit_name,omitempty"`
	Count    int    `json:"count"`
}

func BuildPlayerSeries(path string, opts PlayerSeriesOptions) (*PlayerSeriesReport, error) {
	if opts.PlayerID < 0 || opts.PlayerID > 8 {
		return nil, fmt.Errorf("player must be 1..8")
	}
	sync, err := BuildSyncStream(path, SyncOptions{ChecksumsOnly: true})
	if err != nil {
		return nil, err
	}
	rec, err := Open(path)
	if err != nil {
		return nil, err
	}
	names := map[int]PlayerSlot{}
	for _, player := range rec.Players {
		if player.Number > 0 {
			names[player.Number] = player
		}
	}

	report := &PlayerSeriesReport{
		Path:          path,
		Method:        "body_op2_checksum_matrix_player_series",
		Verification:  "structure_verified_checksum_words_with_tiered_semantics_no_kill_attribution",
		WordSemantics: sync.WordSemantics,
		Warnings:      append([]string{}, sync.Warnings...),
	}
	report.Summary.ChecksumSamples = sync.Summary.ChecksumDE
	report.Summary.DurationMS = sync.Summary.DurationMS
	report.Summary.Duration = sync.Summary.Duration

	players := make(map[int]*PlayerSeriesPlayer)
	lastSample := make(map[int]PlayerStateSample)
	for _, ev := range sync.Events {
		if ev.Form != "checksum_de" || len(ev.Matrix) != 8 {
			continue
		}
		for row := 0; row < 8; row++ {
			playerID := row + 1
			if opts.PlayerID > 0 && playerID != opts.PlayerID {
				continue
			}
			words := ev.Matrix[row]
			if len(words) < 11 || allZeroU32(words) {
				continue
			}
			player := players[playerID]
			if player == nil {
				player = &PlayerSeriesPlayer{
					PlayerID:               playerID,
					PlayerLabel:            fmt.Sprintf("P%d", playerID),
					ProducedEstimateSource: "checksum word_6 object-count increases; own-side object additions only",
					LostEstimateSource:     "checksum word_6 object-count decreases after known artifacts are filtered; death replacements are separate; not kill attribution",
					AttributionConfidence:  "object_add_remove_proxy_no_death_or_kill_claim",
				}
				if slot, ok := names[playerID]; ok {
					player.PlayerName = slot.Name
					player.ProfileID = slot.ProfileID
				}
				players[playerID] = player
			}
			sample := sampleFromMatrixRow(ev, words)
			player.Samples++
			if player.FirstTimeMS == 0 {
				player.FirstTimeMS = sample.TimeMS
				player.FirstTime = sample.Time
			}
			player.LastTimeMS = sample.TimeMS
			player.LastTime = sample.Time
			player.FinalObjectCount = int(sample.ObjectCount)
			player.FinalUnitTypeSum = sample.UnitTypeSum
			player.FinalObjectIDSum = sample.ObjectIDSum
			player.FinalResourceStockpile = sample.ResourceStockpile
			player.FinalPositionSum = sample.PositionSum
			player.FinalScoreCandidate = sample.ScoreCandidate
			emit := opts.Limit == 0 || player.EmittedSamples < opts.Limit
			if opts.ChangesOnly {
				prev, ok := lastSample[playerID]
				emit = emit && (!ok || sampleChanged(prev, sample))
			}
			if emit {
				player.SamplesOut = append(player.SamplesOut, sample)
				player.EmittedSamples++
				report.Summary.EmittedSamples++
			}
			lastSample[playerID] = sample
		}
	}

	addedTypes := map[int]map[int]int{}
	removedTypes := map[int]map[int]int{}
	deathTypes := map[int]map[int]int{}
	for _, delta := range sync.StateDeltas {
		if opts.PlayerID > 0 && delta.PlayerID != opts.PlayerID {
			continue
		}
		player := players[delta.PlayerID]
		if player == nil {
			continue
		}
		report.Deltas = append(report.Deltas, delta)
		if delta.Kind == "live_unit_replaced_by_corpse" {
			player.DeathReplacements++
			if delta.UnitID > 0 {
				if deathTypes[delta.PlayerID] == nil {
					deathTypes[delta.PlayerID] = map[int]int{}
				}
				deathTypes[delta.PlayerID][delta.UnitID]++
			}
			continue
		}
		if delta.ObjectCountDelta > 0 {
			if checksumDeltaIsKnownNonLossArtifact(delta) {
				player.FilteredArtifactAdds += int(delta.ObjectCountDelta)
				continue
			}
			player.ObjectAdds += int(delta.ObjectCountDelta)
			if delta.ObjectCountDelta == 1 && delta.UnitID > 0 {
				player.SingleObjectAdds++
				if addedTypes[delta.PlayerID] == nil {
					addedTypes[delta.PlayerID] = map[int]int{}
				}
				addedTypes[delta.PlayerID][delta.UnitID]++
			} else {
				player.MultiObjectAddEvents++
			}
		} else if delta.ObjectCountDelta < 0 {
			if checksumDeltaIsKnownNonLossArtifact(delta) {
				player.FilteredArtifactRemoves += int(-delta.ObjectCountDelta)
				continue
			}
			player.ObjectRemoves += int(-delta.ObjectCountDelta)
			if delta.ObjectCountDelta == -1 && delta.UnitID > 0 {
				player.SingleObjectRemoves++
				if removedTypes[delta.PlayerID] == nil {
					removedTypes[delta.PlayerID] = map[int]int{}
				}
				removedTypes[delta.PlayerID][delta.UnitID]++
			} else {
				player.MultiObjectRemoveEvents++
			}
		}
	}

	keys := make([]int, 0, len(players))
	for playerID := range players {
		keys = append(keys, playerID)
	}
	sort.Ints(keys)
	for _, playerID := range keys {
		player := players[playerID]
		player.ProducedEstimate = player.ObjectAdds
		player.LostEstimate = player.ObjectRemoves
		player.AddedUnitTypes = unitDeltaCounts(addedTypes[playerID])
		player.RemovedUnitTypes = unitDeltaCounts(removedTypes[playerID])
		player.DeathReplacementUnits = unitDeltaCounts(deathTypes[playerID])
		report.Players = append(report.Players, *player)
	}
	report.Summary.Players = len(report.Players)
	report.Summary.EmittedDeltas = len(report.Deltas)
	return report, nil
}

func sampleFromMatrixRow(ev SyncEvent, words []uint32) PlayerStateSample {
	raw := append([]uint32(nil), words[:11]...)
	return PlayerStateSample{
		TimeMS:            ev.TimeMS,
		Time:              ev.Time,
		ResourceStockpile: words[1],
		UnitTypeSum:       words[2],
		ForceMetric:       words[3],
		Word4:             words[4],
		ObjectCount:       words[6],
		PositionSum:       words[7],
		PlayerNumber:      words[8],
		ScoreCandidate:    words[9],
		ObjectIDSum:       words[10],
		RawWords:          raw,
	}
}

func sampleChanged(prev, curr PlayerStateSample) bool {
	return prev.ResourceStockpile != curr.ResourceStockpile ||
		prev.UnitTypeSum != curr.UnitTypeSum ||
		prev.ForceMetric != curr.ForceMetric ||
		prev.Word4 != curr.Word4 ||
		prev.ObjectCount != curr.ObjectCount ||
		prev.PositionSum != curr.PositionSum ||
		prev.ScoreCandidate != curr.ScoreCandidate ||
		prev.ObjectIDSum != curr.ObjectIDSum
}

func unitDeltaCounts(counts map[int]int) []UnitDeltaCount {
	if len(counts) == 0 {
		return nil
	}
	unitIDs := make([]int, 0, len(counts))
	for unitID := range counts {
		unitIDs = append(unitIDs, unitID)
	}
	sort.Ints(unitIDs)
	out := make([]UnitDeltaCount, 0, len(unitIDs))
	for _, unitID := range unitIDs {
		out = append(out, UnitDeltaCount{
			UnitID:   unitID,
			UnitName: UnitDisplayName(unitID),
			Count:    counts[unitID],
		})
	}
	return out
}
