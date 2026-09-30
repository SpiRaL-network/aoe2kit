package xs

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// WriteTypesDocument writes the sibling schema contract used by generated
// XS sidecars. Keeping this operation here prevents generators from silently
// drifting on empty or malformed schema documents.
func WriteTypesDocument(path, document string) error {
	if strings.TrimSpace(document) == "" {
		return fmt.Errorf("types document is empty")
	}
	if !strings.Contains(document, "header =") {
		return fmt.Errorf("types document %q has no header declaration", path)
	}
	return os.WriteFile(path, []byte(document), 0644)
}

// DiscoverSiblingTypesFile finds the schema belonging to an XS sidecar. A
// matching basename wins; otherwise discovery only succeeds when exactly one
// recognized schema remains. Ambiguous directories return no result rather
// than allowing an unrelated schema to misdecode the run.
func DiscoverSiblingTypesFile(input string) string {
	matching := strings.TrimSuffix(input, filepath.Ext(input)) + ".types"
	if _, err := os.Stat(matching); err == nil {
		return matching
	}
	entries, err := os.ReadDir(filepath.Dir(input))
	if err != nil {
		return ""
	}
	inputStem := normalizeTypesStem(strings.TrimSuffix(filepath.Base(input), filepath.Ext(input)))
	type candidate struct {
		path  string
		score int
	}
	var candidates []candidate
	for _, entry := range entries {
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".types") {
			continue
		}
		path := filepath.Join(filepath.Dir(input), entry.Name())
		document, readErr := os.ReadFile(path)
		if readErr != nil {
			continue
		}
		if _, ok, schemaErr := SchemaFromTypesDocument(string(document)); schemaErr != nil || !ok {
			continue
		}
		candidateStem := normalizeTypesStem(strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)))
		score := 0
		if strings.Contains(inputStem, candidateStem) || strings.Contains(candidateStem, inputStem) {
			score = 2
		}
		candidates = append(candidates, candidate{path: path, score: score})
	}
	if len(candidates) == 0 {
		return ""
	}
	bestScore := 0
	best := ""
	ambiguous := false
	for _, candidate := range candidates {
		if candidate.score > bestScore {
			best, bestScore, ambiguous = candidate.path, candidate.score, false
		} else if candidate.score == bestScore && candidate.score > 0 {
			ambiguous = true
		}
	}
	if bestScore == 0 {
		if len(candidates) == 1 {
			return candidates[0].path
		}
		return ""
	}
	if !ambiguous {
		return best
	}
	return ""
}

func normalizeTypesStem(stem string) string {
	var out strings.Builder
	for _, r := range strings.ToLower(stem) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			out.WriteRune(r)
		}
	}
	return out.String()
}

// SchemaFromTypesDocument resolves the documented multi-record .types format
// emitted by Kit generators. It intentionally returns false for the older
// comma-separated --types form, which remains handled by ParseDataTypes.
func SchemaFromTypesDocument(document string) (schema string, ok bool, err error) {
	var identity string
	for _, raw := range strings.Split(document, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "#") {
			identity = strings.TrimSpace(strings.TrimPrefix(line, "#"))
			break
		}
		if strings.Contains(line, "=") {
			return "", false, nil
		}
	}
	if identity == "" || !strings.Contains(document, "header =") {
		return "", false, nil
	}
	identity = strings.TrimSuffix(filepath.Base(identity), filepath.Ext(identity))
	identity = strings.TrimSuffix(identity, ".xsdat")
	schema = normalizeDataSchema(identity)
	if schema == identity || schema == "" {
		return "", false, fmt.Errorf("unsupported xsdat types document %q", identity)
	}
	return schema, true, nil
}
