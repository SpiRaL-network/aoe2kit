package geotrace

import (
	"hash/fnv"
	"math"
	"sort"

	"aoe2kit/pkg/scenario"
)

type WeightedUnit struct {
	UnitConst int     `json:"unit_const"`
	Weight    float64 `json:"weight"`
}

// Layers describe habitat and density independently of a particular region.
// Rate is objects per eligible tile, scaled by the corresponding CLI density.
type HabitatLayer struct {
	Name                  string            `json:"name"`
	Role                  string            `json:"role"`
	EcologyRole           string            `json:"ecology_role,omitempty"`
	Bounds                *BBox             `json:"bounds,omitempty"`
	MinShoreDistance      float64           `json:"min_shore_distance,omitempty"`
	MaxShoreDistance      float64           `json:"max_shore_distance,omitempty"`
	AllowWaterBuffer      bool              `json:"allow_water_buffer,omitempty"`
	LandcoverKeys         []string          `json:"landcover_keys,omitempty"`
	Climates              []string          `json:"climates,omitempty"`
	WaterBand             string            `json:"water_band,omitempty"`
	TerrainIDs            []int             `json:"terrain_ids,omitempty"`
	Units                 []WeightedUnit    `json:"units"`
	Felt                  []FeltSelection   `json:"felt,omitempty"`
	FeltEntries           map[int]feltEntry `json:"felt_entries,omitempty"`
	Rate                  float64           `json:"rate"`
	Limit                 int               `json:"limit"`
	Spacing               float64           `json:"spacing,omitempty"`
	PatchScale            float64           `json:"patch_scale,omitempty"`
	PopulationCenterOnly  bool              `json:"population_center_only,omitempty"`
	PopulationCenterLayer string            `json:"population_center_layer,omitempty"`
	ClusterCount          int               `json:"cluster_count,omitempty"`
	ClusterSize           int               `json:"cluster_size,omitempty"`
	ClusterRadius         float64           `json:"cluster_radius,omitempty"`
	ClusterSpacing        float64           `json:"cluster_spacing,omitempty"`
	FootprintRadius       int               `json:"footprint_radius,omitempty"`
}

type LayerDiagnostics struct {
	Name                    string         `json:"name"`
	Role                    string         `json:"role"`
	EligibleCells           int            `json:"eligible_cells"`
	Requested               int            `json:"requested"`
	Emitted                 int            `json:"emitted"`
	TruncatedBySpacing      int            `json:"truncated_by_spacing"`
	UnitCounts              map[int]int    `json:"unit_counts"`
	FamilyCounts            map[string]int `json:"family_counts,omitempty"`
	AmphibiousEligibleCells int            `json:"amphibious_eligible_cells,omitempty"`
	AmphibiousEmitted       int            `json:"amphibious_emitted,omitempty"`
	ClustersEmitted         int            `json:"clusters_emitted,omitempty"`
	FootprintTiles          int            `json:"footprint_tiles,omitempty"`
}

func equalWeights(ids []int) []WeightedUnit {
	out := make([]WeightedUnit, len(ids))
	for i, id := range ids {
		out[i] = WeightedUnit{UnitConst: id, Weight: 1}
	}
	return out
}

