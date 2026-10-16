package main

import "testing"

func TestArtworkStudyFlags(t *testing.T) {
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
