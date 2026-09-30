package xs

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type ShimOptions struct {
	Paths             []string
	Prefix            string
	Enabled           bool
	IncludeParamFuncs bool
}

type XSFunction struct {
	Name       string `json:"name"`
	ReturnType string `json:"return_type"`
	Params     string `json:"params"`
	File       string `json:"file"`
	Line       int    `json:"line"`
	Emitted    bool   `json:"emitted"`
	SkipReason string `json:"skip_reason,omitempty"`
}

type ShimReport struct {
	Verification VerificationClaim `json:"verification"`
	Functions    []XSFunction      `json:"functions"`
	Emitted      int               `json:"emitted"`
	Skipped      int               `json:"skipped"`
	Recipe       map[string]any    `json:"recipe"`
}

func GenerateShims(opts ShimOptions) (ShimReport, error) {
	if len(opts.Paths) == 0 {
		return ShimReport{}, fmt.Errorf("at least one XS file or directory is required")
	}
	if opts.Prefix == "" {
		opts.Prefix = "XS Shim"
	}
	functions, err := scanFunctions(opts.Paths)
	if err != nil {
		return ShimReport{}, err
	}
	triggers := make([]map[string]any, 0)
	for i := range functions {
		fn := &functions[i]
		if fn.Params != "" && !opts.IncludeParamFuncs {
			fn.Emitted = false
			fn.SkipReason = "has parameters; script_call shims are parameterless by default"
			continue
		}
		fn.Emitted = true
		triggers = append(triggers, map[string]any{
			"op":      "add_trigger",
			"name":    fmt.Sprintf("%s %s", opts.Prefix, fn.Name),
			"enabled": opts.Enabled,
			"looping": false,
			"effects": []map[string]any{
				{"op": "script_call", "message": fn.Name + "();"},
			},
		})
	}
	report := ShimReport{
		Verification: StructureVerifiedClaim("Shim recipe emits trigger script_call effects using the message field; engine execution depends on the scenario's XS runtime being present."),
		Functions:    functions,
		Recipe:       map[string]any{"triggers": triggers},
	}
	for _, fn := range functions {
		if fn.Emitted {
			report.Emitted++
		} else {
			report.Skipped++
		}
	}
	return report, nil
}

func WriteShimRecipe(report ShimReport, output string) error {
	if output == "" {
		return nil
	}
	return writeJSONFile(output, report.Recipe)
}

var functionRE = regexp.MustCompile(`^\s*(void|int|float|bool|string|vector)\s+([A-Za-z_][A-Za-z0-9_]*)\s*\(([^)]*)\)\s*\{?`)

func ScanFunctions(paths []string) ([]XSFunction, error) {
	return scanFunctions(paths)
}

func ScanFunctionSource(path, content string) []XSFunction {
	out := []XSFunction{}
	scanner := bufio.NewScanner(strings.NewReader(content))
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := stripLineComment(scanner.Text())
		match := functionRE.FindStringSubmatch(line)
		if match == nil {
			continue
		}
		name := match[2]
		if isControlWord(name) {
			continue
		}
		if err := validateIdentifier(name); err != nil {
			continue
		}
		params := strings.TrimSpace(match[3])
		if params == "void" {
			params = ""
		}
		out = append(out, XSFunction{Name: name, ReturnType: match[1], Params: params, File: path, Line: lineNo})
	}
	return out
}

func scanFunctions(paths []string) ([]XSFunction, error) {
	seen := map[string]bool{}
	out := []XSFunction{}
	for _, root := range paths {
		info, err := os.Stat(root)
		if err != nil {
			return nil, err
		}
		if info.IsDir() {
			err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if d.IsDir() || strings.ToLower(filepath.Ext(path)) != ".xs" {
					return nil
				}
				funcs, err := scanFunctionFile(path)
				if err != nil {
					return err
				}
				for _, fn := range funcs {
					key := fn.Name + "\x00" + fn.Params
					if !seen[key] {
						seen[key] = true
						out = append(out, fn)
					}
				}
				return nil
			})
			if err != nil {
				return nil, err
			}
			continue
		}
		funcs, err := scanFunctionFile(root)
		if err != nil {
			return nil, err
		}
		for _, fn := range funcs {
			key := fn.Name + "\x00" + fn.Params
			if !seen[key] {
				seen[key] = true
				out = append(out, fn)
			}
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Name == out[j].Name {
			return out[i].File < out[j].File
		}
		return out[i].Name < out[j].Name
	})
	return out, nil
}

func scanFunctionFile(path string) ([]XSFunction, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ScanFunctionSource(path, string(data)), nil
}

func stripLineComment(line string) string {
	if idx := strings.Index(line, "//"); idx >= 0 {
		return line[:idx]
	}
	return line
}

func isControlWord(name string) bool {
	switch name {
	case "if", "for", "while", "switch":
		return true
	default:
		return false
	}
}
