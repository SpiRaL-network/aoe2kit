package replay

import (
	"fmt"
	"sort"
)

const SummaryVersion = "aoe2kit.replay.summary.v1"

type SummaryReport struct {
	Version       string            `json:"version"`
	Path          string            `json:"path,omitempty"`
	RecordSHA256  string            `json:"record_sha256,omitempty"`
	GameVersion   string            `json:"game_version"`
	SaveVersion   float64           `json:"save_version"`
	LogVersion    uint32            `json:"log_version,omitempty"`
	HeaderLength  int               `json:"header_length"`
	InflatedBytes int               `json:"inflated_header_bytes"`
	DurationMS    int               `json:"duration_ms,omitempty"`
	Duration      string            `json:"duration,omitempty"`
	Map           *MapInfo          `json:"map,omitempty"`
	DataSet       DataSetIdentity   `json:"data_set_identity"`
	LobbySettings *LobbySettings    `json:"lobby_settings,omitempty"`
	ScenarioName  string            `json:"scenario_name,omitempty"`
	ScenarioMeta  *ScenarioMetadata `json:"scenario_metadata,omitempty"`
	HeaderMeta    *HeaderMetadata   `json:"header_metadata,omitempty"`
	Players       []SummaryPlayer   `json:"players,omitempty"`
	Result        MatchResult       `json:"result"`
	Postgame      *PostgameSummary  `json:"postgame_summary,omitempty"`
	Warnings      []string          `json:"warnings,omitempty"`
}

type SummaryPlayer struct {
	Slot           int    `json:"slot"`
	PlayerID       int    `json:"player_id,omitempty"`
	Name           string `json:"name,omitempty"`
	AIName         string `json:"ai_name,omitempty"`
	Kind           string `json:"kind"`
	Human          bool   `json:"human"`
	Civ            int    `json:"civ,omitempty"`
	CivName        string `json:"civ_name,omitempty"`
	Color          int    `json:"color,omitempty"`
	Team           int    `json:"team,omitempty"`
	ProfileID      int    `json:"profile_id,omitempty"`
	PlayerType     int    `json:"player_type"`
	Rating         *int   `json:"rating,omitempty"`
	Winner         *bool  `json:"winner,omitempty"`
	Resigned       bool   `json:"resigned,omitempty"`
	SelectedTeamID int    `json:"selected_team_id,omitempty"`
	ResolvedTeamID int    `json:"resolved_team_id,omitempty"`
}

type SummaryOptions struct {
	IncludeMapTiles bool
}

func BuildSummary(path string) (*SummaryReport, error) {
	return BuildSummaryWithOptions(path, SummaryOptions{})
}

