package gfx

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func syntheticDDS(format string, w, h uint32, payload []byte) []byte {
	b := make([]byte, 128)
	copy(b, "DDS ")
	for off, v := range map[int]uint32{4: 124, 12: h, 16: w, 76: 32, 80: 4} {
		binary.LittleEndian.PutUint32(b[off:], v)
	}
	copy(b[84:], format)
	return append(b, payload...)
}

func TestDDSBlocks(t *testing.T) {
	for _, format := range []string{"DXT1", "DXT3", "DXT5"} {
		p := make([]byte, 16)
		if format == "DXT1" {
			p = p[:8]
			p[1] = 0xf8
		} else {
			// c0 < c1, index 3 must remain a color, not transparent BC1.
			p[10], p[11] = 0xff, 0xff
			for i := 12; i < 16; i++ {
				p[i] = 0xff
			}
			if format == "DXT3" {
				for i := 0; i < 8; i++ {
					p[i] = 0x88
				}
			} else {
				p[0] = 128
			}
		}
		img, err := DecodeDDS(bytes.NewReader(syntheticDDS(format, 3, 2, p)))
		if err != nil {
			t.Fatal(err)
		}
		c := img.RGBAAt(2, 1)
		if format == "DXT1" {
			if c.R != 255 || c.A != 255 {
				t.Fatal(c)
			}
		} else {
			want := uint8(128)
			if format == "DXT3" {
				want = 136
			}
			if c.A != want || c.R != uint8(170*uint16(want)/255) {
				t.Fatal(format, c)
			}
		}
	}
}

func TestDDSRejectsMalformed(t *testing.T) {
	for _, b := range [][]byte{nil, syntheticDDS("DXT1", 0, 1, nil), syntheticDDS("DXT1", 4097, 1, nil), syntheticDDS("DX10", 1, 1, nil), syntheticDDS("DXT1", 4, 4, make([]byte, 7))} {
		if _, err := DecodeDDS(bytes.NewReader(b)); err == nil {
			t.Fatal("accepted malformed DDS")
		}
	}
}
