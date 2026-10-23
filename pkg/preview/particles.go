package preview

import (
	"encoding/json"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"

	"aoe2kit/pkg/gfx"
	xdraw "golang.org/x/image/draw"
)

// ParticleReport distinguishes an explicit DAT reference from the as-yet
// uncalibrated angle/anchor convention used to preview a static loop frame.
type ParticleReport struct {
	UnitID    int     `json:"unit_id"`
	X         float64 `json:"x"`
	Y         float64 `json:"y"`
	GraphicID int     `json:"graphic_id"`
	Name      string  `json:"particle"`
	Mapping   string  `json:"mapping_evidence"`
	Status    string  `json:"status"`
	Reason    string  `json:"reason,omitempty"`
	Layer     string  `json:"layer,omitempty"`
	Angle     int     `json:"angle"`
	Angles    int     `json:"angles,omitempty"`
	Frame     int     `json:"atlas_frame"`
	Filename  string  `json:"frame_name,omitempty"`
	Scale     float64 `json:"scale,omitempty"`
	Placement string  `json:"placement_evidence"`
}

type particleDefinition struct {
	AtlasFile   string
	Type        string
	ImageFirst  int
	ImageCount  int
	ImageAngles int
	Scale       float64
	Layer       string
}

type particleRect struct{ X, Y, W, H int }
type particleSize struct{ W, H int }
type particlePivot struct{ X, Y float64 }
type particleAtlasFrame struct {
	Filename         string         `json:"filename"`
	Frame            particleRect   `json:"frame"`
	Rotated          bool           `json:"rotated"`
	SpriteSourceSize particleRect   `json:"spriteSourceSize"`
	SourceSize       particleSize   `json:"sourceSize"`
	Pivot            *particlePivot `json:"pivot"`
}
type particleAtlas struct {
	Frames []particleAtlasFrame `json:"frames"`
}
type particleFrame struct {
	image              *image.RGBA // cropped pixels only; never retains an evicted atlas
	hotspotX, hotspotY float64
	meta               particleAtlasFrame
	ordinal            int
}
type particleLoader struct {
	root                     string
	definitions              map[string]particleDefinition
	atlases                  map[string]particleAtlas
	textures                 map[string]*image.RGBA
	frames                   map[string]particleFrame
	textureBytes, frameBytes int
}

func newParticleLoader(gameDir string) *particleLoader {
	return &particleLoader{root: filepath.Join(gameDir, "resources", "_common", "particles"), definitions: map[string]particleDefinition{}, atlases: map[string]particleAtlas{}, textures: map[string]*image.RGBA{}, frames: map[string]particleFrame{}}
}

// Definitions contain Windows-style paths even on Unix. Resolve only assets
// within the particle directory; reject traversal, drives and escaping links.
func particleAssetPath(root, name string) (string, error) {
	name = strings.ReplaceAll(name, "\\", "/")
	clean := filepath.Clean(filepath.FromSlash(name))
	if name == "" || filepath.IsAbs(clean) || strings.Contains(name, ":") || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("invalid particle asset path %q", name)
	}
	base, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", err
	}
	path, err := filepath.EvalSymlinks(filepath.Join(root, clean))
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(base, path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("particle asset escapes particle directory")
	}
	return path, nil
}

func readParticleJSON(path string, v any) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	b, err := io.ReadAll(io.LimitReader(f, 8*1024*1024+1))
	if err != nil {
		return err
	}
	if len(b) > 8*1024*1024 {
		return fmt.Errorf("particle JSON exceeds 8 MiB")
	}
	return json.Unmarshal(b, v)
}

func (l *particleLoader) definition(name string) (particleDefinition, error) {
	if d, ok := l.definitions[name]; ok {
		return d, nil
	}
	var d particleDefinition
	path, err := particleAssetPath(l.root, name+".json")
	if err != nil {
		return d, err
	}
	if err = readParticleJSON(path, &d); err != nil {
		return d, err
	}
	if d.Type != "Loop" {
		return d, fmt.Errorf("unsupported particle Type %q (need Loop)", d.Type)
	}
	if d.Layer != "" && d.Layer != "Terrain" {
		return d, fmt.Errorf("unsupported particle Layer %q", d.Layer)
	}
	if d.ImageFirst < 0 || d.ImageCount <= 0 || d.ImageCount > 100000 || d.ImageAngles <= 0 || d.ImageCount%d.ImageAngles != 0 || d.ImageFirst > 100000-d.ImageCount {
		return d, fmt.Errorf("invalid particle image range or angle count")
	}
	if math.IsNaN(d.Scale) || math.IsInf(d.Scale, 0) || d.Scale <= 0 || d.Scale > 16 {
		return d, fmt.Errorf("particle Scale must be in (0,16]")
	}
	l.definitions[name] = d
	return d, nil
}

