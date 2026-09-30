# AI GUIDE

Read this before changing any bundled AoE2 artifact.

## Rules

1. One Go source tree, one local build command, no Python. Only the fetch
   commands (`replay fetch`, `player stats`, geo-trace data downloads) use the
   network. If a task tempts you to write a Python/bash script for durable work,
   STOP. That means a Go tool is missing. Record the gap; do not script around
   it.

2. AoE2Kit is project-neutral. Shared kit code must not assume any particular
   scenario, project, or local path. Scenario-specific interpretation lives in
   its own layer (see `pkg/cba`), and project facts belong in the bundle's own
   artifacts and notes.

3. Read before you mutate. Every write path must follow:

```text
inspect -> dry-run/explain -> apply -> verify -> report evidence
```

## First Moves

```sh
go build -o kit ./cmd/kit
./kit doctor
./kit summary .
./kit inventory
./kit project inspect . --limit 50 --text
./kit recipe list --text
```

Use the output to identify what the bundle contains: scenarios, records, data
files, AI files, and local mod folders.

## Resource Guardrails

Kit sets a default soft Go heap limit of 1 GiB unless `GOMEMLIMIT` or
`AOE2KIT_GOMEMLIMIT_MB` is set. Scenario parsing refuses inflated bodies above
32 MiB by default because some community scenarios inflate to very large trigger
graphs. For an intentional large teardown, run one command at a time and set
`AOE2KIT_MAX_SCENARIO_MB` higher or use a command-specific cap such as
`kit scen effects huge.aoe2scenario --census --max-mb 192`,
`kit scen effects huge.aoe2scenario --where 105 --max-mb 192`,
`kit scen triggers huge.aoe2scenario --grep mana --max-mb 192`,
`kit scen glossary huge.aoe2scenario --max-mb 192`, or
`kit scen idioms file.aoe2scenario --text`. Do not use
`AOE2KIT_ALLOW_HUGE_SCENARIO=1` unless you genuinely need an unlimited full
parse.

## Engine Facts

Engine-verified XS facts are documented in
`docs/XS_ENGINE_VERIFIED_2026-09.md`, scoped to a game build by
`docs/ENGINE_BUILD_SCOPE.md`. Scenario lint and deploycheck findings cite a
`fact_id`. When a bundle ships an engine-facts ledger
(`data/engine_facts.json`), `kit facts check --text` validates it and
`kit facts list --text` lists it; the public release does not include one.

## Current Write Boundary

Scenario writing is implemented as a controlled, typed recipe patcher, not a
general editor clone and not an unrestricted map generator. Kit reads and writes
DE `1.58` and `1.59` scenarios with typed codecs and rebuilds them exactly;
older versions are rejected (re-save them in the DE editor to convert). See
`docs/SCENARIO_WRITER.md` for the full recipe surface. Main surfaces:

- `players`: set player slot active/human state and AI name/type fields.
- `diplomacy`: set one player-to-player stance at a time.
- `resources`: set food, wood, stone, gold, and trade-goods values.
- `victory`: set authored GlobalVictory fields such as `conquest_required`.
  For lobby-proof diagnostics, set `conquest_required=0` in the scenario file;
  DE single-player scenario lobbies do not expose the random-map Victory
  dropdown.
- `xs`: embed an XS entry script payload and scenario XS name/path fields.
  Follow `docs/XS_AUTHORING.md`; runnable bundles should also ship the XS module
  tree under `resources/_common/xs/`.
- `units`: add, edit, and remove units in existing player unit sections.
- `map`: edit existing terrain tiles by rectangle, circle, or line. Tile
  terrain id, elevation, and layer are supported; map dimensions are not
  resized.
- `triggers`: add, copy, edit, and remove triggers, their effects and
  conditions, and variables, while maintaining counts and display-order arrays.
  `kit scen import-triggers` copies triggers between scenarios.

The supported effect and condition types are listed in
`docs/SCENARIO_WRITER.md`. The DE editor fills some values on save that a
minimal recipe does not; see "Editor Normalization" in
`docs/AOE2KIT_WRITE_SMOKE.md`.

Data-mod writing is available for the current AoE2DE `.dat` patch surfaces
documented in `docs/DAT_ENGINE.md` and `docs/DAT_AUTHORING.md`: graphic record
strings/fixed fields; existing unit/common/Type50/Creatable fixed fields,
row patches, and action task-row patches; graphic/unit record creation;
codec-backed Effects/Techs/Civ/Terrain/Sound/PlayerColour recipe edits;
semantic unit tombstones; and semantic sound mutes. AoE2Kit does not target
obsolete `.dat` layouts unless a current workflow proves they matter.

