# XS Engine-Verified Facts, 2026-09

Engine-tagged entries below were observed in-game on DE (a Windows test box,
replay chat + eyes). The reference-ID finding is separately structure-verified.
Status tags: [VERIFIED] observed live; [HYPOTHESIS] pending a probe.

**Build scope:** all entries below were observed against AoE2DE executable
`101.103.48987.0` and `empires2_x2_p1.dat` SHA256
`ce3530df36cf0b333a9751cb0ff94460fe904f811feecec8ae9794701622b4cf`, captured
2026-09-22 before the pending DLC patch. See
[`ENGINE_BUILD_SCOPE.md`](ENGINE_BUILD_SCOPE.md). New entries must state their
own executable version and DAT hash rather than inheriting this scope silently.

## Motion and timing
- [VERIFIED, 2026-09-23, ticked5 at Slow 1.0 and Fast 2.0] `rule x active highFrequency {}` runs once per engine turn. The engine advances 60 turns per real second at Slow 1.0 and 30 turns per game second at Fast 2.0; `xsGetWorldTime` advanced ~1000 ms per game second in both. Therefore the earlier trigger-removal v2 observation of 30 was valid for Fast speed, not a universal constant; the guide's 60 physical-turns-per-second rule is consistent. Tick-counted animation or harness pacing is intentionally speed-dependent. Do not use `xsGetWorldTime` as a multiplayer pacing clock: the engine operator reports it can cause desync; reserve it for single-player diagnostics where that tradeoff is acceptable.
- [VERIFIED 2026-09-19, trigger-removal probe v2] xsGetGameTime() and xsGetTime() = whole game seconds (identical); xsGetWorldTime() = game milliseconds.
- [VERIFIED] `xsCreateUnit(type, player, vector, ...)` accepts FLOAT positions: sub-tile placement from XS with no editor grid. Returns -1 on failure.
- [VERIFIED] `xsSetUnitPosition(id, vector, false)` moves any unit including eyecandy; a move per tick renders as travel/streak. Works where deletion does not.
- [VERIFIED] Z in the vector is rendered, but displaced diagonally on screen (a Z stack leans right of its X/Y), not straight up. Use Z for lean effects, not vertical columns.
- [VERIFIED 2026-09-19, glyph-grid strip probe, visual at drawing scale] ISO TEXT LAYOUT: to draw glyph cells that are SQUARE on screen, one cell right = map step (+d, +d) and one cell up = (-2d, +2d). The vertical step is exactly TWICE the horizontal step in map units (aspect 2.000). Confirmed against nine patches spanning 1.400 to 2.236 drawn at the real font scale (FLAME2, pitch 0.175); 1.732 was close, 2.000 won. Agrees with a physical ruler measurement of the rendered minimap diamond. An earlier coarse strip (FLAME1 at 0.5 pitch) read low because the flames overlapped into a slab and squareness could not be judged; judge aspect at the pitch you will actually draw at.
- [FIELD OBSERVATION 2026-09-19, first fire-glyph clock] Real glyphs were inverted. The requested correction is a 180-degree turn: cell-right (-d,-d), cell-down (-2d,+2d), still aspect2. A square cannot establish orientation. The revised Go Mono Bold clock defaults to this basis; correct reading direction and legibility of that revision remain pending its engine run.
- [FIELD OBSERVATION 2026-09-19, corrected fire-glyph clock request] The first 180-degree correction made glyphs upright but left/right reversed. The requested basis is orientation 180 composed with a separate mirror: cell-right (+d,+d), cell-down (-2d,+2d). Orientation and reflection are independent; a square cannot distinguish either. This composition and FLAME1 material remain pending engine confirmation.
- [VERIFIED 2026-09-20, fire-glyph clock fixed run] The Gaia-art renderer is engine-verified end to end with FLAME1/939, orientation180 plus mirror, Go Mono Bold 8x16 glyphs, and incremental cell diffing. The sidecar had a footer, 69 update rows, zero non-success results, calibration_created=81, sentence_cells=303, and four ladder rows at 109 cells each. Totals were 1,159 added and 1,023 removed cells; the clock averaged about 150 live cells and 32 flames per character. The readable default pitch remains an author choice among 0.0875/0.125/0.175/0.25.
- [VERIFIED] Editor: Ctrl+G toggles grid snap (no UI indication). Sub-tile placement of a BLOCKING unit is what lags pathing; kit auto-snaps only when footprint AND collision_size_z are nonzero.

