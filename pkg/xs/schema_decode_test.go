package xs

import (
	"bytes"
	"encoding/binary"
	"math"
	"strings"
	"testing"
)

func TestDecodeBarracksWallV2Sidecar(t *testing.T) {
	var data bytes.Buffer
	writeString := func(s string) {
		_ = binary.Write(&data, binary.LittleEndian, uint32(len(s)))
		_, _ = data.WriteString(s)
	}
	writeInt := func(v int32) { _ = binary.Write(&data, binary.LittleEndian, v) }
	writeFloat := func(v float32) { _ = binary.Write(&data, binary.LittleEndian, math.Float32bits(v)) }
	writeString("A2K_BARRACKS_WALL_V2")
	writeInt(1)
	writeString("spawned-control")
	writeString("unit")
	writeInt(26)
	writeInt(1)
	writeFloat(10)
	writeFloat(10)
	writeFloat(1.5)
	writeFloat(1.5)
	writeFloat(1.5)
	writeFloat(1.5)
	writeString("end")
	writeInt(424242)

	report := DecodeDataBytesSchema(data.Bytes(), DataSchemaDecodeOptions{Schema: "barracks-wall-v2"})
	if !report.OK || len(report.Rows) != 1 || !report.Summary.HasFooter {
		t.Fatalf("report=%+v errors=%v", report, report.Errors)
	}
	if !report.Summary.WithinByteCap || report.Summary.ByteCap != MaxSidecarBytes || report.Summary.TagCounts["unit"] != 1 || report.Summary.TagCounts["end"] != 1 {
		t.Fatalf("sidecar coverage=%+v", report.Summary)
	}
	row := report.Rows[0].Fields
	if intFromAny(row["unit_id"]) != 26 || intFromAny(row["owner"]) != 1 || row["x"] != float32(10) || row["attr_200"] != float32(1.5) {
		t.Fatalf("decoded row=%v", row)
	}
}

func TestSchemaDecodeReportsOverCapWithoutLosingRows(t *testing.T) {
	data := make([]byte, MaxSidecarBytes+1)
	report := DecodeDataBytesSchema(data, DataSchemaDecodeOptions{Schema: "unknown"})
	if report.Summary.WithinByteCap || report.Summary.ByteCap != MaxSidecarBytes {
		t.Fatalf("cap summary=%+v", report.Summary)
	}
	if len(report.Warnings) == 0 || !strings.Contains(report.Warnings[0], "exceeds engine byte cap") {
		t.Fatalf("warnings=%v", report.Warnings)
	}
}

func TestDecodeSnakeV0Sidecar(t *testing.T) {
	var data bytes.Buffer
	writeString := func(s string) {
		_ = binary.Write(&data, binary.LittleEndian, uint32(len(s)))
		_, _ = data.WriteString(s)
	}
	writeInt := func(v int32) { _ = binary.Write(&data, binary.LittleEndian, v) }
	writeFloat := func(v float32) { _ = binary.Write(&data, binary.LittleEndian, math.Float32bits(v)) }
	writeString("A2K_SNAKE_V0")
	writeInt(1)
	writeString("native-flame-multiplayer")
	writeInt(8)
	writeInt(24)
	writeInt(16)
	writeString("event")
	writeInt(1)
	writeInt(59)
	writeInt(12)
	writeInt(200)
	writeString("selftest")
	writeInt(1)
	writeInt(20)
	writeFloat(90)
	writeFloat(83.5)
	writeInt(333)
	writeString("end")
	writeInt(1)
	writeInt(424242)
	writeString("death")
	writeInt(2)
	writeInt(4)
	writeInt(30)
	writeInt(500)

	report := DecodeDataBytesSchema(data.Bytes(), DataSchemaDecodeOptions{Schema: "snake-v0"})
	if !report.OK || len(report.Rows) != 3 || !report.Summary.HasFooter || report.Summary.BytesConsumed != len(data.Bytes()) {
		t.Fatalf("report=%+v errors=%v", report, report.Errors)
	}
	heroHP, ok := report.Rows[1].Fields["hero_hp"].(float32)
	if intFromAny(report.Header["players"]) != 8 || intFromAny(report.Rows[0].Fields["value"]) != 59 || !ok || heroHP != 90 {
		t.Fatalf("decoded snake=%+v", report)
	}
}

