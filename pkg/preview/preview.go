// Package preview produces deterministic, local previews without bundling art.
package preview

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"os"
	"sort"
	"strings"

	"aoe2kit/pkg/datfile"
	"aoe2kit/pkg/replay"
	"aoe2kit/pkg/scenario"
)

const SnapshotSchema = "aoe2kit.preview.snapshot.v1"
const maxImagePixels = 64 * 1024 * 1024

type Tile struct {
	X         int     `json:"x"`
	Y         int     `json:"y"`
	TerrainID int     `json:"terrain_id"`
	Layer     *int    `json:"layer,omitempty"`
	Elevation float64 `json:"elevation"`
}

type Object struct {
	UnitID                int     `json:"unit_id"`
	X                     float64 `json:"x"`
	Y                     float64 `json:"y"`
	Z                     float64 `json:"z,omitempty"`
	Rotation              float64 `json:"rotation"`
	Status                int     `json:"status,omitempty"`
	InitialAnimationFrame int     `json:"initial_animation_frame,omitempty"`
	Variant               int     `json:"variant,omitempty"`
	Player                int     `json:"player"`
	Class                 string  `json:"class"`
	Name                  string  `json:"name,omitempty"`
	FootprintX            float64 `json:"footprint_x,omitempty"`
	FootprintY            float64 `json:"footprint_y,omitempty"`
	Blocking              bool    `json:"blocking,omitempty"`
	MapDrawLevel          uint8   `json:"map_draw_level,omitempty"`
	MapColor              uint8   `json:"map_color,omitempty"`
	HasMapFields          bool    `json:"has_map_fields,omitempty"`
	MinimapMode           int     `json:"minimap_mode,omitempty"`
	HasMinimapMode        bool    `json:"has_minimap_mode,omitempty"`
}

type Snapshot struct {
	ScenarioVersion string      `json:"scenario_version,omitempty"`
	Schema          string      `json:"schema"`
	Stage           string      `json:"stage"`
	Width           int         `json:"width"`
	Height          int         `json:"height"`
	Terrain         []Tile      `json:"terrain"`
	Objects         []Object    `json:"objects,omitempty"`
	PlayerColors    map[int]int `json:"player_colors,omitempty"`
	SourceWarnings  []string    `json:"source_warnings,omitempty"`
}

type Crop struct {
	X1 int `json:"x1"`
	Y1 int `json:"y1"`
	X2 int `json:"x2"`
	Y2 int `json:"y2"`
}

type Options struct {
	Projection         string
	Scale              int
	Layers             map[string]bool
	Crop               *Crop
	MinimapScale       int
	MinimapOrientation string
	Palette            []color.RGBA
	SpriteTilePixels   int
	Dat                *datfile.Index
	Textures           *TerrainTextureOptions
	Shadows            *ShadowOptions
	Elevation          bool // Opt-in height field; shared-corner rule and lighting are provisional.
	RotationModel      string
}

type Report struct {
	Crop               *Crop                 `json:"crop,omitempty"`
	Scale              int                   `json:"scale,omitempty"`
	Schema             string                `json:"schema"`
	OK                 bool                  `json:"ok"`
	Stage              string                `json:"stage"`
	Projection         string                `json:"projection"`
	ImageWidth         int                   `json:"image_width"`
	ImageHeight        int                   `json:"image_height"`
	Drawn              int                   `json:"drawn"`
	CountsByClass      map[string]int        `json:"counts_by_class"`
	Skipped            []Skipped             `json:"skipped,omitempty"`
	TerrainColors      string                `json:"terrain_colors"`
	PixelSHA256        string                `json:"pixel_sha256"`
	Minimap            bool                  `json:"minimap,omitempty"`
	MinimapOrientation string                `json:"minimap_orientation,omitempty"`
	ObjectColors       string                `json:"object_colors,omitempty"`
	ObjectMarkerSizing string                `json:"object_marker_sizing,omitempty"`
	Palette            string                `json:"palette,omitempty"`
	ElevationMode      string                `json:"elevation_mode,omitempty"`
	ElevationPadding   int                   `json:"elevation_padding,omitempty"`
	ObjectHeightMode   string                `json:"object_height_mode,omitempty"`
	Fallbacks          []string              `json:"fallbacks,omitempty"`
	Textures           *TerrainTextureReport `json:"textures,omitempty"`
	Mood               *MoodReport           `json:"mood,omitempty"`
	Shadows            *ShadowReport         `json:"shadows,omitempty"`
	Particles          []ParticleReport      `json:"particles,omitempty"`
	RotationModel      string                `json:"rotation_model,omitempty"`
	ScenarioVersion    string                `json:"scenario_version,omitempty"`
}

type Skipped struct {
	UnitID int    `json:"unit_id"`
	Reason string `json:"reason"`
}

