package geotrace

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
)

// EcologyProfile contains game-facing choices, while geography providers
// supply the evidence that selects where those choices apply. It is generic
// by design: a location profile can replace these palettes without changing
// the geotrace pipeline.
type EcologyProfile struct {
	EnableUnvalidatedUrban bool                     `json:"enable_unvalidated_urban,omitempty"`
	PlayerStarts           *PlayerStartProfile      `json:"player_starts,omitempty"`
	NeutralResources       []NeutralResourceProfile `json:"neutral_resources,omitempty"`
	// Explicit exceptions to the land-flora rule; DAT eligibility still applies.
	FloraWaterTerrains map[int][]int `json:"flora_water_terrains,omitempty"`
	// Notes preserve author intent and provenance; they never affect generation.
	Notes               map[string]json.RawMessage `json:"notes,omitempty"`
	Schema              string                     `json:"schema"`
	Name                string                     `json:"name"`
	CompendiumSHA256    string                     `json:"compendium_sha256,omitempty"`
	ResourceTreeUnits   []int                      `json:"resource_tree_units,omitempty"`
	DecorativeFloraUnit int                        `json:"decorative_flora_unit,omitempty"`
	WaterFloraUnit      int                        `json:"water_flora_unit,omitempty"`
	Fauna               EcologyFaunaProfile        `json:"fauna,omitempty"`
	FishByBand          map[string][]int           `json:"fish_by_band,omitempty"`
	Biome               string                     `json:"biome,omitempty"`
	Palettes            map[string][]WeightedUnit  `json:"palettes,omitempty"`
	SpeciesUnits        map[string][]int           `json:"species_units,omitempty"`
	Overlays            []GeographicOverlay        `json:"overlays,omitempty"`
	Landmarks           []LandmarkCluster          `json:"landmarks,omitempty"`
	CalibrationAxes     []CalibrationAxis          `json:"calibration_axes,omitempty"`
	DiagnosticRegions   []DiagnosticRegion         `json:"diagnostic_regions,omitempty"`
	Layers              []HabitatLayer             `json:"layers"`
	DockUnit            int                        `json:"dock_unit"`
	DockCiv             int                        `json:"dock_civ"`
	TerrainByLandcover  map[string][]int           `json:"terrain_by_landcover,omitempty"`
}

type GeoPoint struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

// Overlays are explicit design additions, not observations from landcover.
type GeographicOverlay struct {
	Name          string     `json:"name"`
	Kind          string     `json:"kind"` // inferred_corridor or admin_line
	Provenance    string     `json:"provenance"`
	Points        []GeoPoint `json:"points"`
	Radius        int        `json:"radius,omitempty"`
	RequireBounds *BBox      `json:"require_bounds,omitempty"`
}

type LandmarkCluster struct {
	Name              string   `json:"name"`
	Provenance        string   `json:"provenance"`
	Bounds            BBox     `json:"bounds"`
	RequireBounds     *BBox    `json:"require_bounds,omitempty"`
	UnitConst         int      `json:"unit_const"`
	LandcoverKeys     []string `json:"landcover_keys,omitempty"`
	TerrainIDs        []int    `json:"terrain_ids,omitempty"`
	AllowedTerrainIDs []int    `json:"allowed_terrain_ids"`
	MinCount          int      `json:"min_count,omitempty"`
	MaxCount          int      `json:"max_count"`
	CellsPerUnit      float64  `json:"cells_per_unit"`
	Density           float64  `json:"density"`
	CenterLimit       int      `json:"center_limit"`
	Spread            float64  `json:"spread"`
	MinDistance       float64  `json:"min_distance"`
}

type CalibrationAxis struct {
	Name string   `json:"name"`
	From GeoPoint `json:"from"`
	To   GeoPoint `json:"to"`
}

type DiagnosticRegion struct {
	Name   string `json:"name"`
	Bounds BBox   `json:"bounds"`
}

type EcologyFaunaProfile struct {
	Deer    int `json:"deer,omitempty"`
	Boar    int `json:"boar,omitempty"`
	Wetland int `json:"wetland,omitempty"`
	Forest  int `json:"forest,omitempty"`
	Swamp   int `json:"swamp,omitempty"`
}

func defaultEcologyProfile() EcologyProfile {
	return EcologyProfile{
		Schema:              "aoe2kit-geotrace-ecology/v1",
		Name:                "generic-default",
		ResourceTreeUnits:   []int{350, 2567, 349},
		DecorativeFloraUnit: 1366,
		WaterFloraUnit:      2536,
		FloraWaterTerrains:  map[int][]int{1144: {54, 55}},
		DockUnit:            45,
		DockCiv:             1,
		Fauna: EcologyFaunaProfile{
			Deer: 65, Boar: 48, Wetland: 1031, Forest: 2089, Swamp: 812,
		},
		FishByBand: map[string][]int{
			"shore":  {69},
			"lake":   {53, 456},
			"bay":    {53, 456, 458, 69},
			"medium": {458, 457},
			"deep":   {457, 450, 2625},
		},
	}
}

