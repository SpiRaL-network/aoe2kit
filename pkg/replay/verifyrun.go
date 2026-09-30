package replay

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
)

type VerifyRunContract struct {
	Name               string                 `json:"name,omitempty"`
	ContextPath        string                 `json:"context,omitempty"`
	Scenario           *ScenarioExpectation   `json:"scenario,omitempty"`
	DataSet            *DataSetExpectation    `json:"data_set,omitempty"`
	Players            *PlayersExpectation    `json:"players,omitempty"`
	PlayerProfile      *ProfileExpectation    `json:"player_profile,omitempty"`
	Telemetry          []TelemetryExpectation `json:"telemetry,omitempty"`
	ForbiddenTelemetry []TelemetryExpectation `json:"forbidden_telemetry,omitempty"`
	Chat               []ChatExpectation      `json:"chat,omitempty"`
	ForbiddenChat      []ChatExpectation      `json:"forbidden_chat,omitempty"`
	Result             *ResultExpectation     `json:"result,omitempty"`
	RenderExpectations []RenderExpectation    `json:"render_expectations,omitempty"`
}

type ScenarioExpectation struct {
	Tier        string `json:"tier,omitempty"`
	Fingerprint string `json:"fingerprint,omitempty"`
}

type DataSetExpectation struct {
	Status         string   `json:"status,omitempty"`
	ActiveDataSet  string   `json:"active_data_set,omitempty"`
	ActiveDataSets []string `json:"active_data_sets,omitempty"`
}

type PlayersExpectation struct {
	MinCount     int   `json:"min_count,omitempty"`
	ExactCount   int   `json:"exact_count,omitempty"`
	RequiredIDs  []int `json:"required_ids,omitempty"`
	ForbiddenIDs []int `json:"forbidden_ids,omitempty"`
}

type ProfileExpectation struct {
	MinReplayActions  int                       `json:"min_replay_actions,omitempty"`
	MinDecodedPercent float64                   `json:"min_decoded_percent,omitempty"`
	Players           []PlayerActionExpectation `json:"players,omitempty"`
}

type PlayerActionExpectation struct {
	PlayerID          int     `json:"player_id"`
	MinActions        int     `json:"min_actions,omitempty"`
	MinDecodedPercent float64 `json:"min_decoded_percent,omitempty"`
}

type TelemetryExpectation struct {
	Name     string            `json:"name"`
	Fields   map[string]string `json:"fields,omitempty"`
	MinCount int               `json:"min_count,omitempty"`
}

type ChatExpectation struct {
	Contains string `json:"contains,omitempty"`
	Exact    string `json:"exact,omitempty"`
	MinCount int    `json:"min_count,omitempty"`
}

type ResultExpectation struct {
	Winners []int `json:"winners,omitempty"`
	Losers  []int `json:"losers,omitempty"`
}

type RenderExpectation struct {
	ID          string `json:"id,omitempty"`
	Description string `json:"description"`
}

type VerifyRunReport struct {
	Replay   string               `json:"replay"`
	Contract string               `json:"contract,omitempty"`
	Name     string               `json:"name,omitempty"`
	OK       bool                 `json:"ok"`
	Summary  VerifyRunSummary     `json:"summary"`
	Story    *StoryReport         `json:"story,omitempty"`
	Profile  *PlayerProfileReport `json:"player_profile,omitempty"`
	Claims   []VerifyRunClaim     `json:"claims"`
	Warnings []string             `json:"warnings,omitempty"`
}

type VerifyRunSummary struct {
	Passed  int `json:"passed"`
	Failed  int `json:"failed"`
	Unknown int `json:"unknown"`
}

type VerifyRunClaim struct {
	Name        string `json:"name"`
	Status      string `json:"status"`
	Tier        string `json:"tier"`
	Source      string `json:"source"`
	Expected    string `json:"expected,omitempty"`
	Observed    string `json:"observed,omitempty"`
	Message     string `json:"message,omitempty"`
	Gotcha      string `json:"gotcha,omitempty"`
	Remediation string `json:"remediation,omitempty"`
}

