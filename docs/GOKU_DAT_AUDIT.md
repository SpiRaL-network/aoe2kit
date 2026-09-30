# GoKu DAT Audit

AoE2Kit uses GoKuModder's DAT work as a credited LGPL-compatible oracle and checklist, not as an unexamined source of truth. The useful split is:

- **Adopt** when the item is a stable catalog/ergonomic pattern and agrees with Kit evidence.
- **Verify** when GoKu's label is plausible but Kit has a competing interpretation or no engine proof.
- **Defer/reject** when the pattern is Python architecture rather than a Kit capability, or when it would weaken Kit's evidence discipline.

## Adopt

- **Named DAT catalogs**: attributes, resources, task ids, task attributes, tech modifiers, unit classes/types, store modes.
- **Manager/ownership thinking**: units, graphics, sounds, techs, effects, and civ data are different authoring surfaces and recipes should validate cross-links explicitly.
- **Stable registry pattern**: created DAT objects should have stable human/AI-facing keys rather than only integer ids.
- **Reference validation as first-class tooling**: a delete/move is not safe until inbound and outbound references are known.

## Verify

- **Effect commands 200-206** are the main disagreement zone.
  - AoE2Kit: `200/201` are source-verified local-building set/add, `202` is strong-hypothesis local-building multiply, `204` is strong-hypothesis local-building advanced/packed add.
  - GoKu: `200-202` are own-master-objects modifiers and `203-206` are selected-unit modifiers/transform.
  - Action: keep both readings visible in `kit dat facts` and require engine fixtures before promoting `202/204/203/205/206`.
- **Task authoring**: the task id catalog is useful, but authoring tasks safely requires per-task field validation. Kit should not imply every listed task is write-safe.
- **Full reference map**: GoKu's field-discovery docs are a strong checklist, but Kit should mark each reference rule by verified rewrite support before enabling destructive operations.

## Defer / Do Not Copy Blindly

- **Python manager architecture**: useful as a mental model, but Kit stays Go-native and recipe-oriented.
- **Parser dependency shape**: GoKu's tooling delegates low-level parsing to a Rust-backed GenieDatParser. AoE2Kit's DAT codec should continue to own Go roundtrip/read/write coverage directly.
- **Unverified command labels**: especially in the 200-series, labels stay hypotheses until the fixture evidence says otherwise.

## Kit Surfaces Added From This Audit

- `kit dat facts effect-types`
- `kit dat facts effect-command <id>`
- `kit dat facts attributes`
- `kit dat facts resources`
- `kit dat facts task-types`
- `kit dat facts tech-modifiers`

These are authoring catalogs. They complement, but do not replace:

- `kit dat command-matrix <dat>` for a structural census of one DAT file.
- `kit dat effect-explain <dat> <effect_id>` for concrete effect row interpretation.
- `kit dat refs <dat> <section> <id>` for inbound/outbound reference safety.

## Credit

GoKuModder's `aoe2-genie-tooling` is LGPL-3.0-only and is credited in `data/credits.json` / `NOTICE.md`. `AoE2_AI_Modder` is MIT and is credited separately as an agent-knowledge architecture reference.
