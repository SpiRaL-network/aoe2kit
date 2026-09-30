package replay

import (
	"encoding/binary"
	"fmt"
	"math"
	"sort"
)

type ObjectIndexOptions struct {
	Limit          int
	ReferencedOnly bool
}

type ObjectIndexReport struct {
	Path         string                `json:"path,omitempty"`
	Method       string                `json:"method"`
	Verification string                `json:"verification"`
	Summary      ObjectIndexSummary    `json:"summary"`
	Players      []ObjectPlayerSummary `json:"players,omitempty"`
	Shown        int                   `json:"shown"`
	Objects      []ObjectCandidate     `json:"objects,omitempty"`
	Warnings     []string              `json:"warnings,omitempty"`
}

type ObjectShapeReport struct {
	Path         string             `json:"path,omitempty"`
	Method       string             `json:"method"`
	Verification string             `json:"verification"`
	Summary      ObjectShapeSummary `json:"summary"`
	Shown        int                `json:"shown"`
	Shapes       []ObjectShape      `json:"shapes,omitempty"`
	Warnings     []string           `json:"warnings,omitempty"`
}

type ObjectShapeSummary struct {
	Candidates             int `json:"candidates"`
	ReferencedCandidates   int `json:"referenced_candidates"`
	ShapeCount             int `json:"shape_count"`
	DecodedPrefixBytes     int `json:"decoded_prefix_bytes"`
	DecodedBodyPrefixBytes int `json:"decoded_body_prefix_bytes,omitempty"`
	BoundedBodyBytes       int `json:"bounded_body_bytes"`
	OpaqueBodyBytes        int `json:"opaque_body_bytes,omitempty"`
	MinRecordBytes         int `json:"min_record_bytes,omitempty"`
	MaxRecordBytes         int `json:"max_record_bytes,omitempty"`
	FinalBodyShapeCount    int `json:"final_body_shape_count,omitempty"`
	ReferencedShapeCount   int `json:"referenced_shape_count,omitempty"`
	UnreferencedShapeCount int `json:"unreferenced_shape_count,omitempty"`
}

type ObjectShape struct {
	RecordType             int         `json:"record_type"`
	RecordTypeName         string      `json:"record_type_name"`
	UnitID                 int         `json:"unit_id"`
	UnitName               string      `json:"unit_name,omitempty"`
	Class                  string      `json:"class"`
	RecordBytes            int         `json:"record_bytes"`
	PrefixBytes            int         `json:"prefix_bytes"`
	BodyBytes              int         `json:"body_bytes"`
	DecodedBodyPrefixEach  int         `json:"decoded_body_prefix_each,omitempty"`
	DecodedBodyPrefixBytes int         `json:"decoded_body_prefix_bytes,omitempty"`
	OpaqueBodyEach         int         `json:"opaque_body_each,omitempty"`
	OpaqueBodyBytes        int         `json:"opaque_body_bytes,omitempty"`
	EndConfidence          string      `json:"end_confidence"`
	Count                  int         `json:"count"`
	Referenced             int         `json:"referenced"`
	TargetRefs             int         `json:"target_refs"`
	SelectedRefs           int         `json:"selected_refs"`
	Owners                 map[int]int `json:"owners,omitempty"`
	ExampleObjectIDs       []int       `json:"example_object_ids,omitempty"`
	ExampleOffsets         []int       `json:"example_offsets,omitempty"`
	BodySampleHex          string      `json:"body_sample_hex,omitempty"`
	OpaqueSampleHex        string      `json:"opaque_sample_hex,omitempty"`
}

type ObjectIndexSummary struct {
	Candidates              int `json:"candidates"`
	ReferencedCandidates    int `json:"referenced_candidates"`
	CommandReferencedIDs    int `json:"command_referenced_ids"`
	TargetReferencedIDs     int `json:"target_referenced_ids"`
	AmbiguousCandidateIDs   int `json:"ambiguous_candidate_ids"`
	UnresolvedReferencedIDs int `json:"unresolved_referenced_ids"`
	DecodedPrefixBytes      int `json:"decoded_prefix_bytes"`
	DecodedBodyPrefixBytes  int `json:"decoded_body_prefix_bytes,omitempty"`
	BoundedBodyBytes        int `json:"bounded_body_bytes"`
	OpaqueBodyBytes         int `json:"opaque_body_bytes,omitempty"`
}

type ObjectPlayerSummary struct {
	PlayerID     int            `json:"player_id"`
	Label        string         `json:"label"`
	Candidates   int            `json:"candidates"`
	Referenced   int            `json:"referenced"`
	ByClass      map[string]int `json:"by_class,omitempty"`
	ByUnitID     map[int]int    `json:"by_unit_id,omitempty"`
	ByRecordType map[int]int    `json:"by_record_type,omitempty"`
}

type ObjectCandidate struct {
	ObjectID                   int      `json:"object_id"`
	OwnerID                    int      `json:"owner_id"`
	OwnerLabel                 string   `json:"owner_label,omitempty"`
	RecordType                 int      `json:"record_type"`
	RecordTypeName             string   `json:"record_type_name"`
	UnitID                     int      `json:"unit_id"`
	Class                      string   `json:"class"`
	HitPoints                  float64  `json:"hitpoints,omitempty"`
	ObjectState                int      `json:"object_state,omitempty"`
	Facet                      int      `json:"facet,omitempty"`
	X                          float64  `json:"x"`
	Y                          float64  `json:"y"`
	Z                          float64  `json:"z,omitempty"`
	ResourceType               int      `json:"resource_type,omitempty"`
	Amount                     float64  `json:"amount,omitempty"`
	WorkerCount                int      `json:"worker_count,omitempty"`
	CurrentDamage              int      `json:"current_damage,omitempty"`
	UnderAttack                bool     `json:"under_attack,omitempty"`
	GroupID                    int      `json:"group_id,omitempty"`
	HasObjectProps             int      `json:"has_object_props,omitempty"`
	HasSpriteList              int      `json:"has_sprite_list,omitempty"`
	SpriteEntries              int      `json:"sprite_entries,omitempty"`
	SpriteListBytes            int      `json:"sprite_list_bytes,omitempty"`
	ParticleTypes              []int    `json:"particle_types,omitempty"`
	ParticleNames              []string `json:"particle_names,omitempty"`
	DEExtensionBytes           int      `json:"de_extension_bytes,omitempty"`
	DEExtensionStrings         []string `json:"de_extension_strings,omitempty"`
	StaticTailBytes            int      `json:"static_tail_bytes,omitempty"`
	TurnSpeed                  float64  `json:"turn_speed,omitempty"`
	MovingPrefixBytes          int      `json:"moving_prefix_bytes,omitempty"`
	Angle                      float64  `json:"angle,omitempty"`
	NumPathData                int      `json:"num_path_data,omitempty"`
	HasFuturePathData          int      `json:"has_future_path_data,omitempty"`
	HasMovementData            int      `json:"has_movement_data,omitempty"`
	NumUserWaypoints           int      `json:"num_user_waypoints,omitempty"`
	HasSubstitutePosition      int      `json:"has_substitute_position,omitempty"`
	ActionCombatPrefixBytes    int      `json:"action_combat_prefix_bytes,omitempty"`
	ActionWaiting              int      `json:"action_waiting,omitempty"`
	ActionCommandFlag          int      `json:"action_command_flag,omitempty"`
	ActionSelectedGroupInfo    int      `json:"action_selected_group_info,omitempty"`
	ActionType                 int      `json:"action_type,omitempty"`
	FormationID                int      `json:"formation_id,omitempty"`
	FormationRow               int      `json:"formation_row,omitempty"`
	FormationCol               int      `json:"formation_col,omitempty"`
	AttackTimer                float64  `json:"attack_timer,omitempty"`
	CaptureFlag                int      `json:"capture_flag,omitempty"`
	AttackCount                int      `json:"attack_count,omitempty"`
	CombatPrefixBytes          int      `json:"combat_prefix_bytes,omitempty"`
	HasAI                      int      `json:"has_ai,omitempty"`
	CombatTailPrefixBytes      int      `json:"combat_tail_prefix_bytes,omitempty"`
	HasDEPosition              bool     `json:"has_de_position,omitempty"`
	TownBellFlag               int      `json:"town_bell_flag,omitempty"`
	TownBellTargetID           int      `json:"town_bell_target_id,omitempty"`
	TownBellTargetX            float64  `json:"town_bell_target_x,omitempty"`
	TownBellTargetY            float64  `json:"town_bell_target_y,omitempty"`
	TownBellAction             int      `json:"town_bell_action,omitempty"`
	BerserkerTimer             float64  `json:"berserker_timer,omitempty"`
	NumBuilders                int      `json:"num_builders,omitempty"`
	NumHealers                 int      `json:"num_healers,omitempty"`
	CombatV68TrailerBytes      int      `json:"combat_v68_trailer_bytes,omitempty"`
	CombatV68PrimaryObjectID   int      `json:"combat_v68_primary_object_id,omitempty"`
	CombatV68SecondaryObjectID int      `json:"combat_v68_secondary_object_id,omitempty"`
	CombatV68TrailerKind       int      `json:"combat_v68_trailer_kind,omitempty"`
	CombatV68TrailerCounter    int      `json:"combat_v68_trailer_counter,omitempty"`
	V68ActionBlockBytes        int      `json:"v68_action_block_bytes,omitempty"`
	V68ActionBlockCount        int      `json:"v68_action_block_count,omitempty"`
	V68PostActionStateBytes    int      `json:"v68_post_action_state_bytes,omitempty"`
	V68PositionOrderTailBytes  int      `json:"v68_position_order_tail_bytes,omitempty"`
	FinalSpanTailMarkerBytes   int      `json:"final_span_tail_marker_bytes,omitempty"`
	BuildingPrefixBytes        int      `json:"building_prefix_bytes,omitempty"`
	Built                      int      `json:"built,omitempty"`
	BuildPoints                float64  `json:"build_points,omitempty"`
	UniqueBuildID              int      `json:"unique_build_id,omitempty"`
	Culture                    int      `json:"culture,omitempty"`
	Burning                    int      `json:"burning,omitempty"`
	LastBurnTime               int      `json:"last_burn_time,omitempty"`
	LastGarrisonTime           int      `json:"last_garrison_time,omitempty"`
	RelicCount                 int      `json:"relic_count,omitempty"`
	SpecificRelicCount         int      `json:"specific_relic_count,omitempty"`
	GatherPointExists          int      `json:"gather_point_exists,omitempty"`
	GatherPointX               float64  `json:"gather_point_x,omitempty"`
	GatherPointY               float64  `json:"gather_point_y,omitempty"`
	GatherPointObjectID        int      `json:"gather_point_object_id,omitempty"`
	GatherPointUnitTypeID      int      `json:"gather_point_unit_type_id,omitempty"`
	DesolidFlag                int      `json:"desolid_flag,omitempty"`
	PendingOrder               int      `json:"pending_order,omitempty"`
	LinkedOwner                int      `json:"linked_owner,omitempty"`
	LinkedChildren             []int    `json:"linked_children,omitempty"`
	CapturedUnitCount          int      `json:"captured_unit_count,omitempty"`
	BuildingExtraActionsBytes  int      `json:"building_extra_actions_bytes,omitempty"`
	BuildingQueueHeaderBytes   int      `json:"building_queue_header_bytes,omitempty"`
	ProductionQueueCapacity    int      `json:"production_queue_capacity,omitempty"`
	BuildingTailPrefixBytes    int      `json:"building_tail_prefix_bytes,omitempty"`
	EndpointX                  float64  `json:"endpoint_x,omitempty"`
	EndpointY                  float64  `json:"endpoint_y,omitempty"`
	Endpoint2X                 float64  `json:"endpoint_2_x,omitempty"`
	Endpoint2Y                 float64  `json:"endpoint_2_y,omitempty"`
	GateLocked                 int      `json:"gate_locked,omitempty"`
	FirstUpdateRaw             int      `json:"first_update_raw,omitempty"`
	CloseTimerRaw              int      `json:"close_timer_raw,omitempty"`
	TerrainType                int      `json:"terrain_type,omitempty"`
	SemiAsleep                 int      `json:"semi_asleep,omitempty"`
	SnowFlag                   int      `json:"snow_flag,omitempty"`
	BuildingV68TrailerBytes    int      `json:"building_v68_trailer_bytes,omitempty"`
	BuildingV68TrailerFlag     int      `json:"building_v68_trailer_flag,omitempty"`
	BuildingV68TrailerValue    float64  `json:"building_v68_trailer_value,omitempty"`
	BuildingV68TrailerKind     int      `json:"building_v68_trailer_kind,omitempty"`
	BuildingV68LinkedObjectIDs []int    `json:"building_v68_linked_object_ids,omitempty"`
	ZeroTailPaddingBytes       int      `json:"zero_tail_padding_bytes,omitempty"`
	Offset                     int      `json:"offset"`
	SpanStart                  int      `json:"span_start"`
	SpanEnd                    int      `json:"span_end"`
	PrefixBytes                int      `json:"prefix_bytes"`
	NextOffset                 int      `json:"next_offset,omitempty"`
	RecordBytes                int      `json:"record_bytes,omitempty"`
	BodyBytes                  int      `json:"body_bytes,omitempty"`
	BodySampleHex              string   `json:"body_sample_hex,omitempty"`
	DecodedBodyPrefixBytes     int      `json:"decoded_body_prefix_bytes,omitempty"`
	OpaqueSampleHex            string   `json:"opaque_sample_hex,omitempty"`
	EndConfidence              string   `json:"end_confidence,omitempty"`
	CommandRefs                int      `json:"command_refs,omitempty"`
	TargetRefs                 int      `json:"target_refs,omitempty"`
	SelectedRefs               int      `json:"selected_refs,omitempty"`
	Confidence                 string   `json:"confidence"`
	Score                      int      `json:"score"`
	Ambiguous                  bool     `json:"ambiguous,omitempty"`
}

