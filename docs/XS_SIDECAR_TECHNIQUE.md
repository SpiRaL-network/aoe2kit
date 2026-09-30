# The XS Sidecar Technique — assert engine state without playing back

A self-contained guide to the debug loop AoE2Kit uses for engine observability:
your scenario's XS writes phase-stamped, typed state rows into a `.xsdat`
sidecar file while the game runs; the Kit decodes and asserts that file
offline. It replaces the observer-AI / invisible-chat workaround entirely —
typed values instead of parsed chat, no spam, no observer slot, and it works
in tests you never watch.

Everything marked **engine-verified** below was proven with live DE runs
against generated fixtures (failures included — that's how the flush semantics
were found).

## 1. The loop

1. Instrument: add a small XS writer (pattern below) to the scenario; call its
   phase functions from triggers (`script_call`) at the moments you care about.
2. Run the scenario (a real game, a solo test, an unattended AI game).
3. Harvest: the sidecar appears in the DE profile directory as
   `<profile>\<scenario_base_name>.xsdat`.
4. Decode: `kit xsdat decode <file.xsdat>` prints every typed row.
5. Assert: `kit xsdat decode <file.xsdat> --ledger expected.json` checks the
   run against a declared expectation ledger and emits pass/fail per row —
   your engine-state test harness.

## 2. The writer pattern (engine-verified)

Minimal shape, distilled from calibration fixtures:

```c
bool sc_file_open = false;
int  sc_rows = 0;

bool SC_open() {
    if (sc_file_open == false) {
        sc_file_open = xsCreateFile(false);   // false = truncate/new
        if ((sc_file_open == true) && (sc_rows == 0)) {
            xsWriteString("MY_SIDECAR_HEADER");  // magic + version header
            xsWriteInt(1);
        }
    }
    return(sc_file_open);
}

void SC_write_row(int phase_id = -1) {
    if (SC_open() == false) {
        return;
    }
    xsWriteString("phase");
    xsWriteInt(phase_id);                       // your semantic label
    xsWriteInt(xsGetGameTime());                // timestamp every row
    xsWriteFloat(xsPlayerAttribute(1, 20));     // e.g. attr 20 = kills
    xsWriteFloat(xsPlayerAttribute(1, 154));    // attr 154 = deaths
    xsWriteInt(xsGetObjectCount(1, 82));        // e.g. castles alive
    sc_rows = sc_rows + 1;
}

void SC_close() {
    SC_write_row(900);
    xsWriteString("end");
    xsWriteInt(sc_rows);                        // row count = integrity check
    xsCloseFile();
    sc_file_open = false;
}
```

Call `SC_write_row` from triggers at your instrumented moments (a feed event,
an AFK check, a spawn tick, a threshold crossing) — each row arrives typed and
timestamped. For feeder/AFK detection this is exactly "data and timestamp":
write the player id, the counter, and `xsGetGameTime()` at the moment your
detection condition fires, then read the file after the game.

## 3. File semantics you must know (engine-verified, the hard way)

- **Location:** the file lands at `<DE profile dir>\<scenario>.xsdat` (the
  scenario's base name; you do not choose the path).
- **Flush happens at scenario unload.** Rows are buffered; do not expect the
  file on disk mid-game. `xsCloseFile()` + game end makes it durable.
- **`xsCreateFile(false)` truncates** any prior file of the same name;
  `xsCreateFile(true)` appends. Header-once logic matters on reruns.
- **DE holds an exclusive lock on the file while the game runs.** Harvest
  after exit, or copy with a share-mode-tolerant reader.
- **Cross-scenario read works** (`xsOpenFile` with the bare name from another
  scenario, even in a later session) — this is real campaign persistence, and
  it is engine-verified including string+int+float typed rows surviving.
- **Multiplayer law:** File I/O is per-client. In MP, all clients execute the
  same triggers, so same-party writes are mirrored identically on each
  machine — natively sync-safe. But any file a scenario READS must exist with
  identical content on every client, or you desync. Same-party campaigns are
  safe by construction; mixed parties need a pre-lobby file sync or a
  re-declare step driven through synced inputs.

## 4. The decoder

```
kit xsdat decode <file.xsdat>                    # print typed rows
kit xsdat decode <file.xsdat> --ledger exp.json  # assert against expectations
```

The `.xsdat` container is a simple typed stream (strings, ints, floats in
write order); the decoder reconstructs rows without any game client. The
ledger mode turns a live run into a regression test: declare the rows a
correct run must produce, run unattended, decode, read pass/fail.

## 5. Why this beats the alternatives

| Workaround | Cost | Sidecar |
| --- | --- | --- |
| Observer AI harvesting state | needs a slot, AI plumbing | none |
| Invisible chat as debug log | string parsing, spam, chat is NOT retained in recordings | typed rows, durable file |
| Reading recordings | command stream only; trigger/XS state largely invisible | direct state export at chosen moments |

Recordings do not carry trigger chat, trigger research, or custom resource
state; this was verified repeatedly with controlled fixtures. The sidecar is the channel that makes
scenario-internal state observable at all. Instrumented scenarios get ground
truth on tap; that ground truth then calibrates what CAN be inferred from
plain recordings of uninstrumented games.

## 6. Generators that use this pattern

- `kit xs harness` generates unattended engine-fact probes with a sidecar
  schema and fact manifest; see `ENGINE_HARNESS.md`.
- `kit ci init` scaffolds a project-local contract plus a small `a2k_debug.xs`
  sidecar include; `kit ci check` evaluates it after a playtest.
- `kit campaign generate` writes cross-scenario persistence writer/reader
  modules; see `CAMPAIGN_SPEC_EXAMPLE.json`.
- `kit rpg custom-store` embeds a bounded purchase sidecar; see
  `CUSTOM_STORE.md`.
