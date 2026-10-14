package gamestrings

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestLoadQuotedTable(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, tableFileName)
	if err := os.WriteFile(path, []byte("// header\n10500 \"[None]\" // comment\n15523 \"Agoge Hoplite Aura Enabled\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	table, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got, ok := table.Lookup(15523); !ok || got != "Agoge Hoplite Aura Enabled" {
		t.Fatalf("lookup = %q, %v", got, ok)
	}
	if _, ok := table.Lookup(1); ok {
		t.Fatal("unexpected missing key")
	}
}

func TestFindGameDirRecognizesAoE2DEFolder(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("this test exercises the Steam-on-Linux candidate list")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	gameDir := filepath.Join(home, ".steam", "steam", "steamapps", "common", "AoE2DE")
	if err := os.MkdirAll(filepath.Join(gameDir, "resources"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(gameDir, "resources", tableFileName), []byte("1 \"test\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := FindGameDir()
	if err != nil {
		t.Fatalf("FindGameDir: %v", err)
	}
	if got != gameDir {
		t.Fatalf("FindGameDir=%q, want %q", got, gameDir)
	}
}
