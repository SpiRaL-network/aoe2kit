package xs

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type SourceFile struct {
	Path    string
	Content string
}

type AnalyzeOptions struct {
	EntryPath          string
	Files              []SourceFile
	CheckCreateResults bool
	DatPath            string
}

type IncludeRef struct {
	From    string `json:"from"`
	Line    int    `json:"line"`
	Path    string `json:"path"`
	Target  string `json:"target,omitempty"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

type SymbolDecl struct {
	Name   string `json:"name"`
	Kind   string `json:"kind"`
	Type   string `json:"type"`
	Extern bool   `json:"extern"`
	File   string `json:"file"`
	Line   int    `json:"line"`
}

type CrossFileUse struct {
	Name        string `json:"name"`
	DeclaredIn  string `json:"declared_in"`
	DeclaredAt  int    `json:"declared_at"`
	UsedIn      string `json:"used_in"`
	UsedAt      int    `json:"used_at"`
	Finding     string `json:"finding"`
	Recommended string `json:"recommended_fix"`
}

type AnalysisReport struct {
	Findings      []SourceFinding `json:"findings,omitempty"`
	Entry         string          `json:"entry,omitempty"`
	Files         []string        `json:"files"`
	Includes      []IncludeRef    `json:"includes,omitempty"`
	Declarations  []SymbolDecl    `json:"declarations,omitempty"`
	CrossFileUses []CrossFileUse  `json:"cross_file_uses,omitempty"`
	Warnings      []string        `json:"warnings,omitempty"`
}

var (
	xsIncludeRE = regexp.MustCompile(`\binclude\s+"([^"]+)"\s*;`)
	xsDeclRE    = regexp.MustCompile(`^\s*(extern\s+)?(const\s+)?(int|float|bool|string|vector)\s+([A-Za-z_][A-Za-z0-9_]*)\b`)
	xsTokenRE   = regexp.MustCompile(`[A-Za-z_][A-Za-z0-9_]*`)
)

func AnalyzeFiles(opts AnalyzeOptions) (AnalysisReport, error) {
	report := AnalysisReport{Entry: opts.EntryPath}
	contents := map[string]string{}
	for _, file := range opts.Files {
		path := normalizeXSPath(file.Path)
		contents[path] = file.Content
	}
	if opts.EntryPath != "" {
		opts.EntryPath = normalizeXSPath(opts.EntryPath)
	}
	reachable := reachableXSFiles(opts.EntryPath, contents, &report)
	report.Files = sortedXSKeys(reachable)
	decls := map[string][]SymbolDecl{}
	tokenLines := map[string]map[string][]int{}
	for _, path := range report.Files {
		content := contents[path]
		lintOpts := SourceLintOptions{CheckCreateResults: opts.CheckCreateResults, DatPath: opts.DatPath}
		report.Findings = append(report.Findings, LintSourceWithOptions(path, content, lintOpts)...)
		fileDecls, fileTokens := scanXSSource(path, content)
		for _, decl := range fileDecls {
			report.Declarations = append(report.Declarations, decl)
			decls[decl.Name] = append(decls[decl.Name], decl)
		}
		tokenLines[path] = fileTokens
	}
	sort.Slice(report.Declarations, func(i, j int) bool {
		if report.Declarations[i].File == report.Declarations[j].File {
			return report.Declarations[i].Line < report.Declarations[j].Line
		}
		return report.Declarations[i].File < report.Declarations[j].File
	})
	report.CrossFileUses = crossFileNonExternUses(decls, tokenLines)
	return report, nil
}

func LoadSourceTree(entryPath string) ([]SourceFile, error) {
	entryPath = filepath.Clean(entryPath)
	root := filepath.Dir(entryPath)
	var files []SourceFile
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
		files = append(files, SourceFile{Path: rel, Content: string(data)})
		return nil
	})
	return files, err
}

