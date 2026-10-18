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
	text             bool
	fetch            bool
	from, to         string
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
	var units string
	player := -1
	f.StringVar(&crop, "crop", "", "half-open crop x1,y1,x2,y2")
	if kind == "assets" {
		f.BoolVar(&a.fetch, "fetch", false, "copy missing loose SLDs into a local mirror")
		f.StringVar(&a.from, "from", "", "source game installation")
		f.StringVar(&a.to, "to", "", "destination mirror")
		f.BoolVar(&a.paths, "paths", false, "print one relative asset path per line")
		f.BoolVar(&a.missing, "missing-only", false, "only missing assets (requires an installation/mirror)")
	} else {
		if kind == "facets" {
			f.StringVar(&units, "unit", "", "comma-separated unit IDs")
		}
		f.BoolVar(&a.options.GaiaOnly, "gaia", kind == "hotspots", "only Gaia units")
		f.IntVar(&player, "player", -1, "player slot (0 is Gaia)")
		f.BoolVar(&a.text, "text", false, "text output")
		f.Bool("json", false, "JSON output (default)")
		if kind == "hotspots" {
			f.IntVar(&a.options.Cell, "cell", 8, "cell side length in tiles")
			f.IntVar(&a.options.Top, "top", 20, "maximum cells")
			f.StringVar(&a.options.Rank, "rank", "variety", "variety or count")
			f.BoolVar(&a.options.IncludeForestFish, "include-forest-fish", false, "include tree/fish classes")
			f.StringVar(&units, "unit", "", "comma-separated unit IDs")
			f.BoolVar(&a.options.Diverse, "diverse", false, "rarity-weighted Gaia art diversity")
			f.IntVar(&a.options.MinCount, "min-count", 0, "minimum matching objects per cell")
		}
	}
	if err := f.Parse(args[1:]); err != nil {
		return a, err
	}
	if f.NArg() != 0 {
		return a, fmt.Errorf("unexpected arguments: %v", f.Args())
	}
	if a.fetch {
		if a.from == "" || a.to == "" || a.game != "" || a.paths {
			return a, fmt.Errorf("--fetch requires --from and --to, without --game-dir or --paths")
		}
	} else if a.from != "" || a.to != "" {
		return a, fmt.Errorf("--from and --to require --fetch")
	}
	if player >= 0 {
		if player > 16 {
			return a, fmt.Errorf("player must be 0..16")
		}
		a.options.Player = &player
		if player != 0 {
			a.options.GaiaOnly = false
		}
	}
	if units != "" {
		ids, e := parseIntList(units, "--unit")
		if e != nil {
			return a, e
		}
		for _, id := range ids {
			if id < 0 {
				return a, fmt.Errorf("unit IDs must be nonnegative")
			}
		}
		a.options.UnitIDs = ids
	}
	if a.options.MinCount < 0 {
		return a, fmt.Errorf("min-count cannot be negative")
	}
	if a.options.Diverse {
		if (a.options.Player != nil && *a.options.Player != 0) || !a.options.GaiaOnly || len(a.options.UnitIDs) > 0 {
			return a, fmt.Errorf("--diverse requires Gaia selection and cannot be combined with --unit")
		}
		set := false
		f.Visit(func(flag *flag.Flag) {
			if flag.Name == "cell" {
				set = true
			}
		})
		if !set {
			a.options.Cell = 16
		}
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
	if kind == "census" {
		runCorpusCensus(args)
		return
	}
	a, err := parseArtworkStudyArgs(kind, args)
	if err != nil {
		die("kit artwork "+kind, err)
	}
	if a.fetch {
		runPreviewAssetFetch(a)
		return
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
	case "facets":
		r, e := preview.ArtworkFacets(art, idx, a.options)
		if e != nil {
			die("kit scen facets", e)
		}
		if a.text {
			for _, u := range r.Units {
				fmt.Printf("unit=%d count=%d stored=%v available=%v placed=%v missing=%v unexpected=%v unknown=%q\n", u.UnitID, u.Count, u.Stored, u.Available, u.Placed, u.Missing, u.Unexpected, u.Unknown)
			}
		} else {
			printJSON(r)
		}
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
		if a.text {
			for _, h := range r.Hotspots {
				fmt.Printf("score=%.3f count=%d distinct=%d bounds=%d,%d,%d,%d %s\n", h.Score, h.Count, h.Variety, h.Bounds.X1, h.Bounds.Y1, h.Bounds.X2, h.Bounds.Y2, h.CropArg)
			}
		} else {
			printJSON(r)
		}
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
