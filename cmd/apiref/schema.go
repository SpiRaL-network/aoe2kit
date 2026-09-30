package main

// Static schema extraction. Where the probe captures only the top-level keys of
// one sample response, this reads the source and emits the complete nested
// shape of every report type: field names, JSON names, Go types, optionality,
// and cross-references between types. It runs in seconds and executes nothing.

import (
	"context"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"
)

type FieldSchema struct {
	Name     string `json:"name"`
	JSON     string `json:"json,omitempty"`
	GoType   string `json:"go_type"`
	Ref      string `json:"ref,omitempty"`
	Optional bool   `json:"optional,omitempty"`
	Embedded bool   `json:"embedded,omitempty"`
	Doc      string `json:"doc,omitempty"`
}

type TypeSchema struct {
	Package string        `json:"package"`
	Name    string        `json:"name"`
	Doc     string        `json:"doc,omitempty"`
	Fields  []FieldSchema `json:"fields"`
}

type SchemaCatalog struct {
	Note  string                `json:"note"`
	Count int                   `json:"type_count"`
	Types map[string]TypeSchema `json:"types"`
}

// ExtractSchemas walks the package tree and returns every exported struct type
// keyed as "package.TypeName".
func ExtractSchemas(roots ...string) (map[string]TypeSchema, error) {
	out := map[string]TypeSchema{}
	fset := token.NewFileSet()
	for _, root := range roots {
		err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if info.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			file, parseErr := parser.ParseFile(fset, path, nil, parser.ParseComments)
			if parseErr != nil {
				return nil
			}
			pkg := file.Name.Name
			for _, decl := range file.Decls {
				gen, ok := decl.(*ast.GenDecl)
				if !ok || gen.Tok != token.TYPE {
					continue
				}
				for _, spec := range gen.Specs {
					ts, ok := spec.(*ast.TypeSpec)
					if !ok || !ts.Name.IsExported() {
						continue
					}
					st, ok := ts.Type.(*ast.StructType)
					if !ok {
						continue
					}
					schema := TypeSchema{
						Package: pkg,
						Name:    ts.Name.Name,
						Doc:     firstSentence(docText(gen.Doc, ts.Doc)),
					}
					for _, field := range st.Fields.List {
						schema.Fields = append(schema.Fields, fieldSchemas(field, pkg)...)
					}
					out[pkg+"."+ts.Name.Name] = schema
				}
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	resolveRefs(out)
	return out, nil
}

func fieldSchemas(field *ast.Field, pkg string) []FieldSchema {
	goType := typeString(field.Type)
	jsonName, optional := jsonTag(field)
	doc := firstSentence(docText(field.Doc, nil))

	if len(field.Names) == 0 { // embedded
		return []FieldSchema{{
			Name: lastSegment(goType), GoType: goType, JSON: jsonName,
			Optional: optional, Embedded: true, Doc: doc,
			Ref: refCandidate(field.Type, pkg),
		}}
	}
	var out []FieldSchema
	for _, name := range field.Names {
		if !name.IsExported() {
			continue
		}
		out = append(out, FieldSchema{
			Name: name.Name, JSON: jsonName, GoType: goType,
			Optional: optional, Doc: doc, Ref: refCandidate(field.Type, pkg),
		})
	}
	return out
}

func jsonTag(field *ast.Field) (string, bool) {
	if field.Tag == nil {
		return "", false
	}
	tag := reflect.StructTag(strings.Trim(field.Tag.Value, "`"))
	value := tag.Get("json")
	if value == "" {
		return "", false
	}
	parts := strings.Split(value, ",")
	optional := false
	for _, p := range parts[1:] {
		if p == "omitempty" {
			optional = true
		}
	}
	return parts[0], optional
}

// refCandidate unwraps pointers, slices, and maps to name the underlying type.
func refCandidate(expr ast.Expr, pkg string) string {
	switch t := expr.(type) {
	case *ast.StarExpr:
		return refCandidate(t.X, pkg)
	case *ast.ArrayType:
		return refCandidate(t.Elt, pkg)
	case *ast.MapType:
		return refCandidate(t.Value, pkg)
	case *ast.Ident:
		if t.IsExported() {
			return pkg + "." + t.Name
		}
	case *ast.SelectorExpr:
		if base, ok := t.X.(*ast.Ident); ok {
			return base.Name + "." + t.Sel.Name
		}
	}
	return ""
}

// resolveRefs clears references that point at types outside this catalog, so a
// ref in the output always resolves.
func resolveRefs(types map[string]TypeSchema) {
	for key, schema := range types {
		for i, f := range schema.Fields {
			if f.Ref == "" {
				continue
			}
			if _, ok := types[f.Ref]; !ok {
				schema.Fields[i].Ref = ""
			}
		}
		types[key] = schema
	}
}

func typeString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + typeString(t.X)
	case *ast.ArrayType:
		return "[]" + typeString(t.Elt)
	case *ast.MapType:
		return "map[" + typeString(t.Key) + "]" + typeString(t.Value)
	case *ast.SelectorExpr:
		return typeString(t.X) + "." + t.Sel.Name
	case *ast.InterfaceType:
		return "any"
	case *ast.StructType:
		return "struct{...}"
	case *ast.FuncType:
		return "func(...)"
	case *ast.Ellipsis:
		return "..." + typeString(t.Elt)
	}
	return "unknown"
}

func docText(groups ...*ast.CommentGroup) string {
	for _, g := range groups {
		if g != nil && strings.TrimSpace(g.Text()) != "" {
			return strings.TrimSpace(g.Text())
		}
	}
	return ""
}

