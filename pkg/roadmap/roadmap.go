package roadmap

import (
	"fmt"
	"sort"
	"strings"

	"aoe2kit/pkg/aoe2"
	"aoe2kit/pkg/enginefacts"
)

const Version = "2026-09-27"

type MatrixReport struct {
	Version      string                 `json:"version"`
	Domain       string                 `json:"domain"`
	Method       string                 `json:"method"`
	Verification aoe2.VerificationClaim `json:"verification"`
	Summary      MatrixSummary          `json:"summary"`
	Rows         []RWDRow               `json:"rows"`
	Notes        []string               `json:"notes,omitempty"`
}

type MatrixSummary struct {
	Domains       []string `json:"domains"`
	Rows          int      `json:"rows"`
	ReadFull      int      `json:"read_full"`
	WriteFull     int      `json:"write_full"`
	CreateFull    int      `json:"create_full"`
	DeleteFull    int      `json:"delete_full"`
	PartialRows   int      `json:"partial_rows"`
	Unsupported   int      `json:"unsupported"`
	NotApplicable int      `json:"not_applicable"`
}

type CRUDReport struct {
	Version      string                 `json:"version"`
	Domain       string                 `json:"domain"`
	Method       string                 `json:"method"`
	Verification aoe2.VerificationClaim `json:"verification"`
	Summary      CRUDSummary            `json:"summary"`
	Rows         []CRUDRow              `json:"rows"`
	Notes        []string               `json:"notes,omitempty"`
}

type CRUDSummary struct {
	Domains             []string `json:"domains"`
	Rows                int      `json:"rows"`
	CreateFull          int      `json:"create_full"`
	ReadFull            int      `json:"read_full"`
	UpdateFull          int      `json:"update_full"`
	DeleteFull          int      `json:"delete_full"`
	PartialCapabilities int      `json:"partial_capabilities"`
	Unsupported         int      `json:"unsupported"`
	NotApplicable       int      `json:"not_applicable"`
	NeedsEngineRun      int      `json:"needs_engine_run"`
}

type RWDRow struct {
	Domain       string `json:"domain"`
	Section      string `json:"section"`
	Read         string `json:"read"`
	Write        string `json:"write"`
	Create       string `json:"create"`
	Delete       string `json:"delete"`
	Verification string `json:"verification"`
	Evidence     string `json:"evidence"`
	Next         string `json:"next"`
}

type CRUDRow struct {
	Domain       string         `json:"domain"`
	Section      string         `json:"section"`
	Create       CRUDCapability `json:"create"`
	Read         CRUDCapability `json:"read"`
	Update       CRUDCapability `json:"update"`
	Delete       CRUDCapability `json:"delete"`
	Verification string         `json:"verification"`
	Evidence     string         `json:"evidence"`
	Next         string         `json:"next"`
}

type CRUDCapability struct {
	Status string `json:"status"`
	Mode   string `json:"mode,omitempty"`
	Detail string `json:"detail,omitempty"`
}

type DarkReport struct {
	Version      string                 `json:"version"`
	Domain       string                 `json:"domain"`
	Method       string                 `json:"method"`
	Verification aoe2.VerificationClaim `json:"verification"`
	Summary      DarkSummary            `json:"summary"`
	Frontiers    []DarkFrontier         `json:"frontiers"`
	Notes        []string               `json:"notes,omitempty"`
}

type DarkSummary struct {
	Domains   []string `json:"domains"`
	Frontiers int      `json:"frontiers"`
	Priority1 int      `json:"priority_1"`
	Priority2 int      `json:"priority_2"`
	Priority3 int      `json:"priority_3"`
}

type DarkFrontier struct {
	Domain       string `json:"domain"`
	Region       string `json:"region"`
	Priority     int    `json:"priority"`
	Known        string `json:"known"`
	Unknown      string `json:"unknown"`
	Tools        string `json:"tools"`
	Next         string `json:"next"`
	Verification string `json:"verification"`
}

func RWDMatrix(domain string) (MatrixReport, error) {
	filter, err := normalizeDomain(domain)
	if err != nil {
		return MatrixReport{}, err
	}
	rows := filterRWDRows(allRWDRows(), filter)
	report := MatrixReport{
		Version:      Version,
		Domain:       domainLabel(filter),
		Method:       "code-owned capability ledger; statuses describe implemented Kit surfaces, not promised engine behavior",
		Verification: aoe2.StructureVerification(true),
		Summary:      summarizeRWD(rows),
		Rows:         rows,
		Notes: []string{
			"full means a practical end-to-end surface exists for the named section; it does not mean every unknown subfield is semantically decoded",
			"partial means the Kit can perform useful, verified operations but still has known unsupported fields, helpers, or delete/reference cases",
			"recorded-game files are analysis artifacts; write/create/delete are intentionally not modeled as authoring operations",
		},
	}
	return report, nil
}

func CRUDMatrix(domain string) (CRUDReport, error) {
	filter, err := normalizeDomain(domain)
	if err != nil {
		return CRUDReport{}, err
	}
	rows := crudRows(filterRWDRows(allRWDRows(), filter))
	report := CRUDReport{
		Version:      Version,
		Domain:       domainLabel(filter),
		Method:       "code-owned CRUD capability ledger derived from the R/W/D source rows; update is the authoring write surface",
		Verification: aoe2.StructureVerification(true),
		Summary:      summarizeCRUD(rows),
		Rows:         rows,
		Notes: []string{
			"CRUD full means a practical end-to-end Kit surface exists for that operation and section; it does not imply every subfield is semantically decoded",
			"update is named explicitly here because the older roadmap called the same operation write",
			"delete modes matter: fixed-id game data often needs semantic disable, tombstone, child-row rebuild, or guarded reference cleanup instead of physical compaction",
			"structure_verified_not_engine_verified means the Kit can parse/write/read back the artifact structure; live DE/editor runs remain the rendering oracle",
		},
	}
	return report, nil
}

