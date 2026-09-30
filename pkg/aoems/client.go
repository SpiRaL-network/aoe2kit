package aoems

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	DefaultBaseURL = "https://api.ageofempires.com/api"
	DefaultTTL     = 120 * time.Second
)

type Client struct {
	BaseURL    string
	HTTPClient *http.Client
	CacheDir   string
	TTL        time.Duration
	UserAgent  string
	Now        func() time.Time
}

type PlayerStatsOptions struct {
	ProfileID string
	MatchType int
	Force     bool
}

type PlayerStatsReport struct {
	ProfileID    string          `json:"profile_id"`
	MatchType    int             `json:"match_type"`
	Method       string          `json:"method"`
	Verification string          `json:"verification"`
	URL          string          `json:"url"`
	Cached       bool            `json:"cached"`
	CachePath    string          `json:"cache_path,omitempty"`
	FetchedAt    string          `json:"fetched_at,omitempty"`
	AgeSeconds   int64           `json:"age_seconds,omitempty"`
	StatusCode   int             `json:"status_code,omitempty"`
	User         PlayerStatsUser `json:"user,omitempty"`
	CareerStats  map[string]any  `json:"career_stats,omitempty"`
	MPStatList   any             `json:"mp_stat_list,omitempty"`
	Raw          json.RawMessage `json:"raw,omitempty"`
	Warnings     []string        `json:"warnings,omitempty"`
}

type PlayerStatsUser struct {
	ProfileID            any     `json:"profileId,omitempty"`
	UserName             string  `json:"userName,omitempty"`
	ELO                  any     `json:"elo,omitempty"`
	PlayerStanding       any     `json:"playerStanding,omitempty"`
	PlayerStandingNumber float64 `json:"player_standing_number,omitempty"`
	AvatarURL            string  `json:"avatarUrl,omitempty"`
	IsHuman              *bool   `json:"isHuman,omitempty"`
	MatchReplayAvailable *bool   `json:"matchReplayAvailable,omitempty"`
}

type ReplayFetchOptions struct {
	GameID    string
	ProfileID string
	OutDir    string
	Force     bool
}

type ReplayFetchReport struct {
	GameID       string   `json:"game_id"`
	ProfileID    string   `json:"profile_id"`
	Method       string   `json:"method"`
	Verification string   `json:"verification"`
	URL          string   `json:"url"`
	Path         string   `json:"path"`
	Cached       bool     `json:"cached"`
	Bytes        int64    `json:"bytes"`
	StatusCode   int      `json:"status_code,omitempty"`
	ContentType  string   `json:"content_type,omitempty"`
	Warnings     []string `json:"warnings,omitempty"`
}

type ReplayFetchAllReport struct {
	GameID       string              `json:"game_id"`
	SeedProfile  string              `json:"seed_profile_id,omitempty"`
	Method       string              `json:"method"`
	Verification string              `json:"verification"`
	OutDir       string              `json:"out_dir"`
	Fetched      []ReplayFetchReport `json:"fetched"`
	Profiles     []string            `json:"profiles"`
	Warnings     []string            `json:"warnings,omitempty"`
}

func NewClient(cacheDir string) *Client {
	return &Client{
		BaseURL:    DefaultBaseURL,
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
		CacheDir:   cacheDir,
		TTL:        DefaultTTL,
		UserAgent:  "AoE2Kit/0.1 (+https://www.ageofempires.com/)",
		Now:        time.Now,
	}
}

func DefaultCacheDir() string {
	if root := os.Getenv("AOE2KIT_CACHE_DIR"); root != "" {
		return root
	}
	if root, err := os.UserCacheDir(); err == nil && root != "" {
		return filepath.Join(root, "aoe2kit")
	}
	return ".aoe2kit-cache"
}

