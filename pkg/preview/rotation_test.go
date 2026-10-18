package preview

import (
	"math"
	"reflect"
	"testing"

	"aoe2kit/pkg/datfile"
)

func TestRotationModels(t *testing.T) {
	for _, c := range []struct {
		model                string
		rotation             float64
		cached, angles, want int
	}{
		{"", 3, 0, 8, 3}, {"stored-index", .5, 0, 8, 1}, {"stored-index", .5, 6, 8, 6},
		{"stored-index", 12, 0, 8, 12}, // No silent wrapping in the existing model.
		{"radians", math.Pi / 2, 6, 8, 2}, {"radians-reversed", math.Pi / 2, 6, 8, 6},
		{"radians", 2 * math.Pi, 6, 8, 0}, {"radians", -math.Pi / 2, 0, 8, 6},
		{"radians", math.Pi / 2, 6, 1, 0},
	} {
		got, err := RotationIndex(Object{Rotation: c.rotation, Variant: c.cached}, c.model, c.angles)
		if err != nil || got != c.want {
			t.Fatalf("%+v: %d %v", c, got, err)
		}
	}
	for _, v := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		if _, err := RotationIndex(Object{Rotation: v}, "radians", 8); err == nil {
			t.Fatal("nonfinite rotation accepted")
		}
	}
	if _, err := ResolveRotationModel("guess"); err == nil {
		t.Fatal("unknown model accepted")
	}
}

func TestParticleRotationCandidateUsesDefinitionAngles(t *testing.T) {
	root, idx := spriteFixture(t)
	particleFixture(t, root, "")
	idx.Graphics = append(idx.Graphics, datfile.Graphic{Present: true, AngleCount: 8, ParticleEffectName: datfile.DebugString{Value: "test"}})
	idx.Civs[0].Units = append(idx.Civs[0].Units, datfile.UnitSummary{Present: true, Index: 8, StandingGraphic1: 1})
	s := spriteSnapshot()
	s.Objects = []Object{{UnitID: 8, X: 5, Y: 5, Rotation: math.Pi, Variant: 3}}
	_, r, err := RenderSprites(s, Options{Scale: 4, RotationModel: "radians"}, idx, root)
	if err != nil || len(r.Particles) != 1 || r.Particles[0].Angle != 1 || r.Particles[0].Status != "drawn" {
		t.Fatalf("%+v %v", r, err)
	}
}

func TestRotationModelRenderAndSnapshotPreservation(t *testing.T) {
	root, idx := spriteFixture(t)
	s := spriteSnapshot()
	s.ScenarioVersion = "1.57"
	s.Objects[0].Rotation = 2 * math.Pi
	s.Objects[0].Variant = 6
	before := s.Objects[0]
	_, direct, err := RenderSprites(s, Options{Scale: 8}, idx, root)
	if err != nil || direct.Drawn != 0 {
		t.Fatalf("direct: %+v %v", direct, err)
	}
	_, candidate, err := RenderSprites(s, Options{Scale: 8, RotationModel: "radians"}, idx, root)
	if err != nil || candidate.Drawn != 1 || candidate.RotationModel != "radians" || candidate.ScenarioVersion != "1.57" {
		t.Fatalf("candidate: %+v %v", candidate, err)
	}
	if !reflect.DeepEqual(before, s.Objects[0]) {
		t.Fatal("render rewrote the snapshot")
	}
	if _, _, err := RenderSprites(s, Options{RotationModel: "guess"}, idx, root); err == nil {
		t.Fatal("invalid model rendered")
	}
}
