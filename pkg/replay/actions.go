package replay

import (
	"encoding/binary"
	"encoding/hex"
	"math"
)

const (
	actionOrder              = 0
	actionStop               = 1
	actionMove               = 3
	actionAIOrder            = 10
	actionResign             = 11
	actionStance             = 18
	actionPatrol             = 21
	actionFormation          = 23
	actionAttackMove         = 33
	actionDEMultiGatherPoint = 45
	actionMake               = 100
	actionResearch           = 101
	actionBuild              = 102
	actionGame               = 103
	actionWall               = 105
	actionDelete             = 106
	actionAttackGround       = 107
	actionRepair             = 110
	actionUngarrison         = 111
	actionSpecial            = 117
	actionGatherPoint        = 120
	actionBuy                = 122
	actionSell               = 123
	actionTownBell           = 127
	actionBackToWork         = 128
	actionDEQueue            = 129
	actionGate               = 114
	actionFlare              = 115
	actionPostgame           = 255
	// Named from the aoc-mgz v68 reference table (payloads stay raw unless decoded below).
	actionGuard        = 19
	actionFollow       = 20
	actionDERetreat    = 35
	actionDEAutoscout  = 38
	actionDETransform  = 41
	actionRathaAbility = 43
	actionDE107A       = 44
	actionAICommand    = 53
	actionDE107B       = 140
	actionDETribute    = 196
)