## Deletion (the hard part)
- [VERIFIED] Flames (type 20) crossfade on create and die to `xsSetUnitHitpoints(u, 0.0)`. Campfire 304 pops instead of fading.
- [VERIFIED] `xsRemoveUnit` does NOT remove eyecandy (gaia or owned): returns false for gaia.
- [VERIFIED] Static plants (type 10, class 14) cannot be removed by any XS call (hitpoints 0 leaves them). Triggers CAN delete them (remove_object by area or selected_object_ids). Rules-only design: POOL them (create once, park off-map, move with xsSetUnitPosition).
- [VERIFIED] Parking many units on ONE tile makes later `xsCreateUnit` fail (-1). Park on a per-unit grid.
- [VERIFIED] `xsCreateUnit` returns -1 for player-OWNED plants; create plants as gaia.
- [VERIFIED 2026-09-22, Gaia stump retirement probe v1/v2/v3] Parking a unit with `xsSetUnitPosition` outside the map bounds caused a deterministic DE null-pointer read (`EXCEPTION_ACCESS_VIOLATION`, READ target `0x14`, executable offset `0x11E1A1F`) during the second retirement burst. Two crashing runs had the same dump signature; removing the stale-ID occupancy query did not change it. V3 changed only the pool geometry from 64 columns to a 16x16 sub-tile grid inside the 112..116 kill zone on a 120x120 map, then completed 30 ticks with footer, zero move failures, zero invalid IDs, and bounded tracked cells (46..76). The narrow engine fact is that keeping retirement coordinates in bounds removes this crash sequence; the engine's internal null lookup is not identified.

## Lookup
- [VERIFIED] `xsGetPlayerUnitIds(player, type)` RETURNS an array id; read with `xsArrayGetSize`/`xsArrayGetInt`. It does not fill an array you pass.
- [VERIFIED] Classes are addressed as `900 + class_id` in that call (e.g. 906 infantry, 900 archer). Bare class ids are read as unit ids (only archers died: unit 4 == class 4 coincidence).
- [VERIFIED] Gaia eyecandy is invisible to `xsGetPlayerUnitIds(0, ...)`; keep your own id arrays.
- [VERIFIED] Gaia wolf/sheep stand still in scenarios; gaia hawk 96 wanders on its own.

## Modify Attribute from XS
- [VERIFIED] `xsEffectAmount(op, unitType, attr, value, player)` is trigger Modify Attribute. op 0 = set. attr 71 = standing graphic: value 0 makes the type INVISIBLE while captions still render (the caption-carrier trick). attr 0 = hitpoints (king given 20000 + per-tick `xsSetUnitHitpoints` = immortal until scripted death).
- [VERIFIED] A hero with an attack animation reappears when it fights (only the standing graphic was blanked); use a non-fighting carrier (gaia trade cart 128) and keep gaia/owner neutral.

