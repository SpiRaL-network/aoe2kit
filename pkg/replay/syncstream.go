package replay

import (
	"encoding/json"
	"fmt"
	"path/filepath"
)

// SyncStream surfaces the op=2 sync heartbeat: simulation-time deltas plus the
// periodic checksum payloads. Honesty boundary: time framing (deltas, forms,
// spans) is decoded structure. The checksum payload's named words are tiered by
// fixture evidence; unnamed words stay raw.

type SyncReport struct {
	Path          string            `json:"path,omitempty"`
	Method        string            `json:"method"`
	Verification  string            `json:"verification"`
	Summary       SyncSummary       `json:"summary"`
	DeltaShapes   []DeltaShape      `json:"delta_shapes,omitempty"`
	WordSemantics map[string]string `json:"checksum_word_semantics,omitempty"`
	Events        []SyncEvent       `json:"events,omitempty"`
	RawWords      []SyncRawWordRow  `json:"raw_words,omitempty"`
	StateDeltas   []SyncStateDelta  `json:"state_deltas,omitempty"`
	Warnings      []string          `json:"warnings,omitempty"`
}

// syncWordSemantics documents per-word evidence for the DE checksum 8x11 matrix.
// Tiers: decoded (tested invariant) > strong_hypothesis (multi-fixture anchor)
// > behavior_observed (pattern seen, semantics not claimed).
func syncWordSemantics() map[string]string {
	return map[string]string{
		"row":     "decoded: rows are per-player slots P1..P8; inactive players are all-zero (1-player fixture control)",
		"word_8":  "decoded: player number (equals row's P number in every fixture; golden-tested)",
		"word_0":  "constant_zero_observed in all fixtures",
		"word_5":  "constant_zero_observed in all fixtures",
		"word_1":  "decoded: current total stockpiled standard resources, food+wood+stone+gold. v12 wrote each resource separately and every sampled carrier matched the four-resource sum exactly.",
		"word_3":  "strong_hypothesis_near_decode: sum of the engine per-object state field for the player's current objects; exact fixture deltas (+2 live object, +3 corpse/flare state) and desync p0-sync rough comparison support this, exact-overlap log still desired",
		"word_2":  "decoded: sum of current living object unit-type ids for that player (diagnostic v2 deltas exactly match +83 Villager Male and +448 Scout Cavalry creates; collapse behavior matches prior wipe observations)",
		"word_4":  "decoded: sum of each checksum-counted object's carry field. Meaning is object-type dependent: villagers carry resources, monks carry faith, corpses/rubble carry decay timer, and flares/markers carry lifetime",
		"word_6":  "decoded: current living object count for that player (diagnostic v2 increments exactly +1 per create and then stops; blank fixture is static)",
		"word_7":  "decoded: sum of (x+y)*100 over current checksum-counted objects. Kill Factory trigger-create at tile (6,10) adds +1700 via center (6.5,10.5); desync p0-sync object dump uses the same XY100 candidate.",
		"word_9":  "provisional synced multiplayer player-state digest/check word: cross-POV multiplayer recordings match exactly; visible-score, object-HP, and exploration-only mappings are disproven; SP input-sensitive observations must not be generalized to MP; a frozen MP fixture shows a five-value successor ring with forward skips",
		"word_10": "decoded: sum of current living object instance ids for that player (diagnostic v2 increments exactly by new object ids +4,+5,+6,+7,+8,+9)",
		"trailer": "decoded: world time in ms (equals accumulated sync time; golden-tested)",
	}
}

type SyncSummary struct {
	SyncCount      int     `json:"sync_count"`
	DeltaOnly      int     `json:"delta_only_syncs"`
	ChecksumDE     int     `json:"checksum_de_syncs"`
	ChecksumLegacy int     `json:"checksum_legacy_syncs"`
	EmittedEvents  int     `json:"emitted_events"`
	FirstTimeMS    int     `json:"first_time_ms"`
	FirstTime      string  `json:"first_time"`
	LastTimeMS     int     `json:"last_time_ms"`
	LastTime       string  `json:"last_time"`
	DurationMS     int     `json:"duration_ms"`
	Duration       string  `json:"duration"`
	MinDeltaMS     int     `json:"min_delta_ms"`
	MaxDeltaMS     int     `json:"max_delta_ms"`
	MeanDeltaMS    float64 `json:"mean_delta_ms"`
}

type DeltaShape struct {
	DeltaMS int `json:"delta_ms"`
	Count   int `json:"count"`
}