type ObjectReference struct {
	ObjectID       int     `json:"object_id"`
	OwnerID        int     `json:"owner_id"`
	OwnerLabel     string  `json:"owner_label,omitempty"`
	RecordType     int     `json:"record_type"`
	RecordTypeName string  `json:"record_type_name"`
	UnitID         int     `json:"unit_id"`
	Class          string  `json:"class"`
	X              float64 `json:"x"`
	Y              float64 `json:"y"`
	Confidence     string  `json:"confidence"`
}

type commandObjectRefs struct {
	Any      map[int]int
	Target   map[int]int
	Selected map[int]int
}

func BuildObjectIndex(path string, opts ObjectIndexOptions) (*ObjectIndexReport, error) {
	eventReport, err := ExtractEvents(path, EventOptions{IncludeSystemEvents: true, IncludeUntypedAction: true})
	if err != nil {
		return nil, err
	}
	return BuildObjectIndexFromEvents(path, eventReport.Events, opts)
}

func BuildObjectIndexFromEvents(path string, events []ReplayEvent, opts ObjectIndexOptions) (*ObjectIndexReport, error) {
	data, err := ReadRecordBytes(path)
	if err != nil {
		return nil, err
	}
	rec, err := Parse(data)
	if err != nil {
		return nil, err
	}
	coverage, err := BuildCoverage(path)
	if err != nil {
		return nil, err
	}
	if coverage.HeaderSpine == nil {
		return nil, fmt.Errorf("object index requires header spine coverage")
	}
	refs := collectRefsFromEvents(events)
	objects, ambiguous := scanObjectCandidates(rec.HeaderBytes(), coverage.HeaderSpine, coverage, refs, opts)
	sanitized := sanitizeObjectCandidates(objects)
	players := summarizeObjectPlayers(objects)
	referenced := 0
	decodedPrefixBytes := 0
	boundedBodyBytes := 0
	decodedBodyPrefixBytes := 0
	for _, obj := range objects {
		if obj.CommandRefs > 0 {
			referenced++
		}
		decodedPrefixBytes += obj.PrefixBytes
		boundedBodyBytes += obj.BodyBytes
		decodedBodyPrefixBytes += obj.DecodedBodyPrefixBytes
	}
	shownObjects := objects
	if opts.Limit > 0 && len(shownObjects) > opts.Limit {
		shownObjects = shownObjects[:opts.Limit]
	}
	report := &ObjectIndexReport{
		Path:         path,
		Method:       "save68_initial_payload_prefix_scan",
		Verification: "structure_verified_candidate_index_not_full_object_parse",
		Summary: ObjectIndexSummary{
			Candidates:              len(objects),
			ReferencedCandidates:    referenced,
			CommandReferencedIDs:    len(refs.Any),
			TargetReferencedIDs:     len(refs.Target),
			AmbiguousCandidateIDs:   ambiguous,
			UnresolvedReferencedIDs: maxReplayInt(0, len(refs.Any)-referenced),
			DecodedPrefixBytes:      decodedPrefixBytes,
			DecodedBodyPrefixBytes:  decodedBodyPrefixBytes,
			BoundedBodyBytes:        boundedBodyBytes,
			OpaqueBodyBytes:         boundedBodyBytes - decodedBodyPrefixBytes,
		},
		Players: players,
		Shown:   len(shownObjects),
		Objects: shownObjects,
	}
	if sanitized > 0 {
		report.Warnings = append(report.Warnings, fmt.Sprintf("sanitized %d non-finite float fields from raw object candidates before JSON output", sanitized))
	}
	return report, nil
}

func sanitizeObjectCandidates(objects []ObjectCandidate) int {
	count := 0
	for i := range objects {
		count += sanitizeFloat64(&objects[i].HitPoints)
		count += sanitizeFloat64(&objects[i].X)
		count += sanitizeFloat64(&objects[i].Y)
		count += sanitizeFloat64(&objects[i].Z)
		count += sanitizeFloat64(&objects[i].Amount)
		count += sanitizeFloat64(&objects[i].TurnSpeed)
		count += sanitizeFloat64(&objects[i].Angle)
		count += sanitizeFloat64(&objects[i].AttackTimer)
		count += sanitizeFloat64(&objects[i].TownBellTargetX)
		count += sanitizeFloat64(&objects[i].TownBellTargetY)
		count += sanitizeFloat64(&objects[i].BerserkerTimer)
		count += sanitizeFloat64(&objects[i].BuildPoints)
		count += sanitizeFloat64(&objects[i].GatherPointX)
		count += sanitizeFloat64(&objects[i].GatherPointY)
		count += sanitizeFloat64(&objects[i].EndpointX)
		count += sanitizeFloat64(&objects[i].EndpointY)
		count += sanitizeFloat64(&objects[i].Endpoint2X)
		count += sanitizeFloat64(&objects[i].Endpoint2Y)
		count += sanitizeFloat64(&objects[i].BuildingV68TrailerValue)
	}
	return count
}

func sanitizeFloat64(value *float64) int {
	if value == nil || (!math.IsNaN(*value) && !math.IsInf(*value, 0)) {
		return 0
	}
	*value = 0
	return 1
}

func BuildObjectShapes(path string, opts ObjectIndexOptions) (*ObjectShapeReport, error) {
	index, err := BuildObjectIndex(path, ObjectIndexOptions{ReferencedOnly: opts.ReferencedOnly})
	if err != nil {
		return nil, err
	}
	shapes := summarizeObjectShapes(index.Objects)
	shownShapes := shapes
	if opts.Limit > 0 && len(shownShapes) > opts.Limit {
		shownShapes = shownShapes[:opts.Limit]
	}
	summary := ObjectShapeSummary{
		Candidates:             index.Summary.Candidates,
		ReferencedCandidates:   index.Summary.ReferencedCandidates,
		ShapeCount:             len(shapes),
		DecodedPrefixBytes:     index.Summary.DecodedPrefixBytes,
		DecodedBodyPrefixBytes: index.Summary.DecodedBodyPrefixBytes,
		BoundedBodyBytes:       index.Summary.BoundedBodyBytes,
		OpaqueBodyBytes:        index.Summary.OpaqueBodyBytes,
	}
	for i, shape := range shapes {
		if i == 0 || shape.RecordBytes < summary.MinRecordBytes {
			summary.MinRecordBytes = shape.RecordBytes
		}
		if shape.RecordBytes > summary.MaxRecordBytes {
			summary.MaxRecordBytes = shape.RecordBytes
		}
		if shape.EndConfidence == "final_body_to_player_span_end" {
			summary.FinalBodyShapeCount++
		}
		if shape.Referenced > 0 {
			summary.ReferencedShapeCount++
		} else {
			summary.UnreferencedShapeCount++
		}
	}
	return &ObjectShapeReport{
		Path:         path,
		Method:       "save68_initial_object_shape_profile_from_candidate_spans",
		Verification: "structure_verified_shape_profile_not_object_body_parse",
		Summary:      summary,
		Shown:        len(shownShapes),
		Shapes:       shownShapes,
		Warnings:     index.Warnings,
	}, nil
}

func collectRefsFromEvents(events []ReplayEvent) commandObjectRefs {
	refs := commandObjectRefs{Any: map[int]int{}, Target: map[int]int{}, Selected: map[int]int{}}
	for _, event := range events {
		for _, id := range event.ObjectIDs {
			if id > 0 {
				refs.Any[id]++
				refs.Selected[id]++
			}
		}
		if event.TargetID > 0 {
			refs.Any[event.TargetID]++
			refs.Target[event.TargetID]++
		}
	}
	return refs
}

func AnnotateEventObjects(events []ReplayEvent, index *ObjectIndexReport) {
	byID := map[int]ObjectReference{}
	for _, object := range index.Objects {
		byID[object.ObjectID] = object.Reference()
	}
	for i := range events {
		if ref, ok := byID[events[i].TargetID]; ok {
			events[i].TargetObject = &ref
		}
		if ref, ok := byID[events[i].BuildingID]; ok {
			events[i].BuildingObject = &ref
		}
		seen := map[int]bool{}
		for _, id := range events[i].ObjectIDs {
			if seen[id] {
				continue
			}
			seen[id] = true
			if ref, ok := byID[id]; ok {
				events[i].ObjectRefs = append(events[i].ObjectRefs, ref)
			}
		}
	}
}

func (object ObjectCandidate) Reference() ObjectReference {
	return ObjectReference{
		ObjectID:       object.ObjectID,
		OwnerID:        object.OwnerID,
		OwnerLabel:     object.OwnerLabel,
		RecordType:     object.RecordType,
		RecordTypeName: object.RecordTypeName,
		UnitID:         object.UnitID,
		Class:          object.Class,
		X:              object.X,
		Y:              object.Y,
		Confidence:     object.Confidence,
	}
}