## Trigger bridge
- [AUTHORING CONVENTION, 2026-09-19] Every active non-Gaia diagnostic player gets a Barracks Age1 (unit 12), away from the experiment. Outpost-only probes repeatedly required repair in testing; the exact engine cause is NOT established. Do not infer that outposts are not buildings or universally cause defeat. `kit scen blank` now includes barracks for every active slot by default (including the editor-compatible minimum second slot); use `--no-starters` only for an intentionally empty authoring canvas. `lint --load-safety` warns if a slot lacks a recognized stock production-building starter. Custom DAT buildings require engine verification, not a blanket rejection.
- [VERIFIED 2026-09-19, trigger-removal probe v1 sidecar] TRIGGER AMMO: a NON-looping trigger gated on variable_value(var == k) with remove_object(selected_object_ids=[id]) deletes class-14 eyecandy (Plant 1366, ROCKX 623) that no XS call can delete. XS fires shot k with xsSetTriggerVariable(var, k). Paced one shot per 20 ticks: 220/220 landed, 0 missed, latency mean 20.0 ticks (~333 ms at 60 Hz), max 30. BURST: 20 shots on 20 consecutive ticks through ONE variable -> 1 of 20 landed (the trigger system samples the variable once per its cycle; intermediate values are lost). So: one deletion per variable per ~20 ticks; for more, spread shots across MULTIPLE variables (each with its own trigger set). xsCreateUnit of a fresh gaia plant after each shot worked (visible marching line).
- [VERIFIED 2026-09-19, trigger-removal probe v2 sidecar] Variable-gated triggers are EVALUATED ONCE PER GAME SECOND (removals landed in batches at ticks 210, 240, 270...; latency = wait for the next boundary: min 3, mean 14, max 32 ticks). A variable changed twice inside one second loses the first value (45/240 lost with 24 vars fed one shot per tick). CEILING: deletions per game second = number of variables; each variable changes at most once per game second. Not a per-frame primitive; a lifecycle one.
- [VERIFIED 2026-09-19, trigger-removal probe v3 sidecar; "very fast, all items moved"] KILL-ZONE SWEEP is the production removal primitive: xsSetUnitPosition the doomed unit into a fixed off-corner area (works on ALL eyecandy), and one-shot remove_object(area, gaia) triggers gated on variable_value(sweep == k), k advanced by XS once per game second, clear the whole zone. 240/240 removed, 0 missed, one unit per tick in; latency = wait for the next second boundary (min 3, mean 17, max 32 ticks); sweeps landed at ticks 122,151,181,211... Ammo = seconds of runtime (600 one-shots = 10 min), independent of unit count. Supersedes per-unit trigger ammo (v1/v2) for anything but tiny counts.
- [POLICY, 2026-09-19] Removal: anything that DIES to hit points (fire; trees ONLY IF they carry a wood resource, a decorative tree does not die; buildings; units) may use xsSetUnitHitpoints(0); the engine plays its death. Anything that ignores hp (1-hp class-14 eyecandy: plants, rocks, rugs) goes through the kill-zone sweep. The line is "does it die to hp", not the unit's category.
- [VERIFIED by structure, 2026-09-18] Explicit `reference_id`s in a recipe used to COLLIDE with kit's auto-numbered later units (one multi-site test scenario had 10 duplicate ids, every site but one; another had two). Caption triggers targeted IDs shared by red soldiers. Fixed in kit (reserve explicit ids first, reject collisions). Check imported files with `kit scen lint`.
- [VERIFIED] A test scenario rebuilt with unique reference IDs has a repeatable countdown (2026-09-18 replay, tester: "the countdown works on this one repeatedly"). This is one in-game run supporting the collision repair as the countdown fix; it is not a separate confirmation of every carrier site.
- [VERIFIED] `xsSetTriggerVariable(n, v)` + a looping trigger per digit `variable_value(n) == d` with `change_object_caption` on `selected_object_ids [reference_id]` paints digits on an invisible carrier: the SpiRaL Front Towers countdown, reproduced. Reliability across several carriers in one scenario is still open.
- [VERIFIED] kit recipe `xs.mode = "carrier"` embeds the whole script in the Script Call trigger; no loose `.xs` needed (Error 0354 was the external-file mode).
- [VERIFIED] kit diplomacy recipe: `from`/`to` are 0-based, stance 0 ally / 1 neutral / 3 enemy. Neutral units still auto-attack; only ally stops it.

- [VERIFIED 2026-09-19, string-comparison atom probe] Tested ASCII case pairs compare equal: `A == a` is true; `A < a`, `a < A`, `T < t`, `t < T`, and `z > Z` are false. Single-character equality, <= on equal strings, concatenation, prefix ordering, and Boolean OR all behaved consistently across the four tested expression contexts. Comparison alone cannot recover original ASCII case. On build 101.103.54800.0 the string builtins `ord`, `strCharAt`, `strSubstring`, and `strToUpper` also execute (see the trig-long entry below), so case-exact character access no longer needs comparison-based decoding. General Unicode collation is not verified by this probe.

