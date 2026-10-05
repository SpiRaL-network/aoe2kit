package scenario

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const blankScenarioSeedSHA256 = "ffc08c5d34d194e3a20fe1baf5f87a7d0e2cbea484f4a6e29db23ca5e1b091c5"
const defaultBlankDummyUnit = 12

type BlankOptions struct {
	PlayerCount        int
	HumanSlots         int
	MapWidth           int
	MapHeight          int
	Timestamp          int
	ClearTriggers      bool
	KeepSeedTriggers   bool
	DummyStarters      bool
	NoStarters         bool // Explicit editor-empty opt-out; playable blanks include starters.
	DummyUnit          int
	DummyStartX        float64
	DummyStartY        float64
	DummySpacing       float64
	StartingResources  bool
	StartingFood       int
	StartingWood       int
	StartingGold       int
	StartingStone      int
	StartingTradeGoods int
	InactiveRestHuman  bool
	GaiaActive         bool
	NoConquest         bool
	CloseoutSeconds    int
	CloseoutPlayer     int
}

type BlankReport struct {
	Output             string `json:"output"`
	SeedSHA256         string `json:"seed_sha256"`
	Version            string `json:"version"`
	PlayerCount        int    `json:"player_count"`
	HumanSlots         int    `json:"human_slots"`
	MapWidth           int    `json:"map_width"`
	MapHeight          int    `json:"map_height"`
	TileCount          int    `json:"tile_count"`
	ClearTriggers      bool   `json:"clear_triggers"`
	DummyStarters      bool   `json:"dummy_starters"`
	DummyUnit          int    `json:"dummy_unit,omitempty"`
	GaiaActive         bool   `json:"gaia_active"`
	NoConquest         bool   `json:"no_conquest"`
	CloseoutSeconds    int    `json:"closeout_seconds,omitempty"`
	CloseoutPlayer     int    `json:"closeout_player,omitempty"`
	TriggerCountBefore int    `json:"trigger_count_before"`
	TriggerCountAfter  int    `json:"trigger_count_after"`
	UnitCountBefore    int    `json:"unit_count_before"`
	UnitCountAfter     int    `json:"unit_count_after"`
	RebuildOK          bool   `json:"rebuild_ok"`
	InvariantOK        bool   `json:"invariant_ok"`
	EditorParityOK     bool   `json:"editor_parity_ok"`
	EditorParityNote   string `json:"editor_parity_note,omitempty"`
	Verification       string `json:"verification"`
}

func BlankScenarioSeedBytes() ([]byte, error) {
	data, err := hex.DecodeString(blankScenarioSeedHex)
	if err != nil {
		return nil, fmt.Errorf("decode embedded blank scenario seed: %w", err)
	}
	sum := sha256.Sum256(data)
	if got := hex.EncodeToString(sum[:]); got != blankScenarioSeedSHA256 {
		return nil, fmt.Errorf("embedded blank scenario seed sha256=%s, want %s", got, blankScenarioSeedSHA256)
	}
	return data, nil
}

func WriteBlankScenarioFile(output string, opts BlankOptions) (BlankReport, error) {
	opts = normalizeBlankOptions(opts)
	if opts.PlayerCount < 1 || opts.PlayerCount > 8 {
		return BlankReport{}, fmt.Errorf("blank scenario player count %d out of range 1..8", opts.PlayerCount)
	}
	if opts.HumanSlots < 0 || opts.HumanSlots > opts.PlayerCount {
		return BlankReport{}, fmt.Errorf("blank scenario human slots %d out of range 0..player_count", opts.HumanSlots)
	}
	if err := ValidateScenarioMapSize(opts.MapWidth, opts.MapHeight); err != nil {
		return BlankReport{}, err
	}
	if err := validateBlankDummyPositions(opts); err != nil {
		return BlankReport{}, err
	}
	if err := validateBlankCloseout(opts); err != nil {
		return BlankReport{}, err
	}
	seed, err := BlankScenarioSeedBytes()
	if err != nil {
		return BlankReport{}, err
	}
	file, err := Parse(seed)
	if err != nil {
		return BlankReport{}, fmt.Errorf("parse embedded blank scenario seed: %w", err)
	}
	beforeTriggers := 0
	if file.Triggers != nil {
		beforeTriggers = file.Triggers.Count
	}
	beforeUnits := 0
	if file.Units != nil {
		beforeUnits = file.Units.Total
	}
	recipe := blankScenarioRecipe(opts)
	if err := file.ApplyRecipe(recipe); err != nil {
		return BlankReport{}, err
	}
	if err := file.ResizeMap(opts.MapWidth, opts.MapHeight); err != nil {
		return BlankReport{}, err
	}
	if dir := filepath.Dir(output); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return BlankReport{}, err
		}
	}
	if err := file.Write(output); err != nil {
		return BlankReport{}, err
	}
	verified, err := Open(output)
	if err != nil {
		return BlankReport{}, err
	}
	if err := verified.VerifyRebuild(); err != nil {
		return BlankReport{}, err
	}
	afterTriggers := 0
	invariantOK := false
	if verified.Triggers != nil {
		afterTriggers = verified.Triggers.Count
		invariantOK = verified.Triggers.InvariantOK
	}
	afterUnits := 0
	if verified.Units != nil {
		afterUnits = verified.Units.Total
	}
	mapWidth, mapHeight, tileCount := 0, 0, 0
	if verified.Map != nil {
		mapWidth = verified.Map.Width
		mapHeight = verified.Map.Height
		tileCount = verified.Map.TileCount
	}
	if !invariantOK {
		return BlankReport{}, fmt.Errorf("blank scenario trigger invariants failed: %s", verified.Triggers.InvariantNote)
	}
	editorParityOK, editorParityNote := editorParityPlayerInit(verified, opts.PlayerCount)
	return BlankReport{
		Output:             output,
		SeedSHA256:         blankScenarioSeedSHA256,
		Version:            verified.Version,
		PlayerCount:        max(2, opts.PlayerCount),
		HumanSlots:         opts.HumanSlots,
		MapWidth:           mapWidth,
		MapHeight:          mapHeight,
		TileCount:          tileCount,
		ClearTriggers:      opts.ClearTriggers,
		DummyStarters:      opts.DummyStarters,
		DummyUnit:          opts.DummyUnit,
		GaiaActive:         opts.GaiaActive,
		NoConquest:         opts.NoConquest,
		CloseoutSeconds:    opts.CloseoutSeconds,
		CloseoutPlayer:     opts.CloseoutPlayer,
		TriggerCountBefore: beforeTriggers,
		TriggerCountAfter:  afterTriggers,
		UnitCountBefore:    beforeUnits,
		UnitCountAfter:     afterUnits,
		RebuildOK:          true,
		InvariantOK:        invariantOK,
		EditorParityOK:     editorParityOK,
		EditorParityNote:   editorParityNote,
		Verification:       scenarioWriteVerification().Label,
	}, nil
}

