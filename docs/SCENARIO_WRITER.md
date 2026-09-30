# Scenario Writer Architecture

AoE2Kit reads and writes current AoE2DE `.aoe2scenario` files with its own typed
Go codecs. Scenario files are sectioned, nested, count-heavy artifacts, so there
is no safe small byte poke for the sections that matter (triggers, units, map):
Kit decodes a section completely, edits the model, and re-encodes it.

## Versions

| version | read | write |
| --- | --- | --- |
| DE 1.59 | yes | yes |
| DE 1.58 | yes | yes |
| 1.57 and older | no: fails with `unsupported scenario version` | no |

Opening an older scenario in the DE editor and saving it converts it to 1.59.
The format differences between 1.58 and 1.59 are recorded in
[`SCENARIO_159_NOTES.md`](SCENARIO_159_NOTES.md).

## Model

Kit's scenario model has three layers:

1. **Layout**: the byte-level field order of every section, owned by Kit and
   established from editor saves and the scenario corpus.
2. **Typed codecs with preservation**: each section decodes into typed values;
   bytes whose meaning is not yet known are carried as opaque raw fields, so an
   unmodified file always rebuilds byte-for-byte.
3. **Semantics**: player labels, enums, units, bitfields, and fields the file
   stores more than once (for example diplomacy and trigger counts), with one
   copy treated as authoritative and the others derived from it on write.

The write pipeline is:

```text
read uncompressed file header
inflate scenario body
decode every section with the typed codecs
apply recipe edits to the model
re-derive mirrored fields and counts
encode, deflate, and write the output
reopen the output and verify it
```

`kit scen verify` checks the exact-rebuild property on any supported file, and
Kit's test corpus must rebuild byte-for-byte before a codec change lands.
Unrecognized structures fail closed: Kit reports an error or preserves the
section opaquely instead of guessing.

## Trigger Invariants

The trigger block must satisfy these invariants on every read and write:

```text
number_of_triggers == len(trigger_data)
number_of_triggers == len(trigger_display_order_array)
trigger_display_order_array is a valid permutation of trigger indexes
number_of_effects == len(effect_data)
effect_display_order_array is empty or a valid permutation of effect indexes
number_of_conditions == len(condition_data)
condition_display_order_array is empty or a valid permutation of condition indexes
num_selected == len(selected_object_ids)
```

Forgetting the top-level trigger display-order array is a known corruption
class; the writer maintains it explicitly.

## Verification

Every write path proves:

```text
output reparses to EOF
unedited sections rebuild byte-for-byte
trigger/effect/condition/display-order counts are consistent
added or changed fields read back exactly
decompressed payload round-trips through deflate/inflate
```

Do not compare compressed bytes; deflate output is not stable across encoders.
Structural verification is not game verification: see
[`AOE2KIT_WRITE_SMOKE.md`](AOE2KIT_WRITE_SMOKE.md) for the editor/game smoke
test and for the values the DE editor normalizes on save.

## Commands

```sh
./kit scen triggers in.aoe2scenario
./kit scen units in.aoe2scenario
./kit scen map in.aoe2scenario
./kit scen verify in.aoe2scenario
./kit scen plan in.aoe2scenario --recipe docs/SCEN_RECIPE_EXAMPLE.json
./kit scen patch in.aoe2scenario out.aoe2scenario --recipe docs/SCEN_RECIPE_EXAMPLE.json
./kit scen smoke-recipe --x 145 --y 175
./kit scen smoke in.aoe2scenario AoE2Kit_WriteSmoke.aoe2scenario --x 145 --y 175
```

`plan` is read-only, but it is not just a counter: it validates current
selectors and guarded delete dependencies for `remove_trigger`, `edit_trigger`,
`remove_unit`, and `edit_unit` before reporting the operation list.

## Write Primitives

