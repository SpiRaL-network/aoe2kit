package replay

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

type FeedbackReport struct {
	Path      string                     `json:"path,omitempty"`
	Method    string                     `json:"method"`
	Players   []FeedbackPlayer           `json:"players,omitempty"`
	Timeline  []FeedbackEvent            `json:"timeline"`
	PerPlayer map[string][]FeedbackEvent `json:"per_player"`
	Taunts    []TauntCount               `json:"taunts,omitempty"`
	Counts    FeedbackCounts             `json:"counts"`
	Warnings  []string                   `json:"warnings,omitempty"`
}

type FeedbackPlayer struct {
	PlayerID int    `json:"player_id"`
	Name     string `json:"name,omitempty"`
	Human    bool   `json:"human"`
	Kind     string `json:"kind,omitempty"`
}

type FeedbackEvent struct {
	Index          int     `json:"index"`
	Kind           string  `json:"kind"`
	TimeMS         int     `json:"time_ms,omitempty"`
	Time           string  `json:"time,omitempty"`
	Phase          string  `json:"phase,omitempty"`
	PlayerID       int     `json:"player_id,omitempty"`
	PlayerName     string  `json:"player_name,omitempty"`
	Channel        int     `json:"channel,omitempty"`
	ChannelName    string  `json:"channel_name,omitempty"`
	Text           string  `json:"text,omitempty"`
	Raw            string  `json:"raw,omitempty"`
	TauntNumber    int     `json:"taunt_number,omitempty"`
	TauntText      string  `json:"taunt_text,omitempty"`
	DestinationMap int     `json:"destination_map,omitempty"`
	MessageAGP     string  `json:"message_agp,omitempty"`
	X              float64 `json:"x,omitempty"`
	Y              float64 `json:"y,omitempty"`
	Targets        []int   `json:"targets,omitempty"`
}

type TauntCount struct {
	Number int    `json:"number"`
	Count  int    `json:"count"`
	Text   string `json:"text,omitempty"`
}

type FeedbackCounts struct {
	Chat   int `json:"chat"`
	Taunts int `json:"taunts"`
	Flares int `json:"flares"`
}

type chatWire struct {
	Player         *int   `json:"player"`
	Channel        *int   `json:"channel"`
	Message        string `json:"message"`
	TauntNumber    *int   `json:"tauntNumber"`
	DestinationMap *int   `json:"destinationMap"`
	MessageAGP     string `json:"messageAGP"`
}

func ExtractFeedback(path string) (*FeedbackReport, error) {
	eventReport, err := ExtractEvents(path, EventOptions{})
	if err != nil {
		return nil, err
	}
	events := eventsToFeedback(eventReport.Events)
	players := eventReport.Players
	taunts := countTaunts(events)
	nameByID := map[int]string{}
	for _, player := range players {
		nameByID[player.PlayerID] = player.Name
	}
	perPlayer := map[string][]FeedbackEvent{}
	for _, event := range events {
		key := "unknown"
		if event.PlayerID > 0 {
			name := nameByID[event.PlayerID]
			if name == "" {
				name = "Unknown"
			}
			key = fmt.Sprintf("P%d %s", event.PlayerID, name)
		}
		perPlayer[key] = append(perPlayer[key], event)
	}
	if perPlayer == nil {
		perPlayer = map[string][]FeedbackEvent{}
	}
	report := &FeedbackReport{
		Path:      path,
		Method:    eventReport.Method,
		Players:   players,
		Timeline:  events,
		PerPlayer: perPlayer,
		Taunts:    taunts,
		Counts: FeedbackCounts{
			Chat:   countKind(events, "CHAT"),
			Taunts: len(taunts),
			Flares: countKind(events, "FLARE"),
		},
		Warnings: eventReport.Warnings,
	}
	return report, nil
}

