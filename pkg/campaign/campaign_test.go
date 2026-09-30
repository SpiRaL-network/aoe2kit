package campaign

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateSamePartyCampaign(t *testing.T) {
	dir := t.TempDir()
	spec := &Spec{
		SchemaVersion:  1,
		Name:           "Test Campaign",
		WriterScenario: "Writer Scenario",
		Players:        2,
		Fields: []Field{
			{Name: "level", Type: "int", PerPlayer: true, Default: 1},
			{Name: "chapter", Type: "string", Default: "start"},
		},
	}
	report, err := Generate(spec, GenerateOptions{OutDir: dir, Prefix: "Camp"})
	if err != nil {
		t.Fatal(err)
	}
	if !report.OK || len(report.Files) != 4 {
		t.Fatalf("report = %+v", report)
	}
	reader, err := os.ReadFile(filepath.Join(dir, "campaign_reader.xs"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(reader)
	for _, want := range []string{
		`xsOpenFile("Writer Scenario")`,
		"Camp_p1_level = xsReadInt();",
		"Camp_p2_level = xsReadInt();",
		`xsChatData("A2KCAMPAIGN READ_OK")`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("reader missing %q:\n%s", want, text)
		}
	}
}

func TestGenerateRejectsBadFieldName(t *testing.T) {
	_, err := Generate(&Spec{
		Name:           "Bad",
		WriterScenario: "Writer",
		Players:        1,
		Fields:         []Field{{Name: "not valid", Type: "int"}},
	}, GenerateOptions{OutDir: t.TempDir()})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestExtractCampaignContainer(t *testing.T) {
	dir := t.TempDir()
	campaignPath := filepath.Join(dir, "test.aoe2campaign")
	payload1 := []byte("not a real scenario one")
	payload2 := []byte("not a real scenario two")
	data := fakeCampaign(t, []fakeCampaignEntry{
		{name: "First.aoe2scenario", filename: "First.aoe2scenario", payload: payload1},
		{name: "Second.aoe2scenario", filename: "Second.aoe2scenario", payload: payload2},
	})
	if err := os.WriteFile(campaignPath, data, 0o644); err != nil {
		t.Fatal(err)
	}
	outDir := filepath.Join(dir, "out")
	report, err := ExtractFile(campaignPath, ExtractOptions{OutDir: outDir, Scenario: 2})
	if err != nil {
		t.Fatal(err)
	}
	if !report.OK || report.DirectoryCount != 2 || len(report.Entries) != 2 {
		t.Fatalf("report = %+v", report)
	}
	if report.Entries[0].Selected {
		t.Fatalf("first entry selected unexpectedly: %+v", report.Entries[0])
	}
	selected := report.Entries[1]
	if !selected.Selected || selected.OutPath == "" {
		t.Fatalf("second entry not extracted: %+v", selected)
	}
	got, err := os.ReadFile(selected.OutPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, payload2) {
		t.Fatalf("payload = %q, want %q", got, payload2)
	}
	if selected.FullParseOK {
		t.Fatalf("fake payload parsed as real scenario: %+v", selected)
	}
}

type fakeCampaignEntry struct {
	name     string
	filename string
	payload  []byte
}

func fakeCampaign(t *testing.T, entries []fakeCampaignEntry) []byte {
	t.Helper()
	var table bytes.Buffer
	offset := 4 + 4 + 4
	for _, entry := range entries {
		offset += 8 + campaignStringLen(entry.name) + campaignStringLen(entry.filename)
	}
	payloadOffset := offset
	for _, entry := range entries {
		binary.Write(&table, binary.LittleEndian, uint32(len(entry.payload)))
		binary.Write(&table, binary.LittleEndian, uint32(payloadOffset))
		writeCampaignStringForTest(&table, entry.name)
		writeCampaignStringForTest(&table, entry.filename)
		payloadOffset += len(entry.payload)
	}
	var out bytes.Buffer
	out.WriteString("2.00")
	binary.Write(&out, binary.LittleEndian, uint32(len(entries)))
	binary.Write(&out, binary.LittleEndian, uint32(len(entries)))
	out.Write(table.Bytes())
	for _, entry := range entries {
		out.Write(entry.payload)
	}
	return out.Bytes()
}

func campaignStringLen(value string) int {
	return 4 + len(value)
}

func writeCampaignStringForTest(out *bytes.Buffer, value string) {
	binary.Write(out, binary.LittleEndian, uint16(0x0a60))
	binary.Write(out, binary.LittleEndian, uint16(len(value)))
	out.WriteString(value)
}
