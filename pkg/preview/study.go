package preview

import (
	"fmt"
	"math"
	"sort"

	"aoe2kit/pkg/datfile"
	"aoe2kit/pkg/scenario"
)

type StudyOptions struct {
	Crop              *Crop
	GaiaOnly          bool
	IncludeForestFish bool
	Cell, Top         int
	Rank              string
}

type RotationCount struct {
	Rotation float64 `json:"rotation"`
	Count    int     `json:"count"`
}

type CensusEntry struct {
	UnitID    int             `json:"unit_id"`
	Name      string          `json:"dat_name,omitempty"`
	ClassID   int16           `json:"class_id"`
	Known     bool            `json:"dat_known"`
	Count     int             `json:"count"`
	Players   map[int]int     `json:"players"`
	Rotations []RotationCount `json:"rotations"`
}

type CensusReport struct {
	Schema   string        `json:"schema"`
	Crop     Crop          `json:"crop"`
	Total    int           `json:"total"`
	Distinct int           `json:"distinct_unit_kinds"`
	Units    []CensusEntry `json:"units"`
	Notes    []string      `json:"notes"`
}

type Hotspot struct {
	Crop    Crop          `json:"crop"`
	Count   int           `json:"count"`
	Variety int           `json:"distinct_unit_kinds"`
	Units   []CensusEntry `json:"units"`
}

type HotspotReport struct {
	Schema             string    `json:"schema"`
	Rank               string    `json:"rank"`
	GaiaOnly           bool      `json:"gaia_only"`
	ExcludedForestFish int       `json:"excluded_forest_fish"`
	MatchedCells       int       `json:"matched_cells"`
	Hotspots           []Hotspot `json:"hotspots"`
	Notes              []string  `json:"notes"`
}

func studyCrop(art *scenario.Artwork, c *Crop) (Crop, error) {
	if art == nil || art.Width <= 0 || art.Height <= 0 {
		return Crop{}, fmt.Errorf("study requires decoded artwork")
	}
	out := Crop{X2: art.Width, Y2: art.Height}
	if c != nil {
		out = *c
	}
	if out.X1 < 0 || out.Y1 < 0 || out.X2 > art.Width || out.Y2 > art.Height || out.X1 >= out.X2 || out.Y1 >= out.Y2 {
		return Crop{}, fmt.Errorf("crop must be a nonempty half-open rectangle within the map")
	}
	return out, nil
}

func inStudyCrop(u scenario.UnitSummary, c Crop) bool {
	return !math.IsNaN(u.X) && !math.IsNaN(u.Y) && u.X >= float64(c.X1) && u.Y >= float64(c.Y1) && u.X < float64(c.X2) && u.Y < float64(c.Y2)
}

type censusAccumulator struct {
	entries   map[int]*CensusEntry
	rotations map[int]map[float64]int
	total     int
}

func (a *censusAccumulator) add(u scenario.UnitSummary, player int, meta datfile.UnitSummary, known bool) {
	if a.entries == nil {
		a.entries = map[int]*CensusEntry{}
		a.rotations = map[int]map[float64]int{}
	}
	e := a.entries[u.UnitConst]
	if e == nil {
		e = &CensusEntry{UnitID: u.UnitConst, Name: meta.Name, ClassID: meta.Class, Known: known, Players: map[int]int{}}
		a.entries[u.UnitConst] = e
		a.rotations[u.UnitConst] = map[float64]int{}
	}
	e.Count++
	e.Players[player]++
	a.rotations[u.UnitConst][u.Rotation]++
	a.total++
}

func (a *censusAccumulator) list() []CensusEntry {
	out := make([]CensusEntry, 0, len(a.entries))
	for id, entry := range a.entries {
		e := *entry
		for r, n := range a.rotations[id] {
			e.Rotations = append(e.Rotations, RotationCount{r, n})
		}
		sort.Slice(e.Rotations, func(i, j int) bool { return e.Rotations[i].Rotation < e.Rotations[j].Rotation })
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].UnitID < out[j].UnitID })
	return out
}

func studyMetadata(idx *datfile.Index) map[int]datfile.UnitSummary {
	out := map[int]datfile.UnitSummary{}
	if idx == nil || len(idx.Civs) == 0 {
		return out
	}
	// Match previews: Gaia DAT standing artwork, not a civilization simulation.
	for _, u := range idx.Civs[0].Units {
		if u.Present {
			out[u.Index] = u
		}
	}
	return out
}

