package geotrace

import (
	"aoe2kit/pkg/fetch"
	"context"
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
	Seed                 int
	BBox                 BBox
	GridSize             int
	Provider             string
	LandmaskProvider     string
	LandmaskPath         string
	InlandWaterProvider  string
	InlandWaterPath      string
	LandcoverProvider    string
	LandcoverCacheDir    string
	LandcoverPath        string
	LandcoverDir         string
	ClimateZone          string
	FeltMappingPath      string
	ProfilePath          string
	CompendiumPath       string
	USGSWoodyPath        string
	DatPath              string
	ProjectionAspect     float64
	ProjectionNorthScale float64
	Orient               string
	TargetWaterFraction  float64
	SeamWaterColumns     int
	FloraDensity         float64
	LandFaunaDensity     float64
	FishDensity          float64
	ShoreFishDensity     float64
	OceanFishDensity     float64
	ShoreBandWidth       float64
	WaterBandWidth       float64
	RoadsProvider        string
	RoadsPath            string
	RoadsCacheDir        string
	RoadsTimeout         time.Duration
	Progress             io.Writer
	roadContext          context.Context
	roadFetchState       *roadFetchState
	RoadSimplifyTiles    float64
	RoadWidthTiles       float64
	AdminBorders         bool
	Client               *http.Client
}

