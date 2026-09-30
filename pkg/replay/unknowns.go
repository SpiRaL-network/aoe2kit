package replay

import (
	"encoding/hex"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
)

type UnknownsReport struct {
	Root       string                    `json:"root"`
	Replays    int                       `json:"replays"`
	ActionIDs  []UnknownActionSummary    `json:"action_ids,omitempty"`
	Operations []UnknownOperationSummary `json:"operations,omitempty"`
	Warnings   []string                  `json:"warnings,omitempty"`
}

type UnknownActionSummary struct {
	ActionID     int      `json:"action_id"`
	Count        int      `json:"count"`
	PayloadBytes []int    `json:"payload_bytes,omitempty"`
	SampleHex    string   `json:"sample_hex,omitempty"`
	ReplayPaths  []string `json:"replay_paths,omitempty"`
}

type UnknownOperationSummary struct {
	OperationID int      `json:"operation_id"`
	Count       int      `json:"count"`
	ReplayPaths []string `json:"replay_paths,omitempty"`
}

func BuildUnknowns(root string) (*UnknownsReport, error) {
	report := &UnknownsReport{Root: root}
	actions := map[int]*UnknownActionSummary{}
	ops := map[int]*UnknownOperationSummary{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			report.Warnings = append(report.Warnings, fmt.Sprintf("walk %s: %v", path, err))
			return nil
		}
		if d.IsDir() || !isReplayPath(path) {
			return nil
		}
		report.Replays++
		events, runErr := ExtractEvents(path, EventOptions{IncludeUntypedAction: true, IncludeRaw: true})
		if runErr != nil {
			report.Warnings = append(report.Warnings, fmt.Sprintf("%s: %v", path, runErr))
			return nil
		}
		for _, event := range events.Events {
			switch event.Type {
			case "action":
				item := actions[event.ActionID]
				if item == nil {
					item = &UnknownActionSummary{ActionID: event.ActionID}
					actions[event.ActionID] = item
				}
				item.Count++
				item.PayloadBytes = appendUniqueInt(item.PayloadBytes, event.PayloadBytes)
				item.ReplayPaths = appendUniqueString(item.ReplayPaths, path)
				if item.SampleHex == "" {
					item.SampleHex = event.RawHex
				}
			case "unknown_operation":
				item := ops[event.OperationID]
				if item == nil {
					item = &UnknownOperationSummary{OperationID: event.OperationID}
					ops[event.OperationID] = item
				}
				item.Count++
				item.ReplayPaths = appendUniqueString(item.ReplayPaths, path)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	for _, item := range actions {
		sort.Ints(item.PayloadBytes)
		sort.Strings(item.ReplayPaths)
		report.ActionIDs = append(report.ActionIDs, *item)
	}
	sort.Slice(report.ActionIDs, func(i, j int) bool {
		if report.ActionIDs[i].Count != report.ActionIDs[j].Count {
			return report.ActionIDs[i].Count > report.ActionIDs[j].Count
		}
		return report.ActionIDs[i].ActionID < report.ActionIDs[j].ActionID
	})
	for _, item := range ops {
		sort.Strings(item.ReplayPaths)
		report.Operations = append(report.Operations, *item)
	}
	sort.Slice(report.Operations, func(i, j int) bool {
		if report.Operations[i].Count != report.Operations[j].Count {
			return report.Operations[i].Count > report.Operations[j].Count
		}
		return report.Operations[i].OperationID < report.Operations[j].OperationID
	})
	return report, nil
}

func appendUniqueInt(values []int, value int) []int {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func appendUniqueString(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func sampleHex(data []byte) string {
	if len(data) > 64 {
		data = data[:64]
	}
	return hex.EncodeToString(data)
}
