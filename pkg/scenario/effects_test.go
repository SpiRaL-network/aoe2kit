package scenario

import (
	"aoe2kit/pkg/testfixtures"
	"os"
	"testing"
)

func TestScenarioEffectsFrontTowersGolden(t *testing.T) {
	path := testfixtures.Path(t, "save-analysis/CB_FRONT_TOWERS_V247.aoe2scenario")
	if _, err := os.Stat(path); err != nil {
		t.Skipf("golden scenario not present: %v", err)
	}
	scen, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	report := scen.Effects()
	for _, name := range []string{"create_object", "kill_object", "display_instructions"} {
		if bucketByName(report.Types, name) == nil {
			t.Fatalf("missing effect bucket %q", name)
		}
	}
	create := bucketByName(report.Types, "create_object")
	if create.Count == 0 || len(create.Sample) == 0 {
		t.Fatalf("missing create_object sample")
	}
	if create.Sample[0].UnitConst == 0 || len(create.Sample[0].Location) != 2 {
		t.Fatalf("create_object sample lacks unit/location: %+v", create.Sample[0])
	}
	display := bucketByName(report.Types, "display_instructions")
	if display.Count == 0 || len(display.Sample) == 0 {
		t.Fatalf("missing display_instructions sample")
	}
	if display.Sample[0].Text == "" {
		t.Fatalf("display_instructions sample lacks text: %+v", display.Sample[0])
	}
}

func TestEffectsCensusMatchesFullReport(t *testing.T) {
	path := testfixtures.Path(t, "save-analysis/CB_FRONT_TOWERS_V247.aoe2scenario")
	if _, err := os.Stat(path); err != nil {
		t.Skipf("golden scenario not present: %v", err)
	}
	scen, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	full := scen.Effects()
	census, err := EffectsCensusFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if census.EffectCount != full.Total {
		t.Fatalf("census effect_count = %d, want %d", census.EffectCount, full.Total)
	}
	if census.TriggerCount != scen.Triggers.Count {
		t.Fatalf("census trigger_count = %d, want %d", census.TriggerCount, scen.Triggers.Count)
	}
	for _, fullBucket := range full.Types {
		censusBucket := bucketByName(census.Types, fullBucket.TypeName)
		if censusBucket == nil {
			t.Fatalf("census missing bucket %q", fullBucket.TypeName)
		}
		if censusBucket.Count != fullBucket.Count {
			t.Fatalf("census bucket %q count = %d, want %d", fullBucket.TypeName, censusBucket.Count, fullBucket.Count)
		}
	}
}

func TestEffectTypeNamesIncludeModernDEIDs(t *testing.T) {
	cases := map[int]string{
		61:  "disable_unit_targeting",
		74:  "disable_object_deletion",
		88:  "change_object_caption",
		98:  "disable_unit_attackable",
		105: "modify_object_attribute",
		106: "modify_object_attribute_by_variable",
	}
	for id, want := range cases {
		if got := EffectTypeName(id); got != want {
			t.Fatalf("EffectTypeName(%d) = %q, want %q", id, got, want)
		}
	}
}

func bucketByName(types []EffectTypeBucket, name string) *EffectTypeBucket {
	for i := range types {
		if types[i].TypeName == name {
			return &types[i]
		}
	}
	return nil
}
