package geotrace

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestProfileNotesAreNonOperative(t *testing.T) {
	plain := defaultEcologyProfile()
	raw, err := json.Marshal(plain)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "profile.json")
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	before, err := loadEcologyProfile(path)
	if err != nil {
		t.Fatal(err)
	}
	plain.Notes = map[string]json.RawMessage{"historical_intent": json.RawMessage(`{"factor":0.8,"standing":"non-operative"}`)}
	raw, err = json.Marshal(plain)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	after, err := loadEcologyProfile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(after.Notes) != 1 {
		t.Fatal("notes lost")
	}
	after.Notes = nil
	if !reflect.DeepEqual(before, after) {
		t.Fatal("notes changed effective profile")
	}
}

func TestDefaultMappingRetainsGenericRulesOnly(t *testing.T) {
	var root map[string]json.RawMessage
	if err := json.Unmarshal(defaultFeltMappingJSON, &root); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"florida_region_favoring", "coast_type_rule"} {
		if _, ok := root[key]; ok {
			t.Fatalf("regional rule %s leaked into defaults", key)
		}
	}
	for _, key := range []string{"water_isolation_rule", "water_fauna_diversity_rule", "water_fauna_by_band"} {
		if _, ok := root[key]; !ok {
			t.Fatalf("generic rule %s lost", key)
		}
	}
}
