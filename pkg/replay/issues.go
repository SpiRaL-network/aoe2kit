package replay

import (
	"fmt"
	"sort"
	"strings"
)

type IssuesOptions struct {
	ContextPath string
}

type IssuesReport struct {
	Path     string       `json:"path"`
	Cards    []IssueCard  `json:"cards,omitempty"`
	Groups   []IssueGroup `json:"groups"`
	Warnings []string     `json:"warnings,omitempty"`
}

type IssueGroup struct {
	Key        string      `json:"key"`
	Region     string      `json:"region,omitempty"`
	PlayerID   int         `json:"player_id,omitempty"`
	PlayerName string      `json:"player_name,omitempty"`
	Count      int         `json:"count"`
	Items      []IssueItem `json:"items"`
}

type IssueItem struct {
	Time string  `json:"time,omitempty"`
	Type string  `json:"type"`
	Text string  `json:"text"`
	X    float64 `json:"x,omitempty"`
	Y    float64 `json:"y,omitempty"`
}

type IssueCard struct {
	ID          string          `json:"id"`
	Key         string          `json:"key"`
	Replay      string          `json:"replay,omitempty"`
	Scenario    StoryIdentity   `json:"scenario,omitempty"`
	Chapter     string          `json:"chapter,omitempty"`
	Region      string          `json:"region,omitempty"`
	PlayerID    int             `json:"player_id,omitempty"`
	PlayerName  string          `json:"player_name,omitempty"`
	Type        string          `json:"type"`
	Text        string          `json:"text"`
	Count       int             `json:"count"`
	FirstTime   string          `json:"first_time,omitempty"`
	LastTime    string          `json:"last_time,omitempty"`
	FirstTimeMS int             `json:"first_time_ms,omitempty"`
	LastTimeMS  int             `json:"last_time_ms,omitempty"`
	X           float64         `json:"x,omitempty"`
	Y           float64         `json:"y,omitempty"`
	Source      string          `json:"source"`
	Confidence  string          `json:"confidence"`
	Evidence    []IssueEvidence `json:"evidence,omitempty"`
}

type IssueEvidence struct {
	Replay     string  `json:"replay,omitempty"`
	Time       string  `json:"time,omitempty"`
	TimeMS     int     `json:"time_ms,omitempty"`
	PlayerID   int     `json:"player_id,omitempty"`
	PlayerName string  `json:"player_name,omitempty"`
	Region     string  `json:"region,omitempty"`
	Type       string  `json:"type"`
	Text       string  `json:"text"`
	X          float64 `json:"x,omitempty"`
	Y          float64 `json:"y,omitempty"`
	Chapter    string  `json:"chapter,omitempty"`
}

func BuildIssues(path string, opts IssuesOptions) (*IssuesReport, error) {
	context, err := LoadContext(opts.ContextPath)
	if err != nil {
		return nil, err
	}
	eventReport, err := ExtractEvents(path, EventOptions{TelemetryPrefixes: context.TelemetryPrefixes})
	if err != nil {
		return nil, err
	}
	AnnotateRegions(eventReport.Events, context)
	cache, err := OpenReplayCache(path)
	if err != nil {
		return nil, err
	}
	rec, err := cache.Metadata()
	if err != nil {
		return nil, err
	}
	report := &IssuesReport{
		Path:     path,
		Cards:    BuildIssueCards(path, storyIdentity(rec), eventReport.Events, eventReport.Counts.DurationMS),
		Warnings: append([]string{}, eventReport.Warnings...),
	}
	report.Groups = buildLegacyIssueGroups(feedbackStoryLines(eventReport.Events))
	return report, nil
}

func buildLegacyIssueGroups(lines []StoryLine) []IssueGroup {
	byKey := map[string]*IssueGroup{}
	for _, line := range lines {
		key := "unknown"
		if line.Region != "" {
			key = "region:" + line.Region
		} else if line.PlayerID > 0 {
			key = fmt.Sprintf("player:%d", line.PlayerID)
		}
		group := byKey[key]
		if group == nil {
			group = &IssueGroup{Key: key, Region: line.Region, PlayerID: line.PlayerID, PlayerName: line.PlayerName}
			byKey[key] = group
		}
		group.Count++
		group.Items = append(group.Items, IssueItem{Time: line.Time, Type: line.Type, Text: line.Text})
	}
	keys := make([]string, 0, len(byKey))
	for key := range byKey {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		a, b := byKey[keys[i]], byKey[keys[j]]
		if a.Count != b.Count {
			return a.Count > b.Count
		}
		return a.Key < b.Key
	})
	out := make([]IssueGroup, 0, len(keys))
	for _, key := range keys {
		out = append(out, *byKey[key])
	}
	return out
}

