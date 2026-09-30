package replay

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
)

type InboxOptions struct {
	ContextPath string
}

type InboxReport struct {
	Root           string               `json:"root"`
	Count          int                  `json:"count"`
	Summary        InboxSummary         `json:"summary"`
	ScenarioGroups []InboxScenarioGroup `json:"scenario_groups,omitempty"`
	IssueCards     []IssueCard          `json:"issue_cards,omitempty"`
	Duplicates     []Duplicate          `json:"duplicates,omitempty"`
	Items          []InboxItem          `json:"items"`
	Warnings       []string             `json:"warnings,omitempty"`
}

type InboxItem struct {
	Path            string           `json:"path"`
	RecordSHA256    string           `json:"record_sha256,omitempty"`
	DuplicateOf     string           `json:"duplicate_of,omitempty"`
	Identity        StoryIdentity    `json:"identity"`
	DataSet         DataSetIdentity  `json:"data_set_identity"`
	Players         []FeedbackPlayer `json:"players,omitempty"`
	EventCounts     EventCounts      `json:"event_counts"`
	ProfileCoverage ProfileCoverage  `json:"profile_coverage,omitempty"`
	Result          MatchResult      `json:"result"`
	Moments         []StoryMoment    `json:"moments,omitempty"`
	Brief           []string         `json:"brief,omitempty"`
	IssueCards      []IssueCard      `json:"issue_cards,omitempty"`
	Warnings        []string         `json:"warnings,omitempty"`
	Error           string           `json:"error,omitempty"`
}

type Duplicate struct {
	SHA256 string   `json:"sha256"`
	Paths  []string `json:"paths"`
}

type InboxSummary struct {
	Parsed                int     `json:"parsed"`
	Errors                int     `json:"errors"`
	DuplicateItems        int     `json:"duplicate_items"`
	ScenarioGroups        int     `json:"scenario_groups"`
	RecognizedScenarios   int     `json:"recognized_scenarios"`
	VanillaDataSets       int     `json:"vanilla_data_sets"`
	ModdedDataSets        int     `json:"modded_data_sets"`
	UnknownDataSets       int     `json:"unknown_data_sets"`
	Completed             int     `json:"completed"`
	WinnerKnown           int     `json:"winner_known"`
	FeedbackEvents        int     `json:"feedback_events"`
	IssueCards            int     `json:"issue_cards"`
	ReplayActions         int     `json:"replay_actions"`
	UntypedActions        int     `json:"untyped_actions"`
	AverageDecodedPercent float64 `json:"average_decoded_percent,omitempty"`
}

type InboxScenarioGroup struct {
	Key            string         `json:"key"`
	Tier           string         `json:"tier,omitempty"`
	SHA256         string         `json:"sha256,omitempty"`
	Count          int            `json:"count"`
	Paths          []string       `json:"paths,omitempty"`
	DataSets       map[string]int `json:"data_sets,omitempty"`
	Results        map[string]int `json:"results,omitempty"`
	FeedbackEvents int            `json:"feedback_events"`
	IssueCards     int            `json:"issue_cards"`
	Warnings       []string       `json:"warnings,omitempty"`
}

