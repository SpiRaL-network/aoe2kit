package geotrace

import (
	"fmt"
	"math"
	"sort"

	"aoe2kit/pkg/scenario"
)

type PlayerStartProfile struct {
	Mode                string          `json:"mode,omitempty"` // explicit, engine, none, calibration
	Count               int             `json:"count,omitempty"`
	TownCenter          int             `json:"town_center,omitempty"`
	Villager            int             `json:"villager,omitempty"`
	Scout               int             `json:"scout,omitempty"`
	Civ                 int             `json:"civ,omitempty"`
	MinSeparation       float64         `json:"min_separation,omitempty"`
	ClearRadius         int             `json:"clear_radius,omitempty"`
	FloraClearance      int             `json:"flora_clearance,omitempty"`
	TownCenterFootprint int             `json:"town_center_footprint,omitempty"`
	Inventory           []StartResource `json:"inventory,omitempty"`
}

type StartResource struct {
	Role     string `json:"role"`
	Unit     int    `json:"unit,omitempty"`
	Count    int    `json:"count"`
	Distance int    `json:"distance"`
}

type StartPosition struct {
	Player               int              `json:"player"`
	X                    float64          `json:"x"`
	Y                    float64          `json:"y"`
	Lat                  float64          `json:"lat"`
	Lon                  float64          `json:"lon"`
	Reason               string           `json:"reason"`
	Inventory            map[int]int      `json:"inventory"`
	Resources            []StartPlacement `json:"resources"`
	Objects              []StartObject    `json:"objects,omitempty"`
	NaturalWoodlineTrees int              `json:"natural_woodline_trees,omitempty"`
}

type StartObject struct {
	Player int     `json:"player"`
	Unit   int     `json:"unit"`
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
}

type StartPlacement struct {
	Role         string  `json:"role"`
	Unit         int     `json:"unit"`
	Count        int     `json:"count"`
	Distance     float64 `json:"distance"`
	X            float64 `json:"x"`
	Y            float64 `json:"y"`
	SpeciesUnits []int   `json:"species_units,omitempty"`
}

type PlayerStartReport struct {
	Mode           string             `json:"mode"`
	Config         PlayerStartProfile `json:"config"`
	Evidence       string             `json:"evidence"`
	Provenance     []string           `json:"provenance"`
	Positions      []StartPosition    `json:"positions"`
	RemovedAmbient int                `json:"removed_ambient_objects"`
	AmbientBefore  map[string]int     `json:"ambient_before"`
	AmbientAfter   map[string]int     `json:"ambient_after"`
	Budget         map[string]float64 `json:"budget_comparison"`
	BudgetNote     string             `json:"budget_note"`
}

