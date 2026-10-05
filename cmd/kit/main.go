package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/debug"
	"sort"
	"strconv"
	"strings"
	"time"

	"aoe2kit/pkg/aifile"
	"aoe2kit/pkg/aoe2"
	"aoe2kit/pkg/aoems"
	"aoe2kit/pkg/cba"
	"aoe2kit/pkg/customstore"
	"aoe2kit/pkg/datcache"
	"aoe2kit/pkg/datcodec"
	"aoe2kit/pkg/datfacts"
	"aoe2kit/pkg/datfile"
	"aoe2kit/pkg/diagnostics"
	"aoe2kit/pkg/enginefacts"
	"aoe2kit/pkg/fx"
	"aoe2kit/pkg/geom"
	"aoe2kit/pkg/geotrace"
	"aoe2kit/pkg/gfx"
	"aoe2kit/pkg/jsonout"
	"aoe2kit/pkg/kit"
	"aoe2kit/pkg/modpack"
	"aoe2kit/pkg/registry"
	"aoe2kit/pkg/replay"
	"aoe2kit/pkg/roadmap"
	"aoe2kit/pkg/rpgshop"
	"aoe2kit/pkg/scenario"
	"aoe2kit/pkg/triggergraph"
)

const defaultGoMemoryLimitMB = 1024

func main() {
	applyDefaultMemoryLimit()
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "version":
		fmt.Println(kit.Version)
	case "commands":
		commandsCommand(os.Args[2:])
	case "doctor":
		doctor()
	case "docs":
		runDocs(os.Args[2:])
	case "inventory", "scan":
		root := "."
		if len(os.Args) >= 3 {
			root = os.Args[2]
		}
		inventory(root)
	case "summary":
		root := "."
		if len(os.Args) >= 3 {
			root = os.Args[2]
		}
		summary(root)
	case "manifest":
		manifest()
	case "credits":
		credits(os.Args[2:])
	case "verify":
		root := "."
		runGoChecks := false
		for _, arg := range os.Args[2:] {
			switch arg {
			case "--go-checks":
				runGoChecks = true
			default:
				root = arg
			}
		}
		verify(root, runGoChecks)
	case "pack":
		packCommand(os.Args[2:])
	case "portable-check":
		portableCheck(os.Args[2:])
	case "identify":
		identify(os.Args[2:])
	case "collect":
		collect(os.Args[2:])
	case "release-check":
		releaseCheck(os.Args[2:])
	case "verify-run":
		verifyRun(os.Args[2:])
	case "ci":
		runCI(os.Args[2:])
	case "campaign":
		runCampaign(os.Args[2:])
	case "roadmap":
		runRoadmap(os.Args[2:])
	case "rpg":
		runRPG(os.Args[2:])
	case "facts":
		runFacts(os.Args[2:])
	case "project":
		runProject(os.Args[2:])
	case "recipe":
		runRecipe(os.Args[2:])
	case "scen":
		runScen(os.Args[2:])
	case "replay":
		runReplay(os.Args[2:])
	case "ai":
		runAI(os.Args[2:])
	case "player":
		runPlayer(os.Args[2:])
	case "dat":
		runDat(os.Args[2:])
	case "gfx":
		runGFX(os.Args[2:])
	case "geo":
		runGeo(os.Args[2:])
	case "fx":
		runFX(os.Args[2:])
	case "xs":
		runXS(os.Args[2:])
	case "font":
		runFont(os.Args[2:])
	case "xsdat":
		runXSDat(os.Args[2:])
	case "cba":
		runCBA(os.Args[2:])
	case "mod":
		runMod(os.Args[2:])
	case "swatch":
		runSwatch(os.Args[2:])
	default:
		usage()
		os.Exit(2)
	}
}

func applyDefaultMemoryLimit() {
	if os.Getenv("GOMEMLIMIT") != "" {
		return
	}
	limitMB := defaultGoMemoryLimitMB
	if raw := strings.TrimSpace(os.Getenv("AOE2KIT_GOMEMLIMIT_MB")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err == nil && value > 0 {
			limitMB = value
		}
	}
	debug.SetMemoryLimit(int64(limitMB) * 1024 * 1024)
}

func runRoadmap(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: kit roadmap <crud|rwd|dark> [--domain scenario|dat|replay|all] [--text|--json]")
		os.Exit(2)
	}
	command := args[0]
	domain := "all"
	textOut := false
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--domain":
			i++
			if i >= len(args) {
				die("kit roadmap", fmt.Errorf("--domain needs a value"))
			}
			domain = args[i]
		case "--text":
			textOut = true
		case "--json":
			textOut = false
		default:
			die("kit roadmap", fmt.Errorf("unknown argument %q", args[i]))
		}
	}
	switch command {
	case "crud", "capabilities":
		report, err := roadmap.CRUDMatrix(domain)
		if err != nil {
			die("kit roadmap crud", err)
		}
		if textOut {
			printRoadmapCRUDText(report)
			return
		}
		printJSON(report)
	case "rwd", "matrix":
		report, err := roadmap.RWDMatrix(domain)
		if err != nil {
			die("kit roadmap rwd", err)
		}
		if textOut {
			printRoadmapRWDText(report)
			return
		}
		printJSON(report)
	case "dark", "frontier", "dark-bytes":
		report, err := roadmap.DarkBytes(domain)
		if err != nil {
			die("kit roadmap dark", err)
		}
		if textOut {
			printRoadmapDarkText(report)
			return
		}
		printJSON(report)
	default:
		die("kit roadmap", fmt.Errorf("unknown roadmap command %q", command))
	}
}

func printRoadmapCRUDText(report roadmap.CRUDReport) {
	fmt.Printf("AoE2Kit CRUD roadmap %s domain=%s\n", report.Version, report.Domain)
	fmt.Printf("verification=%s\n", report.Verification.Label)
	fmt.Printf("rows=%d create_full=%d read_full=%d update_full=%d delete_full=%d partial_capabilities=%d unsupported=%d n/a=%d needs_engine_run=%d\n",
		report.Summary.Rows,
		report.Summary.CreateFull,
		report.Summary.ReadFull,
		report.Summary.UpdateFull,
		report.Summary.DeleteFull,
		report.Summary.PartialCapabilities,
		report.Summary.Unsupported,
		report.Summary.NotApplicable,
		report.Summary.NeedsEngineRun,
	)
	for _, row := range report.Rows {
		fmt.Printf("\n[%s] %s\n", row.Domain, row.Section)
		printCRUDCapability("create", row.Create)
		printCRUDCapability("read", row.Read)
		printCRUDCapability("update", row.Update)
		printCRUDCapability("delete", row.Delete)
		fmt.Printf("  proof:  %s (%s)\n", row.Verification, row.Evidence)
		fmt.Printf("  next:   %s\n", row.Next)
	}
	if len(report.Notes) > 0 {
		fmt.Println("\nnotes:")
		for _, note := range report.Notes {
			fmt.Printf("  - %s\n", note)
		}
	}
}

func printCRUDCapability(label string, capability roadmap.CRUDCapability) {
	fmt.Printf("  %-6s %s", label+":", capability.Status)
	if capability.Mode != "" {
		fmt.Printf(" [%s]", capability.Mode)
	}
	if capability.Detail != "" {
		fmt.Printf(": %s", capability.Detail)
	}
	fmt.Println()
}

func printRoadmapRWDText(report roadmap.MatrixReport) {
	fmt.Printf("AoE2Kit R/W/D roadmap %s domain=%s\n", report.Version, report.Domain)
	fmt.Printf("verification=%s\n", report.Verification.Label)
	fmt.Printf("rows=%d read_full=%d write_full=%d create_full=%d delete_full=%d partial_statuses=%d unsupported=%d n/a=%d\n",
		report.Summary.Rows,
		report.Summary.ReadFull,
		report.Summary.WriteFull,
		report.Summary.CreateFull,
		report.Summary.DeleteFull,
		report.Summary.PartialRows,
		report.Summary.Unsupported,
		report.Summary.NotApplicable,
	)
	for _, row := range report.Rows {
		fmt.Printf("\n[%s] %s\n", row.Domain, row.Section)
		fmt.Printf("  read:   %s\n", row.Read)
		fmt.Printf("  write:  %s\n", row.Write)
		fmt.Printf("  create: %s\n", row.Create)
		fmt.Printf("  delete: %s\n", row.Delete)
		fmt.Printf("  proof:  %s (%s)\n", row.Verification, row.Evidence)
		fmt.Printf("  next:   %s\n", row.Next)
	}
	if len(report.Notes) > 0 {
		fmt.Println("\nnotes:")
		for _, note := range report.Notes {
			fmt.Printf("  - %s\n", note)
		}
	}
}

func printRoadmapDarkText(report roadmap.DarkReport) {
	fmt.Printf("AoE2Kit dark-byte frontier %s domain=%s\n", report.Version, report.Domain)
	fmt.Printf("verification=%s\n", report.Verification.Label)
	fmt.Printf("frontiers=%d p1=%d p2=%d p3=%d\n",
		report.Summary.Frontiers,
		report.Summary.Priority1,
		report.Summary.Priority2,
		report.Summary.Priority3,
	)
	for _, frontier := range report.Frontiers {
		fmt.Printf("\n[P%d][%s] %s\n", frontier.Priority, frontier.Domain, frontier.Region)
		fmt.Printf("  known:   %s\n", frontier.Known)
		fmt.Printf("  unknown: %s\n", frontier.Unknown)
		fmt.Printf("  tools:   %s\n", frontier.Tools)
		fmt.Printf("  next:    %s\n", frontier.Next)
		fmt.Printf("  proof:   %s\n", frontier.Verification)
	}
	if len(report.Notes) > 0 {
		fmt.Println("\nnotes:")
		for _, note := range report.Notes {
			fmt.Printf("  - %s\n", note)
		}
	}
}

func runGeo(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, geoTraceUsage())
		os.Exit(2)
	}
	switch args[0] {
	case "trace-dem":
		bboxRaw := ""
		centerRaw := ""
		spanKm := 0.0
		gridSize := 32
		provider := "opentopodata"
		landmaskProvider := ""
		landmaskPath := ""
		inlandWaterProvider := ""
		inlandWaterPath := ""
		landcoverProvider := ""
		landcoverCacheDir := ""
		landcoverPath := ""
		landcoverDir := ""
		climateZone := ""
		feltMappingPath := ""
		orient := ""
		outPath := ""
		recipeOutPath := ""
		scenarioOutPath := ""
		omitReportCells := false
		omitReportRecipe := false
		waterFraction := -1.0
		waterFractionSet := false
		seamWaterColumns := 0
		floraDensity := 10.0
		landFaunaDensity := 10.0
		fishDensity := 10.0
		shoreFishDensity := -1.0
		oceanFishDensity := -1.0
		shoreBandWidth := 1.0
		waterBandWidth := 2.0
		roadsProvider := ""
		roadsPath := ""
		roadsCacheDir := ""
		roadSimplifyTiles := 0.0
		adminBorders := false
		wrapEW := false
		wrapPlayers := []int{1}
		sailDemo := false
		for i := 1; i < len(args); i++ {
			switch args[i] {
			case "--bbox":
				i++
				if i >= len(args) {
					die("kit geo trace-dem", fmt.Errorf("--bbox needs minLat,minLon,maxLat,maxLon"))
				}
				bboxRaw = args[i]
			case "--center":
				i++
				if i >= len(args) {
					die("kit geo trace-dem", fmt.Errorf("--center needs lat,lon"))
				}
				centerRaw = args[i]
			case "--span-km":
				i++
				if i >= len(args) {
					die("kit geo trace-dem", fmt.Errorf("--span-km needs a positive kilometer span"))
				}
				value, err := strconv.ParseFloat(args[i], 64)
				if err != nil {
					die("kit geo trace-dem", err)
				}
				if value <= 0 {
					die("kit geo trace-dem", fmt.Errorf("--span-km needs a positive kilometer span"))
				}
				spanKm = value
			case "--grid":
				i++
				if i >= len(args) {
					die("kit geo trace-dem", fmt.Errorf("--grid needs a size"))
				}
				value, err := strconv.Atoi(args[i])
				if err != nil {
					die("kit geo trace-dem", err)
				}
				gridSize = value
			case "--provider":
				i++
				if i >= len(args) {
					die("kit geo trace-dem", fmt.Errorf("--provider needs a value"))
				}
				provider = args[i]
			case "--landmask":
				i++
				if i >= len(args) {
					die("kit geo trace-dem", fmt.Errorf("--landmask needs a value"))
				}
				landmaskProvider = args[i]
			case "--landmask-file":
				i++
				if i >= len(args) {
					die("kit geo trace-dem", fmt.Errorf("--landmask-file needs a path"))
				}
				landmaskPath = args[i]
			case "--inland-water":
				i++
				if i >= len(args) {
					die("kit geo trace-dem", fmt.Errorf("--inland-water needs a value"))
				}
				inlandWaterProvider = args[i]
			case "--inland-water-file":
				i++
				if i >= len(args) {
					die("kit geo trace-dem", fmt.Errorf("--inland-water-file needs a path"))
				}
				inlandWaterPath = args[i]
			case "--landcover":
				i++
				if i >= len(args) {
					die("kit geo trace-dem", fmt.Errorf("--landcover needs a value"))
				}
				landcoverProvider = args[i]
			case "--landcover-cache":
				i++
				if i >= len(args) {
					die("kit geo trace-dem", fmt.Errorf("--landcover-cache needs a directory"))
				}
				landcoverCacheDir = args[i]
			case "--landcover-file":
				i++
				if i >= len(args) {
					die("kit geo trace-dem", fmt.Errorf("--landcover-file needs a path"))
				}
				landcoverPath = args[i]
			case "--landcover-dir":
				i++
				if i >= len(args) {
					die("kit geo trace-dem", fmt.Errorf("--landcover-dir needs a directory"))
				}
				landcoverDir = args[i]
			case "--climate":
				i++
				if i >= len(args) {
					die("kit geo trace-dem", fmt.Errorf("--climate needs a zone"))
				}
				climateZone = args[i]
			case "--felt-mapping":
				i++
				if i >= len(args) {
					die("kit geo trace-dem", fmt.Errorf("--felt-mapping needs a path"))
				}
				feltMappingPath = args[i]
			case "--orient":
				i++
				if i >= len(args) {
					die("kit geo trace-dem", fmt.Errorf("--orient needs iso, iso-cw, iso-ccw, iso-ccw-screen, grid, or diamond"))
				}
				orient = args[i]
			case "--out":
				i++
				if i >= len(args) {
					die("kit geo trace-dem", fmt.Errorf("--out needs a path"))
				}
				outPath = args[i]
			case "--recipe-out":
				i++
				if i >= len(args) {
					die("kit geo trace-dem", fmt.Errorf("--recipe-out needs a path"))
				}
				recipeOutPath = args[i]
			case "--scenario-out":
				i++
				if i >= len(args) {
					die("kit geo trace-dem", fmt.Errorf("--scenario-out needs a path"))
				}
				scenarioOutPath = args[i]
			case "--omit-report-cells":
				omitReportCells = true
			case "--omit-report-recipe":
				omitReportRecipe = true
			case "--water-fraction":
				i++
				if i >= len(args) {
					die("kit geo trace-dem", fmt.Errorf("--water-fraction needs a value between 0 and 1"))
				}
				value, err := strconv.ParseFloat(args[i], 64)
				if err != nil {
					die("kit geo trace-dem", err)
				}
				if value <= 0 || value >= 1 {
					die("kit geo trace-dem", fmt.Errorf("--water-fraction needs a value between 0 and 1"))
				}
				waterFraction = value
				waterFractionSet = true
			case "--seam-water-cols":
				i++
				if i >= len(args) {
					die("kit geo trace-dem", fmt.Errorf("--seam-water-cols needs a count"))
				}
				value, err := strconv.Atoi(args[i])
				if err != nil {
					die("kit geo trace-dem", err)
				}
				if value < 0 {
					die("kit geo trace-dem", fmt.Errorf("--seam-water-cols must be >= 0"))
				}
				seamWaterColumns = value
			case "--flora-density":
				i++
				if i >= len(args) {
					die("kit geo trace-dem", fmt.Errorf("--flora-density needs a 1-10 value"))
				}
				value, err := parseDensityDial(args[i])
				if err != nil {
					die("kit geo trace-dem", err)
				}
				floraDensity = value
			case "--land-fauna-density":
				i++
				if i >= len(args) {
					die("kit geo trace-dem", fmt.Errorf("--land-fauna-density needs a 1-10 value"))
				}
				value, err := parseDensityDial(args[i])
				if err != nil {
					die("kit geo trace-dem", err)
				}
				landFaunaDensity = value
			case "--fish-density":
				i++
				if i >= len(args) {
					die("kit geo trace-dem", fmt.Errorf("--fish-density needs a 1-10 value"))
				}
				value, err := parseDensityDial(args[i])
				if err != nil {
					die("kit geo trace-dem", err)
				}
				fishDensity = value
			case "--shore-fish-density":
				i++
				if i >= len(args) {
					die("kit geo trace-dem", fmt.Errorf("--shore-fish-density needs a 1-10 value"))
				}
				value, err := parseDensityDial(args[i])
				if err != nil {
					die("kit geo trace-dem", err)
				}
				shoreFishDensity = value
			case "--ocean-fish-density":
				i++
				if i >= len(args) {
					die("kit geo trace-dem", fmt.Errorf("--ocean-fish-density needs a 1-10 value"))
				}
				value, err := parseDensityDial(args[i])
				if err != nil {
					die("kit geo trace-dem", err)
				}
				oceanFishDensity = value
			case "--shore-band-width":
				i++
				if i >= len(args) {
					die("kit geo trace-dem", fmt.Errorf("--shore-band-width needs a positive tile width"))
				}
				value, err := strconv.ParseFloat(args[i], 64)
				if err != nil {
					die("kit geo trace-dem", err)
				}
				if value <= 0 {
					die("kit geo trace-dem", fmt.Errorf("--shore-band-width needs a positive tile width"))
				}
				shoreBandWidth = value
			case "--water-band-width":
				i++
				if i >= len(args) {
					die("kit geo trace-dem", fmt.Errorf("--water-band-width needs a positive tile width"))
				}
				value, err := strconv.ParseFloat(args[i], 64)
				if err != nil {
					die("kit geo trace-dem", err)
				}
				if value <= 0 {
					die("kit geo trace-dem", fmt.Errorf("--water-band-width needs a positive tile width"))
				}
				waterBandWidth = value
			case "--roads":
				i++
				if i >= len(args) {
					die("kit geo trace-dem", fmt.Errorf("--roads needs osm-interstate, osm-motorway, osm-major, or nhpn-interstate with --roads-file"))
				}
				roadsProvider = args[i]
			case "--roads-file":
				i++
				if i >= len(args) {
					die("kit geo trace-dem", fmt.Errorf("--roads-file needs a path"))
				}
				roadsPath = args[i]
			case "--roads-cache":
				i++
				if i >= len(args) {
					die("kit geo trace-dem", fmt.Errorf("--roads-cache needs a directory"))
				}
				roadsCacheDir = args[i]
			case "--road-simplify-tiles":
				i++
				if i >= len(args) {
					die("kit geo trace-dem", fmt.Errorf("--road-simplify-tiles needs a non-negative tile tolerance"))
				}
				value, err := strconv.ParseFloat(args[i], 64)
				if err != nil {
					die("kit geo trace-dem", err)
				}
				if value < 0 {
					die("kit geo trace-dem", fmt.Errorf("--road-simplify-tiles needs a non-negative tile tolerance"))
				}
				roadSimplifyTiles = value
			case "--admin-borders":
				adminBorders = true
			case "--wrap-ew":
				wrapEW = true
			case "--wrap-players":
				i++
				if i >= len(args) {
					die("kit geo trace-dem", fmt.Errorf("--wrap-players needs comma-separated player ids"))
				}
				value, err := parseCommaInts(args[i])
				if err != nil {
					die("kit geo trace-dem", err)
				}
				wrapPlayers = value
			case "--sail-demo":
				sailDemo = true
			default:
				die("kit geo trace-dem", fmt.Errorf("unknown arg %q", args[i]))
			}
		}
		var bbox geotrace.BBox
		if bboxRaw != "" {
			if centerRaw != "" || spanKm > 0 {
				die("kit geo trace-dem", fmt.Errorf("--bbox cannot be combined with --center/--span-km"))
			}
			var err error
			bbox, err = parseGeoBBox(bboxRaw)
			if err != nil {
				die("kit geo trace-dem", err)
			}
		} else {
			if centerRaw == "" || spanKm <= 0 {
				die("kit geo trace-dem", fmt.Errorf("--bbox or --center lat,lon with --span-km is required"))
			}
			centerLat, centerLon, err := parseGeoCenter(centerRaw)
			if err != nil {
				die("kit geo trace-dem", err)
			}
			bbox = geoBBoxFromCenterSpan(centerLat, centerLon, spanKm)
		}
		if wrapEW {
			if !waterFractionSet {
				waterFraction = 0.40
			}
			if seamWaterColumns == 0 {
				seamWaterColumns = 4
			}
		}
		if shoreFishDensity <= 0 {
			shoreFishDensity = fishDensity
		}
		if oceanFishDensity <= 0 {
			oceanFishDensity = fishDensity
		}
		report, err := geotrace.TraceDEM(geotrace.Options{
			BBox:                bbox,
			GridSize:            gridSize,
			Provider:            provider,
			LandmaskProvider:    landmaskProvider,
			LandmaskPath:        landmaskPath,
			InlandWaterProvider: inlandWaterProvider,
			InlandWaterPath:     inlandWaterPath,
			LandcoverProvider:   landcoverProvider,
			LandcoverCacheDir:   landcoverCacheDir,
			LandcoverPath:       landcoverPath,
			LandcoverDir:        landcoverDir,
			ClimateZone:         climateZone,
			FeltMappingPath:     feltMappingPath,
			Orient:              orient,
			TargetWaterFraction: waterFraction,
			SeamWaterColumns:    seamWaterColumns,
			FloraDensity:        floraDensity,
			LandFaunaDensity:    landFaunaDensity,
			FishDensity:         fishDensity,
			ShoreFishDensity:    shoreFishDensity,
			OceanFishDensity:    oceanFishDensity,
			ShoreBandWidth:      shoreBandWidth,
			WaterBandWidth:      waterBandWidth,
			RoadsProvider:       roadsProvider,
			RoadsPath:           roadsPath,
			RoadsCacheDir:       roadsCacheDir,
			RoadSimplifyTiles:   roadSimplifyTiles,
			AdminBorders:        adminBorders,
		})
		if err != nil {
			die("kit geo trace-dem", err)
		}
		if report.Recipe != nil {
			if wrapEW {
				if report.Orient == "diamond" {
					report.Recipe.Triggers = append(report.Recipe.Triggers, geotrace.BuildDiamondLongitudeWrapTriggers(gridSize, wrapPlayers, seamWaterColumns)...)
				} else {
					report.Recipe.Triggers = append(report.Recipe.Triggers, geotrace.BuildEastWestWrapTriggers(gridSize, wrapPlayers)...)
				}
			}
			if sailDemo {
				report.Recipe.Units = append(report.Recipe.Units, geotrace.BuildSailDemoUnits(gridSize)...)
			}
			if wrapEW || sailDemo {
				zero := 0
				report.Recipe.Victory = &scenario.VictoryRecipe{ConquestRequired: &zero}
			}
		}
		if recipeOutPath != "" {
			if report.Recipe == nil {
				die("kit geo trace-dem", fmt.Errorf("--recipe-out requires a landmask so the emitted recipe has explicit land/water masks"))
			}
			if err := writeJSONFile(recipeOutPath, report.Recipe); err != nil {
				die("kit geo trace-dem", err)
			}
		}
		if scenarioOutPath != "" {
			if report.Recipe == nil {
				die("kit geo trace-dem", fmt.Errorf("--scenario-out requires a landmask so the emitted scenario has explicit land/water masks"))
			}
			if err := writeGeoTraceScenario(scenarioOutPath, gridSize, *report.Recipe); err != nil {
				die("kit geo trace-dem", err)
			}
		}
		reportForJSON := report
		if omitReportCells {
			reportForJSON.Cells = nil
		}
		if omitReportRecipe {
			reportForJSON.Recipe = nil
		}
		if outPath != "" {
			file, err := os.Create(outPath)
			if err != nil {
				die("kit geo trace-dem", err)
			}
			err = jsonout.Encode(file, reportForJSON)
			closeErr := file.Close()
			if err != nil {
				die("kit geo trace-dem", err)
			}
			if closeErr != nil {
				die("kit geo trace-dem", closeErr)
			}
			fmt.Printf("wrote %s cells=%d coastline=%s tile_m=%.0f", outPath, len(report.Cells), report.CoastlineConfidence, report.ApproxTileMeters)
			if report.RoadCells > 0 {
				fmt.Printf(" roads=%d ways=%d vertices=%d/%d simplify_tiles=%.2f", report.RoadCells, report.RoadWayCount, report.RoadVertexCount, report.RoadVertexCountRaw, report.RoadSimplifyTiles)
			}
			if recipeOutPath != "" {
				fmt.Printf(" recipe=%s", recipeOutPath)
			}
			if scenarioOutPath != "" {
				fmt.Printf(" scenario=%s", scenarioOutPath)
			}
			fmt.Println()
			return
		}
		if err := jsonout.Print(reportForJSON); err != nil {
			die("kit geo trace-dem", err)
		}
	default:
		fmt.Fprintln(os.Stderr, geoTraceUsage())
		os.Exit(2)
	}
}

func geoTraceUsage() string {
	return "usage: kit geo trace-dem [--bbox minLat,minLon,maxLat,maxLon] [--center lat,lon] [--span-km N] [--grid N] [--provider opentopodata|flat] [--landmask naturalearth|geojson] [--landmask-file path] [--inland-water naturalearth|geojson] [--inland-water-file path] [--landcover worldcover|glc-fcs30d] [--landcover-cache dir] [--landcover-dir dir] [--landcover-file path] [--climate zone] [--felt-mapping path] [--orient iso|iso-cw|iso-ccw|iso-ccw-screen|grid|diamond] [--water-fraction 0..1] [--flora-density 1..10] [--land-fauna-density 1..10] [--fish-density 1..10] [--shore-fish-density 1..10] [--ocean-fish-density 1..10] [--shore-band-width tiles] [--water-band-width tiles] [--roads osm-interstate|osm-motorway|osm-major|nhpn-interstate] [--roads-file roads.json] [--roads-cache dir] [--road-simplify-tiles N] [--admin-borders] [--seam-water-cols N] [--wrap-ew] [--wrap-players 1,2] [--sail-demo] [--omit-report-cells] [--omit-report-recipe] [--out report.json] [--recipe-out recipe.json] [--scenario-out out.aoe2scenario]"
}

func writeGeoTraceScenario(outPath string, gridSize int, recipe scenario.Recipe) error {
	dir := filepath.Dir(outPath)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	tmp, err := os.CreateTemp(dir, ".geotrace-blank-*.aoe2scenario")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	if err := tmp.Close(); err != nil {
		return err
	}
	defer os.Remove(tmpPath)
	if _, err := scenario.WriteBlankScenarioFile(tmpPath, scenario.BlankOptions{
		PlayerCount: 1,
		HumanSlots:  1,
		MapWidth:    gridSize,
		MapHeight:   gridSize,
	}); err != nil {
		return err
	}
	terrainPath := tmpPath + ".terrain.aoe2scenario"
	defer os.Remove(terrainPath)
	terrainRecipe := recipe
	terrainRecipe.Units = nil
	if _, err := scenario.PatchRecipeFile(tmpPath, terrainPath, terrainRecipe); err != nil {
		return err
	}
	unitRecipe := scenario.Recipe{Units: recipe.Units}
	_, err = scenario.PatchRecipeFile(terrainPath, outPath, unitRecipe)
	return err
}

func writeJSONFile(path string, value any) error {
	if dir := filepath.Dir(path); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

func parseGeoBBox(raw string) (geotrace.BBox, error) {
	parts := strings.Split(raw, ",")
	if len(parts) != 4 {
		return geotrace.BBox{}, fmt.Errorf("bbox must be minLat,minLon,maxLat,maxLon")
	}
	values := make([]float64, 4)
	for i, part := range parts {
		value, err := strconv.ParseFloat(strings.TrimSpace(part), 64)
		if err != nil {
			return geotrace.BBox{}, err
		}
		values[i] = value
	}
	return geotrace.BBox{MinLat: values[0], MinLon: values[1], MaxLat: values[2], MaxLon: values[3]}, nil
}

func parseGeoCenter(raw string) (lat, lon float64, err error) {
	parts := strings.Split(raw, ",")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("center must be lat,lon")
	}
	lat, err = strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	if err != nil {
		return 0, 0, err
	}
	lon, err = strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	if err != nil {
		return 0, 0, err
	}
	if lat < -90 || lat > 90 || lon < -180 || lon > 180 {
		return 0, 0, fmt.Errorf("center is out of range")
	}
	return lat, lon, nil
}

func geoBBoxFromCenterSpan(lat, lon, spanKm float64) geotrace.BBox {
	half := spanKm / 2
	kmPerLat := 111.320
	kmPerLon := 111.320 * math.Cos(lat*math.Pi/180)
	if kmPerLon <= 0 {
		kmPerLon = 1
	}
	latHalf := half / kmPerLat
	lonHalf := half / kmPerLon
	return geotrace.BBox{
		MinLat: lat - latHalf,
		MinLon: lon - lonHalf,
		MaxLat: lat + latHalf,
		MaxLon: lon + lonHalf,
	}
}

func parseCommaInts(raw string) ([]int, error) {
	parts := strings.Split(raw, ",")
	out := make([]int, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		value, err := strconv.Atoi(part)
		if err != nil {
			return nil, err
		}
		out = append(out, value)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("expected at least one integer")
	}
	return out, nil
}

func parseDensityDial(raw string) (float64, error) {
	value, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil {
		return 0, err
	}
	if value < 1 || value > 10 {
		return 0, fmt.Errorf("density dial must be in range 1..10")
	}
	return value, nil
}

func runRPG(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: kit rpg <shop-demo|custom-store> [--out-dir DIR] [--name TEXT] [--unit N] [--instances N] [--players N] [--active-players LIST] [--timestamp UNIX] [--xs-check PATH] [--allow-missing-xs-check] [--text|--json]")
		os.Exit(2)
	}
	switch args[0] {
	case "shop-demo":
		opts := rpgshop.DemoOptions{}
		textOut := false
		for i := 1; i < len(args); i++ {
			switch args[i] {
			case "--out-dir":
				i++
				if i >= len(args) {
					die("kit rpg shop-demo", fmt.Errorf("--out-dir needs a value"))
				}
				opts.OutputDir = args[i]
			case "--name":
				i++
				if i >= len(args) {
					die("kit rpg shop-demo", fmt.Errorf("--name needs a value"))
				}
				opts.ScenarioName = args[i]
			case "--timestamp":
				i++
				if i >= len(args) {
					die("kit rpg shop-demo", fmt.Errorf("--timestamp needs a unix value"))
				}
				value, err := strconv.Atoi(args[i])
				if err != nil {
					die("kit rpg shop-demo", fmt.Errorf("--timestamp needs an integer"))
				}
				opts.Timestamp = value
			case "--text":
				textOut = true
			case "--json":
				textOut = false
			default:
				die("kit rpg shop-demo", fmt.Errorf("unknown option %q", args[i]))
			}
		}
		report, err := rpgshop.BuildHeroShopDemo(opts)
		if err != nil {
			die("kit rpg shop-demo", err)
		}
		if textOut {
			printRPGShopDemoText(report)
			return
		}
		printJSON(report)
	case "custom-store", "tower-store":
		opts := customstore.Options{}
		requireXSCheck := true
		textOut := false
		for i := 1; i < len(args); i++ {
			switch args[i] {
			case "--unit":
				if i+1 >= len(args) {
					die("kit rpg custom-store", fmt.Errorf("--unit needs a value"))
				}
				v, err := strconv.Atoi(args[i+1])
				if err != nil {
					die("kit rpg custom-store", fmt.Errorf("--unit needs an integer: %w", err))
				}
				opts.ObjectUnitConst = v
				i++
			case "--instances":
				if i+1 >= len(args) {
					die("kit rpg custom-store", fmt.Errorf("--instances needs a value"))
				}
				v, err := strconv.Atoi(args[i+1])
				if err != nil {
					die("kit rpg custom-store", fmt.Errorf("--instances needs an integer: %w", err))
				}
				opts.InstanceCount = v
				i++
			case "--players":
				if i+1 >= len(args) {
					die("kit rpg custom-store", fmt.Errorf("--players needs a value"))
				}
				v, err := strconv.Atoi(args[i+1])
				if err != nil {
					die("kit rpg custom-store", fmt.Errorf("--players needs an integer: %w", err))
				}
				opts.PlayerCount = v
				i++
			case "--active-players":
				if i+1 >= len(args) {
					die("kit rpg custom-store", fmt.Errorf("--active-players needs a comma-separated list"))
				}
				players, err := parseCommaInts(args[i+1])
				if err != nil {
					die("kit rpg custom-store", fmt.Errorf("--active-players needs comma-separated integers: %w", err))
				}
				opts.ActivePlayers = players
				i++
			case "--out-dir":
				i++
				if i >= len(args) {
					die("kit rpg custom-store", fmt.Errorf("--out-dir needs a value"))
				}
				opts.OutputDir = args[i]
			case "--name":
				i++
				if i >= len(args) {
					die("kit rpg custom-store", fmt.Errorf("--name needs a value"))
				}
				opts.ScenarioName = args[i]
			case "--timestamp":
				i++
				if i >= len(args) {
					die("kit rpg custom-store", fmt.Errorf("--timestamp needs a unix value"))
				}
				value, err := strconv.Atoi(args[i])
				if err != nil {
					die("kit rpg custom-store", fmt.Errorf("--timestamp needs an integer"))
				}
				opts.Timestamp = value
			case "--xs-check":
				i++
				if i >= len(args) {
					die("kit rpg custom-store", fmt.Errorf("--xs-check needs a path"))
				}
				opts.XSCheckPath = args[i]
			case "--allow-missing-xs-check":
				requireXSCheck = false
			case "--text":
				textOut = true
			case "--json":
				textOut = false
			default:
				die("kit rpg custom-store", fmt.Errorf("unknown option %q", args[i]))
			}
		}
		opts.RequireXSCheck = requireXSCheck
		report, err := customstore.Build(opts)
		if err != nil {
			die("kit rpg custom-store", err)
		}
		if textOut {
			printCustomStoreText(report)
			return
		}
		printJSON(report)
	default:
		die("kit rpg", fmt.Errorf("unknown rpg command %q", args[0]))
	}
}

func printCustomStoreText(report *customstore.Report) {
	fmt.Printf("custom store: %s\n", report.ScenarioPath)
	fmt.Printf("scenario_sha256=%s xs_sha256=%s\n", report.ScenarioSHA256, report.XSSHA256)
	fmt.Printf("recipe=%s\nxs=%s\ntypes=%s\n", report.RecipePath, report.XSPath, report.TypesPath)
	fmt.Printf("scale: objects=%d objects_per_player=%d players=%d active_players=%v unit_const=%d units=%d triggers=%d\n", report.InstanceCount, report.InstancesPerPlayer, report.PlayerCount, report.ActivePlayers, report.ObjectUnitConst, report.UnitCount, report.TriggerCount)
	fmt.Printf("xs_parse_check: %s", report.XSParseCheck)
	if report.XSParseChecker != "" {
		fmt.Printf(" (%s)", report.XSParseChecker)
	}
	fmt.Println()
	fmt.Printf("semantic_lint: ok=%t issues=%d\n", report.SemanticLint.OK, len(report.SemanticLint.Issues))
	fmt.Printf("verification: %s\n", report.Verification)
	for _, gap := range report.KnownGaps {
		fmt.Printf("known_gap: %s\n", gap)
	}
}

func printRPGShopDemoText(report *rpgshop.DemoReport) {
	fmt.Printf("rpg shop demo: %s\n", report.ScenarioPath)
	fmt.Printf("scenario_sha256=%s xs_sha256=%s\n", report.ScenarioSHA256, report.XSSHA256)
	fmt.Printf("recipe=%s\nxs=%s\nmanifest=%s\n", report.RecipePath, report.XSPath, report.ManifestPath)
	fmt.Printf("semantic_lint: ok=%t issues=%d\n", report.SemanticLint.OK, len(report.SemanticLint.Issues))
	fmt.Printf("blank: triggers %d->%d units %d->%d rebuild_ok=%t invariant_ok=%t\n",
		report.BlankReport.TriggerCountBefore,
		report.BlankReport.TriggerCountAfter,
		report.BlankReport.UnitCountBefore,
		report.BlankReport.UnitCountAfter,
		report.BlankReport.RebuildOK,
		report.BlankReport.InvariantOK,
	)
	fmt.Printf("patch: triggers=%d units=%d verification=%s\n",
		report.PatchReport.TriggerCountAfter,
		report.PatchReport.UnitCountAfter,
		report.PatchReport.Verification.Label,
	)
	fmt.Printf("variables=%d shop_items=%d\n", len(report.Variables), len(report.ShopItems))
	for _, claim := range report.VerificationClaims {
		fmt.Printf("- %s\n", claim)
	}
	if len(report.KnownGaps) > 0 {
		fmt.Println("known gaps:")
		for _, gap := range report.KnownGaps {
			fmt.Printf("- %s\n", gap)
		}
	}
}

type releaseCheckReport struct {
	Scenario     string                 `json:"scenario"`
	Previous     string                 `json:"previous,omitempty"`
	Mod          string                 `json:"mod,omitempty"`
	OK           bool                   `json:"ok"`
	Verification aoe2.VerificationClaim `json:"verification"`
	Identity     *identityResult        `json:"identity,omitempty"`
	ScenarioLint *scenario.LintReport   `json:"scenario_lint,omitempty"`
	ScenarioDiff *scenario.DiffReport   `json:"scenario_diff,omitempty"`
	ModCheck     *modpack.CheckReport   `json:"mod_check,omitempty"`
	Pack         *kit.PackReport        `json:"pack,omitempty"`
	Warnings     []string               `json:"warnings,omitempty"`
	Errors       []string               `json:"errors,omitempty"`
}

func releaseCheck(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: kit release-check <scenario.aoe2scenario> [--previous old.aoe2scenario] [--mod moddir] [--status mod-status.json] [--pack out.zip] [--text]")
		os.Exit(2)
	}
	scenarioPath := args[0]
	previous := ""
	modDir := ""
	statusPath := ""
	packOut := ""
	textOut := false
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--previous":
			i++
			if i >= len(args) {
				die("kit release-check", fmt.Errorf("--previous needs a scenario path"))
			}
			previous = args[i]
		case "--mod":
			i++
			if i >= len(args) {
				die("kit release-check", fmt.Errorf("--mod needs a mod directory"))
			}
			modDir = args[i]
		case "--status":
			i++
			if i >= len(args) {
				die("kit release-check", fmt.Errorf("--status needs a mod-status.json path"))
			}
			statusPath = args[i]
		case "--pack":
			i++
			if i >= len(args) {
				die("kit release-check", fmt.Errorf("--pack needs an output zip path"))
			}
			packOut = args[i]
		case "--text":
			textOut = true
		default:
			die("kit release-check", fmt.Errorf("unknown option %q", args[i]))
		}
	}
	report := releaseCheckReport{Scenario: scenarioPath, Previous: previous, Mod: modDir}
	report.Verification = aoe2.StructureVerification(true)
	report.Verification.Note = "release-check composes AoE2Kit structural/package checks only; in-engine load/render/play remains a separate oracle."
	structureOK := true
	activationFailed := false
	if result, err := fingerprintFile(scenarioPath); err != nil {
		report.Errors = append(report.Errors, "identify scenario: "+err.Error())
		structureOK = false
	} else {
		report.Identity = &result
	}
	if lint, err := scenario.LintFile(scenarioPath); err != nil {
		report.Errors = append(report.Errors, "scenario lint: "+err.Error())
		structureOK = false
	} else {
		report.ScenarioLint = &lint
		if !lint.OK {
			report.Errors = append(report.Errors, "scenario lint failed")
			structureOK = false
		}
	}
	if previous != "" {
		if diff, err := scenario.DiffFiles(previous, scenarioPath); err != nil {
			report.Errors = append(report.Errors, "scenario diff: "+err.Error())
			structureOK = false
		} else {
			report.ScenarioDiff = &diff
		}
	}
	if modDir != "" {
		modReport, err := modpack.CheckWithOptions(modDir, modpack.CheckOptions{StatusPath: statusPath})
		if err != nil {
			report.Errors = append(report.Errors, "mod check: "+err.Error())
			structureOK = false
		} else {
			report.ModCheck = &modReport
			if !modReport.OK() {
				report.Errors = append(report.Errors, "mod check failed")
				if !modReport.Verification.StructureVerified {
					structureOK = false
				}
				if modReport.Activation != nil && len(modReport.Activation.Errors) > 0 {
					activationFailed = true
				}
			}
		}
	}
	if packOut != "" {
		packReport, err := kit.Pack(".", packOut)
		report.Pack = &packReport
		if err != nil {
			report.Errors = append(report.Errors, "pack: "+err.Error())
			structureOK = false
		}
	}
	report.OK = len(report.Errors) == 0
	report.Verification = aoe2.StructureVerification(structureOK)
	if structureOK && activationFailed {
		report.Verification = aoe2.StructureOKActivationFailed()
	}
	report.Verification.Note = "release-check composes AoE2Kit structural/package checks only; in-engine load/render/play remains a separate oracle."
	if textOut {
		printReleaseCheck(report)
	} else {
		printJSON(report)
	}
	if !report.OK {
		os.Exit(1)
	}
}

func printReleaseCheck(report releaseCheckReport) {
	status := "OK"
	if !report.OK {
		status = "FAIL"
	}
	fmt.Printf("scenario: %s\n", report.Scenario)
	fmt.Printf("status: %s\n", status)
	fmt.Printf("verification: %s\n", report.Verification.Label)
	if report.Identity != nil {
		fmt.Printf("identity: %s %s\n", report.Identity.FingerprintTier, shortHash(report.Identity.Fingerprint))
	}
	if report.ScenarioLint != nil {
		fmt.Printf("scenario_lint: ok=%t triggers=%d units=%d\n", report.ScenarioLint.OK, report.ScenarioLint.Summary.Triggers, report.ScenarioLint.Summary.Units)
	}
	if report.ScenarioDiff != nil {
		fmt.Printf("scenario_diff: same=%t changes=%d\n", report.ScenarioDiff.Same, len(report.ScenarioDiff.Changes))
	}
	if report.ModCheck != nil {
		activation := "not_checked"
		if report.ModCheck.Activation != nil {
			activation = fmt.Sprintf("ok=%t", report.ModCheck.Activation.ActivationOK)
		}
		fmt.Printf("mod_check: ok=%t activation=%s warnings=%d\n", report.ModCheck.OK(), activation, len(report.ModCheck.Warnings))
	}
	if report.Pack != nil {
		fmt.Printf("pack: verified=%t output=%s sha256=%s\n", report.Pack.Verified, report.Pack.Output, report.Pack.SHA256)
	}
	for _, warning := range report.Warnings {
		fmt.Printf("warning: %s\n", warning)
	}
	for _, err := range report.Errors {
		fmt.Printf("error: %s\n", err)
	}
}

func doctor() {
	fmt.Println("AoE2Kit doctor")
	fmt.Printf("- Version: %s\n", kit.Version)
	if _, err := exec.LookPath("go"); err == nil {
		fmt.Println("- Go toolchain: available")
	} else {
		fmt.Println("- Go toolchain: not found (bundled linux-amd64 binary can run, but source build cannot be verified here)")
	}
	fmt.Println("- Durable language: Go")
	fmt.Println("- Python required: no")
	fmt.Println("- Network required: no")
	fmt.Println("- Use `kit inventory <folder>` to discover bundled scenarios, dat files, records, AI files, and mod folders.")
}

func manifest() {
	data, err := kit.ManifestJSON()
	if err != nil {
		fmt.Fprintf(os.Stderr, "kit manifest: %v\n", err)
		os.Exit(1)
	}
	os.Stdout.Write(data)
	os.Stdout.Write([]byte("\n"))
}

func verify(root string, runGoChecks bool) {
	report := kit.Verify(root, runGoChecks)
	if err := jsonout.Print(report); err != nil {
		fmt.Fprintf(os.Stderr, "kit verify: %v\n", err)
		os.Exit(1)
	}
	if !report.OK() {
		os.Exit(1)
	}
}

func pack(root, output string) {
	report, err := kit.Pack(root, output)
	if err != nil {
		printJSON(report)
		die("kit pack", err)
	}
	printJSON(report)
}

func credits(args []string) {
	root := "."
	text := false
	notice := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "notice":
			notice = true
		case "--text":
			text = true
		case "--json":
			text = false
		default:
			if strings.HasPrefix(args[i], "-") {
				die("kit credits", fmt.Errorf("unknown flag %q", args[i]))
			}
			root = args[i]
		}
	}
	doc, err := kit.Credits(root)
	if err != nil {
		die("kit credits", err)
	}
	if notice {
		fmt.Print(string(kit.CreditsNoticeMarkdown(doc)))
		return
	}
	if text {
		fmt.Printf("%s (%s)\n", doc.Project, doc.License)
		for _, entry := range doc.References {
			fmt.Printf("- %s: %s\n", entry.Name, entry.Credit)
		}
		return
	}
	printJSON(doc)
}

func portableCheck(args []string) {
	root := "."
	opts := kit.PortableCheckOptions{Profile: kit.ProfileHandoff}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--profile":
			if i+1 >= len(args) {
				die("kit portable-check", fmt.Errorf("--profile requires a value (full|handoff|sandbox|public)"))
			}
			opts.Profile = kit.PackProfile(args[i+1])
			i++
		case strings.HasPrefix(arg, "--profile="):
			opts.Profile = kit.PackProfile(strings.TrimPrefix(arg, "--profile="))
		case strings.HasPrefix(arg, "-"):
			die("kit portable-check", fmt.Errorf("unknown flag %q", arg))
		default:
			root = arg
		}
	}
	report, err := kit.PortableCheckRootWithOptions(root, opts)
	if err != nil {
		die("kit portable-check", err)
	}
	printJSON(report)
	if !report.OK {
		os.Exit(1)
	}
}

type docsLintReport struct {
	Root         string          `json:"root"`
	OK           bool            `json:"ok"`
	Verification string          `json:"verification"`
	FilesScanned int             `json:"files_scanned"`
	Examples     int             `json:"examples"`
	Issues       []docsLintIssue `json:"issues,omitempty"`
}

type docsLintIssue struct {
	Path    string `json:"path"`
	Line    int    `json:"line"`
	Command string `json:"command,omitempty"`
	Message string `json:"message"`
}

func runDocs(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: kit docs lint [folder] [--text|--json]")
		os.Exit(2)
	}
	switch args[0] {
	case "lint":
		root := "docs"
		textOut := false
		for _, arg := range args[1:] {
			switch arg {
			case "--text":
				textOut = true
			case "--json":
				textOut = false
			default:
				root = arg
			}
		}
		report, err := lintDocs(root)
		if err != nil {
			die("kit docs lint", err)
		}
		if textOut {
			printDocsLint(report)
		} else {
			printJSON(report)
		}
		if !report.OK {
			os.Exit(1)
		}
	default:
		fmt.Fprintln(os.Stderr, "usage: kit docs lint [folder] [--text|--json]")
		os.Exit(2)
	}
}

func lintDocs(root string) (docsLintReport, error) {
	st, err := os.Stat(root)
	if err != nil {
		return docsLintReport{}, err
	}
	if !st.IsDir() {
		return docsLintReport{}, fmt.Errorf("%s is not a directory", root)
	}
	report := docsLintReport{
		Root:         root,
		Verification: "structure_verified_not_engine_verified",
	}
	known := map[string]bool{}
	for _, spec := range commandCatalog {
		known[spec.Name] = true
	}
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			report.Issues = append(report.Issues, docsLintIssue{Path: path, Message: walkErr.Error()})
			return nil
		}
		if d.IsDir() {
			if strings.HasPrefix(d.Name(), ".") || d.Name() == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		base := filepath.Base(path)
		if base == "PACKAGE_API.md" || base == "API_REFERENCE.md" {
			return nil
		}
		if strings.ToLower(filepath.Ext(path)) != ".md" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			report.Issues = append(report.Issues, docsLintIssue{Path: path, Message: err.Error()})
			return nil
		}
		report.FilesScanned++
		lines := strings.Split(string(data), "\n")
		inFence := false
		for idx, line := range lines {
			if strings.HasPrefix(strings.TrimSpace(line), "```") {
				inFence = !inFence
				continue
			}
			cmd, ok := parseDocsKitCommand(line, known, inFence)
			if !ok {
				continue
			}
			report.Examples++
			if cmd == "" {
				report.Issues = append(report.Issues, docsLintIssue{
					Path:    path,
					Line:    idx + 1,
					Message: "kit command example does not match the command catalog",
				})
			}
		}
		return nil
	})
	if err != nil {
		return docsLintReport{}, err
	}
	report.OK = len(report.Issues) == 0
	return report, nil
}

func parseDocsKitCommand(line string, known map[string]bool, inFence bool) (string, bool) {
	trimmed := strings.TrimSpace(line)
	shellPrompt := strings.HasPrefix(trimmed, "$") || strings.HasPrefix(trimmed, ">")
	directKit := strings.HasPrefix(trimmed, "./kit")
	if !inFence && !shellPrompt && !directKit {
		return "", false
	}
	for {
		before := trimmed
		trimmed = strings.TrimSpace(strings.TrimPrefix(trimmed, "$"))
		trimmed = strings.TrimSpace(strings.TrimPrefix(trimmed, ">"))
		if before == trimmed {
			break
		}
	}
	if strings.HasPrefix(trimmed, "go run ./cmd/kit ") {
		trimmed = "kit " + strings.TrimPrefix(trimmed, "go run ./cmd/kit ")
	} else if strings.HasPrefix(trimmed, "./kit ") {
		trimmed = "kit " + strings.TrimPrefix(trimmed, "./kit ")
	} else if trimmed == "./kit" {
		trimmed = "kit"
	}
	if hash := strings.Index(trimmed, "#"); hash >= 0 {
		trimmed = strings.TrimSpace(trimmed[:hash])
	}
	if trimmed == "kit" {
		return "kit", true
	}
	if !strings.HasPrefix(trimmed, "kit ") {
		return "", false
	}
	fields := strings.Fields(strings.TrimPrefix(trimmed, "kit "))
	if len(fields) == 0 {
		return "", true
	}
	var words []string
	for _, field := range fields {
		if !isCommandWord(field) {
			break
		}
		words = append(words, field)
		if len(words) >= 4 {
			break
		}
	}
	for n := len(words); n >= 1; n-- {
		name := strings.Join(words[:n], " ")
		if known[name] {
			return name, true
		}
	}
	return "", true
}

func isCommandWord(word string) bool {
	if word == "" || strings.HasPrefix(word, "-") || strings.HasPrefix(word, "<") || strings.HasPrefix(word, "[") {
		return false
	}
	for _, r := range word {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-' {
			continue
		}
		return false
	}
	return true
}

func printDocsLint(report docsLintReport) {
	status := "OK"
	if !report.OK {
		status = "FAIL"
	}
	fmt.Printf("root: %s\n", report.Root)
	fmt.Printf("status: %s\n", status)
	fmt.Printf("verification: %s\n", report.Verification)
	fmt.Printf("files_scanned: %d\n", report.FilesScanned)
	fmt.Printf("examples: %d\n", report.Examples)
	if len(report.Issues) == 0 {
		fmt.Println("issues: none")
		return
	}
	fmt.Println("issues:")
	for _, issue := range report.Issues {
		loc := issue.Path
		if issue.Line > 0 {
			loc = fmt.Sprintf("%s:%d", loc, issue.Line)
		}
		fmt.Printf("- %s %s\n", loc, issue.Message)
	}
}

func inventory(root string) {
	inv, err := kit.Scan(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "kit inventory: %v\n", err)
		os.Exit(1)
	}
	if err := jsonout.Print(inv); err != nil {
		fmt.Fprintf(os.Stderr, "kit inventory: %v\n", err)
		os.Exit(1)
	}
}

func summary(root string) {
	inv, err := kit.Scan(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "kit summary: %v\n", err)
		os.Exit(1)
	}
	kinds := map[string]int{}
	for _, file := range inv.Files {
		kinds[file.Kind]++
	}
	fmt.Printf("root: %s\n", inv.Root)
	fmt.Printf("files: %d\n", len(inv.Files))
	fmt.Printf("mod_dirs: %d\n", len(inv.ModDirs))
	for _, kind := range []string{"scenario", "record", "dat", "ai", "json", "text"} {
		if count := kinds[kind]; count > 0 {
			fmt.Printf("%s: %d\n", kind, count)
		}
	}
	if len(inv.ModDirs) > 0 {
		fmt.Println("mods:")
		for _, mod := range inv.ModDirs {
			fmt.Printf("- %s\n", mod.Path)
		}
	}
	if len(inv.Warnings) > 0 {
		fmt.Println("warnings:")
		for _, warning := range inv.Warnings {
			fmt.Printf("- %s\n", warning)
		}
	}
}

func runProject(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: kit project <inspect|snapshot|diff|lineage> ...")
		os.Exit(2)
	}
	switch args[0] {
	case "inspect":
		root := "."
		textOut := false
		opts := kit.ProjectInspectOptions{}
		for i := 1; i < len(args); i++ {
			switch args[i] {
			case "--text":
				textOut = true
			case "--json":
				textOut = false
			case "--limit":
				i++
				if i >= len(args) {
					die("kit project inspect", fmt.Errorf("--limit needs an integer"))
				}
				value, err := strconv.Atoi(args[i])
				if err != nil || value < 0 {
					die("kit project inspect", fmt.Errorf("--limit needs a non-negative integer"))
				}
				opts.Limit = value
			default:
				root = args[i]
			}
		}
		report, err := kit.InspectProject(root, opts)
		if err != nil {
			die("kit project inspect", err)
		}
		if textOut {
			printProjectInspect(report)
		} else {
			printJSON(report)
		}
	case "snapshot":
		runProjectSnapshot(args[1:])
	case "diff":
		runProjectDiff(args[1:])
	case "lineage":
		runProjectLineage(args[1:])
	default:
		fmt.Fprintln(os.Stderr, "usage: kit project <inspect|snapshot|diff|lineage> ...")
		os.Exit(2)
	}
}

func runProjectSnapshot(args []string) {
	root := "."
	outPath := ""
	textOut := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--out":
			i++
			if i >= len(args) {
				die("kit project snapshot", fmt.Errorf("--out requires a path"))
			}
			outPath = args[i]
		case "--text":
			textOut = true
		case "--json":
			textOut = false
		default:
			root = args[i]
		}
	}
	if outPath == "" {
		die("kit project snapshot", fmt.Errorf("--out is required"))
	}
	report, err := kit.WriteProjectSnapshot(root, outPath)
	if err != nil {
		die("kit project snapshot", err)
	}
	if textOut {
		fmt.Printf("root: %s\n", report.Root)
		fmt.Printf("output: %s\n", report.Output)
		fmt.Printf("verification: %s\n", report.Verification)
		fmt.Printf("files: %d\n", report.Inspect.FilesScanned)
	} else {
		printJSON(report)
	}
}

func runProjectDiff(args []string) {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: kit project diff <before-snapshot.json> <after-snapshot.json> [--limit N] [--text|--json]")
		os.Exit(2)
	}
	before, after := args[0], args[1]
	limit := 0
	textOut := false
	for i := 2; i < len(args); i++ {
		switch args[i] {
		case "--text":
			textOut = true
		case "--json":
			textOut = false
		case "--limit":
			i++
			if i >= len(args) {
				die("kit project diff", fmt.Errorf("--limit needs an integer"))
			}
			value, err := strconv.Atoi(args[i])
			if err != nil || value < 0 {
				die("kit project diff", fmt.Errorf("--limit needs a non-negative integer"))
			}
			limit = value
		default:
			die("kit project diff", fmt.Errorf("unknown option %q", args[i]))
		}
	}
	report, err := kit.DiffProjectSnapshots(before, after, limit)
	if err != nil {
		die("kit project diff", err)
	}
	if textOut {
		printProjectDiff(report)
	} else {
		printJSON(report)
	}
}

func runProjectLineage(args []string) {
	var inputs []string
	var outputs []string
	var notes []string
	manifestPath := ""
	toolName := ""
	textOut := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--input":
			i++
			if i >= len(args) {
				die("kit project lineage", fmt.Errorf("--input requires a path"))
			}
			inputs = append(inputs, args[i])
		case "--output":
			i++
			if i >= len(args) {
				die("kit project lineage", fmt.Errorf("--output requires a path"))
			}
			outputs = append(outputs, args[i])
		case "--tool":
			i++
			if i >= len(args) {
				die("kit project lineage", fmt.Errorf("--tool requires text"))
			}
			toolName = args[i]
		case "--note":
			i++
			if i >= len(args) {
				die("kit project lineage", fmt.Errorf("--note requires text"))
			}
			notes = append(notes, args[i])
		case "--manifest":
			i++
			if i >= len(args) {
				die("kit project lineage", fmt.Errorf("--manifest requires a path"))
			}
			manifestPath = args[i]
		case "--text":
			textOut = true
		case "--json":
			textOut = false
		default:
			die("kit project lineage", fmt.Errorf("unknown option %q", args[i]))
		}
	}
	command := append([]string{"kit", "project", "lineage"}, args...)
	report, err := kit.BuildArtifactLineage(inputs, outputs, kit.ArtifactLineageOptions{
		Tool:    toolName,
		Command: command,
		Notes:   notes,
	})
	if err != nil {
		die("kit project lineage", err)
	}
	if manifestPath != "" {
		if err := kit.WriteArtifactLineage(manifestPath, report); err != nil {
			die("kit project lineage", err)
		}
	}
	if textOut {
		printProjectLineage(report, manifestPath)
	} else {
		printJSON(report)
	}
}

func printProjectInspect(report kit.ProjectInspectReport) {
	fmt.Printf("root: %s\n", report.Root)
	fmt.Printf("verification: %s\n", report.Verification)
	fmt.Printf("files: scanned=%d shown=%d\n", report.FilesScanned, report.Shown)
	fmt.Println("roles:")
	for _, key := range sortedIntMapKeys(report.RoleCounts) {
		fmt.Printf("- %s: %d\n", key, report.RoleCounts[key])
	}
	fmt.Println("kinds:")
	for _, key := range sortedIntMapKeys(report.KindCounts) {
		fmt.Printf("- %s: %d\n", key, report.KindCounts[key])
	}
	if len(report.Recommendations) > 0 {
		fmt.Println("recommendations:")
		for _, rec := range report.Recommendations {
			fmt.Printf("- %s\n", rec)
		}
	}
	if len(report.Files) > 0 {
		fmt.Println("files:")
		for _, file := range report.Files {
			warn := ""
			if len(file.Warnings) > 0 {
				warn = " warnings=" + strings.Join(file.Warnings, "; ")
			}
			fmt.Printf("- [%s/%s] %s bytes=%d sha256=%s%s\n", file.Role, file.Kind, file.Path, file.SizeBytes, shortHash(file.SHA256), warn)
		}
	}
	if len(report.Warnings) > 0 {
		fmt.Println("warnings:")
		for _, warning := range report.Warnings {
			fmt.Printf("- %s\n", warning)
		}
	}
}

func printProjectDiff(report kit.ProjectDiffReport) {
	fmt.Printf("before: %s\n", report.Before)
	fmt.Printf("after: %s\n", report.After)
	fmt.Printf("verification: %s\n", report.Verification)
	fmt.Printf("same: %t\n", report.Same)
	fmt.Printf("changes: added=%d removed=%d modified=%d shown=%d", report.Summary.Added, report.Summary.Removed, report.Summary.Modified, report.Summary.Shown)
	if report.Summary.Limit > 0 {
		fmt.Printf(" limit=%d", report.Summary.Limit)
	}
	fmt.Println()
	if len(report.Changes) == 0 {
		fmt.Println("change_list: none")
		return
	}
	fmt.Println("change_list:")
	for _, change := range report.Changes {
		fields := ""
		if len(change.Fields) > 0 {
			fields = " fields=" + strings.Join(change.Fields, ",")
		}
		detail := ""
		switch change.Change {
		case "added":
			if change.After != nil {
				detail = fmt.Sprintf(" role=%s kind=%s bytes=%d sha256=%s", change.After.Role, change.After.Kind, change.After.SizeBytes, shortHash(change.After.SHA256))
			}
		case "removed":
			if change.Before != nil {
				detail = fmt.Sprintf(" role=%s kind=%s bytes=%d sha256=%s", change.Before.Role, change.Before.Kind, change.Before.SizeBytes, shortHash(change.Before.SHA256))
			}
		case "modified":
			if change.Before != nil && change.After != nil {
				detail = fmt.Sprintf(" role=%s->%s kind=%s->%s bytes=%d->%d sha256=%s->%s",
					change.Before.Role, change.After.Role, change.Before.Kind, change.After.Kind,
					change.Before.SizeBytes, change.After.SizeBytes,
					shortHash(change.Before.SHA256), shortHash(change.After.SHA256))
			}
		}
		fmt.Printf("- [%s] %s%s%s\n", change.Change, change.Path, fields, detail)
	}
}

func printProjectLineage(report kit.ArtifactLineageReport, manifestPath string) {
	fmt.Printf("ok: %t\n", report.OK)
	fmt.Printf("verification: %s\n", report.Verification)
	if report.Tool != "" {
		fmt.Printf("tool: %s\n", report.Tool)
	}
	if manifestPath != "" {
		fmt.Printf("manifest: %s\n", manifestPath)
	}
	fmt.Printf("inputs: %d\n", len(report.Inputs))
	for _, input := range report.Inputs {
		fmt.Printf("- %s kind=%s bytes=%d sha256=%s\n", input.Path, input.Kind, input.SizeBytes, shortHash(input.SHA256))
	}
	fmt.Printf("outputs: %d\n", len(report.Outputs))
	for _, output := range report.Outputs {
		fmt.Printf("- %s kind=%s bytes=%d sha256=%s\n", output.Path, output.Kind, output.SizeBytes, shortHash(output.SHA256))
	}
	if len(report.Notes) > 0 {
		fmt.Println("notes:")
		for _, note := range report.Notes {
			fmt.Printf("- %s\n", note)
		}
	}
	if len(report.Warnings) > 0 {
		fmt.Println("warnings:")
		for _, warning := range report.Warnings {
			fmt.Printf("- %s\n", warning)
		}
	}
}

func sortedIntMapKeys(values map[string]int) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func runRecipe(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: kit recipe <list|show|export> ...")
		os.Exit(2)
	}
	switch args[0] {
	case "list":
		runRecipeList(args[1:])
	case "show":
		runRecipeShow(args[1:])
	case "export":
		runRecipeExport(args[1:])
	default:
		fmt.Fprintln(os.Stderr, "usage: kit recipe <list|show|export> ...")
		os.Exit(2)
	}
}

func runRecipeList(args []string) {
	domain := ""
	textOut := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--domain":
			i++
			if i >= len(args) {
				die("kit recipe list", fmt.Errorf("--domain requires a value"))
			}
			domain = args[i]
		case "--text":
			textOut = true
		case "--json":
			textOut = false
		default:
			die("kit recipe list", fmt.Errorf("unknown option %q", args[i]))
		}
	}
	report, err := kit.RecipeTemplateSummaries(domain)
	if err != nil {
		die("kit recipe list", err)
	}
	if textOut {
		printRecipeList(report)
	} else {
		printJSON(report)
	}
}

func runRecipeShow(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: kit recipe show <name> [--recipe-only] [--text|--json]")
		os.Exit(2)
	}
	name := args[0]
	recipeOnly := false
	textOut := false
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--recipe-only":
			recipeOnly = true
		case "--text":
			textOut = true
		case "--json":
			textOut = false
		default:
			die("kit recipe show", fmt.Errorf("unknown option %q", args[i]))
		}
	}
	template, ok := kit.RecipeTemplateByName(name)
	if !ok {
		die("kit recipe show", fmt.Errorf("unknown recipe template %q", name))
	}
	if recipeOnly {
		payload, err := kit.RecipeTemplatePayload(template)
		if err != nil {
			die("kit recipe show", err)
		}
		printJSON(payload)
		return
	}
	if textOut {
		printRecipeTemplate(template)
	} else {
		printJSON(template)
	}
}

func runRecipeExport(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: kit recipe export <out-dir> [--domain scenario|dat|all] [--force] [--text|--json]")
		os.Exit(2)
	}
	outDir := args[0]
	domain := ""
	force := false
	textOut := false
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--domain":
			i++
			if i >= len(args) {
				die("kit recipe export", fmt.Errorf("--domain requires a value"))
			}
			domain = args[i]
		case "--force":
			force = true
		case "--text":
			textOut = true
		case "--json":
			textOut = false
		default:
			die("kit recipe export", fmt.Errorf("unknown option %q", args[i]))
		}
	}
	report, err := kit.ExportRecipeTemplates(outDir, domain, force)
	if err != nil {
		die("kit recipe export", err)
	}
	if textOut {
		printRecipeExport(report)
	} else {
		printJSON(report)
	}
}

func printRecipeList(report kit.RecipeTemplateListReport) {
	fmt.Printf("verification: %s\n", report.Verification)
	if report.Domain != "" {
		fmt.Printf("domain: %s\n", report.Domain)
	}
	fmt.Printf("templates: %d\n", report.Count)
	for _, template := range report.Templates {
		fmt.Printf("- %s [%s]: %s\n", template.Name, template.Domain, template.Summary)
		fmt.Printf("  command: %s\n", template.Command)
	}
}

func printRecipeTemplate(template kit.RecipeTemplate) {
	fmt.Printf("name: %s\n", template.Name)
	fmt.Printf("domain: %s\n", template.Domain)
	fmt.Printf("verification: %s\n", template.Verification)
	fmt.Printf("summary: %s\n", template.Summary)
	fmt.Printf("applies_to: %s\n", template.AppliesTo)
	fmt.Printf("command: %s\n", template.Command)
	if len(template.Notes) > 0 {
		fmt.Println("notes:")
		for _, note := range template.Notes {
			fmt.Printf("- %s\n", note)
		}
	}
	payload, err := kit.RecipeTemplatePayload(template)
	if err != nil {
		die("kit recipe show", err)
	}
	fmt.Println("recipe:")
	if err := jsonout.Print(payload); err != nil {
		die("kit recipe show", err)
	}
}

func printRecipeExport(report kit.RecipeTemplateExportReport) {
	fmt.Printf("verification: %s\n", report.Verification)
	fmt.Printf("output_dir: %s\n", report.OutputDir)
	if report.Domain != "" {
		fmt.Printf("domain: %s\n", report.Domain)
	}
	fmt.Printf("files: %d\n", report.Count)
	for _, file := range report.Files {
		suffix := ""
		if file.Overwritten {
			suffix = " overwritten"
		}
		fmt.Printf("- %s [%s] -> %s (%d bytes%s)\n", file.Name, file.Domain, file.Path, file.Bytes, suffix)
	}
}

func runFacts(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: kit facts <list|check> [--include-provisional] [--domain DOMAIN] [--text]")
		os.Exit(2)
	}
	includeProvisional := false
	textOut := false
	domain := ""
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--include-provisional":
			includeProvisional = true
		case "--text":
			textOut = true
		case "--json":
			textOut = false
		case "--domain":
			i++
			if i >= len(args) {
				die("kit facts", fmt.Errorf("--domain needs a value"))
			}
			domain = args[i]
		default:
			die("kit facts", fmt.Errorf("unknown option %q", args[i]))
		}
	}
	ledger, err := enginefacts.LoadDefault()
	if err != nil {
		die("kit facts", err)
	}
	switch args[0] {
	case "check":
		rows := ledger.Filter(includeProvisional, domain)
		report := struct {
			OK                 bool               `json:"ok"`
			SchemaVersion      int                `json:"schema_version"`
			Generated          string             `json:"generated"`
			IncludeProvisional bool               `json:"include_provisional"`
			Domain             string             `json:"domain,omitempty"`
			TotalFacts         int                `json:"total_facts"`
			Shown              int                `json:"shown"`
			Verification       string             `json:"verification"`
			Facts              []enginefacts.Fact `json:"facts,omitempty"`
		}{
			OK:                 true,
			SchemaVersion:      ledger.SchemaVersion,
			Generated:          ledger.Generated,
			IncludeProvisional: includeProvisional,
			Domain:             domain,
			TotalFacts:         len(ledger.Facts),
			Shown:              len(rows),
			Verification:       "structure_verified_fact_ledger_schema_and_required_evidence_fields",
			Facts:              rows,
		}
		if textOut {
			printFactsCheck(report.OK, report.SchemaVersion, report.Generated, report.TotalFacts, report.Shown, report.Verification)
		} else {
			printJSON(report)
		}
	case "list":
		rows := ledger.Filter(includeProvisional, domain)
		report := struct {
			SchemaVersion      int                `json:"schema_version"`
			Generated          string             `json:"generated"`
			IncludeProvisional bool               `json:"include_provisional"`
			Domain             string             `json:"domain,omitempty"`
			TotalFacts         int                `json:"total_facts"`
			Shown              int                `json:"shown"`
			Facts              []enginefacts.Fact `json:"facts"`
		}{
			SchemaVersion:      ledger.SchemaVersion,
			Generated:          ledger.Generated,
			IncludeProvisional: includeProvisional,
			Domain:             domain,
			TotalFacts:         len(ledger.Facts),
			Shown:              len(rows),
			Facts:              rows,
		}
		if textOut {
			printFactsList(report.Facts, report.TotalFacts, report.IncludeProvisional, report.Domain)
		} else {
			printJSON(report)
		}
	default:
		fmt.Fprintln(os.Stderr, "usage: kit facts <list|check> [--include-provisional] [--domain DOMAIN] [--text]")
		os.Exit(2)
	}
}

func runScen(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: kit scen <blank|info|check|describe|settings|field|strings|refs|delete-plan|delete|disconnect|effects|diff-effects|glossary|idioms|analyze|triggers|trigger-neighborhood|audit-player-coverage|mechanic|trigger-flow|units|map|terrain|palette-usage|regions|coverage|verify|lint|xs|deploycheck|diff|diff-triggers|bytediff|write-check|dump-body|plan|patch|shop-catalog|import-triggers|smoke|smoke-recipe> <file.aoe2scenario> [args...]")
		os.Exit(2)
	}
	if args[0] == "blank" {
		runScenarioBlank(args[1:])
		return
	}
	if args[0] == "shop-catalog" {
		runScenarioShopCatalog(args[1:])
		return
	}
	if args[0] == "diff-effects" {
		runScenarioEffectDiff(args[1:])
		return
	}
	if args[0] == "smoke-recipe" {
		opts, err := parseScenarioSmokeOptions(args[1:])
		if err != nil {
			die("kit scen smoke-recipe", err)
		}
		printJSON(scenario.SmokeRecipe(opts))
		return
	}
	if args[0] == "smoke" {
		if len(args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: kit scen smoke <in.aoe2scenario> <out.aoe2scenario> [--x N] [--y N] [--player N] [--unit N] [--terrain N] [--elevation N] [--layer N]")
			os.Exit(2)
		}
		opts, err := parseScenarioSmokeOptions(args[3:])
		if err != nil {
			die("kit scen smoke", err)
		}
		report, err := scenario.PatchSmokeRecipeFile(args[1], args[2], opts)
		if err != nil {
			die("kit scen smoke", err)
		}
		printJSON(report)
		return
	}
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: kit scen <blank|info|check|describe|settings|field|strings|refs|delete-plan|delete|disconnect|effects|diff-effects|glossary|idioms|analyze|triggers|trigger-neighborhood|audit-player-coverage|mechanic|trigger-flow|units|map|terrain|palette-usage|regions|coverage|verify|lint|xs|deploycheck|diff|diff-triggers|bytediff|write-check|dump-body|plan|patch|shop-catalog|import-triggers|smoke|smoke-recipe> <file.aoe2scenario> [args...]")
		os.Exit(2)
	}
	switch args[0] {
	case "import-triggers":
		if len(args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: kit scen import-triggers <src.aoe2scenario> <dst.aoe2scenario> [--select names/ids/ranges] [--prefix PREFIX] --out <file.aoe2scenario>")
			os.Exit(2)
		}
		opts := scenario.ImportTriggersOptions{}
		for i := 3; i < len(args); i++ {
			switch args[i] {
			case "--select":
				if i+1 >= len(args) {
					die("kit scen import-triggers", fmt.Errorf("--select needs names, ids, or ranges"))
				}
				opts.Select = append(opts.Select, args[i+1])
				i++
			case "--prefix":
				if i+1 >= len(args) {
					die("kit scen import-triggers", fmt.Errorf("--prefix needs text"))
				}
				opts.Prefix = args[i+1]
				i++
			case "--out":
				if i+1 >= len(args) {
					die("kit scen import-triggers", fmt.Errorf("--out needs a scenario path"))
				}
				opts.Out = args[i+1]
				i++
			default:
				die("kit scen import-triggers", fmt.Errorf("unexpected argument %q", args[i]))
			}
		}
		if opts.Out == "" {
			die("kit scen import-triggers", fmt.Errorf("--out is required; refusing implicit in-place overwrite"))
		}
		report, err := scenario.ImportTriggersFile(args[1], args[2], opts)
		if err != nil {
			die("kit scen import-triggers", err)
		}
		printJSON(report)
		return
	case "coverage":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: kit scen coverage <file.aoe2scenario> [--text|--json] | kit scen coverage --root DIR [--root DIR ...] [--text|--json] [--strict]")
			os.Exit(2)
		}
		textOut := false
		strict := false
		roots := []string{}
		input := ""
		for i := 1; i < len(args); i++ {
			switch args[i] {
			case "--root":
				i++
				if i >= len(args) {
					die("kit scen coverage", fmt.Errorf("--root needs a directory"))
				}
				roots = append(roots, args[i])
			case "--text":
				textOut = true
			case "--json":
				textOut = false
			case "--strict":
				strict = true
			default:
				if input != "" || len(roots) > 0 {
					die("kit scen coverage", fmt.Errorf("unexpected argument %q", args[i]))
				}
				input = args[i]
			}
		}
		if len(roots) > 0 {
			report, err := scenario.CoverageCorpusWithOptions(scenario.CoverageCorpusOptions{Strict: strict}, roots...)
			if textOut {
				printScenarioCoverageCorpus(report)
			} else {
				printJSON(report)
			}
			if err != nil {
				fmt.Fprintf(os.Stderr, "kit scen coverage: %v\n", err)
				os.Exit(1)
			}
			return
		}
		if input == "" {
			die("kit scen coverage", fmt.Errorf("a scenario path or --root DIR is required"))
		}
		report, err := scenario.CoverageFile(input)
		if err != nil {
			die("kit scen coverage", err)
		}
		if textOut {
			printScenarioCoverage(report)
		} else {
			printJSON(report)
		}
		return
	case "info", "inspect":
		info, err := scenarioInfoArg(args[1])
		if err != nil {
			die("kit scen", err)
		}
		printJSON(info)
	case "check":
		info, err := inspectScenarioArg(args[1])
		if err != nil {
			die("kit scen", err)
		}
		fmt.Printf("%s: %d bytes sha256=%s [OK]\n", info.Base, info.SizeBytes, info.SHA256)
	case "triggers":
		if len(args) > 2 {
			opts := scenario.TriggerSearchOptions{}
			conditionOpts := scenario.TriggerConditionsOptions{}
			conditions := false
			textOut := false
			for i := 2; i < len(args); i++ {
				switch args[i] {
				case "--text":
					textOut = true
				case "--json":
					textOut = false
				case "--conditions":
					conditions = true
				case "--grep":
					i++
					if i >= len(args) {
						die("kit scen triggers", fmt.Errorf("--grep needs text"))
					}
					opts.Grep = args[i]
				case "--effect":
					i++
					if i >= len(args) {
						die("kit scen triggers", fmt.Errorf("--effect needs an id or name"))
					}
					opts.EffectQuery = args[i]
				case "--message":
					i++
					if i >= len(args) {
						die("kit scen triggers", fmt.Errorf("--message needs text"))
					}
					opts.Message = args[i]
				case "--condition":
					i++
					if i >= len(args) {
						die("kit scen triggers", fmt.Errorf("--condition needs an id or name"))
					}
					opts.ConditionQuery = args[i]
				case "--unit-ref":
					value, err := parseNextInt(args, &i, "--unit-ref")
					if err != nil {
						die("kit scen triggers", err)
					}
					opts.UnitRef = &value
				case "--unit-type":
					value, err := parseNextInt(args, &i, "--unit-type")
					if err != nil {
						die("kit scen triggers", err)
					}
					opts.UnitType = &value
				case "--player":
					value, err := parseNextInt(args, &i, "--player")
					if err != nil {
						die("kit scen triggers", err)
					}
					opts.Player = &value
				case "--variable":
					value, err := parseNextInt(args, &i, "--variable")
					if err != nil {
						die("kit scen triggers", err)
					}
					opts.Variable = &value
				case "--area":
					i++
					if i >= len(args) {
						die("kit scen triggers", fmt.Errorf("--area needs x1,y1,x2,y2"))
					}
					area, err := parseIntList(args[i], "--area")
					if err != nil || len(area) != 4 {
						die("kit scen triggers", fmt.Errorf("--area must be x1,y1,x2,y2"))
					}
					opts.Area = area
				case "--limit":
					i++
					if i >= len(args) {
						die("kit scen triggers", fmt.Errorf("--limit needs a value"))
					}
					value, err := strconv.Atoi(args[i])
					if err != nil || value < 0 {
						die("kit scen triggers", fmt.Errorf("--limit must be a non-negative integer"))
					}
					opts.Limit = value
					conditionOpts.Limit = value
				case "--max-mb":
					i++
					if i >= len(args) {
						die("kit scen triggers", fmt.Errorf("--max-mb needs a value"))
					}
					value, err := strconv.Atoi(args[i])
					if err != nil || value < 0 {
						die("kit scen triggers", fmt.Errorf("--max-mb must be a non-negative integer"))
					}
					opts.MaxInflatedBytes = value * 1024 * 1024
					conditionOpts.MaxInflatedBytes = value * 1024 * 1024
				default:
					die("kit scen triggers", fmt.Errorf("unknown option %q", args[i]))
				}
			}
			if conditions {
				if opts.Grep != "" || opts.EffectQuery != "" || opts.Message != "" {
					die("kit scen triggers", fmt.Errorf("--conditions cannot be combined with --grep, --effect, or --message"))
				}
				report, err := scenario.TriggerConditionsFile(args[1], conditionOpts)
				if err != nil {
					die("kit scen triggers", err)
				}
				if textOut {
					printScenarioTriggerConditions(report)
					return
				}
				printJSON(report)
				return
			}
			report, err := scenario.TriggerSearchFile(args[1], opts)
			if err != nil {
				die("kit scen triggers", err)
			}
			if textOut {
				printScenarioTriggerSearch(report)
				return
			}
			printJSON(report)
			return
		}
		scen, err := scenario.Open(args[1])
		if err != nil {
			die("kit scen", err)
		}
		printJSON(scen.Triggers)
	case "trigger-neighborhood":
		opts, textOut, err := parseTriggerNeighborhoodArgs(args[2:])
		if err != nil {
			die("kit scen trigger-neighborhood", err)
		}
		report, err := scenario.TriggerNeighborhoodFile(args[1], opts)
		if err != nil {
			die("kit scen trigger-neighborhood", err)
		}
		if textOut {
			printTriggerNeighborhood(report)
		} else {
			printJSON(report)
		}
	case "audit-player-coverage":
		opts, textOut, err := parsePlayerCoverageArgs(args[2:])
		if err != nil {
			die("kit scen audit-player-coverage", err)
		}
		report, err := scenario.PlayerCoverageFile(args[1], opts)
		if err != nil {
			die("kit scen audit-player-coverage", err)
		}
		if textOut {
			printPlayerCoverage(report)
		} else {
			printJSON(report)
		}
	case "mechanic":
		opts, textOut, err := parseMechanicArgs(args[2:])
		if err != nil {
			die("kit scen mechanic", err)
		}
		report, err := scenario.MechanicFile(args[1], opts)
		if err != nil {
			die("kit scen mechanic", err)
		}
		if textOut {
			printMechanic(report)
		} else {
			printJSON(report)
		}
	case "trigger-flow":
		opts, textOut, err := parseTriggerFlowArgs(args[2:])
		if err != nil {
			die("kit scen trigger-flow", err)
		}
		report, err := scenario.TriggerFlowFile(args[1], opts)
		if err != nil {
			die("kit scen trigger-flow", err)
		}
		if textOut {
			printTriggerFlow(report)
		} else {
			printJSON(report)
		}
	case "strings":
		report, err := scenario.StringsFile(args[1])
		if err != nil {
			die("kit scen strings", err)
		}
		printJSON(report)
	case "settings", "players":
		textOut := false
		for i := 2; i < len(args); i++ {
			arg := args[i]
			switch arg {
			case "--text":
				textOut = true
			case "--json":
				textOut = false
			default:
				die("kit scen settings", fmt.Errorf("unknown option %q", arg))
			}
		}
		report, err := scenario.SettingsFile(args[1])
		if err != nil {
			die("kit scen settings", err)
		}
		if textOut {
			printScenarioSettings(report)
			return
		}
		printJSON(report)
	case "field":
		if len(args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: kit scen field <path> <file.aoe2scenario|--root DIR> [--strict]")
			os.Exit(2)
		}
		input := ""
		strict := false
		for i := 2; i < len(args); i++ {
			switch args[i] {
			case "--strict":
				strict = true
			case "--root":
				if i+1 >= len(args) {
					die("kit scen field", fmt.Errorf("--root requires a directory"))
				}
				i++
				if input != "" {
					die("kit scen field", fmt.Errorf("multiple inputs provided"))
				}
				input = args[i]
			default:
				if input != "" {
					die("kit scen field", fmt.Errorf("unexpected argument %q", args[i]))
				}
				input = args[i]
			}
		}
		if input == "" {
			die("kit scen field", fmt.Errorf("a scenario path or --root DIR is required"))
		}
		results, err := scenario.QueryFieldWithOptions(args[1], input, scenario.FieldQueryOptions{Strict: strict})
		fmt.Println("file\tversion\tstatus\tvalue")
		for _, result := range results {
			value := ""
			if result.Exists {
				switch typed := result.Value.(type) {
				case string:
					value = typed
				default:
					data, marshalErr := json.Marshal(typed)
					if marshalErr != nil {
						value = fmt.Sprint(typed)
					} else {
						value = string(data)
					}
				}
			}
			fmt.Printf("%s\t%s\t%s\t%s\n", result.File, result.Version, result.Status, value)
			if result.Error != "" && result.Status != "ok" {
				fmt.Fprintf(os.Stderr, "kit scen field: %s: %s\n", result.File, result.Error)
			}
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "kit scen field: %v\n", err)
			os.Exit(1)
		}
	case "refs", "references":
		opts := scenario.ReferenceOptions{}
		textOut := false
		for i := 2; i < len(args); i++ {
			switch args[i] {
			case "--kind":
				i++
				if i >= len(args) {
					die("kit scen refs", fmt.Errorf("--kind needs unit, trigger, variable, string, or all"))
				}
				opts.Kind = args[i]
			case "--id":
				i++
				if i >= len(args) {
					die("kit scen refs", fmt.Errorf("--id needs an integer target id"))
				}
				id, err := strconv.Atoi(args[i])
				if err != nil {
					die("kit scen refs", fmt.Errorf("--id needs an integer target id"))
				}
				opts.TargetID = &id
			case "--text":
				textOut = true
			default:
				die("kit scen refs", fmt.Errorf("unknown option %q", args[i]))
			}
		}
		report, err := scenario.ReferencesFile(args[1], opts)
		if err != nil {
			die("kit scen refs", err)
		}
		if textOut {
			printScenarioReferences(report)
			return
		}
		printJSON(report)
	case "delete-plan":
		if len(args) < 4 {
			fmt.Fprintln(os.Stderr, "usage: kit scen delete-plan <file.aoe2scenario> <unit|trigger|variable|string> <id> [--text]")
			fmt.Fprintln(os.Stderr, "       kit scen delete-plan <file.aoe2scenario> system-prefix <prefix> [--text]")
			fmt.Fprintln(os.Stderr, "       kit scen delete-plan <file.aoe2scenario> unit-caption <caption> [--player N] [--text]")
			fmt.Fprintln(os.Stderr, "       kit scen delete-plan <file.aoe2scenario> unit-caption-prefix <prefix> [--player N] [--text]")
			fmt.Fprintln(os.Stderr, "       kit scen delete-plan <file.aoe2scenario> unit-caption-contains <text> [--player N] [--text]")
			fmt.Fprintln(os.Stderr, "       kit scen delete-plan <file.aoe2scenario> unit-type <unit_const> [--player N] [--text]")
			fmt.Fprintln(os.Stderr, "       kit scen delete-plan <file.aoe2scenario> units-player <player> [--text]")
			fmt.Fprintln(os.Stderr, "       kit scen delete-plan <file.aoe2scenario> trigger-name <name> [--text]")
			fmt.Fprintln(os.Stderr, "       kit scen delete-plan <file.aoe2scenario> trigger-prefix <prefix> [--text]")
			fmt.Fprintln(os.Stderr, "       kit scen delete-plan <file.aoe2scenario> trigger-contains <text> [--text]")
			fmt.Fprintln(os.Stderr, "       kit scen delete-plan <file.aoe2scenario> variable-name <name> [--text]")
			fmt.Fprintln(os.Stderr, "       kit scen delete-plan <file.aoe2scenario> variable-prefix <prefix> [--text]")
			fmt.Fprintln(os.Stderr, "       kit scen delete-plan <file.aoe2scenario> variable-contains <text> [--text]")
			fmt.Fprintln(os.Stderr, "       kit scen delete-plan <file.aoe2scenario> string-text <exact_text> [--text]")
			fmt.Fprintln(os.Stderr, "       kit scen delete-plan <file.aoe2scenario> string-prefix <prefix> [--text]")
			fmt.Fprintln(os.Stderr, "       kit scen delete-plan <file.aoe2scenario> string-contains <text> [--text]")
			fmt.Fprintln(os.Stderr, "       kit scen delete-plan <file.aoe2scenario> units-area <x1,y1,x2,y2> [--player N] [--unit N] [--text]")
			fmt.Fprintln(os.Stderr, "       kit scen delete-plan <file.aoe2scenario> <effect|condition> <trigger_index>:<child_index> [--text]")
			fmt.Fprintln(os.Stderr, "       kit scen delete-plan <file.aoe2scenario> <effect-type|condition-type> <type_id_or_name> [--trigger N|--trigger-prefix PREFIX] [--text]")
			fmt.Fprintln(os.Stderr, "       kit scen delete-plan <file.aoe2scenario> effect-text-prefix <prefix> [--trigger N|--trigger-prefix PREFIX] [--text]")
			fmt.Fprintln(os.Stderr, "       kit scen delete-plan <file.aoe2scenario> effect-text-contains <text> [--trigger N|--trigger-prefix PREFIX] [--text]")
			os.Exit(2)
		}
		request, err := parseScenarioDeletePlanRequest(args[2], args[3])
		if err != nil {
			die("kit scen delete-plan", err)
		}
		textOut, err := parseScenarioDeleteOptions(&request, args[4:], true)
		if err != nil {
			die("kit scen delete-plan", err)
		}
		report, err := scenario.DeletePlanFile(args[1], request)
		if err != nil {
			die("kit scen delete-plan", err)
		}
		if textOut {
			printScenarioDeletePlan(report)
			return
		}
		printJSON(report)
	case "delete":
		if len(args) < 5 {
			fmt.Fprintln(os.Stderr, "usage: kit scen delete <in.aoe2scenario> <out.aoe2scenario> <unit|trigger|variable|string> <id>")
			fmt.Fprintln(os.Stderr, "       kit scen delete <in.aoe2scenario> <out.aoe2scenario> system-prefix <prefix>")
			fmt.Fprintln(os.Stderr, "       kit scen delete <in.aoe2scenario> <out.aoe2scenario> unit-caption <caption> [--player N]")
			fmt.Fprintln(os.Stderr, "       kit scen delete <in.aoe2scenario> <out.aoe2scenario> unit-caption-prefix <prefix> [--player N]")
			fmt.Fprintln(os.Stderr, "       kit scen delete <in.aoe2scenario> <out.aoe2scenario> unit-caption-contains <text> [--player N]")
			fmt.Fprintln(os.Stderr, "       kit scen delete <in.aoe2scenario> <out.aoe2scenario> unit-type <unit_const> [--player N]")
			fmt.Fprintln(os.Stderr, "       kit scen delete <in.aoe2scenario> <out.aoe2scenario> units-player <player>")
			fmt.Fprintln(os.Stderr, "       kit scen delete <in.aoe2scenario> <out.aoe2scenario> trigger-name <name>")
			fmt.Fprintln(os.Stderr, "       kit scen delete <in.aoe2scenario> <out.aoe2scenario> trigger-prefix <prefix>")
			fmt.Fprintln(os.Stderr, "       kit scen delete <in.aoe2scenario> <out.aoe2scenario> trigger-contains <text>")
			fmt.Fprintln(os.Stderr, "       kit scen delete <in.aoe2scenario> <out.aoe2scenario> variable-name <name>")
			fmt.Fprintln(os.Stderr, "       kit scen delete <in.aoe2scenario> <out.aoe2scenario> variable-prefix <prefix>")
			fmt.Fprintln(os.Stderr, "       kit scen delete <in.aoe2scenario> <out.aoe2scenario> variable-contains <text>")
			fmt.Fprintln(os.Stderr, "       kit scen delete <in.aoe2scenario> <out.aoe2scenario> string-text <exact_text>")
			fmt.Fprintln(os.Stderr, "       kit scen delete <in.aoe2scenario> <out.aoe2scenario> string-prefix <prefix>")
			fmt.Fprintln(os.Stderr, "       kit scen delete <in.aoe2scenario> <out.aoe2scenario> string-contains <text>")
			fmt.Fprintln(os.Stderr, "       kit scen delete <in.aoe2scenario> <out.aoe2scenario> units-area <x1,y1,x2,y2> [--player N] [--unit N]")
			fmt.Fprintln(os.Stderr, "       kit scen delete <in.aoe2scenario> <out.aoe2scenario> <effect|condition> <trigger_index>:<child_index>")
			fmt.Fprintln(os.Stderr, "       kit scen delete <in.aoe2scenario> <out.aoe2scenario> <effect-type|condition-type> <type_id_or_name> [--trigger N|--trigger-prefix PREFIX]")
			fmt.Fprintln(os.Stderr, "       kit scen delete <in.aoe2scenario> <out.aoe2scenario> effect-text-prefix <prefix> [--trigger N|--trigger-prefix PREFIX]")
			fmt.Fprintln(os.Stderr, "       kit scen delete <in.aoe2scenario> <out.aoe2scenario> effect-text-contains <text> [--trigger N|--trigger-prefix PREFIX]")
			os.Exit(2)
		}
		request, err := parseScenarioDeletePlanRequest(args[3], args[4])
		if err != nil {
			die("kit scen delete", err)
		}
		if _, err := parseScenarioDeleteOptions(&request, args[5:], false); err != nil {
			die("kit scen delete", err)
		}
		plan, err := scenario.DeletePlanFile(args[1], request)
		if err != nil {
			die("kit scen delete", err)
		}
		if (!plan.CanDelete && !plan.CanTombstone) || plan.SuggestedRecipe == nil {
			die("kit scen delete", fmt.Errorf("refusing unsupported delete: strategy=%s", plan.Strategy))
		}
		data, err := json.Marshal(plan.SuggestedRecipe)
		if err != nil {
			die("kit scen delete", fmt.Errorf("delete-plan recipe hint did not encode: %w", err))
		}
		var recipe scenario.Recipe
		if err := json.Unmarshal(data, &recipe); err != nil {
			die("kit scen delete", fmt.Errorf("delete-plan recipe hint did not decode: %w", err))
		}
		patch, err := scenario.PatchRecipeFile(args[1], args[2], recipe)
		if err != nil {
			die("kit scen delete", err)
		}
		printJSON(struct {
			Version      string                     `json:"version"`
			Request      scenario.DeletePlanRequest `json:"request"`
			DeletePlan   scenario.DeletePlanReport  `json:"delete_plan"`
			PatchReport  scenario.PatchReport       `json:"patch_report"`
			Verification string                     `json:"verification"`
		}{
			Version:      plan.Version,
			Request:      request,
			DeletePlan:   plan,
			PatchReport:  patch,
			Verification: "structure_verified_not_engine_verified",
		})
	case "disconnect":
		if len(args) < 5 {
			fmt.Fprintln(os.Stderr, "usage: kit scen disconnect <in.aoe2scenario> <out.aoe2scenario> <trigger|unit|variable|string> <id>")
			fmt.Fprintln(os.Stderr, "       kit scen disconnect <in.aoe2scenario> <out.aoe2scenario> unit-caption <caption> [--player N]")
			fmt.Fprintln(os.Stderr, "       kit scen disconnect <in.aoe2scenario> <out.aoe2scenario> unit-caption-prefix <prefix> [--player N]")
			fmt.Fprintln(os.Stderr, "       kit scen disconnect <in.aoe2scenario> <out.aoe2scenario> unit-caption-contains <text> [--player N]")
			fmt.Fprintln(os.Stderr, "       kit scen disconnect <in.aoe2scenario> <out.aoe2scenario> unit-type <unit_const> [--player N]")
			fmt.Fprintln(os.Stderr, "       kit scen disconnect <in.aoe2scenario> <out.aoe2scenario> units-player <player>")
			fmt.Fprintln(os.Stderr, "       kit scen disconnect <in.aoe2scenario> <out.aoe2scenario> trigger-name <name>")
			fmt.Fprintln(os.Stderr, "       kit scen disconnect <in.aoe2scenario> <out.aoe2scenario> trigger-prefix <prefix>")
			fmt.Fprintln(os.Stderr, "       kit scen disconnect <in.aoe2scenario> <out.aoe2scenario> trigger-contains <text>")
			fmt.Fprintln(os.Stderr, "       kit scen disconnect <in.aoe2scenario> <out.aoe2scenario> variable-name <name>")
			fmt.Fprintln(os.Stderr, "       kit scen disconnect <in.aoe2scenario> <out.aoe2scenario> variable-prefix <prefix>")
			fmt.Fprintln(os.Stderr, "       kit scen disconnect <in.aoe2scenario> <out.aoe2scenario> variable-contains <text>")
			fmt.Fprintln(os.Stderr, "       kit scen disconnect <in.aoe2scenario> <out.aoe2scenario> string-text <exact_text>")
			fmt.Fprintln(os.Stderr, "       kit scen disconnect <in.aoe2scenario> <out.aoe2scenario> string-prefix <prefix>")
			fmt.Fprintln(os.Stderr, "       kit scen disconnect <in.aoe2scenario> <out.aoe2scenario> string-contains <text>")
			fmt.Fprintln(os.Stderr, "       kit scen disconnect <in.aoe2scenario> <out.aoe2scenario> units-area <x1,y1,x2,y2> [--player N] [--unit N]")
			os.Exit(2)
		}
		disconnectKind := strings.ToLower(strings.TrimSpace(args[3]))
		var refs scenario.ReferenceReport
		var recipe scenario.Recipe
		var err error
		request := map[string]any{}
		if isScenarioDeletePlanAreaKind(disconnectKind) {
			deleteRequest, err := parseScenarioDeletePlanRequest(args[3], args[4])
			if err != nil {
				die("kit scen disconnect", err)
			}
			if _, err := parseScenarioDeleteOptions(&deleteRequest, args[5:], false); err != nil {
				die("kit scen disconnect", err)
			}
			file, err := scenario.Open(args[1])
			if err != nil {
				die("kit scen disconnect", err)
			}
			refs, recipe, err = file.UnitsAreaDisconnectRecipe(deleteRequest)
			if err != nil {
				die("kit scen disconnect", err)
			}
			disconnectKind = "units_area"
			request = map[string]any{
				"kind":    disconnectKind,
				"area_x1": floatPtrValue(deleteRequest.AreaX1),
				"area_y1": floatPtrValue(deleteRequest.AreaY1),
				"area_x2": floatPtrValue(deleteRequest.AreaX2),
				"area_y2": floatPtrValue(deleteRequest.AreaY2),
			}
			if deleteRequest.Player != nil {
				request["player"] = *deleteRequest.Player
			}
			if deleteRequest.UnitConst != nil {
				request["unit_const"] = *deleteRequest.UnitConst
			}
		} else if disconnectKind == "unit_caption" || disconnectKind == "unit-caption" || disconnectKind == "object_caption" || disconnectKind == "object-caption" || disconnectKind == "caption" {
			deleteRequest, err := parseScenarioDeletePlanRequest(args[3], args[4])
			if err != nil {
				die("kit scen disconnect", err)
			}
			if _, err := parseScenarioDeleteOptions(&deleteRequest, args[5:], false); err != nil {
				die("kit scen disconnect", err)
			}
			resolvedID, captionRefs, captionRecipe, err := scenario.UnitCaptionDisconnectRecipeFile(args[1], deleteRequest.TargetCaption, deleteRequest.Player)
			if err != nil {
				die("kit scen disconnect", err)
			}
			refs = captionRefs
			recipe = captionRecipe
			disconnectKind = "unit_caption"
			request = map[string]any{
				"kind":                  disconnectKind,
				"target_caption":        deleteRequest.TargetCaption,
				"resolved_reference_id": resolvedID,
			}
			if deleteRequest.Player != nil {
				request["player"] = *deleteRequest.Player
			}
		} else if disconnectKind == "unit_caption_prefix" || disconnectKind == "unit-caption-prefix" || disconnectKind == "object_caption_prefix" || disconnectKind == "object-caption-prefix" || disconnectKind == "caption_prefix" || disconnectKind == "caption-prefix" {
			deleteRequest, err := parseScenarioDeletePlanRequest(args[3], args[4])
			if err != nil {
				die("kit scen disconnect", err)
			}
			if _, err := parseScenarioDeleteOptions(&deleteRequest, args[5:], false); err != nil {
				die("kit scen disconnect", err)
			}
			resolvedIDs, captionRefs, captionRecipe, err := scenario.UnitCaptionPrefixDisconnectRecipeFile(args[1], deleteRequest.TargetPrefix, deleteRequest.Player)
			if err != nil {
				die("kit scen disconnect", err)
			}
			refs = captionRefs
			recipe = captionRecipe
			disconnectKind = "unit_caption_prefix"
			request = map[string]any{
				"kind":                   disconnectKind,
				"target_prefix":          deleteRequest.TargetPrefix,
				"resolved_reference_ids": resolvedIDs,
			}
			if deleteRequest.Player != nil {
				request["player"] = *deleteRequest.Player
			}
		} else if disconnectKind == "unit_caption_contains" || disconnectKind == "unit-caption-contains" || disconnectKind == "object_caption_contains" || disconnectKind == "object-caption-contains" || disconnectKind == "caption_contains" || disconnectKind == "caption-contains" {
			deleteRequest, err := parseScenarioDeletePlanRequest(args[3], args[4])
			if err != nil {
				die("kit scen disconnect", err)
			}
			if _, err := parseScenarioDeleteOptions(&deleteRequest, args[5:], false); err != nil {
				die("kit scen disconnect", err)
			}
			resolvedIDs, captionRefs, captionRecipe, err := scenario.UnitCaptionContainsDisconnectRecipeFile(args[1], deleteRequest.TargetText, deleteRequest.Player)
			if err != nil {
				die("kit scen disconnect", err)
			}
			refs = captionRefs
			recipe = captionRecipe
			disconnectKind = "unit_caption_contains"
			request = map[string]any{
				"kind":                   disconnectKind,
				"target_text":            deleteRequest.TargetText,
				"resolved_reference_ids": resolvedIDs,
			}
			if deleteRequest.Player != nil {
				request["player"] = *deleteRequest.Player
			}
		} else if isScenarioDeletePlanUnitTypeKind(disconnectKind) {
			deleteRequest, err := parseScenarioDeletePlanRequest(args[3], args[4])
			if err != nil {
				die("kit scen disconnect", err)
			}
			if _, err := parseScenarioDeleteOptions(&deleteRequest, args[5:], false); err != nil {
				die("kit scen disconnect", err)
			}
			resolvedIDs, unitRefs, unitRecipe, err := scenario.UnitTypeDisconnectRecipeFile(args[1], *deleteRequest.UnitConst, deleteRequest.Player)
			if err != nil {
				die("kit scen disconnect", err)
			}
			refs = unitRefs
			recipe = unitRecipe
			disconnectKind = "unit_type"
			request = map[string]any{
				"kind":                   disconnectKind,
				"unit_const":             *deleteRequest.UnitConst,
				"resolved_reference_ids": resolvedIDs,
			}
			if deleteRequest.Player != nil {
				request["player"] = *deleteRequest.Player
			}
		} else if isScenarioDeletePlanUnitsPlayerKind(disconnectKind) {
			if len(args) != 5 {
				fmt.Fprintln(os.Stderr, "usage: kit scen disconnect <in.aoe2scenario> <out.aoe2scenario> units-player <player>")
				os.Exit(2)
			}
			deleteRequest, err := parseScenarioDeletePlanRequest(args[3], args[4])
			if err != nil {
				die("kit scen disconnect", err)
			}
			resolvedIDs, unitRefs, unitRecipe, err := scenario.UnitsPlayerDisconnectRecipeFile(args[1], *deleteRequest.Player)
			if err != nil {
				die("kit scen disconnect", err)
			}
			refs = unitRefs
			recipe = unitRecipe
			disconnectKind = "units_player"
			request = map[string]any{
				"kind":                   disconnectKind,
				"player":                 *deleteRequest.Player,
				"resolved_reference_ids": resolvedIDs,
			}
		} else if disconnectKind == "trigger_name" || disconnectKind == "trigger-name" {
			if len(args) != 5 {
				fmt.Fprintln(os.Stderr, "usage: kit scen disconnect <in.aoe2scenario> <out.aoe2scenario> trigger-name <name>")
				os.Exit(2)
			}
			resolvedID, triggerRefs, triggerRecipe, err := scenario.TriggerNameDisconnectRecipeFile(args[1], args[4])
			if err != nil {
				die("kit scen disconnect", err)
			}
			refs = triggerRefs
			recipe = triggerRecipe
			disconnectKind = "trigger_name"
			request = map[string]any{
				"kind":           disconnectKind,
				"target_name":    args[4],
				"resolved_index": resolvedID,
			}
		} else if disconnectKind == "trigger_prefix" || disconnectKind == "trigger-prefix" {
			if len(args) != 5 {
				fmt.Fprintln(os.Stderr, "usage: kit scen disconnect <in.aoe2scenario> <out.aoe2scenario> trigger-prefix <prefix>")
				os.Exit(2)
			}
			resolvedIDs, triggerRefs, triggerRecipe, err := scenario.TriggerPrefixDisconnectRecipeFile(args[1], args[4])
			if err != nil {
				die("kit scen disconnect", err)
			}
			refs = triggerRefs
			recipe = triggerRecipe
			disconnectKind = "trigger_prefix"
			request = map[string]any{
				"kind":          disconnectKind,
				"target_prefix": args[4],
				"resolved_ids":  resolvedIDs,
			}
		} else if disconnectKind == "trigger_contains" || disconnectKind == "trigger-contains" || disconnectKind == "trigger_grep" || disconnectKind == "trigger-grep" {
			if len(args) != 5 {
				fmt.Fprintln(os.Stderr, "usage: kit scen disconnect <in.aoe2scenario> <out.aoe2scenario> trigger-contains <text>")
				os.Exit(2)
			}
			resolvedIDs, triggerRefs, triggerRecipe, err := scenario.TriggerContainsDisconnectRecipeFile(args[1], args[4])
			if err != nil {
				die("kit scen disconnect", err)
			}
			refs = triggerRefs
			recipe = triggerRecipe
			disconnectKind = "trigger_contains"
			request = map[string]any{
				"kind":         disconnectKind,
				"target_text":  args[4],
				"resolved_ids": resolvedIDs,
			}
		} else if disconnectKind == "variable_name" || disconnectKind == "variable-name" {
			if len(args) != 5 {
				fmt.Fprintln(os.Stderr, "usage: kit scen disconnect <in.aoe2scenario> <out.aoe2scenario> variable-name <name>")
				os.Exit(2)
			}
			resolvedID, variableRefs, variableRecipe, err := scenario.VariableNameDisconnectRecipeFile(args[1], args[4])
			if err != nil {
				die("kit scen disconnect", err)
			}
			refs = variableRefs
			recipe = variableRecipe
			disconnectKind = "variable_name"
			request = map[string]any{
				"kind":        disconnectKind,
				"target_name": args[4],
				"resolved_id": resolvedID,
			}
		} else if disconnectKind == "variable_prefix" || disconnectKind == "variable-prefix" {
			if len(args) != 5 {
				fmt.Fprintln(os.Stderr, "usage: kit scen disconnect <in.aoe2scenario> <out.aoe2scenario> variable-prefix <prefix>")
				os.Exit(2)
			}
			resolvedIDs, variableRefs, variableRecipe, err := scenario.VariablePrefixDisconnectRecipeFile(args[1], args[4])
			if err != nil {
				die("kit scen disconnect", err)
			}
			refs = variableRefs
			recipe = variableRecipe
			disconnectKind = "variable_prefix"
			request = map[string]any{
				"kind":          disconnectKind,
				"target_prefix": args[4],
				"resolved_ids":  resolvedIDs,
			}
		} else if disconnectKind == "variable_contains" || disconnectKind == "variable-contains" {
			if len(args) != 5 {
				fmt.Fprintln(os.Stderr, "usage: kit scen disconnect <in.aoe2scenario> <out.aoe2scenario> variable-contains <text>")
				os.Exit(2)
			}
			resolvedIDs, variableRefs, variableRecipe, err := scenario.VariableContainsDisconnectRecipeFile(args[1], args[4])
			if err != nil {
				die("kit scen disconnect", err)
			}
			refs = variableRefs
			recipe = variableRecipe
			disconnectKind = "variable_contains"
			request = map[string]any{
				"kind":         disconnectKind,
				"target_text":  args[4],
				"resolved_ids": resolvedIDs,
			}
		} else if disconnectKind == "string_text" || disconnectKind == "string-text" {
			if len(args) != 5 {
				fmt.Fprintln(os.Stderr, "usage: kit scen disconnect <in.aoe2scenario> <out.aoe2scenario> string-text <exact_text>")
				os.Exit(2)
			}
			resolvedID, stringRefs, stringRecipe, err := scenario.StringTextDisconnectRecipeFile(args[1], args[4])
			if err != nil {
				die("kit scen disconnect", err)
			}
			refs = stringRefs
			recipe = stringRecipe
			disconnectKind = "string_text"
			request = map[string]any{
				"kind":        disconnectKind,
				"target_text": args[4],
				"resolved_id": resolvedID,
			}
		} else if disconnectKind == "string_prefix" || disconnectKind == "string-prefix" {
			if len(args) != 5 {
				fmt.Fprintln(os.Stderr, "usage: kit scen disconnect <in.aoe2scenario> <out.aoe2scenario> string-prefix <prefix>")
				os.Exit(2)
			}
			resolvedIDs, stringRefs, stringRecipe, err := scenario.StringPrefixDisconnectRecipeFile(args[1], args[4])
			if err != nil {
				die("kit scen disconnect", err)
			}
			refs = stringRefs
			recipe = stringRecipe
			disconnectKind = "string_prefix"
			request = map[string]any{
				"kind":          disconnectKind,
				"target_prefix": args[4],
				"resolved_ids":  resolvedIDs,
			}
		} else if disconnectKind == "string_contains" || disconnectKind == "string-contains" {
			if len(args) != 5 {
				fmt.Fprintln(os.Stderr, "usage: kit scen disconnect <in.aoe2scenario> <out.aoe2scenario> string-contains <text>")
				os.Exit(2)
			}
			resolvedIDs, stringRefs, stringRecipe, err := scenario.StringContainsDisconnectRecipeFile(args[1], args[4])
			if err != nil {
				die("kit scen disconnect", err)
			}
			refs = stringRefs
			recipe = stringRecipe
			disconnectKind = "string_contains"
			request = map[string]any{
				"kind":         disconnectKind,
				"target_text":  args[4],
				"resolved_ids": resolvedIDs,
			}
		} else {
			if len(args) != 5 {
				fmt.Fprintln(os.Stderr, "usage: kit scen disconnect <in.aoe2scenario> <out.aoe2scenario> <trigger|unit|variable|string> <id>")
				fmt.Fprintln(os.Stderr, "       kit scen disconnect <in.aoe2scenario> <out.aoe2scenario> unit-caption <caption> [--player N]")
				fmt.Fprintln(os.Stderr, "       kit scen disconnect <in.aoe2scenario> <out.aoe2scenario> unit-caption-prefix <prefix> [--player N]")
				fmt.Fprintln(os.Stderr, "       kit scen disconnect <in.aoe2scenario> <out.aoe2scenario> unit-caption-contains <text> [--player N]")
				fmt.Fprintln(os.Stderr, "       kit scen disconnect <in.aoe2scenario> <out.aoe2scenario> unit-type <unit_const> [--player N]")
				fmt.Fprintln(os.Stderr, "       kit scen disconnect <in.aoe2scenario> <out.aoe2scenario> units-player <player>")
				fmt.Fprintln(os.Stderr, "       kit scen disconnect <in.aoe2scenario> <out.aoe2scenario> trigger-name <name>")
				fmt.Fprintln(os.Stderr, "       kit scen disconnect <in.aoe2scenario> <out.aoe2scenario> trigger-prefix <prefix>")
				fmt.Fprintln(os.Stderr, "       kit scen disconnect <in.aoe2scenario> <out.aoe2scenario> trigger-contains <text>")
				fmt.Fprintln(os.Stderr, "       kit scen disconnect <in.aoe2scenario> <out.aoe2scenario> variable-name <name>")
				fmt.Fprintln(os.Stderr, "       kit scen disconnect <in.aoe2scenario> <out.aoe2scenario> variable-prefix <prefix>")
				fmt.Fprintln(os.Stderr, "       kit scen disconnect <in.aoe2scenario> <out.aoe2scenario> variable-contains <text>")
				fmt.Fprintln(os.Stderr, "       kit scen disconnect <in.aoe2scenario> <out.aoe2scenario> string-text <exact_text>")
				fmt.Fprintln(os.Stderr, "       kit scen disconnect <in.aoe2scenario> <out.aoe2scenario> string-prefix <prefix>")
				fmt.Fprintln(os.Stderr, "       kit scen disconnect <in.aoe2scenario> <out.aoe2scenario> string-contains <text>")
				fmt.Fprintln(os.Stderr, "       kit scen disconnect <in.aoe2scenario> <out.aoe2scenario> units-area <x1,y1,x2,y2> [--player N] [--unit N]")
				os.Exit(2)
			}
			targetID, err := strconv.Atoi(args[4])
			if err != nil {
				die("kit scen disconnect", fmt.Errorf("invalid %s id %q: %w", disconnectKind, args[4], err))
			}
			switch disconnectKind {
			case "trigger":
				refs, recipe, err = scenario.TriggerDisconnectRecipeFile(args[1], targetID)
			case "unit", "object":
				disconnectKind = "unit"
				refs, recipe, err = scenario.UnitDisconnectRecipeFile(args[1], targetID)
			case "variable":
				refs, recipe, err = scenario.VariableDisconnectRecipeFile(args[1], targetID)
			case "string":
				refs, recipe, err = scenario.StringDisconnectRecipeFile(args[1], targetID)
			default:
				fmt.Fprintln(os.Stderr, "usage: kit scen disconnect <in.aoe2scenario> <out.aoe2scenario> <trigger|unit|variable|string> <id>")
				fmt.Fprintln(os.Stderr, "       kit scen disconnect <in.aoe2scenario> <out.aoe2scenario> unit-caption <caption> [--player N]")
				fmt.Fprintln(os.Stderr, "       kit scen disconnect <in.aoe2scenario> <out.aoe2scenario> unit-caption-prefix <prefix> [--player N]")
				fmt.Fprintln(os.Stderr, "       kit scen disconnect <in.aoe2scenario> <out.aoe2scenario> unit-type <unit_const> [--player N]")
				fmt.Fprintln(os.Stderr, "       kit scen disconnect <in.aoe2scenario> <out.aoe2scenario> units-player <player>")
				fmt.Fprintln(os.Stderr, "       kit scen disconnect <in.aoe2scenario> <out.aoe2scenario> trigger-name <name>")
				fmt.Fprintln(os.Stderr, "       kit scen disconnect <in.aoe2scenario> <out.aoe2scenario> trigger-prefix <prefix>")
				fmt.Fprintln(os.Stderr, "       kit scen disconnect <in.aoe2scenario> <out.aoe2scenario> variable-name <name>")
				fmt.Fprintln(os.Stderr, "       kit scen disconnect <in.aoe2scenario> <out.aoe2scenario> variable-prefix <prefix>")
				fmt.Fprintln(os.Stderr, "       kit scen disconnect <in.aoe2scenario> <out.aoe2scenario> string-text <exact_text>")
				fmt.Fprintln(os.Stderr, "       kit scen disconnect <in.aoe2scenario> <out.aoe2scenario> string-prefix <prefix>")
				fmt.Fprintln(os.Stderr, "       kit scen disconnect <in.aoe2scenario> <out.aoe2scenario> units-area <x1,y1,x2,y2> [--player N] [--unit N]")
				os.Exit(2)
			}
			request = map[string]any{"kind": disconnectKind, "id": targetID}
		}
		if err != nil {
			die("kit scen disconnect", err)
		}
		if refs.Summary.Total == 0 {
			die("kit scen disconnect", fmt.Errorf("refusing no-op disconnect: no %s references found", disconnectKind))
		}
		if len(recipe.Triggers) == 0 && len(recipe.Units) == 0 {
			die("kit scen disconnect", fmt.Errorf("refusing unsupported disconnect: %s references were found, but none are removable direct trigger rows, clearable placed-unit fields, or clearable trigger string-id fields", disconnectKind))
		}
		patch, err := scenario.PatchRecipeFile(args[1], args[2], recipe)
		if err != nil {
			die("kit scen disconnect", err)
		}
		printJSON(struct {
			Version          string                    `json:"version"`
			Request          map[string]any            `json:"request"`
			ReferenceSummary scenario.ReferenceSummary `json:"reference_summary"`
			PatchReport      scenario.PatchReport      `json:"patch_report"`
			Verification     string                    `json:"verification"`
		}{
			Version:          refs.Version,
			Request:          request,
			ReferenceSummary: refs.Summary,
			PatchReport:      patch,
			Verification:     "structure_verified_not_engine_verified",
		})
	case "effects":
		opts := scenario.EffectsOptions{}
		census := false
		where := ""
		whereLimit := 25
		maxScenarioMB := 0
		var triggerIndex *int
		textOut := false
		for i := 2; i < len(args); i++ {
			arg := args[i]
			switch arg {
			case "--raw":
				opts.IncludeRawFields = true
			case "--census":
				census = true
			case "--text":
				textOut = true
			case "--where":
				i++
				if i >= len(args) {
					die("kit scen effects", fmt.Errorf("--where needs an effect id or name"))
				}
				where = args[i]
			case "--trigger":
				value, err := parseNextInt(args, &i, "--trigger")
				if err != nil || value < 0 {
					die("kit scen effects", fmt.Errorf("--trigger must be a non-negative integer"))
				}
				triggerIndex = &value
			case "--limit":
				i++
				if i >= len(args) {
					die("kit scen effects", fmt.Errorf("--limit needs a value"))
				}
				value, err := strconv.Atoi(args[i])
				if err != nil || value < 0 {
					die("kit scen effects", fmt.Errorf("--limit must be a non-negative integer"))
				}
				whereLimit = value
			case "--max-mb":
				i++
				if i >= len(args) {
					die("kit scen effects", fmt.Errorf("--max-mb needs a value"))
				}
				value, err := strconv.Atoi(args[i])
				if err != nil || value < 0 {
					die("kit scen effects", fmt.Errorf("--max-mb must be a non-negative integer"))
				}
				maxScenarioMB = value
			default:
				die("kit scen effects", fmt.Errorf("unknown option %q", arg))
			}
		}
		if where != "" {
			whereOpts := scenario.EffectWhereOptions{Query: where, Limit: whereLimit, TriggerIndex: triggerIndex, IncludeRawFields: opts.IncludeRawFields}
			if maxScenarioMB > 0 {
				whereOpts.MaxInflatedBytes = maxScenarioMB * 1024 * 1024
			}
			report, err := scenario.EffectsWhereFile(args[1], whereOpts)
			if err != nil {
				die("kit scen effects", err)
			}
			if textOut {
				printScenarioEffectWhere(report)
				return
			}
			printJSON(report)
			return
		}
		if census {
			censusOpts := scenario.EffectsCensusOptions{}
			if maxScenarioMB > 0 {
				censusOpts.MaxInflatedBytes = maxScenarioMB * 1024 * 1024
			}
			report, err := scenario.EffectsCensusFileWithOptions(args[1], censusOpts)
			if err != nil {
				die("kit scen effects", err)
			}
			if textOut {
				printScenarioEffectsCensus(report)
				return
			}
			printJSON(report)
			return
		}
		scen, err := scenario.Open(args[1])
		if err != nil {
			die("kit scen effects", err)
		}
		report := scen.EffectsWithOptions(opts)
		if textOut {
			printScenarioEffects(report)
			return
		}
		printJSON(report)
	case "glossary":
		opts := scenario.ScenarioGlossaryOptions{Limit: 25}
		textOut := false
		for i := 2; i < len(args); i++ {
			switch args[i] {
			case "--text":
				textOut = true
			case "--limit":
				i++
				if i >= len(args) {
					die("kit scen glossary", fmt.Errorf("--limit needs a value"))
				}
				value, err := strconv.Atoi(args[i])
				if err != nil || value < 0 {
					die("kit scen glossary", fmt.Errorf("--limit must be a non-negative integer"))
				}
				opts.Limit = value
			case "--max-mb":
				i++
				if i >= len(args) {
					die("kit scen glossary", fmt.Errorf("--max-mb needs a value"))
				}
				value, err := strconv.Atoi(args[i])
				if err != nil || value < 0 {
					die("kit scen glossary", fmt.Errorf("--max-mb must be a non-negative integer"))
				}
				opts.MaxInflatedBytes = value * 1024 * 1024
			default:
				die("kit scen glossary", fmt.Errorf("unknown option %q", args[i]))
			}
		}
		report, err := scenario.ScenarioGlossaryFile(args[1], opts)
		if err != nil {
			die("kit scen glossary", err)
		}
		if textOut {
			printScenarioGlossary(report)
			return
		}
		printJSON(report)
	case "idioms":
		textOut := false
		for i := 2; i < len(args); i++ {
			arg := args[i]
			switch arg {
			case "--text":
				textOut = true
			case "--json":
				textOut = false
			default:
				die("kit scen idioms", fmt.Errorf("unknown option %q", arg))
			}
		}
		scen, err := scenario.Open(args[1])
		if err != nil {
			die("kit scen idioms", err)
		}
		report := scen.Idioms()
		if textOut {
			printScenarioIdioms(report)
			return
		}
		printJSON(report)
	case "analyze":
		scen, err := scenario.Open(args[1])
		if err != nil {
			die("kit scen analyze", err)
		}
		printJSON(scen.Analyze())
	case "units":
		named := false
		for _, arg := range args[2:] {
			switch arg {
			case "--named":
				named = true
			default:
				die("kit scen units", fmt.Errorf("unknown option %q", arg))
			}
		}
		scen, err := scenario.Open(args[1])
		if err != nil {
			// Fall back to a partial parse (read units, tolerate undecoded
			// later sections like Microsoft-campaign triggers).
			scen2, err2 := scenario.OpenWithOptions(args[1], scenario.ParseOptions{StopBeforeSection: "Triggers"})
			if err2 != nil {
				die("kit scen", err)
			}
			fmt.Fprintf(os.Stderr, "kit scen units: partial parse (triggers not decoded): %v\n", err)
			scen = scen2
		}
		if named {
			printJSON(namedScenarioUnits(scen.Units))
		} else {
			printJSON(scen.Units)
		}
	case "map":
		mapInfo, err := scenario.MapFile(args[1])
		if err != nil {
			die("kit scen map", err)
		}
		printJSON(mapInfo)
	case "terrain":
		opts := scenario.TerrainOptions{}
		textOut := false
		for i := 2; i < len(args); i++ {
			switch args[i] {
			case "--text":
				textOut = true
			case "--json":
				textOut = false
			case "--tiles":
				opts.IncludeTiles = true
			case "--id":
				i++
				if i >= len(args) {
					die("kit scen terrain", fmt.Errorf("--id needs a terrain id"))
				}
				id, err := strconv.Atoi(args[i])
				if err != nil || id < 0 {
					die("kit scen terrain", fmt.Errorf("--id must be a non-negative integer"))
				}
				opts.ID = &id
			default:
				die("kit scen terrain", fmt.Errorf("unknown option %q", args[i]))
			}
		}
		report, err := scenario.TerrainFile(args[1], opts)
		if err != nil {
			die("kit scen terrain", err)
		}
		if textOut {
			printScenarioTerrain(report)
			return
		}
		printJSON(report)
	case "palette-usage":
		datPath := ""
		textOut := false
		for i := 2; i < len(args); i++ {
			switch args[i] {
			case "--text":
				textOut = true
			case "--json":
				textOut = false
			case "--dat":
				i++
				if i >= len(args) {
					die("kit scen palette-usage", fmt.Errorf("--dat needs an empires*.dat path"))
				}
				datPath = args[i]
			default:
				die("kit scen palette-usage", fmt.Errorf("unknown option %q", args[i]))
			}
		}
		if datPath == "" {
			die("kit scen palette-usage", fmt.Errorf("missing required --dat <empires*.dat>"))
		}
		report, err := scenario.PaletteUsageFile(args[1], datPath)
		if err != nil {
			die("kit scen palette-usage", err)
		}
		if textOut {
			printScenarioPaletteUsage(report)
			return
		}
		printJSON(report)
	case "regions":
		report, err := scenario.GuessRegionsFile(args[1])
		if err != nil {
			die("kit scen regions", err)
		}
		printJSON(report)
	case "describe":
		describeScenario(args[1:])
	case "verify":
		scen, err := scenario.Open(args[1])
		if err != nil {
			die("kit scen", err)
		}
		if err := scen.VerifyRebuild(); err != nil {
			die("kit scen", err)
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
			Path:          args[1],
			Version:       scen.Version,
			HeaderBytes:   scen.HeaderBytes,
			InflatedBytes: scen.InflatedBytes,
			Sections:      scen.Sections,
			Triggers:      scen.Triggers,
			RebuildOK:     true,
		})
	case "lint":
		textOut := false
		includeProvisional := false
		loadSafety := false
		semanticInvariants := false
		datPath := ""
		for i := 2; i < len(args); i++ {
			arg := args[i]
			switch arg {
			case "--text":
				textOut = true
			case "--include-provisional":
				includeProvisional = true
			case "--load-safety":
				loadSafety = true
			case "--semantic":
				semanticInvariants = true
			case "--dat":
				i++
				if i >= len(args) || strings.HasPrefix(args[i], "--") {
					die("kit scen lint", fmt.Errorf("--dat requires a DAT path"))
				}
				datPath = args[i]
			default:
				die("kit scen lint", fmt.Errorf("unknown option %q", arg))
			}
		}
		report, err := scenario.LintFileWithOptions(args[1], scenario.LintOptions{IncludeProvisional: includeProvisional, LoadSafety: loadSafety, SemanticInvariants: semanticInvariants, DatPath: datPath})
		if err != nil {
			die("kit scen lint", err)
		}
		if textOut {
			printScenarioLint(report)
		} else {
			printJSON(report)
		}
		if !report.OK {
			os.Exit(1)
		}
	case "xs":
		if len(args) >= 2 && isScenarioXSSubcommand(args[1]) {
			runScenarioXSSubcommand(args[1:])
			return
		}
		opts := scenario.XSCensusOptions{}
		textOut := false
		for i := 2; i < len(args); i++ {
			switch args[i] {
			case "--text":
				textOut = true
			case "--json":
				textOut = false
			case "--deploy-tree":
				i++
				if i >= len(args) {
					die("kit scen xs", fmt.Errorf("--deploy-tree requires a path"))
				}
				opts.DeployTree = args[i]
			case "--strict-create-results":
				opts.CheckCreateResults = true
				opts.RequireXSCheck = true
			case "--require-xs-check":
				opts.RequireXSCheck = true
			case "--xs-check":
				i++
				if i >= len(args) || strings.HasPrefix(args[i], "--") {
					die("kit scen xs", fmt.Errorf("--xs-check requires a path"))
				}
				opts.XSCheckPath = args[i]
			case "--allow-missing-xs-check":
				opts.AllowMissingXSCheck = true
			case "--dat":
				i++
				if i >= len(args) || strings.HasPrefix(args[i], "--") {
					die("kit scen xs", fmt.Errorf("--dat requires a DAT path"))
				}
				opts.DatPath = args[i]
			default:
				die("kit scen xs", fmt.Errorf("unknown option %q", args[i]))
			}
		}
		report, err := scenario.XSCensusFile(args[1], opts)
		if err != nil {
			die("kit scen xs", err)
		}
		if textOut {
			printScenarioXSCensus(report)
		} else {
			printJSON(report)
		}
		if !report.OK {
			os.Exit(1)
		}
	case "deploycheck":
		if len(args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: kit scen deploycheck <file.aoe2scenario> <deploy-tree> [--text] [--include-provisional]")
			os.Exit(2)
		}
		textOut := false
		includeProvisional := false
		for _, arg := range args[3:] {
			switch arg {
			case "--text":
				textOut = true
			case "--json":
				textOut = false
			case "--include-provisional":
				includeProvisional = true
			default:
				die("kit scen deploycheck", fmt.Errorf("unknown option %q", arg))
			}
		}
		report, err := scenario.DeployCheckFileWithOptions(args[1], args[2], scenario.DeployCheckOptions{IncludeProvisional: includeProvisional})
		if err != nil {
			die("kit scen deploycheck", err)
		}
		if textOut {
			printScenarioDeployCheck(report)
		} else {
			printJSON(report)
		}
		if !report.OK {
			os.Exit(1)
		}
	case "diff":
		if len(args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: kit scen diff <before.aoe2scenario> <after.aoe2scenario> [--all-fields] [--text|--json]")
			os.Exit(2)
		}
		textOut := false
		allFields := false
		for _, arg := range args[3:] {
			switch arg {
			case "--all-fields":
				allFields = true
			case "--text":
				textOut = true
			case "--json":
				textOut = false
			default:
				die("kit scen diff", fmt.Errorf("unknown option %q", arg))
			}
		}
		report, err := scenario.DiffFilesWithOptions(args[1], args[2], scenario.DiffOptions{AllFields: allFields})
		if err != nil {
			die("kit scen diff", err)
		}
		if textOut {
			printScenarioDiff(report)
		} else {
			printJSON(report)
		}
	case "diff-triggers":
		if len(args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: kit scen diff-triggers <before.aoe2scenario> <after.aoe2scenario> [--limit N] [--text]")
			os.Exit(2)
		}
		opts, textOut, err := parseTriggerGraphDiffOptions(args[3:], "kit scen diff-triggers")
		if err != nil {
			die("kit scen diff-triggers", err)
		}
		report, err := buildScenarioTriggerGraphDiff(args[1], args[2], opts)
		if err != nil {
			die("kit scen diff-triggers", err)
		}
		if textOut {
			printTriggerGraphDiff(report)
		} else {
			printJSON(report)
		}
	case "bytediff":
		if len(args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: kit scen bytediff <before.aoe2scenario> <after.aoe2scenario> [--text|--json] [--include-noise]")
			os.Exit(2)
		}
		textOut := true
		opts := scenario.ByteDiffOptions{}
		for _, arg := range args[3:] {
			switch arg {
			case "--text":
				textOut = true
			case "--json":
				textOut = false
			case "--include-noise":
				opts.IncludeNoise = true
			default:
				die("kit scen bytediff", fmt.Errorf("unknown option %q", arg))
			}
		}
		report, err := scenario.ByteDiffFiles(args[1], args[2], opts)
		if err != nil {
			die("kit scen bytediff", err)
		}
		if textOut {
			printScenarioByteDiff(report)
		} else {
			printJSON(report)
		}
	case "write-check":
		runScenarioWriteCheck(args[1:])
	case "dump-body":
		if len(args) != 3 || args[1] != "--inflated" {
			fmt.Fprintln(os.Stderr, "usage: kit scen dump-body --inflated <file.aoe2scenario>")
			os.Exit(2)
		}
		body, err := scenario.InflatedBodyFile(args[2])
		if err != nil {
			die("kit scen dump-body", err)
		}
		if _, err := os.Stdout.Write(body); err != nil {
			die("kit scen dump-body", err)
		}
	case "plan":
		if len(args) != 4 || args[2] != "--recipe" {
			fmt.Fprintln(os.Stderr, "usage: kit scen plan <in.aoe2scenario> --recipe recipe.json")
			os.Exit(2)
		}
		recipe, err := scenario.LoadRecipe(args[3])
		if err != nil {
			die("kit scen", err)
		}
		plan, err := scenario.PlanRecipeFile(args[1], recipe)
		if err != nil {
			die("kit scen", err)
		}
		printJSON(plan)
	case "patch":
		const patchUsage = "usage: kit scen patch <in.aoe2scenario> <out.aoe2scenario> --recipe recipe.json [--text|--json]"
		if len(args) < 5 || args[3] != "--recipe" {
			fmt.Fprintln(os.Stderr, patchUsage)
			os.Exit(2)
		}
		text := false
		for _, arg := range args[5:] {
			switch arg {
			case "--text":
				text = true
			case "--json":
				text = false
			default:
				fmt.Fprintf(os.Stderr, "kit scen patch: unknown flag %q\n", arg)
				fmt.Fprintln(os.Stderr, patchUsage)
				os.Exit(2)
			}
		}
		recipe, err := scenario.LoadRecipe(args[4])
		if err != nil {
			die("kit scen", err)
		}
		report, err := scenario.PatchRecipeFile(args[1], args[2], recipe)
		if err != nil {
			die("kit scen", err)
		}
		if text {
			printScenarioPatch(report)
			return
		}
		printJSON(report)
	default:
		fmt.Fprintln(os.Stderr, "usage: kit scen <blank|info|check|describe|settings|field|strings|refs|delete-plan|delete|disconnect|effects|diff-effects|glossary|idioms|analyze|triggers|trigger-neighborhood|audit-player-coverage|mechanic|trigger-flow|units|map|terrain|palette-usage|regions|coverage|verify|lint|xs|deploycheck|diff|diff-triggers|bytediff|write-check|dump-body|plan|patch|shop-catalog|import-triggers|smoke|smoke-recipe> <file.aoe2scenario> [args...]")
		os.Exit(2)
	}
}

func runScenarioShopCatalog(args []string) {
	if len(args) == 1 && args[0] == "--example" {
		printJSON(scenario.ShopCatalogExample())
		return
	}
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: kit scen shop-catalog <catalog.json> --out recipe.json [--text|--json]")
		fmt.Fprintln(os.Stderr, "       kit scen shop-catalog --example")
		os.Exit(2)
	}
	catalogPath := args[0]
	outPath := ""
	textOut := false
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--out":
			if i+1 >= len(args) {
				die("kit scen shop-catalog", fmt.Errorf("--out needs a value"))
			}
			outPath = args[i+1]
			i++
		case "--text":
			textOut = true
		case "--json":
			textOut = false
		default:
			die("kit scen shop-catalog", fmt.Errorf("unknown option %q", args[i]))
		}
	}
	if outPath == "" {
		die("kit scen shop-catalog", fmt.Errorf("missing required --out recipe.json"))
	}
	catalog, err := scenario.LoadShopCatalogFile(catalogPath)
	if err != nil {
		die("kit scen shop-catalog", err)
	}
	report, err := scenario.ShopCatalogRecipe(catalog)
	if err != nil {
		die("kit scen shop-catalog", err)
	}
	data, err := json.MarshalIndent(report.Recipe, "", "  ")
	if err != nil {
		die("kit scen shop-catalog", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(outPath, data, 0644); err != nil {
		die("kit scen shop-catalog", err)
	}
	if textOut {
		printScenarioShopCatalogReport(report, outPath)
	} else {
		type outputReport struct {
			scenario.ShopCatalogReport
			RecipePath string `json:"recipe_path"`
		}
		printJSON(outputReport{ShopCatalogReport: report, RecipePath: outPath})
	}
}

func printScenarioShopCatalogReport(report scenario.ShopCatalogReport, outPath string) {
	fmt.Printf("recipe: %s\n", outPath)
	fmt.Printf("verification: %s\n", report.Verification)
	fmt.Printf("shops: %d\n", report.ShopCount)
	fmt.Printf("items: %d\n", report.ItemCount)
	fmt.Printf("cleanup: %t\n", report.Cleanup)
	for _, shop := range report.ShopSummaries {
		unitText := "existing"
		if shop.HasUnit {
			unitText = "created"
		}
		buttons := ""
		if shop.ButtonMin > 0 {
			buttons = fmt.Sprintf(" buttons=%d..%d", shop.ButtonMin, shop.ButtonMax)
		}
		fmt.Printf("- %s unit=%d player=%d items=%d %s%s\n", shop.Name, shop.ShopUnit, shop.Player, shop.ItemCount, unitText, buttons)
	}
	for _, warning := range report.Warnings {
		fmt.Printf("warning: %s\n", warning)
	}
}

type scenarioInfoReport struct {
	aoe2.FileInfo
	Version          string                 `json:"version"`
	HeaderBytes      int                    `json:"header_bytes"`
	CompressedBytes  int                    `json:"compressed_body_bytes"`
	InflatedBytes    int                    `json:"inflated_body_bytes"`
	Sections         int                    `json:"sections"`
	Triggers         int                    `json:"triggers"`
	Units            int                    `json:"units"`
	Variables        int                    `json:"variables"`
	Messages         map[string]string      `json:"messages"`
	Strings          int                    `json:"strings"`
	StringTable      int                    `json:"string_table"`
	EffectText       int                    `json:"effect_text"`
	MarkupSummary    scenario.MarkupSummary `json:"markup_summary"`
	VerificationNote string                 `json:"verification"`
}

type namedScenarioUnitInfo struct {
	NumberOfUnitSections int                    `json:"number_of_unit_sections"`
	NumberOfPlayers      int                    `json:"number_of_players"`
	Sections             []namedPlayerUnitsInfo `json:"sections"`
	Total                int                    `json:"total"`
}

type namedPlayerUnitsInfo struct {
	Player int                `json:"player"`
	Count  int                `json:"count"`
	Units  []namedUnitSummary `json:"units,omitempty"`
}

type namedUnitSummary struct {
	scenario.UnitSummary
	UnitName string `json:"unit_name,omitempty"`
}

func namedScenarioUnits(info *scenario.UnitInfo) *namedScenarioUnitInfo {
	if info == nil {
		return nil
	}
	out := &namedScenarioUnitInfo{
		NumberOfUnitSections: info.NumberOfUnitSections,
		NumberOfPlayers:      info.NumberOfPlayers,
		Total:                info.Total,
	}
	for _, section := range info.Sections {
		namedSection := namedPlayerUnitsInfo{Player: section.Player, Count: section.Count}
		for _, unit := range section.Units {
			namedSection.Units = append(namedSection.Units, namedUnitSummary{
				UnitSummary: unit,
				UnitName:    replay.UnitDisplayName(unit.UnitConst),
			})
		}
		out.Sections = append(out.Sections, namedSection)
	}
	return out
}

func printScenarioLint(report scenario.LintReport) {
	status := "OK"
	if !report.OK {
		status = "FAIL"
	}
	fmt.Printf("path: %s\n", report.Path)
	fmt.Printf("status: %s\n", status)
	fmt.Printf("summary: triggers=%d units=%d players=%d ai_files=%d\n", report.Summary.Triggers, report.Summary.Units, report.Summary.Players, report.Summary.AIFiles)
	if len(report.Issues) == 0 {
		fmt.Println("issues: none")
		return
	}
	fmt.Println("issues:")
	for _, issue := range report.Issues {
		fact := ""
		if issue.FactID != "" {
			fact = fmt.Sprintf(" fact=%s/%s verified=%s", issue.FactID, issue.FactTier, issue.VerifiedDate)
		}
		fmt.Printf("- [%s] %s: %s%s\n", issue.Severity, issue.Code, issue.Message, fact)
		if issue.Why != "" {
			fmt.Printf("  why: %s\n", issue.Why)
		}
		if issue.Fix != "" {
			fmt.Printf("  fix: %s\n", issue.Fix)
		}
	}
}

func printScenarioDeployCheck(report scenario.DeployCheckReport) {
	status := "OK"
	if !report.OK {
		status = "FAIL"
	}
	fmt.Printf("path: %s\n", report.Path)
	fmt.Printf("deploy_tree: %s\n", report.DeployTree)
	fmt.Printf("status: %s\n", status)
	fmt.Printf("verification: %s\n", report.Verification)
	fmt.Printf("xs: script_name=%q script_file_path=%q content_bytes=%d\n", report.XS.ScriptName, report.XS.ScriptFilePath, report.XS.ScriptFileContentBytes)
	if report.XS.ResolvedPath != "" {
		fmt.Printf("resolved_xs: %s rel=%s\n", report.XS.ResolvedPath, report.XS.ResolvedRelativePath)
	}
	if report.Analysis != nil {
		resolved, missing := 0, 0
		for _, include := range report.Analysis.Includes {
			if include.Status == "resolved" {
				resolved++
			} else {
				missing++
			}
		}
		fmt.Printf("analysis: files=%d includes=%d resolved=%d missing=%d declarations=%d cross_file_nonextern=%d\n",
			len(report.Analysis.Files), len(report.Analysis.Includes), resolved, missing, len(report.Analysis.Declarations), len(report.Analysis.CrossFileUses))
	}
	if len(report.Findings) == 0 {
		fmt.Println("findings: none")
		return
	}
	fmt.Println("findings:")
	for _, finding := range report.Findings {
		location := ""
		if finding.Path != "" {
			location = " path=" + finding.Path
			if finding.Line > 0 {
				location += fmt.Sprintf(":%d", finding.Line)
			}
		}
		fact := ""
		if finding.FactID != "" {
			fact = fmt.Sprintf(" fact=%s/%s verified=%s", finding.FactID, finding.FactTier, finding.VerifiedDate)
		}
		fmt.Printf("- [%s] %s%s fix=%q%s\n", finding.Severity, finding.What, location, finding.Fix, fact)
	}
}

func printScenarioXSCensus(report scenario.XSCensusReport) {
	status := "OK"
	if !report.OK {
		status = "FAIL"
	}
	fmt.Printf("path: %s\n", report.Path)
	fmt.Printf("status: %s\n", status)
	fmt.Printf("verification: %s\n", report.Verification)
	fmt.Printf("xs: script_name=%q script_file_path=%q content_bytes=%d\n", report.XS.ScriptName, report.XS.ScriptFilePath, report.XS.ScriptFileContentBytes)
	if report.ScriptContentEmbedded {
		fmt.Printf("script_content: embedded bytes=%d sha256=%s\n", report.XS.ScriptFileContentBytes, shortHash(report.XS.ScriptFileContentSHA256))
	}
	if len(report.EmbeddedCarriers) > 0 {
		fmt.Println("embedded_carriers:")
		for _, carrier := range report.EmbeddedCarriers {
			fmt.Printf("- trigger=%d effect=%d name=%q title=%q content_bytes=%d content_sha256=%s source_bytes=%d source_sha256=%s\n",
				carrier.TriggerIndex, carrier.EffectIndex, carrier.TriggerName, carrier.Title,
				carrier.ContentBytes, shortHash(carrier.ContentSHA256), carrier.SourceBytes, shortHash(carrier.SourceSHA256))
		}
	}
	if report.DeployTree != "" {
		fmt.Printf("deploy_tree: %s\n", report.DeployTree)
	}
	if report.XS.ResolvedPath != "" {
		fmt.Printf("resolved_xs: %s rel=%s\n", report.XS.ResolvedPath, report.XS.ResolvedRelativePath)
	}
	fmt.Printf("counts: script_content=%d embedded_carriers=%d script_calls=%d unique_calls=%d functions=%d includes=%d resolved=%d missing=%d declarations=%d cross_file_nonextern=%d findings=%d\n",
		report.Counts.ScriptContentAttachments, report.Counts.EmbeddedCarriers, report.Counts.ScriptCalls, report.Counts.UniqueCalledFunctions, report.Counts.Functions,
		report.Counts.Includes, report.Counts.ResolvedIncludes, report.Counts.MissingIncludes,
		report.Counts.Declarations, report.Counts.CrossFileNonExtern, report.Counts.Findings)
	if len(report.UniqueCalledFunctions) > 0 {
		fmt.Printf("called_functions: %s\n", strings.Join(report.UniqueCalledFunctions, ", "))
	}
	if len(report.Functions) > 0 {
		fmt.Println("functions:")
		limit := len(report.Functions)
		if limit > 20 {
			limit = 20
		}
		for _, fn := range report.Functions[:limit] {
			params := fn.Params
			if params == "" {
				params = "void"
			}
			fmt.Printf("- %s %s(%s) path=%s:%d\n", fn.ReturnType, fn.Name, params, fn.File, fn.Line)
		}
		if len(report.Functions) > limit {
			fmt.Printf("- ... %d more\n", len(report.Functions)-limit)
		}
	}
	if len(report.ScriptCalls) > 0 {
		fmt.Println("script_calls:")
		limit := len(report.ScriptCalls)
		if limit > 20 {
			limit = 20
		}
		for _, call := range report.ScriptCalls[:limit] {
			suffix := ""
			if len(call.Calls) > 0 {
				suffix = " calls=" + strings.Join(call.Calls, ",")
			}
			fmt.Printf("- trigger=%d effect=%d name=%q message=%q%s\n", call.TriggerIndex, call.EffectIndex, call.TriggerName, call.Message, suffix)
		}
		if len(report.ScriptCalls) > limit {
			fmt.Printf("- ... %d more\n", len(report.ScriptCalls)-limit)
		}
	}
	if len(report.Findings) == 0 {
		fmt.Println("findings: none")
		return
	}
	fmt.Println("findings:")
	for _, finding := range report.Findings {
		location := ""
		if finding.Path != "" {
			location = " path=" + finding.Path
			if finding.Line > 0 {
				location += fmt.Sprintf(":%d", finding.Line)
			}
		}
		fmt.Printf("- [%s] %s%s fix=%q\n", finding.Severity, finding.What, location, finding.Fix)
	}
}

type scenarioXSEmbedReport struct {
	Input        string                  `json:"input"`
	Output       string                  `json:"output"`
	XSPath       string                  `json:"xs_path"`
	Title        string                  `json:"title,omitempty"`
	PatchReport  scenario.PatchReport    `json:"patch_report"`
	XSCensus     scenario.XSCensusReport `json:"xs_census"`
	Verification string                  `json:"verification"`
}

type scenarioXSDeployReport struct {
	scenario.XSDeployReport
	DeployCheck *scenario.DeployCheckReport `json:"deploy_check,omitempty"`
}

type scenarioXSExtractReport struct {
	Input         string `json:"input"`
	Output        string `json:"output"`
	Source        string `json:"source"`
	CarrierIndex  int    `json:"carrier_index"`
	TriggerIndex  int    `json:"trigger_index"`
	EffectIndex   int    `json:"effect_index"`
	Title         string `json:"title,omitempty"`
	ContentBytes  int    `json:"content_bytes"`
	ContentSHA256 string `json:"content_sha256"`
	Verification  string `json:"verification"`
}

type scenarioXSCompareReport struct {
	Scenario                 string `json:"scenario"`
	XSPath                   string `json:"xs_path"`
	Source                   string `json:"source"`
	CarrierIndex             int    `json:"carrier_index"`
	TriggerIndex             int    `json:"trigger_index"`
	EffectIndex              int    `json:"effect_index"`
	Title                    string `json:"title,omitempty"`
	ExactMatch               bool   `json:"exact_match"`
	NormalizedMatch          bool   `json:"normalized_match"`
	ScenarioContentBytes     int    `json:"scenario_content_bytes"`
	ScenarioContentSHA256    string `json:"scenario_content_sha256"`
	FileBytes                int    `json:"file_bytes"`
	FileSHA256               string `json:"file_sha256"`
	NormalizedScenarioSHA256 string `json:"normalized_scenario_sha256"`
	NormalizedFileSHA256     string `json:"normalized_file_sha256"`
	Verification             string `json:"verification"`
}

func isScenarioXSSubcommand(arg string) bool {
	switch arg {
	case "attach", "deploy", "embed", "extract", "compare":
		return true
	default:
		return false
	}
}

func runScenarioXSSubcommand(args []string) {
	switch args[0] {
	case "attach":
		runScenarioXSAttach(args[1:])
	case "deploy":
		runScenarioXSDeploy(args[1:])
	case "embed":
		runScenarioXSEmbed(args[1:])
	case "extract":
		runScenarioXSExtract(args[1:])
	case "compare":
		runScenarioXSCompare(args[1:])
	default:
		die("kit scen xs", fmt.Errorf("unknown subcommand %q", args[0]))
	}
}

func runScenarioXSAttach(args []string) {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: kit scen xs attach <in.aoe2scenario> <out.aoe2scenario> --xs file.xs [--name Entry.xs] [--also-carrier] [--title TITLE] [--trigger-name NAME] [--text|--json]")
		os.Exit(2)
	}
	input, output := args[0], args[1]
	var xsPath, name, title, triggerName string
	alsoCarrier := false
	textOut := false
	for i := 2; i < len(args); i++ {
		switch args[i] {
		case "--xs":
			i++
			if i >= len(args) {
				die("kit scen xs attach", fmt.Errorf("--xs requires a path"))
			}
			xsPath = args[i]
		case "--name":
			i++
			if i >= len(args) {
				die("kit scen xs attach", fmt.Errorf("--name requires a filename"))
			}
			name = args[i]
		case "--also-carrier":
			alsoCarrier = true
		case "--title":
			i++
			if i >= len(args) {
				die("kit scen xs attach", fmt.Errorf("--title requires text"))
			}
			title = args[i]
		case "--trigger-name":
			i++
			if i >= len(args) {
				die("kit scen xs attach", fmt.Errorf("--trigger-name requires text"))
			}
			triggerName = args[i]
		case "--text":
			textOut = true
		case "--json":
			textOut = false
		default:
			die("kit scen xs attach", fmt.Errorf("unknown option %q", args[i]))
		}
	}
	if xsPath == "" {
		die("kit scen xs attach", fmt.Errorf("--xs is required"))
	}
	if name == "" {
		name = filepath.Base(xsPath)
	}
	mode := "attachment"
	if alsoCarrier {
		mode = "attachment_and_carrier"
	}
	recipe := scenario.Recipe{XS: &scenario.XSRecipe{
		Mode:               mode,
		Name:               name,
		ContentFile:        xsPath,
		CarrierTitle:       title,
		CarrierTriggerName: triggerName,
	}}
	patchReport, err := scenario.PatchRecipeFile(input, output, recipe)
	if err != nil {
		die("kit scen xs attach", err)
	}
	census, err := scenario.XSCensusFile(output, scenario.XSCensusOptions{})
	if err != nil {
		die("kit scen xs attach", err)
	}
	if title == "" {
		title = filepath.Base(xsPath)
	}
	report := scenarioXSEmbedReport{
		Input:        input,
		Output:       output,
		XSPath:       xsPath,
		Title:        title,
		PatchReport:  patchReport,
		XSCensus:     census,
		Verification: scenarioWriteVerificationText(),
	}
	if textOut {
		printScenarioXSAttach(report)
	} else {
		printJSON(report)
	}
}

func runScenarioXSDeploy(args []string) {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: kit scen xs deploy <file.aoe2scenario> <deploy-tree> [--name Entry.xs] [--carrier N|--trigger N] [--force] [--check] [--text|--json]")
		os.Exit(2)
	}
	input, deployTree := args[0], args[1]
	var name string
	var carrierIndex, triggerIndex *int
	force := false
	check := false
	textOut := false
	for i := 2; i < len(args); i++ {
		switch args[i] {
		case "--name":
			i++
			if i >= len(args) {
				die("kit scen xs deploy", fmt.Errorf("--name requires a filename"))
			}
			name = args[i]
		case "--carrier":
			value, err := parseNextInt(args, &i, "--carrier")
			if err != nil {
				die("kit scen xs deploy", err)
			}
			carrierIndex = &value
		case "--trigger":
			value, err := parseNextInt(args, &i, "--trigger")
			if err != nil {
				die("kit scen xs deploy", err)
			}
			triggerIndex = &value
		case "--force":
			force = true
		case "--check":
			check = true
		case "--text":
			textOut = true
		case "--json":
			textOut = false
		default:
			die("kit scen xs deploy", fmt.Errorf("unknown option %q", args[i]))
		}
	}
	if carrierIndex != nil && triggerIndex != nil {
		die("kit scen xs deploy", fmt.Errorf("--carrier cannot be combined with --trigger"))
	}
	deployed, err := scenario.DeployXSFile(input, scenario.XSDeployOptions{
		DeployTree:   deployTree,
		Name:         name,
		CarrierIndex: carrierIndex,
		TriggerIndex: triggerIndex,
		Force:        force,
	})
	if err != nil {
		die("kit scen xs deploy", err)
	}
	report := scenarioXSDeployReport{XSDeployReport: deployed}
	if check {
		deployCheck, err := scenario.DeployCheckFile(input, deployTree)
		if err != nil {
			die("kit scen xs deploy", err)
		}
		report.DeployCheck = &deployCheck
		if !deployCheck.OK {
			if textOut {
				printScenarioXSDeploy(report)
			} else {
				printJSON(report)
			}
			os.Exit(1)
		}
	}
	if textOut {
		printScenarioXSDeploy(report)
	} else {
		printJSON(report)
	}
}

func runScenarioXSEmbed(args []string) {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: kit scen xs embed <in.aoe2scenario> <out.aoe2scenario> --xs file.xs [--title TITLE] [--trigger-name NAME] [--replace-trigger N] [--append] [--keep-attachment] [--text|--json]")
		os.Exit(2)
	}
	input, output := args[0], args[1]
	var xsPath, title, triggerName string
	var replaceTrigger *int
	appendCarrier := false
	keepAttachment := false
	textOut := false
	for i := 2; i < len(args); i++ {
		switch args[i] {
		case "--xs":
			i++
			if i >= len(args) {
				die("kit scen xs embed", fmt.Errorf("--xs requires a path"))
			}
			xsPath = args[i]
		case "--title":
			i++
			if i >= len(args) {
				die("kit scen xs embed", fmt.Errorf("--title requires text"))
			}
			title = args[i]
		case "--trigger-name":
			i++
			if i >= len(args) {
				die("kit scen xs embed", fmt.Errorf("--trigger-name requires text"))
			}
			triggerName = args[i]
		case "--replace-trigger":
			value, err := parseNextInt(args, &i, "--replace-trigger")
			if err != nil {
				die("kit scen xs embed", err)
			}
			replaceTrigger = &value
		case "--append":
			appendCarrier = true
		case "--keep-attachment":
			keepAttachment = true
		case "--text":
			textOut = true
		case "--json":
			textOut = false
		default:
			die("kit scen xs embed", fmt.Errorf("unknown option %q", args[i]))
		}
	}
	if xsPath == "" {
		die("kit scen xs embed", fmt.Errorf("--xs is required"))
	}
	if appendCarrier && replaceTrigger != nil {
		die("kit scen xs embed", fmt.Errorf("--append cannot be combined with --replace-trigger"))
	}
	replace := !appendCarrier
	clearAttachment := !keepAttachment
	recipe := scenario.Recipe{XS: &scenario.XSRecipe{
		Mode:                "carrier",
		ContentFile:         xsPath,
		CarrierTitle:        title,
		CarrierTriggerName:  triggerName,
		CarrierTriggerIndex: replaceTrigger,
		ReplaceCarrier:      &replace,
		ClearAttachment:     &clearAttachment,
	}}
	patchReport, err := scenario.PatchRecipeFile(input, output, recipe)
	if err != nil {
		die("kit scen xs embed", err)
	}
	census, err := scenario.XSCensusFile(output, scenario.XSCensusOptions{})
	if err != nil {
		die("kit scen xs embed", err)
	}
	if title == "" {
		title = filepath.Base(xsPath)
	}
	report := scenarioXSEmbedReport{
		Input:        input,
		Output:       output,
		XSPath:       xsPath,
		Title:        title,
		PatchReport:  patchReport,
		XSCensus:     census,
		Verification: scenarioWriteVerificationText(),
	}
	if textOut {
		printScenarioXSEmbed(report)
	} else {
		printJSON(report)
	}
}

func runScenarioXSExtract(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: kit scen xs extract <file.aoe2scenario> --out file.xs [--carrier N|--trigger N] [--force] [--text|--json]")
		os.Exit(2)
	}
	input := args[0]
	var outPath string
	var carrierIndex, triggerIndex *int
	force := false
	textOut := false
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--out":
			i++
			if i >= len(args) {
				die("kit scen xs extract", fmt.Errorf("--out requires a path"))
			}
			outPath = args[i]
		case "--carrier":
			value, err := parseNextInt(args, &i, "--carrier")
			if err != nil {
				die("kit scen xs extract", err)
			}
			carrierIndex = &value
		case "--trigger":
			value, err := parseNextInt(args, &i, "--trigger")
			if err != nil {
				die("kit scen xs extract", err)
			}
			triggerIndex = &value
		case "--force":
			force = true
		case "--text":
			textOut = true
		case "--json":
			textOut = false
		default:
			die("kit scen xs extract", fmt.Errorf("unknown option %q", args[i]))
		}
	}
	if outPath == "" {
		die("kit scen xs extract", fmt.Errorf("--out is required"))
	}
	if carrierIndex != nil && triggerIndex != nil {
		die("kit scen xs extract", fmt.Errorf("--carrier cannot be combined with --trigger"))
	}
	if !force {
		if _, err := os.Stat(outPath); err == nil {
			die("kit scen xs extract", fmt.Errorf("%s already exists; pass --force to overwrite", outPath))
		} else if !errors.Is(err, os.ErrNotExist) {
			die("kit scen xs extract", err)
		}
	}
	census, err := scenario.XSCensusFile(input, scenario.XSCensusOptions{})
	if err != nil {
		die("kit scen xs extract", err)
	}
	source := "embedded_carrier"
	idx := -1
	trigger := -1
	effect := -1
	title := ""
	content := ""
	contentSHA := ""
	if carrierIndex == nil && triggerIndex == nil && census.XS.ScriptFileContentBytes > 0 {
		scen, err := scenario.Open(input)
		if err != nil {
			die("kit scen xs extract", err)
		}
		content = normalizeTextForCompare(scen.XSAttachmentContent())
		contentSHA = sha256Hex([]byte(content))
		source = "scenario_attachment"
		title = census.XS.ScriptFilePath
	} else {
		carrierIdx, carrier, err := selectScenarioXSCarrier(census, carrierIndex, triggerIndex)
		if err != nil {
			die("kit scen xs extract", err)
		}
		idx = carrierIdx
		trigger = carrier.TriggerIndex
		effect = carrier.EffectIndex
		title = carrier.Title
		content = carrier.Content
		contentSHA = carrier.ContentSHA256
	}
	if err := os.WriteFile(outPath, []byte(content), 0644); err != nil {
		die("kit scen xs extract", err)
	}
	report := scenarioXSExtractReport{
		Input:         input,
		Output:        outPath,
		Source:        source,
		CarrierIndex:  idx,
		TriggerIndex:  trigger,
		EffectIndex:   effect,
		Title:         title,
		ContentBytes:  len([]byte(content)),
		ContentSHA256: contentSHA,
		Verification:  "structure_verified_not_engine_verified",
	}
	if textOut {
		printScenarioXSExtract(report)
	} else {
		printJSON(report)
	}
}

func runScenarioXSCompare(args []string) {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: kit scen xs compare <file.aoe2scenario> <file.xs> [--carrier N|--trigger N] [--text|--json]")
		os.Exit(2)
	}
	input, xsPath := args[0], args[1]
	var carrierIndex, triggerIndex *int
	textOut := false
	for i := 2; i < len(args); i++ {
		switch args[i] {
		case "--carrier":
			value, err := parseNextInt(args, &i, "--carrier")
			if err != nil {
				die("kit scen xs compare", err)
			}
			carrierIndex = &value
		case "--trigger":
			value, err := parseNextInt(args, &i, "--trigger")
			if err != nil {
				die("kit scen xs compare", err)
			}
			triggerIndex = &value
		case "--text":
			textOut = true
		case "--json":
			textOut = false
		default:
			die("kit scen xs compare", fmt.Errorf("unknown option %q", args[i]))
		}
	}
	if carrierIndex != nil && triggerIndex != nil {
		die("kit scen xs compare", fmt.Errorf("--carrier cannot be combined with --trigger"))
	}
	census, err := scenario.XSCensusFile(input, scenario.XSCensusOptions{})
	if err != nil {
		die("kit scen xs compare", err)
	}
	source := "embedded_carrier"
	idx := -1
	trigger := -1
	effect := -1
	title := ""
	scenarioContent := ""
	if carrierIndex == nil && triggerIndex == nil && census.XS.ScriptFileContentBytes > 0 {
		scen, err := scenario.Open(input)
		if err != nil {
			die("kit scen xs compare", err)
		}
		scenarioContent = normalizeTextForCompare(scen.XSAttachmentContent())
		source = "scenario_attachment"
		title = census.XS.ScriptFilePath
	} else {
		carrierIdx, carrier, err := selectScenarioXSCarrier(census, carrierIndex, triggerIndex)
		if err != nil {
			die("kit scen xs compare", err)
		}
		idx = carrierIdx
		trigger = carrier.TriggerIndex
		effect = carrier.EffectIndex
		title = carrier.Title
		scenarioContent = carrier.Content
	}
	fileContent, err := os.ReadFile(xsPath)
	if err != nil {
		die("kit scen xs compare", err)
	}
	scenarioBytes := []byte(scenarioContent)
	carrierNorm := normalizeTextForCompare(scenarioContent)
	fileNorm := normalizeTextForCompare(string(fileContent))
	report := scenarioXSCompareReport{
		Scenario:                 input,
		XSPath:                   xsPath,
		Source:                   source,
		CarrierIndex:             idx,
		TriggerIndex:             trigger,
		EffectIndex:              effect,
		Title:                    title,
		ExactMatch:               string(scenarioBytes) == string(fileContent),
		NormalizedMatch:          carrierNorm == fileNorm,
		ScenarioContentBytes:     len(scenarioBytes),
		ScenarioContentSHA256:    sha256Hex(scenarioBytes),
		FileBytes:                len(fileContent),
		FileSHA256:               sha256Hex(fileContent),
		NormalizedScenarioSHA256: sha256Hex([]byte(carrierNorm)),
		NormalizedFileSHA256:     sha256Hex([]byte(fileNorm)),
		Verification:             "structure_verified_not_engine_verified",
	}
	if textOut {
		printScenarioXSCompare(report)
	} else {
		printJSON(report)
	}
	if !report.NormalizedMatch {
		os.Exit(1)
	}
}

func selectScenarioXSCarrier(report scenario.XSCensusReport, carrierIndex, triggerIndex *int) (int, scenario.XSEmbeddedCarrier, error) {
	if len(report.EmbeddedCarriers) == 0 {
		return 0, scenario.XSEmbeddedCarrier{}, fmt.Errorf("scenario has no embedded XS carriers")
	}
	if triggerIndex != nil {
		for idx, carrier := range report.EmbeddedCarriers {
			if carrier.TriggerIndex == *triggerIndex {
				return idx, carrier, nil
			}
		}
		return 0, scenario.XSEmbeddedCarrier{}, fmt.Errorf("no embedded XS carrier on trigger %d", *triggerIndex)
	}
	idx := 0
	if carrierIndex != nil {
		idx = *carrierIndex
	}
	if idx < 0 || idx >= len(report.EmbeddedCarriers) {
		return 0, scenario.XSEmbeddedCarrier{}, fmt.Errorf("carrier index %d out of range 0..%d", idx, len(report.EmbeddedCarriers)-1)
	}
	return idx, report.EmbeddedCarriers[idx], nil
}

func normalizeTextForCompare(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	text = strings.TrimRight(text, "\n")
	if text == "" {
		return ""
	}
	return text + "\n"
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func scenarioWriteVerificationText() string {
	return "structure_verified_not_engine_verified"
}

func printScenarioXSEmbed(report scenarioXSEmbedReport) {
	fmt.Printf("input: %s\n", report.Input)
	fmt.Printf("output: %s\n", report.Output)
	fmt.Printf("xs: %s\n", report.XSPath)
	fmt.Printf("verification: %s\n", report.Verification)
	fmt.Printf("triggers: %d -> %d units: %d -> %d rebuild_ok=%t invariant_ok=%t\n",
		report.PatchReport.TriggerCountBefore, report.PatchReport.TriggerCountAfter,
		report.PatchReport.UnitCountBefore, report.PatchReport.UnitCountAfter,
		report.PatchReport.RebuildOK, report.PatchReport.InvariantOK)
	if report.PatchReport.TimestampOfLastSave > 0 {
		fmt.Printf("timestamp_of_last_save: %d\n", report.PatchReport.TimestampOfLastSave)
	}
	fmt.Printf("embedded_carriers: %d\n", len(report.XSCensus.EmbeddedCarriers))
	for idx, carrier := range report.XSCensus.EmbeddedCarriers {
		fmt.Printf("- carrier=%d trigger=%d effect=%d title=%q content_bytes=%d content_sha256=%s source_sha256=%s\n",
			idx, carrier.TriggerIndex, carrier.EffectIndex, carrier.Title, carrier.ContentBytes,
			shortHash(carrier.ContentSHA256), shortHash(carrier.SourceSHA256))
	}
}

func printScenarioXSAttach(report scenarioXSEmbedReport) {
	fmt.Printf("input: %s\n", report.Input)
	fmt.Printf("output: %s\n", report.Output)
	fmt.Printf("xs: %s\n", report.XSPath)
	fmt.Printf("verification: %s\n", report.Verification)
	fmt.Printf("triggers: %d -> %d units: %d -> %d rebuild_ok=%t invariant_ok=%t\n",
		report.PatchReport.TriggerCountBefore, report.PatchReport.TriggerCountAfter,
		report.PatchReport.UnitCountBefore, report.PatchReport.UnitCountAfter,
		report.PatchReport.RebuildOK, report.PatchReport.InvariantOK)
	if report.PatchReport.TimestampOfLastSave > 0 {
		fmt.Printf("timestamp_of_last_save: %d\n", report.PatchReport.TimestampOfLastSave)
	}
	fmt.Printf("attachment: script_name=%q script_file_path=%q content_bytes=%d sha256=%s\n",
		report.XSCensus.XS.ScriptName, report.XSCensus.XS.ScriptFilePath,
		report.XSCensus.XS.ScriptFileContentBytes, shortHash(report.XSCensus.XS.ScriptFileContentSHA256))
	fmt.Printf("embedded_carriers: %d\n", len(report.XSCensus.EmbeddedCarriers))
}

func printScenarioXSDeploy(report scenarioXSDeployReport) {
	fmt.Printf("scenario: %s\n", report.Path)
	fmt.Printf("deploy_tree: %s\n", report.DeployTree)
	fmt.Printf("output: %s\n", report.Output)
	fmt.Printf("rel: %s\n", report.ResolvedRelativePath)
	fmt.Printf("source: %s name=%q bytes=%d sha256=%s\n", report.Source, report.Name, report.ContentBytes, shortHash(report.ContentSHA256))
	fmt.Printf("verification: %s\n", report.Verification)
	if report.DeployCheck != nil {
		status := "OK"
		if !report.DeployCheck.OK {
			status = "FAIL"
		}
		fmt.Printf("deploycheck: %s findings=%d\n", status, len(report.DeployCheck.Findings))
		for _, finding := range report.DeployCheck.Findings {
			fmt.Printf("- [%s] %s fix=%q\n", finding.Severity, finding.What, finding.Fix)
		}
	}
}

func printScenarioXSExtract(report scenarioXSExtractReport) {
	fmt.Printf("input: %s\n", report.Input)
	fmt.Printf("output: %s\n", report.Output)
	fmt.Printf("verification: %s\n", report.Verification)
	fmt.Printf("source: %s\n", report.Source)
	fmt.Printf("carrier: index=%d trigger=%d effect=%d title=%q content_bytes=%d sha256=%s\n",
		report.CarrierIndex, report.TriggerIndex, report.EffectIndex, report.Title,
		report.ContentBytes, shortHash(report.ContentSHA256))
}

func printScenarioXSCompare(report scenarioXSCompareReport) {
	status := "DIFF"
	if report.ExactMatch {
		status = "EXACT"
	} else if report.NormalizedMatch {
		status = "NORMALIZED_MATCH"
	}
	fmt.Printf("scenario: %s\n", report.Scenario)
	fmt.Printf("xs: %s\n", report.XSPath)
	fmt.Printf("status: %s\n", status)
	fmt.Printf("verification: %s\n", report.Verification)
	fmt.Printf("source: %s\n", report.Source)
	fmt.Printf("carrier: index=%d trigger=%d effect=%d title=%q\n", report.CarrierIndex, report.TriggerIndex, report.EffectIndex, report.Title)
	fmt.Printf("exact_match: %t\n", report.ExactMatch)
	fmt.Printf("normalized_match: %t\n", report.NormalizedMatch)
	fmt.Printf("scenario_content: bytes=%d sha256=%s normalized_sha256=%s\n",
		report.ScenarioContentBytes, shortHash(report.ScenarioContentSHA256), shortHash(report.NormalizedScenarioSHA256))
	fmt.Printf("file: bytes=%d sha256=%s normalized_sha256=%s\n",
		report.FileBytes, shortHash(report.FileSHA256), shortHash(report.NormalizedFileSHA256))
}

type scenarioWriteCheckReport struct {
	Before       string                   `json:"before"`
	After        string                   `json:"after"`
	OK           bool                     `json:"ok"`
	Verification aoe2.VerificationClaim   `json:"verification"`
	BeforeVerify *scenarioVerifySummary   `json:"before_verify,omitempty"`
	AfterVerify  *scenarioVerifySummary   `json:"after_verify,omitempty"`
	Lint         *scenario.LintReport     `json:"lint,omitempty"`
	Diff         *scenario.DiffReport     `json:"diff,omitempty"`
	XS           *scenario.XSCensusReport `json:"xs,omitempty"`
	Warnings     []string                 `json:"warnings,omitempty"`
	Errors       []string                 `json:"errors,omitempty"`
}

type scenarioVerifySummary struct {
	Path          string `json:"path"`
	Version       string `json:"version"`
	HeaderBytes   int    `json:"header_bytes"`
	InflatedBytes int    `json:"inflated_body_bytes"`
	Sections      int    `json:"sections"`
	Triggers      int    `json:"triggers"`
	Units         int    `json:"units"`
	RebuildOK     bool   `json:"rebuild_ok"`
	InvariantOK   bool   `json:"invariant_ok"`
	InvariantNote string `json:"invariant_note,omitempty"`
}

func runScenarioWriteCheck(args []string) {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: kit scen write-check <before.aoe2scenario> <after.aoe2scenario> [--text|--json] [--include-provisional]")
		os.Exit(2)
	}
	before, after := args[0], args[1]
	textOut := false
	includeProvisional := false
	for _, arg := range args[2:] {
		switch arg {
		case "--text":
			textOut = true
		case "--json":
			textOut = false
		case "--include-provisional":
			includeProvisional = true
		default:
			die("kit scen write-check", fmt.Errorf("unknown option %q", arg))
		}
	}
	report := scenarioWriteCheckReport{
		Before: before,
		After:  after,
		Verification: aoe2.VerificationClaim{
			Label:             "structure_verified_not_engine_verified",
			StructureVerified: true,
			EngineVerified:    false,
			Note:              "write-check composes parser rebuild, lint, diff, and XS census; editor/game load remains the engine oracle.",
		},
	}
	if summary, err := verifyScenarioSummary(before); err != nil {
		report.Errors = append(report.Errors, "before verify: "+err.Error())
	} else {
		report.BeforeVerify = summary
	}
	if summary, err := verifyScenarioSummary(after); err != nil {
		report.Errors = append(report.Errors, "after verify: "+err.Error())
	} else {
		report.AfterVerify = summary
		if !summary.InvariantOK {
			report.Errors = append(report.Errors, "after trigger invariant failed: "+summary.InvariantNote)
		}
	}
	if lint, err := scenario.LintFileWithOptions(after, scenario.LintOptions{IncludeProvisional: includeProvisional}); err != nil {
		report.Errors = append(report.Errors, "lint: "+err.Error())
	} else {
		report.Lint = &lint
		if !lint.OK {
			report.Errors = append(report.Errors, "lint failed")
		}
	}
	if diff, err := scenario.DiffFiles(before, after); err != nil {
		report.Errors = append(report.Errors, "diff: "+err.Error())
	} else {
		report.Diff = &diff
		if diff.Same {
			report.Warnings = append(report.Warnings, "before and after scenarios have no parsed diff")
		}
	}
	if xs, err := scenario.XSCensusFile(after, scenario.XSCensusOptions{}); err != nil {
		report.Errors = append(report.Errors, "xs census: "+err.Error())
	} else {
		report.XS = &xs
		if !xs.OK {
			report.Errors = append(report.Errors, "xs census failed")
		}
	}
	report.OK = len(report.Errors) == 0
	if textOut {
		printScenarioWriteCheck(report)
	} else {
		printJSON(report)
	}
	if !report.OK {
		os.Exit(1)
	}
}

func verifyScenarioSummary(path string) (*scenarioVerifySummary, error) {
	scen, err := scenario.Open(path)
	if err != nil {
		return nil, err
	}
	if err := scen.VerifyRebuild(); err != nil {
		return nil, err
	}
	summary := &scenarioVerifySummary{
		Path:          path,
		Version:       scen.Version,
		HeaderBytes:   scen.HeaderBytes,
		InflatedBytes: scen.InflatedBytes,
		Sections:      len(scen.Sections),
		RebuildOK:     true,
		InvariantOK:   true,
	}
	if scen.Triggers != nil {
		summary.Triggers = scen.Triggers.Count
		summary.InvariantOK = scen.Triggers.InvariantOK
		summary.InvariantNote = scen.Triggers.InvariantNote
	}
	if scen.Units != nil {
		summary.Units = scen.Units.Total
	}
	return summary, nil
}

func printScenarioWriteCheck(report scenarioWriteCheckReport) {
	status := "OK"
	if !report.OK {
		status = "FAIL"
	}
	fmt.Printf("before: %s\n", report.Before)
	fmt.Printf("after: %s\n", report.After)
	fmt.Printf("status: %s\n", status)
	fmt.Printf("verification: %s\n", report.Verification.Label)
	if report.BeforeVerify != nil {
		fmt.Printf("before_verify: version=%s rebuild_ok=%t triggers=%d units=%d\n",
			report.BeforeVerify.Version, report.BeforeVerify.RebuildOK, report.BeforeVerify.Triggers, report.BeforeVerify.Units)
	}
	if report.AfterVerify != nil {
		fmt.Printf("after_verify: version=%s rebuild_ok=%t invariant_ok=%t triggers=%d units=%d\n",
			report.AfterVerify.Version, report.AfterVerify.RebuildOK, report.AfterVerify.InvariantOK, report.AfterVerify.Triggers, report.AfterVerify.Units)
	}
	if report.Lint != nil {
		fmt.Printf("lint: ok=%t issues=%d triggers=%d units=%d\n", report.Lint.OK, len(report.Lint.Issues), report.Lint.Summary.Triggers, report.Lint.Summary.Units)
	}
	if report.Diff != nil {
		fmt.Printf("diff: same=%t changes=%d\n", report.Diff.Same, len(report.Diff.Changes))
	}
	if report.XS != nil {
		fmt.Printf("xs: ok=%t carriers=%d script_calls=%d findings=%d\n", report.XS.OK, report.XS.Counts.EmbeddedCarriers, report.XS.Counts.ScriptCalls, report.XS.Counts.Findings)
	}
	if len(report.Warnings) > 0 {
		fmt.Println("warnings:")
		for _, warning := range report.Warnings {
			fmt.Printf("- %s\n", warning)
		}
	}
	if len(report.Errors) > 0 {
		fmt.Println("errors:")
		for _, err := range report.Errors {
			fmt.Printf("- %s\n", err)
		}
	}
}

func printScenarioDiff(report scenario.DiffReport) {
	fmt.Printf("before: %s\n", report.Before)
	fmt.Printf("after: %s\n", report.After)
	fmt.Printf("same: %t\n", report.Same)
	if len(report.Changes) == 0 {
		fmt.Println("changes: none")
	} else {
		fmt.Println("changes:")
		for _, change := range report.Changes {
			detail := ""
			if change.Detail != "" {
				detail = " (" + change.Detail + ")"
			}
			fmt.Printf("- %s %s: %v -> %v%s\n", change.Kind, change.Field, change.Before, change.After, detail)
		}
	}
	if len(report.Fields) == 0 {
		return
	}
	fmt.Println("field_changes:")
	for _, field := range report.Fields {
		fmt.Printf("- %s: %s -> %s", field.Path, field.BeforeHex, field.AfterHex)
		if field.BeforeExists && field.AfterExists {
			fmt.Printf(" (%v -> %v)", field.BeforeValue, field.AfterValue)
		} else if field.BeforeExists {
			fmt.Printf(" (%v -> <missing>)", field.BeforeValue)
		} else {
			fmt.Printf(" (<missing> -> %v)", field.AfterValue)
		}
		fmt.Println()
	}
}

func printScenarioByteDiff(report scenario.ByteDiffReport) {
	fmt.Printf("before: %s\n", report.BeforePath)
	fmt.Printf("after: %s\n", report.AfterPath)
	fmt.Printf("inflated_bytes: %d -> %d\n", report.BeforeBytes, report.AfterBytes)
	fmt.Printf("MODE: %s", report.Mode)
	if report.LengthDelta != 0 {
		fmt.Printf(" (length delta = %+d bytes)", report.LengthDelta)
	}
	fmt.Println()
	fmt.Printf("first_divergence: %d", report.FirstDivergence)
	if report.FirstDivergence >= 0 {
		fmt.Printf(" (0x%x)", report.FirstDivergence)
	}
	fmt.Println()
	if report.Warning != "" {
		fmt.Printf("WARNING: %s\n", report.Warning)
	}
	fmt.Printf("compared_span: %d..%d\n", report.ComparedByteSpan.Start, report.ComparedByteSpan.End)
	if len(report.NoiseSuppressed) > 0 {
		fmt.Println("suppressed_noise:")
		for _, run := range report.NoiseSuppressed {
			fmt.Printf("- %s %d..%d (0x%x..0x%x)", run.Kind, run.Start, run.End, run.Start, run.End)
			if run.Detail != "" {
				fmt.Printf(": %s", run.Detail)
			}
			fmt.Println()
		}
	}
	if len(report.Changes) == 0 {
		fmt.Println("changes: none")
		return
	}
	fmt.Println("changes:")
	for _, change := range report.Changes {
		fmt.Printf("- offset %d (%s) len=%d bytes=%s -> %s ascii=%q -> %q\n",
			change.Start, change.StartHex, change.Length, change.BeforeHex, change.AfterHex, change.BeforeText, change.AfterText)
		if change.Decodes.Uint8 != nil {
			fmt.Printf("  uint8: %d -> %d\n", change.Decodes.Uint8.Before, change.Decodes.Uint8.After)
		}
		if change.Decodes.Int16LE != nil {
			fmt.Printf("  int16_le: %d -> %d\n", change.Decodes.Int16LE.Before, change.Decodes.Int16LE.After)
		}
		if change.Decodes.Int32LE != nil {
			fmt.Printf("  int32_le: %d -> %d\n", change.Decodes.Int32LE.Before, change.Decodes.Int32LE.After)
		}
		if change.Decodes.FieldLE != nil {
			fmt.Printf("  field_int32_le: %d -> %d\n", change.Decodes.FieldLE.Before, change.Decodes.FieldLE.After)
		}
	}
}

func printTriggerGraphDiff(report *triggergraph.DiffReport) {
	fmt.Printf("before: %s\n", report.Before)
	fmt.Printf("after: %s\n", report.After)
	fmt.Printf("verification: %s\n", report.Verification)
	fmt.Printf("same: %t\n", report.Same)
	fmt.Printf("sha256: %s -> %s\n", shortHash(report.BeforeSHA256), shortHash(report.AfterSHA256))
	fmt.Printf("counts: triggers %d -> %d, effects %d -> %d, conditions %d -> %d, messages %d -> %d\n",
		report.BeforeCounts.Triggers, report.AfterCounts.Triggers,
		report.BeforeCounts.Effects, report.AfterCounts.Effects,
		report.BeforeCounts.Conditions, report.AfterCounts.Conditions,
		report.BeforeCounts.Messages, report.AfterCounts.Messages)
	fmt.Printf("changes: total=%d shown=%d added=%d removed=%d modified=%d count_fields=%d",
		report.Summary.Total, report.Summary.Shown, report.Summary.Added, report.Summary.Removed, report.Summary.Modified, report.Summary.CountFields)
	if report.Summary.Limit > 0 {
		fmt.Printf(" limit=%d", report.Summary.Limit)
	}
	fmt.Println()
	for _, warning := range report.Warnings {
		fmt.Printf("warning: %s\n", warning)
	}
	if len(report.Changes) == 0 {
		fmt.Println("changes: none")
		return
	}
	for _, change := range report.Changes {
		location := ""
		if change.TriggerName != "" {
			location = fmt.Sprintf(" trigger=%d %q", change.TriggerIndex, change.TriggerName)
		} else if change.TriggerIndex != 0 || strings.HasPrefix(change.Kind, "trigger_") {
			location = fmt.Sprintf(" trigger=%d", change.TriggerIndex)
		}
		field := ""
		if change.Field != "" {
			field = " " + change.Field
		}
		detail := ""
		if change.Detail != "" {
			detail = " (" + change.Detail + ")"
		}
		fmt.Printf("- %s%s%s: %s -> %s%s\n", change.Kind, location, field, triggerDiffValue(change.Before), triggerDiffValue(change.After), detail)
	}
}

func triggerDiffValue(value any) string {
	if value == nil {
		return "<nil>"
	}
	if m, ok := value.(map[string]any); ok {
		parts := make([]string, 0, 6)
		for _, key := range []string{"name", "description", "short_description", "enabled", "looping", "effects", "conditions", "type", "message", "raw_sha256", "xs_function"} {
			v, exists := m[key]
			if !exists {
				continue
			}
			if key == "raw_sha256" {
				parts = append(parts, fmt.Sprintf("%s=%s", key, shortHash(fmt.Sprint(v))))
				continue
			}
			parts = append(parts, fmt.Sprintf("%s=%s", key, oneLine(fmt.Sprint(v), 96)))
		}
		if fields, ok := m["fields_prefix"]; ok {
			parts = append(parts, fmt.Sprintf("fields_prefix=%s", compactIntListText(fields)))
		}
		if fields, ok := m["fields"]; ok {
			parts = append(parts, fmt.Sprintf("fields=%s", compactIntListText(fields)))
		}
		if order, ok := m["effect_order"]; ok {
			parts = append(parts, fmt.Sprintf("effect_order=%s", compactIntListText(order)))
		}
		if order, ok := m["condition_order"]; ok {
			parts = append(parts, fmt.Sprintf("condition_order=%s", compactIntListText(order)))
		}
		if len(parts) > 0 {
			return "{" + strings.Join(parts, " ") + "}"
		}
	}
	return oneLine(fmt.Sprint(value), 160)
}

func compactIntListText(value any) string {
	switch list := value.(type) {
	case []int:
		return compactInts(list)
	case []any:
		ints := make([]int, 0, len(list))
		for _, item := range list {
			switch n := item.(type) {
			case int:
				ints = append(ints, n)
			case int64:
				ints = append(ints, int(n))
			case float64:
				ints = append(ints, int(n))
			default:
				return fmt.Sprintf("len=%d", len(list))
			}
		}
		return compactInts(ints)
	default:
		return fmt.Sprint(value)
	}
}

func compactInts(values []int) string {
	if len(values) <= 10 {
		return fmt.Sprint(values)
	}
	return fmt.Sprintf("len=%d head=%v tail=%v", len(values), values[:6], values[len(values)-3:])
}

func printScenarioEffectWhere(report scenario.EffectWhereReport) {
	fmt.Printf("path: %s\n", report.Path)
	fmt.Printf("verification: %s\n", report.Verification)
	fmt.Printf("scenario: version=%s inflated_bytes=%d compressed_bytes=%d\n", report.Version, report.InflatedBytes, report.CompressedBytes)
	fmt.Printf("query: %q total_matches=%d shown=%d limit=%d\n", report.Query, report.TotalMatches, len(report.Matches), report.Limit)
	for _, effect := range report.Matches {
		fmt.Printf("- trigger=%d effect=%d %s\n", effect.TriggerIndex, effect.EffectIndex, scenarioEffectSummaryText(effect))
		if effect.TriggerName != "" {
			fmt.Printf("  trigger_name: %s\n", effect.TriggerName)
		}
		if effect.Text != "" {
			fmt.Printf("  text: %s\n", oneLine(effect.Text, 180))
		}
		if len(effect.KnownFields) > 0 {
			fmt.Printf("  known_fields: %s\n", triggerDiffValue(map[string]any(effect.KnownFields)))
		}
		if len(effect.RawFields) > 0 {
			fmt.Printf("  raw_fields: %s\n", compactInts(effect.RawFields))
		}
	}
}

func runScenarioEffectDiff(args []string) {
	usage := "usage: kit scen diff-effects <before.aoe2scenario> <after.aoe2scenario> --where EFFECT [--trigger N] [--limit N] [--max-mb N] [--raw] [--text|--json]"
	if len(args) < 3 {
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(2)
	}
	opts := scenario.EffectWhereOptions{Limit: 25}
	textOut := false
	for i := 2; i < len(args); i++ {
		switch args[i] {
		case "--where":
			i++
			if i >= len(args) {
				die("kit scen diff-effects", fmt.Errorf("--where needs an effect id or name"))
			}
			opts.Query = args[i]
		case "--trigger":
			value, err := parseNextInt(args, &i, "--trigger")
			if err != nil || value < 0 {
				die("kit scen diff-effects", fmt.Errorf("--trigger must be a non-negative integer"))
			}
			opts.TriggerIndex = &value
		case "--limit":
			value, err := parseNextInt(args, &i, "--limit")
			if err != nil || value < 0 {
				die("kit scen diff-effects", fmt.Errorf("--limit must be a non-negative integer"))
			}
			opts.Limit = value
		case "--max-mb":
			value, err := parseNextInt(args, &i, "--max-mb")
			if err != nil || value < 0 {
				die("kit scen diff-effects", fmt.Errorf("--max-mb must be a non-negative integer"))
			}
			opts.MaxInflatedBytes = value * 1024 * 1024
		case "--raw":
			opts.IncludeRawFields = true
		case "--text":
			textOut = true
		case "--json":
			textOut = false
		default:
			die("kit scen diff-effects", fmt.Errorf("unknown option %q", args[i]))
		}
	}
	if strings.TrimSpace(opts.Query) == "" {
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(2)
	}
	report, err := scenario.DiffEffectsFile(args[0], args[1], opts)
	if err != nil {
		die("kit scen diff-effects", err)
	}
	if textOut {
		printScenarioEffectDiff(report)
		return
	}
	printJSON(report)
}

func printScenarioEffectDiff(report scenario.EffectDiffReport) {
	fmt.Printf("before: %s\nafter: %s\nverification: %s\n", report.Before, report.After, report.Verification)
	fmt.Printf("query: %q before=%d after=%d compared=%d modified=%d added=%d removed=%d\n", report.Query, report.BeforeMatches, report.AfterMatches, report.Compared, report.Modified, report.Added, report.Removed)
	for _, change := range report.Changes {
		fmt.Printf("- ordinal=%d before=trigger %d effect %d after=trigger %d effect %d\n", change.Ordinal, change.BeforeTrigger, change.BeforeEffect, change.AfterTrigger, change.AfterEffect)
		keys := make([]string, 0, len(change.Fields))
		for key := range change.Fields {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			delta := change.Fields[key]
			fmt.Printf("  %s: %s -> %s\n", key, triggerDiffValue(delta.Before), triggerDiffValue(delta.After))
		}
	}
}

func runScenarioBlank(args []string) {
	usage := "usage: kit scen blank <out.aoe2scenario> [--size N] [--width N] [--height N] [--players N] [--human-slots N] [--timestamp UNIX] [--no-starters] [--dummy-unit UNIT_ID] [--dummy-origin X,Y] [--dummy-spacing N] [--no-conquest] [--closeout-seconds N] [--closeout-player N] [--keep-seed-triggers] [--text]"
	usage += "\nBarracks starters are included by default. Use --no-starters for an intentionally empty canvas."
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(2)
	}
	if args[0] == "--help" || args[0] == "-h" || args[0] == "help" {
		fmt.Println(usage)
		return
	}
	opts := scenario.BlankOptions{PlayerCount: 1, HumanSlots: 1}
	textOut := false
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--players":
			i++
			if i >= len(args) {
				die("kit scen blank", fmt.Errorf("--players needs a value"))
			}
			value, err := strconv.Atoi(args[i])
			if err != nil {
				die("kit scen blank", fmt.Errorf("--players needs an integer"))
			}
			opts.PlayerCount = value
		case "--size":
			i++
			if i >= len(args) {
				die("kit scen blank", fmt.Errorf("--size needs a value"))
			}
			value, err := strconv.Atoi(args[i])
			if err != nil {
				die("kit scen blank", fmt.Errorf("--size needs an integer"))
			}
			opts.MapWidth = value
			opts.MapHeight = value
		case "--width":
			i++
			if i >= len(args) {
				die("kit scen blank", fmt.Errorf("--width needs a value"))
			}
			value, err := strconv.Atoi(args[i])
			if err != nil {
				die("kit scen blank", fmt.Errorf("--width needs an integer"))
			}
			opts.MapWidth = value
		case "--height":
			i++
			if i >= len(args) {
				die("kit scen blank", fmt.Errorf("--height needs a value"))
			}
			value, err := strconv.Atoi(args[i])
			if err != nil {
				die("kit scen blank", fmt.Errorf("--height needs an integer"))
			}
			opts.MapHeight = value
		case "--human-slots":
			i++
			if i >= len(args) {
				die("kit scen blank", fmt.Errorf("--human-slots needs a value"))
			}
			value, err := strconv.Atoi(args[i])
			if err != nil {
				die("kit scen blank", fmt.Errorf("--human-slots needs an integer"))
			}
			opts.HumanSlots = value
		case "--timestamp":
			i++
			if i >= len(args) {
				die("kit scen blank", fmt.Errorf("--timestamp needs a unix value"))
			}
			value, err := strconv.Atoi(args[i])
			if err != nil {
				die("kit scen blank", fmt.Errorf("--timestamp needs an integer"))
			}
			opts.Timestamp = value
		case "--dummy-starters":
			opts.DummyStarters = true
		case "--no-starters":
			opts.NoStarters = true
		case "--gaia-active":
			opts.GaiaActive = true
		case "--no-conquest":
			opts.NoConquest = true
		case "--closeout-seconds":
			i++
			if i >= len(args) {
				die("kit scen blank", fmt.Errorf("--closeout-seconds needs a value"))
			}
			value, err := strconv.Atoi(args[i])
			if err != nil {
				die("kit scen blank", fmt.Errorf("--closeout-seconds needs an integer"))
			}
			opts.CloseoutSeconds = value
		case "--closeout-player":
			i++
			if i >= len(args) {
				die("kit scen blank", fmt.Errorf("--closeout-player needs a value"))
			}
			value, err := strconv.Atoi(args[i])
			if err != nil {
				die("kit scen blank", fmt.Errorf("--closeout-player needs an integer"))
			}
			opts.CloseoutPlayer = value
		case "--dummy-unit":
			i++
			if i >= len(args) {
				die("kit scen blank", fmt.Errorf("--dummy-unit needs a unit id"))
			}
			value, err := strconv.Atoi(args[i])
			if err != nil {
				die("kit scen blank", fmt.Errorf("--dummy-unit needs an integer"))
			}
			opts.DummyUnit = value
		case "--dummy-origin":
			i++
			if i >= len(args) {
				die("kit scen blank", fmt.Errorf("--dummy-origin needs X,Y"))
			}
			x, y, err := parseScenarioBlankPair(args[i])
			if err != nil {
				die("kit scen blank", err)
			}
			opts.DummyStartX = x
			opts.DummyStartY = y
		case "--dummy-spacing":
			i++
			if i >= len(args) {
				die("kit scen blank", fmt.Errorf("--dummy-spacing needs a value"))
			}
			value, err := strconv.ParseFloat(args[i], 64)
			if err != nil {
				die("kit scen blank", fmt.Errorf("--dummy-spacing needs a number"))
			}
			opts.DummySpacing = value
		case "--keep-seed-triggers":
			opts.KeepSeedTriggers = true
		case "--text":
			textOut = true
		case "--json":
			textOut = false
		default:
			die("kit scen blank", fmt.Errorf("unknown option %q", args[i]))
		}
	}
	report, err := scenario.WriteBlankScenarioFile(args[0], opts)
	if err != nil {
		die("kit scen blank", err)
	}
	if textOut {
		printScenarioBlank(report)
		return
	}
	printJSON(report)
}

func parseScenarioBlankPair(raw string) (float64, float64, error) {
	parts := strings.Split(raw, ",")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("expected X,Y, got %q", raw)
	}
	x, err := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid X in %q", raw)
	}
	y, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid Y in %q", raw)
	}
	return x, y, nil
}

func printScenarioPatch(report scenario.PatchReport) {
	fmt.Printf("input: %s\n", report.Input)
	fmt.Printf("output: %s\n", report.Output)
	fmt.Printf("verification: %s\n", report.Verification.Label)
	fmt.Printf("triggers: %d -> %d units: %d -> %d rebuild_ok=%t invariant_ok=%t\n",
		report.TriggerCountBefore, report.TriggerCountAfter,
		report.UnitCountBefore, report.UnitCountAfter,
		report.RebuildOK, report.InvariantOK)
	if report.MapTilesChanged > 0 {
		fmt.Printf("map_tiles_changed: %d\n", report.MapTilesChanged)
	}
	if report.TimestampOfLastSave > 0 {
		fmt.Printf("timestamp_of_last_save: %d\n", report.TimestampOfLastSave)
	}
}

func printScenarioBlank(report scenario.BlankReport) {
	fmt.Printf("output=%s version=%s players=%d human_slots=%d map=%dx%d tiles=%d seed_sha256=%s\n",
		report.Output,
		report.Version,
		report.PlayerCount,
		report.HumanSlots,
		report.MapWidth,
		report.MapHeight,
		report.TileCount,
		report.SeedSHA256,
	)
	fmt.Printf("triggers=%d->%d units=%d->%d clear_triggers=%t dummy_starters=%t dummy_unit=%d gaia_active=%t no_conquest=%t closeout_seconds=%d closeout_player=%d\n",
		report.TriggerCountBefore,
		report.TriggerCountAfter,
		report.UnitCountBefore,
		report.UnitCountAfter,
		report.ClearTriggers,
		report.DummyStarters,
		report.DummyUnit,
		report.GaiaActive,
		report.NoConquest,
		report.CloseoutSeconds,
		report.CloseoutPlayer,
	)
	fmt.Printf("verification=%s rebuild_ok=%t invariant_ok=%t\n", report.Verification, report.RebuildOK, report.InvariantOK)
	fmt.Printf("editor_parity_ok=%t note=%q\n", report.EditorParityOK, report.EditorParityNote)
}

func printScenarioTriggerSearch(report scenario.TriggerSearchReport) {
	fmt.Printf("path: %s\n", report.Path)
	fmt.Printf("verification: %s\n", report.Verification)
	fmt.Printf("scenario: version=%s inflated_bytes=%d compressed_bytes=%d\n", report.Version, report.InflatedBytes, report.CompressedBytes)
	fmt.Printf("query: grep=%q effect=%q condition=%q message=%q unit_ref=%s unit_type=%s player=%s variable=%s area=%v total_matches=%d shown=%d limit=%d\n",
		report.Query.Grep, report.Query.EffectQuery, report.Query.ConditionQuery, report.Query.Message,
		intPtrText(report.Query.UnitRef), intPtrText(report.Query.UnitType), intPtrText(report.Query.Player), intPtrText(report.Query.Variable), report.Query.Area,
		report.TotalMatches, len(report.Matches), report.Limit)
	for _, match := range report.Matches {
		child := ""
		if match.EffectIndex != nil {
			child = fmt.Sprintf(" effect=%d type=%d/%s", *match.EffectIndex, match.EffectType, match.EffectTypeName)
		}
		if match.ConditionIndex != nil {
			child = fmt.Sprintf(" condition=%d type=%d/%s", *match.ConditionIndex, match.ConditionType, match.ConditionName)
		}
		fmt.Printf("- trigger=%d%s enabled=%d looping=%d effects=%d conditions=%d name=%q\n",
			match.TriggerIndex, child, match.Enabled, match.Looping, match.EffectCount, match.ConditionCount, match.TriggerName)
		if match.Description != "" {
			fmt.Printf("  description: %s\n", oneLine(match.Description, 180))
		}
		if match.ShortDescription != "" {
			fmt.Printf("  short_description: %s\n", oneLine(match.ShortDescription, 180))
		}
		if len(match.MatchedFields) > 0 {
			keys := make([]string, 0, len(match.MatchedFields))
			for key := range match.MatchedFields {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			for _, key := range keys {
				fmt.Printf("  %s: %s\n", key, oneLine(fmt.Sprint(match.MatchedFields[key]), 180))
			}
		}
	}
}

func parseTriggerNeighborhoodArgs(args []string) (scenario.TriggerNeighborhoodOptions, bool, error) {
	opts := scenario.TriggerNeighborhoodOptions{}
	textOut := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--text":
			textOut = true
		case "--json":
			textOut = false
		case "--trigger":
			value, err := parseNextInt(args, &i, "--trigger")
			if err != nil {
				return opts, false, err
			}
			opts.TriggerIndex = &value
		case "--unit-ref":
			value, err := parseNextInt(args, &i, "--unit-ref")
			if err != nil {
				return opts, false, err
			}
			opts.UnitRef = &value
		case "--variable":
			value, err := parseNextInt(args, &i, "--variable")
			if err != nil {
				return opts, false, err
			}
			opts.Variable = &value
		case "--depth":
			value, err := parseNextInt(args, &i, "--depth")
			if err != nil {
				return opts, false, err
			}
			opts.Depth = value
		case "--limit":
			value, err := parseNextInt(args, &i, "--limit")
			if err != nil {
				return opts, false, err
			}
			opts.Limit = value
		case "--max-mb":
			value, err := parseNextInt(args, &i, "--max-mb")
			if err != nil {
				return opts, false, err
			}
			opts.MaxInflatedBytes = value * 1024 * 1024
		default:
			return opts, false, fmt.Errorf("unknown option %q", args[i])
		}
	}
	return opts, textOut, nil
}

func parsePlayerCoverageArgs(args []string) (scenario.PlayerCoverageOptions, bool, error) {
	opts := scenario.PlayerCoverageOptions{}
	textOut := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--text":
			textOut = true
		case "--json":
			textOut = false
		case "--players":
			i++
			if i >= len(args) {
				return opts, false, fmt.Errorf("--players needs a comma list")
			}
			values, err := parseIntList(args[i], "--players")
			if err != nil {
				return opts, false, err
			}
			opts.Players = values
		case "--max-mb":
			value, err := parseNextInt(args, &i, "--max-mb")
			if err != nil {
				return opts, false, err
			}
			opts.MaxInflatedBytes = value * 1024 * 1024
		default:
			return opts, false, fmt.Errorf("unknown option %q", args[i])
		}
	}
	return opts, textOut, nil
}

func parseMechanicArgs(args []string) (scenario.MechanicOptions, bool, error) {
	opts := scenario.MechanicOptions{}
	textOut := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--text":
			textOut = true
		case "--json":
			textOut = false
		case "--kind":
			i++
			if i >= len(args) {
				return opts, false, fmt.Errorf("--kind needs a value")
			}
			opts.Kind = args[i]
		case "--grep":
			i++
			if i >= len(args) {
				return opts, false, fmt.Errorf("--grep needs text")
			}
			opts.Grep = args[i]
		case "--max-mb":
			value, err := parseNextInt(args, &i, "--max-mb")
			if err != nil {
				return opts, false, err
			}
			opts.MaxInflatedBytes = value * 1024 * 1024
		default:
			return opts, false, fmt.Errorf("unknown option %q", args[i])
		}
	}
	return opts, textOut, nil
}

func parseTriggerFlowArgs(args []string) (scenario.TriggerFlowOptions, bool, error) {
	opts := scenario.TriggerFlowOptions{}
	textOut := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--text":
			textOut = true
		case "--json":
			textOut = false
		case "--grep":
			i++
			if i >= len(args) {
				return opts, false, fmt.Errorf("--grep needs text")
			}
			opts.Grep = args[i]
		case "--unit-ref":
			value, err := parseNextInt(args, &i, "--unit-ref")
			if err != nil {
				return opts, false, err
			}
			opts.UnitRef = &value
		case "--variable":
			value, err := parseNextInt(args, &i, "--variable")
			if err != nil {
				return opts, false, err
			}
			opts.Variable = &value
		case "--limit":
			value, err := parseNextInt(args, &i, "--limit")
			if err != nil {
				return opts, false, err
			}
			opts.Limit = value
		case "--max-mb":
			value, err := parseNextInt(args, &i, "--max-mb")
			if err != nil {
				return opts, false, err
			}
			opts.MaxInflatedBytes = value * 1024 * 1024
		default:
			return opts, false, fmt.Errorf("unknown option %q", args[i])
		}
	}
	return opts, textOut, nil
}

func printTriggerNeighborhood(report scenario.TriggerNeighborhoodReport) {
	fmt.Printf("path: %s\n", report.Path)
	fmt.Printf("verification: %s\n", report.Verification)
	fmt.Printf("summary: roots=%v nodes=%d edges=%d depth=%d\n", report.RootTriggers, len(report.Nodes), len(report.Edges), report.Query.Depth)
	for _, warning := range report.Warnings {
		fmt.Printf("warning: %s\n", warning)
	}
	for _, node := range report.Nodes {
		fmt.Printf("- depth=%d trigger=%d enabled=%d looping=%d effects=%d conditions=%d name=%q\n", node.Depth, node.Index, node.Enabled, node.Looping, node.Effects, node.Conditions, node.Name)
		for _, reason := range node.Reasons {
			fmt.Printf("  reason: %s\n", reason)
		}
		for _, edge := range report.Edges {
			if edge.From == node.Index {
				fmt.Printf("  -> trigger=%d kind=%s %s\n", edge.To, edge.Kind, edge.SourceChild)
			}
		}
	}
}

func printPlayerCoverage(report scenario.PlayerCoverageReport) {
	fmt.Printf("path: %s\n", report.Path)
	fmt.Printf("verification: %s\n", report.Verification)
	fmt.Printf("summary: players=%v families=%d issues=%d\n", report.Players, report.Summary.TriggerFamilies, report.Summary.Issues)
	for _, issue := range report.Issues {
		fmt.Printf("- %s %s: %s\n", issue.Severity, issue.Code, issue.Message)
	}
	for _, family := range report.Families {
		if len(family.Missing) == 0 {
			continue
		}
		fmt.Printf("family=%q players=%v missing=%v triggers=%d\n", family.Key, family.Players, family.Missing, len(family.Triggers))
		for _, trigger := range family.Triggers {
			fmt.Printf("  trigger=%d enabled=%d looping=%d name=%q\n", trigger.Index, trigger.Enabled, trigger.Looping, trigger.Name)
		}
	}
}

func printMechanic(report scenario.MechanicReport) {
	fmt.Printf("path: %s\n", report.Path)
	fmt.Printf("verification: %s\n", report.Verification)
	fmt.Printf("summary: kind=%s triggers=%d players=%v\n", report.Kind, report.Summary.TriggerCount, report.Summary.Players)
	for _, warning := range report.Warnings {
		fmt.Printf("warning: %s\n", warning)
	}
	for _, trigger := range report.Triggers {
		fmt.Printf("- trigger=%d role=%s players=%v name=%q\n", trigger.Index, trigger.Role, trigger.Players, trigger.Name)
		if len(trigger.Conditions) > 0 {
			fmt.Printf("  conditions: %s\n", strings.Join(trigger.Conditions, ", "))
		}
		if len(trigger.Effects) > 0 {
			fmt.Printf("  effects: %s\n", strings.Join(trigger.Effects, ", "))
		}
		for _, note := range trigger.Notes {
			fmt.Printf("  note: %s\n", note)
		}
	}
}

func printTriggerFlow(report scenario.TriggerFlowReport) {
	fmt.Printf("path: %s\n", report.Path)
	fmt.Printf("verification: %s\n", report.Verification)
	fmt.Printf("query: grep=%q unit_ref=%s variable=%s stages=%d\n", report.Query.Grep, intPtrText(report.Query.UnitRef), intPtrText(report.Query.Variable), len(report.Stages))
	for _, warning := range report.Warnings {
		fmt.Printf("warning: %s\n", warning)
	}
	for _, stage := range report.Stages {
		fmt.Printf("\n%s:\n", stage.Stage)
		for _, trigger := range stage.Triggers {
			fmt.Printf("- trigger=%d players=%v name=%q\n", trigger.Index, trigger.Players, trigger.Name)
			if len(trigger.Conditions) > 0 {
				fmt.Printf("  conditions: %s\n", strings.Join(trigger.Conditions, ", "))
			}
			if len(trigger.Effects) > 0 {
				fmt.Printf("  effects: %s\n", strings.Join(trigger.Effects, ", "))
			}
		}
	}
}

func printScenarioTriggerConditions(report scenario.TriggerConditionsReport) {
	fmt.Printf("path: %s\n", report.Path)
	fmt.Printf("verification: %s\n", report.Verification)
	fmt.Printf("scenario: version=%s inflated_bytes=%d compressed_bytes=%d\n", report.Version, report.InflatedBytes, report.CompressedBytes)
	fmt.Printf("triggers: total=%d shown=%d", report.TotalTriggers, len(report.Triggers))
	if report.Limit > 0 {
		fmt.Printf(" limit=%d", report.Limit)
	}
	fmt.Println()
	for _, trigger := range report.Triggers {
		fmt.Printf("- trigger=%d enabled=%d looping=%d effects=%d conditions=%d name=%q\n",
			trigger.Index, trigger.Enabled, trigger.Looping, len(trigger.Effects), len(trigger.Conditions), trigger.Name)
		for i, condition := range trigger.Conditions {
			fmt.Printf("  condition=%d %s\n", i, conditionSummaryText(condition))
		}
	}
}

func conditionSummaryText(condition map[string]any) string {
	keys := make([]string, 0, len(condition))
	for key := range condition {
		if key == "type" || key == "type_name" {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	typeID, _ := condition["type"].(int)
	typeName, _ := condition["type_name"].(string)
	parts := []string{fmt.Sprintf("type=%d/%s", typeID, typeName)}
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%v", key, condition[key]))
	}
	return strings.Join(parts, " ")
}

func printScenarioGlossary(report scenario.ScenarioGlossaryReport) {
	fmt.Printf("path: %s\n", report.Path)
	fmt.Printf("verification: %s\n", report.Verification)
	fmt.Printf("scenario: version=%s inflated_bytes=%d compressed_bytes=%d\n", report.Version, report.InflatedBytes, report.CompressedBytes)
	fmt.Printf("summary: triggers=%d effects=%d named_variables=%d\n", report.TriggerCount, report.EffectCount, report.VariableCount)
	printScenarioEffectBuckets("effect_types", report.EffectTypes, report.Limit)
	printScenarioStringCounts("trigger_prefixes", report.TriggerPrefixes)
	if len(report.Variables) > 0 {
		fmt.Println("variables:")
		for _, variable := range report.Variables {
			fmt.Printf("- %d %q\n", variable.ID, variable.Name)
		}
	}
	printScenarioStringCounts("xs_calls", report.XSCalls)
	printScenarioStringCounts("text_samples", report.TextSamples)
	printScenarioIntCounts("unit_ids", report.UnitIDs)
	printScenarioIntCounts("technology_ids", report.TechnologyIDs)
	printScenarioIntCounts("attribute_ids", report.AttributeIDs)
}

func printScenarioIdioms(report scenario.IdiomReport) {
	fmt.Printf("path: %s\n", report.Path)
	fmt.Printf("verification: %s\n", report.Verification)
	fmt.Printf("summary: version=%s idioms=%d\n", report.Version, report.Count)
	for _, idiom := range report.Idioms {
		fmt.Printf("- %s [%s] count=%d - %s\n", idiom.ID, idiom.Confidence, idiom.Count, idiom.Evidence)
		for _, sample := range idiom.Samples {
			fmt.Printf("  sample: %s\n", sample)
		}
	}
}

func printScenarioReferences(report scenario.ReferenceReport) {
	fmt.Printf("path: %s\n", report.Path)
	fmt.Printf("verification: %s\n", report.Verification)
	fmt.Printf("summary: version=%s refs=%d\n", report.Version, report.Summary.Total)
	kinds := make([]string, 0, len(report.Summary.ByKind))
	for kind := range report.Summary.ByKind {
		kinds = append(kinds, kind)
	}
	sort.Strings(kinds)
	for _, kind := range kinds {
		fmt.Printf("  %s=%d\n", kind, report.Summary.ByKind[kind])
	}
	for _, ref := range report.References {
		label := ""
		if ref.TargetLabel != "" {
			label = " " + ref.TargetLabel
		}
		fmt.Printf("- %s %d%s <- %s\n", ref.Kind, ref.TargetID, label, ref.SourcePath)
	}
}

func printScenarioSettings(report scenario.SettingsReport) {
	fmt.Printf("path: %s\n", report.Path)
	fmt.Printf("verification: %s\n", report.Verification)
	fmt.Printf("summary: version=%s player_count=%d players=%d\n", report.Version, report.PlayerCount, len(report.Players))
	if len(report.Victory) > 0 {
		keys := make([]string, 0, len(report.Victory))
		for key := range report.Victory {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		fmt.Println("victory:")
		for _, key := range keys {
			fmt.Printf("- %s=%d\n", key, report.Victory[key])
		}
	}
	fmt.Printf("options: full_tech_tree=%t collide_and_correcting=%t villager_force_drop=%t lock_coop_alliances=%t trigger_execution_order=%d/%q\n",
		report.Options.FullTechTree, report.Options.CollideAndCorrecting,
		report.Options.VillagerForceDrop, report.Options.LockCoopAlliances,
		report.Options.TriggerExecutionOrder, report.Options.TriggerExecutionOrderName)
	fmt.Printf("map: water_definition=%q\n", report.Map.WaterDefinition)
	if len(report.Options.InitialPlayerViews) > 0 {
		fmt.Print("initial_player_views:")
		for _, view := range report.Options.InitialPlayerViews {
			if view.Set {
				fmt.Printf(" P%d=(%d,%d)", view.Player, view.X, view.Y)
			}
		}
		fmt.Println()
	}
	fmt.Printf("diplomacy: lock_teams=%t allow_choose_teams=%t random_start_points=%t max_teams=%d\n",
		report.Diplomacy.LockTeams, report.Diplomacy.AllowPlayersChooseTeams,
		report.Diplomacy.RandomStartPoints, report.Diplomacy.MaxNumberOfTeams)
	fmt.Println("messages:")
	printScenarioMessageSlot("instructions", report.Messages.Instructions)
	printScenarioMessageSlot("hints", report.Messages.Hints)
	printScenarioMessageSlot("victory", report.Messages.Victory)
	printScenarioMessageSlot("loss", report.Messages.Loss)
	printScenarioMessageSlot("history", report.Messages.History)
	printScenarioMessageSlot("scouts", report.Messages.Scouts)
	fmt.Printf("cinematics: pregame=%q victory=%q loss=%q\n", report.Cinematics.Pregame, report.Cinematics.Victory, report.Cinematics.Loss)
	for _, player := range report.Players {
		fmt.Printf("- P%d active=%t human=%t civ=%q architecture=%q tribe=%q name_string_id=%d ai=%q ai_type=%d lock_civ=%t lock_personality=%t starting_age=%d/%q color=%d/%q base_priority=%d population_limit=%d allied_victory=%t resources=gold:%d wood:%d food:%d stone:%d trade:%d",
			player.Player, player.Active, player.Human, player.Civilization, player.Architecture, player.TribeName, player.NameStringID,
			player.AIName, player.AIType, player.LockCivilization, player.LockPersonality,
			player.StartingAge, player.StartingAgeName, player.Color, player.ColorName, player.BasePriority, player.PopulationLimit, player.AlliedVictory,
			player.Resources.Gold, player.Resources.Wood, player.Resources.Food, player.Resources.Stone, player.Resources.TradeGoods)
		if len(player.Diplomacy) > 0 {
			fmt.Printf(" diplomacy=%v", player.Diplomacy)
		}
		if len(player.DisabledTechs) > 0 {
			fmt.Printf(" disabled_techs=%v", player.DisabledTechs)
		}
		if len(player.DisabledUnits) > 0 {
			fmt.Printf(" disabled_units=%v", player.DisabledUnits)
		}
		if len(player.DisabledBuildings) > 0 {
			fmt.Printf(" disabled_buildings=%v", player.DisabledBuildings)
		}
		fmt.Println()
	}
}

func printScenarioMessageSlot(name string, slot scenario.MessageSlotSettings) {
	fmt.Printf("- %s source=%s string_table_id=%d uses_string_table_id=%t literal=%q\n",
		name, slot.EffectiveSource, slot.StringTableID, slot.UsesStringTableID, slot.Literal)
}

func printScenarioDeletePlan(report scenario.DeletePlanReport) {
	fmt.Printf("path: %s\n", report.Path)
	fmt.Printf("verification: %s\n", report.Verification)
	if report.Request.Kind == "system_prefix" {
		fmt.Printf("request: kind=%s target_prefix=%q\n", report.Request.Kind, report.Request.TargetPrefix)
	} else if report.Request.Kind == "effect_text_prefix" || report.Request.Kind == "effect_text_contains" {
		if report.Request.Kind == "effect_text_prefix" {
			fmt.Printf("request: kind=%s text_prefix=%q", report.Request.Kind, report.Request.TargetText)
		} else {
			fmt.Printf("request: kind=%s text=%q", report.Request.Kind, report.Request.TargetText)
		}
		if report.Request.TriggerID != nil {
			fmt.Printf(" trigger=%d", *report.Request.TriggerID)
		}
		if report.Request.TargetPrefix != "" {
			fmt.Printf(" trigger_prefix=%q", report.Request.TargetPrefix)
		}
		fmt.Println()
	} else if report.Request.Kind == "effect_type" || report.Request.Kind == "condition_type" {
		fmt.Printf("request: kind=%s type=%d", report.Request.Kind, report.Request.ID)
		if report.Request.TargetName != "" {
			fmt.Printf(" name=%q", report.Request.TargetName)
		}
		if report.Request.TriggerID != nil {
			fmt.Printf(" trigger=%d", *report.Request.TriggerID)
		}
		if report.Request.TargetPrefix != "" {
			fmt.Printf(" trigger_prefix=%q", report.Request.TargetPrefix)
		}
		fmt.Println()
	} else if report.Request.TriggerID != nil || report.Request.ChildIndex != nil {
		triggerIndex := -1
		childIndex := report.Request.ID
		if report.Request.TriggerID != nil {
			triggerIndex = *report.Request.TriggerID
		}
		if report.Request.ChildIndex != nil {
			childIndex = *report.Request.ChildIndex
		}
		fmt.Printf("request: kind=%s trigger_index=%d child_index=%d\n", report.Request.Kind, triggerIndex, childIndex)
	} else if report.Request.Kind == "units_area" {
		fmt.Printf("request: kind=%s area=%.2f,%.2f,%.2f,%.2f", report.Request.Kind,
			floatPtrValue(report.Request.AreaX1), floatPtrValue(report.Request.AreaY1),
			floatPtrValue(report.Request.AreaX2), floatPtrValue(report.Request.AreaY2))
		if report.Request.Player != nil {
			fmt.Printf(" player=%d", *report.Request.Player)
		}
		if report.Request.UnitConst != nil {
			fmt.Printf(" unit_const=%d", *report.Request.UnitConst)
		}
		fmt.Println()
	} else if report.Request.Kind == "unit_caption" {
		fmt.Printf("request: kind=%s target_caption=%q", report.Request.Kind, report.Request.TargetCaption)
		if report.Request.Player != nil {
			fmt.Printf(" player=%d", *report.Request.Player)
		}
		fmt.Println()
	} else if report.Request.Kind == "unit_caption_prefix" {
		fmt.Printf("request: kind=%s target_prefix=%q", report.Request.Kind, report.Request.TargetPrefix)
		if report.Request.Player != nil {
			fmt.Printf(" player=%d", *report.Request.Player)
		}
		fmt.Println()
	} else if report.Request.Kind == "unit_caption_contains" {
		fmt.Printf("request: kind=%s target_text=%q", report.Request.Kind, report.Request.TargetText)
		if report.Request.Player != nil {
			fmt.Printf(" player=%d", *report.Request.Player)
		}
		fmt.Println()
	} else if report.Request.Kind == "unit_type" {
		fmt.Printf("request: kind=%s unit_const=%d", report.Request.Kind, intPtrValue(report.Request.UnitConst))
		if report.Request.Player != nil {
			fmt.Printf(" player=%d", *report.Request.Player)
		}
		fmt.Println()
	} else if report.Request.Kind == "units_player" {
		fmt.Printf("request: kind=%s player=%d\n", report.Request.Kind, intPtrValue(report.Request.Player))
	} else if report.Request.Kind == "trigger_name" {
		fmt.Printf("request: kind=%s target_name=%q\n", report.Request.Kind, report.Request.TargetName)
	} else if report.Request.Kind == "trigger_prefix" {
		fmt.Printf("request: kind=%s target_prefix=%q\n", report.Request.Kind, report.Request.TargetPrefix)
	} else if report.Request.Kind == "trigger_contains" {
		fmt.Printf("request: kind=%s target_text=%q\n", report.Request.Kind, report.Request.TargetText)
	} else if report.Request.Kind == "variable_name" {
		fmt.Printf("request: kind=%s target_name=%q\n", report.Request.Kind, report.Request.TargetName)
	} else if report.Request.Kind == "variable_prefix" {
		fmt.Printf("request: kind=%s target_prefix=%q\n", report.Request.Kind, report.Request.TargetPrefix)
	} else if report.Request.Kind == "variable_contains" {
		fmt.Printf("request: kind=%s target_text=%q\n", report.Request.Kind, report.Request.TargetText)
	} else if report.Request.Kind == "string_text" {
		fmt.Printf("request: kind=%s target_text=%q\n", report.Request.Kind, report.Request.TargetText)
	} else if report.Request.Kind == "string_prefix" {
		fmt.Printf("request: kind=%s target_prefix=%q\n", report.Request.Kind, report.Request.TargetPrefix)
	} else if report.Request.Kind == "string_contains" {
		fmt.Printf("request: kind=%s target_text=%q\n", report.Request.Kind, report.Request.TargetText)
	} else {
		fmt.Printf("request: kind=%s id=%d\n", report.Request.Kind, report.Request.ID)
	}
	fmt.Printf("decision: can_delete=%t can_tombstone=%t strategy=%s\n", report.CanDelete, report.CanTombstone, report.Strategy)
	if len(report.ResolvedIDs) > 0 {
		fmt.Printf("resolved_ids: %s\n", compactIntListText(report.ResolvedIDs))
	}
	if len(report.ResolvedSets) > 0 {
		keys := make([]string, 0, len(report.ResolvedSets))
		for key := range report.ResolvedSets {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			fmt.Printf("resolved_%s: %s\n", key, compactIntListText(report.ResolvedSets[key]))
		}
	}
	fmt.Printf("references: total=%d\n", report.ReferenceSummary.Total)
	kinds := make([]string, 0, len(report.ReferenceSummary.ByKind))
	for kind := range report.ReferenceSummary.ByKind {
		kinds = append(kinds, kind)
	}
	sort.Strings(kinds)
	for _, kind := range kinds {
		fmt.Printf("  %s=%d\n", kind, report.ReferenceSummary.ByKind[kind])
	}
	if len(report.BlockingRefs) > 0 {
		fmt.Println("blocking_refs:")
		const maxTextBlockingRefs = 50
		rows := report.BlockingRefs
		if len(rows) > maxTextBlockingRefs {
			rows = rows[:maxTextBlockingRefs]
		}
		for _, ref := range rows {
			label := ""
			if ref.TargetLabel != "" {
				label = " " + ref.TargetLabel
			}
			fmt.Printf("- %s %d%s <- %s\n", ref.Kind, ref.TargetID, label, ref.SourcePath)
		}
		if omitted := len(report.BlockingRefs) - len(rows); omitted > 0 {
			fmt.Printf("... omitted %d blocking ref row(s); use JSON output for the full ledger\n", omitted)
		}
	}
	if report.SuggestedRecipe != nil {
		data, err := json.MarshalIndent(report.SuggestedRecipe, "", "  ")
		if err == nil {
			fmt.Println("suggested_recipe:")
			fmt.Println(string(data))
		}
	}
	if report.CleanupCommand != "" {
		fmt.Printf("cleanup_command: %s\n", report.CleanupCommand)
	}
	if report.CleanupRecipe != nil {
		data, err := json.MarshalIndent(report.CleanupRecipe, "", "  ")
		if err == nil {
			fmt.Println("cleanup_recipe:")
			fmt.Println(string(data))
		}
	}
	for _, caveat := range report.StructuralCaveats {
		fmt.Printf("caveat: %s\n", caveat)
	}
}

func printDatDeletePlan(report datcodec.DeletePlanReport) {
	fmt.Printf("verification: %s\n", report.Verification.Label)
	if report.Verification.Note != "" {
		fmt.Printf("verification_note: %s\n", report.Verification.Note)
	}
	fmt.Printf("request: section=%s id=%d", report.Request.Section, report.Request.ID)
	if report.Request.CivID != nil {
		fmt.Printf(" civ=%d", *report.Request.CivID)
	}
	if report.Request.EffectID != nil && report.Request.CommandIndex != nil {
		fmt.Printf(" effect=%d command=%d", *report.Request.EffectID, *report.Request.CommandIndex)
	}
	if report.Request.SoundID != nil && report.Request.ItemIndex != nil {
		fmt.Printf(" sound=%d item=%d", *report.Request.SoundID, *report.Request.ItemIndex)
	}
	if report.Request.GraphicID != nil && report.Request.RowIndex != nil {
		fmt.Printf(" graphic=%d row=%d", *report.Request.GraphicID, *report.Request.RowIndex)
	}
	if report.Request.UnitID != nil && report.Request.RowIndex != nil {
		fmt.Printf(" unit=%d row=%d", *report.Request.UnitID, *report.Request.RowIndex)
	}
	if report.Request.UnitHeaderID != nil && report.Request.RowIndex != nil {
		fmt.Printf(" unit_header=%d row=%d", *report.Request.UnitHeaderID, *report.Request.RowIndex)
	}
	if report.Request.AllCivs {
		fmt.Print(" all_civs=true")
	}
	fmt.Println()
	fmt.Printf("decision: supported=%t mutates=%t strategy=%s\n", report.Supported, report.Mutates, report.Strategy)
	if report.Reason != "" {
		fmt.Printf("reason: %s\n", report.Reason)
	}
	if report.TechMigration != nil {
		fmt.Printf("tech_migration: delete=%d tail=%d ready=%t covered_refs=%d unsupported_refs=%d\n",
			report.TechMigration.DeleteID,
			report.TechMigration.TailID,
			report.TechMigration.Ready,
			report.TechMigration.CoveredReferences,
			report.TechMigration.UnsupportedReferences)
	}
	if len(report.References) > 0 {
		byConfidence := map[string]int{}
		bySection := map[string]int{}
		for _, ref := range report.References {
			confidence := ref.Confidence
			if confidence == "" {
				confidence = "decoded"
			}
			byConfidence[confidence]++
			bySection[ref.Section]++
		}
		fmt.Printf("references: total=%d\n", len(report.References))
		printSortedCounts("  by_confidence", byConfidence)
		printSortedCounts("  by_section", bySection)
		const maxTextReferenceRows = 50
		fmt.Println("reference_rows:")
		rows := report.References
		if len(rows) > maxTextReferenceRows {
			rows = rows[:maxTextReferenceRows]
		}
		for _, ref := range rows {
			confidence := ref.Confidence
			if confidence == "" {
				confidence = "decoded"
			}
			fmt.Printf("- %s id=%d field=%s confidence=%s\n", ref.Section, ref.ID, ref.Field, confidence)
		}
		if omitted := len(report.References) - len(rows); omitted > 0 {
			fmt.Printf("... omitted %d reference row(s); use JSON output for the full ledger\n", omitted)
		}
	} else {
		fmt.Println("references: total=0")
	}
	if len(report.RecipeHint) > 0 {
		var pretty any
		if err := json.Unmarshal(report.RecipeHint, &pretty); err == nil {
			data, err := json.MarshalIndent(pretty, "", "  ")
			if err == nil {
				fmt.Println("recipe_hint:")
				fmt.Println(string(data))
			}
		} else {
			fmt.Printf("recipe_hint: %s\n", string(report.RecipeHint))
		}
	}
	if report.CleanupCommand != "" {
		fmt.Printf("cleanup_command: %s\n", report.CleanupCommand)
	}
	if len(report.CleanupRecipe) > 0 {
		var pretty any
		if err := json.Unmarshal(report.CleanupRecipe, &pretty); err == nil {
			data, err := json.MarshalIndent(pretty, "", "  ")
			if err == nil {
				fmt.Println("cleanup_recipe:")
				fmt.Println(string(data))
			}
		} else {
			fmt.Printf("cleanup_recipe: %s\n", string(report.CleanupRecipe))
		}
	}
	for _, warning := range report.Warnings {
		fmt.Printf("warning: %s\n", warning)
	}
}

func applyDatDeleteFile(diePrefix, inPath, outPath string, request datcodec.DeletePlanRequest) {
	plan, err := datcodec.DeletePlanFile(inPath, request)
	if err != nil {
		die(diePrefix, err)
	}
	if !plan.Supported || !plan.Mutates || len(plan.RecipeHint) == 0 {
		reason := plan.Reason
		if reason == "" {
			reason = "delete-plan did not produce an actionable recipe"
		}
		die(diePrefix, fmt.Errorf("refusing unsupported delete: strategy=%s reason=%s", plan.Strategy, reason))
	}
	var recipe datcodec.Recipe
	if err := json.Unmarshal(plan.RecipeHint, &recipe); err != nil {
		die(diePrefix, fmt.Errorf("delete-plan recipe hint did not decode: %w", err))
	}
	patch, err := datcodec.PatchRecipeFile(inPath, outPath, recipe)
	if err != nil {
		die(diePrefix, err)
	}
	printJSON(struct {
		Version      string                     `json:"version"`
		Request      datcodec.DeletePlanRequest `json:"request"`
		DeletePlan   datcodec.DeletePlanReport  `json:"delete_plan"`
		PatchReport  datcodec.PatchReport       `json:"patch_report"`
		Verification string                     `json:"verification"`
	}{
		Version:      datcodec.Version,
		Request:      request,
		DeletePlan:   plan,
		PatchReport:  patch,
		Verification: "structure_verified_not_engine_verified",
	})
}

func parseDatDeleteRequest(section, idText string, args []string, allowText bool) (datcodec.DeletePlanRequest, bool, error) {
	request, err := parseDatDeleteTarget(section, idText)
	if err != nil {
		return datcodec.DeletePlanRequest{}, false, err
	}
	text := false
	sawCivScope := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--all-civs":
			request.AllCivs = true
			sawCivScope = true
		case "--civ":
			i++
			if i >= len(args) {
				return datcodec.DeletePlanRequest{}, false, errors.New("--civ needs a value")
			}
			civID, err := strconv.Atoi(args[i])
			if err != nil {
				return datcodec.DeletePlanRequest{}, false, fmt.Errorf("invalid civ id %q: %w", args[i], err)
			}
			request.CivID = &civID
			sawCivScope = true
		case "--text":
			if !allowText {
				return datcodec.DeletePlanRequest{}, false, errors.New("--text is not supported for kit dat delete; inspect the JSON patch report")
			}
			text = true
		default:
			return datcodec.DeletePlanRequest{}, false, fmt.Errorf("unknown option %q", args[i])
		}
	}
	if (request.Section == "effect_command" || request.Section == "sound_item" || request.RowIndex != nil) && sawCivScope {
		return datcodec.DeletePlanRequest{}, false, fmt.Errorf("%s delete does not accept --civ or --all-civs", strings.ReplaceAll(request.Section, "_", "-"))
	}
	return request, text, nil
}

func parseDatDeleteTarget(section, idText string) (datcodec.DeletePlanRequest, error) {
	normalized := strings.ToLower(strings.TrimSpace(section))
	switch normalized {
	case "effect_command", "effect-command", "command":
		parts := strings.Split(idText, ":")
		if len(parts) != 2 {
			return datcodec.DeletePlanRequest{}, fmt.Errorf("%s target must be formatted as <effect_id>:<command_index>", section)
		}
		effectID, err := strconv.Atoi(parts[0])
		if err != nil {
			return datcodec.DeletePlanRequest{}, fmt.Errorf("invalid effect id %q: %w", parts[0], err)
		}
		commandIndex, err := strconv.Atoi(parts[1])
		if err != nil {
			return datcodec.DeletePlanRequest{}, fmt.Errorf("invalid command index %q: %w", parts[1], err)
		}
		return datcodec.DeletePlanRequest{
			Section:      "effect_command",
			ID:           effectID,
			EffectID:     &effectID,
			CommandIndex: &commandIndex,
		}, nil
	case "sound_item", "sound-item":
		parts := strings.Split(idText, ":")
		if len(parts) != 2 {
			return datcodec.DeletePlanRequest{}, fmt.Errorf("%s target must be formatted as <sound_id>:<item_index>", section)
		}
		soundID, err := strconv.Atoi(parts[0])
		if err != nil {
			return datcodec.DeletePlanRequest{}, fmt.Errorf("invalid sound id %q: %w", parts[0], err)
		}
		itemIndex, err := strconv.Atoi(parts[1])
		if err != nil {
			return datcodec.DeletePlanRequest{}, fmt.Errorf("invalid item index %q: %w", parts[1], err)
		}
		return datcodec.DeletePlanRequest{
			Section:   "sound_item",
			ID:        soundID,
			SoundID:   &soundID,
			ItemIndex: &itemIndex,
		}, nil
	case "unit_header_task", "unit-header-task":
		parts := strings.Split(idText, ":")
		if len(parts) != 2 {
			return datcodec.DeletePlanRequest{}, fmt.Errorf("%s target must be formatted as <unit_header_id>:<row_index>", section)
		}
		headerID, err := strconv.Atoi(parts[0])
		if err != nil {
			return datcodec.DeletePlanRequest{}, fmt.Errorf("invalid unit header id %q: %w", parts[0], err)
		}
		rowIndex, err := strconv.Atoi(parts[1])
		if err != nil {
			return datcodec.DeletePlanRequest{}, fmt.Errorf("invalid row index %q: %w", parts[1], err)
		}
		return datcodec.DeletePlanRequest{
			Section:      "unit_header_task",
			ID:           headerID,
			UnitHeaderID: &headerID,
			RowIndex:     &rowIndex,
		}, nil
	case "graphic_delta", "graphic-delta", "graphic_angle_sound", "graphic-angle-sound":
		parts := strings.Split(idText, ":")
		if len(parts) != 2 {
			return datcodec.DeletePlanRequest{}, fmt.Errorf("%s target must be formatted as <graphic_id>:<row_index>", section)
		}
		graphicID, err := strconv.Atoi(parts[0])
		if err != nil {
			return datcodec.DeletePlanRequest{}, fmt.Errorf("invalid graphic id %q: %w", parts[0], err)
		}
		rowIndex, err := strconv.Atoi(parts[1])
		if err != nil {
			return datcodec.DeletePlanRequest{}, fmt.Errorf("invalid row index %q: %w", parts[1], err)
		}
		normalizedSection := "graphic_delta"
		if normalized == "graphic_angle_sound" || normalized == "graphic-angle-sound" {
			normalizedSection = "graphic_angle_sound"
		}
		return datcodec.DeletePlanRequest{
			Section:   normalizedSection,
			ID:        graphicID,
			GraphicID: &graphicID,
			RowIndex:  &rowIndex,
		}, nil
	case "unit_damage_graphic", "unit-damage-graphic", "unit_damage", "unit-damage",
		"unit_attack", "unit-attack",
		"unit_armour", "unit-armour", "unit_armor", "unit-armor",
		"unit_train_location", "unit-train-location",
		"unit_drop_site", "unit-drop-site",
		"unit_task", "unit-task":
		parts := strings.Split(idText, ":")
		if len(parts) != 3 {
			return datcodec.DeletePlanRequest{}, fmt.Errorf("%s target must be formatted as <civ_id>:<unit_id>:<row_index>", section)
		}
		civID, err := strconv.Atoi(parts[0])
		if err != nil {
			return datcodec.DeletePlanRequest{}, fmt.Errorf("invalid civ id %q: %w", parts[0], err)
		}
		unitID, err := strconv.Atoi(parts[1])
		if err != nil {
			return datcodec.DeletePlanRequest{}, fmt.Errorf("invalid unit id %q: %w", parts[1], err)
		}
		rowIndex, err := strconv.Atoi(parts[2])
		if err != nil {
			return datcodec.DeletePlanRequest{}, fmt.Errorf("invalid row index %q: %w", parts[2], err)
		}
		return datcodec.DeletePlanRequest{
			Section:  normalizeDatUnitRowDeleteSection(normalized),
			ID:       unitID,
			CivID:    &civID,
			UnitID:   &unitID,
			RowIndex: &rowIndex,
		}, nil
	default:
		id, err := strconv.Atoi(idText)
		if err != nil {
			return datcodec.DeletePlanRequest{}, fmt.Errorf("invalid id %q: %w", idText, err)
		}
		return datcodec.DeletePlanRequest{Section: section, ID: id}, nil
	}
}

func datUnitChildDeleteSection(kind string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "damage-graphic", "damage_graphic", "unit-damage-graphic", "unit_damage_graphic", "unit-damage", "unit_damage":
		return "unit_damage_graphic", nil
	case "attack", "unit-attack", "unit_attack":
		return "unit_attack", nil
	case "armour", "armor", "unit-armour", "unit_armour", "unit-armor", "unit_armor":
		return "unit_armour", nil
	case "train-location", "train_location", "unit-train-location", "unit_train_location":
		return "unit_train_location", nil
	case "drop-site", "drop_site", "unit-drop-site", "unit_drop_site":
		return "unit_drop_site", nil
	case "task", "unit-task", "unit_task":
		return "unit_task", nil
	default:
		return "", fmt.Errorf("unknown unit child delete kind %q; want damage-graphic, attack, armour, train-location, drop-site, or task", kind)
	}
}

func normalizeDatUnitRowDeleteSection(section string) string {
	switch section {
	case "unit_damage_graphic", "unit-damage-graphic", "unit_damage", "unit-damage":
		return "unit_damage_graphic"
	case "unit_attack", "unit-attack":
		return "unit_attack"
	case "unit_armour", "unit-armour", "unit_armor", "unit-armor":
		return "unit_armour"
	case "unit_train_location", "unit-train-location":
		return "unit_train_location"
	case "unit_drop_site", "unit-drop-site":
		return "unit_drop_site"
	case "unit_task", "unit-task":
		return "unit_task"
	default:
		return section
	}
}

type datRefsOptions struct {
	Filters datcodec.ReferenceFilters
	Text    bool
}

func parseDatRefsOptions(args []string) (datRefsOptions, error) {
	var opts datRefsOptions
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--text":
			opts.Text = true
		case "--json":
			opts.Text = false
		case "--class":
			i++
			if i >= len(args) {
				return opts, errors.New("--class needs a value")
			}
			switch args[i] {
			case "rewrite_supported", "known_readonly", "possible_operand", "candidate_operand", "unsupported":
				opts.Filters.Class = args[i]
			default:
				return opts, fmt.Errorf("--class must be rewrite_supported, known_readonly, possible_operand, candidate_operand, or unsupported")
			}
		case "--all":
			opts.Filters.Limit = 0
		case "--confidence":
			i++
			if i >= len(args) {
				return opts, errors.New("--confidence needs a value")
			}
			opts.Filters.Confidence = args[i]
		case "--source-section":
			i++
			if i >= len(args) {
				return opts, errors.New("--source-section needs a value")
			}
			opts.Filters.SourceSection = args[i]
		case "--limit":
			i++
			if i >= len(args) {
				return opts, errors.New("--limit needs a value")
			}
			limit, err := strconv.Atoi(args[i])
			if err != nil {
				return opts, fmt.Errorf("invalid --limit %q: %w", args[i], err)
			}
			if limit < 0 {
				return opts, errors.New("--limit must be non-negative")
			}
			opts.Filters.Limit = limit
		default:
			return opts, fmt.Errorf("unknown kit dat refs option %q", args[i])
		}
	}
	return opts, nil
}

func parseDatDisconnectRequest(section, idText string) (datcodec.DeletePlanRequest, datcodec.Recipe, error) {
	id, err := strconv.Atoi(idText)
	if err != nil {
		return datcodec.DeletePlanRequest{}, datcodec.Recipe{}, fmt.Errorf("invalid id %q: %w", idText, err)
	}
	if id < 0 {
		return datcodec.DeletePlanRequest{}, datcodec.Recipe{}, fmt.Errorf("id must be non-negative")
	}
	normalized := strings.ToLower(strings.TrimSpace(section))
	switch normalized {
	case "tech", "research", "researches":
		return datcodec.DeletePlanRequest{Section: "tech", ID: id}, datcodec.Recipe{DisconnectTechs: []int{id}}, nil
	case "unit", "units":
		return datcodec.DeletePlanRequest{Section: "unit", ID: id}, datcodec.Recipe{DisconnectUnits: []int{id}}, nil
	default:
		return datcodec.DeletePlanRequest{}, datcodec.Recipe{}, fmt.Errorf("unsupported disconnect section %q; expected tech or unit", section)
	}
}

func printSortedCounts(label string, counts map[string]int) {
	keys := make([]string, 0, len(counts))
	for key := range counts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	fmt.Printf("%s:", label)
	for _, key := range keys {
		fmt.Printf(" %s=%d", key, counts[key])
	}
	fmt.Println()
}

func parseScenarioDeletePlanRequest(kind, target string) (scenario.DeletePlanRequest, error) {
	normalized := strings.ToLower(strings.TrimSpace(kind))
	switch normalized {
	case "system_prefix", "system-prefix":
		if strings.TrimSpace(target) == "" {
			return scenario.DeletePlanRequest{}, fmt.Errorf("system-prefix target must be non-empty")
		}
		return scenario.DeletePlanRequest{Kind: "system_prefix", TargetPrefix: target}, nil
	case "unit_caption", "unit-caption", "object_caption", "object-caption", "caption":
		if strings.TrimSpace(target) == "" {
			return scenario.DeletePlanRequest{}, fmt.Errorf("unit-caption target must be non-empty")
		}
		return scenario.DeletePlanRequest{Kind: "unit_caption", TargetCaption: target}, nil
	case "unit_caption_prefix", "unit-caption-prefix", "object_caption_prefix", "object-caption-prefix", "caption_prefix", "caption-prefix":
		if strings.TrimSpace(target) == "" {
			return scenario.DeletePlanRequest{}, fmt.Errorf("unit-caption-prefix target must be non-empty")
		}
		return scenario.DeletePlanRequest{Kind: "unit_caption_prefix", TargetPrefix: target}, nil
	case "unit_caption_contains", "unit-caption-contains", "object_caption_contains", "object-caption-contains", "caption_contains", "caption-contains":
		if strings.TrimSpace(target) == "" {
			return scenario.DeletePlanRequest{}, fmt.Errorf("unit-caption-contains target must be non-empty")
		}
		return scenario.DeletePlanRequest{Kind: "unit_caption_contains", TargetText: target}, nil
	case "unit_type", "unit-type", "units_type", "units-type", "unit_const", "unit-const", "units_const", "units-const":
		unitConst, err := strconv.Atoi(target)
		if err != nil {
			return scenario.DeletePlanRequest{}, fmt.Errorf("invalid unit-type unit const %q: %w", target, err)
		}
		if unitConst < 0 {
			return scenario.DeletePlanRequest{}, fmt.Errorf("unit-type unit const must be non-negative")
		}
		return scenario.DeletePlanRequest{Kind: "unit_type", UnitConst: &unitConst}, nil
	case "units_player", "units-player", "player_units", "player-units":
		player, err := strconv.Atoi(target)
		if err != nil {
			return scenario.DeletePlanRequest{}, fmt.Errorf("invalid units-player player %q: %w", target, err)
		}
		if player < 0 {
			return scenario.DeletePlanRequest{}, fmt.Errorf("units-player player must be non-negative")
		}
		return scenario.DeletePlanRequest{Kind: "units_player", Player: &player}, nil
	case "trigger_name", "trigger-name":
		if strings.TrimSpace(target) == "" {
			return scenario.DeletePlanRequest{}, fmt.Errorf("trigger-name target must be non-empty")
		}
		return scenario.DeletePlanRequest{Kind: "trigger_name", TargetName: target}, nil
	case "trigger_prefix", "trigger-prefix":
		if strings.TrimSpace(target) == "" {
			return scenario.DeletePlanRequest{}, fmt.Errorf("trigger-prefix target must be non-empty")
		}
		return scenario.DeletePlanRequest{Kind: "trigger_prefix", TargetPrefix: target}, nil
	case "trigger_contains", "trigger-contains", "trigger_grep", "trigger-grep":
		if strings.TrimSpace(target) == "" {
			return scenario.DeletePlanRequest{}, fmt.Errorf("trigger-contains target must be non-empty")
		}
		return scenario.DeletePlanRequest{Kind: "trigger_contains", TargetText: target}, nil
	case "variable_name", "variable-name":
		if strings.TrimSpace(target) == "" {
			return scenario.DeletePlanRequest{}, fmt.Errorf("variable-name target must be non-empty")
		}
		return scenario.DeletePlanRequest{Kind: "variable_name", TargetName: target}, nil
	case "variable_prefix", "variable-prefix":
		if strings.TrimSpace(target) == "" {
			return scenario.DeletePlanRequest{}, fmt.Errorf("variable-prefix target must be non-empty")
		}
		return scenario.DeletePlanRequest{Kind: "variable_prefix", TargetPrefix: target}, nil
	case "variable_contains", "variable-contains":
		if strings.TrimSpace(target) == "" {
			return scenario.DeletePlanRequest{}, fmt.Errorf("variable-contains target must be non-empty")
		}
		return scenario.DeletePlanRequest{Kind: "variable_contains", TargetText: target}, nil
	case "string_text", "string-text":
		if strings.TrimSpace(target) == "" {
			return scenario.DeletePlanRequest{}, fmt.Errorf("string-text target must be non-empty")
		}
		return scenario.DeletePlanRequest{Kind: "string_text", TargetText: target}, nil
	case "string_prefix", "string-prefix":
		if strings.TrimSpace(target) == "" {
			return scenario.DeletePlanRequest{}, fmt.Errorf("string-prefix target must be non-empty")
		}
		return scenario.DeletePlanRequest{Kind: "string_prefix", TargetPrefix: target}, nil
	case "string_contains", "string-contains":
		if strings.TrimSpace(target) == "" {
			return scenario.DeletePlanRequest{}, fmt.Errorf("string-contains target must be non-empty")
		}
		return scenario.DeletePlanRequest{Kind: "string_contains", TargetText: target}, nil
	case "units_area", "units-area", "area_units", "area-units":
		coords, err := parseScenarioDeletePlanArea(target)
		if err != nil {
			return scenario.DeletePlanRequest{}, err
		}
		return scenario.DeletePlanRequest{
			Kind:   kind,
			AreaX1: &coords[0],
			AreaY1: &coords[1],
			AreaX2: &coords[2],
			AreaY2: &coords[3],
		}, nil
	case "effect", "condition":
		parts := strings.Split(target, ":")
		if len(parts) != 2 {
			return scenario.DeletePlanRequest{}, fmt.Errorf("%s target must be trigger_index:child_index", normalized)
		}
		triggerIndex, err := strconv.Atoi(parts[0])
		if err != nil {
			return scenario.DeletePlanRequest{}, fmt.Errorf("invalid trigger index %q: %w", parts[0], err)
		}
		childIndex, err := strconv.Atoi(parts[1])
		if err != nil {
			return scenario.DeletePlanRequest{}, fmt.Errorf("invalid %s index %q: %w", normalized, parts[1], err)
		}
		if triggerIndex < 0 || childIndex < 0 {
			return scenario.DeletePlanRequest{}, fmt.Errorf("%s target indices must be non-negative", normalized)
		}
		return scenario.DeletePlanRequest{Kind: normalized, ID: childIndex, TriggerID: &triggerIndex, ChildIndex: &childIndex}, nil
	case "effect_type", "effect-type", "effects_type", "effects-type":
		typeID, name, err := parseScenarioEffectTypeTarget(target)
		if err != nil {
			return scenario.DeletePlanRequest{}, err
		}
		return scenario.DeletePlanRequest{Kind: "effect_type", ID: typeID, TargetName: name}, nil
	case "condition_type", "condition-type", "conditions_type", "conditions-type":
		typeID, name, err := parseScenarioConditionTypeTarget(target)
		if err != nil {
			return scenario.DeletePlanRequest{}, err
		}
		return scenario.DeletePlanRequest{Kind: "condition_type", ID: typeID, TargetName: name}, nil
	case "effect_text_prefix", "effect-text-prefix", "effects_text_prefix", "effects-text-prefix":
		if strings.TrimSpace(target) == "" {
			return scenario.DeletePlanRequest{}, fmt.Errorf("effect-text-prefix target must be non-empty")
		}
		return scenario.DeletePlanRequest{Kind: "effect_text_prefix", TargetText: target}, nil
	case "effect_text_contains", "effect-text-contains", "effects_text_contains", "effects-text-contains":
		if strings.TrimSpace(target) == "" {
			return scenario.DeletePlanRequest{}, fmt.Errorf("effect-text-contains target must be non-empty")
		}
		return scenario.DeletePlanRequest{Kind: "effect_text_contains", TargetText: target}, nil
	default:
		id, err := strconv.Atoi(target)
		if err != nil {
			return scenario.DeletePlanRequest{}, fmt.Errorf("invalid id %q: %w", target, err)
		}
		return scenario.DeletePlanRequest{Kind: kind, ID: id}, nil
	}
}

func parseScenarioEffectTypeTarget(target string) (int, string, error) {
	target = strings.TrimSpace(target)
	if target == "" {
		return 0, "", errors.New("effect-type target must be non-empty")
	}
	if id, err := strconv.Atoi(target); err == nil {
		if id < 0 {
			return 0, "", errors.New("effect-type id must be non-negative")
		}
		return id, scenario.EffectTypeName(id), nil
	}
	name := strings.ToLower(target)
	if id, ok := scenario.EffectTypeForOp(name); ok {
		return id, name, nil
	}
	return 0, "", fmt.Errorf("unknown effect-type %q; use a numeric effect_type id or known recipe op name", target)
}

func parseScenarioConditionTypeTarget(target string) (int, string, error) {
	target = strings.TrimSpace(target)
	if target == "" {
		return 0, "", errors.New("condition-type target must be non-empty")
	}
	if id, err := strconv.Atoi(target); err == nil {
		if id < 0 {
			return 0, "", errors.New("condition-type id must be non-negative")
		}
		return id, scenario.ConditionTypeName(id), nil
	}
	name := strings.ToLower(target)
	if id, ok := scenario.ConditionTypeForName(name); ok {
		return id, name, nil
	}
	return 0, "", fmt.Errorf("unknown condition-type %q; use a numeric condition_type id or known condition type name", target)
}

func isScenarioDeletePlanAreaKind(kind string) bool {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "units_area", "units-area", "area_units", "area-units":
		return true
	default:
		return false
	}
}

func isScenarioDeletePlanUnitTypeKind(kind string) bool {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "unit_type", "unit-type", "units_type", "units-type", "unit_const", "unit-const", "units_const", "units-const":
		return true
	default:
		return false
	}
}

func isScenarioDeletePlanUnitsPlayerKind(kind string) bool {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "units_player", "units-player", "player_units", "player-units":
		return true
	default:
		return false
	}
}

func isScenarioDeletePlanChildTypeKind(kind string) bool {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "effect_type", "effect-type", "effects_type", "effects-type", "condition_type", "condition-type", "conditions_type", "conditions-type", "effect_text_prefix", "effect-text-prefix", "effects_text_prefix", "effects-text-prefix", "effect_text_contains", "effect-text-contains", "effects_text_contains", "effects-text-contains":
		return true
	default:
		return false
	}
}

func parseScenarioDeleteOptions(request *scenario.DeletePlanRequest, args []string, allowText bool) (bool, error) {
	textOut := false
	playerOptionSeen := false
	unitOptionSeen := false
	triggerOptionSeen := false
	triggerPrefixOptionSeen := false
	for i := 0; i < len(args); i++ {
		switch arg := args[i]; arg {
		case "--text":
			if !allowText {
				return false, errors.New("--text is not supported for kit scen delete; inspect the JSON patch report")
			}
			textOut = true
		case "--player":
			i++
			if i >= len(args) {
				return false, errors.New("--player needs a value")
			}
			player, err := strconv.Atoi(args[i])
			if err != nil {
				return false, fmt.Errorf("invalid player %q: %w", args[i], err)
			}
			playerOptionSeen = true
			request.Player = &player
		case "--unit":
			i++
			if i >= len(args) {
				return false, errors.New("--unit needs a value")
			}
			unitConst, err := strconv.Atoi(args[i])
			if err != nil {
				return false, fmt.Errorf("invalid unit %q: %w", args[i], err)
			}
			unitOptionSeen = true
			request.UnitConst = &unitConst
		case "--trigger":
			i++
			if i >= len(args) {
				return false, errors.New("--trigger needs a value")
			}
			triggerIndex, err := strconv.Atoi(args[i])
			if err != nil {
				return false, fmt.Errorf("invalid trigger %q: %w", args[i], err)
			}
			if triggerIndex < 0 {
				return false, errors.New("--trigger must be non-negative")
			}
			triggerOptionSeen = true
			request.TriggerID = &triggerIndex
		case "--trigger-prefix":
			i++
			if i >= len(args) {
				return false, errors.New("--trigger-prefix needs a value")
			}
			if strings.TrimSpace(args[i]) == "" {
				return false, errors.New("--trigger-prefix must be non-empty")
			}
			triggerPrefixOptionSeen = true
			request.TargetPrefix = args[i]
		default:
			return false, fmt.Errorf("unknown option %q", arg)
		}
	}
	if triggerOptionSeen && triggerPrefixOptionSeen {
		return false, errors.New("--trigger and --trigger-prefix are mutually exclusive")
	}
	kind := strings.ToLower(strings.TrimSpace(request.Kind))
	if playerOptionSeen && kind != "unit_caption" && kind != "unit-caption" && kind != "unit_caption_prefix" && kind != "unit-caption-prefix" && kind != "unit_caption_contains" && kind != "unit-caption-contains" && !isScenarioDeletePlanUnitTypeKind(request.Kind) && !isScenarioDeletePlanAreaKind(request.Kind) {
		return false, errors.New("--player is only valid for unit-caption, unit-caption-prefix, unit-caption-contains, unit-type, and units-area delete plans")
	}
	if unitOptionSeen && !isScenarioDeletePlanAreaKind(request.Kind) {
		return false, errors.New("--unit is only valid for units-area delete plans")
	}
	if (triggerOptionSeen || triggerPrefixOptionSeen) && !isScenarioDeletePlanChildTypeKind(request.Kind) {
		return false, errors.New("--trigger and --trigger-prefix are only valid for effect-type, condition-type, effect-text-prefix, and effect-text-contains delete plans")
	}
	return textOut, nil
}

func parseScenarioDeletePlanArea(target string) ([4]float64, error) {
	parts := strings.Split(target, ",")
	if len(parts) != 4 {
		return [4]float64{}, fmt.Errorf("units-area target must be x1,y1,x2,y2")
	}
	var coords [4]float64
	for i, part := range parts {
		value, err := strconv.ParseFloat(strings.TrimSpace(part), 64)
		if err != nil {
			return [4]float64{}, fmt.Errorf("invalid units-area coordinate %q: %w", part, err)
		}
		if value < 0 {
			return [4]float64{}, fmt.Errorf("units-area coordinates must be non-negative")
		}
		coords[i] = value
	}
	return coords, nil
}

func printScenarioEffects(report scenario.EffectsReport) {
	fmt.Printf("path: %s\n", report.Path)
	fmt.Printf("verification: %s\n", report.Verification)
	fmt.Printf("summary: version=%s total_effects=%d effect_types=%d\n", report.Version, report.Total, len(report.Types))
	printScenarioEffectBuckets("effect_types", report.Types, len(report.Types))
}

func printScenarioEffectsCensus(report scenario.EffectsCensusReport) {
	fmt.Printf("path: %s\n", report.Path)
	fmt.Printf("verification: %s\n", report.Verification)
	fmt.Printf("scenario: version=%s inflated_bytes=%d compressed_bytes=%d parsed_through=%s\n",
		report.Version, report.InflatedBytes, report.CompressedBytes, report.ParsedThroughSection)
	fmt.Printf("summary: triggers=%d effects=%d effect_types=%d\n", report.TriggerCount, report.EffectCount, len(report.Types))
	printScenarioEffectBuckets("effect_types", report.Types, len(report.Types))
}

func printScenarioEffectBuckets(label string, buckets []scenario.EffectTypeBucket, limit int) {
	if len(buckets) == 0 {
		return
	}
	if limit <= 0 || limit > len(buckets) {
		limit = len(buckets)
	}
	fmt.Printf("%s:\n", label)
	for _, bucket := range buckets[:limit] {
		fmt.Printf("- %d/%s count=%d\n", bucket.Type, bucket.TypeName, bucket.Count)
		for _, sample := range bucket.Sample {
			fmt.Printf("  sample trigger=%d effect=%d %s\n", sample.TriggerIndex, sample.EffectIndex, scenarioEffectSummaryText(sample))
			if sample.TriggerName != "" {
				fmt.Printf("    trigger_name: %s\n", sample.TriggerName)
			}
		}
	}
	if len(buckets) > limit {
		fmt.Printf("- ... %d more\n", len(buckets)-limit)
	}
}

func printScenarioStringCounts(label string, rows []scenario.StringCount) {
	if len(rows) == 0 {
		return
	}
	fmt.Printf("%s:\n", label)
	for _, row := range rows {
		fmt.Printf("- count=%d value=%q\n", row.Count, oneLine(row.Value, 160))
	}
}

func printScenarioIntCounts(label string, rows []scenario.IntCount) {
	if len(rows) == 0 {
		return
	}
	fmt.Printf("%s:\n", label)
	for _, row := range rows {
		fmt.Printf("- id=%d count=%d\n", row.ID, row.Count)
	}
}

func scenarioEffectSummaryText(effect scenario.EffectSummary) string {
	parts := []string{fmt.Sprintf("%d/%s", effect.Type, effect.TypeName)}
	addIntPart := func(name string, value int) {
		if value != 0 {
			parts = append(parts, fmt.Sprintf("%s=%d", name, value))
		}
	}
	addIntPart("unit", effect.UnitConst)
	addIntPart("source", effect.SourcePlayer)
	addIntPart("target", effect.TargetPlayer)
	addIntPart("target_trigger", effect.TargetTrigger)
	addIntPart("string", effect.StringID)
	addIntPart("timer", effect.TimerID)
	addIntPart("variable", effect.Variable)
	addIntPart("variable2", effect.Variable2)
	addIntPart("operation", effect.Operation)
	addIntPart("quantity", effect.Quantity)
	addIntPart("technology", effect.Technology)
	addIntPart("resource", effect.Resource)
	addIntPart("attribute", effect.ObjectAttribute)
	if len(effect.Location) > 0 {
		parts = append(parts, fmt.Sprintf("location=%v", effect.Location))
	}
	if len(effect.Area) > 0 {
		parts = append(parts, fmt.Sprintf("area=%v", effect.Area))
	}
	if len(effect.SelectedObjectIDs) > 0 {
		parts = append(parts, "selected="+intSampleText(effect.SelectedObjectIDs, 5))
	}
	if effect.Sound != "" {
		parts = append(parts, fmt.Sprintf("sound=%q", effect.Sound))
	}
	return strings.Join(parts, " ")
}

func intSampleText(values []int, limit int) string {
	if limit <= 0 || limit > len(values) {
		limit = len(values)
	}
	out := make([]string, 0, limit+1)
	for _, value := range values[:limit] {
		out = append(out, strconv.Itoa(value))
	}
	if len(values) > limit {
		out = append(out, fmt.Sprintf("...+%d", len(values)-limit))
	}
	return "[" + strings.Join(out, ",") + "]"
}

func floatSampleText(values []float64, limit int) string {
	if limit <= 0 || limit > len(values) {
		limit = len(values)
	}
	out := make([]string, 0, limit+1)
	for _, value := range values[:limit] {
		out = append(out, fmt.Sprintf("%.3f", value))
	}
	if len(values) > limit {
		out = append(out, fmt.Sprintf("...+%d", len(values)-limit))
	}
	return "[" + strings.Join(out, ",") + "]"
}

func oneLine(s string, max int) string {
	s = strings.Join(strings.Fields(s), " ")
	if max > 0 && len(s) > max {
		return truncate(s, max)
	}
	return s
}

func parseScenarioSmokeOptions(args []string) (scenario.SmokeRecipeOptions, error) {
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

func describeScenario(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: kit scen describe <file.aoe2scenario> [--section triggers|units|map|players|victory|messages|disables|ai|resources|all] [--full] [--json] [--no-path|--quiet-header]")
		os.Exit(2)
	}
	path := args[0]
	opts := scenario.DescribeOptions{}
	jsonOut := false
	printPath := true
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--section":
			i++
			if i >= len(args) {
				die("kit scen describe", fmt.Errorf("--section needs a value"))
			}
			opts.Sections = append(opts.Sections, args[i])
		case "--full":
			opts.Full = true
		case "--json":
			jsonOut = true
		case "--no-path", "--quiet-header":
			printPath = false
		default:
			die("kit scen describe", fmt.Errorf("unknown option %q", args[i]))
		}
	}
	scen, err := scenario.Open(path)
	if err != nil {
		die("kit scen describe", err)
	}
	desc := scen.Describe(opts)
	if jsonOut {
		printJSON(desc)
		return
	}
	printScenarioDescription(desc, printPath)
}

func printScenarioDescription(desc scenario.Description, printPath bool) {
	if printPath {
		fmt.Printf("path: %s\n", desc.Path)
	}
	fmt.Printf("version: %s\n", desc.Version)
	fmt.Printf("bytes: header=%d compressed_body=%d inflated_body=%d\n", desc.HeaderBytes, desc.CompressedBytes, desc.InflatedBytes)
	fmt.Printf("sections: %d\n", len(desc.SectionIndex))
	if desc.Map != nil {
		fmt.Printf("map: %dx%d tiles=%d terrain_types=%d\n", desc.Map.Width, desc.Map.Height, desc.Map.TileCount, len(desc.Map.TerrainCounts))
	}
	if desc.Triggers != nil {
		fmt.Printf("triggers: count=%d variables=%d graph_sha256=%s invariant_ok=%t\n", desc.Triggers.Count, desc.Triggers.Variables, desc.Triggers.GraphSHA256, desc.Triggers.InvariantOK)
	}
	if desc.Units != nil {
		fmt.Printf("units: total=%d player_sections=%d\n", desc.Units.Total, len(desc.Units.Sections))
	}
	if len(desc.Players) > 0 {
		fmt.Println("players:")
		for _, p := range desc.Players {
			if !p.Active && p.TribeName == "" && p.Civilization == "" && p.AIName == "" {
				continue
			}
			fmt.Printf("- P%d active=%t human=%t civ=%q tribe=%q ai=%q ai_type=%d\n", p.Player, p.Active, p.Human, p.Civilization, p.TribeName, p.AIName, p.AIType)
		}
	}
	if len(desc.AI) > 0 {
		fmt.Println("embedded scenario AI files:")
		for _, ai := range desc.AI {
			fmt.Printf("- %s bytes=%d sha256=%s\n", ai.Name, ai.ContentBytes, ai.ContentSHA256)
		}
	}
	if len(desc.Sections) > 0 {
		fmt.Println("full sections were requested; use --json to inspect the structured field dump.")
	}
}

func printScenarioCoverage(report scenario.CoverageReport) {
	fmt.Printf("path: %s\n", report.Path)
	fmt.Printf("version: %s verification=%s\n", report.Version, report.Verification)
	fmt.Printf("bytes: file=%d header=%d inflated_body=%d accounted=%d named=%d dark=%d gaps=%d\n",
		report.FileBytes, report.HeaderBytes, report.InflatedBodyBytes, report.AccountedBytes,
		report.NamedBytes, report.DarkBytes, report.GapBytes)
	for _, section := range report.Sections {
		fmt.Printf("- %s/%s %d..%d bytes=%d named=%d dark=%d gaps=%d status=%s\n",
			section.Space, section.Name, section.Start, section.End, section.Bytes,
			section.Named, section.Dark, section.Gaps, section.Status)
		for _, span := range section.DarkSpans {
			fmt.Printf("  dark-span %s %d..%d bytes=%d\n", span.Path, span.Start, span.End, span.Bytes)
		}
	}
}

func printScenarioCoverageCorpus(report scenario.CoverageCorpusReport) {
	fmt.Printf("roots=%s files=%d parsed=%d errors=%d spans=%d\n", strings.Join(report.Roots, ","), report.Files, report.Parsed, len(report.Errors), len(report.Spans))
	for _, span := range report.Spans {
		fmt.Printf("- version=%s path=%s bytes=%d occurrences=%d files=%d hashes=%d nonzero_bytes=%v triggers=%d..%d conditions=%d..%d\n",
			span.Version, span.Path, span.Bytes, span.Occurrences, span.Files, len(span.SHA256), span.NonZeroBytes,
			span.MinTriggerCount, span.MaxTriggerCount, span.MinConditionCount, span.MaxConditionCount)
	}
	for _, item := range report.Errors {
		fmt.Printf("error: %s: %s\n", item.Path, item.Error)
	}
}

func printScenarioTerrain(report scenario.TerrainReport) {
	fmt.Printf("path: %s\n", report.Path)
	fmt.Printf("verification: %s\n", report.Verification)
	if report.Filter != nil {
		fmt.Printf("filter: terrain_id=%d\n", report.Filter.ID)
	}
	fmt.Printf("summary: map=%dx%d tiles=%d returned_tiles=%d terrain_types=%d\n", report.Width, report.Height, report.TileCount, report.Returned, len(report.Aggregates))
	fmt.Println("aggregates:")
	for _, agg := range report.Aggregates {
		fmt.Printf("- terrain_id=%d count=%d bounds=(%d,%d)-(%d,%d) centroid=(%.2f,%.2f) components=%d largest=%d\n",
			agg.TerrainID, agg.Count, agg.Bounds.MinX, agg.Bounds.MinY, agg.Bounds.MaxX, agg.Bounds.MaxY,
			agg.Centroid.X, agg.Centroid.Y, agg.Components, agg.LargestComponent)
		for _, component := range agg.ComponentBounds {
			fmt.Printf("  - component=%d count=%d bounds=(%d,%d)-(%d,%d) centroid=(%.2f,%.2f)\n",
				component.Index, component.Count, component.Bounds.MinX, component.Bounds.MinY,
				component.Bounds.MaxX, component.Bounds.MaxY, component.Centroid.X, component.Centroid.Y)
		}
	}
	if len(report.Tiles) == 0 {
		fmt.Println("tiles: omitted without --id or --tiles; use --id N --text for filtered rows or --tiles for full per-tile output")
		return
	}
	fmt.Println("tiles:")
	for _, tile := range report.Tiles {
		fmt.Printf("- x=%d y=%d terrain_id=%d elevation=%d layer=%d unused=%s\n",
			tile.X, tile.Y, tile.TerrainID, tile.Elevation, tile.Layer, tile.UnusedHex)
	}
}

func printScenarioPaletteUsage(report scenario.PaletteUsageReport) {
	fmt.Printf("path: %s\n", report.Path)
	fmt.Printf("dat: %s\n", report.DatPath)
	fmt.Printf("verification: %s\n", report.Verification)
	fmt.Printf("summary: unit_types=%d placements=%d rows=%d\n", report.UnitTypes, report.Placements, len(report.Rows))
	fmt.Printf("palette_summary: multi_variant_type_count=%d artworks_available=%d artworks_used=%d\n",
		report.Summary.MultiVariantTypeCount, report.Summary.ArtworksAvailable, report.Summary.ArtworksUsed)
	if len(report.Summary.BiggestUntapped) > 0 {
		fmt.Println("biggest_untapped:")
		for _, row := range report.Summary.BiggestUntapped {
			fmt.Printf("- unit=%d %q graphic=%d %q placements=%d available_variant_count=%d used_variant_count=%d unused_variant_count=%d\n",
				row.UnitID, row.UnitName, row.StandingGraphic1, row.GraphicName, row.Placements,
				row.AvailableVariantCount, row.UsedVariantCount, row.UnusedVariantCount)
		}
	}
	for _, row := range report.Rows {
		available := "n/a"
		if row.AvailableVariantCount != nil {
			available = strconv.Itoa(*row.AvailableVariantCount)
		}
		fmt.Printf("- unit=%d %q placements=%d graphic=%d %q file=%q slp=%d angle_count=%d frame_count=%d sequence_type=%d class=%s confidence=%s available_variant_count=%s used_indices=%s unused_indices=%s raw_rotations=%s\n",
			row.UnitID, row.UnitName, row.Placements, row.StandingGraphic1, row.GraphicName,
			row.FileName, row.SLP, row.AngleCount, row.FrameCount, row.SequenceType, row.Classification, row.Confidence, available,
			intSampleText(row.UsedIndices, 64), intSampleText(row.UnusedIndices, 64), floatSampleText(row.RawRotations, 16))
		if row.Note != "" {
			fmt.Printf("  note: %s\n", row.Note)
		}
	}
}

type identityResult struct {
	Path            string                 `json:"path"`
	Kind            string                 `json:"kind"`
	FingerprintTier string                 `json:"fingerprint_tier"`
	Fingerprint     string                 `json:"fingerprint"`
	RegistryKey     string                 `json:"registry_key"`
	GraphSHA256     string                 `json:"graph_sha256,omitempty"`
	TriggerCount    int                    `json:"trigger_count,omitempty"`
	EffectCount     int                    `json:"effect_count,omitempty"`
	ConditionCount  int                    `json:"condition_count,omitempty"`
	MessageCount    int                    `json:"message_count,omitempty"`
	Registered      bool                   `json:"registered"`
	Known           *registry.Entry        `json:"known,omitempty"`
	RegistryPath    string                 `json:"registry_path,omitempty"`
	RegisteredName  string                 `json:"registered_name,omitempty"`
	Warnings        []string               `json:"warnings,omitempty"`
	AILoadout       *aifile.Loadout        `json:"ai_loadout,omitempty"`
	DataSet         replay.DataSetIdentity `json:"data_set_identity"`
	Players         []replay.PlayerSlot    `json:"players,omitempty"`
	KnownAI         *aifile.RegistryEntry  `json:"known_ai,omitempty"`
	AIRegistryPath  string                 `json:"ai_registry_path,omitempty"`
	RegisteredAI    bool                   `json:"registered_ai,omitempty"`
}

func identify(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: kit identify <file.aoe2scenario|file.aoe2record> [--registry known_scenarios.json] [--ai-registry known_ais.json] [--register NAME] [--family FAMILY] [--notes NOTES] [--register-ai NAME] [--ai-mod MOD] [--ai-notes NOTES]")
		os.Exit(2)
	}
	path := args[0]
	registryPath := "known_scenarios.json"
	aiRegistryPath := "known_ais.json"
	registerName := ""
	registerAIName := ""
	family := ""
	aiMod := ""
	description := ""
	notes := ""
	aiNotes := ""
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--registry":
			i++
			if i >= len(args) {
				die("kit identify", fmt.Errorf("--registry needs a path"))
			}
			registryPath = args[i]
		case "--ai-registry":
			i++
			if i >= len(args) {
				die("kit identify", fmt.Errorf("--ai-registry needs a path"))
			}
			aiRegistryPath = args[i]
		case "--register":
			i++
			if i >= len(args) {
				die("kit identify", fmt.Errorf("--register needs a name"))
			}
			registerName = args[i]
		case "--register-ai":
			i++
			if i >= len(args) {
				die("kit identify", fmt.Errorf("--register-ai needs a name"))
			}
			registerAIName = args[i]
		case "--family":
			i++
			if i >= len(args) {
				die("kit identify", fmt.Errorf("--family needs a value"))
			}
			family = args[i]
		case "--ai-mod":
			i++
			if i >= len(args) {
				die("kit identify", fmt.Errorf("--ai-mod needs a value"))
			}
			aiMod = args[i]
		case "--notes":
			i++
			if i >= len(args) {
				die("kit identify", fmt.Errorf("--notes needs a value"))
			}
			notes = args[i]
		case "--description":
			i++
			if i >= len(args) {
				die("kit identify", fmt.Errorf("--description needs a value"))
			}
			description = args[i]
		case "--ai-notes":
			i++
			if i >= len(args) {
				die("kit identify", fmt.Errorf("--ai-notes needs a value"))
			}
			aiNotes = args[i]
		default:
			die("kit identify", fmt.Errorf("unknown option %q", args[i]))
		}
	}
	result, err := fingerprintFile(path)
	if err != nil {
		die("kit identify", err)
	}
	result.RegistryPath = registryPath
	result.AIRegistryPath = aiRegistryPath
	reg, err := registry.Load(registryPath)
	if err != nil {
		die("kit identify", err)
	}
	aiReg, err := aifile.LoadRegistry(aiRegistryPath)
	if err != nil {
		die("kit identify", err)
	}
	if registerName != "" {
		entry := registry.Entry{Name: registerName, Family: family, Description: description, Tier: result.FingerprintTier, Fingerprint: result.Fingerprint, TriggerCount: result.TriggerCount, Notes: notes}
		if _, err := registry.Register(registryPath, result.RegistryKey, entry); err != nil {
			die("kit identify", err)
		}
		result.Registered = true
		result.RegisteredName = registerName
		result.Known = &entry
	} else if entry, ok := reg[result.RegistryKey]; ok {
		result.Registered = true
		result.Known = &entry
	}
	if result.AILoadout != nil {
		if registerAIName != "" {
			entry := aifile.RegistryEntry{
				Name:      registerAIName,
				Mod:       aiMod,
				Notes:     aiNotes,
				Signature: result.AILoadout.Signature,
			}
			for _, group := range result.AILoadout.Groups {
				entry.Files = append(entry.Files, group.Files...)
				if entry.Mod == "" {
					entry.Mod = group.SourceMod
				}
			}
			sort.Strings(entry.Files)
			aiReg[result.AILoadout.Signature] = entry
			if err := aifile.SaveRegistry(aiRegistryPath, aiReg); err != nil {
				die("kit identify", err)
			}
			result.KnownAI = &entry
			result.RegisteredAI = true
		} else if entry, ok := aiReg[result.AILoadout.Signature]; ok {
			result.KnownAI = &entry
		}
	}
	printJSON(result)
}

func fingerprintFile(path string) (identityResult, error) {
	info, err := aoe2.InspectFile(path)
	if err != nil {
		return identityResult{}, err
	}
	switch info.Kind {
	case "scenario":
		scen, err := scenario.Open(path)
		if err != nil {
			return identityResult{}, err
		}
		if scen.Triggers == nil || scen.Triggers.GraphSHA256 == "" {
			return identityResult{}, fmt.Errorf("scenario has no trigger graph fingerprint")
		}
		return identityResult{
			Path:            path,
			Kind:            "scenario",
			FingerprintTier: "trigger_graph",
			Fingerprint:     scen.Triggers.GraphSHA256,
			RegistryKey:     scen.Triggers.GraphSHA256,
			GraphSHA256:     scen.Triggers.GraphSHA256,
			TriggerCount:    scen.Triggers.Count,
		}, nil
	case "record", "zip":
		rec, err := replay.Open(path)
		if err != nil {
			return identityResult{}, err
		}
		if rec.TriggerGraph != nil {
			return identityResult{
				Path:            path,
				Kind:            info.Kind,
				FingerprintTier: "trigger_graph",
				Fingerprint:     rec.TriggerGraph.SHA256,
				RegistryKey:     rec.TriggerGraph.SHA256,
				GraphSHA256:     rec.TriggerGraph.SHA256,
				TriggerCount:    rec.TriggerGraph.TriggerCount,
				EffectCount:     rec.TriggerGraph.EffectCount,
				ConditionCount:  rec.TriggerGraph.ConditionCount,
				MessageCount:    rec.TriggerGraph.MessageCount,
				Warnings:        rec.TriggerGraph.Warnings,
				AILoadout:       rec.AILoadout,
				DataSet:         rec.DataSet,
				Players:         rec.Players,
			}, nil
		}
		if rec.Fallback != nil {
			warnings := append([]string{}, rec.Fallback.Warnings...)
			if rec.TriggerGraphErr != "" {
				warnings = append(warnings, "trigger_graph unavailable: "+rec.TriggerGraphErr)
			}
			key := rec.Fallback.Tier + ":" + rec.Fallback.SHA256
			return identityResult{
				Path:            path,
				Kind:            info.Kind,
				FingerprintTier: rec.Fallback.Tier,
				Fingerprint:     rec.Fallback.SHA256,
				RegistryKey:     key,
				Warnings:        warnings,
				AILoadout:       rec.AILoadout,
				DataSet:         rec.DataSet,
				Players:         rec.Players,
			}, nil
		}
		return identityResult{}, fmt.Errorf("record fingerprint unavailable: trigger_graph=%s fallback=%s", rec.TriggerGraphErr, rec.FallbackErr)
	default:
		return identityResult{}, fmt.Errorf("%s is %q, not scenario or record", path, info.Kind)
	}
}

type collectReport struct {
	Root          string         `json:"root"`
	RegistryPath  string         `json:"registry_path"`
	Scanned       int            `json:"scanned"`
	Fingerprinted int            `json:"fingerprinted"`
	Known         int            `json:"known"`
	New           int            `json:"new"`
	Registered    int            `json:"registered"`
	DryRun        bool           `json:"dry_run"`
	Items         []collectItem  `json:"items"`
	Errors        []collectError `json:"errors,omitempty"`
}

type collectItem struct {
	Path            string          `json:"path"`
	Kind            string          `json:"kind"`
	FingerprintTier string          `json:"fingerprint_tier"`
	Fingerprint     string          `json:"fingerprint"`
	RegistryKey     string          `json:"registry_key"`
	Known           bool            `json:"known"`
	Registered      bool            `json:"registered"`
	Entry           *registry.Entry `json:"entry,omitempty"`
}

type collectError struct {
	Path  string `json:"path"`
	Error string `json:"error"`
}

func collect(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: kit collect <folder> [--registry known_scenarios.json] [--dry-run]")
		os.Exit(2)
	}
	root := args[0]
	registryPath := "known_scenarios.json"
	dryRun := false
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--registry":
			i++
			if i >= len(args) {
				die("kit collect", fmt.Errorf("--registry needs a path"))
			}
			registryPath = args[i]
		case "--dry-run":
			dryRun = true
		default:
			die("kit collect", fmt.Errorf("unknown option %q", args[i]))
		}
	}
	reg, err := registry.Load(registryPath)
	if err != nil {
		die("kit collect", err)
	}
	report := collectReport{Root: root, RegistryPath: registryPath, DryRun: dryRun}
	err = filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			report.Errors = append(report.Errors, collectError{Path: path, Error: walkErr.Error()})
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if !collectablePath(path) {
			return nil
		}
		report.Scanned++
		result, err := fingerprintFile(path)
		if err != nil {
			report.Errors = append(report.Errors, collectError{Path: path, Error: err.Error()})
			return nil
		}
		report.Fingerprinted++
		item := collectItem{
			Path:            path,
			Kind:            result.Kind,
			FingerprintTier: result.FingerprintTier,
			Fingerprint:     result.Fingerprint,
			RegistryKey:     result.RegistryKey,
		}
		if entry, ok := reg[result.RegistryKey]; ok {
			report.Known++
			item.Known = true
			item.Entry = &entry
		} else {
			report.New++
			entry := registry.Entry{
				Name:         "UNLABELED " + result.FingerprintTier + " " + shortHash(result.Fingerprint),
				Description:  "Needs human-reviewed orientation text so this fingerprint teaches a fresh AI what scenario/map this is.",
				Tier:         result.FingerprintTier,
				Fingerprint:  result.Fingerprint,
				TriggerCount: result.TriggerCount,
				Notes:        "Added by kit collect; replace placeholder name/description after human review.",
			}
			item.Entry = &entry
			if !dryRun {
				reg[result.RegistryKey] = entry
				item.Registered = true
				report.Registered++
			}
		}
		report.Items = append(report.Items, item)
		return nil
	})
	if err != nil {
		die("kit collect", err)
	}
	if !dryRun {
		if err := registry.Save(registryPath, reg); err != nil {
			die("kit collect", err)
		}
	}
	printJSON(report)
}

func runAI(args []string) {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: kit ai <fingerprint|lint|diff> <file-or-folder> [args...]")
		os.Exit(2)
	}
	switch args[0] {
	case "fingerprint":
		aiFingerprint(args[1:])
	case "lint":
		textOut := false
		for _, arg := range args[2:] {
			switch arg {
			case "--text":
				textOut = true
			default:
				die("kit ai lint", fmt.Errorf("unknown option %q", arg))
			}
		}
		report, err := aifile.LintPath(args[1])
		if err != nil {
			die("kit ai lint", err)
		}
		if textOut {
			printAILint(report)
		} else {
			printJSON(report)
		}
		if !report.OK {
			os.Exit(1)
		}
	case "diff":
		if len(args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: kit ai diff <before-file-or-folder> <after-file-or-folder> [--text]")
			os.Exit(2)
		}
		textOut := false
		for _, arg := range args[3:] {
			switch arg {
			case "--text":
				textOut = true
			default:
				die("kit ai diff", fmt.Errorf("unknown option %q", arg))
			}
		}
		report, err := aifile.DiffPaths(args[1], args[2])
		if err != nil {
			die("kit ai diff", err)
		}
		if textOut {
			printAIDiff(report)
		} else {
			printJSON(report)
		}
	default:
		fmt.Fprintln(os.Stderr, "usage: kit ai <fingerprint|lint|diff> <file-or-folder> [args...]")
		os.Exit(2)
	}
}

func runPlayer(args []string) {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: kit player <stats> <profileId> [--match-type N] [--cache-dir DIR] [--force] [--text]")
		os.Exit(2)
	}
	switch args[0] {
	case "stats":
		profileID := args[1]
		textOut := false
		cacheDir := ""
		force := false
		matchType := 3
		for i := 2; i < len(args); i++ {
			switch args[i] {
			case "--text":
				textOut = true
			case "--force":
				force = true
			case "--cache-dir":
				i++
				if i >= len(args) {
					die("kit player stats", fmt.Errorf("--cache-dir needs a path"))
				}
				cacheDir = args[i]
			case "--match-type":
				i++
				if i >= len(args) {
					die("kit player stats", fmt.Errorf("--match-type needs an integer"))
				}
				value, err := strconv.Atoi(args[i])
				if err != nil || value <= 0 {
					die("kit player stats", fmt.Errorf("--match-type needs a positive integer"))
				}
				matchType = value
			default:
				die("kit player stats", fmt.Errorf("unknown option %q", args[i]))
			}
		}
		client := aoems.NewClient(cacheDir)
		report, err := client.PlayerStats(aoems.PlayerStatsOptions{ProfileID: profileID, MatchType: matchType, Force: force})
		if err != nil {
			die("kit player stats", err)
		}
		if textOut {
			printPlayerStats(report)
		} else {
			printJSON(report)
		}
	default:
		fmt.Fprintln(os.Stderr, "usage: kit player <stats> <profileId> [--match-type N] [--cache-dir DIR] [--force] [--text]")
		os.Exit(2)
	}
}

func printPlayerStats(report *aoems.PlayerStatsReport) {
	fmt.Printf("profile_id: %s\n", report.ProfileID)
	fmt.Printf("method: %s\n", report.Method)
	fmt.Printf("verification: %s\n", report.Verification)
	fmt.Printf("match_type: %d cached=%v", report.MatchType, report.Cached)
	if report.CachePath != "" {
		fmt.Printf(" cache=%s", report.CachePath)
	}
	if report.AgeSeconds > 0 {
		fmt.Printf(" age=%ds", report.AgeSeconds)
	}
	fmt.Println()
	if report.User.UserName != "" || report.User.ELO != nil || report.User.PlayerStanding != nil {
		fmt.Printf("user: name=%q elo=%s standing=%s avatar=%q\n",
			report.User.UserName, anyText(report.User.ELO), anyText(report.User.PlayerStanding), report.User.AvatarURL)
	}
	if len(report.CareerStats) > 0 {
		keys := []string{"totalGames", "totalWins", "unitsKilled", "unitsLost", "buildingsRaised", "buildingsRazed", "buildingsLost", "castlesBuilt", "trebsBuilt", "farmsBuilt"}
		var parts []string
		for _, key := range keys {
			if value, ok := statLookup(report.CareerStats, key); ok {
				parts = append(parts, fmt.Sprintf("%s=%s", key, anyText(value)))
			}
		}
		if len(parts) > 0 {
			fmt.Printf("career: %s\n", strings.Join(parts, " "))
		} else {
			fmt.Printf("career: keys=%d\n", len(report.CareerStats))
		}
	}
	if rows := mpStatRows(report.MPStatList); len(rows) > 0 {
		fmt.Printf("mp_stat_list: %d row(s)\n", len(rows))
		for i, row := range rows {
			if i >= 4 {
				fmt.Printf("- ... %d more\n", len(rows)-i)
				break
			}
			fmt.Printf("- %s\n", strings.Join(mapSummary(row), " "))
		}
	}
	if len(report.Warnings) > 0 {
		fmt.Println("warnings:")
		for _, warning := range report.Warnings {
			fmt.Printf("- %s\n", warning)
		}
	}
}

func fetchAllReplayPOVs(client *aoems.Client, gameID string, seedProfile string, rosterPath string, outDir string, force bool) (*aoems.ReplayFetchAllReport, error) {
	report := &aoems.ReplayFetchAllReport{
		GameID:       gameID,
		SeedProfile:  seedProfile,
		Method:       "microsoft_ageii_get_match_replay_all_povs_from_replay_roster",
		Verification: "downloaded_public_replays_or_cache_not_engine_verified",
		OutDir:       outDir,
	}
	if rosterPath == "" {
		if seedProfile == "" {
			return nil, fmt.Errorf("--all-povs needs a roster source: pass --profile <seedProfileId> to fetch one POV first, or --from-roster <replay.aoe2record|zip> to reuse an existing replay roster")
		}
		seed, err := client.FetchReplay(aoems.ReplayFetchOptions{GameID: gameID, ProfileID: seedProfile, OutDir: outDir, Force: force})
		if err != nil {
			return nil, err
		}
		report.Fetched = append(report.Fetched, *seed)
		rosterPath = seed.Path
	}
	profiles, warnings, err := replayRosterProfileIDs(rosterPath)
	if err != nil {
		return nil, err
	}
	report.Warnings = append(report.Warnings, warnings...)
	report.Profiles = profiles
	if len(profiles) == 0 {
		report.Warnings = append(report.Warnings, "no positive profile ids found in replay roster; parser may not expose this replay mode's roster yet")
		return report, nil
	}
	already := map[string]bool{}
	for _, fetched := range report.Fetched {
		already[fetched.ProfileID] = true
	}
	for _, profileID := range profiles {
		if already[profileID] {
			continue
		}
		fetched, err := client.FetchReplay(aoems.ReplayFetchOptions{GameID: gameID, ProfileID: profileID, OutDir: outDir, Force: force})
		if err != nil {
			report.Warnings = append(report.Warnings, fmt.Sprintf("profile %s fetch failed: %v", profileID, err))
			continue
		}
		report.Fetched = append(report.Fetched, *fetched)
	}
	return report, nil
}

func replayRosterProfileIDs(path string) ([]string, []string, error) {
	rec, err := replay.Open(path)
	if err != nil {
		return nil, nil, err
	}
	seen := map[string]bool{}
	var ids []string
	var warnings []string
	if len(rec.Players) == 0 {
		warnings = append(warnings, "replay roster parsed with zero player slots")
	}
	for _, player := range rec.Players {
		if !player.Active || player.ProfileID <= 0 {
			continue
		}
		id := strconv.Itoa(player.ProfileID)
		if seen[id] {
			continue
		}
		seen[id] = true
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids, warnings, nil
}

func printReplayFetch(report *aoems.ReplayFetchReport) {
	fmt.Printf("game_id: %s profile_id: %s\n", report.GameID, report.ProfileID)
	fmt.Printf("method: %s\n", report.Method)
	fmt.Printf("verification: %s\n", report.Verification)
	fmt.Printf("path: %s\n", report.Path)
	fmt.Printf("cached: %v bytes=%d", report.Cached, report.Bytes)
	if report.StatusCode != 0 {
		fmt.Printf(" status=%d", report.StatusCode)
	}
	if report.ContentType != "" {
		fmt.Printf(" content_type=%q", report.ContentType)
	}
	fmt.Println()
	if len(report.Warnings) > 0 {
		fmt.Println("warnings:")
		for _, warning := range report.Warnings {
			fmt.Printf("- %s\n", warning)
		}
	}
}

func printReplayFetchAll(report *aoems.ReplayFetchAllReport) {
	fmt.Printf("game_id: %s seed_profile=%s\n", report.GameID, report.SeedProfile)
	fmt.Printf("method: %s\n", report.Method)
	fmt.Printf("verification: %s\n", report.Verification)
	fmt.Printf("out_dir: %s\n", report.OutDir)
	if len(report.Profiles) > 0 {
		fmt.Printf("profiles: %s\n", strings.Join(report.Profiles, ", "))
	}
	fmt.Printf("fetched: %d\n", len(report.Fetched))
	for _, fetched := range report.Fetched {
		fmt.Printf("- profile=%s cached=%v bytes=%d path=%s\n", fetched.ProfileID, fetched.Cached, fetched.Bytes, fetched.Path)
	}
	if len(report.Warnings) > 0 {
		fmt.Println("warnings:")
		for _, warning := range report.Warnings {
			fmt.Printf("- %s\n", warning)
		}
	}
}

func statLookup(stats map[string]any, key string) (any, bool) {
	if value, ok := stats[key]; ok {
		return value, true
	}
	lower := strings.ToLower(key)
	for candidate, value := range stats {
		if strings.ToLower(candidate) == lower {
			return value, true
		}
	}
	return nil, false
}

func mpStatRows(value any) []map[string]any {
	switch v := value.(type) {
	case nil:
		return nil
	case []map[string]any:
		return v
	case map[string]any:
		return []map[string]any{v}
	case []any:
		var rows []map[string]any
		for _, item := range v {
			if row, ok := item.(map[string]any); ok {
				rows = append(rows, row)
			}
		}
		return rows
	default:
		return []map[string]any{{"value": v}}
	}
}

func mapSummary(row map[string]any) []string {
	keys := make([]string, 0, len(row))
	for key := range row {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var out []string
	for _, key := range keys {
		out = append(out, fmt.Sprintf("%s=%s", key, anyText(row[key])))
	}
	return out
}

func anyText(value any) string {
	if value == nil {
		return ""
	}
	switch v := value.(type) {
	case string:
		return v
	case json.Number:
		return v.String()
	default:
		raw, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprint(v)
		}
		return string(raw)
	}
}

func printAILint(report *aifile.LintReport) {
	status := "OK"
	if !report.OK {
		status = "FAIL"
	}
	fmt.Printf("path: %s\n", report.Path)
	fmt.Printf("status: %s\n", status)
	fmt.Printf("summary: files=%d errors=%d warnings=%d\n", report.Summary.Files, report.Summary.Errors, report.Summary.Warns)
	if len(report.Issues) == 0 {
		fmt.Println("issues: none")
		return
	}
	fmt.Println("issues:")
	for _, issue := range report.Issues {
		file := ""
		if issue.File != "" {
			file = issue.File + ": "
		}
		fmt.Printf("- [%s] %s%s: %s\n", issue.Severity, file, issue.Code, issue.Message)
	}
}

func printAIDiff(report *aifile.DiffReport) {
	fmt.Printf("before: %s\n", report.Before)
	fmt.Printf("after: %s\n", report.After)
	fmt.Printf("same: %t\n", report.Same)
	if len(report.Changes) == 0 {
		fmt.Println("changes: none")
		return
	}
	fmt.Println("changes:")
	for _, change := range report.Changes {
		file := ""
		if change.File != "" {
			file = " " + change.File
		}
		fmt.Printf("- %s%s: %v -> %v\n", change.Kind, file, change.Before, change.After)
	}
}

func aiFingerprint(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: kit ai fingerprint <file-or-folder> [--registry known_ais.json] [--register NAME] [--signature AI_FILES_SIGNATURE] [--mod MOD] [--notes NOTES]")
		os.Exit(2)
	}
	path := args[0]
	registryPath := "known_ais.json"
	registerName := ""
	signature := ""
	mod := ""
	notes := ""
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--registry":
			i++
			if i >= len(args) {
				die("kit ai fingerprint", fmt.Errorf("--registry needs a path"))
			}
			registryPath = args[i]
		case "--register":
			i++
			if i >= len(args) {
				die("kit ai fingerprint", fmt.Errorf("--register needs a name"))
			}
			registerName = args[i]
		case "--signature":
			i++
			if i >= len(args) {
				die("kit ai fingerprint", fmt.Errorf("--signature needs a value"))
			}
			signature = args[i]
		case "--mod":
			i++
			if i >= len(args) {
				die("kit ai fingerprint", fmt.Errorf("--mod needs a value"))
			}
			mod = args[i]
		case "--notes":
			i++
			if i >= len(args) {
				die("kit ai fingerprint", fmt.Errorf("--notes needs a value"))
			}
			notes = args[i]
		default:
			die("kit ai fingerprint", fmt.Errorf("unknown option %q", args[i]))
		}
	}
	fp, err := aifile.FingerprintPath(path)
	if err != nil {
		die("kit ai fingerprint", err)
	}
	reg, err := aifile.LoadRegistry(registryPath)
	if err != nil {
		die("kit ai fingerprint", err)
	}
	key := signature
	if key == "" {
		key = fp.RegistryKey
	}
	if registerName != "" {
		entry := aifile.RegistryEntry{
			Name:          registerName,
			Mod:           mod,
			Notes:         notes,
			Signature:     signature,
			ContentSHA256: fp.SHA256,
		}
		for _, file := range fp.Files {
			entry.Files = append(entry.Files, file.RelativePath)
		}
		sort.Strings(entry.Files)
		reg[key] = entry
		if err := aifile.SaveRegistry(registryPath, reg); err != nil {
			die("kit ai fingerprint", err)
		}
		fp.Known = &entry
	} else if entry, ok := reg[key]; ok {
		fp.Known = &entry
	} else {
		for _, entry := range reg {
			if entry.ContentSHA256 == fp.SHA256 {
				copy := entry
				fp.Known = &copy
				break
			}
		}
	}
	printJSON(fp)
}

func collectablePath(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".aoe2scenario", ".aoe2record", ".zip":
		return true
	default:
		return false
	}
}

func shortHash(hash string) string {
	if len(hash) <= 12 {
		return hash
	}
	return hash[:12]
}

func runReplay(args []string) {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: kit replay <info|summary|graph|triggers|trigger-neighborhood|diff-triggers|coverage|opaque-spans|frontier|corpus|opaque-clusters|ai-manifest|effective-data|effective-units|datamod-check|fetch|diff-state|scan-value|header-anchors|objects|object-state|object-shapes|spawns|lifecycle|postgame|camera|sync|sync-log|checksum-phase|checksum-probe|combat|deaths|player-series|playtest|carrier|sidecar-sync|xs-telemetry|events|actions|player-events|chat|feedback|story|player-profile|inbox|telemetry|issues|unknowns> <path> [options]")
		os.Exit(2)
	}
	switch args[0] {
	case "info", "inspect":
		textOut := false
		includeTiles := false
		full := false
		memReport := false
		for _, arg := range args[2:] {
			switch arg {
			case "--json":
				textOut = false
			case "--text":
				textOut = true
			case "--tiles":
				includeTiles = true
			case "--full":
				full = true
			case "--mem-report":
				memReport = true
			default:
				die("kit replay info", fmt.Errorf("unknown option %q", arg))
			}
		}
		if memReport {
			defer replay.TrackMemory(os.Stderr)()
		}
		if full {
			if textOut {
				die("kit replay info", fmt.Errorf("--full is JSON-only; omit --text"))
			}
			rec := openReplayArg(args[1])
			printJSON(rec)
			return
		}
		report, err := replay.BuildSummaryWithOptions(args[1], replay.SummaryOptions{IncludeMapTiles: includeTiles})
		if err != nil {
			die("kit replay info", err)
		}
		if textOut {
			printReplaySummary(report)
		} else {
			printJSON(report)
		}
	case "summary":
		textOut := false
		includeTiles := false
		for _, arg := range args[2:] {
			switch arg {
			case "--text":
				textOut = true
			case "--tiles":
				includeTiles = true
			default:
				die("kit replay summary", fmt.Errorf("unknown option %q", arg))
			}
		}
		report, err := replay.BuildSummaryWithOptions(args[1], replay.SummaryOptions{IncludeMapTiles: includeTiles})
		if err != nil {
			die("kit replay summary", err)
		}
		if textOut {
			printReplaySummary(report)
		} else {
			printJSON(report)
		}
	case "graph":
		rec := openReplayArg(args[1])
		if rec.TriggerGraph == nil {
			die("kit replay", fmt.Errorf("trigger graph unavailable: %s", rec.TriggerGraphErr))
		}
		printJSON(rec.TriggerGraph)
	case "triggers":
		opts, textOut, err := parseReplayTriggerSearchArgs(args[2:])
		if err != nil {
			die("kit replay triggers", err)
		}
		rec := openReplayArg(args[1])
		if rec.TriggerGraph == nil {
			die("kit replay triggers", fmt.Errorf("trigger graph unavailable: %s", rec.TriggerGraphErr))
		}
		report, err := triggergraph.Search(rec.TriggerGraph, opts)
		if err != nil {
			die("kit replay triggers", err)
		}
		if textOut {
			printReplayTriggerSearch(report)
		} else {
			printJSON(report)
		}
	case "trigger-neighborhood":
		opts, textOut, err := parseReplayTriggerNeighborhoodArgs(args[2:])
		if err != nil {
			die("kit replay trigger-neighborhood", err)
		}
		rec := openReplayArg(args[1])
		if rec.TriggerGraph == nil {
			die("kit replay trigger-neighborhood", fmt.Errorf("trigger graph unavailable: %s", rec.TriggerGraphErr))
		}
		report, err := triggergraph.Neighborhood(rec.TriggerGraph, opts)
		if err != nil {
			die("kit replay trigger-neighborhood", err)
		}
		if textOut {
			printReplayTriggerNeighborhood(report)
		} else {
			printJSON(report)
		}
	case "diff-triggers":
		if len(args) < 3 {
			die("kit replay diff-triggers", fmt.Errorf("usage: kit replay diff-triggers <before.aoe2record|zip> <after.aoe2record|zip> [--limit N] [--text]"))
		}
		opts, textOut, err := parseTriggerGraphDiffOptions(args[3:], "kit replay diff-triggers")
		if err != nil {
			die("kit replay diff-triggers", err)
		}
		report, err := buildReplayTriggerGraphDiff(args[1], args[2], opts)
		if err != nil {
			die("kit replay diff-triggers", err)
		}
		if textOut {
			printTriggerGraphDiff(report)
		} else {
			printJSON(report)
		}
	case "fetch":
		gameID := args[1]
		profileID := ""
		outDir := "."
		cacheDir := ""
		rosterPath := ""
		force := false
		allPOVs := false
		textOut := false
		for i := 2; i < len(args); i++ {
			switch args[i] {
			case "--text":
				textOut = true
			case "--force":
				force = true
			case "--all-povs":
				allPOVs = true
			case "--profile":
				i++
				if i >= len(args) {
					die("kit replay fetch", fmt.Errorf("--profile needs a profile id"))
				}
				profileID = args[i]
			case "--out":
				i++
				if i >= len(args) {
					die("kit replay fetch", fmt.Errorf("--out needs a directory"))
				}
				outDir = args[i]
			case "--cache-dir":
				i++
				if i >= len(args) {
					die("kit replay fetch", fmt.Errorf("--cache-dir needs a directory"))
				}
				cacheDir = args[i]
			case "--from-roster":
				i++
				if i >= len(args) {
					die("kit replay fetch", fmt.Errorf("--from-roster needs a replay path"))
				}
				rosterPath = args[i]
			default:
				die("kit replay fetch", fmt.Errorf("unknown option %q", args[i]))
			}
		}
		client := aoems.NewClient(cacheDir)
		if allPOVs {
			report, err := fetchAllReplayPOVs(client, gameID, profileID, rosterPath, outDir, force)
			if err != nil {
				die("kit replay fetch", err)
			}
			if textOut {
				printReplayFetchAll(report)
			} else {
				printJSON(report)
			}
			return
		}
		if profileID == "" {
			die("kit replay fetch", fmt.Errorf("--profile is required unless --all-povs can use --from-roster"))
		}
		report, err := client.FetchReplay(aoems.ReplayFetchOptions{GameID: gameID, ProfileID: profileID, OutDir: outDir, Force: force})
		if err != nil {
			die("kit replay fetch", err)
		}
		if textOut {
			printReplayFetch(report)
		} else {
			printJSON(report)
		}
	case "coverage":
		checkReplayArg(args[1])
		textOut := false
		failOnOpaque := false
		for _, arg := range args[2:] {
			switch arg {
			case "--text":
				textOut = true
			case "--json":
				textOut = false
			case "--fail-on-opaque":
				failOnOpaque = true
			default:
				die("kit replay coverage", fmt.Errorf("unknown option %q", arg))
			}
		}
		report, err := replay.BuildCoverage(args[1])
		if err != nil {
			die("kit replay coverage", err)
		}
		if textOut {
			printReplayCoverage(report)
		} else {
			printJSON(report)
		}
		if failOnOpaque && (report.Summary.HeaderOpaqueBytes > 0 || report.Summary.BodyOpaqueBytes > 0) {
			fmt.Fprintf(os.Stderr, "kit replay coverage: opaque bytes remain: header=%d body=%d\n", report.Summary.HeaderOpaqueBytes, report.Summary.BodyOpaqueBytes)
			os.Exit(1)
		}
	case "opaque-spans":
		checkReplayArg(args[1])
		textOut := false
		opts := replay.OpaqueSpanOptions{}
		for i := 2; i < len(args); i++ {
			switch args[i] {
			case "--text":
				textOut = true
			case "--space":
				i++
				if i >= len(args) {
					die("kit replay opaque-spans", fmt.Errorf("--space needs a value"))
				}
				opts.Space = args[i]
			case "--min-bytes":
				i++
				if i >= len(args) {
					die("kit replay opaque-spans", fmt.Errorf("--min-bytes needs an integer"))
				}
				value, err := strconv.Atoi(args[i])
				if err != nil || value < 0 {
					die("kit replay opaque-spans", fmt.Errorf("--min-bytes needs a non-negative integer"))
				}
				opts.MinBytes = value
			case "--limit":
				i++
				if i >= len(args) {
					die("kit replay opaque-spans", fmt.Errorf("--limit needs an integer"))
				}
				value, err := strconv.Atoi(args[i])
				if err != nil || value < 0 {
					die("kit replay opaque-spans", fmt.Errorf("--limit needs a non-negative integer"))
				}
				opts.Limit = value
			default:
				die("kit replay opaque-spans", fmt.Errorf("unknown option %q", args[i]))
			}
		}
		report, err := replay.BuildOpaqueSpans(args[1], opts)
		if err != nil {
			die("kit replay opaque-spans", err)
		}
		if textOut {
			printReplayOpaqueSpans(report)
		} else {
			printJSON(report)
		}
	case "frontier":
		checkReplayArg(args[1])
		textOut := false
		opts := replay.FrontierOptions{MinDuplicateBytes: 256, Limit: 8}
		for i := 2; i < len(args); i++ {
			switch args[i] {
			case "--text":
				textOut = true
			case "--json":
				textOut = false
			case "--min-duplicate-bytes":
				i++
				if i >= len(args) {
					die("kit replay frontier", fmt.Errorf("--min-duplicate-bytes needs an integer"))
				}
				value, err := strconv.Atoi(args[i])
				if err != nil || value < 0 {
					die("kit replay frontier", fmt.Errorf("--min-duplicate-bytes needs a non-negative integer"))
				}
				opts.MinDuplicateBytes = value
			case "--limit":
				i++
				if i >= len(args) {
					die("kit replay frontier", fmt.Errorf("--limit needs an integer"))
				}
				value, err := strconv.Atoi(args[i])
				if err != nil || value < 0 {
					die("kit replay frontier", fmt.Errorf("--limit needs a non-negative integer"))
				}
				opts.Limit = value
			default:
				die("kit replay frontier", fmt.Errorf("unknown option %q", args[i]))
			}
		}
		report, err := replay.BuildFrontier(args[1], opts)
		if err != nil {
			die("kit replay frontier", err)
		}
		if textOut {
			printReplayFrontier(report)
		} else {
			printJSON(report)
		}
	case "corpus":
		textOut := false
		opts := replay.CorpusOptions{Recursive: true, IncludeZip: true, UnknownSamples: 2}
		for i := 2; i < len(args); i++ {
			switch args[i] {
			case "--text":
				textOut = true
			case "--flat":
				opts.Recursive = false
			case "--no-zip":
				opts.IncludeZip = false
			case "--unknown-samples":
				i++
				if i >= len(args) {
					die("kit replay corpus", fmt.Errorf("--unknown-samples needs an integer"))
				}
				value, err := strconv.Atoi(args[i])
				if err != nil || value < 0 {
					die("kit replay corpus", fmt.Errorf("--unknown-samples needs a non-negative integer"))
				}
				opts.UnknownSamples = value
			default:
				die("kit replay corpus", fmt.Errorf("unknown option %q", args[i]))
			}
		}
		fmt.Fprintln(os.Stderr, "workers: 1 (serial replay corpus)")
		report, err := replay.BuildReplayCorpus(args[1], opts)
		if err != nil {
			die("kit replay corpus", err)
		}
		if textOut {
			printReplayCorpus(report)
		} else {
			printJSON(report)
		}
	case "opaque-clusters":
		textOut := false
		opts := replay.OpaqueClusterOptions{Recursive: true, IncludeZip: true, MinBytes: 64, Limit: 20, SampleLimit: 2}
		for i := 2; i < len(args); i++ {
			switch args[i] {
			case "--text":
				textOut = true
			case "--flat":
				opts.Recursive = false
			case "--no-zip":
				opts.IncludeZip = false
			case "--space":
				i++
				if i >= len(args) {
					die("kit replay opaque-clusters", fmt.Errorf("--space needs a value"))
				}
				opts.Space = args[i]
			case "--min-bytes":
				i++
				if i >= len(args) {
					die("kit replay opaque-clusters", fmt.Errorf("--min-bytes needs an integer"))
				}
				value, err := strconv.Atoi(args[i])
				if err != nil || value < 0 {
					die("kit replay opaque-clusters", fmt.Errorf("--min-bytes needs a non-negative integer"))
				}
				opts.MinBytes = value
			case "--limit":
				i++
				if i >= len(args) {
					die("kit replay opaque-clusters", fmt.Errorf("--limit needs an integer"))
				}
				value, err := strconv.Atoi(args[i])
				if err != nil || value < 0 {
					die("kit replay opaque-clusters", fmt.Errorf("--limit needs a non-negative integer"))
				}
				opts.Limit = value
			case "--samples":
				i++
				if i >= len(args) {
					die("kit replay opaque-clusters", fmt.Errorf("--samples needs an integer"))
				}
				value, err := strconv.Atoi(args[i])
				if err != nil || value < 0 {
					die("kit replay opaque-clusters", fmt.Errorf("--samples needs a non-negative integer"))
				}
				opts.SampleLimit = value
			default:
				die("kit replay opaque-clusters", fmt.Errorf("unknown option %q", args[i]))
			}
		}
		report, err := replay.BuildOpaqueClusters(args[1], opts)
		if err != nil {
			die("kit replay opaque-clusters", err)
		}
		if textOut {
			printReplayOpaqueClusters(report)
		} else {
			printJSON(report)
		}
	case "ai-manifest":
		checkReplayArg(args[1])
		textOut := false
		opts := replay.AIManifestOptions{}
		for i := 2; i < len(args); i++ {
			switch args[i] {
			case "--text":
				textOut = true
			case "--promisory-root":
				i++
				if i >= len(args) {
					die("kit replay ai-manifest", fmt.Errorf("--promisory-root needs a folder"))
				}
				opts.PromisoryRoot = args[i]
			default:
				die("kit replay ai-manifest", fmt.Errorf("unknown option %q", args[i]))
			}
		}
		report, err := replay.BuildAIManifest(args[1], opts)
		if err != nil {
			die("kit replay ai-manifest", err)
		}
		if textOut {
			printReplayAIManifest(report)
		} else {
			printJSON(report)
		}
	case "effective-data":
		checkReplayArg(args[1])
		textOut := false
		treeOut := false
		limitSet := false
		opts := replay.EffectiveDataOptions{TailOffset: replay.DefaultEffectiveAvailabilityOffset, Count: replay.DefaultEffectiveAvailabilityCount, Limit: 40}
		for i := 2; i < len(args); i++ {
			switch args[i] {
			case "--text":
				textOut = true
			case "--tree":
				treeOut = true
				textOut = true
			case "--all":
				opts.IncludeAll = true
			case "--dat":
				i++
				if i >= len(args) {
					die("kit replay effective-data", fmt.Errorf("--dat needs a path"))
				}
				opts.DatPath = args[i]
			case "--player":
				i++
				if i >= len(args) {
					die("kit replay effective-data", fmt.Errorf("--player needs an integer"))
				}
				value, err := strconv.Atoi(args[i])
				if err != nil || value < 0 {
					die("kit replay effective-data", fmt.Errorf("--player needs a non-negative integer"))
				}
				opts.Player = value
			case "--tail-offset":
				i++
				if i >= len(args) {
					die("kit replay effective-data", fmt.Errorf("--tail-offset needs an integer"))
				}
				value, err := strconv.Atoi(args[i])
				if err != nil || value < 0 {
					die("kit replay effective-data", fmt.Errorf("--tail-offset needs a non-negative integer"))
				}
				opts.TailOffset = value
			case "--count":
				i++
				if i >= len(args) {
					die("kit replay effective-data", fmt.Errorf("--count needs an integer"))
				}
				value, err := strconv.Atoi(args[i])
				if err != nil || value < 0 {
					die("kit replay effective-data", fmt.Errorf("--count needs a non-negative integer"))
				}
				opts.Count = value
			case "--limit":
				i++
				if i >= len(args) {
					die("kit replay effective-data", fmt.Errorf("--limit needs an integer"))
				}
				value, err := strconv.Atoi(args[i])
				if err != nil || value < 0 {
					die("kit replay effective-data", fmt.Errorf("--limit needs a non-negative integer"))
				}
				opts.Limit = value
				limitSet = true
			case "--known-template":
				i++
				if i >= len(args) {
					die("kit replay effective-data", fmt.Errorf("--known-template needs a sha256"))
				}
				opts.KnownTemplate = args[i]
			default:
				die("kit replay effective-data", fmt.Errorf("unknown option %q", args[i]))
			}
		}
		if treeOut {
			opts.IncludeAll = true
			if !limitSet {
				opts.Limit = 0
			}
		}
		report, err := replay.BuildEffectiveData(args[1], opts)
		if err != nil {
			die("kit replay effective-data", err)
		}
		if treeOut {
			printReplayEffectiveTree(report)
		} else if textOut {
			printReplayEffectiveData(report)
		} else {
			printJSON(report)
		}
	case "effective-units":
		checkReplayArg(args[1])
		textOut := false
		opts := replay.EffectiveUnitStatsOptions{Limit: 80}
		for i := 2; i < len(args); i++ {
			switch args[i] {
			case "--text":
				textOut = true
			case "--dat":
				i++
				if i >= len(args) {
					die("kit replay effective-units", fmt.Errorf("--dat needs a path"))
				}
				opts.DatPath = args[i]
			case "--player":
				i++
				if i >= len(args) {
					die("kit replay effective-units", fmt.Errorf("--player needs an integer"))
				}
				value, err := strconv.Atoi(args[i])
				if err != nil || value < 0 {
					die("kit replay effective-units", fmt.Errorf("--player needs a non-negative integer"))
				}
				opts.Player = value
			case "--limit":
				i++
				if i >= len(args) {
					die("kit replay effective-units", fmt.Errorf("--limit needs an integer"))
				}
				value, err := strconv.Atoi(args[i])
				if err != nil || value < 0 {
					die("kit replay effective-units", fmt.Errorf("--limit needs a non-negative integer"))
				}
				opts.Limit = value
			case "--include-unchanged":
				opts.IncludeUnchanged = true
			case "--all-types":
				opts.IncludeAllTypes = true
			case "--include-duplicates":
				opts.IncludeDuplicateNames = true
			default:
				die("kit replay effective-units", fmt.Errorf("unknown option %q", args[i]))
			}
		}
		report, err := replay.BuildEffectiveUnitStats(args[1], opts)
		if err != nil {
			die("kit replay effective-units", err)
		}
		if textOut {
			printReplayEffectiveUnits(report)
		} else {
			printJSON(report)
		}
	case "datamod-check":
		checkReplayArg(args[1])
		textOut := false
		opts := replay.EffectiveDataModCheckOptions{TailOffset: replay.DefaultEffectiveAvailabilityOffset, Count: replay.DefaultEffectiveAvailabilityCount, Limit: 40}
		for i := 2; i < len(args); i++ {
			switch args[i] {
			case "--text":
				textOut = true
			case "--dat":
				i++
				if i >= len(args) {
					die("kit replay datamod-check", fmt.Errorf("--dat needs a path"))
				}
				opts.DatPath = args[i]
			case "--baseline-replay":
				i++
				if i >= len(args) {
					die("kit replay datamod-check", fmt.Errorf("--baseline-replay needs a replay path"))
				}
				opts.BaselineReplay = args[i]
			case "--known-template":
				i++
				if i >= len(args) {
					die("kit replay datamod-check", fmt.Errorf("--known-template needs a sha256"))
				}
				opts.KnownTemplate = args[i]
			case "--tail-offset":
				i++
				if i >= len(args) {
					die("kit replay datamod-check", fmt.Errorf("--tail-offset needs an integer"))
				}
				value, err := strconv.Atoi(args[i])
				if err != nil || value < 0 {
					die("kit replay datamod-check", fmt.Errorf("--tail-offset needs a non-negative integer"))
				}
				opts.TailOffset = value
			case "--count":
				i++
				if i >= len(args) {
					die("kit replay datamod-check", fmt.Errorf("--count needs an integer"))
				}
				value, err := strconv.Atoi(args[i])
				if err != nil || value < 0 {
					die("kit replay datamod-check", fmt.Errorf("--count needs a non-negative integer"))
				}
				opts.Count = value
			case "--limit":
				i++
				if i >= len(args) {
					die("kit replay datamod-check", fmt.Errorf("--limit needs an integer"))
				}
				value, err := strconv.Atoi(args[i])
				if err != nil || value < 0 {
					die("kit replay datamod-check", fmt.Errorf("--limit needs a non-negative integer"))
				}
				opts.Limit = value
			default:
				die("kit replay datamod-check", fmt.Errorf("unknown option %q", args[i]))
			}
		}
		report, err := replay.BuildEffectiveDataModCheck(args[1], opts)
		if err != nil {
			die("kit replay datamod-check", err)
		}
		if textOut {
			printReplayDataModCheck(report)
		} else {
			printJSON(report)
		}
	case "diff-state":
		if len(args) < 3 {
			die("kit replay diff-state", fmt.Errorf("usage: kit replay diff-state <before.aoe2record> <after.aoe2record> [--focus all|opaque] [--limit N] [--text]"))
		}
		checkReplayArg(args[1])
		checkReplayArg(args[2])
		textOut := false
		opts := replay.DiffStateOptions{}
		for i := 3; i < len(args); i++ {
			switch args[i] {
			case "--text":
				textOut = true
			case "--focus":
				i++
				if i >= len(args) {
					die("kit replay diff-state", fmt.Errorf("--focus needs all or opaque"))
				}
				if args[i] != "all" && args[i] != "opaque" {
					die("kit replay diff-state", fmt.Errorf("--focus needs all or opaque"))
				}
				opts.Focus = args[i]
			case "--limit":
				i++
				if i >= len(args) {
					die("kit replay diff-state", fmt.Errorf("--limit needs an integer"))
				}
				value, err := strconv.Atoi(args[i])
				if err != nil || value < 0 {
					die("kit replay diff-state", fmt.Errorf("--limit needs a non-negative integer"))
				}
				opts.Limit = value
			default:
				die("kit replay diff-state", fmt.Errorf("unknown option %q", args[i]))
			}
		}
		report, err := replay.BuildDiffState(args[1], args[2], opts)
		if err != nil {
			die("kit replay diff-state", err)
		}
		if textOut {
			printReplayDiffState(report)
		} else {
			printJSON(report)
		}
	case "scan-value":
		checkReplayArg(args[1])
		textOut := false
		opts := replay.ValueScanOptions{}
		for i := 2; i < len(args); i++ {
			switch args[i] {
			case "--text":
				textOut = true
			case "--json":
				textOut = false
			case "--value":
				i++
				if i >= len(args) {
					die("kit replay scan-value", fmt.Errorf("--value needs a number"))
				}
				opts.Values = append(opts.Values, args[i])
			case "--hex":
				i++
				if i >= len(args) {
					die("kit replay scan-value", fmt.Errorf("--hex needs bytes"))
				}
				opts.Hex = append(opts.Hex, args[i])
			case "--string":
				i++
				if i >= len(args) {
					die("kit replay scan-value", fmt.Errorf("--string needs text"))
				}
				opts.Text = append(opts.Text, args[i])
			case "--width":
				i++
				if i >= len(args) {
					die("kit replay scan-value", fmt.Errorf("--width needs 8, 16, 32, or 64"))
				}
				value, err := strconv.Atoi(args[i])
				if err != nil || (value != 8 && value != 16 && value != 32 && value != 64) {
					die("kit replay scan-value", fmt.Errorf("--width needs 8, 16, 32, or 64"))
				}
				opts.NumericWidths = append(opts.NumericWidths, value)
			case "--space":
				i++
				if i >= len(args) {
					die("kit replay scan-value", fmt.Errorf("--space needs inflated_header or body"))
				}
				if args[i] != "inflated_header" && args[i] != "body" {
					die("kit replay scan-value", fmt.Errorf("--space needs inflated_header or body"))
				}
				opts.Space = args[i]
			case "--limit":
				i++
				if i >= len(args) {
					die("kit replay scan-value", fmt.Errorf("--limit needs an integer"))
				}
				value, err := strconv.Atoi(args[i])
				if err != nil || value < 0 {
					die("kit replay scan-value", fmt.Errorf("--limit needs a non-negative integer"))
				}
				opts.Limit = value
			default:
				die("kit replay scan-value", fmt.Errorf("unknown option %q", args[i]))
			}
		}
		if len(opts.Values) == 0 && len(opts.Hex) == 0 && len(opts.Text) == 0 {
			die("kit replay scan-value", fmt.Errorf("provide at least one --value, --hex, or --string"))
		}
		report, err := replay.BuildValueScan(args[1], opts)
		if err != nil {
			die("kit replay scan-value", err)
		}
		if textOut {
			printReplayValueScan(report)
		} else {
			printJSON(report)
		}
	case "header-anchors":
		checkReplayArg(args[1])
		textOut := false
		opts := replay.HeaderAnchorsOptions{Limit: 40}
		for i := 2; i < len(args); i++ {
			switch args[i] {
			case "--text":
				textOut = true
			case "--json":
				textOut = false
			case "--limit":
				i++
				if i >= len(args) {
					die("kit replay header-anchors", fmt.Errorf("--limit needs an integer"))
				}
				value, err := strconv.Atoi(args[i])
				if err != nil || value < 0 {
					die("kit replay header-anchors", fmt.Errorf("--limit needs a non-negative integer"))
				}
				opts.Limit = value
			default:
				die("kit replay header-anchors", fmt.Errorf("unknown option %q", args[i]))
			}
		}
		report, err := replay.BuildHeaderAnchors(args[1], opts)
		if err != nil {
			die("kit replay header-anchors", err)
		}
		if textOut {
			printReplayHeaderAnchors(report)
		} else {
			printJSON(report)
		}
	case "inbox":
		contextPath := ""
		textOut := false
		for i := 2; i < len(args); i++ {
			switch args[i] {
			case "--context":
				i++
				if i >= len(args) {
					die("kit replay inbox", fmt.Errorf("--context needs a JSON path"))
				}
				contextPath = args[i]
			case "--text":
				textOut = true
			default:
				die("kit replay inbox", fmt.Errorf("unknown option %q", args[i]))
			}
		}
		report, err := replay.BuildInbox(args[1], replay.InboxOptions{ContextPath: contextPath})
		if err != nil {
			die("kit replay inbox", err)
		}
		if textOut {
			printReplayInbox(report)
		} else {
			printJSON(report)
		}
	case "objects":
		checkReplayArg(args[1])
		opts := replay.ObjectIndexOptions{}
		textOut := false
		for i := 2; i < len(args); i++ {
			switch args[i] {
			case "--text":
				textOut = true
			case "--referenced":
				opts.ReferencedOnly = true
			case "--limit":
				i++
				if i >= len(args) {
					die("kit replay objects", fmt.Errorf("--limit needs an integer"))
				}
				value, err := strconv.Atoi(args[i])
				if err != nil {
					die("kit replay objects", err)
				}
				opts.Limit = value
			default:
				die("kit replay objects", fmt.Errorf("unknown option %q", args[i]))
			}
		}
		report, err := replay.BuildObjectIndex(args[1], opts)
		if err != nil {
			die("kit replay objects", err)
		}
		if textOut {
			printReplayObjects(report)
		} else {
			printJSON(report)
		}
	case "object-state":
		checkReplayArg(args[1])
		opts := replay.ObjectStateOptions{}
		textOut := false
		for i := 2; i < len(args); i++ {
			switch args[i] {
			case "--text":
				textOut = true
			case "--referenced":
				opts.ReferencedOnly = true
			case "--class":
				i++
				if i >= len(args) {
					die("kit replay object-state", fmt.Errorf("--class needs a value"))
				}
				opts.Class = args[i]
			case "--limit":
				i++
				if i >= len(args) {
					die("kit replay object-state", fmt.Errorf("--limit needs an integer"))
				}
				value, err := strconv.Atoi(args[i])
				if err != nil {
					die("kit replay object-state", err)
				}
				opts.Limit = value
			default:
				die("kit replay object-state", fmt.Errorf("unknown option %q", args[i]))
			}
		}
		report, err := replay.BuildObjectState(args[1], opts)
		if err != nil {
			die("kit replay object-state", err)
		}
		if textOut {
			printReplayObjectState(report)
		} else {
			printJSON(report)
		}
	case "object-shapes":
		checkReplayArg(args[1])
		opts := replay.ObjectIndexOptions{}
		textOut := false
		for i := 2; i < len(args); i++ {
			switch args[i] {
			case "--text":
				textOut = true
			case "--referenced":
				opts.ReferencedOnly = true
			case "--limit":
				i++
				if i >= len(args) {
					die("kit replay object-shapes", fmt.Errorf("--limit needs an integer"))
				}
				value, err := strconv.Atoi(args[i])
				if err != nil {
					die("kit replay object-shapes", err)
				}
				opts.Limit = value
			default:
				die("kit replay object-shapes", fmt.Errorf("unknown option %q", args[i]))
			}
		}
		report, err := replay.BuildObjectShapes(args[1], opts)
		if err != nil {
			die("kit replay object-shapes", err)
		}
		if textOut {
			printReplayObjectShapes(report)
		} else {
			printJSON(report)
		}
	case "spawns":
		checkReplayArg(args[1])
		textOut := false
		for _, arg := range args[2:] {
			switch arg {
			case "--text":
				textOut = true
			default:
				die("kit replay spawns", fmt.Errorf("unknown option %q", arg))
			}
		}
		report, err := replay.BuildSpawnCatalog(args[1])
		if err != nil {
			die("kit replay spawns", err)
		}
		if textOut {
			printReplaySpawns(report)
		} else {
			printJSON(report)
		}
	case "lifecycle":
		checkReplayArg(args[1])
		textOut := false
		for _, arg := range args[2:] {
			switch arg {
			case "--text":
				textOut = true
			default:
				die("kit replay lifecycle", fmt.Errorf("unknown option %q", arg))
			}
		}
		report, err := replay.BuildLifecycle(args[1], replay.LifecycleOptions{})
		if err != nil {
			die("kit replay lifecycle", err)
		}
		if textOut {
			printReplayLifecycle(report)
		} else {
			printJSON(report)
		}
	case "postgame":
		checkReplayArg(args[1])
		textOut := false
		for _, arg := range args[2:] {
			switch arg {
			case "--text":
				textOut = true
			case "--json":
				textOut = false
			default:
				die("kit replay postgame", fmt.Errorf("unknown option %q", arg))
			}
		}
		report, err := replay.BuildPostgame(args[1])
		if err != nil {
			die("kit replay postgame", err)
		}
		if textOut {
			printReplayPostgame(report)
		} else {
			printJSON(report)
		}
	case "opaque-target":
		checkReplayArg(args[1])
		textOnly := false
		for _, arg := range args[2:] {
			switch arg {
			case "--text":
				textOnly = true
			default:
				die("kit replay opaque-target", fmt.Errorf("unknown option %q", arg))
			}
		}
		report, err := replay.BuildOpaqueTarget(args[1])
		if err != nil {
			die("kit replay opaque-target", err)
		}
		if textOnly {
			fmt.Printf("path: %s\n", report.Path)
			fmt.Printf("method: %s\n", report.Method)
			fmt.Printf("verification: %s\n", report.Verification)
			fmt.Printf("header_opaque_bytes: %d\n", report.HeaderOpaqueBytes)
			for _, tail := range report.PlayerTails {
				fmt.Printf("- %s bytes=%d prefix_match=%d ratio=%.4f ref=%t\n", tail.Label, tail.Bytes, tail.PrefixMatchVsRef, tail.IdenticalRatio, tail.Reference)
			}
			fmt.Printf("common_prefix_bytes: %d\n", report.CommonPrefixBytes)
			fmt.Printf("reducible_estimate: %d\n", report.ReducibleEstimate)
			fmt.Printf("selected_target: %s\n", report.SelectedTarget)
			for _, r := range report.Rationale {
				fmt.Printf("rationale: %s\n", r)
			}
			fmt.Printf("next_increment: %s\n", report.NextIncrement)
		} else {
			printJSON(report)
		}
	case "postgame-corpus":
		textOut := false
		for _, arg := range args[2:] {
			switch arg {
			case "--text":
				textOut = true
			default:
				die("kit replay postgame-corpus", fmt.Errorf("unknown option %q", arg))
			}
		}
		report, err := replay.BuildPostgameCorpus(args[1])
		if err != nil {
			die("kit replay postgame-corpus", err)
		}
		if textOut {
			fmt.Printf("folder: %s\n", report.Folder)
			fmt.Printf("method: %s\n", report.Method)
			fmt.Printf("verification: %s\n", report.Verification)
			fmt.Printf("summary: scanned=%d failures=%d op6=%d metadata_only=%d unparsed_tails=%d action255=%d candidates=%d\n",
				report.Scanned, report.Failures, report.Summary.Op6Present, report.Summary.Op6MetadataOnly,
				report.Summary.Op6UnparsedTails, report.Summary.Action255Carriers, report.Summary.Candidates)
			for _, row := range report.Rows {
				fmt.Printf("- %s op6=%t tail=%dB blocks=%d action255=%d candidate=%t confidence=%s%s\n",
					row.Path, row.Op6Seen, row.Op6TailBytes, row.Op6Blocks, row.Action255Payloads, row.Candidate, row.Confidence,
					map[bool]string{true: " error=" + row.Error, false: ""}[row.Error != ""])
			}
		} else {
			printJSON(report)
		}
	case "health":
		checkReplayArg(args[1])
		opts := replay.HealthOptions{}
		textOut, outPath := false, ""
		for i := 2; i < len(args); i++ {
			switch args[i] {
			case "--text":
				textOut = true
			case "--json":
				textOut = false
			case "--window", "--out":
				flag := args[i]
				i++
				if i >= len(args) {
					die("kit replay health", fmt.Errorf("%s needs a value", flag))
				}
				if flag == "--out" {
					outPath = args[i]
				} else {
					d, err := time.ParseDuration(args[i])
					if err != nil || d < time.Second {
						die("kit replay health", fmt.Errorf("--window needs a duration of at least 1s"))
					}
					opts.Window = d
				}
			default:
				die("kit replay health", fmt.Errorf("unknown option %q", args[i]))
			}
		}
		report, err := replay.BuildHealth(args[1], opts)
		if err != nil {
			die("kit replay health", err)
		}
		out := os.Stdout
		if outPath != "" {
			out, err = os.Create(outPath)
			if err != nil {
				die("kit replay health", err)
			}
			defer out.Close()
		}
		if textOut {
			replay.WriteHealthText(out, report)
		} else {
			if err := jsonout.Encode(out, report); err != nil {
				die("kit replay health", err)
			}
		}
	case "sync":
		checkReplayArg(args[1])
		opts := replay.SyncOptions{}
		textOut := false
		for i := 2; i < len(args); i++ {
			switch args[i] {
			case "--text":
				textOut = true
			case "--json":
				textOut = false
			case "--checksums":
				opts.ChecksumsOnly = true
			case "--raw-words":
				opts.RawWords = true
				opts.ChecksumsOnly = true
			case "--limit":
				i++
				if i >= len(args) {
					die("kit replay sync", fmt.Errorf("--limit needs an integer"))
				}
				n, err := strconv.Atoi(args[i])
				if err != nil || n < 0 {
					die("kit replay sync", fmt.Errorf("--limit needs a non-negative integer"))
				}
				opts.Limit = n
			default:
				die("kit replay sync", fmt.Errorf("unknown option %q", args[i]))
			}
		}
		if textOut && opts.Limit == 0 {
			opts.Limit = 10
		}
		if !textOut {
			if err := replay.WriteSyncJSON(os.Stdout, args[1], opts); err != nil {
				die("kit replay sync", err)
			}
			return
		}
		report, err := replay.BuildSyncStream(args[1], opts)
		if err != nil {
			die("kit replay sync", err)
		}
		if textOut {
			printReplaySync(report)
		} else {
			printJSON(report)
		}
	case "sync-log":
		opts := replay.SyncLogOptions{Limit: 12}
		textOut := false
		for i := 2; i < len(args); i++ {
			switch args[i] {
			case "--text":
				textOut = true
			case "--json":
				textOut = false
			case "--replay":
				i++
				if i >= len(args) {
					die("kit replay sync-log", fmt.Errorf("--replay needs a path"))
				}
				opts.ReplayPath = args[i]
			case "--limit":
				i++
				if i >= len(args) {
					die("kit replay sync-log", fmt.Errorf("--limit needs an integer"))
				}
				n, err := strconv.Atoi(args[i])
				if err != nil || n < 0 {
					die("kit replay sync-log", fmt.Errorf("--limit needs a non-negative integer"))
				}
				opts.Limit = n
			default:
				die("kit replay sync-log", fmt.Errorf("unknown option %q", args[i]))
			}
		}
		info, inspectErr := aoe2.InspectFile(args[1])
		if inspectErr == nil && (info.Kind == "record" || info.Kind == "zip") {
			if opts.ReplayPath != "" {
				die("kit replay sync-log", fmt.Errorf("--replay is only for p0-sync text logs; when the first argument is a replay, omit --replay"))
			}
			syncOpts := replay.SyncOptions{Limit: opts.Limit}
			report, err := replay.BuildSyncStream(args[1], syncOpts)
			if err != nil {
				die("kit replay sync-log", err)
			}
			if textOut {
				printReplaySync(report)
			} else {
				printJSON(report)
			}
			return
		}
		report, err := replay.BuildSyncLogReport(args[1], opts)
		if err != nil {
			die("kit replay sync-log", err)
		}
		if textOut {
			printReplaySyncLog(report)
		} else {
			printJSON(report)
		}
	case "checksum-phase":
		checkReplayArg(args[1])
		opts := replay.ChecksumPhaseOptions{WordIndex: 9}
		textOut := false
		allWords := false
		for i := 2; i < len(args); i++ {
			switch args[i] {
			case "--text":
				textOut = true
			case "--json":
				textOut = false
			case "--all-words":
				allWords = true
			case "--word":
				i++
				if i >= len(args) {
					die("kit replay checksum-phase", fmt.Errorf("--word needs an integer"))
				}
				n, err := strconv.Atoi(args[i])
				if err != nil || n < 0 || n > 10 {
					die("kit replay checksum-phase", fmt.Errorf("--word needs an integer 0..10"))
				}
				opts.WordIndex = n
			case "--player":
				i++
				if i >= len(args) {
					die("kit replay checksum-phase", fmt.Errorf("--player needs an integer"))
				}
				n, err := strconv.Atoi(args[i])
				if err != nil || n < 1 || n > 8 {
					die("kit replay checksum-phase", fmt.Errorf("--player needs an integer 1..8"))
				}
				opts.PlayerID = n
			default:
				die("kit replay checksum-phase", fmt.Errorf("unknown option %q", args[i]))
			}
		}
		if allWords {
			report, err := replay.BuildChecksumPhaseAll(args[1], opts)
			if err != nil {
				die("kit replay checksum-phase", err)
			}
			if textOut {
				printReplayChecksumPhaseAll(report)
			} else {
				printJSON(report)
			}
			return
		}
		report, err := replay.BuildChecksumPhase(args[1], opts)
		if err != nil {
			die("kit replay checksum-phase", err)
		}
		if textOut {
			printReplayChecksumPhase(report)
		} else {
			printJSON(report)
		}
	case "checksum-probe":
		checkReplayArg(args[1])
		opts := replay.ChecksumProbeOptions{}
		textOut := false
		for i := 2; i < len(args); i++ {
			switch args[i] {
			case "--text":
				textOut = true
			case "--json":
				textOut = false
			case "--preset":
				i++
				if i >= len(args) {
					die("kit replay checksum-probe", fmt.Errorf("--preset needs a name"))
				}
				opts.Preset = args[i]
			case "--player":
				i++
				if i >= len(args) {
					die("kit replay checksum-probe", fmt.Errorf("--player needs an integer"))
				}
				n, err := strconv.Atoi(args[i])
				if err != nil || n <= 0 || n > 8 {
					die("kit replay checksum-probe", fmt.Errorf("--player needs an integer 1..8"))
				}
				opts.PlayerID = n
			case "--object-count-delta":
				i++
				if i >= len(args) {
					die("kit replay checksum-probe", fmt.Errorf("--object-count-delta needs an integer"))
				}
				value, err := strconv.ParseInt(args[i], 10, 64)
				if err != nil {
					die("kit replay checksum-probe", fmt.Errorf("--object-count-delta needs an integer"))
				}
				opts.ObjectCountDelta = &value
			case "--unit-type-sum-delta":
				i++
				if i >= len(args) {
					die("kit replay checksum-probe", fmt.Errorf("--unit-type-sum-delta needs an integer"))
				}
				value, err := strconv.ParseInt(args[i], 10, 64)
				if err != nil {
					die("kit replay checksum-probe", fmt.Errorf("--unit-type-sum-delta needs an integer"))
				}
				opts.UnitTypeSumDelta = &value
			case "--object-id-sum-delta":
				i++
				if i >= len(args) {
					die("kit replay checksum-probe", fmt.Errorf("--object-id-sum-delta needs an integer"))
				}
				value, err := strconv.ParseInt(args[i], 10, 64)
				if err != nil {
					die("kit replay checksum-probe", fmt.Errorf("--object-id-sum-delta needs an integer"))
				}
				opts.ObjectIDSumDelta = &value
			case "--word3-delta", "--state-sum-delta":
				i++
				if i >= len(args) {
					die("kit replay checksum-probe", fmt.Errorf("%s needs an integer", args[i-1]))
				}
				value, err := strconv.ParseInt(args[i], 10, 64)
				if err != nil {
					die("kit replay checksum-probe", fmt.Errorf("%s needs an integer", args[i-1]))
				}
				opts.Word3Delta = &value
			case "--word4-delta", "--carry-sum-delta":
				i++
				if i >= len(args) {
					die("kit replay checksum-probe", fmt.Errorf("%s needs an integer", args[i-1]))
				}
				value, err := strconv.ParseInt(args[i], 10, 64)
				if err != nil {
					die("kit replay checksum-probe", fmt.Errorf("%s needs an integer", args[i-1]))
				}
				opts.Word4Delta = &value
			case "--score-candidate-delta", "--digest-delta":
				i++
				if i >= len(args) {
					die("kit replay checksum-probe", fmt.Errorf("%s needs an integer", args[i-1]))
				}
				value, err := strconv.ParseInt(args[i], 10, 64)
				if err != nil {
					die("kit replay checksum-probe", fmt.Errorf("%s needs an integer", args[i-1]))
				}
				opts.ScoreDelta = &value
			case "--limit":
				i++
				if i >= len(args) {
					die("kit replay checksum-probe", fmt.Errorf("--limit needs an integer"))
				}
				n, err := strconv.Atoi(args[i])
				if err != nil || n < 0 {
					die("kit replay checksum-probe", fmt.Errorf("--limit needs a non-negative integer"))
				}
				opts.Limit = n
			default:
				die("kit replay checksum-probe", fmt.Errorf("unknown option %q", args[i]))
			}
		}
		if textOut && opts.Limit == 0 {
			opts.Limit = 12
		}
		report, err := replay.BuildChecksumProbe(args[1], opts)
		if err != nil {
			die("kit replay checksum-probe", err)
		}
		if textOut {
			printReplayChecksumProbe(report)
		} else {
			printJSON(report)
		}
	case "combat":
		checkReplayArg(args[1])
		opts := replay.CombatOptions{WindowMS: 5000, Limit: 40, Sort: "loss"}
		textOut := false
		for i := 2; i < len(args); i++ {
			switch args[i] {
			case "--text":
				textOut = true
			case "--json":
				textOut = false
			case "--limit":
				i++
				if i >= len(args) {
					die("kit replay combat", fmt.Errorf("--limit needs an integer"))
				}
				n, err := strconv.Atoi(args[i])
				if err != nil || n < 0 {
					die("kit replay combat", fmt.Errorf("--limit needs a non-negative integer"))
				}
				opts.Limit = n
			case "--window-ms":
				i++
				if i >= len(args) {
					die("kit replay combat", fmt.Errorf("--window-ms needs an integer"))
				}
				n, err := strconv.Atoi(args[i])
				if err != nil || n < 0 {
					die("kit replay combat", fmt.Errorf("--window-ms needs a non-negative integer"))
				}
				opts.WindowMS = n
			case "--min-net-loss":
				i++
				if i >= len(args) {
					die("kit replay combat", fmt.Errorf("--min-net-loss needs an integer"))
				}
				n, err := strconv.Atoi(args[i])
				if err != nil || n <= 0 {
					die("kit replay combat", fmt.Errorf("--min-net-loss needs a positive integer"))
				}
				opts.MinNetLoss = n
			case "--sort":
				i++
				if i >= len(args) {
					die("kit replay combat", fmt.Errorf("--sort needs loss or time"))
				}
				if args[i] != "loss" && args[i] != "time" {
					die("kit replay combat", fmt.Errorf("--sort must be loss or time"))
				}
				opts.Sort = args[i]
			default:
				die("kit replay combat", fmt.Errorf("unknown option %q", args[i]))
			}
		}
		report, err := replay.BuildCombatStory(args[1], opts)
		if err != nil {
			die("kit replay combat", err)
		}
		if textOut {
			printReplayCombat(report)
		} else {
			printJSON(report)
		}
	case "deaths":
		checkReplayArg(args[1])
		opts := replay.DeathReportOptions{Limit: 40}
		textOut := false
		for i := 2; i < len(args); i++ {
			switch args[i] {
			case "--text":
				textOut = true
			case "--json":
				textOut = false
			case "--limit":
				i++
				if i >= len(args) {
					die("kit replay deaths", fmt.Errorf("--limit needs an integer"))
				}
				n, err := strconv.Atoi(args[i])
				if err != nil || n < 0 {
					die("kit replay deaths", fmt.Errorf("--limit needs a non-negative integer"))
				}
				opts.Limit = n
			default:
				die("kit replay deaths", fmt.Errorf("unknown option %q", args[i]))
			}
		}
		report, err := replay.BuildDeathReport(args[1], opts)
		if err != nil {
			die("kit replay deaths", err)
		}
		if textOut {
			printReplayDeaths(report)
		} else {
			printJSON(report)
		}
	case "player-series":
		checkReplayArg(args[1])
		opts := replay.PlayerSeriesOptions{}
		textOut := false
		for i := 2; i < len(args); i++ {
			switch args[i] {
			case "--text":
				textOut = true
			case "--json":
				textOut = false
			case "--changes-only":
				opts.ChangesOnly = true
			case "--player":
				i++
				if i >= len(args) {
					die("kit replay player-series", fmt.Errorf("--player needs an integer"))
				}
				n, err := strconv.Atoi(args[i])
				if err != nil || n <= 0 || n > 8 {
					die("kit replay player-series", fmt.Errorf("--player needs an integer 1..8"))
				}
				opts.PlayerID = n
			case "--limit":
				i++
				if i >= len(args) {
					die("kit replay player-series", fmt.Errorf("--limit needs an integer"))
				}
				n, err := strconv.Atoi(args[i])
				if err != nil || n < 0 {
					die("kit replay player-series", fmt.Errorf("--limit needs a non-negative integer"))
				}
				opts.Limit = n
			default:
				die("kit replay player-series", fmt.Errorf("unknown option %q", args[i]))
			}
		}
		if textOut && opts.Limit == 0 {
			opts.Limit = 12
		}
		report, err := replay.BuildPlayerSeries(args[1], opts)
		if err != nil {
			die("kit replay player-series", err)
		}
		if textOut {
			printReplayPlayerSeries(report)
		} else {
			printJSON(report)
		}
	case "playtest":
		checkReplayArg(args[1])
		opts := replay.PlaytestOptions{Window: 2}
		textOut := false
		for i := 2; i < len(args); i++ {
			switch args[i] {
			case "--text":
				textOut = true
			case "--json":
				textOut = false
			case "--include-computers":
				opts.IncludeComputers = true
			case "--window":
				i++
				if i >= len(args) {
					die("kit replay playtest", fmt.Errorf("--window needs an integer"))
				}
				n, err := strconv.Atoi(args[i])
				if err != nil || n < 0 {
					die("kit replay playtest", fmt.Errorf("--window needs a non-negative integer"))
				}
				opts.Window = n
			default:
				die("kit replay playtest", fmt.Errorf("unknown option %q", args[i]))
			}
		}
		report, err := replay.BuildPlaytestReport(args[1], opts)
		if err != nil {
			die("kit replay playtest", err)
		}
		if textOut {
			printReplayPlaytest(report)
		} else {
			printJSON(report)
		}
	case "carrier":
		checkReplayArg(args[1])
		opts := replay.CarrierOptions{}
		textOut := false
		for i := 2; i < len(args); i++ {
			switch args[i] {
			case "--text":
				textOut = true
			case "--json":
				textOut = false
			case "--ledger":
				i++
				if i >= len(args) {
					die("kit replay carrier", fmt.Errorf("--ledger needs a path"))
				}
				opts.LedgerPath = args[i]
			default:
				die("kit replay carrier", fmt.Errorf("unknown option %q", args[i]))
			}
		}
		report, err := replay.BuildCarrierReport(args[1], opts)
		if err != nil {
			die("kit replay carrier", err)
		}
		if textOut {
			printReplayCarrier(report)
		} else {
			printJSON(report)
		}
	case "sidecar-sync":
		checkReplayArg(args[1])
		opts := replay.SidecarSyncOptions{}
		textOut := false
		failOnMismatch := false
		for i := 2; i < len(args); i++ {
			switch args[i] {
			case "--text":
				textOut = true
			case "--json":
				textOut = false
			case "--fail-on-mismatch":
				failOnMismatch = true
			case "--xsdat":
				i++
				if i >= len(args) {
					die("kit replay sidecar-sync", fmt.Errorf("--xsdat needs a path"))
				}
				opts.SidecarPath = args[i]
			case "--ledger":
				i++
				if i >= len(args) {
					die("kit replay sidecar-sync", fmt.Errorf("--ledger needs a path"))
				}
				opts.LedgerPath = args[i]
			case "--schema":
				i++
				if i >= len(args) {
					die("kit replay sidecar-sync", fmt.Errorf("--schema needs a schema name"))
				}
				opts.Schema = args[i]
			case "--player":
				i++
				if i >= len(args) {
					die("kit replay sidecar-sync", fmt.Errorf("--player needs an integer"))
				}
				n, err := strconv.Atoi(args[i])
				if err != nil {
					die("kit replay sidecar-sync", fmt.Errorf("--player needs an integer"))
				}
				opts.PlayerID = n
			case "--window-ms":
				i++
				if i >= len(args) {
					die("kit replay sidecar-sync", fmt.Errorf("--window-ms needs an integer"))
				}
				n, err := strconv.Atoi(args[i])
				if err != nil || n <= 0 {
					die("kit replay sidecar-sync", fmt.Errorf("--window-ms needs a positive integer"))
				}
				opts.WindowMS = n
			default:
				die("kit replay sidecar-sync", fmt.Errorf("unknown option %q", args[i]))
			}
		}
		report, err := replay.BuildSidecarSyncReport(args[1], opts)
		if err != nil {
			die("kit replay sidecar-sync", err)
		}
		if textOut {
			printReplaySidecarSync(report)
		} else {
			printJSON(report)
		}
		if failOnMismatch && (!report.Summary.SidecarDecodeOK || report.Summary.MissingSamples > 0 || report.Summary.KnownWordMismatches > 0) {
			os.Exit(1)
		}
	case "xs-telemetry":
		checkReplayArg(args[1])
		textOut := false
		opts := replay.XSTelemetryOptions{}
		for _, arg := range args[2:] {
			switch arg {
			case "--text":
				textOut = true
			default:
				if strings.HasPrefix(arg, "--food-base=") {
					n, err := strconv.Atoi(strings.TrimPrefix(arg, "--food-base="))
					if err != nil || n < 0 {
						die("kit replay xs-telemetry", fmt.Errorf("--food-base needs a non-negative integer"))
					}
					opts.FoodBase = n
					continue
				}
				die("kit replay xs-telemetry", fmt.Errorf("unknown option %q", arg))
			}
		}
		report, err := replay.BuildXSTelemetryProbeWithOptions(args[1], opts)
		if err != nil {
			die("kit replay xs-telemetry", err)
		}
		if textOut {
			printReplayXSTelemetry(report)
		} else {
			printJSON(report)
		}
	case "camera", "viewlock":
		checkReplayArg(args[1])
		opts := replay.CameraOptions{Tail: -1}
		textOut := false
		for i := 2; i < len(args); i++ {
			switch args[i] {
			case "--text":
				textOut = true
			case "--json":
				textOut = false
			case "--limit":
				i++
				if i >= len(args) {
					die("kit replay camera", fmt.Errorf("--limit needs an integer"))
				}
				n, err := strconv.Atoi(args[i])
				if err != nil || n < 0 {
					die("kit replay camera", fmt.Errorf("--limit needs a non-negative integer"))
				}
				opts.Limit = n
			case "--tail":
				i++
				if i >= len(args) {
					die("kit replay camera", fmt.Errorf("--tail needs an integer"))
				}
				n, err := strconv.Atoi(args[i])
				if err != nil || n < 0 {
					die("kit replay camera", fmt.Errorf("--tail needs a non-negative integer"))
				}
				opts.Tail = n
			default:
				die("kit replay camera", fmt.Errorf("unknown option %q", args[i]))
			}
		}
		if textOut && opts.Limit == 0 {
			opts.Limit = 20
		}
		report, err := replay.BuildCamera(args[1], opts)
		if err != nil {
			die("kit replay camera", err)
		}
		if textOut {
			printReplayCamera(report)
		} else {
			printJSON(report)
		}
	case "events":
		checkReplayArg(args[1])
		opts := replay.EventOptions{}
		contextPath := ""
		textOut := false
		for i := 2; i < len(args); i++ {
			switch args[i] {
			case "--all":
				opts.IncludeSystemEvents = true
				opts.IncludeUntypedAction = true
			case "--system":
				opts.IncludeSystemEvents = true
			case "--untyped-actions":
				opts.IncludeUntypedAction = true
			case "--raw":
				opts.IncludeRaw = true
			case "--objects":
				opts.IncludeObjectIndex = true
				opts.IncludeUntypedAction = true
			case "--context":
				i++
				if i >= len(args) {
					die("kit replay events", fmt.Errorf("--context needs a JSON path"))
				}
				contextPath = args[i]
			case "--text":
				textOut = true
			default:
				die("kit replay events", fmt.Errorf("unknown option %q", args[i]))
			}
		}
		if contextPath != "" {
			context, err := replay.LoadContext(contextPath)
			if err != nil {
				die("kit replay events", err)
			}
			opts.TelemetryPrefixes = context.TelemetryPrefixes
			report, err := replay.ExtractEvents(args[1], opts)
			if err != nil {
				die("kit replay events", err)
			}
			replay.AnnotateRegions(report.Events, context)
			if textOut {
				printReplayEvents(report)
			} else {
				printJSON(report)
			}
			return
		}
		report, err := replay.ExtractEvents(args[1], opts)
		if err != nil {
			die("kit replay events", err)
		}
		if textOut {
			printReplayEvents(report)
		} else {
			printJSON(report)
		}
	case "actions":
		checkReplayArg(args[1])
		actionID := -1
		limit := 50
		textOut := false
		includeRaw := false
		includeObjects := false
		unknownOnly := false
		sampleCount := 4
		for i := 2; i < len(args); i++ {
			switch args[i] {
			case "--unknown-only":
				unknownOnly = true
			case "--samples":
				i++
				if i >= len(args) {
					die("kit replay actions", fmt.Errorf("--samples needs an integer"))
				}
				value, err := strconv.Atoi(args[i])
				if err != nil || value < 0 {
					die("kit replay actions", fmt.Errorf("--samples needs a non-negative integer"))
				}
				sampleCount = value
			case "--action-id":
				i++
				if i >= len(args) {
					die("kit replay actions", fmt.Errorf("--action-id needs an integer"))
				}
				value, err := strconv.Atoi(args[i])
				if err != nil {
					die("kit replay actions", fmt.Errorf("--action-id needs an integer: %w", err))
				}
				actionID = value
			case "--limit":
				i++
				if i >= len(args) {
					die("kit replay actions", fmt.Errorf("--limit needs an integer"))
				}
				value, err := strconv.Atoi(args[i])
				if err != nil || value < 0 {
					die("kit replay actions", fmt.Errorf("--limit needs a non-negative integer"))
				}
				limit = value
			case "--raw":
				includeRaw = true
			case "--objects":
				includeObjects = true
			case "--text":
				textOut = true
			case "--json":
				textOut = false
			default:
				die("kit replay actions", fmt.Errorf("unknown option %q", args[i]))
			}
		}
		if unknownOnly {
			unknownReport, err := replay.BuildUnknownActions(args[1], sampleCount)
			if err != nil {
				die("kit replay actions", err)
			}
			if textOut {
				printReplayUnknownActions(unknownReport)
			} else {
				printJSON(unknownReport)
			}
			return
		}
		report, err := replay.ExtractEvents(args[1], replay.EventOptions{IncludeUntypedAction: true, IncludeRaw: includeRaw, IncludeObjectIndex: includeObjects})
		if err != nil {
			die("kit replay actions", err)
		}
		actions := filterReplayActions(report.Events, actionID, limit)
		var filter *int
		if actionID >= 0 {
			value := actionID
			filter = &value
		}
		output := struct {
			Path     string               `json:"path"`
			Filter   *int                 `json:"filter_action_id,omitempty"`
			Limit    int                  `json:"limit"`
			Counts   replay.EventCounts   `json:"counts"`
			Actions  []replay.ReplayEvent `json:"actions"`
			Warnings []string             `json:"warnings,omitempty"`
		}{
			Path:     report.Path,
			Filter:   filter,
			Limit:    limit,
			Counts:   report.Counts,
			Actions:  actions,
			Warnings: report.Warnings,
		}
		if textOut {
			printReplayActions(output.Path, actionID, actionID >= 0, output.Limit, output.Counts, output.Actions, output.Warnings)
		} else {
			printJSON(output)
		}
	case "player-events":
		checkReplayArg(args[1])
		opts := replay.PlayerEventsOptions{}
		textOut := false
		for i := 2; i < len(args); i++ {
			switch args[i] {
			case "--context":
				i++
				if i >= len(args) {
					die("kit replay player-events", fmt.Errorf("--context needs a JSON path"))
				}
				opts.ContextPath = args[i]
			case "--player":
				i++
				if i >= len(args) {
					die("kit replay player-events", fmt.Errorf("--player needs an integer"))
				}
				value, err := strconv.Atoi(args[i])
				if err != nil || value <= 0 {
					die("kit replay player-events", fmt.Errorf("--player needs a positive integer"))
				}
				opts.PlayerID = value
			case "--type":
				i++
				if i >= len(args) {
					die("kit replay player-events", fmt.Errorf("--type needs an event type"))
				}
				opts.Type = args[i]
			case "--action-id":
				i++
				if i >= len(args) {
					die("kit replay player-events", fmt.Errorf("--action-id needs an integer"))
				}
				value, err := strconv.Atoi(args[i])
				if err != nil || value < 0 {
					die("kit replay player-events", fmt.Errorf("--action-id needs a non-negative integer"))
				}
				opts.ActionID = value
				opts.ActionIDSet = true
			case "--phase":
				i++
				if i >= len(args) {
					die("kit replay player-events", fmt.Errorf("--phase needs a value"))
				}
				opts.Phase = args[i]
			case "--from":
				i++
				if i >= len(args) {
					die("kit replay player-events", fmt.Errorf("--from needs a time"))
				}
				value, err := replay.ParseClockMS(args[i])
				if err != nil {
					die("kit replay player-events", err)
				}
				opts.FromMS = value
			case "--to":
				i++
				if i >= len(args) {
					die("kit replay player-events", fmt.Errorf("--to needs a time"))
				}
				value, err := replay.ParseClockMS(args[i])
				if err != nil {
					die("kit replay player-events", err)
				}
				opts.ToMS = value
			case "--limit":
				i++
				if i >= len(args) {
					die("kit replay player-events", fmt.Errorf("--limit needs an integer"))
				}
				value, err := strconv.Atoi(args[i])
				if err != nil || value < 0 {
					die("kit replay player-events", fmt.Errorf("--limit needs a non-negative integer"))
				}
				opts.Limit = value
			case "--raw":
				opts.IncludeRaw = true
			case "--objects":
				opts.IncludeObjects = true
			case "--text":
				textOut = true
			default:
				die("kit replay player-events", fmt.Errorf("unknown option %q", args[i]))
			}
		}
		report, err := replay.BuildPlayerEvents(args[1], opts)
		if err != nil {
			die("kit replay player-events", err)
		}
		if textOut {
			printPlayerEvents(report)
		} else {
			printJSON(report)
		}
	case "chat":
		checkReplayArg(args[1])
		textOut := false
		for _, arg := range args[2:] {
			switch arg {
			case "--text":
				textOut = true
			case "--json":
				textOut = false
			default:
				die("kit replay chat", fmt.Errorf("unknown option %q", arg))
			}
		}
		report, err := replay.ExtractChat(args[1])
		if err != nil {
			die("kit replay chat", err)
		}
		if textOut {
			printReplayChat(report)
		} else {
			printJSON(report)
		}
	case "feedback":
		checkReplayArg(args[1])
		textOut := false
		for _, arg := range args[2:] {
			switch arg {
			case "--text":
				textOut = true
			default:
				die("kit replay feedback", fmt.Errorf("unknown option %q", arg))
			}
		}
		report, err := replay.ExtractFeedback(args[1])
		if err != nil {
			die("kit replay feedback", err)
		}
		if textOut {
			printFeedback(report)
		} else {
			printJSON(report)
		}
	case "story":
		checkReplayArg(args[1])
		opts := replay.StoryOptions{}
		textOut := false
		for i := 2; i < len(args); i++ {
			switch args[i] {
			case "--context":
				i++
				if i >= len(args) {
					die("kit replay story", fmt.Errorf("--context needs a JSON path"))
				}
				opts.ContextPath = args[i]
			case "--text", "--brief":
				textOut = true
			default:
				die("kit replay story", fmt.Errorf("unknown option %q", args[i]))
			}
		}
		report, err := replay.BuildStory(args[1], opts)
		if err != nil {
			die("kit replay story", err)
		}
		if textOut {
			printReplayStory(report)
		} else {
			printJSON(report)
		}
	case "player-profile":
		checkReplayArg(args[1])
		opts := replay.PlayerProfileOptions{}
		textOut := false
		for i := 2; i < len(args); i++ {
			switch args[i] {
			case "--context":
				i++
				if i >= len(args) {
					die("kit replay player-profile", fmt.Errorf("--context needs a JSON path"))
				}
				opts.ContextPath = args[i]
			case "--window-sec":
				i++
				if i >= len(args) {
					die("kit replay player-profile", fmt.Errorf("--window-sec needs an integer"))
				}
				value, err := strconv.Atoi(args[i])
				if err != nil || value <= 0 {
					die("kit replay player-profile", fmt.Errorf("--window-sec needs a positive integer"))
				}
				opts.WindowSec = value
			case "--dead-gap-sec":
				i++
				if i >= len(args) {
					die("kit replay player-profile", fmt.Errorf("--dead-gap-sec needs an integer"))
				}
				value, err := strconv.Atoi(args[i])
				if err != nil || value <= 0 {
					die("kit replay player-profile", fmt.Errorf("--dead-gap-sec needs a positive integer"))
				}
				opts.DeadGapSec = value
			case "--cell-size":
				i++
				if i >= len(args) {
					die("kit replay player-profile", fmt.Errorf("--cell-size needs a number"))
				}
				value, err := strconv.ParseFloat(args[i], 64)
				if err != nil || value <= 0 {
					die("kit replay player-profile", fmt.Errorf("--cell-size needs a positive number"))
				}
				opts.CellSize = value
			case "--include-events":
				opts.IncludeEvents = true
			case "--text":
				textOut = true
			case "--json":
				textOut = false
			default:
				die("kit replay player-profile", fmt.Errorf("unknown option %q", args[i]))
			}
		}
		report, err := replay.BuildPlayerProfile(args[1], opts)
		if err != nil {
			die("kit replay player-profile", err)
		}
		if textOut {
			printPlayerProfile(report)
		} else {
			printJSON(report)
		}
	case "telemetry":
		contextPath := ""
		schemaPath := ""
		textOut := false
		for i := 2; i < len(args); i++ {
			switch args[i] {
			case "--context":
				i++
				if i >= len(args) {
					die("kit replay telemetry", fmt.Errorf("--context needs a JSON path"))
				}
				contextPath = args[i]
			case "--schema":
				i++
				if i >= len(args) {
					die("kit replay telemetry", fmt.Errorf("--schema needs a JSON path"))
				}
				schemaPath = args[i]
			case "--text":
				textOut = true
			default:
				die("kit replay telemetry", fmt.Errorf("unknown option %q", args[i]))
			}
		}
		var output any
		if st, err := os.Stat(args[1]); err == nil && st.IsDir() {
			reports, err := replay.ValidateTelemetryFolder(args[1], contextPath, schemaPath)
			if err != nil {
				die("kit replay telemetry", err)
			}
			output = reports
			if textOut {
				printTelemetryReports(reports)
				return
			}
		} else {
			checkReplayArg(args[1])
			report, err := replay.ValidateTelemetry(args[1], contextPath, schemaPath)
			if err != nil {
				die("kit replay telemetry", err)
			}
			output = report
			if textOut {
				printTelemetryReports([]replay.TelemetryReport{*report})
				return
			}
			if !report.OK {
				defer os.Exit(1)
			}
		}
		printJSON(output)
	case "issues":
		checkReplayArg(args[1])
		contextPath := ""
		textOut := false
		for i := 2; i < len(args); i++ {
			switch args[i] {
			case "--context":
				i++
				if i >= len(args) {
					die("kit replay issues", fmt.Errorf("--context needs a JSON path"))
				}
				contextPath = args[i]
			case "--text":
				textOut = true
			default:
				die("kit replay issues", fmt.Errorf("unknown option %q", args[i]))
			}
		}
		report, err := replay.BuildIssues(args[1], replay.IssuesOptions{ContextPath: contextPath})
		if err != nil {
			die("kit replay issues", err)
		}
		if textOut {
			printReplayIssues(report)
		} else {
			printJSON(report)
		}
	case "unknowns":
		textOut := false
		for _, arg := range args[2:] {
			switch arg {
			case "--text":
				textOut = true
			default:
				die("kit replay unknowns", fmt.Errorf("unknown option %q", arg))
			}
		}
		report, err := replay.BuildUnknowns(args[1])
		if err != nil {
			die("kit replay unknowns", err)
		}
		if textOut {
			printReplayUnknowns(report)
		} else {
			printJSON(report)
		}
	default:
		fmt.Fprintln(os.Stderr, "usage: kit replay <info|summary|graph|triggers|trigger-neighborhood|diff-triggers|coverage|opaque-spans|frontier|corpus|opaque-clusters|ai-manifest|effective-data|effective-units|datamod-check|fetch|diff-state|scan-value|header-anchors|objects|object-state|object-shapes|spawns|lifecycle|postgame|camera|sync|sync-log|checksum-phase|checksum-probe|combat|deaths|player-series|playtest|carrier|sidecar-sync|xs-telemetry|events|actions|player-events|chat|feedback|story|player-profile|inbox|telemetry|issues|unknowns> <path> [options]")
		os.Exit(2)
	}
}

func checkReplayArg(path string) {
	info, err := aoe2.InspectFile(path)
	if err != nil {
		die("kit replay", err)
	}
	if info.Kind != "record" && info.Kind != "zip" {
		die("kit replay", fmt.Errorf("%s is %q, not record", path, info.Kind))
	}
}

func openReplayArg(path string) *replay.File {
	checkReplayArg(path)
	rec, err := replay.OpenMetadata(path)
	if err != nil {
		die("kit replay", err)
	}
	return rec
}

func parseReplayTriggerSearchArgs(args []string) (triggergraph.SearchOptions, bool, error) {
	opts := triggergraph.SearchOptions{}
	textOut := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--text":
			textOut = true
		case "--json":
			textOut = false
		case "--grep":
			i++
			if i >= len(args) {
				return opts, false, fmt.Errorf("--grep needs text")
			}
			opts.Grep = args[i]
		case "--effect":
			i++
			if i >= len(args) {
				return opts, false, fmt.Errorf("--effect needs an id or name")
			}
			opts.Effect = args[i]
		case "--condition":
			i++
			if i >= len(args) {
				return opts, false, fmt.Errorf("--condition needs an id or name")
			}
			opts.Condition = args[i]
		case "--player":
			value, err := parseNextInt(args, &i, "--player")
			if err != nil {
				return opts, false, err
			}
			opts.Player = &value
		case "--variable":
			value, err := parseNextInt(args, &i, "--variable")
			if err != nil {
				return opts, false, err
			}
			opts.Variable = &value
		case "--unit-const":
			value, err := parseNextInt(args, &i, "--unit-const")
			if err != nil {
				return opts, false, err
			}
			opts.UnitConst = &value
		case "--trigger-limit":
			value, err := parseNextInt(args, &i, "--trigger-limit")
			if err != nil {
				return opts, false, err
			}
			if value < 0 {
				return opts, false, fmt.Errorf("--trigger-limit must be non-negative")
			}
			opts.TriggerLimit = value
		case "--limit", "--match-limit":
			flag := args[i]
			value, err := parseNextInt(args, &i, flag)
			if err != nil {
				return opts, false, err
			}
			if value < 0 {
				return opts, false, fmt.Errorf("--limit must be non-negative")
			}
			opts.MatchLimit = value
		default:
			return opts, false, fmt.Errorf("unknown option %q", args[i])
		}
	}
	return opts, textOut, nil
}

func printReplayTriggerSearch(report *triggergraph.SearchReport) {
	fmt.Printf("verification=%s graph_sha256=%s triggers=%d effects=%d conditions=%d total_matches=%d shown=%d\n",
		report.Verification,
		shortHash(report.GraphSHA256),
		report.TriggerCount,
		report.EffectCount,
		report.ConditionCount,
		report.TotalMatches,
		len(report.Matches),
	)
	fmt.Printf("query: grep=%q effect=%q condition=%q player=%s variable=%s unit_const=%s trigger_limit=%d match_limit=%d\n",
		report.Query.Grep,
		report.Query.Effect,
		report.Query.Condition,
		intPtrText(report.Query.Player),
		intPtrText(report.Query.Variable),
		intPtrText(report.Query.UnitConst),
		report.Query.TriggerLimit,
		report.Query.MatchLimit,
	)
	for _, match := range report.Matches {
		child := ""
		if match.EffectIndex != nil {
			child = fmt.Sprintf(" effect=%d type=%d/%s", *match.EffectIndex, match.EffectType, match.EffectTypeName)
		}
		if match.ConditionIndex != nil {
			child = fmt.Sprintf(" condition=%d type=%d/%s", *match.ConditionIndex, match.ConditionType, match.ConditionName)
		}
		fmt.Printf("- trigger=%d%s enabled=%d looping=%d name=%q\n", match.TriggerIndex, child, match.Enabled, match.Looping, match.TriggerName)
		if match.Message != "" {
			fmt.Printf("  message: %s\n", oneLine(match.Message, 180))
		}
		if match.Description != "" {
			fmt.Printf("  description: %s\n", oneLine(match.Description, 180))
		}
		if match.ShortDescription != "" {
			fmt.Printf("  short_description: %s\n", oneLine(match.ShortDescription, 180))
		}
		if len(match.MatchedFields) > 0 {
			keys := make([]string, 0, len(match.MatchedFields))
			for key := range match.MatchedFields {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			for _, key := range keys {
				fmt.Printf("  %s: %s\n", key, oneLine(fmt.Sprint(match.MatchedFields[key]), 180))
			}
		}
	}
	if len(report.Warnings) > 0 {
		fmt.Println("warnings:")
		for _, warning := range report.Warnings {
			fmt.Printf("- %s\n", warning)
		}
	}
}

func parseReplayTriggerNeighborhoodArgs(args []string) (triggergraph.NeighborhoodOptions, bool, error) {
	opts := triggergraph.NeighborhoodOptions{}
	textOut := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--text":
			textOut = true
		case "--json":
			textOut = false
		case "--trigger":
			value, err := parseNextInt(args, &i, "--trigger")
			if err != nil {
				return opts, false, err
			}
			opts.TriggerIndex = &value
		case "--depth":
			value, err := parseNextInt(args, &i, "--depth")
			if err != nil {
				return opts, false, err
			}
			if value < 0 {
				return opts, false, fmt.Errorf("--depth must be non-negative")
			}
			opts.Depth = value
		case "--limit":
			value, err := parseNextInt(args, &i, "--limit")
			if err != nil {
				return opts, false, err
			}
			if value < 0 {
				return opts, false, fmt.Errorf("--limit must be non-negative")
			}
			opts.Limit = value
		default:
			return opts, false, fmt.Errorf("unknown option %q", args[i])
		}
	}
	return opts, textOut, nil
}

func printReplayTriggerNeighborhood(report *triggergraph.NeighborhoodReport) {
	fmt.Printf("verification=%s graph_sha256=%s triggers=%d nodes=%d edges=%d\n",
		report.Verification,
		shortHash(report.GraphSHA256),
		report.TriggerCount,
		len(report.Nodes),
		len(report.Edges),
	)
	fmt.Printf("query: trigger=%s depth=%d limit=%d\n", intPtrText(report.Query.TriggerIndex), report.Query.Depth, report.Query.Limit)
	fmt.Println("nodes:")
	for _, node := range report.Nodes {
		fmt.Printf("- #%d distance=%d enabled=%d looping=%d name=%q\n", node.Index, node.Distance, node.Enabled, node.Looping, node.Name)
		if len(node.ConditionTypes) > 0 {
			fmt.Printf("  conditions: %s\n", strings.Join(node.ConditionTypes, ", "))
		}
		if len(node.EffectTypes) > 0 {
			fmt.Printf("  effects: %s\n", strings.Join(node.EffectTypes, ", "))
		}
	}
	if len(report.Edges) > 0 {
		fmt.Println("edges:")
		for _, edge := range report.Edges {
			child := ""
			if edge.EffectIndex != nil {
				child = fmt.Sprintf(" effect=%d", *edge.EffectIndex)
			}
			if edge.ConditionIndex != nil {
				child = fmt.Sprintf(" condition=%d", *edge.ConditionIndex)
			}
			fmt.Printf("- %d -> %d kind=%s%s\n", edge.From, edge.To, edge.Kind, child)
		}
	}
	if len(report.Warnings) > 0 {
		fmt.Println("warnings:")
		for _, warning := range report.Warnings {
			fmt.Printf("- %s\n", warning)
		}
	}
}

func parseTriggerGraphDiffOptions(args []string, prefix string) (triggergraph.DiffOptions, bool, error) {
	opts := triggergraph.DiffOptions{}
	textOut := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--text":
			textOut = true
		case "--json":
			textOut = false
		case "--limit":
			i++
			if i >= len(args) {
				return opts, textOut, fmt.Errorf("--limit needs an integer")
			}
			value, err := strconv.Atoi(args[i])
			if err != nil || value < 0 {
				return opts, textOut, fmt.Errorf("--limit needs a non-negative integer")
			}
			opts.Limit = value
		default:
			return opts, textOut, fmt.Errorf("unknown option %q", args[i])
		}
	}
	return opts, textOut, nil
}

func buildReplayTriggerGraphDiff(beforePath, afterPath string, opts triggergraph.DiffOptions) (*triggergraph.DiffReport, error) {
	before := openReplayArg(beforePath)
	after := openReplayArg(afterPath)
	if before.TriggerGraph == nil {
		return nil, fmt.Errorf("before trigger graph unavailable: %s", before.TriggerGraphErr)
	}
	if after.TriggerGraph == nil {
		return nil, fmt.Errorf("after trigger graph unavailable: %s", after.TriggerGraphErr)
	}
	report := triggergraph.Diff(before.TriggerGraph, after.TriggerGraph, opts)
	report.Before = beforePath
	report.After = afterPath
	return report, nil
}

func buildScenarioTriggerGraphDiff(beforePath, afterPath string, opts triggergraph.DiffOptions) (*triggergraph.DiffReport, error) {
	before, err := scenario.Open(beforePath)
	if err != nil {
		return nil, fmt.Errorf("open before: %w", err)
	}
	after, err := scenario.Open(afterPath)
	if err != nil {
		return nil, fmt.Errorf("open after: %w", err)
	}
	beforeGraph, err := before.TriggerGraph()
	if err != nil {
		return nil, fmt.Errorf("before trigger graph: %w", err)
	}
	afterGraph, err := after.TriggerGraph()
	if err != nil {
		return nil, fmt.Errorf("after trigger graph: %w", err)
	}
	report := triggergraph.Diff(beforeGraph, afterGraph, opts)
	report.Before = beforePath
	report.After = afterPath
	return report, nil
}

func printReplayEvents(report *replay.EventReport) {
	fmt.Printf("path: %s\n", report.Path)
	fmt.Printf("method: %s\n", report.Method)
	fmt.Printf("data_set: %s\n", formatDataSetIdentity(report.DataSet))
	backlog := ""
	if report.Counts.BacklogChat > 0 {
		backlog = fmt.Sprintf(" backlog=%d", report.Counts.BacklogChat)
	}
	fmt.Printf("events: total=%d duration=%s chat=%d%s flares=%d telemetry=%d resigns=%d spatial=%d unknown_ops=%d untyped_actions=%d\n", report.Counts.Total, report.Counts.Duration, report.Counts.Chat, backlog, report.Counts.Flares, report.Counts.Telemetry, report.Counts.Resigns, report.Counts.SpatialActions, report.Counts.UnknownOps, report.Counts.UntypedActions)
	if report.Result.WinnerKnown {
		fmt.Printf("result: %s winners=%v losers=%v\n", report.Result.Method, report.Result.Winners, report.Result.Losers)
	}
	if len(report.Counts.EventTypes) > 0 {
		fmt.Printf("event_types: %s\n", strings.Join(replayEventTypeCounts(report.Counts.EventTypes), " "))
	}
	if len(report.Events) > 0 {
		fmt.Println("timeline:")
		for _, event := range report.Events {
			printReplayEventLine(event)
		}
	}
	if len(report.Warnings) > 0 {
		fmt.Println("warnings:")
		for _, warning := range report.Warnings {
			fmt.Printf("- %s\n", warning)
		}
	}
}

func filterReplayActions(events []replay.ReplayEvent, actionID int, limit int) []replay.ReplayEvent {
	var out []replay.ReplayEvent
	for _, event := range events {
		if !event.ReplayAction {
			continue
		}
		if actionID >= 0 && event.ActionID != actionID {
			continue
		}
		out = append(out, event)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out
}

func printReplayActions(path string, filter int, filtered bool, limit int, counts replay.EventCounts, actions []replay.ReplayEvent, warnings []string) {
	fmt.Printf("path: %s\n", path)
	if filtered {
		fmt.Printf("filter: action_id=%d\n", filter)
	}
	fmt.Printf("actions: total=%d shown=%d limit=%d spatial=%d resigns=%d untyped=%d saves=%d duration=%s\n", counts.Actions, len(actions), limit, counts.SpatialActions, counts.Resigns, counts.UntypedActions, counts.Saves, counts.Duration)
	if len(counts.ActionIDs) > 0 {
		fmt.Printf("action_ids: %s\n", strings.Join(replayActionIDCounts(counts.ActionIDs), " "))
	}
	if len(actions) > 0 {
		fmt.Println("timeline:")
		for _, event := range actions {
			printReplayEventLine(event)
		}
	}
	if len(warnings) > 0 {
		fmt.Println("warnings:")
		for _, warning := range warnings {
			fmt.Printf("- %s\n", warning)
		}
	}
}

func printPlayerEvents(report *replay.PlayerEventsReport) {
	fmt.Printf("path: %s\n", report.Path)
	if report.ContextName != "" {
		fmt.Printf("context: %s\n", report.ContextName)
	}
	fmt.Printf("method: %s\n", report.Method)
	fmt.Printf("verification: %s\n", report.Verification)
	fmt.Printf("events: total=%d shown=%d duration=%s actions=%d decoded_untyped=%d chat=%d flares=%d viewlocks=%d\n", report.Counts.Total, report.Shown, report.Counts.Duration, report.Counts.Actions, report.Counts.UntypedActions, report.Counts.Chat, report.Counts.Flares, report.Counts.Viewlocks)
	filterParts := playerEventFilterParts(report.Filters)
	if len(filterParts) > 0 {
		fmt.Printf("filters: %s\n", strings.Join(filterParts, " "))
	}
	if len(report.Events) > 0 {
		fmt.Println("timeline:")
		for _, event := range report.Events {
			printReplayEventLine(event)
		}
	}
	if len(report.Claims) > 0 {
		fmt.Println("claims:")
		for _, claim := range report.Claims {
			detail := ""
			if claim.Detail != "" {
				detail = " " + claim.Detail
			}
			fmt.Printf("- %s status=%s source=%s confidence=%s%s\n", claim.Name, claim.Status, claim.Source, claim.Confidence, detail)
		}
	}
	if len(report.Warnings) > 0 {
		fmt.Println("warnings:")
		for _, warning := range report.Warnings {
			fmt.Printf("- %s\n", warning)
		}
	}
}

func playerEventFilterParts(filters replay.PlayerEventsFilters) []string {
	var parts []string
	if filters.PlayerID > 0 {
		parts = append(parts, fmt.Sprintf("player=%d", filters.PlayerID))
	}
	if filters.Type != "" {
		parts = append(parts, "type="+filters.Type)
	}
	if filters.ActionID != nil {
		parts = append(parts, fmt.Sprintf("action_id=%d", *filters.ActionID))
	}
	if filters.Phase != "" {
		parts = append(parts, "phase="+filters.Phase)
	}
	if filters.FromMS > 0 {
		parts = append(parts, "from="+replay.FormatTime(filters.FromMS))
	}
	if filters.ToMS > 0 {
		parts = append(parts, "to="+replay.FormatTime(filters.ToMS))
	}
	if filters.Limit > 0 {
		parts = append(parts, fmt.Sprintf("limit=%d", filters.Limit))
	}
	return parts
}

func printReplayInbox(report *replay.InboxReport) {
	fmt.Printf("root: %s\n", report.Root)
	fmt.Printf("replays: %d parsed=%d errors=%d duplicate_items=%d scenario_groups=%d recognized=%d feedback=%d issue_cards=%d actions=%d untyped=%d avg_decoded=%.2f\n", report.Count, report.Summary.Parsed, report.Summary.Errors, report.Summary.DuplicateItems, report.Summary.ScenarioGroups, report.Summary.RecognizedScenarios, report.Summary.FeedbackEvents, report.Summary.IssueCards, report.Summary.ReplayActions, report.Summary.UntypedActions, report.Summary.AverageDecodedPercent)
	fmt.Printf("data_sets: vanilla=%d modded=%d unknown=%d results: completed=%d winner_known=%d\n", report.Summary.VanillaDataSets, report.Summary.ModdedDataSets, report.Summary.UnknownDataSets, report.Summary.Completed, report.Summary.WinnerKnown)
	if len(report.ScenarioGroups) > 0 {
		fmt.Println("scenario_groups:")
		for _, group := range report.ScenarioGroups {
			hash := group.SHA256
			if hash != "" {
				hash = shortHash(hash)
			} else {
				hash = "unknown"
			}
			fmt.Printf("- %s %s count=%d feedback=%d issues=%d data_sets=%s results=%s\n", group.Tier, hash, group.Count, group.FeedbackEvents, group.IssueCards, formatStringIntMap(group.DataSets), formatStringIntMap(group.Results))
		}
	}
	if len(report.IssueCards) > 0 {
		fmt.Println("top_issue_cards:")
		limit := len(report.IssueCards)
		if limit > 12 {
			limit = 12
		}
		for _, card := range report.IssueCards[:limit] {
			where := card.Chapter
			if card.Region != "" {
				where += "/" + card.Region
			}
			player := ""
			if card.PlayerID > 0 {
				player = fmt.Sprintf(" P%d", card.PlayerID)
				if card.PlayerName != "" {
					player += " " + card.PlayerName
				}
			}
			fmt.Printf("- %s count=%d%s %s %s first=%s last=%s confidence=%s\n", card.ID, card.Count, player, where, card.Text, card.FirstTime, card.LastTime, card.Confidence)
		}
	}
	if len(report.Duplicates) > 0 {
		fmt.Printf("duplicates: %d groups\n", len(report.Duplicates))
	}
	for _, item := range report.Items {
		status := "OK"
		if item.Error != "" {
			status = "ERROR"
		} else if item.DuplicateOf != "" {
			status = "DUPLICATE"
		}
		identity := item.Identity.Tier
		if item.Identity.SHA256 != "" {
			identity += " " + shortHash(item.Identity.SHA256)
		}
		result := "unknown"
		if item.Result.WinnerKnown {
			result = fmt.Sprintf("winners=%v losers=%v", item.Result.Winners, item.Result.Losers)
		} else if item.Result.Completed {
			result = "completed_result_unknown"
		}
		fmt.Printf("- %s [%s] %s data_set=%s duration=%s result=%s players=%d chat=%d flares=%d telemetry=%d actions=%d decoded=%.2f issues=%d\n", item.Path, status, identity, formatDataSetIdentity(item.DataSet), item.EventCounts.Duration, result, len(item.Players), item.EventCounts.Chat, item.EventCounts.Flares, item.EventCounts.Telemetry, item.EventCounts.Actions, item.ProfileCoverage.DecodedPercent, len(item.IssueCards))
		if item.Error != "" {
			fmt.Printf("  error: %s\n", item.Error)
		}
		if len(item.Moments) > 0 {
			for _, moment := range item.Moments {
				if moment.Kind == "first_resign" || moment.Kind == "peak_action_window" {
					fmt.Printf("  %s %s: %s\n", moment.Time, moment.Kind, moment.Detail)
				}
			}
		}
		for _, line := range item.Brief {
			fmt.Printf("  %s\n", line)
			break
		}
	}
	if len(report.Warnings) > 0 {
		fmt.Println("warnings:")
		for _, warning := range report.Warnings {
			fmt.Printf("- %s\n", warning)
		}
	}
}

func printReplayEventLine(event replay.ReplayEvent) {
	time := ""
	if event.Time != "" {
		time = event.Time + " "
	}
	player := ""
	if event.PlayerID > 0 {
		player = fmt.Sprintf("P%d", event.PlayerID)
		if event.PlayerName != "" {
			player += " " + event.PlayerName
		}
		player += " "
	}
	switch event.Type {
	case "flare":
		region := ""
		if event.Region != "" {
			region = " region=" + event.Region
		}
		fmt.Printf("- #%04d %s%sFLARE x=%.2f y=%.2f%s targets=%v\n", event.Index, time, player, event.X, event.Y, region, event.Targets)
	case "chat":
		suffix := ""
		if event.Source == "backlog" {
			suffix += " [backlog]"
		}
		if event.TauntNumber > 0 {
			suffix += fmt.Sprintf(" [taunt %d]", event.TauntNumber)
		}
		if event.Telemetry != nil {
			suffix += " [telemetry]"
		}
		fmt.Printf("- #%04d %s%s%s %q%s\n", event.Index, time, player, event.ChannelName, event.Text, suffix)
	case "action":
		raw := ""
		if event.RawHex != "" {
			raw = " raw=" + event.RawHex
		}
		name := ""
		if event.ActionName != "" {
			name = " name=" + event.ActionName
		}
		fmt.Printf("- #%04d %s%sACTION id=%d%s payload=%d offset=%d confidence=%s%s\n", event.Index, time, player, event.ActionID, name, event.PayloadBytes, event.SourceOffset, event.Confidence, raw)
	case "resign":
		fmt.Printf("- #%04d %s%sRESIGN sequence=%d\n", event.Index, time, player, event.Sequence)
	case "move", "order", "patrol", "attack_move", "attack_ground", "build", "gather_point", "de_multi_gatherpoint", "ungarrison", "special", "ai_order":
		target := ""
		if event.TargetID != 0 {
			target = fmt.Sprintf(" target=%d", event.TargetID)
		}
		fmt.Printf("- #%04d %s%s%s x=%.2f y=%.2f%s objects=%v%s\n", event.Index, time, player, strings.ToUpper(event.Type), event.X, event.Y, target, event.ObjectIDs, replayObjectSuffix(event))
	case "wall":
		fmt.Printf("- #%04d %s%sWALL x=%.2f y=%.2f x_end=%.2f y_end=%.2f building=%d objects=%v%s\n", event.Index, time, player, event.X, event.Y, event.XEnd, event.YEnd, event.TargetID, event.ObjectIDs, replayObjectSuffix(event))
	case "viewlock":
		region := ""
		if event.Region != "" {
			region = " region=" + event.Region
		}
		fmt.Printf("- #%04d %sVIEWLOCK x=%.2f y=%.2f%s confidence=%s\n", event.Index, time, event.X, event.Y, region, event.Confidence)
	case "delete", "gate":
		fmt.Printf("- #%04d %s%s%s objects=%v%s\n", event.Index, time, player, strings.ToUpper(event.Type), event.ObjectIDs, replayObjectSuffix(event))
	case "research":
		fmt.Printf("- #%04d %s%sRESEARCH tech=%d objects=%v%s\n", event.Index, time, player, event.TechnologyID, event.ObjectIDs, replayObjectSuffix(event))
	case "de_queue":
		fmt.Printf("- #%04d %s%sDE_QUEUE building=%d unit=%d amount=%d objects=%v%s\n", event.Index, time, player, event.BuildingID, event.UnitID, event.Amount, event.ObjectIDs, replayObjectSuffix(event))
	case "stop", "stance", "formation", "back_to_work", "town_bell", "make":
		detail := ""
		switch event.Type {
		case "stance":
			detail = fmt.Sprintf(" stance=%d", event.StanceID)
		case "formation":
			detail = fmt.Sprintf(" formation=%d", event.FormationID)
		case "town_bell":
			detail = fmt.Sprintf(" building=%d mode=%d", event.BuildingID, event.ModeID)
		case "make":
			detail = fmt.Sprintf(" building=%d unit=%d", event.BuildingID, event.UnitID)
		}
		fmt.Printf("- #%04d %s%s%s%s objects=%v%s\n", event.Index, time, player, strings.ToUpper(event.Type), detail, event.ObjectIDs, replayObjectSuffix(event))
	case "game_command":
		fmt.Printf("- #%04d %s%sGAME command=%d\n", event.Index, time, player, event.CommandID)
	case "unknown_operation":
		fmt.Printf("- #%04d %sUNKNOWN op=%d offset=%d confidence=%s\n", event.Index, time, event.OperationID, event.SourceOffset, event.Confidence)
	default:
		fmt.Printf("- #%04d %s%s%s\n", event.Index, time, player, event.Type)
	}
}

func replayObjectSuffix(event replay.ReplayEvent) string {
	var parts []string
	if event.TargetObject != nil {
		parts = append(parts, "target="+formatObjectReference(*event.TargetObject))
	}
	if event.BuildingObject != nil {
		parts = append(parts, "building="+formatObjectReference(*event.BuildingObject))
	}
	if len(event.ObjectRefs) > 0 {
		limit := len(event.ObjectRefs)
		if limit > 4 {
			limit = 4
		}
		refs := make([]string, 0, limit)
		for _, ref := range event.ObjectRefs[:limit] {
			refs = append(refs, formatObjectReference(ref))
		}
		if len(event.ObjectRefs) > limit {
			refs = append(refs, fmt.Sprintf("+%d more", len(event.ObjectRefs)-limit))
		}
		parts = append(parts, "refs=["+strings.Join(refs, "; ")+"]")
	}
	if len(parts) == 0 {
		return ""
	}
	return " " + strings.Join(parts, " ")
}

func formatObjectReference(ref replay.ObjectReference) string {
	return fmt.Sprintf("%d/%s/%s/u%d@%.2f,%.2f", ref.ObjectID, ref.OwnerLabel, ref.Class, ref.UnitID, ref.X, ref.Y)
}

func printReplayStory(report *replay.StoryReport) {
	fmt.Printf("path: %s\n", report.Path)
	if report.ContextName != "" {
		fmt.Printf("context: %s\n", report.ContextName)
	}
	if report.Identity.SHA256 != "" {
		fmt.Printf("identity: %s %s\n", report.Identity.Tier, shortHash(report.Identity.SHA256))
	}
	fmt.Printf("data_set: %s\n", formatDataSetIdentity(report.DataSet))
	if report.EventCounts.Duration != "" {
		fmt.Printf("duration: %s\n", report.EventCounts.Duration)
	}
	if report.Result.WinnerKnown {
		fmt.Printf("result: %s winners=%v losers=%v\n", report.Result.Method, report.Result.Winners, report.Result.Losers)
	}
	fmt.Printf("verification: %s\n", report.Verification)
	fmt.Println("brief:")
	for _, line := range report.Brief {
		fmt.Printf("- %s\n", line)
	}
	if len(report.PlayerSummary) > 0 {
		fmt.Println("players:")
		for _, player := range report.PlayerSummary {
			name := player.PlayerName
			if name == "" {
				name = "Unknown"
			}
			fmt.Printf("- P%d %s kind=%s chat=%d taunts=%d flares=%d telemetry=%d first=%s last=%s\n", player.PlayerID, name, player.Kind, player.Chat, player.Taunts, player.Flares, player.Telemetry, player.FirstTime, player.LastTime)
		}
	}
	if len(report.FeedbackClusters) > 0 {
		fmt.Println("feedback_clusters:")
		for _, cluster := range report.FeedbackClusters {
			label := cluster.Key
			if cluster.Region != "" {
				label = cluster.Region
			} else if cluster.PlayerID > 0 {
				name := cluster.PlayerName
				if name == "" {
					name = "Unknown"
				}
				label = fmt.Sprintf("P%d %s", cluster.PlayerID, name)
			}
			fmt.Printf("- %s count=%d first=%s last=%s\n", label, cluster.Count, cluster.FirstTime, cluster.LastTime)
		}
	}
	if len(report.Chapters) > 0 {
		fmt.Println("chapters:")
		for _, chapter := range report.Chapters {
			fmt.Printf("- %s %s-%s events=%d actions=%d chat=%d flares=%d telemetry=%d resigns=%d viewlocks=%d\n", chapter.Name, chapter.Start, chapter.End, chapter.Counts.Events, chapter.Counts.Actions, chapter.Counts.Chat, chapter.Counts.Flares, chapter.Counts.Telemetry, chapter.Counts.Resigns, chapter.Counts.Viewlocks)
			for _, line := range chapter.Highlights {
				fmt.Print("  ")
				printStoryLine(line)
			}
		}
	}
	if len(report.Moments) > 0 {
		fmt.Println("moments:")
		for _, moment := range report.Moments {
			player := ""
			if moment.PlayerID > 0 {
				player = fmt.Sprintf(" P%d", moment.PlayerID)
				if moment.PlayerName != "" {
					player += " " + moment.PlayerName
				}
			}
			fmt.Printf("- %s%s %s: %s confidence=%s\n", moment.Time, player, moment.Kind, moment.Detail, moment.Confidence)
		}
	}
	if len(report.Feedback) > 0 {
		fmt.Println("feedback:")
		for _, line := range report.Feedback {
			printStoryLine(line)
		}
	}
	if len(report.Telemetry) > 0 {
		fmt.Println("telemetry:")
		for _, line := range report.Telemetry {
			printStoryLine(line)
		}
	}
	if len(report.Claims) > 0 {
		fmt.Println("claims:")
		for _, claim := range report.Claims {
			detail := ""
			if claim.Detail != "" {
				detail = " " + claim.Detail
			}
			fmt.Printf("- %s status=%s source=%s confidence=%s%s\n", claim.Name, claim.Status, claim.Source, claim.Confidence, detail)
		}
	}
	if len(report.Warnings) > 0 {
		fmt.Println("warnings:")
		for _, warning := range report.Warnings {
			fmt.Printf("- %s\n", warning)
		}
	}
	if len(report.Missing) > 0 {
		fmt.Println("missing:")
		for _, missing := range report.Missing {
			fmt.Printf("- %s\n", missing)
		}
	}
}

func printPlayerProfile(report *replay.PlayerProfileReport) {
	fmt.Printf("path: %s\n", report.Path)
	if report.ContextName != "" {
		fmt.Printf("context: %s\n", report.ContextName)
	}
	if report.Identity.SHA256 != "" {
		fmt.Printf("identity: %s %s\n", report.Identity.Tier, shortHash(report.Identity.SHA256))
	}
	fmt.Printf("data_set: %s\n", formatDataSetIdentity(report.DataSet))
	if report.EventCounts.Duration != "" {
		fmt.Printf("duration: %s\n", report.EventCounts.Duration)
	}
	if report.Result.WinnerKnown {
		fmt.Printf("result: %s winners=%v losers=%v\n", report.Result.Method, report.Result.Winners, report.Result.Losers)
	}
	fmt.Printf("verification: %s\n", report.Verification)
	fmt.Printf("coverage: replay_actions=%d decoded=%d undecoded=%d decoded_percent=%.2f coordinate_actions=%d feedback=%d viewlocks=%d\n", report.Coverage.ReplayActions, report.Coverage.DecodedActions, report.Coverage.UndecodedActions, report.Coverage.DecodedPercent, report.Coverage.CoordinateActions, report.Coverage.FeedbackEvents, report.Coverage.ViewlockEvents)
	for _, player := range report.Players {
		name := player.PlayerName
		if name == "" {
			name = "Unknown"
		}
		fmt.Printf("\nP%d %s kind=%s actions=%d decoded=%d undecoded=%d apm=%.2f spatial=%d cells=%d switches=%d chat=%d taunts=%d flares=%d telemetry=%d\n", player.PlayerID, name, player.Kind, player.Actions, player.DecodedActions, player.Undecoded, player.APM, player.Spatial.Commands, player.Spatial.CellsVisited, player.Spatial.CentroidSwitches, player.Feedback.Chat, player.Feedback.Taunts, player.Feedback.Flares, player.Feedback.Telemetry)
		if len(player.Feedback.ByPhase) > 0 {
			fmt.Printf("  phases: %s\n", formatStringIntMap(player.Feedback.ByPhase))
		}
		if len(player.Vocabulary) > 0 {
			fmt.Println("  vocabulary:")
			for _, item := range player.Vocabulary {
				fmt.Printf("  - %s count=%d share=%.2f%% decoded=%t\n", item.Key, item.Count, item.SharePercent, item.Decoded)
			}
		}
		if len(player.APMWindows) > 0 {
			peak := player.APMWindows[0]
			for _, window := range player.APMWindows[1:] {
				if window.APM > peak.APM {
					peak = window
				}
			}
			fmt.Printf("  peak_window: %s-%s actions=%d apm=%.2f\n", peak.Start, peak.End, peak.Actions, peak.APM)
		}
		if len(player.DeadGaps) > 0 {
			longest := player.DeadGaps[0]
			for _, gap := range player.DeadGaps[1:] {
				if gap.DurationMS > longest.DurationMS {
					longest = gap
				}
			}
			fmt.Printf("  longest_dead_gap: %s %s-%s\n", longest.Duration, longest.Start, longest.End)
		}
		if len(player.Spatial.TopCells) > 0 {
			fmt.Println("  top_cells:")
			for _, cell := range player.Spatial.TopCells {
				fmt.Printf("  - %s min=(%.0f,%.0f) count=%d\n", cell.Cell, cell.MinX, cell.MinY, cell.Count)
			}
		}
		if len(player.Spatial.TopRegions) > 0 {
			fmt.Println("  top_regions:")
			for _, region := range player.Spatial.TopRegions {
				fmt.Printf("  - %s events=%d flares=%d\n", region.Name, region.Events, region.Flares)
			}
		}
		if len(player.Notables) > 0 {
			fmt.Println("  notables:")
			for _, notable := range player.Notables {
				fmt.Printf("  - %s: %s\n", notable.Name, notable.Detail)
			}
		}
	}
	if len(report.Camera) > 0 {
		fmt.Printf("\ncamera_observed: %d viewlock events (global, not attributed to players)\n", len(report.Camera))
	}
	if len(report.Claims) > 0 {
		fmt.Println("\nclaims:")
		for _, claim := range report.Claims {
			detail := ""
			if claim.Detail != "" {
				detail = " " + claim.Detail
			}
			fmt.Printf("- %s status=%s source=%s confidence=%s%s\n", claim.Name, claim.Status, claim.Source, claim.Confidence, detail)
		}
	}
	if len(report.Missing) > 0 {
		fmt.Println("missing:")
		for _, missing := range report.Missing {
			fmt.Printf("- %s\n", missing)
		}
	}
	if len(report.Warnings) > 0 {
		fmt.Println("warnings:")
		for _, warning := range report.Warnings {
			fmt.Printf("- %s\n", warning)
		}
	}
}

func formatStringIntMap(values map[string]int) string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%d", key, values[key]))
	}
	return strings.Join(parts, " ")
}

func printTelemetryReports(reports []replay.TelemetryReport) {
	for _, report := range reports {
		status := "OK"
		if !report.OK {
			status = "FAIL"
		}
		fmt.Printf("%s: %s events=%d issues=%d\n", report.Path, status, len(report.Events), len(report.Issues))
		for _, event := range report.Events {
			name := event.Name
			if name == "" {
				name = "(unnamed)"
			}
			fmt.Printf("- %s P%d %s fields=%v\n", event.Time, event.PlayerID, name, event.Fields)
		}
		for _, issue := range report.Issues {
			fmt.Printf("- [%s] %s: %s\n", issue.Severity, issue.Code, issue.Message)
		}
	}
}

func printReplayIssues(report *replay.IssuesReport) {
	fmt.Printf("path: %s\n", report.Path)
	fmt.Printf("cards: %d groups=%d\n", len(report.Cards), len(report.Groups))
	if len(report.Cards) > 0 {
		fmt.Println("cards:")
		for _, card := range report.Cards {
			where := card.Chapter
			if card.Region != "" {
				where += "/" + card.Region
			}
			player := ""
			if card.PlayerID > 0 {
				player = fmt.Sprintf(" P%d", card.PlayerID)
				if card.PlayerName != "" {
					player += " " + card.PlayerName
				}
			}
			fmt.Printf("- %s count=%d%s %s %s first=%s last=%s confidence=%s\n", card.ID, card.Count, player, where, card.Text, card.FirstTime, card.LastTime, card.Confidence)
			limit := len(card.Evidence)
			if limit > 3 {
				limit = 3
			}
			for _, evidence := range card.Evidence[:limit] {
				fmt.Printf("  - %s P%d %s\n", evidence.Time, evidence.PlayerID, evidence.Text)
			}
		}
	}
	fmt.Printf("groups: %d\n", len(report.Groups))
	for _, group := range report.Groups {
		label := group.Key
		if group.Region != "" {
			label = group.Region
		} else if group.PlayerID > 0 {
			name := group.PlayerName
			if name == "" {
				name = "Unknown"
			}
			label = fmt.Sprintf("P%d %s", group.PlayerID, name)
		}
		fmt.Printf("- %s count=%d\n", label, group.Count)
		for _, item := range group.Items {
			fmt.Printf("  - %s %s\n", item.Time, item.Text)
		}
	}
	if len(report.Warnings) > 0 {
		fmt.Println("warnings:")
		for _, warning := range report.Warnings {
			fmt.Printf("- %s\n", warning)
		}
	}
}

func printReplayUnknowns(report *replay.UnknownsReport) {
	fmt.Printf("root: %s\n", report.Root)
	fmt.Printf("replays: %d action_ids=%d unknown_ops=%d\n", report.Replays, len(report.ActionIDs), len(report.Operations))
	if len(report.ActionIDs) > 0 {
		fmt.Println("action_ids:")
		for _, item := range report.ActionIDs {
			sample := ""
			if item.SampleHex != "" {
				sample = " sample=" + item.SampleHex
			}
			fmt.Printf("- id=%d count=%d payload_bytes=%v replays=%d%s\n", item.ActionID, item.Count, item.PayloadBytes, len(item.ReplayPaths), sample)
		}
	}
	if len(report.Operations) > 0 {
		fmt.Println("unknown_operations:")
		for _, item := range report.Operations {
			fmt.Printf("- op=%d count=%d replays=%d\n", item.OperationID, item.Count, len(item.ReplayPaths))
		}
	}
	if len(report.Warnings) > 0 {
		fmt.Println("warnings:")
		for _, warning := range report.Warnings {
			fmt.Printf("- %s\n", warning)
		}
	}
}

func printStoryLine(line replay.StoryLine) {
	time := ""
	if line.Time != "" {
		time = line.Time + " "
	}
	player := ""
	if line.PlayerID > 0 {
		player = fmt.Sprintf("P%d", line.PlayerID)
		if line.PlayerName != "" {
			player += " " + line.PlayerName
		}
		player += " "
	}
	region := ""
	if line.Region != "" {
		region = " [" + line.Region + "]"
	}
	fmt.Printf("- %s%s%s%s\n", time, player, line.Text, region)
}

func replayEventTypeCounts(counts map[string]int) []string {
	keys := make([]string, 0, len(counts))
	for key := range counts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]string, 0, len(keys))
	for _, key := range keys {
		out = append(out, fmt.Sprintf("%s=%d", key, counts[key]))
	}
	return out
}

func replayActionIDCounts(counts map[int]int) []string {
	keys := make([]int, 0, len(counts))
	for key := range counts {
		keys = append(keys, key)
	}
	sort.Ints(keys)
	out := make([]string, 0, len(keys))
	for _, key := range keys {
		out = append(out, fmt.Sprintf("%d=%d", key, counts[key]))
	}
	return out
}

func stringCounts(counts map[string]int) []string {
	keys := make([]string, 0, len(counts))
	for key := range counts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]string, 0, len(keys))
	for _, key := range keys {
		out = append(out, fmt.Sprintf("%q=%d", key, counts[key]))
	}
	return out
}

func formatDataSetIdentity(identity replay.DataSetIdentity) string {
	switch identity.Status {
	case "vanilla":
		return "vanilla"
	case "modded":
		suffix := ""
		if identity.Checksum != 0 {
			suffix += fmt.Sprintf(" checksum=%d", identity.Checksum)
		}
		if identity.WorkshopID != 0 {
			suffix += fmt.Sprintf(" workshop_id=%d", identity.WorkshopID)
		}
		if len(identity.ActiveDataSets) > 0 {
			return "modded:" + strings.Join(identity.ActiveDataSets, ",") + suffix
		}
		if identity.ActiveDataSet != "" {
			return "modded:" + identity.ActiveDataSet + suffix
		}
		return "modded" + suffix
	case "unknown":
		if identity.Error != "" {
			return "unknown(" + identity.Error + ")"
		}
		return "unknown"
	default:
		return "unknown"
	}
}

func printFeedback(report *replay.FeedbackReport) {
	fmt.Printf("path: %s\n", report.Path)
	fmt.Printf("method: %s\n", report.Method)
	fmt.Printf("events: chat=%d taunt_numbers=%d flares=%d\n", report.Counts.Chat, report.Counts.Taunts, report.Counts.Flares)
	if len(report.Players) > 0 {
		fmt.Println("players:")
		for _, player := range report.Players {
			label := player.Name
			if label == "" {
				label = "Unknown"
			}
			fmt.Printf("- P%d %s human=%t kind=%s\n", player.PlayerID, label, player.Human, player.Kind)
		}
	}
	if len(report.Taunts) > 0 {
		fmt.Println("taunts:")
		for _, taunt := range report.Taunts {
			label := ""
			if taunt.Text != "" {
				label = " " + taunt.Text
			}
			fmt.Printf("- %d:%s count=%d\n", taunt.Number, label, taunt.Count)
		}
	}
	if len(report.Timeline) > 0 {
		fmt.Println("timeline:")
		for _, event := range report.Timeline {
			time := ""
			if event.Time != "" {
				time = event.Time + " "
			}
			name := event.PlayerName
			if name != "" {
				name = " " + name
			}
			switch event.Kind {
			case "FLARE":
				fmt.Printf("- #%04d %sP%d%s FLARE x=%.2f y=%.2f targets=%v\n", event.Index, time, event.PlayerID, name, event.X, event.Y, event.Targets)
			default:
				suffix := ""
				if event.TauntNumber > 0 {
					if event.TauntText != "" {
						suffix = fmt.Sprintf(" [taunt %d: %s]", event.TauntNumber, event.TauntText)
					} else {
						suffix = fmt.Sprintf(" [taunt %d]", event.TauntNumber)
					}
				}
				fmt.Printf("- #%04d %sP%d%s %s %q%s\n", event.Index, time, event.PlayerID, name, event.ChannelName, event.Text, suffix)
			}
		}
	}
	if len(report.Warnings) > 0 {
		fmt.Println("warnings:")
		for _, warning := range report.Warnings {
			fmt.Printf("- %s\n", warning)
		}
	}
}

func printReplayChat(report *replay.ChatReport) {
	fmt.Printf("path: %s\n", report.Path)
	fmt.Printf("method: %s\n", report.Method)
	fmt.Printf("verification: %s (%s)\n", report.Verification.Label, report.Verification.Note)
	fmt.Printf("chat: total=%d lobby=%d backlog=%d game=%d duplicates_hidden=%d\n", report.Counts.Total, report.Counts.Lobby, report.Counts.Backlog, report.Counts.Game, report.Counts.DuplicatesHidden)
	if len(report.Players) > 0 {
		fmt.Println("players:")
		for _, player := range report.Players {
			label := player.Name
			if label == "" {
				label = "Unknown"
			}
			fmt.Printf("- P%d %s human=%t kind=%s\n", player.PlayerID, label, player.Human, player.Kind)
		}
	}
	if len(report.Lines) > 0 {
		fmt.Println("transcript:")
		for _, line := range report.Lines {
			when := "lobby"
			if line.Time != "" {
				when = line.Time
			}
			name := line.PlayerName
			if name == "" && line.PlayerID > 0 {
				name = fmt.Sprintf("P%d", line.PlayerID)
			}
			if name == "" {
				name = "Unknown"
			}
			channel := line.ChannelName
			if channel == "" {
				channel = "chat"
			}
			suffix := ""
			if line.TauntNumber > 0 {
				if line.TauntText != "" {
					suffix = fmt.Sprintf(" [taunt %d: %s]", line.TauntNumber, line.TauntText)
				} else {
					suffix = fmt.Sprintf(" [taunt %d]", line.TauntNumber)
				}
			}
			fmt.Printf("- #%04d [%s] %s %s %s: %q%s\n", line.Index, line.Source, when, name, channel, line.Text, suffix)
		}
	}
	if len(report.Warnings) > 0 {
		fmt.Println("warnings:")
		for _, warning := range report.Warnings {
			fmt.Printf("- %s\n", warning)
		}
	}
}

func printReplayCoverage(report *replay.CoverageReport) {
	fmt.Printf("path: %s\n", report.Path)
	fmt.Printf("file_bytes: %d compressed_header=%d inflated_header=%d body=%d\n", report.FileBytes, report.Summary.CompressedHeaderBytes, report.Summary.InflatedHeaderBytes, report.Summary.BodyBytes)
	fmt.Printf("header: decoded=%d opaque=%d decoded_percent=%.2f\n", report.Summary.HeaderDecodedBytes, report.Summary.HeaderOpaqueBytes, report.Summary.HeaderDecodedPercent)
	fmt.Printf("body: decoded=%d opaque=%d decoded_percent=%.2f\n", report.Summary.BodyDecodedBytes, report.Summary.BodyOpaqueBytes, report.Summary.BodyDecodedPercent)
	if len(report.BodyOps) > 0 {
		fmt.Printf("body_ops: %s\n", strings.Join(replayActionIDCounts(report.BodyOps), " "))
	}
	if report.HeaderSpine != nil {
		spine := report.HeaderSpine
		fmt.Printf("initial_spine: initial=%d..%d players=%d restore_time=%d particles=%d trigger_start=%d trigger_boundary=%s\n",
			spine.InitialStart, spine.InitialEnd, len(spine.Players), spine.RestoreTime, spine.NumParticles, spine.TriggerStart, spine.TriggerBoundaryCheck)
		for _, player := range spine.Players {
			attr := ""
			if player.AttributesBytes > 0 {
				attr = fmt.Sprintf(" attrs=%d camera=%.2f,%.2f spawn=%d,%d civ_key=%q", player.AttributesBytes, player.CameraX, player.CameraY, player.SpawnX, player.SpawnY, player.CivilizationKey)
			}
			objectBand := ""
			if player.ObjectCandidateCount > 0 {
				objectBand = fmt.Sprintf(" object_band=%d..%d candidates=%d", player.ObjectCandidateBandStart, player.ObjectCandidateBandEnd, player.ObjectCandidateCount)
			}
			fmt.Printf("- initial %s name=%q span=%d..%d bytes=%d payload=%d%s%s object_boundary=%s\n",
				player.Label, player.Name, player.Start, player.End, player.Bytes, player.PayloadSpanBytes, attr, objectBand, player.ObjectBoundaryConfidence)
		}
	}
	if len(report.Regions) > 0 {
		fmt.Println("regions:")
		for _, region := range report.Regions {
			if region.Space == "body" && strings.HasPrefix(region.Name, "op_") {
				continue
			}
			if strings.Contains(region.Name, "_object_candidate_prefix_") ||
				strings.Contains(region.Name, "_object_candidate_body_") ||
				strings.Contains(region.Name, "_object_sprite_list_prefix_") ||
				strings.Contains(region.Name, "_object_de_extension_prefix_") ||
				strings.Contains(region.Name, "_object_static_tail_prefix_") ||
				strings.Contains(region.Name, "_object_moving_prefix_") ||
				strings.Contains(region.Name, "_object_tail_gap_before_prefix_") {
				continue
			}
			fmt.Printf("- %s %s %d..%d bytes=%d status=%s confidence=%s\n", region.Space, region.Name, region.Start, region.End, region.Bytes, region.Status, region.Confidence)
		}
	}
	if len(report.Samples) > 0 {
		fmt.Println("body_samples:")
		for _, sample := range report.Samples {
			fmt.Printf("- %s %d..%d bytes=%d status=%s confidence=%s\n", sample.Name, sample.Start, sample.End, sample.Bytes, sample.Status, sample.Confidence)
		}
	}
	if len(report.Warnings) > 0 {
		fmt.Println("warnings:")
		for _, warning := range report.Warnings {
			fmt.Printf("- %s\n", warning)
		}
	}
}

func printReplayOpaqueSpans(report *replay.OpaqueSpanReport) {
	fmt.Printf("path: %s\n", report.Path)
	fmt.Printf("verification: %s (%s)\n", report.Verification.Label, report.Verification.Note)
	fmt.Printf("header: inflated=%d opaque=%d body: bytes=%d opaque=%d spans=%d shown=%d\n",
		report.Summary.InflatedHeaderBytes, report.Summary.HeaderOpaqueBytes, report.Summary.BodyBytes, report.Summary.BodyOpaqueBytes, report.Summary.SpanCount, report.Summary.Shown)
	for _, span := range report.Spans {
		hints := ""
		if len(span.ShapeHints) > 0 {
			hints = " hints=" + strings.Join(span.ShapeHints, ",")
		}
		fmt.Printf("- %s %s %d..%d bytes=%d entropy=%.3f distinct=%d zero=%d ff=%d printable=%d prev=%q next=%q%s\n",
			span.Space, span.Name, span.Start, span.End, span.Bytes, span.Entropy, span.DistinctBytes, span.ZeroBytes, span.FFBytes, span.PrintableBytes, span.PrevRegion, span.NextRegion, hints)
		if span.HexSample != "" {
			fmt.Printf("  sample_hex: %s\n", span.HexSample)
		}
		if len(span.TextSamples) > 0 {
			fmt.Printf("  text_samples: %q\n", span.TextSamples)
		}
	}
	if len(report.Warnings) > 0 {
		fmt.Println("warnings:")
		for _, warning := range report.Warnings {
			fmt.Printf("- %s\n", warning)
		}
	}
}

func printReplayFrontier(report *replay.FrontierReport) {
	fmt.Printf("path: %s\n", report.Path)
	fmt.Printf("verification: %s (%s)\n", report.Verification.Label, report.Verification.Note)
	fmt.Printf("header: decoded=%d opaque=%d decoded_percent=%.2f body_opaque=%d opaque_spans=%d duplicate_groups=%d shown_duplicates=%d repeated_duplicate_bytes=%d\n",
		report.Summary.HeaderDecodedBytes,
		report.Summary.HeaderOpaqueBytes,
		report.Summary.HeaderDecodedPct,
		report.Summary.BodyOpaqueBytes,
		report.Summary.OpaqueSpans,
		report.Summary.DuplicateGroups,
		report.Summary.ShownDuplicates,
		report.Summary.DuplicateBytes)
	if report.Summary.TemplateHitCount > 0 {
		fmt.Printf("template_hits: count=%d shown=%d bytes=%d\n", report.Summary.TemplateHitCount, report.Summary.ShownTemplateHits, report.Summary.TemplateHitBytes)
	}
	if len(report.Buckets) > 0 {
		fmt.Println("buckets:")
		for _, bucket := range report.Buckets {
			fmt.Printf("- %s bytes=%d residual=%d template_hit_bytes=%d spans=%d pct=%.2f confidence=%s\n",
				bucket.Name, bucket.Bytes, bucket.ResidualBytes, bucket.TemplateHitBytes, bucket.Spans, bucket.Percent, bucket.Confidence)
			fmt.Printf("  hypothesis: %s\n", bucket.Hypothesis)
			fmt.Printf("  next: %s\n", bucket.NextMove)
			if len(bucket.TextSamples) > 0 {
				fmt.Printf("  text_samples: %s\n", strings.Join(bucket.TextSamples, " | "))
			}
			for _, span := range bucket.Largest {
				fmt.Printf("  span: %s %d..%d bytes=%d entropy=%.3f zero=%d ff=%d prev=%q next=%q\n",
					span.Name, span.Start, span.End, span.Bytes, span.Entropy, span.ZeroBytes, span.FFBytes, span.PrevRegion, span.NextRegion)
			}
		}
	}
	if len(report.Duplicates) > 0 {
		fmt.Println("duplicates:")
		for _, group := range report.Duplicates {
			hash := group.SHA256
			if len(hash) > 12 {
				hash = hash[:12]
			}
			fmt.Printf("- sha=%s bytes=%d count=%d repeated=%d buckets=%s sample=%s\n",
				hash, group.Bytes, group.Count, group.RepeatedBytes, strings.Join(group.Buckets, ","), group.HexSample)
			for _, span := range group.Spans {
				fmt.Printf("  %s %s %d..%d bucket=%s\n", span.Space, span.Name, span.Start, span.End, span.Bucket)
			}
		}
	}
	if len(report.TemplateHits) > 0 {
		fmt.Println("template_hits:")
		for _, hit := range report.TemplateHits {
			hash := hit.MatchedPayloadSHA
			if len(hash) > 12 {
				hash = hash[:12]
			}
			fmt.Printf("- %s %d..%d bytes=%d bucket=%s source=%s %d..%d rel=%d sha=%s sample=%s\n",
				hit.Name, hit.Start, hit.End, hit.Bytes, hit.Bucket, hit.SourceRegion, hit.SourceStart, hit.SourceEnd, hit.SourceRelative, hash, hit.MatchedPayloadHead)
		}
	}
	if len(report.Warnings) > 0 {
		fmt.Println("warnings:")
		for _, warning := range report.Warnings {
			fmt.Printf("- %s\n", warning)
		}
	}
}

func printReplayCorpus(report *replay.ReplayCorpusReport) {
	fmt.Printf("folder: %s\n", report.Folder)
	fmt.Printf("method: %s\n", report.Method)
	fmt.Printf("verification: %s\n", report.Verification)
	fmt.Printf("summary: scanned=%d failures=%d scenario=%d non_scenario=%d players=%d humans=%d ai=%d actions=%d untyped=%d unknown_ops=%d chat=%d taunts=%d flares=%d resigns=%d postgames=%d\n",
		report.Summary.Scanned, report.Summary.Failures, report.Summary.ScenarioReplays, report.Summary.NonScenarioReplays,
		report.Summary.Players, report.Summary.Humans, report.Summary.AIs, report.Summary.TotalActions,
		report.Summary.UntypedActions, report.Summary.UnknownOps, report.Summary.Chat, report.Summary.Taunts,
		report.Summary.Flares, report.Summary.Resigns, report.Summary.Postgames)
	fmt.Printf("coverage: header=%dB opaque=%dB min=%.2f avg=%.2f body=%dB opaque=%dB min=%.2f avg=%.2f\n",
		report.Summary.HeaderBytes, report.Summary.HeaderOpaqueBytes, report.Summary.HeaderDecodedPercentMin, report.Summary.HeaderDecodedPercentAvg,
		report.Summary.BodyBytes, report.Summary.BodyOpaqueBytes, report.Summary.BodyDecodedPercentMin, report.Summary.BodyDecodedPercentAvg)
	if len(report.Summary.BodyOps) > 0 {
		fmt.Printf("body_ops: %s\n", strings.Join(replayActionIDCounts(report.Summary.BodyOps), " "))
	}
	if len(report.Summary.PostgameShapes) > 0 {
		fmt.Printf("postgame_shapes: %s\n", strings.Join(stringCounts(report.Summary.PostgameShapes), " "))
	}
	if len(report.Unknowns) > 0 {
		fmt.Println("unknown_actions:")
		for _, group := range report.Unknowns {
			fmt.Printf("- action=%d count=%d files=%d", group.ActionID, group.Count, group.Files)
			if len(group.Players) > 0 {
				fmt.Printf(" players=%v", group.Players)
			}
			if len(group.Shapes) > 0 {
				var shapes []string
				for _, shape := range group.Shapes {
					shapes = append(shapes, fmt.Sprintf("%dB:%d", shape.DeltaMS, shape.Count))
				}
				fmt.Printf(" shapes=%s", strings.Join(shapes, ","))
			}
			fmt.Println()
		}
	}
	fmt.Println("replays:")
	for _, row := range report.Rows {
		if !row.OK {
			fmt.Printf("- %s ERROR %s\n", row.Path, row.Error)
			continue
		}
		identity := shortHash(row.ScenarioIdentity)
		if identity == "" {
			identity = "none"
		}
		fmt.Printf("- %s kind=%s id=%s tier=%s save=%.2f players=%d duration=%s actions=%d untyped=%d body_decoded=%.2f opaque=%dB postgame=%s\n",
			row.Path, row.ReplayKind, identity, row.ScenarioIdentityTier, row.SaveVersion, row.PlayerCount,
			row.Duration, row.Actions, row.UntypedActions, row.BodyDecodedPercent, row.BodyOpaqueBytes, row.PostgameShape)
		if len(row.PlayerNames) > 0 {
			fmt.Printf("  players: %s\n", strings.Join(row.PlayerNames, ", "))
		}
		if len(row.UnknownActionIDs) > 0 {
			fmt.Printf("  unknown_actions: %s\n", strings.Join(replayActionIDCounts(row.UnknownActionIDs), " "))
		}
	}
	if len(report.Warnings) > 0 {
		fmt.Println("warnings:")
		for _, warning := range report.Warnings {
			fmt.Printf("- %s\n", warning)
		}
	}
}

func printReplayAIManifest(report *replay.AIManifestReport) {
	fmt.Printf("path: %s\n", report.Path)
	fmt.Printf("method: %s\n", report.Method)
	fmt.Printf("verification: %s (%s)\n", report.Verification.Label, report.Verification.Note)
	fmt.Printf("coverage: header_decoded=%d header_opaque=%d inflated_header=%d\n",
		report.Summary.HeaderDecodedBytes, report.Summary.HeaderOpaqueBytes, report.Summary.InflatedHeaderBytes)
	fmt.Printf("promide_mentions=%d load_directives=%d xs_includes=%d unique_modules=%d resolved=%d missing=%d promisory_root_ok=%t\n",
		report.Summary.PromiDEMentions, report.Summary.LoadDirectives, report.Summary.XSIncludes, report.Summary.UniqueModules,
		report.Summary.ResolvedModules, report.Summary.MissingModules, report.PromisoryRootOK)
	if report.PromisoryRoot != "" {
		fmt.Printf("promisory_root: %s\n", report.PromisoryRoot)
	}
	if len(report.UniqueModules) > 0 {
		fmt.Println("modules:")
		for _, module := range report.UniqueModules {
			resolved := "missing"
			if module.Resolved {
				resolved = module.ResolvedPath
			}
			fmt.Printf("- %s count=%d resolved=%s\n", module.Name, module.Count, resolved)
		}
	}
	if len(report.References) > 0 {
		fmt.Println("references:")
		for _, ref := range report.References {
			resolved := ""
			if ref.ResolvedPath != "" {
				resolved = " -> " + ref.ResolvedPath
			}
			fmt.Printf("- %d..%d %s %q%s region=%s\n", ref.Start, ref.End, ref.ModuleName, ref.Directive, resolved, ref.Region)
		}
	}
	if len(report.XSIncludes) > 0 {
		fmt.Println("xs_includes:")
		for _, include := range report.XSIncludes {
			fmt.Printf("- %d..%d %s %q region=%s\n", include.Start, include.End, include.IncludeName, include.Directive, include.Region)
		}
	}
	if len(report.PromiDEMentions) > 0 {
		fmt.Println("promide_mentions:")
		for _, mention := range report.PromiDEMentions {
			fmt.Printf("- %d..%d region=%s status=%s\n", mention.Offset, mention.End, mention.Region, mention.Status)
		}
	}
	if len(report.Warnings) > 0 {
		fmt.Println("warnings:")
		for _, warning := range report.Warnings {
			fmt.Printf("- %s\n", warning)
		}
	}
}

func printReplayOpaqueClusters(report *replay.OpaqueClusterReport) {
	fmt.Printf("folder: %s\n", report.Folder)
	fmt.Printf("method: %s\n", report.Method)
	fmt.Printf("verification: %s\n", report.Verification)
	fmt.Printf("summary: files=%d failures=%d spans=%d clusters=%d shown=%d opaque=%dB header=%dB body=%dB\n",
		report.Summary.Files, report.Summary.Failures, report.Summary.Spans, report.Summary.Clusters,
		report.Summary.Shown, report.Summary.TotalOpaqueBytes, report.Summary.HeaderOpaqueBytes, report.Summary.BodyOpaqueBytes)
	for _, cluster := range report.Clusters {
		fmt.Printf("- key=%s files=%d count=%d total=%dB bytes=%d..%d avg=%.2f entropy=%.3f zero=%.2f%% ff=%.2f%% printable=%.2f%% space=%s name=%q bucket=%s\n",
			cluster.Key, cluster.Files, cluster.Count, cluster.TotalBytes, cluster.MinBytes, cluster.MaxBytes,
			cluster.AvgBytes, cluster.AvgEntropy, cluster.AvgZeroPercent, cluster.AvgFFPercent,
			cluster.AvgPrintablePercent, cluster.Space, cluster.NamePattern, cluster.ByteBucket)
		if cluster.PrevPattern != "" || cluster.NextPattern != "" {
			fmt.Printf("  neighbors: prev=%q next=%q\n", cluster.PrevPattern, cluster.NextPattern)
		}
		if len(cluster.ShapeHints) > 0 {
			fmt.Printf("  hints: %s\n", strings.Join(cluster.ShapeHints, ","))
		}
		for _, sample := range cluster.Samples {
			fmt.Printf("  sample: %s %s %d..%d bytes=%d hex=%s\n", sample.Path, sample.SpanName, sample.Start, sample.End, sample.Bytes, sample.HexSample)
		}
	}
	if len(report.Warnings) > 0 {
		fmt.Println("warnings:")
		for _, warning := range report.Warnings {
			fmt.Printf("- %s\n", warning)
		}
	}
}

func printReplayEffectiveData(report *replay.EffectiveDataReport) {
	fmt.Printf("path: %s\n", report.Path)
	if report.DatPath != "" {
		fmt.Printf("dat: %s\n", report.DatPath)
	}
	fmt.Printf("method: %s\n", report.Method)
	fmt.Printf("verification: %s\n", report.Verification)
	fmt.Printf("template: ref_player=%d label=%s bytes=%d sha256=%s",
		report.Template.ReferencePlayer, report.Template.ReferenceLabel, report.Template.TailBytes, report.Template.SHA256)
	if report.Template.KnownTemplateStatus != "" {
		fmt.Printf(" known_template=%s", report.Template.KnownTemplateStatus)
	}
	fmt.Println()
	fmt.Printf("template_check: status=%s baseline=%s expected=%s confidence=%s\n",
		report.TemplateCheck.Status, report.TemplateCheck.BaselineName, report.TemplateCheck.ExpectedSHA256, report.TemplateCheck.Confidence)
	fmt.Printf("availability_array: tail_offset=%d count=%d bytes=%d abs=%d..%d confidence=%s\n",
		report.AvailabilityArray.TailOffset, report.AvailabilityArray.Count, report.AvailabilityArray.Bytes,
		report.AvailabilityArray.AbsStart, report.AvailabilityArray.AbsEnd, report.AvailabilityArray.Confidence)
	for _, player := range report.Players {
		fmt.Printf("- player=%d name=%q civ=%d/%s tail=%d..%d bytes=%d identical=%.2f%% diff_runs=%d entries=%d changed=%d available=%d unavailable=%d levels=0:%d 1:%d 2:%d 3:%d other:%d shown=%d\n",
			player.PlayerNumber, player.PlayerName, player.CivID, player.CivName, player.TailStart, player.TailEnd,
			player.TailBytes, player.IdenticalRatio, player.DiffRuns, player.EntryCount, player.ChangedEntries, player.Available,
			player.Unavailable, player.Level0, player.Level1, player.Level2, player.Level3, player.OtherValues, player.Shown)
		for _, entry := range player.Entries {
			enabled := ""
			if entry.DatEnabled != nil {
				enabled = fmt.Sprintf(" dat_enabled=%d", *entry.DatEnabled)
			}
			fmt.Printf("  - idx=%d off=%d value=%d ref=%d changed=%t replay_available=%t meaning=%s unit_slot=%d unit_id=%d name=%q present=%t%s confidence=%s\n",
				entry.Index, entry.TailOffset, entry.Value, entry.ReferenceValue, entry.Changed, entry.ReplayAvailable, entry.Meaning,
				entry.UnitSlot, entry.UnitID, entry.UnitName, entry.DatPresent, enabled, entry.Confidence)
		}
		for _, warning := range player.Warnings {
			fmt.Printf("  warning: %s\n", warning)
		}
	}
	if len(report.Warnings) > 0 {
		fmt.Println("warnings:")
		for _, warning := range report.Warnings {
			fmt.Printf("- %s\n", warning)
		}
	}
}

func printReplayEffectiveTree(report *replay.EffectiveDataReport) {
	fmt.Printf("path: %s\n", report.Path)
	if report.DatPath != "" {
		fmt.Printf("dat: %s\n", report.DatPath)
	}
	fmt.Printf("verification: %s\n", report.Verification)
	fmt.Printf("template_detector: status=%s baseline=%s expected=%s observed=%s confidence=%s\n",
		report.TemplateCheck.Status, report.TemplateCheck.BaselineName, report.TemplateCheck.ExpectedSHA256,
		report.TemplateCheck.ObservedSHA256, report.TemplateCheck.Confidence)
	if report.TemplateCheck.Note != "" {
		fmt.Printf("template_note: %s\n", report.TemplateCheck.Note)
	}
	fmt.Printf("array: effective_unit_slot_availability offset=%d count=%d abs=%d..%d\n",
		report.AvailabilityArray.TailOffset, report.AvailabilityArray.Count, report.AvailabilityArray.AbsStart, report.AvailabilityArray.AbsEnd)
	for _, player := range report.Players {
		fmt.Printf("\nplayer %d %q civ=%d/%s\n", player.PlayerNumber, player.PlayerName, player.CivID, player.CivName)
		fmt.Printf("summary: entries=%d changed_vs_template=%d available=%d unavailable=%d levels=0:%d 1:%d 2:%d 3:%d other:%d dat_named=%d identical=%.2f%% diff_runs=%d shown=%d\n",
			player.EntryCount, player.ChangedEntries, player.Available, player.Unavailable, player.Level0, player.Level1,
			player.Level2, player.Level3, player.OtherValues, player.DatNamed, player.IdenticalRatio, player.DiffRuns, player.Shown)
		fmt.Println("effective unit availability:")
		for _, entry := range player.Entries {
			name := entry.UnitName
			if name == "" {
				name = fmt.Sprintf("unit_slot_%d", entry.UnitSlot)
			}
			enabled := "unknown"
			if entry.DatEnabled != nil {
				enabled = fmt.Sprintf("%d", *entry.DatEnabled)
			}
			change := ""
			if entry.Changed {
				change = fmt.Sprintf(" ref=%d", entry.ReferenceValue)
			}
			fmt.Printf("  [%03d] unit_id=%d %-32s replay_value=%d %-26s available=%t dat_enabled=%s%s\n",
				entry.UnitSlot, entry.UnitID, clippedField(name, 32), entry.Value, entry.Meaning, entry.ReplayAvailable, enabled, change)
		}
		for _, warning := range player.Warnings {
			fmt.Printf("warning: player %d: %s\n", player.PlayerNumber, warning)
		}
	}
	if len(report.Warnings) > 0 {
		fmt.Println("\nwarnings:")
		for _, warning := range report.Warnings {
			fmt.Printf("- %s\n", warning)
		}
	}
}

func printReplayEffectiveUnits(report *replay.EffectiveUnitStatsReport) {
	fmt.Printf("path: %s\n", report.Path)
	if report.DatPath != "" {
		fmt.Printf("dat: %s\n", report.DatPath)
	}
	fmt.Printf("method: %s\n", report.Method)
	fmt.Printf("verification: %s\n", report.Verification)
	fmt.Printf("template: ref_player=%d label=%s bytes=%d sha256=%s\n",
		report.TargetTemplate.ReferencePlayer, report.TargetTemplate.ReferenceLabel,
		report.TargetTemplate.TailBytes, report.TargetTemplate.SHA256)
	fmt.Printf("template_check: status=%s baseline=%s expected=%s observed=%s confidence=%s\n",
		report.TemplateCheck.Status, report.TemplateCheck.BaselineName, report.TemplateCheck.ExpectedSHA256,
		report.TemplateCheck.ObservedSHA256, report.TemplateCheck.Confidence)
	fmt.Printf("decoded_fields: %s\n", strings.Join(report.DecodedFields, ", "))
	fmt.Printf("not_decoded: %s\n", strings.Join(report.NotDecoded, ", "))
	fmt.Printf("summary: players=%d units_considered=%d rows=%d changed_hp=%d mod_confirmed_multi_civ=%d unresolved_single_civ=%d duplicate_name_anchors=%d missing_name_anchors=%d invalid_hp_reads=%d\n",
		report.Summary.Players, report.Summary.UnitsConsidered, report.Summary.Rows, report.Summary.ChangedHitPoints,
		report.Summary.ModConfirmedMultiCiv, report.Summary.UnresolvedSingleCiv,
		report.Summary.DuplicateNameAnchors, report.Summary.MissingNameAnchors, report.Summary.InvalidHitPointReads)
	for _, player := range report.Players {
		fmt.Printf("- player=%d name=%q civ=%d/%s tail_bytes=%d identical=%.2f%% diff_runs=%d units=%d rows=%d changed_hp=%d duplicate_names=%d missing_names=%d invalid_hp=%d\n",
			player.PlayerIndex, player.PlayerName, player.CivID, player.CivName, player.TailBytes,
			player.IdenticalRatioVsRef, player.DiffRunsVsRef, player.UnitsConsidered, player.Rows,
			player.ChangedHitPoints, player.DuplicateNameAnchors, player.MissingNameAnchors, player.InvalidHitPointReads)
	}
	if len(report.Rows) > 0 {
		fmt.Println("rows:")
		for _, row := range report.Rows {
			fmt.Printf("- player=%d civ=%d/%s unit=%d/%s type=%d hp=%d->%d delta=%+d civ_span=%d class=%s name_off=%d hp_off=%d occurrences=%d changed=%t confidence=%s\n",
				row.PlayerIndex, row.CivID, row.CivName, row.UnitID, row.UnitName, row.UnitType,
				row.VanillaHitPoints, row.EffectiveHitPoints, row.HitPointDelta, row.CivSpan,
				row.Classification, row.NameTailOffset, row.HitPointOffset, row.NameOccurrences,
				row.Changed, row.Confidence)
		}
	}
	if len(report.Warnings) > 0 {
		fmt.Println("warnings:")
		for _, warning := range report.Warnings {
			fmt.Printf("- %s\n", warning)
		}
	}
}

func printReplayDataModCheck(report *replay.EffectiveDataModCheckReport) {
	fmt.Printf("path: %s\n", report.Path)
	if report.BaselineReplay != "" {
		fmt.Printf("baseline_replay: %s\n", report.BaselineReplay)
	}
	if report.DatPath != "" {
		fmt.Printf("dat: %s\n", report.DatPath)
	}
	fmt.Printf("method: %s\n", report.Method)
	fmt.Printf("verification: %s\n", report.Verification)
	fmt.Printf("verdict: %s\n", report.Verdict)
	if report.DataSet.Status != "" {
		fmt.Printf("data_set_identity: status=%s", report.DataSet.Status)
		if report.DataSet.ActiveDataSet != "" {
			fmt.Printf(" active=%q", report.DataSet.ActiveDataSet)
		}
		if report.DataSet.Checksum != 0 {
			fmt.Printf(" checksum=%d", report.DataSet.Checksum)
		}
		if report.DataSet.WorkshopID != 0 {
			fmt.Printf(" workshop_id=%d", report.DataSet.WorkshopID)
		}
		fmt.Printf(" confidence=%s source=%s", report.DataSet.Confidence, report.DataSet.Source)
		if report.DataSet.Error != "" {
			fmt.Printf(" error=%q", report.DataSet.Error)
		}
		fmt.Println()
	}
	fmt.Printf("target_template: player=%d label=%s bytes=%d sha256=%s\n",
		report.TargetTemplate.ReferencePlayer, report.TargetTemplate.ReferenceLabel, report.TargetTemplate.TailBytes, report.TargetTemplate.SHA256)
	if report.BaselineTemplate != nil {
		fmt.Printf("baseline_template: player=%d label=%s bytes=%d sha256=%s\n",
			report.BaselineTemplate.ReferencePlayer, report.BaselineTemplate.ReferenceLabel, report.BaselineTemplate.TailBytes, report.BaselineTemplate.SHA256)
	}
	fmt.Printf("template_check: status=%s baseline=%s expected=%s observed=%s confidence=%s\n",
		report.TemplateCheck.Status, report.TemplateCheck.BaselineName, report.TemplateCheck.ExpectedSHA256,
		report.TemplateCheck.ObservedSHA256, report.TemplateCheck.Confidence)
	if report.TemplateCheck.Note != "" {
		fmt.Printf("template_note: %s\n", report.TemplateCheck.Note)
	}
	fmt.Printf("availability_array: tail_offset=%d count=%d bytes=%d abs=%d..%d confidence=%s\n",
		report.AvailabilityArray.TailOffset, report.AvailabilityArray.Count, report.AvailabilityArray.Bytes,
		report.AvailabilityArray.AbsStart, report.AvailabilityArray.AbsEnd, report.AvailabilityArray.Confidence)
	if report.BaselineReplay != "" {
		fmt.Printf("tail_comparison: compared_players=%d equal=%d different=%d missing=%d compared_bytes=%d changed_bytes=%d diff_runs=%d\n",
			report.TailComparison.ComparedPlayers, report.TailComparison.EqualPlayers, report.TailComparison.DifferentPlayers,
			report.TailComparison.MissingPlayers, report.TailComparison.ComparedBytes, report.TailComparison.ChangedBytes,
			report.TailComparison.DiffRuns)
		for _, player := range report.TailComparison.Players {
			fmt.Printf("- player=%d label=%s equal=%t bytes=%d/%d identical=%.2f%% changed_bytes=%d diff_runs=%d sha=%s baseline_sha=%s\n",
				player.PlayerIndex, player.Label, player.Equal, player.TargetBytes, player.BaselineBytes,
				player.IdenticalRatio, player.ChangedBytes, player.DiffRuns, player.SHA256, player.BaselineSHA256)
		}
		fmt.Printf("availability_diff: compared=%t entries=%d changed=%d flips=%d target_available=%d baseline_available=%d shown=%d\n",
			report.AvailabilityDiff.Compared, report.AvailabilityDiff.ComparedEntries, report.AvailabilityDiff.ChangedEntries,
			report.AvailabilityDiff.AvailabilityFlips, report.AvailabilityDiff.TargetAvailable, report.AvailabilityDiff.BaselineAvailable,
			report.AvailabilityDiff.Shown)
		for _, entry := range report.AvailabilityDiff.Entries {
			name := entry.UnitName
			if name == "" {
				name = fmt.Sprintf("unit_slot_%d", entry.UnitSlot)
			}
			fmt.Printf("  - idx=%d unit_id=%d name=%q target=%d/%s baseline=%d/%s flip=%t confidence=%s\n",
				entry.Index, entry.UnitID, name, entry.TargetValue, entry.TargetMeaning, entry.BaselineValue,
				entry.BaselineMeaning, entry.AvailabilityFlip, entry.Confidence)
		}
	}
	if len(report.Warnings) > 0 {
		fmt.Println("warnings:")
		for _, warning := range report.Warnings {
			fmt.Printf("- %s\n", warning)
		}
	}
}

func clippedField(s string, width int) string {
	if width <= 0 || len(s) <= width {
		return s
	}
	if width <= 3 {
		return s[:width]
	}
	return s[:width-3] + "..."
}

func printReplayDiffState(report *replay.DiffStateReport) {
	fmt.Printf("before: %s\n", report.Before)
	fmt.Printf("after: %s\n", report.After)
	fmt.Printf("verification: %s (%s)\n", report.Verification.Label, report.Verification.Note)
	fmt.Printf("focus=%s trigger_graph=%s same=%v shown=%d changed_spans=%d\n",
		report.Summary.Focus, report.Summary.TriggerGraphCheck, report.Summary.SameTriggerGraph, report.Summary.Shown, report.Summary.ChangedSpanCount)
	fmt.Printf("header: before=%d after=%d changed=%d body: before=%d after=%d changed=%d opaque_changed=%d nonopaque_changed=%d\n",
		report.Summary.HeaderBytesBefore, report.Summary.HeaderBytesAfter, report.Summary.HeaderChangedBytes,
		report.Summary.BodyBytesBefore, report.Summary.BodyBytesAfter, report.Summary.BodyChangedBytes,
		report.Summary.ChangedOpaqueBytes, report.Summary.ChangedNonOpaqueBytes)
	if report.Sync != nil {
		fmt.Printf("sync: before_samples=%d after_samples=%d comparable_players=%d changed_players=%d changed_words=%d before_duration=%s after_duration=%s verification=%s\n",
			report.Sync.BeforeChecksumSamples, report.Sync.AfterChecksumSamples, report.Sync.ComparablePlayers,
			report.Sync.ChangedPlayers, report.Sync.ChangedWords, report.Sync.BeforeDuration, report.Sync.AfterDuration, report.Sync.Verification)
		for _, player := range report.Sync.PlayerDeltas {
			fmt.Printf("  - %s changed_words=%d object_count=%+d unit_type_sum=%+d object_id_sum=%+d position_sum=%+d state_sum_word3=%+d carry_sum_word4=%+d digest_word9=%+d resources=%+d\n",
				player.PlayerLabel, player.ChangedWords, player.ObjectCountDelta, player.UnitTypeSumDelta,
				player.ObjectIDSumDelta, player.PositionSumDelta, player.Word3Delta, player.Word4Delta,
				player.ScoreCandidateDelta, player.ResourceStockpileDelta)
		}
	}
	for _, span := range report.Spans {
		hints := ""
		if len(span.ShapeHints) > 0 {
			hints = " hints=" + strings.Join(span.ShapeHints, ",")
		}
		fmt.Printf("- %s %d..%d bytes=%d before=%q/%s after=%q/%s%s\n",
			span.Space, span.Start, span.End, span.Bytes, span.BeforeRegion, span.BeforeStatus, span.AfterRegion, span.AfterStatus, hints)
		if span.BeforeHex != "" || span.AfterHex != "" {
			fmt.Printf("  before_hex: %s\n", span.BeforeHex)
			fmt.Printf("  after_hex:  %s\n", span.AfterHex)
		}
	}
	if len(report.Warnings) > 0 {
		fmt.Println("warnings:")
		for _, warning := range report.Warnings {
			fmt.Printf("- %s\n", warning)
		}
	}
}

func printReplayValueScan(report *replay.ValueScanReport) {
	fmt.Printf("path: %s\n", report.Path)
	fmt.Printf("verification: %s (%s)\n", report.Verification.Label, report.Verification.Note)
	fmt.Printf("header=%d body=%d queries=%d hits=%d shown=%d\n",
		report.Summary.InflatedHeaderBytes, report.Summary.BodyBytes, report.Summary.QueryCount, report.Summary.HitCount, report.Summary.Shown)
	if len(report.Queries) > 0 {
		fmt.Println("queries:")
		for _, query := range report.Queries {
			fmt.Printf("- %s kind=%s bytes=%d pattern=%s\n", query.Label, query.Kind, query.Bytes, query.Pattern)
		}
	}
	if len(report.Hits) > 0 {
		fmt.Println("hits:")
		for _, hit := range report.Hits {
			fmt.Printf("- %s/%s %s %d..%d region=%q status=%s context=%s\n",
				hit.QueryLabel, hit.QueryKind, hit.Space, hit.Offset, hit.End, hit.Region, hit.Status, hit.ContextHex)
		}
	}
	if len(report.Warnings) > 0 {
		fmt.Println("warnings:")
		for _, warning := range report.Warnings {
			fmt.Printf("- %s\n", warning)
		}
	}
}

func printReplayHeaderAnchors(report *replay.HeaderAnchorsReport) {
	fmt.Printf("path: %s\n", report.Path)
	fmt.Printf("verification: %s (%s)\n", report.Verification.Label, report.Verification.Note)
	fmt.Printf("header=%d captions=%d shown_captions=%d resource_blocks=%d shown_resource_blocks=%d caption_stride=%d min=%d max=%d\n",
		report.Summary.InflatedHeaderBytes,
		report.Summary.CaptionAnchors,
		report.Summary.ShownCaptionAnchors,
		report.Summary.PlayerResourceBlocks,
		report.Summary.ShownPlayerResourceBlocks,
		report.Summary.CaptionMedianStrideBytes,
		report.Summary.CaptionMinStrideBytes,
		report.Summary.CaptionMaxStrideBytes)
	if len(report.Captions) > 0 {
		fmt.Println("captions:")
		for _, anchor := range report.Captions {
			text := anchor.Text
			if len(text) > 80 {
				text = text[:80]
			}
			fmt.Printf("- marker=%d text=%d end=%d bytes=%d len=%d lead_ff=%t confidence=%s text=%q\n",
				anchor.MarkerStart, anchor.TextStart, anchor.End, anchor.Bytes, anchor.Length, anchor.HasLeadFF, anchor.Confidence, text)
		}
	}
	if len(report.Resources) > 0 {
		fmt.Println("player_resource_blocks:")
		for _, block := range report.Resources {
			fmt.Printf("- %d..%d bytes=%d stride=%d players=%d confidence=%s\n", block.Start, block.End, block.Bytes, block.Stride, len(block.Players), block.Confidence)
			for _, player := range block.Players {
				fmt.Printf("  P%d record=%d..%d unknown=%d,%d gold=%d wood=%d food=%d stone=%d\n",
					player.Player, player.Start, player.End, player.Unknown0, player.Unknown1, player.Gold, player.Wood, player.Food, player.Stone)
			}
		}
	}
	if len(report.Warnings) > 0 {
		fmt.Println("warnings:")
		for _, warning := range report.Warnings {
			fmt.Printf("- %s\n", warning)
		}
	}
}

func printReplayObjects(report *replay.ObjectIndexReport) {
	fmt.Printf("path: %s\n", report.Path)
	fmt.Printf("method: %s\n", report.Method)
	fmt.Printf("verification: %s\n", report.Verification)
	fmt.Printf("summary: candidates=%d referenced=%d shown=%d command_refs=%d target_refs=%d ambiguous=%d unresolved_refs=%d decoded_prefix_bytes=%d decoded_body_prefix_bytes=%d bounded_body_bytes=%d opaque_body_bytes=%d\n",
		report.Summary.Candidates, report.Summary.ReferencedCandidates, report.Shown, report.Summary.CommandReferencedIDs, report.Summary.TargetReferencedIDs, report.Summary.AmbiguousCandidateIDs, report.Summary.UnresolvedReferencedIDs, report.Summary.DecodedPrefixBytes, report.Summary.DecodedBodyPrefixBytes, report.Summary.BoundedBodyBytes, report.Summary.OpaqueBodyBytes)
	if len(report.Players) > 0 {
		fmt.Println("players:")
		for _, player := range report.Players {
			fmt.Printf("- %s candidates=%d referenced=%d classes=%s\n", player.Label, player.Candidates, player.Referenced, stringIntMapText(player.ByClass))
		}
	}
	if len(report.Objects) > 0 {
		fmt.Println("objects:")
		for _, obj := range report.Objects {
			opaqueBody := obj.BodyBytes - obj.DecodedBodyPrefixBytes
			fmt.Printf("- id=%d owner=%s unit=%d class=%s record=%s hp=%.1f state=%d xy=%.2f,%.2f refs=%d target_refs=%d prefix=%d decoded_body_prefix=%d body=%d opaque_body=%d end=%s confidence=%s\n",
				obj.ObjectID, obj.OwnerLabel, obj.UnitID, obj.Class, obj.RecordTypeName, obj.HitPoints, obj.ObjectState, obj.X, obj.Y, obj.CommandRefs, obj.TargetRefs, obj.PrefixBytes, obj.DecodedBodyPrefixBytes, obj.BodyBytes, opaqueBody, obj.EndConfidence, obj.Confidence)
		}
	}
	if len(report.Warnings) > 0 {
		fmt.Println("warnings:")
		for _, warning := range report.Warnings {
			fmt.Printf("- %s\n", warning)
		}
	}
}

func printReplayObjectState(report *replay.ObjectStateReport) {
	fmt.Printf("path: %s\n", report.Path)
	fmt.Printf("method: %s\n", report.Method)
	fmt.Printf("verification: %s\n", report.Verification)
	fmt.Printf("summary: objects=%d referenced=%d shown=%d buildings=%d gates=%d damaged=%d under_attack=%d action=%d combat_tail=%d building_prefix=%d building_tail=%d building_tail_plausible=%d queues=%d gather_points=%d gate_locked_nonzero=%d v68_actions=%d v68_trailers=%d decoded_body_prefix_bytes=%d opaque_body_bytes=%d\n",
		report.Summary.Objects, report.Summary.Referenced, report.Shown, report.Summary.Buildings, report.Summary.Gates, report.Summary.Damaged, report.Summary.UnderAttack, report.Summary.ActionPrefixDecoded, report.Summary.CombatTailDecoded, report.Summary.BuildingPrefixDecoded, report.Summary.BuildingTailDecoded, report.Summary.BuildingTailPlausible, report.Summary.ProductionQueueHeaders, report.Summary.GatherPoints, report.Summary.GateLockedNonZero, report.Summary.V68ActionBlockObjects, report.Summary.V68TrailerObjects, report.Summary.DecodedBodyPrefixBytes, report.Summary.OpaqueBodyBytes)
	if len(report.Players) > 0 {
		fmt.Println("players:")
		for _, player := range report.Players {
			fmt.Printf("- %s candidates=%d referenced=%d classes=%s\n", player.Label, player.Candidates, player.Referenced, stringIntMapText(player.ByClass))
		}
	}
	if len(report.Objects) > 0 {
		fmt.Println("objects:")
		for _, obj := range report.Objects {
			fmt.Printf("- id=%d owner=%s unit=%s class=%s hp=%.1f state=%d xy=%.2f,%.2f refs=%d target_refs=%d selected_refs=%d coverage=%d/%d opaque=%d confidence=%s\n",
				obj.ObjectID, obj.OwnerLabel, replayUnitText(obj.UnitID, obj.UnitName), obj.Class, obj.HitPoints, obj.ObjectState, obj.X, obj.Y, obj.CommandRefs, obj.TargetRefs, obj.SelectedRefs, obj.Coverage.DecodedBodyPrefixBytes, obj.Coverage.BodyBytes, obj.Coverage.OpaqueBodyBytes, obj.Confidence)
			if obj.Action != nil {
				fmt.Printf("  action: waiting=%d command_flag=%d action_type=%d formation=%d/%d/%d attack_timer=%.2f attack_count=%d v68_blocks=%d\n",
					obj.Action.Waiting, obj.Action.CommandFlag, obj.Action.ActionType, obj.Action.FormationID, obj.Action.FormationRow, obj.Action.FormationCol, obj.Action.AttackTimer, obj.Action.AttackCount, obj.Action.V68ActionBlocks)
			}
			if obj.Combat != nil {
				fmt.Printf("  combat: has_ai=%d town_bell=%d target=%d target_xy=%.2f,%.2f action=%d berserker_timer=%.2f builders=%d healers=%d primary=%d secondary=%d kind=%d counter=%d position_tail=%v\n",
					obj.Combat.HasAI, obj.Combat.TownBellFlag, obj.Combat.TownBellTargetID, obj.Combat.TownBellTargetX, obj.Combat.TownBellTargetY, obj.Combat.TownBellAction, obj.Combat.BerserkerTimer, obj.Combat.NumBuilders, obj.Combat.NumHealers, obj.Combat.PrimaryObjectID, obj.Combat.SecondaryObjectID, obj.Combat.V68TrailerKind, obj.Combat.V68TrailerCounter, obj.Combat.PositionOrderTailDecoded)
			}
			if obj.Building != nil {
				fmt.Printf("  building: built=%d build_points=%.2f burning=%d gather_exists=%d gather_xy=%.2f,%.2f gather_obj=%d pending_order=%d queue_cap=%d tail_plausible=%v gate_locked=%d endpoint=%.2f,%.2f endpoint2=%.2f,%.2f first_update_raw=%d close_timer_raw=%d terrain=%d semi_asleep=%d snow=%d v68_kind=%d linked=%v\n",
					obj.Building.Built, obj.Building.BuildPoints, obj.Building.Burning, obj.Building.GatherPointExists, obj.Building.GatherPointX, obj.Building.GatherPointY, obj.Building.GatherPointObjectID, obj.Building.PendingOrder, obj.Building.ProductionQueueCapacity, obj.Building.BuildingTailPlausible, obj.Building.GateLocked, obj.Building.EndpointX, obj.Building.EndpointY, obj.Building.Endpoint2X, obj.Building.Endpoint2Y, obj.Building.FirstUpdateRaw, obj.Building.CloseTimerRaw, obj.Building.TerrainType, obj.Building.SemiAsleep, obj.Building.SnowFlag, obj.Building.V68TrailerKind, obj.Building.LinkedObjectIDs)
			}
			for _, note := range obj.Notes {
				fmt.Printf("  note: %s\n", note)
			}
		}
	}
	if len(report.Warnings) > 0 {
		fmt.Println("warnings:")
		for _, warning := range report.Warnings {
			fmt.Printf("- %s\n", warning)
		}
	}
}

func printReplayObjectShapes(report *replay.ObjectShapeReport) {
	fmt.Printf("path: %s\n", report.Path)
	fmt.Printf("method: %s\n", report.Method)
	fmt.Printf("verification: %s\n", report.Verification)
	fmt.Printf("summary: candidates=%d referenced=%d shapes=%d shown=%d decoded_prefix_bytes=%d decoded_body_prefix_bytes=%d bounded_body_bytes=%d opaque_body_bytes=%d record_bytes=%d..%d final_shapes=%d referenced_shapes=%d unreferenced_shapes=%d\n",
		report.Summary.Candidates, report.Summary.ReferencedCandidates, report.Summary.ShapeCount, report.Shown, report.Summary.DecodedPrefixBytes, report.Summary.DecodedBodyPrefixBytes, report.Summary.BoundedBodyBytes, report.Summary.OpaqueBodyBytes, report.Summary.MinRecordBytes, report.Summary.MaxRecordBytes, report.Summary.FinalBodyShapeCount, report.Summary.ReferencedShapeCount, report.Summary.UnreferencedShapeCount)
	if len(report.Shapes) > 0 {
		fmt.Println("shapes:")
		for _, shape := range report.Shapes {
			unit := replayUnitText(shape.UnitID, shape.UnitName)
			sample := ""
			if shape.BodySampleHex != "" {
				sample = " body_sample=" + shape.BodySampleHex
			}
			if shape.OpaqueSampleHex != "" {
				sample += " opaque_sample=" + shape.OpaqueSampleHex
			}
			fmt.Printf("- record=%s type=%d unit=%s class=%s count=%d referenced=%d target_refs=%d selected_refs=%d bytes=%d prefix=%d decoded_body_prefix_each=%d decoded_body_prefix_total=%d body=%d opaque_body_each=%d opaque_body_total=%d end=%s owners=%s examples=%v offsets=%v%s\n",
				shape.RecordTypeName, shape.RecordType, unit, shape.Class, shape.Count, shape.Referenced, shape.TargetRefs, shape.SelectedRefs, shape.RecordBytes, shape.PrefixBytes, shape.DecodedBodyPrefixEach, shape.DecodedBodyPrefixBytes, shape.BodyBytes, shape.OpaqueBodyEach, shape.OpaqueBodyBytes, shape.EndConfidence, intIntMapText(shape.Owners), shape.ExampleObjectIDs, shape.ExampleOffsets, sample)
		}
	}
	if len(report.Warnings) > 0 {
		fmt.Println("warnings:")
		for _, warning := range report.Warnings {
			fmt.Printf("- %s\n", warning)
		}
	}
}

func printReplaySpawns(report *replay.SpawnCatalogReport) {
	fmt.Printf("path: %s\n", report.Path)
	fmt.Printf("method: %s\n", report.Method)
	fmt.Printf("verification: %s\n", report.Verification)
	fmt.Printf("summary: create_effects=%d plausible_spawns=%d players=%d unit_types=%d\n", report.Summary.CreateObjectEffects, report.Summary.PlausibleSpawns, report.Summary.Players, report.Summary.UnitTypes)
	if len(report.Players) > 0 {
		fmt.Println("players:")
		for _, player := range report.Players {
			fmt.Printf("- %s spawns=%d units=%s\n", player.Label, player.Spawns, intIntMapText(player.Units))
		}
	}
	if len(report.Spawns) > 0 {
		fmt.Println("spawns:")
		limit := len(report.Spawns)
		if limit > 80 {
			limit = 80
		}
		for _, spawn := range report.Spawns[:limit] {
			name := ""
			if spawn.TriggerName != "" {
				name = " trigger=" + strconv.Quote(spawn.TriggerName)
			}
			fmt.Printf("- trigger=%d effect=%d player=%s unit=%s xy=%d,%d confidence=%s%s\n", spawn.TriggerIndex, spawn.EffectIndex, initialLabel(spawn.TargetPlayer), replayUnitText(spawn.UnitID, spawn.UnitName), spawn.X, spawn.Y, spawn.Confidence, name)
		}
		if len(report.Spawns) > limit {
			fmt.Printf("- ... %d more\n", len(report.Spawns)-limit)
		}
	}
	if len(report.Warnings) > 0 {
		fmt.Println("warnings:")
		for _, warning := range report.Warnings {
			fmt.Printf("- %s\n", warning)
		}
	}
}

func printReplayLifecycle(report *replay.LifecycleReport) {
	fmt.Printf("path: %s\n", report.Path)
	fmt.Printf("method: %s\n", report.Method)
	fmt.Printf("verification: %s\n", report.Verification)
	fmt.Printf("summary: first_seen_objects=%d spawn_recipes=%d spawn_matched_candidates=%d villager_candidates=%d players_with_villager=%d overmatched_near_villager_events=%d\n",
		report.Summary.FirstSeenObjects, report.Summary.SpawnRecipes, report.Summary.SpawnMatchedCandidates, report.Summary.VillagerCandidates, report.Summary.PlayersWithVillagerCandidate, report.Summary.OvermatchedNearVillagerEvents)
	if len(report.Players) > 0 {
		fmt.Println("players:")
		for _, player := range report.Players {
			first := "none"
			if player.FirstVillagerTime != "" {
				first = player.FirstVillagerTime
			}
			fmt.Printf("- %s first_villager=%s villager_candidates=%d spawn_matched_objects=%d overmatched_near_spawn=%d\n",
				player.Label, first, player.VillagerCandidates, player.SpawnMatchedObjects, player.OvermatchedNearSpawn)
		}
	}
	if len(report.Candidates) > 0 {
		fmt.Println("candidates:")
		limit := len(report.Candidates)
		if limit > 80 {
			limit = 80
		}
		for _, candidate := range report.Candidates[:limit] {
			unit := replayUnitText(candidate.UnitID, candidate.UnitName)
			if candidate.UnitID == 0 && len(candidate.CandidateUnitIDs) > 0 {
				unit = fmt.Sprintf("%d possible units", len(candidate.CandidateUnitIDs))
			}
			fmt.Printf("- %s %s object=%d kind=%s unit=%s event=%s action=%s selected=%d xy=%.2f,%.2f spawn=%d,%d dist=%.2f confidence=%s reason=%s\n",
				candidate.Time, candidate.PlayerLabel, candidate.ObjectID, candidate.Kind, unit, candidate.Type, candidate.ActionName, candidate.SelectedCount, candidate.X, candidate.Y, candidate.SpawnX, candidate.SpawnY, candidate.Distance, candidate.Confidence, candidate.Reason)
		}
		if len(report.Candidates) > limit {
			fmt.Printf("- ... %d more\n", len(report.Candidates)-limit)
		}
	}
	if len(report.Warnings) > 0 {
		fmt.Println("warnings:")
		for _, warning := range report.Warnings {
			fmt.Printf("- %s\n", warning)
		}
	}
}

func printReplayUnknownActions(report *replay.UnknownActionsReport) {
	fmt.Printf("path: %s\n", report.Path)
	fmt.Printf("method: %s\n", report.Method)
	fmt.Printf("verification: %s\n", report.Verification)
	fmt.Printf("summary: total_actions=%d unknown_total=%d unknown_ids=%d\n", report.TotalActions, report.UnknownTotal, len(report.Groups))
	for _, g := range report.Groups {
		fmt.Printf("- action_id=%d count=%d players=%v span=%s..%s payload_shapes=", g.ActionID, g.Count, g.Players, g.FirstTime, g.LastTime)
		for i, shape := range g.PayloadShapes {
			if i > 0 {
				fmt.Printf(",")
			}
			fmt.Printf("%dB=%d", shape.DeltaMS, shape.Count)
		}
		fmt.Println()
		for _, sample := range g.SampleHex {
			fmt.Printf("    hex %s\n", sample)
		}
	}
	for _, w := range report.Warnings {
		fmt.Printf("warning: %s\n", w)
	}
}

func printReplaySync(report *replay.SyncReport) {
	fmt.Printf("path: %s\n", report.Path)
	fmt.Printf("method: %s\n", report.Method)
	fmt.Printf("verification: %s\n", report.Verification)
	fmt.Printf("summary: syncs=%d delta_only=%d checksum_de=%d checksum_legacy=%d duration=%s delta_ms(min/mean/max)=%d/%.2f/%d\n",
		report.Summary.SyncCount, report.Summary.DeltaOnly, report.Summary.ChecksumDE, report.Summary.ChecksumLegacy,
		report.Summary.Duration, report.Summary.MinDeltaMS, report.Summary.MeanDeltaMS, report.Summary.MaxDeltaMS)
	if len(report.DeltaShapes) > 0 {
		fmt.Printf("delta_shapes:")
		for _, shape := range report.DeltaShapes {
			fmt.Printf(" %dms=%d", shape.DeltaMS, shape.Count)
		}
		fmt.Println()
	}
	for _, e := range report.Events {
		line := fmt.Sprintf("- %s delta=%dms form=%s offset=%d", e.Time, e.DeltaMS, e.Form, e.SourceOffset)
		if e.TrailerU32 != nil {
			line += fmt.Sprintf(" trailer=%d", *e.TrailerU32)
		}
		if len(e.Matrix) == 8 {
			line += fmt.Sprintf(" row0=%v", e.Matrix[0][:4])
		}
		fmt.Println(line)
	}
	if len(report.RawWords) > 0 {
		fmt.Println("raw_words:")
		limit := len(report.RawWords)
		if limit > 80 {
			limit = 80
		}
		for _, row := range report.RawWords[:limit] {
			fmt.Printf("- sample=%d time=%s player=P%d words=%s\n", row.SampleIndex, row.Time, row.PlayerID, formatChecksumWords(row.Words))
		}
		if len(report.RawWords) > limit {
			fmt.Printf("- ... %d more raw word rows\n", len(report.RawWords)-limit)
		}
	}
	if len(report.StateDeltas) > 0 {
		fmt.Println("state_deltas:")
		limit := len(report.StateDeltas)
		if limit > 80 {
			limit = 80
		}
		for _, d := range report.StateDeltas[:limit] {
			unit := ""
			if d.UnitID != 0 {
				unit = fmt.Sprintf(" unit=%d", d.UnitID)
				if d.UnitName != "" {
					unit += "/" + d.UnitName
				}
			}
			replacement := ""
			if d.ReplacementUnitID != 0 {
				replacement = fmt.Sprintf(" replacement=%d", d.ReplacementUnitID)
				if d.ReplacementUnitName != "" {
					replacement += "/" + d.ReplacementUnitName
				}
			}
			object := ""
			if d.ObjectID != 0 {
				object = fmt.Sprintf(" object_id=%d", d.ObjectID)
			}
			fmt.Printf("- %s->%s %s kind=%s count_delta=%d type_sum_delta=%d object_id_sum_delta=%d position_sum_delta=%d state_sum_word3_delta=%d carry_sum_word4_delta=%d digest_word9_delta=%d%s%s%s confidence=%s\n",
				d.FromTime, d.ToTime, d.PlayerLabel, d.Kind, d.ObjectCountDelta, d.UnitTypeSumDelta, d.ObjectIDSumDelta, d.PositionSumDelta, d.Word3Delta, d.Word4Delta, d.ScoreDelta, unit, replacement, object, d.Confidence)
		}
		if len(report.StateDeltas) > limit {
			fmt.Printf("- ... %d more\n", len(report.StateDeltas)-limit)
		}
	}
	for _, w := range report.Warnings {
		fmt.Printf("warning: %s\n", w)
	}
}

func printReplayChecksumPhase(report *replay.ChecksumPhaseReport) {
	fmt.Printf("path: %s\n", report.Path)
	fmt.Printf("method: %s\n", report.Method)
	fmt.Printf("verification: %s\n", report.Verification)
	fmt.Printf("summary: word=%d/%s checksum_samples=%d players=%d\n",
		report.Summary.WordIndex, report.Summary.WordName, report.Summary.Samples, report.Summary.Players)
	for _, player := range report.Players {
		fmt.Printf("- P%d samples=%d distinct=%d known_state_changes=%d transitions=%d dominant=%d share=%.2f repeats=%d ambiguous_from=%d\n",
			player.PlayerID,
			player.Samples,
			player.DistinctValues,
			player.KnownStateChanges,
			player.Transitions,
			player.DominantTransitions,
			player.DominantTransitionShare,
			player.RepeatTransitions,
			player.AmbiguousFromValues)
		fmt.Print("  values:")
		valueLimit := len(player.Values)
		if valueLimit > 12 {
			valueLimit = 12
		}
		for _, value := range player.Values[:valueLimit] {
			fmt.Printf(" %s:%d", formatChecksumWord(value.U32), value.Count)
		}
		if len(player.Values) > valueLimit {
			fmt.Printf(" ...+%d", len(player.Values)-valueLimit)
		}
		fmt.Println()
		if player.Ring != nil && player.Ring.Detected {
			fmt.Printf("  ring: detected length=%d forward_skips=%d reverse=%d off_ring=%d values=",
				player.Ring.Length, player.Ring.ForwardSkips, player.Ring.ReverseTransitions, player.Ring.OffRingTransitions)
			for i, value := range player.Ring.Values {
				if i > 0 {
					fmt.Print("->")
				}
				fmt.Print(formatChecksumWord(value))
			}
			if len(player.Ring.Values) > 0 {
				fmt.Printf("->%s", formatChecksumWord(player.Ring.Values[0]))
			}
			fmt.Println()
		}
		fmt.Print("  successors:")
		successorLimit := len(player.Successors)
		if successorLimit > 12 {
			successorLimit = 12
		}
		for _, successor := range player.Successors[:successorLimit] {
			marker := ""
			if successor.Dominant {
				marker = "*"
			}
			fmt.Printf(" %s->%s:%d%s", formatChecksumWord(successor.FromU32), formatChecksumWord(successor.ToU32), successor.Count, marker)
		}
		if len(player.Successors) > successorLimit {
			fmt.Printf(" ...+%d", len(player.Successors)-successorLimit)
		}
		fmt.Println()
	}
	for _, w := range report.Warnings {
		fmt.Printf("warning: %s\n", w)
	}
}

func printReplaySyncLog(report *replay.SyncLogReport) {
	fmt.Printf("path: %s\n", report.Path)
	fmt.Printf("method: %s\n", report.Method)
	fmt.Printf("verification: %s\n", report.Verification)
	fmt.Printf("summary: lines=%d turns=%d world=%s->%s duration=%s player_blocks=%d attr_blocks=%d attr_values=%d attr_indices=%v objects=%d declared=%d rng_events=%d set_seed=%d rand=%d update_player=%d\n",
		report.Summary.Lines,
		report.Summary.Turns,
		replay.FormatTime(report.Summary.FirstWorldTimeMS),
		replay.FormatTime(report.Summary.LastWorldTimeMS),
		report.Summary.Duration,
		report.Summary.PlayerBlocks,
		report.Summary.AttributeBlocks,
		report.Summary.AttributeValues,
		report.Summary.AttributeIndices,
		report.Summary.ObjectRows,
		report.Summary.DeclaredObjects,
		report.Summary.RNGEvents,
		report.Summary.RNGSetSeeds,
		report.Summary.RNGRands,
		report.Summary.UpdatePlayerLines)
	if report.Comparison != nil {
		c := report.Comparison
		fmt.Printf("replay_compare: samples=%d replay=%s->%s log=%s->%s overlap=%t overlapping_samples=%d exact_matches=%d\n",
			c.ChecksumSamples,
			replay.FormatTime(c.ReplayFirstTimeMS),
			replay.FormatTime(c.ReplayLastTimeMS),
			replay.FormatTime(c.LogFirstWorldTimeMS),
			replay.FormatTime(c.LogLastWorldTimeMS),
			c.Overlap,
			c.OverlappingSamples,
			len(c.ExactWorldTimeMatches))
		if c.NearestBeforeLogStartMS != 0 {
			fmt.Printf("  nearest_before_log_start=%s\n", c.NearestBeforeLogStart)
		}
		if c.NearestAfterLogEndMS != 0 {
			fmt.Printf("  nearest_after_log_end=%s\n", c.NearestAfterLogEnd)
		}
		if c.Warning != "" {
			fmt.Printf("  warning: %s\n", c.Warning)
		}
		if c.NearestBeforeToFirstLog != nil {
			a := c.NearestBeforeToFirstLog
			fmt.Printf("  rough_nearest_before_to_first_log: replay=%s log=%s gap=%s confidence=%s\n",
				a.ReplayTime, a.LogWorldTime, a.Gap, a.Confidence)
			for _, p := range a.Players {
				fmt.Printf("    P%d word1_attr0_3 log=%.3f replay=%d diff=%.3f word6 log=%d replay=%d diff=%d word2 log=%d replay=%d diff=%d word3_state log=%d replay=%d diff=%d word4_carry log=%d replay=%d diff=%d per_object=%.2f candidates=%s word7_xy100 log=%d replay=%d diff=%d word10 log=%d replay=%d diff=%d\n",
					p.PlayerID,
					p.LogResourceSum, p.ReplayWord1, p.ResourceSumDiff,
					p.LogObjectCount, p.ReplayWord6, p.ObjectCountDiff,
					p.LogUnitTypeSum, p.ReplayWord2, p.UnitTypeSumDiff,
					p.LogStateSum, p.ReplayWord3, p.StateSumDiff,
					p.Checksum.CarrySum, p.ReplayWord4, p.Checksum.CarrySum-int64(p.ReplayWord4), p.Word4PerObject,
					formatSyncLogWord4Candidates(p.Word4Candidates),
					p.LogPositionXY100, p.ReplayWord7, p.PositionXY100Diff,
					p.LogObjectIDSum, p.ReplayWord10, p.ObjectIDSumDiff)
			}
		}
		for _, match := range c.ExactWorldTimeMatches {
			fmt.Printf("  exact %s:\n", match.Time)
			for _, p := range match.Players {
				fmt.Printf("    P%d word1_attr0_3 log=%.3f replay=%d diff=%.3f word6 log=%d replay=%d diff=%d word2 log=%d replay=%d diff=%d word3_state log=%d replay=%d diff=%d word4_carry log=%d replay=%d diff=%d per_object=%.2f candidates=%s word7_xy100 log=%d replay=%d diff=%d word10 log=%d replay=%d diff=%d\n",
					p.PlayerID,
					p.LogResourceSum, p.ReplayWord1, p.ResourceSumDiff,
					p.LogObjectCount, p.ReplayWord6, p.ObjectCountDiff,
					p.LogUnitTypeSum, p.ReplayWord2, p.UnitTypeSumDiff,
					p.LogStateSum, p.ReplayWord3, p.StateSumDiff,
					p.Checksum.CarrySum, p.ReplayWord4, p.Checksum.CarrySum-int64(p.ReplayWord4), p.Word4PerObject,
					formatSyncLogWord4Candidates(p.Word4Candidates),
					p.LogPositionXY100, p.ReplayWord7, p.PositionXY100Diff,
					p.LogObjectIDSum, p.ReplayWord10, p.ObjectIDSumDiff)
			}
		}
	}
	if len(report.RNGStreams) > 0 {
		fmt.Println("rng_streams:")
		for _, stream := range report.RNGStreams {
			fmt.Printf("- %s events=%d set_seed=%d rand=%d", stream.Stream, stream.Events, stream.SetSeeds, stream.Rands)
			if len(stream.Sites) > 0 {
				fmt.Print(" sites:")
				for _, site := range stream.Sites {
					fmt.Printf(" %s@%d=%d", strings.ReplaceAll(site.Operation, " ", "_"), site.Line, site.Count)
				}
			}
			fmt.Println()
		}
	}
	if len(report.ObjectCensus) > 0 {
		fmt.Println("object_census:")
		for _, row := range report.ObjectCensus {
			fmt.Printf("- %s dbid=%d count=%d\n", row.Name, row.DBID, row.Count)
		}
	}
	if len(report.StateCensus) > 0 {
		fmt.Println("state_census:")
		for _, row := range report.StateCensus {
			fmt.Printf("- state=%d rows=%d hp_zero=%d carry_sum=%d names=%s\n",
				row.State, row.Rows, row.HPZeroRows, row.CarrySum, strings.Join(row.Names, ","))
		}
	}
	if len(report.CorpseCarry) > 0 {
		fmt.Println("corpse_carry:")
		for _, row := range report.CorpseCarry {
			fmt.Printf("- P%d object=%d unit=%d/%s samples=%d %s:%d -> %s:%d drop=%d rate=%.3f/s\n",
				row.PlayerID, row.ObjectID, row.UnitID, row.UnitName, row.Samples,
				row.FirstTime, row.FirstCarry, row.LastTime, row.LastCarry, row.CarryDrop, row.ObservedRatePS)
		}
	}
	if len(report.CorpseDecay) > 0 {
		fmt.Println("corpse_decay_baselines:")
		for _, row := range report.CorpseDecay {
			lifetime := ""
			if row.EstimatedLifetime != "" {
				lifetime = " lifetime~" + row.EstimatedLifetime
			}
			fmt.Printf("- unit=%d/%s tracks=%d moving=%d samples=%d carry=%d..%d -> %d..%d drop=%d duration=%s rate=%.3f/s range=%.3f..%.3f/s%s confidence=%s\n",
				row.UnitID, row.UnitName, row.Tracks, row.MovingTracks, row.Samples,
				row.FirstCarryMin, row.FirstCarryMax, row.LastCarryMin, row.LastCarryMax,
				row.CarryDropTotal, replay.FormatTime(row.DurationMSTotal), row.ObservedRatePS,
				row.ObservedRateMinPS, row.ObservedRateMaxPS,
				lifetime, row.Confidence)
		}
	}
	if len(report.Turns) > 0 {
		fmt.Println("turns:")
		turnLimit := len(report.Turns)
		if turnLimit > 4 {
			turnLimit = 4
		}
		for _, turn := range report.Turns[:turnLimit] {
			fmt.Printf("- turn=%d world=%s pturns=%v update_players=%v players=%d\n", turn.Turn, turn.WorldTime, turn.Pturns, turn.UpdatePlayers, len(turn.Players))
			for _, player := range turn.Players {
				c := player.Checksum
				fmt.Printf("  P%d %s attrs_printed=%d/%d word1_attr0_3=%.3f objects=%d declared=%d word2_type_sum=%d word6_count=%d word10_id_sum=%d pos_xy100=%d retarget_sum=%d word3_state=%d word4_carry=%d action_state=%d hp100=%d\n",
					player.PlayerID,
					player.Kind,
					len(player.Attributes),
					player.DeclaredAttrs,
					c.ResourceStockpileSum,
					player.ObjectRows,
					player.DeclaredObjects,
					c.UnitTypeSum,
					c.ObjectCount,
					c.ObjectIDSum,
					c.PositionXY100,
					c.RetargetTimerSum,
					c.StateSum,
					c.CarrySum,
					c.ActionStateSum,
					c.HP100Sum)
			}
		}
		if len(report.Turns) > turnLimit {
			fmt.Printf("- ... %d more turns\n", len(report.Turns)-turnLimit)
		}
	}
	for _, w := range report.Warnings {
		fmt.Printf("warning: %s\n", w)
	}
}

func formatSyncLogWord4Candidates(candidates []replay.SyncLogWord4Candidate) string {
	if len(candidates) == 0 {
		return "[]"
	}
	parts := make([]string, len(candidates))
	for i, candidate := range candidates {
		parts[i] = fmt.Sprintf("%s:%d/%d(diff=%d)", candidate.Name, candidate.LogValue, candidate.ReplayWord4, candidate.Diff)
	}
	return "[" + strings.Join(parts, " ") + "]"
}

func printReplayChecksumPhaseAll(report *replay.ChecksumPhaseAllReport) {
	fmt.Printf("path: %s\n", report.Path)
	fmt.Printf("method: %s\n", report.Method)
	fmt.Printf("verification: %s\n", report.Verification)
	for _, word := range report.Words {
		fmt.Printf("\nword=%d/%s checksum_samples=%d players=%d\n",
			word.Summary.WordIndex, word.Summary.WordName, word.Summary.Samples, word.Summary.Players)
		for _, player := range word.Players {
			ring := ""
			if player.Ring != nil && player.Ring.Detected {
				ring = fmt.Sprintf(" ring_length=%d forward_skips=%d", player.Ring.Length, player.Ring.ForwardSkips)
			}
			fmt.Printf("- P%d samples=%d distinct=%d known_state_changes=%d transitions=%d dominant_share=%.2f repeats=%d ambiguous_from=%d%s\n",
				player.PlayerID,
				player.Samples,
				player.DistinctValues,
				player.KnownStateChanges,
				player.Transitions,
				player.DominantTransitionShare,
				player.RepeatTransitions,
				player.AmbiguousFromValues,
				ring)
		}
	}
}

func formatChecksumWords(words []uint32) string {
	parts := make([]string, len(words))
	for i, word := range words {
		parts[i] = formatChecksumWord(word)
	}
	return "[" + strings.Join(parts, " ") + "]"
}

func formatChecksumWord(word uint32) string {
	signed := int32(word)
	if signed < 0 {
		return fmt.Sprintf("%d/%d", word, signed)
	}
	return fmt.Sprintf("%d", word)
}

func printReplayPlayerSeries(report *replay.PlayerSeriesReport) {
	fmt.Printf("path: %s\n", report.Path)
	fmt.Printf("method: %s\n", report.Method)
	fmt.Printf("verification: %s\n", report.Verification)
	fmt.Printf("summary: checksum_samples=%d players=%d emitted_samples=%d emitted_deltas=%d duration=%s\n",
		report.Summary.ChecksumSamples, report.Summary.Players, report.Summary.EmittedSamples, report.Summary.EmittedDeltas, report.Summary.Duration)
	for _, player := range report.Players {
		name := ""
		if player.PlayerName != "" {
			name = " " + player.PlayerName
		}
		fmt.Printf("- %s%s samples=%d final_objects=%d produced_est=%d removed_est=%d death_replacements=%d single_adds=%d single_removes=%d multi_add_events=%d multi_remove_events=%d final_resource=%d final_type_sum=%d confidence=%s\n",
			player.PlayerLabel, name, player.Samples, player.FinalObjectCount, player.ProducedEstimate, player.LostEstimate, player.DeathReplacements,
			player.SingleObjectAdds, player.SingleObjectRemoves, player.MultiObjectAddEvents, player.MultiObjectRemoveEvents,
			player.FinalResourceStockpile, player.FinalUnitTypeSum, player.AttributionConfidence)
		if player.FilteredArtifactAdds > 0 || player.FilteredArtifactRemoves > 0 {
			fmt.Printf("  filtered_artifacts: adds=%d removes=%d\n", player.FilteredArtifactAdds, player.FilteredArtifactRemoves)
		}
		if len(player.AddedUnitTypes) > 0 {
			fmt.Printf("  added_units:")
			for _, unit := range player.AddedUnitTypes {
				fmt.Printf(" %d/%s=%d", unit.UnitID, clippedField(unit.UnitName, 24), unit.Count)
			}
			fmt.Println()
		}
		if len(player.RemovedUnitTypes) > 0 {
			fmt.Printf("  removed_units:")
			for _, unit := range player.RemovedUnitTypes {
				fmt.Printf(" %d/%s=%d", unit.UnitID, clippedField(unit.UnitName, 24), unit.Count)
			}
			fmt.Println()
		}
		if len(player.DeathReplacementUnits) > 0 {
			fmt.Printf("  death_replacement_units:")
			for _, unit := range player.DeathReplacementUnits {
				fmt.Printf(" %d/%s=%d", unit.UnitID, clippedField(unit.UnitName, 24), unit.Count)
			}
			fmt.Println()
		}
		for _, sample := range player.SamplesOut {
			fmt.Printf("  %s objects=%d unit_sum=%d resource=%d state_sum_word3=%d carry_sum_word4=%d pos_sum=%d object_id_sum=%d digest_word9=%d\n",
				sample.Time, sample.ObjectCount, sample.UnitTypeSum, sample.ResourceStockpile, sample.ForceMetric,
				sample.Word4, sample.PositionSum, sample.ObjectIDSum, sample.ScoreCandidate)
		}
	}
	if len(report.Deltas) > 0 {
		fmt.Println("deltas:")
		limit := len(report.Deltas)
		if limit > 80 {
			limit = 80
		}
		for _, d := range report.Deltas[:limit] {
			unit := ""
			if d.UnitID != 0 {
				unit = fmt.Sprintf(" unit=%d", d.UnitID)
				if d.UnitName != "" {
					unit += "/" + d.UnitName
				}
			}
			replacement := ""
			if d.ReplacementUnitID != 0 {
				replacement = fmt.Sprintf(" replacement=%d", d.ReplacementUnitID)
				if d.ReplacementUnitName != "" {
					replacement += "/" + d.ReplacementUnitName
				}
			}
			object := ""
			if d.ObjectID != 0 {
				object = fmt.Sprintf(" object_id=%d", d.ObjectID)
			}
			fmt.Printf("- %s->%s %s kind=%s count_delta=%d type_sum_delta=%d object_id_sum_delta=%d state_sum_word3_delta=%d carry_sum_word4_delta=%d digest_word9_delta=%d%s%s%s confidence=%s\n",
				d.FromTime, d.ToTime, d.PlayerLabel, d.Kind, d.ObjectCountDelta, d.UnitTypeSumDelta, d.ObjectIDSumDelta, d.Word3Delta, d.Word4Delta, d.ScoreDelta, unit, replacement, object, d.Confidence)
		}
		if len(report.Deltas) > limit {
			fmt.Printf("- ... %d more\n", len(report.Deltas)-limit)
		}
	}
	for _, w := range report.Warnings {
		fmt.Printf("warning: %s\n", w)
	}
}

func printReplayPlaytest(report *replay.PlaytestReport) {
	fmt.Printf("Playtest report: %s\n", report.Session.ScenarioName)
	fmt.Printf("replay: duration=%s humans=%d computers=%d data_set=%s\n",
		report.Session.Duration, report.Session.HumanCount, report.Session.ComputerCount, formatDataSetIdentity(report.Session.DataSet))
	if report.Session.TriggerGraphSHA256 != "" {
		fmt.Printf("scenario build fingerprint: %s\n", shortHash(report.Session.TriggerGraphSHA256))
	} else {
		fmt.Println("scenario build fingerprint: unavailable")
	}
	if len(report.Session.Players) > 0 {
		fmt.Println("players:")
		for _, player := range report.Session.Players {
			kind := player.Kind
			if kind == "" {
				if player.Human {
					kind = "human"
				} else {
					kind = "computer"
				}
			}
			fmt.Printf("- %s kind=%s\n", playtestCLIPlayerLabel(player), kind)
		}
	}
	fmt.Printf("\nfeedback moments: %d", report.Summary.FeedbackMoments)
	if report.Summary.BacklogRowsExcluded > 0 {
		fmt.Printf(" (%d injected backlog chat row(s) excluded)", report.Summary.BacklogRowsExcluded)
	}
	fmt.Println()
	if len(report.Moments) == 0 {
		fmt.Println("No live chat, taunt, or flare feedback was found in this replay.")
	} else {
		for _, moment := range report.Moments {
			fmt.Println()
			fmt.Printf("%s - %s", moment.Time, strings.ToUpper(moment.Type))
			if moment.PlayerID > 0 {
				name := moment.PlayerName
				if name == "" {
					name = "Unknown"
				}
				fmt.Printf(" from P%d %s", moment.PlayerID, name)
			}
			if moment.Text != "" {
				fmt.Printf(": %q", moment.Text)
			}
			if moment.RecordedCount > 1 {
				fmt.Printf(" (recorded %dx", moment.RecordedCount)
				if moment.LastTime != "" && moment.LastTime != moment.Time {
					fmt.Printf(", last %s", moment.LastTime)
				}
				fmt.Print(")")
			}
			if moment.Type == "flare" {
				fmt.Printf(" at %.2f,%.2f", moment.X, moment.Y)
				if moment.Region != "" {
					fmt.Printf(" (%s)", moment.Region)
				}
			}
			fmt.Println()
			if len(moment.ChangedPlayers) > 0 || len(moment.UnchangedPlayers) > 0 {
				fmt.Printf("split: changed=%s unchanged=%s\n", playtestCLIList(moment.ChangedPlayers), playtestCLIList(moment.UnchangedPlayers))
			}
			if len(moment.Window.Intervals) == 0 {
				fmt.Println("state window: no checksum samples available around this moment")
				continue
			}
			fmt.Printf("state window: %s to %s (%d sample(s) before, %d after)\n",
				moment.Window.StartTime, moment.Window.EndTime, moment.Window.SamplesBefore, moment.Window.SamplesAfter)
			for i, interval := range moment.Window.Intervals {
				changed, unchanged := playtestCLIIntervalSplit(interval)
				focus := ""
				if i == moment.Window.FocusIntervalIndex {
					focus = " nearest-before-feedback"
				}
				fmt.Printf("- %s -> %s%s changed=%s unchanged=%s\n", interval.FromTime, interval.ToTime, focus, playtestCLIList(changed), playtestCLIList(unchanged))
				for _, change := range interval.Players {
					fmt.Printf("  - %s: %s\n", playtestCLIStatePlayerLabel(change), playtestCLIChangeText(change))
				}
			}
		}
	}
	fmt.Println()
	fmt.Println("honesty:")
	for _, note := range report.Honesty {
		fmt.Printf("- %s\n", note)
	}
	if len(report.Warnings) > 0 {
		fmt.Println("warnings:")
		for _, warning := range report.Warnings {
			fmt.Printf("- %s\n", warning)
		}
	}
}

func playtestCLIPlayerLabel(player replay.PlaytestPlayer) string {
	if player.Name == "" {
		return player.Label
	}
	return player.Label + " " + player.Name
}

func playtestCLIStatePlayerLabel(change replay.PlaytestStateChange) string {
	if change.PlayerName == "" {
		return change.PlayerLabel
	}
	return change.PlayerLabel + " " + change.PlayerName
}

func playtestCLIChangeText(change replay.PlaytestStateChange) string {
	if !change.Changed {
		return "no object/type change"
	}
	var parts []string
	if change.ObjectCountDelta != 0 {
		parts = append(parts, fmt.Sprintf("objects %+d", change.ObjectCountDelta))
	}
	if change.UnitTypeSumDelta != 0 {
		parts = append(parts, fmt.Sprintf("unit-type sum %+d", change.UnitTypeSumDelta))
	}
	if change.UnitName != "" {
		parts = append(parts, clippedField(change.UnitName, 28))
	} else if change.UnitID != 0 {
		parts = append(parts, fmt.Sprintf("unit %d", change.UnitID))
	}
	if len(parts) == 0 {
		parts = append(parts, "state changed")
	}
	return change.Kind + " (" + strings.Join(parts, ", ") + ")"
}

func playtestCLIList(values []string) string {
	if len(values) == 0 {
		return "none"
	}
	return strings.Join(values, ", ")
}

func playtestCLIIntervalSplit(interval replay.PlaytestStateInterval) ([]string, []string) {
	var changed, unchanged []string
	for _, player := range interval.Players {
		label := playtestCLIStatePlayerLabel(player)
		if player.Changed {
			changed = append(changed, label)
		} else {
			unchanged = append(unchanged, label)
		}
	}
	return changed, unchanged
}

func printReplayChecksumProbe(report *replay.ChecksumProbeReport) {
	fmt.Printf("path: %s\n", report.Path)
	fmt.Printf("method: %s\n", report.Method)
	fmt.Printf("verification: %s\n", report.Verification)
	fmt.Printf("summary: checksum_samples=%d state_deltas=%d probes=%d matched=%d duration=%s",
		report.Summary.ChecksumSamples, report.Summary.StateDeltas, report.Summary.ProbeCount,
		report.Summary.MatchedProbes, report.Summary.Duration)
	if report.Summary.Preset != "" {
		fmt.Printf(" preset=%s", report.Summary.Preset)
	}
	fmt.Println()
	for _, probe := range report.Probes {
		player := "any"
		if probe.PlayerID > 0 {
			player = fmt.Sprintf("P%d", probe.PlayerID)
		}
		fmt.Printf("- %s player=%s result=%s matches=%d", probe.Name, player, probe.Result, probe.MatchCount)
		if probe.ObjectCountDelta != nil {
			fmt.Printf(" count_delta=%d", *probe.ObjectCountDelta)
		}
		if probe.UnitTypeSumDelta != nil {
			fmt.Printf(" type_sum_delta=%d", *probe.UnitTypeSumDelta)
		}
		if probe.ObjectIDSumDelta != nil {
			fmt.Printf(" object_id_sum_delta=%d", *probe.ObjectIDSumDelta)
		}
		if probe.Word3Delta != nil {
			fmt.Printf(" state_sum_word3_delta=%d", *probe.Word3Delta)
		}
		if probe.Word4Delta != nil {
			fmt.Printf(" carry_sum_word4_delta=%d", *probe.Word4Delta)
		}
		if probe.ScoreDelta != nil {
			fmt.Printf(" digest_word9_delta=%d", *probe.ScoreDelta)
		}
		if probe.Description != "" {
			fmt.Printf(" -- %s", probe.Description)
		}
		fmt.Println()
		for _, c := range probe.Candidates {
			unit := ""
			if c.UnitID != 0 {
				unit = fmt.Sprintf(" unit=%d", c.UnitID)
				if c.UnitName != "" {
					unit += "/" + clippedField(c.UnitName, 24)
				}
			}
			object := ""
			if c.ObjectID != 0 {
				object = fmt.Sprintf(" object_id=%d", c.ObjectID)
			}
			fmt.Printf("  %s->%s %s kind=%s count_delta=%d type_sum_delta=%d object_id_sum_delta=%d position_sum_delta=%d state_sum_word3_delta=%d carry_sum_word4_delta=%d digest_word9_delta=%d%s%s confidence=%s\n",
				c.FromTime, c.ToTime, c.PlayerLabel, c.Kind, c.ObjectCountDelta, c.UnitTypeSumDelta,
				c.ObjectIDSumDelta, c.PositionSumDelta, c.Word3Delta, c.Word4Delta, c.ScoreDelta, unit, object, c.Confidence)
		}
		if probe.MatchCount > len(probe.Candidates) {
			fmt.Printf("  ... %d more matches\n", probe.MatchCount-len(probe.Candidates))
		}
	}
	for _, w := range report.Warnings {
		fmt.Printf("warning: %s\n", w)
	}
}

func printReplayCombat(report *replay.CombatReport) {
	fmt.Printf("path: %s\n", report.Path)
	fmt.Printf("method: %s\n", report.Method)
	fmt.Printf("verification: %s\n", report.Verification)
	fmt.Printf("summary: checksum_samples=%d state_deltas=%d death_replacements=%d shown_deaths=%d loss_pulses=%d shown=%d net_objects_lost=%d single_removals=%d filtered_artifacts=%d pressure_links=%d targets=%d window_ms=%d sort=%s\n",
		report.Summary.ChecksumSamples, report.Summary.StateDeltas, report.Summary.DeathReplacements, report.Summary.ShownDeathReplacements,
		report.Summary.LossPulses, report.Summary.ShownLossPulses,
		report.Summary.NetObjectsLost, report.Summary.SingleObjectRemovals, report.Summary.FilteredArtifacts, report.Summary.PressureLinks, report.Summary.Targets,
		report.Summary.WindowMS, report.Summary.Sort)
	if len(report.Players) > 0 {
		fmt.Println("players:")
		for _, player := range report.Players {
			name := combatNameText(player.PlayerID, player.PlayerName)
			fmt.Printf("- %s death_replacements=%d losses=%d pulses=%d largest=%d pressure_links=%d",
				name, player.DeathReplacements, player.NetObjectsLost, player.LossPulses, player.LargestNetLoss, player.PressureLinksAgainst)
			if player.FirstDeathTime != "" {
				fmt.Printf(" first_death=%s", player.FirstDeathTime)
			}
			if player.FirstLossTime != "" {
				fmt.Printf(" first_loss=%s", player.FirstLossTime)
			}
			if len(player.PressureByAttacker) > 0 {
				fmt.Printf(" pressure_by=")
				for i, attacker := range player.PressureByAttacker {
					if i > 0 {
						fmt.Printf(",")
					}
					fmt.Printf("%s:%d", combatNameText(attacker.PlayerID, attacker.PlayerName), attacker.Events)
				}
			}
			fmt.Println()
		}
	}
	if len(report.Targets) > 0 {
		fmt.Println("targets:")
		targetLimit := len(report.Targets)
		if targetLimit > 25 {
			targetLimit = 25
		}
		for _, target := range report.Targets[:targetLimit] {
			fmt.Printf("- target=%d owner=%s unit=%d/%s class=%s xy=%.1f,%.1f pressure=%d first=%s last=%s nearby_loss_pulses=%d net_lost_nearby=%d confidence=%s",
				target.TargetObjectID, combatNameText(target.TargetOwnerID, target.TargetOwnerName), target.TargetUnitID,
				target.TargetUnitName, target.TargetClass, target.TargetX, target.TargetY, target.PressureEvents,
				target.FirstPressureTime, target.LastPressureTime, target.NearbyLossPulses, target.NetObjectsLostNearby, target.Confidence)
			if len(target.PressureByAttacker) > 0 {
				fmt.Printf(" pressure_by=")
				for i, attacker := range target.PressureByAttacker {
					if i > 0 {
						fmt.Printf(",")
					}
					fmt.Printf("%s:%d", combatNameText(attacker.PlayerID, attacker.PlayerName), attacker.Events)
				}
			}
			fmt.Println()
		}
		if len(report.Targets) > targetLimit {
			fmt.Printf("- ... %d more targets\n", len(report.Targets)-targetLimit)
		}
	}
	if len(report.DeathEvents) > 0 {
		fmt.Println("death_events:")
		for _, event := range report.DeathEvents {
			fmt.Printf("- %s->%s victim=%s live=%d/%s corpse=%d/%s type_sum_delta=%d object_id_sum_delta=%d state_sum_word3_delta=%d carry_sum_word4_delta=%d confidence=%s\n",
				event.FromTime, event.ToTime, combatNameText(event.PlayerID, event.PlayerName),
				event.LiveUnitID, event.LiveUnitName, event.CorpseUnitID, event.CorpseUnitName,
				event.UnitTypeSumDelta, event.ObjectIDSumDelta, event.Word3Delta, event.Word4Delta, event.Confidence)
		}
	}
	if len(report.LossPulses) > 0 {
		fmt.Println("loss_pulses:")
		for _, pulse := range report.LossPulses {
			fmt.Printf("- %s->%s victim=%s net_lost=%d type_sum_delta=%d object_id_sum_delta=%d pressure=%d confidence=%s\n",
				pulse.FromTime, pulse.ToTime, combatNameText(pulse.PlayerID, pulse.PlayerName), pulse.NetObjectsLost,
				pulse.UnitTypeSumDelta, pulse.ObjectIDSumDelta, pulse.CandidatePressureNum, pulse.Confidence)
			if pulse.SingleObjectRemoval {
				fmt.Printf("  removed: unit=%d/%s object_id=%d\n", pulse.RemovedUnitID, pulse.RemovedUnitName, pulse.RemovedObjectID)
			}
			for _, pressure := range pulse.CandidatePressure {
				fmt.Printf("  pressure: %s attacker=%s type=%s target=%d unit=%d/%s class=%s selected=%d xy=%.1f,%.1f confidence=%s\n",
					pressure.Time, combatNameText(pressure.PlayerID, pressure.PlayerName), pressure.Type,
					pressure.TargetObjectID, pressure.TargetUnitID, pressure.TargetUnitName, pressure.TargetClass,
					pressure.SelectedObjects, pressure.TargetX, pressure.TargetY, pressure.Confidence)
			}
		}
	}
	for _, w := range report.Warnings {
		fmt.Printf("warning: %s\n", w)
	}
}

func printReplayDeaths(report *replay.DeathReport) {
	fmt.Printf("path: %s\n", report.Path)
	fmt.Printf("method: %s\n", report.Method)
	fmt.Printf("verification: %s\n", report.Verification)
	fmt.Printf("summary: checksum_samples=%d replacements=%d shown=%d ambiguous_flat_transforms=%d shown_ambiguous=%d\n",
		report.Summary.ChecksumSamples, report.Summary.ReplacementEvents, report.Summary.ShownReplacementEvents,
		report.Summary.AmbiguousFlatTransforms, report.Summary.ShownAmbiguous)
	fmt.Printf("corpse_table: rows=%d engine_verified=%d heuristic=%d method=%s\n",
		report.CorpseTable.Rows, report.CorpseTable.EngineVerified, report.CorpseTable.HeuristicRows, report.CorpseTable.Method)
	if len(report.Events) > 0 {
		fmt.Println("replacement_events:")
		for _, event := range report.Events {
			timing := ""
			if event.EstimatedDeath != "" {
				timing = fmt.Sprintf(" estimated_death=%s method=%s fresh_carry=%d", event.EstimatedDeath, event.DeathTimeMethod, event.FreshCorpseCarry)
			}
			fmt.Printf("- %s->%s %s live=%d/%s corpse=%d/%s type_sum_delta=%d object_id_sum_delta=%d state_sum_word3_delta=%d carry_sum_word4_delta=%d%s confidence=%s\n",
				event.FromTime, event.ToTime, event.PlayerLabel, event.LiveUnitID, event.LiveUnitName,
				event.CorpseUnitID, event.CorpseUnitName, event.UnitTypeSumDelta, event.ObjectIDSumDelta,
				event.Word3Delta, event.Word4Delta, timing, event.Confidence)
		}
	}
	if len(report.Ambiguous) > 0 {
		fmt.Println("ambiguous_flat_transforms:")
		for _, event := range report.Ambiguous {
			fmt.Printf("- %s->%s %s type_sum_delta=%d object_id_sum_delta=%d state_sum_word3_delta=%d carry_sum_word4_delta=%d confidence=%s\n",
				event.FromTime, event.ToTime, event.PlayerLabel, event.UnitTypeSumDelta, event.ObjectIDSumDelta,
				event.Word3Delta, event.Word4Delta, event.Confidence)
		}
	}
	for _, w := range report.Warnings {
		fmt.Printf("warning: %s\n", w)
	}
}

func combatNameText(id int, name string) string {
	if name == "" {
		return fmt.Sprintf("P%d", id)
	}
	return fmt.Sprintf("P%d/%s", id, name)
}

func printReplayCarrier(report *replay.CarrierReport) {
	fmt.Printf("path: %s\n", report.Path)
	fmt.Printf("method: %s\n", report.Method)
	fmt.Printf("verification: %s\n", report.Verification)
	fmt.Printf("ledger: %s\n", report.LedgerPath)
	fmt.Printf("carrier: player=P%d attr=%d/%s word=%d\n", report.Carrier.PlayerID, report.Carrier.Attribute, report.Carrier.AttributeName, report.Carrier.WordIndex)
	fmt.Printf("summary: checksum_samples=%d attempts=%d passed=%d failed=%d unknown=%d unsampled=%d cadence_ms min=%d median=%d max=%d\n",
		report.Summary.ChecksumSamples, report.Summary.AttemptCount, report.Summary.Passed, report.Summary.Failed,
		report.Summary.Unknown, report.Summary.Unsampled, report.Summary.MinChecksumGapMS, report.Summary.MedianChecksumGapMS, report.Summary.MaxChecksumGapMS)
	if len(report.Attempts) > 0 {
		fmt.Println("attempts:")
		for _, attempt := range report.Attempts {
			window := fmt.Sprintf("%s", replay.FormatTime(attempt.WindowStartMS))
			if attempt.WindowEndMS > 0 {
				window += ".." + replay.FormatTime(attempt.WindowEndMS)
			}
			label := attempt.Label
			if label == "" {
				label = fmt.Sprintf("attempt_%d", attempt.Index)
			}
			fmt.Printf("- #%d %s verdict=%s matched=%s:%d observed=%d confidence=%s",
				attempt.Index, window, attempt.Verdict, attempt.MatchedKey, attempt.MatchedValue, len(attempt.Observed), attempt.Confidence)
			if label != "" {
				fmt.Printf(" label=%q", label)
			}
			if attempt.Note != "" {
				fmt.Printf(" note=%q", attempt.Note)
			}
			fmt.Println()
			if len(attempt.ExpectedValues) > 0 {
				keys := make([]string, 0, len(attempt.ExpectedValues))
				for key := range attempt.ExpectedValues {
					keys = append(keys, key)
				}
				sort.Strings(keys)
				fmt.Printf("  expected:")
				for _, key := range keys {
					fmt.Printf(" %s=%d", key, attempt.ExpectedValues[key])
				}
				fmt.Println()
			}
			for _, sample := range attempt.Observed {
				fmt.Printf("  sample: %s value=%d\n", sample.Time, sample.Value)
			}
		}
	}
	if len(report.Samples) > 0 {
		fmt.Println("carrier_samples:")
		limit := len(report.Samples)
		if limit > 30 {
			limit = 30
		}
		for _, sample := range report.Samples[:limit] {
			fmt.Printf("- %s value=%d\n", sample.Time, sample.Value)
		}
		if len(report.Samples) > limit {
			fmt.Printf("- ... %d more\n", len(report.Samples)-limit)
		}
	}
	for _, w := range report.Warnings {
		fmt.Printf("warning: %s\n", w)
	}
}

func printReplaySidecarSync(report *replay.SidecarSyncReport) {
	fmt.Printf("path: %s\n", report.Path)
	fmt.Printf("method: %s\n", report.Method)
	fmt.Printf("verification: %s\n", report.Verification)
	fmt.Printf("xsdat: %s\n", report.SidecarPath)
	if report.LedgerPath != "" {
		fmt.Printf("ledger: %s\n", report.LedgerPath)
	}
	if report.Schema != "" {
		fmt.Printf("schema: %s\n", report.Schema)
	}
	fmt.Printf("sidecar: magic=%q version=%d label=%q rows_written=%d append_probe=%d\n",
		report.Sidecar.Magic, report.Sidecar.Version, report.Sidecar.Label, report.Sidecar.RowsWritten, report.Sidecar.AppendProbe)
	fmt.Printf("summary: phases=%d checksum_samples=%d sidecar_ok=%t ledger=%d/%d failed=%d unknown=%d matched=%d missing=%d known_words=%d/%d mismatches=%d carrier_word1=%d unit_type_word2=%d object_count_word6=%d attr_mirrors=%d\n",
		report.Summary.PhaseCount, report.Summary.ChecksumSamples, report.Summary.SidecarDecodeOK,
		report.Summary.LedgerPassed, report.Summary.LedgerAssertions, report.Summary.LedgerFailed, report.Summary.LedgerUnknown,
		report.Summary.MatchedPhases, report.Summary.MissingSamples,
		report.Summary.KnownWordMatches, report.Summary.KnownWordChecks, report.Summary.KnownWordMismatches,
		report.Summary.CarrierMatches, report.Summary.UnitTypeMatches, report.Summary.ObjectCountMatches, report.Summary.AttrMirrorHits)
	if len(report.Correlations) > 0 {
		fmt.Println("phases:")
		for _, row := range report.Correlations {
			phase := row.Phase
			label := phase.Label
			if label == "" {
				label = fmt.Sprintf("phase_%d", phase.PhaseID)
			}
			sampleText := "sample=missing"
			wordsText := ""
			if row.Sample != nil {
				sampleText = fmt.Sprintf("sample=%s lag=%dms", row.Sample.Time, row.Sample.LagMS)
				if len(row.Sample.Words) >= 11 {
					wordsText = fmt.Sprintf(" words[1,2,3,4,6,7,9,10]=[%d,%d,%d,%d,%d,%d,%d,%d]",
						row.Sample.Words[1], row.Sample.Words[2], row.Sample.Words[3], row.Sample.Words[4],
						row.Sample.Words[6], row.Sample.Words[7], row.Sample.Words[9], row.Sample.Words[10])
				}
			}
			expectedParts := []string{fmt.Sprintf("resource=%d", phase.Expected.ResourceStockpile)}
			if phase.Expected.UnitTypeSumKnown {
				expectedParts = append(expectedParts, fmt.Sprintf("unit_sum=%d", phase.Expected.UnitTypeSum))
			}
			if phase.Expected.ObjectCountKnown {
				expectedParts = append(expectedParts, fmt.Sprintf("objects=%d", phase.Expected.ObjectCount))
			}
			fmt.Printf("- phase=%d label=%q verdict=%s xs_time=%s food=%.0f expected(%s) %s",
				phase.PhaseID, label, row.Verdict, replay.FormatTime(phase.TargetMS), phase.Food, strings.Join(expectedParts, " "), sampleText)
			if row.CarrierMatch != nil {
				fmt.Printf(" w1=%t", *row.CarrierMatch)
			}
			if row.UnitTypeMatch != nil {
				fmt.Printf(" w2=%t", *row.UnitTypeMatch)
			}
			if row.ObjectCountMatch != nil {
				fmt.Printf(" w6=%t", *row.ObjectCountMatch)
			}
			fmt.Print(wordsText)
			if len(row.DeltaFromPrevious) >= 11 {
				fmt.Printf(" delta[1,2,3,4,6,7,9,10]=[%d,%d,%d,%d,%d,%d,%d,%d]",
					row.DeltaFromPrevious[1], row.DeltaFromPrevious[2], row.DeltaFromPrevious[3], row.DeltaFromPrevious[4],
					row.DeltaFromPrevious[6], row.DeltaFromPrevious[7], row.DeltaFromPrevious[9], row.DeltaFromPrevious[10])
			}
			if len(row.AttributeMirrors) > 0 {
				fmt.Print(" attr_mirrors=")
				for i, mirror := range row.AttributeMirrors {
					if i > 0 {
						fmt.Print(",")
					}
					fmt.Printf("%s:%d@word%d", mirror.Attribute, mirror.Value, mirror.WordIndex)
				}
			}
			fmt.Println()
			for _, check := range row.KnownWordChecks {
				if check.Match {
					continue
				}
				fmt.Printf("  mismatch: word_%d %s expected=%d actual=%d\n", check.WordIndex, check.Name, check.Expected, check.Actual)
			}
			for _, note := range row.Notes {
				fmt.Printf("  note: %s\n", note)
			}
		}
	}
	for _, w := range report.Warnings {
		fmt.Printf("warning: %s\n", w)
	}
}

func printReplayXSTelemetry(report *replay.XSTelemetryReport) {
	fmt.Printf("path: %s\n", report.Path)
	fmt.Printf("method: %s\n", report.Method)
	fmt.Printf("verification: %s\n", report.Verification)
	fmt.Printf("summary: checksum_samples=%d decoded=%d food_carrier=%d unused_220_carrier=%d chat_markers=%d food_moved=%t unused_220_moved=%t\n",
		report.Summary.ChecksumSamples, report.Summary.DecodedSamples, report.Summary.FoodCarrierSamples, report.Summary.Unused220CarrierSamples,
		report.Summary.ChatMarkers, report.Summary.FoodCarrierMoved, report.Summary.Unused220CarrierMoved)
	if len(report.Samples) > 0 {
		fmt.Println("samples:")
		for _, sample := range report.Samples {
			fmt.Printf("- %s %s carrier=%s word1=%d digest_word9=%d attr20=%d attr154_kill_events=%d attr43=%d confidence=%s\n",
				sample.Time, sample.PlayerLabel, sample.Carrier, sample.Word1, sample.Word9, sample.Attr20Value, sample.KillEventsFromAttr154, sample.Attr43Value, sample.Confidence)
		}
	}
	if len(report.ChatMarkers) > 0 {
		fmt.Println("chat_markers:")
		for _, marker := range report.ChatMarkers {
			fmt.Printf("- %s P%d %q confidence=%s\n", marker.Time, marker.PlayerID, marker.Text, marker.Confidence)
		}
	}
	for _, w := range report.Warnings {
		fmt.Printf("warning: %s\n", w)
	}
}

func printReplayCamera(report *replay.CameraReport) {
	fmt.Printf("path: %s\n", report.Path)
	fmt.Printf("method: %s\n", report.Method)
	fmt.Printf("verification: %s\n", report.Verification)
	fmt.Printf("summary: events=%d emitted=%d duration=%s distinct_tails=%d tails_match_roster=%t\n",
		report.Summary.Events, report.Summary.EmittedEvents, report.Summary.Duration, report.Summary.DistinctTails, report.Summary.TailsMatchRoster)
	fmt.Printf("tail_mapping: %s\n", report.Summary.TailMappingNote)
	for _, s := range report.Streams {
		who := ""
		if s.PlayerNameIfTail != "" {
			who = fmt.Sprintf(" player_if_tail=P%d/%s", s.PlayerNumberIfTail, s.PlayerNameIfTail)
		}
		fmt.Printf("- tail=%d%s events=%d span=%s..%s bbox=(%.1f,%.1f)-(%.1f,%.1f) dist=%.0f mean_step=%.2f max_step=%.1f jumps_over_20=%d confidence=%s\n",
			s.TailU32, who, s.Events, s.FirstTime, s.LastTime, s.MinX, s.MinY, s.MaxX, s.MaxY,
			s.DistanceTraveled, s.MeanStepDistance, s.MaxStepDistance, s.LargeJumps, s.MappingConfidence)
	}
	if len(report.Events) > 0 {
		fmt.Printf("events (first %d):\n", len(report.Events))
		for _, e := range report.Events {
			fmt.Printf("  %s tail=%d x=%.2f y=%.2f\n", e.Time, e.TailU32, e.X, e.Y)
		}
	}
	for _, w := range report.Warnings {
		fmt.Printf("warning: %s\n", w)
	}
}

func printReplayPostgame(report *replay.PostgameReport) {
	fmt.Printf("path: %s\n", report.Path)
	fmt.Printf("method: %s\n", report.Method)
	fmt.Printf("verification: %s\n", report.Verification)
	fmt.Printf("summary: op6_seen=%t op6_tail_bytes=%d op6_blocks=%d action255_seen=%t action255_payloads=%d leaderboard=%t world_time=%t player_kills=%t\n",
		report.Summary.Op6Seen, report.Summary.Op6TailBytes, report.Summary.Op6Blocks, report.Summary.Action255Seen, report.Summary.Action255Payloads, report.Summary.HasLeaderboardBlock, report.Summary.HasWorldTimeBlock, report.Summary.HasPlayerKills)
	if report.DE != nil {
		fmt.Printf("de_postgame: offset=%d time=%s tail_bytes=%d version=%d block_count=%d confidence=%s\n",
			report.DE.SourceOffset, report.DE.Time, report.DE.TailBytes, report.DE.Version, report.DE.BlockCount, report.DE.Confidence)
		if report.DE.WorldTimeMS > 0 {
			fmt.Printf("world_time_ms: %d (%s)\n", report.DE.WorldTimeMS, replay.FormatTime(int(report.DE.WorldTimeMS)))
		}
		if len(report.DE.Blocks) > 0 {
			fmt.Println("blocks:")
			for _, block := range report.DE.Blocks {
				fmt.Printf("- id=%d name=%s length=%d parsed=%t confidence=%s\n", block.ID, block.Name, block.Length, block.Parsed, block.Confidence)
			}
		}
		if len(report.DE.Leaderboards) > 0 {
			fmt.Println("leaderboards:")
			for _, board := range report.DE.Leaderboards {
				fmt.Printf("- id=%d unknown=%d players=%d\n", board.ID, board.Unknown, len(board.Players))
				for _, player := range board.Players {
					fmt.Printf("  - player=%d rank=%d rating=%d\n", player.PlayerNumber, player.Rank, player.Rating)
				}
			}
		}
	}
	if len(report.Action255) > 0 {
		fmt.Println("action_255_postgame:")
		for _, action := range report.Action255 {
			fmt.Printf("- offset=%d time=%s payload_bytes=%d sequence=%d confidence=%s\n", action.SourceOffset, action.Time, action.PayloadBytes, action.Sequence, action.Confidence)
		}
	}
	if len(report.Players) > 0 {
		fmt.Println("players:")
		for _, player := range report.Players {
			fmt.Printf("- %s name=%q confidence=%s\n", player.Label, player.Name, player.Confidence)
		}
	}
	if len(report.Warnings) > 0 {
		fmt.Println("warnings:")
		for _, warning := range report.Warnings {
			fmt.Printf("- %s\n", warning)
		}
	}
}

func printReplaySummary(report *replay.SummaryReport) {
	fmt.Printf("replay: players=%d duration=%s scenario=%q\n", len(report.Players), report.Duration, report.ScenarioName)
	if report.RecordSHA256 != "" {
		fmt.Printf("record: sha256=%s\n", shortHash(report.RecordSHA256))
	}
	if report.HeaderMeta != nil && report.HeaderMeta.TailGUIDHex != "" {
		fmt.Printf("metadata: tail_guid=%s offset=%d confidence=%s\n", report.HeaderMeta.TailGUIDHex, report.HeaderMeta.TailGUIDOffset, report.HeaderMeta.Confidence)
	}
	fmt.Printf("version: game=%s save=%.2f log=%d\n", report.GameVersion, report.SaveVersion, report.LogVersion)
	if report.Map != nil {
		fmt.Printf("map: %dx%d tiles=%d terrain_sha256=%s\n", report.Map.Width, report.Map.Height, report.Map.TileCount, report.Map.TerrainSHA256)
	}
	if report.LobbySettings != nil {
		lobby := report.LobbySettings
		fmt.Printf("lobby: game_type=%d difficulty=%d pop=%d victory=%d/%d speed=%d treaty=%d diplomacy=%d ranked=%d\n",
			lobby.GameType, lobby.Difficulty, lobby.PopulationLimit, lobby.VictoryType, lobby.VictoryValue,
			lobby.GameSpeed, lobby.TreatyLength, lobby.DiplomacyType, lobby.Ranked)
	}
	if report.Result.WinnerKnown {
		fmt.Printf("result: %s winners=%v losers=%v\n", report.Result.Method, report.Result.Winners, report.Result.Losers)
	} else {
		fmt.Printf("result: %s winner_known=false completed=%t\n", report.Result.Method, report.Result.Completed)
	}
	for _, player := range report.Players {
		rating := ""
		if player.Rating != nil {
			rating = fmt.Sprintf(" rating=%d", *player.Rating)
		}
		winner := ""
		if player.Winner != nil {
			winner = fmt.Sprintf(" winner=%t", *player.Winner)
		}
		fmt.Printf("- slot=%d p=%d %q civ=%d/%s color=%d team=%d profile=%d kind=%s%s%s\n",
			player.Slot, player.PlayerID, player.Name, player.Civ, player.CivName, player.Color,
			player.Team, player.ProfileID, player.Kind, rating, winner)
	}
	for _, warning := range report.Warnings {
		fmt.Printf("warning: %s\n", warning)
	}
}

func replayUnitText(unitID int, unitName string) string {
	if unitName == "" {
		unitName = replay.UnitDisplayName(unitID)
	}
	if unitName == "" {
		return fmt.Sprintf("%d", unitID)
	}
	return fmt.Sprintf("%d/%s", unitID, unitName)
}

func stringIntMapText(values map[string]int) string {
	if len(values) == 0 {
		return "{}"
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%d", key, values[key]))
	}
	return strings.Join(parts, ",")
}

func intIntMapText(values map[int]int) string {
	if len(values) == 0 {
		return "{}"
	}
	keys := make([]int, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Ints(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%d=%d", key, values[key]))
	}
	return strings.Join(parts, ",")
}

func printDatDiff(report *datfile.DiffReport) {
	fmt.Printf("base: %s\n", report.Base)
	fmt.Printf("mod: %s\n", report.Mod)
	fmt.Printf("same: %t\n", report.Same)
	fmt.Printf("verification: %s\n", report.Verification)
	fmt.Printf("summary: sections=%d added=%d removed=%d changed=%d emitted=%d truncated=%t\n",
		report.Summary.SectionsCompared, report.Summary.Added, report.Summary.Removed,
		report.Summary.Changed, report.Summary.Emitted, report.Summary.Truncated)
	for _, section := range report.Sections {
		if section.Added == 0 && section.Removed == 0 && section.Changed == 0 {
			continue
		}
		fmt.Printf("- %s base=%d mod=%d added=%d removed=%d changed=%d",
			section.Name, section.BaseCount, section.ModCount, section.Added, section.Removed, section.Changed)
		if section.Truncated {
			fmt.Printf(" truncated=true")
		}
		fmt.Println()
		for _, entry := range section.Entries {
			name := ""
			if entry.Name != "" {
				name = " " + clippedField(entry.Name, 64)
			}
			fmt.Printf("  %s key=%s%s", entry.Kind, entry.Key, name)
			if entry.BaseSHA256 != "" {
				fmt.Printf(" base=%s", shortHash(entry.BaseSHA256))
			}
			if entry.ModSHA256 != "" {
				fmt.Printf(" mod=%s", shortHash(entry.ModSHA256))
			}
			fmt.Println()
		}
	}
	for _, warning := range report.Warnings {
		fmt.Printf("warning: %s\n", warning)
	}
}

func initialLabel(playerID int) string {
	if playerID == 0 {
		return "Gaia"
	}
	return fmt.Sprintf("P%d", playerID)
}

func verifyRun(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: kit verify-run <replay.aoe2record|replay.zip> --contract contract.json [--text]")
		os.Exit(2)
	}
	replayPath := args[0]
	contractPath := ""
	textOut := false
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--contract":
			i++
			if i >= len(args) {
				die("kit verify-run", fmt.Errorf("--contract needs a JSON path"))
			}
			contractPath = args[i]
		case "--text":
			textOut = true
		default:
			die("kit verify-run", fmt.Errorf("unknown option %q", args[i]))
		}
	}
	if contractPath == "" {
		die("kit verify-run", fmt.Errorf("--contract is required"))
	}
	report, err := replay.VerifyRun(replayPath, contractPath)
	if err != nil {
		die("kit verify-run", err)
	}
	if textOut {
		printVerifyRun(report)
	} else {
		printJSON(report)
	}
	if !report.OK {
		os.Exit(1)
	}
}

func printVerifyRun(report *replay.VerifyRunReport) {
	fmt.Printf("replay: %s\n", report.Replay)
	fmt.Printf("contract: %s\n", report.Contract)
	if report.Name != "" {
		fmt.Printf("name: %s\n", report.Name)
	}
	fmt.Printf("ok: %t passed=%d failed=%d unknown=%d\n", report.OK, report.Summary.Passed, report.Summary.Failed, report.Summary.Unknown)
	if report.Story != nil {
		if report.Story.Identity.SHA256 != "" {
			fmt.Printf("identity: %s %s\n", report.Story.Identity.Tier, shortHash(report.Story.Identity.SHA256))
		}
		fmt.Printf("data_set: %s\n", formatDataSetIdentity(report.Story.DataSet))
		if report.Story.EventCounts.Duration != "" {
			fmt.Printf("duration: %s\n", report.Story.EventCounts.Duration)
		}
		if report.Story.Result.WinnerKnown {
			fmt.Printf("result: %s winners=%v losers=%v\n", report.Story.Result.Method, report.Story.Result.Winners, report.Story.Result.Losers)
		}
	}
	if len(report.Claims) > 0 {
		fmt.Println("claims:")
		for _, claim := range report.Claims {
			fmt.Printf("- [%s] %s tier=%s source=%s", strings.ToUpper(claim.Status), claim.Name, claim.Tier, claim.Source)
			if claim.Expected != "" {
				fmt.Printf(" expected=%s", claim.Expected)
			}
			if claim.Observed != "" {
				fmt.Printf(" observed=%s", claim.Observed)
			}
			if claim.Message != "" {
				fmt.Printf(" - %s", claim.Message)
			}
			fmt.Println()
			if claim.Gotcha != "" || claim.Remediation != "" {
				fmt.Printf("  gotcha: %s\n", claim.Gotcha)
				fmt.Printf("  fix: %s\n", claim.Remediation)
			}
		}
	}
	if len(report.Warnings) > 0 {
		fmt.Println("warnings:")
		for _, warning := range report.Warnings {
			fmt.Printf("- %s\n", warning)
		}
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

func scenarioInfoArg(path string) (scenarioInfoReport, error) {
	info, err := inspectScenarioArg(path)
	if err != nil {
		return scenarioInfoReport{}, err
	}
	scen, err := scenario.Open(path)
	if err != nil {
		return scenarioInfoReport{}, err
	}
	stringsReport := scen.Strings()
	units := 0
	if scen.Units != nil {
		units = scen.Units.Total
	}
	triggers := 0
	variables := 0
	if scen.Triggers != nil {
		triggers = scen.Triggers.Count
		variables = scen.Triggers.Variables
	}
	return scenarioInfoReport{
		FileInfo:         info,
		Version:          scen.Version,
		HeaderBytes:      scen.HeaderBytes,
		CompressedBytes:  scen.CompressedBytes,
		InflatedBytes:    scen.InflatedBytes,
		Sections:         len(scen.Sections),
		Triggers:         triggers,
		Units:            units,
		Variables:        variables,
		Messages:         stringsReport.Messages,
		Strings:          len(stringsReport.Messages) + len(stringsReport.StringTable) + len(stringsReport.EffectText),
		StringTable:      len(stringsReport.StringTable),
		EffectText:       len(stringsReport.EffectText),
		MarkupSummary:    stringsReport.MarkupSummary,
		VerificationNote: stringsReport.Verification,
	}, nil
}

func runDat(args []string) {
	if len(args) >= 1 && args[0] == "facts" {
		runDatFacts(args[1:])
		return
	}
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: kit dat <facts|info|check|roundtrip|diff|sprite|semantic-priors|sql|unit-headers|unit-header|civs|civ-patch|spans|graphics|graphic|graphic-create|graphic-patch|graphic-delete|palette|effects|effect|effect-explain|effect-create|effect-patch|effect-disable|effect-delete|effect-command-delete|command-matrix|techs|tech|tech-explain|tech-create|tech-patch|tech-delete|abilities|ability|ability-create|ability-patch|ability-disable|ability-delete|tech-tree|tech-tree-connection-create|tech-tree-connection-patch|tech-tree-connection-delete|units|unit|unit-create|unit-delete|unit-child-delete|availability|availability-set|terrain-restrictions|terrain-restriction-patch|terrains|terrain|terrain-patch|sounds|sound|sound-create|sound-patch|sound-delete|sound-item-delete|player-colours|player-colour-create|player-colour-patch|player-colour-delete|random-maps|patch-graphic|patch-unit|graphic-delta-delete|graphic-angle-sound-delete|unit-header-task-delete|refs|delete-plan|delete|codec-plan|codec-patch|semantics-pack|semantics-readback|plan|patch> <empires*.dat> [args...]")
		os.Exit(2)
	}
	if args[0] == "semantics-readback" {
		runDatSemanticsReadback(args)
		return
	}
	if args[0] == "sprite" {
		options, text := parseSLDExportOptions("kit dat sprite", args[2:])
		report, err := gfx.ExportSLD(args[1], options)
		if err != nil {
			die("kit dat sprite", err)
		}
		if text {
			printSLDExportText(report)
		} else {
			printJSON(report)
		}
		return
	}
	if args[0] == "semantic-priors" {
		textOut := false
		for _, arg := range args[2:] {
			switch arg {
			case "--text":
				textOut = true
			case "--json":
				textOut = false
			default:
				die("kit dat semantic-priors", fmt.Errorf("unknown option %q", arg))
			}
		}
		cache, err := datcache.Open(args[1])
		if err != nil {
			die("kit dat semantic-priors", err)
		}
		report, err := cache.SemanticPriors()
		_ = cache.Close()
		if err != nil {
			die("kit dat semantic-priors", err)
		}
		if textOut {
			printDatSemanticPriors(report)
		} else {
			printJSON(report)
		}
		return
	}
	if args[0] == "sql" {
		if len(args) != 3 {
			fmt.Fprintln(os.Stderr, "usage: kit dat sql <empires*.dat> <out.db>")
			os.Exit(2)
		}
		if err := datfile.ExportSQLite(args[1], args[2]); err != nil {
			die("kit dat sql", err)
		}
		printJSON(struct {
			Operation    string `json:"operation"`
			DAT          string `json:"dat"`
			Output       string `json:"output"`
			Verification string `json:"verification"`
		}{"dat_sql_export", args[1], args[2], "structure_verified_not_engine_verified"})
		return
	}
	info, err := aoe2.InspectFile(args[1])
	if err != nil {
		die("kit dat", err)
	}
	if info.Kind != "dat" {
		die("kit dat", fmt.Errorf("%s is %q, not dat", args[1], info.Kind))
	}
	switch args[0] {
	case "info", "inspect":
		printJSON(info)
	case "check":
		fmt.Printf("%s: %d bytes sha256=%s [OK]\n", info.Base, info.SizeBytes, info.SHA256)
	case "diff":
		if len(args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: kit dat diff <base.dat> <mod.dat> [--section NAME|all] [--limit N|--all] [--text]")
			os.Exit(2)
		}
		opts, textOut, err := parseDatDiffOptions(args[3:])
		if err != nil {
			die("kit dat diff", err)
		}
		report, err := datfile.DiffFiles(args[1], args[2], opts)
		if err != nil {
			die("kit dat diff", err)
		}
		if textOut {
			printDatDiff(report)
		} else {
			printJSON(report)
		}
	case "roundtrip":
		full := false
		for _, arg := range args[2:] {
			switch arg {
			case "--full":
				full = true
			default:
				die("kit dat roundtrip", fmt.Errorf("unknown option %q", arg))
			}
		}
		report, err := datcodec.RoundTripFile(args[1])
		if err != nil {
			die("kit dat roundtrip", err)
		}
		if !full {
			report.TypedTargets.Effects.Spans = nil
			report.TypedTargets.Effects.Records = nil
			report.TypedTargets.Techs.Spans = nil
			report.TypedTargets.Techs.Records = nil
			report.TypedTargets.Civs.Spans = nil
			report.TypedTargets.Civs.Records = nil
		}
		printJSON(report)
		if !report.OK {
			os.Exit(1)
		}
	case "unit-headers":
		limit, all, err := parseDatListOptions(args[2:], "kit dat unit-headers")
		if err != nil {
			die("kit dat", err)
		}
		idx, err := datfile.Open(args[1])
		if err != nil {
			die("kit dat", err)
		}
		presentCount := 0
		taskCount := 0
		for _, header := range idx.UnitHeaders {
			if header.Exists {
				presentCount++
			}
			taskCount += len(header.Tasks)
		}
		headers := idx.UnitHeaders
		truncated := false
		if !all && len(headers) > limit {
			headers = headers[:limit]
			truncated = true
		}
		printJSON(struct {
			Version       string               `json:"version"`
			HeaderCount   int                  `json:"header_count"`
			PresentCount  int                  `json:"present_count"`
			TaskCount     int                  `json:"task_count"`
			ReturnedCount int                  `json:"returned_count"`
			Truncated     bool                 `json:"truncated"`
			UnitHeaders   []datfile.UnitHeader `json:"unit_headers"`
		}{Version: idx.Version, HeaderCount: len(idx.UnitHeaders), PresentCount: presentCount, TaskCount: taskCount, ReturnedCount: len(headers), Truncated: truncated, UnitHeaders: headers})
	case "unit-header":
		if len(args) != 3 {
			fmt.Fprintln(os.Stderr, "usage: kit dat unit-header <empires*.dat> <header_id>")
			os.Exit(2)
		}
		id, err := strconv.Atoi(args[2])
		if err != nil {
			die("kit dat unit-header", fmt.Errorf("invalid header id %q: %w", args[2], err))
		}
		idx, err := datfile.Open(args[1])
		if err != nil {
			die("kit dat unit-header", err)
		}
		if id < 0 || id >= len(idx.UnitHeaders) {
			die("kit dat unit-header", fmt.Errorf("header id %d outside unit header table length %d", id, len(idx.UnitHeaders)))
		}
		printJSON(struct {
			Version    string             `json:"version"`
			HeaderID   int                `json:"header_id"`
			UnitHeader datfile.UnitHeader `json:"unit_header"`
		}{Version: idx.Version, HeaderID: id, UnitHeader: idx.UnitHeaders[id]})
	case "civs":
		idx, err := datfile.Open(args[1])
		if err != nil {
			die("kit dat", err)
		}
		type civSummary struct {
			Index         int    `json:"index"`
			Type          uint8  `json:"type"`
			Name          string `json:"name"`
			ResourcesSize int    `json:"resources_size"`
			TechTreeID    int16  `json:"tech_tree_id"`
			TeamBonusID   int16  `json:"team_bonus_id"`
			IconSet       uint8  `json:"icon_set"`
			UnitsSize     int    `json:"units_size"`
			PresentUnits  int    `json:"present_units"`
			RecordStart   int    `json:"record_start"`
			RecordEnd     int    `json:"record_end"`
		}
		out := make([]civSummary, 0, len(idx.Civs))
		for _, civ := range idx.Civs {
			present := 0
			for _, unit := range civ.Units {
				if unit.Present {
					present++
				}
			}
			out = append(out, civSummary{
				Index:         civ.Index,
				Type:          civ.Type,
				Name:          civ.Name,
				ResourcesSize: civ.ResourcesSize,
				TechTreeID:    civ.TechTreeID,
				TeamBonusID:   civ.TeamBonusID,
				IconSet:       civ.IconSet,
				UnitsSize:     civ.UnitsSize,
				PresentUnits:  present,
				RecordStart:   civ.Span.Start,
				RecordEnd:     civ.Span.End,
			})
		}
		printJSON(struct {
			Version string       `json:"version"`
			Civs    []civSummary `json:"civs"`
		}{Version: idx.Version, Civs: out})
	case "civ-patch", "patch-civ":
		if len(args) < 4 {
			fmt.Fprintln(os.Stderr, "usage: kit dat civ-patch <in.dat> <out.dat> <civ_id> [--name TEXT] [--tech-tree-id N] [--team-bonus-id N] [--icon-set N] [--resource INDEX,VALUE]")
			os.Exit(2)
		}
		id, err := strconv.Atoi(args[3])
		if err != nil {
			die("kit dat civ-patch", fmt.Errorf("invalid civ id %q: %w", args[3], err))
		}
		recipe, err := parseDatCivPatchOptions(id, args[4:])
		if err != nil {
			die("kit dat civ-patch", err)
		}
		report, err := datcodec.PatchRecipeFile(args[1], args[2], datcodec.Recipe{Civs: []datcodec.CivPatchRecipe{recipe}})
		if err != nil {
			die("kit dat civ-patch", err)
		}
		printJSON(struct {
			Version      string                  `json:"version"`
			Operation    string                  `json:"operation"`
			CivID        int                     `json:"civ_id"`
			Request      datcodec.CivPatchRecipe `json:"request"`
			PatchReport  datcodec.PatchReport    `json:"patch_report"`
			Verification string                  `json:"verification"`
		}{
			Version:      datcodec.Version,
			Operation:    "civ_patch",
			CivID:        id,
			Request:      recipe,
			PatchReport:  report,
			Verification: "structure_verified_not_engine_verified",
		})
	case "spans":
		idx, err := datfile.Open(args[1])
		if err != nil {
			die("kit dat", err)
		}
		printJSON(struct {
			Version       string         `json:"version"`
			InflatedBytes int            `json:"inflated_bytes"`
			SpanCount     int            `json:"span_count"`
			Spans         []datfile.Span `json:"spans"`
		}{Version: idx.Version, InflatedBytes: idx.Inflated, SpanCount: len(idx.Spans), Spans: idx.Spans})
	case "player-colours", "player-colors", "colors", "colours":
		idx, err := datfile.Open(args[1])
		if err != nil {
			die("kit dat", err)
		}
		printJSON(struct {
			Version       string                 `json:"version"`
			ColourCount   int                    `json:"colour_count"`
			PlayerColours []datfile.PlayerColour `json:"player_colours"`
		}{Version: idx.Version, ColourCount: len(idx.PlayerColours), PlayerColours: idx.PlayerColours})
	case "player-colour-create", "player-color-create", "create-player-colour", "create-player-color":
		if len(args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: kit dat player-colour-create <in.dat> <out.dat> --from N [colour-flags]")
			os.Exit(2)
		}
		recipe, err := parseDatPlayerColourCreateOptions(args[3:])
		if err != nil {
			die("kit dat player-colour-create", err)
		}
		report, err := datcodec.PatchRecipeFile(args[1], args[2], datcodec.Recipe{CreatePlayerColour: &recipe})
		if err != nil {
			die("kit dat player-colour-create", err)
		}
		printJSON(struct {
			Version      string                            `json:"version"`
			Operation    string                            `json:"operation"`
			Request      datcodec.PlayerColourCreateRecipe `json:"request"`
			PatchReport  datcodec.PatchReport              `json:"patch_report"`
			Verification string                            `json:"verification"`
		}{
			Version:      datcodec.Version,
			Operation:    "player_colour_create",
			Request:      recipe,
			PatchReport:  report,
			Verification: "structure_verified_not_engine_verified",
		})
	case "player-colour-patch", "player-color-patch", "patch-player-colour", "patch-player-color":
		if len(args) < 4 {
			fmt.Fprintln(os.Stderr, "usage: kit dat player-colour-patch <in.dat> <out.dat> <colour_id> [colour-flags]")
			os.Exit(2)
		}
		id, err := strconv.Atoi(args[3])
		if err != nil {
			die("kit dat player-colour-patch", fmt.Errorf("invalid colour id %q: %w", args[3], err))
		}
		recipe, err := parseDatPlayerColourPatchOptions(id, args[4:])
		if err != nil {
			die("kit dat player-colour-patch", err)
		}
		report, err := datcodec.PatchRecipeFile(args[1], args[2], datcodec.Recipe{PlayerColours: []datcodec.PlayerColourPatchRecipe{recipe}})
		if err != nil {
			die("kit dat player-colour-patch", err)
		}
		printJSON(struct {
			Version      string                           `json:"version"`
			Operation    string                           `json:"operation"`
			ColourID     int                              `json:"colour_id"`
			Request      datcodec.PlayerColourPatchRecipe `json:"request"`
			PatchReport  datcodec.PatchReport             `json:"patch_report"`
			Verification string                           `json:"verification"`
		}{
			Version:      datcodec.Version,
			Operation:    "player_colour_patch",
			ColourID:     id,
			Request:      recipe,
			PatchReport:  report,
			Verification: "structure_verified_not_engine_verified",
		})
	case "player-colour-delete", "player-color-delete", "delete-player-colour", "delete-player-color":
		if len(args) != 4 {
			fmt.Fprintln(os.Stderr, "usage: kit dat player-colour-delete <in.dat> <out.dat> <colour_id>")
			os.Exit(2)
		}
		id, err := strconv.Atoi(args[3])
		if err != nil {
			die("kit dat player-colour-delete", fmt.Errorf("invalid colour id %q: %w", args[3], err))
		}
		plan, err := datcodec.DeletePlanFile(args[1], datcodec.DeletePlanRequest{Section: "player_colour", ID: id})
		if err != nil {
			die("kit dat player-colour-delete", err)
		}
		if !plan.Supported || !plan.Mutates || len(plan.RecipeHint) == 0 {
			reason := plan.Reason
			if reason == "" {
				reason = "delete-plan did not produce an actionable recipe"
			}
			die("kit dat player-colour-delete", fmt.Errorf("refusing unsupported delete: strategy=%s reason=%s", plan.Strategy, reason))
		}
		var recipe datcodec.Recipe
		if err := json.Unmarshal(plan.RecipeHint, &recipe); err != nil {
			die("kit dat player-colour-delete", fmt.Errorf("delete-plan recipe hint did not decode: %w", err))
		}
		report, err := datcodec.PatchRecipeFile(args[1], args[2], recipe)
		if err != nil {
			die("kit dat player-colour-delete", err)
		}
		printJSON(struct {
			Version      string                    `json:"version"`
			Operation    string                    `json:"operation"`
			ColourID     int                       `json:"colour_id"`
			DeletePlan   datcodec.DeletePlanReport `json:"delete_plan"`
			PatchReport  datcodec.PatchReport      `json:"patch_report"`
			Verification string                    `json:"verification"`
		}{
			Version:      datcodec.Version,
			Operation:    "player_colour_delete",
			ColourID:     id,
			DeletePlan:   plan,
			PatchReport:  report,
			Verification: "structure_verified_not_engine_verified",
		})
	case "random-maps":
		idx, err := datfile.Open(args[1])
		if err != nil {
			die("kit dat", err)
		}
		totalLands := 0
		totalTerrains := 0
		totalUnits := 0
		totalElevations := 0
		for _, randomMap := range idx.RandomMaps {
			totalLands += int(randomMap.Lands)
			totalTerrains += int(randomMap.Terrains)
			totalUnits += int(randomMap.Units)
			totalElevations += int(randomMap.Elevations)
		}
		printJSON(struct {
			Version         string                  `json:"version"`
			MapCount        int                     `json:"map_count"`
			TotalLands      int                     `json:"total_lands"`
			TotalTerrains   int                     `json:"total_terrains"`
			TotalUnits      int                     `json:"total_units"`
			TotalElevations int                     `json:"total_elevations"`
			Note            string                  `json:"note,omitempty"`
			RandomMaps      []datfile.RandomMapInfo `json:"random_maps"`
		}{
			Version:         idx.Version,
			MapCount:        len(idx.RandomMaps),
			TotalLands:      totalLands,
			TotalTerrains:   totalTerrains,
			TotalUnits:      totalUnits,
			TotalElevations: totalElevations,
			Note:            "random map records are structurally framed; write/delete support waits for a non-empty current-DE fixture.",
			RandomMaps:      idx.RandomMaps,
		})
	case "terrain-restrictions":
		limit, all, err := parseDatListOptions(args[2:], "kit dat terrain-restrictions")
		if err != nil {
			die("kit dat", err)
		}
		idx, err := datfile.Open(args[1])
		if err != nil {
			die("kit dat", err)
		}
		type terrainRestrictionRow struct {
			Index              int          `json:"index"`
			Passability        *float32     `json:"passability,omitempty"`
			PassabilitySpecial string       `json:"passability_special,omitempty"`
			PassGraphicRawSize int          `json:"pass_graphic_raw_size"`
			Span               datfile.Span `json:"span"`
			PassabilitySpan    datfile.Span `json:"passability_span"`
			PassGraphicSpan    datfile.Span `json:"pass_graphic_span"`
		}
		type terrainRestrictionSummary struct {
			Index       int                     `json:"index"`
			TerrainRows []terrainRestrictionRow `json:"terrain_rows,omitempty"`
			Span        datfile.Span            `json:"span"`
		}
		rowCount := 0
		restrictions := make([]terrainRestrictionSummary, 0, len(idx.TerrainRestrictions))
		selectedRestrictions := idx.TerrainRestrictions
		truncated := false
		if !all && len(selectedRestrictions) > limit {
			selectedRestrictions = selectedRestrictions[:limit]
			truncated = true
		}
		for _, restriction := range selectedRestrictions {
			rowCount += len(restriction.TerrainRows)
			rows := make([]terrainRestrictionRow, 0, len(restriction.TerrainRows))
			for _, row := range restriction.TerrainRows {
				item := terrainRestrictionRow{
					Index:              row.Index,
					PassGraphicRawSize: row.PassGraphicRawSize,
					Span:               row.Span,
					PassabilitySpan:    row.PassabilitySpan,
					PassGraphicSpan:    row.PassGraphicSpan,
				}
				switch {
				case math.IsNaN(float64(row.Passability)):
					item.PassabilitySpecial = "nan"
				case math.IsInf(float64(row.Passability), 1):
					item.PassabilitySpecial = "+inf"
				case math.IsInf(float64(row.Passability), -1):
					item.PassabilitySpecial = "-inf"
				default:
					value := row.Passability
					item.Passability = &value
				}
				rows = append(rows, item)
			}
			restrictions = append(restrictions, terrainRestrictionSummary{
				Index:       restriction.Index,
				TerrainRows: rows,
				Span:        restriction.Span,
			})
		}
		printJSON(struct {
			Version             string                      `json:"version"`
			RestrictionCount    int                         `json:"restriction_count"`
			ReturnedCount       int                         `json:"returned_count"`
			Truncated           bool                        `json:"truncated"`
			TerrainRowCount     int                         `json:"terrain_row_count"`
			TerrainRestrictions []terrainRestrictionSummary `json:"terrain_restrictions"`
		}{Version: idx.Version, RestrictionCount: len(idx.TerrainRestrictions), ReturnedCount: len(restrictions), Truncated: truncated, TerrainRowCount: rowCount, TerrainRestrictions: restrictions})
	case "terrain-restriction-patch", "patch-terrain-restriction":
		if len(args) < 4 {
			fmt.Fprintln(os.Stderr, "usage: kit dat terrain-restriction-patch <in.dat> <out.dat> <restriction_id> --terrain TERRAIN_ID,PASSABILITY")
			os.Exit(2)
		}
		id, err := strconv.Atoi(args[3])
		if err != nil {
			die("kit dat terrain-restriction-patch", fmt.Errorf("invalid terrain restriction id %q: %w", args[3], err))
		}
		recipe, err := parseDatTerrainRestrictionPatchOptions(id, args[4:])
		if err != nil {
			die("kit dat terrain-restriction-patch", err)
		}
		report, err := datcodec.PatchRecipeFile(args[1], args[2], datcodec.Recipe{TerrainRestrictions: []datcodec.TerrainRestrictionPatchRecipe{recipe}})
		if err != nil {
			die("kit dat terrain-restriction-patch", err)
		}
		printJSON(struct {
			Version       string                                 `json:"version"`
			Operation     string                                 `json:"operation"`
			RestrictionID int                                    `json:"restriction_id"`
			Request       datcodec.TerrainRestrictionPatchRecipe `json:"request"`
			PatchReport   datcodec.PatchReport                   `json:"patch_report"`
			Verification  string                                 `json:"verification"`
		}{
			Version:       datcodec.Version,
			Operation:     "terrain_restriction_patch",
			RestrictionID: id,
			Request:       recipe,
			PatchReport:   report,
			Verification:  "structure_verified_not_engine_verified",
		})
	case "terrains":
		limit, all, err := parseDatListOptions(args[2:], "kit dat terrains")
		if err != nil {
			die("kit dat", err)
		}
		idx, err := datfile.Open(args[1])
		if err != nil {
			die("kit dat", err)
		}
		terrains := idx.Terrains
		truncated := false
		if !all && len(terrains) > limit {
			terrains = terrains[:limit]
			truncated = true
		}
		printJSON(struct {
			Version       string            `json:"version"`
			TerrainCount  int               `json:"terrain_count"`
			ReturnedCount int               `json:"returned_count"`
			Truncated     bool              `json:"truncated"`
			Terrains      []datfile.Terrain `json:"terrains"`
		}{Version: idx.Version, TerrainCount: len(idx.Terrains), ReturnedCount: len(terrains), Truncated: truncated, Terrains: terrains})
	case "terrain":
		if len(args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: kit dat terrain <empires*.dat> <terrain_id> [--json]")
			os.Exit(2)
		}
		for _, arg := range args[3:] {
			if arg != "--json" {
				die("kit dat", fmt.Errorf("unknown kit dat terrain option %q", arg))
			}
		}
		id, err := strconv.Atoi(args[2])
		if err != nil {
			die("kit dat", fmt.Errorf("invalid terrain id %q: %w", args[2], err))
		}
		idx, err := datfile.Open(args[1])
		if err != nil {
			die("kit dat", err)
		}
		if id < 0 || id >= len(idx.Terrains) {
			die("kit dat", fmt.Errorf("terrain %d is outside terrain table", id))
		}
		printJSON(idx.Terrains[id])
	case "terrain-patch", "patch-terrain":
		if len(args) < 4 {
			fmt.Fprintln(os.Stderr, "usage: kit dat terrain-patch <in.dat> <out.dat> <terrain_id> [--name TEXT] [--name-2 TEXT] [--overlay-mask-name TEXT]")
			os.Exit(2)
		}
		id, err := strconv.Atoi(args[3])
		if err != nil {
			die("kit dat terrain-patch", fmt.Errorf("invalid terrain id %q: %w", args[3], err))
		}
		recipe, err := parseDatTerrainPatchOptions(id, args[4:])
		if err != nil {
			die("kit dat terrain-patch", err)
		}
		report, err := datcodec.PatchRecipeFile(args[1], args[2], datcodec.Recipe{Terrains: []datcodec.TerrainPatchRecipe{recipe}})
		if err != nil {
			die("kit dat terrain-patch", err)
		}
		printJSON(struct {
			Version      string                      `json:"version"`
			Operation    string                      `json:"operation"`
			TerrainID    int                         `json:"terrain_id"`
			Request      datcodec.TerrainPatchRecipe `json:"request"`
			PatchReport  datcodec.PatchReport        `json:"patch_report"`
			Verification string                      `json:"verification"`
		}{
			Version:      datcodec.Version,
			Operation:    "terrain_patch",
			TerrainID:    id,
			Request:      recipe,
			PatchReport:  report,
			Verification: "structure_verified_not_engine_verified",
		})
	case "sounds":
		limit, all, err := parseDatListOptions(args[2:], "kit dat sounds")
		if err != nil {
			die("kit dat", err)
		}
		idx, err := datfile.Open(args[1])
		if err != nil {
			die("kit dat", err)
		}
		itemCount := 0
		for _, sound := range idx.Sounds {
			itemCount += len(sound.Items)
		}
		sounds := idx.Sounds
		truncated := false
		if !all && len(sounds) > limit {
			sounds = sounds[:limit]
			truncated = true
		}
		printJSON(struct {
			Version       string          `json:"version"`
			SoundCount    int             `json:"sound_count"`
			ItemCount     int             `json:"item_count"`
			ReturnedCount int             `json:"returned_count"`
			Truncated     bool            `json:"truncated"`
			Sounds        []datfile.Sound `json:"sounds"`
		}{Version: idx.Version, SoundCount: len(idx.Sounds), ItemCount: itemCount, ReturnedCount: len(sounds), Truncated: truncated, Sounds: sounds})
	case "sound":
		if len(args) != 3 {
			fmt.Fprintln(os.Stderr, "usage: kit dat sound <empires*.dat> <sound_id>")
			os.Exit(2)
		}
		id, err := strconv.Atoi(args[2])
		if err != nil {
			die("kit dat", fmt.Errorf("invalid sound id %q: %w", args[2], err))
		}
		idx, err := datfile.Open(args[1])
		if err != nil {
			die("kit dat", err)
		}
		if id < 0 || id >= len(idx.Sounds) {
			die("kit dat", fmt.Errorf("sound %d is outside sound table", id))
		}
		printJSON(idx.Sounds[id])
	case "sound-create", "create-sound":
		if len(args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: kit dat sound-create <in.dat> <out.dat> --from N [sound-flags] [--item file,resource,probability,civ,icon_set]")
			os.Exit(2)
		}
		recipe, err := parseDatSoundCreateOptions(args[3:])
		if err != nil {
			die("kit dat sound-create", err)
		}
		report, err := datcodec.PatchRecipeFile(args[1], args[2], datcodec.Recipe{CreateSound: &recipe})
		if err != nil {
			die("kit dat sound-create", err)
		}
		printJSON(struct {
			Version      string                     `json:"version"`
			Operation    string                     `json:"operation"`
			Request      datcodec.SoundCreateRecipe `json:"request"`
			PatchReport  datcodec.PatchReport       `json:"patch_report"`
			Verification string                     `json:"verification"`
		}{
			Version:      datcodec.Version,
			Operation:    "sound_create",
			Request:      recipe,
			PatchReport:  report,
			Verification: "structure_verified_not_engine_verified",
		})
	case "sound-patch", "patch-sound":
		if len(args) < 4 {
			fmt.Fprintln(os.Stderr, "usage: kit dat sound-patch <in.dat> <out.dat> <sound_id> [sound-flags] [--item INDEX,field=value...] [--remove-item N]")
			os.Exit(2)
		}
		id, err := strconv.Atoi(args[3])
		if err != nil {
			die("kit dat sound-patch", fmt.Errorf("invalid sound id %q: %w", args[3], err))
		}
		recipe, err := parseDatSoundPatchOptions(id, args[4:])
		if err != nil {
			die("kit dat sound-patch", err)
		}
		report, err := datcodec.PatchRecipeFile(args[1], args[2], datcodec.Recipe{Sounds: []datcodec.SoundPatchRecipe{recipe}})
		if err != nil {
			die("kit dat sound-patch", err)
		}
		printJSON(struct {
			Version      string                    `json:"version"`
			Operation    string                    `json:"operation"`
			SoundID      int                       `json:"sound_id"`
			Request      datcodec.SoundPatchRecipe `json:"request"`
			PatchReport  datcodec.PatchReport      `json:"patch_report"`
			Verification string                    `json:"verification"`
		}{
			Version:      datcodec.Version,
			Operation:    "sound_patch",
			SoundID:      id,
			Request:      recipe,
			PatchReport:  report,
			Verification: "structure_verified_not_engine_verified",
		})
	case "sound-delete", "delete-sound":
		if len(args) != 4 {
			fmt.Fprintln(os.Stderr, "usage: kit dat sound-delete <in.dat> <out.dat> <sound_id>")
			os.Exit(2)
		}
		id, err := strconv.Atoi(args[3])
		if err != nil {
			die("kit dat sound-delete", fmt.Errorf("invalid sound id %q: %w", args[3], err))
		}
		plan, err := datcodec.DeletePlanFile(args[1], datcodec.DeletePlanRequest{Section: "sound", ID: id})
		if err != nil {
			die("kit dat sound-delete", err)
		}
		if !plan.Supported || !plan.Mutates || len(plan.RecipeHint) == 0 {
			reason := plan.Reason
			if reason == "" {
				reason = "delete-plan did not produce an actionable recipe"
			}
			die("kit dat sound-delete", fmt.Errorf("refusing unsupported delete: strategy=%s reason=%s", plan.Strategy, reason))
		}
		var recipe datcodec.Recipe
		if err := json.Unmarshal(plan.RecipeHint, &recipe); err != nil {
			die("kit dat sound-delete", fmt.Errorf("delete-plan recipe hint did not decode: %w", err))
		}
		report, err := datcodec.PatchRecipeFile(args[1], args[2], recipe)
		if err != nil {
			die("kit dat sound-delete", err)
		}
		printJSON(struct {
			Version      string                    `json:"version"`
			Operation    string                    `json:"operation"`
			SoundID      int                       `json:"sound_id"`
			DeletePlan   datcodec.DeletePlanReport `json:"delete_plan"`
			PatchReport  datcodec.PatchReport      `json:"patch_report"`
			Verification string                    `json:"verification"`
		}{
			Version:      datcodec.Version,
			Operation:    "sound_delete",
			SoundID:      id,
			DeletePlan:   plan,
			PatchReport:  report,
			Verification: "structure_verified_not_engine_verified",
		})
	case "graphics":
		opts, err := parseDatGraphicsOptions(args[2:], "kit dat graphics")
		if err != nil {
			die("kit dat", err)
		}
		idx, err := datfile.Open(args[1])
		if err != nil {
			die("kit dat", err)
		}
		presentCount := len(idx.PresentGraphics())
		allGraphics := idx.PresentGraphicsFiltered(opts.Filter())
		graphics := allGraphics
		truncated := false
		if !opts.All && len(graphics) > opts.Limit {
			graphics = graphics[:opts.Limit]
			truncated = true
			fmt.Fprintf(os.Stderr, "WARNING: showing %d/%d; pass --limit 0 for all\n", len(graphics), len(allGraphics))
		}
		if opts.Spans {
			printJSON(struct {
				Version         string            `json:"version"`
				InflatedBytes   int               `json:"inflated_bytes"`
				GraphicsSize    int               `json:"graphics_size"`
				PresentCount    int               `json:"present_count"`
				MatchingCount   int               `json:"matching_count"`
				ReturnedCount   int               `json:"returned_count"`
				Truncated       bool              `json:"truncated"`
				PresentGraphics []datfile.Graphic `json:"present_graphics"`
			}{
				Version:         idx.Version,
				InflatedBytes:   idx.Inflated,
				GraphicsSize:    idx.GraphicsSize,
				PresentCount:    presentCount,
				MatchingCount:   len(allGraphics),
				ReturnedCount:   len(graphics),
				Truncated:       truncated,
				PresentGraphics: graphics,
			})
			return
		}
		summaries := make([]datfile.GraphicSummary, 0, len(graphics))
		for _, graphic := range graphics {
			summaries = append(summaries, graphic.Summary())
		}
		printJSON(struct {
			Version         string                   `json:"version"`
			InflatedBytes   int                      `json:"inflated_bytes"`
			GraphicsSize    int                      `json:"graphics_size"`
			PresentCount    int                      `json:"present_count"`
			MatchingCount   int                      `json:"matching_count"`
			ReturnedCount   int                      `json:"returned_count"`
			Truncated       bool                     `json:"truncated"`
			PresentGraphics []datfile.GraphicSummary `json:"present_graphics"`
		}{
			Version:         idx.Version,
			InflatedBytes:   idx.Inflated,
			GraphicsSize:    idx.GraphicsSize,
			PresentCount:    presentCount,
			MatchingCount:   len(allGraphics),
			ReturnedCount:   len(summaries),
			Truncated:       truncated,
			PresentGraphics: summaries,
		})
	case "graphic":
		if len(args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: kit dat graphic <empires*.dat> <graphic_id> [--spans|--raw]")
			os.Exit(2)
		}
		spans := false
		for _, arg := range args[3:] {
			switch arg {
			case "--spans", "--raw":
				spans = true
			default:
				die("kit dat", fmt.Errorf("unknown kit dat graphic option %q", arg))
			}
		}
		id, err := strconv.Atoi(args[2])
		if err != nil {
			die("kit dat", fmt.Errorf("invalid graphic id %q: %w", args[2], err))
		}
		idx, err := datfile.Open(args[1])
		if err != nil {
			die("kit dat", err)
		}
		graphic, ok := idx.Graphic(id)
		if !ok {
			die("kit dat", fmt.Errorf("graphic %d is absent or outside graphics table", id))
		}
		if spans {
			printJSON(graphic)
		} else {
			printJSON(graphic.Summary())
		}
	case "graphic-create", "create-graphic":
		if len(args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: kit dat graphic-create <in.dat> <out.dat> --from N [graphic scalar flags]")
			os.Exit(2)
		}
		recipe, err := parseDatGraphicCreateOptions(args[3:])
		if err != nil {
			die("kit dat graphic-create", err)
		}
		report, err := datcodec.PatchRecipeFile(args[1], args[2], datcodec.Recipe{CreateGraphic: &recipe})
		if err != nil {
			die("kit dat graphic-create", err)
		}
		printJSON(struct {
			Version      string                      `json:"version"`
			Operation    string                      `json:"operation"`
			Request      datfile.GraphicCreateRecipe `json:"request"`
			PatchReport  datcodec.PatchReport        `json:"patch_report"`
			Verification string                      `json:"verification"`
		}{
			Version:      datcodec.Version,
			Operation:    "graphic_create",
			Request:      recipe,
			PatchReport:  report,
			Verification: "structure_verified_not_engine_verified",
		})
	case "palette":
		opts, textOut, err := parseDatPaletteOptions(args[2:])
		if err != nil {
			die("kit dat palette", err)
		}
		report, err := datfile.PaletteFile(args[1], opts)
		if err != nil {
			die("kit dat palette", err)
		}
		if textOut {
			printDatPalette(report)
		} else {
			printJSON(report)
		}
	case "effects":
		opts, err := parseDatEffectListOptions(args[2:], "kit dat effects")
		if err != nil {
			die("kit dat", err)
		}
		idx, err := datfile.Open(args[1])
		if err != nil {
			die("kit dat", err)
		}
		allEffects := compactEffects(idx.Effects, opts)
		effects := allEffects
		truncated := false
		if !opts.All && len(effects) > opts.Limit {
			effects = effects[:opts.Limit]
			truncated = true
		}
		printJSON(struct {
			Version       string              `json:"version"`
			EffectCount   int                 `json:"effect_count"`
			FilteredCount int                 `json:"filtered_count"`
			ReturnedCount int                 `json:"returned_count"`
			Truncated     bool                `json:"truncated"`
			Effects       []effectListSummary `json:"effects"`
		}{Version: idx.Version, EffectCount: len(idx.Effects), FilteredCount: len(allEffects), ReturnedCount: len(effects), Truncated: truncated, Effects: effects})
	case "effect":
		if len(args) != 3 {
			fmt.Fprintln(os.Stderr, "usage: kit dat effect <empires*.dat> <effect_id>")
			os.Exit(2)
		}
		id, err := strconv.Atoi(args[2])
		if err != nil {
			die("kit dat", fmt.Errorf("invalid effect id %q: %w", args[2], err))
		}
		idx, err := datfile.Open(args[1])
		if err != nil {
			die("kit dat", err)
		}
		effect, ok := idx.Effect(id)
		if !ok {
			die("kit dat", fmt.Errorf("effect %d is outside effects table", id))
		}
		printJSON(effect)
	case "effect-explain", "explain-effect":
		if len(args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: kit dat effect-explain <empires*.dat> <effect_id> [--text|--json]")
			os.Exit(2)
		}
		id, err := strconv.Atoi(args[2])
		if err != nil {
			die("kit dat effect-explain", fmt.Errorf("invalid effect id %q: %w", args[2], err))
		}
		text, err := parseTextJSONOptions(args[3:], "kit dat effect-explain")
		if err != nil {
			die("kit dat effect-explain", err)
		}
		idx, err := datfile.Open(args[1])
		if err != nil {
			die("kit dat effect-explain", err)
		}
		report, err := datcodec.ExplainEffect(idx, id)
		if err != nil {
			die("kit dat effect-explain", err)
		}
		if text {
			printDatEffectExplain(report)
		} else {
			printJSON(report)
		}
	case "effect-create", "create-effect":
		if len(args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: kit dat effect-create <in.dat> <out.dat> --name TEXT [--from-effect N] [--command type,a,b,c,d|json] [--append-command type,a,b,c,d|json] [--remove-command N]")
			os.Exit(2)
		}
		recipe, err := parseDatEffectCreateOptions(args[3:])
		if err != nil {
			die("kit dat effect-create", err)
		}
		report, err := datcodec.PatchRecipeFile(args[1], args[2], datcodec.Recipe{CreateEffect: &recipe})
		if err != nil {
			die("kit dat effect-create", err)
		}
		printJSON(struct {
			Version        string                      `json:"version"`
			Operation      string                      `json:"operation"`
			Request        datcodec.EffectCreateRecipe `json:"request"`
			AuthoringNotes []string                    `json:"authoring_notes,omitempty"`
			PatchReport    datcodec.PatchReport        `json:"patch_report"`
			Verification   string                      `json:"verification"`
		}{
			Version:        datcodec.Version,
			Operation:      "effect_create",
			Request:        recipe,
			AuthoringNotes: datcodec.RecipeAuthoringWarnings(datcodec.Recipe{CreateEffect: &recipe}),
			PatchReport:    report,
			Verification:   "structure_verified_not_engine_verified",
		})
	case "effect-patch", "patch-effect":
		if len(args) < 4 {
			fmt.Fprintln(os.Stderr, "usage: kit dat effect-patch <in.dat> <out.dat> <effect_id> [--name TEXT] [--clear-commands] [--command type,a,b,c,d|json] [--append-command type,a,b,c,d|json] [--remove-command N]")
			os.Exit(2)
		}
		id, err := strconv.Atoi(args[3])
		if err != nil {
			die("kit dat effect-patch", fmt.Errorf("invalid effect id %q: %w", args[3], err))
		}
		recipe, err := parseDatEffectPatchOptions(id, args[4:])
		if err != nil {
			die("kit dat effect-patch", err)
		}
		report, err := datcodec.PatchRecipeFile(args[1], args[2], datcodec.Recipe{Effects: []datcodec.EffectPatchRecipe{recipe}})
		if err != nil {
			die("kit dat effect-patch", err)
		}
		printJSON(struct {
			Version        string                     `json:"version"`
			Operation      string                     `json:"operation"`
			EffectID       int                        `json:"effect_id"`
			Request        datcodec.EffectPatchRecipe `json:"request"`
			AuthoringNotes []string                   `json:"authoring_notes,omitempty"`
			PatchReport    datcodec.PatchReport       `json:"patch_report"`
			Verification   string                     `json:"verification"`
		}{
			Version:        datcodec.Version,
			Operation:      "effect_patch",
			EffectID:       id,
			Request:        recipe,
			AuthoringNotes: datcodec.RecipeAuthoringWarnings(datcodec.Recipe{Effects: []datcodec.EffectPatchRecipe{recipe}}),
			PatchReport:    report,
			Verification:   "structure_verified_not_engine_verified",
		})
	case "effect-disable", "disable-effect":
		if len(args) != 4 {
			fmt.Fprintln(os.Stderr, "usage: kit dat effect-disable <in.dat> <out.dat> <effect_id>")
			os.Exit(2)
		}
		id, err := strconv.Atoi(args[3])
		if err != nil {
			die("kit dat effect-disable", fmt.Errorf("invalid effect id %q: %w", args[3], err))
		}
		report, err := datcodec.PatchRecipeFile(args[1], args[2], datcodec.Recipe{DisableEffects: []int{id}})
		if err != nil {
			die("kit dat effect-disable", err)
		}
		printJSON(struct {
			Version      string               `json:"version"`
			Operation    string               `json:"operation"`
			EffectID     int                  `json:"effect_id"`
			PatchReport  datcodec.PatchReport `json:"patch_report"`
			Verification string               `json:"verification"`
		}{
			Version:      datcodec.Version,
			Operation:    "effect_disable",
			EffectID:     id,
			PatchReport:  report,
			Verification: "structure_verified_not_engine_verified",
		})
	case "effect-delete", "delete-effect":
		if len(args) != 4 {
			fmt.Fprintln(os.Stderr, "usage: kit dat effect-delete <in.dat> <out.dat> <effect_id>")
			os.Exit(2)
		}
		id, err := strconv.Atoi(args[3])
		if err != nil {
			die("kit dat effect-delete", fmt.Errorf("invalid effect id %q: %w", args[3], err))
		}
		plan, err := datcodec.DeletePlanFile(args[1], datcodec.DeletePlanRequest{Section: "effect", ID: id})
		if err != nil {
			die("kit dat effect-delete", err)
		}
		if !plan.Supported || !plan.Mutates || len(plan.RecipeHint) == 0 {
			reason := plan.Reason
			if reason == "" {
				reason = "delete-plan did not produce an actionable recipe"
			}
			die("kit dat effect-delete", fmt.Errorf("refusing unsupported delete: strategy=%s reason=%s", plan.Strategy, reason))
		}
		var recipe datcodec.Recipe
		if err := json.Unmarshal(plan.RecipeHint, &recipe); err != nil {
			die("kit dat effect-delete", fmt.Errorf("delete-plan recipe hint did not decode: %w", err))
		}
		report, err := datcodec.PatchRecipeFile(args[1], args[2], recipe)
		if err != nil {
			die("kit dat effect-delete", err)
		}
		printJSON(struct {
			Version      string                    `json:"version"`
			Operation    string                    `json:"operation"`
			EffectID     int                       `json:"effect_id"`
			DeletePlan   datcodec.DeletePlanReport `json:"delete_plan"`
			PatchReport  datcodec.PatchReport      `json:"patch_report"`
			Verification string                    `json:"verification"`
		}{
			Version:      datcodec.Version,
			Operation:    "effect_delete",
			EffectID:     id,
			DeletePlan:   plan,
			PatchReport:  report,
			Verification: "structure_verified_not_engine_verified",
		})
	case "command-matrix", "effect-command-matrix":
		opts, err := parseDatCommandMatrixOptions(args[2:])
		if err != nil {
			die("kit dat command-matrix", err)
		}
		idx, err := datfile.Open(args[1])
		if err != nil {
			die("kit dat command-matrix", err)
		}
		report := datcodec.EffectCommandMatrixWithOptions(idx, datcodec.EffectCommandMatrixOptions{
			ExampleLimit:   opts.ExampleLimit,
			CommandType:    opts.CommandType,
			Operand:        opts.Operand,
			ReferenceKind:  opts.ReferenceKind,
			ReferenceField: opts.ReferenceField,
			UnknownOnly:    opts.UnknownOnly,
			TypedOnly:      opts.TypedOnly,
			CandidateOnly:  opts.CandidateOnly,
		})
		if opts.Text {
			printDatEffectCommandMatrix(report)
		} else {
			printJSON(report)
		}
	case "techs":
		opts, err := parseDatTechListOptions(args[2:])
		if err != nil {
			die("kit dat", err)
		}
		idx, err := datfile.Open(args[1])
		if err != nil {
			die("kit dat", err)
		}
		allTechs := collectTechs(idx, opts)
		techs := compactTechs(allTechs)
		truncated := false
		if !opts.All && len(techs) > opts.Limit {
			techs = techs[:opts.Limit]
			truncated = true
		}
		printJSON(struct {
			Version       string             `json:"version"`
			TechCount     int                `json:"tech_count"`
			Filters       datTechListOptions `json:"filters"`
			FilteredCount int                `json:"filtered_count"`
			ReturnedCount int                `json:"returned_count"`
			Truncated     bool               `json:"truncated"`
			Techs         []techListSummary  `json:"techs"`
		}{Version: idx.Version, TechCount: len(idx.Techs), Filters: opts, FilteredCount: len(allTechs), ReturnedCount: len(techs), Truncated: truncated, Techs: techs})
	case "tech":
		if len(args) != 3 {
			fmt.Fprintln(os.Stderr, "usage: kit dat tech <empires*.dat> <tech_id>")
			os.Exit(2)
		}
		id, err := strconv.Atoi(args[2])
		if err != nil {
			die("kit dat", fmt.Errorf("invalid tech id %q: %w", args[2], err))
		}
		idx, err := datfile.Open(args[1])
		if err != nil {
			die("kit dat", err)
		}
		tech, ok := idx.Tech(id)
		if !ok {
			die("kit dat", fmt.Errorf("tech %d is outside tech table", id))
		}
		printJSON(tech)
	case "tech-explain", "explain-tech":
		if len(args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: kit dat tech-explain <empires*.dat> <tech_id> [--text|--json]")
			os.Exit(2)
		}
		id, err := strconv.Atoi(args[2])
		if err != nil {
			die("kit dat tech-explain", fmt.Errorf("invalid tech id %q: %w", args[2], err))
		}
		text, err := parseTextJSONOptions(args[3:], "kit dat tech-explain")
		if err != nil {
			die("kit dat tech-explain", err)
		}
		idx, err := datfile.Open(args[1])
		if err != nil {
			die("kit dat tech-explain", err)
		}
		report, err := datcodec.ExplainTech(idx, id)
		if err != nil {
			die("kit dat tech-explain", err)
		}
		if text {
			printDatTechExplain(report)
		} else {
			printJSON(report)
		}
	case "tech-create", "create-tech":
		if len(args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: kit dat tech-create <in.dat> <out.dat> --from N [tech-flags]")
			os.Exit(2)
		}
		recipe, err := parseDatTechCreateOptions(args[3:])
		if err != nil {
			die("kit dat tech-create", err)
		}
		report, err := datcodec.PatchRecipeFile(args[1], args[2], datcodec.Recipe{CreateTech: &recipe})
		if err != nil {
			die("kit dat tech-create", err)
		}
		printJSON(struct {
			Version      string                    `json:"version"`
			Operation    string                    `json:"operation"`
			Request      datcodec.TechCreateRecipe `json:"request"`
			PatchReport  datcodec.PatchReport      `json:"patch_report"`
			Verification string                    `json:"verification"`
		}{
			Version:      datcodec.Version,
			Operation:    "tech_create",
			Request:      recipe,
			PatchReport:  report,
			Verification: "structure_verified_not_engine_verified",
		})
	case "tech-patch", "patch-tech":
		if len(args) < 4 {
			fmt.Fprintln(os.Stderr, "usage: kit dat tech-patch <in.dat> <out.dat> <tech_id> [tech-flags]")
			os.Exit(2)
		}
		id, err := strconv.Atoi(args[3])
		if err != nil {
			die("kit dat tech-patch", fmt.Errorf("invalid tech id %q: %w", args[3], err))
		}
		recipe, err := parseDatTechPatchOptions(id, args[4:])
		if err != nil {
			die("kit dat tech-patch", err)
		}
		report, err := datcodec.PatchRecipeFile(args[1], args[2], datcodec.Recipe{Techs: []datcodec.TechPatchRecipe{recipe}})
		if err != nil {
			die("kit dat tech-patch", err)
		}
		printJSON(struct {
			Version      string                   `json:"version"`
			Operation    string                   `json:"operation"`
			TechID       int                      `json:"tech_id"`
			Request      datcodec.TechPatchRecipe `json:"request"`
			PatchReport  datcodec.PatchReport     `json:"patch_report"`
			Verification string                   `json:"verification"`
		}{
			Version:      datcodec.Version,
			Operation:    "tech_patch",
			TechID:       id,
			Request:      recipe,
			PatchReport:  report,
			Verification: "structure_verified_not_engine_verified",
		})
	case "tech-delete", "delete-tech":
		if len(args) != 4 {
			fmt.Fprintln(os.Stderr, "usage: kit dat tech-delete <in.dat> <out.dat> <tech_id>")
			os.Exit(2)
		}
		id, err := strconv.Atoi(args[3])
		if err != nil {
			die("kit dat tech-delete", fmt.Errorf("invalid tech id %q: %w", args[3], err))
		}
		plan, err := datcodec.DeletePlanFile(args[1], datcodec.DeletePlanRequest{Section: "tech", ID: id})
		if err != nil {
			die("kit dat tech-delete", err)
		}
		if !plan.Supported || !plan.Mutates || len(plan.RecipeHint) == 0 {
			reason := plan.Reason
			if reason == "" {
				reason = "delete-plan did not produce an actionable recipe"
			}
			die("kit dat tech-delete", fmt.Errorf("refusing unsupported delete: strategy=%s reason=%s", plan.Strategy, reason))
		}
		var recipe datcodec.Recipe
		if err := json.Unmarshal(plan.RecipeHint, &recipe); err != nil {
			die("kit dat tech-delete", fmt.Errorf("delete-plan recipe hint did not decode: %w", err))
		}
		report, err := datcodec.PatchRecipeFile(args[1], args[2], recipe)
		if err != nil {
			die("kit dat tech-delete", err)
		}
		printJSON(struct {
			Version      string                    `json:"version"`
			Operation    string                    `json:"operation"`
			TechID       int                       `json:"tech_id"`
			DeletePlan   datcodec.DeletePlanReport `json:"delete_plan"`
			PatchReport  datcodec.PatchReport      `json:"patch_report"`
			Verification string                    `json:"verification"`
		}{
			Version:      datcodec.Version,
			Operation:    "tech_delete",
			TechID:       id,
			DeletePlan:   plan,
			PatchReport:  report,
			Verification: "structure_verified_not_engine_verified",
		})
	case "abilities":
		opts, err := parseDatAbilityListOptions(args[2:], "kit dat abilities")
		if err != nil {
			die("kit dat abilities", err)
		}
		idx, err := datfile.Open(args[1])
		if err != nil {
			die("kit dat abilities", err)
		}
		allAbilities := collectAbilities(idx, opts)
		abilities := allAbilities
		truncated := false
		if !opts.All && len(abilities) > opts.Limit {
			abilities = abilities[:opts.Limit]
			truncated = true
		}
		printJSON(struct {
			Version       string                 `json:"version"`
			TechCount     int                    `json:"tech_count"`
			EffectCount   int                    `json:"effect_count"`
			Filters       datAbilityListOptions  `json:"filters"`
			FilteredCount int                    `json:"filtered_count"`
			ReturnedCount int                    `json:"returned_count"`
			Truncated     bool                   `json:"truncated"`
			Abilities     []abilityListSummary   `json:"abilities"`
			Verification  aoe2.VerificationClaim `json:"verification"`
		}{
			Version:       idx.Version,
			TechCount:     len(idx.Techs),
			EffectCount:   len(idx.Effects),
			Filters:       opts,
			FilteredCount: len(allAbilities),
			ReturnedCount: len(abilities),
			Truncated:     truncated,
			Abilities:     abilities,
			Verification:  datAbilityVerification(),
		})
	case "ability":
		if len(args) != 3 {
			fmt.Fprintln(os.Stderr, "usage: kit dat ability <empires*.dat> <tech_id>")
			os.Exit(2)
		}
		id, err := strconv.Atoi(args[2])
		if err != nil {
			die("kit dat ability", fmt.Errorf("invalid tech id %q: %w", args[2], err))
		}
		idx, err := datfile.Open(args[1])
		if err != nil {
			die("kit dat ability", err)
		}
		if id < 0 || id >= len(idx.Techs) {
			die("kit dat ability", fmt.Errorf("tech %d is outside tech table", id))
		}
		tech := idx.Techs[id]
		if tech.EffectID < 0 || int(tech.EffectID) >= len(idx.Effects) {
			die("kit dat ability", fmt.Errorf("tech %d does not reference a valid effect_id: %d", id, tech.EffectID))
		}
		effect := idx.Effects[tech.EffectID]
		printJSON(struct {
			Version      string                 `json:"version"`
			Tech         datfile.Tech           `json:"tech"`
			Effect       datfile.Effect         `json:"effect"`
			Summary      abilityListSummary     `json:"summary"`
			Verification aoe2.VerificationClaim `json:"verification"`
		}{
			Version:      idx.Version,
			Tech:         tech,
			Effect:       effect,
			Summary:      abilitySummary(tech, effect, nil),
			Verification: datAbilityVerification(),
		})
	case "ability-create", "create-ability":
		if len(args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: kit dat ability-create <in.dat> <out.dat> --from-tech N --name TEXT [--effect-name TEXT] [--from-effect N] [--command type,a,b,c,d|json] [--append-command type,a,b,c,d|json] [tech flags]")
			os.Exit(2)
		}
		recipe, err := parseDatAbilityCreateOptions(args[3:])
		if err != nil {
			die("kit dat ability-create", err)
		}
		report, err := datcodec.PatchRecipeFile(args[1], args[2], datcodec.Recipe{CreateAbility: &recipe})
		if err != nil {
			die("kit dat ability-create", err)
		}
		printJSON(struct {
			Version        string                       `json:"version"`
			Operation      string                       `json:"operation"`
			Request        datcodec.AbilityCreateRecipe `json:"request"`
			AuthoringNotes []string                     `json:"authoring_notes,omitempty"`
			PatchReport    datcodec.PatchReport         `json:"patch_report"`
			Verification   string                       `json:"verification"`
		}{
			Version:        datcodec.Version,
			Operation:      "ability_create",
			Request:        recipe,
			AuthoringNotes: datcodec.RecipeAuthoringWarnings(datcodec.Recipe{CreateAbility: &recipe}),
			PatchReport:    report,
			Verification:   "structure_verified_not_engine_verified",
		})
	case "ability-patch", "patch-ability":
		if len(args) < 4 {
			fmt.Fprintln(os.Stderr, "usage: kit dat ability-patch <in.dat> <out.dat> <tech_id> [--name TEXT] [--effect-name TEXT] [--clear-commands] [--command type,a,b,c,d|json] [--append-command type,a,b,c,d|json] [--remove-command N] [tech flags]")
			os.Exit(2)
		}
		techID, err := strconv.Atoi(args[3])
		if err != nil {
			die("kit dat ability-patch", fmt.Errorf("invalid tech id %q: %w", args[3], err))
		}
		recipe, err := parseDatAbilityPatchOptions(args[1], techID, args[4:])
		if err != nil {
			die("kit dat ability-patch", err)
		}
		report, err := datcodec.PatchRecipeFile(args[1], args[2], recipe)
		if err != nil {
			die("kit dat ability-patch", err)
		}
		printJSON(struct {
			Version        string               `json:"version"`
			Operation      string               `json:"operation"`
			TechID         int                  `json:"tech_id"`
			Request        datcodec.Recipe      `json:"request"`
			AuthoringNotes []string             `json:"authoring_notes,omitempty"`
			PatchReport    datcodec.PatchReport `json:"patch_report"`
			Verification   string               `json:"verification"`
		}{
			Version:        datcodec.Version,
			Operation:      "ability_patch",
			TechID:         techID,
			Request:        recipe,
			AuthoringNotes: datcodec.RecipeAuthoringWarnings(recipe),
			PatchReport:    report,
			Verification:   "structure_verified_not_engine_verified",
		})
	case "ability-disable", "disable-ability":
		if len(args) != 4 {
			fmt.Fprintln(os.Stderr, "usage: kit dat ability-disable <in.dat> <out.dat> <tech_id>")
			os.Exit(2)
		}
		techID, err := strconv.Atoi(args[3])
		if err != nil {
			die("kit dat ability-disable", fmt.Errorf("invalid tech id %q: %w", args[3], err))
		}
		report, err := datcodec.PatchRecipeFile(args[1], args[2], datcodec.Recipe{DisableAbility: &techID})
		if err != nil {
			die("kit dat ability-disable", err)
		}
		printJSON(struct {
			Version      string               `json:"version"`
			Operation    string               `json:"operation"`
			TechID       int                  `json:"tech_id"`
			PatchReport  datcodec.PatchReport `json:"patch_report"`
			Verification string               `json:"verification"`
		}{
			Version:      datcodec.Version,
			Operation:    "ability_disable",
			TechID:       techID,
			PatchReport:  report,
			Verification: "structure_verified_not_engine_verified",
		})
	case "ability-delete", "delete-ability":
		if len(args) != 4 {
			fmt.Fprintln(os.Stderr, "usage: kit dat ability-delete <in.dat> <out.dat> <tech_id>")
			os.Exit(2)
		}
		techID, err := strconv.Atoi(args[3])
		if err != nil {
			die("kit dat ability-delete", fmt.Errorf("invalid tech id %q: %w", args[3], err))
		}
		report, err := datcodec.PatchRecipeFile(args[1], args[2], datcodec.Recipe{DeleteAbility: &techID})
		if err != nil {
			die("kit dat ability-delete", err)
		}
		printJSON(struct {
			Version      string               `json:"version"`
			Operation    string               `json:"operation"`
			TechID       int                  `json:"tech_id"`
			PatchReport  datcodec.PatchReport `json:"patch_report"`
			Verification string               `json:"verification"`
		}{
			Version:      datcodec.Version,
			Operation:    "ability_delete",
			TechID:       techID,
			PatchReport:  report,
			Verification: "structure_verified_not_engine_verified",
		})
	case "tech-tree":
		full, err := parseDatTechTreeOptions(args[2:])
		if err != nil {
			die("kit dat", err)
		}
		idx, err := datfile.Open(args[1])
		if err != nil {
			die("kit dat", err)
		}
		if full {
			printJSON(struct {
				Version     string           `json:"version"`
				GameMetrics datfile.Metrics  `json:"game_metrics"`
				TechTree    datfile.TechTree `json:"tech_tree"`
			}{Version: idx.Version, GameMetrics: idx.GameMetrics, TechTree: idx.TechTree})
		} else {
			printJSON(techTreeSummary(idx))
		}
	case "tech-tree-connection-create":
		if len(args) < 5 {
			fmt.Fprintln(os.Stderr, "usage: kit dat tech-tree-connection-create <in.dat> <out.dat> <building|unit|research> <from_index> [connection flags]")
			os.Exit(2)
		}
		family, err := normalizeTechTreeConnectionFamily(args[3])
		if err != nil {
			die("kit dat tech-tree-connection-create", err)
		}
		from, err := strconv.Atoi(args[4])
		if err != nil {
			die("kit dat tech-tree-connection-create", fmt.Errorf("invalid source index %q: %w", args[4], err))
		}
		recipe, request, err := parseDatTechTreeConnectionCreateOptions(family, from, args[5:])
		if err != nil {
			die("kit dat tech-tree-connection-create", err)
		}
		report, err := datcodec.PatchRecipeFile(args[1], args[2], datcodec.Recipe{TechTree: &recipe})
		if err != nil {
			die("kit dat tech-tree-connection-create", err)
		}
		printJSON(struct {
			Version      string               `json:"version"`
			Operation    string               `json:"operation"`
			Family       string               `json:"family"`
			From         int                  `json:"from"`
			Request      any                  `json:"request"`
			PatchReport  datcodec.PatchReport `json:"patch_report"`
			Verification string               `json:"verification"`
		}{
			Version:      datcodec.Version,
			Operation:    "tech_tree_connection_create",
			Family:       family,
			From:         from,
			Request:      request,
			PatchReport:  report,
			Verification: "structure_verified_not_engine_verified",
		})
	case "tech-tree-connection-patch":
		if len(args) < 5 {
			fmt.Fprintln(os.Stderr, "usage: kit dat tech-tree-connection-patch <in.dat> <out.dat> <building|unit|research> <index> [connection flags]")
			os.Exit(2)
		}
		family, err := normalizeTechTreeConnectionFamily(args[3])
		if err != nil {
			die("kit dat tech-tree-connection-patch", err)
		}
		index, err := strconv.Atoi(args[4])
		if err != nil {
			die("kit dat tech-tree-connection-patch", fmt.Errorf("invalid connection index %q: %w", args[4], err))
		}
		recipe, request, err := parseDatTechTreeConnectionPatchOptions(family, index, args[5:])
		if err != nil {
			die("kit dat tech-tree-connection-patch", err)
		}
		report, err := datcodec.PatchRecipeFile(args[1], args[2], datcodec.Recipe{TechTree: &recipe})
		if err != nil {
			die("kit dat tech-tree-connection-patch", err)
		}
		printJSON(struct {
			Version      string               `json:"version"`
			Operation    string               `json:"operation"`
			Family       string               `json:"family"`
			Index        int                  `json:"index"`
			Request      any                  `json:"request"`
			PatchReport  datcodec.PatchReport `json:"patch_report"`
			Verification string               `json:"verification"`
		}{
			Version:      datcodec.Version,
			Operation:    "tech_tree_connection_patch",
			Family:       family,
			Index:        index,
			Request:      request,
			PatchReport:  report,
			Verification: "structure_verified_not_engine_verified",
		})
	case "tech-tree-connection-delete":
		if len(args) != 5 {
			fmt.Fprintln(os.Stderr, "usage: kit dat tech-tree-connection-delete <in.dat> <out.dat> <building|unit|research> <index>")
			os.Exit(2)
		}
		family, err := normalizeTechTreeConnectionFamily(args[3])
		if err != nil {
			die("kit dat tech-tree-connection-delete", err)
		}
		index, err := strconv.Atoi(args[4])
		if err != nil {
			die("kit dat tech-tree-connection-delete", fmt.Errorf("invalid connection index %q: %w", args[4], err))
		}
		recipe, err := datTechTreeConnectionDeleteRecipe(family, index)
		if err != nil {
			die("kit dat tech-tree-connection-delete", err)
		}
		report, err := datcodec.PatchRecipeFile(args[1], args[2], datcodec.Recipe{TechTree: &recipe})
		if err != nil {
			die("kit dat tech-tree-connection-delete", err)
		}
		printJSON(struct {
			Version      string               `json:"version"`
			Operation    string               `json:"operation"`
			Family       string               `json:"family"`
			Index        int                  `json:"index"`
			PatchReport  datcodec.PatchReport `json:"patch_report"`
			Verification string               `json:"verification"`
		}{
			Version:      datcodec.Version,
			Operation:    "tech_tree_connection_delete",
			Family:       family,
			Index:        index,
			PatchReport:  report,
			Verification: "structure_verified_not_engine_verified",
		})
	case "units":
		unitFilters, err := parseDatUnitsOptions(args[2:])
		if err != nil {
			die("kit dat", err)
		}
		if report, ok, err := cachedDatUnitsReport(args[1], unitFilters); err == nil && ok {
			printJSON(report)
			return
		}
		idx, err := datfile.Open(args[1])
		if err != nil {
			die("kit dat", err)
		}
		allUnits := collectUnits(idx, unitFilters)
		units := allUnits
		truncated := false
		if !unitFilters.All && len(units) > unitFilters.Limit {
			units = units[:unitFilters.Limit]
			truncated = true
		}
		printJSON(datUnitsReport{
			Version:       idx.Version,
			CivCount:      len(idx.Civs),
			Filters:       unitFilters,
			PresentCount:  len(allUnits),
			ReturnedCount: len(units),
			Truncated:     truncated,
			ClassNote:     unitFilters.classNote(),
			Units:         compactUnits(units),
		})
	case "unit":
		if len(args) != 4 {
			fmt.Fprintln(os.Stderr, "usage: kit dat unit <empires*.dat> <civ_id> <unit_id>")
			os.Exit(2)
		}
		civID, err := strconv.Atoi(args[2])
		if err != nil {
			die("kit dat", fmt.Errorf("invalid civ id %q: %w", args[2], err))
		}
		unitID, err := strconv.Atoi(args[3])
		if err != nil {
			die("kit dat", fmt.Errorf("invalid unit id %q: %w", args[3], err))
		}
		if unit, ok, err := cachedDatUnit(args[1], civID, unitID); err == nil && ok {
			if unit.FullFields != nil {
				printJSON(unit)
				return
			}
		}
		idx, err := datfile.Open(args[1])
		if err != nil {
			die("kit dat", err)
		}
		if civID < 0 || civID >= len(idx.Civs) {
			die("kit dat", fmt.Errorf("civ %d is outside civ table", civID))
		}
		civ := idx.Civs[civID]
		if unitID < 0 || unitID >= len(civ.Units) {
			die("kit dat", fmt.Errorf("unit %d is outside civ %d unit table", unitID, civID))
		}
		unit, _, err := datfile.DecodeUnitFile(args[1], civID, unitID)
		if err != nil {
			die("kit dat", err)
		}
		printJSON(unit)
	case "unit-create", "create-unit":
		if len(args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: kit dat unit-create <in.dat> <out.dat> --from-civ N --from-unit N (--civ N|--all-civs) [unit patch flags]")
			os.Exit(2)
		}
		recipe, err := parseDatUnitCreateOptions(args[3:])
		if err != nil {
			die("kit dat unit-create", err)
		}
		report, err := datcodec.PatchRecipeFile(args[1], args[2], datcodec.Recipe{CreateUnit: &recipe})
		if err != nil {
			die("kit dat unit-create", err)
		}
		printJSON(struct {
			Version      string                   `json:"version"`
			Operation    string                   `json:"operation"`
			Request      datfile.UnitCreateRecipe `json:"request"`
			PatchReport  datcodec.PatchReport     `json:"patch_report"`
			Verification string                   `json:"verification"`
		}{
			Version:      datcodec.Version,
			Operation:    "unit_create",
			Request:      recipe,
			PatchReport:  report,
			Verification: "structure_verified_not_engine_verified",
		})
	case "unit-delete", "delete-unit":
		if len(args) < 4 {
			fmt.Fprintln(os.Stderr, "usage: kit dat unit-delete <in.dat> <out.dat> <unit_id> (--civ N|--all-civs)")
			os.Exit(2)
		}
		id, err := strconv.Atoi(args[3])
		if err != nil {
			die("kit dat unit-delete", fmt.Errorf("invalid unit id %q: %w", args[3], err))
		}
		request, err := parseDatUnitDeleteRequest(id, args[4:])
		if err != nil {
			die("kit dat unit-delete", err)
		}
		plan, err := datcodec.DeletePlanFile(args[1], request)
		if err != nil {
			die("kit dat unit-delete", err)
		}
		if !plan.Supported || !plan.Mutates || len(plan.RecipeHint) == 0 {
			reason := plan.Reason
			if reason == "" {
				reason = "delete-plan did not produce an actionable recipe"
			}
			die("kit dat unit-delete", fmt.Errorf("refusing unsupported delete: strategy=%s reason=%s", plan.Strategy, reason))
		}
		var recipe datcodec.Recipe
		if err := json.Unmarshal(plan.RecipeHint, &recipe); err != nil {
			die("kit dat unit-delete", fmt.Errorf("delete-plan recipe hint did not decode: %w", err))
		}
		report, err := datcodec.PatchRecipeFile(args[1], args[2], recipe)
		if err != nil {
			die("kit dat unit-delete", err)
		}
		printJSON(struct {
			Version      string                    `json:"version"`
			Operation    string                    `json:"operation"`
			UnitID       int                       `json:"unit_id"`
			DeletePlan   datcodec.DeletePlanReport `json:"delete_plan"`
			PatchReport  datcodec.PatchReport      `json:"patch_report"`
			Verification string                    `json:"verification"`
		}{
			Version:      datcodec.Version,
			Operation:    "unit_delete",
			UnitID:       id,
			DeletePlan:   plan,
			PatchReport:  report,
			Verification: "structure_verified_not_engine_verified",
		})
	case "availability":
		if len(args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: kit dat availability <empires*.dat> <unit_id> [--civ N|--all-civs]")
			os.Exit(2)
		}
		unitID, err := strconv.Atoi(args[2])
		if err != nil {
			die("kit dat availability", fmt.Errorf("invalid unit id %q: %w", args[2], err))
		}
		request, err := parseDatAvailabilityRequest(unitID, args[3:])
		if err != nil {
			die("kit dat availability", err)
		}
		report, err := datcodec.UnitAvailabilityFile(args[1], request)
		if err != nil {
			die("kit dat availability", err)
		}
		printJSON(report)
	case "availability-set", "set-availability":
		if len(args) < 4 {
			fmt.Fprintln(os.Stderr, "usage: kit dat availability-set <in.dat> <out.dat> <unit_id> (--civ N|--all-civs) --enabled true|false|1|0")
			os.Exit(2)
		}
		unitID, err := strconv.Atoi(args[3])
		if err != nil {
			die("kit dat availability-set", fmt.Errorf("invalid unit id %q: %w", args[3], err))
		}
		recipe, err := parseDatAvailabilitySetOptions(unitID, args[4:])
		if err != nil {
			die("kit dat availability-set", err)
		}
		report, err := datcodec.PatchRecipeFile(args[1], args[2], datcodec.Recipe{UnitAvailability: []datcodec.UnitAvailabilityRecipe{recipe}})
		if err != nil {
			die("kit dat availability-set", err)
		}
		printJSON(struct {
			Version      string                          `json:"version"`
			Operation    string                          `json:"operation"`
			UnitID       int                             `json:"unit_id"`
			Request      datcodec.UnitAvailabilityRecipe `json:"request"`
			PatchReport  datcodec.PatchReport            `json:"patch_report"`
			Verification string                          `json:"verification"`
		}{
			Version:      datcodec.Version,
			Operation:    "availability_set",
			UnitID:       unitID,
			Request:      recipe,
			PatchReport:  report,
			Verification: "structure_verified_not_engine_verified",
		})
	case "patch-graphic", "graphic-patch":
		if len(args) < 4 {
			fmt.Fprintln(os.Stderr, "usage: kit dat patch-graphic <in.dat> <out.dat> <graphic_id> [graphic scalar flags]")
			os.Exit(2)
		}
		id, err := strconv.Atoi(args[3])
		if err != nil {
			die("kit dat", fmt.Errorf("invalid graphic id %q: %w", args[3], err))
		}
		patch, err := parseGraphicPatchOptions(args[4:])
		if err != nil {
			die("kit dat", err)
		}
		report, err := datfile.PatchGraphicFile(args[1], args[2], id, patch)
		if err != nil {
			die("kit dat", err)
		}
		printJSON(report)
	case "graphic-delete", "delete-graphic":
		if len(args) != 4 {
			fmt.Fprintln(os.Stderr, "usage: kit dat graphic-delete <in.dat> <out.dat> <graphic_id>")
			os.Exit(2)
		}
		id, err := strconv.Atoi(args[3])
		if err != nil {
			die("kit dat graphic-delete", fmt.Errorf("invalid graphic id %q: %w", args[3], err))
		}
		applyDatDeleteFile("kit dat graphic-delete", args[1], args[2], datcodec.DeletePlanRequest{Section: "graphic", ID: id})
	case "patch-unit":
		if len(args) < 5 {
			fmt.Fprintln(os.Stderr, "usage: kit dat patch-unit <in.dat> <out.dat> <civ_id> <unit_id> [scalar flags] [--type50-attack I,field=value] [--cost I,field=value] [--task I,field=value]")
			os.Exit(2)
		}
		civID, err := strconv.Atoi(args[3])
		if err != nil {
			die("kit dat", fmt.Errorf("invalid civ id %q: %w", args[3], err))
		}
		unitID, err := strconv.Atoi(args[4])
		if err != nil {
			die("kit dat", fmt.Errorf("invalid unit id %q: %w", args[4], err))
		}
		patch, err := parseUnitPatchOptions(args[5:])
		if err != nil {
			die("kit dat", err)
		}
		report, err := datfile.PatchUnitFile(args[1], args[2], civID, unitID, patch)
		if err != nil {
			die("kit dat", err)
		}
		printJSON(report)
	case "delete-plan":
		if len(args) < 4 {
			fmt.Fprintln(os.Stderr, "usage: kit dat delete-plan <in.dat> <section|effect-command|sound-item|graphic-row-kind|unit-header-task|unit-row-kind> <target> [--civ N|--all-civs] [--text]")
			os.Exit(2)
		}
		request, text, err := parseDatDeleteRequest(args[2], args[3], args[4:], true)
		if err != nil {
			die("kit dat delete-plan", err)
		}
		report, err := datcodec.DeletePlanFile(args[1], request)
		if err != nil {
			die("kit dat delete-plan", err)
		}
		if text {
			printDatDeletePlan(report)
		} else {
			printJSON(report)
		}
	case "effect-command-delete":
		if len(args) != 5 {
			fmt.Fprintln(os.Stderr, "usage: kit dat effect-command-delete <in.dat> <out.dat> <effect_id> <command_index>")
			os.Exit(2)
		}
		request, err := parseDatDeleteTarget("effect-command", args[3]+":"+args[4])
		if err != nil {
			die("kit dat effect-command-delete", err)
		}
		applyDatDeleteFile("kit dat effect-command-delete", args[1], args[2], request)
	case "sound-item-delete":
		if len(args) != 5 {
			fmt.Fprintln(os.Stderr, "usage: kit dat sound-item-delete <in.dat> <out.dat> <sound_id> <item_index>")
			os.Exit(2)
		}
		request, err := parseDatDeleteTarget("sound-item", args[3]+":"+args[4])
		if err != nil {
			die("kit dat sound-item-delete", err)
		}
		applyDatDeleteFile("kit dat sound-item-delete", args[1], args[2], request)
	case "graphic-delta-delete":
		if len(args) != 5 {
			fmt.Fprintln(os.Stderr, "usage: kit dat graphic-delta-delete <in.dat> <out.dat> <graphic_id> <row_index>")
			os.Exit(2)
		}
		request, err := parseDatDeleteTarget("graphic-delta", args[3]+":"+args[4])
		if err != nil {
			die("kit dat graphic-delta-delete", err)
		}
		applyDatDeleteFile("kit dat graphic-delta-delete", args[1], args[2], request)
	case "graphic-angle-sound-delete":
		if len(args) != 5 {
			fmt.Fprintln(os.Stderr, "usage: kit dat graphic-angle-sound-delete <in.dat> <out.dat> <graphic_id> <row_index>")
			os.Exit(2)
		}
		request, err := parseDatDeleteTarget("graphic-angle-sound", args[3]+":"+args[4])
		if err != nil {
			die("kit dat graphic-angle-sound-delete", err)
		}
		applyDatDeleteFile("kit dat graphic-angle-sound-delete", args[1], args[2], request)
	case "unit-header-task-delete":
		if len(args) != 5 {
			fmt.Fprintln(os.Stderr, "usage: kit dat unit-header-task-delete <in.dat> <out.dat> <unit_header_id> <task_index>")
			os.Exit(2)
		}
		request, err := parseDatDeleteTarget("unit-header-task", args[3]+":"+args[4])
		if err != nil {
			die("kit dat unit-header-task-delete", err)
		}
		applyDatDeleteFile("kit dat unit-header-task-delete", args[1], args[2], request)
	case "unit-child-delete":
		if len(args) != 7 {
			fmt.Fprintln(os.Stderr, "usage: kit dat unit-child-delete <in.dat> <out.dat> <damage-graphic|attack|armour|armor|train-location|drop-site|task> <civ_id> <unit_id> <row_index>")
			os.Exit(2)
		}
		section, err := datUnitChildDeleteSection(args[3])
		if err != nil {
			die("kit dat unit-child-delete", err)
		}
		request, err := parseDatDeleteTarget(section, args[4]+":"+args[5]+":"+args[6])
		if err != nil {
			die("kit dat unit-child-delete", err)
		}
		applyDatDeleteFile("kit dat unit-child-delete", args[1], args[2], request)
	case "delete":
		if len(args) < 5 {
			fmt.Fprintln(os.Stderr, "usage: kit dat delete <in.dat> <out.dat> <section|effect-command|sound-item|graphic-row-kind|unit-header-task|unit-row-kind> <target> [--civ N|--all-civs]")
			os.Exit(2)
		}
		request, _, err := parseDatDeleteRequest(args[3], args[4], args[5:], false)
		if err != nil {
			die("kit dat delete", err)
		}
		applyDatDeleteFile("kit dat delete", args[1], args[2], request)
	case "disconnect":
		if len(args) != 5 {
			fmt.Fprintln(os.Stderr, "usage: kit dat disconnect <in.dat> <out.dat> <tech|unit> <id>")
			os.Exit(2)
		}
		request, recipe, err := parseDatDisconnectRequest(args[3], args[4])
		if err != nil {
			die("kit dat disconnect", err)
		}
		refs, err := datcodec.ReferencesFile(args[1], request)
		if err != nil {
			die("kit dat disconnect", err)
		}
		if refs.Summary.RewriteSupported == 0 {
			die("kit dat disconnect", fmt.Errorf("refusing no-op disconnect: no verified rewrite-supported references found for %s id=%d", request.Section, request.ID))
		}
		patch, err := datcodec.PatchRecipeFile(args[1], args[2], recipe)
		if err != nil {
			die("kit dat disconnect", err)
		}
		printJSON(struct {
			Version          string                     `json:"version"`
			Request          datcodec.DeletePlanRequest `json:"request"`
			ReferenceSummary datcodec.ReferenceSummary  `json:"reference_summary"`
			PatchReport      datcodec.PatchReport       `json:"patch_report"`
			Verification     string                     `json:"verification"`
		}{
			Version:          datcodec.Version,
			Request:          request,
			ReferenceSummary: refs.Summary,
			PatchReport:      patch,
			Verification:     "structure_verified_not_engine_verified",
		})
	case "refs":
		if len(args) < 4 {
			fmt.Fprintln(os.Stderr, "usage: kit dat refs <in.dat> <section> <id> [--class CLASS] [--confidence NAME] [--source-section SECTION] [--limit N|--all] [--text|--json]")
			os.Exit(2)
		}
		id, err := strconv.Atoi(args[3])
		if err != nil {
			die("kit dat refs", fmt.Errorf("invalid id %q: %w", args[3], err))
		}
		opts, err := parseDatRefsOptions(args[4:])
		if err != nil {
			die("kit dat refs", err)
		}
		report, err := datcodec.ReferencesFile(args[1], datcodec.DeletePlanRequest{Section: args[2], ID: id})
		if err != nil {
			die("kit dat refs", err)
		}
		report = datcodec.FilterReferenceReport(report, opts.Filters)
		if opts.Text {
			printDatReferences(report)
		} else {
			printJSON(report)
		}
	case "codec-plan":
		if len(args) != 4 || args[2] != "--recipe" {
			fmt.Fprintln(os.Stderr, "usage: kit dat codec-plan <in.dat> --recipe recipe.json")
			os.Exit(2)
		}
		recipe, err := readCodecRecipe(args[3])
		if err != nil {
			die("kit dat codec-plan", err)
		}
		report, err := datcodec.PlanRecipeFile(args[1], recipe)
		if err != nil {
			die("kit dat codec-plan", err)
		}
		report.AuthoringNotes = datcodec.RecipeAuthoringWarnings(recipe)
		printJSON(report)
	case "codec-patch":
		if len(args) != 5 || args[3] != "--recipe" {
			fmt.Fprintln(os.Stderr, "usage: kit dat codec-patch <in.dat> <out.dat> --recipe recipe.json")
			os.Exit(2)
		}
		recipe, err := readCodecRecipe(args[4])
		if err != nil {
			die("kit dat codec-patch", err)
		}
		report, err := datcodec.PatchRecipeFile(args[1], args[2], recipe)
		if err != nil {
			die("kit dat codec-patch", err)
		}
		report.AuthoringNotes = datcodec.RecipeAuthoringWarnings(recipe)
		printJSON(report)
	case "semantics-pack":
		if len(args) != 3 && len(args) != 5 {
			fmt.Fprintln(os.Stderr, "usage: kit dat semantics-pack <in.dat> <out-dir> [--feature full|local-building-effects|class-scope-effects]")
			os.Exit(2)
		}
		opts := diagnostics.SemanticsPackOptions{}
		if len(args) == 5 {
			if args[3] != "--feature" {
				die("kit dat semantics-pack", fmt.Errorf("unknown option %q", args[3]))
			}
			opts.Feature = args[4]
		}
		pack, files, err := diagnostics.WriteDATCommandSemanticsPackWithOptions(args[1], args[2], opts)
		if err != nil {
			die("kit dat semantics-pack", err)
		}
		printJSON(struct {
			Version       string                    `json:"version"`
			Status        string                    `json:"status"`
			GeneratedFrom diagnostics.GeneratedFrom `json:"generated_from"`
			Files         []string                  `json:"files"`
			Lanes         int                       `json:"lanes"`
			PendingLanes  int                       `json:"pending_lanes"`
		}{
			Version:       pack.Version,
			Status:        pack.Status,
			GeneratedFrom: pack.GeneratedFrom,
			Files:         files,
			Lanes:         len(pack.Expected.Lanes),
			PendingLanes:  len(pack.Expected.PendingLanes),
		})
	case "plan":
		if len(args) != 4 || args[2] != "--recipe" {
			fmt.Fprintln(os.Stderr, "usage: kit dat plan <in.dat> --recipe recipe.json")
			os.Exit(2)
		}
		recipe, err := readCombinedDatRecipe(args[3])
		if err != nil {
			die("kit dat", err)
		}
		_, report, err := applyCombinedDatRecipeFile(args[1], "", recipe, false)
		if err != nil {
			die("kit dat", err)
		}
		printJSON(report)
	case "patch":
		if len(args) != 5 || args[3] != "--recipe" {
			fmt.Fprintln(os.Stderr, "usage: kit dat patch <in.dat> <out.dat> --recipe recipe.json")
			os.Exit(2)
		}
		recipe, err := readCombinedDatRecipe(args[4])
		if err != nil {
			die("kit dat", err)
		}
		_, report, err := applyCombinedDatRecipeFile(args[1], args[2], recipe, true)
		if err != nil {
			die("kit dat", err)
		}
		printJSON(report)
	default:
		fmt.Fprintln(os.Stderr, "usage: kit dat <facts|info|check|roundtrip|sprite|unit-headers|unit-header|civs|civ-patch|spans|graphics|graphic|graphic-create|graphic-patch|graphic-delete|palette|effects|effect|effect-explain|effect-create|effect-patch|effect-disable|effect-delete|effect-command-delete|command-matrix|techs|tech|tech-explain|tech-create|tech-patch|tech-delete|abilities|ability|ability-create|ability-patch|ability-disable|ability-delete|tech-tree|tech-tree-connection-create|tech-tree-connection-patch|tech-tree-connection-delete|units|unit|unit-create|unit-delete|unit-child-delete|availability|availability-set|terrain-restrictions|terrain-restriction-patch|terrains|terrain|terrain-patch|sounds|sound|sound-create|sound-patch|sound-delete|sound-item-delete|player-colours|player-colour-create|player-colour-patch|player-colour-delete|random-maps|patch-graphic|patch-unit|graphic-delta-delete|graphic-angle-sound-delete|unit-header-task-delete|refs|delete-plan|delete|codec-plan|codec-patch|semantics-pack|semantics-readback|plan|patch> <empires*.dat> [args...]")
		os.Exit(2)
	}
}

func runDatFacts(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: kit dat facts <effect-types|attributes|resources|task-types|tech-modifiers|effect-command> [id] [--text|--json]")
		os.Exit(2)
	}
	text := false
	filtered := make([]string, 0, len(args))
	for _, arg := range args {
		switch arg {
		case "--text":
			text = true
		case "--json":
			text = false
		default:
			filtered = append(filtered, arg)
		}
	}
	if len(filtered) == 0 {
		die("kit dat facts", fmt.Errorf("missing facts domain"))
	}
	if filtered[0] == "effect-command" || filtered[0] == "effect-type" {
		if len(filtered) != 2 {
			die("kit dat facts", fmt.Errorf("usage: kit dat facts effect-command <id> [--text|--json]"))
		}
		id, err := strconv.Atoi(filtered[1])
		if err != nil {
			die("kit dat facts", fmt.Errorf("invalid effect-command id %q: %w", filtered[1], err))
		}
		row, err := datfacts.ExplainEffectCommandType(id)
		if err != nil {
			die("kit dat facts", err)
		}
		if text {
			printDatFactRowText(row)
			return
		}
		printJSON(row)
		return
	}
	if len(filtered) != 1 {
		die("kit dat facts", fmt.Errorf("domain %q does not take an id; use effect-command <id> for a single command", filtered[0]))
	}
	report, err := datfacts.Catalog(filtered[0])
	if err != nil {
		die("kit dat facts", err)
	}
	if text {
		printDatFactsText(report)
		return
	}
	printJSON(report)
}

func printDatFactsText(report datfacts.CatalogReport) {
	fmt.Printf("DAT facts %s domain=%s rows=%d verification=%s\n",
		report.Version, report.Domain, report.Summary.Rows, report.Verification.Label)
	if report.Summary.KnownDisagreements > 0 {
		fmt.Printf("known_disagreements=%d\n", report.Summary.KnownDisagreements)
	}
	for _, row := range report.Rows {
		printDatFactRowText(row)
	}
}

func printDatFactRowText(row datfacts.FactRow) {
	fmt.Printf("\n[%d] %s\n", row.ID, row.Name)
	if row.Family != "" || row.Scope != "" {
		fmt.Printf("  family=%s scope=%s\n", row.Family, row.Scope)
	}
	fmt.Printf("  confidence=%s\n", row.Confidence)
	fmt.Printf("  meaning=%s\n", row.Meaning)
	if row.SafeUse != "" {
		fmt.Printf("  safe_use=%s\n", row.SafeUse)
	}
	for _, note := range row.Notes {
		fmt.Printf("  note=%s\n", note)
	}
	for _, disagreement := range row.Disagreements {
		fmt.Printf("  disagreement=%s\n", disagreement)
	}
	for _, source := range row.Sources {
		fmt.Printf("  source=%s\n", source)
	}
}

func runDatSemanticsReadback(args []string) {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: kit dat semantics-readback <expected.json> --dat empires*.dat --scenario file.aoe2scenario --replay file.aoe2record [--xsdat file.xsdat] [--out report.md] [--text]")
		os.Exit(2)
	}
	opts := diagnostics.SemanticsReadbackOptions{ExpectedPath: args[1]}
	text := false
	outPath := ""
	for i := 2; i < len(args); i++ {
		switch args[i] {
		case "--dat":
			i++
			if i >= len(args) {
				die("kit dat semantics-readback", fmt.Errorf("--dat needs a path"))
			}
			opts.DatPath = args[i]
		case "--scenario":
			i++
			if i >= len(args) {
				die("kit dat semantics-readback", fmt.Errorf("--scenario needs a path"))
			}
			opts.ScenarioPath = args[i]
		case "--replay":
			i++
			if i >= len(args) {
				die("kit dat semantics-readback", fmt.Errorf("--replay needs a path"))
			}
			opts.ReplayPath = args[i]
		case "--xsdat":
			i++
			if i >= len(args) {
				die("kit dat semantics-readback", fmt.Errorf("--xsdat needs a path"))
			}
			opts.XSDataPath = args[i]
		case "--out":
			i++
			if i >= len(args) {
				die("kit dat semantics-readback", fmt.Errorf("--out needs a path"))
			}
			outPath = args[i]
		case "--text":
			text = true
		case "--json":
			text = false
		default:
			die("kit dat semantics-readback", fmt.Errorf("unknown option %q", args[i]))
		}
	}
	if opts.DatPath == "" || opts.ScenarioPath == "" || opts.ReplayPath == "" {
		fmt.Fprintln(os.Stderr, "usage: kit dat semantics-readback <expected.json> --dat empires*.dat --scenario file.aoe2scenario --replay file.aoe2record [--xsdat file.xsdat] [--out report.md] [--text]")
		os.Exit(2)
	}
	report, err := diagnostics.ReadDATCommandSemantics(opts)
	if err != nil {
		die("kit dat semantics-readback", err)
	}
	if outPath != "" {
		if err := diagnostics.WriteDATCommandSemanticsReadbackMarkdown(report, outPath); err != nil {
			die("kit dat semantics-readback", err)
		}
	}
	if text {
		printDATSemanticsReadbackText(report, outPath)
	} else {
		printJSON(report)
	}
}

func printDATSemanticsReadbackText(report *diagnostics.SemanticsReadbackReport, outPath string) {
	fmt.Printf("status=%s promoted=%d failed=%d inconclusive=%d not_implemented=%d\n",
		report.Status,
		report.Summary.Promoted,
		report.Summary.Failed,
		report.Summary.Inconclusive,
		report.Summary.NotImplemented,
	)
	fmt.Printf("replay duration=%s trigger_graph_ok=%t data_set_status=%s\n",
		report.Replay.Duration,
		report.Replay.TriggerGraphOK,
		report.Replay.DataSetStatus,
	)
	if report.Inputs.XSDataPath != "" {
		fmt.Printf("sidecar ok=%t schema=%s rows=%d fixture=%s\n",
			report.Sidecar.OK,
			report.Sidecar.Schema,
			report.Sidecar.Rows,
			report.Sidecar.Fixture,
		)
	}
	if outPath != "" {
		fmt.Printf("wrote=%s\n", outPath)
	}
	for _, gap := range report.ParserGaps {
		fmt.Printf("parser_gap: %s\n", gap)
	}
	for _, lane := range report.Lanes {
		fmt.Printf("%s: %s", lane.ID, lane.Status)
		if lane.PromotedTier != "" {
			fmt.Printf(" (%s)", lane.PromotedTier)
		}
		fmt.Printf(" - %s\n", lane.Conclusion)
	}
}

func readRecipe(path string) (datfile.Recipe, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return datfile.Recipe{}, err
	}
	var recipe datfile.Recipe
	if err := json.Unmarshal(data, &recipe); err != nil {
		return datfile.Recipe{}, err
	}
	return recipe, nil
}

type combinedDatRecipe struct {
	Span  datfile.Recipe  `json:"-"`
	Codec datcodec.Recipe `json:"-"`
}

func (recipe *combinedDatRecipe) UnmarshalJSON(data []byte) error {
	if err := json.Unmarshal(data, &recipe.Span); err != nil {
		return err
	}
	if err := json.Unmarshal(data, &recipe.Codec); err != nil {
		return err
	}
	recipe.routeOverlappingCreatesToCodec()
	return nil
}

func (recipe *combinedDatRecipe) routeOverlappingCreatesToCodec() {
	if recipe.Codec.CreateGraphic != nil || len(recipe.Codec.CreateGraphics) > 0 {
		recipe.Span.CreateGraphic = nil
		recipe.Span.CreateGraphics = nil
	}
	if recipe.Codec.CreateUnit != nil || len(recipe.Codec.CreateUnits) > 0 {
		recipe.Span.CreateUnit = nil
		recipe.Span.CreateUnits = nil
	}
}

type combinedDatRecipeReport struct {
	InputCompressedBytes  int                    `json:"input_compressed_bytes"`
	OutputCompressedBytes int                    `json:"output_compressed_bytes"`
	SpanPatch             *datfile.RecipeReport  `json:"span_patch,omitempty"`
	CodecPatch            *datcodec.PatchReport  `json:"codec_patch,omitempty"`
	Verified              bool                   `json:"verified"`
	Verification          aoe2.VerificationClaim `json:"verification"`
}

func readCombinedDatRecipe(path string) (combinedDatRecipe, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return combinedDatRecipe{}, err
	}
	var recipe combinedDatRecipe
	if err := json.Unmarshal(data, &recipe); err != nil {
		return combinedDatRecipe{}, fmt.Errorf("parse recipe %s: %w", path, err)
	}
	return recipe, nil
}

func applyCombinedDatRecipeFile(inputPath, outputPath string, recipe combinedDatRecipe, writeOutput bool) ([]byte, combinedDatRecipeReport, error) {
	if recipe.datfileEmpty() && recipe.datcodecEmpty() {
		return nil, combinedDatRecipeReport{}, errors.New("recipe has no dat operations")
	}
	if writeOutput {
		inputAbs, err := filepath.Abs(inputPath)
		if err != nil {
			return nil, combinedDatRecipeReport{}, err
		}
		outputAbs, err := filepath.Abs(outputPath)
		if err != nil {
			return nil, combinedDatRecipeReport{}, err
		}
		if inputAbs == outputAbs {
			return nil, combinedDatRecipeReport{}, errors.New("refusing in-place dat patch; choose a separate output path")
		}
	}
	compressed, err := os.ReadFile(inputPath)
	if err != nil {
		return nil, combinedDatRecipeReport{}, err
	}
	output, report, err := applyCombinedDatRecipe(compressed, recipe)
	if err != nil {
		return nil, combinedDatRecipeReport{}, err
	}
	if writeOutput {
		if err := os.WriteFile(outputPath, output, 0644); err != nil {
			return nil, combinedDatRecipeReport{}, err
		}
	}
	return output, report, nil
}

func applyCombinedDatRecipe(compressed []byte, recipe combinedDatRecipe) ([]byte, combinedDatRecipeReport, error) {
	if recipe.datfileEmpty() && recipe.datcodecEmpty() {
		return nil, combinedDatRecipeReport{}, errors.New("recipe has no dat operations")
	}
	current := append([]byte(nil), compressed...)
	report := combinedDatRecipeReport{InputCompressedBytes: len(compressed)}
	if !recipe.datfileEmpty() {
		output, spanReport, err := datfile.PatchRecipe(current, recipe.Span)
		if err != nil {
			return nil, combinedDatRecipeReport{}, fmt.Errorf("span recipe: %w", err)
		}
		current = output
		report.SpanPatch = &spanReport
	}
	if !recipe.datcodecEmpty() {
		output, codecReport, err := datcodec.PatchRecipe(current, recipe.Codec)
		if err != nil {
			return nil, combinedDatRecipeReport{}, fmt.Errorf("codec recipe: %w", err)
		}
		current = output
		report.CodecPatch = &codecReport
	}
	report.OutputCompressedBytes = len(current)
	report.Verified = (report.SpanPatch == nil || report.SpanPatch.Verified) && (report.CodecPatch == nil || report.CodecPatch.Verified)
	report.Verification = aoe2.StructureVerification(report.Verified)
	report.Verification.Note = "combined dat recipe applied the span-backed graphic/unit engine first and the codec-backed Effects/Techs/Civ/Terrain/Sound/PlayerColour engine second; each sub-engine performed its own reparse/readback checks, while in-engine behavior remains a separate oracle."
	return current, report, nil
}

func (recipe combinedDatRecipe) datfileEmpty() bool {
	return recipe.Span.Empty()
}

func (recipe combinedDatRecipe) datcodecEmpty() bool {
	return recipe.Codec.Empty()
}

func readCodecRecipe(path string) (datcodec.Recipe, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return datcodec.Recipe{}, err
	}
	var recipe datcodec.Recipe
	if err := json.Unmarshal(data, &recipe); err != nil {
		return datcodec.Recipe{}, fmt.Errorf("parse recipe %s: %w", path, err)
	}
	return recipe, nil
}

func runFX(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: kit fx <new|bind|lint> [args]")
		os.Exit(2)
	}
	switch args[0] {
	case "new":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: kit fx new <name> --preset trail|explosion|aura|projectile-fire [--grid RxC] [descriptor overrides]")
			os.Exit(2)
		}
		name := args[1]
		preset := "trail"
		grid := fx.Grid{Rows: 1, Cols: 1, Frames: 1}
		overrides := fx.DescriptorOverrides{}
		out := ""
		for i := 2; i < len(args); i++ {
			switch args[i] {
			case "--preset":
				i++
				if i >= len(args) {
					die("kit fx new", fmt.Errorf("--preset needs a value"))
				}
				preset = args[i]
			case "--grid":
				i++
				if i >= len(args) {
					die("kit fx new", fmt.Errorf("--grid needs a value"))
				}
				parsed, err := fx.ParseGrid(args[i])
				if err != nil {
					die("kit fx new", err)
				}
				grid = parsed
			case "--out":
				i++
				if i >= len(args) {
					die("kit fx new", fmt.Errorf("--out needs a value"))
				}
				out = args[i]
			default:
				if err := parseFXDescriptorOverride(args, &i, &overrides); err != nil {
					die("kit fx new", err)
				}
			}
		}
		data, err := fx.DescriptorJSON(name, preset, grid, overrides)
		if err != nil {
			die("kit fx new", err)
		}
		data = append(data, '\n')
		if out != "" {
			if err := os.WriteFile(out, data, 0644); err != nil {
				die("kit fx new", err)
			}
			printJSON(map[string]any{"version": fx.Version, "ok": true, "descriptor": out})
			return
		}
		fmt.Print(string(data))
	case "bind":
		options, err := parseFXBindOptions(args[1:])
		if err != nil {
			die("kit fx bind", err)
		}
		report, err := fx.Bind(options)
		if err != nil {
			die("kit fx bind", err)
		}
		printJSON(report)
	case "lint":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: kit fx lint <mod-or-resources/_common> [--dat empires*.dat] [--text]")
			os.Exit(2)
		}
		options := fx.LintOptions{ModPath: args[1]}
		text := false
		for i := 2; i < len(args); i++ {
			switch args[i] {
			case "--dat":
				i++
				if i >= len(args) {
					die("kit fx lint", fmt.Errorf("--dat needs a value"))
				}
				options.DatPath = args[i]
			case "--text":
				text = true
			default:
				die("kit fx lint", fmt.Errorf("unknown option %q", args[i]))
			}
		}
		report, err := fx.Lint(options)
		if err != nil {
			die("kit fx lint", err)
		}
		if text {
			printFXLintText(report)
		} else {
			printJSON(report)
		}
		if !report.OK {
			os.Exit(1)
		}
	default:
		fmt.Fprintln(os.Stderr, "usage: kit fx <new|bind|lint> [args]")
		os.Exit(2)
	}
}

func runGFX(args []string) {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: kit gfx <info|export> <file.sld> [--out dir] [--limit N] [--text]")
		os.Exit(2)
	}
	switch args[0] {
	case "info":
		text := false
		for _, arg := range args[2:] {
			switch arg {
			case "--text":
				text = true
			default:
				die("kit gfx info", fmt.Errorf("unknown option %q", arg))
			}
		}
		file, err := gfx.OpenSLD(args[1])
		if err != nil {
			die("kit gfx info", err)
		}
		if text {
			printSLDInfoText(file)
		} else {
			printJSON(file)
		}
	case "export":
		options, text := parseSLDExportOptions("kit gfx export", args[2:])
		report, err := gfx.ExportSLD(args[1], options)
		if err != nil {
			die("kit gfx export", err)
		}
		if text {
			printSLDExportText(report)
		} else {
			printJSON(report)
		}
	default:
		fmt.Fprintln(os.Stderr, "usage: kit gfx <info|export> <file.sld> [--out dir] [--limit N] [--text]")
		os.Exit(2)
	}
}

func parseSLDExportOptions(prefix string, args []string) (gfx.SLDExportOptions, bool) {
	options := gfx.SLDExportOptions{}
	text := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--out":
			i++
			if i >= len(args) {
				die(prefix, fmt.Errorf("--out needs a directory"))
			}
			options.OutDir = args[i]
		case "--limit":
			value, err := parseNextInt32(args, &i, "--limit")
			if err != nil {
				die(prefix, err)
			}
			if value < 0 {
				die(prefix, fmt.Errorf("--limit must be non-negative"))
			}
			options.Limit = int(value)
		case "--text":
			text = true
		case "--stats":
			options.Stats = true
		default:
			die(prefix, fmt.Errorf("unknown option %q", args[i]))
		}
	}
	return options, text
}

func printSLDInfoText(file *gfx.SLD) {
	fmt.Printf("sld: %s\n", file.Path)
	fmt.Printf("version: %d\n", file.Version)
	fmt.Printf("frames: %d\n", len(file.Frames))
	for _, frame := range file.Frames {
		fmt.Printf("frame %d index=%d canvas=%dx%d hotspot=%d,%d type=0x%02x layers=%d\n",
			frame.Ordinal, frame.Index, frame.Width, frame.Height, frame.HotspotX, frame.HotspotY, frame.Type, len(frame.Layers))
		for _, layer := range frame.Layers {
			fmt.Printf("  %s compression=%s rect=%d,%d..%d,%d commands=%d draw_blocks=%d bytes=%d\n",
				layer.Name, layer.Compression, layer.OffsetX1, layer.OffsetY1, layer.OffsetX2, layer.OffsetY2,
				layer.CommandCount, layer.DrawBlocks, layer.ContentLength)
		}
	}
}

func printSLDExportText(report gfx.SLDExportReport) {
	fmt.Printf("ok: %t\n", report.OK)
	fmt.Printf("source: %s\n", report.Source)
	fmt.Printf("output_dir: %s\n", report.OutputDir)
	fmt.Printf("frames: %d exported=%d\n", report.Frames, len(report.Exported))
	fmt.Printf("verification: %s\n", report.Verification.Label)
	if report.ManifestPath != "" {
		fmt.Printf("manifest: %s\n", report.ManifestPath)
	}
	if report.ContactSheet != "" {
		fmt.Printf("contact_sheet: %s\n", report.ContactSheet)
	}
	for _, exported := range report.Exported {
		fmt.Printf("frame %d index=%d %s %dx%d anchor=%d,%d opaque=%d -> %s\n",
			exported.FrameOrdinal, exported.FrameIndex, exported.Layer, exported.Width, exported.Height,
			exported.HotspotX, exported.HotspotY, exported.OpaquePixels, exported.Path)
	}
	for _, stats := range report.Stats {
		fmt.Printf("stats frame=%d index=%d state=%s bbox=%d,%d,%dx%d coverage=%.4f mean_rgb=%.1f,%.1f,%.1f sat=%.3f buckets=green:%.3f yellow:%.3f red:%.3f brown:%.3f grey:%.3f white:%.3f other:%.3f\n",
			stats.FrameOrdinal, stats.FrameIndex, stats.State,
			stats.Bounds.X, stats.Bounds.Y, stats.Bounds.Width, stats.Bounds.Height,
			stats.CanvasCoverage, stats.MeanR, stats.MeanG, stats.MeanB, stats.MeanSaturation,
			stats.HueBuckets.Green, stats.HueBuckets.Yellow, stats.HueBuckets.Red, stats.HueBuckets.Brown,
			stats.HueBuckets.Grey, stats.HueBuckets.White, stats.HueBuckets.Other)
	}
	for _, warning := range report.Warnings {
		fmt.Printf("warning: %s\n", warning)
	}
}

func parseFXBindOptions(args []string) (fx.BindOptions, error) {
	options := fx.BindOptions{Preset: "trail", Slot: "standing", FromGraphic: -1}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--atlas":
			i++
			if i >= len(args) {
				return options, fmt.Errorf("--atlas needs a PNG path")
			}
			options.AtlasPath = args[i]
		case "--dds":
			i++
			if i >= len(args) {
				return options, fmt.Errorf("--dds needs a DDS path")
			}
			options.DDSPath = args[i]
		case "--grid":
			i++
			if i >= len(args) {
				return options, fmt.Errorf("--grid needs a value")
			}
			grid, err := fx.ParseGrid(args[i])
			if err != nil {
				return options, err
			}
			options.Grid = grid
		case "--frames":
			value, err := parseNextInt32(args, &i, "--frames")
			if err != nil {
				return options, err
			}
			if value <= 0 {
				return options, fmt.Errorf("--frames must be positive")
			}
			options.Grid.Frames = int(value)
		case "--frame-size":
			i++
			if i >= len(args) {
				return options, fmt.Errorf("--frame-size needs a value")
			}
			size, err := fx.ParseFrameSize(args[i])
			if err != nil {
				return options, err
			}
			options.Grid.Width = size.Width
			options.Grid.Height = size.Height
		case "--into":
			i++
			if i >= len(args) {
				return options, fmt.Errorf("--into needs a resources/_common path")
			}
			options.IntoCommon = args[i]
		case "--dat":
			i++
			if i >= len(args) {
				return options, fmt.Errorf("--dat needs a path")
			}
			options.DatPath = args[i]
		case "--unit":
			value, err := parseNextInt32(args, &i, "--unit")
			if err != nil {
				return options, err
			}
			options.UnitID = int(value)
		case "--slot":
			i++
			if i >= len(args) {
				return options, fmt.Errorf("--slot needs a value")
			}
			options.Slot = args[i]
		case "--from":
			value, err := parseNextInt32(args, &i, "--from")
			if err != nil {
				return options, err
			}
			options.FromGraphic = int(value)
		case "--name":
			i++
			if i >= len(args) {
				return options, fmt.Errorf("--name needs a value")
			}
			options.Name = args[i]
		case "--preset":
			i++
			if i >= len(args) {
				return options, fmt.Errorf("--preset needs a value")
			}
			options.Preset = args[i]
		case "--clear-slp":
			options.ClearSLP = true
		case "--keep-slp":
			options.ClearSLP = false
		case "--dry-run":
			options.DryRun = true
		default:
			if err := parseFXDescriptorOverride(args, &i, &options.Overrides); err != nil {
				return options, err
			}
		}
	}
	if (options.Grid.Rows == 0 || options.Grid.Cols == 0) && (options.Grid.Width == 0 || options.Grid.Height == 0) {
		return options, fmt.Errorf("--grid RxC or --frames N --frame-size WxH is required")
	}
	if options.FromGraphic < 0 {
		return options, fmt.Errorf("--from graphic id is required")
	}
	return options, nil
}

func parseFXDescriptorOverride(args []string, i *int, overrides *fx.DescriptorOverrides) error {
	arg := args[*i]
	switch arg {
	case "--type":
		value, err := parseNextString(args, i, arg)
		if err != nil {
			return err
		}
		overrides.Type = &value
	case "--duration":
		value, err := parseNextFloat64(args, i, arg)
		if err != nil {
			return err
		}
		overrides.Duration = &value
	case "--alpha-start":
		value, err := parseNextFloat64(args, i, arg)
		if err != nil {
			return err
		}
		overrides.AlphaStart = &value
	case "--alpha-end":
		value, err := parseNextFloat64(args, i, arg)
		if err != nil {
			return err
		}
		overrides.AlphaEnd = &value
	case "--scale":
		value, err := parseNextFloat64(args, i, arg)
		if err != nil {
			return err
		}
		overrides.Scale = &value
	case "--scale-start":
		value, err := parseNextFloat64(args, i, arg)
		if err != nil {
			return err
		}
		overrides.ScaleStart = &value
	case "--scale-end":
		value, err := parseNextFloat64(args, i, arg)
		if err != nil {
			return err
		}
		overrides.ScaleEnd = &value
	case "--rotation":
		value, err := parseNextFloat64(args, i, arg)
		if err != nil {
			return err
		}
		overrides.Rotation = &value
	case "--stop-mode":
		value, err := parseNextString(args, i, arg)
		if err != nil {
			return err
		}
		overrides.StopMode = &value
	case "--is-fire":
		value, err := parseNextBool(args, i, arg)
		if err != nil {
			return err
		}
		overrides.IsFire = &value
	case "--display-in-fog":
		value, err := parseNextBool(args, i, arg)
		if err != nil {
			return err
		}
		overrides.DisplayInFog = &value
	default:
		return fmt.Errorf("unknown option %q", arg)
	}
	return nil
}

func parseNextString(args []string, i *int, name string) (string, error) {
	if *i+1 >= len(args) {
		return "", fmt.Errorf("%s needs a value", name)
	}
	*i = *i + 1
	return args[*i], nil
}

func parseNextFloat64(args []string, i *int, name string) (float64, error) {
	if *i+1 >= len(args) {
		return 0, fmt.Errorf("%s needs a value", name)
	}
	*i = *i + 1
	value, err := strconv.ParseFloat(args[*i], 64)
	if err != nil {
		return 0, fmt.Errorf("invalid %s %q: %w", name, args[*i], err)
	}
	return value, nil
}

func parseNextBool(args []string, i *int, name string) (bool, error) {
	if *i+1 >= len(args) {
		return false, fmt.Errorf("%s needs true or false", name)
	}
	*i = *i + 1
	value, err := strconv.ParseBool(args[*i])
	if err != nil {
		return false, fmt.Errorf("invalid %s %q: %w", name, args[*i], err)
	}
	return value, nil
}

func parseNextBoolish(args []string, i *int, name string) (bool, error) {
	if *i+1 >= len(args) {
		return false, fmt.Errorf("%s needs true|false|1|0", name)
	}
	*i = *i + 1
	switch strings.ToLower(strings.TrimSpace(args[*i])) {
	case "1", "true", "t", "yes", "y", "on":
		return true, nil
	case "0", "false", "f", "no", "n", "off":
		return false, nil
	default:
		return false, fmt.Errorf("invalid %s %q: use true|false|1|0", name, args[*i])
	}
}

func printFXLintText(report fx.LintReport) {
	fmt.Printf("ok=%t descriptors=%d errors=%d common=%s\n", report.OK, len(report.Descriptors), len(report.Errors), report.CommonRoot)
	if report.DatPath != "" {
		fmt.Printf("dat=%s\n", report.DatPath)
	}
	for _, descriptor := range report.Descriptors {
		status := "ok"
		if len(descriptor.Errors) > 0 {
			status = "error"
		}
		fmt.Printf("- %s %s images=%d frames=%d referenced=%t atlas=%s\n", descriptor.Name, status, descriptor.ImageCount, descriptor.AtlasFrames, descriptor.ReferencedByDAT, descriptor.AtlasFile)
		for _, err := range descriptor.Errors {
			fmt.Printf("  error: %s\n", err)
		}
		for _, warning := range descriptor.Warnings {
			fmt.Printf("  warning: %s\n", warning)
		}
	}
}

func parseGraphicPatchOptions(args []string) (datfile.GraphicPatch, error) {
	var patch datfile.GraphicPatch
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--name":
			value, err := parseNextString(args, &i, "--name")
			if err != nil {
				return patch, err
			}
			patch.Name = &value
		case "--slp":
			value, err := parseNextInt32(args, &i, "--slp")
			if err != nil {
				return patch, err
			}
			patch.SLP = &value
		case "--file-name":
			value, err := parseNextString(args, &i, "--file-name")
			if err != nil {
				return patch, err
			}
			patch.FileName = &value
		case "--particle-effect-name":
			value, err := parseNextString(args, &i, "--particle-effect-name")
			if err != nil {
				return patch, err
			}
			patch.ParticleEffectName = &value
		case "--is-loaded":
			value, err := parseNextInt8(args, &i, "--is-loaded")
			if err != nil {
				return patch, err
			}
			patch.IsLoaded = &value
		case "--old-color-flag":
			value, err := parseNextInt8(args, &i, "--old-color-flag")
			if err != nil {
				return patch, err
			}
			patch.OldColorFlag = &value
		case "--layer":
			value, err := parseNextInt8(args, &i, "--layer")
			if err != nil {
				return patch, err
			}
			patch.Layer = &value
		case "--player-color":
			value, err := parseNextInt8(args, &i, "--player-color")
			if err != nil {
				return patch, err
			}
			patch.PlayerColor = &value
		case "--rainbow":
			value, err := parseNextInt8(args, &i, "--rainbow")
			if err != nil {
				return patch, err
			}
			patch.Rainbow = &value
		case "--transparent-selection":
			value, err := parseNextInt8(args, &i, "--transparent-selection")
			if err != nil {
				return patch, err
			}
			patch.TransparentSelect = &value
		case "--coordinates":
			value, err := parseNextInt16List(args, &i, "--coordinates", 4)
			if err != nil {
				return patch, err
			}
			patch.Coordinates = value
		case "--sound-id":
			value, err := parseNextInt16(args, &i, "--sound-id")
			if err != nil {
				return patch, err
			}
			patch.SoundID = &value
		case "--wwise-sound-id":
			value, err := parseNextUint32(args, &i, "--wwise-sound-id")
			if err != nil {
				return patch, err
			}
			patch.WwiseSoundID = &value
		case "--frame-count":
			value, err := parseNextInt16(args, &i, "--frame-count")
			if err != nil {
				return patch, err
			}
			if value < 0 {
				return patch, fmt.Errorf("--frame-count must be non-negative")
			}
			patch.FrameCount = &value
		case "--speed-multiplier":
			value, err := parseNextFloat32(args, &i, "--speed-multiplier")
			if err != nil {
				return patch, err
			}
			patch.SpeedMultiplier = &value
		case "--frame-duration":
			value, err := parseNextFloat32(args, &i, "--frame-duration")
			if err != nil {
				return patch, err
			}
			patch.FrameDuration = &value
		case "--replay-delay":
			value, err := parseNextFloat32(args, &i, "--replay-delay")
			if err != nil {
				return patch, err
			}
			patch.ReplayDelay = &value
		case "--sequence-type":
			value, err := parseNextUint8(args, &i, "--sequence-type")
			if err != nil {
				return patch, err
			}
			patch.SequenceType = &value
		case "--mirroring-mode":
			value, err := parseNextInt8(args, &i, "--mirroring-mode")
			if err != nil {
				return patch, err
			}
			patch.MirroringMode = &value
		case "--editor-flag":
			value, err := parseNextInt8(args, &i, "--editor-flag")
			if err != nil {
				return patch, err
			}
			patch.EditorFlag = &value
		default:
			return patch, fmt.Errorf("unknown patch option %q", args[i])
		}
	}
	if patch.Empty() {
		return patch, fmt.Errorf("provide at least one patch option")
	}
	return patch, nil
}

func parseDatGraphicCreateOptions(args []string) (datfile.GraphicCreateRecipe, error) {
	var recipe datfile.GraphicCreateRecipe
	fromSet := false
	var patchArgs []string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--from":
			value, err := parseNextInt(args, &i, "--from")
			if err != nil {
				return recipe, err
			}
			recipe.From = value
			fromSet = true
		default:
			patchArgs = append(patchArgs, args[i])
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "--") {
				i++
				patchArgs = append(patchArgs, args[i])
			}
		}
	}
	if !fromSet {
		return recipe, errors.New("--from is required")
	}
	if len(patchArgs) > 0 {
		patch, err := parseGraphicPatchOptions(patchArgs)
		if err != nil {
			return recipe, err
		}
		applyGraphicPatchToCreateRecipe(&recipe, patch)
	}
	return recipe, nil
}

func applyGraphicPatchToCreateRecipe(recipe *datfile.GraphicCreateRecipe, patch datfile.GraphicPatch) {
	recipe.Name = patch.Name
	recipe.FileName = patch.FileName
	recipe.ParticleEffectName = patch.ParticleEffectName
	recipe.SLP = patch.SLP
	recipe.IsLoaded = patch.IsLoaded
	recipe.OldColorFlag = patch.OldColorFlag
	recipe.Layer = patch.Layer
	recipe.PlayerColor = patch.PlayerColor
	recipe.Rainbow = patch.Rainbow
	recipe.TransparentSelect = patch.TransparentSelect
	recipe.Coordinates = patch.Coordinates
	recipe.SoundID = patch.SoundID
	recipe.WwiseSoundID = patch.WwiseSoundID
	recipe.FrameCount = patch.FrameCount
	recipe.SpeedMultiplier = patch.SpeedMultiplier
	recipe.FrameDuration = patch.FrameDuration
	recipe.ReplayDelay = patch.ReplayDelay
	recipe.SequenceType = patch.SequenceType
	recipe.MirroringMode = patch.MirroringMode
	recipe.EditorFlag = patch.EditorFlag
	recipe.SetDeltas = patch.SetDeltas
	recipe.SetAngleSounds = patch.SetAngleSounds
}

func parseUnitPatchOptions(args []string) (datfile.UnitPatch, error) {
	var patch datfile.UnitPatch
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--class":
			value, err := parseNextInt16(args, &i, "--class")
			if err != nil {
				return patch, err
			}
			patch.Class = &value
		case "--hit-points":
			value, err := parseNextInt16(args, &i, "--hit-points")
			if err != nil {
				return patch, err
			}
			patch.HitPoints = &value
		case "--line-of-sight":
			value, err := parseNextFloat32(args, &i, "--line-of-sight")
			if err != nil {
				return patch, err
			}
			patch.LineOfSight = &value
		case "--movement-type":
			if i+1 >= len(args) {
				return patch, fmt.Errorf("--movement-type needs a value")
			}
			i++
			value, err := strconv.ParseUint(args[i], 10, 8)
			if err != nil {
				return patch, fmt.Errorf("invalid --movement-type %q: %w", args[i], err)
			}
			movement := uint8(value)
			patch.MovementType = &movement
		case "--standing-graphic":
			value, err := parseNextInt16(args, &i, "--standing-graphic")
			if err != nil {
				return patch, err
			}
			patch.StandingGraphic1 = &value
		case "--standing-graphic-2":
			value, err := parseNextInt16(args, &i, "--standing-graphic-2")
			if err != nil {
				return patch, err
			}
			patch.StandingGraphic2 = &value
		case "--dying-graphic":
			value, err := parseNextInt16(args, &i, "--dying-graphic")
			if err != nil {
				return patch, err
			}
			patch.DyingGraphic = &value
		case "--blood-unit":
			value, err := parseNextInt16(args, &i, "--blood-unit")
			if err != nil {
				return patch, err
			}
			patch.BloodUnitID = &value
		case "--icon-id":
			value, err := parseNextInt16(args, &i, "--icon-id")
			if err != nil {
				return patch, err
			}
			patch.IconID = &value
		case "--enabled":
			if i+1 >= len(args) {
				return patch, fmt.Errorf("--enabled needs a value")
			}
			i++
			value, err := strconv.ParseUint(args[i], 10, 8)
			if err != nil {
				return patch, fmt.Errorf("invalid --enabled %q: %w", args[i], err)
			}
			enabled := uint8(value)
			patch.Enabled = &enabled
		case "--type50-projectile-unit":
			value, err := parseNextInt16(args, &i, "--type50-projectile-unit")
			if err != nil {
				return patch, err
			}
			patch.Type50ProjectileUnitID = &value
		case "--type50-max-range":
			value, err := parseNextFloat32(args, &i, "--type50-max-range")
			if err != nil {
				return patch, err
			}
			patch.Type50MaxRange = &value
		case "--type50-blast-width":
			value, err := parseNextFloat32(args, &i, "--type50-blast-width")
			if err != nil {
				return patch, err
			}
			patch.Type50BlastWidth = &value
		case "--type50-attack-graphic":
			value, err := parseNextInt16(args, &i, "--type50-attack-graphic")
			if err != nil {
				return patch, err
			}
			patch.Type50AttackGraphic = &value
		case "--type50-blast-damage":
			value, err := parseNextFloat32(args, &i, "--type50-blast-damage")
			if err != nil {
				return patch, err
			}
			patch.Type50BlastDamage = &value
		case "--train-time":
			value, err := parseNextInt16(args, &i, "--train-time")
			if err != nil {
				return patch, err
			}
			patch.TrainTime0 = &value
		case "--train-unit":
			value, err := parseNextInt16(args, &i, "--train-unit")
			if err != nil {
				return patch, err
			}
			patch.TrainUnitID0 = &value
		case "--train-button":
			if i+1 >= len(args) {
				return patch, fmt.Errorf("--train-button needs a value")
			}
			i++
			value, err := strconv.ParseUint(args[i], 10, 8)
			if err != nil {
				return patch, fmt.Errorf("invalid --train-button %q: %w", args[i], err)
			}
			button := uint8(value)
			patch.TrainButtonID0 = &button
		case "--train-hotkey":
			value, err := parseNextInt32(args, &i, "--train-hotkey")
			if err != nil {
				return patch, err
			}
			patch.TrainHotKeyID0 = &value
		case "--creatable-button-icon":
			value, err := parseNextInt16(args, &i, "--creatable-button-icon")
			if err != nil {
				return patch, err
			}
			patch.CreatableButtonIconID = &value
		case "--creatable-hotkey-action":
			value, err := parseNextInt16(args, &i, "--creatable-hotkey-action")
			if err != nil {
				return patch, err
			}
			patch.CreatableButtonHotkeyAction = &value
		case "--attribute":
			item, err := parseUnitAttributePatchFlag(args, &i, "--attribute")
			if err != nil {
				return patch, err
			}
			patch.Attributes = append(patch.Attributes, item)
		case "--damage-graphic":
			item, err := parseDamageGraphicPatchFlag(args, &i, "--damage-graphic")
			if err != nil {
				return patch, err
			}
			patch.DamageGraphics = append(patch.DamageGraphics, item)
		case "--type50-attack":
			item, err := parseWeaponInfoPatchFlag(args, &i, "--type50-attack")
			if err != nil {
				return patch, err
			}
			patch.Type50Attacks = append(patch.Type50Attacks, item)
		case "--type50-armour", "--type50-armor":
			item, err := parseWeaponInfoPatchFlag(args, &i, args[i])
			if err != nil {
				return patch, err
			}
			patch.Type50Armours = append(patch.Type50Armours, item)
		case "--cost":
			item, err := parseAttributeCostPatchFlag(args, &i, "--cost")
			if err != nil {
				return patch, err
			}
			patch.Costs = append(patch.Costs, item)
		case "--train-location":
			item, err := parseTrainLocationPatchFlag(args, &i, "--train-location")
			if err != nil {
				return patch, err
			}
			patch.TrainLocations = append(patch.TrainLocations, item)
		case "--task":
			item, err := parseTaskPatchFlag(args, &i, "--task")
			if err != nil {
				return patch, err
			}
			patch.Tasks = append(patch.Tasks, item)
		default:
			return patch, fmt.Errorf("unknown unit patch option %q", args[i])
		}
	}
	if patch.Empty() {
		return patch, fmt.Errorf("provide at least one unit patch option")
	}
	return patch, nil
}

func parseDatUnitCreateOptions(args []string) (datfile.UnitCreateRecipe, error) {
	var recipe datfile.UnitCreateRecipe
	fromCivSet := false
	fromUnitSet := false
	sawCivScope := false
	var patchArgs []string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--from-civ":
			value, err := parseNextInt(args, &i, "--from-civ")
			if err != nil {
				return recipe, err
			}
			recipe.FromCivID = value
			fromCivSet = true
		case "--from-unit":
			value, err := parseNextInt(args, &i, "--from-unit")
			if err != nil {
				return recipe, err
			}
			recipe.FromUnitID = value
			fromUnitSet = true
		case "--civ":
			value, err := parseNextIntList(args, &i, "--civ")
			if err != nil {
				return recipe, err
			}
			recipe.CivIDs = append(recipe.CivIDs, value...)
			sawCivScope = true
		case "--all-civs":
			all := true
			recipe.AllCivs = &all
			sawCivScope = true
		default:
			patchArgs = append(patchArgs, args[i])
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "--") {
				i++
				patchArgs = append(patchArgs, args[i])
			}
		}
	}
	if !fromCivSet {
		return recipe, errors.New("--from-civ is required")
	}
	if !fromUnitSet {
		return recipe, errors.New("--from-unit is required")
	}
	if !sawCivScope {
		return recipe, errors.New("set --civ N or --all-civs")
	}
	if recipe.AllCivs != nil && len(recipe.CivIDs) > 0 {
		return recipe, errors.New("set either --all-civs or --civ, not both")
	}
	if len(patchArgs) > 0 {
		patch, err := parseUnitPatchOptions(patchArgs)
		if err != nil {
			return recipe, err
		}
		applyUnitPatchToCreateRecipe(&recipe, patch)
	}
	return recipe, nil
}

func parseDatUnitDeleteRequest(unitID int, args []string) (datcodec.DeletePlanRequest, error) {
	if unitID < 0 {
		return datcodec.DeletePlanRequest{}, fmt.Errorf("negative unit id %d", unitID)
	}
	request := datcodec.DeletePlanRequest{Section: "unit", ID: unitID}
	sawScope := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--all-civs":
			request.AllCivs = true
			sawScope = true
		case "--civ":
			value, err := parseNextInt(args, &i, "--civ")
			if err != nil {
				return request, err
			}
			request.CivID = &value
			sawScope = true
		default:
			return request, fmt.Errorf("unknown kit dat unit-delete option %q", args[i])
		}
	}
	if !sawScope {
		return request, errors.New("unit-delete needs --civ N or --all-civs")
	}
	if request.AllCivs && request.CivID != nil {
		return request, errors.New("set either --all-civs or --civ, not both")
	}
	return request, nil
}

func parseDatCivPatchOptions(id int, args []string) (datcodec.CivPatchRecipe, error) {
	if id < 0 {
		return datcodec.CivPatchRecipe{}, fmt.Errorf("negative civ id %d", id)
	}
	recipe := datcodec.CivPatchRecipe{ID: id}
	changed := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--name":
			value, err := parseNextString(args, &i, "--name")
			if err != nil {
				return recipe, err
			}
			recipe.Name = &value
			changed = true
		case "--tech-tree-id":
			value, err := parseNextInt16(args, &i, "--tech-tree-id")
			if err != nil {
				return recipe, err
			}
			recipe.TechTreeID = &value
			changed = true
		case "--team-bonus-id":
			value, err := parseNextInt16(args, &i, "--team-bonus-id")
			if err != nil {
				return recipe, err
			}
			recipe.TeamBonusID = &value
			changed = true
		case "--icon-set":
			value, err := parseNextUint8(args, &i, "--icon-set")
			if err != nil {
				return recipe, err
			}
			recipe.IconSet = &value
			changed = true
		case "--resource":
			resource, err := parseNextCivResourcePatch(args, &i, "--resource")
			if err != nil {
				return recipe, err
			}
			recipe.Resources = append(recipe.Resources, resource)
			changed = true
		default:
			return recipe, fmt.Errorf("unknown kit dat civ-patch option %q", args[i])
		}
	}
	if !changed {
		return recipe, errors.New("civ-patch has no operations")
	}
	return recipe, nil
}

func parseDatTerrainRestrictionPatchOptions(id int, args []string) (datcodec.TerrainRestrictionPatchRecipe, error) {
	if id < 0 {
		return datcodec.TerrainRestrictionPatchRecipe{}, fmt.Errorf("negative terrain restriction id %d", id)
	}
	recipe := datcodec.TerrainRestrictionPatchRecipe{ID: id}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--terrain", "--row":
			row, err := parseNextTerrainPassabilityPatch(args, &i, args[i])
			if err != nil {
				return recipe, err
			}
			recipe.Rows = append(recipe.Rows, row)
		default:
			return recipe, fmt.Errorf("unknown kit dat terrain-restriction-patch option %q", args[i])
		}
	}
	if len(recipe.Rows) == 0 {
		return recipe, errors.New("terrain-restriction-patch needs at least one --terrain TERRAIN_ID,PASSABILITY")
	}
	return recipe, nil
}

func parseDatTerrainPatchOptions(id int, args []string) (datcodec.TerrainPatchRecipe, error) {
	if id < 0 {
		return datcodec.TerrainPatchRecipe{}, fmt.Errorf("negative terrain id %d", id)
	}
	recipe := datcodec.TerrainPatchRecipe{ID: id}
	changed := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--name":
			value, err := parseNextString(args, &i, "--name")
			if err != nil {
				return recipe, err
			}
			recipe.Name = &value
			changed = true
		case "--name-2":
			value, err := parseNextString(args, &i, "--name-2")
			if err != nil {
				return recipe, err
			}
			recipe.Name2 = &value
			changed = true
		case "--overlay-mask-name":
			value, err := parseNextString(args, &i, "--overlay-mask-name")
			if err != nil {
				return recipe, err
			}
			recipe.OverlayMaskName = &value
			changed = true
		default:
			return recipe, fmt.Errorf("unknown kit dat terrain-patch option %q", args[i])
		}
	}
	if !changed {
		return recipe, errors.New("terrain-patch has no operations")
	}
	return recipe, nil
}

func parseDatAvailabilitySetOptions(unitID int, args []string) (datcodec.UnitAvailabilityRecipe, error) {
	if unitID < 0 {
		return datcodec.UnitAvailabilityRecipe{}, fmt.Errorf("negative unit id %d", unitID)
	}
	recipe := datcodec.UnitAvailabilityRecipe{UnitID: unitID}
	sawScope := false
	enabledSet := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--all-civs":
			recipe.AllCivs = true
			sawScope = true
		case "--civ":
			value, err := parseNextIntList(args, &i, "--civ")
			if err != nil {
				return recipe, err
			}
			recipe.CivIDs = append(recipe.CivIDs, value...)
			sawScope = true
		case "--enabled":
			value, err := parseNextBoolish(args, &i, "--enabled")
			if err != nil {
				return recipe, err
			}
			recipe.Enabled = value
			enabledSet = true
		default:
			return recipe, fmt.Errorf("unknown kit dat availability-set option %q", args[i])
		}
	}
	if !sawScope {
		return recipe, errors.New("availability-set needs --civ N or --all-civs")
	}
	if recipe.AllCivs && len(recipe.CivIDs) > 0 {
		return recipe, errors.New("set either --all-civs or --civ, not both")
	}
	if !enabledSet {
		return recipe, errors.New("availability-set needs --enabled true|false|1|0")
	}
	if !recipe.AllCivs && len(recipe.CivIDs) == 1 {
		recipe.CivID = &recipe.CivIDs[0]
		recipe.CivIDs = nil
	}
	return recipe, nil
}

func parseNextCivResourcePatch(args []string, i *int, name string) (datcodec.CivResourcePatch, error) {
	if *i+1 >= len(args) {
		return datcodec.CivResourcePatch{}, fmt.Errorf("%s needs INDEX,VALUE", name)
	}
	*i = *i + 1
	parts := strings.Split(args[*i], ",")
	if len(parts) != 2 {
		return datcodec.CivResourcePatch{}, fmt.Errorf("%s %q needs INDEX,VALUE", name, args[*i])
	}
	index, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil || index < 0 {
		if err == nil {
			err = fmt.Errorf("negative index")
		}
		return datcodec.CivResourcePatch{}, fmt.Errorf("%s invalid resource index %q: %w", name, parts[0], err)
	}
	value, err := parseFloat32(strings.TrimSpace(parts[1]), name+".value")
	if err != nil {
		return datcodec.CivResourcePatch{}, err
	}
	return datcodec.CivResourcePatch{Index: index, Value: value}, nil
}

func parseNextTerrainPassabilityPatch(args []string, i *int, name string) (datcodec.TerrainPassabilityPatch, error) {
	if *i+1 >= len(args) {
		return datcodec.TerrainPassabilityPatch{}, fmt.Errorf("%s needs TERRAIN_ID,PASSABILITY", name)
	}
	*i = *i + 1
	parts := strings.Split(args[*i], ",")
	if len(parts) != 2 {
		return datcodec.TerrainPassabilityPatch{}, fmt.Errorf("%s %q needs TERRAIN_ID,PASSABILITY", name, args[*i])
	}
	terrainID, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil || terrainID < 0 {
		if err == nil {
			err = fmt.Errorf("negative terrain id")
		}
		return datcodec.TerrainPassabilityPatch{}, fmt.Errorf("%s invalid terrain id %q: %w", name, parts[0], err)
	}
	passability, err := parseFloat32(strings.TrimSpace(parts[1]), name+".passability")
	if err != nil {
		return datcodec.TerrainPassabilityPatch{}, err
	}
	return datcodec.TerrainPassabilityPatch{TerrainID: terrainID, Passability: passability}, nil
}

func applyUnitPatchToCreateRecipe(recipe *datfile.UnitCreateRecipe, patch datfile.UnitPatch) {
	recipe.Class = patch.Class
	recipe.HitPoints = patch.HitPoints
	recipe.LineOfSight = patch.LineOfSight
	recipe.MovementType = patch.MovementType
	recipe.StandingGraphic1 = patch.StandingGraphic1
	recipe.StandingGraphic2 = patch.StandingGraphic2
	recipe.DyingGraphic = patch.DyingGraphic
	recipe.BloodUnitID = patch.BloodUnitID
	recipe.IconID = patch.IconID
	recipe.Enabled = patch.Enabled
	recipe.Attributes = patch.Attributes
	recipe.DamageGraphics = patch.DamageGraphics
	recipe.SetDamageGraphics = patch.SetDamageGraphics
	recipe.Type50ProjectileUnitID = patch.Type50ProjectileUnitID
	recipe.Type50MaxRange = patch.Type50MaxRange
	recipe.Type50BlastWidth = patch.Type50BlastWidth
	recipe.Type50AttackGraphic = patch.Type50AttackGraphic
	recipe.Type50BlastDamage = patch.Type50BlastDamage
	recipe.Type50Attacks = patch.Type50Attacks
	recipe.Type50Armours = patch.Type50Armours
	recipe.SetType50Attacks = patch.SetType50Attacks
	recipe.SetType50Armours = patch.SetType50Armours
	recipe.TrainTime0 = patch.TrainTime0
	recipe.TrainUnitID0 = patch.TrainUnitID0
	recipe.TrainButtonID0 = patch.TrainButtonID0
	recipe.TrainHotKeyID0 = patch.TrainHotKeyID0
	recipe.CreatableButtonIconID = patch.CreatableButtonIconID
	recipe.CreatableButtonHotkeyAction = patch.CreatableButtonHotkeyAction
	recipe.Costs = patch.Costs
	recipe.TrainLocations = patch.TrainLocations
	recipe.SetTrainLocations = patch.SetTrainLocations
	recipe.SetDropSites = patch.SetDropSites
	recipe.LinkedBuildings = patch.LinkedBuildings
	recipe.Tasks = patch.Tasks
	recipe.SetTasks = patch.SetTasks
}

func parseUnitAttributePatchFlag(args []string, i *int, name string) (datfile.UnitAttributePatch, error) {
	index, fields, err := parseRowPatchSpec(args, i, name)
	if err != nil {
		return datfile.UnitAttributePatch{}, err
	}
	out := datfile.UnitAttributePatch{Index: index}
	for key, raw := range fields {
		switch key {
		case "attribute_type", "type":
			value, err := parseUint16(raw, name+"."+key)
			if err != nil {
				return out, err
			}
			out.AttributeType = &value
		case "amount":
			value, err := parseFloat32(raw, name+"."+key)
			if err != nil {
				return out, err
			}
			out.Amount = &value
		case "flag":
			value, err := parseUint8(raw, name+"."+key)
			if err != nil {
				return out, err
			}
			out.Flag = &value
		default:
			return out, fmt.Errorf("%s unknown field %q", name, key)
		}
	}
	return out, nil
}

func parseDamageGraphicPatchFlag(args []string, i *int, name string) (datfile.DamageGraphicPatch, error) {
	index, fields, err := parseRowPatchSpec(args, i, name)
	if err != nil {
		return datfile.DamageGraphicPatch{}, err
	}
	out := datfile.DamageGraphicPatch{Index: index}
	for key, raw := range fields {
		switch key {
		case "graphic_id":
			value, err := parseUint16(raw, name+"."+key)
			if err != nil {
				return out, err
			}
			out.GraphicID = &value
		case "damage_percent":
			value, err := parseUint16(raw, name+"."+key)
			if err != nil {
				return out, err
			}
			out.DamagePercent = &value
		case "flag":
			value, err := parseUint8(raw, name+"."+key)
			if err != nil {
				return out, err
			}
			out.Flag = &value
		default:
			return out, fmt.Errorf("%s unknown field %q", name, key)
		}
	}
	return out, nil
}

func parseWeaponInfoPatchFlag(args []string, i *int, name string) (datfile.WeaponInfoPatch, error) {
	index, fields, err := parseRowPatchSpec(args, i, name)
	if err != nil {
		return datfile.WeaponInfoPatch{}, err
	}
	out := datfile.WeaponInfoPatch{Index: index}
	for key, raw := range fields {
		switch key {
		case "class":
			value, err := parseInt16(raw, name+"."+key)
			if err != nil {
				return out, err
			}
			out.Class = &value
		case "value":
			value, err := parseInt16(raw, name+"."+key)
			if err != nil {
				return out, err
			}
			out.Value = &value
		default:
			return out, fmt.Errorf("%s unknown field %q", name, key)
		}
	}
	return out, nil
}

func parseAttributeCostPatchFlag(args []string, i *int, name string) (datfile.AttributeCostPatch, error) {
	index, fields, err := parseRowPatchSpec(args, i, name)
	if err != nil {
		return datfile.AttributeCostPatch{}, err
	}
	out := datfile.AttributeCostPatch{Index: index}
	for key, raw := range fields {
		switch key {
		case "attribute_type", "type":
			value, err := parseInt16(raw, name+"."+key)
			if err != nil {
				return out, err
			}
			out.AttributeType = &value
		case "amount":
			value, err := parseInt16(raw, name+"."+key)
			if err != nil {
				return out, err
			}
			out.Amount = &value
		case "flag":
			value, err := parseUint8(raw, name+"."+key)
			if err != nil {
				return out, err
			}
			out.Flag = &value
		case "padding":
			value, err := parseUint8(raw, name+"."+key)
			if err != nil {
				return out, err
			}
			out.Padding = &value
		default:
			return out, fmt.Errorf("%s unknown field %q", name, key)
		}
	}
	return out, nil
}

func parseTrainLocationPatchFlag(args []string, i *int, name string) (datfile.TrainLocationPatch, error) {
	index, fields, err := parseRowPatchSpec(args, i, name)
	if err != nil {
		return datfile.TrainLocationPatch{}, err
	}
	out := datfile.TrainLocationPatch{Index: index}
	for key, raw := range fields {
		switch key {
		case "train_time", "time":
			value, err := parseInt16(raw, name+"."+key)
			if err != nil {
				return out, err
			}
			out.TrainTime = &value
		case "train_unit_id", "unit":
			value, err := parseInt16(raw, name+"."+key)
			if err != nil {
				return out, err
			}
			out.TrainUnitID = &value
		case "train_button", "button":
			value, err := parseUint8(raw, name+"."+key)
			if err != nil {
				return out, err
			}
			out.TrainButton = &value
		case "train_hotkey", "hotkey":
			value, err := parseInt32(raw, name+"."+key)
			if err != nil {
				return out, err
			}
			out.TrainHotkey = &value
		default:
			return out, fmt.Errorf("%s unknown field %q", name, key)
		}
	}
	return out, nil
}

func parseTaskPatchFlag(args []string, i *int, name string) (datfile.TaskPatch, error) {
	index, fields, err := parseRowPatchSpec(args, i, name)
	if err != nil {
		return datfile.TaskPatch{}, err
	}
	out := datfile.TaskPatch{Index: index}
	for key, raw := range fields {
		switch key {
		case "record_type":
			value, err := parseInt16(raw, name+"."+key)
			if err != nil {
				return out, err
			}
			out.RecordType = &value
		case "id":
			value, err := parseInt16(raw, name+"."+key)
			if err != nil {
				return out, err
			}
			out.ID = &value
		case "is_default":
			value, err := parseUint8(raw, name+"."+key)
			if err != nil {
				return out, err
			}
			out.IsDefault = &value
		case "action_type":
			value, err := parseInt16(raw, name+"."+key)
			if err != nil {
				return out, err
			}
			out.ActionType = &value
		case "object_class":
			value, err := parseInt16(raw, name+"."+key)
			if err != nil {
				return out, err
			}
			out.ObjectClass = &value
		case "object_id":
			value, err := parseInt16(raw, name+"."+key)
			if err != nil {
				return out, err
			}
			out.ObjectID = &value
		case "terrain_id":
			value, err := parseInt16(raw, name+"."+key)
			if err != nil {
				return out, err
			}
			out.TerrainID = &value
		case "attribute_types":
			values, err := parseInt16List(raw, name+"."+key, 4)
			if err != nil {
				return out, err
			}
			out.AttributeTypes = values
		case "work_value_1":
			value, err := parseFloat32(raw, name+"."+key)
			if err != nil {
				return out, err
			}
			out.WorkValue1 = &value
		case "work_value_2":
			value, err := parseFloat32(raw, name+"."+key)
			if err != nil {
				return out, err
			}
			out.WorkValue2 = &value
		case "work_range":
			value, err := parseFloat32(raw, name+"."+key)
			if err != nil {
				return out, err
			}
			out.WorkRange = &value
		case "auto_search_targets":
			value, err := parseUint8(raw, name+"."+key)
			if err != nil {
				return out, err
			}
			out.AutoSearchTargets = &value
		case "search_wait_time":
			value, err := parseFloat32(raw, name+"."+key)
			if err != nil {
				return out, err
			}
			out.SearchWaitTime = &value
		case "enable_targeting":
			value, err := parseUint8(raw, name+"."+key)
			if err != nil {
				return out, err
			}
			out.EnableTargeting = &value
		case "combat_level":
			value, err := parseUint8(raw, name+"."+key)
			if err != nil {
				return out, err
			}
			out.CombatLevel = &value
		default:
			return out, fmt.Errorf("%s unknown field %q", name, key)
		}
	}
	return out, nil
}

func parseRowPatchSpec(args []string, i *int, name string) (int, map[string]string, error) {
	if *i+1 >= len(args) {
		return 0, nil, fmt.Errorf("%s needs INDEX,field=value[,field=value...]", name)
	}
	*i = *i + 1
	raw := args[*i]
	parts := strings.Split(raw, ",")
	if len(parts) < 2 {
		return 0, nil, fmt.Errorf("%s %q needs INDEX,field=value[,field=value...]", name, raw)
	}
	index, err := strconv.Atoi(parts[0])
	if err != nil || index < 0 {
		if err == nil {
			err = fmt.Errorf("negative index")
		}
		return 0, nil, fmt.Errorf("%s invalid row index %q: %w", name, parts[0], err)
	}
	fields := make(map[string]string, len(parts)-1)
	for _, part := range parts[1:] {
		key, value, ok := strings.Cut(part, "=")
		if !ok || key == "" {
			return 0, nil, fmt.Errorf("%s invalid field spec %q; use field=value", name, part)
		}
		if value == "" {
			return 0, nil, fmt.Errorf("%s field %q needs a value", name, key)
		}
		if _, exists := fields[key]; exists {
			return 0, nil, fmt.Errorf("%s duplicate field %q", name, key)
		}
		fields[key] = value
	}
	return index, fields, nil
}

func parseNextInt(args []string, i *int, name string) (int, error) {
	if *i+1 >= len(args) {
		return 0, fmt.Errorf("%s needs a value", name)
	}
	*i = *i + 1
	value, err := strconv.Atoi(args[*i])
	if err != nil {
		return 0, fmt.Errorf("invalid %s %q: %w", name, args[*i], err)
	}
	return value, nil
}

func parseNextInt16(args []string, i *int, name string) (int16, error) {
	if *i+1 >= len(args) {
		return 0, fmt.Errorf("%s needs a value", name)
	}
	*i = *i + 1
	return parseInt16(args[*i], name)
}

func parseNextInt8(args []string, i *int, name string) (int8, error) {
	if *i+1 >= len(args) {
		return 0, fmt.Errorf("%s needs a value", name)
	}
	*i = *i + 1
	return parseInt8(args[*i], name)
}

func parseNextUint8(args []string, i *int, name string) (uint8, error) {
	if *i+1 >= len(args) {
		return 0, fmt.Errorf("%s needs a value", name)
	}
	*i = *i + 1
	return parseUint8(args[*i], name)
}

func parseNextUint32(args []string, i *int, name string) (uint32, error) {
	if *i+1 >= len(args) {
		return 0, fmt.Errorf("%s needs a value", name)
	}
	*i = *i + 1
	return parseUint32(args[*i], name)
}

func parseNextInt32(args []string, i *int, name string) (int32, error) {
	if *i+1 >= len(args) {
		return 0, fmt.Errorf("%s needs a value", name)
	}
	*i = *i + 1
	return parseInt32(args[*i], name)
}

func parseNextFloat32(args []string, i *int, name string) (float32, error) {
	if *i+1 >= len(args) {
		return 0, fmt.Errorf("%s needs a value", name)
	}
	*i = *i + 1
	return parseFloat32(args[*i], name)
}

func parseNextInt16List(args []string, i *int, name string, want int) ([]int16, error) {
	if *i+1 >= len(args) {
		return nil, fmt.Errorf("%s needs a value", name)
	}
	*i = *i + 1
	return parseInt16List(args[*i], name, want)
}

func parseNextIntList(args []string, i *int, name string) ([]int, error) {
	if *i+1 >= len(args) {
		return nil, fmt.Errorf("%s needs a value", name)
	}
	*i = *i + 1
	return parseIntList(args[*i], name)
}

func parseNextInt32List(args []string, i *int, name string) ([]int32, error) {
	values, err := parseNextIntList(args, i, name)
	if err != nil {
		return nil, err
	}
	out := make([]int32, 0, len(values))
	for idx, value := range values {
		if value < math.MinInt32 || value > math.MaxInt32 {
			return nil, fmt.Errorf("%s[%d] %d outside int32 range", name, idx, value)
		}
		out = append(out, int32(value))
	}
	return out, nil
}

func parseNextUint8List(args []string, i *int, name string) ([]uint8, error) {
	values, err := parseNextIntList(args, i, name)
	if err != nil {
		return nil, err
	}
	out := make([]uint8, 0, len(values))
	for idx, value := range values {
		if value < 0 || value > math.MaxUint8 {
			return nil, fmt.Errorf("%s[%d] %d outside uint8 range", name, idx, value)
		}
		out = append(out, uint8(value))
	}
	return out, nil
}

func parseIntList(raw, name string) ([]int, error) {
	parts := strings.Split(raw, ",")
	out := make([]int, 0, len(parts))
	for idx, part := range parts {
		value, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil {
			return nil, fmt.Errorf("invalid %s[%d] %q: %w", name, idx, part, err)
		}
		out = append(out, value)
	}
	return out, nil
}

func parseFloat32(raw, name string) (float32, error) {
	value, err := strconv.ParseFloat(raw, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid %s %q: %w", name, raw, err)
	}
	return float32(value), nil
}

func parseInt32(raw, name string) (int32, error) {
	value, err := strconv.ParseInt(raw, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid %s %q: %w", name, raw, err)
	}
	return int32(value), nil
}

func parseInt8(raw, name string) (int8, error) {
	value, err := strconv.ParseInt(raw, 10, 8)
	if err != nil {
		return 0, fmt.Errorf("invalid %s %q: %w", name, raw, err)
	}
	return int8(value), nil
}

func parseInt16(raw, name string) (int16, error) {
	value, err := strconv.ParseInt(raw, 10, 16)
	if err != nil {
		return 0, fmt.Errorf("invalid %s %q: %w", name, raw, err)
	}
	return int16(value), nil
}

func parseUint16(raw, name string) (uint16, error) {
	value, err := strconv.ParseUint(raw, 10, 16)
	if err != nil {
		return 0, fmt.Errorf("invalid %s %q: %w", name, raw, err)
	}
	return uint16(value), nil
}

func parseUint32(raw, name string) (uint32, error) {
	value, err := strconv.ParseUint(raw, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid %s %q: %w", name, raw, err)
	}
	return uint32(value), nil
}

func parseUint8(raw, name string) (uint8, error) {
	value, err := strconv.ParseUint(raw, 10, 8)
	if err != nil {
		return 0, fmt.Errorf("invalid %s %q: %w", name, raw, err)
	}
	return uint8(value), nil
}

func parseInt16List(raw, name string, want int) ([]int16, error) {
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ':' || r == '/' || r == '|' || r == ','
	})
	if len(parts) != want {
		return nil, fmt.Errorf("%s length=%d, want %d", name, len(parts), want)
	}
	out := make([]int16, 0, len(parts))
	for i, part := range parts {
		value, err := parseInt16(part, fmt.Sprintf("%s[%d]", name, i))
		if err != nil {
			return nil, err
		}
		out = append(out, value)
	}
	return out, nil
}

type datGraphicsOptions struct {
	Limit        int
	All          bool
	NameContains string
	ParticleOnly bool
	Spans        bool
}

func (o datGraphicsOptions) Filter() datfile.GraphicFilter {
	return datfile.GraphicFilter{
		NameContains: o.NameContains,
		ParticleOnly: o.ParticleOnly,
	}
}

func parseDatGraphicsOptions(args []string, label string) (datGraphicsOptions, error) {
	opts := datGraphicsOptions{Limit: 200}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--all":
			opts.All = true
		case "--limit":
			if i+1 >= len(args) {
				return datGraphicsOptions{}, fmt.Errorf("--limit needs a value")
			}
			i++
			limit, err := strconv.Atoi(args[i])
			if err != nil {
				return datGraphicsOptions{}, fmt.Errorf("invalid --limit %q: %w", args[i], err)
			}
			if limit < 0 {
				return datGraphicsOptions{}, fmt.Errorf("--limit must be non-negative")
			}
			opts.Limit = limit
			if limit == 0 {
				opts.All = true
			}
		case "--name-contains":
			if i+1 >= len(args) {
				return datGraphicsOptions{}, fmt.Errorf("--name-contains needs a value")
			}
			i++
			opts.NameContains = args[i]
		case "--particle":
			opts.ParticleOnly = true
		case "--spans", "--raw":
			opts.Spans = true
		default:
			return datGraphicsOptions{}, fmt.Errorf("unknown %s option %q", label, args[i])
		}
	}
	return opts, nil
}

func parseDatPaletteOptions(args []string) (datfile.PaletteOptions, bool, error) {
	opts := datfile.PaletteOptions{}
	textOut := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--text":
			textOut = true
		case "--json":
			textOut = false
		case "--id":
			if i+1 >= len(args) {
				return opts, textOut, fmt.Errorf("--id needs a unit id")
			}
			i++
			id, err := strconv.Atoi(args[i])
			if err != nil || id < 0 {
				return opts, textOut, fmt.Errorf("--id must be a non-negative integer")
			}
			opts.ID = &id
		case "--min-variants":
			if i+1 >= len(args) {
				return opts, textOut, fmt.Errorf("--min-variants needs an integer")
			}
			i++
			minVariants, err := strconv.Atoi(args[i])
			if err != nil || minVariants < 0 {
				return opts, textOut, fmt.Errorf("--min-variants must be a non-negative integer")
			}
			opts.MinVariants = minVariants
		default:
			return opts, textOut, fmt.Errorf("unknown kit dat palette option %q", args[i])
		}
	}
	return opts, textOut, nil
}

func parseDatListOptions(args []string, label string) (limit int, all bool, err error) {
	limit = 200
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--all":
			all = true
		case "--json":
			// JSON is the default for these read-only list commands; accept the
			// explicit flag so scripts can use a uniform output option.
		case "--limit":
			if i+1 >= len(args) {
				return 0, false, fmt.Errorf("--limit needs a value")
			}
			i++
			limit, err = strconv.Atoi(args[i])
			if err != nil {
				return 0, false, fmt.Errorf("invalid --limit %q: %w", args[i], err)
			}
		default:
			return 0, false, fmt.Errorf("unknown %s option %q", label, args[i])
		}
	}
	if limit < 1 {
		return 0, false, fmt.Errorf("--limit must be positive")
	}
	return limit, all, nil
}

func parseDatDiffOptions(args []string) (datfile.DiffOptions, bool, error) {
	opts := datfile.DiffOptions{Limit: 200}
	textOut := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--all":
			opts.All = true
		case "--text":
			textOut = true
		case "--json":
			textOut = false
		case "--limit":
			if i+1 >= len(args) {
				return opts, false, fmt.Errorf("--limit needs a value")
			}
			i++
			limit, err := strconv.Atoi(args[i])
			if err != nil {
				return opts, false, fmt.Errorf("invalid --limit %q: %w", args[i], err)
			}
			opts.Limit = limit
		case "--section":
			if i+1 >= len(args) {
				return opts, false, fmt.Errorf("--section needs a value")
			}
			i++
			opts.Sections = append(opts.Sections, args[i])
		default:
			return opts, false, fmt.Errorf("unknown kit dat diff option %q", args[i])
		}
	}
	if opts.Limit < 0 {
		return opts, false, fmt.Errorf("--limit must be non-negative")
	}
	return opts, textOut, nil
}

type datEffectListOptions struct {
	Limit       int
	All         bool
	CommandType *int
	Operand     *int
}

type datCommandMatrixOptions struct {
	ExampleLimit   int
	Text           bool
	CommandType    *int
	Operand        *int
	ReferenceKind  string
	ReferenceField string
	UnknownOnly    bool
	TypedOnly      bool
	CandidateOnly  bool
}

type datTechListOptions struct {
	Limit        int    `json:"limit"`
	All          bool   `json:"all,omitempty"`
	NameContains string `json:"name_contains,omitempty"`
}

type datAbilityListOptions struct {
	Limit        int    `json:"limit"`
	All          bool   `json:"all,omitempty"`
	NameContains string `json:"name_contains,omitempty"`
	EffectID     *int   `json:"effect_id,omitempty"`
	CommandType  *int   `json:"command_type,omitempty"`
	Operand      *int   `json:"operand,omitempty"`
}

type datAbilityPatchOptions struct {
	Recipe    datcodec.Recipe
	HasTech   bool
	HasEffect bool
}

func parseDatEffectListOptions(args []string, label string) (datEffectListOptions, error) {
	opts := datEffectListOptions{Limit: 200}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--all":
			opts.All = true
		case "--limit":
			if i+1 >= len(args) {
				return opts, fmt.Errorf("--limit needs a value")
			}
			i++
			limit, err := strconv.Atoi(args[i])
			if err != nil {
				return opts, fmt.Errorf("invalid --limit %q: %w", args[i], err)
			}
			opts.Limit = limit
		case "--command-type":
			if i+1 >= len(args) {
				return opts, fmt.Errorf("--command-type needs a value")
			}
			i++
			value, err := strconv.Atoi(args[i])
			if err != nil {
				return opts, fmt.Errorf("invalid --command-type %q: %w", args[i], err)
			}
			opts.CommandType = &value
		case "--operand":
			if i+1 >= len(args) {
				return opts, fmt.Errorf("--operand needs a value")
			}
			i++
			value, err := strconv.Atoi(args[i])
			if err != nil {
				return opts, fmt.Errorf("invalid --operand %q: %w", args[i], err)
			}
			opts.Operand = &value
		default:
			return opts, fmt.Errorf("unknown %s option %q", label, args[i])
		}
	}
	if opts.Limit < 1 {
		return opts, fmt.Errorf("--limit must be positive")
	}
	return opts, nil
}

func parseDatCommandMatrixOptions(args []string) (datCommandMatrixOptions, error) {
	opts := datCommandMatrixOptions{ExampleLimit: 3}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--text":
			opts.Text = true
		case "--json":
			opts.Text = false
		case "--limit":
			if i+1 >= len(args) {
				return opts, fmt.Errorf("--limit needs a value")
			}
			i++
			limit, err := strconv.Atoi(args[i])
			if err != nil {
				return opts, fmt.Errorf("invalid --limit %q: %w", args[i], err)
			}
			opts.ExampleLimit = limit
		case "--all":
			opts.ExampleLimit = int(^uint(0) >> 1)
		case "--command-type":
			if i+1 >= len(args) {
				return opts, fmt.Errorf("--command-type needs a value")
			}
			i++
			value, err := strconv.Atoi(args[i])
			if err != nil {
				return opts, fmt.Errorf("invalid --command-type %q: %w", args[i], err)
			}
			if value < 0 || value > 255 {
				return opts, fmt.Errorf("--command-type must be 0..255")
			}
			opts.CommandType = &value
		case "--operand":
			if i+1 >= len(args) {
				return opts, fmt.Errorf("--operand needs a value")
			}
			i++
			value, err := strconv.Atoi(args[i])
			if err != nil {
				return opts, fmt.Errorf("invalid --operand %q: %w", args[i], err)
			}
			opts.Operand = &value
		case "--reference-kind":
			if i+1 >= len(args) {
				return opts, fmt.Errorf("--reference-kind needs a value")
			}
			i++
			switch args[i] {
			case "unit", "tech":
				opts.ReferenceKind = args[i]
			default:
				return opts, fmt.Errorf("--reference-kind must be unit or tech")
			}
		case "--reference-field":
			if i+1 >= len(args) {
				return opts, fmt.Errorf("--reference-field needs a value")
			}
			i++
			opts.ReferenceField = args[i]
		case "--unknown-only", "--unknown":
			opts.UnknownOnly = true
		case "--typed-only", "--typed":
			opts.TypedOnly = true
		case "--candidate-only", "--candidate":
			opts.CandidateOnly = true
		default:
			return opts, fmt.Errorf("unknown kit dat command-matrix option %q", args[i])
		}
	}
	if opts.ExampleLimit < 0 {
		return opts, fmt.Errorf("--limit must be non-negative")
	}
	return opts, nil
}

func parseDatTechListOptions(args []string) (datTechListOptions, error) {
	opts := datTechListOptions{Limit: 200}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--all":
			opts.All = true
		case "--limit":
			if i+1 >= len(args) {
				return opts, fmt.Errorf("--limit needs a value")
			}
			i++
			limit, err := strconv.Atoi(args[i])
			if err != nil {
				return opts, fmt.Errorf("invalid --limit %q: %w", args[i], err)
			}
			opts.Limit = limit
		case "--name-contains", "--name":
			if i+1 >= len(args) {
				return opts, fmt.Errorf("%s needs a value", args[i])
			}
			i++
			opts.NameContains = strings.ToLower(args[i])
		default:
			return opts, fmt.Errorf("unknown kit dat techs option %q", args[i])
		}
	}
	if opts.Limit < 1 {
		return opts, fmt.Errorf("--limit must be positive")
	}
	return opts, nil
}

func parseDatAbilityListOptions(args []string, label string) (datAbilityListOptions, error) {
	opts := datAbilityListOptions{Limit: 200}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--all":
			opts.All = true
		case "--limit":
			if i+1 >= len(args) {
				return opts, fmt.Errorf("--limit needs a value")
			}
			i++
			limit, err := strconv.Atoi(args[i])
			if err != nil {
				return opts, fmt.Errorf("invalid --limit %q: %w", args[i], err)
			}
			opts.Limit = limit
		case "--name-contains", "--name":
			if i+1 >= len(args) {
				return opts, fmt.Errorf("%s needs a value", args[i])
			}
			i++
			opts.NameContains = strings.ToLower(args[i])
		case "--effect-id":
			if i+1 >= len(args) {
				return opts, fmt.Errorf("--effect-id needs a value")
			}
			i++
			value, err := strconv.Atoi(args[i])
			if err != nil {
				return opts, fmt.Errorf("invalid --effect-id %q: %w", args[i], err)
			}
			opts.EffectID = &value
		case "--command-type":
			if i+1 >= len(args) {
				return opts, fmt.Errorf("--command-type needs a value")
			}
			i++
			value, err := strconv.Atoi(args[i])
			if err != nil {
				return opts, fmt.Errorf("invalid --command-type %q: %w", args[i], err)
			}
			opts.CommandType = &value
		case "--operand":
			if i+1 >= len(args) {
				return opts, fmt.Errorf("--operand needs a value")
			}
			i++
			value, err := strconv.Atoi(args[i])
			if err != nil {
				return opts, fmt.Errorf("invalid --operand %q: %w", args[i], err)
			}
			opts.Operand = &value
		default:
			return opts, fmt.Errorf("unknown %s option %q", label, args[i])
		}
	}
	if opts.Limit < 1 {
		return opts, fmt.Errorf("--limit must be positive")
	}
	return opts, nil
}

func parseDatEffectCreateOptions(args []string) (datcodec.EffectCreateRecipe, error) {
	var recipe datcodec.EffectCreateRecipe
	nameSet := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--name":
			value, err := parseNextString(args, &i, "--name")
			if err != nil {
				return recipe, err
			}
			recipe.Name = value
			nameSet = true
		case "--from-effect":
			value, err := parseNextInt(args, &i, "--from-effect")
			if err != nil {
				return recipe, err
			}
			recipe.FromEffect = &value
		case "--command", "--set-command":
			command, err := parseNextEffectCommandRecipe(args, &i, args[i])
			if err != nil {
				return recipe, err
			}
			recipe.Commands = append(recipe.Commands, command)
		case "--append-command":
			command, err := parseNextEffectCommandRecipe(args, &i, "--append-command")
			if err != nil {
				return recipe, err
			}
			recipe.AppendCommands = append(recipe.AppendCommands, command)
		case "--remove-command":
			value, err := parseNextInt(args, &i, "--remove-command")
			if err != nil {
				return recipe, err
			}
			recipe.RemoveCommands = append(recipe.RemoveCommands, value)
		default:
			return recipe, fmt.Errorf("unknown kit dat effect-create option %q", args[i])
		}
	}
	if !nameSet || strings.TrimSpace(recipe.Name) == "" {
		return recipe, errors.New("--name is required")
	}
	return recipe, nil
}

func parseDatEffectPatchOptions(id int, args []string) (datcodec.EffectPatchRecipe, error) {
	if id < 0 {
		return datcodec.EffectPatchRecipe{}, fmt.Errorf("negative effect id %d", id)
	}
	recipe := datcodec.EffectPatchRecipe{ID: id}
	changed := false
	var setCommands []datcodec.EffectCommandRecipe
	setCommandsSet := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--name":
			value, err := parseNextString(args, &i, "--name")
			if err != nil {
				return recipe, err
			}
			recipe.Name = &value
			changed = true
		case "--clear-commands":
			setCommands = []datcodec.EffectCommandRecipe{}
			setCommandsSet = true
			changed = true
		case "--command", "--set-command":
			command, err := parseNextEffectCommandRecipe(args, &i, args[i])
			if err != nil {
				return recipe, err
			}
			setCommands = append(setCommands, command)
			setCommandsSet = true
			changed = true
		case "--append-command":
			command, err := parseNextEffectCommandRecipe(args, &i, "--append-command")
			if err != nil {
				return recipe, err
			}
			recipe.AppendCommands = append(recipe.AppendCommands, command)
			changed = true
		case "--remove-command":
			value, err := parseNextInt(args, &i, "--remove-command")
			if err != nil {
				return recipe, err
			}
			recipe.RemoveCommands = append(recipe.RemoveCommands, value)
			changed = true
		default:
			return recipe, fmt.Errorf("unknown kit dat effect-patch option %q", args[i])
		}
	}
	if setCommandsSet {
		recipe.Commands = &setCommands
	}
	if !changed {
		return recipe, errors.New("effect-patch has no operations")
	}
	return recipe, nil
}

func parseTextJSONOptions(args []string, label string) (bool, error) {
	text := false
	for _, arg := range args {
		switch arg {
		case "--text":
			text = true
		case "--json":
			text = false
		default:
			return false, fmt.Errorf("unknown %s option %q", label, arg)
		}
	}
	return text, nil
}

func parseDatAbilityCreateOptions(args []string) (datcodec.AbilityCreateRecipe, error) {
	var recipe datcodec.AbilityCreateRecipe
	fromTechSet := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--from-tech":
			value, err := parseNextInt(args, &i, "--from-tech")
			if err != nil {
				return recipe, err
			}
			recipe.FromTech = value
			fromTechSet = true
		case "--name":
			value, err := parseNextString(args, &i, "--name")
			if err != nil {
				return recipe, err
			}
			recipe.Name = value
		case "--effect-name":
			value, err := parseNextString(args, &i, "--effect-name")
			if err != nil {
				return recipe, err
			}
			recipe.EffectName = value
		case "--from-effect":
			value, err := parseNextInt(args, &i, "--from-effect")
			if err != nil {
				return recipe, err
			}
			recipe.FromEffect = &value
		case "--command":
			command, err := parseNextEffectCommandRecipe(args, &i, "--command")
			if err != nil {
				return recipe, err
			}
			recipe.Commands = append(recipe.Commands, command)
		case "--append-command":
			command, err := parseNextEffectCommandRecipe(args, &i, "--append-command")
			if err != nil {
				return recipe, err
			}
			recipe.AppendCommands = append(recipe.AppendCommands, command)
		case "--required-techs":
			value, err := parseNextInt16List(args, &i, "--required-techs", 6)
			if err != nil {
				return recipe, err
			}
			recipe.RequiredTechs = value
		case "--required-tech-count":
			value, err := parseNextInt16(args, &i, "--required-tech-count")
			if err != nil {
				return recipe, err
			}
			recipe.RequiredTechCount = &value
		case "--civ":
			value, err := parseNextInt16(args, &i, "--civ")
			if err != nil {
				return recipe, err
			}
			recipe.Civ = &value
		case "--full-tech-mode":
			value, err := parseNextInt16(args, &i, "--full-tech-mode")
			if err != nil {
				return recipe, err
			}
			recipe.FullTechMode = &value
		case "--dll-name":
			value, err := parseNextInt32(args, &i, "--dll-name")
			if err != nil {
				return recipe, err
			}
			recipe.LanguageDLLName = &value
		case "--dll-description":
			value, err := parseNextInt32(args, &i, "--dll-description")
			if err != nil {
				return recipe, err
			}
			recipe.LanguageDLLDescription = &value
		case "--type":
			value, err := parseNextInt16(args, &i, "--type")
			if err != nil {
				return recipe, err
			}
			recipe.Type = &value
		case "--icon-id":
			value, err := parseNextInt16(args, &i, "--icon-id")
			if err != nil {
				return recipe, err
			}
			recipe.IconID = &value
		case "--dll-help":
			value, err := parseNextInt32(args, &i, "--dll-help")
			if err != nil {
				return recipe, err
			}
			recipe.LanguageDLLHelp = &value
		case "--dll-tech-tree":
			value, err := parseNextInt32(args, &i, "--dll-tech-tree")
			if err != nil {
				return recipe, err
			}
			recipe.LanguageDLLTechTree = &value
		case "--repeatable":
			value, err := parseNextUint8(args, &i, "--repeatable")
			if err != nil {
				return recipe, err
			}
			recipe.Repeatable = &value
		default:
			return recipe, fmt.Errorf("unknown kit dat ability-create option %q", args[i])
		}
	}
	if !fromTechSet {
		return recipe, errors.New("--from-tech is required")
	}
	if strings.TrimSpace(recipe.Name) == "" {
		return recipe, errors.New("--name is required")
	}
	return recipe, nil
}

func parseDatTechCreateOptions(args []string) (datcodec.TechCreateRecipe, error) {
	var recipe datcodec.TechCreateRecipe
	fromSet := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--from":
			value, err := parseNextInt(args, &i, "--from")
			if err != nil {
				return recipe, err
			}
			recipe.From = value
			fromSet = true
		default:
			if err := parseDatTechCreateField(&recipe, args, &i); err != nil {
				return recipe, err
			}
		}
	}
	if !fromSet {
		return recipe, errors.New("--from is required")
	}
	return recipe, nil
}

func parseDatTechPatchOptions(id int, args []string) (datcodec.TechPatchRecipe, error) {
	if id < 0 {
		return datcodec.TechPatchRecipe{}, fmt.Errorf("negative tech id %d", id)
	}
	recipe := datcodec.TechPatchRecipe{ID: id}
	changed := false
	for i := 0; i < len(args); i++ {
		if err := parseDatTechPatchField(&recipe, args, &i); err != nil {
			return recipe, err
		}
		changed = true
	}
	if !changed {
		return recipe, errors.New("tech-patch has no operations")
	}
	return recipe, nil
}

func parseDatTechCreateField(recipe *datcodec.TechCreateRecipe, args []string, i *int) error {
	switch args[*i] {
	case "--name":
		value, err := parseNextString(args, i, "--name")
		if err != nil {
			return err
		}
		recipe.Name = &value
	case "--required-techs":
		value, err := parseNextInt16List(args, i, "--required-techs", 6)
		if err != nil {
			return err
		}
		recipe.RequiredTechs = value
	case "--required-tech-count":
		value, err := parseNextInt16(args, i, "--required-tech-count")
		if err != nil {
			return err
		}
		recipe.RequiredTechCount = &value
	case "--civ":
		value, err := parseNextInt16(args, i, "--civ")
		if err != nil {
			return err
		}
		recipe.Civ = &value
	case "--full-tech-mode":
		value, err := parseNextInt16(args, i, "--full-tech-mode")
		if err != nil {
			return err
		}
		recipe.FullTechMode = &value
	case "--dll-name":
		value, err := parseNextInt32(args, i, "--dll-name")
		if err != nil {
			return err
		}
		recipe.LanguageDLLName = &value
	case "--dll-description":
		value, err := parseNextInt32(args, i, "--dll-description")
		if err != nil {
			return err
		}
		recipe.LanguageDLLDescription = &value
	case "--effect-id":
		value, err := parseNextInt16(args, i, "--effect-id")
		if err != nil {
			return err
		}
		recipe.EffectID = &value
	case "--type":
		value, err := parseNextInt16(args, i, "--type")
		if err != nil {
			return err
		}
		recipe.Type = &value
	case "--icon-id":
		value, err := parseNextInt16(args, i, "--icon-id")
		if err != nil {
			return err
		}
		recipe.IconID = &value
	case "--dll-help":
		value, err := parseNextInt32(args, i, "--dll-help")
		if err != nil {
			return err
		}
		recipe.LanguageDLLHelp = &value
	case "--dll-tech-tree":
		value, err := parseNextInt32(args, i, "--dll-tech-tree")
		if err != nil {
			return err
		}
		recipe.LanguageDLLTechTree = &value
	case "--repeatable":
		value, err := parseNextUint8(args, i, "--repeatable")
		if err != nil {
			return err
		}
		recipe.Repeatable = &value
	default:
		return fmt.Errorf("unknown kit dat tech-create option %q", args[*i])
	}
	return nil
}

func parseDatTechPatchField(recipe *datcodec.TechPatchRecipe, args []string, i *int) error {
	switch args[*i] {
	case "--name":
		value, err := parseNextString(args, i, "--name")
		if err != nil {
			return err
		}
		recipe.Name = &value
	case "--required-techs":
		value, err := parseNextInt16List(args, i, "--required-techs", 6)
		if err != nil {
			return err
		}
		recipe.RequiredTechs = value
	case "--required-tech-count":
		value, err := parseNextInt16(args, i, "--required-tech-count")
		if err != nil {
			return err
		}
		recipe.RequiredTechCount = &value
	case "--civ":
		value, err := parseNextInt16(args, i, "--civ")
		if err != nil {
			return err
		}
		recipe.Civ = &value
	case "--full-tech-mode":
		value, err := parseNextInt16(args, i, "--full-tech-mode")
		if err != nil {
			return err
		}
		recipe.FullTechMode = &value
	case "--dll-name":
		value, err := parseNextInt32(args, i, "--dll-name")
		if err != nil {
			return err
		}
		recipe.LanguageDLLName = &value
	case "--dll-description":
		value, err := parseNextInt32(args, i, "--dll-description")
		if err != nil {
			return err
		}
		recipe.LanguageDLLDescription = &value
	case "--effect-id":
		value, err := parseNextInt16(args, i, "--effect-id")
		if err != nil {
			return err
		}
		recipe.EffectID = &value
	case "--type":
		value, err := parseNextInt16(args, i, "--type")
		if err != nil {
			return err
		}
		recipe.Type = &value
	case "--icon-id":
		value, err := parseNextInt16(args, i, "--icon-id")
		if err != nil {
			return err
		}
		recipe.IconID = &value
	case "--dll-help":
		value, err := parseNextInt32(args, i, "--dll-help")
		if err != nil {
			return err
		}
		recipe.LanguageDLLHelp = &value
	case "--dll-tech-tree":
		value, err := parseNextInt32(args, i, "--dll-tech-tree")
		if err != nil {
			return err
		}
		recipe.LanguageDLLTechTree = &value
	case "--repeatable":
		value, err := parseNextUint8(args, i, "--repeatable")
		if err != nil {
			return err
		}
		recipe.Repeatable = &value
	default:
		return fmt.Errorf("unknown kit dat tech-patch option %q", args[*i])
	}
	return nil
}

func parseDatAbilityPatchOptions(inputPath string, techID int, args []string) (datcodec.Recipe, error) {
	if techID < 0 {
		return datcodec.Recipe{}, fmt.Errorf("negative tech id %d", techID)
	}
	idx, err := datfile.Open(inputPath)
	if err != nil {
		return datcodec.Recipe{}, err
	}
	if techID >= len(idx.Techs) {
		return datcodec.Recipe{}, fmt.Errorf("tech %d is outside tech table", techID)
	}
	effectID := int(idx.Techs[techID].EffectID)
	if effectID < 0 || effectID >= len(idx.Effects) {
		return datcodec.Recipe{}, fmt.Errorf("tech %d does not reference a valid effect_id: %d", techID, effectID)
	}
	options := datAbilityPatchOptions{
		Recipe: datcodec.Recipe{
			Techs:   []datcodec.TechPatchRecipe{{ID: techID}},
			Effects: []datcodec.EffectPatchRecipe{{ID: effectID}},
		},
	}
	tech := &options.Recipe.Techs[0]
	effect := &options.Recipe.Effects[0]
	var setCommands []datcodec.EffectCommandRecipe
	setCommandsSet := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--name":
			value, err := parseNextString(args, &i, "--name")
			if err != nil {
				return datcodec.Recipe{}, err
			}
			tech.Name = &value
			options.HasTech = true
		case "--effect-name":
			value, err := parseNextString(args, &i, "--effect-name")
			if err != nil {
				return datcodec.Recipe{}, err
			}
			effect.Name = &value
			options.HasEffect = true
		case "--clear-commands":
			setCommands = []datcodec.EffectCommandRecipe{}
			setCommandsSet = true
			options.HasEffect = true
		case "--command", "--set-command":
			command, err := parseNextEffectCommandRecipe(args, &i, args[i])
			if err != nil {
				return datcodec.Recipe{}, err
			}
			setCommands = append(setCommands, command)
			setCommandsSet = true
			options.HasEffect = true
		case "--append-command":
			command, err := parseNextEffectCommandRecipe(args, &i, "--append-command")
			if err != nil {
				return datcodec.Recipe{}, err
			}
			effect.AppendCommands = append(effect.AppendCommands, command)
			options.HasEffect = true
		case "--remove-command":
			value, err := parseNextInt(args, &i, "--remove-command")
			if err != nil {
				return datcodec.Recipe{}, err
			}
			effect.RemoveCommands = append(effect.RemoveCommands, value)
			options.HasEffect = true
		case "--required-techs":
			value, err := parseNextInt16List(args, &i, "--required-techs", 6)
			if err != nil {
				return datcodec.Recipe{}, err
			}
			tech.RequiredTechs = value
			options.HasTech = true
		case "--required-tech-count":
			value, err := parseNextInt16(args, &i, "--required-tech-count")
			if err != nil {
				return datcodec.Recipe{}, err
			}
			tech.RequiredTechCount = &value
			options.HasTech = true
		case "--civ":
			value, err := parseNextInt16(args, &i, "--civ")
			if err != nil {
				return datcodec.Recipe{}, err
			}
			tech.Civ = &value
			options.HasTech = true
		case "--full-tech-mode":
			value, err := parseNextInt16(args, &i, "--full-tech-mode")
			if err != nil {
				return datcodec.Recipe{}, err
			}
			tech.FullTechMode = &value
			options.HasTech = true
		case "--dll-name":
			value, err := parseNextInt32(args, &i, "--dll-name")
			if err != nil {
				return datcodec.Recipe{}, err
			}
			tech.LanguageDLLName = &value
			options.HasTech = true
		case "--dll-description":
			value, err := parseNextInt32(args, &i, "--dll-description")
			if err != nil {
				return datcodec.Recipe{}, err
			}
			tech.LanguageDLLDescription = &value
			options.HasTech = true
		case "--type":
			value, err := parseNextInt16(args, &i, "--type")
			if err != nil {
				return datcodec.Recipe{}, err
			}
			tech.Type = &value
			options.HasTech = true
		case "--icon-id":
			value, err := parseNextInt16(args, &i, "--icon-id")
			if err != nil {
				return datcodec.Recipe{}, err
			}
			tech.IconID = &value
			options.HasTech = true
		case "--dll-help":
			value, err := parseNextInt32(args, &i, "--dll-help")
			if err != nil {
				return datcodec.Recipe{}, err
			}
			tech.LanguageDLLHelp = &value
			options.HasTech = true
		case "--dll-tech-tree":
			value, err := parseNextInt32(args, &i, "--dll-tech-tree")
			if err != nil {
				return datcodec.Recipe{}, err
			}
			tech.LanguageDLLTechTree = &value
			options.HasTech = true
		case "--repeatable":
			value, err := parseNextUint8(args, &i, "--repeatable")
			if err != nil {
				return datcodec.Recipe{}, err
			}
			tech.Repeatable = &value
			options.HasTech = true
		default:
			return datcodec.Recipe{}, fmt.Errorf("unknown kit dat ability-patch option %q", args[i])
		}
	}
	if setCommandsSet {
		effect.Commands = &setCommands
	}
	if !options.HasTech {
		options.Recipe.Techs = nil
	}
	if !options.HasEffect {
		options.Recipe.Effects = nil
	}
	if options.Recipe.Empty() {
		return datcodec.Recipe{}, errors.New("ability-patch has no operations")
	}
	return options.Recipe, nil
}

func parseDatSoundCreateOptions(args []string) (datcodec.SoundCreateRecipe, error) {
	var recipe datcodec.SoundCreateRecipe
	fromSet := false
	var items []datcodec.SoundItemRecipe
	itemsSet := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--from":
			value, err := parseNextInt(args, &i, "--from")
			if err != nil {
				return recipe, err
			}
			recipe.From = value
			fromSet = true
		case "--item":
			item, err := parseNextSoundItemRecipe(args, &i, "--item")
			if err != nil {
				return recipe, err
			}
			items = append(items, item)
			itemsSet = true
		default:
			if err := parseDatSoundCreateField(&recipe, args, &i); err != nil {
				return recipe, err
			}
		}
	}
	if !fromSet {
		return recipe, errors.New("--from is required")
	}
	if itemsSet {
		recipe.Items = &items
	}
	return recipe, nil
}

func parseDatSoundPatchOptions(id int, args []string) (datcodec.SoundPatchRecipe, error) {
	if id < 0 {
		return datcodec.SoundPatchRecipe{}, fmt.Errorf("negative sound id %d", id)
	}
	recipe := datcodec.SoundPatchRecipe{ID: id}
	changed := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--item":
			item, err := parseNextSoundItemPatchRecipe(args, &i, "--item")
			if err != nil {
				return recipe, err
			}
			recipe.Items = append(recipe.Items, item)
			changed = true
		case "--remove-item":
			value, err := parseNextInt(args, &i, "--remove-item")
			if err != nil {
				return recipe, err
			}
			recipe.RemoveItems = append(recipe.RemoveItems, value)
			changed = true
		default:
			if err := parseDatSoundPatchField(&recipe, args, &i); err != nil {
				return recipe, err
			}
			changed = true
		}
	}
	if !changed {
		return recipe, errors.New("sound-patch has no operations")
	}
	return recipe, nil
}

func parseDatSoundCreateField(recipe *datcodec.SoundCreateRecipe, args []string, i *int) error {
	switch args[*i] {
	case "--sound-id":
		value, err := parseNextInt16(args, i, "--sound-id")
		if err != nil {
			return err
		}
		recipe.SoundID = &value
	case "--play-delay":
		value, err := parseNextInt16(args, i, "--play-delay")
		if err != nil {
			return err
		}
		recipe.PlayDelay = &value
	case "--cache-time":
		value, err := parseNextInt32(args, i, "--cache-time")
		if err != nil {
			return err
		}
		recipe.CacheTime = &value
	case "--total-probability", "--probability":
		value, err := parseNextInt16(args, i, args[*i])
		if err != nil {
			return err
		}
		recipe.TotalProbability = &value
	default:
		return fmt.Errorf("unknown kit dat sound-create option %q", args[*i])
	}
	return nil
}

func parseDatSoundPatchField(recipe *datcodec.SoundPatchRecipe, args []string, i *int) error {
	switch args[*i] {
	case "--sound-id":
		value, err := parseNextInt16(args, i, "--sound-id")
		if err != nil {
			return err
		}
		recipe.SoundID = &value
	case "--play-delay":
		value, err := parseNextInt16(args, i, "--play-delay")
		if err != nil {
			return err
		}
		recipe.PlayDelay = &value
	case "--cache-time":
		value, err := parseNextInt32(args, i, "--cache-time")
		if err != nil {
			return err
		}
		recipe.CacheTime = &value
	case "--total-probability", "--probability":
		value, err := parseNextInt16(args, i, args[*i])
		if err != nil {
			return err
		}
		recipe.TotalProbability = &value
	default:
		return fmt.Errorf("unknown kit dat sound-patch option %q", args[*i])
	}
	return nil
}

func parseNextSoundItemRecipe(args []string, i *int, name string) (datcodec.SoundItemRecipe, error) {
	if *i+1 >= len(args) {
		return datcodec.SoundItemRecipe{}, fmt.Errorf("%s needs file,resource,probability,civ,icon_set", name)
	}
	*i = *i + 1
	parts := strings.Split(args[*i], ",")
	if len(parts) != 5 {
		return datcodec.SoundItemRecipe{}, fmt.Errorf("%s %q needs file,resource,probability,civ,icon_set", name, args[*i])
	}
	resourceID, err := parseInt32(parts[1], name+".resource_id")
	if err != nil {
		return datcodec.SoundItemRecipe{}, err
	}
	probability, err := parseInt16(parts[2], name+".probability")
	if err != nil {
		return datcodec.SoundItemRecipe{}, err
	}
	civ, err := parseInt16(parts[3], name+".civ")
	if err != nil {
		return datcodec.SoundItemRecipe{}, err
	}
	iconSet, err := parseInt16(parts[4], name+".icon_set")
	if err != nil {
		return datcodec.SoundItemRecipe{}, err
	}
	if strings.TrimSpace(parts[0]) == "" {
		return datcodec.SoundItemRecipe{}, fmt.Errorf("%s file name is empty", name)
	}
	return datcodec.SoundItemRecipe{
		FileName:    parts[0],
		ResourceID:  resourceID,
		Probability: probability,
		Civ:         civ,
		IconSet:     iconSet,
	}, nil
}

func parseNextSoundItemPatchRecipe(args []string, i *int, name string) (datcodec.SoundItemPatchRecipe, error) {
	index, fields, err := parseRowPatchSpec(args, i, name)
	if err != nil {
		return datcodec.SoundItemPatchRecipe{}, err
	}
	out := datcodec.SoundItemPatchRecipe{Index: index}
	for key, raw := range fields {
		switch key {
		case "file_name", "file", "name":
			out.FileName = &raw
		case "resource_id", "resource":
			value, err := parseInt32(raw, name+"."+key)
			if err != nil {
				return out, err
			}
			out.ResourceID = &value
		case "probability", "prob":
			value, err := parseInt16(raw, name+"."+key)
			if err != nil {
				return out, err
			}
			out.Probability = &value
		case "civ":
			value, err := parseInt16(raw, name+"."+key)
			if err != nil {
				return out, err
			}
			out.Civ = &value
		case "icon_set", "icon":
			value, err := parseInt16(raw, name+"."+key)
			if err != nil {
				return out, err
			}
			out.IconSet = &value
		default:
			return out, fmt.Errorf("%s unknown field %q", name, key)
		}
	}
	return out, nil
}

func parseDatPlayerColourCreateOptions(args []string) (datcodec.PlayerColourCreateRecipe, error) {
	var recipe datcodec.PlayerColourCreateRecipe
	fromSet := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--from":
			value, err := parseNextInt(args, &i, "--from")
			if err != nil {
				return recipe, err
			}
			recipe.From = value
			fromSet = true
		default:
			if err := parseDatPlayerColourCreateField(&recipe, args, &i); err != nil {
				return recipe, err
			}
		}
	}
	if !fromSet {
		return recipe, errors.New("--from is required")
	}
	return recipe, nil
}

func parseDatPlayerColourPatchOptions(id int, args []string) (datcodec.PlayerColourPatchRecipe, error) {
	if id < 0 {
		return datcodec.PlayerColourPatchRecipe{}, fmt.Errorf("negative player colour id %d", id)
	}
	recipe := datcodec.PlayerColourPatchRecipe{ID: id}
	changed := false
	for i := 0; i < len(args); i++ {
		if err := parseDatPlayerColourPatchField(&recipe, args, &i); err != nil {
			return recipe, err
		}
		changed = true
	}
	if !changed {
		return recipe, errors.New("player-colour-patch has no operations")
	}
	return recipe, nil
}

func parseDatPlayerColourCreateField(recipe *datcodec.PlayerColourCreateRecipe, args []string, i *int) error {
	switch args[*i] {
	case "--colour-id", "--color-id":
		value, err := parseNextInt32(args, i, args[*i])
		if err != nil {
			return err
		}
		recipe.ColourID = &value
	case "--base":
		value, err := parseNextInt32(args, i, "--base")
		if err != nil {
			return err
		}
		recipe.Base = &value
	case "--unit-outline-colour", "--unit-outline-color", "--outline":
		value, err := parseNextInt32(args, i, args[*i])
		if err != nil {
			return err
		}
		recipe.UnitOutlineColour = &value
	case "--selection-colour-1", "--selection-color-1", "--selection-1":
		value, err := parseNextInt32(args, i, args[*i])
		if err != nil {
			return err
		}
		recipe.SelectionColour1 = &value
	case "--selection-colour-2", "--selection-color-2", "--selection-2":
		value, err := parseNextInt32(args, i, args[*i])
		if err != nil {
			return err
		}
		recipe.SelectionColour2 = &value
	case "--minimap-colour-1", "--minimap-color-1", "--minimap-1":
		value, err := parseNextInt32(args, i, args[*i])
		if err != nil {
			return err
		}
		recipe.MinimapColour1 = &value
	case "--minimap-colour-2", "--minimap-color-2", "--minimap-2":
		value, err := parseNextInt32(args, i, args[*i])
		if err != nil {
			return err
		}
		recipe.MinimapColour2 = &value
	case "--minimap-colour-3", "--minimap-color-3", "--minimap-3":
		value, err := parseNextInt32(args, i, args[*i])
		if err != nil {
			return err
		}
		recipe.MinimapColour3 = &value
	case "--statistics-text-colour", "--statistics-text-color", "--stats":
		value, err := parseNextInt32(args, i, args[*i])
		if err != nil {
			return err
		}
		recipe.StatisticsTextColor = &value
	default:
		return fmt.Errorf("unknown kit dat player-colour-create option %q", args[*i])
	}
	return nil
}

func parseDatPlayerColourPatchField(recipe *datcodec.PlayerColourPatchRecipe, args []string, i *int) error {
	switch args[*i] {
	case "--colour-id", "--color-id":
		value, err := parseNextInt32(args, i, args[*i])
		if err != nil {
			return err
		}
		recipe.ColourID = &value
	case "--base":
		value, err := parseNextInt32(args, i, "--base")
		if err != nil {
			return err
		}
		recipe.Base = &value
	case "--unit-outline-colour", "--unit-outline-color", "--outline":
		value, err := parseNextInt32(args, i, args[*i])
		if err != nil {
			return err
		}
		recipe.UnitOutlineColour = &value
	case "--selection-colour-1", "--selection-color-1", "--selection-1":
		value, err := parseNextInt32(args, i, args[*i])
		if err != nil {
			return err
		}
		recipe.SelectionColour1 = &value
	case "--selection-colour-2", "--selection-color-2", "--selection-2":
		value, err := parseNextInt32(args, i, args[*i])
		if err != nil {
			return err
		}
		recipe.SelectionColour2 = &value
	case "--minimap-colour-1", "--minimap-color-1", "--minimap-1":
		value, err := parseNextInt32(args, i, args[*i])
		if err != nil {
			return err
		}
		recipe.MinimapColour1 = &value
	case "--minimap-colour-2", "--minimap-color-2", "--minimap-2":
		value, err := parseNextInt32(args, i, args[*i])
		if err != nil {
			return err
		}
		recipe.MinimapColour2 = &value
	case "--minimap-colour-3", "--minimap-color-3", "--minimap-3":
		value, err := parseNextInt32(args, i, args[*i])
		if err != nil {
			return err
		}
		recipe.MinimapColour3 = &value
	case "--statistics-text-colour", "--statistics-text-color", "--stats":
		value, err := parseNextInt32(args, i, args[*i])
		if err != nil {
			return err
		}
		recipe.StatisticsTextColor = &value
	default:
		return fmt.Errorf("unknown kit dat player-colour-patch option %q", args[*i])
	}
	return nil
}

func parseNextEffectCommandRecipe(args []string, i *int, name string) (datcodec.EffectCommandRecipe, error) {
	if *i+1 >= len(args) {
		return datcodec.EffectCommandRecipe{}, fmt.Errorf("%s needs a value", name)
	}
	*i = *i + 1
	return parseEffectCommandRecipeSpec(args[*i], name)
}

func parseEffectCommandRecipeSpec(raw, name string) (datcodec.EffectCommandRecipe, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return datcodec.EffectCommandRecipe{}, fmt.Errorf("%s needs a non-empty command spec", name)
	}
	if strings.HasPrefix(raw, "{") {
		var command datcodec.EffectCommandRecipe
		if err := json.Unmarshal([]byte(raw), &command); err != nil {
			return datcodec.EffectCommandRecipe{}, fmt.Errorf("%s JSON command: %w", name, err)
		}
		return command, nil
	}
	parts := strings.Split(raw, ",")
	if len(parts) != 5 {
		return datcodec.EffectCommandRecipe{}, fmt.Errorf("%s raw command %q needs type,a,b,c,d or a JSON object", name, raw)
	}
	commandType, err := parseUint8(parts[0], name+".type")
	if err != nil {
		return datcodec.EffectCommandRecipe{}, err
	}
	a, err := parseInt16(parts[1], name+".a")
	if err != nil {
		return datcodec.EffectCommandRecipe{}, err
	}
	b, err := parseInt16(parts[2], name+".b")
	if err != nil {
		return datcodec.EffectCommandRecipe{}, err
	}
	c, err := parseInt16(parts[3], name+".c")
	if err != nil {
		return datcodec.EffectCommandRecipe{}, err
	}
	d, err := parseFloat32(parts[4], name+".d")
	if err != nil {
		return datcodec.EffectCommandRecipe{}, err
	}
	return datcodec.EffectCommandRecipe{Type: commandType, A: a, B: b, C: c, D: d}, nil
}

type datUnitFilters struct {
	Limit        int    `json:"limit"`
	All          bool   `json:"all,omitempty"`
	Civ          *int   `json:"civ,omitempty"`
	ID           *int   `json:"id,omitempty"`
	Name         string `json:"name,omitempty"`
	NameContains string `json:"name_contains,omitempty"`
	Class        string `json:"class,omitempty"`
}

type datUnitsReport struct {
	Version       string            `json:"version"`
	CivCount      int               `json:"civ_count"`
	Filters       datUnitFilters    `json:"filters"`
	PresentCount  int               `json:"present_count"`
	ReturnedCount int               `json:"returned_count"`
	Truncated     bool              `json:"truncated"`
	ClassNote     string            `json:"class_note,omitempty"`
	Units         []unitListSummary `json:"units"`
}

func (f datUnitFilters) classNote() string {
	switch f.Class {
	case "building":
		return "building filter is derived from DAT unit type byte 80/90"
	case "unit":
		return "unit filter excludes DAT unit type byte 80/90"
	case "creatable":
		return "creatable filter means the parsed unit record has a creatable/train-location block"
	default:
		return ""
	}
}

func parseDatUnitsOptions(args []string) (datUnitFilters, error) {
	filters := datUnitFilters{Limit: 200}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
		case "--all":
			filters.All = true
		case "--limit":
			if i+1 >= len(args) {
				return filters, fmt.Errorf("--limit needs a value")
			}
			i++
			limit, err := strconv.Atoi(args[i])
			if err != nil {
				return filters, fmt.Errorf("invalid --limit %q: %w", args[i], err)
			}
			filters.Limit = limit
		case "--civ":
			if i+1 >= len(args) {
				return filters, fmt.Errorf("--civ needs a value")
			}
			i++
			civ, err := strconv.Atoi(args[i])
			if err != nil {
				return filters, fmt.Errorf("invalid --civ %q: %w", args[i], err)
			}
			filters.Civ = &civ
		case "--id":
			if i+1 >= len(args) {
				return filters, fmt.Errorf("--id needs a value")
			}
			i++
			id, err := strconv.Atoi(args[i])
			if err != nil {
				return filters, fmt.Errorf("invalid --id %q: %w", args[i], err)
			}
			filters.ID = &id
		case "--name":
			if i+1 >= len(args) {
				return filters, fmt.Errorf("--name needs a value")
			}
			i++
			filters.Name = strings.ToLower(args[i])
		case "--name-contains":
			if i+1 >= len(args) {
				return filters, fmt.Errorf("--name-contains needs a value")
			}
			i++
			filters.NameContains = strings.ToLower(args[i])
		case "--class":
			if i+1 >= len(args) {
				return filters, fmt.Errorf("--class needs a value")
			}
			i++
			class := strings.ToLower(args[i])
			switch class {
			case "building", "unit", "creatable":
				filters.Class = class
			default:
				return filters, fmt.Errorf("--class must be building, unit, or creatable")
			}
		default:
			return filters, fmt.Errorf("unknown dat units option %q", args[i])
		}
	}
	if filters.Limit < 1 {
		return filters, fmt.Errorf("--limit must be positive")
	}
	return filters, nil
}

func cachedDatUnit(datPath string, civID, unitID int) (datfile.UnitSummary, bool, error) {
	cache, err := datcache.Open(datPath)
	if err != nil {
		return datfile.UnitSummary{}, false, err
	}
	defer cache.Close()
	return cache.Unit(civID, unitID)
}

func cachedDatUnitsReport(datPath string, filters datUnitFilters) (datUnitsReport, bool, error) {
	cache, err := datcache.Open(datPath)
	if err != nil {
		return datUnitsReport{}, false, err
	}
	defer cache.Close()
	rows, total, err := cache.QueryUnits(datcache.UnitQuery{
		Limit:        filters.Limit,
		All:          filters.All,
		Civ:          filters.Civ,
		ID:           filters.ID,
		Name:         filters.Name,
		NameContains: filters.NameContains,
		Class:        filters.Class,
	})
	if err != nil {
		return datUnitsReport{}, false, err
	}
	units := compactCachedUnits(rows)
	return datUnitsReport{
		Version:       cache.Version,
		CivCount:      cache.CivCount,
		Filters:       filters,
		PresentCount:  total,
		ReturnedCount: len(units),
		Truncated:     !filters.All && total > len(units),
		ClassNote:     filters.classNote(),
		Units:         units,
	}, true, nil
}

func compactCachedUnits(rows []datcache.UnitRow) []unitListSummary {
	out := make([]unitListSummary, 0, len(rows))
	for _, row := range rows {
		out = append(out, unitListSummary{
			CivIndex:           row.CivIndex,
			Index:              row.Index,
			Type:               row.Type,
			ID:                 row.ID,
			Name:               row.Name,
			Class:              row.Class,
			ClassName:          row.ClassName,
			HitPoints:          row.HitPoints,
			CollisionSizeX:     row.CollisionSizeX,
			CollisionSizeY:     row.CollisionSizeY,
			CollisionSizeZ:     row.CollisionSizeZ,
			ClearanceSizeX:     row.ClearanceSizeX,
			ClearanceSizeY:     row.ClearanceSizeY,
			TerrainTableID:     row.TerrainTableID,
			ObstructionTypeID:  row.ObstructionTypeID,
			ObstructionClassID: row.ObstructionClassID,
			IconID:             row.IconID,
			Enabled:            row.Enabled,
			HasType50:          row.HasType50,
			HasCreatable:       row.HasCreatable,
			RecordStart:        row.RecordStart,
			RecordEnd:          row.RecordEnd,
		})
	}
	return out
}

func parseDatAvailabilityRequest(unitID int, args []string) (datcodec.UnitAvailabilityRequest, error) {
	request := datcodec.UnitAvailabilityRequest{UnitID: unitID}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--all-civs":
			request.AllCivs = true
		case "--civ":
			if i+1 >= len(args) {
				return request, fmt.Errorf("--civ needs a value")
			}
			i++
			civID, err := strconv.Atoi(args[i])
			if err != nil {
				return request, fmt.Errorf("invalid --civ %q: %w", args[i], err)
			}
			request.CivIDs = append(request.CivIDs, civID)
		default:
			return request, fmt.Errorf("unknown dat availability option %q", args[i])
		}
	}
	if request.AllCivs && len(request.CivIDs) > 0 {
		return request, errors.New("set either --all-civs or --civ, not both")
	}
	if !request.AllCivs && len(request.CivIDs) == 1 {
		request.CivID = &request.CivIDs[0]
		request.CivIDs = nil
	}
	return request, nil
}

func parseDatTechTreeOptions(args []string) (bool, error) {
	full := false
	for _, arg := range args {
		switch arg {
		case "--full":
			full = true
		default:
			return false, fmt.Errorf("unknown kit dat tech-tree option %q", arg)
		}
	}
	return full, nil
}

func normalizeTechTreeConnectionFamily(raw string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "building", "buildings", "building_connection", "tech_tree_building_connection":
		return "building", nil
	case "unit", "units", "unit_connection", "tech_tree_unit_connection":
		return "unit", nil
	case "research", "researches", "research_connection", "tech_tree_research_connection":
		return "research", nil
	default:
		return "", fmt.Errorf("unknown tech-tree connection family %q; want building, unit, or research", raw)
	}
}

func parseDatTechTreeConnectionCreateOptions(family string, from int, args []string) (datcodec.TechTreePatchRecipe, any, error) {
	if from < 0 {
		return datcodec.TechTreePatchRecipe{}, nil, fmt.Errorf("negative source index %d", from)
	}
	switch family {
	case "building":
		patch, _, err := parseDatTechTreeBuildingConnectionPatchFlags(args)
		if err != nil {
			return datcodec.TechTreePatchRecipe{}, nil, err
		}
		create := datcodec.TechTreeBuildingConnectionCreate{
			From:             from,
			ID:               patch.ID,
			Status:           patch.Status,
			Buildings:        patch.Buildings,
			Units:            patch.Units,
			Techs:            patch.Techs,
			UnitResearch:     patch.UnitResearch,
			Mode:             patch.Mode,
			LocationInAge:    patch.LocationInAge,
			UnitsTechsTotal:  patch.UnitsTechsTotal,
			UnitsTechsFirst:  patch.UnitsTechsFirst,
			LineMode:         patch.LineMode,
			EnablingResearch: patch.EnablingResearch,
		}
		return datcodec.TechTreePatchRecipe{CreateBuildingConnections: []datcodec.TechTreeBuildingConnectionCreate{create}}, create, nil
	case "unit":
		patch, _, err := parseDatTechTreeUnitConnectionPatchFlags(args)
		if err != nil {
			return datcodec.TechTreePatchRecipe{}, nil, err
		}
		create := datcodec.TechTreeUnitConnectionCreate{
			From:             from,
			ID:               patch.ID,
			Status:           patch.Status,
			UpperBuilding:    patch.UpperBuilding,
			UnitResearch:     patch.UnitResearch,
			Mode:             patch.Mode,
			VerticalLine:     patch.VerticalLine,
			Units:            patch.Units,
			LocationInAge:    patch.LocationInAge,
			RequiredResearch: patch.RequiredResearch,
			LineMode:         patch.LineMode,
			EnablingResearch: patch.EnablingResearch,
		}
		return datcodec.TechTreePatchRecipe{CreateUnitConnections: []datcodec.TechTreeUnitConnectionCreate{create}}, create, nil
	case "research":
		patch, _, err := parseDatTechTreeResearchConnectionPatchFlags(args)
		if err != nil {
			return datcodec.TechTreePatchRecipe{}, nil, err
		}
		create := datcodec.TechTreeResearchConnectionCreate{
			From:          from,
			ID:            patch.ID,
			Status:        patch.Status,
			UpperBuilding: patch.UpperBuilding,
			Buildings:     patch.Buildings,
			Units:         patch.Units,
			Techs:         patch.Techs,
			UnitResearch:  patch.UnitResearch,
			Mode:          patch.Mode,
			VerticalLine:  patch.VerticalLine,
			LocationInAge: patch.LocationInAge,
			LineMode:      patch.LineMode,
		}
		return datcodec.TechTreePatchRecipe{CreateResearchConnections: []datcodec.TechTreeResearchConnectionCreate{create}}, create, nil
	default:
		return datcodec.TechTreePatchRecipe{}, nil, fmt.Errorf("unknown tech-tree connection family %q", family)
	}
}

func parseDatTechTreeConnectionPatchOptions(family string, index int, args []string) (datcodec.TechTreePatchRecipe, any, error) {
	if index < 0 {
		return datcodec.TechTreePatchRecipe{}, nil, fmt.Errorf("negative connection index %d", index)
	}
	switch family {
	case "building":
		patch, changed, err := parseDatTechTreeBuildingConnectionPatchFlags(args)
		if err != nil {
			return datcodec.TechTreePatchRecipe{}, nil, err
		}
		if !changed {
			return datcodec.TechTreePatchRecipe{}, nil, errors.New("tech-tree building connection patch has no operations")
		}
		patch.Index = index
		return datcodec.TechTreePatchRecipe{BuildingConnections: []datcodec.TechTreeBuildingConnectionPatch{patch}}, patch, nil
	case "unit":
		patch, changed, err := parseDatTechTreeUnitConnectionPatchFlags(args)
		if err != nil {
			return datcodec.TechTreePatchRecipe{}, nil, err
		}
		if !changed {
			return datcodec.TechTreePatchRecipe{}, nil, errors.New("tech-tree unit connection patch has no operations")
		}
		patch.Index = index
		return datcodec.TechTreePatchRecipe{UnitConnections: []datcodec.TechTreeUnitConnectionPatch{patch}}, patch, nil
	case "research":
		patch, changed, err := parseDatTechTreeResearchConnectionPatchFlags(args)
		if err != nil {
			return datcodec.TechTreePatchRecipe{}, nil, err
		}
		if !changed {
			return datcodec.TechTreePatchRecipe{}, nil, errors.New("tech-tree research connection patch has no operations")
		}
		patch.Index = index
		return datcodec.TechTreePatchRecipe{ResearchConnections: []datcodec.TechTreeResearchConnectionPatch{patch}}, patch, nil
	default:
		return datcodec.TechTreePatchRecipe{}, nil, fmt.Errorf("unknown tech-tree connection family %q", family)
	}
}

func datTechTreeConnectionDeleteRecipe(family string, index int) (datcodec.TechTreePatchRecipe, error) {
	if index < 0 {
		return datcodec.TechTreePatchRecipe{}, fmt.Errorf("negative connection index %d", index)
	}
	switch family {
	case "building":
		return datcodec.TechTreePatchRecipe{DeleteBuildingConnections: []int{index}}, nil
	case "unit":
		return datcodec.TechTreePatchRecipe{DeleteUnitConnections: []int{index}}, nil
	case "research":
		return datcodec.TechTreePatchRecipe{DeleteResearchConnections: []int{index}}, nil
	default:
		return datcodec.TechTreePatchRecipe{}, fmt.Errorf("unknown tech-tree connection family %q", family)
	}
}

func parseDatTechTreeBuildingConnectionPatchFlags(args []string) (datcodec.TechTreeBuildingConnectionPatch, bool, error) {
	var patch datcodec.TechTreeBuildingConnectionPatch
	changed := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--id":
			value, err := parseNextInt32(args, &i, "--id")
			if err != nil {
				return patch, false, err
			}
			patch.ID = &value
		case "--status":
			value, err := parseNextUint8(args, &i, "--status")
			if err != nil {
				return patch, false, err
			}
			patch.Status = &value
		case "--buildings":
			value, err := parseNextInt32List(args, &i, "--buildings")
			if err != nil {
				return patch, false, err
			}
			patch.Buildings = &value
		case "--units":
			value, err := parseNextInt32List(args, &i, "--units")
			if err != nil {
				return patch, false, err
			}
			patch.Units = &value
		case "--techs":
			value, err := parseNextInt32List(args, &i, "--techs")
			if err != nil {
				return patch, false, err
			}
			patch.Techs = &value
		case "--unit-research":
			value, err := parseNextInt32List(args, &i, "--unit-research")
			if err != nil {
				return patch, false, err
			}
			patch.UnitResearch = &value
		case "--mode":
			value, err := parseNextInt32List(args, &i, "--mode")
			if err != nil {
				return patch, false, err
			}
			patch.Mode = &value
		case "--location-in-age":
			value, err := parseNextUint8(args, &i, "--location-in-age")
			if err != nil {
				return patch, false, err
			}
			patch.LocationInAge = &value
		case "--units-techs-total":
			value, err := parseNextUint8List(args, &i, "--units-techs-total")
			if err != nil {
				return patch, false, err
			}
			patch.UnitsTechsTotal = &value
		case "--units-techs-first":
			value, err := parseNextUint8List(args, &i, "--units-techs-first")
			if err != nil {
				return patch, false, err
			}
			patch.UnitsTechsFirst = &value
		case "--line-mode":
			value, err := parseNextInt32(args, &i, "--line-mode")
			if err != nil {
				return patch, false, err
			}
			patch.LineMode = &value
		case "--enabling-research":
			value, err := parseNextInt32(args, &i, "--enabling-research")
			if err != nil {
				return patch, false, err
			}
			patch.EnablingResearch = &value
		default:
			return patch, false, fmt.Errorf("unknown tech-tree building connection option %q", args[i])
		}
		changed = true
	}
	return patch, changed, nil
}

func parseDatTechTreeUnitConnectionPatchFlags(args []string) (datcodec.TechTreeUnitConnectionPatch, bool, error) {
	var patch datcodec.TechTreeUnitConnectionPatch
	changed := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--id":
			value, err := parseNextInt32(args, &i, "--id")
			if err != nil {
				return patch, false, err
			}
			patch.ID = &value
		case "--status":
			value, err := parseNextUint8(args, &i, "--status")
			if err != nil {
				return patch, false, err
			}
			patch.Status = &value
		case "--upper-building":
			value, err := parseNextInt32(args, &i, "--upper-building")
			if err != nil {
				return patch, false, err
			}
			patch.UpperBuilding = &value
		case "--unit-research":
			value, err := parseNextInt32List(args, &i, "--unit-research")
			if err != nil {
				return patch, false, err
			}
			patch.UnitResearch = &value
		case "--mode":
			value, err := parseNextInt32List(args, &i, "--mode")
			if err != nil {
				return patch, false, err
			}
			patch.Mode = &value
		case "--vertical-line":
			value, err := parseNextInt32(args, &i, "--vertical-line")
			if err != nil {
				return patch, false, err
			}
			patch.VerticalLine = &value
		case "--units":
			value, err := parseNextInt32List(args, &i, "--units")
			if err != nil {
				return patch, false, err
			}
			patch.Units = &value
		case "--location-in-age":
			value, err := parseNextInt32(args, &i, "--location-in-age")
			if err != nil {
				return patch, false, err
			}
			patch.LocationInAge = &value
		case "--required-research":
			value, err := parseNextInt32(args, &i, "--required-research")
			if err != nil {
				return patch, false, err
			}
			patch.RequiredResearch = &value
		case "--line-mode":
			value, err := parseNextInt32(args, &i, "--line-mode")
			if err != nil {
				return patch, false, err
			}
			patch.LineMode = &value
		case "--enabling-research":
			value, err := parseNextInt32(args, &i, "--enabling-research")
			if err != nil {
				return patch, false, err
			}
			patch.EnablingResearch = &value
		default:
			return patch, false, fmt.Errorf("unknown tech-tree unit connection option %q", args[i])
		}
		changed = true
	}
	return patch, changed, nil
}

func parseDatTechTreeResearchConnectionPatchFlags(args []string) (datcodec.TechTreeResearchConnectionPatch, bool, error) {
	var patch datcodec.TechTreeResearchConnectionPatch
	changed := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--id":
			value, err := parseNextInt32(args, &i, "--id")
			if err != nil {
				return patch, false, err
			}
			patch.ID = &value
		case "--status":
			value, err := parseNextUint8(args, &i, "--status")
			if err != nil {
				return patch, false, err
			}
			patch.Status = &value
		case "--upper-building":
			value, err := parseNextInt32(args, &i, "--upper-building")
			if err != nil {
				return patch, false, err
			}
			patch.UpperBuilding = &value
		case "--buildings":
			value, err := parseNextInt32List(args, &i, "--buildings")
			if err != nil {
				return patch, false, err
			}
			patch.Buildings = &value
		case "--units":
			value, err := parseNextInt32List(args, &i, "--units")
			if err != nil {
				return patch, false, err
			}
			patch.Units = &value
		case "--techs":
			value, err := parseNextInt32List(args, &i, "--techs")
			if err != nil {
				return patch, false, err
			}
			patch.Techs = &value
		case "--unit-research":
			value, err := parseNextInt32List(args, &i, "--unit-research")
			if err != nil {
				return patch, false, err
			}
			patch.UnitResearch = &value
		case "--mode":
			value, err := parseNextInt32List(args, &i, "--mode")
			if err != nil {
				return patch, false, err
			}
			patch.Mode = &value
		case "--vertical-line":
			value, err := parseNextInt32(args, &i, "--vertical-line")
			if err != nil {
				return patch, false, err
			}
			patch.VerticalLine = &value
		case "--location-in-age":
			value, err := parseNextInt32(args, &i, "--location-in-age")
			if err != nil {
				return patch, false, err
			}
			patch.LocationInAge = &value
		case "--line-mode":
			value, err := parseNextInt32(args, &i, "--line-mode")
			if err != nil {
				return patch, false, err
			}
			patch.LineMode = &value
		default:
			return patch, false, fmt.Errorf("unknown tech-tree research connection option %q", args[i])
		}
		changed = true
	}
	return patch, changed, nil
}

func collectUnits(idx *datfile.Index, filters datUnitFilters) []datfile.UnitSummary {
	var out []datfile.UnitSummary
	for _, civ := range idx.Civs {
		if filters.Civ != nil && civ.Index != *filters.Civ {
			continue
		}
		for _, unit := range civ.Units {
			if !unit.Present {
				continue
			}
			if filters.ID != nil && int(unit.ID) != *filters.ID && unit.Index != *filters.ID {
				continue
			}
			name := strings.ToLower(unit.Name)
			if filters.Name != "" && !strings.Contains(name, filters.Name) {
				continue
			}
			if filters.NameContains != "" && !strings.Contains(name, filters.NameContains) {
				continue
			}
			if filters.Class != "" && !datUnitMatchesClass(unit, filters.Class) {
				continue
			}
			out = append(out, unit)
		}
	}
	return out
}

func collectTechs(idx *datfile.Index, opts datTechListOptions) []datfile.Tech {
	out := make([]datfile.Tech, 0, len(idx.Techs))
	for _, tech := range idx.Techs {
		if opts.NameContains != "" && !strings.Contains(strings.ToLower(tech.Name), opts.NameContains) {
			continue
		}
		out = append(out, tech)
	}
	return out
}

func collectAbilities(idx *datfile.Index, opts datAbilityListOptions) []abilityListSummary {
	out := make([]abilityListSummary, 0, len(idx.Techs))
	effectOpts := datEffectListOptions{CommandType: opts.CommandType, Operand: opts.Operand}
	for _, tech := range idx.Techs {
		if tech.EffectID < 0 || int(tech.EffectID) >= len(idx.Effects) {
			continue
		}
		if opts.NameContains != "" && !strings.Contains(strings.ToLower(tech.Name), opts.NameContains) {
			continue
		}
		if opts.EffectID != nil && int(tech.EffectID) != *opts.EffectID {
			continue
		}
		effect := idx.Effects[tech.EffectID]
		matchingCommands := matchingEffectCommands(effect.Commands, effectOpts)
		if (opts.CommandType != nil || opts.Operand != nil) && len(matchingCommands) == 0 {
			continue
		}
		out = append(out, abilitySummary(tech, effect, matchingCommands))
	}
	return out
}

func abilitySummary(tech datfile.Tech, effect datfile.Effect, matchingCommands []datfile.EffectCommand) abilityListSummary {
	return abilityListSummary{
		TechID:                tech.Index,
		Name:                  tech.Name,
		Civ:                   tech.Civ,
		Type:                  tech.Type,
		IconID:                tech.IconID,
		RequiredTechCount:     tech.RequiredTechCount,
		ResearchLocationCount: len(tech.ResearchLocations),
		EffectID:              tech.EffectID,
		EffectName:            effect.Name,
		CommandCount:          len(effect.Commands),
		MatchingCommands:      matchingCommands,
		TechRecordStart:       tech.Span.Start,
		TechRecordEnd:         tech.Span.End,
		EffectRecordStart:     effect.Span.Start,
		EffectRecordEnd:       effect.Span.End,
	}
}

func datAbilityVerification() aoe2.VerificationClaim {
	claim := aoe2.StructureVerification(true)
	claim.Note = "ability views join DAT tech records to their referenced effect records; this is structure-verified authoring context, not an in-engine proof that the ability appears in a UI or can be researched."
	return claim
}

func datUnitMatchesClass(unit datfile.UnitSummary, class string) bool {
	isBuilding := unit.Type == 80 || unit.Type == 90
	switch class {
	case "building":
		return isBuilding
	case "unit":
		return !isBuilding
	case "creatable":
		return unit.Creatable != nil
	default:
		return true
	}
}

type unitListSummary struct {
	CivIndex           int     `json:"civ_index"`
	Index              int     `json:"index"`
	Type               int     `json:"type"`
	ID                 int16   `json:"id"`
	Name               string  `json:"name"`
	Class              int16   `json:"class"`
	ClassName          string  `json:"class_name,omitempty"`
	HitPoints          int16   `json:"hit_points"`
	CollisionSizeX     float32 `json:"collision_size_x"`
	CollisionSizeY     float32 `json:"collision_size_y"`
	CollisionSizeZ     float32 `json:"collision_size_z"`
	ClearanceSizeX     float32 `json:"clearance_size_x"`
	ClearanceSizeY     float32 `json:"clearance_size_y"`
	TerrainTableID     int16   `json:"terrain_table_id"`
	ObstructionTypeID  uint8   `json:"obstruction_type_id"`
	ObstructionClassID uint8   `json:"obstruction_class_id"`
	IconID             int16   `json:"icon_id"`
	Enabled            uint8   `json:"enabled"`
	HasType50          bool    `json:"has_type50"`
	HasCreatable       bool    `json:"has_creatable"`
	RecordStart        int     `json:"record_start"`
	RecordEnd          int     `json:"record_end"`
}

type effectListSummary struct {
	Index            int                     `json:"index"`
	Name             string                  `json:"name"`
	CommandCount     int                     `json:"command_count"`
	MatchingCommands []datfile.EffectCommand `json:"matching_commands,omitempty"`
	RecordStart      int                     `json:"record_start"`
	RecordEnd        int                     `json:"record_end"`
}

type techListSummary struct {
	Index                 int    `json:"index"`
	Name                  string `json:"name"`
	Civ                   int16  `json:"civ"`
	EffectID              int16  `json:"effect_id"`
	Type                  int16  `json:"type"`
	IconID                int16  `json:"icon_id"`
	RequiredTechCount     int16  `json:"required_tech_count"`
	ResearchLocationCount int    `json:"research_location_count"`
	RecordStart           int    `json:"record_start"`
	RecordEnd             int    `json:"record_end"`
}

type abilityListSummary struct {
	TechID                int                     `json:"tech_id"`
	Name                  string                  `json:"name"`
	Civ                   int16                   `json:"civ"`
	Type                  int16                   `json:"type"`
	IconID                int16                   `json:"icon_id"`
	RequiredTechCount     int16                   `json:"required_tech_count"`
	ResearchLocationCount int                     `json:"research_location_count"`
	EffectID              int16                   `json:"effect_id"`
	EffectName            string                  `json:"effect_name"`
	CommandCount          int                     `json:"command_count"`
	MatchingCommands      []datfile.EffectCommand `json:"matching_commands,omitempty"`
	TechRecordStart       int                     `json:"tech_record_start"`
	TechRecordEnd         int                     `json:"tech_record_end"`
	EffectRecordStart     int                     `json:"effect_record_start"`
	EffectRecordEnd       int                     `json:"effect_record_end"`
}

type techTreeListSummary struct {
	Version             string          `json:"version"`
	GameMetrics         datfile.Metrics `json:"game_metrics"`
	AgeCount            int             `json:"age_count"`
	BuildingCount       int             `json:"building_count"`
	UnitCount           int             `json:"unit_count"`
	ResearchCount       int             `json:"research_count"`
	TotalUnitTechGroups int32           `json:"total_unit_tech_groups"`
	Span                datfile.Span    `json:"span"`
}

func techTreeSummary(idx *datfile.Index) techTreeListSummary {
	return techTreeListSummary{
		Version:             idx.Version,
		GameMetrics:         idx.GameMetrics,
		AgeCount:            idx.TechTree.AgeCount,
		BuildingCount:       idx.TechTree.BuildingCount,
		UnitCount:           idx.TechTree.UnitCount,
		ResearchCount:       idx.TechTree.ResearchCount,
		TotalUnitTechGroups: idx.TechTree.TotalUnitTechGroups,
		Span:                idx.TechTree.Span,
	}
}

func compactEffects(effects []datfile.Effect, opts datEffectListOptions) []effectListSummary {
	out := make([]effectListSummary, 0, len(effects))
	for _, effect := range effects {
		matchingCommands := matchingEffectCommands(effect.Commands, opts)
		if (opts.CommandType != nil || opts.Operand != nil) && len(matchingCommands) == 0 {
			continue
		}
		out = append(out, effectListSummary{
			Index:            effect.Index,
			Name:             effect.Name,
			CommandCount:     len(effect.Commands),
			MatchingCommands: matchingCommands,
			RecordStart:      effect.Span.Start,
			RecordEnd:        effect.Span.End,
		})
	}
	return out
}

func matchingEffectCommands(commands []datfile.EffectCommand, opts datEffectListOptions) []datfile.EffectCommand {
	if opts.CommandType == nil && opts.Operand == nil {
		return nil
	}
	out := make([]datfile.EffectCommand, 0)
	for _, command := range commands {
		if opts.CommandType != nil && int(command.Type) != *opts.CommandType {
			continue
		}
		if opts.Operand != nil && !effectCommandOperandEquals(command, *opts.Operand) {
			continue
		}
		out = append(out, command)
	}
	return out
}

func effectCommandOperandEquals(command datfile.EffectCommand, value int) bool {
	if int(command.A) == value || int(command.B) == value || int(command.C) == value {
		return true
	}
	d := int(command.D)
	return command.D == float32(d) && d == value
}

func printDatSemanticPriors(report datcache.SemanticPriorsReport) {
	fmt.Printf("version: %s civs=%d source_rows=%d unit_identities=%d\n", report.Version, report.CivCount, report.SourceRows, report.UnitIdentities)
	fmt.Printf("verification: %s\n", report.Verification)
	fmt.Printf("zero_collision: units=%d creatable=%d\n", report.ZeroCollision, report.ZeroCollisionCreatable)
	fmt.Println("collision_signatures:")
	for _, bucket := range report.CollisionSignatures {
		fmt.Printf("- x_positive=%t y_positive=%t z_positive=%t units=%d creatable=%d\n", bucket.XPositive, bucket.YPositive, bucket.ZPositive, bucket.Units, bucket.Creatable)
	}
	fmt.Println("removal_buckets:")
	for _, bucket := range report.RemovalBuckets {
		fmt.Printf("- %s units=%d\n", bucket.Name, bucket.Units)
	}
}

func printDatEffectCommandMatrix(report datcodec.EffectCommandMatrixReport) {
	fmt.Printf("DAT effect command matrix: effects=%d commands=%d types=%d verification=%s\n",
		report.EffectCount, report.CommandCount, report.TypeCount, report.Verification.Label)
	filters := report.Filters
	if filters.CommandType != nil || filters.Operand != nil || filters.ReferenceKind != "" || filters.ReferenceField != "" || filters.UnknownOnly || filters.TypedOnly || filters.CandidateOnly {
		parts := make([]string, 0, 5)
		if filters.CommandType != nil {
			parts = append(parts, fmt.Sprintf("command_type=%d", *filters.CommandType))
		}
		if filters.Operand != nil {
			parts = append(parts, fmt.Sprintf("operand=%d", *filters.Operand))
		}
		if filters.ReferenceKind != "" {
			parts = append(parts, "reference_kind="+filters.ReferenceKind)
		}
		if filters.ReferenceField != "" {
			parts = append(parts, "reference_field="+filters.ReferenceField)
		}
		if filters.UnknownOnly {
			parts = append(parts, "unknown_only=true")
		}
		if filters.TypedOnly {
			parts = append(parts, "typed_only=true")
		}
		if filters.CandidateOnly {
			parts = append(parts, "candidate_only=true")
		}
		fmt.Printf("filters: %s\n", strings.Join(parts, " "))
		fmt.Printf("matching_effects=%d\n", report.MatchingEffectCount)
	}
	for _, typ := range report.CommandTypes {
		fmt.Printf("\ntype %d %s: commands=%d effects=%d\n", typ.Type, typ.TypeName, typ.CommandCount, typ.EffectCount)
		fmt.Printf("  operands: A[%s] B[%s] C[%s] D[%s]\n",
			formatOperandProfile(typ.Operands.A),
			formatOperandProfile(typ.Operands.B),
			formatOperandProfile(typ.Operands.C),
			formatOperandProfile(typ.Operands.D),
		)
		if len(typ.TypedReferences) > 0 {
			fmt.Print("  typed_refs:")
			for _, ref := range typ.TypedReferences {
				fmt.Printf(" %s.%s=%d", ref.Kind, ref.Field, ref.Count)
				if len(ref.SampleIDs) > 0 {
					fmt.Printf(" samples=%s", formatValueCounts(ref.SampleIDs, true))
				}
			}
			fmt.Println()
		}
		if len(typ.CandidateRefs) > 0 {
			fmt.Print("  candidate_refs:")
			for _, ref := range typ.CandidateRefs {
				fmt.Printf(" %s.%s=%d", ref.Kind, ref.Field, ref.Count)
				if len(ref.SampleIDs) > 0 {
					fmt.Printf(" samples=%s", formatValueCounts(ref.SampleIDs, true))
				}
			}
			fmt.Println()
		}
		if len(typ.Attributes) > 0 {
			fmt.Printf("  attributes: %s\n", formatValueCounts(typ.Attributes, true))
		}
		if len(typ.PackedTypeIDs) > 0 || len(typ.PackedAmounts) > 0 {
			fmt.Printf("  packed_attack_armor:")
			if len(typ.PackedTypeIDs) > 0 {
				fmt.Printf(" type_ids=%s", formatValueCounts(typ.PackedTypeIDs, false))
			}
			if len(typ.PackedAmounts) > 0 {
				fmt.Printf(" amounts=%s", formatValueCounts(typ.PackedAmounts, false))
			}
			fmt.Println()
		}
		if len(typ.Examples) > 0 {
			fmt.Println("  examples:")
			for _, example := range typ.Examples {
				c := example.Command
				fmt.Printf("    effect=%d command=%d name=%q row={type:%d a:%d b:%d c:%d d:%g}\n",
					example.EffectID, c.Index, example.EffectName, c.Type, c.A, c.B, c.C, c.D)
			}
		}
	}
}

func printDatEffectExplain(report datcodec.EffectExplainReport) {
	fmt.Printf("DAT effect %d/%s: %s\n", report.EffectID, report.EffectName, report.Summary)
	fmt.Printf("verification: %s\n", report.Verification.Label)
	if report.Verification.Note != "" {
		fmt.Printf("note: %s\n", report.Verification.Note)
	}
	for _, command := range report.Commands {
		fmt.Printf("- command[%d] type=%d/%s: %s\n", command.Index, command.Type, command.TypeName, command.Summary)
		for _, detail := range command.Details {
			fmt.Printf("  detail: %s\n", detail)
		}
		for _, warning := range command.Warnings {
			fmt.Printf("  warning: %s\n", warning)
		}
	}
}

func printDatTechExplain(report datcodec.TechExplainReport) {
	fmt.Printf("DAT tech %d/%s: %s\n", report.TechID, report.TechName, report.Summary)
	fmt.Printf("verification: %s\n", report.Verification.Label)
	if report.Verification.Note != "" {
		fmt.Printf("note: %s\n", report.Verification.Note)
	}
	for _, line := range report.TechLines {
		fmt.Printf("- %s\n", line)
	}
	if report.Effect != nil {
		fmt.Println("effect:")
		for _, command := range report.Effect.Commands {
			fmt.Printf("- command[%d] type=%d/%s: %s\n", command.Index, command.Type, command.TypeName, command.Summary)
			for _, detail := range command.Details {
				fmt.Printf("  detail: %s\n", detail)
			}
			for _, warning := range command.Warnings {
				fmt.Printf("  warning: %s\n", warning)
			}
		}
	}
	for _, warning := range report.Warnings {
		fmt.Printf("warning: %s\n", warning)
	}
}

func printDatReferences(report datcodec.ReferenceReport) {
	fmt.Printf("DAT references: target=%s id=%d total=%d returned=%d rewrite_supported=%d known_readonly=%d possible_operand=%d candidate_operand=%d unsupported=%d verification=%s\n",
		report.Request.Section, report.Request.ID, report.Summary.Total, report.Summary.Returned, report.Summary.RewriteSupported,
		report.Summary.KnownReadonly, report.Summary.PossibleOperand, report.Summary.CandidateOperand, report.Summary.Unsupported, report.Verification.Label)
	filters := report.Filters
	if filters.Class != "" || filters.Confidence != "" || filters.SourceSection != "" || filters.Limit > 0 {
		parts := make([]string, 0, 4)
		if filters.Class != "" {
			parts = append(parts, "class="+filters.Class)
		}
		if filters.Confidence != "" {
			parts = append(parts, "confidence="+filters.Confidence)
		}
		if filters.SourceSection != "" {
			parts = append(parts, "source_section="+filters.SourceSection)
		}
		if filters.Limit > 0 {
			parts = append(parts, fmt.Sprintf("limit=%d", filters.Limit))
		}
		fmt.Printf("filters: %s\n", strings.Join(parts, " "))
	}
	for _, ref := range report.References {
		fmt.Printf("- class=%s section=%s id=%d field=%s", ref.Class, ref.Section, ref.ID, ref.Field)
		if ref.Confidence != "" {
			fmt.Printf(" confidence=%s", ref.Confidence)
		}
		if ref.Reason != "" {
			fmt.Printf(" reason=%q", ref.Reason)
		}
		fmt.Println()
	}
}

func formatOperandProfile(profile datcodec.EffectCommandOperandProfile) string {
	return fmt.Sprintf("min=%g max=%g neg=%d zero=%d pos=%d distinct=%d",
		profile.Min, profile.Max, profile.Negative, profile.Zero, profile.Positive, profile.Distinct)
}

func formatValueCounts(values []datcodec.EffectCommandValueCount, withNames bool) string {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		label := strconv.Itoa(value.Value)
		if withNames && value.Name != "" {
			label += ":" + value.Name
		}
		parts = append(parts, fmt.Sprintf("%s(%d)", label, value.Count))
	}
	return strings.Join(parts, ",")
}

func compactTechs(techs []datfile.Tech) []techListSummary {
	out := make([]techListSummary, 0, len(techs))
	for _, tech := range techs {
		out = append(out, techListSummary{
			Index:                 tech.Index,
			Name:                  tech.Name,
			Civ:                   tech.Civ,
			EffectID:              tech.EffectID,
			Type:                  tech.Type,
			IconID:                tech.IconID,
			RequiredTechCount:     tech.RequiredTechCount,
			ResearchLocationCount: len(tech.ResearchLocations),
			RecordStart:           tech.Span.Start,
			RecordEnd:             tech.Span.End,
		})
	}
	return out
}

func compactUnits(units []datfile.UnitSummary) []unitListSummary {
	out := make([]unitListSummary, 0, len(units))
	for _, unit := range units {
		out = append(out, unitListSummary{
			CivIndex:           unit.CivIndex,
			Index:              unit.Index,
			Type:               unit.Type,
			ID:                 unit.ID,
			Name:               unit.Name,
			Class:              unit.Class,
			ClassName:          unit.ClassName,
			HitPoints:          unit.HitPoints,
			CollisionSizeX:     unit.CollisionSizeX,
			CollisionSizeY:     unit.CollisionSizeY,
			CollisionSizeZ:     unit.CollisionSizeZ,
			ClearanceSizeX:     unit.ClearanceSizeX,
			ClearanceSizeY:     unit.ClearanceSizeY,
			TerrainTableID:     unit.TerrainTableID,
			ObstructionTypeID:  unit.ObstructionTypeID,
			ObstructionClassID: unit.ObstructionClassID,
			IconID:             unit.IconID,
			Enabled:            unit.Enabled,
			HasType50:          unit.Type50 != nil,
			HasCreatable:       unit.Creatable != nil,
			RecordStart:        unit.RecordStart,
			RecordEnd:          unit.RecordEnd,
		})
	}
	return out
}

func printDatPalette(report datfile.PaletteReport) {
	fmt.Printf("path: %s\n", report.Path)
	fmt.Printf("version: %s\n", report.Version)
	fmt.Printf("verification: %s\n", report.Verification)
	fmt.Printf("summary: unit_records=%d returned=%d\n", report.UnitCount, report.Returned)
	if report.Filters.ID != nil || report.Filters.MinVariants > 0 {
		id := "any"
		if report.Filters.ID != nil {
			id = strconv.Itoa(*report.Filters.ID)
		}
		fmt.Printf("filters: id=%s min_variants=%d\n", id, report.Filters.MinVariants)
	}
	for _, row := range report.Rows {
		variants := "n/a"
		if row.VariantCount != nil {
			variants = strconv.Itoa(*row.VariantCount)
		}
		fmt.Printf("- unit=%d %q graphic=%d %q file=%q slp=%d angles=%d frames=%d sequence=%d class=%s variants=%s confidence=%s civs=%s\n",
			row.UnitID, row.UnitName, row.StandingGraphic1, row.GraphicName, row.FileName, row.SLP,
			row.AngleCount, row.FrameCount, row.SequenceType, row.Classification, variants, row.Confidence,
			intSampleText(row.CivIndices, 12))
		if row.VariantNote != "" {
			fmt.Printf("  note: %s\n", row.VariantNote)
		}
	}
}

const cbaUsage = `usage:
  kit cba balance [--text|--json]
  kit cba sidechannels <file.aoe2record|replay.zip> [--text]
  kit cba phase-facts <file.aoe2record|replay.zip> --dat <empires2_x2_p1.dat> [--text]
  kit cba trigger-razes <file.aoe2record|replay.zip> [--dat <empires2_x2_p1.dat>] [--text]
  kit cba trigger-spawns <file.aoe2record|replay.zip> [--dat <empires2_x2_p1.dat>] [--text]
  kit cba replay <progression|razes|raze-pressure|phases|perf|doctrine> <file.aoe2record|replay.zip> [--dat <empires2_x2_p1.dat>] [--text]
  kit cba registry <replay|dir>... [--out <file.json>] [--text]
  kit cba ingest --corpus <file.jsonl> --registry <file.json> [--force] [--limit N]
  kit cba ingest --corpus <file.jsonl> --unverified <replay|dir>...  [--force]
  kit cba archive --queue <q.json> --out <dir> --max N [--delay-ms MS] [--manifest f.jsonl]
  kit cba axes   --corpus <file.jsonl> [--min-games N] [--names <file.json>] [--text]
  kit cba export --corpus <file.jsonl> [--names <file.json>] [--sql]`

func runCBA(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, cbaUsage)
		os.Exit(2)
	}
	switch args[0] {
	case "ingest":
		runCBAIngest(args[1:])
		return
	case "axes":
		runCBAAxes(args[1:])
		return
	case "export":
		runCBAExport(args[1:])
		return
	case "registry":
		runCBARegistry(args[1:])
		return
	case "archive":
		runCBAArchive(args[1:])
		return
	case "balance":
		textOut := false
		for _, arg := range args[1:] {
			switch arg {
			case "--text":
				textOut = true
			case "--json":
				textOut = false
			default:
				die("kit cba balance", fmt.Errorf("unknown option %q", arg))
			}
		}
		report := cba.BuildBalance()
		if textOut {
			printCBABalance(report)
		} else {
			printJSON(report)
		}
		return
	case "sidechannels":
		if len(args) < 2 {
			die("kit cba sidechannels", fmt.Errorf("need a replay path"))
		}
		path := args[1]
		textOut := false
		for i := 2; i < len(args); i++ {
			switch args[i] {
			case "--text":
				textOut = true
			case "--json":
				textOut = false
			default:
				die("kit cba sidechannels", fmt.Errorf("unknown option %q", args[i]))
			}
		}
		report, err := cba.BuildSideChannels(path)
		if err != nil {
			die("kit cba sidechannels", err)
		}
		if textOut {
			printCBASideChannels(report)
		} else {
			printJSON(report)
		}
		return
	case "phase-facts":
		if len(args) < 2 {
			die("kit cba phase-facts", fmt.Errorf("need a replay path"))
		}
		path := args[1]
		datPath := ""
		textOut := false
		for i := 2; i < len(args); i++ {
			switch args[i] {
			case "--dat":
				i++
				if i >= len(args) {
					die("kit cba phase-facts", fmt.Errorf("--dat requires a DAT path"))
				}
				datPath = args[i]
			case "--text":
				textOut = true
			case "--json":
				textOut = false
			default:
				die("kit cba phase-facts", fmt.Errorf("unknown option %q", args[i]))
			}
		}
		report, err := cba.BuildPhaseFacts(path, datPath)
		if err != nil {
			die("kit cba phase-facts", err)
		}
		if textOut {
			printCBAPhaseFacts(report)
		} else {
			printJSON(report)
		}
		return
	case "trigger-razes":
		if len(args) < 2 {
			die("kit cba trigger-razes", fmt.Errorf("need a replay path"))
		}
		path := args[1]
		datPath := ""
		textOut := false
		for i := 2; i < len(args); i++ {
			switch args[i] {
			case "--dat":
				i++
				if i >= len(args) {
					die("kit cba trigger-razes", fmt.Errorf("--dat requires a DAT path"))
				}
				datPath = args[i]
			case "--text":
				textOut = true
			case "--json":
				textOut = false
			default:
				die("kit cba trigger-razes", fmt.Errorf("unknown option %q", args[i]))
			}
		}
		report, err := cba.BuildTriggerRazes(path, datPath)
		if err != nil {
			die("kit cba trigger-razes", err)
		}
		if textOut {
			printCBATriggerRazes(report)
		} else {
			printJSON(report)
		}
		return
	case "trigger-spawns":
		if len(args) < 2 {
			die("kit cba trigger-spawns", fmt.Errorf("need a replay path"))
		}
		path := args[1]
		datPath := ""
		textOut := false
		for i := 2; i < len(args); i++ {
			switch args[i] {
			case "--dat":
				i++
				if i >= len(args) {
					die("kit cba trigger-spawns", fmt.Errorf("--dat requires a DAT path"))
				}
				datPath = args[i]
			case "--text":
				textOut = true
			case "--json":
				textOut = false
			default:
				die("kit cba trigger-spawns", fmt.Errorf("unknown option %q", args[i]))
			}
		}
		report, err := cba.BuildTriggerSpawns(path, datPath)
		if err != nil {
			die("kit cba trigger-spawns", err)
		}
		if textOut {
			printCBATriggerSpawns(report)
		} else {
			printJSON(report)
		}
		return
	}
	if len(args) < 3 || args[0] != "replay" {
		fmt.Fprintln(os.Stderr, cbaUsage)
		os.Exit(2)
	}
	textOut := false
	datPath := ""
	for i := 3; i < len(args); i++ {
		switch args[i] {
		case "--dat":
			i++
			if i >= len(args) {
				die("kit cba replay", fmt.Errorf("--dat requires a DAT path"))
			}
			datPath = args[i]
		case "--text":
			textOut = true
		case "--json":
			textOut = false
		default:
			die("kit cba replay", fmt.Errorf("unknown option %q", args[i]))
		}
	}
	switch args[1] {
	case "progression":
		report, err := cba.BuildProgression(args[2])
		if err != nil {
			die("kit cba replay progression", err)
		}
		if textOut {
			printCBAProgression(report)
		} else {
			printJSON(report)
		}
	case "razes":
		report, err := cba.BuildRazes(args[2])
		if err != nil {
			die("kit cba replay razes", err)
		}
		if textOut {
			printCBARazes(report)
		} else {
			printJSON(report)
		}
	case "raze-pressure":
		report, err := cba.BuildRazePressure(args[2])
		if err != nil {
			die("kit cba replay raze-pressure", err)
		}
		if textOut {
			printCBARazePressure(report)
		} else {
			printJSON(report)
		}
	case "phases":
		report, err := cba.BuildPhaseTimeline(args[2], datPath)
		if err != nil {
			die("kit cba replay phases", err)
		}
		if textOut {
			printCBAPhaseTimeline(report)
		} else {
			printJSON(report)
		}
	case "perf":
		report, err := cba.BuildPerformance(args[2])
		if err != nil {
			die("kit cba replay perf", err)
		}
		if textOut {
			printCBAPerformance(report)
		} else {
			printJSON(report)
		}
	case "doctrine":
		report, err := cba.BuildDoctrine(args[2])
		if err != nil {
			die("kit cba replay doctrine", err)
		}
		if textOut {
			printCBADoctrine(report)
		} else {
			printJSON(report)
		}
	default:
		fmt.Fprintln(os.Stderr, "usage: kit cba replay <progression|razes|raze-pressure|phases|perf|doctrine> <file.aoe2record|replay.zip> [--dat <empires2_x2_p1.dat>] [--text]")
		os.Exit(2)
	}
}

func printCBASideChannels(report *cba.SideChannelReport) {
	fmt.Printf("path: %s\n", report.Path)
	fmt.Printf("method: %s\n", report.Method)
	fmt.Printf("verification: %s\n", report.Verification)
	fmt.Printf("graph: sha256=%s token=%q ladder=%t reason=%q\n",
		report.GraphSHA256, report.ScenarioToken, report.Scenario.Ladder, report.Scenario.Reason)
	if report.DataSet.Status != "" {
		fmt.Printf("data_set: status=%s active=%q source=%s confidence=%s\n",
			report.DataSet.Status, report.DataSet.ActiveDataSet, report.DataSet.Source, report.DataSet.Confidence)
	}
	fmt.Printf("summary: variant=%s triggers=%d effects=%d conditions=%d messages=%d modify_resource=%d change_variable=%d accumulate_attribute=%d scoreboard_rows=%d engine_attrs=%d resources=%d thresholds=%d\n",
		report.Summary.SideChannelVariant,
		report.Summary.TriggerCount, report.Summary.EffectCount, report.Summary.ConditionCount,
		report.Summary.MessageCount, report.Summary.ModifyResourceEffects, report.Summary.ChangeVariableEffects,
		report.Summary.AccumAttributeConds, report.Summary.ScoreboardRows, report.Summary.EngineAttrChannels,
		report.Summary.ResourcesWritten, report.Summary.ThresholdChannels)
	if len(report.Scoreboard) > 0 {
		fmt.Println("scoreboard_rows:")
		for _, row := range report.Scoreboard {
			fmt.Printf("- %s players=%v kills=%d/%d deaths=%d/%d razes=%d/%d confidence=%s\n",
				row.Label, row.Players, row.KillResourceA, row.KillResourceB,
				row.DeathResourceA, row.DeathResourceB, row.RazeResourceA,
				row.RazeResourceB, row.Confidence)
		}
	}
	if len(report.EngineAttrs) > 0 {
		fmt.Println("engine_attributes:")
		for _, row := range report.EngineAttrs {
			fmt.Printf("- attr=%d %s semantic=%s quantities=%v players=%v conditions=%d\n",
				row.Attribute, row.Name, row.Semantic, row.Quantities, row.Players, row.ConditionCount)
		}
	}
	if len(report.Resources) > 0 {
		fmt.Println("resources:")
		for _, row := range report.Resources {
			fmt.Printf("- res=%d family=%s writes=%d amounts=%v players=%v\n",
				row.Resource, row.Family, row.Count, row.Amounts, row.Players)
		}
	}
	if len(report.RegisterMap) > 0 {
		fmt.Println("register_map:")
		for _, row := range report.RegisterMap {
			fmt.Printf("- lane=%d %s metric=%s layer=%s players=%v confidence=%s\n",
				row.Lane, row.Label, row.Metric, row.Layer, row.Players, row.Confidence)
			for _, res := range row.Resources {
				fmt.Printf("  - res=%d role=%s writes=%d amounts=%v players=%v", res.Resource, res.Role, res.WriteCount, res.Amounts, res.Players)
				if len(res.Evidence) > 0 {
					fmt.Printf(" evidence=")
					limit := len(res.Evidence)
					if limit > 3 {
						limit = 3
					}
					for i, ev := range res.Evidence[:limit] {
						if i > 0 {
							fmt.Print(",")
						}
						fmt.Printf("t%d/e%d/p%d", ev.TriggerIndex, ev.EffectIndex, ev.Player)
						if ev.Operation != 0 {
							fmt.Printf("/op%d", ev.Operation)
						}
						fmt.Printf("/q%d", ev.Amount)
					}
				}
				fmt.Println()
			}
		}
	}
	if len(report.Thresholds) > 0 {
		fmt.Println("thresholds:")
		for _, row := range report.Thresholds {
			fmt.Printf("- kind=%s attr=%d players=%v values=%v", row.Kind, row.Attribute, row.Players, row.Values)
			if len(row.Messages) > 0 {
				fmt.Printf(" messages=%d", len(row.Messages))
			}
			fmt.Println()
		}
	}
	if len(report.Templates) > 0 {
		fmt.Println("templates:")
		for _, line := range report.Templates {
			fmt.Printf("- %s\n", strings.ReplaceAll(line, "\n", " / "))
		}
	}
	if len(report.Notes) > 0 {
		fmt.Println("notes:")
		for _, note := range report.Notes {
			fmt.Printf("- %s confidence=%s detail=%q\n", note.Name, note.Confidence, note.Detail)
		}
	}
	for _, warning := range report.Warnings {
		fmt.Printf("warning: %s\n", warning)
	}
}

func printCBABalance(report *cba.BalanceReport) {
	fmt.Printf("method: %s\n", report.Method)
	fmt.Printf("verification: %s\n", report.Verification)
	fmt.Printf("summary: rows=%d civ_rows=%d versions=%d trigger_evidence=%d missing=%d\n",
		report.Summary.Rows, report.Summary.CivRows, report.Summary.VersionRows, report.Summary.TriggerEvidence, report.Summary.MissingRows)
	if len(report.Rows) > 0 {
		fmt.Println("rows:")
		for _, row := range report.Rows {
			fmt.Printf("- version=%s civ=%d/%s razes_to_vill=%s castle_kills=%s imp_kills=%s units=%s spawn_s=%s stable_elephants=%s battle_elephants=%s coverage=%s sources=%s",
				row.Version, row.CivID, row.CivName, intPtrText(row.RazesToVillager), intPtrText(row.CastleKills),
				intPtrText(row.ImperialKills), intPtrText(row.UnitCount), intPtrText(row.SpawnSeconds),
				intPtrText(row.StableElephants), intPtrText(row.BattleElephants), row.Coverage, strings.Join(row.Sources, ","))
			if row.PreferredForV292 {
				fmt.Printf(" preferred_v292=true")
			}
			if row.Notes != "" {
				fmt.Printf(" note=%q", row.Notes)
			}
			fmt.Println()
		}
	}
	if len(report.Missing) > 0 {
		fmt.Println("missing:")
		for _, missing := range report.Missing {
			civ := "all_unmapped"
			if missing.CivName != "" {
				civ = fmt.Sprintf("%d/%s", missing.CivID, missing.CivName)
			}
			fmt.Printf("- version=%s civ=%s reason=%q\n", missing.Version, civ, missing.Reason)
		}
	}
	if len(report.Evidence) > 0 {
		fmt.Println("evidence:")
		for _, ev := range report.Evidence {
			fmt.Printf("- version=%s kind=%s attr=%d amounts=%v counts=%v confidence=%s source=%q",
				ev.Version, ev.Kind, ev.Attribute, ev.Amounts, ev.Counts, ev.Confidence, ev.Source)
			if ev.Notes != "" {
				fmt.Printf(" note=%q", ev.Notes)
			}
			fmt.Println()
		}
	}
	for _, warning := range report.Warnings {
		fmt.Printf("warning: %s\n", warning)
	}
}

func printCBATriggerRazes(report *cba.TriggerRazeReport) {
	fmt.Printf("path: %s\n", report.Path)
	if report.DatPath != "" {
		fmt.Printf("dat: %s\n", report.DatPath)
	}
	fmt.Printf("method: %s\n", report.Method)
	fmt.Printf("verification: %s\n", report.Verification)
	fmt.Printf("mechanics: condition=%d/%s attr=%d/%s reward_effect=%d/%s activator_effect=%d/%s\n",
		report.Mechanics.ConditionType, report.Mechanics.ConditionName,
		report.Mechanics.RazeAttribute, report.Mechanics.RazeAttributeName,
		report.Mechanics.RewardEffectType, report.Mechanics.RewardEffectName,
		report.Mechanics.ActivatorEffectType, report.Mechanics.ActivatorEffectName)
	fmt.Printf("summary: triggers=%d reward_rungs=%d activators=%d rows=%d resolved_civs=%d buckets=%d conflicts=%d unresolved_techs=%d mapuche=%d khmer=%d\n",
		report.Summary.TriggerCount, report.Summary.RewardRungTriggers, report.Summary.ActivatorTriggers,
		report.Summary.Rows, report.Summary.ResolvedCivs, report.Summary.Buckets, report.Summary.Conflicts,
		report.Summary.UnresolvedTechs, report.Summary.MapucheRazesToVillager, report.Summary.KhmerRazesToVillager)
	for _, bucket := range report.Buckets {
		names := strings.Join(bucket.CivNames, ", ")
		if names == "" {
			names = fmt.Sprintf("techs=%v", bucket.TechIDs)
		}
		fmt.Printf("- %d raze(s): %d tech refs, %d civs: %s\n",
			bucket.RazesToVillager, bucket.Count, len(bucket.CivIDs), names)
	}
	if len(report.Conflicts) > 0 {
		fmt.Println("conflicts:")
		for _, conflict := range report.Conflicts {
			fmt.Printf("- tech=%d values=%v\n", conflict.TechID, conflict.Values)
		}
	}
	for _, warning := range report.Warnings {
		fmt.Printf("warning: %s\n", warning)
	}
}

func printCBATriggerSpawns(report *cba.TriggerSpawnReport) {
	fmt.Printf("path: %s\n", report.Path)
	if report.DatPath != "" {
		fmt.Printf("dat: %s\n", report.DatPath)
	}
	fmt.Printf("method: %s\n", report.Method)
	fmt.Printf("verification: %s\n", report.Verification)
	fmt.Printf("mechanics: seconds=%s count_condition=%d/%s tech_condition=%d spawn_effects=%v %s\n",
		report.Mechanics.SecondsSource, report.Mechanics.CountConditionType,
		report.Mechanics.CountConditionName, report.Mechanics.TechnologyCondition,
		report.Mechanics.SpawnEffectTypes, report.Mechanics.SpawnEffectNames)
	fmt.Printf("summary: triggers=%d seconds_triggers=%d count_triggers=%d rows=%d complete=%d resolved_civs=%d buckets=%d conflicts=%d unresolved_techs=%d mapuche=%d/%d khmer=%d/%d persians=%d/%d\n",
		report.Summary.TriggerCount, report.Summary.SpawnerSecondTriggers, report.Summary.SpawnCountTriggers,
		report.Summary.Rows, report.Summary.CompleteRows, report.Summary.ResolvedCivs, report.Summary.Buckets,
		report.Summary.Conflicts, report.Summary.UnresolvedTechs, report.Summary.MapucheSpawnUnitCount,
		report.Summary.MapucheSpawnSeconds, report.Summary.KhmerSpawnUnitCount, report.Summary.KhmerSpawnSeconds,
		report.Summary.PersiansSpawnUnitCount, report.Summary.PersiansSpawnSeconds)
	for _, bucket := range report.Buckets {
		names := strings.Join(bucket.CivNames, ", ")
		if names == "" {
			names = fmt.Sprintf("techs=%v", bucket.TechIDs)
		}
		fmt.Printf("- units=%s seconds=%s: %d tech refs, %d civs: %s\n",
			intPtrText(bucket.SpawnUnitCount), intPtrText(bucket.SpawnSeconds),
			bucket.Count, len(bucket.CivIDs), names)
	}
	if len(report.Conflicts) > 0 {
		fmt.Println("conflicts:")
		for _, conflict := range report.Conflicts {
			fmt.Printf("- tech=%d field=%s values=%v\n", conflict.TechID, conflict.Field, conflict.Values)
		}
	}
	for _, warning := range report.Warnings {
		fmt.Printf("warning: %s\n", warning)
	}
}

// cbaFlags pulls the options shared by the corpus-level subcommands, returning the
// leftover positional arguments.
type cbaFlags struct {
	corpus     string
	names      string
	registry   string
	out        string
	force      bool
	sqlOut     bool
	textOut    bool
	unverified bool
	limit      int
	minGames   int
	workers    int
	queue      string
	manifest   string
	rejectDir  string
	max        int
	delayMS    int
}

func parseCBAFlags(cmd string, args []string) (cbaFlags, []string) {
	f := cbaFlags{}
	var rest []string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--corpus", "--names", "--registry", "--out", "--queue", "--manifest", "--reject-dir":
			if i+1 >= len(args) {
				die(cmd, fmt.Errorf("%s requires a value", args[i]))
			}
			switch args[i] {
			case "--corpus":
				f.corpus = args[i+1]
			case "--names":
				f.names = args[i+1]
			case "--registry":
				f.registry = args[i+1]
			case "--out":
				f.out = args[i+1]
			case "--queue":
				f.queue = args[i+1]
			case "--manifest":
				f.manifest = args[i+1]
			case "--reject-dir":
				f.rejectDir = args[i+1]
			}
			i++
		case "--limit", "--min-games", "--workers", "--max", "--delay-ms":
			if i+1 >= len(args) {
				die(cmd, fmt.Errorf("%s requires a value", args[i]))
			}
			n, err := strconv.Atoi(args[i+1])
			if err != nil {
				die(cmd, fmt.Errorf("%s: %w", args[i], err))
			}
			switch args[i] {
			case "--limit":
				f.limit = n
			case "--min-games":
				f.minGames = n
			case "--workers":
				f.workers = n
			case "--max":
				f.max = n
			case "--delay-ms":
				f.delayMS = n
			}
			i++
		case "--force":
			f.force = true
		case "--unverified":
			f.unverified = true
		case "--sql":
			f.sqlOut = true
		case "--text":
			f.textOut = true
		case "--json":
			f.textOut = false
		default:
			if strings.HasPrefix(args[i], "--") {
				die(cmd, fmt.Errorf("unknown option %q", args[i]))
			}
			rest = append(rest, args[i])
		}
	}
	return f, rest
}

func requireCorpus(cmd string, f cbaFlags) {
	if f.corpus == "" {
		die(cmd, fmt.Errorf("--corpus <file.jsonl> is required"))
	}
}

// loadCBANames reads an optional profile_id -> display name map. Names live outside the
// corpus because they come from lobby/registry data, not from the replay extractor.
func loadCBANames(path string) map[int]string {
	names := map[int]string{}
	if path == "" {
		return names
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		die("kit cba", err)
	}
	asString := map[string]string{}
	if err := json.Unmarshal(raw, &asString); err != nil {
		die("kit cba --names", err)
	}
	for k, v := range asString {
		id, err := strconv.Atoi(k)
		if err != nil {
			continue
		}
		names[id] = v
	}
	return names
}

// collectReplays expands directories into replay paths so a whole pull tree can be
// ingested in one call.
func collectReplays(inputs []string) []string {
	var out []string
	for _, in := range inputs {
		info, err := os.Stat(in)
		if err != nil {
			fmt.Fprintf(os.Stderr, "skip %s: %v\n", in, err)
			continue
		}
		if !info.IsDir() {
			out = append(out, in)
			continue
		}
		filepath.WalkDir(in, func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			switch strings.ToLower(filepath.Ext(p)) {
			case ".aoe2record", ".zip":
				out = append(out, p)
			}
			return nil
		})
	}
	sort.Strings(out)
	return out
}

func runCBAIngest(args []string) {
	f, rest := parseCBAFlags("kit cba ingest", args)
	requireCorpus("kit cba ingest", f)

	// Ingest is GATED on the registry by default. Scenario identity is what keeps
	// non-ladder replays -- other CBA versions, MCB, test scenarios, AI games -- out of
	// the ratings, and an unverified corpus produces plausible-looking numbers computed
	// over the wrong games. Bypassing the gate has to be typed out explicitly.
	var paths []string
	switch {
	case f.registry != "":
		if len(rest) > 0 {
			die("kit cba ingest", fmt.Errorf("--registry supplies the replay list; drop the positional paths"))
		}
		reg, err := cba.LoadRegistry(f.registry)
		if err != nil {
			die("kit cba ingest", err)
		}
		paths = reg.Admitted()
		fmt.Fprintf(os.Stderr, "registry %s: %d admitted of %d scanned\n",
			f.registry, reg.Summary.Admitted, reg.Summary.Scanned)
	case f.unverified:
		if len(rest) == 0 {
			die("kit cba ingest", fmt.Errorf("need at least one replay or directory"))
		}
		paths = collectReplays(rest)
		fmt.Fprintln(os.Stderr,
			"WARNING: --unverified skips the scenario gate; this corpus may contain non-ladder games")
	default:
		die("kit cba ingest", fmt.Errorf(
			"refusing to ingest without a scenario gate: pass --registry <file.json> "+
				"(build it with 'kit cba registry'), or --unverified to bypass deliberately"))
	}

	corpus, err := cba.LoadCorpus(f.corpus)
	if err != nil {
		die("kit cba ingest", err)
	}
	fmt.Fprintf(os.Stderr, "ingesting %d replay(s) into %s (%d already present)\n",
		len(paths), f.corpus, len(corpus.Games))
	fmt.Fprintf(os.Stderr, "extraction peaks near %d MiB per worker; concurrency is chosen "+
		"from available memory (override with --workers)\n", cba.ExtractWorkerMiB)

	res := cba.Ingest(corpus, paths, cba.IngestOptions{
		Force:   f.force,
		Limit:   f.limit,
		Workers: f.workers,
		Progress: func(path, status string, err error) {
			if status == "fail" {
				fmt.Fprintf(os.Stderr, "  fail %s: %v\n", filepath.Base(path), err)
			}
		},
	})
	if err := corpus.Save(f.corpus); err != nil {
		die("kit cba ingest", err)
	}
	printJSON(res)
}

func runCBAArchive(args []string) {
	f, _ := parseCBAFlags("kit cba archive", args)
	if f.queue == "" {
		die("kit cba archive", fmt.Errorf("--queue <file.json|jsonl> is required"))
	}
	if f.out == "" {
		die("kit cba archive", fmt.Errorf("--out <dir> is required"))
	}
	// No default ceiling on purpose. This runs unattended against a public API we do
	// not own; how much to pull is a decision a human states, not one a tool assumes.
	if f.max <= 0 {
		die("kit cba archive", fmt.Errorf("--max N is required (hard ceiling on fetches this run)"))
	}

	queue, err := cba.LoadQueue(f.queue)
	if err != nil {
		die("kit cba archive", err)
	}
	manifest := f.manifest
	if manifest == "" {
		manifest = filepath.Join(f.out, "archive_manifest.jsonl")
	}
	seen, err := cba.LoadManifest(manifest)
	if err != nil {
		die("kit cba archive", err)
	}

	delay := cba.DefaultArchiveDelay
	if f.delayMS > 0 {
		delay = time.Duration(f.delayMS) * time.Millisecond
	}

	fmt.Fprintf(os.Stderr,
		"archive: %d queued, %d already known, max %d this run, %v between requests\n",
		len(queue), len(seen), f.max, delay)

	res, records := cba.Archive(queue, seen, aoems.NewClient(aoems.DefaultCacheDir()), cba.ArchiveOptions{
		OutDir:    f.out,
		RejectDir: f.rejectDir,
		Max:       f.max,
		Delay:     delay,
		Progress: func(rec cba.ArchiveRecord) {
			verdict := "REJECT"
			if rec.Admitted {
				verdict = "KEEP  "
			}
			fmt.Fprintf(os.Stderr, "  %s %-11s %s\n", verdict, rec.GameID, rec.Reason)
		},
	})

	// Persist before reporting: a crash after fetching but before recording would make
	// the next run re-request everything we just pulled.
	if err := cba.AppendManifest(manifest, records); err != nil {
		die("kit cba archive", err)
	}
	printJSON(res)
}

func runCBARegistry(args []string) {
	f, rest := parseCBAFlags("kit cba registry", args)
	if len(rest) == 0 {
		die("kit cba registry", fmt.Errorf("need at least one replay or directory"))
	}
	paths := collectReplays(rest)
	fmt.Fprintf(os.Stderr, "classifying %d replay(s)...\n", len(paths))

	done := 0
	reg := cba.BuildRegistryWithWorkers(paths, f.workers, func(path string, id cba.ScenarioIdentity) {
		done++
		if done%25 == 0 || done == len(paths) {
			fmt.Fprintf(os.Stderr, "  %d/%d classified\n", done, len(paths))
		}
	})

	if f.out != "" {
		fh, err := os.Create(f.out)
		if err != nil {
			die("kit cba registry", err)
		}
		if err := cba.WriteRegistry(fh, reg); err != nil {
			fh.Close()
			die("kit cba registry", err)
		}
		fh.Close()
		fmt.Fprintf(os.Stderr, "registry -> %s\n", f.out)
	}

	if !f.textOut {
		if f.out == "" {
			cba.WriteRegistry(os.Stdout, reg)
		} else {
			printJSON(reg.Summary)
		}
		return
	}

	s := reg.Summary
	fmt.Printf("\n=== CBA REQUIEM V292 REGISTRY ===\n")
	fmt.Printf("scanned %d | ladder %d | admitted %d | duplicates %d | too-few-players %d | rejected %d\n\n",
		s.Scanned, s.Ladder, s.Admitted, s.Duplicates, s.TooFew, s.Rejected)

	// Rejections grouped by reason -- a long tail of one-off reasons usually means the
	// gate is misconfigured rather than the corpus being dirty.
	byReason := map[string]int{}
	for _, e := range reg.Entries {
		if !e.Admit {
			byReason[e.Reason]++
		}
	}
	type kv struct {
		reason string
		n      int
	}
	var reasons []kv
	for r, n := range byReason {
		if strings.HasPrefix(r, "duplicate of ") {
			r = "duplicate of an already-admitted game"
		}
		reasons = append(reasons, kv{r, n})
	}
	merged := map[string]int{}
	for _, r := range reasons {
		merged[r.reason] += r.n
	}
	reasons = reasons[:0]
	for r, n := range merged {
		reasons = append(reasons, kv{r, n})
	}
	sort.Slice(reasons, func(i, j int) bool { return reasons[i].n > reasons[j].n })
	if len(reasons) > 0 {
		fmt.Println("not admitted:")
		for _, r := range reasons {
			fmt.Printf("  %4d  %s\n", r.n, truncate(r.reason, 68))
		}
	}
}

func runCBAAxes(args []string) {
	f, _ := parseCBAFlags("kit cba axes", args)
	requireCorpus("kit cba axes", f)
	corpus, err := cba.LoadCorpus(f.corpus)
	if err != nil {
		die("kit cba axes", err)
	}
	axes := cba.ComputeAxes(corpus, cba.AxesOptions{
		MinGames: f.minGames,
		Names:    loadCBANames(f.names),
	})
	if !f.textOut {
		printJSON(axes)
		return
	}
	minGames := f.minGames
	if minGames <= 0 {
		minGames = 3
	}
	fmt.Printf("scored %d players (>=%d games) from %d games\n\n",
		len(axes), minGames, len(corpus.Games))
	fmt.Printf("%-26s%4s%8s%8s%8s%7s  %s\n", "player", "gms", "eco", "combat", "tempo", "win%", "verdict")
	fmt.Println(strings.Repeat("-", 88))
	for _, p := range axes {
		name := p.Name
		if name == "" {
			name = fmt.Sprintf("#%d", p.ProfileID)
		}
		flag := ""
		if p.IsAnchor {
			flag = " ANCHOR"
		} else if p.IsStacker {
			flag = " STACKER"
		}
		fmt.Printf("%-26s%4d%8s%8s%8s%7d  %s%s\n",
			truncate(name, 26), p.Games,
			optStr(p.Eco, p.HasEco), optStr(p.Combat, p.HasCombat), optStr(p.Tempo, p.HasTempo),
			p.WinPct, p.Verdict, flag)
	}
}

func runCBAExport(args []string) {
	f, _ := parseCBAFlags("kit cba export", args)
	requireCorpus("kit cba export", f)
	corpus, err := cba.LoadCorpus(f.corpus)
	if err != nil {
		die("kit cba export", err)
	}
	names := loadCBANames(f.names)
	axes := cba.ComputeAxes(corpus, cba.AxesOptions{MinGames: f.minGames, Names: names})
	if !f.sqlOut {
		printJSON(map[string]any{"games": len(corpus.Games), "players": axes})
		return
	}
	if err := cba.ExportSQL(os.Stdout, corpus, axes, names); err != nil {
		die("kit cba export", err)
	}
}

func optStr(v float64, ok bool) string {
	if !ok {
		return "-"
	}
	return fmt.Sprintf("%.2f", v)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

func printCBAProgression(report *cba.ProgressionReport) {
	fmt.Printf("path: %s\n", report.Path)
	fmt.Printf("method: %s\n", report.Method)
	fmt.Printf("verification: %s\n", report.Verification)
	fmt.Printf("summary: production_events=%d research_events=%d players_with_production=%d players_with_research=%d feudal_flags=%d imperial_proxy_signals=%d\n",
		report.Summary.ProductionEvents, report.Summary.ResearchEvents, report.Summary.PlayersWithProduction, report.Summary.PlayersWithResearch, report.Summary.FeudalProductionFlags, report.Summary.ImperialProxySignals)
	for _, player := range report.Players {
		fmt.Printf("- %s name=%q civ=%d/%s first_production=%s imperial_proxy=%s feudal_flag=%t confidence=%s\n",
			player.Label, player.Name, player.CivID, player.CivName, player.FirstProductionTime, player.FirstImperialProxyAt, player.FeudalProductionFlag, player.Confidence)
	}
	for _, warning := range report.Warnings {
		fmt.Printf("warning: %s\n", warning)
	}
}

func printCBARazes(report *cba.RazeReport) {
	fmt.Printf("path: %s\n", report.Path)
	fmt.Printf("method: %s\n", report.Method)
	fmt.Printf("verification: %s\n", report.Verification)
	fmt.Printf("summary: building_target_events=%d pressured_targets=%d multi_player_targets=%d raze_candidates=%d set_play_candidates=%d\n",
		report.Summary.BuildingTargetEvents, report.Summary.PressuredTargets, report.Summary.MultiPlayerTargets, report.Summary.RazeCandidates, report.Summary.SetPlayCandidates)
	for _, player := range report.Players {
		fmt.Printf("- %s name=%q civ=%d/%s archetype=%s first_pressure=%s first_raze_candidate=%s set_finishes=%d confidence=%s\n",
			player.Label, player.Name, player.CivID, player.CivName, player.RazeArchetype, player.FirstPressureTime, player.FirstRazeCandidate, player.SetPlayFinishes, player.Confidence)
	}
	for _, warning := range report.Warnings {
		fmt.Printf("warning: %s\n", warning)
	}
}

func printCBARazePressure(report *cba.RazePressureReport) {
	fmt.Printf("path: %s\n", report.Path)
	fmt.Printf("method: %s\n", report.Method)
	fmt.Printf("verification: %s\n", report.Verification)
	fmt.Printf("summary: targets=%d primary=%d gates=%d castles=%d walls=%d other_buildings=%d pressure_events=%d nearby_loss_pulses=%d net_lost_nearby=%d players=%d\n",
		report.Summary.Targets, report.Summary.PrimaryTargets, report.Summary.GateTargets,
		report.Summary.CastleTargets, report.Summary.WallTargets, report.Summary.OtherBuildingTargets,
		report.Summary.PressureEvents, report.Summary.NearbyLossPulses,
		report.Summary.NetObjectsLostNearby, report.Summary.PlayersWithPressure)
	if len(report.Players) > 0 {
		fmt.Println("players:")
		for _, player := range report.Players {
			fmt.Printf("- %s name=%q pressure=%d primary_pressure=%d targets=%d primary_targets=%d nearby_loss_targets=%d net_lost_nearby=%d\n",
				player.PlayerLabel, player.PlayerName, player.PressureEvents, player.PrimaryPressureEvents,
				player.Targets, player.PrimaryTargets, player.NearbyLossTargets, player.NetObjectsLostNearby)
		}
	}
	if len(report.Targets) > 0 {
		fmt.Println("targets:")
		limit := len(report.Targets)
		if limit > 40 {
			limit = 40
		}
		for _, target := range report.Targets[:limit] {
			fmt.Printf("- target=%d kind=%s primary=%t owner=%s/%s unit=%d/%s xy=%.1f,%.1f pressure=%d first=%s last=%s nearby_loss_pulses=%d net_lost_nearby=%d confidence=%s",
				target.TargetID, target.Kind, target.Primary, target.OwnerLabel, target.OwnerName,
				target.UnitID, target.UnitName, target.X, target.Y, target.PressureEvents,
				target.FirstPressureTime, target.LastPressureTime, target.NearbyLossPulses,
				target.NetObjectsLostNearby, target.Confidence)
			if len(target.Attackers) > 0 {
				fmt.Print(" attackers=")
				for i, attacker := range target.Attackers {
					if i > 0 {
						fmt.Print(",")
					}
					fmt.Printf("P%d/%s:%d", attacker.PlayerID, attacker.PlayerName, attacker.Events)
				}
			}
			fmt.Println()
		}
		if len(report.Targets) > limit {
			fmt.Printf("- ... %d more targets\n", len(report.Targets)-limit)
		}
	}
	for _, warning := range report.Warnings {
		fmt.Printf("warning: %s\n", warning)
	}
}

func printCBAPhaseFacts(report *cba.PhaseFactsReport) {
	fmt.Printf("path: %s\n", report.Path)
	fmt.Printf("method: %s\n", report.Method)
	fmt.Printf("verification: %s\n", report.Verification)
	fmt.Printf("summary: triggers=%d castle_rungs=%d imperial_rungs=%d activators=%d groups=%d civs=%d defaults=%d raze_joined=%d lane_symmetry=%t unresolved_groups=%d multi_civ_groups=%d\n",
		report.Summary.TriggerCount, report.Summary.CastleRungs, report.Summary.ImperialRungs,
		report.Summary.ActivatorTriggers, report.Summary.ConditionGroups, report.Summary.CivFacts,
		report.Summary.DefaultInferred, report.Summary.RazeFactsJoined, report.Summary.LaneSymmetryOK,
		report.Summary.UnresolvedGroups, report.Summary.MultiCivGroups)
	if len(report.Civs) > 0 {
		fmt.Println("civ_facts:")
		for _, row := range report.Civs {
			fmt.Printf("- civ=%d/%s castle=%d imperial=%d razes=%d confidence=%s\n",
				row.CivID, row.CivName, row.CastleThreshold, row.ImperialThreshold, row.RazesToVillager, row.Confidence)
			if row.CastleEvidence != "" {
				fmt.Printf("  castle: %s\n", row.CastleEvidence)
			}
			if row.ImperialEvidence != "" {
				fmt.Printf("  imperial: %s\n", row.ImperialEvidence)
			}
			if row.RazeEvidence != "" {
				fmt.Printf("  raze: %s\n", row.RazeEvidence)
			}
		}
	}
	if len(report.Rungs) > 0 {
		fmt.Println("rungs:")
		limit := len(report.Rungs)
		if limit > 32 {
			limit = 32
		}
		for _, row := range report.Rungs[:limit] {
			fmt.Printf("- player=%d age=%s threshold=%d trigger=%d activators=%v groups=%d confidence=%s\n",
				row.Player, row.Age, row.ThresholdKills, row.RungTriggerID, row.ActivatorIDs, len(row.ConditionGroups), row.Confidence)
			for _, group := range row.ConditionGroups {
				fmt.Printf("  - activator=%d civs=%v confidence=%s terms=", group.ActivatorTriggerID, group.ResolvedCivNames, group.Confidence)
				for i, term := range group.Terms {
					if i > 0 {
						fmt.Print(",")
					}
					if term.Inverted {
						fmt.Print("!")
					}
					if term.CivName != "" {
						fmt.Printf("%d/%s", term.TechnologyID, term.CivName)
					} else {
						fmt.Printf("%d", term.TechnologyID)
					}
				}
				fmt.Println()
			}
		}
		if len(report.Rungs) > limit {
			fmt.Printf("- ... %d more rungs\n", len(report.Rungs)-limit)
		}
	}
	for _, warning := range report.Warnings {
		fmt.Printf("warning: %s\n", warning)
	}
}

func printCBAPhaseTimeline(report *cba.PhaseTimelineReport) {
	fmt.Printf("path: %s\n", report.Path)
	if report.DatPath != "" {
		fmt.Printf("dat: %s\n", report.DatPath)
	}
	fmt.Printf("method: %s\n", report.Method)
	fmt.Printf("verification: %s\n", report.Verification)
	fmt.Printf("summary: players=%d castle_observed=%d imperial_observed=%d phase_facts_civs=%d sync_deltas=%d duration=%s confidence=%s\n",
		report.Summary.Players, report.Summary.CastleObserved, report.Summary.ImperialObserved,
		report.Summary.PhaseFactsCivs, report.Summary.SyncDeltas, report.Summary.Duration, report.Summary.Confidence)
	if len(report.Players) > 0 {
		fmt.Println("players:")
		for _, row := range report.Players {
			fmt.Printf("- %s name=%q civ=%d/%s castle_kills=%d imperial_kills=%d razes_to_vill=%d early_spawn=%s confidence=%s\n",
				row.PlayerLabel, row.PlayerName, row.CivID, row.CivName, row.CastleThresholdKills,
				row.ImperialThresholdKills, row.RazesToVillager, replayUnitText(row.EarlySpawnUnitID, row.EarlySpawnUnitName),
				row.EarlySpawnConfidence)
			if row.Castle != nil {
				printCBAPhaseAnchor("  castle", row.Castle)
			}
			if row.Imperial != nil {
				printCBAPhaseAnchor("  imperial", row.Imperial)
			}
			for _, warning := range row.DetectionWarnings {
				fmt.Printf("  note: %s\n", warning)
			}
		}
	}
	for _, warning := range report.Warnings {
		fmt.Printf("warning: %s\n", warning)
	}
}

func printCBAPhaseAnchor(prefix string, anchor *cba.PhaseAnchor) {
	if anchor == nil {
		return
	}
	unit := ""
	if anchor.UnitID != 0 {
		unit = fmt.Sprintf(" unit=%s", replayUnitText(anchor.UnitID, anchor.UnitName))
	}
	fmt.Printf("%s: time=%s kill_anchor=%d signal=%s confidence=%s objects_delta=%d type_sum_delta=%d avg_type=%.2f%s evidence=%q\n",
		prefix, anchor.Time, anchor.KillAnchor, anchor.Signal, anchor.Confidence,
		anchor.ObjectCountDelta, anchor.UnitTypeSumDelta, anchor.AverageAddedTypeID, unit, anchor.Evidence)
}

func printCBAPerformance(report *cba.PerformanceReport) {
	fmt.Printf("path: %s\n", report.Path)
	fmt.Printf("method: %s\n", report.Method)
	fmt.Printf("verification: %s\n", report.Verification)
	fmt.Printf("summary: players=%d duration=%s winner_known=%t profile_ids=%d raze_candidates=%d set_plays=%d checksum_series_rows=%d\n",
		report.Summary.Players, report.Summary.Duration, report.Summary.WinnerKnown,
		report.Summary.RowsWithProfileID, report.Summary.RazeCandidates, report.Summary.SetPlayCandidates,
		report.Summary.ChecksumSeriesRows)
	for _, row := range report.Rows {
		fmt.Printf("- slot=%d profile=%d team=%s civ=%s won=%s apm=%s cg=%s switches=%s spatial=%s chat=%s taunts=%s own_razes=%s own_vills=%s own_first_raze=%s own_first_vill=%s team_razes=%s team_vills=%s team_first_raze=%s team_first_vill=%s prod=%s first_prod=%s defensive=%s viewlocks=%s dist=%s jumps=%s units_produced=%s units_lost=%s confidence=%s\n",
			row.Slot, row.ProfileID, row.Team, row.Civ,
			intPtrText(row.Won), floatPtrText(row.APM), intPtrText(row.ControlGroups), intPtrText(row.CGSwitches),
			intPtrText(row.Spatial), intPtrText(row.Chat), intPtrText(row.Taunts), intPtrText(row.OwnRazes),
			intPtrText(row.OwnVills), intPtrText(row.OwnFirstRazeS), intPtrText(row.OwnFirstVillS),
			intPtrText(row.TeamRazes), intPtrText(row.TeamVills), intPtrText(row.TeamFirstRazeS), intPtrText(row.TeamFirstVillS),
			intPtrText(row.ProdBuildings), intPtrText(row.FirstProdBuildS),
			intPtrText(row.DefensiveBuilds), intPtrText(row.ViewlockEvents), intPtrText(row.ViewlockDist),
			intPtrText(row.ViewlockJumps), intPtrText(row.UnitsProduced), intPtrText(row.UnitsLost), row.MetricConfidence)
		for _, warning := range row.AttributionWarnings {
			fmt.Printf("  note: %s\n", warning)
		}
	}
	if len(report.Events) > 0 {
		limit := len(report.Events)
		if limit > 24 {
			limit = 24
		}
		fmt.Printf("events (first %d of %d):\n", limit, len(report.Events))
		for _, event := range report.Events[:limit] {
			fmt.Printf("- %s %s %s count=%d unit=%d/%s building=%d/%s target=%d/%s confidence=%s\n",
				event.Time, event.PlayerLabel, event.Kind, event.Count, event.UnitID, event.UnitName,
				event.BuildingID, event.BuildingName, event.TargetID, event.TargetClass, event.Confidence)
		}
	}
	for _, warning := range report.Warnings {
		fmt.Printf("warning: %s\n", warning)
	}
}

func printCBADoctrine(report *cba.DoctrineReport) {
	fmt.Printf("path: %s\n", report.Path)
	fmt.Printf("method: %s\n", report.Method)
	fmt.Printf("verification: %s\n", report.Verification)
	fmt.Printf("summary: players=%d duration=%s winner_known=%t raze_candidates=%d set_plays=%d own_vill=%d team_vill=%d duties=%d kill_attr=%s raze_attr=%s phase_basis=%s\n",
		report.Summary.Players, report.Summary.Duration, report.Summary.WinnerKnown,
		report.Summary.RazeCandidates, report.Summary.SetPlayCandidates,
		report.Summary.PlayersWithOwnVill, report.Summary.PlayersWithTeamVill,
		report.Summary.PlayersWithDuty, report.Summary.DirectKillAttribution,
		report.Summary.DirectRazeAttribution, report.Summary.ReplayPhaseBoundaryBasis)
	for _, row := range report.Players {
		name := row.Name
		if name == "" {
			name = "unknown"
		}
		fmt.Printf("- slot=%d profile=%d name=%q team=%s civ=%s won=%s phase=%s duty=%q duty_confidence=%s own_vill=%s team_vill=%s own_razes=%s team_razes=%s first_pressure=%s first_raze=%s set_finishes=%d prod=%s first_prod=%s combat_proxy=%q confidence=%s\n",
			row.Slot, row.ProfileID, name, row.Team, row.Civ, intPtrText(row.Won),
			row.PhaseRead, row.Duty, row.DutyConfidence, intPtrText(row.OwnFirstVillS),
			intPtrText(row.TeamFirstVillS), intPtrText(row.OwnRazes), intPtrText(row.TeamRazes),
			blankText(row.FirstPressureTime), blankText(row.FirstRazeCandidate), row.SetPlayFinishes,
			intPtrText(row.ProdBuildings), intPtrText(row.FirstProdBuildS), row.CombatProxy, row.MetricConfidence)
		if row.ProgressionSignal != "" {
			fmt.Printf("  progression: %s\n", row.ProgressionSignal)
		}
		for _, warning := range row.AttributionWarnings {
			fmt.Printf("  note: %s\n", warning)
		}
	}
	for _, warning := range report.Warnings {
		fmt.Printf("warning: %s\n", warning)
	}
}

func blankText(value string) string {
	if value == "" {
		return "none"
	}
	return value
}

func intPtrText(value *int) string {
	if value == nil {
		return "null"
	}
	return fmt.Sprintf("%d", *value)
}

func floatPtrText(value *float64) string {
	if value == nil {
		return "null"
	}
	return fmt.Sprintf("%.2f", *value)
}

func floatPtrValue(value *float64) float64 {
	if value == nil {
		return 0
	}
	return *value
}

func intPtrValue(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}

func runMod(args []string) {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: kit mod <check|inspect> <moddir> [--status mod-status.json]")
		os.Exit(2)
	}
	switch args[0] {
	case "check", "inspect":
		opts := modpack.CheckOptions{}
		for i := 2; i < len(args); i++ {
			switch args[i] {
			case "--status":
				i++
				if i >= len(args) {
					die("kit mod", fmt.Errorf("--status needs a path"))
				}
				opts.StatusPath = args[i]
			default:
				die("kit mod", fmt.Errorf("unknown option %q", args[i]))
			}
		}
		report, err := modpack.CheckWithOptions(args[1], opts)
		if err != nil {
			die("kit mod", err)
		}
		printJSON(report)
		if !report.OK() {
			os.Exit(1)
		}
	default:
		fmt.Fprintln(os.Stderr, "usage: kit mod <check|inspect> <moddir> [--status mod-status.json]")
		os.Exit(2)
	}
}

func runSwatch(args []string) {
	if len(args) < 1 || args[0] != "patterns" {
		fmt.Fprintln(os.Stderr, "usage: kit swatch patterns <list|summary|validate|json>")
		os.Exit(2)
	}
	if len(args) == 1 {
		swatchSummary()
		return
	}
	switch args[1] {
	case "list":
		for _, name := range geom.PatternNames() {
			fmt.Println(name)
		}
	case "summary":
		swatchSummary()
	case "validate":
		if !swatchValidate() {
			os.Exit(1)
		}
	case "json":
		if len(args) != 3 {
			fmt.Fprintln(os.Stderr, "usage: kit swatch patterns json <name>")
			os.Exit(2)
		}
		frames, err := geom.Generate(args[2])
		if err != nil {
			die("kit swatch", err)
		}
		validation := geom.Validate(frames)
		if len(validation.Errors) != 0 {
			die("kit swatch", fmt.Errorf("%s failed validation: %v", args[2], validation.Errors))
		}
		printJSON(struct {
			Name   string       `json:"name"`
			Frames []geom.Frame `json:"frames"`
		}{Name: args[2], Frames: frames})
	default:
		fmt.Fprintln(os.Stderr, "usage: kit swatch patterns <list|summary|validate|json>")
		os.Exit(2)
	}
}

func swatchSummary() {
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

func swatchValidate() bool {
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

func printFactsCheck(ok bool, schemaVersion int, generated string, total int, shown int, verification string) {
	status := "OK"
	if !ok {
		status = "FAIL"
	}
	fmt.Printf("status: %s\n", status)
	fmt.Printf("verification: %s\n", verification)
	fmt.Printf("ledger: schema=%d generated=%s facts=%d shown=%d\n", schemaVersion, generated, total, shown)
}

func printFactsList(facts []enginefacts.Fact, total int, includeProvisional bool, domain string) {
	fmt.Printf("facts: shown=%d total=%d include_provisional=%t", len(facts), total, includeProvisional)
	if domain != "" {
		fmt.Printf(" domain=%s", domain)
	}
	fmt.Println()
	for _, fact := range facts {
		rule := ""
		if fact.LintRule != nil {
			rule = fmt.Sprintf(" rule=%s/%s/%s", fact.LintRule.Target, fact.LintRule.Predicate, fact.LintRule.Severity)
		}
		fmt.Printf("- %s [%s verified=%s] %s%s\n", fact.ID, fact.Tier, fact.VerifiedDate, fact.Statement, rule)
		fmt.Printf("  fixture: %s\n", fact.FixtureRef)
		if len(fact.Domains) > 0 {
			fmt.Printf("  domains: %s\n", strings.Join(fact.Domains, ","))
		}
	}
}

func printJSON(v any) {
	if err := jsonout.Print(v); err != nil {
		die("kit", err)
	}
}

func die(prefix string, err error) {
	fmt.Fprintf(os.Stderr, "%s: %v\n", prefix, err)
	os.Exit(1)
}

func usage() {
	fmt.Fprint(os.Stderr, `Usage:
  kit version
  kit doctor
  kit docs <lint> ...
  kit docs lint [folder] [--text|--json]
  kit inventory [folder]
  kit summary [folder]
  kit manifest
  kit credits [notice] [folder] [--text|--json]
  kit verify [folder] [--go-checks]
  kit pack [out.zip] [folder] [--profile full|handoff|sandbox|public]
  kit portable-check [folder] [--profile full|handoff|sandbox|public]
  kit project <inspect|snapshot|diff|lineage> ...
  kit project inspect [folder] [--limit N] [--text|--json]
  kit project snapshot [folder] --out project-snapshot.json [--text|--json]
  kit project diff <before-snapshot.json> <after-snapshot.json> [--limit N] [--text|--json]
  kit project lineage --input PATH --output PATH [--input PATH...] [--output PATH...] [--tool TEXT] [--note TEXT] [--manifest lineage.json] [--text|--json]
  kit recipe <list|show> ...
  kit recipe list [--domain scenario|dat|all] [--text|--json]
  kit recipe show <name> [--recipe-only] [--text|--json]
  kit recipe export <out-dir> [--domain scenario|dat|all] [--force] [--text|--json]
  kit release-check <scenario.aoe2scenario> [--previous old.aoe2scenario] [--mod moddir] [--status mod-status.json] [--pack out.zip] [--text]
  kit verify-run <replay.aoe2record|replay.zip> --contract contract.json [--text]
  kit ci <init|check> ...
  kit campaign <generate|extract> ...
  kit roadmap <crud|rwd|dark> [--domain scenario|dat|replay|all] [--text|--json]
  kit roadmap crud [--domain scenario|dat|replay|all] [--text|--json]
  kit roadmap rwd [--domain scenario|dat|replay|all] [--text|--json]
  kit roadmap dark [--domain scenario|dat|replay|all] [--text|--json]
  kit rpg shop-demo [--out-dir DIR] [--name TEXT] [--timestamp UNIX] [--text|--json]
  kit rpg custom-store [--out-dir DIR] [--name TEXT] [--unit N] [--instances N] [--players N] [--timestamp UNIX] [--xs-check PATH] [--text|--json]
  kit font glyphs <font.ttf> [--chars TEXT] [--material N] [--rotation DEGREES] [--out FILE]
  kit geo trace-dem [--bbox minLat,minLon,maxLat,maxLon] [--center lat,lon] [--span-km N] [--grid N] [--landmask naturalearth|geojson] [--landcover worldcover|glc-fcs30d] [--roads osm-interstate|osm-motorway|osm-major|nhpn-interstate] [--out report.json] [--recipe-out recipe.json] [--scenario-out out.aoe2scenario]
  kit identify <file.aoe2scenario|file.aoe2record> [--registry known_scenarios.json] [--ai-registry known_ais.json] [--register NAME] [--family FAMILY] [--description TEXT]
  kit collect <folder> [--registry known_scenarios.json] [--dry-run]
  kit player stats <profileId> [--match-type N] [--cache-dir DIR] [--force] [--text]
  kit cba replay <progression|razes|raze-pressure|phases|perf|doctrine> <file.aoe2record|replay.zip> [--dat <empires2_x2_p1.dat>] [--text|--json]
  kit cba balance [--text|--json]
  kit cba sidechannels <file.aoe2record|replay.zip> [--text]
  kit cba phase-facts <file.aoe2record|replay.zip> --dat <empires2_x2_p1.dat> [--text]
  kit cba trigger-razes <file.aoe2record|replay.zip> [--dat <empires2_x2_p1.dat>] [--text]
  kit cba trigger-spawns <file.aoe2record|replay.zip> [--dat <empires2_x2_p1.dat>] [--text]
  kit cba registry <replay|dir>... [--out <file.json>] [--text]
  kit cba ingest --corpus <file.jsonl> (--registry <file.json>|--unverified <replay|dir>...) [--force] [--limit N]
  kit cba archive --queue <q.json> --out <dir> --max N [--delay-ms MS] [--manifest f.jsonl]
  kit cba axes --corpus <file.jsonl> [--min-games N] [--names <file.json>] [--text]
  kit cba export --corpus <file.jsonl> [--names <file.json>] [--sql]
  kit facts <list|check> [--include-provisional] [--domain DOMAIN] [--text]
  kit replay <info|summary|graph|triggers|trigger-neighborhood|diff-triggers|coverage|opaque-spans|frontier|corpus|opaque-clusters|ai-manifest|effective-data|effective-units|datamod-check|fetch|diff-state|scan-value|header-anchors|objects|object-state|object-shapes|spawns|lifecycle|postgame|camera|sync|sync-log|checksum-phase|checksum-probe|combat|deaths|player-series|playtest|carrier|sidecar-sync|xs-telemetry|events|actions|player-events|chat|feedback|story|player-profile|inbox|telemetry|issues|unknowns> <path>
  kit replay info <file.aoe2record|zip> [--text|--json] [--tiles] [--full]
  kit replay fetch <gameId> --profile <profileId> [--out dir] [--force] [--text]
  kit replay fetch <gameId> --all-povs --profile <seedProfileId> [--out dir] [--force] [--text]
  kit replay fetch <gameId> --all-povs --from-roster <replay.aoe2record|zip> [--out dir] [--force] [--text]
  kit replay coverage <file.aoe2record> [--text|--json] [--fail-on-opaque]
  kit replay opaque-spans <file.aoe2record> [--space inflated_header|body] [--min-bytes N] [--limit N] [--text]
  kit replay frontier <file.aoe2record> [--min-duplicate-bytes N] [--limit N] [--text|--json]
  kit replay corpus <folder-or-replay> [--flat] [--no-zip] [--unknown-samples N] [--text]
  kit replay opaque-clusters <folder-or-replay> [--space inflated_header|body] [--min-bytes N] [--limit N] [--samples N] [--flat] [--no-zip] [--text]
  kit replay ai-manifest <file.aoe2record> [--promisory-root DIR] [--text]
  kit replay effective-data <file.aoe2record> [--dat empires2_x2_p1.dat] [--player N] [--tail-offset N] [--count N] [--limit N] [--known-template SHA256] [--all] [--text|--tree]
  kit replay effective-units <file.aoe2record> --dat empires2_x2_p1.dat [--player N] [--limit N] [--include-unchanged] [--all-types] [--include-duplicates] [--text]
  kit replay datamod-check <file.aoe2record> [--baseline-replay baseline.aoe2record] [--dat empires2_x2_p1.dat] [--known-template SHA256] [--tail-offset N] [--count N] [--limit N] [--text]
  kit replay triggers <file.aoe2record|zip> [--grep TEXT] [--effect ID_OR_NAME] [--condition ID_OR_NAME] [--player N] [--variable N] [--unit-const ID] [--limit N] [--text]
  kit replay trigger-neighborhood <file.aoe2record|zip> --trigger N [--depth N] [--limit N] [--text]
  kit replay diff-triggers <before.aoe2record|zip> <after.aoe2record|zip> [--limit N] [--text]
  kit replay diff-state <before.aoe2record> <after.aoe2record> [--focus all|opaque] [--limit N] [--text]
  kit replay scan-value <file.aoe2record> [--value N] [--width 8|16|32|64] [--hex BYTES] [--string TEXT] [--space inflated_header|body] [--limit N] [--text]
  kit replay header-anchors <file.aoe2record> [--limit N] [--text|--json]
  kit replay objects <file.aoe2record> [--referenced] [--limit N] [--text]
  kit replay object-state <file.aoe2record> [--referenced] [--class CLASS] [--limit N] [--text]
  kit replay object-shapes <file.aoe2record> [--referenced] [--limit N] [--text]
  kit replay spawns <file.aoe2record> [--text]
  kit replay lifecycle <file.aoe2record> [--text]
  kit replay postgame <file.aoe2record> [--text]
  kit replay sync <file.aoe2record> [--checksums] [--raw-words] [--limit N] [--text|--json]
  kit replay health <file.aoe2record> [--window 5m] [--text|--json] [--out path]
  kit replay sync-log <p0-sync.txt> [--replay file.aoe2record] [--limit N] [--text|--json]
  kit replay checksum-phase <file.aoe2record> [--word N|--all-words] [--player N] [--text|--json]
  kit replay checksum-probe <file.aoe2record> [--preset v6-playground-helper|v7-checksum-calibration] [--player N] [--object-count-delta N] [--unit-type-sum-delta N] [--object-id-sum-delta N] [--state-sum-delta N] [--carry-sum-delta N] [--digest-delta N] [--limit N] [--text|--json]
  kit replay combat <file.aoe2record> [--window-ms N] [--min-net-loss N] [--limit N] [--sort loss|time] [--text|--json]
  kit replay deaths <file.aoe2record> [--limit N] [--text|--json]
  kit replay player-series <file.aoe2record> [--player N] [--changes-only] [--limit N] [--text|--json]
  kit replay playtest <file.aoe2record> [--window N] [--include-computers] [--text|--json]
  kit replay carrier <file.aoe2record> --ledger expected.json [--text|--json]
  kit replay sidecar-sync <file.aoe2record> --xsdat run.xsdat (--ledger expected.json|--schema rtv12|rtv13|rtv14|rtv141|rtv142|rtv15|rtv16|rtv17|a2ksem2|customstore|engine-harness) [--player N] [--window-ms N] [--fail-on-mismatch] [--text|--json]
  kit replay xs-telemetry <file.aoe2record> [--text]
  kit replay events <file.aoe2record> [--all] [--system] [--untyped-actions] [--objects] [--raw] [--context context.json] [--text]
  kit replay actions <file.aoe2record> [--action-id N] [--limit N] [--objects] [--raw] [--text|--json]
  kit replay player-events <file.aoe2record> [--player N] [--type TYPE] [--action-id N] [--phase PHASE] [--from TIME] [--to TIME] [--limit N] [--objects] [--raw] [--text]
  kit replay chat <file.aoe2record> [--text|--json]
  kit replay feedback <file.aoe2record> [--text]
  kit replay story <file.aoe2record> [--context context.json] [--text|--brief]
  kit replay player-profile <file.aoe2record> [--context context.json] [--window-sec N] [--dead-gap-sec N] [--cell-size N] [--include-events] [--text|--json]
  kit replay inbox <folder> [--context context.json] [--text]
  kit replay telemetry <file-or-folder> [--context context.json] [--schema telemetry.json] [--text]
  kit replay issues <file.aoe2record> [--context context.json] [--text]
  kit replay unknowns <folder> [--text]
  kit ci init [scenario.aoe2scenario] [--root DIR] [--overwrite] [--text]
  kit ci check [aoe2kit-ci.json] [--text]
  kit campaign extract <campaign.aoe2campaign> [--scenario N | --name STEM] [--out-dir DIR] [--text|--json]
  kit campaign generate <campaign.json> [--out-dir DIR] [--prefix A2KCampaign] [--text]
  kit ai fingerprint <file-or-folder> [--registry known_ais.json] [--register NAME] [--signature AI_FILES_SIGNATURE]
  kit ai lint <file-or-folder> [--text]
  kit ai diff <before-file-or-folder> <after-file-or-folder> [--text]
  kit xs bridge <variables.json> [--out-dir DIR]
  kit xs shims <file-or-dir>... [--out recipe.json] [--prefix TEXT] [--enabled] [--include-params]
  kit xs datagen <arrays.json> [--out module.xs]
  kit xs harness --out-dir DIR [--batch safe|six-arm|trig-long|attr-sweep|long-v2|long-v3] [--name TEXT] [--timestamp UNIX]
  kit xs inspect <file.xsdat> [--types string,int,...] [--text]
  kit xsdat decode <file.xsdat> [--types string,int,...] [--ledger expected.json] [--schema rtv12|rtv13|rtv14|rtv141|rtv142|rtv15|rtv16|rtv17|a2ksem2|customstore|engine-harness|long-v3] [--text]
  kit scen blank <out.aoe2scenario> [--size N] [--width N] [--height N] [--players N] [--human-slots N] [--timestamp UNIX] [--no-starters] [--dummy-unit UNIT_ID] [--dummy-origin X,Y] [--dummy-spacing N] [--no-conquest] [--closeout-seconds N] [--closeout-player N] [--keep-seed-triggers] [--text]
    Barracks starters are included by default; --no-starters opts out.
  kit scen <blank|info|check|describe|settings|field|strings|refs|delete-plan|delete|disconnect|effects|diff-effects|glossary|idioms|analyze|triggers|units|map|terrain|palette-usage|regions|coverage|verify|lint|xs|deploycheck|diff|diff-triggers|bytediff|write-check|dump-body|plan|patch|shop-catalog|import-triggers|smoke|smoke-recipe> <file.aoe2scenario>
  kit scen import-triggers <src.aoe2scenario> <dst.aoe2scenario> [--select names/ids/ranges] [--prefix PREFIX] --out <file.aoe2scenario>
  kit scen coverage <file.aoe2scenario> [--text|--json] | kit scen coverage --root DIR [--root DIR ...] [--text|--json] [--strict]
  kit scen describe <file.aoe2scenario> [--section triggers|units|map|players|victory|messages|disables|ai|resources|all] [--full] [--json] [--no-path|--quiet-header]
  kit scen settings <file.aoe2scenario> [--text|--json]
  kit scen field <path> <file.aoe2scenario|--root DIR> [--strict]
  kit scen strings <file.aoe2scenario>
  kit scen refs <file.aoe2scenario> [--kind unit|trigger|variable|string|all] [--id N] [--text]
  kit scen delete-plan <file.aoe2scenario> <unit|trigger|variable|string> <id> [--text]
  kit scen delete-plan <file.aoe2scenario> unit-caption <caption> [--player N] [--text]
  kit scen delete-plan <file.aoe2scenario> unit-caption-prefix <prefix> [--player N] [--text]
  kit scen delete-plan <file.aoe2scenario> unit-type <unit_const> [--player N] [--text]
  kit scen delete-plan <file.aoe2scenario> units-player <player> [--text]
  kit scen delete-plan <file.aoe2scenario> trigger-name <name> [--text]
  kit scen delete-plan <file.aoe2scenario> trigger-prefix <prefix> [--text]
  kit scen delete-plan <file.aoe2scenario> variable-name <name> [--text]
  kit scen delete-plan <file.aoe2scenario> variable-prefix <prefix> [--text]
  kit scen delete-plan <file.aoe2scenario> string-text <exact_text> [--text]
  kit scen delete-plan <file.aoe2scenario> string-prefix <prefix> [--text]
  kit scen delete-plan <file.aoe2scenario> units-area <x1,y1,x2,y2> [--player N] [--unit N] [--text]
  kit scen delete-plan <file.aoe2scenario> <effect|condition> <trigger_index>:<child_index> [--text]
  kit scen delete <in.aoe2scenario> <out.aoe2scenario> <unit|trigger|variable|string> <id>
  kit scen delete <in.aoe2scenario> <out.aoe2scenario> unit-caption <caption> [--player N]
  kit scen delete <in.aoe2scenario> <out.aoe2scenario> unit-caption-prefix <prefix> [--player N]
  kit scen delete <in.aoe2scenario> <out.aoe2scenario> unit-type <unit_const> [--player N]
  kit scen delete <in.aoe2scenario> <out.aoe2scenario> units-player <player>
  kit scen delete <in.aoe2scenario> <out.aoe2scenario> trigger-name <name>
  kit scen delete <in.aoe2scenario> <out.aoe2scenario> trigger-prefix <prefix>
  kit scen delete <in.aoe2scenario> <out.aoe2scenario> variable-name <name>
  kit scen delete <in.aoe2scenario> <out.aoe2scenario> variable-prefix <prefix>
  kit scen delete <in.aoe2scenario> <out.aoe2scenario> string-text <exact_text>
  kit scen delete <in.aoe2scenario> <out.aoe2scenario> string-prefix <prefix>
  kit scen delete <in.aoe2scenario> <out.aoe2scenario> units-area <x1,y1,x2,y2> [--player N] [--unit N]
  kit scen delete <in.aoe2scenario> <out.aoe2scenario> <effect|condition> <trigger_index>:<child_index>
  kit scen disconnect <in.aoe2scenario> <out.aoe2scenario> <trigger|unit|variable|string> <id>
  kit scen disconnect <in.aoe2scenario> <out.aoe2scenario> unit-caption <caption> [--player N]
  kit scen disconnect <in.aoe2scenario> <out.aoe2scenario> unit-caption-prefix <prefix> [--player N]
  kit scen disconnect <in.aoe2scenario> <out.aoe2scenario> unit-type <unit_const> [--player N]
  kit scen disconnect <in.aoe2scenario> <out.aoe2scenario> units-player <player>
  kit scen disconnect <in.aoe2scenario> <out.aoe2scenario> trigger-name <name>
  kit scen disconnect <in.aoe2scenario> <out.aoe2scenario> trigger-prefix <prefix>
  kit scen disconnect <in.aoe2scenario> <out.aoe2scenario> variable-name <name>
  kit scen disconnect <in.aoe2scenario> <out.aoe2scenario> variable-prefix <prefix>
  kit scen disconnect <in.aoe2scenario> <out.aoe2scenario> string-text <exact_text>
  kit scen disconnect <in.aoe2scenario> <out.aoe2scenario> string-prefix <prefix>
  kit scen disconnect <in.aoe2scenario> <out.aoe2scenario> units-area <x1,y1,x2,y2> [--player N] [--unit N]
  kit scen effects <file.aoe2scenario> [--raw|--census|--where EFFECT] [--trigger N] [--limit N] [--max-mb N] [--text]
  kit scen diff-effects <before.aoe2scenario> <after.aoe2scenario> --where EFFECT [--trigger N] [--limit N] [--max-mb N] [--raw] [--text|--json]
  kit scen triggers <file.aoe2scenario> [--grep TEXT] [--effect ID_OR_NAME] [--condition ID_OR_NAME] [--message TEXT] [--unit-ref ID] [--unit-type ID] [--player N] [--variable N] [--area x1,y1,x2,y2] [--limit N] [--max-mb N] [--text|--json]
  kit scen trigger-neighborhood <file.aoe2scenario> (--trigger N|--unit-ref ID|--variable N) [--depth N] [--limit N] [--text]
  kit scen audit-player-coverage <file.aoe2scenario> [--players 1,2,3,4] [--text]
  kit scen mechanic <file.aoe2scenario> --kind garrison-token|transform-toggle|teleport-transition|refresh-cycle [--grep TEXT] [--text]
  kit scen trigger-flow <file.aoe2scenario> (--grep TEXT|--unit-ref ID|--variable N) [--limit N] [--text]
  kit scen glossary <file.aoe2scenario> [--limit N] [--max-mb N] [--text]
  kit scen idioms <file.aoe2scenario> [--text]
  kit scen analyze <file.aoe2scenario>
  kit scen units <file.aoe2scenario> [--named]
  kit scen lint <file.aoe2scenario> [--text] [--include-provisional] [--load-safety] [--semantic]
  kit scen xs <file.aoe2scenario> [--deploy-tree dir] [--text|--json]
  kit scen xs attach <in.aoe2scenario> <out.aoe2scenario> --xs file.xs [--name Entry.xs] [--also-carrier] [--title TITLE] [--trigger-name NAME] [--text|--json]
  kit scen xs deploy <file.aoe2scenario> <deploy-tree> [--name Entry.xs] [--carrier N|--trigger N] [--force] [--check] [--text|--json]
  kit scen xs embed <in.aoe2scenario> <out.aoe2scenario> --xs file.xs [--title TITLE] [--trigger-name NAME] [--replace-trigger N] [--append] [--keep-attachment] [--text|--json]
  kit scen xs extract <file.aoe2scenario> --out file.xs [--carrier N|--trigger N] [--force] [--text|--json]
  kit scen xs compare <file.aoe2scenario> <file.xs> [--carrier N|--trigger N] [--text|--json]
  kit scen deploycheck <file.aoe2scenario> <deploy-tree> [--text] [--include-provisional]
  kit scen diff <before.aoe2scenario> <after.aoe2scenario> [--all-fields] [--text|--json]
  kit scen diff-triggers <before.aoe2scenario> <after.aoe2scenario> [--limit N] [--text|--json]
  kit scen bytediff <before.aoe2scenario> <after.aoe2scenario> [--text|--json] [--include-noise]
  kit scen write-check <before.aoe2scenario> <after.aoe2scenario> [--text|--json] [--include-provisional]
  kit scen dump-body --inflated <file.aoe2scenario>
  kit scen plan <in.aoe2scenario> --recipe recipe.json
  kit scen patch <in.aoe2scenario> <out.aoe2scenario> --recipe recipe.json [--text|--json]
  kit scen shop-catalog <catalog.json> --out recipe.json [--text|--json]
  kit scen smoke-recipe [--x N] [--y N] [--player N] [--unit N] [--terrain N] [--elevation N] [--layer N]
  kit scen smoke <in.aoe2scenario> <out.aoe2scenario> [--x N] [--y N] [--player N] [--unit N] [--terrain N] [--elevation N] [--layer N]
  kit scen terrain <file.aoe2scenario> [--id N] [--tiles] [--text|--json]
  kit scen palette-usage <file.aoe2scenario> --dat <empires*.dat> [--json|--text]
  kit dat <info|check|roundtrip|codec-plan|codec-patch|delete-plan|delete|plan|patch|...> <empires*.dat>
  kit dat facts <effect-types|attributes|resources|task-types|tech-modifiers> [--text|--json]
  kit dat facts effect-command <id> [--text|--json]
  kit dat diff <base.dat> <mod.dat> [--section NAME|all] [--limit N|--all] [--text|--json]
  kit dat codec-plan <in.dat> --recipe recipe.json
  kit dat codec-patch <in.dat> <out.dat> --recipe recipe.json
  kit dat semantics-pack <in.dat> <out-dir> [--feature full|local-building-effects|class-scope-effects]
  kit dat semantics-readback <expected.json> --dat empires*.dat --scenario file.aoe2scenario --replay file.aoe2record [--out report.md] [--text]
  kit dat unit-headers <empires*.dat> [--limit N|--all] [--json]
  kit dat unit-header <empires*.dat> <header_id>
  kit dat civs <empires*.dat>
  kit dat civ-patch <in.dat> <out.dat> <civ_id> [--name TEXT] [--tech-tree-id N] [--team-bonus-id N] [--icon-set N] [--resource INDEX,VALUE]
  kit dat spans <empires*.dat>
  kit dat graphics <empires*.dat> [--limit N|--all] [--particle] [--name-contains TEXT] [--spans|--raw]
  kit dat graphic <empires*.dat> <graphic_id> [--spans|--raw]
  kit dat graphic-create <in.dat> <out.dat> --from N [graphic scalar flags]
  kit dat graphic-patch <in.dat> <out.dat> <graphic_id> [graphic scalar flags]
  kit dat graphic-delete <in.dat> <out.dat> <graphic_id>
  kit dat palette <empires*.dat> [--id N] [--min-variants N] [--json|--text]
  kit dat effects <empires*.dat> [--limit N|--all] [--command-type N] [--operand N]
  kit dat effect <empires*.dat> <effect_id>
  kit dat effect-explain <empires*.dat> <effect_id> [--text|--json]
  kit dat effect-create <in.dat> <out.dat> --name TEXT [--from-effect N] [--command type,a,b,c,d|json] [--append-command type,a,b,c,d|json] [--remove-command N]
  kit dat effect-patch <in.dat> <out.dat> <effect_id> [--name TEXT] [--clear-commands] [--command type,a,b,c,d|json] [--append-command type,a,b,c,d|json] [--remove-command N]
  kit dat effect-disable <in.dat> <out.dat> <effect_id>
  kit dat effect-delete <in.dat> <out.dat> <effect_id>
  kit dat effect-command-delete <in.dat> <out.dat> <effect_id> <command_index>
  kit dat command-matrix <empires*.dat> [--limit examples_per_type|--all] [--command-type N] [--operand N] [--reference-kind unit|tech] [--reference-field FIELD] [--typed-only|--typed|--unknown-only|--unknown|--candidate-only|--candidate] [--text|--json]
  kit dat techs <empires*.dat> [--limit N|--all] [--name-contains TEXT]
  kit dat tech <empires*.dat> <tech_id>
  kit dat tech-explain <empires*.dat> <tech_id> [--text|--json]
  kit dat tech-create <in.dat> <out.dat> --from N [tech-flags]
  kit dat tech-patch <in.dat> <out.dat> <tech_id> [tech-flags]
  kit dat tech-delete <in.dat> <out.dat> <tech_id>
  kit dat abilities <empires*.dat> [--limit N|--all] [--name-contains TEXT] [--effect-id N] [--command-type N] [--operand N]
  kit dat ability <empires*.dat> <tech_id>
  kit dat ability-create <in.dat> <out.dat> --from-tech N --name TEXT [--effect-name TEXT] [--from-effect N] [--command type,a,b,c,d|json] [--append-command type,a,b,c,d|json] [tech flags]
  kit dat ability-patch <in.dat> <out.dat> <tech_id> [--name TEXT] [--effect-name TEXT] [--clear-commands] [--command type,a,b,c,d|json] [--append-command type,a,b,c,d|json] [--remove-command N] [tech flags]
  kit dat ability-disable <in.dat> <out.dat> <tech_id>
  kit dat ability-delete <in.dat> <out.dat> <tech_id>
  kit dat tech-tree <empires*.dat> [--full]
  kit dat tech-tree-connection-create <in.dat> <out.dat> <building|unit|research> <from_index> [connection flags]
  kit dat tech-tree-connection-patch <in.dat> <out.dat> <building|unit|research> <index> [connection flags]
  kit dat tech-tree-connection-delete <in.dat> <out.dat> <building|unit|research> <index>
  kit dat units <empires*.dat> [--civ N] [--id N] [--name TEXT|--name-contains TEXT] [--class building|unit|creatable] [--limit N|--all] [--json]
  kit dat semantic-priors <empires*.dat> [--text|--json]
  kit dat sql <empires*.dat> <out.db>
  kit dat unit <empires*.dat> <civ_id> <unit_id>
  kit dat unit-create <in.dat> <out.dat> --from-civ N --from-unit N (--civ N|--all-civs) [unit patch flags]
  kit dat unit-delete <in.dat> <out.dat> <unit_id> (--civ N|--all-civs)
  kit dat unit-child-delete <in.dat> <out.dat> <damage-graphic|attack|armour|train-location|drop-site|task> <civ_id> <unit_id> <row_index>
  kit dat availability <empires*.dat> <unit_id> [--civ N|--all-civs]
  kit dat availability-set <in.dat> <out.dat> <unit_id> (--civ N|--all-civs) --enabled true|false|1|0
  kit dat terrain-restrictions <empires*.dat> [--limit N|--all] [--json]
  kit dat terrain-restriction-patch <in.dat> <out.dat> <restriction_id> --terrain TERRAIN_ID,PASSABILITY
  kit dat terrains <empires*.dat> [--limit N|--all] [--json]
  kit dat terrain <empires*.dat> <terrain_id> [--json]
  kit dat terrain-patch <in.dat> <out.dat> <terrain_id> [--name TEXT] [--name-2 TEXT] [--overlay-mask-name TEXT]
  kit dat sounds <empires*.dat> [--limit N|--all] [--json]
  kit dat sound <empires*.dat> <sound_id>
  kit dat sound-create <in.dat> <out.dat> --from N [sound-flags] [--item file,resource,probability,civ,icon_set]
  kit dat sound-patch <in.dat> <out.dat> <sound_id> [sound-flags] [--item INDEX,field=value...] [--remove-item N]
  kit dat sound-delete <in.dat> <out.dat> <sound_id>
  kit dat sound-item-delete <in.dat> <out.dat> <sound_id> <item_index>
  kit dat player-colours <empires*.dat>
  kit dat player-colour-create <in.dat> <out.dat> --from N [colour-flags]
  kit dat player-colour-patch <in.dat> <out.dat> <colour_id> [colour-flags]
  kit dat player-colour-delete <in.dat> <out.dat> <colour_id>
  kit dat random-maps <empires*.dat>
  kit dat sprite <file.sld> [--out dir] [--limit N] [--stats] [--text]
  kit dat patch-graphic <in.dat> <out.dat> <graphic_id> [graphic scalar flags]
  kit dat graphic-delta-delete <in.dat> <out.dat> <graphic_id> <row_index>
  kit dat graphic-angle-sound-delete <in.dat> <out.dat> <graphic_id> <row_index>
  kit dat patch-unit <in.dat> <out.dat> <civ_id> <unit_id> [scalar flags] [--type50-attack I,field=value] [--cost I,field=value] [--task I,field=value]
  kit dat unit-header-task-delete <in.dat> <out.dat> <unit_header_id> <task_index>
  kit dat refs <in.dat> <section> <id> [--class CLASS] [--confidence NAME] [--source-section SECTION] [--limit N|--all] [--text|--json]
  kit dat delete-plan <in.dat> <section|effect-command|sound-item|graphic-row-kind|unit-header-task|unit-row-kind> <target> [--civ N|--all-civs] [--text]
  kit dat delete <in.dat> <out.dat> <section|effect-command|sound-item|graphic-row-kind|unit-header-task|unit-row-kind> <target> [--civ N|--all-civs]
  kit dat disconnect <in.dat> <out.dat> <tech|unit> <id>
  kit dat plan <in.dat> --recipe recipe.json
  kit dat patch <in.dat> <out.dat> --recipe recipe.json
  kit gfx info <file.sld> [--text]
  kit gfx export <file.sld> [--out dir] [--limit N] [--text]
  kit fx new <name> --preset trail|explosion|aura|projectile-fire [--grid RxC] [descriptor overrides]
  kit fx bind --atlas file.png|--dds file.dds --grid RxC|--frames N --frame-size WxH --into resources/_common --dat empires*.dat --unit N --slot flying|standing|attack|dying --from graphic_id --name fx_name [--dry-run]
  kit fx lint <mod-or-resources/_common> [--dat empires*.dat] [--text]
  kit mod <check|inspect> <moddir> [--status mod-status.json]
  kit swatch patterns <list|summary|validate|json>
`)
}
