package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"aoe2kit/pkg/aoe2"
	"aoe2kit/pkg/datfile"
	"aoe2kit/pkg/jsonout"
)

func main() {
	if len(os.Args) < 3 {
		usage()
		os.Exit(2)
	}
	cmd, path := os.Args[1], os.Args[2]
	switch cmd {
	case "info", "inspect":
		info, err := aoe2.InspectFile(path)
		if err != nil {
			die(err)
		}
		if info.Kind != "dat" {
			die(fmt.Errorf("%s is %q, not dat", path, info.Kind))
		}
		if err := jsonout.Print(info); err != nil {
			die(err)
		}
	case "check":
		info, err := aoe2.InspectFile(path)
		if err != nil {
			die(err)
		}
		if info.Kind != "dat" {
			die(fmt.Errorf("%s is %q, not dat", path, info.Kind))
		}
		fmt.Printf("%s: %d bytes sha256=%s [OK]\n", info.Base, info.SizeBytes, info.SHA256)
	case "civs":
		idx, err := datfile.Open(path)
		if err != nil {
			die(err)
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
	case "spans":
		idx, err := datfile.Open(path)
		if err != nil {
			die(err)
		}
		printJSON(struct {
			Version       string         `json:"version"`
			InflatedBytes int            `json:"inflated_bytes"`
			SpanCount     int            `json:"span_count"`
			Spans         []datfile.Span `json:"spans"`
		}{Version: idx.Version, InflatedBytes: idx.Inflated, SpanCount: len(idx.Spans), Spans: idx.Spans})
	case "graphics":
		opts, err := parseGraphicsOptions(os.Args[3:], "dat graphics")
		if err != nil {
			die(err)
		}
		idx, err := datfile.Open(path)
		if err != nil {
			die(err)
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
		if len(os.Args) < 4 {
			usage()
			os.Exit(2)
		}
		spans := false
		for _, arg := range os.Args[4:] {
			switch arg {
			case "--spans", "--raw":
				spans = true
			default:
				die(fmt.Errorf("unknown dat graphic option %q", arg))
			}
		}
		id, err := strconv.Atoi(os.Args[3])
		if err != nil {
			die(fmt.Errorf("invalid graphic id %q: %w", os.Args[3], err))
		}
		idx, err := datfile.Open(path)
		if err != nil {
			die(err)
		}
		graphic, ok := idx.Graphic(id)
		if !ok {
			die(fmt.Errorf("graphic %d is absent or outside graphics table", id))
		}
		if spans {
			printJSON(graphic)
		} else {
			printJSON(graphic.Summary())
		}
	case "effects":
		limit, all, err := parseListOptions(os.Args[3:], "dat effects")
		if err != nil {
			die(err)
		}
		idx, err := datfile.Open(path)
		if err != nil {
			die(err)
		}
		effects := compactEffects(idx.Effects)
		truncated := false
		if !all && len(effects) > limit {
			effects = effects[:limit]
			truncated = true
		}
		printJSON(struct {
			Version       string              `json:"version"`
			EffectCount   int                 `json:"effect_count"`
			ReturnedCount int                 `json:"returned_count"`
			Truncated     bool                `json:"truncated"`
			Effects       []effectListSummary `json:"effects"`
		}{Version: idx.Version, EffectCount: len(idx.Effects), ReturnedCount: len(effects), Truncated: truncated, Effects: effects})
	case "effect":
		if len(os.Args) != 4 {
			usage()
			os.Exit(2)
		}
		id, err := strconv.Atoi(os.Args[3])
		if err != nil {
			die(fmt.Errorf("invalid effect id %q: %w", os.Args[3], err))
		}
		idx, err := datfile.Open(path)
		if err != nil {
			die(err)
		}
		effect, ok := idx.Effect(id)
		if !ok {
			die(fmt.Errorf("effect %d is outside effects table", id))
		}
		printJSON(effect)
	case "techs":
		limit, all, err := parseListOptions(os.Args[3:], "dat techs")
		if err != nil {
			die(err)
		}
		idx, err := datfile.Open(path)
		if err != nil {
			die(err)
		}
		techs := compactTechs(idx.Techs)
		truncated := false
		if !all && len(techs) > limit {
			techs = techs[:limit]
			truncated = true
		}
		printJSON(struct {
			Version       string            `json:"version"`
			TechCount     int               `json:"tech_count"`
			ReturnedCount int               `json:"returned_count"`
			Truncated     bool              `json:"truncated"`
			Techs         []techListSummary `json:"techs"`
		}{Version: idx.Version, TechCount: len(idx.Techs), ReturnedCount: len(techs), Truncated: truncated, Techs: techs})
	case "tech":
		if len(os.Args) != 4 {
			usage()
			os.Exit(2)
		}
		id, err := strconv.Atoi(os.Args[3])
		if err != nil {
			die(fmt.Errorf("invalid tech id %q: %w", os.Args[3], err))
		}
		idx, err := datfile.Open(path)
		if err != nil {
			die(err)
		}
		tech, ok := idx.Tech(id)
		if !ok {
			die(fmt.Errorf("tech %d is outside tech table", id))
		}
		printJSON(tech)
	case "tech-tree":
		full, err := parseTechTreeOptions(os.Args[3:])
		if err != nil {
			die(err)
		}
		idx, err := datfile.Open(path)
		if err != nil {
			die(err)
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
	case "units":
		limit, all, civFilter, err := parseUnitsOptions(os.Args[3:])
		if err != nil {
			die(err)
		}
		idx, err := datfile.Open(path)
		if err != nil {
			die(err)
		}
		allUnits := collectUnits(idx, civFilter)
		units := allUnits
		truncated := false
		if !all && len(units) > limit {
			units = units[:limit]
			truncated = true
		}
		printJSON(struct {
			Version       string            `json:"version"`
			CivCount      int               `json:"civ_count"`
			PresentCount  int               `json:"present_count"`
			ReturnedCount int               `json:"returned_count"`
			Truncated     bool              `json:"truncated"`
			Units         []unitListSummary `json:"units"`
		}{
			Version:       idx.Version,
			CivCount:      len(idx.Civs),
			PresentCount:  len(allUnits),
			ReturnedCount: len(units),
			Truncated:     truncated,
			Units:         compactUnits(units),
		})
	case "unit":
		if len(os.Args) != 5 {
			usage()
			os.Exit(2)
		}
		civID, err := strconv.Atoi(os.Args[3])
		if err != nil {
			die(fmt.Errorf("invalid civ id %q: %w", os.Args[3], err))
		}
		unitID, err := strconv.Atoi(os.Args[4])
		if err != nil {
			die(fmt.Errorf("invalid unit id %q: %w", os.Args[4], err))
		}
		idx, err := datfile.Open(path)
		if err != nil {
			die(err)
		}
		if civID < 0 || civID >= len(idx.Civs) {
			die(fmt.Errorf("civ %d is outside civ table", civID))
		}
		civ := idx.Civs[civID]
		if unitID < 0 || unitID >= len(civ.Units) {
			die(fmt.Errorf("unit %d is outside civ %d unit table", unitID, civID))
		}
		unit := civ.Units[unitID]
		if !unit.Present {
			die(fmt.Errorf("unit %d in civ %d is absent", unitID, civID))
		}
		printJSON(unit)
	case "patch-graphic":
		if len(os.Args) < 5 {
			usage()
			os.Exit(2)
		}
		id, err := strconv.Atoi(os.Args[4])
		if err != nil {
			die(fmt.Errorf("invalid graphic id %q: %w", os.Args[4], err))
		}
		patch, err := parseGraphicPatchOptions(os.Args[5:])
		if err != nil {
			die(err)
		}
		report, err := datfile.PatchGraphicFile(path, os.Args[3], id, patch)
		if err != nil {
			die(err)
		}
		printJSON(report)
	case "patch-unit":
		if len(os.Args) < 6 {
			usage()
			os.Exit(2)
		}
		civID, err := strconv.Atoi(os.Args[4])
		if err != nil {
			die(fmt.Errorf("invalid civ id %q: %w", os.Args[4], err))
		}
		unitID, err := strconv.Atoi(os.Args[5])
		if err != nil {
			die(fmt.Errorf("invalid unit id %q: %w", os.Args[5], err))
		}
		patch, err := parseUnitPatchOptions(os.Args[6:])
		if err != nil {
			die(err)
		}
		report, err := datfile.PatchUnitFile(path, os.Args[3], civID, unitID, patch)
		if err != nil {
			die(err)
		}
		printJSON(report)
	case "plan":
		if len(os.Args) != 5 || os.Args[3] != "--recipe" {
			usage()
			os.Exit(2)
		}
		recipe, err := readRecipe(os.Args[4])
		if err != nil {
			die(err)
		}
		report, err := datfile.PlanRecipeFile(path, recipe)
		if err != nil {
			die(err)
		}
		printJSON(report)
	case "patch":
		if len(os.Args) != 6 || os.Args[4] != "--recipe" {
			usage()
			os.Exit(2)
		}
		recipe, err := readRecipe(os.Args[5])
		if err != nil {
			die(err)
		}
		report, err := datfile.PatchRecipeFile(path, os.Args[3], recipe)
		if err != nil {
			die(err)
		}
		printJSON(report)
	default:
		usage()
		os.Exit(2)
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

func parseGraphicPatchOptions(args []string) (datfile.GraphicPatch, error) {
	var patch datfile.GraphicPatch
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--slp":
			if i+1 >= len(args) {
				return patch, fmt.Errorf("--slp needs a value")
			}
			i++
			value, err := parseInt32(args[i], "--slp")
			if err != nil {
				return patch, err
			}
			patch.SLP = &value
		case "--file-name":
			if i+1 >= len(args) {
				return patch, fmt.Errorf("--file-name needs a value")
			}
			i++
			value := args[i]
			patch.FileName = &value
		case "--particle-effect-name":
			if i+1 >= len(args) {
				return patch, fmt.Errorf("--particle-effect-name needs a value")
			}
			i++
			value := args[i]
			patch.ParticleEffectName = &value
		case "--frame-count":
			if i+1 >= len(args) {
				return patch, fmt.Errorf("--frame-count needs a value")
			}
			i++
			value, err := parseInt16(args[i], "--frame-count")
			if err != nil {
				return patch, err
			}
			if value < 0 {
				return patch, fmt.Errorf("--frame-count must be non-negative")
			}
			patch.FrameCount = &value
		default:
			return patch, fmt.Errorf("unknown patch option %q", args[i])
		}
	}
	if patch.FileName == nil && patch.ParticleEffectName == nil && patch.SLP == nil && patch.FrameCount == nil {
		return patch, fmt.Errorf("provide at least one patch option: --file-name, --particle-effect-name, --slp, or --frame-count")
	}
	return patch, nil
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
		default:
			return patch, fmt.Errorf("unknown unit patch option %q", args[i])
		}
	}
	if patch.Empty() {
		return patch, fmt.Errorf("provide at least one unit patch option")
	}
	return patch, nil
}

func parseNextInt16(args []string, i *int, name string) (int16, error) {
	if *i+1 >= len(args) {
		return 0, fmt.Errorf("%s needs a value", name)
	}
	*i = *i + 1
	return parseInt16(args[*i], name)
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
	value, err := strconv.ParseFloat(args[*i], 32)
	if err != nil {
		return 0, fmt.Errorf("invalid %s %q: %w", name, args[*i], err)
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

func parseInt16(raw, name string) (int16, error) {
	value, err := strconv.ParseInt(raw, 10, 16)
	if err != nil {
		return 0, fmt.Errorf("invalid %s %q: %w", name, raw, err)
	}
	return int16(value), nil
}

type graphicsOptions struct {
	Limit        int
	All          bool
	NameContains string
	ParticleOnly bool
	Spans        bool
}

func (o graphicsOptions) Filter() datfile.GraphicFilter {
	return datfile.GraphicFilter{
		NameContains: o.NameContains,
		ParticleOnly: o.ParticleOnly,
	}
}

func parseGraphicsOptions(args []string, label string) (graphicsOptions, error) {
	opts := graphicsOptions{Limit: 200}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--all":
			opts.All = true
		case "--limit":
			if i+1 >= len(args) {
				return graphicsOptions{}, fmt.Errorf("--limit needs a value")
			}
			i++
			limit, err := strconv.Atoi(args[i])
			if err != nil {
				return graphicsOptions{}, fmt.Errorf("invalid --limit %q: %w", args[i], err)
			}
			if limit < 0 {
				return graphicsOptions{}, fmt.Errorf("--limit must be non-negative")
			}
			opts.Limit = limit
			if limit == 0 {
				opts.All = true
			}
		case "--name-contains":
			if i+1 >= len(args) {
				return graphicsOptions{}, fmt.Errorf("--name-contains needs a value")
			}
			i++
			opts.NameContains = args[i]
		case "--particle":
			opts.ParticleOnly = true
		case "--spans", "--raw":
			opts.Spans = true
		default:
			return graphicsOptions{}, fmt.Errorf("unknown %s option %q", label, args[i])
		}
	}
	return opts, nil
}

func parseListOptions(args []string, label string) (limit int, all bool, err error) {
	limit = 200
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--all":
			all = true
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

func parseUnitsOptions(args []string) (limit int, all bool, civFilter *int, err error) {
	limit = 200
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--all":
			all = true
		case "--limit":
			if i+1 >= len(args) {
				return 0, false, nil, fmt.Errorf("--limit needs a value")
			}
			i++
			limit, err = strconv.Atoi(args[i])
			if err != nil {
				return 0, false, nil, fmt.Errorf("invalid --limit %q: %w", args[i], err)
			}
		case "--civ":
			if i+1 >= len(args) {
				return 0, false, nil, fmt.Errorf("--civ needs a value")
			}
			i++
			civ, err := strconv.Atoi(args[i])
			if err != nil {
				return 0, false, nil, fmt.Errorf("invalid --civ %q: %w", args[i], err)
			}
			civFilter = &civ
		default:
			return 0, false, nil, fmt.Errorf("unknown dat units option %q", args[i])
		}
	}
	if limit < 1 {
		return 0, false, nil, fmt.Errorf("--limit must be positive")
	}
	return limit, all, civFilter, nil
}

