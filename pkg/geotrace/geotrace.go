package geotrace

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"aoe2kit/pkg/scenario"
)

type BBox struct {
	MinLat float64 `json:"min_lat"`
	MinLon float64 `json:"min_lon"`
	MaxLat float64 `json:"max_lat"`
	MaxLon float64 `json:"max_lon"`
}

type Options struct {
	BBox                BBox
	GridSize            int
	Provider            string
	LandmaskProvider    string
	LandmaskPath        string
	InlandWaterProvider string
	InlandWaterPath     string
	LandcoverProvider   string
	LandcoverCacheDir   string
	LandcoverPath       string
	LandcoverDir        string
	ClimateZone         string
	FeltMappingPath     string
	Orient              string
	TargetWaterFraction float64
	SeamWaterColumns    int
	FloraDensity        float64
	LandFaunaDensity    float64
	FishDensity         float64
	ShoreFishDensity    float64
	OceanFishDensity    float64
	ShoreBandWidth      float64
	WaterBandWidth      float64
	RoadsProvider       string
	RoadsPath           string
	RoadsCacheDir       string
	RoadSimplifyTiles   float64
	AdminBorders        bool
	Client              *http.Client
}

type Report struct {
	BBox                  BBox                 `json:"bbox"`
	OverlayBBox           BBox                 `json:"overlay_bbox,omitempty"`
	GridSize              int                  `json:"grid_size"`
	Provider              string               `json:"provider"`
	LandmaskProvider      string               `json:"landmask_provider,omitempty"`
	InlandWaterProvider   string               `json:"inland_water_provider,omitempty"`
	LandcoverProvider     string               `json:"landcover_provider,omitempty"`
	FeltMappingSchema     string               `json:"felt_mapping_schema,omitempty"`
	ClimateZone           string               `json:"climate_zone,omitempty"`
	Orient                string               `json:"orient"`
	ApproxTileMeters      float64              `json:"approx_tile_meters"`
	CoastlineConfidence   string               `json:"coastline_confidence"`
	CoastlineReason       string               `json:"coastline_reason"`
	StandingLadder        []string             `json:"standing_ladder"`
	Warnings              []string             `json:"warnings,omitempty"`
	Cells                 []Cell               `json:"cells"`
	TerrainCounts         map[int]int          `json:"terrain_counts"`
	LandcoverCounts       map[string]int       `json:"landcover_counts,omitempty"`
	LandcoverGLCCounts    map[string]int       `json:"landcover_glc_counts,omitempty"`
	SpeciesResolutions    []SpeciesResolution  `json:"species_resolutions,omitempty"`
	LandCells             int                  `json:"land_cells"`
	WaterCells            int                  `json:"water_cells"`
	LandFraction          float64              `json:"land_fraction"`
	WaterFraction         float64              `json:"water_fraction"`
	TargetWaterFraction   float64              `json:"target_water_fraction,omitempty"`
	LandEmphasisApplied   bool                 `json:"land_emphasis_applied,omitempty"`
	FloraDensity          float64              `json:"flora_density,omitempty"`
	LandFaunaDensity      float64              `json:"land_fauna_density,omitempty"`
	FishDensity           float64              `json:"fish_density,omitempty"`
	ShoreFishDensity      float64              `json:"shore_fish_density,omitempty"`
	OceanFishDensity      float64              `json:"ocean_fish_density,omitempty"`
	ShoreBandWidth        float64              `json:"shore_band_width,omitempty"`
	WaterBandWidth        float64              `json:"water_band_width,omitempty"`
	ShoreCrossfadeCells   int                  `json:"shore_crossfade_cells,omitempty"`
	ShoreCrossfadeCounts  map[string]int       `json:"shore_crossfade_counts,omitempty"`
	ShoreTerrainCounts    map[int]int          `json:"shore_terrain_counts,omitempty"`
	UnbuildableTerrainIDs []int                `json:"unbuildable_terrain_ids,omitempty"`
	UnbuildableCells      int                  `json:"unbuildable_cells,omitempty"`
	PlacementGuard        *PlacementGuard      `json:"placement_guard,omitempty"`
	RoadsProvider         string               `json:"roads_provider,omitempty"`
	RoadWayCount          int                  `json:"road_way_count,omitempty"`
	RoadVertexCountRaw    int                  `json:"road_vertex_count_raw,omitempty"`
	RoadVertexCount       int                  `json:"road_vertex_count,omitempty"`
	RoadSimplifyTiles     float64              `json:"road_simplify_tiles,omitempty"`
	RoadCells             int                  `json:"road_cells,omitempty"`
	RoadsideCells         int                  `json:"roadside_cells,omitempty"`
	WaterBufferCells      int                  `json:"water_buffer_cells,omitempty"`
	KeyWestCausewayCells  int                  `json:"key_west_causeway_cells,omitempty"`
	BorderCells           int                  `json:"border_cells,omitempty"`
	BorderWayCount        int                  `json:"border_way_count,omitempty"`
	RoadCellsNorthOf31    int                  `json:"road_cells_north_of_31,omitempty"`
	BorderCellsNorthOf31  int                  `json:"border_cells_north_of_31,omitempty"`
	RoadRefs              []string             `json:"road_refs,omitempty"`
	LandExtentXTiles      int                  `json:"land_extent_x_tiles,omitempty"`
	LandExtentYTiles      int                  `json:"land_extent_y_tiles,omitempty"`
	LandExtentRatio       float64              `json:"land_extent_y_to_x_ratio,omitempty"`
	LandExtentWHRatio     float64              `json:"land_extent_w_to_h_ratio,omitempty"`
	ScreenWHRatio         float64              `json:"estimated_screen_w_to_h_ratio,omitempty"`
	ViewportTileWHRatio   float64              `json:"viewport_tile_w_to_h_ratio,omitempty"`
	ViewportScreenWHRatio float64              `json:"viewport_screen_w_to_h_ratio,omitempty"`
	PeninsulaAxisAngleDeg float64              `json:"peninsula_axis_angle_deg,omitempty"`
	ProjectionSelfCheck   *ProjectionSelfCheck `json:"projection_self_check,omitempty"`
	MinElevation          float64              `json:"min_elevation"`
	MaxElevation          float64              `json:"max_elevation"`
	MeanElevation         float64              `json:"mean_elevation"`
	Recipe                *scenario.Recipe     `json:"recipe,omitempty"`
}

type PlacementGuard struct {
	Policy                 string         `json:"policy"`
	Evidence               string         `json:"evidence"`
	UnbuildableTerrainIDs  []int          `json:"unbuildable_terrain_ids"`
	UnbuildableCells       map[int]int    `json:"unbuildable_cells,omitempty"`
	CandidateCellsFiltered int            `json:"candidate_cells_filtered,omitempty"`
	AllowedTerrainPrunes   map[string]int `json:"allowed_terrain_prunes,omitempty"`
}

type ProjectionSelfCheck struct {
	DueEastScreenDX  float64 `json:"due_east_screen_dx"`
	DueEastScreenDY  float64 `json:"due_east_screen_dy"`
	DueNorthScreenDX float64 `json:"due_north_screen_dx"`
	DueNorthScreenDY float64 `json:"due_north_screen_dy"`
}

type Cell struct {
	X                    int             `json:"x"`
	Y                    int             `json:"y"`
	Lat                  float64         `json:"lat"`
	Lon                  float64         `json:"lon"`
	OutsideTrace         bool            `json:"outside_trace,omitempty"`
	Elevation            float64         `json:"elevation_m"`
	Land                 *bool           `json:"land,omitempty"`
	InlandWater          bool            `json:"inland_water,omitempty"`
	WaterbodyName        string          `json:"waterbody_name,omitempty"`
	LandcoverClass       int             `json:"landcover_class,omitempty"`
	LandcoverKey         string          `json:"landcover_key,omitempty"`
	LandcoverClassGLC    int             `json:"landcover_class_glc,omitempty"`
	LandcoverKeyGLC      string          `json:"landcover_key_glc,omitempty"`
	LandcoverFraction    float64         `json:"landcover_fraction,omitempty"`
	LandcoverHistogram   []LandcoverStat `json:"landcover_histogram,omitempty"`
	SpeciesUnitConsts    []int           `json:"species_unit_consts,omitempty"`
	DistanceToShoreTiles float64         `json:"distance_to_shore_tiles,omitempty"`
	ClimateZone          string          `json:"climate_zone,omitempty"`
	ElevationBand        string          `json:"elevation_band,omitempty"`
	Standing             string          `json:"standing"`
	TerrainID            int             `json:"terrain_id"`
	Road                 bool            `json:"road,omitempty"`
	Roadside             bool            `json:"roadside,omitempty"`
	WaterBuffer          bool            `json:"water_buffer,omitempty"`
	KeyWestCauseway      bool            `json:"key_west_causeway,omitempty"`
	Border               bool            `json:"border,omitempty"`
	ShoreCrossfadeBand   string          `json:"shore_crossfade_band,omitempty"`
	ShoreTerrainID       int             `json:"shore_terrain_id,omitempty"`
	TerrainConfidence    string          `json:"terrain_confidence,omitempty"`
	TerrainProportions   []TerrainWeight `json:"terrain_proportions,omitempty"`
	Reason               string          `json:"reason"`
}

type TerrainWeight struct {
	TerrainID int     `json:"t"`
	Weight    float64 `json:"w"`
}

type LandcoverStat struct {
	Class  int     `json:"class"`
	Key    string  `json:"key"`
	Weight float64 `json:"weight"`
}

type LandcoverSample struct {
	Valid     bool
	Class     int
	Key       string
	GLCClass  int
	GLCKey    string
	Fraction  float64
	Histogram []LandcoverStat
}

//go:embed default_felt_mapping.json
var defaultFeltMappingJSON []byte

type FeltMapping struct {
	Schema                      string                                        `json:"schema"`
	ConfidenceLadder            []string                                      `json:"confidence_ladder"`
	WaterByDistanceToShore      WaterDistanceMapping                          `json:"water_by_distance_to_shore"`
	LandByClimateAndBand        map[string]map[string]FeltTerrainChoice       `json:"-"`
	RawLandByClimateAndBand     map[string]json.RawMessage                    `json:"land_by_climate_and_band"`
	LandcoverClassMapping       *LandcoverClassMapping                        `json:"landcover_class_mapping,omitempty"`
	SpeciesForClusterScatter    map[string]map[string][]SpeciesRecommendation `json:"-"`
	RawSpeciesForClusterScatter map[string]json.RawMessage                    `json:"species_for_cluster_scatter"`
	LandcoverSpeciesByClass     map[string]map[string][]SpeciesRecommendation `json:"-"`
	RawLandcoverSpeciesByClass  map[string]json.RawMessage                    `json:"landcover_species_by_class"`
	SpeciesResolutions          []SpeciesResolution                           `json:"-"`
}

type WaterDistanceMapping struct {
	Buckets []WaterDistanceBucket `json:"buckets"`
}

type WaterDistanceBucket struct {
	MaxDistanceTiles float64                    `json:"max_dist_tiles"`
	Confidence       string                     `json:"confidence"`
	ByClimate        map[string][]TerrainWeight `json:"by_climate"`
	Default          []TerrainWeight            `json:"default"`
}

type FeltTerrainChoice struct {
	Proportions      []TerrainWeight                    `json:"proportions"`
	Confidence       string                             `json:"confidence"`
	Reason           string                             `json:"reason"`
	Note             string                             `json:"note,omitempty"`
	SpeciesByClimate map[string][]SpeciesRecommendation `json:"species_by_climate,omitempty"`
	WetlandCandidate *FeltTerrainChoice                 `json:"wetland_candidate,omitempty"`
}

type LandcoverClassMapping struct {
	ByClass map[string]FeltTerrainChoice `json:"by_class"`
}

type SpeciesRecommendation struct {
	UnitConst int    `json:"c"`
	Name      string `json:"s,omitempty"`
	Note      string `json:"note,omitempty"`
}

type SpeciesResolution struct {
	Name       string `json:"name"`
	UnitConst  int    `json:"unit_const"`
	Resolution string `json:"resolution"`
}

