package preview

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"aoe2kit/pkg/datfile"
	"aoe2kit/pkg/scenario"
)

type NeededAsset struct {
	Path   string `json:"path"`
	Kind   string `json:"kind"`
	Status string `json:"status"`
	Units  []int  `json:"unit_ids"`
}

type AssetIssue struct {
	UnitID int    `json:"unit_id"`
	Reason string `json:"reason"`
}

type AssetManifest struct {
	Schema     string        `json:"schema"`
	Crop       Crop          `json:"crop"`
	Assets     []NeededAsset `json:"assets"`
	Unresolved []AssetIssue  `json:"unresolved"`
	Complete   bool          `json:"dependency_list_complete"`
	Missing    int           `json:"missing"`
	Notes      []string      `json:"notes"`
}

func safeAssetRelative(name string) (string, error) {
	name = strings.ReplaceAll(name, "\\", "/")
	if name == "" || strings.Contains(name, ":") || strings.HasPrefix(name, "/") {
		return "", fmt.Errorf("invalid asset reference %q", name)
	}
	for _, p := range strings.Split(name, "/") {
		if p == ".." {
			return "", fmt.Errorf("asset traversal %q", name)
		}
	}
	name = path.Clean(name)
	if name == "." {
		return "", fmt.Errorf("empty asset reference")
	}
	return name, nil
}

// ArtworkAssets lists runtime sprite dependencies, never copies/distributes art.
// Paths are relative to an installation, suitable for an external batch fetcher.
// A missing particle definition makes the list explicitly incomplete: fetch the
// JSON and rerun to discover its atlas dependencies.
func ArtworkAssets(art *scenario.Artwork, idx *datfile.Index, gameDir string, crop *Crop) (AssetManifest, error) {
	if idx == nil {
		return AssetManifest{}, fmt.Errorf("asset manifest requires a DAT")
	}
	census, err := ArtworkCensus(art, idx, StudyOptions{Crop: crop})
	if err != nil {
		return AssetManifest{}, err
	}
	r := AssetManifest{Schema: "aoe2kit.preview.assets.v1", Crop: census.Crop, Assets: []NeededAsset{}, Unresolved: []AssetIssue{}, Complete: true, Notes: []string{"Install-relative loose sprite paths; standing graphics from Gaia DAT, matching preview. No terrain textures, animation states, building annexes, graphic deltas or packed DRS extraction. Presence is not decode validation.", "Absent SLDs use the conventional resources/_common/drs/graphics path; existing files use their actual relative path. Particle-only graphics do not request a speculative legacy SLD. Fetch missing particle definitions and rerun for atlas dependencies."}}
	// One directory walk for the entire scenario, not one walk per unit kind.
	slds := map[string]string{}
	if gameDir != "" {
		root := filepath.Join(gameDir, "resources", "_common", "drs")
		err = filepath.WalkDir(root, func(p string, d fs.DirEntry, e error) error {
			if e != nil {
				if p == root && os.IsNotExist(e) {
					return nil
				}
				return e
			}
			if d.IsDir() || !strings.EqualFold(filepath.Ext(d.Name()), ".sld") {
				return nil
			}
			key := strings.ToLower(strings.TrimSuffix(d.Name(), filepath.Ext(d.Name())))
			if _, ok := slds[key]; !ok {
				rel, e := filepath.Rel(gameDir, p)
				if e != nil {
					return e
				}
				slds[key] = filepath.ToSlash(rel)
			}
			return nil
		})
		if err != nil {
			return r, err
		}
	}
	assets := map[string]*NeededAsset{}
	add := func(rel, kind string, id int) {
		a := assets[rel]
		if a == nil {
			status := "unchecked"
			if gameDir != "" {
				status = "missing"
				if st, e := os.Stat(filepath.Join(gameDir, filepath.FromSlash(rel))); e == nil && st.Mode().IsRegular() {
					status = "present"
				}
			}
			a = &NeededAsset{Path: rel, Kind: kind, Status: status}
			assets[rel] = a
		}
		if len(a.Units) == 0 || a.Units[len(a.Units)-1] != id {
			a.Units = append(a.Units, id)
		}
	}
	issue := func(id int, why string) { r.Unresolved = append(r.Unresolved, AssetIssue{id, why}); r.Complete = false }
	meta := studyMetadata(idx)
	for _, entry := range census.Units {
		id := entry.UnitID
		u, ok := meta[id]
		if !ok {
			issue(id, "unit absent from Gaia DAT")
			continue
		}
		g, ok := idx.Graphic(int(u.StandingGraphic1))
		if !ok {
			issue(id, "standing graphic absent from DAT")
			continue
		}
		particle := g.ParticleEffectName.Value
		file := g.FileName.Value
		if file != "" {
			clean, e := safeAssetRelative(file)
			if e != nil || strings.Contains(clean, "/") {
				issue(id, "invalid graphic basename")
				continue
			}
			stem := strings.TrimSuffix(clean, path.Ext(clean))
			if stem == "" {
				issue(id, "empty graphic basename")
				continue
			}
			if rel, ok := slds[strings.ToLower(stem)]; ok {
				add(rel, "sld", id)
			} else if particle == "" {
				add("resources/_common/drs/graphics/"+stem+".sld", "sld", id)
			}
		}
		if particle == "" {
			if file == "" {
				issue(id, "graphic has no SLD or particle reference")
			}
			continue
		}
		name, e := safeAssetRelative(particle + ".json")
		if e != nil {
			issue(id, e.Error())
			continue
		}
		add("resources/_common/particles/"+name, "particle_definition", id)
		if gameDir == "" {
			issue(id, "particle definition not inspected; atlas dependencies unknown")
			continue
		}
		p, e := particleAssetPath(filepath.Join(gameDir, "resources", "_common", "particles"), name)
		if e != nil {
			issue(id, "particle definition unavailable; fetch it and rerun for atlas dependencies")
			continue
		}
		var d particleDefinition
		if e := readParticleJSON(p, &d); e != nil {
			issue(id, "invalid particle definition: "+e.Error())
			continue
		}
		atlas, e := safeAssetRelative(d.AtlasFile)
		if e != nil {
			issue(id, e.Error())
			continue
		}
		add("resources/_common/particles/"+atlas, "particle_texture", id)
		add("resources/_common/particles/"+strings.TrimSuffix(atlas, path.Ext(atlas))+".json", "particle_atlas", id)
	}
	for _, a := range assets {
		r.Assets = append(r.Assets, *a)
		if a.Status == "missing" {
			r.Missing++
		}
	}
	sort.Slice(r.Assets, func(i, j int) bool { return r.Assets[i].Path < r.Assets[j].Path })
	return r, nil
}