func defaultHabitatLayers(p EcologyProfile, hasLandcover bool) []HabitatLayer {
	forestKeys := []string{"10_tree_cover"}
	forestRate := 0.9
	openKeys := []string{"10_tree_cover", "20_shrubland", "30_grassland"}
	if !hasLandcover {
		forestKeys = nil
		openKeys = nil
		forestRate = 0.15
	}
	layers := []HabitatLayer{
		{Name: "forest_resource", Role: "resource", LandcoverKeys: forestKeys, Units: equalWeights(p.ResourceTreeUnits), Rate: forestRate, Limit: 40000, PatchScale: 8},
		{Name: "ground_decoration", Role: "decorative", Units: equalWeights([]int{p.DecorativeFloraUnit}), Rate: 0.4, Limit: 20000, Spacing: 0.35, PatchScale: 3},
		{Name: "wetland_decoration", Role: "decorative", LandcoverKeys: []string{"90_herbaceous_wetland", "95_mangrove"}, Units: equalWeights([]int{p.WaterFloraUnit}), Rate: 0.6, Limit: 3000, Spacing: 0.35},
		{Name: "open_hunt", Role: "fauna", LandcoverKeys: openKeys, Units: equalWeights([]int{p.Fauna.Deer}), Rate: 1.0 / 100, Limit: 260, Spacing: 5},
		{Name: "forest_hunt", Role: "fauna", LandcoverKeys: forestKeys, Units: equalWeights([]int{p.Fauna.Boar}), Rate: 1.0 / 300, Limit: 95, Spacing: 8},
		{Name: "wetland_fauna", Role: "fauna", LandcoverKeys: []string{"90_herbaceous_wetland", "95_mangrove"}, Units: equalWeights([]int{p.Fauna.Wetland}), Rate: 1.0 / 120, Limit: 36, Spacing: 7},
		{Name: "forest_fauna", Role: "fauna", LandcoverKeys: forestKeys, Units: equalWeights([]int{p.Fauna.Forest}), Rate: 1.0 / 650, Limit: 28, Spacing: 10},
	}
	for i, role := range []string{"", "", "RIPARIAN", "HUNTABLE_SMALL", "LUREABLE", "PREDATOR", "FURBEARER"} {
		layers[i].EcologyRole = role
		if units := p.Palettes[role]; len(units) > 0 {
			layers[i].Units = units
		}
	}
	for i, band := range []string{"shore", "lake", "medium", "deep"} {
		role := []string{"NERITIC", "FRESHWATER", "SALTWATER", "SALTWATER"}[i]
		units := p.Palettes[role]
		if len(units) == 0 {
			units = equalWeights(profileFish(p, band))
		}
		layers = append(layers, HabitatLayer{Name: band + "_fish", Role: "fish", EcologyRole: role, WaterBand: band, Units: units, Rate: []float64{1.0 / 24, 1.0 / 30, 1.0 / 300, 1.0 / 400}[i], Limit: []int{200, 100, 50, 35}[i], Spacing: []float64{26, 10, 10, 10}[i]})
	}
	units := p.Palettes["FRESHWATER"]
	if len(units) == 0 {
		units = equalWeights(profileFish(p, "bay"))
	}
	layers = append(layers, HabitatLayer{Name: "bay_fish", Role: "fish", EcologyRole: "FRESHWATER", WaterBand: "bay", Units: units, Rate: 1.0 / 30, Limit: 200, Spacing: 8})
	return layers
}

func ensurePopulationCenterLayer(p *EcologyProfile) {
	if p == nil || p.Layers == nil || !p.EnableUnvalidatedUrban {
		return
	}
	for _, layer := range p.Layers {
		if layer.PopulationCenterOnly || layer.PopulationCenterLayer != "" {
			return
		}
	}
	p.Layers = append(p.Layers, HabitatLayer{
		Name: "population_center_felt", Role: "decorative", PopulationCenterOnly: true,
		Felt: []FeltSelection{{Functions: []string{"eyecandy"}, Tags: []string{"flowerbed"}, Units: []int{859}, Weight: 1}},
		Rate: 0.035, Limit: 2400, Spacing: 2,
	})
}

func layerSeed(name string, seed int) int {
	h := fnv.New32a()
	_, _ = h.Write([]byte(name))
	return int(h.Sum32()&0x7fffffff) ^ seed
}

func member[T comparable](list []T, value T) bool {
	for _, v := range list {
		if v == value {
			return true
		}
	}
	return false
}

func waterBand(cell Cell) string {
	if cell.Land == nil || *cell.Land {
		return ""
	}
	if cell.InlandWater {
		return "lake"
	}
	if cell.ShelteredWater {
		return "bay"
	}
	if cell.DistanceToShoreTiles <= 8 {
		return "shore"
	}
	if cell.DistanceToShoreTiles < 20 {
		return "medium"
	}
	return "deep"
}

