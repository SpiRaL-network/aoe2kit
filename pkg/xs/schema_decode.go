package xs

import (
	"fmt"
	"os"
	"strings"
)

type DataSchemaDecodeOptions struct {
	Schema string
}

// MaxSidecarBytes is the largest sidecar payload the engine can retain for a
// single XS file. Keep this in the shared decoder so generators and readers
// report against the same contract.
const MaxSidecarBytes = (1 << 20) - 1

type DataSchemaDecodeReport struct {
	Path         string                  `json:"path,omitempty"`
	OK           bool                    `json:"ok"`
	Schema       string                  `json:"schema"`
	Verification VerificationClaim       `json:"verification"`
	Header       map[string]any          `json:"header,omitempty"`
	Rows         []DataSchemaRow         `json:"rows,omitempty"`
	Footer       map[string]any          `json:"footer,omitempty"`
	Summary      DataSchemaDecodeSummary `json:"summary"`
	Warnings     []string                `json:"warnings,omitempty"`
	Errors       []string                `json:"errors,omitempty"`
}

type DataSchemaDecodeSummary struct {
	SizeBytes     int            `json:"size_bytes"`
	ByteCap       int            `json:"byte_cap"`
	WithinByteCap bool           `json:"within_byte_cap"`
	BytesConsumed int            `json:"bytes_consumed"`
	Rows          int            `json:"rows"`
	CompleteRows  int            `json:"complete_rows"`
	HasFooter     bool           `json:"has_footer"`
	RemainingFrom int            `json:"remaining_from,omitempty"`
	TagCounts     map[string]int `json:"tag_counts,omitempty"`
}

type DataSchemaRow struct {
	Index   int            `json:"index"`
	Offset  int            `json:"offset"`
	PhaseID int            `json:"phase_id,omitempty"`
	TimeS   int            `json:"time_s,omitempty"`
	Fields  map[string]any `json:"fields"`
}

func DecodeDataFileSchema(path string, opts DataSchemaDecodeOptions) (DataSchemaDecodeReport, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return DataSchemaDecodeReport{}, err
	}
	report := DecodeDataBytesSchema(data, opts)
	report.Path = path
	return report, nil
}

func DecodeDataBytesSchema(data []byte, opts DataSchemaDecodeOptions) DataSchemaDecodeReport {
	schema := normalizeDataSchema(opts.Schema)
	report := DataSchemaDecodeReport{
		Schema: schema,
		Verification: StructureVerifiedClaim(
			"XS sidecar schema decode uses fixture-specific write order. It verifies byte structure and complete rows; engine meaning still depends on the scenario that produced the sidecar.",
		),
		Summary: DataSchemaDecodeSummary{SizeBytes: len(data), ByteCap: MaxSidecarBytes, WithinByteCap: len(data) <= MaxSidecarBytes, TagCounts: map[string]int{}},
	}
	if len(data) > MaxSidecarBytes {
		report.Warnings = append(report.Warnings, fmt.Sprintf("sidecar exceeds engine byte cap: %d > %d", len(data), MaxSidecarBytes))
	}
	switch schema {
	case "rtv12-executioner":
		decodeRTV12Executioner(data, &report)
	case "rtv13-directed-attribution":
		decodeRTV13DirectedAttribution(data, &report)
	case "rtv14-castle-kill-calibration":
		decodeRTV14CastleKillCalibration(data, &report)
	case "rtv141-castle-kill-calibration":
		decodeRTV141CastleKillCalibration(data, &report)
	case "rtv15-castle-kill-ground-truth":
		decodeRTV15CastleKillGroundTruth(data, &report)
	case "rtv16-semantic-promotion":
		decodeRTV16SemanticPromotion(data, &report)
	case "rtv17-packed-test":
		decodeRTV17PackedTest(data, &report)
	case "a2ksem2-dat-command-semantics":
		decodeA2KSEM2DATCommandSemantics(data, &report)
	case "a2k-custom-store":
		decodePhaseRows(data, &report, a2kCustomStoreHeaderFields, a2kCustomStoreRowFields, a2kCustomStoreFooterFields)
	case "a2k-engine-harness":
		decodePhaseRows(data, &report, a2kEngineHarnessHeaderFields, a2kEngineHarnessMarkerFields, a2kEngineHarnessFooterFields)
	case "a2k-barracks-wall-v2":
		decodeBarracksWallV2(data, &report)
	case "a2k-trig-long":
		decodeTrigLong(data, &report)
	case "a2k-long-v3":
		decodeLongV3(data, &report)
	case "a2k-snake-v0":
		decodeSnakeV0(data, &report)
	case "a2k-z-bouncing-ball":
		decodeZBouncingBall(data, &report)
	default:
		report.Errors = append(report.Errors, fmt.Sprintf("unsupported xsdat schema %q", opts.Schema))
	}
	recountSchemaTags(&report)
	if len(report.Errors) == 0 && report.Summary.BytesConsumed == 0 {
		report.Summary.BytesConsumed = len(data)
	}
	report.Summary.Rows = len(report.Rows)
	report.Summary.CompleteRows = len(report.Rows)
	report.OK = len(report.Errors) == 0
	return report
}

func recountSchemaTags(report *DataSchemaDecodeReport) {
	for _, row := range report.Rows {
		if tag, ok := row.Fields["tag"].(string); ok {
			report.Summary.TagCounts[tag]++
		}
	}
	if tag, ok := report.Footer["tag"].(string); ok {
		report.Summary.TagCounts[tag]++
	}
}

func normalizeDataSchema(schema string) string {
	schema = strings.ToLower(strings.TrimSpace(schema))
	switch schema {
	case "rtv12", "v12", "v12-executioner", "rtv12-executioner", "executioner":
		return "rtv12-executioner"
	case "rtv13", "v13", "rtv13-directed", "rtv13-directed-attribution", "directed-attribution":
		return "rtv13-directed-attribution"
	case "rtv14", "v14", "rtv14-castle", "rtv14-castle-kill", "rtv14-castle-kill-calibration", "castle-kill-calibration":
		return "rtv14-castle-kill-calibration"
	case "rtv141", "v141", "v14.1", "rtv14.1", "rtv141-castle", "rtv141-castle-kill", "rtv141-castle-kill-calibration",
		"rtv142", "v142", "v14.2", "rtv14.2", "rtv142-castle", "rtv142-castle-kill", "rtv142-castle-kill-calibration":
		return "rtv141-castle-kill-calibration"
	case "rtv15", "v15", "rtv15-castle", "rtv15-castle-kill", "rtv15-castle-kill-ground-truth", "castle-kill-ground-truth":
		return "rtv15-castle-kill-ground-truth"
	case "rtv16", "v16", "rtv16-semantic", "rtv16-semantic-promotion", "semantic-promotion":
		return "rtv16-semantic-promotion"
	case "rtv17", "v17", "rtv17-packed", "rtv17-packed-test", "packed-test":
		return "rtv17-packed-test"
	case "a2ksem2", "datsem2", "dat-semantics-v2", "a2ksem2-dat-command-semantics":
		return "a2ksem2-dat-command-semantics"
	case "towerstore", "tower-store", "a2k-tower-store", "a2k-per-tower-store", "customstore", "custom-store", "a2k-custom-store", "a2k_custom_store":
		return "a2k-custom-store"
	case "harness", "engine-harness", "a2k-engine-harness", "a2k-engine-harness-v1", "a2k_engine_harness", "a2k_engine_harness_v1":
		return "a2k-engine-harness"
	case "trig-long", "a2k-engine-harness-trig-long", "a2k_engine_harness_trig-long":
		return "a2k-trig-long"
	case "attr-sweep", "a2k-engine-harness-attr-sweep", "a2k_engine_harness_attr-sweep":
		return "a2k-trig-long"
	case "long-v2", "a2k-engine-harness-long-v2", "a2k_engine_harness_long-v2":
		return "a2k-trig-long"
	case "long-v3", "a2k-engine-harness-long-v3", "a2k_engine_harness_long-v3":
		return "a2k-long-v3"
	case "snake-v0", "a2k-snake-v0", "a2k_snake_v0":
		return "a2k-snake-v0"
	case "barracks-wall-v2", "a2k-barracks-wall-v2", "a2k_barracks_wall_v2":
		return "a2k-barracks-wall-v2"
	case "z-bouncing-ball", "a2k-z-bouncing-ball", "a2k_z_bouncing_ball":
		return "a2k-z-bouncing-ball"
	default:
		return schema
	}
}

