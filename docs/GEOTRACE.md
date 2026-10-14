# Experimental Geotrace

`kit geo trace-dem` turns caller-selected geographic evidence into scenario
terrain, object placement and an auditable recipe. **Experimental**: the
output is a playable cartographic interpretation, not a surveying product or
an engine-verified random-map generator. No regional profiles, game artwork,
geographic datasets or run outputs are distributed with this feature.

## Inputs and Outputs

Specify `--bbox minLat,minLon,maxLat,maxLon`, or `--center lat,lon --span-km N`.
`--grid` selects scenario tiles, not source raster resolution. All features use
the same projection: `--orient iso-ccw-screen --projection-aspect 2
--projection-north-scale 1` is the screen-oriented framing policy. Rotated
corners can extend outside the nominal bbox; the report gives the sampling
envelope and source coverage.

- Land: `--landmask geojson --landmask-file land.geojson`, or the optional
  Natural Earth fetch adapter. DEM-only zero elevation cannot distinguish sea
  from low land. `--provider flat` explicitly omits elevation evidence.
- Landcover: `--landcover worldcover --landcover-cache cache/worldcover` uses
  bounded GeoTIFF range requests and scale-selected overviews. A caller may
  instead provide `--landcover-file` or `--landcover-dir`. GLC-FCS30D is a
  separate adapter; WorldCover coastline policies do not apply to it.
- Roads: `--roads osm-major --roads-cache cache/roads` fetches bounded chunks.
  `--roads-file roads.json` accepts caller-supplied Overpass JSON or GeoJSON;
  fetched chunk responses are merged and deduplicated by the adapter. Declare a known query
  extent with `kit_query_bbox`; geometry bounds alone do not prove coverage.
- Art: `--profile profile.json` supplies Kit-owned placement choices;
  `--compendium art.json` is optional caller-supplied felt metadata, never a
  bundled sprite collection. `--usgs-woody-file presence.csv` is also
  caller-supplied. Inspect that dataset's provenance and terms yourself;
  this interface grants no redistribution rights.
- DAT: `--dat game.dat` supplies terrain passability and collision facts.
  It is required for `--scenario-out` and is not distributed by Kit.

For a local-input terrain trace:

```sh
kit geo trace-dem --bbox 40,0,40.5,0.5 --grid 160 --provider flat --landmask geojson --landmask-file land.geojson --profile profile.json --dat game.dat --out report.json --recipe-out recipe.json --scenario-out map.aoe2scenario
```

`--out` retains policy, counts, coverage, source attribution and audit results.
`--omit-report-cells` and `--omit-report-recipe` reduce bulky reports without
discarding attribution or the validation summary. Recipes describe generation;
`placement_sweep_recipe` is a separate follow-up patch against generated IDs.

## Profiles

Profiles use `aoe2kit-geotrace-ecology/v1`. Geography locates habitats; the
profile chooses artwork, quotas, spacing and gameplay resources. Metadata in
`notes` does not influence generation. No location-specific default landmarks
or flora preferences are implicit. The default mapping is Kit-authored rules,
not a copy of an external geographic or game-art dataset.

```json
{
  "schema": "aoe2kit-geotrace-ecology/v1",
  "name": "caller-map",
  "player_starts": {"mode": "explicit", "count": 2},
  "neutral_resources": [
    {"name": "gold", "unit": 66, "pile_size": 4,
     "groups_per_10000": 1.5, "spacing": 48, "start_distance": 48,
     "provenance": "Caller-selected gameplay budget, not geological evidence"}
  ],
  "layers": [
    {"name": "forest", "role": "resource",
     "landcover_keys": ["10_tree_cover"],
     "units": [{"unit_const": 350, "weight": 1}],
     "rate": 0.15, "limit": 40000, "patch_scale": 8}
  ]
}
```

Player starts and neutral mines are **off unless their profile fields are
supplied**. The explicit start package is not civilization-specific RMS
initialization. `flora_clearance` controls exclusion beyond the TC footprint;
resource-search pockets and narrow approaches are separate. Neutral piles are
atomic: the terrain gate rejects an entire pile if a member is invalid.