func ScenarioSnapshot(file *scenario.File, stage string, dat *datfile.Index) (Snapshot, error) {
	if file == nil || file.Map == nil {
		return Snapshot{}, errors.New("scenario has no decoded map")
	}
	terrain, err := file.Terrain(scenario.TerrainOptions{IncludeTiles: true})
	if err != nil {
		return Snapshot{}, err
	}
	colors := map[int]int{}
	for _, player := range file.Players {
		// Player settings are zero-based; unit owners reserve zero for Gaia.
		colors[player.Player+1] = player.Color
	}
	snap := scenarioSnapshot(stage, terrain.Width, terrain.Height, terrain.Tiles, file.Units, colors, dat)
	snap.ScenarioVersion = file.Version
	return snap, nil
}

// ArtworkSnapshot accepts the read-only map/unit projection, including older
// campaigns whose trigger and authoring formats are intentionally unsupported.
func ArtworkSnapshot(art *scenario.Artwork, stage string, dat *datfile.Index) (Snapshot, error) {
	if art == nil {
		return Snapshot{}, errors.New("scenario has no artwork projection")
	}
	snap := scenarioSnapshot(stage, art.Width, art.Height, art.Tiles, &art.Units, art.PlayerColors, dat)
	snap.ScenarioVersion = art.Version
	snap.SourceWarnings = append(snap.SourceWarnings, "scenario "+art.Version+": read-only terrain/units projection; triggers, XS and gameplay settings not decoded; not full-format verification")
	return snap, nil
}

func scenarioSnapshot(stage string, width, height int, tiles []scenario.TerrainTile, units *scenario.UnitInfo, colors map[int]int, dat *datfile.Index) Snapshot {
	snap := Snapshot{Schema: SnapshotSchema, Stage: stage, Width: width, Height: height, PlayerColors: colors}
	for _, tile := range tiles {
		previewTile := Tile{X: tile.X, Y: tile.Y, TerrainID: tile.TerrainID, Elevation: float64(tile.Elevation)}
		if tile.Layer >= 0 {
			layer := tile.Layer
			previewTile.Layer = &layer
		}
		snap.Terrain = append(snap.Terrain, previewTile)
	}
	if units != nil {
		for _, section := range units.Sections {
			for _, unit := range section.Units {
				o := Object{UnitID: unit.UnitConst, X: unit.X, Y: unit.Y, Z: unit.Z, Rotation: unit.Rotation, Variant: int(math.Round(unit.Rotation)), Player: section.Player, Class: "unit"}
				o.Status, o.InitialAnimationFrame = unit.Status, unit.InitialAnimationFrame
				if dat != nil {
					if meta, ok := unitMetadata(dat, 0, unit.UnitConst); ok {
						o.Name = meta.Name
						o.Class = objectClass(meta)
						o.FootprintX = float64(meta.ClearanceSizeX)
						o.FootprintY = float64(meta.ClearanceSizeY)
						o.Blocking = meta.ObstructionTypeID != 0 || meta.ObstructionClassID != 0
						o.MapDrawLevel = meta.MapDrawLevel
						o.MapColor = meta.MapColor
						o.HasMapFields = true
						for _, attribute := range meta.Attributes {
							if attribute.Active && attribute.AttributeType == 144 {
								o.MinimapMode = int(math.Round(float64(attribute.Amount)))
								o.HasMinimapMode = true
								break
							}
						}
					}
				}
				snap.Objects = append(snap.Objects, o)
			}
		}
	}
	return snap
}

// ReplaySnapshot builds a preview snapshot from the terrain grid embedded in a
// replay. Initial object state is included when Kit can decode its header spine.
func ReplaySnapshot(path string, dat *datfile.Index) (Snapshot, error) {
	report, err := replay.BuildSummaryWithOptions(path, replay.SummaryOptions{IncludeMapTiles: true})
	if err != nil {
		return Snapshot{}, err
	}
	var objects []replay.ObjectStateCard
	var objectErr error
	state, objectErr := replay.BuildObjectState(path, replay.ObjectStateOptions{})
	if objectErr == nil {
		objects = state.Objects
	}
	return replaySnapshotFromData(report.Map, report.Players, objects, objectErr, dat)
}