var rtv12HeaderFields = []schemaField{
	{name: "magic", typ: "string"},
	{name: "version", typ: "int"},
	{name: "fixture", typ: "string"},
}

var rtv12RowFields = []schemaField{
	{name: "tag", typ: "string"},
	{name: "phase_id", typ: "int"},
	{name: "expected_kind", typ: "int"},
	{name: "family", typ: "int"},
	{name: "outcome", typ: "int"},
	{name: "xs_time", typ: "int"},
	{name: "p1_resource_sum_carrier_food_attr0", typ: "float"},
	{name: "p1_wood_family_attr1", typ: "float"},
	{name: "p1_gold_outcome_attr3", typ: "float"},
	{name: "p1_stone_kind_attr2", typ: "float"},
	{name: "p1_kills_attr20", typ: "float"},
	{name: "p1_razings_attr43", typ: "float"},
	{name: "p1_killed_by_others_attr154", typ: "float"},
	{name: "p1_razed_by_others_attr155", typ: "float"},
	{name: "p1_kill_value_attr170", typ: "float"},
	{name: "p1_raze_value_attr172", typ: "float"},
	{name: "p2_kills_attr20", typ: "float"},
	{name: "p2_razings_attr43", typ: "float"},
	{name: "p2_killed_by_others_attr154", typ: "float"},
	{name: "p2_razed_by_others_attr155", typ: "float"},
	{name: "p2_kill_value_attr170", typ: "float"},
	{name: "p2_raze_value_attr172", typ: "float"},
	{name: "p3_kills_attr20", typ: "float"},
	{name: "p3_razings_attr43", typ: "float"},
	{name: "p3_killed_by_others_attr154", typ: "float"},
	{name: "p3_razed_by_others_attr155", typ: "float"},
	{name: "p3_kill_value_attr170", typ: "float"},
	{name: "p3_raze_value_attr172", typ: "float"},
	{name: "p1_gaia_kills_attr300", typ: "float"},
	{name: "p1_player1_kills_attr301", typ: "float"},
	{name: "p1_player2_kills_attr302", typ: "float"},
	{name: "p1_player3_kills_attr303", typ: "float"},
	{name: "p1_kills_by_gaia_attr325", typ: "float"},
	{name: "p1_kills_by_player1_attr326", typ: "float"},
	{name: "p1_kills_by_player2_attr327", typ: "float"},
	{name: "p1_kills_by_player3_attr328", typ: "float"},
	{name: "p1_gaia_razings_attr350", typ: "float"},
	{name: "p1_player1_razings_attr351", typ: "float"},
	{name: "p1_player2_razings_attr352", typ: "float"},
	{name: "p1_player3_razings_attr353", typ: "float"},
	{name: "p1_razings_by_gaia_attr375", typ: "float"},
	{name: "p1_razings_by_player1_attr376", typ: "float"},
	{name: "p1_razings_by_player2_attr377", typ: "float"},
	{name: "p1_razings_by_player3_attr378", typ: "float"},
	{name: "p1_cobra_car_748_count", typ: "int"},
	{name: "p2_militia_74_count", typ: "int"},
	{name: "p2_barracks_12_count", typ: "int"},
	{name: "p3_militia_74_count", typ: "int"},
	{name: "p3_barracks_12_count", typ: "int"},
	{name: "gaia_militia_74_count", typ: "int"},
	{name: "gaia_barracks_12_count", typ: "int"},
	{name: "p1_archer_4_count", typ: "int"},
}

var rtv12FooterFields = []schemaField{
	{name: "tag", typ: "string"},
	{name: "rows_written", typ: "int"},
	{name: "append_tag", typ: "string"},
	{name: "append_probe", typ: "int"},
}

var rtv13HeaderFields = rtv12HeaderFields

var rtv13RowFields = buildRTV13RowFields()

var rtv13FooterFields = rtv12FooterFields

var rtv14HeaderFields = rtv12HeaderFields

var rtv14RowFields = buildRTV14RowFields()

var rtv14FooterFields = rtv12FooterFields

var rtv141HeaderFields = rtv12HeaderFields

var rtv141RowFields = buildRTV141RowFields()

var rtv141FooterFields = rtv12FooterFields

var rtv15HeaderFields = rtv12HeaderFields

var rtv15RowFields = buildRTV15RowFields()

var rtv15FooterFields = rtv12FooterFields

var rtv16HeaderFields = rtv12HeaderFields

var rtv16RowFields = buildRTV16RowFields()

var rtv16FooterFields = rtv12FooterFields

var rtv17HeaderFields = rtv12HeaderFields

var rtv17RowFields = buildRTV17RowFields()

var rtv17FooterFields = rtv12FooterFields

var a2ksem2HeaderFields = []schemaField{
	{name: "magic", typ: "string"},
	{name: "version", typ: "int"},
	{name: "fixture", typ: "string"},
}

var a2ksem2RowFields = []schemaField{
	{name: "tag", typ: "string"},
	{name: "phase_id", typ: "int"},
	{name: "lane_id", typ: "int"},
	{name: "xs_time", typ: "int"},
	{name: "p1_food_attr0", typ: "float"},
	{name: "p1_wood_attr1", typ: "float"},
	{name: "p1_stone_attr2", typ: "float"},
	{name: "p1_gold_attr3", typ: "float"},
	{name: "p1_population_attr11", typ: "float"},
	{name: "p1_research_count_attr21", typ: "float"},
	{name: "p1_militia_74_count", typ: "int"},
	{name: "p1_man_at_arms_75_count", typ: "int"},
	{name: "p1_villager_83_count", typ: "int"},
	{name: "p1_scout_448_count", typ: "int"},
	{name: "p1_town_center_109_count", typ: "int"},
	{name: "p1_barracks_12_count", typ: "int"},
	{name: "p1_stable_101_count", typ: "int"},
}

var a2ksem2FooterFields = []schemaField{
	{name: "tag", typ: "string"},
	{name: "rows_written", typ: "int"},
}

var a2kCustomStoreHeaderFields = []schemaField{
	{name: "magic", typ: "string"},
	{name: "version", typ: "int"},
	{name: "fixture", typ: "string"},
	{name: "total_objects", typ: "int"},
	{name: "player_1_objects", typ: "int"},
	{name: "player_2_objects", typ: "int"},
	{name: "player_3_objects", typ: "int"},
	{name: "player_4_objects", typ: "int"},
	{name: "player_5_objects", typ: "int"},
	{name: "player_6_objects", typ: "int"},
	{name: "player_7_objects", typ: "int"},
	{name: "player_8_objects", typ: "int"},
	{name: "array_high_water", typ: "int"},
}

var a2kCustomStoreRowFields = []schemaField{
	{name: "tag", typ: "string"},
	{name: "game_time", typ: "int"},
	{name: "total_objects", typ: "int"},
	{name: "array_high_water", typ: "int"},
	{name: "player", typ: "int"},
	{name: "token_type", typ: "int"},
	{name: "token_id", typ: "int"},
	{name: "token_x", typ: "float"},
	{name: "token_y", typ: "float"},
	{name: "resolved_object_id", typ: "int"},
	{name: "neighbor_object_id", typ: "int"},
	{name: "state_before", typ: "int"},
	{name: "state_after", typ: "int"},
	{name: "neighbor_state_before", typ: "int"},
	{name: "neighbor_state_after", typ: "int"},
}

