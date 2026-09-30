package aoems

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPlayerStatsHeadersAndCache(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method != http.MethodPost || r.URL.Path != "/GameStats/AgeII/GetFullStats" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Origin") != "https://www.ageofempires.com" || r.Header.Get("Referer") != "https://www.ageofempires.com/" {
			t.Fatalf("missing ageofempires headers: origin=%q referer=%q", r.Header.Get("Origin"), r.Header.Get("Referer"))
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if body["profileId"] != "5725572" {
			t.Fatalf("profileId = %#v, want string 5725572", body["profileId"])
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"user":{"profileId":5725572,"userName":"chrae","elo":1800,"playerStanding":0.42,"isHuman":true},"careerStats":{"totalGames":10,"unitsKilled":123},"mpStatList":[{"wins":7}]}`))
	}))
	defer server.Close()

	cacheDir := t.TempDir()
	now := time.Date(2026, 7, 21, 10, 0, 0, 0, time.UTC)
	client := NewClient(cacheDir)
	client.BaseURL = server.URL
	client.Now = func() time.Time { return now }

	first, err := client.PlayerStats(PlayerStatsOptions{ProfileID: "5725572", MatchType: 3})
	if err != nil {
		t.Fatalf("first PlayerStats: %v", err)
	}
	if first.Cached || first.User.UserName != "chrae" || first.Verification != "api_reported_career_aggregate_not_per_match_truth" {
		t.Fatalf("bad first report: %+v", first)
	}
	second, err := client.PlayerStats(PlayerStatsOptions{ProfileID: "5725572", MatchType: 3})
	if err != nil {
		t.Fatalf("second PlayerStats: %v", err)
	}
	if !second.Cached || calls != 1 {
		t.Fatalf("cache miss: cached=%v calls=%d", second.Cached, calls)
	}
	if _, err := os.Stat(filepath.Join(cacheDir, "aoems", "player_stats", "profile_5725572_matchtype_3.json")); err != nil {
		t.Fatalf("cache file missing: %v", err)
	}
}

func TestFetchReplayCachesByGameAndProfile(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method != http.MethodGet || r.URL.Path != "/GameStats/AgeII/GetMatchReplay/" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		query := r.URL.Query()
		if query.Get("gameId") != "493687875" || query.Get("profileId") != "5725572" || query.Get("matchId") != "493687875" {
			t.Fatalf("bad query: %s", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/zip")
		w.Header().Set("Content-Disposition", `attachment; filename="AgeIIDE_Replay_493687875.zip"`)
		_, _ = w.Write([]byte("PK\x03\x04fixture"))
	}))
	defer server.Close()

	outDir := t.TempDir()
	client := NewClient(t.TempDir())
	client.BaseURL = server.URL
	first, err := client.FetchReplay(ReplayFetchOptions{GameID: "493687875", ProfileID: "5725572", OutDir: outDir})
	if err != nil {
		t.Fatalf("first FetchReplay: %v", err)
	}
	if first.Cached || !strings.HasSuffix(first.Path, "AgeIIDE_Replay_493687875_p5725572.zip") {
		t.Fatalf("bad first report: %+v", first)
	}
	second, err := client.FetchReplay(ReplayFetchOptions{GameID: "493687875", ProfileID: "5725572", OutDir: outDir})
	if err != nil {
		t.Fatalf("second FetchReplay: %v", err)
	}
	if !second.Cached || calls != 1 {
		t.Fatalf("cache miss: cached=%v calls=%d", second.Cached, calls)
	}
}
