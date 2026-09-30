package geotrace

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const roadTerrainID = 75
const roadsideTerrainID = 6
const borderTerrainID = 129

type roadStats struct {
	Provider       string
	WayCount       int
	VertexCountRaw int
	VertexCount    int
	SimplifyTiles  float64
	CellCount      int
	Refs           []string
}

type roadWay struct {
	Tags     map[string]string
	Geometry []point
}

func applyRoadLayer(report *Report, opts Options) (roadStats, error) {
	provider := strings.ToLower(strings.TrimSpace(opts.RoadsProvider))
	if provider == "" && opts.RoadsPath != "" {
		provider = "overpass-json"
	}
	if provider == "" {
		return roadStats{}, nil
	}
	ways, effectiveProvider, err := loadRoadWays(provider, opts)
	if err != nil {
		return roadStats{}, err
	}
	if len(ways) == 0 {
		report.Warnings = append(report.Warnings, fmt.Sprintf("road provider %s returned no ways", effectiveProvider))
		return roadStats{Provider: effectiveProvider}, nil
	}
	rawVertexCount := roadVertexCount(ways)
	simplifyTiles := roadSimplifyTiles(opts)
	ways = simplifyRoadWaysForViewport(ways, report.BBox, report.GridSize, simplifyTiles)
	vertexCount := roadVertexCount(ways)
	n := report.GridSize
	roadMask := make(map[int]bool)
	refSet := map[string]bool{}
	for _, way := range ways {
		for _, key := range []string{"ref", "name"} {
			if value := strings.TrimSpace(way.Tags[key]); value != "" {
				refSet[value] = true
			}
		}
		for i := 1; i < len(way.Geometry); i++ {
			for _, idx := range geoSegmentIndices(report.BBox, n, report.Orient, way.Geometry[i-1], way.Geometry[i]) {
				if idx < 0 || idx >= len(report.Cells) {
					continue
				}
				if roadCellAllowed(report.Cells[idx]) {
					roadMask[idx] = true
				}
			}
		}
	}
	for idx := range roadMask {
		report.Cells[idx].Road = true
	}
	refs := make([]string, 0, len(refSet))
	for ref := range refSet {
		refs = append(refs, ref)
	}
	sort.Strings(refs)
	return roadStats{
		Provider:       effectiveProvider,
		WayCount:       len(ways),
		VertexCountRaw: rawVertexCount,
		VertexCount:    vertexCount,
		SimplifyTiles:  simplifyTiles,
		CellCount:      len(roadMask),
		Refs:           refs,
	}, nil
}

func roadVertexCount(ways []roadWay) int {
	total := 0
	for _, way := range ways {
		total += len(way.Geometry)
	}
	return total
}

func roadSimplifyTiles(opts Options) float64 {
	if opts.RoadSimplifyTiles > 0 {
		return opts.RoadSimplifyTiles
	}
	return 1.25
}

func simplifyRoadWaysForViewport(ways []roadWay, b BBox, gridSize int, simplifyTiles float64) []roadWay {
	if simplifyTiles <= 0 {
		return ways
	}
	tolerance := roadSimplifyTolerance(gridSize, simplifyTiles)
	if tolerance <= 0 {
		return ways
	}
	out := make([]roadWay, 0, len(ways))
	for _, way := range ways {
		geom := simplifyRoadGeometry(way.Geometry, b, tolerance)
		if len(geom) < 2 {
			continue
		}
		out = append(out, roadWay{Tags: way.Tags, Geometry: geom})
	}
	return out
}

func roadSimplifyTolerance(gridSize int, simplifyTiles float64) float64 {
	if gridSize <= 0 {
		return 0
	}
	return simplifyTiles / float64(gridSize)
}

func simplifyRoadGeometry(points []point, b BBox, tolerance float64) []point {
	if len(points) <= 2 || tolerance <= 0 {
		return points
	}
	keep := make([]bool, len(points))
	keep[0] = true
	keep[len(points)-1] = true
	rdpKeep(points, b, 0, len(points)-1, tolerance*tolerance, keep)
	out := make([]point, 0, len(points))
	var last point
	haveLast := false
	for i, p := range points {
		if !keep[i] {
			continue
		}
		if haveLast && sameRoadPoint(last, p) {
			continue
		}
		out = append(out, p)
		last = p
		haveLast = true
	}
	return out
}

