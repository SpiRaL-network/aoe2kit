package preview

import (
	"crypto/sha256"
	"encoding/hex"

	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math"

	xdraw "golang.org/x/image/draw"

	"aoe2kit/pkg/datfile"
	"aoe2kit/pkg/gameinstall"
	"aoe2kit/pkg/gfx"
)

type SpriteItem struct {
	UnitID      int      `json:"unit_id"`
	Name        string   `json:"name"`
	GraphicID   int      `json:"graphic_id"`
	GraphicFile string   `json:"graphic_file"`
	Variant     int      `json:"variant"`
	Frame       int      `json:"frame"`
	States      []string `json:"states,omitempty"`
	Status      string   `json:"status"`
	Reason      string   `json:"reason,omitempty"`
}

// angleFrameOrdinal maps a unit's rotation to an SLD frame. SLD frames are
// stored angle by angle, frameCount frames per angle, so angle a starts at
// a*frameCount. Static art (one frame per angle) keeps the rotation as the
// frame, matching the engine-verified oak fact; animated units (a camel has
// 16 angles x 60 frames) get the first frame of their facing. Out-of-range
// rotations are not wrapped, so the renderer still reports them.
func angleFrameOrdinal(rotation, angleCount, frameCount int) int {
	if frameCount <= 1 || angleCount <= 1 || rotation < 0 {
		return rotation
	}
	if rotation >= angleCount {
		return -1 // past the last angle: reported as unavailable
	}
	return rotation * frameCount
}

func drawSpriteFit(dst *image.RGBA, src image.Image, target image.Rectangle) {
	b := src.Bounds()
	if b.Empty() {
		return
	}
	scale := math.Min(float64(target.Dx())/float64(b.Dx()), float64(target.Dy())/float64(b.Dy()))
	if scale > 1 {
		scale = 1
	}
	w, h := max(1, int(float64(b.Dx())*scale)), max(1, int(float64(b.Dy())*scale))
	x, y := target.Min.X+(target.Dx()-w)/2, target.Min.Y+(target.Dy()-h)/2
	if w == b.Dx() && h == b.Dy() {
		draw.Draw(dst, image.Rect(x, y, x+w, y+h), src, b.Min, draw.Over)
		return
	}
	xdraw.CatmullRom.Scale(dst, image.Rect(x, y, x+w, y+h), src, b, draw.Over, nil)
}

