package datcache

import (
	"testing"

	"aoe2kit/pkg/testfixtures"
)

func TestSemanticPriorsReferenceDAT(t *testing.T) {
	path := testfixtures.Path(t, "reference_aoe2_dump/dat/empires2_x2_p1.dat")
	cache, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer cache.Close()
	report, err := cache.SemanticPriors()
	if err != nil {
		t.Fatal(err)
	}
	if report.UnitIdentities != 2642 || report.SourceRows != 128725 {
		t.Fatalf("identity counts = %d/%d, want 2642/128725", report.UnitIdentities, report.SourceRows)
	}
	if report.ZeroCollision != 813 || report.ZeroCollisionCreatable != 55 {
		t.Fatalf("zero collision = %d/%d, want 813/55", report.ZeroCollision, report.ZeroCollisionCreatable)
	}
	if len(report.CollisionSignatures) != 4 {
		t.Fatalf("collision signatures = %d, want 4", len(report.CollisionSignatures))
	}
	wantRemoval := map[string]int{"class14_hp1": 241, "class14_other": 249, "hp1_other": 384}
	for _, bucket := range report.RemovalBuckets {
		if bucket.Units != wantRemoval[bucket.Name] {
			t.Fatalf("removal bucket %s = %d, want %d", bucket.Name, bucket.Units, wantRemoval[bucket.Name])
		}
	}
	if report.Verification != "dat_derived_prior_not_engine_verified" {
		t.Fatalf("verification = %q", report.Verification)
	}
}
