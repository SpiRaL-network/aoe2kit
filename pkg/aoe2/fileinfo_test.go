package aoe2

import "testing"

func TestKindForPath(t *testing.T) {
	cases := map[string]string{
		"map.aoe2scenario":         "scenario",
		"game.aoe2record":          "record",
		"empires2_x2_p1.dat":       "dat",
		"SDSNarrator.per2":         "ai",
		"u_elite_eagle.sld":        "gfx",
		"resources/en/strings.txt": "text",
	}
	for path, want := range cases {
		if got := KindForPath(path); got != want {
			t.Fatalf("KindForPath(%q)=%q, want %q", path, got, want)
		}
	}
}
