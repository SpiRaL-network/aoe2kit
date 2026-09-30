package replay

import (
	"aoe2kit/pkg/testfixtures"
	"os"
	"testing"
)

func TestFrontierLatestCBAGolden(t *testing.T) {
	path := testfixtures.Path(t, "save-analysis/latest_cba.aoe2record")
	if _, err := os.Stat(path); err != nil {
		t.Skipf("golden replay not present: %v", err)
	}
	report, err := BuildFrontier(path, FrontierOptions{MinDuplicateBytes: 256, Limit: 5})
	if err != nil {
		t.Fatal(err)
	}
	if report.Summary.HeaderOpaqueBytes < 1000000 {
		t.Fatalf("header opaque bytes = %d, want substantial frontier", report.Summary.HeaderOpaqueBytes)
	}
	if report.Summary.BodyOpaqueBytes != 0 {
		t.Fatalf("body opaque bytes = %d, want 0", report.Summary.BodyOpaqueBytes)
	}
	if bucket := frontierBucketByName(report.Buckets, "v68_pre_trigger_tail"); bucket == nil || bucket.Bytes < 450000 {
		t.Fatalf("v68_pre_trigger_tail bucket = %+v, want >450KB residual frontier", bucket)
	}
	if bucket := frontierBucketByName(report.Buckets, "referenced_object_body_remainder"); bucket == nil || bucket.Bytes < 300000 {
		t.Fatalf("referenced_object_body_remainder bucket = %+v, want >300KB residual frontier", bucket)
	}
	if bucket := frontierBucketByName(report.Buckets, "v68_pre_trigger_tail"); bucket == nil || bucket.TemplateHitBytes != 0 || bucket.ResidualBytes != bucket.Bytes {
		t.Fatalf("v68_pre_trigger_tail template accounting = %+v, want promoted template duplicates removed from frontier", bucket)
	}
	if len(report.Duplicates) == 0 {
		t.Fatalf("expected exact duplicate opaque groups")
	}
	if report.Summary.DuplicateGroups <= report.Summary.ShownDuplicates {
		t.Fatalf("duplicate totals should exceed shown limited rows: total=%d shown=%d", report.Summary.DuplicateGroups, report.Summary.ShownDuplicates)
	}
	if len(report.TemplateHits) != 0 || report.Summary.TemplateHitCount != 0 || report.Summary.TemplateHitBytes != 0 {
		t.Fatalf("template hits = count %d shown %d bytes %d, want 0 because exact template duplicates are promoted in coverage",
			report.Summary.TemplateHitCount, report.Summary.ShownTemplateHits, report.Summary.TemplateHitBytes)
	}
	foundCrossRegionDuplicate := false
	for _, group := range report.Duplicates {
		hasObject := false
		hasV68 := false
		for _, bucket := range group.Buckets {
			if bucket == "referenced_object_body_remainder" || bucket == "player_object_tail_after_candidates" {
				hasObject = true
			}
			if bucket == "v68_pre_trigger_tail" {
				hasV68 = true
			}
		}
		if hasObject && hasV68 {
			foundCrossRegionDuplicate = true
			break
		}
	}
	if !foundCrossRegionDuplicate {
		t.Fatalf("expected object/v68 cross-region duplicate group")
	}
}

func frontierBucketByName(buckets []FrontierBucket, name string) *FrontierBucket {
	for i := range buckets {
		if buckets[i].Name == name {
			return &buckets[i]
		}
	}
	return nil
}
