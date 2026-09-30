package replay

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"aoe2kit/pkg/aoe2"
)

type ChatReport struct {
	Path         string                 `json:"path,omitempty"`
	Method       string                 `json:"method"`
	Verification aoe2.VerificationClaim `json:"verification"`
	Players      []FeedbackPlayer       `json:"players,omitempty"`
	Lines        []ChatLine             `json:"lines"`
	Counts       ChatCounts             `json:"counts"`
	Warnings     []string               `json:"warnings,omitempty"`
}

type ChatCounts struct {
	Total            int `json:"total"`
	Lobby            int `json:"lobby"`
	Backlog          int `json:"backlog"`
	Game             int `json:"game"`
	DuplicatesHidden int `json:"duplicates_hidden"`
}

type ChatLine struct {
	Index          int    `json:"index"`
	Source         string `json:"source"`
	TimeMS         int    `json:"time_ms,omitempty"`
	Time           string `json:"time,omitempty"`
	PlayerID       int    `json:"player_id,omitempty"`
	PlayerName     string `json:"player_name,omitempty"`
	Channel        int    `json:"channel,omitempty"`
	ChannelName    string `json:"channel_name,omitempty"`
	PlatformIcon   string `json:"platform_icon,omitempty"`
	Text           string `json:"text"`
	TauntNumber    int    `json:"taunt_number,omitempty"`
	TauntText      string `json:"taunt_text,omitempty"`
	DestinationMap int    `json:"destination_map,omitempty"`
	MessageAGP     string `json:"message_agp,omitempty"`
	SourceOffset   int    `json:"source_offset,omitempty"`
	Confidence     string `json:"confidence"`
}

func ExtractChat(path string) (*ChatReport, error) {
	cache, err := OpenReplayCache(path)
	if err != nil {
		return nil, err
	}
	rec, err := cache.Metadata()
	if err != nil {
		return nil, err
	}
	bodyEvents, err := ExtractEvents(path, EventOptions{})
	if err != nil {
		return nil, err
	}
	nameByID := map[int]string{}
	idByName := map[string]int{}
	for _, player := range bodyEvents.Players {
		if player.PlayerID > 0 && player.Name != "" {
			nameByID[player.PlayerID] = player.Name
			idByName[normalizeChatKey(player.Name)] = player.PlayerID
		}
	}
	for _, slot := range rec.Players {
		if slot.Number > 0 && slot.Name != "" {
			nameByID[slot.Number] = slot.Name
			idByName[normalizeChatKey(slot.Name)] = slot.Number
		}
	}

	lobby, err := cache.LobbyChat(idByName)
	if err != nil {
		return nil, err
	}
	var warnings []string
	if len(lobby) == 0 {
		warnings = append(warnings, "no header lobby AGP chat lines found")
	}
	for _, warning := range bodyEvents.Warnings {
		if isBacklogTagWarning(warning) {
			continue
		}
		warnings = append(warnings, warning)
	}

	lines := make([]ChatLine, 0, len(lobby)+bodyEvents.Counts.Chat)
	seenLobby := map[string]bool{}
	for _, line := range lobby {
		if line.PlayerID > 0 && line.PlayerName == "" {
			line.PlayerName = nameByID[line.PlayerID]
		}
		lines = append(lines, line)
		seenLobby[chatDedupeKey(line)] = true
	}
	duplicates := 0
	bodyLines := make([]ChatLine, 0, bodyEvents.Counts.Chat)
	for _, event := range bodyEvents.Events {
		if event.Type != "chat" {
			continue
		}
		line := chatLineFromReplayEvent(event)
		if line.PlayerID > 0 && line.PlayerName == "" {
			line.PlayerName = nameByID[line.PlayerID]
		}
		key := chatDedupeKey(line)
		if line.TimeMS <= 10000 && seenLobby[key] {
			duplicates++
			continue
		}
		bodyLines = append(bodyLines, line)
	}
	backlog := tagInitialChatBacklog(bodyLines)
	if backlog > 0 {
		warnings = appendUniqueWarning(warnings, fmt.Sprintf("tagged %d same-timestamp body chat lines as source=backlog; DE can inject prior session chat into a new recording", backlog))
	}
	lines = append(lines, bodyLines...)
	sort.SliceStable(lines, func(i, j int) bool {
		if lines[i].TimeMS != lines[j].TimeMS {
			return lines[i].TimeMS < lines[j].TimeMS
		}
		if lines[i].Source != lines[j].Source {
			return lines[i].Source < lines[j].Source
		}
		return lines[i].SourceOffset < lines[j].SourceOffset
	})
	counts := ChatCounts{DuplicatesHidden: duplicates}
	for i := range lines {
		lines[i].Index = i + 1
		switch lines[i].Source {
		case "lobby":
			counts.Lobby++
		case "backlog":
			counts.Backlog++
		case "game":
			counts.Game++
		}
	}
	if counts.Backlog > 0 {
		warnings = appendUniqueWarning(warnings, fmt.Sprintf("tagged %d visible body chat lines as source=backlog; DE can inject prior session chat into a new recording", counts.Backlog))
	}
	counts.Total = len(lines)
	players := feedbackPlayers(rec.Players, nil, nil)
	return &ChatReport{
		Path:   path,
		Method: "header_agp_lobby_plus_replay_body_chat",
		Verification: aoe2.VerificationClaim{
			Label:             "structure_verified_not_engine_verified",
			StructureVerified: true,
			EngineVerified:    false,
			Note:              "Chat report parses inflated-header AGP lobby strings and replay body chat JSON/action-stream records; semantic completeness depends on DE replay retention behavior.",
		},
		Players:  players,
		Lines:    lines,
		Counts:   counts,
		Warnings: warnings,
	}, nil
}

