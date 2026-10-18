package kit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPortableIdentityLeaksInText(t *testing.T) {
	name := "Ry" + "an"
	leaks := portableIdentityLeaksInText("data/engine_facts.json", []byte("safe B"+name+"\nleak "+name+" here\n"+name+"ic safe\n"))
	if len(leaks) != 1 {
		t.Fatalf("identity leaks = %+v, want one exact target-name word", leaks)
	}
	if leaks[0] != "data/engine_facts.json:2: "+name {
		t.Fatalf("leak detail = %q", leaks[0])
	}
}

func TestPortableIdentityLeaksUsesHandoffBoundary(t *testing.T) {
	root := t.TempDir()
	name := "Ry" + "an"
	mustWritePortable(t, filepath.Join(root, "README.md"), "portable root\n")
	mustWritePortable(t, filepath.Join(root, "data", "engine_facts.json"), `{"statement":"`+name+` leaked here"}`)
	mustWritePortable(t, filepath.Join(root, "DEX_TASK_internal.md"), name+" is allowed in task notes\n")
	mustWritePortable(t, filepath.Join(root, "docs", "internal.md"), name+" is allowed in reduced-profile docs exclusion\n")
	mustWritePortable(t, filepath.Join(root, "LICENSE"), "Copyright "+name+"\n")

	leaks, err := portableIdentityLeaks(root)
	if err != nil {
		t.Fatalf("portableIdentityLeaks failed: %v", err)
	}
	if len(leaks) != 1 {
		t.Fatalf("identity leaks = %+v, want only data leak", leaks)
	}
	if leaks[0] != "data/engine_facts.json:1: "+name {
		t.Fatalf("leak detail = %q", leaks[0])
	}
}

func TestPortableIdentityLeaksUsesSelectedPublicBoundary(t *testing.T) {
	root := t.TempDir()
	name := "Ry" + "an"
	mustWritePortable(t, filepath.Join(root, "README.md"), "portable root\n")
	mustWritePortable(t, filepath.Join(root, "NOTICE.md"), name+" in public notice\n")
	mustWritePortable(t, filepath.Join(root, "docs", "REPLAY_TRANSPARENCY_INTERNAL.md"), name+" excluded diagnostic\n")
	writePublicAllowlistForTest(t, root, "README.md", "NOTICE.md")

	leaks, err := portableIdentityLeaksForProfile(root, ProfilePublic)
	if err != nil {
		t.Fatalf("portableIdentityLeaksForProfile failed: %v", err)
	}
	if len(leaks) != 1 || leaks[0] != "NOTICE.md:1: "+name {
		t.Fatalf("public identity leaks = %+v, want only NOTICE leak", leaks)
	}
}

func TestPortableLocalPathLeaksUseHandoffBoundary(t *testing.T) {
	root := t.TempDir()
	localPath := "C:" + "\\Users\\" + "Example\\fixture.aoe2record"
	mustWritePortable(t, filepath.Join(root, "README.md"), "portable root\n")
	mustWritePortable(t, filepath.Join(root, "data", "engine_facts.json"), `{"statement":"`+localPath+`"}`)
	mustWritePortable(t, filepath.Join(root, "docs", "internal.md"), localPath+"\n")

	leaks, err := portableLocalPathLeaks(root)
	if err != nil {
		t.Fatalf("portableLocalPathLeaks failed: %v", err)
	}
	if len(leaks) != 1 {
		t.Fatalf("local path leaks = %+v, want only shipped data leak", leaks)
	}
	if leaks[0] != "data/engine_facts.json: C:\\Users\\" {
		t.Fatalf("leak detail = %q", leaks[0])
	}

	mustWritePortable(t, filepath.Join(root, "data", "engine_facts.json"), `{"statement":"clean"}`)
	leaks, err = portableLocalPathLeaks(root)
	if err != nil {
		t.Fatalf("portableLocalPathLeaks after clean shipped file failed: %v", err)
	}
	if len(leaks) != 0 {
		t.Fatalf("local path leaks = %+v, want non-shipped doc ignored", leaks)
	}
	excludedCount, err := portableExcludedLocalPathReferenceCount(root)
	if err != nil {
		t.Fatalf("portableExcludedLocalPathReferenceCount failed: %v", err)
	}
	if excludedCount != 1 {
		t.Fatalf("excluded local path reference count = %d, want 1", excludedCount)
	}
}

