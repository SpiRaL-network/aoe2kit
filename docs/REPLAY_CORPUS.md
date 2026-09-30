# Replay Corpus Commands

AoE2Kit treats a replay folder as a corpus, not as a pile of one-off files.
The goal is to make every new recording advance either verification, storytelling,
or byte-structure transparency.

## `kit replay corpus`

```sh
./kit replay corpus path/to/replay-folder --text
./kit replay corpus path/to/replay-folder --unknown-samples 4
./kit replay corpus path/to/replay-folder --flat --no-zip
```

The corpus report scans `.aoe2record` files and replay `.zip` files recursively
by default. It emits one row per replay plus an aggregate summary:

- scenario vs non-scenario replay count
- trigger-graph identity or fallback fingerprint tier
- players, human slots, AI slots, and names when present
- data-set identity status
- replay duration and action/event counts
- body operation histogram
- decoded/opaque coverage totals and min/average decode percentages
- unknown action ids grouped across the folder
- postgame tail shape inventory

Honesty label:

```text
structure_verified_not_engine_verified
```

This command proves what AoE2Kit can parse from bytes. It does not claim the
AoE2 engine has loaded, rendered, or accepted any artifact.

## `kit replay opaque-clusters`

```sh
./kit replay opaque-clusters path/to/replay-folder --text
./kit replay opaque-clusters path/to/replay-folder --space inflated_header --min-bytes 256 --limit 40
./kit replay opaque-clusters path/to/replay-folder --samples 4
```

The opaque-cluster report runs the coverage/opaque-span profiler across the
corpus and groups still-opaque spans by structural context:

- replay space (`inflated_header` or `body`)
- normalized span name
- normalized neighboring decoded regions
- byte-size bucket
- shape hints

Each cluster reports count, file coverage, total bytes, size range, average
entropy, zero/`0xff`/printable density, hints, and a small set of hex samples.

Honesty label:

```text
structure_verified_opaque_bytes_profiled_not_semantically_decoded
```

This is a decode workbench. A cluster is a bounded unknown with a profile, not a
semantic claim.

## Why This Exists

Single-replay archaeology is too easy to overfit. Corpus commands make the next
decode target obvious:

- if a span appears in every replay with the same size and low entropy, it smells
  like static header/save metadata
- if it varies by player count, it smells like per-player state
- if it appears only in scenario games, it may relate to embedded scenario state
- if it changes with duration or command volume, it may be runtime state
- if a new replay introduces a new action id, the unknown-action aggregate makes
  it visible immediately

These commands are project-neutral. Scenario-specific interpretations (for
example Castle Blood Automatic in `pkg/cba`) consume this generic corpus
evidence in their own packages rather than folding scenario-specific meaning
into `pkg/replay`.
