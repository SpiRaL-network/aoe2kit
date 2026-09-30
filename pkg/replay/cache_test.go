package replay

import (
	"archive/zip"
	"bytes"
	"compress/flate"
	"context"
	"encoding/binary"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func cacheFixture(t *testing.T, headerSize int, body []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.aoe2record")
	f, e := os.Create(path)
	if e != nil {
		t.Fatal(e)
	}
	defer f.Close()
	f.Write(make([]byte, 8))
	z, _ := flate.NewWriter(f, flate.BestSpeed)
	block := make([]byte, 65536)
	for left := headerSize; left > 0; {
		n := min(left, len(block))
		if _, e = z.Write(block[:n]); e != nil {
			t.Fatal(e)
		}
		left -= n
	}
	if e = z.Close(); e != nil {
		t.Fatal(e)
	}
	end, _ := f.Seek(0, io.SeekCurrent)
	var prefix [8]byte
	binary.LittleEndian.PutUint32(prefix[:4], uint32(end))
	if _, e = f.WriteAt(prefix[:], 0); e != nil {
		t.Fatal(e)
	}
	if _, e = f.Write(body); e != nil {
		t.Fatal(e)
	}
	return path
}

func TestReplayCacheParityAndZipHit(t *testing.T) {
	t.Setenv("AOE2KIT_REPLAY_CACHE", t.TempDir())
	var b bytes.Buffer
	healthTestMeta(&b)
	healthTestSync(&b, 1000, 10)
	healthTestChat(&b, "hello")
	healthTestSync(&b, 1000, 11)
	path := cacheFixture(t, 1024, b.Bytes())
	c, e := OpenReplayCache(path)
	if e != nil {
		t.Fatal(e)
	}
	opts := EventOptions{IncludeSystemEvents: true, IncludeUntypedAction: true}
	got, names, counts, _, e := extractCachedEvents(c, opts)
	if e != nil {
		t.Fatal(e)
	}
	want, wn, wc, _ := ExtractActionStreamEvents(b.Bytes(), opts)
	if !reflect.DeepEqual(got, want) || !reflect.DeepEqual(names, wn) || !reflect.DeepEqual(counts, wc) {
		t.Fatalf("events differ: %+v / %+v, counts %+v / %+v", got, want, counts, wc)
	}
	before, _ := os.Stat(filepath.Join(c.Dir, "ops.jsonl"))
	again, e := OpenReplayCache(path)
	if e != nil {
		t.Fatal(e)
	}
	after, _ := os.Stat(filepath.Join(again.Dir, "ops.jsonl"))
	if !before.ModTime().Equal(after.ModTime()) {
		t.Fatal("cache hit rewrote body")
	}
	zp := filepath.Join(t.TempDir(), "test.zip")
	out, _ := os.Create(zp)
	zw := zip.NewWriter(out)
	entry, _ := zw.Create("test.aoe2record")
	in, _ := os.Open(path)
	io.Copy(entry, in)
	in.Close()
	zw.Close()
	out.Close()
	zipped, e := OpenReplayCache(zp)
	if e != nil {
		t.Fatal(e)
	}
	if zipped.Dir != c.Dir {
		t.Fatal("different cache for identical zip content")
	}
	if _, e = os.Stat(zipped.Source); e != nil {
		t.Fatal(e)
	}
	var streamed bytes.Buffer
	if e = WriteSyncJSON(&streamed, path, SyncOptions{RawWords: true}); e != nil {
		t.Fatal(e)
	}
	var decoded SyncReport
	if e = json.Unmarshal(streamed.Bytes(), &decoded); e != nil {
		t.Fatal(e)
	}
	report, e := BuildSyncStream(path, SyncOptions{RawWords: true})
	if e != nil {
		t.Fatal(e)
	}
	if !reflect.DeepEqual(*report, decoded) {
		t.Fatal("streamed sync differs from bounded API")
	}
	_, syncDir, e := c.syncFiles()
	if e != nil {
		t.Fatal(e)
	}
	if e = os.Truncate(filepath.Join(syncDir, "sync.jsonl"), 0); e != nil {
		t.Fatal(e)
	}
	if rebuilt, e := BuildSyncStream(path, SyncOptions{RawWords: true}); e != nil || !reflect.DeepEqual(report, rebuilt) {
		t.Fatalf("derived cache rebuild failed: %v", e)
	}
	if e = os.Truncate(filepath.Join(c.Dir, "ops.jsonl"), 0); e != nil {
		t.Fatal(e)
	}
	if _, e = OpenReplayCache(path); e != nil {
		t.Fatal(e)
	}
	rebuilt, _ := os.Stat(filepath.Join(c.Dir, "ops.jsonl"))
	if rebuilt.Size() == 0 {
		t.Fatal("corrupt cache not rebuilt")
	}
}

func TestReplayCacheMemoryContract(t *testing.T) {
	if os.Getenv("AOE2KIT_CACHE_MEMORY_CHILD") == "1" {
		stop := TrackMemory(os.Stdout)
		var b bytes.Buffer
		healthTestMeta(&b)
		var action [1024]byte
		action[0] = 255
		for i := 0; i < 10240; i++ {
			writeU32(&b, 1)
			writeU32(&b, 1024)
			b.Write(action[:])
			writeU32(&b, uint32(i))
		}
		path := cacheFixture(t, 128<<20, b.Bytes())
		c, e := OpenReplayCache(path)
		if e != nil {
			t.Fatal(e)
		}
		if c.Manifest.HeaderBytes != 128<<20 {
			t.Fatal(c.Manifest)
		}
		if _, e = c.Metadata(); e != nil {
			t.Fatal(e)
		}
		if e = WriteSyncJSON(io.Discard, path, SyncOptions{}); e != nil {
			t.Fatal(e)
		}
		stop()
		return
	}
	if testing.Short() {
		t.Skip("subprocess memory regression")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestReplayCacheMemoryContract$")
	cmd.Env = append(os.Environ(), "AOE2KIT_CACHE_MEMORY_CHILD=1", "AOE2KIT_REPLAY_CACHE="+t.TempDir(), "GOMAXPROCS=2", "GOMEMLIMIT=192MiB")
	raw, e := cmd.CombinedOutput()
	if e != nil {
		t.Fatalf("%s: %v", raw, e)
	}
	var report struct {
		RSS  uint64 `json:"process_peak_rss_bytes"`
		Heap uint64 `json:"sampled_peak_heap_bytes"`
	}
	if e = json.NewDecoder(bytes.NewReader(raw)).Decode(&report); e != nil {
		t.Fatalf("%s: %v", raw, e)
	}
	t.Logf("memory: %s", raw)
	if report.RSS > 300<<20 || report.Heap > 300<<20 {
		t.Fatalf("memory contract exceeded: %+v", report)
	}
}