If the needed write is outside that surface, record the Go-tool gap. Do not use
Python `genieutils` as a durable substitute.

Before authoring a new patch JSON from scratch, check `kit recipe list` and
`kit recipe show <name> --recipe-only` for a native scenario/DAT starting shape.
After generating an output artifact, use `kit project lineage --input PATH
--output PATH --manifest lineage.json` when the handoff needs explicit input and
output hashes.

For visual sprite inspection, use `kit gfx info/export` on `.sld` files. The
current exporter writes main-layer PNG frames for teardown; it does not compose
shadow, damage, or player-color layers into a final in-engine render.

## XS Authoring Helpers

Use `kit xs` for durable XS glue instead of generating anonymous bridge code by
hand:

- `kit xs bridge <variables.json> --out-dir DIR`: generate named
  `xsVariableN` accessors plus trigger-variable definition JSON and read/write
  lint.
- `kit xs shims <file-or-dir>... --out recipe.json`: inventory XS functions and
  emit trigger `script_call` recipe fragments using the effect `message` field.
- `kit xs datagen <arrays.json> --out module.xs`: generate xsArray data modules
  from structured JSON.
- `kit xs inspect <file.xsdat> [--types string,int,...]`: decode XS sidecar
  bytes written by `xsWriteString`, `xsWriteInt`, `xsWriteFloat`, and
  `xsWriteVector`. Without `--types`, this is heuristic because `.xsdat` files
  do not carry embedded type tags.
- `kit xsdat decode <file.xsdat> --ledger expected.json`: decode an XS sidecar
  and compare each value against an explicit expected payload ledger. Use this
  for assertion fixtures where missing, truncated, or wrong runtime writes must
  fail loudly.
- `kit xsdat decode <file.xsdat> --schema NAME --text`: decode a sidecar
  written by one of Kit's built-in generators (for example `engine-harness`,
  `customstore`, or the `rtv*` diagnostic schemas) as named rows, including
  partial runs without a footer.
- `kit replay sidecar-sync <file.aoe2record> --xsdat run.xsdat (--ledger
  expected.json|--schema NAME)`: correlate typed sidecar rows with nearby
  replay sync-matrix samples. Treat unnamed-word and attribute-mirror output as
  candidates until cross-fixture evidence promotes it. Use `--fail-on-mismatch`
  for automated fixture gates; it fails on sidecar decode failures, missing sync
  samples, or known-word mismatches.

These are structural authoring tools. A real scenario run is still the oracle
for XS syntax, include resolution, and runtime behavior.

Before handing off an XS-backed scenario for an engine run, preflight the
scenario against the exact deployed mod/profile tree:

```sh
./kit scen xs path/to/file.aoe2scenario --deploy-tree path/to/deploy-root --text
./kit scen deploycheck path/to/file.aoe2scenario path/to/deploy-root --text
./kit ci init path/to/file.aoe2scenario --root path/to/project-ci --text
./kit ci check path/to/project-ci/aoe2kit-ci.json --text
./kit campaign generate docs/CAMPAIGN_SPEC_EXAMPLE.json --out-dir build/campaign-xs --text
```

`scen xs` inventories attached XS, script_call effects, called function names,
function definitions, includes, declarations, and cross-file symbol hazards.
`scen deploycheck` gates handoff: it catches mismatched scenario XS filename
fields, missing deployed XS entry files, unresolved includes, and cross-file
non-`extern` symbol references before the engine opens its modal error box. It
is still
`structure_verified_not_engine_verified`; a clean deploycheck does not replace
the engine run.

For repeat playtests, use `kit ci` as the local loop gate. `ci init` creates a
portable check config, an example replay contract, and a small neutral debug XS
include. `ci check` composes `verify-run` and `replay sidecar-sync`; any failed
or unknown check exits nonzero. Keep project-specific assertions in the CI
config, not in Kit source.

For same-profile campaign persistence, use `kit campaign generate`. It produces
XS writer/reader modules that use the engine-verified bare writer-scenario
`.xsdat` read convention. This is only the same-party fast path; mixed-party
multiplayer continuity still needs explicit synchronized re-declaration logic.

For replay chat, use `kit replay chat <file.aoe2record> --text` when you need a
human/player transcript. Trigger-effect `send_chat` output is not retained in
recordings. Treat `source=backlog` rows as likely prior-session chat injected by
DE at the start of a new recording, not as live commentary from that match.

## Reports

When you finish work, report:

```text
what changed
commands run
evidence: counts, hashes, parsed artifacts, before/after checks
gaps found in AoE2Kit
```

Never report only "done."
