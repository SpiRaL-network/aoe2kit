package geotrace

import "testing"

func TestMapDiagnosticsRequireDATAndRespectTerrainPairs(t *testing.T) {
	land, water := true, false
	r := Report{GridSize: 8}
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			c := Cell{X: x, Y: y, Land: &land, TerrainID: 2}
			if x >= 4 {
				c.Land = &water
				c.TerrainID = 1
			}
			r.Cells = append(r.Cells, c)
		}
	}
	absent := measureMapDiagnostics(r, DockConstraints{State: "unavailable"})
	if absent.DATCandidateDockFootprints != 0 || absent.ShorePairs[0].AllowedByDAT {
		t.Fatal("missing DAT became a pass")
	}
	d := measureMapDiagnostics(r, DockConstraints{State: "dat_filtered_candidates", AllowedTerrains: []int{1, 2}, FootprintTiles: 3})
	if d.SourceLandComponents != 1 || d.SourceWaterComponents != 1 || d.LargestLandComponent != 32 || d.EdgeConnectedLandCells != 32 || d.LandBoundsFill != 1 {
		t.Fatalf("wrong connectivity %+v", d)
	}
	if len(d.ShorePairs) != 1 || d.ShorePairs[0].Edges != 8 || !d.ShorePairs[0].AllowedByDAT || d.DATCandidateDockFootprints != 6 {
		t.Fatalf("wrong shore diagnostics %+v", d)
	}
	blocked := measureMapDiagnostics(r, DockConstraints{State: "dat_filtered_candidates", AllowedTerrains: []int{1}, FootprintTiles: 3})
	if blocked.DATCandidateDockFootprints != 0 || blocked.ShorePairs[0].AllowedByDAT {
		t.Fatal("disallowed land terrain accepted")
	}
}

func TestUnknownLandmaskDoesNotInventConnectivity(t *testing.T) {
	r := Report{GridSize: 2, Cells: make([]Cell, 4)}
	d := measureMapDiagnostics(r, DockConstraints{State: "unavailable"})
	if d.SourceLandComponents != 0 || d.SourceWaterComponents != 0 || len(d.ShorePairs) != 0 {
		t.Fatal("unknown mask treated as water")
	}
}
