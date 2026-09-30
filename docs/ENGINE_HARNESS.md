# Unattended engine-fact harness

`kit xs harness` generates a small scenario, inline runtime XS module, sidecar
schema, fact manifest, and patch recipe for an unattended engine run.

```text
kit xs harness --out-dir ./harness-run --batch safe --timestamp 123
```

For a single unattended capability pass, use the long probe:

```text
kit xs harness --out-dir ./harness-trig-long --batch trig-long --timestamp 123
```

Available batches (`--batch`):

| batch | what it probes |
| --- | --- |
| `safe` | conservative first pass (default) |
| `six-arm` | six independent rules, each a separate probe arm |
| `trig-long` | long single-scenario capability pass (below) |
| `attr-sweep` | object attribute reads across an attribute range |
| `long-v2` | attribute blocks, a local-technology block, the trig-long budget, and a trigger-driven chain with per-second rate rows |
| `long-v3` | DAT catalog versus engine: each unit type created, read, treated, and retired in one paced step, plus native trails, area combat, and local technology |

None of the batches includes deliberate crash tests. Generated XS is checked
with xs-check before the scenario is written; see `XS_CHECK.md`.

The `trig-long` batch runs 91 bounded facts in one scenario: trigonometry,
string operations, time and resource reads/writes, unit position and attribute
reads for the complete `167..221` window, runtime geometry, two known
retirement paths, and a local-technology write. It writes a `before` row before
every fact and stops cleanly with the same `424242` footer. It deliberately
excludes crash probes and parameterized `script_call`; an incomplete sidecar
identifies the last fact reached without asking the engine to execute an
intentionally invalid construct. Numeric results marked status `2` are
observations rather than pass/fail assertions.

The first batch is intentionally conservative. It covers string equality, array
and vector round trips, fractional Gaia placement, existence before and after
retirement, and a bounded loop. Known silent-crash candidates are not mixed into
this batch. Future hazardous batches must be separate scenarios.

The generated harness keeps the two-slot structural baseline but deactivates P2;
only P1 is an active actor. This prevents AI actions or resource changes from
polluting unattended fact observations.

Each fact writes a `before` marker using append-open-write-close before the
operation, followed by a `result` marker. The fact ID is written into every
marker, so a truncated sidecar identifies the operation that was in flight. A
completed run ends with integrity value `424242`.

The generated XS also advances P1's food resource from a `100000` base once per
poll. Food is used because it is a replay-visible, engine-verified state
carrier; the unused resource 220 must not be used as a carrier because writes
to it did not appear in the replay sync stream.

Decode a harvested sidecar with:

```text
kit xsdat decode run.xsdat --schema engine-harness --text
```

For the replay-visible food carrier, read the harness with the declared base:

```text
kit replay xs-telemetry run.aoe2record --food-base=100000 --text
```

The decoder reports measured values separately from manifest expectations. A
missing footer is not treated as a pass; when the last complete row is a
`before` marker, the report calls out that fact as an early-termination candidate.
The harness is a probe generator and structural decoder, not an engine/compiler
verdict. Engine verification still requires a replay from the generated scenario.
