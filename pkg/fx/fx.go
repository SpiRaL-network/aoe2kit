package fx

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	_ "image/png"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"aoe2kit/pkg/aoe2"
	"aoe2kit/pkg/datfile"
)

const Version = "aoe2kit.fx.v1"

type BindOptions struct {
	AtlasPath   string
	DDSPath     string
	Grid        Grid
	IntoCommon  string
	DatPath     string
	UnitID      int
	Slot        string
	FromGraphic int
	Name        string
	Preset      string
	ClearSLP    bool
	DryRun      bool
	Overrides   DescriptorOverrides
}

type DescriptorOverrides struct {
	Type         *string
	Duration     *float64
	AlphaStart   *float64
	AlphaEnd     *float64
	Scale        *float64
	ScaleStart   *float64
	ScaleEnd     *float64
	Rotation     *float64
	StopMode     *string
	IsFire       *bool
	DisplayInFog *bool
}

type Grid struct {
	Rows   int `json:"rows"`
	Cols   int `json:"cols"`
	Frames int `json:"frames"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

type BindReport struct {
	Version       string                 `json:"version"`
	OK            bool                   `json:"ok"`
	DryRun        bool                   `json:"dry_run,omitempty"`
	Verification  aoe2.VerificationClaim `json:"verification"`
	Name          string                 `json:"name"`
	Preset        string                 `json:"preset"`
	Grid          Grid                   `json:"grid"`
	Descriptor    string                 `json:"descriptor"`
	AtlasMeta     string                 `json:"atlas_meta"`
	AtlasImage    string                 `json:"atlas_image"`
	DatPath       string                 `json:"dat_path,omitempty"`
	UnitID        int                    `json:"unit_id,omitempty"`
	Slot          string                 `json:"slot,omitempty"`
	FromGraphic   int                    `json:"from_graphic,omitempty"`
	NewGraphicID  int                    `json:"new_graphic_id,omitempty"`
	UnitPatches   []UnitBindPatch        `json:"unit_patches,omitempty"`
	Checksums     map[string]string      `json:"checksums,omitempty"`
	DescriptorDoc map[string]any         `json:"descriptor_doc,omitempty"`
	Warnings      []string               `json:"warnings,omitempty"`
}

type UnitBindPatch struct {
	CivID  int `json:"civ_id"`
	UnitID int `json:"unit_id"`
	Before int `json:"before"`
	After  int `json:"after"`
}

type LintOptions struct {
	ModPath string
	DatPath string
}

type LintReport struct {
	Version      string                 `json:"version"`
	OK           bool                   `json:"ok"`
	Verification aoe2.VerificationClaim `json:"verification"`
	CommonRoot   string                 `json:"common_root"`
	DatPath      string                 `json:"dat_path,omitempty"`
	Descriptors  []DescriptorLint       `json:"descriptors"`
	Errors       []string               `json:"errors,omitempty"`
	Warnings     []string               `json:"warnings,omitempty"`
}

type DescriptorLint struct {
	Name            string   `json:"name"`
	Path            string   `json:"path"`
	AtlasFile       string   `json:"atlas_file,omitempty"`
	AtlasPath       string   `json:"atlas_path,omitempty"`
	AtlasMetaPath   string   `json:"atlas_meta_path,omitempty"`
	ImageFirst      int      `json:"image_first"`
	ImageCount      int      `json:"image_count"`
	AtlasFrames     int      `json:"atlas_frames,omitempty"`
	ReferencedByDAT bool     `json:"referenced_by_dat,omitempty"`
	DDS             *DDSInfo `json:"dds,omitempty"`
	Errors          []string `json:"errors,omitempty"`
	Warnings        []string `json:"warnings,omitempty"`
}

type DDSInfo struct {
	Path       string `json:"path"`
	Width      int    `json:"width"`
	Height     int    `json:"height"`
	FourCC     string `json:"fourcc"`
	DXT5       bool   `json:"dxt5"`
	HeaderSize uint32 `json:"header_size"`
}

func NewDescriptor(name, preset string, grid Grid, overrides DescriptorOverrides) (map[string]any, error) {
	if err := validateName(name); err != nil {
		return nil, err
	}
	if grid.Frames <= 0 {
		return nil, errors.New("grid frame count must be positive")
	}
	doc, err := presetDescriptor(preset)
	if err != nil {
		return nil, err
	}
	doc["AtlasFile"] = `textures\atlases\` + atlasBase(name) + ".dds"
	doc["ImageFirst"] = 0
	doc["ImageCount"] = grid.Frames
	applyOverrides(doc, overrides)
	return doc, nil
}

func Bind(options BindOptions) (BindReport, error) {
	report := BindReport{
		Version: Version,
		DryRun:  options.DryRun,
		Verification: aoe2.VerificationClaim{
			StructureVerified: true,
			EngineVerified:    false,
			Note:              "kit fx verifies file structure, DAT readback, and asset references; in-engine particle rendering remains the external oracle.",
		},
		Preset: options.Preset,
		Slot:   options.Slot,
		UnitID: options.UnitID,
	}
	if options.AtlasPath == "" && options.DDSPath == "" {
		return report, errors.New("provide --atlas PNG or --dds DDS")
	}
	if options.AtlasPath != "" && options.DDSPath != "" {
		return report, errors.New("provide only one of --atlas or --dds")
	}
	if options.IntoCommon == "" {
		return report, errors.New("--into needs a resources/_common directory")
	}
	if options.DatPath == "" {
		return report, errors.New("--dat needs an empires*.dat path")
	}
	if options.UnitID < 0 {
		return report, errors.New("--unit must be non-negative")
	}
	if options.FromGraphic < 0 {
		return report, errors.New("--from must be non-negative")
	}
	if options.Name == "" {
		source := options.DDSPath
		if source == "" {
			source = options.AtlasPath
		}
		options.Name = strings.TrimSuffix(filepath.Base(source), filepath.Ext(source))
	}
	if options.Preset == "" {
		options.Preset = "trail"
	}
	if err := validateName(options.Name); err != nil {
		return report, err
	}
	if err := validateSlot(options.Slot); err != nil {
		return report, err
	}

	commonRoot, err := filepath.Abs(options.IntoCommon)
	if err != nil {
		return report, err
	}
	particlesDir := filepath.Join(commonRoot, "particles")
	atlasDir := filepath.Join(particlesDir, "textures", "atlases")
	descriptorPath := filepath.Join(particlesDir, options.Name+".json")
	atlasMetaPath := filepath.Join(particlesDir, atlasBase(options.Name)+".json")
	atlasPath := filepath.Join(atlasDir, atlasBase(options.Name)+".dds")

	grid := options.Grid
	if options.AtlasPath != "" {
		config, err := pngConfig(options.AtlasPath)
		if err != nil {
			return report, err
		}
		grid, err = fillGridDimensions(grid, config.Width, config.Height)
		if err != nil {
			return report, err
		}
	} else {
		info, err := ReadDDS(options.DDSPath)
		if err != nil {
			return report, err
		}
		if !info.DXT5 {
			return report, fmt.Errorf("%s is %s; expected DXT5/BC3 DDS", options.DDSPath, info.FourCC)
		}
		grid, err = fillGridDimensions(grid, info.Width, info.Height)
		if err != nil {
			return report, err
		}
	}
	report.Name = options.Name
	report.Preset = options.Preset
	report.Grid = grid
	report.Descriptor = descriptorPath
	report.AtlasMeta = atlasMetaPath
	report.AtlasImage = atlasPath
	report.DatPath = options.DatPath
	report.FromGraphic = options.FromGraphic

	descriptor, err := NewDescriptor(options.Name, options.Preset, grid, options.Overrides)
	if err != nil {
		return report, err
	}
	meta := atlasMeta(options.Name, grid)
	report.DescriptorDoc = descriptor

	if options.DryRun {
		report.OK = true
		return report, nil
	}

	if err := os.MkdirAll(atlasDir, 0755); err != nil {
		return report, err
	}
	if options.AtlasPath != "" {
		if err := encodePNGToDDS(options.AtlasPath, atlasPath); err != nil {
			return report, err
		}
	} else if err := copyFile(options.DDSPath, atlasPath); err != nil {
		return report, err
	}
	if err := writeJSONFile(descriptorPath, descriptor); err != nil {
		return report, err
	}
	if err := writeJSONFile(atlasMetaPath, meta); err != nil {
		return report, err
	}

	compressed, err := os.ReadFile(options.DatPath)
	if err != nil {
		return report, err
	}
	var slp *int32
	if options.ClearSLP {
		zero := int32(0)
		slp = &zero
	}
	graphicName := options.Name
	newCompressed, createReport, err := datfile.CreateGraphic(compressed, datfile.GraphicCreateRecipe{
		From:               options.FromGraphic,
		Name:               &graphicName,
		ParticleEffectName: &graphicName,
		SLP:                slp,
	})
	if err != nil {
		return report, fmt.Errorf("create graphic: %w", err)
	}
	report.NewGraphicID = createReport.NewGraphicID
	graphID := int16(createReport.NewGraphicID)
	newCompressed, patchReports, err := datfile.PatchUnitGraphicSlotAllCivs(newCompressed, options.UnitID, options.Slot, graphID)
	if err != nil {
		return report, fmt.Errorf("bind unit %d slot %s: %w", options.UnitID, options.Slot, err)
	}
	for _, patchReport := range patchReports {
		report.UnitPatches = append(report.UnitPatches, UnitBindPatch{
			CivID:  patchReport.CivID,
			UnitID: patchReport.UnitID,
			Before: slotValue(patchReport.Before, options.Slot),
			After:  int(graphID),
		})
		if !patchReport.Verified {
			return report, fmt.Errorf("unit patch %d/%d did not verify", patchReport.CivID, options.UnitID)
		}
	}
	if err := os.WriteFile(options.DatPath, newCompressed, 0644); err != nil {
		return report, err
	}

	checksums := map[string]string{}
	for _, path := range []string{descriptorPath, atlasMetaPath, atlasPath, options.DatPath} {
		sum, err := fileSHA256(path)
		if err != nil {
			return report, err
		}
		checksums[path] = sum
	}
	report.Checksums = checksums
	report.OK = true
	return report, nil
}

func Lint(options LintOptions) (LintReport, error) {
	commonRoot, err := resolveCommonRoot(options.ModPath)
	if err != nil {
		return LintReport{}, err
	}
	report := LintReport{
		Version: Version,
		Verification: aoe2.VerificationClaim{
			StructureVerified: true,
			EngineVerified:    false,
			Note:              "kit fx lint checks descriptor files, atlas references, DDS headers, and optional DAT particle bindings; it is not an in-engine render test.",
		},
		CommonRoot: commonRoot,
	}
	particlesDir := filepath.Join(commonRoot, "particles")
	entries, err := os.ReadDir(particlesDir)
	if err != nil {
		return report, fmt.Errorf("read particles dir: %w", err)
	}
	datPath := options.DatPath
	if datPath == "" {
		candidate := filepath.Join(commonRoot, "dat", "empires2_x2_p1.dat")
		if _, err := os.Stat(candidate); err == nil {
			datPath = candidate
		}
	}
	report.DatPath = datPath
	datParticles := map[string]bool{}
	if datPath != "" {
		idx, err := datfile.Open(datPath)
		if err != nil {
			report.Errors = append(report.Errors, fmt.Sprintf("read dat particle references: %v", err))
		} else {
			for _, graphic := range idx.PresentGraphicSummariesFiltered(datfile.GraphicFilter{ParticleOnly: true}) {
				datParticles[graphic.ParticleEffectName] = true
			}
		}
	} else {
		report.Warnings = append(report.Warnings, "no dat found; particle_effect_name reference checks skipped")
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".json") {
			continue
		}
		path := filepath.Join(particlesDir, entry.Name())
		desc, ok, err := readDescriptorCandidate(path)
		if err != nil {
			report.Errors = append(report.Errors, err.Error())
			continue
		}
		if !ok {
			continue
		}
		lint := lintDescriptor(commonRoot, path, desc, datParticles)
		report.Descriptors = append(report.Descriptors, lint)
		if len(lint.Errors) > 0 {
			for _, msg := range lint.Errors {
				report.Errors = append(report.Errors, lint.Name+": "+msg)
			}
		}
	}
	sort.Slice(report.Descriptors, func(i, j int) bool {
		return report.Descriptors[i].Name < report.Descriptors[j].Name
	})
	report.OK = len(report.Errors) == 0
	return report, nil
}

func ReadDDS(path string) (DDSInfo, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return DDSInfo{}, err
	}
	if len(data) < 128 {
		return DDSInfo{}, fmt.Errorf("%s is too small for a DDS header", path)
	}
	if string(data[:4]) != "DDS " {
		return DDSInfo{}, fmt.Errorf("%s missing DDS magic", path)
	}
	headerSize := binary.LittleEndian.Uint32(data[4:8])
	width := binary.LittleEndian.Uint32(data[16:20])
	height := binary.LittleEndian.Uint32(data[12:16])
	fourCC := string(data[84:88])
	return DDSInfo{
		Path:       path,
		Width:      int(width),
		Height:     int(height),
		FourCC:     strings.TrimRight(fourCC, "\x00 "),
		DXT5:       fourCC == "DXT5" || fourCC == "BC3 ",
		HeaderSize: headerSize,
	}, nil
}

func presetDescriptor(preset string) (map[string]any, error) {
	switch preset {
	case "", "trail":
		return map[string]any{
			"Type":           "Once",
			"Duration":       1.0,
			"StopMode":       "Complete",
			"AlphaStart":     1.0,
			"AlphaEnd":       0.0,
			"Rotation1":      -360,
			"Rotation2":      360,
			"RotationSpeed1": -180,
			"RotationSpeed2": 180,
			"Scale":          0.25,
			"IsFire":         true,
			"DisplayInFog":   true,
		}, nil
	case "projectile-fire":
		return map[string]any{
			"Type":           "Once",
			"Duration":       0.8,
			"StopMode":       "Complete",
			"AlphaStart":     1.0,
			"AlphaEnd":       0.2,
			"Rotation1":      -180,
			"Rotation2":      180,
			"RotationSpeed1": -90,
			"RotationSpeed2": 90,
			"Scale":          0.35,
			"IsFire":         true,
			"DisplayInFog":   true,
		}, nil
	case "explosion":
		return map[string]any{
			"Type":       "Once",
			"Duration":   0.7,
			"StopMode":   "Complete",
			"AlphaStart": 1.0,
			"AlphaEnd":   0.0,
			"Scale":      0.8,
			"ScaleSpeed": 0.4,
			"IsFire":     true,
		}, nil
	case "aura":
		return map[string]any{
			"Type":         "Loop",
			"Duration":     1.0,
			"AlphaStart":   0.8,
			"AlphaEnd":     0.8,
			"Scale":        0.5,
			"DisplayInFog": true,
		}, nil
	default:
		return nil, fmt.Errorf("unknown preset %q", preset)
	}
}

func applyOverrides(doc map[string]any, overrides DescriptorOverrides) {
	if overrides.Type != nil {
		doc["Type"] = *overrides.Type
	}
	if overrides.Duration != nil {
		doc["Duration"] = *overrides.Duration
	}
	if overrides.AlphaStart != nil {
		doc["AlphaStart"] = *overrides.AlphaStart
	}
	if overrides.AlphaEnd != nil {
		doc["AlphaEnd"] = *overrides.AlphaEnd
	}
	if overrides.Scale != nil {
		doc["Scale"] = *overrides.Scale
	}
	if overrides.ScaleStart != nil {
		doc["ScaleStart"] = *overrides.ScaleStart
	}
	if overrides.ScaleEnd != nil {
		doc["ScaleEnd"] = *overrides.ScaleEnd
	}
	if overrides.Rotation != nil {
		doc["Rotation"] = *overrides.Rotation
	}
	if overrides.StopMode != nil {
		doc["StopMode"] = *overrides.StopMode
	}
	if overrides.IsFire != nil {
		doc["IsFire"] = *overrides.IsFire
	}
	if overrides.DisplayInFog != nil {
		doc["DisplayInFog"] = *overrides.DisplayInFog
	}
}

func atlasMeta(name string, grid Grid) map[string]any {
	frames := make([]map[string]any, 0, grid.Frames)
	for i := 0; i < grid.Frames; i++ {
		x := (i % grid.Cols) * grid.Width
		y := (i / grid.Cols) * grid.Height
		frameName := fmt.Sprintf("%s_%03d.png", name, i)
		frames = append(frames, map[string]any{
			"filename": frameName,
			"frame": map[string]int{
				"x": x, "y": y, "w": grid.Width, "h": grid.Height,
			},
			"rotated": false,
			"trimmed": false,
			"spriteSourceSize": map[string]int{
				"x": 0, "y": 0, "w": grid.Width, "h": grid.Height,
			},
			"sourceSize": map[string]int{
				"w": grid.Width, "h": grid.Height,
			},
			"pivot": map[string]float64{"x": 0.5, "y": 0.5},
		})
	}
	return map[string]any{
		"frames": frames,
		"meta": map[string]any{
			"app":    "AoE2Kit",
			"format": "RGBA8888",
			"image":  atlasBase(name) + ".dds",
			"size": map[string]int{
				"w": grid.Cols * grid.Width,
				"h": grid.Rows * grid.Height,
			},
			"scale": "1",
		},
	}
}

func lintDescriptor(commonRoot, path string, desc map[string]any, datParticles map[string]bool) DescriptorLint {
	name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	lint := DescriptorLint{Name: name, Path: path, ImageFirst: intFromMap(desc, "ImageFirst"), ImageCount: intFromMap(desc, "ImageCount")}
	atlasFile, _ := desc["AtlasFile"].(string)
	lint.AtlasFile = atlasFile
	if atlasFile == "" {
		lint.Errors = append(lint.Errors, "missing AtlasFile")
		return lint
	}
	if strings.Contains(atlasFile, "/") || !strings.Contains(atlasFile, `\`) {
		lint.Errors = append(lint.Errors, "AtlasFile must use AoE2 backslash separators")
	}
	if _, ok := desc["AlphaStart"]; !ok {
		if _, ok := desc["Alpha"]; !ok {
			lint.Warnings = append(lint.Warnings, "no AlphaStart/Alpha ramp or Alpha key")
		}
	}
	if lint.ImageCount <= 0 {
		lint.Errors = append(lint.Errors, "ImageCount must be positive")
	}
	rel := strings.ReplaceAll(atlasFile, `\`, string(os.PathSeparator))
	atlasPath := filepath.Join(commonRoot, "particles", rel)
	if _, err := os.Stat(atlasPath); err != nil {
		ddsAlt := strings.TrimSuffix(atlasPath, filepath.Ext(atlasPath)) + ".dds"
		if _, altErr := os.Stat(ddsAlt); altErr == nil {
			atlasPath = ddsAlt
		} else {
			lint.Errors = append(lint.Errors, "AtlasFile target does not exist")
		}
	}
	lint.AtlasPath = atlasPath
	if strings.EqualFold(filepath.Ext(atlasPath), ".dds") {
		info, err := ReadDDS(atlasPath)
		if err != nil {
			lint.Errors = append(lint.Errors, "DDS header invalid: "+err.Error())
		} else {
			lint.DDS = &info
			if !info.DXT5 {
				lint.Errors = append(lint.Errors, "DDS FourCC is "+info.FourCC+", expected DXT5/BC3")
			}
		}
	}
	atlasMetaName := filepath.Base(strings.ReplaceAll(atlasFile, `\`, `/`))
	metaPath := filepath.Join(commonRoot, "particles", strings.TrimSuffix(atlasMetaName, filepath.Ext(atlasMetaName))+".json")
	lint.AtlasMetaPath = metaPath
	frames, err := countAtlasFrames(metaPath)
	if err != nil {
		lint.Errors = append(lint.Errors, "atlas metadata invalid: "+err.Error())
	} else {
		lint.AtlasFrames = frames
		if lint.ImageFirst+lint.ImageCount > frames {
			lint.Errors = append(lint.Errors, fmt.Sprintf("ImageFirst+ImageCount=%d exceeds atlas frames=%d", lint.ImageFirst+lint.ImageCount, frames))
		}
		if lint.ImageFirst == 0 && lint.ImageCount != frames {
			lint.Warnings = append(lint.Warnings, fmt.Sprintf("ImageCount=%d differs from atlas frames=%d", lint.ImageCount, frames))
		}
	}
	if datParticles != nil {
		lint.ReferencedByDAT = datParticles[name]
		if !lint.ReferencedByDAT {
			lint.Errors = append(lint.Errors, "descriptor is not referenced by any graphic particle_effect_name in dat")
		}
	}
	return lint
}

func readDescriptorCandidate(path string) (map[string]any, bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false, err
	}
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, false, fmt.Errorf("%s: invalid json: %w", path, err)
	}
	if _, metadata := doc["frames"]; metadata {
		return nil, false, nil
	}
	if _, descriptor := doc["AtlasFile"]; !descriptor {
		return nil, false, nil
	}
	return doc, true, nil
}

func countAtlasFrames(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	var doc struct {
		Frames []json.RawMessage `json:"frames"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return 0, err
	}
	if len(doc.Frames) == 0 {
		return 0, errors.New("missing non-empty frames array")
	}
	return len(doc.Frames), nil
}

