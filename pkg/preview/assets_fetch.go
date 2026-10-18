package preview

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"aoe2kit/pkg/gameinstall"
)

type AssetFetchReport struct {
	Schema     string        `json:"schema"`
	Copied     []string      `json:"copied"`
	Present    []string      `json:"already_present"`
	Remaining  []NeededAsset `json:"remaining"`
	Unresolved []AssetIssue  `json:"unresolved,omitempty"`
	Notes      []string      `json:"notes"`
}

// FetchMissingSLDs only copies loose graphics needed by this manifest. It never
// overwrites files or extracts archives, and rejects links escaping graphics.
func FetchMissingSLDs(manifest AssetManifest, source, mirror string) (AssetFetchReport, error) {
	r := AssetFetchReport{Schema: "aoe2kit.preview.asset-fetch.v1", Copied: []string{}, Present: []string{}, Remaining: []NeededAsset{}, Unresolved: manifest.Unresolved, Notes: []string{"Local game art only: do not commit or redistribute the mirror. Only missing loose SLDs are copied; particle and terrain assets are not fetched."}}
	sourceRoot, err := filepath.EvalSymlinks(source)
	if err != nil {
		return r, err
	}
	sourceRoot, err = filepath.Abs(sourceRoot)
	if err != nil {
		return r, err
	}
	graphics, err := filepath.EvalSymlinks(gameinstall.GraphicsDir(sourceRoot))
	if err != nil {
		return r, err
	}
	if !assetWithin(sourceRoot, graphics) {
		return r, fmt.Errorf("source graphics directory escapes the installation")
	}
	files := map[string]string{}
	err = filepath.WalkDir(graphics, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.EqualFold(filepath.Ext(d.Name()), ".sld") {
			return nil
		}
		key := strings.ToLower(d.Name())
		if prior := files[key]; prior != "" {
			return fmt.Errorf("ambiguous source SLD basename %q", d.Name())
		}
		files[key] = p
		return nil
	})
	if err != nil {
		return r, err
	}
	// Validate the full copy plan before creating directories or writing files.
	type copyEntry struct{ source, name string }
	var plan []copyEntry
	seen := map[string]bool{}
	for _, a := range manifest.Assets {
		if a.Status != "missing" {
			continue
		}
		rel, err := safeAssetRelative(a.Path)
		if err != nil {
			return r, err
		}
		const prefix = "resources/_common/drs/graphics/"
		if a.Kind != "sld" {
			r.Remaining = append(r.Remaining, a)
			continue
		}
		if !strings.HasPrefix(rel, prefix) || !strings.EqualFold(filepath.Ext(rel), ".sld") {
			return r, fmt.Errorf("SLD destination is outside graphics: %q", a.Path)
		}
		name := strings.TrimPrefix(rel, prefix)
		if strings.Contains(name, "/") {
			return r, fmt.Errorf("expected an SLD basename: %q", a.Path)
		}
		key := strings.ToLower(name)
		if seen[key] {
			continue
		}
		seen[key] = true
		src := files[key]
		if src == "" {
			r.Remaining = append(r.Remaining, a)
			continue
		}
		resolved, err := filepath.EvalSymlinks(src)
		if err != nil {
			return r, err
		}
		if !assetWithin(graphics, resolved) {
			return r, fmt.Errorf("SLD source escapes graphics: %q", name)
		}
		st, err := os.Stat(resolved)
		if err != nil {
			return r, err
		}
		if !st.Mode().IsRegular() {
			return r, fmt.Errorf("SLD source is not a regular file: %q", name)
		}
		plan = append(plan, copyEntry{resolved, name})
	}
	if len(plan) > 0 {
		if err := os.MkdirAll(mirror, 0755); err != nil {
			return r, err
		}
		root, err := filepath.EvalSymlinks(mirror)
		if err != nil {
			return r, err
		}
		root, err = filepath.Abs(root)
		if err != nil {
			return r, err
		}
		dest := root
		for _, part := range []string{"resources", "_common", "drs", "graphics"} {
			dest = filepath.Join(dest, part)
			if err := os.Mkdir(dest, 0755); err != nil && !os.IsExist(err) {
				return r, err
			}
			st, err := os.Lstat(dest)
			if err != nil {
				return r, err
			}
			if !st.IsDir() || st.Mode()&os.ModeSymlink != 0 {
				return r, fmt.Errorf("mirror graphics path contains a link or non-directory")
			}
		}
		for _, p := range plan {
			name := filepath.Join(dest, p.name)
			rel := "resources/_common/drs/graphics/" + p.name
			copied, err := copySLDExclusive(p.source, name)
			if err != nil {
				return r, err
			}
			if copied {
				r.Copied = append(r.Copied, rel)
			} else {
				r.Present = append(r.Present, rel)
			}
		}
	}
	sort.Strings(r.Copied)
	sort.Strings(r.Present)
	if len(r.Remaining) > 0 || len(r.Unresolved) > 0 {
		return r, fmt.Errorf("asset fetch incomplete: %d remaining assets, %d unresolved references", len(r.Remaining), len(r.Unresolved))
	}
	return r, nil
}

func assetWithin(root, name string) bool {
	rel, err := filepath.Rel(root, name)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)
}

func copySLDExclusive(source, destination string) (bool, error) {
	if st, err := os.Lstat(destination); err == nil {
		if !st.Mode().IsRegular() {
			return false, fmt.Errorf("existing destination is not a regular SLD")
		}
		return false, nil
	} else if !os.IsNotExist(err) {
		return false, err
	}
	in, err := os.Open(source)
	if err != nil {
		return false, err
	}
	defer in.Close()
	out, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		return false, err
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil || closeErr != nil {
		_ = os.Remove(destination)
		if copyErr != nil {
			return false, copyErr
		}
		return false, closeErr
	}
	return true, nil
}