func firstSentence(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if idx := strings.Index(s, ". "); idx > 0 {
		s = s[:idx+1]
	}
	if len(s) > 240 {
		s = s[:240] + "..."
	}
	return strings.TrimSpace(s)
}

func lastSegment(s string) string {
	s = strings.TrimLeft(s, "*[]")
	if idx := strings.LastIndex(s, "."); idx >= 0 {
		return s[idx+1:]
	}
	return s
}

func CommandResponseTypeHint(commandName string, types map[string]TypeSchema) string {
	hints := map[string]string{
		"roadmap crud": "roadmap.CRUDReport",
		"roadmap dark": "roadmap.DarkReport",
		"roadmap rwd":  "roadmap.MatrixReport",
	}
	hint := hints[commandName]
	if hint == "" {
		return ""
	}
	if _, ok := types[hint]; !ok {
		return ""
	}
	return hint
}

// MatchResponseType finds the schema whose top-level JSON names best cover the
// keys a command actually returned. This is the join between the two methods:
// the probe says which type a command emits, the source says its full shape.
func MatchResponseType(outputKeys []string, types map[string]TypeSchema) string {
	if len(outputKeys) == 0 {
		return ""
	}
	want := map[string]bool{}
	for _, k := range outputKeys {
		want[k] = true
	}
	typeNames := make([]string, 0, len(types))
	for name := range types {
		typeNames = append(typeNames, name)
	}
	sort.Strings(typeNames)
	best, bestScore, bestFields := "", 0.0, 0
	for _, name := range typeNames {
		schema := types[name]
		have := map[string]bool{}
		requiredMissing := 0
		for _, f := range schema.Fields {
			if f.JSON != "" && f.JSON != "-" {
				have[f.JSON] = true
				if !f.Optional && !want[f.JSON] {
					requiredMissing++
				}
			}
		}
		if len(have) == 0 {
			continue
		}
		covered := 0
		for k := range want {
			if have[k] {
				covered++
			}
		}
		if covered == 0 {
			continue
		}
		// Reward covering the observed keys; penalize types far larger than
		// needed, especially when required fields are absent from the sample.
		score := float64(covered)/float64(len(want)) -
			0.02*float64(len(have)-covered) -
			0.12*float64(requiredMissing)
		if score > bestScore || (score == bestScore && len(have) < bestFields) {
			best, bestScore, bestFields = name, score, len(have)
		}
	}
	if bestScore < 0.75 {
		return ""
	}
	return best
}

// WriteGoDoc renders the package-level Go API using the toolchain's own
// documentation, rather than reimplementing it.
func WriteGoDoc(outPath string) error {
	list, err := exec.Command("go", "list", "./...").Output()
	if err != nil {
		return err
	}
	var b strings.Builder
	b.WriteString("# AoE2Kit Package API\n\n")
	b.WriteString("Go package documentation, emitted by `go doc -all` for every package that\n")
	b.WriteString("ships in the public release (`release/public-files.json`). This is the\n")
	b.WriteString("library-level API for importing AoE2Kit directly; `API_REFERENCE.md` covers\n")
	b.WriteString("the command line, and `api_schemas.json` carries the full nested shape of\n")
	b.WriteString("every report type.\n\n")
	b.WriteString("Regenerate with:\n\n```sh\ngo run ./cmd/apiref\n```\n\n")

	public, err := publicPackageDirs("release/public-files.json")
	if err != nil {
		return err
	}
	packages := strings.Fields(string(list))
	sort.Strings(packages)
	for _, pkg := range packages {
		// Workshop-only packages (tools/, private experiments) must not leak into the
		// public API docs: document a package only if the release ships its files.
		if !public[strings.TrimPrefix(strings.TrimPrefix(pkg, modulePath(packages)), "/")] {
			continue
		}
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		out, docErr := exec.CommandContext(ctx, "go", "doc", "-all", pkg).Output()
		cancel()
		if docErr != nil || len(out) == 0 {
			continue
		}
		fmt.Fprintf(&b, "## %s\n\n```go\n%s\n```\n\n", pkg, strings.TrimSpace(string(out)))
	}
	return os.WriteFile(outPath, []byte(b.String()), 0o644)
}

// publicPackageDirs returns the set of directories (relative to the module root)
// that contain at least one Go file listed in the public release allowlist.
func publicPackageDirs(allowlistPath string) (map[string]bool, error) {
	raw, err := os.ReadFile(allowlistPath)
	if err != nil {
		return nil, fmt.Errorf("read public allowlist: %w", err)
	}
	var doc struct {
		Files []string `json:"files"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil || len(doc.Files) == 0 {
		var list []string
		if err2 := json.Unmarshal(raw, &list); err2 != nil {
			return nil, fmt.Errorf("parse public allowlist %s: expected {\"files\": [...]} or [...]", allowlistPath)
		}
		doc.Files = list
	}
	dirs := map[string]bool{}
	for _, f := range doc.Files {
		if strings.HasSuffix(f, ".go") {
			dirs[filepath.ToSlash(filepath.Dir(f))] = true
		}
	}
	return dirs, nil
}

// modulePath is the shortest import path in the list (the module root package
// or the common prefix every package shares).
func modulePath(packages []string) string {
	if len(packages) == 0 {
		return ""
	}
	root := packages[0]
	for _, p := range packages[1:] {
		for !strings.HasPrefix(p+"/", root+"/") && strings.Contains(root, "/") {
			root = root[:strings.LastIndex(root, "/")]
		}
	}
	if i := strings.Index(root, "/"); i >= 0 {
		return root[:i]
	}
	return root
}
