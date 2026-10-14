package datfile

import (
	"bytes"
	"compress/flate"
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

// ReadTerrainRestrictions reads the DE DAT prefix without decoding unit or
// graphic tables. The decompressed prefix is bounded to 8 MiB.
func ReadTerrainRestrictions(path string) (string, []TerrainRestriction, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", nil, err
	}
	defer f.Close()
	r := flate.NewReader(f)
	defer r.Close()
	header := make([]byte, 12)
	if _, err := io.ReadFull(r, header); err != nil {
		return "", nil, err
	}
	version := string(bytes.TrimRight(header[:8], "\x00"))
	if version != "VER 8.8" && version != "VER 8.9" {
		return version, nil, fmt.Errorf("terrain restriction prefix unsupported for %q", version)
	}
	n := int(int16(binary.LittleEndian.Uint16(header[8:])))
	m := int(int16(binary.LittleEndian.Uint16(header[10:])))
	if n < 0 || m < 0 {
		return "", nil, fmt.Errorf("negative terrain restriction counts")
	}
	size := int64(n)*8 + int64(n)*int64(m)*20
	if size > 8<<20 {
		return "", nil, fmt.Errorf("terrain restriction prefix exceeds 8 MiB")
	}
	payload := make([]byte, 12+int(size))
	copy(payload, header)
	if _, err := io.ReadFull(r, payload[12:]); err != nil {
		return "", nil, err
	}
	c := cursor{data: payload, off: 12 + n*8}
	rows, err := parseTerrainRestrictions(&c, n, m)
	return version, rows, err
}