func scanObjectCandidates(header []byte, spine *HeaderSpine, coverage *CoverageReport, refs commandObjectRefs, opts ObjectIndexOptions) ([]ObjectCandidate, int) {
	width, height := mapDimensionsFromCoverage(coverage)
	byID := map[int][]ObjectCandidate{}
	for _, player := range spine.Players {
		var playerCandidates []ObjectCandidate
		for off := player.PayloadSpanStart; off+initialObjectCandidatePrefixBytes <= player.PayloadSpanEnd && off+initialObjectCandidatePrefixBytes <= len(header); off++ {
			candidate, ok := objectCandidateAt(header, off, player, width, height, refs)
			if !ok {
				continue
			}
			playerCandidates = append(playerCandidates, candidate)
		}
		sort.Slice(playerCandidates, func(i, j int) bool { return playerCandidates[i].Offset < playerCandidates[j].Offset })
		for i := range playerCandidates {
			prefixEnd := playerCandidates[i].Offset + playerCandidates[i].PrefixBytes
			next := player.PayloadSpanEnd
			confidence := "final_body_to_player_span_end"
			if i+1 < len(playerCandidates) {
				next = playerCandidates[i+1].Offset
				confidence = "bounded_by_next_candidate_prefix"
			}
			playerCandidates[i].NextOffset = next
			playerCandidates[i].RecordBytes = next - playerCandidates[i].Offset
			playerCandidates[i].BodyBytes = maxReplayInt(0, next-prefixEnd)
			sampleEnd := next
			if sampleEnd > prefixEnd+32 {
				sampleEnd = prefixEnd + 32
			}
			if sampleEnd > prefixEnd && sampleEnd <= len(header) {
				playerCandidates[i].BodySampleHex = HexSample(header[prefixEnd:sampleEnd], 32)
			}
			if sprite, ok := parseObjectSpriteListPrefix(header, prefixEnd, next, playerCandidates[i].HasObjectProps); ok {
				playerCandidates[i].HasSpriteList = sprite.HasSpriteList
				playerCandidates[i].SpriteEntries = sprite.Entries
				playerCandidates[i].SpriteListBytes = sprite.End - prefixEnd
				playerCandidates[i].DecodedBodyPrefixBytes = sprite.End - prefixEnd
				if ext, ok := parseObjectDEExtensionPrefix(header, sprite.End, next); ok {
					playerCandidates[i].ParticleTypes = ext.ParticleTypes
					playerCandidates[i].ParticleNames = ext.ParticleNames
					playerCandidates[i].DEExtensionBytes = ext.End - sprite.End
					playerCandidates[i].DEExtensionStrings = ext.Strings
					playerCandidates[i].DecodedBodyPrefixBytes = ext.End - prefixEnd
					if staticTail, ok := parseObjectStaticTailPrefix(header, ext.End, next); ok {
						playerCandidates[i].StaticTailBytes = staticTail.End - ext.End
						playerCandidates[i].DecodedBodyPrefixBytes = staticTail.End - prefixEnd
						if moving, ok := parseObjectMovingPrefix(header, staticTail.End, next, playerCandidates[i].RecordType); ok {
							playerCandidates[i].TurnSpeed = roundedCoord(moving.TurnSpeed)
							playerCandidates[i].MovingPrefixBytes = moving.End - staticTail.End
							playerCandidates[i].Angle = roundedCoord(moving.Angle)
							playerCandidates[i].NumPathData = moving.NumPathData
							playerCandidates[i].HasFuturePathData = moving.HasFuturePathData
							playerCandidates[i].HasMovementData = moving.HasMovementData
							playerCandidates[i].NumUserWaypoints = moving.NumUserWaypoints
							playerCandidates[i].HasSubstitutePosition = moving.HasSubstitutePosition
							playerCandidates[i].DecodedBodyPrefixBytes = moving.End - prefixEnd
							if actionCombat, ok := parseObjectActionCombatPrefix(header, moving.End, next, playerCandidates[i].RecordType); ok {
								playerCandidates[i].ActionCombatPrefixBytes = actionCombat.End - moving.End
								playerCandidates[i].ActionWaiting = actionCombat.Waiting
								playerCandidates[i].ActionCommandFlag = actionCombat.CommandFlag
								playerCandidates[i].ActionSelectedGroupInfo = actionCombat.SelectedGroupInfo
								playerCandidates[i].ActionType = actionCombat.ActionType
								playerCandidates[i].FormationID = actionCombat.FormationID
								playerCandidates[i].FormationRow = actionCombat.FormationRow
								playerCandidates[i].FormationCol = actionCombat.FormationCol
								playerCandidates[i].AttackTimer = roundedCoord(actionCombat.AttackTimer)
								playerCandidates[i].CaptureFlag = actionCombat.CaptureFlag
								playerCandidates[i].AttackCount = actionCombat.AttackCount
								playerCandidates[i].DecodedBodyPrefixBytes = actionCombat.End - prefixEnd
								if combat, ok := parseObjectCombatPrefix(header, actionCombat.End, next, playerCandidates[i].RecordType); ok {
									playerCandidates[i].CombatPrefixBytes = combat.End - actionCombat.End
									playerCandidates[i].HasAI = combat.HasAI
									playerCandidates[i].DecodedBodyPrefixBytes = combat.End - prefixEnd
									if combatTail, ok := parseObjectCombatTailPrefix(header, combat.End, next, playerCandidates[i].RecordType, combat.HasAI); ok {
										playerCandidates[i].CombatTailPrefixBytes = combatTail.End - combat.End
										playerCandidates[i].HasDEPosition = combatTail.HasDEPosition
										playerCandidates[i].TownBellFlag = combatTail.TownBellFlag
										playerCandidates[i].TownBellTargetID = combatTail.TownBellTargetID
										playerCandidates[i].TownBellTargetX = roundedCoord(combatTail.TownBellTargetX)
										playerCandidates[i].TownBellTargetY = roundedCoord(combatTail.TownBellTargetY)
										playerCandidates[i].TownBellAction = combatTail.TownBellAction
										playerCandidates[i].BerserkerTimer = roundedCoord(combatTail.BerserkerTimer)
										playerCandidates[i].NumBuilders = combatTail.NumBuilders
										playerCandidates[i].NumHealers = combatTail.NumHealers
										playerCandidates[i].DecodedBodyPrefixBytes = combatTail.End - prefixEnd
										if trailer, ok := parseObjectCombatV68TrailerPrefix(header, combatTail.End, next, playerCandidates[i].RecordType); ok {
											playerCandidates[i].CombatV68TrailerBytes = trailer.End - combatTail.End
											playerCandidates[i].CombatV68PrimaryObjectID = trailer.PrimaryObjectID
											playerCandidates[i].CombatV68SecondaryObjectID = trailer.SecondaryObjectID
											playerCandidates[i].CombatV68TrailerKind = trailer.Kind
											playerCandidates[i].CombatV68TrailerCounter = trailer.Counter
											playerCandidates[i].DecodedBodyPrefixBytes = trailer.End - prefixEnd
										}
										actionStart := prefixEnd + playerCandidates[i].DecodedBodyPrefixBytes
										if actions, ok := parseObjectV68ActionBlocksPrefix(header, actionStart, next); ok {
											playerCandidates[i].V68ActionBlockBytes = actions.End - actionStart
											playerCandidates[i].V68ActionBlockCount = actions.Count
											playerCandidates[i].DecodedBodyPrefixBytes = actions.End - prefixEnd
										}
										stateStart := prefixEnd + playerCandidates[i].DecodedBodyPrefixBytes
										if state, ok := parseObjectV68PostActionStatePrefix(header, stateStart, next); ok {
											playerCandidates[i].V68PostActionStateBytes += state.End - stateStart
											playerCandidates[i].DecodedBodyPrefixBytes = state.End - prefixEnd
											actionStart = prefixEnd + playerCandidates[i].DecodedBodyPrefixBytes
											if actions, ok := parseObjectV68ActionBlocksPrefix(header, actionStart, next); ok {
												playerCandidates[i].V68ActionBlockBytes += actions.End - actionStart
												playerCandidates[i].V68ActionBlockCount += actions.Count
												playerCandidates[i].DecodedBodyPrefixBytes = actions.End - prefixEnd
											}
										} else if state, ok := parseObjectV68PostActionStateExtendedPrefix(header, stateStart, next); ok {
											playerCandidates[i].V68PostActionStateBytes += state.End - stateStart
											playerCandidates[i].DecodedBodyPrefixBytes = state.End - prefixEnd
											positionStart := prefixEnd + playerCandidates[i].DecodedBodyPrefixBytes
											if position, ok := parseObjectV68PositionOrderTailPrefix(header, positionStart, next); ok {
												playerCandidates[i].V68PositionOrderTailBytes += position.End - positionStart
												playerCandidates[i].DecodedBodyPrefixBytes = position.End - prefixEnd
												actionStart = prefixEnd + playerCandidates[i].DecodedBodyPrefixBytes
												if actions, ok := parseObjectV68ActionBlocksPrefix(header, actionStart, next); ok {
													playerCandidates[i].V68ActionBlockBytes += actions.End - actionStart
													playerCandidates[i].V68ActionBlockCount += actions.Count
													playerCandidates[i].DecodedBodyPrefixBytes = actions.End - prefixEnd
												}
											}
										}
										finalStart := prefixEnd + playerCandidates[i].DecodedBodyPrefixBytes
										if marker, ok := parseObjectFinalSpanTailMarker(header, finalStart, next, confidence); ok {
											playerCandidates[i].FinalSpanTailMarkerBytes = marker.End - finalStart
											playerCandidates[i].DecodedBodyPrefixBytes = marker.End - prefixEnd
										}
										if building, ok := parseObjectBuildingPrefix(header, combatTail.End, next, playerCandidates[i].RecordType); ok {
											playerCandidates[i].BuildingPrefixBytes = building.End - combatTail.End
											playerCandidates[i].Built = building.Built
											playerCandidates[i].BuildPoints = roundedCoord(building.BuildPoints)
											playerCandidates[i].UniqueBuildID = building.UniqueBuildID
											playerCandidates[i].Culture = building.Culture
											playerCandidates[i].Burning = building.Burning
											playerCandidates[i].LastBurnTime = building.LastBurnTime
											playerCandidates[i].LastGarrisonTime = building.LastGarrisonTime
											playerCandidates[i].RelicCount = building.RelicCount
											playerCandidates[i].SpecificRelicCount = building.SpecificRelicCount
											playerCandidates[i].GatherPointExists = building.GatherPointExists
											playerCandidates[i].GatherPointX = roundedCoord(building.GatherPointX)
											playerCandidates[i].GatherPointY = roundedCoord(building.GatherPointY)
											playerCandidates[i].GatherPointObjectID = building.GatherPointObjectID
											playerCandidates[i].GatherPointUnitTypeID = building.GatherPointUnitTypeID
											playerCandidates[i].DesolidFlag = building.DesolidFlag
											playerCandidates[i].PendingOrder = building.PendingOrder
											playerCandidates[i].LinkedOwner = building.LinkedOwner
											playerCandidates[i].LinkedChildren = building.LinkedChildren
											playerCandidates[i].CapturedUnitCount = building.CapturedUnitCount
											playerCandidates[i].DecodedBodyPrefixBytes = building.End - prefixEnd
											if extraActions, ok := parseEmptyActionListPrefix(header, building.End, next); ok {
												playerCandidates[i].BuildingExtraActionsBytes = extraActions.End - building.End
												playerCandidates[i].DecodedBodyPrefixBytes = extraActions.End - prefixEnd
												if queue, ok := parseObjectBuildingQueueHeader(header, extraActions.End, next); ok {
													playerCandidates[i].BuildingQueueHeaderBytes = queue.End - extraActions.End
													playerCandidates[i].ProductionQueueCapacity = queue.Capacity
													playerCandidates[i].DecodedBodyPrefixBytes = queue.End - prefixEnd
													if tail, ok := parseObjectBuildingTailPrefix(header, queue.End, next); ok {
														playerCandidates[i].BuildingTailPrefixBytes = tail.End - queue.End
														playerCandidates[i].EndpointX = roundedCoord(tail.EndpointX)
														playerCandidates[i].EndpointY = roundedCoord(tail.EndpointY)
														playerCandidates[i].Endpoint2X = roundedCoord(tail.Endpoint2X)
														playerCandidates[i].Endpoint2Y = roundedCoord(tail.Endpoint2Y)
														playerCandidates[i].GateLocked = tail.GateLocked
														playerCandidates[i].FirstUpdateRaw = tail.FirstUpdateRaw
														playerCandidates[i].CloseTimerRaw = tail.CloseTimerRaw
														playerCandidates[i].TerrainType = tail.TerrainType
														playerCandidates[i].SemiAsleep = tail.SemiAsleep
														playerCandidates[i].SnowFlag = tail.SnowFlag
														playerCandidates[i].DecodedBodyPrefixBytes = tail.End - prefixEnd
														if trailer, ok := parseObjectBuildingV68TrailerPrefix(header, tail.End, next); ok {
															playerCandidates[i].BuildingV68TrailerBytes = trailer.End - tail.End
															playerCandidates[i].BuildingV68TrailerFlag = trailer.Flag
															playerCandidates[i].BuildingV68TrailerValue = roundedCoord(trailer.Value)
															playerCandidates[i].BuildingV68TrailerKind = trailer.Kind
															playerCandidates[i].BuildingV68LinkedObjectIDs = trailer.LinkedObjectIDs
															playerCandidates[i].DecodedBodyPrefixBytes = trailer.End - prefixEnd
														}
														if confidence == "bounded_by_next_candidate_prefix" {
															zeroStart := prefixEnd + playerCandidates[i].DecodedBodyPrefixBytes
															if zeroTail, ok := parseZeroPrefixPadding(header, zeroStart, next, 16); ok {
																playerCandidates[i].ZeroTailPaddingBytes = zeroTail.End - zeroStart
																playerCandidates[i].DecodedBodyPrefixBytes = zeroTail.End - prefixEnd
																if trailer, ok := parseObjectBuildingV68TrailerPrefix(header, zeroTail.End, next); ok {
																	playerCandidates[i].BuildingV68TrailerBytes = trailer.End - zeroTail.End
																	playerCandidates[i].BuildingV68TrailerFlag = trailer.Flag
																	playerCandidates[i].BuildingV68TrailerValue = roundedCoord(trailer.Value)
																	playerCandidates[i].BuildingV68TrailerKind = trailer.Kind
																	playerCandidates[i].BuildingV68LinkedObjectIDs = trailer.LinkedObjectIDs
																	playerCandidates[i].DecodedBodyPrefixBytes = trailer.End - prefixEnd
																}
															}
														}
														if playerCandidates[i].BuildingV68TrailerBytes == 0 {
															zeroStart := prefixEnd + playerCandidates[i].DecodedBodyPrefixBytes
															if zeroTail, ok := parseZeroPrefixPadding(header, zeroStart, next, 16); ok {
																if trailer, ok := parseObjectBuildingV68TrailerPrefix(header, zeroTail.End, next); ok {
																	playerCandidates[i].ZeroTailPaddingBytes = zeroTail.End - zeroStart
																	playerCandidates[i].DecodedBodyPrefixBytes = zeroTail.End - prefixEnd
																	playerCandidates[i].BuildingV68TrailerBytes = trailer.End - zeroTail.End
																	playerCandidates[i].BuildingV68TrailerFlag = trailer.Flag
																	playerCandidates[i].BuildingV68TrailerValue = roundedCoord(trailer.Value)
																	playerCandidates[i].BuildingV68TrailerKind = trailer.Kind
																	playerCandidates[i].BuildingV68LinkedObjectIDs = trailer.LinkedObjectIDs
																	playerCandidates[i].DecodedBodyPrefixBytes = trailer.End - prefixEnd
																}
															}
														}
														actionStart := prefixEnd + playerCandidates[i].DecodedBodyPrefixBytes
														if actions, ok := parseObjectV68ActionBlocksPrefix(header, actionStart, next); ok {
															playerCandidates[i].V68ActionBlockBytes = actions.End - actionStart
															playerCandidates[i].V68ActionBlockCount = actions.Count
															playerCandidates[i].DecodedBodyPrefixBytes = actions.End - prefixEnd
														}
														stateStart := prefixEnd + playerCandidates[i].DecodedBodyPrefixBytes
														if state, ok := parseObjectV68PostActionStatePrefix(header, stateStart, next); ok {
															playerCandidates[i].V68PostActionStateBytes += state.End - stateStart
															playerCandidates[i].DecodedBodyPrefixBytes = state.End - prefixEnd
															actionStart = prefixEnd + playerCandidates[i].DecodedBodyPrefixBytes
															if actions, ok := parseObjectV68ActionBlocksPrefix(header, actionStart, next); ok {
																playerCandidates[i].V68ActionBlockBytes += actions.End - actionStart
																playerCandidates[i].V68ActionBlockCount += actions.Count
																playerCandidates[i].DecodedBodyPrefixBytes = actions.End - prefixEnd
															}
														} else if state, ok := parseObjectV68PostActionStateExtendedPrefix(header, stateStart, next); ok {
															playerCandidates[i].V68PostActionStateBytes += state.End - stateStart
															playerCandidates[i].DecodedBodyPrefixBytes = state.End - prefixEnd
															positionStart := prefixEnd + playerCandidates[i].DecodedBodyPrefixBytes
															if position, ok := parseObjectV68PositionOrderTailPrefix(header, positionStart, next); ok {
																playerCandidates[i].V68PositionOrderTailBytes += position.End - positionStart
																playerCandidates[i].DecodedBodyPrefixBytes = position.End - prefixEnd
																actionStart = prefixEnd + playerCandidates[i].DecodedBodyPrefixBytes
																if actions, ok := parseObjectV68ActionBlocksPrefix(header, actionStart, next); ok {
																	playerCandidates[i].V68ActionBlockBytes += actions.End - actionStart
																	playerCandidates[i].V68ActionBlockCount += actions.Count
																	playerCandidates[i].DecodedBodyPrefixBytes = actions.End - prefixEnd
																}
															}
														}
														finalStart := prefixEnd + playerCandidates[i].DecodedBodyPrefixBytes
														if marker, ok := parseObjectFinalSpanTailMarker(header, finalStart, next, confidence); ok {
															playerCandidates[i].FinalSpanTailMarkerBytes = marker.End - finalStart
															playerCandidates[i].DecodedBodyPrefixBytes = marker.End - prefixEnd
														}
													}
												}
											}
										}
									}
								}
							}
						}
					}
				}
			}
			opaqueStart := prefixEnd + playerCandidates[i].DecodedBodyPrefixBytes
			opaqueSampleEnd := next
			if opaqueSampleEnd > opaqueStart+96 {
				opaqueSampleEnd = opaqueStart + 96
			}
			if opaqueSampleEnd > opaqueStart && opaqueSampleEnd <= len(header) {
				playerCandidates[i].OpaqueSampleHex = HexSample(header[opaqueStart:opaqueSampleEnd], 96)
			}
			playerCandidates[i].EndConfidence = confidence
			if opts.ReferencedOnly && playerCandidates[i].CommandRefs == 0 {
				continue
			}
			byID[playerCandidates[i].ObjectID] = append(byID[playerCandidates[i].ObjectID], playerCandidates[i])
		}
	}
	objects := make([]ObjectCandidate, 0, len(byID))
	ambiguous := 0
	for _, candidates := range byID {
		sort.Slice(candidates, func(i, j int) bool {
			if candidates[i].Score == candidates[j].Score {
				return candidates[i].Offset < candidates[j].Offset
			}
			return candidates[i].Score > candidates[j].Score
		})
		best := candidates[0]
		if len(candidates) > 1 {
			best.Ambiguous = true
			ambiguous++
		}
		objects = append(objects, best)
	}
	sort.Slice(objects, func(i, j int) bool {
		if objects[i].CommandRefs != objects[j].CommandRefs {
			return objects[i].CommandRefs > objects[j].CommandRefs
		}
		if objects[i].TargetRefs != objects[j].TargetRefs {
			return objects[i].TargetRefs > objects[j].TargetRefs
		}
		return objects[i].ObjectID < objects[j].ObjectID
	})
	return objects, ambiguous
}

