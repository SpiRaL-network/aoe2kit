package replay

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
)

func walkJSONRows(path string, visit func([]byte) error) error {
	f, e := os.Open(path)
	if e != nil {
		return e
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	s.Buffer(make([]byte, 64<<10), 2*maxCachedOp+65536)
	for s.Scan() {
		if e = visit(s.Bytes()); e != nil {
			return e
		}
	}
	return s.Err()
}

func (c *ReplayCache) syncFiles() (*SyncReport, string, error) {
	dir := filepath.Join(c.Dir, "sync-v2")
	read := func() (*SyncReport, error) {
		f, e := os.Open(filepath.Join(dir, "summary.json"))
		if e != nil {
			return nil, e
		}
		defer f.Close()
		var r SyncReport
		e = json.NewDecoder(io.LimitReader(f, 1<<20)).Decode(&r)
		if e != nil {
			return nil, e
		}
		sizesRaw, e := os.ReadFile(filepath.Join(dir, "sizes.json"))
		if e != nil {
			return nil, e
		}
		var sizes map[string]int64
		if e = json.Unmarshal(sizesRaw, &sizes); e != nil {
			return nil, e
		}
		for _, name := range []string{"sync.jsonl", "lifecycle.jsonl", "raw_words.jsonl"} {
			st, err := os.Stat(filepath.Join(dir, name))
			if err != nil {
				return nil, err
			}
			if n, ok := sizes[name]; !ok || st.Size() != n {
				return nil, fmt.Errorf("invalid cached sync size for %s", name)
			}
		}
		return &r, e
	}
	if r, e := read(); e == nil {
		return r, dir, nil
	}
	tmp, e := os.MkdirTemp(c.Dir, ".sync-")
	if e != nil {
		return nil, "", e
	}
	defer os.RemoveAll(tmp)
	var files []*os.File
	var buffers []*bufio.Writer
	var encoders []*json.Encoder
	for _, name := range []string{"sync.jsonl", "lifecycle.jsonl", "raw_words.jsonl"} {
		f, e := os.Create(filepath.Join(tmp, name))
		if e != nil {
			return nil, "", e
		}
		defer f.Close()
		b := bufio.NewWriterSize(f, 64<<10)
		files = append(files, f)
		buffers = append(buffers, b)
		encoders = append(encoders, json.NewEncoder(b))
	}
	r := &SyncReport{Method: "body_op2_sync_stream", Verification: "structure_verified_time_framing_plus_tiered_word_semantics", WordSemantics: syncWordSemantics(), Warnings: append([]string(nil), c.Manifest.Warnings...)}
	hist := map[int]int{}
	sum := 0
	sample := 0
	var previous *SyncEvent
	e = c.WalkOps(func(op CachedOp) error {
		if op.Sync == nil {
			return nil
		}
		ev := op.Sync
		if e := encoders[0].Encode(ev); e != nil {
			return e
		}
		r.Summary.SyncCount++
		hist[ev.DeltaMS]++
		sum += ev.DeltaMS
		if len(hist) > 10000 {
			return fmt.Errorf("sync histogram exceeds 10000 buckets")
		}
		if r.Summary.SyncCount == 1 {
			r.Summary.FirstTimeMS = ev.TimeMS
			r.Summary.FirstTime = ev.Time
			r.Summary.MinDeltaMS = ev.DeltaMS
			r.Summary.MaxDeltaMS = ev.DeltaMS
		}
		r.Summary.MinDeltaMS = min(r.Summary.MinDeltaMS, ev.DeltaMS)
		r.Summary.MaxDeltaMS = max(r.Summary.MaxDeltaMS, ev.DeltaMS)
		r.Summary.LastTimeMS = ev.TimeMS
		r.Summary.LastTime = ev.Time
		switch ev.Form {
		case "delta_only":
			r.Summary.DeltaOnly++
		case "checksum_legacy":
			r.Summary.ChecksumLegacy++
		case "checksum_de":
			r.Summary.ChecksumDE++
			sample++
			for _, row := range syncRawWordRows(*ev, sample) {
				if e := encoders[2].Encode(row); e != nil {
					return e
				}
			}
			if previous != nil {
				for _, row := range syncStateDeltas(*previous, *ev) {
					if e := encoders[1].Encode(row); e != nil {
						return e
					}
				}
			}
			previous = ev
		}
		return nil
	})
	if e != nil {
		return nil, "", e
	}
	r.Summary.DurationMS = r.Summary.LastTimeMS
	r.Summary.Duration = FormatTime(r.Summary.DurationMS)
	if r.Summary.SyncCount > 0 {
		r.Summary.MeanDeltaMS = float64(sum) / float64(r.Summary.SyncCount)
	}
	for delta, count := range hist {
		r.DeltaShapes = append(r.DeltaShapes, DeltaShape{DeltaMS: delta, Count: count})
	}
	sort.Slice(r.DeltaShapes, func(i, j int) bool {
		a, b := r.DeltaShapes[i], r.DeltaShapes[j]
		if a.Count != b.Count {
			return a.Count > b.Count
		}
		return a.DeltaMS < b.DeltaMS
	})
	if len(r.DeltaShapes) > 12 {
		r.DeltaShapes = r.DeltaShapes[:12]
	}
	for i, b := range buffers {
		if e = b.Flush(); e != nil {
			return nil, "", e
		}
		if e = files[i].Close(); e != nil {
			return nil, "", e
		}
	}
	sizes := map[string]int64{}
	for _, f := range files {
		st, err := os.Stat(f.Name())
		if err != nil {
			return nil, "", err
		}
		sizes[filepath.Base(f.Name())] = st.Size()
	}
	if e = writeReplayJSON(filepath.Join(tmp, "sizes.json"), sizes); e != nil {
		return nil, "", e
	}
	if e = writeReplayJSON(filepath.Join(tmp, "summary.json"), r); e != nil {
		return nil, "", e
	}
	if e = os.Rename(tmp, dir); e != nil {
		if existing, readErr := read(); readErr == nil {
			return existing, dir, nil
		}
		stale, err := os.MkdirTemp(c.Dir, ".invalid-sync-")
		if err != nil {
			return nil, "", err
		}
		if err = os.Remove(stale); err != nil {
			return nil, "", err
		}
		if err = os.Rename(dir, stale); err != nil {
			return nil, "", err
		}
		if err = os.Rename(tmp, dir); err != nil {
			return nil, "", err
		}
	}
	return r, dir, nil
}

// WriteSyncJSON streams all arrays from cache. The library's slice-returning
// API remains bounded and fails explicitly if a caller asks for too much.
func WriteSyncJSON(out io.Writer, path string, opts SyncOptions) error {
	c, e := OpenReplayCache(path)
	if e != nil {
		return e
	}
	r, dir, e := c.syncFiles()
	if e != nil {
		return e
	}
	r.Path = path
	n := r.Summary.SyncCount
	if opts.ChecksumsOnly {
		n = r.Summary.ChecksumDE + r.Summary.ChecksumLegacy
	}
	if opts.Limit > 0 {
		n = min(n, opts.Limit)
	}
	r.Summary.EmittedEvents = n
	b := bufio.NewWriterSize(out, 64<<10)
	if _, e = io.WriteString(b, "{"); e != nil {
		return e
	}
	first := true
	field := func(name string, v any) error {
		if !first {
			if _, e := io.WriteString(b, ","); e != nil {
				return e
			}
		}
		first = false
		key, _ := json.Marshal(name)
		if _, e := b.Write(key); e != nil {
			return e
		}
		if e := b.WriteByte(':'); e != nil {
			return e
		}
		return json.NewEncoder(b).Encode(v)
	}
	for _, f := range []struct {
		k string
		v any
	}{{"path", path}, {"method", r.Method}, {"verification", r.Verification}, {"summary", r.Summary}, {"delta_shapes", r.DeltaShapes}, {"checksum_word_semantics", r.WordSemantics}} {
		if e = field(f.k, f.v); e != nil {
			return e
		}
	}
	if len(r.Warnings) > 0 {
		if e = field("warnings", r.Warnings); e != nil {
			return e
		}
	}
	array := func(name, file string, filter func([]byte) (bool, error)) error {
		// Omit empty arrays, matching the existing report's omitempty shape.
		started := false
		e := walkJSONRows(filepath.Join(dir, file), func(raw []byte) error {
			ok, e := filter(raw)
			if e != nil || !ok {
				return e
			}
			if !started {
				key, _ := json.Marshal(name)
				if _, e = fmt.Fprintf(b, ",%s:[", key); e != nil {
					return e
				}
				started = true
			} else {
				if e = b.WriteByte(','); e != nil {
					return e
				}
			}
			_, e = b.Write(raw)
			return e
		})
		if e != nil {
			return e
		}
		if started {
			return b.WriteByte(']')
		}
		return nil
	}
	emitted := 0
	if e = array("events", "sync.jsonl", func(raw []byte) (bool, error) {
		if opts.Limit > 0 && emitted >= opts.Limit {
			return false, nil
		}
		if opts.ChecksumsOnly {
			var ev SyncEvent
			if e := json.Unmarshal(raw, &ev); e != nil {
				return false, e
			}
			if ev.Form == "delta_only" {
				return false, nil
			}
		}
		emitted++
		return true, nil
	}); e != nil {
		return e
	}
	all := func([]byte) (bool, error) { return true, nil }
	if opts.RawWords {
		if e = array("raw_words", "raw_words.jsonl", all); e != nil {
			return e
		}
	}
	if e = array("state_deltas", "lifecycle.jsonl", all); e != nil {
		return e
	}
	if _, e = io.WriteString(b, "}\n"); e != nil {
		return e
	}
	return b.Flush()
}
