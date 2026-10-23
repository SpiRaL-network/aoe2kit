package preview

import (
	"math"
	"testing"
)

func TestDiamondRegistrationIndependentHoldout(t *testing.T) {
	as := []RegistrationAnchor{
		{X: 1, Y: 2, Height: 3}, {X: 8, Y: 4, Height: 6},
		{X: 3, Y: 12, Height: 15}, {X: 11, Y: 9, Height: 14, Holdout: true},
	}
	for i := range as {
		a := &as[i]
		a.PX = -120 + (a.X+a.Y)*48
		a.PY = 880 + (a.Y-a.X-a.Height)*24
	}
	// A wrong holdout height must remain visible, not shift the fitted camera.
	as[3].Height--
	got, err := FitDiamondRegistration(as)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(got.Scale-96) > 1e-9 || math.Abs(got.OX+120) > 1e-9 ||
		math.Abs(got.OY-880) > 1e-9 || got.FitRMS > 1e-9 ||
		math.Abs(got.Residuals[3].DY-24) > 1e-9 {
		t.Fatalf("unexpected fit: %+v", got)
	}
}

func TestDiamondRegistrationRejectsInvalidAnchors(t *testing.T) {
	for _, as := range [][]RegistrationAnchor{
		nil,
		{{}, {}, {}},
		{{}, {X: 1}, {X: 2, PX: math.NaN()}},
		{{}, {X: 1, PX: -1}, {X: 2, PX: -2}},
		{{}, {X: 1}, {X: 2, Holdout: true}},
	} {
		if _, err := FitDiamondRegistration(as); err == nil {
			t.Fatalf("accepted invalid anchors: %+v", as)
		}
	}
}
