package geotrace

import (
	"fmt"
	"math"

	"aoe2kit/pkg/datcache"
	"aoe2kit/pkg/datfile"
	"aoe2kit/pkg/scenario"
)

// PlacementAudit describes a static DAT/terrain check, not an engine pathing test.
type PlacementAudit struct {
	State                            string      `json:"state"`
	DATHash                          string      `json:"dat_sha256,omitempty"`
	Checked                          int         `json:"checked_objects"`
	FloraTerrain                     int         `json:"flora_invalid_terrain"`
	FaunaTerrain                     int         `json:"fauna_invalid_terrain"`
	FaunaBlockers                    int         `json:"fauna_overlapping_blockers"`
	Removed                          int         `json:"sweep_removed_objects"`
	Rejected                         int         `json:"rejected_candidates"`
	UnitCounts                       map[int]int `json:"unit_counts,omitempty"`
	InvalidByUnit                    map[int]int `json:"invalid_by_unit,omitempty"`
	Deferred                         int         `json:"deferred_operations,omitempty"`
	ResourceTerrain                  int         `json:"resource_invalid_terrain"`
	ResourceBlockers                 int         `json:"resource_overlapping_blockers"`
	BuildingTerrain                  int         `json:"building_invalid_terrain"`
	BuildingBlockers                 int         `json:"building_overlapping_blockers"`
	BuildingConnectivity             string      `json:"building_static_connectivity,omitempty"`
	LandComponentsBefore             int         `json:"land_components_before_buildings,omitempty"`
	LandComponentsAfter              int         `json:"land_components_after_buildings,omitempty"`
	BuildingsRejectedForConnectivity int         `json:"buildings_rejected_for_connectivity,omitempty"`
	FloraAmphibiousObjects           int         `json:"flora_amphibious_objects,omitempty"`
}

type placementObject struct {
	scenario.UnitSummary
	player int
	row    datcache.UnitRow
	role   string
}

func (o placementObject) bounds() (int, int, int, int) {
	rx, ry := math.Abs(float64(o.row.CollisionSizeX)), math.Abs(float64(o.row.CollisionSizeY))
	return int(math.Floor(o.X - rx)), int(math.Floor(o.Y - ry)), int(math.Floor(o.X + rx - 1e-7)), int(math.Floor(o.Y + ry - 1e-7))
}

func placementTiles(o placementObject, visit func(int, int) bool) bool {
	x1, y1, x2, y2 := o.bounds()
	x2 = max(x1, x2)
	y2 = max(y1, y2)
	for y := y1; y <= y2; y++ {
		for x := x1; x <= x2; x++ {
			if !visit(x, y) {
				return false
			}
		}
	}
	return true
}

func placementRoles(p EcologyProfile) map[int]string {
	m := map[int]string{}
	for _, id := range p.ResourceTreeUnits {
		m[id] = "flora"
	}
	m[p.DecorativeFloraUnit] = "flora"
	m[p.WaterFloraUnit] = "water_flora"
	layers := p.Layers
	if len(layers) == 0 {
		layers = defaultHabitatLayers(p, true)
	}
	for _, l := range layers {
		units := l.Units
		if len(units) == 0 {
			units = p.Palettes[l.EcologyRole]
		}
		role := l.Role
		if l.EcologyRole == "OYSTER" {
			role = "shellfish"
		}
		if role == "resource" || role == "decorative" {
			role = "flora"
			if l.WaterBand != "" {
				role = "water_flora"
			}
		}
		for _, u := range units {
			m[u.UnitConst] = role
		}
	}
	for role, units := range p.Palettes {
		r := "fauna"
		switch role {
		case "FORAGE_PLANT":
			r = "flora"
		case "NERITIC", "FRESHWATER", "SALTWATER", "WHALE":
			r = "fish"
		case "OYSTER":
			r = "shellfish"
		case "BIRD":
			continue
		}
		for _, u := range units {
			if _, exists := m[u.UnitConst]; !exists {
				m[u.UnitConst] = r
			}
		}
	}
	if p.WaterFloraUnit > 0 {
		m[p.WaterFloraUnit] = "water_flora"
	}
	for _, q := range p.NeutralResources {
		m[q.Unit] = "mineral"
	}
	m[66] = "mineral"
	m[102] = "mineral"
	return m
}

type placementContext struct {
	byID          map[int]datcache.UnitRow
	fractional    map[int]bool
	terrain       map[[2]int]int
	allowed       map[int]map[int]bool
	roles         map[int]string
	profile       EcologyProfile
	width, height int
	hash          string
}

