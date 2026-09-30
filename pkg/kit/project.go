package kit

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"aoe2kit/pkg/aoe2"
)

type ProjectInspectReport struct {
	Root            string         `json:"root"`
	Verification    string         `json:"verification"`
	FilesScanned    int            `json:"files_scanned"`
	Shown           int            `json:"shown"`
	RoleCounts      map[string]int `json:"role_counts"`
	KindCounts      map[string]int `json:"kind_counts"`
	Files           []ProjectFile  `json:"files"`
	Warnings        []string       `json:"warnings,omitempty"`
	Recommendations []string       `json:"recommendations,omitempty"`
}

type ProjectFile struct {
	Path      string   `json:"path"`
	Kind      string   `json:"kind"`
	Role      string   `json:"role"`
	SizeBytes int64    `json:"size_bytes"`
	SHA256    string   `json:"sha256,omitempty"`
	Reasons   []string `json:"reasons,omitempty"`
	Warnings  []string `json:"warnings,omitempty"`
}

type ProjectInspectOptions struct {
	Limit int
}

type ProjectSnapshotReport struct {
	Root         string               `json:"root"`
	Output       string               `json:"output,omitempty"`
	OK           bool                 `json:"ok"`
	Verification string               `json:"verification"`
	Inspect      ProjectInspectReport `json:"inspect"`
}

type ProjectDiffReport struct {
	Before       string              `json:"before"`
	After        string              `json:"after"`
	Same         bool                `json:"same"`
	Verification string              `json:"verification"`
	Summary      ProjectDiffSummary  `json:"summary"`
	Changes      []ProjectFileChange `json:"changes,omitempty"`
}

type ProjectDiffSummary struct {
	Added    int `json:"added"`
	Removed  int `json:"removed"`
	Modified int `json:"modified"`
	Shown    int `json:"shown"`
	Limit    int `json:"limit,omitempty"`
}

type ProjectFileChange struct {
	Path   string       `json:"path"`
	Change string       `json:"change"`
	Before *ProjectFile `json:"before,omitempty"`
	After  *ProjectFile `json:"after,omitempty"`
	Fields []string     `json:"fields,omitempty"`
}

func InspectProject(root string, opts ProjectInspectOptions) (ProjectInspectReport, error) {
	st, err := os.Stat(root)
	if err != nil {
		return ProjectInspectReport{}, err
	}
	if !st.IsDir() {
		return ProjectInspectReport{}, &notDirError{path: root}
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return ProjectInspectReport{}, err
	}
	report := ProjectInspectReport{
		Root:         root,
		Verification: "structure_verified_not_engine_verified",
		RoleCounts:   map[string]int{},
		KindCounts:   map[string]int{},
	}
	err = filepath.WalkDir(rootAbs, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			report.Warnings = append(report.Warnings, walkErr.Error())
			return nil
		}
		if d.IsDir() {
			if shouldSkipDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(rootAbs, path)
		if err != nil {
			report.Warnings = append(report.Warnings, err.Error())
			return nil
		}
		rel = filepath.ToSlash(rel)
		info, err := aoe2.InspectFile(path)
		if err != nil {
			report.Warnings = append(report.Warnings, err.Error())
			return nil
		}
		file := ProjectFile{
			Path:      rel,
			Kind:      info.Kind,
			Role:      classifyProjectRole(rel, info.Kind),
			SizeBytes: info.SizeBytes,
			SHA256:    info.SHA256,
			Reasons:   projectRoleReasons(rel, info.Kind),
		}
		if isDurableScript(rel) {
			file.Warnings = append(file.Warnings, "durable workflow script; consider replacing with a Go command or documenting as disposable")
		}
		if info.SizeBytes <= 2*1024*1024 && mayContainText(info.Kind, rel) {
			if data, err := os.ReadFile(path); err == nil && !looksBinary(data) {
				for _, leak := range localPathLeaksInText(data) {
					file.Warnings = append(file.Warnings, "local path reference: "+leak)
				}
			}
		}
		report.Files = append(report.Files, file)
		report.FilesScanned++
		report.RoleCounts[file.Role]++
		report.KindCounts[file.Kind]++
		return nil
	})
	if err != nil {
		return ProjectInspectReport{}, err
	}
	sort.Slice(report.Files, func(i, j int) bool {
		if report.Files[i].Role != report.Files[j].Role {
			return report.Files[i].Role < report.Files[j].Role
		}
		return report.Files[i].Path < report.Files[j].Path
	})
	if opts.Limit > 0 && len(report.Files) > opts.Limit {
		report.Files = report.Files[:opts.Limit]
	}
	report.Shown = len(report.Files)
	report.Recommendations = projectRecommendations(report)
	return report, nil
}

