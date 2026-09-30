package replay

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// PostgameCorpus scans a folder of recordings for postgame variants. Current
// scenario-mode plus ranked-RM evidence is negative: op=6 carries
// metadata/leaderboard stubs, while per-player achievement stats appear to be
// display-time aggregates, not serialized replay facts. Campaign mode remains
// unsampled. Any "candidate" here is only an unparsed shape anomaly worth
// preserving, not proof of an achievement payload.

type PostgameCorpusReport struct {
	Folder       string                `json:"folder"`
	Method       string                `json:"method"`
	Verification string                `json:"verification"`
	Scanned      int                   `json:"replays_scanned"`
	Failures     int                   `json:"parse_failures"`
	Rows         []PostgameCorpusRow   `json:"rows,omitempty"`
	Summary      PostgameCorpusSummary `json:"summary"`
	Warnings     []string              `json:"warnings,omitempty"`
}

type PostgameCorpusRow struct {
	Path              string `json:"path"`
	Op6Seen           bool   `json:"op6_seen"`
	Op6TailBytes      int    `json:"op6_tail_bytes,omitempty"`
	Op6Blocks         int    `json:"op6_blocks,omitempty"`
	WorldTimeBlock    bool   `json:"world_time_block"`
	LeaderboardBlock  bool   `json:"leaderboard_block"`
	Action255Payloads int    `json:"action255_payloads"`
	UnparsedOp6Tail   bool   `json:"unparsed_op6_tail"`
	Candidate         bool   `json:"achievements_candidate"`
	Confidence        string `json:"confidence"`
	Error             string `json:"error,omitempty"`
}

type PostgameCorpusSummary struct {
	Op6Present        int `json:"op6_present"`
	Op6MetadataOnly   int `json:"op6_metadata_only"`
	Op6UnparsedTails  int `json:"op6_unparsed_tails"`
	Action255Carriers int `json:"action255_carriers"`
	Candidates        int `json:"achievements_candidates"`
}

func BuildPostgameCorpus(folder string) (*PostgameCorpusReport, error) {
	report := &PostgameCorpusReport{
		Folder:       folder,
		Method:       "folder_scan_postgame_probe_per_replay",
		Verification: "structure_observed_scenario_and_ranked_rm_postgame_stats_not_serialized",
	}
	var paths []string
	err := filepath.WalkDir(folder, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		lower := strings.ToLower(d.Name())
		if strings.HasSuffix(lower, ".aoe2record") || strings.HasSuffix(lower, ".zip") {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(paths)
	for _, path := range paths {
		row := PostgameCorpusRow{Path: path}
		probe, err := BuildPostgame(path)
		if err != nil {
			row.Error = err.Error()
			row.Confidence = "parse_failed"
			report.Failures++
			report.Rows = append(report.Rows, row)
			continue
		}
		report.Scanned++
		row.Op6Seen = probe.Summary.Op6Seen
		row.Op6TailBytes = probe.Summary.Op6TailBytes
		row.Op6Blocks = probe.Summary.Op6Blocks
		row.WorldTimeBlock = probe.Summary.HasWorldTimeBlock
		row.LeaderboardBlock = probe.Summary.HasLeaderboardBlock
		row.Action255Payloads = probe.Summary.Action255Payloads
		row.UnparsedOp6Tail = probe.DE != nil && len(probe.DE.Blocks) == 0 && probe.DE.TailBytes > 0
		row.Candidate = row.UnparsedOp6Tail || row.Action255Payloads > 0
		switch {
		case row.Action255Payloads > 0:
			row.Confidence = "action255_payload_present_undecoded"
		case row.UnparsedOp6Tail:
			row.Confidence = "op6_tail_present_unparsed_shape"
		case row.Op6Seen:
			row.Confidence = "op6_metadata_only"
		default:
			row.Confidence = "no_postgame_ops"
		}
		if row.Op6Seen {
			report.Summary.Op6Present++
			if row.UnparsedOp6Tail {
				report.Summary.Op6UnparsedTails++
			} else {
				report.Summary.Op6MetadataOnly++
			}
		}
		if row.Action255Payloads > 0 {
			report.Summary.Action255Carriers++
		}
		if row.Candidate {
			report.Summary.Candidates++
		}
		report.Rows = append(report.Rows, row)
	}
	return report, nil
}
