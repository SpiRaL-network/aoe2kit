package geotrace

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestFeltSelectionStandingWeightsAndRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "felt.json")
	raw := `{"schema":"aoe2-art-felt-compendium.v1","entries":[
 {"unit_const":101,"form":"plant","function":["undergrowth"],"placement_hint":"scatter","standing":"ai_felt_from_render"},
 {"unit_const":101,"form":"plant","function":["undergrowth"],"placement_hint":"waterside","standing":"human_felt"},
 {"unit_const":102,"form":"plant","function":["undergrowth"],"placement_hint":"scatter","standing":"ai_felt_from_render"},
 {"unit_const":103,"form":"rock","function":["rock"],"placement_hint":"landmark","standing":"ai_felt_from_render"}]}`
	if err := os.WriteFile(path, []byte(raw), 0600); err != nil {
		t.Fatal(err)
	}
	p := defaultEcologyProfile()
	p.Layers = []HabitatLayer{{Name: "mixed", Role: "decorative", Rate: 1, Limit: 100, Felt: []FeltSelection{{Functions: []string{"undergrowth"}, Weight: 2}, {Functions: []string{"rock"}, Weight: 1}}}}
	if err := resolveFeltLayers(&p, path); err != nil {
		t.Fatal(err)
	}
	if len(p.CompendiumSHA256) != 64 || len(p.Layers[0].Units) != 3 {
		t.Fatalf("bad resolution: %+v", p)
	}
	for _, u := range p.Layers[0].Units {
		if u.Weight != 1 {
			t.Fatalf("family weighting: %+v", u)
		}
	}
	land := true
	c := Cell{Land: &land, TerrainID: 0, DistanceToShoreTiles: 20, OutsideTrace: true}
	if !matchesLayer(c, p.Layers[0]) {
		t.Fatal("nominal bbox still clips projected land")
	}
	if got := habitatUnits(c, p.Layers[0]); len(got) != 2 {
		t.Fatalf("human waterside hint lost: %+v", got)
	}
	data, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	var copy EcologyProfile
	if err := json.Unmarshal(data, &copy); err != nil {
		t.Fatal(err)
	}
	if got := habitatUnits(c, copy.Layers[0]); len(got) != 2 {
		t.Fatal("report round-trip lost habitat restrictions")
	}
	if err := resolveFeltLayers(&copy, ""); err == nil {
		t.Fatal("missing source silently accepted")
	}
}

func TestFeltPlacementHints(t *testing.T) {
	land, water := true, false
	cases := []struct {
		hint string
		cell Cell
	}{
		{"coast", Cell{Land: &land, DistanceToShoreTiles: 2}},
		{"waterside", Cell{Land: &land, DistanceToShoreTiles: 20, LandcoverKey: "90_herbaceous_wetland"}},
		{"on_water", Cell{Land: &water, DistanceToShoreTiles: 2}},
		{"avenue", Cell{Land: &land, Roadside: true}},
		{"carpet", Cell{Land: &land}},
	}
	for _, tc := range cases {
		l := HabitatLayer{Units: []WeightedUnit{{101, 1}}, FeltEntries: map[int]feltEntry{101: {PlacementHint: tc.hint}}}
		if len(habitatUnits(tc.cell, l)) != 1 {
			t.Errorf("hint %s rejected", tc.hint)
		}
	}
	l := HabitatLayer{Units: []WeightedUnit{{101, 1}}, FeltEntries: map[int]feltEntry{101: {PlacementHint: "on_water"}}}
	if len(habitatUnits(Cell{Land: &land, TerrainID: 111}, l)) != 0 {
		t.Fatal("wetland land classification is not proof of water terrain")
	}
}

func TestWoodyProfileRetainsAdditionalTaxa(t *testing.T) {
	path := filepath.Join(t.TempDir(), "atlas.csv")
	if err := os.WriteFile(path, []byte("LATITUDE,LONGITUDE,Sabal palmetto,Rhizophora mangle\n26,-81,1,1\n"), 0600); err != nil {
		t.Fatal(err)
	}
	p := defaultEcologyProfile()
	p.SpeciesUnits = map[string][]int{"Sabal palmetto": {351}, "Rhizophora mangle": {1144}}
	g, _, err := loadUSGSWoodyWithProfile(path, p)
	if err != nil {
		t.Fatal(err)
	}
	got := p.woodyUnitChoices(g.nearestSpecies(26, -81))
	if len(got) != 2 || !member(got, 351) || !member(got, 1144) {
		t.Fatalf("taxa lost: %v", got)
	}
}
