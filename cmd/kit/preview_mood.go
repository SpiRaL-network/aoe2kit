package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"aoe2kit/pkg/gameinstall"
	"aoe2kit/pkg/preview"
)

func runPreviewCalibrateMood(args []string) {
	fs := flag.NewFlagSet("preview calibrate-mood", flag.ContinueOnError)
	manifest := fs.String("manifest", "", "JSON image pairs, relative paths, optional regions and offsets")
	samples := fs.String("samples", "", "previous calibrate-mood report (reuse its samples without images)")
	gameDir := fs.String("game-dir", "", "installed game root")
	holdout := fs.String("holdout", "", "mood excluded from fitting and model selection (required with --fit)")
	fit := fs.Bool("fit", false, "fit shared tint/bloom/fog parameters")
	shader := fs.String("shader-options", "", "JSON engine options for a fixed shader uniform-patch audit (no fitting)")
	out := fs.String("out", "", "JSON output path (default stdout)")
	if err := fs.Parse(args); err != nil {
		die("kit preview calibrate-mood", err)
	}
	if (*manifest == "") == (*samples == "") || fs.NArg() != 0 || (*fit && (*holdout == "" || *shader != "")) {
		die("kit preview calibrate-mood", fmt.Errorf("use --manifest FILE --game-dir DIR (or --samples REPORT) [--fit --holdout MOOD | --shader-options FILE] [--out FILE]"))
	}
	var cases []preview.MoodFitCase
	var alignment []preview.MoodAlignment
	var err error
	if *samples != "" {
		var saved struct {
			Samples    []preview.MoodFitCase   `json:"samples"`
			Alignments []preview.MoodAlignment `json:"alignments"`
		}
		var b []byte
		b, err = os.ReadFile(*samples)
		if err == nil {
			err = json.Unmarshal(b, &saved)
		}
		cases, alignment = saved.Samples, saved.Alignments
	} else {
		var root string
		root, err = gameinstall.Resolve(*gameDir)
		if err == nil {
			cases, alignment, err = preview.PrepareMoodCalibration(*manifest, root)
		}
	}
	if err != nil {
		die("kit preview calibrate-mood", err)
	}
	if err = preview.ValidateMoodFitCases(cases); err != nil {
		die("kit preview calibrate-mood", err)
	}
	model := preview.CurrentMoodModel()
	var scores map[string]float64
	var audit *preview.ShaderMoodAudit
	if *shader != "" {
		var opts preview.ShaderMoodOptions
		b, err := os.ReadFile(*shader)
		if err == nil {
			err = json.Unmarshal(b, &opts)
		}
		if err != nil {
			die("kit preview calibrate-mood", err)
		}
		r, err := preview.ShaderMoodResiduals(cases, opts, *holdout)
		if err != nil {
			die("kit preview calibrate-mood", err)
		}
		audit = &r
	}
	if *fit {
		model, scores, err = preview.FitMoodModel(cases, *holdout)
		if err != nil {
			die("kit preview calibrate-mood", err)
		}
	}
	method := "display-RGB RMS (0..255); equal case weights; fit uses at most 1024 evenly spaced samples per case, two initializations; residuals use all samples; held-out mood excluded from fitting and model selection"
	results := preview.MoodResiduals(cases, model, *holdout)
	var reportedModel *preview.MoodModel = &model
	if audit != nil {
		method = "fixed shader equations under explicit uniform-patch assumptions; no fitted coefficients; display-RGB RMS (0..255) vs current default; not an engine rendering"
		results = audit.Results
		reportedModel = nil
	}
	report := struct {
		Method     string                   `json:"method"`
		Model      *preview.MoodModel       `json:"model,omitempty"`
		Shader     *preview.ShaderMoodAudit `json:"shader,omitempty"`
		Candidates map[string]float64       `json:"training_rms_by_tint_shape,omitempty"`
		Alignments []preview.MoodAlignment  `json:"alignments"`
		Results    []preview.MoodFitResult  `json:"results"`
		Samples    []preview.MoodFitCase    `json:"samples"`
	}{method, reportedModel, audit, scores, alignment, results, cases}
	b, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		die("kit preview calibrate-mood", err)
	}
	b = append(b, '\n')
	if *out != "" {
		if err = os.WriteFile(*out, b, 0644); err != nil {
			die("kit preview calibrate-mood", err)
		}
	} else {
		fmt.Print(string(b))
	}
}
