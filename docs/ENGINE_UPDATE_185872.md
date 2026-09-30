# Update 185872 compatibility boundary

Update 185872 (released 2026-09-22) changed the XS surface substantially.
Engine-verified facts observed before it remain valid only for their stamped
baseline:

`101.103.48987.0` / `empires2_x2_p1.dat` SHA256
`ce3530df36cf0b333a9751cb0ff94460fe904f811feecec8ae9794701622b4cf`

Source: [Update 185872 release notes](https://www.ageofempires.com/news/age-of-empires-ii-definitive-edition-update-185872/)

## Status

- The post-update executable (`101.103.54800.0`), DAT, `Constants.xs`,
  `Effects.xs`, and flattened enums are captured and hashed; see
  [`ENGINE_BUILD_SCOPE.md`](ENGINE_BUILD_SCOPE.md).
- The unattended `trig-long` harness ran on the post-update build: `ord`,
  `strCharAt`, `strSubstring`, `strToUpper`, `strSplit`, `strJoin`, `fstr`,
  `sin`, `cos`, and `radians` executed. See
  [`XS_ENGINE_VERIFIED_2026-09.md`](XS_ENGINE_VERIFIED_2026-09.md).
- The remaining items below are still `needs_reverification`.

## Impact inventory

### New XS string/runtime surface

The update adds character conversion, indexing, length, substring, search,
replacement, splitting/joining, trimming, case conversion, reverse, numeric
parsing, formatting, and interpolation functions. This supersedes workarounds
that decoded strings with comparison plus concatenation because XS had no
indexing or length operation, and it removes the runtime mixed-case limitation
of that approach. The functions listed under Status are engine-verified; the
rest are declared but not yet individually probed.

### Breaking names

| old name | new name | action |
| --- | --- | --- |
| `xsGetLocalPlayerId` | `xsUnsyncGetLocalPlayerId` | confirm against the captured post-update constants before updating generated or foreign XS |
| `xsGetTimerTimeRemaining` | `xsUnsyncGetTimerTimeRemaining` | same |

The authoring validator must make this symbol set build-scoped. It must not
reject a script solely because it targets the old stamped build.

### Previously catalogued behavior requiring re-check

The release notes report fixes or changes touching:

- top-level/global string initialization (`TopStrInit`);
- escape characters in XS string literals;
- `runImmediately` rule startup;
- one-shot behavior of rules in `Effects.xs`;
- caption/name APIs and unit property APIs;
- object attributes 167 through 221;
- map tile attribute access and camera controls.

These are `needs_reverification`, not retracted facts. The old facts remain
engine-verified for the old build.

## Revalidation order

1. Capture and hash the post-update inputs (done).
2. `kit dat diff` the post-update DAT against the pre-update baseline, and diff
   the symbol files (done; see `ENGINE_BUILD_SCOPE.md`).
3. Rebuild semantic priors and inspect new civ, unit, artwork, attribute, and
   effect IDs before touching recipes.
4. Run the unattended harness, beginning with load/parse and carrier write
   readback; record any changed parser or runtime behavior separately.
5. Recheck fixture constants, attribute 110, collision signatures, removal
   buckets, and the two renamed APIs.
6. Add a new stamped ledger section for the updated build. Do not overwrite
   the old baseline claims.

## Product consequence

Kit should expose build-scoped XS symbols and compatibility facts as data, not
hard-code a single universal language profile. Generated XS can target a named
profile; foreign XS can be analyzed against the profile selected by the author.
