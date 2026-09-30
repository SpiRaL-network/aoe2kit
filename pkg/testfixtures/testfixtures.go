package testfixtures

import (
	"os"
	"path/filepath"
	"testing"
)

const RootEnv = "AOE2KIT_FIXTURE_ROOT"

func Path(t testing.TB, rel string) string {
	t.Helper()
	root := os.Getenv(RootEnv)
	if root == "" {
		t.Skipf("fixture root not set; set %s to run golden fixture tests", RootEnv)
	}
	path := filepath.Join(root, filepath.FromSlash(rel))
	if _, err := os.Stat(path); err != nil {
		t.Skipf("fixture %s unavailable under %s: %v", rel, root, err)
	}
	return path
}
