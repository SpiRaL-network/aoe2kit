# Felt Compendium data

This directory contains a machine-readable visual catalog for AoE2DE decorative Gaia objects and
terrain. It helps scenario authors select assets by material, mood, role, scale, and placement
behavior.

The public data package includes:

- `data/art-felt-compendium.json` — 232 decorative-object entries.
- `data/terrain-felt-compendium.json` — 52 terrain entries.
- `data/gaia_palette.json` — DAT-derived Gaia graphics, animation, and variant metadata.
- `data/px-dims.json` — measured rendered bounds and tile-footprint data.
- `data/visual_variants.json` — seasonal and visual variant mappings.
- `data/manifests/` — render and terrain-scrape manifests.

See [`docs/FIELD_SCHEMA.md`](docs/FIELD_SCHEMA.md) for the main fields and confidence rules.

This is data only. Rendered contact sheets and terrain swatches are not included. Paths in manifest
and catalog records identify optional companion images; they do not imply those images are shipped.

Treat each entry according to its `standing` or `confidence` value. A rendered or DAT-derived
classification is not engine verification. Check context-dependent placement and appearance in
AoE2DE when correctness matters.