func DarkBytes(domain string) (DarkReport, error) {
	filter, err := normalizeDomain(domain)
	if err != nil {
		return DarkReport{}, err
	}
	frontiers := filterDarkFrontiers(allDarkFrontiers(), filter)
	report := DarkReport{
		Version:      Version,
		Domain:       domainLabel(filter),
		Method:       "frontier ledger from decoded replay/scenario/DAT work; unknowns stay named instead of becoming claims",
		Verification: aoe2.StructureVerification(true),
		Summary:      summarizeDark(frontiers),
		Frontiers:    frontiers,
		Notes: []string{
			"priority 1 frontiers are the next best work for replay transparency or authoring completeness",
			"engine_verified evidence requires a live DE/editor run or a replay/sidecar produced by one",
		},
	}
	return report, nil
}

func Domains() []string {
	return []string{"all", "scenario", "dat", "replay"}
}

func normalizeDomain(domain string) (string, error) {
	domain = strings.ToLower(strings.TrimSpace(domain))
	if domain == "" {
		domain = "all"
	}
	switch domain {
	case "all", "scenario", "scen", "aoe2scenario":
		if domain == "scen" || domain == "aoe2scenario" {
			return "scenario", nil
		}
		return domain, nil
	case "dat", "data", "datamod":
		return "dat", nil
	case "replay", "record", "aoe2record":
		return "replay", nil
	default:
		return "", fmt.Errorf("unknown roadmap domain %q; expected one of %s", domain, strings.Join(Domains(), ", "))
	}
}

func domainLabel(filter string) string {
	if filter == "" {
		return "all"
	}
	return filter
}

func filterRWDRows(rows []RWDRow, filter string) []RWDRow {
	if filter == "" || filter == "all" {
		return rows
	}
	out := make([]RWDRow, 0, len(rows))
	for _, row := range rows {
		if row.Domain == filter {
			out = append(out, row)
		}
	}
	return out
}

func filterDarkFrontiers(frontiers []DarkFrontier, filter string) []DarkFrontier {
	if filter == "" || filter == "all" {
		return frontiers
	}
	out := make([]DarkFrontier, 0, len(frontiers))
	for _, frontier := range frontiers {
		if frontier.Domain == filter {
			out = append(out, frontier)
		}
	}
	return out
}

func summarizeRWD(rows []RWDRow) MatrixSummary {
	domainSet := map[string]bool{}
	var summary MatrixSummary
	for _, row := range rows {
		domainSet[row.Domain] = true
		summary.Rows++
		countRWDStatus(row.Read, &summary.ReadFull, &summary)
		countRWDStatus(row.Write, &summary.WriteFull, &summary)
		countRWDStatus(row.Create, &summary.CreateFull, &summary)
		countRWDStatus(row.Delete, &summary.DeleteFull, &summary)
	}
	summary.Domains = sortedKeys(domainSet)
	return summary
}

func summarizeCRUD(rows []CRUDRow) CRUDSummary {
	domainSet := map[string]bool{}
	var summary CRUDSummary
	for _, row := range rows {
		domainSet[row.Domain] = true
		summary.Rows++
		countCRUDCapability(row.Create, &summary.CreateFull, &summary)
		countCRUDCapability(row.Read, &summary.ReadFull, &summary)
		countCRUDCapability(row.Update, &summary.UpdateFull, &summary)
		countCRUDCapability(row.Delete, &summary.DeleteFull, &summary)
		if strings.Contains(row.Verification, "not_engine_verified") {
			summary.NeedsEngineRun++
		}
	}
	summary.Domains = sortedKeys(domainSet)
	return summary
}

func countRWDStatus(status string, fullCounter *int, summary *MatrixSummary) {
	switch statusPrefix(status) {
	case "full":
		(*fullCounter)++
	case "partial":
		summary.PartialRows++
	case "unsupported":
		summary.Unsupported++
	case "n/a":
		summary.NotApplicable++
	}
}

func countCRUDCapability(capability CRUDCapability, fullCounter *int, summary *CRUDSummary) {
	switch capability.Status {
	case "full":
		(*fullCounter)++
	case "partial":
		summary.PartialCapabilities++
	case "unsupported":
		summary.Unsupported++
	case "n/a":
		summary.NotApplicable++
	}
}

func summarizeDark(frontiers []DarkFrontier) DarkSummary {
	domainSet := map[string]bool{}
	var summary DarkSummary
	for _, frontier := range frontiers {
		domainSet[frontier.Domain] = true
		summary.Frontiers++
		switch frontier.Priority {
		case 1:
			summary.Priority1++
		case 2:
			summary.Priority2++
		default:
			summary.Priority3++
		}
	}
	summary.Domains = sortedKeys(domainSet)
	return summary
}

func statusPrefix(status string) string {
	if i := strings.IndexByte(status, ':'); i >= 0 {
		return strings.ToLower(strings.TrimSpace(status[:i]))
	}
	fields := strings.Fields(strings.ToLower(status))
	if len(fields) == 0 {
		return ""
	}
	return fields[0]
}

func crudRows(rows []RWDRow) []CRUDRow {
	out := make([]CRUDRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, CRUDRow{
			Domain:       row.Domain,
			Section:      row.Section,
			Create:       crudCapability("create", row.Create),
			Read:         crudCapability("read", row.Read),
			Update:       crudCapability("update", row.Write),
			Delete:       crudCapability("delete", row.Delete),
			Verification: row.Verification,
			Evidence:     row.Evidence,
			Next:         row.Next,
		})
	}
	return out
}

