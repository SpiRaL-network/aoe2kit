// Package text plans bounded Gaia text windows without coupling callers to XS
// geometry. The standalone examples/text/gaia_text.xs remains the no-toolchain
// runtime path; this package is Kit's optional document/cost convenience layer.
package text

import "fmt"

type Renderer string

const (
	RendererBitmap Renderer = "bitmap"
	RendererStroke Renderer = "stroke"
)

type Behavior string

const (
	BehaviorStatic     Behavior = "static"
	BehaviorCrawl      Behavior = "crawl"
	BehaviorTypewriter Behavior = "typewriter"
	BehaviorPage       Behavior = "page"
	BehaviorDialogue   Behavior = "dialogue"
)

type Lifecycle string

const (
	LifecyclePermanent      Lifecycle = "permanent"
	LifecycleHold           Lifecycle = "hold"
	LifecycleUntilOffscreen Lifecycle = "until_offscreen"
	LifecycleWhileVariable  Lifecycle = "while_variable"
	LifecycleReadingTime    Lifecycle = "reading_time"
)

type Style struct {
	Name               string
	Renderer           Renderer
	ForegroundMaterial int
	BackgroundMaterial int
	ScaleTilesPerCell  float64
	SpriteWidthTiles   float64
	CoverageRatio      float64
	Budget             int
	GlyphCosts         []int
}

func (s Style) SamplingCells() (float64, error) {
	if s.Renderer != RendererStroke {
		return 0, nil
	}
	if s.ScaleTilesPerCell <= 0 || s.SpriteWidthTiles <= 0 || s.CoverageRatio <= 0 {
		return 0, fmt.Errorf("style %q has invalid footprint geometry", s.Name)
	}
	return s.SpriteWidthTiles * s.CoverageRatio / s.ScaleTilesPerCell, nil
}

type Document struct {
	Source string
	Glyphs []int
}

func Compile(source, alphabet string) (Document, error) {
	doc := Document{Source: source, Glyphs: make([]int, 0, len(source))}
	for i := 0; i < len(source); i++ {
		id := -1
		for j := 0; j < len(alphabet); j++ {
			if alphabet[j] == source[i] {
				id = j
				break
			}
		}
		if id < 0 {
			return Document{}, fmt.Errorf("unsupported text byte %q at offset %d", source[i], i)
		}
		doc.Glyphs = append(doc.Glyphs, id)
	}
	return doc, nil
}

type Intent struct {
	Style     Style
	Document  Document
	Behavior  Behavior
	Lifecycle Lifecycle
	Budget    int
	MaxWindow int
}

type Window struct {
	Start      int
	End        int
	Glyphs     []int
	ObjectCost int
	NextCursor int
	Complete   bool
}

func (s Style) glyphCost(id int) (int, error) {
	if id < 0 || id >= len(s.GlyphCosts) {
		return 0, fmt.Errorf("style %q has no cost for glyph %d", s.Name, id)
	}
	if s.GlyphCosts[id] <= 0 {
		return 0, fmt.Errorf("style %q has invalid cost %d for glyph %d", s.Name, s.GlyphCosts[id], id)
	}
	return s.GlyphCosts[id], nil
}

// MaterializeWindow returns the largest document prefix that fits the declared
// object ceiling. A glyph that cannot fit as the first item is an error; no
// request is silently truncated.
func MaterializeWindow(intent Intent, cursor int) (Window, error) {
	if cursor < 0 || cursor > len(intent.Document.Glyphs) {
		return Window{}, fmt.Errorf("invalid document cursor %d", cursor)
	}
	budget := intent.Budget
	if budget <= 0 {
		budget = intent.Style.Budget
	}
	if budget <= 0 {
		return Window{}, fmt.Errorf("text budget must be declared")
	}
	limit := len(intent.Document.Glyphs)
	if intent.MaxWindow > 0 && cursor+intent.MaxWindow < limit {
		limit = cursor + intent.MaxWindow
	}
	window := Window{Start: cursor, NextCursor: cursor}
	for i := cursor; i < limit; i++ {
		cost, err := intent.Style.glyphCost(intent.Document.Glyphs[i])
		if err != nil {
			return Window{}, err
		}
		if window.ObjectCost+cost > budget {
			if i == cursor {
				return Window{}, fmt.Errorf("text window cannot fit glyph at %d: needs %d objects, budget %d", i, cost, budget)
			}
			break
		}
		window.Glyphs = append(window.Glyphs, intent.Document.Glyphs[i])
		window.ObjectCost += cost
		window.NextCursor = i + 1
	}
	window.End = window.NextCursor
	window.Complete = window.End == len(intent.Document.Glyphs)
	return window, nil
}
