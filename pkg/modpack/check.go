package modpack

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"aoe2kit/pkg/aoe2"
)

type CheckOptions struct {
	StatusPath string
}

type CheckReport struct {
	Path            string                 `json:"path"`
	InfoJSON        bool                   `json:"info_json"`
	Info            map[string]any         `json:"info,omitempty"`
	Title           string                 `json:"title,omitempty"`
	Verification    aoe2.VerificationClaim `json:"verification"`
	ResourcesCommon bool                   `json:"resources_common"`
	DataFiles       []string               `json:"data_files,omitempty"`
	StringFiles     []string               `json:"string_files,omitempty"`
	AIFiles         []string               `json:"ai_files,omitempty"`
	XSFiles         []string               `json:"xs_files,omitempty"`
	DRSFiles        []string               `json:"drs_files,omitempty"`
	ScenarioFiles   []string               `json:"scenario_files,omitempty"`
	OtherKnownFiles []string               `json:"other_known_files,omitempty"`
	Activation      *ActivationReport      `json:"activation,omitempty"`
	Warnings        []string               `json:"warnings,omitempty"`
	Errors          []string               `json:"errors,omitempty"`
}

type ActivationReport struct {
	Checked        bool             `json:"checked"`
	StatusPath     string           `json:"status_path,omitempty"`
	ModTitle       string           `json:"mod_title,omitempty"`
	DataMod        bool             `json:"data_mod"`
	Matched        []ModStatusEntry `json:"matched,omitempty"`
	Enabled        *bool            `json:"enabled,omitempty"`
	Published      *bool            `json:"published,omitempty"`
	ActiveDataMods []string         `json:"active_data_mods,omitempty"`
	ActivationOK   bool             `json:"activation_ok"`
	Warnings       []string         `json:"warnings,omitempty"`
	Errors         []string         `json:"errors,omitempty"`
}

type ModStatusEntry struct {
	Title     string         `json:"title,omitempty"`
	ID        string         `json:"id,omitempty"`
	Path      string         `json:"path,omitempty"`
	Enabled   *bool          `json:"enabled,omitempty"`
	Published *bool          `json:"published,omitempty"`
	Type      string         `json:"type,omitempty"`
	RawKeys   map[string]any `json:"raw_keys,omitempty"`
}

func Check(path string) (CheckReport, error) {
	return CheckWithOptions(path, CheckOptions{})
}

