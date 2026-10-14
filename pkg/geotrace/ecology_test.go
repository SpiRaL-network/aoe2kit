package geotrace

import (
	"bytes"
	"encoding/json"
	"math"
	"testing"
)

func syntheticEcologyReport() Report {
	land, water := true, false
	r := Report{GridSize: 64, FloraDensity: 10, LandFaunaDensity: 10, FishDensity: 10}
	for y := 0; y < 64; y++ {
		for x := 0; x < 64; x++ {
			c := Cell{X: x, Y: y, Land: &land, TerrainID: 10, LandcoverKey: "10_tree_cover", ClimateZone: "temperate"}
			if x >= 40 {
				c.Land = &water
				c.TerrainID = 1
				c.LandcoverKey = ""
				c.DistanceToShoreTiles = float64(x - 39)
			}
			if y == 32 {
				c.Road = true
			}
			r.Cells = append(r.Cells, c)
		}
	}
	return r
}

func TestEcologyHabitatDeterminismAndSpacing(t *testing.T) {
	r := syntheticEcologyReport()
	p := defaultEcologyProfile()
	p.Layers = []HabitatLayer{
		{Name: "forest/resource", Role: "resource", LandcoverKeys: []string{"10_tree_cover"}, Units: []WeightedUnit{{101, 1}, {102, 3}}, Rate: 0.6, Limit: 10000, PatchScale: 8},
		{Name: "forest/decor", Role: "decorative", LandcoverKeys: []string{"10_tree_cover"}, Units: []WeightedUnit{{103, 1}}, Rate: 0.5, Limit: 10000},
		{Name: "forest/hunt", Role: "fauna", Units: []WeightedUnit{{104, 1}}, Rate: 1, Limit: 10000, Spacing: 5},
		{Name: "shore/fish", Role: "fish", WaterBand: "shore", Units: []WeightedUnit{{105, 1}}, Rate: 0.2, Limit: 10000, Spacing: 4},
	}
	units, stats := buildHabitatEcology(r, p)
	other, _ := buildHabitatEcology(r, p)
	a, _ := json.Marshal(units)
	b, _ := json.Marshal(other)
	if !bytes.Equal(a, b) {
		t.Fatal("same seed changes output")
	}
	r.Seed = 123
	different, _ := buildHabitatEcology(r, p)
	b, _ = json.Marshal(different)
	if bytes.Equal(a, b) {
		t.Fatal("seed has no effect")
	}
	seen := map[int]map[[2]float64]bool{}
	var hunts [][2]float64
	counts := map[int]int{}
	fractional := false
	for _, u := range units {
		x, y := *u.X, *u.Y
		c := r.Cells[int(math.Floor(y))*r.GridSize+int(math.Floor(x))]
		if c.Road {
			t.Fatal("placed on a road")
		}
		if seen[u.UnitConst] == nil {
			seen[u.UnitConst] = map[[2]float64]bool{}
		}
		point := [2]float64{x, y}
		if seen[u.UnitConst][point] {
			t.Fatal("duplicate placement")
		}
		seen[u.UnitConst][point] = true
		counts[u.UnitConst]++
		switch u.UnitConst {
		case 101, 102:
			if c.LandcoverKey != "10_tree_cover" || u.Snap != "grid" || x != math.Floor(x)+0.5 || y != math.Floor(y)+0.5 {
				t.Fatal("resource placement violated habitat/grid")
			}
		case 103:
			if u.Snap != "auto" {
				t.Fatal("decoration bypasses DAT placement policy")
			}
			fractional = fractional || x != math.Floor(x)+0.5
		case 104:
			for _, h := range hunts {
				if squaredDistance(x, y, h[0], h[1]) < 25 {
					t.Fatal("fauna spacing violated")
				}
			}
			hunts = append(hunts, point)
		case 105:
			if waterBand(c) != "shore" {
				t.Fatal("fish outside band")
			}
		}
	}
	if !fractional || counts[101] == 0 || counts[102] <= counts[101] || counts[104] == 0 || counts[105] == 0 {
		t.Fatalf("incomplete palette/coverage %v", counts)
	}
	var huntStats *LayerDiagnostics
	for i := range stats {
		if stats[i].Name == "forest/hunt" {
			huntStats = &stats[i]
		}
	}
	if huntStats == nil || huntStats.TruncatedBySpacing <= 0 {
		t.Fatal("saturated habitat should report unmet quota")
	}
}

