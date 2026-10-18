package preview

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"aoe2kit/pkg/datfile"
	"aoe2kit/pkg/replay"
	"aoe2kit/pkg/scenario"
)

type CorpusUnit struct {
	CensusEntry
	Maps         int  `json:"maps"`
	AgainstCount int  `json:"against_count"`
	AgainstMaps  int  `json:"against_maps"`
	CountDelta   int  `json:"count_delta"`
	MapDelta     int  `json:"map_delta"`
	OnlyInInput  bool `json:"only_in_input"`
}

type CensusFile struct {
	Path       string `json:"path"`
	Version    string `json:"version"`
	Units      int    `json:"matched_units"`
	PeakRSSKiB int64  `json:"peak_rss_kib,omitempty"`
}

type CorpusReport struct {
	Schema       string       `json:"schema"`
	Total        int          `json:"total"`
	Returned     int          `json:"returned"`
	Files        []CensusFile `json:"files"`
	AgainstFiles []CensusFile `json:"against_files,omitempty"`
	Units        []CorpusUnit `json:"units"`
	Notes        []string     `json:"notes"`
}

// ScenarioPaths expands directories deterministically and deduplicates aliases.
// Directory symlinks are not traversed; explicit file symlinks are canonicalized.
func ScenarioPaths(inputs []string) ([]string, error) {
	seen := map[string]bool{}
	var out []string
	add := func(path string) error {
		if !strings.EqualFold(filepath.Ext(path), ".aoe2scenario") {
			return fmt.Errorf("not a scenario: %s", path)
		}
		p, err := filepath.EvalSymlinks(path)
		if err != nil {
			return err
		}
		p, err = filepath.Abs(p)
		if err != nil {
			return err
		}
		if !seen[p] {
			seen[p] = true
			out = append(out, p)
		}
		return nil
	}
	for _, input := range inputs {
		s, err := os.Stat(input)
		if err != nil {
			return nil, err
		}
		if !s.IsDir() {
			if err := add(input); err != nil {
				return nil, err
			}
			continue
		}
		err = filepath.WalkDir(input, func(p string, d fs.DirEntry, e error) error {
			if e != nil {
				return e
			}
			if d.IsDir() || !strings.EqualFold(filepath.Ext(p), ".aoe2scenario") {
				return nil
			}
			return add(p)
		})
		if err != nil {
			return nil, err
		}
	}
	sort.Strings(out)
	if len(out) == 0 {
		return nil, fmt.Errorf("no scenario files found")
	}
	return out, nil
}

type CorpusCounter struct {
	opts   StudyOptions
	meta   map[int]datfile.UnitSummary
	counts censusAccumulator
	maps   map[int]int
	files  []CensusFile
	seen   map[string]bool
}

func NewCorpusCounter(idx *datfile.Index, opts StudyOptions) *CorpusCounter {
	return &CorpusCounter{opts: opts, meta: studyMetadata(idx), maps: map[int]int{}, seen: map[string]bool{}}
}

// Add accumulates one file at a time, retaining counts rather than its objects.
func (c *CorpusCounter) Add(path string, art *scenario.Artwork, peakRSSKiB int64) error {
	if art == nil || c.seen[path] {
		return fmt.Errorf("nil or duplicate census input %s", path)
	}
	crop, err := studyCrop(art, c.opts.Crop)
	if err != nil {
		return err
	}
	seen := map[int]bool{}
	before := c.counts.total
	for _, section := range art.Units.Sections {
		for _, u := range section.Units {
			if !studySelected(section.Player, u, c.opts) || !inStudyCrop(u, crop) {
				continue
			}
			m, ok := c.meta[u.UnitConst]
			if !ok {
				m.Name = replay.UnitDisplayName(u.UnitConst)
			}
			c.counts.add(u, section.Player, m, ok)
			seen[u.UnitConst] = true
		}
	}
	for id := range seen {
		c.maps[id]++
	}
	c.files = append(c.files, CensusFile{path, art.Version, c.counts.total - before, peakRSSKiB})
	c.seen[path] = true
	return nil
}

func (c *CorpusCounter) Report(against *CorpusCounter, minMaps int) (CorpusReport, error) {
	if minMaps < 1 {
		return CorpusReport{}, fmt.Errorf("min-maps must be positive")
	}
	r := CorpusReport{Schema: "aoe2kit.artwork.corpus-census.v1", Total: c.counts.total, Files: c.files, Units: []CorpusUnit{}, Notes: []string{"Serial anchor census; one occurrence per scenario contributes to maps. Duplicate file aliases are counted once. Rotations are stored values. No trigger-created objects. DAT internal names are used when supplied, otherwise the built-in unit name table is used. Against counts use the same filters; units present only in the comparison are omitted."}}
	other := map[int]CensusEntry{}
	if against != nil {
		r.AgainstFiles = against.files
		for _, e := range against.counts.list() {
			other[e.UnitID] = e
		}
	}
	for _, e := range c.counts.list() {
		if c.maps[e.UnitID] < minMaps {
			continue
		}
		b := other[e.UnitID]
		n := 0
		if against != nil {
			n = against.maps[e.UnitID]
		}
		r.Units = append(r.Units, CorpusUnit{CensusEntry: e, Maps: c.maps[e.UnitID], AgainstCount: b.Count, AgainstMaps: n, CountDelta: e.Count - b.Count, MapDelta: c.maps[e.UnitID] - n, OnlyInInput: against != nil && n == 0})
	}
	r.Returned = len(r.Units)
	return r, nil
}
