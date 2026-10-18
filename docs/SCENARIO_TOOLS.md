# Scenario Tools

`kit scen` reads, analyzes, compares, and safely edits DE 1.58 and 1.59
`.aoe2scenario` files. This guide covers reading and analysis; writing and
recipes are in [`SCENARIO_WRITER.md`](SCENARIO_WRITER.md). Flags and output
keys for every command are in [`API_REFERENCE.md`](API_REFERENCE.md).

`units` (including `--named` and `--text`), `map`, `terrain`, and `kit preview`
also read artwork from older measured versions: 1.36, 1.37, and 1.40 through
1.57 (`1.5` is accepted as 1.50). These are **read-only terrain/unit queries**,
not full scenario support. Versions 1.38 and 1.39 have no corpus evidence and
remain unsupported. Triggers, messages and AI are not decoded by this path;
old files cannot be patched, rebuilt or written.

`units` JSON retains the usual sections and adds `version`, `scope`, and
`rotation_interpretation`. Legacy rotations are the stored values, without
conversion to radians or a claim that every old version's visual semantics
have been engine-tested. Map/terrain queries stop before Units, so they also
work when later sections cannot be decoded. Terrain ids, elevations, layers
and opaque tile bytes are retained; terrain filters and component reports
work as on modern files. Preview uses the user's installed art, which may
differ from the assets originally used by an old campaign.

## Orientation

Start with an unfamiliar scenario here:

```sh
./kit scen glossary big.aoe2scenario --limit 25 --text   # vocabulary of the scenario
./kit scen idioms big.aoe2scenario --text                # named techniques it uses
./kit scen describe big.aoe2scenario                     # players, map, units, triggers, AI files
./kit scen settings big.aoe2scenario --text              # players, victory, diplomacy, options
```

- `glossary` builds a compact vocabulary: effect families, trigger-name
  prefixes, variable ids, XS calls, text samples, and unit, tech, and attribute
  ids. It streams, so it is cheap on huge scenarios.
- `idioms` names detected techniques without claiming author intent: colored
  text, variable HUD text, create/kill display loops, caption label objects, XS
  script-call bridges, trigger relay graphs, timer choreography, and economy
  signals, each with a confidence.
- `describe` composes the readers into one report; `--full --json` emits the
  full parsed field tree. `--no-path` gives stable output for use as a git
  textconv.
- `settings` is the audit view of player slots, AI names, resources,
  diplomacy (all stored copies), allied victory, victory settings, and options.
- `info`, `check`, `units --named`, `strings`, `map`, and `terrain` are the
  focused readers. `terrain` reports per-terrain counts, bounds, centroids, and
  connected components.

## Campaign Census And Hotspots