func rdpKeep(points []point, b BBox, start, end int, toleranceSq float64, keep []bool) {
	if end <= start+1 {
		return
	}
	ax, ay := normalizedRoadPoint(points[start], b)
	zx, zy := normalizedRoadPoint(points[end], b)
	bestIdx := -1
	bestDist := -1.0
	for i := start + 1; i < end; i++ {
		px, py := normalizedRoadPoint(points[i], b)
		dist := pointSegmentDistanceSq(px, py, ax, ay, zx, zy)
		if dist > bestDist {
			bestDist = dist
			bestIdx = i
		}
	}
	if bestIdx >= 0 && bestDist > toleranceSq {
		keep[bestIdx] = true
		rdpKeep(points, b, start, bestIdx, toleranceSq, keep)
		rdpKeep(points, b, bestIdx, end, toleranceSq, keep)
	}
}

func normalizedRoadPoint(p point, b BBox) (float64, float64) {
	lonSpan := math.Max(1e-9, b.MaxLon-b.MinLon)
	latSpan := math.Max(1e-9, b.MaxLat-b.MinLat)
	return (p.Lon - b.MinLon) / lonSpan, (p.Lat - b.MinLat) / latSpan
}

func pointSegmentDistanceSq(px, py, ax, ay, bx, by float64) float64 {
	dx := bx - ax
	dy := by - ay
	if dx == 0 && dy == 0 {
		return squaredDistance(px, py, ax, ay)
	}
	t := ((px-ax)*dx + (py-ay)*dy) / (dx*dx + dy*dy)
	if t < 0 {
		t = 0
	} else if t > 1 {
		t = 1
	}
	return squaredDistance(px, py, ax+t*dx, ay+t*dy)
}

func squaredDistance(ax, ay, bx, by float64) float64 {
	dx := ax - bx
	dy := ay - by
	return dx*dx + dy*dy
}

func sameRoadPoint(a, b point) bool {
	return a.Lat == b.Lat && a.Lon == b.Lon
}

func applyAdministrativeBorderLayer(report *Report, opts Options) (int, error) {
	if strings.TrimSpace(opts.RoadsProvider) == "" && strings.TrimSpace(opts.RoadsPath) == "" {
		return 0, nil
	}
	ways, err := loadAdminBorderWays(opts)
	if err != nil {
		report.Warnings = append(report.Warnings, "admin border layer skipped: "+err.Error())
		return 0, nil
	}
	for _, way := range ways {
		for i := 1; i < len(way.Geometry); i++ {
			for _, idx := range geoSegmentIndices(report.BBox, report.GridSize, report.Orient, way.Geometry[i-1], way.Geometry[i]) {
				if idx < 0 || idx >= len(report.Cells) {
					continue
				}
				markBorderCell(report, idx, 1)
			}
		}
	}
	return len(ways), nil
}

func geoSegmentIndices(b BBox, n int, orient string, a, z point) []int {
	if n <= 0 {
		return nil
	}
	width := math.Max(1e-9, b.MaxLon-b.MinLon)
	height := math.Max(1e-9, b.MaxLat-b.MinLat)
	normLen := math.Hypot((z.Lon-a.Lon)/width, (z.Lat-a.Lat)/height)
	steps := maxInt(1, int(math.Ceil(normLen*float64(n)*4)))
	seen := map[int]bool{}
	var out []int
	prevX, prevY := 0, 0
	havePrev := false
	for s := 0; s <= steps; s++ {
		t := float64(s) / float64(steps)
		lat := a.Lat + (z.Lat-a.Lat)*t
		lon := a.Lon + (z.Lon-a.Lon)*t
		x, y, ok := gridCellForLatLon(b, n, orient, lat, lon)
		if !ok {
			havePrev = false
			continue
		}
		var cells []int
		if havePrev {
			cells = bresenhamIndices(prevX, prevY, x, y, n)
		} else {
			cells = []int{y*n + x}
		}
		for _, idx := range cells {
			if !seen[idx] {
				seen[idx] = true
				out = append(out, idx)
			}
		}
		prevX, prevY = x, y
		havePrev = true
	}
	return out
}

