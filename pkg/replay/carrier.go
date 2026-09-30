package replay

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
)

type CarrierOptions struct {
	LedgerPath string
}

type CarrierReport struct {
	Path         string           `json:"path,omitempty"`
	LedgerPath   string           `json:"ledger_path,omitempty"`
	Method       string           `json:"method"`
	Verification string           `json:"verification"`
	Summary      CarrierSummary   `json:"summary"`
	Carrier      CarrierSpec      `json:"carrier"`
	Attempts     []CarrierAttempt `json:"attempts,omitempty"`
	Samples      []CarrierSample  `json:"samples,omitempty"`
	Warnings     []string         `json:"warnings,omitempty"`
}

type CarrierSummary struct {
	ChecksumSamples     int    `json:"checksum_samples"`
	PlayerID            int    `json:"player_id"`
	Attribute           int    `json:"attribute"`
	AttributeName       string `json:"attribute_name,omitempty"`
	WordIndex           int    `json:"word_index"`
	AttemptCount        int    `json:"attempt_count"`
	Passed              int    `json:"passed"`
	Failed              int    `json:"failed"`
	Unknown             int    `json:"unknown"`
	Unsampled           int    `json:"unsampled"`
	FirstCarrierTimeMS  int    `json:"first_carrier_time_ms,omitempty"`
	FirstCarrierTime    string `json:"first_carrier_time,omitempty"`
	LastCarrierTimeMS   int    `json:"last_carrier_time_ms,omitempty"`
	LastCarrierTime     string `json:"last_carrier_time,omitempty"`
	MinChecksumGapMS    int    `json:"min_checksum_gap_ms,omitempty"`
	MedianChecksumGapMS int    `json:"median_checksum_gap_ms,omitempty"`
	MaxChecksumGapMS    int    `json:"max_checksum_gap_ms,omitempty"`
}

type CarrierSpec struct {
	PlayerID      int    `json:"player_id"`
	Attribute     int    `json:"attribute"`
	AttributeName string `json:"attribute_name,omitempty"`
	WordIndex     int    `json:"word_index"`
}

type CarrierAttempt struct {
	Index          int             `json:"index"`
	TimerS         int             `json:"timer_s"`
	WindowStartMS  int             `json:"window_start_ms"`
	WindowEndMS    int             `json:"window_end_ms,omitempty"`
	Label          string          `json:"label,omitempty"`
	XSOpenFile     string          `json:"xs_open_file"`
	ExpectedValues map[string]int  `json:"expected_values"`
	Observed       []CarrierSample `json:"observed,omitempty"`
	Verdict        string          `json:"verdict"`
	MatchedKey     string          `json:"matched_key,omitempty"`
	MatchedValue   int             `json:"matched_value,omitempty"`
	Confidence     string          `json:"confidence"`
	Note           string          `json:"note,omitempty"`
}

type CarrierSample struct {
	TimeMS int    `json:"time_ms"`
	Time   string `json:"time"`
	Value  int    `json:"value"`
}

type carrierLedger struct {
	Reader struct {
		SyncCarrier struct {
			Player        int    `json:"player"`
			Attribute     int    `json:"attribute"`
			AttributeName string `json:"attribute_name"`
		} `json:"sync_carrier"`
		Attempts []map[string]any `json:"attempts"`
	} `json:"reader"`
}

func BuildCarrierReport(path string, opts CarrierOptions) (*CarrierReport, error) {
	if strings.TrimSpace(opts.LedgerPath) == "" {
		return nil, fmt.Errorf("--ledger is required; pass the diagnostic scenario manifest/ledger JSON, or use kit replay sidecar-sync --schema for schema-based XS sidecar readback")
	}
	ledger, err := readCarrierLedger(opts.LedgerPath)
	if err != nil {
		return nil, err
	}
	spec := CarrierSpec{
		PlayerID:      ledger.Reader.SyncCarrier.Player,
		Attribute:     ledger.Reader.SyncCarrier.Attribute,
		AttributeName: ledger.Reader.SyncCarrier.AttributeName,
		WordIndex:     carrierWordIndex(ledger.Reader.SyncCarrier.Attribute),
	}
	if spec.PlayerID <= 0 || spec.PlayerID > 8 {
		return nil, fmt.Errorf("ledger sync_carrier.player must be 1..8")
	}
	if spec.WordIndex < 0 {
		return nil, fmt.Errorf("carrier attribute %d is not mapped to a checksum word", spec.Attribute)
	}
	attempts := carrierAttemptsFromLedger(ledger)
	sync, err := BuildSyncStream(path, SyncOptions{ChecksumsOnly: true})
	if err != nil {
		return nil, err
	}
	report := &CarrierReport{
		Path:         path,
		LedgerPath:   opts.LedgerPath,
		Method:       "ledger_expected_values_mapped_to_sync_checksum_carrier_word",
		Verification: "structure_verified_probe_readback_not_engine_generalization",
		Carrier:      spec,
		Attempts:     attempts,
		Warnings:     append([]string{}, sync.Warnings...),
	}
	report.Summary.ChecksumSamples = sync.Summary.ChecksumDE
	report.Summary.PlayerID = spec.PlayerID
	report.Summary.Attribute = spec.Attribute
	report.Summary.AttributeName = spec.AttributeName
	report.Summary.WordIndex = spec.WordIndex
	report.Summary.AttemptCount = len(attempts)

	samples := carrierSamples(sync.Events, spec)
	report.Samples = samples
	report.Summary.MinChecksumGapMS, report.Summary.MedianChecksumGapMS, report.Summary.MaxChecksumGapMS = checksumGaps(samples)
	if len(samples) > 0 {
		report.Summary.FirstCarrierTimeMS = samples[0].TimeMS
		report.Summary.FirstCarrierTime = samples[0].Time
		last := samples[len(samples)-1]
		report.Summary.LastCarrierTimeMS = last.TimeMS
		report.Summary.LastCarrierTime = last.Time
	}
	scoreCarrierAttempts(report, samples)
	return report, nil
}

