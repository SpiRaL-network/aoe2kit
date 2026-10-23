package scenario

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// MapPatchDiffReport describes a deterministic map-art-only recipe that turns
// Before into After. It deliberately refuses to represent changes outside
// terrain, elevation, and placed-unit fields.
type MapPatchDiffReport struct {
	Before              string               `json:"before,omitempty"`
	After               string               `json:"after,omitempty"`
	Patchable           bool                 `json:"patchable"`
	Width               int                  `json:"width"`
	Height              int                  `json:"height"`
	ChangedCells        int                  `json:"changed_cells"`
	AddedUnits          int                  `json:"added_units"`
	RemovedUnits        int                  `json:"removed_units"`
	EditedUnits         int                  `json:"edited_units"`
	UnsupportedChanges  []string             `json:"unsupported_changes,omitempty"`
	OpaqueDifferences   []StructureFieldDiff `json:"opaque_differences,omitempty"`
	IgnoredSaveMetadata []StructureFieldDiff `json:"ignored_save_metadata,omitempty"`
	Recipe              Recipe               `json:"recipe"`
}

// MapPatchDiffFiles computes a map-art patch recipe from two scenarios.
func MapPatchDiffFiles(beforePath, afterPath string) (MapPatchDiffReport, error) {
	before, err := Open(beforePath)
	if err != nil {
		return MapPatchDiffReport{}, fmt.Errorf("open before: %w", err)
	}
	after, err := Open(afterPath)
	if err != nil {
		return MapPatchDiffReport{}, fmt.Errorf("open after: %w", err)
	}
	report, err := MapPatchDiff(before, after)
	report.Before, report.After = beforePath, afterPath
	return report, err
}

// MapPatchDiff computes terrain/elevation and unit changes. Any unsupported
// semantic edit makes Patchable false, preventing callers from silently
// turning a full scenario edit into a partial map-art patch.
func MapPatchDiff(before, after *File) (MapPatchDiffReport, error) {
	if before == nil || after == nil {
		return MapPatchDiffReport{}, errors.New("map patch diff needs before and after scenarios")
	}
	report := MapPatchDiffReport{Patchable: true}
	if before.Map == nil || after.Map == nil {
		return report, errors.New("map patch diff needs parsed maps")
	}
	width, height := before.Map.Width, before.Map.Height
	report.Width, report.Height = width, height
	if width != after.Map.Width || height != after.Map.Height {
		report.Patchable = false
		report.UnsupportedChanges = append(report.UnsupportedChanges, "map dimensions changed")
		return report, nil
	}

	beforeTerrain, err := before.Terrain(TerrainOptions{IncludeTiles: true})
	if err != nil {
		return report, fmt.Errorf("read before terrain: %w", err)
	}
	afterTerrain, err := after.Terrain(TerrainOptions{IncludeTiles: true})
	if err != nil {
		return report, fmt.Errorf("read after terrain: %w", err)
	}
	want := width * height
	if len(beforeTerrain.Tiles) != want || len(afterTerrain.Tiles) != want {
		return report, fmt.Errorf("terrain tile count mismatch: before=%d after=%d want=%d", len(beforeTerrain.Tiles), len(afterTerrain.Tiles), want)
	}
	changed := make([]bool, want)
	for i := 0; i < want; i++ {
		b, a := beforeTerrain.Tiles[i], afterTerrain.Tiles[i]
		if b.X != a.X || b.Y != a.Y {
			return report, fmt.Errorf("terrain coordinate mismatch at index %d", i)
		}
		changed[i] = b.TerrainID != a.TerrainID || b.Elevation != a.Elevation || b.Layer != a.Layer
		if changed[i] {
			report.ChangedCells++
		}
	}
	report.Recipe.Map = terrainCellRectangles(afterTerrain.Tiles, changed, width, height)

	unitOps, unitCounts, unitUnsupported, err := diffSpatialUnits(before.Units, after.Units)
	if err != nil {
		return report, err
	}
	report.Recipe.Units = unitOps
	report.AddedUnits, report.RemovedUnits, report.EditedUnits = unitCounts[0], unitCounts[1], unitCounts[2]
	report.UnsupportedChanges = append(report.UnsupportedChanges, unitUnsupported...)

	semantic := Diff(before, after)
	for _, change := range semantic.Changes {
		if change.Kind == "raw" && change.Field == "opaque_fields" {
			continue // classified by the path-aligned raw field report below
		}
		if change.Kind == "scenario" && change.Field == "inflated_body_bytes" {
			continue // expected when encoded terrain or unit records change
		}
		if change.Kind == "scenario" && isEditorVersionBump(before.Version, after.Version) && change.Field == "version" {
			continue // 1.59 save migration is metadata; map edits remain diffed semantically
		}
		if change.Kind == "map" && isTerrainCountChange(change.Field) {
			continue
		}
		if change.Kind == "units" && isSpatialUnitChange(change.Field) {
			continue
		}
		report.UnsupportedChanges = append(report.UnsupportedChanges, change.Kind+":"+change.Field)
	}
	for _, raw := range semantic.RawFields {
		if isMapPatchSaveMetadata(raw.Path) {
			report.IgnoredSaveMetadata = append(report.IgnoredSaveMetadata, raw)
			continue
		}
		report.OpaqueDifferences = append(report.OpaqueDifferences, raw)
		report.UnsupportedChanges = append(report.UnsupportedChanges, "opaque:"+raw.Path)
	}
	report.UnsupportedChanges = uniqueSortedStrings(report.UnsupportedChanges)
	report.Patchable = len(report.UnsupportedChanges) == 0
	return report, nil
}

