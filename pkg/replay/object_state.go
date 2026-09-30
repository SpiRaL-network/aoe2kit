package replay

import (
	"fmt"
	"math"
)

type ObjectStateOptions struct {
	Limit          int
	ReferencedOnly bool
	Class          string
}

type ObjectStateReport struct {
	Path         string                `json:"path,omitempty"`
	Method       string                `json:"method"`
	Verification string                `json:"verification"`
	Summary      ObjectStateSummary    `json:"summary"`
	Players      []ObjectPlayerSummary `json:"players,omitempty"`
	Shown        int                   `json:"shown"`
	Objects      []ObjectStateCard     `json:"objects,omitempty"`
	Warnings     []string              `json:"warnings,omitempty"`
}

type ObjectStateSummary struct {
	Objects                int `json:"objects"`
	Referenced             int `json:"referenced"`
	Buildings              int `json:"buildings"`
	Gates                  int `json:"gates"`
	Damaged                int `json:"damaged"`
	UnderAttack            int `json:"under_attack"`
	ActionPrefixDecoded    int `json:"action_prefix_decoded"`
	CombatTailDecoded      int `json:"combat_tail_decoded"`
	BuildingPrefixDecoded  int `json:"building_prefix_decoded"`
	BuildingTailDecoded    int `json:"building_tail_decoded"`
	BuildingTailPlausible  int `json:"building_tail_plausible"`
	ProductionQueueHeaders int `json:"production_queue_headers"`
	GatherPoints           int `json:"gather_points"`
	GateLockedNonZero      int `json:"gate_locked_nonzero"`
	V68ActionBlockObjects  int `json:"v68_action_block_objects"`
	V68TrailerObjects      int `json:"v68_trailer_objects"`
	DecodedBodyPrefixBytes int `json:"decoded_body_prefix_bytes"`
	OpaqueBodyBytes        int `json:"opaque_body_bytes"`
}

type ObjectStateCard struct {
	ObjectID       int                  `json:"object_id"`
	OwnerID        int                  `json:"owner_id"`
	OwnerLabel     string               `json:"owner_label,omitempty"`
	UnitID         int                  `json:"unit_id"`
	UnitName       string               `json:"unit_name,omitempty"`
	Class          string               `json:"class"`
	RecordType     int                  `json:"record_type"`
	RecordTypeName string               `json:"record_type_name"`
	HitPoints      float64              `json:"hitpoints,omitempty"`
	ObjectState    int                  `json:"object_state,omitempty"`
	X              float64              `json:"x"`
	Y              float64              `json:"y"`
	CurrentDamage  int                  `json:"current_damage,omitempty"`
	UnderAttack    bool                 `json:"under_attack,omitempty"`
	CommandRefs    int                  `json:"command_refs,omitempty"`
	TargetRefs     int                  `json:"target_refs,omitempty"`
	SelectedRefs   int                  `json:"selected_refs,omitempty"`
	Action         *ObjectActionState   `json:"action,omitempty"`
	Combat         *ObjectCombatState   `json:"combat,omitempty"`
	Building       *ObjectBuildingState `json:"building,omitempty"`
	Coverage       ObjectStateCoverage  `json:"coverage"`
	Confidence     string               `json:"confidence"`
	Notes          []string             `json:"notes,omitempty"`
}

type ObjectActionState struct {
	Decoded           bool    `json:"decoded"`
	Waiting           int     `json:"waiting,omitempty"`
	CommandFlag       int     `json:"command_flag,omitempty"`
	SelectedGroupInfo int     `json:"selected_group_info,omitempty"`
	ActionType        int     `json:"action_type,omitempty"`
	FormationID       int     `json:"formation_id,omitempty"`
	FormationRow      int     `json:"formation_row,omitempty"`
	FormationCol      int     `json:"formation_col,omitempty"`
	AttackTimer       float64 `json:"attack_timer,omitempty"`
	CaptureFlag       int     `json:"capture_flag,omitempty"`
	AttackCount       int     `json:"attack_count,omitempty"`
	V68ActionBlocks   int     `json:"v68_action_blocks,omitempty"`
}

