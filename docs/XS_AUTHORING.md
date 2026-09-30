# XS Authoring Model

AoE2Kit treats the AoE2DE UGC Guide as the current public source of truth for
XS authoring. Source repo:

`https://github.com/Divy1211/AoE2DE_UGC_Guide`

The older Forgotten Empires-era XS notes are superseded for Kit decisions. Keep
them only as historical context when encountered.

## Published Syntax Hazards

Kit's XS lint incorporates the statically decidable hazards from the AoE2DE UGC
Guide's XS bug catalogue (`docs/general/xs/bugs/Language Syntax.md` in the
guide repository). It rejects
non-literal `vector(...)` coordinates, int/bool comparisons, unary negatives,
single-quoted strings, unsupported `long`/`double`/`char` types, unparenthesized
return expressions, loop self-assignment, integer literals from 1,000,000,000,
and calls with more than twelve arguments. Findings cite the catalogue section.

The catalogue also documents hazards that are not safely decidable from source:
operation result typing, float modulo, static variables in recursive functions,
`infiniteLoopLimit`/`infiniteRecursionLimit` exact behavior, malformed effect
IDs, duplicate research, and runtime string formatting. Treat those as runtime
constraints and leave headroom; do not infer that a lint-clean script is
engine-clean. The repository's `Fixed.md` supersedes the corresponding older
crash entries. Where the Kit's own engine probe disagrees with the catalogue,
the probe remains the active ledger fact and the distinction is documented.

## Runtime Placement

For custom scenarios, the UGC guide's programmer reference says the scenario's
Map tab names an XS file without the `.xs` suffix, and trigger `Script Call`
effects call parameterless functions from that XS script.

AoE2Kit supports three XS placement modes:

- preferred for self-contained generated scenarios: enabled inline runtime
  carrier trigger;
- source-escrow carrier mode: parser-style embedded carrier trigger;
- legacy/explicit external-file mode: scenario attachment fields plus a shipped
  `.xs` file.

The inline runtime mode mirrors the engine-verified SpiRaL/CB Front Towers
self-contained shape: trigger 0, effect 0 is an enabled `script_call` effect
titled `XS string`, and the effect `message` is the full inline XS source.
`Map.script_name`, `Files.script_file_path`, and `Files.script_file_content` are
empty, so DE has no external filename to open. This is the mode to use when a
generated scenario must run without any sibling `.xs` file.
The empty fields must be encoded as true zero-length strings (`00 00` for
`str16`, `00 00 00 00` for `str32`), not as NUL-only strings; DE rejected the
NUL-only form during `readScenario` with a TEMP path-spec error even though the
parsed view looked identical.

Use this recipe shape for inline runtime mode:

```json
{
  "xs": {
    "mode": "inline_runtime",
    "carrier_title": "XS string",
    "content_file": "MyScript.xs"
  }
}
```

The source-escrow `carrier` mode stores the full XS source in the `message` field
of a disabled trigger `script_call` effect. That source travels inside the
`.aoe2scenario` itself for inspection/extraction, but this mode is not the
self-contained runtime path. `kit scen xs` reports both runtime and escrow source
effects as `embedded_carriers`, analyzes their functions, and excludes the
carrier payload from ordinary runtime `script_call` function invocations.

Use this recipe shape for escrow carrier mode:

```json
{
  "xs": {
    "mode": "carrier",
    "carrier_title": "XS string",
    "carrier_trigger_index": 0,
    "content_file": "MyScript.xs"
  }
}
```

By default `mode:"carrier"` clears the older attachment fields and replaces the
first existing carrier, avoiding stale duplicate XS blobs. Set
`carrier_trigger_index` when converting an existing placeholder `script_call`
trigger into the carrier slot, which keeps the trigger list stable.
`mode:"inline_runtime"` clears the older attachment fields and inserts an enabled
trigger-0 carrier when trigger 0 is not already a carrier, shifting trigger-id
references forward. Use `"mode":"attachment"` only when you deliberately want the
old external-file deployment path, or `"mode":"attachment_and_carrier"` when you
are comparing both paths.

The same carrier workflow is available without hand-writing a JSON recipe:

```sh
./kit scen xs attach in.aoe2scenario out.aoe2scenario --xs MyScript.xs --name MyScript.xs --text
./kit scen xs deploy out.aoe2scenario path/to/mod-or-profile-root --force --check --text
./kit scen xs embed in.aoe2scenario out.aoe2scenario --xs MyScript.xs --replace-trigger 0 --text
./kit scen xs compare out.aoe2scenario MyScript.xs --text
./kit scen xs extract out.aoe2scenario --out /tmp/MyScript.xs --force --text
```