func DecodeActionEvent(actionID int, payload []byte, timeMS int, offset int, includeRaw bool) (ReplayEvent, bool) {
	playerID, body, ok := actionEnvelope(payload)
	if !ok && len(payload) > 0 {
		playerID = int(int8(payload[0]))
		body = payload
	}
	event := ReplayEvent{
		Type:         "action",
		TimeMS:       timeMS,
		Time:         FormatTime(timeMS),
		PlayerID:     playerID,
		OperationID:  1,
		ReplayAction: true,
		ActionID:     actionID,
		ActionName:   actionName(actionID),
		SourceOffset: offset,
		PayloadBytes: len(payload),
		Source:       "action_stream",
		Confidence:   "parsed",
	}
	if includeRaw {
		event.RawHex = hex.EncodeToString(payload)
	}
	switch actionID {
	case actionResign:
		event.Type = "resign"
		return event, playerID > 0
	case actionMove:
		if len(body) < 20 {
			return ReplayEvent{}, false
		}
		selected := int(int16(binary.LittleEndian.Uint16(body[12:14])))
		event.Type = "move"
		event.X = f32(body[4:8])
		event.Y = f32(body[8:12])
		event.ObjectIDs = readObjectIDs(body[20:], selected)
		return event, plausibleXY(event.X, event.Y)
	case actionOrder:
		if len(body) < 20 {
			return ReplayEvent{}, false
		}
		selected := int(int16(binary.LittleEndian.Uint16(body[12:14])))
		event.Type = "order"
		event.TargetID = int(binary.LittleEndian.Uint32(body[0:4]))
		event.X = f32(body[4:8])
		event.Y = f32(body[8:12])
		event.ObjectIDs = readObjectIDs(body[20:], selected)
		return event, event.TargetID > 0 || plausibleXY(event.X, event.Y)
	case actionPatrol:
		if len(body) < 52 {
			return ReplayEvent{}, false
		}
		selected := int(binary.LittleEndian.Uint32(body[0:4]))
		event.Type = "patrol"
		event.X = f32(body[8:12])
		event.Y = f32(body[48:52])
		if len(body) > 88 {
			event.ObjectIDs = readObjectIDs(body[88:], selected)
		}
		return event, plausibleXY(event.X, event.Y)
	case actionAttackMove:
		if len(body) < 80 {
			return ReplayEvent{}, false
		}
		selected := int(binary.LittleEndian.Uint32(body[0:4]))
		event.Type = "attack_move"
		event.X = f32(body[8:12])
		event.Y = f32(body[48:52])
		if len(body) > 80 {
			event.ObjectIDs = readObjectIDs(body[80:], selected)
		}
		return event, plausibleXY(event.X, event.Y)
	case actionAttackGround:
		if len(body) < 12 {
			return ReplayEvent{}, false
		}
		selected := int(binary.LittleEndian.Uint32(body[0:4]))
		event.Type = "attack_ground"
		event.X = f32(body[4:8])
		event.Y = f32(body[8:12])
		if len(body) > 16 {
			event.ObjectIDs = readObjectIDs(body[16:], selected)
		}
		return event, plausibleXY(event.X, event.Y)
	case actionResearch:
		if len(body) < 13 {
			return ReplayEvent{}, false
		}
		selected := int(int16(binary.LittleEndian.Uint16(body[4:6])))
		event.Type = "research"
		event.ObjectIDs = []int{int(binary.LittleEndian.Uint32(body[0:4]))}
		event.TechnologyID = int(int16(binary.LittleEndian.Uint16(body[6:8])))
		if len(body) >= 13+selected*4 {
			event.ObjectIDs = append(event.ObjectIDs, readObjectIDs(body[13:], selected)...)
		}
		return event, event.TechnologyID > 0
	case actionDEQueue:
		if len(body) < 18 {
			return ReplayEvent{}, false
		}
		selected := int(int16(binary.LittleEndian.Uint16(body[0:2])))
		event.Type = "de_queue"
		event.BuildingID = int(int16(binary.LittleEndian.Uint16(body[6:8])))
		event.UnitID = int(int16(binary.LittleEndian.Uint16(body[8:10])))
		event.Amount = int(int16(binary.LittleEndian.Uint16(body[10:12])))
		event.ObjectIDs = readObjectIDs(body[18:], selected)
		return event, event.UnitID > 0 || event.BuildingID > 0
	case actionBuild:
		if len(body) < 18 {
			return ReplayEvent{}, false
		}
		selected := int(int16(binary.LittleEndian.Uint16(body[0:2])))
		event.Type = "build"
		event.X = f32(body[4:8])
		event.Y = f32(body[8:12])
		event.TargetID = int(binary.LittleEndian.Uint32(body[12:16]))
		if len(body) > 28 {
			event.ObjectIDs = readObjectIDs(body[28:], selected)
		}
		return event, plausibleXY(event.X, event.Y)
	case actionGatherPoint:
		if len(body) < 18 {
			return ReplayEvent{}, false
		}
		selected := int(int16(binary.LittleEndian.Uint16(body[0:2])))
		event.Type = "gather_point"
		event.X = f32(body[4:8])
		event.Y = f32(body[8:12])
		event.TargetID = int(int32(binary.LittleEndian.Uint32(body[12:16])))
		if len(body) > 21 {
			event.ObjectIDs = readObjectIDs(body[21:], selected)
		}
		return event, event.TargetID > 0 || plausibleXY(event.X, event.Y)
	case actionWall:
		if len(body) < 20 {
			return ReplayEvent{}, false
		}
		selected := int(binary.LittleEndian.Uint32(body[0:4]))
		event.Type = "wall"
		event.X = float64(binary.LittleEndian.Uint16(body[4:6]))
		event.Y = float64(binary.LittleEndian.Uint16(body[6:8]))
		event.XEnd = float64(binary.LittleEndian.Uint16(body[8:10]))
		event.YEnd = float64(binary.LittleEndian.Uint16(body[10:12]))
		event.TargetID = int(binary.LittleEndian.Uint32(body[12:16]))
		if len(body) > 28 {
			event.ObjectIDs = readObjectIDs(body[28:], selected)
		}
		return event, plausibleXY(event.X, event.Y)
	case actionGame:
		if len(body) < 2 {
			return ReplayEvent{}, false
		}
		event.Type = "game_command"
		event.CommandID = int(int16(binary.LittleEndian.Uint16(body[0:2])))
		return event, true
	case actionStop:
		if len(body) < 4 {
			return ReplayEvent{}, false
		}
		selected := int(binary.LittleEndian.Uint32(body[0:4]))
		event.Type = "stop"
		event.ObjectIDs = readObjectIDs(body[4:], selected)
		return event, true
	case actionStance:
		if len(body) < 8 {
			return ReplayEvent{}, false
		}
		selected := int(binary.LittleEndian.Uint32(body[0:4]))
		event.Type = "stance"
		event.StanceID = int(binary.LittleEndian.Uint32(body[4:8]))
		event.ObjectIDs = readObjectIDs(body[8:], selected)
		return event, true
	case actionFormation:
		if len(body) < 8 {
			return ReplayEvent{}, false
		}
		selected := int(binary.LittleEndian.Uint32(body[0:4]))
		event.Type = "formation"
		event.FormationID = int(binary.LittleEndian.Uint32(body[4:8]))
		event.ObjectIDs = readObjectIDs(body[8:], selected)
		return event, true
	case actionDelete, actionGate:
		if len(body) < 4 {
			return ReplayEvent{}, false
		}
		if actionID == actionDelete {
			event.Type = "delete"
		} else {
			event.Type = "gate"
		}
		event.ObjectIDs = []int{int(binary.LittleEndian.Uint32(body[0:4]))}
		return event, true
	case actionBackToWork:
		if len(body) < 4 {
			return ReplayEvent{}, false
		}
		event.Type = "back_to_work"
		event.ObjectIDs = []int{int(binary.LittleEndian.Uint32(body[0:4]))}
		return event, true
	case actionTownBell:
		if len(body) < 5 {
			return ReplayEvent{}, false
		}
		event.Type = "town_bell"
		event.BuildingID = int(binary.LittleEndian.Uint32(body[0:4]))
		event.ModeID = int(int8(body[4]))
		return event, true
	case actionMake:
		if len(body) < 10 {
			return ReplayEvent{}, false
		}
		event.Type = "make"
		event.BuildingID = int(binary.LittleEndian.Uint16(body[0:2]))
		event.UnitID = int(int16(binary.LittleEndian.Uint16(body[8:10])))
		return event, event.UnitID > 0
	case actionDEMultiGatherPoint:
		if len(body) < 12 {
			return ReplayEvent{}, false
		}
		event.Type = "de_multi_gatherpoint"
		event.TargetID = int(int32(binary.LittleEndian.Uint32(body[0:4])))
		event.X = f32(body[4:8])
		event.Y = f32(body[8:12])
		return event, event.TargetID > 0 || plausibleXY(event.X, event.Y)
	case actionUngarrison:
		if len(body) < 20 {
			return ReplayEvent{}, false
		}
		selected := int(binary.LittleEndian.Uint32(body[0:4]))
		event.Type = "ungarrison"
		event.X = f32(body[4:8])
		event.Y = f32(body[8:12])
		event.TargetID = int(int32(binary.LittleEndian.Uint32(body[12:16])))
		if len(body) > 20 {
			event.ObjectIDs = readObjectIDs(body[20:], selected)
		}
		return event, event.TargetID > 0 || plausibleXY(event.X, event.Y)
	case actionSpecial:
		if len(body) < 28 {
			return ReplayEvent{}, false
		}
		selected := int(binary.LittleEndian.Uint32(body[0:4]))
		event.Type = "special"
		event.TargetID = int(int32(binary.LittleEndian.Uint32(body[4:8])))
		event.X = f32(body[8:12])
		event.Y = f32(body[12:16])
		event.SlotID = int(int16(binary.LittleEndian.Uint16(body[20:22])))
		event.OrderID = int(int16(binary.LittleEndian.Uint16(body[24:26])))
		if len(body) > 28 {
			event.ObjectIDs = readObjectIDs(body[28:], selected)
		}
		return event, event.TargetID > 0 || plausibleXY(event.X, event.Y) || event.OrderID > 0
	case actionAIOrder:
		if len(body) < 24 {
			return ReplayEvent{}, false
		}
		event.Type = "ai_order"
		event.ObjectIDs = []int{int(binary.LittleEndian.Uint32(body[4:8]))}
		event.X = f32(body[16:20])
		event.Y = f32(body[20:24])
		return event, plausibleXY(event.X, event.Y)
	case actionFlare:
		feedback, ok := decodeFlarePayload(payload)
		if ok {
			event.Type = "flare"
			event.X = feedback.X
			event.Y = feedback.Y
			event.Targets = feedback.Targets
			return event, plausibleXY(event.X, event.Y)
		}
	case actionPostgame:
		event.Type = "postgame_action"
		return event, true
	case actionRathaAbility:
		// Structure derived from corpus samples + length math, named by the mgz v68
		// reference: [count u32][mode u32][count x object id u32]. Mode observed as
		// adjacent small values (e.g. 153/154) consistent with the Ratha's two forms;
		// mode SEMANTICS (which value = melee vs ranged) are not claimed.
		if len(body) < 8 {
			return ReplayEvent{}, false
		}
		count := int(binary.LittleEndian.Uint32(body[0:4]))
		if count <= 0 || count > 512 || len(body) < 8+count*4 {
			return ReplayEvent{}, false
		}
		event.Type = "ratha_ability"
		event.ModeID = int(binary.LittleEndian.Uint32(body[4:8]))
		event.ObjectIDs = readObjectIDs(body[8:], count)
		return event, len(event.ObjectIDs) == count
	}
	return ReplayEvent{}, false
}

