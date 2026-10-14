package geotrace

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"aoe2kit/pkg/scenario"
)

func TestProfileOverlaysAreExplicitAndUseProjection(t *testing.T) {
	r := Report{BBox: BBox{MinLat: -44, MinLon: 165, MaxLat: -34, MaxLon: 179}, GridSize: 32, Orient: "iso-ccw-screen", ProjectionAspect: 3, ProjectionNorthScale: 2}
	r.Cells = gridPoints(r.projection(), r.GridSize)
	applyDerivedFeatureMasks(&r)
	if r.RoadCells != 0 || r.CausewayCells != 0 {
		t.Fatal("implicit regional road")
	}
	r.EcologyProfile.Overlays = []GeographicOverlay{{Name: "connection", Kind: "inferred_corridor", Provenance: "designed", Points: []GeoPoint{{Lat: -42, Lon: 170}, {Lat: -36, Lon: 174}}}}
	applyDerivedFeatureMasks(&r)
	for _, point := range r.EcologyProfile.Overlays[0].Points {
		x, y, ok := r.projection().gridCellForLatLon(r.GridSize, point.Lat, point.Lon)
		if !ok || !r.Cells[y*r.GridSize+x].Causeway {
			t.Fatal("overlay missed projected endpoint")
		}
	}
	if r.RoadCells == 0 || r.CausewayCells != r.RoadCells {
		t.Fatal("missing road diagnostics")
	}
}

func TestProfileRejectsInvalidInput(t *testing.T) {
	for _, data := range []string{
		`{"schema":"unknown"}`, `{"decorative_flora_unt":1366}`, `{"resource_tree_units":[-1]}`,
		`{"fish_by_band":{"typo":[1]}}`, `{"landmarks":[{"name":"invalid"}]}`, `{} {}`,
		`{"overlays":[{"name":"route","kind":"inferred_corridor","points":[{"lat":0,"lon":0},{"lat":1,"lon":1}]}]}`,
	} {
		path := filepath.Join(t.TempDir(), "profile.json")
		if err := os.WriteFile(path, []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := loadEcologyProfile(path); err == nil {
			t.Fatalf("accepted invalid profile %s", data)
		}
	}
}

func TestGeneratedRotationsAreValidForMappedUnits(t *testing.T) {
	land := true
	r := Report{BBox: BBox{MinLat: 0, MinLon: 0, MaxLat: 1, MaxLon: 1}, GridSize: 32, Orient: "grid", FloraDensity: 10, LandFaunaDensity: 10, EcologyProfile: defaultEcologyProfile()}
	for y := 0; y < 32; y++ {
		for x := 0; x < 32; x++ {
			r.Cells = append(r.Cells, Cell{X: x, Y: y, Land: &land, LandcoverKey: "10_tree_cover", TerrainID: 10, ClimateZone: "temperate"})
		}
	}
	recipe := BuildRecipe(r)
	if len(recipe.Units) == 0 {
		t.Fatal("no units tested")
	}
	for _, u := range recipe.Units {
		angles := u.RotationChoices
		if u.Rotation != nil {
			angles = append(append([]float64{}, angles...), *u.Rotation)
		}
		if len(angles) == 0 {
			t.Fatalf("unit %d leaves rotation to writer", u.UnitConst)
		}
		for _, a := range angles {
			if record, ok := visualVariantRecords[u.UnitConst]; ok {
				allowed := false
				for _, facets := range record.States {
					for _, facet := range facets {
						allowed = allowed || a == float64(facet)
					}
				}
				if !allowed {
					t.Fatalf("unit %d has invalid artwork index %g", u.UnitConst, a)
				}
			} else if a < 0 || a >= 2*math.Pi || !finite(a) {
				t.Fatalf("unit %d has invalid rotation %g", u.UnitConst, a)
			}
		}
	}
	if _, err := json.Marshal(recipe); err != nil {
		t.Fatal(err)
	}
}

func TestOakPlacementUsesManyAllowedFacetsDeterministically(t *testing.T) {
	land := true
	report := Report{GridSize: 40, Seed: 187, EcologyProfile: defaultEcologyProfile()}
	report.EcologyProfile.Layers = []HabitatLayer{{
		Name: "oak-test", Role: "resource", Units: []WeightedUnit{{UnitConst: 349, Weight: 1}},
		Rate: 1, Limit: 1600,
	}}
	for y := 0; y < report.GridSize; y++ {
		for x := 0; x < report.GridSize; x++ {
			report.Cells = append(report.Cells, Cell{X: x, Y: y, Land: &land, LandcoverKey: "10_tree_cover", ClimateZone: "subtropical"})
		}
	}
	first := BuildRecipe(report)
	second := BuildRecipe(report)
	allowed := map[int]bool{}
	for _, facet := range visualVariantRecords[349].States["green"] {
		allowed[facet] = true
	}
	histogram := map[int]int{}
	var firstRotations, secondRotations []float64
	for run, recipe := range []scenario.Recipe{first, second} {
		rotations := &firstRotations
		if run == 1 {
			rotations = &secondRotations
		}
		for _, unit := range recipe.Units {
			if unit.UnitConst != 349 || unit.Rotation == nil {
				continue
			}
			facet := int(*unit.Rotation)
			if float64(facet) != *unit.Rotation || !allowed[facet] {
				t.Fatalf("oak rotation %g is outside the green facet list", *unit.Rotation)
			}
			*rotations = append(*rotations, *unit.Rotation)
			if run == 0 {
				histogram[facet]++
			}
		}
	}
	if len(firstRotations) < 500 {
		t.Fatalf("oak placements = %d, want a large sample", len(firstRotations))
	}
	if !reflect.DeepEqual(firstRotations, secondRotations) {
		t.Fatal("same seed produced different oak facet sequence")
	}
	if len(histogram) < 20 {
		t.Fatalf("oak facet histogram has only %d distinct facets: %v", len(histogram), histogram)
	}
}