type ObjectCombatState struct {
	Decoded                  bool    `json:"decoded"`
	HasAI                    int     `json:"has_ai,omitempty"`
	HasDEPosition            bool    `json:"has_de_position,omitempty"`
	TownBellFlag             int     `json:"town_bell_flag,omitempty"`
	TownBellTargetID         int     `json:"town_bell_target_id,omitempty"`
	TownBellTargetX          float64 `json:"town_bell_target_x,omitempty"`
	TownBellTargetY          float64 `json:"town_bell_target_y,omitempty"`
	TownBellAction           int     `json:"town_bell_action,omitempty"`
	BerserkerTimer           float64 `json:"berserker_timer,omitempty"`
	NumBuilders              int     `json:"num_builders,omitempty"`
	NumHealers               int     `json:"num_healers,omitempty"`
	PrimaryObjectID          int     `json:"primary_object_id,omitempty"`
	SecondaryObjectID        int     `json:"secondary_object_id,omitempty"`
	V68TrailerKind           int     `json:"v68_trailer_kind,omitempty"`
	V68TrailerCounter        int     `json:"v68_trailer_counter,omitempty"`
	PositionOrderTailDecoded bool    `json:"position_order_tail_decoded,omitempty"`
}

type ObjectBuildingState struct {
	Decoded                 bool    `json:"decoded"`
	Built                   int     `json:"built,omitempty"`
	BuildPoints             float64 `json:"build_points,omitempty"`
	UniqueBuildID           int     `json:"unique_build_id,omitempty"`
	Culture                 int     `json:"culture,omitempty"`
	Burning                 int     `json:"burning,omitempty"`
	LastBurnTime            int     `json:"last_burn_time,omitempty"`
	LastGarrisonTime        int     `json:"last_garrison_time,omitempty"`
	RelicCount              int     `json:"relic_count,omitempty"`
	SpecificRelicCount      int     `json:"specific_relic_count,omitempty"`
	GatherPointExists       int     `json:"gather_point_exists,omitempty"`
	GatherPointX            float64 `json:"gather_point_x,omitempty"`
	GatherPointY            float64 `json:"gather_point_y,omitempty"`
	GatherPointObjectID     int     `json:"gather_point_object_id,omitempty"`
	GatherPointUnitTypeID   int     `json:"gather_point_unit_type_id,omitempty"`
	DesolidFlag             int     `json:"desolid_flag,omitempty"`
	PendingOrder            int     `json:"pending_order,omitempty"`
	LinkedOwner             int     `json:"linked_owner,omitempty"`
	LinkedChildren          []int   `json:"linked_children,omitempty"`
	CapturedUnitCount       int     `json:"captured_unit_count,omitempty"`
	ProductionQueueCapacity int     `json:"production_queue_capacity,omitempty"`
	EndpointX               float64 `json:"endpoint_x,omitempty"`
	EndpointY               float64 `json:"endpoint_y,omitempty"`
	Endpoint2X              float64 `json:"endpoint_2_x,omitempty"`
	Endpoint2Y              float64 `json:"endpoint_2_y,omitempty"`
	GateLocked              int     `json:"gate_locked,omitempty"`
	FirstUpdateRaw          int     `json:"first_update_raw,omitempty"`
	CloseTimerRaw           int     `json:"close_timer_raw,omitempty"`
	TerrainType             int     `json:"terrain_type,omitempty"`
	SemiAsleep              int     `json:"semi_asleep,omitempty"`
	SnowFlag                int     `json:"snow_flag,omitempty"`
	BuildingTailPlausible   bool    `json:"building_tail_plausible,omitempty"`
	V68TrailerKind          int     `json:"v68_trailer_kind,omitempty"`
	V68TrailerValue         float64 `json:"v68_trailer_value,omitempty"`
	LinkedObjectIDs         []int   `json:"linked_object_ids,omitempty"`
}

