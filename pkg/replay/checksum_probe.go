package replay

import "fmt"

type ChecksumProbeOptions struct {
	Preset           string
	PlayerID         int
	ObjectCountDelta *int64
	UnitTypeSumDelta *int64
	ObjectIDSumDelta *int64
	Word3Delta       *int64
	Word4Delta       *int64
	ScoreDelta       *int64
	Limit            int
}

type ChecksumProbeReport struct {
	Path         string                `json:"path,omitempty"`
	Method       string                `json:"method"`
	Verification string                `json:"verification"`
	Summary      ChecksumProbeSummary  `json:"summary"`
	Probes       []ChecksumProbeResult `json:"probes,omitempty"`
	Warnings     []string              `json:"warnings,omitempty"`
}

type ChecksumProbeSummary struct {
	ChecksumSamples int    `json:"checksum_samples"`
	StateDeltas     int    `json:"state_deltas"`
	ProbeCount      int    `json:"probe_count"`
	MatchedProbes   int    `json:"matched_probes"`
	DurationMS      int    `json:"duration_ms"`
	Duration        string `json:"duration"`
	Preset          string `json:"preset,omitempty"`
}

type ChecksumProbeSpec struct {
	Name             string `json:"name"`
	Description      string `json:"description,omitempty"`
	PlayerID         int    `json:"player_id,omitempty"`
	ObjectCountDelta *int64 `json:"object_count_delta,omitempty"`
	UnitTypeSumDelta *int64 `json:"unit_type_sum_delta,omitempty"`
	ObjectIDSumDelta *int64 `json:"object_id_sum_delta,omitempty"`
	Word3Delta       *int64 `json:"word_3_delta,omitempty"`
	Word4Delta       *int64 `json:"word_4_delta,omitempty"`
	ScoreDelta       *int64 `json:"word_9_score_candidate_delta,omitempty"`
}

type ChecksumProbeResult struct {
	ChecksumProbeSpec
	Matched    bool                     `json:"matched"`
	MatchCount int                      `json:"match_count"`
	Result     string                   `json:"result"`
	Candidates []ChecksumProbeCandidate `json:"candidates,omitempty"`
}

type ChecksumProbeCandidate struct {
	PlayerID         int    `json:"player_id"`
	PlayerLabel      string `json:"player_label"`
	FromTimeMS       int    `json:"from_time_ms"`
	FromTime         string `json:"from_time"`
	ToTimeMS         int    `json:"to_time_ms"`
	ToTime           string `json:"to_time"`
	ObjectCountDelta int64  `json:"object_count_delta,omitempty"`
	UnitTypeSumDelta int64  `json:"unit_type_sum_delta,omitempty"`
	ObjectIDSumDelta int64  `json:"object_id_sum_delta,omitempty"`
	PositionSumDelta int64  `json:"position_sum_delta,omitempty"`
	Word3Delta       int64  `json:"word_3_delta,omitempty"`
	Word4Delta       int64  `json:"word_4_delta,omitempty"`
	ScoreDelta       int64  `json:"word_9_score_candidate_delta,omitempty"`
	Kind             string `json:"kind"`
	UnitID           int    `json:"unit_id,omitempty"`
	UnitName         string `json:"unit_name,omitempty"`
	ObjectID         int64  `json:"object_id,omitempty"`
	Confidence       string `json:"confidence"`
}

