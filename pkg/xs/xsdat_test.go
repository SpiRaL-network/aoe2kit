package xs

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"math"
	"testing"
)

func TestDataValueJSONEncodesNonFiniteFloats(t *testing.T) {
	nan := float32(math.NaN())
	positiveInf := float32(math.Inf(1))
	data, err := json.Marshal(DataValue{Index: 1, Type: "float", Float: &nan, FloatCandidate: &positiveInf})
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if !bytes.Contains(data, []byte(`"float":"NaN"`)) || !bytes.Contains(data, []byte(`"float_candidate":"+Inf"`)) {
		t.Fatalf("non-finite values were not made JSON-safe: %s", text)
	}
}

func TestSchemaFromTypesDocument(t *testing.T) {
	schema, ok, err := SchemaFromTypesDocument("# A2K_ENGINE_HARNESS_trig-long.xsdat\nheader = string,int,string,int\n")
	if err != nil || !ok || schema != "a2k-trig-long" {
		t.Fatalf("schema=%q ok=%t err=%v", schema, ok, err)
	}
}

func TestInspectDataBytesAutoV9Payload(t *testing.T) {
	data := v9Payload(t)
	report := InspectDataBytes(data, DataInspectOptions{})
	if !report.OK {
		t.Fatalf("report not OK: %#v", report.Errors)
	}
	if report.Mode != "auto_heuristic" {
		t.Fatalf("mode = %q", report.Mode)
	}
	if len(report.Values) != 10 {
		t.Fatalf("values = %d, want 10", len(report.Values))
	}
	if report.Values[0].String != "A2K_RTV9_PERSIST" {
		t.Fatalf("magic = %q", report.Values[0].String)
	}
	if report.Values[3].Int == nil || *report.Values[3].Int != 424242 {
		t.Fatalf("sentinel = %#v", report.Values[3].Int)
	}
	if report.Values[8].String != "end" {
		t.Fatalf("footer label = %q", report.Values[8].String)
	}
}

func TestAutoShortUnalignedMixedStream(t *testing.T) {
	var data []byte
	str := func(s string) {
		data = binary.LittleEndian.AppendUint32(data, uint32(len(s)))
		data = append(data, s...)
	}
	word := func(v uint32) { data = binary.LittleEndian.AppendUint32(data, v) }
	str("SMALL_STREAM")
	word(1)
	str("form")
	str("Q")
	word(1)
	word(3)
	str("end")
	word(1)
	if len(data) != 52 {
		t.Fatalf("test stream is %d bytes", len(data))
	}
	r := InspectDataBytes(data, DataInspectOptions{})
	if !r.OK || r.Mode != "auto_heuristic" || len(r.Values) != 8 || len(r.HeuristicWarnings) == 0 {
		t.Fatalf("%+v", r)
	}
	offsets := []int{0, 16, 20, 28, 33, 37, 41, 48}
	for i, v := range r.Values {
		if v.Offset != offsets[i] || !v.Heuristic || v.RawHex == "" {
			t.Fatalf("value %d: %+v", i, v)
		}
	}
	if r.Values[0].String != "SMALL_STREAM" || r.Values[2].String != "form" || r.Values[3].String != "Q" || r.Values[6].String != "end" {
		t.Fatalf("strings: %+v", r.Values)
	}
	for i, want := range map[int]int32{1: 1, 4: 1, 5: 3, 7: 1} {
		v := r.Values[i]
		if v.Type != "int_or_float" || v.Int == nil || *v.Int != want || v.FloatCandidate == nil {
			t.Fatalf("word %d: %+v", i, v)
		}
	}
	typed := InspectDataBytes(data, DataInspectOptions{Types: []string{"string", "int", "string", "string", "int", "int", "string", "int"}})
	if !typed.OK || len(typed.Values) != 8 || typed.Values[5].Int == nil || *typed.Values[5].Int != 3 {
		t.Fatalf("typed: %+v", typed)
	}
}

func TestInspectDataBytesTypedV9Payload(t *testing.T) {
	types, err := ParseDataTypes("string,int,string,int,string,int,int,int,string,int")
	if err != nil {
		t.Fatal(err)
	}
	report := InspectDataBytes(v9Payload(t), DataInspectOptions{Types: types})
	if !report.OK {
		t.Fatalf("report not OK: %#v", report.Errors)
	}
	if report.Mode != "typed" {
		t.Fatalf("mode = %q", report.Mode)
	}
	if report.Values[7].Int == nil || *report.Values[7].Int != 123456 {
		t.Fatalf("xp = %#v", report.Values[7].Int)
	}
	if report.RemainingBytes != 0 {
		t.Fatalf("remaining = %d", report.RemainingBytes)
	}
}

func v9Payload(t *testing.T) []byte {
	t.Helper()
	var b bytes.Buffer
	writeString := func(s string) {
		if err := binary.Write(&b, binary.LittleEndian, uint32(len(s))); err != nil {
			t.Fatal(err)
		}
		b.WriteString(s)
	}
	writeInt := func(v int32) {
		if err := binary.Write(&b, binary.LittleEndian, v); err != nil {
			t.Fatal(err)
		}
	}
	writeString("A2K_RTV9_PERSIST")
	writeInt(9)
	writeString("writer_v9a")
	writeInt(424242)
	writeString("player")
	writeInt(3)
	writeInt(17)
	writeInt(123456)
	writeString("end")
	writeInt(9001)
	return b.Bytes()
}
