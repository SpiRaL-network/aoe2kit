package gameinstall

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveOrder(t *testing.T) {
	base := t.TempDir()
	explicit := filepath.Join(base, "explicit")
	envDir := filepath.Join(base, "env")
	discovered := filepath.Join(base, "discovered")
	for _, dir := range []string{explicit, envDir, discovered} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv(EnvDir, envDir)
	got, err := Resolve(explicit)
	if err != nil || got != explicit {
		t.Fatalf("explicit resolution = %q, %v", got, err)
	}
	got, err = Resolve("")
	if err != nil || got != envDir {
		t.Fatalf("environment resolution = %q, %v", got, err)
	}
}

func TestFindLooseSLDByDATBasename(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "resources", "_common", "drs", "graphics", "Oak.sld")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("synthetic"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := FindLooseSLD(dir, "oak.slp")
	if err != nil || got != path {
		t.Fatalf("FindLooseSLD = %q, %v", got, err)
	}
}

func TestTerrainAssetLookup(t *testing.T) {
	root := t.TempDir()
	if got := BlendomaticPath(root); got != filepath.Join(root, "resources", "_common", "dat", "blendomatic_x1.dat") {
		t.Fatal(got)
	}
	dir := filepath.Join(root, "resources", "_common", "terrain", "masks")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "Grass.PNG")
	if err := os.WriteFile(path, []byte("synthetic"), 0600); err != nil {
		t.Fatal(err)
	}
	if got, err := TerrainAssetPath(root, "masks", "grass.png"); err != nil || got != path {
		t.Fatal(got, err)
	}
	for _, name := range []string{"../grass.png", `..\grass.png`, "C:grass.png", "", ".."} {
		if _, err := TerrainAssetPath(root, "masks", name); err == nil {
			t.Fatal("unsafe name accepted", name)
		}
	}
	if _, err := TerrainAssetPath(root, "../masks", "grass.png"); err == nil {
		t.Fatal("unsafe group accepted")
	}
}

func TestInspectNoInstallIsNonFatal(t *testing.T) {
	t.Setenv(EnvDir, filepath.Join(t.TempDir(), "missing"))
	got := Inspect("")
	if got.Found || got.SpritesReadable || got.Message != "no install; flat previews only" {
		t.Fatalf("status = %+v", got)
	}
}

func TestInspectDoesNotCallInvalidLooseSLDReadable(t *testing.T) {
	dir := t.TempDir()
	drs := filepath.Join(dir, "resources", "_common", "drs")
	if err := os.MkdirAll(filepath.Join(drs, "graphics"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(drs, "graphics.drs"), []byte("archive"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(drs, "graphics", "bad.sld"), []byte("not an SLD"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := Inspect(dir)
	if !got.Found || got.SpriteAssetCount != 1 || got.SpritesReadable {
		t.Fatalf("status = %+v, want install found and unreadable SLD", got)
	}
}