func BuildSummaryWithOptions(path string, opts SummaryOptions) (*SummaryReport, error) {
	cache, err := OpenReplayCache(path)
	if err != nil {
		return nil, err
	}
	rec, err := cache.Metadata()
	if err != nil {
		return nil, err
	}
	events, err := ExtractEvents(path, EventOptions{})
	if err != nil {
		return nil, err
	}
	postgame, postgameErr := BuildPostgame(path)
	ratings := map[int]int{}
	if postgameErr == nil && postgame != nil && postgame.DE != nil {
		for _, board := range postgame.DE.Leaderboards {
			for _, player := range board.Players {
				if player.PlayerNumber <= 0 {
					continue
				}
				ratings[int(player.PlayerNumber)] = int(player.Rating)
			}
		}
	}
	winnerSet := map[int]bool{}
	for _, id := range events.Result.Winners {
		winnerSet[id] = true
	}
	loserSet := map[int]bool{}
	for _, id := range events.Result.Losers {
		loserSet[id] = true
	}
	resignedSet := map[int]bool{}
	for _, id := range events.Result.Resigned {
		resignedSet[id] = true
	}
	players := make([]SummaryPlayer, 0, len(rec.Players))
	for _, player := range rec.Players {
		if !player.Active {
			continue
		}
		row := SummaryPlayer{
			Slot:           player.Slot,
			PlayerID:       player.Number,
			Name:           player.Name,
			AIName:         player.AIName,
			Kind:           player.Kind,
			Human:          player.Human,
			Civ:            player.Civ,
			CivName:        CivDisplayName(player.Civ),
			Color:          player.Color,
			Team:           player.Team,
			ProfileID:      player.ProfileID,
			PlayerType:     player.PlayerType,
			Resigned:       resignedSet[player.Number],
			SelectedTeamID: player.SelectedTeamID,
			ResolvedTeamID: player.ResolvedTeamID,
		}
		if rating, ok := ratings[player.Number]; ok {
			row.Rating = &rating
		}
		if events.Result.WinnerKnown {
			winner := winnerSet[player.Number]
			if winner || loserSet[player.Number] {
				row.Winner = &winner
			}
		}
		players = append(players, row)
	}
	sort.Slice(players, func(i, j int) bool { return players[i].Slot < players[j].Slot })
	var warnings []string
	if len(players) == 0 {
		fallbackPlayers, err := checksumFallbackSummaryPlayers(path)
		if err == nil && len(fallbackPlayers) > 0 {
			players = fallbackPlayers
			warnings = append(warnings, "header roster unavailable; players are checksum-row placeholders with unknown names/civs")
		} else if err != nil {
			warnings = append(warnings, "checksum-row player fallback unavailable: "+err.Error())
		}
	}
	report := &SummaryReport{
		Version:       SummaryVersion,
		Path:          path,
		RecordSHA256:  rec.RecordSHA256,
		GameVersion:   rec.GameVersion,
		SaveVersion:   rec.SaveVersion,
		LogVersion:    rec.LogVersion,
		HeaderLength:  rec.HeaderLength,
		InflatedBytes: rec.InflatedBytes,
		DurationMS:    events.Counts.DurationMS,
		Duration:      events.Counts.Duration,
		DataSet:       rec.DataSet,
		LobbySettings: rec.LobbySettings,
		ScenarioName:  rec.ScenarioName,
		ScenarioMeta:  rec.ScenarioMeta,
		HeaderMeta:    rec.HeaderMeta,
		Players:       players,
		Result:        events.Result,
		Warnings:      warnings,
	}
	if rec.Fallback != nil && rec.Fallback.Map != nil {
		report.Map = rec.Fallback.Map
	}
	if opts.IncludeMapTiles {
		header, err := cache.HeaderReader()
		if err != nil {
			return nil, err
		}
		prefix := make([]byte, min(cache.Manifest.HeaderBytes, headerMetadataWindow))
		_, err = header.ReadAt(prefix, 0)
		header.Close()
		if err != nil {
			return nil, err
		}
		mapInfo, err := parseMapInfoWithOptions(prefix, rec.SaveVersion, MapInfoOptions{IncludeTiles: true})
		if err != nil {
			report.Warnings = append(report.Warnings, "map tile grid unavailable: "+err.Error())
		} else {
			report.Map = mapInfo
		}
	}
	if postgameErr != nil {
		report.Warnings = append(report.Warnings, "postgame summary unavailable: "+postgameErr.Error())
	} else if postgame != nil {
		report.Postgame = &postgame.Summary
	}
	report.Warnings = append(report.Warnings, events.Warnings...)
	return report, nil
}

func checksumFallbackSummaryPlayers(path string) ([]SummaryPlayer, error) {
	series, err := BuildPlayerSeries(path, PlayerSeriesOptions{})
	if err != nil {
		return nil, err
	}
	if len(series.Players) == 0 {
		return nil, fmt.Errorf("no checksum player rows found")
	}
	players := make([]SummaryPlayer, 0, len(series.Players))
	for _, player := range series.Players {
		if player.PlayerID <= 0 {
			continue
		}
		players = append(players, SummaryPlayer{
			Slot:     player.PlayerID,
			PlayerID: player.PlayerID,
			Kind:     "checksum_row",
			Human:    false,
		})
	}
	sort.Slice(players, func(i, j int) bool { return players[i].Slot < players[j].Slot })
	return players, nil
}
