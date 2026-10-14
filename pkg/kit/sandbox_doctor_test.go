package kit

import (
	"runtime"
	"testing"
)

func TestRunSandboxDoctor(t *testing.T) {
	report := RunSandboxDoctor()
	if !report.OK {
		t.Fatalf("sandbox doctor failed: %+v", report.Failures)
	}
	if report.Platform != runtime.GOOS+"/"+runtime.GOARCH {
		t.Fatalf("platform = %q", report.Platform)
	}
	if report.NetworkRequired {
		t.Fatal("sandbox doctor must not require network")
	}
	if len(report.WritableDirs) == 0 {
		t.Fatal("sandbox doctor did not report writable directories")
	}
	seen := map[string]bool{}
	for _, check := range report.Checks {
		seen[check.Name] = check.OK
	}
	for _, name := range []string{"blank_fixture_codec", "blank_scenario_writer"} {
		if !seen[name] {
			t.Fatalf("missing passing check %q: %+v", name, report.Checks)
		}
	}
}

func TestCurrentBuildInfo(t *testing.T) {
	oldVersion, oldCommit, oldBuildType := Version, Commit, BuildType
	Version, Commit, BuildType = "v9.8.7", "abc123", "release"
	t.Cleanup(func() { Version, Commit, BuildType = oldVersion, oldCommit, oldBuildType })
	got := CurrentBuildInfo("linux/amd64")
	if got.Version != "v9.8.7" || got.Commit != "abc123" || got.Platform != "linux/amd64" || got.BuildType != "release" {
		t.Fatalf("build info = %+v", got)
	}
}
