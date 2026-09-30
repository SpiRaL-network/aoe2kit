package diagnostics

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sort"
	"strings"

	"aoe2kit/pkg/datfile"
	"aoe2kit/pkg/replay"
	"aoe2kit/pkg/scenario"
	xsauthor "aoe2kit/pkg/xs"
)

const SemanticsReadbackVersion = "dat-command-semantics-readback-v1"

type SemanticsReadbackOptions struct {
	ExpectedPath string
	DatPath      string
	ScenarioPath string
	ReplayPath   string
	XSDataPath   string
}

type SemanticsReadbackReport struct {
	Version      string                   `json:"version"`
	Verification string                   `json:"verification"`
	Status       string                   `json:"status"`
	Inputs       SemanticsReadbackInput   `json:"inputs"`
	Summary      SemanticsReadbackSummary `json:"summary"`
	Replay       ReplayReadbackSummary    `json:"replay"`
	Sidecar      SidecarReadbackSummary   `json:"sidecar,omitempty"`
	ParserGaps   []string                 `json:"parser_gaps,omitempty"`
	Lanes        []LaneReadback           `json:"lanes"`
}

type SemanticsReadbackInput struct {
	ExpectedPath string `json:"expected_path"`
	DatPath      string `json:"dat_path"`
	ScenarioPath string `json:"scenario_path"`
	ReplayPath   string `json:"replay_path"`
	XSDataPath   string `json:"xsdat_path,omitempty"`
}

type SemanticsReadbackSummary struct {
	Promoted       int `json:"promoted"`
	Failed         int `json:"failed"`
	Inconclusive   int `json:"inconclusive"`
	NotImplemented int `json:"not_implemented"`
}

type ReplayReadbackSummary struct {
	Duration               string `json:"duration,omitempty"`
	DurationMS             int    `json:"duration_ms,omitempty"`
	TriggerGraphOK         bool   `json:"trigger_graph_ok"`
	TriggerGraphError      string `json:"trigger_graph_error,omitempty"`
	ScenarioIdentityTier   string `json:"scenario_identity_tier,omitempty"`
	ScenarioIdentitySHA    string `json:"scenario_identity_sha256,omitempty"`
	DataSetStatus          string `json:"data_set_status,omitempty"`
	DataSetName            string `json:"data_set_name,omitempty"`
	DataSetSource          string `json:"data_set_source,omitempty"`
	DataSetConfidence      string `json:"data_set_confidence,omitempty"`
	DataSetError           string `json:"data_set_error,omitempty"`
	FinalResourceStockpile uint32 `json:"final_resource_stockpile,omitempty"`
	FinalObjectCount       int    `json:"final_object_count,omitempty"`
}

type SidecarReadbackSummary struct {
	Path        string `json:"path,omitempty"`
	OK          bool   `json:"ok"`
	Schema      string `json:"schema,omitempty"`
	Magic       string `json:"magic,omitempty"`
	Version     int    `json:"version,omitempty"`
	Fixture     string `json:"fixture,omitempty"`
	Rows        int    `json:"rows,omitempty"`
	HasFooter   bool   `json:"has_footer,omitempty"`
	RowsWritten int    `json:"rows_written,omitempty"`
	Error       string `json:"error,omitempty"`
}

type LaneReadback struct {
	ID           string             `json:"id"`
	Label        string             `json:"label"`
	ExpectedTier string             `json:"expected_tier"`
	Status       string             `json:"status"`
	PromotedTier string             `json:"promoted_tier,omitempty"`
	Conclusion   string             `json:"conclusion"`
	Evidence     []ReadbackEvidence `json:"evidence"`
}

type ReadbackEvidence struct {
	Source      string `json:"source"`
	Status      string `json:"status"`
	Observation string `json:"observation"`
}

func ReadDATCommandSemantics(opts SemanticsReadbackOptions) (*SemanticsReadbackReport, error) {
	expected, err := readExpectedLedger(opts.ExpectedPath)
	if err != nil {
		return nil, err
	}
	idx, err := datfile.Open(opts.DatPath)
	if err != nil {
		return nil, fmt.Errorf("open dat: %w", err)
	}
	scen, err := scenario.Open(opts.ScenarioPath)
	if err != nil {
		return nil, fmt.Errorf("open scenario: %w", err)
	}
	rec, err := replay.Open(opts.ReplayPath)
	if err != nil {
		return nil, fmt.Errorf("open replay: %w", err)
	}
	series, err := replay.BuildPlayerSeries(opts.ReplayPath, replay.PlayerSeriesOptions{PlayerID: 1, ChangesOnly: true})
	if err != nil {
		return nil, fmt.Errorf("build replay player series: %w", err)
	}
	var sidecar *xsauthor.DataSchemaDecodeReport
	var sidecarSummary SidecarReadbackSummary
	if opts.XSDataPath != "" {
		decoded, err := xsauthor.DecodeDataFileSchema(opts.XSDataPath, xsauthor.DataSchemaDecodeOptions{Schema: "a2ksem2"})
		if err != nil {
			sidecarSummary = SidecarReadbackSummary{Path: opts.XSDataPath, OK: false, Error: err.Error()}
		} else {
			sidecar = &decoded
			sidecarSummary = sidecarReadbackSummary(decoded)
		}
	}
	ctx := readbackContext{
		expected: expected,
		dat:      idx,
		scen:     scen,
		rec:      rec,
		series:   series,
		sidecar:  sidecar,
	}
	report := &SemanticsReadbackReport{
		Version:      SemanticsReadbackVersion,
		Verification: "structure_verified_plus_replay_observed_with_explicit_inconclusive_oracles",
		Inputs: SemanticsReadbackInput{
			ExpectedPath: opts.ExpectedPath,
			DatPath:      opts.DatPath,
			ScenarioPath: opts.ScenarioPath,
			ReplayPath:   opts.ReplayPath,
			XSDataPath:   opts.XSDataPath,
		},
		Replay:  replayReadbackSummary(rec, series),
		Sidecar: sidecarSummary,
	}
	if !rec.TriggerGraphOK {
		report.ParserGaps = append(report.ParserGaps, "trigger graph parser did not locate the embedded VER scenario; fallback scenario identity is weaker than trigger-graph identity")
	}
	if rec.DataSet.Status == "unknown" {
		report.ParserGaps = append(report.ParserGaps, "data-set identity parser returned unknown for this single-player local data-mod replay")
	}
	if opts.XSDataPath == "" {
		report.ParserGaps = append(report.ParserGaps, "no XS sidecar supplied; DAT command semantics that require exact phase rows remain weaker or inconclusive")
	} else if sidecar == nil || !sidecar.OK {
		report.ParserGaps = append(report.ParserGaps, "XS sidecar decode failed; DAT command semantics that require exact phase rows remain weaker or inconclusive")
	}
	for _, lane := range expected.Lanes {
		readback := ctx.readLane(lane)
		report.Lanes = append(report.Lanes, readback)
		switch readback.Status {
		case "promoted":
			report.Summary.Promoted++
		case "failed":
			report.Summary.Failed++
		default:
			report.Summary.Inconclusive++
		}
	}
	for _, lane := range expected.PendingLanes {
		report.Lanes = append(report.Lanes, LaneReadback{
			ID:           lane.ID,
			Label:        lane.Label,
			ExpectedTier: lane.Tier,
			Status:       "not_implemented",
			Conclusion:   "Not present in this packed test.",
			Evidence: []ReadbackEvidence{
				{Source: "expected ledger", Status: "not_implemented", Observation: strings.Join(lane.Expected, "; ")},
			},
		})
		report.Summary.NotImplemented++
	}
	if report.Summary.Failed > 0 {
		report.Status = "failed"
	} else if report.Summary.Inconclusive > 0 {
		report.Status = "partial"
	} else {
		report.Status = "ok"
	}
	return report, nil
}