func TestDecodeZBouncingBallSidecar(t *testing.T) {
	var data bytes.Buffer
	writeString := func(s string) {
		_ = binary.Write(&data, binary.LittleEndian, uint32(len(s)))
		_, _ = data.WriteString(s)
	}
	writeInt := func(v int32) { _ = binary.Write(&data, binary.LittleEndian, v) }
	writeFloat := func(v float32) { _ = binary.Write(&data, binary.LittleEndian, math.Float32bits(v)) }
	writeString("A2K_Z_BOUNCING_BALL")
	writeInt(1)
	writeString("move-z-and-create-z")
	writeInt(20)
	writeInt(5)
	writeString("sample")
	writeInt(10)
	writeInt(0)
	writeInt(4242)
	writeFloat(1.0)
	writeFloat(0.0)
	writeFloat(0.0)
	writeFloat(0.0)
	writeFloat(1.0)
	writeFloat(2.0)
	writeFloat(3.0)
	writeFloat(4.0)
	writeString("end")
	writeInt(1)
	writeInt(424242)

	report := DecodeDataBytesSchema(data.Bytes(), DataSchemaDecodeOptions{Schema: "a2k_z_bouncing_ball"})
	if !report.OK || len(report.Rows) != 1 || !report.Summary.HasFooter || report.Summary.BytesConsumed != len(data.Bytes()) {
		t.Fatalf("report=%+v errors=%v", report, report.Errors)
	}
	if report.Summary.TagCounts["sample"] != 1 || report.Summary.TagCounts["end"] != 1 {
		t.Fatalf("sidecar coverage=%+v", report.Summary)
	}
	row := report.Rows[0].Fields
	if intFromAny(row["tick"]) != 10 || row["expected_z"] != float32(1) || row["ruler_z4"] != float32(4) {
		t.Fatalf("decoded row=%v", row)
	}
}

func TestDecodeA2KCustomStoreSidecar(t *testing.T) {
	var data bytes.Buffer
	writeString := func(s string) {
		_ = binary.Write(&data, binary.LittleEndian, uint32(len(s)))
		_, _ = data.WriteString(s)
	}
	writeInt := func(v int32) { _ = binary.Write(&data, binary.LittleEndian, v) }
	writeFloat := func(v float32) { _ = binary.Write(&data, binary.LittleEndian, math.Float32bits(v)) }

	writeString("A2K_CUSTOM_STORE")
	writeInt(1)
	writeString("per_tower_store_440")
	writeInt(440)
	for i := 0; i < 8; i++ {
		writeInt(55)
	}
	writeInt(440)
	writeString("resource")
	writeInt(0)
	writeInt(100000)
	writeInt(100000)
	writeInt(100000)
	writeString("purchase")
	writeInt(17)
	writeInt(440)
	writeInt(440)
	writeInt(3)
	writeInt(74)
	writeInt(990123)
	writeFloat(42.5)
	writeFloat(43.75)
	writeInt(920207)
	writeInt(920208)
	writeInt(0)
	writeInt(1)
	writeInt(0)
	writeInt(0)
	writeString("end")
	writeInt(1)
	writeInt(440)
	writeInt(424242)

	report := DecodeDataBytesSchema(data.Bytes(), DataSchemaDecodeOptions{Schema: "towerstore"})
	if !report.OK {
		t.Fatalf("towerstore decode failed: %v", report.Errors)
	}
	if report.Schema != "a2k-custom-store" {
		t.Fatalf("schema=%q", report.Schema)
	}
	if len(report.Rows) != 1 || !report.Summary.HasFooter {
		t.Fatalf("summary=%+v", report.Summary)
	}
	row := report.Rows[0]
	if row.TimeS != 17 || row.Fields["resolved_object_id"] != 920207 || row.Fields["token_x"] != float32(42.5) {
		t.Fatalf("row=%+v", row)
	}
	if report.Footer["integrity"] != 424242 {
		t.Fatalf("footer=%v", report.Footer)
	}
	if report.Header["p1_food"] != 100000 || report.Header["p1_wood"] != 100000 || report.Header["p1_gold"] != 100000 {
		t.Fatalf("resource probe=%v", report.Header)
	}
}

