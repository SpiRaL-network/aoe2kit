package scenario

import (
	"path/filepath"
	"runtime"
)

func scenarioProjectPath(parts ...string) string {
	_, source, _, _ := runtime.Caller(0)
	repo := filepath.Clean(filepath.Join(filepath.Dir(source), "..", ".."))
	args := append([]string{repo, "..", ".."}, parts...)
	return filepath.Join(args...)
}

func scenarioAoe2DEPath(parts ...string) string {
	_, source, _, _ := runtime.Caller(0)
	repo := filepath.Clean(filepath.Join(filepath.Dir(source), "..", ".."))
	args := append([]string{repo, ".."}, parts...)
	return filepath.Join(args...)
}
