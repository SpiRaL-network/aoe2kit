package scenario

import (
	"fmt"
	"math"
	"sort"

	"aoe2kit/pkg/datfile"
)

type PaletteUsageReport struct {
	Path         string              `json:"path,omitempty"`
	DatPath      string              `json:"dat_path,omitempty"`
	UnitTypes    int                 `json:"unit_types"`
	Placements   int                 `json:"placements"`
	Summary      PaletteUsageSummary `json:"summary"`
	Verification string              `json:"verification"`
	Rows         []PaletteUsageRow   `json:"rows"`
}

type PaletteUsageSummary struct {
	MultiVariantTypeCount int                      `json:"multi_variant_type_count"`
	ArtworksAvailable     int                      `json:"artworks_available"`
	ArtworksUsed          int                      `json:"artworks_used"`
	BiggestUntapped       []PaletteUntappedSummary `json:"biggest_untapped,omitempty"`
}

type PaletteUntappedSummary struct {
	UnitID                int    `json:"unit_id"`
	UnitName              string `json:"unit_name"`
	StandingGraphic1      int    `json:"standing_graphic_1"`
	GraphicName           string `json:"graphic_name,omitempty"`
	AvailableVariantCount int    `json:"available_variant_count"`
	UsedVariantCount      int    `json:"used_variant_count"`
	UnusedVariantCount    int    `json:"unused_variant_count"`
	Placements            int    `json:"placements"`
}

type PaletteUsageRow struct {
	UnitID                int       `json:"unit_id"`
	UnitName              string    `json:"unit_name"`
	Placements            int       `json:"placements"`
	UnitType              int       `json:"unit_type"`
	UnitClass             int16     `json:"unit_class"`
	UnitClassName         string    `json:"unit_class_name,omitempty"`
	StandingGraphic1      int       `json:"standing_graphic_1"`
	GraphicName           string    `json:"graphic_name,omitempty"`
	FileName              string    `json:"file_name,omitempty"`
	SLP                   int32     `json:"slp"`
	AngleCount            int       `json:"angle_count"`
	FrameCount            int       `json:"frame_count"`
	SequenceType          uint8     `json:"sequence_type"`
	Classification        string    `json:"classification"`
	Confidence            string    `json:"confidence"`
	RotationEncoding      string    `json:"rotation_encoding,omitempty"`
	AvailableVariantCount *int      `json:"available_variant_count,omitempty"`
	UsedIndices           []int     `json:"used_indices,omitempty"`
	UnusedIndices         []int     `json:"unused_indices,omitempty"`
	RawRotations          []float64 `json:"raw_rotations,omitempty"`
	Note                  string    `json:"note,omitempty"`
}

func PaletteUsageFile(path, datPath string) (PaletteUsageReport, error) {
	file, err := Open(path)
	if err != nil {
		return PaletteUsageReport{}, err
	}
	palette, err := datfile.PaletteFile(datPath, datfile.PaletteOptions{})
	if err != nil {
		return PaletteUsageReport{}, err
	}
	report := file.PaletteUsage(palette)
	report.Path = path
	report.DatPath = datPath
	return report, nil
}