type objectSpriteListPrefix struct {
	Start         int
	End           int
	HasSpriteList int
	Entries       int
	Types         []int
}

type objectDEExtensionPrefix struct {
	Start         int
	End           int
	ParticleTypes []int
	ParticleNames []string
	Strings       []string
}

type objectStaticTailPrefix struct {
	Start int
	End   int
}

type objectMovingPrefix struct {
	Start                 int
	End                   int
	TurnSpeed             float64
	Angle                 float64
	NumPathData           int
	HasFuturePathData     int
	HasMovementData       int
	NumUserWaypoints      int
	HasSubstitutePosition int
}

type objectActionCombatPrefix struct {
	Start             int
	End               int
	Waiting           int
	CommandFlag       int
	SelectedGroupInfo int
	ActionType        int
	FormationID       int
	FormationRow      int
	FormationCol      int
	AttackTimer       float64
	CaptureFlag       int
	AttackCount       int
}

type objectCombatPrefix struct {
	Start int
	End   int
	HasAI int
}

type objectCombatTailPrefix struct {
	Start            int
	End              int
	HasDEPosition    bool
	TownBellFlag     int
	TownBellTargetID int
	TownBellTargetX  float64
	TownBellTargetY  float64
	TownBellAction   int
	BerserkerTimer   float64
	NumBuilders      int
	NumHealers       int
}

type objectCombatV68TrailerPrefix struct {
	Start             int
	End               int
	PrimaryObjectID   int
	SecondaryObjectID int
	Kind              int
	Counter           int
}

type objectV68ActionBlocksPrefix struct {
	Start int
	End   int
	Count int
}

type objectV68PostActionStatePrefix struct {
	Start int
	End   int
}

type objectV68PositionOrderTailPrefix struct {
	Start int
	End   int
}

type objectFinalSpanTailMarker struct {
	Start int
	End   int
}

type objectBuildingPrefix struct {
	Start                 int
	End                   int
	Built                 int
	BuildPoints           float64
	UniqueBuildID         int
	Culture               int
	Burning               int
	LastBurnTime          int
	LastGarrisonTime      int
	RelicCount            int
	SpecificRelicCount    int
	GatherPointExists     int
	GatherPointX          float64
	GatherPointY          float64
	GatherPointObjectID   int
	GatherPointUnitTypeID int
	DesolidFlag           int
	PendingOrder          int
	LinkedOwner           int
	LinkedChildren        []int
	CapturedUnitCount     int
}

type objectEmptyActionListPrefix struct {
	Start int
	End   int
}

type objectBuildingQueueHeader struct {
	Start    int
	End      int
	Capacity int
}

type objectBuildingTailPrefix struct {
	Start          int
	End            int
	EndpointX      float64
	EndpointY      float64
	Endpoint2X     float64
	Endpoint2Y     float64
	GateLocked     int
	FirstUpdateRaw int
	CloseTimerRaw  int
	TerrainType    int
	SemiAsleep     int
	SnowFlag       int
}

type objectBuildingV68TrailerPrefix struct {
	Start           int
	End             int
	Flag            int
	Value           float64
	Kind            int
	LinkedObjectIDs []int
}

type objectZeroTailPadding struct {
	Start int
	End   int
}

func parseObjectSpriteListPrefix(header []byte, start int, limit int, hasObjectProps int) (objectSpriteListPrefix, bool) {
	if hasObjectProps != 0 || start < 0 || start >= limit || limit > len(header) {
		return objectSpriteListPrefix{}, false
	}
	hasSpriteList := int(header[start])
	if hasSpriteList == 0 {
		return objectSpriteListPrefix{Start: start, End: start + 1, HasSpriteList: hasSpriteList}, true
	}
	pos := start + 1
	var types []int
	entries := 0
	for pos < limit && entries <= 64 {
		spriteType := int(header[pos])
		pos++
		types = append(types, spriteType)
		if spriteType == 0 {
			return objectSpriteListPrefix{Start: start, End: pos, HasSpriteList: hasSpriteList, Entries: entries, Types: types}, true
		}
		if pos+13 > limit {
			return objectSpriteListPrefix{}, false
		}
		pos += 13
		if spriteType == 2 {
			if pos+17 > limit {
				return objectSpriteListPrefix{}, false
			}
			pos += 17
		}
		if pos+3 > limit {
			return objectSpriteListPrefix{}, false
		}
		pos += 3
		entries++
	}
	return objectSpriteListPrefix{}, false
}

