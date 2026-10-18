package scenario

import (
	"fmt"
	"math"
	"path/filepath"

	"aoe2kit/pkg/felt"
)

// MotifCrop uses half-open tile bounds, matching preview crop coordinates.
type MotifCrop struct {
	X1 int `json:"x1"`
	Y1 int `json:"y1"`
	X2 int `json:"x2"`
	Y2 int `json:"y2"`
}

type MotifPlacement struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  int     `json:"width"`
	Height int     `json:"height"`
}

type RecipeSource struct {
	Kind      string          `json:"kind"`
	Scenario  string          `json:"scenario"`
	Version   string          `json:"version"`
	Crop      MotifCrop       `json:"crop"`
	Relative  bool            `json:"relative"`
	Placement *MotifPlacement `json:"placement,omitempty"`
	Notes     []string        `json:"notes,omitempty"`
}

type ExtractOptions struct {
	Crop              MotifCrop
	Terrain, Relative bool
}

func ExtractRecipeFile(path string, opts ExtractOptions) (Recipe, error) {
	art, err := OpenArtwork(path)
	if err != nil {
		return Recipe{}, err
	}
	return ExtractRecipe(art, filepath.Base(path), opts)
}

// ExtractRecipe copies placed artwork, leaving target-specific unit references
// for the normal recipe writer to allocate. It does not carry gameplay links.
func ExtractRecipe(art *Artwork, name string, opts ExtractOptions) (Recipe, error) {
	c := opts.Crop
	if art == nil || c.X1 < 0 || c.Y1 < 0 || c.X2 > art.Width || c.Y2 > art.Height || c.X1 >= c.X2 || c.Y1 >= c.Y2 {
		return Recipe{}, fmt.Errorf("crop must be a nonempty half-open rectangle inside the map")
	}
	ox, oy := 0, 0
	if opts.Relative {
		ox, oy = c.X1, c.Y1
	}
	r := Recipe{Source: &RecipeSource{Kind: "artwork_crop", Scenario: filepath.Base(name), Version: art.Version, Crop: c, Relative: opts.Relative, Notes: []string{"Placed anchors only, not sprite or footprint extents. Triggers, AI, string-table links and garrison/reference links are not copied. New unit references are allocated when patched. Rotations retain their stored values; no legacy conversion is inferred."}}, Units: []UnitRecipe{}}
	for _, s := range art.Units.Sections {
		for _, u := range s.Units {
			if u.X < float64(c.X1) || u.Y < float64(c.Y1) || u.X >= float64(c.X2) || u.Y >= float64(c.Y2) {
				continue
			}
			if !motifFinite(u.X, u.Y, u.Z, u.Rotation) {
				return Recipe{}, fmt.Errorf("unit %d has a non-finite placement", u.UnitConst)
			}
			r.Units = append(r.Units, UnitRecipe{Op: "add_unit", Player: s.Player, UnitConst: u.UnitConst, X: motifPtr(u.X - float64(ox)), Y: motifPtr(u.Y - float64(oy)), Z: motifPtr(u.Z), Status: motifPtr(u.Status), Rotation: motifPtr(u.Rotation), InitialAnimationFrame: motifPtr(u.InitialAnimationFrame), CaptionString: u.CaptionString, Snap: "fractional"})
		}
	}
	if opts.Terrain {
		if len(art.Tiles) != art.Width*art.Height {
			return Recipe{}, fmt.Errorf("terrain grid is incomplete")
		}
		w, h := c.X2-c.X1, c.Y2-c.Y1
		m := MapRecipe{Op: "terrain_grid", X1: c.X1 - ox, Y1: c.Y1 - oy, Width: motifPtr(w), Height: motifPtr(h)}
		for y := c.Y1; y < c.Y2; y++ {
			for x := c.X1; x < c.X2; x++ {
				t := art.Tiles[y*art.Width+x]
				m.TerrainIDGrid = append(m.TerrainIDGrid, t.TerrainID)
				m.LayerGrid = append(m.LayerGrid, t.Layer)
				m.ElevationGrid = append(m.ElevationGrid, t.Elevation)
			}
		}
		r.Map = []MapRecipe{m}
	}
	if len(r.Units) == 0 && len(r.Map) == 0 {
		return Recipe{}, fmt.Errorf("crop contains no placed units; use --terrain to extract ground only")
	}
	return r, nil
}

type MotifTransform struct {
	At      *PointRecipe
	Rotate  int
	Grammar felt.Catalog
}

