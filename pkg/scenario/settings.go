package scenario

import "fmt"

type SettingsReport struct {
	Path               string                    `json:"path,omitempty"`
	Version            string                    `json:"version"`
	Verification       string                    `json:"verification"`
	PlayerCount        int                       `json:"player_count"`
	Players            []PlayerSettings          `json:"players,omitempty"`
	Options            OptionsSettings           `json:"options"`
	Map                MapSettings               `json:"map"`
	Diplomacy          DiplomacySettings         `json:"diplomacy"`
	DiplomacyMirrors   []DiplomacyMirrorSettings `json:"diplomacy_mirrors,omitempty"`
	Messages           MessagesSettings          `json:"messages"`
	Cinematics         CinematicsSettings        `json:"cinematics"`
	Victory            map[string]int            `json:"victory,omitempty"`
	VictoryModeName    string                    `json:"victory_mode_name,omitempty"`
	SecondaryGameModes SecondaryGameModesInfo    `json:"secondary_game_modes"`
	Resources          []PlayerResourceValues    `json:"resources,omitempty"`
}

type MapSettings struct {
	WaterDefinition string `json:"water_definition"`
}

type PlayerSettings struct {
	Player            int                        `json:"player"`
	PlayerLabel       string                     `json:"player_label,omitempty"`
	SectionIndex      int                        `json:"section_index"`
	IndexBase         string                     `json:"index_base,omitempty"`
	FieldReferences   map[string]PlayerReference `json:"field_references,omitempty"`
	Active            bool                       `json:"active"`
	Human             bool                       `json:"human"`
	TribeName         string                     `json:"tribe_name,omitempty"`
	NameStringID      int                        `json:"name_string_id"`
	Civilization      string                     `json:"civilization,omitempty"`
	Architecture      string                     `json:"architecture,omitempty"`
	LockCivilization  bool                       `json:"lock_civilization"`
	LockPersonality   bool                       `json:"lock_personality"`
	StartingAge       int                        `json:"starting_age"`
	StartingAgeName   string                     `json:"starting_age_name,omitempty"`
	Color             int                        `json:"color"`
	ColorName         string                     `json:"color_name,omitempty"`
	BasePriority      int                        `json:"base_priority"`
	PopulationLimit   int                        `json:"population_limit"`
	AIName            string                     `json:"ai_name,omitempty"`
	AIType            int                        `json:"ai_type,omitempty"`
	Resources         PlayerResourceValues       `json:"resources"`
	Diplomacy         []int                      `json:"diplomacy,omitempty"`
	AlliedVictory     bool                       `json:"allied_victory"`
	DisabledTechs     []int                      `json:"disabled_techs,omitempty"`
	DisabledUnits     []int                      `json:"disabled_units,omitempty"`
	DisabledBuildings []int                      `json:"disabled_buildings,omitempty"`
}

type PlayerResourceValues struct {
	Player     int `json:"player"`
	Gold       int `json:"gold"`
	Wood       int `json:"wood"`
	Food       int `json:"food"`
	Stone      int `json:"stone"`
	TradeGoods int `json:"trade_goods"`
}

type DiplomacySettings struct {
	Matrix                  [][]int `json:"matrix,omitempty"`
	AlliedVictory           []bool  `json:"allied_victory,omitempty"`
	LockTeams               bool    `json:"lock_teams"`
	AllowPlayersChooseTeams bool    `json:"allow_players_choose_teams"`
	RandomStartPoints       bool    `json:"random_start_points"`
	MaxNumberOfTeams        int     `json:"max_number_of_teams"`
}

type DiplomacyMirrorSettings struct {
	Player        int    `json:"player"`
	PlayerLabel   string `json:"player_label"`
	SectionIndex  int    `json:"section_index"`
	IndexBase     string `json:"index_base"`
	Interaction   []int  `json:"interaction,omitempty"`
	AISystem      []int  `json:"ai_system,omitempty"`
	AlliedVictory int    `json:"allied_victory"`
}

type OptionsSettings struct {
	FullTechTree              bool                 `json:"full_tech_tree"`
	CollideAndCorrecting      bool                 `json:"collide_and_correcting"`
	VillagerForceDrop         bool                 `json:"villager_force_drop"`
	LockCoopAlliances         bool                 `json:"lock_coop_alliances"`
	TriggerExecutionOrder     int                  `json:"trigger_execution_order"`
	TriggerExecutionOrderName string               `json:"trigger_execution_order_name"`
	InitialPlayerViews        []PlayerViewSettings `json:"initial_player_views,omitempty"`
}

