// Package gamestrings reads user-provided Age of Empires II string resources.
// The game data is intentionally not bundled with Kit.
package gamestrings

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

const tableFileName = "key-value-strings-utf8.txt"

// Table is a best-effort view of the game's integer string table.
type Table struct {
	Path   string
	Values map[int]string
}

func (t Table) Lookup(id int) (string, bool) {
	v, ok := t.Values[id]
	return v, ok && strings.TrimSpace(v) != ""
}

// Load parses a key-value-strings-utf8.txt file. Lines that are not integer
// keys followed by a quoted value are ignored, allowing comments and headers.
func Load(path string) (Table, error) {
	f, err := os.Open(path)
	if err != nil {
		return Table{}, err
	}
	defer f.Close()

	table := Table{Path: path, Values: make(map[int]string)}
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 4096), 4*1024*1024)
	line := 0
	for scanner.Scan() {
		line++
		fields := strings.Fields(scanner.Text())
		if len(fields) < 2 {
			continue
		}
		id, err := strconv.Atoi(fields[0])
		if err != nil {
			continue
		}
		value, err := quotedValue(scanner.Text())
		if err != nil {
			return Table{}, fmt.Errorf("%s:%d: invalid string value: %w", path, line, err)
		}
		table.Values[id] = value
	}
	if err := scanner.Err(); err != nil {
		return Table{}, fmt.Errorf("read %s: %w", path, err)
	}
	return table, nil
}

func quotedValue(line string) (string, error) {
	start := strings.IndexByte(line, '"')
	if start < 0 {
		return "", fmt.Errorf("missing quoted value")
	}
	escaped := false
	for i := start + 1; i < len(line); i++ {
		switch {
		case escaped:
			escaped = false
		case line[i] == '\\':
			escaped = true
		case line[i] == '"':
			return strconv.Unquote(line[start : i+1])
		}
	}
	return "", fmt.Errorf("unterminated quoted value")
}

// LoadFromGameDir locates the game's string table below an installation
// directory. It never downloads, probes a remote machine, or falls back to
// Kit-owned data.
func LoadFromGameDir(gameDir string) (Table, error) {
	gameDir = filepath.Clean(gameDir)
	if info, err := os.Stat(gameDir); err == nil && !info.IsDir() {
		return Load(gameDir)
	}
	candidates := []string{
		filepath.Join(gameDir, tableFileName),
		filepath.Join(gameDir, "resources", tableFileName),
		filepath.Join(gameDir, "resources", "en", tableFileName),
		filepath.Join(gameDir, "resources", "_common", tableFileName),
		filepath.Join(gameDir, "resources", "_common", "strings", tableFileName),
	}
	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return Load(path)
		}
	}
	return Table{}, fmt.Errorf("%s not found below game directory %s", tableFileName, gameDir)
}

// FindGameDir returns the first conventional local Steam installation that
// contains the string table. It is deliberately conservative and returns an
// error when no local installation is discoverable.
func FindGameDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	var candidates []string
	if runtime.GOOS == "windows" {
		candidates = []string{
			filepath.Join(os.Getenv("PROGRAMFILES(X86)"), "Steam", "steamapps", "common", "Age of Empires II Definitive Edition"),
			filepath.Join(os.Getenv("PROGRAMFILES(X86)"), "Steam", "steamapps", "common", "AoE2DE"),
			filepath.Join(os.Getenv("PROGRAMFILES"), "Steam", "steamapps", "common", "Age of Empires II Definitive Edition"),
			filepath.Join(os.Getenv("PROGRAMFILES"), "Steam", "steamapps", "common", "AoE2DE"),
		}
	} else {
		for _, root := range []string{
			filepath.Join(home, ".steam", "steam"),
			filepath.Join(home, ".local", "share", "Steam"),
			filepath.Join(home, ".steam", "root"),
		} {
			candidates = append(candidates, filepath.Join(root, "steamapps", "common", "Age of Empires II Definitive Edition"))
			candidates = append(candidates, filepath.Join(root, "steamapps", "common", "AoE2DE"))
		}
	}
	for _, dir := range candidates {
		if dir == "" {
			continue
		}
		if _, err := LoadFromGameDir(dir); err == nil {
			return dir, nil
		}
	}
	return "", errors.New("could not find a local Age of Empires II installation; use --game-dir")
}

// LoadAuto uses an explicit directory when provided, otherwise it tries the
// conventional local Steam locations. A missing installation is not fatal to
// callers; they can keep numeric IDs or built-in compatibility names.
func LoadAuto(gameDir string) (Table, error) {
	if strings.TrimSpace(gameDir) != "" {
		return LoadFromGameDir(gameDir)
	}
	dir, err := FindGameDir()
	if err != nil {
		return Table{}, err
	}
	return LoadFromGameDir(dir)
}