func markBorderCell(report *Report, idx, radius int) {
	if report == nil || report.GridSize <= 0 || idx < 0 || idx >= len(report.Cells) {
		return
	}
	n := report.GridSize
	x := idx % n
	y := idx / n
	for dy := -radius; dy <= radius; dy++ {
		for dx := -radius; dx <= radius; dx++ {
			xx := x + dx
			yy := y + dy
			if xx < 0 || yy < 0 || xx >= n || yy >= n {
				continue
			}
			j := yy*n + xx
			if borderCellAllowed(report.Cells[j]) {
				report.Cells[j].Border = true
			}
		}
	}
}

func borderCellAllowed(cell Cell) bool {
	if cell.Land == nil || !*cell.Land {
		return false
	}
	return cell.DistanceToShoreTiles > 1.1
}

func applyDerivedFeatureMasks(report *Report) {
	if report == nil || report.GridSize <= 0 || len(report.Cells) != report.GridSize*report.GridSize {
		return
	}
	addFloridaKeysCauseway(report)
	markRoadsideCells(report)
	markWaterBufferCells(report)
	report.RoadCells = 0
	report.RoadsideCells = 0
	report.WaterBufferCells = 0
	report.KeyWestCausewayCells = 0
	report.BorderCells = 0
	for i := range report.Cells {
		if report.Cells[i].Road {
			report.RoadCells++
		}
		if report.Cells[i].Roadside {
			report.RoadsideCells++
		}
		if report.Cells[i].WaterBuffer {
			report.WaterBufferCells++
		}
		if report.Cells[i].KeyWestCauseway {
			report.KeyWestCausewayCells++
		}
		if report.Cells[i].Border {
			report.BorderCells++
		}
	}
}

func addFloridaKeysCauseway(report *Report) {
	if !bboxContains(report.BBox, 24.56, -81.78) || !bboxContains(report.BBox, 25.47, -80.45) {
		return
	}
	route := []point{
		{Lat: 25.47, Lon: -80.45},
		{Lat: 25.10, Lon: -80.45},
		{Lat: 24.92, Lon: -80.64},
		{Lat: 24.71, Lon: -81.09},
		{Lat: 24.67, Lon: -81.35},
		{Lat: 24.56, Lon: -81.78},
	}
	for i := 1; i < len(route); i++ {
		x0, y0, ok0 := gridCellForLatLon(report.BBox, report.GridSize, report.Orient, route[i-1].Lat, route[i-1].Lon)
		x1, y1, ok1 := gridCellForLatLon(report.BBox, report.GridSize, report.Orient, route[i].Lat, route[i].Lon)
		if !ok0 || !ok1 {
			continue
		}
		for _, idx := range bresenhamIndices(x0, y0, x1, y1, report.GridSize) {
			if idx < 0 || idx >= len(report.Cells) {
				continue
			}
			report.Cells[idx].Road = true
			report.Cells[idx].KeyWestCauseway = true
		}
	}
}

func addFloridaGeorgiaBorder(report *Report) {
	if report.BBox.MinLat > 31.05 || report.BBox.MaxLat < 30.25 || report.BBox.MinLon > -81.1 || report.BBox.MaxLon < -87.7 {
		return
	}
	line := floridaSharedStateBorder()
	n := report.GridSize
	for i := 1; i < len(line); i++ {
		x0, y0, ok0 := gridCellForLatLon(report.BBox, report.GridSize, report.Orient, line[i-1].Lat, line[i-1].Lon)
		x1, y1, ok1 := gridCellForLatLon(report.BBox, report.GridSize, report.Orient, line[i].Lat, line[i].Lon)
		if !ok0 || !ok1 {
			continue
		}
		for _, idx := range bresenhamIndices(x0, y0, x1, y1, report.GridSize) {
			if idx < 0 || idx >= len(report.Cells) {
				continue
			}
			x := idx % n
			y := idx / n
			for dy := -1; dy <= 1; dy++ {
				for dx := -1; dx <= 1; dx++ {
					xx := x + dx
					yy := y + dy
					if xx < 0 || yy < 0 || xx >= n || yy >= n {
						continue
					}
					j := yy*n + xx
					if report.Cells[j].Land != nil && *report.Cells[j].Land {
						report.Cells[j].Border = true
					}
				}
			}
		}
	}
}

