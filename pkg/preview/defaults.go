package preview

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// ProjectDefaults supplies optional CLI defaults, never output destinations.
type ProjectDefaults struct {
	Dat       string `json:"dat,omitempty"`
	GameDir   string `json:"game_dir,omitempty"`
	Sprites   bool   `json:"sprites,omitempty"`
	Textures  bool   `json:"textures,omitempty"`
	Shadows   bool   `json:"shadows,omitempty"`
	Mood      string `json:"mood,omitempty"`
	Elevation bool   `json:"elevation,omitempty"`
	Scale     int    `json:"scale,omitempty"`
}

// LoadProjectDefaults searches upward for kit-preview.json. Relative asset paths
// are resolved from that file, not from the caller's working directory.
func LoadProjectDefaults(dir string) (ProjectDefaults, error) {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return ProjectDefaults{}, err
	}
	for {
		path := filepath.Join(dir, "kit-preview.json")
		data, err := os.ReadFile(path)
		if err == nil {
			var d ProjectDefaults
			dec := json.NewDecoder(bytes.NewReader(data))
			dec.DisallowUnknownFields()
			if err := dec.Decode(&d); err != nil {
				return d, fmt.Errorf("preview defaults: %w", err)
			}
			if err := dec.Decode(new(any)); err != io.EOF {
				return d, fmt.Errorf("preview defaults: expected one JSON object")
			}
			if d.Scale < 0 || d.Scale > 128 {
				return d, fmt.Errorf("preview defaults: scale must be 1..128, or 0 for the default")
			}
			for _, p := range []*string{&d.Dat, &d.GameDir} {
				if *p != "" && !filepath.IsAbs(*p) {
					*p = filepath.Join(dir, *p)
				}
			}
			return d, nil
		}
		if !os.IsNotExist(err) {
			return ProjectDefaults{}, err
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ProjectDefaults{}, nil
		}
		dir = parent
	}
}
