package scenario

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	xsauthor "aoe2kit/pkg/xs"
)

type XSCensusOptions struct {
	DeployTree          string
	CheckCreateResults  bool
	DatPath             string
	RequireXSCheck      bool
	XSCheckPath         string
	AllowMissingXSCheck bool
}

type XSCensusReport struct {
	Path                  string                   `json:"path,omitempty"`
	Version               string                   `json:"version,omitempty"`
	Verification          string                   `json:"verification"`
	OK                    bool                     `json:"ok"`
	XS                    XSDeployAttachment       `json:"xs"`
	ScriptContentEmbedded bool                     `json:"script_content_embedded"`
	EmbeddedCarriers      []XSEmbeddedCarrier      `json:"embedded_carriers,omitempty"`
	DeployTree            string                   `json:"deploy_tree,omitempty"`
	ScriptCalls           []XSScriptCall           `json:"script_calls,omitempty"`
	UniqueCalledFunctions []string                 `json:"unique_called_functions,omitempty"`
	Functions             []xsauthor.XSFunction    `json:"functions,omitempty"`
	Analysis              *xsauthor.AnalysisReport `json:"analysis,omitempty"`
	Findings              []DeployCheckFinding     `json:"findings,omitempty"`
	Counts                XSCensusCounts           `json:"counts"`
}

type XSScriptCall struct {
	TriggerIndex int      `json:"trigger_index"`
	TriggerName  string   `json:"trigger_name,omitempty"`
	EffectIndex  int      `json:"effect_index"`
	Message      string   `json:"message"`
	Calls        []string `json:"calls,omitempty"`
}

type XSEmbeddedCarrier struct {
	TriggerIndex  int    `json:"trigger_index"`
	TriggerName   string `json:"trigger_name,omitempty"`
	EffectIndex   int    `json:"effect_index"`
	Title         string `json:"title,omitempty"`
	SourceBytes   int    `json:"source_bytes"`
	SourceSHA256  string `json:"source_sha256"`
	ContentBytes  int    `json:"content_bytes"`
	ContentSHA256 string `json:"content_sha256"`
	Source        string `json:"-"`
	Content       string `json:"-"`
}

type XSCensusCounts struct {
	ScriptContentAttachments int `json:"script_content_attachments"`
	EmbeddedCarriers         int `json:"embedded_carriers"`
	ScriptCalls              int `json:"script_calls"`
	UniqueCalledFunctions    int `json:"unique_called_functions"`
	Functions                int `json:"functions"`
	Includes                 int `json:"includes"`
	ResolvedIncludes         int `json:"resolved_includes"`
	MissingIncludes          int `json:"missing_includes"`
	Declarations             int `json:"declarations"`
	CrossFileNonExtern       int `json:"cross_file_nonextern"`
	Findings                 int `json:"findings"`
}

type XSDeployOptions struct {
	DeployTree   string
	Name         string
	CarrierIndex *int
	TriggerIndex *int
	Force        bool
}

type XSDeployReport struct {
	Path                 string `json:"path,omitempty"`
	DeployTree           string `json:"deploy_tree"`
	Output               string `json:"output"`
	ResolvedRelativePath string `json:"resolved_relative_path"`
	Name                 string `json:"name"`
	Source               string `json:"source"`
	ContentBytes         int    `json:"content_bytes"`
	ContentSHA256        string `json:"content_sha256"`
	Verification         string `json:"verification"`
}

var xsCallRE = regexp.MustCompile(`\b([A-Za-z_][A-Za-z0-9_]*)\s*\(`)
var xsCarrierTitleRE = regexp.MustCompile(`(?m)^// -{25,} (.+?) -{25,}\s*$`)

func XSCensusFile(path string, opts XSCensusOptions) (XSCensusReport, error) {
	file, err := Open(path)
	if err != nil {
		return XSCensusReport{}, err
	}
	report, err := file.XSCensus(opts)
	if err != nil {
		return XSCensusReport{}, err
	}
	report.Path = path
	return report, nil
}

