package geotrace

import (
	"math"
	"os"
	"path/filepath"
	"testing"
)

func absFloat(v float64) float64 {
	return math.Abs(v)
}

func TestTraceDEMFlatProvider(t *testing.T) {
	report, err := TraceDEM(Options{
		BBox:     BBox{MinLat: 25, MinLon: -81, MaxLat: 26, MaxLon: -80},
		GridSize: 4,
		Provider: "flat",
		Orient:   "grid",
	})
	if err != nil {
		t.Fatalf("TraceDEM: %v", err)
	}
	if len(report.Cells) != 16 {
		t.Fatalf("cells=%d want 16", len(report.Cells))
	}
	if report.CoastlineConfidence != "approximate" {
		t.Fatalf("coastline confidence=%q", report.CoastlineConfidence)
	}
	if report.TerrainCounts[58] != 16 {
		t.Fatalf("terrain counts=%v want all approximate shallow water", report.TerrainCounts)
	}
	if len(report.StandingLadder) != 3 || report.StandingLadder[0] != "from_data" {
		t.Fatalf("standing ladder=%v", report.StandingLadder)
	}
}

func TestNormalizeDockableShore(t *testing.T) {
	land := true
	water := false
	report := Report{
		LandmaskProvider: "geojson",
		ShoreBandWidth:   1,
		Cells: []Cell{
			{Land: &water, DistanceToShoreTiles: 0.5, TerrainID: 58},
			{Land: &water, InlandWater: true, DistanceToShoreTiles: 0.5, TerrainID: 23},
			{Land: &land, DistanceToShoreTiles: 0.5, TerrainID: 53},
		},
	}
	normalizeDockableShore(&report)
	if report.DockableShoreCells != 1 {
		t.Fatalf("dockable shore cells=%d want 1", report.DockableShoreCells)
	}
	if report.Cells[0].TerrainID != 1 {
		t.Fatalf("coastal water terrain=%d want 1", report.Cells[0].TerrainID)
	}
	if report.Cells[1].TerrainID != 23 {
		t.Fatalf("inland water terrain=%d want 23", report.Cells[1].TerrainID)
	}
	if report.Cells[2].TerrainID != 53 {
		t.Fatalf("land terrain=%d want 53", report.Cells[2].TerrainID)
	}
}

func TestBuildRecipeCarriesPlacementAndDockableShorePolicy(t *testing.T) {
	land := true
	water := false
	report := Report{
		DatPath:        filepath.Join(t.TempDir(), "empires2_x2_p1.dat"),
		GridSize:       4,
		ShoreBandWidth: 1,
		Cells:          []Cell{{X: 0, Y: 0, Land: &water, TerrainID: 1, DistanceToShoreTiles: 0.5}, {X: 1, Y: 1, Land: &land, TerrainID: 53}},
	}
	recipe := BuildRecipe(report)
	if recipe.DatPath != report.DatPath {
		t.Fatalf("recipe dat_path=%q want %q", recipe.DatPath, report.DatPath)
	}
	var foundDockable bool
	for _, mask := range recipe.Masks {
		if mask.Name == "dockable_shore" {
			foundDockable = true
		}
	}
	if !foundDockable {
		t.Fatal("recipe missing dockable_shore mask")
	}
	var foundDockableOp bool
	for _, op := range recipe.Map {
		if op.Op == "set_terrain_mask" && op.Mask == "dockable_shore" && op.TerrainID != nil && *op.TerrainID == 1 {
			foundDockableOp = true
		}
	}
	if !foundDockableOp {
		t.Fatal("recipe missing dockable shoreline terrain operation")
	}
}

func TestLoadUSGSWoodyCSV(t *testing.T) {
	path := filepath.Join(t.TempDir(), "woody.csv")
	data := "ID,LATITUDE,LONGITUDE,Quercus virginiana,Pinus elliottii,Unmapped shrub\n1,27.5,-81.5,1,1,1\n2,30.0,-82.0,0,1,1\n"
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	grid, summary, err := loadUSGSWoody(path)
	if err != nil {
		t.Fatalf("loadUSGSWoody: %v", err)
	}
	if summary.PointCount != 2 || summary.SpeciesCount != 2 {
		t.Fatalf("summary=%+v, want two points and two mapped species", summary)
	}
	if got := usgsWoodyUnitChoices(grid.nearestSpecies(27.6, -81.5)); len(got) != 2 {
		t.Fatalf("nearest mapped units=%v, want two choices", got)
	}
	if got := grid.nearestSpecies(29.9, -82.1); len(got) != 1 || got[0] != "Pinus elliottii" {
		t.Fatalf("nearest species=%v, want Pinus elliottii", got)
	}
}