var a2kCustomStoreResourceFields = []schemaField{
	{name: "tag", typ: "string"},
	{name: "game_time", typ: "int"},
	{name: "p1_food", typ: "int"},
	{name: "p1_wood", typ: "int"},
	{name: "p1_gold", typ: "int"},
}

var a2kCustomStoreFooterFields = []schemaField{
	{name: "tag", typ: "string"},
	{name: "rows_written", typ: "int"},
	{name: "array_high_water", typ: "int"},
	{name: "integrity", typ: "int"},
}

var a2kEngineHarnessHeaderFields = []schemaField{
	{name: "magic", typ: "string"},
	{name: "version", typ: "int"},
	{name: "batch", typ: "string"},
	{name: "fact_count", typ: "int"},
}

var a2kEngineHarnessMarkerFields = []schemaField{
	{name: "tag", typ: "string"},
	{name: "batch", typ: "string"},
	{name: "fact_id", typ: "string"},
	{name: "ordinal", typ: "int"},
	{name: "expected", typ: "int"},
	{name: "observed", typ: "int"},
	{name: "pass", typ: "int"},
	{name: "game_time", typ: "int"},
}

var a2kEngineHarnessFooterFields = []schemaField{
	{name: "tag", typ: "string"},
	{name: "fact_count", typ: "int"},
	{name: "integrity", typ: "int"},
}

var a2kBarracksWallV2HeaderFields = []schemaField{
	{name: "magic", typ: "string"}, {name: "version", typ: "int"}, {name: "fixture", typ: "string"},
}

var a2kBarracksWallV2RowFields = []schemaField{
	{name: "tag", typ: "string"}, {name: "unit_id", typ: "int"}, {name: "owner", typ: "int"},
	{name: "x", typ: "float"}, {name: "y", typ: "float"}, {name: "attr_3", typ: "float"},
	{name: "attr_4", typ: "float"}, {name: "attr_200", typ: "float"}, {name: "attr_201", typ: "float"},
}

var a2kBarracksWallV2FooterFields = []schemaField{
	{name: "tag", typ: "string"}, {name: "integrity", typ: "int"},
}

var a2kTrigLongHeaderFields = []schemaField{
	{name: "magic", typ: "string"}, {name: "version", typ: "int"},
	{name: "fixture", typ: "string"}, {name: "fact_count", typ: "int"},
}

var a2kTrigLongBeforeFields = []schemaField{
	{name: "tag", typ: "string"}, {name: "fact_id", typ: "string"},
	{name: "ordinal", typ: "int"}, {name: "game_time", typ: "int"},
}

var a2kTrigLongResultFields = []schemaField{
	{name: "tag", typ: "string"}, {name: "fact_id", typ: "string"},
	{name: "fact_index", typ: "int"}, {name: "expected", typ: "int"},
	{name: "observed", typ: "int"}, {name: "status", typ: "int"},
	{name: "game_time", typ: "int"},
}

var a2kTrigLongRateFields = []schemaField{
	{name: "tag", typ: "string"}, {name: "game_time", typ: "int"},
	{name: "calls_this_second", typ: "int"}, {name: "turn", typ: "int"},
	{name: "world_time_ms", typ: "int"},
}

var a2kTrigLongFooterFields = []schemaField{
	{name: "tag", typ: "string"}, {name: "fact_count", typ: "int"},
	{name: "integrity", typ: "int"},
}

var a2kLongV3HeaderFields = []schemaField{
	{name: "magic", typ: "string"}, {name: "version", typ: "int"}, {name: "fixture", typ: "string"},
	{name: "fact_count", typ: "int"}, {name: "expected_records", typ: "int"},
}

var a2kSnakeV0HeaderFields = []schemaField{
	{name: "magic", typ: "string"}, {name: "version", typ: "int"}, {name: "fixture", typ: "string"},
	{name: "players", typ: "int"}, {name: "max_flame", typ: "int"}, {name: "apples", typ: "int"},
}

var a2kSnakeV0EventFields = []schemaField{
	{name: "tag", typ: "string"}, {name: "who", typ: "int"}, {name: "value", typ: "int"}, {name: "tick", typ: "int"}, {name: "game_time", typ: "int"},
}

var a2kSnakeV0ContactFields = []schemaField{
	{name: "tag", typ: "string"}, {name: "who", typ: "int"}, {name: "flame_owner", typ: "int"}, {name: "tick", typ: "int"}, {name: "game_time", typ: "int"}, {name: "hero_hp", typ: "float"},
}

var a2kSnakeV0SelfTestFields = []schemaField{
	{name: "tag", typ: "string"}, {name: "phase", typ: "int"}, {name: "tick", typ: "int"}, {name: "hero_hp", typ: "float"}, {name: "flame_hp", typ: "float"}, {name: "game_time", typ: "int"},
}

var a2kSnakeV0FooterFields = []schemaField{
	{name: "tag", typ: "string"}, {name: "eat_total", typ: "int"}, {name: "integrity", typ: "int"},
}

var a2kLongV3UnitFields = []schemaField{
	{name: "tag", typ: "string"}, {name: "fact_id", typ: "int"}, {name: "unit_type", typ: "int"},
	{name: "attr", typ: "int"}, {name: "value", typ: "float"}, {name: "game_time", typ: "int"},
}

var a2kLongV3TreatmentFields = []schemaField{
	{name: "tag", typ: "string"}, {name: "fact_id", typ: "int"}, {name: "attr", typ: "int"},
	{name: "state", typ: "int"}, {name: "before", typ: "float"}, {name: "after", typ: "float"}, {name: "game_time", typ: "int"},
}

var a2kLongV3TrailFields = []schemaField{
	{name: "tag", typ: "string"}, {name: "fact_id", typ: "int"}, {name: "mode", typ: "int"},
	{name: "density", typ: "int"}, {name: "live_count", typ: "float"}, {name: "death_count", typ: "float"}, {name: "game_time", typ: "int"},
}

var a2kLongV3CombatFields = []schemaField{
	{name: "tag", typ: "string"}, {name: "fact_id", typ: "int"}, {name: "variant", typ: "int"},
	{name: "enemy_hp", typ: "float"}, {name: "friend_hp", typ: "float"}, {name: "enemy_kills", typ: "int"},
	{name: "friend_kills", typ: "int"}, {name: "game_time", typ: "int"},
}

var a2kLongV3PassengerFields = []schemaField{
	{name: "tag", typ: "string"}, {name: "fact_id", typ: "int"}, {name: "kind", typ: "int"},
	{name: "before", typ: "float"}, {name: "after", typ: "float"}, {name: "state", typ: "int"}, {name: "game_time", typ: "int"},
}

var a2kLongV3ZFields = []schemaField{
	{name: "tag", typ: "string"}, {name: "fact_id", typ: "int"}, {name: "sample", typ: "int"},
	{name: "phase", typ: "int"}, {name: "expected", typ: "float"}, {name: "observed", typ: "float"},
	{name: "state", typ: "int"}, {name: "game_time", typ: "int"},
}

var a2kLongV3RateFields = []schemaField{
	{name: "tag", typ: "string"}, {name: "game_time", typ: "int"}, {name: "calls_this_second", typ: "int"},
	{name: "turn", typ: "int"}, {name: "world_time_ms", typ: "int"},
}

var a2kLongV3MilestoneFields = []schemaField{
	{name: "tag", typ: "string"}, {name: "ordinal", typ: "int"},
}

var a2kLongV3FooterFields = []schemaField{
	{name: "tag", typ: "string"}, {name: "fact_count", typ: "int"}, {name: "records", typ: "int"}, {name: "integrity", typ: "int"},
}

var a2kZBouncingBallHeaderFields = []schemaField{
	{name: "magic", typ: "string"}, {name: "version", typ: "int"}, {name: "fixture", typ: "string"},
	{name: "flame_count", typ: "int"}, {name: "ruler_count", typ: "int"},
}

