package main

import (
	"crypto/sha256"
	"fmt"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"aoe2kit/pkg/datfile"
	"aoe2kit/pkg/gameinstall"
	"aoe2kit/pkg/preview"
	"aoe2kit/pkg/scenario"
)

func runPreview(args []string) {
	if len(args) > 0 && args[0] == "assets" {
		runArtworkStudy("assets", args[1:])
		return
	}
	if len(args) > 0 && args[0] == "calibrate-mood" {
		runPreviewCalibrateMood(args[1:])
		return
	}
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Project defaults: kit-preview.json; shortcuts: --around x,y --radius r | --fit-units ID[,ID]; --elevation experimental; --strict requires sprites; --rotation-model stored-index|radians|radians-reversed")
		fmt.Fprintln(os.Stderr, "usage: kit preview <scenario|replay|snapshot> [--out file.png] [--minimap] [--minimap-orientation normal|transpose|flipx|flipy|rotate90|rotate180|rotate270 (default rotate270)] [--snapshot-out file.json] [--projection topdown|diamond] [--scale N] [--layers LIST] [--crop x1,y1,x2,y2] [--dat FILE] [--palette FILE] [--game-dir DIR] [--sprites] [--shadows] [--shadow-overlap max|sum] [--textures] [--terrain-repeat TILES] [--texture-repeat TILES] [--no-terrain-edges] [--mood NAME|scenario|none]")
		os.Exit(2)
	}
	if args[0] == "snapshot" {
		runPreviewSnapshot(args[1:])
		return
	}
	input := args[0]
	cwd, err := os.Getwd()
	if err != nil {
		die("kit preview", err)
	}
	defaults, err := preview.LoadProjectDefaults(cwd)
	if err != nil {
		die("kit preview", err)
	}
	var outPath, snapshotPath, datPath, palettePath, gameDir, projection, minimapOrientation string
	datPath, gameDir = defaults.Dat, defaults.GameDir
	scale := 4
	if defaults.Scale != 0 {
		scale = defaults.Scale
	}
	spriteTilePixels := 96
	layers := map[string]bool{}
	var crop *preview.Crop
	sprites := defaults.Sprites
	shadows := defaults.Shadows
	elevation := defaults.Elevation
	strict := false
	rotationModel := ""
	var fitIDs []int
	var around []float64
	radius := 8.0
	radiusSet := false
	shadowOverlap := ""
	minimap := false
	moodFlag := "scenario"
	if defaults.Mood != "" {
		moodFlag = defaults.Mood
	}
	moodModelPath := ""
	scenarioMood := ""
	textures, terrainEdges := defaults.Textures, true
	terrainRepeat := 0.0 // 0 = package default (calibrated)
	textureRepeat := 0.0
	for i := 1; i < len(args); i++ {
		need := func() string {
			if i+1 >= len(args) {
				die("kit preview", fmt.Errorf("%s needs a value", args[i]))
			}
			i++
			return args[i]
		}
		switch args[i] {
		case "--out":
			outPath = need()
		case "--snapshot-out":
			snapshotPath = need()
		case "--dat":
			datPath = need()
		case "--palette":
			palettePath = need()
		case "--minimap-orientation":
			minimapOrientation = need()
		case "--game-dir":
			gameDir = need()
		case "--projection":
			projection = need()
		case "--scale":
			v, err := strconv.Atoi(need())
			if err != nil {
				die("kit preview", err)
			}
			scale = v
		case "--sprite-tile-width":
			v, err := strconv.Atoi(need())
			if err != nil {
				die("kit preview", err)
			}
			spriteTilePixels = v
		case "--layers":
			for _, name := range strings.Split(need(), ",") {
				name = strings.TrimSpace(name)
				if name != "terrain" && name != "elevation" && name != "objects" && name != "starts" {
					die("kit preview", fmt.Errorf("unknown layer %q", name))
				}
				layers[name] = true
			}
		case "--crop":
			parts := strings.Split(need(), ",")
			if len(parts) != 4 {
				die("kit preview", fmt.Errorf("crop must be x1,y1,x2,y2"))
			}
			vals := make([]int, 4)
			for j, p := range parts {
				v, err := strconv.Atoi(p)
				if err != nil {
					die("kit preview", err)
				}
				vals[j] = v
			}
			crop = &preview.Crop{X1: vals[0], Y1: vals[1], X2: vals[2], Y2: vals[3]}
		case "--sprites":
			sprites = true
		case "--strict":
			strict = true
		case "--rotation-model":
			rotationModel = need()
		case "--no-sprites":
			sprites = false
		case "--elevation":
			elevation = true
		case "--no-elevation":
			elevation = false
		case "--fit-units":
			for _, part := range strings.Split(need(), ",") {
				id, err := strconv.Atoi(part)
				if err != nil || id < 0 {
					die("kit preview", fmt.Errorf("fit-units needs nonnegative unit IDs"))
				}
				fitIDs = append(fitIDs, id)
			}
		case "--around":
			parts := strings.Split(need(), ",")
			if len(parts) != 2 {
				die("kit preview", fmt.Errorf("around must be x,y"))
			}
			around = nil
			for _, part := range parts {
				value, err := strconv.ParseFloat(part, 64)
				if err != nil {
					die("kit preview", err)
				}
				around = append(around, value)
			}
		case "--radius":
			value, err := strconv.ParseFloat(need(), 64)
			if err != nil {
				die("kit preview", err)
			}
			radius, radiusSet = value, true
		case "--shadows":
			shadows = true
		case "--no-shadows":
			shadows = false
		case "--shadow-overlap":
			shadowOverlap = need()
		case "--textures":
			textures = true
		case "--no-textures":
			textures = false
		case "--no-terrain-edges":
			terrainEdges = false
		case "--terrain-repeat":
			v, err := strconv.ParseFloat(need(), 64)
			if err != nil {
				die("kit preview", err)
			}
			terrainRepeat = v
		case "--texture-repeat":
			v, err := strconv.ParseFloat(need(), 64)
			if err != nil {
				die("kit preview", err)
			}
			textureRepeat = v
		case "--mood":
			moodFlag = need()
		case "--mood-model":
			moodModelPath = need()
		case "--minimap":
			minimap = true
		default:
			die("kit preview", fmt.Errorf("unknown option %q", args[i]))
		}
	}
	if outPath == "" {
		die("kit preview", fmt.Errorf("--out is required"))
	}
	if strict && (!sprites || minimap) {
		die("kit preview", fmt.Errorf("--strict requires --sprites, without --minimap"))
	}
	if rotationModel != "" {
		if !sprites || minimap {
			die("kit preview", fmt.Errorf("--rotation-model requires --sprites, without --minimap"))
		}
		if _, err := preview.ResolveRotationModel(rotationModel); err != nil {
			die("kit preview", err)
		}
	}
	if (crop != nil && (around != nil || len(fitIDs) > 0)) || (around != nil && len(fitIDs) > 0) || (radiusSet && around == nil) {
		die("kit preview", fmt.Errorf("choose one of --crop, --fit-units, --around; --radius requires --around"))
	}
	if minimap && (around != nil || len(fitIDs) > 0 || elevation) {
		die("kit preview", fmt.Errorf("minimap does not use crop shortcuts or experimental elevation"))
	}
	if shadows && (!sprites || minimap) {
		die("kit preview", fmt.Errorf("--shadows requires --sprites, without --minimap"))
	}
	if shadowOverlap != "" && (!shadows || (shadowOverlap != "max" && shadowOverlap != "sum")) {
		die("kit preview", fmt.Errorf("--shadow-overlap requires --shadows and max or sum"))
	}
	moodModel := preview.CurrentMoodModel()
	if moodModelPath != "" {
		if !textures || minimap || moodFlag == "none" {
			die("kit preview", fmt.Errorf("--mood-model requires textured, non-minimap rendering with mood enabled"))
		}
		var err error
		moodModel, err = preview.LoadMoodModel(moodModelPath)
		if err != nil {
			die("kit preview", err)
		}
	}
	if minimap && sprites {
		die("kit preview", fmt.Errorf("--minimap does not use sprites; omit --sprites"))
	}
	if minimap && textures {
		die("kit preview", fmt.Errorf("--minimap uses palette colors, not --textures"))
	}
	if !textures && (!terrainEdges || terrainRepeat != 0 || textureRepeat != 0) {
		die("kit preview", fmt.Errorf("terrain texture options require --textures"))
	}
	if minimap && crop != nil {
		die("kit preview", fmt.Errorf("--minimap always renders the whole map; omit --crop"))
	}
	var dat *datfile.Index
	resolvedDir, resolveErr := gameinstall.Resolve(gameDir)
	if gameDir != "" && resolveErr != nil {
		die("kit preview", resolveErr)
	}
	if datPath == "" && resolveErr == nil {
		candidate := gameinstall.DataPath(resolvedDir)
		if _, err := os.Stat(candidate); err == nil {
			datPath = candidate
		}
	}
	if datPath != "" {
		var err error
		dat, err = datfile.Open(datPath)
		if err != nil {
			die("kit preview", fmt.Errorf("load DAT: %w", err))
		}
	}
	if palettePath == "" && resolvedDir != "" {
		palettePath = gameinstall.MinimapPalettePath(resolvedDir)
	}
	var palette []color.RGBA
	if palettePath != "" {
		var err error
		palette, err = preview.ReadMinimapPalette(palettePath)
		if err != nil {
			die("kit preview", fmt.Errorf("load minimap palette: %w", err))
		}
	}
	var snap preview.Snapshot
	if strings.HasSuffix(strings.ToLower(input), ".json") {
		var err error
		snap, err = preview.ReadSnapshot(input)
		if err != nil {
			die("kit preview", err)
		}
	} else if strings.EqualFold(filepath.Ext(input), ".aoe2record") {
		var err error
		snap, err = preview.ReplaySnapshot(input, dat)
		if err != nil {
			die("kit preview", err)
		}
	} else {
		file, err := scenario.OpenArtwork(input)
		if err != nil {
			die("kit preview", err)
		}
		scenarioMood = file.ColorMood
		snap, err = preview.ArtworkSnapshot(file, "scenario", dat)
		if err != nil {
			die("kit preview", err)
		}
	}
	if snapshotPath != "" {
		if err := preview.WriteSnapshot(snapshotPath, snap); err != nil {
			die("kit preview", err)
		}
	}
	if around != nil {
		crop, err = preview.AroundCrop(snap, around[0], around[1], radius)
		if err != nil {
			die("kit preview", err)
		}
	}
	if len(fitIDs) > 0 {
		crop, err = preview.FitUnitCrop(snap, fitIDs)
		if err != nil {
			die("kit preview", err)
		}
	}
	previewOpts := preview.Options{Projection: projection, Scale: scale, Layers: layers, Crop: crop, MinimapOrientation: minimapOrientation, Palette: palette, Dat: dat, SpriteTilePixels: spriteTilePixels, Elevation: elevation, RotationModel: rotationModel}
	if textures {
		if resolveErr != nil {
			die("kit preview", fmt.Errorf("textured terrain needs a game install: %w", resolveErr))
		}
		previewOpts.Textures = &preview.TerrainTextureOptions{GameDir: resolvedDir, Repeat: terrainRepeat, TextureRepeat: textureRepeat, Edges: terrainEdges}
	}
	var renderMood *preview.Mood
	moodSource := "--mood"
	if !minimap && (textures || shadows) && moodFlag != "none" {
		name := moodFlag
		if moodFlag == "scenario" {
			name, moodSource = scenarioMood, "scenario map_color_mood"
		}
		m, err := preview.LoadMood(resolvedDir, name)
		if err != nil {
			die("kit preview", err)
		}
		renderMood = &m
	}
	if shadows {
		previewOpts.Shadows = &preview.ShadowOptions{Mood: renderMood, Overlap: shadowOverlap}
	}
	var img image.Image
	var report any
	err = nil
	if minimap {
		var r preview.Report
		img, r, err = preview.RenderMinimap(snap, dat, previewOpts)
		report = r
	} else if sprites {
		if resolveErr != nil {
			die("kit preview", fmt.Errorf("sprite preview needs a game install; %w", resolveErr))
		}
		if dat == nil {
			die("kit preview", fmt.Errorf("sprite preview needs a DAT; provide --dat or install the game data"))
		}
		var r preview.Report
		img, r, err = preview.RenderSprites(snap, previewOpts, dat, resolvedDir)
		report = r
	} else {
		var r preview.Report
		img, r, err = preview.RenderFlat(snap, previewOpts)
		report = r
	}
	if err != nil {
		die("kit preview", err)
	}
	if strict {
		if r, ok := report.(preview.Report); ok {
			if err := preview.CheckSpriteCompleteness(r); err != nil {
				r.OK = false
				printJSON(r)
				die("kit preview", err)
			}
		}
	}
	if rgba, ok := img.(*image.RGBA); ok && !minimap && textures && moodFlag != "none" {
		source := moodSource
		if moodModelPath != "" {
			source += "; explicit calibration model"
		}
		mr, err := preview.ApplyMoodModel(rgba, *renderMood, source, moodModel)
		if err != nil {
			die("kit preview", err)
		}
		if r, ok := report.(preview.Report); ok {
			if r.Shadows != nil {
				for i, item := range mr.NotModelled {
					if strings.Contains(item, "sprite shadows are not drawn yet") {
						mr.NotModelled[i] = "sprite shadow receiver metadata; terrain coverage pass is experimental"
					}
				}
			}
			r.Mood = &mr
			r.PixelSHA256 = fmt.Sprintf("%x", sha256.Sum256(rgba.Pix))
			report = r
		}
	}
	if err := preview.WritePNG(outPath, img); err != nil {
		die("kit preview", err)
	}
	printJSON(report)
}

