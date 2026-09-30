package scenario

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"aoe2kit/pkg/enginefacts"
	xsauthor "aoe2kit/pkg/xs"
)

type DeployCheckReport struct {
	Path         string                   `json:"path,omitempty"`
	DeployTree   string                   `json:"deploy_tree"`
	OK           bool                     `json:"ok"`
	Verification string                   `json:"verification"`
	XS           XSDeployAttachment       `json:"xs"`
	Findings     []DeployCheckFinding     `json:"findings,omitempty"`
	Analysis     *xsauthor.AnalysisReport `json:"analysis,omitempty"`
}

type XSDeployAttachment struct {
	ScriptName              string `json:"script_name,omitempty"`
	ScriptFilePath          string `json:"script_file_path,omitempty"`
	ScriptFileContentBytes  int    `json:"script_file_content_bytes,omitempty"`
	ScriptFileContentSHA256 string `json:"script_file_content_sha256,omitempty"`
	ResolvedPath            string `json:"resolved_path,omitempty"`
	ResolvedRelativePath    string `json:"resolved_relative_path,omitempty"`
}

type DeployCheckFinding struct {
	Severity     string `json:"severity"`
	What         string `json:"what"`
	Path         string `json:"path,omitempty"`
	Line         int    `json:"line,omitempty"`
	Fix          string `json:"fix"`
	FactID       string `json:"fact_id,omitempty"`
	FactTier     string `json:"fact_tier,omitempty"`
	VerifiedDate string `json:"verified_date,omitempty"`
	FixtureRef   string `json:"fixture_ref,omitempty"`
}

type DeployCheckOptions struct {
	IncludeProvisional bool
}

func DeployCheckFile(path, deployTree string) (DeployCheckReport, error) {
	return DeployCheckFileWithOptions(path, deployTree, DeployCheckOptions{})
}

func DeployCheckFileWithOptions(path, deployTree string, opts DeployCheckOptions) (DeployCheckReport, error) {
	file, err := Open(path)
	if err != nil {
		return DeployCheckReport{}, err
	}
	report, err := file.DeployCheckWithOptions(deployTree, opts)
	if err != nil {
		return DeployCheckReport{}, err
	}
	report.Path = path
	return report, nil
}

func (f *File) DeployCheck(deployTree string) (DeployCheckReport, error) {
	return f.DeployCheckWithOptions(deployTree, DeployCheckOptions{})
}

func (f *File) DeployCheckWithOptions(deployTree string, opts DeployCheckOptions) (DeployCheckReport, error) {
	if strings.TrimSpace(deployTree) == "" {
		return DeployCheckReport{}, fmt.Errorf("deploy tree is required")
	}
	attachment := f.XSAttachment()
	report := DeployCheckReport{
		DeployTree:   deployTree,
		Verification: "structure_verified_not_engine_verified",
		XS:           attachment,
	}
	if attachment.ScriptName == "" && attachment.ScriptFilePath == "" {
		report.OK = true
		return report, nil
	}
	if attachment.ScriptName == "" {
		report.addFindingFact("error", "Map.script_name is empty but Files.script_file_path is set", "", 0, "Set Map.script_name to the exact XS filename DE should open.", "xs.scenario_entry_filename_literal", opts)
	}
	if attachment.ScriptFilePath == "" {
		report.addFindingFact("error", "Files.script_file_path is empty but Map.script_name is set", "", 0, "Set Files.script_file_path to the same exact XS filename as Map.script_name.", "xs.scenario_entry_filename_literal", opts)
	}
	if attachment.ScriptName != "" && attachment.ScriptFilePath != "" && cleanXSName(attachment.ScriptName) != cleanXSName(attachment.ScriptFilePath) {
		report.addFindingFact("error", fmt.Sprintf("Map.script_name %q does not match Files.script_file_path %q", attachment.ScriptName, attachment.ScriptFilePath), "", 0, "Use the same literal filename in both fields, including the .xs extension.", "xs.scenario_entry_filename_literal", opts)
	}
	entryName := attachment.ScriptName
	if entryName == "" {
		entryName = attachment.ScriptFilePath
	}
	if entryName != "" && !strings.HasSuffix(strings.ToLower(entryName), ".xs") {
		report.addFindingFact("error", fmt.Sprintf("XS entry %q has no .xs extension", entryName), "", 0, "Use the literal .xs filename in Map.script_name and Files.script_file_path.", "xs.scenario_entry_filename_literal", opts)
	}
	resolved, root, rel, candidates := resolveDeployedXS(deployTree, entryName)
	if resolved == "" {
		report.addFindingFact("error", fmt.Sprintf("XS entry %q does not resolve under deploy tree", entryName), "", 0, "Copy the XS file into resources/_common/xs under the active mod/profile tree, or fix the scenario XS filename fields. Checked: "+strings.Join(candidates, ", "), "xs.scenario_entry_filename_literal", opts)
	} else {
		report.XS.ResolvedPath = resolved
		report.XS.ResolvedRelativePath = rel
		sources, err := loadDeployXSSources(root)
		if err != nil {
			report.addFinding("error", "failed to read XS deploy tree: "+err.Error(), root, 0, "Fix file permissions or deploy tree path.")
		} else {
			analysis, err := xsauthor.AnalyzeFiles(xsauthor.AnalyzeOptions{EntryPath: rel, Files: sources})
			if err != nil {
				return DeployCheckReport{}, err
			}
			report.Analysis = &analysis
			for _, include := range analysis.Includes {
				if include.Status != "resolved" {
					report.addFindingFact("error", fmt.Sprintf("XS include %q from %s does not resolve", include.Path, include.From), include.From, include.Line, "Deploy the included file beside the including module or fix the include path.", "xs.include_resolves_deployed_tree", opts)
				}
			}
			for _, warning := range analysis.Warnings {
				report.addFinding("warning", warning, "", 0, "Review the XS include graph.")
			}
			for _, use := range analysis.CrossFileUses {
				report.addFindingFact("error", fmt.Sprintf("XS symbol %q is declared without extern in %s and used from %s", use.Name, use.DeclaredIn, use.UsedIn), use.UsedIn, use.UsedAt, use.Recommended, "xs.cross_file_symbols_require_extern", opts)
			}
		}
	}
	report.OK = true
	for _, finding := range report.Findings {
		if finding.Severity == "error" {
			report.OK = false
			break
		}
	}
	sort.Slice(report.Findings, func(i, j int) bool {
		if report.Findings[i].Severity == report.Findings[j].Severity {
			return report.Findings[i].What < report.Findings[j].What
		}
		return report.Findings[i].Severity < report.Findings[j].Severity
	})
	return report, nil
}

