package kit

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"aoe2kit/pkg/aoe2"
	"aoe2kit/pkg/modpack"
)

type Inventory struct {
	Root     string                `json:"root"`
	Files    []aoe2.FileInfo       `json:"files"`
	ModDirs  []modpack.CheckReport `json:"mod_dirs"`
	Warnings []string              `json:"warnings,omitempty"`
}

func Scan(root string) (Inventory, error) {
	st, err := os.Stat(root)
	if err != nil {
		return Inventory{}, err
	}
	if !st.IsDir() {
		return Inventory{}, &notDirError{path: root}
	}

	inv := Inventory{Root: root}
	seenModDirs := map[string]struct{}{}
	err = filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			inv.Warnings = append(inv.Warnings, walkErr.Error())
			return nil
		}
		name := d.Name()
		if d.IsDir() {
			if shouldSkipDir(name) {
				return filepath.SkipDir
			}
			infoPath := filepath.Join(path, "info.json")
			resCommon := filepath.Join(path, "resources", "_common")
			if exists(infoPath) && isDir(resCommon) {
				report, err := modpack.Check(path)
				if err != nil {
					inv.Warnings = append(inv.Warnings, err.Error())
					return nil
				}
				if _, ok := seenModDirs[path]; !ok {
					seenModDirs[path] = struct{}{}
					inv.ModDirs = append(inv.ModDirs, report)
				}
			}
			return nil
		}
		kind := aoe2.KindForPath(path)
		if kind == "" || kind == "go" {
			return nil
		}
		switch kind {
		case "scenario", "record", "dat", "ai", "gfx", "json", "text":
			info, err := aoe2.InspectFile(path)
			if err != nil {
				inv.Warnings = append(inv.Warnings, err.Error())
				return nil
			}
			inv.Files = append(inv.Files, info)
		}
		return nil
	})
	if err != nil {
		return Inventory{}, err
	}

	sort.Slice(inv.Files, func(i, j int) bool { return inv.Files[i].Path < inv.Files[j].Path })
	sort.Slice(inv.ModDirs, func(i, j int) bool { return inv.ModDirs[i].Path < inv.ModDirs[j].Path })
	return inv, nil
}

func shouldSkipDir(name string) bool {
	low := strings.ToLower(name)
	if strings.HasPrefix(low, ".venv") {
		return true
	}
	switch low {
	case ".git", ".hg", ".svn", "node_modules", "__pycache__", ".pytest_cache", "build", ".venv", "venv":
		return true
	}
	return false
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func isDir(path string) bool {
	st, err := os.Stat(path)
	return err == nil && st.IsDir()
}

type notDirError struct {
	path string
}

func (e *notDirError) Error() string {
	return e.path + " is not a directory"
}