func CheckWithOptions(path string, opts CheckOptions) (CheckReport, error) {
	st, err := os.Stat(path)
	if err != nil {
		return CheckReport{}, err
	}
	if !st.IsDir() {
		return CheckReport{}, fmt.Errorf("%s is not a directory", path)
	}

	report := CheckReport{Path: path}
	infoPath := filepath.Join(path, "info.json")
	if data, err := os.ReadFile(infoPath); err == nil {
		report.InfoJSON = true
		if err := json.Unmarshal(data, &report.Info); err != nil {
			report.Errors = append(report.Errors, fmt.Sprintf("info.json is not valid JSON: %v", err))
		} else if modTitle(report.Info) == "" {
			report.Warnings = append(report.Warnings, "info.json has no Title/name field")
		}
		report.Title = modTitle(report.Info)
	} else if os.IsNotExist(err) {
		report.Errors = append(report.Errors, "missing info.json")
	} else {
		report.Errors = append(report.Errors, fmt.Sprintf("cannot read info.json: %v", err))
	}

	common := filepath.Join(path, "resources", "_common")
	if st, err := os.Stat(common); err == nil && st.IsDir() {
		report.ResourcesCommon = true
	} else {
		report.Warnings = append(report.Warnings, "missing resources/_common")
	}

	if err := filepath.WalkDir(path, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			report.Warnings = append(report.Warnings, fmt.Sprintf("walk %s: %v", p, err))
			return nil
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(path, p)
		if err != nil {
			rel = p
		}
		rel = filepath.ToSlash(rel)
		low := strings.ToLower(rel)
		if st, statErr := d.Info(); statErr == nil && st.Size() == 0 {
			report.Warnings = append(report.Warnings, fmt.Sprintf("%s is empty", rel))
		}
		switch {
		case strings.HasSuffix(low, ".dat"):
			report.DataFiles = append(report.DataFiles, rel)
			if !strings.HasPrefix(low, "resources/_common/dat/") {
				report.Warnings = append(report.Warnings, fmt.Sprintf("%s is a dat file outside resources/_common/dat", rel))
			}
		case strings.Contains(low, "/strings/") || strings.HasSuffix(low, "key-value-strings-utf8.txt") || strings.HasSuffix(low, "strings.txt"):
			report.StringFiles = append(report.StringFiles, rel)
			checkStringFile(path, rel, &report)
		case strings.HasSuffix(low, ".ai") || strings.HasSuffix(low, ".per") || strings.HasSuffix(low, ".per2"):
			report.AIFiles = append(report.AIFiles, rel)
			if !strings.Contains(low, "/ai/") {
				report.Warnings = append(report.Warnings, fmt.Sprintf("%s is an AI file outside an ai folder", rel))
			}
		case strings.HasSuffix(low, ".xs"):
			report.XSFiles = append(report.XSFiles, rel)
		case strings.HasSuffix(low, ".aoe2scenario"):
			report.ScenarioFiles = append(report.ScenarioFiles, rel)
			if !strings.Contains(low, "/scenario/") {
				report.Warnings = append(report.Warnings, fmt.Sprintf("%s is a scenario outside a scenario folder", rel))
			}
		case strings.Contains(low, "/drs/"):
			report.DRSFiles = append(report.DRSFiles, rel)
		case strings.HasPrefix(low, "resources/"):
			report.OtherKnownFiles = append(report.OtherKnownFiles, rel)
		}
		return nil
	}); err != nil {
		return CheckReport{}, err
	}

	sort.Strings(report.DataFiles)
	sort.Strings(report.StringFiles)
	sort.Strings(report.AIFiles)
	sort.Strings(report.XSFiles)
	sort.Strings(report.DRSFiles)
	sort.Strings(report.ScenarioFiles)
	sort.Strings(report.OtherKnownFiles)
	checkAIPairs(&report)
	structureOK := len(report.Errors) == 0
	activation := CheckActivation(path, report.Title, len(report.DataFiles) > 0, opts.StatusPath)
	report.Activation = &activation
	report.Warnings = append(report.Warnings, prefixedMessages("activation: ", activation.Warnings)...)
	report.Errors = append(report.Errors, prefixedMessages("activation: ", activation.Errors)...)
	report.Verification = aoe2.StructureVerification(structureOK)
	if structureOK && len(activation.Errors) > 0 {
		report.Verification = aoe2.StructureOKActivationFailed()
	}
	return report, nil
}

func (r CheckReport) OK() bool {
	return len(r.Errors) == 0
}

func CheckActivation(modPath, title string, dataMod bool, statusPath string) ActivationReport {
	report := ActivationReport{Checked: true, ModTitle: title, DataMod: dataMod}
	if statusPath == "" {
		statusPath = findModStatus(modPath)
	}
	if statusPath == "" {
		if dataMod {
			report.Warnings = append(report.Warnings, "no mod-status.json found; cannot prove this local data mod is enabled/published/selectable")
		} else {
			report.Warnings = append(report.Warnings, "no mod-status.json found; activation state not checked")
		}
		report.ActivationOK = !dataMod
		return report
	}
	report.StatusPath = statusPath
	data, err := os.ReadFile(statusPath)
	if err != nil {
		report.Errors = append(report.Errors, fmt.Sprintf("cannot read %s: %v", statusPath, err))
		return report
	}
	var root any
	if err := json.Unmarshal(data, &root); err != nil {
		report.Errors = append(report.Errors, fmt.Sprintf("%s is not valid JSON: %v", statusPath, err))
		return report
	}
	entries := extractModStatusEntries(root)
	report.Matched = matchModStatusEntries(entries, title, modPath)
	activeData := activeDataMods(entries)
	report.ActiveDataMods = activeData
	if len(report.Matched) == 0 {
		report.Warnings = append(report.Warnings, "mod not found in mod-status.json")
	} else {
		enabled := anyBool(report.Matched, func(e ModStatusEntry) *bool { return e.Enabled })
		published := anyBool(report.Matched, func(e ModStatusEntry) *bool { return e.Published })
		report.Enabled = enabled
		report.Published = published
		if dataMod {
			if enabled == nil {
				report.Warnings = append(report.Warnings, "matched data mod but Enabled field was not found")
			} else if !*enabled {
				report.Errors = append(report.Errors, "data mod is present in mod-status.json with Enabled=false; AoE2DE will silently use vanilla unless the data set is activated")
			}
			if published == nil {
				report.Warnings = append(report.Warnings, "matched data mod but publish/private-public state was not found")
			} else if !*published {
				report.Warnings = append(report.Warnings, "data mod appears unpublished; local data mods may not become selectable data sets until published, private is fine")
			}
		}
	}
	if dataMod && len(activeData) > 1 {
		report.Warnings = append(report.Warnings, fmt.Sprintf("%d enabled data mods detected; data mods can fight over empires2_x2_p1.dat", len(activeData)))
	}
	report.ActivationOK = len(report.Errors) == 0
	return report
}

