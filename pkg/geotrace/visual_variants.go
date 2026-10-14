package geotrace

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
)

//go:embed visual_variants.json
var visualVariantBytes []byte

type visualVariantRecord struct {
	UnitConst    int              `json:"unit_const"`
	AddressField string           `json:"address_field"`
	Axis         string           `json:"axis"`
	Confidence   string           `json:"confidence"`
	States       map[string][]int `json:"states"`
}

var visualVariantRecords = parseVisualVariantRecords()

func parseVisualVariantRecords() map[int]visualVariantRecord {
	var doc struct {
		SchemaVersion int                            `json:"schema_version"`
		Records       map[string]visualVariantRecord `json:"records"`
	}
	if err := json.Unmarshal(visualVariantBytes, &doc); err != nil {
		panic(fmt.Sprintf("parse embedded visual variants: %v", err))
	}
	if doc.SchemaVersion != 1 {
		panic(fmt.Sprintf("unsupported embedded visual variant schema %d", doc.SchemaVersion))
	}
	out := make(map[int]visualVariantRecord, len(doc.Records))
	for key, record := range doc.Records {
		if record.UnitConst <= 0 || key != fmt.Sprint(record.UnitConst) || record.AddressField != "rotation" || record.Axis != "angle_variant" {
			panic(fmt.Sprintf("invalid visual variant record %q", key))
		}
		for state, indices := range record.States {
			seen := map[int]bool{}
			for _, index := range indices {
				if index < 0 || seen[index] {
					panic(fmt.Sprintf("invalid facet %d in visual variant %d state %q", index, record.UnitConst, state))
				}
				seen[index] = true
			}
		}
		out[record.UnitConst] = record
	}
	return out
}

func preferredVariantStates(context string) []string {
	context = strings.ToLower(context)
	switch {
	case strings.Contains(context, "winter"), strings.Contains(context, "snow"):
		return []string{"snow", "bare-dead", "green", "autumn-gold", "autumn-red"}
	case strings.Contains(context, "autumn"), strings.Contains(context, "fall"):
		return []string{"autumn-gold", "autumn-red", "bare-dead", "green", "snow"}
	case strings.Contains(context, "bare"), strings.Contains(context, "dead"), strings.Contains(context, "dormant"):
		return []string{"bare-dead", "snow", "autumn-gold", "autumn-red", "green"}
	default:
		return []string{"green", "autumn-gold", "autumn-red", "bare-dead", "snow"}
	}
}

func variantFacets(unit int, context string) []int {
	record, ok := visualVariantRecords[unit]
	if !ok {
		return nil
	}
	for _, state := range preferredVariantStates(context) {
		if indices := record.States[state]; len(indices) > 0 {
			return indices
		}
	}
	return nil
}

func visualVariantRotation(unit int, climate, biome string, x, y, seed int) (float64, bool) {
	indices := variantFacets(unit, biome+" "+climate)
	if len(indices) == 0 {
		return 0, false
	}
	return float64(indices[int(hashUnitFloat(x, y, seed)*float64(len(indices)))]), true
}

func visualVariantRotationChoices(unit int, biome string) []float64 {
	indices := variantFacets(unit, biome)
	choices := make([]float64, len(indices))
	for i, index := range indices {
		choices[i] = float64(index)
	}
	return choices
}