func (l *particleLoader) texture(path string) (*image.RGBA, error) {
	if img := l.textures[path]; img != nil {
		return img, nil
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var img *image.RGBA
	switch strings.ToLower(filepath.Ext(path)) {
	case ".dds":
		img, err = gfx.DecodeDDS(f)
	case ".png":
		var config image.Config
		config, err = png.DecodeConfig(f)
		if err != nil {
			return nil, err
		}
		if config.Width < 1 || config.Height < 1 || config.Width > 4096 || config.Height > 4096 {
			return nil, fmt.Errorf("particle PNG dimensions exceed 4096 per axis")
		}
		if _, err = f.Seek(0, io.SeekStart); err != nil {
			return nil, err
		}
		var decoded image.Image
		decoded, err = png.Decode(f)
		if err == nil {
			img = image.NewRGBA(decoded.Bounds())
			draw.Draw(img, img.Bounds(), decoded, decoded.Bounds().Min, draw.Src)
		}
	default:
		return nil, fmt.Errorf("unsupported particle atlas image extension")
	}
	if err != nil {
		return nil, err
	}
	if l.textureBytes+len(img.Pix) > 128*1024*1024 {
		clear(l.textures)
		l.textureBytes = 0
	}
	l.textures[path] = img
	l.textureBytes += len(img.Pix)
	return img, nil
}

func (l *particleLoader) frame(d particleDefinition, angle int) (particleFrame, error) {
	if angle < 0 || angle >= d.ImageAngles {
		return particleFrame{}, fmt.Errorf("particle angle %d outside 0..%d", angle, d.ImageAngles-1)
	}
	ordinal := d.ImageFirst + angle*(d.ImageCount/d.ImageAngles)
	key := fmt.Sprintf("%s/%d", d.AtlasFile, ordinal)
	path, err := particleAssetPath(l.root, d.AtlasFile)
	if err != nil {
		return particleFrame{}, err
	}
	metaPath := strings.TrimSuffix(d.AtlasFile, filepath.Ext(d.AtlasFile)) + ".json"
	atlas, ok := l.atlases[metaPath]
	if !ok {
		p, err := particleAssetPath(l.root, metaPath)
		if err != nil {
			return particleFrame{}, err
		}
		if err = readParticleJSON(p, &atlas); err != nil {
			return particleFrame{}, err
		}
		l.atlases[metaPath] = atlas
	}
	if d.ImageFirst+d.ImageCount > len(atlas.Frames) {
		return particleFrame{}, fmt.Errorf("particle range exceeds %d atlas frames", len(atlas.Frames))
	}
	if frame, ok := l.frames[key]; ok {
		return frame, nil
	}
	f := atlas.Frames[ordinal]
	if f.Pivot == nil || math.IsNaN(f.Pivot.X) || math.IsNaN(f.Pivot.Y) || f.Pivot.X < 0 || f.Pivot.X > 1 || f.Pivot.Y < 0 || f.Pivot.Y > 1 {
		return particleFrame{}, fmt.Errorf("missing or invalid atlas pivot")
	}
	if f.SourceSize.W <= 0 || f.SourceSize.H <= 0 || f.SourceSize.W > 4096 || f.SourceSize.H > 4096 {
		return particleFrame{}, fmt.Errorf("invalid source canvas dimensions")
	}
	r, trim := f.Frame, f.SpriteSourceSize
	// The installed JSON array uses unrotated frame w/h, even when packed
	// clockwise. Swap the sampled footprint, not the trim or source canvas.
	packedW, packedH := r.W, r.H
	if f.Rotated {
		packedW, packedH = r.H, r.W
	}
	if r.X < 0 || r.Y < 0 || r.W <= 0 || r.H <= 0 || r.X > 4096-packedW || r.Y > 4096-packedH || trim.X < 0 || trim.Y < 0 || trim.W != r.W || trim.H != r.H || trim.X > f.SourceSize.W-trim.W || trim.Y > f.SourceSize.H-trim.H {
		return particleFrame{}, fmt.Errorf("invalid atlas or trimmed frame rectangle")
	}
	atlasImage, err := l.texture(path)
	if err != nil {
		return particleFrame{}, err
	}
	if r.X+packedW > atlasImage.Bounds().Dx() || r.Y+packedH > atlasImage.Bounds().Dy() {
		return particleFrame{}, fmt.Errorf("frame rectangle outside atlas image")
	}
	img := image.NewRGBA(image.Rect(0, 0, r.W, r.H))
	if f.Rotated {
		for y := 0; y < r.H; y++ {
			for x := 0; x < r.W; x++ {
				img.SetRGBA(x, y, atlasImage.RGBAAt(r.X+r.H-1-y, r.Y+x))
			}
		}
	} else {
		draw.Draw(img, img.Bounds(), atlasImage, image.Pt(r.X, r.Y), draw.Src)
	}
	frame := particleFrame{image: img, hotspotX: f.Pivot.X*float64(f.SourceSize.W) - float64(trim.X), hotspotY: f.Pivot.Y*float64(f.SourceSize.H) - float64(trim.Y), meta: f, ordinal: ordinal}
	if l.frameBytes+len(img.Pix) > 64*1024*1024 {
		clear(l.frames)
		l.frameBytes = 0
	}
	l.frames[key] = frame
	l.frameBytes += len(img.Pix)
	return frame, nil
}

type particleObject struct {
	definition  particleDefinition
	reportIndex int
}

func drawParticle(base *image.RGBA, loader *particleLoader, o Object, p particleObject, crop Crop, scale, nativeTile int, report *Report) {
	r := &report.Particles[p.reportIndex]
	f, err := loader.frame(p.definition, r.Angle)
	if err != nil {
		r.Status = "skipped"
		r.Reason = err.Error()
		return
	}
	r.Frame = f.ordinal
	r.Filename = f.meta.Filename
	factor := p.definition.Scale * float64(scale) / float64(nativeTile)
	x, y := mapPoint(o.X-float64(crop.X1), o.Y-float64(crop.Y1), crop, scale, "diamond")
	y += report.ElevationPadding
	left, top := x-int(math.Round(f.hotspotX*factor)), y-int(math.Round(f.hotspotY*factor))
	rect := image.Rect(left, top, left+max(1, int(math.Round(float64(f.image.Bounds().Dx())*factor))), top+max(1, int(math.Round(float64(f.image.Bounds().Dy())*factor))))
	xdraw.CatmullRom.Scale(base, rect, f.image, f.image.Bounds(), draw.Over, nil)
	r.Status = "drawn"
	report.Drawn++
	report.CountsByClass[o.Class]++
}
