# Engine build scope

Engine-verified facts are scoped to the game build that produced the evidence.
They are not timeless properties of every Age of Empires II: Definitive Edition
patch.

## Current baseline

| component | value |
| --- | --- |
| executable | `101.103.48987.0` |
| executable date | `2026-09-15` |
| `empires2_x2_p1.dat` SHA256 | `ce3530df36cf0b333a9751cb0ff94460fe904f811feecec8ae9794701622b4cf` |
| `Constants.xs` SHA256 | `d9c94e6f5894c530fbad647c1c6a6b2d80943d855515899e50ed05f012072b0a` |
| `Effects.xs` SHA256 | `ab1ccaaaf13cd40228c2b5af6ff3b77d4efef4921f2ea55321e5b5d80e42844e` |
| flattened enum list SHA256 | `5cd8686d6e5b0cd326eb68129faffb1ce2ca138e764fcc76e283c4abbbb4a4ca` |
| English key-value strings SHA256 | `fa8bc9c0c90cd17e0cfce85db0b9a75c8a6cc67ab580932e86dbaad41df9ae4e` |
| DAT version | `VER 8.9` |
| captured | `2026-09-22`, before the pending DLC patch |

Every entry in `XS_ENGINE_VERIFIED_2026-09.md` was observed against this baseline unless an
entry states a narrower fixture or date. New engine-verified entries must record
the executable version and DAT SHA256; do not silently inherit this baseline.

The baseline also includes the shipped enum/symbol inputs: `Constants.xs`,
`Effects.xs`, `enums_flat_*.txt`, and the English key-value string table. Prefer
these local, versioned inputs over paraphrased community constants when decoding
civs, attributes, effects, unit names, and artwork labels.

## After an engine update

1. Preserve the old DAT and executable metadata before replacing files.
2. Run `kit dat diff` against the baseline.
3. Re-run `kit dat semantic-priors` and compare its output with the stamped prior.
4. Re-check fixture constants and semantics before upgrading any engine fact.

A changed DAT invalidates the scope of an observation; it does not automatically
disprove the observation. Until re-run, label it as verified for the prior build.

## Captured post-update profile

The installed post-update profile is `101.103.54800.0`, dated `2026-09-22`.
Its DAT is 12,023,754 bytes with SHA256
`4aa2f0a719e88e5f1502517eddb27c669aeb40c2fe9d8c4f3eec7751c01e7baa`.
The captured `Constants.xs` SHA256 is
`240a0102456d8c5edd7c4cb73198a5ff1c78aee52be19842d4d8cb3b6d902987` and the
flattened enum SHA256 is
`ea3a44c1f9f6d8b344df4c637884afe5c6a0bce709060c454754fc70761f09fb`.
A semantic-priors snapshot was generated for this build with
`kit dat semantic-priors`.

The post-update `Effects.xs` snapshot is also captured at 53,983 bytes with
SHA256 `4f111fd37ebdbdf644dbeed95ce4994d5257534a7f0569baa00e0740e19956c5`.
The pre- and post-update files are now diffable, and the post-update file adds
108 top-level unit-ID constants. That makes it a current local symbol source
and a new identifier-collision surface for generated XS. The symbol capture does
not by itself re-verify effect semantics in the engine.

Structural DAT comparison against the pre-update baseline reports 4,028 added
and 338 changed decoded records, with the graphics section accounting for 3,933
added records. The new prior changes are: civs 60->63, unit identities
2642->2693, class-14/HP-1 removal candidates 241->263, and zero-collision
identities 813->803. These are data deltas, not engine-verification upgrades.

Effect IDs and names now have a post-update local snapshot. Treat their
semantics as build-scoped and re-verify them with targeted engine probes before
promoting behavior claims beyond the captured build.