func TestTraceDEMLandmaskGeoJSONEmitsRecipe(t *testing.T) {
	dir := t.TempDir()
	landmask := filepath.Join(dir, "land.geojson")
	data := []byte(`{
  "type": "FeatureCollection",
  "features": [
    {
      "type": "Feature",
      "geometry": {
        "type": "Polygon",
        "coordinates": [[
          [0.5, 0.0],
          [1.0, 0.0],
          [1.0, 1.0],
          [0.5, 1.0],
          [0.5, 0.0]
        ]]
      }
    }
  ]
}`)
	if err := os.WriteFile(landmask, data, 0o644); err != nil {
		t.Fatalf("write landmask: %v", err)
	}
	report, err := TraceDEM(Options{
		BBox:         BBox{MinLat: 0, MinLon: 0, MaxLat: 1, MaxLon: 1},
		GridSize:     4,
		Provider:     "flat",
		LandmaskPath: landmask,
		Orient:       "grid",
	})
	if err != nil {
		t.Fatalf("TraceDEM: %v", err)
	}
	if report.CoastlineConfidence != "from_data" {
		t.Fatalf("coastline confidence=%q", report.CoastlineConfidence)
	}
	if report.LandmaskProvider != "geojson" {
		t.Fatalf("landmask provider=%q", report.LandmaskProvider)
	}
	if report.TerrainCounts[53] == 0 || report.TerrainCounts[58] == 0 {
		t.Fatalf("terrain counts=%v want land beach and water", report.TerrainCounts)
	}
	if report.Recipe == nil {
		t.Fatal("recipe=nil")
	}
	if len(report.Recipe.Masks) < 2 {
		t.Fatalf("masks=%d want land/water masks", len(report.Recipe.Masks))
	}
	if len(report.Recipe.Map) == 0 {
		t.Fatal("recipe has no map operations")
	}
	var hasLand, hasWater bool
	for _, mask := range report.Recipe.Masks {
		switch mask.Name {
		case "land":
			hasLand = true
		case "water":
			hasWater = true
		}
	}
	if !hasLand || !hasWater {
		t.Fatalf("missing land/water masks: %#v", report.Recipe.Masks)
	}
}

func TestTraceDEMIsoOutsideTraceDoesNotOverrideLandmaskLand(t *testing.T) {
	dir := t.TempDir()
	landmask := filepath.Join(dir, "all-land.geojson")
	data := []byte(`{
  "type": "Polygon",
  "coordinates": [[[-1,-1],[2,-1],[2,2],[-1,2],[-1,-1]]]
}`)
	if err := os.WriteFile(landmask, data, 0o644); err != nil {
		t.Fatalf("write landmask: %v", err)
	}
	report, err := TraceDEM(Options{
		BBox:         BBox{MinLat: 0, MinLon: 0, MaxLat: 1, MaxLon: 1},
		GridSize:     4,
		Provider:     "flat",
		LandmaskPath: landmask,
		Orient:       "iso",
	})
	if err != nil {
		t.Fatalf("TraceDEM: %v", err)
	}
	foundOutsideLand := false
	for _, cell := range report.Cells {
		if cell.OutsideTrace && cell.Land != nil && *cell.Land {
			foundOutsideLand = true
			break
		}
	}
	if !foundOutsideLand {
		t.Fatalf("expected an outside_trace cell to preserve landmask land")
	}
}

func TestTraceDEMScreenIsoPreservesViewportEarth(t *testing.T) {
	dir := t.TempDir()
	landmask := filepath.Join(dir, "all-land.geojson")
	data := []byte(`{
  "type": "Polygon",
  "coordinates": [[[-1,-1],[2,-1],[2,2],[-1,2],[-1,-1]]]
}`)
	if err := os.WriteFile(landmask, data, 0o644); err != nil {
		t.Fatalf("write landmask: %v", err)
	}
	report, err := TraceDEM(Options{
		BBox:         BBox{MinLat: 0, MinLon: 0, MaxLat: 1, MaxLon: 1},
		GridSize:     4,
		Provider:     "flat",
		LandmaskPath: landmask,
		Orient:       "iso-ccw-screen",
	})
	if err != nil {
		t.Fatalf("TraceDEM: %v", err)
	}
	var outsideLand, insideLand int
	for _, cell := range report.Cells {
		if cell.OutsideTrace {
			if cell.Land != nil && *cell.Land {
				outsideLand++
			}
			continue
		}
		if cell.Land != nil && *cell.Land {
			insideLand++
		}
	}
	if outsideLand == 0 || insideLand == 0 {
		t.Fatalf("outsideLand=%d insideLand=%d, want both", outsideLand, insideLand)
	}
}

