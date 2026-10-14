package geotrace

import (
	"aoe2kit/pkg/datcache"
	"aoe2kit/pkg/datfile"
	"fmt"
	"math"
	"sort"
)

type DockConstraints struct {
	State           string `json:"state"`
	DATHash         string `json:"dat_sha256,omitempty"`
	Unit            int    `json:"unit"`
	Civ             int    `json:"civ"`
	Table           int    `json:"terrain_table"`
	AllowedTerrains []int  `json:"allowed_terrains,omitempty"`
	FootprintTiles  int    `json:"footprint_tiles,omitempty"`
	Note            string `json:"note"`
}
type ShorePair struct {
	LandTerrain  int  `json:"land_terrain"`
	WaterTerrain int  `json:"water_terrain"`
	Edges        int  `json:"edges"`
	AllowedByDAT bool `json:"allowed_by_dat"`
}
type MapDiagnostics struct {
	SourceLandComponents       int             `json:"source_land_components"`
	LargestLandComponent       int             `json:"largest_land_component"`
	EdgeConnectedLandCells     int             `json:"edge_connected_land_cells"`
	SourceWaterComponents      int             `json:"source_water_components"`
	LandBoundsFill             float64         `json:"land_bounds_fill"`
	ShorePairs                 []ShorePair     `json:"shore_pairs"`
	DATCandidateDockFootprints int             `json:"dat_candidate_dock_footprints"`
	Dock                       DockConstraints `json:"dock"`
}

func loadDockConstraints(path string, p EcologyProfile) (DockConstraints, error) {
	d := DockConstraints{State: "unavailable", Unit: p.DockUnit, Civ: p.DockCiv, Note: "DAT terrain eligibility is necessary but does not prove engine construction behavior. Final dock placement remains an engine check."}
	if path == "" {
		return d, nil
	}
	cache, err := datcache.Open(path)
	if err != nil {
		return d, fmt.Errorf("dock DAT cache: %w", err)
	}
	defer cache.Close()
	rows, _, err := cache.QueryUnits(datcache.UnitQuery{Civ: &p.DockCiv, ID: &p.DockUnit, All: true})
	if err != nil {
		return d, err
	}
	if len(rows) != 1 {
		return d, fmt.Errorf("dock unit %d missing from civilization %d", p.DockUnit, p.DockCiv)
	}
	d.Table = int(rows[0].TerrainTableID)
	d.DATHash = cache.SHA256
	_, tables, err := datfile.ReadTerrainRestrictions(path)
	if err != nil {
		return d, err
	}
	if d.Table < 0 || d.Table >= len(tables) {
		return d, fmt.Errorf("dock terrain table %d out of range", d.Table)
	}
	for _, row := range tables[d.Table].TerrainRows {
		if !finite(float64(row.Passability)) {
			return d, fmt.Errorf("nonfinite dock multiplier for terrain %d", row.Index)
		}
		if row.Passability > 0 {
			d.AllowedTerrains = append(d.AllowedTerrains, row.Index)
		}
	}
	d.FootprintTiles = int(math.Ceil(2 * math.Max(float64(rows[0].ClearanceSizeX), float64(rows[0].ClearanceSizeY))))
	if d.FootprintTiles < 1 || d.FootprintTiles > 16 {
		return d, fmt.Errorf("unsupported dock clearance size %d", d.FootprintTiles)
	}
	d.State = "dat_filtered_candidates"
	return d, nil
}

// Follow the recipe's coast base-terrain paint order, including road overrides.
func coastBaseTerrain(c Cell, r Report) int {
	t := c.TerrainID
	shore := c.ShoreTerrainID
	if shore == 0 {
		shore = 2
	}
	switch c.ShoreCrossfadeBand {
	case "A_over_B", "B", "C_over_B":
		t = shore
	case "A_over_M", "M":
		t = 23
	case "M_over_D", "D":
		t = 57
	}
	if c.TerrainID == 1 && c.Land != nil && !*c.Land && !c.InlandWater && c.DistanceToShoreTiles <= normalizeShoreBandWidth(r.ShoreBandWidth) {
		t = 1
	}
	if c.Roadside {
		t = roadsideTerrainID
	}
	if c.Road {
		t = c.RoadTerrainID
		if t == 0 {
			t = roadTerrainID
		}
	}
	if c.Border {
		t = c.TerrainID
	}
	return t
}

