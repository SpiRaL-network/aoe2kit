package main

import (
	"encoding/json"
	"testing"

	"aoe2kit/pkg/scenario"
)

func TestArtworkUnitsReportPreservesShapeAndScope(t *testing.T) {
	for _, version := range []string{"1.36", "1.47", "1.51", "1.54", "1.55", "1.57", "1.59"} {
		for _, named := range []bool{false, true} {
			art := &scenario.Artwork{Version: version, Scope: "terrain_units_only_read_only", Units: scenario.UnitInfo{
				Total: 1, NumberOfPlayers: 1, NumberOfUnitSections: 1,
				Sections: []scenario.PlayerUnitsInfo{{Player: 0, Count: 1, Units: []scenario.UnitSummary{{UnitConst: 349, X: 1.25, Y: 2.5, Z: .75, Rotation: 3, Status: 2}}}},
			}}
			b, err := json.Marshal(artworkUnitsReport(art, named))
			if err != nil {
				t.Fatal(err)
			}
			var report struct {
				scenario.UnitInfo
				Version  string `json:"version"`
				Scope    string `json:"scope"`
				Rotation string `json:"rotation_interpretation"`
			}
			if err := json.Unmarshal(b, &report); err != nil {
				t.Fatal(err)
			}
			if report.Version != version || report.Scope != art.Scope || report.Total != 1 || len(report.Sections) != 1 || report.Sections[0].Units[0] != art.Units.Sections[0].Units[0] {
				t.Fatalf("version=%s named=%v report=%s", version, named, b)
			}
			want := "stored_value"
			if version < "1.58" {
				want = "stored_value_visual_semantics_not_engine_verified_for_version"
			}
			if report.Rotation != want {
				t.Fatalf("rotation: %s", b)
			}
		}
	}
}

func TestArtworkStudyFlags(t *testing.T) {
	facets, facetErr := parseArtworkStudyArgs("facets", []string{"map", "--unit", "1,2", "--gaia", "--json"})
	if facetErr != nil || len(facets.options.UnitIDs) != 2 || !facets.options.GaiaOnly {
		t.Fatalf("%+v %v", facets, facetErr)
	}
	fetch, err := parseArtworkStudyArgs("assets", []string{"map.aoe2scenario", "--missing-only", "--fetch", "--from", "game", "--to", "mirror"})
	if err != nil || !fetch.fetch || fetch.from != "game" || fetch.to != "mirror" {
		t.Fatalf("%+v %v", fetch, err)
	}
	for _, args := range [][]string{{"map", "--fetch"}, {"map", "--from", "game"}, {"map", "--fetch", "--from", "game", "--to", "mirror", "--paths"}} {
		if _, err := parseArtworkStudyArgs("assets", args); err == nil {
			t.Fatalf("accepted %v", args)
		}
	}
	a, e := parseArtworkStudyArgs("hotspots", []string{"map.aoe2scenario", "--gaia=false", "--cell", "12", "--top", "5", "--rank", "count", "--crop", "1,2,13,14"})
	if e != nil || a.options.GaiaOnly || a.options.Cell != 12 || a.options.Top != 5 || a.options.Crop.Y2 != 14 {
		t.Fatalf("%+v %v", a, e)
	}
	a, e = parseArtworkStudyArgs("assets", []string{"map.aoe2scenario", "--paths", "--missing-only"})
	if e != nil || !a.paths || !a.missing {
		t.Fatalf("%+v %v", a, e)
	}
	for _, args := range [][]string{nil, {"map", "--crop", "1,2"}, {"map", "--crop", "a,2,3,4"}, {"map", "--unknown"}, {"map", "stray"}} {
		if _, e := parseArtworkStudyArgs("census", args); e == nil {
			t.Fatalf("accepted %v", args)
		}
	}
}

func TestCorpusAndDiverseOptions(t *testing.T) {
	a, e := parseCorpusArgs([]string{"maps", "second.aoe2scenario", "--gaia", "--min-maps", "3", "--against", "other", "--verbose"})
	if e != nil || len(a.inputs) != 2 || !a.verbose || a.minMaps != 3 || a.against != "other" {
		t.Fatalf("%+v %v", a, e)
	}
	for _, args := range [][]string{{"maps", "--gaia", "--player", "2"}, {"maps", "--min-maps", "0"}, {"maps", "--player", "-1"}} {
		if _, err := parseCorpusArgs(args); err == nil {
			t.Fatalf("accepted %v", args)
		}
	}
	b, e := parseArtworkStudyArgs("hotspots", []string{"map", "--diverse", "--min-count", "5", "--top", "3", "--json"})
	if e != nil || b.options.Cell != 16 || !b.options.Diverse || b.options.MinCount != 5 || b.options.Top != 3 {
		t.Fatalf("%+v %v", b, e)
	}
	b, e = parseArtworkStudyArgs("hotspots", []string{"map", "--unit", "1062,1888", "--cell", "20"})
	if e != nil || len(b.options.UnitIDs) != 2 || b.options.Cell != 20 {
		t.Fatalf("%+v %v", b, e)
	}
	if _, err := parseArtworkStudyArgs("hotspots", []string{"map", "--diverse", "--unit", "1"}); err == nil {
		t.Fatal("mixed diversity and target selector accepted")
	}
}
