package gfx

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"image"
	"io"
)

// BlendomaticMode holds the installed file's flags, packed bit masks and alpha
// diamonds. Alpha stores normalized file weights (0..128 becomes 0..255),
// not necessarily incoming-terrain opacity; the renderer chooses polarity.
// PackedBits is preserved without asserting how DE uses it alongside alpha.
type BlendomaticMode struct {
	Flags      [31]byte
	PackedBits []byte
	Alpha      [31]*image.Gray
}

// DecodeBlendomatic supports the measured 97x49, 2353-pixel diamond layout.
// Format and directional-mask facts are documented by openage (see NOTICE).
// Unknown layouts, truncated records and trailing data are rejected.
func DecodeBlendomatic(r io.Reader) ([]BlendomaticMode, error) {
	r = bufio.NewReader(r)
	var h [8]byte
	if _, err := io.ReadFull(r, h[:]); err != nil {
		return nil, fmt.Errorf("blendomatic header: %w", err)
	}
	n, tiles := binary.LittleEndian.Uint32(h[:4]), binary.LittleEndian.Uint32(h[4:])
	if n < 1 || n > 32 || tiles != 31 {
		return nil, fmt.Errorf("unsupported blendomatic dimensions: %d modes, %d tiles", n, tiles)
	}
	modes := make([]BlendomaticMode, n)
	for i := range modes {
		var size uint32
		if err := binary.Read(r, binary.LittleEndian, &size); err != nil {
			return nil, fmt.Errorf("blendomatic mode %d size: %w", i, err)
		}
		if size != 2353 {
			return nil, fmt.Errorf("unsupported blendomatic mode %d diamond size %d", i, size)
		}
		m := &modes[i]
		if _, err := io.ReadFull(r, m.Flags[:]); err != nil {
			return nil, fmt.Errorf("blendomatic mode %d flags: %w", i, err)
		}
		m.PackedBits = make([]byte, 4*size)
		if _, err := io.ReadFull(r, m.PackedBits); err != nil {
			return nil, fmt.Errorf("blendomatic mode %d bits: %w", i, err)
		}
		var packed [2353]byte
		for j := range m.Alpha {
			if _, err := io.ReadFull(r, packed[:]); err != nil {
				return nil, fmt.Errorf("blendomatic mode %d tile %d: %w", i, j, err)
			}
			a := image.NewGray(image.Rect(0, 0, 97, 49))
			p := 0
			for y := 0; y < 49; y++ {
				width := 1 + 4*min(y, 48-y)
				for x := (97 - width) / 2; x < (97+width)/2; x++ {
					v := packed[p]
					if v > 128 {
						return nil, fmt.Errorf("blendomatic mode %d tile %d alpha %d exceeds 128", i, j, v)
					}
					a.Pix[y*a.Stride+x] = uint8((uint16(v)*255 + 64) / 128)
					p++
				}
			}
			m.Alpha[j] = a
		}
	}
	var tail [1]byte
	if _, err := io.ReadFull(r, tail[:]); err != io.EOF {
		if err == nil {
			err = fmt.Errorf("trailing bytes")
		}
		return nil, fmt.Errorf("blendomatic end: %w", err)
	}
	return modes, nil
}
