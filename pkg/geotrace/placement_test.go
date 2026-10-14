package geotrace

import (
	"aoe2kit/pkg/datcache"
	"aoe2kit/pkg/scenario"
	"os"
	"path/filepath"
	"testing"
)

func TestWrittenPlacementAuditWithDAT(t *testing.T) {
	dat := os.Getenv("AOE2KIT_PLACEMENT_DAT")
	if dat == "" {
		t.Skip("set AOE2KIT_PLACEMENT_DAT for real DAT audit")
	}
	if _, err := os.Stat(dat); err != nil {
		t.Skip("DAT unavailable")
	}
	dir := t.TempDir()
	blank := filepath.Join(dir, "blank.aoe2scenario")
	placed := filepath.Join(dir, "placed.aoe2scenario")
	if _, err := scenario.WriteBlankScenarioFile(blank, scenario.BlankOptions{PlayerCount: 2, HumanSlots: 1, NoStarters: true, MapWidth: 80, MapHeight: 80}); err != nil {
		t.Fatal(err)
	}
	units := []scenario.UnitRecipe{startUnit(0, 349, 10.5, 10.5), startUnit(0, 65, 10.5, 10.5), startUnit(0, 349, 20.5, 20.5), startUnit(0, 65, 30.5, 30.5)}
	r := scenario.Recipe{Units: units, Masks: []scenario.MaskRecipe{{Name: "water", Op: "cells", Cells: []scenario.MaskCellRecipe{{X: 20, Y: 20}}}}, Map: []scenario.MapRecipe{{Op: "set_terrain_mask", Mask: "water", TerrainID: intPtr(1)}}}
	if _, err := scenario.PatchRecipeFile(blank, placed, r); err != nil {
		t.Fatal(err)
	}
	p := EcologyProfile{ResourceTreeUnits: []int{349}}
	a, removals, err := AuditPlacement(placed, dat, p, nil)
	if err != nil {
		t.Fatal(err)
	}
	if a.FloraTerrain != 1 || a.FaunaBlockers != 1 || len(removals) != 2 {
		t.Fatalf("audit=%+v removals=%d", a, len(removals))
	}
	terrainOnly := filepath.Join(dir, "terrain.aoe2scenario")
	r.Units = nil
	if _, err := scenario.PatchRecipeFile(blank, terrainOnly, r); err != nil {
		t.Fatal(err)
	}
	gate, kept, err := GateRecipePlacement(terrainOnly, dat, p, nil, units)
	if err != nil || len(kept) != 2 || gate.Rejected != 2 || gate.FloraTerrain != 1 || gate.FaunaBlockers != 1 {
		t.Fatalf("gate=%+v kept=%d err=%v", gate, len(kept), err)
	}
	_, _, err = AuditPlacement(placed, dat, p, &PlayerStartReport{Mode: "explicit", Config: PlayerStartProfile{ClearRadius: 16}, Positions: []StartPosition{{X: 10.5, Y: 10.5}}})
	if err == nil {
		t.Fatal("silently removed required start inventory")
	}
	clean := filepath.Join(dir, "clean.aoe2scenario")
	if _, err := scenario.PatchRecipeFile(placed, clean, scenario.Recipe{Units: removals}); err != nil {
		t.Fatal(err)
	}
	a, remaining, err := AuditPlacement(clean, dat, p, nil)
	if err != nil || len(remaining) != 0 {
		t.Fatalf("audit=%+v remaining=%d err=%v", a, len(remaining), err)
	}
}

func TestPlacementFootprintIncludesAdjacentTile(t *testing.T) {
	o := placementObject{UnitSummary: scenario.UnitSummary{X: 10.9, Y: 20.5}, row: datcache.UnitRow{CollisionSizeX: .3, CollisionSizeY: .3}}
	seen := map[[2]int]bool{}
	placementTiles(o, func(x, y int) bool { seen[[2]int{x, y}] = true; return true })
	if len(seen) != 2 || !seen[[2]int{10, 20}] || !seen[[2]int{11, 20}] {
		t.Fatalf("footprint: %v", seen)
	}
	if placementTiles(o, func(x, y int) bool { return x != 11 }) {
		t.Fatal("adjacent blocker missed")
	}
}

func TestPlacementZeroFootprintAndBoundary(t *testing.T) {
	for _, x := range []float64{10, 10.5} {
		o := placementObject{UnitSummary: scenario.UnitSummary{X: x, Y: 20}}
		n := 0
		placementTiles(o, func(x, y int) bool {
			n++
			if x != 10 || y != 20 {
				t.Fatalf("wrong point %d,%d", x, y)
			}
			return true
		})
		if n != 1 {
			t.Fatalf("visited %d tiles", n)
		}
	}
}

func TestPlacementAuditUnavailableIsNotPass(t *testing.T) {
	a, _, err := AuditPlacement("", "", EcologyProfile{}, nil)
	if err == nil || a.State != "unavailable" {
		t.Fatalf("audit=%+v err=%v", a, err)
	}
}

func TestPlacementWaterTerrain(t *testing.T) {
	for _, id := range []int{1, 22, 23, 54, 55, 58, 59, 95, 111} {
		if !placementWaterTerrain(id) {
			t.Fatalf("water %d", id)
		}
	}
	for _, id := range []int{0, 2, 10, 53, 93, 94} {
		if placementWaterTerrain(id) {
			t.Fatalf("land %d", id)
		}
	}
}