func LoadVerifyRunContract(path string) (*VerifyRunContract, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var contract VerifyRunContract
	if err := json.Unmarshal(data, &contract); err != nil {
		return nil, fmt.Errorf("%s is not valid verify-run contract JSON: %w", path, err)
	}
	return &contract, nil
}

func VerifyRun(replayPath string, contractPath string) (*VerifyRunReport, error) {
	contract, err := LoadVerifyRunContract(contractPath)
	if err != nil {
		return nil, err
	}
	story, err := BuildStory(replayPath, StoryOptions{ContextPath: contract.ContextPath})
	if err != nil {
		return nil, err
	}
	report := &VerifyRunReport{
		Replay:   replayPath,
		Contract: contractPath,
		Name:     contract.Name,
		Story:    story,
		Warnings: append([]string{}, story.Warnings...),
	}
	report.Claims = append(report.Claims, verifyScenario(contract.Scenario, story)...)
	report.Claims = append(report.Claims, verifyDataSet(contract.DataSet, story)...)
	report.Claims = append(report.Claims, verifyPlayers(contract.Players, story)...)
	if contract.PlayerProfile != nil {
		profile, err := BuildPlayerProfile(replayPath, PlayerProfileOptions{ContextPath: contract.ContextPath})
		if err != nil {
			report.Claims = append(report.Claims, VerifyRunClaim{Name: "player_profile", Status: "unknown", Tier: "logic_verified", Source: "replay_player_profile", Message: "player profile could not be built: " + err.Error()})
		} else {
			report.Profile = profile
			report.Claims = append(report.Claims, verifyPlayerProfile(contract.PlayerProfile, profile)...)
		}
	}
	report.Claims = append(report.Claims, verifyTelemetry(contract.Telemetry, story, false)...)
	report.Claims = append(report.Claims, verifyTelemetry(contract.ForbiddenTelemetry, story, true)...)
	report.Claims = append(report.Claims, verifyChat(contract.Chat, story, false)...)
	report.Claims = append(report.Claims, verifyChat(contract.ForbiddenChat, story, true)...)
	report.Claims = append(report.Claims, verifyResult(contract.Result, story)...)
	report.Claims = append(report.Claims, verifyRenderExpectations(contract.RenderExpectations)...)
	for _, claim := range report.Claims {
		switch claim.Status {
		case "pass":
			report.Summary.Passed++
		case "fail":
			report.Summary.Failed++
		case "unknown":
			report.Summary.Unknown++
		}
	}
	report.OK = report.Summary.Failed == 0 && report.Summary.Unknown == 0
	return report, nil
}

func verifyScenario(expect *ScenarioExpectation, story *StoryReport) []VerifyRunClaim {
	if expect == nil {
		return nil
	}
	expected := expect.Fingerprint
	observed := story.Identity.SHA256
	status := "pass"
	message := "scenario identity matched"
	if expected == "" {
		status = "unknown"
		message = "contract did not provide a scenario fingerprint"
	} else if observed == "" {
		status = "unknown"
		message = "replay scenario identity is unavailable"
	} else if observed != expected {
		status = "fail"
		message = "scenario identity mismatch"
	}
	if expect.Tier != "" && story.Identity.Tier != "" && expect.Tier != story.Identity.Tier && status == "pass" {
		status = "fail"
		message = "scenario identity tier mismatch"
	}
	return []VerifyRunClaim{{
		Name:     "scenario_identity",
		Status:   status,
		Tier:     "logic_verified",
		Source:   scenarioIdentitySource(story.Identity),
		Expected: formatScenarioExpectation(expect),
		Observed: formatScenarioObservation(story.Identity),
		Message:  message,
	}}
}