type Report struct {
	Experimental                     bool                         `json:"experimental"`
	Attribution                      map[string]SourceAttribution `json:"attribution,omitempty"`
	PlayerStarts                     *PlayerStartReport           `json:"player_starts,omitempty"`
	PlacementSweep                   *PlacementAudit              `json:"placement_sweep,omitempty"`
	PlacementGate                    *PlacementAudit              `json:"placement_gate,omitempty"`
	NeutralResources                 []NeutralResourceReport      `json:"neutral_resources,omitempty"`
	WrittenPlacementAudit            *PlacementAudit              `json:"written_placement_audit,omitempty"`
	PlacementSweepRecipe             *scenario.Recipe             `json:"placement_sweep_recipe,omitempty"`
	CoastRefinement                  CoastRefinement              `json:"coast_refinement"`
	Seed                             int                          `json:"seed"`
	EcologyLayers                    []LayerDiagnostics           `json:"ecology_layers,omitempty"`
	Diagnostics                      *MapDiagnostics              `json:"diagnostics,omitempty"`
	SourceCoverage                   map[string]SourceCoverage    `json:"source_coverage"`
	BBox                             BBox                         `json:"bbox"`
	OverlayBBox                      BBox                         `json:"overlay_bbox,omitempty"`
	GridSize                         int                          `json:"grid_size"`
	Provider                         string                       `json:"provider"`
	LandmaskProvider                 string                       `json:"landmask_provider,omitempty"`
	InlandWaterProvider              string                       `json:"inland_water_provider,omitempty"`
	LandcoverProvider                string                       `json:"landcover_provider,omitempty"`
	FeltMappingSchema                string                       `json:"felt_mapping_schema,omitempty"`
	EcologyProfile                   EcologyProfile               `json:"ecology_profile,omitempty"`
	USGSWoodyProvider                string                       `json:"usgs_woody_provider,omitempty"`
	USGSWoodyPointCount              int                          `json:"usgs_woody_point_count,omitempty"`
	USGSWoodySpeciesCount            int                          `json:"usgs_woody_species_count,omitempty"`
	USGSWoodySHA256                  string                       `json:"usgs_woody_sha256,omitempty"`
	USGSWoodyMaxDistanceKM           float64                      `json:"usgs_woody_max_distance_km,omitempty"`
	DatPath                          string                       `json:"dat_path,omitempty"`
	ClimateZone                      string                       `json:"climate_zone,omitempty"`
	Orient                           string                       `json:"orient"`
	ApproxTileMeters                 float64                      `json:"approx_tile_meters"`
	CoastlineConfidence              string                       `json:"coastline_confidence"`
	CoastlineReason                  string                       `json:"coastline_reason"`
	StandingLadder                   []string                     `json:"standing_ladder"`
	Warnings                         []string                     `json:"warnings,omitempty"`
	Cells                            []Cell                       `json:"cells"`
	TerrainCounts                    map[int]int                  `json:"terrain_counts"`
	LandcoverCounts                  map[string]int               `json:"landcover_counts,omitempty"`
	LandcoverMissingTiles            map[string]int               `json:"landcover_missing_tiles,omitempty"`
	LandcoverGLCCounts               map[string]int               `json:"landcover_glc_counts,omitempty"`
	SpeciesResolutions               []SpeciesResolution          `json:"species_resolutions,omitempty"`
	LandCells                        int                          `json:"land_cells"`
	WaterCells                       int                          `json:"water_cells"`
	LandFraction                     float64                      `json:"land_fraction"`
	WaterFraction                    float64                      `json:"water_fraction"`
	TargetWaterFraction              float64                      `json:"target_water_fraction,omitempty"`
	LandEmphasisApplied              bool                         `json:"land_emphasis_applied,omitempty"`
	FloraDensity                     float64                      `json:"flora_density,omitempty"`
	LandFaunaDensity                 float64                      `json:"land_fauna_density,omitempty"`
	FishDensity                      float64                      `json:"fish_density,omitempty"`
	ShoreFishDensity                 float64                      `json:"shore_fish_density,omitempty"`
	OceanFishDensity                 float64                      `json:"ocean_fish_density,omitempty"`
	ShoreBandWidth                   float64                      `json:"shore_band_width,omitempty"`
	WaterBandWidth                   float64                      `json:"water_band_width,omitempty"`
	ShoreCrossfadeCells              int                          `json:"shore_crossfade_cells,omitempty"`
	ShoreCrossfadeCounts             map[string]int               `json:"shore_crossfade_counts,omitempty"`
	ShoreTerrainCounts               map[int]int                  `json:"shore_terrain_counts,omitempty"`
	DockableShoreCells               int                          `json:"dockable_shore_cells,omitempty"`
	UnbuildableTerrainIDs            []int                        `json:"unbuildable_terrain_ids,omitempty"`
	UnbuildableCells                 int                          `json:"unbuildable_cells,omitempty"`
	PlacementGuard                   *PlacementGuard              `json:"placement_guard,omitempty"`
	RoadsProvider                    string                       `json:"roads_provider,omitempty"`
	RoadWayCount                     int                          `json:"road_way_count,omitempty"`
	RoadVertexCountRaw               int                          `json:"road_vertex_count_raw,omitempty"`
	RoadVertexCount                  int                          `json:"road_vertex_count,omitempty"`
	RoadSimplifyTiles                float64                      `json:"road_simplify_tiles,omitempty"`
	RoadWidthTiles                   float64                      `json:"road_width_tiles,omitempty"`
	RoadCells                        int                          `json:"road_cells,omitempty"`
	PopulationCenterCells            int                          `json:"population_center_cells,omitempty"`
	PopulationCenterBuiltUpCells     int                          `json:"population_center_built_up_cells,omitempty"`
	PopulationCenterRoadDensityCells int                          `json:"population_center_road_density_cells,omitempty"`
	RoadsideCells                    int                          `json:"roadside_cells,omitempty"`
	WaterBufferCells                 int                          `json:"water_buffer_cells,omitempty"`
	CausewayCells                    int                          `json:"causeway_cells,omitempty"`
	BorderCells                      int                          `json:"border_cells,omitempty"`
	BorderWayCount                   int                          `json:"border_way_count,omitempty"`
	RegionDiagnostics                []RegionDiagnostics          `json:"region_diagnostics,omitempty"`
	RoadRefs                         []string                     `json:"road_refs,omitempty"`
	LandExtentXTiles                 int                          `json:"land_extent_x_tiles,omitempty"`
	LandExtentYTiles                 int                          `json:"land_extent_y_tiles,omitempty"`
	LandExtentRatio                  float64                      `json:"land_extent_y_to_x_ratio,omitempty"`
	LandExtentWHRatio                float64                      `json:"land_extent_w_to_h_ratio,omitempty"`
	ScreenWHRatio                    float64                      `json:"estimated_screen_w_to_h_ratio,omitempty"`
	ViewportTileWHRatio              float64                      `json:"viewport_tile_w_to_h_ratio,omitempty"`
	ViewportScreenWHRatio            float64                      `json:"viewport_screen_w_to_h_ratio,omitempty"`
	CalibrationAxes                  []AxisDiagnostics            `json:"calibration_axes,omitempty"`
	ProjectionSelfCheck              *ProjectionSelfCheck         `json:"projection_self_check,omitempty"`
	ProjectionAspect                 float64                      `json:"projection_aspect,omitempty"`
	ProjectionNorthScale             float64                      `json:"projection_north_scale,omitempty"`
	MinElevation                     float64                      `json:"min_elevation"`
	MaxElevation                     float64                      `json:"max_elevation"`
	MeanElevation                    float64                      `json:"mean_elevation"`
	Recipe                           *scenario.Recipe             `json:"recipe,omitempty"`
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
	CoastalLandFraction  float64         `json:"coastal_land_fraction,omitempty"`
	SourceLandcoverClass int             `json:"source_landcover_class,omitempty"`
	X                    int             `json:"x"`
	Y                    int             `json:"y"`
	Lat                  float64         `json:"lat"`
	Lon                  float64         `json:"lon"`
	OutsideTrace         bool            `json:"outside_trace,omitempty"`
	Elevation            float64         `json:"elevation_m"`
	Land                 *bool           `json:"land,omitempty"`
	InlandWater          bool            `json:"inland_water,omitempty"`
	ShelteredWater       bool            `json:"sheltered_water,omitempty"`
	WaterbodyName        string          `json:"waterbody_name,omitempty"`
	LandcoverClass       int             `json:"landcover_class,omitempty"`
	LandcoverKey         string          `json:"landcover_key,omitempty"`
	LandcoverClassGLC    int             `json:"landcover_class_glc,omitempty"`
	LandcoverKeyGLC      string          `json:"landcover_key_glc,omitempty"`
	LandcoverFraction    float64         `json:"landcover_fraction,omitempty"`
	LandcoverHistogram   []LandcoverStat `json:"landcover_histogram,omitempty"`
	USGSWoodySpecies     []string        `json:"usgs_woody_species,omitempty"`
	SpeciesUnitConsts    []int           `json:"species_unit_consts,omitempty"`
	DistanceToShoreTiles float64         `json:"distance_to_shore_tiles,omitempty"`
	ClimateZone          string          `json:"climate_zone,omitempty"`
	ElevationBand        string          `json:"elevation_band,omitempty"`
	Standing             string          `json:"standing"`
	TerrainID            int             `json:"terrain_id"`
	Road                 bool            `json:"road,omitempty"`
	RoadTerrainID        int             `json:"road_terrain_id,omitempty"`
	RoadWidth            float64         `json:"road_width_tiles,omitempty"`
	Roadside             bool            `json:"roadside,omitempty"`
	PopulationCenter     bool            `json:"population_center,omitempty"`
	BuiltUpCore          bool            `json:"built_up_core,omitempty"`
	RoadCorridor         bool            `json:"road_corridor,omitempty"`
	WaterBuffer          bool            `json:"water_buffer,omitempty"`
	Causeway             bool            `json:"causeway,omitempty"`
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
	MissingTile         string
	CoastalLandFraction float64
	PixelMeters         float64
	Valid               bool
	Class               int
	Key                 string
	GLCClass            int
	GLCKey              string
	Fraction            float64
	Histogram           []LandcoverStat
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
	for _, value := range []float64{opts.BBox.MinLat, opts.BBox.MaxLat, opts.BBox.MinLon, opts.BBox.MaxLon, opts.ProjectionAspect, opts.ProjectionNorthScale, opts.RoadWidthTiles} {
		if !finite(value) {
			return Report{}, fmt.Errorf("geographic bounds, projection settings, and road width must be finite")
		}
	}
	if opts.RoadWidthTiles < 0 || opts.RoadWidthTiles > 32 {
		return Report{}, fmt.Errorf("road width must be between 0 and 32 tiles")
	}
	if opts.BBox.MinLat < -90 || opts.BBox.MaxLat > 90 || opts.BBox.MinLon < -180 || opts.BBox.MaxLon > 180 {
		return Report{}, fmt.Errorf("bounds must use latitude -90..90 and longitude -180..180")
	}
	if opts.ProjectionAspect < 0 || opts.ProjectionNorthScale < 0 {
		return Report{}, fmt.Errorf("projection settings must be positive, or zero for defaults")
	}
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
	profile, err := loadEcologyProfile(opts.ProfilePath)
	if err != nil {
		return Report{}, err
	}
	ensurePopulationCenterLayer(&profile)
	if err := resolveFeltLayers(&profile, opts.CompendiumPath); err != nil {
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
	proj := newGeoProjection(opts.BBox, orient, ProjectionOptions{Aspect: opts.ProjectionAspect, NorthScale: opts.ProjectionNorthScale})
	points := gridPoints(proj, opts.GridSize)
	overlayBBox := projectionEnvelope(proj)
	if !finite(overlayBBox.MinLat) || !finite(overlayBBox.MaxLat) || !finite(overlayBBox.MinLon) || !finite(overlayBBox.MaxLon) || overlayBBox.MinLat < -90 || overlayBBox.MaxLat > 90 || overlayBBox.MinLon < -180 || overlayBBox.MaxLon > 180 {
		return Report{}, fmt.Errorf("projected map envelope exceeds geographic coordinates; reduce span or change orientation")
	}
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
	originalLandmask := append([]bool(nil), landmask...)
	coastRefinement := refineCoastMask(landmask, landcover, landcoverProvider, approximateTileMeters(opts.BBox, opts.GridSize))
	usgsWoody, usgsSummary, err := loadUSGSWoodyWithProfile(opts.USGSWoodyPath, profile)
	if err != nil {
		return Report{}, err
	}
	usgsWoodyProvider := ""
	if opts.USGSWoodyPath != "" {
		usgsWoodyProvider = "usgs-woody-csv"
	}
	report := Report{
		Experimental:          true,
		CoastRefinement:       coastRefinement,
		Seed:                  opts.Seed,
		BBox:                  opts.BBox,
		OverlayBBox:           overlayBBox,
		GridSize:              opts.GridSize,
		Provider:              provider,
		LandmaskProvider:      landmaskProvider,
		InlandWaterProvider:   inlandWaterProvider,
		LandcoverProvider:     landcoverProvider,
		FeltMappingSchema:     mapping.Schema,
		EcologyProfile:        profile,
		USGSWoodyProvider:     usgsWoodyProvider,
		USGSWoodyPointCount:   usgsSummary.PointCount,
		USGSWoodySpeciesCount: usgsSummary.SpeciesCount,
		USGSWoodySHA256:       usgsSummary.SHA256,
		DatPath:               opts.DatPath,
		ClimateZone:           climateZoneForLat((opts.BBox.MinLat+opts.BBox.MaxLat)/2, opts.ClimateZone),
		Orient:                orient,
		ApproxTileMeters:      approximateTileMeters(opts.BBox, opts.GridSize),
		CoastlineConfidence:   "approximate",
		CoastlineReason:       "DEM-only sampling cannot distinguish ocean 0.0m from low coastal land; provide landmask/bathymetry/coastline vector to upgrade this claim.",
		StandingLadder:        []string{"from_data", "inferred", "default"},
		TerrainCounts:         map[int]int{},
		SpeciesResolutions:    mapping.SpeciesResolutions,
		FloraDensity:          normalizeDensity(opts.FloraDensity),
		LandFaunaDensity:      normalizeDensity(opts.LandFaunaDensity),
		FishDensity:           normalizeDensity(opts.FishDensity),
		ShoreFishDensity:      normalizeDensity(firstPositive(opts.ShoreFishDensity, opts.FishDensity)),
		OceanFishDensity:      normalizeDensity(firstPositive(opts.OceanFishDensity, opts.FishDensity)),
		ShoreBandWidth:        normalizeShoreBandWidth(opts.ShoreBandWidth),
		WaterBandWidth:        normalizeWaterBandWidth(opts.WaterBandWidth),
		ProjectionAspect:      normalizeProjectionAspect(opts.ProjectionAspect),
		ProjectionNorthScale:  normalizeProjectionNorthScale(opts.ProjectionNorthScale),
	}
	if profile.EnableUnvalidatedUrban {
		report.Warnings = append(report.Warnings, "Urban buildings and accents explicitly enabled: engine behavior is unvalidated")
	}
	if landcoverProvider == "worldcover" {
		report.Attribution = map[string]SourceAttribution{"landcover": worldCoverAttribution(overlayBBox)}
	}
	if usgsWoody != nil {
		report.USGSWoodyMaxDistanceKM = usgsWoodyMaxDistanceKM
	}
	if orient == "iso-ccw-screen" {
		report.ViewportTileWHRatio = viewportTileWHRatio(proj)
		report.ViewportScreenWHRatio = report.ViewportTileWHRatio * screenRhombusWidthToHeight
		report.ProjectionSelfCheck = projectionSelfCheck(proj)
		measureCalibrationAxes(&report)
	}
	if landcover != nil {
		report.LandcoverCounts = map[string]int{}
		report.LandcoverGLCCounts = map[string]int{}
		for _, sample := range landcover {
			if sample.MissingTile != "" {
				if report.LandcoverMissingTiles == nil {
					report.LandcoverMissingTiles = map[string]int{}
				}
				report.LandcoverMissingTiles[sample.MissingTile]++
			}
		}
		if len(report.LandcoverMissingTiles) > 0 {
			report.Warnings = append(report.Warnings, "WorldCover 404 coverage gaps retain the source landmask; absence is not independent water evidence")
		}
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
		if coastRefinement.Enabled {
			report.CoastlineReason = "source landmask refined by resolution-qualified WorldCover classes; see coast_refinement for policy and directional counts"
		}
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
		if i < len(originalLandmask) && i < len(landcover) && worldCoverInlandWater(originalLandmask[i], landcover[i]) {
			inWater = true
		}
		waterName := ""
		if inWater && i < len(inlandWaterNames) {
			waterName = inlandWaterNames[i]
		}
		var lc *LandcoverSample
		if landcover != nil {
			lc = &landcover[i]
		}
		cell := classifyCell(point, elev, land, shoreDistance[i], climateZone, mapping, inWater, waterName, lc)
		if usgsWoody != nil {
			cell.USGSWoodySpecies = usgsWoody.nearestSpecies(point.Lat, point.Lon)
		}
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
	normalizeDockableShore(&report)
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
		report.RoadWidthTiles = roadStats.WidthTiles
		report.RoadCells = roadStats.CellCount
		report.RoadRefs = roadStats.Refs
		if report.SourceCoverage == nil {
			report.SourceCoverage = map[string]SourceCoverage{}
		}
		report.SourceCoverage["roads"] = roadStats.Coverage
		if a, ok := osmRoadAttribution(opts, roadStats.Coverage); ok {
			if report.Attribution == nil {
				report.Attribution = map[string]SourceAttribution{}
			}
			report.Attribution["roads"] = a
		}
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
	markPopulationCenters(&report)
	markShoreCrossfadeCells(&report)
	classifyShelteredWater(&report)
	measureProfileRegions(&report)
	measureLandExtents(&report)
	constraints, err := loadDockConstraints(opts.DatPath, profile)
	if err != nil {
		return Report{}, err
	}
	report.Diagnostics = measureMapDiagnostics(report, constraints)
	measureSourceCoverage(&report)
	if landmask != nil {
		recipe := buildRecipe(report, &report.EcologyLayers)
		report.Recipe = &recipe
		if err := applyPlayerStarts(&report); err != nil {
			return Report{}, err
		}
		if err := applyNeutralResources(&report); err != nil {
			return Report{}, err
		}
	}
	return report, nil
}

// normalizeDockableShore makes the shoreline contract explicit. Terrain 1 is
// the shallow-water terrain used by dock placement; only a real landmask can
// justify applying this override, so DEM-only traces remain approximate.
func normalizeDockableShore(report *Report) {
	if report == nil || report.LandmaskProvider == "" {
		return
	}
	width := normalizeShoreBandWidth(report.ShoreBandWidth)
	report.DockableShoreCells = 0
	report.TerrainCounts = map[int]int{}
	for i := range report.Cells {
		cell := &report.Cells[i]
		if cell.Land != nil && *cell.Land {
			report.TerrainCounts[cell.TerrainID]++
			continue
		}
		if cell.InlandWater || cell.DistanceToShoreTiles > width {
			report.TerrainCounts[cell.TerrainID]++
			continue
		}
		cell.TerrainID = 1
		cell.TerrainConfidence = "inferred"
		cell.Reason = "landmask coastal water; shallow-water candidate for dock placement, engine confirmation pending"
		report.DockableShoreCells++
		report.TerrainCounts[cell.TerrainID]++
	}
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

func gridPoints(proj geoProjection, n int) []Cell {
	b, orient := proj.b, proj.orient
	out := make([]Cell, 0, n*n)
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

type geoProjection struct {
	b            BBox
	orient       string
	midLat       float64
	midLon       float64
	kmPerLat     float64
	kmPerLon     float64
	rotHalfKm    float64
	northScale   float64
	tileScaleU   float64
	matrixScale  float64
	halfEastKm   float64
	halfNorthKm  float64
	screenAspect float64
}

const (
	screenRhombusWidthToHeight = 2.0
)

type ProjectionOptions struct {
	Aspect     float64 `json:"aspect,omitempty"`
	NorthScale float64 `json:"north_scale,omitempty"`
}

func (r Report) projection() geoProjection {
	return newGeoProjection(r.BBox, r.Orient, ProjectionOptions{Aspect: r.ProjectionAspect, NorthScale: r.ProjectionNorthScale})
}

func newGeoProjection(b BBox, orient string, options ProjectionOptions) geoProjection {
	midLat := (b.MinLat + b.MaxLat) / 2
	kmPerLat := 111.320
	kmPerLon := 111.320 * math.Cos(midLat*math.Pi/180)
	if kmPerLon <= 0 {
		kmPerLon = 1
	}
	halfEast := (b.MaxLon - b.MinLon) * kmPerLon / 2
	halfNorth := (b.MaxLat - b.MinLat) * kmPerLat / 2
	aspect := normalizeProjectionAspect(options.Aspect)
	northScale := normalizeProjectionNorthScale(options.NorthScale)
	tileScaleU := 1.0
	scaledHalfNorth := halfNorth * northScale
	rotHalf := (halfEast + scaledHalfNorth) / math.Sqrt2
	if rotHalf <= 0 {
		rotHalf = 1
	}
	matrixScale := 0.0
	if orient == "iso-ccw-screen" {
		denom := scaledHalfNorth + (1/aspect)*halfEast
		if denom <= 0 {
			denom = 1
		}
		matrixScale = 0.5 / denom
	}
	return geoProjection{
		b:            b,
		orient:       orient,
		midLat:       midLat,
		midLon:       (b.MinLon + b.MaxLon) / 2,
		kmPerLat:     kmPerLat,
		kmPerLon:     kmPerLon,
		rotHalfKm:    rotHalf,
		northScale:   northScale,
		tileScaleU:   tileScaleU,
		matrixScale:  matrixScale,
		halfEastKm:   halfEast,
		halfNorthKm:  halfNorth,
		screenAspect: aspect,
	}
}

func normalizeProjectionAspect(value float64) float64 {
	if value <= 0 {
		return screenRhombusWidthToHeight
	}
	if value < 0.5 {
		return 0.5
	}
	if value > 8 {
		return 8
	}
	return value
}

func normalizeProjectionNorthScale(value float64) float64 {
	if value <= 0 {
		return 1
	}
	if value < 0.25 {
		return 0.25
	}
	if value > 4 {
		return 4
	}
	return value
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
		eastKm := (u + v) / (2 * (1 / p.screenAspect) * scale)
		northKm := (u - v) / (2 * scale * p.northScale)
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
		return u, v, withinProjectionFrame(u, v)
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
		eastScale := 1 / p.screenAspect
		u = scale * (eastScale*eastKm + northKm)
		v = scale * (eastScale*eastKm - northKm)
		return u, v, withinProjectionFrame(u, v)
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
	return u, v, withinProjectionFrame(u, v)
}

func withinProjectionFrame(u, v float64) bool {
	// Corner arithmetic can round a fitted half-tile bound a few ulps outward.
	return math.Abs(u) <= 0.5+1e-12 && math.Abs(v) <= 0.5+1e-12
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
	eastKm := math.Min(100, p.halfEastKm*0.5)
	northKm := math.Min(100, p.halfNorthKm*0.5)
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
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	session := fetch.Session{Client: client}
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
		var payload struct {
			Status  string `json:"status"`
			Results []struct {
				Elevation *float64 `json:"elevation"`
			} `json:"results"`
			Error string `json:"error"`
		}
		_, err := session.Get(ctx, fetch.Part{Name: fmt.Sprintf("DEM part %d", start/batchSize+1), Endpoints: []string{u}, Validate: func(data []byte) error {
			if err := json.Unmarshal(data, &payload); err != nil {
				return err
			}
			if payload.Status != "OK" {
				return fmt.Errorf("OpenTopoData status %s: %s", payload.Status, payload.Error)
			}
			if len(payload.Results) != end-start {
				return fmt.Errorf("incomplete DEM results")
			}
			return nil
		}})
		if err != nil {
			return nil, err
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
		if landcover.CoastalLandFraction > 0 && !inlandWater {
			point.CoastalLandFraction = landcover.CoastalLandFraction
			point.SourceLandcoverClass = landcover.Class
			copy := *landcover
			copy.Histogram = nil
			for _, stat := range landcover.Histogram {
				if stat.Class == 80 || !knownWorldCoverClass(stat.Class) {
					continue
				}
				stat.Weight /= landcover.CoastalLandFraction
				copy.Histogram = append(copy.Histogram, stat)
			}
			if len(copy.Histogram) > 0 {
				copy.Class, copy.Key, copy.Fraction = copy.Histogram[0].Class, copy.Histogram[0].Key, copy.Histogram[0].Weight
				landcover = &copy
			}
		}
		point.LandcoverClass = landcover.Class
		point.LandcoverKey = landcover.Key
		point.LandcoverClassGLC = landcover.GLCClass
		point.LandcoverKeyGLC = landcover.GLCKey
		point.LandcoverFraction = landcover.Fraction
		point.LandcoverHistogram = landcover.Histogram
		point.SpeciesUnitConsts = mapping.speciesUnitConsts(landcover.GLCKey, climateZone)
		if landcover.Class == 80 || landcover.Key == "80_permanent_water" {
			// Permanent water includes the ocean. Only water on the source land
			// mask (or an explicit inland polygon) supplies inland-water evidence.
			point.InlandWater = inlandWater || (land != nil && *land)
			point.ElevationBand = "water"
			water := false
			point.Land = &water
		} else if !inlandWater && (land == nil || *land) {
			point.ElevationBand = "landcover_" + landcover.Key
		}
		choice := mapping.landcoverChoice(landcover.Key)
		if len(choice.Proportions) > 0 && !point.InlandWater && (point.Land == nil || *point.Land) {
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
	if point.InlandWater {
		water := false
		point.Land = &water
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
	if point.Land != nil {
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
	polygons = polygonsForPoints(polygons, points)
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
	polygons = polygonsForPoints(polygons, points)
	mask, names := classifyPointsByNamedPolygons(points, polygons)
	return mask, names, provider, nil
}

func polygonsForPoints(polygons []polygon, points []Cell) []polygon {
	if len(polygons) == 0 || len(points) == 0 {
		return nil
	}
	minLon, minLat := math.MaxFloat64, math.MaxFloat64
	maxLon, maxLat := -math.MaxFloat64, -math.MaxFloat64
	for _, cell := range points {
		minLon = math.Min(minLon, cell.Lon)
		maxLon = math.Max(maxLon, cell.Lon)
		minLat = math.Min(minLat, cell.Lat)
		maxLat = math.Max(maxLat, cell.Lat)
	}
	filtered := make([]polygon, 0, len(polygons))
	for _, poly := range polygons {
		if poly.MaxLon < minLon || poly.MinLon > maxLon || poly.MaxLat < minLat || poly.MinLat > maxLat {
			continue
		}
		filtered = append(filtered, poly)
	}
	return filtered
}

func fetchNaturalEarthLand(client *http.Client) ([]polygon, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	session := fetch.Session{Client: client}
	data, err := session.Get(ctx, fetch.Part{Name: "Natural Earth Land", Endpoints: []string{"https://raw.githubusercontent.com/nvkelso/natural-earth-vector/master/geojson/ne_10m_land.geojson"}, MaxBytes: 32 << 20, Validate: func(data []byte) error { _, err := parseGeoJSONPolygons(data); return err }})
	if err != nil {
		return nil, err
	}
	return parseGeoJSONPolygons(data)
}

func fetchNaturalEarthLakes(client *http.Client) ([]polygon, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	session := fetch.Session{Client: client}
	data, err := session.Get(ctx, fetch.Part{Name: "Natural Earth Lakes", Endpoints: []string{"https://raw.githubusercontent.com/nvkelso/natural-earth-vector/master/geojson/ne_10m_lakes.geojson"}, MaxBytes: 32 << 20, Validate: func(data []byte) error { _, err := parseGeoJSONPolygons(data); return err }})
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
	return buildRecipe(report, nil)
}

func buildRecipe(report Report, diagnostics *[]LayerDiagnostics) scenario.Recipe {
	profile := report.EcologyProfile
	if profile.Schema == "" {
		profile = defaultEcologyProfile()
	}
	land := make([]scenario.MaskCellRecipe, 0, len(report.Cells))
	water := make([]scenario.MaskCellRecipe, 0, len(report.Cells))
	beach := make([]scenario.MaskCellRecipe, 0, len(report.Cells))
	lowland := make([]scenario.MaskCellRecipe, 0, len(report.Cells))
	upland := make([]scenario.MaskCellRecipe, 0, len(report.Cells))
	highland := make([]scenario.MaskCellRecipe, 0, len(report.Cells))
	terrainCells := map[int][]scenario.MaskCellRecipe{}
	landcoverCells := map[string][]Cell{}
	roadCellsByTerrain := map[int][]scenario.MaskCellRecipe{}
	var roadsideCells []scenario.MaskCellRecipe
	var borderCells []scenario.MaskCellRecipe
	borderCellsByTerrain := map[int][]scenario.MaskCellRecipe{}
	shoreBandCellsByTerrain := map[int][]scenario.MaskCellRecipe{}
	var shoreMediumCells []scenario.MaskCellRecipe
	var shoreDeepCells []scenario.MaskCellRecipe
	var dockableShoreCells []scenario.MaskCellRecipe
	shoreLayerCells := map[[2]int][]scenario.MaskCellRecipe{}
	for _, cell := range report.Cells {
		maskCell := scenario.MaskCellRecipe{X: cell.X, Y: cell.Y}
		shoreTerrainID := cell.ShoreTerrainID
		if shoreTerrainID == 0 {
			shoreTerrainID = 2
		}
		terrainCells[cell.TerrainID] = append(terrainCells[cell.TerrainID], maskCell)
		if cell.Road {
			terrainID := cell.RoadTerrainID
			if terrainID == 0 {
				terrainID = roadTerrainID
			}
			roadCellsByTerrain[terrainID] = append(roadCellsByTerrain[terrainID], maskCell)
		}
		if cell.Roadside {
			roadsideCells = append(roadsideCells, maskCell)
		}
		if cell.Border {
			borderCells = append(borderCells, maskCell)
			borderCellsByTerrain[cell.TerrainID] = append(borderCellsByTerrain[cell.TerrainID], maskCell)
		}
		if cell.TerrainID == 1 && cell.Land != nil && !*cell.Land &&
			!cell.InlandWater &&
			cell.DistanceToShoreTiles <= normalizeShoreBandWidth(report.ShoreBandWidth) {
			dockableShoreCells = append(dockableShoreCells, maskCell)
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
		if cell.LandcoverKey != "" && ((cell.Land != nil && *cell.Land) || (cell.InlandWater && cell.LandcoverKey == "80_permanent_water")) {
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
		DatPath:  report.DatPath,
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
		if override := profile.TerrainByLandcover[key]; len(override) > 0 {
			terrainIDs = override
			if len(override) == 1 {
				recipe.Map = append(recipe.Map, scenario.MapRecipe{Op: "set_terrain_mask", Mask: "landcover_" + key, TerrainID: intPtr(override[0])})
			}
		}
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
			seed := report.Seed + 24000 + len(recipe.Map)*17
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
	appendNonEmptyMask("dockable_shore", dockableShoreCells)
	if len(dockableShoreCells) > 0 {
		recipe.Map = append(recipe.Map, scenario.MapRecipe{
			Op:        "set_terrain_mask",
			Mask:      "dockable_shore",
			TerrainID: intPtr(1),
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
	roadTerrainIDs := make([]int, 0, len(roadCellsByTerrain))
	for terrainID := range roadCellsByTerrain {
		roadTerrainIDs = append(roadTerrainIDs, terrainID)
	}
	sort.Ints(roadTerrainIDs)
	for _, terrainID := range roadTerrainIDs {
		name := fmt.Sprintf("roads_%d", terrainID)
		appendNonEmptyMask(name, roadCellsByTerrain[terrainID])
		recipe.Map = append(recipe.Map, scenario.MapRecipe{
			Op:        "set_terrain_mask",
			Mask:      name,
			TerrainID: intPtr(terrainID),
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
	units, layers := buildHabitatEcology(report, profile)
	recipe.Units = append(recipe.Units, units...)
	recipe.Units = append(recipe.Units, buildLandmarks(report, profile)...)
	if diagnostics != nil {
		*diagnostics = layers
	}
	if profile.PlayerStarts != nil && profile.PlayerStarts.Mode == "calibration" {
		recipe.Units = append(recipe.Units, projectionCalibrationMarkers(report)...)
	}
	return recipe
}

func projectionCalibrationMarkers(report Report) []scenario.UnitRecipe {
	if report.GridSize <= 0 || report.Orient != "iso-ccw-screen" {
		return nil
	}
	proj := report.projection()
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
		x, y, ok := proj.gridCellForLatLon(report.GridSize, marker.lat, marker.lon)
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

func rotationAngles(count int) []float64 {
	out := make([]float64, count)
	for i := range out {
		out[i] = float64(i) * 2 * math.Pi / float64(count)
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