func TraceDEM(opts Options) (Report, error) {
	if opts.GridSize <= 0 {
		return Report{}, fmt.Errorf("grid size must be > 0")
	}
	if opts.GridSize > 480 {
		return Report{}, fmt.Errorf("grid size %d is too large for the current geo-trace cap of 480", opts.GridSize)
	}
	if opts.BBox.MinLat >= opts.BBox.MaxLat || opts.BBox.MinLon >= opts.BBox.MaxLon {
		return Report{}, fmt.Errorf("bbox must be minLat,minLon,maxLat,maxLon")
	}
	mapping, err := loadFeltMapping(opts.FeltMappingPath)
	if err != nil {
		return Report{}, err
	}
	provider := opts.Provider
	if provider == "" {
		provider = "opentopodata"
	}
	orient, err := normalizeOrient(opts.Orient)
	if err != nil {
		return Report{}, err
	}
	points := gridPoints(opts.BBox, opts.GridSize, orient)
	overlayBBox := bboxFromCells(points)
	elevations, err := sampleElevations(provider, opts.Client, points)
	if err != nil {
		return Report{}, err
	}
	landmask, landmaskProvider, err := sampleLandmask(opts, points)
	if err != nil {
		return Report{}, err
	}
	inlandWater, inlandWaterNames, inlandWaterProvider, err := sampleInlandWater(opts, points)
	if err != nil {
		return Report{}, err
	}
	landcover, landcoverProvider, err := sampleLandcover(opts, points, landmask)
	if err != nil {
		return Report{}, err
	}
	report := Report{
		BBox:                opts.BBox,
		OverlayBBox:         overlayBBox,
		GridSize:            opts.GridSize,
		Provider:            provider,
		LandmaskProvider:    landmaskProvider,
		InlandWaterProvider: inlandWaterProvider,
		LandcoverProvider:   landcoverProvider,
		FeltMappingSchema:   mapping.Schema,
		ClimateZone:         climateZoneForLat((opts.BBox.MinLat+opts.BBox.MaxLat)/2, opts.ClimateZone),
		Orient:              orient,
		ApproxTileMeters:    approximateTileMeters(opts.BBox, opts.GridSize),
		CoastlineConfidence: "approximate",
		CoastlineReason:     "DEM-only sampling cannot distinguish ocean 0.0m from low coastal land; provide landmask/bathymetry/coastline vector to upgrade this claim.",
		StandingLadder:      []string{"from_data", "inferred", "default"},
		TerrainCounts:       map[int]int{},
		SpeciesResolutions:  mapping.SpeciesResolutions,
		FloraDensity:        normalizeDensity(opts.FloraDensity),
		LandFaunaDensity:    normalizeDensity(opts.LandFaunaDensity),
		FishDensity:         normalizeDensity(opts.FishDensity),
		ShoreFishDensity:    normalizeDensity(firstPositive(opts.ShoreFishDensity, opts.FishDensity)),
		OceanFishDensity:    normalizeDensity(firstPositive(opts.OceanFishDensity, opts.FishDensity)),
		ShoreBandWidth:      normalizeShoreBandWidth(opts.ShoreBandWidth),
		WaterBandWidth:      normalizeWaterBandWidth(opts.WaterBandWidth),
	}
	if orient == "iso-ccw-screen" {
		proj := newGeoProjection(opts.BBox, orient)
		report.ViewportTileWHRatio = viewportTileWHRatio(proj)
		report.ViewportScreenWHRatio = 1.0
		report.ProjectionSelfCheck = projectionSelfCheck(proj)
		report.PeninsulaAxisAngleDeg = peninsulaAxisAngleDeg(proj)
	}
	if landcover != nil {
		report.LandcoverCounts = map[string]int{}
		report.LandcoverGLCCounts = map[string]int{}
	}
	if report.ApproxTileMeters > 1000 {
		report.Warnings = append(report.Warnings, fmt.Sprintf("coarse trace: each AoE2 tile is about %.0fm; features smaller than 1-2 tiles will blur", report.ApproxTileMeters))
	}
	if provider == "opentopodata" && opts.GridSize > 50 {
		report.Warnings = append(report.Warnings, "OpenTopoData point sampling is a convenience adapter, not a bulk raster source; prefer a local raster for large grids")
	}
	if landmask != nil {
		report.CoastlineConfidence = "from_data"
		report.CoastlineReason = "land/water classification came from an explicit landmask provider, not DEM thresholding"
		if opts.TargetWaterFraction > 0 && opts.TargetWaterFraction < 1 {
			adjusted, applied := emphasizeLandmask(landmask, opts.GridSize, opts.TargetWaterFraction, opts.SeamWaterColumns, orient)
			landmask = adjusted
			report.TargetWaterFraction = opts.TargetWaterFraction
			report.LandEmphasisApplied = applied
			report.Warnings = append(report.Warnings, fmt.Sprintf("land emphasis dial applied: target water fraction %.2f; coastline is creatively expanded for playability, not literal geography", opts.TargetWaterFraction))
		}
	}
	shoreDistance := distanceToOppositeMask(landmask, opts.GridSize)
	minElev := math.MaxFloat64
	maxElev := -math.MaxFloat64
	sum := 0.0
	for i, point := range points {
		elev := elevations[i]
		var land *bool
		if landmask != nil {
			v := landmask[i]
			land = &v
		}
		climateZone := climateZoneForLat(point.Lat, opts.ClimateZone)
		inWater := inlandWater != nil && inlandWater[i]
		waterName := ""
		if inWater && i < len(inlandWaterNames) {
			waterName = inlandWaterNames[i]
		}
		var lc *LandcoverSample
		if landcover != nil {
			lc = &landcover[i]
		}
		cell := classifyCell(point, elev, land, shoreDistance[i], climateZone, mapping, inWater, waterName, lc)
		report.Cells = append(report.Cells, cell)
		report.TerrainCounts[cell.TerrainID]++
		if cell.Land != nil && *cell.Land {
			report.LandCells++
		} else {
			report.WaterCells++
		}
		if cell.LandcoverKey != "" {
			report.LandcoverCounts[cell.LandcoverKey]++
		}
		if cell.LandcoverKeyGLC != "" {
			report.LandcoverGLCCounts[cell.LandcoverKeyGLC]++
		}
		if terrainUnbuildableForGaiaPlacement(cell.TerrainID) {
			report.UnbuildableCells++
		}
		if elev < minElev {
			minElev = elev
		}
		if elev > maxElev {
			maxElev = elev
		}
		sum += elev
	}
	report.UnbuildableTerrainIDs = unbuildableTerrainIDsForGaiaPlacement()
	report.PlacementGuard = buildPlacementGuard(report)
	if report.PlacementGuard != nil {
		report.PlacementGuard.CandidateCellsFiltered = report.UnbuildableCells
	}
	report.MinElevation = minElev
	report.MaxElevation = maxElev
	report.MeanElevation = sum / float64(len(points))
	if len(points) > 0 {
		report.LandFraction = float64(report.LandCells) / float64(len(points))
		report.WaterFraction = float64(report.WaterCells) / float64(len(points))
	}
	if opts.RoadsProvider != "" || opts.RoadsPath != "" {
		overlayOpts := opts
		overlayOpts.BBox = overlayBBox
		roadStats, err := applyRoadLayer(&report, overlayOpts)
		if err != nil {
			return Report{}, err
		}
		report.RoadsProvider = roadStats.Provider
		report.RoadWayCount = roadStats.WayCount
		report.RoadVertexCountRaw = roadStats.VertexCountRaw
		report.RoadVertexCount = roadStats.VertexCount
		report.RoadSimplifyTiles = roadStats.SimplifyTiles
		report.RoadCells = roadStats.CellCount
		report.RoadRefs = roadStats.Refs
	}
	if opts.AdminBorders {
		overlayOpts := opts
		overlayOpts.BBox = overlayBBox
		borderWayCount, err := applyAdministrativeBorderLayer(&report, overlayOpts)
		if err != nil {
			return Report{}, err
		}
		report.BorderWayCount = borderWayCount
	}
	applyDerivedFeatureMasks(&report)
	markShoreCrossfadeCells(&report)
	measureOverlayNorthOf31(&report)
	measureLandExtents(&report)
	if landmask != nil {
		recipe := BuildRecipe(report)
		report.Recipe = &recipe
	}
	return report, nil
}

func measureLandExtents(report *Report) {
	if report == nil {
		return
	}
	minX, minY := math.MaxInt, math.MaxInt
	maxX, maxY := -1, -1
	for _, cell := range report.Cells {
		if cell.Land == nil || !*cell.Land {
			continue
		}
		if cell.X < minX {
			minX = cell.X
		}
		if cell.X > maxX {
			maxX = cell.X
		}
		if cell.Y < minY {
			minY = cell.Y
		}
		if cell.Y > maxY {
			maxY = cell.Y
		}
	}
	if maxX < minX || maxY < minY {
		return
	}
	report.LandExtentXTiles = maxX - minX + 1
	report.LandExtentYTiles = maxY - minY + 1
	if report.LandExtentXTiles > 0 {
		report.LandExtentRatio = float64(report.LandExtentYTiles) / float64(report.LandExtentXTiles)
	}
	if report.LandExtentYTiles > 0 {
		report.LandExtentWHRatio = float64(report.LandExtentXTiles) / float64(report.LandExtentYTiles)
		report.ScreenWHRatio = report.LandExtentWHRatio * screenRhombusWidthToHeight
	}
}

func emphasizeLandmask(land []bool, n int, targetWaterFraction float64, seamWaterCells int, orient string) ([]bool, bool) {
	out := append([]bool(nil), land...)
	if len(out) != n*n || n <= 0 {
		return out, false
	}
	forceSeamWater(out, n, seamWaterCells, orient)
	targetLand := int(math.Round(float64(n*n) * (1 - targetWaterFraction)))
	if targetLand < 0 {
		targetLand = 0
	}
	if targetLand > n*n {
		targetLand = n * n
	}
	landCount := 0
	for _, v := range out {
		if v {
			landCount++
		}
	}
	applied := landCount != countBool(land)
	for landCount < targetLand {
		candidates := landDilationCandidates(out, n, seamWaterCells, orient)
		if len(candidates) == 0 {
			break
		}
		sort.Ints(candidates)
		needed := targetLand - landCount
		if needed > len(candidates) {
			needed = len(candidates)
		}
		for _, idx := range candidates[:needed] {
			if !out[idx] {
				out[idx] = true
				landCount++
				applied = true
			}
		}
	}
	forceSeamWater(out, n, seamWaterCells, orient)
	return out, applied
}

func landDilationCandidates(land []bool, n int, seamWaterCells int, orient string) []int {
	seen := make([]bool, len(land))
	var out []int
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			idx := y*n + x
			if land[idx] {
				continue
			}
			if inSeamWaterCell(x, y, n, seamWaterCells, orient) {
				continue
			}
			for dy := -1; dy <= 1; dy++ {
				for dx := -1; dx <= 1; dx++ {
					if dx == 0 && dy == 0 {
						continue
					}
					xx := x + dx
					yy := y + dy
					if xx < 0 || yy < 0 || xx >= n || yy >= n {
						continue
					}
					if land[yy*n+xx] && !seen[idx] {
						seen[idx] = true
						out = append(out, idx)
					}
				}
			}
		}
	}
	return out
}

func forceSeamWater(land []bool, n int, seamWaterCells int, orient string) {
	if seamWaterCells <= 0 {
		return
	}
	if orient == "diamond" {
		for sum := 0; sum <= 2*n-2; sum++ {
			xMin, xMax, length := diagonalRange(n, sum)
			if length == 0 {
				continue
			}
			limit := seamWaterCells
			if limit*2 > length {
				limit = (length + 1) / 2
			}
			for i := 0; i < limit; i++ {
				x := xMin + i
				land[(sum-x)*n+x] = false
				x = xMax - i
				land[(sum-x)*n+x] = false
			}
		}
		return
	}
	if seamWaterCells*2 >= n {
		seamWaterCells = n / 4
	}
	for y := 0; y < n; y++ {
		for x := 0; x < seamWaterCells; x++ {
			land[y*n+x] = false
			land[y*n+(n-1-x)] = false
		}
	}
}

func inSeamWaterCell(x, y, n, seamWaterCells int, orient string) bool {
	if seamWaterCells <= 0 {
		return false
	}
	if orient == "diamond" {
		xMin, xMax, length := diagonalRange(n, x+y)
		if length == 0 {
			return false
		}
		limit := seamWaterCells
		if limit*2 > length {
			limit = (length + 1) / 2
		}
		return x < xMin+limit || x > xMax-limit
	}
	return x < seamWaterCells || x >= n-seamWaterCells
}

func countBool(values []bool) int {
	count := 0
	for _, value := range values {
		if value {
			count++
		}
	}
	return count
}

func normalizeOrient(raw string) (string, error) {
	raw = strings.ToLower(strings.TrimSpace(raw))
	if raw == "" {
		return "iso-ccw", nil
	}
	switch raw {
	case "grid", "iso-cw", "iso-ccw", "iso-ccw-screen", "diamond":
		return raw, nil
	case "globe", "globe-diamond", "rhombic":
		return "diamond", nil
	case "iso":
		return "iso-ccw", nil
	default:
		return "", fmt.Errorf("unsupported geo orientation %q", raw)
	}
}

func gridPoints(b BBox, n int, orient string) []Cell {
	out := make([]Cell, 0, n*n)
	proj := newGeoProjection(b, orient)
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			if orient == "diamond" {
				latSpan := b.MaxLat - b.MinLat
				lonSpan := b.MaxLon - b.MinLon
				sum := x + y
				xMin, xMax, length := diagonalRange(n, sum)
				latT := float64(sum) / float64(2*(n-1))
				lonT := 0.5
				if length > 1 {
					lonT = float64(x-xMin) / float64(xMax-xMin)
				}
				lon := b.MinLon + lonT*lonSpan
				lat := b.MaxLat - latT*latSpan
				out = append(out, Cell{X: x, Y: y, Lat: lat, Lon: lon})
				continue
			}
			u := (float64(x)+0.5)/float64(n) - 0.5
			v := (float64(y)+0.5)/float64(n) - 0.5
			lat, lon, outside := proj.gridToLatLon(u, v)
			out = append(out, Cell{X: x, Y: y, Lat: lat, Lon: lon, OutsideTrace: outside})
		}
	}
	return out
}

func bboxFromCells(cells []Cell) BBox {
	if len(cells) == 0 {
		return BBox{}
	}
	out := BBox{
		MinLat: math.MaxFloat64,
		MinLon: math.MaxFloat64,
		MaxLat: -math.MaxFloat64,
		MaxLon: -math.MaxFloat64,
	}
	for _, cell := range cells {
		if cell.Lat < out.MinLat {
			out.MinLat = cell.Lat
		}
		if cell.Lat > out.MaxLat {
			out.MaxLat = cell.Lat
		}
		if cell.Lon < out.MinLon {
			out.MinLon = cell.Lon
		}
		if cell.Lon > out.MaxLon {
			out.MaxLon = cell.Lon
		}
	}
	return out
}

func markShoreCrossfadeCells(report *Report) {
	if report == nil || report.GridSize <= 0 {
		return
	}
	width := normalizeShoreBandWidth(report.ShoreBandWidth)
	waterWidth := normalizeWaterBandWidth(report.WaterBandWidth)
	report.ShoreBandWidth = width
	report.WaterBandWidth = waterWidth
	report.ShoreCrossfadeCells = 0
	report.ShoreCrossfadeCounts = map[string]int{}
	report.ShoreTerrainCounts = map[int]int{}
	for i := range report.Cells {
		cell := &report.Cells[i]
		if cell.InlandWater || math.IsNaN(cell.DistanceToShoreTiles) {
			continue
		}
		isLand := cell.Land != nil && *cell.Land
		isWater := cell.Land != nil && !*cell.Land
		switch {
		case isWater && cell.DistanceToShoreTiles <= waterWidth:
			cell.ShoreCrossfadeBand = "B_over_A"
		case isWater && cell.DistanceToShoreTiles <= 2*waterWidth:
			cell.ShoreCrossfadeBand = "A"
		case isWater && cell.DistanceToShoreTiles <= 3*waterWidth:
			cell.ShoreCrossfadeBand = "A_over_M"
		case isWater && cell.DistanceToShoreTiles <= 4*waterWidth:
			cell.ShoreCrossfadeBand = "M"
		case isWater && cell.DistanceToShoreTiles <= 5*waterWidth:
			cell.ShoreCrossfadeBand = "M_over_D"
		case isWater && cell.DistanceToShoreTiles <= 6*waterWidth:
			cell.ShoreCrossfadeBand = "D"
		case isLand && cell.DistanceToShoreTiles <= width:
			cell.ShoreCrossfadeBand = "B"
		case isLand && cell.DistanceToShoreTiles <= 2*width:
			cell.ShoreCrossfadeBand = "B_over_C"
		case isLand && cell.DistanceToShoreTiles <= 3*width:
			cell.ShoreCrossfadeBand = "C_over_B"
		}
		if cell.ShoreCrossfadeBand != "" {
			if strings.Contains(cell.ShoreCrossfadeBand, "B") {
				cell.ShoreTerrainID = shoreTerrainIDForClimate(shoreClimateForCell(report.ClimateZone, cell.ClimateZone), cell.X, cell.Y)
				report.ShoreTerrainCounts[cell.ShoreTerrainID]++
			}
			report.ShoreCrossfadeCells++
			report.ShoreCrossfadeCounts[cell.ShoreCrossfadeBand]++
		}
	}
	if report.ShoreCrossfadeCells == 0 {
		report.ShoreCrossfadeCounts = nil
		report.ShoreTerrainCounts = nil
	}
	if len(report.ShoreTerrainCounts) == 0 {
		report.ShoreTerrainCounts = nil
	}
}

func shoreClimateForCell(regionClimate, cellClimate string) string {
	switch regionClimate {
	case "boreal", "cold", "humid_continental":
		return regionClimate
	default:
		return cellClimate
	}
}