func TestScreenIsoProjectionUsesSingleMatrix(t *testing.T) {
	proj := newGeoProjection(BBox{MinLat: 23, MinLon: -87.75, MaxLat: 31.05, MaxLon: -79.55}, "iso-ccw-screen", ProjectionOptions{})
	check := projectionSelfCheck(proj)
	if check == nil {
		t.Fatal("projection self-check is nil")
	}
	if absFloat(check.DueEastScreenDY) > 1e-9 {
		t.Fatalf("due east dy=%g, want horizontal", check.DueEastScreenDY)
	}
	if check.DueEastScreenDX <= 0 {
		t.Fatalf("due east dx=%g, want east-right", check.DueEastScreenDX)
	}
	if absFloat(check.DueNorthScreenDX) > 1e-9 {
		t.Fatalf("due north dx=%g, want vertical", check.DueNorthScreenDX)
	}
	if check.DueNorthScreenDY >= 0 {
		t.Fatalf("due north dy=%g, want north-up", check.DueNorthScreenDY)
	}
	report := Report{BBox: proj.b, Orient: proj.orient, EcologyProfile: EcologyProfile{CalibrationAxes: []CalibrationAxis{{Name: "peninsula", From: GeoPoint{Lat: 30.3, Lon: -81.7}, To: GeoPoint{Lat: 25.4, Lon: -80.5}}}}}
	measureCalibrationAxes(&report)
	angle := report.CalibrationAxes[0].AngleFromSouthDeg
	if angle < 10 || angle > 14 {
		t.Fatalf("peninsula angle=%g, want about 12 degrees east of south", angle)
	}
}

func TestScreenIsoProjectionAspectIsConfigurable(t *testing.T) {
	box := BBox{MinLat: 24, MinLon: -88, MaxLat: 32, MaxLon: -79.5}
	wide := newGeoProjection(box, "iso-ccw-screen", ProjectionOptions{Aspect: 2.4})
	compact := newGeoProjection(box, "iso-ccw-screen", ProjectionOptions{Aspect: 1.4})
	if wide.screenAspect != 2.4 || compact.screenAspect != 1.4 {
		t.Fatalf("projection aspects=(%g,%g)", wide.screenAspect, compact.screenAspect)
	}
	if wide.matrixScale == compact.matrixScale {
		t.Fatal("projection aspect did not change matrix scale")
	}
}

func TestLoadEcologyProfile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "profile.json")
	data := []byte(`{"schema":"aoe2kit-geotrace-ecology/v1","name":"test","resource_tree_units":[11,12],"decorative_flora_unit":13,"fish_by_band":{"shore":[14]}}`)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	profile, err := loadEcologyProfile(path)
	if err != nil {
		t.Fatalf("loadEcologyProfile: %v", err)
	}
	if profile.Name != "test" || len(profile.ResourceTreeUnits) != 2 || profile.DecorativeFloraUnit != 13 {
		t.Fatalf("profile=%+v", profile)
	}
	if got := profileFish(profile, "shore"); len(got) != 1 || got[0] != 14 {
		t.Fatalf("shore fish=%v", got)
	}
	if got := profileFish(profile, "deep"); len(got) == 0 {
		t.Fatal("missing bands should use defaults")
	}
}

func TestTraceDEMInlandWaterOverridesLandTerrain(t *testing.T) {
	dir := t.TempDir()
	landmask := filepath.Join(dir, "all-land.geojson")
	if err := os.WriteFile(landmask, []byte(`{"type":"Polygon","coordinates":[[[-1,-1],[2,-1],[2,2],[-1,2],[-1,-1]]]}`), 0o644); err != nil {
		t.Fatalf("write landmask: %v", err)
	}
	lakes := filepath.Join(dir, "lakes.geojson")
	lakeData := []byte(`{
  "type": "FeatureCollection",
  "features": [{
    "type": "Feature",
    "properties": {"name": "Lake Test"},
    "geometry": {
      "type": "Polygon",
      "coordinates": [[[0,0],[1,0],[1,1],[0,1],[0,0]]]
    }
  }]
}`)
	if err := os.WriteFile(lakes, lakeData, 0o644); err != nil {
		t.Fatalf("write lakes: %v", err)
	}
	report, err := TraceDEM(Options{
		BBox:            BBox{MinLat: 0, MinLon: 0, MaxLat: 1, MaxLon: 1},
		GridSize:        4,
		Provider:        "flat",
		LandmaskPath:    landmask,
		InlandWaterPath: lakes,
		Orient:          "grid",
	})
	if err != nil {
		t.Fatalf("TraceDEM: %v", err)
	}
	if report.InlandWaterProvider != "geojson" {
		t.Fatalf("inland water provider=%q", report.InlandWaterProvider)
	}
	for _, cell := range report.Cells {
		if !cell.InlandWater {
			t.Fatalf("expected every cell in synthetic lake to be inland water")
		}
		if cell.WaterbodyName != "Lake Test" {
			t.Fatalf("waterbody name=%q", cell.WaterbodyName)
		}
		if cell.ElevationBand != "inland_water" {
			t.Fatalf("elevation band=%q", cell.ElevationBand)
		}
	}
}

