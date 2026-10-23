# Felt Compendium Field Schema

This document describes the most important fields in the felt compendium data files.

The JSON files contain richer notes than this document can summarize. Treat this as the stable
orientation layer.

## Art Entries

Primary file:

- `../data/art-felt-compendium.json`

Important fields:

- `unit_const`: AoE2 unit id to place in scenario recipes.
- `name`: human-readable unit/object name.
- `graphic_file`: source graphic family when known.
- `form`: rough object shape, such as `tree`, `plant`, `statue`, `prop`, `modular_tile_set`,
  `single_artwork`, or `terrain_district`.
- `variant_semantics`: what rotation/frame indices mean. Examples: `rotation`, `variety`,
  `hue_gradient`, `state_direction`, `tile_set_topological`.
- `material`: broad visual material: greenery, stonery, woodery, metal, water, ice, bone, and mixed
  variants.
- `function`: design role: border, enclosure, rug, path, eyecandy, marker, feature, elevation_faker.
- `placement_hint`: how it wants to be placed: scatter, landmark, line, assemble_on_grid.
- `vibe_tags`: searchable mood and context tags.
- `felt`: prose interpretation of what the asset says in a scene.
- `use_when`: practical authoring guidance.
- `standing`: confidence/source tier. Preserve this when generating reports.
- `source`: where the observation came from.
- `sld_size_kb`: size of the source graphic file when known. This is a complexity hint, not a safety
  guarantee.

## Measurement Fields

Primary file:

- `../data/px-dims.json`

Important ideas:

- Pixel dimensions measure the rendered content bounds from contact sheets.
- Tile footprint comes from DAT radius fields where available.
- A visually large object may still be walkable.
- A visually small object may still block a tile.

Use footprint and pixel size together:

- pixel size answers "how much screen presence does this have?"
- tile footprint answers "will units path around it?"

## Terrain Entries

Primary file:

- `../data/terrain-felt-compendium.json`

Important fields:

- `terrain_id`: AoE2 terrain id for map painting.
- `name` / `name_2`: readable names and internal terrain texture names.
- `category`: broad family such as grass, snow, water, road, forest floor, beach, dirt.
- `material`: surface material.
- `biome`: biome associations.
- `mood`: short felt summary.
- `vibe_tags`: searchable design tags.
- `felt`: prose interpretation.
- `use_when`: authoring guidance.
- `standing`: evidence tier.

Terrain swatches are raw material evidence. They do not fully capture in-engine blending, edge masks,
or large-scale composition.

## Confidence Rules

Do not erase uncertainty.

- If the field says `ai_felt_from_render`, report that as grounded in rendered contact sheets, not
  as engine proof.
- If an object requires terrain context, mention that context. Examples: water lilies, fish traps,
  garden bridges.
- If an object is civ-selected or context-selected, do not treat every visual culture variant as a
  directly placeable independent unit.
- If a scene recipe is structure-verified but not opened in DE, say so.