func TestDecodeLongV2RateRow(t *testing.T) {
	var data bytes.Buffer
	writeString := func(s string) {
		_ = binary.Write(&data, binary.LittleEndian, uint32(len(s)))
		_, _ = data.WriteString(s)
	}
	writeInt := func(v int32) { _ = binary.Write(&data, binary.LittleEndian, v) }
	writeString("A2K_ATTR_TECH_TRIG_V2")
	writeInt(1)
	writeString("attributes-local-tech-trig-recheck")
	writeInt(196)
	writeString("rate")
	writeInt(12)
	writeInt(30)
	writeInt(360)
	writeInt(12000)
	writeString("end")
	writeInt(196)
	writeInt(424242)

	report := DecodeDataBytesSchema(data.Bytes(), DataSchemaDecodeOptions{Schema: "long-v2"})
	if !report.OK {
		t.Fatalf("long-v2 rate decode failed: %v", report.Errors)
	}
	if len(report.Rows) != 1 || intFromAny(report.Rows[0].Fields["calls_this_second"]) != 30 || intFromAny(report.Rows[0].Fields["world_time_ms"]) != 12000 {
		t.Fatalf("rate row=%+v", report.Rows)
	}
	if !report.Summary.HasFooter {
		t.Fatal("rate fixture lost footer")
	}
}

func TestDecodeRTV12ExecutionerPartialSidecar(t *testing.T) {
	data := []byte{}
	appendString := func(s string) {
		var lenBuf [4]byte
		binary.LittleEndian.PutUint32(lenBuf[:], uint32(len(s)))
		data = append(data, lenBuf[:]...)
		data = append(data, []byte(s)...)
	}
	appendInt := func(v int) {
		var buf [4]byte
		binary.LittleEndian.PutUint32(buf[:], uint32(v))
		data = append(data, buf[:]...)
	}
	appendFloat := func(v float32) {
		var buf [4]byte
		binary.LittleEndian.PutUint32(buf[:], math.Float32bits(v))
		data = append(data, buf[:]...)
	}

	appendString("A2K_RTV12_EXECUTIONER_LEDGER")
	appendInt(12)
	appendString("manual_combat_attr_matrix_v12")
	appendString("phase")
	appendInt(20)
	appendInt(20)
	appendInt(1)
	appendInt(1)
	appendInt(20)
	for i := 0; i < 38; i++ {
		appendFloat(float32(i))
	}
	for i := 0; i < 8; i++ {
		appendInt(i)
	}

	report := DecodeDataBytesSchema(data, DataSchemaDecodeOptions{Schema: "rtv12"})
	if !report.OK {
		t.Fatalf("DecodeDataBytesSchema OK=false errors=%v", report.Errors)
	}
	if report.Summary.HasFooter {
		t.Fatalf("partial sidecar unexpectedly has footer")
	}
	if len(report.Rows) != 1 {
		t.Fatalf("rows=%d, want 1", len(report.Rows))
	}
	if got := report.Rows[0].PhaseID; got != 20 {
		t.Fatalf("phase=%d, want 20", got)
	}
	if got := report.Rows[0].Fields["p1_kills_attr20"]; got != float32(4) {
		t.Fatalf("p1_kills_attr20=%v, want 4", got)
	}
	if len(report.Warnings) == 0 {
		t.Fatalf("expected no-footer warning")
	}
}