func floridaSharedStateBorder() []point {
	return []point{
		{Lon: -87.447336, Lat: 30.313137},
		{Lon: -87.426829, Lat: 30.374579},
		{Lon: -87.378774, Lat: 30.417467},
		{Lon: -87.387074, Lat: 30.463284},
		{Lon: -87.439032, Lat: 30.528666},
		{Lon: -87.411224, Lat: 30.610697},
		{Lon: -87.405194, Lat: 30.67017},
		{Lon: -87.525214, Lat: 30.749394},
		{Lon: -87.625311, Lat: 30.865214},
		{Lon: -87.599945, Lat: 30.934086},
		{Lon: -87.593646, Lat: 30.999711},
		{Lon: -87.23078, Lat: 30.999784},
		{Lon: -86.754169, Lat: 30.999833},
		{Lon: -86.277557, Lat: 30.999906},
		{Lon: -85.800946, Lat: 31.000004},
		{Lon: -85.324335, Lat: 31.000077},
		{Lon: -85.006586, Lat: 31.000102},
		{Lon: -84.966205, Lat: 30.922709},
		{Lon: -84.920477, Lat: 30.760087},
		{Lon: -84.780072, Lat: 30.715287},
		{Lon: -84.48288, Lat: 30.697636},
		{Lon: -84.114667, Lat: 30.675834},
		{Lon: -83.682709, Lat: 30.650199},
		{Lon: -83.235199, Lat: 30.623612},
		{Lon: -82.037372, Lat: 30.62393},
		{Lon: -82.047308, Lat: 30.650053},
		{Lon: -82.038837, Lat: 30.702299},
		{Lon: -82.034955, Lat: 30.749394},
		{Lon: -81.998212, Lat: 30.782475},
		{Lon: -81.953387, Lat: 30.82061},
		{Lon: -81.886591, Lat: 30.80518},
		{Lon: -81.781805, Lat: 30.766191},
		{Lon: -81.708759, Lat: 30.741069},
		{Lon: -81.608588, Lat: 30.71746},
		{Lon: -81.525262, Lat: 30.718144},
		{Lon: -81.46935, Lat: 30.69717},
		{Lon: -81.437815, Lat: 30.637763},
		{Lon: -81.489003, Lat: 30.620998},
	}
}

func markRoadsideCells(report *Report) {
	n := report.GridSize
	for i, cell := range report.Cells {
		if !cell.Road {
			continue
		}
		x := i % n
		y := i / n
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				if dx == 0 && dy == 0 {
					continue
				}
				xx := x + dx
				yy := y + dy
				if xx < 0 || yy < 0 || xx >= n || yy >= n {
					continue
				}
				idx := yy*n + xx
				neighbor := report.Cells[idx]
				if neighbor.Road || neighbor.Border || neighbor.Land == nil || !*neighbor.Land {
					continue
				}
				report.Cells[idx].Roadside = true
			}
		}
	}
}

func markWaterBufferCells(report *Report) {
	n := report.GridSize
	openWater := make([]bool, len(report.Cells))
	for i, cell := range report.Cells {
		openWater[i] = (cell.Land == nil || !*cell.Land) && !cell.InlandWater
	}
	waterAdjacent := make([]bool, len(report.Cells))
	for i, cell := range report.Cells {
		if cell.Land == nil || !*cell.Land || cell.Road || cell.Roadside || cell.Border {
			continue
		}
		x := i % n
		y := i / n
		for dy := -2; dy <= 2; dy++ {
			for dx := -2; dx <= 2; dx++ {
				if dx == 0 && dy == 0 {
					continue
				}
				xx := x + dx
				yy := y + dy
				if xx < 0 || yy < 0 || xx >= n || yy >= n {
					continue
				}
				if openWater[yy*n+xx] {
					report.Cells[i].WaterBuffer = true
					waterAdjacent[i] = true
					dy = 3
					break
				}
			}
		}
	}
	for i, cell := range report.Cells {
		if !cell.Roadside || cell.Border {
			continue
		}
		x := i % n
		y := i / n
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				xx := x + dx
				yy := y + dy
				if xx < 0 || yy < 0 || xx >= n || yy >= n {
					continue
				}
				if waterAdjacent[yy*n+xx] || openWater[yy*n+xx] {
					report.Cells[i].WaterBuffer = true
					return
				}
			}
		}
	}
}

func bboxContains(b BBox, lat, lon float64) bool {
	return lat >= b.MinLat && lat <= b.MaxLat && lon >= b.MinLon && lon <= b.MaxLon
}

