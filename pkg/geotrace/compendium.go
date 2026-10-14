package geotrace

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"sort"
)

// Each selector has a total weight, divided between its matching artworks.
// Large families therefore do not overwhelm smaller ones merely by size.
type FeltSelection struct {
	Functions []string `json:"functions"`
	Tags      []string `json:"tags,omitempty"`
	Materials []string `json:"materials,omitempty"`
	Units     []int    `json:"unit_consts,omitempty"`
	Weight    float64  `json:"weight"`
}

type feltEntry struct {
	UnitConst     int      `json:"unit_const"`
	Form          string   `json:"form"`
	Functions     []string `json:"function"`
	Tags          []string `json:"vibe_tags"`
	Materials     []string `json:"material"`
	PlacementHint string   `json:"placement_hint"`
	Standing      string   `json:"standing"`
}

func feltStanding(s string) int {
	switch s {
	case "human_felt":
		return 3
	case "ai_felt_from_render":
		return 2
	case "ai_felt_from_metadata":
		return 1
	}
	return 0
}

func intersects(a, b []string) bool {
	for _, v := range a {
		if member(b, v) {
			return true
		}
	}
	return false
}

func resolveFeltLayers(p *EcologyProfile, path string) error {
	needed := false
	for _, l := range p.Layers {
		needed = needed || len(l.Felt) > 0
	}
	if !needed {
		return nil
	}
	if path == "" {
		return fmt.Errorf("felt habitat selectors require --compendium")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read felt compendium: %w", err)
	}
	var doc struct {
		Schema  string      `json:"schema"`
		Entries []feltEntry `json:"entries"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return fmt.Errorf("parse felt compendium: %w", err)
	}
	if doc.Schema != "aoe2-art-felt-compendium.v1" {
		return fmt.Errorf("unsupported felt compendium schema %q", doc.Schema)
	}
	entries := map[int]feltEntry{}
	for _, e := range doc.Entries {
		if e.UnitConst <= 0 {
			continue
		}
		old, ok := entries[e.UnitConst]
		if !ok || feltStanding(e.Standing) > feltStanding(old.Standing) {
			entries[e.UnitConst] = e
		}
	}
	for i := range p.Layers {
		l := &p.Layers[i]
		if len(l.Felt) == 0 {
			continue
		}
		if len(l.Units) > 0 {
			return fmt.Errorf("layer %q cannot combine explicit units and felt selectors", l.Name)
		}
		weights := map[int]float64{}
		l.FeltEntries = map[int]feltEntry{}
		for _, s := range l.Felt {
			var selected []feltEntry
			for _, e := range entries {
				if !intersects(s.Functions, e.Functions) || (len(s.Tags) > 0 && !intersects(s.Tags, e.Tags)) || (len(s.Materials) > 0 && !intersects(s.Materials, e.Materials)) || (len(s.Units) > 0 && !member(s.Units, e.UnitConst)) {
					continue
				}
				switch e.PlacementHint {
				case "coast", "waterside", "on_water", "carpet", "scatter", "avenue", "landmark":
					selected = append(selected, e)
				}
			}
			if len(selected) == 0 {
				return fmt.Errorf("felt selector in %q matches no supported artwork", l.Name)
			}
			for _, e := range selected {
				weights[e.UnitConst] += s.Weight / float64(len(selected))
				l.FeltEntries[e.UnitConst] = e
			}
		}
		var ids []int
		for id := range weights {
			ids = append(ids, id)
		}
		sort.Ints(ids)
		for _, id := range ids {
			l.Units = append(l.Units, WeightedUnit{id, weights[id]})
		}
	}
	p.CompendiumSHA256 = fmt.Sprintf("%x", sha256.Sum256(raw))
	return nil
}

func habitatUnits(c Cell, l HabitatLayer) []WeightedUnit {
	if len(l.FeltEntries) == 0 {
		return l.Units
	}
	var out []WeightedUnit
	for _, u := range l.Units {
		e := l.FeltEntries[u.UnitConst]
		land := floraSurface(c)
		wet := c.LandcoverKey == "90_herbaceous_wetland" || c.LandcoverKey == "95_mangrove"
		ok := false
		switch e.PlacementHint {
		case "coast":
			ok = land && c.DistanceToShoreTiles <= 8 && !c.Roadside
		case "waterside":
			ok = land && (wet || c.DistanceToShoreTiles <= 3) && !c.Roadside
		case "on_water":
			ok = c.Land != nil && !land && (c.InlandWater || c.DistanceToShoreTiles <= 8)
		case "avenue":
			ok = land && c.Roadside
		case "carpet", "scatter", "landmark":
			ok = land && (!c.Roadside || l.PopulationCenterLayer == "road_corridor")
		}
		if ok {
			out = append(out, u)
		}
	}
	return out
}
