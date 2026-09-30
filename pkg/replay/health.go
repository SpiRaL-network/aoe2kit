package replay

import (
	"bufio"
	"compress/flate"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"
)

const healthRowLimit = 10000
const healthPayloadLimit = 1 << 20
const healthChatBytesLimit = 4 << 20

type HealthOptions struct{ Window time.Duration }

type HealthAnchor struct {
	LastSync       *int   `json:"last_sync_time_ms"`
	LastAction     *int   `json:"last_action_time_ms"`
	StatedDuration *int   `json:"stated_duration_ms"`
	LastChat       *int   `json:"last_chat_time_ms"`
	LastResign     *int   `json:"last_resign_time_ms"`
	ChatGap        *int   `json:"last_chat_to_sync_gap_ms"`
	ResignGap      *int   `json:"last_resign_to_sync_gap_ms"`
	EndsCleanly    bool   `json:"ends_cleanly"`
	Termination    string `json:"termination"`
	StopOffset     int64  `json:"body_stop_offset"`
	FileStopOffset int64  `json:"file_stop_offset"`
}

type HealthObjects struct {
	PlayerID    int    `json:"player_id"`
	PlayerName  string `json:"player_name,omitempty"`
	ObjectCount uint32 `json:"object_count"`
}

type HealthWindow struct {
	From          int             `json:"from_ms"`
	To            int             `json:"to_ms"`
	SyncCount     int             `json:"sync_count"`
	ActionCount   int             `json:"action_count"`
	ActionsPerMin float64         `json:"actions_per_min"`
	ChatCount     int             `json:"chat_count"`
	FlareCount    int             `json:"flare_count"`
	SampleTime    *int            `json:"object_sample_time_ms"`
	Objects       []HealthObjects `json:"players"`
	ObjectTotal   *uint64         `json:"object_count_total"`
}

type HealthLoad struct {
	PeakObjects     *uint64         `json:"peak_object_count_total"`
	PeakObjectsTime *int            `json:"peak_object_count_time_ms"`
	PeakActions     float64         `json:"peak_actions_per_min"`
	PeakActionsTime *int            `json:"peak_actions_window_from_ms"`
	FirstSlope      *float64        `json:"first_30m_objects_per_min"`
	LastSlope       *float64        `json:"last_30m_objects_per_min"`
	EndObjects      []HealthObjects `json:"end_players_by_object_count"`
	EndSampleTime   *int            `json:"end_object_sample_time_ms"`
}

type HealthReport struct {
	Schema             string            `json:"schema"`
	Path               string            `json:"path"`
	WindowMS           int               `json:"window_ms"`
	EndAnchor          HealthAnchor      `json:"end_anchor"`
	Windows            []HealthWindow    `json:"windows"`
	LoadSummary        HealthLoad        `json:"load_summary"`
	Signals            []ChatLine        `json:"signals"`
	BacklogExcluded    int               `json:"backlog_chat_excluded"`
	TriggerGraphOK     bool              `json:"trigger_graph_ok"`
	TriggerGraphStatus string            `json:"trigger_graph_status"`
	TriggersNearEnd    any               `json:"triggers_near_end"`
	WordSemantics      map[string]string `json:"checksum_word_semantics"`
	Warnings           []string          `json:"warnings"`
}

// A buffered section reader keeps body storage constant even for very long games.
// Seeking only discards the small read buffer, never materializes the body.
type healthReader struct {
	section *io.SectionReader
	buffer  *bufio.Reader
	pos     int64
}

func newHealthReader(r io.ReaderAt, offset, size int64) *healthReader {
	s := io.NewSectionReader(r, offset, size)
	return &healthReader{section: s, buffer: bufio.NewReaderSize(s, 64<<10)}
}
func (r *healthReader) Read(p []byte) (int, error) {
	n, e := r.buffer.Read(p)
	r.pos += int64(n)
	return n, e
}
func (r *healthReader) ReadByte() (byte, error) {
	b, e := r.buffer.ReadByte()
	if e == nil {
		r.pos++
	}
	return b, e
}
func (r *healthReader) ReadAt(p []byte, off int64) (int, error) { return r.section.ReadAt(p, off) }
func (r *healthReader) Size() int64                             { return r.section.Size() }
func (r *healthReader) Len() int                                { return int(r.Size() - r.pos) }
func (r *healthReader) Seek(off int64, whence int) (int64, error) {
	if whence == io.SeekCurrent {
		off += r.pos
		whence = io.SeekStart
	}
	p, e := r.section.Seek(off, whence)
	if e == nil {
		r.pos = p
		r.buffer.Reset(r.section)
	}
	return p, e
}