func (f *File) XSAttachment() XSDeployAttachment {
	var out XSDeployAttachment
	if f == nil || f.root == nil {
		return out
	}
	if mapSection := f.root.section("Map"); mapSection != nil {
		out.ScriptName, _ = mapSection.stringValue("script_name")
	}
	if files := f.root.section("Files"); files != nil {
		out.ScriptFilePath, _ = files.stringValue("script_file_path")
		content, _ := files.stringValue("script_file_content")
		out.ScriptFileContentBytes = len(content)
		if content != "" {
			sum := sha256.Sum256([]byte(content))
			out.ScriptFileContentSHA256 = hex.EncodeToString(sum[:])
		}
	}
	return out
}

func (r *DeployCheckReport) addFinding(severity, what, path string, line int, fix string) {
	r.Findings = append(r.Findings, DeployCheckFinding{Severity: severity, What: what, Path: path, Line: line, Fix: fix})
}

func (r *DeployCheckReport) addFindingFact(severity, what, path string, line int, fix string, factID string, opts DeployCheckOptions) {
	finding := DeployCheckFinding{Severity: severity, What: what, Path: path, Line: line, Fix: fix}
	citation := enginefacts.CitationFor(factID, opts.IncludeProvisional)
	finding.FactID = citation.FactID
	finding.FactTier = citation.Tier
	finding.VerifiedDate = citation.VerifiedDate
	finding.FixtureRef = citation.FixtureRef
	r.Findings = append(r.Findings, finding)
}

func resolveDeployedXS(deployTree, entry string) (resolved, root, rel string, candidates []string) {
	entry = strings.TrimSpace(strings.ReplaceAll(entry, `\`, string(filepath.Separator)))
	if entry == "" {
		return "", "", "", nil
	}
	base := filepath.Clean(deployTree)
	roots := []string{
		base,
		filepath.Join(base, "resources", "_common", "xs"),
		filepath.Join(base, "resources", "_common", "scenario"),
		filepath.Join(base, "xs"),
	}
	seen := map[string]bool{}
	for _, candidateRoot := range roots {
		candidate := filepath.Clean(filepath.Join(candidateRoot, entry))
		if seen[candidate] {
			continue
		}
		seen[candidate] = true
		candidates = append(candidates, candidate)
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			rel, err := filepath.Rel(candidateRoot, candidate)
			if err != nil {
				rel = filepath.Base(candidate)
			}
			return candidate, candidateRoot, filepath.ToSlash(rel), candidates
		}
	}
	return "", "", "", candidates
}

func loadDeployXSSources(root string) ([]xsauthor.SourceFile, error) {
	var sources []xsauthor.SourceFile
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || strings.ToLower(filepath.Ext(path)) != ".xs" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			rel = filepath.Base(path)
		}
		sources = append(sources, xsauthor.SourceFile{Path: filepath.ToSlash(rel), Content: string(data)})
		return nil
	})
	return sources, err
}

func cleanXSName(name string) string {
	return filepath.ToSlash(filepath.Clean(strings.TrimSpace(strings.ReplaceAll(name, `\`, "/"))))
}
