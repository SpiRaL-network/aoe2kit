# Visual Previews

Kit generates local PNG previews from scenario data and, when available, the
user's own game installation. Preview images are outputs only: they are not
checked into the repository or included in Kit packs or releases.

Known visual limits: water edges differ from the engine, the mood grade is
approximate, and beach and road brightness is off. Geometric terrain elevation
is opt-in and provisional. The reports distinguish approximations from draws.

## Read-Only Campaign Artwork

Scenario previews and `kit preview snapshot` accept measured layout versions
1.36, 1.37 and 1.40 through 1.59. Versions 1.38 and 1.39 have no corpus evidence
and remain unsupported. This is a separate read-only projection, not an
extension of full scenario parsing, authoring, writing or verification support.

`scenario.OpenArtwork` / `ParseArtwork` read the bounded sequential prefix
through the unit records: map dimensions, terrain IDs, elevations and layers,
unit owners, IDs, XYZ, rotation, status and initial animation frame. The reader
does not interpret triggers, XS or gameplay settings. Reports and snapshots
carry that limitation explicitly. `preview.ArtworkSnapshot` converts the
projection; `preview.ScenarioSnapshot` remains available for full scenario files.
Status and frame are retained as data, not a promise of animation/state rendering.

The parser has synthetic tests for every supported artwork version. Private
campaign validation is opt-in with `AOE2KIT_SCENARIO_ARTWORK_CORPUS`; no game
scenarios or art are bundled. The measured campaign corpus yielded map and units
for 235/235 files; its 19 version-1.58/1.59 files also matched the full parser's
terrain and unit fields. This is corpus evidence, not an engine-load check or a
claim that every possible file in those versions is supported.

## Studying Campaign Compositions

```sh
kit scen hotspots campaign.aoe2scenario --game-dir game --cell 8 --top 10
kit scen hotspots campaign.aoe2scenario --dat game.dat --rank count --gaia=false
kit scen census campaign.aoe2scenario --dat game.dat --crop 32,40,48,56 --gaia
kit preview assets campaign.aoe2scenario --dat game.dat --game-dir mirror --crop 32,40,48,56
kit preview assets campaign.aoe2scenario --dat game.dat --game-dir mirror --missing-only --paths
```

These read-only commands accept the artwork versions above. They require a DAT
for names/classification, supplied explicitly or resolved from `--game-dir`,
`AOE2KIT_GAME_DIR`, or local installation discovery. No remote-access setup is
part of Kit. A real local or mounted installation already works directly with
`kit preview --game-dir`; copying into a mirror is unnecessary in that case.

Hotspots default to Gaia, eight-tile cells and descending unit-kind variety,
then count, then row/column order. `--rank count` reverses the first two keys.
`--include-forest-fish` disables the default exclusion of DAT classes 15 (trees)
and 5, 31, 33, 63 (fish). Stumps, ruins, cliffs and unknown units remain visible.
Results include clipped crop rectangles and a census with internal DAT names,
owner counts and stored rotations. Crops are half-open and use unit anchors,
not sprite extents. Census defaults to all owners; `--gaia` restricts it.
Rotations are reported as raw values, not guaranteed valid visual variants.
The census describes starting placements, not trigger-spawned units or a full
reconstruction recipe.

`preview assets` deduplicates standing-graphic SLD paths and particle JSON/atlas
dependencies. Its JSON paths are install-relative, with `present`, `missing`,
or `unchecked` status. An external fetch tool can consume the manifest in one
batch; Kit neither transfers nor redistributes game assets. `--paths` prints
one relative path per line. `--missing-only` requires an install/mirror to check.
Missing SLDs use the conventional `resources/_common/drs/graphics` location;
existing files use their actual location. A missing particle definition makes
the dependency list explicitly incomplete: fetch the JSON and rerun to discover
its atlas. Path-only output exits nonzero in that case rather than pretending
the list is complete. Presence is not validation of the asset contents.
Terrain textures, animation states, annexes, graphic deltas and packed DRS
extraction are outside this sprite manifest's scope.

## Reusable Composition Library

Go scene generators can import `aoe2kit/pkg/compose` instead of copying RNG,
noise, grid, reservation and recipe-writing helpers. Parameters belong in the
generator's configuration; a critic round changes those inputs, not generated
Go source. There is no private-store, host or remote-machine dependency.