// ArtworkCensus counts anchors in a half-open crop; rotations are stored values,
// not animation frame ordinals or a claim that every value is a valid variant.
func ArtworkCensus(art *scenario.Artwork, idx *datfile.Index, opts StudyOptions) (CensusReport, error) {
	c, err := studyCrop(art, opts.Crop)
	if err != nil {
		return CensusReport{}, err
	}
	meta := studyMetadata(idx)
	a := censusAccumulator{}
	for _, section := range art.Units.Sections {
		if opts.GaiaOnly && section.Player != 0 {
			continue
		}
		for _, u := range section.Units {
			if !inStudyCrop(u, c) {
				continue
			}
			m, ok := meta[u.UnitConst]
			a.add(u, section.Player, m, ok)
		}
	}
	return CensusReport{Schema: "aoe2kit.artwork.census.v1", Crop: c, Total: a.total, Distinct: len(a.entries), Units: a.list(), Notes: []string{"Anchor-in-crop census, not a complete placement recipe. DAT names are internal names from Gaia; rotations are raw stored values. Triggers and spawned units are not evaluated."}}, nil
}

// ArtworkHotspots ranks occupied grid cells, with deterministic ties. Only the
// DAT tree and fish classes are excluded; unknown units stay visible for study.
func ArtworkHotspots(art *scenario.Artwork, idx *datfile.Index, opts StudyOptions) (HotspotReport, error) {
	c, err := studyCrop(art, opts.Crop)
	if err != nil {
		return HotspotReport{}, err
	}
	if idx == nil {
		return HotspotReport{}, fmt.Errorf("hotspots requires a DAT for class filtering")
	}
	if opts.Cell == 0 {
		opts.Cell = 8
	}
	if opts.Top == 0 {
		opts.Top = 20
	}
	if opts.Rank == "" {
		opts.Rank = "variety"
	}
	if opts.Cell < 1 || opts.Cell > 1024 || opts.Top < 1 || opts.Top > 10000 || (opts.Rank != "count" && opts.Rank != "variety") {
		return HotspotReport{}, fmt.Errorf("invalid cell/top/rank (cell 1..1024, top 1..10000, rank count|variety)")
	}
	r := HotspotReport{Schema: "aoe2kit.artwork.hotspots.v1", Rank: opts.Rank, GaiaOnly: opts.GaiaOnly, Hotspots: []Hotspot{}, Notes: []string{"Cells are anchored at map origin and clipped to the requested crop. Ranking uses anchors, not sprite extents. No trigger-created units. Forest/fish exclusion uses DAT class IDs 15 (trees), 5, 31, 33 and 63 (fish), not name guesses; unknown units are retained."}}
	meta := studyMetadata(idx)
	cells := map[[2]int]*censusAccumulator{}
	for _, section := range art.Units.Sections {
		if opts.GaiaOnly && section.Player != 0 {
			continue
		}
		for _, u := range section.Units {
			if !inStudyCrop(u, c) {
				continue
			}
			m, ok := meta[u.UnitConst]
			if !opts.IncludeForestFish && ok && (m.Class == 15 || m.Class == 5 || m.Class == 31 || m.Class == 33 || m.Class == 63) {
				r.ExcludedForestFish++
				continue
			}
			key := [2]int{int(u.X) / opts.Cell, int(u.Y) / opts.Cell}
			if cells[key] == nil {
				cells[key] = &censusAccumulator{}
			}
			cells[key].add(u, section.Player, m, ok)
		}
	}
	for k, a := range cells {
		x, y := k[0]*opts.Cell, k[1]*opts.Cell
		r.Hotspots = append(r.Hotspots, Hotspot{Crop: Crop{max(x, c.X1), max(y, c.Y1), min(x+opts.Cell, c.X2), min(y+opts.Cell, c.Y2)}, Count: a.total, Variety: len(a.entries), Units: a.list()})
	}
	sort.Slice(r.Hotspots, func(i, j int) bool {
		a, b := r.Hotspots[i], r.Hotspots[j]
		if opts.Rank == "variety" && a.Variety != b.Variety {
			return a.Variety > b.Variety
		}
		if a.Count != b.Count {
			return a.Count > b.Count
		}
		if a.Variety != b.Variety {
			return a.Variety > b.Variety
		}
		if a.Crop.Y1 != b.Crop.Y1 {
			return a.Crop.Y1 < b.Crop.Y1
		}
		return a.Crop.X1 < b.Crop.X1
	})
	r.MatchedCells = len(r.Hotspots)
	if len(r.Hotspots) > opts.Top {
		r.Hotspots = r.Hotspots[:opts.Top]
	}
	return r, nil
}