func isEditorVersionBump(before, after string) bool {
	return before == "1.58" && after == "1.59"
}

// Only documented save bookkeeping is ignored. Unknown Options fields and
// trigger/settings data stay visible and block recipe export.
func isMapPatchSaveMetadata(path string) bool {
	if path == "FileHeader.timestamp_of_last_save" || path == "DataHeader.next_unit_id_to_place" {
		return true
	}
	if stringsHasPrefixAny(path, "Units.player_data_3[") && (strings.HasSuffix(path, "].editor_camera_x") || strings.HasSuffix(path, "].editor_camera_y")) {
		return true
	}
	return strings.Contains(path, ".unknown_1_59_before_caption_string_id")
}

func isTerrainCountChange(field string) bool {
	return len(field) > len("terrain_")+len("_count") && field[:len("terrain_")] == "terrain_" && field[len(field)-len("_count"):] == "_count"
}

func isSpatialUnitChange(field string) bool {
	return stringsHasPrefixAny(field, "player_", "units/player_") || field == "total"
}

func stringsHasPrefixAny(s string, prefixes ...string) bool {
	for _, p := range prefixes {
		if len(s) >= len(p) && s[:len(p)] == p {
			return true
		}
	}
	return false
}

// terrainCellRectangles greedily covers changed tiles with equal resulting
// values using disjoint rectangles, yielding a stable minimal-cell recipe.
func terrainCellRectangles(tiles []TerrainTile, changed []bool, width, height int) []MapRecipe {
	used := make([]bool, len(changed))
	var out []MapRecipe
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			i := y*width + x
			if !changed[i] || used[i] {
				continue
			}
			v := tiles[i]
			x2 := x
			for x2+1 < width {
				j := y*width + x2 + 1
				if !changed[j] || used[j] || !sameTerrainValue(tiles[j], v) {
					break
				}
				x2++
			}
			y2 := y
			for y2+1 < height {
				ok := true
				for xx := x; xx <= x2; xx++ {
					j := (y2+1)*width + xx
					if !changed[j] || used[j] || !sameTerrainValue(tiles[j], v) {
						ok = false
						break
					}
				}
				if !ok {
					break
				}
				y2++
			}
			for yy := y; yy <= y2; yy++ {
				for xx := x; xx <= x2; xx++ {
					used[yy*width+xx] = true
				}
			}
			terrainID, elevation, layer := v.TerrainID, v.Elevation, v.Layer
			out = append(out, MapRecipe{Op: "set_terrain_rect", X1: x, Y1: y, X2: x2, Y2: y2,
				TerrainID: &terrainID, Elevation: &elevation, Layer: &layer})
		}
	}
	return out
}

func sameTerrainValue(a, b TerrainTile) bool {
	return a.TerrainID == b.TerrainID && a.Elevation == b.Elevation && a.Layer == b.Layer
}

