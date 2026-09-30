package xs

import (
	"bytes"
	"encoding/binary"
	"math"
	"os"
	"path/filepath"
	"testing"
)

func TestDecodeDataFileWithV9Ledger(t *testing.T) {
	dir := t.TempDir()
	dataPath := filepath.Join(dir, "v9.xsdat")
	if err := os.WriteFile(dataPath, v9Payload(t), 0o644); err != nil {
		t.Fatal(err)
	}
	ledgerPath := filepath.Join(dir, "ledger.json")
	ledger := `{"writer":{"payload":["string A2K_RTV9_PERSIST","int 9","string writer_v9a","int 424242","string player","int 3","int 17","int 123456","string end","int 9001"]}}`
	if err := os.WriteFile(ledgerPath, []byte(ledger), 0o644); err != nil {
		t.Fatal(err)
	}
	report, err := DecodeDataFile(dataPath, DataDecodeOptions{Ledger: ledgerPath})
	if err != nil {
		t.Fatal(err)
	}
	if !report.OK || report.Summary.Passed != 10 || report.Summary.Failed != 0 {
		t.Fatalf("report = %+v", report)
	}
}

func TestDecodeDataFileWithTruncatedLedgerFailsGracefully(t *testing.T) {
	dir := t.TempDir()
	dataPath := filepath.Join(dir, "short.xsdat")
	if err := os.WriteFile(dataPath, v9Payload(t)[:30], 0o644); err != nil {
		t.Fatal(err)
	}
	ledgerPath := filepath.Join(dir, "ledger.json")
	ledger := `{"payload":["string A2K_RTV9_PERSIST","int 9","string writer_v9a","int 424242"]}`
	if err := os.WriteFile(ledgerPath, []byte(ledger), 0o644); err != nil {
		t.Fatal(err)
	}
	report, err := DecodeDataFile(dataPath, DataDecodeOptions{Ledger: ledgerPath})
	if err != nil {
		t.Fatal(err)
	}
	if report.OK || len(report.Errors) == 0 || report.Summary.Failed == 0 {
		t.Fatalf("truncated report should fail gracefully: %+v", report)
	}
}

func TestDecodeDataFileWithStructuredV8LedgerFailsOnMissingRows(t *testing.T) {
	dir := t.TempDir()
	dataPath := filepath.Join(dir, "v8.xsdat")
	if err := os.WriteFile(dataPath, v8PartialPayload(t), 0o644); err != nil {
		t.Fatal(err)
	}
	ledgerPath := filepath.Join(dir, "ledger.json")
	ledger := `{
  "fixture": "Replay Transparency Diagnostic v8.2 Assertion Fixture",
  "rows": [
    { "phase_id": 0, "timer": 5, "expected_food": 880000 },
    { "phase_id": 80, "timer": 100, "expected_food": 880080, "expected_counts": { "p2_scout_448": 0 } }
  ],
  "append_probe": { "expected_tail": "append_probe + int 812382" }
}`
	if err := os.WriteFile(ledgerPath, []byte(ledger), 0o644); err != nil {
		t.Fatal(err)
	}
	report, err := DecodeDataFile(dataPath, DataDecodeOptions{Ledger: ledgerPath})
	if err != nil {
		t.Fatal(err)
	}
	if report.OK || report.Summary.Failed == 0 {
		t.Fatalf("structured v8 ledger should fail on missing phase/tail rows: %+v", report)
	}
	if report.Summary.Passed == 0 {
		t.Fatalf("structured v8 ledger should still pass present rows: %+v", report)
	}
}

func v8PartialPayload(t *testing.T) []byte {
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
	writeFloat := func(v float32) {
		if err := binary.Write(&b, binary.LittleEndian, math.Float32bits(v)); err != nil {
			t.Fatal(err)
		}
	}
	writeString("A2K_RTV8_ASSERTION")
	writeInt(82)
	writeString("open_once_close_once_plus_append_probe")
	writeString("phase")
	writeInt(0)
	writeInt(5)
	writeFloat(880000)
	writeInt(1)
	writeInt(0)
	writeInt(0)
	writeInt(2)
	writeInt(0)
	writeInt(0)
	writeInt(1)
	writeFloat(28)
	return b.Bytes()
}
