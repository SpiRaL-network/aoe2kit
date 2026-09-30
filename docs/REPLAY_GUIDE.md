# Replay Guide

`kit replay` reads AoE2DE recordings (`.aoe2record`, or replay `.zip` files as
downloaded) without the game. This guide explains what each family of commands
answers and how far its claims go. Flags and output keys for every command are
in [`API_REFERENCE.md`](API_REFERENCE.md).

What a recording contains: a compressed header (the lobby, players, map, the
embedded scenario and its trigger graph, and each player's initial state) and a
body of player commands plus a periodic sync record. It does **not** contain
trigger execution, trigger-generated chat, XS state, per-kill events, or
postgame statistics. Everything below is read from what is there; anything
inferred says so in its label.

## Orientation

```sh
./kit replay summary game.aoe2record --text     # players, civs, teams, result, map
./kit replay info game.aoe2record --text        # compact metadata
./kit replay chat game.aoe2record --text        # lobby + in-game transcript
./kit replay health game.aoe2record --text      # triage for lagged/frozen/crashed games
```

- `summary` is the one-object rollup: record hash, header versions, scenario,
  map hash, active data set, lobby settings (game type, speed, population,
  treaty, and more), duration and result, and players with civ, color, team,
  profile id, winner/resign, and rating when a leaderboard block is present.
  `--tiles` adds the full terrain grid.
- `info` reports the same routine metadata compactly. `--full` emits the raw
  parsed replay object, which can be very large on custom scenarios.
- `chat` merges in-game chat with lobby chat recorded in the header, removes
  echoes, and tags chat that DE injects from a previous session as
  `source=backlog`. Backlog rows stay visible in `chat` and `events` but are
  excluded from feedback counts.
- `health` is a low-memory single pass for problem recordings: where the
  recording ends and whether its framing is clean, per-window action/chat/flare
  counts, object-count trend, and chat mentioning problems. Clean framing is
  not a claim about how the game exited, and simulation time cannot measure
  client freezes. Unzip archives first.

## Identity

Replays are identified by the embedded scenario, not by file names.

- `kit identify <file>` fingerprints scenarios and replays, trigger-graph hash
  first (`trigger_graph`), then map dimensions plus terrain and elevation
  (`fallback:terrain`). The terrain fallback is stable for recognition but does
  not prove trigger integrity.
- `known_scenarios.json` and `known_ais.json` map fingerprints to names and
  descriptions. Descriptions are written as orientation text for whoever
  matches the hash. Grow them with `kit identify --register` and
  `kit collect <folder>` (add `--dry-run` to preview).
- AI identity comes from the header's AI file list: grouped AI name, file
  names, source mod, and a stable signature. An empty list means the default AI.
  Recordings do not contain custom AI script content; fingerprint `.ai`/`.per`
  files with `kit ai fingerprint`.
- Active data set: `status=vanilla` means no data mod was named, `modded`
  lists the selected data sets, `unknown` means the field was not parsed. This
  proves which data set loaded, not how anything rendered.
- `replay graph`, `replay triggers`, and `replay trigger-neighborhood` read the
  embedded trigger graph: search by text, effect/condition, player, variable,
  or unit, and follow activate/deactivate edges. This is authoring structure,
  not proof that any trigger fired.

## Events and actions

- `events` is the shared event core: chat, taunts, flares, resigns, telemetry
  markers, and result inference from resigns. `--all` adds sync and camera rows
  and low-level actions; `--raw` adds payload hex.
- `feedback` is the human-feedback view of `events`: chat, taunts (with labels
  for taunts 1–105), and flares with coordinates.
- `actions` decodes the command stream: move, order, patrol, attack-move,
  attack-ground, build, gather point, delete, gate, flare, resign, queue,
  research, and more. Unknown action ids are preserved raw;
  `--unknown-only --samples N` aggregates them, and `replay unknowns <folder>`
  mines them across a corpus.
- `player-events` is the canonical filterable event table (player, type,
  action id, phase, time range). Action id `0` is a valid filter.
- `player-profile` summarizes per-player activity: action counts, APM and APM
  windows, dead gaps, command vocabulary, coordinate footprint, optional named
  regions, and chat/taunt/flare counts. It does not claim skill, intent, or
  economy state.
- `camera` (alias `viewlock`) is the recording player's own camera stream,
  which shows attention, not every player's view.
- `--objects` on `events`, `actions`, and `player-events` names target objects
  (for example "P6's gate") instead of printing numeric ids.

## The sync record

The body carries a periodic sync record with an 11-word per-player matrix.
`replay sync` surfaces it with these decoded meanings:

| word | meaning |
| --- | --- |
| trailer | current world time in ms |
| 1 | resource stockpile |
| 2 | sum of living objects' unit-type ids |
| 3 | sum of objects' engine `state` field (living objects mostly 2, flares 3, corpses/rubble 5) |
| 4 | sum of objects' `carry` field (villager load, monk faith, flare lifetime, corpse decay) |
| 6 | living object count |
| 7 | position sum, `(x+y)*100` |
| 8 | player number |
| 10 | sum of living objects' instance ids |
| 9 | undecoded; best hypothesis is a per-player state digest |

Each meaning was established with controlled fixtures whose ground truth was
known (see [`SIDECAR.md`](SIDECAR.md)).

- `player-series` turns the matrix into per-player state curves.
- `checksum-probe` matches expected fixture pulses (object-count, unit-type,
  instance-id, state, carry, or digest deltas). A match means the sums moved as
  expected, not that the engine reported a death or kill.
- `checksum-phase` summarizes one word's value alphabet and successor
  relation, for probing undecoded words.
- `combat` reports windows where a player's living object count drops, joined
  to nearby targeted commands and contested objects. It is pressure, not kill
  attribution; production inside an interval can mask losses.
- `deaths` narrows that to intervals whose unit-type sum matches a known
  live-unit-to-corpse replacement (`data/corpse_units.json`). Engine death,
  trigger kills, and deletes can produce the same shape.
- `sync-log <p0-sync.txt>` parses the text log DE writes when a game goes out
  of sync: turn and world-time headers, per-player object dumps, checksum-word
  candidates, and named RNG streams. `--replay` compares it with the
  recording's sync samples.

## Objects and initial state

- `objects` indexes initial object records from the header: owner, object id,
  unit id, record class, position, and how often commands reference each one.
  It decodes each record's fixed base and, where guards pass, the following
  action, combat, building, production-queue, and position blocks. Some record
  tails remain opaque; use the confidence fields.
- `object-state` presents those records as readable state cards (HP, state,
  position, queue capacity, linked ids).
- `object-shapes` groups records by class and length as a map for further
  decoding.
- `spawns` lists create-object recipes from the embedded trigger graph;
  `lifecycle` joins first-seen runtime object ids to nearby spawn points as
  candidates.

## Game data inside recordings

Each player's effective game-data table is embedded in the header.

- `effective-data` reports its hash, compares it with a vanilla baseline, and
  decodes the unit-availability array; with `--dat` each slot is joined to the
  DAT unit, and `--tree` prints it as a tech-tree-style list.
- `effective-units` reads per-civ effective hit points and compares them with
  a supplied DAT. Attack, armor, cost, and speed are not decoded.
- `datamod-check` gives a direct "was this game modded" verdict, strongest
  with `--baseline-replay` from a known-vanilla game.

## Byte accounting

Kit accounts for every byte of a recording.

- `coverage` reports each region as `decoded` or `bounded_opaque`; the two must
  sum to the source size, so parser gaps cannot hide. `--fail-on-opaque` makes
  it a hard gate.
- `opaque-spans` profiles unknown spans (entropy, byte densities, neighbors,
  shape hints). `frontier` buckets them into hypotheses and finds repeated
  payloads. `corpus` and `opaque-clusters` do the same across a folder; see
  [`REPLAY_CORPUS.md`](REPLAY_CORPUS.md).
- `diff-state` compares two recordings' headers, bodies, and final sync
  matrix, attaching changed bytes to coverage regions; `scan-value` finds a
  planted number or string. Together they support controlled experiments: plant
  a known value, find where it lands.
- `header-anchors` reports graduated header structures such as caption records.

## Postgame

`postgame` decodes the tail block when present. Across ranked and scenario
games it has contained metadata only (world time and leaderboard blocks):
DE recomputes scoreboard statistics for display and does not serialize them.
`postgame-corpus` checks a folder for variants.

## Stories and playtest intake

- `story` builds a designer-facing brief: identity, players, feedback, telemetry,
  result, a `claims` ledger, and a `missing` list of what Kit cannot know from
  the recording. `--brief` shortens it.
- `playtest` is the author's single-replay report: feedback moments with the
  sync state around each one. It shows correlation near feedback; it cannot name
  the trigger, what was on screen, or intent.
- `inbox <folder>` is folder-level intake: duplicate detection by record hash,
  scenario groups, issue cards, and a line per replay. `issues` ranks feedback
  into issue cards.
- Replay context JSON ([`REPLAY_CONTEXT_EXAMPLE.json`](REPLAY_CONTEXT_EXAMPLE.json))
  names your map's regions and telemetry prefixes. Region names are enrichment
  from the context file, not facts from the recording.
- `telemetry` validates telemetry markers against a schema
  ([`TELEMETRY_SCHEMA_EXAMPLE.json`](TELEMETRY_SCHEMA_EXAMPLE.json)).

## Verification loops

For scenarios you instrument yourself:

- `verify-run <replay> --contract contract.json` checks a pre-registered
  contract: scenario identity, data set, required or forbidden telemetry and
  chat markers, result, and render expectations. Render expectations always
  report `unknown`, and overall `ok` requires zero failed and zero unknown
  claims. See [`VERIFY_RUN_CONTRACT_EXAMPLE.json`](VERIFY_RUN_CONTRACT_EXAMPLE.json).
- `kit ci init` scaffolds a project-local contract and a small XS debug
  include; `kit ci check` runs them after a playtest.
- `carrier --ledger` reads self-reporting probes that write known values into a
  sync word.
- `sidecar-sync --xsdat run.xsdat` correlates an XS sidecar with the sync
  matrix and reports whether decoded words agree with the sidecar's known
  values. Named schemas (`rtv12`, `rtv13`, `rtv14`, `rtv141`, `rtv142`) decode
  Kit's built-in diagnostic sidecars; `--ledger` handles your own. See
  [`SIDECAR.md`](SIDECAR.md).
- `xs-telemetry` reads the food-carrier packing used by engine-harness probes
  (`--food-base`).

## Acquisition

`replay fetch <gameId> --profile <profileId>` downloads from Microsoft's public
replay endpoint, naming files `AgeIIDE_Replay_<gameId>_p<profileId>.zip` so
different players' recordings of one game do not overwrite each other. Existing
files are cache hits unless `--force`. `--all-povs` fetches every player's
recording when profile ids can be read from a roster. `kit player stats`
wraps the career-stats endpoint and labels its output as career aggregates, not
per-match facts. See [`AOE_MS_REPLAY_API.md`](AOE_MS_REPLAY_API.md).

## Memory

Several commands share a content-addressed disk cache and stream large results
instead of holding them in RAM. Recordings from long, large games are still
heavy: run one replay command at a time. Details and limits are in
[`REPLAY_MEMORY.md`](REPLAY_MEMORY.md).

## Limits

- No per-kill attribution: recordings carry no death or kill-credit events.
- Some object record tails and sync word 9 remain undecoded.
- Effective technology arrays are not yet mapped to DAT tech ids.
- Full deterministic trigger-section walking in recordings is not complete;
  some recordings identify by `fallback:terrain`.