func loadRoadWays(provider string, opts Options) ([]roadWay, string, error) {
	if opts.RoadsPath != "" {
		data, err := os.ReadFile(opts.RoadsPath)
		if err != nil {
			return nil, "", err
		}
		ways, err := parseRoadFile(data)
		if err != nil {
			return nil, "", err
		}
		ways = filterRoadWaysForProvider(ways, provider)
		return ways, "file:" + opts.RoadsPath, nil
	}
	switch provider {
	case "osm-interstate", "osm-motorway", "osm-major":
		data, err := fetchOverpassRoads(provider, opts)
		if err != nil {
			return nil, "", err
		}
		ways, err := parseOverpassRoadWays(data)
		return ways, provider, err
	case "overpass-json":
		return nil, "", fmt.Errorf("overpass-json roads require --roads-file")
	case "nhpn-interstate", "nhs-interstate", "federal-interstate":
		return nil, "", fmt.Errorf("%s roads require --roads-file with a federal GeoJSON export", provider)
	default:
		return nil, "", fmt.Errorf("unsupported roads provider %q", provider)
	}
}

func filterRoadWaysForProvider(ways []roadWay, provider string) []roadWay {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "osm-interstate", "nhpn-interstate", "nhs-interstate", "federal-interstate":
		out := make([]roadWay, 0, len(ways))
		for _, way := range ways {
			if roadWayLooksInterstate(way) {
				out = append(out, way)
			}
		}
		return out
	default:
		return ways
	}
}

func roadWayLooksInterstate(way roadWay) bool {
	for _, key := range []string{"ref", "name", "route", "route_id", "routeid", "route_num", "sign1", "signt1", "class", "nhs", "nhsdesg"} {
		value := strings.ToUpper(strings.TrimSpace(way.Tags[key]))
		if value == "" {
			continue
		}
		if value == "I" || strings.Contains(value, "INTERSTATE") || strings.Contains(value, "I-") || strings.Contains(value, "I ") || strings.HasPrefix(value, "I") && len(value) > 1 && value[1] >= '0' && value[1] <= '9' {
			return true
		}
	}
	return false
}

func fetchOverpassRoads(provider string, opts Options) ([]byte, error) {
	cachePath := ""
	if opts.RoadsCacheDir != "" {
		if err := os.MkdirAll(opts.RoadsCacheDir, 0755); err != nil {
			return nil, err
		}
		key := fmt.Sprintf("%s|%.6f|%.6f|%.6f|%.6f", provider, opts.BBox.MinLat, opts.BBox.MinLon, opts.BBox.MaxLat, opts.BBox.MaxLon)
		sum := sha256.Sum256([]byte(key))
		cachePath = filepath.Join(opts.RoadsCacheDir, "roads_"+hex.EncodeToString(sum[:8])+".overpass.json")
		if data, err := os.ReadFile(cachePath); err == nil && len(data) > 0 {
			if overpassPayloadUsable(data) {
				return data, nil
			}
		}
	}
	regex := "^(motorway)$"
	if provider == "osm-major" {
		regex = "^(motorway|trunk|primary)$"
	}
	refClause := ""
	if provider == "osm-interstate" {
		refClause = `["ref"~"(^|;| )I[- ]?[0-9]"]`
	}
	query := fmt.Sprintf(`[out:json][timeout:60];
(
  way["highway"~%q]%s(%.7f,%.7f,%.7f,%.7f);
);
out geom tags;`, regex, refClause, opts.BBox.MinLat, opts.BBox.MinLon, opts.BBox.MaxLat, opts.BBox.MaxLon)
	var data []byte
	var err error
	if shouldChunkOverpass(opts.BBox) {
		data, err = fetchOverpassRoadChunks(provider, regex, opts)
	} else {
		data, err = fetchOverpassQuery(opts.Client, query)
	}
	if err != nil {
		return nil, err
	}
	if cachePath != "" {
		if err := os.WriteFile(cachePath, data, 0644); err != nil {
			return nil, err
		}
	}
	return data, nil
}

func overpassPayloadUsable(data []byte) bool {
	if bytes.Contains(data, []byte(`"remark"`)) && bytes.Contains(data, []byte("runtime error")) {
		return false
	}
	ways, err := parseOverpassRoadWays(data)
	return err == nil && len(ways) > 0
}