`attach` writes the executable scenario XS fields for an intentional external
module tree. `deploy` copies the scenario-carried source to
`resources/_common/xs/` under the supplied deploy tree, then `--check` verifies
the scenario fields, on-disk entry file, includes, and cross-file `extern`
requirements. This is the normal path only when external modules should actually
run in DE.

`embed` writes the carrier and clears stale attachment fields by default.
`compare` checks scenario-carried XS against a source file and exits non-zero
when normalized content differs. `extract` pulls the script payload back out of
the scenario; it does not include the carrier title wrapper line. If executable
attachment XS exists, `compare` and `extract` use it by default; pass `--carrier`
or `--trigger` to target parser-style carrier escrow instead.

Attachment mode sets:

- `Map.script_name`
- `Files.script_file_path`
- `Files.script_file_content`

The v3 telemetry probe proved an important distribution caveat: embedded
`script_file_content` is portable source and an inspection aid, but DE still
resolved the runnable script path against `resources/_common/xs/` in that test.
The first engine run failed with XS Error 0354 until the matching `.xs` file was
present on disk. The v8 assertion fixture refined this: DE opened the literal
value stored in `Map.script_name`. A scenario where `Map.script_name` was bare
but `Files.script_file_path` included `.xs` failed by trying to open the bare
name. AoE2Kit now writes both fields consistently with the `.xs` extension.

So the safe authoring rule is:

- use `inline_runtime` mode for single-file generated XS whenever possible;
- use disabled `carrier` mode only as source escrow / parser-compatible
  inspection aid;
- use attachment mode only when a real external module tree is intentional;
- if using attachment mode, ship the same `.xs` file under the runnable bundle's
  `resources/_common/xs/`;
- set both scenario XS name/path fields to the same extension-bearing filename;
- use ordinary trigger `script_call` effects to invoke named parameterless XS
  functions.

Front Towers provides live-author evidence for the self-contained inline source
pattern: its scenario attachment fields are empty, while a trigger carries the
full XS source in a `script_call` message titled `XS string`. Kit's inline
runtime writer is structure-verified by round-trip tests and engine-verified by
Kit-authored fixtures: the unattended `kit xs harness` scenarios use
`inline_runtime` and executed on DE `101.103.54800.0`.

## Multi-File XS

The UGC guide documents ordinary XS imports with:

```xs
include "absolute/or/relative/path/to/file.xs";
```

This is plain XS syntax, not RMS `#includeXS`. `#includeXS` belongs to random
map scripts; it is not the syntax used inside `.xs` files.

Working shipped mods prove multi-file XS is used in practice:

- `RPG_scripts.xs` includes `RNG_Function.xs`, `AI_Scripts.xs`, and
  `smart_mob_control.xs` at lines 1001-1003, after 1000 extern declarations.
- `smart_mob_control.xs` includes generated data files at the top and
  `mob_targeting.xs` at EOF.
- `zone_constants.xs` and `zone_arrays.xs` are generated data modules.

The practical Kit model is therefore a module tree:

- a named entry script selected by the scenario;
- sibling helper modules included with `include "file.xs";`;
- generated data modules kept separate from handwritten runtime logic.

Resolution against `resources/_common/xs/` is proven for the main script path by
the v3 probe. GoKu's layout strongly supports the same folder as the include
root for mod-shipped sibling XS files. Include ordering, deduplication, and
cycle behavior are not yet characterized; avoid cycles and put constants/data
before code that needs them.

## Trigger Bridge

There are two relevant bridge directions.

Scenario trigger to XS:

- set scenario XS name/path/content with the `xs` recipe block;
- add trigger effect `script_call` with `message`, for example `BootProof();`;
- call parameterless XS functions from trigger effects.

XS to scenario variables:

- GoKu declares `extern int xsVariable0 = -1;` through
  `extern int xsVariable999 = -1;` in `RPG_scripts.xs`.
- This is working-corpus evidence that scenario trigger variables can be exposed
  as XS globals under the `xsVariableN` naming convention.
- The v8 fixture proved cross-file symbols need `extern`: plain `const int`
  constants in one included module were invisible to a sibling included module.
  Shared generated constants should therefore use `extern const int NAME = N;`.

AoE2Kit should prefer generated constants/helpers around `xsVariableN` rather
than asking authors or AIs to hand-track raw variable numbers in large systems.

## File I/O

The UGC function reference documents `.xsdat` File I/O:

- `xsCreateFile`
- `xsOpenFile`
- `xsWriteString`, `xsWriteInt`, `xsWriteFloat`, `xsWriteVector`
- `xsReadString`, `xsReadInt`, `xsReadFloat`, `xsReadVector`
- file-position helpers and size helpers

Important behavioral claims from UGC:

