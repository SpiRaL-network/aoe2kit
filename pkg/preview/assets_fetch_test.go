package preview

import (
	"os"
	"path/filepath"
	"testing"

	"aoe2kit/pkg/gameinstall"
)

func fakeSLDInstall(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	g := gameinstall.GraphicsDir(root)
	if err := os.MkdirAll(g, 0755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Oak.SLD", "unused.sld"} {
		if err := os.WriteFile(filepath.Join(g, name), []byte(name), 0644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func missingSLD(path string) AssetManifest {
	return AssetManifest{Complete: true, Assets: []NeededAsset{{Path: path, Kind: "sld", Status: "missing"}}}
}

func TestFetchMissingSLDs(t *testing.T) {
	src, dest := fakeSLDInstall(t), filepath.Join(t.TempDir(), "mirror")
	m := missingSLD("resources/_common/drs/graphics/oak.sld")
	r, err := FetchMissingSLDs(m, src, dest)
	if err != nil || len(r.Copied) != 1 {
		t.Fatalf("%+v %v", r, err)
	}
	file := filepath.Join(gameinstall.GraphicsDir(dest), "oak.sld")
	data, err := os.ReadFile(file)
	if err != nil || string(data) != "Oak.SLD" {
		t.Fatalf("%q %v", data, err)
	}
	if _, err := os.Stat(filepath.Join(gameinstall.GraphicsDir(dest), "unused.sld")); !os.IsNotExist(err) {
		t.Fatal("copied unneeded asset")
	}
	if err := os.WriteFile(file, []byte("keep"), 0644); err != nil {
		t.Fatal(err)
	}
	r, err = FetchMissingSLDs(m, src, dest)
	if err != nil || len(r.Copied) != 0 || len(r.Present) != 1 {
		t.Fatalf("%+v %v", r, err)
	}
	data, _ = os.ReadFile(file)
	if string(data) != "keep" {
		t.Fatal("overwrote existing mirror asset")
	}
	r, err = FetchMissingSLDs(missingSLD("resources/_common/drs/graphics/absent.sld"), src, dest)
	if err == nil || len(r.Remaining) != 1 {
		t.Fatalf("missing source: %+v %v", r, err)
	}
}

func TestAssetFetchRejectsTraversalAndLinks(t *testing.T) {
	src := fakeSLDInstall(t)
	for _, name := range []string{"../oak.sld", "resources/_common/dat/oak.sld", "resources/_common/drs/graphics/../oak.sld", "resources/_common/drs/graphics/sub/oak.sld"} {
		if _, err := FetchMissingSLDs(missingSLD(name), src, t.TempDir()); err == nil {
			t.Fatalf("accepted %q", name)
		}
	}
	outside := filepath.Join(t.TempDir(), "secret.sld")
	if err := os.WriteFile(outside, []byte("no"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(gameinstall.GraphicsDir(src), "escape.sld")); err != nil {
		t.Skip("symlinks unavailable:", err)
	}
	if _, err := FetchMissingSLDs(missingSLD("resources/_common/drs/graphics/escape.sld"), src, t.TempDir()); err == nil {
		t.Fatal("source link escaped graphics")
	}
	dest := t.TempDir()
	if err := os.Symlink(t.TempDir(), filepath.Join(dest, "resources")); err != nil {
		t.Fatal(err)
	}
	if _, err := FetchMissingSLDs(missingSLD("resources/_common/drs/graphics/oak.sld"), src, dest); err == nil {
		t.Fatal("destination link accepted")
	}
}

func TestStrictSpriteCompleteness(t *testing.T) {
	if err := CheckSpriteCompleteness(Report{Skipped: []Skipped{{Reason: "outside crop"}, {Reason: "particle: outside crop"}, {Reason: "objects layer disabled"}}}); err != nil {
		t.Fatal(err)
	}
	for _, reason := range []string{"missing sprite: no loose file", "SLD decode: truncated", "standing graphic absent from DAT", "visual variant 2 unavailable", "particle: no definition"} {
		if err := CheckSpriteCompleteness(Report{Skipped: []Skipped{{UnitID: 1, Reason: reason}}}); err == nil {
			t.Fatalf("accepted %s", reason)
		}
	}
}

func TestStrictOutsideCropParticleDoesNotHideFailures(t *testing.T) {
	r := Report{Skipped: []Skipped{{UnitID: 896, Reason: "particle: outside crop"}}}
	if err := CheckSpriteCompleteness(r); err != nil {
		t.Fatal(err)
	}
	r.Skipped = append(r.Skipped, Skipped{UnitID: 1635, Reason: "particle: missing atlas"})
	if err := CheckSpriteCompleteness(r); err == nil {
		t.Fatal("outside-crop exclusion hid a real particle failure")
	}
}
