// Package compose builds deterministic terrain and unit recipes from reusable
// art primitives. These are preview-derived composition recipes, not physics,
// pathing, passability or engine-verified visual guarantees. No art is bundled.
package compose

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"

	"aoe2kit/pkg/scenario"
)

type Point struct{ X, Y float64 }
type Placement struct {
	UnitID, Player, Variant, Status int
	X, Y, Z                         float64
}
type Grid struct {
	Width     int   `json:"width"`
	Height    int   `json:"height"`
	Terrain   []int `json:"terrain_id"`
	Layer     []int `json:"layer"`
	Elevation []int `json:"elevation"`
}
type reservation struct {
	p      Point
	radius float64
}

// Scene uses sticky errors: invalid placement never silently disappears, and a
// failed scene cannot export. Claims are circular art spacing, not collision.
type Scene struct {
	width, height             int
	terrain, layer, elevation []int
	units                     []Placement
	claims                    []reservation
	err                       error
	Random                    *RNG
}

func New(width, height, terrain int, seed uint64) (*Scene, error) {
	if width < 1 || height < 1 || width > 480 || height > 480 || terrain < 0 || terrain > 255 {
		return nil, fmt.Errorf("compose: invalid dimensions or terrain")
	}
	s := &Scene{width: width, height: height, terrain: make([]int, width*height), layer: make([]int, width*height), elevation: make([]int, width*height), Random: NewRNG(seed)}
	for i := range s.terrain {
		s.terrain[i] = terrain
		s.layer[i] = -1
	}
	return s, nil
}
func (s *Scene) Err() error { return s.err }
func (s *Scene) fail(message string) {
	if s.err == nil {
		s.err = fmt.Errorf("compose: %s", message)
	}
}
func finite(values ...float64) bool {
	for _, v := range values {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return false
		}
	}
	return true
}
func (s *Scene) inside(x, y float64) bool {
	return finite(x, y) && x >= 0 && y >= 0 && x < float64(s.width) && y < float64(s.height)
}
func (s *Scene) Dimensions() (int, int) { return s.width, s.height }
func (s *Scene) TerrainAt(x, y int) (int, bool) {
	if x < 0 || y < 0 || x >= s.width || y >= s.height {
		return 0, false
	}
	return s.terrain[y*s.width+x], true
}
func (s *Scene) SetTerrain(x, y, id int) {
	if s.err != nil {
		return
	}
	if _, ok := s.TerrainAt(x, y); !ok || id < 0 || id > 255 {
		s.fail("terrain coordinate or id out of range")
		return
	}
	s.terrain[y*s.width+x] = id
}
func (s *Scene) SetLayer(x, y, id int) {
	if s.err != nil {
		return
	}
	if _, ok := s.TerrainAt(x, y); !ok || id < -1 || id > 32767 {
		s.fail("layer coordinate or id out of range")
		return
	}
	s.layer[y*s.width+x] = id
}
func (s *Scene) SetElevation(x, y, value int) {
	if s.err != nil {
		return
	}
	if _, ok := s.TerrainAt(x, y); !ok || value < 0 || value > 255 {
		s.fail("elevation coordinate or value out of range")
		return
	}
	s.elevation[y*s.width+x] = value
}
func (s *Scene) Put(p Placement) {
	if s.err != nil {
		return
	}
	if !s.inside(p.X, p.Y) || !finite(p.Z) || p.UnitID < 0 || p.UnitID > 65535 || p.Player < 0 || p.Player > 8 || p.Variant < 0 || p.Variant > 65535 || p.Status < 0 || p.Status > 255 {
		s.fail(fmt.Sprintf("invalid placement: %+v", p))
		return
	}
	if len(s.units) >= 1000000 {
		s.fail("unit budget exceeded")
		return
	}
	s.units = append(s.units, p)
}

// Gaia places an explicit visual index, never radians. Validate unit-specific
// variant availability using the selected DAT/art before compiling the recipe.
func (s *Scene) Gaia(id int, p Point, variant int) {
	s.Put(Placement{UnitID: id, X: p.X, Y: p.Y, Variant: variant, Status: 2})
}
func (s *Scene) Claim(p Point, radius float64) {
	if s.err != nil {
		return
	}
	if !s.inside(p.X, p.Y) || !finite(radius) || radius < 0 {
		s.fail("invalid spacing claim")
		return
	}
	s.claims = append(s.claims, reservation{p, radius})
}

// Free tests a candidate point against reserved radii; touching is allowed.
func (s *Scene) Free(p Point) bool { return s.FreeRadius(p, 0) }
func (s *Scene) FreeRadius(p Point, radius float64) bool {
	if s.err != nil || !s.inside(p.X, p.Y) || !finite(radius) || radius < 0 {
		return false
	}
	for _, c := range s.claims {
		if math.Hypot(p.X-c.p.X, p.Y-c.p.Y) < c.radius+radius {
			return false
		}
	}
	return true
}
func (s *Scene) Grid() Grid {
	return Grid{s.width, s.height, append([]int(nil), s.terrain...), append([]int(nil), s.layer...), append([]int(nil), s.elevation...)}
}
func (s *Scene) Units() []Placement { return append([]Placement(nil), s.units...) }
func ptr[T any](v T) *T             { return &v }

// Recipe returns Kit's existing typed recipe, including an in-memory grid.
// The target scenario must already have the same dimensions. It is not reset:
// existing units remain unless the caller explicitly adds a removal operation.
func (s *Scene) Recipe() (scenario.Recipe, error) {
	if s.err != nil {
		return scenario.Recipe{}, s.err
	}
	g := s.Grid()
	r := scenario.Recipe{Triggers: []scenario.TriggerRecipe{}, Map: []scenario.MapRecipe{{Op: "terrain_grid", Width: ptr(s.width), Height: ptr(s.height), TerrainIDGrid: g.Terrain, LayerGrid: g.Layer, ElevationGrid: g.Elevation}}}
	for _, u := range s.units {
		r.Units = append(r.Units, scenario.UnitRecipe{Op: "add_unit", Player: u.Player, UnitConst: u.UnitID, X: ptr(u.X), Y: ptr(u.Y), Z: ptr(u.Z), Status: ptr(u.Status), Rotation: ptr(float64(u.Variant)), Snap: "fractional"})
	}
	return r, nil
}

// WriteBundle writes recipe.json and grid.json into a NEW directory. Refusing
// an existing directory protects previous critic rounds from partial overwrite.
func (s *Scene) WriteBundle(dir string) error {
	r, err := s.Recipe()
	if err != nil {
		return err
	}
	r.Map[0].GridFile = "grid.json"
	r.Map[0].TerrainIDGrid = nil
	r.Map[0].LayerGrid = nil
	r.Map[0].ElevationGrid = nil
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	g, err := json.Marshal(s.Grid())
	if err != nil {
		return err
	}
	if err := os.Mkdir(dir, 0755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "grid.json"), g, 0644); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "recipe.json"), b, 0644)
}

type Brush struct {
	UnitID   int
	Variants []int
}

func (s *Scene) stamp(b Brush, p Point) {
	if len(b.Variants) == 0 {
		s.fail("brush has no explicit variants")
		return
	}
	s.Gaia(b.UnitID, p, b.Variants[s.Random.Intn(len(b.Variants))])
}
