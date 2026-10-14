package geotrace

import (
	"crypto/sha256"
	"encoding/csv"
	"fmt"
	"io"
	"math"
	"os"
	"strconv"
	"strings"
)

// USGSWoodyPoint is one approximately 25 km atlas grid point. The release is
// presence/absence data, so it selects a regional tree palette rather than
// claiming an individual tree location.
type USGSWoodyPoint struct {
	Lat     float64
	Lon     float64
	Species []string
}

type usgsWoodyGrid struct {
	Points []USGSWoodyPoint
	bins   map[[2]int][]int
}

type usgsWoodySummary struct {
	PointCount   int
	SpeciesCount int
	SHA256       string
}

func loadUSGSWoody(path string) (*usgsWoodyGrid, usgsWoodySummary, error) {
	return loadUSGSWoodyWithProfile(path, EcologyProfile{})
}

func loadUSGSWoodyWithProfile(path string, profile EcologyProfile) (*usgsWoodyGrid, usgsWoodySummary, error) {
	if path == "" {
		return nil, usgsWoodySummary{}, nil
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, usgsWoodySummary{}, fmt.Errorf("open USGS woody CSV: %w", err)
	}
	defer f.Close()
	hash := sha256.New()
	r := csv.NewReader(io.TeeReader(f, hash))
	header, err := r.Read()
	if err != nil {
		return nil, usgsWoodySummary{}, fmt.Errorf("read USGS woody header: %w", err)
	}
	latIndex, lonIndex := -1, -1
	for i, name := range header {
		switch strings.ToUpper(strings.TrimSpace(name)) {
		case "LATITUDE":
			latIndex = i
		case "LONGITUDE":
			lonIndex = i
		}
	}
	if latIndex < 0 || lonIndex < 0 {
		return nil, usgsWoodySummary{}, fmt.Errorf("USGS woody CSV is missing LATITUDE or LONGITUDE columns")
	}
	grid := &usgsWoodyGrid{bins: map[[2]int][]int{}}
	seenSpecies := map[string]struct{}{}
	line := 1
	for {
		line++
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, usgsWoodySummary{}, fmt.Errorf("read USGS woody CSV line %d: %w", line, err)
		}
		if len(record) <= latIndex || len(record) <= lonIndex {
			return nil, usgsWoodySummary{}, fmt.Errorf("USGS woody CSV line %d has %d fields, want latitude/longitude", line, len(record))
		}
		lat, err := strconv.ParseFloat(strings.TrimSpace(record[latIndex]), 64)
		if err != nil {
			return nil, usgsWoodySummary{}, fmt.Errorf("USGS woody CSV line %d latitude: %w", line, err)
		}
		lon, err := strconv.ParseFloat(strings.TrimSpace(record[lonIndex]), 64)
		if err != nil {
			return nil, usgsWoodySummary{}, fmt.Errorf("USGS woody CSV line %d longitude: %w", line, err)
		}
		if !finite(lat) || !finite(lon) || lat < -90 || lat > 90 || lon < -180 || lon > 180 {
			return nil, usgsWoodySummary{}, fmt.Errorf("USGS woody CSV line %d invalid coordinate", line)
		}
		point := USGSWoodyPoint{Lat: lat, Lon: lon}
		for i := 0; i < len(header) && i < len(record); i++ {
			if i == latIndex || i == lonIndex || strings.TrimSpace(record[i]) != "1" {
				continue
			}
			name := strings.TrimSpace(header[i])
			if usgsWoodyUnitConst(name) == 0 && len(profile.SpeciesUnits[name]) == 0 {
				continue
			}
			point.Species = append(point.Species, name)
			seenSpecies[name] = struct{}{}
		}
		key := [2]int{int(math.Floor(lat)), longitudeBin(lon)}
		grid.bins[key] = append(grid.bins[key], len(grid.Points))
		grid.Points = append(grid.Points, point)
	}
	return grid, usgsWoodySummary{PointCount: len(grid.Points), SpeciesCount: len(seenSpecies), SHA256: fmt.Sprintf("%x", hash.Sum(nil))}, nil
}