func shoreTerrainIDForClimate(climate string, x, y int) int {
	switch climate {
	case "boreal", "cold":
		switch (x + y) % 4 {
		case 0:
			return 37
		case 1:
			return 32
		default:
			return 82
		}
	case "humid_continental":
		switch (x + y) % 4 {
		case 0:
			return 2
		case 1:
			return 37
		default:
			return 52
		}
	case "temperate":
		if (x+y)%3 == 0 {
			return 52
		}
		return 2
	case "tropical", "subtropical":
		if (x+y)%3 == 0 {
			return 53
		}
		return 2
	default:
		return 2
	}
}

func unbuildableTerrainIDsForGaiaPlacement() []int {
	return []int{35, 40, 79, 80, 81, 82}
}

func terrainUnbuildableForGaiaPlacement(terrainID int) bool {
	switch terrainID {
	case 35, 40, 79, 80, 81, 82:
		return true
	default:
		return false
	}
}

func cellBuildableForGaiaPlacement(cell Cell) bool {
	return !terrainUnbuildableForGaiaPlacement(cell.TerrainID)
}

func buildPlacementGuard(report Report) *PlacementGuard {
	counts := map[int]int{}
	for _, cell := range report.Cells {
		if terrainUnbuildableForGaiaPlacement(cell.TerrainID) {
			counts[cell.TerrainID]++
		}
	}
	if len(counts) == 0 {
		counts = nil
	}
	return &PlacementGuard{
		Policy:                "Gaia decorative unit recipes skip known no-place land terrain; recipe allowed-terrain lists are pruned before scatter.",
		Evidence:              "terrain-restriction passability plus engine runs: sand beach 2 accepts Gaia, while cracked ice 35, rock 40, and non-navigable beach variants 79-82 reject normal placed Gaia objects.",
		UnbuildableTerrainIDs: unbuildableTerrainIDsForGaiaPlacement(),
		UnbuildableCells:      counts,
		AllowedTerrainPrunes:  map[string]int{},
	}
}

func measureOverlayNorthOf31(report *Report) {
	if report == nil {
		return
	}
	report.RoadCellsNorthOf31 = 0
	report.BorderCellsNorthOf31 = 0
	for _, cell := range report.Cells {
		if cell.Lat <= 31 {
			continue
		}
		if cell.Road {
			report.RoadCellsNorthOf31++
		}
		if cell.Border {
			report.BorderCellsNorthOf31++
		}
	}
}

type geoProjection struct {
	b           BBox
	orient      string
	midLat      float64
	midLon      float64
	kmPerLat    float64
	kmPerLon    float64
	rotHalfKm   float64
	northScale  float64
	tileScaleU  float64
	matrixScale float64
	halfEastKm  float64
	halfNorthKm float64
}

const (
	screenRhombusWidthToHeight = 2.0
)

func newGeoProjection(b BBox, orient string) geoProjection {
	midLat := (b.MinLat + b.MaxLat) / 2
	kmPerLat := 111.320
	kmPerLon := 111.320 * math.Cos(midLat*math.Pi/180)
	if kmPerLon <= 0 {
		kmPerLon = 1
	}
	halfEast := (b.MaxLon - b.MinLon) * kmPerLon / 2
	halfNorth := (b.MaxLat - b.MinLat) * kmPerLat / 2
	northScale := 1.0
	tileScaleU := 1.0
	scaledHalfNorth := halfNorth * northScale
	rotHalf := (halfEast + scaledHalfNorth) / math.Sqrt2
	if rotHalf <= 0 {
		rotHalf = 1
	}
	matrixScale := 0.0
	if orient == "iso-ccw-screen" {
		denom := halfNorth + 0.5*halfEast
		if denom <= 0 {
			denom = 1
		}
		matrixScale = 0.5 / denom
	}
	return geoProjection{
		b:           b,
		orient:      orient,
		midLat:      midLat,
		midLon:      (b.MinLon + b.MaxLon) / 2,
		kmPerLat:    kmPerLat,
		kmPerLon:    kmPerLon,
		rotHalfKm:   rotHalf,
		northScale:  northScale,
		tileScaleU:  tileScaleU,
		matrixScale: matrixScale,
		halfEastKm:  halfEast,
		halfNorthKm: halfNorth,
	}
}

func (p geoProjection) gridToLatLon(u, v float64) (lat, lon float64, outside bool) {
	if p.orient == "grid" {
		lon = p.midLon + u*(p.b.MaxLon-p.b.MinLon)
		lat = p.midLat - v*(p.b.MaxLat-p.b.MinLat)
		return lat, lon, false
	}
	if p.orient == "iso-ccw-screen" {
		scale := p.matrixScale
		if scale == 0 {
			scale = 1
		}
		// Real minimap calibration: AoE2's displayed minimap frame reads the
		// emitted tile axes as sx=u+v, sy=(v-u)/2. This is the accepted
		// proper rotation turned 180 degrees after in-engine readback showed
		// correct chirality but upside-down orientation.
		eastKm := (u + v) / scale
		northKm := (u - v) / (2 * scale)
		lon = p.midLon + eastKm/p.kmPerLon
		lat = p.midLat + northKm/p.kmPerLat
		outside = lat < p.b.MinLat || lat > p.b.MaxLat || lon < p.b.MinLon || lon > p.b.MaxLon
		return lat, lon, outside
	}
	if p.tileScaleU != 0 && p.tileScaleU != 1 {
		u /= p.tileScaleU
	}
	r1 := u * 2 * p.rotHalfKm
	r2 := v * 2 * p.rotHalfKm
	var eastKm, northKm float64
	switch p.orient {
	case "iso-cw":
		eastKm = (r1 + r2) / math.Sqrt2
		northKm = -(r1 - r2) / math.Sqrt2
	default: // iso-ccw
		eastKm = (r1 + r2) / math.Sqrt2
		northKm = (r1 - r2) / math.Sqrt2
	}
	if p.northScale != 0 && p.northScale != 1 {
		northKm /= p.northScale
	}
	lon = p.midLon + eastKm/p.kmPerLon
	lat = p.midLat + northKm/p.kmPerLat
	outside = lat < p.b.MinLat || lat > p.b.MaxLat || lon < p.b.MinLon || lon > p.b.MaxLon
	return lat, lon, outside
}

func (p geoProjection) latLonToGridUnit(lat, lon float64) (u, v float64, ok bool) {
	if p.orient == "grid" {
		if lat < p.b.MinLat || lat > p.b.MaxLat || lon < p.b.MinLon || lon > p.b.MaxLon {
			return 0, 0, false
		}
		u = (lon - p.midLon) / (p.b.MaxLon - p.b.MinLon)
		v = -(lat - p.midLat) / (p.b.MaxLat - p.b.MinLat)
		return u, v, math.Abs(u) <= 0.5 && math.Abs(v) <= 0.5
	}
	eastKm := (lon - p.midLon) * p.kmPerLon
	northKm := (lat - p.midLat) * p.kmPerLat
	if p.northScale != 0 && p.northScale != 1 {
		northKm *= p.northScale
	}
	if p.orient == "iso-ccw-screen" {
		scale := p.matrixScale
		if scale == 0 {
			scale = 1
		}
		// Determinant-positive rotation of the raw shear-free axes, turned 180
		// degrees from v11. The older v10 frame flipped only one axis, which
		// preserved angle and size but mirrored the geography.
		u = scale * (0.5*eastKm + northKm)
		v = scale * (0.5*eastKm - northKm)
		return u, v, math.Abs(u) <= 0.5 && math.Abs(v) <= 0.5
	}
	var r1, r2 float64
	switch p.orient {
	case "iso-cw":
		r1 = (eastKm - northKm) / math.Sqrt2
		r2 = (eastKm + northKm) / math.Sqrt2
	default: // iso-ccw
		r1 = (eastKm + northKm) / math.Sqrt2
		r2 = (eastKm - northKm) / math.Sqrt2
	}
	u = r1 / (2 * p.rotHalfKm)
	v = r2 / (2 * p.rotHalfKm)
	if p.tileScaleU != 0 && p.tileScaleU != 1 {
		u *= p.tileScaleU
	}
	return u, v, math.Abs(u) <= 0.5 && math.Abs(v) <= 0.5
}

func screenPointForLatLon(p geoProjection, lat, lon float64) (sx, sy float64, ok bool) {
	u, v, ok := p.latLonToGridUnit(lat, lon)
	if !ok {
		return 0, 0, false
	}
	if p.orient == "iso-ccw-screen" {
		return u + v, (v - u) / 2, true
	}
	return u - v, (u + v) / 2, true
}

func viewportTileWHRatio(p geoProjection) float64 {
	points := []struct {
		lat float64
		lon float64
	}{
		{p.b.MinLat, p.b.MinLon},
		{p.b.MinLat, p.b.MaxLon},
		{p.b.MaxLat, p.b.MinLon},
		{p.b.MaxLat, p.b.MaxLon},
	}
	minSX, minSY := math.MaxFloat64, math.MaxFloat64
	maxSX, maxSY := -math.MaxFloat64, -math.MaxFloat64
	for _, point := range points {
		sx, sy, ok := screenPointForLatLon(p, point.lat, point.lon)
		if !ok {
			continue
		}
		if sx < minSX {
			minSX = sx
		}
		if sx > maxSX {
			maxSX = sx
		}
		if sy < minSY {
			minSY = sy
		}
		if sy > maxSY {
			maxSY = sy
		}
	}
	screenH := maxSY - minSY
	if screenH <= 0 {
		return 0
	}
	return (maxSX - minSX) / screenH / screenRhombusWidthToHeight
}

func projectionSelfCheck(p geoProjection) *ProjectionSelfCheck {
	lat := p.midLat
	lon := p.midLon
	eastKm := 100.0
	northKm := 100.0
	baseSX, baseSY, ok := screenPointForLatLon(p, lat, lon)
	if !ok {
		return nil
	}
	eastSX, eastSY, eastOK := screenPointForLatLon(p, lat, lon+eastKm/p.kmPerLon)
	northSX, northSY, northOK := screenPointForLatLon(p, lat+northKm/p.kmPerLat, lon)
	if !eastOK || !northOK {
		return nil
	}
	return &ProjectionSelfCheck{
		DueEastScreenDX:  eastSX - baseSX,
		DueEastScreenDY:  eastSY - baseSY,
		DueNorthScreenDX: northSX - baseSX,
		DueNorthScreenDY: northSY - baseSY,
	}
}

func peninsulaAxisAngleDeg(p geoProjection) float64 {
	// Acceptance anchor: Jacksonville-ish north end to the Florida tip
	// should lean about 12 degrees east of south.
	jaxSX, jaxSY, ok0 := screenPointForLatLon(p, 30.3, -81.7)
	tipSX, tipSY, ok1 := screenPointForLatLon(p, 25.4, -80.5)
	if !ok0 || !ok1 {
		return 0
	}
	dx := tipSX - jaxSX
	dySouth := tipSY - jaxSY
	if dySouth == 0 {
		return 0
	}
	return math.Atan2(dx, dySouth) * 180 / math.Pi
}

func diagonalRange(n, sum int) (xMin, xMax, length int) {
	if n <= 0 || sum < 0 || sum > 2*n-2 {
		return 0, -1, 0
	}
	xMin = maxInt(0, sum-(n-1))
	xMax = minInt(n-1, sum)
	return xMin, xMax, xMax - xMin + 1
}

func sampleElevations(provider string, client *http.Client, points []Cell) ([]float64, error) {
	switch provider {
	case "opentopodata":
		return sampleOpenTopoData(client, points)
	case "flat":
		out := make([]float64, len(points))
		return out, nil
	default:
		return nil, fmt.Errorf("unsupported DEM provider %q", provider)
	}
}