var a2kZBouncingBallSampleFields = []schemaField{
	{name: "tag", typ: "string"}, {name: "tick", typ: "int"}, {name: "game_time", typ: "int"},
	{name: "ball_id", typ: "int"}, {name: "expected_z", typ: "float"}, {name: "ball_z", typ: "float"},
	{name: "twin_z", typ: "float"}, {name: "ruler_z0", typ: "float"}, {name: "ruler_z1", typ: "float"},
	{name: "ruler_z2", typ: "float"}, {name: "ruler_z3", typ: "float"}, {name: "ruler_z4", typ: "float"},
}

var a2kZBouncingBallFooterFields = []schemaField{
	{name: "tag", typ: "string"}, {name: "samples", typ: "int"}, {name: "integrity", typ: "int"},
}

type schemaField struct {
	name string
	typ  string
}

func decodeRTV12Executioner(data []byte, report *DataSchemaDecodeReport) {
	decodePhaseRows(data, report, rtv12HeaderFields, rtv12RowFields, rtv12FooterFields)
}

func decodeBarracksWallV2(data []byte, report *DataSchemaDecodeReport) {
	offset := 0
	header, next, err := readSchemaFields(data, offset, 0, a2kBarracksWallV2HeaderFields)
	if err != nil {
		report.Errors = append(report.Errors, err.Error())
		return
	}
	report.Header = header
	offset = next
	for offset < len(data) {
		rowOffset := offset
		tag, _, tagErr := readTypedValue(data, offset, 3, "string", false)
		if tagErr != nil {
			report.Errors = append(report.Errors, tagErr.Error())
			report.Summary.RemainingFrom = offset
			return
		}
		switch tag.String {
		case "unit":
			fields, after, fieldErr := readSchemaFields(data, offset, 3, a2kBarracksWallV2RowFields)
			if fieldErr != nil {
				report.Errors = append(report.Errors, fieldErr.Error())
				report.Summary.RemainingFrom = offset
				return
			}
			report.Rows = append(report.Rows, DataSchemaRow{Index: len(report.Rows), Offset: rowOffset, Fields: fields})
			offset = after
		case "end":
			footer, after, footerErr := readSchemaFields(data, offset, 3+len(report.Rows)*len(a2kBarracksWallV2RowFields), a2kBarracksWallV2FooterFields)
			if footerErr != nil {
				report.Errors = append(report.Errors, footerErr.Error())
				report.Summary.RemainingFrom = offset
				return
			}
			report.Footer = footer
			report.Summary.HasFooter = true
			offset = after
			if offset < len(data) {
				report.Errors = append(report.Errors, fmt.Sprintf("trailing bytes after footer at offset %d", offset))
				report.Summary.RemainingFrom = offset
			}
			return
		default:
			report.Errors = append(report.Errors, fmt.Sprintf("unexpected row tag %q at offset %d", tag.String, offset))
			report.Summary.RemainingFrom = offset
			return
		}
	}
	report.Warnings = append(report.Warnings, "sidecar has no end/footer row; decoded complete unit rows only")
}

func decodeRTV13DirectedAttribution(data []byte, report *DataSchemaDecodeReport) {
	decodePhaseRows(data, report, rtv13HeaderFields, rtv13RowFields, rtv13FooterFields)
}

func decodeRTV14CastleKillCalibration(data []byte, report *DataSchemaDecodeReport) {
	decodePhaseRows(data, report, rtv14HeaderFields, rtv14RowFields, rtv14FooterFields)
}

func decodeRTV141CastleKillCalibration(data []byte, report *DataSchemaDecodeReport) {
	decodePhaseRows(data, report, rtv141HeaderFields, rtv141RowFields, rtv141FooterFields)
}

func decodeRTV15CastleKillGroundTruth(data []byte, report *DataSchemaDecodeReport) {
	decodePhaseRows(data, report, rtv15HeaderFields, rtv15RowFields, rtv15FooterFields)
}

func decodeRTV16SemanticPromotion(data []byte, report *DataSchemaDecodeReport) {
	decodePhaseRows(data, report, rtv16HeaderFields, rtv16RowFields, rtv16FooterFields)
}

func decodeRTV17PackedTest(data []byte, report *DataSchemaDecodeReport) {
	decodePhaseRows(data, report, rtv17HeaderFields, rtv17RowFields, rtv17FooterFields)
}

func decodeA2KSEM2DATCommandSemantics(data []byte, report *DataSchemaDecodeReport) {
	decodePhaseRows(data, report, a2ksem2HeaderFields, a2ksem2RowFields, a2ksem2FooterFields)
}

func decodeTrigLong(data []byte, report *DataSchemaDecodeReport) {
	offset := 0
	header, next, err := readSchemaFields(data, offset, 0, a2kTrigLongHeaderFields)
	if err != nil {
		report.Errors = append(report.Errors, err.Error())
		return
	}
	report.Header = header
	offset = next
	for offset < len(data) {
		tag, _, tagErr := readTypedValue(data, offset, 0, "string", false)
		if tagErr != nil {
			report.Errors = append(report.Errors, tagErr.Error())
			report.Summary.RemainingFrom = offset
			return
		}
		rowOffset := offset
		var fields map[string]any
		switch tag.String {
		case "before":
			fields, offset, err = readSchemaFields(data, offset, 0, a2kTrigLongBeforeFields)
		case "result":
			fields, offset, err = readSchemaFields(data, offset, 0, a2kTrigLongResultFields)
		case "value":
			fields, offset, err = readTrigLongValue(data, offset)
		case "rate":
			fields, offset, err = readSchemaFields(data, offset, 0, a2kTrigLongRateFields)
		case "end":
			fields, offset, err = readSchemaFields(data, offset, 0, a2kTrigLongFooterFields)
			if err == nil {
				report.Footer = fields
				report.Summary.HasFooter = true
				if offset < len(data) {
					report.Errors = append(report.Errors, fmt.Sprintf("trailing bytes after footer at offset %d", offset))
					report.Summary.RemainingFrom = offset
				}
				report.Summary.BytesConsumed = offset
				return
			}
		default:
			err = fmt.Errorf("unexpected trig-long row tag %q at offset %d", tag.String, offset)
		}
		if err != nil {
			report.Errors = append(report.Errors, err.Error())
			report.Summary.RemainingFrom = rowOffset
			return
		}
		row := DataSchemaRow{Index: len(report.Rows), Offset: rowOffset, Fields: fields}
		row.PhaseID = intFromAny(fields["ordinal"])
		if row.PhaseID == 0 {
			row.PhaseID = intFromAny(fields["fact_index"])
		}
		row.TimeS = intFromAny(fields["game_time"])
		report.Rows = append(report.Rows, row)
	}
	report.Warnings = append(report.Warnings, "sidecar has no end/footer row; decoded complete trig-long rows only")
}