Current write primitives are append-only helper triggers, unit placement, map
tile edits, player/diplomacy/resource setup, and XS embedding. See
`docs/SCEN_RECIPE_EXAMPLE.json` for a broad recipe and
`docs/SCEN_WRITE_SMOKE_RECIPE.json` for the canonical smoke fixture.

The broad recipe intentionally includes a small scenario-setup block as well as
trigger/unit/map edits:

```json
"scenario": { "player_count": 4 },
"victory": { "conquest_required": 0, "all_custom_conditions_required": 1, "mode": 4 },
"players": [{ "player": 1, "active": true, "human": true }],
"diplomacy": [{ "from": 1, "to": 2, "stance": 3 }],
"resources": [{ "player": 1, "food": 500, "wood": 500, "gold": 250, "stone": 100 }]
```

`scenario.player_count` writes the FileHeader player count advertised to the
engine/lobby. It must be kept in sync with the intended active non-Gaia player
slots. AoE2Kit also syncs the Units-section `number_of_players` field to
`player_count + 1` including Gaia when writing the scenario count. Player-slot
indexes elsewhere remain `0` for Gaia and `1..8` for normal player slots.

`kit scen blank` keeps P0/Gaia active by default, matching the editor-authored
blank baseline and normal scenario authoring expectations. Do not assume
Gaia-inactive authoring is supported until the separate Gaia active-state field
is decoded and write-verified. By default the blank gets one Barracks (unit 12)
per active normal player so DE does not fill empty active slots with starter TC
state; `--dummy-unit` changes the starter unit and `--no-starters` omits them.
`--no-conquest` sets `conquest_required=0` for diagnostics where empty or
intentionally inert slots should not end the game by normal conquest rules.
For controlled replay probes, start from a fresh blank scenario with its default
starters, disable conquest, and use `--closeout-seconds N` when the probe
should end itself through a timer-gated declare-victory trigger. Use the
`scen.timer-declare-victory` recipe for the same closeout shape when patching an
existing scenario instead of creating a blank.

`kit scen patch` refreshes FileHeader `timestamp_of_last_save` to the current
Unix time by default so generated scenario forks do not keep an inherited stale
date in the DE browser. Set `scenario.timestamp_of_last_save` explicitly in a
recipe when a deterministic or intentionally pinned timestamp is needed.

Player, diplomacy, and resource indexes use the scenario's player-slot indexes:
`0` is Gaia, `1..8` are normal player slots. Diplomacy stance values are the raw
scenario/editor values until a tighter portable enum table is added.

Supported effect ops include `display_instructions`, `display_timer`,
`send_chat`, `create_object`, `kill_object`, `remove_object`, `task_object`,
`change_ownership`, `change_object_name`, `change_object_description`,
`change_object_caption`, `change_object_hp`, `teleport_object`,
`change_object_stance`, `set_player_visibility` / `reveal_map`,
`research_technology`, `enable_disable_technology`,
`change_technology_name`, `change_technology_description`,
`change_technology_cost`, `change_technology_icon`,
`change_technology_location`, `change_technology_research_time`,
`modify_attribute`, `modify_resource`, `script_call`, `activate_trigger`,
`deactivate_trigger`, and `declare_victory`.
Supported condition ops include `timer`, `object_selected`, `object_in_area` /
`objects_in_area`, `own_objects`, `own_fewer_objects`,
`accumulate_attribute`, `object_visible`, and `variable_value`.

