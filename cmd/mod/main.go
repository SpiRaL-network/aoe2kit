package main

import (
	"fmt"
	"os"

	"aoe2kit/pkg/jsonout"
	"aoe2kit/pkg/modpack"
)

func main() {
	if len(os.Args) < 3 {
		usage()
		os.Exit(2)
	}
	cmd, path := os.Args[1], os.Args[2]
	switch cmd {
	case "check", "inspect":
		report, err := modpack.Check(path)
		if err != nil {
			die(err)
		}
		if err := jsonout.Print(report); err != nil {
			die(err)
		}
		if !report.OK() {
			os.Exit(1)
		}
	default:
		usage()
		os.Exit(2)
	}
}

func die(err error) {
	fmt.Fprintf(os.Stderr, "mod: %v\n", err)
	os.Exit(1)
}

func usage() {
	fmt.Fprintf(os.Stderr, `Usage:
  mod check   <moddir>
  mod inspect <moddir>
`)
}
