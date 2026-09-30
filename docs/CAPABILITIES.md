# AoE2Kit Capabilities

A domain-by-domain map of what AoE2Kit does. Exact invocations, flags, and
returned JSON keys for every command are in `API_REFERENCE.md` (machine-readable
twin: `api_reference.json`), generated from the binary itself.

Where a capability has a hard boundary, this page says so. Reports carry a
`structure_verified` vs `engine_verified` label: structure-verified means Kit
re-reads and rebuilds the data exactly; engine-verified means the behavior was
observed in the game.

---

## 1. Scenario files (`kit scen`, `pkg/scenario`)

**Read:** full typed decode of DE 1.58 and 1.59 `.aoe2scenario` files, every
section from the file header through Files. Kit proves each read by rebuilding
the decompressed body byte-for-byte (`kit scen verify`). Older versions fail
with `unsupported scenario version`; re-saving them in the DE editor converts
them to 1.59. On top of the decode sit graded views:

- `info` / `check` / `describe` / `settings`: players, map, units, trigger
  fingerprint, embedded AI files, victory and diplomacy settings, and the full
  field tree on request. `field <path>` reads one field.
- `triggers` (with `--conditions`), `effects` (per-type summaries, `--census`
  for count-only passes over huge files, `--where` to stream one effect
  family), `units --named`, `strings`, `map`, `terrain`.
- `glossary`: the first command to run on an unfamiliar scenario (effect
  families, name prefixes, variable ids, XS calls, unit/tech/attribute
  vocabulary). `idioms` names detected techniques (colored text, HUD loops,
  caption objects, XS bridges, relay graphs) with confidence, without claiming
  author intent.
- `diff` compares two scenarios by parsed facts (`--all-fields` for every
  field); `bytediff` and `coverage` report which bytes Kit decodes.

**Write:** recipe-driven patching to a separate output file, always re-verified
by reopen, rebuild, and invariant checks. Recipes can append, edit, copy, and
remove triggers (effects and conditions included), add/edit/move/copy/remove
units, paint and generate terrain, attach XS, edit player slots, diplomacy,
starting resources, and victory settings. `import-triggers` copies triggers
between scenarios. `blank` creates a new scenario; `smoke` writes a canonical
editor-visible test pattern. Structural map resize is not exposed. See
[`SCENARIO_WRITER.md`](SCENARIO_WRITER.md).

**Safety:** `lint` catches engine hazards (looping display spam, corrupt
strings, invariant violations, reference-id collisions) and cites engine-fact
IDs. `deploycheck` statically preflights XS
deployment: script-name/path literal match, on-disk resolution, include chains,
and cross-file `extern` discipline. `delete-plan` answers what a safe delete
means before anything mutates.

## 2. Replays (`kit replay`, `pkg/replay`)

**Identity and metadata:** `summary` gives record hash, versions, scenario
name, map hash, active data-mod identity, lobby settings (game type, speed,
treaty, population), duration, and players with civ/color/team/profile id/
rating. `identify` fingerprints against growable registries (trigger-graph hash
first, terrain hash as fallback).

