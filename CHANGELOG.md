# Changelog

## 0.3.0 — 2026-10-14

### Added
- Deterministic Geotrace visual-facet selection for mapped units, using direct
  scenario artwork indices and the selected non-empty visual state.
- A discoverable GitHub Issues feedback protocol in the getting-started, AI,
  and contributor guides.
- Exact platform artifact names and release-run selection guidance for GitHub
  connector users, including checksum matching.

### Fixed
- Geotrace no longer feeds random radian rotations to mapped multi-facet
  objects; deterministic placement selects only indices in the unit's mapped
  visual state. This is a structural correction; the rebuilt map still needs
  an engine visual check.
- Connector instructions identify the successful release run for the selected
  tag (or a refresh run naming that tag), rather than assuming the newest run
  is the correct artifact source.

## 0.2.0 — 2026-10-14

### Added
- Scenario recipe operations `resize_map` and `regenerate_map` for measured
  square editor presets, plus per-player architecture, starting age,
  population limit, colour, active-slot and player-count settings.
- Six scenario message slots with string-table IDs and literal fallback; when
  game strings are unavailable, IDs are preserved and unresolved text is
  reported with a warning.
- Map colour mood authoring and `--game-dir` / `AOE2KIT_GAME_DIR` support for
  message resolution, including `AoE2DE` game-directory discovery.
- Byte-diff save-noise masking and section names, corrected player labels for
  AI names and map-colour-mood diffs, condition editor fields in trigger
  listings, and `scen units --text` output.
- Trust and execution guidance for users and AI-mediated workflows, including
  GitHub connector artifact discovery and verification details.
- GitHub Actions release builds now attest archives and standalone binaries;
  monthly/manual refresh runs rebuild stable artifacts without changing the
  published release. Container bootstrap guidance covers connector artifacts,
  checksum verification, and embedded commit checks.

These scenario operations are structure-verified; they have not yet been
engine-verified. A smoke scenario exists, but no in-game check has been
reported.

## Experimental Geotrace — 2026-10-14

### Added
- Experimental geographic scenario generation: shared projection framing,
  caller profiles, ecology roles and optional user-supplied felt metadata.
- Fine coastline refinement and coarse thin-island preservation, explicitly
  labeled cartographic exaggeration rather than geographic accuracy.
- Stroked vector roads, deduplicated road-chunk merging, bounded resumable fetch integration,
  and reported WorldCover 404 coverage gaps.
- Preplacement terrain/collision checks, final sweep and zero-violation written
  audit; opt-in player starts and atomic neutral resource piles; sheltered-bay
  fish habitat.
- OSM/ODbL and ESA WorldCover/CC BY 4.0 attribution in source-backed reports,
  including known or unknown road-query extents; focused geotrace documentation.
- Gaia buildings and urban accents are available only through an explicit
  unvalidated-feature opt-in, disabled by default. No engine success claim.

### Fixed
- Empty, complete Overpass road files remain valid empty sources instead of
  being misclassified as GeoJSON; source attribution is retained.

No regional profiles, game art, geographic datasets, official RMS sources or
run outputs are included. Static checks do not prove engine behavior.

## DAT, Fetch, and Scenario Writer Fixes — 2026-10-14

### Fixed
- DAT terrain-restriction decoding now reads passability multipliers and pass
  graphics from their separate arrays, preserving accurate byte spans.
- Scenario unit placement carries its resolved grid/fractional snap policy
  through the AddUnits handoff.
- Batch scenario unit removal validates references before mutation and removes
  all requested records without shifting later indices.

### Added
- `pkg/fetch` provides bounded, resumable HTTP fetch sessions with validated,
  atomic cache writes. It is a standalone package; providers and callers own
  source-specific validation and attribution.

## First Community Pull Requests — 2026-10-06

Kit's first pull requests from outside, all from SpiRaL (#1-#4).

