package xs

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateBridge(t *testing.T) {
	report, err := GenerateBridge(BridgeSpec{
		Variables: map[string]int{"p1_level": 12, "p1_mana": 13},
		Reads:     []string{"p1_level"},
		Writes:    []string{"p1_mana"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if report.OK {
		t.Fatalf("expected dangling read/write warnings")
	}
	if !strings.Contains(report.XSModule, "int get_p1_level()") {
		t.Fatalf("missing getter:\n%s", report.XSModule)
	}
	if !strings.Contains(report.XSModule, "xsTriggerVariable(A2K_VAR_P1_LEVEL)") {
		t.Fatalf("getter does not wrap xsTriggerVariable:\n%s", report.XSModule)
	}
	if !strings.Contains(report.XSModule, "extern const int A2K_VAR_P1_LEVEL = 12;") {
		t.Fatalf("named constants must be extern for cross-file XS visibility:\n%s", report.XSModule)
	}
}

func TestGenerateShims(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "runtime.xs")
	if err := os.WriteFile(path, []byte("void boot() {}\nint with_arg(int x = 0) { return(x); }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	report, err := GenerateShims(ShimOptions{Paths: []string{dir}, Prefix: "Shim", Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	if report.Emitted != 1 || report.Skipped != 1 {
		t.Fatalf("emitted/skipped = %d/%d", report.Emitted, report.Skipped)
	}
	triggers := report.Recipe["triggers"].([]map[string]any)
	if triggers[0]["name"] != "Shim boot" {
		t.Fatalf("unexpected trigger: %+v", triggers[0])
	}
}

func TestGenerateDatagen(t *testing.T) {
	report, err := GenerateDatagen(DatagenSpec{
		Function: "init_arrays",
		Arrays: []DatagenArray{
			{Name: "zone_ids", Type: "int", Values: rawValues("1", "2", "3")},
			{Name: "zone_names", Type: "string", Values: rawValues(`"a"`, `"b"`)},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !report.Verified {
		t.Fatalf("expected verified report")
	}
	for _, needle := range []string{"zone_ids = xsArrayCreateInt(3, 0, \"zone_ids\");", "xsArraySetString(zone_names, 1, \"b\");"} {
		if !strings.Contains(report.XSModule, needle) {
			t.Fatalf("missing %q in:\n%s", needle, report.XSModule)
		}
	}
}

func rawValues(values ...string) []json.RawMessage {
	out := make([]json.RawMessage, 0, len(values))
	for _, value := range values {
		out = append(out, json.RawMessage(value))
	}
	return out
}