func TestDecodeRTV13DirectedAttributionPartialSidecar(t *testing.T) {
	data := []byte{}
	appendString := func(s string) {
		var lenBuf [4]byte
		binary.LittleEndian.PutUint32(lenBuf[:], uint32(len(s)))
		data = append(data, lenBuf[:]...)
		data = append(data, []byte(s)...)
	}
	appendInt := func(v int) {
		var buf [4]byte
		binary.LittleEndian.PutUint32(buf[:], uint32(v))
		data = append(data, buf[:]...)
	}
	appendFloat := func(v float32) {
		var buf [4]byte
		binary.LittleEndian.PutUint32(buf[:], math.Float32bits(v))
		data = append(data, buf[:]...)
	}

	appendString("A2K_RTV13_DIRECTED_ATTRIBUTION")
	appendInt(13)
	appendString("directed_kill_raze_matrix_v13")
	appendString("phase")
	appendInt(101)
	appendInt(4)
	appendInt(1)
	appendInt(205)
	for i := 0; i < 4; i++ {
		appendFloat(float32(884101 + i))
	}
	for i := 0; i < 48; i++ {
		appendFloat(float32(i))
	}
	for i := 0; i < 18; i++ {
		appendFloat(float32(100 + i))
	}
	for i := 0; i < 14; i++ {
		appendFloat(float32(200 + i))
	}
	for i := 0; i < 16; i++ {
		appendInt(300 + i)
	}

	report := DecodeDataBytesSchema(data, DataSchemaDecodeOptions{Schema: "rtv13"})
	if !report.OK {
		t.Fatalf("DecodeDataBytesSchema OK=false errors=%v", report.Errors)
	}
	if len(report.Rows) != 1 {
		t.Fatalf("rows=%d, want 1", len(report.Rows))
	}
	row := report.Rows[0]
	if got := row.PhaseID; got != 101 {
		t.Fatalf("phase=%d, want 101", got)
	}
	if got := row.TimeS; got != 205 {
		t.Fatalf("time=%d, want 205", got)
	}
	if got := row.Fields["case_player"]; got != 4 {
		t.Fatalf("case_player=%v, want 4", got)
	}
	if got := row.Fields["p1_kills_attr20"]; got != float32(0) {
		t.Fatalf("p1_kills_attr20=%v, want 0", got)
	}
	if got := row.Fields["p1_player8_razings_attr358"]; got != float32(117) {
		t.Fatalf("p1_player8_razings_attr358=%v, want 117", got)
	}
	if got := row.Fields["p8_razings_by_player1_attr376"]; got != float32(213) {
		t.Fatalf("p8_razings_by_player1_attr376=%v, want 213", got)
	}
	if got := row.Fields["p8_barracks_12_count"]; got != 315 {
		t.Fatalf("p8_barracks_12_count=%v, want 315", got)
	}
	if len(report.Warnings) == 0 {
		t.Fatalf("expected no-footer warning")
	}
}

func TestDecodeRTV14CastleKillCalibrationPartialSidecar(t *testing.T) {
	data := []byte{}
	appendString := func(s string) {
		var lenBuf [4]byte
		binary.LittleEndian.PutUint32(lenBuf[:], uint32(len(s)))
		data = append(data, lenBuf[:]...)
		data = append(data, []byte(s)...)
	}
	appendInt := func(v int) {
		var buf [4]byte
		binary.LittleEndian.PutUint32(buf[:], uint32(v))
		data = append(data, buf[:]...)
	}
	appendFloat := func(v float32) {
		var buf [4]byte
		binary.LittleEndian.PutUint32(buf[:], math.Float32bits(v))
		data = append(data, buf[:]...)
	}

	appendString("A2K_RTV14_CASTLE_KILL_CALIBRATION")
	appendInt(14)
	appendString("castle_combat_death_matrix_v14")
	appendString("phase")
	appendInt(73)
	appendInt(4)
	appendInt(93)
	appendInt(2)
	appendInt(7)
	appendInt(85)
	for i := 0; i < 4; i++ {
		appendFloat(float32(885073 + i))
	}
	for i := 0; i < 48; i++ {
		appendFloat(float32(i))
	}
	for i := 0; i < 18; i++ {
		appendFloat(float32(100 + i))
	}
	for i := 0; i < 14; i++ {
		appendFloat(float32(200 + i))
	}
	appendInt(1)
	for i := 0; i < 42; i++ {
		appendInt(300 + i)
	}

	report := DecodeDataBytesSchema(data, DataSchemaDecodeOptions{Schema: "rtv14"})
	if !report.OK {
		t.Fatalf("DecodeDataBytesSchema OK=false errors=%v", report.Errors)
	}
	if len(report.Rows) != 1 {
		t.Fatalf("rows=%d, want 1", len(report.Rows))
	}
	row := report.Rows[0]
	if got := row.PhaseID; got != 73 {
		t.Fatalf("phase=%d, want 73", got)
	}
	if got := row.TimeS; got != 85 {
		t.Fatalf("time=%d, want 85", got)
	}
	if got := row.Fields["case_unit"]; got != 93 {
		t.Fatalf("case_unit=%v, want 93", got)
	}
	if got := row.Fields["expected_p1_kills"]; got != 7 {
		t.Fatalf("expected_p1_kills=%v, want 7", got)
	}
	if got := row.Fields["p1_player8_razings_attr358"]; got != float32(117) {
		t.Fatalf("p1_player8_razings_attr358=%v, want 117", got)
	}
	if got := row.Fields["p8_razings_by_player1_attr376"]; got != float32(213) {
		t.Fatalf("p8_razings_by_player1_attr376=%v, want 213", got)
	}
	if got := row.Fields["p1_castle_82_count"]; got != 1 {
		t.Fatalf("p1_castle_82_count=%v, want 1", got)
	}
	if got := row.Fields["p8_knight_38_count"]; got != 341 {
		t.Fatalf("p8_knight_38_count=%v, want 341", got)
	}
	if len(report.Warnings) == 0 {
		t.Fatalf("expected no-footer warning")
	}
}

