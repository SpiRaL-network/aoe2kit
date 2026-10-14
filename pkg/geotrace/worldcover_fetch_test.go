package geotrace

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestWorldCoverCachesOnlyObservedMissingTiles(t *testing.T) {
	calls := 0
	client := &http.Client{Transport: roadTestTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		return &http.Response{StatusCode: 404, Header: http.Header{}, Body: io.NopCloser(strings.NewReader("missing"))}, nil
	})}
	dir := t.TempDir()
	for i := 0; i < 2; i++ {
		s := worldCoverSampler{ctx: context.Background(), provider: "worldcover", cacheDir: dir, client: client}
		got, err := s.sample(25, -85)
		if err != nil || got.Valid || got.MissingTile == "" {
			t.Fatal(got, err)
		}
	}
	if calls != 1 {
		t.Fatalf("missing tile fetched %d times", calls)
	}
}

func TestWorldCoverFootprintFindsOffCentreThinLand(t *testing.T) {
	pixel := 74.0 / 111320
	raw := make([]byte, 21*21)
	for i := range raw {
		raw[i] = 80
	}
	raw[10*21+16] = 10
	tile := worldCoverTile{image: tiffImage{width: 21, height: 21, tileWidth: 21, tileHeight: 21, tileOffsets: []int64{0}, pixelWidth: pixel, pixelHeight: pixel, maxLat: 21 * pixel, maxLon: 21 * pixel}, tileCache: map[int][]byte{0: raw}}
	lat, lon := 10.5*pixel, 10.5*pixel
	narrow, err := tile.sample(lat, lon)
	if err != nil || len(narrow.Histogram) != 1 {
		t.Fatal(narrow, err)
	}
	tile.footprintMeters = 982
	wide, err := tile.sample(lat, lon)
	if err != nil || len(wide.Histogram) != 2 {
		t.Fatal(wide, err)
	}
	m := []bool{false}
	samples := []LandcoverSample{wide}
	d := refineCoastMask(m, samples, "worldcover", 982)
	if !m[0] || d.WaterToLand != 1 || samples[0].CoastalLandFraction <= 0 {
		t.Fatal(m, d, samples)
	}
}

func TestWorldCoverResolutionTracksRequestedScale(t *testing.T) {
	images := []tiffImage{{width: 36000}, {width: 18000}, {width: 9000}, {width: 4500}, {width: 2250}, {width: 1125}}
	for _, tc := range []struct {
		target float64
		want   int
	}{{36, 18000}, {650, 1125}, {5, 36000}} {
		got := chooseWorldCoverResolution(images, 10, tc.target)
		if got.width != tc.want {
			t.Fatalf("target=%g got=%d want=%d", tc.target, got.width, tc.want)
		}
	}
}

func TestWorldCoverRejectsIgnoredRangeAndCachesValidRange(t *testing.T) {
	calls := 0
	valid := false
	source := httpRangeSource{url: "https://example.invalid/tile", cacheDir: t.TempDir(),
		client: &http.Client{Transport: roadTestTransport(func(r *http.Request) (*http.Response, error) {
			calls++
			code := 200
			headers := http.Header{}
			if valid {
				code = 206
				headers.Set("Content-Range", "bytes 10-13/100")
			}
			return &http.Response{StatusCode: code, Header: headers, Body: io.NopCloser(strings.NewReader("data"))}, nil
		})}}
	if _, err := source.readAt(10, 4); err == nil {
		t.Fatal("accepted ignored range")
	}
	valid = true
	if _, err := source.readAt(10, 4); err != nil {
		t.Fatal(err)
	}
	before := calls
	data, err := source.readAt(10, 4)
	if err != nil || string(data) != "data" || calls != before {
		t.Fatal(fmt.Sprint(string(data), calls, err))
	}
}