Layers support `resource`, `decorative`, `fauna` and `fish` placement roles,
weighted units, habitat filters, irregular patches, spacing and quotas.
Ecology roles select caller palettes (including freshwater, riparian, hunt,
herdable, predator and oyster roles). Water band `bay` is a land-surrounding
geometry proxy, not measured salinity or water depth. Source forest cover
informs patches; individual trees are generated, not measured locations.
USGS presence/absence informs a palette, not abundance or exact tree positions.

For the units listed in `pkg/geotrace/visual_variants.json`, Geotrace writes
the selected integer facet index directly to the scenario `rotation` field;
it does not convert that index to radians. Only indices present in the
selected non-empty state are written. The default state is `green`; an
explicit seasonal word in a profile's `biome` selects a matching non-empty
state when available. Selection is deterministic from the map seed and
placement cell. The oak mapping is engine-verified; other entries remain
classifier-derived and should not be treated as engine-confirmed state
mappings. Units without a record retain their existing rotation behavior.

Felt selectors replace explicit units with `functions`, optional `tags`,
`materials`, `unit_consts` and a positive `weight`. Selection weight is shared
among matches; the report records resolved palettes and the input checksum.
This opt-in accepts metadata schema `aoe2-art-felt-compendium.v1`; it neither
downloads nor redistributes artwork.

Gaia buildings and urban accents are **disabled by default and unvalidated in
the engine**. Their profile layers require `enable_unvalidated_urban: true`.
Static footprint/connectivity tests are not evidence that these objects work
as intended in game. Population-centre masks alone do not add buildings.

## Coast, Roads and Missing Data

Fine-scale WorldCover coastline refinement uses qualified pixels at tiles up
to 250 m. From 250 m through 1,500 m, thin-land preservation samples an
approximately tile-sized footprint at finer resolution and exaggerates narrow
islands to tile width. It is **cartographic exaggeration**, not exact coastline
or rotated-cell-area reconstruction. Explicit inland-water polygons override.
The report states policy, fallback counts, source class and land fraction.

Roads are simplified and stroked as vector polylines with round joins/caps,
then rasterized against tile footprints. Width is configurable; class factors
and surface tags refine it. Land and causeway eligibility still constrain
strokes. Raster roads are not proof of unit navigation or accurate bridges.

Incomplete/timeout road responses fail, rather than silently becoming empty
coverage. Completed validated chunks resume from atomic caches. Fetch-session
limits bound attempts, time and bytes. WorldCover observed 404s are cached for
24 hours and reported as coverage gaps. They retain the source landmask:
missing data is **not independent evidence of water**. Transport failures are
not cached as missing tiles. Unknown DAT/source evidence is never a pass.

## Validation and Attribution

The preplacement gate uses final terrain, snapped positions and DAT collision
footprints. A serialized-file sweep removes invalid ambient objects; required
start objects fail the build rather than shrink the package. The reopened
written audit must have zero remaining violations. Exact rebuild proves byte
preservation, not engine pathfinding, dock construction or gameplay balance.
Fine islands, docks and intended movement still require an in-game review.

Reports carry OpenStreetMap attribution, ODbL license and the road query extent
(explicitly unknown for local files lacking extent metadata). Preserve
`(c) OpenStreetMap contributors` and the [OpenStreetMap copyright/license
link](https://www.openstreetmap.org/copyright) with shared derived outputs.

WorldCover reports carry `CC BY 4.0`, the 2021 v200 dataset citation, and
`(c) ESA WorldCover project 2021 / Contains modified Copernicus Sentinel data
(2021) processed by ESA WorldCover consortium`. See the [official license and
citation](https://esa-worldcover.org/en/data-access). Kit changes the data's
projection, resolution and representation; preserve that attribution and
describe those modifications when sharing maps. Review source terms before
redistributing geographic inputs or derived outputs; Kit's software license
does not replace their licenses.
