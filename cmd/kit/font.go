package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"aoe2kit/pkg/glyph"
)

func runFont(args []string) {
	if len(args) < 1 || args[0] != "glyphs" {
		die("kit font", fmt.Errorf("usage: kit font glyphs <font.ttf> [--chars TEXT] [--material N] [--rotation DEGREES] [--out FILE]"))
	}
	if len(args) < 2 {
		die("kit font glyphs", fmt.Errorf("font path is required"))
	}
	path, out, chars := args[1], "", ""
	opts := glyph.DefaultOptions()
	for i := 2; i < len(args); i++ {
		switch args[i] {
		case "--chars":
			i++
			if i >= len(args) {
				die("kit font glyphs", fmt.Errorf("--chars needs text"))
			}
			chars = args[i]
		case "--material":
			i++
			if i >= len(args) {
				die("kit font glyphs", fmt.Errorf("--material needs integer"))
			}
			v, e := strconv.Atoi(args[i])
			if e != nil {
				die("kit font glyphs", e)
			}
			opts.Material = v
		case "--rotation":
			i++
			if i >= len(args) {
				die("kit font glyphs", fmt.Errorf("--rotation needs degrees"))
			}
			v, e := strconv.ParseFloat(args[i], 64)
			if e != nil {
				die("kit font glyphs", e)
			}
			opts.Rotation = v
		case "--out":
			i++
			if i >= len(args) {
				die("kit font glyphs", fmt.Errorf("--out needs file"))
			}
			out = args[i]
		default:
			die("kit font glyphs", fmt.Errorf("unexpected argument %q", args[i]))
		}
	}
	if chars != "" {
		opts.Chars = []rune(chars)
	}
	library, err := glyph.FromTTF(path, opts)
	if err != nil {
		die("kit font glyphs", err)
	}
	var data []byte
	data, err = json.MarshalIndent(library, "", "  ")
	if err != nil {
		die("kit font glyphs", err)
	}
	data = append(data, '\n')
	if out != "" {
		if err := os.WriteFile(out, data, 0644); err != nil {
			die("kit font glyphs", err)
		}
	} else {
		_, _ = os.Stdout.Write(data)
	}
}