func ExtractActionStreamFeedback(body []byte) ([]FeedbackEvent, map[int]string, []string) {
	reader := bytes.NewReader(body)
	var warnings []string
	if err := readReplayMeta(reader); err != nil {
		return nil, nil, []string{"action-stream meta parse failed: " + err.Error()}
	}
	var events []FeedbackEvent
	names := map[int]string{}
	seenChat := map[string]bool{}
	timeMS := 0
	for {
		opOffset := len(body) - reader.Len()
		op, err := readU32(reader)
		if err != nil {
			if err != io.EOF {
				warnings = append(warnings, "action-stream parse stopped: "+err.Error())
			}
			break
		}
		switch op {
		case 1:
			actionID, payload, _, err := readReplayAction(reader)
			if err != nil {
				warnings = append(warnings, "action parse stopped: "+err.Error())
				return events, names, warnings
			}
			if actionID == 115 {
				event, ok := decodeFlarePayload(payload)
				if ok {
					event.Index = len(events) + 1
					event.Kind = "FLARE"
					event.TimeMS = timeMS
					event.Time = FormatTime(timeMS)
					events = append(events, event)
				}
			}
		case 2:
			delta, err := readReplaySync(reader)
			if err != nil {
				warnings = append(warnings, "sync parse stopped: "+err.Error())
				return events, names, warnings
			}
			timeMS += int(delta)
		case 3:
			if err := skipN(reader, 12); err != nil {
				warnings = append(warnings, "viewlock parse stopped: "+err.Error())
				return events, names, warnings
			}
		case 4:
			if err := skipN(reader, 4); err != nil {
				warnings = append(warnings, "chat prefix parse stopped: "+err.Error())
				return events, names, warnings
			}
			length, err := readU32(reader)
			if err != nil {
				warnings = append(warnings, "chat length parse stopped: "+err.Error())
				return events, names, warnings
			}
			if length > uint32(reader.Len()) {
				warnings = append(warnings, fmt.Sprintf("chat length %d exceeds remaining body %d", length, reader.Len()))
				return events, names, warnings
			}
			raw := make([]byte, int(length))
			if _, err := io.ReadFull(reader, raw); err != nil {
				warnings = append(warnings, "chat payload parse stopped: "+err.Error())
				return events, names, warnings
			}
			event, ok, name := decodeActionChat(raw)
			if !ok {
				continue
			}
			key := fmt.Sprintf("%d\x00%d\x00%s\x00%d", event.PlayerID, event.Channel, event.Text, timeMS)
			if seenChat[key] {
				continue
			}
			seenChat[key] = true
			event.Index = len(events) + 1
			event.Kind = "CHAT"
			event.TimeMS = timeMS
			event.Time = FormatTime(timeMS)
			if name != "" {
				names[event.PlayerID] = name
			}
			events = append(events, event)
		case 5:
			result, err := skipStart(reader)
			if err != nil {
				warnings = append(warnings, "start op parse stopped: "+err.Error())
				return events, names, warnings
			}
			if result.Terminal {
				return events, names, warnings
			}
		case 6:
			return events, names, warnings
		default:
			if err := skipReplaySaveChapter(reader, len(body)); err != nil {
				warnings = append(warnings, fmt.Sprintf("unknown operation id %d at body offset %d", op, opOffset))
				warnings = append(warnings, fmt.Sprintf("save chapter skip failed: %v", err))
				return events, names, warnings
			}
		}
	}
	return events, names, warnings
}