func verifyDataSet(expect *DataSetExpectation, story *StoryReport) []VerifyRunClaim {
	if expect == nil {
		return nil
	}
	expected := normalizeDataSetExpectation(expect)
	observed := normalizeDataSetObservation(story.DataSet)
	status := "pass"
	message := "active data set matched"
	var gotcha, remediation string
	if expected == "" {
		status = "unknown"
		message = "contract did not provide a data-set expectation"
	} else if story.DataSet.Status == "" || story.DataSet.Status == "unknown" {
		status = "unknown"
		message = "replay active data set is unavailable"
	} else if expected != observed {
		status = "fail"
		message = "active data set mismatch"
		if story.DataSet.Status == "vanilla" && expect.Status == "modded" {
			gotcha = "Data-Mod Lock"
			remediation = "Publish/update the data mod and select it in the Data-Mod dropdown at game creation; otherwise AoE2DE silently runs the vanilla data set."
		}
	}
	return []VerifyRunClaim{{
		Name:        "data_set_identity",
		Status:      status,
		Tier:        "logic_verified",
		Source:      "replay_header.embedded_mod_block",
		Expected:    expected,
		Observed:    observed,
		Message:     message,
		Gotcha:      gotcha,
		Remediation: remediation,
	}}
}

func verifyPlayers(expect *PlayersExpectation, story *StoryReport) []VerifyRunClaim {
	if expect == nil {
		return nil
	}
	ids := make([]int, 0, len(story.Players))
	present := map[int]bool{}
	for _, player := range story.Players {
		if player.PlayerID <= 0 {
			continue
		}
		ids = append(ids, player.PlayerID)
		present[player.PlayerID] = true
	}
	sort.Ints(ids)
	var claims []VerifyRunClaim
	if expect.MinCount > 0 {
		status := "pass"
		message := "minimum player count matched"
		if len(ids) < expect.MinCount {
			status = "fail"
			message = "minimum player count not met"
		}
		claims = append(claims, VerifyRunClaim{Name: "players_min_count", Status: status, Tier: "logic_verified", Source: "replay_header.player_roster", Expected: fmt.Sprintf("min_count=%d", expect.MinCount), Observed: fmt.Sprintf("count=%d ids=%v", len(ids), ids), Message: message})
	}
	if expect.ExactCount > 0 {
		status := "pass"
		message := "exact player count matched"
		if len(ids) != expect.ExactCount {
			status = "fail"
			message = "exact player count mismatch"
		}
		claims = append(claims, VerifyRunClaim{Name: "players_exact_count", Status: status, Tier: "logic_verified", Source: "replay_header.player_roster", Expected: fmt.Sprintf("exact_count=%d", expect.ExactCount), Observed: fmt.Sprintf("count=%d ids=%v", len(ids), ids), Message: message})
	}
	for _, playerID := range expect.RequiredIDs {
		status := "pass"
		message := "required player was present"
		if !present[playerID] {
			status = "fail"
			message = "required player was not present"
		}
		claims = append(claims, VerifyRunClaim{Name: "player_required", Status: status, Tier: "logic_verified", Source: "replay_header.player_roster", Expected: fmt.Sprintf("P%d present", playerID), Observed: fmt.Sprintf("ids=%v", ids), Message: message})
	}
	for _, playerID := range expect.ForbiddenIDs {
		status := "pass"
		message := "forbidden player was absent"
		if present[playerID] {
			status = "fail"
			message = "forbidden player was present"
		}
		claims = append(claims, VerifyRunClaim{Name: "player_forbidden", Status: status, Tier: "logic_verified", Source: "replay_header.player_roster", Expected: fmt.Sprintf("P%d absent", playerID), Observed: fmt.Sprintf("ids=%v", ids), Message: message})
	}
	if len(claims) == 0 {
		claims = append(claims, VerifyRunClaim{Name: "players", Status: "unknown", Tier: "logic_verified", Source: "replay_header.player_roster", Message: "players expectation block did not contain any checks"})
	}
	return claims
}

