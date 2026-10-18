package preview

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"

	"aoe2kit/pkg/datfile"
	"aoe2kit/pkg/felt"
	"aoe2kit/pkg/gameinstall"
	"aoe2kit/pkg/gfx"
	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

type GrammarOptions struct {
	UnitID                  int
	Dat                     *datfile.Index
	GameDir, CompendiumPath string
}

type GrammarReport struct {
	Schema      string       `json:"schema"`
	Entries     []felt.Entry `json:"entries"`
	GraphicFile string       `json:"graphic_file"`
	FrameCount  int          `json:"animation_frames_per_variant"`
	Items       []SpriteItem `json:"items"`
	Notes       []string     `json:"notes"`
}

// RenderGrammar renders installed art only; grammar and generated images carry
// no assertion about collision, passability or engine-facing semantics.
func RenderGrammar(o GrammarOptions) (*image.RGBA, GrammarReport, error) {
	r := GrammarReport{Schema: "aoe2kit.felt.grammar.v1", Items: []SpriteItem{}, Notes: []string{"DAT facings enumerate variants, using their first animation frame. Fitted previews on grass/water are for visual review, not footprint or passability calibration. Unmapped fields require review; preview_reviewed is not engine_verified."}}
	if o.Dat == nil || o.GameDir == "" {
		return nil, r, fmt.Errorf("grammar needs a DAT and installed assets")
	}
	u, ok := unitMetadata(o.Dat, 0, o.UnitID)
	if !ok {
		return nil, r, fmt.Errorf("unit %d absent from Gaia DAT", o.UnitID)
	}
	g, ok := o.Dat.Graphic(int(u.StandingGraphic1))
	if !ok {
		return nil, r, fmt.Errorf("standing graphic absent")
	}
	c, err := felt.Load(o.CompendiumPath)
	if err != nil {
		return nil, r, err
	}
	e, ok := c.Find(o.UnitID)
	if !ok {
		e = felt.Entry{UnitID: o.UnitID, Name: u.Name, Contexts: []string{"grass"}}
	}
	e, err = felt.Skeleton(e, max(1, int(g.AngleCount)))
	if err != nil {
		return nil, r, err
	}
	if len(e.Contexts) == 0 {
		e.Contexts = []string{"grass"}
	}
	r.Entries = []felt.Entry{e}
	r.GraphicFile = g.FileName.Value
	r.FrameCount = int(g.FrameCount)
	path, err := gameinstall.FindLooseSLD(o.GameDir, g.FileName.Value)
	if err != nil {
		return nil, r, err
	}
	sld, err := gfx.OpenSLD(path)
	if err != nil {
		return nil, r, err
	}
	const cellW, cellH, cols = 250, 210, 4
	n := len(e.Grammar) * len(e.Contexts)
	rows := (n + cols - 1) / cols
	if rows*cellH*1000 > maxImagePixels {
		return nil, r, fmt.Errorf("grammar sheet exceeds pixel limit")
	}
	canvas := image.NewRGBA(image.Rect(0, 0, 1000, rows*cellH))
	draw.Draw(canvas, canvas.Bounds(), image.NewUniform(color.RGBA{32, 35, 39, 255}), image.Point{}, draw.Src)
	backgrounds := map[string]*image.RGBA{}
	for _, context := range e.Contexts {
		id := 0
		if context == "water" {
			id = 1
		}
		snap := Snapshot{Schema: SnapshotSchema, Width: 4, Height: 4}
		for y := 0; y < 4; y++ {
			for x := 0; x < 4; x++ {
				snap.Terrain = append(snap.Terrain, Tile{X: x, Y: y, TerrainID: id})
			}
		}
		bg, _, err := RenderFlat(snap, Options{Projection: "diamond", Scale: 50, Dat: o.Dat, Textures: &TerrainTextureOptions{GameDir: o.GameDir}})
		if err != nil {
			return nil, r, err
		}
		backgrounds[context] = bg
	}
	for i, v := range e.Grammar {
		ordinal := angleFrameOrdinal(v.Index, int(g.AngleCount), int(g.FrameCount))
		var sprite *image.RGBA
		var renderErr error
		if ordinal < 0 || ordinal >= len(sld.Frames) {
			renderErr = fmt.Errorf("frame %d outside SLD", ordinal)
		} else {
			sprite, _, renderErr = sld.RenderFrame(ordinal, color.RGBA{})
		}
		item := SpriteItem{UnitID: o.UnitID, Name: u.Name, GraphicID: int(u.StandingGraphic1), GraphicFile: g.FileName.Value, Variant: v.Index, Frame: ordinal, Status: "drawn"}
		if renderErr != nil {
			item.Status = "skipped"
			item.Reason = renderErr.Error()
		}
		r.Items = append(r.Items, item)
		for j, context := range e.Contexts {
			k := i*len(e.Contexts) + j
			x, y := (k%cols)*cellW, (k/cols)*cellH
			drawSpriteFit(canvas, backgrounds[context], image.Rect(x+4, y+4, x+cellW-4, y+152))
			if sprite != nil {
				drawGrammarSprite(canvas, sprite, image.Rect(x+12, y+4, x+cellW-12, y+148))
			}
			largeGrammarLabel(canvas, x+8, y+155, fmt.Sprintf("%d v%d %s", o.UnitID, v.Index, context))
			label := v.Role
			if v.Axis != "" {
				label += "/" + v.Axis
			}
			if renderErr != nil {
				label = "SPRITE UNAVAILABLE"
			}
			largeGrammarLabel(canvas, x+8, y+183, label)
		}
	}
	return canvas, r, nil
}

func drawGrammarSprite(dst, src *image.RGBA, target image.Rectangle) {
	b := image.Rectangle{}
	for y := src.Bounds().Min.Y; y < src.Bounds().Max.Y; y++ {
		for x := src.Bounds().Min.X; x < src.Bounds().Max.X; x++ {
			if src.RGBAAt(x, y).A != 0 {
				b = b.Union(image.Rect(x, y, x+1, y+1))
			}
		}
	}
	if b.Empty() {
		return
	}
	w, h := b.Dx(), b.Dy()
	for w*2 <= target.Dx() && h*2 <= target.Dy() && w < 4*b.Dx() {
		w *= 2
		h *= 2
	}
	if w > target.Dx() {
		h = h * target.Dx() / w
		w = target.Dx()
	}
	if h > target.Dy() {
		w = w * target.Dy() / h
		h = target.Dy()
	}
	x, y := target.Min.X+(target.Dx()-w)/2, target.Min.Y+(target.Dy()-h)/2
	xdraw.NearestNeighbor.Scale(dst, image.Rect(x, y, x+max(1, w), y+max(1, h)), src, b, draw.Over, nil)
}

func largeGrammarLabel(dst *image.RGBA, x, y int, text string) {
	if len(text) > 16 {
		text = text[:16]
	}
	tmp := image.NewRGBA(image.Rect(0, 0, 7*len(text), 14))
	d := font.Drawer{Dst: tmp, Src: image.NewUniform(color.RGBA{245, 245, 245, 255}), Face: basicfont.Face7x13, Dot: fixed.P(0, 11)}
	d.DrawString(text)
	for ty := 0; ty < tmp.Bounds().Dy(); ty++ {
		for tx := 0; tx < tmp.Bounds().Dx(); tx++ {
			c := tmp.RGBAAt(tx, ty)
			if c.A == 0 {
				continue
			}
			for dy := 0; dy < 2; dy++ {
				for dx := 0; dx < 2; dx++ {
					dst.SetRGBA(x+tx*2+dx, y+ty*2+dy, c)
				}
			}
		}
	}
}