func sidecarReadbackSummary(report xsauthor.DataSchemaDecodeReport) SidecarReadbackSummary {
	out := SidecarReadbackSummary{
		Path:      report.Path,
		OK:        report.OK,
		Schema:    report.Schema,
		Rows:      report.Summary.Rows,
		HasFooter: report.Summary.HasFooter,
	}
	if len(report.Errors) > 0 {
		out.Error = strings.Join(report.Errors, "; ")
	}
	out.Magic = stringAny(report.Header["magic"])
	out.Version = intAny(report.Header["version"])
	out.Fixture = stringAny(report.Header["fixture"])
	out.RowsWritten = intAny(report.Footer["rows_written"])
	return out
}

func WriteDATCommandSemanticsReadbackMarkdown(report *SemanticsReadbackReport, path string) error {
	var b strings.Builder
	fmt.Fprintf(&b, "# DAT Command Semantics Readback\n\n")
	fmt.Fprintf(&b, "Version: `%s`\n\n", report.Version)
	fmt.Fprintf(&b, "Verification: `%s`\n\n", report.Verification)
	fmt.Fprintf(&b, "Status: `%s`\n\n", report.Status)
	fmt.Fprintf(&b, "## Inputs\n\n")
	fmt.Fprintf(&b, "- expected: `%s`\n", report.Inputs.ExpectedPath)
	fmt.Fprintf(&b, "- dat: `%s`\n", report.Inputs.DatPath)
	fmt.Fprintf(&b, "- scenario: `%s`\n", report.Inputs.ScenarioPath)
	fmt.Fprintf(&b, "- replay: `%s`\n", report.Inputs.ReplayPath)
	if report.Inputs.XSDataPath != "" {
		fmt.Fprintf(&b, "- xsdat: `%s`\n", report.Inputs.XSDataPath)
	}
	fmt.Fprintf(&b, "\n")
	fmt.Fprintf(&b, "## Summary\n\n")
	fmt.Fprintf(&b, "- promoted: %d\n", report.Summary.Promoted)
	fmt.Fprintf(&b, "- failed: %d\n", report.Summary.Failed)
	fmt.Fprintf(&b, "- inconclusive: %d\n", report.Summary.Inconclusive)
	fmt.Fprintf(&b, "- not implemented in this pack: %d\n\n", report.Summary.NotImplemented)
	fmt.Fprintf(&b, "## Replay Readback\n\n")
	fmt.Fprintf(&b, "- duration: `%s`\n", report.Replay.Duration)
	fmt.Fprintf(&b, "- trigger_graph_ok: `%t`\n", report.Replay.TriggerGraphOK)
	if report.Replay.TriggerGraphError != "" {
		fmt.Fprintf(&b, "- trigger_graph_error: `%s`\n", report.Replay.TriggerGraphError)
	}
	if report.Replay.ScenarioIdentitySHA != "" {
		fmt.Fprintf(&b, "- scenario identity fallback: `%s` `%s`\n", report.Replay.ScenarioIdentityTier, report.Replay.ScenarioIdentitySHA)
	}
	fmt.Fprintf(&b, "- data_set_identity: `%s`", report.Replay.DataSetStatus)
	if report.Replay.DataSetName != "" {
		fmt.Fprintf(&b, " `%s`", report.Replay.DataSetName)
	}
	if report.Replay.DataSetError != "" {
		fmt.Fprintf(&b, " (error: `%s`)", report.Replay.DataSetError)
	}
	fmt.Fprintf(&b, "\n")
	fmt.Fprintf(&b, "- final P1 resource stockpile checksum word: `%d`\n", report.Replay.FinalResourceStockpile)
	fmt.Fprintf(&b, "- final P1 object count checksum word: `%d`\n\n", report.Replay.FinalObjectCount)
	if report.Inputs.XSDataPath != "" {
		fmt.Fprintf(&b, "## XS Sidecar\n\n")
		fmt.Fprintf(&b, "- ok: `%t`\n", report.Sidecar.OK)
		fmt.Fprintf(&b, "- schema: `%s`\n", report.Sidecar.Schema)
		fmt.Fprintf(&b, "- magic: `%s`\n", report.Sidecar.Magic)
		fmt.Fprintf(&b, "- fixture: `%s`\n", report.Sidecar.Fixture)
		fmt.Fprintf(&b, "- rows: `%d`\n", report.Sidecar.Rows)
		fmt.Fprintf(&b, "- footer: `%t` rows_written=`%d`\n", report.Sidecar.HasFooter, report.Sidecar.RowsWritten)
		if report.Sidecar.Error != "" {
			fmt.Fprintf(&b, "- error: `%s`\n", report.Sidecar.Error)
		}
		fmt.Fprintf(&b, "\n")
	}
	if len(report.ParserGaps) > 0 {
		fmt.Fprintf(&b, "## Parser Gaps\n\n")
		for _, gap := range report.ParserGaps {
			fmt.Fprintf(&b, "- %s\n", gap)
		}
		fmt.Fprintf(&b, "\n")
	}
	fmt.Fprintf(&b, "## Lane Verdicts\n\n")
	for _, lane := range report.Lanes {
		fmt.Fprintf(&b, "### %s\n\n", lane.ID)
		fmt.Fprintf(&b, "- label: %s\n", lane.Label)
		fmt.Fprintf(&b, "- status: `%s`\n", lane.Status)
		if lane.PromotedTier != "" {
			fmt.Fprintf(&b, "- promoted_tier: `%s`\n", lane.PromotedTier)
		}
		fmt.Fprintf(&b, "- conclusion: %s\n", lane.Conclusion)
		for _, ev := range lane.Evidence {
			fmt.Fprintf(&b, "- `%s` `%s`: %s\n", ev.Source, ev.Status, ev.Observation)
		}
		fmt.Fprintf(&b, "\n")
	}
	data := strings.TrimRight(b.String(), "\n") + "\n"
	return os.WriteFile(path, []byte(data), 0644)
}

