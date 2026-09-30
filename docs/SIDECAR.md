# The XS Sidecar

## What problem it solves

AoE2DE is a black box at runtime. A recording gives you player commands (move,
build, attack) plus a periodic sync record. It does not give you trigger
state, XS variables, custom resources, or anything the scenario computed:
trigger-generated chat never enters a recording, and the postgame block is
metadata only, not serialized statistics.

Without a sidecar, a scenario author debugging a large trigger machine has two
options: print state as in-game chat and read it off the screen, or run an
observer AI in a player slot that harvests state and emits it as invisible
chat. Both need someone to watch a game, and both produce strings to parse
rather than values.

The sidecar removes the watcher. The scenario writes its own state to a file,
in typed binary, timestamped, at moments you choose.

## How it works

**Write side.** XS exposes a small file API: `xsCreateFile(append_bool)`,
`xsWriteString` / `xsWriteInt` / `xsWriteFloat` / `xsWriteVector`,
`xsCloseFile`, `xsOpenFile`. Call these from functions that triggers invoke via
`script_call`. There is no path argument: DE names the file after the scenario
stem, `<DE profile dir>\<scenario name>.xsdat`.

**File format.** The container has no type tags. It is a bare concatenation of
values in write order:

- `string`: u32 little-endian byte length, then the raw bytes
- `int` / `uint`: 4 bytes little-endian
- `float`: 4 bytes IEEE-754 little-endian
- `vector`: three consecutive floats, 12 bytes

A `.xsdat` is a headerless typed stream in which the writer's call order is the
schema. Nothing in the file distinguishes `1092616192` from `10.0`; they are
the same bytes.

**Read side.** `kit xsdat decode` runs in one of two modes. Given the write
order (`--types string,int,float,...`, a `--ledger`, or a named `--schema`), it
walks the stream deterministically and emits typed rows. Without one, it falls
back to a labeled heuristic: a plausible length-prefixed printable run is read
as a string; otherwise 4 bytes are reported as both int and float candidates,
flagged `heuristic: true`. It never silently guesses.

A decode is `OK` only if there were no errors and `RemainingBytes == 0`. The
stream must be consumed exactly, so a schema that is wrong by one field leaves
a tail, and the tail is the alarm.

**Assertion layer.** With `--ledger expected.json`, you declare what a correct
run must produce. The decoder compares position by position, emits per-row
`passed` / `failed` / `unknown` plus a summary, and exits non-zero on mismatch.
Pre-register the expectation, run the game unattended, read a verdict.

**Correlation layer.** `kit replay sidecar-sync` decodes the sidecar's phase
rows, then for each phase finds the first sync sample at or after that
timestamp for the chosen player, within a window (default 20 s), and compares
the sidecar's known-true values with the replay's sync words (stockpile,
unit-type sum, object count). When no sample lands in the window it reports
`no_checksum_sample_in_phase_window` rather than inventing a match. This is how
an unknown replay field gets named: what was true inside the engine at time T,
next to what the replay's opaque word said at time T.

## File semantics

- Rows are buffered and flushed at scenario unload. The file does not exist
  mid-game, so there is no live streaming.
- DE holds an exclusive lock while the game runs; harvesting needs a
  share-mode-tolerant read or a wait until exit.
- `xsCreateFile(false)` truncates and `xsCreateFile(true)` appends, so
  header-once logic matters across reruns.
- A run that ends before writing its footer leaves a valid partial file; the
  schemas accept partial runs.
- A single sidecar stops at `1,048,575` bytes (`2^20 - 1`); see
  `XS_ENGINE_VERIFIED_2026-09.md`.
- Opening the bare stem of another scenario's file works in a later session:
  this is cross-scenario persistence, the capability behind
  `kit campaign generate`.
- Multiplayer: file I/O is per client. Every client runs the same triggers, so
  writes mirror identically and are sync-safe, but any file you read must be
  byte-identical on every client or the game desyncs.

## What it has established

- **Sync-word meanings.** A probe wrote food values `900000`, `900100`, and
  `900200` from XS; sync word 1 carried them exactly, which named it the
  resource stockpile. Trigger-only calibrations named further words the same
  way. Every decoded word traces back to a controlled fixture with known
  ground truth.
- **Negative results.** In a castle-kill run, the sidecar showed a player's
  credited kills rising 0→1→2→3 while no sync word mirrored it; that is why
  Kit infers kills from anchors rather than searching the sync stream for a
  kill counter. The same method showed that `xsChatData` does not serialize
  and that resource 220 does not move sync word 1.
- **Cross-scenario persistence**, verified with a writer/reader scenario pair.

## Uses

It is printf debugging plus unit tests for an engine that offers neither:
regression-test a scenario across versions, measure balance from real runs,
verify that a mod actually loaded, see which branch of a quest players take,
or carry state between maps of a campaign. For feed or AFK detection, write
`player_id`, a counter, and `xsGetGameTime()` when the detector fires; that
replaces an observer-AI slot and invisible-chat parsing with typed values.
`kit ci init` scaffolds a project-local contract plus a small `a2k_debug.xs`
include, and `kit ci check` evaluates it after a playtest.

**Limit:** it only works on scenarios you instrumented. It says nothing about
an uninstrumented game, such as a random ladder replay. The two halves
complement each other: the sidecar gives ground truth on fixtures, and that
ground truth calibrates what can be inferred from ordinary recordings.

See also [`XS_SIDECAR_TECHNIQUE.md`](XS_SIDECAR_TECHNIQUE.md) and
[`ENGINE_HARNESS.md`](ENGINE_HARNESS.md).