func readCarrierLedger(path string) (carrierLedger, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return carrierLedger{}, err
	}
	var ledger carrierLedger
	if err := json.Unmarshal(data, &ledger); err != nil {
		return carrierLedger{}, err
	}
	return ledger, nil
}

func carrierWordIndex(attribute int) int {
	switch attribute {
	case 0:
		return 1
	default:
		return -1
	}
}

func carrierAttemptsFromLedger(ledger carrierLedger) []CarrierAttempt {
	out := make([]CarrierAttempt, 0, len(ledger.Reader.Attempts))
	for i, raw := range ledger.Reader.Attempts {
		attempt := CarrierAttempt{
			Index:          i + 1,
			ExpectedValues: map[string]int{},
			Confidence:     "sync_carrier_word_expected_value_match",
		}
		for key, value := range raw {
			switch key {
			case "timer":
				attempt.TimerS = intFromJSONNumber(value)
				attempt.WindowStartMS = attempt.TimerS * 1000
			case "xs_open_file":
				if text, ok := value.(string); ok {
					attempt.XSOpenFile = text
					if text == "" {
						attempt.Label = "(empty xs_open_file)"
					} else {
						attempt.Label = text
					}
				}
			default:
				if strings.HasSuffix(key, "_food") || strings.HasSuffix(key, "_value") || strings.HasSuffix(key, "_carrier") {
					if n := intFromJSONNumber(value); n != 0 {
						attempt.ExpectedValues[key] = n
					}
				}
			}
		}
		out = append(out, attempt)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].WindowStartMS < out[j].WindowStartMS
	})
	for i := range out {
		out[i].Index = i + 1
		if i+1 < len(out) {
			out[i].WindowEndMS = out[i+1].WindowStartMS
		}
	}
	return out
}

func intFromJSONNumber(value any) int {
	switch v := value.(type) {
	case float64:
		return int(v)
	case int:
		return v
	default:
		return 0
	}
}

func carrierSamples(events []SyncEvent, spec CarrierSpec) []CarrierSample {
	var out []CarrierSample
	for _, ev := range events {
		if ev.Form != "checksum_de" || len(ev.Matrix) < spec.PlayerID {
			continue
		}
		row := ev.Matrix[spec.PlayerID-1]
		if len(row) <= spec.WordIndex || allZeroU32(row) {
			continue
		}
		value := int(row[spec.WordIndex])
		if len(out) > 0 && out[len(out)-1].Value == value {
			continue
		}
		out = append(out, CarrierSample{TimeMS: ev.TimeMS, Time: ev.Time, Value: value})
	}
	return out
}

func scoreCarrierAttempts(report *CarrierReport, samples []CarrierSample) {
	for i := range report.Attempts {
		attempt := &report.Attempts[i]
		for _, sample := range samples {
			if sample.TimeMS < attempt.WindowStartMS {
				continue
			}
			if attempt.WindowEndMS > 0 && sample.TimeMS >= attempt.WindowEndMS {
				continue
			}
			attempt.Observed = append(attempt.Observed, sample)
		}
		attempt.Verdict = "unknown"
		attempt.Confidence = "carrier_window_sampled_no_expected_value_match"
		for _, sample := range attempt.Observed {
			for key, value := range attempt.ExpectedValues {
				if sample.Value != value {
					continue
				}
				attempt.MatchedKey = key
				attempt.MatchedValue = value
				attempt.Verdict = carrierVerdictFromKey(key)
				attempt.Confidence = "carrier_expected_value_observed_in_attempt_window"
				break
			}
			if attempt.MatchedKey != "" {
				break
			}
		}
		if len(attempt.Observed) == 0 {
			attempt.Verdict = "unsampled"
			attempt.Confidence = "no_checksum_sample_in_attempt_window"
			if report.Summary.MedianChecksumGapMS > 0 && attempt.WindowEndMS > attempt.WindowStartMS && attempt.WindowEndMS-attempt.WindowStartMS < report.Summary.MedianChecksumGapMS {
				attempt.Note = "attempt window is shorter than median checksum cadence"
			}
		}
		switch attempt.Verdict {
		case "success":
			report.Summary.Passed++
		case "unsampled":
			report.Summary.Unsampled++
		case "unknown":
			report.Summary.Unknown++
		default:
			report.Summary.Failed++
		}
	}
}

func carrierVerdictFromKey(key string) string {
	switch {
	case strings.Contains(key, "success"):
		return "success"
	case strings.Contains(key, "fail"):
		return strings.TrimSuffix(key, "_food")
	case strings.Contains(key, "bad"):
		return strings.TrimSuffix(key, "_food")
	default:
		return "matched_" + strings.TrimSuffix(key, "_food")
	}
}

func checksumGaps(samples []CarrierSample) (int, int, int) {
	if len(samples) < 2 {
		return 0, 0, 0
	}
	gaps := make([]int, 0, len(samples)-1)
	for i := 1; i < len(samples); i++ {
		gap := samples[i].TimeMS - samples[i-1].TimeMS
		if gap > 0 {
			gaps = append(gaps, gap)
		}
	}
	if len(gaps) == 0 {
		return 0, 0, 0
	}
	sort.Ints(gaps)
	return gaps[0], gaps[len(gaps)/2], gaps[len(gaps)-1]
}