```go
scene, err := compose.New(80, 80, 0, 7373)
if err != nil { return err }
road := compose.WoodenRoad{Scene: scene, UnitID: 2748}
turtle := compose.Turtle{Road: road, Next: compose.Point{X: 20.5, Y: 30.5}, DX: 1}
turtle.Forward(6)
turtle.Turn(0, 1)
turtle.Forward(4)
scene.SetLayer(25, 25, 32)
return scene.WriteBundle("revision-2")
```

`New` sets dimensions, base terrain and deterministic seed. `SetTerrain`,
`SetLayer`, `SetElevation`, `Put` and `Gaia` edit the scene. `Claim`/`Free` use
circular art-spacing reservations; `FreeRadius` also accounts for the candidate
radius. They do **not** establish game collision, walkability or land/water
suitability. `RNG`, `Noise` and four-octave `FBM` have no global random state.
`Recipe` returns Kit's existing typed recipe, while `WriteBundle` writes
`recipe.json` plus relative `grid.json` into a **new** directory. Invalid values
or out-of-bounds placements set a sticky error and prevent export. The target
scenario must have matching dimensions; existing units are not removed.

Included construction functions and their evidence boundaries:

- `CliffFacets` / `CliffWall`: preview-observed face versus lip orientation;
  short 1-2-slab runs at three-tile pitch, separated by matching mortar art.
  Axis X faces +Y, axis Y faces -X. High ground on the camera side uses lips.
  Position callbacks permit bends but cannot make rigid slabs continuously curve.
- `LatticeStair`: the York campaign's v15/v3 three-tile corner lattice, with
  optional v4 flats, v12/v14 caps and a two-step gully. `GapStart: -1` disables
  the gully. This is placement/preview evidence, not a traversable staircase.
- `Terrace`: preview-derived v4/v5 runs of 2-3 slabs at whole-tile levels;
  discrete cuts use rotating caller-supplied coverup groups downhill of the face.
- `WoodenRoad`: preview-measured v0 squares, axis strips, v12-15 curves/rings,
  v16-19 diagonal corners with v5/v7 extensions, v8/v9 triangle-squared bands,
  and cardinal turtle turns. Unit 2748 is the measured material; other IDs must
  share its visual layout. Fractional mouth offsets are measurements, not a
  bridge/pathfinding rule.
- `River`: polyline widths, 55% core, shallow water, bank and 60%-deep plunge
  pools. Tile-centre rasterization replaces overlapping half-tile stamps;
  strongest coverage wins at crossings. Previewed narrow streams spread at
  blended borders, so source width is not rendered or traversable width.
- `MixedTrack`: FBM dirt/mud with optional missing plank squares and dry
  half-boards. Water exclusions, probabilities and material IDs are explicit.
- `WaterfallPair`: paired background/overlay at an explicit lip and angle.
  The particle anchor/angle convention remains **uncalibrated in the engine**.

All are reusable **preview-derived art recipes**, not engine-proven behavior.
Materials, exact variant lists, coverup groups and probabilities remain caller
choices. Validate against the chosen DAT/installed art and use the normal Kit
recipe checks; the builder does not certify arbitrary unit/variant combinations.

## Textured Terrain (Experimental)

```sh
kit preview scenario.aoe2scenario --textures --sprites --scale 32 --crop 40,20,80,60 --out textured.png
kit preview scenario.aoe2scenario --textures --terrain-repeat 9 --no-terrain-edges --out layers.png
```

`--textures` uses the shared game-install resolver (`--game-dir`, then
`AOE2KIT_GAME_DIR`, then discovery) and requires a DAT. Art is read at runtime
from `resources/_common/terrain/textures/2x`, `masks`, and `blends` in that
installation. No game images are included in Kit or its releases. Flat
previews remain available without an installation; minimaps do not use textures.

Texture and mask repeats default to ten world tiles, with separate overrides
`--texture-repeat` and `--terrain-repeat`. The grass texture default is fitted
to registered flat-ground samples: ten tiles gave correlations 0.476, 0.520
and 0.560 across three views, versus approximately zero at the former nine-tile
default. Only the first view selected the repeat/orientation/phase; the other
two were checks. This calibrates grass texture scale and origin, not exact
colour, other terrain materials, lighting, or filtering.