type PlayerViewSettings struct {
	Player int  `json:"player"`
	X      int  `json:"x"`
	Y      int  `json:"y"`
	Set    bool `json:"set"`
}

type MessagesSettings struct {
	Instructions MessageSlotSettings `json:"instructions"`
	Hints        MessageSlotSettings `json:"hints"`
	Victory      MessageSlotSettings `json:"victory"`
	Loss         MessageSlotSettings `json:"loss"`
	History      MessageSlotSettings `json:"history"`
	Scouts       MessageSlotSettings `json:"scouts"`
}

type MessageSlotSettings struct {
	Literal           string `json:"literal,omitempty"`
	StringTableID     int    `json:"string_table_id"`
	UsesStringTableID bool   `json:"uses_string_table_id"`
	EffectiveSource   string `json:"effective_source"`
}

type CinematicsSettings struct {
	Pregame string `json:"pregame,omitempty"`
	Victory string `json:"victory,omitempty"`
	Loss    string `json:"loss,omitempty"`
}

func SettingsFile(path string) (SettingsReport, error) {
	file, err := Open(path)
	if err != nil {
		return SettingsReport{}, err
	}
	report := file.Settings()
	report.Path = path
	return report, nil
}

func (f *File) Settings() SettingsReport {
	options := f.optionsSettings()
	disabledTechs, disabledUnits, disabledBuildings := f.disabledSettings()
	report := SettingsReport{
		Path:               f.Path,
		Version:            f.Version,
		Verification:       "structure_verified_not_engine_verified",
		PlayerCount:        f.PlayerCount,
		Players:            append([]PlayerSettings(nil), f.playerSettings()...),
		Options:            options,
		Map:                f.mapSettings(),
		Diplomacy:          f.diplomacySettings(),
		DiplomacyMirrors:   f.diplomacyMirrorSettings(),
		Messages:           f.messagesSettings(),
		Cinematics:         f.cinematicsSettings(),
		Victory:            f.victorySettings(),
		VictoryModeName:    victoryModeName(f.victorySettings()["mode"]),
		SecondaryGameModes: secondaryGameModesSettings(f),
		Resources:          f.resourceSettings(),
	}
	for i := range report.Players {
		if i < len(report.Resources) {
			report.Players[i].Resources = report.Resources[i]
		}
		if i < len(report.Diplomacy.Matrix) {
			report.Players[i].Diplomacy = report.Diplomacy.Matrix[i]
		}
		if i < len(report.Diplomacy.AlliedVictory) {
			report.Players[i].AlliedVictory = report.Diplomacy.AlliedVictory[i]
		}
		if i < len(disabledTechs) {
			report.Players[i].DisabledTechs = disabledTechs[i]
		}
		if i < len(disabledUnits) {
			report.Players[i].DisabledUnits = disabledUnits[i]
		}
		if i < len(disabledBuildings) {
			report.Players[i].DisabledBuildings = disabledBuildings[i]
		}
	}
	return report
}

func (f *File) mapSettings() MapSettings {
	mapSection := f.root.section("Map")
	if mapSection == nil {
		return MapSettings{}
	}
	waterDefinition, _ := mapSection.stringValue("water_definition")
	return MapSettings{WaterDefinition: waterDefinition}
}

func victoryModeName(mode int) string {
	switch mode {
	case 0:
		return "Standard Victory"
	case 6:
		return "Secondary Game Mode"
	default:
		return fmt.Sprintf("Unknown (%d)", mode)
	}
}

func secondaryGameModesSettings(f *File) SecondaryGameModesInfo {
	if f.Map != nil {
		return f.Map.SecondaryGameModes
	}
	return SecondaryGameModesInfo{}
}

func secondaryGameModeNames(bits int) []string {
	known := []struct {
		bit  int
		name string
	}{
		{1, "Empire Wars"},
		{2, "Sudden Death"},
		{4, "Regicide"},
		{8, "King of the Hill"},
	}
	names := make([]string, 0, len(known))
	remaining := bits
	for _, mode := range known {
		if bits&mode.bit != 0 {
			names = append(names, mode.name)
			remaining &^= mode.bit
		}
	}
	for bit := 1; remaining != 0; bit <<= 1 {
		if remaining&bit != 0 {
			names = append(names, fmt.Sprintf("Unknown bit %d", bit))
			remaining &^= bit
		}
	}
	return names
}

