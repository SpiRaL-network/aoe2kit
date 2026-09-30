package harness

import "fmt"

// ExpandRemovalSweep expands one removal probe template into manifest entries.
// Expected is intentionally unset: the point of this sweep is to measure which
// units retire under HP zero, including units for which we have no prediction.
func ExpandRemovalSweep(unitConsts []int) []Fact {
	out := make([]Fact, 0, len(unitConsts))
	for _, unit := range unitConsts {
		out = append(out, Fact{
			ID:          fmt.Sprintf("remove_hp_zero_unit_%d", unit),
			Description: fmt.Sprintf("unit %d existence before and after xsSetUnitHitpoints(0.0)", unit),
			Category:    "removal_matrix",
			Setup:       fmt.Sprintf("Create unit constant %d in a bounded test lane.", unit),
			Operation:   "Record existence, set hit points to zero, then record existence and final disposition.",
			Risk:        "safe",
			Evidence:    "measurement required; no prediction supplied",
		})
	}
	return out
}
