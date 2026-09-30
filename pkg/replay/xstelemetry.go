package replay

import "strings"

const (
	xsProbeFoodBase      = 900000
	xsProbeUnused220Base = 700000
	xsProbePackedWidth   = 100000
)

type XSTelemetryReport struct {
	Path         string              `json:"path,omitempty"`
	Method       string              `json:"method"`
	Verification string              `json:"verification"`
	Summary      XSTelemetrySummary  `json:"summary"`
	Samples      []XSTelemetrySample `json:"samples,omitempty"`
	ChatMarkers  []ReplayEvent       `json:"chat_markers,omitempty"`
	Warnings     []string            `json:"warnings,omitempty"`
}

// XSTelemetryOptions selects the authored carrier bases used by a probe.
// The defaults preserve the historical v3 probe contract; newer generators
// should pass the base they declare in their manifest.
type XSTelemetryOptions struct {
	FoodBase      int
	Unused220Base int
}

type XSTelemetrySummary struct {
	ChecksumSamples         int  `json:"checksum_samples"`
	DecodedSamples          int  `json:"decoded_samples"`
	FoodCarrierSamples      int  `json:"food_carrier_samples"`
	Unused220CarrierSamples int  `json:"unused_220_carrier_samples"`
	ChatMarkers             int  `json:"chat_markers"`
	FoodCarrierMoved        bool `json:"food_carrier_moved"`
	Unused220CarrierMoved   bool `json:"unused_220_carrier_moved"`
}

type XSTelemetrySample struct {
	TimeMS                int    `json:"time_ms"`
	Time                  string `json:"time"`
	PlayerID              int    `json:"player_id"`
	PlayerLabel           string `json:"player_label"`
	Word1                 uint32 `json:"word_1"`
	Word9                 uint32 `json:"word_9_score_candidate,omitempty"`
	Carrier               string `json:"carrier"`
	Attr20Value           int    `json:"attr_20_value"`
	Attr154Value          int    `json:"attr_154_value"`
	Attr43Value           int    `json:"attr_43_value"`
	KillEventsFromAttr154 int    `json:"kill_events_from_attr_154"`
	Confidence            string `json:"confidence"`
}

func BuildXSTelemetryProbe(path string) (*XSTelemetryReport, error) {
	return BuildXSTelemetryProbeWithOptions(path, XSTelemetryOptions{})
}

func BuildXSTelemetryProbeWithOptions(path string, opts XSTelemetryOptions) (*XSTelemetryReport, error) {
	if opts.FoodBase == 0 {
		opts.FoodBase = xsProbeFoodBase
	}
	if opts.Unused220Base == 0 {
		opts.Unused220Base = xsProbeUnused220Base
	}
	syncReport, err := BuildSyncStream(path, SyncOptions{ChecksumsOnly: true})
	if err != nil {
		return nil, err
	}
	report := &XSTelemetryReport{
		Path:         path,
		Method:       "xs_probe_sync_word_1_and_chat_marker_readback",
		Verification: "xs_probe_structure_readback_not_engine_claim",
		Warnings:     append([]string(nil), syncReport.Warnings...),
	}
	seen := map[string]bool{}
	for _, ev := range syncReport.Events {
		if ev.Form != "checksum_de" || len(ev.Matrix) == 0 {
			continue
		}
		report.Summary.ChecksumSamples++
		for p, row := range ev.Matrix {
			if len(row) < 2 {
				continue
			}
			sample, ok := decodeXSTelemetryWord1WithBases(row[1], opts.FoodBase, opts.Unused220Base)
			if !ok {
				continue
			}
			sample.TimeMS = ev.TimeMS
			sample.Time = ev.Time
			sample.PlayerID = p + 1
			sample.PlayerLabel = "P" + itoa(p+1)
			if len(row) > 9 {
				sample.Word9 = row[9]
			}
			key := sample.PlayerLabel + "|" + sample.Carrier + "|" + itoa(int(sample.Word1))
			if seen[key] {
				continue
			}
			seen[key] = true
			report.Samples = append(report.Samples, sample)
			report.Summary.DecodedSamples++
			switch sample.Carrier {
			case "food_attr_0":
				report.Summary.FoodCarrierSamples++
				report.Summary.FoodCarrierMoved = true
			case "unused_attr_220_candidate":
				report.Summary.Unused220CarrierSamples++
				report.Summary.Unused220CarrierMoved = true
			}
		}
	}
	events, err := ExtractEvents(path, EventOptions{IncludeSystemEvents: true})
	if err != nil {
		report.Warnings = append(report.Warnings, "chat marker extraction failed: "+err.Error())
		return report, nil
	}
	for _, event := range events.Events {
		if strings.Contains(event.Text, "SDSTELEM") {
			report.ChatMarkers = append(report.ChatMarkers, event)
		}
	}
	report.Summary.ChatMarkers = len(report.ChatMarkers)
	return report, nil
}

func decodeXSTelemetryWord1(value uint32) (XSTelemetrySample, bool) {
	return decodeXSTelemetryWord1WithBases(value, xsProbeFoodBase, xsProbeUnused220Base)
}

func decodeXSTelemetryWord1WithBases(value uint32, foodBase, unused220Base int) (XSTelemetrySample, bool) {
	raw := int(value)
	base := 0
	carrier := ""
	switch {
	case raw >= foodBase && raw < foodBase+xsProbePackedWidth:
		base = foodBase
		carrier = "food_attr_0"
	case raw >= unused220Base && raw < unused220Base+xsProbePackedWidth:
		base = unused220Base
		carrier = "unused_attr_220_candidate"
	default:
		return XSTelemetrySample{}, false
	}
	encoded := raw - base
	attr20 := encoded / 10000
	attr154 := (encoded / 100) % 100
	attr43 := encoded % 100
	return XSTelemetrySample{
		Word1:                 value,
		Carrier:               carrier,
		Attr20Value:           attr20,
		Attr154Value:          attr154,
		Attr43Value:           attr43,
		KillEventsFromAttr154: attr154,
		Confidence:            "xs_probe_packed_decimal_readback_attr154_empirically_kill_events",
	}, true
}

func itoa(value int) string {
	if value == 0 {
		return "0"
	}
	negative := value < 0
	if negative {
		value = -value
	}
	var buf [20]byte
	i := len(buf)
	for value > 0 {
		i--
		buf[i] = byte('0' + value%10)
		value /= 10
	}
	if negative {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