- `xsCreateFile` creates/appends to an `.xsdat` file named after the RMS or
  scenario.
- In multiplayer, create/write is duplicated to each player.
- `xsOpenFile` requires the file to exist for all multiplayer players with the
  same data to avoid out-of-sync risk.
- String data is length-prefixed with a 32-bit length.

GoKu does not use File I/O. The v8 fixture proved scenario-context file
creation: DE created the expected `.xsdat` sidecar under the profile folder.
The first run produced a 0-byte file, strongly suggesting writes are buffered
until `xsCloseFile`. The v8.1 fixture proved that close-before-victory fires,
but also proved that reopening with `xsCreateFile(false)` truncates the
sidecar. The v8.2 fixture therefore opens once, writes all rows, closes once,
then uses `xsCreateFile(true)` for a separate append probe. Its first engine
run produced a nonzero 492-byte sidecar, with full readback still pending.

DE holds an exclusive lock on a live `.xsdat` while the game is loaded. Pull or
inspect sidecars only after the match has unloaded or DE has exited.

The v9 persistence probe engine-verified cross-scenario reads: a reader
scenario opened a writer scenario's sidecar with
`xsOpenFile("RTV9A_Persistence_Writer")`. Do not include the `.xsdat`
extension; the same name with the extension failed because DE appends it
internally. This makes `.xsdat` a viable substrate for same-profile,
multi-scenario campaign persistence.

Future carrier probes should hold each result long enough to be sampled by the
replay checksum stream. v9 used 10-second spacing and one naming attempt was
overwritten between approximately 16-second checksum samples.

## Rule Model

UGC documents top-level `rule` blocks with active/inactive state,
`minInterval`/`maxInterval`, optional `highFrequency`, group, priority, and a
body. GoKu uses inactive one-second rules for per-player mob control.

For continuous animation, use one `active highFrequency` rule as the driver.
highFrequency runs once per engine turn. The engine advances 60 turns per real
second, so the number of calls per game second depends on lobby speed (60 at
speed 1.0, 30 at speed 2.0); tick-counted animation therefore changes with
game speed. Keep pacing deterministic for multiplayer: count turns, and if a
speed-invariant look is needed, measure turns in the first game second (lobby
speed is shared, so this is synced) and scale by it. `xsGetWorldTime` is a
single-player diagnostic clock only; using it for live multiplayer pacing can
desync the simulation. Keep game time and animation ticks distinct. Use
`minInterval` for slower bookkeeping.
Throttle chat and display output to transitions, never every animation tick.
Use XS rules for recurring behavior; looping caption triggers are the narrow
exception for the digit-display bridge below.

### Motion, lifetime, and lookup

The [September engine ledger](XS_ENGINE_VERIFIED_2026-09.md) records the tested
specimens and limits behind these patterns:

| Object/action | Observed result | Authoring approach |
| --- | --- | --- |
| `xsCreateUnit` with float X/Y | Sub-tile placement; failure returns -1 | Retain successful returned IDs; check failure |
| `xsSetUnitPosition(id, vector, false)` | Moves the tested eyecandy, including plants | Move a retained ID each animation tick |
| Flame type 20 + `xsSetUnitHitpoints(id, 0.0)` | Dies with fade | Keep bounded lifetime arrays and reuse slots |
| Campfire 304 | Pops instead of fading | Choose its visual deliberately |
| `xsRemoveUnit` on tested eyecandy | Did not remove it | Do not assume a successful cleanup |
| Static plant type 10/class 14 | Tested XS deletion calls left it present | Pool and park; trigger `remove_object` works when explicit removal is needed |
| Player-owned plant creation | Returned -1 in the probes | Create tested plants as Gaia |
| Many pooled objects on one parking tile | Later creation failed | Give each parked unit its own grid position |

Use a bounded pool of IDs and move dormant objects to distinct unused positions
inside the map. "Park off-map" is a visual intent, not permission to assume
out-of-bounds coordinates work. Z rendered as a diagonal displacement in the
column probe; do not equate a Z stack with a vertical screen column.

`xsGetPlayerUnitIds(player, type)` returns an array ID, not a count and not an
output parameter. Read its size with `xsArrayGetSize`, then its entries with
`xsArrayGetInt`. For class queries use `900 + class_id` (906 infantry, 900
archers). A bare class number is interpreted as a unit type, which caused the
"only archers died" probe failure. Kit cannot infer whether a literal was meant
as a class and therefore does not reject bare unit IDs. Gaia eyecandy was absent
from lookup results in these probes: retain creation IDs yourself.

### Invisible captions and identity

