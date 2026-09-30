package campaign

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"aoe2kit/pkg/scenario"
)

type ExtractOptions struct {
	OutDir   string
	Scenario int
	Name     string
}

type ExtractReport struct {
	OK             bool           `json:"ok"`
	Path           string         `json:"path"`
	Version        string         `json:"version"`
	DirectoryAt    int            `json:"directory_at"`
	DirectoryCount int            `json:"directory_count"`
	OutDir         string         `json:"out_dir"`
	Entries        []ExtractEntry `json:"entries"`
	Warnings       []string       `json:"warnings,omitempty"`
}

type ExtractEntry struct {
	Index       int    `json:"index"`
	Name        string `json:"name"`
	Filename    string `json:"filename"`
	Offset      int    `json:"offset"`
	Size        int    `json:"size"`
	OutPath     string `json:"out_path,omitempty"`
	SHA256      string `json:"sha256,omitempty"`
	Selected    bool   `json:"selected"`
	FullParseOK bool   `json:"full_parse_ok"`
	ParseError  string `json:"parse_error,omitempty"`
}

type campaignDirectoryEntry struct {
	name     string
	filename string
	offset   int
	size     int
}

func ExtractFile(path string, opts ExtractOptions) (*ExtractReport, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	report, err := Extract(data, path, opts)
	if err != nil {
		return report, err
	}
	return report, nil
}

func Extract(data []byte, path string, opts ExtractOptions) (*ExtractReport, error) {
	report := &ExtractReport{OK: true, Path: path}
	if len(data) < 8 {
		report.OK = false
		return report, fmt.Errorf("campaign file too short")
	}
	report.Version = strings.TrimRight(string(data[:4]), "\x00")
	if report.Version == "" {
		report.OK = false
		return report, fmt.Errorf("campaign has empty version")
	}
	if opts.Scenario < 0 {
		report.OK = false
		return report, fmt.Errorf("--scenario must be 1-based")
	}
	if opts.OutDir == "" {
		stem := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		if stem == "" || stem == "." {
			stem = "campaign"
		}
		opts.OutDir = stem + "-extract"
	}
	tableAt, entries, err := findCampaignDirectory(data)
	if err != nil {
		report.OK = false
		return report, err
	}
	report.DirectoryAt = tableAt
	report.DirectoryCount = len(entries)
	report.OutDir = opts.OutDir
	if err := os.MkdirAll(opts.OutDir, 0o755); err != nil {
		return nil, err
	}
	nameFilter := strings.TrimSpace(opts.Name)
	for i, entry := range entries {
		selected := extractEntrySelected(i+1, entry, opts.Scenario, nameFilter)
		row := ExtractEntry{
			Index:    i + 1,
			Name:     entry.name,
			Filename: entry.filename,
			Offset:   entry.offset,
			Size:     entry.size,
			Selected: selected,
		}
		if selected {
			blob := data[entry.offset : entry.offset+entry.size]
			name := safeScenarioFilename(entry.filename, i+1)
			outPath := filepath.Join(opts.OutDir, name)
			if err := os.WriteFile(outPath, blob, 0o644); err != nil {
				report.OK = false
				return report, err
			}
			sum := sha256.Sum256(blob)
			row.OutPath = outPath
			row.SHA256 = hex.EncodeToString(sum[:])
			if _, err := scenario.Open(outPath); err != nil {
				row.ParseError = err.Error()
				report.Warnings = append(report.Warnings, fmt.Sprintf("%s extracted but full scenario parse failed: %v", name, err))
			} else {
				row.FullParseOK = true
			}
		}
		report.Entries = append(report.Entries, row)
	}
	if opts.Scenario != 0 || nameFilter != "" {
		found := false
		for _, entry := range report.Entries {
			if entry.Selected {
				found = true
				break
			}
		}
		if !found {
			report.OK = false
			return report, fmt.Errorf("no campaign scenario matched selection")
		}
	}
	return report, nil
}