func verifyBindReadback(datPath, name string, graphicID, unitID int, slot string, patches []UnitBindPatch) error {
	idx, err := datfile.Open(datPath)
	if err != nil {
		return err
	}
	graphic, ok := idx.Graphic(graphicID)
	if !ok {
		return fmt.Errorf("created graphic %d missing from dat", graphicID)
	}
	if graphic.ParticleEffectName.Value != name {
		return fmt.Errorf("created graphic particle_effect_name=%q want %q", graphic.ParticleEffectName.Value, name)
	}
	for _, patch := range patches {
		if patch.CivID < 0 || patch.CivID >= len(idx.Civs) {
			return fmt.Errorf("civ %d missing after bind", patch.CivID)
		}
		civ := idx.Civs[patch.CivID]
		if unitID < 0 || unitID >= len(civ.Units) || !civ.Units[unitID].Present {
			return fmt.Errorf("unit %d missing in civ %d after bind", unitID, patch.CivID)
		}
		if got := slotValue(civ.Units[unitID], slot); got != graphicID {
			return fmt.Errorf("unit %d civ %d slot %s=%d want %d", unitID, patch.CivID, slot, got, graphicID)
		}
	}
	return nil
}

func slotValue(unit datfile.UnitSummary, slot string) int {
	switch slot {
	case "flying", "standing":
		return int(unit.StandingGraphic1)
	case "standing2":
		return int(unit.StandingGraphic2)
	case "attack":
		if unit.Type50 != nil {
			return int(unit.Type50.AttackGraphic)
		}
		return 0
	case "dying":
		return int(unit.DyingGraphic)
	default:
		return 0
	}
}

