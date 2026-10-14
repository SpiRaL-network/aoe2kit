package geotrace

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSourceAttributionAndUnknownQueryExtent(t *testing.T) {
	b := BBox{MinLat: 0, MinLon: 0, MaxLat: 1, MaxLon: 1}
	a, ok := osmRoadAttribution(Options{RoadsProvider: "osm-major"}, SourceCoverage{State: "query_extent_known", QueryBounds: &b})
	if !ok || a.License != "ODbL 1.0" || a.QueryBounds != &b {
		t.Fatal(a, ok)
	}
	w := worldCoverAttribution(b)
	if w.License != "CC BY 4.0" || !strings.Contains(w.Text, "Copernicus") {
		t.Fatal(w)
	}
	p := filepath.Join(t.TempDir(), "roads.json")
	if err := os.WriteFile(p, []byte(`{"elements":[]}`), 0644); err != nil {
		t.Fatal(err)
	}
	a, ok = osmRoadAttribution(Options{RoadsPath: p}, SourceCoverage{State: "query_extent_unknown"})
	if !ok || a.QueryBounds != nil {
		t.Fatal(a, ok)
	}
	raw, err := json.Marshal(a)
	if err != nil || !strings.Contains(string(raw), `"query_bounds":null`) {
		t.Fatal(string(raw), err)
	}
	if _, ok := osmRoadAttribution(Options{RoadsProvider: "nhpn-interstate"}, SourceCoverage{}); ok {
		t.Fatal("non-OSM source mislabeled")
	}
}

func TestTraceReportRetainsOSMAttributionForEmptyLocalSource(t *testing.T) {
	p := filepath.Join(t.TempDir(), "roads.json")
	if err := os.WriteFile(p, []byte(`{"elements":[],"kit_query_bbox":{"min_lat":0,"min_lon":0,"max_lat":1,"max_lon":1}}`), 0644); err != nil {
		t.Fatal(err)
	}
	r, err := TraceDEM(Options{BBox: BBox{MinLat: 0, MinLon: 0, MaxLat: 1, MaxLon: 1}, GridSize: 4, Provider: "flat", RoadsPath: p})
	if err != nil {
		t.Fatal(err)
	}
	a, ok := r.Attribution["roads"]
	if !ok || !r.Experimental || a.QueryBounds == nil || a.ExtentState != "query_extent_declared" || a.License != "ODbL 1.0" {
		t.Fatal(r.Attribution, r.Experimental)
	}
}
