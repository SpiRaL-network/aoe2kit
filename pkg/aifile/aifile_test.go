package aifile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAIReferenceSignatureIsOrderStable(t *testing.T) {
	a := ParseReferences([]string{
		"LOCALMODS:AI:SDSNarrator.per:Decima:false",
		"LOCALMODS:AI:SDSNarrator.ai:Decima:false",
	})
	b := ParseReferences([]string{
		"LOCALMODS:AI:SDSNarrator.ai:Decima:false",
		"LOCALMODS:AI:SDSNarrator.per:Decima:false",
	})
	if Signature(a) != Signature(b) {
		t.Fatalf("signature changed with order: %s != %s", Signature(a), Signature(b))
	}
	loadout := NewLoadout(a)
	if loadout.Empty {
		t.Fatal("loadout unexpectedly empty")
	}
	if len(loadout.Groups) != 1 {
		t.Fatalf("groups=%d want 1", len(loadout.Groups))
	}
	if loadout.Groups[0].Name != "SDSNarrator" || loadout.Groups[0].SourceMod != "Decima" {
		t.Fatalf("bad group: %+v", loadout.Groups[0])
	}
}

func TestAIContentFingerprintNormalizesLineEndings(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Test.per"), []byte("(defrule)\r\n"), 0644); err != nil {
		t.Fatal(err)
	}
	fpCRLF, err := FingerprintPath(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "Test.per"), []byte("(defrule)\n"), 0644); err != nil {
		t.Fatal(err)
	}
	fpLF, err := FingerprintPath(dir)
	if err != nil {
		t.Fatal(err)
	}
	if fpCRLF.SHA256 != fpLF.SHA256 {
		t.Fatalf("line ending normalization failed: %s != %s", fpCRLF.SHA256, fpLF.SHA256)
	}
}
