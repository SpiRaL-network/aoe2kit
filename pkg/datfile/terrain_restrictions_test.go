package datfile

import (
	"encoding/binary"
	"math"
	"os"
	"path/filepath"
	"testing"
)

func TestTerrainRestrictionArraysAndSpans(t *testing.T) {
	data := make([]byte, 2*3*20)
	for group := 0; group < 2; group++ {
		for terrain := 0; terrain < 3; terrain++ {
			binary.LittleEndian.PutUint32(data[group*60+terrain*4:], math.Float32bits(float32(group*10+terrain)+0.5))
			for b := 0; b < 16; b++ {
				data[group*60+12+terrain*16+b] = 0xff
			}
		}
	}
	c := cursor{data: data}
	rows, err := parseTerrainRestrictions(&c, 2, 3)
	if err != nil {
		t.Fatal(err)
	}
	if c.off != len(data) {
		t.Fatal("incorrect consumed bytes")
	}
	for group, r := range rows {
		for terrain, row := range r.TerrainRows {
			if row.Passability != float32(group*10+terrain)+0.5 {
				t.Fatalf("multiplier read from graphics: %+v", row)
			}
			if row.PassabilitySpan.Start != group*60+terrain*4 || row.PassGraphicSpan.Start != group*60+12+terrain*16 || row.PassabilitySpan.Len() != 4 || row.PassGraphicSpan.Len() != 16 {
				t.Fatalf("bad spans %+v", row)
			}
		}
	}
	prefix := make([]byte, 12+2*8)
	copy(prefix, "VER 8.9")
	binary.LittleEndian.PutUint16(prefix[8:], 2)
	binary.LittleEndian.PutUint16(prefix[10:], 3)
	compressed, err := Deflate(append(prefix, data...))
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "synthetic.dat")
	if err := os.WriteFile(path, compressed, 0600); err != nil {
		t.Fatal(err)
	}
	version, got, err := ReadTerrainRestrictions(path)
	if err != nil {
		t.Fatal(err)
	}
	if version != "VER 8.9" || got[1].TerrainRows[2].Passability != 12.5 {
		t.Fatalf("wrong streamed prefix %s %+v", version, got)
	}
}
