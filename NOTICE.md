# Notices and Credits

AoE2Kit is licensed LGPL-3.0-only.

This notice is generated from `data/credits.json`, the credit source of truth. If a source, author, project, or community should be credited and is missing, that is a bug.

## Credit Principle

Attribution is abundant and automatic. Personal thanks is scarce, deliberate, and human. The ledger keeps unpaid gratitude visible instead of pretending it has been paid.

## Built On

Published work studied, built on, or used as a reference. Attribution is abundant and automatic.

### AoE2ScenarioParser

- Authors: KSneijders and contributors
- URL: https://github.com/KSneijders/AoE2ScenarioParser
- License: GPL-3.0
- Permission: GPL-3.0
- Gave: A consulted structural scenario-format reference used during AoE2Kit's independent byte-corpus reconstruction of the DE scenario parser.
- Role: Scenario format specification reference
- Relationship: consulted_format_reference
- Rung earned: credited
- Thanks status: owed
- Correction state: open_to_correction

AoE2Kit credits KSneijders and contributors for the AoE2ScenarioParser structural specification as a consulted format reference during Kit's independent byte-corpus reconstruction. AoE2Kit does not vendor the AoE2ScenarioParser implementation or its schema artifact.

Notes:
- The credited project is a consulted format reference, not a vendored copy of its Python implementation.
- This credit does not change AoE2Kit's LGPL-3.0-only license.
- Personal thanks to KSneijders and contributors is a human action and remains owed until it is actually sent.

### Advanced Genie Editor (AGE)

- Authors: Tapsa and contributors
- URL: https://github.com/Tapsa/AGE
- License: GPL-3.0
- Permission: GPL-3.0
- Gave: A long-standing open-source Genie DAT editor and practical oracle for DAT editing behavior, command terminology, and field naming.
- Role: DAT editor oracle and terminology reference
- Relationship: reference_implementation_and_behavior_oracle
- Rung earned: credited
- Thanks status: owed
- Give-back status: prepared_by_lgpl_release_and_notice
- Correction state: open_to_correction

AoE2Kit's DAT authoring and semantic naming work was informed by AGE's long-standing Genie DAT editor behavior, command names, and field terminology. AGE is credited deliberately as a reference and behavior oracle.

Notes:
- AoE2Kit does not vendor AGE source code.
- AGE remains the community reference editor for Genie DAT files.
- Where AoE2Kit uses AGE-shaped terminology, behavior is still verified through AoE2Kit roundtrip/readback tests and engine evidence when available.
- Personal thanks to Tapsa is a human action and remains owed until it is actually sent.

### genieutils

- Authors: Tapsa and contributors
- URL: https://github.com/Tapsa/genieutils
- License: LGPL-3.0
- Permission: LGPL-3.0
- Gave: Open-source Genie file library/tooling and a cautionary reference for codec scope, API shape, and file-corruption risk.
- Role: Genie file tooling reference
- Relationship: format_reference
- Rung earned: credited
- Thanks status: owed
- Correction state: open_to_correction

AoE2Kit's DAT codec work was designed in conversation with the shape and risks of existing Genie tooling, including genieutils, while choosing a Go-native codec and verification-first architecture.

Notes:
- AoE2Kit does not vendor genieutils source code.
- The codec architecture favors explicit spans, roundtrip gates, and narrow verified writes.

### aoe2-genie-tooling

- Authors: GoKuModder
- URL: https://github.com/GoKuModder/aoe2-genie-tooling
- License: LGPL-3.0-only
- Permission: LGPL-3.0-only
- Gave: A high-level LGPL DAT authoring toolkit and catalog of Genie attributes, resources, tasks, tech modifiers, effect commands, registry patterns, and reference-validation ideas.
- Role: DAT authoring semantics oracle and ergonomics reference
- Relationship: compatible_open_source_reference_and_checklist
- Rung earned: credited
- Thanks status: owed
- Give-back status: prepared_by_lgpl_release_and_notice
- Correction state: open_to_correction

AoE2Kit's DAT facts catalog, effect-command disagreement ledger, registry thinking, and reference-validation roadmap were informed by GoKuModder's aoe2-genie-tooling. AoE2Kit treats it as a credited LGPL oracle/checklist while keeping Kit's Go implementation, evidence tiers, and engine-verification claims explicit.

Notes:
- aoe2-genie-tooling is LGPL-3.0-only, matching AoE2Kit's license posture.
- AoE2Kit does not currently vendor aoe2-genie-tooling source code.
- Where GoKu's labels disagree with AoE2Kit's observed current-DE rows, the disagreement is preserved in Kit output instead of silently resolved.
- Personal thanks to GoKuModder is a human action and remains owed until it is actually sent.

### AoE2_AI_Modder

- Authors: GoKuModder
- URL: https://github.com/GoKuModder/AoE2_AI_Modder
- License: MIT
- Permission: MIT
- Gave: An agent-facing AoE2 knowledge base that models compact routing across XS, AI scripts, scenario triggers, and DAT/genie tooling.
- Role: Agent knowledge-base architecture reference
- Relationship: documentation_and_agent_workflow_reference
- Rung earned: credited
- Thanks status: owed
- Correction state: open_to_correction

