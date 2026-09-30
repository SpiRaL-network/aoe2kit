package aifile

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"aoe2kit/pkg/aoe2"
)

type LintReport struct {
	Path         string                 `json:"path"`
	OK           bool                   `json:"ok"`
	Verification aoe2.VerificationClaim `json:"verification"`
	Files        []LintFile             `json:"files"`
	Issues       []LintIssue            `json:"issues,omitempty"`
	Summary      LintSummary            `json:"summary"`
}

type LintFile struct {
	Path            string `json:"path"`
	RelativePath    string `json:"relative_path"`
	SizeBytes       int64  `json:"size_bytes"`
	NormalizedBytes int    `json:"normalized_bytes"`
	LineEndingsNote string `json:"line_endings_note,omitempty"`
}

type LintIssue struct {
	Severity string `json:"severity"`
	Code     string `json:"code"`
	File     string `json:"file,omitempty"`
	Message  string `json:"message"`
}

type LintSummary struct {
	Files  int `json:"files"`
	Errors int `json:"errors"`
	Warns  int `json:"warnings"`
}

type DiffReport struct {
	Before  string       `json:"before"`
	After   string       `json:"after"`
	Same    bool         `json:"same"`
	Changes []DiffChange `json:"changes,omitempty"`
}

type DiffChange struct {
	Kind   string `json:"kind"`
	File   string `json:"file,omitempty"`
	Before any    `json:"before,omitempty"`
	After  any    `json:"after,omitempty"`
}

func LintPath(path string) (*LintReport, error) {
	files, root, err := collectAIPaths(path)
	if err != nil {
		return nil, err
	}
	report := &LintReport{Path: path}
	for _, file := range files {
		st, err := os.Stat(file)
		if err != nil {
			report.addIssue("error", "stat_failed", file, err.Error())
			continue
		}
		data, err := os.ReadFile(file)
		if err != nil {
			report.addIssue("error", "read_failed", file, err.Error())
			continue
		}
		normalized, note := normalizeText(data)
		rel, err := filepath.Rel(root, file)
		if err != nil {
			rel = filepath.Base(file)
		}
		rel = filepath.ToSlash(rel)
		report.Files = append(report.Files, LintFile{
			Path:            file,
			RelativePath:    rel,
			SizeBytes:       st.Size(),
			NormalizedBytes: len(normalized),
			LineEndingsNote: note,
		})
		lintAIFile(rel, normalized, report)
	}
	report.Summary.Files = len(report.Files)
	for _, issue := range report.Issues {
		switch issue.Severity {
		case "error":
			report.Summary.Errors++
		case "warning":
			report.Summary.Warns++
		}
	}
	report.OK = report.Summary.Errors == 0
	report.Verification = aoe2.StructureVerification(report.OK)
	return report, nil
}

func DiffPaths(beforePath, afterPath string) (*DiffReport, error) {
	before, err := FingerprintPath(beforePath)
	if err != nil {
		return nil, fmt.Errorf("fingerprint before: %w", err)
	}
	after, err := FingerprintPath(afterPath)
	if err != nil {
		return nil, fmt.Errorf("fingerprint after: %w", err)
	}
	report := &DiffReport{Before: beforePath, After: afterPath, Same: before.SHA256 == after.SHA256}
	if before.SHA256 != after.SHA256 {
		report.Changes = append(report.Changes, DiffChange{Kind: "signature", Before: before.SHA256, After: after.SHA256})
	}
	beforeFiles := map[string]FingerprintFile{}
	afterFiles := map[string]FingerprintFile{}
	for _, file := range before.Files {
		beforeFiles[file.RelativePath] = file
	}
	for _, file := range after.Files {
		afterFiles[file.RelativePath] = file
	}
	names := map[string]bool{}
	for name := range beforeFiles {
		names[name] = true
	}
	for name := range afterFiles {
		names[name] = true
	}
	sorted := make([]string, 0, len(names))
	for name := range names {
		sorted = append(sorted, name)
	}
	sort.Strings(sorted)
	for _, name := range sorted {
		b, bok := beforeFiles[name]
		a, aok := afterFiles[name]
		switch {
		case !bok:
			report.Changes = append(report.Changes, DiffChange{Kind: "file_added", File: name, After: a.ContentSHA256})
		case !aok:
			report.Changes = append(report.Changes, DiffChange{Kind: "file_removed", File: name, Before: b.ContentSHA256})
		case b.ContentSHA256 != a.ContentSHA256:
			report.Changes = append(report.Changes, DiffChange{Kind: "file_changed", File: name, Before: b.ContentSHA256, After: a.ContentSHA256})
		}
	}
	return report, nil
}

func collectAIPaths(path string) ([]string, string, error) {
	st, err := os.Stat(path)
	if err != nil {
		return nil, "", err
	}
	root := path
	var files []string
	if st.IsDir() {
		err = filepath.WalkDir(path, func(p string, d os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if d.IsDir() {
				return nil
			}
			if IsAIPath(p) {
				files = append(files, p)
			}
			return nil
		})
		if err != nil {
			return nil, "", err
		}
	} else {
		root = filepath.Dir(path)
		if !IsAIPath(path) {
			return nil, "", fmt.Errorf("%s is not an .ai/.per file", path)
		}
		files = append(files, path)
	}
	if len(files) == 0 {
		return nil, "", fmt.Errorf("%s contains no .ai/.per files", path)
	}
	sort.Strings(files)
	return files, root, nil
}

func lintAIFile(rel string, data []byte, report *LintReport) {
	text := string(data)
	if len(strings.TrimSpace(text)) == 0 {
		if strings.EqualFold(filepath.Ext(rel), ".ai") {
			report.addIssue("warning", "empty_ai_loader", rel, ".ai loader is empty; this is valid only when the lobby loads same-base .per/.per2 content separately")
		} else {
			report.addIssue("error", "empty_ai_file", rel, "AI file is empty")
		}
		return
	}
	if ok, note := balancedParensIgnoringComments(text); !ok {
		report.addIssue("error", "unbalanced_parentheses", rel, note)
	}
	ext := strings.ToLower(filepath.Ext(rel))
	lower := strings.ToLower(text)
	if (ext == ".per" || ext == ".per2") && !strings.Contains(lower, "(defrule") && !strings.Contains(lower, "(defconst") && !strings.Contains(lower, "(set-strategic-number") {
		report.addIssue("warning", "per_file_has_no_rules_or_consts", rel, ".per/.per2 file has no defrule, defconst, or set-strategic-number form")
	}
}

func balancedParensIgnoringComments(text string) (bool, string) {
	depth := 0
	line := 1
	col := 0
	for i := 0; i < len(text); i++ {
		ch := text[i]
		col++
		if ch == '\n' {
			line++
			col = 0
			continue
		}
		if ch == ';' {
			for i < len(text) && text[i] != '\n' {
				i++
			}
			i--
			continue
		}
		switch ch {
		case '(':
			depth++
		case ')':
			depth--
			if depth < 0 {
				return false, fmt.Sprintf("extra closing parenthesis at line %d column %d", line, col)
			}
		}
	}
	if depth != 0 {
		return false, fmt.Sprintf("parenthesis depth ended at %d", depth)
	}
	return true, ""
}

func (r *LintReport) addIssue(severity, code, file, message string) {
	r.Issues = append(r.Issues, LintIssue{Severity: severity, Code: code, File: file, Message: message})
}
