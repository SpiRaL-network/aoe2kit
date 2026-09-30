package datfile

import (
	"fmt"
	"sort"
	"strings"
)

type PaletteOptions struct {
	ID          *int
	MinVariants int
}

type PaletteReport struct {
	Path         string         `json:"path,omitempty"`
	Version      string         `json:"version"`
	UnitCount    int            `json:"unit_count"`
	Returned     int            `json:"returned"`
	Filters      PaletteOptions `json:"filters"`
	Verification string         `json:"verification"`
	Rows         []PaletteRow   `json:"rows"`
}

type PaletteRow struct {
	UnitID           int    `json:"unit_id"`
	UnitName         string `json:"unit_name"`
	CivIndices       []int  `json:"civ_indices,omitempty"`
	UnitType         int    `json:"unit_type"`
	UnitClass        int16  `json:"unit_class"`
	UnitClassName    string `json:"unit_class_name,omitempty"`
	StandingGraphic1 int    `json:"standing_graphic_1"`
	GraphicName      string `json:"graphic_name,omitempty"`
	FileName         string `json:"file_name,omitempty"`
	SLP              int32  `json:"slp"`
	AngleCount       int    `json:"angle_count"`
	FrameCount       int    `json:"frame_count"`
	SequenceType     uint8  `json:"sequence_type"`
	VariantCount     *int   `json:"variant_count,omitempty"`
	VariantNote      string `json:"variant_note,omitempty"`
	Classification   string `json:"classification"`
	Confidence       string `json:"confidence"`
	Evidence         string `json:"evidence"`
	RotationEncoding string `json:"rotation_encoding,omitempty"`
}

func PaletteFile(path string, opts PaletteOptions) (PaletteReport, error) {
	idx, err := Open(path)
	if err != nil {
		return PaletteReport{}, err
	}
	report := idx.Palette(opts)
	report.Path = path
	return report, nil
}

func (idx *Index) Palette(opts PaletteOptions) PaletteReport {
	rowsByKey := map[string]*PaletteRow{}
	total := 0
	for _, civ := range idx.Civs {
		for _, unit := range civ.Units {
			if !unit.Present {
				continue
			}
			total++
			unitID := int(unit.ID)
			if unitID < 0 {
				unitID = unit.Index
			}
			if opts.ID != nil && unitID != *opts.ID && unit.Index != *opts.ID {
				continue
			}
			graphicID := int(unit.StandingGraphic1)
			graphic, ok := idx.Graphic(graphicID)
			key := fmt.Sprintf("%d/%d", unitID, graphicID)
			row, exists := rowsByKey[key]
			if !exists {
				newRow := PaletteRow{
					UnitID:           unitID,
					UnitName:         unit.Name,
					UnitType:         unit.Type,
					UnitClass:        unit.Class,
					UnitClassName:    unit.ClassName,
					StandingGraphic1: graphicID,
				}
				if ok {
					newRow.GraphicName = graphic.Name.Value
					newRow.FileName = graphic.FileName.Value
					newRow.SLP = graphic.SLP
					newRow.AngleCount = int(graphic.AngleCount)
					newRow.FrameCount = int(graphic.FrameCount)
					newRow.SequenceType = graphic.SequenceType
				}
				classifyPaletteRow(&newRow)
				row = &newRow
				rowsByKey[key] = row
			}
			row.CivIndices = appendUniqueInt(row.CivIndices, civ.Index)
		}
	}
	rows := make([]PaletteRow, 0, len(rowsByKey))
	for _, row := range rowsByKey {
		if opts.MinVariants > 0 {
			if row.VariantCount == nil || *row.VariantCount < opts.MinVariants {
				continue
			}
		}
		sort.Ints(row.CivIndices)
		rows = append(rows, *row)
	}
	sort.Slice(rows, func(i, j int) bool {
		vi := paletteSortVariant(rows[i])
		vj := paletteSortVariant(rows[j])
		if vi != vj {
			return vi > vj
		}
		if rows[i].FrameCount != rows[j].FrameCount {
			return rows[i].FrameCount > rows[j].FrameCount
		}
		if rows[i].AngleCount != rows[j].AngleCount {
			return rows[i].AngleCount > rows[j].AngleCount
		}
		if rows[i].UnitID != rows[j].UnitID {
			return rows[i].UnitID < rows[j].UnitID
		}
		return rows[i].StandingGraphic1 < rows[j].StandingGraphic1
	})
	return PaletteReport{
		Path:         idx.Path,
		Version:      idx.Version,
		UnitCount:    total,
		Returned:     len(rows),
		Filters:      opts,
		Verification: "structure_verified_hypothesis_not_engine_verified; IndianStatues angle-addressing is engine-measured, other classifications derive from DAT counts",
		Rows:         rows,
	}
}