func decodeLongV3(data []byte, report *DataSchemaDecodeReport) {
	offset := 0
	header, next, err := readSchemaFields(data, offset, 0, a2kLongV3HeaderFields)
	if err != nil {
		report.Errors = append(report.Errors, err.Error())
		return
	}
	report.Header = header
	offset = next
	for offset < len(data) {
		rowOffset := offset
		tag, _, tagErr := readTypedValue(data, offset, 0, "string", false)
		if tagErr != nil {
			report.Errors = append(report.Errors, tagErr.Error())
			report.Summary.RemainingFrom = offset
			return
		}
		var fields map[string]any
		switch tag.String {
		case "unit":
			fields, offset, err = readSchemaFields(data, offset, 0, a2kLongV3UnitFields)
		case "treatment":
			fields, offset, err = readSchemaFields(data, offset, 0, a2kLongV3TreatmentFields)
		case "trail":
			fields, offset, err = readSchemaFields(data, offset, 0, a2kLongV3TrailFields)
		case "combat":
			fields, offset, err = readSchemaFields(data, offset, 0, a2kLongV3CombatFields)
		case "passenger":
			fields, offset, err = readSchemaFields(data, offset, 0, a2kLongV3PassengerFields)
		case "z":
			fields, offset, err = readSchemaFields(data, offset, 0, a2kLongV3ZFields)
		case "rate":
			fields, offset, err = readSchemaFields(data, offset, 0, a2kLongV3RateFields)
		case "milestone":
			fields, offset, err = readSchemaFields(data, offset, 0, a2kLongV3MilestoneFields)
		case "end":
			fields, offset, err = readSchemaFields(data, offset, 0, a2kLongV3FooterFields)
			if err == nil {
				report.Footer = fields
				report.Summary.HasFooter = true
				if offset < len(data) {
					report.Errors = append(report.Errors, fmt.Sprintf("trailing bytes after footer at offset %d", offset))
					report.Summary.RemainingFrom = offset
				}
				return
			}
		default:
			err = fmt.Errorf("unexpected long-v3 row tag %q at offset %d", tag.String, offset)
		}
		if err != nil {
			report.Errors = append(report.Errors, err.Error())
			report.Summary.RemainingFrom = rowOffset
			return
		}
		row := DataSchemaRow{Index: len(report.Rows), Offset: rowOffset, Fields: fields}
		row.PhaseID = intFromAny(fields["fact_id"])
		if row.PhaseID == 0 {
			row.PhaseID = intFromAny(fields["ordinal"])
		}
		row.TimeS = intFromAny(fields["game_time"])
		report.Rows = append(report.Rows, row)
	}
	report.Warnings = append(report.Warnings, "sidecar has no end/footer row; decoded complete long-v3 rows only")
}

func decodeSnakeV0(data []byte, report *DataSchemaDecodeReport) {
	offset := 0
	footerOffset := -1
	header, next, err := readSchemaFields(data, offset, 0, a2kSnakeV0HeaderFields)
	if err != nil {
		report.Errors = append(report.Errors, err.Error())
		return
	}
	report.Header = header
	offset = next
	report.Summary.BytesConsumed = offset
	for offset < len(data) {
		rowOffset := offset
		tag, _, tagErr := readTypedValue(data, offset, 0, "string", false)
		if tagErr != nil {
			report.Errors = append(report.Errors, tagErr.Error())
			report.Summary.RemainingFrom = offset
			report.Summary.BytesConsumed = offset
			return
		}
		var fields map[string]any
		switch tag.String {
		case "event", "tick", "retire", "eat", "grow", "shell", "death", "respawn":
			fields, offset, err = readSchemaFields(data, offset, 0, a2kSnakeV0EventFields)
		case "contact":
			fields, offset, err = readSchemaFields(data, offset, 0, a2kSnakeV0ContactFields)
		case "selftest":
			fields, offset, err = readSchemaFields(data, offset, 0, a2kSnakeV0SelfTestFields)
		case "end":
			fields, offset, err = readSchemaFields(data, offset, 0, a2kSnakeV0FooterFields)
			if err == nil {
				report.Footer = fields
				report.Summary.HasFooter = true
				footerOffset = offset
				if intFromAny(fields["integrity"]) != 424242 {
					report.Errors = append(report.Errors, "snake-v0 footer integrity mismatch")
				}
				report.Summary.BytesConsumed = offset
				continue
			}
		default:
			err = fmt.Errorf("unexpected snake-v0 row tag %q at offset %d", tag.String, offset)
		}
		if err != nil {
			report.Errors = append(report.Errors, err.Error())
			report.Summary.RemainingFrom = rowOffset
			report.Summary.BytesConsumed = rowOffset
			return
		}
		row := DataSchemaRow{Index: len(report.Rows), Offset: rowOffset, Fields: fields}
		row.PhaseID = intFromAny(fields["who"])
		row.TimeS = intFromAny(fields["game_time"])
		report.Rows = append(report.Rows, row)
		report.Summary.BytesConsumed = offset
	}
	report.Summary.BytesConsumed = offset
	if !report.Summary.HasFooter {
		report.Warnings = append(report.Warnings, "sidecar has no end/footer row; decoded complete snake-v0 rows only")
	} else if footerOffset >= 0 && footerOffset < len(data) {
		report.Warnings = append(report.Warnings, fmt.Sprintf("snake-v0 footer occurred before later rows at offset %d; all rows were decoded", footerOffset))
	}
}

func decodeZBouncingBall(data []byte, report *DataSchemaDecodeReport) {
	offset := 0
	header, next, err := readSchemaFields(data, offset, 0, a2kZBouncingBallHeaderFields)
	if err != nil {
		report.Errors = append(report.Errors, err.Error())
		return
	}
	report.Header = header
	offset = next
	for offset < len(data) {
		rowOffset := offset
		tag, _, tagErr := readTypedValue(data, offset, 0, "string", false)
		if tagErr != nil {
			report.Errors = append(report.Errors, tagErr.Error())
			report.Summary.RemainingFrom = offset
			return
		}
		var fields map[string]any
		switch tag.String {
		case "sample":
			fields, offset, err = readSchemaFields(data, offset, 0, a2kZBouncingBallSampleFields)
		case "end", "footer":
			fields, offset, err = readSchemaFields(data, offset, 0, a2kZBouncingBallFooterFields)
			if err == nil {
				report.Footer = fields
				report.Summary.HasFooter = true
				if intFromAny(fields["integrity"]) != 424242 {
					report.Errors = append(report.Errors, "z-bouncing-ball footer integrity mismatch")
				}
				if offset < len(data) {
					report.Errors = append(report.Errors, fmt.Sprintf("trailing bytes after footer at offset %d", offset))
					report.Summary.RemainingFrom = offset
				}
				report.Summary.BytesConsumed = offset
				return
			}
		default:
			err = fmt.Errorf("unexpected z-bouncing-ball row tag %q at offset %d", tag.String, offset)
		}
		if err != nil {
			report.Errors = append(report.Errors, err.Error())
			report.Summary.RemainingFrom = rowOffset
			report.Summary.BytesConsumed = rowOffset
			return
		}
		row := DataSchemaRow{Index: len(report.Rows), Offset: rowOffset, Fields: fields}
		row.PhaseID = intFromAny(fields["tick"])
		row.TimeS = intFromAny(fields["game_time"])
		report.Rows = append(report.Rows, row)
		report.Summary.BytesConsumed = offset
	}
	if !report.Summary.HasFooter {
		report.Warnings = append(report.Warnings, "sidecar has no end/footer row; decoded complete z-bouncing-ball samples only")
	}
}

func readTrigLongValue(data []byte, offset int) (map[string]any, int, error) {
	fields, next, err := readSchemaFields(data, offset, 0, []schemaField{
		{name: "tag", typ: "string"}, {name: "id", typ: "string"},
	})
	if err != nil {
		return nil, offset, err
	}
	kind, nextKind, err := readTypedValue(data, next, 2, "string", false)
	if err != nil {
		return nil, offset, err
	}
	fields["kind"] = kind.String
	if kind.String == "float" {
		value, afterValue, valueErr := readTypedValue(data, nextKind, 3, "float", false)
		if valueErr != nil {
			return nil, offset, valueErr
		}
		fields["value_float"] = scalarDataValue(value)
		time, afterTime, timeErr := readTypedValue(data, afterValue, 4, "int", false)
		if timeErr != nil {
			return nil, offset, timeErr
		}
		fields["game_time"] = scalarDataValue(time)
		return fields, afterTime, nil
	}
	fields["value_string"] = kind.String
	time, afterTime, timeErr := readTypedValue(data, nextKind, 3, "int", false)
	if timeErr != nil {
		return nil, offset, timeErr
	}
	fields["game_time"] = scalarDataValue(time)
	return fields, afterTime, nil
}

