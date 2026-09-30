# XS Trigonometry Capability Audit

Status date: 2026-09-23

## Finding

XS can use trigonometry. The UGC guide prelude declares the scalar functions
`sin`, `cos`, `tan`, `asin`, `acos`, `atan`, `atan2`, `radians`, `degrees`,
`pow`, `sqrt`, and `dist`, and AoE2Kit's generated builtin inventory contains
them (guide commit `1f0f7e8`).

Runtime status on DE `101.103.54800.0`:

| function | status |
| --- | --- |
| `sin`, `cos`, `radians` | executed in an engine probe (see `XS_ENGINE_VERIFIED_2026-09.md`, trig-long entry) |
| `tan`, `asin`, `acos`, `atan`, `atan2`, `degrees`, `pow`, `sqrt`, `dist` | declared; expected to compile, not yet runtime-verified |

## Practical guidance

- Generated geometry (rings, arcs, rotations, distance placement) can compute
  vectors at runtime with `radians`, `cos`, and `sin` instead of emitting large
  precomputed `xsArraySetVector` tables.
- Keep baked tables for static data that has no runtime benefit, such as glyph
  coordinates.
- When replacing a baked table with runtime trig, compare the runtime result
  against the old table in-engine before changing point counts or placement.

## Audit rule

Guide declarations are capability inventory, not proof. Generated XS should
use typed/scalar intermediates and should remain behind static validation;
foreign XS may use the same capability only with an explicit expected-compile
label until a target build has executed it.