func crudCapability(operation, raw string) CRUDCapability {
	status := statusPrefix(raw)
	capability := CRUDCapability{
		Status: status,
		Mode:   crudMode(operation, status, raw),
		Detail: statusDetail(raw),
	}
	if capability.Detail == "" {
		capability.Detail = raw
	}
	return capability
}

func statusDetail(raw string) string {
	if i := strings.IndexByte(raw, ':'); i >= 0 {
		return strings.TrimSpace(raw[i+1:])
	}
	return ""
}

func crudMode(operation, status, raw string) string {
	lower := strings.ToLower(raw)
	switch status {
	case "n/a":
		return "not_an_authoring_surface"
	case "unsupported":
		return "ledger_only"
	}
	switch operation {
	case "read":
		if status == "full" {
			return "inspector"
		}
		return "partial_inspector"
	case "create":
		if strings.Contains(lower, "clone") {
			return "clone_or_append"
		}
		if strings.Contains(lower, "fixed") {
			return "fixed_slot_create"
		}
		if strings.Contains(lower, "recipe") {
			return "recipe_create"
		}
		if status == "full" {
			return "direct_create"
		}
		return "partial_create"
	case "update":
		if strings.Contains(lower, "recipe") || strings.Contains(lower, "patch") {
			return "recipe_patch"
		}
		if strings.Contains(lower, "replacement") {
			return "replacement_patch"
		}
		return "guarded_update"
	case "delete":
		hasPhysical := strings.Contains(lower, "physical")
		hasSemantic := strings.Contains(lower, "semantic") || strings.Contains(lower, "tombstone") || strings.Contains(lower, "disable")
		hasChildRows := strings.Contains(lower, "child-row") || strings.Contains(lower, "row delete") || strings.Contains(lower, "row deletes") || strings.Contains(lower, "row removal")
		switch {
		case hasPhysical && hasSemantic:
			return "guarded_physical_or_semantic"
		case hasChildRows:
			return "child_row_rebuild"
		case hasSemantic:
			return "semantic_delete"
		case hasPhysical:
			return "guarded_physical_delete"
		case strings.Contains(lower, "replacement"):
			return "replacement_delete"
		}
		if status == "full" {
			return "direct_delete"
		}
		return "guarded_delete"
	default:
		return status
	}
}