func normalizeBlankOptions(opts BlankOptions) BlankOptions {
	opts.DummyStarters = !opts.NoStarters
	if opts.PlayerCount == 0 {
		opts.PlayerCount = 1
	}
	if opts.HumanSlots == 0 {
		opts.HumanSlots = 1
	}
	if opts.MapWidth == 0 {
		opts.MapWidth = 120
	}
	if opts.MapHeight == 0 {
		opts.MapHeight = 120
	}
	if opts.Timestamp == 0 {
		opts.Timestamp = int(time.Now().Unix())
	}
	if opts.DummyUnit == 0 {
		opts.DummyUnit = defaultBlankDummyUnit
	}
	if opts.DummyStartX == 0 {
		opts.DummyStartX = 5.5
	}
	if opts.DummyStartY == 0 {
		opts.DummyStartY = 5.5
	}
	if opts.DummySpacing == 0 {
		opts.DummySpacing = 6
	}
	opts.GaiaActive = true
	opts.ClearTriggers = !opts.KeepSeedTriggers
	if opts.CloseoutSeconds > 0 && opts.CloseoutPlayer == 0 {
		opts.CloseoutPlayer = 1
	}
	return opts
}

func editorParityPlayerInit(file *File, playableSlots int) (bool, string) {
	if playableSlots < 2 {
		playableSlots = 2
	}
	if file == nil {
		return false, "missing scenario"
	}
	if len(file.Players) < 16 {
		return false, fmt.Sprintf("parsed %d player records; expected 16", len(file.Players))
	}
	for _, player := range file.Players {
		wantActive := player.Player >= 0 && player.Player < playableSlots
		if player.Active != wantActive {
			return false, fmt.Sprintf("%s active=%t want %t for editor-style blank player init", playerLabel(player.Player), player.Active, wantActive)
		}
		if !player.Human {
			return false, fmt.Sprintf("%s human=false; editor-authored blank keeps all player records human metadata unless explicitly changed", playerLabel(player.Player))
		}
	}
	return true, "player active/human metadata matches editor-style blank baseline"
}

func validateBlankDummyPositions(opts BlankOptions) error {
	if !opts.DummyStarters {
		return nil
	}
	for player := 1; player <= max(2, opts.PlayerCount); player++ {
		x := opts.DummyStartX + float64(player-1)*opts.DummySpacing
		y := opts.DummyStartY
		if x < 0 || y < 0 || x >= float64(opts.MapWidth) || y >= float64(opts.MapHeight) {
			return fmt.Errorf("blank dummy starter P%d at %.2f,%.2f is outside map %dx%d; adjust --dummy-origin/--dummy-spacing or use a larger map", player, x, y, opts.MapWidth, opts.MapHeight)
		}
	}
	return nil
}

func validateBlankCloseout(opts BlankOptions) error {
	if opts.CloseoutSeconds < 0 {
		return fmt.Errorf("blank closeout seconds %d must be >= 0", opts.CloseoutSeconds)
	}
	if opts.CloseoutSeconds == 0 {
		return nil
	}
	if opts.CloseoutPlayer < 1 || opts.CloseoutPlayer > opts.PlayerCount {
		return fmt.Errorf("blank closeout player %d out of range 1..player_count", opts.CloseoutPlayer)
	}
	return nil
}