func verifyPlayerProfile(expect *ProfileExpectation, profile *PlayerProfileReport) []VerifyRunClaim {
	if expect == nil {
		return nil
	}
	var claims []VerifyRunClaim
	if expect.MinReplayActions > 0 {
		status := "pass"
		message := "minimum replay action count matched"
		if profile.Coverage.ReplayActions < expect.MinReplayActions {
			status = "fail"
			message = "minimum replay action count not met"
		}
		claims = append(claims, VerifyRunClaim{Name: "profile_min_replay_actions", Status: status, Tier: "logic_verified", Source: "replay_player_profile.coverage", Expected: fmt.Sprintf("min_replay_actions=%d", expect.MinReplayActions), Observed: fmt.Sprintf("replay_actions=%d", profile.Coverage.ReplayActions), Message: message})
	}
	if expect.MinDecodedPercent > 0 {
		status := "pass"
		message := "minimum decoded action coverage matched"
		if profile.Coverage.DecodedPercent < expect.MinDecodedPercent {
			status = "fail"
			message = "minimum decoded action coverage not met"
		}
		claims = append(claims, VerifyRunClaim{Name: "profile_min_decoded_percent", Status: status, Tier: "logic_verified", Source: "replay_player_profile.coverage", Expected: fmt.Sprintf("min_decoded_percent=%.2f", expect.MinDecodedPercent), Observed: fmt.Sprintf("decoded_percent=%.2f", profile.Coverage.DecodedPercent), Message: message})
	}
	byID := map[int]PlayerProfile{}
	for _, player := range profile.Players {
		byID[player.PlayerID] = player
	}
	for _, expectedPlayer := range expect.Players {
		player, ok := byID[expectedPlayer.PlayerID]
		if !ok {
			claims = append(claims, VerifyRunClaim{Name: "profile_player", Status: "fail", Tier: "logic_verified", Source: "replay_player_profile.players", Expected: fmt.Sprintf("P%d present", expectedPlayer.PlayerID), Message: "expected player profile was absent"})
			continue
		}
		if expectedPlayer.MinActions > 0 {
			status := "pass"
			message := "minimum player action count matched"
			if player.Actions < expectedPlayer.MinActions {
				status = "fail"
				message = "minimum player action count not met"
			}
			claims = append(claims, VerifyRunClaim{Name: "profile_player_min_actions", Status: status, Tier: "logic_verified", Source: "replay_player_profile.players", Expected: fmt.Sprintf("P%d min_actions=%d", expectedPlayer.PlayerID, expectedPlayer.MinActions), Observed: fmt.Sprintf("actions=%d", player.Actions), Message: message})
		}
		if expectedPlayer.MinDecodedPercent > 0 {
			decodedPercent := 0.0
			if player.Actions > 0 {
				decodedPercent = round2(float64(player.DecodedActions) * 100 / float64(player.Actions))
			}
			status := "pass"
			message := "minimum player decoded coverage matched"
			if decodedPercent < expectedPlayer.MinDecodedPercent {
				status = "fail"
				message = "minimum player decoded coverage not met"
			}
			claims = append(claims, VerifyRunClaim{Name: "profile_player_min_decoded_percent", Status: status, Tier: "logic_verified", Source: "replay_player_profile.players", Expected: fmt.Sprintf("P%d min_decoded_percent=%.2f", expectedPlayer.PlayerID, expectedPlayer.MinDecodedPercent), Observed: fmt.Sprintf("decoded_percent=%.2f", decodedPercent), Message: message})
		}
	}
	if len(claims) == 0 {
		claims = append(claims, VerifyRunClaim{Name: "player_profile", Status: "unknown", Tier: "logic_verified", Source: "replay_player_profile", Message: "player_profile expectation block did not contain any checks"})
	}
	return claims
}

