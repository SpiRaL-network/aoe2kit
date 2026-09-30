package replay

import (
	"encoding/hex"
	"fmt"
)

func extractCachedEvents(cache *ReplayCache, opts EventOptions) ([]ReplayEvent, map[int]string, EventCounts, []string, error) {
	var events []ReplayEvent
	names := map[int]string{}
	seenChat := map[string]bool{}
	counts := EventCounts{ActionIDs: map[int]int{}}
	bytesUsed := 0
	emit := func(ev ReplayEvent) error {
		bytesUsed += 1024 + len(ev.Text) + len(ev.RawHex) + len(ev.ObjectIDs)*8
		if bytesUsed > maxReportBytes {
			return fmt.Errorf("event result exceeds 32 MiB report budget; use streaming cached ops.jsonl or a narrower command")
		}
		events = append(events, ev)
		return nil
	}
	e := cache.WalkOps(func(op CachedOp) error {
		switch op.ID {
		case 1:
			counts.Actions++
			counts.ActionIDs[op.ActionID]++
			ev, ok := DecodeActionEvent(op.ActionID, op.Payload, op.TimeMS, op.Offset, opts.IncludeRaw)
			if ok {
				ev.Sequence = op.Sequence
				switch ev.Type {
				case "flare":
					counts.Flares++
					return emit(ev)
				case "resign":
					counts.Resigns++
					return emit(ev)
				default:
					if ev.X != 0 || ev.Y != 0 || ev.TargetID != 0 {
						counts.SpatialActions++
					}
					if opts.IncludeUntypedAction {
						return emit(ev)
					}
				}
			} else {
				counts.UntypedActions++
				name := actionName(op.ActionID)
				if name != "" {
					counts.UntypedActions--
				}
				if opts.IncludeUntypedAction {
					ev = ReplayEvent{Type: "action", TimeMS: op.TimeMS, Time: FormatTime(op.TimeMS), OperationID: int(op.ID), ReplayAction: true, ActionID: op.ActionID, ActionName: name, SourceOffset: op.Offset, PayloadBytes: len(op.Payload), Source: "action_stream", Confidence: "raw_preserved", Sequence: op.Sequence}
					if name != "" {
						ev.Confidence = "raw_preserved_named_action"
					}
					if opts.IncludeRaw {
						ev.RawHex = hex.EncodeToString(op.Payload)
					}
					return emit(ev)
				}
			}
		case 2:
			counts.Syncs++
			counts.DurationMS = op.TimeMS
			if opts.IncludeSystemEvents {
				return emit(ReplayEvent{Type: "sync", TimeMS: op.TimeMS, Time: FormatTime(op.TimeMS), OperationID: 2, SourceOffset: op.Offset, Source: "action_stream", Confidence: "parsed"})
			}
		case 3:
			counts.Viewlocks++
			if opts.IncludeSystemEvents {
				return emit(ReplayEvent{Type: "viewlock", TimeMS: op.TimeMS, Time: FormatTime(op.TimeMS), OperationID: 3, SourceOffset: op.Offset, X: f32(op.Payload[:4]), Y: f32(op.Payload[4:8]), Source: "action_stream", Confidence: "camera_observed_no_player"})
			}
		case 4:
			feedback, ok, name := decodeActionChat(op.Payload)
			if !ok {
				return nil
			}
			key := fmt.Sprintf("%d\x00%d\x00%s\x00%d", feedback.PlayerID, feedback.Channel, feedback.Text, op.TimeMS)
			if seenChat[key] {
				return nil
			}
			seenChat[key] = true
			if len(seenChat) > 10000 {
				return fmt.Errorf("chat dedupe exceeds 10000 row budget")
			}
			ev := eventFromFeedback(feedback, "chat", op.TimeMS, op.Offset, "action_stream", "parsed")
			ev.Telemetry = ParseTelemetry(ev.Text, opts.TelemetryPrefixes)
			if opts.IncludeRaw {
				ev.RawHex = hex.EncodeToString(op.Payload)
			}
			if name != "" {
				names[ev.PlayerID] = name
			}
			counts.Chat++
			if ev.TauntNumber > 0 {
				counts.Taunts++
			}
			if ev.Telemetry != nil {
				counts.Telemetry++
			}
			return emit(ev)
		case 5:
			kind, confidence := "start", "parsed"
			if op.Terminal {
				counts.EmbeddedTails++
				kind = "embedded_tail"
				confidence = "raw_preserved"
			} else {
				counts.Starts++
			}
			if opts.IncludeSystemEvents {
				payloadBytes := 0
				if op.Terminal {
					payloadBytes = op.SkippedBytes
				}
				return emit(ReplayEvent{Type: kind, TimeMS: op.TimeMS, Time: FormatTime(op.TimeMS), OperationID: 5, SourceOffset: op.Offset, PayloadBytes: payloadBytes, Source: "action_stream", Confidence: confidence})
			}
		case 6:
			counts.Ends++
			counts.Postgames++
			if opts.IncludeSystemEvents {
				return emit(ReplayEvent{Type: "postgame", TimeMS: op.TimeMS, Time: FormatTime(op.TimeMS), OperationID: 6, SourceOffset: op.Offset, Source: "action_stream", Confidence: "parsed"})
			}
		default:
			counts.Saves++
			if opts.IncludeSystemEvents {
				return emit(ReplayEvent{Type: "save", TimeMS: op.TimeMS, Time: FormatTime(op.TimeMS), OperationID: int(op.ID), SourceOffset: op.Offset, Source: "action_stream", Confidence: "skipped"})
			}
		}
		return nil
	})
	return finishReplayEvents(events, counts), names, counts, append([]string(nil), cache.Manifest.Warnings...), e
}
