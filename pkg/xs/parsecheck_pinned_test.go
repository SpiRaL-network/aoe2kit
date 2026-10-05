package xs

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestPinnedXSCheckNameAndDigestFollowTheOS(t *testing.T) {
	name, digest := pinnedXSCheck("windows")
	if name != "xs-check.exe" || digest != pinnedXSCheckWindowsSHA256 {
		t.Errorf("windows: got %q %q", name, digest)
	}
	for _, goos := range []string{"linux", "darwin"} {
		name, digest := pinnedXSCheck(goos)
		if name != "xs-check" || digest != pinnedXSCheckSHA256 {
			t.Errorf("%s: got %q %q", goos, name, digest)
		}
	}
}

// The user-local pinned install is looked up under this OS's file name and
// checked against this OS's digest.
func TestResolvePinnedParserChecksTheUserLocalInstallForThisOS(t *testing.T) {
	dataHome := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataHome)
	name, want := pinnedXSCheck(runtime.GOOS)
	path := filepath.Join(dataHome, "aoe2kit", name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("not the release binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	_, err := resolvePinnedParser()
	if err == nil {
		t.Fatalf("a %s with the wrong digest was accepted", name)
	}
	if !strings.Contains(err.Error(), name) || !strings.Contains(err.Error(), want) {
		t.Fatalf("error should name %s and the expected digest %s: %v", name, want, err)
	}
}
