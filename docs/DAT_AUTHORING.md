# DAT Authoring

`kit dat` reads and edits AoE2DE `empires2_x2_p1.dat` files. How the engine
works (span index, owned-record splices, verification) is in
[`DAT_ENGINE.md`](DAT_ENGINE.md); this guide covers what you can read and
write. Flags and output keys for every command are in
[`API_REFERENCE.md`](API_REFERENCE.md).

Every write goes to a separate output file and is verified by re-indexing the
output to EOF, reading the patched values back, and checking neighboring
records. Structure-verified output means the file is internally consistent,
not that the game behaves as intended.

## Reading

```sh
./kit dat info empires2_x2_p1.dat
./kit dat units empires2_x2_p1.dat --name-contains castle --class building
./kit dat unit empires2_x2_p1.dat 0 83
./kit dat effect-explain empires2_x2_p1.dat 1 --text
./kit dat refs empires2_x2_p1.dat tech 244
```

- **Units:** `units` (filter by id, `--name`/`--name-contains`, `--civ`,
  `--class building|unit|creatable`), `unit`, `unit-headers`, `unit-header`,
  `availability` (per-civ cards: record present, enabled flag, train locations,
  tech-tree links, and a structural verdict).
- **Techs and effects:** `techs`, `tech`, `tech-explain`, `effects`, `effect`,
  `effect-explain`, `tech-tree` (`--full` for every connection row),
  `command-matrix` (a census of effect-command types and operands across the
  DAT).
- **Graphics and art:** `graphics` (`--particle`, `--name-contains`), `graphic`
  (`--spans` for internals), `palette` (units joined to their standing art,
  classified by angle and frame counts), `sprite` / `kit gfx export` (SLD
  frames to PNG with a contact sheet; main layer only), `kit gfx info`.
- **Other sections:** `civs`, `terrains`, `terrain`, `terrain-restrictions`,
  `sounds`, `sound`, `player-colours`, `random-maps`, `spans`.
- **Comparison and safety:** `diff base.dat mod.dat` (decoded records by
  stable key; `--section units` for per-civ units), `roundtrip` (decode every
  section and re-encode byte-identically), `refs` (who points at an id).
- **Queries:** `sql` exports a SQLite view of the unit table
  ([`DAT_SQL_EXPORT.md`](DAT_SQL_EXPORT.md)); `semantic-priors` summarizes
  data-derived priors ([`DAT_SEMANTIC_PRIORS_2026-09-22.md`](DAT_SEMANTIC_PRIORS_2026-09-22.md));
  `dat facts` lists catalogs of effect types, attributes, resources, task
  types, and tech modifiers ([`GOKU_DAT_AUDIT.md`](GOKU_DAT_AUDIT.md)).

`effect-explain` and `tech-explain` translate command rows into authoring
language: typed references, named unit classes, attributes, resources, and
operations, and packed attack/armor values (for example `771` reads as
`attack[pierce +3]`). Rows whose structure is known but whose runtime
behavior is not engine-verified carry a warning.

## Direct edit commands

Each command reads one DAT and writes another.

| area | commands |
| --- | --- |
| units | `patch-unit`, `unit-create`, `unit-delete`, `availability-set` |
| graphics | `patch-graphic` (alias `graphic-patch`), `graphic-create`, `graphic-delete` |
| effects | `effect-create`, `effect-patch`, `effect-disable`, `effect-delete` |
| techs | `tech-create`, `tech-patch`, `tech-delete` |
| abilities (paired tech + effect) | `ability-create`, `ability-patch`, `ability-disable`, `ability-delete` |
| sounds | `sound-create`, `sound-patch`, `sound-delete` |
| player colours | `player-colour-create`, `player-colour-patch`, `player-colour-delete` |
| civs, terrains | `civ-patch`, `terrain-patch`, `terrain-restriction-patch` |
| tech tree | `tech-tree-connection-create`, `tech-tree-connection-patch`, `tech-tree-connection-delete` |

```sh
./kit dat patch-unit in.dat out.dat 0 83 --hit-points 26 --type50-attack 2,value=9
./kit dat patch-graphic in.dat out.dat 3396 --particle-effect-name my_fireball
./kit dat availability-set in.dat out.dat 448 --all-civs --enabled true
```

`patch-unit` edits fixed fields (class, HP, line of sight, movement type,
graphics, icon, enabled, attributes) and rows given as
`INDEX,field=value[,field=value...]`: `--attribute`, `--damage-graphic`,
`--type50-attack`, `--type50-armour`, `--cost`, `--train-location`, `--task`.
`patch-graphic` edits names, SLP, particle binding, layer and colour fields,
sound ids, frame and timing fields, and sequence flags.

## Recipes

For anything beyond one edit, write a JSON recipe
([`DAT_RECIPE_EXAMPLE.json`](DAT_RECIPE_EXAMPLE.json)) and apply it:

```sh
./kit dat plan in.dat --recipe recipe.json           # verify in memory, write nothing
./kit dat patch in.dat out.dat --recipe recipe.json
./kit recipe list --domain dat --text                # starter recipes
```

`patch` applies graphic and unit operations first, then codec operations,
verifying after each. Supported operations:

- **Graphics:** patch fields, replace `set_deltas` and `set_angle_sounds`
  child lists, `create_graphic` (clone a template to the next id).
- **Units:** patch fields, replace damage-graphic, attack, armour,
  train-location, drop-site, and task lists (count-changing allowed; task rows
  must keep their exact `raw_tail`), `create_unit` (clone across chosen civs).