func ExtractFeedbackEvents(body []byte) ([]FeedbackEvent, map[int]string, []string) {
	var events []FeedbackEvent
	names := map[int]string{}
	var warnings []string
	seen := map[string]bool{}
	searchFrom := 0

	for {
		rel := bytes.Index(body[searchFrom:], []byte(`{"player"`))
		if rel < 0 {
			break
		}
		start := searchFrom + rel
		end := balancedJSONObjectEnd(body, start)
		if end < 0 {
			searchFrom = start + 1
			continue
		}

		var wire chatWire
		raw := body[start : end+1]
		if err := json.Unmarshal(raw, &wire); err != nil {
			searchFrom = start + 1
			continue
		}
		if wire.Player == nil {
			searchFrom = end + 1
			continue
		}

		channel := 0
		if wire.Channel != nil {
			channel = *wire.Channel
		}
		taunt := 0
		if wire.TauntNumber != nil {
			taunt = *wire.TauntNumber
		}
		dest := 0
		if wire.DestinationMap != nil {
			dest = *wire.DestinationMap
		}
		msg := strings.TrimSpace(wire.Message)
		if msg == "" && taunt <= 0 {
			searchFrom = end + 1
			continue
		}

		key := fmt.Sprintf("%d\x00%d\x00%s\x00%d", *wire.Player, channel, msg, taunt)
		if !seen[key] {
			seen[key] = true
			event := FeedbackEvent{
				Index:          len(events) + 1,
				Kind:           "CHAT",
				PlayerID:       *wire.Player,
				Channel:        channel,
				ChannelName:    ChannelName(channel),
				Text:           msg,
				Raw:            string(raw),
				DestinationMap: dest,
			}
			if taunt > 0 {
				event.TauntNumber = taunt
				event.TauntText = TauntText(taunt)
			}
			if wire.MessageAGP != "" {
				event.MessageAGP = strings.TrimSpace(wire.MessageAGP)
				if name := nameFromMessageAGP(wire.MessageAGP); name != "" {
					names[*wire.Player] = name
				}
			}
			events = append(events, event)
		}
		searchFrom = end + 1
	}

	if len(events) == 0 && len(body) > 0 {
		warnings = append(warnings, "no chat JSON objects found in replay body")
	}
	return events, names, warnings
}

func ChannelName(channel int) string {
	switch channel {
	case 0:
		return "all"
	case 1:
		return "team"
	default:
		return "channel " + strconv.Itoa(channel)
	}
}

func FormatTime(ms int) string {
	seconds := ms / 1000
	millis := ms % 1000
	minutes := seconds / 60
	sec := seconds % 60
	hours := minutes / 60
	minute := minutes % 60
	return fmt.Sprintf("%02d:%02d:%02d.%03d", hours, minute, sec, millis)
}

func decodeActionChat(raw []byte) (FeedbackEvent, bool, string) {
	trimmed := bytes.Trim(raw, "\x00 \r\n\t")
	if len(trimmed) == 0 {
		return FeedbackEvent{}, false, ""
	}
	text := string(trimmed)
	var wire chatWire
	if err := json.Unmarshal(trimmed, &wire); err == nil && wire.Player != nil {
		channel := 0
		if wire.Channel != nil {
			channel = *wire.Channel
		}
		taunt := 0
		if wire.TauntNumber != nil {
			taunt = *wire.TauntNumber
		}
		dest := 0
		if wire.DestinationMap != nil {
			dest = *wire.DestinationMap
		}
		event := FeedbackEvent{
			PlayerID:       *wire.Player,
			Channel:        channel,
			ChannelName:    ChannelName(channel),
			Text:           strings.TrimSpace(wire.Message),
			Raw:            text,
			DestinationMap: dest,
		}
		if taunt > 0 {
			event.TauntNumber = taunt
			event.TauntText = TauntText(taunt)
		}
		name := ""
		if wire.MessageAGP != "" {
			event.MessageAGP = strings.TrimSpace(wire.MessageAGP)
			name = nameFromMessageAGP(wire.MessageAGP)
		}
		if event.Text == "" && event.TauntNumber <= 0 {
			return FeedbackEvent{}, false, ""
		}
		return event, true, name
	}
	return FeedbackEvent{Text: text, Raw: text, ChannelName: "unknown"}, true, ""
}

func decodeFlarePayload(data []byte) (FeedbackEvent, bool) {
	if _, body, ok := actionEnvelope(data); ok {
		data = body
	}
	if len(data) < 82 {
		return decodeShortFlarePayload(data)
	}
	x := math.Float32frombits(binary.LittleEndian.Uint32(data[4:8]))
	y := math.Float32frombits(binary.LittleEndian.Uint32(data[44:48]))
	numTargets := int(int8(data[81]))
	targets := []int{}
	if numTargets > 0 && 82+numTargets <= len(data) {
		targets = make([]int, 0, numTargets)
		for _, target := range data[82 : 82+numTargets] {
			targets = append(targets, int(int8(target)))
		}
	}
	return FeedbackEvent{X: float64(x), Y: float64(y), Targets: targets}, true
}