func decodePhaseRows(data []byte, report *DataSchemaDecodeReport, headerFields, rowFields, footerFields []schemaField) {
	offset := 0
	header, next, err := readSchemaFields(data, offset, 0, headerFields)
	if err != nil {
		report.Errors = append(report.Errors, err.Error())
		return
	}
	report.Header = header
	offset = next
	indexBase := len(headerFields)

	for offset < len(data) {
		tagValue, _, err := readTypedValue(data, offset, indexBase, "string", false)
		if err != nil {
			report.Errors = append(report.Errors, err.Error())
			report.Summary.RemainingFrom = offset
			return
		}
		switch tagValue.String {
		case "resource":
			if report.Schema != "a2k-custom-store" {
				report.Errors = append(report.Errors, fmt.Sprintf("unexpected row tag %q at offset %d", tagValue.String, offset))
				report.Summary.RemainingFrom = offset
				return
			}
			fields, next, err := readSchemaFields(data, offset, indexBase, a2kCustomStoreResourceFields)
			if err != nil {
				report.Errors = append(report.Errors, err.Error())
				report.Summary.RemainingFrom = offset
				return
			}
			for name, value := range fields {
				if name != "tag" {
					report.Header[name] = value
				}
			}
			offset = next
			indexBase += len(a2kCustomStoreResourceFields)
		case "phase", "purchase", "marker", "before", "result":
			if (tagValue.String == "marker" || tagValue.String == "before" || tagValue.String == "result") && report.Schema != "a2k-engine-harness" {
				report.Errors = append(report.Errors, fmt.Sprintf("unexpected row tag %q at offset %d", tagValue.String, offset))
				report.Summary.RemainingFrom = offset
				return
			}
			rowOffset := offset
			fields, next, err := readSchemaFields(data, offset, indexBase, rowFields)
			if err != nil {
				report.Errors = append(report.Errors, err.Error())
				report.Summary.RemainingFrom = offset
				return
			}
			row := DataSchemaRow{
				Index:  len(report.Rows),
				Offset: rowOffset,
				Fields: fields,
			}
			row.PhaseID = intFromAny(fields["phase_id"])
			row.TimeS = intFromAny(fields["xs_time"])
			if _, ok := fields["game_time"]; ok {
				row.TimeS = intFromAny(fields["game_time"])
			}
			report.Rows = append(report.Rows, row)
			offset = next
			indexBase += len(rowFields)
		case "end":
			footer, next, err := readSchemaFields(data, offset, indexBase, footerFields)
			if err != nil {
				report.Errors = append(report.Errors, err.Error())
				report.Summary.RemainingFrom = offset
				return
			}
			report.Footer = footer
			report.Summary.HasFooter = true
			offset = next
			indexBase += len(footerFields)
			if offset < len(data) {
				report.Errors = append(report.Errors, fmt.Sprintf("trailing bytes after footer at offset %d", offset))
				report.Summary.RemainingFrom = offset
			}
			return
		default:
			report.Errors = append(report.Errors, fmt.Sprintf("unexpected row tag %q at offset %d", tagValue.String, offset))
			report.Summary.RemainingFrom = offset
			return
		}
	}

	report.Warnings = append(report.Warnings, "sidecar has no end/footer row; decoded complete phase rows only")
	if report.Schema == "a2k-engine-harness" && len(report.Rows) > 0 {
		last := report.Rows[len(report.Rows)-1]
		if last.Fields["tag"] == "before" {
			report.Warnings = append(report.Warnings, fmt.Sprintf("harness ended after pre-operation marker for fact %q; treat it as a crash or early termination candidate", last.Fields["fact_id"]))
		}
	}
}

func buildRTV13RowFields() []schemaField {
	fields := []schemaField{
		{name: "tag", typ: "string"},
		{name: "phase_id", typ: "int"},
		{name: "case_player", typ: "int"},
		{name: "case_kind", typ: "int"},
		{name: "xs_time", typ: "int"},
		{name: "p1_resource_sum_carrier_food_attr0", typ: "float"},
		{name: "p1_wood_case_player_attr1", typ: "float"},
		{name: "p1_gold_case_kind_attr3", typ: "float"},
		{name: "p1_stone_row_index_attr2", typ: "float"},
	}
	for player := 1; player <= 8; player++ {
		prefix := fmt.Sprintf("p%d", player)
		fields = append(fields,
			schemaField{name: prefix + "_kills_attr20", typ: "float"},
			schemaField{name: prefix + "_razings_attr43", typ: "float"},
			schemaField{name: prefix + "_killed_by_others_attr154", typ: "float"},
			schemaField{name: prefix + "_razed_by_others_attr155", typ: "float"},
			schemaField{name: prefix + "_kill_value_attr170", typ: "float"},
			schemaField{name: prefix + "_raze_value_attr172", typ: "float"},
		)
	}
	fields = append(fields,
		schemaField{name: "p1_gaia_kills_attr300", typ: "float"},
		schemaField{name: "p1_player1_kills_attr301", typ: "float"},
		schemaField{name: "p1_player2_kills_attr302", typ: "float"},
		schemaField{name: "p1_player3_kills_attr303", typ: "float"},
		schemaField{name: "p1_player4_kills_attr304", typ: "float"},
		schemaField{name: "p1_player5_kills_attr305", typ: "float"},
		schemaField{name: "p1_player6_kills_attr306", typ: "float"},
		schemaField{name: "p1_player7_kills_attr307", typ: "float"},
		schemaField{name: "p1_player8_kills_attr308", typ: "float"},

		schemaField{name: "p1_gaia_razings_attr350", typ: "float"},
		schemaField{name: "p1_player1_razings_attr351", typ: "float"},
		schemaField{name: "p1_player2_razings_attr352", typ: "float"},
		schemaField{name: "p1_player3_razings_attr353", typ: "float"},
		schemaField{name: "p1_player4_razings_attr354", typ: "float"},
		schemaField{name: "p1_player5_razings_attr355", typ: "float"},
		schemaField{name: "p1_player6_razings_attr356", typ: "float"},
		schemaField{name: "p1_player7_razings_attr357", typ: "float"},
		schemaField{name: "p1_player8_razings_attr358", typ: "float"},
	)
	for player := 2; player <= 8; player++ {
		fields = append(fields, schemaField{name: fmt.Sprintf("p%d_kills_by_player1_attr326", player), typ: "float"})
	}
	for player := 2; player <= 8; player++ {
		fields = append(fields, schemaField{name: fmt.Sprintf("p%d_razings_by_player1_attr376", player), typ: "float"})
	}
	fields = append(fields,
		schemaField{name: "p1_militia_74_count", typ: "int"},
		schemaField{name: "p1_barracks_12_count", typ: "int"},
	)
	for player := 2; player <= 8; player++ {
		fields = append(fields,
			schemaField{name: fmt.Sprintf("p%d_militia_74_count", player), typ: "int"},
			schemaField{name: fmt.Sprintf("p%d_barracks_12_count", player), typ: "int"},
		)
	}
	return fields
}

