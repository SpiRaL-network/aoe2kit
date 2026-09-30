# DAT Engine Architecture

AoE2Kit will not port the `genieutils` eager object graph. The Go `.dat` engine
uses a patch-engine model:

```text
inflate .dat
scan the sequential payload into byte spans
index known sections and fields
plan patches
patch fixed-width fields in place, or splice owned records
recompress
verify by re-indexing the output payload
```

## Format Facts

AoE2Kit targets current AoE2DE `.dat` files. It does not spend complexity on
old Genie layouts unless a real current workflow brings one back into scope.

The Genie `.dat` is a raw-DEFLATE compressed, sequential, position-independent
byte stream. Records are found by parsing previous records, not by absolute
offsets. There are no internal byte-address pointers that need rebasing after a
record splice.

The structural invariants are counts:

```text
graphics_count
per-record sub-array counts
per-civ unit counts
terrain/effect/tech counts
```

The hard failure mode is a bad parse of a variable-length field, especially a
debug string, which shifts every downstream span. The guard is span coverage.

## Span Coverage Gate

Before any patch, the indexer must prove:

```text
spans are ordered
spans have no gaps
spans have no overlaps
concatenating spans reproduces the inflated input exactly
parse reaches EOF exactly
```

No patch may run without this gate.

## Debug String

DE debug strings must be modeled exactly:

```text
uint16 marker/prefix 0x0A60
uint16 byte length
UTF-8 bytes
```

Misreading this field silently shifts downstream spans.

## Patch Modes

### Numeric In-Place Patch

Use for fixed-width fields only:

```text
int8
int16
int32
float32
```

Examples:

```text
graphic.slp
graphic.frame_count
graphic.layer/player_color/sound_id/timing/scalar flags
unit.dying_graphic
unit.blood_unit_id
unit.standing_graphic
type50.projectile_unit_id
type50.attack rows
type50.armour rows
creatable.button_hotkey_action
creatable.cost rows
unit.action task rows
```

### Owned Record Splice

Use for variable-length fields and owned records.

`Graphic.particle_effect_name` and `Graphic.file_name` are debug strings, and
the common particle case is empty string to descriptor name. Do not implement an
equal-length string hack; record-splice owns all string writes.

Effects, Techs, and Sounds are now owned codec sections: AoE2Kit can rebuild
their count-prefixed tables from typed records, append records, and reparse the
output. Terrain debug-string edits also use owned record splices inside the
terrain table.

## Verification After Write

Every write must:

```text
inflate output
re-index to EOF
assert counts unchanged
assert intended patch reads back
assert neighboring records still parse and key fields match
assert downstream canaries still parse
assert length delta is exact
```

Do not compare compressed bytes. Deflate output is not stable across encoders.
Use the decompressed payload as the verification target.

## Milestones

1. `pkg/datfile/raw`: raw DEFLATE inflate/recompress, version read, cursor. DONE.
2. `pkg/datfile/graphics`: index graphics, inspect scalar fields, deltas, and
   DE angle-sound rows; patch selected verified fields. DONE.
3. `kit dat graphic`: graphics inspection and patch surface. DONE for `patch-graphic`.
4. Graphic record splice: `particle_effect_name` and `file_name`. DONE.
5. `pkg/datfile/units`: index civ/unit common fields. DONE for current `VER 8.8+` records.
6. `kit dat unit`: read-only and fixed-field unit patches. DONE for common current-DE unit fields.
7. `kit dat patch --recipe`: plan/apply/verify. DONE.
8. Effects, tech/research, game metrics, and tech-tree read-only indexing. DONE.
9. Codec-backed Effects/Techs editing. DONE for create/patch/delete Effects,
   create/patch Techs, tail-delete Techs, and tech-tree connection field/list
   patches with readback. Non-tail Tech deletion remains blocked until full
   reference rewrites are implemented.
10. Civ, terrain, sound, and player-colour codec coverage. DONE for Civ
   header/resources, terrain restriction passability, terrain debug strings,
   Sound patch/create/semantic mute, and PlayerColour fixed-width rows.
