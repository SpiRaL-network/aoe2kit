package datfile

import (
	"bytes"
	"fmt"
	"os"
	"testing"

	"aoe2kit/pkg/testfixtures"
)

func TestEffectOperandNamesRoundTrip(t *testing.T) {
	resourceCases := []struct {
		id   int16
		name string
	}{
		{0, "food"},
		{20, "kills"},
		{22, "exploration"},
		{43, "razings"},
		{154, "killed_by_others"},
		{155, "razed_by_others"},
		{170, "total_value_of_kills"},
		{172, "total_value_of_razings"},
		{301, "player_1_kills"},
		{326, "kills_by_player_1"},
		{351, "player_1_razings"},
		{376, "razings_by_player_1"},
	}
	for _, tc := range resourceCases {
		if got := EffectResourceName(tc.id); got != tc.name {
			t.Fatalf("EffectResourceName(%d) = %q, want %q", tc.id, got, tc.name)
		}
		got, ok := EffectResourceID(tc.name)
		if !ok || got != tc.id {
			t.Fatalf("EffectResourceID(%q) = %d/%v, want %d/true", tc.name, got, ok, tc.id)
		}
	}
	if got, ok := EffectResourceID("cAttributeKills"); !ok || got != 20 {
		t.Fatalf("EffectResourceID(cAttributeKills) = %d/%v, want 20/true", got, ok)
	}
	if got, ok := EffectResourceID("cAttributePlayer8Razings"); !ok || got != 358 {
		t.Fatalf("EffectResourceID(cAttributePlayer8Razings) = %d/%v, want 358/true", got, ok)
	}
	if got, ok := EffectOperationID("cAttributeAdd"); !ok || got != 1 {
		t.Fatalf("EffectOperationID(cAttributeAdd) = %d/%v, want 1/true", got, ok)
	}
	if got, ok := EffectAttributeID("cAttack"); !ok || got != 9 {
		t.Fatalf("EffectAttributeID(cAttack) = %d/%v, want 9/true", got, ok)
	}
	if got, ok := EffectTechAttributeID("cAttrSetGoldCost"); !ok || got != 3 {
		t.Fatalf("EffectTechAttributeID(cAttrSetGoldCost) = %d/%v, want 3/true", got, ok)
	}
}

func TestCurrentDEDatParsesThroughTechTreeGolden(t *testing.T) {
	path := testfixtures.Path(t, "reference_aoe2_dump/dat/empires2_x2_p1.dat")
	if _, err := os.Stat(path); err != nil {
		t.Skipf("golden dat not present: %v", err)
	}
	idx, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if idx.Version != "VER 8.9" {
		t.Fatalf("version = %q, want VER 8.9", idx.Version)
	}
	if got, want := len(idx.Effects), 1409; got != want {
		t.Fatalf("effects = %d, want %d", got, want)
	}
	if got, want := len(idx.Civs), 60; got != want {
		t.Fatalf("civs = %d, want %d", got, want)
	}
	if got, want := len(idx.Techs), 1510; got != want {
		t.Fatalf("techs = %d, want %d", got, want)
	}
	if idx.Effects[1].Name != "Hindustanis Tech Tree" || len(idx.Effects[1].Commands) != 36 {
		t.Fatalf("effect[1] = %q commands=%d, want Hindustanis Tech Tree/36", idx.Effects[1].Name, len(idx.Effects[1].Commands))
	}
	firstCommand := idx.Effects[1].Commands[0]
	if firstCommand.Semantic == nil {
		t.Fatalf("effect[1].command[0] missing semantic overlay")
	}
	if firstCommand.Semantic.TypeName != "disable_tech" || firstCommand.Semantic.ReferenceKind != "tech" || firstCommand.Semantic.ReferenceID == nil || *firstCommand.Semantic.ReferenceID != 244 {
		t.Fatalf("effect[1].command[0] semantic = %+v, want disable_tech tech 244", firstCommand.Semantic)
	}
	if idx.Techs[2].Name != "Elite Tarkan" || idx.Techs[2].EffectID != 454 {
		t.Fatalf("tech[2] = %q effect=%d, want Elite Tarkan/454", idx.Techs[2].Name, idx.Techs[2].EffectID)
	}
	if idx.GameMetrics.TimeSlice != 30 || idx.GameMetrics.UnitKillRate != 7 || idx.GameMetrics.RazingKillTotal != 10 {
		t.Fatalf("game metrics = %+v", idx.GameMetrics)
	}
	if idx.TechTree.AgeCount != 4 || idx.TechTree.BuildingCount != 37 || idx.TechTree.UnitCount != 255 || idx.TechTree.ResearchCount != 233 {
		t.Fatalf("tech tree counts = %+v", idx.TechTree)
	}
	if len(idx.Spans) == 0 {
		t.Fatalf("no spans")
	}
	particleGraphics := idx.PresentGraphicSummariesFiltered(GraphicFilter{ParticleOnly: true})
	if got, want := len(particleGraphics), 184; got != want {
		t.Fatalf("particle graphics = %d, want %d", got, want)
	}
	smokeTrail, ok := idx.Graphic(1711)
	if !ok {
		t.Fatal("graphic 1711 absent")
	}
	if smokeTrail.Summary().ParticleEffectName != "smoke_trail" {
		t.Fatalf("graphic 1711 particle_effect_name = %q, want smoke_trail", smokeTrail.Summary().ParticleEffectName)
	}
	last := idx.Spans[len(idx.Spans)-1]
	if last.Name != "tech_tree" || last.End != idx.Inflated {
		t.Fatalf("last span = %+v, inflated=%d; want tech_tree ending at EOF", last, idx.Inflated)
	}
}

