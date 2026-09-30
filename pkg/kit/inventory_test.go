package kit

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanFindsScenarioDatAndMod(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "sample.aoe2scenario"), "scenario")
	mustWrite(t, filepath.Join(root, "data", "empires2_x2_p1.dat"), "dat")
	mustWrite(t, filepath.Join(root, "MyMod", "info.json"), `{"Title":"My Mod"}`)
	mustWrite(t, filepath.Join(root, "MyMod", "resources", "_common", "ai", "bot.per2"), "(defrule)")

	inv, err := Scan(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(inv.ModDirs) != 1 {
		t.Fatalf("expected 1 mod dir, got %d", len(inv.ModDirs))
	}
	kinds := map[string]int{}
	for _, file := range inv.Files {
		kinds[file.Kind]++
	}
	if kinds["scenario"] != 1 || kinds["dat"] != 1 {
		t.Fatalf("unexpected kinds: %#v", kinds)
	}
}

func mustWrite(t *testing.T, path, data string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
}