## Parser landmines (game tokenizer, not xs-check)
- [VERIFIED 2026-09-19, glyph-renderer load failure] `vector(parkX,parkY,-1.0)` failed with Error0341 (parkX not const), Error0180, and parseVectorConstant failure. Use `xsVectorSet(x,y,z)` for runtime coordinates, including computed expressions; `vector(...)` is the compile-time form. Kit's conservative `xs_vector_nonliteral` error accepts only three signed numeric literals, not arbitrary constant-expression evaluation. This is an authoring restriction, not a claim that the engine rejects every named compile-time constant.
- [VERIFIED] Non-ASCII inside a string literal = "string not terminated", whole script dead. `LC_ALL=C grep -nP '[^\x00-\x7F]'` the generated file.
- [VERIFIED] `if (a && b && c && d)` = Error 0308; group pairwise `(((a)&&(b))&&((c)&&(d)))`.
- [VERIFIED] Names are unique per function scope, for-loops declare their own counter; `int` first in mixed arithmetic errors (put the float operand first).
- [VERIFIED] `%` inside an `if` condition (`if (e % 2 == 0)`) = Error 0308 "illogical or invalid expression". Hoist it: `int m = e % 2; if (m == 0)`. Modulo in a plain assignment is fine.
- [VERIFIED] Subtraction inside an `if` condition (`if (now - born >= 12)`) produces Error 0308, as does the separately observed modulo-in-condition case. Hoist the arithmetic: `int age = now - born; if (age >= 12)`. The broader binary-arithmetic lint is conservative; addition, multiplication, division, and arithmetic specifically inside `while`/`for` conditions are not individually engine-verified here. Unary sentinels such as `if (old != -1)` remain valid.
- [VERIFIED] Reported line numbers are offset ~+4 in carrier mode; the cascade names an innocent function, read the FIRST error.

## Visual notes from review (design, not engine)
- Particle counts read as timid at hundreds; think thousands on screen (3,233 static units were barely noticeable on a low-end test machine).
- Explosion signature = slow "wall of fire" front + afterglow carpet; fast blast rings second; 8-spoke starburst and spirals read as magic, not explosions.
- Trails: chariot technique (lay flames behind a mover) beats halos.

## Pending probes
- [HYPOTHESIS] xsEffectAmount attr 70/72/75/76 (attack, standing2, walking, running; ids from the UGC prelude.xs) = 0 on SHARKATZOR 1222 makes a TASKED (walking) carrier invisible while its caption glides. A one-shot trigger (variable_value, task_object) tests it; upgrade to VERIFIED on the next rerun.
- [HYPOTHESIS] A 43-field sidecar probe is not yet run: xsSetAttribute behavior, xsTriggerVariable read-back, and more.

## Builtin-name collisions (2026-09-19, corrected)

A test scenario's Error 0014 (`ln` already defined) was caused by declaring
`int ln`: `ln` is the natural-logarithm builtin. Hoisting did NOT fix it;
renaming was required. The previous loop-scoping diagnosis is retracted,
and Kit's loop-body-declaration warning has been removed.

Kit embeds 204 function names from the UGC Guide's
`docs/general/xs/prelude.xs` and reports variable, parameter, and function
declarations colliding with them as `xs_builtin_name_collision` errors.
Source: https://github.com/Divy1211/AoE2DE_UGC_Guide . The snapshot contains
27 (not 26) unprefixed helpers: abs, acos, asin, atan, atan2, atan2v, bitAnd,
bitCastToFloat, bitCastToInt, bitNot, bitOr, bitXor, ceil, cos, degrees, dist,
exp, floor, ln, log10, log2, pow, radians, round, sin, sqrt, tan.
The ln collision is engine-observed; the full list is reference-derived.
Rename declarations, not calls. No reference checkout is needed at runtime.
The separate same-function duplicate-declaration check remains; the `ln` case
is not evidence for its scope semantics. These checks are not an XS compiler.

## Sidecar and trigger probe constraints

- [VERIFIED 2026-09-23, trig-long on DE 101.103.54800.0] A single XS sidecar stops at exactly `1,048,575` bytes (`2^20 - 1`). Writes beyond that boundary were silently absent from the append-readable file. Treat this as a hard budget when designing unattended probes; 91 one-shot facts with before/result rows fit comfortably, but looping or broad attribute sweeps can exhaust it. The exact failure mode beyond the boundary remains open.
- [VERIFIED 2026-09-23, trig-long] A fact trigger must be both serialized with `looping = no` and explicitly deactivated after its result. The trigger-driven chain otherwise re-fired activated fact triggers on the live run; only the driver should loop. This is a probe-authoring rule, not a claim that every hand-authored trigger chain behaves identically.
- [VERIFIED 2026-09-23, trig-long on DE 101.103.54800.0] `sin`, `cos`, `radians`, `ord`, `strCharAt`, `strSubstring`, `strToUpper`, `strSplit`, `strJoin`, and `fstr` executed. The probe included non-finite arithmetic, so decoders must preserve NaN/Inf as explicit non-numeric values rather than failing JSON serialization.
