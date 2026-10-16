package preview

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"aoe2kit/pkg/datfile"
	"aoe2kit/pkg/scenario"
)

func studyFixture() (*scenario.Artwork, *datfile.Index) {
	a := &scenario.Artwork{Width: 17, Height: 17, Units: scenario.UnitInfo{Sections: []scenario.PlayerUnitsInfo{
		{Player: 0, Units: []scenario.UnitSummary{
			{UnitConst: 1, X: 1, Y: 1, Rotation: 3}, {UnitConst: 1, X: 2, Y: 1, Rotation: 1}, {UnitConst: 1, X: 3, Y: 1, Rotation: 1},
			{UnitConst: 2, X: 9, Y: 1}, {UnitConst: 3, X: 10, Y: 1}, {UnitConst: 4, X: 11, Y: 1},
			{UnitConst: 5, X: 1, Y: 1}, {UnitConst: 6, X: 1, Y: 1}, {UnitConst: 7, X: 1, Y: 1}, {UnitConst: 8, X: 1, Y: 1}, {UnitConst: 9, X: 1, Y: 1},
			{UnitConst: 99, X: 16, Y: 16}, {UnitConst: 1, X: -1, Y: 0}, {UnitConst: 1, X: 17, Y: 1},
		}},
		{Player: 2, Units: []scenario.UnitSummary{{UnitConst: 1, X: 1, Y: 1}}},
	}}}
	idx := &datfile.Index{Civs: []datfile.Civ{{}}}
	for id := 1; id <= 9; id++ {
		class := int16(11)
		if id >= 5 {
			class = []int16{15, 5, 31, 33, 63}[id-5]
		}
		idx.Civs[0].Units = append(idx.Civs[0].Units, datfile.UnitSummary{Index: id, Present: true, Name: "TEST", Class: class, StandingGraphic1: 0})
	}
	idx.Graphics = []datfile.Graphic{{Index: 0, Present: true, FileName: datfile.DebugString{Value: "TEST"}}}
	return a, idx
}

func TestArtworkCensusAndHotspotRanking(t *testing.T) {
	a, idx := studyFixture()
	c := Crop{0, 0, 8, 8}
	r, err := ArtworkCensus(a, idx, StudyOptions{Crop: &c, GaiaOnly: true})
	if err != nil || r.Total != 8 || r.Distinct != 6 || r.Units[0].Count != 3 || !reflect.DeepEqual(r.Units[0].Rotations, []RotationCount{{1, 2}, {3, 1}}) {
		t.Fatalf("census=%+v err=%v", r, err)
	}
	h, err := ArtworkHotspots(a, idx, StudyOptions{GaiaOnly: true, Top: 2})
	if err != nil || h.ExcludedForestFish != 5 || h.MatchedCells != 3 || len(h.Hotspots) != 2 || h.Hotspots[0].Crop.X1 != 8 || h.Hotspots[0].Variety != 3 {
		t.Fatalf("hotspots=%+v err=%v", h, err)
	}
	// Density wins after adding one copy, while variety still prefers the other cell.
	a.Units.Sections[0].Units = append(a.Units.Sections[0].Units, scenario.UnitSummary{UnitConst: 1, X: 4, Y: 1})
	h, err = ArtworkHotspots(a, idx, StudyOptions{GaiaOnly: true, Rank: "count", Top: 1})
	if err != nil || h.Hotspots[0].Count != 4 || h.Hotspots[0].Crop.X1 != 0 {
		t.Fatalf("density=%+v %v", h, err)
	}
	h, err = ArtworkHotspots(a, idx, StudyOptions{GaiaOnly: true, IncludeForestFish: true, Top: 100})
	if err != nil || h.ExcludedForestFish != 0 || h.Hotspots[0].Variety != 6 || h.Hotspots[2].Units[0].Known || h.Hotspots[2].Crop.X2 != 17 {
		t.Fatalf("unfiltered=%+v %v", h, err)
	}
	b1, _ := json.Marshal(h)
	h2, _ := ArtworkHotspots(a, idx, StudyOptions{GaiaOnly: true, IncludeForestFish: true, Top: 100})
	b2, _ := json.Marshal(h2)
	if string(b1) != string(b2) {
		t.Fatal("unstable result order")
	}
	for _, opts := range []StudyOptions{{Cell: -1}, {Top: -1}, {Rank: "mystery"}, {Crop: &Crop{-1, 0, 8, 8}}} {
		if _, err := ArtworkHotspots(a, idx, opts); err == nil {
			t.Fatalf("accepted %+v", opts)
		}
	}
	if _, err := ArtworkHotspots(a, nil, StudyOptions{}); err == nil {
		t.Fatal("unclassified filtering accepted")
	}
}

func writeStudyAsset(t *testing.T, root, rel, body string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
}

func TestArtworkAssetManifest(t *testing.T) {
	a, idx := studyFixture()
	c := Crop{0, 0, 8, 8}
	root := t.TempDir()
	r, err := ArtworkAssets(a, idx, root, &c)
	if err != nil || !r.Complete || len(r.Assets) != 1 || r.Missing != 1 || len(r.Assets[0].Units) != 6 || r.Assets[0].Path != "resources/_common/drs/graphics/TEST.sld" {
		t.Fatalf("manifest=%+v %v", r, err)
	}
	writeStudyAsset(t, root, "resources/_common/drs/alternate/Test.SLD", "synthetic presence only")
	r, err = ArtworkAssets(a, idx, root, &c)
	if err != nil || r.Missing != 0 || r.Assets[0].Status != "present" || r.Assets[0].Path != "resources/_common/drs/alternate/Test.SLD" {
		t.Fatalf("installed=%+v %v", r, err)
	}
	r, err = ArtworkAssets(a, idx, "", &c)
	if err != nil || r.Assets[0].Status != "unchecked" {
		t.Fatalf("unchecked=%+v %v", r, err)
	}
	idx.Graphics[0].FileName.Value = "../private"
	r, err = ArtworkAssets(a, idx, root, &c)
	if err != nil || r.Complete || len(r.Assets) != 0 {
		t.Fatalf("unsafe=%+v %v", r, err)
	}
}

func TestArtworkParticleDependencyDiscovery(t *testing.T) {
	a, idx := studyFixture()
	c := Crop{0, 0, 8, 8}
	root := t.TempDir()
	idx.Graphics[0].ParticleEffectName.Value = "effect"
	r, err := ArtworkAssets(a, idx, root, &c)
	if err != nil || r.Complete || len(r.Assets) != 1 || r.Assets[0].Kind != "particle_definition" {
		t.Fatalf("missing definition=%+v %v", r, err)
	}
	writeStudyAsset(t, root, "resources/_common/particles/effect.json", `{"AtlasFile":"textures\\atlas.dds"}`)
	r, err = ArtworkAssets(a, idx, root, &c)
	if err != nil || !r.Complete || len(r.Assets) != 3 || r.Missing != 2 {
		t.Fatalf("resolved=%+v %v", r, err)
	}
	for _, asset := range r.Assets {
		if strings.Contains(asset.Path, "\\") || filepath.IsAbs(asset.Path) {
			t.Fatalf("not portable: %+v", asset)
		}
	}
	writeStudyAsset(t, root, "resources/_common/particles/effect.json", `{"AtlasFile":"../escape.dds"}`)
	r, err = ArtworkAssets(a, idx, root, &c)
	if err != nil || r.Complete || len(r.Assets) != 1 {
		t.Fatalf("unsafe atlas=%+v %v", r, err)
	}
}
