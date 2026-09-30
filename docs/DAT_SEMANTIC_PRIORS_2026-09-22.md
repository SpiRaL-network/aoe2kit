# DAT Semantic Priors

This is a data-derived prior, not an engine fact. It records what the
reference DE DAT says before an unattended engine harness tests the same
identities. A disagreement is valuable: the DAT field name or value is not
treated as the engine's final semantic rule.

## Source

- DAT: the stock `empires2_x2_p1.dat` of build `101.103.48987.0`
- SHA256: `ce3530df36cf0b333a9751cb0ff94460fe904f811feecec8ae9794701622b4cf`
- Cache schema: `2`
- DAT version: `VER 8.9`
- Civ rows: `60`

The queries below run against the read-only SQLite cache created by
`pkg/datcache`. Unit counts in the summary tables are distinct `unit_index`
identities, not duplicated civ rows.

## Collision Prior

Prediction rule for fractional placement: a unit is *predicted fractional-safe*
only when `collision_size_x = 0`, `collision_size_y = 0`, and
`collision_size_z = 0`. Any other signature is a harness candidate, not an
automatic claim that the unit blocks movement.

Distinct identity counts:

| X | Y | Z | Units | Creatable |
|---:|---:|---:|---:|---:|
| 0 | 0 | 0 | 813 | 55 |
| 0 | 0 | >0 | 13 | 13 |
| >0 | >0 | 0 | 392 | 140 |
| >0 | >0 | >0 | 1,424 | 1,102 |

The sign-bucket counts across all DAT rows are 42,349 / 780 / 20,570 /
65,026 respectively. The identity table is the appropriate sampling frame;
civ-row counts would overweight common units.

The known cactus result is therefore a deliberate falsification target: the
`X>0,Y>0,Z=0` bucket contains many identities, but engine movement must not be
predicted from X/Y alone.

## Removal Prior

This prior was written before the removal harness ran. The original hypothesis
was that class `14` plus `hit_points = 1` would behave as removable eyecandy
under `xsSetUnitHitpoints(u, 0.0)`. Later engine runs showed the opposite for
static class-14 plants: they ignore hit points and must be removed by triggers
(the kill-zone sweep); see `XS_ENGINE_VERIFIED_2026-09.md`. The working rule is
whether an object dies to hit points, not its class. The buckets below remain
useful as sampling frames for further probes.

Distinct identity counts:

| Bucket | Units |
|---|---:|
| class 14 and HP 1 | 241 |
| class 14 and HP != 1 | 249 |
| HP 1 and class != 14 | 384 |

The first removal harness should select representatives from all three
populations, including a creatable class-14/HP-1 control where available. A
successful removal result upgrades only the tested predicate; it does not make
the whole bucket engine-verified.

## Taxonomy Priors

The DAT contains 813 zero-footprint identities. They are distributed across
decorative, disabled, and creatable records, so `collision=0` is a placement
predicate, not an artwork predicate.

The broad class/type query reports many class-14 records with `has_creatable=0`
and both enabled and disabled states. Class 14 is therefore a useful eyecandy
candidate filter but is not, by itself, a removal or passability guarantee.

## Reproduction Queries

```sql
-- Use max() only after grouping by unit_index; civ rows duplicate identities.
WITH u AS (
  SELECT unit_index,
         max(name) AS name,
         max(class) AS class,
         max(hit_points) AS hit_points,
         max(has_creatable) AS creatable,
         max(collision_size_x) AS x,
         max(collision_size_y) AS y,
         max(collision_size_z) AS z
  FROM units
  GROUP BY unit_index
)
SELECT x > 0 AS x_pos, y > 0 AS y_pos, z > 0 AS z_pos,
       count(*) AS units, sum(creatable) AS creatable
FROM u
GROUP BY 1, 2, 3
ORDER BY 1, 2, 3;

WITH u AS (
  SELECT unit_index, max(class) AS class, max(hit_points) AS hit_points
  FROM units
  GROUP BY unit_index
)
SELECT CASE
         WHEN class = 14 AND hit_points = 1 THEN 'class14_hp1'
         WHEN class = 14 THEN 'class14_other'
         WHEN hit_points = 1 THEN 'hp1_other'
         ELSE 'other'
       END AS bucket,
       count(*) AS units
FROM u
WHERE class = 14 OR hit_points = 1
GROUP BY bucket;

WITH u AS (
  SELECT unit_index, max(type) AS type, max(class) AS class,
         max(has_creatable) AS creatable,
         max(collision_size_x) AS x,
         max(collision_size_y) AS y,
         max(collision_size_z) AS z
  FROM units
  GROUP BY unit_index
)
SELECT type, class, creatable, count(*) AS units
FROM u
WHERE x = 0 AND y = 0 AND z = 0
GROUP BY type, class, creatable
ORDER BY type, class, creatable;
```

## Status

All claims in this document are `scenario_decode`/catalogued priors pending
engine verification, except where noted. The collision Z rule is separately
`in_game_verified` from the cactus walk test, and the class-14 removal
hypothesis was falsified in-engine (see Removal Prior); this document intentionally preserves the broader
DAT-derived populations so future harness runs can look for exceptions.
