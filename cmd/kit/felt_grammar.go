package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"

	"aoe2kit/pkg/datfile"
	"aoe2kit/pkg/gameinstall"
	"aoe2kit/pkg/preview"
)

func runFelt(args []string) {
	if len(args) == 0 || args[0] != "grammar" {
		die("kit felt", fmt.Errorf("usage: kit felt grammar <unit-id> --out sheet.png --json grammar.json [--game-dir DIR] [--dat FILE] [--compendium FILE]"))
	}
	runFeltGrammar(args[1:])
}

func runFeltGrammar(args []string) {
	if len(args) == 0 {
		die("kit felt grammar", fmt.Errorf("unit id is required"))
	}
	id, err := strconv.Atoi(args[0])
	if err != nil || id < 0 {
		die("kit felt grammar", fmt.Errorf("invalid unit id"))
	}
	f := flag.NewFlagSet("felt grammar", flag.ContinueOnError)
	out := f.String("out", "", "local PNG output")
	report := f.String("json", "", "editable grammar JSON output")
	game := f.String("game-dir", "", "installed assets")
	dat := f.String("dat", "", "DAT file")
	compendium := f.String("compendium", "", "reviewed felt compendium with grammar")
	if err := f.Parse(args[1:]); err != nil {
		die("kit felt grammar", err)
	}
	if *out == "" || *report == "" || *out == *report || f.NArg() != 0 {
		die("kit felt grammar", fmt.Errorf("use --out sheet.png --json grammar.json (distinct files)"))
	}
	resolved, err := gameinstall.Resolve(*game)
	if err != nil {
		die("kit felt grammar", err)
	}
	if *dat == "" {
		*dat = gameinstall.DataPath(resolved)
	}
	idx, err := datfile.Open(*dat)
	if err != nil {
		die("kit felt grammar", err)
	}
	img, r, err := preview.RenderGrammar(preview.GrammarOptions{UnitID: id, Dat: idx, GameDir: resolved, CompendiumPath: *compendium})
	if err != nil {
		die("kit felt grammar", err)
	}
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		die("kit felt grammar", err)
	}
	if err := preview.WritePNG(*out, img); err != nil {
		die("kit felt grammar", err)
	}
	if err := os.WriteFile(*report, append(b, '\n'), 0644); err != nil {
		die("kit felt grammar", err)
	}
	printJSON(r)
}
