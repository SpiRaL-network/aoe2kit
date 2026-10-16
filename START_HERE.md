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

## Preview a map

`kit preview` renders flat maps, minimaps, sprites, textured terrain, and
supported particle units. `--crop x1,y1,x2,y2` and `--scale N` select a view;
`--sprites --textures --shadows --mood scenario` enables the installed-art
preview stack. Use `--out file.png` for the local output.

Local agents can use `--game-dir GAME`, `AOE2KIT_GAME_DIR`, or Steam discovery.
Kit reads the user's DAT, sprites, textures, palettes, blend masks, mood
definitions and particle atlases at runtime; none are shipped. In containers
without those files, use flat previews, or explicitly supplied local assets.
`kit preview assets` lists missing sprite dependencies without fetching them.

The preview is not an engine screenshot: water edges differ, mood grading is
approximate, beach and road brightness is off, and geometric elevation is not
rendered. See [Visual Previews](docs/PREVIEW.md) for commands and limits.

## Running inside an AI container

The container platform determines the operating-system and architecture pair.
The following are available bootstrap paths, in preference order.

1. **GitHub connector:** The latest published release tag is listed in
   [chrae/aoe2kit](https://github.com/chrae/aoe2kit/releases). The associated
   run is a successful `release` run for that exact tag, or a successful
   `refresh-release-artifacts` run whose summary names that tag. Do not select
   a run only because it is the newest overall. Download the artifact named
   `kit-assets-<os>-<arch>` for the container's platform:

   | Platform | Artifact name | Standalone file |
   | --- | --- | --- |
   | Linux x86-64 | `kit-assets-linux-amd64` | `aoe2kit_<tag>_linux_amd64` |
   | Linux ARM64 | `kit-assets-linux-arm64` | `aoe2kit_<tag>_linux_arm64` |
   | Windows x86-64 | `kit-assets-windows-amd64` | `aoe2kit_<tag>_windows_amd64.exe` |
   | macOS ARM64 | `kit-assets-darwin-arm64` | `aoe2kit_<tag>_darwin_arm64` |
   | macOS x86-64 | `kit-assets-darwin-amd64` | `aoe2kit_<tag>_darwin_amd64` |

   The connector's `download_workflow_artifact` action returns the artifact
   contents, including the standalone binary and platform archive. Workflow
   and artifact IDs are time-sensitive: artifacts are refreshed monthly and
   expire after 90 days. A refresh-run summary names the release tag it
   refreshed.

   For verification, read `SHA256SUMS` from that tag's official release and
   select the line whose filename exactly matches the versioned standalone
   name above (release entries begin `./`). Compare that digest with a
   locally computed SHA-256 of the downloaded file. The release also has
   unversioned aliases; the versioned name avoids ambiguity. `kit version
   --json` reports the embedded version and commit for comparison with the
   release tag and commit shown by GitHub. This checksum-and-commit check does
   not verify the cryptographic attestation.
   The sandbox readiness report is available as:

   ```sh
   ./kit doctor --sandbox
   ```

2. **Direct release download:** When the container can fetch GitHub release
   assets, choose the platform archive from the
   [latest release](https://github.com/chrae/aoe2kit/releases/latest):
   `aoe2kit_<os>_<arch>.zip` (for example,
   [Linux x86-64](https://github.com/chrae/aoe2kit/releases/latest/download/aoe2kit_linux_amd64.zip)).
   The release also includes standalone binaries named
   `aoe2kit_<tag>_<os>_<arch>` (with `.exe` on Windows) and unversioned
   aliases. The platform table above gives the exact platform mapping.
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

## Feedback

GitHub Issues is the public channel for bug reports, capability gaps, and
documentation problems. A useful report identifies the Kit version, commands
and evidence, and its verification level: ran, structural, or engine. Agents
prepare a report for their human to review before posting; when they cannot
post, they can provide a ready-to-file issue body instead. See the same
protocol in [AI_GUIDE.md](AI_GUIDE.md) and [CONTRIBUTING.md](CONTRIBUTING.md).

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