func replaySnapshotFromData(mapInfo *replay.MapInfo, players []replay.SummaryPlayer, objects []replay.ObjectStateCard, objectErr error, dat *datfile.Index) (Snapshot, error) {
	if mapInfo == nil || len(mapInfo.Tiles) != mapInfo.TileCount {
		return Snapshot{}, errors.New("replay does not expose a complete embedded terrain grid")
	}
	snap := Snapshot{
		Schema:       SnapshotSchema,
		Stage:        "replay_initial_map",
		Width:        mapInfo.Width,
		Height:       mapInfo.Height,
		PlayerColors: map[int]int{},
	}
	for _, tile := range mapInfo.Tiles {
		snap.Terrain = append(snap.Terrain, Tile{X: tile.X, Y: tile.Y, TerrainID: tile.Terrain, Elevation: float64(tile.Elevation)})
	}
	for _, player := range players {
		if player.PlayerID > 0 {
			snap.PlayerColors[player.PlayerID] = player.Color
		}
	}
	if objectErr != nil {
		snap.SourceWarnings = append(snap.SourceWarnings, "initial objects unavailable: "+objectErr.Error())
	} else {
		for _, object := range objects {
			unit := Object{UnitID: object.UnitID, X: object.X, Y: object.Y, Player: object.OwnerID, Class: object.Class}
			if dat != nil {
				if meta, ok := unitMetadata(dat, 0, object.UnitID); ok {
					unit.Name = meta.Name
					unit.Class = objectClass(meta)
					unit.FootprintX = float64(meta.ClearanceSizeX)
					unit.FootprintY = float64(meta.ClearanceSizeY)
					unit.Blocking = meta.ObstructionTypeID != 0 || meta.ObstructionClassID != 0
					unit.MapDrawLevel = meta.MapDrawLevel
					unit.MapColor = meta.MapColor
					unit.HasMapFields = true
					for _, attribute := range meta.Attributes {
						if attribute.Active && attribute.AttributeType == 144 {
							unit.MinimapMode = int(math.Round(float64(attribute.Amount)))
							unit.HasMinimapMode = true
							break
						}
					}
				}
			}
			snap.Objects = append(snap.Objects, unit)
		}
	}
	return snap, nil
}

