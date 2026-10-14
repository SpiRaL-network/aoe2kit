# AoE2Kit Architecture and Conventions

`aoe2kit` is a Go toolkit for Age of Empires II: Definitive Edition scenario,
replay, data-mod, XS, and packaging work. It is a single Go module whose direct
dependencies are pure Go (`golang.org/x/image`, `modernc.org/sqlite`), so it
builds with `CGO_ENABLED=0` into one static binary.

This page covers how the kit is organized and the conventions every command
follows. Each domain has its own guide (see [Guides](#guides)), and every
command's flags and output keys are in the generated
[`API_REFERENCE.md`](API_REFERENCE.md).

## Build

```sh
go build -o kit ./cmd/kit
./kit                      # list every command
```

For a static binary to drop into a sandbox without a Go toolchain:

```sh
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -buildvcs=false -o kit ./cmd/kit
```

## Layout

```text
cmd/kit          the CLI: scen, replay, dat, xs, xsdat, gfx, fx, ai, mod, cba, ...
cmd/apiref       generates API_REFERENCE.md, api_reference.json, api_schemas.json, PACKAGE_API.md
cmd/scen|dat|mod|swatch|cba   optional single-domain entry points

pkg/scenario     .aoe2scenario codecs, writer, lint, XS deploy checks
pkg/replay       .aoe2record parsing: header, actions, sync matrix, objects, coverage
pkg/datfile      .dat span/index inspection and patch engine
pkg/datcodec     typed .dat section codecs (effects, techs, civs, sounds, terrains, ...)
pkg/datcache     SQLite cache for DAT unit queries
pkg/xs           XS authoring generators, lint, and the .xsdat sidecar codec
pkg/harness      unattended engine-probe scenario generator
pkg/cba          Castle Blood Automatic interpretation (domain layer)
pkg/geotrace     real-map terrain tracing into scenario recipes
pkg/kit          bundle inventory, verification, packaging, portability checks
pkg/...          supporting packages: gfx, fx, glyph, text, geom, aifile, modpack, campaign, ci, ...
```

Package documentation for importing the library directly is in
[`PACKAGE_API.md`](PACKAGE_API.md).

## Geotrace Profiles

Geotrace is **experimental**. See [GEOTRACE.md](GEOTRACE.md) for inputs,
attribution, missing-data behavior, opt-in features and validation limits.

`kit geo trace-dem --profile profile.json --seed 7` separates geographic evidence
from game-facing choices. Profiles use schema `aoe2kit-geotrace-ecology/v1` and
can supply weighted habitat layers, terrain palettes, explicit geographic
overlays and landmark accents. No location-specific landmarks are implicit.
The optional `notes` object preserves author intent and provenance in reports;
it is metadata only and never changes generation. Put operative regional
choices in layers, palettes, species mappings and overlays.
`--projection-aspect` (default 2) and `--projection-north-scale` (default 1)
control framing; the same transform places terrain, roads and calibration axes.

Player starts are disabled unless a profile supplies `player_starts`. An explicit
start package defaults to two players with a town centre, three villagers,
a scout and equal resource packages. `player_starts` controls
`mode` (`explicit`, `engine`, `none`, or `calibration`), `count`, unit IDs,
`clear_radius`, `min_separation`, and an `inventory` of role/unit/count/distance
groups. `clear_radius` reserves the connected resource-search pocket.
`flora_clearance` defaults to six tiles beyond `town_center_footprint` (four
tiles); only that core and narrow approaches to required piles are cleared.
Villagers use a two-and-one arrangement at offsets `(1,5)`, `(2,5)`, `(-5,0)`
from each TC. `engine` leaves players empty; automatic initialization is not verified.
Only `calibration` adds test barracks and projection markers. Small preview
grids omit automatic starts. Explicit starts clear connected dry pockets;
the report records inventories, positions and the source/policy distinction.

Scenario output requires `--dat`. Vegetation is placed before fauna. A named
`placement_gate` checks proposed, snapped positions against the finished terrain
before individual objects are written. `placement_sweep` checks serialized positions against final terrain and DAT
collision footprints, removes invalid ambient objects, then reopens the result
for `written_placement_audit`. Publication requires zero remaining violations;
an invalid required start object fails instead of reducing that player's package.
This static gate does not prove engine pathfinding. Land flora is excluded from
water; profile `flora_water_terrains` maps water-tolerant unit IDs to explicit
terrain exceptions, still subject to DAT eligibility. Water-flora layers are
limited to shallow/freshwater terrain. Reports distinguish candidate budgets,
post-clearing counts, sweep removals and final audit counts.
The exported generation recipe precedes the sweep; `placement_sweep_recipe`
records a separate follow-up patch against that generated file's reference IDs.

`neutral_resources` configures scattered mineral piles: `name`, `unit`,
`pile_size`, `groups_per_10000`, `spacing`, `start_distance`, and `provenance`.
Neutral piles are disabled unless explicitly supplied in the profile. A suggested
gameplay policy uses four-object piles with 48-tile spacing and start separation,
and targets 1.5 gold groups and one stone group per 10,000 map tiles. These are
explicit gameplay policies informed by Nomad and remote-resource RMS norms.
Reports give requested and realized groups; placement constraints may reduce them.
An empty array disables neutral piles. Terrain and blocker checks also apply.

At nominal tile sizes up to 250 m, WorldCover samples with selected pixels
no larger than one third of a tile can refine the source landmask in either
direction before shoreline distances are calculated. Unrecognized classes,
nodata and insufficient resolution retain the source mask. Between 250 m and
1,500 m, a preservation mode selects pixels no larger than one twelfth of a
tile and examines an approximately tile-sized footprint. It retains source land
and adds land when any recognized land pixel occurs in that footprint.
Mixed coastal cells use the land-only representative class; their source class
and land fraction remain in the report. This intentionally exaggerates thin
islands to playable tile width, and is not exact cell-area coverage. Coarser maps
retain the source-mask coastline policy. These are visualization thresholds,
not accuracy guarantees. `coast_refinement` reports the policy, fallback counts
and land/water flips; explicit inland-water polygons still override.

Each habitat layer has a unique `name`, placement `role` (`resource`,
`decorative`, `fauna`, `fish`), weighted `units` (`unit_const`, `weight`),
`rate` (objects per eligible tile), `limit`, and optional minimum `spacing`,
`patch_scale`, `landcover_keys`, `climates`, `water_band`, or `terrain_ids`.
Water bands include `bay`: a sheltered-water proxy with land in at least four
of eight directions within 32 tiles, including an opposing pair of banks. This describes gameplay habitat rather
than measured salinity, depth or exposure. Inland-water evidence takes precedence.
An optional `ecology_role` can select a profile `palettes` entry instead of
inline units. Role vocabulary includes `NERITIC`, `SALTWATER`, `FRESHWATER`,
`RIPARIAN`, `PREDATOR`, `HUNTABLE`, `HUNTABLE_SMALL`, `LUREABLE`, `FURBEARER`,
`HERDABLE`, `FORAGE_PLANT`, `BIRD`, `OYSTER`, and `WHALE`. `OYSTER` uses compatible
beach and shallow terrain instead of the open-water fish rule. A `biome` label is explicit
metadata, not an automatically inferred ecological classification.
Omitted/null `layers` use defaults; `layers: []` disables procedural ecology.

With `--compendium art.json`, layers can replace `units` with `felt` selectors.
Each selector supplies `functions`, optional `tags`, `materials`, `unit_consts`,
and a positive `weight`. Filters are OR within a category and AND across
categories. Its weight is shared between matching artworks, preventing large
families from dominating merely by entry count. Human felt takes precedence
over render-based and metadata-based judgments for duplicate unit IDs.
Placement hints constrain coast, wetland/edge, shallow water, land interior,
and roadside placement. Unsupported hints are not silently scattered.
The resolved palette, compendium checksum, and per-layer `family_counts`
(form plus placement hint) are recorded in reports. Profiles may also supply
`species_units` to map additional atlas taxa to game-art analogues; this does
not imply that the artwork depicts the exact botanical species.

Layer `bounds`, `min_shore_distance`, and `max_shore_distance` constrain
regional habitats. Projected-map corners outside the nominal requested bbox
remain eligible. Reports distinguish sampled coverage from missing evidence;
road geometry bounds alone never prove source query coverage. Local road JSON
may declare its original query extent in `kit_query_bbox`.
New road caches retain that extent automatically. Network road fetches print
chunk and endpoint progress to stderr and share a ten-minute overall deadline;
`--roads-timeout 25m` overrides it. Partial Overpass responses are rejected,
not treated as empty geographic coverage.
Completed chunks are cached atomically and reused after an interrupted fetch.
A mirror that times out while dialing is skipped for the remainder of that
fetch; another invocation can try it again.
Road vectors are simplified before rasterization, buffered as polylines with
round joins and caps, then rasterized against tile footprints. The base width
comes from `--road-width-tiles` (default one tile, maximum 32); OSM motorway,
trunk, primary, secondary, and tertiary classes apply descending width factors.
Explicit OSM gravel/unpaved surface tags select the DAT gravel-road terrain;
other surfaces use standard road terrain. Strokes still obey land and
near-shore causeway eligibility. The report records final road-cell counts.

Resource trees use irregular patches with independently mixed species and grid
placement. Decoration uses the scenario writer's DAT-backed `auto` footprint
policy, with conservative grid fallback when DAT evidence is unavailable.
USGS woody presence/absence data selects a palette within 50 km, not tree
abundance or exact locations. Without landcover, the sparse tree fallback is
an artistic default, not measured forest cover. Fish spacing applies across
water-band boundaries; requested quotas need not be achievable.

Reports include per-layer counts, source-mask connectivity, shoreline terrain
pairs, and DAT-filtered dock-footprint candidates when `--dat` is supplied.
These are necessary terrain checks, not an engine construction guarantee:
elevation, obstructing objects and final dock placement still need validation
in the game. Missing DAT evidence is reported as unavailable, never passed.

## Conventions

**JSON first.** Every command emits JSON by default. `--text` renders a
human-readable view; programs should parse the JSON.

**Every claim carries its strength.** Reports carry a verification label:
`structure_verified` means Kit re-read the data and it round-tripped;
`engine_verified` means the game or editor was the oracle; labels such as
`*_candidate`, `behavior_inferred_*`, and `*_not_engine_verified` mark
hypotheses. `verification.engine_verified=false` means the game has not been
used to confirm the result. Do not promote a claim past its label.

**Writes never touch the input.** Every patch command writes a separate output
file, reopens it, re-verifies it, and reports readback before claiming success.

**Fail closed.** Unrecognized structures are reported as errors or preserved
byte-for-byte as opaque data; Kit does not guess.

**Bounded output.** Table commands (`dat graphics`, `dat units`, ...) return
the first 200 rows by default and say so; use `--limit 0` or `--all` for full
tables.

**Resource guardrails.** `kit` sets a 1 GiB soft Go heap limit unless
`GOMEMLIMIT` or `AOE2KIT_GOMEMLIMIT_MB` is set. Scenario parsing refuses
inflated bodies above 32 MiB unless `AOE2KIT_MAX_SCENARIO_MB` is raised or
`AOE2KIT_ALLOW_HUGE_SCENARIO=1` is set. Run heavy replay commands one at a
time; see [`REPLAY_MEMORY.md`](REPLAY_MEMORY.md).

**Generic truth and domain interpretation stay separate.** `pkg/replay` and
`pkg/scenario` describe file formats. Scenario-specific interpretation (for
example `pkg/cba`) consumes those facts without redefining them. New shared
capability goes into a package first and is then exposed through a CLI domain.

**Identify replays from the replay.** Do not infer what was played from the
newest file in a scenario folder: DE recreates some built-in scenario state on
its own. Use `kit identify`, `kit replay info`, or `kit replay summary`.

## Guides

| domain | guide |
| --- | --- |
| scenario reading, analysis, deletes | [`SCENARIO_TOOLS.md`](SCENARIO_TOOLS.md) |
| scenario writing and recipes | [`SCENARIO_WRITER.md`](SCENARIO_WRITER.md), [`SCEN_RECIPE_EXAMPLE.json`](SCEN_RECIPE_EXAMPLE.json) |
| DE 1.59 format notes | [`SCENARIO_159_NOTES.md`](SCENARIO_159_NOTES.md) |
| editor/game write smoke test | [`AOE2KIT_WRITE_SMOKE.md`](AOE2KIT_WRITE_SMOKE.md) |
| replays | [`REPLAY_GUIDE.md`](REPLAY_GUIDE.md), [`REPLAY_CORPUS.md`](REPLAY_CORPUS.md), [`REPLAY_MEMORY.md`](REPLAY_MEMORY.md) |
| Microsoft replay/stats API | [`AOE_MS_REPLAY_API.md`](AOE_MS_REPLAY_API.md) |
| `.dat` engine architecture | [`DAT_ENGINE.md`](DAT_ENGINE.md) |
| `.dat` authoring, particles | [`DAT_AUTHORING.md`](DAT_AUTHORING.md), [`DAT_RECIPE_EXAMPLE.json`](DAT_RECIPE_EXAMPLE.json) |
| `.dat` SQL export, priors, GoKu audit | [`DAT_SQL_EXPORT.md`](DAT_SQL_EXPORT.md), [`DAT_SEMANTIC_PRIORS_2026-09-22.md`](DAT_SEMANTIC_PRIORS_2026-09-22.md), [`GOKU_DAT_AUDIT.md`](GOKU_DAT_AUDIT.md) |
| XS authoring | [`XS_AUTHORING.md`](XS_AUTHORING.md), [`XS_AUTHORING_TIER1.md`](XS_AUTHORING_TIER1.md), [`XS_CHECK.md`](XS_CHECK.md) |
| XS sidecars and engine probes | [`SIDECAR.md`](SIDECAR.md), [`XS_SIDECAR_TECHNIQUE.md`](XS_SIDECAR_TECHNIQUE.md), [`ENGINE_HARNESS.md`](ENGINE_HARNESS.md) |
| engine-verified XS facts | [`XS_ENGINE_VERIFIED_2026-09.md`](XS_ENGINE_VERIFIED_2026-09.md), [`ENGINE_BUILD_SCOPE.md`](ENGINE_BUILD_SCOPE.md), [`ENGINE_UPDATE_185872.md`](ENGINE_UPDATE_185872.md) |
| Gaia text library | [`TEXT_API.md`](TEXT_API.md) |
| experimental geographic scenarios | [`GEOTRACE.md`](GEOTRACE.md) |
| per-object custom stores | [`CUSTOM_STORE.md`](CUSTOM_STORE.md) |
| Castle Blood Automatic | [`CBA.md`](CBA.md) |

## Packaging and release

```sh
./kit verify .                                   # buildability, required docs, manifest drift
./kit portable-check . --profile public          # packaging and leak checks
./kit pack AoE2Kit.zip --profile public --sha-sidecar
```

`kit pack` profiles:

| profile | contents |
| --- | --- |
| `full` | everything tracked, plus the built binary |
| `handoff` | source, data, and root documents, without `docs/` |
| `sandbox` | `handoff` plus a static linux-amd64 binary, for containers without Go |
| `public` | default-deny: exactly the files listed in `release/public-files.json` |

`kit pack` verifies the kit before packing and reports the archive SHA-256;
`--sha-sidecar` writes it next to the zip. `kit portable-check` adds
profile-specific checks: buildability, manifest health, local-path and
personal-name leak detection, and refusal of executable payloads in public
archives.

Other project commands: `kit project inspect|snapshot|diff|lineage` map and
track a project folder before and after edits; `kit recipe list|show|export`
exposes starter scenario and DAT recipes; `kit docs lint` checks command
examples in Markdown against the live command catalog; `kit credits` prints
the attribution ledger behind `NOTICE.md`.

## Regenerating the reference

After a CLI change:

```sh
go build -o kit ./cmd/kit
go run ./cmd/apiref
```

This writes `docs/API_REFERENCE.md`, `docs/api_reference.json`,
`docs/api_schemas.json`, and `docs/PACKAGE_API.md`. The runtime probe executes
read-only commands against whatever fixtures you pass (`--scenario`,
`--replay`, `--dat`, `--xsdat`, `--folder`); commands without a fixture are
labeled rather than guessed. `--reuse-probe docs/api_reference.json` reuses
the previous probe results when only documentation changed, and
`--probe=false` skips execution entirely.
