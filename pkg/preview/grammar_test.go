package preview

import (
	"encoding/json"
	"image/color"
	"os"
	"path/filepath"
	"testing"

	"aoe2kit/pkg/felt"
)

func TestGrammarSheetAndEditableSkeleton(t *testing.T) {
	root, idx := spriteFixture(t)
	idx.Graphics[0].AngleCount = 1
	idx.Graphics[0].FrameCount = 1
	img, r, err := RenderGrammar(GrammarOptions{UnitID: 7, Dat: idx, GameDir: root})
	if err != nil {
		t.Fatal(err)
	}
	if img.Bounds().Dx() != 1000 || len(r.Entries) != 1 || len(r.Entries[0].Grammar) != 1 || r.Entries[0].Grammar[0].Evidence != "unmapped" || len(r.Items) != 1 || r.Items[0].Frame != 0 || r.Items[0].Status != "drawn" {
		t.Fatalf("bad report: %+v", r)
	}
	if !imageHasColor(img, color.RGBA{R: 255, A: 255}) {
		t.Fatal("sprite absent")
	}
	path := filepath.Join(t.TempDir(), "editable.json")
	b, _ := json.Marshal(r)
	if err := os.WriteFile(path, b, 0600); err != nil {
		t.Fatal(err)
	}
	c, err := felt.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if e, ok := c.Find(7); !ok || len(e.Grammar) != 1 {
		t.Fatalf("skeleton not reusable: %+v", c)
	}
}
