package text

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"aoe2kit/pkg/xs"
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

func TestMetricDocumentMeasureAlignAndGenerate(t *testing.T) {
	doc, err := LoadMetrics([]byte(`{"schema":"aoe2kit.font-metrics.v1","build":"test","source":"fixture","fonts":{"sans":{"space_width":1,"widths":{"65":2,"66":3}}}}`))
	if err != nil {
		t.Fatal(err)
	}
	width, err := doc.Measure("sans", "A B")
	if err != nil || width != 6 {
		t.Fatalf("measure=%v err=%v, want 6", width, err)
	}
	aligned, err := doc.AlignRight("sans", "AB", 10)
	if err != nil || aligned.Padding != 5 {
		t.Fatalf("alignment=%+v err=%v, want padding 5", aligned, err)
	}
	code, err := doc.GenerateXSWidthHelper("sans", "measureSans")
	if err != nil || !strings.Contains(code, "strCharAt") || !strings.Contains(code, "measureSans") {
		t.Fatalf("generated helper invalid: %v\n%s", err, code)
	}
	if !strings.Contains(code, `float measureSans(string text = "")`) {
		t.Fatalf("generated helper lacks XS default parameter:\n%s", code)
	}
	if !strings.Contains(code, "return (width);") {
		t.Fatalf("generated helper lacks parenthesized return:\n%s", code)
	}
	if findings := xs.LintSource("font_metrics.xs", code); len(findings) != 0 {
		t.Fatalf("generated helper has Kit lint findings: %+v\n%s", findings, code)
	}

	checker, err := xs.ResolveParser("")
	if err != nil {
		t.Fatalf("resolve xs-check: %v", err)
	}
	if checker == "" {
		t.Skip("xs-check unavailable")
	}
	path := filepath.Join(t.TempDir(), "font_metrics.xs")
	if err := os.WriteFile(path, []byte(code), 0o644); err != nil {
		t.Fatal(err)
	}
	if report, err := xs.CheckFile(path, checker); err != nil {
		t.Fatalf("xs-check report=%+v: %v", report, err)
	}
}
