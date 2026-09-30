package replay

import "testing"

func TestChecksumProbeV6PresetSpecs(t *testing.T) {
	specs, err := checksumProbeSpecs(ChecksumProbeOptions{Preset: "v6-playground-helper"})
	if err != nil {
		t.Fatalf("checksumProbeSpecs returned error: %v", err)
	}
	if len(specs) != 4 {
		t.Fatalf("spec count = %d, want 4", len(specs))
	}
	hostile := specs[0]
	if hostile.Name != "p3_hostile_dummies_removed" || hostile.PlayerID != 3 {
		t.Fatalf("hostile spec = %+v", hostile)
	}
	if hostile.ObjectCountDelta == nil || *hostile.ObjectCountDelta != -3 {
		t.Fatalf("hostile count delta = %v, want -3", hostile.ObjectCountDelta)
	}
	if hostile.UnitTypeSumDelta == nil || *hostile.UnitTypeSumDelta != -526 {
		t.Fatalf("hostile type-sum delta = %v, want -526", hostile.UnitTypeSumDelta)
	}
	gaia := specs[1]
	if gaia.Name != "gaia_neutral_dummies_removed_any_row" || gaia.PlayerID != 0 {
		t.Fatalf("gaia spec = %+v", gaia)
	}
	if gaia.UnitTypeSumDelta == nil || *gaia.UnitTypeSumDelta != -501 {
		t.Fatalf("gaia type-sum delta = %v, want -501", gaia.UnitTypeSumDelta)
	}
}

func TestChecksumProbeV7PresetSpecs(t *testing.T) {
	specs, err := checksumProbeSpecs(ChecksumProbeOptions{Preset: "v7-checksum-calibration"})
	if err != nil {
		t.Fatalf("checksumProbeSpecs returned error: %v", err)
	}
	if len(specs) != 21 {
		t.Fatalf("spec count = %d, want 21", len(specs))
	}
	if specs[0].Name != "create_villager_male" || specs[0].PlayerID != 1 {
		t.Fatalf("first v7 spec = %+v", specs[0])
	}
	if specs[7].Name != "kill_villager_male" || specs[7].PlayerID != 1 {
		t.Fatalf("first v7 kill spec = %+v", specs[7])
	}
	if specs[14].Name != "kill_object_replacement_villager_male" || specs[14].PlayerID != 1 {
		t.Fatalf("first v7 replacement spec = %+v", specs[14])
	}
}

func TestChecksumProbeMatches(t *testing.T) {
	neg3 := int64(-3)
	neg526 := int64(-526)
	spec := ChecksumProbeSpec{
		PlayerID:         3,
		ObjectCountDelta: &neg3,
		UnitTypeSumDelta: &neg526,
	}
	if !checksumProbeMatches(spec, SyncStateDelta{PlayerID: 3, ObjectCountDelta: -3, UnitTypeSumDelta: -526}) {
		t.Fatalf("expected exact delta match")
	}
	if checksumProbeMatches(spec, SyncStateDelta{PlayerID: 2, ObjectCountDelta: -3, UnitTypeSumDelta: -526}) {
		t.Fatalf("wrong player matched")
	}
	if checksumProbeMatches(spec, SyncStateDelta{PlayerID: 3, ObjectCountDelta: -3, UnitTypeSumDelta: -501}) {
		t.Fatalf("wrong unit-type sum matched")
	}
}