// Population centers are inferred from the generic built-up land-cover class
// and dense mapped streets; this does not encode any regional coordinates.
func markPopulationCenters(report *Report) {
	if report == nil || report.GridSize <= 0 || len(report.Cells) != report.GridSize*report.GridSize {
		return
	}
	n := report.GridSize
	for i := range report.Cells {
		cell := &report.Cells[i]
		builtUp := cell.LandcoverClass == 50 || cell.SourceLandcoverClass == 50 || cell.LandcoverKey == "50_built_up"
		cell.BuiltUpCore = builtUp && cell.Land != nil && *cell.Land && cellBuildableForGaiaPlacement(*cell)
		if builtUp {
			report.PopulationCenterBuiltUpCells++
		}
		roads, total := 0, 0
		for y := maxInt(0, cell.Y-4); y <= minInt(n-1, cell.Y+4); y++ {
			for x := maxInt(0, cell.X-4); x <= minInt(n-1, cell.X+4); x++ {
				total++
				if report.Cells[y*n+x].Road {
					roads++
				}
			}
		}
		denseRoads := roads >= 9 && float64(roads)/float64(total) >= 0.11
		if denseRoads {
			report.PopulationCenterRoadDensityCells++
		}
		cell.RoadCorridor = cell.Roadside && !cell.BuiltUpCore && cell.Land != nil && *cell.Land && cellBuildableForGaiaPlacement(*cell)
		cell.PopulationCenter = builtUp || denseRoads
		if cell.PopulationCenter {
			report.PopulationCenterCells++
		}
	}
}

func matchesLayer(cell Cell, layer HabitatLayer) bool {
	roadsideAllowed := layer.PopulationCenterLayer == "road_corridor" && layer.Role == "decorative"
	if cell.Road || (cell.Roadside && len(layer.Felt) == 0 && !roadsideAllowed) || cell.Border || cell.Causeway {
		return false
	}
	if layer.Bounds != nil && !bboxContains(*layer.Bounds, cell.Lat, cell.Lon) {
		return false
	}
	if layer.PopulationCenterOnly && !cell.PopulationCenter {
		return false
	}
	switch layer.PopulationCenterLayer {
	case "":
	case "built_up_core":
		if !cell.BuiltUpCore {
			return false
		}
	case "road_corridor":
		if !cell.RoadCorridor {
			return false
		}
	default:
		return false
	}
	if cell.DistanceToShoreTiles < layer.MinShoreDistance || (layer.MaxShoreDistance > 0 && cell.DistanceToShoreTiles > layer.MaxShoreDistance) {
		return false
	}
	if layer.WaterBand != "" {
		if waterBand(cell) != layer.WaterBand {
			return false
		}
	} else {
		surfaceOK := cell.Land != nil && *cell.Land
		if layer.Role == "resource" || layer.Role == "decorative" {
			surfaceOK = floraSurface(cell)
		}
		if !surfaceOK || !cellBuildableForGaiaPlacement(cell) {
			return false
		}
		if layer.Role == "resource" && cell.WaterBuffer && !layer.AllowWaterBuffer {
			return false
		}
	}
	if len(layer.LandcoverKeys) > 0 && !member(layer.LandcoverKeys, cell.LandcoverKey) {
		return false
	}
	if len(layer.Climates) > 0 && !member(layer.Climates, cell.ClimateZone) {
		return false
	}
	return len(layer.TerrainIDs) == 0 || member(layer.TerrainIDs, cell.TerrainID)
}

func layerDensity(r Report, l HabitatLayer) float64 {
	switch l.Role {
	case "fauna":
		return normalizeDensity(r.LandFaunaDensity) / 10
	case "fish":
		if l.WaterBand == "shore" || l.WaterBand == "lake" {
			return normalizeDensity(firstPositive(r.ShoreFishDensity, r.FishDensity)) / 10
		}
		return normalizeDensity(firstPositive(r.OceanFishDensity, r.FishDensity)) / 10
	case "building":
		return 1
	default:
		return normalizeDensity(r.FloraDensity) / 10
	}
}

func weightedChoice(units []WeightedUnit, value float64) int {
	total := 0.0
	for _, u := range units {
		total += u.Weight
	}
	value *= total
	for _, u := range units {
		value -= u.Weight
		if value < 0 {
			return u.UnitConst
		}
	}
	return units[len(units)-1].UnitConst
}