func sortedKeys(values map[string]bool) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func allRWDRows() []RWDRow {
	return []RWDRow{
		{
			Domain:       "scenario",
			Section:      "trigger graph",
			Read:         "full: triggers, effects, conditions, messages, display order, variables, fingerprints",
			Write:        "partial: append/copy triggers, edit trigger settings, append/remove/clear/replace child effects and conditions, type-selected and message-text child-row cleanup, generated-system prefix cleanup, direct trigger-reference disconnects, expanded effect recipes for diplomacy/sound/view/object mutators/build/train/research/control, and add/rename/remove/tombstone variables through recipes",
			Create:       "full: trigger/effect/condition/variable creation plus cloned trigger creation through recipe model",
			Delete:       "partial: plan-gated direct deletes for triggers/effects/conditions/variables, effect-type/condition-type child-row bulk deletes with trigger scopes, effect-text-prefix/effect-text-contains deletes for raw-message cleanup, system-prefix generated-bundle deletes, trigger-name/trigger-prefix/trigger-contains and variable-name/variable-prefix/variable-contains delete plans, atomic remove_triggers batches for generated blocks with internal trigger-control refs, cleanup recipes/commands for referenced triggers/placed units/variables/strings, direct reference disconnects, trigger-control reference guards/rewrites, script_call semantic delete warnings, physical unreferenced variable removal, batch generated-block/state removal, and reference-guarded trigger/variable tombstones",
			Verification: "structure_verified_not_engine_verified",
			Evidence:     "kit scen triggers/effects/refs/delete-plan/delete/disconnect/analyze/patch plus writer roundtrip tests",
			Next:         "add broader multi-op dependency simulation before deleting trigger rows",
		},
		{
			Domain:       "scenario",
			Section:      "units and objects",
			Read:         "full: unit census, ownership, locations, named/captioned objects",
			Write:        "partial: add/edit/copy/move/remove unit recipes with reference-id, player/index, unique-caption and caption-marker selectors, player-wide removals, guarded area operations, direct placed-unit reference disconnects, and bulk area field edits",
			Create:       "full: create_object-style unit recipes, guarded area copy, and smoke fixtures",
			Delete:       "partial: plan-gated direct unit deletes, exact-caption, caption-prefix, caption-contains, unit-type, units-player, and area delete plans, cleanup recipes/commands for direct placed-unit references, plus selector-based physical unit removal with direct trigger object-reference guards",
			Verification: "structure_verified_not_engine_verified",
			Evidence:     "kit scen units/refs/delete-plan/delete/describe/patch/smoke and diagnostic scenarios v4-v15",
			Next:         "field-level edits for less-common object properties and broader dependency planning across multi-op recipes",
		},
		{
			Domain:       "scenario",
			Section:      "map, terrain, elevation, regions",
			Read:         "full: dimensions, terrain/elevation grids, region guesses",
			Write:        "partial: rectangle/circle/line/border terrain and elevation recipe edits plus terrain-area copy",
			Create:       "n/a: map sections exist as scenario state, not independent records",
			Delete:       "n/a: terrain deletion is replacement with another terrain/elevation",
			Verification: "structure_verified_not_engine_verified",
			Evidence:     "kit scen map/regions/patch/smoke",
			Next:         "bulk paint primitives for lane networks, mirroring, and terrain-object alignment",
		},
		{
			Domain:       "scenario",
			Section:      "players, diplomacy, resources, victory",
			Read:         "full: settings inspector surfaces player slots, AI names/types, diplomacy matrix, allied victory flags, resources, and global victory fields",
			Write:        "partial: recipe support exists for player slots, pairwise diplomacy, diplomacy options/allied victory, resources, global victory, blank-scenario player-count setup, Gaia-active blank baseline, and Units-section player-count synchronization",
			Create:       "n/a: players are fixed scenario slots",
			Delete:       "n/a: player slots are disabled/configured, not deleted",
			Verification: "structure_verified_not_engine_verified",
			Evidence:     "kit scen settings/describe/patch and v8-v15 diagnostics",
			Next:         "add less-common player/lobby fields only when an authoring case needs them; do not advertise Gaia-inactive authoring until a separate active-state field is decoded and write-verified",
		},
		{
			Domain:       "scenario",
			Section:      "strings, messages, XS",
			Read:         "full: scenario string table, display text, chat/display effects, XS attachment/carrier census, source extraction/comparison, and deploy checks",
			Write:        "partial: fixed-slot string add/set/clear/tombstone recipes, message recipes, XS executable attachment, parser-style carrier escrow, deploy-tree materialization under resources/_common/xs, and script_call shim emission",
			Create:       "full: new fixed-slot strings and display/chat/string references through supported trigger recipes",
			Delete:       "partial: plan-gated direct string clears/tombstones by id, unique exact text, text prefix, or text contains marker; unreferenced strings can be cleared; referenced strings can be semantically tombstoned or direct-reference-disconnected before clear while preserving fixed IDs; duplicate exact text is rejected and no string-id compaction is attempted",
			Verification: "structure_verified_not_engine_verified",
			Evidence:     "kit scen strings/xs/deploycheck/xs attach/xs deploy/xs extract/xs compare, kit xs bridge/shims/datagen",
			Next:         "richer XS-sidecar authoring helpers and broader string reference coverage",
		},
		{
			Domain:       "dat",
			Section:      "effects and commands",
			Read:         "full: typed effect records and command rows with semantic overlays for known command types, including upgrade source/destination refs, rename-unit string IDs, resource/operation/unit-class/tech-attribute operands, and command-matrix filters for typed/unknown/reference-focused inspection",
			Write:        "partial: direct effect name/command patches, patch/append/remove/replace command rows, disable_effect semantic command clears, plus named helpers and AI-facing templates for proven command families",
			Create:       "full: create effects, clone/tweak source effects, resource/attribute/cost/time template effects, and ability-paired effects",
			Delete:       "partial: plan-gated direct effect deletes, direct effect-disable semantic clears, and effect-command row deletes; referenced effects can be semantically cleared to an inert stable ID; physical effect delete only when unreferenced and rewrite-safe",
			Verification: "structure_verified_not_engine_verified",
			Evidence:     "kit dat effects/effect/effect-create/effect-patch/effect-disable/effect-delete/effect-command-delete/command-matrix/refs/delete-plan/delete/codec-plan/codec-patch; filtered command/reference matrix inspection; plan-gated effect-command row delete readback; remove_commands and create_effect clone readbacks",
			Next:         "promote additional command helpers only when field meanings are evidence-backed",
		},
		{
			Domain:       "dat",
			Section:      "techs, researches, abilities",
			Read:         "full: costs, requirements, locations, DLL ids, effect links, paired ability view",
			Write:        "partial: direct tech scalar patches, direct ability-patch wrappers, direct semantic disconnects with cleanup commands/recipes in blocked delete plans, guarded non-tail tail-swap delete, ability creation/clone-tweak, paired ability deletion through the current tech delete strategy, and disable_ability semantic effect clears when physical delete is unsafe",
			Create:       "full: direct tech clone-to-tail create plus paired Tech+Effect abilities, including clone-from-source-effect recipes",
			Delete:       "partial: plan-gated direct tech/ability deletes plus direct ability-disable and ability-delete wrappers; physical tail tech delete plus guarded non-tail tail-swap delete; referenced tech plans expose cleanup command/recipe fields before physical retry; paired ability delete supports both when the paired effect is private; referenced-but-unshared abilities can be semantically disabled by clearing their effect commands",
			Verification: "structure_verified_not_engine_verified",
			Evidence:     "kit dat techs/tech/tech-create/tech-patch/tech-delete/abilities/ability/ability-create/ability-patch/ability-disable/ability-delete/delete-plan/delete/disconnect/codec-patch; create_tech/create_ability clone readback and direct wrapper smoke",
			Next:         "keep broad renumber-all deletion blocked unless a real authoring case needs it",
		},
		{
			Domain:       "dat",
			Section:      "civs and unit availability",
			Read:         "full: civ headers/resources/unit slots and availability cards",
			Write:        "partial: direct civ field/resource patches, direct unit enabled flags across civ selections with disabled-card recipe hints, and direct stable-ID unit disconnects across verified progression/effect/header-task/unit-record refs",
			Create:       "unsupported: stable civ-table creation is not implemented",
			Delete:       "unsupported: civ deletion is ledger-only",
			Verification: "structure_verified_not_engine_verified",
			Evidence:     "kit dat civs/civ-patch/availability/availability-set/unit_availability/disconnect",
			Next:         "promote more typed effect-command operand semantics so fewer unit references stay warning-grade; candidate operands are counted separately from arbitrary possible operands",
		},
		{
			Domain:       "dat",
			Section:      "units and unit headers",
			Read:         "full: broad current-DE unit records, costs, tasks, attacks, armor, creatable fields",
			Write:        "partial: broad direct scalar/list patches, task rows, drop-sites, create-unit recipes, AI-facing train-button and availability templates, unit/header child-row deletion by list rebuild, and optional disconnect cleanup for rewrite-supported unit refs in delete plans",
			Create:       "full: direct clone/create unit records through codec recipes",
			Delete:       "partial: plan-gated direct semantic unit deletes, direct unit-delete wrapper, unit child-row deletes, and unit-header task row deletes are supported; delete plans suggest paired disconnect_units reference cleanup for unit IDs; physical unit deletion is not the default",
			Verification: "structure_verified_not_engine_verified",
			Evidence:     "kit dat units/unit/unit-create/unit-delete/unit-child-delete/unit-headers/unit-header-task-delete/delete-plan/delete/patch-unit/codec-patch; plan-gated unit attack/armour/train-location/drop-site/task/damage-graphic and unit-header task row delete readbacks",
			Next:         "finish edge-field naming and promote more recipes only from repeated designer friction",
		},
		{
			Domain:       "dat",
			Section:      "graphics, sounds, particles",
			Read:         "full: graphics, deltas, angle sounds, sound items, particle-relevant fields",
			Write:        "partial: direct graphic/sound scalar fields, deltas, angle sounds, particle binds, sound mutes, and graphic/sound child-row removal",
			Create:       "full: direct create graphics and sounds; FX descriptors bind assets to DAT rows",
			Delete:       "partial: plan-gated direct semantic sound mute, unreferenced stable-ID graphic delete, graphic-delta, guarded graphic-angle-sound, and sound-item row deletes are supported; referenced graphic deletes stay ledger-blocked",
			Verification: "structure_verified_not_engine_verified",
			Evidence:     "kit dat graphics/graphic/graphic-create/graphic-patch/graphic-delete/graphic-delta-delete/graphic-angle-sound-delete/sounds/sound/sound-create/sound-patch/sound-delete/sound-item-delete/delete-plan/delete, kit fx new/bind/lint; plan-gated graphic stable-slot delete plus graphic-delta/graphic-angle-sound/sound-item row delete readbacks",
			Next:         "complete asset-to-engine authoring loops with stronger readback and editor smoke checks",
		},
		{
			Domain:       "dat",
			Section:      "terrain, restrictions, player colours, random maps",
			Read:         "partial: terrains/restrictions/player-colours typed; random maps framed",
			Write:        "partial: direct terrain names/overlays, direct restriction passability, player-colour fields",
			Create:       "partial: player-colour append clone is structure-verified; terrain/restriction/random-map row creation remains unsupported",
			Delete:       "partial: unreferenced tail player-colour rows can be physically removed; terrain/restriction/random-map and non-tail palette deletes stay blocked",
			Verification: "structure_verified_not_engine_verified",
			Evidence:     "kit dat terrains/terrain/terrain-patch/terrain-restrictions/terrain-restriction-patch/player-colours/player-colour-create/player-colour-patch/player-colour-delete/random-maps; delete-plan/delete player_colour supports unreferenced tail rows",
			Next:         "defer RandomMap writes until a trusted non-empty current-DE fixture exists",
		},
		{
			Domain:       "dat",
			Section:      "tech tree and references",
			Read:         "full: connection rows, verified references, neutral refs explorer",
			Write:        "partial: direct connection field/list patches and semantic rewrites for known references",
			Create:       "partial: direct connection-family clone append is structure-verified",
			Delete:       "partial: direct connection-row removal, plan-gated row removal, and direct tech/unit reference cleanup exist; physical core-record deletes remain conservative",
			Verification: "structure_verified_not_engine_verified",
			Evidence:     "kit dat tech-tree/tech-tree-connection-create/tech-tree-connection-patch/tech-tree-connection-delete/refs/delete-plan/delete/disconnect/reference_rewrites/codec-patch",
			Next:         "complete reference rewrite coverage before enabling broad physical deletion",
		},
		{
			Domain:       "replay",
			Section:      "header, lobby, identity, data set",
			Read:         "full: current DE metadata, players, settings, scenario identity, data-set name/checksum, embedded trigger graph fingerprint/search/neighborhood",
			Write:        "n/a: replays are immutable evidence, not authoring targets",
			Create:       "n/a: created by the game engine",
			Delete:       "n/a: not a Kit mutation surface",
			Verification: "structure_verified_not_engine_verified",
			Evidence:     "kit replay info/summary/identify/datamod-check/fetch/graph/triggers/trigger-neighborhood",
			Next:         "keep variants such as SP/MP and random-position CBA covered by fixtures; promote trigger graph locator fallbacks only when exact boundaries are proven",
		},
		{
			Domain:       "replay",
			Section:      "actions, chat, events, story",
			Read:         "partial: typed actions, chat, taunts, flares, telemetry, object references, player event tables",
			Write:        "n/a: replays are immutable evidence, not authoring targets",
			Create:       "n/a: created by the game engine",
			Delete:       "n/a: not a Kit mutation surface",
			Verification: "structure_verified_not_engine_verified",
			Evidence:     "kit replay actions/events/chat/feedback/story/player-profile/inbox",
			Next:         "decode remaining action op payloads and enrich object/context linkage",
		},
		{
			Domain:       "replay",
			Section:      "sync, combat, lifecycle, sidecars",
			Read:         "partial: sync matrix words, sidecar assertions, attr20/154 aggregate channels, lifecycle heuristics",
			Write:        "n/a: replays are immutable evidence, not authoring targets",
			Create:       "n/a: created by the game engine",
			Delete:       "n/a: not a Kit mutation surface",
			Verification: "mixed: replay decode is structural; sidecar rows are engine-produced when a fixture ran live",
			Evidence:     "kit replay sync/checksum-probe/combat/player-series/sidecar-sync/xs-telemetry",
			Next:         "separate direct replay facts from sidecar-proven facts and finish dark-byte decode",
		},
	}
}

