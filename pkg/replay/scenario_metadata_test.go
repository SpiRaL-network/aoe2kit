package replay

import (
	"encoding/binary"
	"testing"
)

func TestScanEarlyScenarioMetadataFindsScenarioLikeString(t *testing.T) {
	header := make([]byte, 2200)
	copy(header[400:], deStringRecord("resources/_common/ai/Promisory/init.per"))
	copy(header[1900:], deStringRecord("CBA Mini Castle Blood2 www.aoe2cba.com"))
	meta := scanEarlyScenarioMetadata(header)
	if meta.Name != "CBA Mini Castle Blood2 www.aoe2cba.com" {
		t.Fatalf("scenario name = %q", meta.Name)
	}
	if meta.Source != "early_de_string_scan" || meta.Confidence == "" {
		t.Fatalf("metadata source/confidence = %+v", meta)
	}
}

func TestPlausibleScenarioTitleRejectsPaths(t *testing.T) {
	for _, text := range []string{
		"resources/_common/ai/Promisory/init.per",
		"RANDOM-MAP-SCRIPTS:CBA_=REQUIEM=_V292",
		"Promisory/general",
	} {
		if plausibleScenarioTitle(text) {
			t.Fatalf("accepted path/script string %q", text)
		}
	}
	if !plausibleScenarioTitle("CBA - Sorry Noobs") {
		t.Fatal("rejected plausible scenario title")
	}
}

func TestNormalizeScenarioTitleCandidateFromScenarioPath(t *testing.T) {
	got, source, ok := normalizeScenarioTitleCandidate("USER:SCENARIOS:MCB EXCAL VIII July 2026 Edition.aoe2scenario::false")
	if !ok {
		t.Fatal("normalizer rejected scenario path")
	}
	if got != "MCB EXCAL VIII July 2026 Edition" || source != "early_de_scenario_path" {
		t.Fatalf("normalized = %q source=%q", got, source)
	}
}

func deStringRecord(text string) []byte {
	payload := append([]byte(text), 0)
	out := make([]byte, 4+len(payload))
	out[0] = 0x60
	out[1] = 0x0a
	binary.LittleEndian.PutUint16(out[2:], uint16(len(payload)))
	copy(out[4:], payload)
	return out
}
