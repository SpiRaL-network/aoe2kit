package xs

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

// ParseCheckReport records the result of an optional external XS parser.
// Kit's source lint remains useful, but it is not a replacement for the
// game's grammar or an external parser that models it.
type ParseCheckReport struct {
	Checker string `json:"checker"`
	Status  string `json:"status"`
	Output  string `json:"output,omitempty"`
}

type ExternalFinding struct {
	Code     string `json:"code"`
	Severity string `json:"severity"`
	File     string `json:"file,omitempty"`
	Line     int    `json:"line,omitempty"`
	Message  string `json:"message"`
}

type DifferentialFinding struct {
	Class      string `json:"class"`
	File       string `json:"file,omitempty"`
	Line       int    `json:"line,omitempty"`
	Severity   string `json:"severity"`
	KitCode    string `json:"kit_code,omitempty"`
	KitMessage string `json:"kit_message,omitempty"`
	XSCode     string `json:"xscheck_code,omitempty"`
	XSMessage  string `json:"xscheck_message,omitempty"`
}

// CheckASCIIFile enforces the byte contract of generated XS. DE's tokenizer
// accepts ASCII comments but rejects non-ASCII bytes in string literals; Kit's
// generators own their output, so rejecting every non-ASCII byte keeps the
// generated/runtime boundary deterministic and avoids a misleading external
// parser pass.
func CheckASCIIFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read XS %s: %w", path, err)
	}
	line, column := 1, 1
	for offset, b := range data {
		if b >= 0x80 {
			return fmt.Errorf("XS %s contains non-ASCII byte 0x%02x at byte %d (line %d, column %d)", path, b, offset, line, column)
		}
		if b == '\n' {
			line++
			column = 1
		} else {
			column++
		}
	}
	return nil
}

var parserDiagnostic = regexp.MustCompile(`(?i)(^|\s)error(?:\s|:|$)`)
var ansiEscape = regexp.MustCompile(`\x1b\[[0-?]*[ -/]*[@-~]`)
var externalFindingRE = regexp.MustCompile(`(?m)^\s*\[\d+\]\s+(Warning|Error):\s+([A-Za-z0-9_]+)\s*$`)
var sourceLineRE = regexp.MustCompile(`(?m)([^\s:]+):(\d+)(?::\d+)?`)

// ResolveParser finds an explicitly requested checker, AOE2KIT_XS_CHECK, the
// user-local pinned install, or an xs-check executable on PATH. An empty
// result means unavailable.
func ResolveParser(explicit string) (string, error) {
	candidate := strings.TrimSpace(explicit)
	if candidate == "" {
		candidate = strings.TrimSpace(os.Getenv("AOE2KIT_XS_CHECK"))
	}
	if candidate == "" {
		if pinned, err := resolvePinnedParser(); err != nil {
			return "", err
		} else if pinned != "" {
			return pinned, nil
		}
		found, err := exec.LookPath("xs-check")
		if err != nil {
			return "", nil
		}
		return found, nil
	}
	if strings.ContainsRune(candidate, os.PathSeparator) {
		info, err := os.Stat(candidate)
		if err != nil {
			return "", fmt.Errorf("XS parser %q: %w", candidate, err)
		}
		if info.IsDir() {
			return "", fmt.Errorf("XS parser %q is a directory", candidate)
		}
		return filepath.Clean(candidate), nil
	}
	found, err := exec.LookPath(candidate)
	if err != nil {
		return "", fmt.Errorf("XS parser %q: %w", candidate, err)
	}
	return found, nil
}

const pinnedXSCheckSHA256 = "b35a5fecd8512b5ac9ec5111ea5848f1f92c20958bd9d13fbe47a5f8c1149d05"

func resolvePinnedParser() (string, error) {
	candidates := []string{}
	if local := localXSCheckPath(); local != "" {
		candidates = append(candidates, local)
	}
	if root := strings.TrimSpace(os.Getenv("AOE2KIT_ROOT")); root != "" {
		candidates = append(candidates, filepath.Join(root, "tools", "bin", "xs-check"))
	}
	if _, source, _, ok := runtime.Caller(0); ok {
		candidates = append(candidates, filepath.Join(filepath.Dir(source), "..", "..", "tools", "bin", "xs-check"))
	}
	for _, base := range []string{workingDirectory(), executableDirectory()} {
		for dir := base; dir != ""; dir = filepath.Dir(dir) {
			candidates = append(candidates, filepath.Join(dir, "tools", "bin", "xs-check"))
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
		}
	}
	seen := map[string]bool{}
	for _, candidate := range candidates {
		candidate = filepath.Clean(candidate)
		if seen[candidate] {
			continue
		}
		seen[candidate] = true
		info, err := os.Stat(candidate)
		if err != nil || info.IsDir() {
			continue
		}
		digest, err := fileSHA256(candidate)
		if err != nil {
			return "", fmt.Errorf("hash pinned xs-check %q: %w", candidate, err)
		}
		if digest != pinnedXSCheckSHA256 {
			return "", fmt.Errorf("pinned xs-check %q sha256=%s, want %s", candidate, digest, pinnedXSCheckSHA256)
		}
		return candidate, nil
	}
	return "", nil
}