type readbackContext struct {
	expected ExpectedLedger
	dat      *datfile.Index
	scen     *scenario.File
	rec      *replay.File
	series   *replay.PlayerSeriesReport
	sidecar  *xsauthor.DataSchemaDecodeReport
}

func readExpectedLedger(path string) (ExpectedLedger, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return ExpectedLedger{}, err
	}
	var expected ExpectedLedger
	if err := json.Unmarshal(data, &expected); err != nil {
		return ExpectedLedger{}, fmt.Errorf("parse expected ledger: %w", err)
	}
	return expected, nil
}

func replayReadbackSummary(rec *replay.File, series *replay.PlayerSeriesReport) ReplayReadbackSummary {
	out := ReplayReadbackSummary{
		Duration:          series.Summary.Duration,
		DurationMS:        series.Summary.DurationMS,
		TriggerGraphOK:    rec.TriggerGraphOK,
		TriggerGraphError: rec.TriggerGraphErr,
		DataSetStatus:     rec.DataSet.Status,
		DataSetName:       rec.DataSet.ActiveDataSet,
		DataSetSource:     rec.DataSet.Source,
		DataSetConfidence: rec.DataSet.Confidence,
		DataSetError:      rec.DataSet.Error,
	}
	if rec.TriggerGraph != nil {
		out.ScenarioIdentityTier = "trigger_graph"
		out.ScenarioIdentitySHA = rec.TriggerGraph.SHA256
	} else if rec.Fallback != nil {
		out.ScenarioIdentityTier = rec.Fallback.Tier
		out.ScenarioIdentitySHA = rec.Fallback.SHA256
	}
	if p1 := findSeriesPlayer(series, 1); p1 != nil {
		out.FinalResourceStockpile = p1.FinalResourceStockpile
		out.FinalObjectCount = p1.FinalObjectCount
	}
	return out
}

