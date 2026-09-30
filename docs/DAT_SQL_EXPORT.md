# DAT SQL Export

`kit dat sql <empires*.dat> <out.db>` writes a query-oriented SQLite view of
the decoded unit table.

The `units` table has one row per present civ/unit record. Stable scalar fields
decoded by the current DE parser are columns. Variable-length data is normalized
into `unit_attributes`, `unit_attacks`, `unit_armours`, and `unit_fields`.
`attribute_map` records the UGC attribute id, the current descriptive name when
known, a candidate DAT field path, the confidence label, and an `exported` flag.
`exported=1` means the named path is available in this database (either in
`unit_fields` or the corresponding child table); it does not mean the semantic
meaning is engine-verified. A `layout_correlated` label is structural evidence
from the DAT layout, not an engine-verified claim.

Unit-reference fields use the DAT signed-sentinel convention: `-1` means that
the reference is absent. In particular, `annex_unit_*`, `stack_unit`,
`head_unit`, `transform_unit`, `pile_unit`, and `death_spawn` are signed
references. The raw DAT encoding for an absent 16-bit reference is `0xFFFF`;
the Go decoder and SQL export normalize that value to `-1`, so queries do not
need to special-case `65535`.

The exporter also writes `attribute_map_*` coverage keys to `metadata` and emits
a warning when mapped attributes point at fields the decoder does not export.
This is intentional: the SQL database is a partial projection until the unit
record decoder has proven the remaining DE layout blocks. Consumers should
filter on `attribute_map.exported=1` and treat `exported=0` as an explicit
decode gap, not as evidence that no unit carries the attribute.

The normal DAT index does not retain full field maps for every unit. `kit dat
unit` decodes one requested record on demand; the SQL exporter does the same one
record at a time while inserting rows. This keeps the normal parser compact and
avoids multiplying the in-memory cost of the inflated DAT by retaining JSON
copies of every unit.

The current decoder includes the type-30 movement/trail block and the named
type-50 combat/projectile fields, including the type-60 missile-specials block.
Some older and DLC-specific fields remain intentionally opaque; those remain
unexported until their byte offsets are established. In particular, the charge
tail and secondary-projectile fields are still a decode target;
their zero-row result must not be interpreted as absence in the DAT. The export is
structure-verified against the parser and fixture corpus. It is not an engine
verification of semantic meanings for fields that remain labeled
`layout_correlated` or `unmapped`.
