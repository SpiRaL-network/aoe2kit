# Gaia Text

`gaia_text.xs` is the standalone, no-data-mod Gaia-art text library for
Age of Empires II: Definitive Edition. It is plain XS: no Go toolchain, Kit
binary, include path, or external data file is required to use it.

## Minimal Use

1. Create or open a scenario with one active player and a visible Gaia carrier
   trigger.
2. Paste the contents of `gaia_text.xs` into the carrier's Script Call source.
3. Add this entry point after the library:

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

The checked-in `gaia_text_minimal.xs` already combines those two pieces. The
small example recipe and scenario can be applied with Kit, but Kit is optional
for using the XS directly from the editor or a Python scenario tool.

For a fuller demonstration, apply `gaia_text_showcase.recipe.json`. It shows
runtime text, a changing score, a changing stroke counter, and two sampled
quadratic/cubic paths in one no-data-mod scenario.

The ready-to-run scenario is `Gaia Text Showcase.aoe2scenario`; the assembled
source is `gaia_text_showcase.xs`.

For a static-material performance pass, apply `gaia_text_static_material.recipe.json`.
Its stump counter uses the non-fire retirement adapter and one-shot sweep triggers;
the same scenario also lays out the static-material swatch beside the animated flame
control. The ready-to-run scenario is `Gaia Text Static Material.aoe2scenario`;
`gaia_text_static_material.types` is the matching sidecar schema for the run.

For an isolated retirement diagnosis, apply `gaia_text_static_retire_probe.recipe.json`
to a blank scenario. `Gaia Text Static Retire Probe.aoe2scenario` is already built from
the blank Kit seed: it contains only a P1 barracks keep-alive, one stump-number carrier,
and 60 one-shot Gaia removal triggers. Its sidecar schema records retirement calls,
successful moves, move failures, invalid IDs, and the retirement cursor independently.
This is an engine probe, not an engine-verified result; the game run is the oracle.

## Public Calls

- `drawSentence(slot, x, y, literal, material, scale)` renders runtime text.
- `drawNumber(slot, x, y, value, minDigits, material, scale)` renders numbers.
- `setSentence` and `setNumber` update existing runtime slots incrementally.
- `drawMessage(slot, x, y, messageID, material, scale)` is only for authored
  mixed case supplied through a scenario string table.
- `drawStrokeMessageTable(...)` selects the large-text outline layer explicitly.
- `drawStrokeNumber(...)` renders an incremental-friendly outline counter.
- `pathReset`, `pathMove`, `pathLine`, `pathQuadratic`, and `pathCubic` build
  cell-space paths; `pathStroke` samples and renders the current path.
- `measureText(literal)` returns bitmap width in cell pitches, or `-1` for
  unsupported/overlong input.
- `clearSlot` retires existing slots.

Call `fontInit()` and `fontLoadTables()` once before any draw call. Slots own
their Gaia objects and updates diff the existing cells. Fire materials retire
through hit points; non-fire materials require a configured kill-zone sweep.

## Limits

Runtime literals are case-folded by the DE XS engine. Use editor string-table
messages through `drawMessage` only when authored case matters. The default bitmap
path is the small-text renderer. Layered stroke is for large titles, counters,
and monuments; it uses up to 256 points per glyph and a declared 10,000-cell
runtime budget. A long document must be presented as bounded windows rather
than drawn as one enormous object set.

Glyph geometry is baked into the library once. Text and numeric values are
runtime inputs, so a scoreboard or timer changes without regenerating XS. The
verified clock touched 37 cells on a minute rollover out of roughly 150 live
cells.

The standalone file is the product. Kit is optional and adds offline document
compilation, custom styles/material footprint derivation, bounded-window cost
planning, scenario patching, XS embedding, and structural verification. The
planner does not run a presentation scheduler for you; your scenario logic
advances windows and calls the draw/update API. Kit does not replace the
game/editor as the engine-rendering oracle.

The path API is a general sampled-geometry layer. It supports fractional
coordinates and quadratic/cubic Bézier construction, but its visual density
and curve quality should still be checked in the game because the final
renderer is Gaia artwork, not a native DE vector canvas.

Prerequisites: Age of Empires II: Definitive Edition with the current XS
runtime, a scenario Script Call carrier, and a Gaia-visible map position. No
data mod is required.
