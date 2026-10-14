package geotrace

import (
	"fmt"
	"math"
	"sort"

	"aoe2kit/pkg/datcache"
	"aoe2kit/pkg/scenario"
)

type NeutralResourceProfile struct {
	Name           string  `json:"name"`
	Unit           int     `json:"unit"`
	PileSize       int     `json:"pile_size"`
	GroupsPer10000 float64 `json:"groups_per_10000"`
	Spacing        float64 `json:"spacing"`
	StartDistance  float64 `json:"start_distance"`
	Provenance     string  `json:"provenance"`
}

type NeutralResourceReport struct {
	Config          NeutralResourceProfile `json:"config"`
	AreaScale       float64                `json:"area_scale"`
	RequestedGroups int                    `json:"requested_groups"`
	EmittedGroups   int                    `json:"emitted_groups"`
	CandidateGroups int                    `json:"candidate_groups"`
	RejectedGroups  int                    `json:"rejected_groups"`
	Objects         int                    `json:"objects"`
	Centres         []scenario.PointRecipe `json:"centres"`
	Piles           []NeutralPile          `json:"piles"`
}

type NeutralPile struct {
	Centre  scenario.PointRecipe `json:"centre"`
	Objects []StartObject        `json:"objects"`
}

func defaultNeutralResources() []NeutralResourceProfile {
	return []NeutralResourceProfile{
		{Name: "neutral_gold", Unit: 66, PileSize: 4, GroupsPer10000: 1.5, Spacing: 48, StartDistance: 48, Provenance: "Nomad: six-object piles, candidate group cap1024, spacing24; remote_resources: four-object piles, spacing48, start distance100, cap1024. Kit chooses four-object piles at spacing48, 1.5 groups per10000 map tiles, start distance48. Area scaling is explicit Kit policy; candidate caps are not realized totals. empire_gold is an Empire Wars starting-miner module, not an ambient density source."},
		{Name: "neutral_stone", Unit: 102, PileSize: 4, GroupsPer10000: 1, Spacing: 48, StartDistance: 48, Provenance: "Same Nomad and remote_resources pile/spacing references as gold. Kit chooses one group per10000 map tiles and48-tile start separation. No implicit RMS scaling multiplier assumed."},
	}
}

func validateNeutralResources(profiles []NeutralResourceProfile) error {
	names := map[string]bool{}
	for _, p := range profiles {
		if p.Name == "" || names[p.Name] || p.Unit <= 0 || p.PileSize < 1 || p.PileSize > 9 || !finite(p.GroupsPer10000) || p.GroupsPer10000 < 0 || p.GroupsPer10000 > 16 || !finite(p.Spacing) || p.Spacing < 4 || p.Spacing > 200 || !finite(p.StartDistance) || p.StartDistance < 16 || p.StartDistance > 200 || p.Provenance == "" {
			return fmt.Errorf("invalid neutral resource profile %q", p.Name)
		}
		names[p.Name] = true
	}
	return nil
}

