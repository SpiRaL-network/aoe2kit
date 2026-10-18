package preview

import (
	"fmt"
	"math"
	"sort"

	"aoe2kit/pkg/datfile"
	"aoe2kit/pkg/scenario"
)

type FacetEntry struct {
	UnitID     int             `json:"unit_id"`
	Name       string          `json:"dat_name,omitempty"`
	Count      int             `json:"count"`
	Stored     []RotationCount `json:"stored_rotations"`
	Available  []int           `json:"available"`
	Placed     []int           `json:"placed"`
	Missing    []int           `json:"missing"`
	Unexpected []RotationCount `json:"unexpected,omitempty"`
	Complete   *bool           `json:"complete,omitempty"`
	Unknown    string          `json:"unknown,omitempty"`
}

type FacetReport struct {
	Schema  string       `json:"schema"`
	Version string       `json:"scenario_version"`
	Crop    Crop         `json:"crop"`
	Units   []FacetEntry `json:"units"`
	Notes   []string     `json:"notes"`
}

// ArtworkFacets compares stored index coverage with DAT standing-graphic angle
// slots. Animation frames are deliberately not a visual variant inventory.
func ArtworkFacets(art *scenario.Artwork, idx *datfile.Index, opts StudyOptions) (FacetReport, error) {
	if idx == nil {
		return FacetReport{}, fmt.Errorf("facets requires a DAT")
	}
	census, err := ArtworkCensus(art, idx, opts)
	if err != nil {
		return FacetReport{}, err
	}
	r := FacetReport{Schema: "aoe2kit.artwork.facets.v1", Version: art.Version, Crop: census.Crop, Units: []FacetEntry{}, Notes: []string{
		"Available means DAT Gaia standing-graphic angle slots (zero angle count uses one slot, matching preview), not animation frames. Loose SLD presence, distinct pixels and engine rendering are not verified by this report.",
		"Placed uses the preview's direct stored-index model for 1.58/1.59. Older versions retain raw rotations but do not claim variant coverage until engine-calibrated. Triggers and spawned objects are not evaluated.",
	}}
	entries := map[int]CensusEntry{}
	for _, e := range census.Units {
		entries[e.UnitID] = e
	}
	for _, id := range opts.UnitIDs {
		if _, ok := entries[id]; !ok {
			entries[id] = CensusEntry{UnitID: id}
		}
	}
	meta := studyMetadata(idx)
	ids := make([]int, 0, len(entries))
	for id := range entries {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	for _, id := range ids {
		e := entries[id]
		f := FacetEntry{UnitID: id, Count: e.Count, Stored: e.Rotations, Available: []int{}, Placed: []int{}, Missing: []int{}}
		if f.Stored == nil {
			f.Stored = []RotationCount{}
		}
		u, ok := meta[id]
		if !ok {
			f.Unknown = "unit absent from Gaia DAT"
			r.Units = append(r.Units, f)
			continue
		}
		f.Name = u.Name
		g, ok := idx.Graphic(int(u.StandingGraphic1))
		if !ok {
			f.Unknown = "standing graphic absent from DAT"
			r.Units = append(r.Units, f)
			continue
		}
		if g.ParticleEffectName.Value != "" {
			f.Unknown = "particle graphics require their own installed angle definition"
			r.Units = append(r.Units, f)
			continue
		}
		if g.AngleCount < 0 {
			f.Unknown = "invalid DAT angle count"
			r.Units = append(r.Units, f)
			continue
		}
		n := max(1, int(g.AngleCount))
		for i := 0; i < n; i++ {
			f.Available = append(f.Available, i)
		}
		if art.Version != "1.58" && art.Version != "1.59" {
			f.Unknown = "stored rotation visual semantics not engine-verified for this version"
			r.Units = append(r.Units, f)
			continue
		}
		placed := map[int]bool{}
		for _, rotation := range e.Rotations {
			v := rotation.Rotation
			if math.IsNaN(v) || math.IsInf(v, 0) || v < 0 || v >= float64(n) || math.Trunc(v) != v {
				f.Unexpected = append(f.Unexpected, rotation)
				continue
			}
			placed[int(v)] = true
		}
		for _, v := range f.Available {
			if placed[v] {
				f.Placed = append(f.Placed, v)
			} else {
				f.Missing = append(f.Missing, v)
			}
		}
		complete := len(f.Missing) == 0 && len(f.Unexpected) == 0
		f.Complete = &complete
		r.Units = append(r.Units, f)
	}
	return r, nil
}