func (f *File) PaletteUsage(palette datfile.PaletteReport) PaletteUsageReport {
	type usage struct {
		count     int
		rotations []float64
	}
	byUnit := map[int]*usage{}
	if f.Units != nil {
		for _, section := range f.Units.Sections {
			for _, unit := range section.Units {
				u := byUnit[unit.UnitConst]
				if u == nil {
					u = &usage{}
					byUnit[unit.UnitConst] = u
				}
				u.count++
				u.rotations = appendUniqueFloat(u.rotations, unit.Rotation)
			}
		}
	}
	paletteRowsByUnit := map[int][]datfile.PaletteRow{}
	for _, row := range palette.Rows {
		paletteRowsByUnit[row.UnitID] = append(paletteRowsByUnit[row.UnitID], row)
	}
	paletteByUnit := map[int]datfile.PaletteRow{}
	for unitID, rows := range paletteRowsByUnit {
		paletteByUnit[unitID] = choosePaletteUsageRow(rows)
	}
	report := PaletteUsageReport{
		Path:         f.Path,
		DatPath:      palette.Path,
		UnitTypes:    len(byUnit),
		Verification: "structure_verified_hypothesis_not_engine_verified; usage joins scenario unit_const/rotation to DAT standing-graphic counts",
	}
	for unitID, used := range byUnit {
		report.Placements += used.count
		paletteRow, ok := paletteByUnit[unitID]
		if !ok {
			report.Rows = append(report.Rows, PaletteUsageRow{
				UnitID:       unitID,
				Placements:   used.count,
				RawRotations: sortedFloats(used.rotations),
				Note:         "unit id not found in DAT palette",
			})
			continue
		}
		row := PaletteUsageRow{
			UnitID:                unitID,
			UnitName:              paletteRow.UnitName,
			Placements:            used.count,
			UnitType:              paletteRow.UnitType,
			UnitClass:             paletteRow.UnitClass,
			UnitClassName:         paletteRow.UnitClassName,
			StandingGraphic1:      paletteRow.StandingGraphic1,
			GraphicName:           paletteRow.GraphicName,
			FileName:              paletteRow.FileName,
			SLP:                   paletteRow.SLP,
			AngleCount:            paletteRow.AngleCount,
			FrameCount:            paletteRow.FrameCount,
			SequenceType:          paletteRow.SequenceType,
			Classification:        paletteRow.Classification,
			Confidence:            paletteRow.Confidence,
			RotationEncoding:      paletteRow.RotationEncoding,
			AvailableVariantCount: paletteRow.VariantCount,
			RawRotations:          sortedFloats(used.rotations),
		}
		if paletteRow.VariantCount != nil {
			row.UsedIndices = usedVariantIndices(used.rotations, *paletteRow.VariantCount)
			row.UnusedIndices = missingIndices(*paletteRow.VariantCount, row.UsedIndices)
			if len(row.UsedIndices) == 0 {
				row.Note = "no integer-like scenario rotation values fell within the available variant range"
			}
		} else if paletteRow.Classification == "animated" {
			row.Note = fmt.Sprintf("animated unit; rotation is treated as a phase/frame selector; phase distribution: %d placement%s across %d distinct raw rotation value%s; shared phase can be deliberate authorial alignment", used.count, pluralS(used.count), len(row.RawRotations), pluralS(len(row.RawRotations)))
		} else {
			row.Note = "graphic has frame and/or facing axes whose rotation semantics are type-dependent; raw rotations are shown without collapsing them into variant indices"
		}
		report.Rows = append(report.Rows, row)
	}
	sort.Slice(report.Rows, func(i, j int) bool {
		vi := usageSortVariant(report.Rows[i])
		vj := usageSortVariant(report.Rows[j])
		if vi != vj {
			return vi > vj
		}
		if report.Rows[i].Placements != report.Rows[j].Placements {
			return report.Rows[i].Placements > report.Rows[j].Placements
		}
		if report.Rows[i].UnitID != report.Rows[j].UnitID {
			return report.Rows[i].UnitID < report.Rows[j].UnitID
		}
		return report.Rows[i].StandingGraphic1 < report.Rows[j].StandingGraphic1
	})
	report.Summary = summarizePaletteUsage(report.Rows)
	return report
}

