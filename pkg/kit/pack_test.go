package kit

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPackSkipsGitAndOutput(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, ".gitignore"), "/bin/\n/build/go/\n/AoE2Kit.zip\n/kit\n")
	mustWrite(t, filepath.Join(root, "kit"), "fake kit")
	mustWrite(t, filepath.Join(root, "START_HERE.md"), "start")
	mustWrite(t, filepath.Join(root, "AI_GUIDE.md"), "guide")
	mustWrite(t, filepath.Join(root, "docs", "GO_AOE2KIT.md"), "go guide")
	mustWrite(t, filepath.Join(root, "docs", "DAT_ENGINE.md"), "dat")
	mustWrite(t, filepath.Join(root, "docs", "DAT_RECIPE_EXAMPLE.json"), "{}")
	mustWrite(t, filepath.Join(root, "KIT_MANIFEST.json"), string(mustManifestJSON(t)))
	mustWrite(t, filepath.Join(root, ".git", "config"), "secret")
	mustWrite(t, filepath.Join(root, "bin", "linux-amd64", "kit"), "stale")
	mustWrite(t, filepath.Join(root, "build", "go", "kit"), "stale")
	mustWrite(t, filepath.Join(root, "AoE2Kit.zip"), "old archive")

	out := filepath.Join(root, "aoe2kit.zip")
	files, err := packFileList(root, out, nil, true)
	if err != nil {
		t.Fatal(err)
	}
	sawKit := false
	for _, file := range files {
		rel, err := filepath.Rel(root, file)
		if err != nil {
			t.Fatal(err)
		}
		rel = filepath.ToSlash(rel)
		if rel == "kit" {
			sawKit = true
		}
		if filepath.Base(file) == "config" || file == out || strings.HasPrefix(rel, "bin/") || strings.HasPrefix(rel, "build/go/") || rel == "AoE2Kit.zip" {
			t.Fatalf("unexpected packed file: %s", file)
		}
	}
	if !sawKit {
		t.Fatalf("root kit binary should be packed even though it is gitignored")
	}
}

func TestPackSkipsGitWorktreeFile(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, ".git"), "gitdir: /tmp/aoe2kit/.git/worktrees/public\n")
	mustWrite(t, filepath.Join(root, "README.md"), "readme")

	files, err := packFileList(root, filepath.Join(root, "out.zip"), nil, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		rel, err := filepath.Rel(root, file)
		if err != nil {
			t.Fatal(err)
		}
		if filepath.ToSlash(rel) == ".git" {
			t.Fatalf("worktree .git pointer file should not be packed: %v", files)
		}
	}
}

func TestPackCreatesZip(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "README.md"), "readme")
	out := filepath.Join(t.TempDir(), "kit.zip")
	files, err := packFileList(root, out, nil, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	f, err := os.Create(out)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	if err := addZipFile(zw, root, files[0]); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	r, err := zip.OpenReader(out)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	if len(r.File) != 1 || r.File[0].Name != "README.md" {
		t.Fatalf("unexpected zip entries: %#v", r.File)
	}
}

func TestPublicProfileKeepsReleaseDocsAndDropsInternalDiagnostics(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, ".gitignore"), "/kit\n")
	mustWrite(t, filepath.Join(root, "README.md"), "readme")
	mustWrite(t, filepath.Join(root, "NOTICE.md"), "notice")
	mustWrite(t, filepath.Join(root, "kit"), "local binary")
	mustWrite(t, filepath.Join(root, "DEX_TASK_internal.md"), "internal")
	mustWrite(t, filepath.Join(root, "docs", "GO_AOE2KIT.md"), "guide")
	mustWrite(t, filepath.Join(root, "docs", "DAT_ENGINE.md"), "dat")
	mustWrite(t, filepath.Join(root, "docs", "REPLAY_TRANSPARENCY_DIAGNOSTIC_V15_READBACK.md"), "internal readback")
	mustWrite(t, filepath.Join(root, "docs", "diagnostics", "v15", "probe.xs"), "internal")
	mustWrite(t, filepath.Join(root, "tools", "internal_compile", "main.go"), "package main\n")
	mustWrite(t, filepath.Join(root, "release", "public-files.json"), `{"profile":"public","baseline_commit":"test-baseline","files":["README.md","NOTICE.md","docs/GO_AOE2KIT.md","docs/DAT_ENGINE.md","release/public-files.json"]}`)

	files, err := packFileListForProfile(root, filepath.Join(root, "out.zip"), ProfilePublic, ProfilePublic.excludes(), ProfilePublic.includesBinary())
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, file := range files {
		rel, err := filepath.Rel(root, file)
		if err != nil {
			t.Fatal(err)
		}
		seen[filepath.ToSlash(rel)] = true
	}
	for _, rel := range []string{"README.md", "NOTICE.md", "docs/GO_AOE2KIT.md", "docs/DAT_ENGINE.md"} {
		if !seen[rel] {
			t.Fatalf("public profile should include %s; got %#v", rel, seen)
		}
	}
	for _, rel := range []string{"kit", "DEX_TASK_internal.md", "docs/REPLAY_TRANSPARENCY_DIAGNOSTIC_V15_READBACK.md", "docs/diagnostics/v15/probe.xs", "tools/internal_compile/main.go"} {
		if seen[rel] {
			t.Fatalf("public profile should exclude %s", rel)
		}
	}
}

func TestPublicProfileUsesExactAllowlist(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "README.md"), "readme")
	mustWrite(t, filepath.Join(root, "release", "public-files.json"), `{"profile":"public","baseline_commit":"test-baseline","files":["README.md","release/public-files.json"]}`)
	mustWrite(t, filepath.Join(root, "secret.txt"), "must not ship")

	files, err := packFileListForProfile(root, filepath.Join(root, "out.zip"), ProfilePublic, ProfilePublic.excludes(), false)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, file := range files {
		rel, err := filepath.Rel(root, file)
		if err != nil {
			t.Fatal(err)
		}
		seen[filepath.ToSlash(rel)] = true
	}
	if seen["secret.txt"] {
		t.Fatal("unlisted public file was packed")
	}
	if !seen["README.md"] || !seen["release/public-files.json"] {
		t.Fatalf("allowlisted files missing: %v", seen)
	}
}

func mustManifestJSON(t *testing.T) []byte {
	t.Helper()
	data, err := ManifestJSON()
	if err != nil {
		t.Fatal(err)
	}
	return data
}
