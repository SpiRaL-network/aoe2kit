package preview

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProjectDefaults(t *testing.T) {
	root := t.TempDir()
	child := filepath.Join(root, "scenes")
	if err := os.Mkdir(child, 0755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "kit-preview.json")
	if err := os.WriteFile(path, []byte(`{"dat":"assets/game.dat","game_dir":"assets","sprites":true,"shadows":true,"mood":"none","scale":60}`), 0644); err != nil {
		t.Fatal(err)
	}
	d, err := LoadProjectDefaults(child)
	if err != nil || d.Dat != filepath.Join(root, "assets", "game.dat") || d.GameDir != filepath.Join(root, "assets") || !d.Sprites || !d.Shadows || d.Mood != "none" || d.Scale != 60 {
		t.Fatalf("%+v: %v", d, err)
	}
	for _, text := range []string{`{"unknown":true}`, `{} {}`, `{"sprites":"yes"}`, `{"scale":129}`, `{"scale":-1}`} {
		if err := os.WriteFile(path, []byte(text), 0644); err != nil {
			t.Fatal(err)
		}
		if _, err := LoadProjectDefaults(child); err == nil {
			t.Fatalf("accepted %s", text)
		}
	}
}