func extractEntrySelected(index int, entry campaignDirectoryEntry, scenarioIndex int, nameFilter string) bool {
	if scenarioIndex != 0 {
		return index == scenarioIndex
	}
	if nameFilter == "" {
		return true
	}
	filter := strings.ToLower(strings.TrimSuffix(nameFilter, filepath.Ext(nameFilter)))
	name := strings.ToLower(strings.TrimSuffix(entry.name, filepath.Ext(entry.name)))
	filename := strings.ToLower(strings.TrimSuffix(entry.filename, filepath.Ext(entry.filename)))
	return filter == name || filter == filename || strings.Contains(name, filter) || strings.Contains(filename, filter)
}

func findCampaignDirectory(data []byte) (int, []campaignDirectoryEntry, error) {
	var bestAt int
	var best []campaignDirectoryEntry
	for pos := 8; pos+4 < len(data); pos += 4 {
		maxCount := int(binary.LittleEndian.Uint32(data[pos:]))
		if maxCount < 1 || maxCount > 64 {
			continue
		}
		for count := maxCount; count >= 1; count-- {
			entries, end, ok := parseCampaignDirectoryAt(data, pos+4, count)
			if !ok {
				continue
			}
			if len(entries) > len(best) || (len(entries) == len(best) && end <= firstScenarioOffset(entries)) {
				bestAt = pos
				best = entries
			}
			break
		}
	}
	if len(best) == 0 {
		return 0, nil, fmt.Errorf("no valid scenario directory found in campaign container")
	}
	return bestAt, best, nil
}

func parseCampaignDirectoryAt(data []byte, pos, count int) ([]campaignDirectoryEntry, int, bool) {
	entries := make([]campaignDirectoryEntry, 0, count)
	cursor := pos
	for i := 0; i < count; i++ {
		if cursor+8 > len(data) {
			return nil, 0, false
		}
		size := int(binary.LittleEndian.Uint32(data[cursor:]))
		offset := int(binary.LittleEndian.Uint32(data[cursor+4:]))
		cursor += 8
		name, next, ok := readCampaignString(data, cursor)
		if !ok {
			return nil, 0, false
		}
		cursor = next
		filename, next, ok := readCampaignString(data, cursor)
		if !ok {
			return nil, 0, false
		}
		cursor = next
		if size <= 0 || offset <= cursor || offset+size > len(data) {
			return nil, 0, false
		}
		if !strings.Contains(strings.ToLower(filename), ".aoe2scenario") {
			return nil, 0, false
		}
		entries = append(entries, campaignDirectoryEntry{name: name, filename: filename, offset: offset, size: size})
	}
	if cursor > firstScenarioOffset(entries) {
		return nil, 0, false
	}
	for i := 1; i < len(entries); i++ {
		if entries[i-1].offset+entries[i-1].size > entries[i].offset {
			return nil, 0, false
		}
	}
	return entries, cursor, true
}

func readCampaignString(data []byte, pos int) (string, int, bool) {
	if pos+4 > len(data) {
		return "", pos, false
	}
	marker := binary.LittleEndian.Uint16(data[pos:])
	if marker != 0x0a60 {
		return "", pos, false
	}
	n := int(binary.LittleEndian.Uint16(data[pos+2:]))
	if n <= 0 || n > 512 || pos+4+n > len(data) {
		return "", pos, false
	}
	return string(data[pos+4 : pos+4+n]), pos + 4 + n, true
}

func firstScenarioOffset(entries []campaignDirectoryEntry) int {
	if len(entries) == 0 {
		return 0
	}
	min := entries[0].offset
	for _, entry := range entries[1:] {
		if entry.offset < min {
			min = entry.offset
		}
	}
	return min
}

func safeScenarioFilename(name string, index int) string {
	name = filepath.Base(strings.TrimSpace(name))
	if name == "" || name == "." || name == string(filepath.Separator) {
		name = fmt.Sprintf("scenario_%02d.aoe2scenario", index)
	}
	if filepath.Ext(strings.ToLower(name)) != ".aoe2scenario" {
		name += ".aoe2scenario"
	}
	return name
}
