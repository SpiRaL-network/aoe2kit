package geotrace

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"testing"
)

func TestCoastCorpusPoints(t *testing.T) {
	root := os.Getenv("AOE2KIT_COAST_CORPUS")
	if root == "" {
		t.Skip("optional coastline corpus")
	}
	if _, err := os.Stat(root); err != nil {
		t.Skip(err)
	}
	points := []Cell{{Lat: 26.452, Lon: -81.951}, {Lat: 26.440, Lon: -81.935}, {Lat: 26.425, Lon: -81.915}}
	opts := Options{BBox: BBox{26.3680771, -81.9859368, 26.5028007, -81.6757310}, GridSize: 480, LandmaskProvider: "geojson", LandmaskPath: filepath.Join(root, "natural-earth-land.geojson"), LandcoverProvider: "worldcover", LandcoverCacheDir: filepath.Join(root, "worldcover")}
	mask, _, err := sampleLandmask(opts, points)
	if err != nil {
		t.Fatal(err)
	}
	samples, _, err := sampleLandcover(opts, points, nil)
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(root, "report.json"))
	if err != nil {
		t.Fatal(err)
	}
	var report Report
	if err = json.Unmarshal(b, &report); err != nil {
		t.Fatal(err)
	}
	for i, p := range points {
		var near Cell
		best := math.Inf(1)
		for _, c := range report.Cells {
			d := math.Hypot(c.Lat-p.Lat, (c.Lon-p.Lon)*math.Cos(p.Lat*math.Pi/180))
			if d < best {
				best = d
				near = c
			}
		}
		t.Logf("point %.6f,%.6f mask_land=%v class=%d valid=%v nearest=(%d,%d) terrain=%d land=%v distance_m=%.1f", p.Lat, p.Lon, mask[i], samples[i].Class, samples[i].Valid, near.X, near.Y, near.TerrainID, *near.Land, best*111320)
	}
}
