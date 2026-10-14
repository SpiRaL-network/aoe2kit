# START HERE

## Trust and execution

Release binaries are built by GitHub Actions from this public source and
published with `SHA256SUMS` and signed build-provenance attestations. Online
verification is available with `gh attestation verify`; an offline check can
compare the GitHub artifact digest with the local SHA-256 and compare
`kit version --json`'s commit with the release tag's commit.

Scenario-authoring commands read supplied inputs and write requested outputs;
they do not modify the input files. Local authoring does not require network
access, installation, or system changes. Explicit network-facing commands,
such as replay/player fetches, map-data acquisition, and release downloads, use
the network. Whether a binary is executed depends on the user's request and
the environment's execution rules; this section describes trust properties,
not an instruction to run it.

## Running inside an AI container

The container platform determines the operating-system and architecture pair.
The following are available bootstrap paths, in preference order.

1. **GitHub connector:** The latest published release tag is listed in
   [chrae/aoe2kit](https://github.com/chrae/aoe2kit/releases). The associated
   run is the newest successful one for that tag across the `release` and
   `refresh-release-artifacts` workflows. Its stable artifact is named
   `kit-assets-<os>-<arch>` (for example, `kit-assets-linux-amd64`); the
   connector's `download_workflow_artifact` action returns its contents.
   Workflow and artifact IDs are time-sensitive: artifacts are refreshed
   monthly and expire after 90 days. A refresh-run summary names the release
   tag it refreshed. The downloaded artifact contains a standalone binary for
   the selected platform.

   The connector can read `SHA256SUMS` from the same official release; its
   matching versioned binary digest can be compared with a locally computed
   SHA-256. `kit version --json` reports the embedded version and commit for
   comparison with the release tag and commit shown by GitHub. This
   checksum-and-commit check does not verify the cryptographic attestation.
   The sandbox readiness report is available as:

   ```sh
   ./kit doctor --sandbox
   ```

2. **Direct release download:** When the container can fetch GitHub release
   assets, a compatible archive is available from the
   [latest release](https://github.com/chrae/aoe2kit/releases/latest). For
   example, Linux x86-64 uses
   [aoe2kit_linux_amd64.zip](https://github.com/chrae/aoe2kit/releases/latest/download/aoe2kit_linux_amd64.zip).
   Online verification is available with
   `gh attestation verify <file> -R chrae/aoe2kit` when the GitHub CLI is
   available. Otherwise, `SHA256SUMS` from that release provides a matching
   digest to compare. The version and sandbox reports are `kit version --json`
   and `kit doctor --sandbox`.

   When the container has Go but cannot use the prebuilt binary, the
   `aoe2kit_source-vendored.zip` archive from the same release is an offline
   build option: `go build -mod=vendor -o kit ./cmd/kit`. The archive includes
   its Go dependencies; no module-proxy access is needed.

3. **User attachment:** If neither connector artifact download nor direct
   release download is available, a compatible binary archive from the
   official latest release can be attached by the user. This route does not
   require the user to install Go or run commands.

### Known environments

Observed and tested on 2026-10-14:

| Environment | Verified capabilities |
| --- | --- |
| ChatGPT with the GitHub connector | Go 1.23.2 is present; shell network access and the web tool are unavailable. `download_workflow_artifact` can place `kit-assets-<os>-<arch>` under `/mnt/data`. |
| Claude.ai (Opus 5.5) | GitHub release assets are downloadable, but the Go module proxy is blocked. The vendored source archive builds offline with `-mod=vendor`; the prebuilt `linux_amd64` archive also works. |

Gemini, ChatGPT without the GitHub connector, and Codex cloud have not been
tested. Results from those environments can be reported in a GitHub issue.

## Source-build example

This is AoE2Kit: a portable Go workbench for Age of Empires II: Definitive
Edition scenario, data-mod, local-mod, replay, and sandbox work.

It is meant to travel inside a zip beside the scenario/mod/dat/replay artifacts
you need to work on. The kit is project-neutral; project facts live in the
bundle around it.

A possible local build and inspection sequence is:

```sh
go build -o kit ./cmd/kit
./kit doctor
./kit summary .
./kit inventory
./kit project inspect . --limit 50 --text
./kit verify .
```

AI-oriented conventions are described in:

```text
AI_GUIDE.md
```

`docs/GO_AOE2KIT.md` is a command reference.
