# START HERE

This is AoE2Kit: a portable Go workbench for Age of Empires II: Definitive
Edition scenario, data-mod, local-mod, replay, and sandbox work.

It is meant to travel inside a zip beside the scenario/mod/dat/replay artifacts
you need to work on. The kit is project-neutral; project facts live in the
bundle around it.

Run these first, in order:

```sh
go build -o kit ./cmd/kit
./kit doctor
./kit summary .
./kit inventory
./kit project inspect . --limit 50 --text
./kit verify .
```

Then read:

```text
AI_GUIDE.md
```

Use `docs/GO_AOE2KIT.md` only as command reference when needed.