// RenderMinimap draws a compact, whole-map 2:1 isometric overview using DAT
// terrain colors when the indexed game palette is available.
func RenderMinimap(s Snapshot, dat *datfile.Index, opts Options) (*image.RGBA, Report, error) {
	if err := validateSnapshot(s); err != nil {
		return nil, Report{}, err
	}
	orientation := opts.MinimapOrientation
	if orientation == "" {
		orientation = engineOrientation
	}
	switch orientation {
	case "", "normal", "transpose", "flipx", "flipy", "rotate90", "rotate180", "rotate270":
	default:
		return nil, Report{}, fmt.Errorf("unsupported minimap orientation %q", opts.MinimapOrientation)
	}
	pxPerTile := opts.MinimapScale
	if pxPerTile == 0 {
		pxPerTile = 1
	}
	if pxPerTile < 1 || pxPerTile > 4 {
		return nil, Report{}, errors.New("minimap scale must be between 1 and 4 pixels per tile")
	}
	w := (s.Width + s.Height) * pxPerTile
	h := ((s.Width+s.Height)*pxPerTile + 1) / 2
	transform := diamondTransform(s.Width, s.Height, float64(2*pxPerTile), 0, orientation)
	if int64(w)*int64(h) > maxImagePixels {
		return nil, Report{}, fmt.Errorf("minimap dimensions %dx%d exceed %d-pixel limit", w, h, maxImagePixels)
	}
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: color.RGBA{R: 18, G: 20, B: 22, A: 255}}, image.Point{}, draw.Src)
	terrain := make([]Tile, len(s.Terrain))
	for _, t := range s.Terrain {
		terrain[t.Y*s.Width+t.X] = t
	}
	if layerEnabled(opts.Layers, "terrain") {
		// Inverse-project pixel centers into tile space rather than allocating
		// one polygon per tile. This keeps the 480-map path O(output pixels).
		for py := 0; py < h; py++ {
			for px := 0; px < w; px++ {
				sx, sy := transform.inverse(float64(px)+0.5, float64(py)+0.5)
				x, y := int(math.Floor(sx)), int(math.Floor(sy))
				if x < 0 || y < 0 || x >= s.Width || y >= s.Height {
					continue
				}
				t := terrain[y*s.Width+x]
				// Swatch cell interiors follow the base terrain. Layer metadata
				// is retained, but engine boundary blending is not modeled here.
				c := minimapTerrainColor(t.TerrainID, dat, opts.Palette)
				img.SetRGBA(px, py, c)
			}
		}
	}
	report := Report{
		Schema: "aoe2kit.preview.report.v1", OK: true, Stage: s.Stage,
		Projection: "minimap-diamond-2-to-1", ImageWidth: w, ImageHeight: h,
		CountsByClass: map[string]int{}, Minimap: true, MinimapOrientation: orientation,
		TerrainColors:      "base terrain; semantic fallback",
		ObjectColors:       "scenario player colors; DAT map draw level and minimap mode filter objects",
		ObjectMarkerSizing: "approximate engine apparent size at a 480-pixel-wide minimap: units 2x2, buildings 4x3; scaled with output width",
		ElevationMode:      "disabled; engine behavior has not been verified",
	}
	for _, t := range s.Terrain {
		if t.Layer != nil && *t.Layer != t.TerrainID {
			report.Fallbacks = append(report.Fallbacks, "layered terrain uses base interior colors; engine boundary blending is not modeled")
			break
		}
	}
	if dat != nil {
		if len(opts.Palette) == 256 {
			report.TerrainColors = "base terrain; DAT minimap medium-color index resolved through 256-color palette"
			report.Palette = "indexed 256-color palette"
		} else {
			report.Fallbacks = append(report.Fallbacks, "DAT terrain color indices decoded but no 256-color palette supplied; semantic terrain colors used")
		}
	} else {
		report.Fallbacks = append(report.Fallbacks, "no DAT supplied; semantic terrain and class palettes used")
	}
	if layerEnabled(opts.Layers, "objects") {
		for _, o := range s.Objects {
			if o.Player == 0 {
				continue
			}
			if o.X < 0 || o.Y < 0 || o.X >= float64(s.Width) || o.Y >= float64(s.Height) {
				report.Skipped = append(report.Skipped, Skipped{UnitID: o.UnitID, Reason: "outside map"})
				continue
			}
			if o.HasMapFields && o.MapDrawLevel == 0 {
				continue
			}
			if o.HasMinimapMode && o.MinimapMode == 0 {
				continue
			}
			if o.HasMinimapMode && o.MinimapMode != 1 && o.MinimapMode != 2 {
				continue
			}
			colorIndex, playerColorAvailable := s.PlayerColors[o.Player]
			c := minimapObjectColor(o.Class, colorIndex, playerColorAvailable)
			if playerColorAvailable && dat != nil && colorIndex >= 0 && colorIndex < len(dat.PlayerColours) && len(opts.Palette) == 256 {
				index := int(dat.PlayerColours[colorIndex].MinimapColour1)
				if index >= 0 && index < len(opts.Palette) {
					c = opts.Palette[index]
				}
			}
			if o.MapColor != 0 && len(opts.Palette) == 256 {
				c = opts.Palette[o.MapColor]
			}
			sx, sy := transform.point(o.X, o.Y)
			px, py := int(sx), int(sy)
			markerW, markerH := 2, 2
			if o.Class == "building" {
				markerW, markerH = 4, 3
			}
			markerW = max(1, (markerW*w+479)/480)
			markerH = max(1, (markerH*w+479)/480)
			fillRect(img, px-markerW/2, py-markerH/2, markerW, markerH, c)
			report.Drawn++
			class := o.Class
			if class == "" {
				class = "unit"
			}
			report.CountsByClass[class]++
		}
	}
	report.Fallbacks = append(report.Fallbacks, s.SourceWarnings...)
	hash := sha256.Sum256(img.Pix)
	report.PixelSHA256 = hex.EncodeToString(hash[:])
	return img, report, nil
}

func minimapTerrainColor(id int, dat *datfile.Index, palette []color.RGBA) color.RGBA {
	if dat != nil && id >= 0 && id < len(dat.Terrains) && len(palette) == 256 {
		colorIndex := int(dat.Terrains[id].MinimapColorMedium)
		return palette[colorIndex]
	}
	if dat != nil && id >= 0 && id < len(dat.Terrains) {
		return terrainFallback(dat.Terrains[id].Name.Value+" "+dat.Terrains[id].Name2.Value, id)
	}
	return terrainFallback("", id)
}

const engineOrientation = "rotate270"

// tileTransform maps tile coordinates to screen coordinates. Its inverse is
// used for minimap sampling, so terrain and objects cannot disagree on axes.
type tileTransform struct {
	xx, xy, yx, yy, ox, oy float64
}

func (t tileTransform) point(x, y float64) (float64, float64) {
	return t.xx*x + t.xy*y + t.ox, t.yx*x + t.yy*y + t.oy
}

func (t tileTransform) inverse(x, y float64) (float64, float64) {
	x, y = x-t.ox, y-t.oy
	det := t.xx*t.yy - t.xy*t.yx
	return (t.yy*x - t.xy*y) / det, (t.xx*y - t.yx*x) / det
}