func (f *File) XSCensus(opts XSCensusOptions) (XSCensusReport, error) {
	report := XSCensusReport{
		Version:      f.Version,
		Verification: "structure_verified_not_engine_verified",
		OK:           true,
		XS:           f.XSAttachment(),
		DeployTree:   opts.DeployTree,
	}
	report.EmbeddedCarriers = f.xsEmbeddedCarriers()
	report.ScriptCalls = f.xsScriptCalls()
	report.UniqueCalledFunctions = uniqueScriptCallNames(report.ScriptCalls)

	var sources []xsauthor.SourceFile
	entryPath := cleanXSName(report.XS.ScriptFilePath)
	if entryPath == "" {
		entryPath = cleanXSName(report.XS.ScriptName)
	}
	if entryPath == "" && report.XS.ScriptFileContentBytes > 0 {
		entryPath = "embedded.xs"
	}
	if strings.TrimSpace(opts.DeployTree) != "" && entryPath != "" {
		resolved, root, rel, _ := resolveDeployedXS(opts.DeployTree, entryPath)
		if resolved != "" {
			report.XS.ResolvedPath = resolved
			report.XS.ResolvedRelativePath = rel
			loaded, err := loadDeployXSSources(root)
			if err != nil {
				return XSCensusReport{}, err
			}
			sources = loaded
			entryPath = rel
		} else if report.XS.ScriptFileContentBytes > 0 {
			sources = append(sources, xsauthor.SourceFile{Path: entryPath, Content: f.xsEmbeddedContent()})
		}
	} else if report.XS.ScriptFileContentBytes > 0 {
		sources = append(sources, xsauthor.SourceFile{Path: entryPath, Content: f.xsEmbeddedContent()})
	}
	if len(sources) == 0 && len(report.EmbeddedCarriers) > 0 {
		for i, carrier := range report.EmbeddedCarriers {
			path := fmt.Sprintf("embedded_carrier_t%d_e%d.xs", carrier.TriggerIndex, carrier.EffectIndex)
			if carrier.Title != "" {
				path = fmt.Sprintf("embedded_carrier_%02d_%s.xs", i, sanitizeXSCarrierTitle(carrier.Title))
			}
			if i == 0 {
				entryPath = path
			}
			sources = append(sources, xsauthor.SourceFile{Path: path, Content: carrier.Source})
		}
		// Carrier triggers are independent entry points, not include descendants.
		entryPath = ""
	}
	if len(sources) > 0 {
		analysis, err := xsauthor.AnalyzeFiles(xsauthor.AnalyzeOptions{EntryPath: entryPath, Files: sources, CheckCreateResults: opts.CheckCreateResults, DatPath: opts.DatPath})
		if err != nil {
			return XSCensusReport{}, err
		}
		report.Analysis = &analysis
		report.Functions = xsFunctionsFromAnalysis(analysis, sources)
	}
	if strings.TrimSpace(opts.DeployTree) != "" {
		deploy, err := f.DeployCheck(opts.DeployTree)
		if err != nil {
			return XSCensusReport{}, err
		}
		report.OK = deploy.OK
		report.Findings = deploy.Findings
		if deploy.XS.ResolvedPath != "" {
			report.XS.ResolvedPath = deploy.XS.ResolvedPath
			report.XS.ResolvedRelativePath = deploy.XS.ResolvedRelativePath
		}
		if deploy.Analysis != nil {
			report.Analysis = deploy.Analysis
			report.Functions = xsFunctionsFromAnalysis(*deploy.Analysis, sources)
		}
	} else if report.Analysis != nil {
		for _, include := range report.Analysis.Includes {
			if include.Status != "resolved" {
				report.Findings = append(report.Findings, DeployCheckFinding{
					Severity: "info",
					What:     "embedded XS include cannot be resolved without --deploy-tree: " + include.Path,
					Path:     include.From,
					Line:     include.Line,
					Fix:      "Pass --deploy-tree to resolve includes against the shipped mod/profile tree.",
				})
			}
		}
	}
	if report.Analysis != nil {
		for _, finding := range report.Analysis.Findings {
			report.Findings = append(report.Findings, DeployCheckFinding{
				Severity: finding.Severity, What: finding.Code + ": " + finding.Message,
				Path: finding.File, Line: finding.Line, Fix: finding.Fix,
			})
			if finding.Severity == "error" {
				report.OK = false
			}
		}
	}
	if opts.RequireXSCheck && len(sources) > 0 {
		for _, finding := range checkExternalXS(sources, opts) {
			report.Findings = append(report.Findings, finding)
			if finding.Severity == "error" {
				report.OK = false
			}
		}
	}
	report.finishXSCensusCounts()
	return report, nil
}

