package kit

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"aoe2kit/pkg/scenario"
)

type SandboxDoctorReport struct {
	Platform        string               `json:"platform"`
	NetworkRequired bool                 `json:"network_required"`
	WritableDirs    []SandboxWritableDir `json:"writable_dirs"`
	Checks          []SandboxDoctorCheck `json:"checks"`
	Failures        []string             `json:"failures,omitempty"`
	OK              bool                 `json:"ok"`
}

type SandboxWritableDir struct {
	Path     string `json:"path"`
	OK       bool   `json:"ok"`
	Required bool   `json:"required"`
}

type SandboxDoctorCheck struct {
	Name   string `json:"name"`
	OK     bool   `json:"ok"`
	Detail string `json:"detail,omitempty"`
}

// RunSandboxDoctor exercises local scenario codec and writer primitives only.
func RunSandboxDoctor() SandboxDoctorReport {
	report := SandboxDoctorReport{Platform: runtime.GOOS + "/" + runtime.GOARCH}
	addCheck := func(name string, err error, detail string) {
		check := SandboxDoctorCheck{Name: name, OK: err == nil, Detail: detail}
		if err != nil {
			check.Detail = err.Error()
			report.Failures = append(report.Failures, name+": "+err.Error())
		}
		report.Checks = append(report.Checks, check)
	}
	checkDir := func(dir string, required bool) {
		file, err := os.CreateTemp(dir, ".aoe2kit-doctor-*")
		ok := err == nil
		if err == nil {
			name := file.Name()
			if closeErr := file.Close(); closeErr != nil {
				err, ok = closeErr, false
			}
			if removeErr := os.Remove(name); removeErr != nil {
				err, ok = removeErr, false
			}
		}
		report.WritableDirs = append(report.WritableDirs, SandboxWritableDir{Path: dir, OK: ok, Required: required})
		if required && !ok {
			report.Failures = append(report.Failures, fmt.Sprintf("required writable directory %s: %v", dir, err))
		}
	}

	tempRoot, err := os.MkdirTemp("", "aoe2kit-doctor-")
	if err != nil {
		addCheck("temporary_directory", err, "")
		addCheck("blank_fixture_codec", err, "not run")
		addCheck("blank_scenario_writer", err, "not run")
		return report
	}
	defer os.RemoveAll(tempRoot)
	checkDir(tempRoot, true)
	if cwd, cwdErr := os.Getwd(); cwdErr == nil {
		checkDir(cwd, false)
	}

	seed, err := scenario.BlankScenarioSeedBytes()
	if err == nil {
		var file *scenario.File
		file, err = scenario.Parse(seed)
		if err == nil {
			err = file.VerifyRebuild()
		}
	}
	addCheck("blank_fixture_codec", err, "embedded blank scenario parses and rebuilds")
	if err != nil {
		addCheck("blank_scenario_writer", err, "not run")
		report.OK = len(report.Failures) == 0
		return report
	}

	path := filepath.Join(tempRoot, "sandbox-check.aoe2scenario")
	_, err = scenario.WriteBlankScenarioFile(path, scenario.BlankOptions{
		PlayerCount: 2, HumanSlots: 1, ClearTriggers: true, NoStarters: true,
	})
	if err == nil {
		var file *scenario.File
		file, err = scenario.Open(path)
		if err == nil {
			zero := 0
			err = file.ApplyRecipe(scenario.Recipe{Triggers: []scenario.TriggerRecipe{{
				Op: "add_trigger", Name: "AoE2Kit sandbox doctor",
				Conditions: []scenario.ConditionRecipe{{Op: "timer", Timer: &zero}},
			}}})
		}
		if err == nil {
			err = file.Write(path)
		}
		if err == nil {
			file, err = scenario.Open(path)
		}
		if err == nil {
			err = file.VerifyRebuild()
			if err == nil && (file.Triggers == nil || file.Triggers.Count != 1 || len(file.Triggers.Triggers) != 1 || file.Triggers.Triggers[0].Name != "AoE2Kit sandbox doctor") {
				err = fmt.Errorf("trigger readback did not match the written trigger")
			}
		}
	}
	addCheck("blank_scenario_writer", err, "blank scenario, trigger write, reopen, and structural readback")
	for _, dir := range report.WritableDirs {
		if dir.Required && !dir.OK {
			return report
		}
	}
	report.OK = len(report.Failures) == 0
	return report
}