func (ctx readbackContext) readLane(lane ExpectedLane) LaneReadback {
	readback := LaneReadback{
		ID:           lane.ID,
		Label:        lane.Label,
		ExpectedTier: lane.Tier,
		Status:       "inconclusive",
	}
	readback.Evidence = append(readback.Evidence, ctx.datEvidence(lane)...)
	readback.Evidence = append(readback.Evidence, ctx.scenarioEvidence(lane)...)
	switch lane.ID {
	case "lane_a_resource_commands":
		readback.Evidence = append(readback.Evidence, ctx.resourceCommandSidecarEvidence()...)
		if allEvidenceOK(readback.Evidence) {
			readback.Status = "promoted"
			readback.PromotedTier = "engine_verified_sidecar_oracle"
			readback.Conclusion = "DAT resource set/add/multiply commands match exact XS sidecar rows: food=1111, wood=100 then 300, and gold=222."
		}
	case "lane_b_attribute_hp_commands":
		readback.Evidence = append(readback.Evidence, ctx.hitPointSidecarEvidence()...)
		if allEvidenceOK(readback.Evidence) {
			readback.Status = "promoted"
			readback.PromotedTier = "engine_observed_survival_oracle"
			readback.Conclusion = "DAT hit-point set/add/multiply commands are observed through the survival oracle: the normal villager disappears under castle fire while the boosted villager remains through the later phase. This proves behavior, not direct HP values."
		} else {
			readback.Conclusion = "DAT hit-point rows are structurally present, but the sidecar must show the normal-villager death and boosted-villager survival before this lane is promoted."
		}
	case "lane_c_enable_unit":
		readback.Evidence = append(readback.Evidence, ctx.enableUnitSidecarEvidence()...)
		if allEvidenceOK(readback.Evidence) {
			readback.Status = "promoted"
			readback.PromotedTier = "engine_observed_create_bypass"
			readback.Conclusion = "Both pre-enable and post-enable scout create_object triggers produced scouts. This proves scenario create_object bypasses civ availability; it does not prove UI trainability."
		} else {
			readback.Conclusion = "Scout availability remains unresolved without exact sidecar counts or a UI/trainability oracle."
		}
	case "lane_d_upgrade_unit":
		readback.Evidence = append(readback.Evidence, ctx.upgradeUnitSidecarEvidence()...)
		if allEvidenceOK(readback.Evidence) {
			readback.Status = "promoted"
			readback.PromotedTier = "engine_verified_sidecar_oracle"
			readback.Conclusion = "DAT upgrade_unit transformed the existing militia into man-at-arms, and a later militia create appeared as the upgraded form."
		} else {
			readback.Conclusion = "Negative result for this scenario-research path: the command row fired structurally, but sidecar rows show militia stayed militia and later create_object militia also stayed militia."
		}
	case "lane_e_spawn_unit":
		readback.Evidence = append(readback.Evidence, ctx.spawnUnitSidecarEvidence()...)
		if allEvidenceOK(readback.Evidence) {
			readback.Status = "promoted"
			readback.PromotedTier = "engine_verified_sidecar_oracle"
			readback.Conclusion = "DAT spawn_unit added two villagers while the Town Center remained present, so this lane has a clean count oracle."
		} else {
			readback.Conclusion = "Negative result for this scenario-research path when the Town Center remains stable: the command row fired structurally, but villager count did not increase."
		}
	case "lane_f_tech_cost_modifier":
		readback.Evidence = append(readback.Evidence, ctx.techCostSidecarEvidence()...)
		if allEvidenceOK(readback.Evidence) {
			readback.Status = "promoted"
			readback.PromotedTier = "engine_observed_with_scope_limit"
			readback.Conclusion = "The sidecar proves unforced scenario research fired despite a 9000-food configured cost, then the separate post-cost-zero target also fired. This promotes the force=0 cost-bypass finding and leaves cost mutation as not isolated."
		} else {
			readback.Conclusion = "Tech-cost behavior remains unresolved until the sidecar shows both target-tech resource additions with exact stone totals."
		}
	case "lane_g_disable_tech":
		readback.Evidence = append(readback.Evidence, ctx.disableTechSidecarEvidence()...)
		if allEvidenceOK(readback.Evidence) {
			readback.Status = "promoted"
			readback.PromotedTier = "engine_verified_sidecar_oracle"
			readback.Conclusion = "DAT disable_tech blocked the later unforced target tech: the positive-control target added stone, but the disabled target did not add its +333."
		} else {
			readback.Conclusion = "Disable-tech semantics remain unresolved without the positive-control stone gain and the disabled-target no-op."
		}
	case "lane_h_raw_class_target":
		readback.Evidence = append(readback.Evidence, ctx.rawClassSidecarEvidence()...)
		readback.Conclusion = "The raw class-target probe can prove the trigger fired, but this sidecar cannot read class-wide hit-point mutation directly; keep this as a combat/UI-oracle frontier."
	case "lane_d_resource_control":
		readback.Evidence = append(readback.Evidence, ctx.resourceControlEvidence()...)
		if allEvidenceOK(readback.Evidence) {
			readback.Status = "promoted"
			readback.PromotedTier = "engine_observed_command_semantics"
			readback.Conclusion = "Scenario modify-resource control is engine-observed: the replay checksum matrix reaches the exact sentinel total 2345+3456=5801 after the trigger window."
		}
	case "lane_b_enable_unit":
		readback.Evidence = append(readback.Evidence, ctx.unitAddedEvidence(448, 1, "scout marker object appears after enable-unit trigger"))
		readback.Conclusion = "DAT row and scenario trigger are structurally correct, and the scout marker appears, but this replay does not prove the Stable UI/trainability state changed."
	case "lane_c_attribute_numeric_ops":
		readback.Evidence = append(readback.Evidence,
			ctx.unitAddedEvidence(83, 2, "two villager marker objects appear after HP DAT triggers"),
			ctx.unitAddedEvidence(74, 1, "scenario modify-attribute control militia marker appears"),
			ReadbackEvidence{Source: "replay object attributes", Status: "inconclusive", Observation: "current object-state readback does not expose live hit points for the spawned units in this single-player fixture"},
		)
		readback.Conclusion = "DAT effect rows and trigger markers line up, but HP set/add/multiply semantics remain inconclusive until a replay/XS/UI oracle exposes live hit points."
	case "lane_a_disable_tech":
		readback.Conclusion = "The disable_tech row and the later unforced target-tech trigger are structurally present, but this replay has no direct oracle for whether the target tech was blocked."
	default:
		readback.Conclusion = "No lane-specific replay oracle implemented."
	}
	if readback.Conclusion == "" {
		readback.Conclusion = "Structural and replay evidence collected."
	}
	if hasFailedEvidence(readback.Evidence) {
		readback.Status = "failed"
	}
	return readback
}

type a2ksem2Row struct {
	PhaseID   int
	TimeS     int
	Food      int
	Wood      int
	Stone     int
	Gold      int
	Pop       int
	Research  int
	Militia   int
	ManAtArms int
	Villager  int
	Scout     int
	TC        int
	Barracks  int
	Stable    int
}

func (ctx readbackContext) resourceCommandSidecarEvidence() []ReadbackEvidence {
	var out []ReadbackEvidence
	out = append(out, ctx.sidecarHasRowsEvidence(2, 12, 20, 35))
	out = append(out, ctx.rowValuesEvidence(2, "phase 002 baseline", map[string]int{"food": 0, "wood": 0, "stone": 0, "gold": 0}))
	out = append(out, ctx.rowValuesEvidence(12, "phase 012 after resource set commands", map[string]int{"food": 1111, "wood": 100}))
	out = append(out, ctx.rowValuesEvidence(20, "phase 020 after gold add command", map[string]int{"gold": 222}))
	out = append(out, ctx.rowValuesEvidence(35, "phase 035 after wood multiply command", map[string]int{"wood": 300}))
	return out
}

func (ctx readbackContext) hitPointSidecarEvidence() []ReadbackEvidence {
	var out []ReadbackEvidence
	out = append(out, ctx.sidecarHasRowsEvidence(40, 58, 68, 92))
	out = append(out, ctx.rowValuesEvidence(40, "phase 040 before normal-villager castle-fire control", map[string]int{"villager": 0}))
	out = append(out, ctx.rowValuesEvidence(58, "phase 058 after normal-villager castle-fire control", map[string]int{"villager": 0}))
	out = append(out, ctx.rowValuesEvidence(68, "phase 068 after boosted-villager spawn", map[string]int{"villager": 1}))
	out = append(out, ctx.rowValuesEvidence(92, "phase 092 boosted-villager survival check", map[string]int{"villager": 1}))
	if strings.Contains(ctx.sidecarFixture(), "run2_") {
		out = append(out, ReadbackEvidence{Source: "fixture hygiene", Status: "inconclusive", Observation: "run2 was polluted by an enemy castle near the P1 base; treat HP survival as suggestive until run3 repeats it in isolation"})
	}
	return out
}