### Fixed
- **Windows:** the pinned xs-check is found as `xs-check.exe` and verified
  against the Windows release digest, so the strict XS checks work on Windows
  without setting `AOE2KIT_XS_CHECK` (#2).
- **Windows:** `go test ./cmd/kit` no longer hangs; the help text was larger
  than a Windows pipe buffer (#3).
- `kit scen lint` and `kit scen blank` name players P1-P8 like the editor; they
  used to call the second player "P1" (#1).

### Added
- `kit scen patch` accepts `--text` and `--json`, and names an unknown flag
  instead of only printing the usage line (#4).

## Fixes & Panel Text Alignment — 2026-10-05

### Changed (breaking)
- **Recipe player numbering is now P1..P8 everywhere.** In scenario recipes,
  `players`, `diplomacy`, and `resources` use 1-8 for the eight players, the
  same as the editor. Before, these three sections counted from 0, so
  `"player": 1` edited the second player. A recipe that still uses 0 there now
  fails with a clear error instead of editing the wrong player. To update an
  older recipe, add 1 to every player number in those three sections. Units are
  unchanged: 0 is Gaia and 1-8 are the players.
- `kit scen settings` and diffs label players P1-P8, matching the editor and
  `kit scen lint`.

### Added
- **Panel text alignment** (`pkg/text`): measure a string with the game's
  per-character font widths and generate an XS helper that measures runtime
  text, so text in the timer and instruction panels can be lined up in
  proportional fonts. Technique credited to SpiRaL. Structure-verified; you
  supply the font metric data. See `docs/TEXT_API.md`.
- `kit replay feedback --prior <earlier recording>` tags chat carried over from
  a previous game, even when it has its own timestamp.
- Civ names for Saxons, Varangians, and Danes (ids 60-62).
- The command catalog and `docs/API_REFERENCE.md` now cover `dat info`,
  `dat check`, `dat roundtrip`, `replay health`, `replay postgame-corpus`,
  and `replay opaque-target`.

### Fixed
- `kit replay feedback` could show the wrong player name; it now uses the
  replay's roster.

## Own Codecs & DE 1.59 — 2026-09-30

AoE2Kit now decodes scenarios entirely with its own typed codecs, reads and
writes DE 1.59, and ships a much tighter public release.

### Added
- **DE 1.59 scenarios, read and write.** Every 1.59 section is decoded and
  rebuilds byte-for-byte, including both condition layouts and the extra unit
  byte. One official trigger-record variant is detected structurally and
  preserved opaquely. See `docs/SCENARIO_159_NOTES.md`.
- **`kit scen import-triggers`**: copy selected triggers from one scenario into
  another, preserving their records, with an optional name prefix. Requested
  by Sekiro.
- **Trigger type fields in recipes**, and a calibrated **Replace Object**
  effect, checked against editor-saved fixtures.
- **Editor normalization** is documented: the values the DE editor fills in on
  save (trigger names, effect defaults, map defaults). See
  `docs/AOE2KIT_WRITE_SMOKE.md`.
- **Scenario analysis:** `scen diff --all-fields` (every decoded field, by
  path), `scen field` (one field, across a folder), `scen coverage`
  (decoded-byte accounting), `scen bytediff`, per-unit and settings diffs, and
  diplomacy's three stored copies in `scen settings`.
- **Unattended engine harness** (`kit xs harness`): generates probe scenarios
  that write typed sidecars, with six batches including a 91-fact capability
  pass. See `docs/ENGINE_HARNESS.md`.
- **Gaia text library** (`examples/text/gaia_text.xs`): a standalone XS
  library that draws runtime text, numbers, and paths with Gaia objects. See
  `docs/TEXT_API.md`.
- **Per-object custom stores** (`kit rpg custom-store`). See
  `docs/CUSTOM_STORE.md`.
- **DAT:** full unit decode with a SQLite export (`kit dat sql`), a lazy
  SQLite unit cache, semantic-prior queries, and unit footprint fields.
- **Replays:** `replay health` for problem recordings, a bounded disk-backed
  replay cache, and 1.59 trigger graphs embedded in recordings.
- **XS checking:** lint from the UGC Guide's published syntax-hazard catalogue,
  builtin-name collisions, forward calls, int-first arithmetic, non-ASCII in
  strings, and arithmetic in conditions. Optional external
  [xs-check](https://github.com/Divy1211/xs-check) gate for generated XS.
- **Docs:** domain guides `docs/SCENARIO_TOOLS.md`, `docs/REPLAY_GUIDE.md`,
  `docs/DAT_AUTHORING.md`, and `docs/CBA.md`; `docs/GO_AOE2KIT.md` is now the
  architecture and conventions guide.

### Changed
- The AoE2ScenarioParser-derived schema and its expression rules are removed.
  Kit's scenario layout is its own, reconstructed from a corpus of real files;
  AoE2ScenarioParser remains credited as a consulted format reference.
- Scenarios before DE 1.58 are rejected with `unsupported scenario version`
  instead of being read through a compatibility path. Re-save them in the DE
  editor to convert.
- `kit scen blank` adds a Barracks per active player by default
  (`--no-starters` to omit), and blocking units are placed on tile centers.
- Explicit unit reference ids are reserved before automatic ids are assigned,
  and `scen lint` reports duplicates.
- Trigger `modify_attribute` operations use the trigger enum (Set = 1), not
  XS's.
- Absent DAT unit references read as `-1`.
- XS facts are scoped to the game build that produced them; see
  `docs/ENGINE_BUILD_SCOPE.md` and `docs/ENGINE_UPDATE_185872.md`.

### Removed
- **The public release is now default-deny.** It ships exactly the files in
  `release/public-files.json`; `kit pack --profile public` has no other rules.
- Game-derived art (felt-compendium contact sheets and sprite samples) is no
  longer in the public release.
- Internal test readbacks, probe specifications, and task notes, and the
  engine-facts ledger (`data/engine_facts.json`), are no longer published.

### Known limits
- Scenarios before DE 1.58 are not read.
- Some effect-command meanings (local-building types 202 and 204) remain
  strong hypotheses until an engine fixture confirms them.

## v0.1.0 — 2026-10-14

### Added
- AI-container onboarding, stable latest-release downloads, build metadata, and `kit doctor --sandbox`.

## Geo-Trace & Felt — 2026-09-16

AoE2Kit can now trace real Earth into playable maps, and author terrain by aesthetic *intent*
instead of terrain IDs.

### Added
- **`kit geo trace-dem` — real-map tracing.** Aim a viewport (center + span, or a bbox) at any
  latitude/longitude and render real geography into a scenario: elevation, coastlines,
  land-cover (ESA WorldCover / GLC_FCS30D forest types), lakes and rivers, interstates, and
  political borders — translated through a **felt** palette and **climate-aware** from real data
  (warm sandy shores vs. cold rock; tropical vs. boreal species).
- **Felt compendiums.** Queryable indexes of what terrains, gaia art, species, and terrain
  *layerings* feel like — so you can ask for "a cold rocky shore" or "an Everglades gloom"
  instead of memorizing IDs. Includes contact sheets, terrain sheets, and sampled animations.
- **Organic terrain primitives.** Composable recipe ops — masks, `noise_fill`,
  `erode` / `semantic_erode`, `cluster_scatter`, `layered_crossfade` (top/bottom terrain
  blending), and a bulk `terrain_grid` op for fast full-map terrain + layer writes.

### Changed
- SLD sprite stats/export improvements, including compact first-frame headers.
- Documented the geo-trace CLI and the scenario authoring primitives.
- Credits are now split into **Built On** (studied prior art) and **Personal Thanks**
  (individual people who directly helped); `NOTICE.md` is generated from `data/credits.json`.
- Public packaging pass (`portable-check`, `pack --profile public`); the terrain scrape no
  longer bakes a local path.

### Known limits
- Full-resolution (480) raster land-cover sampling isn't container-friendly yet (multi-GB RSS);
  the container path uses pre-fetched vector data + `--omit-report-cells` / `--omit-report-recipe`.
- Geo-trace output is structure-verified; the in-engine look is confirmed by eye, not
  automatically.

### Thanks
**Sekiro** taught us **terrain layering** — the per-tile `layer` field, used as a top/bottom
crossfade — and it reshaped this entire release. Every graded coastline, every water-depth
shelf, every biome seam, the layered political borders, and the whole hybrid-terrain approach
are built on it. Sekiro is also AoE2Kit's first active external user. **Thank you.**

Thanks also to **SpiRaL** and **krmyth9**, fellow designers who have directly helped along the
way. Full attribution and the standing gratitude ledger live in `NOTICE.md` / `data/credits.json`.
