package compose

import (
	"bytes"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"aoe2kit/pkg/scenario"
)

func scene(t *testing.T, seed uint64) *Scene {
	t.Helper()
	s, e := New(80, 80, 0, seed)
	if e != nil {
		t.Fatal(e)
	}
	return s
}
func good(t *testing.T, s *Scene) {
	t.Helper()
	if s.Err() != nil {
		t.Fatal(s.Err())
	}
}

func TestRandomNoiseDeterministic(t *testing.T) {
	r := NewRNG(0)
	if got, want := r.Float64(), float64(uint64(1082269761)>>11)/float64(uint64(1)<<53); got != want {
		t.Fatalf("sequence changed: %g != %g", got, want)
	}
	a, b := NewRNG(77), NewRNG(77)
	for i := 0; i < 100; i++ {
		if a.Float64() != b.Float64() {
			t.Fatal("nondeterministic RNG")
		}
	}
	if Noise(0, 0, 0) != 0 {
		t.Fatal("noise origin")
	}
	for y := -5.0; y < 5; y += .25 {
		for x := -5.0; x < 5; x += .25 {
			v := FBM(x, y, 77)
			if v < 0 || v > 1 || math.IsNaN(v) {
				t.Fatal(v)
			}
		}
	}
	if math.Abs(Noise(1-.00001, .5, 7)-Noise(1+.00001, .5, 7)) > .001 {
		t.Fatal("noise discontinuity")
	}
}

func TestSceneSpacingAndFailClosed(t *testing.T) {
	s := scene(t, 1)
	s.Claim(Point{10, 10}, 2)
	if s.Free(Point{11, 10}) || !s.Free(Point{12, 10}) || s.FreeRadius(Point{12, 10}, .1) {
		t.Fatal("spacing semantics")
	}
	s.SetTerrain(1, 2, 6)
	s.SetLayer(1, 2, 32)
	s.SetElevation(1, 2, 3)
	g := s.Grid()
	g.Terrain[0] = 99
	if v, _ := s.TerrainAt(0, 0); v != 0 {
		t.Fatal("grid aliases scene")
	}
	s.Gaia(264, Point{80, 10}, 4)
	if _, e := s.Recipe(); e == nil {
		t.Fatal("invalid placement exported")
	}
	if e := s.WriteBundle(filepath.Join(t.TempDir(), "bad")); e == nil {
		t.Fatal("invalid bundle exported")
	}
	for _, bad := range []func(*Scene){func(s *Scene) { s.SetLayer(0, 0, -2) }, func(s *Scene) { s.SetTerrain(-1, 0, 0) }, func(s *Scene) { s.Claim(Point{2, 2}, math.NaN()) }, func(s *Scene) { s.Gaia(1, Point{math.NaN(), 1}, 0) }, func(s *Scene) { s.SetElevation(0, 0, 256) }} {
		s := scene(t, 0)
		bad(s)
		if s.Err() == nil {
			t.Fatal("invalid input accepted")
		}
	}
	if _, e := New(100000, 1, 0, 0); e == nil {
		t.Fatal("unbounded grid")
	}
}