func ResolveInclude(fromPath, includePath string, contents map[string]string) (string, bool) {
	includePath = strings.ReplaceAll(includePath, `\`, "/")
	var candidates []string
	if fromPath != "" {
		candidates = append(candidates, normalizeXSPath(filepath.Join(filepath.Dir(fromPath), includePath)))
	}
	candidates = append(candidates, normalizeXSPath(includePath))
	candidates = append(candidates, normalizeXSPath(filepath.Base(includePath)))
	for _, candidate := range candidates {
		if _, ok := contents[candidate]; ok {
			return candidate, true
		}
	}
	return "", false
}

func reachableXSFiles(entry string, contents map[string]string, report *AnalysisReport) map[string]bool {
	reachable := map[string]bool{}
	visiting := map[string]bool{}
	var walk func(string)
	walk = func(path string) {
		if path == "" {
			for file := range contents {
				reachable[file] = true
			}
			return
		}
		if visiting[path] {
			report.Warnings = append(report.Warnings, "include cycle detected at "+path)
			return
		}
		if reachable[path] {
			return
		}
		content, ok := contents[path]
		if !ok {
			report.Warnings = append(report.Warnings, "entry XS file not loaded: "+path)
			return
		}
		visiting[path] = true
		reachable[path] = true
		for _, inc := range scanIncludes(path, content, contents) {
			report.Includes = append(report.Includes, inc)
			if inc.Status == "resolved" {
				walk(inc.Target)
			}
		}
		visiting[path] = false
	}
	walk(entry)
	sort.Slice(report.Includes, func(i, j int) bool {
		if report.Includes[i].From == report.Includes[j].From {
			return report.Includes[i].Line < report.Includes[j].Line
		}
		return report.Includes[i].From < report.Includes[j].From
	})
	return reachable
}

func scanIncludes(path, content string, contents map[string]string) []IncludeRef {
	var out []IncludeRef
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		clean := stripLineComment(line)
		for _, match := range xsIncludeRE.FindAllStringSubmatch(clean, -1) {
			inc := IncludeRef{From: path, Line: i + 1, Path: match[1], Status: "missing", Message: "include target not found"}
			if target, ok := ResolveInclude(path, match[1], contents); ok {
				inc.Target = target
				inc.Status = "resolved"
				inc.Message = ""
			}
			out = append(out, inc)
		}
	}
	return out
}

func scanXSSource(path, content string) ([]SymbolDecl, map[string][]int) {
	var decls []SymbolDecl
	tokens := map[string][]int{}
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		clean := stripLineComment(line)
		clean = stripXSStrings(clean)
		if match := xsDeclRE.FindStringSubmatch(clean); match != nil {
			kind := "variable"
			if strings.TrimSpace(match[2]) != "" {
				kind = "constant"
			}
			decls = append(decls, SymbolDecl{
				Name:   match[4],
				Kind:   kind,
				Type:   match[3],
				Extern: strings.TrimSpace(match[1]) != "",
				File:   path,
				Line:   i + 1,
			})
		}
		for _, token := range xsTokenRE.FindAllString(clean, -1) {
			if isXSKeyword(token) || strings.HasPrefix(token, "xs") {
				continue
			}
			tokens[token] = append(tokens[token], i+1)
		}
	}
	return decls, tokens
}

func crossFileNonExternUses(decls map[string][]SymbolDecl, tokenLines map[string]map[string][]int) []CrossFileUse {
	seen := map[string]bool{}
	var out []CrossFileUse
	for name, nameDecls := range decls {
		for _, decl := range nameDecls {
			if decl.Extern {
				continue
			}
			for file, tokens := range tokenLines {
				if file == decl.File {
					continue
				}
				lines := tokens[name]
				if len(lines) == 0 {
					continue
				}
				key := fmt.Sprintf("%s\x00%s\x00%d\x00%s", name, decl.File, decl.Line, file)
				if seen[key] {
					continue
				}
				seen[key] = true
				out = append(out, CrossFileUse{
					Name:        name,
					DeclaredIn:  decl.File,
					DeclaredAt:  decl.Line,
					UsedIn:      file,
					UsedAt:      lines[0],
					Finding:     "symbol is declared without extern in one XS module and referenced from another",
					Recommended: "declare shared XS constants/variables with extern, or keep the use in the declaring file",
				})
			}
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Name == out[j].Name {
			if out[i].DeclaredIn == out[j].DeclaredIn {
				return out[i].UsedIn < out[j].UsedIn
			}
			return out[i].DeclaredIn < out[j].DeclaredIn
		}
		return out[i].Name < out[j].Name
	})
	return out
}

func stripXSStrings(line string) string {
	var b strings.Builder
	inString := false
	escaped := false
	for _, r := range line {
		if inString {
			if escaped {
				escaped = false
				continue
			}
			if r == '\\' {
				escaped = true
				continue
			}
			if r == '"' {
				inString = false
			}
			continue
		}
		if r == '"' {
			inString = true
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func normalizeXSPath(path string) string {
	path = strings.ReplaceAll(filepath.Clean(path), `\`, "/")
	if path == "." {
		return ""
	}
	return strings.TrimPrefix(path, "./")
}

func sortedXSKeys(values map[string]bool) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func isXSKeyword(token string) bool {
	switch token {
	case "include", "extern", "const", "int", "float", "bool", "string", "vector", "void",
		"true", "false", "return", "if", "else", "for", "while", "do", "break", "continue":
		return true
	default:
		return false
	}
}