func decodeShortFlarePayload(data []byte) (FeedbackEvent, bool) {
	if len(data) < 16 {
		return FeedbackEvent{}, false
	}
	targetSlots := int(data[12])
	if targetSlots <= 0 || targetSlots > 16 || 13+targetSlots != len(data) {
		return FeedbackEvent{}, false
	}
	x := math.Float32frombits(binary.LittleEndian.Uint32(data[4:8]))
	y := math.Float32frombits(binary.LittleEndian.Uint32(data[8:12]))
	if !plausibleXY(float64(x), float64(y)) {
		return FeedbackEvent{}, false
	}
	targets := []int{}
	for slot, flag := range data[13 : 13+targetSlots] {
		if flag != 0 {
			targets = append(targets, slot)
		}
	}
	return FeedbackEvent{X: float64(x), Y: float64(y), Targets: targets}, true
}

type replayBodyReader interface {
	io.Reader
	io.ReaderAt
	io.Seeker
	io.ByteReader
	Len() int
	Size() int64
}

func readReplayAction(reader replayBodyReader) (int, []byte, int, error) {
	length, err := readU32(reader)
	if err != nil {
		return 0, nil, 0, err
	}
	if length == 0 {
		return 0, nil, 0, fmt.Errorf("zero-length action")
	}
	if length > uint32(reader.Len()) {
		return 0, nil, 0, fmt.Errorf("action length %d exceeds remaining body %d", length, reader.Len())
	}
	actionID, err := reader.ReadByte()
	if err != nil {
		return 0, nil, 0, err
	}
	payload := make([]byte, int(length)-1)
	if _, err := io.ReadFull(reader, payload); err != nil {
		return 0, nil, 0, err
	}
	sequence, err := readU32(reader)
	if err != nil {
		return 0, nil, 0, err
	}
	return int(actionID), payload, int(sequence), nil
}

func skipReplaySaveChapter(reader replayBodyReader, bodyLength int) error {
	if _, err := reader.Seek(-4, io.SeekCurrent); err != nil {
		return err
	}
	pos := bodyLength - reader.Len()
	length, err := readU32(reader)
	if err != nil {
		return err
	}
	if _, err := readU32(reader); err != nil {
		return err
	}
	target := int(length)
	if target < pos+8 {
		return fmt.Errorf("save chapter target %d precedes current payload start %d", target, pos+8)
	}
	return skipN(reader, target-pos-8)
}

func readReplaySync(reader replayBodyReader) (uint32, error) {
	increment, err := readU32(reader)
	if err != nil {
		return 0, err
	}
	marker, err := readU32(reader)
	if err != nil {
		return 0, err
	}
	if marker != 0 {
		if _, err := reader.Seek(-4, io.SeekCurrent); err != nil {
			return 0, err
		}
		return increment, nil
	}
	tailStart := reader.Len()
	var probe [16]byte
	if _, err := io.ReadFull(reader, probe[:]); err != nil {
		return 0, err
	}
	isDE := binary.LittleEndian.Uint32(probe[12:16])
	if isDE == 0 {
		if err := skipN(reader, 8); err != nil {
			return 0, err
		}
		return increment, nil
	}
	if _, err := reader.Seek(int64(reader.Len()-tailStart), io.SeekCurrent); err != nil {
		return 0, err
	}
	if err := skipN(reader, 8*11*4); err != nil {
		return 0, err
	}
	if _, err := readU32(reader); err != nil {
		return 0, err
	}
	return increment, nil
}