func BuildChecksumProbe(path string, opts ChecksumProbeOptions) (*ChecksumProbeReport, error) {
	if opts.PlayerID < 0 || opts.PlayerID > 8 {
		return nil, fmt.Errorf("player must be 1..8")
	}
	specs, err := checksumProbeSpecs(opts)
	if err != nil {
		return nil, err
	}
	sync, err := BuildSyncStream(path, SyncOptions{ChecksumsOnly: true})
	if err != nil {
		return nil, err
	}
	report := &ChecksumProbeReport{
		Path:         path,
		Method:       "body_op2_checksum_state_delta_probe",
		Verification: "structure_verified_checksum_delta_match_no_death_or_kill_attribution",
		Warnings:     append([]string{}, sync.Warnings...),
	}
	report.Summary.ChecksumSamples = sync.Summary.ChecksumDE
	report.Summary.StateDeltas = len(sync.StateDeltas)
	report.Summary.DurationMS = sync.Summary.DurationMS
	report.Summary.Duration = sync.Summary.Duration
	report.Summary.Preset = opts.Preset
	for _, spec := range specs {
		result := ChecksumProbeResult{
			ChecksumProbeSpec: spec,
			Result:            "no_match",
		}
		for _, delta := range sync.StateDeltas {
			if !checksumProbeMatches(spec, delta) {
				continue
			}
			result.MatchCount++
			if opts.Limit == 0 || len(result.Candidates) < opts.Limit {
				result.Candidates = append(result.Candidates, checksumProbeCandidate(delta))
			}
		}
		if result.MatchCount > 0 {
			result.Matched = true
			result.Result = "matched_observed_checksum_delta"
			report.Summary.MatchedProbes++
		}
		report.Probes = append(report.Probes, result)
	}
	report.Summary.ProbeCount = len(report.Probes)
	if opts.Preset == "v6-playground-helper" {
		report.Warnings = append(report.Warnings,
			"v6 preset probes checksum object-state deltas only; a match is not engine kill attribution",
			"Gaia has no dedicated P0 row in the 8x11 checksum matrix, so Gaia probes search any P1..P8 row for leakage",
			"if actor-side P2 kill credit remains absent, prefer trigger reconstruction over more checksum grinding",
		)
	}
	if opts.Preset == "v7-checksum-calibration" {
		report.Warnings = append(report.Warnings,
			"v7 preset matches known object-count and unit-type-sum anchors; use candidate word3/word4/score deltas to name fuzzy words",
			"missing building create/kill anchors may indicate the engine rejected that trigger effect, not a replay parser failure",
		)
	}
	return report, nil
}

func checksumProbeSpecs(opts ChecksumProbeOptions) ([]ChecksumProbeSpec, error) {
	switch opts.Preset {
	case "":
		if opts.ObjectCountDelta == nil && opts.UnitTypeSumDelta == nil && opts.ObjectIDSumDelta == nil && opts.Word3Delta == nil && opts.Word4Delta == nil && opts.ScoreDelta == nil {
			return nil, fmt.Errorf("custom checksum probe requires at least one delta flag")
		}
		return []ChecksumProbeSpec{{
			Name:             "custom",
			PlayerID:         opts.PlayerID,
			ObjectCountDelta: opts.ObjectCountDelta,
			UnitTypeSumDelta: opts.UnitTypeSumDelta,
			ObjectIDSumDelta: opts.ObjectIDSumDelta,
			Word3Delta:       opts.Word3Delta,
			Word4Delta:       opts.Word4Delta,
			ScoreDelta:       opts.ScoreDelta,
		}}, nil
	case "v6-playground-helper":
		neg3 := int64(-3)
		neg6 := int64(-6)
		neg526 := int64(-526)
		neg501 := int64(-501)
		neg1027 := int64(-1027)
		plus3 := int64(3)
		return []ChecksumProbeSpec{
			{
				Name:             "p3_hostile_dummies_removed",
				Description:      "P3 inert hostile dummy band refs 960201..960203; unit_const 74+4+448=526",
				PlayerID:         3,
				ObjectCountDelta: &neg3,
				UnitTypeSumDelta: &neg526,
			},
			{
				Name:             "gaia_neutral_dummies_removed_any_row",
				Description:      "Gaia neutral dummy band refs 960401..960403; unit_const 83+125+293=501; Gaia has no P0 checksum row",
				ObjectCountDelta: &neg3,
				UnitTypeSumDelta: &neg501,
			},
			{
				Name:             "hostile_plus_gaia_combined_same_row",
				Description:      "all six dummy removals in one row if Gaia leaks into a player row and coalesces with P3",
				ObjectCountDelta: &neg6,
				UnitTypeSumDelta: &neg1027,
			},
			{
				Name:             "p2_actor_object_credit_candidate",
				Description:      "weak actor-side candidate: P2 object count +3, not a kill counter by itself",
				PlayerID:         2,
				ObjectCountDelta: &plus3,
			},
		}, nil
	case "v7-checksum-calibration":
		return checksumProbeV7CalibrationSpecs(), nil
	default:
		return nil, fmt.Errorf("unknown checksum probe preset %q", opts.Preset)
	}
}