func localXSCheckPath() string {
	dataHome := strings.TrimSpace(os.Getenv("XDG_DATA_HOME"))
	if dataHome == "" {
		home, err := os.UserHomeDir()
		if err != nil || strings.TrimSpace(home) == "" {
			return ""
		}
		dataHome = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(dataHome, "aoe2kit", "xs-check")
}

func workingDirectory() string {
	dir, _ := os.Getwd()
	return dir
}

func executableDirectory() string {
	path, err := os.Executable()
	if err != nil {
		return ""
	}
	return filepath.Dir(path)
}

func fileSHA256(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:]), nil
}

// RequireParser resolves the external checker for generated XS. Generated
// output must never be reported as ready when the independent parser was not
// available; callers that intentionally support foreign or optional XS should
// continue using ResolveParser directly.
func RequireParser(explicit string) (string, error) {
	checker, err := ResolveParser(explicit)
	if err != nil {
		return "", err
	}
	if checker == "" {
		return "", fmt.Errorf("xs-check is required for generated XS; set AOE2KIT_XS_CHECK or put xs-check on PATH")
	}
	return checker, nil
}

// CheckFile runs an external checker against one XS file. xs-check currently
// exits zero even when it prints parser errors, so output is part of the gate.
func CheckFile(path, checker string) (ParseCheckReport, error) {
	report := ParseCheckReport{Checker: checker, Status: "passed"}
	if err := CheckASCIIFile(path); err != nil {
		report.Status = "failed"
		report.Output = err.Error()
		return report, err
	}
	cmd := exec.Command(checker, path)
	output, runErr := cmd.CombinedOutput()
	clean := strings.TrimSpace(ansiEscape.ReplaceAllString(string(output), ""))
	report.Output = clean
	if runErr != nil || parserDiagnostic.MatchString(clean) {
		report.Status = "failed"
		if clean == "" && runErr != nil {
			clean = runErr.Error()
		}
		return report, fmt.Errorf("XS parser rejected %s: %s", path, clean)
	}
	return report, nil
}

func ExternalFindings(report ParseCheckReport) []ExternalFinding {
	clean := strings.TrimSpace(ansiEscape.ReplaceAllString(report.Output, ""))
	matches := externalFindingRE.FindAllStringSubmatchIndex(clean, -1)
	out := make([]ExternalFinding, 0, len(matches))
	for _, match := range matches {
		severity := clean[match[2]:match[3]]
		code := clean[match[4]:match[5]]
		end := len(clean)
		if len(match) > 1 {
			for _, next := range matches {
				if next[0] > match[1] && next[0] < end {
					end = next[0]
				}
			}
		}
		segment := clean[match[1]:end]
		file := ""
		line := 0
		if ref := sourceLineRE.FindStringSubmatch(segment); len(ref) == 3 {
			file = ref[1]
			line, _ = strconv.Atoi(ref[2])
		}
		out = append(out, ExternalFinding{Code: code, Severity: strings.ToLower(severity), File: file, Line: line, Message: strings.TrimSpace(segment)})
	}
	return out
}

func CompareFindings(kit []SourceFinding, external []ExternalFinding) []DifferentialFinding {
	used := make([]bool, len(external))
	out := make([]DifferentialFinding, 0)
	for _, finding := range kit {
		match := -1
		for i, other := range external {
			if used[i] || finding.Line != other.Line || finding.Severity != other.Severity {
				continue
			}
			match = i
			break
		}
		if match >= 0 {
			used[match] = true
			other := external[match]
			out = append(out, DifferentialFinding{Class: "BOTH", File: finding.File, Line: finding.Line, Severity: finding.Severity, KitCode: finding.Code, KitMessage: finding.Message, XSCode: other.Code, XSMessage: other.Message})
		} else {
			out = append(out, DifferentialFinding{Class: "KIT_ONLY", File: finding.File, Line: finding.Line, Severity: finding.Severity, KitCode: finding.Code, KitMessage: finding.Message})
		}
	}
	for i, finding := range external {
		if used[i] {
			continue
		}
		out = append(out, DifferentialFinding{Class: "XSCHECK_ONLY", File: finding.File, Line: finding.Line, Severity: finding.Severity, XSCode: finding.Code, XSMessage: finding.Message})
	}
	return out
}