`edit_trigger` selects an existing trigger by `target_index` or `target_name`.
It can set `enabled`, `looping`, and `set_name`, and can append the same
supported effect/condition ops while updating that trigger's local counts and
display-order arrays. It can also remove existing child rows with
`remove_effects` and `remove_conditions`, or clear the child lists with
`clear_effects` and `clear_conditions`; these operations repair
`number_of_effects`, `number_of_conditions`, and both local display-order arrays.
Use `replace_effects` and `replace_conditions` when the intended operation is
to replace an entire child list with a new declared list; replacement cannot be
mixed with remove/clear/append for the same child list.
`copy_trigger` selects a source trigger by `target_index` or `target_name`,
appends a raw clone, and then applies the same rename, enabled/looping, child
append/remove/clear/replace edits to the clone. Trigger-id references inside the
clone are preserved exactly; rewrite them explicitly after copying if needed.
`remove_trigger` deletes one existing trigger by `target_index` or `target_name`;
`clear_triggers` removes the whole trigger block before adding replacements. Both
repair scenario trigger counts and the trigger display-order array. `remove_trigger`
also protects trigger-control references: if another trigger references the
removed trigger id, the write fails; references above the removed index are
decremented so they keep pointing at the same surviving trigger after the index
shift.

The `variables` recipe block supports `add_variable`, `edit_variable`,
`remove_variable`, and `tombstone_variable`. `add_variable` appends a
`VariableStruct`, chooses the next free id unless `id` is supplied, and updates
`number_of_variables`. `edit_variable` targets an existing variable by
`target_id`/`id`, `target_name`, or `name` and rewrites only `variable_name`.
`remove_variable` physically removes an unreferenced variable record and updates
`number_of_variables`; surviving variable ids are explicit and are not
renumbered. `tombstone_variable` preserves the record but renames it to
`_DeletedVariable<ID>` by default. Both delete forms refuse to run while trigger
effects or conditions still reference that id. Physical variable removal is
structure-verified by write/reopen tests; treat it as not engine-verified until
a DE editor/game smoke confirms the scenario loads.

The `strings` recipe block supports `add_string`, `set_string`, `clear_string`,
and `tombstone_string` for the fixed 32-slot `PlayerDataTwo.strings` table.
`add_string` uses the first empty slot unless `id` is supplied, `set_string`
rewrites a known slot, `clear_string` empties an unreferenced slot so it can be
reused, and `tombstone_string` preserves the id while replacing the text with
`_DeletedString<ID>` by default. `old_text` can be supplied as a guard for
set/clear/tombstone operations, and both delete forms refuse to run while
scenario trigger effects still reference the slot by `string_id`. AoE2Kit does
not compact string ids because that would renumber later slots.

XS support follows the current AoE2DE UGC Guide model plus the engine-verified
inline source pattern; see `docs/XS_AUTHORING.md`. The preferred generated recipe
uses `xs.mode:"inline_runtime"` to clear external XS filename fields and place
full XS source in an enabled trigger-0 `script_call` message titled `XS string`.
`xs.mode:"carrier"` is disabled source escrow for extraction/inspection.
`xs.mode:"attachment"` keeps the older `Map.script_name` + `Files.script_file_*`
path for deliberate external module deployment, and
`xs.mode:"attachment_and_carrier"` writes both. Ordinary trigger `script_call`
effects invoke parameterless XS functions through the effect `message` field.

