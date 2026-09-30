package glyph

import (
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/image/font/gofont/goregular"
)

func TestFromTTFProducesCanonicalRotatableCells(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "font.ttf")
	if err := os.WriteFile(path, goregular.TTF, 0o600); err != nil {
		t.Fatal(err)
	}
	lib, err := FromTTF(path, Options{Chars: []rune("Ag8"), Material: 1366, Rotation: 90, CellScale: 8, SampleSpacing: 0.6, MaxCellsPerGlyph: 4096})
	if err != nil {
		t.Fatal(err)
	}
	if lib.Format != "aoe2kit-gaia-glyphs-v1" || len(lib.Glyphs) != 3 {
		t.Fatalf("unexpected library shape: %#v", lib)
	}
	for _, g := range lib.Glyphs {
		if len(g.Cells) == 0 || g.Width <= 0 || g.Height <= 0 {
			t.Fatalf("glyph %q was not rasterized: %#v", g.Char, g)
		}
		for _, cell := range g.Cells {
			if cell.MaterialUnitConst != 1366 || cell.Rotation != 90 {
				t.Fatalf("cell metadata was not preserved: %#v", cell)
			}
		}
	}
}