// Interpolated noise selects connected, irregular canopy patches. Species
// selection uses an independent per-tile sample within each patch.
func patchValue(x, y int, scale float64, seed int) float64 {
	if scale <= 0 {
		return hashUnitFloat(x, y, seed)
	}
	fx, fy := float64(x)/scale, float64(y)/scale
	x0, y0 := int(math.Floor(fx)), int(math.Floor(fy))
	tx, ty := fx-float64(x0), fy-float64(y0)
	tx = tx * tx * (3 - 2*tx)
	ty = ty * ty * (3 - 2*ty)
	a := hashUnitFloat(x0, y0, seed)*(1-tx) + hashUnitFloat(x0+1, y0, seed)*tx
	b := hashUnitFloat(x0, y0+1, seed)*(1-tx) + hashUnitFloat(x0+1, y0+1, seed)*tx
	return 0.8*(a*(1-ty)+b*ty) + 0.2*hashUnitFloat(x, y, seed+41)
}

type ecologyCandidate struct {
	cell       Cell
	rank       float64
	amphibious bool
}
type ecologyPoint struct{ x, y float64 }

func buildHabitatEcology(r Report, p EcologyProfile) ([]scenario.UnitRecipe, []LayerDiagnostics) {
	layers := p.Layers
	if layers == nil {
		hasCover := false
		for _, c := range r.Cells {
			if c.LandcoverKey != "" {
				hasCover = true
				break
			}
		}
		layers = defaultHabitatLayers(p, hasCover)
	}
	hasPopulationCenterLayer := false
	for _, layer := range layers {
		hasPopulationCenterLayer = hasPopulationCenterLayer || layer.PopulationCenterOnly || layer.PopulationCenterLayer != ""
	}
	if p.EnableUnvalidatedUrban && !hasPopulationCenterLayer && layers != nil && len(layers) > 0 {
		layers = append(append([]HabitatLayer(nil), layers...), HabitatLayer{
			Name: "population_center_felt", Role: "decorative", PopulationCenterOnly: true,
			Units: equalWeights([]int{859}), Rate: 0.035, Limit: 2400, Spacing: 2,
		})
	}
	var out []scenario.UnitRecipe
	// Reserve vegetation before fauna. The serialized-file sweep subsequently
	// checks full DAT footprints and final terrain, including later overlays.
	layers = append([]HabitatLayer(nil), layers...)
	sort.SliceStable(layers, func(i, j int) bool {
		if (layers[i].Role == "building") != (layers[j].Role == "building") {
			return layers[i].Role == "building"
		}
		animal := func(l HabitatLayer) bool { return l.Role == "fauna" || l.Role == "fish" }
		return !animal(layers[i]) && animal(layers[j])
	})
	var diagnostics []LayerDiagnostics
	resourceOccupied := map[[2]int]bool{}
	buildingFootprints := map[[2]int]bool{}
	var fishPoints []struct{ x, y, spacing float64 }
	for _, layer := range layers {
		if len(layer.Units) == 0 {
			layer.Units = p.Palettes[layer.EcologyRole]
		}
		seed := layerSeed(layer.Name, r.Seed)
		candidates := make([]ecologyCandidate, 0)
		for _, c := range r.Cells {
			if matchesLayer(c, layer) && len(habitatUnits(c, layer)) > 0 {
				amphibious := isAmphibiousTerrain(c.TerrainID)
				candidates = append(candidates, ecologyCandidate{cell: c, rank: patchValue(c.X, c.Y, layer.PatchScale, seed), amphibious: amphibious})
			}
		}
		sort.Slice(candidates, func(i, j int) bool {
			if candidates[i].rank != candidates[j].rank {
				return candidates[i].rank > candidates[j].rank
			}
			if candidates[i].cell.Y != candidates[j].cell.Y {
				return candidates[i].cell.Y < candidates[j].cell.Y
			}
			return candidates[i].cell.X < candidates[j].cell.X
		})
		count := minInt(layer.Limit, int(math.Round(float64(len(candidates))*layer.Rate*layerDensity(r, layer))))
		if layer.Role == "building" {
			count = minInt(layer.Limit, layer.ClusterCount*layer.ClusterSize)
		}
		if layer.Role != "decorative" {
			count = minInt(count, len(candidates))
		}
		d := LayerDiagnostics{Name: layer.Name, Role: layer.Role, EligibleCells: len(candidates), Requested: count, UnitCounts: map[int]int{}, FamilyCounts: map[string]int{}}
		for _, candidate := range candidates {
			if candidate.amphibious {
				d.AmphibiousEligibleCells++
			}
		}
		if layer.Role == "building" {
			emitted, clusters := emitBuildingClusters(r, layer, candidates, seed, buildingFootprints, &d)
			out = append(out, emitted...)
			d.ClustersEmitted = clusters
			d.TruncatedBySpacing = d.Requested - d.Emitted
			diagnostics = append(diagnostics, d)
			continue
		}
		spacing := layer.Spacing
		if layer.Role == "resource" {
			spacing = math.Max(1, spacing)
		}
		bins := map[[2]int][]ecologyPoint{}
		passes := 1
		if len(candidates) > 0 && count > len(candidates) {
			passes = (count + len(candidates) - 1) / len(candidates)
		}
		for pass := 0; pass < passes && d.Emitted < count; pass++ {
			for _, candidate := range candidates {
				if d.Emitted >= count {
					break
				}
				c := candidate.cell
				if buildingFootprints[[2]int{c.X, c.Y}] {
					continue
				}
				if (layer.Role == "resource" || layer.Role == "fauna" || layer.Role == "fish") && resourceOccupied[[2]int{c.X, c.Y}] {
					continue
				}
				x, y := float64(c.X)+0.5, float64(c.Y)+0.5
				snap := "grid"
				if layer.Role == "decorative" {
					snap = "auto"
					x += (hashUnitFloat(c.X, c.Y, seed+pass*31+7) - 0.5) * 0.8
					y += (hashUnitFloat(c.X, c.Y, seed+pass*31+13) - 0.5) * 0.8
				}
				bucket := [2]int{}
				if spacing > 0 {
					bucket = [2]int{int(math.Floor(x / spacing)), int(math.Floor(y / spacing))}
					close := false
					for dy := -1; dy <= 1; dy++ {
						for dx := -1; dx <= 1; dx++ {
							for _, v := range bins[[2]int{bucket[0] + dx, bucket[1] + dy}] {
								if squaredDistance(x, y, v.x, v.y) < spacing*spacing {
									close = true
								}
							}
						}
					}
					if close {
						continue
					}
				}
				available := habitatUnits(c, layer)
				unit := weightedChoice(available, hashUnitFloat(c.X, c.Y, seed+pass*31+73))
				if layer.Role == "fish" && layer.EcologyRole != "OYSTER" {
					close := false
					for _, other := range fishPoints {
						distance := math.Max(spacing, other.spacing)
						if squaredDistance(x, y, other.x, other.y) < distance*distance {
							close = true
							break
						}
					}
					if close {
						continue
					}
					fishPoints = append(fishPoints, struct{ x, y, spacing float64 }{x, y, spacing})
				}
				if layer.Role == "resource" {
					choices := p.woodyUnitChoices(c.USGSWoodySpecies)
					if len(choices) == 0 {
						choices = c.SpeciesUnitConsts
					}
					var supported []WeightedUnit
					for _, option := range available {
						if member(choices, option.UnitConst) {
							supported = append(supported, option)
						}
					}
					if len(supported) > 0 {
						unit = weightedChoice(supported, hashUnitFloat(c.X, c.Y, seed+73))
					}
					resourceOccupied[[2]int{c.X, c.Y}] = true
				}
				rotation := hashUnitFloat(c.X, c.Y, seed+pass*31+83) * 2 * math.Pi
				out = append(out, scenario.UnitRecipe{Op: "add_unit", Player: 0, UnitConst: unit, X: float64Ptr(x), Y: float64Ptr(y), Z: float64Ptr(0), Status: intPtr(2), Rotation: &rotation, Snap: snap})
				if spacing > 0 {
					bins[bucket] = append(bins[bucket], ecologyPoint{x, y})
				}
				d.Emitted++
				if candidate.amphibious {
					d.AmphibiousEmitted++
				}
				d.UnitCounts[unit]++
				if entry, ok := layer.FeltEntries[unit]; ok {
					d.FamilyCounts[entry.Form+":"+entry.PlacementHint]++
				}
			}
		}
		d.TruncatedBySpacing = count - d.Emitted
		diagnostics = append(diagnostics, d)
	}
	return out, diagnostics
}

