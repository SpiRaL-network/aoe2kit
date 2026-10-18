package preview

import (
	"os"
	"path/filepath"
	"testing"

	"aoe2kit/pkg/scenario"
)

func TestCorpusCountsMapsAndComparison(t *testing.T) {
	a, idx := studyFixture()
	c := NewCorpusCounter(idx, StudyOptions{GaiaOnly: true})
	if err := c.Add("first", a, 123); err != nil {
		t.Fatal(err)
	}
	b := &scenario.Artwork{Width: 17, Height: 17, Units: scenario.UnitInfo{Sections: []scenario.PlayerUnitsInfo{{Player: 0, Units: []scenario.UnitSummary{{UnitConst: 1, X: 1, Y: 1}, {UnitConst: 2, X: 2, Y: 2}}}}}}
	if err := c.Add("second", b, 234); err != nil {
		t.Fatal(err)
	}
	if err := c.Add("second", b, 0); err == nil {
		t.Fatal("duplicate map counted")
	}
	other := NewCorpusCounter(idx, StudyOptions{GaiaOnly: true})
	if err := other.Add("other", b, 0); err != nil {
		t.Fatal(err)
	}
	r, err := c.Report(other, 2)
	if err != nil {
		t.Fatal(err)
	}
	if r.Returned != 2 || r.Units[0].Count != 4 || r.Units[0].Maps != 2 || r.Units[0].AgainstCount != 1 || r.Units[0].AgainstMaps != 1 || r.Units[0].CountDelta != 3 || r.Units[0].OnlyInInput || r.Files[0].PeakRSSKiB != 123 {
		t.Fatalf("%+v", r)
	}
	r, err = c.Report(other, 1)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, u := range r.Units {
		if u.UnitID == 3 && u.OnlyInInput {
			found = true
		}
	}
	if !found {
		t.Fatal("missing comparison-only-in-input marker")
	}
	player := 2
	p := NewCorpusCounter(idx, StudyOptions{Player: &player})
	if err := p.Add("first", a, 0); err != nil {
		t.Fatal(err)
	}
	r, _ = p.Report(nil, 1)
	if r.Total != 1 || len(r.Units) != 1 || r.Units[0].Players[2] != 1 {
		t.Fatalf("player filter %+v", r)
	}
}

func TestScenarioPathsDeduplicateAndRejectEmpty(t *testing.T) {
	d := t.TempDir()
	p := filepath.Join(d, "one.aoe2scenario")
	if err := os.WriteFile(p, nil, 0600); err != nil {
		t.Fatal(err)
	}
	got, err := ScenarioPaths([]string{d, p})
	if err != nil || len(got) != 1 {
		t.Fatalf("%v %v", got, err)
	}
	if _, err := ScenarioPaths([]string{t.TempDir()}); err == nil {
		t.Fatal("empty corpus accepted")
	}
}

func TestDiverseHotspotsPreferRareArtAndHonorFloor(t *testing.T) {
	a, idx := studyFixture()
	a.Width = 48
	a.Height = 16
	a.Units.Sections = a.Units.Sections[:1]
	a.Units.Sections[0].Units = nil
	for i := 0; i < 40; i++ {
		a.Units.Sections[0].Units = append(a.Units.Sections[0].Units, scenario.UnitSummary{UnitConst: 1, X: 1, Y: 1})
	}
	for _, u := range []scenario.UnitSummary{{UnitConst: 1, X: 17, Y: 1}, {UnitConst: 2, X: 18, Y: 1}, {UnitConst: 3, X: 19, Y: 1}, {UnitConst: 4, X: 33, Y: 1}, {UnitConst: 5, X: 18, Y: 1}, {UnitConst: 99, X: 18, Y: 1}} {
		a.Units.Sections[0].Units = append(a.Units.Sections[0].Units, u)
	}
	h, err := ArtworkHotspots(a, idx, StudyOptions{Diverse: true, Top: 3})
	if err != nil {
		t.Fatal(err)
	}
	if h.Rank != "diverse" || len(h.Hotspots) != 2 || h.Hotspots[0].Count != 3 || h.Hotspots[0].Variety != 3 || h.Hotspots[0].Crop.X1 != 16 || h.Hotspots[0].Score <= h.Hotspots[1].Score || h.Hotspots[0].CropArg != "--crop 16,0,32,16" || h.Hotspots[0].Bounds.X2 != 20 {
		t.Fatalf("%+v", h)
	}
	wantScore := h.Hotspots[0].Score
	// Explicit targeting can find forest units even though generic study excludes them.
	h, err = ArtworkHotspots(a, idx, StudyOptions{UnitIDs: []int{5}, Cell: 16})
	if err != nil || len(h.Hotspots) != 1 || h.Hotspots[0].Count != 1 {
		t.Fatalf("targeted trees %+v %v", h, err)
	}
	crop := Crop{16, 0, 32, 16}
	cropped, err := ArtworkHotspots(a, idx, StudyOptions{Diverse: true, Crop: &crop})
	if err != nil || cropped.Hotspots[0].Score != wantScore {
		t.Fatalf("rarity changed with crop: %+v %v", cropped, err)
	}
}
