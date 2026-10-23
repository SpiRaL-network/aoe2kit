package preview

import (
	"fmt"
	"math"
)

// RegistrationAnchor pairs a world position and explicit ground height with an
// observed screenshot point. Height uses stored levels, not editor UI levels.
// Holdout points are evaluated but do not influence the fitted camera.
type RegistrationAnchor struct {
	X, Y, Height float64
	PX, PY       float64
	Holdout      bool `json:",omitempty"`
}

type RegistrationResidual struct {
	DX, DY, Distance float64
	Holdout          bool
}

// DiamondRegistration fits the rotate270 diamond convention used by preview:
// px = OX + (x+y)*Scale/2; py = OY + (y-x-height)*Scale/4.
// It does not establish that the supplied heights or ground model are correct.
type DiamondRegistration struct {
	Scale, OX, OY float64
	FitRMS        float64
	FitCount      int
	Residuals     []RegistrationResidual
}

// FitDiamondRegistration estimates one scale and two translations by least
// squares. At least three fitting anchors with distinct projected coordinates
// are required. Use independent holdouts to test a candidate height model.
func FitDiamondRegistration(anchors []RegistrationAnchor) (DiamondRegistration, error) {
	var out DiamondRegistration
	var ma, mb, mx, my float64
	for _, a := range anchors {
		for _, v := range []float64{a.X, a.Y, a.Height, a.PX, a.PY} {
			if math.IsNaN(v) || math.IsInf(v, 0) {
				return out, fmt.Errorf("registration anchors must be finite")
			}
		}
		if a.Holdout {
			continue
		}
		out.FitCount++
		ma += (a.X + a.Y) / 2
		mb += (a.Y - a.X - a.Height) / 4
		mx += a.PX
		my += a.PY
	}
	if out.FitCount < 3 {
		return out, fmt.Errorf("registration needs at least three fitting anchors")
	}
	n := float64(out.FitCount)
	ma, mb, mx, my = ma/n, mb/n, mx/n, my/n
	var num, den float64
	for _, a := range anchors {
		if a.Holdout {
			continue
		}
		u, v := (a.X+a.Y)/2-ma, (a.Y-a.X-a.Height)/4-mb
		num += u*(a.PX-mx) + v*(a.PY-my)
		den += u*u + v*v
	}
	if den <= 1e-12 {
		return out, fmt.Errorf("registration anchors have degenerate projected coordinates")
	}
	out.Scale = num / den
	out.OX, out.OY = mx-out.Scale*ma, my-out.Scale*mb
	if out.Scale <= 0 || math.IsInf(out.Scale, 0) || math.IsNaN(out.Scale) ||
		math.IsInf(out.OX, 0) || math.IsInf(out.OY, 0) || math.IsNaN(out.OX) || math.IsNaN(out.OY) {
		return DiamondRegistration{}, fmt.Errorf("registration did not produce a finite positive camera scale")
	}
	for _, a := range anchors {
		dx := out.OX + (a.X+a.Y)*out.Scale/2 - a.PX
		dy := out.OY + (a.Y-a.X-a.Height)*out.Scale/4 - a.PY
		d := math.Hypot(dx, dy)
		out.Residuals = append(out.Residuals, RegistrationResidual{dx, dy, d, a.Holdout})
		if !a.Holdout {
			out.FitRMS += d * d
		}
	}
	out.FitRMS = math.Sqrt(out.FitRMS / n)
	return out, nil
}
