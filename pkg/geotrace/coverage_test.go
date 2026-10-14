package geotrace

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCoverageCountsProjectedCorners(t *testing.T) {
	land := true
	r := Report{LandmaskProvider: "fixture", LandcoverProvider: "fixture", USGSWoodyProvider: "fixture", Cells: []Cell{
		{Land: &land, OutsideTrace: true, LandcoverKey: "10_tree_cover", USGSWoodySpecies: []string{"Pinus test"}},
		{Land: &land, OutsideTrace: true},
	}}
	measureSourceCoverage(&r)
	d := r.SourceCoverage["landcover"]
	if d.State != "partial" || d.OutsideNominalCovered != 1 || d.OutsideNominalCells != 2 || d.MissingCells != 1 {
		t.Fatalf("coverage %+v", d)
	}
}

func TestLocalRoadGeometryDoesNotProveQueryExtent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "roads.json")
	if err := os.WriteFile(path, []byte(`{"elements":[]}`), 0600); err != nil {
		t.Fatal(err)
	}
	ways := []roadWay{{Geometry: []point{{Lat: 10, Lon: 20}, {Lat: 12, Lon: 22}}}}
	d := roadSourceCoverage(ways, Options{RoadsPath: path})
	if d.QueryBounds != nil || d.State != "query_extent_unknown" || d.GeometryBounds.MaxLat != 12 {
		t.Fatalf("coverage %+v", d)
	}
	if err := os.WriteFile(path, []byte(`{"kit_query_bbox":{"min_lat":9,"min_lon":19,"max_lat":13,"max_lon":23}}`), 0600); err != nil {
		t.Fatal(err)
	}
	d = roadSourceCoverage(ways, Options{RoadsPath: path})
	if d.QueryBounds == nil || d.State != "query_extent_declared" {
		t.Fatalf("coverage %+v", d)
	}
}

func TestProjectionEnvelopeIncludesAllGridPoints(t *testing.T) {
	b := BBox{MinLat: 20, MinLon: -90, MaxLat: 30, MaxLon: -80}
	p := newGeoProjection(b, "iso-ccw-screen", ProjectionOptions{Aspect: 2, NorthScale: 1})
	envelope := projectionEnvelope(p)
	outside := 0
	for _, c := range gridPoints(p, 40) {
		if !bboxContains(envelope, c.Lat, c.Lon) {
			t.Fatalf("grid point not covered: %g,%g", c.Lat, c.Lon)
		}
		if !bboxContains(b, c.Lat, c.Lon) {
			outside++
		}
	}
	if outside == 0 {
		t.Fatal("fixture does not exercise expanded envelope")
	}
}
