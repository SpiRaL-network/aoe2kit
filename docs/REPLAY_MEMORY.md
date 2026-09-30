# Replay memory contract

The migrated path hashes the recording with a fixed buffer, streams its inflated
header to disk, and parses body framing once into `ops.jsonl`. Cache root:
`~/.cache/aoe2kit/replay-cache/<record-sha256>/`, overridden by
`AOE2KIT_REPLAY_CACHE`. ZIP inputs use the extracted recording's hash. Completed
manifests are published atomically; size-invalid entries are rebuilt and kept
under `.invalid-*` for diagnosis. Caches contain replay data, including chat;
do not publish them as anonymous artifacts. Cache cleanup is manual for now.

## Migrated paths

- `info`/`summary`, chat/events, issues, postgame, camera, deaths.
- Full JSON sync output streams `sync.jsonl`, `raw_words.jsonl`, and
  `lifecycle.jsonl` from disk. `--limit` still limits displayed sync events, not
  lifecycle analysis. The slice-returning Go API has a 32 MiB result budget.
- Graph/trigger CLI inspection uses bounded metadata. For headers above 8 MiB,
  graphs are explicitly **bounded candidates**, not complete embedded scenarios.
- Corpus processes one record at a time and shares cached metadata/body ops.
  Its detailed coverage analyzer is still a bounded legacy path and can report
  coverage unavailable. Corpus still retains compact per-file report rows.

`HeaderReader()` gives callers a read-only file handle (caller must close).
Legacy `Open`, `Parse`, `HeaderBytes`, and snapshot analysis are not yet migrated:
their whole-record/header materialization now refuses inputs over 32 MiB.
This guard is not a claim that every legacy decoder satisfies the RSS target.
Metadata inspects the first/last 8 MiB, not the entire initial object snapshot.
Unsupported framing reports partial results with warnings, never an inferred
death or kill attribution. The old unframed chat-JSON fallback is not used by
the new action-stream reader.

## Limits and measurement

Target: under 300 MiB peak RSS for the migrated 10 MB workload. Operations are
bounded to 16 MiB, retained event reports to an estimated 32 MiB, and inflated
headers to 2 GiB **on disk**. Oversized reports fail explicitly, not truncate.
Camera/death reports can use `--limit`. Runtime memory limits are soft GC
targets, not safety guarantees. Run one heavy command at a time.

`kit replay info file.aoe2record --mem-report` leaves ordinary JSON on stdout,
and emits sampled peak Go heap/System bytes, Linux process high-water RSS,
target RSS, and worker count on stderr. Runtime samples every 10 ms can miss
short allocation peaks; Linux VmHWM is the process high-water mark. Corpus
prints `workers: 1` on stderr and does not fan out.

Measured serially on a large community recording (116 MB inflated header):

| command | before RSS KiB / elapsed | after RSS KiB / elapsed |
| --- | --- | --- |
| info | 2,895,812 / timeout 30.33s | 36,484 / 3.21s cold |
| issues | 2,911,060 / timeout 30.23s | 20,244 / 0.50s warm |
| chat | 2,698,736 / timeout 30.43s | 26,552 / 0.60s warm |
| sync JSON | 13,588 / guarded failure | 19,148 / 0.70s success |

The regression test creates a 10 MB action body plus a 128 MiB inflated header
in a subprocess and checks peak RSS below 300 MiB (Linux), and sampled heap on
other systems. First measured RSS was 77,975,552 bytes. It uses no private fixture.
Cache reuse, ZIP/plain equivalence, corrupt-size rebuild, raw event parity, and
streamed-vs-slice sync JSON also have tests.

Remaining migration: structural ReaderAt header parsers (not bounded-window
search), complete coverage/initial objects/effective data, large story/profile
consumers, bounded per-file corpus output, and cache eviction. Do not describe
the entire replay subsystem as streaming until those paths are migrated.