func diamondTransform(width, height int, tileWidth, padding float64, orientation string) tileTransform {
	if orientation == "" {
		orientation = engineOrientation
	}
	// Alternate axes are diagnostic inputs only. Every normal renderer uses
	// the same engine orientation, differing only in scale and canvas padding.
	orient := func(x, y float64) (float64, float64) {
		switch orientation {
		case "transpose":
			return y, x
		case "flipx":
			return float64(width) - x, y
		case "flipy":
			return x, float64(height) - y
		case "rotate90":
			return float64(height) - y, x
		case "rotate180":
			return float64(width) - x, float64(height) - y
		case "rotate270":
			return y, float64(width) - x
		default:
			return x, y
		}
	}
	mapHeight := height
	if orientation == "transpose" || orientation == "rotate90" || orientation == "rotate270" {
		mapHeight = width
	}
	project := func(x, y float64) (float64, float64) {
		u, v := orient(x, y)
		return (u - v + float64(mapHeight)) * tileWidth / 2, (u+v)*tileWidth/4 + padding
	}
	ox, oy := project(0, 0)
	xx, yx := project(1, 0)
	xy, yy := project(0, 1)
	return tileTransform{xx: xx - ox, xy: xy - ox, yx: yx - oy, yy: yy - oy, ox: ox, oy: oy}
}

func terrainFallback(name string, id int) color.RGBA {
	name = strings.ToLower(name)
	switch {
	case strings.Contains(name, "deep water"), strings.Contains(name, "water"), strings.Contains(name, "shallow"), strings.Contains(name, "shallows"):
		return color.RGBA{R: 18, G: 72, B: 145, A: 255}
	case strings.Contains(name, "beach"), strings.Contains(name, "shore"), strings.Contains(name, "sand"):
		return color.RGBA{R: 194, G: 170, B: 117, A: 255}
	case strings.Contains(name, "snow"), strings.Contains(name, "ice"):
		return color.RGBA{R: 207, G: 218, B: 220, A: 255}
	case strings.Contains(name, "forest"), strings.Contains(name, "woods"):
		return color.RGBA{R: 43, G: 91, B: 46, A: 255}
	case strings.Contains(name, "desert"), strings.Contains(name, "dirt"), strings.Contains(name, "road"):
		return color.RGBA{R: 157, G: 132, B: 91, A: 255}
	}
	switch id {
	case 1, 4, 15, 22, 23:
		return color.RGBA{R: 18, G: 72, B: 145, A: 255}
	case 2:
		return color.RGBA{R: 194, G: 170, B: 117, A: 255}
	case 0, 3, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 16, 17, 18, 19, 20, 21:
		return color.RGBA{R: 89, G: 132, B: 67, A: 255}
	default:
		return color.RGBA{R: 122, G: 130, B: 125, A: 255}
	}
}

func minimapObjectColor(class string, playerColor int, playerColorAvailable bool) color.RGBA {
	if playerColorAvailable && playerColor >= 0 && playerColor < len(minimapPlayerColors) {
		return minimapPlayerColors[playerColor]
	}
	return classColor(class)
}

var minimapPlayerColors = [...]color.RGBA{
	{R: 48, G: 100, B: 220, A: 255}, {R: 210, G: 58, B: 48, A: 255},
	{R: 64, G: 170, B: 75, A: 255}, {R: 220, G: 200, B: 48, A: 255},
	{R: 45, G: 190, B: 190, A: 255}, {R: 160, G: 80, B: 190, A: 255},
	{R: 135, G: 135, B: 135, A: 255}, {R: 220, G: 130, B: 45, A: 255},
}

func WriteSnapshot(path string, snap Snapshot) error {
	if err := validateSnapshot(snap); err != nil {
		return err
	}
	b, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	return os.WriteFile(path, b, 0o644)
}

func ReadSnapshot(path string) (Snapshot, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Snapshot{}, err
	}
	var snap Snapshot
	if err := json.Unmarshal(b, &snap); err != nil {
		return Snapshot{}, fmt.Errorf("decode preview snapshot: %w", err)
	}
	if err := validateSnapshot(snap); err != nil {
		return Snapshot{}, err
	}
	return snap, nil
}

func validateSnapshot(s Snapshot) error {
	if s.Schema != SnapshotSchema {
		return fmt.Errorf("unsupported preview snapshot schema %q", s.Schema)
	}
	if s.Width <= 0 || s.Height <= 0 || s.Width > 480 || s.Height > 480 {
		return fmt.Errorf("invalid preview snapshot dimensions %dx%d", s.Width, s.Height)
	}
	if len(s.Terrain) != s.Width*s.Height {
		return fmt.Errorf("snapshot has %d terrain cells, want %d", len(s.Terrain), s.Width*s.Height)
	}
	return nil
}