type ObjectStateCoverage struct {
	RecordBytes            int    `json:"record_bytes"`
	PrefixBytes            int    `json:"prefix_bytes"`
	BodyBytes              int    `json:"body_bytes"`
	DecodedBodyPrefixBytes int    `json:"decoded_body_prefix_bytes"`
	OpaqueBodyBytes        int    `json:"opaque_body_bytes"`
	EndConfidence          string `json:"end_confidence,omitempty"`
}

func BuildObjectState(path string, opts ObjectStateOptions) (*ObjectStateReport, error) {
	index, err := BuildObjectIndex(path, ObjectIndexOptions{ReferencedOnly: opts.ReferencedOnly})
	if err != nil {
		return nil, err
	}
	cards := make([]ObjectStateCard, 0, len(index.Objects))
	summary := ObjectStateSummary{
		DecodedBodyPrefixBytes: index.Summary.DecodedBodyPrefixBytes,
		OpaqueBodyBytes:        index.Summary.OpaqueBodyBytes,
	}
	for _, object := range index.Objects {
		if opts.Class != "" && object.Class != opts.Class {
			continue
		}
		card := objectStateCard(object)
		cards = append(cards, card)
		summarizeObjectStateCard(&summary, card)
	}
	shown := cards
	if opts.Limit > 0 && len(shown) > opts.Limit {
		shown = shown[:opts.Limit]
	}
	return &ObjectStateReport{
		Path:         path,
		Method:       "initial_object_state_cards_from_v68_candidate_prefix_and_decoded_tail_islands",
		Verification: "structure_verified_initial_object_state_not_runtime_delta_stream",
		Summary:      summary,
		Players:      index.Players,
		Shown:        len(shown),
		Objects:      shown,
		Warnings:     index.Warnings,
	}, nil
}

