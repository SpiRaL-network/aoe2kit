# XS Authoring Tier 1

Verification status: structure-verified, not engine-verified by generation
alone.

These commands turn the GoKu-style production pattern into generic AoE2Kit
authoring primitives: named trigger variables, generated trigger shims for XS
functions, and generated xsArray data modules.

## Named Variable Bridge

```sh
./kit xs bridge variables.json --out-dir build/xs_bridge
```

Minimal input can be a raw name-to-id map:

```json
{
  "p1_level": 12,
  "p1_mana": 13
}
```

Full input may include read/write intent for linting:

```json
{
  "module_name": "quest_variables",
  "variables": {
    "p1_level": 12,
    "p1_mana": 13,
    "gate_open": 40
  },
  "xs_reads": ["p1_level", "p1_mana"],
  "trigger_writes": ["p1_level"],
  "xs_writes": ["gate_open"],
  "trigger_reads": ["gate_open"]
}
```

Outputs:

- `<module_name>.xs`: `extern int xsVariableN` declarations,
  `extern const int` named constants, and `get_name()` wrappers around
  `xsTriggerVariable(N)`.
- `trigger_variables.json`: the same ids as named trigger-layer definitions for
  scenario builders.
- `lint.json`: read/write mismatch report.

The generator intentionally refuses invalid XS identifiers and duplicate ids.
The current bridge generates read helpers; writing scenario variables remains a
trigger-layer responsibility until an XS write primitive is locally proven.

## Script Call Shims

```sh
./kit xs shims resources/_common/xs --out build/xs_shims.recipe.json
```

This scans `.xs` files, finds function declarations, and emits a scenario recipe
fragment with one trigger per parameterless function:

```json
{
  "triggers": [
    {
      "op": "add_trigger",
      "name": "XS Shim Boot",
      "enabled": false,
      "looping": false,
      "effects": [
        { "op": "script_call", "message": "Boot();" }
      ]
    }
  ]
}
```

By default parameterized functions are inventoried but skipped because trigger
`script_call` effects are safest as parameterless calls. Use `--include-params`
only when the runtime intentionally supplies defaults and the scenario designer
accepts that risk.

## XS Data Modules

```sh
./kit xs datagen arrays.json --out resources/_common/xs/zone_arrays.xs
```

Input:

```json
{
  "function": "smc_generated_init_arrays",
  "arrays": [
    { "name": "zone_ids", "type": "int", "values": [1, 2, 3] },
    { "name": "zone_names", "type": "string", "values": ["north", "south"] }
  ]
}
```

Output follows the working GoKu shape:

```xs
extern int zone_ids = -1;

void smc_generated_init_arrays() {
    zone_ids = xsArrayCreateInt(3, 0, "zone_ids");
    xsArraySetInt(zone_ids, 0, 1);
}
```

The command verifies array names, supported value types, and generated set-counts
against the JSON input. It does not prove that the target scenario called the
init function; that is an engine-run assertion.
