package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"aoe2kit/pkg/felt"
	"aoe2kit/pkg/scenario"
)

func runScenarioExtract(args []string) {
	if len(args) == 0 {
		die("kit scen extract", fmt.Errorf("scenario input required"))
	}
	f := flag.NewFlagSet("scen extract", flag.ContinueOnError)
	crop := f.String("crop", "", "half-open x1,y1,x2,y2")
	out := f.String("out", "", "recipe JSON output")
	terrain := f.Bool("terrain", false, "include terrain, layers and elevation")
	relative := f.Bool("relative", false, "coordinates relative to crop origin")
	if err := f.Parse(args[1:]); err != nil {
		die("kit scen extract", err)
	}
	if *out == "" || *crop == "" || f.NArg() != 0 {
		die("kit scen extract", fmt.Errorf("use --crop x1,y1,x2,y2 --out recipe.json [--terrain] [--relative]"))
	}
	if in, err := os.Stat(args[0]); err == nil {
		if outInfo, err := os.Stat(*out); err == nil && os.SameFile(in, outInfo) {
			die("kit scen extract", fmt.Errorf("output must not overwrite the source scenario"))
		}
	}
	values, err := parseIntList(*crop, "--crop")
	if err != nil || len(values) != 4 {
		die("kit scen extract", fmt.Errorf("crop must be x1,y1,x2,y2"))
	}
	r, err := scenario.ExtractRecipeFile(args[0], scenario.ExtractOptions{Crop: scenario.MotifCrop{X1: values[0], Y1: values[1], X2: values[2], Y2: values[3]}, Terrain: *terrain, Relative: *relative})
	if err != nil {
		die("kit scen extract", err)
	}
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		die("kit scen extract", err)
	}
	if err := os.WriteFile(*out, append(b, '\n'), 0644); err != nil {
		die("kit scen extract", err)
	}
	printJSON(struct {
		Recipe  string                 `json:"recipe"`
		Units   int                    `json:"units"`
		Terrain bool                   `json:"terrain"`
		Source  *scenario.RecipeSource `json:"source"`
	}{*out, len(r.Units), *terrain, r.Source})
}

type motifPatchOptions struct {
	At      *scenario.PointRecipe
	Rotate  int
	Grammar string
}

func applyMotifOptions(r scenario.Recipe, o motifPatchOptions) (scenario.Recipe, error) {
	if o.At == nil && o.Rotate == 0 {
		if o.Grammar != "" {
			return scenario.Recipe{}, fmt.Errorf("--grammar requires --at or a nonzero --rotate")
		}
		return r, nil
	}
	g, err := felt.Load(o.Grammar)
	if err != nil {
		return scenario.Recipe{}, err
	}
	return scenario.TransformMotif(r, scenario.MotifTransform{At: o.At, Rotate: o.Rotate, Grammar: g})
}

// Extract motif options while leaving normal patch flags to their existing
// parser, so ordinary recipes retain their established CLI behavior.
func parseMotifPatchOptions(args []string) (motifPatchOptions, []string, error) {
	o := motifPatchOptions{}
	var remaining []string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--at", "--rotate", "--grammar":
			name := args[i]
			i++
			if i >= len(args) {
				return o, nil, fmt.Errorf("%s needs a value", name)
			}
			switch name {
			case "--at":
				parts := strings.Split(args[i], ",")
				if len(parts) != 2 {
					return o, nil, fmt.Errorf("at must be x,y")
				}
				x, e1 := strconv.ParseFloat(parts[0], 64)
				y, e2 := strconv.ParseFloat(parts[1], 64)
				if e1 != nil || e2 != nil {
					return o, nil, fmt.Errorf("at must be numeric x,y")
				}
				o.At = &scenario.PointRecipe{X: x, Y: y}
			case "--rotate":
				n, e := strconv.Atoi(args[i])
				if e != nil || (n != 0 && n != 90 && n != 180 && n != 270) {
					return o, nil, fmt.Errorf("rotate must be 0,90,180,270")
				}
				o.Rotate = n
			case "--grammar":
				o.Grammar = args[i]
			}
		default:
			remaining = append(remaining, args[i])
		}
	}
	return o, remaining, nil
}