func TestDecodeRTV141CastleKillCalibrationPartialSidecar(t *testing.T) {
	data := []byte{}
	appendString := func(s string) {
		var lenBuf [4]byte
		binary.LittleEndian.PutUint32(lenBuf[:], uint32(len(s)))
		data = append(data, lenBuf[:]...)
		data = append(data, []byte(s)...)
	}
	appendInt := func(v int) {
		var buf [4]byte
		binary.LittleEndian.PutUint32(buf[:], uint32(v))
		data = append(data, buf[:]...)
	}
	appendFloat := func(v float32) {
		var buf [4]byte
		binary.LittleEndian.PutUint32(buf[:], math.Float32bits(v))
		data = append(data, buf[:]...)
	}

	appendString("A2K_RTV141_CASTLE_KILL_CALIBRATION")
	appendInt(141)
	appendString("castle_combat_death_matrix_v141")
	appendString("phase")
	appendInt(93)
	appendInt(2)
	appendInt(83)
	appendInt(1)
	appendInt(8)
	appendInt(105)
	for i := 0; i < 4; i++ {
		appendFloat(float32(885193 + i))
	}
	for i := 0; i < 18; i++ {
		appendFloat(float32(i))
	}
	for i := 0; i < 12; i++ {
		appendFloat(float32(100 + i))
	}
	appendInt(1)
	for i := 0; i < 12; i++ {
		appendInt(300 + i)
	}

	report := DecodeDataBytesSchema(data, DataSchemaDecodeOptions{Schema: "rtv142"})
	if !report.OK {
		t.Fatalf("DecodeDataBytesSchema OK=false errors=%v", report.Errors)
	}
	if report.Schema != "rtv141-castle-kill-calibration" {
		t.Fatalf("schema=%q, want rtv141-castle-kill-calibration", report.Schema)
	}
	if len(report.Rows) != 1 {
		t.Fatalf("rows=%d, want 1", len(report.Rows))
	}
	row := report.Rows[0]
	if got := row.PhaseID; got != 93 {
		t.Fatalf("phase=%d, want 93", got)
	}
	if got := row.Fields["case_unit"]; got != 83 {
		t.Fatalf("case_unit=%v, want 83", got)
	}
	if got := row.Fields["p1_player3_razings_attr353"]; got != float32(107) {
		t.Fatalf("p1_player3_razings_attr353=%v, want 107", got)
	}
	if got := row.Fields["p3_razings_by_player1_attr376"]; got != float32(111) {
		t.Fatalf("p3_razings_by_player1_attr376=%v, want 111", got)
	}
	if got := row.Fields["p1_castle_82_count"]; got != 1 {
		t.Fatalf("p1_castle_82_count=%v, want 1", got)
	}
	if got := row.Fields["p3_knight_38_count"]; got != 311 {
		t.Fatalf("p3_knight_38_count=%v, want 311", got)
	}
}