func TestRoadGeometry(t *testing.T) {
	s := scene(t, 1)
	r := WoodenRoad{s, 2748}
	r.Ring(Point{20, 20})
	want := []Placement{{UnitID: 2748, X: 21, Y: 21, Variant: 12, Status: 2}, {UnitID: 2748, X: 21, Y: 19, Variant: 13, Status: 2}, {UnitID: 2748, X: 19, Y: 21, Variant: 14, Status: 2}, {UnitID: 2748, X: 19, Y: 19, Variant: 15, Status: 2}}
	if !reflect.DeepEqual(s.Units(), want) {
		t.Fatalf("ring=%+v", s.Units())
	}
	turtle := Turtle{Road: r, Next: Point{10.5, 10.5}, DX: 1}
	turtle.Forward(2)
	turtle.Turn(0, 1)
	if turtle.Next != (Point{13.5, 12.5}) {
		t.Fatalf("mouth join next=%+v", turtle.Next)
	}
	u := s.Units()
	if u[len(u)-1].Variant != 13 || u[len(u)-1].X != 13 || u[len(u)-1].Y != 11 {
		t.Fatalf("curve=%+v", u[len(u)-1])
	}
	r.DiagonalCorner(Point{40, 40}, 16, 1)
	good(t, s)
	u = s.Units()
	if len(u) != 10 || math.Abs(u[8].X-(40-.33-.685)) > 1e-9 || u[8].Variant != 5 || u[9].Variant != 7 {
		t.Fatalf("diagonal=%+v", u)
	}
	r.Band(30, 50, 3)
	good(t, s)
	u = s.Units()
	if len(u) != 20 || u[16].Variant != 9 || u[17].Variant != 8 {
		t.Fatalf("band=%+v", u[10:])
	}
	bad := scene(t, 1)
	WoodenRoad{bad, 2748}.Curve(Point{20, 20}, 0, 1)
	if bad.Err() == nil {
		t.Fatal("invalid corner accepted")
	}
}

func TestCliffsAndTerraces(t *testing.T) {
	for _, tc := range []struct {
		a      Axis
		side   int
		facets []int
	}{{AxisX, -1, []int{4, 5}}, {AxisX, 1, []int{20, 21, 22, 23}}, {AxisY, 1, []int{1, 2}}, {AxisY, -1, []int{7, 8, 9, 10, 11}}} {
		got, e := CliffFacets(tc.a, tc.side)
		if e != nil || !reflect.DeepEqual(got, tc.facets) {
			t.Fatalf("facing=%v %v", got, e)
		}
	}
	s := scene(t, 4)
	foot := s.LatticeStair(Point{10.5, 10.5}, StairOptions{UnitID: 264, Steps: 3, GapStart: -1, FlatChance: 0})
	u := s.Units()
	if len(u) != 6 || len(foot) != 3 || u[0].Variant != 15 || u[1].Variant != 3 || u[4].X != 16.5 || u[5].Y != 19.5 {
		t.Fatalf("stair=%+v", u)
	}
	gap := scene(t, 4)
	gap.LatticeStair(Point{10.5, 10.5}, StairOptions{UnitID: 264, Steps: 3, GapStart: 1, GapTerrain: 6})
	good(t, gap)
	u = gap.Units()
	if len(u) != 4 || u[2].Variant != 12 || u[3].Variant != 14 {
		t.Fatalf("gap=%+v", u)
	}
	mortar := []Brush{{2009, []int{0, 1, 2, 3}}}
	s.CliffWall(10, 45, func(x float64) Point { return Point{x, 30} }, WallOptions{UnitID: 264, Axis: AxisX, HighSide: -1, Mortar: mortar})
	cuts := s.Terrace(10, 50, func(x float64) float64 { return 50 + x/10 }, TerraceOptions{UnitID: 2651, Coverups: [][]Brush{{{2009, []int{0}}}, {{1967, []int{1}}}}})
	good(t, s)
	if len(cuts) < 2 {
		t.Fatal("missing cuts")
	}
	for _, p := range cuts {
		if p.Y != math.Round(p.Y) {
			t.Fatal("unrounded terrace level")
		}
	}
	for _, p := range s.Units() {
		if p.UnitID == 2651 && (p.Variant != 4 && p.Variant != 5) {
			t.Fatal("bad terrace facet")
		}
	}
}

func TestEveryDiagonalCorner(t *testing.T) {
	for v := 16; v <= 19; v++ {
		s := scene(t, 0)
		WoodenRoad{s, 2748}.DiagonalCorner(Point{40, 40}, v, 2)
		good(t, s)
		u := s.Units()
		if len(u) != 5 || u[0].Variant != v {
			t.Fatalf("corner %d: %+v", v, u)
		}
		for i, m := range diagonalRoadMouths[v-16] {
			for k := 0; k < 2; k++ {
				p := u[1+i*2+k]
				d := .685 + 1.37*float64(k)
				if p.Variant != m.v || math.Abs(p.X-(40+m.x+m.dx*d)) > 1e-9 || math.Abs(p.Y-(40+m.y+m.dy*d)) > 1e-9 {
					t.Fatalf("corner %d mouth %d: %+v", v, i, p)
				}
			}
		}
	}
}