type SyncEvent struct {
	TimeMS       int        `json:"time_ms"`
	Time         string     `json:"time"`
	SourceOffset int        `json:"source_offset"`
	DeltaMS      int        `json:"delta_ms"`
	Form         string     `json:"form"`
	ProbeHex     string     `json:"probe_hex,omitempty"`
	Matrix       [][]uint32 `json:"matrix_8x11_u32,omitempty"`
	TrailerU32   *uint32    `json:"trailer_u32,omitempty"`
	PayloadHex   string     `json:"payload_hex,omitempty"`
}

type SyncRawWordRow struct {
	SampleIndex  int      `json:"sample_index"`
	TimeMS       int      `json:"time_ms"`
	Time         string   `json:"time"`
	SourceOffset int      `json:"source_offset"`
	PlayerID     int      `json:"player_id"`
	Words        []uint32 `json:"words_u32"`
}

type SyncStateDelta struct {
	PlayerID            int    `json:"player_id"`
	PlayerLabel         string `json:"player_label"`
	FromTimeMS          int    `json:"from_time_ms"`
	FromTime            string `json:"from_time"`
	ToTimeMS            int    `json:"to_time_ms"`
	ToTime              string `json:"to_time"`
	ObjectCountDelta    int64  `json:"object_count_delta,omitempty"`
	UnitTypeSumDelta    int64  `json:"unit_type_sum_delta,omitempty"`
	ObjectIDSumDelta    int64  `json:"object_id_sum_delta,omitempty"`
	PositionSumDelta    int64  `json:"position_sum_delta,omitempty"`
	Word3Delta          int64  `json:"word_3_delta,omitempty"`
	Word4Delta          int64  `json:"word_4_delta,omitempty"`
	ScoreDelta          int64  `json:"word_9_score_candidate_delta,omitempty"`
	Kind                string `json:"kind"`
	UnitID              int    `json:"unit_id,omitempty"`
	UnitName            string `json:"unit_name,omitempty"`
	ReplacementUnitID   int    `json:"replacement_unit_id,omitempty"`
	ReplacementUnitName string `json:"replacement_unit_name,omitempty"`
	ObjectID            int64  `json:"object_id,omitempty"`
	Confidence          string `json:"confidence"`
}

type SyncOptions struct {
	// Limit caps emitted event rows (0 = all). Summary always covers every sync.
	Limit int
	// ChecksumsOnly emits only checksum-carrying syncs as rows.
	ChecksumsOnly bool
	// RawWords emits one row per DE checksum sample and player slot with all 11 words.
	RawWords bool
}

func BuildSyncStream(path string, opts SyncOptions) (*SyncReport, error) {
	cache, err := OpenReplayCache(path)
	if err != nil {
		return nil, err
	}
	report, dir, err := cache.syncFiles()
	if err != nil {
		return nil, err
	}
	report.Path = path
	budget := 0
	add := func(n int) error {
		budget += n
		if budget > maxReportBytes {
			return fmt.Errorf("sync result exceeds 32 MiB report budget; use kit replay sync JSON streaming")
		}
		return nil
	}
	err = walkJSONRows(filepath.Join(dir, "sync.jsonl"), func(raw []byte) error {
		var ev SyncEvent
		if err := json.Unmarshal(raw, &ev); err != nil {
			return err
		}
		if opts.ChecksumsOnly && ev.Form == "delta_only" {
			return nil
		}
		if opts.Limit > 0 && len(report.Events) >= opts.Limit {
			return nil
		}
		if err := add(len(raw) + 256); err != nil {
			return err
		}
		report.Events = append(report.Events, ev)
		return nil
	})
	if err != nil {
		return nil, err
	}
	err = walkJSONRows(filepath.Join(dir, "lifecycle.jsonl"), func(raw []byte) error {
		if err := add(len(raw) + 512); err != nil {
			return err
		}
		var d SyncStateDelta
		if err := json.Unmarshal(raw, &d); err != nil {
			return err
		}
		report.StateDeltas = append(report.StateDeltas, d)
		return nil
	})
	if err != nil {
		return nil, err
	}
	if opts.RawWords {
		err = walkJSONRows(filepath.Join(dir, "raw_words.jsonl"), func(raw []byte) error {
			if err := add(len(raw) + 256); err != nil {
				return err
			}
			var d SyncRawWordRow
			if err := json.Unmarshal(raw, &d); err != nil {
				return err
			}
			report.RawWords = append(report.RawWords, d)
			return nil
		})
	}
	report.Summary.EmittedEvents = len(report.Events)
	return report, err
}