func findModStatus(modPath string) string {
	candidates := []string{}
	if env := os.Getenv("AOE2KIT_MOD_STATUS"); env != "" {
		candidates = append(candidates, env)
	}
	for dir := modPath; dir != "" && dir != string(filepath.Separator); dir = filepath.Dir(dir) {
		candidates = append(candidates, filepath.Join(dir, "mod-status.json"))
		if filepath.Base(dir) == "mods" {
			break
		}
	}
	if home, err := os.UserHomeDir(); err == nil {
		candidates = append(candidates,
			filepath.Join(home, "Games", "Age of Empires 2 DE", "mods", "mod-status.json"),
			filepath.Join(home, ".local", "share", "Steam", "steamapps", "compatdata", "813780", "pfx", "drive_c", "users", "steamuser", "Games", "Age of Empires 2 DE", "mods", "mod-status.json"),
		)
	}
	seen := map[string]bool{}
	for _, path := range candidates {
		if path == "" || seen[path] {
			continue
		}
		seen[path] = true
		if st, err := os.Stat(path); err == nil && !st.IsDir() {
			return path
		}
	}
	return ""
}

func extractModStatusEntries(root any) []ModStatusEntry {
	var entries []ModStatusEntry
	var walk func(any)
	walk = func(value any) {
		switch v := value.(type) {
		case []any:
			for _, item := range v {
				walk(item)
			}
		case map[string]any:
			if entry, ok := modStatusEntryFromMap(v); ok {
				entries = append(entries, entry)
			}
			for _, item := range v {
				walk(item)
			}
		}
	}
	walk(root)
	return entries
}

func modStatusEntryFromMap(m map[string]any) (ModStatusEntry, bool) {
	title := firstString(m, "Title", "title", "Name", "name", "DisplayName", "displayName")
	id := firstString(m, "Id", "ID", "id", "ModId", "modId", "WorkshopId", "workshopId")
	path := firstString(m, "Path", "path", "Folder", "folder", "Directory", "directory")
	enabled := firstBool(m, "Enabled", "enabled", "IsEnabled", "isEnabled", "Subscribed", "subscribed")
	published := firstBool(m, "Published", "published", "IsPublished", "isPublished", "Publish", "publish", "Uploaded", "uploaded")
	modType := firstString(m, "Type", "type", "ModType", "modType", "Category", "category")
	if title == "" && id == "" && path == "" && enabled == nil && published == nil {
		return ModStatusEntry{}, false
	}
	raw := map[string]any{}
	for _, key := range []string{"Title", "Name", "Enabled", "Published", "Type", "ModType", "Path", "Id", "WorkshopId"} {
		if value, ok := m[key]; ok {
			raw[key] = value
		}
	}
	return ModStatusEntry{Title: title, ID: id, Path: path, Enabled: enabled, Published: published, Type: modType, RawKeys: raw}, true
}

