# Contributing

AoE2Kit is a Go-first toolkit for AoE2DE scenario, replay, data-mod, and local
mod work. Durable tooling belongs in Go. If you reach for a Python or shell
script for a durable workflow, record the missing Kit capability instead of
quietly adding another side tool.

## Development Loop

```sh
go build -o kit ./cmd/kit
GOMAXPROCS=2 go test -p 2 ./... -count=1
./kit docs lint
./kit portable-check . --profile public
```

Use `GOMAXPROCS=2` and `-p 2` on ordinary machines. Several commands inspect
large game files, so memory discipline matters.

## Verification Rules

- Structural verification is not engine verification.
- Roundtrip-identical bytes prove codec preservation, not gameplay behavior.
- Engine behavior claims need a real game/editor run or an explicitly labeled
  fixture.
- Unknown bytes, fields, commands, and semantics must remain labeled unknown
  until evidence closes them.

## Attribution

Credits are first-class. If a tool, project, document, community source, or
person materially shapes a feature, add it to `data/credits.json` and regenerate
`NOTICE.md`. Check the ledger with:

```sh
kit credits --text
kit credits notice > NOTICE.md
```

## Generated Artifacts

Refresh the manifest after command/doc changes:

```sh
./kit manifest > KIT_MANIFEST.json
```

API docs are generated from the live command catalog. Regenerate them when
command surfaces change.