func TestRiverTrackAndBundle(t *testing.T) {
	build := func() *Scene {
		s := scene(t, 12)
		s.River([]RiverPoint{{Point{10.5, 20.5}, 2}, {Point{40.5, 20.5}, 2}}, RiverOptions{Core: 23, Shallow: 1, Bank: 11, Deep: 22, BankWidth: 1, Pools: []Pool{{Point{40.5, 20.5}, 3}}})
		s.MixedTrack([]Point{{10.5, 10.5}, {30.5, 10.5}}, TrackOptions{Dry: 6, Mud: 76, HalfWidth: .8, WetThreshold: 0, MissingBoards: 0, RoadID: 2748, AvoidTerrains: []int{1, 22, 23}})
		s.WaterfallPair(Point{40.5, 20.5}, 1635, 896, 1)
		return s
	}
	a, b := build(), build()
	good(t, a)
	for _, tc := range []struct{ x, y, id int }{{20, 20, 23}, {20, 22, 11}, {40, 20, 22}, {20, 10, 76}} {
		v, _ := a.TerrainAt(tc.x, tc.y)
		if v != tc.id {
			t.Fatalf("terrain %d,%d=%d want%d", tc.x, tc.y, v, tc.id)
		}
	}
	ra, _ := a.Recipe()
	rb, _ := b.Recipe()
	ba, _ := json.Marshal(ra)
	bb, _ := json.Marshal(rb)
	if !bytes.Equal(ba, bb) {
		t.Fatal("same seed produced different recipe")
	}
	root := t.TempDir()
	dir := filepath.Join(root, "bundle")
	if e := a.WriteBundle(dir); e != nil {
		t.Fatal(e)
	}
	if e := a.WriteBundle(dir); e == nil {
		t.Fatal("overwrote bundle")
	}
	r, e := scenario.LoadRecipe(filepath.Join(dir, "recipe.json"))
	if e != nil {
		t.Fatal(e)
	}
	input, output := filepath.Join(root, "blank.aoe2scenario"), filepath.Join(root, "scene.aoe2scenario")
	if _, e := scenario.WriteBlankScenarioFile(input, scenario.BlankOptions{MapWidth: 80, MapHeight: 80, NoStarters: true}); e != nil {
		t.Fatal(e)
	}
	if _, e := scenario.PatchRecipeFile(input, output, r); e != nil {
		t.Fatal(e)
	}
	art, e := scenario.OpenArtwork(output)
	if e != nil {
		t.Fatal(e)
	}
	if art.Units.Total != len(a.Units()) || art.Tiles[20*80+20].TerrainID != 23 {
		t.Fatal("recipe readback diverged")
	}
	if _, e := os.Stat(filepath.Join(dir, "grid.json")); e != nil {
		t.Fatal(e)
	}
	bad := scene(t, 1)
	bad.River([]RiverPoint{{Point{1, 1}, 0}, {Point{2, 2}, 1}}, RiverOptions{})
	if bad.Err() == nil {
		t.Fatal("zero-width river")
	}
}

func TestTrackAvoidsWaterAndDegenerateSegments(t *testing.T) {
	s := scene(t, 0)
	s.SetTerrain(15, 10, 1)
	s.MixedTrack([]Point{{10.5, 10.5}, {10.5, 10.5}, {20.5, 10.5}}, TrackOptions{Dry: 6, Mud: 76, HalfWidth: .8, WetThreshold: 0, MissingBoards: 0, RoadID: 2748, AvoidTerrains: []int{1}})
	good(t, s)
	if v, _ := s.TerrainAt(15, 10); v != 1 {
		t.Fatal("painted water")
	}
	for _, p := range s.Units() {
		if int(p.X) == 15 && int(p.Y) == 10 {
			t.Fatal("board over excluded water")
		}
	}
}