func TestDecodeUnitFileUsesCurrentDELayout(t *testing.T) {
	path := testfixtures.Path(t, "reference_aoe2_dump/dat/empires2_x2_p1.dat")
	if _, err := os.Stat(path); err != nil {
		t.Skipf("golden dat not present: %v", err)
	}
	unit, fields, err := DecodeUnitFile(path, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if unit.Name != "OLD-ACADEMY" {
		t.Fatalf("unit name = %q, want OLD-ACADEMY", unit.Name)
	}
	for _, key := range []string{"type", "id", "name", "attributes", "costs", "train_locations"} {
		if _, ok := fields[key]; !ok {
			t.Fatalf("decoded fields missing %q", key)
		}
	}
	if unit.FullFields == nil {
		t.Fatal("single-unit decode did not attach full_fields")
	}
}

func TestDecodeUnitFileExposesStableSkippedAndBuildingFields(t *testing.T) {
	path := testfixtures.Path(t, "reference_aoe2_dump/dat/empires2_x2_p1.dat")
	if _, err := os.Stat(path); err != nil {
		t.Skipf("golden dat not present: %v", err)
	}
	unit, fields, err := DecodeUnitFile(path, 0, 109)
	if err != nil {
		t.Fatal(err)
	}
	if unit.Building == nil {
		t.Fatalf("unit 109 type = %d, want building", unit.Type)
	}
	for _, key := range []string{
		"death_spawn", "train_sound", "damage_sound", "undead_graphic", "garrison_capacity",
		"spawning_graphic", "upgrade_graphic", "stack_unit", "head_unit", "transform_unit", "pile_unit",
		"annex_unit_1", "annex_unit_1_offset_x", "annex_unit_4_offset_y", "garrison_type", "salvage_unit",
	} {
		if _, ok := fields[key]; !ok {
			t.Errorf("decoded fields missing %q", key)
		}
	}
	for _, row := range unit.Building.LinkedBuildings {
		if row.Active {
			continue
		}
		key := fmt.Sprintf("annex_unit_%d", row.Index+1)
		value, ok := fields[key]
		if !ok {
			t.Fatalf("decoded fields missing %q", key)
		}
		got, ok := value.(int16)
		if !ok || got != -1 {
			t.Errorf("%s = %#v (%T), want signed sentinel -1", key, value, value)
		}
	}
}

func TestGraphicFilters(t *testing.T) {
	idx := &Index{Graphics: []Graphic{
		{
			Index:              1,
			Present:            true,
			Name:               DebugString{Value: "Smoke Trail"},
			FileName:           DebugString{Value: "smoke.smx"},
			ParticleEffectName: DebugString{Value: "smoke_trail"},
		},
		{
			Index:              2,
			Present:            true,
			Name:               DebugString{Value: "Archer"},
			FileName:           DebugString{Value: "archer.smx"},
			ParticleEffectName: DebugString{},
		},
		{
			Index:              3,
			Present:            false,
			Name:               DebugString{Value: "Hidden Smoke"},
			ParticleEffectName: DebugString{Value: "hidden"},
		},
	}}
	if got := idx.PresentGraphicSummariesFiltered(GraphicFilter{ParticleOnly: true}); len(got) != 1 || got[0].Index != 1 {
		t.Fatalf("particle filter = %+v, want only index 1", got)
	}
	if got := idx.PresentGraphicSummariesFiltered(GraphicFilter{NameContains: "SMOKE"}); len(got) != 1 || got[0].Index != 1 {
		t.Fatalf("name filter = %+v, want only index 1", got)
	}
	if got := idx.PresentGraphicSummariesFiltered(GraphicFilter{NameContains: "trail", ParticleOnly: true}); len(got) != 1 || got[0].Index != 1 {
		t.Fatalf("combined filter = %+v, want only index 1", got)
	}
}

func TestPaletteClassifiesAndCollapsesCivRows(t *testing.T) {
	graphics := make([]Graphic, 12)
	graphics[10] = Graphic{
		Index:        10,
		Present:      true,
		Name:         DebugString{Value: "Many Statues"},
		FileName:     DebugString{Value: "s_many_statues"},
		SLP:          1234,
		AngleCount:   16,
		FrameCount:   1,
		SequenceType: 6,
	}
	graphics[11] = Graphic{
		Index:        11,
		Present:      true,
		Name:         DebugString{Value: "Moving Fish"},
		FileName:     DebugString{Value: "s_fish"},
		SLP:          5678,
		AngleCount:   16,
		FrameCount:   8,
		SequenceType: 2,
	}
	graphics = append(graphics, Graphic{
		Index:        12,
		Present:      true,
		Name:         DebugString{Value: "Fish Marlin1"},
		FileName:     DebugString{Value: "a_fish_marlin1_x1"},
		SLP:          9012,
		AngleCount:   1,
		FrameCount:   60,
		SequenceType: 7,
	})
	idx := &Index{
		Version:  "VER 8.9",
		Graphics: graphics,
		Civs: []Civ{
			{Index: 0, Units: []UnitSummary{{Present: true, Index: 1777, ID: 1777, Name: "STATUE", StandingGraphic1: 10}}},
			{Index: 1, Units: []UnitSummary{{Present: true, Index: 1777, ID: 1777, Name: "STATUE", StandingGraphic1: 10}}},
			{Index: 2, Units: []UnitSummary{{Present: true, Index: 42, ID: 42, Name: "FISH", StandingGraphic1: 11}}},
			{Index: 3, Units: []UnitSummary{{Present: true, Index: 455, ID: 455, Name: "FISH1", StandingGraphic1: 11}}},
			{Index: 4, Units: []UnitSummary{{Present: true, Index: 450, ID: 450, Name: "DOLP1", StandingGraphic1: 12}}},
		},
	}
	report := idx.Palette(PaletteOptions{MinVariants: 2})
	if report.Returned != 1 {
		t.Fatalf("returned = %d, want 1", report.Returned)
	}
	row := report.Rows[0]
	if row.UnitID != 1777 || row.StandingGraphic1 != 10 || row.Classification != "multi_variant" {
		t.Fatalf("palette row = %+v, want statue multi_variant", row)
	}
	if row.VariantCount == nil || *row.VariantCount != 16 {
		t.Fatalf("variant_count = %+v, want 16", row.VariantCount)
	}
	if len(row.CivIndices) != 2 || row.CivIndices[0] != 0 || row.CivIndices[1] != 1 {
		t.Fatalf("civ indices = %v, want [0 1]", row.CivIndices)
	}
	fishID := 455
	fishReport := idx.Palette(PaletteOptions{ID: &fishID})
	if len(fishReport.Rows) != 1 {
		t.Fatalf("fish rows = %d, want 1", len(fishReport.Rows))
	}
	fish := fishReport.Rows[0]
	if fish.Classification != "animated" || fish.Confidence != "author_confirmed" || fish.VariantCount != nil {
		t.Fatalf("fish palette row = %+v, want author-confirmed animated without variant_count", fish)
	}
	marlinID := 450
	marlinReport := idx.Palette(PaletteOptions{ID: &marlinID})
	if len(marlinReport.Rows) != 1 || marlinReport.Rows[0].Classification != "animated" || marlinReport.Rows[0].Confidence != "author_confirmed" {
		t.Fatalf("marlin palette row = %+v, want graphic-name fish classified animated/author_confirmed", marlinReport.Rows)
	}
}

func TestPaletteClassifiesType10EyecandyAsIntegerArtworkIndex(t *testing.T) {
	graphics := make([]Graphic, 6)
	graphics[5] = Graphic{
		Index:        5,
		Present:      true,
		Name:         DebugString{Value: "Rock Limestone Hover"},
		FileName:     DebugString{Value: "n_rock_limestone_x1"},
		SLP:          8886,
		AngleCount:   4,
		FrameCount:   1,
		SequenceType: 6,
	}
	idx := &Index{
		Version:  "VER 8.9",
		Graphics: graphics,
		Civs: []Civ{
			{Index: 0, Units: []UnitSummary{{
				Present:          true,
				Index:            2411,
				ID:               2411,
				Name:             "Rock Limestone Hover",
				Type:             10,
				Class:            14,
				ClassName:        "Eye Candy",
				StandingGraphic1: 5,
			}}},
		},
	}
	id := 2411
	report := idx.Palette(PaletteOptions{ID: &id})
	if len(report.Rows) != 1 {
		t.Fatalf("rows = %d, want 1", len(report.Rows))
	}
	row := report.Rows[0]
	if row.UnitType != 10 || row.UnitClass != 14 || row.UnitClassName != "Eye Candy" {
		t.Fatalf("unit type/class = %d/%d/%q, want 10/14/Eye Candy", row.UnitType, row.UnitClass, row.UnitClassName)
	}
	if row.Classification != "multi_variant" || row.RotationEncoding != "integer_artwork_index" || row.Confidence != "editor_fixture" {
		t.Fatalf("type-10 row = %+v, want editor-fixture integer artwork index", row)
	}
	if row.VariantCount == nil || *row.VariantCount != 4 {
		t.Fatalf("variant_count = %+v, want 4", row.VariantCount)
	}
}

func TestCreateGraphicGolden(t *testing.T) {
	path := testfixtures.Path(t, "reference_aoe2_dump/dat/empires2_x2_p1.dat")
	compressed, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	name := "AoE2Kit Test Smoke Trail"
	particle := "aoe2kit_test_trail"
	slp := int32(971)
	layer := int8(20)
	playerColor := int8(3)
	frameDuration := float32(0.125)
	out, report, err := CreateGraphic(compressed, GraphicCreateRecipe{
		From:               1711,
		Name:               &name,
		ParticleEffectName: &particle,
		SLP:                &slp,
		Layer:              &layer,
		PlayerColor:        &playerColor,
		FrameDuration:      &frameDuration,
	})
	if err != nil {
		t.Fatalf("CreateGraphic: %v", err)
	}
	if len(out) == 0 {
		t.Fatalf("CreateGraphic returned empty output")
	}
	if !report.Verified {
		t.Fatalf("report not verified: %+v", report)
	}
	if got, want := report.NewGraphicID, report.BeforeGraphicsSize; got != want {
		t.Fatalf("new graphic id = %d, want old graphics size %d", got, want)
	}
	idx, err := Parse(mustInflate(t, out))
	if err != nil {
		t.Fatalf("parse output: %v", err)
	}
	created, ok := idx.Graphic(report.NewGraphicID)
	if !ok {
		t.Fatalf("created graphic %d missing", report.NewGraphicID)
	}
	if created.Name.Value != name ||
		created.ParticleEffectName.Value != particle ||
		created.SLP != slp ||
		created.Layer != layer ||
		created.PlayerColor != playerColor ||
		created.FrameDuration != frameDuration {
		t.Fatalf("created graphic = %+v", created.Summary())
	}
	if idx.GraphicsSize != report.BeforeGraphicsSize+1 {
		t.Fatalf("graphics size = %d, want %d", idx.GraphicsSize, report.BeforeGraphicsSize+1)
	}
}

func TestDeleteGraphicStableIDGolden(t *testing.T) {
	path := testfixtures.Path(t, "reference_aoe2_dump/dat/empires2_x2_p1.dat")
	compressed, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	name := "AoE2Kit Delete Graphic Probe"
	createdDat, createReport, err := CreateGraphic(compressed, GraphicCreateRecipe{
		From: 1711,
		Name: &name,
	})
	if err != nil {
		t.Fatalf("CreateGraphic: %v", err)
	}
	deletedDat, deleteReport, err := DeleteGraphic(createdDat, createReport.NewGraphicID)
	if err != nil {
		t.Fatalf("DeleteGraphic: %v", err)
	}
	if !deleteReport.Verified {
		t.Fatalf("delete report not verified: %+v", deleteReport)
	}
	if deleteReport.BeforeGraphicsSize != deleteReport.AfterGraphicsSize {
		t.Fatalf("graphics size changed: before=%d after=%d", deleteReport.BeforeGraphicsSize, deleteReport.AfterGraphicsSize)
	}
	idx, err := Parse(mustInflate(t, deletedDat))
	if err != nil {
		t.Fatalf("parse output: %v", err)
	}
	if idx.GraphicsSize != createReport.AfterGraphicsSize {
		t.Fatalf("graphics size = %d, want %d", idx.GraphicsSize, createReport.AfterGraphicsSize)
	}
	if _, ok := idx.Graphic(createReport.NewGraphicID); ok {
		t.Fatalf("deleted graphic %d still present", createReport.NewGraphicID)
	}
}

func TestPatchGraphicScalarFieldsGolden(t *testing.T) {
	path := testfixtures.Path(t, "reference_aoe2_dump/dat/empires2_x2_p1.dat")
	compressed, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	isLoaded := int8(1)
	oldColorFlag := int8(1)
	layer := int8(25)
	playerColor := int8(2)
	rainbow := int8(0)
	transparent := int8(1)
	coordinates := []int16{1, 2, 3, 4}
	soundID := int16(12)
	wwiseSoundID := uint32(34)
	frameCount := int16(31)
	speedMultiplier := float32(1.5)
	frameDuration := float32(0.125)
	replayDelay := float32(0.25)
	sequenceType := uint8(7)
	mirroringMode := int8(1)
	editorFlag := int8(1)
	out, report, err := PatchGraphic(compressed, 3396, GraphicPatch{
		IsLoaded:          &isLoaded,
		OldColorFlag:      &oldColorFlag,
		Layer:             &layer,
		PlayerColor:       &playerColor,
		Rainbow:           &rainbow,
		TransparentSelect: &transparent,
		Coordinates:       coordinates,
		SoundID:           &soundID,
		WwiseSoundID:      &wwiseSoundID,
		FrameCount:        &frameCount,
		SpeedMultiplier:   &speedMultiplier,
		FrameDuration:     &frameDuration,
		ReplayDelay:       &replayDelay,
		SequenceType:      &sequenceType,
		MirroringMode:     &mirroringMode,
		EditorFlag:        &editorFlag,
	})
	if err != nil {
		t.Fatalf("PatchGraphic: %v", err)
	}
	if len(out) == 0 || !report.Verified {
		t.Fatalf("patch report = %+v", report)
	}
	idx, err := Parse(mustInflate(t, out))
	if err != nil {
		t.Fatalf("parse output: %v", err)
	}
	graphic, ok := idx.Graphic(3396)
	if !ok {
		t.Fatal("patched graphic missing")
	}
	if graphic.IsLoaded != isLoaded ||
		graphic.OldColorFlag != oldColorFlag ||
		graphic.Layer != layer ||
		graphic.PlayerColor != playerColor ||
		graphic.Rainbow != rainbow ||
		graphic.TransparentSelect != transparent ||
		graphic.SoundID != soundID ||
		graphic.WwiseSoundID != wwiseSoundID ||
		graphic.FrameCount != frameCount ||
		graphic.SpeedMultiplier != speedMultiplier ||
		graphic.FrameDuration != frameDuration ||
		graphic.ReplayDelay != replayDelay ||
		graphic.SequenceType != sequenceType ||
		graphic.MirroringMode != mirroringMode ||
		graphic.EditorFlag != editorFlag {
		t.Fatalf("patched graphic = %+v", graphic)
	}
	for i, want := range coordinates {
		if graphic.Coordinates[i] != want {
			t.Fatalf("coordinate[%d] = %d, want %d", i, graphic.Coordinates[i], want)
		}
	}
}

func TestPatchGraphicChildListsGolden(t *testing.T) {
	path := testfixtures.Path(t, "reference_aoe2_dump/dat/empires2_x2_p1.dat")
	compressed, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	before, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	graphic, ok := before.Graphic(5)
	if !ok {
		t.Fatal("graphic 5 missing")
	}
	if len(graphic.Deltas) != 3 || graphic.AngleCount != 1 || len(graphic.AngleSounds) != 0 {
		t.Fatalf("graphic 5 shape = deltas %d angles %d angle_sounds %d", len(graphic.Deltas), graphic.AngleCount, len(graphic.AngleSounds))
	}
	deltas := []GraphicDeltaRow{graphicDeltaRow(graphic.Deltas[0])}
	deltas[0].OffsetX += 2
	deltas[0].OffsetY -= 1
	angleSounds := []GraphicAngleSoundRow{{
		FrameNum:      1,
		SoundID:       12,
		WwiseSoundID:  34,
		FrameNum2:     -1,
		WwiseSoundID2: 0,
		SoundID2:      -1,
		FrameNum3:     -1,
		WwiseSoundID3: 0,
		SoundID3:      -1,
	}}
	out, report, err := PatchGraphic(compressed, 5, GraphicPatch{
		SetDeltas:      &deltas,
		SetAngleSounds: &angleSounds,
	})
	if err != nil {
		t.Fatalf("PatchGraphic: %v", err)
	}
	if len(out) == 0 || !report.Verified || report.InflatedLengthDelta != -8 {
		t.Fatalf("patch report = %+v", report)
	}
	idx, err := Parse(mustInflate(t, out))
	if err != nil {
		t.Fatalf("parse output: %v", err)
	}
	after, ok := idx.Graphic(5)
	if !ok {
		t.Fatal("patched graphic missing")
	}
	if after.DeltaCount != 1 || len(after.Deltas) != 1 || graphicDeltaRow(after.Deltas[0]) != deltas[0] {
		t.Fatalf("patched deltas = count %d rows %+v want %+v", after.DeltaCount, after.Deltas, deltas)
	}
	if after.AngleSoundsUsed == 0 || len(after.AngleSounds) != 1 || graphicAngleSoundRow(after.AngleSounds[0]) != angleSounds[0] {
		t.Fatalf("patched angle sounds = used %d rows %+v want %+v", after.AngleSoundsUsed, after.AngleSounds, angleSounds)
	}

	emptyAngleSounds := []GraphicAngleSoundRow{}
	out, report, err = PatchGraphic(out, 5, GraphicPatch{SetAngleSounds: &emptyAngleSounds})
	if err != nil {
		t.Fatalf("PatchGraphic disable angle sounds: %v", err)
	}
	if !report.Verified || report.InflatedLengthDelta != -24 {
		t.Fatalf("disable angle-sound report = %+v", report)
	}
	idx, err = Parse(mustInflate(t, out))
	if err != nil {
		t.Fatalf("parse disable output: %v", err)
	}
	after, ok = idx.Graphic(5)
	if !ok {
		t.Fatal("patched graphic missing after disable")
	}
	if after.AngleSoundsUsed != 0 || len(after.AngleSounds) != 0 {
		t.Fatalf("angle sounds after disable = used %d rows %+v", after.AngleSoundsUsed, after.AngleSounds)
	}
}

func TestCreateUnitGoldenSubset(t *testing.T) {
	path := testfixtures.Path(t, "reference_aoe2_dump/dat/empires2_x2_p1.dat")
	compressed, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	hp := int16(77)
	enabled := uint8(1)
	out, report, err := CreateUnit(compressed, UnitCreateRecipe{
		FromCivID:  0,
		FromUnitID: 83,
		CivIDs:     []int{0, 1},
		HitPoints:  &hp,
		Enabled:    &enabled,
	})
	if err != nil {
		t.Fatalf("CreateUnit: %v", err)
	}
	if len(out) == 0 {
		t.Fatalf("CreateUnit returned empty output")
	}
	if !report.Verified {
		t.Fatalf("report not verified: %+v", report)
	}
	if got, want := report.NewUnitID, 2701; got != want {
		t.Fatalf("new unit id = %d, want %d", got, want)
	}
	idx, err := Parse(mustInflate(t, out))
	if err != nil {
		t.Fatalf("parse output: %v", err)
	}
	for _, civID := range []int{0, 1} {
		unit, err := unitByID(idx, civID, report.NewUnitID)
		if err != nil {
			t.Fatalf("created unit civ %d: %v", civID, err)
		}
		if unit.ID != int16(report.NewUnitID) || unit.HitPoints != hp || unit.Enabled != enabled {
			t.Fatalf("created unit civ %d = %+v", civID, unit)
		}
	}
	if idx.Civs[2].UnitsSize != 2701 {
		t.Fatalf("untouched civ 2 units size = %d, want 2701", idx.Civs[2].UnitsSize)
	}
}

func TestPatchUnitExpandedRowsGolden(t *testing.T) {
	path := testfixtures.Path(t, "reference_aoe2_dump/dat/empires2_x2_p1.dat")
	compressed, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	class := int16(6)
	los := float32(7.5)
	movement := uint8(1)
	attrAmount := float32(3)
	attackValue := int16(9)
	armourValue := int16(4)
	costAmount := int16(55)
	trainTime := int16(19)
	taskAction := int16(3)
	taskWorkRange := float32(2.25)
	taskCombatLevel := uint8(1)
	out, report, err := PatchUnit(compressed, 1, 74, UnitPatch{
		Class:        &class,
		LineOfSight:  &los,
		MovementType: &movement,
		Attributes: []UnitAttributePatch{{
			Index:  1,
			Amount: &attrAmount,
		}},
		Type50Attacks: []WeaponInfoPatch{{
			Index: 2,
			Value: &attackValue,
		}},
		Type50Armours: []WeaponInfoPatch{{
			Index: 2,
			Value: &armourValue,
		}},
		Costs: []AttributeCostPatch{{
			Index:  0,
			Amount: &costAmount,
		}},
		TrainLocations: []TrainLocationPatch{{
			Index:     0,
			TrainTime: &trainTime,
		}},
		Tasks: []TaskPatch{{
			Index:       0,
			ActionType:  &taskAction,
			WorkRange:   &taskWorkRange,
			CombatLevel: &taskCombatLevel,
		}},
	})
	if err != nil {
		t.Fatalf("PatchUnit: %v", err)
	}
	if len(out) == 0 || !report.Verified {
		t.Fatalf("patch report = %+v", report)
	}
	if report.InflatedLengthDelta != 0 {
		t.Fatalf("inflated length delta = %d, want 0", report.InflatedLengthDelta)
	}
	idx, err := Parse(mustInflate(t, out))
	if err != nil {
		t.Fatalf("parse output: %v", err)
	}
	unit, err := unitByID(idx, 1, 74)
	if err != nil {
		t.Fatal(err)
	}
	if unit.Class != class || unit.LineOfSight != los || unit.MovementType != movement {
		t.Fatalf("patched scalar fields = class %d los %f movement %d", unit.Class, unit.LineOfSight, unit.MovementType)
	}
	if unit.Attributes[1].Amount != attrAmount {
		t.Fatalf("attribute amount = %f, want %f", unit.Attributes[1].Amount, attrAmount)
	}
	if unit.Type50.Attacks[2].Value != attackValue || unit.Type50.Armours[2].Value != armourValue {
		t.Fatalf("type50 rows = attack %d armour %d", unit.Type50.Attacks[2].Value, unit.Type50.Armours[2].Value)
	}
	if unit.Creatable.Costs[0].Amount != costAmount || unit.Creatable.TrainLocations[0].TrainTime != trainTime {
		t.Fatalf("creatable rows = cost %d train %d", unit.Creatable.Costs[0].Amount, unit.Creatable.TrainLocations[0].TrainTime)
	}
	if unit.Action.Tasks[0].ActionType != taskAction || unit.Action.Tasks[0].WorkRange != taskWorkRange || unit.Action.Tasks[0].CombatLevel != taskCombatLevel {
		t.Fatalf("task row = action %d range %f combat %d", unit.Action.Tasks[0].ActionType, unit.Action.Tasks[0].WorkRange, unit.Action.Tasks[0].CombatLevel)
	}
}

func TestPatchUnitRebuildsCountPrefixedListsGolden(t *testing.T) {
	path := testfixtures.Path(t, "reference_aoe2_dump/dat/empires2_x2_p1.dat")
	compressed, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	before, err := Parse(mustInflate(t, compressed))
	if err != nil {
		t.Fatal(err)
	}
	beforeUnit, err := unitByID(before, 1, 74)
	if err != nil {
		t.Fatal(err)
	}
	if beforeUnit.Action == nil || len(beforeUnit.Action.Tasks) < 2 {
		t.Fatalf("unit 74 action tasks = %+v, want at least 2", beforeUnit.Action)
	}
	damageGraphics := []DamageGraphicRow{{
		GraphicID:     123,
		DamagePercent: 50,
		Flag:          1,
	}}
	attacks := []WeaponInfoRow{
		{Class: 4, Value: 12},
		{Class: 3, Value: 7},
	}
	armours := []WeaponInfoRow{{
		Class: 4,
		Value: 2,
	}}
	trainLocations := []TrainLocationRow{
		{TrainTime: 19, TrainUnitID: 74, TrainButton: 2, TrainHotkey: 16121},
		{TrainTime: 5, TrainUnitID: 83, TrainButton: 3, TrainHotkey: 16122},
	}
	dropSites := []int16{109, 110}
	tasks := []TaskRow{
		taskRowFromSummary(beforeUnit.Action.Tasks[0]),
		taskRowFromSummary(beforeUnit.Action.Tasks[1]),
		taskRowFromSummary(beforeUnit.Action.Tasks[0]),
	}
	tasks[2].ActionType++
	tasks[2].WorkRange += 1.5

	out, report, err := PatchUnit(compressed, 1, 74, UnitPatch{
		SetDamageGraphics: &damageGraphics,
		SetType50Attacks:  &attacks,
		SetType50Armours:  &armours,
		SetTrainLocations: &trainLocations,
		SetDropSites:      &dropSites,
		SetTasks:          &tasks,
	})
	if err != nil {
		t.Fatalf("PatchUnit: %v", err)
	}
	if len(out) == 0 || !report.Verified {
		t.Fatalf("patch report = %+v", report)
	}
	if report.InflatedLengthDelta == 0 {
		t.Fatalf("inflated length delta = 0, want count-changing record rebuild")
	}

	idx, err := Parse(mustInflate(t, out))
	if err != nil {
		t.Fatalf("parse output: %v", err)
	}
	unit, err := unitByID(idx, 1, 74)
	if err != nil {
		t.Fatal(err)
	}
	if len(unit.DamageGraphics) != len(damageGraphics) ||
		unit.DamageGraphics[0].GraphicID != damageGraphics[0].GraphicID ||
		unit.DamageGraphics[0].DamagePercent != damageGraphics[0].DamagePercent ||
		unit.DamageGraphics[0].Flag != damageGraphics[0].Flag {
		t.Fatalf("damage graphics readback = %+v", unit.DamageGraphics)
	}
	if len(unit.Type50.Attacks) != len(attacks) ||
		unit.Type50.Attacks[0].Class != attacks[0].Class ||
		unit.Type50.Attacks[1].Value != attacks[1].Value {
		t.Fatalf("attacks readback = %+v", unit.Type50.Attacks)
	}
	if len(unit.Type50.Armours) != len(armours) ||
		unit.Type50.Armours[0].Class != armours[0].Class ||
		unit.Type50.Armours[0].Value != armours[0].Value {
		t.Fatalf("armours readback = %+v", unit.Type50.Armours)
	}
	if len(unit.Creatable.TrainLocations) != len(trainLocations) ||
		unit.Creatable.TrainLocations[0].TrainHotkey != trainLocations[0].TrainHotkey ||
		unit.Creatable.TrainLocations[1].TrainUnitID != trainLocations[1].TrainUnitID {
		t.Fatalf("train locations readback = %+v", unit.Creatable.TrainLocations)
	}
	if unit.Action == nil || len(unit.Action.DropSites) != len(dropSites) || unit.Action.DropSites[0] != dropSites[0] || unit.Action.DropSites[1] != dropSites[1] {
		t.Fatalf("drop sites readback = %+v", unit.Action)
	}
	if unit.Action == nil || len(unit.Action.Tasks) != len(tasks) ||
		unit.Action.Tasks[2].ActionType != tasks[2].ActionType ||
		unit.Action.Tasks[2].WorkRange != tasks[2].WorkRange ||
		!bytes.Equal(unit.Action.Tasks[2].RawTail, tasks[2].RawTail) {
		t.Fatalf("tasks readback = %+v", unit.Action)
	}
}

func TestRecipeUnitExpandedRowsGolden(t *testing.T) {
	path := testfixtures.Path(t, "reference_aoe2_dump/dat/empires2_x2_p1.dat")
	compressed, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	class := int16(12)
	los := float32(8.25)
	movement := uint8(2)
	attrAmount := float32(4.5)
	attackValue := int16(11)
	armourValue := int16(6)
	costAmount := int16(80)
	trainTime := int16(23)
	createHP := int16(456)
	createAttackValue := int16(13)
	createCostAmount := int16(99)
	createArmours := []WeaponInfoRow{{Class: 4, Value: 5}}
	createTrainLocations := []TrainLocationRow{{TrainTime: 31, TrainUnitID: 74, TrainButton: 4, TrainHotkey: 16123}}
	createDropSites := []int16{109, 110, 562}
	taskAction := int16(7)
	taskWorkRange := float32(3.5)
	taskCombatLevel := uint8(1)
	createTaskAction := int16(109)
	out, report, err := PatchRecipe(compressed, Recipe{
		Units: []UnitRecipePatch{{
			CivID:        1,
			UnitID:       74,
			Class:        &class,
			LineOfSight:  &los,
			MovementType: &movement,
			Attributes: []UnitAttributePatch{{
				Index:  1,
				Amount: &attrAmount,
			}},
			Type50Attacks: []WeaponInfoPatch{{
				Index: 2,
				Value: &attackValue,
			}},
			Type50Armours: []WeaponInfoPatch{{
				Index: 2,
				Value: &armourValue,
			}},
			Costs: []AttributeCostPatch{{
				Index:  0,
				Amount: &costAmount,
			}},
			TrainLocations: []TrainLocationPatch{{
				Index:     0,
				TrainTime: &trainTime,
			}},
			Tasks: []TaskPatch{{
				Index:       0,
				ActionType:  &taskAction,
				WorkRange:   &taskWorkRange,
				CombatLevel: &taskCombatLevel,
			}},
		}},
		CreateUnit: &UnitCreateRecipe{
			FromCivID:  1,
			FromUnitID: 74,
			CivIDs:     []int{1},
			HitPoints:  &createHP,
			Type50Attacks: []WeaponInfoPatch{{
				Index: 2,
				Value: &createAttackValue,
			}},
			SetType50Armours:  &createArmours,
			SetTrainLocations: &createTrainLocations,
			SetDropSites:      &createDropSites,
			Costs: []AttributeCostPatch{{
				Index:  0,
				Amount: &createCostAmount,
			}},
			Tasks: []TaskPatch{{
				Index:      0,
				ActionType: &createTaskAction,
			}},
		},
	})
	if err != nil {
		t.Fatalf("PatchRecipe: %v", err)
	}
	if len(out) == 0 || !report.Verified || len(report.UnitReports) != 1 || len(report.CreatedUnits) != 1 {
		t.Fatalf("recipe report = %+v", report)
	}
	if report.UnitReports[0].InflatedLengthDelta != 0 || report.CreatedUnits[0].NewUnitID <= 0 {
		t.Fatalf("recipe unit reports = patch %+v create %+v", report.UnitReports[0], report.CreatedUnits[0])
	}
	idx, err := Parse(mustInflate(t, out))
	if err != nil {
		t.Fatalf("parse output: %v", err)
	}
	unit, err := unitByID(idx, 1, 74)
	if err != nil {
		t.Fatal(err)
	}
	if unit.Class != class || unit.LineOfSight != los || unit.MovementType != movement {
		t.Fatalf("patched scalar fields = class %d los %f movement %d", unit.Class, unit.LineOfSight, unit.MovementType)
	}
	if unit.Attributes[1].Amount != attrAmount ||
		unit.Type50.Attacks[2].Value != attackValue ||
		unit.Type50.Armours[2].Value != armourValue ||
		unit.Creatable.Costs[0].Amount != costAmount ||
		unit.Creatable.TrainLocations[0].TrainTime != trainTime ||
		unit.Action.Tasks[0].ActionType != taskAction ||
		unit.Action.Tasks[0].WorkRange != taskWorkRange ||
		unit.Action.Tasks[0].CombatLevel != taskCombatLevel {
		t.Fatalf("patched row fields = attr %f attack %d armour %d cost %d train %d task_action %d task_range %f task_combat %d",
			unit.Attributes[1].Amount,
			unit.Type50.Attacks[2].Value,
			unit.Type50.Armours[2].Value,
			unit.Creatable.Costs[0].Amount,
			unit.Creatable.TrainLocations[0].TrainTime,
			unit.Action.Tasks[0].ActionType,
			unit.Action.Tasks[0].WorkRange,
			unit.Action.Tasks[0].CombatLevel)
	}
	created, err := unitByID(idx, 1, report.CreatedUnits[0].NewUnitID)
	if err != nil {
		t.Fatal(err)
	}
	if created.HitPoints != createHP ||
		created.Type50.Attacks[2].Value != createAttackValue ||
		len(created.Type50.Armours) != len(createArmours) ||
		created.Type50.Armours[0].Value != createArmours[0].Value ||
		created.Creatable.Costs[0].Amount != createCostAmount ||
		len(created.Creatable.TrainLocations) != len(createTrainLocations) ||
		created.Creatable.TrainLocations[0].TrainHotkey != createTrainLocations[0].TrainHotkey ||
		len(created.Action.DropSites) != len(createDropSites) ||
		created.Action.DropSites[2] != createDropSites[2] ||
		created.Action.Tasks[0].ActionType != createTaskAction {
		t.Fatalf("created unit fields = hp %d attack %d armours %+v cost %d train %+v drop_sites %+v task_action %d", created.HitPoints, created.Type50.Attacks[2].Value, created.Type50.Armours, created.Creatable.Costs[0].Amount, created.Creatable.TrainLocations, created.Action.DropSites, created.Action.Tasks[0].ActionType)
	}
}

func taskRowFromSummary(task TaskSummary) TaskRow {
	return TaskRow{
		RecordType:        task.RecordType,
		ID:                task.ID,
		IsDefault:         task.IsDefault,
		ActionType:        task.ActionType,
		ObjectClass:       task.ObjectClass,
		ObjectID:          task.ObjectID,
		TerrainID:         task.TerrainID,
		AttributeTypes:    append([]int16(nil), task.AttributeTypes...),
		WorkValue1:        task.WorkValue1,
		WorkValue2:        task.WorkValue2,
		WorkRange:         task.WorkRange,
		AutoSearchTargets: task.AutoSearchTargets,
		SearchWaitTime:    task.SearchWaitTime,
		EnableTargeting:   task.EnableTargeting,
		CombatLevel:       task.CombatLevel,
		RawTail:           append([]byte(nil), task.RawTail...),
	}
}

func mustInflate(t testing.TB, compressed []byte) []byte {
	t.Helper()
	payload, err := Inflate(compressed)
	if err != nil {
		t.Fatal(err)
	}
	return payload
}
