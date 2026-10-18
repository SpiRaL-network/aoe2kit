package main

import (
	"encoding/binary"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	xsauthor "aoe2kit/pkg/xs"
)

func captureXSDatOutput(t *testing.T, run func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = old; r.Close(); w.Close() }()
	run()
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	os.Stdout = old
	data, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestXSDatDecodeShortMixedStreamShowsValues(t *testing.T) {
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
	path := filepath.Join(t.TempDir(), "sample.xsdat")
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
	text := captureXSDatOutput(t, func() { runXSDatDecode([]string{path, "--text"}) })
	for _, want := range []string{"bytes=52 mode=auto_heuristic", "warning: auto mode has no type tags", `[00] @0000 string`, `"SMALL_STREAM"`, `"form"`, `"Q"`, "[04] @0033 int_or_float", "[05] @0037 int_or_float size=4 int=3", `"end"`, "[07] @0048 int_or_float"} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q:\n%s", want, text)
		}
	}
	jsonText := captureXSDatOutput(t, func() { runXSDatDecode([]string{path, "--json"}) })
	var report xsauthor.DataDecodeReport
	if err := json.Unmarshal([]byte(jsonText), &report); err != nil {
		t.Fatal(err)
	}
	if !report.OK || len(report.Decode.Values) != 8 || report.Decode.Values[5].Int == nil || *report.Decode.Values[5].Int != 3 {
		t.Fatalf("%s", jsonText)
	}
}
