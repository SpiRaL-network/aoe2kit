package ci

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"aoe2kit/pkg/replay"
)

const DefaultConfigName = "aoe2kit-ci.json"

type Config struct {
	SchemaVersion int     `json:"schema_version"`
	Name          string  `json:"name,omitempty"`
	Scenario      string  `json:"scenario,omitempty"`
	Checks        []Check `json:"checks"`
}

type Check struct {
	Name     string `json:"name,omitempty"`
	Type     string `json:"type"`
	Replay   string `json:"replay,omitempty"`
	Contract string `json:"contract,omitempty"`
	XSDat    string `json:"xsdat,omitempty"`
	Ledger   string `json:"ledger,omitempty"`
	Schema   string `json:"schema,omitempty"`
	Player   int    `json:"player,omitempty"`
	WindowMS int    `json:"window_ms,omitempty"`
}

type InitReport struct {
	Root       string   `json:"root"`
	ConfigPath string   `json:"config_path"`
	DebugXS    string   `json:"debug_xs"`
	Created    []string `json:"created,omitempty"`
	Updated    []string `json:"updated,omitempty"`
}

type Report struct {
	Path    string   `json:"path"`
	Root    string   `json:"root"`
	Name    string   `json:"name,omitempty"`
	OK      bool     `json:"ok"`
	Summary Summary  `json:"summary"`
	Checks  []Result `json:"checks"`
	Warning []string `json:"warnings,omitempty"`
	Config  *Config  `json:"config,omitempty"`
}

type Summary struct {
	Passed  int `json:"passed"`
	Failed  int `json:"failed"`
	Unknown int `json:"unknown"`
}

type Result struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Status   string `json:"status"`
	Source   string `json:"source,omitempty"`
	Message  string `json:"message,omitempty"`
	Replay   string `json:"replay,omitempty"`
	Contract string `json:"contract,omitempty"`
	XSDat    string `json:"xsdat,omitempty"`
	Ledger   string `json:"ledger,omitempty"`
	Schema   string `json:"schema,omitempty"`
	Details  any    `json:"details,omitempty"`
}

func Init(root string, scenario string, overwrite bool) (*InitReport, error) {
	if strings.TrimSpace(root) == "" {
		root = "."
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Join(absRoot, "ci"), 0o755); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Join(absRoot, "xs"), 0o755); err != nil {
		return nil, err
	}
	report := &InitReport{
		Root:       absRoot,
		ConfigPath: filepath.Join(absRoot, DefaultConfigName),
		DebugXS:    filepath.Join(absRoot, "xs", "a2k_debug.xs"),
	}
	cfg := Config{
		SchemaVersion: 1,
		Name:          "AoE2Kit scenario CI",
		Scenario:      scenario,
		Checks: []Check{
			{
				Name:     "example verify-run contract",
				Type:     "verify_run",
				Replay:   "replace-with-playtest.aoe2record",
				Contract: "ci/verify-run.contract.json",
			},
			{
				Name:     "optional sidecar sync",
				Type:     "sidecar_sync",
				Replay:   "replace-with-playtest.aoe2record",
				XSDat:    "replace-with-sidecar.xsdat",
				Schema:   "rtv14",
				Player:   1,
				WindowMS: 20000,
			},
		},
	}
	if err := writeJSONIfAllowed(report.ConfigPath, cfg, overwrite, &report.Created, &report.Updated); err != nil {
		return nil, err
	}
	contract := replay.VerifyRunContract{
		Name: "Example AoE2Kit CI contract",
		Scenario: &replay.ScenarioExpectation{
			Tier:        "trigger_graph",
			Fingerprint: "replace-with-expected-trigger-graph-sha256",
		},
		Players: &replay.PlayersExpectation{MinCount: 1},
		Chat: []replay.ChatExpectation{
			{Contains: "replace-with-observed-marker", MinCount: 1},
		},
		RenderExpectations: []replay.RenderExpectation{
			{ID: "manual_render_check", Description: "Describe the expected in-engine visual result here."},
		},
	}
	if err := writeJSONIfAllowed(filepath.Join(absRoot, "ci", "verify-run.contract.json"), contract, overwrite, &report.Created, &report.Updated); err != nil {
		return nil, err
	}
	if err := writeTextIfAllowed(report.DebugXS, defaultDebugXS(), overwrite, &report.Created, &report.Updated); err != nil {
		return nil, err
	}
	return report, nil
}

