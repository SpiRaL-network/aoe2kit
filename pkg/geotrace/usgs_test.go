package geotrace

import (
	"fmt"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWoodySpatialLookupMatchesBoundedScan(t *testing.T) {
	rng := rand.New(rand.NewSource(91))
	var csv strings.Builder
	csv.WriteString("LATITUDE,LONGITUDE,Pinus test\n")
	for i := 0; i < 2000; i++ {
		fmt.Fprintf(&csv, "%.8f,%.8f,1\n", rng.Float64()*180-90, rng.Float64()*360-180)
	}
	csv.WriteString("0,179.99,1\n89.99,-120,1\n")
	path := filepath.Join(t.TempDir(), "atlas.csv")
	if err := os.WriteFile(path, []byte(csv.String()), 0600); err != nil {
		t.Fatal(err)
	}
	grid, summary, err := loadUSGSWoody(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(summary.SHA256) != 64 {
		t.Fatal("missing checksum")
	}
	for i := range grid.Points {
		grid.Points[i].Species = []string{fmt.Sprint(i)}
	}
	for i := 0; i < 500; i++ {
		lat, lon := rng.Float64()*180-90, rng.Float64()*360-180
		if i < 2 {
			lat, lon = 0, -179.99
			if i == 1 {
				lat, lon = 89.99, 70
			}
		}
		nearest := -1
		distance := usgsWoodyMaxDistanceKM
		for j, p := range grid.Points {
			d := greatCircleKM(lat, lon, p.Lat, p.Lon)
			if d < distance {
				nearest = j
				distance = d
			}
		}
		got := grid.nearestSpecies(lat, lon)
		if (nearest >= 0) != (len(got) > 0) {
			t.Fatalf("coverage mismatch at %g,%g: scan=%d indexed=%v", lat, lon, nearest, got)
		}
		if nearest >= 0 && got[0] != fmt.Sprint(nearest) {
			t.Fatalf("wrong nearest point: want %d got %v", nearest, got)
		}
	}
	if d := greatCircleKM(0, 179.99, 0, -179.99); math.Abs(d-2.2239) > 0.001 {
		t.Fatalf("dateline distance %g", d)
	}
}

func TestWoodyRejectsInvalidCoordinates(t *testing.T) {
	for _, value := range []string{"NaN", "Inf", "91"} {
		p := filepath.Join(t.TempDir(), "invalid.csv")
		if err := os.WriteFile(p, []byte("LATITUDE,LONGITUDE\n"+value+",0\n"), 0600); err != nil {
			t.Fatal(err)
		}
		if _, _, err := loadUSGSWoody(p); err == nil {
			t.Fatalf("accepted latitude %s", value)
		}
	}
}