func tagInitialChatBacklog(lines []ChatLine) int {
	const (
		backlogWindowMS       = 120000
		backlogRepeatWindowMS = 60000
		minBacklogBurst       = 5
	)
	if len(lines) < minBacklogBurst {
		return 0
	}
	bursts := map[int]int{}
	for _, line := range lines {
		if line.Source != "game" || line.TimeMS < 0 || line.TimeMS > backlogWindowMS {
			continue
		}
		bursts[line.TimeMS]++
	}
	tagged := 0
	for i := range lines {
		if lines[i].Source != "game" || lines[i].TimeMS < 0 || lines[i].TimeMS > backlogWindowMS {
			continue
		}
		if bursts[lines[i].TimeMS] < minBacklogBurst || shouldPreserveDiagnosticChatMarker(lines[i].Text, lines[i].TimeMS) {
			continue
		}
		lines[i].Source = "backlog"
		lines[i].Confidence = appendConfidence(lines[i].Confidence, "same_timestamp_backlog_heuristic")
		tagged++
	}
	backlogTexts := map[string]bool{}
	for _, line := range lines {
		if line.Source != "backlog" {
			continue
		}
		if text := normalizedBacklogText(line.Text); text != "" {
			backlogTexts[text] = true
		}
	}
	for i := range lines {
		if lines[i].Source != "game" || lines[i].TimeMS < 0 || lines[i].TimeMS > backlogRepeatWindowMS {
			continue
		}
		if shouldPreserveDiagnosticChatMarker(lines[i].Text, lines[i].TimeMS) {
			continue
		}
		if !backlogTexts[normalizedBacklogText(lines[i].Text)] {
			continue
		}
		lines[i].Source = "backlog"
		lines[i].Confidence = appendConfidence(lines[i].Confidence, "exact_backlog_text_repeat_heuristic")
		tagged++
	}
	return tagged
}

func normalizedBacklogText(text string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(text)), " ")
}

func shouldPreserveDiagnosticChatMarker(text string, timeMS int) bool {
	if timeMS == 0 {
		return false
	}
	return isDiagnosticChatMarker(text)
}