func objectStateCard(object ObjectCandidate) ObjectStateCard {
	card := ObjectStateCard{
		ObjectID:       object.ObjectID,
		OwnerID:        object.OwnerID,
		OwnerLabel:     object.OwnerLabel,
		UnitID:         object.UnitID,
		UnitName:       UnitDisplayName(object.UnitID),
		Class:          object.Class,
		RecordType:     object.RecordType,
		RecordTypeName: object.RecordTypeName,
		HitPoints:      finiteReplayFloat(object.HitPoints),
		ObjectState:    object.ObjectState,
		X:              finiteReplayFloat(object.X),
		Y:              finiteReplayFloat(object.Y),
		CurrentDamage:  object.CurrentDamage,
		UnderAttack:    object.UnderAttack,
		CommandRefs:    object.CommandRefs,
		TargetRefs:     object.TargetRefs,
		SelectedRefs:   object.SelectedRefs,
		Coverage: ObjectStateCoverage{
			RecordBytes:            object.RecordBytes,
			PrefixBytes:            object.PrefixBytes,
			BodyBytes:              object.BodyBytes,
			DecodedBodyPrefixBytes: object.DecodedBodyPrefixBytes,
			OpaqueBodyBytes:        object.BodyBytes - object.DecodedBodyPrefixBytes,
			EndConfidence:          object.EndConfidence,
		},
		Confidence: object.Confidence,
	}
	if object.ActionCombatPrefixBytes > 0 || object.V68ActionBlockCount > 0 {
		card.Action = &ObjectActionState{
			Decoded:           true,
			Waiting:           object.ActionWaiting,
			CommandFlag:       object.ActionCommandFlag,
			SelectedGroupInfo: object.ActionSelectedGroupInfo,
			ActionType:        object.ActionType,
			FormationID:       object.FormationID,
			FormationRow:      object.FormationRow,
			FormationCol:      object.FormationCol,
			AttackTimer:       finiteReplayFloat(object.AttackTimer),
			CaptureFlag:       object.CaptureFlag,
			AttackCount:       object.AttackCount,
			V68ActionBlocks:   object.V68ActionBlockCount,
		}
	}
	if object.CombatTailPrefixBytes > 0 || object.CombatV68TrailerBytes > 0 || object.V68PositionOrderTailBytes > 0 {
		card.Combat = &ObjectCombatState{
			Decoded:                  true,
			HasAI:                    boundedNonNegative(object.HasAI, 1000),
			HasDEPosition:            object.HasDEPosition,
			TownBellFlag:             replayFlag(object.TownBellFlag),
			TownBellTargetID:         boundedReplayID(object.TownBellTargetID),
			TownBellTargetX:          finiteReplayFloat(object.TownBellTargetX),
			TownBellTargetY:          finiteReplayFloat(object.TownBellTargetY),
			TownBellAction:           boundedNonNegative(object.TownBellAction, 200),
			BerserkerTimer:           finiteReplayFloat(object.BerserkerTimer),
			NumBuilders:              boundedNonNegative(object.NumBuilders, 200),
			NumHealers:               boundedNonNegative(object.NumHealers, 200),
			PrimaryObjectID:          boundedReplayID(object.CombatV68PrimaryObjectID),
			SecondaryObjectID:        boundedReplayID(object.CombatV68SecondaryObjectID),
			V68TrailerKind:           object.CombatV68TrailerKind,
			V68TrailerCounter:        object.CombatV68TrailerCounter,
			PositionOrderTailDecoded: object.V68PositionOrderTailBytes > 0,
		}
	}
	if object.BuildingPrefixBytes > 0 {
		card.Building = &ObjectBuildingState{
			Decoded:                 true,
			Built:                   replayFlag(object.Built),
			BuildPoints:             finiteReplayFloat(object.BuildPoints),
			UniqueBuildID:           boundedReplayID(object.UniqueBuildID),
			Culture:                 boundedPositive(object.Culture, 64),
			Burning:                 replayFlag(object.Burning),
			LastBurnTime:            boundedNonNegative(object.LastBurnTime, 1000000),
			LastGarrisonTime:        boundedNonNegative(object.LastGarrisonTime, 1000000),
			RelicCount:              boundedNonNegative(object.RelicCount, 1000),
			SpecificRelicCount:      boundedNonNegative(object.SpecificRelicCount, 1000),
			GatherPointExists:       replayFlag(object.GatherPointExists),
			GatherPointX:            finiteReplayFloat(object.GatherPointX),
			GatherPointY:            finiteReplayFloat(object.GatherPointY),
			GatherPointObjectID:     boundedReplayID(object.GatherPointObjectID),
			GatherPointUnitTypeID:   boundedPositive(object.GatherPointUnitTypeID, 20000),
			DesolidFlag:             replayFlag(object.DesolidFlag),
			PendingOrder:            boundedNonNegative(object.PendingOrder, 200),
			LinkedOwner:             boundedPositive(object.LinkedOwner, 16),
			LinkedChildren:          cleanedObjectIDs(object.LinkedChildren),
			CapturedUnitCount:       plausibleLinkedCount(object.LinkedOwner, object.CapturedUnitCount),
			ProductionQueueCapacity: object.ProductionQueueCapacity,
			EndpointX:               replayCoordinate(object.EndpointX),
			EndpointY:               replayCoordinate(object.EndpointY),
			Endpoint2X:              replayCoordinate(object.Endpoint2X),
			Endpoint2Y:              replayCoordinate(object.Endpoint2Y),
			GateLocked:              replayFlag(object.GateLocked),
			FirstUpdateRaw:          boundedNonNegative(object.FirstUpdateRaw, 1000000),
			CloseTimerRaw:           boundedNonNegative(object.CloseTimerRaw, 1000000),
			TerrainType:             boundedNonNegative(object.TerrainType, 250),
			SemiAsleep:              replayFlag(object.SemiAsleep),
			SnowFlag:                replayFlag(object.SnowFlag),
			V68TrailerKind:          object.BuildingV68TrailerKind,
			BuildingTailPlausible:   plausibleBuildingTail(object),
			V68TrailerValue:         finiteReplayFloat(object.BuildingV68TrailerValue),
			LinkedObjectIDs:         object.BuildingV68LinkedObjectIDs,
		}
	}
	if card.Coverage.OpaqueBodyBytes > 0 {
		card.Notes = append(card.Notes, fmt.Sprintf("%d object body bytes remain opaque", card.Coverage.OpaqueBodyBytes))
	}
	return card
}