func validateSlot(slot string) error {
	switch slot {
	case "flying", "standing", "standing2", "attack", "dying":
		return nil
	default:
		return fmt.Errorf("unknown slot %q; expected flying, standing, attack, or dying", slot)
	}
}

func fillGridDimensions(grid Grid, imageWidth, imageHeight int) (Grid, error) {
	switch {
	case grid.Rows > 0 && grid.Cols > 0:
		if imageWidth%grid.Cols != 0 || imageHeight%grid.Rows != 0 {
			return grid, fmt.Errorf("image dimensions %dx%d are not divisible by grid %dx%d", imageWidth, imageHeight, grid.Cols, grid.Rows)
		}
		grid.Width = imageWidth / grid.Cols
		grid.Height = imageHeight / grid.Rows
	case grid.Width > 0 && grid.Height > 0:
		if imageWidth%grid.Width != 0 || imageHeight%grid.Height != 0 {
			return grid, fmt.Errorf("image dimensions %dx%d are not divisible by frame size %dx%d", imageWidth, imageHeight, grid.Width, grid.Height)
		}
		grid.Cols = imageWidth / grid.Width
		grid.Rows = imageHeight / grid.Height
	default:
		return grid, errors.New("provide --grid RxC or --frames N --frame-size WxH")
	}
	if grid.Frames <= 0 {
		grid.Frames = grid.Rows * grid.Cols
	}
	if grid.Frames > grid.Rows*grid.Cols {
		return grid, fmt.Errorf("frames %d exceed grid cells %d", grid.Frames, grid.Rows*grid.Cols)
	}
	return grid, nil
}