func (ctx readbackContext) enableUnitSidecarEvidence() []ReadbackEvidence {
	var out []ReadbackEvidence
	out = append(out, ctx.sidecarHasRowsEvidence(96, 104))
	out = append(out, ctx.rowValuesEvidence(96, "phase 096 pre-enable scout create attempt", map[string]int{"scout": 1}))
	out = append(out, ctx.rowValuesEvidence(104, "phase 104 post-enable scout create attempt", map[string]int{"scout": 2}))
	return out
}

func (ctx readbackContext) upgradeUnitSidecarEvidence() []ReadbackEvidence {
	var out []ReadbackEvidence
	out = append(out, ctx.sidecarHasRowsEvidence(112, 124, 136))
	out = append(out, ctx.rowValuesEvidence(112, "phase 112 pre-upgrade unit count", map[string]int{"militia": 1, "maa": 0}))
	out = append(out, ctx.rowValuesEvidence(124, "phase 124 existing-unit upgrade count", map[string]int{"militia": 0, "maa": 1}))
	out = append(out, ctx.rowValuesEvidence(136, "phase 136 post-upgrade create count", map[string]int{"militia": 0, "maa": 2}))
	return out
}

func (ctx readbackContext) spawnUnitSidecarEvidence() []ReadbackEvidence {
	var out []ReadbackEvidence
	out = append(out, ctx.sidecarHasRowsEvidence(144, 160))
	before, okBefore := ctx.a2ksem2Row(144)
	after, okAfter := ctx.a2ksem2Row(160)
	if !okBefore || !okAfter {
		return append(out, ReadbackEvidence{Source: "XS sidecar a2ksem2", Status: "failed", Observation: "phase 144 or 160 is missing, cannot compare spawn_unit delta"})
	}
	if before.TC != after.TC {
		out = append(out, ReadbackEvidence{Source: "XS sidecar a2ksem2", Status: "failed", Observation: fmt.Sprintf("Town Center count changed %d -> %d during spawn lane; fixture is polluted", before.TC, after.TC)})
	} else {
		out = append(out, ReadbackEvidence{Source: "XS sidecar a2ksem2", Status: "ok", Observation: fmt.Sprintf("Town Center count stayed stable at %d during spawn lane", after.TC)})
	}
	delta := after.Villager - before.Villager
	if delta == 2 {
		out = append(out, ReadbackEvidence{Source: "XS sidecar a2ksem2", Status: "ok", Observation: fmt.Sprintf("villager count changed %d -> %d (+2)", before.Villager, after.Villager)})
	} else {
		out = append(out, ReadbackEvidence{Source: "XS sidecar a2ksem2", Status: "failed", Observation: fmt.Sprintf("villager count changed %d -> %d (%+d), expected +2", before.Villager, after.Villager, delta)})
	}
	return out
}

func (ctx readbackContext) techCostSidecarEvidence() []ReadbackEvidence {
	var out []ReadbackEvidence
	out = append(out, ctx.sidecarHasRowsEvidence(178, 202))
	out = append(out, ctx.rowValuesEvidence(178, "phase 178 after unforced 9000-food target tech", map[string]int{"stone": 777}))
	out = append(out, ctx.rowValuesEvidence(202, "phase 202 after set-cost-zero target tech", map[string]int{"stone": 1665}))
	out = append(out, ReadbackEvidence{Source: "semantics scope", Status: "ok", Observation: "stone=777 at phase 178 means scenario research with force=0 bypassed the configured target-tech cost; this does not isolate set_tech_cost enforcement"})
	return out
}

func (ctx readbackContext) disableTechSidecarEvidence() []ReadbackEvidence {
	var out []ReadbackEvidence
	out = append(out, ctx.sidecarHasRowsEvidence(216, 236))
	out = append(out, ctx.rowValuesEvidence(216, "phase 216 after zero-cost positive-control target", map[string]int{"stone": 2109}))
	out = append(out, ctx.rowValuesEvidence(236, "phase 236 after disabled target attempt", map[string]int{"stone": 2109}))
	before, okBefore := ctx.a2ksem2Row(216)
	after, okAfter := ctx.a2ksem2Row(236)
	if okBefore && okAfter && after.Research == before.Research+1 {
		out = append(out, ReadbackEvidence{Source: "XS sidecar a2ksem2", Status: "ok", Observation: fmt.Sprintf("research count changed %d -> %d, consistent with the forced disable command and no additional disabled-target research", before.Research, after.Research)})
	} else if okBefore && okAfter {
		out = append(out, ReadbackEvidence{Source: "XS sidecar a2ksem2", Status: "failed", Observation: fmt.Sprintf("research count changed %d -> %d; expected exactly one forced-disable bump", before.Research, after.Research)})
	}
	return out
}

func (ctx readbackContext) rawClassSidecarEvidence() []ReadbackEvidence {
	var out []ReadbackEvidence
	out = append(out, ctx.sidecarHasRowsEvidence(244, 260))
	out = append(out, ctx.rowValuesEvidence(244, "phase 244 before raw class-target probe", map[string]int{"stone": 2109}))
	out = append(out, ctx.rowValuesEvidence(260, "phase 260 after raw class-target probe", map[string]int{"stone": 2109}))
	out = append(out, ReadbackEvidence{Source: "semantics scope", Status: "inconclusive", Observation: "no sidecar field directly exposes class-wide hit-point changes, so a stable stone/object row only proves the probe window completed"})
	return out
}

