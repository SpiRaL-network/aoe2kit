package scenario

import (
	"bytes"
	"compress/flate"
	"encoding/binary"
	"fmt"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// Explicit test layouts, independent of the reader's version predicates.
// No bytes from a game scenario are embedded in these generated fixtures.
type artTestLayout struct {
	v                                                                                                                       string
	headerExtra, capacity, personality, playerWords, mapStrings, mapFlags, views, lock, mapExtra, pop, secondary, unitBytes int
}

var artTestLayouts = []artTestLayout{
	{"1.36", 4, 0, 0, 4, 2, 1, 0, 0, 0, 0, 0, 29},
	{"1.37", 4, 0, 0, 4, 2, 2, 0, 0, 0, 0, 0, 29},
	{"1.40", 0, 0, 0, 5, 3, 2, 128, 0, 0, 0, 0, 29},
	{"1.41", 0, 0, 0, 5, 3, 2, 128, 1, 0, 0, 0, 29},
	{"1.42", 0, 0, 0, 5, 3, 2, 128, 1, 4, 0, 0, 29},
	{"1.43", 0, 0, 0, 5, 3, 2, 128, 1, 4, 0, 0, 29},
	{"1.44", 0, 0, 0, 5, 3, 2, 128, 1, 4, 64, 0, 29},
	{"1.45", 0, 0, 0, 5, 3, 2, 128, 1, 4, 64, 4, 29},
	{"1.46", 0, 0, 0, 5, 3, 2, 128, 1, 4, 64, 4, 29},
	{"1.47", 0, 0, 0, 5, 3, 2, 128, 1, 0, 64, 4, 29},
	{"1.48", 0, 0, 0, 5, 3, 2, 128, 1, 0, 64, 4, 29},
	{"1.49", 0, 0, 0, 5, 3, 2, 128, 1, 0, 64, 4, 29},
	{"1.50", 0, 0, 0, 5, 3, 2, 128, 1, 0, 64, 4, 29},
	{"1.51", 0, 0, 0, 5, 3, 2, 128, 1, 0, 64, 4, 29},
	{"1.52", 0, 8, 0, 5, 3, 2, 128, 1, 0, 64, 4, 29},
	{"1.53", 0, 8, 64, 5, 3, 2, 128, 1, 0, 64, 4, 29},
	{"1.54", 0, 8, 64, 5, 3, 2, 128, 1, 0, 64, 4, 33},
	{"1.55", 0, 8, 64, 5, 3, 2, 128, 1, 0, 64, 4, 37},
	{"1.56", 0, 8, 64, 0, 3, 2, 128, 1, 0, 64, 4, 37},
	{"1.57", 0, 8, 64, 0, 3, 2, 128, 1, 0, 64, 4, 37},
	{"1.58", 0, 8, 64, 0, 3, 2, 128, 1, 0, 64, 4, 37},
	{"1.59", 0, 8, 64, 0, 3, 2, 128, 1, 0, 64, 4, 38},
}

type artTestBytes struct{ bytes.Buffer }

func (b *artTestBytes) put(v any) {
	if err := binary.Write(&b.Buffer, binary.LittleEndian, v); err != nil {
		panic(err)
	}
}
func (b *artTestBytes) zeros(n int) { b.Write(make([]byte, n)) }
func (b *artTestBytes) str(s string, width int) {
	if width == 2 {
		b.put(uint16(len(s)))
	} else {
		b.put(uint32(len(s)))
	}
	b.WriteString(s)
}

func artSynthetic(t *testing.T, p artTestLayout) (header, body []byte, marks map[string]int) {
	t.Helper()
	marks = map[string]int{}
	h := &artTestBytes{}
	h.WriteString(p.v)
	h.zeros(12)
	h.str("synthetic instructions", 4)
	h.zeros(p.headerExtra)
	h.put(uint32(2))
	h.zeros(8)
	h.put(uint32(2))
	h.put([2]uint32{2, 7})
	h.str("synthetic", 4)
	h.put(uint32(99))
	b := &artTestBytes{}
	b.put(uint32(10))
	var version float32
	fmt.Sscan(p.v, &version)
	b.put(version)
	b.zeros(p.capacity)
	b.zeros(16*256 + 16*4)
	for i := 0; i < 16; i++ {
		if p.playerWords > 0 {
			b.zeros(p.playerWords * 4)
		} else {
			b.zeros(8)
			b.put(uint16(0xa60))
			b.str("CIV", 2)
			b.put(uint16(0xa60))
			b.str("ARCH", 2)
			b.zeros(4)
		}
	}
	b.zeros(64 + p.personality + 9)
	b.str("synthetic.aoe2scenario", 2)
	b.zeros(24)
	for i := 0; i < 9; i++ {
		b.str("message", 2)
	}
	// Empty bitmap, followed by AI strings and one nonempty embedded payload.
	b.str("", 2)
	b.zeros(14)
	for i := 0; i < 48; i++ {
		b.str("ai", 2)
	}
	for i := 0; i < 16; i++ {
		b.zeros(8)
		b.str("(defrule (true) => (do-nothing))", 4)
	}
	b.zeros(16)
	b.put(uint32(0xffffff9d))
	for i := 0; i < 16; i++ {
		b.zeros(24)
		b.put(int32(7 - i%8))
	}
	b.zeros(44 + 16*16*4 + 11520)
	b.put(uint32(0xffffff9d))
	b.zeros(16*4 + 4)
	marks["options"] = b.Len()
	for list := 0; list < 3; list++ {
		b.put(uint32(2))
		b.zeros(15 * 4)
		b.put([2]uint32{3, 4})
	}
	b.zeros(112) // modes, ages, opaque fields and trigger count
	for i := 0; i < p.mapStrings; i++ {
		b.put(uint16(0xa60))
		b.str("Default", 2)
	}
	b.zeros(p.mapFlags + p.views + p.lock + p.mapExtra + p.pop + p.secondary)
	marks["map_marker"] = b.Len()
	b.put(uint32(0xdeadf00d))
	b.put(uint32(2))
	b.zeros(1)
	marks["dimensions"] = b.Len()
	b.put(uint32(2))
	b.put(uint32(3))
	for i := 0; i < 6; i++ {
		b.put(uint8(i + 1))
		b.put(uint8(i))
		b.zeros(3)
		b.put(int16(i - 1))
	}
	b.put(uint32(9))
	b.zeros(8 * 28)
	b.put(uint32(9))
	for i := 0; i < 8; i++ {
		b.str("player", 2)
		b.zeros(8 + 4 + 1)
		b.put(uint16(3))
		b.zeros(3 * 5)
		b.put(uint32(i)) // Deliberately differs from the authoritative color.
		if i%2 == 0 {
			b.put(float32(2))
			b.put(uint16(1))
			b.zeros(7 + 44)
			b.put(uint8(1))
			b.zeros(7 + 32 + 4)
		} else {
			b.put(float32(1.5))
			b.put(uint16(0))
			b.zeros(7 + 4)
		}
	}
	marks["unit_count"] = b.Len()
	for player := 0; player < 9; player++ {
		if player > 1 {
			b.put(uint32(0))
			continue
		}
		b.put(uint32(1))
		start := b.Len()
		if player == 0 {
			marks["unit"] = start
		}
		b.put([3]float32{1.25, 2.5, .75})
		b.put(int32(9 + player))
		b.put(uint16(896 + player))
		b.put(uint8(2))
		b.put(float32(1))
		b.put(uint16(3))
		b.put(int32(-1))
		if p.unitBytes == 38 {
			b.put(uint8(7))
		}
		if p.unitBytes >= 33 {
			b.put(int32(1234))
		}
		if p.unitBytes >= 37 {
			b.str("caption", 4)
		}
		want := p.unitBytes
		if p.unitBytes >= 37 {
			want += len("caption")
		}
		if b.Len()-start != want {
			t.Fatal("bad synthetic unit length")
		}
	}
	marks["units_end"] = b.Len()
	b.WriteString("deliberately unsupported trigger tail")
	return h.Bytes(), b.Bytes(), marks
}
func artCompressed(t *testing.T, h, b []byte) []byte {
	t.Helper()
	var out bytes.Buffer
	out.Write(h)
	w, e := flate.NewWriter(&out, flate.BestSpeed)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = w.Write(b); e != nil {
		t.Fatal(e)
	}
	if e = w.Close(); e != nil {
		t.Fatal(e)
	}
	return out.Bytes()
}

func TestArtworkVersionsSynthetic(t *testing.T) {
	for _, p := range artTestLayouts {
		t.Run(p.v, func(t *testing.T) {
			h, b, m := artSynthetic(t, p)
			data := artCompressed(t, h, b)
			a, e := ParseArtwork(data)
			if e != nil {
				t.Fatal(e)
			}
			if a.Version != p.v || a.Width != 2 || a.Height != 3 || a.ColorMood != "Default" || a.Units.Total != 2 || a.UnparsedBytes != len(b)-m["units_end"] {
				t.Fatalf("bad artwork metadata: %+v", a)
			}
			if a.PlayerColors[1] != 7 || a.PlayerColors[8] != 0 {
				t.Fatalf("colors: %v", a.PlayerColors)
			}
			for i, tile := range a.Tiles {
				if tile.X != i%2 || tile.Y != i/2 || tile.TerrainID != i+1 || tile.Elevation != i || tile.Layer != i-1 {
					t.Fatalf("tile%d: %+v", i, tile)
				}
			}
			for player := 0; player < 2; player++ {
				u := a.Units.Sections[player].Units[0]
				if u.UnitConst != 896+player || u.X != 1.25 || u.Y != 2.5 || u.Z != .75 || u.Rotation != 1 || u.Status != 2 || u.InitialAnimationFrame != 3 {
					t.Fatalf("unit: %+v", u)
				}
				if p.unitBytes >= 37 && u.CaptionString != "caption" {
					t.Fatal("caption framing")
				}
			}
			if _, ok := any(a).(interface{ Write(string) error }); ok {
				t.Fatal("artwork unexpectedly writable")
			}
			if p.v < "1.58" {
				if SupportsReadVersion(p.v) {
					t.Fatal("legacy full parsing enabled")
				}
				if _, e := Parse(data); e == nil {
					t.Fatal("full parse unexpectedly accepted legacy file")
				}
			}
			if p.v == "1.50" {
				copy(data[:4], []byte{'1', '.', '5', 0})
				alias, e := ParseArtwork(data)
				if e != nil || alias.Version != "1.50" {
					t.Fatalf("1.5 NUL alias: %v", e)
				}
			}
		})
	}
	for _, v := range []string{"1.35", "1.38", "1.39", "1.60", "2.00"} {
		if _, e := ParseArtwork([]byte(v)); e == nil {
			t.Fatalf("accepted unmeasured %s", v)
		}
	}
}

func TestArtworkRejectsMalformed(t *testing.T) {
	h, b, m := artSynthetic(t, artTestLayouts[len(artTestLayouts)-1])
	for _, tc := range []struct {
		name   string
		change func([]byte)
	}{
		{"body version", func(b []byte) { binary.LittleEndian.PutUint32(b[4:], math.Float32bits(1.55)) }},
		{"options allocation", func(b []byte) { binary.LittleEndian.PutUint32(b[m["options"]:], 0xffffffff) }},
		{"map marker", func(b []byte) { b[m["map_marker"]] = 0 }},
		{"map allocation", func(b []byte) { binary.LittleEndian.PutUint32(b[m["dimensions"]:], 0xffffffff) }},
		{"unit allocation", func(b []byte) { binary.LittleEndian.PutUint32(b[m["unit_count"]:], 0xffffffff) }},
		{"unit NaN", func(b []byte) { binary.LittleEndian.PutUint32(b[m["unit"]:], math.Float32bits(float32(math.NaN()))) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bad := append([]byte(nil), b...)
			tc.change(bad)
			if _, e := ParseArtwork(artCompressed(t, h, bad)); e == nil {
				t.Fatal("accepted malformed input")
			}
		})
	}
	for _, end := range []int{0, 7, 4000, m["options"] + 4, m["map_marker"] + 4, m["unit"] + 20, m["units_end"] - 1} {
		if _, e := ParseArtwork(artCompressed(t, h, b[:end])); e == nil {
			t.Fatalf("accepted truncated body@%d", end)
		}
	}
	data := artCompressed(t, h, b)
	for _, end := range []int{0, 3, 16, len(h) - 1, len(data) - 2} {
		if _, e := ParseArtwork(data[:end]); e == nil {
			t.Fatalf("accepted truncated file@%d", end)
		}
	}
}

func compareArtworkFull(t *testing.T, data []byte) {
	t.Helper()
	f, e := ParseWithOptions(data, ParseOptions{StopBeforeSection: "Triggers"})
	if e != nil {
		t.Fatal(e)
	}
	a, e := ParseArtwork(data)
	if e != nil {
		t.Fatal(e)
	}
	terrain, e := f.Terrain(TerrainOptions{IncludeTiles: true})
	if e != nil {
		t.Fatal(e)
	}
	if a.Width != terrain.Width || a.Height != terrain.Height || len(a.Tiles) != len(terrain.Tiles) {
		t.Fatal("map dimensions diverged")
	}
	for i, want := range terrain.Tiles {
		want.UnusedHex = ""
		if a.Tiles[i] != want {
			t.Fatalf("tile%d diverged", i)
		}
	}
	if !reflect.DeepEqual(a.Units, *f.Units) {
		t.Fatal("units diverged from full parser")
	}
	if a.ColorMood != f.Settings().Map.ColorMood {
		t.Fatal("mood diverged")
	}
	for _, p := range f.Players {
		if p.Player < 8 && a.PlayerColors[p.Player+1] != p.Color {
			t.Fatalf("color player%d diverged", p.Player)
		}
	}
}

func TestArtworkCurrentCodecParity(t *testing.T) {
	path := filepath.Join(t.TempDir(), "synthetic.aoe2scenario")
	if _, e := WriteBlankScenarioFile(path, BlankOptions{MapWidth: 80, MapHeight: 80}); e != nil {
		t.Fatal(e)
	}
	b, e := os.ReadFile(path)
	if e != nil {
		t.Fatal(e)
	}
	compareArtworkFull(t, b)
}

func TestArtworkPrivateCorpus(t *testing.T) {
	root := os.Getenv("AOE2KIT_SCENARIO_ARTWORK_CORPUS")
	if root == "" {
		t.Skip("set AOE2KIT_SCENARIO_ARTWORK_CORPUS for private campaign checks")
	}
	if _, e := os.Stat(root); os.IsNotExist(e) {
		t.Skip("private corpus missing")
	} else if e != nil {
		t.Fatal(e)
	}
	counts := map[string]int{}
	total := 0
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.EqualFold(filepath.Ext(path), ".aoe2scenario") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		a, err := ParseArtwork(data)
		if err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		counts[a.Version]++
		total++
		if SupportsReadVersion(a.Version) {
			t.Run(filepath.Base(path), func(t *testing.T) { compareArtworkFull(t, data) })
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if total == 0 {
		t.Fatal("no scenarios in corpus")
	}
	t.Logf("artwork reads=%d versions=%v", total, counts)
}