func pngConfig(path string) (image.Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return image.Config{}, err
	}
	defer f.Close()
	config, format, err := image.DecodeConfig(f)
	if err != nil {
		return image.Config{}, err
	}
	if format != "png" {
		return image.Config{}, fmt.Errorf("%s is %s, expected png", path, format)
	}
	return config, nil
}

func encodePNGToDDS(input, output string) error {
	if path, err := exec.LookPath("convert"); err == nil {
		return runConvert(path, input, output)
	}
	if path, err := exec.LookPath("magick"); err == nil {
		cmd := exec.Command(path, "convert", input, "-define", "dds:compression=dxt5", "-define", "dds:mipmaps=0", output)
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("ImageMagick failed: %w: %s", err, strings.TrimSpace(string(out)))
		}
		return nil
	}
	return errors.New("--atlas PNG requires ImageMagick convert; provide a .dds with --dds, or install ImageMagick")
}

func runConvert(path, input, output string) error {
	cmd := exec.Command(path, input, "-define", "dds:compression=dxt5", "-define", "dds:mipmaps=0", output)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("ImageMagick failed: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func writeJSONFile(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(value, "", "\t")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0644)
}

func copyFile(input, output string) error {
	in, err := os.Open(input)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(output), 0755); err != nil {
		return err
	}
	out, err := os.Create(output)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}