func isDiagnosticChatMarker(text string) bool {
	fields := strings.Fields(strings.TrimSpace(text))
	if len(fields) < 2 {
		return false
	}
	phase := fields[0]
	if strings.HasPrefix(strings.ToUpper(phase), "V") && len(phase) > 1 {
		phase = phase[1:]
	}
	if len(phase) != 3 {
		return false
	}
	for _, r := range phase {
		if r < '0' || r > '9' {
			return false
		}
	}
	switch strings.ToLower(fields[1]) {
	case "start", "done", "blocked":
		return true
	default:
		return false
	}
}

func appendConfidence(existing, tag string) string {
	if strings.TrimSpace(existing) == "" {
		return tag
	}
	if strings.Contains(existing, tag) {
		return existing
	}
	return existing + "+" + tag
}

func appendUniqueWarning(warnings []string, warning string) []string {
	for _, existing := range warnings {
		if existing == warning {
			return warnings
		}
	}
	return append(warnings, warning)
}

func isBacklogTagWarning(warning string) bool {
	return strings.HasPrefix(warning, "tagged ") && strings.Contains(warning, "source=backlog")
}

var headerLobbyChatRE = regexp.MustCompile(`@#([0-9]{1,3})\s+<([^>\x00\r\n]{1,96})>\s+([^:\x00\r\n]{1,64}):\s*([^\x00\r\n]{0,512})`)

func extractHeaderLobbyChat(header []byte, idByName map[string]int) []ChatLine {
	matches := headerLobbyChatRE.FindAllStringSubmatchIndex(string(header), -1)
	out := make([]ChatLine, 0, len(matches))
	seen := map[string]bool{}
	asString := string(header)
	for _, match := range matches {
		raw := asString[match[0]:match[1]]
		slotText := asString[match[2]:match[3]]
		icon := strings.TrimSpace(asString[match[4]:match[5]])
		name := strings.TrimSpace(asString[match[6]:match[7]])
		text := strings.TrimSpace(asString[match[8]:match[9]])
		if !plausiblePlayerName(name) || !plausibleChatText(text) {
			continue
		}
		playerID, _ := strconv.Atoi(slotText)
		if mapped := idByName[normalizeChatKey(name)]; mapped > 0 {
			playerID = mapped
		}
		line := ChatLine{
			Source:       "lobby",
			PlayerID:     playerID,
			PlayerName:   name,
			ChannelName:  "lobby",
			PlatformIcon: icon,
			Text:         text,
			MessageAGP:   raw,
			SourceOffset: match[0],
			Confidence:   "header_agp_string",
		}
		key := fmt.Sprintf("%d\x00%s\x00%s", line.SourceOffset, normalizeChatKey(name), normalizeChatKey(text))
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, line)
	}
	return out
}

func chatLineFromReplayEvent(event ReplayEvent) ChatLine {
	source := "game"
	if event.Source == "backlog" {
		source = "backlog"
	}
	return ChatLine{
		Source:         source,
		TimeMS:         event.TimeMS,
		Time:           event.Time,
		PlayerID:       event.PlayerID,
		PlayerName:     event.PlayerName,
		Channel:        event.Channel,
		ChannelName:    event.ChannelName,
		Text:           event.Text,
		TauntNumber:    event.TauntNumber,
		TauntText:      event.TauntText,
		DestinationMap: event.DestinationMap,
		MessageAGP:     event.MessageAGP,
		SourceOffset:   event.SourceOffset,
		Confidence:     event.Confidence,
	}
}

func chatDedupeKey(line ChatLine) string {
	player := normalizeChatKey(line.PlayerName)
	if player == "" && line.PlayerID > 0 {
		player = "p" + strconv.Itoa(line.PlayerID)
	}
	return player + "\x00" + normalizeChatKey(line.Text)
}

func normalizeChatKey(s string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(s))), " ")
}

func plausibleChatText(s string) bool {
	if s == "" || len(s) > 512 || !utf8.ValidString(s) {
		return false
	}
	printable := 0
	for _, r := range s {
		if r == '\t' {
			continue
		}
		if unicode.IsPrint(r) {
			printable++
			continue
		}
		return false
	}
	return printable > 0
}
