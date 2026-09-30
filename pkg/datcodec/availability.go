package datcodec

import (
	"encoding/json"
	"fmt"

	"aoe2kit/pkg/aoe2"
	"aoe2kit/pkg/datfile"
)

type UnitAvailabilityRequest struct {
	UnitID  int   `json:"unit_id"`
	CivID   *int  `json:"civ_id,omitempty"`
	CivIDs  []int `json:"civ_ids,omitempty"`
	AllCivs bool  `json:"all_civs,omitempty"`
}

type UnitAvailabilityCardsReport struct {
	Version      string                  `json:"version"`
	Request      UnitAvailabilityRequest `json:"request"`
	Summary      UnitAvailabilitySummary `json:"summary"`
	Cards        []UnitAvailabilityCard  `json:"cards"`
	Verification aoe2.VerificationClaim  `json:"verification"`
}

type UnitAvailabilitySummary struct {
	CivCount            int `json:"civ_count"`
	Present             int `json:"present"`
	Enabled             int `json:"enabled"`
	Creatable           int `json:"creatable"`
	WithTrainLocations  int `json:"with_train_locations"`
	WithTechTreeRefs    int `json:"with_tech_tree_refs"`
	PlausiblyTrainable  int `json:"plausibly_trainable"`
	Absent              int `json:"absent"`
	Disabled            int `json:"disabled"`
	NeedsMoreInspection int `json:"needs_more_inspection"`
}

type UnitAvailabilityCard struct {
	CivID              int                       `json:"civ_id"`
	CivName            string                    `json:"civ_name,omitempty"`
	UnitID             int                       `json:"unit_id"`
	UnitName           string                    `json:"unit_name,omitempty"`
	Present            bool                      `json:"present"`
	Enabled            bool                      `json:"enabled"`
	RawEnabled         uint8                     `json:"raw_enabled,omitempty"`
	Creatable          bool                      `json:"creatable"`
	TrainLocationCount int                       `json:"train_location_count"`
	TrainLocations     []TrainLocationCard       `json:"train_locations,omitempty"`
	TechTreeRefs       []TechTreeAvailabilityRef `json:"tech_tree_refs,omitempty"`
	Status             string                    `json:"status"`
	RecipeHint         json.RawMessage           `json:"recipe_hint,omitempty"`
	Caveats            []string                  `json:"caveats,omitempty"`
}

type TrainLocationCard struct {
	Index       int   `json:"index"`
	TrainTime   int16 `json:"train_time"`
	TrainUnitID int16 `json:"train_unit_id"`
	TrainButton uint8 `json:"train_button"`
	TrainHotkey int32 `json:"train_hotkey"`
}

type TechTreeAvailabilityRef struct {
	Section          string `json:"section"`
	Index            int    `json:"index"`
	Field            string `json:"field"`
	Status           uint8  `json:"status,omitempty"`
	RequiredResearch int32  `json:"required_research,omitempty"`
	EnablingResearch int32  `json:"enabling_research,omitempty"`
	UpperBuilding    int32  `json:"upper_building,omitempty"`
	LocationInAge    int32  `json:"location_in_age,omitempty"`
}

func UnitAvailabilityFile(path string, request UnitAvailabilityRequest) (UnitAvailabilityCardsReport, error) {
	idx, err := datfile.Open(path)
	if err != nil {
		return UnitAvailabilityCardsReport{}, err
	}
	return UnitAvailability(idx, request)
}

func UnitAvailability(idx *datfile.Index, request UnitAvailabilityRequest) (UnitAvailabilityCardsReport, error) {
	if request.UnitID < 0 {
		return UnitAvailabilityCardsReport{}, fmt.Errorf("negative unit_id %d", request.UnitID)
	}
	civIDs, err := availabilityCivIDs(idx, request)
	if err != nil {
		return UnitAvailabilityCardsReport{}, err
	}
	report := UnitAvailabilityCardsReport{
		Version:      Version,
		Request:      request,
		Cards:        make([]UnitAvailabilityCard, 0, len(civIDs)),
		Verification: aoe2.StructureVerification(true),
	}
	report.Verification.Note = "unit availability is a structural DAT inspection over unit records, train-location rows, and decoded tech-tree references; in-engine trainability remains a separate oracle."
	techRefs := unitAvailabilityTechTreeRefs(idx, request.UnitID)
	for _, civID := range civIDs {
		card := unitAvailabilityCard(idx, civID, request.UnitID, techRefs)
		report.Cards = append(report.Cards, card)
		report.Summary.CivCount++
		if card.Present {
			report.Summary.Present++
		} else {
			report.Summary.Absent++
		}
		if card.Enabled {
			report.Summary.Enabled++
		}
		if card.Creatable {
			report.Summary.Creatable++
		}
		if card.TrainLocationCount > 0 {
			report.Summary.WithTrainLocations++
		}
		if len(card.TechTreeRefs) > 0 {
			report.Summary.WithTechTreeRefs++
		}
		if card.Status == "plausibly_trainable_structurally" {
			report.Summary.PlausiblyTrainable++
		}
		if card.Present && !card.Enabled {
			report.Summary.Disabled++
		}
		if len(card.Caveats) > 0 {
			report.Summary.NeedsMoreInspection++
		}
	}
	return report, nil
}