func measureMapDiagnostics(r Report, dock DockConstraints) *MapDiagnostics {
	d := &MapDiagnostics{Dock: dock}
	n := r.GridSize
	if n <= 0 || len(r.Cells) != n*n {
		return d
	}
	land, known, seen := make([]bool, n*n), make([]bool, n*n), make([]bool, n*n)
	minX, minY, maxX, maxY, total := n, n, -1, -1, 0
	for i, c := range r.Cells {
		known[i] = c.Land != nil
		land[i] = known[i] && *c.Land
		if land[i] {
			x, y := i%n, i/n
			minX = minInt(minX, x)
			minY = minInt(minY, y)
			maxX = maxInt(maxX, x)
			maxY = maxInt(maxY, y)
			total++
		}
	}
	if total > 0 {
		d.LandBoundsFill = float64(total) / float64((maxX-minX+1)*(maxY-minY+1))
	}
	neighbors := func(i int, visit func(int)) {
		x, y := i%n, i/n
		if x > 0 {
			visit(i - 1)
		}
		if x < n-1 {
			visit(i + 1)
		}
		if y > 0 {
			visit(i - n)
		}
		if y < n-1 {
			visit(i + n)
		}
	}
	queue := make([]int, 0, n*n)
	for i := range land {
		if seen[i] || !known[i] {
			continue
		}
		queue = append(queue[:0], i)
		seen[i] = true
		edge := false
		for head := 0; head < len(queue); head++ {
			j := queue[head]
			x, y := j%n, j/n
			edge = edge || x == 0 || y == 0 || x == n-1 || y == n-1
			neighbors(j, func(k int) {
				if !seen[k] && known[k] && land[k] == land[i] {
					seen[k] = true
					queue = append(queue, k)
				}
			})
		}
		if land[i] {
			d.SourceLandComponents++
			d.LargestLandComponent = maxInt(d.LargestLandComponent, len(queue))
			if edge {
				d.EdgeConnectedLandCells += len(queue)
			}
		} else {
			d.SourceWaterComponents++
		}
	}
	allowed := map[int]bool{}
	for _, t := range dock.AllowedTerrains {
		allowed[t] = true
	}
	pairs := map[[2]int]int{}
	for i, c := range r.Cells {
		if !known[i] || land[i] {
			continue
		}
		neighbors(i, func(k int) {
			if land[k] {
				pairs[[2]int{coastBaseTerrain(r.Cells[k], r), coastBaseTerrain(c, r)}]++
			}
		})
		if dock.State != "dat_filtered_candidates" || dock.FootprintTiles <= 0 {
			continue
		}
		x0, y0 := i%n-dock.FootprintTiles/2, i/n-dock.FootprintTiles/2
		valid, hasLand, hasWater := true, false, false
		for dy := 0; dy < dock.FootprintTiles && valid; dy++ {
			for dx := 0; dx < dock.FootprintTiles; dx++ {
				x, y := x0+dx, y0+dy
				if x < 0 || y < 0 || x >= n || y >= n {
					valid = false
					break
				}
				k := y*n + x
				if !known[k] || !allowed[coastBaseTerrain(r.Cells[k], r)] {
					valid = false
					break
				}
				hasLand = hasLand || land[k]
				hasWater = hasWater || !land[k]
			}
		}
		if valid && hasLand && hasWater {
			d.DATCandidateDockFootprints++
		}
	}
	for pair, count := range pairs {
		d.ShorePairs = append(d.ShorePairs, ShorePair{LandTerrain: pair[0], WaterTerrain: pair[1], Edges: count, AllowedByDAT: allowed[pair[0]] && allowed[pair[1]]})
	}
	sort.Slice(d.ShorePairs, func(i, j int) bool {
		a, b := d.ShorePairs[i], d.ShorePairs[j]
		if a.LandTerrain != b.LandTerrain {
			return a.LandTerrain < b.LandTerrain
		}
		return a.WaterTerrain < b.WaterTerrain
	})
	return d
}