func BuildIssueCards(replayPath string, identity StoryIdentity, events []ReplayEvent, durationMS int) []IssueCard {
	byKey := map[string]*IssueCard{}
	for _, event := range events {
		if !isLiveFeedbackEvent(event) {
			continue
		}
		text := issueText(event)
		key := issueKey(event, durationMS, text)
		card := byKey[key]
		if card == nil {
			card = &IssueCard{
				ID:         fmt.Sprintf("issue-%04d", len(byKey)+1),
				Key:        key,
				Replay:     replayPath,
				Scenario:   identity,
				Chapter:    chapterNameForTime(event.TimeMS, durationMS),
				Region:     event.Region,
				PlayerID:   event.PlayerID,
				PlayerName: event.PlayerName,
				Type:       event.Type,
				Text:       text,
				Source:     event.Source,
				Confidence: issueConfidence(event),
			}
			byKey[key] = card
		}
		card.Count++
		if card.FirstTime == "" || event.TimeMS < card.FirstTimeMS {
			card.FirstTime = event.Time
			card.FirstTimeMS = event.TimeMS
			card.X = event.X
			card.Y = event.Y
		}
		if card.LastTime == "" || event.TimeMS >= card.LastTimeMS {
			card.LastTime = event.Time
			card.LastTimeMS = event.TimeMS
		}
		card.Evidence = append(card.Evidence, IssueEvidence{
			Replay:     replayPath,
			Time:       event.Time,
			TimeMS:     event.TimeMS,
			PlayerID:   event.PlayerID,
			PlayerName: event.PlayerName,
			Region:     event.Region,
			Type:       event.Type,
			Text:       text,
			X:          event.X,
			Y:          event.Y,
			Chapter:    chapterNameForTime(event.TimeMS, durationMS),
		})
	}
	cards := make([]IssueCard, 0, len(byKey))
	for _, card := range byKey {
		cards = append(cards, *card)
	}
	sortIssueCards(cards)
	for i := range cards {
		cards[i].ID = fmt.Sprintf("issue-%04d", i+1)
	}
	return cards
}

func BuildIssueCardsFromStoryLines(replayPath string, identity StoryIdentity, lines []StoryLine, durationMS int) []IssueCard {
	byKey := map[string]*IssueCard{}
	for _, line := range lines {
		eventType := line.Type
		if eventType != "chat" && eventType != "flare" {
			continue
		}
		text := line.Text
		key := strings.Join([]string{eventType, chapterNameForTime(line.TimeMS, durationMS), issueLineAnchor(line), normalizeIssueText(text)}, "|")
		card := byKey[key]
		if card == nil {
			card = &IssueCard{
				ID:         fmt.Sprintf("issue-%04d", len(byKey)+1),
				Key:        key,
				Replay:     replayPath,
				Scenario:   identity,
				Chapter:    chapterNameForTime(line.TimeMS, durationMS),
				Region:     line.Region,
				PlayerID:   line.PlayerID,
				PlayerName: line.PlayerName,
				Type:       eventType,
				Text:       text,
				Source:     "replay_story.feedback",
				Confidence: "replay_visible",
			}
			if eventType == "flare" {
				card.Confidence = "replay_action_verified"
			}
			byKey[key] = card
		}
		card.Count++
		if card.FirstTime == "" || line.TimeMS < card.FirstTimeMS {
			card.FirstTime = line.Time
			card.FirstTimeMS = line.TimeMS
		}
		if card.LastTime == "" || line.TimeMS >= card.LastTimeMS {
			card.LastTime = line.Time
			card.LastTimeMS = line.TimeMS
		}
		card.Evidence = append(card.Evidence, IssueEvidence{
			Replay:     replayPath,
			Time:       line.Time,
			TimeMS:     line.TimeMS,
			PlayerID:   line.PlayerID,
			PlayerName: line.PlayerName,
			Region:     line.Region,
			Type:       eventType,
			Text:       text,
			Chapter:    chapterNameForTime(line.TimeMS, durationMS),
		})
	}
	cards := make([]IssueCard, 0, len(byKey))
	for _, card := range byKey {
		cards = append(cards, *card)
	}
	sortIssueCards(cards)
	for i := range cards {
		cards[i].ID = fmt.Sprintf("issue-%04d", i+1)
	}
	return cards
}

func issueLineAnchor(line StoryLine) string {
	if line.Region != "" {
		return line.Region
	}
	if line.PlayerID > 0 {
		return fmt.Sprintf("p%d", line.PlayerID)
	}
	return "unknown"
}

func sortIssueCards(cards []IssueCard) {
	sort.Slice(cards, func(i, j int) bool {
		if cards[i].Count != cards[j].Count {
			return cards[i].Count > cards[j].Count
		}
		if cards[i].FirstTimeMS != cards[j].FirstTimeMS {
			return cards[i].FirstTimeMS < cards[j].FirstTimeMS
		}
		return cards[i].Key < cards[j].Key
	})
}

func issueText(event ReplayEvent) string {
	if event.Type == "flare" {
		if event.Region != "" {
			return "flare in " + event.Region
		}
		return fmt.Sprintf("flare x=%.2f y=%.2f", event.X, event.Y)
	}
	if event.Telemetry != nil {
		return event.Telemetry.Raw
	}
	if event.TauntNumber > 0 {
		return fmt.Sprintf("taunt %d: %s", event.TauntNumber, event.Text)
	}
	return event.Text
}

func issueKey(event ReplayEvent, durationMS int, text string) string {
	anchor := event.Region
	if anchor == "" && event.PlayerID > 0 {
		anchor = fmt.Sprintf("p%d", event.PlayerID)
	}
	if anchor == "" {
		anchor = "unknown"
	}
	return strings.Join([]string{
		event.Type,
		chapterNameForTime(event.TimeMS, durationMS),
		anchor,
		normalizeIssueText(text),
	}, "|")
}

func normalizeIssueText(text string) string {
	text = strings.ToLower(strings.TrimSpace(text))
	text = strings.Join(strings.Fields(text), " ")
	if len(text) > 80 {
		text = text[:80]
	}
	if text == "" {
		return "(empty)"
	}
	return text
}

func issueConfidence(event ReplayEvent) string {
	if event.Type == "flare" {
		return "replay_action_verified"
	}
	if event.Telemetry != nil {
		return "scenario_declared"
	}
	return "replay_visible"
}