func checkExternalXS(sources []xsauthor.SourceFile, opts XSCensusOptions) []DeployCheckFinding {
	checker, err := xsauthor.ResolveParser(opts.XSCheckPath)
	if err != nil {
		return []DeployCheckFinding{{Severity: "error", What: "xs-check resolution failed: " + err.Error(), Fix: "Install xs-check (see docs/XS_CHECK.md), pass --xs-check /path/to/xs-check, or pass --allow-missing-xs-check only for an explicitly non-gated inspection."}}
	}
	if checker == "" {
		if opts.AllowMissingXSCheck {
			return []DeployCheckFinding{{Severity: "warning", What: "xs-check unavailable; external XS verification was explicitly allowed to be skipped", Fix: "Install xs-check or remove --allow-missing-xs-check for a fail-closed result."}}
		}
		return []DeployCheckFinding{{Severity: "error", What: "xs-check is required but unavailable", Fix: "Install the pinned checker, set AOE2KIT_XS_CHECK, or pass --allow-missing-xs-check only when intentionally bypassing the gate."}}
	}
	var findings []DeployCheckFinding
	for _, source := range sources {
		file, err := os.CreateTemp("", "aoe2kit-xs-check-*.xs")
		if err != nil {
			findings = append(findings, DeployCheckFinding{Severity: "error", What: "could not stage XS for xs-check: " + err.Error(), Path: source.Path})
			continue
		}
		path := file.Name()
		_, writeErr := file.WriteString(source.Content)
		closeErr := file.Close()
		if writeErr == nil {
			writeErr = closeErr
		}
		if writeErr != nil {
			_ = os.Remove(path)
			findings = append(findings, DeployCheckFinding{Severity: "error", What: "could not stage XS for xs-check: " + writeErr.Error(), Path: source.Path})
			continue
		}
		_, checkErr := xsauthor.CheckFile(path, checker)
		_ = os.Remove(path)
		if checkErr != nil {
			findings = append(findings, DeployCheckFinding{Severity: "error", What: "xs-check rejected generated XS: " + checkErr.Error(), Path: source.Path, Fix: "Fix the independent checker finding before deploying this XS."})
		}
	}
	return findings
}

func DeployXSFile(path string, opts XSDeployOptions) (XSDeployReport, error) {
	file, err := Open(path)
	if err != nil {
		return XSDeployReport{}, err
	}
	report, err := file.DeployXS(opts)
	if err != nil {
		return XSDeployReport{}, err
	}
	report.Path = path
	return report, nil
}

func (f *File) DeployXS(opts XSDeployOptions) (XSDeployReport, error) {
	if strings.TrimSpace(opts.DeployTree) == "" {
		return XSDeployReport{}, fmt.Errorf("deploy tree is required")
	}
	name, content, source, err := f.selectXSForDeploy(opts)
	if err != nil {
		return XSDeployReport{}, err
	}
	if content == "" {
		return XSDeployReport{}, fmt.Errorf("selected XS source is empty")
	}
	name, err = safeDeployXSName(name)
	if err != nil {
		return XSDeployReport{}, err
	}
	outputRoot := filepath.Join(opts.DeployTree, "resources", "_common", "xs")
	output := filepath.Join(outputRoot, filepath.FromSlash(name))
	if !opts.Force {
		if _, err := os.Stat(output); err == nil {
			return XSDeployReport{}, fmt.Errorf("%s already exists; pass --force to overwrite", output)
		} else if !os.IsNotExist(err) {
			return XSDeployReport{}, err
		}
	}
	if err := os.MkdirAll(filepath.Dir(output), 0755); err != nil {
		return XSDeployReport{}, err
	}
	if err := os.WriteFile(output, []byte(normalizeXSFileContent(content)), 0644); err != nil {
		return XSDeployReport{}, err
	}
	written, err := os.ReadFile(output)
	if err != nil {
		return XSDeployReport{}, err
	}
	sum := sha256.Sum256(written)
	rel, err := filepath.Rel(opts.DeployTree, output)
	if err != nil {
		rel = filepath.Join("resources", "_common", "xs", filepath.Base(output))
	}
	return XSDeployReport{
		DeployTree:           opts.DeployTree,
		Output:               output,
		ResolvedRelativePath: filepath.ToSlash(rel),
		Name:                 name,
		Source:               source,
		ContentBytes:         len(written),
		ContentSHA256:        hex.EncodeToString(sum[:]),
		Verification:         "structure_verified_not_engine_verified",
	}, nil
}

