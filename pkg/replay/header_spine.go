package replay

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
)

type HeaderSpine struct {
	InitialStart          int                 `json:"initial_start"`
	InitialEnd            int                 `json:"initial_end,omitempty"`
	RestoreTime           uint32              `json:"restore_time"`
	NumParticles          uint32              `json:"num_particles"`
	Identifier            uint32              `json:"identifier"`
	PlayerCount           int                 `json:"player_count"`
	Players               []InitialPlayerSpan `json:"players,omitempty"`
	PostInitialTailStart  int                 `json:"post_initial_tail_start,omitempty"`
	TriggerStart          int                 `json:"trigger_start"`
	TriggerBoundaryCheck  string              `json:"trigger_boundary_check"`
	TriggerBoundarySource string              `json:"trigger_boundary_source"`
	Notes                 []string            `json:"notes,omitempty"`
}

type InitialPlayerSpan struct {
	Index                    int     `json:"index"`
	Label                    string  `json:"label"`
	Start                    int     `json:"start"`
	End                      int     `json:"end,omitempty"`
	Bytes                    int     `json:"bytes,omitempty"`
	Type                     byte    `json:"type"`
	Unknown                  byte    `json:"unknown"`
	Name                     string  `json:"name,omitempty"`
	NumHeaderData            uint32  `json:"num_header_data"`
	PayloadSpanStart         int     `json:"payload_span_start,omitempty"`
	PayloadSpanEnd           int     `json:"payload_span_end,omitempty"`
	PayloadSpanBytes         int     `json:"payload_span_bytes,omitempty"`
	AttributesStart          int     `json:"attributes_start,omitempty"`
	AttributesEnd            int     `json:"attributes_end,omitempty"`
	AttributesBytes          int     `json:"attributes_bytes,omitempty"`
	CameraX                  float64 `json:"camera_x,omitempty"`
	CameraY                  float64 `json:"camera_y,omitempty"`
	PostCameraUnknown        int32   `json:"post_camera_unknown,omitempty"`
	SpawnX                   uint16  `json:"spawn_x,omitempty"`
	SpawnY                   uint16  `json:"spawn_y,omitempty"`
	StartMetaByte            byte    `json:"start_meta_byte,omitempty"`
	CivilizationKey          string  `json:"civilization_key,omitempty"`
	ObjectMarkerStart        int     `json:"object_marker_start,omitempty"`
	ObjectMarkerEnd          int     `json:"object_marker_end,omitempty"`
	ObjectSpanStart          int     `json:"object_span_start,omitempty"`
	ObjectSpanEnd            int     `json:"object_span_end,omitempty"`
	ObjectSpanBytes          int     `json:"object_span_bytes,omitempty"`
	ObjectTailMarkerStart    int     `json:"object_tail_marker_start,omitempty"`
	ObjectTailMarkerEnd      int     `json:"object_tail_marker_end,omitempty"`
	ObjectCandidateBandStart int     `json:"object_candidate_band_start,omitempty"`
	ObjectCandidateBandEnd   int     `json:"object_candidate_band_end,omitempty"`
	ObjectCandidateBandBytes int     `json:"object_candidate_band_bytes,omitempty"`
	ObjectCandidateCount     int     `json:"object_candidate_count,omitempty"`
	ObjectBoundaryConfidence string  `json:"object_boundary_confidence,omitempty"`
	BoundaryMethod           string  `json:"boundary_method,omitempty"`
}

var initialObjectMarker = []byte{0x0b, 0x00, 0x2e, 0x00, 0x00, 0x00, 0x02, 0x00, 0x00}