func parseObjectDEExtensionPrefix(header []byte, start int, limit int) (objectDEExtensionPrefix, bool) {
	if start < 0 || start >= limit || limit > len(header) {
		return objectDEExtensionPrefix{}, false
	}
	pos := start
	result := objectDEExtensionPrefix{Start: start}
	for i := 0; i < 5; i++ {
		if pos+2 > limit {
			return objectDEExtensionPrefix{}, false
		}
		particleType := int(binary.LittleEndian.Uint16(header[pos:]))
		result.ParticleTypes = append(result.ParticleTypes, particleType)
		pos += 2
		if particleType == 1 {
			value, next, ok := deStringAt(header, pos, limit)
			if !ok || next+34 > limit {
				return objectDEExtensionPrefix{}, false
			}
			result.ParticleNames = append(result.ParticleNames, value)
			pos = next + 34
		}
	}
	if pos+7 > limit {
		return objectDEExtensionPrefix{}, false
	}
	pos += 7
	result.End = pos
	if pos+2 > limit {
		return result, true
	}
	optional := pos + 2
	first, next, ok := deStringAt(header, optional, limit)
	if !ok {
		return result, true
	}
	second, next, ok := deStringAt(header, next, limit)
	if !ok || next+2 > limit {
		return result, true
	}
	result.Strings = []string{first, second}
	result.End = next + 2
	return result, true
}

func parseObjectStaticTailPrefix(header []byte, start int, limit int) (objectStaticTailPrefix, bool) {
	if start < 0 || start+24 > limit || limit > len(header) {
		return objectStaticTailPrefix{}, false
	}
	return objectStaticTailPrefix{Start: start, End: start + 24}, true
}

func parseObjectMovingPrefix(header []byte, start int, limit int, recordType int) (objectMovingPrefix, bool) {
	if start < 0 || start+4 > limit || limit > len(header) {
		return objectMovingPrefix{}, false
	}
	pos := start
	turnSpeed := float64(math.Float32frombits(binary.LittleEndian.Uint32(header[pos:])))
	if math.IsNaN(turnSpeed) || math.IsInf(turnSpeed, 0) || turnSpeed < -10000 || turnSpeed > 10000 {
		return objectMovingPrefix{}, false
	}
	pos += 4
	if !objectRecordHasMovingPrefix(recordType) {
		return objectMovingPrefix{Start: start, End: pos, TurnSpeed: turnSpeed}, true
	}
	if pos+32 > limit {
		return objectMovingPrefix{}, false
	}
	pos += 4  // trail_remainder
	pos += 12 // velocity vector
	pos++     // de_move_byte
	angle := float64(math.Float32frombits(binary.LittleEndian.Uint32(header[pos:])))
	if math.IsNaN(angle) || math.IsInf(angle, 0) || angle < -10000 || angle > 10000 {
		return objectMovingPrefix{}, false
	}
	pos += 4
	pos += 4 // turn_towards_time
	pos += 3 // waiting_to_move + wait_delays_count + on_ground
	numPathData := int(binary.LittleEndian.Uint32(header[pos:]))
	if numPathData < 0 || numPathData > 10000 {
		return objectMovingPrefix{}, false
	}
	pos += 4
	if pos+numPathData*44 > limit {
		return objectMovingPrefix{}, false
	}
	pos += numPathData * 44
	if pos+4 > limit {
		return objectMovingPrefix{}, false
	}
	hasFuturePathData := int(binary.LittleEndian.Uint32(header[pos:]))
	pos += 4
	if hasFuturePathData < 0 || hasFuturePathData > 1000 {
		return objectMovingPrefix{}, false
	}
	if hasFuturePathData > 0 {
		if pos+44 > limit {
			return objectMovingPrefix{}, false
		}
		pos += 44
	}
	if pos+4 > limit {
		return objectMovingPrefix{}, false
	}
	hasMovementData := int(binary.LittleEndian.Uint32(header[pos:]))
	pos += 4
	if hasMovementData < 0 || hasMovementData > 1000 {
		return objectMovingPrefix{}, false
	}
	if hasMovementData > 0 {
		if pos+24 > limit {
			return objectMovingPrefix{}, false
		}
		pos += 24
	}
	if pos+36 > limit || !plausibleVector(header[pos:pos+12]) || !plausibleVector(header[pos+12:pos+24]) || !plausibleVector(header[pos+24:pos+36]) {
		return objectMovingPrefix{}, false
	}
	pos += 36
	if pos+8 > limit {
		return objectMovingPrefix{}, false
	}
	pos += 4 // last_move_time
	numUserWaypoints := int(binary.LittleEndian.Uint32(header[pos:]))
	pos += 4
	if numUserWaypoints < 0 || numUserWaypoints > 10000 || pos+numUserWaypoints*12 > limit {
		return objectMovingPrefix{}, false
	}
	pos += numUserWaypoints * 12
	if pos+4 > limit {
		return objectMovingPrefix{}, false
	}
	hasSubstitutePosition := int(binary.LittleEndian.Uint32(header[pos:]))
	pos += 4
	if hasSubstitutePosition < 0 || hasSubstitutePosition > 1000 {
		return objectMovingPrefix{}, false
	}
	if hasSubstitutePosition > 0 {
		if pos+12 > limit || !plausibleVector(header[pos:pos+12]) {
			return objectMovingPrefix{}, false
		}
		pos += 12
	}
	if pos+4 > limit {
		return objectMovingPrefix{}, false
	}
	pos += 4 // consecutive_substitute_count
	return objectMovingPrefix{
		Start:                 start,
		End:                   pos,
		TurnSpeed:             turnSpeed,
		Angle:                 angle,
		NumPathData:           numPathData,
		HasFuturePathData:     hasFuturePathData,
		HasMovementData:       hasMovementData,
		NumUserWaypoints:      numUserWaypoints,
		HasSubstitutePosition: hasSubstitutePosition,
	}, true
}

func parseObjectActionCombatPrefix(header []byte, start int, limit int, recordType int) (objectActionCombatPrefix, bool) {
	if start < 0 || limit > len(header) || !objectRecordHasActionPrefix(recordType) || start+6 > limit {
		return objectActionCombatPrefix{}, false
	}
	pos := start
	waiting := int(header[pos])
	pos++
	commandFlag := int(header[pos])
	pos++
	selectedGroupInfo := int(binary.LittleEndian.Uint16(header[pos:]))
	pos += 2
	actionType := int(binary.LittleEndian.Uint16(header[pos:]))
	if actionType != 0 {
		return objectActionCombatPrefix{}, false
	}
	pos += 2
	if !objectRecordHasBaseCombatPrefix(recordType) {
		return objectActionCombatPrefix{Start: start, End: pos, Waiting: waiting, CommandFlag: commandFlag, SelectedGroupInfo: selectedGroupInfo, ActionType: actionType}, true
	}
	if pos+14 > limit {
		return objectActionCombatPrefix{}, false
	}
	formationID := int(header[pos])
	formationRow := int(header[pos+1])
	formationCol := int(header[pos+2])
	pos += 3
	attackTimer := float64(math.Float32frombits(binary.LittleEndian.Uint32(header[pos:])))
	if math.IsNaN(attackTimer) || math.IsInf(attackTimer, 0) || attackTimer < -100000 || attackTimer > 100000000 {
		return objectActionCombatPrefix{}, false
	}
	pos += 4
	captureFlag := int(header[pos])
	pos += 3 // capture_flag + multi_unified_points + large_object_radius
	attackCount := int(int32(binary.LittleEndian.Uint32(header[pos:])))
	pos += 4
	return objectActionCombatPrefix{
		Start:             start,
		End:               pos,
		Waiting:           waiting,
		CommandFlag:       commandFlag,
		SelectedGroupInfo: selectedGroupInfo,
		ActionType:        actionType,
		FormationID:       formationID,
		FormationRow:      formationRow,
		FormationCol:      formationCol,
		AttackTimer:       attackTimer,
		CaptureFlag:       captureFlag,
		AttackCount:       attackCount,
	}, true
}

func parseObjectCombatPrefix(header []byte, start int, limit int, recordType int) (objectCombatPrefix, bool) {
	if start < 0 || limit > len(header) || !objectRecordHasCombatPrefix(recordType) || start+87 > limit {
		return objectCombatPrefix{}, false
	}
	pos := start
	pos += 14 // DE combat bytes
	pos += 4  // de_unknown_66_3_1
	pos += 16 // de_2
	pos += 4  // de_4
	pos += 19 // de_5
	pos++     // next_volley
	pos++     // using_special_animation
	ownBase := int(header[pos])
	if ownBase != 0 {
		return objectCombatPrefix{}, false
	}
	pos++     // own_base
	pos += 12 // attribute_amounts
	pos += 2  // decay_timer
	pos += 4  // raider_build_countdown
	pos += 4  // locked_down_count
	pos++     // inside_garrison_count
	hasAI := int(binary.LittleEndian.Uint32(header[pos:]))
	if hasAI < 0 || hasAI > 1000 {
		return objectCombatPrefix{}, false
	}
	pos += 4
	return objectCombatPrefix{Start: start, End: pos, HasAI: hasAI}, true
}

func parseObjectCombatTailPrefix(header []byte, start int, limit int, recordType int, hasAI int) (objectCombatTailPrefix, bool) {
	if start < 0 || limit > len(header) || !objectRecordHasCombatPrefix(recordType) || hasAI != 0 || start+5 > limit {
		return objectCombatTailPrefix{}, false
	}
	pos := start
	hasDEPosition := header[pos] != 0 && !bytesEqual(header[pos:pos+5], []byte{0x00, 0xff, 0xff, 0xff, 0xff})
	if hasDEPosition {
		if pos+13 > limit || !plausibleVector(header[pos:pos+12]) {
			return objectCombatTailPrefix{}, false
		}
		pos += 13 // position vector + flag
	}
	if pos+43 > limit {
		return objectCombatTailPrefix{}, false
	}
	townBellFlag := int(header[pos])
	pos++
	townBellTargetID := int(int32(binary.LittleEndian.Uint32(header[pos:])))
	pos += 4
	if !plausibleFloat32OrUnset(header[pos:]) {
		return objectCombatTailPrefix{}, false
	}
	townBellTargetX := float64(math.Float32frombits(binary.LittleEndian.Uint32(header[pos:])))
	pos += 4
	if !plausibleFloat32OrUnset(header[pos:]) {
		return objectCombatTailPrefix{}, false
	}
	townBellTargetY := float64(math.Float32frombits(binary.LittleEndian.Uint32(header[pos:])))
	pos += 4
	pos += 4 // town_bell_target_id_2
	pos += 4 // town_bell_target_type
	townBellAction := int(int32(binary.LittleEndian.Uint32(header[pos:])))
	pos += 4
	if !plausibleFloat32OrUnset(header[pos:]) {
		return objectCombatTailPrefix{}, false
	}
	berserkerTimer := float64(math.Float32frombits(binary.LittleEndian.Uint32(header[pos:])))
	pos += 4
	numBuilders := int(header[pos])
	pos++
	numHealers := int(header[pos])
	pos++
	pos += 4 // de_unknown
	pos += 4 // de_unknown2
	pos += 4 // de_unknown4
	if pos+93 > limit {
		return objectCombatTailPrefix{}, false
	}
	pos += 48 // de_unknown5
	pos += 40 // de_unknown6
	pos += 2  // de_unknown7
	pos += 2  // de_unknown8 for has_ai != 15
	pos++     // de_unknown_64_3_1
	return objectCombatTailPrefix{
		Start:            start,
		End:              pos,
		HasDEPosition:    hasDEPosition,
		TownBellFlag:     townBellFlag,
		TownBellTargetID: townBellTargetID,
		TownBellTargetX:  townBellTargetX,
		TownBellTargetY:  townBellTargetY,
		TownBellAction:   townBellAction,
		BerserkerTimer:   berserkerTimer,
		NumBuilders:      numBuilders,
		NumHealers:       numHealers,
	}, true
}