func TestPortablePublicRejectsExecutablePayloads(t *testing.T) {
	root := t.TempDir()
	mustWritePortable(t, filepath.Join(root, "README.md"), "portable root\n")
	mustWritePortable(t, filepath.Join(root, "LICENSE"), "license\n")
	bad := filepath.Join(root, "bad-tool")
	if err := os.WriteFile(bad, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	writePublicAllowlistForTest(t, root, "README.md", "LICENSE", "bad-tool")
	report, err := PortableCheckRootWithOptions(root, PortableCheckOptions{Profile: ProfilePublic})
	if err != nil {
		t.Fatal(err)
	}
	for _, check := range report.Checks {
		if check.Name == "third_party_binaries" {
			if check.OK || !strings.Contains(check.Detail, "bad-tool") {
				t.Fatalf("binary gate did not identify executable: %+v", check)
			}
			return
		}
	}
	t.Fatal("missing third_party_binaries check")
}

func TestIsExecutableMagicRecognizesScriptsAndBinaries(t *testing.T) {
	cases := []struct {
		name string
		data string
		want bool
	}{
		{"shebang script", "#!/bin/sh\necho hi\n", true},
		{"env shebang", "#!/usr/bin/env python3\n", true},
		{"ELF", "\x7fELF\x02\x01", true},
		{"PE", "MZ\x90\x00", true},
		{"Mach-O", "\xcf\xfa\xed\xfe", true},
		{"markdown heading", "# Title\n", false},
		{"text", "portable root\n", false},
		{"empty", "", false},
	}
	for _, c := range cases {
		if got := isExecutableMagic([]byte(c.data)); got != c.want {
			t.Errorf("%s: isExecutableMagic = %t, want %t", c.name, got, c.want)
		}
	}
}

func TestPortablePublicRejectsImageFilesByExtensionAndContent(t *testing.T) {
	if !publicReleaseForbidden("preview.webp") {
		t.Fatal("public allowlist accepted a WebP image path")
	}
	root := t.TempDir()
	mustWritePortable(t, filepath.Join(root, "README.md"), "portable root\n")
	mustWritePortable(t, filepath.Join(root, "LICENSE"), "license\n")
	if err := os.WriteFile(filepath.Join(root, "extensionless"), []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\x0dIHDR\x00\x00\x00\x01\x00\x00\x00\x01"), 0o644); err != nil {
		t.Fatal(err)
	}
	writePublicAllowlistForTest(t, root, "README.md", "LICENSE", "extensionless")
	report, err := PortableCheckRootWithOptions(root, PortableCheckOptions{Profile: ProfilePublic})
	if err != nil {
		t.Fatal(err)
	}
	for _, check := range report.Checks {
		if check.Name == "public_images" {
			if check.OK || !strings.Contains(check.Detail, "extensionless") {
				t.Fatalf("image gate did not identify extensionless image payload: %+v", check)
			}
			return
		}
	}
	t.Fatal("missing public_images check")
}

func writePublicAllowlistForTest(t *testing.T, root string, files ...string) {
	t.Helper()
	data := `{"profile":"public","baseline_commit":"test-baseline","files":[` + `"` + strings.Join(files, `","`) + `"]}`
	mustWritePortable(t, filepath.Join(root, publicAllowlistPath), data)
}

func mustWritePortable(t *testing.T, path, text string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// In a git worktree, .git is a file pointing at the main repository.
func TestPublicAllowlistIgnoresWorktreeGitFile(t *testing.T) {
	root := t.TempDir()
	mustWritePortable(t, filepath.Join(root, "README.md"), "portable root\n")
	mustWritePortable(t, filepath.Join(root, ".git"), "gitdir: ../main/.git/worktrees/x\n")
	mustWritePortable(t, filepath.Join(root, "stray.txt"), "not listed\n")
	leaks, err := publicAllowlistLeaks(root, map[string]bool{"README.md": true})
	if err != nil {
		t.Fatal(err)
	}
	if len(leaks) != 1 || leaks[0] != "stray.txt" {
		t.Fatalf("leaks = %v, want only stray.txt", leaks)
	}
}