func sampleOpenTopoData(client *http.Client, points []Cell) ([]float64, error) {
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	out := make([]float64, 0, len(points))
	const batchSize = 80
	for start := 0; start < len(points); start += batchSize {
		end := start + batchSize
		if end > len(points) {
			end = len(points)
		}
		var locations []string
		for _, point := range points[start:end] {
			locations = append(locations, fmt.Sprintf("%.7f,%.7f", point.Lat, point.Lon))
		}
		u := "https://api.opentopodata.org/v1/srtm30m?locations=" + url.QueryEscape(strings.Join(locations, "|"))
		resp, err := client.Get(u)
		if err != nil {
			return nil, err
		}
		var payload struct {
			Status  string `json:"status"`
			Results []struct {
				Elevation *float64 `json:"elevation"`
			} `json:"results"`
			Error string `json:"error"`
		}
		err = json.NewDecoder(resp.Body).Decode(&payload)
		closeErr := resp.Body.Close()
		if err != nil {
			return nil, err
		}
		if closeErr != nil {
			return nil, closeErr
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 || payload.Status != "OK" {
			return nil, fmt.Errorf("OpenTopoData status %d/%s: %s", resp.StatusCode, payload.Status, payload.Error)
		}
		if len(payload.Results) != end-start {
			return nil, fmt.Errorf("OpenTopoData returned %d result(s), want %d", len(payload.Results), end-start)
		}
		for _, result := range payload.Results {
			if result.Elevation == nil {
				out = append(out, 0)
				continue
			}
			out = append(out, *result.Elevation)
		}
	}
	return out, nil
}

func classifyCell(point Cell, elevation float64, land *bool, shoreDistance float64, climateZone string, mapping FeltMapping, inlandWater bool, waterbodyName string, landcover *LandcoverSample) Cell {
	point.Elevation = elevation
	point.Land = land
	point.InlandWater = inlandWater
	point.WaterbodyName = waterbodyName
	point.DistanceToShoreTiles = shoreDistance
	point.ClimateZone = climateZone
	point.ElevationBand = elevationBand(elevation, land, shoreDistance)
	point.Standing = "from_data"
	if landcover != nil && landcover.Valid {
		point.LandcoverClass = landcover.Class
		point.LandcoverKey = landcover.Key
		point.LandcoverClassGLC = landcover.GLCClass
		point.LandcoverKeyGLC = landcover.GLCKey
		point.LandcoverFraction = landcover.Fraction
		point.LandcoverHistogram = landcover.Histogram
		point.SpeciesUnitConsts = mapping.speciesUnitConsts(landcover.GLCKey, climateZone)
		if landcover.Class == 80 || landcover.Key == "80_permanent_water" {
			point.InlandWater = true
			point.ElevationBand = "inland_water"
			water := false
			point.Land = &water
		} else {
			point.ElevationBand = "landcover_" + landcover.Key
		}
		choice := mapping.landcoverChoice(landcover.Key)
		if len(choice.Proportions) > 0 {
			if landcover.Fraction < 0.65 && len(landcover.Histogram) > 1 {
				if blend := mapping.blendedLandcoverChoice(landcover.Histogram[:minInt(2, len(landcover.Histogram))]); len(blend.Proportions) > 0 {
					choice = blend
					choice.Reason = fmt.Sprintf("WorldCover weak-majority mosaic: %s %.0f%% + %s %.0f%%", landcover.Histogram[0].Key, landcover.Histogram[0].Weight*100, landcover.Histogram[1].Key, landcover.Histogram[1].Weight*100)
				}
			}
			point.TerrainID = chooseTerrain(point.X, point.Y, choice.Proportions)
			point.TerrainConfidence = nonEmpty(choice.Confidence, "from_data")
			point.TerrainProportions = choice.Proportions
			point.Reason = nonEmpty(choice.Reason, nonEmpty(choice.Note, fmt.Sprintf("WorldCover class %d %s", landcover.Class, landcover.Key)))
			return point
		}
	}
	if inlandWater {
		point.ElevationBand = "inland_water"
		choice := mapping.landcoverChoice("80_permanent_water")
		if len(choice.Proportions) == 0 {
			choice = FeltTerrainChoice{
				Proportions: []TerrainWeight{{TerrainID: 1, Weight: 0.5}, {TerrainID: 23, Weight: 0.4}, {TerrainID: 22, Weight: 0.1}},
				Confidence:  "from_data",
				Reason:      "inland water polygon",
			}
		}
		point.TerrainID = chooseTerrain(point.X, point.Y, choice.Proportions)
		point.TerrainConfidence = nonEmpty(choice.Confidence, "from_data")
		point.TerrainProportions = choice.Proportions
		point.Reason = nonEmpty(choice.Reason, "inland water polygon")
		if waterbodyName != "" {
			point.Reason = fmt.Sprintf("%s: %s", waterbodyName, point.Reason)
		}
		return point
	}
	if land != nil {
		choice := mappingChoice(mapping, point.ElevationBand, climateZone, shoreDistance)
		if len(choice.Proportions) > 0 {
			point.TerrainID = chooseTerrain(point.X, point.Y, choice.Proportions)
			point.TerrainConfidence = choice.Confidence
			point.TerrainProportions = choice.Proportions
			point.Reason = choice.Reason
			if point.Reason == "" {
				point.Reason = fmt.Sprintf("felt mapping %s/%s", climateZone, point.ElevationBand)
			}
			return point
		}
	}
	switch {
	case elevation <= 0:
		point.TerrainID = 58
		point.ElevationBand = "water"
		point.TerrainConfidence = "default"
		point.Reason = "DEM elevation <= 0m; classified as approximate shallow/shore water without landmask"
	case elevation < 20:
		point.TerrainID = 0
		point.ElevationBand = "lowland"
		point.TerrainConfidence = "default"
		point.Reason = "lowland DEM elevation"
	case elevation < 120:
		point.TerrainID = 100
		point.ElevationBand = "upland"
		point.TerrainConfidence = "default"
		point.Reason = "upland DEM elevation"
	default:
		point.TerrainID = 70
		point.ElevationBand = "highland"
		point.TerrainConfidence = "default"
		point.Reason = "highland DEM elevation"
	}
	return point
}

func loadFeltMapping(path string) (FeltMapping, error) {
	data := defaultFeltMappingJSON
	if path != "" {
		var err error
		data, err = os.ReadFile(path)
		if err != nil {
			return FeltMapping{}, err
		}
	}
	var mapping FeltMapping
	if err := json.Unmarshal(data, &mapping); err != nil {
		return FeltMapping{}, err
	}
	if mapping.Schema == "" {
		return FeltMapping{}, fmt.Errorf("felt mapping missing schema")
	}
	mapping.LandcoverSpeciesByClass = parseSpeciesByClass(mapping.RawLandcoverSpeciesByClass)
	resolveSpeciesRecommendations(mapping.LandcoverSpeciesByClass, &mapping.SpeciesResolutions)
	mapping.LandByClimateAndBand = map[string]map[string]FeltTerrainChoice{}
	for climate, rawClimate := range mapping.RawLandByClimateAndBand {
		if strings.HasPrefix(climate, "_") {
			continue
		}
		var rawBands map[string]json.RawMessage
		if err := json.Unmarshal(rawClimate, &rawBands); err != nil {
			continue
		}
		bands := map[string]FeltTerrainChoice{}
		for band, rawChoice := range rawBands {
			if strings.HasPrefix(band, "_") {
				continue
			}
			var choice FeltTerrainChoice
			if err := json.Unmarshal(rawChoice, &choice); err != nil {
				continue
			}
			if len(choice.Proportions) > 0 {
				bands[band] = choice
			}
		}
		if len(bands) > 0 {
			mapping.LandByClimateAndBand[climate] = bands
		}
	}
	return mapping, nil
}

func parseSpeciesByClass(raw map[string]json.RawMessage) map[string]map[string][]SpeciesRecommendation {
	out := map[string]map[string][]SpeciesRecommendation{}
	for classKey, rawClass := range raw {
		if strings.HasPrefix(classKey, "_") {
			continue
		}
		var byClimate map[string][]SpeciesRecommendation
		if err := json.Unmarshal(rawClass, &byClimate); err != nil {
			continue
		}
		for climate, species := range byClimate {
			filtered := species[:0]
			for _, item := range species {
				if item.UnitConst != 0 || item.Name != "" || item.Note != "" {
					filtered = append(filtered, item)
				}
			}
			if len(filtered) == 0 {
				delete(byClimate, climate)
				continue
			}
			byClimate[climate] = filtered
		}
		if len(byClimate) > 0 {
			out[classKey] = byClimate
		}
	}
	return out
}

func resolveSpeciesRecommendations(byClass map[string]map[string][]SpeciesRecommendation, resolutions *[]SpeciesResolution) {
	seen := map[string]bool{}
	for classKey, byClimate := range byClass {
		for climate, species := range byClimate {
			for i := range species {
				if species[i].UnitConst != 0 {
					continue
				}
				unitConst, resolution := resolveSpeciesName(species[i].Name)
				if unitConst == 0 {
					continue
				}
				species[i].UnitConst = unitConst
				if species[i].Note == "" {
					species[i].Note = resolution
				}
				key := fmt.Sprintf("%s|%s|%s|%d|%s", classKey, climate, species[i].Name, unitConst, resolution)
				if !seen[key] {
					seen[key] = true
					*resolutions = append(*resolutions, SpeciesResolution{Name: species[i].Name, UnitConst: unitConst, Resolution: resolution})
				}
			}
			byClimate[climate] = species
		}
	}
}

func resolveSpeciesName(name string) (int, string) {
	n := strings.ToLower(name)
	switch {
	case strings.Contains(n, "cabbage") || strings.Contains(n, "sabal"):
		return 351, "resolved to Palm Tree (351) from felt palette"
	case strings.Contains(n, "sugar maple"):
		return 2027, "resolved to Asian Maple Green (2027) as closest maple-like standing tree"
	case strings.Contains(n, "beech"):
		return 349, "fallback to Oak Tree (349); no distinct beech unit found in current scrape"
	case strings.Contains(n, "black spruce"):
		return 413, "fallback to Snow Pine Tree (413) for dark/cold conifer"
	case strings.Contains(n, "spruce"):
		return 413, "fallback to Snow Pine Tree (413) for spruce-like conifer"
	case strings.Contains(n, "tamarack") || strings.Contains(n, "larch"):
		return 413, "fallback to Snow Pine Tree (413) for tamarack/larch"
	case strings.Contains(n, "bald cypress"):
		return 1984, "fallback to Lush Bamboo (1984) cypress-strand feel per felt table"
	case strings.Contains(n, "prairie grass"):
		return 1358, "fallback to Grass Patch (1358)"
	default:
		return 0, ""
	}
}

func elevationBand(elevation float64, land *bool, shoreDistance float64) string {
	if land != nil {
		if !*land {
			return "water"
		}
		if shoreDistance <= 1.1 {
			return "beach"
		}
	}
	switch {
	case elevation <= 0 && land == nil:
		return "water"
	case elevation < 20:
		return "lowland"
	case elevation < 120:
		return "upland"
	default:
		return "highland"
	}
}

func climateZoneForLat(lat float64, override string) string {
	if override != "" {
		return strings.ToLower(strings.TrimSpace(override))
	}
	absLat := math.Abs(lat)
	switch {
	case absLat < 23.5:
		return "tropical"
	case absLat < 35:
		return "subtropical"
	case absLat < 40:
		return "temperate"
	case absLat < 55:
		return "humid_continental"
	default:
		return "boreal"
	}
}

func mappingChoice(mapping FeltMapping, band, climate string, shoreDistance float64) FeltTerrainChoice {
	if band == "water" {
		for _, bucket := range mapping.WaterByDistanceToShore.Buckets {
			if shoreDistance > bucket.MaxDistanceTiles {
				continue
			}
			proportions := bucket.ByClimate[climate]
			for _, fallback := range climateFallbacks(climate, band) {
				if len(proportions) != 0 {
					break
				}
				proportions = bucket.ByClimate[fallback]
			}
			if len(proportions) == 0 {
				proportions = bucket.Default
			}
			if len(proportions) == 0 {
				for _, candidate := range bucket.ByClimate {
					proportions = candidate
					break
				}
			}
			return FeltTerrainChoice{
				Proportions: proportions,
				Confidence:  nonEmpty(bucket.Confidence, "from_data"),
				Reason:      fmt.Sprintf("water distance %.1f tiles, climate=%s", shoreDistance, climate),
			}
		}
	}
	if byBand := mapping.LandByClimateAndBand[climate]; byBand != nil {
		if choice, ok := byBand[band]; ok {
			return choice
		}
	}
	for _, fallback := range climateFallbacks(climate, band) {
		if byBand := mapping.LandByClimateAndBand[fallback]; byBand != nil {
			if choice, ok := byBand[band]; ok {
				if choice.Confidence == "" {
					choice.Confidence = "default"
				}
				if choice.Reason == "" {
					choice.Reason = fmt.Sprintf("%s fallback for climate=%s band=%s", fallback, climate, band)
				}
				return choice
			}
		}
	}
	if byBand := mapping.LandByClimateAndBand["temperate"]; byBand != nil {
		if choice, ok := byBand[band]; ok {
			if choice.Confidence == "" {
				choice.Confidence = "default"
			}
			if choice.Reason == "" {
				choice.Reason = fmt.Sprintf("default temperate fallback for climate=%s band=%s", climate, band)
			}
			return choice
		}
	}
	return FeltTerrainChoice{}
}

func climateFallbacks(climate, band string) []string {
	switch climate {
	case "humid_continental":
		if band == "beach" {
			return []string{"boreal", "temperate"}
		}
		return []string{"temperate", "boreal"}
	case "cold":
		return []string{"boreal", "temperate"}
	default:
		return nil
	}
}

func (mapping FeltMapping) landcoverChoice(classKey string) FeltTerrainChoice {
	if mapping.LandcoverClassMapping == nil || mapping.LandcoverClassMapping.ByClass == nil {
		return FeltTerrainChoice{}
	}
	return mapping.LandcoverClassMapping.ByClass[classKey]
}

func (mapping FeltMapping) blendedLandcoverChoice(stats []LandcoverStat) FeltTerrainChoice {
	if len(stats) == 0 {
		return FeltTerrainChoice{}
	}
	weights := map[int]float64{}
	total := 0.0
	confidence := "from_data"
	for _, stat := range stats {
		choice := mapping.landcoverChoice(stat.Key)
		if len(choice.Proportions) == 0 || stat.Weight <= 0 {
			continue
		}
		total += stat.Weight
		if choice.Confidence != "" {
			confidence = choice.Confidence
		}
		for _, p := range choice.Proportions {
			if p.Weight > 0 {
				weights[p.TerrainID] += p.Weight * stat.Weight
			}
		}
	}
	if total <= 0 || len(weights) == 0 {
		return FeltTerrainChoice{}
	}
	proportions := make([]TerrainWeight, 0, len(weights))
	for terrainID, weight := range weights {
		proportions = append(proportions, TerrainWeight{TerrainID: terrainID, Weight: weight / total})
	}
	sort.Slice(proportions, func(i, j int) bool { return proportions[i].TerrainID < proportions[j].TerrainID })
	return FeltTerrainChoice{Proportions: proportions, Confidence: confidence}
}

func (mapping FeltMapping) speciesUnitConsts(classKey, climate string) []int {
	if classKey == "" || mapping.LandcoverSpeciesByClass == nil {
		return nil
	}
	byClimate := mapping.LandcoverSpeciesByClass[classKey]
	if byClimate == nil {
		return nil
	}
	candidates := byClimate[climate]
	if len(candidates) == 0 {
		candidates = byClimate["general"]
	}
	if len(candidates) == 0 && climate == "temperate" {
		candidates = byClimate["humid_continental"]
	}
	if len(candidates) == 0 {
		for _, species := range byClimate {
			candidates = species
			break
		}
	}
	seen := map[int]bool{}
	var out []int
	for _, item := range candidates {
		if item.UnitConst == 0 || seen[item.UnitConst] {
			continue
		}
		seen[item.UnitConst] = true
		out = append(out, item.UnitConst)
	}
	return out
}

func chooseTerrain(x, y int, proportions []TerrainWeight) int {
	if len(proportions) == 0 {
		return 0
	}
	total := 0.0
	for _, p := range proportions {
		if p.Weight > 0 {
			total += p.Weight
		}
	}
	if total <= 0 {
		return proportions[0].TerrainID
	}
	roll := hashUnitFloat(x, y, 9137) * total
	acc := 0.0
	for _, p := range proportions {
		if p.Weight <= 0 {
			continue
		}
		acc += p.Weight
		if roll <= acc {
			return p.TerrainID
		}
	}
	return proportions[len(proportions)-1].TerrainID
}

func hashUnitFloat(x, y, seed int) float64 {
	n := uint64(uint32(x))*0x9e3779b185ebca87 ^ uint64(uint32(y))*0xc2b2ae3d27d4eb4f ^ uint64(uint32(seed))*0x165667b19e3779f9
	n ^= n >> 33
	n *= 0xff51afd7ed558ccd
	n ^= n >> 33
	n *= 0xc4ceb9fe1a85ec53
	n ^= n >> 33
	return float64(n&0x1fffffffffffff) / float64(0x20000000000000)
}

func nonEmpty(value, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}

func approximateTileMeters(b BBox, n int) float64 {
	latMeters := math.Abs(b.MaxLat-b.MinLat) * 111_320
	meanLat := (b.MinLat + b.MaxLat) / 2
	lonMeters := math.Abs(b.MaxLon-b.MinLon) * 111_320 * math.Cos(meanLat*math.Pi/180)
	if latMeters > lonMeters {
		return latMeters / float64(n)
	}
	return lonMeters / float64(n)
}

func sampleLandmask(opts Options, points []Cell) ([]bool, string, error) {
	provider := strings.ToLower(strings.TrimSpace(opts.LandmaskProvider))
	if provider == "" && opts.LandmaskPath != "" {
		provider = "geojson"
	}
	if provider == "" {
		return nil, "", nil
	}
	var (
		polygons []polygon
		err      error
	)
	switch provider {
	case "naturalearth", "natural-earth":
		polygons, err = fetchNaturalEarthLand(opts.Client)
		provider = "naturalearth"
	case "geojson":
		if opts.LandmaskPath == "" {
			return nil, "", fmt.Errorf("geojson landmask requires landmask path")
		}
		data, readErr := os.ReadFile(opts.LandmaskPath)
		if readErr != nil {
			return nil, "", readErr
		}
		polygons, err = parseGeoJSONPolygons(data)
	default:
		return nil, "", fmt.Errorf("unsupported landmask provider %q", opts.LandmaskProvider)
	}
	if err != nil {
		return nil, "", err
	}
	if len(polygons) == 0 {
		return nil, "", fmt.Errorf("landmask provider %q returned no polygons", provider)
	}
	return classifyPointsByPolygons(points, polygons), provider, nil
}

func sampleInlandWater(opts Options, points []Cell) ([]bool, []string, string, error) {
	provider := strings.ToLower(strings.TrimSpace(opts.InlandWaterProvider))
	if provider == "" && opts.InlandWaterPath != "" {
		provider = "geojson"
	}
	if provider == "" {
		return nil, nil, "", nil
	}
	var (
		polygons []polygon
		err      error
	)
	switch provider {
	case "naturalearth", "natural-earth":
		polygons, err = fetchNaturalEarthLakes(opts.Client)
		provider = "naturalearth"
	case "geojson":
		if opts.InlandWaterPath == "" {
			return nil, nil, "", fmt.Errorf("geojson inland water requires inland water path")
		}
		data, readErr := os.ReadFile(opts.InlandWaterPath)
		if readErr != nil {
			return nil, nil, "", readErr
		}
		polygons, err = parseGeoJSONPolygons(data)
	default:
		return nil, nil, "", fmt.Errorf("unsupported inland water provider %q", opts.InlandWaterProvider)
	}
	if err != nil {
		return nil, nil, "", err
	}
	if len(polygons) == 0 {
		return nil, nil, "", fmt.Errorf("inland water provider %q returned no polygons", provider)
	}
	mask, names := classifyPointsByNamedPolygons(points, polygons)
	return mask, names, provider, nil
}

func fetchNaturalEarthLand(client *http.Client) ([]polygon, error) {
	if client == nil {
		client = &http.Client{Timeout: 60 * time.Second}
	}
	const naturalEarthLandURL = "https://raw.githubusercontent.com/nvkelso/natural-earth-vector/master/geojson/ne_10m_land.geojson"
	resp, err := client.Get(naturalEarthLandURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("Natural Earth landmask status %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return nil, err
	}
	return parseGeoJSONPolygons(data)
}

func fetchNaturalEarthLakes(client *http.Client) ([]polygon, error) {
	if client == nil {
		client = &http.Client{Timeout: 60 * time.Second}
	}
	const naturalEarthLakesURL = "https://raw.githubusercontent.com/nvkelso/natural-earth-vector/master/geojson/ne_10m_lakes.geojson"
	resp, err := client.Get(naturalEarthLakesURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("Natural Earth lakes status %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return nil, err
	}
	return parseGeoJSONPolygons(data)
}

type point struct {
	Lon float64
	Lat float64
}

type polygon struct {
	Name   string
	Rings  [][]point
	MinLon float64
	MinLat float64
	MaxLon float64
	MaxLat float64
}

func parseGeoJSONPolygons(data []byte) ([]polygon, error) {
	var root struct {
		Type        string          `json:"type"`
		Coordinates json.RawMessage `json:"coordinates"`
		Geometry    *struct {
			Type        string          `json:"type"`
			Coordinates json.RawMessage `json:"coordinates"`
		} `json:"geometry"`
		Features []struct {
			Properties map[string]any `json:"properties"`
			Geometry   *struct {
				Type        string          `json:"type"`
				Coordinates json.RawMessage `json:"coordinates"`
			} `json:"geometry"`
		} `json:"features"`
	}
	if err := json.Unmarshal(data, &root); err != nil {
		return nil, err
	}
	var out []polygon
	addGeometry := func(kind string, coords json.RawMessage, name string) error {
		switch kind {
		case "Polygon":
			var raw [][][]float64
			if err := json.Unmarshal(coords, &raw); err != nil {
				return err
			}
			poly, ok := coordsToPolygon(raw)
			if ok {
				poly.Name = name
				out = append(out, poly)
			}
		case "MultiPolygon":
			var raw [][][][]float64
			if err := json.Unmarshal(coords, &raw); err != nil {
				return err
			}
			for _, rawPoly := range raw {
				poly, ok := coordsToPolygon(rawPoly)
				if ok {
					poly.Name = name
					out = append(out, poly)
				}
			}
		}
		return nil
	}
	switch root.Type {
	case "FeatureCollection":
		for _, feature := range root.Features {
			if feature.Geometry == nil {
				continue
			}
			if err := addGeometry(feature.Geometry.Type, feature.Geometry.Coordinates, geoJSONFeatureName(feature.Properties)); err != nil {
				return nil, err
			}
		}
	case "Feature":
		if root.Geometry != nil {
			if err := addGeometry(root.Geometry.Type, root.Geometry.Coordinates, ""); err != nil {
				return nil, err
			}
		}
	case "Polygon", "MultiPolygon":
		if err := addGeometry(root.Type, root.Coordinates, ""); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported GeoJSON type %q", root.Type)
	}
	return out, nil
}

func geoJSONFeatureName(properties map[string]any) string {
	for _, key := range []string{"name", "name_en", "gn_name", "label"} {
		if value, ok := properties[key]; ok {
			if s, ok := value.(string); ok && s != "" {
				return s
			}
		}
	}
	return ""
}

func coordsToPolygon(raw [][][]float64) (polygon, bool) {
	poly := polygon{
		MinLon: math.MaxFloat64,
		MinLat: math.MaxFloat64,
		MaxLon: -math.MaxFloat64,
		MaxLat: -math.MaxFloat64,
	}
	for _, rawRing := range raw {
		ring := make([]point, 0, len(rawRing))
		for _, coord := range rawRing {
			if len(coord) < 2 {
				continue
			}
			p := point{Lon: coord[0], Lat: coord[1]}
			ring = append(ring, p)
			if p.Lon < poly.MinLon {
				poly.MinLon = p.Lon
			}
			if p.Lon > poly.MaxLon {
				poly.MaxLon = p.Lon
			}
			if p.Lat < poly.MinLat {
				poly.MinLat = p.Lat
			}
			if p.Lat > poly.MaxLat {
				poly.MaxLat = p.Lat
			}
		}
		if len(ring) >= 4 {
			poly.Rings = append(poly.Rings, ring)
		}
	}
	return poly, len(poly.Rings) > 0
}

func classifyPointsByPolygons(points []Cell, polygons []polygon) []bool {
	out := make([]bool, len(points))
	for i, cell := range points {
		for _, poly := range polygons {
			if cell.Lon < poly.MinLon || cell.Lon > poly.MaxLon || cell.Lat < poly.MinLat || cell.Lat > poly.MaxLat {
				continue
			}
			if polygonContains(poly, point{Lon: cell.Lon, Lat: cell.Lat}) {
				out[i] = true
				break
			}
		}
	}
	return out
}

func classifyPointsByNamedPolygons(points []Cell, polygons []polygon) ([]bool, []string) {
	mask := make([]bool, len(points))
	names := make([]string, len(points))
	for i, cell := range points {
		for _, poly := range polygons {
			if cell.Lon < poly.MinLon || cell.Lon > poly.MaxLon || cell.Lat < poly.MinLat || cell.Lat > poly.MaxLat {
				continue
			}
			if polygonContains(poly, point{Lon: cell.Lon, Lat: cell.Lat}) {
				mask[i] = true
				names[i] = poly.Name
				break
			}
		}
	}
	return mask, names
}

func polygonContains(poly polygon, p point) bool {
	if len(poly.Rings) == 0 || !ringContains(poly.Rings[0], p) {
		return false
	}
	for _, hole := range poly.Rings[1:] {
		if ringContains(hole, p) {
			return false
		}
	}
	return true
}

func ringContains(ring []point, p point) bool {
	inside := false
	j := len(ring) - 1
	for i := range ring {
		pi := ring[i]
		pj := ring[j]
		if (pi.Lat > p.Lat) != (pj.Lat > p.Lat) {
			x := (pj.Lon-pi.Lon)*(p.Lat-pi.Lat)/(pj.Lat-pi.Lat) + pi.Lon
			if p.Lon < x {
				inside = !inside
			}
		}
		j = i
	}
	return inside
}

func distanceToOppositeMask(land []bool, n int) []float64 {
	out := make([]float64, n*n)
	if land == nil {
		for i := range out {
			out[i] = math.NaN()
		}
		return out
	}
	distToLand := distanceToClass(land, n, true)
	distToWater := distanceToClass(land, n, false)
	for i := range out {
		if land[i] {
			out[i] = float64(distToWater[i])
		} else {
			out[i] = float64(distToLand[i])
		}
	}
	return out
}

func distanceToClass(land []bool, n int, target bool) []int {
	const inf = int(^uint(0) >> 1)
	dist := make([]int, n*n)
	queue := make([]int, 0, n*n)
	for i, v := range land {
		if v == target {
			dist[i] = 0
			queue = append(queue, i)
		} else {
			dist[i] = inf
		}
	}
	for head := 0; head < len(queue); head++ {
		idx := queue[head]
		x := idx % n
		y := idx / n
		next := dist[idx] + 1
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				if dx == 0 && dy == 0 {
					continue
				}
				xx := x + dx
				yy := y + dy
				if xx < 0 || yy < 0 || xx >= n || yy >= n {
					continue
				}
				other := yy*n + xx
				if next < dist[other] {
					dist[other] = next
					queue = append(queue, other)
				}
			}
		}
	}
	for i, v := range dist {
		if v == inf {
			dist[i] = 0
		}
	}
	return dist
}

func BuildRecipe(report Report) scenario.Recipe {
	land := make([]scenario.MaskCellRecipe, 0, len(report.Cells))
	water := make([]scenario.MaskCellRecipe, 0, len(report.Cells))
	beach := make([]scenario.MaskCellRecipe, 0, len(report.Cells))
	lowland := make([]scenario.MaskCellRecipe, 0, len(report.Cells))
	upland := make([]scenario.MaskCellRecipe, 0, len(report.Cells))
	highland := make([]scenario.MaskCellRecipe, 0, len(report.Cells))
	terrainCells := map[int][]scenario.MaskCellRecipe{}
	landcoverCells := map[string][]Cell{}
	var roadCells []scenario.MaskCellRecipe
	var roadsideCells []scenario.MaskCellRecipe
	var borderCells []scenario.MaskCellRecipe
	borderCellsByTerrain := map[int][]scenario.MaskCellRecipe{}
	shoreBandCellsByTerrain := map[int][]scenario.MaskCellRecipe{}
	var shoreMediumCells []scenario.MaskCellRecipe
	var shoreDeepCells []scenario.MaskCellRecipe
	shoreLayerCells := map[[2]int][]scenario.MaskCellRecipe{}
	for _, cell := range report.Cells {
		maskCell := scenario.MaskCellRecipe{X: cell.X, Y: cell.Y}
		shoreTerrainID := cell.ShoreTerrainID
		if shoreTerrainID == 0 {
			shoreTerrainID = 2
		}
		terrainCells[cell.TerrainID] = append(terrainCells[cell.TerrainID], maskCell)
		if cell.Road {
			roadCells = append(roadCells, maskCell)
		}
		if cell.Roadside {
			roadsideCells = append(roadsideCells, maskCell)
		}
		if cell.Border {
			borderCells = append(borderCells, maskCell)
			borderCellsByTerrain[cell.TerrainID] = append(borderCellsByTerrain[cell.TerrainID], maskCell)
		}
		switch cell.ShoreCrossfadeBand {
		case "A_over_B":
			shoreLayerCells[[2]int{shoreTerrainID, cell.TerrainID}] = append(shoreLayerCells[[2]int{shoreTerrainID, cell.TerrainID}], maskCell)
		case "B_over_A":
			shoreLayerCells[[2]int{cell.TerrainID, shoreTerrainID}] = append(shoreLayerCells[[2]int{cell.TerrainID, shoreTerrainID}], maskCell)
		case "A":
			// Current terrain is already the shore-water A band.
		case "A_over_M":
			shoreLayerCells[[2]int{23, cell.TerrainID}] = append(shoreLayerCells[[2]int{23, cell.TerrainID}], maskCell)
		case "M":
			shoreMediumCells = append(shoreMediumCells, maskCell)
		case "M_over_D":
			shoreLayerCells[[2]int{57, 23}] = append(shoreLayerCells[[2]int{57, 23}], maskCell)
		case "D":
			shoreDeepCells = append(shoreDeepCells, maskCell)
		case "B":
			shoreBandCellsByTerrain[shoreTerrainID] = append(shoreBandCellsByTerrain[shoreTerrainID], maskCell)
		case "B_over_C":
			shoreLayerCells[[2]int{cell.TerrainID, shoreTerrainID}] = append(shoreLayerCells[[2]int{cell.TerrainID, shoreTerrainID}], maskCell)
		case "C_over_B":
			shoreLayerCells[[2]int{shoreTerrainID, cell.TerrainID}] = append(shoreLayerCells[[2]int{shoreTerrainID, cell.TerrainID}], maskCell)
		}
		if cell.LandcoverKey != "" {
			landcoverCells[cell.LandcoverKey] = append(landcoverCells[cell.LandcoverKey], cell)
		}
		if cell.Land != nil && *cell.Land {
			land = append(land, maskCell)
			switch cell.ElevationBand {
			case "beach":
				beach = append(beach, maskCell)
			case "lowland":
				lowland = append(lowland, maskCell)
			case "upland":
				upland = append(upland, maskCell)
			case "highland":
				highland = append(highland, maskCell)
			}
			continue
		}
		water = append(water, maskCell)
	}
	onePlayer := 1
	recipe := scenario.Recipe{
		Scenario: &scenario.ScenarioRecipe{PlayerCount: &onePlayer},
	}
	appendNonEmptyMask := func(name string, cells []scenario.MaskCellRecipe) {
		if len(cells) == 0 {
			return
		}
		recipe.Masks = append(recipe.Masks, scenario.MaskRecipe{Name: name, Op: "cells", Cells: cells})
	}
	appendNonEmptyMask("land", land)
	appendNonEmptyMask("water", water)
	appendNonEmptyMask("beach", beach)
	appendNonEmptyMask("lowland", lowland)
	appendNonEmptyMask("upland", upland)
	appendNonEmptyMask("highland", highland)
	classKeys := make([]string, 0, len(landcoverCells))
	for key := range landcoverCells {
		classKeys = append(classKeys, key)
	}
	sort.Strings(classKeys)
	for _, key := range classKeys {
		maskCells := make([]scenario.MaskCellRecipe, 0, len(landcoverCells[key]))
		for _, cell := range landcoverCells[key] {
			maskCells = append(maskCells, scenario.MaskCellRecipe{X: cell.X, Y: cell.Y})
		}
		appendNonEmptyMask("landcover_"+key, maskCells)
	}
	terrainIDs := make([]int, 0, len(terrainCells))
	for terrainID := range terrainCells {
		terrainIDs = append(terrainIDs, terrainID)
	}
	sort.Ints(terrainIDs)
	for _, terrainID := range terrainIDs {
		name := fmt.Sprintf("terrain_%d", terrainID)
		appendNonEmptyMask(name, terrainCells[terrainID])
		recipe.Map = append(recipe.Map, scenario.MapRecipe{
			Op:        "set_terrain_mask",
			Mask:      name,
			TerrainID: intPtr(terrainID),
		})
	}
	if len(water) > 0 {
		recipe.Map = append(recipe.Map, scenario.MapRecipe{
			Op:         "semantic_erode",
			Mask:       "water",
			TerrainIDs: []int{23, 57, 58},
			Iterations: intPtr(1),
		})
	}
	for _, key := range classKeys {
		terrainIDs := terrainIDsForLandcoverKey(key)
		if len(terrainIDs) == 0 {
			continue
		}
		recipe.Map = append(recipe.Map, scenario.MapRecipe{
			Op:         "semantic_erode",
			Mask:       "landcover_" + key,
			TerrainIDs: terrainIDs,
			Iterations: intPtr(1),
		})
		if len(terrainIDs) >= 2 {
			scale := 9.0
			threshold := 0.56
			seed := 24000 + len(recipe.Map)*17
			recipe.Map = append(recipe.Map, scenario.MapRecipe{
				Op:         "noise_fill",
				Mask:       "landcover_" + key,
				TerrainID:  intPtr(terrainIDs[0]),
				TerrainID2: intPtr(terrainIDs[1]),
				Seed:       intPtr(seed),
				Scale:      float64Ptr(scale),
				Threshold:  float64Ptr(threshold),
			})
		}
	}
	appendPolarCaps(report, &recipe)
	shoreTerrainIDs := make([]int, 0, len(shoreBandCellsByTerrain))
	for terrainID := range shoreBandCellsByTerrain {
		shoreTerrainIDs = append(shoreTerrainIDs, terrainID)
	}
	sort.Ints(shoreTerrainIDs)
	for _, terrainID := range shoreTerrainIDs {
		name := fmt.Sprintf("shore_band_terrain_%d", terrainID)
		appendNonEmptyMask(name, shoreBandCellsByTerrain[terrainID])
		recipe.Map = append(recipe.Map, scenario.MapRecipe{
			Op:        "set_terrain_mask",
			Mask:      name,
			TerrainID: intPtr(terrainID),
		})
	}
	appendNonEmptyMask("shore_band_medium_water", shoreMediumCells)
	if len(shoreMediumCells) > 0 {
		recipe.Map = append(recipe.Map, scenario.MapRecipe{
			Op:        "set_terrain_mask",
			Mask:      "shore_band_medium_water",
			TerrainID: intPtr(23),
		})
	}
	appendNonEmptyMask("shore_band_deep_water", shoreDeepCells)
	if len(shoreDeepCells) > 0 {
		recipe.Map = append(recipe.Map, scenario.MapRecipe{
			Op:        "set_terrain_mask",
			Mask:      "shore_band_deep_water",
			TerrainID: intPtr(57),
		})
	}
	shoreLayerKeys := make([][2]int, 0, len(shoreLayerCells))
	for key := range shoreLayerCells {
		shoreLayerKeys = append(shoreLayerKeys, key)
	}
	sort.Slice(shoreLayerKeys, func(i, j int) bool {
		if shoreLayerKeys[i][0] != shoreLayerKeys[j][0] {
			return shoreLayerKeys[i][0] < shoreLayerKeys[j][0]
		}
		return shoreLayerKeys[i][1] < shoreLayerKeys[j][1]
	})
	for _, key := range shoreLayerKeys {
		name := fmt.Sprintf("shore_crossfade_%d_over_%d", key[1], key[0])
		appendNonEmptyMask(name, shoreLayerCells[key])
		recipe.Map = append(recipe.Map, scenario.MapRecipe{
			Op:         "layered_crossfade",
			Mask:       name,
			TerrainID:  intPtr(key[0]),
			TerrainID2: intPtr(key[1]),
		})
	}
	appendNonEmptyMask("roadside", roadsideCells)
	if len(roadsideCells) > 0 {
		recipe.Map = append(recipe.Map, scenario.MapRecipe{
			Op:        "set_terrain_mask",
			Mask:      "roadside",
			TerrainID: intPtr(6),
		})
	}
	appendNonEmptyMask("roads", roadCells)
	if len(roadCells) > 0 {
		recipe.Map = append(recipe.Map, scenario.MapRecipe{
			Op:        "set_terrain_mask",
			Mask:      "roads",
			TerrainID: intPtr(roadTerrainID),
		})
	}
	appendNonEmptyMask("state_borders", borderCells)
	if len(borderCells) > 0 {
		borderTerrainIDs := make([]int, 0, len(borderCellsByTerrain))
		for terrainID := range borderCellsByTerrain {
			borderTerrainIDs = append(borderTerrainIDs, terrainID)
		}
		sort.Ints(borderTerrainIDs)
		for _, terrainID := range borderTerrainIDs {
			name := fmt.Sprintf("state_border_on_%d", terrainID)
			appendNonEmptyMask(name, borderCellsByTerrain[terrainID])
			recipe.Map = append(recipe.Map, scenario.MapRecipe{
				Op:         "layered_crossfade",
				Mask:       name,
				TerrainID:  intPtr(borderTerrainID),
				TerrainID2: intPtr(terrainID),
			})
		}
	}
	if len(landcoverCells) > 0 {
		recipe.Units = append(recipe.Units, landcoverVegetation(report, landcoverCells)...)
	} else {
		recipe.Units = append(recipe.Units, fallbackWorldVegetation(report)...)
	}
	recipe.Units = append(recipe.Units, projectionCalibrationMarkers(report)...)
	return recipe
}

func projectionCalibrationMarkers(report Report) []scenario.UnitRecipe {
	if report.GridSize <= 0 || report.Orient != "iso-ccw-screen" {
		return nil
	}
	proj := newGeoProjection(report.BBox, report.Orient)
	offsetKm := 0.82 * math.Min(proj.halfEastKm, proj.halfNorthKm)
	if offsetKm <= 0 {
		return nil
	}
	type marker struct {
		label string
		lat   float64
		lon   float64
		owner int
	}
	markers := []marker{
		{
			label: "A2K CALIBRATION TRUE NORTH - should sit at TOP corner on minimap",
			lat:   proj.midLat + offsetKm/proj.kmPerLat,
			lon:   proj.midLon,
			owner: 1,
		},
		{
			label: "A2K CALIBRATION TRUE EAST - should sit at RIGHT corner on minimap",
			lat:   proj.midLat,
			lon:   proj.midLon + offsetKm/proj.kmPerLon,
			owner: 0,
		},
	}
	status := 2
	z := 0.0
	var out []scenario.UnitRecipe
	for _, marker := range markers {
		x, y, ok := gridCellForLatLon(report.BBox, report.GridSize, report.Orient, marker.lat, marker.lon)
		if !ok {
			continue
		}
		fx := float64(x) + 0.5
		fy := float64(y) + 0.5
		rotation := 0.0
		out = append(out, scenario.UnitRecipe{
			Op:            "add_unit",
			Player:        marker.owner,
			UnitConst:     600,
			X:             &fx,
			Y:             &fy,
			Z:             &z,
			Status:        &status,
			Rotation:      &rotation,
			CaptionString: marker.label,
		})
	}
	return out
}

func appendPolarCaps(report Report, recipe *scenario.Recipe) {
	var northIce, southIce, northSnow, southSnow []scenario.MaskCellRecipe
	for _, cell := range report.Cells {
		if math.Abs(cell.Lat) < 66 {
			continue
		}
		maskCell := scenario.MaskCellRecipe{X: cell.X, Y: cell.Y}
		land := cell.Land != nil && *cell.Land
		if cell.Lat >= 66 {
			if land {
				northSnow = append(northSnow, maskCell)
			} else {
				northIce = append(northIce, maskCell)
			}
			continue
		}
		if land {
			southSnow = append(southSnow, maskCell)
		} else {
			southIce = append(southIce, maskCell)
		}
	}
	appendMaskAndPaint := func(name string, cells []scenario.MaskCellRecipe, terrainID int) {
		if len(cells) == 0 {
			return
		}
		recipe.Masks = append(recipe.Masks, scenario.MaskRecipe{Name: name, Op: "cells", Cells: cells})
		recipe.Map = append(recipe.Map, scenario.MapRecipe{
			Op:        "set_terrain_mask",
			Mask:      name,
			TerrainID: intPtr(terrainID),
		})
	}
	appendMaskAndPaint("polar_north_ice", northIce, 35)
	appendMaskAndPaint("polar_south_ice", southIce, 35)
	appendMaskAndPaint("polar_north_snow", northSnow, 32)
	appendMaskAndPaint("polar_south_snow", southSnow, 32)
}

func BuildEastWestWrapTriggers(gridSize int, players []int) []scenario.TriggerRecipe {
	if gridSize <= 4 || len(players) == 0 {
		return nil
	}
	enabled := true
	looping := true
	quantity := 1
	maxAffected := -1
	leftX1, leftX2 := 0, 1
	rightX1, rightX2 := gridSize-2, gridSize-1
	leftDestX := 3
	rightDestX := gridSize - 4
	if rightDestX < 1 {
		rightDestX = 1
	}
	var triggers []scenario.TriggerRecipe
	for _, player := range players {
		if player < 1 || player > 8 {
			continue
		}
		for y := 0; y < gridSize; y++ {
			row := y
			triggers = append(triggers, scenario.TriggerRecipe{
				Op:      "add_trigger",
				Name:    fmt.Sprintf("A2K Geo Wrap P%d west row %03d", player, y),
				Enabled: &enabled,
				Looping: &looping,
				Conditions: []scenario.ConditionRecipe{{
					Op:           "objects_in_area",
					SourcePlayer: intPtr(player),
					Quantity:     &quantity,
					AreaX1:       &leftX1,
					AreaY1:       &row,
					AreaX2:       &leftX2,
					AreaY2:       &row,
				}},
				Effects: []scenario.EffectRecipe{{
					Op:               "teleport_object",
					SourcePlayer:     intPtr(player),
					LocationX:        &rightDestX,
					LocationY:        &row,
					AreaX1:           &leftX1,
					AreaY1:           &row,
					AreaX2:           &leftX2,
					AreaY2:           &row,
					MaxUnitsAffected: &maxAffected,
				}},
			})
			triggers = append(triggers, scenario.TriggerRecipe{
				Op:      "add_trigger",
				Name:    fmt.Sprintf("A2K Geo Wrap P%d east row %03d", player, y),
				Enabled: &enabled,
				Looping: &looping,
				Conditions: []scenario.ConditionRecipe{{
					Op:           "objects_in_area",
					SourcePlayer: intPtr(player),
					Quantity:     &quantity,
					AreaX1:       &rightX1,
					AreaY1:       &row,
					AreaX2:       &rightX2,
					AreaY2:       &row,
				}},
				Effects: []scenario.EffectRecipe{{
					Op:               "teleport_object",
					SourcePlayer:     intPtr(player),
					LocationX:        &leftDestX,
					LocationY:        &row,
					AreaX1:           &rightX1,
					AreaY1:           &row,
					AreaX2:           &rightX2,
					AreaY2:           &row,
					MaxUnitsAffected: &maxAffected,
				}},
			})
		}
	}
	return triggers
}

func BuildDiamondLongitudeWrapTriggers(gridSize int, players []int, bandWidth int) []scenario.TriggerRecipe {
	if gridSize <= 8 || len(players) == 0 {
		return nil
	}
	if bandWidth <= 0 {
		bandWidth = 2
	}
	enabled := true
	looping := true
	quantity := 1
	maxAffected := -1
	var triggers []scenario.TriggerRecipe
	for _, player := range players {
		if player < 1 || player > 8 {
			continue
		}
		for sum := 0; sum <= 2*gridSize-2; sum++ {
			xMin, xMax, length := diagonalRange(gridSize, sum)
			if length < bandWidth*2+3 {
				continue
			}
			for offset := 0; offset < bandWidth; offset++ {
				westX := xMin + offset
				westY := sum - westX
				eastX := xMax - offset
				eastY := sum - eastX
				westDestX := xMax - bandWidth - 1 + offset
				westDestY := sum - westDestX
				eastDestX := xMin + bandWidth + 1 - offset
				eastDestY := sum - eastDestX
				triggers = append(triggers, scenario.TriggerRecipe{
					Op:      "add_trigger",
					Name:    fmt.Sprintf("A2K Geo Diamond Wrap P%d west lat %03d.%d", player, sum, offset),
					Enabled: &enabled,
					Looping: &looping,
					Conditions: []scenario.ConditionRecipe{{
						Op:           "objects_in_area",
						SourcePlayer: intPtr(player),
						Quantity:     &quantity,
						AreaX1:       &westX,
						AreaY1:       &westY,
						AreaX2:       &westX,
						AreaY2:       &westY,
					}},
					Effects: []scenario.EffectRecipe{{
						Op:               "teleport_object",
						SourcePlayer:     intPtr(player),
						LocationX:        &westDestX,
						LocationY:        &westDestY,
						AreaX1:           &westX,
						AreaY1:           &westY,
						AreaX2:           &westX,
						AreaY2:           &westY,
						MaxUnitsAffected: &maxAffected,
					}},
				})
				triggers = append(triggers, scenario.TriggerRecipe{
					Op:      "add_trigger",
					Name:    fmt.Sprintf("A2K Geo Diamond Wrap P%d east lat %03d.%d", player, sum, offset),
					Enabled: &enabled,
					Looping: &looping,
					Conditions: []scenario.ConditionRecipe{{
						Op:           "objects_in_area",
						SourcePlayer: intPtr(player),
						Quantity:     &quantity,
						AreaX1:       &eastX,
						AreaY1:       &eastY,
						AreaX2:       &eastX,
						AreaY2:       &eastY,
					}},
					Effects: []scenario.EffectRecipe{{
						Op:               "teleport_object",
						SourcePlayer:     intPtr(player),
						LocationX:        &eastDestX,
						LocationY:        &eastDestY,
						AreaX1:           &eastX,
						AreaY1:           &eastY,
						AreaX2:           &eastX,
						AreaY2:           &eastY,
						MaxUnitsAffected: &maxAffected,
					}},
				})
			}
		}
	}
	return triggers
}

func BuildSailDemoUnits(gridSize int) []scenario.UnitRecipe {
	if gridSize <= 0 {
		return nil
	}
	x := float64(gridSize) / 2
	y := float64(gridSize) / 2
	z := 0.0
	status := 2
	rotation := 0.0
	return []scenario.UnitRecipe{{
		Op:        "add_unit",
		Player:    1,
		UnitConst: 545,
		X:         &x,
		Y:         &y,
		Z:         &z,
		Status:    &status,
		Rotation:  &rotation,
	}}
}

func filterInfrastructureFreeCells(cells []Cell, allowMangroveWaterBuffer bool) []Cell {
	if len(cells) == 0 {
		return nil
	}
	out := make([]Cell, 0, len(cells))
	for _, cell := range cells {
		if !cellBuildableForGaiaPlacement(cell) {
			continue
		}
		if cell.Road || cell.Roadside || cell.Border || cell.KeyWestCauseway {
			continue
		}
		if cell.WaterBuffer && !(allowMangroveWaterBuffer && cell.LandcoverKey == "95_mangrove") {
			continue
		}
		out = append(out, cell)
	}
	return out
}

func roadsidePlantCells(report Report) []Cell {
	out := make([]Cell, 0, report.RoadsideCells)
	for _, cell := range report.Cells {
		if cell.Roadside && cellBuildableForGaiaPlacement(cell) {
			out = append(out, cell)
		}
	}
	return out
}

func coastalPalmCells(report Report) []Cell {
	out := make([]Cell, 0, report.WaterBufferCells)
	for _, cell := range report.Cells {
		if !cellBuildableForGaiaPlacement(cell) {
			continue
		}
		if cell.Land == nil || !*cell.Land || cell.Road || cell.Roadside || cell.Border || cell.KeyWestCauseway {
			continue
		}
		if !cell.WaterBuffer {
			continue
		}
		if cell.ClimateZone != "tropical" && cell.ClimateZone != "subtropical" {
			continue
		}
		if cell.LandcoverKey == "95_mangrove" || cell.LandcoverKey == "90_herbaceous_wetland" {
			continue
		}
		out = append(out, cell)
	}
	return out
}

func alligatorAlleyCells(report Report) []Cell {
	if !bboxContains(report.BBox, 26.15, -81.5) || !bboxContains(report.BBox, 26.05, -80.2) {
		return nil
	}
	var out []Cell
	for _, cell := range report.Cells {
		if !cellBuildableForGaiaPlacement(cell) {
			continue
		}
		if cell.Land == nil || !*cell.Land || cell.Road || cell.Roadside || cell.Border {
			continue
		}
		if cell.Lat < 25.75 || cell.Lat > 26.45 || cell.Lon < -82.1 || cell.Lon > -80.05 {
			continue
		}
		if cell.LandcoverKey == "90_herbaceous_wetland" || cell.LandcoverKey == "95_mangrove" || cell.TerrainID == 111 || cell.TerrainID == 55 || cell.TerrainID == 101 || cell.TerrainID == 95 {
			out = append(out, cell)
		}
	}
	return out
}

func directUnitsFromCells(cells []Cell, unitConsts []int, stride, maxCount, seed int) []scenario.UnitRecipe {
	if len(cells) == 0 || len(unitConsts) == 0 {
		return nil
	}
	if stride < 1 {
		stride = 1
	}
	status := 2
	z := 0.0
	out := make([]scenario.UnitRecipe, 0, minInt(len(cells)/stride+1, maxCount))
	offset := seed % stride
	for i, cell := range cells {
		if i%stride != offset {
			continue
		}
		if maxCount > 0 && len(out) >= maxCount {
			break
		}
		jitterX := (hashUnitFloat(cell.X, cell.Y, seed) - 0.5) * 0.22
		jitterY := (hashUnitFloat(cell.X, cell.Y, seed+37) - 0.5) * 0.22
		x := float64(cell.X) + 0.5 + jitterX
		y := float64(cell.Y) + 0.5 + jitterY
		rotation := math.Floor(hashUnitFloat(cell.X, cell.Y, seed+73) * 32)
		out = append(out, scenario.UnitRecipe{
			Op:        "add_unit",
			Player:    0,
			UnitConst: unitConsts[(len(out)+seed)%len(unitConsts)],
			X:         &x,
			Y:         &y,
			Z:         &z,
			Status:    &status,
			Rotation:  &rotation,
		})
	}
	return out
}

type waterUnitCandidate struct {
	cell Cell
	hash float64
}

func sampledUnitsFromCells(cells []Cell, unitConsts []int, desired, maxCount, seed int) []scenario.UnitRecipe {
	if len(cells) == 0 || len(unitConsts) == 0 || desired <= 0 {
		return nil
	}
	if maxCount > 0 && desired > maxCount {
		desired = maxCount
	}
	if desired > len(cells) {
		desired = len(cells)
	}
	candidates := make([]waterUnitCandidate, 0, len(cells))
	for _, cell := range cells {
		candidates = append(candidates, waterUnitCandidate{
			cell: cell,
			hash: hashUnitFloat(cell.X, cell.Y, seed),
		})
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].hash == candidates[j].hash {
			if candidates[i].cell.Y == candidates[j].cell.Y {
				return candidates[i].cell.X < candidates[j].cell.X
			}
			return candidates[i].cell.Y < candidates[j].cell.Y
		}
		return candidates[i].hash < candidates[j].hash
	})
	status := 2
	z := 0.0
	out := make([]scenario.UnitRecipe, 0, desired)
	minDist2 := 25
	for pass := 0; pass < 2 && len(out) < desired; pass++ {
		for _, candidate := range candidates {
			if len(out) >= desired {
				break
			}
			if pass == 0 && tooCloseToPlaced(candidate.cell, out, minDist2) {
				continue
			}
			cell := candidate.cell
			jitterX := (hashUnitFloat(cell.X, cell.Y, seed+11) - 0.5) * 0.22
			jitterY := (hashUnitFloat(cell.X, cell.Y, seed+47) - 0.5) * 0.22
			x := float64(cell.X) + 0.5 + jitterX
			y := float64(cell.Y) + 0.5 + jitterY
			rotation := math.Floor(hashUnitFloat(cell.X, cell.Y, seed+83) * 32)
			out = append(out, scenario.UnitRecipe{
				Op:        "add_unit",
				Player:    0,
				UnitConst: unitConsts[(len(out)+seed)%len(unitConsts)],
				X:         &x,
				Y:         &y,
				Z:         &z,
				Status:    &status,
				Rotation:  &rotation,
			})
		}
		minDist2 = 9
	}
	return out
}