func TestDecodeRTV15CastleKillGroundTruthPartialSidecar(t *testing.T) {
	data := []byte{}
	appendString := func(s string) {
		var lenBuf [4]byte
		binary.LittleEndian.PutUint32(lenBuf[:], uint32(len(s)))
		data = append(data, lenBuf[:]...)
		data = append(data, []byte(s)...)
	}
	appendInt := func(v int) {
		var buf [4]byte
		binary.LittleEndian.PutUint32(buf[:], uint32(v))
		data = append(data, buf[:]...)
	}
	appendFloat := func(v float32) {
		var buf [4]byte
		binary.LittleEndian.PutUint32(buf[:], math.Float32bits(v))
		data = append(data, buf[:]...)
	}

	appendString("A2K_RTV15_CASTLE_KILL_GROUND_TRUTH")
	appendInt(15)
	appendString("autonomous_castle_kill_matrix_v15")
	appendString("phase")
	appendInt(133)
	appendInt(234)
	appendInt(74)
	appendInt(3)
	appendInt(12)
	appendInt(150)
	for i := 0; i < 4; i++ {
		appendFloat(float32(886133 + i))
	}
	for i := 0; i < 24; i++ {
		appendFloat(float32(i))
	}
	for i := 0; i < 5; i++ {
		appendFloat(float32(100 + i))
	}
	for i := 0; i < 3; i++ {
		appendFloat(float32(200 + i))
	}
	appendInt(1)
	for i := 0; i < 21; i++ {
		appendInt(300 + i)
	}

	report := DecodeDataBytesSchema(data, DataSchemaDecodeOptions{Schema: "v15"})
	if !report.OK {
		t.Fatalf("DecodeDataBytesSchema OK=false errors=%v", report.Errors)
	}
	if report.Schema != "rtv15-castle-kill-ground-truth" {
		t.Fatalf("schema=%q, want rtv15-castle-kill-ground-truth", report.Schema)
	}
	if len(report.Rows) != 1 {
		t.Fatalf("rows=%d, want 1", len(report.Rows))
	}
	row := report.Rows[0]
	if got := row.PhaseID; got != 133 {
		t.Fatalf("phase=%d, want 133", got)
	}
	if got := row.TimeS; got != 150 {
		t.Fatalf("time=%d, want 150", got)
	}
	if got := row.Fields["case_player"]; got != 234 {
		t.Fatalf("case_player=%v, want 234", got)
	}
	if got := row.Fields["case_unit"]; got != 74 {
		t.Fatalf("case_unit=%v, want 74", got)
	}
	if got := row.Fields["spawn_count"]; got != 3 {
		t.Fatalf("spawn_count=%v, want 3", got)
	}
	if got := row.Fields["expected_p1_kills"]; got != 12 {
		t.Fatalf("expected_p1_kills=%v, want 12", got)
	}
	if got := row.Fields["p4_raze_value_attr172"]; got != float32(23) {
		t.Fatalf("p4_raze_value_attr172=%v, want 23", got)
	}
	if got := row.Fields["p1_player4_kills_attr304"]; got != float32(104) {
		t.Fatalf("p1_player4_kills_attr304=%v, want 104", got)
	}
	if got := row.Fields["p4_kills_by_player1_attr326"]; got != float32(202) {
		t.Fatalf("p4_kills_by_player1_attr326=%v, want 202", got)
	}
	if got := row.Fields["p1_castle_82_count"]; got != 1 {
		t.Fatalf("p1_castle_82_count=%v, want 1", got)
	}
	if got := row.Fields["p4_knight_38_count"]; got != 320 {
		t.Fatalf("p4_knight_38_count=%v, want 320", got)
	}
}

