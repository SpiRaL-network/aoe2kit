# Gaia Text API

AoE2Kit ships a standalone `examples/text/gaia_text.xs` library for scenario
authors who want to turn text into Gaia artwork. The XS file is the product:
it can be pasted into an AoE2DE Script Call trigger from the editor, Python
scenario tools, or another authoring workflow.

Kit is optional. Use it when you want to compile a long document, define a
custom style/material set, calculate cost before drawing, embed the script, or
verify a scenario. A user does not need Go, Kit, a data mod, or an external
include file just to render a line.

## Prerequisites

- Age of Empires II: Definitive Edition with the current XS runtime.
- A scenario with a Script Call carrier trigger and a visible map position.
- A Gaia material available in the scenario's effective data set.
- No data mod is required by the library.

Paste `examples/text/gaia_text.xs` into the carrier, then add a runtime entry
point. This example renders a live timer and runtime-updated score slot; no
text build step or precompiled document is involved:

```xs
int lastSecond = -1;

void main() {
  fontInit();
  fontLoadTables();
  drawSentence(0, 86.0, 50.0, "LIVE SCORE", 939, 0.0875);
  drawNumber(1, 86.0, 54.0, 0, 4, 939, 0.0875);
}

rule updateScore {
  active
  minInterval 1
  maxInterval 1
  {
    int now = xsGetGameTime();
    if (now != lastSecond) {
      lastSecond = now;
      setNumber(1, now, 4);
    }
  }
}
```

`examples/text/gaia_text_minimal.xs` contains the library and a live timer entry
point already. `examples/text/gaia_text_minimal.recipe.json` shows the optional
Kit recipe form.

## Calls

- `drawSentence(slot, x, y, literal, material, scale)` draws runtime text.
- `drawNumber(slot, x, y, value, minDigits, material, scale)` draws runtime
  scores, timers, and counters.
- `setSentence` and `setNumber` update existing runtime slots incrementally.
- `drawMessage(slot, x, y, messageID, material, scale)` is only for authored
  mixed case supplied through a scenario string table.
- `drawStrokeMessageTable(...)` and `drawStrokeNumber(...)` select the large
  outline layer.
- `pathReset`, `pathMove`, `pathLine`, `pathQuadratic`, and `pathCubic` build a
  reusable cell-space path; `pathStroke` samples and renders it with the same
  Gaia lifecycle and fractional placement rules as the font renderer.
- `measureText(literal)` returns bitmap width in cell pitches, or `-1` when
  unsupported/overlong.
- `clearSlot` retires an existing slot.

Call `fontInit()` and `fontLoadTables()` once before drawing. Slots own their
Gaia objects and updates retain unchanged cells. Fire materials retire through
hit points. Non-fire materials require the configured kill-zone sweep contract;
the library does not silently delete or leak retired objects.

## Text Size And Cost

Bitmap is the small-text renderer. Layered stroke is for titles, counters,
monuments, and other large text. It uses a high-DPI foreground and sparse
decorative background, with density derived from each material's sprite
footprint. Do not copy a foreground sampling value to a different material.

Runtime literals are case-folded by the DE XS engine. Use string-table messages
only when authored case matters. A single large text draw creates one Gaia object per cell or
outline point. The current stroke ceiling is 256 points per glyph and 10,000
tracked cells. A long document should be compiled and presented as bounded
windows, not drawn as one unbounded object set.

The glyph tables are baked into the library once. Text, numbers, scores, and
timers remain runtime data and can change without regenerating XS. The verified
clock touched 37 cells on a minute rollover out of roughly 150 live cells.

The optional `pkg/text` planner in Kit compiles arbitrary-length source offline,
computes the exact cost of a bounded window, and refuses a window that cannot
fit its declared budget. It is a planning primitive, not a hidden runtime
scheduler: authors still choose how to advance a window and which XS draw call
to use. Static, crawl, typewriter, page, and dialogue are useful presentation
intent names for an authoring layer, but only the low-level draw/update calls
listed above are currently provided by the standalone library. This keeps the
human/AI authoring unit as a document plus presentation rather than a pile of
unit placements without claiming that the engine behavior is already
automated.

The path API is the general drawing layer beneath future glyphs, diagrams, and
ornamental strokes. Quadratic and cubic curves are sampled at runtime; this is
programmable geometry rendered as Gaia objects, not a claim that DE exposes a
native line or vector primitive. Its source and structure are verified, but
the exact visual density and curve behavior still require a live engine run.

## Verification Boundary

Kit can structurally parse, embed, patch, lint, and cost-check the script. The
game/editor is still the oracle for XS compilation, material availability,
rendering, passability, and runtime behavior. Treat Kit reports as
structure-verified until a live engine run confirms the result.