func buildRTV14RowFields() []schemaField {
	fields := []schemaField{
		{name: "tag", typ: "string"},
		{name: "phase_id", typ: "int"},
		{name: "case_player", typ: "int"},
		{name: "case_unit", typ: "int"},
		{name: "spawn_count", typ: "int"},
		{name: "expected_p1_kills", typ: "int"},
		{name: "xs_time", typ: "int"},
		{name: "p1_resource_sum_carrier_food_attr0", typ: "float"},
		{name: "p1_wood_case_player_attr1", typ: "float"},
		{name: "p1_gold_case_unit_attr3", typ: "float"},
		{name: "p1_stone_expected_p1_kills_attr2", typ: "float"},
	}
	for player := 1; player <= 8; player++ {
		prefix := fmt.Sprintf("p%d", player)
		fields = append(fields,
			schemaField{name: prefix + "_kills_attr20", typ: "float"},
			schemaField{name: prefix + "_razings_attr43", typ: "float"},
			schemaField{name: prefix + "_killed_by_others_attr154", typ: "float"},
			schemaField{name: prefix + "_razed_by_others_attr155", typ: "float"},
			schemaField{name: prefix + "_kill_value_attr170", typ: "float"},
			schemaField{name: prefix + "_raze_value_attr172", typ: "float"},
		)
	}
	fields = append(fields,
		schemaField{name: "p1_gaia_kills_attr300", typ: "float"},
		schemaField{name: "p1_player1_kills_attr301", typ: "float"},
		schemaField{name: "p1_player2_kills_attr302", typ: "float"},
		schemaField{name: "p1_player3_kills_attr303", typ: "float"},
		schemaField{name: "p1_player4_kills_attr304", typ: "float"},
		schemaField{name: "p1_player5_kills_attr305", typ: "float"},
		schemaField{name: "p1_player6_kills_attr306", typ: "float"},
		schemaField{name: "p1_player7_kills_attr307", typ: "float"},
		schemaField{name: "p1_player8_kills_attr308", typ: "float"},

		schemaField{name: "p1_gaia_razings_attr350", typ: "float"},
		schemaField{name: "p1_player1_razings_attr351", typ: "float"},
		schemaField{name: "p1_player2_razings_attr352", typ: "float"},
		schemaField{name: "p1_player3_razings_attr353", typ: "float"},
		schemaField{name: "p1_player4_razings_attr354", typ: "float"},
		schemaField{name: "p1_player5_razings_attr355", typ: "float"},
		schemaField{name: "p1_player6_razings_attr356", typ: "float"},
		schemaField{name: "p1_player7_razings_attr357", typ: "float"},
		schemaField{name: "p1_player8_razings_attr358", typ: "float"},
	)
	for player := 2; player <= 8; player++ {
		fields = append(fields, schemaField{name: fmt.Sprintf("p%d_kills_by_player1_attr326", player), typ: "float"})
	}
	for player := 2; player <= 8; player++ {
		fields = append(fields, schemaField{name: fmt.Sprintf("p%d_razings_by_player1_attr376", player), typ: "float"})
	}
	fields = append(fields, schemaField{name: "p1_castle_82_count", typ: "int"})
	for player := 2; player <= 8; player++ {
		prefix := fmt.Sprintf("p%d", player)
		fields = append(fields,
			schemaField{name: prefix + "_militia_74_count", typ: "int"},
			schemaField{name: prefix + "_archer_4_count", typ: "int"},
			schemaField{name: prefix + "_spearman_93_count", typ: "int"},
			schemaField{name: prefix + "_villager_83_count", typ: "int"},
			schemaField{name: prefix + "_scout_448_count", typ: "int"},
			schemaField{name: prefix + "_knight_38_count", typ: "int"},
		)
	}
	return fields
}

func buildRTV141RowFields() []schemaField {
	fields := []schemaField{
		{name: "tag", typ: "string"},
		{name: "phase_id", typ: "int"},
		{name: "case_player", typ: "int"},
		{name: "case_unit", typ: "int"},
		{name: "spawn_count", typ: "int"},
		{name: "expected_p1_kills", typ: "int"},
		{name: "xs_time", typ: "int"},
		{name: "p1_resource_sum_carrier_food_attr0", typ: "float"},
		{name: "p1_wood_case_player_attr1", typ: "float"},
		{name: "p1_gold_case_unit_attr3", typ: "float"},
		{name: "p1_stone_expected_p1_kills_attr2", typ: "float"},
	}
	for player := 1; player <= 3; player++ {
		prefix := fmt.Sprintf("p%d", player)
		fields = append(fields,
			schemaField{name: prefix + "_kills_attr20", typ: "float"},
			schemaField{name: prefix + "_razings_attr43", typ: "float"},
			schemaField{name: prefix + "_killed_by_others_attr154", typ: "float"},
			schemaField{name: prefix + "_razed_by_others_attr155", typ: "float"},
			schemaField{name: prefix + "_kill_value_attr170", typ: "float"},
			schemaField{name: prefix + "_raze_value_attr172", typ: "float"},
		)
	}
	fields = append(fields,
		schemaField{name: "p1_gaia_kills_attr300", typ: "float"},
		schemaField{name: "p1_player1_kills_attr301", typ: "float"},
		schemaField{name: "p1_player2_kills_attr302", typ: "float"},
		schemaField{name: "p1_player3_kills_attr303", typ: "float"},
		schemaField{name: "p1_gaia_razings_attr350", typ: "float"},
		schemaField{name: "p1_player1_razings_attr351", typ: "float"},
		schemaField{name: "p1_player2_razings_attr352", typ: "float"},
		schemaField{name: "p1_player3_razings_attr353", typ: "float"},
		schemaField{name: "p2_kills_by_player1_attr326", typ: "float"},
		schemaField{name: "p3_kills_by_player1_attr326", typ: "float"},
		schemaField{name: "p2_razings_by_player1_attr376", typ: "float"},
		schemaField{name: "p3_razings_by_player1_attr376", typ: "float"},
		schemaField{name: "p1_castle_82_count", typ: "int"},
	)
	for player := 2; player <= 3; player++ {
		prefix := fmt.Sprintf("p%d", player)
		fields = append(fields,
			schemaField{name: prefix + "_militia_74_count", typ: "int"},
			schemaField{name: prefix + "_archer_4_count", typ: "int"},
			schemaField{name: prefix + "_spearman_93_count", typ: "int"},
			schemaField{name: prefix + "_villager_83_count", typ: "int"},
			schemaField{name: prefix + "_scout_448_count", typ: "int"},
			schemaField{name: prefix + "_knight_38_count", typ: "int"},
		)
	}
	return fields
}

func buildRTV15RowFields() []schemaField {
	fields := []schemaField{
		{name: "tag", typ: "string"},
		{name: "phase_id", typ: "int"},
		{name: "case_player", typ: "int"},
		{name: "case_unit", typ: "int"},
		{name: "spawn_count", typ: "int"},
		{name: "expected_p1_kills", typ: "int"},
		{name: "xs_time", typ: "int"},
		{name: "p1_resource_sum_carrier_food_attr0", typ: "float"},
		{name: "p1_wood_case_player_attr1", typ: "float"},
		{name: "p1_gold_case_unit_attr3", typ: "float"},
		{name: "p1_stone_expected_p1_kills_attr2", typ: "float"},
	}
	for player := 1; player <= 4; player++ {
		prefix := fmt.Sprintf("p%d", player)
		fields = append(fields,
			schemaField{name: prefix + "_kills_attr20", typ: "float"},
			schemaField{name: prefix + "_razings_attr43", typ: "float"},
			schemaField{name: prefix + "_killed_by_others_attr154", typ: "float"},
			schemaField{name: prefix + "_razed_by_others_attr155", typ: "float"},
			schemaField{name: prefix + "_kill_value_attr170", typ: "float"},
			schemaField{name: prefix + "_raze_value_attr172", typ: "float"},
		)
	}
	fields = append(fields,
		schemaField{name: "p1_gaia_kills_attr300", typ: "float"},
		schemaField{name: "p1_player1_kills_attr301", typ: "float"},
		schemaField{name: "p1_player2_kills_attr302", typ: "float"},
		schemaField{name: "p1_player3_kills_attr303", typ: "float"},
		schemaField{name: "p1_player4_kills_attr304", typ: "float"},
		schemaField{name: "p2_kills_by_player1_attr326", typ: "float"},
		schemaField{name: "p3_kills_by_player1_attr326", typ: "float"},
		schemaField{name: "p4_kills_by_player1_attr326", typ: "float"},
		schemaField{name: "p1_castle_82_count", typ: "int"},
	)
	for player := 2; player <= 4; player++ {
		prefix := fmt.Sprintf("p%d", player)
		fields = append(fields,
			schemaField{name: prefix + "_barracks_12_count", typ: "int"},
			schemaField{name: prefix + "_militia_74_count", typ: "int"},
			schemaField{name: prefix + "_archer_4_count", typ: "int"},
			schemaField{name: prefix + "_spearman_93_count", typ: "int"},
			schemaField{name: prefix + "_villager_83_count", typ: "int"},
			schemaField{name: prefix + "_scout_448_count", typ: "int"},
			schemaField{name: prefix + "_knight_38_count", typ: "int"},
		)
	}
	return fields
}