func TestLandcoverOverridesElevationBand(t *testing.T) {
	mapping, err := loadFeltMapping("")
	if err != nil {
		t.Fatalf("load mapping: %v", err)
	}
	land := true
	cell := classifyCell(
		Cell{X: 3, Y: 7, Lat: 25.5, Lon: -80.7},
		0,
		&land,
		9,
		"subtropical",
		mapping,
		false,
		"",
		&LandcoverSample{
			Valid:    true,
			Class:    90,
			Key:      "90_herbaceous_wetland",
			Fraction: 1,
			Histogram: []LandcoverStat{{
				Class:  90,
				Key:    "90_herbaceous_wetland",
				Weight: 1,
			}},
		},
	)
	if cell.ElevationBand != "landcover_90_herbaceous_wetland" {
		t.Fatalf("elevation band=%q", cell.ElevationBand)
	}
	if cell.TerrainConfidence != "from_data" {
		t.Fatalf("terrain confidence=%q", cell.TerrainConfidence)
	}
	if cell.TerrainID != 111 && cell.TerrainID != 55 && cell.TerrainID != 101 && cell.TerrainID != 95 {
		t.Fatalf("terrain id=%d not wetland mapped", cell.TerrainID)
	}
}

func TestWorldCoverTileName(t *testing.T) {
	got := worldCoverTileName(25.5, -80.7)
	want := "ESA_WorldCover_10m_2021_v200_N24W081_Map.tif"
	if got != want {
		t.Fatalf("tile=%q want %q", got, want)
	}
}

func TestSortedLandcoverArrayStatsMatchesPixelSemantics(t *testing.T) {
	var counts [256]int
	counts[10] = 5
	counts[30] = 4
	counts[80] = 1
	got := sortedLandcoverArrayStats(counts, 10, worldCoverClassKey)
	if len(got) != 3 || got[0].Class != 10 || got[1].Class != 30 || got[2].Class != 80 {
		t.Fatalf("stats=%+v", got)
	}
	if got[0].Weight != 0.5 || got[1].Weight != 0.4 || got[2].Weight != 0.1 {
		t.Fatalf("weights=%+v", got)
	}
}

func TestRoadSimplificationUsesTileScale(t *testing.T) {
	way := roadWay{Geometry: []point{}}
	for i := 0; i <= 100; i++ {
		x := float64(i) / 100
		way.Geometry = append(way.Geometry, point{
			Lat: 25 + x,
			Lon: -82 + x + math.Sin(x*math.Pi*12)*0.0001,
		})
	}
	ways := simplifyRoadWaysForViewport([]roadWay{way}, BBox{MinLat: 25, MinLon: -82, MaxLat: 26, MaxLon: -81}, 100, 1.25)
	if len(ways) != 1 {
		t.Fatalf("ways=%d", len(ways))
	}
	if len(ways[0].Geometry) >= len(way.Geometry)/2 {
		t.Fatalf("road simplification did not reduce dense near-straight geometry: got %d raw %d", len(ways[0].Geometry), len(way.Geometry))
	}
	if ways[0].Geometry[0] != way.Geometry[0] || ways[0].Geometry[len(ways[0].Geometry)-1] != way.Geometry[len(way.Geometry)-1] {
		t.Fatalf("simplification must preserve line endpoints")
	}
}

func TestFederalGeoJSONInterstateTags(t *testing.T) {
	data := []byte(`{
  "type": "FeatureCollection",
  "features": [
    {
      "type": "Feature",
      "properties": {"ROUTE_ID": "I-35", "FULLNAME": "Interstate 35"},
      "geometry": {"type": "LineString", "coordinates": [[-98, 30], [-97, 31]]}
    },
    {
      "type": "Feature",
      "properties": {"ROUTE_ID": "US-290", "FULLNAME": "US 290"},
      "geometry": {"type": "LineString", "coordinates": [[-99, 30], [-98, 31]]}
    }
  ]
}`)
	ways, err := parseGeoJSONRoadWays(data)
	if err != nil {
		t.Fatalf("parseGeoJSONRoadWays: %v", err)
	}
	if len(ways) != 2 {
		t.Fatalf("ways=%d", len(ways))
	}
	filtered := filterRoadWaysForProvider(ways, "nhpn-interstate")
	if len(filtered) != 1 {
		t.Fatalf("interstate ways=%d", len(filtered))
	}
	if filtered[0].Tags["ref"] != "I-35" || filtered[0].Tags["name"] != "Interstate 35" {
		t.Fatalf("tags=%v", filtered[0].Tags)
	}
}