func tooCloseToPlaced(cell Cell, units []scenario.UnitRecipe, minDist2 int) bool {
	for _, unit := range units {
		if unit.X == nil || unit.Y == nil {
			continue
		}
		dx := float64(cell.X) + 0.5 - *unit.X
		dy := float64(cell.Y) + 0.5 - *unit.Y
		if dx*dx+dy*dy < float64(minDist2) {
			return true
		}
	}
	return false
}

func waterFaunaUnits(shore, lake, medium, deep []Cell) []scenario.UnitRecipe {
	var out []scenario.UnitRecipe
	// Shore fish are intentionally near-continuous so coasts read as harvestable.
	out = append(out, directUnitsFromCells(shore, []int{455, 458, 2170, 455, 458}, 5, 520, 71001)...)
	// Inland water uses a different species band than open ocean.
	out = append(out, sampledUnitsFromCells(lake, []int{53, 53, 53, 456}, maxInt(24, len(lake)/4), 720, 71002)...)
	// Open ocean is deliberately reduced to roughly a third of the old density.
	out = append(out, sampledUnitsFromCells(medium, []int{456}, maxInt(40, len(medium)/80), 180, 71003)...)
	out = append(out, sampledUnitsFromCells(deep, []int{457, 457, 450, 450, 2625}, maxInt(32, len(deep)/140), 130, 71004)...)
	return out
}

