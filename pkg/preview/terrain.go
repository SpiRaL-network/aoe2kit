package preview

import (
	"crypto/sha256"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"math"
	"os"
	"sort"
	"strings"

	"aoe2kit/pkg/datfile"
	"aoe2kit/pkg/gameinstall"
	"aoe2kit/pkg/gfx"
	xdraw "golang.org/x/image/draw"
)

// TerrainTextureOptions opts in to installed art. Repeat is in world tiles,
// independent of DDS dimensions and crop. Zero selects the candidate 9 tiles.
type TerrainTextureOptions struct {
	GameDir       string
	Repeat        float64 // Mask repeat in world tiles.
	TextureRepeat float64
	Edges         bool
}

type TerrainAssetReport struct {
	ID              int      `json:"terrain_id"`
	Texture         string   `json:"texture"`
	TextureWidth    int      `json:"texture_width,omitempty"`
	TextureHeight   int      `json:"texture_height,omitempty"`
	Mask            string   `json:"mask,omitempty"`
	MaskTopFraction *float64 `json:"mask_top_fraction,omitempty"`
	BlendType       int32    `json:"blend_type"`
	BlendPriority   int32    `json:"blend_priority"`
	BlendFamily     string   `json:"candidate_blend_family"`
	Issues          []string `json:"issues,omitempty"`
}

type TerrainTextureReport struct {
	Repeat            float64              `json:"mask_repeat_tiles"`
	TextureRepeat     float64              `json:"texture_repeat_tiles"`
	LayerRule         string               `json:"layer_rule"`
	EmptyMaskRule     string               `json:"empty_mask_rule"`
	EdgeRule          string               `json:"edge_rule"`
	EdgeEvidence      []string             `json:"edge_evidence,omitempty"`
	BlendomaticSHA256 string               `json:"blendomatic_sha256,omitempty"`
	BlendomaticModes  int                  `json:"blendomatic_modes,omitempty"`
	EdgeIssue         string               `json:"edge_issue,omitempty"`
	Assumptions       []string             `json:"assumptions"`
	Assets            []TerrainAssetReport `json:"assets"`
}

type terrainMaterial struct {
	texture   *image.RGBA
	mask      *image.Gray
	blends    []gfx.BlendomaticMode
	invert    bool
	priority  int32
	blendType int32
	fallback  color.RGBA
}

// Candidate families from DAT type/name groups, not engine-verified atlas IDs.
func terrainBlendFamily(kind int32) string {
	return map[int32]string{0: "landland", 1: "farmland", 2: "watershore", 3: "waterwater", 4: "shallowswater", 5: "roadland", 6: "icewater", 7: "snowland"}[kind]
}

type terrainAssets struct {
	opts         TerrainTextureOptions
	textureLimit int
	images       map[string]*image.RGBA
	grays        map[string]*image.Gray
	errors       map[string]error
	bytes        int
}

func (a *terrainAssets) load(group, name string) (*image.RGBA, error) {
	key := group + "/" + name
	if img, ok := a.images[key]; ok {
		return img, nil
	}
	if err, ok := a.errors[key]; ok {
		return nil, err
	}
	img, err := a.read(group, name)
	if err != nil {
		a.errors[key] = err
		return nil, err
	}
	if a.bytes+len(img.Pix) > 128*1024*1024 {
		err = fmt.Errorf("terrain asset cache exceeds 128 MiB")
		a.errors[key] = err
		return nil, err
	}
	a.bytes += len(img.Pix)
	a.images[key] = img
	return img, nil
}

func (a *terrainAssets) read(group, name string) (*image.RGBA, error) {
	path, err := gameinstall.TerrainAssetPath(a.opts.GameDir, group, name)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var src image.Image
	if group == "textures/2x" {
		src, err = gfx.DecodeDDS(f)
	} else {
		var cfg image.Config
		cfg, err = png.DecodeConfig(f)
		if err == nil && (cfg.Width < 1 || cfg.Height < 1 || cfg.Width > 2048 || cfg.Height > 2048) {
			err = fmt.Errorf("terrain PNG exceeds 2048 per axis")
		}
		if err == nil {
			_, err = f.Seek(0, 0)
		}
		if err == nil {
			src, err = png.Decode(f)
		}
	}
	if err != nil {
		return nil, err
	}
	b := src.Bounds()
	limit := 512
	if group == "textures/2x" && a.textureLimit > 0 {
		limit = a.textureLimit
	}
	dst := image.NewRGBA(image.Rect(0, 0, min(limit, b.Dx()), min(limit, b.Dy())))
	xdraw.ApproxBiLinear.Scale(dst, dst.Bounds(), src, b, draw.Src, nil)
	return dst, nil
}

