package text

import (
	"strings"
	"testing"
)

func TestCompileAndWindowLargeDocument(t *testing.T) {
	doc, err := Compile(strings.Repeat("abc ", 1000), " abc")
	if err != nil {
		t.Fatal(err)
	}
	style := Style{Name: "dialogue", Renderer: RendererBitmap, Budget: 10, GlyphCosts: []int{2, 3, 4, 1}}
	w, err := MaterializeWindow(Intent{Style: style, Document: doc, Behavior: BehaviorCrawl, Lifecycle: LifecycleUntilOffscreen}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if w.ObjectCost > 10 || w.NextCursor == 0 || w.Complete {
		t.Fatalf("unexpected bounded window: %+v", w)
	}
}

func TestOverBudgetIsLoud(t *testing.T) {
	doc, err := Compile("b", "ab")
	if err != nil {
		t.Fatal(err)
	}
	_, err = MaterializeWindow(Intent{Style: Style{Name: "hint", Budget: 1, GlyphCosts: []int{2, 3}}, Document: doc}, 0)
	if err == nil || !strings.Contains(err.Error(), "needs 3") {
		t.Fatalf("expected cost error, got %v", err)
	}
}

func TestStrokeDensityUsesFootprint(t *testing.T) {
	s := Style{Name: "monument", Renderer: RendererStroke, ScaleTilesPerCell: 0.2, SpriteWidthTiles: 0.3, CoverageRatio: 0.2333333333}
	got, err := s.SamplingCells()
	if err != nil || got < 0.349 || got > 0.351 {
		t.Fatalf("sampling=%v err=%v", got, err)
	}
}
