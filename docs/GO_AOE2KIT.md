# AoE2Kit Architecture and Conventions

`aoe2kit` is a Go toolkit for Age of Empires II: Definitive Edition scenario,
replay, data-mod, XS, and packaging work. It is a single Go module whose direct
dependencies are pure Go (`golang.org/x/image`, `modernc.org/sqlite`), so it
builds with `CGO_ENABLED=0` into one static binary.

This page covers how the kit is organized and the conventions every command
follows. Each domain has its own guide (see [Guides](#guides)), and every
command's flags and output keys are in the generated
[`API_REFERENCE.md`](API_REFERENCE.md).

## Build

```sh
go build -o kit ./cmd/kit
./kit                      # list every command
```

For a static binary to drop into a sandbox without a Go toolchain:

```sh
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -buildvcs=false -o kit ./cmd/kit
```

## Layout

```text
cmd/kit          the CLI: scen, replay, dat, xs, xsdat, gfx, fx, ai, mod, cba, ...
cmd/apiref       generates API_REFERENCE.md, api_reference.json, api_schemas.json, PACKAGE_API.md
cmd/scen|dat|mod|swatch|cba   optional single-domain entry points

pkg/scenario     .aoe2scenario codecs, writer, lint, XS deploy checks
pkg/replay       .aoe2record parsing: header, actions, sync matrix, objects, coverage
pkg/datfile      .dat span/index inspection and patch engine
pkg/datcodec     typed .dat section codecs (effects, techs, civs, sounds, terrains, ...)
pkg/datcache     SQLite cache for DAT unit queries
pkg/xs           XS authoring generators, lint, and the .xsdat sidecar codec
pkg/harness      unattended engine-probe scenario generator
pkg/cba          Castle Blood Automatic interpretation (domain layer)
pkg/geotrace     real-map terrain tracing into scenario recipes
pkg/kit          bundle inventory, verification, packaging, portability checks
pkg/...          supporting packages: gfx, fx, glyph, text, geom, aifile, modpack, campaign, ci, ...
```

Package documentation for importing the library directly is in
[`PACKAGE_API.md`](PACKAGE_API.md).

## Conventions

**JSON first.** Every command emits JSON by default. `--text` renders a
human-readable view; programs should parse the JSON.

**Every claim carries its strength.** Reports carry a verification label:
`structure_verified` means Kit re-read the data and it round-tripped;
`engine_verified` means the game or editor was the oracle; labels such as
`*_candidate`, `behavior_inferred_*`, and `*_not_engine_verified` mark
hypotheses. `verification.engine_verified=false` means the game has not been
used to confirm the result. Do not promote a claim past its label.

**Writes never touch the input.** Every patch command writes a separate output
file, reopens it, re-verifies it, and reports readback before claiming success.

**Fail closed.** Unrecognized structures are reported as errors or preserved
byte-for-byte as opaque data; Kit does not guess.

**Bounded output.** Table commands (`dat graphics`, `dat units`, ...) return
the first 200 rows by default and say so; use `--limit 0` or `--all` for full
tables.

**Resource guardrails.** `kit` sets a 1 GiB soft Go heap limit unless
`GOMEMLIMIT` or `AOE2KIT_GOMEMLIMIT_MB` is set. Scenario parsing refuses
inflated bodies above 32 MiB unless `AOE2KIT_MAX_SCENARIO_MB` is raised or
`AOE2KIT_ALLOW_HUGE_SCENARIO=1` is set. Run heavy replay commands one at a
time; see [`REPLAY_MEMORY.md`](REPLAY_MEMORY.md).

**Generic truth and domain interpretation stay separate.** `pkg/replay` and
`pkg/scenario` describe file formats. Scenario-specific interpretation (for
example `pkg/cba`) consumes those facts without redefining them. New shared
capability goes into a package first and is then exposed through a CLI domain.

**Identify replays from the replay.** Do not infer what was played from the
newest file in a scenario folder: DE recreates some built-in scenario state on
its own. Use `kit identify`, `kit replay info`, or `kit replay summary`.

## Guides

| domain | guide |
| --- | --- |
| scenario reading, analysis, deletes | [`SCENARIO_TOOLS.md`](SCENARIO_TOOLS.md) |
| scenario writing and recipes | [`SCENARIO_WRITER.md`](SCENARIO_WRITER.md), [`SCEN_RECIPE_EXAMPLE.json`](SCEN_RECIPE_EXAMPLE.json) |
| DE 1.59 format notes | [`SCENARIO_159_NOTES.md`](SCENARIO_159_NOTES.md) |
| editor/game write smoke test | [`AOE2KIT_WRITE_SMOKE.md`](AOE2KIT_WRITE_SMOKE.md) |
| replays | [`REPLAY_GUIDE.md`](REPLAY_GUIDE.md), [`REPLAY_CORPUS.md`](REPLAY_CORPUS.md), [`REPLAY_MEMORY.md`](REPLAY_MEMORY.md) |
| Microsoft replay/stats API | [`AOE_MS_REPLAY_API.md`](AOE_MS_REPLAY_API.md) |
| `.dat` engine architecture | [`DAT_ENGINE.md`](DAT_ENGINE.md) |
| `.dat` authoring, particles | [`DAT_AUTHORING.md`](DAT_AUTHORING.md), [`DAT_RECIPE_EXAMPLE.json`](DAT_RECIPE_EXAMPLE.json) |
| `.dat` SQL export, priors, GoKu audit | [`DAT_SQL_EXPORT.md`](DAT_SQL_EXPORT.md), [`DAT_SEMANTIC_PRIORS_2026-09-22.md`](DAT_SEMANTIC_PRIORS_2026-09-22.md), [`GOKU_DAT_AUDIT.md`](GOKU_DAT_AUDIT.md) |
| XS authoring | [`XS_AUTHORING.md`](XS_AUTHORING.md), [`XS_AUTHORING_TIER1.md`](XS_AUTHORING_TIER1.md), [`XS_CHECK.md`](XS_CHECK.md) |
| XS sidecars and engine probes | [`SIDECAR.md`](SIDECAR.md), [`XS_SIDECAR_TECHNIQUE.md`](XS_SIDECAR_TECHNIQUE.md), [`ENGINE_HARNESS.md`](ENGINE_HARNESS.md) |
| engine-verified XS facts | [`XS_ENGINE_VERIFIED_2026-09.md`](XS_ENGINE_VERIFIED_2026-09.md), [`ENGINE_BUILD_SCOPE.md`](ENGINE_BUILD_SCOPE.md), [`ENGINE_UPDATE_185872.md`](ENGINE_UPDATE_185872.md) |
| Gaia text library | [`TEXT_API.md`](TEXT_API.md) |
| per-object custom stores | [`CUSTOM_STORE.md`](CUSTOM_STORE.md) |
| Castle Blood Automatic | [`CBA.md`](CBA.md) |

## Packaging and release

```sh
./kit verify .                                   # buildability, required docs, manifest drift
./kit portable-check . --profile public          # packaging and leak checks
./kit pack AoE2Kit.zip --profile public --sha-sidecar
```

`kit pack` profiles:

| profile | contents |
| --- | --- |
| `full` | everything tracked, plus the built binary |
| `handoff` | source, data, and root documents, without `docs/` |
| `sandbox` | `handoff` plus a static linux-amd64 binary, for containers without Go |
| `public` | default-deny: exactly the files listed in `release/public-files.json` |

`kit pack` verifies the kit before packing and reports the archive SHA-256;
`--sha-sidecar` writes it next to the zip. `kit portable-check` adds
profile-specific checks: buildability, manifest health, local-path and
personal-name leak detection, and refusal of executable payloads in public
archives.

Other project commands: `kit project inspect|snapshot|diff|lineage` map and
track a project folder before and after edits; `kit recipe list|show|export`
exposes starter scenario and DAT recipes; `kit docs lint` checks command
examples in Markdown against the live command catalog; `kit credits` prints
the attribution ledger behind `NOTICE.md`.

## Regenerating the reference

After a CLI change:

```sh
go build -o kit ./cmd/kit
go run ./cmd/apiref
```

This writes `docs/API_REFERENCE.md`, `docs/api_reference.json`,
`docs/api_schemas.json`, and `docs/PACKAGE_API.md`. The runtime probe executes
read-only commands against whatever fixtures you pass (`--scenario`,
`--replay`, `--dat`, `--xsdat`, `--folder`); commands without a fixture are
labeled rather than guessed. `--reuse-probe docs/api_reference.json` reuses
the previous probe results when only documentation changed, and
`--probe=false` skips execution entirely.
