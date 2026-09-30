package enginefacts

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	EngineVerified = "engine_verified"
	SourceVerified = "source_verified"
)

type Ledger struct {
	SchemaVersion int    `json:"schema_version"`
	Generated     string `json:"generated"`
	Facts         []Fact `json:"facts"`
}

type Fact struct {
	ID           string   `json:"id"`
	Statement    string   `json:"statement"`
	Tier         string   `json:"tier"`
	VerifiedDate string   `json:"verified_date"`
	FixtureRef   string   `json:"fixture_ref"`
	Domains      []string `json:"domains,omitempty"`
	LintRule     *Rule    `json:"lint_rule,omitempty"`
}

type Rule struct {
	Target    string `json:"target,omitempty"`
	Predicate string `json:"predicate,omitempty"`
	Severity  string `json:"severity,omitempty"`
}

type Citation struct {
	FactID       string `json:"fact_id,omitempty"`
	Tier         string `json:"fact_tier,omitempty"`
	VerifiedDate string `json:"verified_date,omitempty"`
	FixtureRef   string `json:"fixture_ref,omitempty"`
}

func LoadDefault() (Ledger, error) {
	candidates := defaultLedgerCandidates()
	var readErrs []string
	for _, candidate := range candidates {
		data, err := os.ReadFile(candidate)
		if err != nil {
			readErrs = append(readErrs, candidate+": "+err.Error())
			continue
		}
		ledger, err := Parse(data)
		if err != nil {
			return Ledger{}, fmt.Errorf("%s: %w", candidate, err)
		}
		return ledger, nil
	}
	return Ledger{}, fmt.Errorf("engine facts ledger not found; checked %s", strings.Join(readErrs, "; "))
}

func Parse(data []byte) (Ledger, error) {
	var ledger Ledger
	if err := json.Unmarshal(data, &ledger); err != nil {
		return Ledger{}, err
	}
	if err := ledger.Validate(); err != nil {
		return Ledger{}, err
	}
	ledger.Sort()
	return ledger, nil
}

func (l Ledger) Validate() error {
	if l.SchemaVersion <= 0 {
		return fmt.Errorf("schema_version must be positive")
	}
	seen := map[string]bool{}
	for i, fact := range l.Facts {
		if strings.TrimSpace(fact.ID) == "" {
			return fmt.Errorf("facts[%d] missing id", i)
		}
		if seen[fact.ID] {
			return fmt.Errorf("duplicate fact id %q", fact.ID)
		}
		seen[fact.ID] = true
		if strings.TrimSpace(fact.Statement) == "" {
			return fmt.Errorf("fact %s missing statement", fact.ID)
		}
		if !validTier(fact.Tier) {
			return fmt.Errorf("fact %s has invalid tier %q", fact.ID, fact.Tier)
		}
		if strings.TrimSpace(fact.VerifiedDate) == "" {
			return fmt.Errorf("fact %s missing verified_date", fact.ID)
		}
		if strings.TrimSpace(fact.FixtureRef) == "" {
			return fmt.Errorf("fact %s missing fixture_ref", fact.ID)
		}
	}
	return nil
}

func (l Ledger) Sort() {
	sort.Slice(l.Facts, func(i, j int) bool {
		if l.Facts[i].Tier == l.Facts[j].Tier {
			return l.Facts[i].ID < l.Facts[j].ID
		}
		return tierRank(l.Facts[i].Tier) < tierRank(l.Facts[j].Tier)
	})
}

func (l Ledger) ByID(id string) (Fact, bool) {
	for _, fact := range l.Facts {
		if fact.ID == id {
			return fact, true
		}
	}
	return Fact{}, false
}

func (l Ledger) Filter(includeProvisional bool, domain string) []Fact {
	domain = strings.TrimSpace(strings.ToLower(domain))
	var out []Fact
	for _, fact := range l.Facts {
		if !includeProvisional && !trustedByDefault(fact.Tier) {
			continue
		}
		if domain != "" && !factHasDomain(fact, domain) {
			continue
		}
		out = append(out, fact)
	}
	return out
}

func CitationFor(id string, includeProvisional bool) Citation {
	ledger, err := LoadDefault()
	if err != nil {
		return Citation{FactID: id}
	}
	fact, ok := ledger.ByID(id)
	if !ok {
		return Citation{FactID: id}
	}
	if !includeProvisional && !trustedByDefault(fact.Tier) {
		return Citation{}
	}
	return Citation{
		FactID:       fact.ID,
		Tier:         fact.Tier,
		VerifiedDate: fact.VerifiedDate,
		FixtureRef:   fact.FixtureRef,
	}
}

func IsEngineVerified(id string) bool {
	ledger, err := LoadDefault()
	if err != nil {
		return false
	}
	fact, ok := ledger.ByID(id)
	return ok && fact.Tier == EngineVerified
}

func defaultLedgerCandidates() []string {
	var out []string
	addAncestors := func(start string) {
		dir := filepath.Clean(start)
		for {
			out = append(out, filepath.Join(dir, "data", "engine_facts.json"))
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}
	if cwd, err := os.Getwd(); err == nil {
		addAncestors(cwd)
	}
	if exe, err := os.Executable(); err == nil {
		addAncestors(filepath.Dir(exe))
	}
	return dedupeStrings(out)
}

func factHasDomain(fact Fact, domain string) bool {
	for _, value := range fact.Domains {
		if strings.EqualFold(value, domain) {
			return true
		}
	}
	return false
}

func validTier(tier string) bool {
	switch tier {
	case EngineVerified, SourceVerified, "structure_verified", "strong_hypothesis", "heuristic":
		return true
	default:
		return false
	}
}

func trustedByDefault(tier string) bool {
	switch tier {
	case EngineVerified, SourceVerified:
		return true
	default:
		return false
	}
}

func tierRank(tier string) int {
	switch tier {
	case EngineVerified:
		return 0
	case SourceVerified:
		return 1
	case "structure_verified":
		return 2
	case "strong_hypothesis":
		return 3
	case "heuristic":
		return 4
	default:
		return 9
	}
}

func dedupeStrings(values []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}