func matchModStatusEntries(entries []ModStatusEntry, title, modPath string) []ModStatusEntry {
	title = strings.ToLower(strings.TrimSpace(title))
	base := strings.ToLower(filepath.Base(modPath))
	cleanPath := strings.ToLower(filepath.ToSlash(modPath))
	var out []ModStatusEntry
	for _, entry := range entries {
		entryTitle := strings.ToLower(strings.TrimSpace(entry.Title))
		entryPath := strings.ToLower(filepath.ToSlash(entry.Path))
		if title != "" && entryTitle == title {
			out = append(out, entry)
			continue
		}
		if base != "" && entryTitle == base {
			out = append(out, entry)
			continue
		}
		if entryPath != "" && (strings.Contains(cleanPath, entryPath) || strings.Contains(entryPath, base)) {
			out = append(out, entry)
		}
	}
	return out
}

func activeDataMods(entries []ModStatusEntry) []string {
	var out []string
	for _, entry := range entries {
		if entry.Enabled == nil || !*entry.Enabled {
			continue
		}
		if !looksDataMod(entry) {
			continue
		}
		name := entry.Title
		if name == "" {
			name = entry.ID
		}
		if name == "" {
			name = entry.Path
		}
		if name != "" {
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out
}

func looksDataMod(entry ModStatusEntry) bool {
	text := strings.ToLower(strings.Join([]string{entry.Type, entry.Title, entry.Path}, " "))
	return strings.Contains(text, "data")
}

func anyBool(entries []ModStatusEntry, get func(ModStatusEntry) *bool) *bool {
	for _, entry := range entries {
		if value := get(entry); value != nil {
			return value
		}
	}
	return nil
}

func firstString(m map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := m[key]; ok {
			switch v := value.(type) {
			case string:
				return strings.TrimSpace(v)
			case float64:
				return fmt.Sprintf("%.0f", v)
			}
		}
	}
	return ""
}

func firstBool(m map[string]any, keys ...string) *bool {
	for _, key := range keys {
		if value, ok := m[key]; ok {
			switch v := value.(type) {
			case bool:
				return &v
			case string:
				lower := strings.ToLower(strings.TrimSpace(v))
				if lower == "true" || lower == "yes" || lower == "1" {
					value := true
					return &value
				}
				if lower == "false" || lower == "no" || lower == "0" {
					value := false
					return &value
				}
			case float64:
				value := v != 0
				return &value
			}
		}
	}
	return nil
}

func modTitle(info map[string]any) string {
	for _, key := range []string{"Title", "title", "name", "Name"} {
		if value, ok := info[key].(string); ok && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func prefixedMessages(prefix string, messages []string) []string {
	out := make([]string, 0, len(messages))
	for _, message := range messages {
		out = append(out, prefix+message)
	}
	return out
}

func checkStringFile(root, rel string, report *CheckReport) {
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		report.Errors = append(report.Errors, fmt.Sprintf("cannot read %s: %v", rel, err))
		return
	}
	lines := strings.Split(string(data), "\n")
	malformed := 0
	checked := 0
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		checked++
		fields := strings.Fields(line)
		if len(fields) < 2 || !allDigits(fields[0]) {
			malformed++
		}
	}
	if checked > 0 && malformed > 0 {
		report.Warnings = append(report.Warnings, fmt.Sprintf("%s has %d malformed key-value string line(s)", rel, malformed))
	}
}

func allDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func checkAIPairs(report *CheckReport) {
	type aiSet struct {
		ai   bool
		per  bool
		per2 bool
	}
	sets := map[string]*aiSet{}
	for _, rel := range report.AIFiles {
		ext := strings.ToLower(filepath.Ext(rel))
		base := strings.TrimSuffix(rel, filepath.Ext(rel))
		set := sets[base]
		if set == nil {
			set = &aiSet{}
			sets[base] = set
		}
		switch ext {
		case ".ai":
			set.ai = true
		case ".per":
			set.per = true
		case ".per2":
			set.per2 = true
		}
	}
	keys := make([]string, 0, len(sets))
	for key := range sets {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		set := sets[key]
		if set.ai && !set.per && !set.per2 {
			report.Warnings = append(report.Warnings, fmt.Sprintf("%s.ai has no same-base .per/.per2 script", key))
		}
		if (set.per || set.per2) && !set.ai {
			report.Warnings = append(report.Warnings, fmt.Sprintf("%s script has no same-base .ai loader", key))
		}
	}
}