func normalizedStarts(p *PlayerStartProfile) (PlayerStartProfile, error) {
	c := PlayerStartProfile{Mode: "explicit", Count: 2, TownCenter: 109, Villager: 83, Scout: 448, Civ: 1, MinSeparation: 64, ClearRadius: 32, FloraClearance: 6, TownCenterFootprint: 4}
	if p != nil {
		c = *p
		if c.Mode == "" {
			c.Mode = "explicit"
		}
		if c.Count == 0 {
			c.Count = 2
		}
		if c.TownCenter == 0 {
			c.TownCenter = 109
		}
		if c.Villager == 0 {
			c.Villager = 83
		}
		if c.Scout == 0 {
			c.Scout = 448
		}
		if c.Civ == 0 {
			c.Civ = 1
		}
		if c.MinSeparation == 0 {
			c.MinSeparation = 64
		}
		if c.ClearRadius == 0 {
			c.ClearRadius = 32
		}
		if c.FloraClearance == 0 {
			c.FloraClearance = 6
		}
		if c.TownCenterFootprint == 0 {
			c.TownCenterFootprint = 4
		}
	}
	if c.Mode != "explicit" && c.Mode != "engine" && c.Mode != "none" && c.Mode != "calibration" {
		return c, fmt.Errorf("invalid player-start mode %q", c.Mode)
	}
	if c.Count < 1 || c.Count > 8 || c.ClearRadius < 16 || c.ClearRadius > 64 || !finite(c.MinSeparation) || c.MinSeparation < float64(c.ClearRadius*2) || c.MinSeparation > 680 || c.TownCenter < 1 || c.Villager < 1 || c.Scout < 1 || c.Civ < 1 {
		return c, fmt.Errorf("invalid player-start count, units, civ, separation or clearing radius")
	}
	if c.FloraClearance < 1 || c.FloraClearance > 20 || c.TownCenterFootprint < 1 || c.TownCenterFootprint > 12 || (c.TownCenterFootprint+1)/2+c.FloraClearance >= c.ClearRadius {
		return c, fmt.Errorf("invalid town-center footprint or flora clearance")
	}
	if c.Inventory == nil {
		c.Inventory = []StartResource{
			{"HERDABLE", 0, 4, 8}, {"HERDABLE", 0, 2, 22}, {"HERDABLE", 0, 2, 22},
			{"HUNTABLE", 0, 3, 14}, {"LUREABLE", 0, 1, 18}, {"LUREABLE", 0, 1, 18},
			{"FORAGE_PLANT", 59, 6, 12}, {"GOLD", 66, 7, 12}, {"GOLD", 66, 4, 22}, {"GOLD", 66, 4, 28},
			{"STONE", 102, 5, 16}, {"STONE", 102, 4, 22}, {"WOOD", 0, 5, 10}, {"WOOD", 0, 40, 24}}
	}
	for _, g := range c.Inventory {
		if g.Count < 1 || g.Count > 100 || g.Distance < 5 || g.Distance > c.ClearRadius-3 || g.Unit < 0 {
			return c, fmt.Errorf("invalid start inventory %q", g.Role)
		}
		if g.Role != "GOLD" && g.Role != "STONE" && g.Role != "WOOD" && !validEcologyRole(g.Role) {
			return c, fmt.Errorf("unknown start role %q", g.Role)
		}
	}
	return c, nil
}

func startRoleUnit(p EcologyProfile, g StartResource, index int) int {
	if units := p.Palettes[g.Role]; len(units) > 0 {
		return units[index%len(units)].UnitConst
	}
	if g.Unit > 0 {
		return g.Unit
	}
	switch g.Role {
	case "HERDABLE":
		return 594
	case "HUNTABLE":
		return p.Fauna.Deer
	case "HUNTABLE_SMALL":
		return 65
	case "LUREABLE":
		return p.Fauna.Boar
	case "WOOD":
		if len(p.ResourceTreeUnits) > 0 {
			return p.ResourceTreeUnits[index%len(p.ResourceTreeUnits)]
		}
	}
	return 0
}

func woodTreeIDs(p EcologyProfile) map[int]bool {
	ids := map[int]bool{}
	for _, id := range p.ResourceTreeUnits {
		ids[id] = true
	}
	for _, u := range p.Palettes["WOOD"] {
		ids[u.UnitConst] = true
	}
	for _, layer := range p.Layers {
		if layer.Role == "resource" {
			for _, u := range layer.Units {
				ids[u.UnitConst] = true
			}
		}
	}
	return ids
}

func naturalWoodlineCount(units []scenario.UnitRecipe, treeIDs map[int]bool, centerX, centerY float64, clearSet map[int]bool, gridSize int) int {
	count := 0
	for _, u := range units {
		if u.Op != "add_unit" || u.Player != 0 || !treeIDs[u.UnitConst] || u.X == nil || u.Y == nil {
			continue
		}
		distance := math.Hypot(*u.X-centerX, *u.Y-centerY)
		if distance < 18 || distance > 28 {
			continue
		}
		x, y := int(math.Floor(*u.X)), int(math.Floor(*u.Y))
		if x < 0 || y < 0 || x >= gridSize || y >= gridSize || clearSet[y*gridSize+x] {
			continue
		}
		count++
	}
	return count
}

func mixedWoodUnit(p EcologyProfile, cell Cell, x, y, seed int) int {
	choices := p.woodyUnitChoices(cell.USGSWoodySpecies)
	if len(choices) == 0 {
		choices = append([]int(nil), p.ResourceTreeUnits...)
	}
	if len(choices) == 0 {
		return 0
	}
	index := minInt(len(choices)-1, int(hashUnitFloat(x, y, seed)*float64(len(choices))))
	return choices[index]
}