func classifyPaletteRow(row *PaletteRow) {
	row.Confidence = "hypothesis"
	row.Evidence = "derived from DAT standing_graphic_1 -> graphic angle_count/frame_count/sequence_type"
	if isAuthorConfirmedAnimatedFish(row) {
		row.Classification = "animated"
		row.Confidence = "author_confirmed"
		row.Evidence = "chrae author confirmation: Gaia fish units use rotation as an animation-frame selector"
		row.VariantNote = fmt.Sprintf("angle_count=%d frame_count=%d sequence_type=%d; fish-family rotation is author-confirmed animation-frame selection", row.AngleCount, row.FrameCount, row.SequenceType)
		row.RotationEncoding = "integer_animation_phase"
		return
	}
	switch {
	case row.AngleCount <= 1 && row.FrameCount <= 1:
		row.Classification = "single_artwork"
		one := 1
		row.VariantCount = &one
		row.RotationEncoding = "ignored_or_single_variant"
	case row.UnitType == 10 && row.FrameCount <= 1 && row.AngleCount > 1:
		row.Classification = "multi_variant"
		count := row.AngleCount
		row.VariantCount = &count
		row.Confidence = "editor_fixture"
		row.Evidence = "editor-authored references show type-10 eyecandy stores small integer scenario rotation values as artwork/angle indices"
		row.RotationEncoding = "integer_artwork_index"
	case row.FrameCount <= 1 && row.AngleCount > 1 && row.SequenceType == 6:
		row.Classification = "multi_variant"
		count := row.AngleCount
		row.VariantCount = &count
		row.RotationEncoding = "integer_artwork_index"
	case row.FrameCount <= 1 && row.AngleCount > 1:
		row.Classification = "rotating"
		row.VariantNote = fmt.Sprintf("angle_count=%d frame_count=%d sequence_type=%d; treated as facing angles, not addressable artwork variants", row.AngleCount, row.FrameCount, row.SequenceType)
		row.RotationEncoding = "radians_facing_angle"
	default:
		row.Classification = "ambiguous"
		row.VariantNote = fmt.Sprintf("angle_count=%d frame_count=%d sequence_type=%d; animation/facing/frame-addressing semantics are type-dependent, so raw counts are reported without choosing a variant axis", row.AngleCount, row.FrameCount, row.SequenceType)
		row.RotationEncoding = "type_dependent_unknown"
	}
	if row.UnitID == 1777 && row.StandingGraphic1 == 12530 && row.AngleCount == 16 && row.FrameCount == 1 {
		row.Confidence = "engine_measured_one_case"
		row.Evidence = "field measurement: scenario rotation values select distinct IndianStatues artworks"
		row.RotationEncoding = "integer_artwork_index"
	}
}

func isAuthorConfirmedAnimatedFish(row *PaletteRow) bool {
	switch row.UnitID {
	case 53, 455, 456, 457, 458, 2625:
		return true
	default:
		return strings.HasPrefix(strings.ToLower(row.GraphicName), "fish ")
	}
}

func paletteSortVariant(row PaletteRow) int {
	if row.VariantCount != nil {
		return *row.VariantCount
	}
	if row.AngleCount > row.FrameCount {
		return row.AngleCount
	}
	return row.FrameCount
}

func appendUniqueInt(values []int, value int) []int {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}
