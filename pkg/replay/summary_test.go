package replay

import (
	"os"
	"regexp"
	"testing"

	"aoe2kit/pkg/testfixtures"
)

func TestSummaryMapTilesOptIn(t *testing.T) {
	path := testfixtures.Path(t, "save-analysis/latest_cba.aoe2record")
	if _, err := os.Stat(path); err != nil {
		t.Skipf("golden replay not present: %v", err)
	}
	compact, err := BuildSummary(path)
	if err != nil {
		t.Fatal(err)
	}
	if compact.Map == nil || compact.Map.TileCount == 0 {
		t.Fatalf("compact summary missing map metadata: %+v", compact.Map)
	}
	if len(compact.Map.Tiles) != 0 {
		t.Fatalf("compact summary emitted %d tiles without opt-in", len(compact.Map.Tiles))
	}
	withTiles, err := BuildSummaryWithOptions(path, SummaryOptions{IncludeMapTiles: true})
	if err != nil {
		t.Fatal(err)
	}
	if withTiles.Map == nil {
		t.Fatal("tile summary missing map metadata")
	}
	if withTiles.Map.Width != 120 || withTiles.Map.Height != 120 || withTiles.Map.TileCount != 14400 {
		t.Fatalf("map dimensions = %dx%d/%d, want 120x120/14400", withTiles.Map.Width, withTiles.Map.Height, withTiles.Map.TileCount)
	}
	if len(withTiles.Map.Tiles) != withTiles.Map.TileCount {
		t.Fatalf("tiles = %d, want %d", len(withTiles.Map.Tiles), withTiles.Map.TileCount)
	}
	if withTiles.Map.Tiles[0].X != 0 || withTiles.Map.Tiles[0].Y != 0 {
		t.Fatalf("first tile coordinates = %+v, want origin", withTiles.Map.Tiles[0])
	}
	last := withTiles.Map.Tiles[len(withTiles.Map.Tiles)-1]
	if last.X != withTiles.Map.Width-1 || last.Y != withTiles.Map.Height-1 {
		t.Fatalf("last tile coordinates = %+v, want map corner", last)
	}
}

func TestSummaryIncludesLobbyPlayersAndResult(t *testing.T) {
	path := testfixtures.Path(t, "save-analysis/latest_cba.aoe2record")
	if _, err := os.Stat(path); err != nil {
		t.Skipf("golden replay not present: %v", err)
	}
	report, err := BuildSummary(path)
	if err != nil {
		t.Fatal(err)
	}
	if ok, _ := regexp.MatchString(`^[0-9a-f]{64}$`, report.RecordSHA256); !ok {
		t.Fatalf("record_sha256 = %q, want 64 lowercase hex chars", report.RecordSHA256)
	}
	if report.HeaderMeta == nil || report.HeaderMeta.TailGUIDHex == "" {
		t.Fatalf("summary missing header tail GUID: %+v", report.HeaderMeta)
	}
	if ok, _ := regexp.MatchString(`^[0-9a-f]{32}$`, report.HeaderMeta.TailGUIDHex); !ok {
		t.Fatalf("tail_guid_hex = %q, want 32 lowercase hex chars", report.HeaderMeta.TailGUIDHex)
	}
	if report.LobbySettings == nil {
		t.Fatal("summary missing lobby settings")
	}
	if report.LobbySettings.PlayerCount != 8 || report.LobbySettings.PopulationLimit != 200 || report.LobbySettings.MapDimension != 120 {
		t.Fatalf("lobby settings = %+v, want player_count=8 pop=200 map_dimension=120", report.LobbySettings)
	}
	if report.ScenarioName != "CBA - Sorry Noobs" {
		t.Fatalf("scenario_name = %q", report.ScenarioName)
	}
	if len(report.Players) != 8 {
		t.Fatalf("players = %d, want 8", len(report.Players))
	}
	if report.Result.WinnerKnown || len(report.Result.Winners) != 0 || len(report.Result.Losers) == 0 {
		t.Fatalf("partial resign stream must retain losers without inventing winners: %+v", report.Result)
	}
	ratedPlayers := 0
	for _, player := range report.Players {
		if player.Name == "" || player.CivName == "" {
			t.Fatalf("player row missing name/civ: %+v", player)
		}
		if player.Rating != nil {
			ratedPlayers++
		}
	}
	if ratedPlayers == 0 {
		t.Fatal("summary did not join any postgame leaderboard ratings onto players")
	}
}
