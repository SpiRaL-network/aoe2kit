# swatch

`swatch` is the Go owner for Scenario Sandbox moving-pattern geometry.

It currently exposes the `pkg/geom` pattern registry as a CLI:

```sh
go run ./cmd/swatch patterns
go run ./cmd/swatch patterns list
go run ./cmd/swatch patterns validate
go run ./cmd/swatch patterns json radar_sweep
```

The durable contract is `[]Frame`, where each frame is a list of integer AoE2
map tiles bounded to `0..299`, with default patterns kept within the shared
swatch budget of 8-48 frames and at most 40 tiles per frame.
