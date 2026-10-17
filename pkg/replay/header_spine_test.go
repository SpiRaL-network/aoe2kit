package replay

import (
	"bytes"
	"encoding/binary"
	"math"
	"os"
	"testing"
)

func TestSeekDEPlayerMarkerAcceptsCurrentTeamIDs(t *testing.T) {
	data := make([]byte, 80)
	c := &cursor{data: data, off: 16}
	marker := 32
	binary.LittleEndian.PutUint32(data[marker-8:], 15)
	binary.LittleEndian.PutUint32(data[marker-4:], 5)
	data[marker] = 0xff
	data[marker+1] = 6
	data[marker+2] = 3

	got, err := seekDEPlayerMarker(c, 0)
	if err != nil {
		t.Fatal(err)
	}
	if got != marker {
		t.Fatalf("marker = %d, want %d", got, marker)
	}
}

func TestSeekDEPlayerMarkerMissingDoesNotPoisonCursor(t *testing.T) {
	c := &cursor{data: make([]byte, 64), off: 12}
	if _, err := seekDEPlayerMarker(c, 2); err == nil {
		t.Fatal("missing marker returned no error")
	}
	if c.err != nil {
		t.Fatalf("optional-slot search mutated cursor error: %v", c.err)
	}
}

func TestParseInitialPlayerAttributesSupportsLegacyAndCurrentStartCoordinates(t *testing.T) {
	t.Run("legacy", func(t *testing.T) {
		header := makeInitialPlayerAttributePayload(t, false)
		attrs, err := parseInitialPlayerAttributes(header, 0, len(header), 198)
		if err != nil {
			t.Fatal(err)
		}
		if attrs.StartX != 0 || attrs.StartY != 0 {
			t.Fatalf("legacy start floats = %.2f,%.2f, want zero", attrs.StartX, attrs.StartY)
		}
		assertInitialPlayerAttributes(t, attrs, "BENGALIS-CIV", 4, 15)
	})

	t.Run("current save 68", func(t *testing.T) {
		header := makeInitialPlayerAttributePayload(t, true)
		attrs, err := parseInitialPlayerAttributes(header, 0, len(header), 198)
		if err != nil {
			t.Fatal(err)
		}
		if attrs.StartX != 4.5 || attrs.StartY != 15.5 {
			t.Fatalf("current start floats = %.2f,%.2f, want 4.50,15.50", attrs.StartX, attrs.StartY)
		}
		assertInitialPlayerAttributes(t, attrs, "BENGALIS-CIV", 4, 15)
		if attrBytes := attrs.End - attrs.Start; attrBytes <= 0 {
			t.Fatalf("attributes bytes = %d, want positive", attrBytes)
		}
	})
}

func makeInitialPlayerAttributePayload(t *testing.T, includeStartFloats bool) []byte {
	t.Helper()
	buf := make([]byte, 198*2*4)
	buf = append(buf, 0x0b)
	appendF32 := func(value float32) {
		var raw [4]byte
		binary.LittleEndian.PutUint32(raw[:], math.Float32bits(value))
		buf = append(buf, raw[:]...)
	}
	appendU32 := func(value uint32) {
		var raw [4]byte
		binary.LittleEndian.PutUint32(raw[:], value)
		buf = append(buf, raw[:]...)
	}
	appendU16 := func(value uint16) {
		var raw [2]byte
		binary.LittleEndian.PutUint16(raw[:], value)
		buf = append(buf, raw[:]...)
	}

	appendF32(40)
	appendF32(47)
	appendU32(1)
	if includeStartFloats {
		appendF32(4.5)
		appendF32(15.5)
	}
	appendU16(4)
	appendU16(15)
	buf = append(buf, 4)
	buf = append(buf, 0x60, 0x0a)
	appendU16(uint16(len("BENGALIS-CIV")))
	buf = append(buf, []byte("BENGALIS-CIV")...)
	buf = append(buf, 0, 0, 0, 0, 0, 0)
	return buf
}

func assertInitialPlayerAttributes(t *testing.T, attrs initialPlayerAttributes, civ string, spawnX, spawnY uint16) {
	t.Helper()
	if attrs.CameraX != 40 || attrs.CameraY != 47 {
		t.Fatalf("camera = %.2f,%.2f, want 40,47", attrs.CameraX, attrs.CameraY)
	}
	if attrs.PostCameraUnknown != 1 {
		t.Fatalf("post-camera unknown = %d, want 1", attrs.PostCameraUnknown)
	}
	if attrs.SpawnX != spawnX || attrs.SpawnY != spawnY {
		t.Fatalf("spawn = %d,%d, want %d,%d", attrs.SpawnX, attrs.SpawnY, spawnX, spawnY)
	}
	if attrs.StartMetaByte != 4 {
		t.Fatalf("start meta = %d, want 4", attrs.StartMetaByte)
	}
	if attrs.CivilizationKey != civ {
		t.Fatalf("civilization key = %q, want %q", attrs.CivilizationKey, civ)
	}
	if attrs.End != len(makeInitialPlayerAttributePayload(t, attrs.StartX != 0 || attrs.StartY != 0)) {
		t.Fatalf("end = %d, want payload length", attrs.End)
	}
}