// RenderSprites overlays individual sprites onto the terrain preview in
// painter order. It loads only graphics referenced by objects in this snapshot.
func RenderSprites(s Snapshot, opts Options, idx *datfile.Index, gameDir string) (*image.RGBA, Report, error) {
	model, err := ResolveRotationModel(opts.RotationModel)
	if err != nil {
		return nil, Report{}, err
	}
	if idx == nil {
		return nil, Report{}, fmt.Errorf("sprite preview requires a DAT")
	}
	if defaultProjection(opts.Projection) != "diamond" {
		return nil, Report{}, errors.New("sprite preview requires diamond projection")
	}
	nativeTile := opts.SpriteTilePixels
	if nativeTile == 0 {
		nativeTile = 96
	}
	if nativeTile < 1 || nativeTile > 1024 {
		return nil, Report{}, errors.New("sprite tile width must be between 1 and 1024 pixels")
	}
	var coverage *image.Alpha
	overlap := ""
	if opts.Shadows != nil {
		var err error
		overlap, err = shadowOverlap(*opts.Shadows)
		if err != nil {
			return nil, Report{}, err
		}
	}
	base, report, err := RenderFlat(s, Options{Projection: opts.Projection, Scale: opts.Scale, Layers: map[string]bool{"terrain": layerEnabled(opts.Layers, "terrain"), "elevation": layerEnabled(opts.Layers, "elevation"), "starts": false}, Crop: opts.Crop, Dat: idx, Palette: opts.Palette, Textures: opts.Textures, Elevation: opts.Elevation})
	if err != nil {
		return nil, Report{}, err
	}
	if gameDir == "" {
		return nil, Report{}, fmt.Errorf("sprite preview requires a game install")
	}
	report.Skipped = nil
	report.RotationModel, report.ScenarioVersion = model, s.ScenarioVersion
	if model != "stored-index" {
		report.Fallbacks = append(report.Fallbacks, "experimental rotation model: "+model+"; not engine-verified; raw scenario and snapshot values are unchanged")
	}
	report.Drawn = 0
	report.CountsByClass = map[string]int{}
	crop := Crop{X2: s.Width, Y2: s.Height}
	if opts.Crop != nil {
		crop = *opts.Crop
	}
	scale := opts.Scale
	if scale == 0 {
		scale = 4
	}
	objects := depthSortedObjects(s.Objects, crop, scale, defaultProjection(opts.Projection))
	particleAssets := newParticleLoader(gameDir)
	particleObjects := map[int]particleObject{}
	particleHasSLD := map[int]bool{}
	if layerEnabled(opts.Layers, "objects") {
		for i, o := range objects {
			u, ok := unitMetadata(idx, 0, o.UnitID)
			if !ok {
				continue
			}
			g, ok := idx.Graphic(int(u.StandingGraphic1))
			if !ok || g.ParticleEffectName.Value == "" {
				continue
			}
			angle := o.Variant
			if angle == 0 {
				angle = int(math.Round(o.Rotation))
			}
			r := ParticleReport{UnitID: o.UnitID, X: o.X, Y: o.Y, GraphicID: int(u.StandingGraphic1), Name: g.ParticleEffectName.Value, Mapping: "DAT graphic.ParticleEffectName -> installed particle JSON", Status: "pending", Angle: angle, Frame: -1, Placement: "candidate rotation model " + model + ", first frame per angle, source-canvas pivot at unit anchor; not engine-calibrated"}
			hybrid, checked := particleHasSLD[r.GraphicID]
			if !checked {
				if g.FileName.Value != "" {
					_, err := gameinstall.FindLooseSLD(gameDir, g.FileName.Value)
					hybrid = err == nil
				}
				particleHasSLD[r.GraphicID] = hybrid
			}
			if hybrid {
				r.Status, r.Reason = "skipped", "hybrid SLD and particle composition unsupported; existing SLD retained"
				report.Particles = append(report.Particles, r)
				continue
			}
			p := particleObject{reportIndex: len(report.Particles)}
			if o.X < float64(crop.X1) || o.Y < float64(crop.Y1) || o.X >= float64(crop.X2) || o.Y >= float64(crop.Y2) {
				r.Status, r.Reason = "skipped", "outside crop"
			} else {
				d, err := particleAssets.definition(r.Name)
				if err != nil {
					r.Status, r.Reason = "skipped", err.Error()
				} else {
					p.definition = d
					r.Layer, r.Angles, r.Scale = d.Layer, d.ImageAngles, d.Scale
					mapped, e := RotationIndex(o, model, d.ImageAngles)
					if e != nil {
						r.Status, r.Reason = "skipped", e.Error()
					} else {
						r.Angle = mapped
					}
					if r.Layer == "" {
						r.Layer = "Sprite"
					}
				}
			}
			report.Particles = append(report.Particles, r)
			particleObjects[i] = p
			if r.Status == "pending" && p.definition.Layer == "Terrain" {
				drawParticle(base, particleAssets, o, p, crop, scale, nativeTile, &report)
			}
		}
	}
	loaded := map[int]*gfx.SLD{}
	loadedBytes := 0
	paths := map[int]string{}
	pathErrors := map[int]error{}
	type frameKey struct {
		graphic, variant int
		color            color.RGBA
	}
	type cachedFrame struct {
		image *image.RGBA
		frame gfx.SLDFrame
	}
	decoded := map[frameKey]cachedFrame{}
	decodedBytes := 0
	passes := 1
	if opts.Shadows != nil {
		passes = 2
		coverage = image.NewAlpha(base.Bounds())
		report.Shadows = &ShadowReport{Overlap: overlap, Mood: "none", Limitations: []string{"experimental: overlap blend state not engine-verified", "terrain receives shadows; sprite receiver metadata is not composed", "casters with anchors outside the crop are omitted"}}
		if opts.Shadows.Mood != nil {
			report.Shadows.Mood = opts.Shadows.Mood.Name
		}
	}
	for pass := 0; pass < passes; pass++ {
		shadowPass := passes == 2 && pass == 0
		// Keep bounded frame caches separate between the two passes.
		clear(decoded)
		decodedBytes = 0
		for i, o := range objects {
			if p, ok := particleObjects[i]; ok {
				if !shadowPass && report.Particles[p.reportIndex].Status == "pending" {
					drawParticle(base, particleAssets, o, p, crop, scale, nativeTile, &report)
				}
				continue
			}
			if !layerEnabled(opts.Layers, "objects") {
				continue
			}
			if opts.Crop != nil && (o.X < float64(opts.Crop.X1) || o.Y < float64(opts.Crop.Y1) || o.X >= float64(opts.Crop.X2) || o.Y >= float64(opts.Crop.Y2)) {
				report.Skipped = append(report.Skipped, Skipped{UnitID: o.UnitID, Reason: "outside crop"})
				continue
			}
			u, ok := unitMetadata(idx, 0, o.UnitID)
			if !ok {
				report.Skipped = append(report.Skipped, Skipped{UnitID: o.UnitID, Reason: "unit absent from Gaia DAT"})
				continue
			}
			graphicID := int(u.StandingGraphic1)
			g, ok := idx.Graphic(graphicID)
			if !ok {
				report.Skipped = append(report.Skipped, Skipped{UnitID: o.UnitID, Reason: "standing graphic absent from DAT"})
				continue
			}
			path, known := paths[graphicID]
			e := pathErrors[graphicID]
			if !known && e == nil {
				path, e = gameinstall.FindLooseSLD(gameDir, g.FileName.Value)
				if e != nil {
					pathErrors[graphicID] = e
				} else {
					paths[graphicID] = path
				}
			}
			if e != nil {
				report.Skipped = append(report.Skipped, Skipped{UnitID: o.UnitID, Reason: "missing sprite: " + e.Error()})
				continue
			}
			sld := loaded[graphicID]
			if sld == nil {
				var e error
				sld, e = gfx.OpenSLD(path)
				if e != nil {
					report.Skipped = append(report.Skipped, Skipped{UnitID: o.UnitID, Reason: "SLD decode: " + e.Error()})
					continue
				}
				if sld.Size > 0 && loadedBytes+sld.Size <= 128*1024*1024 {
					loaded[graphicID] = sld
					loadedBytes += sld.Size
				}
			}
			variant, rotationErr := RotationIndex(o, model, int(g.AngleCount))
			if rotationErr != nil {
				report.Skipped = append(report.Skipped, Skipped{UnitID: o.UnitID, Reason: rotationErr.Error()})
				continue
			}
			variant = angleFrameOrdinal(variant, int(g.AngleCount), int(g.FrameCount))
			pc := playerColor(o.Player)
			if index, ok := s.PlayerColors[o.Player]; ok && o.Player > 0 {
				pc = playerColor(index + 1)
			}
			key := frameKey{graphicID, variant, pc}
			cached, exists := decoded[key]
			if !exists {
				var img *image.RGBA
				var frame gfx.SLDFrame
				var e error
				if shadowPass {
					img, frame, e = sld.RenderShadowFrame(variant)
				} else {
					img, frame, e = sld.RenderFrame(variant, pc)
				}
				if e != nil {
					report.Skipped = append(report.Skipped, Skipped{UnitID: o.UnitID, Reason: fmt.Sprintf("visual variant %d unavailable: %v", variant, e)})
					continue
				}
				cached = cachedFrame{img, frame}
				if img == nil {
					decoded[key] = cached
				} else if decodedBytes+len(img.Pix) <= 64*1024*1024 {
					decoded[key] = cached
					decodedBytes += len(img.Pix)
				}
			}
			img, frame := cached.image, cached.frame
			if img == nil {
				report.Shadows.Missing++
				continue
			}
			factor := float64(scale) / float64(nativeTile)
			x, y := mapPoint(o.X-float64(crop.X1), o.Y-float64(crop.Y1), crop, scale, defaultProjection(opts.Projection))
			left, top := x-int(math.Round(float64(frame.HotspotX)*factor)), y-int(math.Round(float64(frame.HotspotY)*factor))
			dest := image.Rect(left, top, left+max(1, int(math.Round(float64(img.Bounds().Dx())*factor))), top+max(1, int(math.Round(float64(img.Bounds().Dy())*factor))))
			if shadowPass {
				accumulateShadow(coverage, img, dest, overlap)
				report.Shadows.Drawn++
				continue
			}
			xdraw.CatmullRom.Scale(base, dest, img, img.Bounds(), draw.Over, nil)
			report.Drawn++
			report.CountsByClass[o.Class]++
		}
		if shadowPass {
			report.Shadows.Skipped = report.Skipped
			report.Skipped = nil
			report.Shadows.CoveredPixels = applyShadows(base, coverage, opts.Shadows.Mood)
		}
	}
	report.Projection = defaultProjection(opts.Projection)
	for _, p := range report.Particles {
		if p.Status == "skipped" {
			report.Skipped = append(report.Skipped, Skipped{UnitID: p.UnitID, Reason: "particle: " + p.Reason})
		}
	}
	if len(report.Particles) > 0 {
		report.Fallbacks = append(report.Fallbacks, "particles: static first loop frame; rotation and pivot convention await engine calibration; emitter simulation, fog and particle shadows are not composed")
	}
	report.ImageWidth = base.Bounds().Dx()
	report.ImageHeight = base.Bounds().Dy()
	if layerEnabled(opts.Layers, "starts") {
		paintStarts(base, s.Objects, crop, scale, defaultProjection(opts.Projection))
	}
	report.Fallbacks = append(report.Fallbacks, fmt.Sprintf("sprite scale convention: %d native pixels per tile; not engine-calibrated", nativeTile), "standing graphics from Gaia DAT; building annexes and animation are not composed")
	if opts.Shadows == nil {
		report.Fallbacks = append(report.Fallbacks, "sprite shadows are disabled")
	}
	hash := sha256.Sum256(base.Pix)
	report.PixelSHA256 = hex.EncodeToString(hash[:])
	return base, report, nil
}

func defaultProjection(p string) string {
	if p == "" {
		return "diamond"
	}
	return p
}
func playerColor(p int) color.RGBA {
	if p <= 0 {
		return color.RGBA{}
	}
	return minimapPlayerColors[(p-1)%len(minimapPlayerColors)]
}
