package aoe2

type VerificationClaim struct {
	Label             string `json:"label"`
	StructureVerified bool   `json:"structure_verified"`
	EngineVerified    bool   `json:"engine_verified"`
	Note              string `json:"note"`
}

func StructureVerification(ok bool) VerificationClaim {
	label := "structure_failed"
	if ok {
		label = "structure_verified_not_engine_verified"
	}
	return VerificationClaim{
		Label:             label,
		StructureVerified: ok,
		EngineVerified:    false,
		Note:              "AoE2Kit verifies file structure, parser readback, packaging, and declared invariants only; in-engine load/render/play behavior requires a separate live game/editor oracle.",
	}
}

func StructureOKActivationFailed() VerificationClaim {
	return VerificationClaim{
		Label:             "structure_ok_activation_failed",
		StructureVerified: true,
		EngineVerified:    false,
		Note:              "AoE2Kit verified file structure, parser readback, packaging, and declared invariants, but the local activation gate failed; in-engine load/render/play behavior still requires a separate live game/editor oracle.",
	}
}