func fallbackWorldVegetation(report Report) []scenario.UnitRecipe {
	type scatterPlan struct {
		name            string
		cells           []Cell
		unitConst       int
		count           int
		density         float64
		centerLimit     int
		spread          float64
		minDist         float64
		clusterShape    string
		terrains        []int
		rotationChoices []float64
	}
	var tropicalLand, temperateLand, borealLand, polarLand, allPlayableLand []Cell
	for _, cell := range report.Cells {
		if cell.Land == nil || !*cell.Land {
			continue
		}
		if !cellBuildableForGaiaPlacement(cell) {
			continue
		}
		if cell.Road || cell.Roadside || cell.Border || cell.WaterBuffer || cell.KeyWestCauseway {
			continue
		}
		if math.Abs(cell.Lat) >= 66 {
			polarLand = append(polarLand, cell)
			continue
		}
		allPlayableLand = append(allPlayableLand, cell)
		switch cell.ClimateZone {
		case "tropical", "subtropical":
			tropicalLand = append(tropicalLand, cell)
		case "boreal":
			borealLand = append(borealLand, cell)
		default:
			temperateLand = append(temperateLand, cell)
		}
	}
	roadsidePlants := roadsidePlantCells(report)
	coastalPalms := coastalPalmCells(report)
	shoreWater, lakeWater, mediumWater, deepWater := waterResourceCells(report)
	plans := []scatterPlan{
		{name: "tropical_resource_groves", cells: tropicalLand, unitConst: 2567, count: minInt(5200, len(tropicalLand)/2), density: report.FloraDensity, centerLimit: 56, spread: 5.5, minDist: 0.45, clusterShape: "packed", terrains: []int{0, 10, 12, 55, 95, 100}, rotationChoices: rangeFloatChoices(0, 26)},
		{name: "temperate_resource_forest", cells: temperateLand, unitConst: 350, count: minInt(5200, len(temperateLand)/2), density: report.FloraDensity, centerLimit: 56, spread: 5.5, minDist: 0.45, clusterShape: "packed", terrains: []int{0, 10, 12, 100}, rotationChoices: floatChoices(3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 21, 22, 23, 24, 25, 26)},
		{name: "boreal_resource_forest", cells: borealLand, unitConst: 350, count: minInt(3800, len(borealLand)/2), density: report.FloraDensity, centerLimit: 44, spread: 5.2, minDist: 0.45, clusterShape: "packed", terrains: []int{0, 10, 32, 40, 100}, rotationChoices: floatChoices(3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 21, 22, 23, 24, 25, 26)},
		{name: "world_edge_trees", cells: allPlayableLand, unitConst: 2567, count: minInt(180, maxInt(1, len(allPlayableLand)/120)), density: math.Min(report.FloraDensity, 4), centerLimit: 60, spread: 2.0, minDist: 2.4, terrains: []int{0, 10, 12, 32, 40, 55, 95, 100, 111}, rotationChoices: rangeFloatChoices(0, 26)},
		{name: "roadside_sparse_palmetto", cells: roadsidePlants, unitConst: 1054, count: minInt(160, maxInt(1, len(roadsidePlants)/16)), density: 3, centerLimit: 36, spread: 2.0, minDist: 1.4, terrains: []int{6}, rotationChoices: rangeFloatChoices(0, 14)},
		{name: "coastal_palms", cells: coastalPalms, unitConst: 351, count: minInt(900, maxInt(1, len(coastalPalms)*4/5)), density: report.FloraDensity, centerLimit: 48, spread: 3.5, minDist: 0.55, clusterShape: "packed", terrains: []int{53}, rotationChoices: rangeFloatChoices(0, 38)},
		{name: "polar_sparse_cover", cells: polarLand, unitConst: 350, count: minInt(320, len(polarLand)/18), density: report.FloraDensity, centerLimit: 20, spread: 5.0, minDist: 0.6, clusterShape: "packed", terrains: []int{32, 35, 40}},
		{name: "deer_hunt_world", cells: allPlayableLand, unitConst: 65, count: minInt(520, maxInt(4, len(allPlayableLand)/70)), density: report.LandFaunaDensity, centerLimit: 120, spread: 7.0, minDist: 1.0, terrains: []int{0, 10, 12, 32, 40, 55, 95, 100, 111}},
		{name: "boar_hunt_world", cells: allPlayableLand, unitConst: 48, count: minInt(180, maxInt(2, len(allPlayableLand)/220)), density: report.LandFaunaDensity, centerLimit: 80, spread: 7.5, minDist: 1.7, terrains: []int{0, 10, 12, 55, 95, 100, 111}},
	}
	var out []scenario.UnitRecipe
	for i, plan := range plans {
		plan.count = densityScaledCount(plan.count, plan.density)
		if len(plan.cells) == 0 || plan.count <= 0 {
			continue
		}
		centers := representativeCenters(plan.cells, minInt(plan.centerLimit, maxInt(4, plan.count/90)))
		exact := false
		seed := 43000 + i*1543
		x1, y1 := 0.0, 0.0
		x2, y2 := float64(report.GridSize), float64(report.GridSize)
		out = append(out, scenario.UnitRecipe{
			Op:                "cluster_scatter",
			Player:            0,
			UnitConst:         plan.unitConst,
			Count:             intPtr(plan.count),
			Centers:           centers,
			ClusterShape:      plan.clusterShape,
			Spread:            float64Ptr(plan.spread),
			MinDistance:       float64Ptr(plan.minDist),
			AllowedTerrainIDs: pruneUnbuildableTerrainIDs(plan.terrains),
			Exact:             &exact,
			Seed:              intPtr(seed),
			RotationChoices:   plan.rotationChoices,
			TargetAreaX1:      &x1,
			TargetAreaY1:      &y1,
			TargetAreaX2:      &x2,
			TargetAreaY2:      &y2,
		})
	}
	out = append(out, waterFaunaUnits(shoreWater, lakeWater, mediumWater, deepWater)...)
	return out
}