func WriteProjectSnapshot(root, output string) (ProjectSnapshotReport, error) {
	inspect, err := InspectProject(root, ProjectInspectOptions{})
	if err != nil {
		return ProjectSnapshotReport{}, err
	}
	report := ProjectSnapshotReport{
		Root:         root,
		Output:       output,
		OK:           true,
		Verification: "structure_verified_not_engine_verified",
		Inspect:      inspect,
	}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return ProjectSnapshotReport{}, err
	}
	data = append(data, '\n')
	if err := os.WriteFile(output, data, 0644); err != nil {
		return ProjectSnapshotReport{}, err
	}
	return report, nil
}

func LoadProjectSnapshot(path string) (ProjectSnapshotReport, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return ProjectSnapshotReport{}, err
	}
	var snapshot ProjectSnapshotReport
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return ProjectSnapshotReport{}, err
	}
	if snapshot.Inspect.Root != "" {
		return snapshot, nil
	}
	var inspect ProjectInspectReport
	if err := json.Unmarshal(data, &inspect); err != nil {
		return ProjectSnapshotReport{}, err
	}
	if inspect.Root == "" {
		return ProjectSnapshotReport{}, fmt.Errorf("%s is not a project snapshot or inspect report", path)
	}
	return ProjectSnapshotReport{
		Root:         inspect.Root,
		Output:       path,
		OK:           true,
		Verification: inspect.Verification,
		Inspect:      inspect,
	}, nil
}

func DiffProjectSnapshots(beforePath, afterPath string, limit int) (ProjectDiffReport, error) {
	before, err := LoadProjectSnapshot(beforePath)
	if err != nil {
		return ProjectDiffReport{}, fmt.Errorf("load before: %w", err)
	}
	after, err := LoadProjectSnapshot(afterPath)
	if err != nil {
		return ProjectDiffReport{}, fmt.Errorf("load after: %w", err)
	}
	report := ProjectDiffReport{
		Before:       beforePath,
		After:        afterPath,
		Verification: "structure_verified_not_engine_verified",
	}
	beforeFiles := map[string]ProjectFile{}
	afterFiles := map[string]ProjectFile{}
	for _, file := range before.Inspect.Files {
		beforeFiles[file.Path] = file
	}
	for _, file := range after.Inspect.Files {
		afterFiles[file.Path] = file
	}
	seen := map[string]bool{}
	for path, beforeFile := range beforeFiles {
		seen[path] = true
		afterFile, ok := afterFiles[path]
		if !ok {
			report.Changes = append(report.Changes, ProjectFileChange{Path: path, Change: "removed", Before: &beforeFile})
			report.Summary.Removed++
			continue
		}
		fields := changedProjectFileFields(beforeFile, afterFile)
		if len(fields) > 0 {
			b, a := beforeFile, afterFile
			report.Changes = append(report.Changes, ProjectFileChange{Path: path, Change: "modified", Before: &b, After: &a, Fields: fields})
			report.Summary.Modified++
		}
	}
	for path, afterFile := range afterFiles {
		if seen[path] {
			continue
		}
		file := afterFile
		report.Changes = append(report.Changes, ProjectFileChange{Path: path, Change: "added", After: &file})
		report.Summary.Added++
	}
	sort.Slice(report.Changes, func(i, j int) bool {
		if report.Changes[i].Change != report.Changes[j].Change {
			return projectChangeRank(report.Changes[i].Change) < projectChangeRank(report.Changes[j].Change)
		}
		return report.Changes[i].Path < report.Changes[j].Path
	})
	if limit > 0 && len(report.Changes) > limit {
		report.Changes = report.Changes[:limit]
		report.Summary.Limit = limit
	}
	report.Summary.Shown = len(report.Changes)
	report.Same = report.Summary.Added == 0 && report.Summary.Removed == 0 && report.Summary.Modified == 0
	return report, nil
}

func changedProjectFileFields(before, after ProjectFile) []string {
	var fields []string
	if before.Kind != after.Kind {
		fields = append(fields, "kind")
	}
	if before.Role != after.Role {
		fields = append(fields, "role")
	}
	if before.SizeBytes != after.SizeBytes {
		fields = append(fields, "size_bytes")
	}
	if before.SHA256 != after.SHA256 {
		fields = append(fields, "sha256")
	}
	if strings.Join(before.Warnings, "\n") != strings.Join(after.Warnings, "\n") {
		fields = append(fields, "warnings")
	}
	return fields
}