func TestDecodeRTV16SemanticPromotionPartialSidecar(t *testing.T) {
	data := []byte{}
	appendString := func(s string) {
		var lenBuf [4]byte
		binary.LittleEndian.PutUint32(lenBuf[:], uint32(len(s)))
		data = append(data, lenBuf[:]...)
		data = append(data, []byte(s)...)
	}
	appendInt := func(v int) {
		var buf [4]byte
		binary.LittleEndian.PutUint32(buf[:], uint32(v))
		data = append(data, buf[:]...)
	}
	appendFloat := func(v float32) {
		var buf [4]byte
		binary.LittleEndian.PutUint32(buf[:], math.Float32bits(v))
		data = append(data, buf[:]...)
	}

	appendString("A2K_RTV16_SEMANTIC_PROMOTION")
	appendInt(16)
	appendString("semantic_promotion_kill_raze_score_fog_v16")
	appendString("phase")
	appendInt(900)
	appendInt(9)
	appendInt(9)
	appendInt(4)
	appendInt(1)
	appendInt(245)
	for i := 0; i < 4; i++ {
		appendFloat(float32(887900 + i))
	}
	for i := 0; i < 104; i++ {
		appendFloat(float32(i))
	}
	for i := 0; i < 20; i++ {
		appendFloat(float32(200 + i))
	}
	for i := 0; i < 6; i++ {
		appendFloat(float32(300 + i))
	}
	appendInt(1)
	appendInt(1)
	for _, v := range []int{10, 11, 12, 13, 20, 21, 22, 23, 30, 31, 32, 33, 40} {
		appendInt(v)
	}

	report := DecodeDataBytesSchema(data, DataSchemaDecodeOptions{Schema: "v16"})
	if !report.OK {
		t.Fatalf("DecodeDataBytesSchema OK=false errors=%v", report.Errors)
	}
	if report.Schema != "rtv16-semantic-promotion" {
		t.Fatalf("schema=%q, want rtv16-semantic-promotion", report.Schema)
	}
	if len(report.Rows) != 1 {
		t.Fatalf("rows=%d, want 1", len(report.Rows))
	}
	row := report.Rows[0]
	if got := row.PhaseID; got != 900 {
		t.Fatalf("phase=%d, want 900", got)
	}
	if got := row.TimeS; got != 245 {
		t.Fatalf("time=%d, want 245", got)
	}
	if got := row.Fields["expected_p1_kills"]; got != 4 {
		t.Fatalf("expected_p1_kills=%v, want 4", got)
	}
	if got := row.Fields["expected_p1_razes"]; got != 1 {
		t.Fatalf("expected_p1_razes=%v, want 1", got)
	}
	if got := row.Fields["p1_kills_attr20"]; got != float32(4) {
		t.Fatalf("p1_kills_attr20=%v, want 4", got)
	}
	if got := row.Fields["p4_gold_score_attr188"]; got != float32(100) {
		t.Fatalf("p4_gold_score_attr188=%v, want 100", got)
	}
	if got := row.Fields["p1_player4_razings_attr354"]; got != float32(209) {
		t.Fatalf("p1_player4_razings_attr354=%v, want 209", got)
	}
	if got := row.Fields["p1_player3_kill_value_attr403"]; got != float32(213) {
		t.Fatalf("p1_player3_kill_value_attr403=%v, want 213", got)
	}
	if got := row.Fields["p4_razings_by_player1_attr376"]; got != float32(305) {
		t.Fatalf("p4_razings_by_player1_attr376=%v, want 305", got)
	}
	if got := row.Fields["p1_castle_82_count"]; got != 1 {
		t.Fatalf("p1_castle_82_count=%v, want 1", got)
	}
	if got := row.Fields["p4_spearman_93_count"]; got != 33 {
		t.Fatalf("p4_spearman_93_count=%v, want 33", got)
	}
	if got := row.Fields["gaia_militia_74_count"]; got != 40 {
		t.Fatalf("gaia_militia_74_count=%v, want 40", got)
	}
}