func CheckFile(path string) (*Report, error) {
	if strings.TrimSpace(path) == "" {
		path = DefaultConfigName
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("%s is not valid AoE2Kit CI JSON: %w", path, err)
	}
	root := filepath.Dir(absPath)
	report := &Report{Path: absPath, Root: root, Name: cfg.Name, Config: &cfg}
	if cfg.SchemaVersion != 1 {
		report.Warning = append(report.Warning, fmt.Sprintf("schema_version=%d is not recognized; expected 1", cfg.SchemaVersion))
	}
	for idx, check := range cfg.Checks {
		result := runCheck(root, idx, check)
		report.Checks = append(report.Checks, result)
		switch result.Status {
		case "pass":
			report.Summary.Passed++
		case "fail":
			report.Summary.Failed++
		default:
			report.Summary.Unknown++
		}
	}
	report.OK = report.Summary.Failed == 0 && report.Summary.Unknown == 0
	return report, nil
}

func runCheck(root string, idx int, check Check) Result {
	name := strings.TrimSpace(check.Name)
	if name == "" {
		name = fmt.Sprintf("check_%d", idx+1)
	}
	result := Result{Name: name, Type: check.Type}
	switch check.Type {
	case "verify_run":
		return runVerifyRunCheck(root, check, result)
	case "sidecar_sync":
		return runSidecarSyncCheck(root, check, result)
	default:
		result.Status = "unknown"
		result.Message = "unknown CI check type"
		return result
	}
}

func runVerifyRunCheck(root string, check Check, result Result) Result {
	replayPath := resolvePath(root, check.Replay)
	contractPath := resolvePath(root, check.Contract)
	result.Replay = replayPath
	result.Contract = contractPath
	result.Source = "kit verify-run"
	if check.Replay == "" || check.Contract == "" {
		result.Status = "unknown"
		result.Message = "verify_run requires replay and contract"
		return result
	}
	report, err := replay.VerifyRun(replayPath, contractPath)
	if err != nil {
		result.Status = "unknown"
		result.Message = err.Error()
		return result
	}
	result.Details = report.Summary
	if report.Summary.Failed > 0 {
		result.Status = "fail"
		result.Message = "verify-run contract failed"
		return result
	}
	if report.Summary.Unknown > 0 {
		result.Status = "unknown"
		result.Message = "verify-run contract has unknown claims"
		return result
	}
	result.Status = "pass"
	result.Message = "verify-run contract passed"
	return result
}

func runSidecarSyncCheck(root string, check Check, result Result) Result {
	replayPath := resolvePath(root, check.Replay)
	xsdatPath := resolvePath(root, check.XSDat)
	ledgerPath := resolvePath(root, check.Ledger)
	result.Replay = replayPath
	result.XSDat = xsdatPath
	result.Ledger = ledgerPath
	result.Schema = check.Schema
	result.Source = "kit replay sidecar-sync"
	if check.Replay == "" || check.XSDat == "" || (check.Ledger == "" && check.Schema == "") {
		result.Status = "unknown"
		result.Message = "sidecar_sync requires replay, xsdat, and ledger or schema"
		return result
	}
	report, err := replay.BuildSidecarSyncReport(replayPath, replay.SidecarSyncOptions{
		SidecarPath: xsdatPath,
		LedgerPath:  ledgerPath,
		Schema:      check.Schema,
		PlayerID:    check.Player,
		WindowMS:    check.WindowMS,
	})
	if err != nil {
		result.Status = "unknown"
		result.Message = err.Error()
		return result
	}
	result.Details = report.Summary
	if report.Summary.LedgerFailed > 0 || report.Summary.KnownWordMismatches > 0 {
		result.Status = "fail"
		result.Message = "sidecar-sync assertions failed"
		return result
	}
	if !report.Summary.SidecarDecodeOK || report.Summary.LedgerUnknown > 0 || report.Summary.MissingSamples > 0 {
		result.Status = "unknown"
		result.Message = "sidecar-sync has unknown or missing evidence"
		return result
	}
	result.Status = "pass"
	result.Message = "sidecar-sync assertions passed"
	return result
}

func resolvePath(root string, path string) string {
	if strings.TrimSpace(path) == "" {
		return ""
	}
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(root, path)
}

func writeJSONIfAllowed(path string, value any, overwrite bool, created *[]string, updated *[]string) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return writeTextIfAllowed(path, string(data), overwrite, created, updated)
}

func writeTextIfAllowed(path string, body string, overwrite bool, created *[]string, updated *[]string) error {
	if _, err := os.Stat(path); err == nil && !overwrite {
		return nil
	} else if err == nil {
		*updated = append(*updated, path)
	} else if os.IsNotExist(err) {
		*created = append(*created, path)
	} else {
		return err
	}
	return os.WriteFile(path, []byte(body), 0o644)
}

func defaultDebugXS() string {
	return `// AoE2Kit debug include.
// Keep this module project-neutral. Scenario-specific probes should wrap these helpers.

void a2kDebugChat(string marker = "") {
    xsChatData("A2KDBG " + marker);
}

void a2kDebugChatKV(string key = "", int value = 0) {
    xsChatData("A2KDBG " + key + "=" + value);
}
`
}
