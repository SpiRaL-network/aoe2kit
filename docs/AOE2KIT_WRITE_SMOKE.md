# AoE2Kit Write Smoke

Purpose: flip AoE2Kit scenario writes from structurally verified to editor/game
verified with one cheap visual test.

This recipe expects an input scenario with map coordinates through at least
`(147,177)`.

## Build The Smoke Output

From the repository root, either apply the built-in smoke directly:

```sh
./kit scen smoke \
  path/to/input.aoe2scenario \
  AoE2Kit_WriteSmoke.aoe2scenario \
  --x 145 --y 175
```

Or generate/apply the canonical recipe explicitly:

```sh
./kit scen smoke-recipe --x 145 --y 175 > docs/SCEN_WRITE_SMOKE_RECIPE.json

./kit scen patch \
  path/to/input.aoe2scenario \
  AoE2Kit_WriteSmoke.aoe2scenario \
  --recipe docs/SCEN_WRITE_SMOKE_RECIPE.json

./kit scen verify AoE2Kit_WriteSmoke.aoe2scenario
```

Open `AoE2Kit_WriteSmoke.aoe2scenario` in the AoE2DE editor.

## Expected Visuals

- Terrain write: a 3x3 patch at tiles `(145,175)` through `(147,177)` is terrain
  id `47`, elevation `0`, layer `-1`.
- Unit write: player 1 has one added unit `83` at `(146.5,176.5)`.
- Trigger write: trigger list contains three disabled `AOE2KIT WRITE SMOKE`
  triggers covering display/chat/timer, create/task/kill/remove, and
  activate/deactivate trigger-control primitives.

## Status Rule

Until this output opens in the editor and game and the three expected visuals are
confirmed, these write primitives count as structurally verified (Kit re-reads and
rebuilds them exactly) but not game-verified. After confirmation, record the exact
AoE2DE build and date alongside the result.

## Editor Normalization

The DE editor performs normalization while saving: it fills convenience values
that were not explicitly authored and may rewrite derived fields after loading
the scenario. Kit treats these as a named authoring concern rather than
scattered writer quirks.

Measured editor-normalized values currently include:

- generated trigger names (`Trigger N`);
- `replace_object.target_player = 2` when the editor default is untouched;
- unused effect floats encoded as the editor's NaN value;
- map defaults such as `water_definition = "Default"`,
  `map_color_mood = "Default"`, and `ai_map_type = 157`;
- a per-scenario list of dataset or DLC identifiers in the file header, set when a
  scenario is created and preserved on later saves (its exact meaning is not yet
  confirmed).

The editor also appears to derive `initial_camera_x/y` from a player's first
unit on load (`52,54` matched a unit at `52.5,54.5`), but that remains an
untested hypothesis and is not a writer rule.

When a caller needs editor parity, the writer should apply measured
normalization and be checked against an editor-saved fixture byte-for-byte.
When a caller needs a minimal patch, it should emit only requested recipe
fields and report omitted editor defaults. Every normalization entry needs a
calibration fixture; future editor saves extend this list rather than silently
introducing special cases.