type syncWordFact struct {
	Word         string
	ID           string
	Label        string
	RequiredTier string
	Hypothesis   bool
}

func syncMatrixFrontierText() (known, unknown, next, verification string) {
	facts := []syncWordFact{
		{Word: "word_1", ID: "replay.sync_word_1_resource_sum", Label: "word_1 is stockpiled food+wood+stone+gold", RequiredTier: enginefacts.EngineVerified},
		{Word: "word_2", ID: "replay.sync_word_2_unit_type_sum", Label: "word_2 is living object unit-type-id sum", RequiredTier: enginefacts.EngineVerified},
		{Word: "word_3", ID: "replay.sync_word_3_object_state_weight", Label: "word_3 is a strong object-state-sum hypothesis", RequiredTier: "strong_hypothesis", Hypothesis: true},
		{Word: "word_4", ID: "replay.sync_word_4_object_carry_sum", Label: "word_4 is current object carry sum", RequiredTier: enginefacts.EngineVerified},
		{Word: "word_6", ID: "replay.sync_word_6_object_count", Label: "word_6 is living object count", RequiredTier: enginefacts.EngineVerified},
		{Word: "word_7", ID: "replay.sync_word_7_position_linked", Label: "word_7 is checksum-counted object position sum as (x+y)*100", RequiredTier: enginefacts.EngineVerified},
		{Word: "word_8", ID: "replay.sync_word_8_player_number", Label: "word_8 is player number", RequiredTier: enginefacts.EngineVerified},
		{Word: "word_10", ID: "replay.sync_word_10_object_id_sum", Label: "word_10 is living object instance-id sum", RequiredTier: enginefacts.EngineVerified},
	}

	ledger, err := enginefacts.LoadDefault()
	if err != nil {
		return "rows are players; word_1/2/4/6/7/8/10 have promoted aggregate meanings in the engine-facts ledger, but the ledger could not be loaded for citation",
			"word_0, word_5, word_9, rare object-state values, and mode-specific score/fog/exploration contributions remain unresolved",
			"restore the engine facts ledger, then use desync sync logs plus packed fixtures to resolve word_0/5/9",
			"mixed: fallback roadmap text; engine-facts ledger unavailable"
	}

	var knownParts []string
	var verifiedWords []string
	var hypothesisWords []string
	var missing []string
	for _, item := range facts {
		fact, ok := ledger.ByID(item.ID)
		if !ok || fact.Tier != item.RequiredTier {
			missing = append(missing, item.Word)
			continue
		}
		knownParts = append(knownParts, item.Label)
		if item.Hypothesis {
			hypothesisWords = append(hypothesisWords, item.Word)
		} else {
			verifiedWords = append(verifiedWords, item.Word)
		}
	}

	if len(knownParts) == 0 {
		knownParts = append(knownParts, "rows are players")
	} else {
		knownParts = append([]string{"rows are players"}, knownParts...)
	}
	unresolved := []string{"word_0", "word_5", "word_9", "rare object-state values", "mode-specific score/fog/exploration contributions"}
	unresolved = append(unresolved, missing...)

	known = strings.Join(knownParts, "; ")
	unknown = strings.Join(dedupeStringList(unresolved), ", ") + " remain unresolved"
	next = "use desync sync logs plus packed fixtures to resolve word_0/5/9 and promote word_3 only when state enumeration matches independent ground truth"
	verification = "mixed: " + strings.Join(verifiedWords, "/") + " engine_verified"
	if len(hypothesisWords) > 0 {
		verification += "; " + strings.Join(hypothesisWords, "/") + " strong_hypothesis"
	}
	verification += "; word_0/5/9 unresolved"
	return known, unknown, next, verification
}

