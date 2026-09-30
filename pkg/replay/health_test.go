package replay

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"aoe2kit/pkg/testfixtures"
)

func healthTestMeta(b *bytes.Buffer) {
	writeU32(b, 500)
	for i := 0; i < 8; i++ {
		writeU32(b, 0)
	}
}
func healthTestSync(b *bytes.Buffer, delta, count uint32) {
	writeU32(b, 2)
	writeU32(b, delta)
	writeU32(b, 0)
	var matrix [352]byte
	binary.LittleEndian.PutUint32(matrix[12:16], 2)
	binary.LittleEndian.PutUint32(matrix[24:28], count)
	binary.LittleEndian.PutUint32(matrix[32:36], 1)
	b.Write(matrix[:])
	writeU32(b, delta)
}
func healthTestChat(b *bytes.Buffer, text string) {
	raw := []byte(fmt.Sprintf("{%q:1,%q:0,%q:%q}", "player", "channel", "message", text))
	writeU32(b, 4)
	writeU32(b, 0)
	writeU32(b, uint32(len(raw)))
	b.Write(raw)
}
func TestHealthWindowsBacklogAndAnchors(t *testing.T) {
	var b bytes.Buffer
	healthTestMeta(&b)
	healthTestSync(&b, 1000, 10)
	for i := 0; i < 5; i++ {
		healthTestChat(&b, fmt.Sprintf("lag old %d", i))
	}
	healthTestSync(&b, 300000, 20)
	healthTestChat(&b, "well well who crashed")
	writeU32(&b, 1)
	writeU32(&b, 3)
	b.Write([]byte{11, 1, 0})
	writeU32(&b, 1)
	healthTestSync(&b, 1000, 21)
	r, err := buildHealthBody(bytes.NewReader(b.Bytes()), HealthOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Windows) != 2 || !r.EndAnchor.EndsCleanly || *r.EndAnchor.LastSync != 302000 {
		t.Fatalf("%+v", r)
	}
	if len(r.Signals) != 1 || r.Signals[0].Text != "well well who crashed" || r.BacklogExcluded != 5 {
		t.Fatalf("signals=%+v backlog=%d", r.Signals, r.BacklogExcluded)
	}
	if *r.Windows[0].ObjectTotal != 10 || *r.Windows[1].ObjectTotal != 21 || *r.LoadSummary.PeakObjects != 21 {
		t.Fatal(r.LoadSummary)
	}
	if *r.EndAnchor.LastChat != 301000 || *r.EndAnchor.ChatGap != 1000 {
		t.Fatal(r.EndAnchor)
	}
	for _, c := range r.Signals {
		if c.Source == "backlog" {
			t.Fatal(c)
		}
	}
}

func TestHealthTruncatedFrames(t *testing.T) {
	for _, tail := range [][]byte{{1}, {1, 0, 0, 0}, {2, 0, 0, 0, 1, 0, 0, 0}} {
		var b bytes.Buffer
		healthTestMeta(&b)
		b.Write(tail)
		r, e := buildHealthBody(bytes.NewReader(b.Bytes()), HealthOptions{})
		if e != nil {
			t.Fatal(e)
		}
		if r.EndAnchor.EndsCleanly || r.EndAnchor.StopOffset != 36 {
			t.Fatal(r.EndAnchor)
		}
	}
}

func TestHealthBufferedReaderParity(t *testing.T) {
	var b bytes.Buffer
	healthTestMeta(&b)
	healthTestSync(&b, 200, 7)
	healthTestChat(&b, "no lag")
	writeU32(&b, 6)
	r, e := buildHealthBody(newHealthReader(bytes.NewReader(b.Bytes()), 0, int64(b.Len())), HealthOptions{Window: time.Second})
	if e != nil || !r.EndAnchor.EndsCleanly || *r.EndAnchor.LastSync != 200 || len(r.Signals) != 1 {
		t.Fatalf("%+v %v", r, e)
	}
}

func TestHealthSafetyBounds(t *testing.T) {
	var b bytes.Buffer
	healthTestMeta(&b)
	writeU32(&b, 1)
	writeU32(&b, healthPayloadLimit+1)
	r, e := buildHealthBody(bytes.NewReader(b.Bytes()), HealthOptions{})
	if e != nil || r.EndAnchor.EndsCleanly || !strings.Contains(strings.Join(r.Warnings, " "), "1 MiB") {
		t.Fatalf("%+v %v", r, e)
	}
	if _, e = buildHealthBody(bytes.NewReader(b.Bytes()), HealthOptions{Window: -time.Second}); e == nil {
		t.Fatal("accepted negative window")
	}
}

func TestHealthSignalWords(t *testing.T) {
	for _, s := range []string{"who crashed", "laggy", "freezes every few sec", "out of sync", "dc", "disconnected"} {
		if !healthSignalPattern.MatchString(s) {
			t.Fatal(s)
		}
	}
	for _, s := range []string{"flag", "dcinside", "lagoon"} {
		if healthSignalPattern.MatchString(s) {
			t.Fatal(s)
		}
	}
}

func TestHealthMissingSamplesAndPostgameTruncation(t *testing.T) {
	var b bytes.Buffer
	healthTestMeta(&b)
	writeU32(&b, 2)
	writeU32(&b, 1000)
	writeU32(&b, 6)
	r, e := buildHealthBody(bytes.NewReader(b.Bytes()), HealthOptions{})
	if e != nil || !r.EndAnchor.EndsCleanly || len(r.Windows) != 1 || r.Windows[0].ObjectTotal != nil || r.LoadSummary.PeakObjects != nil {
		t.Fatalf("%+v %v", r, e)
	}
	b.WriteByte(0)
	r, e = buildHealthBody(bytes.NewReader(b.Bytes()), HealthOptions{})
	if e != nil || r.EndAnchor.EndsCleanly {
		t.Fatalf("%+v %v", r, e)
	}
}

func TestHealthEarlyChatByteCap(t *testing.T) {
	var b bytes.Buffer
	healthTestMeta(&b)
	healthTestSync(&b, 1000, 1)
	for i := 0; i < 6; i++ {
		healthTestChat(&b, "lag "+strings.Repeat("x", 800000))
	}
	r, e := buildHealthBody(bytes.NewReader(b.Bytes()), HealthOptions{})
	if e != nil {
		t.Fatal(e)
	}
	if len(r.Signals) != 0 || !strings.Contains(strings.Join(r.Warnings, " "), "classification is incomplete") {
		t.Fatal("capped backlog must not become game signals")
	}
}

func TestHealthGoldenBlank(t *testing.T) {
	path := testfixtures.Path(t, "save-analysis/latest_test_20260720_233538.aoe2record")
	if _, err := os.Stat(path); err != nil {
		t.Skip(err)
	}
	r, err := BuildHealth(path, HealthOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if r.EndAnchor.LastSync == nil || *r.EndAnchor.LastSync != 19760 || len(r.Windows) != 1 || !r.EndAnchor.EndsCleanly || r.EndAnchor.Termination != "postgame" {
		t.Fatalf("%+v", r.EndAnchor)
	}
	if r.EndAnchor.StopOffset != 27279 || r.EndAnchor.LastResign == nil || *r.EndAnchor.LastResign != 19760 {
		t.Fatal("wrong end anchor", r.EndAnchor)
	}
	for _, c := range r.Signals {
		if c.Source == "backlog" {
			t.Fatal("backlog leaked")
		}
	}
}
