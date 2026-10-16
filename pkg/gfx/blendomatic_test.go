package gfx

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestDecodeBlendomatic(t *testing.T) {
	var b bytes.Buffer
	for _, v := range []uint32{1, 31, 2353} {
		if err := binary.Write(&b, binary.LittleEndian, v); err != nil {
			t.Fatal(err)
		}
	}
	b.Write(bytes.Repeat([]byte{1}, 31))
	b.Write(bytes.Repeat([]byte{0xa5}, 4*2353))
	for i := 0; i < 31; i++ {
		v := byte(128)
		if i == 1 {
			v = 64
		}
		if i == 2 {
			v = 0
		}
		b.Write(bytes.Repeat([]byte{v}, 2353))
	}
	m, err := DecodeBlendomatic(bytes.NewReader(b.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	if len(m) != 1 || m[0].Flags[30] != 1 || m[0].PackedBits[0] != 0xa5 {
		t.Fatal("framing")
	}
	for i, want := range []uint8{255, 128, 0} {
		im := m[0].Alpha[i]
		for _, xy := range [][2]int{{48, 0}, {0, 24}, {96, 24}, {48, 48}, {48, 24}} {
			if got := im.GrayAt(xy[0], xy[1]).Y; got != want {
				t.Fatalf("tile %d at %v=%d want %d", i, xy, got, want)
			}
		}
		if im.GrayAt(0, 0).Y != 0 {
			t.Fatal("diamond padding")
		}
	}
	for _, n := range []int{0, 7, 11, 42, 100, len(b.Bytes()) - 1} {
		if _, err := DecodeBlendomatic(bytes.NewReader(b.Bytes()[:n])); err == nil {
			t.Fatalf("accepted truncated length %d", n)
		}
	}
	for _, edit := range []struct {
		off   int
		value byte
	}{{0, 0}, {0, 33}, {4, 32}, {8, 1}, {12 + 31 + 4*2353, 129}} {
		bad := bytes.Clone(b.Bytes())
		bad[edit.off] = edit.value
		if _, err := DecodeBlendomatic(bytes.NewReader(bad)); err == nil {
			t.Fatalf("accepted invalid field %+v", edit)
		}
	}
	if _, err := DecodeBlendomatic(bytes.NewReader(append(bytes.Clone(b.Bytes()), 0))); err == nil {
		t.Fatal("accepted trailing bytes")
	}
}