The pure-Go decoder supports 2D DDS BC1/DXT1, BC2/DXT3 and BC3/DXT5, rejects
unsupported formats, and bounds dimensions at 4096 per axis. Rendering samples
the first mip at 512-2048 pixels per axis according to output scale, with a 128 MiB asset-cache
budget. Missing or unsupported textures use the flat fallback; missing layer
masks retain the base. Each case is reported under `textures.assets[].issues`.

Evidence and assumptions are separate in the JSON report:

- Named layer masks use white for the top terrain, with a crisp cutoff. Road,
  snow and azure-water screenshots support this polarity and appearance.
- The exact cutoff (128), world-axis UV origin, static texture repeat, and
  nine-tile mask repeat are candidates, not engine calibration. The observed
  mask-repeat range is roughly eight to ten tiles. `--terrain-repeat` controls
  mask repeat, while `--texture-repeat` independently controls texture repeat,
  both in world tiles (default 9). Nine texture tiles give roughly four to nine
  paving-stone widths per tile in the road texture, consistent with the gridded
  screenshot's approximate 0.1-0.25-tile stones, not a pixel-exact calibration.
  Cropping never resets either pattern. Use `--scale 128` on a small crop to
  inspect stone size; low-resolution whole-map previews obscure this detail.
- Empty DAT mask names use **unverified inverted default_strong**, chosen only
  as a candidate for the observed black-on-azure coverage.
- Edge blending reads installed `resources/_common/dat/blendomatic_x1.dat`:
  97x49 alpha diamonds selected by the eight neighbouring tiles, not fitted
  radial footprints. Higher DAT priority draws last; equal priorities stay hard.
  Classic pair-to-mode and coordinate-variant selection are **DE candidates**.
  The measured file's weights retain the base (0 replaces it, 128 retains it).
  Road single/ring crops have irregular edges closer to the editor than the
  previous radial model; diagonal roads join but the pinch remains too weak.
  Water remains unresolved:
  classic grass/water mode 1 leaves a ring opening, contrary to the human
  observation that it closes. No synthetic fill conceals that mismatch.
  Reports identify the installed mask hash, modes, evidence and assumptions.
  Missing/unsupported masks retain hard borders with an explicit issue.
  DE's extra blend texture/shader, other material pairs, variant phase and
  precise filtering are not calibrated. `--no-terrain-edges` isolates layers.
- Water animation, shaders, lighting, and geometric elevation are not rendered.

