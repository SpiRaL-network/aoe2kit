// Package gameinstall resolves a user's local Age of Empires II installation.
package gameinstall

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"aoe2kit/pkg/gfx"
)

const EnvDir = "AOE2KIT_GAME_DIR"

// Resolve applies Kit's shared precedence: explicit option, environment, then
// conventional Steam discovery. An empty result means discovery found none.
func Resolve(explicit string) (string, error) {
	if dir := strings.TrimSpace(explicit); dir != "" {
		if !isDir(dir) {
			return "", fmt.Errorf("game directory %q is not a directory", dir)
		}
		return filepath.Clean(dir), nil
	}
	if dir := strings.TrimSpace(os.Getenv(EnvDir)); dir != "" {
		if !isDir(dir) {
			return "", fmt.Errorf("%s=%q is not a directory", EnvDir, dir)
		}
		return filepath.Clean(dir), nil
	}
	return Discover()
}

// ResolveOptional chooses a configured path without requiring it to exist. If
// no path is configured, failed platform discovery is non-fatal.
func ResolveOptional(explicit string) (string, error) {
	if dir := strings.TrimSpace(explicit); dir != "" {
		return filepath.Clean(dir), nil
	}
	if dir := strings.TrimSpace(os.Getenv(EnvDir)); dir != "" {
		return filepath.Clean(dir), nil
	}
	dir, err := Discover()
	if err != nil {
		return "", nil
	}
	return dir, nil
}

// Discover searches conventional Steam install locations without assuming a
// particular user's home directory or Steam library path.
func Discover() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	var roots []string
	if runtime.GOOS == "windows" {
		roots = []string{os.Getenv("PROGRAMFILES(X86)"), os.Getenv("PROGRAMFILES")}
	} else {
		roots = []string{
			filepath.Join(home, ".steam", "steam"),
			filepath.Join(home, ".local", "share", "Steam"),
			filepath.Join(home, ".steam", "root"),
		}
	}
	for _, root := range roots {
		if root == "" {
			continue
		}
		for _, name := range []string{"Age of Empires II Definitive Edition", "AoE2DE"} {
			var dir string
			if runtime.GOOS == "windows" {
				dir = filepath.Join(root, "Steam", "steamapps", "common", name)
			} else {
				dir = filepath.Join(root, "steamapps", "common", name)
			}
			if isDir(dir) && hasGameAssets(dir) {
				return dir, nil
			}
		}
	}
	return "", errors.New("could not find a local Age of Empires II installation; use --game-dir or set AOE2KIT_GAME_DIR")
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func hasGameAssets(dir string) bool {
	for _, path := range []string{
		filepath.Join(dir, "resources", "_common", "dat", "empires2_x2_p1.dat"),
		filepath.Join(dir, "resources", "_common", "drs", "graphics.drs"),
		filepath.Join(dir, "resources", "_common", "drs", "graphics.sld"),
		filepath.Join(dir, "resources", "_common", "strings", "key-value-strings-utf8.txt"),
		filepath.Join(dir, "resources", "key-value-strings-utf8.txt"),
	} {
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			return true
		}
	}
	return false
}

// DataPath returns the conventional DE DAT path below a resolved game root.
func DataPath(gameDir string) string {
	return filepath.Join(gameDir, "resources", "_common", "dat", "empires2_x2_p1.dat")
}

// BlendomaticPath locates the installed terrain-transition masks beside the DAT.
func BlendomaticPath(gameDir string) string {
	return filepath.Join(gameDir, "resources", "_common", "dat", "blendomatic_x1.dat")
}

// TerrainAssetPath resolves only a basename in a known installed terrain group.
// DAT references are data, not paths: separators and traversal are rejected.
func TerrainAssetPath(gameDir, group, name string) (string, error) {
	if name == "" || name == "." || name == ".." || strings.ContainsAny(name, "/\\:") {
		return "", fmt.Errorf("invalid terrain asset name %q", name)
	}
	if group != "textures/2x" && group != "masks" && group != "blends" {
		return "", fmt.Errorf("invalid terrain asset group %q", group)
	}
	root := filepath.Join(gameDir, "resources", "_common", "terrain", filepath.FromSlash(group))
	entries, err := os.ReadDir(root)
	if err != nil {
		return "", err
	}
	for _, entry := range entries {
		if !entry.IsDir() && strings.EqualFold(entry.Name(), name) {
			return filepath.Join(root, entry.Name()), nil
		}
	}
	return "", fmt.Errorf("installed terrain asset %s/%s not found", group, name)
}

// MinimapPalettePath returns the conventional terrain palette path when it
// exists in the resolved game installation.
func MinimapPalettePath(gameDir string) string {
	for _, rel := range []string{
		filepath.Join("widgetui", "palette.json"),
		filepath.Join("resources", "_common", "widgetui", "palette.json"),
	} {
		path := filepath.Join(gameDir, rel)
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			return path
		}
	}
	return ""
}

// GraphicsDir returns the conventional loose-asset directory. Packed DRS
// archives are deliberately not treated as loose SLD files.
func GraphicsDir(gameDir string) string {
	return filepath.Join(gameDir, "resources", "_common", "drs", "graphics")
}

// FindLooseSLD finds a loose SLD whose basename matches the DAT graphic file
// reference. Packed DRS content remains unsupported and is never extracted.
func FindLooseSLD(gameDir, fileName string) (string, error) {
	want := strings.TrimSuffix(strings.ToLower(filepath.Base(fileName)), filepath.Ext(fileName))
	if want == "" {
		return "", errors.New("empty graphic file name")
	}
	root := filepath.Join(gameDir, "resources", "_common", "drs")
	var found string
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".sld") {
			return nil
		}
		stem := strings.TrimSuffix(strings.ToLower(entry.Name()), filepath.Ext(entry.Name()))
		if stem == want {
			found = path
			return fs.SkipAll
		}
		return nil
	})
	if err != nil && !errors.Is(err, fs.SkipAll) {
		return "", err
	}
	if found == "" {
		return "", fmt.Errorf("no loose SLD for graphic %q under the game graphics directory", fileName)
	}
	return found, nil
}

type Status struct {
	Found               bool   `json:"found"`
	GameDir             string `json:"game_dir,omitempty"`
	SpritesReadable     bool   `json:"sprites_readable"`
	SpriteAssetCount    int    `json:"sprite_asset_count"`
	SpriteAssetLocation string `json:"sprite_asset_location,omitempty"`
	Message             string `json:"message"`
}

func Inspect(explicit string) Status {
	dir, err := Resolve(explicit)
	if err != nil || !hasGameAssets(dir) {
		return Status{Message: "no install; flat previews only"}
	}
	status := Status{Found: true, GameDir: dir, Message: "game install found; sprite preview requires readable loose SLD assets"}
	root := filepath.Join(dir, "resources", "_common", "drs")
	_ = filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".sld") {
			return nil
		}
		status.SpriteAssetCount++
		if status.SpriteAssetLocation == "" {
			status.SpriteAssetLocation = filepath.Dir(path)
		}
		if !status.SpritesReadable {
			if _, parseErr := gfx.OpenSLD(path); parseErr == nil {
				status.SpritesReadable = true
			}
		}
		return nil
	})
	if status.SpritesReadable {
		status.Message = "game install found; loose SLD sprites are readable"
	} else if status.SpriteAssetCount > 0 {
		status.Message = "game install found; loose SLD files exist but none inspected are readable"
	}
	return status
}