func parseTechTreeOptions(args []string) (bool, error) {
	full := false
	for _, arg := range args {
		switch arg {
		case "--full":
			full = true
		default:
			return false, fmt.Errorf("unknown dat tech-tree option %q", arg)
		}
	}
	return full, nil
}

func collectUnits(idx *datfile.Index, civFilter *int) []datfile.UnitSummary {
	var out []datfile.UnitSummary
	for _, civ := range idx.Civs {
		if civFilter != nil && civ.Index != *civFilter {
			continue
		}
		for _, unit := range civ.Units {
			if unit.Present {
				out = append(out, unit)
			}
		}
	}
	return out
}

type unitListSummary struct {
	CivIndex     int    `json:"civ_index"`
	Index        int    `json:"index"`
	Type         int    `json:"type"`
	ID           int16  `json:"id"`
	Name         string `json:"name"`
	HitPoints    int16  `json:"hit_points"`
	IconID       int16  `json:"icon_id"`
	Enabled      uint8  `json:"enabled"`
	HasType50    bool   `json:"has_type50"`
	HasCreatable bool   `json:"has_creatable"`
	RecordStart  int    `json:"record_start"`
	RecordEnd    int    `json:"record_end"`
}

type effectListSummary struct {
	Index        int    `json:"index"`
	Name         string `json:"name"`
	CommandCount int    `json:"command_count"`
	RecordStart  int    `json:"record_start"`
	RecordEnd    int    `json:"record_end"`
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

func compactEffects(effects []datfile.Effect) []effectListSummary {
	out := make([]effectListSummary, 0, len(effects))
	for _, effect := range effects {
		out = append(out, effectListSummary{
			Index:        effect.Index,
			Name:         effect.Name,
			CommandCount: len(effect.Commands),
			RecordStart:  effect.Span.Start,
			RecordEnd:    effect.Span.End,
		})
	}
	return out
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
			CivIndex:     unit.CivIndex,
			Index:        unit.Index,
			Type:         unit.Type,
			ID:           unit.ID,
			Name:         unit.Name,
			HitPoints:    unit.HitPoints,
			IconID:       unit.IconID,
			Enabled:      unit.Enabled,
			HasType50:    unit.Type50 != nil,
			HasCreatable: unit.Creatable != nil,
			RecordStart:  unit.RecordStart,
			RecordEnd:    unit.RecordEnd,
		})
	}
	return out
}