func (f *File) diplomacyMirrorSettings() []DiplomacyMirrorSettings {
	units := f.root.section("Units")
	if units == nil {
		return nil
	}
	rows := units.list("player_data_3")
	out := make([]DiplomacyMirrorSettings, 0, len(rows))
	for i, row := range rows {
		out = append(out, DiplomacyMirrorSettings{
			Player:        i,
			PlayerLabel:   fmt.Sprintf("P%d", i+1),
			SectionIndex:  i,
			IndexBase:     "players_from_1",
			Interaction:   row.intList("diplomacy_for_interaction"),
			AISystem:      row.intList("diplomacy_for_ai_system"),
			AlliedVictory: intValueOrZero(row, "aok_allied_victory"),
		})
	}
	return out
}

func intValueOrZero(node *parsedNode, field string) int {
	value, _ := node.intValue(field)
	return value
}

func (f *File) messagesSettings() MessagesSettings {
	messages := f.root.section("Messages")
	if messages == nil {
		return MessagesSettings{}
	}
	return MessagesSettings{
		Instructions: messageSlotSettings(messages, "instructions"),
		Hints:        messageSlotSettings(messages, "hints"),
		Victory:      messageSlotSettings(messages, "victory"),
		Loss:         messageSlotSettings(messages, "loss"),
		History:      messageSlotSettings(messages, "history"),
		Scouts:       messageSlotSettings(messages, "scouts"),
	}
}

func messageSlotSettings(messages *parsedNode, name string) MessageSlotSettings {
	id, ok := signedMessageID(messages, name)
	if !ok {
		id = -2
	}
	literal, _ := messages.stringValue("ascii_" + name)
	source := "unset"
	usesID := id >= 0
	if usesID {
		source = "string_table_id"
	} else if literal != "" {
		source = "literal"
	}
	return MessageSlotSettings{
		Literal:           literal,
		StringTableID:     id,
		UsesStringTableID: usesID,
		EffectiveSource:   source,
	}
}

func signedMessageID(node *parsedNode, name string) (int, bool) {
	field := node.field(name)
	if field == nil {
		return 0, false
	}
	switch value := field.Value.(type) {
	case int:
		return value, true
	case int32:
		return int(value), true
	case uint32:
		return int(int32(value)), true
	default:
		return node.intValue(name)
	}
}

func (f *File) cinematicsSettings() CinematicsSettings {
	cinematics := f.root.section("Cinematics")
	if cinematics == nil {
		return CinematicsSettings{}
	}
	pregame, _ := cinematics.stringValue("ascii_pregame")
	victory, _ := cinematics.stringValue("ascii_victory")
	loss, _ := cinematics.stringValue("ascii_loss")
	return CinematicsSettings{Pregame: pregame, Victory: victory, Loss: loss}
}

func (f *File) playerSettings() []PlayerSettings {
	out := make([]PlayerSettings, 0, len(f.Players))
	for _, player := range f.Players {
		out = append(out, PlayerSettings{
			Player:           player.Player + 1,
			PlayerLabel:      player.PlayerLabel,
			SectionIndex:     player.SectionIndex,
			IndexBase:        player.IndexBase,
			FieldReferences:  player.FieldReferences,
			Active:           player.Active,
			Human:            player.Human,
			TribeName:        player.TribeName,
			NameStringID:     player.NameStringID,
			Civilization:     player.Civilization,
			Architecture:     player.Architecture,
			LockCivilization: player.LockCivilization,
			LockPersonality:  player.LockPersonality,
			StartingAge:      player.StartingAge,
			StartingAgeName:  player.StartingAgeName,
			Color:            player.Color,
			ColorName:        player.ColorName,
			BasePriority:     player.BasePriority,
			PopulationLimit:  player.PopulationLimit,
			AIName:           player.AIName,
			AIType:           player.AIType,
		})
	}
	return out
}

func (f *File) resourceSettings() []PlayerResourceValues {
	playerDataTwo := f.root.section("PlayerDataTwo")
	if playerDataTwo == nil {
		return nil
	}
	resources := playerDataTwo.list("resources")
	out := make([]PlayerResourceValues, 0, len(resources))
	for i, resource := range resources {
		gold, _ := resource.intValue("gold")
		wood, _ := resource.intValue("wood")
		food, _ := resource.intValue("food")
		stone, _ := resource.intValue("stone")
		tradeGoods, _ := resource.intValue("trade_goods")
		out = append(out, PlayerResourceValues{
			Player:     i + 1,
			Gold:       gold,
			Wood:       wood,
			Food:       food,
			Stone:      stone,
			TradeGoods: tradeGoods,
		})
	}
	return out
}