func emitBuildingClusters(r Report, layer HabitatLayer, candidates []ecologyCandidate, seed int, occupied map[[2]int]bool, d *LayerDiagnostics) ([]scenario.UnitRecipe, int) {
	if layer.FootprintRadius > 64 || r.GridSize <= 0 || len(r.Cells) != r.GridSize*r.GridSize {
		return nil, 0
	}
	cells := make(map[[2]int]Cell, len(r.Cells))
	for _, c := range r.Cells {
		cells[[2]int{c.X, c.Y}] = c
	}
	footprint := func(cx, cy int) ([][2]int, bool) {
		var tiles [][2]int
		for y := cy - layer.FootprintRadius; y <= cy+layer.FootprintRadius; y++ {
			for x := cx - layer.FootprintRadius; x <= cx+layer.FootprintRadius; x++ {
				key := [2]int{x, y}
				c, ok := cells[key]
				if !ok || c.Land == nil || !*c.Land || c.Road || c.Roadside || c.Border || c.Causeway || !cellBuildableForGaiaPlacement(c) || occupied[key] {
					return nil, false
				}
				if len(layer.TerrainIDs) > 0 && !member(layer.TerrainIDs, c.TerrainID) {
					return nil, false
				}
				tiles = append(tiles, key)
			}
		}
		return tiles, true
	}
	var out []scenario.UnitRecipe
	clusterCenters := make([][2]float64, 0, layer.ClusterCount)
	clusters := 0
	for _, center := range candidates {
		if clusters >= layer.ClusterCount || len(out) >= d.Requested {
			break
		}
		cx, cy := center.cell.X, center.cell.Y
		tooClose := false
		for _, point := range clusterCenters {
			if squaredDistance(float64(cx)+0.5, float64(cy)+0.5, point[0], point[1]) < layer.ClusterSpacing*layer.ClusterSpacing {
				tooClose = true
				break
			}
		}
		if tooClose {
			continue
		}
		firstFootprint, ok := footprint(cx, cy)
		if !ok {
			continue
		}
		members := []ecologyCandidate{center}
		memberFootprints := [][][2]int{firstFootprint}
		for _, candidate := range candidates {
			if len(members) >= layer.ClusterSize {
				break
			}
			c := candidate.cell
			if c.X == cx && c.Y == cy || squaredDistance(float64(cx)+0.5, float64(cy)+0.5, float64(c.X)+0.5, float64(c.Y)+0.5) > layer.ClusterRadius*layer.ClusterRadius {
				continue
			}
			separated := true
			for _, previous := range members {
				if squaredDistance(float64(c.X)+0.5, float64(c.Y)+0.5, float64(previous.cell.X)+0.5, float64(previous.cell.Y)+0.5) < layer.Spacing*layer.Spacing {
					separated = false
					break
				}
			}
			if !separated {
				continue
			}
			tiles, safe := footprint(c.X, c.Y)
			if !safe {
				continue
			}
			members = append(members, candidate)
			memberFootprints = append(memberFootprints, tiles)
		}
		if len(members) < layer.ClusterSize {
			continue
		}
		for i, candidate := range members {
			c := candidate.cell
			available := habitatUnits(c, layer)
			if len(available) == 0 {
				continue
			}
			unit := weightedChoice(available, hashUnitFloat(c.X, c.Y, seed+clusters*97+i))
			x, y := float64(c.X)+0.5, float64(c.Y)+0.5
			rotation := hashUnitFloat(c.X, c.Y, seed+clusters*97+i+31) * 2 * math.Pi
			out = append(out, scenario.UnitRecipe{Op: "add_unit", Player: 0, UnitConst: unit, X: float64Ptr(x), Y: float64Ptr(y), Z: float64Ptr(0), Status: intPtr(2), Rotation: &rotation, Snap: "grid"})
			for _, tile := range memberFootprints[i] {
				occupied[tile] = true
			}
			d.UnitCounts[unit]++
			d.FootprintTiles += len(memberFootprints[i])
			d.Emitted++
		}
		clusterCenters = append(clusterCenters, [2]float64{float64(cx) + 0.5, float64(cy) + 0.5})
		clusters++
	}
	return out, clusters
}