func syncRawWordRows(ev SyncEvent, sampleIndex int) []SyncRawWordRow {
	if len(ev.Matrix) != 8 {
		return nil
	}
	rows := make([]SyncRawWordRow, 0, 8)
	for p := 0; p < 8; p++ {
		words := append([]uint32(nil), ev.Matrix[p]...)
		rows = append(rows, SyncRawWordRow{
			SampleIndex:  sampleIndex,
			TimeMS:       ev.TimeMS,
			Time:         ev.Time,
			SourceOffset: ev.SourceOffset,
			PlayerID:     p + 1,
			Words:        words,
		})
	}
	return rows
}

func syncStateDeltas(prev, curr SyncEvent) []SyncStateDelta {
	if len(prev.Matrix) != 8 || len(curr.Matrix) != 8 {
		return nil
	}
	var out []SyncStateDelta
	for p := 0; p < 8; p++ {
		if len(prev.Matrix[p]) < 11 || len(curr.Matrix[p]) < 11 {
			continue
		}
		if allZeroU32(prev.Matrix[p]) && allZeroU32(curr.Matrix[p]) {
			continue
		}
		objectCountDelta := checksumWordDelta(prev.Matrix[p][6], curr.Matrix[p][6])
		unitTypeSumDelta := checksumWordDelta(prev.Matrix[p][2], curr.Matrix[p][2])
		objectIDSumDelta := checksumWordDelta(prev.Matrix[p][10], curr.Matrix[p][10])
		positionSumDelta := checksumWordDelta(prev.Matrix[p][7], curr.Matrix[p][7])
		word3Delta := checksumWordDelta(prev.Matrix[p][3], curr.Matrix[p][3])
		word4Delta := checksumWordDelta(prev.Matrix[p][4], curr.Matrix[p][4])
		scoreDelta := checksumWordDelta(prev.Matrix[p][9], curr.Matrix[p][9])
		if objectCountDelta == 0 && unitTypeSumDelta == 0 && objectIDSumDelta == 0 && positionSumDelta == 0 && word3Delta == 0 && word4Delta == 0 && scoreDelta == 0 {
			continue
		}
		delta := SyncStateDelta{
			PlayerID:         p + 1,
			PlayerLabel:      fmt.Sprintf("P%d", p+1),
			FromTimeMS:       prev.TimeMS,
			FromTime:         prev.Time,
			ToTimeMS:         curr.TimeMS,
			ToTime:           curr.Time,
			ObjectCountDelta: objectCountDelta,
			UnitTypeSumDelta: unitTypeSumDelta,
			ObjectIDSumDelta: objectIDSumDelta,
			PositionSumDelta: positionSumDelta,
			Word3Delta:       word3Delta,
			Word4Delta:       word4Delta,
			ScoreDelta:       scoreDelta,
			Kind:             "sync_state_sum_delta",
			Confidence:       "checksum_delta_observed",
		}
		if objectCountDelta == 1 && unitTypeSumDelta > 0 {
			delta.Kind = "object_added"
			delta.UnitID = int(unitTypeSumDelta)
			delta.UnitName = UnitDisplayName(delta.UnitID)
			if objectIDSumDelta > 0 {
				delta.ObjectID = objectIDSumDelta
			}
			delta.Confidence = "checksum_single_object_delta_fixture_validated"
			classifyKnownChecksumObjectDelta(&delta)
		} else if objectCountDelta == -1 && unitTypeSumDelta < 0 {
			delta.Kind = "object_removed"
			delta.UnitID = int(-unitTypeSumDelta)
			delta.UnitName = UnitDisplayName(delta.UnitID)
			if objectIDSumDelta < 0 {
				delta.ObjectID = -objectIDSumDelta
			}
			delta.Confidence = "checksum_single_object_delta_fixture_validated"
			classifyKnownChecksumObjectDelta(&delta)
		} else if objectCountDelta != 0 {
			delta.Kind = "object_count_changed"
			classifyKnownChecksumObjectDelta(&delta)
		} else if objectIDSumDelta != 0 && unitTypeSumDelta != 0 {
			delta.Kind = "object_replaced_or_transformed"
			classifyKnownChecksumObjectDelta(&delta)
		} else if unitTypeSumDelta != 0 || objectIDSumDelta != 0 {
			delta.Kind = "object_sum_changed"
			classifyKnownChecksumObjectDelta(&delta)
		} else if positionSumDelta != 0 {
			delta.Kind = "position_sum_changed"
		} else if word3Delta != 0 || word4Delta != 0 || scoreDelta != 0 {
			delta.Kind = "unnamed_checksum_word_changed"
			classifyKnownChecksumObjectDelta(&delta)
		}
		out = append(out, delta)
	}
	return out
}