func readReplayMeta(reader replayBodyReader) error {
	first, err := readU32(reader)
	if err != nil {
		return err
	}
	if first != 500 {
		if err := skipN(reader, 4); err != nil {
			return err
		}
	}
	if err := skipN(reader, 20); err != nil {
		return err
	}
	posBefore := reader.Len()
	a, err := readU32(reader)
	if err != nil {
		return err
	}
	b, err := readU32(reader)
	if err != nil {
		return err
	}
	if _, err := readU32(reader); err != nil {
		return err
	}
	if a != 0 {
		if _, err := reader.Seek(-12, io.SeekCurrent); err != nil {
			return err
		}
	}
	if b == 2 {
		consumed := posBefore - reader.Len()
		if consumed < 8 {
			return fmt.Errorf("cannot rewind DE meta tail after consuming %d bytes", consumed)
		}
		if _, err := reader.Seek(-8, io.SeekCurrent); err != nil {
			return err
		}
	}
	return nil
}

type startSkipResult struct {
	Terminal     bool
	SkippedBytes int
}

func skipStart(reader replayBodyReader) (startSkipResult, error) {
	for _, skip := range []int{16, 20, 24, 28, 32} {
		if plausibleNextOperation(reader, skip) {
			return startSkipResult{SkippedBytes: skip}, skipN(reader, skip)
		}
	}
	remaining := reader.Len()
	return startSkipResult{Terminal: true, SkippedBytes: remaining}, skipN(reader, remaining)
}

func plausibleNextOperation(reader replayBodyReader, skip int) bool {
	if skip < 0 || skip+4 > reader.Len() {
		return false
	}
	op, ok := peekU32(reader, skip)
	if !ok {
		return false
	}
	remainingFromOp := reader.Len() - skip
	switch op {
	case 1:
		length, ok := peekU32(reader, skip+4)
		if !ok || length == 0 {
			return false
		}
		return int(length)+12 <= remainingFromOp
	case 2:
		return remainingFromOp >= 12
	case 3:
		if remainingFromOp < 16 {
			return false
		}
		xBits, okX := peekU32(reader, skip+4)
		yBits, okY := peekU32(reader, skip+8)
		if !okX || !okY {
			return false
		}
		return plausibleXY(float64(math.Float32frombits(xBits)), float64(math.Float32frombits(yBits)))
	case 4:
		length, ok := peekU32(reader, skip+8)
		if !ok {
			return false
		}
		return int(length)+12 <= remainingFromOp
	case 5:
		return remainingFromOp >= 20
	case 6:
		return true
	default:
		return false
	}
}

func peekU32(reader replayBodyReader, rel int) (uint32, bool) {
	if rel < 0 || rel+4 > reader.Len() {
		return 0, false
	}
	var buf [4]byte
	pos := reader.Size() - int64(reader.Len()) + int64(rel)
	if _, err := reader.ReadAt(buf[:], pos); err != nil {
		return 0, false
	}
	return binary.LittleEndian.Uint32(buf[:]), true
}

func readU32(reader replayBodyReader) (uint32, error) {
	var buf [4]byte
	if _, err := io.ReadFull(reader, buf[:]); err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint32(buf[:]), nil
}

func skipN(reader replayBodyReader, n int) error {
	if n < 0 {
		return fmt.Errorf("negative skip %d", n)
	}
	if n > reader.Len() {
		return io.ErrUnexpectedEOF
	}
	_, err := reader.Seek(int64(n), io.SeekCurrent)
	return err
}

func countKind(events []FeedbackEvent, kind string) int {
	count := 0
	for _, event := range events {
		if event.Kind == kind {
			count++
		}
	}
	return count
}

func balancedJSONObjectEnd(data []byte, start int) int {
	depth := 0
	inString := false
	escaped := false
	for i := start; i < len(data); i++ {
		b := data[i]
		if inString {
			if escaped {
				escaped = false
				continue
			}
			switch b {
			case '\\':
				escaped = true
			case '"':
				inString = false
			}
			continue
		}
		switch b {
		case '"':
			inString = true
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return i
			}
			if depth < 0 {
				return -1
			}
		}
	}
	return -1
}

func nameFromMessageAGP(s string) string {
	re := regexp.MustCompile(`@#\d+\s+<[^>]+>\s+([^\r\n:]+)`)
	match := re.FindStringSubmatch(s)
	if len(match) < 2 {
		return ""
	}
	name := strings.TrimSpace(match[1])
	if plausiblePlayerName(name) {
		return name
	}
	return ""
}

