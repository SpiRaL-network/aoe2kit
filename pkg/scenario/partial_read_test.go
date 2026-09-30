package scenario

import "testing"

func TestMapFileReadsStablePrefixWithoutUnits(t *testing.T) {
	path := t.TempDir() + "/map.aoe2scenario"
	if _, err := WriteBlankScenarioFile(path, BlankOptions{MapWidth: 80, MapHeight: 80, NoStarters: true}); err != nil {
		t.Fatalf("WriteBlankScenarioFile: %v", err)
	}
	mapInfo, err := MapFile(path)
	if err != nil {
		t.Fatalf("MapFile: %v", err)
	}
	if mapInfo.Width != 80 || mapInfo.Height != 80 || mapInfo.TileCount != 80*80 {
		t.Fatalf("map = %+v, want 80x80/%d", mapInfo, 80*80)
	}
}
