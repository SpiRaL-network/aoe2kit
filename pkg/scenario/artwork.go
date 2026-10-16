package scenario

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
)

// Artwork is a read-only projection, not a File. It deliberately has no write,
// recipe, rebuild or trigger APIs. Reading it does not extend full-format support.
type Artwork struct {
	Version       string        `json:"version"`
	Scope         string        `json:"scope"`
	Width         int           `json:"width"`
	Height        int           `json:"height"`
	ColorMood     string        `json:"color_mood,omitempty"`
	Tiles         []TerrainTile `json:"tiles"`
	Units         UnitInfo      `json:"units"`
	PlayerColors  map[int]int   `json:"player_colors"`
	Sections      []SectionInfo `json:"sections"`
	UnparsedBytes int           `json:"unparsed_bytes"`
}

// ArtworkVersions lists measured layout families, not full read/write support.
// 1.38 and 1.39 have no corpus evidence and remain unsupported.
func ArtworkVersions() []string {
	out := []string{"1.36", "1.37"}
	for v := 40; v <= 59; v++ {
		out = append(out, fmt.Sprintf("1.%02d", v))
	}
	return out
}

func artworkVersion(s string) (int, error) {
	s = strings.TrimRight(s, "\x00")
	if s == "1.5" {
		s = "1.50"
	}
	for _, supported := range ArtworkVersions() {
		if s == supported {
			v, _ := strconv.Atoi(s[2:])
			return v, nil
		}
	}
	return 0, fmt.Errorf("unsupported artwork scenario version %q (read-only artwork: 1.36, 1.37, 1.40..1.59; full parsing unchanged)", s)
}

func OpenArtwork(path string) (*Artwork, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParseArtwork(b)
}

// ParseArtwork reads the sequential prefix through Units and ignores all later
// sections. Version branches come from byte measurements of campaign families;
// no offset scanning, plausible-map search or generic schema walker is used.
func ParseArtwork(data []byte) (*Artwork, error) {
	if len(data) < 4 {
		return nil, fmt.Errorf("artwork: truncated scenario version")
	}
	v, err := artworkVersion(string(data[:4]))
	if err != nil {
		return nil, err
	}
	h := artReader{byteCursor: byteCursor{data: data}}
	h.skip(16) // version, unused header length, savable marker, timestamp
	h.str(4)
	if v < 40 {
		h.skip(4)
	} // extra header word in measured 1.36/1.37 saves
	h.skip(12) // player count and two header words
	n := h.count(1024)
	h.skip(n * 4)
	h.str(4)
	h.skip(4) // creator and trigger count (not parsed)
	if h.err != nil {
		return nil, fmt.Errorf("artwork header: %w", h.err)
	}
	body, err := inflateRawLimited(data[h.off:], maxInflatedScenarioBytes())
	if err != nil {
		return nil, fmt.Errorf("artwork inflate: %w", err)
	}
	r := artReader{byteCursor: byteCursor{data: body}}
	a := &Artwork{Version: fmt.Sprintf("1.%02d", v), Scope: "terrain_units_only_read_only", PlayerColors: map[int]int{}}
	section := func(name string, read func()) {
		if r.err != nil {
			return
		}
		start := r.off
		read()
		if r.err != nil {
			r.err = fmt.Errorf("artwork %s at body offset %d: %w", name, r.off, r.err)
			return
		}
		a.Sections = append(a.Sections, SectionInfo{Name: name, Start: start, End: r.off})
	}
	section("DataHeader", func() {
		r.skip(4)
		version := r.f32()
		if math.Abs(version-(1+float64(v)/100)) > .00001 {
			r.fail("body version does not match header")
			return
		}
		if v >= 52 {
			r.skip(8)
		}
		r.skip(16*256 + 16*4) // fixed tribe names and string-table ids
		for i := 0; i < 16; i++ {
			r.skip(8) // active/human
			if v >= 56 {
				r.skip(2)
				r.str(2)
				r.skip(2)
				r.str(2)
			} else {
				r.skip(4)
				if v >= 40 {
					r.skip(4)
				}
			}
			r.skip(4) // city mode
		}
		r.skip(16 * 4) // civilization locks
		if v >= 53 {
			r.skip(16 * 4)
		} // personality locks
		r.skip(9)
		r.str(2)
	})
	section("Messages", func() {
		r.skip(6 * 4)
		for i := 0; i < 6; i++ {
			r.str(2)
		}
	})
	section("Cinematics", func() {
		for i := 0; i < 3; i++ {
			r.str(2)
		}
	})
	section("BackgroundImage", func() {
		_, n, err := DecodeBackgroundImageNode(body[r.off:])
		if err != nil {
			r.err = err
			return
		}
		r.skip(n)
	})
	section("PlayerDataTwo", func() {
		for i := 0; i < 48; i++ {
			r.str(2)
		}
		for i := 0; i < 16; i++ {
			r.skip(8)
			r.str(4)
		}
		r.skip(16)
		r.expect32(0xFFFFFF9D) // section separator (-99)
		for player := 1; player <= 16; player++ {
			r.skip(24)
			// Match PlayerInfo's color authority; the Units mirror can differ.
			a.PlayerColors[player] = int(int32(r.u32()))
		}
	})
	section("GlobalVictory", func() { r.skip(44) })
	section("Diplomacy", func() { r.skip(16*16*4 + 11520); r.expect32(0xFFFFFF9D); r.skip(16*4 + 4) })
	section("Options", func() {
		for list := 0; list < 3; list++ {
			var counts [16]int
			for i := range counts {
				counts[i] = r.count(maxScenarioFieldRepeat)
			}
			for _, count := range counts[:8] {
				r.skip(count * 4)
			}
		}
		r.skip(3*4 + 16*4 + 12 + 1 + 4 + 8 + 7 + 4)
	})
	section("Map", func() {
		r.skip(2)
		r.str(2)
		r.skip(2)
		a.ColorMood = r.str(2)
		if v >= 40 {
			r.skip(2)
			r.str(2)
		}
		r.skip(1)
		if v >= 37 {
			r.skip(1)
		}
		if v >= 40 {
			r.skip(16 * 8)
		}
		if v >= 41 {
			r.skip(1)
		}
		if v >= 42 && v <= 46 {
			r.skip(4)
		}
		if v >= 44 {
			r.skip(16 * 4)
		}
		if v >= 45 {
			r.skip(4)
		}
		r.expect32(0xDEADF00D)
		r.skip(4 + 1)
		a.Width = r.count(1024)
		a.Height = r.count(1024)
		if r.err != nil {
			return
		}
		count := a.Width * a.Height
		if count == 0 || count > maxScenarioFieldRepeat || count > (len(body)-r.off)/7 {
			r.fail("invalid or truncated terrain grid")
			return
		}
		a.Tiles = make([]TerrainTile, count)
		for i := range a.Tiles {
			t := &a.Tiles[i]
			t.X = i % a.Width
			t.Y = i / a.Width
			t.TerrainID = int(r.u8())
			t.Elevation = int(r.u8())
			r.skip(3)
			t.Layer = int(int16(r.u16()))
		}
	})
	section("Units", func() {
		a.Units.NumberOfUnitSections = r.count(17)
		if a.Units.NumberOfUnitSections == 0 {
			r.fail("no unit sections")
			return
		}
		r.skip(8 * 28) // resource/population mirrors
		a.Units.NumberOfPlayers = r.count(17)
		for player := 1; player <= 8; player++ {
			r.str(2)
			r.skip(8 + 4 + 1) // name, cameras, allied victory
			count := int(r.u16())
			if count > 64 {
				r.fail("diplomacy count exceeds limit")
				return
			}
			r.skip(count * 5)
			r.skip(4) // color mirror, not the PlayerInfo color authority
			victory := r.f32()
			count = int(r.u16())
			if count > 64 {
				r.fail("victory entry count exceeds limit")
				return
			}
			if victory >= 2 {
				r.skip(7)
			}
			r.skip(count * 44)
			tail := 0
			if victory >= 2 {
				tail = int(r.u8())
			}
			r.skip(7 + tail*32 + 4)
		}
		for player := 0; player < a.Units.NumberOfUnitSections; player++ {
			count := r.count(maxScenarioFieldRepeat - a.Units.Total)
			if r.err != nil {
				return
			}
			if count > (len(body)-r.off)/29 {
				r.fail("truncated unit records")
				return
			}
			p := PlayerUnitsInfo{Player: player, Count: count}
			if count > 0 {
				p.Units = make([]UnitSummary, 0, count)
			}
			for i := 0; i < count; i++ {
				u := UnitSummary{Index: i, X: r.f32(), Y: r.f32(), Z: r.f32(), ReferenceID: int(int32(r.u32())), UnitConst: int(r.u16()), Status: int(r.u8()), Rotation: r.f32(), InitialAnimationFrame: int(r.u16())}
				r.skip(4) // garrison reference
				if v >= 59 {
					r.skip(1)
				}
				if v >= 54 {
					u.CaptionStringID = int(int32(r.u32()))
				}
				if v >= 55 {
					u.CaptionString = r.str(4)
				}
				if r.err != nil {
					return
				}
				p.Units = append(p.Units, u)
			}
			a.Units.Sections = append(a.Units.Sections, p)
			a.Units.Total += count
		}
	})
	if r.err != nil {
		return nil, r.err
	}
	a.UnparsedBytes = len(body) - r.off
	return a, nil
}

