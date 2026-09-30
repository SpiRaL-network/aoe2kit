package main

import (
	"fmt"
	"os"

	"aoe2kit/pkg/geom"
	"aoe2kit/pkg/jsonout"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "patterns":
		runPatterns(os.Args[2:])
	default:
		usage()
		os.Exit(2)
	}
}

func runPatterns(args []string) {
	if len(args) < 1 {
		patternsSummary()
		return
	}

	switch args[0] {
	case "list":
		for _, name := range geom.PatternNames() {
			fmt.Println(name)
		}
	case "summary":
		patternsSummary()
	case "validate":
		if !patternsValidate() {
			os.Exit(1)
		}
	case "json":
		if len(args) != 2 {
			fmt.Fprintln(os.Stderr, "usage: swatch patterns json <name>")
			os.Exit(2)
		}
		if err := patternJSON(args[1]); err != nil {
			fmt.Fprintf(os.Stderr, "swatch patterns json: %v\n", err)
			os.Exit(1)
		}
	default:
		usage()
		os.Exit(2)
	}
}

func patternsSummary() {
	for _, name := range geom.PatternNames() {
		frames, err := geom.Generate(name)
		if err != nil {
			fmt.Printf("%s: ERROR %v\n", name, err)
			continue
		}
		validation := geom.Validate(frames)
		status := "OK"
		if len(validation.Errors) != 0 {
			status = "BAD"
		}
		fmt.Printf("%s: %d frames, max %d tiles/frame [%s]\n", name, validation.Frames, validation.MaxTiles, status)
	}
}

func patternsValidate() bool {
	ok := true
	for _, name := range geom.PatternNames() {
		frames, err := geom.Generate(name)
		if err != nil {
			fmt.Printf("%s: ERROR %v\n", name, err)
			ok = false
			continue
		}
		validation := geom.Validate(frames)
		if len(validation.Errors) == 0 {
			fmt.Printf("%s: %d frames, max %d tiles/frame [OK]\n", name, validation.Frames, validation.MaxTiles)
			continue
		}
		ok = false
		fmt.Printf("%s: %d frames, max %d tiles/frame [BAD]\n", name, validation.Frames, validation.MaxTiles)
		for _, err := range validation.Errors {
			fmt.Printf("  - %s\n", err)
		}
	}
	return ok
}

func patternJSON(name string) error {
	frames, err := geom.Generate(name)
	if err != nil {
		return err
	}
	validation := geom.Validate(frames)
	if len(validation.Errors) != 0 {
		return fmt.Errorf("%s failed validation: %v", name, validation.Errors)
	}

	payload := struct {
		Name   string       `json:"name"`
		Frames []geom.Frame `json:"frames"`
	}{
		Name:   name,
		Frames: frames,
	}

	return jsonout.Print(payload)
}

func usage() {
	fmt.Fprintf(os.Stderr, `Usage:
  swatch patterns
  swatch patterns summary
  swatch patterns list
  swatch patterns validate
  swatch patterns json <name>
`)
}