var healthSignalPattern = regexp.MustCompile(`(?i)\b(crash(?:ed|es|ing)?|lag(?:gy|ging)?|freez(?:e|es|ing)|frozen|desync|out\s+of\s+sync|dc|disconnect(?:ed|ing)?)\b`)

// BuildHealth deliberately avoids Parse: its eager header models and graph search
// are unnecessary for body triage. No sync/action event table is retained.
func BuildHealth(path string, opts HealthOptions) (*HealthReport, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	stat, err := f.Stat()
	if err != nil {
		return nil, err
	}
	var prefix [8]byte
	if _, err = io.ReadFull(f, prefix[:]); err != nil {
		return nil, err
	}
	headerLen := int64(binary.LittleEndian.Uint32(prefix[:4]))
	if headerLen < 8 || headerLen > stat.Size() {
		return nil, fmt.Errorf("invalid header length %d (health requires an unzipped recording)", headerLen)
	}
	// Validate a bounded header prefix without retaining it. Large legitimate
	// recorded snapshots can inflate past this limit; body framing is independent.
	z := flate.NewReader(io.NewSectionReader(f, 8, headerLen-8))
	n, err := io.Copy(io.Discard, io.LimitReader(z, (64<<20)+1))
	z.Close()
	if err != nil {
		return nil, fmt.Errorf("header inflate: %w", err)
	}
	r, err := buildHealthBody(newHealthReader(f, headerLen, stat.Size()-headerLen), opts)
	if err != nil {
		return nil, err
	}
	r.Path = path
	if n > 64<<20 {
		r.Warnings = append(r.Warnings, "header inflation validation stopped at 64 MiB; remaining snapshot skipped, body offset taken from file framing")
	}
	r.EndAnchor.FileStopOffset = headerLen + r.EndAnchor.StopOffset
	return r, nil
}

