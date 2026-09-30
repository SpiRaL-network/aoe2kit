package replay

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"aoe2kit/pkg/aoe2"
)

const PromisoryRootEnv = "AOE2KIT_PROMISORY_ROOT"

type AIManifestOptions struct {
	PromisoryRoot string
}

type AIManifestReport struct {
	Path            string                 `json:"path,omitempty"`
	Method          string                 `json:"method"`
	Verification    aoe2.VerificationClaim `json:"verification"`
	Summary         AIManifestSummary      `json:"summary"`
	PromisoryRoot   string                 `json:"promisory_root,omitempty"`
	PromisoryRootOK bool                   `json:"promisory_root_ok"`
	PromiDEMentions []AIManifestMention    `json:"promide_mentions,omitempty"`
	References      []AIManifestReference  `json:"references,omitempty"`
	XSIncludes      []AIManifestXSInclude  `json:"xs_includes,omitempty"`
	UniqueModules   []AIManifestModule     `json:"unique_modules,omitempty"`
	Warnings        []string               `json:"warnings,omitempty"`
}

type AIManifestSummary struct {
	PromiDEMentions     int `json:"promide_mentions"`
	LoadDirectives      int `json:"load_directives"`
	XSIncludes          int `json:"xs_includes"`
	UniqueModules       int `json:"unique_modules"`
	ResolvedModules     int `json:"resolved_modules"`
	MissingModules      int `json:"missing_modules"`
	HeaderOpaqueBytes   int `json:"header_opaque_bytes"`
	HeaderDecodedBytes  int `json:"header_decoded_bytes"`
	InflatedHeaderBytes int `json:"inflated_header_bytes"`
}

type AIManifestMention struct {
	Space  string `json:"space"`
	Offset int    `json:"offset"`
	End    int    `json:"end"`
	Region string `json:"region,omitempty"`
	Status string `json:"status,omitempty"`
}

type AIManifestReference struct {
	ModulePath   string `json:"module_path"`
	ModuleName   string `json:"module_name"`
	Directive    string `json:"directive"`
	Space        string `json:"space"`
	Start        int    `json:"start"`
	End          int    `json:"end"`
	Region       string `json:"region"`
	SourceRegion string `json:"source_region,omitempty"`
	ResolvedPath string `json:"resolved_path,omitempty"`
	Resolved     bool   `json:"resolved"`
}

type AIManifestXSInclude struct {
	IncludePath  string `json:"include_path"`
	IncludeName  string `json:"include_name"`
	Directive    string `json:"directive"`
	Space        string `json:"space"`
	Start        int    `json:"start"`
	End          int    `json:"end"`
	Region       string `json:"region"`
	SourceRegion string `json:"source_region,omitempty"`
}

type AIManifestModule struct {
	Name         string `json:"name"`
	Count        int    `json:"count"`
	ResolvedPath string `json:"resolved_path,omitempty"`
	Resolved     bool   `json:"resolved"`
}

