package replay

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"sort"
)

type PostgameReport struct {
	Path         string                 `json:"path,omitempty"`
	Method       string                 `json:"method"`
	Verification string                 `json:"verification"`
	Summary      PostgameSummary        `json:"summary"`
	DE           *DEPostgame            `json:"de_postgame,omitempty"`
	Action255    []Action255Postgame    `json:"action_255_postgame,omitempty"`
	Players      []PostgamePlayerResult `json:"players,omitempty"`
	Warnings     []string               `json:"warnings,omitempty"`
}

type PostgameSummary struct {
	Op6Seen             bool `json:"op6_seen"`
	Op6TailBytes        int  `json:"op6_tail_bytes,omitempty"`
	Op6Blocks           int  `json:"op6_blocks,omitempty"`
	Action255Seen       bool `json:"action255_seen"`
	Action255Payloads   int  `json:"action255_payloads,omitempty"`
	HasLeaderboardBlock bool `json:"has_leaderboard_block"`
	HasWorldTimeBlock   bool `json:"has_world_time_block"`
	HasPlayerKills      bool `json:"has_player_kills"`
}

type DEPostgame struct {
	SourceOffset int                     `json:"source_offset"`
	TimeMS       int                     `json:"time_ms,omitempty"`
	Time         string                  `json:"time,omitempty"`
	TailBytes    int                     `json:"tail_bytes"`
	Version      uint32                  `json:"version,omitempty"`
	BlockCount   uint32                  `json:"block_count,omitempty"`
	Blocks       []DEPostgameBlock       `json:"blocks,omitempty"`
	WorldTimeMS  uint32                  `json:"world_time_ms,omitempty"`
	Leaderboards []DEPostgameLeaderboard `json:"leaderboards,omitempty"`
	RawTailHex   string                  `json:"raw_tail_hex,omitempty"`
	Confidence   string                  `json:"confidence"`
}

type DEPostgameBlock struct {
	ID         uint32 `json:"id"`
	Name       string `json:"name,omitempty"`
	Length     uint32 `json:"length"`
	Parsed     bool   `json:"parsed"`
	Confidence string `json:"confidence"`
}

type DEPostgameLeaderboard struct {
	ID      uint32                        `json:"id"`
	Unknown uint16                        `json:"unknown"`
	Players []DEPostgameLeaderboardPlayer `json:"players,omitempty"`
}

type DEPostgameLeaderboardPlayer struct {
	PlayerNumber int32 `json:"player_number"`
	Rank         int32 `json:"rank"`
	Rating       int32 `json:"rating"`
}

type Action255Postgame struct {
	SourceOffset int    `json:"source_offset"`
	TimeMS       int    `json:"time_ms,omitempty"`
	Time         string `json:"time,omitempty"`
	PayloadBytes int    `json:"payload_bytes"`
	Sequence     int    `json:"sequence,omitempty"`
	Confidence   string `json:"confidence"`
}

type PostgamePlayerResult struct {
	PlayerID       int    `json:"player_id"`
	Label          string `json:"label"`
	Name           string `json:"name,omitempty"`
	Winner         *bool  `json:"winner,omitempty"`
	Score          *int   `json:"score,omitempty"`
	UnitsKilled    *int   `json:"units_killed,omitempty"`
	UnitsLost      *int   `json:"units_lost,omitempty"`
	BuildingsRazed *int   `json:"buildings_razed,omitempty"`
	BuildingsLost  *int   `json:"buildings_lost,omitempty"`
	FeudalTimeMS   *int   `json:"feudal_time_ms,omitempty"`
	CastleTimeMS   *int   `json:"castle_time_ms,omitempty"`
	ImperialTimeMS *int   `json:"imperial_time_ms,omitempty"`
	Confidence     string `json:"confidence"`
}