// AuditPlacement checks actual serialized positions against actual final tiles.
// Returned removals are suitable for the named post-processing sweep. Required
// start inventory is never silently repaired: its invalid placement is an error.
func AuditPlacement(path, datPath string, p EcologyProfile, starts *PlayerStartReport) (PlacementAudit, []scenario.UnitRecipe, error) {
	f, err := scenario.Open(path)
	if err != nil {
		return PlacementAudit{State: "unavailable"}, nil, err
	}
	c, err := loadPlacementContext(f, datPath, p)
	if err != nil {
		return PlacementAudit{State: "unavailable"}, nil, err
	}
	var objects []placementObject
	for _, s := range f.Units.Sections {
		for _, u := range s.Units {
			o, err := c.object(u, s.Player)
			if err != nil {
				return PlacementAudit{State: "unavailable"}, nil, err
			}
			objects = append(objects, o)
		}
	}
	a, bad, err := c.audit(objects, starts)
	if err != nil {
		return a, nil, err
	}
	var removals []scenario.UnitRecipe
	for _, o := range objects {
		if bad[o.ReferenceID] {
			id := o.ReferenceID
			removals = append(removals, scenario.UnitRecipe{Op: "remove_unit", ReferenceID: &id})
		}
	}
	return a, removals, nil
}

// GateRecipePlacement checks individual additions against already-painted terrain
// before serialization. Expanded scatter operations remain covered by the final audit.
func GateRecipePlacement(path, datPath string, p EcologyProfile, starts *PlayerStartReport, units []scenario.UnitRecipe) (PlacementAudit, []scenario.UnitRecipe, error) {
	f, err := scenario.Open(path)
	if err != nil {
		return PlacementAudit{State: "unavailable"}, nil, err
	}
	if f.Units != nil && f.Units.Total != 0 {
		return PlacementAudit{State: "unavailable"}, nil, fmt.Errorf("placement gate requires a terrain-only input; use the written-file audit for populated scenarios")
	}
	c, err := loadPlacementContext(f, datPath, p)
	if err != nil {
		return PlacementAudit{State: "unavailable"}, nil, err
	}
	var objects []placementObject
	deferred := 0
	for i, u := range units {
		if u.Op != "add_unit" {
			deferred++
			continue
		}
		if u.X == nil || u.Y == nil {
			return PlacementAudit{State: "unavailable"}, nil, fmt.Errorf("placement gate: addition needs coordinates")
		}
		x, y := *u.X, *u.Y
		if u.Snap == "" || u.Snap == "grid" || (u.Snap == "auto" && !c.fractional[u.UnitConst]) {
			x = math.Floor(x) + .5
			y = math.Floor(y) + .5
		}
		o, err := c.object(scenario.UnitSummary{ReferenceID: i + 1, UnitConst: u.UnitConst, X: x, Y: y}, u.Player)
		if err != nil {
			return PlacementAudit{State: "unavailable"}, nil, err
		}
		objects = append(objects, o)
	}
	a, bad, err := c.audit(objects, starts)
	if err != nil {
		return a, nil, err
	}
	a.Deferred = deferred
	kept := make([]scenario.UnitRecipe, 0, len(units))
	for i, u := range units {
		if !bad[i+1] {
			kept = append(kept, u)
		} else {
			a.Rejected++
		}
	}
	return a, kept, nil
}

func loadPlacementContext(f *scenario.File, datPath string, p EcologyProfile) (placementContext, error) {
	c := placementContext{byID: map[int]datcache.UnitRow{}, fractional: map[int]bool{}, terrain: map[[2]int]int{}, allowed: map[int]map[int]bool{}, roles: placementRoles(p), profile: p}
	if datPath == "" {
		return c, fmt.Errorf("placement audit requires --dat; unchecked placement is not a pass")
	}
	tr, err := f.Terrain(scenario.TerrainOptions{IncludeTiles: true})
	if err != nil {
		return c, err
	}
	cache, err := datcache.Open(datPath)
	if err != nil {
		return c, err
	}
	defer cache.Close()
	rows, _, err := cache.QueryUnits(datcache.UnitQuery{All: true})
	if err != nil {
		return c, err
	}
	byID := map[int]datcache.UnitRow{}
	for _, row := range rows {
		id := int(row.ID)
		if _, ok := c.fractional[id]; !ok {
			c.fractional[id] = true
		}
		if scenario.CollisionFootprintBlocks(row.CollisionSizeX, row.CollisionSizeY, row.CollisionSizeZ) {
			c.fractional[id] = false
		}
		old, ok := byID[id]
		if !ok || row.CivIndex == 0 || (old.CivIndex != 0 && row.CivIndex == 1) {
			byID[id] = row
		}
	}
	c.byID = byID
	c.hash = cache.SHA256
	c.width = tr.Width
	c.height = tr.Height
	_, tables, err := datfile.ReadTerrainRestrictions(datPath)
	if err != nil {
		return c, err
	}
	terrain := map[[2]int]int{}
	for _, t := range tr.Tiles {
		terrain[[2]int{t.X, t.Y}] = t.TerrainID
	}
	c.terrain = terrain
	for i, table := range tables {
		m := map[int]bool{}
		for _, r := range table.TerrainRows {
			m[r.Index] = r.Passability > 0
		}
		c.allowed[i] = m
	}
	return c, nil
}