The DAT blend-word layout is corroborated by
[genieutils Terrain serialization](https://github.com/Tapsa/genieutils/blob/master/src/dat/Terrain.cpp);
that layout does not establish the engine's blending policy.
Classic mask layout and directional/pair mapping facts are documented by
[openage](https://github.com/SFTtech/openage/blob/master/doc/media/blendomatic.md).
Kit uses an independent Go decoder and selector, not upstream implementation code.

## Particle Sprites (Experimental)

`kit preview scenario.aoe2scenario --sprites --out preview.png` also reads
static loop particles from the selected installation. The standing graphic's
DAT `ParticleEffectName` names the definition under
`resources/_common/particles`; its `AtlasFile` names a PNG or DDS atlas with an
adjacent TexturePacker JSON array. This is an explicit DAT reference, not a
guessed name match or a hardcoded list of waterfall unit IDs.

The preview uses the first frame of each angle's contiguous image range,
with counts from the particle definition rather than the legacy DAT graphic.
It restores clockwise-packed rectangles, accounts for trimmed source-canvas
offsets, places the atlas pivot at the unit anchor, and applies the definition's
`Scale` using the sprite tile convention. TexturePacker distinguishes packed
and unrotated frame sizes in its
[exporter documentation](https://www.codeandweb.com/texturepacker/documentation/custom-exporter).
The installed atlas format uses unrotated frame dimensions; the sampled
footprint is swapped for rotated entries. Coordinates are used as stored,
without another correction for atlas `meta.scale`.

`Layer: "Terrain"` particles draw below sprites, before the optional terrain
shadow pass. Other supported loop particles join the sprite painter order.
The per-unit `particles` report separates DAT mapping evidence from the
**uncalibrated** direct rotation-index, pivot, and scale convention. A local
probe renders all three overlay angles, three background angles, and two small
waterfall angles; that does not yet prove the engine's placement convention.

Missing assets, invalid ranges, and unsupported particle types/layers are
reported as skipped, never replaced with guessed artwork. A graphic with both
an SLD and a particle retains its SLD; hybrid composition is explicitly skipped.
Animation, emitter simulation, fog visibility, and particle shadow casting are
not reproduced. This does not change flat previews or minimaps. Images remain
local runtime inputs/outputs, never repository or release content.

## Sprite Shadows (Opt-In)

```text
kit preview scenario.aoe2scenario --textures --sprites --shadows --mood scenario --scale 96 --crop 40,20,60,40 --out shadows.png
```

`--shadows` decodes installed SLD BC4 shadow coverage on the same canvas and
hotspot as the main frame. It accumulates coverage, darkens terrain, then draws
sprites in painter order. It requires `--sprites`; it never affects minimaps.
`--shadow-overlap max|sum` selects a diagnostic overlap policy (default `max`).
No shadow silhouette or game image is bundled with Kit.

Terrain darkening uses `2 * min(coverage, 0.375)`, the mood's shadow strength
(saturated after multiplication), and its RGB shadow colour. With `--mood none`,
colour correction is disabled and the factor is simply one minus coverage
after that cap and doubling. Textured previews then apply the existing mood
grade; that grade remains approximate.

The pass remains experimental: overlap blend state and sprite shadow-receiver
metadata are not engine-verified. Sprites do not receive this terrain pass.
Casters whose anchors lie outside an explicit crop are omitted, as in the main
sprite pass. Missing layers and failed decodes are reported separately in
`shadows`; a missing layer is not replaced by a guessed shape.

## Flat preview

```text
kit preview scenario.aoe2scenario --out preview.png
kit preview scenario.aoe2scenario --minimap --out minimap.png
kit preview match.aoe2record --minimap --out replay-minimap.png
kit preview scenario.aoe2scenario --out crop.png --crop 40,20,100,80 --projection diamond
kit preview snapshot.json --out stage.png --projection topdown --scale 6
kit preview snapshot scenario.aoe2scenario --out stage.json --stage terrain
```

Flat preview works without a game install. It draws terrain, object-class
markers, known blocking footprints, elevation shading, and player starts.
Terrain colors use a deterministic categorical palette. A preview is a
composition aid, not an engine rendering or placement guarantee.

`--minimap` renders the entire map into a compact, 2:1 isometric overview
without sprites. It uses a low-memory per-pixel inverse projection and does not
apply elevation shading by default. With a DAT and the game's
`widgetui/palette.json`, it resolves terrain minimap color indices and uses
unit map-draw/minimap fields to filter markers. For layered scenario tiles, the
preview uses the base terrain's interior color. Layer metadata is retained,
but engine boundary blending is not modeled; layered maps report this limitation.
An installed palette is discovered with `--game-dir`;
otherwise pass `--palette FILE` (and `--dat FILE`).
Flat diamond previews, sprite previews, and minimaps share one tile-to-screen
transform: increasing scenario x goes up-right and increasing y goes
down-right. Sprite painter order follows that screen depth. Cropping changes
the origin, not the orientation. The explicit topdown diagnostic view remains
an unrotated tile grid, not an imitation of the game camera.
The default `rotate270` orientation matches the engine's geographic minimap
reference and the terrain-pair swatch: increasing scenario x goes up-right,
and increasing y goes down-right. No per-map rotation is needed.
`--minimap-orientation` is a diagnostic override. Object visibility and
elevation behavior can still differ from the engine; use the report's fallback
and status fields to see what was actually available.
Player markers use DAT minimap player colors when available, with scenario
slot numbers translated to unit-owner numbers. Marker sizes approximate the
engine screenshot at a 480-pixel-wide minimap (2x2 for units, 4x3 for buildings)
and scale with output width. They are not an engine-exact shape model.

Replay input uses the embedded map terrain grid and includes initial object
state when the replay header's object sections are decoded. If Kit cannot
decode that section, the render report records the omission; it does not infer
player markers from later replay actions.

A snapshot can be rendered directly with
`kit preview snapshot.json --out preview.png`.

## Colour Mood Calibration

Textured previews apply the installed mood definition (`--mood NAME`, or the
scenario's mood by default). `--mood none` exports the ungraded image with
terrain lighting still included. The default grade is an approximation, not
an engine shader implementation. Minimap colours are not mood-graded.

```text
kit preview calibrate-mood --manifest pairs.json --game-dir GAME --fit --holdout Steppes --out fit.json
kit preview calibrate-mood --samples fit.json --fit --holdout Steppes --out refit.json
kit preview scenario.aoe2scenario --textures --mood Autumn --mood-model fit.json --out candidate.png
```

`--mood-model` explicitly opts into a fitted model; fitting never changes the
default. Without `--fit`, calibration reports the current default's residuals.
`--samples` reuses a previous report without needing images or an install.
All images and game definitions remain local; none are shipped with Kit.

A minimal manifest (paths relative to the manifest):

```json
{"cases":[
  {"name":"scene/autumn","preview":"raw.png","capture":"autumn.png","mood":"Autumn","offset":[-240,-268],"stride":12},
  {"name":"scene/steppes","preview":"default-capture.png","capture":"steppes.png","input_mood":"Default","mood":"Steppes"}
]}
```

Omit `offset` for translation-only luminance registration, within 512 pixels
per axis. Capture coordinates equal preview coordinates plus `offset`; images
must have the same pixel scale. Inspect the reported correlation and alignment
before trusting a fit. Uniform preview corners, transparent pixels and near-black
backgrounds are excluded. Already-graded input pixels near clipping are excluded
because their original values cannot be recovered.

For swatches, replace `stride` with `regions`, each containing `name`, `points`
(an array of preview `[x,y]` pixel coordinates), and optional `exclude: true`.
Each region contributes a paired RGB median. Excluded regions still get
residuals but never influence fitting. Each region can also name a raw DDS
`texture`; the report includes its full-mip mean, median and file hash.
Set the case's `texture_input: true` to fit these texture medians directly
against capture medians, bypassing the preview pixels. An optional
`texture_gain: [1,1,1]` applies explicit lighting gains before grading.
Texture-wide statistics are not spatially registered texture samples and cannot
by themselves distinguish incorrect UVs, blending, lighting, or sprite shadows.

The fitter compares lerp, multiplicative and mixed tint; shared coefficients
also cover contrast pivot, colour-dependent bloom, highlight suppression and
a distance-fog-colour veil. Deterministic bounded coordinate descent uses two
initializations and at most 1,024 evenly spaced samples per case, with equal
case weights. Per-case and per-sample residuals use all supplied samples, in
display-RGB RMS on a 0-255 scale. A required held-out mood is excluded from both
coefficient fitting and model selection. This is a validation split, not proof
of generalization to every scene or mood.

The 2026-10-16 meadow/swatch fit lowered Autumn error but worsened Savannah
and held-out Steppes. It was **not promoted to the default**. Spatial bloom,
shader terrain correction, fog of war, vignette, sharpening, cloud shadows and
sprite shadows remain unmodelled; a global grade cannot explain all of them.

### Shader Equation Audit

`kit preview calibrate-mood --manifest pairs.json --game-dir GAME
--shader-options options.json --holdout Steppes --out audit.json` evaluates
recovered stage equations without fitting their coefficients. It does **not**
change the default renderer. It is incompatible with `--fit`. The same option
works with `--samples`, but reports made before shader fields were decoded
must first be regenerated from the manifest and installation.

An explicit diagnostic configuration, **not measured engine defaults**:

```json
{
  "color_correction": true,
  "bloom": true,
  "bloom_option_scale": 1,
  "horizontal_tap_sum": 1,
  "vertical_tap_sum": 1,
  "gamma_exponent": 1,
  "input_lighting": [1, 1, 1],
  "modulation_alpha": 0,
  "distance_fog": false,
  "distance_fog_option_scale": 1,
  "screen_y": 0.5
}
```

The decoded operations differ from the empirical grade:

- Terrain tint and bloom terrain correction precede lighting and shadows.
  Lighting is `diffuse * lightmap + ambient + base`, with engine-supplied inputs.
- Modulation alpha controls capped shadow coverage. Its RGB is not a terrain
  colour lookup in the combine pass. The terrain attribute texture carries
  visibility in R/G and lightmap values in B/A, not terrain IDs.
- After sprite composition, distance fog is an additive, screen-height-dependent
  term. Vignette uses rectangular edge ramps before colour correction.
- Post correction is saturation about Rec.709 luma, contrast about exactly
  0.5, brightness, then ordinary tint interpolation. Each step clips to 0..1.
  The engine option gamma exponent follows; the mood JSON's gamma is unused
  by this pass.
- Bloom extraction averages four fetches, multiplies by scene colour intensity
  and the effective option intensity, thresholds **each channel**, then applies
  bloom saturation. Each blur pass sums supplied weighted fetches; the vertical
  pass applies bloom tint. The copy pass adds bloom without clipping in its
  arithmetic. Render-target conversion is a separate unknown.

The Go stage helpers implement these equations with caller-supplied inputs.
The calibration audit evaluates only a **uniform, fully visible terrain patch**:
blur reduces to the supplied tap sums; it assumes combine output feeds bloom
extraction followed by copy-add. That binding order, the option-to-JSON intensity
mapping, screen position and intermediate formats are explicitly unverified.
The report lists these assumptions and inverse residuals for already-graded
inputs. It is not an exact scene renderer or a spatial-bloom simulation.
Animated fog and fog-border noise are outside this audit. Stage math was checked
locally against disassembly; no shader text, binaries, palettes or images ship
with Kit.

The fixed-equation audit improved Autumn but worsened other captures, including
Arctic, Winter and Misty. It therefore was **not promoted**. Recovering the
remaining runtime buffers/options is more informative than fitting more shape
constants; the equations alone do not determine the captured engine state.

## Installed Sprites And Connector Grammar

Local agents can use game assets from an installed copy of AoE2DE. Asset
resolution order is `--game-dir`, `AOE2KIT_GAME_DIR`, then platform Steam
discovery. `kit doctor` reports the resolved install and whether loose SLD
sprites are readable. No game assets are copied into the repository, packed
bundle, or release.

```text
kit preview scenario.aoe2scenario --out crop.png --sprites --crop 40,20,80,60
kit felt grammar 1062 --out fence.png --json fence.json --game-dir game
```

The grammar command renders a selected connector's DAT angle slots on labelled
grass/water contexts and emits editable grammar JSON. The embedded connector
table contains only unit IDs, roles, axes, explicit rotation mappings and
evidence labels, never art bytes. `preview_reviewed` does not mean
engine-verified or passability-tested. Supply `--compendium` to use a different
connector grammar; no private metadata location is discovered implicitly.
General felt-sheet queries, paging and tag/state filtering are not included.

Sprite previews currently read loose SLD files. Packed DRS archive decoding is
not implemented, so a typical installation without loose SLD files can still
produce flat previews but cannot yet render sprites.
Player colors, SLD frame identity, and hotspot anchoring are applied by the
renderer, but engine behavior remains the final visual reference. Sprite
previews require diamond projection; `--scale` is the tile width in output
pixels. `--sprite-tile-width` sets the native art's reference tile width
(default 96, a preview convention not yet engine-calibrated). Sprites and
hotspots are scaled together, including downscaling. Standing graphics are
used; building annexes and running animations are not composed. Shadows are
optional with `--shadows`. Compressed
sprite caching is capped at 128 MiB and decoded frame caching at 64 MiB.

In a web or sandbox container without a game install, previews are flat and
contain no game art. Sprite previews and grammar sheets require running Kit on a
local machine with readable game assets. `kit doctor --sandbox` reports
`no install; flat previews only` without treating the missing install as an
error.

## Project Defaults And Crop Shortcuts

Preview searches the working directory and its parents for `kit-preview.json`.
The nearest file supplies defaults; command-line flags override it. Relative
asset paths are resolved from the defaults file's directory. Unknown keys and
malformed configuration fail rather than silently changing the render.

```json
{"dat":"assets/game.dat","game_dir":"assets/install","sprites":true,"textures":true,"shadows":true,"mood":"scenario","elevation":false,"scale":60}
```

Keep asset paths local to your project; do not publish game assets. The supported
keys are exactly those shown above. Use `--no-sprites`, `--no-textures`,
`--no-shadows`, or `--no-elevation` to disable a configured option. For a minimap,
disable incompatible configured options explicitly.
`scale` is the tile width in output pixels (1..128); omitted or zero retains
the built-in default. `--scale` overrides the configured value.

```text
kit preview scene.aoe2scenario --around 30,40 --radius 8 --out close.png
kit preview scene.aoe2scenario --fit-units 1062,1888 --out fences.png
kit preview scene.aoe2scenario --elevation --out slopes.png
```

Choose only one crop mode (`--crop`, `--around`, or `--fit-units`). Around uses
tile coordinates and a positive radius (default 8), clipped to map bounds.
Fit-units bounds all matching objects, including footprints and a one-tile
margin; distant clusters remain in one window. A missing selection is an error.
These shortcuts do not change the scenario or saved snapshot.

`--elevation` draws diamond terrain as a triangular height field. Each stored
level lifts the surface by one quarter of the tile width (`scale/4` pixels),
measured from single-tile and pyramid editor captures through stored level 15.
Shared corners use the maximum of the adjacent tiles, consistent with registered
grid vertices in the available grass plateau, pit and terrace captures. A quad
with one odd-height corner folds through it; ambiguous cases prefer the more
level diagonal, with A-C on ties. Coverage beyond these fixtures remains limited.
Lighting coefficients retain a single Lambert-style fit to flat ground
and all four axial grass faces in one capture, not recovered engine maths.
Opposing-pair averages differ in that capture; the least-squares fit leaves
about six percentage points of error per axial face. Lighting now uses shared
vertex values from central terrain gradients and interpolates them across each
triangle, removing the former abrupt face-lighting seams. With calibrated grass
texture repeat, this reduced sampled slope-vertex luminance RMSE in three
registered views from 9.24/9.16/9.60 to 7.73/7.18/8.62. These are correlated
grass samples with flat-ground brightness normalization, not general engine
accuracy scores. The light field remains an approximation.
Textures, layer masks and border blends retain
their undeformed world coordinates. This option is off by default.

The canvas grows upward by `ceil(max_level * scale/4)` pixels. `--print-extent`,
pixel windows and tile pyramids use that same padded frame; `map_origin_y` and
`elevation_padding` report the ground offset. Objects, particles and their
shadows lift by the ground height at their anchor without changing sprite size.
Attachment is verified only on a flat level-two terrace; bilinear interpolation
of corner heights on slopes remains a candidate, not an engine-calibrated rule.
For editor-saved scenarios, add `--stored-object-heights` with `--elevation`
to use each object's recorded Z, including zero, instead of that approximation.
Flat-control registration of a dense editor capture verified 17 black-tile
markers against saved Z to within 0.05 pixels at a 72-pixel tile scale.
This verifies those marker anchors, not every sprite class or shadow surface.
Authored files can contain placeholder Z values that DE changes on load; use
the editor-saved copy for this mode. The default remains inferred attachment.
Sprites, particles, labels and shadow anchors share the selected lift; frame
padding, pixel windows and tile pyramids include stored-height headroom.
Stored heights must be finite and in 0..255. This option does not change terrain
geometry or establish how the engine computes attachment heights.
Depth sorting still uses flat positions, shadows are not warped across slopes,
and terrain does not occlude objects. Topdown output retains slope shading only.
The categorical elevation layer is a separate diagnostic.

### Registering elevation captures

The Go API `preview.FitDiamondRegistration` fits a screenshot camera from
explicit `RegistrationAnchor` world coordinates, stored heights, and observed
pixel coordinates. It returns tile scale, origin, fitting RMS and per-anchor
residuals. Set `Holdout` on independent test anchors; they do not influence the
camera. The projection is `px=OX+(x+y)*scale/2`,
`py=OY+(y-x-height)*scale/4`. Heights are inputs, not inferred terrain semantics.
Use at least three distinct fitting anchors and inspect residuals rather than
assuming a good fit validates the corner rule.

Render at the fitted scale and obtain the padded origin from `--print-extent`.
For an integer-scale fit, the registered pixel-window offset is the reported
map origin minus the fitted screenshot origin, rounded to pixels. Keep that
rounding error in the comparison; do not assume the origin is simply half the
map height. The API accepts fractional scales, but the preview CLI currently
requires integer scales.

An additional high-ground grass capture at 96 pixels per tile supports the
max-corner geometry through editor UI level 16 (stored 15). Six marker controls
fit at the recorded centroid precision; two held-out markers on UI-level-15
tiles sit one level higher, matching their adjacent UI-level-16 corners.
Four ridge regions favor max corners over mean/min corners by grid contrast,
with median local grid-match offsets of zero pixels. This is capture-specific
evidence, not universal engine validation. Odd-corner diagonal contrast remains
inconclusive. Relative face luminance still differs by region, particularly
on rising `-y` faces; texture, mood and grid effects have not been isolated.
The renderer's fold and lighting rules are unchanged by this calibration.

## Local Missing-Sprite Repair

```text
kit preview assets scene.aoe2scenario --missing-only --fetch --from game-install --to mirror
kit preview scene.aoe2scenario --game-dir mirror --dat game.dat --sprites --strict --out scene.png
```

`--fetch` requires explicit source-install and destination-mirror directories.
The DAT defaults to the source installation's conventional DAT; `--dat` can
override it. Only missing loose SLDs needed by the selected scenario/crop are
copied, matched case-insensitively within the source graphics directory.
Reports list copied, already-present, remaining and unresolved dependencies.
Existing files are never overwritten. Path traversal and links escaping the
source graphics folder or redirecting destination subdirectories are rejected.
Packed DRS extraction, particle assets and terrain assets are not fetched;
remaining dependencies produce a nonzero exit. This is local asset repair,
not a redistribution mechanism: do not publish the mirror.

`--strict` requires sprite rendering and fails before writing a PNG when an
object in the requested view cannot be rendered. The JSON report has `ok:false`
and lists reasons. Intentional outside-crop/disabled-layer exclusions and
optional shadow omissions are not failures. This checks render completeness,
not engine fidelity or passability.

## Facet Coverage

```text
kit scen facets scene.aoe2scenario --dat game.dat --unit 349,1062
```

The report lists each unit's exact stored rotations and counts, DAT-declared
standing-graphic angle slots (`available`), `placed`, `missing`, and unexpected
stored values. Animation frames do not inflate the variant inventory. Use
`--gaia`, `--player N`, or `--crop` to narrow the scope; `--text` emits a compact
summary. Explicitly selected but absent unit types report zero placements.

`complete` compares against the preview's direct-index interpretation for
1.58/1.59, not verified distinct art or engine behavior for every type. The
report does not inspect SLD availability. Missing metadata and particle-only
graphics are marked unknown; old scenario versions retain their stored values
but do not claim coverage until their rotation semantics are calibrated.
Triggers and dynamically spawned units are not included.

## Rotation Calibration

`--rotation-model` is an explicit sprite-preview experiment, never a scenario
conversion. The default remains `stored-index`; file versions and value ranges
do not silently select another model. The report names the selected model and
the source scenario version when known. Saved snapshots retain raw rotations.

| Model | Mapping to a visual angle index |
| --- | --- |
| `stored-index` | Existing preview convention: explicit nonzero snapshot variant, otherwise rounded stored rotation; no wrapping |
| `radians` | Raw rotation / (2 pi), times angle count, rounded to nearest index and wrapped |
| `radians-reversed` | Same candidate with the index direction reversed |

Radian candidates ignore the snapshot's cached variant and use the DAT standing
graphic's angle count, or the installed particle definition's angle count.
SLD animation frames remain separate: preview chooses the first frame of the
selected angle. These candidates do not assert which angular origin or direction
the engine uses; they make competing hypotheses testable without rebuilding.

| Scenario version family | Evidence/status |
| --- | --- |
| 1.36, 1.37, 1.40-1.54 | Raw rotations decoded; visual mapping not engine-calibrated |
| 1.55-1.57 | Raw rotations decoded; visual mapping not engine-calibrated |
| 1.58 | Existing stored-index preview convention; no separate version-specific engine calibration |
| 1.59 | The modern oak fixture confirms stored index N renders SLD visual index N on build 101.103.54800.0; not proof for every unit/particle |

To calibrate an older family, compare the same original scenario/crop, unit IDs,
raw rotations, DAT/assets, and camera view against engine screenshots. Hold all
other preview settings fixed and vary only the model. Record engine build,
scenario version, unit/graphic identity and matching indices; a re-saved modern
scenario alone does not establish the original version's semantics. Screenshots
and rendered game art stay local; only the resulting measured table belongs in
Kit. Until those comparisons are supplied, the legacy rows remain unverified.

```text
kit preview old.aoe2scenario --sprites --rotation-model radians --game-dir game --crop 20,20,40,40 --out candidate.png
```