func (f *File) selectXSForDeploy(opts XSDeployOptions) (name, content, source string, err error) {
	if opts.CarrierIndex != nil || opts.TriggerIndex != nil {
		census, err := f.XSCensus(XSCensusOptions{})
		if err != nil {
			return "", "", "", err
		}
		carrier, err := selectXSCensusCarrier(census, opts.CarrierIndex, opts.TriggerIndex)
		if err != nil {
			return "", "", "", err
		}
		name = opts.Name
		if name == "" {
			name = carrier.Title
		}
		if name == "" {
			name = fmt.Sprintf("trigger_%d_effect_%d.xs", carrier.TriggerIndex, carrier.EffectIndex)
		}
		return name, carrier.Content, "embedded_carrier", nil
	}
	attachment := f.XSAttachment()
	content = f.xsEmbeddedContent()
	if strings.TrimSpace(content) != "" {
		name = opts.Name
		if name == "" {
			name = attachment.ScriptFilePath
		}
		if name == "" {
			name = attachment.ScriptName
		}
		if name == "" {
			name = "scenario.xs"
		}
		return name, content, "scenario_attachment", nil
	}
	census, err := f.XSCensus(XSCensusOptions{})
	if err != nil {
		return "", "", "", err
	}
	carrier, err := selectXSCensusCarrier(census, opts.CarrierIndex, opts.TriggerIndex)
	if err != nil {
		return "", "", "", err
	}
	name = opts.Name
	if name == "" {
		name = carrier.Title
	}
	if name == "" {
		name = fmt.Sprintf("trigger_%d_effect_%d.xs", carrier.TriggerIndex, carrier.EffectIndex)
	}
	return name, carrier.Content, "embedded_carrier", nil
}

func selectXSCensusCarrier(report XSCensusReport, carrierIndex, triggerIndex *int) (XSEmbeddedCarrier, error) {
	if len(report.EmbeddedCarriers) == 0 {
		return XSEmbeddedCarrier{}, fmt.Errorf("scenario has no embedded XS carrier")
	}
	if triggerIndex != nil {
		for _, carrier := range report.EmbeddedCarriers {
			if carrier.TriggerIndex == *triggerIndex {
				return carrier, nil
			}
		}
		return XSEmbeddedCarrier{}, fmt.Errorf("no embedded XS carrier on trigger %d", *triggerIndex)
	}
	idx := 0
	if carrierIndex != nil {
		idx = *carrierIndex
	}
	if idx < 0 || idx >= len(report.EmbeddedCarriers) {
		return XSEmbeddedCarrier{}, fmt.Errorf("carrier index %d out of range 0..%d", idx, len(report.EmbeddedCarriers)-1)
	}
	return report.EmbeddedCarriers[idx], nil
}

