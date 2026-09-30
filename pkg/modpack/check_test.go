package modpack

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCheckBasicMod(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "info.json"), `{"name":"Test Mod"}`)
	mustWrite(t, filepath.Join(dir, "resources", "_common", "ai", "Test.per2"), "(defrule)")
	mustWrite(t, filepath.Join(dir, "resources", "_common", "dat", "empires2_x2_p1.dat"), "dat")

	report, err := Check(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !report.OK() {
		t.Fatalf("unexpected errors: %v", report.Errors)
	}
	if !report.InfoJSON || !report.ResourcesCommon {
		t.Fatalf("expected info.json and resources/_common")
	}
	if len(report.AIFiles) != 1 || len(report.DataFiles) != 1 {
		t.Fatalf("unexpected files: ai=%v dat=%v", report.AIFiles, report.DataFiles)
	}
}

func TestCheckDataModDisabledInModStatus(t *testing.T) {
	dir := t.TempDir()
	mod := filepath.Join(dir, "mods", "local", "DecimaData")
	status := filepath.Join(dir, "mods", "mod-status.json")
	mustWrite(t, filepath.Join(mod, "info.json"), `{"Title":"DecimaData"}`)
	mustWrite(t, filepath.Join(mod, "resources", "_common", "dat", "empires2_x2_p1.dat"), "dat")
	mustWrite(t, status, `{"Mods":[{"Title":"DecimaData","Type":"Data","Enabled":false,"Published":false}]}`)

	report, err := CheckWithOptions(mod, CheckOptions{StatusPath: status})
	if err != nil {
		t.Fatal(err)
	}
	if report.OK() {
		t.Fatalf("expected disabled data mod to fail activation, got %#v", report.Activation)
	}
	if !report.Verification.StructureVerified {
		t.Fatalf("activation failure should not mark structure unverified")
	}
	if report.Verification.Label != "structure_ok_activation_failed" {
		t.Fatalf("unexpected verification label: %s", report.Verification.Label)
	}
	if report.Activation == nil || report.Activation.Enabled == nil || *report.Activation.Enabled {
		t.Fatalf("expected enabled=false in activation report: %#v", report.Activation)
	}
}

func TestCheckDataModEnabledInModStatus(t *testing.T) {
	dir := t.TempDir()
	mod := filepath.Join(dir, "mods", "local", "DecimaData")
	status := filepath.Join(dir, "mods", "mod-status.json")
	mustWrite(t, filepath.Join(mod, "info.json"), `{"Title":"DecimaData"}`)
	mustWrite(t, filepath.Join(mod, "resources", "_common", "dat", "empires2_x2_p1.dat"), "dat")
	mustWrite(t, status, `{"Mods":[{"Title":"DecimaData","Type":"Data","Enabled":true,"Published":true}]}`)

	report, err := CheckWithOptions(mod, CheckOptions{StatusPath: status})
	if err != nil {
		t.Fatal(err)
	}
	if !report.OK() {
		t.Fatalf("expected enabled data mod to pass, errors=%v warnings=%v", report.Errors, report.Warnings)
	}
	if report.Activation == nil || !report.Activation.ActivationOK {
		t.Fatalf("expected activation ok: %#v", report.Activation)
	}
}

func mustWrite(t *testing.T, path, data string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
}