func verifyTelemetry(expectations []TelemetryExpectation, story *StoryReport, forbidden bool) []VerifyRunClaim {
	var claims []VerifyRunClaim
	for _, expect := range expectations {
		minCount := expect.MinCount
		if minCount <= 0 {
			minCount = 1
		}
		count := 0
		matchExpect := expect
		if forbidden {
			matchExpect.Fields = nil
		}
		for _, line := range story.Telemetry {
			if telemetryLineMatches(line.Text, matchExpect) {
				count++
			}
		}
		status := "pass"
		message := "telemetry expectation matched"
		name := "telemetry_required"
		expected := fmt.Sprintf("%s min_count=%d", expect.Name, minCount)
		if len(expect.Fields) > 0 {
			expected += " fields=" + formatStringMap(expect.Fields)
		}
		if forbidden {
			name = "telemetry_forbidden"
			expected = expect.Name
			if count > 0 {
				status = "fail"
				message = "forbidden telemetry marker was observed"
			} else {
				message = "forbidden telemetry marker was absent"
			}
		} else if count < minCount {
			status = "fail"
			message = "required telemetry marker was not observed enough times"
		}
		claims = append(claims, VerifyRunClaim{
			Name:     name,
			Status:   status,
			Tier:     "logic_verified",
			Source:   "replay_chat.telemetry_markers",
			Expected: expected,
			Observed: fmt.Sprintf("count=%d", count),
			Message:  message,
		})
	}
	return claims
}

func verifyChat(expectations []ChatExpectation, story *StoryReport, forbidden bool) []VerifyRunClaim {
	var claims []VerifyRunClaim
	for _, expect := range expectations {
		minCount := expect.MinCount
		if minCount <= 0 {
			minCount = 1
		}
		count := 0
		for _, line := range story.Feedback {
			if chatLineMatches(line.Text, expect) {
				count++
			}
		}
		status := "pass"
		message := "chat expectation matched"
		name := "chat_required"
		expected := formatChatExpectation(expect, minCount)
		if forbidden {
			name = "chat_forbidden"
			if count > 0 {
				status = "fail"
				message = "forbidden chat marker was observed"
			} else {
				message = "forbidden chat marker was absent"
			}
		} else if count < minCount {
			status = "fail"
			message = "required chat marker was not observed enough times"
		}
		claims = append(claims, VerifyRunClaim{
			Name:     name,
			Status:   status,
			Tier:     "logic_verified",
			Source:   "replay_chat.messages",
			Expected: expected,
			Observed: fmt.Sprintf("count=%d", count),
			Message:  message,
		})
	}
	return claims
}

func verifyResult(expect *ResultExpectation, story *StoryReport) []VerifyRunClaim {
	if expect == nil {
		return nil
	}
	status := "pass"
	message := "result matched"
	if !story.Result.WinnerKnown {
		status = "unknown"
		message = "replay result is unavailable"
	} else if !sameInts(expect.Winners, story.Result.Winners) || !sameInts(expect.Losers, story.Result.Losers) {
		status = "fail"
		message = "result mismatch"
	}
	return []VerifyRunClaim{{
		Name:     "result",
		Status:   status,
		Tier:     "logic_verified",
		Source:   "replay_actions.resign",
		Expected: fmt.Sprintf("winners=%v losers=%v", sortedInts(expect.Winners), sortedInts(expect.Losers)),
		Observed: fmt.Sprintf("winners=%v losers=%v", story.Result.Winners, story.Result.Losers),
		Message:  message,
	}}
}

func verifyRenderExpectations(expectations []RenderExpectation) []VerifyRunClaim {
	var claims []VerifyRunClaim
	for _, expect := range expectations {
		name := "render_expectation"
		if expect.ID != "" {
			name += ":" + expect.ID
		}
		claims = append(claims, VerifyRunClaim{
			Name:     name,
			Status:   "unknown",
			Tier:     "render_verified",
			Source:   "render_oracle_missing",
			Expected: expect.Description,
			Message:  "render expectation is pre-registered but cannot pass without human screenshot/clip evidence",
		})
	}
	return claims
}