- **Effects:** `create_effect` (empty, from `commands`, or cloned
  `from_effect`), patch with `commands` / `remove_commands` /
  `append_commands`, `disable_effect` (keep the id, clear the commands),
  `delete_effects` (physical removal with id rewrites).
- **Techs:** `create_tech`, patch scalar fields, tail delete, and guarded
  non-tail delete (below).
- **Abilities:** `create_ability` (paired tech + effect, commands cloned from
  a tech or effect plus `append_commands`), `disable_ability`,
  `delete_ability`.
- **Other sections:** civ fields and resources, terrain names, terrain
  passability rows, sound create/patch/mute, player-colour create/patch and
  tail delete, unit-header task rows, tech-tree connection rows (patch,
  clone-append, delete).
- **Designer intents:** `unit_availability` (set the enabled flag by civ),
  `reference_rewrites` (move or remove references to a unit or tech across
  verified surfaces), `disconnect_techs` and `disconnect_units` (remove verified
  references and report what remains).

### Effect commands

Commands can be raw `{ "type", "a", "b", "c", "d" }` rows or named helpers with
typed operands: `disable_tech`, `enable_unit`, `upgrade_unit`,
`set_attribute` / `add_attribute` / `multiply_attribute` (with optional unit
class), the exact-unit forms `set_unit_attribute` / `add_unit_attribute` /
`multiply_unit_attribute`, `resource_modifier`, `resource_multiplier`,
`spawn_unit`, `modify_tech`, `set_tech_cost` / `add_tech_cost`,
`tech_time_modifier`, and the local-building helpers
(`multiply_local_building_attribute`, `add_local_building_armor`,
`add_local_building_attack`). Operands can be given by XS constant names
(`cAttributeKills`, `cAttributeAdd`, `cAttrSetGoldCost`, `cCavalryClass`) or
plain names (`hit_points`, `cavalry`); if both an id and a name are given they
must agree. When Kit clones existing commands it re-expresses known command
types as named helpers and keeps unknown ones raw.

Command-type naming follows Advanced Genie Editor for types `0`–`8`, `10`–`18`,
`20`–`28`, `30`–`38`, `40`–`48`, and `101`–`103`. Types `200` and `201`
(local-building set and add) are named from official DE update notes. Types
`202` (local-building multiply) and `204` (advanced local-building add, with
packed attack/armor) are strong hypotheses from DAT evidence and await an
engine fixture; `kit dat semantics-pack --feature local-building-effects`
generates one.

## Deleting safely

```sh
./kit dat delete-plan in.dat tech 244 --text
./kit dat delete in.dat out.dat tech 244
./kit dat disconnect in.dat out.dat unit 83
```

`delete-plan` answers what deleting a target safely means before anything is
mutated; `delete` runs the plan and refuses unsupported or cleanup-only
results; `disconnect` removes verified references without deleting the target.

| target | what delete means |
| --- | --- |
| effect | unreferenced: physical removal with effect-id rewrites; referenced: clear the commands, keep the id |
| tech | unreferenced tail: physical removal; unreferenced non-tail: move the tail tech into the slot when all its references can be rewritten; referenced: blocked, with a `disconnect_techs` cleanup |
| ability | paired tech + effect delete when the effect is private; otherwise `disable_ability` |
| unit | semantic: `enabled=0` for the chosen civs, optionally with `disconnect_units` |
| sound | semantic: probabilities set to `0`, id kept |
| graphic | unreferenced: pointer zeroed and record removed, ids preserved; referenced: blocked |
| player colour | unreferenced tail row only; no compaction |
| child rows | effect commands (`effect-command 12:3`), sound items, graphic deltas and angle sounds, unit-header tasks, unit attack/armour/damage-graphic/train-location/drop-site/task rows |
| tech-tree connection | removes the relationship row only |
| civ, terrain, terrain restriction, random map | unsupported, with a stated reason |

Reference rows are classified as `rewrite_supported`, `known_readonly`,
`possible_operand`, or `unsupported`. `candidate_operand` is the subset of
possible operands that match a command's shape. Candidate and possible rows
are evidence for inspection, not surfaces Kit will rewrite.

## Particles (`kit fx`)

```sh
./kit fx new my_trail --preset trail
./kit fx bind --dds atlas.dds --grid 4x4 --into resources/_common --dat empires2_x2_p1.dat --unit 83 --slot flying --from 1234 --name my_trail
./kit fx lint path/to/mod/resources/_common --dat empires2_x2_p1.dat
```

- `fx new` prints a particle descriptor (presets `trail`, `explosion`, `aura`,
  `projectile-fire`).
- `fx bind` writes the three files a particle needs (descriptor
  `particles/<name>.json`, atlas `particles/textures/atlases/<name>_atlas.dds`,
  metadata `particles/<name>_atlas.json`) and patches the DAT graphic.
  `--frames N --frame-size WxH` can replace `--grid`. `--atlas file.png`
  converts with ImageMagick
  (`convert <png> -define dds:compression=dxt5 -define dds:mipmaps=0 <dds>`);
  otherwise supply `--dds`.
- `fx lint` validates descriptors, atlas coverage, DXT5 headers, and DAT
  `particle_effect_name` references.

A green FX report means the files and DAT binding are consistent, not that the
particle rendered correctly in game.

## Limits

- Random-map records are framed and counted but not writable.
- Resource-cost and research-location arrays on techs are edited through
  recipes, not direct flags.
- The effect types listed as hypotheses above stay hypotheses until an engine
  fixture confirms them.