func diffSpatialUnits(before, after *UnitInfo) ([]UnitRecipe, [3]int, []string, error) {
	var counts [3]int // additions, removals, edits
	if before == nil || after == nil {
		return nil, counts, nil, errors.New("map patch diff needs parsed units")
	}
	bm, bdup := spatialUnitMap(before)
	am, adup := spatialUnitMap(after)
	if len(bdup)+len(adup) > 0 {
		return nil, counts, []string{"unit reference ids are ambiguous"}, nil
	}
	ids := make(map[int]bool, len(bm)+len(am))
	for id := range bm {
		ids[id] = true
	}
	for id := range am {
		ids[id] = true
	}
	ordered := make([]int, 0, len(ids))
	for id := range ids {
		ordered = append(ordered, id)
	}
	sort.Ints(ordered)
	var removes, edits, adds []UnitRecipe
	var unsupported []string
	for _, id := range ordered {
		b, bok := bm[id]
		a, aok := am[id]
		switch {
		case bok && !aok:
			ref := id
			removes = append(removes, UnitRecipe{Op: "remove_unit", ReferenceID: &ref})
			counts[1]++
		case !bok && aok:
			adds = append(adds, recipeAddUnit(a))
			counts[0]++
		case bok && aok:
			op, changed, unpatchable := recipeEditUnit(id, b, a)
			if unpatchable != "" {
				unsupported = append(unsupported, fmt.Sprintf("unit %d: %s", id, unpatchable))
			} else if changed {
				edits = append(edits, op)
				counts[2]++
			}
		}
	}
	// Removals first, then edits, then additions; IDs in the base remain stable
	// until their targeted operation has been applied.
	return append(append(removes, edits...), adds...), counts, unsupported, nil
}

type spatialUnit struct {
	player int
	u      UnitSummary
}

func spatialUnitMap(info *UnitInfo) (map[int]spatialUnit, []int) {
	out := map[int]spatialUnit{}
	var duplicate []int
	for _, section := range info.Sections {
		for _, unit := range section.Units {
			if unit.ReferenceID <= 0 {
				duplicate = append(duplicate, unit.ReferenceID)
				continue
			}
			if _, exists := out[unit.ReferenceID]; exists {
				duplicate = append(duplicate, unit.ReferenceID)
				continue
			}
			out[unit.ReferenceID] = spatialUnit{player: section.Player, u: unit}
		}
	}
	return out, duplicate
}

func recipeAddUnit(s spatialUnit) UnitRecipe {
	u := s.u
	status, frame, captionID := u.Status, u.InitialAnimationFrame, u.CaptionStringID
	return UnitRecipe{Op: "add_unit", Player: s.player, UnitConst: u.UnitConst,
		X: mapPatchFloatPtr(u.X), Y: mapPatchFloatPtr(u.Y), Z: mapPatchFloatPtr(u.Z), Rotation: mapPatchFloatPtr(u.Rotation),
		Status: &status, InitialAnimationFrame: &frame,
		CaptionStringID: &captionID, CaptionString: u.CaptionString}
}

func recipeEditUnit(id int, before, after spatialUnit) (UnitRecipe, bool, string) {
	u, v := before.u, after.u
	if u.CaptionString != v.CaptionString || u.CaptionStringID != v.CaptionStringID {
		return UnitRecipe{}, false, "caption edits are not map-art patch operations"
	}
	op := UnitRecipe{Op: "edit_unit", ReferenceID: mapPatchIntPtr(id)}
	changed := false
	if u.UnitConst != v.UnitConst {
		if v.UnitConst <= 0 {
			return UnitRecipe{}, false, "kit edit_unit cannot set unit_const to zero"
		}
		op.UnitConst, changed = v.UnitConst, true
	}
	if before.player != after.player {
		op.SetPlayer, changed = mapPatchIntPtr(after.player), true
	}
	if !sameFloat(u.X, v.X) {
		op.X, changed = mapPatchFloatPtr(v.X), true
	}
	if !sameFloat(u.Y, v.Y) {
		op.Y, changed = mapPatchFloatPtr(v.Y), true
	}
	if !sameFloat(u.Z, v.Z) {
		op.Z, changed = mapPatchFloatPtr(v.Z), true
	}
	if !sameFloat(u.Rotation, v.Rotation) {
		op.Rotation, changed = mapPatchFloatPtr(v.Rotation), true
	}
	if u.Status != v.Status {
		op.Status, changed = mapPatchIntPtr(v.Status), true
	}
	if u.InitialAnimationFrame != v.InitialAnimationFrame {
		op.InitialAnimationFrame, changed = mapPatchIntPtr(v.InitialAnimationFrame), true
	}
	return op, changed, ""
}

func sameFloat(a, b float64) bool         { return math.Float64bits(a) == math.Float64bits(b) }
func mapPatchFloatPtr(v float64) *float64 { return &v }
func mapPatchIntPtr(v int) *int           { return &v }

func uniqueSortedStrings(values []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value != "" && !seen[value] {
			seen[value] = true
			out = append(out, value)
		}
	}
	sort.Strings(out)
	return out
}

// WriteMapPatchRecipe writes only patchable results, using an atomic rename.
func WriteMapPatchRecipe(path string, report MapPatchDiffReport) error {
	if !report.Patchable {
		return fmt.Errorf("scenario changes are not map-art-patchable: %v", report.UnsupportedChanges)
	}
	data, err := json.MarshalIndent(report.Recipe, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(data, '\n'), 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}