func buildHealthBody(reader replayBodyReader, opts HealthOptions) (*HealthReport, error) {
	if opts.Window == 0 {
		opts.Window = 5 * time.Minute
	}
	if opts.Window < time.Second || opts.Window > 24*time.Hour {
		return nil, fmt.Errorf("window must be between 1s and 24h")
	}
	width := int(opts.Window / time.Millisecond)
	r := &HealthReport{Schema: "aoe2kit.replay.health.v1", WindowMS: width,
		Signals: []ChatLine{}, Windows: []HealthWindow{}, WordSemantics: syncWordSemantics(),
		TriggerGraphStatus: "not_inspected_no_runtime_fire_channel",
		Warnings: []string{
			"Simulation time is not wall time: no client freeze duration or crash cause is inferred.",
			"ends_cleanly describes body framing only; boundary EOF is not proof of a clean game exit.",
			"Header stated duration unavailable in the current decoder; it is not substituted with observed duration.",
			"Trigger fire counts unavailable: embedded definitions do not establish runtime executions; graph inspection skipped.",
		}}
	now := 0
	names := map[int]string{}
	early := []ChatLine{}
	earlyFlushed := false
	earlyOverflow := false
	signalOverflow := false
	earlyBytes, signalBytes := 0, 0
	addChat := func(c ChatLine) {
		if c.Source == "backlog" {
			r.BacklogExcluded++
			return
		}
		t := c.TimeMS
		r.EndAnchor.LastChat = &t
		if healthSignalPattern.MatchString(strings.Join(strings.Fields(c.Text), " ")) {
			size := len(c.Text) + len(c.PlayerName)
			if len(r.Signals) < healthRowLimit && signalBytes+size <= healthChatBytesLimit {
				r.Signals = append(r.Signals, c)
				signalBytes += size
			} else if !signalOverflow {
				r.Warnings = append(r.Warnings, "signal cap (10000 rows / 4 MiB text) reached; matching chat may be omitted")
				signalOverflow = true
			}
		}
	}
	flushEarly := func() {
		if earlyFlushed {
			return
		}
		if !earlyOverflow {
			tagInitialChatBacklog(early)
			for _, c := range early {
				addChat(c)
			}
		} else {
			r.Warnings = append(r.Warnings, "initial chat exceeded 10000 rows / 4 MiB; early signals withheld because backlog classification is incomplete")
		}
		early = nil
		earlyFlushed = true
	}
	getWindow := func() (*HealthWindow, error) {
		index := now / width
		if index >= healthRowLimit {
			return nil, fmt.Errorf("window table exceeds 10000 rows; use a wider --window")
		}
		for len(r.Windows) <= index {
			i := len(r.Windows)
			r.Windows = append(r.Windows, HealthWindow{From: i * width, To: (i + 1) * width})
		}
		return &r.Windows[index], nil
	}
	stop := func(offset int, err error) {
		r.EndAnchor.StopOffset = int64(offset)
		r.EndAnchor.Termination = "malformed_or_truncated"
		r.Warnings = append(r.Warnings, fmt.Sprintf("body offset %d: %v", offset, err))
	}
	if err := readReplayMeta(reader); err != nil {
		stop(0, fmt.Errorf("meta: %w", err))
		return r, nil
	}
walk:
	for {
		offset := int(reader.Size()) - reader.Len()
		if reader.Len() == 0 {
			r.EndAnchor.EndsCleanly = true
			r.EndAnchor.Termination = "boundary_eof"
			r.EndAnchor.StopOffset = int64(offset)
			break
		}
		op, err := readU32(reader)
		if err != nil {
			stop(offset, err)
			break
		}
		if now > 120000 {
			flushEarly()
		}
		w, err := getWindow()
		if err != nil {
			return nil, err
		}
		switch op {
		case 1:
			length, ok := peekU32(reader, 0)
			if ok && length > healthPayloadLimit {
				stop(offset, fmt.Errorf("action payload exceeds 1 MiB limit"))
				break walk
			}
			id, payload, _, e := readReplayAction(reader)
			if e != nil {
				stop(offset, e)
				break walk
			}
			w.ActionCount++
			t := now
			r.EndAnchor.LastAction = &t
			if id == 11 || id == 115 {
				if ev, ok := DecodeActionEvent(id, payload, now, offset, false); ok {
					if ev.Type == "resign" {
						r.EndAnchor.LastResign = &t
					}
					if ev.Type == "flare" {
						w.FlareCount++
					}
				}
			}
		case 2:
			delta, e := readU32(reader)
			if e != nil {
				stop(offset, e)
				break walk
			}
			marker, e := readU32(reader)
			if e != nil {
				stop(offset, e)
				break walk
			}
			var block [352]byte
			hasMatrix := false
			if marker != 0 {
				_, e = reader.Seek(-4, io.SeekCurrent)
			} else {
				_, e = io.ReadFull(reader, block[:16])
				if e == nil {
					if binary.LittleEndian.Uint32(block[12:16]) == 0 {
						e = skipN(reader, 8)
					} else {
						_, e = io.ReadFull(reader, block[16:])
						if e == nil {
							_, e = readU32(reader)
							hasMatrix = e == nil
						}
					}
				}
			}
			if e != nil {
				stop(offset, e)
				break walk
			}
			if uint64(now)+uint64(delta) > uint64(healthRowLimit*width-1) {
				return nil, fmt.Errorf("simulation time exceeds window-table limit; use a wider --window")
			}
			now += int(delta)
			t := now
			r.EndAnchor.LastSync = &t
			w, e = getWindow()
			if e != nil {
				return nil, e
			}
			w.SyncCount++
			if hasMatrix {
				rows := make([]HealthObjects, 0, 8)
				var total uint64
				for p := 0; p < 8; p++ {
					row := block[p*44 : (p+1)*44]
					if binary.LittleEndian.Uint32(row[32:36]) == 0 {
						continue
					}
					count := binary.LittleEndian.Uint32(row[24:28])
					rows = append(rows, HealthObjects{PlayerID: p + 1, ObjectCount: count})
					total += uint64(count)
				}
				w.Objects = rows
				w.ObjectTotal = &total
				w.SampleTime = &t
				r.LoadSummary.EndObjects = append([]HealthObjects(nil), rows...)
				r.LoadSummary.EndSampleTime = &t
				if r.LoadSummary.PeakObjects == nil || total > *r.LoadSummary.PeakObjects {
					r.LoadSummary.PeakObjects = &total
					r.LoadSummary.PeakObjectsTime = &t
				}
			}
		case 3:
			if e := skipN(reader, 12); e != nil {
				stop(offset, e)
				break walk
			}
		case 4:
			if e := skipN(reader, 4); e != nil {
				stop(offset, e)
				break walk
			}
			n, e := readU32(reader)
			if e != nil {
				stop(offset, e)
				break walk
			}
			if n > healthPayloadLimit || uint64(n) > uint64(reader.Len()) {
				stop(offset, fmt.Errorf("invalid/bounded chat length %d", n))
				break walk
			}
			raw := make([]byte, int(n))
			if _, e = io.ReadFull(reader, raw); e != nil {
				stop(offset, e)
				break walk
			}
			ev, ok, name := decodeActionChat(raw)
			if !ok {
				continue
			}
			if name != "" && ev.PlayerID >= 1 && ev.PlayerID <= 8 {
				names[ev.PlayerID] = name
			}
			w.ChatCount++
			c := ChatLine{TimeMS: now, Time: FormatTime(now), PlayerID: ev.PlayerID, PlayerName: name, Text: ev.Text, Source: "game", SourceOffset: offset, Confidence: "parsed", Channel: ev.Channel}
			if now <= 120000 {
				size := len(c.Text) + len(c.PlayerName)
				if len(early) < healthRowLimit && earlyBytes+size <= healthChatBytesLimit {
					early = append(early, c)
					earlyBytes += size
				} else {
					earlyOverflow = true
				}
			} else {
				addChat(c)
			}
		case 5:
			result, e := skipStart(reader)
			if e != nil {
				stop(offset, e)
				break walk
			}
			if result.Terminal {
				r.EndAnchor.Termination = "opaque_embedded_tail"
				r.EndAnchor.StopOffset = int64(offset)
				r.Warnings = append(r.Warnings, "terminal op=5 tail is opaque; clean framing cannot be established")
				break walk
			}
		case 6:
			// The postgame container is reverse-framed. Validate lengths without
			// decoding or copying its potentially large contents.
			e := healthPostgameBoundary(reader)
			if e != nil {
				stop(offset, e)
			} else {
				r.EndAnchor.EndsCleanly = true
				r.EndAnchor.Termination = "postgame"
				r.EndAnchor.StopOffset = reader.Size()
			}
			break walk
		default:
			if e := skipReplaySaveChapter(reader, int(reader.Size())); e != nil {
				stop(offset, fmt.Errorf("unknown op %d: %w", op, e))
				break walk
			}
		}
	}
	flushEarly()
	for i := range r.Signals {
		if n := names[r.Signals[i].PlayerID]; n != "" {
			r.Signals[i].PlayerName = n
		}
	}
	for i := range r.Windows {
		w := &r.Windows[i]
		if w.To > now {
			w.To = now
		}
		if w.To > w.From {
			w.ActionsPerMin = float64(w.ActionCount) * 60000 / float64(w.To-w.From)
		}
		if w.ActionsPerMin > r.LoadSummary.PeakActions {
			r.LoadSummary.PeakActions = w.ActionsPerMin
			t := w.From
			r.LoadSummary.PeakActionsTime = &t
		}
		for j := range w.Objects {
			w.Objects[j].PlayerName = names[w.Objects[j].PlayerID]
		}
	}
	for i := range r.LoadSummary.EndObjects {
		p := &r.LoadSummary.EndObjects[i]
		p.PlayerName = names[p.PlayerID]
	}
	sort.Slice(r.LoadSummary.EndObjects, func(i, j int) bool {
		return r.LoadSummary.EndObjects[i].ObjectCount > r.LoadSummary.EndObjects[j].ObjectCount
	})
	r.LoadSummary.FirstSlope = healthSlope(r.Windows, 0, 1800000)
	r.LoadSummary.LastSlope = healthSlope(r.Windows, max(0, now-1800000), now)
	if r.EndAnchor.LastSync != nil {
		if r.EndAnchor.LastChat != nil {
			d := *r.EndAnchor.LastSync - *r.EndAnchor.LastChat
			r.EndAnchor.ChatGap = &d
		}
		if r.EndAnchor.LastResign != nil {
			d := *r.EndAnchor.LastSync - *r.EndAnchor.LastResign
			r.EndAnchor.ResignGap = &d
		}
	}
	return r, nil
}