func summarizePaletteUsage(rows []PaletteUsageRow) PaletteUsageSummary {
	summary := PaletteUsageSummary{}
	for _, row := range rows {
		if row.AvailableVariantCount == nil {
			continue
		}
		summary.MultiVariantTypeCount++
		summary.ArtworksAvailable += *row.AvailableVariantCount
		summary.ArtworksUsed += len(row.UsedIndices)
		summary.BiggestUntapped = append(summary.BiggestUntapped, PaletteUntappedSummary{
			UnitID:                row.UnitID,
			UnitName:              row.UnitName,
			StandingGraphic1:      row.StandingGraphic1,
			GraphicName:           row.GraphicName,
			AvailableVariantCount: *row.AvailableVariantCount,
			UsedVariantCount:      len(row.UsedIndices),
			UnusedVariantCount:    len(row.UnusedIndices),
			Placements:            row.Placements,
		})
	}
	sort.Slice(summary.BiggestUntapped, func(i, j int) bool {
		if summary.BiggestUntapped[i].UnusedVariantCount != summary.BiggestUntapped[j].UnusedVariantCount {
			return summary.BiggestUntapped[i].UnusedVariantCount > summary.BiggestUntapped[j].UnusedVariantCount
		}
		if summary.BiggestUntapped[i].AvailableVariantCount != summary.BiggestUntapped[j].AvailableVariantCount {
			return summary.BiggestUntapped[i].AvailableVariantCount > summary.BiggestUntapped[j].AvailableVariantCount
		}
		if summary.BiggestUntapped[i].UnitID != summary.BiggestUntapped[j].UnitID {
			return summary.BiggestUntapped[i].UnitID < summary.BiggestUntapped[j].UnitID
		}
		return summary.BiggestUntapped[i].StandingGraphic1 < summary.BiggestUntapped[j].StandingGraphic1
	})
	if len(summary.BiggestUntapped) > 10 {
		summary.BiggestUntapped = summary.BiggestUntapped[:10]
	}
	return summary
}

func choosePaletteUsageRow(rows []datfile.PaletteRow) datfile.PaletteRow {
	if len(rows) == 0 {
		return datfile.PaletteRow{}
	}
	best := rows[0]
	for _, row := range rows[1:] {
		if paletteUsageRowLess(best, row) {
			best = row
		}
	}
	return best
}

func paletteUsageRowLess(a, b datfile.PaletteRow) bool {
	aGaia := containsInt(a.CivIndices, 0)
	bGaia := containsInt(b.CivIndices, 0)
	if aGaia != bGaia {
		return bGaia
	}
	if len(a.CivIndices) != len(b.CivIndices) {
		return len(a.CivIndices) < len(b.CivIndices)
	}
	av := datPaletteVariantSort(a)
	bv := datPaletteVariantSort(b)
	if av != bv {
		return av < bv
	}
	if a.UnitID != b.UnitID {
		return a.UnitID > b.UnitID
	}
	return a.StandingGraphic1 > b.StandingGraphic1
}

func datPaletteVariantSort(row datfile.PaletteRow) int {
	if row.VariantCount != nil {
		return *row.VariantCount
	}
	if row.AngleCount > row.FrameCount {
		return row.AngleCount
	}
	return row.FrameCount
}

func containsInt(values []int, needle int) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}

func pluralS(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

func usedVariantIndices(rotations []float64, variantCount int) []int {
	set := map[int]bool{}
	for _, rotation := range rotations {
		rounded := math.Round(rotation)
		if math.Abs(rotation-rounded) > 0.001 {
			continue
		}
		index := int(rounded)
		if index < 0 || index >= variantCount {
			continue
		}
		set[index] = true
	}
	out := make([]int, 0, len(set))
	for index := range set {
		out = append(out, index)
	}
	sort.Ints(out)
	return out
}

func missingIndices(count int, used []int) []int {
	usedSet := map[int]bool{}
	for _, index := range used {
		usedSet[index] = true
	}
	out := make([]int, 0)
	for i := 0; i < count; i++ {
		if !usedSet[i] {
			out = append(out, i)
		}
	}
	return out
}

func usageSortVariant(row PaletteUsageRow) int {
	if row.AvailableVariantCount != nil {
		return *row.AvailableVariantCount
	}
	if row.AngleCount > row.FrameCount {
		return row.AngleCount
	}
	return row.FrameCount
}

func appendUniqueFloat(values []float64, value float64) []float64 {
	for _, existing := range values {
		if math.Abs(existing-value) <= 0.0001 {
			return values
		}
	}
	return append(values, value)
}

func sortedFloats(values []float64) []float64 {
	out := append([]float64(nil), values...)
	sort.Float64s(out)
	return out
}
