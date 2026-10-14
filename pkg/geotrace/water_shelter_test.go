package geotrace

import "testing"

func TestWaterShelterDistinguishesBayFromOpenCoast(t *testing.T) {
	r := startTestReport(100)
	for i := range r.Cells {
		c := &r.Cells[i]
		land := c.X < 45
		c.Land = &land
	}
	classifyShelteredWater(&r)
	if r.Cells[50*100+50].ShelteredWater {
		t.Fatal("flat open coast classified as bay")
	}
	for i := range r.Cells {
		c := &r.Cells[i]
		land := c.Y < 40 || c.Y > 60
		c.Land = &land
	}
	classifyShelteredWater(&r)
	if !r.Cells[50*100+50].ShelteredWater || waterBand(r.Cells[50*100+50]) != "bay" {
		t.Fatal("enclosed channel missed")
	}
	r.Cells[50*100+50].InlandWater = true
	if waterBand(r.Cells[50*100+50]) != "lake" {
		t.Fatal("freshwater precedence")
	}
}

func TestWaterShelterFourBanksNeedsOpposition(t *testing.T) {
	r := startTestReport(100)
	for i := range r.Cells {
		land := false
		r.Cells[i].Land = &land
	}
	for _, p := range [][2]int{{60, 50}, {40, 50}, {50, 40}, {60, 60}} {
		land := true
		r.Cells[p[1]*100+p[0]].Land = &land
	}
	classifyShelteredWater(&r)
	if !r.Cells[50*100+50].ShelteredWater {
		t.Fatal("opposing banks missed")
	}
	for i := range r.Cells {
		land := false
		r.Cells[i].Land = &land
	}
	for _, p := range [][2]int{{60, 50}, {60, 60}, {50, 60}, {40, 60}} {
		land := true
		r.Cells[p[1]*100+p[0]].Land = &land
	}
	classifyShelteredWater(&r)
	if r.Cells[50*100+50].ShelteredWater {
		t.Fatal("one-sided coastline treated as enclosed")
	}
}

func TestDefaultEcologyHasShelteredWaterFish(t *testing.T) {
	for _, l := range defaultHabitatLayers(defaultEcologyProfile(), true) {
		if l.WaterBand == "bay" && l.Role == "fish" && len(l.Units) > 0 {
			return
		}
	}
	t.Fatal("sheltered water has no default fish layer")
}