func healthSlope(windows []HealthWindow, from, to int) *float64 {
	var n, x, y, xx, xy float64
	for _, w := range windows {
		if w.SampleTime == nil || *w.SampleTime < from || *w.SampleTime > to {
			continue
		}
		t := float64(*w.SampleTime-from) / 60000
		v := float64(*w.ObjectTotal)
		n++
		x += t
		y += v
		xx += t * t
		xy += t * v
	}
	if n < 2 || n*xx == x*x {
		return nil
	}
	s := (n*xy - x*y) / (n*xx - x*x)
	return &s
}

func healthPostgameBoundary(r replayBodyReader) error {
	if r.Len() == 0 {
		return nil
	}
	end := r.Size()
	start := end - int64(r.Len())
	if end-start < 16 {
		return fmt.Errorf("truncated postgame footer")
	}
	var b [4]byte
	read := func(off int64) (uint32, error) {
		_, e := r.ReadAt(b[:], off)
		return binary.LittleEndian.Uint32(b[:]), e
	}
	count, e := read(end - 16)
	if e != nil {
		return e
	}
	pos := end - 16
	if uint64(count) > uint64((pos-start)/8) {
		return fmt.Errorf("invalid postgame block count %d", count)
	}
	for i := uint32(0); i < count; i++ {
		if pos-start < 8 {
			return io.ErrUnexpectedEOF
		}
		length, e := read(pos - 8)
		if e != nil {
			return e
		}
		pos -= 8
		if uint64(length) > uint64(pos-start) {
			return fmt.Errorf("truncated postgame block %d", i)
		}
		pos -= int64(length)
	}
	if pos != start {
		return fmt.Errorf("unframed postgame bytes: %d", pos-start)
	}
	return nil
}

