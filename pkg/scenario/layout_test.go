package scenario

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestFirstSliceHeaderAndMapCodecMatchesParser(t *testing.T) {
	path := filepath.Join("testdata", "Clean v159.aoe2scenario")
	file, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	header, headerLen, err := DecodeScenarioHeader(file.original)
	if err != nil {
		t.Fatal(err)
	}
	if headerLen != file.HeaderBytes || header.Version != file.Version || int(header.PlayerCount) != file.PlayerCount {
		t.Fatalf("header mismatch: %#v len=%d parser=(%s,%d,%d)", header, headerLen, file.Version, file.HeaderBytes, file.PlayerCount)
	}
	if len(header.UnknownNumbers) == 0 {
		t.Fatal("header unknown-number list was not decoded")
	}
	rawMap := file.rawSection("Map")
	if len(rawMap) == 0 {
		t.Fatal("parser did not expose Map bytes")
	}
	decoded, consumed, err := DecodeMapLayout(rawMap)
	if err != nil {
		t.Fatal(err)
	}
	if consumed != len(rawMap) {
		t.Fatalf("map codec consumed %d of %d bytes", consumed, len(rawMap))
	}
	if decoded.Width != int32(file.Map.Width) || decoded.Height != int32(file.Map.Height) {
		t.Fatalf("map dimensions mismatch: codec=%dx%d parser=%dx%d", decoded.Width, decoded.Height, file.Map.Width, file.Map.Height)
	}
	if len(decoded.Terrain) != file.Map.TileCount {
		t.Fatalf("terrain count mismatch: codec=%d parser=%d", len(decoded.Terrain), file.Map.TileCount)
	}
	if !bytes.Equal(decoded.Raw, rawMap) {
		t.Fatal("map codec raw preservation differs")
	}
}

func TestLayoutDeclaresEvidenceAndNoMirrorsByAssumption(t *testing.T) {
	if len(DE158Layout.Revisions) != 1 || DE158Layout.Revisions[0].Status != "corpus_verified" {
		t.Fatalf("unexpected revisions: %#v", DE158Layout.Revisions)
	}
	for _, section := range DE158Layout.Sections {
		for _, field := range section.Fields {
			if field.Evidence.Tier == "" || field.Evidence.Source == "" {
				t.Fatalf("field %s/%s lacks evidence", section.Name, field.Name)
			}
		}
	}
	if len(DE158Layout.MirrorGroups) != 2 {
		t.Fatalf("mirror groups = %d, want player diplomacy and allied victory", len(DE158Layout.MirrorGroups))
	}
}

func TestPreservationDocumentCopiesUnknownBytesVerbatim(t *testing.T) {
	header := []byte{1, 2, 3}
	body := []byte{9, 8, 7, 6}
	doc := NewPreservedDocument(header, body)
	body[0] = 0
	header[0] = 0
	if got := doc.RebuildBody(); !bytes.Equal(got, []byte{9, 8, 7, 6}) {
		t.Fatalf("body was aliased or normalized: %v", got)
	}
	if got := doc.RebuildFile(); !bytes.Equal(got, []byte{1, 2, 3, 9, 8, 7, 6}) {
		t.Fatalf("file preservation mismatch: %v", got)
	}
	if len(doc.Opaque) != 0 {
		t.Fatal("first slice should not invent opaque spans")
	}
}

func TestFilePreservedDocumentMatchesInflatedBody(t *testing.T) {
	path := filepath.Join("testdata", "Clean v159.aoe2scenario")
	file, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	doc := file.PreservedDocument()
	if !bytes.Equal(doc.Body, file.originalBody) || !bytes.Equal(doc.Header, file.originalHeader) {
		t.Fatal("preserved document does not match parser input bytes")
	}
}

func TestSemanticPlayersDiplomacyVictoryUnitsAndMirrors(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "editor-refs", "I added the same object, 2 rotations.aoe2scenario")
	file, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	doc := file.SemanticDocument()
	if doc.Version != "1.58" {
		t.Fatalf("semantic version=%q", doc.Version)
	}
	if len(doc.Players) == 0 || len(doc.Units) == 0 {
		t.Fatalf("semantic projection lost players or units: players=%d units=%d", len(doc.Players), len(doc.Units))
	}
	if err := ValidateSemanticMirrors(doc); err != nil {
		t.Fatal(err)
	}
	if doc.Victory == nil {
		t.Fatal("semantic projection lost victory settings")
	}
}

func TestRawTriggerCodecPreservesVariableSection(t *testing.T) {
	path := filepath.Join("..", "..", "examples", "text", "Gaia Text Showcase.aoe2scenario")
	file, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	section, err := file.RawTriggerSection()
	if err != nil {
		t.Fatal(err)
	}
	if len(section.Records) != file.Triggers.Count {
		t.Fatalf("raw trigger records=%d parser count=%d", len(section.Records), file.Triggers.Count)
	}
	if !section.Unchanged() {
		t.Fatal("raw trigger codec changed untouched bytes")
	}
	first := append([]byte(nil), section.Records[0].Bytes...)
	if err := section.ReplaceRecord(0, first); err != nil {
		t.Fatal(err)
	}
	if !section.Unchanged() {
		t.Fatal("same-payload record replacement changed bytes")
	}
	first[0] ^= 1
	if err := section.ReplaceRecord(0, first); err != nil {
		t.Fatal(err)
	}
	if section.Unchanged() {
		t.Fatal("changed record was not reflected in rebuilt bytes")
	}
	if len(section.Rebuild()) != len(section.Original) {
		t.Fatal("raw trigger replacement dropped or added framing bytes")
	}
}

func TestPreservationWriterKeepsOriginalFileBytes(t *testing.T) {
	path := filepath.Join("testdata", "Clean v159.aoe2scenario")
	file, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), "preserved.aoe2scenario")
	if err := file.PreservedDocument().Write(out); err != nil {
		t.Fatal(err)
	}
	written, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(written, original) {
		t.Fatal("preservation writer changed original file bytes")
	}
}

func TestLayoutDocumentComposesAllFirstSliceLayers(t *testing.T) {
	path := filepath.Join("..", "..", "examples", "text", "Gaia Text Showcase.aoe2scenario")
	doc, err := OpenLayoutDocument(path)
	if err != nil {
		t.Fatal(err)
	}
	if doc.Semantic.Version == "" || len(doc.Semantic.Players) == 0 {
		t.Fatal("layout document lost semantic layer")
	}
	if len(doc.Raw.Body) == 0 || len(doc.Triggers.Records) == 0 {
		t.Fatal("layout document lost preservation or trigger layer")
	}
	if !doc.Triggers.Unchanged() {
		t.Fatal("layout document changed untouched trigger bytes")
	}
}