func feedbackPlayers(slots []PlayerSlot, chatNames map[int]string, events []FeedbackEvent) []FeedbackPlayer {
	seenIDs := map[int]bool{}
	byID := map[int]FeedbackPlayer{}
	for _, slot := range slots {
		if slot.Number <= 0 {
			continue
		}
		seenIDs[slot.Number] = true
		byID[slot.Number] = FeedbackPlayer{
			PlayerID: slot.Number,
			Name:     slot.Name,
			Human:    slot.Human,
			Kind:     slot.Kind,
		}
	}
	for id, name := range chatNames {
		seenIDs[id] = true
		player := byID[id]
		player.PlayerID = id
		player.Name = name
		if player.Kind == "" {
			player.Kind = "seen"
		}
		byID[id] = player
	}
	for _, event := range events {
		if event.PlayerID <= 0 {
			continue
		}
		seenIDs[event.PlayerID] = true
		if _, ok := byID[event.PlayerID]; !ok {
			byID[event.PlayerID] = FeedbackPlayer{PlayerID: event.PlayerID, Name: "Unknown", Kind: "seen"}
		}
	}
	ids := make([]int, 0, len(seenIDs))
	for id := range seenIDs {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	out := make([]FeedbackPlayer, 0, len(ids))
	for _, id := range ids {
		out = append(out, byID[id])
	}
	return out
}

func countTaunts(events []FeedbackEvent) []TauntCount {
	counts := map[int]int{}
	for _, event := range events {
		if event.TauntNumber > 0 {
			counts[event.TauntNumber]++
		}
	}
	nums := make([]int, 0, len(counts))
	for num := range counts {
		nums = append(nums, num)
	}
	sort.Ints(nums)
	out := make([]TauntCount, 0, len(nums))
	for _, num := range nums {
		out = append(out, TauntCount{Number: num, Count: counts[num], Text: TauntText(num)})
	}
	return out
}

func plausiblePlayerName(s string) bool {
	if len(s) < 2 || len(s) > 32 {
		return false
	}
	if strings.ContainsAny(s, `{}[]<>\/=;`) {
		return false
	}
	lower := strings.ToLower(s)
	badParts := []string{
		"scenario", ".aoe2", "random map", "empire", "dataset", "trigger",
		"message", "player", "difficulty", "resources", "common", "unknown",
	}
	for _, bad := range badParts {
		if strings.Contains(lower, bad) {
			return false
		}
	}
	hasLetter := false
	for _, r := range s {
		if unicode.IsLetter(r) {
			hasLetter = true
			continue
		}
		if unicode.IsDigit(r) || unicode.IsSpace(r) || strings.ContainsRune("_-'().", r) {
			continue
		}
		return false
	}
	return hasLetter
}

func ExtractPrintableStrings(data []byte, minLen int) []string {
	var out []string
	out = append(out, extractUTF8Strings(data, minLen)...)
	out = append(out, extractASCIIStrings(data, minLen)...)
	return out
}

func extractUTF8Strings(data []byte, minRunes int) []string {
	var out []string
	for i := 0; i < len(data); {
		start := i
		var b strings.Builder
		runeCount := 0
		for i < len(data) {
			r, size := utf8.DecodeRune(data[i:])
			if r == utf8.RuneError && size == 1 {
				break
			}
			if !printableTextRune(r) {
				break
			}
			b.WriteRune(r)
			runeCount++
			i += size
		}
		if runeCount >= minRunes {
			out = append(out, b.String())
		}
		if i == start {
			i++
		}
	}
	return out
}

func extractASCIIStrings(data []byte, minLen int) []string {
	var out []string
	for i := 0; i < len(data); {
		for i < len(data) && !printableASCII(data[i]) {
			i++
		}
		start := i
		for i < len(data) && printableASCII(data[i]) {
			i++
		}
		if i-start >= minLen {
			out = append(out, string(data[start:i]))
		}
	}
	return out
}

func printableASCII(b byte) bool {
	return b >= 32 && b <= 126
}

func printableTextRune(r rune) bool {
	return r >= 32 && r != 127 && !unicode.IsControl(r)
}