func (f *File) diplomacySettings() DiplomacySettings {
	diplomacy := f.root.section("Diplomacy")
	if diplomacy == nil {
		return DiplomacySettings{}
	}
	rows := diplomacy.list("per_player_diplomacy")
	matrix := make([][]int, 0, len(rows))
	for _, row := range rows {
		matrix = append(matrix, row.intList("stance_with_each_player"))
	}
	alliedRaw := diplomacy.intList("per_player_allied_victory")
	allied := make([]bool, 0, len(alliedRaw))
	for _, value := range alliedRaw {
		allied = append(allied, value != 0)
	}
	lockTeams, _ := diplomacy.intValue("lock_teams")
	allowTeams, _ := diplomacy.intValue("allow_players_choose_teams")
	randomStarts, _ := diplomacy.intValue("random_start_points")
	maxTeams, _ := diplomacy.intValue("max_number_of_teams")
	return DiplomacySettings{
		Matrix:                  matrix,
		AlliedVictory:           allied,
		LockTeams:               lockTeams != 0,
		AllowPlayersChooseTeams: allowTeams != 0,
		RandomStartPoints:       randomStarts != 0,
		MaxNumberOfTeams:        maxTeams,
	}
}

func (f *File) optionsSettings() OptionsSettings {
	options := f.root.section("Options")
	mapSection := f.root.section("Map")
	triggers := f.root.section("Triggers")
	var out OptionsSettings
	if options != nil {
		if value, ok := options.intValue("all_techs"); ok {
			out.FullTechTree = value != 0
		}
	}
	if mapSection != nil {
		if value, ok := mapSection.intValue("collide_and_correct"); ok {
			out.CollideAndCorrecting = value != 0
		}
		if value, ok := mapSection.intValue("villager_force_drop"); ok {
			out.VillagerForceDrop = value != 0
		}
		if value, ok := mapSection.intValue("lock_coop_alliances"); ok {
			out.LockCoopAlliances = value != 0
		}
		for i, view := range mapSection.list("initial_player_views") {
			x, _ := view.intValue("location_x")
			y, _ := view.intValue("location_y")
			out.InitialPlayerViews = append(out.InitialPlayerViews, PlayerViewSettings{
				Player: i,
				X:      x,
				Y:      y,
				Set:    x != -1 || y != -1,
			})
		}
	}
	if triggers != nil {
		if value, ok := triggers.intValue("legacy_exec_order"); ok {
			out.TriggerExecutionOrder = value
			if value == 0 {
				out.TriggerExecutionOrderName = "Legacy"
			} else if value == 1 {
				out.TriggerExecutionOrderName = "New Behaviour"
			}
		}
	}
	return out
}

func (f *File) disabledSettings() ([][]int, [][]int, [][]int) {
	options := f.root.section("Options")
	if options == nil {
		return nil, nil, nil
	}
	return disabledLists(options, "disabled_tech_ids_player_"),
		disabledLists(options, "disabled_unit_ids_player_"),
		disabledLists(options, "disabled_building_ids_player_")
}

func disabledLists(options *parsedNode, prefix string) [][]int {
	playerSuffixes := []string{"1", "2", "3", "4", "5", "6", "7", "8"}
	out := make([][]int, len(playerSuffixes))
	for i, suffix := range playerSuffixes {
		out[i] = options.intList(prefix + suffix)
	}
	return out
}

func (f *File) victorySettings() map[string]int {
	globalVictory := f.root.section("GlobalVictory")
	if globalVictory == nil {
		return nil
	}
	fields := []string{
		"conquest_required",
		"ruins",
		"artifacts_required",
		"discovery",
		"explored_percent_of_map_required",
		"gold_required",
		"all_custom_conditions_required",
		"mode",
		"required_score_for_score_victory",
		"time_for_timed_game_in_10ths_of_a_year",
	}
	out := map[string]int{}
	for _, field := range fields {
		value, ok := globalVictory.intValue(field)
		if ok {
			out[field] = value
		}
	}
	return out
}