func checksumProbeV7CalibrationSpecs() []ChecksumProbeSpec {
	units := []struct {
		name string
		id   int64
	}{
		{"villager_male", 83},
		{"archer", 4},
		{"militia", 74},
		{"scout_cavalry", 448},
		{"paladin", 569},
		{"monk", 125},
		{"castle", 82},
	}
	specs := make([]ChecksumProbeSpec, 0, len(units)*2)
	for _, unit := range units {
		plusOne := int64(1)
		plusID := unit.id
		specs = append(specs, ChecksumProbeSpec{
			Name:             "create_" + unit.name,
			Description:      fmt.Sprintf("v7 timed P1 create-object calibration for unit_const %d", unit.id),
			PlayerID:         1,
			ObjectCountDelta: &plusOne,
			UnitTypeSumDelta: &plusID,
		})
	}
	for _, unit := range units {
		negOne := int64(-1)
		negID := -unit.id
		specs = append(specs, ChecksumProbeSpec{
			Name:             "kill_" + unit.name,
			Description:      fmt.Sprintf("v7 timed P1 kill-object calibration for preplaced unit_const %d", unit.id),
			PlayerID:         1,
			ObjectCountDelta: &negOne,
			UnitTypeSumDelta: &negID,
		})
	}
	replacements := []struct {
		name       string
		typeDelta  int64
		word3Delta int64
		word4Delta int64
	}{
		{"villager_male", 141, 1, 292},
		{"archer", -1, 1, 284},
		{"militia", 78, 1, 276},
		{"scout_cavalry", 1, 1, 269},
		{"paladin", 1, 1, 261},
		{"monk", 9, 1, 156},
		{"castle", 1348, -1, 7},
	}
	for _, replacement := range replacements {
		zero := int64(0)
		objectIDDelta := int64(-970192)
		typeDelta := replacement.typeDelta
		word3Delta := replacement.word3Delta
		word4Delta := replacement.word4Delta
		specs = append(specs, ChecksumProbeSpec{
			Name:             "kill_object_replacement_" + replacement.name,
			Description:      "v7 observed selected-object kill effect: object count stays flat while the high ref is replaced by a low runtime object",
			PlayerID:         1,
			ObjectCountDelta: &zero,
			UnitTypeSumDelta: &typeDelta,
			ObjectIDSumDelta: &objectIDDelta,
			Word3Delta:       &word3Delta,
			Word4Delta:       &word4Delta,
		})
	}
	return specs
}

func checksumProbeMatches(spec ChecksumProbeSpec, delta SyncStateDelta) bool {
	if spec.PlayerID > 0 && spec.PlayerID != delta.PlayerID {
		return false
	}
	if spec.ObjectCountDelta != nil && *spec.ObjectCountDelta != delta.ObjectCountDelta {
		return false
	}
	if spec.UnitTypeSumDelta != nil && *spec.UnitTypeSumDelta != delta.UnitTypeSumDelta {
		return false
	}
	if spec.ObjectIDSumDelta != nil && *spec.ObjectIDSumDelta != delta.ObjectIDSumDelta {
		return false
	}
	if spec.Word3Delta != nil && *spec.Word3Delta != delta.Word3Delta {
		return false
	}
	if spec.Word4Delta != nil && *spec.Word4Delta != delta.Word4Delta {
		return false
	}
	if spec.ScoreDelta != nil && *spec.ScoreDelta != delta.ScoreDelta {
		return false
	}
	return true
}

func checksumProbeCandidate(delta SyncStateDelta) ChecksumProbeCandidate {
	return ChecksumProbeCandidate{
		PlayerID:         delta.PlayerID,
		PlayerLabel:      delta.PlayerLabel,
		FromTimeMS:       delta.FromTimeMS,
		FromTime:         delta.FromTime,
		ToTimeMS:         delta.ToTimeMS,
		ToTime:           delta.ToTime,
		ObjectCountDelta: delta.ObjectCountDelta,
		UnitTypeSumDelta: delta.UnitTypeSumDelta,
		ObjectIDSumDelta: delta.ObjectIDSumDelta,
		PositionSumDelta: delta.PositionSumDelta,
		Word3Delta:       delta.Word3Delta,
		Word4Delta:       delta.Word4Delta,
		ScoreDelta:       delta.ScoreDelta,
		Kind:             delta.Kind,
		UnitID:           delta.UnitID,
		UnitName:         delta.UnitName,
		ObjectID:         delta.ObjectID,
		Confidence:       delta.Confidence,
	}
}
