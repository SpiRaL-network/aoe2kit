package main

import (
	"fmt"
	"os"

	"aoe2kit/pkg/campaign"
)

func runCampaign(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: kit campaign <generate> ...")
		os.Exit(2)
	}
	switch args[0] {
	case "generate":
		runCampaignGenerate(args[1:])
	case "extract":
		runCampaignExtract(args[1:])
	default:
		fmt.Fprintln(os.Stderr, "usage: kit campaign <generate|extract> ...")
		os.Exit(2)
	}
}

func runCampaignGenerate(args []string) {
	input := ""
	outDir := "campaign-out"
	prefix := "A2KCampaign"
	textOut := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--out-dir":
			i++
			if i >= len(args) {
				die("kit campaign generate", fmt.Errorf("--out-dir needs a directory"))
			}
			outDir = args[i]
		case "--prefix":
			i++
			if i >= len(args) {
				die("kit campaign generate", fmt.Errorf("--prefix needs an XS identifier prefix"))
			}
			prefix = args[i]
		case "--text":
			textOut = true
		default:
			if input != "" {
				die("kit campaign generate", fmt.Errorf("unexpected argument %q", args[i]))
			}
			input = args[i]
		}
	}
	if input == "" {
		fmt.Fprintln(os.Stderr, "usage: kit campaign generate <campaign.json> [--out-dir DIR] [--prefix A2KCampaign] [--text]")
		os.Exit(2)
	}
	report, err := campaign.GenerateFile(input, campaign.GenerateOptions{OutDir: outDir, Prefix: prefix})
	if err != nil {
		die("kit campaign generate", err)
	}
	if textOut {
		printCampaignGenerate(report)
	} else {
		printJSON(report)
	}
	if !report.OK {
		os.Exit(1)
	}
}

func runCampaignExtract(args []string) {
	input := ""
	opts := campaign.ExtractOptions{}
	textOut := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--out-dir":
			i++
			if i >= len(args) {
				die("kit campaign extract", fmt.Errorf("--out-dir needs a directory"))
			}
			opts.OutDir = args[i]
		case "--scenario":
			value, err := parseNextInt(args, &i, "--scenario")
			if err != nil {
				die("kit campaign extract", err)
			}
			opts.Scenario = value
		case "--name":
			i++
			if i >= len(args) {
				die("kit campaign extract", fmt.Errorf("--name needs a stem"))
			}
			opts.Name = args[i]
		case "--text":
			textOut = true
		case "--json":
			textOut = false
		default:
			if input != "" {
				die("kit campaign extract", fmt.Errorf("unexpected argument %q", args[i]))
			}
			input = args[i]
		}
	}
	if input == "" {
		fmt.Fprintln(os.Stderr, "usage: kit campaign extract <campaign.aoe2campaign> [--scenario N | --name STEM] [--out-dir DIR] [--text|--json]")
		os.Exit(2)
	}
	report, err := campaign.ExtractFile(input, opts)
	if err != nil {
		die("kit campaign extract", err)
	}
	if textOut {
		printCampaignExtract(report)
	} else {
		printJSON(report)
	}
	if !report.OK {
		os.Exit(1)
	}
}

func printCampaignGenerate(report *campaign.GenerateReport) {
	fmt.Printf("campaign: %s\n", report.Name)
	fmt.Printf("writer_scenario: %s\n", report.WriterScenario)
	fmt.Printf("out_dir: %s\n", report.OutDir)
	fmt.Printf("ok: %t files=%d\n", report.OK, len(report.Files))
	for _, path := range report.Files {
		fmt.Printf("- %s\n", path)
	}
	for _, warning := range report.Warnings {
		fmt.Printf("warning: %s\n", warning)
	}
}

func printCampaignExtract(report *campaign.ExtractReport) {
	fmt.Printf("campaign: %s\n", report.Path)
	fmt.Printf("version: %s directory_at=%d entries=%d\n", report.Version, report.DirectoryAt, report.DirectoryCount)
	fmt.Printf("out_dir: %s\n", report.OutDir)
	for _, entry := range report.Entries {
		if !entry.Selected {
			continue
		}
		status := "full_parse_ok"
		if !entry.FullParseOK {
			status = "full_parse_failed"
		}
		fmt.Printf("[%d] %s offset=%d size=%d sha256=%s %s\n", entry.Index, entry.OutPath, entry.Offset, entry.Size, entry.SHA256, status)
		if entry.ParseError != "" {
			fmt.Printf("  parse_error: %s\n", entry.ParseError)
		}
	}
	for _, warning := range report.Warnings {
		fmt.Printf("warning: %s\n", warning)
	}
}