func (a *terrainAssets) gray(group, name string) (*image.Gray, error) {
	key := group + "/" + name
	if img, ok := a.grays[key]; ok {
		return img, nil
	}
	src, err := a.load(group, name)
	if err != nil {
		return nil, err
	}
	img := image.NewGray(src.Bounds())
	if a.bytes+len(img.Pix) > 128*1024*1024 {
		return nil, fmt.Errorf("terrain asset cache exceeds 128 MiB")
	}
	draw.Draw(img, img.Bounds(), src, image.Point{}, draw.Src)
	a.bytes += len(img.Pix)
	a.grays[key] = img
	return img, nil
}

func loadTerrainMaterials(tiles []Tile, s Snapshot, crop Crop, dat *datfile.Index, opts TerrainTextureOptions, palette []color.RGBA, scale int) (map[int]terrainMaterial, TerrainTextureReport) {
	limit := 512
	for limit < 2048 && float64(limit) < float64(scale)*opts.TextureRepeat {
		limit *= 2
	}
	r := TerrainTextureReport{Repeat: opts.Repeat, TextureRepeat: opts.TextureRepeat,
		LayerRule:     "top blended over base with alpha = named mask value; mask UV = world (x, y) / repeat, anchored at the map origin, no flip; fitted to marker-registered editor screenshots of road over grass and snow over dirt (96.7% and 98.2% pixel agreement)",
		EmptyMaskRule: "UNVERIFIED: inverse default_strong as alpha; closest observed black-on-azure coverage, not an engine rule",
		EdgeRule:      "disabled",
		Assumptions:   []string{"mask repeat defaults to 10 tiles, origin-anchored (fitted on road over grass; other pairs not yet fitted); texture repeat independently defaults to 9 and is not calibrated", fmt.Sprintf("textures sampled at at most %dx%d; first mip, static water, no shaders/lighting/geometric elevation", limit, limit)}}
	var blends []gfx.BlendomaticMode
	if opts.Edges {
		r.EdgeRule = "CANDIDATE: installed blendomatic diamonds selected by eight-neighbor topology; stored weight retains base, higher DAT priority draws last. Classic DAT-type pair-to-mode mapping and coordinate variants remain candidates for DE. Road single/ring outlines are closer to the probe, but the diagonal pinch is weaker; water mode 1 leaves a ring opening contrary to the human observation. No radial footprints. DE blend textures/shader are not reproduced."
		r.Assumptions = append(r.Assumptions, "blendomatic flags and packed bit masks are preserved by the decoder, not used for shading; only normalized byte weights are sampled")
		r.EdgeEvidence = []string{
			"editor screenshots: a single shallow-water tile spreads softly beyond its diamond onto grass; water closes the one-tile hole in a 3x3 ring (human report)",
			"editor screenshots: a single road tile spreads as a rounded frayed patch over grass; diagonal road tiles merge; a 3x3 road ring leaves a small grass speck",
			"larger road/snow layer-pair screenshots support frayed crisp edges, not exact footprint parameters",
		}
		f, err := os.Open(gameinstall.BlendomaticPath(opts.GameDir))
		if err == nil {
			h := sha256.New()
			blends, err = gfx.DecodeBlendomatic(io.TeeReader(f, h))
			f.Close()
			if err == nil {
				r.BlendomaticSHA256 = fmt.Sprintf("%x", h.Sum(nil))
				r.BlendomaticModes = len(blends)
			}
		}
		if err != nil {
			r.EdgeIssue = "blendomatic: " + err.Error() + "; hard borders retained"
		}
	}
	a := terrainAssets{opts: opts, textureLimit: limit, images: map[string]*image.RGBA{}, grays: map[string]*image.Gray{}, errors: map[string]error{}}
	ids, layered := map[int]bool{}, map[int]bool{}
	for y := max(0, crop.Y1-1); y < min(s.Height, crop.Y2+1); y++ {
		for x := max(0, crop.X1-1); x < min(s.Width, crop.X2+1); x++ {
			t := tiles[y*s.Width+x]
			ids[t.TerrainID] = true
			if t.Layer != nil {
				ids[*t.Layer] = true
				layered[*t.Layer] = true
			}
		}
	}
	ordered := make([]int, 0, len(ids))
	for id := range ids {
		ordered = append(ordered, id)
	}
	sort.Ints(ordered)
	materials := map[int]terrainMaterial{}
	for _, id := range ordered {
		m := terrainMaterial{fallback: minimapTerrainColor(id, dat, palette), blends: blends}
		ar := TerrainAssetReport{ID: id}
		if id < 0 || id >= len(dat.Terrains) {
			ar.Issues = append(ar.Issues, "terrain missing from DAT; flat fallback")
			materials[id] = m
			r.Assets = append(r.Assets, ar)
			continue
		}
		t := dat.Terrains[id]
		m.priority = t.BlendPriority
		m.blendType = t.BlendType
		ar.Texture = t.Name2.Value
		ar.BlendPriority = t.BlendPriority
		ar.BlendType = t.BlendType
		ar.BlendFamily = terrainBlendFamily(t.BlendType)
		name := ar.Texture
		if !strings.HasSuffix(strings.ToLower(name), ".dds") {
			name += ".dds"
		}
		var err error
		m.texture, err = a.load("textures/2x", name)
		if err != nil {
			ar.Issues = append(ar.Issues, "texture: "+err.Error()+"; flat fallback")
		} else {
			ar.TextureWidth = m.texture.Bounds().Dx()
			ar.TextureHeight = m.texture.Bounds().Dy()
		}
		if layered[id] {
			ar.Mask = t.OverlayMaskName.Value
			if ar.Mask == "" {
				ar.Mask = "default_strong.png"
				m.invert = true
				ar.Issues = append(ar.Issues, "empty DAT mask; UNVERIFIED inverted default_strong assumption")
			}
			m.mask, err = a.gray("masks", ar.Mask)
			if err != nil {
				ar.Issues = append(ar.Issues, "mask: "+err.Error()+"; base retained")
			} else {
				sum := 0
				for _, v := range m.mask.Pix {
					if m.invert {
						v = 255 - v
					}
					sum += int(v)
				}
				fraction := float64(sum) / 255 / float64(len(m.mask.Pix)) // mean top alpha
				ar.MaskTopFraction = &fraction
			}
		}
		if opts.Edges && (t.BlendType < 0 || t.BlendType > 7) {
			ar.Issues = append(ar.Issues, "unknown blend-type pair mapping; hard border retained")
		}
		if opts.Edges && len(blends) > 0 {
			for _, other := range ordered {
				if other < 0 || other >= len(dat.Terrains) {
					continue
				}
				if mode := blendMode(t.BlendType, dat.Terrains[other].BlendType); mode >= len(blends) {
					ar.Issues = append(ar.Issues, fmt.Sprintf("blend mode %d for pair with terrain %d absent; hard border retained", mode, other))
				}
			}
		}
		materials[id] = m
		r.Assets = append(r.Assets, ar)
	}
	return materials, r
}