func safeDeployXSName(name string) (string, error) {
	name = normalizeXSFileName(strings.ReplaceAll(strings.TrimSpace(name), `\`, "/"))
	if name == "" {
		return "", fmt.Errorf("XS deploy name is empty")
	}
	clean := filepath.ToSlash(filepath.Clean(name))
	if strings.HasPrefix(clean, "../") || clean == ".." || filepath.IsAbs(clean) {
		return "", fmt.Errorf("unsafe XS deploy name %q", name)
	}
	if !strings.HasSuffix(strings.ToLower(clean), ".xs") {
		clean += ".xs"
	}
	return clean, nil
}

func normalizeXSFileContent(content string) string {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")
	content = strings.TrimRight(content, "\n")
	if content == "" {
		return ""
	}
	return content + "\n"
}

func (f *File) xsScriptCalls() []XSScriptCall {
	if f == nil || f.Triggers == nil {
		return nil
	}
	var out []XSScriptCall
	for _, trigger := range f.Triggers.Triggers {
		for _, effect := range trigger.EffectData {
			if effect.Type != 55 {
				continue
			}
			if isXSCarrierMessage(effect.Text) {
				continue
			}
			out = append(out, XSScriptCall{
				TriggerIndex: trigger.Index,
				TriggerName:  trigger.Name,
				EffectIndex:  effect.EffectIndex,
				Message:      effect.Text,
				Calls:        extractXSCallNames(effect.Text),
			})
		}
	}
	return out
}

func (f *File) xsEmbeddedCarriers() []XSEmbeddedCarrier {
	if f == nil || f.Triggers == nil {
		return nil
	}
	var out []XSEmbeddedCarrier
	for _, trigger := range f.Triggers.Triggers {
		for _, effect := range trigger.EffectData {
			if effect.Type != 55 || !isXSCarrierMessage(effect.Text) {
				continue
			}
			sum := sha256.Sum256([]byte(effect.Text))
			content := XSCarrierContent(effect.Text)
			contentSum := sha256.Sum256([]byte(content))
			out = append(out, XSEmbeddedCarrier{
				TriggerIndex:  trigger.Index,
				TriggerName:   trigger.Name,
				EffectIndex:   effect.EffectIndex,
				Title:         xsCarrierTitle(effect.Text),
				SourceBytes:   len([]byte(effect.Text)),
				SourceSHA256:  hex.EncodeToString(sum[:]),
				ContentBytes:  len([]byte(content)),
				ContentSHA256: hex.EncodeToString(contentSum[:]),
				Source:        effect.Text,
				Content:       content,
			})
		}
	}
	return out
}

func XSCarrierContent(message string) string {
	if !isXSCarrierMessage(message) {
		return message
	}
	idx := strings.IndexByte(message, '\n')
	if idx < 0 || idx+1 >= len(message) {
		return ""
	}
	content := strings.TrimRight(message[idx+1:], "\r\n")
	if content == "" {
		return ""
	}
	return content + "\n"
}

func isXSCarrierMessage(message string) bool {
	if !strings.Contains(message, "\n") || !xsCarrierTitleRE.MatchString(message) {
		return false
	}
	clean := strings.ToLower(message)
	return strings.Contains(clean, "\nvoid ") ||
		strings.Contains(clean, "\nrule ") ||
		strings.Contains(clean, "\ninclude ") ||
		strings.Contains(clean, "\nconst ") ||
		strings.Contains(clean, "\nextern ")
}

func xsCarrierTitle(message string) string {
	match := xsCarrierTitleRE.FindStringSubmatch(message)
	if match == nil {
		return ""
	}
	return strings.TrimSpace(match[1])
}

func sanitizeXSCarrierTitle(title string) string {
	title = strings.TrimSpace(title)
	var b strings.Builder
	for _, r := range title {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '_' || r == '-' || r == '.':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	out := strings.Trim(b.String(), "_")
	if out == "" {
		return "xs"
	}
	if len(out) > 48 {
		out = out[:48]
	}
	return out
}

func (f *File) xsEmbeddedContent() string {
	if f == nil || f.root == nil {
		return ""
	}
	if files := f.root.section("Files"); files != nil {
		content, _ := files.stringValue("script_file_content")
		return content
	}
	return ""
}

func (f *File) XSAttachmentContent() string {
	return f.xsEmbeddedContent()
}

func extractXSCallNames(message string) []string {
	seen := map[string]bool{}
	var out []string
	for _, match := range xsCallRE.FindAllStringSubmatch(message, -1) {
		name := match[1]
		if seen[name] {
			continue
		}
		seen[name] = true
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

func uniqueScriptCallNames(calls []XSScriptCall) []string {
	seen := map[string]bool{}
	var out []string
	for _, call := range calls {
		for _, name := range call.Calls {
			if seen[name] {
				continue
			}
			seen[name] = true
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out
}

func xsFunctionsFromAnalysis(analysis xsauthor.AnalysisReport, sources []xsauthor.SourceFile) []xsauthor.XSFunction {
	reachable := map[string]bool{}
	for _, file := range analysis.Files {
		reachable[filepath.ToSlash(file)] = true
	}
	var out []xsauthor.XSFunction
	for _, source := range sources {
		path := filepath.ToSlash(source.Path)
		if len(reachable) > 0 && !reachable[path] {
			continue
		}
		out = append(out, xsauthor.ScanFunctionSource(path, source.Content)...)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].File == out[j].File {
			return out[i].Line < out[j].Line
		}
		return out[i].File < out[j].File
	})
	return out
}

func (r *XSCensusReport) finishXSCensusCounts() {
	if r.XS.ScriptFileContentBytes > 0 {
		r.ScriptContentEmbedded = true
		r.Counts.ScriptContentAttachments = 1
	}
	r.Counts.EmbeddedCarriers = len(r.EmbeddedCarriers)
	r.Counts.ScriptCalls = len(r.ScriptCalls)
	r.Counts.UniqueCalledFunctions = len(r.UniqueCalledFunctions)
	r.Counts.Functions = len(r.Functions)
	r.Counts.Findings = len(r.Findings)
	if r.Analysis == nil {
		return
	}
	r.Counts.Includes = len(r.Analysis.Includes)
	r.Counts.Declarations = len(r.Analysis.Declarations)
	r.Counts.CrossFileNonExtern = len(r.Analysis.CrossFileUses)
	for _, include := range r.Analysis.Includes {
		if include.Status == "resolved" {
			r.Counts.ResolvedIncludes++
		} else {
			r.Counts.MissingIncludes++
		}
	}
}