func parseObjectCombatV68TrailerPrefix(header []byte, start int, limit int, recordType int) (objectCombatV68TrailerPrefix, bool) {
	const trailerBytes = 209
	if recordType != 70 || start < 0 || start+trailerBytes > limit || limit > len(header) {
		return objectCombatV68TrailerPrefix{}, false
	}
	raw := header[start : start+trailerBytes]
	if !allBytes(raw[0:5], 0x00) || raw[5] != 1 || !plausibleVector(raw[6:18]) || raw[18] != 0 {
		return objectCombatV68TrailerPrefix{}, false
	}
	if binary.LittleEndian.Uint32(raw[19:23]) != 0xffffffff || !allBytes(raw[23:35], 0x00) {
		return objectCombatV68TrailerPrefix{}, false
	}
	primary := int(binary.LittleEndian.Uint32(raw[35:39]))
	secondary := int(binary.LittleEndian.Uint32(raw[39:43]))
	if primary < 0 || primary > 300000 || secondary < 0 || secondary > 300000 {
		return objectCombatV68TrailerPrefix{}, false
	}
	if !allBytes(raw[43:47], 0x00) || binary.LittleEndian.Uint32(raw[47:51]) != 0xffffffff || !allBytes(raw[51:57], 0x00) {
		return objectCombatV68TrailerPrefix{}, false
	}
	if raw[57] != 1 || !plausibleVector(raw[58:70]) {
		return objectCombatV68TrailerPrefix{}, false
	}
	kind := int(binary.LittleEndian.Uint16(raw[70:72]))
	if kind < 0 || kind > 1000 || raw[72] != 0 {
		return objectCombatV68TrailerPrefix{}, false
	}
	if binary.LittleEndian.Uint32(raw[73:77]) != 0xffffffff || !bytesEqual(raw[77:85], []byte{0x00, 0x00, 0x80, 0xbf, 0x00, 0x00, 0x80, 0xbf}) {
		return objectCombatV68TrailerPrefix{}, false
	}
	if !allBytes(raw[85:101], 0xff) || !bytesEqual(raw[101:109], []byte{0x00, 0x00, 0x80, 0xbf, 0x00, 0x00, 0x80, 0xbf}) {
		return objectCombatV68TrailerPrefix{}, false
	}
	if !allBytes(raw[109:121], 0xff) || !allBytes(raw[121:147], 0x00) {
		return objectCombatV68TrailerPrefix{}, false
	}
	counter := int(binary.LittleEndian.Uint32(raw[147:151]))
	if counter < 0 || counter > 100000 || !allBytes(raw[151:209], 0x00) {
		return objectCombatV68TrailerPrefix{}, false
	}
	return objectCombatV68TrailerPrefix{
		Start:             start,
		End:               start + trailerBytes,
		PrimaryObjectID:   primary,
		SecondaryObjectID: secondary,
		Kind:              kind,
		Counter:           counter,
	}, true
}

func parseObjectV68ActionBlocksPrefix(header []byte, start int, limit int) (objectV68ActionBlocksPrefix, bool) {
	if start < 0 || start >= limit || limit > len(header) {
		return objectV68ActionBlocksPrefix{}, false
	}
	pos := start
	count := 0
	for pos < limit {
		n, ok := parseObjectV68ActionBlockLength(header, pos, limit)
		if !ok {
			break
		}
		pos += n
		count++
	}
	if count == 0 {
		return objectV68ActionBlocksPrefix{}, false
	}
	return objectV68ActionBlocksPrefix{Start: start, End: pos, Count: count}, true
}

func parseObjectV68ActionBlockLength(header []byte, start int, limit int) (int, bool) {
	if start+139 > limit || limit > len(header) {
		return 0, false
	}
	raw := header[start:limit]
	if (raw[0] != 0x14 && raw[0] != 0x46) || raw[1] > 8 {
		return 0, false
	}
	if binary.LittleEndian.Uint32(raw[6:10]) != 0xffffffff || !plausibleFloat32(raw[10:14]) {
		return 0, false
	}
	mode := int(binary.LittleEndian.Uint32(raw[14:18]))
	if mode < 0 || mode > 1000 {
		return 0, false
	}
	if !plausibleFloat32(raw[24:28]) || !plausibleFloat32(raw[28:32]) {
		return 0, false
	}
	if !bytesEqual(raw[68:76], []byte{0xff, 0xff, 0xff, 0xff, 0xfe, 0xff, 0xff, 0xff}) {
		return 0, false
	}
	if binary.LittleEndian.Uint32(raw[76:80]) != 1 {
		return 0, false
	}
	if start+173 <= limit && parseObjectV68ActionExtendedTail(raw[80:173]) {
		return 173, true
	}
	if parseObjectV68ActionShortTail(raw[80:139]) {
		return 139, true
	}
	return 0, false
}

func parseObjectV68ActionShortTail(raw []byte) bool {
	if len(raw) != 59 {
		return false
	}
	if raw[0] != 1 || !allBytes(raw[1:13], 0x00) || !allBytes(raw[13:21], 0xff) {
		return false
	}
	if !bytesEqual(raw[21:29], []byte{0x60, 0x0a, 0x00, 0x00, 0x60, 0x0a, 0x00, 0x00}) || !bytesEqual(raw[29:33], []byte{0x01, 0x01, 0x01, 0x00}) {
		return false
	}
	if !allBytes(raw[33:43], 0xff) || !allBytes(raw[43:47], 0x00) || binary.LittleEndian.Uint32(raw[47:51]) != 0xffffffff {
		return false
	}
	return allBytes(raw[51:59], 0x00)
}

func parseObjectV68ActionExtendedTail(raw []byte) bool {
	if len(raw) != 93 {
		return false
	}
	if raw[0] != 1 || raw[1] > 8 {
		return false
	}
	if !allBytes(raw[47:55], 0xff) || !bytesEqual(raw[55:63], []byte{0x60, 0x0a, 0x00, 0x00, 0x60, 0x0a, 0x00, 0x00}) {
		return false
	}
	if !bytesEqual(raw[63:67], []byte{0x01, 0x01, 0x01, 0x00}) {
		return false
	}
	if !allBytes(raw[67:77], 0xff) || !allBytes(raw[77:81], 0x00) || binary.LittleEndian.Uint32(raw[81:85]) != 0xffffffff {
		return false
	}
	return allBytes(raw[85:93], 0x00)
}

func parseObjectV68PostActionStatePrefix(header []byte, start int, limit int) (objectV68PostActionStatePrefix, bool) {
	const blockBytes = 356
	if start < 0 || start+blockBytes > limit || limit > len(header) {
		return objectV68PostActionStatePrefix{}, false
	}
	raw := header[start : start+blockBytes]
	if binary.LittleEndian.Uint32(raw[204:208]) != 0x1388 {
		return objectV68PostActionStatePrefix{}, false
	}
	if binary.LittleEndian.Uint32(raw[220:224]) != 0xffffffff {
		return objectV68PostActionStatePrefix{}, false
	}
	if !bytesEqual(raw[224:232], []byte{0x00, 0x00, 0x80, 0xbf, 0x00, 0x00, 0x80, 0xbf}) {
		return objectV68PostActionStatePrefix{}, false
	}
	if !allBytes(raw[232:248], 0xff) || !bytesEqual(raw[248:256], []byte{0x00, 0x00, 0x80, 0xbf, 0x00, 0x00, 0x80, 0xbf}) {
		return objectV68PostActionStatePrefix{}, false
	}
	if !allBytes(raw[256:268], 0xff) || !allBytes(raw[268:294], 0x00) {
		return objectV68PostActionStatePrefix{}, false
	}
	counter := int(binary.LittleEndian.Uint32(raw[294:298]))
	if counter < 0 || counter > 100000 || !allBytes(raw[298:356], 0x00) {
		return objectV68PostActionStatePrefix{}, false
	}
	return objectV68PostActionStatePrefix{Start: start, End: start + blockBytes}, true
}

func parseObjectV68PostActionStateExtendedPrefix(header []byte, start int, limit int) (objectV68PostActionStatePrefix, bool) {
	const blockBytes = 340
	if start < 0 || start+blockBytes > limit || limit > len(header) {
		return objectV68PostActionStatePrefix{}, false
	}
	raw := header[start : start+blockBytes]
	if binary.LittleEndian.Uint32(raw[204:208]) != 0x1388 {
		return objectV68PostActionStatePrefix{}, false
	}
	if !allBytes(raw[219:243], 0xff) || !bytesEqual(raw[243:259], []byte{0x00, 0x00, 0x80, 0xbf, 0x00, 0x00, 0x80, 0xbf, 0x00, 0x00, 0x80, 0xbf, 0x00, 0x00, 0x00, 0x40}) {
		return objectV68PostActionStatePrefix{}, false
	}
	if !allBytes(raw[259:275], 0xff) || binary.LittleEndian.Uint32(raw[303:307]) != 0xffffffff {
		return objectV68PostActionStatePrefix{}, false
	}
	if !bytesEqual(raw[307:323], []byte{0x00, 0x00, 0x00, 0x40, 0x00, 0x00, 0x80, 0xbf, 0x00, 0x00, 0x80, 0xbf, 0x00, 0x00, 0x80, 0xbf}) {
		return objectV68PostActionStatePrefix{}, false
	}
	if !allBytes(raw[323:340], 0x00) {
		return objectV68PostActionStatePrefix{}, false
	}
	return objectV68PostActionStatePrefix{Start: start, End: start + blockBytes}, true
}

func parseObjectV68PositionOrderTailPrefix(header []byte, start int, limit int) (objectV68PositionOrderTailPrefix, bool) {
	const blockBytes = 204
	if start < 0 || start+blockBytes > limit || limit > len(header) {
		return objectV68PositionOrderTailPrefix{}, false
	}
	raw := header[start : start+blockBytes]
	if raw[0] != 1 || !plausibleVector(raw[1:13]) {
		return objectV68PositionOrderTailPrefix{}, false
	}
	if raw[13] != 0 || binary.LittleEndian.Uint32(raw[14:18]) != 0xffffffff || !allBytes(raw[18:30], 0x00) {
		return objectV68PositionOrderTailPrefix{}, false
	}
	if !plausibleLooseObjectIDOrUnset(binary.LittleEndian.Uint32(raw[30:34])) || !plausibleLooseObjectIDOrUnset(binary.LittleEndian.Uint32(raw[34:38])) {
		return objectV68PositionOrderTailPrefix{}, false
	}
	if !allBytes(raw[38:42], 0x00) || binary.LittleEndian.Uint32(raw[42:46]) != 0xffffffff || !allBytes(raw[46:52], 0x00) {
		return objectV68PositionOrderTailPrefix{}, false
	}
	if raw[52] != 1 || !plausibleVector(raw[53:65]) {
		return objectV68PositionOrderTailPrefix{}, false
	}
	kind := int(binary.LittleEndian.Uint16(raw[65:67]))
	if kind < 0 || kind > 1000 || raw[67] != 0 || binary.LittleEndian.Uint32(raw[68:72]) != 0xffffffff {
		return objectV68PositionOrderTailPrefix{}, false
	}
	if !bytesEqual(raw[72:80], []byte{0x00, 0x00, 0x80, 0xbf, 0x00, 0x00, 0x80, 0xbf}) || !allBytes(raw[80:96], 0xff) {
		return objectV68PositionOrderTailPrefix{}, false
	}
	if !bytesEqual(raw[96:104], []byte{0x00, 0x00, 0x80, 0xbf, 0x00, 0x00, 0x80, 0xbf}) || !allBytes(raw[104:116], 0xff) {
		return objectV68PositionOrderTailPrefix{}, false
	}
	if !allBytes(raw[116:142], 0x00) {
		return objectV68PositionOrderTailPrefix{}, false
	}
	counter := int(binary.LittleEndian.Uint32(raw[142:146]))
	if counter < 0 || counter > 100000 || !allBytes(raw[146:204], 0x00) {
		return objectV68PositionOrderTailPrefix{}, false
	}
	return objectV68PositionOrderTailPrefix{Start: start, End: start + blockBytes}, true
}

func parseObjectFinalSpanTailMarker(header []byte, start int, limit int, confidence string) (objectFinalSpanTailMarker, bool) {
	if confidence != "final_body_to_player_span_end" || start < 0 || start+50 != limit || limit > len(header) {
		return objectFinalSpanTailMarker{}, false
	}
	raw := header[start:limit]
	expected := []byte{
		0x00, 0x0b, 0x40, 0x00, 0x00, 0x80, 0x00, 0x00, 0x00, 0x00,
		0x0b, 0x00, 0x02, 0x00, 0x00, 0x02, 0x00, 0x00, 0x00, 0x0b,
		0x00, 0x0b, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0b,
	}
	if !bytesEqual(raw, expected) {
		return objectFinalSpanTailMarker{}, false
	}
	return objectFinalSpanTailMarker{Start: start, End: limit}, true
}

