# DE 1.59 Scenario Format Notes

Status: **read and write supported.** AoE2Kit decodes 1.59 scenarios with its own
typed codecs and rebuilds every supported file in its test corpus byte-for-byte
(`kit scen verify`). This page records what differs from 1.58 and how each
difference was established.

## What changed from 1.58

| area | 1.59 difference | evidence |
| --- | --- | --- |
| Units | one extra byte in each plain unit record, between `garrisoned_in_id` and `caption_string_id` | a crashing/fixed specimen pair walked all nine player-unit sections only with the byte present |
| Triggers | trigger section version `5.0` | section header of every 1.59 file |
| Conditions | two record layouts, selected per condition by a marker read before the variable part: marker `33` (compact) and marker `34` (editor form, with one extra 4-byte field, always `ff ff ff ff`, before the XS function name) | editor saves carry `34`; files written by other tools carry `33`; both forms occur in the corpus and rebuild exactly |
| Effects | effect records carry marker `83` in the same position | editor calibration saves |

The extra unit byte and the extra condition field are kept byte-for-byte and are
not given a meaning: nothing observed so far varies them.

## Known variant: preserved, not decoded

One official campaign file uses a different trigger record framing (a five-byte
separator between trigger records). Kit detects this structurally and preserves
the whole trigger section byte-for-byte, so the file still rebuilds exactly, but it
reports no structured triggers for that file. Re-saving such a file in the editor
converts it to the standard framing.

## Editor behavior worth knowing

The editor fills in several convenience values on its own (default trigger names,
some effect defaults, map defaults on load). These are documented, with the saves
that established them, in [`AOE2KIT_WRITE_SMOKE.md`](AOE2KIT_WRITE_SMOKE.md)
under *editor normalization*.

## Older versions

Reading 1.57 and older is not supported. Kit reports
`unsupported scenario version` for those files instead of guessing. Opening an old
scenario in the editor and saving it converts it to 1.59.
