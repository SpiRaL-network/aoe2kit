package gfx

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"io"
)

// DecodeDDS reads the first mip of a single 2D BC1/BC2/BC3 DDS texture.
// Header facts: Microsoft DDS_HEADER and DDS_PIXELFORMAT documentation.
// Unsupported formats fail explicitly; no assets or external decoder are used.
func DecodeDDS(r io.Reader) (*image.RGBA, error) {
	// A block is only 8 or 16 bytes; avoid one file syscall per block.
	r = bufio.NewReaderSize(r, 64*1024)
	var h [128]byte
	if _, err := io.ReadFull(r, h[:]); err != nil {
		return nil, fmt.Errorf("DDS header: %w", err)
	}
	u32 := func(i int) uint32 { return binary.LittleEndian.Uint32(h[i : i+4]) }
	if string(h[:4]) != "DDS " || u32(4) != 124 || u32(76) != 32 {
		return nil, fmt.Errorf("invalid DDS header")
	}
	w, height := int(u32(16)), int(u32(12))
	if w < 1 || height < 1 || w > 4096 || height > 4096 {
		return nil, fmt.Errorf("DDS dimensions %dx%d exceed 4096 per axis", w, height)
	}
	if u32(24) > 1 || u32(112)&0x20fe00 != 0 {
		return nil, fmt.Errorf("DDS cube/volume textures unsupported")
	}
	format := string(h[84:88])
	blockBytes := 16
	if format == "DXT1" {
		blockBytes = 8
	} else if format != "DXT3" && format != "DXT5" {
		return nil, fmt.Errorf("unsupported DDS format %q (need DXT1, DXT3 or DXT5)", format)
	}
	if u32(80)&4 == 0 {
		return nil, fmt.Errorf("DDS pixel format is not FourCC")
	}
	img := image.NewRGBA(image.Rect(0, 0, w, height))
	var buf [16]byte
	for by := 0; by < height; by += 4 {
		for bx := 0; bx < w; bx += 4 {
			if _, err := io.ReadFull(r, buf[:blockBytes]); err != nil {
				return nil, fmt.Errorf("DDS block (%d,%d): %w", bx/4, by/4, err)
			}
			var block [16]color.RGBA
			if format == "DXT1" {
				block = decodeBC1Block(buf[:8])
			} else {
				// BC2/3 always use four colors, even when color0 <= color1.
				a, b := rgb565(binary.LittleEndian.Uint16(buf[8:10])), rgb565(binary.LittleEndian.Uint16(buf[10:12]))
				table := [4]color.RGBA{a, b, mixColor(a, b, 2, 1, 3, 255), mixColor(a, b, 1, 2, 3, 255)}
				indices := binary.LittleEndian.Uint32(buf[12:16])
				alpha := decodeBC4Block(buf[:8])
				for i := range block {
					c := table[(indices>>uint(2*i))&3]
					v := alpha[i]
					if format == "DXT3" {
						v = ((buf[i/2] >> uint(4*(i%2))) & 15) * 17
					}
					block[i] = color.RGBA{uint8(uint16(c.R) * uint16(v) / 255), uint8(uint16(c.G) * uint16(v) / 255), uint8(uint16(c.B) * uint16(v) / 255), v}
				}
			}
			for i, c := range block {
				if x, y := bx+i%4, by+i/4; x < w && y < height {
					img.SetRGBA(x, y, c)
				}
			}
		}
	}
	return img, nil
}