func BuildInbox(root string, opts InboxOptions) (*InboxReport, error) {
	report := &InboxReport{Root: root}
	var items []InboxItem
	hashPaths := map[string][]string{}
	var allIssueCards []IssueCard
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			report.Warnings = append(report.Warnings, fmt.Sprintf("walk %s: %v", path, err))
			return nil
		}
		if d.IsDir() || !isReplayPath(path) {
			return nil
		}
		item := InboxItem{Path: path}
		data, readErr := ReadRecordBytes(path)
		if readErr != nil {
			item.Error = readErr.Error()
			items = append(items, item)
			return nil
		}
		sum := sha256.Sum256(data)
		item.RecordSHA256 = hex.EncodeToString(sum[:])
		hashPaths[item.RecordSHA256] = append(hashPaths[item.RecordSHA256], path)
		story, storyErr := BuildStory(path, StoryOptions{ContextPath: opts.ContextPath})
		if storyErr != nil {
			item.Error = storyErr.Error()
		} else {
			item.Identity = story.Identity
			item.DataSet = story.DataSet
			item.Players = story.Players
			item.EventCounts = story.EventCounts
			item.Result = story.Result
			item.Moments = story.Moments
			item.Brief = story.Brief
			item.Warnings = story.Warnings
			item.ProfileCoverage = profileCoverageFromCounts(story.EventCounts)
			item.IssueCards = BuildIssueCardsFromStoryLines(path, story.Identity, story.Feedback, story.EventCounts.DurationMS)
			allIssueCards = append(allIssueCards, item.IssueCards...)
		}
		items = append(items, item)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Path < items[j].Path })
	report.Items = items
	report.Count = len(items)
	for hash, paths := range hashPaths {
		if len(paths) < 2 {
			continue
		}
		sort.Strings(paths)
		report.Duplicates = append(report.Duplicates, Duplicate{SHA256: hash, Paths: paths})
		for i := range report.Items {
			if report.Items[i].RecordSHA256 == hash && report.Items[i].Path != paths[0] {
				report.Items[i].DuplicateOf = paths[0]
			}
		}
	}
	sort.Slice(report.Duplicates, func(i, j int) bool { return report.Duplicates[i].SHA256 < report.Duplicates[j].SHA256 })
	report.IssueCards = mergeIssueCards(allIssueCards)
	report.ScenarioGroups = buildInboxScenarioGroups(report.Items)
	report.Summary = summarizeInbox(report)
	return report, nil
}

func profileCoverageFromCounts(counts EventCounts) ProfileCoverage {
	coverage := ProfileCoverage{
		ReplayActions:    counts.Actions,
		UndecodedActions: counts.UntypedActions,
		ViewlockEvents:   counts.Viewlocks,
		FeedbackEvents:   counts.Chat + counts.Flares,
	}
	if counts.Actions >= counts.UntypedActions {
		coverage.DecodedActions = counts.Actions - counts.UntypedActions
	}
	if counts.Actions > 0 {
		coverage.DecodedPercent = round2(float64(coverage.DecodedActions) * 100 / float64(counts.Actions))
	}
	return coverage
}

func isReplayPath(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".aoe2record" || ext == ".zip"
}

func mergeIssueCards(cards []IssueCard) []IssueCard {
	byKey := map[string]*IssueCard{}
	for _, card := range cards {
		key := scenarioGroupKey(card.Scenario) + "|" + card.Key
		merged := byKey[key]
		if merged == nil {
			copied := card
			copied.ID = fmt.Sprintf("issue-%04d", len(byKey)+1)
			merged = &copied
			byKey[key] = merged
			continue
		}
		merged.Count += card.Count
		if merged.FirstTime == "" || card.FirstTimeMS < merged.FirstTimeMS {
			merged.FirstTime = card.FirstTime
			merged.FirstTimeMS = card.FirstTimeMS
			merged.X = card.X
			merged.Y = card.Y
			merged.Replay = card.Replay
		}
		if merged.LastTime == "" || card.LastTimeMS >= merged.LastTimeMS {
			merged.LastTime = card.LastTime
			merged.LastTimeMS = card.LastTimeMS
		}
		merged.Evidence = append(merged.Evidence, card.Evidence...)
	}
	out := make([]IssueCard, 0, len(byKey))
	for _, card := range byKey {
		out = append(out, *card)
	}
	sortIssueCards(out)
	for i := range out {
		out[i].ID = fmt.Sprintf("issue-%04d", i+1)
	}
	return out
}