func actionEnvelope(payload []byte) (int, []byte, bool) {
	if len(payload) < 3 {
		return 0, payload, false
	}
	length := int(binary.LittleEndian.Uint16(payload[1:3]))
	if length+3 != len(payload) {
		return int(int8(payload[0])), payload, false
	}
	return int(int8(payload[0])), payload[3:], true
}

func actionName(actionID int) string {
	switch actionID {
	case actionOrder:
		return "ORDER"
	case actionMove:
		return "MOVE"
	case actionStop:
		return "STOP"
	case actionAIOrder:
		return "AI_ORDER"
	case actionResign:
		return "RESIGN"
	case actionStance:
		return "STANCE"
	case actionPatrol:
		return "PATROL"
	case actionFormation:
		return "FORMATION"
	case actionAttackMove:
		return "DE_ATTACK_MOVE"
	case actionDEMultiGatherPoint:
		return "DE_MULTI_GATHERPOINT"
	case actionMake:
		return "MAKE"
	case actionResearch:
		return "RESEARCH"
	case actionBuild:
		return "BUILD"
	case actionGame:
		return "GAME"
	case actionWall:
		return "WALL"
	case actionDelete:
		return "DELETE"
	case actionAttackGround:
		return "ATTACK_GROUND"
	case actionRepair:
		return "REPAIR"
	case actionUngarrison:
		return "UNGARRISON"
	case actionSpecial:
		return "SPECIAL"
	case actionGate:
		return "GATE"
	case actionFlare:
		return "FLARE"
	case actionGatherPoint:
		return "GATHER_POINT"
	case actionBuy:
		return "BUY"
	case actionSell:
		return "SELL"
	case actionTownBell:
		return "TOWN_BELL"
	case actionBackToWork:
		return "BACK_TO_WORK"
	case actionDEQueue:
		return "DE_QUEUE"
	case actionPostgame:
		return "POSTGAME"
	case actionGuard:
		return "GUARD"
	case actionFollow:
		return "FOLLOW"
	case actionDERetreat:
		return "DE_RETREAT"
	case actionDEAutoscout:
		return "DE_AUTOSCOUT"
	case actionDETransform:
		return "DE_TRANSFORM"
	case actionRathaAbility:
		return "RATHA_ABILITY"
	case actionDE107A:
		return "DE_107_A"
	case actionAICommand:
		return "AI_COMMAND"
	case actionDE107B:
		return "DE_107_B"
	case actionDETribute:
		return "DE_TRIBUTE"
	default:
		return ""
	}
}

func f32(data []byte) float64 {
	return float64(math.Float32frombits(binary.LittleEndian.Uint32(data)))
}

func plausibleXY(x, y float64) bool {
	return x >= -1024 && y >= -1024 && x <= 16384 && y <= 16384
}

func readObjectIDs(data []byte, count int) []int {
	if count <= 0 || count > 512 || len(data) < count*4 {
		return nil
	}
	out := make([]int, 0, count)
	for i := 0; i < count; i++ {
		out = append(out, int(binary.LittleEndian.Uint32(data[i*4:i*4+4])))
	}
	return out
}
