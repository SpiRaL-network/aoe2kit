package geotrace

import (
	"fmt"
	"math"
	"math/rand"
	"testing"
)

func TestScreenProjectionFitsAndRoundTrips(t *testing.T) {
	bounds := []BBox{
		{MinLat: 23, MaxLat: 32, MinLon: -88, MaxLon: -79},
		{MinLat: -44, MaxLat: -34, MinLon: 166, MaxLon: 179},
		{MinLat: 59, MaxLat: 60, MinLon: 9, MaxLon: 11},
		{MinLat: -0.01, MaxLat: 0.01, MinLon: 10, MaxLon: 10.02},
	}
	for bi, box := range bounds {
		for _, aspect := range []float64{1, 2, 3} {
			for _, north := range []float64{0.5, 1, 2} {
				t.Run(fmt.Sprintf("bounds%d/aspect%g/north%g", bi, aspect, north), func(t *testing.T) {
					p := newGeoProjection(box, "iso-ccw-screen", ProjectionOptions{Aspect: aspect, NorthScale: north})
					rng := rand.New(rand.NewSource(17))
					for i := 0; i < 100; i++ {
						lat := box.MinLat + rng.Float64()*(box.MaxLat-box.MinLat)
						lon := box.MinLon + rng.Float64()*(box.MaxLon-box.MinLon)
						u, v, ok := p.latLonToGridUnit(lat, lon)
						if !ok {
							t.Fatalf("interior point clipped: %g,%g -> %g,%g", lat, lon, u, v)
						}
						gotLat, gotLon, outside := p.gridToLatLon(u, v)
						if outside || math.Abs(gotLat-lat) > 1e-10 || math.Abs(gotLon-lon) > 1e-10 {
							t.Fatalf("round trip: (%g,%g) -> (%g,%g), outside=%v", lat, lon, gotLat, gotLon, outside)
						}
					}
					maxAxis := 0.0
					for _, lat := range []float64{box.MinLat, box.MaxLat} {
						for _, lon := range []float64{box.MinLon, box.MaxLon} {
							u, v, ok := p.latLonToGridUnit(lat, lon)
							if !ok || math.Abs(u) > 0.5+1e-12 || math.Abs(v) > 0.5+1e-12 {
								t.Fatalf("corner clipped: %g,%g -> %g,%g", lat, lon, u, v)
							}
							maxAxis = math.Max(maxAxis, math.Max(math.Abs(u), math.Abs(v)))
						}
					}
					if math.Abs(maxAxis-0.5) > 1e-12 {
						t.Fatalf("bounds do not fill frame: %g", maxAxis)
					}
				})
			}
		}
	}
}