`add_unit` appends one `UnitStruct` to an existing player unit section and
updates that section's `unit_count`. `edit_unit` and `remove_unit` target
existing units by `reference_id`, by `target_player` + `target_index`, or by a
unique `target_caption`. Caption targeting fails loudly if more than one unit
matches; add `target_player` to disambiguate. `copy_units_in_area` clones every
matched unit inside `target_area_x1/y1/x2/y2`, optionally narrowed by
`target_player` and `target_unit_const`, offsets positions by `offset_x` /
`offset_y` or by placing the source area's top-left at `target_x,target_y`,
assigns fresh reference ids from `reference_id_base` or the next free id, can
move the copies with `set_player`, and can append `caption_suffix`.
Garrison links are preserved only when the referenced container is also copied;
otherwise the copied unit is ungarrisoned to avoid pointing back at the source
set. `move_units_in_area` relocates matched units by `offset_x` and/or
`offset_y`, or by placing the source area's top-left at `target_x,target_y`,
without changing owner or reference ids, and refuses moves that would place any
matched unit outside the map. `edit_units_in_area` applies bulk
field edits to matched units: `unit_const`, `status`, `rotation`,
`initial_animation_frame`, `garrisoned_in_id`, `caption_string_id`,
`caption_string`, and/or `set_player`. Position edits are intentionally kept in
`edit_unit` and `move_units_in_area`. `remove_units_in_area` physically deletes
every matched unit in the same kind of area selector; the whole operation fails
before mutation if any matched unit is still referenced.
`edit_unit` can move an existing unit to another owner
section with `set_player`, and the writer repairs both source and destination
`unit_count` fields. Unit delete operations also protect direct trigger object
references before deletion: selected-object effect lists, condition
`unit_object` / `next_object`, and location object references are checked, and
the write fails with the exact trigger/child/field path if a reference would
dangle.
`cluster_scatter` expands to normal `add_unit` rows. It places `count` copies
near one or more `centers`, using deterministic `seed`, optional `spread`,
`min_distance`, `target_area_*` bounds, and `allowed_terrain_ids` habitat
filters. Exact placement is the default: if the habitat/min-distance constraints
cannot fit all requested units, the recipe fails instead of silently spilling
onto the wrong terrain. Set `exact:false` to allow best-effort partial
placement.

Scatter placement has three orthogonal knobs:
`snap:"grid"|"fractional"|"auto"`, `distribution:"contiguous"|"dispersed"`,
and `jitter`. `grid` snaps to exact tile centers (`floor(x)+0.5`), ignoring
jitter; use it for blocking resource units. `fractional` keeps sub-tile
coordinates and applies caller-owned jitter; use it for zero-collision
decorative units. `auto` is the default. When the recipe root provides
`dat_path`, auto uses DAT `collision_size_x/y`: zero-collision units stay
fractional, anything blocking snaps to grid. Without `dat_path`, auto falls
back to grid rather than risking fractional blocking units. `contiguous` fills
valid tiles nearest the center first for clumpy resource forests; `dispersed`
uses Gaussian spread for organic scatter. Legacy `cluster_shape` is still
accepted: `packed` maps to `snap:auto, distribution:contiguous, jitter:0`, and
`random` maps to `snap:auto, distribution:dispersed, jitter:0`. This
intentionally removes the old packed-mode baked-in +/-0.09 jitter and may
grid-snap old random recipes for blocking units when `dat_path` is supplied.
Set `rotation_choices` to a list of numeric rotation/variant values when the
unit uses rotation as an artwork-addressing slot; scatter samples from that
pool instead of using an arbitrary facing angle, which is useful for dense
forests and other repeated visual variants.