func TestCurrentFormatReplaySmokeOptional(t *testing.T) {
	path := os.Getenv("AOE2KIT_CURRENT_REPLAY_FIXTURE")
	if path == "" {
		t.Skip("set AOE2KIT_CURRENT_REPLAY_FIXTURE to a current VER 9.4/save_version 68 replay")
	}
	rec, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if rec.GameVersion != "VER 9.4" || rec.SaveVersion != 68 {
		t.Fatalf("version = %s/%.2f, want VER 9.4/68", rec.GameVersion, rec.SaveVersion)
	}
	if len(rec.Players) == 0 {
		t.Fatalf("players missing; ai_parse_error=%q data_set_error=%q", rec.AIParseErr, rec.DataSet.Error)
	}
	if rec.DataSet.Error != "" {
		t.Fatalf("data-set identity error = %q", rec.DataSet.Error)
	}
	coverage, err := BuildCoverage(path)
	if err != nil {
		t.Fatal(err)
	}
	if coverage.HeaderSpine == nil {
		t.Fatalf("header spine missing; warnings=%v", coverage.Warnings)
	}
	if len(coverage.HeaderSpine.Players) == 0 {
		t.Fatalf("header spine players missing")
	}
	sync, err := BuildSyncStream(path, SyncOptions{ChecksumsOnly: true, Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if sync.Summary.SyncCount == 0 || sync.Summary.ChecksumDE == 0 {
		t.Fatalf("sync summary = %+v, want sync and DE checksum samples", sync.Summary)
	}
	datamod, err := BuildEffectiveDataModCheck(path, EffectiveDataModCheckOptions{Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if datamod.DataSet.Status == "" || datamod.DataSet.Error != "" {
		t.Fatalf("datamod data-set identity = %+v, want parsed status without error", datamod.DataSet)
	}
	if datamod.TargetTemplate.TailBytes == 0 {
		t.Fatalf("datamod target template missing")
	}
}

// deTestLobbyPrefix builds the fields readDELobbySettingsPrefix reads, with the
// given player count and no DLCs.
func deTestLobbyPrefix(numPlayers uint32) []byte {
	var b bytes.Buffer
	u32 := func(v uint32) { binary.Write(&b, binary.LittleEndian, v) }
	for i := 0; i < 6; i++ { // build, timestamp, three raw words, DLC count 0
		u32(0)
	}
	for i := 0; i < 15; i++ { // game type ... population limit
		u32(0)
	}
	u32(numPlayers)
	b.Write(make([]byte, 14+4+10)) // pre-difficulty, four flags, team settings
	for i := 0; i < 3; i++ {
		u32(0)
	}
	b.Write([]byte{0, 0}) // diplomacy type, ranked
	return b.Bytes()
}

func deTestString(b *bytes.Buffer, s string) {
	b.Write([]byte{0x60, 0x0a})
	binary.Write(b, binary.LittleEndian, uint16(len(s)))
	b.WriteString(s)
}

// deTestPlayerRecord builds one lobby slot record as readDEPlayers walks it.
func deTestPlayerRecord(color int32, name string, playerType uint32) []byte {
	var b bytes.Buffer
	u32 := func(v uint32) { binary.Write(&b, binary.LittleEndian, v) }
	u32(15) // DLC id
	binary.Write(&b, binary.LittleEndian, color)
	b.Write([]byte{0xff, 1, 1}) // marker, selected and resolved team
	b.Write(make([]byte, 8+1))  // dat CRC, MP game version
	u32(1)                      // civ
	u32(0)                      // custom civ count
	deTestString(&b, "")        // data string
	b.WriteByte(0)
	deTestString(&b, "") // AI name
	deTestString(&b, "") // censored name
	deTestString(&b, name)
	u32(playerType)
	u32(0) // profile id
	u32(0)
	u32(uint32(color + 1)) // player number
	b.Write(make([]byte, 1+1+8+4))
	return b.Bytes()
}

// DE writes all eight lobby slot records even when fewer players are in the
// game; the AI-file section starts only after the eighth record.
func TestReadDEPlayersConsumesAllEightLobbySlots(t *testing.T) {
	var header bytes.Buffer
	header.Write(deTestLobbyPrefix(2))
	header.Write(deTestPlayerRecord(0, "Alpha", 2))
	header.Write(deTestPlayerRecord(1, "Bravo", 2))
	for slot := 2; slot < 8; slot++ {
		header.Write(deTestPlayerRecord(-1, "", 1))
	}
	end := header.Len()
	header.Write([]byte("AFTER-PLAYERS"))

	c := &cursor{data: header.Bytes()}
	players, err := readDEPlayers(c, 68.9)
	if err != nil {
		t.Fatal(err)
	}
	if len(players) != 2 || players[0].Name != "Alpha" || players[1].Name != "Bravo" {
		t.Fatalf("players = %+v, want Alpha and Bravo only", players)
	}
	if c.off != end {
		t.Fatalf("cursor at %d after the players, want %d (end of the eighth slot)", c.off, end)
	}
}