func loadAdminBorderWays(opts Options) ([]roadWay, error) {
	data, err := fetchOverpassAdminBorders(opts)
	if err != nil {
		return nil, err
	}
	return parseOverpassRoadWays(data)
}

func fetchOverpassAdminBorders(opts Options) ([]byte, error) {
	cachePath := ""
	if opts.RoadsCacheDir != "" {
		if err := os.MkdirAll(opts.RoadsCacheDir, 0755); err != nil {
			return nil, err
		}
		key := fmt.Sprintf("admin4-relways|%.6f|%.6f|%.6f|%.6f", opts.BBox.MinLat, opts.BBox.MinLon, opts.BBox.MaxLat, opts.BBox.MaxLon)
		sum := sha256.Sum256([]byte(key))
		cachePath = filepath.Join(opts.RoadsCacheDir, "admin4_relways_"+hex.EncodeToString(sum[:8])+".overpass.json")
		if data, err := os.ReadFile(cachePath); err == nil && len(data) > 0 {
			return data, nil
		}
	}
	query := fmt.Sprintf(`[out:json][timeout:60];
(
  way["boundary"="administrative"]["admin_level"="4"](%.7f,%.7f,%.7f,%.7f);
  relation["boundary"="administrative"]["admin_level"="4"](%.7f,%.7f,%.7f,%.7f);
);
(._;>;);
out geom tags;`,
		opts.BBox.MinLat, opts.BBox.MinLon, opts.BBox.MaxLat, opts.BBox.MaxLon,
		opts.BBox.MinLat, opts.BBox.MinLon, opts.BBox.MaxLat, opts.BBox.MaxLon)
	data, err := fetchOverpassQuery(opts.Client, query)
	if err != nil {
		return nil, err
	}
	if cachePath != "" {
		if err := os.WriteFile(cachePath, data, 0644); err != nil {
			return nil, err
		}
	}
	return data, nil
}

func fetchOverpassRoadChunks(provider, regex string, opts Options) ([]byte, error) {
	chunks := 6
	if provider == "osm-major" {
		chunks = 8
	}
	type overpassRoot struct {
		Elements []struct {
			ID  int64           `json:"id"`
			Raw json.RawMessage `json:"-"`
		} `json:"elements"`
	}
	seen := map[string]bool{}
	var merged struct {
		Elements []json.RawMessage `json:"elements"`
	}
	for latPart := 0; latPart < chunks; latPart++ {
		minLat := opts.BBox.MinLat + (opts.BBox.MaxLat-opts.BBox.MinLat)*float64(latPart)/float64(chunks)
		maxLat := opts.BBox.MinLat + (opts.BBox.MaxLat-opts.BBox.MinLat)*float64(latPart+1)/float64(chunks)
		for lonPart := 0; lonPart < chunks; lonPart++ {
			minLon := opts.BBox.MinLon + (opts.BBox.MaxLon-opts.BBox.MinLon)*float64(lonPart)/float64(chunks)
			maxLon := opts.BBox.MinLon + (opts.BBox.MaxLon-opts.BBox.MinLon)*float64(lonPart+1)/float64(chunks)
			refClause := ""
			if provider == "osm-interstate" {
				refClause = `["ref"~"(^|;| )I[- ]?[0-9]"]`
			}
			query := fmt.Sprintf(`[out:json][timeout:60];
(
  way["highway"~%q]%s(%.7f,%.7f,%.7f,%.7f);
);
out geom tags;`, regex, refClause, minLat, minLon, maxLat, maxLon)
			data, err := fetchOverpassQuery(opts.Client, query)
			if err != nil {
				return nil, err
			}
			var rawRoot struct {
				Elements []json.RawMessage `json:"elements"`
			}
			if err := json.Unmarshal(data, &rawRoot); err != nil {
				return nil, err
			}
			var root overpassRoot
			if err := json.Unmarshal(data, &root); err != nil {
				return nil, err
			}
			for i, element := range root.Elements {
				key := fmt.Sprintf("id:%d", element.ID)
				if element.ID == 0 {
					key = string(rawRoot.Elements[i])
				}
				if seen[key] {
					continue
				}
				seen[key] = true
				merged.Elements = append(merged.Elements, rawRoot.Elements[i])
			}
		}
	}
	return json.Marshal(merged)
}