func normalizeDensity(value float64) float64 {
	if value <= 0 {
		return 10
	}
	if value < 1 {
		return 1
	}
	if value > 10 {
		return 10
	}
	return value
}

func normalizeShoreBandWidth(value float64) float64 {
	if value <= 0 {
		return 1
	}
	if value < 0.5 {
		return 0.5
	}
	if value > 4 {
		return 4
	}
	return value
}

func normalizeWaterBandWidth(value float64) float64 {
	if value <= 0 {
		return 2
	}
	if value < 0.5 {
		return 0.5
	}
	if value > 8 {
		return 8
	}
	return value
}

func firstPositive(values ...float64) float64 {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}

func densityScaledCount(count int, density float64) int {
	if count <= 0 {
		return 0
	}
	scaled := int(math.Round(float64(count) * normalizeDensity(density) / 10))
	if scaled < 1 {
		return 1
	}
	return scaled
}

func terrainIDsForLandcoverKey(key string) []int {
	var ids []int
	switch key {
	case "10_tree_cover":
		ids = []int{10, 0}
	case "20_shrubland":
		ids = []int{0, 100}
	case "30_grassland":
		ids = []int{0, 12}
	case "40_cropland":
		ids = []int{7}
	case "50_built_up":
		ids = []int{6, 24}
	case "60_bare_sparse":
		ids = []int{40, 70, 14}
	case "70_snow_ice":
		ids = []int{32, 35}
	case "80_permanent_water":
		ids = []int{1, 23, 22}
	case "90_herbaceous_wetland":
		ids = []int{111, 55, 101, 95}
	case "95_mangrove":
		ids = []int{55, 95}
	case "100_moss_lichen":
		ids = []int{32, 10, 40}
	default:
		return nil
	}
	return ids
}

func pruneUnbuildableTerrainIDs(ids []int) []int {
	if len(ids) == 0 {
		return nil
	}
	out := make([]int, 0, len(ids))
	for _, id := range ids {
		if terrainUnbuildableForGaiaPlacement(id) {
			continue
		}
		out = append(out, id)
	}
	return out
}