func TestFishDensityControlsCount(t *testing.T) {
	r := syntheticEcologyReport()
	p := defaultEcologyProfile()
	p.Layers = []HabitatLayer{{Name: "fish", Role: "fish", WaterBand: "shore", Units: []WeightedUnit{{45, 1}}, Rate: 0.5, Limit: 500}}
	high, _ := buildHabitatEcology(r, p)
	r.ShoreFishDensity = 1
	low, _ := buildHabitatEcology(r, p)
	if len(low) == 0 || len(low) >= len(high)/2 {
		t.Fatalf("density ignored: %d -> %d", len(high), len(low))
	}
	p.Layers = []HabitatLayer{}
	disabled, _ := buildHabitatEcology(r, p)
	if len(disabled) != 0 {
		t.Fatal("empty layers should disable ecology")
	}
}

func TestAmphibiousFloraAndPopulationCenterDetection(t *testing.T) {
	land, water := true, false
	hybrid := Cell{X: 3, Y: 3, Land: &water, TerrainID: 54}
	pureWater := Cell{X: 4, Y: 3, Land: &water, TerrainID: 22}
	worldCoverLandOnWater := Cell{X: 4, Y: 4, Land: &land, TerrainID: 22}
	hybridMarkedLand := Cell{X: 5, Y: 4, Land: &land, TerrainID: 111}
	if !matchesLayer(hybrid, HabitatLayer{Role: "resource"}) {
		t.Fatal("DAT-gated flora should be eligible on mangrove shallows")
	}
	if matchesLayer(pureWater, HabitatLayer{Role: "resource"}) || matchesLayer(worldCoverLandOnWater, HabitatLayer{Role: "resource"}) {
		t.Fatal("land flora should not be eligible on deep water")
	}
	if !floraSurface(Cell{Land: &land, TerrainID: 0}) || !floraSurface(hybridMarkedLand) || floraSurface(pureWater) {
		t.Fatal("flora surface classification")
	}

	r := Report{GridSize: 24, Cells: make([]Cell, 24*24)}
	for y := 0; y < r.GridSize; y++ {
		for x := 0; x < r.GridSize; x++ {
			c := &r.Cells[y*r.GridSize+x]
			c.X, c.Y = x, y
			c.Land = &land
		}
	}
	r.Cells[5*r.GridSize+5].LandcoverClass = 50
	for y := 10; y < 16; y++ {
		for x := 10; x < 16; x++ {
			if (x+y)%2 == 0 {
				r.Cells[y*r.GridSize+x].Road = true
			}
		}
	}
	markPopulationCenters(&r)
	if !r.Cells[5*r.GridSize+5].PopulationCenter || r.PopulationCenterBuiltUpCells != 1 || r.PopulationCenterRoadDensityCells == 0 {
		t.Fatalf("population-center signals missing: built=%d road=%d", r.PopulationCenterBuiltUpCells, r.PopulationCenterRoadDensityCells)
	}
}

func TestPopulationCenterStrataAndBuildingClusters(t *testing.T) {
	land := true
	r := Report{GridSize: 48, Seed: 7, FloraDensity: 7, Cells: make([]Cell, 48*48)}
	for y := 0; y < r.GridSize; y++ {
		for x := 0; x < r.GridSize; x++ {
			c := Cell{X: x, Y: y, Land: &land, TerrainID: 10, BuiltUpCore: true}
			if x >= 24 {
				c.BuiltUpCore = false
				c.RoadCorridor = true
			}
			if x == 24 {
				c.Road = true
			}
			r.Cells[y*r.GridSize+x] = c
		}
	}
	if !matchesLayer(r.Cells[10*r.GridSize+10], HabitatLayer{Role: "building", PopulationCenterLayer: "built_up_core"}) {
		t.Fatal("built-up core cell rejected")
	}
	if matchesLayer(r.Cells[10*r.GridSize+30], HabitatLayer{Role: "building", PopulationCenterLayer: "built_up_core"}) {
		t.Fatal("road corridor was classified as built-up core")
	}
	if !matchesLayer(r.Cells[10*r.GridSize+30], HabitatLayer{Role: "decorative", PopulationCenterLayer: "road_corridor"}) {
		t.Fatal("road corridor decoration cell rejected")
	}
	p := defaultEcologyProfile()
	p.Layers = []HabitatLayer{{
		Name: "town_houses", Role: "building", PopulationCenterLayer: "built_up_core",
		Units: []WeightedUnit{{UnitConst: 70, Weight: 1}}, Rate: 1, Limit: 6,
		ClusterCount: 3, ClusterSize: 2, ClusterRadius: 6, ClusterSpacing: 14,
		Spacing: 3, FootprintRadius: 1,
	}}
	if err := validateEcologyProfile(p); err == nil {
		t.Fatal("urban layer enabled without explicit unvalidated opt-in")
	}
	p.EnableUnvalidatedUrban = true
	if err := validateEcologyProfile(p); err != nil {
		t.Fatalf("validate building profile: %v", err)
	}
	units, diagnostics := buildHabitatEcology(r, p)
	if len(units) != 6 || len(diagnostics) != 1 || diagnostics[0].ClustersEmitted != 3 {
		t.Fatalf("emitted %d houses in %+v", len(units), diagnostics)
	}
	occupied := map[[2]int]bool{}
	for _, u := range units {
		x, y := int(math.Floor(*u.X)), int(math.Floor(*u.Y))
		if !r.Cells[y*r.GridSize+x].BuiltUpCore {
			t.Fatalf("house center (%d,%d) is outside built-up core", x, y)
		}
		for fy := y - 1; fy <= y+1; fy++ {
			for fx := x - 1; fx <= x+1; fx++ {
				key := [2]int{fx, fy}
				cell := r.Cells[fy*r.GridSize+fx]
				if occupied[key] || cell.Land == nil || !*cell.Land || cell.Road || cell.Roadside {
					t.Fatalf("unsafe/overlapping house footprint at %v", key)
				}
				occupied[key] = true
			}
		}
	}
	if diagnostics[0].FootprintTiles != 54 {
		t.Fatalf("DAT footprint tiles=%d, want 54", diagnostics[0].FootprintTiles)
	}
}