func fileSHA256(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func intFromMap(doc map[string]any, key string) int {
	switch value := doc[key].(type) {
	case float64:
		return int(value)
	case int:
		return value
	case json.Number:
		n, _ := strconv.Atoi(value.String())
		return n
	default:
		return 0
	}
}

func resolveCommonRoot(path string) (string, error) {
	if path == "" {
		path = "."
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	if filepath.Base(abs) == "_common" {
		return abs, nil
	}
	if _, err := os.Stat(filepath.Join(abs, "particles")); err == nil {
		return abs, nil
	}
	candidate := filepath.Join(abs, "resources", "_common")
	if _, err := os.Stat(candidate); err == nil {
		return candidate, nil
	}
	candidate = filepath.Join(abs, "_common")
	if _, err := os.Stat(candidate); err == nil {
		return candidate, nil
	}
	return abs, nil
}

func validateName(name string) error {
	if name == "" {
		return errors.New("name is required")
	}
	if strings.ContainsAny(name, `/\`) || strings.Contains(name, "..") {
		return fmt.Errorf("invalid effect name %q", name)
	}
	return nil
}

func atlasBase(name string) string {
	return name + "_atlas"
}

func ParseGrid(raw string) (Grid, error) {
	parts := strings.Split(strings.ToLower(raw), "x")
	if len(parts) != 2 {
		return Grid{}, fmt.Errorf("invalid --grid %q; expected RxC", raw)
	}
	rows, err := strconv.Atoi(parts[0])
	if err != nil || rows <= 0 {
		return Grid{}, fmt.Errorf("invalid --grid rows %q", parts[0])
	}
	cols, err := strconv.Atoi(parts[1])
	if err != nil || cols <= 0 {
		return Grid{}, fmt.Errorf("invalid --grid cols %q", parts[1])
	}
	return Grid{Rows: rows, Cols: cols, Frames: rows * cols}, nil
}

func ParseFrameSize(raw string) (Grid, error) {
	parts := strings.Split(strings.ToLower(raw), "x")
	if len(parts) != 2 {
		return Grid{}, fmt.Errorf("invalid --frame-size %q; expected WxH", raw)
	}
	width, err := strconv.Atoi(parts[0])
	if err != nil || width <= 0 {
		return Grid{}, fmt.Errorf("invalid --frame-size width %q", parts[0])
	}
	height, err := strconv.Atoi(parts[1])
	if err != nil || height <= 0 {
		return Grid{}, fmt.Errorf("invalid --frame-size height %q", parts[1])
	}
	return Grid{Width: width, Height: height}, nil
}

func DescriptorJSON(name, preset string, grid Grid, overrides DescriptorOverrides) ([]byte, error) {
	doc, err := NewDescriptor(name, preset, grid, overrides)
	if err != nil {
		return nil, err
	}
	return json.MarshalIndent(doc, "", "  ")
}

func HasSingleBackslashAtlasFile(data []byte) bool {
	return bytes.Contains(data, []byte(`"AtlasFile"`)) && bytes.Contains(data, []byte(`textures\\atlases\\`))
}