func WriteHealthText(out io.Writer, r *HealthReport) {
	fmt.Fprintf(out, "Replay health: %s\n", r.Path)
	for _, a := range []struct {
		name  string
		value *int
	}{
		{"Last sync", r.EndAnchor.LastSync}, {"Last action", r.EndAnchor.LastAction},
		{"Last game chat", r.EndAnchor.LastChat}, {"Last resign", r.EndAnchor.LastResign},
		{"Header stated duration", r.EndAnchor.StatedDuration},
		{"Chat-to-sync gap", r.EndAnchor.ChatGap}, {"Resign-to-sync gap", r.EndAnchor.ResignGap},
	} {
		value := "unavailable"
		if a.value != nil {
			value = FormatTime(*a.value)
		}
		fmt.Fprintf(out, "%s: %s\n", a.name, value)
	}
	fmt.Fprintf(out, "End: %s; clean framing=%t; body offset=%d; file offset=%d\n", r.EndAnchor.Termination, r.EndAnchor.EndsCleanly, r.EndAnchor.StopOffset, r.EndAnchor.FileStopOffset)
	for _, w := range r.Windows {
		objects := "unavailable"
		if w.ObjectTotal != nil {
			objects = fmt.Sprint(*w.ObjectTotal)
		}
		fmt.Fprintf(out, "%s..%s syncs=%d actions=%d (%.1f/min) chat=%d flares=%d objects=%s\n", FormatTime(w.From), FormatTime(w.To), w.SyncCount, w.ActionCount, w.ActionsPerMin, w.ChatCount, w.FlareCount, objects)
		if w.SampleTime != nil {
			fmt.Fprintf(out, "  sample %s:", FormatTime(*w.SampleTime))
			for _, p := range w.Objects {
				fmt.Fprintf(out, " P%d=%d", p.PlayerID, p.ObjectCount)
			}
			fmt.Fprintln(out)
		}
	}
	if r.LoadSummary.PeakObjects != nil {
		fmt.Fprintf(out, "Peak objects: %d at %s\n", *r.LoadSummary.PeakObjects, FormatTime(*r.LoadSummary.PeakObjectsTime))
	}
	if r.LoadSummary.PeakActionsTime != nil {
		fmt.Fprintf(out, "Peak actions: %.2f/min in window starting %s\n", r.LoadSummary.PeakActions, FormatTime(*r.LoadSummary.PeakActionsTime))
	}
	for _, s := range []struct {
		name  string
		value *float64
	}{{"First 30m", r.LoadSummary.FirstSlope}, {"Last 30m", r.LoadSummary.LastSlope}} {
		if s.value != nil {
			fmt.Fprintf(out, "%s slope: %.2f objects/min\n", s.name, *s.value)
		} else {
			fmt.Fprintf(out, "%s slope: unavailable\n", s.name)
		}
	}
	if r.LoadSummary.EndSampleTime != nil {
		fmt.Fprintf(out, "End counts sampled at %s:\n", FormatTime(*r.LoadSummary.EndSampleTime))
	}
	for _, p := range r.LoadSummary.EndObjects {
		fmt.Fprintf(out, "  P%d %s: %d\n", p.PlayerID, p.PlayerName, p.ObjectCount)
	}
	fmt.Fprintf(out, "Backlog excluded: %d; trigger fire counts: unavailable (%s)\n", r.BacklogExcluded, r.TriggerGraphStatus)
	for _, c := range r.Signals {
		fmt.Fprintf(out, "%s P%d %s: %s\n", c.Time, c.PlayerID, c.PlayerName, c.Text)
	}
	for _, w := range r.Warnings {
		fmt.Fprintf(out, "Warning: %s\n", w)
	}
}
