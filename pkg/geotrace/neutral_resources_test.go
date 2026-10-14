package geotrace

import (
	"math"
	"reflect"
	"testing"
)

func TestNeutralResourcesScaleAndAvoidStarts(t *testing.T) {
	a, b := startTestReport(160), startTestReport(160)
	for _, r := range []*Report{&a, &b} {
		r.PlayerStarts = &PlayerStartReport{Positions: []StartPosition{{X: 80.5, Y: 80.5}}}
		r.EcologyProfile.NeutralResources = []NeutralResourceProfile{{Name: "gold", Unit: 66, PileSize: 4, GroupsPer10000: 2, Spacing: 32, StartDistance: 32, Provenance: "test policy"}}
		if err := applyNeutralResources(r); err != nil {
			t.Fatal(err)
		}
	}
	if !reflect.DeepEqual(a.Recipe, b.Recipe) {
		t.Fatal("nondeterministic mines")
	}
	d := a.NeutralResources[0]
	if d.RequestedGroups != 5 || d.EmittedGroups != 5 || d.Objects != 20 {
		t.Fatalf("%+v", d)
	}
	for i, c := range d.Centres {
		if math.Hypot(c.X-80.5, c.Y-80.5) < 32 {
			t.Fatal("neutral mine encroaches on start")
		}
		for _, old := range d.Centres[:i] {
			if math.Hypot(c.X-old.X, c.Y-old.Y) < 32 {
				t.Fatal("mine spacing")
			}
		}
	}
	for _, u := range a.Recipe.Units {
		k := int(*u.Y)*160 + int(*u.X)
		if !*a.Cells[k].Land {
			t.Fatal("mine on water")
		}
	}
}

func TestNeutralResourcePileIsAtomicAfterGate(t *testing.T) {
	r := startTestReport(160)
	r.EcologyProfile.NeutralResources = []NeutralResourceProfile{{Name: "gold", Unit: 66, PileSize: 4, GroupsPer10000: 2, Spacing: 32, StartDistance: 32, Provenance: "test policy"}}
	if err := applyNeutralResources(&r); err != nil {
		t.Fatal(err)
	}
	r.Recipe.Units = r.Recipe.Units[1:]
	r.PlacementGate = &PlacementAudit{Rejected: 1}
	FinalizeNeutralResources(&r)
	d := r.NeutralResources[0]
	if d.EmittedGroups != 4 || d.RejectedGroups != 1 || d.Objects != 16 || len(r.Recipe.Units) != 16 || r.PlacementGate.Rejected != 4 {
		t.Fatalf("pile=%+v units=%d gate=%+v", d, len(r.Recipe.Units), r.PlacementGate)
	}
}