func parseHeaderSpine(header []byte, initialStart int, numPlayers int, triggerStart int, save float64, mapWidth int, mapHeight int, refs commandObjectRefs) (*HeaderSpine, []ReplayRegion, []string) {
	var regions []ReplayRegion
	var warnings []string
	if numPlayers <= 0 {
		return nil, nil, []string{"replay metadata reported no initial players"}
	}
	if initialStart < 0 || initialStart+12 > len(header) || initialStart >= triggerStart {
		return nil, nil, []string{fmt.Sprintf("bad initial start %d for header=%d trigger_start=%d", initialStart, len(header), triggerStart)}
	}
	restoreTime := binary.LittleEndian.Uint32(header[initialStart:])
	numParticles := binary.LittleEndian.Uint32(header[initialStart+4:])
	if numParticles > 100000 {
		return nil, nil, []string{fmt.Sprintf("unreasonable initial particle count %d at %d", numParticles, initialStart+4)}
	}
	metadataEnd := initialStart + 8 + int(numParticles)*27 + 4
	if metadataEnd > len(header) || metadataEnd > triggerStart {
		return nil, nil, []string{fmt.Sprintf("initial metadata end %d exceeds trigger/header boundary", metadataEnd)}
	}
	identifier := binary.LittleEndian.Uint32(header[metadataEnd-4:])
	spine := &HeaderSpine{
		InitialStart:          initialStart,
		RestoreTime:           restoreTime,
		NumParticles:          numParticles,
		Identifier:            identifier,
		PlayerCount:           numPlayers,
		TriggerStart:          triggerStart,
		TriggerBoundaryCheck:  "not_independent_v68_tail_bounded_to_heuristic",
		TriggerBoundarySource: "triggergraph_region_locator",
		Notes: []string{
			"save_version 68 initial player payloads are span-framed, not field-decoded.",
			"object starts are not claimed when the older mgz object sentinel is absent.",
			"post-initial v68 tail remains bounded opaque; trigger boundary still uses the existing graph locator.",
		},
	}
	regions = append(regions, replayRegion("inflated_header", "initial_metadata", initialStart, metadataEnd, "decoded", "parsed", map[string]any{
		"restore_time":  restoreTime,
		"num_particles": numParticles,
		"identifier":    identifier,
		"player_count":  numPlayers,
	}))
	off := metadataEnd
	parsedPlayers := 0
	for i := 0; i < numPlayers; i++ {
		player, prefixEnd, objectStart, objectEnd, err := parseInitialPlayerPrefix(header, off, i, numPlayers, triggerStart)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("player %d prefix: %v", i, err))
			break
		}
		boundarySearchStart := prefixEnd
		if objectEnd > 0 {
			boundarySearchStart = objectEnd
		}
		var next int
		var ok bool
		if i < numPlayers-1 {
			next, ok = findNextInitialPlayerStart(header, boundarySearchStart, player.NumHeaderData, numPlayers, save, triggerStart)
			player.BoundaryMethod = "next_player_num_header_data_marker"
		} else {
			next, ok = findLastInitialPlayerEnd(header, boundarySearchStart, numPlayers, triggerStart)
			player.BoundaryMethod = "scenario_header_backtrack"
		}
		if !ok || next <= prefixEnd || next > triggerStart {
			warnings = append(warnings, fmt.Sprintf("player %d boundary not found after prefix at %d", i, prefixEnd))
			break
		}
		player.End = next
		player.Bytes = next - player.Start
		player.PayloadSpanStart = prefixEnd
		player.PayloadSpanEnd = next
		player.PayloadSpanBytes = next - prefixEnd
		attrs, attrErr := parseInitialPlayerAttributes(header, prefixEnd, next, player.NumHeaderData)
		if attrErr != nil {
			warnings = append(warnings, fmt.Sprintf("player %d attributes: %v", i, attrErr))
		} else {
			player.AttributesStart = attrs.Start
			player.AttributesEnd = attrs.End
			player.AttributesBytes = attrs.End - attrs.Start
			player.CameraX = attrs.CameraX
			player.CameraY = attrs.CameraY
			player.PostCameraUnknown = attrs.PostCameraUnknown
			player.SpawnX = attrs.SpawnX
			player.SpawnY = attrs.SpawnY
			player.StartMetaByte = attrs.StartMetaByte
			player.CivilizationKey = attrs.CivilizationKey
		}
		if objectEnd > 0 {
			player.ObjectSpanStart = objectEnd
			player.ObjectSpanEnd = next
			player.ObjectSpanBytes = next - objectEnd
			player.ObjectBoundaryConfidence = "object_marker_found"
		} else if player.AttributesEnd > 0 && player.AttributesEnd < next {
			player.ObjectSpanStart = player.AttributesEnd
			player.ObjectSpanEnd = next
			player.ObjectSpanBytes = next - player.AttributesEnd
			player.ObjectBoundaryConfidence = "after_player_attributes_object_start_unverified"
		} else {
			player.ObjectBoundaryConfidence = "object_start_unresolved_for_save68"
		}
		spine.Players = append(spine.Players, player)
		parsedPlayers++

		regions = append(regions, replayRegion("inflated_header", fmt.Sprintf("initial_player_%d_header", i), player.Start, prefixEnd, "decoded", "parsed_prefix", map[string]any{
			"label":           player.Label,
			"name":            player.Name,
			"type":            player.Type,
			"num_header_data": player.NumHeaderData,
		}))
		if player.AttributesEnd > 0 {
			regions = append(regions, replayRegion("inflated_header", fmt.Sprintf("initial_player_%d_attributes", i), player.AttributesStart, player.AttributesEnd, "decoded", "parsed_mgz_player_stats_and_view", map[string]any{
				"label":               player.Label,
				"stats_float_count":   int(player.NumHeaderData) * 2,
				"camera_x":            player.CameraX,
				"camera_y":            player.CameraY,
				"post_camera_unknown": player.PostCameraUnknown,
				"spawn_x":             player.SpawnX,
				"spawn_y":             player.SpawnY,
				"start_meta_byte":     player.StartMetaByte,
				"civilization_key":    player.CivilizationKey,
			}))
		}
		if objectEnd > 0 {
			if prefixEnd < objectStart {
				regions = append(regions, replayRegion("inflated_header", fmt.Sprintf("initial_player_%d_stats_to_object_marker", i), prefixEnd, objectStart, "bounded_opaque", "framed_by_object_marker", nil))
			}
			regions = append(regions, replayRegion("inflated_header", fmt.Sprintf("initial_player_%d_object_marker", i), objectStart, objectEnd, "decoded", "parsed_sentinel", nil))
			regions = append(regions, replayRegion("inflated_header", fmt.Sprintf("initial_player_%d_object_span", i), objectEnd, next, "bounded_opaque", "object_records_not_field_decoded", map[string]any{
				"label":        player.Label,
				"object_bytes": next - objectEnd,
			}))
		} else if player.AttributesEnd > 0 && player.AttributesEnd < next {
			objectTailRegions := segmentInitialObjectTail(header, &player, i, mapWidth, mapHeight, refs)
			spine.Players[len(spine.Players)-1] = player
			regions = append(regions, objectTailRegions...)
		} else if prefixEnd < next {
			regions = append(regions, replayRegion("inflated_header", fmt.Sprintf("initial_player_%d_payload_span", i), prefixEnd, next, "bounded_opaque", "player_payload_framed_object_start_unresolved", map[string]any{
				"label":          player.Label,
				"payload_bytes":  next - prefixEnd,
				"object_summary": "not_claimed",
			}))
		}
		off = next
	}
	if parsedPlayers == numPlayers && off+21 <= triggerStart {
		spine.InitialEnd = off + 21
		spine.PostInitialTailStart = spine.InitialEnd
		regions = append(regions, replayRegion("inflated_header", "initial_footer", off, spine.InitialEnd, "decoded", "parsed_padding", nil))
	} else {
		spine.InitialEnd = off
		spine.PostInitialTailStart = off
		if parsedPlayers != numPlayers {
			warnings = append(warnings, fmt.Sprintf("initial player walk stopped after %d/%d players", parsedPlayers, numPlayers))
		}
	}
	return spine, regions, warnings
}

const initialObjectCandidatePrefixBytes = 80