func applyNeutralResources(r *Report) error {
	if r.GridSize < scenario.MinScenarioMapSize {
		return nil
	}
	profiles := r.EcologyProfile.NeutralResources
	if profiles == nil {
		return nil
	}
	if err := validateNeutralResources(profiles); err != nil {
		return err
	}
	n := r.GridSize
	blocking := map[int]bool{}
	if r.DatPath != "" {
		cache, err := datcache.Open(r.DatPath)
		if err != nil {
			return err
		}
		rows, _, err := cache.QueryUnits(datcache.UnitQuery{All: true})
		closeErr := cache.Close()
		if err != nil {
			return err
		}
		if closeErr != nil {
			return closeErr
		}
		for _, row := range rows {
			id := int(row.ID)
			blocking[id] = blocking[id] || scenario.CollisionFootprintBlocks(row.CollisionSizeX, row.CollisionSizeY, row.CollisionSizeZ)
		}
	}
	occupied := map[[2]int]bool{}
	for _, u := range r.Recipe.Units {
		if blocks, known := blocking[u.UnitConst]; known && !blocks {
			continue
		}
		if u.X != nil && u.Y != nil {
			occupied[[2]int{int(math.Floor(*u.X)), int(math.Floor(*u.Y))}] = true
		}
	}
	type centre struct{ x, y, spacing float64 }
	var centres []centre
	for _, p := range profiles {
		d := NeutralResourceReport{Config: p, AreaScale: float64(n*n) / 10000, RequestedGroups: int(math.Round(float64(n*n) * p.GroupsPer10000 / 10000))}
		var candidates []int
		for i, c := range r.Cells {
			if c.Land != nil && *c.Land && !c.InlandWater && cellBuildableForGaiaPlacement(c) && c.X >= 6 && c.Y >= 6 && c.X < n-6 && c.Y < n-6 {
				candidates = append(candidates, i)
			}
		}
		seed := layerSeed(p.Name, r.Seed)
		sort.Slice(candidates, func(i, j int) bool {
			a, b := candidates[i], candidates[j]
			va, vb := hashUnitFloat(a%n, a/n, seed), hashUnitFloat(b%n, b/n, seed)
			if va == vb {
				return a < b
			}
			return va > vb
		})
		for _, k := range candidates {
			if d.EmittedGroups >= d.RequestedGroups {
				break
			}
			x, y := k%n, k/n
			cx, cy := float64(x)+.5, float64(y)+.5
			close := false
			if r.PlayerStarts != nil {
				for _, s := range r.PlayerStarts.Positions {
					if math.Hypot(cx-s.X, cy-s.Y) < p.StartDistance {
						close = true
						break
					}
				}
			}
			for _, c := range centres {
				if math.Hypot(cx-c.x, cy-c.y) < math.Max(p.Spacing, c.spacing) {
					close = true
					break
				}
			}
			if close {
				continue
			}
			var cells [][2]int
			valid := true
			for dy := -2; dy <= 2; dy++ {
				for dx := -2; dx <= 2; dx++ {
					c := r.Cells[(y+dy)*n+x+dx]
					if c.Land == nil || !*c.Land || c.InlandWater || !cellBuildableForGaiaPlacement(c) || occupied[[2]int{x + dx, y + dy}] {
						valid = false
					}
					if dx >= 0 && dy >= 0 {
						cells = append(cells, [2]int{x + dx, y + dy})
					}
				}
			}
			if !valid || len(cells) < p.PileSize {
				continue
			}
			sort.SliceStable(cells, func(i, j int) bool {
				dx, dy := cells[i][0]-x, cells[i][1]-y
				ex, ey := cells[j][0]-x, cells[j][1]-y
				return dx*dx+dy*dy < ex*ex+ey*ey
			})
			for _, c := range cells[:p.PileSize] {
				r.Recipe.Units = append(r.Recipe.Units, startUnit(0, p.Unit, float64(c[0])+.5, float64(c[1])+.5))
				occupied[c] = true
			}
			pile := NeutralPile{Centre: scenario.PointRecipe{X: cx, Y: cy}}
			for _, c := range cells[:p.PileSize] {
				pile.Objects = append(pile.Objects, StartObject{Player: 0, Unit: p.Unit, X: float64(c[0]) + .5, Y: float64(c[1]) + .5})
			}
			d.Piles = append(d.Piles, pile)
			centres = append(centres, centre{cx, cy, p.Spacing})
			d.Centres = append(d.Centres, scenario.PointRecipe{X: cx, Y: cy})
			d.EmittedGroups++
			d.Objects += p.PileSize
		}
		d.CandidateGroups = d.EmittedGroups
		r.NeutralResources = append(r.NeutralResources, d)
	}
	return nil
}

// FinalizeNeutralResources rejects an entire pile when the terrain gate rejects
// any member, so reported pile sizes continue to describe the written file.
func FinalizeNeutralResources(r *Report) {
	type key struct {
		player, unit int
		x, y         float64
	}
	keyOf := func(o StartObject) key { return key{o.Player, o.Unit, o.X, o.Y} }
	present := map[key]bool{}
	for _, u := range r.Recipe.Units {
		if u.Op == "add_unit" && u.X != nil && u.Y != nil {
			present[key{u.Player, u.UnitConst, *u.X, *u.Y}] = true
		}
	}
	remove := map[key]bool{}
	for i := range r.NeutralResources {
		d := &r.NeutralResources[i]
		var kept []NeutralPile
		d.Centres = nil
		d.Objects = 0
		for _, p := range d.Piles {
			complete := true
			for _, o := range p.Objects {
				if !present[keyOf(o)] {
					complete = false
				}
			}
			if !complete {
				d.RejectedGroups++
				for _, o := range p.Objects {
					if present[keyOf(o)] {
						remove[keyOf(o)] = true
					}
				}
				continue
			}
			kept = append(kept, p)
			d.Centres = append(d.Centres, p.Centre)
			d.Objects += len(p.Objects)
		}
		d.Piles = kept
		d.EmittedGroups = len(kept)
	}
	kept := r.Recipe.Units[:0]
	for _, u := range r.Recipe.Units {
		if u.X != nil && u.Y != nil && remove[key{u.Player, u.UnitConst, *u.X, *u.Y}] {
			if r.PlacementGate != nil {
				r.PlacementGate.Rejected++
			}
			continue
		}
		kept = append(kept, u)
	}
	r.Recipe.Units = kept
}