`set_terrain_rect` edits existing terrain tiles in an inclusive rectangle. It
can set `terrain_id`, `elevation`, and/or `layer`; it does not resize maps.
`set_terrain_circle` uses `x1,y1` as the center and requires `radius`.
`set_terrain_line` uses `x1,y1` to `x2,y2` with Bresenham tile selection.
`set_terrain_border` paints only the perimeter of an inclusive rectangle and
accepts `thickness` with a default of 1 tile.
`copy_terrain_area` copies an inclusive source rectangle to `target_x,target_y`,
preserving terrain id, elevation, layer, and reserved tile bytes.
`terrain_grid` bulk-writes a rectangular terrain grid in one pass. It accepts
`x1,y1,width,height` (defaulting to the whole remaining map from `x1,y1`),
row-major `terrain_id` and `layer` arrays of length `width*height`, and an
optional row-major `elevation` array. `layer:-1` means no top layer. For large
maps, prefer `grid_file:"path/to/grid.json"`; the grid file contains
`{"width":N,"height":N,"terrain_id":[...],"layer":[...],"elevation":[...]}`.
Relative `grid_file` paths are resolved next to the recipe file when loaded
through `kit scen plan` or `kit scen patch`. This is the intended bridge for
generators that already know every tile, such as real-map tracers.
`layered_crossfade` is the explicit top/bottom terrain-layer primitive. It
sets `terrain_id` as the bottom terrain and `terrain_id_2` as the top terrain
stored in the tile `layer` field. Use paired bands with the values swapped for
terrain seams, for example water-over-sand followed by sand-over-water. This
encoding is structure-verified from DE editor output. Engine passability tests
show the bottom `terrain_id` governs movement: a water-bottom/grass-top tile
remains boat-passable and not scout-passable, while a grass-bottom/water-top
tile remains scout-passable and not boat-passable. The top layer is therefore
the visual blend layer, not a passability override. Dock-buildability and the
full beach-bottom amphibious rule remain narrower engine checks.
The optional top-level `masks` recipe array defines named terrain masks before
map operations run. Supported mask ops are `rect`, `circle`, `blob`,
`from_terrain_class`, `union`, `intersect`, `subtract`, `dilate`, and `erode`.
Circle and blob masks clip safely at map edges, so landscape shapes can extend
past the scenario boundary without making the recipe invalid. Map operations can
then set `mask` to constrain their work to the named mask. `set_terrain_mask`
paints an entire mask.
`noise_fill` mottles an inclusive rectangle between `terrain_id` and
`terrain_id_2` using deterministic value noise. Optional fields are `seed`,
`scale` (default 8.0), and `threshold` (default 0.5). With `mask`, the fill is
constrained to the named mask. The same recipe and seed produce the same tiles
every time.
`erode` runs a cellular majority pass over an inclusive rectangle. It is useful
as a smoothing post-pass after geometric terrain strokes or noise fills.
Optional `iterations` defaults to 1.
`semantic_erode` is a class-aware erode pass. Provide `terrain_ids` to limit the
pass to that terrain class so unrelated tiles, such as deep ocean or land
outside a water class, are preserved.
All map operations validate source and destination bounds. Paint operations can
set `terrain_id`, `elevation`, and/or `layer`.

`settings` is a read-only audit view for player slots, AI names/types,
resources, diplomacy, allied-victory flags, and global victory fields.

The `players` recipe array edits fixed player slots. Supported fields are
`active`, `human`, `tribe_name`, `civilization`, `lock_civilization`,
`lock_personality`, `ai_name`, and `ai_type`.

The `diplomacy_options` recipe block edits section-wide diplomacy settings:
`lock_teams`, `allow_players_choose_teams`, `random_start_points`,
`max_number_of_teams`, and per-player `allied_victory` flags. Pairwise stance
edits still use the existing `diplomacy` array with `from`, `to`, and `stance`.

`delete-plan` is read-only and emits the safest available recipe for a target.
For units, triggers, and unreferenced variables it can propose physical removal
when known direct references are clear. For string ids it proposes fixed-slot
clearing rather than compaction, because the table index remains the
engine-facing contract. Trigger-local child rows are supported with `effect <trigger>:<row>`
and `condition <trigger>:<row>` targets; these emit `edit_trigger` recipes with
`remove_effects` or `remove_conditions`. Child rows are index-addressed rather
than stable-id-addressed, so inspect the trigger again before applying an old
plan. Whole-trigger deletes that include effect type 55 `script_call` now add a
semantic warning to the plan: the row may be structurally removable, but AoE2Kit
cannot prove external XS/module expectations still make sense after the call is
removed.

`smoke-recipe` prints a project-neutral editor-smoke recipe. `smoke` applies it
directly. The smoke creates one visible terrain marker, one visible unit, and
three disabled trigger primitive groups; it remains
`structure_verified_not_engine_verified` until the output opens in the DE
editor/game and the visible markers are confirmed.

When appending triggers, `patch` updates `Triggers:number_of_triggers`,
`Triggers:trigger_display_order_array`, `Options:number_of_triggers`, and
`FileHeader:trigger_count`, then reopens the output and verifies rebuild and
trigger invariants.