func landcoverVegetation(report Report, byClass map[string][]Cell) []scenario.UnitRecipe {
	type scatterPlan struct {
		name            string
		cells           []Cell
		unitConst       int
		count           int
		density         float64
		centerLimit     int
		spread          float64
		minDist         float64
		clusterShape    string
		terrains        []int
		rotationChoices []float64
	}
	treeCells := nonIsolatedLandCells(filterInfrastructureFreeCells(byClass["10_tree_cover"], false), report.GridSize, 2)
	shoreWater, lakeWater, mediumWater, deepWater := waterResourceCells(report)
	forestGrass := append([]Cell{}, treeCells...)
	forestGrass = append(forestGrass, filterInfrastructureFreeCells(byClass["30_grassland"], false)...)
	forestGrass = append(forestGrass, filterInfrastructureFreeCells(byClass["20_shrubland"], false)...)
	wetlandCells := filterInfrastructureFreeCells(byClass["90_herbaceous_wetland"], false)
	mangroveCells := filterInfrastructureFreeCells(byClass["95_mangrove"], true)
	shrublandCells := filterInfrastructureFreeCells(byClass["20_shrubland"], false)
	roadsidePlants := roadsidePlantCells(report)
	coastalPalms := coastalPalmCells(report)
	alligatorCells := alligatorAlleyCells(report)
	dataDrivenSpecies := landcoverSpeciesVegetation(report)
	useDataDrivenSpecies := len(dataDrivenSpecies) > 0
	genericTreeCells := treeCells
	genericWetlandCells := wetlandCells
	genericMangroveCells := mangroveCells
	genericShrublandCells := shrublandCells
	if useDataDrivenSpecies {
		genericTreeCells = nil
		genericWetlandCells = nil
		genericMangroveCells = nil
		genericShrublandCells = nil
	}
	plans := []scatterPlan{
		{name: "pine_flatwoods_resource_forest", cells: genericTreeCells, unitConst: 350, count: minInt(7000, len(genericTreeCells)*2/5), density: report.FloraDensity, centerLimit: 72, spread: 5.5, minDist: 0.45, clusterShape: "packed", terrains: []int{10, 0}, rotationChoices: floatChoices(3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 21, 22, 23, 24, 25, 26)},
		{name: "live_oak_resource_forest", cells: genericTreeCells, unitConst: 2567, count: minInt(6200, len(genericTreeCells)*3/5), density: report.FloraDensity, centerLimit: 64, spread: 5.3, minDist: 0.45, clusterShape: "packed", terrains: []int{10, 0}, rotationChoices: rangeFloatChoices(0, 26)},
		{name: "forest_edge_trees", cells: genericTreeCells, unitConst: 2567, count: minInt(140, maxInt(1, len(genericTreeCells)/160)), density: math.Min(report.FloraDensity, 4), centerLimit: 48, spread: 2.0, minDist: 2.4, terrains: []int{10, 0}, rotationChoices: rangeFloatChoices(0, 26)},
		{name: "sawgrass_reed_wetland_resource", cells: genericWetlandCells, unitConst: 1350, count: minInt(1600, maxInt(1, len(genericWetlandCells)*2)), density: report.FloraDensity, centerLimit: 28, spread: 4.2, minDist: 0.45, clusterShape: "packed", terrains: []int{111, 55, 101, 95}},
		{name: "mangrove_coast_resource", cells: genericMangroveCells, unitConst: 1144, count: minInt(900, maxInt(1, len(genericMangroveCells)*3)), density: report.FloraDensity, centerLimit: 24, spread: 3.8, minDist: 0.45, clusterShape: "packed", terrains: []int{55, 95}},
		{name: "wetland_lilies", cells: append(lakeWater, wetlandCells...), unitConst: 2536, count: minInt(360, maxInt(1, (len(lakeWater)+len(wetlandCells))/14)), density: report.FloraDensity, centerLimit: 24, spread: 3.2, minDist: 0.7, clusterShape: "packed", terrains: []int{1, 23, 22, 111, 95}},
		{name: "shrubland_palmetto", cells: genericShrublandCells, unitConst: 1054, count: minInt(80, maxInt(1, len(genericShrublandCells)*2)), density: report.FloraDensity, centerLimit: 8, spread: 2.6, minDist: 0.55, clusterShape: "packed", terrains: []int{0, 100}},
		{name: "roadside_sparse_palmetto", cells: roadsidePlants, unitConst: 1054, count: minInt(160, maxInt(1, len(roadsidePlants)/16)), density: 3, centerLimit: 36, spread: 2.0, minDist: 1.4, terrains: []int{6}, rotationChoices: rangeFloatChoices(0, 14)},
		{name: "barrier_coast_palms", cells: coastalPalms, unitConst: 351, count: minInt(900, maxInt(1, len(coastalPalms)*4/5)), density: report.FloraDensity, centerLimit: 48, spread: 3.5, minDist: 0.55, clusterShape: "packed", terrains: []int{53}, rotationChoices: rangeFloatChoices(0, 38)},
		{name: "deer_hunt_clusters", cells: forestGrass, unitConst: 65, count: minInt(260, maxInt(4, len(forestGrass)/70)), density: report.LandFaunaDensity, centerLimit: 64, spread: 6.5, minDist: 1.0, terrains: []int{0, 10, 12, 100}},
		{name: "boar_hunt_clusters", cells: forestGrass, unitConst: 48, count: minInt(95, maxInt(2, len(forestGrass)/190)), density: report.LandFaunaDensity, centerLimit: 48, spread: 6.0, minDist: 1.5, terrains: []int{0, 10, 12, 100}},
		{name: "crocodile_wetland_predators", cells: wetlandCells, unitConst: 1031, count: minInt(36, maxInt(1, len(wetlandCells)/90)), density: report.LandFaunaDensity, centerLimit: 24, spread: 7.0, minDist: 3.0, terrains: []int{111, 55, 101, 95}},
		{name: "alligator_alley_meme_cluster", cells: alligatorCells, unitConst: 1031, count: minInt(80, maxInt(12, len(alligatorCells)/5)), density: 10, centerLimit: 12, spread: 4.0, minDist: 1.1, terrains: []int{0, 10, 55, 95, 100, 101, 111}},
		{name: "black_bear_forest_predators", cells: treeCells, unitConst: 2089, count: minInt(28, maxInt(1, len(treeCells)/520)), density: report.LandFaunaDensity, centerLimit: 24, spread: 9.0, minDist: 4.0, terrains: []int{0, 10}},
		{name: "jaguar_swamp_ambiance", cells: wetlandCells, unitConst: 812, count: minInt(20, maxInt(1, len(wetlandCells)/160)), density: report.LandFaunaDensity, centerLimit: 16, spread: 8.0, minDist: 4.0, terrains: []int{111, 55, 101}},
	}
	var out []scenario.UnitRecipe
	out = append(out, dataDrivenSpecies...)
	for i, plan := range plans {
		plan.count = densityScaledCount(plan.count, plan.density)
		if len(plan.cells) == 0 || plan.count <= 0 {
			continue
		}
		centers := representativeCenters(plan.cells, minInt(plan.centerLimit, maxInt(4, plan.count/90)))
		exact := false
		seed := 31000 + i*997
		x1, y1 := 0.0, 0.0
		x2, y2 := float64(report.GridSize), float64(report.GridSize)
		out = append(out, scenario.UnitRecipe{
			Op:                "cluster_scatter",
			Player:            0,
			UnitConst:         plan.unitConst,
			Count:             intPtr(plan.count),
			Centers:           centers,
			ClusterShape:      plan.clusterShape,
			Spread:            float64Ptr(plan.spread),
			MinDistance:       float64Ptr(plan.minDist),
			AllowedTerrainIDs: pruneUnbuildableTerrainIDs(plan.terrains),
			Exact:             &exact,
			Seed:              intPtr(seed),
			RotationChoices:   plan.rotationChoices,
			TargetAreaX1:      &x1,
			TargetAreaY1:      &y1,
			TargetAreaX2:      &x2,
			TargetAreaY2:      &y2,
		})
	}
	out = append(out, waterFaunaUnits(shoreWater, lakeWater, mediumWater, deepWater)...)
	return out
}

func landcoverSpeciesVegetation(report Report) []scenario.UnitRecipe {
	type key struct {
		glc       string
		unitConst int
	}
	byKey := map[key][]Cell{}
	for _, cell := range report.Cells {
		if len(cell.SpeciesUnitConsts) == 0 || cell.LandcoverKeyGLC == "" {
			continue
		}
		if cell.Land == nil || !*cell.Land {
			continue
		}
		if !cellBuildableForGaiaPlacement(cell) {
			continue
		}
		if cell.Road || cell.Roadside || cell.Border || cell.WaterBuffer || cell.KeyWestCauseway {
			continue
		}
		for _, unitConst := range cell.SpeciesUnitConsts {
			if unitConst == 0 {
				continue
			}
			byKey[key{glc: cell.LandcoverKeyGLC, unitConst: unitConst}] = append(byKey[key{glc: cell.LandcoverKeyGLC, unitConst: unitConst}], cell)
		}
	}
	keys := make([]key, 0, len(byKey))
	for k := range byKey {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].glc != keys[j].glc {
			return keys[i].glc < keys[j].glc
		}
		return keys[i].unitConst < keys[j].unitConst
	})
	var out []scenario.UnitRecipe
	for i, k := range keys {
		cells := nonIsolatedLandCells(filterInfrastructureFreeCells(byKey[k], k.unitConst == 1144), report.GridSize, 1)
		if len(cells) == 0 {
			continue
		}
		count := densityScaledCount(minInt(2600, maxInt(1, len(cells)/maxInt(1, len(byKey[k][0].SpeciesUnitConsts)))), report.FloraDensity)
		if count <= 0 {
			continue
		}
		x1, y1 := 0.0, 0.0
		x2, y2 := float64(report.GridSize), float64(report.GridSize)
		exact := false
		out = append(out, scenario.UnitRecipe{
			Op:                "cluster_scatter",
			Player:            0,
			UnitConst:         k.unitConst,
			Count:             intPtr(count),
			Centers:           representativeCenters(cells, minInt(72, maxInt(4, count/90))),
			ClusterShape:      "packed",
			Spread:            float64Ptr(5.4),
			MinDistance:       float64Ptr(0.45),
			AllowedTerrainIDs: pruneUnbuildableTerrainIDs(terrainIDsForLandcoverKey(byKey[k][0].LandcoverKey)),
			Exact:             &exact,
			Seed:              intPtr(62000 + i*733),
			RotationChoices:   rotationChoicesForSpecies(k.unitConst),
			TargetAreaX1:      &x1,
			TargetAreaY1:      &y1,
			TargetAreaX2:      &x2,
			TargetAreaY2:      &y2,
		})
	}
	return out
}

func rotationChoicesForSpecies(unitConst int) []float64 {
	switch unitConst {
	case 349, 350, 351, 413, 1054, 1144, 1347, 1717, 1984, 2027, 2028, 2567, 2580:
		return rangeFloatChoices(0, 26)
	default:
		return nil
	}
}

func waterResourceCells(report Report) (shore []Cell, lake []Cell, medium []Cell, deep []Cell) {
	for _, cell := range report.Cells {
		if cell.Road || cell.Roadside || cell.Border || cell.KeyWestCauseway {
			continue
		}
		if cell.Land != nil && *cell.Land {
			continue
		}
		if cell.InlandWater || cell.LandcoverKey == "80_permanent_water" {
			lake = append(lake, cell)
			continue
		}
		if cell.DistanceToShoreTiles <= 8 {
			shore = append(shore, cell)
			continue
		}
		if cell.DistanceToShoreTiles < 20 {
			medium = append(medium, cell)
			continue
		}
		if cell.DistanceToShoreTiles >= 20 {
			deep = append(deep, cell)
		}
	}
	return shore, lake, medium, deep
}

func nonIsolatedLandCells(cells []Cell, n int, minNeighbors int) []Cell {
	if len(cells) == 0 {
		return nil
	}
	present := map[int]bool{}
	for _, cell := range cells {
		present[cell.Y*n+cell.X] = true
	}
	out := make([]Cell, 0, len(cells))
	for _, cell := range cells {
		neighbors := 0
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				if dx == 0 && dy == 0 {
					continue
				}
				x := cell.X + dx
				y := cell.Y + dy
				if x < 0 || y < 0 || x >= n || y >= n {
					continue
				}
				if present[y*n+x] {
					neighbors++
				}
			}
		}
		if neighbors >= minNeighbors {
			out = append(out, cell)
		}
	}
	return out
}

func floatChoices(values ...int) []float64 {
	out := make([]float64, 0, len(values))
	for _, value := range values {
		out = append(out, float64(value))
	}
	return out
}

func rangeFloatChoices(first, last int) []float64 {
	out := make([]float64, 0, last-first+1)
	for i := first; i <= last; i++ {
		out = append(out, float64(i))
	}
	return out
}

func representativeCenters(cells []Cell, limit int) []scenario.PointRecipe {
	if len(cells) == 0 || limit <= 0 {
		return nil
	}
	step := len(cells) / limit
	if step < 1 {
		step = 1
	}
	centers := make([]scenario.PointRecipe, 0, limit)
	for i := step / 2; i < len(cells) && len(centers) < limit; i += step {
		centers = append(centers, scenario.PointRecipe{
			X: float64(cells[i].X) + 0.5,
			Y: float64(cells[i].Y) + 0.5,
		})
	}
	if len(centers) == 0 {
		cell := cells[len(cells)/2]
		centers = append(centers, scenario.PointRecipe{X: float64(cell.X) + 0.5, Y: float64(cell.Y) + 0.5})
	}
	return centers
}

func terrainRunsFromCells(cells []Cell) []scenario.MapRecipe {
	if len(cells) == 0 {
		return nil
	}
	rows := map[int][]Cell{}
	for _, cell := range cells {
		rows[cell.Y] = append(rows[cell.Y], cell)
	}
	ys := make([]int, 0, len(rows))
	for y := range rows {
		ys = append(ys, y)
	}
	sort.Ints(ys)
	var out []scenario.MapRecipe
	for _, y := range ys {
		row := rows[y]
		sort.Slice(row, func(i, j int) bool { return row[i].X < row[j].X })
		start := row[0]
		prev := row[0]
		for _, cell := range row[1:] {
			if cell.X == prev.X+1 && cell.TerrainID == prev.TerrainID {
				prev = cell
				continue
			}
			out = append(out, scenario.MapRecipe{
				Op:        "set_terrain_rect",
				X1:        start.X,
				Y1:        y,
				X2:        prev.X,
				Y2:        y,
				TerrainID: intPtr(start.TerrainID),
			})
			start = cell
			prev = cell
		}
		out = append(out, scenario.MapRecipe{
			Op:        "set_terrain_rect",
			X1:        start.X,
			Y1:        y,
			X2:        prev.X,
			Y2:        y,
			TerrainID: intPtr(start.TerrainID),
		})
	}
	return out
}

func intPtr(v int) *int {
	return &v
}

func float64Ptr(v float64) *float64 {
	return &v
}