func shouldChunkOverpass(b BBox) bool {
	return math.Abs(b.MaxLat-b.MinLat)*math.Abs(b.MaxLon-b.MinLon) > 25
}

func fetchOverpassQuery(client *http.Client, query string) ([]byte, error) {
	if client == nil {
		client = &http.Client{Timeout: 90 * time.Second}
	}
	form := url.Values{"data": []string{query}}
	endpoints := []string{
		"https://overpass-api.de/api/interpreter",
		"https://overpass.kumi.systems/api/interpreter",
		"https://overpass.openstreetmap.ru/api/interpreter",
	}
	var lastErr error
	for _, endpoint := range endpoints {
		for attempt := 0; attempt < 2; attempt++ {
			req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewBufferString(form.Encode()))
			if err != nil {
				return nil, err
			}
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("User-Agent", "AoE2Kit geotrace road layer (https://github.com/chrae/aoe2kit)")
			resp, err := client.Do(req)
			if err != nil {
				lastErr = err
				continue
			}
			data, readErr := io.ReadAll(io.LimitReader(resp.Body, 64<<20))
			closeErr := resp.Body.Close()
			if readErr != nil {
				lastErr = readErr
				continue
			}
			if closeErr != nil {
				lastErr = closeErr
				continue
			}
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				return data, nil
			}
			lastErr = fmt.Errorf("Overpass %s status %d: %s", endpoint, resp.StatusCode, strings.TrimSpace(string(data[:minInt(len(data), 500)])))
		}
	}
	return nil, lastErr
}

func parseRoadFile(data []byte) ([]roadWay, error) {
	if ways, err := parseOverpassRoadWays(data); err == nil && len(ways) > 0 {
		return ways, nil
	}
	return parseGeoJSONRoadWays(data)
}

func parseOverpassRoadWays(data []byte) ([]roadWay, error) {
	var root struct {
		Elements []struct {
			Type     string            `json:"type"`
			Tags     map[string]string `json:"tags"`
			Geometry []struct {
				Lat float64 `json:"lat"`
				Lon float64 `json:"lon"`
			} `json:"geometry"`
		} `json:"elements"`
	}
	if err := json.Unmarshal(data, &root); err != nil {
		return nil, err
	}
	var ways []roadWay
	for _, element := range root.Elements {
		if element.Type != "way" || len(element.Geometry) < 2 {
			continue
		}
		way := roadWay{Tags: element.Tags}
		for _, p := range element.Geometry {
			way.Geometry = append(way.Geometry, point{Lat: p.Lat, Lon: p.Lon})
		}
		ways = append(ways, way)
	}
	return ways, nil
}

func parseGeoJSONRoadWays(data []byte) ([]roadWay, error) {
	var root struct {
		Type        string          `json:"type"`
		Features    []geoJSONRoad   `json:"features"`
		Geometry    *geoJSONGeom    `json:"geometry"`
		Properties  map[string]any  `json:"properties"`
		Coordinates json.RawMessage `json:"coordinates"`
	}
	if err := json.Unmarshal(data, &root); err != nil {
		return nil, err
	}
	var ways []roadWay
	addGeom := func(geom *geoJSONGeom, props map[string]any) error {
		if geom == nil {
			return nil
		}
		tags := geoJSONRoadTags(props)
		switch geom.Type {
		case "LineString":
			line, err := parseGeoJSONLine(geom.Coordinates)
			if err != nil {
				return err
			}
			if len(line) >= 2 {
				ways = append(ways, roadWay{Tags: tags, Geometry: line})
			}
		case "MultiLineString":
			var raw []json.RawMessage
			if err := json.Unmarshal(geom.Coordinates, &raw); err != nil {
				return err
			}
			for _, lineRaw := range raw {
				line, err := parseGeoJSONLine(lineRaw)
				if err != nil {
					return err
				}
				if len(line) >= 2 {
					ways = append(ways, roadWay{Tags: tags, Geometry: line})
				}
			}
		}
		return nil
	}
	switch root.Type {
	case "FeatureCollection":
		for _, feature := range root.Features {
			if err := addGeom(feature.Geometry, feature.Properties); err != nil {
				return nil, err
			}
		}
	case "Feature":
		if err := addGeom(root.Geometry, root.Properties); err != nil {
			return nil, err
		}
	case "LineString", "MultiLineString":
		if err := addGeom(&geoJSONGeom{Type: root.Type, Coordinates: root.Coordinates}, nil); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported road file JSON type %q", root.Type)
	}
	return ways, nil
}