func (c placementContext) object(u scenario.UnitSummary, player int) (placementObject, error) {
	row, ok := c.byID[u.UnitConst]
	if !ok {
		return placementObject{}, fmt.Errorf("placement audit: unit %d absent from DAT", u.UnitConst)
	}
	if !finite(u.X) || !finite(u.Y) || !finite(float64(row.CollisionSizeX)) || !finite(float64(row.CollisionSizeY)) || !finite(float64(row.CollisionSizeZ)) || math.Abs(float64(row.CollisionSizeX)) > float64(c.width) || math.Abs(float64(row.CollisionSizeY)) > float64(c.height) {
		return placementObject{}, fmt.Errorf("placement audit: invalid geometry for unit %d", u.UnitConst)
	}
	role := c.roles[u.UnitConst]
	switch row.Class {
	case 5:
		if role != "shellfish" {
			role = "fish"
		}
	case 9, 10, 58:
		role = "fauna"
	case 7, 15:
		role = "flora"
	}
	return placementObject{u, player, row, role}, nil
}

func (c placementContext) terrainAllowed(o placementObject, x, y int) bool {
	t, ok := c.terrain[[2]int{x, y}]
	if !ok {
		return false
	}
	i := int(o.row.TerrainTableID)
	if !c.allowed[i][t] {
		return false
	}
	water := placementWaterTerrain(t)
	amphibious := isAmphibiousTerrain(t)
	if (o.role == "flora" || o.role == "water_flora") && water {
		for _, allowed := range c.profile.FloraWaterTerrains[o.UnitConst] {
			if allowed == t {
				return true
			}
		}
	}
	switch o.role {
	case "flora":
		return !water || amphibious
	case "fauna", "mineral":
		return !water
	case "fish":
		return water
	case "shellfish":
		return t == 1 || t == 2 || t == 4 || t == 52 || t == 53 || t == 54 || t == 55 || t == 58 || t == 59 || t == 95
	case "water_flora":
		return t == 1 || t == 4 || t == 23 || t == 55 || t == 58 || t == 59 || t == 95
	}
	return true
}

func isAmphibiousTerrain(id int) bool {
	switch id {
	case 2, 4, 52, 53, 54, 55, 59, 111, 115:
		return true
	default:
		return false
	}
}

func floraSurface(c Cell) bool {
	if isAmphibiousTerrain(c.TerrainID) {
		return true
	}
	if placementWaterTerrain(c.TerrainID) {
		return false
	}
	return c.Land != nil && *c.Land
}