func classifyKnownChecksumObjectDelta(delta *SyncStateDelta) {
	deathReplacement, isDeathReplacement := knownDeathReplacementFromDelta(delta.UnitTypeSumDelta)
	switch {
	case delta.ObjectCountDelta == 3 && delta.UnitTypeSumDelta == 336:
		delta.Kind = "fog_reveal_markers_added"
		delta.UnitID = 112
		delta.UnitName = UnitDisplayName(112)
		delta.Confidence = "engine_verified_kill_factory_fog_marker_objects"
	case delta.ObjectCountDelta == -3 && delta.UnitTypeSumDelta == -336:
		delta.Kind = "fog_reveal_markers_removed"
		delta.UnitID = 112
		delta.UnitName = UnitDisplayName(112)
		delta.Confidence = "engine_verified_kill_factory_fog_marker_objects"
	case delta.ObjectCountDelta == 1 && delta.UnitTypeSumDelta == 274 && delta.Word3Delta == 3:
		delta.Kind = "player_flare_ping_added"
		delta.UnitID = 274
		delta.UnitName = UnitDisplayName(274)
		delta.Confidence = "engine_verified_kill_factory_flare_ping_object"
	case delta.ObjectCountDelta == -1 && delta.UnitTypeSumDelta == -274:
		delta.Kind = "player_flare_ping_removed"
		delta.UnitID = 274
		delta.UnitName = UnitDisplayName(274)
		delta.Confidence = "engine_verified_kill_factory_flare_ping_object"
	case delta.ObjectCountDelta == 0 && isDeathReplacement && delta.Word4Delta > 0:
		delta.Kind = "live_unit_replaced_by_corpse"
		delta.UnitID = deathReplacement.LiveUnitID
		delta.UnitName = deathReplacement.LiveUnitName
		if delta.UnitName == "" {
			delta.UnitName = UnitDisplayName(deathReplacement.LiveUnitID)
		}
		delta.ReplacementUnitID = deathReplacement.CorpseUnitID
		delta.ReplacementUnitName = deathReplacement.CorpseUnitName
		if delta.ReplacementUnitName == "" {
			delta.ReplacementUnitName = UnitDisplayName(deathReplacement.CorpseUnitID)
		}
		delta.Confidence = "corpse_table_" + deathReplacement.Confidence + "_replacement_not_kill_attribution"
	case delta.ObjectCountDelta == -1 && delta.UnitTypeSumDelta == -28:
		delta.Kind = "corpse_expired"
		delta.UnitID = 28
		delta.UnitName = UnitDisplayName(28)
		delta.Confidence = "engine_verified_kill_factory_corpse_disappearance"
	case delta.ObjectCountDelta == 0 && delta.UnitTypeSumDelta == 0 && delta.ObjectIDSumDelta == 0 && delta.Word4Delta < 0:
		delta.Kind = "carry_sum_decreased"
		delta.Confidence = "decoded_word4_carry_sum_decreased_not_cause_specific"
	}
}

type checksumDeathReplacement struct {
	LiveUnitID     int
	LiveUnitName   string
	CorpseUnitID   int
	CorpseUnitName string
	Confidence     string
}

func knownDeathReplacementFromDelta(unitTypeSumDelta int64) (checksumDeathReplacement, bool) {
	table, err := LoadDefaultCorpseTable()
	if err != nil {
		return checksumDeathReplacement{}, false
	}
	row, ok := table.UniqueReplacementForDelta(unitTypeSumDelta)
	if !ok {
		return checksumDeathReplacement{}, false
	}
	return checksumDeathReplacement{
		LiveUnitID:     row.UnitID,
		LiveUnitName:   row.UnitName,
		CorpseUnitID:   row.CorpseID,
		CorpseUnitName: row.CorpseName,
		Confidence:     row.Confidence,
	}, true
}

func checksumDeltaIsKnownNonLossArtifact(delta SyncStateDelta) bool {
	switch delta.Kind {
	case "fog_reveal_markers_added",
		"fog_reveal_markers_removed",
		"player_flare_ping_added",
		"player_flare_ping_removed",
		"corpse_expired":
		return true
	default:
		return false
	}
}

func allZeroU32(values []uint32) bool {
	for _, value := range values {
		if value != 0 {
			return false
		}
	}
	return true
}