func TestDecodeRTV17PackedTestPartialSidecar(t *testing.T) {
	data := []byte{}
	appendString := func(s string) {
		var lenBuf [4]byte
		binary.LittleEndian.PutUint32(lenBuf[:], uint32(len(s)))
		data = append(data, lenBuf[:]...)
		data = append(data, []byte(s)...)
	}
	appendInt := func(v int) {
		var buf [4]byte
		binary.LittleEndian.PutUint32(buf[:], uint32(v))
		data = append(data, buf[:]...)
	}
	appendFloat := func(v float32) {
		var buf [4]byte
		binary.LittleEndian.PutUint32(buf[:], math.Float32bits(v))
		data = append(data, buf[:]...)
	}

	appendString("A2K_RTV17_PACKED_TEST")
	appendInt(17)
	appendString("packed_death_kill_raze_resource_visibility_v17")
	appendString("phase")
	appendInt(900)
	appendInt(9)
	appendInt(900)
	appendInt(1)
	appendInt(1)
	appendInt(245)
	for i := 0; i < 4; i++ {
		appendFloat(float32(888900 + i))
	}
	for i := 0; i < 104; i++ {
		appendFloat(float32(i))
	}
	for i := 0; i < 20; i++ {
		appendFloat(float32(200 + i))
	}
	for i := 0; i < 6; i++ {
		appendFloat(float32(300 + i))
	}
	appendInt(1)
	appendInt(1)
	for _, v := range []int{10, 11, 12, 13, 20, 21, 22, 23, 30, 31, 32, 33, 40} {
		appendInt(v)
	}
	appendInt(4)
	appendInt(4)
	for i := 0; i < 6; i++ {
		appendFloat(float32(500 + i))
	}
	for _, v := range []int{70, 71, 72, 73, 74, 75, 82} {
		appendInt(v)
	}

	report := DecodeDataBytesSchema(data, DataSchemaDecodeOptions{Schema: "v17"})
	if !report.OK {
		t.Fatalf("DecodeDataBytesSchema OK=false errors=%v", report.Errors)
	}
	if report.Schema != "rtv17-packed-test" {
		t.Fatalf("schema=%q, want rtv17-packed-test", report.Schema)
	}
	if len(report.Rows) != 1 {
		t.Fatalf("rows=%d, want 1", len(report.Rows))
	}
	row := report.Rows[0]
	if got := row.Fields["expected_p1_deaths"]; got != 4 {
		t.Fatalf("expected_p1_deaths=%v, want 4", got)
	}
	if got := row.Fields["p2_player1_kills_attr301"]; got != float32(500) {
		t.Fatalf("p2_player1_kills_attr301=%v, want 500", got)
	}
	if got := row.Fields["p1_kills_by_player2_attr327"]; got != float32(504) {
		t.Fatalf("p1_kills_by_player2_attr327=%v, want 504", got)
	}
	if got := row.Fields["p1_knight_38_count"]; got != 75 {
		t.Fatalf("p1_knight_38_count=%v, want 75", got)
	}
	if got := row.Fields["p2_castle_82_count"]; got != 82 {
		t.Fatalf("p2_castle_82_count=%v, want 82", got)
	}
}

func TestDecodeA2KSEM2DATCommandSemanticsSidecar(t *testing.T) {
	data := []byte{}
	appendString := func(s string) {
		var lenBuf [4]byte
		binary.LittleEndian.PutUint32(lenBuf[:], uint32(len(s)))
		data = append(data, lenBuf[:]...)
		data = append(data, []byte(s)...)
	}
	appendInt := func(v int) {
		var buf [4]byte
		binary.LittleEndian.PutUint32(buf[:], uint32(v))
		data = append(data, buf[:]...)
	}
	appendFloat := func(v float32) {
		var buf [4]byte
		binary.LittleEndian.PutUint32(buf[:], math.Float32bits(v))
		data = append(data, buf[:]...)
	}

	appendString("A2K_DAT_COMMAND_SEMANTICS")
	appendInt(2)
	appendString("run1_resource_object_count_contract")
	appendString("phase")
	appendInt(15)
	appendInt(1)
	appendInt(15)
	for _, v := range []float32{1111, 0, 0, 222, 2, 3} {
		appendFloat(v)
	}
	for _, v := range []int{1, 0, 1, 2, 1, 1, 1} {
		appendInt(v)
	}
	appendString("end")
	appendInt(1)

	report := DecodeDataBytesSchema(data, DataSchemaDecodeOptions{Schema: "datsem2"})
	if !report.OK {
		t.Fatalf("DecodeDataBytesSchema OK=false errors=%v", report.Errors)
	}
	if report.Schema != "a2ksem2-dat-command-semantics" {
		t.Fatalf("schema=%q, want a2ksem2-dat-command-semantics", report.Schema)
	}
	if len(report.Rows) != 1 {
		t.Fatalf("rows=%d, want 1", len(report.Rows))
	}
	row := report.Rows[0]
	if row.PhaseID != 15 || row.TimeS != 15 {
		t.Fatalf("row phase/time = %d/%d, want 15/15", row.PhaseID, row.TimeS)
	}
	if got := row.Fields["p1_food_attr0"]; got != float32(1111) {
		t.Fatalf("p1_food_attr0=%v, want 1111", got)
	}
	if got := row.Fields["p1_scout_448_count"]; got != 2 {
		t.Fatalf("p1_scout_448_count=%v, want 2", got)
	}
	if !report.Summary.HasFooter {
		t.Fatalf("missing footer")
	}
	if got := report.Footer["rows_written"]; got != 1 {
		t.Fatalf("rows_written=%v, want 1", got)
	}
}