func loadEcologyProfile(path string) (EcologyProfile, error) {
	profile := defaultEcologyProfile()
	if path == "" {
		return profile, nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return EcologyProfile{}, fmt.Errorf("read ecology profile: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&profile); err != nil {
		return EcologyProfile{}, fmt.Errorf("parse ecology profile: %w", err)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return EcologyProfile{}, fmt.Errorf("ecology profile must contain one JSON object")
	}
	if profile.Schema == "" {
		profile.Schema = "aoe2kit-geotrace-ecology/v1"
	}
	if profile.Name == "" {
		profile.Name = "unnamed"
	}
	if len(profile.ResourceTreeUnits) == 0 {
		profile.ResourceTreeUnits = defaultEcologyProfile().ResourceTreeUnits
	}
	if profile.DecorativeFloraUnit == 0 {
		profile.DecorativeFloraUnit = 1366
	}
	if profile.WaterFloraUnit == 0 {
		profile.WaterFloraUnit = 2536
	}
	if profile.Fauna.Deer == 0 {
		profile.Fauna = defaultEcologyProfile().Fauna
	}
	if len(profile.FishByBand) == 0 {
		profile.FishByBand = defaultEcologyProfile().FishByBand
	}
	if err := validateEcologyProfile(profile); err != nil {
		return EcologyProfile{}, err
	}
	return profile, nil
}

func validateEcologyProfile(p EcologyProfile) error {
	if err := validateNeutralResources(p.NeutralResources); err != nil {
		return err
	}
	if _, err := normalizedStarts(p.PlayerStarts); err != nil {
		return err
	}
	for unit, terrains := range p.FloraWaterTerrains {
		if unit <= 0 || len(terrains) == 0 {
			return fmt.Errorf("invalid flora water exception for unit %d", unit)
		}
		for _, id := range terrains {
			if !placementWaterTerrain(id) {
				return fmt.Errorf("flora water exception terrain %d is not water", id)
			}
		}
	}
	for species, units := range p.SpeciesUnits {
		if species == "" || len(units) == 0 {
			return fmt.Errorf("empty species mapping %q", species)
		}
		for _, id := range units {
			if id <= 0 {
				return fmt.Errorf("invalid species unit %d", id)
			}
		}
	}
	for role, units := range p.Palettes {
		if !validEcologyRole(role) || !validPalette(units) {
			return fmt.Errorf("invalid ecology palette %q", role)
		}
	}
	for key, ids := range p.TerrainByLandcover {
		if key == "" || len(ids) == 0 || len(ids) > 2 {
			return fmt.Errorf("terrain palette %q needs one or two terrain ids", key)
		}
		for _, id := range ids {
			if id < 0 {
				return fmt.Errorf("negative terrain id in palette %q", key)
			}
		}
	}
	if p.DockUnit <= 0 || p.DockCiv < 1 {
		return fmt.Errorf("dock unit and civilization must be positive")
	}
	if p.Schema != "aoe2kit-geotrace-ecology/v1" {
		return fmt.Errorf("unsupported ecology profile schema %q", p.Schema)
	}
	for _, id := range append(append([]int{}, p.ResourceTreeUnits...), p.DecorativeFloraUnit, p.WaterFloraUnit, p.Fauna.Deer, p.Fauna.Boar, p.Fauna.Wetland, p.Fauna.Forest, p.Fauna.Swamp) {
		if id <= 0 {
			return fmt.Errorf("profile unit id must be positive: %d", id)
		}
	}
	for band, ids := range p.FishByBand {
		switch band {
		case "shore", "lake", "bay", "medium", "deep":
		default:
			return fmt.Errorf("unknown fish band %q", band)
		}
		for _, id := range ids {
			if id <= 0 {
				return fmt.Errorf("fish unit id must be positive: %d", id)
			}
		}
	}
	validPoint := func(p GeoPoint) bool {
		return finite(p.Lat) && finite(p.Lon) && p.Lat >= -90 && p.Lat <= 90 && p.Lon >= -180 && p.Lon <= 180
	}
	validBounds := func(b *BBox) bool {
		return b == nil || (validPoint(GeoPoint{b.MinLat, b.MinLon}) && validPoint(GeoPoint{b.MaxLat, b.MaxLon}) && b.MinLat < b.MaxLat && b.MinLon < b.MaxLon)
	}
	for _, o := range p.Overlays {
		if o.Name == "" || o.Provenance == "" || (o.Kind != "inferred_corridor" && o.Kind != "admin_line") || len(o.Points) < 2 || o.Radius < 0 || o.Radius > 16 || !validBounds(o.RequireBounds) {
			return fmt.Errorf("invalid overlay %q: name, provenance, kind, points and valid bounds required", o.Name)
		}
		for _, point := range o.Points {
			if !validPoint(point) {
				return fmt.Errorf("invalid coordinate in overlay %q", o.Name)
			}
		}
	}
	for _, l := range p.Landmarks {
		if l.Name == "" || l.Provenance == "" || !validBounds(&l.Bounds) || !validBounds(l.RequireBounds) || l.UnitConst <= 0 || l.CellsPerUnit <= 0 || !finite(l.CellsPerUnit) || l.MinCount < 0 || l.MaxCount < l.MinCount || l.MaxCount > 100000 || l.CenterLimit <= 0 || l.Spread <= 0 || !finite(l.Spread) || l.MinDistance < 0 || !finite(l.MinDistance) || l.Density <= 0 || l.Density > 10 || len(l.AllowedTerrainIDs) == 0 {
			return fmt.Errorf("invalid landmark cluster %q", l.Name)
		}
	}
	for _, a := range p.CalibrationAxes {
		if a.Name == "" || !validPoint(a.From) || !validPoint(a.To) {
			return fmt.Errorf("invalid calibration axis %q", a.Name)
		}
	}
	for _, r := range p.DiagnosticRegions {
		if r.Name == "" || !validBounds(&r.Bounds) {
			return fmt.Errorf("invalid diagnostic region %q", r.Name)
		}
	}
	names := map[string]bool{}
	for _, l := range p.Layers {
		if (l.Role == "building" || l.PopulationCenterOnly || l.PopulationCenterLayer != "") && !p.EnableUnvalidatedUrban {
			return fmt.Errorf("urban layer %q requires enable_unvalidated_urban; engine behavior is not validated", l.Name)
		}
		if !validBounds(l.Bounds) || !finite(l.MinShoreDistance) || !finite(l.MaxShoreDistance) || l.MinShoreDistance < 0 || l.MaxShoreDistance < 0 || (l.MaxShoreDistance > 0 && l.MaxShoreDistance < l.MinShoreDistance) {
			return fmt.Errorf("invalid geographic habitat bounds %q", l.Name)
		}
		if l.Name == "" || names[l.Name] {
			return fmt.Errorf("habitat layers need unique names: %q", l.Name)
		}
		names[l.Name] = true
		switch l.Role {
		case "resource", "decorative", "fauna", "fish", "building":
		default:
			return fmt.Errorf("invalid habitat role %q", l.Role)
		}
		if l.EcologyRole != "" && !validEcologyRole(l.EcologyRole) {
			return fmt.Errorf("invalid ecology role %q", l.EcologyRole)
		}
		units := l.Units
		if len(units) == 0 {
			units = p.Palettes[l.EcologyRole]
		}
		if !finite(l.Rate) || l.Rate < 0 || l.Rate > 16 || l.Limit < 0 || l.Limit > 100000 || !finite(l.Spacing) || l.Spacing < 0 || !finite(l.PatchScale) || l.PatchScale < 0 || (len(l.Felt) == 0 && !validPalette(units)) {
			return fmt.Errorf("invalid habitat layer %q", l.Name)
		}
		if l.Role == "building" {
			if len(l.Felt) > 0 || l.ClusterCount <= 0 || l.ClusterSize < 2 || !finite(l.ClusterRadius) || l.ClusterRadius <= 0 || !finite(l.ClusterSpacing) || l.ClusterSpacing <= 0 || l.FootprintRadius < 0 || l.Spacing < float64(2*l.FootprintRadius+1) {
				return fmt.Errorf("building layer %q needs explicit cluster and footprint settings", l.Name)
			}
		} else if l.ClusterCount != 0 || l.ClusterSize != 0 || l.ClusterRadius != 0 || l.ClusterSpacing != 0 || l.FootprintRadius != 0 {
			return fmt.Errorf("cluster settings are only valid for building layer %q", l.Name)
		}
		switch l.PopulationCenterLayer {
		case "", "built_up_core", "road_corridor":
		default:
			return fmt.Errorf("unknown population-center layer %q", l.PopulationCenterLayer)
		}
		for _, selection := range l.Felt {
			if !finite(selection.Weight) || selection.Weight <= 0 || len(selection.Functions) == 0 {
				return fmt.Errorf("felt selection in %q needs functions and a positive weight", l.Name)
			}
		}
		for _, u := range l.Units {
			if u.UnitConst <= 0 || !finite(u.Weight) || u.Weight <= 0 {
				return fmt.Errorf("invalid palette in layer %q", l.Name)
			}
		}
		switch l.WaterBand {
		case "", "shore", "lake", "bay", "medium", "deep":
		default:
			return fmt.Errorf("invalid water band in layer %q", l.Name)
		}
	}
	return nil
}

func validEcologyRole(role string) bool {
	switch role {
	case "NERITIC", "SALTWATER", "FRESHWATER", "OYSTER", "RIPARIAN", "WHALE", "PREDATOR", "HUNTABLE", "HUNTABLE_SMALL", "LUREABLE", "FURBEARER", "HERDABLE", "FORAGE_PLANT", "BIRD":
		return true
	}
	return false
}

func validPalette(units []WeightedUnit) bool {
	total := 0.0
	for _, u := range units {
		if u.UnitConst <= 0 || !finite(u.Weight) || u.Weight <= 0 {
			return false
		}
		total += u.Weight
	}
	return len(units) > 0 && finite(total)
}

func finite(n float64) bool { return !math.IsNaN(n) && !math.IsInf(n, 0) }

func profileFish(profile EcologyProfile, band string) []int {
	if units := profile.FishByBand[band]; len(units) > 0 {
		return units
	}
	return defaultEcologyProfile().FishByBand[band]
}
