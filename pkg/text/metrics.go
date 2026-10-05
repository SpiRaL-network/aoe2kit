package text

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"
)

// MetricDocument is the auditable data contract for extracted game font
// metrics. Kit does not ship the game's atlases or private extraction tool.
type MetricDocument struct {
	Schema string                 `json:"schema"`
	Build  string                 `json:"build"`
	Source string                 `json:"source"`
	Fonts  map[string]FontMetrics `json:"fonts"`
}

type FontMetrics struct {
	Widths             map[string]float64 `json:"widths"`
	SpaceWidth         float64            `json:"space_width"`
	RenderedSpaceWidth float64            `json:"rendered_space_width,omitempty"`
}

func LoadMetrics(data []byte) (MetricDocument, error) {
	var doc MetricDocument
	if err := json.Unmarshal(data, &doc); err != nil {
		return MetricDocument{}, fmt.Errorf("decode font metrics: %w", err)
	}
	if len(doc.Fonts) == 0 {
		return MetricDocument{}, fmt.Errorf("font metrics contain no fonts")
	}
	for name, font := range doc.Fonts {
		if font.SpaceWidth <= 0 {
			return MetricDocument{}, fmt.Errorf("font %q has invalid space_width", name)
		}
	}
	return doc, nil
}

func (doc MetricDocument) Measure(fontName, value string) (float64, error) {
	font, ok := doc.Fonts[fontName]
	if !ok {
		return 0, fmt.Errorf("unknown font %q", fontName)
	}
	if !utf8.ValidString(value) {
		return 0, fmt.Errorf("text is not valid UTF-8")
	}
	width := 0.0
	for _, r := range value {
		if r == ' ' {
			width += font.effectiveSpaceWidth()
			continue
		}
		glyph, ok := font.Widths[strconv.Itoa(int(r))]
		if !ok {
			return 0, fmt.Errorf("font %q has no width for U+%04X", fontName, r)
		}
		width += glyph
	}
	return width, nil
}

func (f FontMetrics) effectiveSpaceWidth() float64 {
	if f.RenderedSpaceWidth > 0 {
		return f.RenderedSpaceWidth
	}
	return f.SpaceWidth
}

type Alignment struct {
	Font        string  `json:"font"`
	Text        string  `json:"text"`
	TargetWidth float64 `json:"target_width"`
	TextWidth   float64 `json:"text_width"`
	Padding     float64 `json:"padding"`
}

func (doc MetricDocument) AlignRight(fontName, value string, targetWidth float64) (Alignment, error) {
	if targetWidth < 0 {
		return Alignment{}, fmt.Errorf("target width must be non-negative")
	}
	textWidth, err := doc.Measure(fontName, value)
	if err != nil {
		return Alignment{}, err
	}
	return Alignment{Font: fontName, Text: value, TargetWidth: targetWidth, TextWidth: textWidth, Padding: targetWidth - textWidth}, nil
}

// GenerateXSWidthHelper emits the runtime width function while baking only
// the measured table. Callers choose the invisible-space glyphs separately.
func (doc MetricDocument) GenerateXSWidthHelper(fontName, functionName string) (string, error) {
	font, ok := doc.Fonts[fontName]
	if !ok {
		return "", fmt.Errorf("unknown font %q", fontName)
	}
	if strings.TrimSpace(functionName) == "" {
		return "", fmt.Errorf("function name is required")
	}
	keys := make([]int, 0, len(font.Widths))
	values := make(map[int]float64, len(font.Widths))
	for key, width := range font.Widths {
		codepoint, err := strconv.Atoi(key)
		if err != nil || codepoint < 0 || codepoint > 255 {
			return "", fmt.Errorf("font %q has non-ASCII width key %q", fontName, key)
		}
		keys = append(keys, codepoint)
		values[codepoint] = width
	}
	sort.Ints(keys)
	var b strings.Builder
	fmt.Fprintf(&b, "// Generated from %s (%s); structure-verified until engine-tested.\n", doc.Source, doc.Build)
	fmt.Fprintf(&b, "float %s(string text = \"\") {\n", functionName)
	fmt.Fprint(&b, "  int i = 0; float width = 0.0;\n  while (i < strLen(text)) {\n    int code = ord(strCharAt(text, i));\n")
	fmt.Fprintf(&b, "    if (code == 32) { width = width + %.6f; }\n", font.effectiveSpaceWidth())
	for _, codepoint := range keys {
		fmt.Fprintf(&b, "    else if (code == %d) { width = width + %.6f; }\n", codepoint, values[codepoint])
	}
	fmt.Fprint(&b, "    i = i + 1;\n  }\n  return (width);\n}\n")
	return b.String(), nil
}
