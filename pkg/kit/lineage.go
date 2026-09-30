package kit

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"aoe2kit/pkg/aoe2"
)

type ArtifactLineageOptions struct {
	Tool    string
	Command []string
	Notes   []string
}

type ArtifactLineageReport struct {
	OK           bool            `json:"ok"`
	GeneratedAt  string          `json:"generated_at"`
	Verification string          `json:"verification"`
	Tool         string          `json:"tool,omitempty"`
	Command      []string        `json:"command,omitempty"`
	Inputs       []aoe2.FileInfo `json:"inputs"`
	Outputs      []aoe2.FileInfo `json:"outputs"`
	Notes        []string        `json:"notes,omitempty"`
	Warnings     []string        `json:"warnings,omitempty"`
}

func BuildArtifactLineage(inputs, outputs []string, opts ArtifactLineageOptions) (ArtifactLineageReport, error) {
	if len(inputs) == 0 {
		return ArtifactLineageReport{}, fmt.Errorf("at least one --input is required")
	}
	if len(outputs) == 0 {
		return ArtifactLineageReport{}, fmt.Errorf("at least one --output is required")
	}
	report := ArtifactLineageReport{
		OK:           true,
		GeneratedAt:  time.Now().UTC().Format(time.RFC3339),
		Verification: "structure_verified_not_engine_verified",
		Tool:         opts.Tool,
		Command:      append([]string(nil), opts.Command...),
		Notes:        append([]string(nil), opts.Notes...),
	}
	for _, path := range inputs {
		info, err := aoe2.InspectFile(path)
		if err != nil {
			return ArtifactLineageReport{}, fmt.Errorf("inspect input %q: %w", path, err)
		}
		report.Inputs = append(report.Inputs, info)
	}
	for _, path := range outputs {
		info, err := aoe2.InspectFile(path)
		if err != nil {
			return ArtifactLineageReport{}, fmt.Errorf("inspect output %q: %w", path, err)
		}
		report.Outputs = append(report.Outputs, info)
	}
	if report.Tool == "" && len(report.Command) == 0 {
		report.Warnings = append(report.Warnings, "no tool or command recorded; lineage proves file identity but not the build procedure")
	}
	return report, nil
}

func WriteArtifactLineage(path string, report ArtifactLineageReport) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0644)
}
