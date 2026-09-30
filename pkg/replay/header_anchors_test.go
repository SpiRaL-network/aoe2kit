package replay

import (
	"aoe2kit/pkg/testfixtures"
	"os"
	"testing"
)

func TestHeaderAnchorsV4DiagnosticGolden(t *testing.T) {
	path := testfixtures.Path(t, "save-analysis/diagnostics/v4_run_20260722.aoe2record")
	if _, err := os.Stat(path); err != nil {
		t.Skipf("golden replay not present: %v", err)
	}
	report, err := BuildHeaderAnchors(path, HeaderAnchorsOptions{Limit: 0})
	if err != nil {
		t.Fatal(err)
	}
	if report.Summary.CaptionAnchors < 14 {
		t.Fatalf("caption anchors = %d, want at least 14", report.Summary.CaptionAnchors)
	}
	if report.Summary.CaptionMedianStrideBytes != 745 {
		t.Fatalf("caption median stride = %d, want 745", report.Summary.CaptionMedianStrideBytes)
	}
	firstCaption := headerCaptionByText(report.Captions, "SDSV4_CLICK_SCOUT_920101")
	if firstCaption == nil {
		t.Fatalf("missing SDSV4 click-scout caption")
	}
	if firstCaption.MarkerStart != 5351263 || firstCaption.TextStart != 5351279 || firstCaption.End != 5351303 {
		t.Fatalf("click-scout caption offsets = %d/%d/%d, want 5351263/5351279/5351303", firstCaption.MarkerStart, firstCaption.TextStart, firstCaption.End)
	}
	block := headerResourceBlockByConfidence(report.Resources, "engine_verified_v4_resource_probe_anchor")
	if block == nil {
		t.Fatalf("missing v4 resource sentinel block")
	}
	if block.Start != 8561836 || block.End != 8561920 || len(block.Players) != 3 {
		t.Fatalf("resource block = %d..%d players=%d, want 8561836..8561920 players=3", block.Start, block.End, len(block.Players))
	}
	want := []uint32{91001, 92001, 93001}
	for i, value := range want {
		if block.Players[i].Gold != value || block.Players[i].Wood != value+1 || block.Players[i].Food != value+2 || block.Players[i].Stone != value+3 {
			t.Fatalf("P%d resources = %d/%d/%d/%d, want %d/%d/%d/%d",
				i+1,
				block.Players[i].Gold,
				block.Players[i].Wood,
				block.Players[i].Food,
				block.Players[i].Stone,
				value,
				value+1,
				value+2,
				value+3)
		}
	}
	coverage, err := BuildCoverage(path)
	if err != nil {
		t.Fatal(err)
	}
	if !hasCoverageRegionWithConfidence(coverage, "decoded", "engine_verified_v4_resource_probe_anchor") {
		t.Fatalf("coverage did not carve v4 resource block")
	}
}

func headerCaptionByText(captions []HeaderCaptionAnchor, text string) *HeaderCaptionAnchor {
	for i := range captions {
		if captions[i].Text == text {
			return &captions[i]
		}
	}
	return nil
}

func headerResourceBlockByConfidence(blocks []PlayerResourceBlock, confidence string) *PlayerResourceBlock {
	for i := range blocks {
		if blocks[i].Confidence == confidence {
			return &blocks[i]
		}
	}
	return nil
}

func hasCoverageRegionWithConfidence(report *CoverageReport, status string, confidence string) bool {
	for _, region := range report.Regions {
		if region.Status == status && region.Confidence == confidence {
			return true
		}
	}
	return false
}