func geoJSONRoadTags(props map[string]any) map[string]string {
	tags := map[string]string{}
	for key, value := range props {
		stringValue := roadPropertyString(value)
		if stringValue == "" {
			continue
		}
		switch strings.ToLower(strings.TrimSpace(key)) {
		case "ref", "route", "route_num", "route_id", "routeid", "rte_nm", "rtename", "stateroute", "sign1", "signt1":
			if tags["ref"] == "" {
				tags["ref"] = stringValue
			}
			tags[strings.ToLower(strings.TrimSpace(key))] = stringValue
		case "name", "fullname", "route_name", "routename", "roadname":
			if tags["name"] == "" {
				tags["name"] = stringValue
			}
			tags[strings.ToLower(strings.TrimSpace(key))] = stringValue
		case "highway", "class", "f_system", "nhs", "nhsdesg":
			tags[strings.ToLower(strings.TrimSpace(key))] = stringValue
		}
	}
	return tags
}

func roadPropertyString(value any) string {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case float64:
		if math.Trunc(v) == v {
			return fmt.Sprintf("%.0f", v)
		}
		return fmt.Sprintf("%g", v)
	case int:
		return fmt.Sprintf("%d", v)
	default:
		return ""
	}
}

type geoJSONRoad struct {
	Properties map[string]any `json:"properties"`
	Geometry   *geoJSONGeom   `json:"geometry"`
}

type geoJSONGeom struct {
	Type        string          `json:"type"`
	Coordinates json.RawMessage `json:"coordinates"`
}

func parseGeoJSONLine(data json.RawMessage) ([]point, error) {
	var coords [][]float64
	if err := json.Unmarshal(data, &coords); err != nil {
		return nil, err
	}
	out := make([]point, 0, len(coords))
	for _, coord := range coords {
		if len(coord) < 2 {
			continue
		}
		out = append(out, point{Lon: coord[0], Lat: coord[1]})
	}
	return out, nil
}

func gridCellForLatLon(b BBox, n int, orient string, lat, lon float64) (int, int, bool) {
	if n <= 0 {
		return 0, 0, false
	}
	latSpan := b.MaxLat - b.MinLat
	lonSpan := b.MaxLon - b.MinLon
	if latSpan <= 0 || lonSpan <= 0 {
		return 0, 0, false
	}
	if orient == "diamond" {
		latT := (b.MaxLat - lat) / latSpan
		sum := int(math.Round(latT * float64(2*(n-1))))
		xMin, xMax, length := diagonalRange(n, sum)
		if length <= 0 {
			return 0, 0, false
		}
		lonT := (lon - b.MinLon) / lonSpan
		x := xMin
		if length > 1 {
			x = int(math.Round(float64(xMin) + lonT*float64(xMax-xMin)))
		}
		y := sum - x
		return x, y, x >= 0 && y >= 0 && x < n && y < n
	}
	proj := newGeoProjection(b, orient)
	u, v, ok := proj.latLonToGridUnit(lat, lon)
	if !ok {
		return 0, 0, false
	}
	x := int(math.Round((u + 0.5) * float64(n-1)))
	y := int(math.Round((v + 0.5) * float64(n-1)))
	return x, y, x >= 0 && y >= 0 && x < n && y < n
}

func roadCellAllowed(cell Cell) bool {
	if cell.Land != nil && *cell.Land {
		return true
	}
	if cell.InlandWater {
		return true
	}
	return cell.DistanceToShoreTiles <= 3
}

func bresenhamIndices(x0, y0, x1, y1, n int) []int {
	var out []int
	dx := absInt(x1 - x0)
	dy := -absInt(y1 - y0)
	sx := -1
	if x0 < x1 {
		sx = 1
	}
	sy := -1
	if y0 < y1 {
		sy = 1
	}
	err := dx + dy
	for {
		if x0 >= 0 && y0 >= 0 && x0 < n && y0 < n {
			out = append(out, y0*n+x0)
		}
		if x0 == x1 && y0 == y1 {
			break
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x0 += sx
		}
		if e2 <= dx {
			err += dx
			y0 += sy
		}
	}
	return out
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