**Events and actions:** `events` (chat, taunts, flares, resigns, telemetry, with
result inference), `actions` (decoded move/order/build/delete/resign/queue/
research and more, with unknowns preserved raw), `player-events` (a filterable
canonical event table), `chat` (merges in-game and lobby chat, dedupes echoes,
and tags chat the game injects from a prior session as `backlog`), `feedback`,
and `camera` (the recording player's own view stream).

**Sync/checksum layer:** `sync` surfaces the periodic sync record with decoded
word meanings (stockpile, unit-type sum, object count, player, instance-id sum;
some words partial or undecoded). `player-series` turns that into per-player
state curves. `checksum-probe` matches expected fixture pulses. `combat` joins
net object-loss windows to nearby targeted commands without claiming kills.

**Objects:** `objects` / `object-state` / `object-shapes` index the initial
objects: owner, type, HP, position, production queues. `--objects` enrichment
lets event rows name an object instead of printing its id.

**Byte accounting:** `coverage` is a ledger in which decoded and opaque bytes
must sum to the file size, so parser gaps cannot hide. `opaque-spans`,
`frontier`, and `opaque-clusters` profile what is still undecoded across a
corpus. `diff-state` and `scan-value` support controlled experiments: plant a
known value in a fixture and find the region it lands in.

**Game data in replays:** `effective-data` / `effective-units` /
`datamod-check` read the per-player effective game-data table embedded in
replays, decode the unit-availability array, cross-reference DAT slots, and can
report whether a game was modded against a baseline. `ai-manifest` extracts AI
module references. `postgame` / `postgame-corpus` read post-game metadata; DE
does not serialize scoreboard statistics in the recording, which is why the
sidecar route (below) exists.

**Acquisition:** `fetch` pulls replays from Microsoft's public endpoint by game
id or profile id, with `--all-povs` to fetch every player's recording of a
game, cached. See [`AOE_MS_REPLAY_API.md`](AOE_MS_REPLAY_API.md).

**Batch:** `corpus`, `inbox` (playtest intake: dedupe, issue cards, per-replay
story lines), `issues`, `telemetry` (schema-validated markers), `unknowns`
(corpus-wide undecoded-action mining), and `story` (a designer-facing brief
with a `claims` ledger and a `missing` list stating what it cannot support).

## 3. XS and the sidecar loop (`kit xs`, `kit xsdat`, `pkg/xs`)

Authoring generators: `bridge` (named wrappers over anonymous trigger
variables, with read/write lint), `shims` (scan XS modules and emit
script-call trigger recipes), `datagen` (JSON to xsArray init modules).
`xsdat decode --ledger` decodes `.xsdat` sidecar files with assertion-fixture
semantics: declare expected rows, run the game unattended, get pass/fail.
`replay sidecar-sync` correlates sidecar ground truth with the replay's sync
records, and `replay carrier` generalizes self-reporting probes. See
[`SIDECAR.md`](SIDECAR.md), [`XS_AUTHORING.md`](XS_AUTHORING.md), and
[`XS_CHECK.md`](XS_CHECK.md).

## 4. DAT engine (`kit dat`, `pkg/datfile`, `pkg/datcodec`)

A span/index patch engine rather than an eager object-graph port: unknown bytes
are preserved, and every write is verified by re-index, readback, and neighbor
canaries.

**Read:** units, techs, effects (with semantic overlays and `effect-explain` /
`tech-explain` prose for named command types, packed attack/armor deltas,
local-building effects, and rename-unit string ids), ability joins (tech plus
effect as one authoring concept), civs, graphics (including particles), sounds,
terrains, tech tree, unit headers, `refs` (who points at an id), `availability`
(per-civ trainability), `diff` (decoded comparison of two DATs), `roundtrip`
(codec safety gate), and `sql` (a SQLite export; see
[`DAT_SQL_EXPORT.md`](DAT_SQL_EXPORT.md)).

**Write:** patch units (HP, graphics, attacks, armor, costs, tasks, train
buttons, and more), patch/create/delete unreferenced graphics, create units,
effects, techs, and sounds, create paired ability records, edit tech-tree
connections, and semantic deletes with reference ledgers. `delete-plan`
refuses deletes that would leave dangling references. See
[`DAT_ENGINE.md`](DAT_ENGINE.md).

## 5. Graphics and FX (`kit gfx`, `kit fx`)

`gfx info/export` parses SLD sprite containers and exports PNG frames.
`fx new/bind/lint` is the custom-particle pipeline: descriptor JSON, DDS atlas,
metadata, and DAT binding.

## 6. AI files, mods, packaging (`kit ai`, `kit mod`, kit core)

`ai fingerprint/lint/diff` hash and sanity-check `.ai`/`.per` files, with a
registry keyed to replay-side AI signatures. `mod check` diagnoses mod
activation, including the case where a local mod exists but the game is
running the default data set. `inventory` / `manifest` / `verify` / `pack` /
`portable-check` make the kit self-verifying and self-packaging, with
local-path-leak detection.

## 7. Verification harnesses

`verify-run --contract` checks a replay against pre-registered expectations
(identity, data set, telemetry, chat markers, result); render claims stay
`unknown` until a human confirms them. `ci init/check` is a per-project
post-playtest gate. `release-check` runs identity, lint, diff, mod check, and
pack as one gate. `campaign generate` writes cross-scenario `.xsdat`
persistence modules. `facts list/check` reads an engine-facts ledger
(`data/engine_facts.json`: each fact with tier, date, and fixture citation)
when one is present; the public release does not ship a ledger, and lint
findings then cite fact IDs only. See
[`ENGINE_HARNESS.md`](ENGINE_HARNESS.md).

## 8. Network and registries

`player stats` (Microsoft career-stats API, labeled as career aggregates, not
per-match truth), `replay fetch`, and `identify --register` / `collect`
(folder-scale fingerprint-and-register). `known_scenarios.json` and
`known_ais.json` are registries whose description field is written as
orientation text for whoever matches the hash.

## 9. Castle Blood Automatic (`kit cba`, `pkg/cba`)

Interpretation specific to the Castle Blood Automatic scenario lives here and
consumes the generic facts without redefining them: side-channel decoding,
per-civ phase thresholds (joined with DAT data), phase detection, raze and
spawn analysis, matchup overlays, and a balance table mined from the
scenario's changelog. It also includes a ladder system: a JSONL game corpus,
multi-axis contribution scoring, memory-bounded ingestion workers, and a SQL
export.

## 10. Other

`swatch patterns` (map-tile geometry generators), `scen regions` (region
context files), replay context JSON (name your map's regions without changing
generic facts; see [`REPLAY_CONTEXT_EXAMPLE.json`](REPLAY_CONTEXT_EXAMPLE.json)),
and resource guardrails (a 1 GiB heap soft limit and a 32 MiB scenario-inflate
refusal).

---

## Known limits

- Scenario versions before 1.58 are not read.
- Some replay object-body fields remain opaque.
- Some sync-record words are only partly decoded.
- Per-kill attribution is not present in replays; Kit uses anchors and
  sidecars instead.
- Rendering is not verified automatically; there is no screenshot oracle.
