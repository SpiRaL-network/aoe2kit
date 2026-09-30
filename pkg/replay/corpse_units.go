package replay

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type CorpseTable struct {
	SchemaVersion int              `json:"schema_version"`
	Generated     string           `json:"generated"`
	Method        string           `json:"method"`
	Rows          []CorpseTableRow `json:"rows"`
}

type CorpseTableRow struct {
	UnitID                  int      `json:"unit_id"`
	UnitName                string   `json:"unit_name"`
	CorpseID                int      `json:"corpse_id"`
	CorpseName              string   `json:"corpse_name"`
	Delta                   int64    `json:"delta"`
	Provenance              []string `json:"provenance,omitempty"`
	Confidence              string   `json:"confidence"`
	FixtureRef              string   `json:"fixture_ref,omitempty"`
	ObservedCarryMin        *int     `json:"observed_carry_min,omitempty"`
	ObservedCarryMax        *int     `json:"observed_carry_max,omitempty"`
	ObservedCarryDrop       *int     `json:"observed_carry_drop,omitempty"`
	ObservedDurationMS      *int     `json:"observed_duration_ms,omitempty"`
	ObservedDecayPerSecond  *float64 `json:"observed_decay_per_second,omitempty"`
	ObservedFreshCarryDelta *int     `json:"observed_fresh_carry_delta,omitempty"`
	CarryMeaning            string   `json:"carry_meaning,omitempty"`
}

func LoadDefaultCorpseTable() (CorpseTable, error) {
	for _, candidate := range defaultCorpseTableCandidates() {
		data, err := os.ReadFile(candidate)
		if err != nil {
			continue
		}
		table, err := ParseCorpseTable(data)
		if err != nil {
			return CorpseTable{}, fmt.Errorf("%s: %w", candidate, err)
		}
		return table, nil
	}
	return ParseCorpseTable([]byte(fallbackCorpseTableJSON))
}

func ParseCorpseTable(data []byte) (CorpseTable, error) {
	var table CorpseTable
	if err := json.Unmarshal(data, &table); err != nil {
		return CorpseTable{}, err
	}
	if table.SchemaVersion <= 0 {
		return CorpseTable{}, fmt.Errorf("schema_version must be positive")
	}
	for i := range table.Rows {
		row := &table.Rows[i]
		if row.UnitID <= 0 || row.CorpseID <= 0 {
			return CorpseTable{}, fmt.Errorf("rows[%d] has invalid unit/corpse ids", i)
		}
		if row.Delta == 0 {
			row.Delta = int64(row.CorpseID - row.UnitID)
		}
		if row.Delta != int64(row.CorpseID-row.UnitID) {
			return CorpseTable{}, fmt.Errorf("rows[%d] delta=%d does not match corpse_id-unit_id=%d", i, row.Delta, row.CorpseID-row.UnitID)
		}
		if strings.TrimSpace(row.Confidence) == "" {
			row.Confidence = "heuristic"
		}
	}
	sort.SliceStable(table.Rows, func(i, j int) bool {
		if table.Rows[i].UnitID == table.Rows[j].UnitID {
			return table.Rows[i].CorpseID < table.Rows[j].CorpseID
		}
		return table.Rows[i].UnitID < table.Rows[j].UnitID
	})
	return table, nil
}

func (t CorpseTable) UniqueReplacementForDelta(delta int64) (CorpseTableRow, bool) {
	var match CorpseTableRow
	count := 0
	for _, row := range t.Rows {
		if row.Delta != delta {
			continue
		}
		match = row
		count++
	}
	return match, count == 1
}

func (t CorpseTable) Replacement(unitID, corpseID int) (CorpseTableRow, bool) {
	for _, row := range t.Rows {
		if row.UnitID == unitID && row.CorpseID == corpseID {
			return row, true
		}
	}
	return CorpseTableRow{}, false
}

func defaultCorpseTableCandidates() []string {
	var out []string
	addAncestors := func(start string) {
		dir := filepath.Clean(start)
		for {
			out = append(out, filepath.Join(dir, "data", "corpse_units.json"))
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
	seen := map[string]bool{}
	deduped := make([]string, 0, len(out))
	for _, value := range out {
		if seen[value] {
			continue
		}
		seen[value] = true
		deduped = append(deduped, value)
	}
	return deduped
}

const fallbackCorpseTableJSON = `{
  "schema_version": 1,
  "generated": "2026-08-19",
  "method": "embedded fallback subset of data/corpse_units.json",
  "rows": [
    {"unit_id":73,"unit_name":"CHUKN","corpse_id":28,"corpse_name":"CHUKN_D","delta":-45,"provenance":["observed","name_heuristic","zero_hp"],"confidence":"engine_verified","fixture_ref":"engine fixture: kill-factory run, 2026-08-18"},
    {"unit_id":625,"unit_name":"PAV3","corpse_id":1478,"corpse_name":"Pavilion C (Rubble)","delta":853,"provenance":["observed","zero_hp","rubble_name"],"confidence":"engine_verified","fixture_ref":"engine fixture: executioner ledger run (v12)"},
    {"unit_id":626,"unit_name":"PAV2","corpse_id":1477,"corpse_name":"Pavilion B (Rubble)","delta":851,"provenance":["observed","zero_hp","rubble_name"],"confidence":"engine_verified","fixture_ref":"engine fixture: executioner ledger run (v12)"}
  ]
}`