func (ctx readbackContext) sidecarHasRowsEvidence(phases ...int) ReadbackEvidence {
	if ctx.sidecar == nil {
		return ReadbackEvidence{Source: "XS sidecar a2ksem2", Status: "inconclusive", Observation: "no .xsdat sidecar supplied"}
	}
	if !ctx.sidecar.OK {
		return ReadbackEvidence{Source: "XS sidecar a2ksem2", Status: "failed", Observation: "sidecar did not decode cleanly"}
	}
	var missing []string
	for _, phase := range phases {
		if _, ok := ctx.a2ksem2Row(phase); !ok {
			missing = append(missing, fmt.Sprintf("%03d", phase))
		}
	}
	if len(missing) > 0 {
		return ReadbackEvidence{Source: "XS sidecar a2ksem2", Status: "failed", Observation: fmt.Sprintf("missing phase row(s): %s", strings.Join(missing, ", "))}
	}
	return ReadbackEvidence{Source: "XS sidecar a2ksem2", Status: "ok", Observation: fmt.Sprintf("required phase row(s) present: %s", joinPhases(phases))}
}

func (ctx readbackContext) rowValuesEvidence(phase int, label string, expected map[string]int) ReadbackEvidence {
	row, ok := ctx.a2ksem2Row(phase)
	if !ok {
		return ReadbackEvidence{Source: "XS sidecar a2ksem2", Status: "failed", Observation: fmt.Sprintf("%s: phase %03d is missing", label, phase)}
	}
	var parts []string
	for _, key := range sortedKeys(expected) {
		actual := row.sidecarValue(key)
		want := expected[key]
		parts = append(parts, fmt.Sprintf("%s=%d", key, actual))
		if actual != want {
			return ReadbackEvidence{Source: "XS sidecar a2ksem2", Status: "failed", Observation: fmt.Sprintf("%s: %s, expected %s=%d", label, strings.Join(parts, " "), key, want)}
		}
	}
	return ReadbackEvidence{Source: "XS sidecar a2ksem2", Status: "ok", Observation: fmt.Sprintf("%s: %s", label, strings.Join(parts, " "))}
}

func (row a2ksem2Row) sidecarValue(key string) int {
	switch key {
	case "food":
		return row.Food
	case "wood":
		return row.Wood
	case "stone":
		return row.Stone
	case "gold":
		return row.Gold
	case "pop":
		return row.Pop
	case "research":
		return row.Research
	case "militia":
		return row.Militia
	case "maa", "man_at_arms":
		return row.ManAtArms
	case "villager":
		return row.Villager
	case "scout":
		return row.Scout
	case "tc", "town_center":
		return row.TC
	case "barracks":
		return row.Barracks
	case "stable":
		return row.Stable
	default:
		return 0
	}
}

func (ctx readbackContext) a2ksem2Row(phase int) (a2ksem2Row, bool) {
	if ctx.sidecar == nil || !ctx.sidecar.OK {
		return a2ksem2Row{}, false
	}
	for _, row := range ctx.sidecar.Rows {
		if row.PhaseID != phase {
			continue
		}
		fields := row.Fields
		return a2ksem2Row{
			PhaseID:   row.PhaseID,
			TimeS:     row.TimeS,
			Food:      roundAny(fields["p1_food_attr0"]),
			Wood:      roundAny(fields["p1_wood_attr1"]),
			Stone:     roundAny(fields["p1_stone_attr2"]),
			Gold:      roundAny(fields["p1_gold_attr3"]),
			Pop:       roundAny(fields["p1_population_attr11"]),
			Research:  roundAny(fields["p1_research_count_attr21"]),
			Militia:   intAny(fields["p1_militia_74_count"]),
			ManAtArms: intAny(fields["p1_man_at_arms_75_count"]),
			Villager:  intAny(fields["p1_villager_83_count"]),
			Scout:     intAny(fields["p1_scout_448_count"]),
			TC:        intAny(fields["p1_town_center_109_count"]),
			Barracks:  intAny(fields["p1_barracks_12_count"]),
			Stable:    intAny(fields["p1_stable_101_count"]),
		}, true
	}
	return a2ksem2Row{}, false
}

func (ctx readbackContext) sidecarFixture() string {
	if ctx.sidecar == nil {
		return ""
	}
	return stringAny(ctx.sidecar.Header["fixture"])
}

func (ctx readbackContext) datEvidence(lane ExpectedLane) []ReadbackEvidence {
	var out []ReadbackEvidence
	for _, id := range expectedEffectIDs(lane) {
		effect, ok := ctx.dat.Effect(id)
		if !ok {
			out = append(out, ReadbackEvidence{Source: "kit dat effect", Status: "failed", Observation: fmt.Sprintf("effect %d is missing", id)})
			continue
		}
		out = append(out, ReadbackEvidence{Source: "kit dat effect", Status: "ok", Observation: fmt.Sprintf("effect %d `%s` has %d command(s)", id, effect.Name, len(effect.Commands))})
	}
	for _, claim := range lane.Commands {
		out = append(out, ctx.commandEvidence(lane, claim))
	}
	for _, id := range expectedTechIDs(lane) {
		tech, ok := ctx.dat.Tech(id)
		if !ok {
			out = append(out, ReadbackEvidence{Source: "kit dat tech", Status: "failed", Observation: fmt.Sprintf("tech %d is missing", id)})
			continue
		}
		out = append(out, ReadbackEvidence{Source: "kit dat tech", Status: "ok", Observation: fmt.Sprintf("tech %d `%s` points at effect %d", id, tech.Name, tech.EffectID)})
	}
	return out
}

func (ctx readbackContext) commandEvidence(lane ExpectedLane, claim CommandClaim) ReadbackEvidence {
	ids := expectedEffectIDs(lane)
	if len(ids) == 0 {
		return ReadbackEvidence{Source: "kit dat effect command", Status: "inconclusive", Observation: fmt.Sprintf("no expected effect IDs are listed for `%s`; cannot verify the command row structurally", claim.Kind)}
	}
	for _, id := range ids {
		effect, ok := ctx.dat.Effect(id)
		if !ok {
			continue
		}
		for _, command := range effect.Commands {
			if commandMatchesClaim(command, claim) {
				return ReadbackEvidence{Source: "kit dat effect command", Status: "ok", Observation: fmt.Sprintf("effect %d command %d matches `%s`", id, command.Index, claim.Kind)}
			}
		}
	}
	return ReadbackEvidence{Source: "kit dat effect command", Status: "failed", Observation: fmt.Sprintf("no expected effect command matched `%s`", claim.Kind)}
}