func segmentInitialObjectTail(header []byte, player *InitialPlayerSpan, playerIndex int, mapWidth int, mapHeight int, refs commandObjectRefs) []ReplayRegion {
	start := player.ObjectSpanStart
	end := player.ObjectSpanEnd
	if start <= 0 || end <= start || end > len(header) {
		return []ReplayRegion{replayRegion("inflated_header", fmt.Sprintf("initial_player_%d_object_or_tail_span", playerIndex), start, end, "bounded_opaque", "bad_tail_bounds", map[string]any{
			"label": player.Label,
		})}
	}
	regions := []ReplayRegion{}
	scanStart := start
	if start+3 <= end && header[start] == 0x0b && header[start+2] == 0x0b {
		player.ObjectTailMarkerStart = start
		player.ObjectTailMarkerEnd = start + 2
		markerValue := int(header[start+1])
		expected := (player.Index + 8) % 9
		confidence := "parsed_v68_rotating_list_marker"
		if markerValue != expected {
			confidence = "parsed_v68_list_marker_relation_unverified"
		}
		regions = append(regions, replayRegion("inflated_header", fmt.Sprintf("initial_player_%d_object_tail_marker", playerIndex), start, start+2, "decoded", confidence, map[string]any{
			"label":          player.Label,
			"marker_value":   markerValue,
			"expected_value": expected,
		}))
		scanStart = start + 2
	}
	var candidates []ObjectCandidate
	for off := scanStart; off+initialObjectCandidatePrefixBytes <= end && off+initialObjectCandidatePrefixBytes <= len(header); off++ {
		candidate, ok := objectCandidateAt(header, off, *player, mapWidth, mapHeight, refs)
		if !ok {
			continue
		}
		candidates = append(candidates, candidate)
	}
	if len(candidates) == 0 {
		if scanStart < end {
			regions = append(regions, replayRegion("inflated_header", fmt.Sprintf("initial_player_%d_object_tail_unsegmented", playerIndex), scanStart, end, "bounded_opaque", "no_plausible_object_prefix_band", map[string]any{
				"label":        player.Label,
				"object_bytes": end - scanStart,
			}))
		}
		return regions
	}
	firstCandidate := candidates[0].Offset
	lastCandidate := candidates[len(candidates)-1].Offset
	candidateEnd := lastCandidate + initialObjectCandidatePrefixBytes
	if candidateEnd > end {
		candidateEnd = end
	}
	player.ObjectCandidateBandStart = firstCandidate
	player.ObjectCandidateBandEnd = candidateEnd
	player.ObjectCandidateBandBytes = candidateEnd - firstCandidate
	player.ObjectCandidateCount = len(candidates)
	cursor := scanStart
	var previous *ObjectCandidate
	for idx, candidate := range candidates {
		if cursor < candidate.Offset {
			name := fmt.Sprintf("initial_player_%d_object_tail_gap_before_prefix_%03d", playerIndex, idx)
			confidence := "between_v68_object_prefixes"
			details := map[string]any{"label": player.Label}
			if idx == 0 {
				name = fmt.Sprintf("initial_player_%d_object_tail_before_candidate_prefixes", playerIndex)
				confidence = "before_first_plausible_object_prefix"
			} else if previous != nil {
				name = fmt.Sprintf("initial_player_%d_object_candidate_body_%03d", playerIndex, idx-1)
				confidence = "body_after_decoded_object_prefix"
				details["object_id"] = previous.ObjectID
				details["record_type"] = previous.RecordType
				details["unit_id"] = previous.UnitID
				details["record_bytes_to_next_prefix"] = candidate.Offset - previous.Offset
				details["body_bytes_after_prefix"] = candidate.Offset - cursor
			}
			regions = append(regions, replayRegion("inflated_header", name, cursor, candidate.Offset, "bounded_opaque", confidence, details))
		}
		prefixEnd := candidate.Offset + initialObjectCandidatePrefixBytes
		if prefixEnd > end {
			prefixEnd = end
		}
		regions = append(regions, replayRegion("inflated_header", fmt.Sprintf("initial_player_%d_object_candidate_prefix_%03d", playerIndex, idx), candidate.Offset, prefixEnd, "decoded", "parsed_object_record_prefix_only", map[string]any{
			"label":                        player.Label,
			"object_id":                    candidate.ObjectID,
			"owner_id":                     candidate.OwnerID,
			"record_type":                  candidate.RecordType,
			"record_type_name":             candidate.RecordTypeName,
			"unit_id":                      candidate.UnitID,
			"class":                        candidate.Class,
			"hitpoints":                    candidate.HitPoints,
			"object_state":                 candidate.ObjectState,
			"facet":                        candidate.Facet,
			"x":                            candidate.X,
			"y":                            candidate.Y,
			"z":                            candidate.Z,
			"resource_type":                candidate.ResourceType,
			"amount":                       candidate.Amount,
			"worker_count":                 candidate.WorkerCount,
			"current_damage":               candidate.CurrentDamage,
			"under_attack":                 candidate.UnderAttack,
			"group_id":                     candidate.GroupID,
			"has_object_props":             candidate.HasObjectProps,
			"has_sprite_list":              candidate.HasSpriteList,
			"sprite_entries":               candidate.SpriteEntries,
			"sprite_list_bytes":            candidate.SpriteListBytes,
			"particle_types":               candidate.ParticleTypes,
			"particle_names":               candidate.ParticleNames,
			"de_extension_bytes":           candidate.DEExtensionBytes,
			"de_extension_strings":         candidate.DEExtensionStrings,
			"static_tail_bytes":            candidate.StaticTailBytes,
			"turn_speed":                   candidate.TurnSpeed,
			"moving_prefix_bytes":          candidate.MovingPrefixBytes,
			"angle":                        candidate.Angle,
			"num_path_data":                candidate.NumPathData,
			"has_future_path":              candidate.HasFuturePathData,
			"has_movement":                 candidate.HasMovementData,
			"user_waypoints":               candidate.NumUserWaypoints,
			"has_substitute":               candidate.HasSubstitutePosition,
			"action_combat_prefix_bytes":   candidate.ActionCombatPrefixBytes,
			"combat_prefix_bytes":          candidate.CombatPrefixBytes,
			"has_ai":                       candidate.HasAI,
			"combat_tail_prefix_bytes":     candidate.CombatTailPrefixBytes,
			"has_de_position":              candidate.HasDEPosition,
			"building_prefix_bytes":        candidate.BuildingPrefixBytes,
			"building_extra_actions_bytes": candidate.BuildingExtraActionsBytes,
			"building_queue_header_bytes":  candidate.BuildingQueueHeaderBytes,
			"production_queue_capacity":    candidate.ProductionQueueCapacity,
			"building_tail_prefix_bytes":   candidate.BuildingTailPrefixBytes,
			"zero_tail_padding_bytes":      candidate.ZeroTailPaddingBytes,
			"command_refs":                 candidate.CommandRefs,
			"target_refs":                  candidate.TargetRefs,
			"selected_refs":                candidate.SelectedRefs,
			"confidence":                   candidate.Confidence,
		}))
		nextLimit := end
		if idx+1 < len(candidates) {
			nextLimit = candidates[idx+1].Offset
		}
		cursor = prefixEnd
		if sprite, ok := parseObjectSpriteListPrefix(header, prefixEnd, nextLimit, candidate.HasObjectProps); ok {
			regions = append(regions, replayRegion("inflated_header", fmt.Sprintf("initial_player_%d_object_sprite_list_prefix_%03d", playerIndex, idx), prefixEnd, sprite.End, "decoded", "parsed_sprite_list_prefix_only", map[string]any{
				"label":           player.Label,
				"object_id":       candidate.ObjectID,
				"has_sprite_list": sprite.HasSpriteList,
				"sprite_entries":  sprite.Entries,
				"sprite_types":    sprite.Types,
			}))
			cursor = sprite.End
			if ext, ok := parseObjectDEExtensionPrefix(header, sprite.End, nextLimit); ok {
				regions = append(regions, replayRegion("inflated_header", fmt.Sprintf("initial_player_%d_object_de_extension_prefix_%03d", playerIndex, idx), sprite.End, ext.End, "decoded", "parsed_de_extension_prefix_only", map[string]any{
					"label":          player.Label,
					"object_id":      candidate.ObjectID,
					"particle_types": ext.ParticleTypes,
					"particle_names": ext.ParticleNames,
					"strings":        ext.Strings,
				}))
				cursor = ext.End
				if staticTail, ok := parseObjectStaticTailPrefix(header, ext.End, nextLimit); ok {
					regions = append(regions, replayRegion("inflated_header", fmt.Sprintf("initial_player_%d_object_static_tail_prefix_%03d", playerIndex, idx), ext.End, staticTail.End, "decoded", "parsed_static_tail_prefix_only", map[string]any{
						"label":     player.Label,
						"object_id": candidate.ObjectID,
						"bytes":     staticTail.End - ext.End,
					}))
					cursor = staticTail.End
					if moving, ok := parseObjectMovingPrefix(header, staticTail.End, nextLimit, candidate.RecordType); ok {
						regions = append(regions, replayRegion("inflated_header", fmt.Sprintf("initial_player_%d_object_moving_prefix_%03d", playerIndex, idx), staticTail.End, moving.End, "decoded", "parsed_animated_moving_prefix_only", map[string]any{
							"label":            player.Label,
							"object_id":        candidate.ObjectID,
							"turn_speed":       moving.TurnSpeed,
							"angle":            moving.Angle,
							"num_path_data":    moving.NumPathData,
							"has_future_path":  moving.HasFuturePathData,
							"has_movement":     moving.HasMovementData,
							"user_waypoints":   moving.NumUserWaypoints,
							"has_substitute":   moving.HasSubstitutePosition,
							"record_type":      candidate.RecordType,
							"record_type_name": candidate.RecordTypeName,
						}))
						cursor = moving.End
						if actionCombat, ok := parseObjectActionCombatPrefix(header, moving.End, nextLimit, candidate.RecordType); ok {
							regions = append(regions, replayRegion("inflated_header", fmt.Sprintf("initial_player_%d_object_action_combat_prefix_%03d", playerIndex, idx), moving.End, actionCombat.End, "decoded", "parsed_idle_action_and_base_combat_prefix_only", map[string]any{
								"label":            player.Label,
								"object_id":        candidate.ObjectID,
								"record_type":      candidate.RecordType,
								"record_type_name": candidate.RecordTypeName,
								"bytes":            actionCombat.End - moving.End,
							}))
							cursor = actionCombat.End
							if combat, ok := parseObjectCombatPrefix(header, actionCombat.End, nextLimit, candidate.RecordType); ok {
								regions = append(regions, replayRegion("inflated_header", fmt.Sprintf("initial_player_%d_object_combat_prefix_%03d", playerIndex, idx), actionCombat.End, combat.End, "decoded", "parsed_de_combat_prefix_through_has_ai_only", map[string]any{
									"label":            player.Label,
									"object_id":        candidate.ObjectID,
									"record_type":      candidate.RecordType,
									"record_type_name": candidate.RecordTypeName,
									"has_ai":           combat.HasAI,
									"bytes":            combat.End - actionCombat.End,
								}))
								cursor = combat.End
								if combatTail, ok := parseObjectCombatTailPrefix(header, combat.End, nextLimit, candidate.RecordType, combat.HasAI); ok {
									regions = append(regions, replayRegion("inflated_header", fmt.Sprintf("initial_player_%d_object_combat_tail_prefix_%03d", playerIndex, idx), combat.End, combatTail.End, "decoded", "parsed_de_combat_tail_has_ai_zero_only", map[string]any{
										"label":            player.Label,
										"object_id":        candidate.ObjectID,
										"record_type":      candidate.RecordType,
										"record_type_name": candidate.RecordTypeName,
										"has_de_position":  combatTail.HasDEPosition,
										"bytes":            combatTail.End - combat.End,
									}))
									cursor = combatTail.End
									if trailer, ok := parseObjectCombatV68TrailerPrefix(header, combatTail.End, nextLimit, candidate.RecordType); ok {
										regions = append(regions, replayRegion("inflated_header", fmt.Sprintf("initial_player_%d_object_combat_v68_trailer_%03d", playerIndex, idx), combatTail.End, trailer.End, "decoded", "parsed_v68_combat_trailer_structure_semantics_partial", map[string]any{
											"label":               player.Label,
											"object_id":           candidate.ObjectID,
											"record_type":         candidate.RecordType,
											"record_type_name":    candidate.RecordTypeName,
											"primary_object_id":   trailer.PrimaryObjectID,
											"secondary_object_id": trailer.SecondaryObjectID,
											"kind":                trailer.Kind,
											"counter":             trailer.Counter,
											"bytes":               trailer.End - combatTail.End,
										}))
										cursor = trailer.End
									}
									if actions, ok := parseObjectV68ActionBlocksPrefix(header, cursor, nextLimit); ok {
										start := cursor
										regions = append(regions, replayRegion("inflated_header", fmt.Sprintf("initial_player_%d_object_v68_action_blocks_%03d", playerIndex, idx), start, actions.End, "decoded", "parsed_v68_action_state_blocks_structure_semantics_partial", map[string]any{
											"label":            player.Label,
											"object_id":        candidate.ObjectID,
											"record_type":      candidate.RecordType,
											"record_type_name": candidate.RecordTypeName,
											"count":            actions.Count,
											"bytes":            actions.End - start,
										}))
										cursor = actions.End
									}
									if state, ok := parseObjectV68PostActionStatePrefix(header, cursor, nextLimit); ok {
										start := cursor
										regions = append(regions, replayRegion("inflated_header", fmt.Sprintf("initial_player_%d_object_v68_post_action_state_%03d", playerIndex, idx), start, state.End, "decoded", "parsed_v68_post_action_state_structure_semantics_partial", map[string]any{
											"label":            player.Label,
											"object_id":        candidate.ObjectID,
											"record_type":      candidate.RecordType,
											"record_type_name": candidate.RecordTypeName,
											"bytes":            state.End - start,
										}))
										cursor = state.End
										if actions, ok := parseObjectV68ActionBlocksPrefix(header, cursor, nextLimit); ok {
											actionStart := cursor
											regions = append(regions, replayRegion("inflated_header", fmt.Sprintf("initial_player_%d_object_v68_action_blocks_after_state_%03d", playerIndex, idx), actionStart, actions.End, "decoded", "parsed_v68_action_state_blocks_structure_semantics_partial", map[string]any{
												"label":            player.Label,
												"object_id":        candidate.ObjectID,
												"record_type":      candidate.RecordType,
												"record_type_name": candidate.RecordTypeName,
												"count":            actions.Count,
												"bytes":            actions.End - actionStart,
											}))
											cursor = actions.End
										}
									} else if state, ok := parseObjectV68PostActionStateExtendedPrefix(header, cursor, nextLimit); ok {
										start := cursor
										regions = append(regions, replayRegion("inflated_header", fmt.Sprintf("initial_player_%d_object_v68_post_action_state_ext_%03d", playerIndex, idx), start, state.End, "decoded", "parsed_v68_post_action_state_extended_structure_semantics_partial", map[string]any{
											"label":            player.Label,
											"object_id":        candidate.ObjectID,
											"record_type":      candidate.RecordType,
											"record_type_name": candidate.RecordTypeName,
											"bytes":            state.End - start,
										}))
										cursor = state.End
										if position, ok := parseObjectV68PositionOrderTailPrefix(header, cursor, nextLimit); ok {
											positionStart := cursor
											regions = append(regions, replayRegion("inflated_header", fmt.Sprintf("initial_player_%d_object_v68_position_order_tail_%03d", playerIndex, idx), positionStart, position.End, "decoded", "parsed_v68_position_order_tail_structure_semantics_partial", map[string]any{
												"label":            player.Label,
												"object_id":        candidate.ObjectID,
												"record_type":      candidate.RecordType,
												"record_type_name": candidate.RecordTypeName,
												"bytes":            position.End - positionStart,
											}))
											cursor = position.End
											if actions, ok := parseObjectV68ActionBlocksPrefix(header, cursor, nextLimit); ok {
												actionStart := cursor
												regions = append(regions, replayRegion("inflated_header", fmt.Sprintf("initial_player_%d_object_v68_action_blocks_after_position_%03d", playerIndex, idx), actionStart, actions.End, "decoded", "parsed_v68_action_state_blocks_structure_semantics_partial", map[string]any{
													"label":            player.Label,
													"object_id":        candidate.ObjectID,
													"record_type":      candidate.RecordType,
													"record_type_name": candidate.RecordTypeName,
													"count":            actions.Count,
													"bytes":            actions.End - actionStart,
												}))
												cursor = actions.End
											}
										}
									}
									markerConfidence := "bounded_by_next_candidate_prefix"
									if idx+1 == len(candidates) {
										markerConfidence = "final_body_to_player_span_end"
									}
									if marker, ok := parseObjectFinalSpanTailMarker(header, cursor, nextLimit, markerConfidence); ok {
										start := cursor
										regions = append(regions, replayRegion("inflated_header", fmt.Sprintf("initial_player_%d_object_final_span_tail_marker_%03d", playerIndex, idx), start, marker.End, "decoded", "matched_final_span_tail_marker_structure_semantics_unknown", map[string]any{
											"label":            player.Label,
											"object_id":        candidate.ObjectID,
											"record_type":      candidate.RecordType,
											"record_type_name": candidate.RecordTypeName,
											"bytes":            marker.End - start,
										}))
										cursor = marker.End
									}
									if building, ok := parseObjectBuildingPrefix(header, combatTail.End, nextLimit, candidate.RecordType); ok {
										regions = append(regions, replayRegion("inflated_header", fmt.Sprintf("initial_player_%d_object_building_prefix_%03d", playerIndex, idx), combatTail.End, building.End, "decoded", "parsed_building_prefix_before_action_lists_only", map[string]any{
											"label":            player.Label,
											"object_id":        candidate.ObjectID,
											"record_type":      candidate.RecordType,
											"record_type_name": candidate.RecordTypeName,
											"bytes":            building.End - combatTail.End,
										}))
										cursor = building.End
										if extraActions, ok := parseEmptyActionListPrefix(header, building.End, nextLimit); ok {
											regions = append(regions, replayRegion("inflated_header", fmt.Sprintf("initial_player_%d_object_building_extra_actions_empty_%03d", playerIndex, idx), building.End, extraActions.End, "decoded", "parsed_empty_building_extra_actions_only", map[string]any{
												"label":            player.Label,
												"object_id":        candidate.ObjectID,
												"record_type":      candidate.RecordType,
												"record_type_name": candidate.RecordTypeName,
												"bytes":            extraActions.End - building.End,
											}))
											cursor = extraActions.End
											if queue, ok := parseObjectBuildingQueueHeader(header, extraActions.End, nextLimit); ok {
												regions = append(regions, replayRegion("inflated_header", fmt.Sprintf("initial_player_%d_object_building_queue_header_%03d", playerIndex, idx), extraActions.End, queue.End, "decoded", "parsed_building_queue_header_before_queue_actions_only", map[string]any{
													"label":            player.Label,
													"object_id":        candidate.ObjectID,
													"record_type":      candidate.RecordType,
													"record_type_name": candidate.RecordTypeName,
													"capacity":         queue.Capacity,
													"bytes":            queue.End - extraActions.End,
												}))
												cursor = queue.End
												if tail, ok := parseObjectBuildingTailPrefix(header, queue.End, nextLimit); ok {
													regions = append(regions, replayRegion("inflated_header", fmt.Sprintf("initial_player_%d_object_building_tail_prefix_%03d", playerIndex, idx), queue.End, tail.End, "decoded", "parsed_building_endpoint_and_de_tail_prefix_only", map[string]any{
														"label":            player.Label,
														"object_id":        candidate.ObjectID,
														"record_type":      candidate.RecordType,
														"record_type_name": candidate.RecordTypeName,
														"bytes":            tail.End - queue.End,
													}))
													cursor = tail.End
													if trailer, ok := parseObjectBuildingV68TrailerPrefix(header, tail.End, nextLimit); ok {
														regions = append(regions, replayRegion("inflated_header", fmt.Sprintf("initial_player_%d_object_building_v68_trailer_%03d", playerIndex, idx), tail.End, trailer.End, "decoded", "parsed_v68_building_trailer_structure_semantics_partial", map[string]any{
															"label":             player.Label,
															"object_id":         candidate.ObjectID,
															"record_type":       candidate.RecordType,
															"record_type_name":  candidate.RecordTypeName,
															"flag":              trailer.Flag,
															"value":             roundedCoord(trailer.Value),
															"kind":              trailer.Kind,
															"linked_object_ids": trailer.LinkedObjectIDs,
															"bytes":             trailer.End - tail.End,
														}))
														cursor = trailer.End
													}
													if idx+1 < len(candidates) {
														if zeroTail, ok := parseZeroPrefixPadding(header, cursor, nextLimit, 16); ok {
															zeroStart := cursor
															regions = append(regions, replayRegion("inflated_header", fmt.Sprintf("initial_player_%d_object_zero_tail_padding_%03d", playerIndex, idx), cursor, zeroTail.End, "decoded", "verified_zero_padding_prefix_before_variable_tail", map[string]any{
																"label":     player.Label,
																"object_id": candidate.ObjectID,
																"bytes":     zeroTail.End - zeroStart,
															}))
															cursor = zeroTail.End
															if trailer, ok := parseObjectBuildingV68TrailerPrefix(header, cursor, nextLimit); ok {
																regions = append(regions, replayRegion("inflated_header", fmt.Sprintf("initial_player_%d_object_building_v68_trailer_%03d", playerIndex, idx), cursor, trailer.End, "decoded", "parsed_v68_building_trailer_structure_semantics_partial", map[string]any{
																	"label":             player.Label,
																	"object_id":         candidate.ObjectID,
																	"record_type":       candidate.RecordType,
																	"record_type_name":  candidate.RecordTypeName,
																	"flag":              trailer.Flag,
																	"value":             roundedCoord(trailer.Value),
																	"kind":              trailer.Kind,
																	"linked_object_ids": trailer.LinkedObjectIDs,
																	"bytes":             trailer.End - cursor,
																}))
																cursor = trailer.End
															}
														}
													}
													if zeroTail, ok := parseZeroPrefixPadding(header, cursor, nextLimit, 16); ok {
														if trailer, ok := parseObjectBuildingV68TrailerPrefix(header, zeroTail.End, nextLimit); ok {
															zeroStart := cursor
															regions = append(regions, replayRegion("inflated_header", fmt.Sprintf("initial_player_%d_object_zero_tail_padding_%03d", playerIndex, idx), zeroStart, zeroTail.End, "decoded", "verified_zero_padding_prefix_before_v68_building_trailer", map[string]any{
																"label":     player.Label,
																"object_id": candidate.ObjectID,
																"bytes":     zeroTail.End - zeroStart,
															}))
															cursor = zeroTail.End
															regions = append(regions, replayRegion("inflated_header", fmt.Sprintf("initial_player_%d_object_building_v68_trailer_%03d", playerIndex, idx), cursor, trailer.End, "decoded", "parsed_v68_building_trailer_structure_semantics_partial", map[string]any{
																"label":             player.Label,
																"object_id":         candidate.ObjectID,
																"record_type":       candidate.RecordType,
																"record_type_name":  candidate.RecordTypeName,
																"flag":              trailer.Flag,
																"value":             roundedCoord(trailer.Value),
																"kind":              trailer.Kind,
																"linked_object_ids": trailer.LinkedObjectIDs,
																"bytes":             trailer.End - cursor,
															}))
															cursor = trailer.End
														}
													}
													if actions, ok := parseObjectV68ActionBlocksPrefix(header, cursor, nextLimit); ok {
														start := cursor
														regions = append(regions, replayRegion("inflated_header", fmt.Sprintf("initial_player_%d_object_v68_action_blocks_%03d", playerIndex, idx), start, actions.End, "decoded", "parsed_v68_action_state_blocks_structure_semantics_partial", map[string]any{
															"label":            player.Label,
															"object_id":        candidate.ObjectID,
															"record_type":      candidate.RecordType,
															"record_type_name": candidate.RecordTypeName,
															"count":            actions.Count,
															"bytes":            actions.End - start,
														}))
														cursor = actions.End
													}
													if state, ok := parseObjectV68PostActionStatePrefix(header, cursor, nextLimit); ok {
														start := cursor
														regions = append(regions, replayRegion("inflated_header", fmt.Sprintf("initial_player_%d_object_v68_post_action_state_%03d", playerIndex, idx), start, state.End, "decoded", "parsed_v68_post_action_state_structure_semantics_partial", map[string]any{
															"label":            player.Label,
															"object_id":        candidate.ObjectID,
															"record_type":      candidate.RecordType,
															"record_type_name": candidate.RecordTypeName,
															"bytes":            state.End - start,
														}))
														cursor = state.End
														if actions, ok := parseObjectV68ActionBlocksPrefix(header, cursor, nextLimit); ok {
															actionStart := cursor
															regions = append(regions, replayRegion("inflated_header", fmt.Sprintf("initial_player_%d_object_v68_action_blocks_after_state_%03d", playerIndex, idx), actionStart, actions.End, "decoded", "parsed_v68_action_state_blocks_structure_semantics_partial", map[string]any{
																"label":            player.Label,
																"object_id":        candidate.ObjectID,
																"record_type":      candidate.RecordType,
																"record_type_name": candidate.RecordTypeName,
																"count":            actions.Count,
																"bytes":            actions.End - actionStart,
															}))
															cursor = actions.End
														}
													} else if state, ok := parseObjectV68PostActionStateExtendedPrefix(header, cursor, nextLimit); ok {
														start := cursor
														regions = append(regions, replayRegion("inflated_header", fmt.Sprintf("initial_player_%d_object_v68_post_action_state_ext_%03d", playerIndex, idx), start, state.End, "decoded", "parsed_v68_post_action_state_extended_structure_semantics_partial", map[string]any{
															"label":            player.Label,
															"object_id":        candidate.ObjectID,
															"record_type":      candidate.RecordType,
															"record_type_name": candidate.RecordTypeName,
															"bytes":            state.End - start,
														}))
														cursor = state.End
														if position, ok := parseObjectV68PositionOrderTailPrefix(header, cursor, nextLimit); ok {
															positionStart := cursor
															regions = append(regions, replayRegion("inflated_header", fmt.Sprintf("initial_player_%d_object_v68_position_order_tail_%03d", playerIndex, idx), positionStart, position.End, "decoded", "parsed_v68_position_order_tail_structure_semantics_partial", map[string]any{
																"label":            player.Label,
																"object_id":        candidate.ObjectID,
																"record_type":      candidate.RecordType,
																"record_type_name": candidate.RecordTypeName,
																"bytes":            position.End - positionStart,
															}))
															cursor = position.End
															if actions, ok := parseObjectV68ActionBlocksPrefix(header, cursor, nextLimit); ok {
																actionStart := cursor
																regions = append(regions, replayRegion("inflated_header", fmt.Sprintf("initial_player_%d_object_v68_action_blocks_after_position_%03d", playerIndex, idx), actionStart, actions.End, "decoded", "parsed_v68_action_state_blocks_structure_semantics_partial", map[string]any{
																	"label":            player.Label,
																	"object_id":        candidate.ObjectID,
																	"record_type":      candidate.RecordType,
																	"record_type_name": candidate.RecordTypeName,
																	"count":            actions.Count,
																	"bytes":            actions.End - actionStart,
																}))
																cursor = actions.End
															}
														}
													}
													markerConfidence := "bounded_by_next_candidate_prefix"
													if idx+1 == len(candidates) {
														markerConfidence = "final_body_to_player_span_end"
													}
													if marker, ok := parseObjectFinalSpanTailMarker(header, cursor, nextLimit, markerConfidence); ok {
														start := cursor
														regions = append(regions, replayRegion("inflated_header", fmt.Sprintf("initial_player_%d_object_final_span_tail_marker_%03d", playerIndex, idx), start, marker.End, "decoded", "matched_final_span_tail_marker_structure_semantics_unknown", map[string]any{
															"label":            player.Label,
															"object_id":        candidate.ObjectID,
															"record_type":      candidate.RecordType,
															"record_type_name": candidate.RecordTypeName,
															"bytes":            marker.End - start,
														}))
														cursor = marker.End
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
		previous = &candidates[idx]
	}
	if cursor < end {
		details := map[string]any{"label": player.Label}
		confidence := "after_last_plausible_object_prefix"
		if previous != nil {
			confidence = "final_body_after_decoded_object_prefix"
			details["object_id"] = previous.ObjectID
			details["record_type"] = previous.RecordType
			details["unit_id"] = previous.UnitID
			details["body_bytes_after_prefix"] = end - cursor
		}
		regions = append(regions, replayRegion("inflated_header", fmt.Sprintf("initial_player_%d_object_tail_after_candidate_prefixes", playerIndex), cursor, end, "bounded_opaque", confidence, details))
	}
	return regions
}

type initialPlayerAttributes struct {
	Start             int
	End               int
	CameraX           float64
	CameraY           float64
	PostCameraUnknown int32
	StartX            float64
	StartY            float64
	SpawnX            uint16
	SpawnY            uint16
	StartMetaByte     byte
	CivilizationKey   string
}

func parseInitialPlayerAttributes(header []byte, start int, limit int, numHeaderData uint32) (initialPlayerAttributes, error) {
	attrs := initialPlayerAttributes{Start: start}
	if numHeaderData < 198 || numHeaderData > 10000 {
		return attrs, fmt.Errorf("unreasonable num_header_data %d", numHeaderData)
	}
	statsBytes64 := uint64(numHeaderData) * 2 * 4
	if statsBytes64 > uint64(limit-start) {
		return attrs, fmt.Errorf("stats table needs %d bytes but payload has %d", statsBytes64, limit-start)
	}
	off := start + int(statsBytes64)
	if off+24 > limit || off+24 > len(header) {
		return attrs, fmt.Errorf("need post-stats attributes at %d before limit %d", off, limit)
	}
	off++ // padding byte after player_stats.
	attrs.CameraX = float64(math.Float32frombits(binary.LittleEndian.Uint32(header[off:])))
	off += 4
	attrs.CameraY = float64(math.Float32frombits(binary.LittleEndian.Uint32(header[off:])))
	off += 4
	attrs.PostCameraUnknown = int32(binary.LittleEndian.Uint32(header[off:]))
	off += 4
	tail, err := parseInitialPlayerAttributesTail(header, off, limit)
	if err != nil {
		// Current DE save_version 68 human-player records insert two float32
		// start coordinates before the legacy u16 spawn/meta/civ-key tail.
		if off+8 <= limit {
			startX := float64(math.Float32frombits(binary.LittleEndian.Uint32(header[off:])))
			startY := float64(math.Float32frombits(binary.LittleEndian.Uint32(header[off+4:])))
			if plausibleXY(startX, startY) {
				if shifted, shiftedErr := parseInitialPlayerAttributesTail(header, off+8, limit); shiftedErr == nil {
					attrs.StartX = startX
					attrs.StartY = startY
					tail = shifted
					err = nil
				}
			}
		}
	}
	if err != nil {
		return attrs, err
	}
	attrs.SpawnX = tail.SpawnX
	attrs.SpawnY = tail.SpawnY
	attrs.StartMetaByte = tail.StartMetaByte
	attrs.CivilizationKey = tail.CivilizationKey
	attrs.End = tail.End
	return attrs, nil
}

type initialPlayerAttributesTail struct {
	End             int
	SpawnX          uint16
	SpawnY          uint16
	StartMetaByte   byte
	CivilizationKey string
}

func parseInitialPlayerAttributesTail(header []byte, off int, limit int) (initialPlayerAttributesTail, error) {
	var tail initialPlayerAttributesTail
	if off+5 > limit || off+5 > len(header) {
		return tail, fmt.Errorf("need spawn/meta tail at %d", off)
	}
	tail.SpawnX = binary.LittleEndian.Uint16(header[off:])
	off += 2
	tail.SpawnY = binary.LittleEndian.Uint16(header[off:])
	off += 2
	tail.StartMetaByte = header[off]
	off++
	if off+4 > limit || off+4 > len(header) {
		return tail, fmt.Errorf("need civilization key at %d", off)
	}
	if header[off] != 0x60 || header[off+1] != 0x0a {
		return tail, fmt.Errorf("bad civilization key marker at %d", off)
	}
	off += 2
	length := int(int16(binary.LittleEndian.Uint16(header[off:])))
	off += 2
	if length < 0 || off+length+6 > limit || off+length+6 > len(header) {
		return tail, fmt.Errorf("bad civilization key length %d at %d", length, off-2)
	}
	tail.CivilizationKey = decodeCString(header[off : off+length])
	off += length
	for i := 0; i < 6; i++ {
		if header[off+i] != 0 {
			return tail, fmt.Errorf("expected zero padding after civilization key at %d", off+i)
		}
	}
	off += 6
	tail.End = off
	return tail, nil
}

func parseInitialPlayerPrefix(header []byte, start int, index int, numPlayers int, triggerStart int) (InitialPlayerSpan, int, int, int, error) {
	player := InitialPlayerSpan{Index: index, Label: initialPlayerLabel(index), Start: start}
	if start+2 > triggerStart || start+2 > len(header) {
		return player, 0, 0, 0, fmt.Errorf("need player type bytes at %d", start)
	}
	player.Type = header[start]
	player.Unknown = header[start+1]
	off := start + 2 + numPlayers + numPlayers*4 + 4 + 1
	if off+2 > triggerStart || off+2 > len(header) {
		return player, 0, 0, 0, fmt.Errorf("need player name length at %d", off)
	}
	nameLen := int(binary.LittleEndian.Uint16(header[off:]))
	if nameLen < 1 || nameLen > 4096 {
		return player, 0, 0, 0, fmt.Errorf("unreasonable player name length %d at %d", nameLen, off)
	}
	off += 2
	if off+nameLen > triggerStart || off+nameLen > len(header) {
		return player, 0, 0, 0, fmt.Errorf("player name length %d overruns boundary", nameLen)
	}
	player.Name = decodeCString(header[off : off+nameLen])
	off += nameLen
	if off+6 > triggerStart || off+6 > len(header) {
		return player, 0, 0, 0, fmt.Errorf("need num_header_data marker at %d", off)
	}
	if header[off] != 0x16 || header[off+5] != 0x21 {
		return player, 0, 0, 0, fmt.Errorf("bad num_header_data marker at %d", off)
	}
	player.NumHeaderData = binary.LittleEndian.Uint32(header[off+1:])
	prefixEnd := off + 6
	if player.NumHeaderData < 198 || player.NumHeaderData > 10000 {
		return player, 0, 0, 0, fmt.Errorf("unreasonable num_header_data %d at %d", player.NumHeaderData, off+1)
	}
	limit := triggerStart
	if limit > len(header) {
		limit = len(header)
	}
	objectRel := bytes.Index(header[prefixEnd:limit], initialObjectMarker)
	if objectRel < 0 {
		return player, prefixEnd, 0, 0, nil
	}
	objectStart := prefixEnd + objectRel
	objectEnd := objectStart + len(initialObjectMarker)
	player.ObjectMarkerStart = objectStart
	player.ObjectMarkerEnd = objectEnd
	return player, prefixEnd, objectStart, objectEnd, nil
}

func findNextInitialPlayerStart(header []byte, searchStart int, markerNum uint32, numPlayers int, save float64, limit int) (int, bool) {
	if searchStart < 0 || searchStart >= len(header) {
		return 0, false
	}
	if limit > len(header) {
		limit = len(header)
	}
	var marker [6]byte
	marker[0] = 0x16
	binary.LittleEndian.PutUint32(marker[1:], markerNum)
	marker[5] = 0x21
	rel := bytes.Index(header[searchStart:limit], marker[:])
	if rel < 0 {
		return 0, false
	}
	markerOff := searchStart + rel
	count := 0
	for {
		if markerOff < 2 || count > 4096 {
			return 0, false
		}
		if int(binary.LittleEndian.Uint16(header[markerOff-2:markerOff])) == count {
			break
		}
		markerOff--
		count++
	}
	diplomacyBytes := 9 * 4
	if save >= 61.5 {
		diplomacyBytes = numPlayers * 4
	}
	backtrack := 7 + numPlayers + diplomacyBytes
	start := markerOff - backtrack - 2
	if start < searchStart || start >= limit {
		return 0, false
	}
	return start, true
}

func findLastInitialPlayerEnd(header []byte, searchStart int, numPlayers int, triggerStart int) (int, bool) {
	if searchStart < 0 || searchStart >= len(header) {
		return 0, false
	}
	limit := triggerStart
	if limit > len(header) {
		limit = len(header)
	}
	read := header[searchStart:limit]
	for i, b := range read {
		if b != 63 || i < 11 {
			continue
		}
		if !bytes.Equal(read[i-11:i-3], []byte{0xff, 0xff, 0xff, 0xff, 0x00, 0x00, 0x00, 0x00}) {
			continue
		}
		version := math.Float32frombits(binary.LittleEndian.Uint32(read[i-3 : i+1]))
		if version <= 1.0 || version >= 2.0 {
			continue
		}
		marker := i - 3
		backtrack := (1817 * (numPlayers - 1)) + 4 + 19
		end := searchStart + marker - backtrack - 2
		if end > searchStart && end <= limit {
			return end, true
		}
	}
	return 0, false
}

func initialPlayerLabel(index int) string {
	if index == 0 {
		return "Gaia"
	}
	return fmt.Sprintf("P%d", index)
}

func replayRegion(space, name string, start, end int, status, confidence string, details map[string]any) ReplayRegion {
	return ReplayRegion{
		Space:      space,
		Name:       name,
		Start:      start,
		End:        end,
		Bytes:      end - start,
		Status:     status,
		Confidence: confidence,
		Details:    sanitizeRegionDetails(details),
	}
}

func sanitizeRegionDetails(details map[string]any) map[string]any {
	if details == nil {
		return nil
	}
	out := make(map[string]any, len(details))
	for key, value := range details {
		out[key] = sanitizeRegionDetailValue(value)
	}
	return out
}

func sanitizeRegionDetailValue(value any) any {
	switch v := value.(type) {
	case float64:
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return nil
		}
		return v
	case float32:
		if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
			return nil
		}
		return v
	case map[string]any:
		return sanitizeRegionDetails(v)
	case []any:
		out := make([]any, len(v))
		for i := range v {
			out[i] = sanitizeRegionDetailValue(v[i])
		}
		return out
	default:
		return value
	}
}