func RenderFlat(s Snapshot, opts Options) (*image.RGBA, Report, error) {
	if err := validateSnapshot(s); err != nil {
		return nil, Report{}, err
	}
	projection := opts.Projection
	if projection == "" {
		projection = "diamond"
	}
	if projection != "diamond" && projection != "topdown" {
		return nil, Report{}, fmt.Errorf("projection must be topdown or diamond")
	}
	scale := opts.Scale
	if scale == 0 {
		scale = 4
	}
	if scale < 1 || scale > 128 {
		return nil, Report{}, errors.New("scale must be between 1 and 128")
	}
	crop := Crop{X1: 0, Y1: 0, X2: s.Width, Y2: s.Height}
	if opts.Crop != nil {
		crop = *opts.Crop
	}
	if crop.X1 < 0 || crop.Y1 < 0 || crop.X2 > s.Width || crop.Y2 > s.Height || crop.X1 >= crop.X2 || crop.Y1 >= crop.Y2 {
		return nil, Report{}, errors.New("crop must be a non-empty in-bounds tile rectangle")
	}
	w, h := (crop.X2-crop.X1)*scale, (crop.Y2-crop.Y1)*scale
	if projection == "diamond" {
		w = (crop.X2-crop.X1+crop.Y2-crop.Y1)*scale/2 + 1
		h = (crop.X2-crop.X1+crop.Y2-crop.Y1)*scale/4 + scale*2 + 1
	}
	if int64(w)*int64(h) > maxImagePixels {
		return nil, Report{}, fmt.Errorf("preview dimensions %dx%d exceed %d-pixel limit; crop or lower --scale", w, h, maxImagePixels)
	}
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: color.RGBA{R: 24, G: 27, B: 30, A: 255}}, image.Point{}, draw.Src)
	terrain := make([]Tile, len(s.Terrain))
	for _, t := range s.Terrain {
		terrain[t.Y*s.Width+t.X] = t
	}
	groundOffset := func(o Object) int {
		return pad - objectGroundLift(s, ground, o, scale, projection, opts.StoredObjectHeights)
	}
	for y := crop.Y1; y < crop.Y2; y++ {
		for x := crop.X1; x < crop.X2; x++ {
			if !layerEnabled(opts.Layers, "terrain") {
				continue
			}
			tile := terrain[y*s.Width+x]
			c := terrainColor(tile.TerrainID)
			if opts.Dat != nil {
				c = minimapTerrainColor(tile.TerrainID, opts.Dat, opts.Palette)
			}
			if layerEnabled(opts.Layers, "elevation") {
				c = shadeElevation(c, tile.Elevation)
			}
			if opts.Elevation && projection != "diamond" {
				c = slopeShade(c, terrain, s.Width, s.Height, x, y)
			}
			if !opts.Elevation || projection != "diamond" {
				drawTile(img, x-crop.X1, y-crop.Y1, crop, scale, projection, c)
			}
		}
	}
	if opts.Elevation && projection == "diamond" && opts.Textures == nil && layerEnabled(opts.Layers, "terrain") {
		renderHeightfield(img, terrain, s, crop, scale, pad, func(x, y float64) color.RGBA {
			tile := terrain[int(y)*s.Width+int(x)]
			if opts.Dat != nil {
				return minimapTerrainColor(tile.TerrainID, opts.Dat, opts.Palette)
			}
			return terrainColor(tile.TerrainID)
		})
	}
	report := Report{Schema: "aoe2kit.preview.report.v1", OK: true, Stage: s.Stage, Projection: projection, ImageWidth: w, ImageHeight: h, CountsByClass: map[string]int{}, TerrainColors: "builtin deterministic terrain-id palette"}
	report.Fallbacks = append(report.Fallbacks, s.SourceWarnings...)
	report.Crop, report.Scale = &crop, scale
	if opts.Dat != nil {
		if len(opts.Palette) == 256 {
			report.TerrainColors = "base terrain; DAT minimap medium-color index resolved through 256-color palette"
		} else {
			report.TerrainColors = "semantic fallback from DAT terrain names"
		}
	}
	if opts.Textures != nil && layerEnabled(opts.Layers, "terrain") {
		textureReport, err := renderTexturedTerrain(img, terrain, s, crop, scale, projection, opts)
		if err != nil {
			return nil, Report{}, err
		}
		report.Textures = &textureReport
		report.TerrainColors = "installed DDS textures with named layer masks; inspect textures report for assumptions and fallbacks"
		report.ElevationMode = "flat textured surface; geometric elevation and lighting not modeled"
	}
	if opts.Elevation {
		report.ElevationMode = "experimental height field: scale/4 lift per stored level; max-of-four shared corners; isolated NW/NE/SE raised neighbors use level-diagonal folds with connected-ridge guard; shared-vertex lighting from central terrain gradients; inspect object_height_mode for stored Z or provisional inferred attachment; flat-position depth order; no terrain occlusion"
		if projection != "diamond" {
			report.ElevationMode = "topdown slope shading only; no geometric displacement"
		}
	}
	for _, o := range depthSortedObjects(s.Objects, crop, scale, projection) {
		if o.X < float64(crop.X1) || o.Y < float64(crop.Y1) || o.X >= float64(crop.X2) || o.Y >= float64(crop.Y2) {
			report.Skipped = append(report.Skipped, Skipped{UnitID: o.UnitID, Reason: "outside crop"})
			continue
		}
		if !layerEnabled(opts.Layers, "objects") {
			report.Skipped = append(report.Skipped, Skipped{UnitID: o.UnitID, Reason: "objects layer disabled"})
			continue
		}
		paintObject(groundCanvas(img, groundOffset(o)), o, crop, scale, projection)
		report.Drawn++
		class := o.Class
		if class == "" {
			class = "unit"
		}
		report.CountsByClass[class]++
	}
	if layerEnabled(opts.Layers, "starts") {
		paintStarts(img, s.Objects, crop, scale, projection, groundOffset)
	}
	hash := sha256.Sum256(img.Pix)
	report.PixelSHA256 = hex.EncodeToString(hash[:])
	return img, report, nil
}

