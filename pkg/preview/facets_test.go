package preview

import (
	"reflect"
	"testing"

	"aoe2kit/pkg/scenario"
)

func TestArtworkFacets(t *testing.T) {
	a, idx := studyFixture()
	a.Version = "1.59"
	idx.Graphics[0].AngleCount = 4
	idx.Graphics[0].FrameCount = 60 // Four angles, not 240 animation variants.
	r, err := ArtworkFacets(a, idx, StudyOptions{GaiaOnly: true, UnitIDs: []int{1, 2, 99}})
	if err != nil || len(r.Units) != 3 {
		t.Fatalf("%+v %v", r, err)
	}
	f := r.Units[0]
	if f.Count != 3 || !reflect.DeepEqual(f.Available, []int{0, 1, 2, 3}) || !reflect.DeepEqual(f.Placed, []int{1, 3}) || !reflect.DeepEqual(f.Missing, []int{0, 2}) || f.Complete == nil || *f.Complete {
		t.Fatalf("%+v", f)
	}
	if r.Units[2].Unknown == "" || r.Units[2].Complete != nil {
		t.Fatal("unknown metadata claimed coverage")
	}
	a.Units.Sections[0].Units = append(a.Units.Sections[0].Units, scenario.UnitSummary{UnitConst: 1, X: 4, Y: 4, Rotation: .5}, scenario.UnitSummary{UnitConst: 1, X: 5, Y: 4, Rotation: 9})
	r, err = ArtworkFacets(a, idx, StudyOptions{UnitIDs: []int{1}})
	if err != nil || len(r.Units[0].Unexpected) != 2 {
		t.Fatalf("%+v %v", r, err)
	}
	a.Version = "1.57"
	r, err = ArtworkFacets(a, idx, StudyOptions{UnitIDs: []int{1}})
	if err != nil || r.Units[0].Complete != nil || r.Units[0].Unknown == "" || len(r.Units[0].Placed) != 0 {
		t.Fatalf("legacy semantics guessed: %+v %v", r, err)
	}
}

func TestFacetsAbsentSelectionAndComplete(t *testing.T) {
	a, idx := studyFixture()
	a.Version = "1.59"
	idx.Graphics[0].AngleCount = 1
	r, err := ArtworkFacets(a, idx, StudyOptions{UnitIDs: []int{2}, Crop: &Crop{0, 0, 1, 1}})
	if err != nil || len(r.Units) != 1 || r.Units[0].Count != 0 || !reflect.DeepEqual(r.Units[0].Missing, []int{0}) {
		t.Fatalf("%+v %v", r, err)
	}
	r, err = ArtworkFacets(a, idx, StudyOptions{UnitIDs: []int{2}})
	if err != nil || r.Units[0].Complete == nil || !*r.Units[0].Complete {
		t.Fatalf("%+v %v", r, err)
	}
	if _, err := ArtworkFacets(a, nil, StudyOptions{}); err == nil {
		t.Fatal("accepted missing DAT")
	}
}