func applyPlayerStarts(r *Report) error {
	if r.EcologyProfile.PlayerStarts == nil {
		return nil
	}
	c, err := normalizedStarts(r.EcologyProfile.PlayerStarts)
	if err != nil {
		return err
	}
	d := &PlayerStartReport{Mode: c.Mode, Config: c, Evidence: "Explicit scenario inventory; automatic DE empty-player starts are human-reported but not engine-verified. Explicit inventory is not civilization-specific RMS initialization.", Provenance: []string{
		"Official RMS includes/starting_resources.inc: primary forage6@12, gold7@12+4@22+4@28, stone5@16+4@22; conditional map selections, not universal totals.",
		"Official RMS Coastal.rms: selected large-hunt variant3@14 (alternative small-hunt variant7@14, not additive). includes/lureable.inc: two single-unit groups@18.",
		"Official RMS herdable_starting.inc and herdable.inc: selected four near@8 plus two groups of two@22. stragglers.inc: five; distance10 to honor flora clearance and extra40-tree wood pocket are Kit policy.",
		"Official RMS aquatic freshwater/saltwater includes:8192 candidate ceiling and10-tile spacing, not a realized count. No official universal per-player fish total."}, AmbientBefore: map[string]int{}, AmbientAfter: map[string]int{}, Budget: map[string]float64{}}
	r.PlayerStarts = d
	for _, l := range r.EcologyLayers {
		d.AmbientBefore[l.Role] += l.Emitted
	}
	area := float64(r.GridSize*r.GridSize) / (200 * 200)
	d.BudgetNote = "Normal reference is 200x200; Ludicrous 480x480 has 5.76 times its area. Official aquatic modules use candidate caps and spacing, not guaranteed fish counts. Predator band is 4-8 groups per 10000 tiles (module default through Yucatan). These are policy inputs, not engine-realized counts."
	d.Budget["map_area_reference_ratio"] = area
	d.Budget["start_huntable_baseline"] = float64(c.Count * 3)
	d.Budget["start_lureable_baseline"] = float64(c.Count * 2)
	d.Budget["fish_spacing10_area_proxy"] = float64(r.WaterCells) / 100
	d.Budget["predator_candidate_band_low"] = float64(r.GridSize*r.GridSize) * 4 / 10000
	d.Budget["predator_candidate_band_high"] = float64(r.GridSize*r.GridSize) * 8 / 10000
	for k, v := range d.AmbientBefore {
		d.AmbientAfter[k] = v
	}
	recipe := r.Recipe
	recipe.Scenario.PlayerCount = intPtr(c.Count)
	if c.Mode == "none" || c.Mode == "engine" {
		return nil
	}
	if c.Mode == "calibration" {
		for p := 1; p <= c.Count; p++ {
			recipe.Units = append(recipe.Units, startUnit(p, 12, float64(6+(p-1)*5)+.5, 6.5))
		}
		return nil
	}
	if r.GridSize < scenario.MinScenarioMapSize && r.EcologyProfile.PlayerStarts == nil {
		d.Mode = "preview_only"
		d.Evidence = "Grid below scenario minimum; explicit starts deferred until a scenario-sized trace."
		return nil
	}
	// A uniform grass clearing makes final terrain, footprint and resource access
	// independent of subsequent aesthetic terrain paint operations.
	tc, err := loadDockConstraints(r.DatPath, EcologyProfile{DockUnit: c.TownCenter, DockCiv: c.Civ})
	if err != nil {
		return err
	}
	if r.DatPath != "" {
		found := false
		for _, id := range tc.AllowedTerrains {
			if id == 0 {
				found = true
			}
		}
		if !found {
			return fmt.Errorf("town center terrain table excludes grass0")
		}
	}
	n := r.GridSize
	if len(r.Cells) != n*n {
		return fmt.Errorf("player starts require full grid")
	}
	good := make([]bool, n*n)
	for i, v := range r.Cells {
		good[i] = v.Land != nil && *v.Land && !v.InlandWater && cellBuildableForGaiaPlacement(v)
	}
	// Summed-area land test bounds candidate evaluation, even on480x480 maps.
	sums := make([]int, (n+1)*(n+1))
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			v := 0
			if good[y*n+x] {
				v = 1
			}
			sums[(y+1)*(n+1)+x+1] = v + sums[y*(n+1)+x+1] + sums[(y+1)*(n+1)+x] - sums[y*(n+1)+x]
		}
	}
	square := func(x, y, h int) bool {
		a, b, x2, y2 := x-h, y-h, x+h+1, y+h+1
		if a < 0 || b < 0 || x2 > n || y2 > n {
			return false
		}
		sum := sums[y2*(n+1)+x2] - sums[b*(n+1)+x2] - sums[y2*(n+1)+a] + sums[b*(n+1)+a]
		return sum == (2*h+1)*(2*h+1)
	}
	var chosen []int
	for p := 0; p < c.Count; p++ {
		best, score := -1, -1.0
		for y := c.ClearRadius; y < n-c.ClearRadius; y++ {
			for x := c.ClearRadius; x < n-c.ClearRadius; x++ {
				if !square(x, y, 8) {
					continue
				}
				s := math.Min(float64(x), math.Min(float64(y), math.Min(float64(n-1-x), float64(n-1-y))))
				if len(chosen) > 0 {
					s = math.Inf(1)
					for _, j := range chosen {
						s = math.Min(s, math.Hypot(float64(x-j%n), float64(y-j/n)))
					}
					if s < c.MinSeparation {
						continue
					}
				}
				if s > score {
					best, score = y*n+x, s
				}
			}
		}
		if best < 0 {
			return fmt.Errorf("cannot fit %d safe starts with separation%.0f; adjust profile", c.Count, c.MinSeparation)
		}
		chosen = append(chosen, best)
	}
	occupied := map[int]bool{}
	var added []scenario.UnitRecipe
	var clear []scenario.MaskCellRecipe
	clearSet := map[int]bool{}
	treeIDs := woodTreeIDs(r.EcologyProfile)
	ambientTreeGrid := make([]int, n*n)
	for _, u := range recipe.Units {
		if u.Op == "add_unit" && u.Player == 0 && treeIDs[u.UnitConst] && u.X != nil && u.Y != nil {
			x, y := int(math.Floor(*u.X)), int(math.Floor(*u.Y))
			if x >= 0 && y >= 0 && x < n && y < n {
				ambientTreeGrid[y*n+x]++
			}
		}
	}
	for p, j := range chosen {
		cx, cy := j%n, j/n
		firstAdded := len(added)
		// Flood only the reserved pocket. Inventory never crosses a water gap.
		connected := map[int]bool{j: true}
		parent := map[int]int{j: j}
		q := []int{j}
		for h := 0; h < len(q); h++ {
			k := q[h]
			x, y := k%n, k/n
			for _, off := range [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
				xx, yy := x+off[0], y+off[1]
				if xx < 0 || yy < 0 || xx >= n || yy >= n || math.Hypot(float64(xx-cx), float64(yy-cy)) > float64(c.ClearRadius) {
					continue
				}
				kk := yy*n + xx
				if good[kk] && !connected[kk] {
					connected[kk] = true
					parent[kk] = k
					q = append(q, kk)
				}
			}
		}
		h := float64(c.TownCenterFootprint)/2 + float64(c.FloraClearance)
		for _, k := range q {
			if math.Abs(float64(k%n-cx)) <= h && math.Abs(float64(k/n-cy)) <= h {
				clearSet[k] = true
			}
		}
		pos := StartPosition{Player: p + 1, X: float64(cx) + .5, Y: float64(cy) + .5, Lat: r.Cells[j].Lat, Lon: r.Cells[j].Lon, Reason: "17x17 dry buildable core; maximin separation; resource groups reachable through cleared land pocket; equal explicit inventory", Inventory: map[int]int{}}
		for yy := cy - 3; yy <= cy+3; yy++ {
			for xx := cx - 3; xx <= cx+3; xx++ {
				occupied[yy*n+xx] = true
			}
		}
		added = append(added, startUnit(p+1, c.TownCenter, pos.X, pos.Y))
		pos.Inventory[c.TownCenter]++
		// Replay flare order: two at offset (+1,+5), one at (-5,0).
		for _, off := range [][2]int{{1, 5}, {2, 5}, {-5, 0}} {
			x, y := cx+off[0], cy+off[1]
			occupied[y*n+x] = true
			added = append(added, startUnit(p+1, c.Villager, float64(x)+.5, float64(y)+.5))
			pos.Inventory[c.Villager]++
		}
		occupied[(cy-5)*n+cx] = true
		added = append(added, startUnit(p+1, c.Scout, pos.X, float64(cy)-4.5))
		pos.Inventory[c.Scout]++
		for gi, g := range c.Inventory {
			unit := startRoleUnit(r.EcologyProfile, g, gi)
			if unit <= 0 {
				return fmt.Errorf("no unit for start role %s", g.Role)
			}
			largeWoodline := g.Role == "WOOD" && g.Count >= 40
			if largeWoodline {
				natural := naturalWoodlineCount(recipe.Units, treeIDs, pos.X, pos.Y, clearSet, n)
				if natural >= g.Count {
					pos.NaturalWoodlineTrees = natural
					pos.Resources = append(pos.Resources, StartPlacement{Role: "WOODLINE_NATURAL", Count: natural, Distance: 24, X: pos.X, Y: pos.Y})
					continue
				}
			}
			var group []int
			bx, by := 0, 0
			// Equal radius targets with bounded outward slack; deterministic angular scan.
			for radius := g.Distance; radius <= minInt(g.Distance+3, c.ClearRadius-3) && group == nil; radius++ {
				for a := 0; a < 72; a++ {
					ang := (float64(a) + float64(gi)*7) * 2 * math.Pi / 72
					xx, yy := cx+int(math.Round(float64(radius)*math.Cos(ang))), cy+int(math.Round(float64(radius)*math.Sin(ang)))
					var candidates []int
					searchRadius := 4
					if largeWoodline {
						searchRadius = 8
					}
					for dy := -searchRadius; dy <= searchRadius; dy++ {
						for dx := -searchRadius; dx <= searchRadius; dx++ {
							if (g.Role == "WOOD" || g.Role == "FORAGE_PLANT") && math.Max(math.Abs(float64(xx+dx-cx)), math.Abs(float64(yy+dy-cy))) <= h+.5 {
								continue
							}
							k := (yy+dy)*n + xx + dx
							if xx+dx < 0 || xx+dx >= n || yy+dy < 0 || yy+dy >= n || !connected[k] || occupied[k] {
								continue
							}
							if largeWoodline && ambientTreeGrid[k] > 0 {
								continue
							}
							candidates = append(candidates, k)
						}
					}
					sort.SliceStable(candidates, func(a, b int) bool {
						if largeWoodline {
							score := func(k int) float64 {
								gx, gy := k%n, k/n
								nearby := 0
								for dy := -3; dy <= 3; dy++ {
									for dx := -3; dx <= 3; dx++ {
										x, y := gx+dx, gy+dy
										if x >= 0 && y >= 0 && x < n && y < n {
											nearby += ambientTreeGrid[y*n+x]
										}
									}
								}
								distance := math.Hypot(float64(gx-xx), float64(gy-yy))
								return float64(nearby)*0.12 + patchValue(gx, gy, 3, layerSeed("start-woodline", r.Seed+p*97))*0.7 - distance*0.025
							}
							if sa, sb := score(candidates[a]), score(candidates[b]); sa != sb {
								return sa > sb
							}
						}
						da := math.Hypot(float64(candidates[a]%n-xx), float64(candidates[a]/n-yy))
						db := math.Hypot(float64(candidates[b]%n-xx), float64(candidates[b]/n-yy))
						if da != db {
							return da < db
						}
						return candidates[a] < candidates[b]
					})
					if len(candidates) >= g.Count {
						trial := candidates[:g.Count]
						ok := true
						for _, k := range trial {
							if math.Hypot(float64(k%n-xx), float64(k/n-yy)) > math.Max(2, math.Sqrt(float64(g.Count))) {
								ok = false
							}
						}
						if ok {
							group = trial
							bx, by = xx, yy
							break
						}
					}
				}
			}
			if group == nil {
				return fmt.Errorf("player%d cannot place %s x%d at distance%d", p+1, g.Role, g.Count, g.Distance)
			}
			woodSpecies := map[int]bool{}
			woodPlacementIndex := 0
			for _, k := range group {
				objectUnit := unit
				if largeWoodline {
					cell := r.Cells[k]
					choices := r.EcologyProfile.woodyUnitChoices(cell.USGSWoodySpecies)
					if len(choices) < 2 {
						choices = append([]int(nil), r.EcologyProfile.ResourceTreeUnits...)
					}
					if len(choices) > 0 {
						objectUnit = choices[woodPlacementIndex%len(choices)]
						woodPlacementIndex++
					}
				}
				added = append(added, startUnit(0, objectUnit, float64(k%n)+.5, float64(k/n)+.5))
				pos.Inventory[objectUnit]++
				woodSpecies[objectUnit] = true
				for dy := -1; dy <= 1; dy++ {
					for dx := -1; dx <= 1; dx++ {
						kk := (k/n+dy)*n + k%n + dx
						occupied[kk] = true
						if !largeWoodline && connected[kk] {
							clearSet[kk] = true
						}
					}
				}
			}
			// Reserve a narrow connected approach, rather than erasing the whole pocket.
			for k := group[0]; ; k = parent[k] {
				for dy := -1; dy <= 1; dy++ {
					for dx := -1; dx <= 1; dx++ {
						kk := (k/n+dy)*n + k%n + dx
						if connected[kk] {
							clearSet[kk] = true
						}
					}
				}
				if k == j {
					break
				}
			}
			placement := StartPlacement{Role: g.Role, Unit: unit, Count: g.Count, Distance: math.Hypot(float64(bx-cx), float64(by-cy)), X: float64(bx) + .5, Y: float64(by) + .5}
			if largeWoodline {
				for id := range woodSpecies {
					placement.SpeciesUnits = append(placement.SpeciesUnits, id)
				}
				sort.Ints(placement.SpeciesUnits)
			}
			pos.Resources = append(pos.Resources, placement)
		}
		for _, u := range added[firstAdded:] {
			pos.Objects = append(pos.Objects, StartObject{u.Player, u.UnitConst, *u.X, *u.Y})
		}
		d.Positions = append(d.Positions, pos)
		recipe.Resources = append(recipe.Resources, scenario.ResourceRecipe{Player: p + 1, Food: intPtr(200), Wood: intPtr(200), Gold: intPtr(100), Stone: intPtr(200)})
	}
	kept := recipe.Units[:0]
	for _, u := range recipe.Units {
		remove := false
		if u.X != nil && u.Y != nil {
			x, y := int(math.Floor(*u.X)), int(math.Floor(*u.Y))
			remove = x >= 0 && x < n && y >= 0 && y < n && clearSet[y*n+x]
		}
		if remove {
			d.RemovedAmbient++
		} else {
			kept = append(kept, u)
		}
	}
	recipe.Units = append(kept, added...)
	indices := make([]int, 0, len(clearSet))
	for k := range clearSet {
		indices = append(indices, k)
	}
	sort.Ints(indices)
	for _, k := range indices {
		clear = append(clear, scenario.MaskCellRecipe{X: k % n, Y: k / n})
	}
	recipe.Masks = append(recipe.Masks, scenario.MaskRecipe{Name: "player_start_clearings", Op: "cells", Cells: clear})
	recipe.Map = append(recipe.Map, scenario.MapRecipe{Op: "set_terrain_mask", Mask: "player_start_clearings", TerrainID: intPtr(0)})
	// Final ambient counts exclude authored starts, after local clearing.
	roles := map[int]string{}
	for _, l := range r.EcologyLayers {
		for id := range l.UnitCounts {
			roles[id] = l.Role
		}
	}
	d.AmbientAfter = map[string]int{}
	for _, u := range kept {
		if role := roles[u.UnitConst]; role != "" {
			d.AmbientAfter[role]++
		}
	}
	return nil
}

func startUnit(player, unit int, x, y float64) scenario.UnitRecipe {
	return scenario.UnitRecipe{Op: "add_unit", Player: player, UnitConst: unit, X: float64Ptr(x), Y: float64Ptr(y), Status: intPtr(2), Snap: "grid"}
}
