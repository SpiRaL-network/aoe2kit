package geotrace

import "testing"

func TestWaterEvidencePrecedence(t *testing.T) {
	mapping, err := loadFeltMapping("")
	if err != nil {
		t.Fatal(err)
	}
	land, water := true, false
	permanent := &LandcoverSample{Valid: true, Class: 80, Key: "80_permanent_water", Fraction: 1}
	forest := &LandcoverSample{Valid: true, Class: 10, Key: "10_tree_cover", Fraction: 1}
	for _, tc := range []struct {
		name       string
		land       *bool
		inland     bool
		cover      *LandcoverSample
		wantInland bool
	}{
		{"ocean is not freshwater", &water, false, permanent, false},
		{"landcover lake on land", &land, false, permanent, true},
		{"polygon beats forest", &land, true, forest, true},
		{"ocean beats forest", &water, false, forest, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := classifyCell(Cell{}, 0, tc.land, 2, "temperate", mapping, tc.inland, "", tc.cover)
			if c.Land == nil || *c.Land || c.InlandWater != tc.wantInland {
				t.Fatalf("wrong water classification %+v", c)
			}
			if c.ElevationBand == "landcover_10_tree_cover" {
				t.Fatal("ocean/polygon painted forest")
			}
			if (waterBand(c) == "lake") != tc.wantInland {
				t.Fatal("wrong fish habitat")
			}
			recipe := BuildRecipe(Report{GridSize: 4, Cells: []Cell{c}})
			for _, op := range recipe.Map {
				if op.Mask == "landcover_10_tree_cover" {
					t.Fatal("landcover repaint overrides water")
				}
			}
		})
	}
}