func commandMatchesClaim(command datfile.EffectCommand, claim CommandClaim) bool {
	if claim.RawType != nil && command.Type != *claim.RawType {
		return false
	}
	if claim.Kind != "" && command.Type != commandTypeForKind(claim.Kind) {
		return false
	}
	if claim.UnitID != nil && !commandHasInt16(command, *claim.UnitID) {
		return false
	}
	if claim.AttributeID != nil && !commandHasInt16(command, *claim.AttributeID) {
		return false
	}
	if claim.TechID != nil && !commandHasInt(command, *claim.TechID) {
		return false
	}
	if claim.Amount != nil && command.D != *claim.Amount {
		return false
	}
	return true
}

func commandTypeForKind(kind string) uint8 {
	switch kind {
	case "resource_modifier":
		return 1
	case "set_attribute":
		return 0
	case "enable_unit":
		return 2
	case "upgrade_unit":
		return 3
	case "add_attribute":
		return 4
	case "multiply_attribute":
		return 5
	case "resource_multiplier":
		return 6
	case "spawn_unit":
		return 7
	case "set_tech_cost":
		return 100
	case "add_tech_cost", "tech_cost_modifier":
		return 101
	case "disable_tech":
		return 102
	case "raw_type4_class_target_probe":
		return 4
	default:
		return 255
	}
}

func commandHasInt16(command datfile.EffectCommand, value int16) bool {
	return command.A == value || command.B == value || command.C == value || int16(math.Round(float64(command.D))) == value
}

func commandHasInt(command datfile.EffectCommand, value int) bool {
	return int(command.A) == value || int(command.B) == value || int(command.C) == value || int(math.Round(float64(command.D))) == value
}

func (ctx readbackContext) scenarioEvidence(lane ExpectedLane) []ReadbackEvidence {
	if names := scenarioTriggerNamesForLane(lane); len(names) > 0 {
		var out []ReadbackEvidence
		for _, name := range names {
			out = append(out, ctx.triggerEvidence(name))
		}
		return out
	}
	if lane.ScenarioTriggerName == "" {
		return nil
	}
	if lane.ID == "lane_c_attribute_numeric_ops" {
		names := []string{"A2KSEM 045 DAT ADD HP", "A2KSEM 060 DAT MULTIPLY HP", "A2KSEM 115 SCENARIO ATTR CONTROL"}
		var out []ReadbackEvidence
		for _, name := range names {
			out = append(out, ctx.triggerEvidence(name))
		}
		return out
	}
	return []ReadbackEvidence{ctx.triggerEvidence(lane.ScenarioTriggerName)}
}

func scenarioTriggerNamesForLane(lane ExpectedLane) []string {
	switch lane.ID {
	case "lane_a_resource_commands":
		return []string{
			"A2KSEM2 008 RESOURCE SET WOOD BASELINE",
			"A2KSEM2 012 RESOURCE SET FOOD",
			"A2KSEM2 020 RESOURCE ADD GOLD",
			"A2KSEM2 030 RESOURCE MULTIPLY WOOD",
			"A2KSEM2 035 RESOURCE AFTER MULTIPLY",
		}
	case "lane_b_attribute_hp_commands":
		return []string{
			"A2KSEM2 040 HP CONTROL BEFORE",
			"A2KSEM2 042 HP CONTROL VILLAGER",
			"A2KSEM2 058 HP CONTROL AFTER",
			"A2KSEM2 060 HP COMMANDS",
			"A2KSEM2 064 HP TEST VILLAGER",
			"A2KSEM2 068 HP TEST SPAWNED",
			"A2KSEM2 092 HP TEST AFTER",
		}
	case "lane_c_enable_unit":
		return []string{
			"A2KSEM2 096 SCOUT CREATE BEFORE ENABLE",
			"A2KSEM2 100 ENABLE SCOUT",
			"A2KSEM2 104 SCOUT CREATE AFTER ENABLE",
		}
	case "lane_d_upgrade_unit":
		return []string{
			"A2KSEM2 108 CREATE UPGRADE MILITIA",
			"A2KSEM2 112 UPGRADE BEFORE",
			"A2KSEM2 116 UPGRADE MILITIA",
			"A2KSEM2 124 UPGRADE AFTER RESEARCH",
			"A2KSEM2 128 CREATE MILITIA AFTER UPGRADE",
			"A2KSEM2 136 UPGRADE AFTER NEW CREATE",
		}
	case "lane_e_spawn_unit":
		return []string{
			"A2KSEM2 144 SPAWN BEFORE",
			"A2KSEM2 148 SPAWN VILLAGERS",
			"A2KSEM2 160 SPAWN AFTER",
		}
	case "lane_f_tech_cost_modifier":
		return []string{
			"A2KSEM2 168 COST TARGET BEFORE MOD",
			"A2KSEM2 178 COST BEFORE RESULT",
			"A2KSEM2 184 SET TARGET2 COST ZERO",
			"A2KSEM2 188 COST TARGET2 AFTER MOD",
			"A2KSEM2 202 COST AFTER RESULT",
		}
	case "lane_g_disable_tech":
		return []string{
			"A2KSEM2 208 DISABLE POSITIVE CONTROL",
			"A2KSEM2 216 DISABLE POSITIVE RESULT",
			"A2KSEM2 220 DISABLE TARGET",
			"A2KSEM2 224 DISABLED TARGET ATTEMPT",
			"A2KSEM2 236 DISABLE AFTER",
		}
	case "lane_h_raw_class_target":
		return []string{
			"A2KSEM2 244 RAW CLASS BEFORE",
			"A2KSEM2 248 RAW CLASS HP",
			"A2KSEM2 260 RAW CLASS AFTER",
		}
	default:
		return nil
	}
}

func (ctx readbackContext) triggerEvidence(name string) ReadbackEvidence {
	trig, ok := ctx.triggerByName(name)
	if !ok {
		return ReadbackEvidence{Source: "kit scen effects", Status: "failed", Observation: fmt.Sprintf("trigger `%s` is missing", name)}
	}
	return ReadbackEvidence{Source: "kit scen effects", Status: "ok", Observation: fmt.Sprintf("trigger `%s` is present with %d effect(s)", name, trig.Effects)}
}