func BuildPostgame(path string) (*PostgameReport, error) {
	cache, err := OpenReplayCache(path)
	if err != nil {
		return nil, err
	}
	rec, err := cache.Metadata()
	if err != nil {
		return nil, err
	}
	report := &PostgameReport{Warnings: append([]string(nil), cache.Manifest.Warnings...)}
	err = cache.WalkOps(func(op CachedOp) error {
		if op.ID == 6 {
			report.DE = parseDEPostgameTail(op.Offset, op.TimeMS, op.Payload)
		}
		if op.ID == 1 && op.ActionID == actionPostgame {
			report.Action255 = append(report.Action255, Action255Postgame{SourceOffset: op.Offset, TimeMS: op.TimeMS, Time: FormatTime(op.TimeMS), PayloadBytes: len(op.Payload), Sequence: op.Sequence, Confidence: "payload_framed_not_decoded"})
			if len(report.Action255) > 10000 {
				return fmt.Errorf("postgame action rows exceed budget")
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	report.Path = path
	report.Method = "body_op6_reverse_block_probe_and_action255_sighting"
	report.Verification = "structure_verified_scenario_and_ranked_rm_postgame_stats_not_serialized"
	players := make([]PostgamePlayerResult, 0, len(rec.Players))
	for _, player := range rec.Players {
		if !player.Active || player.Number <= 0 {
			continue
		}
		players = append(players, PostgamePlayerResult{
			PlayerID:   player.Number,
			Label:      initialPlayerLabel(player.Number),
			Name:       player.Name,
			Confidence: "postgame_player_stats_unavailable",
		})
	}
	sort.Slice(players, func(i, j int) bool { return players[i].PlayerID < players[j].PlayerID })
	report.Players = players
	if report.DE != nil {
		report.Summary.Op6Seen = true
		report.Summary.Op6TailBytes = report.DE.TailBytes
		report.Summary.Op6Blocks = len(report.DE.Blocks)
		report.Summary.HasWorldTimeBlock = report.DE.WorldTimeMS > 0
		report.Summary.HasLeaderboardBlock = len(report.DE.Leaderboards) > 0
	}
	report.Summary.Action255Payloads = len(report.Action255)
	report.Summary.Action255Seen = len(report.Action255) > 0
	report.Summary.HasPlayerKills = false
	if report.DE != nil && len(report.DE.Blocks) > 0 && !report.Summary.HasPlayerKills {
		report.Warnings = append(report.Warnings, "DE op=6 postgame tail parsed as metadata blocks; per-player achievement/kills stats are not serialized in the observed scenario-mode and ranked-RM replay corpus; campaign mode remains unsampled")
	}
	if report.DE != nil && len(report.DE.Blocks) == 0 && report.DE.TailBytes > 0 {
		report.Warnings = append(report.Warnings, "DE op=6 tail exists but did not parse as known reverse-block postgame metadata")
	}
	if !report.Summary.Action255Seen {
		report.Warnings = append(report.Warnings, "legacy action-255 achievements payload not observed; treat it as an AoC/UserPatch-era format, not a DE outcome source")
	}
	return report, nil
}

func parsePostgameBody(body []byte) (*PostgameReport, error) {
	reader := bytes.NewReader(body)
	if err := readReplayMeta(reader); err != nil {
		return nil, fmt.Errorf("action-stream meta parse failed: %w", err)
	}
	report := &PostgameReport{}
	timeMS := 0
	for {
		opOffset := len(body) - reader.Len()
		op, err := readU32(reader)
		if err != nil {
			if err == io.EOF {
				return report, nil
			}
			return report, err
		}
		switch op {
		case 1:
			actionID, payload, sequence, err := readReplayAction(reader)
			if err != nil {
				return report, fmt.Errorf("action parse stopped: %w", err)
			}
			if actionID == actionPostgame {
				report.Action255 = append(report.Action255, Action255Postgame{
					SourceOffset: opOffset,
					TimeMS:       timeMS,
					Time:         FormatTime(timeMS),
					PayloadBytes: len(payload),
					Sequence:     sequence,
					Confidence:   "payload_framed_not_decoded",
				})
			}
		case 2:
			delta, err := readReplaySync(reader)
			if err != nil {
				return report, fmt.Errorf("sync parse stopped: %w", err)
			}
			timeMS += int(delta)
		case 3:
			if err := skipN(reader, 12); err != nil {
				return report, fmt.Errorf("viewlock parse stopped: %w", err)
			}
		case 4:
			if err := skipN(reader, 4); err != nil {
				return report, fmt.Errorf("chat prefix parse stopped: %w", err)
			}
			length, err := readU32(reader)
			if err != nil {
				return report, fmt.Errorf("chat length parse stopped: %w", err)
			}
			if length > uint32(reader.Len()) {
				return report, fmt.Errorf("chat length %d exceeds remaining body %d", length, reader.Len())
			}
			if err := skipN(reader, int(length)); err != nil {
				return report, fmt.Errorf("chat payload parse stopped: %w", err)
			}
		case 5:
			result, err := skipStart(reader)
			if err != nil {
				return report, fmt.Errorf("start op parse stopped: %w", err)
			}
			if result.Terminal {
				return report, nil
			}
		case 6:
			tail := make([]byte, reader.Len())
			if _, err := io.ReadFull(reader, tail); err != nil {
				return report, fmt.Errorf("postgame tail read failed: %w", err)
			}
			report.DE = parseDEPostgameTail(opOffset, timeMS, tail)
			return report, nil
		default:
			if err := skipReplaySaveChapter(reader, len(body)); err != nil {
				return report, fmt.Errorf("unknown operation id %d at body offset %d: %w", op, opOffset, err)
			}
		}
	}
}

func parseDEPostgameTail(offset int, timeMS int, tail []byte) *DEPostgame {
	out := &DEPostgame{
		SourceOffset: offset,
		TimeMS:       timeMS,
		Time:         FormatTime(timeMS),
		TailBytes:    len(tail),
		RawTailHex:   hex.EncodeToString(tail),
		Confidence:   "raw_tail_preserved",
	}
	if len(out.RawTailHex) > 128 {
		out.RawTailHex = out.RawTailHex[:128]
	}
	rev := reverseBytes(tail)
	if len(rev) < 16 {
		return out
	}
	cursor := 8
	out.Version = binary.BigEndian.Uint32(rev[cursor : cursor+4])
	cursor += 4
	out.BlockCount = binary.BigEndian.Uint32(rev[cursor : cursor+4])
	cursor += 4
	out.Confidence = "reverse_block_framed"
	for i := uint32(0); i < out.BlockCount && cursor+8 <= len(rev); i++ {
		id := binary.BigEndian.Uint32(rev[cursor : cursor+4])
		cursor += 4
		length := binary.BigEndian.Uint32(rev[cursor : cursor+4])
		cursor += 4
		block := DEPostgameBlock{ID: id, Name: dePostgameBlockName(id), Length: length, Confidence: "reverse_block_framed"}
		if length > uint32(len(rev)-cursor) {
			block.Confidence = "declared_length_exceeds_tail"
			out.Blocks = append(out.Blocks, block)
			out.Confidence = "partial_reverse_block_framing"
			return out
		}
		rawBlock := reverseBytes(rev[cursor : cursor+int(length)])
		cursor += int(length)
		switch id {
		case 1:
			if len(rawBlock) >= 4 {
				out.WorldTimeMS = binary.LittleEndian.Uint32(rawBlock[:4])
				block.Parsed = true
				block.Confidence = "parsed_world_time"
			}
		case 2:
			leaderboards, ok := parseDELeaderboards(rawBlock)
			if ok {
				out.Leaderboards = leaderboards
				block.Parsed = true
				block.Confidence = "parsed_leaderboards"
			}
		}
		out.Blocks = append(out.Blocks, block)
	}
	return out
}

func parseDELeaderboards(raw []byte) ([]DEPostgameLeaderboard, bool) {
	if len(raw) < 4 {
		return nil, false
	}
	cursor := 0
	count := int(binary.LittleEndian.Uint32(raw[cursor : cursor+4]))
	cursor += 4
	if count < 0 || count > 32 {
		return nil, false
	}
	leaderboards := make([]DEPostgameLeaderboard, 0, count)
	for i := 0; i < count; i++ {
		if cursor+10 > len(raw) {
			return nil, false
		}
		board := DEPostgameLeaderboard{
			ID:      binary.LittleEndian.Uint32(raw[cursor : cursor+4]),
			Unknown: binary.LittleEndian.Uint16(raw[cursor+4 : cursor+6]),
		}
		cursor += 6
		players := int(binary.LittleEndian.Uint32(raw[cursor : cursor+4]))
		cursor += 4
		if players < 0 || players > 16 || cursor+players*12 > len(raw) {
			return nil, false
		}
		for j := 0; j < players; j++ {
			board.Players = append(board.Players, DEPostgameLeaderboardPlayer{
				PlayerNumber: int32(binary.LittleEndian.Uint32(raw[cursor : cursor+4])),
				Rank:         int32(binary.LittleEndian.Uint32(raw[cursor+4 : cursor+8])),
				Rating:       int32(binary.LittleEndian.Uint32(raw[cursor+8 : cursor+12])),
			})
			cursor += 12
		}
		leaderboards = append(leaderboards, board)
	}
	return leaderboards, true
}

func dePostgameBlockName(id uint32) string {
	switch id {
	case 1:
		return "world_time"
	case 2:
		return "leaderboards"
	default:
		return "unknown"
	}
}

func reverseBytes(in []byte) []byte {
	out := make([]byte, len(in))
	for i := range in {
		out[len(in)-1-i] = in[i]
	}
	return out
}
