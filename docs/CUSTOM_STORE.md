# Custom Store

AoE2Kit includes a generic recipe for stores attached to many addressable
object instances. It is deliberately content-neutral: the item names, prices,
and effects are placeholders for the scenario author to replace.

Generate the Front Towers-sized fixture from a fresh scenario:

```text
kit rpg custom-store --out-dir build/customstore-440 --unit 236 --instances 440 --players 8 --xs-check /path/to/xs-check --text
```

For a mechanism-isolated run, restrict store setup and token polling to the
human participant while retaining the eight-player object scale:

```text
kit rpg custom-store --out-dir build/customstore-isolated --unit 236 --instances 440 --players 8 --active-players 1 --text
```

`--active-players` is a comma-separated player list. If omitted, all configured
players remain participants for multiplayer/scale tests. Non-participants keep
their addressable objects but receive no store buttons and are never scanned by
the runtime, preventing AI purchases from polluting a single-player probe.

The optional `--xs-check` flag runs an external XS parser against the generated
module before the scenario is emitted. The same checker can be selected with
`AOE2KIT_XS_CHECK`; when neither is available the report says
`xs_parse_check=unavailable` rather than claiming parser verification. The
gate treats diagnostic text containing `Error` as failure even when a checker
exits zero (the current `xs-check` behavior).

The command above is the Front Towers-sized worked example: 440 DAT unit-236
instances (`WCTW4`), 55 for each of eight players. Any unit constant and
evenly divisible instance count can be supplied. Each instance has a unique
scenario reference ID. One setup trigger applies the same two train-token
buttons to every object; the number of store triggers therefore does not grow
with the number of objects.

Every player receives a barracks starter. This is intentional: it keeps the
player alive without relying on the engine to inject a town center when a
player starts with no units.

Every player also starts with `100000` food, wood, gold, stone, and trade goods.
That budget is deliberately generous: this fixture tests store interaction and
position mapping, not economy balance, so a human can buy many placeholder
tokens immediately without preparing the scenario first.

The two demo buttons have real costs: the upgrade token costs 100 gold and the
inspect token costs 100 wood. A successful purchase draws one of two distinct
bounded halos around the resolved object: the upgrade token draws a FLAME1
ring, while the inspect token draws a Flower (unit 1366) ring. Fire retires by
setting hit points to zero. Flowers are class-14 Gaia artwork, so they retire
by teleporting into a bounded Gaia kill zone and letting one variable-gated
`remove_object` sweep per second clear that zone. This deliberately exercises
both generic retirement paths without scenario-specific effects.

The embedded XS module builds a bounded table of runtime object IDs and one
state cell per object. When a token is trained, it finds the nearest object
owned by that player, updates only that object's state, removes the token, and
exposes the last purchase and neighbor state through trigger variables and chat
telemetry.

The embedded XS also writes a bounded `.xsdat` sidecar named after the
scenario. `A2K_CUSTOM_STORE.types` is the portable write-order contract. The
header is created and closed once, and every purchase row is appended and
closed immediately; a run that ends before the 60-second footer still retains
its purchase evidence instead of depending on process shutdown.
The stream contains a header with total and per-player object counts, one
`purchase` row per token consumed (including unresolved position mappings),
and an `end` footer with the array high-water mark and integrity value
`424242`. Decode it with:

```text
kit xsdat decode "A2K Custom Store 440.xsdat" --schema customstore --text
```

Each purchase row records the token position, resolved object ID, neighboring
object ID, and before/after state values. The fixture closes the sidecar at 60
game seconds; chat remains additive for live visual debugging.

This is a mechanism fixture, not an engine guarantee. Before adopting it in a
scenario, verify in AoE2DE that the renamed menu appears on a selected object
and that the token's spawn position reliably identifies its source object. The
fixture reports those as open engine checks rather than silently treating
structural output as proven behavior.