func runPreviewSnapshot(args []string) {
	if len(args) < 1 {
		die("kit preview snapshot", fmt.Errorf("usage: kit preview snapshot <scenario> --out file.json [--stage NAME] [--dat FILE]"))
	}
	in := args[0]
	out, stage, datPath := "", "scenario", ""
	for i := 1; i < len(args); i++ {
		if i+1 >= len(args) {
			die("kit preview snapshot", fmt.Errorf("%s needs a value", args[i]))
		}
		i++
		switch args[i-1] {
		case "--out":
			out = args[i]
		case "--stage":
			stage = args[i]
		case "--dat":
			datPath = args[i]
		default:
			die("kit preview snapshot", fmt.Errorf("unknown option %q", args[i-1]))
		}
	}
	if out == "" {
		die("kit preview snapshot", fmt.Errorf("--out is required"))
	}
	file, err := scenario.OpenArtwork(in)
	if err != nil {
		die("kit preview snapshot", err)
	}
	var dat *datfile.Index
	if datPath != "" {
		dat, err = datfile.Open(datPath)
		if err != nil {
			die("kit preview snapshot", err)
		}
	}
	snap, err := preview.ArtworkSnapshot(file, stage, dat)
	if err != nil {
		die("kit preview snapshot", err)
	}
	if err := preview.WriteSnapshot(out, snap); err != nil {
		die("kit preview snapshot", err)
	}
	printJSON(map[string]any{"ok": true, "stage": stage, "snapshot": out, "objects": len(snap.Objects), "terrain_cells": len(snap.Terrain)})
}