func WritePNG(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

func layerEnabled(layers map[string]bool, name string) bool {
	if len(layers) == 0 {
		return true
	}
	return layers[name]
}

func terrainColor(id int) color.RGBA {
	// Stable categorical colors aid inspection without claiming the game's palette.
	h := uint8((id*47 + 23) % 180)
	return color.RGBA{R: 55 + h/3, G: 65 + (h*3)%115, B: 48 + (h*5)%135, A: 255}
}

func shadeElevation(c color.RGBA, elevation float64) color.RGBA {
	shade := 0.72 + math.Min(0.55, math.Max(0, elevation*0.08))
	return color.RGBA{R: uint8(math.Min(255, float64(c.R)*shade)), G: uint8(math.Min(255, float64(c.G)*shade)), B: uint8(math.Min(255, float64(c.B)*shade)), A: 255}
}

func drawTile(img *image.RGBA, x, y int, crop Crop, scale int, projection string, c color.RGBA) {
	if projection == "topdown" {
		fillRect(img, x*scale, y*scale, scale, scale, c)
		return
	}
	points := make([]image.Point, 4)
	for i, p := range [][2]float64{{float64(x), float64(y)}, {float64(x + 1), float64(y)}, {float64(x + 1), float64(y + 1)}, {float64(x), float64(y + 1)}} {
		px, py := mapPoint(p[0], p[1], crop, scale, projection)
		points[i] = image.Pt(px, py)
	}
	fillPolygon(img, points, c)
}

func paintObject(img *image.RGBA, o Object, crop Crop, scale int, projection string) {
	c := classColor(o.Class)
	x, y := mapPoint(o.X-float64(crop.X1), o.Y-float64(crop.Y1), crop, scale, projection)
	if o.Blocking && o.FootprintX > 0 && o.FootprintY > 0 {
		hx, hy := o.FootprintX/2, o.FootprintY/2
		points := make([]image.Point, 4)
		for i, p := range [][2]float64{{o.X - hx, o.Y - hy}, {o.X + hx, o.Y - hy}, {o.X + hx, o.Y + hy}, {o.X - hx, o.Y + hy}} {
			px, py := mapPoint(p[0]-float64(crop.X1), p[1]-float64(crop.Y1), crop, scale, projection)
			points[i] = image.Pt(px, py)
		}
		fillPolygon(img, points, color.RGBA{R: c.R, G: c.G, B: c.B, A: 92})
	}
	r := max(1, scale/3)
	fillRect(img, x-r, y-r, 2*r+1, 2*r+1, c)
}

func fillPolygon(img *image.RGBA, points []image.Point, c color.RGBA) {
	if len(points) < 3 {
		return
	}
	minY, maxY := img.Bounds().Max.Y, img.Bounds().Min.Y
	for _, p := range points {
		if p.Y < minY {
			minY = p.Y
		}
		if p.Y > maxY {
			maxY = p.Y
		}
	}
	if minY < img.Bounds().Min.Y {
		minY = img.Bounds().Min.Y
	}
	if maxY > img.Bounds().Max.Y {
		maxY = img.Bounds().Max.Y
	}
	for y := minY; y < maxY; y++ {
		yf := float64(y) + 0.5
		intersections := make([]float64, 0, len(points))
		for i, a := range points {
			b := points[(i+1)%len(points)]
			if float64(a.Y) == float64(b.Y) || yf < math.Min(float64(a.Y), float64(b.Y)) || yf >= math.Max(float64(a.Y), float64(b.Y)) {
				continue
			}
			x := float64(a.X) + (yf-float64(a.Y))*float64(b.X-a.X)/float64(b.Y-a.Y)
			intersections = append(intersections, x)
		}
		sort.Float64s(intersections)
		for i := 0; i+1 < len(intersections); i += 2 {
			x1, x2 := int(math.Ceil(intersections[i]-0.5)), int(math.Ceil(intersections[i+1]-0.5))
			fillRect(img, x1, y, x2-x1, 1, c)
		}
	}
}

func paintStarts(img *image.RGBA, objects []Object, crop Crop, scale int, projection string, offsetY func(Object) int) {
	seen := map[int]bool{}
	for _, o := range objects {
		if o.Player <= 0 || seen[o.Player] || o.X < float64(crop.X1) || o.Y < float64(crop.Y1) || o.X >= float64(crop.X2) || o.Y >= float64(crop.Y2) {
			continue
		}
		seen[o.Player] = true
		x, y := mapPoint(o.X-float64(crop.X1), o.Y-float64(crop.Y1), crop, scale, projection)
		if offsetY != nil {
			y += offsetY(o)
		}
		fillRect(img, x-2, y-2, 5, 5, color.RGBA{255, 255, 255, 255})
	}
}

func mapPoint(x, y float64, crop Crop, scale int, projection string) (int, int) {
	if projection == "topdown" {
		return int(x * float64(scale)), int(y * float64(scale))
	}
	px, py := diamondTransform(crop.X2-crop.X1, crop.Y2-crop.Y1, float64(scale), float64(scale), "").point(x, y)
	return int(px), int(py)
}

func depthSortedObjects(objects []Object, crop Crop, scale int, projection string) []Object {
	ordered := append([]Object(nil), objects...)
	t := diamondTransform(crop.X2-crop.X1, crop.Y2-crop.Y1, float64(scale), 0, "")
	depth := func(o Object) float64 {
		if projection == "topdown" {
			return o.Y
		}
		_, y := t.point(o.X-float64(crop.X1), o.Y-float64(crop.Y1))
		return y
	}
	sort.SliceStable(ordered, func(i, j int) bool {
		a, b := depth(ordered[i]), depth(ordered[j])
		if a != b {
			return a < b
		}
		return ordered[i].Player < ordered[j].Player
	})
	return ordered
}

func classColor(class string) color.RGBA {
	switch class {
	case "tree":
		return color.RGBA{R: 24, G: 110, B: 46, A: 255}
	case "cliff":
		return color.RGBA{R: 160, G: 135, B: 117, A: 255}
	case "building":
		return color.RGBA{R: 190, G: 67, B: 49, A: 255}
	case "resource":
		return color.RGBA{R: 232, G: 205, B: 58, A: 255}
	case "decor":
		return color.RGBA{R: 90, G: 181, B: 164, A: 255}
	default:
		return color.RGBA{R: 225, G: 225, B: 230, A: 255}
	}
}

func fillRect(img *image.RGBA, x, y, w, h int, c color.RGBA) {
	if w <= 0 || h <= 0 {
		return
	}
	r := image.Rect(x, y, x+w, y+h).Intersect(img.Bounds())
	draw.Draw(img, r, &image.Uniform{C: c}, image.Point{}, draw.Over)
}

func unitMetadata(idx *datfile.Index, civ, id int) (datfile.UnitSummary, bool) {
	if civ < 0 || civ >= len(idx.Civs) {
		return datfile.UnitSummary{}, false
	}
	for _, u := range idx.Civs[civ].Units {
		if u.Present && (u.Index == id || int(u.ID) == id) {
			return u, true
		}
	}
	return datfile.UnitSummary{}, false
}

func objectClass(u datfile.UnitSummary) string {
	if u.Type == 80 {
		return "building"
	}
	if u.Class == 15 {
		return "tree"
	}
	n := strings.ToLower(u.ClassName + " " + u.Name)
	if strings.Contains(n, "tree") || strings.Contains(n, "palm") || strings.Contains(n, "oak") {
		return "tree"
	}
	if strings.Contains(n, "cliff") {
		return "cliff"
	}
	if strings.Contains(n, "building") || strings.Contains(n, "town center") || strings.Contains(n, "barracks") {
		return "building"
	}
	if strings.Contains(n, "gold") || strings.Contains(n, "stone") || strings.Contains(n, "resource") || strings.Contains(n, "fish") || strings.Contains(n, "berry") {
		return "resource"
	}
	if strings.Contains(n, "decoration") || strings.Contains(n, "flower") || strings.Contains(n, "bush") {
		return "decor"
	}
	return "unit"
}

func SortedObjects(in []Object) []Object {
	out := append([]Object(nil), in...)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Y != out[j].Y {
			return out[i].Y < out[j].Y
		}
		if out[i].X != out[j].X {
			return out[i].X < out[j].X
		}
		if out[i].Player != out[j].Player {
			return out[i].Player < out[j].Player
		}
		return out[i].UnitID < out[j].UnitID
	})
	return out
}