// TransformMotif returns a placement recipe without mutating its input. Source provenance
// remains intact; placement records the new frame for subsequent transforms.
func TransformMotif(r Recipe, o MotifTransform) (Recipe, error) {
	s := r.Source
	if s == nil || s.Kind != "artwork_crop" {
		return Recipe{}, fmt.Errorf("--at/--rotate requires an extracted artwork motif")
	}
	if o.Rotate != 0 && o.Rotate != 90 && o.Rotate != 180 && o.Rotate != 270 {
		return Recipe{}, fmt.Errorf("rotation must be 0, 90, 180 or 270")
	}
	if r.Scenario != nil || r.XS != nil || r.Victory != nil || r.Messages != nil || len(r.Players) > 0 || len(r.Diplomacy) > 0 || r.DiplomacyOptions != nil || len(r.Resources) > 0 || len(r.Strings) > 0 || len(r.Variables) > 0 || len(r.Triggers) > 0 || len(r.Masks) > 0 {
		return Recipe{}, fmt.Errorf("motif placement accepts artwork operations only")
	}
	if err := o.Grammar.Validate(); err != nil {
		return Recipe{}, err
	}
	w, h := s.Crop.X2-s.Crop.X1, s.Crop.Y2-s.Crop.Y1
	ox, oy := float64(s.Crop.X1), float64(s.Crop.Y1)
	if s.Relative {
		ox, oy = 0, 0
	}
	if s.Placement != nil {
		ox, oy = s.Placement.X, s.Placement.Y
		w, h = s.Placement.Width, s.Placement.Height
	}
	if w < 1 || h < 1 || w > 1024 || h > 1024 || !motifFinite(ox, oy) {
		return Recipe{}, fmt.Errorf("invalid motif frame")
	}
	tx, ty := ox, oy
	if o.At != nil {
		tx, ty = o.At.X, o.At.Y
	}
	if !motifFinite(tx, ty) || tx < 0 || ty < 0 {
		return Recipe{}, fmt.Errorf("target origin must be finite and nonnegative")
	}
	newW, newH := w, h
	if o.Rotate == 90 || o.Rotate == 270 {
		newW, newH = h, w
	}
	out := r
	out.Units = append([]UnitRecipe(nil), r.Units...)
	out.Map = append([]MapRecipe(nil), r.Map...)
	for i, u := range r.Units {
		if u.Op != "add_unit" || u.X == nil || u.Y == nil || u.ReferenceID != nil || (u.GarrisonedInID != nil && *u.GarrisonedInID >= 0) {
			return Recipe{}, fmt.Errorf("motif units must be positioned add_unit operations without source reference links")
		}
		x, y := *u.X-ox, *u.Y-oy
		if !motifFinite(x, y) || x < 0 || y < 0 || x > float64(w) || y > float64(h) {
			return Recipe{}, fmt.Errorf("unit %d outside motif frame", u.UnitConst)
		}
		switch o.Rotate {
		case 90:
			x, y = float64(h)-y, x
		case 180:
			x, y = float64(w)-x, float64(h)-y
		case 270:
			x, y = y, float64(w)-x
		}
		out.Units[i].X = motifPtr(x + tx)
		out.Units[i].Y = motifPtr(y + ty)
		if o.Rotate != 0 {
			v := 0.0
			if u.Rotation != nil {
				v = *u.Rotation
			}
			if !motifFinite(v) || v < 0 || v > 1e6 || v != math.Trunc(v) {
				return Recipe{}, fmt.Errorf("unit %d has non-integer stored rotation; no variant conversion is established", u.UnitConst)
			}
			rot, err := o.Grammar.Rotate(u.UnitConst, int(v), o.Rotate)
			if err != nil {
				return Recipe{}, err
			}
			out.Units[i].Rotation = motifPtr(float64(rot))
		}
	}
	for i, m := range r.Map {
		if m.Op != "terrain_grid" || m.GridFile != "" || m.Width == nil || m.Height == nil || *m.Width != w || *m.Height != h || float64(m.X1) != ox || float64(m.Y1) != oy || len(m.TerrainIDGrid) != w*h || len(m.LayerGrid) != w*h || (len(m.ElevationGrid) != 0 && len(m.ElevationGrid) != w*h) {
			return Recipe{}, fmt.Errorf("motif terrain must be a complete inline terrain_grid matching its frame")
		}
		if tx != math.Trunc(tx) || ty != math.Trunc(ty) {
			return Recipe{}, fmt.Errorf("terrain motifs require an integer target origin")
		}
		out.Map[i].X1 = int(tx)
		out.Map[i].Y1 = int(ty)
		out.Map[i].Width = motifPtr(newW)
		out.Map[i].Height = motifPtr(newH)
		rotateGrid := func(values []int) []int {
			if len(values) == 0 {
				return nil
			}
			result := make([]int, len(values))
			for y := 0; y < h; y++ {
				for x := 0; x < w; x++ {
					nx, ny := x, y
					switch o.Rotate {
					case 90:
						nx, ny = h-1-y, x
					case 180:
						nx, ny = w-1-x, h-1-y
					case 270:
						nx, ny = y, w-1-x
					}
					result[ny*newW+nx] = values[y*w+x]
				}
			}
			return result
		}
		out.Map[i].TerrainIDGrid = rotateGrid(m.TerrainIDGrid)
		out.Map[i].LayerGrid = rotateGrid(m.LayerGrid)
		out.Map[i].ElevationGrid = rotateGrid(m.ElevationGrid)
	}
	source := *s
	source.Notes = append([]string(nil), s.Notes...)
	source.Placement = &MotifPlacement{tx, ty, newW, newH}
	out.Source = &source
	return out, nil
}

func motifPtr[T any](v T) *T { return &v }
func motifFinite(values ...float64) bool {
	for _, v := range values {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return false
		}
	}
	return true
}