func repeatUV(v, repeat float64, size int) int {
	return min(size-1, int((v/repeat-math.Floor(v/repeat))*float64(size)))
}

func materialPixel(m terrainMaterial, x, y, repeat float64) color.RGBA {
	if m.texture == nil {
		return m.fallback
	}
	c := m.texture.RGBAAt(repeatUV(x, repeat, m.texture.Bounds().Dx()), repeatUV(y, repeat, m.texture.Bounds().Dy()))
	a := uint16(255 - c.A)
	return color.RGBA{uint8(uint16(c.R) + uint16(m.fallback.R)*a/255), uint8(uint16(c.G) + uint16(m.fallback.G)*a/255), uint8(uint16(c.B) + uint16(m.fallback.B)*a/255), 255}
}

func mixTerrain(a, b color.RGBA, weight float64) color.RGBA {
	return color.RGBA{uint8(float64(a.R)*(1-weight) + float64(b.R)*weight), uint8(float64(a.G)*(1-weight) + float64(b.G)*weight), uint8(float64(a.B)*(1-weight) + float64(b.B)*weight), 255}
}

func tileMaterial(t Tile) int {
	if t.Layer != nil {
		return *t.Layer
	}
	return t.TerrainID
}

func layeredPixel(t Tile, materials map[int]terrainMaterial, x, y, repeat, textureRepeat float64) color.RGBA {
	base := materialPixel(materials[t.TerrainID], x, y, textureRepeat)
	if t.Layer == nil {
		return base
	}
	top := materials[*t.Layer]
	if top.mask == nil {
		return base
	}
	v := top.mask.GrayAt(repeatUV(x, repeat, top.mask.Bounds().Dx()), repeatUV(y, repeat, top.mask.Bounds().Dy())).Y
	if top.invert {
		v = 255 - v
	}
	switch v {
	case 0:
		return base
	case 255:
		return materialPixel(top, x, y, textureRepeat)
	}
	return mixTerrain(base, materialPixel(top, x, y, textureRepeat), float64(v)/255)
}