func parseObjectBuildingPrefix(header []byte, start int, limit int, recordType int) (objectBuildingPrefix, bool) {
	if start < 0 || limit > len(header) || recordType != 80 || start+71 > limit {
		return objectBuildingPrefix{}, false
	}
	pos := start
	built := int(header[pos])
	pos++
	if !plausibleFloat32OrUnset(header[pos:]) {
		return objectBuildingPrefix{}, false
	}
	buildPoints := float64(math.Float32frombits(binary.LittleEndian.Uint32(header[pos:])))
	pos += 4
	uniqueBuildID := int(int32(binary.LittleEndian.Uint32(header[pos:])))
	pos += 4
	culture := int(header[pos])
	pos++
	burning := int(header[pos])
	pos++
	lastBurnTime := int(int32(binary.LittleEndian.Uint32(header[pos:])))
	pos += 4
	lastGarrisonTime := int(int32(binary.LittleEndian.Uint32(header[pos:])))
	pos += 4
	relicCount := int(int32(binary.LittleEndian.Uint32(header[pos:])))
	pos += 4
	specificRelicCount := int(int32(binary.LittleEndian.Uint32(header[pos:])))
	pos += 4
	gatherPointExists := int(int32(binary.LittleEndian.Uint32(header[pos:])))
	pos += 4
	if !plausibleVector(header[pos : pos+12]) {
		return objectBuildingPrefix{}, false
	}
	gatherPointX := float64(math.Float32frombits(binary.LittleEndian.Uint32(header[pos:])))
	gatherPointY := float64(math.Float32frombits(binary.LittleEndian.Uint32(header[pos+4:])))
	pos += 12
	gatherPointObjectID := int(int32(binary.LittleEndian.Uint32(header[pos:])))
	pos += 4
	gatherPointUnitTypeID := int(int16(binary.LittleEndian.Uint16(header[pos:])))
	pos += 2
	desolidFlag := int(header[pos])
	pos++
	pendingOrder := int(int32(binary.LittleEndian.Uint32(header[pos:])))
	pos += 4
	linkedOwner := int(int32(binary.LittleEndian.Uint32(header[pos:])))
	pos += 4
	linkedChildren := []int{
		int(int32(binary.LittleEndian.Uint32(header[pos:]))),
		int(int32(binary.LittleEndian.Uint32(header[pos+4:]))),
		int(int32(binary.LittleEndian.Uint32(header[pos+8:]))),
	}
	pos += 12
	capturedUnitCount := int(header[pos])
	pos++
	return objectBuildingPrefix{
		Start:                 start,
		End:                   pos,
		Built:                 built,
		BuildPoints:           buildPoints,
		UniqueBuildID:         uniqueBuildID,
		Culture:               culture,
		Burning:               burning,
		LastBurnTime:          lastBurnTime,
		LastGarrisonTime:      lastGarrisonTime,
		RelicCount:            relicCount,
		SpecificRelicCount:    specificRelicCount,
		GatherPointExists:     gatherPointExists,
		GatherPointX:          gatherPointX,
		GatherPointY:          gatherPointY,
		GatherPointObjectID:   gatherPointObjectID,
		GatherPointUnitTypeID: gatherPointUnitTypeID,
		DesolidFlag:           desolidFlag,
		PendingOrder:          pendingOrder,
		LinkedOwner:           linkedOwner,
		LinkedChildren:        linkedChildren,
		CapturedUnitCount:     capturedUnitCount,
	}, true
}

func parseEmptyActionListPrefix(header []byte, start int, limit int) (objectEmptyActionListPrefix, bool) {
	if start < 0 || start+2 > limit || limit > len(header) {
		return objectEmptyActionListPrefix{}, false
	}
	if binary.LittleEndian.Uint16(header[start:]) != 0 {
		return objectEmptyActionListPrefix{}, false
	}
	return objectEmptyActionListPrefix{Start: start, End: start + 2}, true
}

func parseObjectBuildingQueueHeader(header []byte, start int, limit int) (objectBuildingQueueHeader, bool) {
	if start < 0 || start+7 > limit || limit > len(header) {
		return objectBuildingQueueHeader{}, false
	}
	pos := start
	capacity := int(binary.LittleEndian.Uint16(header[pos:]))
	if capacity < 0 || capacity > 500 {
		return objectBuildingQueueHeader{}, false
	}
	pos += 2
	if pos+capacity*4+5 > limit {
		return objectBuildingQueueHeader{}, false
	}
	pos += capacity * 4 // production_queue entries: unit_type_id + count
	pos += 2            // size
	pos += 2            // production_queue_total_units
	pos++               // production_queue_enabled
	return objectBuildingQueueHeader{Start: start, End: pos, Capacity: capacity}, true
}

func parseObjectBuildingTailPrefix(header []byte, start int, limit int) (objectBuildingTailPrefix, bool) {
	if start < 0 || start+73 > limit || limit > len(header) {
		return objectBuildingTailPrefix{}, false
	}
	pos := start
	if header[pos] != 0xff {
		return objectBuildingTailPrefix{}, false
	}
	pos++ // DE queue-action sentinel/alignment before endpoints.
	if !plausibleVectorOrUnset(header[pos:pos+12]) || !plausibleVectorOrUnset(header[pos+12:pos+24]) {
		return objectBuildingTailPrefix{}, false
	}
	endpointX := float64(math.Float32frombits(binary.LittleEndian.Uint32(header[pos:])))
	endpointY := float64(math.Float32frombits(binary.LittleEndian.Uint32(header[pos+4:])))
	pos += 12
	endpoint2X := float64(math.Float32frombits(binary.LittleEndian.Uint32(header[pos:])))
	endpoint2Y := float64(math.Float32frombits(binary.LittleEndian.Uint32(header[pos+4:])))
	pos += 12
	gateLocked := int(int32(binary.LittleEndian.Uint32(header[pos:])))
	pos += 4
	firstUpdateRaw := int(int32(binary.LittleEndian.Uint32(header[pos:])))
	pos += 4
	closeTimerRaw := int(int32(binary.LittleEndian.Uint32(header[pos:])))
	pos += 4
	terrainType := int(header[pos])
	pos++
	semiAsleep := int(header[pos])
	pos++
	snowFlag := int(header[pos])
	pos++
	pos++     // de_flag_unk
	pos += 2  // de_unk_2
	pos++     // de_unk_3
	pos += 4  // de_unk_4
	pos += 4  // de_unk_5
	pos += 12 // de_unk_6
	pos += 4  // de_unk_7
	pos += 5  // de_unknown_66_3_2
	return objectBuildingTailPrefix{
		Start:          start,
		End:            pos,
		EndpointX:      endpointX,
		EndpointY:      endpointY,
		Endpoint2X:     endpoint2X,
		Endpoint2Y:     endpoint2Y,
		GateLocked:     gateLocked,
		FirstUpdateRaw: firstUpdateRaw,
		CloseTimerRaw:  closeTimerRaw,
		TerrainType:    terrainType,
		SemiAsleep:     semiAsleep,
		SnowFlag:       snowFlag,
	}, true
}

func parseObjectBuildingV68TrailerPrefix(header []byte, start int, limit int) (objectBuildingV68TrailerPrefix, bool) {
	const trailerBytes = 154
	if start < 0 || start+trailerBytes > limit || limit > len(header) {
		return objectBuildingV68TrailerPrefix{}, false
	}
	raw := header[start : start+trailerBytes]
	if raw[0] != 1 || !plausibleFloat32(raw[1:5]) {
		return objectBuildingV68TrailerPrefix{}, false
	}
	if binary.LittleEndian.Uint32(raw[5:9]) != 0xffffffff {
		return objectBuildingV68TrailerPrefix{}, false
	}
	kind := int(binary.LittleEndian.Uint32(raw[9:13]))
	if kind < 0 || kind > 64 {
		return objectBuildingV68TrailerPrefix{}, false
	}
	if !allBytes(raw[13:43], 0x00) || !bytesEqual(raw[43:50], []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x00}) {
		return objectBuildingV68TrailerPrefix{}, false
	}
	for _, off := range []int{50, 54, 58, 62, 66} {
		if !plausibleObjectIDOrUnset(binary.LittleEndian.Uint32(raw[off : off+4])) {
			return objectBuildingV68TrailerPrefix{}, false
		}
	}
	if !allBytes(raw[70:79], 0x00) || binary.LittleEndian.Uint32(raw[79:83]) != 1 {
		return objectBuildingV68TrailerPrefix{}, false
	}
	if !allBytes(raw[83:110], 0x00) || binary.LittleEndian.Uint32(raw[110:114]) != 1 || !allBytes(raw[114:118], 0x00) {
		return objectBuildingV68TrailerPrefix{}, false
	}
	if !bytesEqual(raw[118:124], []byte{0xff, 0x00, 0x00, 0xff, 0xff, 0xff}) || !allBytes(raw[124:154], 0x00) {
		return objectBuildingV68TrailerPrefix{}, false
	}
	linked := make([]int, 0, 5)
	for _, off := range []int{50, 54, 58, 62, 66} {
		value := binary.LittleEndian.Uint32(raw[off : off+4])
		if value != 0xffffffff {
			linked = append(linked, int(value))
		}
	}
	return objectBuildingV68TrailerPrefix{
		Start:           start,
		End:             start + trailerBytes,
		Flag:            int(raw[0]),
		Value:           float64(math.Float32frombits(binary.LittleEndian.Uint32(raw[1:5]))),
		Kind:            kind,
		LinkedObjectIDs: linked,
	}, true
}

func parseZeroPrefixPadding(header []byte, start int, limit int, minBytes int) (objectZeroTailPadding, bool) {
	if start < 0 || start > limit || limit > len(header) {
		return objectZeroTailPadding{}, false
	}
	if start == limit {
		return objectZeroTailPadding{}, false
	}
	end := start
	for end < limit && header[end] == 0 {
		end++
	}
	if end-start < minBytes {
		return objectZeroTailPadding{}, false
	}
	return objectZeroTailPadding{Start: start, End: end}, true
}

func parseZeroTailPadding(header []byte, start int, limit int) (objectZeroTailPadding, bool) {
	if zero, ok := parseZeroPrefixPadding(header, start, limit, 1); ok {
		if zero.End == limit {
			return zero, true
		}
	}
	return objectZeroTailPadding{}, false
}

func allBytes(raw []byte, value byte) bool {
	for _, b := range raw {
		if b != value {
			return false
		}
	}
	return true
}

func plausibleObjectIDOrUnset(value uint32) bool {
	if value == 0xffffffff {
		return true
	}
	return value >= 100 && value <= 300000
}

func plausibleLooseObjectIDOrUnset(value uint32) bool {
	if value == 0xffffffff {
		return true
	}
	return value <= 300000
}

func plausibleFloat32(raw []byte) bool {
	if len(raw) < 4 {
		return false
	}
	value := float64(math.Float32frombits(binary.LittleEndian.Uint32(raw)))
	return !math.IsNaN(value) && !math.IsInf(value, 0) && value >= -100000000 && value <= 100000000
}

func plausibleFloat32OrUnset(raw []byte) bool {
	if len(raw) < 4 {
		return false
	}
	if binary.LittleEndian.Uint32(raw) == 0xffffffff {
		return true
	}
	return plausibleFloat32(raw)
}