func TestBuildingProfileRequiresNonOverlappingDATFootprintSpacing(t *testing.T) {
	p := defaultEcologyProfile()
	p.EnableUnvalidatedUrban = true
	p.Layers = []HabitatLayer{{Name: "houses", Role: "building", Units: []WeightedUnit{{UnitConst: 70, Weight: 1}}, Rate: 1, Limit: 6, ClusterCount: 3, ClusterSize: 2, ClusterRadius: 6, ClusterSpacing: 14, Spacing: 2.9, FootprintRadius: 1}}
	if validateEcologyProfile(p) == nil {
		t.Fatal("accepted building spacing smaller than the configured footprint")
	}
}

func TestUrbanAccentsAreDisabledByDefault(t *testing.T) {
	p := defaultEcologyProfile()
	p.Layers = []HabitatLayer{{Name: "custom", Role: "decorative", Units: []WeightedUnit{{UnitConst: 1366, Weight: 1}}, Rate: 1}}
	ensurePopulationCenterLayer(&p)
	if len(p.Layers) != 1 {
		t.Fatal("implicit urban accent appended")
	}
	for _, l := range defaultHabitatLayers(p, true) {
		if l.PopulationCenterOnly || l.PopulationCenterLayer != "" || l.Role == "building" {
			t.Fatal("default urban layer", l)
		}
	}
}

func TestResourceLayersDoNotOverlap(t *testing.T) {
	r := syntheticEcologyReport()
	p := defaultEcologyProfile()
	p.Layers = []HabitatLayer{
		{Name: "a", Role: "resource", Units: []WeightedUnit{{1, 1}}, Rate: 1, Limit: 10000},
		{Name: "b", Role: "resource", Units: []WeightedUnit{{2, 1}}, Rate: 1, Limit: 10000},
	}
	units, _ := buildHabitatEcology(r, p)
	seen := map[[2]float64]bool{}
	for _, u := range units {
		key := [2]float64{*u.X, *u.Y}
		if seen[key] {
			t.Fatal("two blocking trees share a tile")
		}
		seen[key] = true
	}
}

func TestEcologyRolePalettesAndFishSeparation(t *testing.T) {
	r := syntheticEcologyReport()
	p := defaultEcologyProfile()
	p.Palettes = map[string][]WeightedUnit{"NERITIC": {{69, 1}}, "SALTWATER": {{458, 1}}}
	layers := defaultHabitatLayers(p, true)
	var fish []HabitatLayer
	for _, layer := range layers {
		if layer.Role == "fish" {
			fish = append(fish, layer)
		}
	}
	p.Layers = fish
	units, _ := buildHabitatEcology(r, p)
	if len(units) == 0 {
		t.Fatal("no fish")
	}
	for i, a := range units {
		for _, b := range units[:i] {
			spacing := 10.0
			if a.UnitConst == 69 || b.UnitConst == 69 {
				spacing = 26
			}
			if squaredDistance(*a.X, *a.Y, *b.X, *b.Y) < spacing*spacing {
				t.Fatal("fish bands violate shared spacing")
			}
		}
	}
	p.Layers = []HabitatLayer{{Name: "custom", Role: "fish", EcologyRole: "NERITIC", WaterBand: "shore", Rate: 0.1, Limit: 10}}
	if err := validateEcologyProfile(p); err != nil {
		t.Fatal(err)
	}
	units, _ = buildHabitatEcology(r, p)
	for _, unit := range units {
		if unit.UnitConst != 69 {
			t.Fatal("role palette ignored")
		}
	}
	p.Palettes["NERITIC"] = []WeightedUnit{{69, math.MaxFloat64}, {1141, math.MaxFloat64}}
	if validateEcologyProfile(p) == nil {
		t.Fatal("overflowing weights accepted")
	}
}