func blankScenarioRecipe(opts BlankOptions) Recipe {
	active := true
	inactive := false
	human := true
	computer := false
	timestamp := opts.Timestamp
	playerCount := opts.PlayerCount
	if playerCount < 2 {
		playerCount = 2
	}
	recipe := Recipe{
		Scenario: &ScenarioRecipe{
			PlayerCount:         &playerCount,
			TimestampOfLastSave: &timestamp,
		},
	}
	if opts.ClearTriggers {
		recipe.Triggers = append(recipe.Triggers, TriggerRecipe{Op: "clear_triggers"})
	}
	for player := 1; player <= 16; player++ {
		playerActive := &inactive
		if player <= playerCount {
			playerActive = &active
		}
		playerHuman := &human
		if player <= opts.PlayerCount && player > opts.HumanSlots {
			playerHuman = &computer
		}
		recipe.Players = append(recipe.Players, PlayerRecipe{
			Player: player,
			Active: playerActive,
			Human:  playerHuman,
		})
	}
	if opts.DummyStarters {
		status := 2
		rotation := 0.0
		z := 0.0
		for player := 1; player <= playerCount; player++ {
			x := opts.DummyStartX + float64(player-1)*opts.DummySpacing
			y := opts.DummyStartY
			caption := fmt.Sprintf("A2K_BLANK_DUMMY_P%d", player)
			recipe.Units = append(recipe.Units, UnitRecipe{
				Op:            "add_unit",
				Player:        player,
				UnitConst:     opts.DummyUnit,
				X:             &x,
				Y:             &y,
				Z:             &z,
				Status:        &status,
				Rotation:      &rotation,
				CaptionString: caption,
			})
		}
	}
	if opts.StartingResources {
		food, wood, gold, stone, tradeGoods := opts.StartingFood, opts.StartingWood, opts.StartingGold, opts.StartingStone, opts.StartingTradeGoods
		// PlayerDataTwo.resources is indexed by active player slot: index 0 is
		// P1. This differs from Units, where index 0 is Gaia and playable
		// ownership starts at 1.
		for player := 1; player <= playerCount; player++ {
			recipe.Resources = append(recipe.Resources, ResourceRecipe{
				Player: player, Food: &food, Wood: &wood, Gold: &gold, Stone: &stone, TradeGoods: &tradeGoods,
			})
		}
	}
	if opts.NoConquest {
		conquest := 0
		recipe.Victory = &VictoryRecipe{ConquestRequired: &conquest}
	}
	if opts.CloseoutSeconds > 0 {
		timer := opts.CloseoutSeconds
		closeoutPlayer := opts.CloseoutPlayer
		recipe.Triggers = append(recipe.Triggers, TriggerRecipe{
			Op:      "add_trigger",
			Name:    "A2K Blank Closeout - declare victory",
			Enabled: &active,
			Looping: &inactive,
			Conditions: []ConditionRecipe{
				{Op: "timer", Timer: &timer},
			},
			Effects: []EffectRecipe{
				{Op: "declare_victory", SourcePlayer: &closeoutPlayer},
			},
		})
	}
	return recipe
}

const blankScenarioSeedHex = "" +
	"312e35380000000006000000a3329d6a010000000002000000e8030000010000001100000002000000030000000400000005000000060000000700000008000000090000000a0000000c0000000d0000000e0000000f000000100000001100000012000000130000000600000063687261650000000000ecdbcd4e13511cc0d15bcab7a2266e5c4ee2c605ba202c310a68a20b3fa2896b9a52a4513b5a1ae303f8129af808ae7c09e243f00c26eec5b9a54d4ae95c509a10f4fc929b93726fe7df0eb09c50f4eef6f73b570a67e30b4992244992f4cff76b7f7fff34ab525c23ae8df9f9f06cf5f1bd278f6eae3f7c71f8d564efcc49cec59c73ee4fce9d65f133765b0eebed46add3c876ea8d56adddcc17b3cd3c6be59ded66ebe562567fddacbfca766aef1bb76a7963a97fa8ecdf6ad4a8eaa8b12aa932d49792bb7aec75127b1389bdc1dfd670a93fdca9c4de74626f26b1379bd89b4beccd27f62e24f62e26f616127b97127b978b157f87c7fde57f2c6efc8deb07f7dfb2acff7349922449922449922449922449922449d279e96f9f77395425f9a8cab92b3eaf739a15ef69bfb2191bc54333fd15c2c01bcea4b05b7c88d3acd8c28faf7bdd479d3ef456ef8b010000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000070ae982b2cefc1fabfb017bfe3b5f0bcde68d5dacd3cbbbfd9ece4edece976add5c9df84dda56f6b7bcb3f5757c24a0873a11aaa07c5f7568a359958b1adade9bb4786c65b5b3e3284b76b71f547f667c677c6cba6c6c6bd718ceccdec8f4c8d9d18d7c883998323cbc6c643e319d99d393cb26cec9846c699a3460e8f9d1ae3c86266d9c8c1b1d3e31cd99d98fc0789636752238ffc70449f3fc5ae1ebd80249da4df0300"
