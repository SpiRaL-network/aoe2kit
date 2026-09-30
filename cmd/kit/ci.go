package main

import (
	"fmt"
	"os"

	"aoe2kit/pkg/ci"
)

func runCI(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: kit ci <init|check> ...")
		os.Exit(2)
	}
	switch args[0] {
	case "init":
		runCIInit(args[1:])
	case "check":
		runCICheck(args[1:])
	default:
		fmt.Fprintln(os.Stderr, "usage: kit ci <init|check> ...")
		os.Exit(2)
	}
}

func runCIInit(args []string) {
	root := "."
	scenario := ""
	overwrite := false
	textOut := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--root":
			i++
			if i >= len(args) {
				die("kit ci init", fmt.Errorf("--root needs a directory"))
			}
			root = args[i]
		case "--scenario":
			i++
			if i >= len(args) {
				die("kit ci init", fmt.Errorf("--scenario needs a path"))
			}
			scenario = args[i]
		case "--overwrite":
			overwrite = true
		case "--text":
			textOut = true
		default:
			if scenario == "" {
				scenario = args[i]
			} else {
				die("kit ci init", fmt.Errorf("unknown option %q", args[i]))
			}
		}
	}
	report, err := ci.Init(root, scenario, overwrite)
	if err != nil {
		die("kit ci init", err)
	}
	if textOut {
		printCIInit(report)
	} else {
		printJSON(report)
	}
}

func runCICheck(args []string) {
	config := ci.DefaultConfigName
	textOut := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--config":
			i++
			if i >= len(args) {
				die("kit ci check", fmt.Errorf("--config needs a path"))
			}
			config = args[i]
		case "--text":
			textOut = true
		default:
			if config == ci.DefaultConfigName {
				config = args[i]
			} else {
				die("kit ci check", fmt.Errorf("unknown option %q", args[i]))
			}
		}
	}
	report, err := ci.CheckFile(config)
	if err != nil {
		die("kit ci check", err)
	}
	if textOut {
		printCICheck(report)
	} else {
		printJSON(report)
	}
	if !report.OK {
		os.Exit(1)
	}
}

func printCIInit(report *ci.InitReport) {
	fmt.Printf("root: %s\n", report.Root)
	fmt.Printf("config: %s\n", report.ConfigPath)
	fmt.Printf("debug_xs: %s\n", report.DebugXS)
	if len(report.Created) > 0 {
		fmt.Println("created:")
		for _, path := range report.Created {
			fmt.Printf("- %s\n", path)
		}
	}
	if len(report.Updated) > 0 {
		fmt.Println("updated:")
		for _, path := range report.Updated {
			fmt.Printf("- %s\n", path)
		}
	}
}

func printCICheck(report *ci.Report) {
	fmt.Printf("ci: %s\n", report.Path)
	if report.Name != "" {
		fmt.Printf("name: %s\n", report.Name)
	}
	fmt.Printf("ok: %t passed=%d failed=%d unknown=%d\n", report.OK, report.Summary.Passed, report.Summary.Failed, report.Summary.Unknown)
	for _, check := range report.Checks {
		fmt.Printf("- [%s] %s type=%s", check.Status, check.Name, check.Type)
		if check.Message != "" {
			fmt.Printf(" - %s", check.Message)
		}
		fmt.Println()
	}
	if len(report.Warning) > 0 {
		fmt.Println("warnings:")
		for _, warning := range report.Warning {
			fmt.Printf("- %s\n", warning)
		}
	}
}
