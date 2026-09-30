package main

import (
	"fmt"
	"os"
	"strconv"

	"aoe2kit/pkg/aoe2"
	"aoe2kit/pkg/jsonout"
	"aoe2kit/pkg/scenario"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	cmd := os.Args[1]
	if cmd == "smoke-recipe" {
		opts, err := parseSmokeOptions(os.Args[2:])
		if err != nil {
			die(err)
		}
		printJSON(scenario.SmokeRecipe(opts))
		return
	}
	if cmd == "smoke" {
		if len(os.Args) < 4 {
			usage()
			os.Exit(2)
		}
		opts, err := parseSmokeOptions(os.Args[4:])
		if err != nil {
			die(err)
		}
		report, err := scenario.PatchSmokeRecipeFile(os.Args[2], os.Args[3], opts)
		if err != nil {
			die(err)
		}
		printJSON(report)
		return
	}
	if len(os.Args) < 3 {
		usage()
		os.Exit(2)
	}
	path := os.Args[2]
	switch cmd {
	case "info", "inspect":
		info, err := inspectScenarioArg(path)
		if err != nil {
			die(err)
		}
		printJSON(info)
	case "check":
		info, err := inspectScenarioArg(path)
		if err != nil {
			die(err)
		}
		fmt.Printf("%s: %d bytes sha256=%s [OK]\n", info.Base, info.SizeBytes, info.SHA256)
	case "triggers":
		scen, err := scenario.Open(path)
		if err != nil {
			die(err)
		}
		printJSON(scen.Triggers)
	case "units":
		scen, err := scenario.Open(path)
		if err != nil {
			die(err)
		}
		printJSON(scen.Units)
	case "map":
		scen, err := scenario.Open(path)
		if err != nil {
			die(err)
		}
		printJSON(scen.Map)
	case "verify":
		scen, err := scenario.Open(path)
		if err != nil {
			die(err)
		}
		if err := scen.VerifyRebuild(); err != nil {
			die(err)
		}
		printJSON(struct {
			Path          string                 `json:"path"`
			Version       string                 `json:"version"`
			HeaderBytes   int                    `json:"header_bytes"`
			InflatedBytes int                    `json:"inflated_body_bytes"`
			Sections      []scenario.SectionInfo `json:"sections"`
			Triggers      *scenario.TriggerInfo  `json:"triggers"`
			RebuildOK     bool                   `json:"rebuild_ok"`
		}{
			Path:          path,
			Version:       scen.Version,
			HeaderBytes:   scen.HeaderBytes,
			InflatedBytes: scen.InflatedBytes,
			Sections:      scen.Sections,
			Triggers:      scen.Triggers,
			RebuildOK:     true,
		})
	case "plan":
		if len(os.Args) != 5 || os.Args[3] != "--recipe" {
			usage()
			os.Exit(2)
		}
		recipe, err := scenario.LoadRecipe(os.Args[4])
		if err != nil {
			die(err)
		}
		plan, err := scenario.PlanRecipeFile(path, recipe)
		if err != nil {
			die(err)
		}
		printJSON(plan)
	case "patch":
		if len(os.Args) != 6 || os.Args[4] != "--recipe" {
			usage()
			os.Exit(2)
		}
		recipe, err := scenario.LoadRecipe(os.Args[5])
		if err != nil {
			die(err)
		}
		report, err := scenario.PatchRecipeFile(path, os.Args[3], recipe)
		if err != nil {
			die(err)
		}
		printJSON(report)
	default:
		usage()
		os.Exit(2)
	}
}

func inspectScenarioArg(path string) (aoe2.FileInfo, error) {
	info, err := aoe2.InspectFile(path)
	if err != nil {
		return aoe2.FileInfo{}, err
	}
	if info.Kind != "scenario" {
		return aoe2.FileInfo{}, fmt.Errorf("%s is %q, not scenario", path, info.Kind)
	}
	return info, nil
}

func printJSON(v any) {
	if err := jsonout.Print(v); err != nil {
		die(err)
	}
}

func die(err error) {
	fmt.Fprintf(os.Stderr, "scen: %v\n", err)
	os.Exit(1)
}

func parseSmokeOptions(args []string) (scenario.SmokeRecipeOptions, error) {
	opts := scenario.SmokeRecipeOptions{}
	for i := 0; i < len(args); i++ {
		if i+1 >= len(args) {
			return opts, fmt.Errorf("%s needs a value", args[i])
		}
		value, err := strconv.Atoi(args[i+1])
		if err != nil {
			return opts, fmt.Errorf("%s value %q: %w", args[i], args[i+1], err)
		}
		switch args[i] {
		case "--x":
			opts.X = value
			opts.XSet = true
		case "--y":
			opts.Y = value
			opts.YSet = true
		case "--player":
			opts.Player = value
			opts.PlayerSet = true
		case "--unit", "--unit-const":
			opts.UnitConst = value
			opts.UnitConstSet = true
		case "--terrain", "--terrain-id":
			opts.TerrainID = value
			opts.TerrainIDSet = true
		case "--elevation":
			opts.Elevation = value
			opts.ElevationSet = true
		case "--layer":
			opts.Layer = value
			opts.LayerSet = true
		default:
			return opts, fmt.Errorf("unknown smoke option %q", args[i])
		}
		i++
	}
	return opts, nil
}

func usage() {
	fmt.Fprintf(os.Stderr, `Usage:
  scen info  <file.aoe2scenario>
  scen check <file.aoe2scenario>
  scen triggers <file.aoe2scenario>
  scen units <file.aoe2scenario>
  scen map <file.aoe2scenario>
  scen verify <file.aoe2scenario>
  scen plan <in.aoe2scenario> --recipe recipe.json
  scen patch <in.aoe2scenario> <out.aoe2scenario> --recipe recipe.json
  scen smoke-recipe [--x N] [--y N] [--player N] [--unit N] [--terrain N] [--elevation N] [--layer N]
  scen smoke <in.aoe2scenario> <out.aoe2scenario> [--x N] [--y N] [--player N] [--unit N] [--terrain N] [--elevation N] [--layer N]
`)
}