func buildInboxScenarioGroups(items []InboxItem) []InboxScenarioGroup {
	byKey := map[string]*InboxScenarioGroup{}
	for _, item := range items {
		if item.Error != "" {
			continue
		}
		key := scenarioGroupKey(item.Identity)
		group := byKey[key]
		if group == nil {
			group = &InboxScenarioGroup{
				Key:      key,
				Tier:     item.Identity.Tier,
				SHA256:   item.Identity.SHA256,
				DataSets: map[string]int{},
				Results:  map[string]int{},
			}
			byKey[key] = group
		}
		group.Count++
		group.Paths = append(group.Paths, item.Path)
		group.DataSets[formatDataSetKey(item.DataSet)]++
		group.Results[formatResultKey(item.Result)]++
		group.FeedbackEvents += item.EventCounts.Chat + item.EventCounts.Flares
		group.IssueCards += len(item.IssueCards)
		group.Warnings = append(group.Warnings, item.Warnings...)
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
	out := make([]InboxScenarioGroup, 0, len(keys))
	for _, key := range keys {
		group := *byKey[key]
		sort.Strings(group.Paths)
		group.Warnings = uniqueStrings(group.Warnings)
		out = append(out, group)
	}
	return out
}

func summarizeInbox(report *InboxReport) InboxSummary {
	var summary InboxSummary
	summary.ScenarioGroups = len(report.ScenarioGroups)
	summary.IssueCards = len(report.IssueCards)
	decodedSamples := 0
	for _, duplicate := range report.Duplicates {
		if len(duplicate.Paths) > 1 {
			summary.DuplicateItems += len(duplicate.Paths) - 1
		}
	}
	for _, item := range report.Items {
		if item.Error != "" {
			summary.Errors++
			continue
		}
		summary.Parsed++
		if item.Identity.SHA256 != "" {
			summary.RecognizedScenarios++
		}
		switch item.DataSet.Status {
		case "vanilla":
			summary.VanillaDataSets++
		case "modded":
			summary.ModdedDataSets++
		default:
			summary.UnknownDataSets++
		}
		if item.Result.Completed {
			summary.Completed++
		}
		if item.Result.WinnerKnown {
			summary.WinnerKnown++
		}
		summary.FeedbackEvents += item.EventCounts.Chat + item.EventCounts.Flares
		summary.ReplayActions += item.EventCounts.Actions
		summary.UntypedActions += item.EventCounts.UntypedActions
		if item.ProfileCoverage.ReplayActions > 0 {
			summary.AverageDecodedPercent += item.ProfileCoverage.DecodedPercent
			decodedSamples++
		}
	}
	if decodedSamples > 0 {
		summary.AverageDecodedPercent = round2(summary.AverageDecodedPercent / float64(decodedSamples))
	}
	return summary
}

func scenarioGroupKey(identity StoryIdentity) string {
	if identity.SHA256 != "" {
		if identity.Tier != "" {
			return identity.Tier + ":" + identity.SHA256
		}
		return identity.SHA256
	}
	return "unknown"
}

func formatDataSetKey(identity DataSetIdentity) string {
	if identity.Status == "modded" {
		suffix := ""
		if identity.Checksum != 0 {
			suffix += fmt.Sprintf(":checksum=%d", identity.Checksum)
		}
		if identity.WorkshopID != 0 {
			suffix += fmt.Sprintf(":workshop_id=%d", identity.WorkshopID)
		}
		if len(identity.ActiveDataSets) > 0 {
			names := append([]string{}, identity.ActiveDataSets...)
			sort.Strings(names)
			return "modded:" + strings.Join(names, ",") + suffix
		}
		if identity.ActiveDataSet != "" {
			return "modded:" + identity.ActiveDataSet + suffix
		}
		return "modded" + suffix
	}
	if identity.Status != "" {
		return identity.Status
	}
	return "unknown"
}

func formatResultKey(result MatchResult) string {
	if result.WinnerKnown {
		return fmt.Sprintf("winners=%v losers=%v", result.Winners, result.Losers)
	}
	if result.Completed {
		return "completed_result_unknown"
	}
	return "unknown"
}

func uniqueStrings(values []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, value := range values {
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}
