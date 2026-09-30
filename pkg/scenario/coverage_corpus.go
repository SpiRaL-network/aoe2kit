package scenario

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
)

// CoverageCorpusReport summarizes dark-byte spans without retaining parsed
// scenarios. CoverageFile is called one file at a time so large scenario
// corpora remain bounded by the largest individual scenario.
type CoverageCorpusReport struct {
	Roots  []string              `json:"roots"`
	Files  int                   `json:"files"`
	Parsed int                   `json:"parsed"`
	Errors []CoverageCorpusError `json:"errors,omitempty"`
	Spans  []CoverageCorpusSpan  `json:"spans"`
}

type CoverageCorpusError struct {
	Path    string `json:"path"`
	Version string `json:"version,omitempty"`
	Status  string `json:"status"`
	Error   string `json:"error"`
}

type CoverageCorpusOptions struct {
	Strict bool
}

type CoverageCorpusSpan struct {
	Version           string   `json:"version"`
	Path              string   `json:"path"`
	Bytes             int      `json:"bytes"`
	Occurrences       int      `json:"occurrences"`
	Files             int      `json:"files"`
	SHA256            []string `json:"sha256"`
	NonZeroBytes      []int    `json:"nonzero_bytes"`
	MinTriggerCount   int      `json:"min_trigger_count"`
	MaxTriggerCount   int      `json:"max_trigger_count"`
	MinConditionCount int      `json:"min_condition_count"`
	MaxConditionCount int      `json:"max_condition_count"`
}

type coverageCorpusAccumulator struct {
	CoverageCorpusSpan
	sha256  map[string]bool
	nonZero map[int]bool
	files   map[string]bool
}

// CoverageCorpus walks roots recursively and analyzes every .aoe2scenario.
// A malformed or unsupported file is reported and does not abort the corpus.
func CoverageCorpus(roots ...string) (CoverageCorpusReport, error) {
	return CoverageCorpusWithOptions(CoverageCorpusOptions{}, roots...)
}

func CoverageCorpusWithOptions(opts CoverageCorpusOptions, roots ...string) (CoverageCorpusReport, error) {
	cleanRoots := make([]string, 0, len(roots))
	for _, root := range roots {
		if root != "" {
			cleanRoots = append(cleanRoots, root)
		}
	}
	if len(cleanRoots) == 0 {
		return CoverageCorpusReport{}, fmt.Errorf("coverage corpus needs at least one root")
	}
	sort.Strings(cleanRoots)
	report := CoverageCorpusReport{Roots: cleanRoots}
	acc := map[string]*coverageCorpusAccumulator{}
	paths := make([]string, 0)
	for _, root := range cleanRoots {
		err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() || filepath.Ext(entry.Name()) != ".aoe2scenario" {
				return nil
			}
			paths = append(paths, path)
			return nil
		})
		if err != nil {
			return report, err
		}
	}
	sort.Strings(paths)
	for _, path := range paths {
		report.Files++
		version, versionErr := ReadScenarioVersionFile(path)
		if versionErr != nil {
			report.Errors = append(report.Errors, CoverageCorpusError{Path: path, Version: version, Status: "error", Error: versionErr.Error()})
			continue
		}
		if !SupportsReadVersion(version) {
			report.Errors = append(report.Errors, CoverageCorpusError{Path: path, Version: version, Status: "unsupported", Error: unsupportedReadVersionError(version).Error()})
			continue
		}
		coverage, err := CoverageFile(path)
		if err != nil {
			report.Errors = append(report.Errors, CoverageCorpusError{Path: path, Version: version, Status: "error", Error: err.Error()})
			continue
		}
		report.Parsed++
		for _, section := range coverage.Sections {
			for _, span := range section.DarkSpans {
				pathKey := canonicalCoveragePath(span.Path)
				key := coverage.Version + "\x00" + pathKey + "\x00" + fmt.Sprint(span.Bytes)
				item := acc[key]
				if item == nil {
					item = &coverageCorpusAccumulator{
						CoverageCorpusSpan: CoverageCorpusSpan{Version: coverage.Version, Path: pathKey, Bytes: span.Bytes, MinTriggerCount: coverage.TriggerCount, MaxTriggerCount: coverage.TriggerCount, MinConditionCount: coverage.ConditionCount, MaxConditionCount: coverage.ConditionCount},
						sha256:             map[string]bool{}, nonZero: map[int]bool{}, files: map[string]bool{},
					}
					acc[key] = item
				}
				item.Occurrences++
				item.files[path] = true
				item.sha256[span.SHA256] = true
				item.nonZero[span.NonZeroBytes] = true
				item.MinTriggerCount = minInt(item.MinTriggerCount, coverage.TriggerCount)
				item.MaxTriggerCount = maxInt(item.MaxTriggerCount, coverage.TriggerCount)
				item.MinConditionCount = minInt(item.MinConditionCount, coverage.ConditionCount)
				item.MaxConditionCount = maxInt(item.MaxConditionCount, coverage.ConditionCount)
			}
		}
	}
	for _, item := range acc {
		item.Files = len(item.files)
		for value := range item.sha256 {
			item.SHA256 = append(item.SHA256, value)
		}
		for value := range item.nonZero {
			item.NonZeroBytes = append(item.NonZeroBytes, value)
		}
		sort.Strings(item.SHA256)
		sort.Ints(item.NonZeroBytes)
		report.Spans = append(report.Spans, item.CoverageCorpusSpan)
	}
	sort.Slice(report.Spans, func(i, j int) bool {
		if report.Spans[i].Version != report.Spans[j].Version {
			return report.Spans[i].Version < report.Spans[j].Version
		}
		if report.Spans[i].Path != report.Spans[j].Path {
			return report.Spans[i].Path < report.Spans[j].Path
		}
		return report.Spans[i].Bytes < report.Spans[j].Bytes
	})
	if report.Files > 0 && report.Parsed == 0 {
		return report, fmt.Errorf("all %d scenario files failed", report.Files)
	}
	if opts.Strict && len(report.Errors) > 0 {
		return report, fmt.Errorf("%d of %d scenario files failed", len(report.Errors), report.Files)
	}
	return report, nil
}

func canonicalCoveragePath(path string) string {
	var out strings.Builder
	for i := 0; i < len(path); {
		if path[i] != '[' {
			out.WriteByte(path[i])
			i++
			continue
		}
		end := strings.IndexByte(path[i:], ']')
		if end < 0 {
			out.WriteString(path[i:])
			break
		}
		end += i
		allDigits := end > i+1
		for j := i + 1; j < end && allDigits; j++ {
			allDigits = path[j] >= '0' && path[j] <= '9'
		}
		if allDigits {
			out.WriteString("[]")
			i = end + 1
		} else {
			out.WriteString(path[i : end+1])
			i = end + 1
		}
	}
	return out.String()
}