func telemetryLineMatches(text string, expect TelemetryExpectation) bool {
	if text == "" {
		return false
	}
	if len(text) > len("telemetry ") && text[:len("telemetry ")] == "telemetry " {
		text = text[len("telemetry "):]
	}
	telemetry := ParseTelemetry(text, nil)
	if telemetry == nil || telemetry.Name != expect.Name {
		return false
	}
	for key, value := range expect.Fields {
		if telemetry.Fields[key] != value {
			return false
		}
	}
	return true
}

func chatLineMatches(text string, expect ChatExpectation) bool {
	if expect.Exact != "" && text != fmt.Sprintf("%q", expect.Exact) && text != expect.Exact {
		return false
	}
	if expect.Contains != "" && !containsText(text, expect.Contains) {
		return false
	}
	return expect.Exact != "" || expect.Contains != ""
}

func containsText(text string, needle string) bool {
	for i := 0; i+len(needle) <= len(text); i++ {
		if text[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}

func formatScenarioExpectation(expect *ScenarioExpectation) string {
	if expect == nil {
		return ""
	}
	if expect.Tier != "" {
		return expect.Tier + ":" + expect.Fingerprint
	}
	return expect.Fingerprint
}

func formatScenarioObservation(identity StoryIdentity) string {
	if identity.Tier != "" {
		return identity.Tier + ":" + identity.SHA256
	}
	return identity.SHA256
}

func scenarioIdentitySource(identity StoryIdentity) string {
	if identity.Tier == "trigger_graph" {
		return "replay_header.embedded_trigger_graph"
	}
	if identity.Tier != "" {
		return "replay_header." + identity.Tier
	}
	return "replay_header.scenario_identity"
}

func normalizeDataSetExpectation(expect *DataSetExpectation) string {
	if expect == nil {
		return ""
	}
	if expect.Status == "vanilla" {
		return "vanilla"
	}
	names := expect.ActiveDataSets
	if len(names) == 0 && expect.ActiveDataSet != "" {
		names = []string{expect.ActiveDataSet}
	}
	sort.Strings(names)
	if expect.Status == "modded" || len(names) > 0 {
		return "modded:" + joinStrings(names)
	}
	return expect.Status
}

func normalizeDataSetObservation(identity DataSetIdentity) string {
	if identity.Status == "vanilla" {
		return "vanilla"
	}
	if identity.Status == "modded" {
		names := append([]string{}, identity.ActiveDataSets...)
		if len(names) == 0 && identity.ActiveDataSet != "" {
			names = []string{identity.ActiveDataSet}
		}
		sort.Strings(names)
		return "modded:" + joinStrings(names)
	}
	if identity.Status == "" {
		return "unknown"
	}
	return identity.Status
}

func formatStringMap(values map[string]string) string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]string, 0, len(keys))
	for _, key := range keys {
		out = append(out, key+"="+values[key])
	}
	return joinStrings(out)
}

func formatChatExpectation(expect ChatExpectation, minCount int) string {
	if expect.Exact != "" {
		return fmt.Sprintf("exact=%q min_count=%d", expect.Exact, minCount)
	}
	return fmt.Sprintf("contains=%q min_count=%d", expect.Contains, minCount)
}

func sameInts(a []int, b []int) bool {
	return joinInts(sortedInts(a)) == joinInts(sortedInts(b))
}

func sortedInts(values []int) []int {
	out := append([]int{}, values...)
	sort.Ints(out)
	return out
}

func joinStrings(values []string) string {
	if len(values) == 0 {
		return ""
	}
	out := values[0]
	for _, value := range values[1:] {
		out += "," + value
	}
	return out
}

func joinInts(values []int) string {
	out := ""
	for i, value := range values {
		if i > 0 {
			out += ","
		}
		out += fmt.Sprintf("%d", value)
	}
	return out
}
