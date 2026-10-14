package geotrace

import "testing"

func TestCoastRefinement(t *testing.T) {
	mask := []bool{false, true, false, true, false, false}
	samples := []LandcoverSample{{Valid: true, Class: 95, PixelMeters: 20}, {Valid: true, Class: 80, PixelMeters: 20}, {Class: 0}, {Valid: true, Class: 80, PixelMeters: 30}, {Valid: true, Class: 255, PixelMeters: 20}, {Valid: true, Class: 50, PixelMeters: 20}}
	d := refineCoastMask(mask, samples, "worldcover", 64)
	want := []bool{true, false, false, true, false, true}
	for i, v := range want {
		if mask[i] != v {
			t.Fatalf("cell %d: %v", i, mask)
		}
	}
	if d.WaterToLand != 2 || d.LandToWater != 1 || d.Fallback != 3 {
		t.Fatal(d)
	}
	for _, provider := range []string{"worldcover", "glc-fcs30d"} {
		m := []bool{false}
		d = refineCoastMask(m, samples, provider, 2000)
		if d.Enabled || m[0] {
			t.Fatal(d)
		}
	}
	if refineWorldCoverCoast("glc-fcs30d", 64) {
		t.Fatal("unverified provider enabled")
	}
}

func TestCoarseCoastPreservesMixedLand(t *testing.T) {
	s := []LandcoverSample{
		{Valid: true, Class: 80, PixelMeters: 74, Histogram: []LandcoverStat{{Class: 80, Weight: 2.0 / 3}, {Class: 95, Key: "95_mangrove", Weight: 1.0 / 3}}},
		{Valid: true, Class: 80, PixelMeters: 74, Histogram: []LandcoverStat{{Class: 80, Weight: 8.0 / 9}, {Class: 10, Weight: 1.0 / 9}}},
		{Valid: true, Class: 80, PixelMeters: 74, Histogram: []LandcoverStat{{Class: 80, Weight: 1}}},
	}
	m := []bool{false, false, true}
	d := refineCoastMask(m, s, "worldcover", 982)
	if worldCoverInlandWater(true, s[0]) || !worldCoverInlandWater(true, s[2]) {
		t.Fatal("coastal mixed land confused with inland water")
	}
	if !m[0] || !m[1] || !m[2] || d.WaterToLand != 2 || d.LandToWater != 0 {
		t.Fatal(m, d)
	}
	c := classifyCell(Cell{}, 0, &m[0], 1, "tropical", FeltMapping{}, false, "", &s[0])
	if c.Land == nil || !*c.Land || c.InlandWater || c.LandcoverClass != 95 || c.SourceLandcoverClass != 80 {
		t.Fatal(c)
	}
	c = classifyCell(Cell{}, 0, &m[0], 1, "tropical", FeltMapping{}, true, "lake", &s[0])
	if c.Land == nil || *c.Land || !c.InlandWater {
		t.Fatal(c)
	}
}