func printJSON(v any) {
	if err := jsonout.Print(v); err != nil {
		die(err)
	}
}

func die(err error) {
	fmt.Fprintf(os.Stderr, "dat: %v\n", err)
	os.Exit(1)
}

func usage() {
	fmt.Fprintf(os.Stderr, `Usage:
  dat info  <empires*.dat>
  dat check <empires*.dat>
  dat civs <empires*.dat>
  dat spans <empires*.dat>
  dat graphics <empires*.dat> [--limit N|--all] [--particle] [--name-contains TEXT] [--spans|--raw]
  dat graphic <empires*.dat> <graphic_id> [--spans|--raw]
  dat effects <empires*.dat> [--limit N|--all]
  dat effect <empires*.dat> <effect_id>
  dat techs <empires*.dat> [--limit N|--all]
  dat tech <empires*.dat> <tech_id>
  dat tech-tree <empires*.dat> [--full]
  dat units <empires*.dat> [--civ N] [--limit N|--all]
  dat unit <empires*.dat> <civ_id> <unit_id>
  dat patch-graphic <in.dat> <out.dat> <graphic_id> [--file-name S] [--particle-effect-name S] [--slp N] [--frame-count N]
  dat patch-unit <in.dat> <out.dat> <civ_id> <unit_id> [--class N] [--hit-points N] [--line-of-sight F] [--movement-type N] [unit patch flags]
  dat plan <in.dat> --recipe recipe.json
  dat patch <in.dat> <out.dat> --recipe recipe.json
`)
}
