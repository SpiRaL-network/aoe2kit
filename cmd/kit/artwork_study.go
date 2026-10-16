package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"aoe2kit/pkg/datfile"
	"aoe2kit/pkg/gameinstall"
	"aoe2kit/pkg/preview"
	"aoe2kit/pkg/scenario"
)

type artworkStudyArgs struct {
	input, dat, game string
	options          preview.StudyOptions
	paths, missing   bool
}

func parseArtworkStudyArgs(kind string, args []string) (artworkStudyArgs, error) {
	a := artworkStudyArgs{}
	if len(args) == 0 {
		return a, fmt.Errorf("usage: kit %s <scenario> --dat FILE [--game-dir DIR] [--crop x1,y1,x2,y2]", kind)
	}
	a.input = args[0]
	f := flag.NewFlagSet(kind, flag.ContinueOnError)
	f.SetOutput(io.Discard)
	f.StringVar(&a.dat, "dat", "", "DAT file")
	f.StringVar(&a.game, "game-dir", "", "installation or local mirror")
	var crop string
	f.StringVar(&crop, "crop", "", "half-open crop x1,y1,x2,y2")
	if kind == "assets" {
		f.BoolVar(&a.paths, "paths", false, "print one relative asset path per line")
		f.BoolVar(&a.missing, "missing-only", false, "only missing assets (requires an installation/mirror)")
	} else {
		f.BoolVar(&a.options.GaiaOnly, "gaia", kind == "hotspots", "only Gaia units")
		if kind == "hotspots" {
			f.IntVar(&a.options.Cell, "cell", 8, "cell side length in tiles")
			f.IntVar(&a.options.Top, "top", 20, "maximum cells")
			f.StringVar(&a.options.Rank, "rank", "variety", "variety or count")
			f.BoolVar(&a.options.IncludeForestFish, "include-forest-fish", false, "include tree/fish classes")
		}
	}
	if err := f.Parse(args[1:]); err != nil {
		return a, err
	}
	if f.NArg() != 0 {
		return a, fmt.Errorf("unexpected arguments: %v", f.Args())
	}
	if kind == "hotspots" && (a.options.Cell < 1 || a.options.Top < 1) {
		return a, fmt.Errorf("--cell and --top must be positive")
	}
	if crop != "" {
		parts := strings.Split(crop, ",")
		if len(parts) != 4 {
			return a, fmt.Errorf("crop needs x1,y1,x2,y2")
		}
		v := [4]int{}
		for i, p := range parts {
			n, e := strconv.Atoi(strings.TrimSpace(p))
			if e != nil {
				return a, e
			}
			v[i] = n
		}
		a.options.Crop = &preview.Crop{X1: v[0], Y1: v[1], X2: v[2], Y2: v[3]}
	}
	return a, nil
}

func runArtworkStudy(kind string, args []string) {
	a, err := parseArtworkStudyArgs(kind, args)
	if err != nil {
		die("kit artwork "+kind, err)
	}
	game, resolveErr := gameinstall.Resolve(a.game)
	if resolveErr != nil {
		if a.game != "" || os.Getenv(gameinstall.EnvDir) != "" {
			die("kit artwork "+kind, resolveErr)
		}
		game = ""
	}
	if a.dat == "" && game != "" {
		a.dat = gameinstall.DataPath(game)
	}
	if a.dat == "" {
		die("kit artwork "+kind, fmt.Errorf("--dat or a discovered --game-dir is required for names and graphics"))
	}
	if a.missing && game == "" {
		die("kit artwork "+kind, fmt.Errorf("--missing-only requires --game-dir or a discovered install/mirror"))
	}
	idx, err := datfile.Open(a.dat)
	if err != nil {
		die("kit artwork "+kind, err)
	}
	art, err := scenario.OpenArtwork(a.input)
	if err != nil {
		die("kit artwork "+kind, err)
	}
	switch kind {
	case "census":
		r, e := preview.ArtworkCensus(art, idx, a.options)
		if e != nil {
			die("kit scen census", e)
		}
		printJSON(r)
	case "hotspots":
		r, e := preview.ArtworkHotspots(art, idx, a.options)
		if e != nil {
			die("kit scen hotspots", e)
		}
		printJSON(r)
	case "assets":
		r, e := preview.ArtworkAssets(art, idx, game, a.options.Crop)
		if e != nil {
			die("kit preview assets", e)
		}
		if a.missing {
			assets := []preview.NeededAsset{}
			for _, asset := range r.Assets {
				if asset.Status == "missing" {
					assets = append(assets, asset)
				}
			}
			r.Assets = assets
		}
		if a.paths {
			for _, asset := range r.Assets {
				fmt.Println(asset.Path)
			}
			if !r.Complete {
				fmt.Fprintln(os.Stderr, "asset dependency list incomplete; use JSON output for unresolved references, fetch definitions and rerun")
				os.Exit(1)
			}
		} else {
			printJSON(r)
		}
	}
}