func (c placementContext) audit(objects []placementObject, starts *PlayerStartReport) (PlacementAudit, map[int]bool, error) {
	a := PlacementAudit{State: "dat_terrain_checked", DATHash: c.hash, UnitCounts: map[int]int{}, InvalidByUnit: map[int]int{}}
	bad := map[int]bool{}
	for _, o := range objects {
		a.UnitCounts[o.UnitConst]++
		if o.role == "" {
			continue
		}
		a.Checked++
		if !placementTiles(o, func(x, y int) bool { return c.terrainAllowed(o, x, y) }) {
			bad[o.ReferenceID] = true
			if o.role == "flora" || o.role == "water_flora" {
				a.FloraTerrain++
			} else if o.role == "mineral" {
				a.ResourceTerrain++
			} else if o.role == "building" {
				a.BuildingTerrain++
			} else {
				a.FaunaTerrain++
			}
		}
		if o.role == "flora" || o.role == "water_flora" {
			if t, ok := c.terrain[[2]int{int(math.Floor(o.X)), int(math.Floor(o.Y))}]; ok && isAmphibiousTerrain(t) && placementTiles(o, func(x, y int) bool { return c.terrainAllowed(o, x, y) }) {
				a.FloraAmphibiousObjects++
			}
		}
	}
	// Index blocking objects before testing fauna, regardless of emission order.
	blockers := map[[2]int][]int{}
	for _, o := range objects {
		if bad[o.ReferenceID] || !scenario.CollisionFootprintBlocks(o.row.CollisionSizeX, o.row.CollisionSizeY, o.row.CollisionSizeZ) {
			continue
		}
		placementTiles(o, func(x, y int) bool { k := [2]int{x, y}; blockers[k] = append(blockers[k], o.ReferenceID); return true })
	}
	for _, o := range objects {
		if o.role != "building" || bad[o.ReferenceID] {
			continue
		}
		overlap := false
		placementTiles(o, func(x, y int) bool {
			for _, id := range blockers[[2]int{x, y}] {
				if id != o.ReferenceID && !bad[id] {
					overlap = true
					return false
				}
			}
			return true
		})
		if overlap {
			a.BuildingBlockers++
			bad[o.ReferenceID] = true
		}
	}
	if a.UnitCounts != nil {
		before, after, rejected := buildingConnectivity(c, objects, bad)
		a.LandComponentsBefore = before
		a.LandComponentsAfter = after
		a.BuildingsRejectedForConnectivity = rejected
		a.BuildingConnectivity = "static_connected"
	}
	for _, o := range objects {
		if o.role != "fauna" && o.role != "fish" && o.role != "shellfish" && o.role != "mineral" {
			continue
		}
		if !placementTiles(o, func(x, y int) bool {
			for _, id := range blockers[[2]int{x, y}] {
				if id != o.ReferenceID {
					return false
				}
			}
			return true
		}) {
			if o.role == "mineral" {
				a.ResourceBlockers++
			} else {
				a.FaunaBlockers++
			}
			bad[o.ReferenceID] = true
		}
	}
	for _, o := range objects {
		if !bad[o.ReferenceID] {
			continue
		}
		a.InvalidByUnit[o.UnitConst]++
		if starts != nil && starts.Mode == "explicit" {
			for _, s := range starts.Positions {
				for _, u := range s.Objects {
					if u.Player == o.player && u.Unit == o.UnitConst && math.Abs(u.X-o.X) < 1e-5 && math.Abs(u.Y-o.Y) < 1e-5 {
						return a, nil, fmt.Errorf("required start object %d at %.2f,%.2f violates placement gate", o.UnitConst, o.X, o.Y)
					}
				}
				if len(s.Objects) == 0 && math.Hypot(o.X-s.X, o.Y-s.Y) <= float64(starts.Config.ClearRadius) {
					return a, nil, fmt.Errorf("required legacy start object %d violates placement gate", o.UnitConst)
				}
			}
		}
	}
	return a, bad, nil
}

func buildingConnectivity(c placementContext, objects []placementObject, bad map[int]bool) (int, int, int) {
	base := map[[2]int]bool{}
	buildings := make([]placementObject, 0)
	for _, o := range objects {
		if bad[o.ReferenceID] || !scenario.CollisionFootprintBlocks(o.row.CollisionSizeX, o.row.CollisionSizeY, o.row.CollisionSizeZ) {
			continue
		}
		if o.role == "building" {
			buildings = append(buildings, o)
			continue
		}
		placementTiles(o, func(x, y int) bool {
			base[[2]int{x, y}] = true
			return true
		})
	}
	before := landComponentCount(c, base)
	combined := make(map[[2]int]bool, len(base))
	for key := range base {
		combined[key] = true
	}
	rejected := 0
	for _, o := range buildings {
		trial := make(map[[2]int]bool, len(combined)+16)
		for key := range combined {
			trial[key] = true
		}
		placementTiles(o, func(x, y int) bool {
			trial[[2]int{x, y}] = true
			return true
		})
		if landComponentCount(c, trial) > before {
			bad[o.ReferenceID] = true
			rejected++
			continue
		}
		combined = trial
	}
	after := landComponentCount(c, combined)
	return before, after, rejected
}

func landComponentCount(c placementContext, blocked map[[2]int]bool) int {
	seen := map[[2]int]bool{}
	count := 0
	for y := 0; y < c.height; y++ {
		for x := 0; x < c.width; x++ {
			origin := [2]int{x, y}
			if seen[origin] || blocked[origin] {
				continue
			}
			terrain, ok := c.terrain[origin]
			if !ok || placementWaterTerrain(terrain) {
				seen[origin] = true
				continue
			}
			count++
			queue := [][2]int{origin}
			seen[origin] = true
			for len(queue) > 0 {
				point := queue[0]
				queue = queue[1:]
				for _, next := range [][2]int{{point[0] - 1, point[1]}, {point[0] + 1, point[1]}, {point[0], point[1] - 1}, {point[0], point[1] + 1}} {
					if seen[next] || blocked[next] {
						continue
					}
					t, exists := c.terrain[next]
					if !exists || placementWaterTerrain(t) {
						seen[next] = true
						continue
					}
					seen[next] = true
					queue = append(queue, next)
				}
			}
		}
	}
	return count
}

func placementWaterTerrain(id int) bool {
	switch id {
	case 1, 4, 15, 22, 23, 26, 54, 55, 57, 58, 59, 95, 96, 111, 114, 115, 116, 130:
		return true
	}
	return false
}