11. Graphic child-list rebuilds. DONE for complete `set_deltas` and
   `set_angle_sounds` recipe replacement by rebuilding exactly one graphic
   record. Delta lists can grow/shrink. Angle-sound lists can be enabled or
   disabled; non-empty replacements must match the graphic's existing
   `angle_count`.
12. Type50/Creatable/action fixed-field unit patches. DONE for projectile,
   range/blast, attack graphic, attack/armour rows, train locations, button
   fields, resource costs, unit attributes, damage graphics, and action task
   rows. DONE for count-changing decoded list replacement on damage graphics,
   Type50 attacks/armours, Creatable train locations, and action drop-sites by
   rebuilding exactly one unit record and reparsing the DAT.
13. Unit-header task rows. DONE for fixed-field patches and full `set_tasks`
   list replacement through codec recipes using the decoded top-level
   UnitHeader task table. Task rows expose named leading fields plus raw tail
   bytes for the still-unnamed current-DE suffix. Count-changing unit action and
   unit-header task list edits are DONE through `set_tasks`, guarded by an exact
   raw-tail-length requirement and DAT reparse/readback.
14. Full typed unit section codec and broad DELETE semantics remain open. The
   current delete rule is conservative: tombstone stable-ID records where
   verified, physically delete only sections with safe reference handling, and
   show DAT reference ledgers before mutating stable ID tables. Tech delete
   plans now report required-tech and tech-tree references; effect-command rows
   carry the upstream genieutils field overlay (`target_unit`, `unit_class_id`,
   `attribute_id`, `amount`) plus Advanced Genie Editor command-family names. Known
   command meanings such as `disable_tech`, `spawn_unit`, scoped
   attribute/resource/tech commands, local-building command-family types
   `200`/`201`/`202`/`204`, and the observed DE rename-unit special
   case of `gaia_set_attribute` on `name_id` become
   `typed_effect_command_reference` delete-plan rows. Unit plans also
   separate remaining command-shape unit candidates
   (`candidate_effect_command_operand`) from arbitrary numeric coincidences
   (`possible_effect_command_operand`); both remain warning-grade and are not
   rewrite-supported until the command type is decoded. Local-building type
   `202` is interpreted as multiply attribute; type `204` is interpreted as an
   advanced additive local-building attribute form used by current-DE
   emplacement-style rows, with packed attack/armor deltas decoded when
   present. `kit dat effect-explain` and `kit dat tech-explain` show those rows
   in authoring language, and `kit dat semantics-pack --feature
   local-building-effects` emits a focused fixture/readback pack. Proof tier is
   deliberately split: `200`/`201` are source-verified from the official
   command wording, while `202`/`204` remain strong hypotheses until a
   dedicated local-building engine fixture confirms their runtime behavior.
   Reference summaries keep
   `possible_operand` as the broad warning class and expose `candidate_operand`
   as its command-shape subset. The tech-tree section now has byte-identical
   encode coverage and
   can patch selected fixed fields plus counted child lists on
   building/unit/research connection rows. Semantic tech-tree rewrite helpers can
   remove/replace tech or unit IDs across the decoded tech-tree ledgers, and
   top-level reference rewrites now also cover `tech.required_techs` plus typed
   effect-command references. Non-tail tech deletion uses a guarded tail-swap
   strategy: the delete target must be unreferenced, and the current tail tech's
   references must all be covered by verified rewrite surfaces before the row is
   moved into the hole. Referenced tech delete plans emit `disconnect_techs`
   cleanup hints where possible, and referenced-but-unshared abilities can be
   semantically disabled by clearing their paired effect commands.

The current indexer parses from the file header through graphics, including
`GraphicDelta` and current-DE `GraphicAngleSound` rows, terrains, effects,
unit-header task rows, current-DE civ/unit records, tech/research records, game
metrics, and the tech-tree connection section. Span coverage is full-payload
coverage: spans are ordered, gapless, and end exactly at EOF. Later milestones
should type more fields only when they become patch targets, and every new
owned section must keep the decode/encode byte-identity gate before writes are
trusted.
