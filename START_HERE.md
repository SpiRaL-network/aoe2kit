# START HERE

## Running inside an AI container

Use the container platform to select an operating-system and architecture
pair. Prefer these bootstrap paths in order.

1. **GitHub connector:** Find the latest published release tag in
   [chrae/aoe2kit](https://github.com/chrae/aoe2kit/releases), then find the
   newest successful run for that tag across the `release` and
   `refresh-release-artifacts` workflows. Use the connector's
   `download_workflow_artifact` action for the stable artifact named
   `kit-assets-<os>-<arch>` (for example, `kit-assets-linux-amd64`). Do not
   hardcode a workflow run ID or an artifact ID; these artifacts are refreshed
   monthly and expire after 90 days. For a refresh run, confirm its run summary
   names the current latest release tag. Extract the downloaded Actions
   artifact and select its standalone binary for this platform.

   Before executing the binary, use the GitHub connector to read `SHA256SUMS`
   from the same official release and compare the matching versioned binary's
   digest with the locally computed SHA-256. Then run `kit version --json` and
   confirm its version and commit match the release tag and its commit shown by
   GitHub. This offline checksum-and-commit check does not verify the
   cryptographic attestation. Finally run:

   ```sh
   ./kit doctor --sandbox
   ```

2. **Direct release download:** If the container can fetch GitHub release
   assets, download a compatible archive from the
   [latest release](https://github.com/chrae/aoe2kit/releases/latest). For
   example, Linux x86-64 uses
   [aoe2kit_linux_amd64.zip](https://github.com/chrae/aoe2kit/releases/latest/download/aoe2kit_linux_amd64.zip).
   Verify online with `gh attestation verify <file> -R chrae/aoe2kit` when the
   GitHub CLI is available. Otherwise, obtain `SHA256SUMS` from that release
   and compare the matching digest before executing. Extract, then run
   `kit version --json` and `kit doctor --sandbox`.

   If the container has Go but cannot use the prebuilt binary, download
   `aoe2kit_source-vendored.zip` from the same release and build offline with
   `go build -mod=vendor -o kit ./cmd/kit`. The vendored source archive
   includes its Go dependencies; no module-proxy access is needed.

3. **User attachment:** If neither connector artifact download nor direct
   release download works, ask the user to download and attach exactly one
   compatible binary archive from the official latest release. The user does
   not need to install Go or run commands.

### Known environments

Observed and tested on 2026-10-14:

| Environment | Verified capabilities |
| --- | --- |
| ChatGPT with the GitHub connector | Go 1.23.2 is present; shell network access and the web tool are unavailable. `download_workflow_artifact` can place `kit-assets-<os>-<arch>` under `/mnt/data`. |
| Claude.ai (Opus 5.5) | GitHub release assets are downloadable, but the Go module proxy is blocked. The vendored source archive builds offline with `-mod=vendor`; the prebuilt `linux_amd64` archive also works. |

Gemini, ChatGPT without the GitHub connector, and Codex cloud have not been
tested. Try the bootstrap routes above in order, then report which route worked
and the environment in a GitHub issue.

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
