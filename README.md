# AoE2Kit

[![License: LGPL-3.0-only](https://img.shields.io/badge/License-LGPL--3.0--only-blue.svg)](LICENSE)
[![Go 1.23+](https://img.shields.io/badge/Go-1.23%2B-00ADD8.svg)](go.mod)
[![CGO: none](https://img.shields.io/badge/CGO-none-brightgreen.svg)](go.mod)

A Go toolkit for Age of Empires II: Definitive Edition files: scenarios,
replays, `.dat` data mods, XS scripts and sidecars, sprites, particles, and
local mods. It reads and writes them **without the game running**, and it is
built so an AI agent can drive it as easily as a person can.

---

## Start here

Requires Go 1.23+. Pure Go, no CGO; one `go build` produces one binary.

```sh
go build -o kit ./cmd/kit
./kit                      # every command
```

Four commands that show the range:

```sh
./kit scen glossary  <file.aoe2scenario> --text   # vocabulary of an unfamiliar scenario
./kit replay summary <file.aoe2record>   --text   # players, civs, teams, result, map
./kit replay chat    <file.aoe2record>   --text   # full transcript, lobby + in-game
./kit xsdat decode   <file.xsdat>        --text   # typed rows from an XS sidecar
```

Output is JSON by default; `--text` renders it for people. Programs should parse
the JSON.

Investigating a recording from a game that lagged, froze, or crashed?

```sh
./kit replay health game.aoe2record --text
```

This low-memory report shows where the recording ends, how object and command
counts changed, and chat mentioning problems. It does not claim to know what
caused a crash. Unzip recordings first.

---

## What it does

- **Scenarios (DE 1.58 and 1.59).** Kit's own typed codecs decode every section
  and rebuild the file byte-for-byte. Writing uses JSON recipes: triggers,
  effects, conditions, variables, units, terrain, players, diplomacy, victory
  settings, and embedded XS. Analysis tools cover trigger search, technique
  detection, lint, reference-safe deletes, and field-level diffs.
- **Replays.** Identity, players and results, chat, flares, decoded commands,
  the per-player sync matrix, initial object state, embedded game data, and a
  byte-accounting ledger that shows exactly what is still undecoded.
- **`.dat` data mods.** Read and edit units, graphics, effects, techs,
  abilities, sounds, civs, terrains, player colours, and the tech tree, with
  reference-aware delete planning and a SQLite export.
- **XS.** Authoring generators, a static checker for known engine hazards, and
  the sidecar technique: a scenario writes typed state to a file you decode
  after the game, which lets you assert engine behavior without watching.
- **Graphics.** SLD sprite export and custom particle packaging.

The full map, with its boundaries, is [`docs/CAPABILITIES.md`](docs/CAPABILITIES.md).

---

## The documents

| file | what it answers |
| --- | --- |
| [`docs/CAPABILITIES.md`](docs/CAPABILITIES.md) | what the kit can do, domain by domain, with limits |
| [`docs/GO_AOE2KIT.md`](docs/GO_AOE2KIT.md) | architecture, conventions, packaging, and the index of domain guides |
| [`docs/API_REFERENCE.md`](docs/API_REFERENCE.md) | every command: flags, inputs, and the JSON keys it returns (generated) |
| [`docs/api_schemas.json`](docs/api_schemas.json) | the nested shape of all 989 report types (generated) |
| [`docs/PACKAGE_API.md`](docs/PACKAGE_API.md) | `go doc` for importing the library directly (generated) |
| [`docs/SIDECAR.md`](docs/SIDECAR.md) | the XS sidecar debug loop: assert engine state without playing |
| [`AI_GUIDE.md`](AI_GUIDE.md) | rules and first moves for an AI agent working with the kit |
| [`CHANGELOG.md`](CHANGELOG.md) | what changed, release by release |

If you only read two: `CAPABILITIES.md` to see whether the kit does what you
need, then `SIDECAR.md`, the technique most likely to be useful in your own
scenarios whether or not you ever use this toolkit.

---

## The one idea that runs through everything

**Every report says how strongly it is claimed.**

The engine is a black box. A parser can be byte-perfect and still be wrong about
what the bytes *mean*, and a tool that reports both kinds of statement in the
same voice will eventually mislead you at the worst moment. So every report
carries an explicit label:

- `structure_verified`: the parser round-tripped it, the readback matched, the
  invariants held. The file really does say this.
- `engine_verified`: a live game or the editor was the oracle. The engine
  really does behave this way.
- `behavior_inferred_*`, `*_candidate`, `*_not_engine_verified`: a hypothesis
  with evidence, deliberately not dressed up as fact.

`kit cba replay razes` labels its output
`behavior_inferred_raze_candidate_not_engine_raze_event`, because recordings
contain no raze event; what it found is pressure correlated with a later
payoff. `kit replay story` ships a `missing` list naming what the kit cannot
know from a recording.

The rule when consuming this kit, especially programmatically: **never promote
a claim past its label.** If a report says candidate, it is a candidate.

---

## Design decisions worth knowing

**One binary, pure Go.** No CGO and a small set of pure-Go dependencies
(`golang.org/x/image`, `modernc.org/sqlite`), so it builds anywhere Go runs and
works in a locked-down AI sandbox. Only the fetch commands use the network.

**Scenarios are decoded by Kit's own typed codecs.** Every section of a 1.58 or
1.59 scenario is decoded into typed values; bytes whose meaning is not yet known
are carried opaquely, so an untouched file rebuilds byte-for-byte. Unrecognized
structures fail closed rather than being guessed. See
[`docs/SCENARIO_WRITER.md`](docs/SCENARIO_WRITER.md).

**The `.dat` engine patches spans; it does not model the world.** It indexes
records, patches the fields it knows, splices whole records when a length
changes, preserves every unknown byte, and verifies by re-inflating and
re-indexing to EOF. That is why it can safely edit a modern DE `.dat` that
nobody has fully mapped. See [`docs/DAT_ENGINE.md`](docs/DAT_ENGINE.md).

**Writes never touch the input.** Every patch writes a new file, reopens it,
re-verifies it, and checks neighboring records before reporting success.

**Replay decoding is byte-accounted.** `kit replay coverage` reports every
region as `decoded` or `bounded_opaque`, and the two must sum to the file size.
A parser gap cannot hide as silence.

**Generic truth and domain interpretation are separate.** `pkg/replay` and
`pkg/scenario` describe file formats. `pkg/cba` interprets one game mode and
consumes the generic facts without redefining them. If you build interpretation
for your own scenario, that is the seam to copy.

**The API reference is generated.** A static pass reads the source and emits the
nested shape of every report type (`api_schemas.json`). A runtime pass executes
read-only commands against fixtures and records the keys they return, which
proves each command works. Commands that could not be executed are labeled with
the reason.

---

## Working with an AI agent

- **Structured output everywhere.** JSON first, complete objects, no scraping.
- **It describes itself.** `./kit` lists every command; `go run ./cmd/apiref`
  regenerates the full reference for your build.
- **Orientation is carried in data.** `known_scenarios.json` and
  `known_ais.json` map fingerprints to descriptions, so a hash match becomes
  context for a model that has never seen the file.
- **Honest failure.** Unknown replay actions are preserved raw with sample
  payloads, so the agent can see what it does not understand.

A workflow that works well: `scen glossary` before editing an unfamiliar
scenario, `replay coverage` before trusting a replay claim, and
[`docs/XS_ENGINE_VERIFIED_2026-09.md`](docs/XS_ENGINE_VERIFIED_2026-09.md) before
asserting anything about engine behavior. Start with [`AI_GUIDE.md`](AI_GUIDE.md).

---

## What it cannot do

Stated plainly, because the frontier moves and a stale claim is worse than none:

- **No per-kill attribution from ordinary replays.** Recordings carry no death
  or kill-credit events. The kit infers combat pressure from object-count
  movement and says so.
- **Object bodies are only partly decoded.** Initial object state is readable;
  the tails of some record types remain opaque.
- **Scenarios before DE 1.58 are not read.** Re-save them in the DE editor to
  convert them. Map resizing is not exposed.
- **No render oracle.** Kit can prove a particle's files and `.dat` binding are
  consistent; it cannot tell you the effect looked right. Reports touching
  visuals say `engine_verified: false`.
- **Postgame statistics are not in replays.** The postgame block is metadata
  only; scoreboard numbers are recomputed for display.

---

## Layout

```
cmd/kit        the CLI (scen, replay, dat, xs, xsdat, gfx, fx, ai, mod, cba, ...)
cmd/apiref     generates the API reference, schemas, and package docs
cmd/scen|dat|mod|swatch|cba   focused entry points for single domains

pkg/scenario   .aoe2scenario codecs, writer, lint, XS deploy checks
pkg/replay     .aoe2record parsing: header, actions, sync matrix, objects, coverage
pkg/datfile    .dat span/index inspection and patch engine
pkg/datcodec   typed .dat section codecs
pkg/xs         XS authoring generators, lint, and the .xsdat sidecar codec
pkg/harness    unattended engine-probe scenario generator
pkg/cba        Castle Blood Automatic interpretation (domain layer)
pkg/geotrace   real-map terrain tracing into scenario recipes
pkg/kit        bundle inventory, verification, packaging, portability checks

examples/text  a standalone XS library that draws text with Gaia objects
docs/          guides and generated references
```

More in [`docs/GO_AOE2KIT.md`](docs/GO_AOE2KIT.md).

---

## Regenerating the reference

After any CLI change:

```sh
go build -o kit ./cmd/kit
go run ./cmd/apiref
```

Pass fixtures (`--scenario`, `--replay`, `--dat`, `--xsdat`) to let the runtime
pass execute commands; commands without a fixture are labeled rather than
guessed. `--reuse-probe docs/api_reference.json` reuses the previous run when
only documentation changed.

---

## License and credits

AoE2Kit is licensed **LGPL-3.0-only** ([LICENSE](LICENSE)). Releases ship source
and attribution, not built binaries.

It stands on other people's work, credited deliberately. AGE (Advanced Genie
Editor), by Tapsa and contributors, informed the `.dat` authoring semantics and
field terminology. AoE2ScenarioParser, by KSneijders and contributors, was
consulted as a format reference while Kit's scenario codecs were reconstructed
independently from a corpus of real files; Kit does not include its code or
schema. The full ledger is in [NOTICE.md](NOTICE.md) and `data/credits.json`:

```sh
./kit credits --text
```

---

## Provenance

Built by **chrae** with AI assistance, as working infrastructure for real
scenario and replay analysis, not as a demo. It is shared because the people
who would find it useful are few and know each other.

Use it, fork it, or take just the sidecar technique and ignore the rest. If you
find a decoding error, that is the most valuable thing you can send back: the
kit is only as honest as its labels, and a mislabeled claim is a bug of the
worst kind.