func BuildAIManifest(path string, opts AIManifestOptions) (*AIManifestReport, error) {
	data, err := ReadRecordBytes(path)
	if err != nil {
		return nil, err
	}
	rec, err := Parse(data)
	if err != nil {
		return nil, err
	}
	coverage, err := BuildCoverage(path)
	if err != nil {
		return nil, err
	}
	root := strings.TrimSpace(opts.PromisoryRoot)
	if root == "" {
		root = strings.TrimSpace(os.Getenv(PromisoryRootEnv))
	}
	rootOK := false
	if root != "" {
		if st, err := os.Stat(root); err == nil && st.IsDir() {
			rootOK = true
		}
	}
	report := &AIManifestReport{
		Path:   path,
		Method: "v68_header_ai_script_module_manifest_scan",
		Verification: aoe2.VerificationClaim{
			Label:             "structure_verified_ai_load_directives_not_engine_verified",
			StructureVerified: true,
			EngineVerified:    false,
			Note:              "PromiDE/Promisory references are exact text directives carved from bounded opaque replay header regions; the surrounding binary AI/default-loadout wrapper is not decoded.",
		},
		PromisoryRoot:   root,
		PromisoryRootOK: rootOK,
		Summary: AIManifestSummary{
			HeaderOpaqueBytes:   coverage.Summary.HeaderOpaqueBytes,
			HeaderDecodedBytes:  coverage.Summary.HeaderDecodedBytes,
			InflatedHeaderBytes: coverage.Summary.InflatedHeaderBytes,
		},
		Warnings: append([]string(nil), coverage.Warnings...),
	}
	if root == "" {
		report.Warnings = append(report.Warnings, fmt.Sprintf("Promisory module resolution disabled; set %s to resolve default DE AI modules", PromisoryRootEnv))
	}
	index := indexRegions(coverage.Regions)
	header := rec.HeaderBytes()
	report.PromiDEMentions = promideMentions(header, index)
	report.Summary.PromiDEMentions = len(report.PromiDEMentions)
	modules := map[string]*AIManifestModule{}
	for _, region := range coverage.Regions {
		if region.Space != "inflated_header" || region.Confidence != "parsed_ai_load_directive_inside_bounded_opaque_header_region" {
			continue
		}
		modulePath := detailString(region.Details, "module_path")
		moduleName := detailString(region.Details, "module_name")
		directive := detailString(region.Details, "text")
		ref := AIManifestReference{
			ModulePath:   modulePath,
			ModuleName:   moduleName,
			Directive:    directive,
			Space:        region.Space,
			Start:        region.Start,
			End:          region.End,
			Region:       region.Name,
			SourceRegion: detailString(region.Details, "source_region"),
		}
		if rootOK {
			candidate := filepath.Join(root, moduleName+".per")
			if st, err := os.Stat(candidate); err == nil && !st.IsDir() {
				ref.Resolved = true
				ref.ResolvedPath = candidate
			}
		}
		report.References = append(report.References, ref)
		mod := modules[moduleName]
		if mod == nil {
			mod = &AIManifestModule{Name: moduleName, Resolved: ref.Resolved, ResolvedPath: ref.ResolvedPath}
			modules[moduleName] = mod
		}
		mod.Count++
		if ref.Resolved {
			mod.Resolved = true
			mod.ResolvedPath = ref.ResolvedPath
		}
	}
	for _, region := range coverage.Regions {
		if region.Space != "inflated_header" || region.Confidence != "parsed_xs_include_directive_inside_bounded_opaque_header_region" {
			continue
		}
		report.XSIncludes = append(report.XSIncludes, AIManifestXSInclude{
			IncludePath:  detailString(region.Details, "include_path"),
			IncludeName:  detailString(region.Details, "include_name"),
			Directive:    detailString(region.Details, "text"),
			Space:        region.Space,
			Start:        region.Start,
			End:          region.End,
			Region:       region.Name,
			SourceRegion: detailString(region.Details, "source_region"),
		})
	}
	sort.Slice(report.References, func(i, j int) bool {
		if report.References[i].Start == report.References[j].Start {
			return report.References[i].ModulePath < report.References[j].ModulePath
		}
		return report.References[i].Start < report.References[j].Start
	})
	sort.Slice(report.XSIncludes, func(i, j int) bool {
		if report.XSIncludes[i].Start == report.XSIncludes[j].Start {
			return report.XSIncludes[i].IncludePath < report.XSIncludes[j].IncludePath
		}
		return report.XSIncludes[i].Start < report.XSIncludes[j].Start
	})
	names := make([]string, 0, len(modules))
	for name := range modules {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		mod := *modules[name]
		report.UniqueModules = append(report.UniqueModules, mod)
		if mod.Resolved {
			report.Summary.ResolvedModules++
		} else {
			report.Summary.MissingModules++
		}
	}
	report.Summary.LoadDirectives = len(report.References)
	report.Summary.XSIncludes = len(report.XSIncludes)
	report.Summary.UniqueModules = len(report.UniqueModules)
	if len(report.Warnings) == 0 {
		report.Warnings = nil
	}
	return report, nil
}

func promideMentions(header []byte, index replayRegionIndex) []AIManifestMention {
	var out []AIManifestMention
	needle := []byte("PromiDE")
	for off := 0; off+len(needle) <= len(header); {
		idx := bytes.Index(header[off:], needle)
		if idx < 0 {
			break
		}
		start := off + idx
		end := start + len(needle)
		region := index.regionForRange("inflated_header", start, end)
		out = append(out, AIManifestMention{Space: "inflated_header", Offset: start, End: end, Region: replayRegionName(region), Status: replayRegionStatus(region)})
		off = end
	}
	return out
}

func detailString(details map[string]any, key string) string {
	if details == nil {
		return ""
	}
	value, _ := details[key].(string)
	return value
}

func (r AIManifestReference) String() string {
	if r.ResolvedPath != "" {
		return fmt.Sprintf("%s -> %s", r.ModulePath, r.ResolvedPath)
	}
	return r.ModulePath
}