// Map only species with a defensible AoE2 visual/resource analogue. Other
// USGS taxa remain available in the source dataset but do not become invented
// game units silently.
func usgsWoodyUnitConst(name string) int {
	name = strings.TrimSpace(name)
	switch {
	case name == "Quercus virginiana":
		return 2567 // live oak
	case strings.HasPrefix(name, "Quercus "):
		return 349 // oak family stand-in
	case strings.HasPrefix(name, "Pinus "):
		return 350 // pine family stand-in
	case name == "Taxodium distichum", name == "Juniperus virginiana":
		return 349 // woody stand-in until a dedicated asset exists
	default:
		return 0
	}
}

func (p EcologyProfile) woodyUnitChoices(species []string) []int {
	var choices []int
	for _, name := range species {
		units := p.SpeciesUnits[name]
		if len(units) == 0 {
			if id := usgsWoodyUnitConst(name); id > 0 {
				units = []int{id}
			}
		}
		for _, id := range units {
			if !member(choices, id) {
				choices = append(choices, id)
			}
		}
	}
	return choices
}

func usgsWoodyUnitChoices(species []string) []int {
	seen := map[int]struct{}{}
	choices := make([]int, 0, len(species))
	for _, name := range species {
		unit := usgsWoodyUnitConst(name)
		if unit == 0 {
			continue
		}
		if _, ok := seen[unit]; ok {
			continue
		}
		seen[unit] = struct{}{}
		choices = append(choices, unit)
	}
	return choices
}

func (grid *usgsWoodyGrid) nearestSpecies(lat, lon float64) []string {
	if grid == nil || len(grid.Points) == 0 || !finite(lat) || !finite(lon) || lat < -90 || lat > 90 {
		return nil
	}
	best := -1
	bestDistance := usgsWoodyMaxDistanceKM
	latRadius := usgsWoodyMaxDistanceKM / earthRadiusKM * 180 / math.Pi
	lonRadius := 180.0
	if math.Abs(lat)+latRadius < 90 {
		lonRadius = math.Asin(math.Sin(latRadius*math.Pi/180)/math.Cos(lat*math.Pi/180)) * 180 / math.Pi
	}
	minLon, maxLon := int(math.Floor(lon-lonRadius)), int(math.Floor(lon+lonRadius))
	if maxLon-minLon >= 360 {
		minLon, maxLon = -180, 179
	}
	for y := int(math.Floor(math.Max(-90, lat-latRadius))); y <= int(math.Floor(math.Min(90, lat+latRadius))); y++ {
		for x := minLon; x <= maxLon; x++ {
			for _, i := range grid.bins[[2]int{y, longitudeBin(float64(x))}] {
				point := grid.Points[i]
				distance := greatCircleKM(lat, lon, point.Lat, point.Lon)
				if distance < bestDistance || (distance == bestDistance && (best < 0 || i < best)) {
					best = i
					bestDistance = distance
				}
			}
		}
	}
	if best < 0 {
		return nil
	}
	return append([]string(nil), grid.Points[best].Species...)
}

// A generous two-cell radius permits coarse atlas sampling but prevents
// extrapolating the North American atlas onto another continent.
const usgsWoodyMaxDistanceKM = 50.0
const earthRadiusKM = 6371.0088

func longitudeBin(lon float64) int {
	bin := int(math.Floor(lon))
	return ((bin+180)%360+360)%360 - 180
}

func greatCircleKM(lat1, lon1, lat2, lon2 float64) float64 {
	a := (lat2 - lat1) * math.Pi / 180
	b := (lon2 - lon1) * math.Pi / 180
	h := math.Sin(a/2)*math.Sin(a/2) + math.Cos(lat1*math.Pi/180)*math.Cos(lat2*math.Pi/180)*math.Sin(b/2)*math.Sin(b/2)
	return 2 * earthRadiusKM * math.Asin(math.Sqrt(math.Min(1, math.Max(0, h))))
}
