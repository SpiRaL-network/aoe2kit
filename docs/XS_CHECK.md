# Optional XS checker

AoE2Kit can run an independent external XS checker in addition to its own
source analysis. The checker is optional for foreign XS, but strict checks for
generated XS fail closed when it is unavailable.

The reference checker is [Divy1211/xs-check](https://github.com/Divy1211/xs-check),
version 0.2.30. That project is licensed under GPL-3.0. AoE2Kit therefore does
not redistribute its executable and does not link to it. Install or build the
checker separately at `~/.local/share/aoe2kit/xs-check` (on Windows,
`%USERPROFILE%\.local\share\aoe2kit\xs-check.exe`); Kit discovers and
SHA-verifies that location automatically. `XDG_DATA_HOME` relocates the
`~/.local/share` portion. `AOE2KIT_XS_CHECK=/path/to/xs-check` or
`--xs-check /path/to/xs-check` remains available for explicit installations.

The reference build's SHA256 is
`b35a5fecd8512b5ac9ec5111ea5848f1f92c20958bd9d13fbe47a5f8c1149d05` for the
Linux release asset `xs-check` and
`dd22d0755aca35d7ad5d77989cda2b98c0caefd4a43a1d37570585e9ff53ce27` for the
Windows release asset `xs-check.exe`. A binary you
build yourself may have a different digest; explicit paths (`--xs-check` or
`AOE2KIT_XS_CHECK`) are accepted regardless of digest.

For generated XS, use the default strict gate:

```text
kit scen xs scenario.aoe2scenario --strict-create-results --xs-check /path/to/xs-check
```

`--allow-missing-xs-check` is an explicit inspection escape hatch and should
not be used for a release or deployment artifact.
