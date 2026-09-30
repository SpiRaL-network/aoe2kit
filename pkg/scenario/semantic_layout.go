package scenario

import (
	"fmt"
)

// SemanticDocument is the editor-facing layer over the byte layout. It keeps
// P1..P8 labels and mirror policies out of byte-field names.
type SemanticDocument struct {
	Version   string
	Players   []SemanticPlayer
	Diplomacy SemanticDiplomacy
	Victory   map[string]int
	Units     []SemanticUnit
}

type SemanticPlayer struct {
	Slot          int
	Label         string
	Settings      PlayerSettings
	Diplomacy     []int
	AlliedVictory bool
}

type SemanticDiplomacy struct {
	Matrix        [][]int
	AlliedVictory []bool
}

type SemanticUnit struct {
	Player int
	Unit   UnitSummary
}

// SemanticDocument projects the current parser result into the Kit-owned
// semantic vocabulary. Until all sections have migrated, this is an adapter;
// its contract is independent of the legacy field representation.
func (f *File) SemanticDocument() SemanticDocument {
	settings := f.Settings()
	doc := SemanticDocument{
		Version: f.Version,
		Diplomacy: SemanticDiplomacy{
			Matrix:        cloneIntMatrix(settings.Diplomacy.Matrix),
			AlliedVictory: append([]bool(nil), settings.Diplomacy.AlliedVictory...),
		},
		Victory: cloneIntMap(settings.Victory),
	}
	for i, player := range settings.Players {
		slot := player.Player
		if slot == 0 && i > 0 {
			slot = i
		}
		doc.Players = append(doc.Players, SemanticPlayer{
			Slot:          slot,
			Label:         fmt.Sprintf("P%d", slot+1),
			Settings:      player,
			Diplomacy:     append([]int(nil), player.Diplomacy...),
			AlliedVictory: player.AlliedVictory,
		})
	}
	if f.Units != nil {
		for _, section := range f.Units.Sections {
			for _, unit := range section.Units {
				doc.Units = append(doc.Units, SemanticUnit{Player: section.Player, Unit: unit})
			}
		}
	}
	return doc
}

// ValidateSemanticMirrors enforces the first migrated mirror groups. The
// authoritative copy is the diplomacy matrix/allied-victory arrays; player
// projections and unit diplomacy mirrors are derived views.
func ValidateSemanticMirrors(doc SemanticDocument) error {
	for i, player := range doc.Players {
		if i >= len(doc.Diplomacy.Matrix) {
			continue
		}
		if !equalInts(player.Diplomacy, doc.Diplomacy.Matrix[i]) {
			return fmt.Errorf("player P%d diplomacy mirror differs from authoritative matrix", i+1)
		}
		if i < len(doc.Diplomacy.AlliedVictory) && player.AlliedVictory != doc.Diplomacy.AlliedVictory[i] {
			return fmt.Errorf("player P%d allied-victory mirror differs from authoritative array", i+1)
		}
	}
	return nil
}

func cloneIntMatrix(in [][]int) [][]int {
	out := make([][]int, len(in))
	for i := range in {
		out[i] = append([]int(nil), in[i]...)
	}
	return out
}

func cloneIntMap(in map[string]int) map[string]int {
	out := make(map[string]int, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func equalInts(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
