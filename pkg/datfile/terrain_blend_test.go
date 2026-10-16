package datfile

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestTerrainBlendWordsDoNotMoveRecordBoundary(t *testing.T) {
	var b bytes.Buffer
	b.Write(make([]byte, 8))
	str := func(s string) {
		binary.Write(&b, binary.LittleEndian, uint16(DebugStringMarker))
		binary.Write(&b, binary.LittleEndian, uint16(len(s)))
		b.WriteString(s)
	}
	str("Synthetic")
	str("texture")
	for _, v := range []int32{-1, 12, 13, 14, 15, 145, 5} {
		binary.Write(&b, binary.LittleEndian, v)
	}
	str("road.png")
	b.Write([]byte{1, 2, 3, 4, 5})
	b.Write(make([]byte, 25+tileTypeCount*6+2+4+terrainUnitsSize*7+2+2))
	c := cursor{data: b.Bytes()}
	terrain, err := parseTerrain(&c, 0)
	if err != nil {
		t.Fatal(err)
	}
	if c.off != b.Len() || terrain.BlendPriority != 145 || terrain.BlendType != 5 || terrain.OverlayMaskName.Value != "road.png" || terrain.MinimapColorMedium != 2 {
		t.Fatalf("record drift %+v at %d/%d", terrain, c.off, b.Len())
	}
	for n := 8; n < b.Len(); n++ {
		c := cursor{data: b.Bytes()[:n]}
		if _, err := parseTerrain(&c, 0); err == nil {
			t.Fatalf("accepted truncation %d", n)
		}
	}
}