`kit scen census campaigns/ more.aoe2scenario --gaia --min-maps 3 --json`
counts objects across directories or files, serially, deduplicating file aliases.
Each unit reports counts, scenario spread (`maps`), players, and stored rotations.
`--against our-maps/` adds comparison counts, map spread, deltas and
`only_in_input`. The same `--gaia`, `--player N` and `--crop` filters apply to
both sets. Names come from `--dat`/`--game-dir` when available, otherwise the
built-in unit name table. `--verbose` measures each read in a separate child
process and logs that file's post-exec peak RSS on Linux; other platforms report
measurement unavailable. This measures the reader, excluding the parent's
optional DAT index. It is not the process-lifetime high-water mark reused for
every file, or the parent's footprint inherited briefly during fork. `scen units
--peak-rss` emits the underlying memory JSON on stderr after its normal output.
Unsupported or malformed files fail the command with their path.

`kit scen hotspots map.aoe2scenario --unit 1062,1888 --cell 20 --dat game.dat`
finds occupied cells for chosen units, including explicitly requested trees or
fish. Each cell reports matching anchor bounds and a ready `--crop` argument.
`--text` prints these directly; JSON is the default.

`kit scen hotspots map.aoe2scenario --diverse --top 10 --min-count 5 --dat game.dat`
finds composed Gaia art without choosing units in advance. Default cells are
16x16 and the default count floor is three. Decorative candidates are known DAT
types below 70 or classes 3/14/27/30, excluding tree/fish classes; unknown types
are excluded. This is a declared study filter, not an engine category. The score
is the sum over distinct cell unit ids of `1 + ln(total qualifying map objects /
frequency of that id across the map)`. Rare types weigh more, repeated copies
of one type do not increase diversity, and frequencies remain map-wide when
`--crop` restricts the search. Ties break deterministically by count, variety,
then coordinates. Use `--unit` for targeted study and `--diverse` for discovery.

## Extract And Paste Motifs

```sh
kit scen extract source.aoe2scenario --crop 20,30,36,46 --terrain --relative --out motif.json
kit scen patch base.aoe2scenario pasted.aoe2scenario --recipe motif.json --at 60,60
```

Extraction selects unit anchors inside half-open tile bounds, not sprite or
footprint extents. Optional terrain includes terrain ids, layers and elevation.
Relative mode subtracts the crop origin; `--at` places either relative or absolute
motifs by their crop origin. Terrain pastes require integer origins. Unit-only
motifs allow fractional placement. Normal recipe patch bounds checks still apply.

The recipe records source basename, version and original crop. Transforms record
the current placement separately, leaving that provenance intact. It preserves
unit z, status, stored rotation, initial animation frame and literal caption text.
It does not copy triggers, AI, string-table links or garrison/reference links;
the patch writer allocates fresh unit references.

`--rotate 90|180|270` rotates terrain and anchors together. Every unit must have
an explicit reviewed rotation mapping in the built-in connector grammar or a
`--grammar review.json` subset. Unknown mappings and fractional legacy rotation
values are rejected rather than guessed. Old supported artwork can be extracted
into a recipe, but its source scenario remains read-only.

## Connector Grammar

`kit felt grammar 1062 --out fence.png --json fence.json --game-dir game`
renders every DAT facing at its first animation frame on labelled grass patches
(and water for bridge/boardwalk families). The sheet is 1000 pixels wide with
large labels. Images are generated locally from installed assets. The JSON is
an editable felt compendium subset: each entry has `grammar` variants with
`axis` (`x`, `y`, `diag_ne`, `diag_nw`, `none`), `role`, optional
`renders_along`, `evidence`, and optional `rotates_to` degree-to-variant maps.
Use `--compendium fence.json` to load a reviewed subset on subsequent runs.
Unobserved fields remain unmapped. Built-in seeds are preview-reviewed visual
claims, not engine collision or passability guarantees. Explicit rotation maps
are required to rotate motifs; an axis does not establish corner polarity.

## Triggers

```sh
./kit scen triggers file.aoe2scenario --grep mana --text
./kit scen triggers file.aoe2scenario --effect create_object --player 3 --area 10,10,20,20 --text
./kit scen trigger-neighborhood file.aoe2scenario --unit-ref 75382 --depth 2 --text
./kit scen effects huge.aoe2scenario --census --max-mb 192
```

- `triggers` with no filter prints every trigger with decoded effects and
  conditions. Filters: `--grep` (names, descriptions, messages, summaries),
  `--effect`, `--condition`, `--message`, `--unit-ref`, `--unit-type`,
  `--player`, `--variable`, `--area`.
- `trigger-neighborhood` expands activate/deactivate and trigger-reference
  edges around a trigger, unit, or variable: "what else is attached to this
  mechanic?"
- `effects` summarizes effects per type. `--census --max-mb N` is a count-only
  pass for very large scenarios; `--where EFFECT` streams matching effect sites
  with their decoded fields; `--raw` shows the raw integer fields.
- `trigger-flow` stages matching triggers as grant, use detection,
  cooldown/state, cleanup, refresh, and transition.
- `mechanic --kind garrison-token|transform-toggle|teleport-transition|refresh-cycle`
  classifies common trigger machinery from structural evidence.
- `audit-player-coverage` groups player-named trigger families (`P1 Heal`,
  `P2 Heal`, ...) and flags missing active players. It is a heuristic for
  slot-specific bugs, not a rule that every mechanic must be symmetric.
- `analyze` gives a broad structural analysis in one report.

## Comparing scenarios

- `diff` compares two scenarios by parsed facts: sizes, trigger counts and
  graph hash, trigger changes, terrain counts, per-player units, player slots,
  and AI file hashes. `--all-fields` compares every decoded field by path,
  labeled by editor player slot.
- `diff-triggers` compares trigger graphs; `diff-effects` compares effects.
- `bytediff` compares inflated bodies and attaches changed bytes to sections.
- `field <path> <file|--root DIR>` reads one field, across a folder of
  scenarios if given `--root`.
- `coverage` reports which body bytes are decoded, per file or across a
  corpus.
- `dump-body --inflated` writes the decompressed body for byte-level work.

## Safety checks

```sh
./kit scen verify file.aoe2scenario
./kit scen lint file.aoe2scenario --load-safety --text
./kit scen write-check before.aoe2scenario after.aoe2scenario --text
```

- `verify` rebuilds the body from the decoded model and checks it is
  byte-identical.
- `lint` reports structural and engine-safety risks: trigger invariants,
  duplicate trigger names, empty enabled triggers, looping display effects that
  spam text, map tile-count mismatches, unit count and reference problems
  (including duplicate reference ids and stale unit-id cursors), player-slot
  warnings, embedded AI problems, and corrupt strings. `--load-safety` adds
  checks for known load and launch breakers: player-count mismatches, too many
  active players, locked human civ slots, train-button collisions, and missing
  starter buildings.
- `write-check` is the standard post-write gate: both files rebuild, the
  output lints cleanly, the diff succeeds, and the XS surface inventories
  cleanly.

## XS in scenarios

- `xs` inventories a scenario's XS: attached script fields, inline runtime and
  escrow carriers, `script_call` effects and the functions they call, function
  definitions, includes, and cross-file `extern` hazards. With
  `--deploy-tree`, includes are resolved on disk.
- `xs attach|embed|extract|compare|deploy` manage XS placement; see
  [`XS_AUTHORING.md`](XS_AUTHORING.md).
- `deploycheck <scenario> <deploy-tree>` preflights external XS deployment:
  script name and path match, the file resolves on disk, includes resolve, and
  shared symbols are declared `extern`.

## References and safe deletes

Delete is planned before anything is removed.

```sh
./kit scen refs file.aoe2scenario --kind unit --id 910101 --text
./kit scen delete-plan file.aoe2scenario trigger-prefix "Probe " --text
./kit scen delete in.aoe2scenario out.aoe2scenario trigger-prefix "Probe "
./kit scen disconnect in.aoe2scenario out.aoe2scenario unit 910101
```

- `refs` lists direct references: trigger control, placed units, variables,
  and string-table ids.
- `delete-plan <kind> <target>` answers what deleting a target safely means.
  Unreferenced targets get a physical-removal recipe; referenced targets report
  their blockers plus a `cleanup_command`/`cleanup_recipe` when `disconnect`
  can clear them. String ids are cleared in place, never compacted, because
  compacting would renumber later ids.
- `delete` runs the plan, refuses blocked plans, and applies the recipe to a
  separate output file.
- `disconnect` removes direct references to a target (trigger rows, garrison
  links, string-id fields) and leaves the target itself in place. The usual
  sequence is disconnect, re-plan, then delete.

Target forms accepted by `delete-plan`, `delete`, and (where they apply)
`disconnect`:

| target | selects |
| --- | --- |
| `unit ID`, `trigger N`, `variable ID`, `string ID` | one target by id |
| `unit-caption`, `unit-caption-prefix`, `unit-caption-contains` | placed units by inline caption |
| `unit-type UNIT [--player N]`, `units-player N` | placed units by type or owner |
| `units-area x1,y1,x2,y2 [--player N] [--unit N]` | placed units in an area |
| `trigger-name`, `trigger-prefix`, `trigger-contains` | triggers by name (or effect message for `contains`) |
| `variable-name`, `variable-prefix`, `variable-contains` | variables by name |
| `string-text`, `string-prefix`, `string-contains` | string-table entries by text |
| `system-prefix PREFIX` | a generated bundle: triggers, variables, strings, and captions sharing a prefix |
| `effect T:I`, `condition T:I` | one child row of a trigger |
| `effect-type`, `condition-type` (with `--trigger` or `--trigger-prefix`) | child rows by type |
| `effect-text-prefix`, `effect-text-contains` | effect rows by message text |

Batch deletes allow references inside the matched set and block on
references from outside it. Duplicate names are rejected where a target must
be unique. Deleting triggers that contain `script_call` effects is allowed when
references are clear, but the plan warns that Kit cannot prove the XS side
still makes sense.

## Other tools

- `blank` creates a new scenario; see [`SCENARIO_WRITER.md`](SCENARIO_WRITER.md).
- `import-triggers <src> <dst> [--select ...] [--prefix P] --out FILE` copies
  triggers from one scenario into another, preserving their records.
- `palette-usage --dat` joins placed units to the DAT art catalog and reports
  which rotation/art variants are used (see `kit dat palette`).
- `regions` writes a starter region file for replay context.
- `shop-catalog` turns a store catalog into a patch recipe.
