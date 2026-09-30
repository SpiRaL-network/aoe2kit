# Castle Blood Automatic

`kit cba` (and `cmd/cba`, `pkg/cba`) interprets replays of the Castle Blood
Automatic (CBA) scenario. It is a domain layer: it consumes the generic facts
from `pkg/replay` and never redefines them. If you build interpretation for
your own scenario, this is the seam to copy.

Everything here is labeled with how strongly it is claimed. CBA mechanics such
as razes and kills are inferred from commands and sync-state movement,
because recordings contain no raze or kill events.

## Replay analysis

```sh
./kit cba replay progression game.aoe2record --text
./kit cba replay razes game.aoe2record --text
./kit cba replay perf game.aoe2record --text
./kit cba replay doctrine game.aoe2record --text
```

- `replay progression` derives production and research progression from queue
  and research commands: first production, high-tier production, and early
  unit production. It does not claim direct age state.
- `replay razes` correlates pressure on enemy buildings and gates with the
  later villager payoff CBA grants for razes: pressure order, payoff order, and
  set plays where several players pressure one target. Label:
  `behavior_inferred_raze_candidate_not_engine_raze_event`. `replay
  raze-pressure` reports the pressure side alone.
- `replay perf` emits one row per player plus timestamped events: own and team
  first villager, villagers, razes, first raze, and production. `units_lost`
  and `units_produced` are own-side sync-state removals and additions, not
  confirmed deaths.
- `replay doctrine` overlays matchup doctrine on these primitives: civ duty,
  phase-boundary evidence, pressure candidates, and production and combat
  signals.
- `replay phases` detects phase boundaries from exact anchors plus sync-state
  evidence.

## Scenario data from the trigger graph

```sh
./kit cba trigger-razes game.aoe2record --dat empires2_x2_p1.dat --text
./kit cba trigger-spawns game.aoe2record --dat empires2_x2_p1.dat --text
./kit cba balance --text
```

- `trigger-razes` reads the replay's embedded trigger graph and maps each
  civ's start-technology activators to its razes-per-villager reward rungs,
  resolving technology ids through a supplied DAT.
- `trigger-spawns` maps the same activators to spawn-wave parameters: seconds
  between waves (from the spawner display messages) and units per wave (from
  the civ-gated spawn conditions). Technology ids that map to more than one
  value are reported as conflicts rather than guessed.
- `balance` is a source-labeled, intentionally partial table of version and
  civ parameters (razes-to-villager, castle and imperial kill thresholds, unit
  counts, spawn seconds) mined from the scenario's published changelogs and
  trigger-graph evidence. Missing civs are unknown, not defaulted.
- `sidechannels` and `phase-facts` decode CBA's accounting side channels and
  per-civ phase thresholds.

## Ladder

`pkg/cba` also provides the pieces of a replay-based ladder:

- `cba registry` identifies CBA replays in files or folders.
- `cba ingest --corpus games.jsonl` extracts per-player performance rows into
  a JSONL corpus keyed by stable game ids (Microsoft match ids when
  available). Workers are sized from available memory and reported.
- `cba archive --queue` downloads queued matches politely.
- `cba axes` computes multi-axis contribution scores; `cba export` writes the
  corpus as JSON or SQL for a ladder site.

## Scenario files

CBA scenario files are DE 1.57, which Kit does not read; re-save them in the DE
editor to convert them. For ladder integrity, prefer the replay: `kit replay
info` parses the scenario graph embedded in the actual submitted game.
