package geotrace

import (
	"math"
	"testing"
)

func TestVectorRoadStrokeRoundCapsWidthAndLandClipping(t *testing.T) {
	land := true
	r := Report{
		GridSize: 12,
		BBox:     BBox{MinLat: 0, MinLon: 0, MaxLat: 1, MaxLon: 1},
		Orient:   "grid",
		Cells:    make([]Cell, 12*12),
	}
	for i := range r.Cells {
		r.Cells[i] = Cell{Land: &land, DistanceToShoreTiles: 10}
	}
	water := false
	r.Cells[5*12+6] = Cell{Land: &water, DistanceToShoreTiles: 10}
	proj := r.projection()
	path := []point{{Lat: .5, Lon: .25}, {Lat: .5, Lon: .75}}

	thin := strokeRoadPolyline(&r, proj, path, 1)
	wide := strokeRoadPolyline(&r, proj, path, 2)
	thinSet, wideSet := map[int]bool{}, map[int]bool{}
	for _, idx := range thin {
		thinSet[idx] = true
	}
	for _, idx := range wide {
		wideSet[idx] = true
	}
	if len(wideSet) <= len(thinSet) {
		t.Fatalf("wider vector stroke did not cover more tiles: thin=%d wide=%d", len(thinSet), len(wideSet))
	}
	if !thinSet[5*12+3] {
		t.Fatal("round cap did not cover the road endpoint tile")
	}
	if thinSet[5*12+6] || wideSet[5*12+6] {
		t.Fatal("road stroke crossed onto disallowed water")
	}
}

func TestRoadHighwayClassStyling(t *testing.T) {
	base := 1.0
	classes := []string{"motorway", "trunk", "primary", "secondary", "tertiary", "residential"}
	last := 100.0
	for _, class := range classes {
		width := roadWidthForHighway(class, base)
		if width > last {
			t.Fatalf("class %q width %.2f exceeds preceding class width %.2f", class, width, last)
		}
		last = width
	}
	if roadTerrainForWay(map[string]string{"highway": "primary"}) != 24 || roadTerrainForWay(map[string]string{"surface": "gravel"}) != 78 {
		t.Fatal("road terrain assignment did not distinguish paved from gravel surfaces")
	}
}

func TestRoadWidthMultipliersStayNearPathScale(t *testing.T) {
	base := 0.7
	want := map[string]float64{"motorway": 0.98, "primary": 0.875, "secondary": 0.805, "tertiary": 0.735, "residential": 0.7}
	for class, expected := range want {
		if got := roadWidthForHighway(class, base); math.Abs(got-expected) > 1e-9 {
			t.Fatalf("%s width = %.3f, want %.3f", class, got, expected)
		}
	}
}

func TestBuildRecipeKeepsRoadSurfaceTerrainsSeparate(t *testing.T) {
	r := Report{GridSize: 2, Cells: []Cell{
		{X: 0, Y: 0, Road: true, RoadTerrainID: 24},
		{X: 1, Y: 0, Road: true, RoadTerrainID: 78},
	}}
	recipe := BuildRecipe(r)
	seen := map[int]int{}
	for _, op := range recipe.Map {
		if op.Op == "set_terrain_mask" && op.TerrainID != nil && (op.Mask == "roads_24" || op.Mask == "roads_78") {
			seen[*op.TerrainID]++
		}
	}
	if seen[24] != 1 || seen[78] != 1 {
		t.Fatalf("road masks did not preserve both terrain styles: %v", seen)
	}
}
