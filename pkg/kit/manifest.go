package kit

import "encoding/json"

const Version = "0.1.0"

type Manifest struct {
	Name        string      `json:"name"`
	Version     string      `json:"version"`
	Neutrality  string      `json:"neutrality"`
	Runtime     RuntimeInfo `json:"runtime"`
	Docs        []string    `json:"docs"`
	Commands    []Command   `json:"commands"`
	Domains     []string    `json:"domains"`
	Limitations []string    `json:"limitations"`
}

type RuntimeInfo struct {
	DurableLanguage string `json:"durable_language"`
	PythonRequired  bool   `json:"python_required"`
	NetworkRequired bool   `json:"network_required"`
}

type Command struct {
	Name        string   `json:"name"`
	Summary     string   `json:"summary"`
	Subcommands []string `json:"subcommands,omitempty"`
}

func CurrentManifest() Manifest {
	return Manifest{
		Name:       "AoE2Kit",
		Version:    Version,
		Neutrality: "Project-neutral AoE2 tooling. Project facts belong in the bundle, not in shared kit code.",
		Runtime: RuntimeInfo{
			DurableLanguage: "Go",
			PythonRequired:  false,
			NetworkRequired: false,
		},
		Docs: []string{
			"README.md",
			"LICENSE",
			"NOTICE.md",
			"CONTRIBUTING.md",
			"SECURITY.md",
			"BRIEFING.md",
			"AUDITS.md",
			"START_HERE.md",
			"AI_GUIDE.md",
			"docs/GO_AOE2KIT.md",
			"docs/AOE_MS_REPLAY_API.md",
			"docs/REPLAY_GUIDE.md",
			"docs/REPLAY_CORPUS.md",
			"docs/DAT_ENGINE.md",
			"docs/DAT_AUTHORING.md",
			"docs/GOKU_DAT_AUDIT.md",
			"docs/SCENARIO_TOOLS.md",
			"docs/SCENARIO_WRITER.md",
			"docs/CBA.md",
			"docs/AOE2KIT_WRITE_SMOKE.md",
			"docs/XS_AUTHORING.md",
			"docs/XS_AUTHORING_TIER1.md",
			"docs/DAT_RECIPE_EXAMPLE.json",
			"docs/SCEN_RECIPE_EXAMPLE.json",
			"docs/SCEN_WRITE_SMOKE_RECIPE.json",
			"docs/CAMPAIGN_SPEC_EXAMPLE.json",
			"docs/REPLAY_CONTEXT_EXAMPLE.json",
			"docs/TELEMETRY_SCHEMA_EXAMPLE.json",
			"data/credits.json",
			"known_scenarios.json",
			"known_ais.json",
			"KIT_MANIFEST.json",
		},
		Commands: []Command{
			{Name: "kit", Summary: "AoE2Kit CLI source with profile-aware release packing, portability checks, and explicit credit/provenance reporting", Subcommands: []string{"version", "doctor", "docs", "summary", "inventory", "scan", "manifest", "credits", "verify", "pack", "portable-check", "project", "recipe", "release-check", "verify-run", "ci", "campaign", "roadmap", "facts", "identify", "collect", "player", "replay", "ai", "scen", "dat", "gfx", "fx", "xs", "xsdat", "cba", "mod", "swatch"}},
			{Name: "kit credits", Summary: "human and machine-readable attribution ledger for tools, docs, data, and upstream references that shaped the kit", Subcommands: []string{"notice", "--text", "--json"}},
			{Name: "kit docs", Summary: "documentation hygiene checks against the live command catalog", Subcommands: []string{"lint", "--text", "--json"}},
			{Name: "kit project", Summary: "project-neutral folder triage, snapshot manifests, snapshot diffs, and explicit artifact-lineage reports for source/reference/generated/evidence roles before and after an AI edits", Subcommands: []string{"inspect", "snapshot", "diff", "lineage", "--limit", "--out", "--manifest", "--input", "--output", "--tool", "--note", "--text", "--json"}},
			{Name: "kit recipe", Summary: "project-neutral reusable scenario and DAT recipe templates that expose the kit's native JSON patch shapes", Subcommands: []string{"list", "show", "--domain", "--recipe-only", "--text", "--json"}},
			{Name: "kit release-check", Summary: "composed pre-release structural gate for scenario, optional previous scenario diff, optional mod activation/package check, and optional pack", Subcommands: []string{"--previous", "--mod", "--status", "--pack", "--text"}},
			{Name: "kit verify-run", Summary: "contract-check a real replay against expected scenario identity, data-set identity, telemetry, chat, result, and render claims", Subcommands: []string{"--contract", "--text"}},
			{Name: "kit ci", Summary: "project-local scenario CI: scaffold a portable check config and run verify-run/sidecar-sync checks after playtests", Subcommands: []string{"init", "check"}},
			{Name: "kit campaign", Summary: "same-party/same-profile XS .xsdat campaign persistence generator using the engine-verified bare writer-scenario read convention", Subcommands: []string{"generate", "--out-dir", "--prefix", "--text"}},
			{Name: "kit roadmap", Summary: "code-owned CRUD/R/W/D capability matrix and dark-byte frontier ledger for scenario, DAT, and replay work", Subcommands: []string{"crud", "rwd", "dark", "--domain", "--text", "--json"}},
			{Name: "kit facts", Summary: "engine-fact ledger listing and schema/evidence validation; engine-verified facts are wired into lint/deploycheck findings", Subcommands: []string{"list", "check", "--include-provisional", "--domain", "--text"}},
			{Name: "kit identify", Summary: "scenario/replay fingerprinting plus AI loadout registry lookup", Subcommands: []string{"--register", "--family", "--description", "--registry", "--ai-registry", "--register-ai"}},
			{Name: "kit collect", Summary: "recursive scenario/replay fingerprint collection and registry growth", Subcommands: []string{"--registry", "--dry-run"}},
			{Name: "kit player", Summary: "AoE2 public web-API player career-stat lookup with short TTL cache", Subcommands: []string{"stats"}},
			{Name: "kit replay", Summary: "AoE2 recorded-game or replay-zip header inspection, consolidated summaries with opt-in map tile grids, trigger-graph fingerprinting/search/diffing, replay download, coverage/opaque-span mapping, frontier bucketing, default AI module-manifest extraction, effective game-data tail inspection, effective unit availability trees, template-vs-baseline data-mod detection, corpus summaries, structural diff-state probes, exact value scanning, header-anchor decoding, typed event timelines, object indexing/state cards, sync/combat lifecycle and corpse-replacement signals, desync sync-log parsing, filterable player event tables, authored carrier/sidecar probe readback, unified lobby/body/backlog chat transcripts, feedback-centered playtest reports, player profiles, factual stories, folder inbox reports, issue cards, telemetry validation, and unknown command mining", Subcommands: []string{"info", "summary", "graph", "triggers", "trigger-neighborhood", "diff-triggers", "fetch", "coverage", "opaque-spans", "frontier", "corpus", "opaque-clusters", "ai-manifest", "effective-data", "effective-units", "datamod-check", "diff-state", "scan-value", "header-anchors", "objects", "object-state", "object-shapes", "spawns", "lifecycle", "postgame", "camera", "viewlock", "health", "sync", "sync-log", "checksum-phase", "checksum-probe", "combat", "deaths", "player-series", "playtest", "carrier", "sidecar-sync", "xs-telemetry", "events", "actions", "player-events", "chat", "feedback", "story", "player-profile", "inbox", "telemetry", "issues", "unknowns"}},
			{Name: "kit ai", Summary: "AI script file/folder linting, diffing, content fingerprinting, and known-AI registry support", Subcommands: []string{"fingerprint", "lint", "diff"}},
			{Name: "kit scen", Summary: "current AoE2DE .aoe2scenario inspection, settings/strings/effects/references/delete-plans/glossary/idioms/analyze, trigger intelligence, structural byte-coverage accounting with explicit dark spans, describe, lint, embedded/deployed XS census, deploy-tree XS preflight, scenario and trigger-graph diffing, path-aligned field queries, filtered effect payload inspection and control-vs-control effect diffs, raw inflated-body dumping and byte-level diffing for byte-map calibration, region-context guessing, position-aware terrain inspection, DAT-backed art-palette usage, blank-root scenario creation with optional Outpost dummy starters, trigger/unit/map recipe patching, generic train-token shop-catalog recipe generation, raw-preserving trigger import with reference remapping, direct plan-gated scenario deletes, direct trigger/unit/variable/string reference disconnects, composed write-checks, and canonical editor-smoke recipe generation/application", Subcommands: []string{"blank", "info", "check", "describe", "settings", "field", "strings", "refs", "delete-plan", "delete", "disconnect", "effects", "diff-effects", "glossary", "idioms", "analyze", "triggers", "trigger-neighborhood", "audit-player-coverage", "mechanic", "trigger-flow", "units", "map", "terrain", "palette-usage", "regions", "coverage", "verify", "lint", "xs", "deploycheck", "diff", "diff-triggers", "bytediff", "write-check", "dump-body", "plan", "patch", "shop-catalog", "import-triggers", "smoke", "smoke-recipe"}},
			{Name: "kit dat", Summary: "current AoE2DE empires*.dat inspection, structured unit decoding, normalized SQLite export, graphics and effect/tech/unit authoring, reference-aware deletes, and replay-backed DAT semantics readback", Subcommands: []string{"facts", "info", "check", "roundtrip", "diff", "sprite", "semantic-priors", "sql", "unit-headers", "unit-header", "civs", "civ-patch", "spans", "graphics", "graphic", "graphic-create", "graphic-patch", "graphic-delete", "graphic-delta-delete", "graphic-angle-sound-delete", "palette", "effects", "effect", "effect-explain", "effect-create", "effect-patch", "effect-disable", "effect-delete", "effect-command-delete", "command-matrix", "techs", "tech", "tech-explain", "tech-create", "tech-patch", "tech-delete", "abilities", "ability", "ability-create", "ability-patch", "ability-disable", "ability-delete", "tech-tree", "tech-tree-connection-create", "tech-tree-connection-patch", "tech-tree-connection-delete", "units", "unit", "unit-create", "unit-delete", "unit-child-delete", "unit-header-task-delete", "availability", "availability-set", "terrain-restrictions", "terrain-restriction-patch", "terrains", "terrain", "terrain-patch", "sounds", "sound", "sound-create", "sound-patch", "sound-delete", "sound-item-delete", "player-colours", "player-colour-create", "player-colour-patch", "player-colour-delete", "random-maps", "patch-graphic", "patch-unit", "refs", "delete-plan", "delete", "disconnect", "codec-plan", "codec-patch", "semantics-pack", "semantics-readback", "plan", "patch"}},
			{Name: "kit gfx", Summary: "SLD sprite inspection and main-layer PNG export for visual asset teardown", Subcommands: []string{"info", "export", "--out", "--limit", "--text"}},
			{Name: "kit fx", Summary: "particle-effect authoring helpers: scaffold descriptors, bind DDS/PNG atlases to DAT graphics and unit slots, and lint particle asset triangles", Subcommands: []string{"new", "bind", "lint"}},
			{Name: "kit xs", Summary: "XS authoring helpers for named trigger-variable bridges, generated script_call shim recipe fragments, generated xsArray data modules, unattended engine-fact harness generation, and .xsdat sidecar inspection", Subcommands: []string{"bridge", "shims", "datagen", "harness", "inspect"}},
			{Name: "kit xsdat", Summary: "XS .xsdat sidecar decoding and optional expected-ledger assertion comparison", Subcommands: []string{"decode", "--types", "--ledger", "--text"}},
			{Name: "kit cba", Summary: "CBA-specific replay and ladder-analysis commands kept outside generic replay interpretation", Subcommands: []string{"balance", "trigger-razes", "trigger-spawns", "replay", "progression", "razes", "perf", "registry", "ingest", "archive", "axes", "export"}},
			{Name: "kit mod", Summary: "local mod folder packaging and activation inspection/check", Subcommands: []string{"check", "inspect", "--status"}},
			{Name: "kit swatch", Summary: "moving tile-pattern geometry", Subcommands: []string{"patterns list", "patterns validate", "patterns json"}},
		},
		Domains: []string{
			"aoe2scenario",
			"aoe2record",
			"dat",
			"dat_codec",
			"dat_effects",
			"dat_facts",
			"dat_techs",
			"dat_tech_tree",
			"dat_terrains",
			"dat_sounds",
			"dat_player_colours",
			"dat_random_maps",
			"dat_delete_plans",
			"gfx_sld",
			"fx_particles",
			"xs_authoring",
			"local_mod",
			"ai_files",
			"ai_content_fingerprints",
			"tile_geometry",
			"swatch_patterns",
			"trigger_graph_fingerprints",
			"replay_events",
			"replay_summary",
			"replay_fetch",
			"replay_corpus",
			"replay_ai_manifest",
			"replay_effective_gamedata",
			"replay_object_state",
			"replay_player_events",
			"replay_story",
			"replay_player_profile",
			"replay_inbox",
			"replay_issue_cards",
			"replay_telemetry",
			"roadmap_rwd_matrix",
			"roadmap_dark_frontier",
			"player_career_stats",
			"portable_checks",
			"recipe_templates",
			"scenario_shop_catalogs",
			"artifact_lineage",
			"release_checks",
		},
		Limitations: []string{
			"AoE2Kit reports structural verification explicitly; green structural checks are not in-engine load/render/play proof",
			"scenario patching currently supports append triggers, narrow existing-trigger edits, add/edit/remove unit recipes, terrain rectangle/circle/line edits, and XS entry-script attachment plus script_call effects",
			"dat patching targets current AoE2DE dat layouts; graphic/unit new-record creation is supported, but broader DAT table creation remains scoped to explicitly implemented record families",
			"trigger_graph is the strongest replay fingerprint; when it is unavailable, identify falls back to a labeled terrain fingerprint",
			"modern DE replays expose AI file references, not AI script content; use kit ai fingerprint on the mod files for content integrity",
			"replay event parsing is intentionally layered: typed chat/flare/telemetry events are facts, named regions come from optional context JSON, and untyped actions remain counted/raw-preserved until decoded",
			"replay player profiles report event-table facts, not psychology; skill, intent, attention, combat lifecycle, and economy state remain outside the current claim set",
			"full deterministic v68 trigger-section layout walking is not complete; the trigger_graph locator still uses the validated dense-default/string-cluster heuristic with a labeled binary-structure scan fallback",
		},
	}
}

func ManifestJSON() ([]byte, error) {
	return json.MarshalIndent(CurrentManifest(), "", "  ")
}