func buildRTV16RowFields() []schemaField {
	fields := []schemaField{
		{name: "tag", typ: "string"},
		{name: "phase_id", typ: "int"},
		{name: "case_player", typ: "int"},
		{name: "case_kind", typ: "int"},
		{name: "expected_p1_kills", typ: "int"},
		{name: "expected_p1_razes", typ: "int"},
		{name: "xs_time", typ: "int"},
		{name: "p1_resource_sum_carrier_food_attr0", typ: "float"},
		{name: "p1_wood_case_player_attr1", typ: "float"},
		{name: "p1_gold_case_kind_attr3", typ: "float"},
		{name: "p1_stone_expected_p1_kills_attr2", typ: "float"},
	}
	for player := 1; player <= 4; player++ {
		prefix := fmt.Sprintf("p%d", player)
		fields = append(fields,
			schemaField{name: prefix + "_current_age_attr6", typ: "float"},
			schemaField{name: prefix + "_population_attr11", typ: "float"},
			schemaField{name: prefix + "_discovery_attr13", typ: "float"},
			schemaField{name: prefix + "_exploration_attr22", typ: "float"},
			schemaField{name: prefix + "_kills_attr20", typ: "float"},
			schemaField{name: prefix + "_razings_attr43", typ: "float"},
			schemaField{name: prefix + "_value_killed_by_others_attr152", typ: "float"},
			schemaField{name: prefix + "_value_razed_by_others_attr153", typ: "float"},
			schemaField{name: prefix + "_killed_by_others_attr154", typ: "float"},
			schemaField{name: prefix + "_razed_by_others_attr155", typ: "float"},
			schemaField{name: prefix + "_value_current_units_attr164", typ: "float"},
			schemaField{name: prefix + "_value_current_buildings_attr165", typ: "float"},
			schemaField{name: prefix + "_food_total_attr166", typ: "float"},
			schemaField{name: prefix + "_wood_total_attr167", typ: "float"},
			schemaField{name: prefix + "_stone_total_attr168", typ: "float"},
			schemaField{name: prefix + "_gold_total_attr169", typ: "float"},
			schemaField{name: prefix + "_kill_value_attr170", typ: "float"},
			schemaField{name: prefix + "_raze_value_attr172", typ: "float"},
			schemaField{name: prefix + "_tribute_score_attr175", typ: "float"},
			schemaField{name: prefix + "_food_score_attr185", typ: "float"},
			schemaField{name: prefix + "_wood_score_attr186", typ: "float"},
			schemaField{name: prefix + "_stone_score_attr187", typ: "float"},
			schemaField{name: prefix + "_gold_score_attr188", typ: "float"},
			schemaField{name: prefix + "_map_reveal_attr203", typ: "float"},
			schemaField{name: prefix + "_unit_reveal_attr204", typ: "float"},
			schemaField{name: prefix + "_temporary_map_reveal_attr209", typ: "float"},
		)
	}
	fields = append(fields,
		schemaField{name: "p1_gaia_kills_attr300", typ: "float"},
		schemaField{name: "p1_player1_kills_attr301", typ: "float"},
		schemaField{name: "p1_player2_kills_attr302", typ: "float"},
		schemaField{name: "p1_player3_kills_attr303", typ: "float"},
		schemaField{name: "p1_player4_kills_attr304", typ: "float"},
		schemaField{name: "p1_gaia_razings_attr350", typ: "float"},
		schemaField{name: "p1_player1_razings_attr351", typ: "float"},
		schemaField{name: "p1_player2_razings_attr352", typ: "float"},
		schemaField{name: "p1_player3_razings_attr353", typ: "float"},
		schemaField{name: "p1_player4_razings_attr354", typ: "float"},
		schemaField{name: "p1_gaia_kill_value_attr400", typ: "float"},
		schemaField{name: "p1_player1_kill_value_attr401", typ: "float"},
		schemaField{name: "p1_player2_kill_value_attr402", typ: "float"},
		schemaField{name: "p1_player3_kill_value_attr403", typ: "float"},
		schemaField{name: "p1_player4_kill_value_attr404", typ: "float"},
		schemaField{name: "p1_gaia_raze_value_attr425", typ: "float"},
		schemaField{name: "p1_player1_raze_value_attr426", typ: "float"},
		schemaField{name: "p1_player2_raze_value_attr427", typ: "float"},
		schemaField{name: "p1_player3_raze_value_attr428", typ: "float"},
		schemaField{name: "p1_player4_raze_value_attr429", typ: "float"},
		schemaField{name: "p2_kills_by_player1_attr326", typ: "float"},
		schemaField{name: "p3_kills_by_player1_attr326", typ: "float"},
		schemaField{name: "p4_kills_by_player1_attr326", typ: "float"},
		schemaField{name: "p2_razings_by_player1_attr376", typ: "float"},
		schemaField{name: "p3_razings_by_player1_attr376", typ: "float"},
		schemaField{name: "p4_razings_by_player1_attr376", typ: "float"},
		schemaField{name: "p1_castle_82_count", typ: "int"},
		schemaField{name: "p1_cobra_748_count", typ: "int"},
	)
	for player := 2; player <= 4; player++ {
		prefix := fmt.Sprintf("p%d", player)
		fields = append(fields,
			schemaField{name: prefix + "_barracks_12_count", typ: "int"},
			schemaField{name: prefix + "_militia_74_count", typ: "int"},
			schemaField{name: prefix + "_archer_4_count", typ: "int"},
			schemaField{name: prefix + "_spearman_93_count", typ: "int"},
		)
	}
	fields = append(fields, schemaField{name: "gaia_militia_74_count", typ: "int"})
	return fields
}

func buildRTV17RowFields() []schemaField {
	fields := append([]schemaField{}, rtv16RowFields...)
	fields = append(fields,
		schemaField{name: "expected_p1_deaths", typ: "int"},
		schemaField{name: "expected_p2_kills", typ: "int"},
		schemaField{name: "p2_player1_kills_attr301", typ: "float"},
		schemaField{name: "p2_player1_razings_attr351", typ: "float"},
		schemaField{name: "p2_player1_kill_value_attr401", typ: "float"},
		schemaField{name: "p2_player1_raze_value_attr426", typ: "float"},
		schemaField{name: "p1_kills_by_player2_attr327", typ: "float"},
		schemaField{name: "p1_razings_by_player2_attr377", typ: "float"},
		schemaField{name: "p1_militia_74_count", typ: "int"},
		schemaField{name: "p1_archer_4_count", typ: "int"},
		schemaField{name: "p1_spearman_93_count", typ: "int"},
		schemaField{name: "p1_villager_83_count", typ: "int"},
		schemaField{name: "p1_scout_448_count", typ: "int"},
		schemaField{name: "p1_knight_38_count", typ: "int"},
		schemaField{name: "p2_castle_82_count", typ: "int"},
	)
	return fields
}

func readSchemaFields(data []byte, offset int, indexBase int, fields []schemaField) (map[string]any, int, error) {
	out := make(map[string]any, len(fields))
	for i, field := range fields {
		value, next, err := readTypedValue(data, offset, indexBase+i, field.typ, false)
		if err != nil {
			return nil, offset, err
		}
		out[field.name] = scalarDataValue(value)
		offset = next
	}
	return out, offset, nil
}

func scalarDataValue(value DataValue) any {
	switch value.Type {
	case "string":
		return value.String
	case "int":
		if value.Int != nil {
			return int(*value.Int)
		}
	case "uint":
		if value.UInt != nil {
			return *value.UInt
		}
	case "float":
		if value.Float != nil {
			return *value.Float
		}
	}
	return nil
}

func intFromAny(value any) int {
	switch v := value.(type) {
	case int:
		return v
	case int32:
		return int(v)
	case uint32:
		return int(v)
	case float32:
		return int(v)
	case float64:
		return int(v)
	default:
		return 0
	}
}