// The layer mask rule was fitted against two marker-registered editor
// screenshots with patches at the map origin edge (road over grass, snow over
// dirt). Both give identity orientation, a 10-tile repeat and zero offset from
// the map origin (96.7% and 98.2% of pixels). The top is blended over the base
// with alpha = mask value, not cut at a threshold: the best hard cutoff came
// out per pair (88 road, 180 snow), and a soft-blended preview reproduces the
// same per-pair fit (108, 180) where a hard-cut preview gives 88 for both.
const defaultMaskRepeat = 10

// terrainLight is the engine's flat-ground lighting, which the mood grade is
// applied on top of. Fitted on the swatch board (16 terrains, Default mood,
// per-square medians): after the Default grade the game was brighter by
// 1.12/1.09/1.13 (R/G/B), and the gain took the per-square error from
// 16-22 to 6-12 RMS. Road (24) was the exception, darker in game; excluded.
var terrainLight = [3]float64{1.123, 1.094, 1.130}

func lightTerrain(c color.RGBA) color.RGBA {
	l := func(v uint8, g float64) uint8 { return uint8(math.Min(255, math.Round(float64(v)*g))) }
	return color.RGBA{l(c.R, terrainLight[0]), l(c.G, terrainLight[1]), l(c.B, terrainLight[2]), c.A}
}

func renderTexturedTerrain(img *image.RGBA, tiles []Tile, s Snapshot, crop Crop, scale int, projection string, opts Options) (TerrainTextureReport, error) {
	o := *opts.Textures
	if opts.Dat == nil || o.GameDir == "" {
		return TerrainTextureReport{}, fmt.Errorf("textured terrain needs a DAT and resolved game directory")
	}
	if o.Repeat == 0 {
		o.Repeat = defaultMaskRepeat
	}
	if o.TextureRepeat == 0 {
		o.TextureRepeat = 9
	}
	if math.IsNaN(o.TextureRepeat) || math.IsInf(o.TextureRepeat, 0) || o.TextureRepeat < 1 || o.TextureRepeat > 256 {
		return TerrainTextureReport{}, fmt.Errorf("texture repeat must be finite, between 1 and 256 tiles")
	}
	if math.IsNaN(o.Repeat) || math.IsInf(o.Repeat, 0) || o.Repeat < 1 || o.Repeat > 256 {
		return TerrainTextureReport{}, fmt.Errorf("terrain repeat must be finite, between 1 and 256 tiles")
	}
	mats, r := loadTerrainMaterials(tiles, s, crop, opts.Dat, o, opts.Palette, scale)
	transform := diamondTransform(crop.X2-crop.X1, crop.Y2-crop.Y1, float64(scale), float64(scale), "")
	for py := 0; py < img.Bounds().Dy(); py++ {
		for px := 0; px < img.Bounds().Dx(); px++ {
			x, y := (float64(px)+.5)/float64(scale), (float64(py)+.5)/float64(scale)
			if projection == "diamond" {
				x, y = transform.inverse(float64(px)+.5, float64(py)+.5)
			}
			x += float64(crop.X1)
			y += float64(crop.Y1)
			if x < float64(crop.X1) || y < float64(crop.Y1) || x >= float64(crop.X2) || y >= float64(crop.Y2) {
				continue
			}
			tx, ty := int(x), int(y)
			t := tiles[ty*s.Width+tx]
			c := layeredPixel(t, mats, x, y, o.Repeat, o.TextureRepeat)
			if o.Edges {
				c = terrainEdgePixel(tiles, s.Width, s.Height, mats, x, y, o)
			}
			img.SetRGBA(px, py, lightTerrain(c))
		}
	}
	return r, nil
}