func (c *Client) PlayerStats(opts PlayerStatsOptions) (*PlayerStatsReport, error) {
	if opts.ProfileID == "" {
		return nil, fmt.Errorf("profile id is required")
	}
	if opts.MatchType == 0 {
		opts.MatchType = 3
	}
	if err := validateNumericString(opts.ProfileID, "profile id"); err != nil {
		return nil, err
	}
	cachePath := c.playerStatsCachePath(opts.ProfileID, opts.MatchType)
	if !opts.Force {
		if report, ok := c.readStatsCache(cachePath); ok {
			report.Cached = true
			report.CachePath = cachePath
			report.AgeSeconds = int64(c.now().Sub(fileModTime(cachePath)).Seconds())
			return report, nil
		}
	}
	endpoint := strings.TrimRight(c.baseURL(), "/") + "/GameStats/AgeII/GetFullStats"
	body, err := json.Marshal(map[string]any{
		"profileId":    opts.ProfileID,
		"gamertag":     "",
		"playerNumber": 0,
		"gameId":       0,
		"matchType":    opts.MatchType,
	})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	c.setJSONHeaders(req)
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusNoContent {
		return nil, fmt.Errorf("GetFullStats returned 204 no content for profile %s", opts.ProfileID)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("GetFullStats returned HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	report, err := parsePlayerStats(raw)
	if err != nil {
		return nil, err
	}
	report.ProfileID = opts.ProfileID
	report.MatchType = opts.MatchType
	report.Method = "microsoft_ageii_get_full_stats"
	report.Verification = "api_reported_career_aggregate_not_per_match_truth"
	report.URL = endpoint
	report.Cached = false
	report.CachePath = cachePath
	report.FetchedAt = c.now().UTC().Format(time.RFC3339)
	report.StatusCode = resp.StatusCode
	if err := writeJSONCache(cachePath, report); err != nil {
		report.Warnings = append(report.Warnings, "cache write failed: "+err.Error())
	}
	return report, nil
}

func (c *Client) FetchReplay(opts ReplayFetchOptions) (*ReplayFetchReport, error) {
	if opts.GameID == "" {
		return nil, fmt.Errorf("game id is required")
	}
	if opts.ProfileID == "" {
		return nil, fmt.Errorf("profile id is required")
	}
	if err := validateNumericString(opts.GameID, "game id"); err != nil {
		return nil, err
	}
	if err := validateNumericString(opts.ProfileID, "profile id"); err != nil {
		return nil, err
	}
	outDir := opts.OutDir
	if outDir == "" {
		outDir = "."
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return nil, err
	}
	outPath := ReplayCachePath(outDir, opts.GameID, opts.ProfileID)
	endpoint := c.replayURL(opts.GameID, opts.ProfileID)
	report := &ReplayFetchReport{
		GameID:       opts.GameID,
		ProfileID:    opts.ProfileID,
		Method:       "microsoft_ageii_get_match_replay",
		Verification: "downloaded_public_replay_or_cache_not_engine_verified",
		URL:          endpoint,
		Path:         outPath,
	}
	if !opts.Force {
		if info, err := os.Stat(outPath); err == nil && info.Size() > 0 {
			report.Cached = true
			report.Bytes = info.Size()
			return report, nil
		}
	}
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	c.setCommonHeaders(req)
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	report.StatusCode = resp.StatusCode
	report.ContentType = resp.Header.Get("Content-Type")
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("GetMatchReplay returned HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	if len(raw) == 0 {
		return nil, fmt.Errorf("GetMatchReplay returned empty body")
	}
	if !looksZip(raw) {
		return nil, fmt.Errorf("GetMatchReplay response is not a zip payload: content_type=%q bytes=%d", report.ContentType, len(raw))
	}
	if name := replayFilenameFromDisposition(resp.Header.Get("Content-Disposition")); name != "" {
		report.Warnings = append(report.Warnings, "server filename preserved in warning only; AoE2Kit stores per-POV cache filename: "+name)
	}
	temp := outPath + ".tmp"
	if err := os.WriteFile(temp, raw, 0o644); err != nil {
		return nil, err
	}
	if err := os.Rename(temp, outPath); err != nil {
		_ = os.Remove(temp)
		return nil, err
	}
	report.Bytes = int64(len(raw))
	return report, nil
}

func ReplayCachePath(outDir string, gameID string, profileID string) string {
	name := fmt.Sprintf("AgeIIDE_Replay_%s_p%s.zip", gameID, profileID)
	return filepath.Join(outDir, name)
}

func (c *Client) replayURL(gameID string, profileID string) string {
	values := url.Values{}
	values.Set("gameId", gameID)
	values.Set("profileId", profileID)
	values.Set("matchId", gameID)
	return strings.TrimRight(c.baseURL(), "/") + "/GameStats/AgeII/GetMatchReplay/?" + values.Encode()
}

func parsePlayerStats(raw []byte) (*PlayerStatsReport, error) {
	var envelope struct {
		User        PlayerStatsUser `json:"user"`
		CareerStats map[string]any  `json:"careerStats"`
		MPStatList  any             `json:"mpStatList"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, err
	}
	report := &PlayerStatsReport{
		User:        envelope.User,
		CareerStats: envelope.CareerStats,
		MPStatList:  envelope.MPStatList,
		Raw:         append([]byte(nil), raw...),
	}
	report.User.PlayerStandingNumber = numberFromAny(report.User.PlayerStanding)
	if report.User.UserName == "" && len(envelope.CareerStats) == 0 && envelope.MPStatList == nil {
		report.Warnings = append(report.Warnings, "GetFullStats JSON parsed but known fields were empty")
	}
	return report, nil
}

func (c *Client) readStatsCache(path string) (*PlayerStatsReport, bool) {
	info, err := os.Stat(path)
	if err != nil || info.Size() == 0 {
		return nil, false
	}
	ttl := c.TTL
	if ttl <= 0 {
		ttl = DefaultTTL
	}
	if c.now().Sub(info.ModTime()) > ttl {
		return nil, false
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	var report PlayerStatsReport
	if err := json.Unmarshal(raw, &report); err != nil {
		return nil, false
	}
	return &report, true
}

func (c *Client) playerStatsCachePath(profileID string, matchType int) string {
	root := c.CacheDir
	if root == "" {
		root = DefaultCacheDir()
	}
	return filepath.Join(root, "aoems", "player_stats", fmt.Sprintf("profile_%s_matchtype_%d.json", profileID, matchType))
}

func writeJSONCache(path string, value any) error {
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(raw, '\n'), 0o644)
}

func (c *Client) setJSONHeaders(req *http.Request) {
	c.setCommonHeaders(req)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Origin", "https://www.ageofempires.com")
	req.Header.Set("Referer", "https://www.ageofempires.com/")
}

func (c *Client) setCommonHeaders(req *http.Request) {
	req.Header.Set("User-Agent", c.userAgent())
}

func (c *Client) baseURL() string {
	if c.BaseURL != "" {
		return c.BaseURL
	}
	return DefaultBaseURL
}

func (c *Client) httpClient() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return &http.Client{Timeout: 30 * time.Second}
}

func (c *Client) userAgent() string {
	if c.UserAgent != "" {
		return c.UserAgent
	}
	return "AoE2Kit/0.1"
}

func (c *Client) now() time.Time {
	if c.Now != nil {
		return c.Now()
	}
	return time.Now()
}

func fileModTime(path string) time.Time {
	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}
	}
	return info.ModTime()
}

func validateNumericString(value string, label string) error {
	if value == "" {
		return fmt.Errorf("%s is required", label)
	}
	if _, err := strconv.ParseInt(value, 10, 64); err != nil {
		return fmt.Errorf("%s must be numeric: %q", label, value)
	}
	return nil
}

func looksZip(raw []byte) bool {
	return len(raw) >= 4 && raw[0] == 'P' && raw[1] == 'K' && raw[2] == 0x03 && raw[3] == 0x04
}

func replayFilenameFromDisposition(value string) string {
	if value == "" {
		return ""
	}
	_, params, err := mime.ParseMediaType(value)
	if err != nil {
		return ""
	}
	return params["filename"]
}

func numberFromAny(value any) float64 {
	switch v := value.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case json.Number:
		f, _ := v.Float64()
		return f
	default:
		return 0
	}
}