func (ctx readbackContext) triggerByName(name string) (scenario.TriggerSummary, bool) {
	if ctx.scen.Triggers == nil {
		return scenario.TriggerSummary{}, false
	}
	for _, trig := range ctx.scen.Triggers.Triggers {
		if trig.Name == name {
			return trig, true
		}
	}
	return scenario.TriggerSummary{}, false
}

func (ctx readbackContext) resourceControlEvidence() []ReadbackEvidence {
	var out []ReadbackEvidence
	trig, ok := ctx.triggerByName("A2KSEM 095 SCENARIO RESOURCE CONTROL")
	if !ok {
		return []ReadbackEvidence{{Source: "kit scen effects", Status: "failed", Observation: "resource-control trigger missing"}}
	}
	seen := map[int]int{}
	for _, effect := range trig.EffectData {
		if effect.Type == 52 && effect.Operation == 1 {
			seen[effect.Resource] += effect.Quantity
		}
	}
	if seen[0] == 2345 && seen[3] == 3456 {
		out = append(out, ReadbackEvidence{Source: "kit scen effects", Status: "ok", Observation: "trigger adds food=2345 and gold=3456"})
	} else {
		out = append(out, ReadbackEvidence{Source: "kit scen effects", Status: "failed", Observation: fmt.Sprintf("resource additions were food=%d gold=%d, expected 2345/3456", seen[0], seen[3])})
	}
	if sample := firstResourceSampleAtOrAbove(ctx.series, 1, 95000, 5801); sample != nil {
		out = append(out, ReadbackEvidence{Source: "kit replay player-series", Status: "ok", Observation: fmt.Sprintf("P1 resource checksum word reached 5801 at %s", sample.Time)})
	} else if p1 := findSeriesPlayer(ctx.series, 1); p1 != nil {
		out = append(out, ReadbackEvidence{Source: "kit replay player-series", Status: "failed", Observation: fmt.Sprintf("P1 final resource checksum word is %d, expected 5801", p1.FinalResourceStockpile)})
	} else {
		out = append(out, ReadbackEvidence{Source: "kit replay player-series", Status: "failed", Observation: "P1 checksum series is missing"})
	}
	return out
}

func (ctx readbackContext) unitAddedEvidence(unitID, count int, label string) ReadbackEvidence {
	if p1 := findSeriesPlayer(ctx.series, 1); p1 != nil {
		for _, item := range p1.AddedUnitTypes {
			if item.UnitID == unitID {
				if item.Count == count {
					return ReadbackEvidence{Source: "kit replay player-series", Status: "ok", Observation: fmt.Sprintf("%s: observed unit %d `%s` added x%d", label, unitID, item.UnitName, item.Count)}
				}
				return ReadbackEvidence{Source: "kit replay player-series", Status: "failed", Observation: fmt.Sprintf("%s: observed unit %d added x%d, expected x%d", label, unitID, item.Count, count)}
			}
		}
	}
	return ReadbackEvidence{Source: "kit replay player-series", Status: "failed", Observation: fmt.Sprintf("%s: unit %d additions not observed", label, unitID)}
}

func expectedEffectIDs(lane ExpectedLane) []int {
	var ids []int
	if lane.DatEffectID != nil {
		ids = append(ids, *lane.DatEffectID)
	}
	ids = append(ids, lane.DatEffectIDs...)
	sort.Ints(ids)
	return ids
}

func expectedTechIDs(lane ExpectedLane) []int {
	var ids []int
	if lane.DatTechID != nil {
		ids = append(ids, *lane.DatTechID)
	}
	ids = append(ids, lane.DatTechIDs...)
	sort.Ints(ids)
	return ids
}

func findSeriesPlayer(series *replay.PlayerSeriesReport, playerID int) *replay.PlayerSeriesPlayer {
	if series == nil {
		return nil
	}
	for i := range series.Players {
		if series.Players[i].PlayerID == playerID {
			return &series.Players[i]
		}
	}
	return nil
}

func firstResourceSampleAtOrAbove(series *replay.PlayerSeriesReport, playerID, minTimeMS int, value uint32) *replay.PlayerStateSample {
	player := findSeriesPlayer(series, playerID)
	if player == nil {
		return nil
	}
	for i := range player.SamplesOut {
		sample := &player.SamplesOut[i]
		if sample.TimeMS >= minTimeMS && sample.ResourceStockpile == value {
			return sample
		}
	}
	return nil
}

func joinPhases(phases []int) string {
	var parts []string
	for _, phase := range phases {
		parts = append(parts, fmt.Sprintf("%03d", phase))
	}
	return strings.Join(parts, ", ")
}

func sortedKeys(values map[string]int) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func intAny(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int8:
		return int(typed)
	case int16:
		return int(typed)
	case int32:
		return int(typed)
	case int64:
		return int(typed)
	case uint:
		return int(typed)
	case uint8:
		return int(typed)
	case uint16:
		return int(typed)
	case uint32:
		return int(typed)
	case uint64:
		return int(typed)
	case float32:
		return int(math.Round(float64(typed)))
	case float64:
		return int(math.Round(typed))
	case json.Number:
		asInt, err := typed.Int64()
		if err == nil {
			return int(asInt)
		}
		asFloat, err := typed.Float64()
		if err == nil {
			return int(math.Round(asFloat))
		}
		return 0
	default:
		return 0
	}
}

func roundAny(value any) int {
	return intAny(value)
}

func stringAny(value any) string {
	if value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return typed
	case fmt.Stringer:
		return typed.String()
	default:
		return fmt.Sprint(typed)
	}
}

func allEvidenceOK(evidence []ReadbackEvidence) bool {
	for _, ev := range evidence {
		if ev.Status != "ok" {
			return false
		}
	}
	return len(evidence) > 0
}

func hasFailedEvidence(evidence []ReadbackEvidence) bool {
	for _, ev := range evidence {
		if ev.Status == "failed" {
			return true
		}
	}
	return false
}