func availabilityCivIDs(idx *datfile.Index, request UnitAvailabilityRequest) ([]int, error) {
	selectorCount := 0
	if request.AllCivs {
		selectorCount++
	}
	if request.CivID != nil {
		selectorCount++
	}
	if len(request.CivIDs) > 0 {
		selectorCount++
	}
	if selectorCount == 0 {
		request.AllCivs = true
		selectorCount = 1
	}
	if selectorCount != 1 {
		return nil, fmt.Errorf("availability needs at most one selector: civ_id, civ_ids, or all_civs")
	}
	if request.AllCivs {
		civIDs := make([]int, 0, len(idx.Civs))
		for _, civ := range idx.Civs {
			civIDs = append(civIDs, civ.Index)
		}
		return civIDs, nil
	}
	if request.CivID != nil {
		if *request.CivID < 0 || *request.CivID >= len(idx.Civs) {
			return nil, fmt.Errorf("civ_id=%d outside civ table", *request.CivID)
		}
		return []int{*request.CivID}, nil
	}
	civIDs := append([]int(nil), request.CivIDs...)
	seen := make(map[int]bool, len(civIDs))
	for _, civID := range civIDs {
		if seen[civID] {
			return nil, fmt.Errorf("duplicate civ_id=%d", civID)
		}
		seen[civID] = true
		if civID < 0 || civID >= len(idx.Civs) {
			return nil, fmt.Errorf("civ_id=%d outside civ table", civID)
		}
	}
	return civIDs, nil
}

func unitAvailabilityCard(idx *datfile.Index, civID, unitID int, techRefs []TechTreeAvailabilityRef) UnitAvailabilityCard {
	civ := idx.Civs[civID]
	card := UnitAvailabilityCard{
		CivID:   civID,
		CivName: civ.Name,
		UnitID:  unitID,
	}
	if unitID >= len(civ.Units) {
		card.Status = "absent_from_civ_unit_table"
		card.Caveats = append(card.Caveats, "The civ unit table does not reach this unit ID; use create-unit/cloning, not an enabled-flag toggle.")
		return card
	}
	unit := civ.Units[unitID]
	if !unit.Present {
		card.Status = "absent_unit_record"
		card.Caveats = append(card.Caveats, "The unit slot has no present record for this civ; use create-unit/cloning, not an enabled-flag toggle.")
		return card
	}
	card.Present = true
	card.UnitName = unit.Name
	card.RawEnabled = unit.Enabled
	card.Enabled = unit.Enabled != 0
	card.TechTreeRefs = append([]TechTreeAvailabilityRef(nil), techRefs...)
	if unit.Creatable != nil {
		card.Creatable = true
		card.TrainLocationCount = len(unit.Creatable.TrainLocations)
		card.TrainLocations = make([]TrainLocationCard, 0, len(unit.Creatable.TrainLocations))
		for _, location := range unit.Creatable.TrainLocations {
			card.TrainLocations = append(card.TrainLocations, TrainLocationCard{
				Index:       location.Index,
				TrainTime:   location.TrainTime,
				TrainUnitID: location.TrainUnitID,
				TrainButton: location.TrainButton,
				TrainHotkey: location.TrainHotkey,
			})
		}
	}
	switch {
	case !card.Enabled:
		card.Status = "disabled_unit_record"
		card.RecipeHint = mustRecipeHint(Recipe{UnitAvailability: []UnitAvailabilityRecipe{{CivID: &civID, UnitID: unitID, Enabled: true}}})
	case !card.Creatable:
		card.Status = "enabled_but_not_creatable_record"
		card.Caveats = append(card.Caveats, "The unit record is enabled but lacks a parsed Creatable block, so ordinary building training is unlikely from this record alone.")
	case card.TrainLocationCount == 0:
		card.Status = "enabled_creatable_without_train_locations"
		card.Caveats = append(card.Caveats, "The unit has a Creatable block but no train-location rows.")
	case len(card.TechTreeRefs) == 0:
		card.Status = "enabled_train_locations_no_tech_tree_refs"
		card.Caveats = append(card.Caveats, "The unit has train-location rows but no decoded tech-tree references; it may still be created by triggers or special rules.")
	default:
		card.Status = "plausibly_trainable_structurally"
		card.Caveats = append(card.Caveats, "This is structural evidence only; final availability requires editor/game verification.")
	}
	return card
}

func unitAvailabilityTechTreeRefs(idx *datfile.Index, unitID int) []TechTreeAvailabilityRef {
	var refs []TechTreeAvailabilityRef
	target := int32(unitID)
	for _, connection := range idx.TechTree.UnitConnections {
		if connection.ID == target {
			refs = append(refs, TechTreeAvailabilityRef{
				Section:          "tech_tree_unit_connection",
				Index:            connection.Index,
				Field:            "id",
				Status:           connection.Status,
				RequiredResearch: connection.RequiredResearch,
				EnablingResearch: connection.EnablingResearch,
				UpperBuilding:    connection.UpperBuilding,
				LocationInAge:    connection.LocationInAge,
			})
		}
		if int32ListContains(connection.Units, target) {
			refs = append(refs, TechTreeAvailabilityRef{
				Section:          "tech_tree_unit_connection",
				Index:            connection.Index,
				Field:            "units",
				Status:           connection.Status,
				RequiredResearch: connection.RequiredResearch,
				EnablingResearch: connection.EnablingResearch,
				UpperBuilding:    connection.UpperBuilding,
				LocationInAge:    connection.LocationInAge,
			})
		}
	}
	for _, connection := range idx.TechTree.ResearchConnections {
		if int32ListContains(connection.Units, target) {
			refs = append(refs, TechTreeAvailabilityRef{
				Section:       "tech_tree_research_connection",
				Index:         connection.Index,
				Field:         "units",
				Status:        connection.Status,
				UpperBuilding: connection.UpperBuilding,
				LocationInAge: connection.LocationInAge,
			})
		}
	}
	return refs
}

func int32ListContains(values []int32, target int32) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