func summarizeObjectStateCard(summary *ObjectStateSummary, card ObjectStateCard) {
	summary.Objects++
	if card.CommandRefs > 0 {
		summary.Referenced++
	}
	if card.Class == "building" {
		summary.Buildings++
	}
	if card.Class == "gate" {
		summary.Gates++
	}
	if card.CurrentDamage > 0 || card.HitPoints <= 0 {
		summary.Damaged++
	}
	if card.UnderAttack {
		summary.UnderAttack++
	}
	if card.Action != nil {
		summary.ActionPrefixDecoded++
		if card.Action.V68ActionBlocks > 0 {
			summary.V68ActionBlockObjects++
		}
	}
	if card.Combat != nil {
		summary.CombatTailDecoded++
		if card.Combat.PrimaryObjectID > 0 || card.Combat.SecondaryObjectID > 0 || card.Combat.V68TrailerKind > 0 {
			summary.V68TrailerObjects++
		}
	}
	if card.Building != nil {
		summary.BuildingPrefixDecoded++
		if card.Building.ProductionQueueCapacity > 0 {
			summary.ProductionQueueHeaders++
		}
		if card.Building.GatherPointExists != 0 {
			summary.GatherPoints++
		}
		if card.Building.EndpointX != 0 || card.Building.EndpointY != 0 || card.Building.Endpoint2X != 0 || card.Building.Endpoint2Y != 0 || card.Building.GateLocked != 0 || card.Building.FirstUpdateRaw != 0 || card.Building.CloseTimerRaw != 0 {
			summary.BuildingTailDecoded++
		}
		if card.Building.BuildingTailPlausible {
			summary.BuildingTailPlausible++
		}
		if card.Building.BuildingTailPlausible && card.Building.GateLocked != 0 {
			summary.GateLockedNonZero++
		}
	}
}

func finiteReplayFloat(v float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0
	}
	return v
}

func replayCoordinate(v float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) || v < 0 || v > 512 {
		return 0
	}
	return v
}

func plausibleBuildingTail(object ObjectCandidate) bool {
	if object.BuildingTailPrefixBytes == 0 {
		return false
	}
	if object.GateLocked != 0 && object.GateLocked != 1 {
		return false
	}
	return plausibleStateCoordinate(object.EndpointX) &&
		plausibleStateCoordinate(object.EndpointY) &&
		plausibleStateCoordinate(object.Endpoint2X) &&
		plausibleStateCoordinate(object.Endpoint2Y)
}

func plausibleStateCoordinate(v float64) bool {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return false
	}
	return v >= 0 && v <= 512
}

func replayFlag(v int) int {
	if v == 1 {
		return 1
	}
	return 0
}

func boundedNonNegative(v int, max int) int {
	if v < 0 || v > max {
		return 0
	}
	return v
}

func boundedPositive(v int, max int) int {
	if v <= 0 || v > max {
		return 0
	}
	return v
}

func boundedReplayID(v int) int {
	if v <= 0 || v > 1000000 {
		return 0
	}
	return v
}

func plausibleLinkedCount(owner int, count int) int {
	if owner <= 0 || owner > 16 {
		return 0
	}
	return boundedNonNegative(count, 1000)
}

func cleanedObjectIDs(ids []int) []int {
	var out []int
	for _, id := range ids {
		if id > 0 && id <= 1000000 {
			out = append(out, id)
		}
	}
	return out
}
