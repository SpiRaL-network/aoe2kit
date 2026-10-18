package preview

import (
	"fmt"
	"math"
)

// ResolveRotationModel never infers a model from file version or value range.
func ResolveRotationModel(name string) (string, error) {
	if name == "" {
		name = "stored-index"
	}
	switch name {
	case "stored-index", "radians", "radians-reversed":
		return name, nil
	default:
		return "", fmt.Errorf("unknown rotation model %q; use stored-index, radians or radians-reversed", name)
	}
}

// RotationIndex is a preview hypothesis, not a conversion of the scenario.
// Candidate radian models use raw Rotation, ignoring the cached Variant index.
func RotationIndex(o Object, model string, angles int) (int, error) {
	model, err := ResolveRotationModel(model)
	if err != nil {
		return 0, err
	}
	v := o.Rotation
	if model == "stored-index" && o.Variant != 0 {
		v = float64(o.Variant)
	}
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0, fmt.Errorf("rotation must be finite")
	}
	if model == "stored-index" {
		v = math.Round(v)
		limit := float64(int(^uint(0) >> 1))
		if v >= limit || v < -limit {
			return 0, fmt.Errorf("rotation index exceeds integer range")
		}
		return int(v), nil
	}
	if angles < 1 {
		angles = 1
	}
	turn := math.Mod(v, 2*math.Pi) / (2 * math.Pi)
	if model == "radians-reversed" {
		turn = -turn
	}
	i := int(math.Round(turn*float64(angles))) % angles
	if i < 0 {
		i += angles
	}
	return i, nil
}