func projectChangeRank(change string) int {
	switch change {
	case "removed":
		return 0
	case "added":
		return 1
	case "modified":
		return 2
	default:
		return 3
	}
}

func classifyProjectRole(rel, kind string) string {
	parts := strings.Split(strings.ToLower(rel), "/")
	base := parts[len(parts)-1]
	for _, part := range parts[:len(parts)-1] {
		switch part {
		case "reference", "references", "ref", "refs", "vanilla", "official", "originals", "archive", "archives":
			return "reference"
		case "output", "outputs", "generated", "build", "dist", "exports", "pack":
			return "generated"
		case "diagnostics", "diagnostic", "tests", "test", "samples", "fixtures":
			return "test_fixture"
		case "wip", "work", "working", "src", "source", "recipes", "recipe", "scripts":
			return "source"
		case "resources", "mods":
			return "packaged_runtime"
		case "docs", "doc":
			return "documentation"
		}
	}
	if strings.Contains(base, "generated") || strings.Contains(base, "output") {
		return "generated"
	}
	if strings.Contains(base, "reference") || strings.Contains(base, "vanilla") || strings.Contains(base, "original") {
		return "reference"
	}
	if strings.Contains(base, "diagnostic") || strings.Contains(base, "test") || strings.Contains(base, "fixture") {
		return "test_fixture"
	}
	switch kind {
	case "md", "txt", "text":
		return "documentation"
	case "json":
		if strings.Contains(base, "recipe") || strings.Contains(base, "manifest") {
			return "source"
		}
		return "metadata"
	case "go", "xs", "ai", "py", "sh", "ps1", "js":
		return "source"
	case "scenario", "dat":
		return "candidate_artifact"
	case "record", "zip":
		return "evidence"
	default:
		return "unknown"
	}
}

func projectRoleReasons(rel, kind string) []string {
	low := strings.ToLower(rel)
	var reasons []string
	if kind != "" {
		reasons = append(reasons, "kind="+kind)
	}
	for _, token := range []string{"reference", "vanilla", "official", "output", "generated", "diagnostic", "test", "fixture", "wip", "recipe", "resources/_common"} {
		if strings.Contains(low, token) {
			reasons = append(reasons, "path_contains="+token)
		}
	}
	if len(reasons) == 0 {
		reasons = append(reasons, "path_heuristic")
	}
	return reasons
}

func isDurableScript(rel string) bool {
	low := strings.ToLower(rel)
	return strings.HasSuffix(low, ".py") || strings.HasSuffix(low, ".sh")
}

func mayContainText(kind, rel string) bool {
	switch kind {
	case "json", "text", "txt", "md", "go", "xs", "ai", "per", "per2", "xml", "yml", "yaml", "toml", "csv":
		return true
	}
	ext := strings.ToLower(filepath.Ext(rel))
	switch ext {
	case ".md", ".txt", ".json", ".go", ".xs", ".ai", ".per", ".per2", ".xml", ".yml", ".yaml", ".toml", ".csv", ".sh", ".py":
		return true
	default:
		return false
	}
}

func localPathLeaksInText(data []byte) []string {
	text := string(data)
	needles := []string{"C:" + "\\Users\\", "/mnt/c/" + "Users/", "/home/"}
	var leaks []string
	for _, needle := range needles {
		if strings.Contains(text, needle) {
			leaks = append(leaks, needle)
		}
	}
	return leaks
}

func projectRecommendations(report ProjectInspectReport) []string {
	var out []string
	if report.RoleCounts["unknown"] > 0 {
		out = append(out, fmt.Sprintf("%d unknown-role file(s); inspect before automated writes", report.RoleCounts["unknown"]))
	}
	if report.RoleCounts["candidate_artifact"] > 0 {
		out = append(out, "candidate AoE2 artifacts found; identify source/reference/generated role before patching")
	}
	if report.KindCounts["scenario"] > 0 {
		out = append(out, "scenario files present; run kit scen verify/lint/xs before writes")
	}
	if report.KindCounts["dat"] > 0 {
		out = append(out, "DAT files present; run kit dat roundtrip or codec-plan before writes")
	}
	if report.KindCounts["record"] > 0 {
		out = append(out, "replays present; use replay chat/coverage/story as evidence before diagnosing behavior")
	}
	if report.KindCounts["json"] > 0 && report.RoleCounts["metadata"] > 0 {
		out = append(out, "project snapshots are best written outside the inspected root, or under an ignored build directory, to avoid self-noise in later diffs")
	}
	return out
}