// Sticky errors keep opaque skipping simple while retaining strict bounds.
type artReader struct {
	byteCursor
	err error
}

func (r *artReader) fail(s string) {
	if r.err == nil {
		r.err = fmt.Errorf("%s", s)
	}
}
func (r *artReader) take(n int) []byte {
	if r.err != nil {
		return nil
	}
	b, e := r.byteCursor.take(n)
	r.err = e
	return b
}
func (r *artReader) skip(n int) { r.take(n) }
func (r *artReader) u8() uint8 {
	b := r.take(1)
	if b == nil {
		return 0
	}
	return b[0]
}
func (r *artReader) u16() uint16 {
	b := r.take(2)
	if b == nil {
		return 0
	}
	return binary.LittleEndian.Uint16(b)
}
func (r *artReader) u32() uint32 {
	b := r.take(4)
	if b == nil {
		return 0
	}
	return binary.LittleEndian.Uint32(b)
}
func (r *artReader) f32() float64 {
	v := float64(math.Float32frombits(r.u32()))
	if math.IsNaN(v) || math.IsInf(v, 0) {
		r.fail("non-finite number")
	}
	return v
}
func (r *artReader) str(width int) string {
	if r.err != nil {
		return ""
	}
	s, e := r.byteCursor.stringN(width)
	r.err = e
	return s
}
func (r *artReader) count(limit int) int {
	n := r.u32()
	if uint64(n) > uint64(max(0, limit)) {
		r.fail(fmt.Sprintf("count %d exceeds limit %d", n, limit))
		return 0
	}
	return int(n)
}
func (r *artReader) expect32(want uint32) {
	got := r.u32()
	if got != want {
		r.fail(fmt.Sprintf("expected marker %08x, got %08x", want, got))
	}
}
