package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"aoe2kit/pkg/datfile"
	"aoe2kit/pkg/gameinstall"
	"aoe2kit/pkg/preview"
	"aoe2kit/pkg/scenario"
)

type corpusArgs struct {
	inputs             []string
	against, dat, game string
	opts               preview.StudyOptions
	minMaps            int
	verbose, text      bool
}

func parseCorpusArgs(args []string) (corpusArgs, error) {
	a := corpusArgs{minMaps: 1}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--gaia", "--gaia=true":
			a.opts.GaiaOnly = true
		case "--gaia=false":
			a.opts.GaiaOnly = false
		case "--json":
			a.text = false
		case "--text":
			a.text = true
		case "--verbose":
			a.verbose = true
		case "--against", "--dat", "--game-dir", "--min-maps", "--player", "--crop":
			flag := args[i]
			i++
			if i >= len(args) {
				return a, fmt.Errorf("%s needs a value", flag)
			}
			v := args[i]
			switch flag {
			case "--against":
				a.against = v
			case "--dat":
				a.dat = v
			case "--game-dir":
				a.game = v
			case "--min-maps":
				n, e := strconv.Atoi(v)
				if e != nil || n < 1 {
					return a, fmt.Errorf("min-maps must be positive")
				}
				a.minMaps = n
			case "--player":
				n, e := strconv.Atoi(v)
				if e != nil || n < 0 || n > 16 {
					return a, fmt.Errorf("player must be 0..16 (0 is Gaia)")
				}
				a.opts.Player = &n
			case "--crop":
				p, e := parseIntList(v, "--crop")
				if e != nil || len(p) != 4 {
					return a, fmt.Errorf("crop must be x1,y1,x2,y2")
				}
				a.opts.Crop = &preview.Crop{X1: p[0], Y1: p[1], X2: p[2], Y2: p[3]}
			}
		default:
			if strings.HasPrefix(args[i], "--") {
				return a, fmt.Errorf("unknown flag %q", args[i])
			}
			a.inputs = append(a.inputs, args[i])
		}
	}
	if len(a.inputs) == 0 {
		return a, fmt.Errorf("census needs a directory or scenario files")
	}
	if a.opts.GaiaOnly && a.opts.Player != nil && *a.opts.Player != 0 {
		return a, fmt.Errorf("--gaia conflicts with --player other than 0")
	}
	return a, nil
}

func readArtworkForCensus(path string, measure bool) (*scenario.Artwork, int64, error) {
	if !measure {
		a, e := scenario.OpenArtwork(path)
		return a, 0, e
	}
	self, err := os.Executable()
	if err != nil {
		return nil, 0, err
	}
	c := exec.Command(self, "scen", "units", path, "--peak-rss")
	var stderr bytes.Buffer
	c.Stderr = &stderr
	pipe, err := c.StdoutPipe()
	if err != nil {
		return nil, 0, err
	}
	if err = c.Start(); err != nil {
		return nil, 0, err
	}
	var row struct {
		scenario.UnitInfo
		Version string `json:"version"`
		Width   int    `json:"width"`
		Height  int    `json:"height"`
	}
	err = json.NewDecoder(pipe).Decode(&row)
	if err != nil {
		_ = c.Process.Kill()
		_ = c.Wait()
		return nil, 0, fmt.Errorf("%s: %w %s", path, err, stderr.String())
	}
	if err = c.Wait(); err != nil {
		return nil, 0, fmt.Errorf("%s: %w %s", path, err, stderr.String())
	}
	var stats struct {
		PeakRSSKiB int64 `json:"peak_rss_kib"`
	}
	if err := json.Unmarshal(stderr.Bytes(), &stats); err != nil {
		return nil, 0, fmt.Errorf("invalid reader memory report: %w", err)
	}
	return &scenario.Artwork{Version: row.Version, Width: row.Width, Height: row.Height, Units: row.UnitInfo}, stats.PeakRSSKiB, nil
}

func runCorpusCensus(args []string) {
	a, err := parseCorpusArgs(args)
	if err != nil {
		die("kit scen census", err)
	}
	paths, err := preview.ScenarioPaths(a.inputs)
	if err != nil {
		die("kit scen census", err)
	}
	var comparison []string
	if a.against != "" {
		comparison, err = preview.ScenarioPaths([]string{a.against})
		if err != nil {
			die("kit scen census", err)
		}
	}
	game, e := gameinstall.Resolve(a.game)
	if e != nil && (a.game != "" || os.Getenv(gameinstall.EnvDir) != "") {
		die("kit scen census", e)
	}
	if a.dat == "" && e == nil {
		candidate := gameinstall.DataPath(game)
		if _, e := os.Stat(candidate); e == nil {
			a.dat = candidate
		}
	}
	var idx *datfile.Index
	if a.dat != "" {
		idx, err = datfile.Open(a.dat)
		if err != nil {
			die("kit scen census", err)
		}
	}
	read := func(files []string) *preview.CorpusCounter {
		c := preview.NewCorpusCounter(idx, a.opts)
		for _, p := range files {
			art, peak, err := readArtworkForCensus(p, a.verbose)
			if err != nil {
				die("kit scen census", err)
			}
			if err := c.Add(p, art, peak); err != nil {
				die("kit scen census", err)
			}
			if a.verbose {
				if peak > 0 {
					fmt.Fprintf(os.Stderr, "%s: peak RSS %d KiB (reader post-exec VmHWM)\n", p, peak)
				} else {
					fmt.Fprintf(os.Stderr, "%s: peak RSS unavailable on this platform\n", p)
				}
			}
		}
		return c
	}
	c := read(paths)
	var other *preview.CorpusCounter
	if comparison != nil {
		other = read(comparison)
	}
	r, err := c.Report(other, a.minMaps)
	if err != nil {
		die("kit scen census", err)
	}
	if a.text {
		fmt.Printf("scenarios=%d matched_units=%d returned_types=%d comparison_scenarios=%d\n", len(r.Files), r.Total, r.Returned, len(r.AgainstFiles))
		for _, u := range r.Units {
			fmt.Printf("unit=%d name=%q count=%d maps=%d against_count=%d against_maps=%d only_in_input=%t\n", u.UnitID, u.Name, u.Count, u.Maps, u.AgainstCount, u.AgainstMaps, u.OnlyInInput)
		}
	} else {
		printJSON(r)
	}
}
