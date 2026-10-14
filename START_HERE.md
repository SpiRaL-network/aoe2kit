# START HERE

## Running inside an AI container

First identify the container platform and try to fetch the matching stable
archive from the [latest release](https://github.com/chrae/aoe2kit/releases/latest).
The Linux x86-64 download is
[aoe2kit_linux_amd64.zip](https://github.com/chrae/aoe2kit/releases/latest/download/aoe2kit_linux_amd64.zip).
Other stable assets are named `aoe2kit_linux_arm64.zip`,
`aoe2kit_windows_amd64.zip`, `aoe2kit_darwin_arm64.zip`,
`aoe2kit_darwin_amd64.zip`, and `aoe2kit_source-vendored.zip`.

If this container cannot fetch and unpack the archive, ask the user for exactly
one action, using the compatible asset URL:

> Please download <link> and attach it here.

Once the archive is available, the AI should unpack it, select the `kit`
executable, and run:

```sh
./kit version
./kit doctor --sandbox
```

The user does not need to install Go or run commands.

## Running from source

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
