package replay

import (
	"archive/zip"
	"bufio"
	"compress/flate"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const replayCacheVersion = 1
const maxCachedHeader = 2 << 30
const maxCachedOp = 16 << 20
const maxReportBytes = 32 << 20

type ReplayCache struct {
	Dir      string
	Source   string
	Manifest CacheManifest
}

type CacheManifest struct {
	Version      int      `json:"version"`
	SHA256       string   `json:"sha256"`
	HeaderLength int64    `json:"header_length"`
	HeaderBytes  int64    `json:"header_bytes"`
	RecordBytes  int64    `json:"record_bytes"`
	OpsBytes     int64    `json:"ops_bytes"`
	DurationMS   int      `json:"duration_ms"`
	Warnings     []string `json:"warnings,omitempty"`
}

type CachedOp struct {
	ID           uint32     `json:"op"`
	Offset       int        `json:"offset"`
	TimeMS       int        `json:"time_ms"`
	ActionID     int        `json:"action_id,omitempty"`
	Sequence     int        `json:"sequence,omitempty"`
	Payload      []byte     `json:"payload,omitempty"`
	Sync         *SyncEvent `json:"sync,omitempty"`
	Terminal     bool       `json:"terminal,omitempty"`
	SkippedBytes int        `json:"skipped_bytes,omitempty"`
}

func replayCacheRoot() (string, error) {
	if root := os.Getenv("AOE2KIT_REPLAY_CACHE"); root != "" {
		return root, nil
	}
	root, e := os.UserCacheDir()
	if e != nil {
		return "", e
	}
	return filepath.Join(root, "aoe2kit", "replay-cache"), nil
}

// OpenReplayCache hashes with a fixed buffer. A completed manifest is the commit
// marker; interrupted builds never become visible as usable cache entries.
func OpenReplayCache(path string) (*ReplayCache, error) {
	root, e := replayCacheRoot()
	if e != nil {
		return nil, e
	}
	if e = os.MkdirAll(root, 0700); e != nil {
		return nil, e
	}
	source := path
	var staged string
	if strings.EqualFold(filepath.Ext(path), ".zip") {
		z, e := zip.OpenReader(path)
		if e != nil {
			return nil, e
		}
		defer z.Close()
		for _, entry := range z.File {
			if !strings.EqualFold(filepath.Ext(entry.Name), ".aoe2record") {
				continue
			}
			in, e := entry.Open()
			if e != nil {
				return nil, e
			}
			out, e := os.CreateTemp(root, ".record-")
			if e != nil {
				in.Close()
				return nil, e
			}
			staged = out.Name()
			defer os.Remove(staged)
			n, copyErr := io.Copy(out, io.LimitReader(in, (1<<30)+1))
			in.Close()
			closeErr := out.Close()
			if copyErr != nil {
				return nil, copyErr
			}
			if closeErr != nil {
				return nil, closeErr
			}
			if n > 1<<30 {
				return nil, fmt.Errorf("unzipped recording exceeds 1 GiB disk limit")
			}
			source = staged
			break
		}
		if staged == "" {
			return nil, fmt.Errorf("zip contains no .aoe2record")
		}
	}
	f, e := os.Open(source)
	if e != nil {
		return nil, e
	}
	defer f.Close()
	before, e := f.Stat()
	if e != nil {
		return nil, e
	}
	if before.Size() > 1<<30 {
		return nil, fmt.Errorf("recording exceeds 1 GiB input budget")
	}
	h := sha256.New()
	if _, e = io.Copy(h, f); e != nil {
		return nil, e
	}
	sum := hex.EncodeToString(h.Sum(nil))
	dir := filepath.Join(root, sum)
	c := &ReplayCache{Dir: dir, Source: source}
	if c.readManifest(sum) == nil {
		// Plain bodies remain in place. Zip bodies use a stable extracted file.
		if staged != "" {
			c.Source = filepath.Join(dir, "record.bin")
			if _, err := os.Stat(c.Source); os.IsNotExist(err) {
				if err := os.Rename(staged, c.Source); err != nil {
					return nil, err
				}
			}
		}
		return c, nil
	}
	tmp, e := os.MkdirTemp(root, ".build-")
	if e != nil {
		return nil, e
	}
	defer os.RemoveAll(tmp)
	if _, e = f.Seek(0, io.SeekStart); e != nil {
		return nil, e
	}
	var prefix [8]byte
	if _, e = io.ReadFull(f, prefix[:]); e != nil {
		return nil, e
	}
	headerLen := int64(binary.LittleEndian.Uint32(prefix[:4]))
	if headerLen < 8 || headerLen > before.Size() {
		return nil, fmt.Errorf("invalid record header length %d", headerLen)
	}
	header, e := os.OpenFile(filepath.Join(tmp, "header.bin"), os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0600)
	if e != nil {
		return nil, e
	}
	z := flate.NewReader(io.NewSectionReader(f, 8, headerLen-8))
	n, e := io.Copy(header, io.LimitReader(z, maxCachedHeader+1))
	z.Close()
	ce := header.Close()
	if e != nil {
		return nil, e
	}
	if ce != nil {
		return nil, ce
	}
	if n > maxCachedHeader {
		return nil, fmt.Errorf("inflated header exceeds 2 GiB disk safety limit")
	}
	c.Manifest = CacheManifest{Version: replayCacheVersion, SHA256: sum, HeaderLength: headerLen, HeaderBytes: n, RecordBytes: before.Size()}
	ops, e := os.OpenFile(filepath.Join(tmp, "ops.jsonl"), os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0600)
	if e != nil {
		return nil, e
	}
	bw := bufio.NewWriterSize(ops, 64<<10)
	encoder := json.NewEncoder(bw)
	duration, warnings, walkErr := walkRecordOps(newHealthReader(f, headerLen, before.Size()-headerLen), func(op CachedOp) error { return encoder.Encode(op) })
	flushErr := bw.Flush()
	ops.Close()
	if walkErr != nil {
		return nil, walkErr
	}
	if flushErr != nil {
		return nil, flushErr
	}
	c.Manifest.DurationMS = duration
	c.Manifest.Warnings = warnings
	stat, e := os.Stat(filepath.Join(tmp, "ops.jsonl"))
	if e != nil {
		return nil, e
	}
	c.Manifest.OpsBytes = stat.Size()
	after, e := f.Stat()
	if e != nil {
		return nil, e
	}
	if after.Size() != before.Size() || !after.ModTime().Equal(before.ModTime()) {
		return nil, fmt.Errorf("recording changed during cache build")
	}
	// Cache zip extraction as well, so random access never re-inflates the zip.
	if staged != "" {
		if e = os.Rename(staged, filepath.Join(tmp, "record.bin")); e != nil {
			return nil, e
		}
		c.Source = filepath.Join(dir, "record.bin")
	}
	raw, e := json.Marshal(c.Manifest)
	if e != nil {
		return nil, e
	}
	if e = os.WriteFile(filepath.Join(tmp, "manifest.json"), raw, 0600); e != nil {
		return nil, e
	}
	// Existing corrupt entries are preserved for diagnosis, not trusted.
	if _, e = os.Stat(dir); e == nil {
		other := &ReplayCache{Dir: dir}
		if other.readManifest(sum) == nil {
			c.Manifest = other.Manifest
			return c, nil
		}
		stale, e := os.MkdirTemp(root, ".invalid-")
		if e != nil {
			return nil, e
		}
		os.Remove(stale)
		if e = os.Rename(dir, stale); e != nil {
			return nil, e
		}
	}
	if e = os.Rename(tmp, dir); e != nil {
		if c.readManifest(sum) != nil {
			return nil, e
		}
	}
	return c, nil
}

func (c *ReplayCache) readManifest(sum string) error {
	f, e := os.Open(filepath.Join(c.Dir, "manifest.json"))
	if e != nil {
		return e
	}
	defer f.Close()
	if e = json.NewDecoder(io.LimitReader(f, 1<<20)).Decode(&c.Manifest); e != nil {
		return e
	}
	m := c.Manifest
	if m.Version != replayCacheVersion || m.SHA256 != sum {
		return fmt.Errorf("stale replay cache")
	}
	for name, size := range map[string]int64{"header.bin": m.HeaderBytes, "ops.jsonl": m.OpsBytes} {
		s, e := os.Stat(filepath.Join(c.Dir, name))
		if e != nil {
			return e
		}
		if s.Size() != size {
			return fmt.Errorf("cache size mismatch for %s", name)
		}
	}
	return nil
}

// HeaderReader transfers ownership of a read-only disk handle to the caller.
func (c *ReplayCache) HeaderReader() (*os.File, error) {
	return os.Open(filepath.Join(c.Dir, "header.bin"))
}

func (c *ReplayCache) WalkOps(visit func(CachedOp) error) error {
	f, e := os.Open(filepath.Join(c.Dir, "ops.jsonl"))
	if e != nil {
		return e
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	s.Buffer(make([]byte, 64<<10), 2*maxCachedOp+65536)
	for s.Scan() {
		var op CachedOp
		if e = json.Unmarshal(s.Bytes(), &op); e != nil {
			return fmt.Errorf("invalid cached operation: %w", e)
		}
		if op.ID == 3 && len(op.Payload) != 12 {
			return fmt.Errorf("invalid cached camera payload")
		}
		if op.ID == 2 && op.Sync == nil {
			return fmt.Errorf("cached sync missing payload")
		}
		if e = visit(op); e != nil {
			return e
		}
	}
	return s.Err()
}

func walkRecordOps(r replayBodyReader, visit func(CachedOp) error) (int, []string, error) {
	if e := readReplayMeta(r); e != nil {
		return 0, []string{"action-stream meta parse failed: " + e.Error()}, nil
	}
	now := 0
	for r.Len() > 0 {
		offset := int(r.Size()) - r.Len()
		id, e := readU32(r)
		if e != nil {
			return now, []string{fmt.Sprintf("body offset %d: %v", offset, e)}, nil
		}
		op := CachedOp{ID: id, Offset: offset, TimeMS: now}
		switch id {
		case 1:
			n, ok := peekU32(r, 0)
			if ok && n > maxCachedOp {
				return now, nil, fmt.Errorf("action at %d exceeds 16 MiB operation budget", offset)
			}
			op.ActionID, op.Payload, op.Sequence, e = readReplayAction(r)
		case 2:
			op.Sync, e = readCachedSync(r, offset, now)
			if e == nil {
				now = op.Sync.TimeMS
				op.TimeMS = now
			}
		case 3:
			op.Payload = make([]byte, 12)
			_, e = io.ReadFull(r, op.Payload)
		case 4:
			e = skipN(r, 4)
			if e == nil {
				var n uint32
				n, e = readU32(r)
				if e == nil {
					if n > maxCachedOp || uint64(n) > uint64(r.Len()) {
						e = fmt.Errorf("chat length %d exceeds bounded input", n)
					} else {
						op.Payload = make([]byte, int(n))
						_, e = io.ReadFull(r, op.Payload)
					}
				}
			}
		case 5:
			var s startSkipResult
			s, e = skipStart(r)
			op.Terminal = s.Terminal
			op.SkippedBytes = s.SkippedBytes
		case 6:
			if r.Len() > maxCachedOp {
				return now, nil, fmt.Errorf("postgame exceeds 16 MiB operation budget")
			}
			op.Payload = make([]byte, r.Len())
			_, e = io.ReadFull(r, op.Payload)
			op.Terminal = true
		default:
			e = skipReplaySaveChapter(r, int(r.Size()))
			op.SkippedBytes = int(r.Size()) - r.Len() - offset
		}
		if e != nil {
			return now, []string{fmt.Sprintf("body op %d offset %d: %v", id, offset, e)}, nil
		}
		if e = visit(op); e != nil {
			return now, nil, e
		}
		if op.Terminal {
			break
		}
	}
	return now, nil, nil
}

func readCachedSync(r replayBodyReader, offset, now int) (*SyncEvent, error) {
	d, e := readU32(r)
	if e != nil {
		return nil, e
	}
	m, e := readU32(r)
	if e != nil {
		return nil, e
	}
	ev := &SyncEvent{SourceOffset: offset, DeltaMS: int(d), TimeMS: now + int(d), Time: FormatTime(now + int(d)), Form: "delta_only"}
	if m != 0 {
		_, e = r.Seek(-4, io.SeekCurrent)
		return ev, e
	}
	var block [352]byte
	if _, e = io.ReadFull(r, block[:16]); e != nil {
		return nil, e
	}
	if binary.LittleEndian.Uint32(block[12:16]) == 0 {
		ev.Form = "checksum_legacy"
		ev.ProbeHex = hex.EncodeToString(block[:16])
		if _, e = io.ReadFull(r, block[16:24]); e != nil {
			return nil, e
		}
		ev.PayloadHex = hex.EncodeToString(block[16:24])
		return ev, nil
	}
	if _, e = io.ReadFull(r, block[16:]); e != nil {
		return nil, e
	}
	t, e := readU32(r)
	if e != nil {
		return nil, e
	}
	ev.Form = "checksum_de"
	ev.TrailerU32 = &t
	ev.Matrix = make([][]uint32, 8)
	for p := 0; p < 8; p++ {
		ev.Matrix[p] = make([]uint32, 11)
		for w := 0; w < 11; w++ {
			i := (p*11 + w) * 4
			ev.Matrix[p][w] = binary.LittleEndian.Uint32(block[i : i+4])
		}
	}
	return ev, nil
}