func dedupeStringList(values []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

func allDarkFrontiers() []DarkFrontier {
	syncKnown, syncUnknown, syncNext, syncVerification := syncMatrixFrontierText()
	return []DarkFrontier{
		{
			Domain:       "scenario",
			Region:       "placement snap and footprint intent",
			Priority:     1,
			Known:        "the DE engine snaps spawned blocking buildings/units according to their current footprint; fractional placement is appropriate for footprint-less Gaia artwork, and the writer now defaults to grid placement while preserving explicit fractional/auto policies",
			Unknown:      "the complete writer-side parity table for every footprint/collision combination and the exact set of placement surfaces that should warn versus snap",
			Tools:        "kit scen units, dat unit footprint, recipe writer, load-safety lint, engine placement fixture",
			Next:         "expand the DAT-backed auto policy beyond the conservative fallback and add engine confirmation for additional building classes",
			Verification: "engine_verified_behavior; writer policy structure_verified",
		},
		{
			Domain:       "dat",
			Region:       "full unit decode and SQL export",
			Priority:     1,
			Known:        "the current DAT codec and SQL cache expose death_spawn 57, spawning/upgrade graphics 183/184, charge-independent building links 202-217, garrison/salvage fields, collision fields, and the stable unit/building scalar slice; SQL marks those fields exported",
			Unknown:      "secondary_projectile 65, charge 59-63/125/162, and the remaining DLC tail semantics; those bytes remain bounded and must not be promoted from layout correlation alone",
			Tools:        "kit dat unit, kit dat sql, sqlite cache, DAT byte diffs, controlled DAT fixtures and engine readback",
			Next:         "map secondary-projectile and charge fields against a current-build byte fixture, then expose the DLC tail as named fields only when offsets survive writer round-trip tests",
			Verification: "partial_decode; structure_verified_not_engine_verified",
		},
		{
			Domain:       "scenario",
			Region:       "1.59 writer parity",
			Priority:     1,
			Known:        "Kit reads and writes editor-saved 1.59 scenarios through the current section-aware layout, with round-trip and engine-load fixtures already covering the main path",
			Unknown:      "whether every newly edited 1.59 section survives mutation without byte drift or a misplaced tail; trigger graphs and editor-added units still need broader differential coverage",
			Tools:        "kit scen info/describe/diff, 1.59 roundtrip fixtures, section-aware bytediff, load-safety, engine load",
			Next:         "expand the 1.59 differential corpus and make layout selection evidence-based rather than relying on the first parse that happens to succeed",
			Verification: "structure_verified_not_engine_verified; current editor fixture engine-tested",
		},
		{
			Domain:       "replay",
			Region:       "sidecar schema pipeline",
			Priority:     1,
			Known:        "generated sidecars use one shared .types writer and sibling-schema discovery path; schema decoders report bytes consumed, tag counts, and reject ambiguous equally plausible schemas",
			Unknown:      "numeric fact-ID conventions across older hand-authored sidecars and whether every foreign generator can be migrated without compatibility aliases",
			Tools:        "kit xsdat, sibling .types discovery, schema registry, replay sidecar-sync, generated probe writers",
			Next:         "add compatibility fixtures for the remaining legacy sidecar families and make generator manifests declare their schema explicitly",
			Verification: "structure_verified_not_engine_verified; shared pipeline unit-tested",
		},
		{
			Domain:       "scenario",
			Region:       "XS safety and differential lint",
			Priority:     1,
			Known:        "Kit and xs-check catch several observed compile hazards, including builtin shadowing, type-invalid conditions, unparenthesized arithmetic conditions, and stale checker drift; harness and custom-store generation run both independent oracles when available and fail closed by default",
			Unknown:      "complete engine-derived rule coverage for runtime crashes and silent non-execution, including parenthesized returns, logical side effects, trigger operations, ASCII-only strings, and MP-safe time usage",
			Tools:        "kit scen lint --xs, xs-check, generated XS IR/emitter, controversy ledger, compile-only passenger fixtures",
			Next:         "finish the typed generated-XS IR boundary and add evidence-tiered runtime/crash signatures without treating catalogue-only rules as compile guarantees",
			Verification: "mixed: several rules engine_verified; rule pack remains incomplete",
		},
		{
			Domain:       "scenario",
			Region:       "load-safety checkpoints",
			Priority:     1,
			Known:        "C0 preview/load selection and C4 keep-alive checks exist in focused forms; 1.59 editor saves and slot/civilization constraints have exposed real load failures",
			Unknown:      "general preview emulation across layouts, C1 lock-civ and slot collisions, and a single checkpoint report that explains the first failing boundary",
			Tools:        "kit scen lint --load-safety, scenario load checkpoints, section-aware parser, engine launch fixtures",
			Next:         "generalize the named C0/C1/C4 families, fail closed on ambiguous layouts, and report the earliest unsafe checkpoint with the evidence used",
			Verification: "structure_verified_not_engine_verified; C0/C4 behavior has live fixture evidence",
		},
		{
			Domain:       "scenario",
			Region:       "RPG shop fixture health",
			Priority:     1,
			Known:        "the shop demo exercises renamed train-token buttons, per-host stores, costs, starting age, resources, and purchase telemetry; recent diplomacy/settings changes made the fixture red",
			Unknown:      "which fixture assumptions conflict with the new diplomacy rules and whether player-indexed settings are still aligned after 1.59 edits",
			Tools:        "kit rpg custom-store, scenario lint, scenario settings/diff, purchase sidecar, current DE shop fixture",
			Next:         "repair the fixture and add a regression test for the accepted player/settings shape; do not skip the failing test or weaken the diplomacy rules",
			Verification: "structure_verified_not_engine_verified; purchase path has engine evidence",
		},
		{
			Domain:       "replay",
			Region:       "replay lobby and feedback transparency",
			Priority:     2,
			Known:        "replay info exposes current DE metadata and player/settings context; chat, flares, and feedback are available through separate readers",
			Unknown:      "game speed/population/map in one stable info surface, plus trigger descriptions and scenario messages alongside player comments in feedback",
			Tools:        "kit replay info, kit replay feedback, trigger graph, scenario message reader, current replay fixtures",
			Next:         "add the missing lobby fields and merge trigger/message context into feedback, using a replay with authored descriptions for verification",
			Verification: "partial_decode; fixture-dependent",
		},
		{
			Domain:       "dat",
			Region:       "effect command semantics",
			Priority:     2,
			Known:        "typed command overlays and the DAT SQL/cache path cover many resource, unit, tech, and attribute commands; local-building 200-series and other scope behavior still has explicit evidence boundaries",
			Unknown:      "runtime semantics for remaining class/scope commands and the newly relevant annex/spawn/death fields",
			Tools:        "kit dat sql/effects/effect/command-matrix, semantics fixtures, sidecar readback, engine-facts ledger",
			Next:         "finish the typed SQL export first, then build packed semantics fixtures only for fields whose structure is stable",
			Verification: "structure_verified_not_engine_verified until readback",
		},
		{
			Domain:       "replay",
			Region:       "VER 9.4 trigger-graph locator hardening",
			Priority:     2,
			Known:        "current VER 9.4/save_version 68 trigger graphs parse on ordinary multiplayer layouts and the known consolidated few-trigger/large-effect-list fixture; the consolidated fixture yields trigger_count=3, effect_count=1873, named effects, and a stable graph_sha256",
			Unknown:      "whether future large consolidated trigger layouts can still defeat the primary string/dense-region locator and rely on fallback candidate ranking",
			Tools:        "kit replay triggers, kit identify, trigger graph extractor, optional private consolidated fixture plus a parsing VER 9.4 control replay",
			Next:         "harden primary trigger start location for large consolidated layouts while preserving the current fallback scan and known VER 9.4 controls",
			Verification: "private_fixture_regression_optional",
		},
		{
			Domain:       "replay",
			Region:       "sync matrix words",
			Priority:     1,
			Known:        syncKnown,
			Unknown:      syncUnknown,
			Tools:        "kit replay sync, checksum-probe, diff-state, player-series, v7/v12/v15 diagnostics",
			Next:         syncNext,
			Verification: syncVerification,
		},
		{
			Domain:       "replay",
			Region:       "combat and death attribution",
			Priority:     1,
			Known:        "aggregate rolling K/D exists through engine attributes 20 and 154; CBA side channels expose 10px-resolution progression and raze pressure",
			Unknown:      "direct per-unit killer-victim event is not proven in the recording stream",
			Tools:        "kit replay combat, player-series, sidecar-sync, kit cba replay perf/progression/razes/raze-pressure",
			Next:         "use v15-style castle kill ledgers and CBA trigger side channels to bound what is replay-native versus sidecar-derived",
			Verification: "mixed: attr channels are replay-observed; exact kill attribution needs sidecar/trigger corroboration",
		},
		{
			Domain:       "replay",
			Region:       "initial object tails",
			Priority:     1,
			Known:        "object index, owner, class, unit id/name, position, rotation, target references, and many spawn fields are decoded",
			Unknown:      "remaining per-object suffix bytes and some lifecycle/runtime fields are named by shape, not meaning",
			Tools:        "kit replay objects, object-state, object-shapes, spawns, opaque-spans",
			Next:         "diff controlled fixtures where only stance, HP, garrison, task, or target state changes",
			Verification: "partial_decode",
		},
		{
			Domain:       "replay",
			Region:       "per-player effective game-data tails",
			Priority:     2,
			Known:        "large per-player header tails are effective DAT tables with shared template bytes and civ/mod deltas",
			Unknown:      "full subregion naming and every DAT cross-reference inside the replay snapshot",
			Tools:        "kit replay effective-data, effective-units, datamod-check with baseline/reference DAT",
			Next:         "finish template-vs-civ-vs-data-mod subregion reports and promote DAT references into human names",
			Verification: "partial_decode",
		},
		{
			Domain:       "replay",
			Region:       "remaining body/header opaque spans",
			Priority:     2,
			Known:        "coverage ledgers identify every byte as decoded or bounded opaque; no invisible holes are acceptable",
			Unknown:      "semantics of the remaining bounded spans vary by replay mode and game state",
			Tools:        "kit replay coverage, opaque-spans, opaque-clusters, frontier, corpus",
			Next:         "keep clustering real-corpus spans and design fixtures only when clusters stay stable but unexplained",
			Verification: "coverage_verified_not_semantics_verified",
		},
		{
			Domain:       "dat",
			Region:       "effect command semantic matrix",
			Priority:     1,
			Known:        "disable_tech, enable_unit, spawn_unit, modify_tech, resource modifiers, tech cost/time modifiers, rename-unit string IDs, upgrade_unit source/destination refs, unit-class names, and unit-attribute modifiers have typed promotion; command-shape candidates are split from arbitrary numeric possible operands; the current reference DAT has zero command rows under command-matrix --unknown",
			Unknown:      "class-targeted and scope-targeted runtime behavior still needs scenario/replay semantics-readback; future data versions may still introduce raw unsupported command types",
			Tools:        "kit dat effects/effect/command-matrix with typed/unknown/reference filters; kit dat refs with class/confidence/source filters; codec-plan/semantics-pack/semantics-readback",
			Next:         "build packed DAT semantics fixtures and add helpers only after readback proves field behavior",
			Verification: "structure_verified_not_engine_verified until fixture replay says otherwise",
		},
		{
			Domain:       "dat",
			Region:       "physical delete frontier",
			Priority:     1,
			Known:        "stable-ID preserving semantic delete is safe for units/sounds; unreferenced graphics can become absent stable-ID slots; unreferenced effects and tail techs can be physically removed; unreferenced non-tail techs can be removed by guarded tail-swap when tail references are rewrite-covered",
			Unknown:      "broad non-tail renumber-all deletion across all verified and possible references is not complete",
			Tools:        "kit dat delete, disconnect, delete-plan, refs, reference_rewrites, codec-patch",
			Next:         "keep expanding reference coverage; only build renumber-all deletion if tail-swap is insufficient for a real workflow",
			Verification: "structure_verified_not_engine_verified",
		},
		{
			Domain:       "scenario",
			Region:       "legacy 1.53 scenario read support",
			Priority:     2,
			Known:        "modern scenario readers cover 1.55-1.58; older 1.53 community/classic scenarios are rejected before inspection",
			Unknown:      "exact per-section field deltas needed for safe read-only header, players, terrain, units, strings, triggers, effects, and conditions",
			Tools:        "kit scen info/describe/units/triggers/terrain/strings, scenario section parser, legacy fixture diffs",
			Next:         "add read-only 1.53 support first, gated by section roundtrip/coverage tests; do not add mutation until the read model is stable",
			Verification: "backlog_logged_not_implemented",
		},
		{
			Domain:       "scenario",
			Region:       "engine-render truth",
			Priority:     2,
			Known:        "scenario writer can roundtrip and produce engine-loadable fixtures when recipes stay inside proven surfaces",
			Unknown:      "render/UI behavior is not guaranteed by structure checks; DE editor/game remains the oracle",
			Tools:        "kit scen patch/smoke/deploycheck/release-check/verify-run",
			Next:         "make generated fixtures carry their own assertions and sidecars so live runs close the loop faster",
			Verification: "structure_verified_not_engine_verified",
		},
		{
			Domain:       "scenario",
			Region:       "general R/W/D coverage",
			Priority:     2,
			Known:        "triggers, units, terrain, strings, XS have practical authoring paths",
			Unknown:      "full arbitrary editor-equivalent writing for every scenario section remains unfinished",
			Tools:        "kit scen describe/analyze/patch/diff/lint/xs",
			Next:         "prioritize authoring helpers by scenario-designer intent, not by raw binary section order",
			Verification: "structure_verified_not_engine_verified",
		},
	}
}