func bytesEqual(a, b []byte) bool {
	if len(a) < len(b) {
		return false
	}
	for i := range b {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func plausibleVector(raw []byte) bool {
	if len(raw) < 12 {
		return false
	}
	for i := 0; i < 3; i++ {
		value := float64(math.Float32frombits(binary.LittleEndian.Uint32(raw[i*4:])))
		if math.IsNaN(value) || math.IsInf(value, 0) || value < -100000 || value > 100000 {
			return false
		}
	}
	return true
}

func plausibleVectorOrUnset(raw []byte) bool {
	if len(raw) < 12 {
		return false
	}
	for i := 0; i < 3; i++ {
		part := raw[i*4:]
		if binary.LittleEndian.Uint32(part) == 0xffffffff {
			continue
		}
		if !plausibleFloat32(part) {
			return false
		}
	}
	return true
}

func objectRecordHasMovingPrefix(recordType int) bool {
	switch recordType {
	case 30, 40, 50, 60, 70, 80:
		return true
	default:
		return false
	}
}

func objectRecordHasActionPrefix(recordType int) bool {
	switch recordType {
	case 40, 50, 70, 80:
		return true
	default:
		return false
	}
}

func objectRecordHasBaseCombatPrefix(recordType int) bool {
	switch recordType {
	case 50, 70, 80:
		return true
	default:
		return false
	}
}

func objectRecordHasCombatPrefix(recordType int) bool {
	switch recordType {
	case 70, 80:
		return true
	default:
		return false
	}
}

func deStringAt(data []byte, off int, limit int) (string, int, bool) {
	if off+4 > limit || off+4 > len(data) {
		return "", off, false
	}
	if data[off] != 0x60 || data[off+1] != 0x0a {
		return "", off, false
	}
	length := int(binary.LittleEndian.Uint16(data[off+2:]))
	start := off + 4
	end := start + length
	if length < 0 || end > limit || end > len(data) {
		return "", off, false
	}
	return decodeCString(data[start:end]), end, true
}

func objectCandidateAt(header []byte, off int, player InitialPlayerSpan, width int, height int, refs commandObjectRefs) (ObjectCandidate, bool) {
	if off+initialObjectCandidatePrefixBytes > len(header) {
		return ObjectCandidate{}, false
	}
	recordType := int(header[off])
	if !knownObjectRecordType(recordType) {
		return ObjectCandidate{}, false
	}
	owner := int(header[off+1])
	if owner != player.Index {
		return ObjectCandidate{}, false
	}
	unitID := int(int16(binary.LittleEndian.Uint16(header[off+2:])))
	if unitID < 0 || unitID > 5000 {
		return ObjectCandidate{}, false
	}
	objectID := int(binary.LittleEndian.Uint32(header[off+18:]))
	if objectID < 100 || objectID > 300000 {
		return ObjectCandidate{}, false
	}
	hitpoints := float64(math.Float32frombits(binary.LittleEndian.Uint32(header[off+10:])))
	if math.IsNaN(hitpoints) || math.IsInf(hitpoints, 0) || hitpoints < -100000 || hitpoints > 10000000 {
		return ObjectCandidate{}, false
	}
	x := float64(math.Float32frombits(binary.LittleEndian.Uint32(header[off+23:])))
	y := float64(math.Float32frombits(binary.LittleEndian.Uint32(header[off+27:])))
	z := float64(math.Float32frombits(binary.LittleEndian.Uint32(header[off+31:])))
	if !plausibleObjectXY(x, y, width, height) {
		return ObjectCandidate{}, false
	}
	commandRefs := refs.Any[objectID]
	targetRefs := refs.Target[objectID]
	selectedRefs := refs.Selected[objectID]
	score := objectCandidateScore(recordType, unitID, x, y, commandRefs, targetRefs)
	if commandRefs == 0 && recordType != 80 {
		return ObjectCandidate{}, false
	}
	if score < 8 {
		return ObjectCandidate{}, false
	}
	confidence := "candidate"
	if commandRefs > 0 && recordType == 80 {
		confidence = "replay_referenced_building_prefix"
	} else if commandRefs > 0 {
		confidence = "replay_referenced_prefix"
	} else if recordType == 80 {
		confidence = "building_prefix_unreferenced"
	}
	return ObjectCandidate{
		ObjectID:       objectID,
		OwnerID:        owner,
		OwnerLabel:     initialPlayerLabel(owner),
		RecordType:     recordType,
		RecordTypeName: objectRecordTypeName(recordType),
		UnitID:         unitID,
		Class:          objectClass(recordType, unitID),
		HitPoints:      roundedCoord(hitpoints),
		ObjectState:    int(header[off+14]),
		Facet:          int(header[off+22]),
		X:              roundedCoord(x),
		Y:              roundedCoord(y),
		Z:              roundedCoord(z),
		ResourceType:   int(int16(binary.LittleEndian.Uint16(header[off+43:]))),
		Amount:         roundedCoord(float64(math.Float32frombits(binary.LittleEndian.Uint32(header[off+45:])))),
		WorkerCount:    int(header[off+49]),
		CurrentDamage:  int(header[off+50]),
		UnderAttack:    header[off+52] != 0,
		GroupID:        int(int32(binary.LittleEndian.Uint32(header[off+55:]))),
		HasObjectProps: int(binary.LittleEndian.Uint16(header[off+78:])),
		Offset:         off,
		SpanStart:      player.Start,
		SpanEnd:        player.End,
		PrefixBytes:    initialObjectCandidatePrefixBytes,
		CommandRefs:    commandRefs,
		TargetRefs:     targetRefs,
		SelectedRefs:   selectedRefs,
		Confidence:     confidence,
		Score:          score,
	}, true
}

func objectCandidateScore(recordType int, unitID int, x float64, y float64, commandRefs int, targetRefs int) int {
	score := 0
	if recordType == 80 {
		score += 6
	}
	if commandRefs > 0 {
		score += 4
	}
	if targetRefs > 0 {
		score += 4
	}
	if unitID <= 3000 {
		score += 2
	}
	if x != 0 || y != 0 {
		score += 2
	}
	if isGateUnit(unitID) {
		score += 2
	}
	return score
}

func mapDimensionsFromCoverage(coverage *CoverageReport) (int, int) {
	for _, region := range coverage.Regions {
		if region.Space != "inflated_header" || region.Name != "map_info" || region.Details == nil {
			continue
		}
		width, _ := region.Details["width"].(int)
		height, _ := region.Details["height"].(int)
		return width, height
	}
	return 0, 0
}

func plausibleObjectXY(x, y float64, width, height int) bool {
	if math.IsNaN(x) || math.IsNaN(y) || math.IsInf(x, 0) || math.IsInf(y, 0) {
		return false
	}
	if x == 0 && y == 0 {
		return false
	}
	maxX := float64(width + 4)
	maxY := float64(height + 4)
	if width <= 0 {
		maxX = 512
	}
	if height <= 0 {
		maxY = 512
	}
	return x >= -4 && y >= -4 && x <= maxX && y <= maxY
}

func knownObjectRecordType(recordType int) bool {
	switch recordType {
	case 10, 20, 25, 30, 40, 50, 60, 70, 80, 90:
		return true
	default:
		return false
	}
}

func objectRecordTypeName(recordType int) string {
	switch recordType {
	case 10:
		return "static"
	case 20, 25:
		return "animated"
	case 30:
		return "moving"
	case 40:
		return "action"
	case 50:
		return "base_combat"
	case 60:
		return "missile"
	case 70:
		return "combat"
	case 80:
		return "building"
	case 90:
		return "static_other"
	default:
		return "unknown"
	}
}

func objectClass(recordType int, unitID int) string {
	if isGateUnit(unitID) {
		return "gate"
	}
	switch recordType {
	case 80:
		return "building"
	case 30, 40, 50, 70:
		return "unit"
	case 60:
		return "projectile"
	case 10, 20, 25, 90:
		return "static"
	default:
		return "unknown"
	}
}

func isGateUnit(unitID int) bool {
	switch unitID {
	case 64, 88, 487, 659, 667, 789, 792, 793, 797, 801:
		return true
	default:
		return false
	}
}

func summarizeObjectPlayers(objects []ObjectCandidate) []ObjectPlayerSummary {
	byPlayer := map[int]*ObjectPlayerSummary{}
	for _, obj := range objects {
		summary := byPlayer[obj.OwnerID]
		if summary == nil {
			summary = &ObjectPlayerSummary{
				PlayerID:     obj.OwnerID,
				Label:        obj.OwnerLabel,
				ByClass:      map[string]int{},
				ByUnitID:     map[int]int{},
				ByRecordType: map[int]int{},
			}
			byPlayer[obj.OwnerID] = summary
		}
		summary.Candidates++
		if obj.CommandRefs > 0 {
			summary.Referenced++
		}
		summary.ByClass[obj.Class]++
		summary.ByUnitID[obj.UnitID]++
		summary.ByRecordType[obj.RecordType]++
	}
	out := make([]ObjectPlayerSummary, 0, len(byPlayer))
	for _, summary := range byPlayer {
		out = append(out, *summary)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].PlayerID < out[j].PlayerID })
	return out
}

func summarizeObjectShapes(objects []ObjectCandidate) []ObjectShape {
	type shapeKey struct {
		recordType    int
		unitID        int
		class         string
		recordBytes   int
		prefixBytes   int
		bodyBytes     int
		decodedBody   int
		endConfidence string
	}
	byKey := map[shapeKey]*ObjectShape{}
	for _, obj := range objects {
		key := shapeKey{
			recordType:    obj.RecordType,
			unitID:        obj.UnitID,
			class:         obj.Class,
			recordBytes:   obj.RecordBytes,
			prefixBytes:   obj.PrefixBytes,
			bodyBytes:     obj.BodyBytes,
			decodedBody:   obj.DecodedBodyPrefixBytes,
			endConfidence: obj.EndConfidence,
		}
		shape := byKey[key]
		if shape == nil {
			shape = &ObjectShape{
				RecordType:            obj.RecordType,
				RecordTypeName:        obj.RecordTypeName,
				UnitID:                obj.UnitID,
				UnitName:              UnitDisplayName(obj.UnitID),
				Class:                 obj.Class,
				RecordBytes:           obj.RecordBytes,
				PrefixBytes:           obj.PrefixBytes,
				BodyBytes:             obj.BodyBytes,
				DecodedBodyPrefixEach: obj.DecodedBodyPrefixBytes,
				OpaqueBodyEach:        obj.BodyBytes - obj.DecodedBodyPrefixBytes,
				EndConfidence:         obj.EndConfidence,
				Owners:                map[int]int{},
				BodySampleHex:         obj.BodySampleHex,
				OpaqueSampleHex:       obj.OpaqueSampleHex,
			}
			byKey[key] = shape
		}
		shape.Count++
		if obj.CommandRefs > 0 {
			shape.Referenced++
		}
		shape.TargetRefs += obj.TargetRefs
		shape.SelectedRefs += obj.SelectedRefs
		shape.DecodedBodyPrefixBytes += obj.DecodedBodyPrefixBytes
		shape.OpaqueBodyBytes += obj.BodyBytes - obj.DecodedBodyPrefixBytes
		shape.Owners[obj.OwnerID]++
		if len(shape.ExampleObjectIDs) < 5 {
			shape.ExampleObjectIDs = append(shape.ExampleObjectIDs, obj.ObjectID)
		}
		if len(shape.ExampleOffsets) < 5 {
			shape.ExampleOffsets = append(shape.ExampleOffsets, obj.Offset)
		}
	}
	out := make([]ObjectShape, 0, len(byKey))
	for _, shape := range byKey {
		out = append(out, *shape)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		if out[i].Referenced != out[j].Referenced {
			return out[i].Referenced > out[j].Referenced
		}
		if out[i].RecordType != out[j].RecordType {
			return out[i].RecordType < out[j].RecordType
		}
		if out[i].UnitID != out[j].UnitID {
			return out[i].UnitID < out[j].UnitID
		}
		if out[i].RecordBytes != out[j].RecordBytes {
			return out[i].RecordBytes < out[j].RecordBytes
		}
		return out[i].EndConfidence < out[j].EndConfidence
	})
	return out
}

func roundedCoord(value float64) float64 {
	return math.Round(value*100) / 100
}

func maxReplayInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
