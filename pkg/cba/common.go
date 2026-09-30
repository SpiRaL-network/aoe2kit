package cba

import (
	"fmt"

	"aoe2kit/pkg/replay"
)

func playerLabel(playerID int) string {
	if playerID == 0 {
		return "Gaia"
	}
	return fmt.Sprintf("P%d", playerID)
}

func CivRazeArchetype(civID int) string {
	name := replay.CivDisplayName(civID)
	switch name {
	case "Mayans":
		return "weak_razer_set_help_expected"
	case "Magyars", "Ethiopians":
		return "raze_first_expected"
	case "Chinese", "Goths":
		return "defer_expected"
	default:
		return "unclassified"
	}
}
