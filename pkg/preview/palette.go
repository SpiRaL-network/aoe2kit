package preview

import (
	"encoding/json"
	"fmt"
	"image/color"
	"os"
)

type minimapPaletteFile struct {
	Palette []struct {
		R int `json:"r"`
		G int `json:"g"`
		B int `json:"b"`
		A int `json:"a"`
	} `json:"Palette"`
}

// ReadMinimapPalette reads the game's indexed widgetui palette used by the minimap.
func ReadMinimapPalette(path string) ([]color.RGBA, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var file minimapPaletteFile
	if err := json.Unmarshal(b, &file); err != nil {
		return nil, fmt.Errorf("decode minimap palette: %w", err)
	}
	if len(file.Palette) != 256 {
		return nil, fmt.Errorf("minimap palette has %d entries, want 256", len(file.Palette))
	}
	colors := make([]color.RGBA, len(file.Palette))
	for i, entry := range file.Palette {
		if entry.R < 0 || entry.R > 255 || entry.G < 0 || entry.G > 255 || entry.B < 0 || entry.B > 255 {
			return nil, fmt.Errorf("minimap palette entry %d has RGB component outside 0..255", i)
		}
		colors[i] = color.RGBA{R: uint8(entry.R), G: uint8(entry.G), B: uint8(entry.B), A: 255}
	}
	return colors, nil
}