AoE2Kit's thinking about compact, local, agent-readable knowledge surfaces was sharpened by GoKuModder's AoE2_AI_Modder repository, especially the idea that tools should ship with routing guides and precise lookup datasets for coding agents.

Notes:
- AoE2_AI_Modder is MIT licensed.
- AoE2Kit does not vendor AoE2_AI_Modder source code.
- Personal thanks to GoKuModder is a human action and remains owed until it is actually sent.

### aoc-mgz

- Authors: happyleavesaoc and contributors
- URL: https://github.com/happyleavesaoc/aoc-mgz
- License: MIT
- Permission: MIT
- Gave: A mature recorded-game parser ecosystem and reference point for AoE2 replay parsing.
- Role: Recorded-game parser reference
- Relationship: format_reference
- Rung earned: credited
- Thanks status: owed
- Correction state: open_to_correction

AoE2Kit's replay work benefits from the aoc-mgz ecosystem and keeps replay claims explicit about what is parsed, inferred, or still opaque.

### AoE2DE UGC Guide

- URL: https://ugc.aoe2.rocks/
- Permission: public documentation reference
- Gave: Public documentation for AoE2DE scenario, trigger, XS, and modding concepts.
- Role: Scenario, trigger, XS, and data-mod documentation reference
- Relationship: documentation_reference
- Rung earned: credited
- Thanks status: owed
- Correction state: open_to_correction

AoE2Kit uses the UGC Guide as a public reference point for scenario, trigger, XS, and modding concepts, then validates critical behavior against real files and engine runs.

### CB Front Towers OG Enhanced

- Authors: SpiRaL
- URL: https://steamcommunity.com/id/theogspiral/
- License: All rights reserved by the author
- Permission: studied for technique only; the author forbids modifying or redistributing the map without consent
- Gave: A masterclass in scenario-only UGC technique: the reroll shop (train-to-consume renamed button units), object and technology renaming with no data mod, the repeatable-technology purchase trick (disable-then-enable reset), and a live Objective-Panel stats display.
- Role: Scenario-design technique oracle and inspiration
- Relationship: behavior_oracle_and_inspiration
- Rung earned: credited
- Thanks status: owed
- Correction state: open_to_correction

AoE2Kit's scenario authoring — its object/technology rename operations, repeatable-technology decode, and shop idioms — was materially informed by studying SpiRaL's CB Front Towers, which has been a direct inspiration. Front Towers served as a behavior oracle; every technique was re-implemented independently and verified against real files.

Notes:
- AoE2Kit does not vendor, modify, or redistribute CB Front Towers content. The map and its XS were studied for technique only, in keeping with the author's license.
- The reroll shop, button/technology rename, repeatable-technology reset, and live-panel idioms were reverse-engineered as a behavior oracle and re-implemented from scratch.
- Personal thanks to SpiRaL is a human action and remains owed until it is actually sent.

## Personal Thanks

Individual people who have directly, personally helped the author. Scarce, deliberate, and human.

### Sekiro

- Authors: Sekiro
- Gave: Taught the terrain-layering technique: the per-tile Genie `layer` field (base terrain = bottom, layer terrain = top; top dominates the blend), used as a top/bottom crossfade. This became the foundation of AoE2Kit's geo-trace terrain pipeline — graded coastlines, water-depth shelving, biome seams, layered political borders, dock-buildable/amphibious shores, and the hybrid-by-default terrain principle.
- Role: Terrain-layering technique oracle; AoE2Kit's first active external user
- Relationship: technique_oracle_and_inspiration
- Rung earned: credited
- Thanks status: owed
- Correction state: open_to_correction

AoE2Kit's terrain-layering work — the coast crossfade, water-depth grading, layered borders, and the hybrid-by-default terrain model — was taught to the project by Sekiro, who is also AoE2Kit's first active external user. The engine-level mechanics (the `layer` field storage, top/bottom blend, and passability rule) were then verified byte-for-byte and in-engine before being built on.

Notes:
- Sekiro is the first designer known to actively use AoE2Kit in their own live scenarios.
- The terrain-layering technique was demonstrated by Sekiro, decoded against real scenario files, and re-implemented in AoE2Kit's writer with verification.
- Personal thanks to Sekiro is a human action and remains owed until it is actually sent — to be given in the next project update.

### SpiRaL

- Authors: SpiRaL
- Gave: Direct personal help to the author, as a fellow AoE2 designer.
- Role: Fellow AoE2 designer who has personally helped the author
- Relationship: personal_help
- Rung earned: credited
- Thanks status: owed
- Correction state: open_to_correction

SpiRaL has personally helped the author. Personal thanks is deliberate and human, and remains owed until it is actually given. (SpiRaL is also credited under Built On as the author of CB Front Towers, studied for technique.)

Notes:
- Personal thanks to SpiRaL is a human action and remains owed until it is actually sent.

### krmyth9 (Myth)

- Authors: krmyth9
- Gave: Direct personal help to the author, as a fellow AoE2 designer.
- Role: Fellow AoE2 designer who has personally helped the author
- Relationship: personal_help
- Rung earned: credited
- Thanks status: owed
- Correction state: open_to_correction

krmyth9 has personally helped the author. Personal thanks is deliberate and human, and remains owed until it is actually given.

Notes:
- Personal thanks to krmyth9 is a human action and remains owed until it is actually sent.