`xsEffectAmount(op, unitType, attribute, value, player)` applies the observed
attribute modification. With op 0 (set), attribute 71 sets the standing graphic;
value 0 made the tested carrier invisible while its caption remained visible.
Attribute 0 sets hitpoints. Blanking the standing graphic does not blank an
attack animation: use non-fighting carriers and protect their ownership.

For digits, declare each scenario variable and set it from XS. A looping caption
trigger can match `variable_value` and write `change_object_caption` to the
carrier's selected reference ID. Keep a site's permanent label on a separate
carrier when its king respawns: `xsCreateUnit` does not inherit the placed king's
per-instance caption. Use stable, unique reference IDs across **all** players.
Kit reserves explicit IDs before automatic allocation and `kit scen lint`
detects duplicates in imported scenarios. In one multi-site test scenario,
every caption site but the first failed because its reference IDs also named
enemy soldiers; that was an ID collision, not a trigger-variable index limit.

### Source preflight

`kit scen xs SCENARIO` checks available embedded/deployed source and reports
file/line, severity, and a suggested fix. It is a static screen, not an engine
compile or a proof of runtime behavior. Its September-ledger checks cover:

- Error: non-ASCII bytes inside strings (Unicode comments are tolerated).
- Error: `%` inside `if`/`while` conditions. Hoist to `int m = e % 2;` first.
- Error: Boolean chains with more than two terms at the same parenthesis level.
  Group pairwise, for example `if ((a && b) && (c && d))`.
- Warning: statically recognizable int-first mixed arithmetic. Convert the int
  to float or put the float first for commutative operations; do not swap the
  operands of subtraction or division.
- Warning: discarded unit-array results or an assigned unit array with no visible
  `xsArrayGetSize` use in the function/rule. A helper may consume it; review that
  warning manually.

Read the first engine error before its cascade. Carrier line numbers were
approximately four lines offset in the observed builds, not a universal offset.

## Known Hazards

UGC's bug pages and Kit probes give these practical authoring constraints:

- XS files do not reliably transfer through multiplayer lobbies; bundle the
  files in a mod or prove a parser-embedded workflow before relying on it.
- `xsResearchTechnology` twice for the same tech/player can crash.
- `%` formatting in `xsChatData` is dangerous; string concatenation is safer.
- Huge formatted ints in `xsChatData` can crash.
- `xsChatData` is not replay telemetry in the v3 probe: none of its marker
  strings serialized, while resource writes did serialize via sync word 1.
- Throttle display/chat diagnostics even when an animation needs `highFrequency`.

## Module Bundles

Real XS authoring means a runnable module bundle plus trigger calls, not a
single attached blob.

Current Kit support:

- scenario `xs` recipe fields attach one entry script by name/content;
- `script_call` trigger effects can call parameterless functions from that
  script via the effect `message` field;
- scenario writer emits consistent extension-bearing XS names in both script
  name/path fields, matching the v8 engine-run finding;
- `kit xs bridge` emits `extern int xsVariableN` and `extern const int` named
  constants for cross-file visibility;
- generated multi-file XS can be shipped in `resources/_common/xs/` by the
  surrounding mod/package workflow.

Not yet provided:

- a bundle/package helper that copies the entry script and all included sibling
  `.xs` files into `resources/_common/xs/`;
- generated `xsVariableN` naming maps from scenario variables.

File I/O is engine-verified through the sidecar fixtures; see `SIDECAR.md`.

The source analysis already inventories includes, reports missing targets and
cycles, and screens the September tokenizer hazards described above. Local
file availability is separate from successful distribution to another player.

## Trigonometric Builtins

The current UGC guide prelude declares the scalar builtins `sin`, `cos`, `tan`,
`asin`, `acos`, `atan`, `atan2`, `radians`, `degrees`, `pow`, `sqrt`, and
`dist`. Kit's generated builtin inventory is derived from guide commit
`1f0f7e8`, so these names are available to authoring tools and should not be
replaced with lookup tables by default.

`sin`, `cos`, and `radians` executed in an engine probe on DE
`101.103.54800.0`. The others are guide-sourced, expected-compile evidence
only: `xs-check` validates spelling and structure but cannot prove that a game
build executes a call correctly. Promote each to `engine_verified` only after an
in-game probe on the target build, and keep the build/version with the result.
See `XS_TRIG_CAPABILITY_AUDIT_2026-09.md`.

The practical authoring pattern is to compute scalar intermediates, then pass
them through `xsVectorSet`. For example, compute `angle = radians(degrees)`,
then `x = cos(angle)` and `y = sin(angle)`. This avoids the vector-constructor
and expression forms that have already produced false confidence in generated
XS. Existing lookup tables remain valid compatibility fallbacks, but they are
no longer the only viable implementation for circles, rotations, arcs, or
distance-based placement.
