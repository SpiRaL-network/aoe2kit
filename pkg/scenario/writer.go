package scenario

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"aoe2kit/pkg/aoe2"
	"aoe2kit/pkg/datcache"
)

type Recipe struct {
	DatPath          string                  `json:"dat_path,omitempty"`
	Scenario         *ScenarioRecipe         `json:"scenario,omitempty"`
	XS               *XSRecipe               `json:"xs,omitempty"`
	Victory          *VictoryRecipe          `json:"victory,omitempty"`
	Players          []PlayerRecipe          `json:"players,omitempty"`
	Diplomacy        []DiplomacyRecipe       `json:"diplomacy,omitempty"`
	DiplomacyOptions *DiplomacyOptionsRecipe `json:"diplomacy_options,omitempty"`
	Resources        []ResourceRecipe        `json:"resources,omitempty"`
	Strings          []StringRecipe          `json:"strings,omitempty"`
	Variables        []VariableRecipe        `json:"variables,omitempty"`
	Triggers         []TriggerRecipe         `json:"triggers"`
	Masks            []MaskRecipe            `json:"masks,omitempty"`
	Units            []UnitRecipe            `json:"units,omitempty"`
	Map              []MapRecipe             `json:"map,omitempty"`
}

type ScenarioRecipe struct {
	PlayerCount         *int `json:"player_count,omitempty"`
	TimestampOfLastSave *int `json:"timestamp_of_last_save,omitempty"`
	// StartingAge applies the editor's per-player starting-age field to all
	// playable slots. It is intentionally one value: a recipe that needs
	// asymmetric ages can use the lower-level scenario authoring API.
	StartingAge *int `json:"starting_age,omitempty"`
}

const (
	MinScenarioMapSize = 80
	MaxScenarioMapSize = 480
)

type ScenarioMapPreset struct {
	Name      string `json:"name"`
	Size      int    `json:"size"`
	StringID  int    `json:"string_id"`
	StringKey string `json:"string_key"`
}

// scenarioMapPresets is grounded in the shipped DE string table keys
// MAPSIZE_* at 25080..25480. The low three digits encode the editor map edge.
var scenarioMapPresets = []ScenarioMapPreset{
	{Name: "Mini", Size: 80, StringID: 25080, StringKey: "MAPSIZE_MINI"},
	{Name: "Tiny", Size: 120, StringID: 25120, StringKey: "MAPSIZE_TINY"},
	{Name: "Small", Size: 144, StringID: 25144, StringKey: "MAPSIZE_SMALL"},
	{Name: "Medium", Size: 168, StringID: 25168, StringKey: "MAPSIZE_MEDIUM"},
	{Name: "Normal", Size: 200, StringID: 25200, StringKey: "MAPSIZE_NORMAL"},
	{Name: "Large", Size: 220, StringID: 25220, StringKey: "MAPSIZE_LARGE"},
	{Name: "Huge", Size: 240, StringID: 25240, StringKey: "MAPSIZE_HUGE"},
	{Name: "Giant", Size: 252, StringID: 25252, StringKey: "MAPSIZE_GIANT"},
	{Name: "Massive", Size: 276, StringID: 25276, StringKey: "MAPSIZE_MASSIVE"},
	{Name: "Enormous", Size: 300, StringID: 25300, StringKey: "MAPSIZE_ENORMOUS"},
	{Name: "Colossal", Size: 320, StringID: 25320, StringKey: "MAPSIZE_COLOSSAL"},
	{Name: "Incredible", Size: 360, StringID: 25360, StringKey: "MAPSIZE_INCREDIBLE"},
	{Name: "Monstrous", Size: 400, StringID: 25400, StringKey: "MAPSIZE_MONSTROUS"},
	{Name: "Ludicrous", Size: 480, StringID: 25480, StringKey: "MAPSIZE_LUDICROUS"},
}

type XSRecipe struct {
	Name                string `json:"name,omitempty"`
	Content             string `json:"content,omitempty"`
	ContentFile         string `json:"content_file,omitempty"`
	Mode                string `json:"mode,omitempty"`
	CarrierTitle        string `json:"carrier_title,omitempty"`
	CarrierTriggerName  string `json:"carrier_trigger_name,omitempty"`
	CarrierTriggerIndex *int   `json:"carrier_trigger_index,omitempty"`
	ReplaceCarrier      *bool  `json:"replace_carrier,omitempty"`
	ClearAttachment     *bool  `json:"clear_attachment,omitempty"`
}

type VictoryRecipe struct {
	ConquestRequired               *int `json:"conquest_required,omitempty"`
	Ruins                          *int `json:"ruins,omitempty"`
	ArtifactsRequired              *int `json:"artifacts_required,omitempty"`
	Discovery                      *int `json:"discovery,omitempty"`
	ExploredPercentOfMapRequired   *int `json:"explored_percent_of_map_required,omitempty"`
	GoldRequired                   *int `json:"gold_required,omitempty"`
	AllCustomConditionsRequired    *int `json:"all_custom_conditions_required,omitempty"`
	Mode                           *int `json:"mode,omitempty"`
	RequiredScoreForScoreVictory   *int `json:"required_score_for_score_victory,omitempty"`
	TimeForTimedGameIn10thsOfAYear *int `json:"time_for_timed_game_in_10ths_of_a_year,omitempty"`
}

type PlayerRecipe struct {
	Player           int     `json:"player"`
	Active           *bool   `json:"active,omitempty"`
	Human            *bool   `json:"human,omitempty"`
	TribeName        *string `json:"tribe_name,omitempty"`
	Civilization     *string `json:"civilization,omitempty"`
	LockCivilization *bool   `json:"lock_civilization,omitempty"`
	LockPersonality  *bool   `json:"lock_personality,omitempty"`
	AIName           *string `json:"ai_name,omitempty"`
	AIType           *int    `json:"ai_type,omitempty"`
}

type DiplomacyRecipe struct {
	From   int `json:"from"`
	To     int `json:"to"`
	Stance int `json:"stance"`
}

type DiplomacyOptionsRecipe struct {
	LockTeams               *bool                 `json:"lock_teams,omitempty"`
	AllowPlayersChooseTeams *bool                 `json:"allow_players_choose_teams,omitempty"`
	RandomStartPoints       *bool                 `json:"random_start_points,omitempty"`
	MaxNumberOfTeams        *int                  `json:"max_number_of_teams,omitempty"`
	AlliedVictory           []AlliedVictoryRecipe `json:"allied_victory,omitempty"`
}

type AlliedVictoryRecipe struct {
	Player  int  `json:"player"`
	Enabled bool `json:"enabled"`
}

type ResourceRecipe struct {
	Player     int  `json:"player"`
	Gold       *int `json:"gold,omitempty"`
	Wood       *int `json:"wood,omitempty"`
	Food       *int `json:"food,omitempty"`
	Stone      *int `json:"stone,omitempty"`
	TradeGoods *int `json:"trade_goods,omitempty"`
}

type VariableRecipe struct {
	Op         string  `json:"op"`
	ID         *int    `json:"id,omitempty"`
	Name       string  `json:"name,omitempty"`
	TargetID   *int    `json:"target_id,omitempty"`
	TargetName string  `json:"target_name,omitempty"`
	SetName    *string `json:"set_name,omitempty"`
}

type StringRecipe struct {
	Op       string  `json:"op"`
	ID       *int    `json:"id,omitempty"`
	Text     string  `json:"text,omitempty"`
	SetText  *string `json:"set_text,omitempty"`
	OldText  string  `json:"old_text,omitempty"`
	Required *bool   `json:"required,omitempty"`
}

type MaskRecipe struct {
	Name       string           `json:"name"`
	Op         string           `json:"op"`
	X1         int              `json:"x1,omitempty"`
	Y1         int              `json:"y1,omitempty"`
	X2         int              `json:"x2,omitempty"`
	Y2         int              `json:"y2,omitempty"`
	Radius     *int             `json:"radius,omitempty"`
	TerrainIDs []int            `json:"terrain_ids,omitempty"`
	Masks      []string         `json:"masks,omitempty"`
	Seed       *int             `json:"seed,omitempty"`
	Scale      *float64         `json:"scale,omitempty"`
	Threshold  *float64         `json:"threshold,omitempty"`
	Iterations *int             `json:"iterations,omitempty"`
	Invert     *bool            `json:"invert,omitempty"`
	Cells      []MaskCellRecipe `json:"cells,omitempty"`
}

type MaskCellRecipe struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type MapRecipe struct {
	Op         string   `json:"op"`
	Mask       string   `json:"mask,omitempty"`
	X1         int      `json:"x1"`
	Y1         int      `json:"y1"`
	X2         int      `json:"x2"`
	Y2         int      `json:"y2"`
	Width      *int     `json:"width,omitempty"`
	Height     *int     `json:"height,omitempty"`
	TargetX    *int     `json:"target_x,omitempty"`
	TargetY    *int     `json:"target_y,omitempty"`
	Radius     *int     `json:"radius,omitempty"`
	Thickness  *int     `json:"thickness,omitempty"`
	TerrainID  *int     `json:"terrain_id,omitempty"`
	TerrainID2 *int     `json:"terrain_id_2,omitempty"`
	Elevation  *int     `json:"elevation,omitempty"`
	Layer      *int     `json:"layer,omitempty"`
	Seed       *int     `json:"seed,omitempty"`
	Scale      *float64 `json:"scale,omitempty"`
	Threshold  *float64 `json:"threshold,omitempty"`
	Iterations *int     `json:"iterations,omitempty"`
	TerrainIDs []int    `json:"terrain_ids,omitempty"`
	GridFile   string   `json:"grid_file,omitempty"`

	TerrainIDGrid []int `json:"-"`
	LayerGrid     []int `json:"-"`
	ElevationGrid []int `json:"-"`
	baseDir       string
}

func (recipe *MapRecipe) UnmarshalJSON(data []byte) error {
	type wire struct {
		Op         string          `json:"op"`
		Mask       string          `json:"mask,omitempty"`
		X1         int             `json:"x1"`
		Y1         int             `json:"y1"`
		X2         int             `json:"x2"`
		Y2         int             `json:"y2"`
		Width      *int            `json:"width,omitempty"`
		Height     *int            `json:"height,omitempty"`
		TargetX    *int            `json:"target_x,omitempty"`
		TargetY    *int            `json:"target_y,omitempty"`
		Radius     *int            `json:"radius,omitempty"`
		Thickness  *int            `json:"thickness,omitempty"`
		TerrainID  json.RawMessage `json:"terrain_id,omitempty"`
		TerrainID2 *int            `json:"terrain_id_2,omitempty"`
		Elevation  json.RawMessage `json:"elevation,omitempty"`
		Layer      json.RawMessage `json:"layer,omitempty"`
		Seed       *int            `json:"seed,omitempty"`
		Scale      *float64        `json:"scale,omitempty"`
		Threshold  *float64        `json:"threshold,omitempty"`
		Iterations *int            `json:"iterations,omitempty"`
		TerrainIDs []int           `json:"terrain_ids,omitempty"`
		GridFile   string          `json:"grid_file,omitempty"`
	}
	var raw wire
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	terrainID, terrainGrid, err := decodeMapRecipeIntOrGrid(raw.TerrainID, "terrain_id")
	if err != nil {
		return err
	}
	elevation, elevationGrid, err := decodeMapRecipeIntOrGrid(raw.Elevation, "elevation")
	if err != nil {
		return err
	}
	layer, layerGrid, err := decodeMapRecipeIntOrGrid(raw.Layer, "layer")
	if err != nil {
		return err
	}
	*recipe = MapRecipe{
		Op:            raw.Op,
		Mask:          raw.Mask,
		X1:            raw.X1,
		Y1:            raw.Y1,
		X2:            raw.X2,
		Y2:            raw.Y2,
		Width:         raw.Width,
		Height:        raw.Height,
		TargetX:       raw.TargetX,
		TargetY:       raw.TargetY,
		Radius:        raw.Radius,
		Thickness:     raw.Thickness,
		TerrainID:     terrainID,
		TerrainID2:    raw.TerrainID2,
		Elevation:     elevation,
		Layer:         layer,
		Seed:          raw.Seed,
		Scale:         raw.Scale,
		Threshold:     raw.Threshold,
		Iterations:    raw.Iterations,
		TerrainIDs:    raw.TerrainIDs,
		GridFile:      raw.GridFile,
		TerrainIDGrid: terrainGrid,
		LayerGrid:     layerGrid,
		ElevationGrid: elevationGrid,
	}
	return nil
}

func (recipe MapRecipe) MarshalJSON() ([]byte, error) {
	type wire struct {
		Op         string   `json:"op"`
		Mask       string   `json:"mask,omitempty"`
		X1         int      `json:"x1"`
		Y1         int      `json:"y1"`
		X2         int      `json:"x2"`
		Y2         int      `json:"y2"`
		Width      *int     `json:"width,omitempty"`
		Height     *int     `json:"height,omitempty"`
		TargetX    *int     `json:"target_x,omitempty"`
		TargetY    *int     `json:"target_y,omitempty"`
		Radius     *int     `json:"radius,omitempty"`
		Thickness  *int     `json:"thickness,omitempty"`
		TerrainID  any      `json:"terrain_id,omitempty"`
		TerrainID2 *int     `json:"terrain_id_2,omitempty"`
		Elevation  any      `json:"elevation,omitempty"`
		Layer      any      `json:"layer,omitempty"`
		Seed       *int     `json:"seed,omitempty"`
		Scale      *float64 `json:"scale,omitempty"`
		Threshold  *float64 `json:"threshold,omitempty"`
		Iterations *int     `json:"iterations,omitempty"`
		TerrainIDs []int    `json:"terrain_ids,omitempty"`
		GridFile   string   `json:"grid_file,omitempty"`
	}
	var terrainID any
	if recipe.TerrainID != nil {
		terrainID = *recipe.TerrainID
	}
	if len(recipe.TerrainIDGrid) > 0 {
		terrainID = recipe.TerrainIDGrid
	}
	var elevation any
	if recipe.Elevation != nil {
		elevation = *recipe.Elevation
	}
	if len(recipe.ElevationGrid) > 0 {
		elevation = recipe.ElevationGrid
	}
	var layer any
	if recipe.Layer != nil {
		layer = *recipe.Layer
	}
	if len(recipe.LayerGrid) > 0 {
		layer = recipe.LayerGrid
	}
	return json.Marshal(wire{
		Op:         recipe.Op,
		Mask:       recipe.Mask,
		X1:         recipe.X1,
		Y1:         recipe.Y1,
		X2:         recipe.X2,
		Y2:         recipe.Y2,
		Width:      recipe.Width,
		Height:     recipe.Height,
		TargetX:    recipe.TargetX,
		TargetY:    recipe.TargetY,
		Radius:     recipe.Radius,
		Thickness:  recipe.Thickness,
		TerrainID:  terrainID,
		TerrainID2: recipe.TerrainID2,
		Elevation:  elevation,
		Layer:      layer,
		Seed:       recipe.Seed,
		Scale:      recipe.Scale,
		Threshold:  recipe.Threshold,
		Iterations: recipe.Iterations,
		TerrainIDs: recipe.TerrainIDs,
		GridFile:   recipe.GridFile,
	})
}

func decodeMapRecipeIntOrGrid(raw json.RawMessage, name string) (*int, []int, error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return nil, nil, nil
	}
	if raw[0] == '[' {
		var values []int
		if err := json.Unmarshal(raw, &values); err != nil {
			return nil, nil, fmt.Errorf("%s array: %w", name, err)
		}
		return nil, values, nil
	}
	var value int
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, nil, fmt.Errorf("%s scalar: %w", name, err)
	}
	return &value, nil, nil
}

type UnitRecipe struct {
	Op                    string        `json:"op"`
	Player                int           `json:"player"`
	UnitConst             int           `json:"unit_const"`
	X                     *float64      `json:"x,omitempty"`
	Y                     *float64      `json:"y,omitempty"`
	Z                     *float64      `json:"z,omitempty"`
	ReferenceID           *int          `json:"reference_id,omitempty"`
	TargetPlayer          *int          `json:"target_player,omitempty"`
	TargetIndex           *int          `json:"target_index,omitempty"`
	TargetCaption         string        `json:"target_caption,omitempty"`
	TargetUnitConst       *int          `json:"target_unit_const,omitempty"`
	TargetAreaX1          *float64      `json:"target_area_x1,omitempty"`
	TargetAreaY1          *float64      `json:"target_area_y1,omitempty"`
	TargetAreaX2          *float64      `json:"target_area_x2,omitempty"`
	TargetAreaY2          *float64      `json:"target_area_y2,omitempty"`
	TargetX               *float64      `json:"target_x,omitempty"`
	TargetY               *float64      `json:"target_y,omitempty"`
	OffsetX               *float64      `json:"offset_x,omitempty"`
	OffsetY               *float64      `json:"offset_y,omitempty"`
	ReferenceIDBase       *int          `json:"reference_id_base,omitempty"`
	Count                 *int          `json:"count,omitempty"`
	Centers               []PointRecipe `json:"centers,omitempty"`
	ClusterShape          string        `json:"cluster_shape,omitempty"`
	Snap                  string        `json:"snap,omitempty"`
	Distribution          string        `json:"distribution,omitempty"`
	Jitter                *float64      `json:"jitter,omitempty"`
	Spread                *float64      `json:"spread,omitempty"`
	MinDistance           *float64      `json:"min_distance,omitempty"`
	AllowedTerrainIDs     []int         `json:"allowed_terrain_ids,omitempty"`
	Exact                 *bool         `json:"exact,omitempty"`
	Seed                  *int          `json:"seed,omitempty"`
	CaptionSuffix         string        `json:"caption_suffix,omitempty"`
	SetPlayer             *int          `json:"set_player,omitempty"`
	Status                *int          `json:"status,omitempty"`
	Rotation              *float64      `json:"rotation,omitempty"`
	RotationChoices       []float64     `json:"rotation_choices,omitempty"`
	InitialAnimationFrame *int          `json:"initial_animation_frame,omitempty"`
	GarrisonedInID        *int          `json:"garrisoned_in_id,omitempty"`
	CaptionStringID       *int          `json:"caption_string_id,omitempty"`
	CaptionString         string        `json:"caption_string,omitempty"`
}

type PointRecipe struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type TriggerRecipe struct {
	Op                        string            `json:"op"`
	Name                      string            `json:"name"`
	Message                   string            `json:"message"`
	Description               string            `json:"description,omitempty"`
	ShortDescription          string            `json:"short_description,omitempty"`
	TargetIndex               *int              `json:"target_index,omitempty"`
	TargetIndexes             []int             `json:"target_indexes,omitempty"`
	TargetName                string            `json:"target_name,omitempty"`
	TargetPrefix              string            `json:"target_prefix,omitempty"`
	SetName                   *string           `json:"set_name,omitempty"`
	DescriptionStringID       *int              `json:"description_string_table_id,omitempty"`
	ShortDescriptionStringID  *int              `json:"short_description_string_table_id,omitempty"`
	DisplayAsObjective        *bool             `json:"display_as_objective,omitempty"`
	DisplayOnScreen           *bool             `json:"display_on_screen,omitempty"`
	MakeHeader                *bool             `json:"make_header,omitempty"`
	MuteObjectives            *bool             `json:"mute_objectives,omitempty"`
	ExecuteOnLoad             *bool             `json:"execute_on_load,omitempty"`
	ObjectiveDescriptionOrder *int              `json:"objective_description_order,omitempty"`
	Enabled                   *bool             `json:"enabled,omitempty"`
	Looping                   *bool             `json:"looping,omitempty"`
	RemoveEffects             []int             `json:"remove_effects,omitempty"`
	RemoveConditions          []int             `json:"remove_conditions,omitempty"`
	ClearEffects              *bool             `json:"clear_effects,omitempty"`
	ClearConditions           *bool             `json:"clear_conditions,omitempty"`
	ReplaceEffects            []EffectRecipe    `json:"replace_effects,omitempty"`
	ReplaceConditions         []ConditionRecipe `json:"replace_conditions,omitempty"`
	Effects                   []EffectRecipe    `json:"effects,omitempty"`
	Conditions                []ConditionRecipe `json:"conditions,omitempty"`
	DisplayTime               *int              `json:"display_time,omitempty"`
	InstructionPanelPosition  *int              `json:"instruction_panel_position,omitempty"`
	SourcePlayer              *int              `json:"source_player,omitempty"`
	PlaySound                 *int              `json:"play_sound,omitempty"`
	UseTagColorForIcon        *int              `json:"use_tag_color_for_icon,omitempty"`
	ObjectListUnitID          *int              `json:"object_list_unit_id,omitempty"`
	LocationX                 *int              `json:"location_x,omitempty"`
	LocationY                 *int              `json:"location_y,omitempty"`
	ItemID                    *int              `json:"item_id,omitempty"`
	Facet                     *int              `json:"facet,omitempty"`
	DisableSound              *int              `json:"disable_sound,omitempty"`
}

type EffectRecipe struct {
	Op                         string   `json:"op"`
	Message                    string   `json:"message,omitempty"`
	ObjectListUnitID           *int     `json:"object_list_unit_id,omitempty"`
	ObjectListUnitID2          *int     `json:"object_list_unit_id_2,omitempty"`
	SourcePlayer               *int     `json:"source_player,omitempty"`
	TargetPlayer               *int     `json:"target_player,omitempty"`
	TriggerID                  *int     `json:"trigger_id,omitempty"`
	Technology                 *int     `json:"technology,omitempty"`
	Diplomacy                  *int     `json:"diplomacy,omitempty"`
	LocationX                  *int     `json:"location_x,omitempty"`
	LocationY                  *int     `json:"location_y,omitempty"`
	LocationObjectReference    *int     `json:"location_object_reference,omitempty"`
	AreaX1                     *int     `json:"area_x1,omitempty"`
	AreaY1                     *int     `json:"area_y1,omitempty"`
	AreaX2                     *int     `json:"area_x2,omitempty"`
	AreaY2                     *int     `json:"area_y2,omitempty"`
	ObjectGroup                *int     `json:"object_group,omitempty"`
	ObjectType                 *int     `json:"object_type,omitempty"`
	Facet2                     *int     `json:"facet2,omitempty"`
	ActionType                 *int     `json:"action_type,omitempty"`
	AttackStance               *int     `json:"attack_stance,omitempty"`
	MaxUnitsAffected           *int     `json:"max_units_affected,omitempty"`
	IssueGroupCommand          *int     `json:"issue_group_command,omitempty"`
	QueueAction                *int     `json:"queue_action,omitempty"`
	DisableGarrisonUnloadSound *int     `json:"disable_garrison_unload_sound,omitempty"`
	DisplayTime                *int     `json:"display_time,omitempty"`
	InstructionPanelPosition   *int     `json:"instruction_panel_position,omitempty"`
	PlaySound                  *int     `json:"play_sound,omitempty"`
	SoundName                  string   `json:"sound_name,omitempty"`
	UseTagColorForIcon         *int     `json:"use_tag_color_for_icon,omitempty"`
	ItemID                     *int     `json:"item_id,omitempty"`
	Quantity                   *int     `json:"quantity,omitempty"`
	QuantityFloat              *float64 `json:"quantity_float,omitempty"`
	Operation                  *int     `json:"operation,omitempty"`
	OperationName              string   `json:"operation_name,omitempty"`
	TributeList                *int     `json:"tribute_list,omitempty"`
	Resource                   *int     `json:"resource,omitempty"`
	Resource1                  *int     `json:"resource_1,omitempty"`
	Resource1Quantity          *int     `json:"resource_1_quantity,omitempty"`
	Resource2                  *int     `json:"resource_2,omitempty"`
	Resource2Quantity          *int     `json:"resource_2_quantity,omitempty"`
	Resource3                  *int     `json:"resource_3,omitempty"`
	Resource3Quantity          *int     `json:"resource_3_quantity,omitempty"`
	StringID                   *int     `json:"string_id,omitempty"`
	ForceResearchTechnology    *int     `json:"force_research_technology,omitempty"`
	VisibilityState            *int     `json:"visibility_state,omitempty"`
	Scroll                     *int     `json:"scroll,omitempty"`
	FlashObject                *int     `json:"flash_object,omitempty"`
	ObjectAttributes           *int     `json:"object_attributes,omitempty"`
	ObjectState                *int     `json:"object_state,omitempty"`
	Facet                      *int     `json:"facet,omitempty"`
	DisableSound               *int     `json:"disable_sound,omitempty"`
	Enabled                    *int     `json:"enabled,omitempty"`
	ButtonLocation             *int     `json:"button_location,omitempty"`
	Hotkey                     *int     `json:"hotkey,omitempty"`
	TrainTime                  *int     `json:"train_time,omitempty"`
	LocalTechnology            *int     `json:"local_technology,omitempty"`
	PlayerColor                *int     `json:"player_color,omitempty"`
	GlobalSound                *int     `json:"global_sound,omitempty"`
	MutualDiplomacy            *int     `json:"mutual_diplomacy,omitempty"`
	ObjectFilter               *int     `json:"object_filter,omitempty"`
	TimeUnit                   *int     `json:"time_unit,omitempty"`
	TimerID                    *int     `json:"timer_id,omitempty"`
	ResetTimer                 *int     `json:"reset_timer,omitempty"`
	Variable                   *int     `json:"variable,omitempty"`
	Variable2                  *int     `json:"variable2,omitempty"`
	XSFunction                 string   `json:"xs_function,omitempty"`
	SelectedObjectIDs          []int    `json:"selected_object_ids,omitempty"`
}

type ConditionRecipe struct {
	Op                             string `json:"op"`
	Timer                          *int   `json:"timer,omitempty"`
	UnitObject                     *int   `json:"unit_object,omitempty"`
	Quantity                       *int   `json:"quantity,omitempty"`
	Attribute                      *int   `json:"attribute,omitempty"`
	ObjectList                     *int   `json:"object_list,omitempty"`
	SourcePlayer                   *int   `json:"source_player,omitempty"`
	Technology                     *int   `json:"technology,omitempty"`
	AreaX1                         *int   `json:"area_x1,omitempty"`
	AreaY1                         *int   `json:"area_y1,omitempty"`
	AreaX2                         *int   `json:"area_x2,omitempty"`
	AreaY2                         *int   `json:"area_y2,omitempty"`
	ObjectGroup                    *int   `json:"object_group,omitempty"`
	ObjectType                     *int   `json:"object_type,omitempty"`
	ObjectState                    *int   `json:"object_state,omitempty"`
	Variable                       *int   `json:"variable,omitempty"`
	Comparison                     *int   `json:"comparison,omitempty"`
	TargetPlayer                   *int   `json:"target_player,omitempty"`
	UnitAIAction                   *int   `json:"unit_ai_action,omitempty"`
	IncludeChangeableWeaponObjects *int   `json:"include_changeable_weapon_objects,omitempty"`
	Inverted                       *int   `json:"inverted,omitempty"`
}

type Plan struct {
	TriggerCountBefore int            `json:"trigger_count_before"`
	TriggerCountAfter  int            `json:"trigger_count_after"`
	UnitCountBefore    int            `json:"unit_count_before,omitempty"`
	UnitCountAfter     int            `json:"unit_count_after,omitempty"`
	MapTilesChanged    int            `json:"map_tiles_changed,omitempty"`
	Operations         []PlanOp       `json:"operations"`
	Warnings           []string       `json:"warnings,omitempty"`
	Recipe             map[string]any `json:"recipe,omitempty"`
}

type PlanOp struct {
	Op      string `json:"op"`
	Name    string `json:"name,omitempty"`
	Enabled *bool  `json:"enabled,omitempty"`
	Message string `json:"message,omitempty"`
}

type PatchReport struct {
	Input               string                 `json:"input"`
	Output              string                 `json:"output"`
	TriggerCountBefore  int                    `json:"trigger_count_before"`
	TriggerCountAfter   int                    `json:"trigger_count_after"`
	UnitCountBefore     int                    `json:"unit_count_before,omitempty"`
	UnitCountAfter      int                    `json:"unit_count_after,omitempty"`
	MapTilesChanged     int                    `json:"map_tiles_changed,omitempty"`
	TimestampOfLastSave int                    `json:"timestamp_of_last_save,omitempty"`
	RebuildOK           bool                   `json:"rebuild_ok"`
	InvariantOK         bool                   `json:"invariant_ok"`
	Verification        aoe2.VerificationClaim `json:"verification"`
}

type SmokeRecipeOptions struct {
	Player       int
	PlayerSet    bool
	UnitConst    int
	UnitConstSet bool
	X            int
	XSet         bool
	Y            int
	YSet         bool
	TerrainID    int
	TerrainIDSet bool
	Elevation    int
	ElevationSet bool
	Layer        int
	LayerSet     bool
}

func LoadRecipe(path string) (Recipe, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Recipe{}, err
	}
	var recipe Recipe
	if err := json.Unmarshal(data, &recipe); err != nil {
		return Recipe{}, err
	}
	baseDir := filepath.Dir(path)
	if recipe.DatPath != "" && !filepath.IsAbs(recipe.DatPath) {
		recipe.DatPath = filepath.Join(baseDir, recipe.DatPath)
	}
	for i := range recipe.Map {
		recipe.Map[i].baseDir = baseDir
	}
	return recipe, nil
}

func PlanRecipeFile(input string, recipe Recipe) (Plan, error) {
	file, err := Open(input)
	if err != nil {
		return Plan{}, err
	}
	return file.Plan(recipe)
}

func PatchRecipeFile(input, output string, recipe Recipe) (PatchReport, error) {
	file, err := Open(input)
	if err != nil {
		return PatchReport{}, err
	}
	if !SupportsWriteVersion(file.Version) {
		return PatchReport{}, fmt.Errorf("scenario version %q is read-only in AoE2Kit; writing currently targets DE 1.57 through 1.59", file.Version)
	}
	before := 0
	if file.Triggers != nil {
		before = file.Triggers.Count
	}
	unitsBefore := 0
	if file.Units != nil {
		unitsBefore = file.Units.Total
	}
	plan, err := file.Plan(recipe)
	if err != nil {
		return PatchReport{}, err
	}
	if err := file.ApplyRecipe(recipe); err != nil {
		return PatchReport{}, err
	}
	timestamp := recipeTimestamp(recipe)
	if err := file.SetScenario(ScenarioRecipe{TimestampOfLastSave: &timestamp}); err != nil {
		return PatchReport{}, err
	}
	if err := file.Write(output); err != nil {
		return PatchReport{}, err
	}
	verified, err := Open(output)
	if err != nil {
		return PatchReport{}, err
	}
	if err := verified.VerifyRebuild(); err != nil {
		return PatchReport{}, err
	}
	after := 0
	invariantOK := false
	if verified.Triggers != nil {
		after = verified.Triggers.Count
		invariantOK = verified.Triggers.InvariantOK
	}
	unitsAfter := 0
	if verified.Units != nil {
		unitsAfter = verified.Units.Total
	}
	if !invariantOK {
		return PatchReport{}, fmt.Errorf("patched scenario trigger invariants failed: %s", verified.Triggers.InvariantNote)
	}
	return PatchReport{
		Input:               input,
		Output:              output,
		TriggerCountBefore:  before,
		TriggerCountAfter:   after,
		UnitCountBefore:     unitsBefore,
		UnitCountAfter:      unitsAfter,
		MapTilesChanged:     plan.MapTilesChanged,
		TimestampOfLastSave: timestamp,
		RebuildOK:           true,
		InvariantOK:         invariantOK,
		Verification:        scenarioWriteVerification(),
	}, nil
}

func recipeTimestamp(recipe Recipe) int {
	if recipe.Scenario != nil && recipe.Scenario.TimestampOfLastSave != nil {
		return *recipe.Scenario.TimestampOfLastSave
	}
	return int(time.Now().Unix())
}

func (f *File) writeSpec() (*Spec, error) {
	if f == nil {
		return LoadCurrentDESpec()
	}
	spec, err := LoadDESpecForVersion(f.Version)
	if err != nil {
		return nil, err
	}
	if f.Version == "1.59" {
		patchDE159EditorConditionSpec(spec)
	}
	return spec, nil
}

func SmokeRecipe(opts SmokeRecipeOptions) Recipe {
	if !opts.PlayerSet {
		opts.Player = 1
	}
	if !opts.UnitConstSet {
		opts.UnitConst = 83
	}
	if !opts.XSet {
		opts.X = 10
	}
	if !opts.YSet {
		opts.Y = 10
	}
	if !opts.TerrainIDSet {
		opts.TerrainID = 47
	}
	if !opts.LayerSet {
		opts.Layer = -1
	}
	enabled := false
	sourcePlayer := opts.Player
	displayTime := 8
	createX := opts.X + 1
	createY := opts.Y + 1
	unitX := float64(opts.X) + 1.5
	unitY := float64(opts.Y) + 1.5
	unitZ := float64(0)
	rotation := float64(0)
	status := 2
	timer := 5
	actionType := 1
	triggerZero := 0
	terrain := opts.TerrainID
	elevation := opts.Elevation
	layer := opts.Layer
	return Recipe{
		Triggers: []TriggerRecipe{
			{
				Op:      "add_trigger",
				Name:    "AOE2KIT WRITE SMOKE - display/chat/timer",
				Enabled: &enabled,
				Conditions: []ConditionRecipe{
					{Op: "timer", Timer: &timer},
				},
				Effects: []EffectRecipe{
					{Op: "display_instructions", SourcePlayer: &sourcePlayer, Message: "AOE2KIT WRITE SMOKE: display-instructions written by Go.", DisplayTime: &displayTime},
					{Op: "send_chat", SourcePlayer: &sourcePlayer, Message: "AOE2KIT WRITE SMOKE: send-chat written by Go."},
					{Op: "display_timer", SourcePlayer: &sourcePlayer, Message: "AOE2KIT WRITE SMOKE TIMER", DisplayTime: &displayTime},
				},
			},
			{
				Op:      "add_trigger",
				Name:    "AOE2KIT WRITE SMOKE - create/task/kill/remove",
				Enabled: &enabled,
				Effects: []EffectRecipe{
					{Op: "create_object", SourcePlayer: &sourcePlayer, ObjectListUnitID: &opts.UnitConst, LocationX: &createX, LocationY: &createY},
					{Op: "task_object", SourcePlayer: &sourcePlayer, ActionType: &actionType, LocationX: &createX, LocationY: &createY},
					{Op: "kill_object", SourcePlayer: &sourcePlayer, ObjectListUnitID: &opts.UnitConst, AreaX1: &opts.X, AreaY1: &opts.Y, AreaX2: &createX, AreaY2: &createY},
					{Op: "remove_object", SourcePlayer: &sourcePlayer, ObjectListUnitID: &opts.UnitConst, AreaX1: &opts.X, AreaY1: &opts.Y, AreaX2: &createX, AreaY2: &createY},
				},
			},
			{
				Op:      "add_trigger",
				Name:    "AOE2KIT WRITE SMOKE - trigger control",
				Enabled: &enabled,
				Effects: []EffectRecipe{
					{Op: "activate_trigger", TriggerID: &triggerZero},
					{Op: "deactivate_trigger", TriggerID: &triggerZero},
				},
			},
		},
		Units: []UnitRecipe{
			{
				Op:        "add_unit",
				Player:    opts.Player,
				UnitConst: opts.UnitConst,
				X:         &unitX,
				Y:         &unitY,
				Z:         &unitZ,
				Status:    &status,
				Rotation:  &rotation,
			},
		},
		Map: []MapRecipe{
			{
				Op:        "set_terrain_rect",
				X1:        opts.X,
				Y1:        opts.Y,
				X2:        opts.X + 2,
				Y2:        opts.Y + 2,
				TerrainID: &terrain,
				Elevation: &elevation,
				Layer:     &layer,
			},
		},
	}
}

func PatchSmokeRecipeFile(input, output string, opts SmokeRecipeOptions) (PatchReport, error) {
	return PatchRecipeFile(input, output, SmokeRecipe(opts))
}

func scenarioWriteVerification() aoe2.VerificationClaim {
	return aoe2.VerificationClaim{
		Label:             "structure_verified_not_engine_verified",
		StructureVerified: true,
		EngineVerified:    false,
		Note:              "scenario writer reparsed the patched scenario, verified body rebuild and trigger invariants, and checked declared counts; DE editor/game load remains the external oracle.",
	}
}

func (f *File) Plan(recipe Recipe) (Plan, error) {
	if f.Triggers == nil {
		return Plan{}, fmt.Errorf("scenario has no trigger info")
	}
	plan := Plan{TriggerCountBefore: f.Triggers.Count, TriggerCountAfter: f.Triggers.Count}
	if f.Units != nil {
		plan.UnitCountBefore = f.Units.Total
		plan.UnitCountAfter = f.Units.Total
	}
	for _, trigger := range recipe.Triggers {
		enabled := false
		if trigger.Enabled != nil {
			enabled = *trigger.Enabled
		}
		switch trigger.Op {
		case "add_display_instructions", "add_create_object", "add_trigger", "copy_trigger", "edit_trigger", "tombstone_trigger", "remove_trigger", "remove_triggers", "remove_system_prefix", "clear_triggers":
			if trigger.Op == "edit_trigger" || trigger.Op == "tombstone_trigger" {
				if _, err := f.findTrigger(trigger); err != nil {
					return Plan{}, err
				}
			}
			if trigger.Op == "copy_trigger" {
				if _, err := f.findTrigger(trigger); err != nil {
					return Plan{}, err
				}
			}
			if trigger.Op == "remove_trigger" {
				index, err := f.findTriggerIndex(trigger)
				if err != nil {
					return Plan{}, err
				}
				deletePlan, err := f.DeletePlan(DeletePlanRequest{Kind: "trigger", ID: index})
				if err != nil {
					return Plan{}, err
				}
				if !deletePlan.CanDelete {
					return Plan{}, fmt.Errorf("remove_trigger target_index %d blocked by %d reference(s)", index, deletePlan.ReferenceSummary.Total)
				}
			}
			if trigger.Op == "remove_triggers" {
				count, err := f.validateRemoveTriggers(trigger.TargetIndexes)
				if err != nil {
					return Plan{}, err
				}
				plan.TriggerCountAfter -= count
			}
			if trigger.Op == "remove_system_prefix" {
				matches, externalRefs, err := f.validateRemoveSystemPrefix(trigger.TargetPrefix)
				if err != nil {
					return Plan{}, err
				}
				if externalRefs.Summary.Total > 0 {
					return Plan{}, fmt.Errorf("remove_system_prefix %q blocked by %d external reference(s)", trigger.TargetPrefix, externalRefs.Summary.Total)
				}
				plan.TriggerCountAfter -= len(matches.TriggerIndexes)
				plan.UnitCountAfter -= len(matches.UnitReferenceIDs)
			}
			plan.Operations = append(plan.Operations, PlanOp{
				Op:      trigger.Op,
				Name:    trigger.Name,
				Enabled: &enabled,
				Message: trigger.Message,
			})
			switch trigger.Op {
			case "add_display_instructions", "add_create_object", "add_trigger", "copy_trigger":
				plan.TriggerCountAfter++
			case "remove_trigger":
				plan.TriggerCountAfter--
			case "remove_triggers", "remove_system_prefix":
			case "clear_triggers":
				plan.TriggerCountAfter = 0
			}
		default:
			return Plan{}, fmt.Errorf("unsupported trigger op %q", trigger.Op)
		}
	}
	if recipe.XS != nil {
		plan.Operations = append(plan.Operations, PlanOp{Op: "set_xs", Name: recipe.XS.Name})
	}
	if recipe.Scenario != nil && recipe.Scenario.PlayerCount != nil {
		plan.Operations = append(plan.Operations, PlanOp{Op: "set_scenario_player_count", Name: fmt.Sprintf("%d", *recipe.Scenario.PlayerCount)})
	}
	if recipe.Victory != nil {
		plan.Operations = append(plan.Operations, PlanOp{Op: "set_global_victory", Name: recipe.Victory.summary()})
	}
	for _, player := range recipe.Players {
		plan.Operations = append(plan.Operations, PlanOp{Op: "set_player", Name: fmt.Sprintf("P%d", player.Player)})
	}
	for _, diplomacy := range recipe.Diplomacy {
		plan.Operations = append(plan.Operations, PlanOp{Op: "set_diplomacy", Name: fmt.Sprintf("P%d->P%d=%d", diplomacy.From, diplomacy.To, diplomacy.Stance)})
	}
	if recipe.DiplomacyOptions != nil {
		if err := f.validateDiplomacyOptions(*recipe.DiplomacyOptions); err != nil {
			return Plan{}, err
		}
		plan.Operations = append(plan.Operations, PlanOp{Op: "set_diplomacy_options", Name: recipe.DiplomacyOptions.summary()})
	}
	for _, resource := range recipe.Resources {
		plan.Operations = append(plan.Operations, PlanOp{Op: "set_resources", Name: fmt.Sprintf("P%d", resource.Player)})
	}
	for _, stringRecipe := range recipe.Strings {
		switch stringRecipe.Op {
		case "add_string":
			if _, err := f.planStringSlot(stringRecipe, true); err != nil {
				return Plan{}, err
			}
		case "set_string":
			if _, err := f.planStringSlot(stringRecipe, false); err != nil {
				return Plan{}, err
			}
			if stringRecipe.SetText == nil && stringRecipe.Text == "" {
				return Plan{}, fmt.Errorf("set_string requires set_text or text")
			}
		case "tombstone_string", "clear_string":
			id, err := f.planStringSlot(stringRecipe, false)
			if err != nil {
				return Plan{}, err
			}
			refs, err := f.References(ReferenceOptions{Kind: "string", TargetID: &id})
			if err != nil {
				return Plan{}, err
			}
			if refs.Summary.Total > 0 {
				return Plan{}, fmt.Errorf("refuses to %s string %d because it is referenced %d time(s)", strings.TrimSuffix(stringRecipe.Op, "_string"), id, refs.Summary.Total)
			}
		default:
			return Plan{}, fmt.Errorf("unsupported string op %q", stringRecipe.Op)
		}
		plan.Operations = append(plan.Operations, PlanOp{Op: stringRecipe.Op, Name: stringRecipe.summary()})
	}
	for _, variable := range recipe.Variables {
		switch variable.Op {
		case "add_variable":
			if variable.Name == "" {
				return Plan{}, fmt.Errorf("add_variable requires name")
			}
		case "edit_variable":
			if _, err := f.findVariable(variable); err != nil {
				return Plan{}, err
			}
			if variable.SetName == nil && variable.Name == "" {
				return Plan{}, fmt.Errorf("edit_variable requires set_name or name")
			}
		case "tombstone_variable", "remove_variable":
			target, err := f.findVariable(variable)
			if err != nil {
				return Plan{}, err
			}
			id, _ := target.intValue("variable_id")
			refs, err := f.References(ReferenceOptions{Kind: "variable", TargetID: &id})
			if err != nil {
				return Plan{}, err
			}
			if refs.Summary.Total > 0 {
				return Plan{}, fmt.Errorf("refuses to tombstone variable %d because it is referenced %d time(s)", id, refs.Summary.Total)
			}
		default:
			return Plan{}, fmt.Errorf("unsupported variable op %q", variable.Op)
		}
		plan.Operations = append(plan.Operations, PlanOp{Op: variable.Op, Name: variable.summary()})
	}
	for _, unit := range recipe.Units {
		switch unit.Op {
		case "add_unit", "cluster_scatter", "edit_unit", "remove_unit", "remove_units_in_area", "remove_units_for_player", "copy_units_in_area", "move_units_in_area", "edit_units_in_area":
			matches := 0
			if unit.Op == "edit_unit" {
				if _, _, _, err := f.findUnit(unit); err != nil {
					return Plan{}, err
				}
			}
			if unit.Op == "cluster_scatter" {
				count, err := f.validateClusterScatter(unit)
				if err != nil {
					return Plan{}, err
				}
				matches = count
			}
			if unit.Op == "remove_unit" {
				target, _, _, err := f.findUnit(unit)
				if err != nil {
					return Plan{}, err
				}
				if referenceID, ok := target.intValue("reference_id"); ok && referenceID >= 0 {
					if err := f.ensureUnitNotReferenced(referenceID); err != nil {
						return Plan{}, err
					}
				}
			}
			if unit.Op == "remove_units_in_area" {
				targets, err := f.findUnitsInArea(unit)
				if err != nil {
					return Plan{}, err
				}
				for _, target := range targets {
					if referenceID, ok := target.Unit.intValue("reference_id"); ok && referenceID >= 0 {
						if err := f.ensureUnitNotReferenced(referenceID); err != nil {
							return Plan{}, err
						}
					}
				}
				matches = len(targets)
			}
			if unit.Op == "remove_units_for_player" {
				targets, err := f.findUnitsByPlayer(unit.TargetPlayer)
				if err != nil {
					return Plan{}, err
				}
				for _, target := range targets {
					if referenceID, ok := target.Unit.intValue("reference_id"); ok && referenceID >= 0 {
						if err := f.ensureUnitNotReferenced(referenceID); err != nil {
							return Plan{}, err
						}
					}
				}
				matches = len(targets)
			}
			if unit.Op == "copy_units_in_area" {
				targets, err := f.findUnitsInArea(unit)
				if err != nil {
					return Plan{}, err
				}
				if err := f.validateCopiedUnitReferenceIDs(targets, unit); err != nil {
					return Plan{}, err
				}
				matches = len(targets)
			}
			if unit.Op == "move_units_in_area" {
				targets, err := f.findUnitsInArea(unit)
				if err != nil {
					return Plan{}, err
				}
				if err := f.validateMoveUnitsInArea(targets, unit); err != nil {
					return Plan{}, err
				}
				matches = len(targets)
			}
			if unit.Op == "edit_units_in_area" {
				targets, err := f.findUnitsInArea(unit)
				if err != nil {
					return Plan{}, err
				}
				if err := f.validateEditUnitsInArea(unit); err != nil {
					return Plan{}, err
				}
				matches = len(targets)
			}
			plan.Operations = append(plan.Operations, PlanOp{Op: unit.Op})
			switch unit.Op {
			case "add_unit":
				plan.UnitCountAfter++
			case "cluster_scatter":
				plan.UnitCountAfter += matches
			case "remove_unit":
				plan.UnitCountAfter--
			case "remove_units_in_area":
				plan.UnitCountAfter -= matches
			case "remove_units_for_player":
				plan.UnitCountAfter -= matches
			case "copy_units_in_area":
				plan.UnitCountAfter += matches
			case "move_units_in_area":
				_ = matches
			case "edit_units_in_area":
				_ = matches
			}
		default:
			return Plan{}, fmt.Errorf("unsupported unit op %q", unit.Op)
		}
	}
	masks, err := f.resolveMaskRecipes(recipe.Masks)
	if err != nil {
		return Plan{}, err
	}
	for _, mapPatch := range recipe.Map {
		switch mapPatch.Op {
		case "set_terrain_rect", "set_terrain_circle", "set_terrain_line", "set_terrain_border", "copy_terrain_area", "noise_fill", "erode", "semantic_erode", "set_terrain_mask", "layered_crossfade", "terrain_grid":
			changed, err := f.countMapTiles(mapPatch, masks)
			if err != nil {
				return Plan{}, err
			}
			plan.Operations = append(plan.Operations, PlanOp{Op: mapPatch.Op})
			plan.MapTilesChanged += changed
		default:
			return Plan{}, fmt.Errorf("unsupported map op %q", mapPatch.Op)
		}
	}
	return plan, nil
}

func (f *File) ApplyRecipe(recipe Recipe) error {
	if recipe.Scenario != nil {
		if err := f.SetScenario(*recipe.Scenario); err != nil {
			return err
		}
	}
	if recipe.Victory != nil {
		if err := f.SetGlobalVictory(*recipe.Victory); err != nil {
			return err
		}
	}
	for _, player := range recipe.Players {
		if err := f.SetPlayer(player); err != nil {
			return err
		}
	}
	for _, diplomacy := range recipe.Diplomacy {
		if err := f.SetDiplomacy(diplomacy); err != nil {
			return err
		}
	}
	if recipe.DiplomacyOptions != nil {
		if err := f.SetDiplomacyOptions(*recipe.DiplomacyOptions); err != nil {
			return err
		}
	}
	for _, resource := range recipe.Resources {
		if err := f.SetResources(resource); err != nil {
			return err
		}
	}
	for _, stringRecipe := range recipe.Strings {
		switch stringRecipe.Op {
		case "add_string":
			if _, err := f.AddString(stringRecipe); err != nil {
				return err
			}
		case "set_string":
			if err := f.SetString(stringRecipe); err != nil {
				return err
			}
		case "tombstone_string":
			if err := f.TombstoneString(stringRecipe); err != nil {
				return err
			}
		case "clear_string":
			if err := f.ClearString(stringRecipe); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported string op %q", stringRecipe.Op)
		}
	}
	for _, variable := range recipe.Variables {
		switch variable.Op {
		case "add_variable":
			if err := f.AddVariable(variable); err != nil {
				return err
			}
		case "edit_variable":
			if err := f.EditVariable(variable); err != nil {
				return err
			}
		case "tombstone_variable":
			if err := f.TombstoneVariable(variable); err != nil {
				return err
			}
		case "remove_variable":
			if err := f.RemoveVariable(variable); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported variable op %q", variable.Op)
		}
	}
	for _, trigger := range recipe.Triggers {
		switch trigger.Op {
		case "add_display_instructions":
			if err := f.AddDisplayInstructionsTrigger(trigger); err != nil {
				return err
			}
		case "add_create_object":
			if err := f.AddCreateObjectTrigger(trigger); err != nil {
				return err
			}
		case "add_trigger":
			if err := f.AddTrigger(trigger); err != nil {
				return err
			}
		case "copy_trigger":
			if err := f.CopyTrigger(trigger); err != nil {
				return err
			}
		case "edit_trigger":
			if err := f.EditTrigger(trigger); err != nil {
				return err
			}
		case "tombstone_trigger":
			if err := f.TombstoneTrigger(trigger); err != nil {
				return err
			}
		case "remove_trigger":
			if err := f.RemoveTrigger(trigger); err != nil {
				return err
			}
		case "remove_triggers":
			if err := f.RemoveTriggers(trigger); err != nil {
				return err
			}
		case "remove_system_prefix":
			if err := f.RemoveSystemPrefix(trigger.TargetPrefix); err != nil {
				return err
			}
		case "clear_triggers":
			if err := f.ClearTriggers(); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported trigger op %q", trigger.Op)
		}
	}
	if recipe.XS != nil {
		if err := f.SetXS(*recipe.XS); err != nil {
			return err
		}
	}
	var pendingAddUnits []UnitRecipe
	footprints, err := newClusterFootprintResolver(recipe.DatPath)
	if err != nil {
		return fmt.Errorf("cluster_scatter dat_path %q: %w", recipe.DatPath, err)
	}
	defer footprints.Close()
	flushAddUnits := func() error {
		if len(pendingAddUnits) == 0 {
			return nil
		}
		resolved := make([]UnitRecipe, 0, len(pendingAddUnits))
		for _, unit := range pendingAddUnits {
			placed, err := resolveUnitPlacement(unit, footprints)
			if err != nil {
				return err
			}
			resolved = append(resolved, placed)
		}
		if err := f.AddUnits(resolved); err != nil {
			return err
		}
		pendingAddUnits = nil
		return nil
	}
	for _, unit := range recipe.Units {
		switch unit.Op {
		case "add_unit":
			pendingAddUnits = append(pendingAddUnits, unit)
		case "cluster_scatter":
			if err := flushAddUnits(); err != nil {
				return err
			}
			adds, err := f.clusterScatterUnits(unit, footprints)
			if err != nil {
				return err
			}
			if err := f.AddUnits(adds); err != nil {
				return err
			}
		case "edit_unit":
			if err := flushAddUnits(); err != nil {
				return err
			}
			if err := f.EditUnit(unit); err != nil {
				return err
			}
		case "remove_unit":
			if err := flushAddUnits(); err != nil {
				return err
			}
			if err := f.RemoveUnit(unit); err != nil {
				return err
			}
		case "remove_units_in_area":
			if err := flushAddUnits(); err != nil {
				return err
			}
			if _, err := f.RemoveUnitsInArea(unit); err != nil {
				return err
			}
		case "remove_units_for_player":
			if err := flushAddUnits(); err != nil {
				return err
			}
			if _, err := f.RemoveUnitsForPlayer(unit); err != nil {
				return err
			}
		case "copy_units_in_area":
			if err := flushAddUnits(); err != nil {
				return err
			}
			if _, err := f.CopyUnitsInArea(unit); err != nil {
				return err
			}
		case "move_units_in_area":
			if err := flushAddUnits(); err != nil {
				return err
			}
			if _, err := f.MoveUnitsInArea(unit); err != nil {
				return err
			}
		case "edit_units_in_area":
			if err := flushAddUnits(); err != nil {
				return err
			}
			if _, err := f.EditUnitsInArea(unit); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported unit op %q", unit.Op)
		}
	}
	if err := flushAddUnits(); err != nil {
		return err
	}
	masks, err := f.resolveMaskRecipes(recipe.Masks)
	if err != nil {
		return err
	}
	for _, mapPatch := range recipe.Map {
		switch mapPatch.Op {
		case "set_terrain_rect", "set_terrain_circle", "set_terrain_line", "set_terrain_border":
			if err := f.SetTerrainWithMasks(mapPatch, masks); err != nil {
				return err
			}
		case "set_terrain_mask":
			if err := f.SetTerrainWithMasks(mapPatch, masks); err != nil {
				return err
			}
		case "layered_crossfade":
			if err := f.LayeredCrossfadeTerrainWithMasks(mapPatch, masks); err != nil {
				return err
			}
		case "noise_fill":
			if err := f.NoiseFillTerrainWithMasks(mapPatch, masks); err != nil {
				return err
			}
		case "erode":
			if err := f.ErodeTerrainWithMasks(mapPatch, masks, false); err != nil {
				return err
			}
		case "semantic_erode":
			if err := f.ErodeTerrainWithMasks(mapPatch, masks, true); err != nil {
				return err
			}
		case "copy_terrain_area":
			if err := f.CopyTerrainArea(mapPatch); err != nil {
				return err
			}
		case "terrain_grid":
			if err := f.ApplyTerrainGrid(mapPatch); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported map op %q", mapPatch.Op)
		}
	}
	return nil
}

func (f *File) SetScenario(recipe ScenarioRecipe) error {
	changed := false
	if recipe.StartingAge != nil {
		if *recipe.StartingAge < 0 || *recipe.StartingAge > 6 {
			return fmt.Errorf("scenario starting_age %d out of range 0..6", *recipe.StartingAge)
		}
		options := f.root.section("Options")
		if options == nil {
			return fmt.Errorf("missing Options section")
		}
		ages := options.intList("per_player_starting_age")
		if len(ages) == 0 {
			return fmt.Errorf("missing per_player_starting_age field")
		}
		for i := range ages {
			if i < f.PlayerCount {
				ages[i] = *recipe.StartingAge
			}
		}
		if err := setIntListField(options, "per_player_starting_age", "u32", ages); err != nil {
			return err
		}
		changed = true
	}
	if recipe.TimestampOfLastSave != nil {
		if f.headerRoot == nil {
			return fmt.Errorf("missing FileHeader section")
		}
		timestamp := *recipe.TimestampOfLastSave
		if timestamp < 0 {
			return fmt.Errorf("scenario timestamp_of_last_save %d must be non-negative", timestamp)
		}
		if err := setIntField(f.headerRoot, "timestamp_of_last_save", "u32", timestamp); err != nil {
			return err
		}
		f.header = f.headerRoot.raw()
		f.HeaderBytes = len(f.header)
		changed = true
	}
	if recipe.PlayerCount != nil {
		if f.headerRoot == nil {
			return fmt.Errorf("missing FileHeader section")
		}
		playerCount := *recipe.PlayerCount
		if playerCount < 1 || playerCount > 8 {
			return fmt.Errorf("scenario player_count %d out of range 1..8", playerCount)
		}
		if err := setIntField(f.headerRoot, "player_count", "u32", playerCount); err != nil {
			return err
		}
		if units := f.root.section("Units"); units != nil {
			if err := setIntField(units, "number_of_players", "u32", len(units.list("players_units"))); err != nil {
				return err
			}
		}
		f.header = f.headerRoot.raw()
		f.HeaderBytes = len(f.header)
		f.PlayerCount = playerCount
		changed = true
	}
	if !changed {
		return fmt.Errorf("scenario recipe has no fields to set")
	}
	return nil
}

func (r VictoryRecipe) summary() string {
	var parts []string
	fields := []struct {
		name  string
		value *int
	}{
		{"conquest_required", r.ConquestRequired},
		{"ruins", r.Ruins},
		{"artifacts_required", r.ArtifactsRequired},
		{"discovery", r.Discovery},
		{"explored_percent_of_map_required", r.ExploredPercentOfMapRequired},
		{"gold_required", r.GoldRequired},
		{"all_custom_conditions_required", r.AllCustomConditionsRequired},
		{"mode", r.Mode},
		{"required_score_for_score_victory", r.RequiredScoreForScoreVictory},
		{"time_for_timed_game_in_10ths_of_a_year", r.TimeForTimedGameIn10thsOfAYear},
	}
	for _, field := range fields {
		if field.value != nil {
			parts = append(parts, fmt.Sprintf("%s=%d", field.name, *field.value))
		}
	}
	if len(parts) == 0 {
		return "no fields"
	}
	return strings.Join(parts, ",")
}

func (f *File) SetGlobalVictory(recipe VictoryRecipe) error {
	victory := f.root.section("GlobalVictory")
	if victory == nil {
		return fmt.Errorf("missing GlobalVictory section")
	}
	fields := []struct {
		name  string
		value *int
	}{
		{"conquest_required", recipe.ConquestRequired},
		{"ruins", recipe.Ruins},
		{"artifacts_required", recipe.ArtifactsRequired},
		{"discovery", recipe.Discovery},
		{"explored_percent_of_map_required", recipe.ExploredPercentOfMapRequired},
		{"gold_required", recipe.GoldRequired},
		{"all_custom_conditions_required", recipe.AllCustomConditionsRequired},
		{"mode", recipe.Mode},
		{"required_score_for_score_victory", recipe.RequiredScoreForScoreVictory},
		{"time_for_timed_game_in_10ths_of_a_year", recipe.TimeForTimedGameIn10thsOfAYear},
	}
	changed := false
	for _, field := range fields {
		if field.value != nil {
			if err := setIntField(victory, field.name, "u32", *field.value); err != nil {
				return err
			}
			changed = true
		}
	}
	if !changed {
		return fmt.Errorf("victory recipe has no fields to set")
	}
	f.body = f.root.raw()
	f.InflatedBytes = len(f.body)
	return nil
}

func (f *File) SetXS(recipe XSRecipe) error {
	content, err := xsRecipeContent(recipe)
	if err != nil {
		return err
	}
	mode := normalizeXSMode(recipe.Mode)
	switch mode {
	case "attachment":
		if recipe.Name == "" {
			return fmt.Errorf("xs attachment mode requires name")
		}
		return f.setXSAttachment(recipe.Name, content)
	case "carrier":
		if content == "" {
			return fmt.Errorf("xs carrier mode requires content or content_file")
		}
		clearAttachment := true
		if recipe.ClearAttachment != nil {
			clearAttachment = *recipe.ClearAttachment
		}
		if clearAttachment {
			if err := f.clearXSAttachment(); err != nil {
				return err
			}
		}
		return f.SetXSCarrier(recipe, content)
	case "inline_runtime":
		if content == "" {
			return fmt.Errorf("xs inline_runtime mode requires content or content_file")
		}
		return f.SetXSInlineRuntime(recipe, content)
	case "attachment_and_carrier":
		if recipe.Name == "" {
			return fmt.Errorf("xs attachment_and_carrier mode requires name")
		}
		if content == "" {
			return fmt.Errorf("xs attachment_and_carrier mode requires content or content_file")
		}
		if err := f.setXSAttachment(recipe.Name, content); err != nil {
			return err
		}
		return f.SetXSCarrier(recipe, content)
	default:
		return fmt.Errorf("unsupported xs mode %q", recipe.Mode)
	}
}

func xsRecipeContent(recipe XSRecipe) (string, error) {
	if recipe.Content != "" && recipe.ContentFile != "" {
		return "", fmt.Errorf("xs supports content or content_file, not both")
	}
	if recipe.ContentFile == "" {
		return recipe.Content, nil
	}
	data, err := os.ReadFile(recipe.ContentFile)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func normalizeXSMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "", "attachment", "file", "files":
		return "attachment"
	case "carrier", "embedded_carrier", "parser_carrier", "parser-style", "parser_style":
		return "carrier"
	case "inline", "inline_runtime", "runtime_inline", "runtime_carrier", "spiral", "spiral_style":
		return "inline_runtime"
	case "both", "attachment_and_carrier", "file_and_carrier", "files_and_carrier":
		return "attachment_and_carrier"
	default:
		return strings.ToLower(strings.TrimSpace(mode))
	}
}

func (f *File) setXSAttachment(name, content string) error {
	name = normalizeXSFileName(name)
	mapSection := f.root.section("Map")
	if mapSection == nil {
		return fmt.Errorf("missing Map section")
	}
	if err := setStringField(mapSection, "script_name", "str16", name); err != nil {
		return err
	}
	files := f.root.section("Files")
	if files == nil {
		return fmt.Errorf("missing Files section")
	}
	if err := setStringField(files, "script_file_path", "str16", name); err != nil {
		return err
	}
	if err := setStringField(files, "script_file_content", "str32", content); err != nil {
		return err
	}
	f.body = f.root.raw()
	f.InflatedBytes = len(f.body)
	return nil
}

func (f *File) clearXSAttachment() error {
	mapSection := f.root.section("Map")
	if mapSection == nil {
		return fmt.Errorf("missing Map section")
	}
	if err := setZeroLengthStringField(mapSection, "script_name", "str16"); err != nil {
		return err
	}
	files := f.root.section("Files")
	if files == nil {
		return fmt.Errorf("missing Files section")
	}
	if err := setZeroLengthStringField(files, "script_file_path", "str16"); err != nil {
		return err
	}
	if err := setZeroLengthStringField(files, "script_file_content", "str32"); err != nil {
		return err
	}
	f.body = f.root.raw()
	f.InflatedBytes = len(f.body)
	return nil
}

func (f *File) SetXSCarrier(recipe XSRecipe, content string) error {
	return f.setXSCarrier(recipe, content, false, false)
}

func (f *File) SetXSInlineRuntime(recipe XSRecipe, content string) error {
	if err := f.clearXSAttachment(); err != nil {
		return err
	}
	if strings.TrimSpace(recipe.CarrierTitle) == "" {
		recipe.CarrierTitle = "XS string"
	}
	if strings.TrimSpace(recipe.CarrierTriggerName) == "" {
		recipe.CarrierTriggerName = "XS SCRIPT"
	}
	if recipe.CarrierTriggerIndex == nil {
		index := 0
		recipe.CarrierTriggerIndex = &index
	}
	return f.setXSCarrier(recipe, content, true, true)
}

func (f *File) setXSCarrier(recipe XSRecipe, content string, enabled bool, preferInsert bool) error {
	replace := true
	if recipe.ReplaceCarrier != nil {
		replace = *recipe.ReplaceCarrier
	}
	triggerName := strings.TrimSpace(recipe.CarrierTriggerName)
	if triggerName == "" {
		triggerName = "XS SCRIPT"
	}
	title := strings.TrimSpace(recipe.CarrierTitle)
	if title == "" {
		if recipe.ContentFile != "" {
			title = filepath.Base(recipe.ContentFile)
		} else if recipe.Name != "" {
			title = normalizeXSFileName(recipe.Name)
		} else {
			title = "XS string"
		}
	}
	message := xsCarrierMessage(title, content)
	triggerEnabled := enabled
	notLooping := false
	sourcePlayer := 1
	effect := EffectRecipe{Op: "script_call", SourcePlayer: &sourcePlayer, Message: message}
	if recipe.CarrierTriggerIndex != nil {
		index := *recipe.CarrierTriggerIndex
		if preferInsert {
			triggers := f.root.section("Triggers")
			if triggers == nil {
				return fmt.Errorf("missing Triggers section")
			}
			triggerNodes := triggers.list("trigger_data")
			if index < 0 || index > len(triggerNodes) {
				return fmt.Errorf("carrier_trigger_index %d out of insert range 0..%d", index, len(triggerNodes))
			}
			if index == len(triggerNodes) || !triggerHasXSCarrier(triggerNodes[index]) {
				insert := TriggerRecipe{
					Op:      "add_trigger",
					Name:    triggerName,
					Enabled: &triggerEnabled,
					Looping: &notLooping,
					Effects: []EffectRecipe{effect},
				}
				raw, err := f.buildTriggerRaw(insert)
				if err != nil {
					return err
				}
				return f.insertRawTrigger(index, raw)
			}
		}
		clearConditions := true
		edit := TriggerRecipe{
			Op:              "edit_trigger",
			TargetIndex:     recipe.CarrierTriggerIndex,
			SetName:         &triggerName,
			Enabled:         &triggerEnabled,
			Looping:         &notLooping,
			ReplaceEffects:  []EffectRecipe{effect},
			ClearConditions: &clearConditions,
		}
		return f.EditTrigger(edit)
	}
	if replace {
		carriers := f.xsEmbeddedCarriers()
		if len(carriers) > 0 {
			clearConditions := true
			edit := TriggerRecipe{
				Op:                "edit_trigger",
				TargetIndex:       &carriers[0].TriggerIndex,
				SetName:           &triggerName,
				Enabled:           &triggerEnabled,
				Looping:           &notLooping,
				ReplaceEffects:    []EffectRecipe{effect},
				ClearConditions:   &clearConditions,
				ClearEffects:      nil,
				ReplaceConditions: nil,
			}
			return f.EditTrigger(edit)
		}
	}
	return f.AddTrigger(TriggerRecipe{
		Op:      "add_trigger",
		Name:    triggerName,
		Enabled: &triggerEnabled,
		Looping: &notLooping,
		Effects: []EffectRecipe{effect},
	})
}

func xsCarrierMessage(title, content string) string {
	return fmt.Sprintf("// ------------------------- %s -------------------------\n%s\n\n", title, strings.TrimRight(content, "\r\n"))
}

func normalizeXSFileName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" || strings.HasSuffix(strings.ToLower(name), ".xs") {
		return name
	}
	return name + ".xs"
}

func (f *File) SetPlayer(recipe PlayerRecipe) error {
	dataHeader := f.root.section("DataHeader")
	if dataHeader == nil {
		return fmt.Errorf("missing DataHeader section")
	}
	playerData := dataHeader.list("player_data_1")
	if recipe.Player < 0 || recipe.Player >= len(playerData) {
		return fmt.Errorf("player %d out of range 0..%d", recipe.Player, len(playerData)-1)
	}
	node := playerData[recipe.Player]
	if recipe.Active != nil {
		if err := setIntField(node, "active", "u32", boolInt(*recipe.Active)); err != nil {
			return err
		}
	}
	if recipe.Human != nil {
		if err := setIntField(node, "human", "u32", boolInt(*recipe.Human)); err != nil {
			return err
		}
	}
	if recipe.Civilization != nil {
		if err := setStringField(node, "civilization", "str16", *recipe.Civilization); err != nil {
			return err
		}
	}
	if recipe.TribeName != nil {
		tribeNames := dataHeader.stringList("tribe_names")
		if recipe.Player >= len(tribeNames) {
			return fmt.Errorf("tribe_names player %d out of range 0..%d", recipe.Player, len(tribeNames)-1)
		}
		tribeNames[recipe.Player] = *recipe.TribeName
		if err := setStringListField(dataHeader, "tribe_names", "c256", tribeNames); err != nil {
			return err
		}
	}
	if recipe.LockCivilization != nil {
		locks := dataHeader.intList("per_player_lock_civilization")
		if recipe.Player >= len(locks) {
			return fmt.Errorf("per_player_lock_civilization player %d out of range 0..%d", recipe.Player, len(locks)-1)
		}
		locks[recipe.Player] = boolInt(*recipe.LockCivilization)
		if err := setIntListField(dataHeader, "per_player_lock_civilization", "u32", locks); err != nil {
			return err
		}
	}
	if recipe.LockPersonality != nil {
		locks := dataHeader.intList("per_player_lock_personality")
		if recipe.Player >= len(locks) {
			return fmt.Errorf("per_player_lock_personality player %d out of range 0..%d", recipe.Player, len(locks)-1)
		}
		locks[recipe.Player] = boolInt(*recipe.LockPersonality)
		if err := setIntListField(dataHeader, "per_player_lock_personality", "u32", locks); err != nil {
			return err
		}
	}
	if recipe.AIName != nil || recipe.AIType != nil {
		playerDataTwo := f.root.section("PlayerDataTwo")
		if playerDataTwo == nil {
			return fmt.Errorf("missing PlayerDataTwo section")
		}
		if recipe.AIName != nil {
			names := playerDataTwo.stringList("ai_names")
			if recipe.Player >= len(names) {
				return fmt.Errorf("ai_names player %d out of range 0..%d", recipe.Player, len(names)-1)
			}
			names[recipe.Player] = *recipe.AIName
			if err := setStringListField(playerDataTwo, "ai_names", "str16", names); err != nil {
				return err
			}
		}
		if recipe.AIType != nil {
			types := playerDataTwo.intList("ai_type")
			if recipe.Player >= len(types) {
				return fmt.Errorf("ai_type player %d out of range 0..%d", recipe.Player, len(types)-1)
			}
			types[recipe.Player] = *recipe.AIType
			if err := setIntListField(playerDataTwo, "ai_type", "u8", types); err != nil {
				return err
			}
		}
	}
	f.body = f.root.raw()
	f.InflatedBytes = len(f.body)
	f.Players = f.root.playerInfo()
	return nil
}

func (f *File) SetDiplomacy(recipe DiplomacyRecipe) error {
	diplomacy := f.root.section("Diplomacy")
	if diplomacy == nil {
		return fmt.Errorf("missing Diplomacy section")
	}
	rows := diplomacy.list("per_player_diplomacy")
	if recipe.From < 0 || recipe.From >= len(rows) {
		return fmt.Errorf("diplomacy from-player %d out of range 0..%d", recipe.From, len(rows)-1)
	}
	row := rows[recipe.From]
	stances := row.uint32List("stance_with_each_player")
	if recipe.To < 0 || recipe.To >= len(stances) {
		return fmt.Errorf("diplomacy to-player %d out of range 0..%d", recipe.To, len(stances)-1)
	}
	stances[recipe.To] = uint32(recipe.Stance)
	if err := setUint32ListField(row, "stance_with_each_player", stances); err != nil {
		return err
	}
	if err := f.setDiplomacyMirrors(recipe.From, recipe.To, recipe.Stance); err != nil {
		return err
	}
	f.body = f.root.raw()
	f.InflatedBytes = len(f.body)
	return nil
}

func diplomacyMirrorValues(stance int) (interaction, aiSystem int, err error) {
	switch stance {
	case 0: // ally
		return 0, 2, nil
	case 1: // neutral
		return 1, 3, nil
	case 3: // enemy
		return 3, 4, nil
	default:
		return 0, 0, fmt.Errorf("unsupported diplomacy stance %d; expected ally=0, neutral=1, or enemy=3", stance)
	}
}

func (f *File) setDiplomacyMirrors(from, to, stance int) error {
	interaction, aiSystem, err := diplomacyMirrorValues(stance)
	if err != nil {
		return err
	}
	units := f.root.section("Units")
	if units == nil {
		return fmt.Errorf("missing Units section for diplomacy mirrors")
	}
	rows := units.list("player_data_3")
	// Both sections use zero-based source-player rows. The mirror target list
	// reserves slot 0 for Gaia, so its target is one past the main-table slot.
	mirrorFrom := from
	if mirrorFrom < 0 || mirrorFrom >= len(rows) {
		return fmt.Errorf("diplomacy mirror from-player %d out of range", from)
	}
	row := rows[mirrorFrom]
	interactions := row.intList("diplomacy_for_interaction")
	aiValues := row.intList("diplomacy_for_ai_system")
	mirrorTarget := to + 1
	if mirrorTarget < 0 || mirrorTarget >= len(interactions) || mirrorTarget >= len(aiValues) {
		return fmt.Errorf("diplomacy mirror to-player %d out of range", to)
	}
	interactions[mirrorTarget] = interaction
	aiValues[mirrorTarget] = aiSystem
	if err := setIntListField(row, "diplomacy_for_interaction", "u8", interactions); err != nil {
		return err
	}
	return setIntListField(row, "diplomacy_for_ai_system", "u32", aiValues)
}

func (r DiplomacyOptionsRecipe) summary() string {
	var parts []string
	if r.LockTeams != nil {
		parts = append(parts, fmt.Sprintf("lock_teams=%t", *r.LockTeams))
	}
	if r.AllowPlayersChooseTeams != nil {
		parts = append(parts, fmt.Sprintf("allow_players_choose_teams=%t", *r.AllowPlayersChooseTeams))
	}
	if r.RandomStartPoints != nil {
		parts = append(parts, fmt.Sprintf("random_start_points=%t", *r.RandomStartPoints))
	}
	if r.MaxNumberOfTeams != nil {
		parts = append(parts, fmt.Sprintf("max_number_of_teams=%d", *r.MaxNumberOfTeams))
	}
	for _, allied := range r.AlliedVictory {
		parts = append(parts, fmt.Sprintf("P%d allied_victory=%t", allied.Player, allied.Enabled))
	}
	if len(parts) == 0 {
		return "no fields"
	}
	return strings.Join(parts, ",")
}

func (f *File) validateDiplomacyOptions(recipe DiplomacyOptionsRecipe) error {
	if recipe.LockTeams == nil && recipe.AllowPlayersChooseTeams == nil && recipe.RandomStartPoints == nil &&
		recipe.MaxNumberOfTeams == nil && len(recipe.AlliedVictory) == 0 {
		return fmt.Errorf("diplomacy_options has no fields to set")
	}
	if recipe.MaxNumberOfTeams != nil && (*recipe.MaxNumberOfTeams < 0 || *recipe.MaxNumberOfTeams > 16) {
		return fmt.Errorf("max_number_of_teams %d out of range 0..16", *recipe.MaxNumberOfTeams)
	}
	for _, allied := range recipe.AlliedVictory {
		if allied.Player < 0 || allied.Player > 15 {
			return fmt.Errorf("allied_victory player %d out of range 0..15", allied.Player)
		}
	}
	return nil
}

func (f *File) SetDiplomacyOptions(recipe DiplomacyOptionsRecipe) error {
	if err := f.validateDiplomacyOptions(recipe); err != nil {
		return err
	}
	diplomacy := f.root.section("Diplomacy")
	if diplomacy == nil {
		return fmt.Errorf("missing Diplomacy section")
	}
	if recipe.LockTeams != nil {
		if err := setBoolByteField(diplomacy, "lock_teams", *recipe.LockTeams); err != nil {
			return err
		}
	}
	if recipe.AllowPlayersChooseTeams != nil {
		if err := setBoolByteField(diplomacy, "allow_players_choose_teams", *recipe.AllowPlayersChooseTeams); err != nil {
			return err
		}
	}
	if recipe.RandomStartPoints != nil {
		if err := setBoolByteField(diplomacy, "random_start_points", *recipe.RandomStartPoints); err != nil {
			return err
		}
	}
	if recipe.MaxNumberOfTeams != nil {
		if err := setIntField(diplomacy, "max_number_of_teams", "u8", *recipe.MaxNumberOfTeams); err != nil {
			return err
		}
	}
	if len(recipe.AlliedVictory) > 0 {
		values := diplomacy.uint32List("per_player_allied_victory")
		if len(values) == 0 {
			return fmt.Errorf("missing per_player_allied_victory")
		}
		for _, allied := range recipe.AlliedVictory {
			if allied.Player < 0 || allied.Player >= len(values) {
				return fmt.Errorf("allied_victory player %d out of range 0..%d", allied.Player, len(values)-1)
			}
			if allied.Enabled {
				values[allied.Player] = 1
			} else {
				values[allied.Player] = 0
			}
		}
		if err := setUint32ListField(diplomacy, "per_player_allied_victory", values); err != nil {
			return err
		}
		units := f.root.section("Units")
		if units == nil {
			return fmt.Errorf("missing Units section for allied-victory mirrors")
		}
		rows := units.list("player_data_3")
		for _, allied := range recipe.AlliedVictory {
			mirrorPlayer := allied.Player
			if mirrorPlayer < 0 || mirrorPlayer >= len(rows) {
				return fmt.Errorf("allied_victory mirror player %d out of range", allied.Player)
			}
			if err := setIntField(rows[mirrorPlayer], "aok_allied_victory", "u8", boolInt(allied.Enabled)); err != nil {
				return err
			}
		}
	}
	f.body = f.root.raw()
	f.InflatedBytes = len(f.body)
	return nil
}

func (f *File) SetResources(recipe ResourceRecipe) error {
	playerDataTwo := f.root.section("PlayerDataTwo")
	if playerDataTwo == nil {
		return fmt.Errorf("missing PlayerDataTwo section")
	}
	resources := playerDataTwo.list("resources")
	if recipe.Player < 0 || recipe.Player >= len(resources) {
		return fmt.Errorf("resource player %d out of range 0..%d", recipe.Player, len(resources)-1)
	}
	node := resources[recipe.Player]
	fields := []struct {
		name  string
		value *int
	}{
		{"gold", recipe.Gold},
		{"wood", recipe.Wood},
		{"food", recipe.Food},
		{"stone", recipe.Stone},
		{"trade_goods", recipe.TradeGoods},
	}
	for _, field := range fields {
		if field.value != nil {
			if err := setIntField(node, field.name, "s32", *field.value); err != nil {
				return err
			}
		}
	}
	f.body = f.root.raw()
	f.InflatedBytes = len(f.body)
	return nil
}

func (r StringRecipe) summary() string {
	switch r.Op {
	case "add_string":
		if r.ID != nil {
			return fmt.Sprintf("%d:%s", *r.ID, r.Text)
		}
		return r.Text
	case "set_string", "tombstone_string", "clear_string":
		if r.ID != nil {
			if r.SetText != nil {
				return fmt.Sprintf("%d->%s", *r.ID, *r.SetText)
			}
			if r.Text != "" {
				return fmt.Sprintf("%d->%s", *r.ID, r.Text)
			}
			return fmt.Sprintf("%d", *r.ID)
		}
		if r.OldText != "" {
			return r.OldText
		}
	}
	return r.Op
}

func (f *File) AddString(recipe StringRecipe) (int, error) {
	text := recipe.Text
	if recipe.SetText != nil {
		text = *recipe.SetText
	}
	if text == "" {
		return 0, fmt.Errorf("add_string requires text or set_text")
	}
	id, err := f.planStringSlot(recipe, true)
	if err != nil {
		return 0, err
	}
	values, err := f.scenarioStringValues()
	if err != nil {
		return 0, err
	}
	if values[id] != "" {
		return 0, fmt.Errorf("string id %d is already occupied", id)
	}
	values[id] = text
	if err := f.setScenarioStringValues(values); err != nil {
		return 0, err
	}
	return id, nil
}

func (f *File) SetString(recipe StringRecipe) error {
	id, err := f.planStringSlot(recipe, false)
	if err != nil {
		return err
	}
	text := recipe.Text
	if recipe.SetText != nil {
		text = *recipe.SetText
	}
	if text == "" {
		return fmt.Errorf("set_string requires text or set_text")
	}
	values, err := f.scenarioStringValues()
	if err != nil {
		return err
	}
	if recipe.OldText != "" && values[id] != recipe.OldText {
		return fmt.Errorf("string id %d text %q does not match required old_text %q", id, values[id], recipe.OldText)
	}
	values[id] = text
	return f.setScenarioStringValues(values)
}

func (f *File) TombstoneString(recipe StringRecipe) error {
	id, err := f.planStringSlot(recipe, false)
	if err != nil {
		return err
	}
	refs, err := f.References(ReferenceOptions{Kind: "string", TargetID: &id})
	if err != nil {
		return err
	}
	if refs.Summary.Total > 0 {
		return fmt.Errorf("refuses to tombstone string %d because it is referenced %d time(s)", id, refs.Summary.Total)
	}
	values, err := f.scenarioStringValues()
	if err != nil {
		return err
	}
	if recipe.OldText != "" && values[id] != recipe.OldText {
		return fmt.Errorf("string id %d text %q does not match required old_text %q", id, values[id], recipe.OldText)
	}
	tombstone := fmt.Sprintf("_DeletedString%d", id)
	if recipe.SetText != nil {
		tombstone = *recipe.SetText
	}
	values[id] = tombstone
	return f.setScenarioStringValues(values)
}

func (f *File) ClearString(recipe StringRecipe) error {
	id, err := f.planStringSlot(recipe, false)
	if err != nil {
		return err
	}
	refs, err := f.References(ReferenceOptions{Kind: "string", TargetID: &id})
	if err != nil {
		return err
	}
	if refs.Summary.Total > 0 {
		return fmt.Errorf("refuses to clear string %d because it is referenced %d time(s)", id, refs.Summary.Total)
	}
	values, err := f.scenarioStringValues()
	if err != nil {
		return err
	}
	if recipe.OldText != "" && values[id] != recipe.OldText {
		return fmt.Errorf("string id %d text %q does not match required old_text %q", id, values[id], recipe.OldText)
	}
	values[id] = ""
	return f.setScenarioStringValues(values)
}

func (f *File) planStringSlot(recipe StringRecipe, allowEmptySearch bool) (int, error) {
	values, err := f.scenarioStringValues()
	if err != nil {
		return 0, err
	}
	if recipe.ID != nil {
		id := *recipe.ID
		if id < 0 || id >= len(values) {
			return 0, fmt.Errorf("string id %d out of range 0..%d", id, len(values)-1)
		}
		return id, nil
	}
	if recipe.OldText != "" {
		found := -1
		for i, value := range values {
			if value == recipe.OldText {
				if found >= 0 {
					return 0, fmt.Errorf("old_text %q matches multiple string ids; supply id", recipe.OldText)
				}
				found = i
			}
		}
		if found >= 0 {
			return found, nil
		}
		if isTrue(recipe.Required) {
			return 0, fmt.Errorf("old_text %q not found", recipe.OldText)
		}
	}
	if allowEmptySearch {
		for i, value := range values {
			if value == "" {
				return i, nil
			}
		}
		return 0, fmt.Errorf("no empty scenario string slots available")
	}
	return 0, fmt.Errorf("%s requires id or old_text", recipe.Op)
}

func (f *File) scenarioStringValues() ([]string, error) {
	playerDataTwo := f.root.section("PlayerDataTwo")
	if playerDataTwo == nil {
		return nil, fmt.Errorf("missing PlayerDataTwo section")
	}
	values := playerDataTwo.stringList("strings")
	if len(values) == 0 {
		return nil, fmt.Errorf("missing PlayerDataTwo strings")
	}
	return values, nil
}

func (f *File) setScenarioStringValues(values []string) error {
	playerDataTwo := f.root.section("PlayerDataTwo")
	if playerDataTwo == nil {
		return fmt.Errorf("missing PlayerDataTwo section")
	}
	if err := setStringListField(playerDataTwo, "strings", "str16", values); err != nil {
		return err
	}
	f.body = f.root.raw()
	f.InflatedBytes = len(f.body)
	return nil
}

func (r VariableRecipe) summary() string {
	switch r.Op {
	case "add_variable":
		if r.ID != nil {
			return fmt.Sprintf("%d:%s", *r.ID, r.Name)
		}
		return r.Name
	case "edit_variable", "tombstone_variable", "remove_variable":
		if r.TargetID != nil {
			if r.SetName != nil {
				return fmt.Sprintf("%d->%s", *r.TargetID, *r.SetName)
			}
			return fmt.Sprintf("%d", *r.TargetID)
		}
		if r.TargetName != "" {
			if r.SetName != nil {
				return fmt.Sprintf("%s->%s", r.TargetName, *r.SetName)
			}
			return r.TargetName
		}
	}
	return r.Op
}

func (f *File) AddVariable(recipe VariableRecipe) error {
	if recipe.Name == "" {
		return fmt.Errorf("add_variable requires name")
	}
	id := f.nextVariableID()
	if recipe.ID != nil {
		id = *recipe.ID
		if id < 0 {
			return fmt.Errorf("variable id %d must be non-negative", id)
		}
		if _, err := f.findVariableByID(id); err == nil {
			return fmt.Errorf("variable id %d already exists", id)
		}
	}
	triggers := f.root.section("Triggers")
	if triggers == nil {
		return fmt.Errorf("missing Triggers section")
	}
	field := triggers.field("variable_data")
	if field == nil {
		return fmt.Errorf("missing variable_data")
	}
	variableSpec, err := f.variableStructSpec()
	if err != nil {
		return err
	}
	raw, err := buildStructRaw(variableSpec, map[string]any{
		"variable_id":   id,
		"variable_name": recipe.Name,
	})
	if err != nil {
		return err
	}
	parser := parser{data: raw, sections: map[string]*parsedSection{}}
	node, err := parser.parseNode("VariableStruct", variableSpec, nil)
	if err != nil {
		return err
	}
	if parser.off != len(raw) {
		return fmt.Errorf("new variable parse stopped at %d of %d", parser.off, len(raw))
	}
	field.Elements = append(field.Elements, node)
	if err := setIntField(triggers, "number_of_variables", "u32", len(field.Elements)); err != nil {
		return err
	}
	return f.refreshTriggers()
}

func (f *File) EditVariable(recipe VariableRecipe) error {
	node, err := f.findVariable(recipe)
	if err != nil {
		return err
	}
	name := recipe.Name
	if recipe.SetName != nil {
		name = *recipe.SetName
	}
	if name == "" {
		return fmt.Errorf("edit_variable requires set_name or name")
	}
	if err := setStringField(node, "variable_name", "str32", name); err != nil {
		return err
	}
	return f.refreshTriggers()
}

func (f *File) TombstoneVariable(recipe VariableRecipe) error {
	node, err := f.findVariable(recipe)
	if err != nil {
		return err
	}
	id, _ := node.intValue("variable_id")
	refs, err := f.References(ReferenceOptions{Kind: "variable", TargetID: &id})
	if err != nil {
		return err
	}
	if refs.Summary.Total > 0 {
		return fmt.Errorf("refuses to tombstone variable %d because it is referenced %d time(s)", id, refs.Summary.Total)
	}
	name := fmt.Sprintf("_DeletedVariable%d", id)
	if recipe.SetName != nil {
		name = *recipe.SetName
	}
	if name == "" {
		return fmt.Errorf("tombstone_variable set_name cannot be empty")
	}
	if err := setStringField(node, "variable_name", "str32", name); err != nil {
		return err
	}
	return f.refreshTriggers()
}

func (f *File) RemoveVariable(recipe VariableRecipe) error {
	node, index, err := f.findVariableWithIndex(recipe)
	if err != nil {
		return err
	}
	id, _ := node.intValue("variable_id")
	refs, err := f.References(ReferenceOptions{Kind: "variable", TargetID: &id})
	if err != nil {
		return err
	}
	if refs.Summary.Total > 0 {
		return fmt.Errorf("refuses to remove variable %d because it is referenced %d time(s)", id, refs.Summary.Total)
	}
	triggers := f.root.section("Triggers")
	if triggers == nil {
		return fmt.Errorf("missing Triggers section")
	}
	field := triggers.field("variable_data")
	if field == nil {
		return fmt.Errorf("missing variable_data")
	}
	field.Elements = append(field.Elements[:index], field.Elements[index+1:]...)
	if err := setIntField(triggers, "number_of_variables", "u32", len(field.Elements)); err != nil {
		return err
	}
	return f.refreshTriggers()
}

func (f *File) AddUnit(recipe UnitRecipe) error {
	return f.AddUnits([]UnitRecipe{recipe})
}

func (f *File) AddUnits(recipes []UnitRecipe) error {
	if len(recipes) == 0 {
		return nil
	}
	unitsSection := f.root.section("Units")
	if unitsSection == nil {
		return fmt.Errorf("missing Units section")
	}
	playerSections := unitsSection.list("players_units")
	spec, err := f.writeSpec()
	if err != nil {
		return err
	}
	unitSpec, err := unitStructSpec(spec)
	if err != nil {
		return err
	}
	nextReferenceID := f.nextUnitReferenceID()
	// Reserve explicit IDs before allocating automatic ones: later entries may
	// name objects referenced by XS or selected-object trigger effects.
	reserved := make(map[int]bool)
	for _, player := range playerSections {
		for _, unit := range player.list("units") {
			if id, ok := unit.intValue("reference_id"); ok {
				reserved[id] = true
			}
		}
	}
	for _, recipe := range recipes {
		if recipe.ReferenceID != nil {
			id := *recipe.ReferenceID
			if reserved[id] {
				return fmt.Errorf("add_unit duplicate reference_id %d", id)
			}
			reserved[id] = true
		}
	}
	for _, recipe := range recipes {
		if recipe.Op != "add_unit" {
			return fmt.Errorf("unsupported unit op %q", recipe.Op)
		}
		if err := validateUnitPlacementSnap(recipe.Snap); err != nil {
			return err
		}
		if recipe.Player < 0 || recipe.Player >= len(playerSections) {
			return fmt.Errorf("unit player %d out of range 0..%d", recipe.Player, len(playerSections)-1)
		}
		fallbackReferenceID := nextReferenceID
		if recipe.ReferenceID != nil {
			if *recipe.ReferenceID >= nextReferenceID {
				nextReferenceID = *recipe.ReferenceID + 1
			}
		} else {
			for reserved[nextReferenceID] {
				nextReferenceID++
			}
			fallbackReferenceID = nextReferenceID
			nextReferenceID++
		}
		// Direct AddUnits callers do not provide a DAT resolver. Treat auto as
		// the conservative grid policy here; ApplyRecipe resolves auto against
		// collision data before reaching this method.
		placed, err := resolveUnitPlacement(recipe, nil)
		if err != nil {
			return err
		}
		raw, err := buildUnitFromRecipe(spec, placed, fallbackReferenceID)
		if err != nil {
			return err
		}
		parser := parser{data: raw, sections: map[string]*parsedSection{}}
		unitNode, err := parser.parseNode("UnitStruct", unitSpec, nil)
		if err != nil {
			return err
		}
		if parser.off != len(raw) {
			return fmt.Errorf("new unit parse stopped at %d of %d", parser.off, len(raw))
		}
		playerSection := playerSections[recipe.Player]
		unitField := playerSection.field("units")
		if unitField == nil {
			return fmt.Errorf("missing units field for player %d", recipe.Player)
		}
		unitField.Elements = append(unitField.Elements, unitNode)
		if err := setIntField(playerSection, "unit_count", "u32", len(unitField.Elements)); err != nil {
			return err
		}
	}
	return f.refreshUnits()
}

func (f *File) validateClusterScatter(recipe UnitRecipe) (int, error) {
	if recipe.UnitConst <= 0 {
		return 0, fmt.Errorf("cluster_scatter requires positive unit_const")
	}
	if recipe.Count == nil || *recipe.Count < 0 {
		return 0, fmt.Errorf("cluster_scatter requires non-negative count")
	}
	if *recipe.Count == 0 {
		return 0, nil
	}
	if len(recipe.Centers) == 0 && (recipe.TargetAreaX1 == nil || recipe.TargetAreaY1 == nil || recipe.TargetAreaX2 == nil || recipe.TargetAreaY2 == nil) {
		return 0, fmt.Errorf("cluster_scatter requires centers or target_area")
	}
	if recipe.Spread != nil && *recipe.Spread <= 0 {
		return 0, fmt.Errorf("cluster_scatter spread must be > 0")
	}
	if recipe.ClusterShape != "" && recipe.ClusterShape != "random" && recipe.ClusterShape != "packed" {
		return 0, fmt.Errorf("cluster_scatter cluster_shape must be random or packed")
	}
	if recipe.Snap != "" && recipe.Snap != "grid" && recipe.Snap != "fractional" && recipe.Snap != "auto" {
		return 0, fmt.Errorf("cluster_scatter snap must be grid, fractional, or auto")
	}
	if recipe.Distribution != "" && recipe.Distribution != "contiguous" && recipe.Distribution != "dispersed" {
		return 0, fmt.Errorf("cluster_scatter distribution must be contiguous or dispersed")
	}
	if recipe.Jitter != nil && *recipe.Jitter < 0 {
		return 0, fmt.Errorf("cluster_scatter jitter must be >= 0")
	}
	if recipe.MinDistance != nil && *recipe.MinDistance < 0 {
		return 0, fmt.Errorf("cluster_scatter min_distance must be >= 0")
	}
	return *recipe.Count, nil
}

func (f *File) ClusterScatterUnits(recipe UnitRecipe) ([]UnitRecipe, error) {
	return f.clusterScatterUnits(recipe, nil)
}

func validateUnitPlacementSnap(snap string) error {
	if snap != "" && snap != "grid" && snap != "fractional" && snap != "auto" {
		return fmt.Errorf("unit placement snap must be grid, fractional, or auto")
	}
	return nil
}

// resolveUnitPlacement makes the writer's placement policy explicit. An
// omitted snap defaults to tile-center placement; explicit fractional preserves
// sub-tile coordinates, and auto uses DAT collision semantics when available.
// Without a DAT, auto is conservative.
func resolveUnitPlacement(recipe UnitRecipe, resolver *clusterFootprintResolver) (UnitRecipe, error) {
	if err := validateUnitPlacementSnap(recipe.Snap); err != nil {
		return UnitRecipe{}, err
	}
	if recipe.Snap == "fractional" {
		return recipe, nil
	}
	snap := recipe.Snap
	if snap == "" {
		snap = "grid"
	}
	if snap == "auto" {
		fractional := false
		if resolver != nil {
			var err error
			fractional, err = resolver.unitCanUseFractional(recipe.UnitConst)
			if err != nil {
				return UnitRecipe{}, fmt.Errorf("resolve unit %d placement: %w", recipe.UnitConst, err)
			}
		}
		if fractional {
			return recipe, nil
		}
		snap = "grid"
	}
	if snap == "grid" {
		if recipe.X != nil {
			x := math.Floor(*recipe.X) + 0.5
			recipe.X = &x
		}
		if recipe.Y != nil {
			y := math.Floor(*recipe.Y) + 0.5
			recipe.Y = &y
		}
	}
	return recipe, nil
}

type clusterFootprintResolver struct {
	cache *datcache.Cache
	memo  map[int]bool
}

func newClusterFootprintResolver(datPath string) (*clusterFootprintResolver, error) {
	if strings.TrimSpace(datPath) == "" {
		return nil, nil
	}
	cache, err := datcache.Open(datPath)
	if err != nil {
		return nil, err
	}
	return &clusterFootprintResolver{cache: cache, memo: map[int]bool{}}, nil
}

func (r *clusterFootprintResolver) Close() error {
	if r == nil {
		return nil
	}
	return r.cache.Close()
}

func (r *clusterFootprintResolver) unitCanUseFractional(unitConst int) (bool, error) {
	if r == nil {
		return false, nil
	}
	if v, ok := r.memo[unitConst]; ok {
		return v, nil
	}
	id := unitConst
	rows, _, err := r.cache.QueryUnits(datcache.UnitQuery{ID: &id, All: true})
	if err != nil {
		return false, err
	}
	if len(rows) == 0 {
		r.memo[unitConst] = false
		return false, nil
	}
	for _, row := range rows {
		if footprintBlocks(row.CollisionSizeX, row.CollisionSizeY, row.CollisionSizeZ) {
			r.memo[unitConst] = false
			return false, nil
		}
	}
	r.memo[unitConst] = true
	return true, nil
}

// footprintBlocks reports whether a unit's DAT collision box obstructs
// pathing. A zero footprint never blocks. A nonzero footprint with ZERO
// HEIGHT does not block either: the engine ignores flat collision boxes
// (cactus 709, rugs 711, crater 723, flags, flares, bonfire, smoke all have
// collision_size_z == 0 and land units walk straight over them — verified
// in-game for cactus and rugs, 2026-09-17). Trees and statues have
// z of 2-3 and block.
func footprintBlocks(x, y, z float32) bool {
	if x == 0 && y == 0 {
		return false
	}
	return z != 0
}

type clusterScatterOptions struct {
	Snap         string
	Distribution string
	Jitter       float64
}

func resolveClusterScatterOptions(recipe UnitRecipe, resolver *clusterFootprintResolver) (clusterScatterOptions, error) {
	opts := clusterScatterOptions{Snap: "auto", Distribution: "dispersed"}
	switch recipe.ClusterShape {
	case "packed":
		opts.Distribution = "contiguous"
	case "random", "":
		opts.Distribution = "dispersed"
	}
	if recipe.Snap != "" {
		opts.Snap = recipe.Snap
	}
	if recipe.Distribution != "" {
		opts.Distribution = recipe.Distribution
	}
	if recipe.Jitter != nil {
		opts.Jitter = *recipe.Jitter
	}
	if opts.Snap == "auto" {
		fractional, err := resolver.unitCanUseFractional(recipe.UnitConst)
		if err != nil {
			return clusterScatterOptions{}, err
		}
		if fractional {
			opts.Snap = "fractional"
		} else {
			opts.Snap = "grid"
		}
	}
	if opts.Snap == "grid" {
		opts.Jitter = 0
	}
	return opts, nil
}

func (f *File) clusterScatterUnits(recipe UnitRecipe, resolver *clusterFootprintResolver) ([]UnitRecipe, error) {
	count, err := f.validateClusterScatter(recipe)
	if err != nil {
		return nil, err
	}
	if count == 0 {
		return nil, nil
	}
	opts, err := resolveClusterScatterOptions(recipe, resolver)
	if err != nil {
		return nil, err
	}
	tiles, width, height, err := f.mapTiles()
	if err != nil {
		return nil, err
	}
	terrainIDs := make([]int, len(tiles))
	for i, tile := range tiles {
		id, _ := tile.intValue("terrain_id")
		terrainIDs[i] = id
	}
	allowed := intSet(recipe.AllowedTerrainIDs)
	x1, y1, x2, y2 := 0.0, 0.0, float64(width), float64(height)
	if recipe.TargetAreaX1 != nil {
		x1 = *recipe.TargetAreaX1
	}
	if recipe.TargetAreaY1 != nil {
		y1 = *recipe.TargetAreaY1
	}
	if recipe.TargetAreaX2 != nil {
		x2 = *recipe.TargetAreaX2
	}
	if recipe.TargetAreaY2 != nil {
		y2 = *recipe.TargetAreaY2
	}
	if x1 > x2 || y1 > y2 {
		return nil, fmt.Errorf("cluster_scatter target area is inverted")
	}
	centers := append([]PointRecipe(nil), recipe.Centers...)
	if len(centers) == 0 {
		centers = []PointRecipe{{X: (x1 + x2) / 2, Y: (y1 + y2) / 2}}
	}
	spread := 4.0
	if recipe.Spread != nil {
		spread = *recipe.Spread
	}
	minDistance := 0.75
	if recipe.MinDistance != nil {
		minDistance = *recipe.MinDistance
	}
	exact := true
	if recipe.Exact != nil {
		exact = *recipe.Exact
	}
	seed := int64(0)
	if recipe.Seed != nil {
		seed = int64(*recipe.Seed)
	}
	r := rand.New(rand.NewSource(seed))
	var placed []PointRecipe
	var adds []UnitRecipe
	for i := 0; i < count; i++ {
		center := centers[i%len(centers)]
		var x, y float64
		var ok bool
		if opts.Distribution == "contiguous" {
			x, y, ok = scatterContiguousPoint(r, center, spread, x1, y1, x2, y2, width, height, terrainIDs, allowed, placed, minDistance, opts)
		} else {
			x, y, ok = scatterDispersedPoint(r, center, spread, x1, y1, x2, y2, width, height, terrainIDs, allowed, placed, minDistance, opts)
		}
		if !ok && opts.Distribution != "contiguous" {
			x, y, ok = scatterFallbackPoint(r, center, x1, y1, x2, y2, width, height, terrainIDs, allowed, placed, minDistance, opts)
		}
		if !ok {
			if exact {
				return nil, fmt.Errorf("cluster_scatter placed %d/%d unit(s) before habitat/min-distance constraints exhausted", len(adds), count)
			}
			continue
		}
		placed = append(placed, PointRecipe{X: x, Y: y})
		add := recipe
		add.Op = "add_unit"
		// Carry the resolved policy into AddUnits so auto uses the DAT-backed
		// decision made above rather than being conservatively re-resolved.
		add.Snap = opts.Snap
		add.X = ptrFloat64(x)
		add.Y = ptrFloat64(y)
		if add.Z == nil {
			add.Z = ptrFloat64(0)
		}
		if add.Status == nil {
			add.Status = ptrInt(2)
		}
		if add.Rotation == nil {
			if len(recipe.RotationChoices) > 0 {
				add.Rotation = ptrFloat64(recipe.RotationChoices[r.Intn(len(recipe.RotationChoices))])
			} else {
				add.Rotation = ptrFloat64(r.Float64() * 31)
			}
		}
		if recipe.ReferenceIDBase != nil {
			ref := *recipe.ReferenceIDBase + i
			add.ReferenceID = &ref
		}
		if recipe.CaptionSuffix != "" && recipe.CaptionString != "" {
			add.CaptionString = fmt.Sprintf("%s%s%d", recipe.CaptionString, recipe.CaptionSuffix, i+1)
		}
		adds = append(adds, add)
	}
	return adds, nil
}

func scatterDispersedPoint(r *rand.Rand, center PointRecipe, spread, x1, y1, x2, y2 float64, width, height int, terrainIDs []int, allowed map[int]bool, placed []PointRecipe, minDistance float64, opts clusterScatterOptions) (float64, float64, bool) {
	for attempt := 0; attempt < 800; attempt++ {
		localSpread := spread
		if attempt > 400 {
			localSpread *= 1.75
		}
		x := center.X + r.NormFloat64()*localSpread
		y := center.Y + r.NormFloat64()*localSpread
		x, y = applyScatterSnapJitter(r, x, y, opts)
		if validScatterPoint(x, y, x1, y1, x2, y2, width, height, terrainIDs, allowed, placed, minDistance) {
			return x, y, true
		}
	}
	return 0, 0, false
}

func scatterContiguousPoint(r *rand.Rand, center PointRecipe, spread, x1, y1, x2, y2 float64, width, height int, terrainIDs []int, allowed map[int]bool, placed []PointRecipe, minDistance float64, opts clusterScatterOptions) (float64, float64, bool) {
	maxRadius := maxInt(1, int(math.Ceil(spread)))
	cx, cy := int(math.Floor(center.X)), int(math.Floor(center.Y))
	type candidate struct {
		x, y  float64
		score float64
	}
	candidates := make([]candidate, 0, (maxRadius*2+1)*(maxRadius*2+1))
	for dy := -maxRadius; dy <= maxRadius; dy++ {
		for dx := -maxRadius; dx <= maxRadius; dx++ {
			tx, ty := cx+dx, cy+dy
			if tx < 0 || ty < 0 || tx >= width || ty >= height {
				continue
			}
			x := float64(tx) + 0.5
			y := float64(ty) + 0.5
			x, y = applyScatterSnapJitter(r, x, y, opts)
			dist := math.Hypot(x-center.X, y-center.Y)
			if dist > spread+0.75 {
				continue
			}
			if !validScatterPoint(x, y, x1, y1, x2, y2, width, height, terrainIDs, allowed, placed, minDistance) {
				continue
			}
			candidates = append(candidates, candidate{x: x, y: y, score: dist + r.Float64()*0.15})
		}
	}
	if len(candidates) == 0 {
		return 0, 0, false
	}
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].score < candidates[j].score
	})
	return candidates[0].x, candidates[0].y, true
}

func scatterFallbackPoint(r *rand.Rand, center PointRecipe, x1, y1, x2, y2 float64, width, height int, terrainIDs []int, allowed map[int]bool, placed []PointRecipe, minDistance float64, opts clusterScatterOptions) (float64, float64, bool) {
	bestX, bestY := 0.0, 0.0
	bestDist := math.MaxFloat64
	for yy := maxInt(0, int(math.Floor(y1))); yy <= minInt(height-1, int(math.Ceil(y2))); yy++ {
		for xx := maxInt(0, int(math.Floor(x1))); xx <= minInt(width-1, int(math.Ceil(x2))); xx++ {
			x, y := float64(xx)+0.5, float64(yy)+0.5
			x, y = applyScatterSnapJitter(r, x, y, opts)
			if !validScatterPoint(x, y, x1, y1, x2, y2, width, height, terrainIDs, allowed, placed, minDistance) {
				continue
			}
			d := math.Hypot(x-center.X, y-center.Y)
			if d < bestDist {
				bestDist = d
				bestX, bestY = x, y
			}
		}
	}
	if bestDist == math.MaxFloat64 {
		return 0, 0, false
	}
	return bestX, bestY, true
}

func applyScatterSnapJitter(r *rand.Rand, x, y float64, opts clusterScatterOptions) (float64, float64) {
	if opts.Snap == "grid" {
		return math.Floor(x) + 0.5, math.Floor(y) + 0.5
	}
	if opts.Jitter > 0 {
		x += (r.Float64()*2 - 1) * opts.Jitter
		y += (r.Float64()*2 - 1) * opts.Jitter
	}
	return x, y
}

func validScatterPoint(x, y, x1, y1, x2, y2 float64, width, height int, terrainIDs []int, allowed map[int]bool, placed []PointRecipe, minDistance float64) bool {
	if x < x1 || y < y1 || x > x2 || y > y2 {
		return false
	}
	tx, ty := int(math.Floor(x)), int(math.Floor(y))
	if tx < 0 || ty < 0 || tx >= width || ty >= height {
		return false
	}
	if len(allowed) > 0 && !allowed[terrainIDs[ty*width+tx]] {
		return false
	}
	for _, p := range placed {
		if math.Hypot(x-p.X, y-p.Y) < minDistance {
			return false
		}
	}
	return true
}

func (f *File) EditUnit(recipe UnitRecipe) error {
	unitNode, playerSection, unitIndex, err := f.findUnit(recipe)
	if err != nil {
		return err
	}
	if recipe.X != nil {
		if err := setFloatField(unitNode, "x", "f32", *recipe.X); err != nil {
			return err
		}
	}
	if recipe.Y != nil {
		if err := setFloatField(unitNode, "y", "f32", *recipe.Y); err != nil {
			return err
		}
	}
	if recipe.Z != nil {
		if err := setFloatField(unitNode, "z", "f32", *recipe.Z); err != nil {
			return err
		}
	}
	if recipe.UnitConst > 0 {
		if err := setIntField(unitNode, "unit_const", "u16", recipe.UnitConst); err != nil {
			return err
		}
	}
	if recipe.Status != nil {
		if err := setIntField(unitNode, "status", "u8", *recipe.Status); err != nil {
			return err
		}
	}
	if recipe.Rotation != nil {
		if err := setFloatField(unitNode, "rotation", "f32", *recipe.Rotation); err != nil {
			return err
		}
	}
	if recipe.InitialAnimationFrame != nil {
		if err := setIntField(unitNode, "initial_animation_frame", "u16", *recipe.InitialAnimationFrame); err != nil {
			return err
		}
	}
	if recipe.GarrisonedInID != nil {
		if err := setIntField(unitNode, "garrisoned_in_id", "s32", *recipe.GarrisonedInID); err != nil {
			return err
		}
	}
	if recipe.CaptionStringID != nil {
		if err := setIntField(unitNode, "caption_string_id", "s32", *recipe.CaptionStringID); err != nil {
			return err
		}
	}
	if recipe.CaptionString != "" {
		if err := setStringField(unitNode, "caption_string", "str32", recipe.CaptionString); err != nil {
			return err
		}
	}
	if recipe.SetPlayer != nil {
		if err := f.moveUnitToPlayer(unitNode, playerSection, unitIndex, *recipe.SetPlayer); err != nil {
			return err
		}
	}
	return f.refreshUnits()
}

func (f *File) RemoveUnit(recipe UnitRecipe) error {
	unit, playerSection, unitIndex, err := f.findUnit(recipe)
	if err != nil {
		return err
	}
	if referenceID, ok := unit.intValue("reference_id"); ok && referenceID >= 0 {
		if err := f.ensureUnitNotReferenced(referenceID); err != nil {
			return err
		}
	}
	unitField := playerSection.field("units")
	if unitField == nil {
		return fmt.Errorf("missing units field")
	}
	unitField.Elements = append(unitField.Elements[:unitIndex], unitField.Elements[unitIndex+1:]...)
	if err := setIntField(playerSection, "unit_count", "u32", len(unitField.Elements)); err != nil {
		return err
	}
	return f.refreshUnits()
}

type unitAreaMatch struct {
	Unit          *parsedNode
	PlayerSection *parsedNode
	Player        int
	Index         int
}

func (f *File) RemoveUnitsInArea(recipe UnitRecipe) (int, error) {
	targets, err := f.findUnitsInArea(recipe)
	if err != nil {
		return 0, err
	}
	for _, target := range targets {
		if referenceID, ok := target.Unit.intValue("reference_id"); ok && referenceID >= 0 {
			if err := f.ensureUnitNotReferenced(referenceID); err != nil {
				return 0, err
			}
		}
	}
	byPlayer := map[int][]unitAreaMatch{}
	for _, target := range targets {
		byPlayer[target.Player] = append(byPlayer[target.Player], target)
	}
	for _, matches := range byPlayer {
		sort.Slice(matches, func(i, j int) bool {
			return matches[i].Index > matches[j].Index
		})
		playerSection := matches[0].PlayerSection
		unitField := playerSection.field("units")
		if unitField == nil {
			return 0, fmt.Errorf("missing units field for player %d", matches[0].Player)
		}
		for _, target := range matches {
			if target.Index < 0 || target.Index >= len(unitField.Elements) || unitField.Elements[target.Index] != target.Unit {
				return 0, fmt.Errorf("unit index changed before area removal for player %d index %d", target.Player, target.Index)
			}
			unitField.Elements = append(unitField.Elements[:target.Index], unitField.Elements[target.Index+1:]...)
		}
		if err := setIntField(playerSection, "unit_count", "u32", len(unitField.Elements)); err != nil {
			return 0, err
		}
	}
	if err := f.refreshUnits(); err != nil {
		return 0, err
	}
	return len(targets), nil
}

func (f *File) RemoveUnitsForPlayer(recipe UnitRecipe) (int, error) {
	targets, err := f.findUnitsByPlayer(recipe.TargetPlayer)
	if err != nil {
		return 0, err
	}
	for _, target := range targets {
		if referenceID, ok := target.Unit.intValue("reference_id"); ok && referenceID >= 0 {
			if err := f.ensureUnitNotReferenced(referenceID); err != nil {
				return 0, err
			}
		}
	}
	playerSection := targets[0].PlayerSection
	unitField := playerSection.field("units")
	if unitField == nil {
		return 0, fmt.Errorf("missing units field for player %d", targets[0].Player)
	}
	unitField.Elements = nil
	if err := setIntField(playerSection, "unit_count", "u32", 0); err != nil {
		return 0, err
	}
	if err := f.refreshUnits(); err != nil {
		return 0, err
	}
	return len(targets), nil
}

func (f *File) CopyUnitsInArea(recipe UnitRecipe) (int, error) {
	targets, err := f.findUnitsInArea(recipe)
	if err != nil {
		return 0, err
	}
	if err := f.validateCopiedUnitReferenceIDs(targets, recipe); err != nil {
		return 0, err
	}
	spec, err := f.writeSpec()
	if err != nil {
		return 0, err
	}
	refMap := copiedUnitReferenceMap(targets, copiedUnitReferenceBase(f, recipe))
	for _, target := range targets {
		add, err := copiedUnitRecipe(target, recipe, refMap)
		if err != nil {
			return 0, err
		}
		raw, err := buildUnitFromRecipe(spec, add, *add.ReferenceID)
		if err != nil {
			return 0, err
		}
		if err := f.addRawUnit(raw, add.Player); err != nil {
			return 0, err
		}
	}
	return len(targets), nil
}

func (f *File) MoveUnitsInArea(recipe UnitRecipe) (int, error) {
	targets, err := f.findUnitsInArea(recipe)
	if err != nil {
		return 0, err
	}
	if err := f.validateMoveUnitsInArea(targets, recipe); err != nil {
		return 0, err
	}
	offsetX, offsetY, err := unitAreaOffset(recipe)
	if err != nil {
		return 0, err
	}
	for _, target := range targets {
		x, _ := target.Unit.floatValue("x")
		y, _ := target.Unit.floatValue("y")
		x += offsetX
		y += offsetY
		if err := setFloatField(target.Unit, "x", "f32", x); err != nil {
			return 0, err
		}
		if err := setFloatField(target.Unit, "y", "f32", y); err != nil {
			return 0, err
		}
	}
	if err := f.refreshUnits(); err != nil {
		return 0, err
	}
	return len(targets), nil
}

func (f *File) EditUnitsInArea(recipe UnitRecipe) (int, error) {
	targets, err := f.findUnitsInArea(recipe)
	if err != nil {
		return 0, err
	}
	if err := f.validateEditUnitsInArea(recipe); err != nil {
		return 0, err
	}
	for _, target := range targets {
		if err := applyUnitAreaFieldEdits(target.Unit, recipe); err != nil {
			return 0, err
		}
	}
	if recipe.SetPlayer != nil {
		byPlayer := map[int][]unitAreaMatch{}
		for _, target := range targets {
			byPlayer[target.Player] = append(byPlayer[target.Player], target)
		}
		for _, matches := range byPlayer {
			sort.Slice(matches, func(i, j int) bool {
				return matches[i].Index > matches[j].Index
			})
			for _, target := range matches {
				if err := f.moveUnitToPlayer(target.Unit, target.PlayerSection, target.Index, *recipe.SetPlayer); err != nil {
					return 0, err
				}
			}
		}
	}
	if err := f.refreshUnits(); err != nil {
		return 0, err
	}
	return len(targets), nil
}

func (f *File) SetTerrainRect(recipe MapRecipe) error {
	recipe.Op = "set_terrain_rect"
	return f.SetTerrain(recipe)
}

type terrainGridPayload struct {
	Width     int   `json:"width"`
	Height    int   `json:"height"`
	TerrainID []int `json:"terrain_id"`
	Layer     []int `json:"layer"`
	Elevation []int `json:"elevation,omitempty"`
}

type resolvedTerrainGrid struct {
	X1        int
	Y1        int
	Width     int
	Height    int
	TerrainID []int
	Layer     []int
	Elevation []int
}

func (f *File) ApplyTerrainGrid(recipe MapRecipe) error {
	tiles, mapWidth, mapHeight, err := f.mapTiles()
	if err != nil {
		return err
	}
	grid, err := resolveTerrainGridRecipe(recipe, mapWidth, mapHeight)
	if err != nil {
		return err
	}
	for y := 0; y < grid.Height; y++ {
		for x := 0; x < grid.Width; x++ {
			src := y*grid.Width + x
			dst := (grid.Y1+y)*mapWidth + grid.X1 + x
			tile := tiles[dst]
			if err := setIntField(tile, "terrain_id", "u8", grid.TerrainID[src]); err != nil {
				return err
			}
			if err := setIntField(tile, "layer", "s16", grid.Layer[src]); err != nil {
				return err
			}
			if grid.Elevation != nil {
				if err := setIntField(tile, "elevation", "u8", grid.Elevation[src]); err != nil {
					return err
				}
			}
		}
	}
	return f.refreshMapAfterTerrainEdit()
}

func resolveTerrainGridRecipe(recipe MapRecipe, mapWidth, mapHeight int) (resolvedTerrainGrid, error) {
	loaded := terrainGridPayload{
		TerrainID: append([]int(nil), recipe.TerrainIDGrid...),
		Layer:     append([]int(nil), recipe.LayerGrid...),
		Elevation: append([]int(nil), recipe.ElevationGrid...),
	}
	if recipe.Width != nil {
		loaded.Width = *recipe.Width
	}
	if recipe.Height != nil {
		loaded.Height = *recipe.Height
	}
	if recipe.GridFile != "" {
		fromFile, err := loadTerrainGridPayload(recipe)
		if err != nil {
			return resolvedTerrainGrid{}, err
		}
		loaded = fromFile
		if recipe.Width != nil {
			loaded.Width = *recipe.Width
		}
		if recipe.Height != nil {
			loaded.Height = *recipe.Height
		}
		if len(recipe.TerrainIDGrid) > 0 {
			loaded.TerrainID = append([]int(nil), recipe.TerrainIDGrid...)
		}
		if len(recipe.LayerGrid) > 0 {
			loaded.Layer = append([]int(nil), recipe.LayerGrid...)
		}
		if len(recipe.ElevationGrid) > 0 {
			loaded.Elevation = append([]int(nil), recipe.ElevationGrid...)
		}
	}
	if loaded.Width == 0 {
		loaded.Width = mapWidth - recipe.X1
	}
	if loaded.Height == 0 {
		loaded.Height = mapHeight - recipe.Y1
	}
	grid := resolvedTerrainGrid{
		X1:        recipe.X1,
		Y1:        recipe.Y1,
		Width:     loaded.Width,
		Height:    loaded.Height,
		TerrainID: loaded.TerrainID,
		Layer:     loaded.Layer,
		Elevation: loaded.Elevation,
	}
	if err := validateTerrainGrid(grid, mapWidth, mapHeight); err != nil {
		return resolvedTerrainGrid{}, err
	}
	return grid, nil
}

func loadTerrainGridPayload(recipe MapRecipe) (terrainGridPayload, error) {
	path := recipe.GridFile
	if recipe.baseDir != "" && !filepath.IsAbs(path) {
		path = filepath.Join(recipe.baseDir, path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return terrainGridPayload{}, fmt.Errorf("terrain_grid grid_file %q: %w", recipe.GridFile, err)
	}
	var payload terrainGridPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return terrainGridPayload{}, fmt.Errorf("terrain_grid grid_file %q: %w", recipe.GridFile, err)
	}
	return payload, nil
}

func validateTerrainGrid(grid resolvedTerrainGrid, mapWidth, mapHeight int) error {
	if grid.Width <= 0 || grid.Height <= 0 {
		return fmt.Errorf("terrain_grid width and height must be > 0")
	}
	if grid.X1 < 0 || grid.Y1 < 0 || grid.X1+grid.Width > mapWidth || grid.Y1+grid.Height > mapHeight {
		return fmt.Errorf("terrain_grid rectangle (%d,%d) %dx%d outside map %dx%d", grid.X1, grid.Y1, grid.Width, grid.Height, mapWidth, mapHeight)
	}
	want := grid.Width * grid.Height
	if len(grid.TerrainID) != want {
		return fmt.Errorf("terrain_grid terrain_id length = %d, want %d", len(grid.TerrainID), want)
	}
	if len(grid.Layer) != want {
		return fmt.Errorf("terrain_grid layer length = %d, want %d", len(grid.Layer), want)
	}
	if len(grid.Elevation) != 0 && len(grid.Elevation) != want {
		return fmt.Errorf("terrain_grid elevation length = %d, want 0 or %d", len(grid.Elevation), want)
	}
	for i, id := range grid.TerrainID {
		if id < 0 || id > 255 {
			return fmt.Errorf("terrain_grid terrain_id[%d] = %d outside u8 range", i, id)
		}
	}
	for i, elevation := range grid.Elevation {
		if elevation < 0 || elevation > 255 {
			return fmt.Errorf("terrain_grid elevation[%d] = %d outside u8 range", i, elevation)
		}
	}
	return nil
}

func ValidateScenarioMapSize(width, height int) error {
	if width != height {
		return fmt.Errorf("scenario map size %dx%d is non-square; kit scen blank only emits canonical square DE map-size presets", width, height)
	}
	if _, ok := ScenarioMapPresetBySize(width); !ok {
		return fmt.Errorf("scenario map size %d is not a canonical DE preset; valid sizes: %s", width, ScenarioMapPresetSizesString())
	}
	return nil
}

func ScenarioMapPresets() []ScenarioMapPreset {
	return append([]ScenarioMapPreset(nil), scenarioMapPresets...)
}

func ScenarioMapPresetBySize(size int) (ScenarioMapPreset, bool) {
	for _, preset := range scenarioMapPresets {
		if preset.Size == size {
			return preset, true
		}
	}
	return ScenarioMapPreset{}, false
}

func ScenarioMapPresetSizesString() string {
	parts := make([]string, 0, len(scenarioMapPresets))
	for _, preset := range scenarioMapPresets {
		parts = append(parts, fmt.Sprintf("%d=%s", preset.Size, preset.Name))
	}
	return strings.Join(parts, ", ")
}

func (f *File) ResizeMap(width, height int) error {
	if err := ValidateScenarioMapSize(width, height); err != nil {
		return err
	}
	mapSection := f.root.section("Map")
	if mapSection == nil {
		return fmt.Errorf("missing Map section")
	}
	oldTiles, oldWidth, oldHeight, err := f.mapTiles()
	if err != nil {
		return err
	}
	if len(oldTiles) == 0 {
		return fmt.Errorf("cannot resize map with no terrain tile template")
	}
	terrainField := mapSection.field("terrain_data")
	if terrainField == nil {
		return fmt.Errorf("missing terrain_data")
	}
	baseTile := oldTiles[0]
	newTiles := make([]*parsedNode, 0, width*height)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			src := baseTile
			if x < oldWidth && y < oldHeight {
				src = oldTiles[y*oldWidth+x]
			}
			newTiles = append(newTiles, cloneParsedNode(src))
		}
	}
	if err := setIntField(mapSection, "map_width", "s32", width); err != nil {
		return err
	}
	if err := setIntField(mapSection, "map_height", "s32", height); err != nil {
		return err
	}
	terrainField.Raw = nil
	terrainField.Elements = newTiles
	terrainField.Value = terrainField.Elements
	f.body = f.root.raw()
	f.InflatedBytes = len(f.body)
	if mapInfo, err := f.root.mapInfo(); err == nil {
		f.Map = mapInfo
	} else {
		return err
	}
	return nil
}

func (f *File) SetTerrain(recipe MapRecipe) error {
	return f.SetTerrainWithMasks(recipe, nil)
}

func (f *File) SetTerrainWithMasks(recipe MapRecipe, masks map[string]terrainMask) error {
	tiles, width, height, err := f.mapTiles()
	if err != nil {
		return err
	}
	points, err := mapPatchPointsWithMasks(recipe, width, height, masks)
	if err != nil {
		return err
	}
	if recipe.TerrainID == nil && recipe.Elevation == nil && recipe.Layer == nil {
		return fmt.Errorf("%s requires terrain_id, elevation, or layer", recipe.Op)
	}
	for _, point := range points {
		tile := tiles[point.Y*width+point.X]
		if recipe.TerrainID != nil {
			if err := setIntField(tile, "terrain_id", "u8", *recipe.TerrainID); err != nil {
				return err
			}
		}
		if recipe.Elevation != nil {
			if err := setIntField(tile, "elevation", "u8", *recipe.Elevation); err != nil {
				return err
			}
		}
		if recipe.Layer != nil {
			if err := setIntField(tile, "layer", "s16", *recipe.Layer); err != nil {
				return err
			}
		}
	}
	f.body = f.root.raw()
	f.InflatedBytes = len(f.body)
	if mapInfo, err := f.root.mapInfo(); err == nil {
		f.Map = mapInfo
	} else {
		return err
	}
	return nil
}

func (f *File) LayeredCrossfadeTerrain(recipe MapRecipe) error {
	return f.LayeredCrossfadeTerrainWithMasks(recipe, nil)
}

func (f *File) LayeredCrossfadeTerrainWithMasks(recipe MapRecipe, masks map[string]terrainMask) error {
	if recipe.TerrainID == nil || recipe.TerrainID2 == nil {
		return fmt.Errorf("layered_crossfade requires terrain_id bottom and terrain_id_2 top")
	}
	layer := *recipe.TerrainID2
	patch := recipe
	patch.TerrainID = recipe.TerrainID
	patch.Layer = &layer
	return f.SetTerrainWithMasks(patch, masks)
}

func (f *File) NoiseFillTerrain(recipe MapRecipe) error {
	return f.NoiseFillTerrainWithMasks(recipe, nil)
}

func (f *File) NoiseFillTerrainWithMasks(recipe MapRecipe, masks map[string]terrainMask) error {
	tiles, width, height, err := f.mapTiles()
	if err != nil {
		return err
	}
	if recipe.TerrainID == nil || recipe.TerrainID2 == nil {
		return fmt.Errorf("noise_fill requires terrain_id and terrain_id_2")
	}
	points, err := mapPatchPointsWithMasks(recipe, width, height, masks)
	if err != nil {
		return err
	}
	scale := 8.0
	if recipe.Scale != nil {
		scale = *recipe.Scale
	}
	if scale <= 0 {
		return fmt.Errorf("noise_fill scale must be > 0")
	}
	threshold := 0.5
	if recipe.Threshold != nil {
		threshold = *recipe.Threshold
	}
	seed := 0
	if recipe.Seed != nil {
		seed = *recipe.Seed
	}
	for _, point := range points {
		id := *recipe.TerrainID
		if valueNoise2D(float64(point.X)/scale, float64(point.Y)/scale, seed) >= threshold {
			id = *recipe.TerrainID2
		}
		if err := setIntField(tiles[point.Y*width+point.X], "terrain_id", "u8", id); err != nil {
			return err
		}
	}
	return f.refreshMapAfterTerrainEdit()
}

func (f *File) ErodeTerrain(recipe MapRecipe) error {
	return f.ErodeTerrainWithMasks(recipe, nil, false)
}

func (f *File) ErodeTerrainWithMasks(recipe MapRecipe, masks map[string]terrainMask, semantic bool) error {
	tiles, width, height, err := f.mapTiles()
	if err != nil {
		return err
	}
	points, err := mapPatchPointsWithMasks(recipe, width, height, masks)
	if err != nil {
		return err
	}
	iterations := 1
	if recipe.Iterations != nil {
		iterations = *recipe.Iterations
	}
	if iterations < 1 {
		return fmt.Errorf("erode iterations must be >= 1")
	}
	current := make([]int, len(tiles))
	for i, tile := range tiles {
		id, _ := tile.intValue("terrain_id")
		current[i] = id
	}
	pointSet := map[mapPoint]bool{}
	for _, point := range points {
		pointSet[point] = true
	}
	classSet := intSet(recipe.TerrainIDs)
	for iter := 0; iter < iterations; iter++ {
		next := append([]int(nil), current...)
		for _, point := range points {
			x, y := point.X, point.Y
			if semantic && len(classSet) > 0 && !classSet[current[y*width+x]] {
				continue
			}
			counts := map[int]int{}
			for dy := -1; dy <= 1; dy++ {
				for dx := -1; dx <= 1; dx++ {
					xx, yy := x+dx, y+dy
					if xx < 0 || yy < 0 || xx >= width || yy >= height {
						continue
					}
					if !pointSet[mapPoint{X: xx, Y: yy}] {
						continue
					}
					id := current[yy*width+xx]
					if semantic && len(classSet) > 0 && !classSet[id] {
						continue
					}
					counts[id]++
				}
			}
			bestID := current[y*width+x]
			bestCount := counts[bestID]
			for id, count := range counts {
				if count > bestCount || (count == bestCount && id == bestID) {
					bestID = id
					bestCount = count
				}
			}
			if bestCount >= 5 {
				next[y*width+x] = bestID
			}
		}
		current = next
	}
	for _, point := range points {
		if err := setIntField(tiles[point.Y*width+point.X], "terrain_id", "u8", current[point.Y*width+point.X]); err != nil {
			return err
		}
	}
	return f.refreshMapAfterTerrainEdit()
}

func (f *File) refreshMapAfterTerrainEdit() error {
	f.body = f.root.raw()
	f.InflatedBytes = len(f.body)
	if mapInfo, err := f.root.mapInfo(); err == nil {
		f.Map = mapInfo
	} else {
		return err
	}
	return nil
}

func (f *File) CopyTerrainArea(recipe MapRecipe) error {
	tiles, width, height, err := f.mapTiles()
	if err != nil {
		return err
	}
	if recipe.TargetX == nil || recipe.TargetY == nil {
		return fmt.Errorf("copy_terrain_area requires target_x and target_y")
	}
	if err := validateMapRect(recipe, width, height); err != nil {
		return err
	}
	srcWidth := recipe.X2 - recipe.X1 + 1
	srcHeight := recipe.Y2 - recipe.Y1 + 1
	target := MapRecipe{
		Op: "copy_terrain_area target",
		X1: *recipe.TargetX,
		Y1: *recipe.TargetY,
		X2: *recipe.TargetX + srcWidth - 1,
		Y2: *recipe.TargetY + srcHeight - 1,
	}
	if err := validateMapRect(target, width, height); err != nil {
		return err
	}
	snapshots := make([]terrainTileSnapshot, 0, srcWidth*srcHeight)
	for dy := 0; dy < srcHeight; dy++ {
		for dx := 0; dx < srcWidth; dx++ {
			src := tiles[(recipe.Y1+dy)*width+recipe.X1+dx]
			snapshot, err := snapshotTerrainTile(src)
			if err != nil {
				return err
			}
			snapshots = append(snapshots, snapshot)
		}
	}
	for dy := 0; dy < srcHeight; dy++ {
		for dx := 0; dx < srcWidth; dx++ {
			dst := tiles[(*recipe.TargetY+dy)*width+*recipe.TargetX+dx]
			if err := applyTerrainTileSnapshot(dst, snapshots[dy*srcWidth+dx]); err != nil {
				return err
			}
		}
	}
	f.body = f.root.raw()
	f.InflatedBytes = len(f.body)
	if mapInfo, err := f.root.mapInfo(); err == nil {
		f.Map = mapInfo
	} else {
		return err
	}
	return nil
}

func (f *File) AddTrigger(recipe TriggerRecipe) error {
	raw, err := f.buildTriggerRaw(recipe)
	if err != nil {
		return err
	}
	return f.addRawTrigger(raw)
}

func (f *File) buildTriggerRaw(recipe TriggerRecipe) ([]byte, error) {
	spec, err := f.writeSpec()
	if err != nil {
		return nil, err
	}
	raw, err := buildTriggerFromRecipe(spec, recipe)
	if err != nil {
		return nil, err
	}
	return raw, nil
}

func (f *File) CopyTrigger(recipe TriggerRecipe) error {
	source, err := f.findTrigger(recipe)
	if err != nil {
		return err
	}
	sourceName, _ := source.stringValue("trigger_name")
	raw := source.raw()
	if err := f.addRawTrigger(raw); err != nil {
		return err
	}
	triggers := f.root.section("Triggers")
	if triggers == nil {
		return fmt.Errorf("missing Triggers section")
	}
	newIndex := len(triggers.list("trigger_data")) - 1
	edit := recipe
	edit.Op = "edit_trigger"
	edit.TargetIndex = &newIndex
	edit.TargetName = ""
	if edit.SetName == nil {
		name := edit.Name
		if name == "" {
			name = strings.TrimSpace(sourceName + " copy")
			if name == "" {
				name = fmt.Sprintf("Trigger %d copy", newIndex)
			}
		}
		edit.SetName = &name
	}
	return f.EditTrigger(edit)
}

func (f *File) EditTrigger(recipe TriggerRecipe) error {
	spec, err := f.writeSpec()
	if err != nil {
		return err
	}
	trigger, err := f.findTrigger(recipe)
	if err != nil {
		return err
	}
	if recipe.Enabled != nil {
		if err := setIntField(trigger, "enabled", "u32", boolInt(*recipe.Enabled)); err != nil {
			return err
		}
	}
	if recipe.Looping != nil {
		if err := setIntField(trigger, "looping", "s8", boolInt(*recipe.Looping)); err != nil {
			return err
		}
	}
	if recipe.SetName != nil {
		if err := setStringField(trigger, "trigger_name", "str32", *recipe.SetName); err != nil {
			return err
		}
	}
	if recipe.DescriptionStringID != nil {
		if err := setIntField(trigger, "description_string_table_id", "s32", *recipe.DescriptionStringID); err != nil {
			return err
		}
	}
	if recipe.ShortDescriptionStringID != nil {
		if err := setIntField(trigger, "short_description_string_table_id", "s32", *recipe.ShortDescriptionStringID); err != nil {
			return err
		}
	}
	if err := editTriggerChildLists(spec, trigger, recipe); err != nil {
		return err
	}
	if len(recipe.Effects) > 0 {
		if err := appendEffectsToTrigger(spec, trigger, recipe.Effects); err != nil {
			return err
		}
	}
	if len(recipe.Conditions) > 0 {
		if err := appendConditionsToTrigger(spec, trigger, recipe.Conditions); err != nil {
			return err
		}
	}
	f.body = f.root.raw()
	f.InflatedBytes = len(f.body)
	if triggers, err := f.root.triggerInfo(); err == nil {
		f.Triggers = triggers
	} else {
		return err
	}
	return nil
}

func (f *File) TombstoneTrigger(recipe TriggerRecipe) error {
	target, err := f.findTrigger(recipe)
	if err != nil {
		return err
	}
	index, err := f.findTriggerIndex(recipe)
	if err != nil {
		return err
	}
	disabled := false
	notLooping := false
	clear := true
	name := recipe.SetName
	if name == nil {
		currentName, _ := target.stringValue("trigger_name")
		tombstoneName := fmt.Sprintf("TOMBSTONED trigger %d", index)
		if strings.TrimSpace(currentName) != "" {
			tombstoneName = "TOMBSTONED: " + currentName
		}
		name = &tombstoneName
	}
	edit := TriggerRecipe{
		Op:              "edit_trigger",
		TargetIndex:     &index,
		Enabled:         &disabled,
		Looping:         &notLooping,
		SetName:         name,
		ClearEffects:    &clear,
		ClearConditions: &clear,
	}
	return f.EditTrigger(edit)
}

func (f *File) AddCreateObjectTrigger(recipe TriggerRecipe) error {
	if recipe.ObjectListUnitID == nil {
		return fmt.Errorf("add_create_object requires object_list_unit_id")
	}
	if recipe.LocationX == nil {
		return fmt.Errorf("add_create_object requires location_x")
	}
	if recipe.LocationY == nil {
		return fmt.Errorf("add_create_object requires location_y")
	}
	spec, err := f.writeSpec()
	if err != nil {
		return err
	}
	raw, err := buildCreateObjectTrigger(spec, recipe)
	if err != nil {
		return err
	}
	return f.addRawTrigger(raw)
}

func (f *File) AddDisplayInstructionsTrigger(recipe TriggerRecipe) error {
	spec, err := f.writeSpec()
	if err != nil {
		return err
	}
	raw, err := buildDisplayInstructionTrigger(spec, recipe)
	if err != nil {
		return err
	}
	return f.addRawTrigger(raw)
}

func (f *File) addRawTrigger(raw []byte) error {
	triggers := f.root.section("Triggers")
	if triggers == nil {
		return fmt.Errorf("missing Triggers section")
	}
	return f.insertRawTrigger(len(triggers.list("trigger_data")), raw)
}

func (f *File) insertRawTrigger(index int, raw []byte) error {
	triggers := f.root.section("Triggers")
	if triggers == nil {
		return fmt.Errorf("missing Triggers section")
	}
	spec, err := f.writeSpec()
	if err != nil {
		return err
	}
	triggerSpec, _, err := triggerEffectSpecs(spec)
	if err != nil {
		return err
	}
	parser := parser{data: raw, sections: map[string]*parsedSection{}}
	triggerNode, err := parser.parseNode("TriggerStruct", triggerSpec, nil)
	if err != nil {
		return err
	}
	if parser.off != len(raw) {
		return fmt.Errorf("new trigger parse stopped at %d of %d", parser.off, len(raw))
	}
	count, _ := triggers.intValue("number_of_triggers")
	triggerData := triggers.field("trigger_data")
	if triggerData == nil {
		return fmt.Errorf("missing trigger_data")
	}
	if index < 0 || index > len(triggerData.Elements) {
		return fmt.Errorf("trigger insert index %d out of range 0..%d", index, len(triggerData.Elements))
	}
	if index < len(triggerData.Elements) {
		if err := rewriteTriggerReferencesAfterInsert(triggerData.Elements, index); err != nil {
			return err
		}
	}
	triggerData.Elements = append(triggerData.Elements, nil)
	copy(triggerData.Elements[index+1:], triggerData.Elements[index:])
	triggerNode.Start = triggerData.Start
	triggerNode.End = triggerData.Start + len(raw)
	triggerData.Elements[index] = triggerNode
	count = len(triggerData.Elements)
	if err := setIntField(triggers, "number_of_triggers", "s32", count); err != nil {
		return err
	}
	if err := setUint32ListField(triggers, "trigger_display_order_array", indexUint32List(count)); err != nil {
		return err
	}
	if options := f.root.section("Options"); options != nil {
		if err := setIntField(options, "number_of_triggers", "u32", count); err != nil {
			return err
		}
	}
	if f.headerRoot != nil {
		if err := setIntField(f.headerRoot, "trigger_count", "u32", count); err != nil {
			return err
		}
		f.header = f.headerRoot.raw()
	}
	f.body = f.root.raw()
	f.InflatedBytes = len(f.body)
	if triggers, err := f.root.triggerInfo(); err == nil {
		f.Triggers = triggers
	} else {
		return err
	}
	return nil
}

func (f *File) RemoveTrigger(recipe TriggerRecipe) error {
	triggers := f.root.section("Triggers")
	if triggers == nil {
		return fmt.Errorf("missing Triggers section")
	}
	triggerData := triggers.field("trigger_data")
	if triggerData == nil {
		return fmt.Errorf("missing trigger_data")
	}
	index, err := f.findTriggerIndex(recipe)
	if err != nil {
		return err
	}
	if index < 0 || index >= len(triggerData.Elements) {
		return fmt.Errorf("trigger index %d out of range 0..%d", index, len(triggerData.Elements)-1)
	}
	if err := rewriteTriggerReferencesAfterRemove(triggerData.Elements, index); err != nil {
		return err
	}
	triggerData.Elements = append(triggerData.Elements[:index], triggerData.Elements[index+1:]...)
	count := len(triggerData.Elements)
	if err := setIntField(triggers, "number_of_triggers", "s32", count); err != nil {
		return err
	}
	if err := setUint32ListField(triggers, "trigger_display_order_array", indexUint32List(count)); err != nil {
		return err
	}
	if options := f.root.section("Options"); options != nil {
		if err := setIntField(options, "number_of_triggers", "u32", count); err != nil {
			return err
		}
	}
	if f.headerRoot != nil {
		if err := setIntField(f.headerRoot, "trigger_count", "u32", count); err != nil {
			return err
		}
		f.header = f.headerRoot.raw()
	}
	f.body = f.root.raw()
	f.InflatedBytes = len(f.body)
	if triggers, err := f.root.triggerInfo(); err == nil {
		f.Triggers = triggers
	} else {
		return err
	}
	return nil
}

func (f *File) RemoveTriggers(recipe TriggerRecipe) error {
	triggers := f.root.section("Triggers")
	if triggers == nil {
		return fmt.Errorf("missing Triggers section")
	}
	triggerData := triggers.field("trigger_data")
	if triggerData == nil {
		return fmt.Errorf("missing trigger_data")
	}
	indexes, err := normalizeTriggerIndexes(recipe.TargetIndexes, len(triggerData.Elements))
	if err != nil {
		return err
	}
	if err := rewriteTriggerReferencesAfterBatchRemove(triggerData.Elements, indexes); err != nil {
		return err
	}
	for _, index := range descendingInts(indexes) {
		triggerData.Elements = append(triggerData.Elements[:index], triggerData.Elements[index+1:]...)
	}
	count := len(triggerData.Elements)
	if err := setIntField(triggers, "number_of_triggers", "s32", count); err != nil {
		return err
	}
	if err := setUint32ListField(triggers, "trigger_display_order_array", indexUint32List(count)); err != nil {
		return err
	}
	if options := f.root.section("Options"); options != nil {
		if err := setIntField(options, "number_of_triggers", "u32", count); err != nil {
			return err
		}
	}
	if f.headerRoot != nil {
		if err := setIntField(f.headerRoot, "trigger_count", "u32", count); err != nil {
			return err
		}
		f.header = f.headerRoot.raw()
	}
	f.body = f.root.raw()
	f.InflatedBytes = len(f.body)
	if triggers, err := f.root.triggerInfo(); err == nil {
		f.Triggers = triggers
	} else {
		return err
	}
	return nil
}

func (f *File) validateRemoveSystemPrefix(prefix string) (systemPrefixMatches, ReferenceReport, error) {
	matches, err := f.matchSystemPrefix(prefix)
	if err != nil {
		return systemPrefixMatches{}, ReferenceReport{}, err
	}
	externalRefs, _, err := f.systemPrefixReferences(matches)
	if err != nil {
		return systemPrefixMatches{}, ReferenceReport{}, err
	}
	return matches, externalRefs, nil
}

func (f *File) RemoveSystemPrefix(prefix string) error {
	matches, externalRefs, err := f.validateRemoveSystemPrefix(prefix)
	if err != nil {
		return err
	}
	if externalRefs.Summary.Total > 0 {
		return fmt.Errorf("refuses to remove system prefix %q because %d external reference(s) point into the matched bundle", prefix, externalRefs.Summary.Total)
	}
	if len(matches.TriggerIndexes) > 0 {
		if err := f.RemoveTriggers(TriggerRecipe{Op: "remove_triggers", TargetIndexes: matches.TriggerIndexes}); err != nil {
			return err
		}
	}
	if len(matches.UnitReferenceIDs) > 0 {
		if err := f.removeUnitsByReferenceIDs(matches.UnitReferenceIDs); err != nil {
			return err
		}
	}
	for _, id := range matches.VariableIDs {
		targetID := id
		if err := f.RemoveVariable(VariableRecipe{Op: "remove_variable", TargetID: &targetID}); err != nil {
			return err
		}
	}
	for _, id := range matches.StringIDs {
		targetID := id
		if err := f.ClearString(StringRecipe{Op: "clear_string", ID: &targetID}); err != nil {
			return err
		}
	}
	return nil
}

func (f *File) removeUnitsByReferenceIDs(referenceIDs []int) error {
	targetSet := intSet(referenceIDs)
	if len(targetSet) == 0 {
		return nil
	}
	playerSections, err := f.unitPlayerSections()
	if err != nil {
		return err
	}
	byPlayer := map[int][]unitAreaMatch{}
	found := map[int]bool{}
	for player, playerSection := range playerSections {
		unitField := playerSection.field("units")
		if unitField == nil {
			continue
		}
		for index, unit := range unitField.Elements {
			referenceID, ok := unit.intValue("reference_id")
			if !ok || !targetSet[referenceID] {
				continue
			}
			found[referenceID] = true
			byPlayer[player] = append(byPlayer[player], unitAreaMatch{Unit: unit, PlayerSection: playerSection, Player: player, Index: index})
		}
	}
	for _, id := range referenceIDs {
		if !found[id] {
			return fmt.Errorf("system-prefix unit reference_id %d was not found at mutation time", id)
		}
	}
	for _, matches := range byPlayer {
		sort.Slice(matches, func(i, j int) bool {
			return matches[i].Index > matches[j].Index
		})
		playerSection := matches[0].PlayerSection
		unitField := playerSection.field("units")
		if unitField == nil {
			return fmt.Errorf("missing units field for player %d", matches[0].Player)
		}
		for _, target := range matches {
			if target.Index < 0 || target.Index >= len(unitField.Elements) || unitField.Elements[target.Index] != target.Unit {
				return fmt.Errorf("unit index changed before system-prefix removal for player %d index %d", target.Player, target.Index)
			}
			unitField.Elements = append(unitField.Elements[:target.Index], unitField.Elements[target.Index+1:]...)
		}
		if err := setIntField(playerSection, "unit_count", "u32", len(unitField.Elements)); err != nil {
			return err
		}
	}
	return f.refreshUnits()
}

func (f *File) validateRemoveTriggers(targetIndexes []int) (int, error) {
	triggers := f.root.section("Triggers")
	if triggers == nil {
		return 0, fmt.Errorf("missing Triggers section")
	}
	triggerData := triggers.field("trigger_data")
	if triggerData == nil {
		return 0, fmt.Errorf("missing trigger_data")
	}
	indexes, err := normalizeTriggerIndexes(targetIndexes, len(triggerData.Elements))
	if err != nil {
		return 0, err
	}
	if err := ensureNoExternalTriggerReferencesToRemoved(triggerData.Elements, indexes); err != nil {
		return 0, err
	}
	return len(indexes), nil
}

func normalizeTriggerIndexes(indexes []int, triggerCount int) ([]int, error) {
	if len(indexes) == 0 {
		return nil, fmt.Errorf("remove_triggers requires target_indexes")
	}
	out := append([]int(nil), indexes...)
	sort.Ints(out)
	write := 0
	for _, index := range out {
		if index < 0 || index >= triggerCount {
			return nil, fmt.Errorf("trigger index %d out of range 0..%d", index, triggerCount-1)
		}
		if write > 0 && out[write-1] == index {
			continue
		}
		out[write] = index
		write++
	}
	return out[:write], nil
}

func rewriteTriggerReferencesAfterRemove(triggerNodes []*parsedNode, removedIndex int) error {
	for triggerIndex, trigger := range triggerNodes {
		if triggerIndex == removedIndex {
			continue
		}
		for effectIndex, effect := range trigger.list("effect_data") {
			ref, ok := effect.intValue("trigger_id")
			if !ok || ref < 0 {
				continue
			}
			if ref == removedIndex {
				name, _ := trigger.stringValue("trigger_name")
				return fmt.Errorf("refuses to remove trigger %d because trigger %d %q effect %d references it", removedIndex, triggerIndex, name, effectIndex)
			}
		}
	}
	for triggerIndex, trigger := range triggerNodes {
		if triggerIndex == removedIndex {
			continue
		}
		for effectIndex, effect := range trigger.list("effect_data") {
			ref, ok := effect.intValue("trigger_id")
			if !ok || ref <= removedIndex {
				continue
			}
			if err := setIntField(effect, "trigger_id", "s32", ref-1); err != nil {
				return fmt.Errorf("rewrite trigger %d effect %d trigger_id: %w", triggerIndex, effectIndex, err)
			}
		}
	}
	return nil
}

func rewriteTriggerReferencesAfterInsert(triggerNodes []*parsedNode, insertedIndex int) error {
	for triggerIndex, trigger := range triggerNodes {
		for effectIndex, effect := range trigger.list("effect_data") {
			ref, ok := effect.intValue("trigger_id")
			if !ok || ref < insertedIndex {
				continue
			}
			if err := setIntField(effect, "trigger_id", "s32", ref+1); err != nil {
				return fmt.Errorf("rewrite trigger %d effect %d trigger_id after insert: %w", triggerIndex, effectIndex, err)
			}
		}
		for conditionIndex, condition := range trigger.list("condition_data") {
			ref, ok := condition.intValue("trigger_id")
			if !ok || ref < insertedIndex {
				continue
			}
			if err := setIntField(condition, "trigger_id", "s32", ref+1); err != nil {
				return fmt.Errorf("rewrite trigger %d condition %d trigger_id after insert: %w", triggerIndex, conditionIndex, err)
			}
		}
	}
	return nil
}

func rewriteTriggerReferencesAfterBatchRemove(triggerNodes []*parsedNode, removedIndexes []int) error {
	indexes, err := normalizeTriggerIndexes(removedIndexes, len(triggerNodes))
	if err != nil {
		return err
	}
	if err := ensureNoExternalTriggerReferencesToRemoved(triggerNodes, indexes); err != nil {
		return err
	}
	removed := intSet(indexes)
	for triggerIndex, trigger := range triggerNodes {
		if removed[triggerIndex] {
			continue
		}
		for effectIndex, effect := range trigger.list("effect_data") {
			ref, ok := effect.intValue("trigger_id")
			if !ok || ref < 0 {
				continue
			}
			shift := countIntsLessThan(indexes, ref)
			if shift == 0 {
				continue
			}
			if err := setIntField(effect, "trigger_id", "s32", ref-shift); err != nil {
				return fmt.Errorf("rewrite trigger %d effect %d trigger_id: %w", triggerIndex, effectIndex, err)
			}
		}
	}
	return nil
}

func triggerHasXSCarrier(trigger *parsedNode) bool {
	if trigger == nil {
		return false
	}
	for _, effect := range trigger.list("effect_data") {
		effectType, _ := effect.intValue("effect_type")
		if effectType != effectTypeID("script_call") {
			continue
		}
		message, _ := effect.stringValue("message")
		if isXSCarrierMessage(message) {
			return true
		}
	}
	return false
}

func ensureNoExternalTriggerReferencesToRemoved(triggerNodes []*parsedNode, removedIndexes []int) error {
	removed := intSet(removedIndexes)
	for triggerIndex, trigger := range triggerNodes {
		if removed[triggerIndex] {
			continue
		}
		for effectIndex, effect := range trigger.list("effect_data") {
			ref, ok := effect.intValue("trigger_id")
			if !ok || ref < 0 || !removed[ref] {
				continue
			}
			name, _ := trigger.stringValue("trigger_name")
			return fmt.Errorf("refuses to remove trigger batch because trigger %d %q effect %d references removed trigger %d", triggerIndex, name, effectIndex, ref)
		}
	}
	return nil
}

func intSet(values []int) map[int]bool {
	out := make(map[int]bool, len(values))
	for _, value := range values {
		out[value] = true
	}
	return out
}

func ptrInt(value int) *int {
	return &value
}

func ptrFloat64(value float64) *float64 {
	return &value
}

func countIntsLessThan(values []int, target int) int {
	count := 0
	for _, value := range values {
		if value >= target {
			break
		}
		count++
	}
	return count
}

func (f *File) ensureUnitNotReferenced(referenceID int) error {
	units := f.root.section("Units")
	if units != nil {
		for player, playerSection := range units.list("players_units") {
			for unitIndex, unit := range playerSection.list("units") {
				ref, ok := unit.intValue("garrisoned_in_id")
				if ok && ref == referenceID {
					sourceRef, _ := unit.intValue("reference_id")
					return fmt.Errorf("refuses to remove unit reference_id %d because player %d unit %d reference_id %d field garrisoned_in_id references it", referenceID, player, unitIndex, sourceRef)
				}
			}
		}
	}
	triggers := f.root.section("Triggers")
	if triggers == nil {
		return nil
	}
	triggerData := triggers.field("trigger_data")
	if triggerData == nil {
		return nil
	}
	for triggerIndex, trigger := range triggerData.Elements {
		triggerName, _ := trigger.stringValue("trigger_name")
		for conditionIndex, condition := range trigger.list("condition_data") {
			for _, field := range []string{"unit_object", "next_object"} {
				ref, ok := condition.intValue(field)
				if ok && ref == referenceID {
					return fmt.Errorf("refuses to remove unit reference_id %d because trigger %d %q condition %d field %s references it", referenceID, triggerIndex, triggerName, conditionIndex, field)
				}
			}
		}
		for effectIndex, effect := range trigger.list("effect_data") {
			for _, field := range []string{"legacy_location_object_reference", "location_object_reference"} {
				ref, ok := effect.intValue(field)
				if ok && ref == referenceID {
					return fmt.Errorf("refuses to remove unit reference_id %d because trigger %d %q effect %d field %s references it", referenceID, triggerIndex, triggerName, effectIndex, field)
				}
			}
			for selectedIndex, ref := range effect.intList("selected_object_ids") {
				if ref == referenceID {
					return fmt.Errorf("refuses to remove unit reference_id %d because trigger %d %q effect %d selected_object_ids[%d] references it", referenceID, triggerIndex, triggerName, effectIndex, selectedIndex)
				}
			}
		}
	}
	return nil
}

func (f *File) ClearTriggers() error {
	triggers := f.root.section("Triggers")
	if triggers == nil {
		return fmt.Errorf("missing Triggers section")
	}
	triggerData := triggers.field("trigger_data")
	if triggerData == nil {
		return fmt.Errorf("missing trigger_data")
	}
	triggerData.Elements = nil
	if err := setIntField(triggers, "number_of_triggers", "s32", 0); err != nil {
		return err
	}
	if err := setUint32ListField(triggers, "trigger_display_order_array", nil); err != nil {
		return err
	}
	if options := f.root.section("Options"); options != nil {
		if err := setIntField(options, "number_of_triggers", "u32", 0); err != nil {
			return err
		}
	}
	if f.headerRoot != nil {
		if err := setIntField(f.headerRoot, "trigger_count", "u32", 0); err != nil {
			return err
		}
		f.header = f.headerRoot.raw()
	}
	f.body = f.root.raw()
	f.InflatedBytes = len(f.body)
	if triggers, err := f.root.triggerInfo(); err == nil {
		f.Triggers = triggers
	} else {
		return err
	}
	return nil
}

func (f *File) addRawUnit(raw []byte, player int) error {
	unitsSection := f.root.section("Units")
	if unitsSection == nil {
		return fmt.Errorf("missing Units section")
	}
	playerSections := unitsSection.list("players_units")
	if player < 0 || player >= len(playerSections) {
		return fmt.Errorf("unit player %d out of range 0..%d", player, len(playerSections)-1)
	}
	spec, err := f.writeSpec()
	if err != nil {
		return err
	}
	unitSpec, err := unitStructSpec(spec)
	if err != nil {
		return err
	}
	parser := parser{data: raw, sections: map[string]*parsedSection{}}
	unitNode, err := parser.parseNode("UnitStruct", unitSpec, nil)
	if err != nil {
		return err
	}
	if parser.off != len(raw) {
		return fmt.Errorf("new unit parse stopped at %d of %d", parser.off, len(raw))
	}
	playerSection := playerSections[player]
	unitField := playerSection.field("units")
	if unitField == nil {
		return fmt.Errorf("missing units field for player %d", player)
	}
	unitField.Elements = append(unitField.Elements, unitNode)
	if err := setIntField(playerSection, "unit_count", "u32", len(unitField.Elements)); err != nil {
		return err
	}
	f.body = f.root.raw()
	f.InflatedBytes = len(f.body)
	if units, err := f.root.unitInfo(); err == nil {
		f.Units = units
	} else {
		return err
	}
	return nil
}

func (f *File) findUnitByReferenceID(referenceID int) (*parsedNode, *parsedNode, int, error) {
	unitsSection := f.root.section("Units")
	if unitsSection == nil {
		return nil, nil, 0, fmt.Errorf("missing Units section")
	}
	playerSections := unitsSection.list("players_units")
	for _, playerSection := range playerSections {
		unitField := playerSection.field("units")
		if unitField == nil {
			continue
		}
		for i, unit := range unitField.Elements {
			refID, _ := unit.intValue("reference_id")
			if refID == referenceID {
				return unit, playerSection, i, nil
			}
		}
	}
	return nil, nil, 0, fmt.Errorf("unit reference_id %d not found", referenceID)
}

func (f *File) findUnit(recipe UnitRecipe) (*parsedNode, *parsedNode, int, error) {
	if recipe.ReferenceID != nil {
		return f.findUnitByReferenceID(*recipe.ReferenceID)
	}
	if recipe.TargetIndex != nil {
		if recipe.TargetPlayer == nil {
			return nil, nil, 0, fmt.Errorf("%s target_index requires target_player", recipe.Op)
		}
		return f.findUnitByPlayerIndex(*recipe.TargetPlayer, *recipe.TargetIndex)
	}
	if recipe.TargetCaption != "" {
		return f.findUnitByCaption(recipe.TargetCaption, recipe.TargetPlayer)
	}
	return nil, nil, 0, fmt.Errorf("%s requires reference_id, target_index+target_player, or unique target_caption", recipe.Op)
}

func (f *File) findUnitsInArea(recipe UnitRecipe) ([]unitAreaMatch, error) {
	if recipe.TargetAreaX1 == nil || recipe.TargetAreaY1 == nil || recipe.TargetAreaX2 == nil || recipe.TargetAreaY2 == nil {
		return nil, fmt.Errorf("%s requires target_area_x1, target_area_y1, target_area_x2, and target_area_y2", unitAreaOpName(recipe))
	}
	x1, x2 := *recipe.TargetAreaX1, *recipe.TargetAreaX2
	y1, y2 := *recipe.TargetAreaY1, *recipe.TargetAreaY2
	if x1 > x2 {
		x1, x2 = x2, x1
	}
	if y1 > y2 {
		y1, y2 = y2, y1
	}
	playerSections, err := f.unitPlayerSections()
	if err != nil {
		return nil, err
	}
	var matches []unitAreaMatch
	for player, playerSection := range playerSections {
		if recipe.TargetPlayer != nil && player != *recipe.TargetPlayer {
			continue
		}
		unitField := playerSection.field("units")
		if unitField == nil {
			continue
		}
		for index, unit := range unitField.Elements {
			if recipe.TargetUnitConst != nil {
				unitConst, ok := unit.intValue("unit_const")
				if !ok || unitConst != *recipe.TargetUnitConst {
					continue
				}
			}
			x, okX := unit.floatValue("x")
			y, okY := unit.floatValue("y")
			if !okX || !okY {
				continue
			}
			if x < x1 || x > x2 || y < y1 || y > y2 {
				continue
			}
			matches = append(matches, unitAreaMatch{Unit: unit, PlayerSection: playerSection, Player: player, Index: index})
		}
	}
	if len(matches) == 0 {
		return nil, fmt.Errorf("%s matched no units", unitAreaOpName(recipe))
	}
	return matches, nil
}

func unitAreaOpName(recipe UnitRecipe) string {
	if recipe.Op != "" {
		return recipe.Op
	}
	return "unit area operation"
}

func (f *File) validateMoveUnitsInArea(targets []unitAreaMatch, recipe UnitRecipe) error {
	if recipe.OffsetX == nil && recipe.OffsetY == nil && recipe.TargetX == nil && recipe.TargetY == nil {
		return fmt.Errorf("move_units_in_area requires offset_x/offset_y or target_x/target_y")
	}
	offsetX, offsetY, err := unitAreaOffset(recipe)
	if err != nil {
		return err
	}
	_, width, height, err := f.mapTiles()
	if err != nil {
		return err
	}
	for _, target := range targets {
		x, okX := target.Unit.floatValue("x")
		y, okY := target.Unit.floatValue("y")
		if !okX || !okY {
			return fmt.Errorf("move_units_in_area source player %d unit %d has no x/y", target.Player, target.Index)
		}
		x += offsetX
		y += offsetY
		if x < 0 || y < 0 || x > float64(width) || y > float64(height) {
			return fmt.Errorf("move_units_in_area would move player %d unit %d to %.2f,%.2f outside map %dx%d", target.Player, target.Index, x, y, width, height)
		}
	}
	return nil
}

func (f *File) validateEditUnitsInArea(recipe UnitRecipe) error {
	if recipe.X != nil || recipe.Y != nil || recipe.Z != nil || recipe.OffsetX != nil || recipe.OffsetY != nil || recipe.TargetX != nil || recipe.TargetY != nil {
		return fmt.Errorf("edit_units_in_area does not edit positions; use edit_unit or move_units_in_area")
	}
	if recipe.UnitConst <= 0 &&
		recipe.Status == nil &&
		recipe.Rotation == nil &&
		recipe.InitialAnimationFrame == nil &&
		recipe.GarrisonedInID == nil &&
		recipe.CaptionStringID == nil &&
		recipe.CaptionString == "" &&
		recipe.SetPlayer == nil {
		return fmt.Errorf("edit_units_in_area requires unit_const, status, rotation, initial_animation_frame, garrisoned_in_id, caption_string_id, caption_string, or set_player")
	}
	if recipe.SetPlayer != nil {
		playerSections, err := f.unitPlayerSections()
		if err != nil {
			return err
		}
		if *recipe.SetPlayer < 0 || *recipe.SetPlayer >= len(playerSections) {
			return fmt.Errorf("set_player %d out of range 0..%d", *recipe.SetPlayer, len(playerSections)-1)
		}
	}
	return nil
}

func unitAreaOffset(recipe UnitRecipe) (float64, float64, error) {
	hasOffset := recipe.OffsetX != nil || recipe.OffsetY != nil
	hasTarget := recipe.TargetX != nil || recipe.TargetY != nil
	if hasOffset && hasTarget {
		return 0, 0, fmt.Errorf("%s cannot combine offset_x/offset_y with target_x/target_y", unitAreaOpName(recipe))
	}
	if hasTarget {
		if recipe.TargetX == nil || recipe.TargetY == nil {
			return 0, 0, fmt.Errorf("%s target placement requires both target_x and target_y", unitAreaOpName(recipe))
		}
		if recipe.TargetAreaX1 == nil || recipe.TargetAreaY1 == nil || recipe.TargetAreaX2 == nil || recipe.TargetAreaY2 == nil {
			return 0, 0, fmt.Errorf("%s target placement requires source target_area bounds", unitAreaOpName(recipe))
		}
		x1 := *recipe.TargetAreaX1
		y1 := *recipe.TargetAreaY1
		if *recipe.TargetAreaX2 < x1 {
			x1 = *recipe.TargetAreaX2
		}
		if *recipe.TargetAreaY2 < y1 {
			y1 = *recipe.TargetAreaY2
		}
		return *recipe.TargetX - x1, *recipe.TargetY - y1, nil
	}
	offsetX := 0.0
	offsetY := 0.0
	if recipe.OffsetX != nil {
		offsetX = *recipe.OffsetX
	}
	if recipe.OffsetY != nil {
		offsetY = *recipe.OffsetY
	}
	return offsetX, offsetY, nil
}

func applyUnitAreaFieldEdits(unitNode *parsedNode, recipe UnitRecipe) error {
	if recipe.UnitConst > 0 {
		if err := setIntField(unitNode, "unit_const", "u16", recipe.UnitConst); err != nil {
			return err
		}
	}
	if recipe.Status != nil {
		if err := setIntField(unitNode, "status", "u8", *recipe.Status); err != nil {
			return err
		}
	}
	if recipe.Rotation != nil {
		if err := setFloatField(unitNode, "rotation", "f32", *recipe.Rotation); err != nil {
			return err
		}
	}
	if recipe.InitialAnimationFrame != nil {
		if err := setIntField(unitNode, "initial_animation_frame", "u16", *recipe.InitialAnimationFrame); err != nil {
			return err
		}
	}
	if recipe.GarrisonedInID != nil {
		if err := setIntField(unitNode, "garrisoned_in_id", "s32", *recipe.GarrisonedInID); err != nil {
			return err
		}
	}
	if recipe.CaptionStringID != nil {
		if err := setIntField(unitNode, "caption_string_id", "s32", *recipe.CaptionStringID); err != nil {
			return err
		}
	}
	if recipe.CaptionString != "" {
		if err := setStringField(unitNode, "caption_string", "str32", recipe.CaptionString); err != nil {
			return err
		}
	}
	return nil
}

func (f *File) validateCopiedUnitReferenceIDs(targets []unitAreaMatch, recipe UnitRecipe) error {
	if _, _, err := unitAreaOffset(recipe); err != nil {
		return err
	}
	base := copiedUnitReferenceBase(f, recipe)
	if base <= 0 {
		return fmt.Errorf("copy_units_in_area reference_id_base %d must be positive", base)
	}
	seen := map[int]bool{}
	for i := range targets {
		id := base + i
		if seen[id] {
			return fmt.Errorf("copy_units_in_area duplicate generated reference_id %d", id)
		}
		seen[id] = true
		if _, _, _, err := f.findUnitByReferenceID(id); err == nil {
			return fmt.Errorf("copy_units_in_area generated reference_id %d already exists", id)
		}
	}
	if recipe.SetPlayer != nil {
		playerSections, err := f.unitPlayerSections()
		if err != nil {
			return err
		}
		if *recipe.SetPlayer < 0 || *recipe.SetPlayer >= len(playerSections) {
			return fmt.Errorf("set_player %d out of range 0..%d", *recipe.SetPlayer, len(playerSections)-1)
		}
	}
	return nil
}

func copiedUnitReferenceBase(f *File, recipe UnitRecipe) int {
	if recipe.ReferenceIDBase != nil {
		return *recipe.ReferenceIDBase
	}
	return f.nextUnitReferenceID()
}

func copiedUnitReferenceMap(targets []unitAreaMatch, base int) map[int]int {
	refMap := map[int]int{}
	for i, target := range targets {
		if oldRef, ok := target.Unit.intValue("reference_id"); ok && oldRef >= 0 {
			refMap[oldRef] = base + i
		}
	}
	return refMap
}

func copiedUnitRecipe(target unitAreaMatch, recipe UnitRecipe, refMap map[int]int) (UnitRecipe, error) {
	unitConst, ok := target.Unit.intValue("unit_const")
	if !ok || unitConst <= 0 {
		return UnitRecipe{}, fmt.Errorf("copy_units_in_area source player %d unit %d has invalid unit_const", target.Player, target.Index)
	}
	x, okX := target.Unit.floatValue("x")
	y, okY := target.Unit.floatValue("y")
	z, _ := target.Unit.floatValue("z")
	if !okX || !okY {
		return UnitRecipe{}, fmt.Errorf("copy_units_in_area source player %d unit %d has no x/y", target.Player, target.Index)
	}
	offsetX, offsetY, err := unitAreaOffset(recipe)
	if err != nil {
		return UnitRecipe{}, err
	}
	x += offsetX
	y += offsetY
	player := target.Player
	if recipe.SetPlayer != nil {
		player = *recipe.SetPlayer
	}
	oldRef, ok := target.Unit.intValue("reference_id")
	if !ok || oldRef < 0 {
		return UnitRecipe{}, fmt.Errorf("copy_units_in_area source player %d unit %d has no non-negative reference_id", target.Player, target.Index)
	}
	newRef := refMap[oldRef]
	status, _ := target.Unit.intValue("status")
	rotation, _ := target.Unit.floatValue("rotation")
	initialFrame, _ := target.Unit.intValue("initial_animation_frame")
	garrisonedInID, ok := target.Unit.intValue("garrisoned_in_id")
	if !ok {
		garrisonedInID = -1
	}
	if mapped, ok := refMap[garrisonedInID]; ok {
		garrisonedInID = mapped
	} else if garrisonedInID >= 0 {
		garrisonedInID = -1
	}
	captionStringID, ok := target.Unit.intValue("caption_string_id")
	if !ok {
		captionStringID = -1
	}
	captionString, _ := target.Unit.stringValue("caption_string")
	if recipe.CaptionSuffix != "" {
		captionString += recipe.CaptionSuffix
	}
	return UnitRecipe{
		Op:                    "add_unit",
		Player:                player,
		UnitConst:             unitConst,
		X:                     &x,
		Y:                     &y,
		Z:                     &z,
		ReferenceID:           &newRef,
		Status:                &status,
		Rotation:              &rotation,
		InitialAnimationFrame: &initialFrame,
		GarrisonedInID:        &garrisonedInID,
		CaptionStringID:       &captionStringID,
		CaptionString:         captionString,
	}, nil
}

func (f *File) findUnitByPlayerIndex(player, index int) (*parsedNode, *parsedNode, int, error) {
	playerSections, err := f.unitPlayerSections()
	if err != nil {
		return nil, nil, 0, err
	}
	if player < 0 || player >= len(playerSections) {
		return nil, nil, 0, fmt.Errorf("unit target_player %d out of range 0..%d", player, len(playerSections)-1)
	}
	playerSection := playerSections[player]
	unitField := playerSection.field("units")
	if unitField == nil {
		return nil, nil, 0, fmt.Errorf("missing units field for player %d", player)
	}
	if index < 0 || index >= len(unitField.Elements) {
		return nil, nil, 0, fmt.Errorf("unit target_index %d out of range for player %d units 0..%d", index, player, len(unitField.Elements)-1)
	}
	return unitField.Elements[index], playerSection, index, nil
}

func (f *File) findUnitByCaption(caption string, targetPlayer *int) (*parsedNode, *parsedNode, int, error) {
	playerSections, err := f.unitPlayerSections()
	if err != nil {
		return nil, nil, 0, err
	}
	var foundUnit *parsedNode
	var foundSection *parsedNode
	foundIndex := -1
	matches := 0
	for player, playerSection := range playerSections {
		if targetPlayer != nil && player != *targetPlayer {
			continue
		}
		unitField := playerSection.field("units")
		if unitField == nil {
			continue
		}
		for i, unit := range unitField.Elements {
			got, _ := unit.stringValue("caption_string")
			if got != caption {
				continue
			}
			matches++
			foundUnit = unit
			foundSection = playerSection
			foundIndex = i
		}
	}
	if matches == 0 {
		if targetPlayer != nil {
			return nil, nil, 0, fmt.Errorf("unit target_caption %q not found for player %d", caption, *targetPlayer)
		}
		return nil, nil, 0, fmt.Errorf("unit target_caption %q not found", caption)
	}
	if matches > 1 {
		return nil, nil, 0, fmt.Errorf("unit target_caption %q matched %d units; use reference_id or target_player+target_index", caption, matches)
	}
	return foundUnit, foundSection, foundIndex, nil
}

func (f *File) findUnitsByCaptionPrefix(prefix string, targetPlayer *int) ([]unitAreaMatch, error) {
	return f.findUnitsByCaptionText(prefix, targetPlayer, true)
}

func (f *File) findUnitsByCaptionText(text string, targetPlayer *int, prefixOnly bool) ([]unitAreaMatch, error) {
	if strings.TrimSpace(text) == "" {
		if prefixOnly {
			return nil, fmt.Errorf("unit caption prefix must be non-empty")
		}
		return nil, fmt.Errorf("unit caption contains marker must be non-empty")
	}
	playerSections, err := f.unitPlayerSections()
	if err != nil {
		return nil, err
	}
	var matches []unitAreaMatch
	for player, playerSection := range playerSections {
		if targetPlayer != nil && player != *targetPlayer {
			continue
		}
		unitField := playerSection.field("units")
		if unitField == nil {
			continue
		}
		for index, unit := range unitField.Elements {
			caption, _ := unit.stringValue("caption_string")
			matched := strings.Contains(caption, text)
			if prefixOnly {
				matched = strings.HasPrefix(caption, text)
			}
			if matched {
				matches = append(matches, unitAreaMatch{Unit: unit, PlayerSection: playerSection, Player: player, Index: index})
			}
		}
	}
	if len(matches) == 0 {
		if targetPlayer != nil {
			if prefixOnly {
				return nil, fmt.Errorf("unit caption prefix %q matched no units for player %d", text, *targetPlayer)
			}
			return nil, fmt.Errorf("unit caption contains marker %q matched no units for player %d", text, *targetPlayer)
		}
		if prefixOnly {
			return nil, fmt.Errorf("unit caption prefix %q matched no units", text)
		}
		return nil, fmt.Errorf("unit caption contains marker %q matched no units", text)
	}
	return matches, nil
}

func (f *File) findUnitsByUnitConst(unitConst int, targetPlayer *int) ([]unitAreaMatch, error) {
	if unitConst < 0 {
		return nil, fmt.Errorf("unit const must be non-negative")
	}
	playerSections, err := f.unitPlayerSections()
	if err != nil {
		return nil, err
	}
	var matches []unitAreaMatch
	for player, playerSection := range playerSections {
		if targetPlayer != nil && player != *targetPlayer {
			continue
		}
		unitField := playerSection.field("units")
		if unitField == nil {
			continue
		}
		for index, unit := range unitField.Elements {
			got, ok := unit.intValue("unit_const")
			if !ok || got != unitConst {
				continue
			}
			matches = append(matches, unitAreaMatch{Unit: unit, PlayerSection: playerSection, Player: player, Index: index})
		}
	}
	if len(matches) == 0 {
		if targetPlayer != nil {
			return nil, fmt.Errorf("unit const %d matched no units for player %d", unitConst, *targetPlayer)
		}
		return nil, fmt.Errorf("unit const %d matched no units", unitConst)
	}
	return matches, nil
}

func (f *File) findUnitsByPlayer(targetPlayer *int) ([]unitAreaMatch, error) {
	if targetPlayer == nil {
		return nil, fmt.Errorf("unit player selector requires target_player")
	}
	playerSections, err := f.unitPlayerSections()
	if err != nil {
		return nil, err
	}
	player := *targetPlayer
	if player < 0 || player >= len(playerSections) {
		return nil, fmt.Errorf("unit target_player %d out of range 0..%d", player, len(playerSections)-1)
	}
	playerSection := playerSections[player]
	unitField := playerSection.field("units")
	if unitField == nil {
		return nil, fmt.Errorf("missing units field for player %d", player)
	}
	matches := make([]unitAreaMatch, 0, len(unitField.Elements))
	for index, unit := range unitField.Elements {
		matches = append(matches, unitAreaMatch{Unit: unit, PlayerSection: playerSection, Player: player, Index: index})
	}
	if len(matches) == 0 {
		return nil, fmt.Errorf("player %d has no placed units", player)
	}
	return matches, nil
}

func (f *File) unitPlayerSections() ([]*parsedNode, error) {
	unitsSection := f.root.section("Units")
	if unitsSection == nil {
		return nil, fmt.Errorf("missing Units section")
	}
	playerSections := unitsSection.list("players_units")
	if len(playerSections) == 0 {
		return nil, fmt.Errorf("missing players_units")
	}
	return playerSections, nil
}

func (f *File) moveUnitToPlayer(unitNode, fromSection *parsedNode, fromIndex, toPlayer int) error {
	playerSections, err := f.unitPlayerSections()
	if err != nil {
		return err
	}
	if toPlayer < 0 || toPlayer >= len(playerSections) {
		return fmt.Errorf("set_player %d out of range 0..%d", toPlayer, len(playerSections)-1)
	}
	toSection := playerSections[toPlayer]
	if fromSection == toSection {
		return nil
	}
	fromField := fromSection.field("units")
	if fromField == nil {
		return fmt.Errorf("missing source units field")
	}
	toField := toSection.field("units")
	if toField == nil {
		return fmt.Errorf("missing destination units field for player %d", toPlayer)
	}
	if fromIndex < 0 || fromIndex >= len(fromField.Elements) || fromField.Elements[fromIndex] != unitNode {
		return fmt.Errorf("source unit index changed before set_player")
	}
	fromField.Elements = append(fromField.Elements[:fromIndex], fromField.Elements[fromIndex+1:]...)
	toField.Elements = append(toField.Elements, unitNode)
	if err := setIntField(fromSection, "unit_count", "u32", len(fromField.Elements)); err != nil {
		return err
	}
	if err := setIntField(toSection, "unit_count", "u32", len(toField.Elements)); err != nil {
		return err
	}
	return nil
}

func (f *File) refreshUnits() error {
	units, err := f.root.unitInfo()
	if err != nil {
		return err
	}
	f.Units = units
	// DE treats next_unit_id_to_place as a monotonic allocation cursor. Keep
	// an editor-provided cursor when it is already ahead, but never emit a
	// cursor that can collide with a reference_id written by Kit.
	dataHeader := f.root.section("DataHeader")
	if dataHeader == nil {
		return fmt.Errorf("missing DataHeader section")
	}
	current, ok := dataHeader.intValue("next_unit_id_to_place")
	if !ok {
		return fmt.Errorf("missing DataHeader.next_unit_id_to_place")
	}
	required := f.nextUnitReferenceID()
	if current < required {
		if err := setIntField(dataHeader, "next_unit_id_to_place", "u32", required); err != nil {
			return err
		}
	}
	f.body = f.root.raw()
	f.InflatedBytes = len(f.body)
	return nil
}

func (f *File) nextUnitReferenceID() int {
	maxID := 0
	if f.Units == nil {
		return 1
	}
	for _, section := range f.Units.Sections {
		for _, unit := range section.Units {
			if unit.ReferenceID > maxID {
				maxID = unit.ReferenceID
			}
		}
	}
	return maxID + 1
}

func (f *File) mapTiles() ([]*parsedNode, int, int, error) {
	mapSection := f.root.section("Map")
	if mapSection == nil {
		return nil, 0, 0, fmt.Errorf("missing Map section")
	}
	width, _ := mapSection.intValue("map_width")
	height, _ := mapSection.intValue("map_height")
	tiles := mapSection.list("terrain_data")
	if width <= 0 || height <= 0 {
		return nil, 0, 0, fmt.Errorf("invalid map size %dx%d", width, height)
	}
	if len(tiles) != width*height {
		return nil, 0, 0, fmt.Errorf("terrain_data has %d tiles, want %d", len(tiles), width*height)
	}
	return tiles, width, height, nil
}

type terrainTileSnapshot map[string]terrainFieldSnapshot

type terrainFieldSnapshot struct {
	Raw   []byte
	Value any
}

func snapshotTerrainTile(src *parsedNode) (terrainTileSnapshot, error) {
	snapshot := terrainTileSnapshot{}
	for _, name := range []string{"terrain_id", "elevation", "layer", "unused"} {
		srcField := src.field(name)
		if srcField == nil {
			return nil, fmt.Errorf("terrain tile missing %s field", name)
		}
		snapshot[name] = terrainFieldSnapshot{
			Raw:   append([]byte(nil), srcField.Raw...),
			Value: srcField.Value,
		}
	}
	return snapshot, nil
}

func applyTerrainTileSnapshot(dst *parsedNode, snapshot terrainTileSnapshot) error {
	for name, srcField := range snapshot {
		dstField := dst.field(name)
		if dstField == nil {
			return fmt.Errorf("terrain tile missing %s field", name)
		}
		dstField.Raw = append([]byte(nil), srcField.Raw...)
		dstField.Value = srcField.Value
	}
	return nil
}

func cloneParsedNode(src *parsedNode) *parsedNode {
	if src == nil {
		return nil
	}
	dst := &parsedNode{
		Name:  src.Name,
		Start: src.Start,
		End:   src.End,
		Raw:   append([]byte(nil), src.Raw...),
		Value: src.Value,
	}
	if len(src.Fields) > 0 {
		dst.Fields = make([]*parsedNode, 0, len(src.Fields))
		for _, field := range src.Fields {
			dst.Fields = append(dst.Fields, cloneParsedNode(field))
		}
	}
	if len(src.Elements) > 0 {
		dst.Elements = make([]*parsedNode, 0, len(src.Elements))
		for _, elem := range src.Elements {
			dst.Elements = append(dst.Elements, cloneParsedNode(elem))
		}
		dst.Value = dst.Elements
	}
	return dst
}

type terrainMask struct {
	Name   string
	Width  int
	Height int
	Cells  []bool
}

func (m terrainMask) at(x, y int) bool {
	if x < 0 || y < 0 || x >= m.Width || y >= m.Height {
		return false
	}
	return m.Cells[y*m.Width+x]
}

func (m terrainMask) count() int {
	n := 0
	for _, v := range m.Cells {
		if v {
			n++
		}
	}
	return n
}

func (f *File) resolveMaskRecipes(recipes []MaskRecipe) (map[string]terrainMask, error) {
	tiles, width, height, err := f.mapTiles()
	if err != nil {
		return nil, err
	}
	out := map[string]terrainMask{}
	for _, recipe := range recipes {
		if recipe.Name == "" {
			return nil, fmt.Errorf("mask op %q requires name", recipe.Op)
		}
		if _, exists := out[recipe.Name]; exists {
			return nil, fmt.Errorf("duplicate mask name %q", recipe.Name)
		}
		mask, err := buildTerrainMask(recipe, out, tiles, width, height)
		if err != nil {
			return nil, err
		}
		if recipe.Invert != nil && *recipe.Invert {
			for i := range mask.Cells {
				mask.Cells[i] = !mask.Cells[i]
			}
		}
		mask.Name = recipe.Name
		out[recipe.Name] = mask
	}
	return out, nil
}

func buildTerrainMask(recipe MaskRecipe, masks map[string]terrainMask, tiles []*parsedNode, width, height int) (terrainMask, error) {
	mask := terrainMask{Name: recipe.Name, Width: width, Height: height, Cells: make([]bool, width*height)}
	switch recipe.Op {
	case "rect", "":
		rect := MapRecipe{X1: recipe.X1, Y1: recipe.Y1, X2: recipe.X2, Y2: recipe.Y2}
		if err := validateMapRect(rect, width, height); err != nil {
			return terrainMask{}, err
		}
		for y := rect.Y1; y <= rect.Y2; y++ {
			for x := rect.X1; x <= rect.X2; x++ {
				mask.Cells[y*width+x] = true
			}
		}
	case "circle":
		if recipe.Radius == nil {
			return terrainMask{}, fmt.Errorf("mask %q circle requires radius", recipe.Name)
		}
		r := *recipe.Radius
		if r < 0 {
			return terrainMask{}, fmt.Errorf("mask %q circle radius must be >= 0", recipe.Name)
		}
		for y := maxInt(0, recipe.Y1-r); y <= minInt(height-1, recipe.Y1+r); y++ {
			for x := maxInt(0, recipe.X1-r); x <= minInt(width-1, recipe.X1+r); x++ {
				if (x-recipe.X1)*(x-recipe.X1)+(y-recipe.Y1)*(y-recipe.Y1) <= r*r {
					mask.Cells[y*width+x] = true
				}
			}
		}
	case "blob":
		if recipe.Radius == nil {
			return terrainMask{}, fmt.Errorf("mask %q blob requires radius", recipe.Name)
		}
		r := *recipe.Radius
		if r < 0 {
			return terrainMask{}, fmt.Errorf("mask %q blob radius must be >= 0", recipe.Name)
		}
		seed := 0
		if recipe.Seed != nil {
			seed = *recipe.Seed
		}
		scale := math.Max(3, float64(r)/3)
		if recipe.Scale != nil {
			scale = *recipe.Scale
		}
		if scale <= 0 {
			return terrainMask{}, fmt.Errorf("mask %q blob scale must be > 0", recipe.Name)
		}
		threshold := 1.0
		if recipe.Threshold != nil {
			threshold = *recipe.Threshold
		}
		extent := int(math.Ceil(float64(r) * 1.25))
		for y := maxInt(0, recipe.Y1-extent); y <= minInt(height-1, recipe.Y1+extent); y++ {
			for x := maxInt(0, recipe.X1-extent); x <= minInt(width-1, recipe.X1+extent); x++ {
				d := math.Hypot(float64(x-recipe.X1), float64(y-recipe.Y1)) / float64(maxInt(1, r))
				edge := threshold + (valueNoise2D(float64(x)/scale, float64(y)/scale, seed)-0.5)*0.35
				if d <= edge {
					mask.Cells[y*width+x] = true
				}
			}
		}
	case "from_terrain_class":
		if len(recipe.TerrainIDs) == 0 {
			return terrainMask{}, fmt.Errorf("mask %q from_terrain_class requires terrain_ids", recipe.Name)
		}
		ids := intSet(recipe.TerrainIDs)
		for i, tile := range tiles {
			id, _ := tile.intValue("terrain_id")
			if ids[id] {
				mask.Cells[i] = true
			}
		}
	case "cells":
		if len(recipe.Cells) == 0 {
			return terrainMask{}, fmt.Errorf("mask %q cells requires cells", recipe.Name)
		}
		for _, cell := range recipe.Cells {
			if cell.X < 0 || cell.Y < 0 || cell.X >= width || cell.Y >= height {
				return terrainMask{}, fmt.Errorf("mask %q cell (%d,%d) outside map %dx%d", recipe.Name, cell.X, cell.Y, width, height)
			}
			mask.Cells[cell.Y*width+cell.X] = true
		}
	case "union", "intersect", "subtract":
		if len(recipe.Masks) == 0 {
			return terrainMask{}, fmt.Errorf("mask %q %s requires masks", recipe.Name, recipe.Op)
		}
		first, err := requireTerrainMask(masks, recipe.Masks[0])
		if err != nil {
			return terrainMask{}, err
		}
		copy(mask.Cells, first.Cells)
		if recipe.Op == "union" {
			for _, name := range recipe.Masks[1:] {
				other, err := requireTerrainMask(masks, name)
				if err != nil {
					return terrainMask{}, err
				}
				for i := range mask.Cells {
					mask.Cells[i] = mask.Cells[i] || other.Cells[i]
				}
			}
		}
		if recipe.Op == "intersect" {
			for _, name := range recipe.Masks[1:] {
				other, err := requireTerrainMask(masks, name)
				if err != nil {
					return terrainMask{}, err
				}
				for i := range mask.Cells {
					mask.Cells[i] = mask.Cells[i] && other.Cells[i]
				}
			}
		}
		if recipe.Op == "subtract" {
			for _, name := range recipe.Masks[1:] {
				other, err := requireTerrainMask(masks, name)
				if err != nil {
					return terrainMask{}, err
				}
				for i := range mask.Cells {
					mask.Cells[i] = mask.Cells[i] && !other.Cells[i]
				}
			}
		}
	case "dilate", "erode":
		if len(recipe.Masks) != 1 {
			return terrainMask{}, fmt.Errorf("mask %q %s requires exactly one input mask", recipe.Name, recipe.Op)
		}
		base, err := requireTerrainMask(masks, recipe.Masks[0])
		if err != nil {
			return terrainMask{}, err
		}
		copy(mask.Cells, base.Cells)
		iterations := 1
		if recipe.Iterations != nil {
			iterations = *recipe.Iterations
		}
		if iterations < 1 {
			return terrainMask{}, fmt.Errorf("mask %q iterations must be >= 1", recipe.Name)
		}
		for i := 0; i < iterations; i++ {
			mask.Cells = maskNeighborhoodPass(mask.Cells, width, height, recipe.Op == "dilate")
		}
	default:
		return terrainMask{}, fmt.Errorf("unsupported mask op %q", recipe.Op)
	}
	if mask.count() == 0 {
		return terrainMask{}, fmt.Errorf("mask %q produced no tiles", recipe.Name)
	}
	return mask, nil
}

func requireTerrainMask(masks map[string]terrainMask, name string) (terrainMask, error) {
	mask, ok := masks[name]
	if !ok {
		return terrainMask{}, fmt.Errorf("unknown mask %q", name)
	}
	return mask, nil
}

func maskNeighborhoodPass(in []bool, width, height int, dilate bool) []bool {
	out := make([]bool, len(in))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			if dilate {
				for yy := maxInt(0, y-1); yy <= minInt(height-1, y+1); yy++ {
					for xx := maxInt(0, x-1); xx <= minInt(width-1, x+1); xx++ {
						if in[yy*width+xx] {
							out[y*width+x] = true
							goto nextDilate
						}
					}
				}
			nextDilate:
				continue
			}
			out[y*width+x] = true
			for yy := maxInt(0, y-1); yy <= minInt(height-1, y+1); yy++ {
				for xx := maxInt(0, x-1); xx <= minInt(width-1, x+1); xx++ {
					if !in[yy*width+xx] {
						out[y*width+x] = false
					}
				}
			}
		}
	}
	return out
}

func (f *File) countMapTiles(recipe MapRecipe, masks map[string]terrainMask) (int, error) {
	_, width, height, err := f.mapTiles()
	if err != nil {
		return 0, err
	}
	if recipe.Op == "terrain_grid" {
		grid, err := resolveTerrainGridRecipe(recipe, width, height)
		if err != nil {
			return 0, err
		}
		return grid.Width * grid.Height, nil
	}
	points, err := mapPatchPointsWithMasks(recipe, width, height, masks)
	if err != nil {
		return 0, err
	}
	return len(points), nil
}

func validateMapRect(recipe MapRecipe, width, height int) error {
	if recipe.X1 > recipe.X2 || recipe.Y1 > recipe.Y2 {
		return fmt.Errorf("bad rectangle (%d,%d)-(%d,%d)", recipe.X1, recipe.Y1, recipe.X2, recipe.Y2)
	}
	if recipe.X1 < 0 || recipe.Y1 < 0 || recipe.X2 >= width || recipe.Y2 >= height {
		return fmt.Errorf("rectangle (%d,%d)-(%d,%d) outside map %dx%d", recipe.X1, recipe.Y1, recipe.X2, recipe.Y2, width, height)
	}
	return nil
}

type mapPoint struct {
	X int
	Y int
}

func mapPatchPoints(recipe MapRecipe, width, height int) ([]mapPoint, error) {
	switch recipe.Op {
	case "set_terrain_rect", "noise_fill", "erode", "semantic_erode", "layered_crossfade", "":
		if recipe.Op == "" {
			recipe.Op = "set_terrain_rect"
		}
		if err := validateMapRect(recipe, width, height); err != nil {
			return nil, err
		}
		var points []mapPoint
		for y := recipe.Y1; y <= recipe.Y2; y++ {
			for x := recipe.X1; x <= recipe.X2; x++ {
				points = append(points, mapPoint{X: x, Y: y})
			}
		}
		return points, nil
	case "set_terrain_circle":
		if recipe.Radius == nil {
			return nil, fmt.Errorf("set_terrain_circle requires radius")
		}
		if *recipe.Radius < 0 {
			return nil, fmt.Errorf("set_terrain_circle radius must be >= 0")
		}
		r := *recipe.Radius
		var points []mapPoint
		for dy := -r; dy <= r; dy++ {
			for dx := -r; dx <= r; dx++ {
				if dx*dx+dy*dy <= r*r {
					points = append(points, mapPoint{X: recipe.X1 + dx, Y: recipe.Y1 + dy})
				}
			}
		}
		return validateMapPoints(recipe.Op, uniqueSortedMapPoints(points), width, height)
	case "set_terrain_line":
		points := bresenhamLine(recipe.X1, recipe.Y1, recipe.X2, recipe.Y2)
		return validateMapPoints(recipe.Op, points, width, height)
	case "set_terrain_border":
		if err := validateMapRect(recipe, width, height); err != nil {
			return nil, err
		}
		thickness := 1
		if recipe.Thickness != nil {
			thickness = *recipe.Thickness
		}
		if thickness <= 0 {
			return nil, fmt.Errorf("set_terrain_border thickness must be > 0")
		}
		var points []mapPoint
		for y := recipe.Y1; y <= recipe.Y2; y++ {
			for x := recipe.X1; x <= recipe.X2; x++ {
				left := x - recipe.X1
				right := recipe.X2 - x
				top := y - recipe.Y1
				bottom := recipe.Y2 - y
				if left < thickness || right < thickness || top < thickness || bottom < thickness {
					points = append(points, mapPoint{X: x, Y: y})
				}
			}
		}
		return points, nil
	case "copy_terrain_area":
		if recipe.TargetX == nil || recipe.TargetY == nil {
			return nil, fmt.Errorf("copy_terrain_area requires target_x and target_y")
		}
		if err := validateMapRect(recipe, width, height); err != nil {
			return nil, err
		}
		srcWidth := recipe.X2 - recipe.X1 + 1
		srcHeight := recipe.Y2 - recipe.Y1 + 1
		target := MapRecipe{
			Op: "copy_terrain_area target",
			X1: *recipe.TargetX,
			Y1: *recipe.TargetY,
			X2: *recipe.TargetX + srcWidth - 1,
			Y2: *recipe.TargetY + srcHeight - 1,
		}
		if err := validateMapRect(target, width, height); err != nil {
			return nil, err
		}
		var points []mapPoint
		for y := recipe.Y1; y <= recipe.Y2; y++ {
			for x := recipe.X1; x <= recipe.X2; x++ {
				points = append(points, mapPoint{X: x, Y: y})
			}
		}
		return points, nil
	default:
		return nil, fmt.Errorf("unsupported map op %q", recipe.Op)
	}
}

func mapPatchPointsWithMasks(recipe MapRecipe, width, height int, masks map[string]terrainMask) ([]mapPoint, error) {
	if recipe.Mask == "" {
		if recipe.Op == "set_terrain_mask" {
			return nil, fmt.Errorf("set_terrain_mask requires mask")
		}
		return mapPatchPoints(recipe, width, height)
	}
	mask, err := requireTerrainMask(masks, recipe.Mask)
	if err != nil {
		return nil, err
	}
	x1, y1, x2, y2 := 0, 0, width-1, height-1
	if mapRecipeHasRect(recipe) {
		if err := validateMapRect(recipe, width, height); err != nil {
			return nil, err
		}
		x1, y1, x2, y2 = recipe.X1, recipe.Y1, recipe.X2, recipe.Y2
	}
	var points []mapPoint
	for y := y1; y <= y2; y++ {
		for x := x1; x <= x2; x++ {
			if mask.at(x, y) {
				points = append(points, mapPoint{X: x, Y: y})
			}
		}
	}
	if len(points) == 0 {
		return nil, fmt.Errorf("%s mask %q selected no tiles", recipe.Op, recipe.Mask)
	}
	return points, nil
}

func mapRecipeHasRect(recipe MapRecipe) bool {
	return recipe.X1 != 0 || recipe.Y1 != 0 || recipe.X2 != 0 || recipe.Y2 != 0
}

func validateMapPoints(op string, points []mapPoint, width, height int) ([]mapPoint, error) {
	if len(points) == 0 {
		return nil, fmt.Errorf("%s produced no tiles", op)
	}
	for _, point := range points {
		if point.X < 0 || point.Y < 0 || point.X >= width || point.Y >= height {
			return nil, fmt.Errorf("%s tile (%d,%d) outside map %dx%d", op, point.X, point.Y, width, height)
		}
	}
	return points, nil
}

func bresenhamLine(x1, y1, x2, y2 int) []mapPoint {
	var points []mapPoint
	dx := absInt(x2 - x1)
	dy := -absInt(y2 - y1)
	sx := -1
	if x1 < x2 {
		sx = 1
	}
	sy := -1
	if y1 < y2 {
		sy = 1
	}
	err := dx + dy
	for {
		points = append(points, mapPoint{X: x1, Y: y1})
		if x1 == x2 && y1 == y2 {
			break
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x1 += sx
		}
		if e2 <= dx {
			err += dx
			y1 += sy
		}
	}
	return uniqueSortedMapPoints(points)
}

func uniqueSortedMapPoints(points []mapPoint) []mapPoint {
	seen := map[mapPoint]bool{}
	out := make([]mapPoint, 0, len(points))
	for _, point := range points {
		if seen[point] {
			continue
		}
		seen[point] = true
		out = append(out, point)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Y != out[j].Y {
			return out[i].Y < out[j].Y
		}
		return out[i].X < out[j].X
	})
	return out
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func valueNoise2D(x, y float64, seed int) float64 {
	x0 := int(math.Floor(x))
	y0 := int(math.Floor(y))
	x1 := x0 + 1
	y1 := y0 + 1
	tx := smoothNoiseStep(x - float64(x0))
	ty := smoothNoiseStep(y - float64(y0))
	a := lerpFloat64(hashUnitNoise(x0, y0, seed), hashUnitNoise(x1, y0, seed), tx)
	b := lerpFloat64(hashUnitNoise(x0, y1, seed), hashUnitNoise(x1, y1, seed), tx)
	return lerpFloat64(a, b, ty)
}

func smoothNoiseStep(t float64) float64 {
	return t * t * t * (t*(t*6-15) + 10)
}

func lerpFloat64(a, b, t float64) float64 {
	return a + (b-a)*t
}

func hashUnitNoise(x, y, seed int) float64 {
	n := uint64(uint32(x))*0x9e3779b185ebca87 ^
		uint64(uint32(y))*0xc2b2ae3d27d4eb4f ^
		uint64(uint32(seed))*0x165667b19e3779f9
	n ^= n >> 33
	n *= 0xff51afd7ed558ccd
	n ^= n >> 33
	n *= 0xc4ceb9fe1a85ec53
	n ^= n >> 33
	return float64(n&0xffffffff) / float64(math.MaxUint32)
}

func (f *File) variableStructSpec() (SectionSpec, error) {
	spec, err := f.writeSpec()
	if err != nil {
		return SectionSpec{}, err
	}
	triggers, ok := spec.section("Triggers")
	if !ok {
		return SectionSpec{}, fmt.Errorf("missing Triggers spec")
	}
	variableSpec, ok := triggers.Structs["VariableStruct"]
	if !ok {
		return SectionSpec{}, fmt.Errorf("missing VariableStruct spec")
	}
	return variableSpec, nil
}

func (f *File) refreshTriggers() error {
	f.body = f.root.raw()
	f.InflatedBytes = len(f.body)
	triggers, err := f.root.triggerInfo()
	if err != nil {
		return err
	}
	f.Triggers = triggers
	return nil
}

func (f *File) nextVariableID() int {
	maxID := -1
	if triggers := f.root.section("Triggers"); triggers != nil {
		for _, variable := range triggers.list("variable_data") {
			id, ok := variable.intValue("variable_id")
			if ok && id > maxID {
				maxID = id
			}
		}
	}
	return maxID + 1
}

func (f *File) findVariable(recipe VariableRecipe) (*parsedNode, error) {
	node, _, err := f.findVariableWithIndex(recipe)
	return node, err
}

func (f *File) findVariableWithIndex(recipe VariableRecipe) (*parsedNode, int, error) {
	if recipe.TargetID != nil {
		return f.findVariableByIDWithIndex(*recipe.TargetID)
	}
	if recipe.ID != nil {
		return f.findVariableByIDWithIndex(*recipe.ID)
	}
	if recipe.TargetName != "" {
		return f.findVariableByNameWithIndex(recipe.TargetName)
	}
	if recipe.Name != "" && recipe.Op != "add_variable" {
		return f.findVariableByNameWithIndex(recipe.Name)
	}
	return nil, -1, fmt.Errorf("%s requires target_id, id, target_name, or name", recipe.Op)
}

func (f *File) findVariableByID(id int) (*parsedNode, error) {
	node, _, err := f.findVariableByIDWithIndex(id)
	return node, err
}

func (f *File) findVariableByIDWithIndex(id int) (*parsedNode, int, error) {
	triggers := f.root.section("Triggers")
	if triggers == nil {
		return nil, -1, fmt.Errorf("missing Triggers section")
	}
	for index, variable := range triggers.list("variable_data") {
		variableID, ok := variable.intValue("variable_id")
		if ok && variableID == id {
			return variable, index, nil
		}
	}
	return nil, -1, fmt.Errorf("variable id %d not found", id)
}

func (f *File) findVariableByName(name string) (*parsedNode, error) {
	node, _, err := f.findVariableByNameWithIndex(name)
	return node, err
}

func (f *File) findVariableByNameWithIndex(name string) (*parsedNode, int, error) {
	triggers := f.root.section("Triggers")
	if triggers == nil {
		return nil, -1, fmt.Errorf("missing Triggers section")
	}
	for index, variable := range triggers.list("variable_data") {
		variableName, _ := variable.stringValue("variable_name")
		if variableName == name {
			return variable, index, nil
		}
	}
	return nil, -1, fmt.Errorf("variable name %q not found", name)
}

func (f *File) findTrigger(recipe TriggerRecipe) (*parsedNode, error) {
	index, err := f.findTriggerIndex(recipe)
	if err != nil {
		return nil, err
	}
	triggers := f.root.section("Triggers")
	if triggers == nil {
		return nil, fmt.Errorf("missing Triggers section")
	}
	triggerNodes := triggers.list("trigger_data")
	return triggerNodes[index], nil
}

func (f *File) findTriggerIndex(recipe TriggerRecipe) (int, error) {
	triggers := f.root.section("Triggers")
	if triggers == nil {
		return 0, fmt.Errorf("missing Triggers section")
	}
	triggerNodes := triggers.list("trigger_data")
	if recipe.TargetIndex != nil {
		if *recipe.TargetIndex < 0 || *recipe.TargetIndex >= len(triggerNodes) {
			return 0, fmt.Errorf("target_index %d out of range 0..%d", *recipe.TargetIndex, len(triggerNodes)-1)
		}
		return *recipe.TargetIndex, nil
	}
	if recipe.TargetName != "" {
		for i, trigger := range triggerNodes {
			name, _ := trigger.stringValue("trigger_name")
			if name == recipe.TargetName {
				return i, nil
			}
		}
		return 0, fmt.Errorf("target_name %q not found", recipe.TargetName)
	}
	return 0, fmt.Errorf("%s requires target_index or target_name", recipe.Op)
}

func appendEffectsToTrigger(spec *Spec, trigger *parsedNode, recipes []EffectRecipe) error {
	_, effectSpec, _, err := triggerChildSpecs(spec)
	if err != nil {
		return err
	}
	field := trigger.field("effect_data")
	if field == nil {
		return fmt.Errorf("missing effect_data")
	}
	for _, recipe := range recipes {
		overrides, err := effectOverrides(recipe)
		if err != nil {
			return err
		}
		raw, err := buildStructRaw(effectSpec, overrides)
		if err != nil {
			return err
		}
		field.Elements = append(field.Elements, &parsedNode{
			Name: "EffectStruct",
			Raw:  raw,
		})
	}
	if err := setIntField(trigger, "number_of_effects", "s32", len(field.Elements)); err != nil {
		return err
	}
	return setIntListField(trigger, "effect_display_order_array", "s32", indexIntList(len(field.Elements)))
}

func appendConditionsToTrigger(spec *Spec, trigger *parsedNode, recipes []ConditionRecipe) error {
	_, _, conditionSpec, err := triggerChildSpecs(spec)
	if err != nil {
		return err
	}
	field := trigger.field("condition_data")
	if field == nil {
		return fmt.Errorf("missing condition_data")
	}
	for _, recipe := range recipes {
		overrides, err := conditionOverrides(recipe)
		if err != nil {
			return err
		}
		raw, err := buildStructRaw(conditionSpec, overrides)
		if err != nil {
			return err
		}
		field.Elements = append(field.Elements, &parsedNode{
			Name: "ConditionStruct",
			Raw:  raw,
		})
	}
	if err := setIntField(trigger, "number_of_conditions", "s32", len(field.Elements)); err != nil {
		return err
	}
	return setIntListField(trigger, "condition_display_order_array", "s32", indexIntList(len(field.Elements)))
}

func editTriggerChildLists(spec *Spec, trigger *parsedNode, recipe TriggerRecipe) error {
	if isTrue(recipe.ClearEffects) && len(recipe.RemoveEffects) > 0 {
		return fmt.Errorf("edit_trigger cannot combine clear_effects with remove_effects")
	}
	if isTrue(recipe.ClearConditions) && len(recipe.RemoveConditions) > 0 {
		return fmt.Errorf("edit_trigger cannot combine clear_conditions with remove_conditions")
	}
	if recipe.ReplaceEffects != nil && (isTrue(recipe.ClearEffects) || len(recipe.RemoveEffects) > 0 || len(recipe.Effects) > 0) {
		return fmt.Errorf("edit_trigger replace_effects cannot be combined with clear_effects, remove_effects, or effects")
	}
	if recipe.ReplaceConditions != nil && (isTrue(recipe.ClearConditions) || len(recipe.RemoveConditions) > 0 || len(recipe.Conditions) > 0) {
		return fmt.Errorf("edit_trigger replace_conditions cannot be combined with clear_conditions, remove_conditions, or conditions")
	}
	if recipe.ReplaceEffects != nil {
		effects, err := buildEffectNodes(spec, recipe.ReplaceEffects)
		if err != nil {
			return err
		}
		if err := replaceTriggerChildList(trigger, "effect_data", "number_of_effects", "effect_display_order_array", effects); err != nil {
			return err
		}
	} else if isTrue(recipe.ClearEffects) {
		if err := replaceTriggerChildList(trigger, "effect_data", "number_of_effects", "effect_display_order_array", nil); err != nil {
			return err
		}
	} else if len(recipe.RemoveEffects) > 0 {
		if err := removeTriggerChildren(trigger, "effect_data", "number_of_effects", "effect_display_order_array", recipe.RemoveEffects); err != nil {
			return err
		}
	}
	if recipe.ReplaceConditions != nil {
		conditions, err := buildConditionNodes(spec, recipe.ReplaceConditions)
		if err != nil {
			return err
		}
		if err := replaceTriggerChildList(trigger, "condition_data", "number_of_conditions", "condition_display_order_array", conditions); err != nil {
			return err
		}
	} else if isTrue(recipe.ClearConditions) {
		if err := replaceTriggerChildList(trigger, "condition_data", "number_of_conditions", "condition_display_order_array", nil); err != nil {
			return err
		}
	} else if len(recipe.RemoveConditions) > 0 {
		if err := removeTriggerChildren(trigger, "condition_data", "number_of_conditions", "condition_display_order_array", recipe.RemoveConditions); err != nil {
			return err
		}
	}
	return nil
}

func buildEffectNodes(spec *Spec, recipes []EffectRecipe) ([]*parsedNode, error) {
	_, effectSpec, _, err := triggerChildSpecs(spec)
	if err != nil {
		return nil, err
	}
	nodes := make([]*parsedNode, 0, len(recipes))
	for _, recipe := range recipes {
		overrides, err := effectOverrides(recipe)
		if err != nil {
			return nil, err
		}
		raw, err := buildEffectRaw(effectSpec, recipe, overrides)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, &parsedNode{Name: "EffectStruct", Raw: raw})
	}
	return nodes, nil
}

func buildConditionNodes(spec *Spec, recipes []ConditionRecipe) ([]*parsedNode, error) {
	_, _, conditionSpec, err := triggerChildSpecs(spec)
	if err != nil {
		return nil, err
	}
	nodes := make([]*parsedNode, 0, len(recipes))
	for _, recipe := range recipes {
		overrides, err := conditionOverrides(recipe)
		if err != nil {
			return nil, err
		}
		raw, err := buildStructRaw(conditionSpec, overrides)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, &parsedNode{Name: "ConditionStruct", Raw: raw})
	}
	return nodes, nil
}

func removeTriggerChildren(trigger *parsedNode, dataField, countField, orderField string, indices []int) error {
	field := trigger.field(dataField)
	if field == nil {
		return fmt.Errorf("missing %s", dataField)
	}
	remove, err := normalizedRemovalIndices(indices, len(field.Elements), dataField)
	if err != nil {
		return err
	}
	next := make([]*parsedNode, 0, len(field.Elements)-len(remove))
	for i, node := range field.Elements {
		if !remove[i] {
			next = append(next, node)
		}
	}
	return replaceTriggerChildList(trigger, dataField, countField, orderField, next)
}

func replaceTriggerChildList(trigger *parsedNode, dataField, countField, orderField string, elements []*parsedNode) error {
	field := trigger.field(dataField)
	if field == nil {
		return fmt.Errorf("missing %s", dataField)
	}
	field.Elements = elements
	if err := setIntField(trigger, countField, "s32", len(field.Elements)); err != nil {
		return err
	}
	return setIntListField(trigger, orderField, "s32", indexIntList(len(field.Elements)))
}

func normalizedRemovalIndices(indices []int, count int, label string) (map[int]bool, error) {
	remove := map[int]bool{}
	for _, index := range indices {
		if index < 0 || index >= count {
			return nil, fmt.Errorf("%s removal index %d out of range 0..%d", label, index, count-1)
		}
		if remove[index] {
			return nil, fmt.Errorf("%s removal index %d repeated", label, index)
		}
		remove[index] = true
	}
	return remove, nil
}

func isTrue(value *bool) bool {
	return value != nil && *value
}

func (f *File) Write(path string) error {
	return f.write(path, true)
}

// WriteReencoded serializes the parsed model instead of reusing the original
// compressed bytes. It is primarily useful for round-trip validation and for
// callers that need a freshly encoded scenario after a read-only inspection.
func (f *File) WriteReencoded(path string) error {
	return f.write(path, false)
}

func (f *File) write(path string, preserveOriginal bool) error {
	if !SupportsWriteVersion(f.Version) {
		return fmt.Errorf("scenario version %q is read-only in AoE2Kit; writing currently targets DE 1.57 through 1.59", f.Version)
	}
	// Preserve the editor's exact compressed representation when the parsed
	// model is untouched. This makes read->write a true byte-preserving copy,
	// including compression choices and opaque bytes not yet semantically named.
	if preserveOriginal && len(f.original) > 0 && bytes.Equal(f.header, f.originalHeader) && bytes.Equal(f.body, f.originalBody) {
		return os.WriteFile(path, f.original, 0644)
	}
	body, err := f.RebuildBody()
	if err != nil {
		return err
	}
	compressed, err := DeflateRaw(body)
	if err != nil {
		return err
	}
	var out bytes.Buffer
	out.Write(f.header)
	out.Write(compressed)
	return os.WriteFile(path, out.Bytes(), 0644)
}

func buildUnitFromRecipe(spec *Spec, recipe UnitRecipe, fallbackReferenceID int) ([]byte, error) {
	if recipe.Op != "add_unit" {
		return nil, fmt.Errorf("unsupported unit op %q", recipe.Op)
	}
	if recipe.UnitConst <= 0 {
		return nil, fmt.Errorf("add_unit requires positive unit_const")
	}
	unitSpec, err := unitStructSpec(spec)
	if err != nil {
		return nil, err
	}
	return buildStructRaw(unitSpec, map[string]any{
		"x":                       floatPtrValue(recipe.X, 0.5),
		"y":                       floatPtrValue(recipe.Y, 0.5),
		"z":                       floatPtrValue(recipe.Z, 0),
		"reference_id":            intValue(recipe.ReferenceID, fallbackReferenceID),
		"unit_const":              recipe.UnitConst,
		"status":                  intValue(recipe.Status, 2),
		"rotation":                floatPtrValue(recipe.Rotation, 0),
		"initial_animation_frame": intValue(recipe.InitialAnimationFrame, 0),
		"garrisoned_in_id":        intValue(recipe.GarrisonedInID, -1),
		"caption_string_id":       intValue(recipe.CaptionStringID, -1),
		"caption_string":          recipe.CaptionString,
	})
}

func buildDisplayInstructionTrigger(spec *Spec, recipe TriggerRecipe) ([]byte, error) {
	return buildSingleEffectTrigger(spec, recipe, map[string]any{
		"effect_type":                      20,
		"source_player":                    intValue(recipe.SourcePlayer, 1),
		"display_time":                     intValue(recipe.DisplayTime, 10),
		"instruction_panel_position":       intValue(recipe.InstructionPanelPosition, 0),
		"play_sound":                       intValue(recipe.PlaySound, 0),
		"message":                          recipe.Message,
		"use_tag_color_for_icon":           intValue(recipe.UseTagColorForIcon, 0),
		"number_of_units_selected":         -1,
		"selected_object_ids":              []any{},
		"message_option1":                  "",
		"message_option2":                  "",
		"sound_name":                       "",
		"object_list_unit_id":              -1,
		"string_id":                        -1,
		"legacy_location_object_reference": -1,
	})
}

func buildCreateObjectTrigger(spec *Spec, recipe TriggerRecipe) ([]byte, error) {
	if recipe.ObjectListUnitID == nil {
		return nil, fmt.Errorf("add_create_object requires object_list_unit_id")
	}
	if recipe.LocationX == nil {
		return nil, fmt.Errorf("add_create_object requires location_x")
	}
	if recipe.LocationY == nil {
		return nil, fmt.Errorf("add_create_object requires location_y")
	}
	return buildSingleEffectTrigger(spec, recipe, map[string]any{
		"effect_type":              11,
		"object_list_unit_id":      *recipe.ObjectListUnitID,
		"source_player":            intValue(recipe.SourcePlayer, 1),
		"location_x":               *recipe.LocationX,
		"location_y":               *recipe.LocationY,
		"item_id":                  intValue(recipe.ItemID, -1),
		"facet":                    intValue(recipe.Facet, -1),
		"disable_sound":            intValue(recipe.DisableSound, 0),
		"number_of_units_selected": -1,
		"selected_object_ids":      []any{},
	})
}

func buildSingleEffectTrigger(spec *Spec, recipe TriggerRecipe, effectOverrides map[string]any) ([]byte, error) {
	return buildTriggerRaw(spec, recipe, []map[string]any{effectOverrides}, nil)
}

func buildTriggerFromRecipe(spec *Spec, recipe TriggerRecipe) ([]byte, error) {
	var effects []map[string]any
	for _, effect := range recipe.Effects {
		overrides, err := effectOverrides(effect)
		if err != nil {
			return nil, err
		}
		effects = append(effects, overrides)
	}
	var conditions []map[string]any
	for _, condition := range recipe.Conditions {
		overrides, err := conditionOverrides(condition)
		if err != nil {
			return nil, err
		}
		conditions = append(conditions, overrides)
	}
	return buildTriggerRaw(spec, recipe, effects, conditions)
}

func buildTriggerRaw(spec *Spec, recipe TriggerRecipe, effectOverrides []map[string]any, conditionOverrides []map[string]any) ([]byte, error) {
	triggerSpec, effectSpec, conditionSpec, err := triggerChildSpecs(spec)
	if err != nil {
		return nil, err
	}
	enabled := 0
	if recipe.Enabled != nil && *recipe.Enabled {
		enabled = 1
	}
	looping := 0
	if recipe.Looping != nil && *recipe.Looping {
		looping = 1
	}
	executeOnLoad := 0
	if recipe.ExecuteOnLoad != nil && *recipe.ExecuteOnLoad {
		executeOnLoad = 1
	}
	displayAsObjective := 0
	if recipe.DisplayAsObjective != nil && *recipe.DisplayAsObjective {
		displayAsObjective = 1
	}
	displayOnScreen := 0
	if recipe.DisplayOnScreen != nil && *recipe.DisplayOnScreen {
		displayOnScreen = 1
	}
	makeHeader := 0
	if recipe.MakeHeader != nil && *recipe.MakeHeader {
		makeHeader = 1
	}
	muteObjectives := 0
	if recipe.MuteObjectives != nil && *recipe.MuteObjectives {
		muteObjectives = 1
	}
	name := recipe.Name
	if name == "" {
		name = "AoE2Kit Trigger"
	}
	effects := make([][]byte, 0, len(effectOverrides))
	for i, overrides := range effectOverrides {
		effectRecipe := EffectRecipe{}
		if i < len(recipe.Effects) {
			effectRecipe = recipe.Effects[i]
		}
		effect, err := buildEffectRaw(effectSpec, effectRecipe, overrides)
		if err != nil {
			return nil, err
		}
		effects = append(effects, effect)
	}
	conditions := make([][]byte, 0, len(conditionOverrides))
	for _, overrides := range conditionOverrides {
		condition, err := buildStructRaw(conditionSpec, overrides)
		if err != nil {
			return nil, err
		}
		conditions = append(conditions, condition)
	}
	return buildStructRaw(triggerSpec, map[string]any{
		"enabled":                           enabled,
		"looping":                           looping,
		"execute_on_load":                   executeOnLoad,
		"description_string_table_id":       intValue(recipe.DescriptionStringID, 0),
		"display_as_objective":              displayAsObjective,
		"objective_description_order":       intValue(recipe.ObjectiveDescriptionOrder, 0),
		"make_header":                       makeHeader,
		"short_description_string_table_id": intValue(recipe.ShortDescriptionStringID, 0),
		"display_on_screen":                 displayOnScreen,
		"mute_objectives":                   muteObjectives,
		"trigger_description":               recipe.Description,
		"trigger_name":                      name,
		"short_description":                 recipe.ShortDescription,
		"number_of_effects":                 len(effects),
		"effect_data":                       effects,
		"effect_display_order_array":        indexAnyList(len(effects)),
		"number_of_conditions":              len(conditions),
		"condition_data":                    conditions,
		"condition_display_order_array":     indexAnyList(len(conditions)),
	})
}

func buildEffectRaw(spec SectionSpec, recipe EffectRecipe, overrides map[string]any) ([]byte, error) {
	raw, err := buildStructRaw(spec, overrides)
	if err != nil {
		return nil, err
	}
	if recipe.Op != "replace_object" {
		return raw, nil
	}
	node, consumed, err := DecodeEffectNode(raw)
	if err != nil {
		return nil, err
	}
	if consumed != len(raw) {
		return nil, fmt.Errorf("replace_object effect parse stopped at %d of %d", consumed, len(raw))
	}
	for _, field := range []string{"message", "sound_name", "message_option1", "message_option2"} {
		if err := setZeroLengthStringField(node, field, "str32"); err != nil {
			return nil, err
		}
	}
	return node.raw(), nil
}

func intValue(value *int, fallback int) int {
	if value == nil {
		return fallback
	}
	return *value
}

func floatPtrValue(value *float64, fallback float64) float64 {
	if value == nil {
		return fallback
	}
	return *value
}

func effectOverrides(effect EffectRecipe) (map[string]any, error) {
	switch effect.Op {
	case "change_diplomacy":
		if effect.TargetPlayer == nil {
			return nil, fmt.Errorf("change_diplomacy requires target_player")
		}
		if effect.Diplomacy == nil {
			return nil, fmt.Errorf("change_diplomacy requires diplomacy")
		}
		return map[string]any{
			"effect_type":              effectTypeID("change_diplomacy"),
			"source_player":            intValue(effect.SourcePlayer, 1),
			"target_player":            *effect.TargetPlayer,
			"diplomacy":                *effect.Diplomacy,
			"mutual_diplomacy":         intValue(effect.MutualDiplomacy, -1),
			"number_of_units_selected": -1,
			"selected_object_ids":      []any{},
			"message":                  "",
			"sound_name":               "",
		}, nil
	case "display_instructions":
		return map[string]any{
			"effect_type":                      effectTypeID("display_instructions"),
			"source_player":                    intValue(effect.SourcePlayer, 1),
			"display_time":                     intValue(effect.DisplayTime, 10),
			"instruction_panel_position":       intValue(effect.InstructionPanelPosition, 0),
			"play_sound":                       intValue(effect.PlaySound, 0),
			"message":                          effect.Message,
			"use_tag_color_for_icon":           intValue(effect.UseTagColorForIcon, 0),
			"number_of_units_selected":         -1,
			"selected_object_ids":              []any{},
			"message_option1":                  "",
			"message_option2":                  "",
			"sound_name":                       "",
			"object_list_unit_id":              -1,
			"string_id":                        -1,
			"legacy_location_object_reference": -1,
		}, nil
	case "display_timer":
		return map[string]any{
			"effect_type":                      effectTypeID("display_timer"),
			"source_player":                    intValue(effect.SourcePlayer, 1),
			"display_time":                     intValue(effect.DisplayTime, 999999),
			"time_unit":                        intValue(effect.TimeUnit, 2),
			"timer":                            intValue(effect.TimerID, 0),
			"reset_timer":                      intValue(effect.ResetTimer, 0),
			"message":                          effect.Message,
			"number_of_units_selected":         -1,
			"selected_object_ids":              []any{},
			"message_option1":                  "",
			"message_option2":                  "",
			"sound_name":                       "",
			"object_list_unit_id":              -1,
			"string_id":                        -1,
			"legacy_location_object_reference": -1,
		}, nil
	case "send_chat":
		return map[string]any{
			"effect_type":                      effectTypeID("send_chat"),
			"source_player":                    intValue(effect.SourcePlayer, 1),
			"message":                          effect.Message,
			"number_of_units_selected":         -1,
			"selected_object_ids":              []any{},
			"message_option1":                  "",
			"message_option2":                  "",
			"sound_name":                       "",
			"object_list_unit_id":              -1,
			"string_id":                        -1,
			"legacy_location_object_reference": -1,
		}, nil
	case "play_sound":
		if effect.SoundName == "" {
			return nil, fmt.Errorf("play_sound requires sound_name")
		}
		return map[string]any{
			"effect_type":              effectTypeID("play_sound"),
			"source_player":            intValue(effect.SourcePlayer, 1),
			"sound_name":               effect.SoundName,
			"play_sound":               intValue(effect.PlaySound, -1),
			"global_sound":             intValue(effect.GlobalSound, -1),
			"number_of_units_selected": -1,
			"selected_object_ids":      []any{},
			"message":                  "",
		}, nil
	case "research_technology":
		if effect.Technology == nil {
			return nil, fmt.Errorf("research_technology requires technology")
		}
		return map[string]any{
			"effect_type":                      effectTypeID("research_technology"),
			"source_player":                    intValue(effect.SourcePlayer, 1),
			"technology":                       *effect.Technology,
			"force_research_technology":        intValue(effect.ForceResearchTechnology, 0),
			"number_of_units_selected":         -1,
			"selected_object_ids":              []any{},
			"message":                          "",
			"sound_name":                       "",
			"legacy_location_object_reference": -1,
		}, nil
	case "create_object":
		if effect.ObjectListUnitID == nil {
			return nil, fmt.Errorf("create_object requires object_list_unit_id")
		}
		if effect.LocationX == nil {
			return nil, fmt.Errorf("create_object requires location_x")
		}
		if effect.LocationY == nil {
			return nil, fmt.Errorf("create_object requires location_y")
		}
		return map[string]any{
			"effect_type":              effectTypeID("create_object"),
			"object_list_unit_id":      *effect.ObjectListUnitID,
			"source_player":            intValue(effect.SourcePlayer, 1),
			"location_x":               *effect.LocationX,
			"location_y":               *effect.LocationY,
			"item_id":                  intValue(effect.ItemID, -1),
			"facet":                    intValue(effect.Facet, -1),
			"disable_sound":            intValue(effect.DisableSound, 0),
			"number_of_units_selected": -1,
			"selected_object_ids":      []any{},
		}, nil
	case "place_foundation", "build_object":
		if effect.ObjectListUnitID == nil {
			return nil, fmt.Errorf("%s requires object_list_unit_id", effect.Op)
		}
		if effect.LocationX == nil {
			return nil, fmt.Errorf("%s requires location_x", effect.Op)
		}
		if effect.LocationY == nil {
			return nil, fmt.Errorf("%s requires location_y", effect.Op)
		}
		return map[string]any{
			"effect_type":              effectTypeID(effect.Op),
			"object_list_unit_id":      *effect.ObjectListUnitID,
			"source_player":            intValue(effect.SourcePlayer, 1),
			"location_x":               *effect.LocationX,
			"location_y":               *effect.LocationY,
			"number_of_units_selected": -1,
			"selected_object_ids":      []any{},
		}, nil
	case "kill_object":
		selected := selectedObjectAnyList(effect.SelectedObjectIDs)
		return map[string]any{
			"effect_type":              effectTypeID("kill_object"),
			"object_list_unit_id":      intValue(effect.ObjectListUnitID, -1),
			"source_player":            intValue(effect.SourcePlayer, 1),
			"area_x1":                  intValue(effect.AreaX1, -1),
			"area_y1":                  intValue(effect.AreaY1, -1),
			"area_x2":                  intValue(effect.AreaX2, -1),
			"area_y2":                  intValue(effect.AreaY2, -1),
			"object_group":             intValue(effect.ObjectGroup, -1),
			"object_type":              intValue(effect.ObjectType, -1),
			"max_units_affected":       intValue(effect.MaxUnitsAffected, -1),
			"number_of_units_selected": selectedCount(selected),
			"selected_object_ids":      selected,
		}, nil
	case "remove_object":
		selected := selectedObjectAnyList(effect.SelectedObjectIDs)
		return map[string]any{
			"effect_type":              effectTypeID("remove_object"),
			"object_list_unit_id":      intValue(effect.ObjectListUnitID, -1),
			"source_player":            intValue(effect.SourcePlayer, 1),
			"area_x1":                  intValue(effect.AreaX1, -1),
			"area_y1":                  intValue(effect.AreaY1, -1),
			"area_x2":                  intValue(effect.AreaX2, -1),
			"area_y2":                  intValue(effect.AreaY2, -1),
			"object_group":             intValue(effect.ObjectGroup, -1),
			"object_type":              intValue(effect.ObjectType, -1),
			"object_state":             intValue(effect.ObjectState, -1),
			"max_units_affected":       intValue(effect.MaxUnitsAffected, -1),
			"number_of_units_selected": selectedCount(selected),
			"selected_object_ids":      selected,
		}, nil
	case "replace_object":
		if effect.ObjectListUnitID2 == nil {
			return nil, fmt.Errorf("replace_object requires object_list_unit_id_2")
		}
		selected := selectedObjectAnyList(effect.SelectedObjectIDs)
		hasSelector := len(selected) > 0 || effect.ObjectListUnitID != nil || effect.AreaX1 != nil || effect.AreaY1 != nil || effect.AreaX2 != nil || effect.AreaY2 != nil || effect.ObjectGroup != nil || effect.ObjectType != nil
		if !hasSelector {
			return nil, fmt.Errorf("replace_object requires selected_object_ids or an area/object-list selector")
		}
		return map[string]any{
			"effect_type":           effectTypeID("replace_object"),
			"object_list_unit_id":   intValue(effect.ObjectListUnitID, -1),
			"object_list_unit_id_2": *effect.ObjectListUnitID2,
			"source_player":         intValue(effect.SourcePlayer, 1),
			"target_player":         intValue(effect.TargetPlayer, 2),
			// DE's editor default is the all-ones f32 NaN payload, not the
			// canonical quiet-NaN payload returned by math.NaN().
			"quantity_float":           float64(math.Float32frombits(0xffffffff)),
			"area_x1":                  intValue(effect.AreaX1, -1),
			"area_y1":                  intValue(effect.AreaY1, -1),
			"area_x2":                  intValue(effect.AreaX2, -1),
			"area_y2":                  intValue(effect.AreaY2, -1),
			"object_group":             intValue(effect.ObjectGroup, -1),
			"object_type":              intValue(effect.ObjectType, -1),
			"facet2":                   intValue(effect.Facet2, -1),
			"max_units_affected":       intValue(effect.MaxUnitsAffected, -1),
			"number_of_units_selected": selectedCount(selected),
			"selected_object_ids":      selected,
		}, nil
	case "task_object":
		selected := selectedObjectAnyList(effect.SelectedObjectIDs)
		return map[string]any{
			"effect_type":                   effectTypeID("task_object"),
			"object_list_unit_id":           intValue(effect.ObjectListUnitID, -1),
			"source_player":                 intValue(effect.SourcePlayer, 1),
			"location_x":                    intValue(effect.LocationX, -1),
			"location_y":                    intValue(effect.LocationY, -1),
			"location_object_reference":     intValue(effect.LocationObjectReference, -1),
			"area_x1":                       intValue(effect.AreaX1, -1),
			"area_y1":                       intValue(effect.AreaY1, -1),
			"area_x2":                       intValue(effect.AreaX2, -1),
			"area_y2":                       intValue(effect.AreaY2, -1),
			"object_group":                  intValue(effect.ObjectGroup, -1),
			"object_type":                   intValue(effect.ObjectType, -1),
			"action_type":                   intValue(effect.ActionType, -1),
			"disable_garrison_unload_sound": intValue(effect.DisableGarrisonUnloadSound, -1),
			"max_units_affected":            intValue(effect.MaxUnitsAffected, -1),
			"issue_group_command":           intValue(effect.IssueGroupCommand, -1),
			"queue_action":                  intValue(effect.QueueAction, -1),
			"number_of_units_selected":      selectedCount(selected),
			"selected_object_ids":           selected,
		}, nil
	case "change_view":
		if effect.LocationX == nil {
			return nil, fmt.Errorf("change_view requires location_x")
		}
		if effect.LocationY == nil {
			return nil, fmt.Errorf("change_view requires location_y")
		}
		return map[string]any{
			"effect_type":              effectTypeID("change_view"),
			"source_player":            intValue(effect.SourcePlayer, 1),
			"location_x":               *effect.LocationX,
			"location_y":               *effect.LocationY,
			"scroll":                   intValue(effect.Scroll, -1),
			"number_of_units_selected": -1,
			"selected_object_ids":      []any{},
			"message":                  "",
			"sound_name":               "",
		}, nil
	case "change_player_name", "change_civilization_name":
		if effect.Message == "" {
			return nil, fmt.Errorf("%s requires message", effect.Op)
		}
		return map[string]any{
			"effect_type":              effectTypeID(effect.Op),
			"source_player":            intValue(effect.SourcePlayer, 1),
			"message":                  effect.Message,
			"number_of_units_selected": -1,
			"selected_object_ids":      []any{},
			"sound_name":               "",
		}, nil
	case "change_player_color":
		if effect.PlayerColor == nil {
			return nil, fmt.Errorf("change_player_color requires player_color")
		}
		return map[string]any{
			"effect_type":              effectTypeID("change_player_color"),
			"source_player":            intValue(effect.SourcePlayer, 1),
			"player_color":             *effect.PlayerColor,
			"number_of_units_selected": -1,
			"selected_object_ids":      []any{},
			"message":                  "",
			"sound_name":               "",
		}, nil
	case "change_ownership":
		selected := selectedObjectAnyList(effect.SelectedObjectIDs)
		return map[string]any{
			"effect_type":              effectTypeID("change_ownership"),
			"object_list_unit_id":      intValue(effect.ObjectListUnitID, -1),
			"source_player":            intValue(effect.SourcePlayer, 1),
			"target_player":            intValue(effect.TargetPlayer, 1),
			"area_x1":                  intValue(effect.AreaX1, -1),
			"area_y1":                  intValue(effect.AreaY1, -1),
			"area_x2":                  intValue(effect.AreaX2, -1),
			"area_y2":                  intValue(effect.AreaY2, -1),
			"object_group":             intValue(effect.ObjectGroup, -1),
			"object_type":              intValue(effect.ObjectType, -1),
			"flash_object":             intValue(effect.FlashObject, 0),
			"max_units_affected":       intValue(effect.MaxUnitsAffected, -1),
			"number_of_units_selected": selectedCount(selected),
			"selected_object_ids":      selected,
		}, nil
	case "change_object_name":
		selected := selectedObjectAnyList(effect.SelectedObjectIDs)
		return map[string]any{
			"effect_type":              effectTypeID("change_object_name"),
			"object_list_unit_id":      intValue(effect.ObjectListUnitID, -1),
			"source_player":            intValue(effect.SourcePlayer, 1),
			"string_id":                intValue(effect.StringID, -1),
			"area_x1":                  intValue(effect.AreaX1, -1),
			"area_y1":                  intValue(effect.AreaY1, -1),
			"area_x2":                  intValue(effect.AreaX2, -1),
			"area_y2":                  intValue(effect.AreaY2, -1),
			"message":                  effect.Message,
			"max_units_affected":       intValue(effect.MaxUnitsAffected, -1),
			"number_of_units_selected": selectedCount(selected),
			"selected_object_ids":      selected,
		}, nil
	case "change_object_description":
		selected := selectedObjectAnyList(effect.SelectedObjectIDs)
		return map[string]any{
			"effect_type":              effectTypeID("change_object_description"),
			"object_list_unit_id":      intValue(effect.ObjectListUnitID, -1),
			"source_player":            intValue(effect.SourcePlayer, 1),
			"string_id":                intValue(effect.StringID, -1),
			"area_x1":                  intValue(effect.AreaX1, -1),
			"area_y1":                  intValue(effect.AreaY1, -1),
			"area_x2":                  intValue(effect.AreaX2, -1),
			"area_y2":                  intValue(effect.AreaY2, -1),
			"message":                  effect.Message,
			"max_units_affected":       intValue(effect.MaxUnitsAffected, -1),
			"number_of_units_selected": selectedCount(selected),
			"selected_object_ids":      selected,
		}, nil
	case "change_object_hp":
		selected := selectedObjectAnyList(effect.SelectedObjectIDs)
		return map[string]any{
			"effect_type":              effectTypeID("change_object_hp"),
			"quantity":                 intValue(effect.Quantity, 0),
			"object_list_unit_id":      intValue(effect.ObjectListUnitID, -1),
			"source_player":            intValue(effect.SourcePlayer, 1),
			"area_x1":                  intValue(effect.AreaX1, -1),
			"area_y1":                  intValue(effect.AreaY1, -1),
			"area_x2":                  intValue(effect.AreaX2, -1),
			"area_y2":                  intValue(effect.AreaY2, -1),
			"object_group":             intValue(effect.ObjectGroup, -1),
			"object_type":              intValue(effect.ObjectType, -1),
			"operation":                intValue(effect.Operation, 1),
			"max_units_affected":       intValue(effect.MaxUnitsAffected, -1),
			"number_of_units_selected": selectedCount(selected),
			"selected_object_ids":      selected,
		}, nil
	case "damage_object", "heal_object", "change_object_attack", "change_object_armor", "change_object_range", "change_object_speed", "change_object_caption":
		selected := selectedObjectAnyList(effect.SelectedObjectIDs)
		return map[string]any{
			"effect_type":              effectTypeID(effect.Op),
			"quantity":                 intValue(effect.Quantity, 0),
			"object_list_unit_id":      intValue(effect.ObjectListUnitID, -1),
			"source_player":            intValue(effect.SourcePlayer, 1),
			"string_id":                intValue(effect.StringID, -1),
			"area_x1":                  intValue(effect.AreaX1, -1),
			"area_y1":                  intValue(effect.AreaY1, -1),
			"area_x2":                  intValue(effect.AreaX2, -1),
			"area_y2":                  intValue(effect.AreaY2, -1),
			"object_group":             intValue(effect.ObjectGroup, -1),
			"object_type":              intValue(effect.ObjectType, -1),
			"operation":                intValue(effect.Operation, 1),
			"message":                  effect.Message,
			"max_units_affected":       intValue(effect.MaxUnitsAffected, -1),
			"number_of_units_selected": selectedCount(selected),
			"selected_object_ids":      selected,
		}, nil
	case "freeze_unit", "stop_unit", "disable_unit_targeting", "enable_unit_targeting", "disable_object_selection", "enable_object_selection", "enable_object_deletion", "disable_object_deletion", "disable_unit_attackable", "enable_unit_attackable":
		selected := selectedObjectAnyList(effect.SelectedObjectIDs)
		return map[string]any{
			"effect_type":              effectTypeID(effect.Op),
			"object_list_unit_id":      intValue(effect.ObjectListUnitID, -1),
			"source_player":            intValue(effect.SourcePlayer, 1),
			"area_x1":                  intValue(effect.AreaX1, -1),
			"area_y1":                  intValue(effect.AreaY1, -1),
			"area_x2":                  intValue(effect.AreaX2, -1),
			"area_y2":                  intValue(effect.AreaY2, -1),
			"object_group":             intValue(effect.ObjectGroup, -1),
			"object_type":              intValue(effect.ObjectType, -1),
			"object_filter":            intValue(effect.ObjectFilter, -1),
			"max_units_affected":       intValue(effect.MaxUnitsAffected, -1),
			"number_of_units_selected": selectedCount(selected),
			"selected_object_ids":      selected,
			"message":                  "",
			"sound_name":               "",
		}, nil
	case "teleport_object":
		if effect.LocationX == nil {
			return nil, fmt.Errorf("teleport_object requires location_x")
		}
		if effect.LocationY == nil {
			return nil, fmt.Errorf("teleport_object requires location_y")
		}
		selected := selectedObjectAnyList(effect.SelectedObjectIDs)
		return map[string]any{
			"effect_type":              effectTypeID("teleport_object"),
			"object_list_unit_id":      intValue(effect.ObjectListUnitID, -1),
			"source_player":            intValue(effect.SourcePlayer, 1),
			"location_x":               *effect.LocationX,
			"location_y":               *effect.LocationY,
			"area_x1":                  intValue(effect.AreaX1, -1),
			"area_y1":                  intValue(effect.AreaY1, -1),
			"area_x2":                  intValue(effect.AreaX2, -1),
			"area_y2":                  intValue(effect.AreaY2, -1),
			"object_group":             intValue(effect.ObjectGroup, -1),
			"object_type":              intValue(effect.ObjectType, -1),
			"max_units_affected":       intValue(effect.MaxUnitsAffected, -1),
			"number_of_units_selected": selectedCount(selected),
			"selected_object_ids":      selected,
		}, nil
	case "change_object_stance":
		selected := selectedObjectAnyList(effect.SelectedObjectIDs)
		return map[string]any{
			"effect_type":              effectTypeID("change_object_stance"),
			"object_list_unit_id":      intValue(effect.ObjectListUnitID, -1),
			"source_player":            intValue(effect.SourcePlayer, 1),
			"area_x1":                  intValue(effect.AreaX1, -1),
			"area_y1":                  intValue(effect.AreaY1, -1),
			"area_x2":                  intValue(effect.AreaX2, -1),
			"area_y2":                  intValue(effect.AreaY2, -1),
			"object_group":             intValue(effect.ObjectGroup, -1),
			"object_type":              intValue(effect.ObjectType, -1),
			"attack_stance":            intValue(effect.AttackStance, -1),
			"max_units_affected":       intValue(effect.MaxUnitsAffected, -1),
			"number_of_units_selected": selectedCount(selected),
			"selected_object_ids":      selected,
		}, nil
	case "set_player_visibility", "set_visibility", "reveal_map":
		return map[string]any{
			"effect_type":              effectTypeID("set_player_visibility"),
			"source_player":            intValue(effect.SourcePlayer, 1),
			"target_player":            intValue(effect.TargetPlayer, 1),
			"visibility_state":         intValue(effect.VisibilityState, 0),
			"number_of_units_selected": -1,
			"selected_object_ids":      []any{},
			"message":                  "",
			"sound_name":               "",
		}, nil
	case "enable_disable_object":
		selected := selectedObjectAnyList(effect.SelectedObjectIDs)
		return map[string]any{
			"effect_type":              effectTypeID("enable_disable_object"),
			"object_list_unit_id":      intValue(effect.ObjectListUnitID, -1),
			"source_player":            intValue(effect.SourcePlayer, 1),
			"enabled":                  intValue(effect.Enabled, 1),
			"area_x1":                  intValue(effect.AreaX1, -1),
			"area_y1":                  intValue(effect.AreaY1, -1),
			"area_x2":                  intValue(effect.AreaX2, -1),
			"area_y2":                  intValue(effect.AreaY2, -1),
			"object_group":             intValue(effect.ObjectGroup, -1),
			"object_type":              intValue(effect.ObjectType, -1),
			"max_units_affected":       intValue(effect.MaxUnitsAffected, -1),
			"number_of_units_selected": selectedCount(selected),
			"selected_object_ids":      selected,
			"message":                  "",
			"sound_name":               "",
		}, nil
	case "enable_disable_technology":
		if effect.Technology == nil {
			return nil, fmt.Errorf("enable_disable_technology requires technology")
		}
		return map[string]any{
			"effect_type":              effectTypeID("enable_disable_technology"),
			"source_player":            intValue(effect.SourcePlayer, 1),
			"technology":               *effect.Technology,
			"enabled":                  intValue(effect.Enabled, 1),
			"number_of_units_selected": -1,
			"selected_object_ids":      []any{},
			"message":                  "",
			"sound_name":               "",
		}, nil
	case "change_train_location", "add_train_location":
		if effect.ObjectListUnitID == nil {
			return nil, fmt.Errorf("%s requires object_list_unit_id", effect.Op)
		}
		return map[string]any{
			"effect_type":              effectTypeID(effect.Op),
			"object_list_unit_id":      *effect.ObjectListUnitID,
			"object_list_unit_id_2":    intValue(effect.ObjectListUnitID2, -1),
			"source_player":            intValue(effect.SourcePlayer, 1),
			"button_location":          intValue(effect.ButtonLocation, -1),
			"hotkey":                   intValue(effect.Hotkey, -1),
			"train_time":               intValue(effect.TrainTime, -1),
			"number_of_units_selected": -1,
			"selected_object_ids":      []any{},
			"message":                  "",
			"sound_name":               "",
		}, nil
	case "change_technology_location":
		if effect.Technology == nil {
			return nil, fmt.Errorf("change_technology_location requires technology")
		}
		return map[string]any{
			"effect_type":              effectTypeID("change_technology_location"),
			"technology":               *effect.Technology,
			"object_list_unit_id":      intValue(effect.ObjectListUnitID, -1),
			"source_player":            intValue(effect.SourcePlayer, 1),
			"button_location":          intValue(effect.ButtonLocation, -1),
			"number_of_units_selected": -1,
			"selected_object_ids":      []any{},
			"message":                  "",
			"sound_name":               "",
		}, nil
	case "modify_attribute":
		if effect.ObjectAttributes == nil {
			return nil, fmt.Errorf("modify_attribute requires object_attributes")
		}
		operation, err := resolveTriggerOperation(effect.Operation, effect.OperationName)
		if err != nil {
			return nil, err
		}
		out := map[string]any{
			"effect_type":              effectTypeID("modify_attribute"),
			"quantity":                 intValue(effect.Quantity, -1),
			"object_list_unit_id":      intValue(effect.ObjectListUnitID, -1),
			"source_player":            intValue(effect.SourcePlayer, 1),
			"item_id":                  intValue(effect.ItemID, -1),
			"operation":                operation,
			"object_attributes":        *effect.ObjectAttributes,
			"message":                  effect.Message,
			"number_of_units_selected": -1,
			"selected_object_ids":      []any{},
		}
		if effect.QuantityFloat != nil {
			out["quantity_float"] = *effect.QuantityFloat
		}
		return out, nil
	case "modify_resource":
		resource := intValue(effect.Resource, intValue(effect.TributeList, 0))
		return map[string]any{
			"effect_type":              effectTypeID("modify_resource"),
			"quantity":                 intValue(effect.Quantity, 0),
			"tribute_list":             resource,
			"source_player":            intValue(effect.SourcePlayer, 1),
			"item_id":                  intValue(effect.ItemID, -1),
			"operation":                intValue(effect.Operation, 1),
			"number_of_units_selected": -1,
			"selected_object_ids":      []any{},
			"message":                  "",
			"sound_name":               "",
		}, nil
	case "change_object_cost":
		if effect.ObjectListUnitID == nil {
			return nil, fmt.Errorf("change_object_cost requires object_list_unit_id")
		}
		return map[string]any{
			"effect_type":              effectTypeID("change_object_cost"),
			"source_player":            intValue(effect.SourcePlayer, 1),
			"object_list_unit_id":      *effect.ObjectListUnitID,
			"resource_1":               intValue(effect.Resource1, -1),
			"resource_1_quantity":      intValue(effect.Resource1Quantity, -1),
			"resource_2":               intValue(effect.Resource2, -1),
			"resource_2_quantity":      intValue(effect.Resource2Quantity, -1),
			"resource_3":               intValue(effect.Resource3, -1),
			"resource_3_quantity":      intValue(effect.Resource3Quantity, -1),
			"number_of_units_selected": -1,
			"selected_object_ids":      []any{},
			"message":                  "",
			"sound_name":               "",
		}, nil
	case "change_technology_cost":
		if effect.Technology == nil {
			return nil, fmt.Errorf("change_technology_cost requires technology")
		}
		return map[string]any{
			"effect_type":              effectTypeID("change_technology_cost"),
			"source_player":            intValue(effect.SourcePlayer, 1),
			"technology":               *effect.Technology,
			"resource_1":               intValue(effect.Resource1, -1),
			"resource_1_quantity":      intValue(effect.Resource1Quantity, -1),
			"resource_2":               intValue(effect.Resource2, -1),
			"resource_2_quantity":      intValue(effect.Resource2Quantity, -1),
			"resource_3":               intValue(effect.Resource3, -1),
			"resource_3_quantity":      intValue(effect.Resource3Quantity, -1),
			"number_of_units_selected": -1,
			"selected_object_ids":      []any{},
			"message":                  "",
			"sound_name":               "",
		}, nil
	case "change_technology_research_time":
		if effect.Technology == nil {
			return nil, fmt.Errorf("change_technology_research_time requires technology")
		}
		return map[string]any{
			"effect_type":              effectTypeID("change_technology_research_time"),
			"source_player":            intValue(effect.SourcePlayer, 1),
			"technology":               *effect.Technology,
			"quantity":                 intValue(effect.Quantity, intValue(effect.TrainTime, -1)),
			"number_of_units_selected": -1,
			"selected_object_ids":      []any{},
			"message":                  "",
			"sound_name":               "",
		}, nil
	case "change_technology_name", "change_technology_description":
		if effect.Technology == nil {
			return nil, fmt.Errorf("%s requires technology", effect.Op)
		}
		if effect.Message == "" {
			return nil, fmt.Errorf("%s requires message", effect.Op)
		}
		return map[string]any{
			"effect_type":              effectTypeID(effect.Op),
			"source_player":            intValue(effect.SourcePlayer, 1),
			"technology":               *effect.Technology,
			"message":                  effect.Message,
			"number_of_units_selected": -1,
			"selected_object_ids":      []any{},
			"sound_name":               "",
		}, nil
	case "change_technology_icon", "change_technology_hotkey":
		if effect.Technology == nil {
			return nil, fmt.Errorf("%s requires technology", effect.Op)
		}
		value := intValue(effect.Quantity, -1)
		if effect.Op == "change_technology_hotkey" {
			value = intValue(effect.Hotkey, value)
		}
		return map[string]any{
			"effect_type":              effectTypeID(effect.Op),
			"source_player":            intValue(effect.SourcePlayer, 1),
			"technology":               *effect.Technology,
			"quantity":                 value,
			"hotkey":                   intValue(effect.Hotkey, -1),
			"number_of_units_selected": -1,
			"selected_object_ids":      []any{},
			"message":                  "",
			"sound_name":               "",
		}, nil
	case "train_unit":
		if effect.ObjectListUnitID == nil {
			return nil, fmt.Errorf("train_unit requires object_list_unit_id")
		}
		selected := selectedObjectAnyList(effect.SelectedObjectIDs)
		return map[string]any{
			"effect_type":              effectTypeID("train_unit"),
			"object_list_unit_id":      *effect.ObjectListUnitID,
			"source_player":            intValue(effect.SourcePlayer, 1),
			"area_x1":                  intValue(effect.AreaX1, -1),
			"area_y1":                  intValue(effect.AreaY1, -1),
			"area_x2":                  intValue(effect.AreaX2, -1),
			"area_y2":                  intValue(effect.AreaY2, -1),
			"location_x":               intValue(effect.LocationX, -1),
			"location_y":               intValue(effect.LocationY, -1),
			"number_of_units_selected": selectedCount(selected),
			"selected_object_ids":      selected,
			"message":                  "",
			"sound_name":               "",
		}, nil
	case "initiate_research", "research_local_technology":
		if effect.Technology == nil {
			return nil, fmt.Errorf("%s requires technology", effect.Op)
		}
		selected := selectedObjectAnyList(effect.SelectedObjectIDs)
		out := map[string]any{
			"effect_type":              effectTypeID(effect.Op),
			"source_player":            intValue(effect.SourcePlayer, 1),
			"technology":               *effect.Technology,
			"number_of_units_selected": selectedCount(selected),
			"selected_object_ids":      selected,
			"message":                  "",
			"sound_name":               "",
		}
		if effect.LocalTechnology != nil {
			out["local_technology"] = *effect.LocalTechnology
		}
		return out, nil
	case "script_call":
		call := effect.Message
		if call == "" {
			call = effect.XSFunction
		}
		if call == "" {
			return nil, fmt.Errorf("script_call requires message or xs_function")
		}
		return map[string]any{
			"effect_type":              effectTypeID("script_call"),
			"source_player":            intValue(effect.SourcePlayer, 1),
			"number_of_units_selected": -1,
			"selected_object_ids":      []any{},
			"message":                  call,
			"sound_name":               "",
		}, nil
	case "change_variable", "modify_variable":
		if effect.Variable == nil {
			return nil, fmt.Errorf("%s requires variable", effect.Op)
		}
		return map[string]any{
			"effect_type":              effectTypeID("change_variable"),
			"quantity":                 intValue(effect.Quantity, 0),
			"operation":                intValue(effect.Operation, 1),
			"variable":                 *effect.Variable,
			"message":                  effect.Message,
			"number_of_units_selected": -1,
			"selected_object_ids":      []any{},
			"sound_name":               "",
		}, nil
	case "modify_variable_by_variable":
		if effect.Variable == nil {
			return nil, fmt.Errorf("modify_variable_by_variable requires variable")
		}
		if effect.Variable2 == nil {
			return nil, fmt.Errorf("modify_variable_by_variable requires variable2")
		}
		return map[string]any{
			"effect_type":              effectTypeID("modify_variable_by_variable"),
			"operation":                intValue(effect.Operation, 1),
			"variable":                 *effect.Variable,
			"variable2":                *effect.Variable2,
			"number_of_units_selected": -1,
			"selected_object_ids":      []any{},
			"message":                  "",
			"sound_name":               "",
		}, nil
	case "clear_timer":
		if effect.TimerID == nil {
			return nil, fmt.Errorf("clear_timer requires timer_id")
		}
		return map[string]any{
			"effect_type":              effectTypeID("clear_timer"),
			"timer":                    *effect.TimerID,
			"number_of_units_selected": -1,
			"selected_object_ids":      []any{},
			"message":                  "",
			"sound_name":               "",
		}, nil
	case "activate_trigger":
		if effect.TriggerID == nil {
			return nil, fmt.Errorf("activate_trigger requires trigger_id")
		}
		return map[string]any{
			"effect_type": effectTypeID("activate_trigger"),
			"trigger_id":  *effect.TriggerID,
		}, nil
	case "deactivate_trigger":
		if effect.TriggerID == nil {
			return nil, fmt.Errorf("deactivate_trigger requires trigger_id")
		}
		return map[string]any{
			"effect_type": effectTypeID("deactivate_trigger"),
			"trigger_id":  *effect.TriggerID,
		}, nil
	case "declare_victory":
		return map[string]any{
			"effect_type":              effectTypeID("declare_victory"),
			"source_player":            intValue(effect.SourcePlayer, 1),
			"enabled":                  intValue(effect.Enabled, 1),
			"number_of_units_selected": -1,
			"selected_object_ids":      []any{},
		}, nil
	default:
		return nil, fmt.Errorf("unsupported effect op %q", effect.Op)
	}
}

func resolveTriggerOperation(numeric *int, named string) (int, error) {
	if strings.TrimSpace(named) != "" {
		if numeric != nil {
			return 0, fmt.Errorf("trigger operation must use either operation or operation_name, not both")
		}
		switch strings.ToLower(strings.TrimSpace(named)) {
		case "set":
			return 1, nil
		case "add":
			return 2, nil
		case "subtract", "sub":
			return 3, nil
		case "multiply", "mul":
			return 4, nil
		case "divide", "div":
			return 5, nil
		default:
			return 0, fmt.Errorf("modify_attribute operation_name %q must be set, add, subtract, multiply, or divide", named)
		}
	}
	operation := intValue(numeric, 1)
	if operation < 1 || operation > 5 {
		return 0, fmt.Errorf("modify_attribute operation %d is invalid; trigger operations are SET=1, ADD=2, SUBTRACT=3, MULTIPLY=4, DIVIDE=5", operation)
	}
	return operation, nil
}

func effectTypeID(op string) int {
	id, ok := EffectTypeForOp(op)
	if !ok {
		return -1
	}
	return id
}

func conditionOverrides(condition ConditionRecipe) (map[string]any, error) {
	switch condition.Op {
	case "chance":
		if condition.Quantity == nil {
			return nil, fmt.Errorf("chance requires quantity")
		}
		return map[string]any{"condition_type": 20, "quantity": *condition.Quantity, "inverted": intValue(condition.Inverted, 0)}, nil
	case "destroy_object":
		if condition.UnitObject == nil {
			return nil, fmt.Errorf("destroy_object requires unit_object")
		}
		return map[string]any{"condition_type": 6, "unit_object": *condition.UnitObject, "inverted": intValue(condition.Inverted, 0)}, nil
	case "object_has_action":
		if condition.UnitObject == nil {
			return nil, fmt.Errorf("object_has_action requires unit_object")
		}
		if condition.UnitAIAction == nil {
			return nil, fmt.Errorf("object_has_action requires unit_ai_action")
		}
		return map[string]any{"condition_type": 28, "unit_object": *condition.UnitObject, "unit_ai_action": *condition.UnitAIAction, "inverted": intValue(condition.Inverted, 0)}, nil
	case "timer":
		if condition.Timer == nil {
			return nil, fmt.Errorf("timer condition requires timer")
		}
		return map[string]any{
			"condition_type": 10,
			"timer":          *condition.Timer,
			"inverted":       intValue(condition.Inverted, 0),
		}, nil
	case "object_selected":
		if condition.UnitObject == nil {
			return nil, fmt.Errorf("object_selected condition requires unit_object")
		}
		return map[string]any{
			"condition_type": 11,
			"unit_object":    *condition.UnitObject,
			"inverted":       intValue(condition.Inverted, 0),
		}, nil
	case "object_in_area", "objects_in_area":
		if condition.AreaX1 == nil || condition.AreaY1 == nil || condition.AreaX2 == nil || condition.AreaY2 == nil {
			return nil, fmt.Errorf("object_in_area requires area_x1, area_y1, area_x2, and area_y2")
		}
		return map[string]any{
			"condition_type":                    5,
			"quantity":                          intValue(condition.Quantity, 1),
			"object_list":                       intValue(condition.ObjectList, -1),
			"source_player":                     intValue(condition.SourcePlayer, 1),
			"area_x1":                           *condition.AreaX1,
			"area_y1":                           *condition.AreaY1,
			"area_x2":                           *condition.AreaX2,
			"area_y2":                           *condition.AreaY2,
			"object_group":                      intValue(condition.ObjectGroup, -1),
			"object_type":                       intValue(condition.ObjectType, -1),
			"object_state":                      intValue(condition.ObjectState, -1),
			"include_changeable_weapon_objects": intValue(condition.IncludeChangeableWeaponObjects, -1),
			"inverted":                          intValue(condition.Inverted, 0),
		}, nil
	case "own_objects", "own_fewer_objects":
		conditionType := 3
		if condition.Op == "own_fewer_objects" {
			conditionType = 4
		}
		return map[string]any{
			"condition_type":                    conditionType,
			"quantity":                          intValue(condition.Quantity, 1),
			"object_list":                       intValue(condition.ObjectList, -1),
			"source_player":                     intValue(condition.SourcePlayer, 1),
			"area_x1":                           intValue(condition.AreaX1, -1),
			"area_y1":                           intValue(condition.AreaY1, -1),
			"area_x2":                           intValue(condition.AreaX2, -1),
			"area_y2":                           intValue(condition.AreaY2, -1),
			"object_group":                      intValue(condition.ObjectGroup, -1),
			"object_type":                       intValue(condition.ObjectType, -1),
			"object_state":                      intValue(condition.ObjectState, -1),
			"include_changeable_weapon_objects": intValue(condition.IncludeChangeableWeaponObjects, -1),
			"inverted":                          intValue(condition.Inverted, 0),
		}, nil
	case "accumulate_attribute":
		if condition.Attribute == nil {
			return nil, fmt.Errorf("accumulate_attribute requires attribute")
		}
		return map[string]any{
			"condition_type": 8,
			"quantity":       intValue(condition.Quantity, 0),
			"attribute":      *condition.Attribute,
			"source_player":  intValue(condition.SourcePlayer, 1),
			"inverted":       intValue(condition.Inverted, 0),
		}, nil
	case "object_visible":
		if condition.UnitObject == nil {
			return nil, fmt.Errorf("object_visible condition requires unit_object")
		}
		return map[string]any{
			"condition_type": 15,
			"unit_object":    *condition.UnitObject,
			"inverted":       intValue(condition.Inverted, 0),
		}, nil
	case "variable_value":
		if condition.Variable == nil {
			return nil, fmt.Errorf("variable_value requires variable")
		}
		if condition.Comparison == nil {
			return nil, fmt.Errorf("variable_value requires comparison")
		}
		return map[string]any{
			"condition_type": 22,
			"quantity":       intValue(condition.Quantity, 0),
			"variable":       *condition.Variable,
			"comparison":     *condition.Comparison,
			"inverted":       intValue(condition.Inverted, 0),
		}, nil
	default:
		return nil, fmt.Errorf("unsupported condition op %q", condition.Op)
	}
}

func selectedObjectAnyList(ids []int) []any {
	out := make([]any, 0, len(ids))
	for _, id := range ids {
		out = append(out, id)
	}
	return out
}

func selectedCount(ids []any) int {
	if len(ids) == 0 {
		return -1
	}
	return len(ids)
}

func triggerEffectSpecs(spec *Spec) (SectionSpec, SectionSpec, error) {
	triggerSpec, effectSpec, _, err := triggerChildSpecs(spec)
	return triggerSpec, effectSpec, err
}

func unitStructSpec(spec *Spec) (SectionSpec, error) {
	units, ok := spec.section("Units")
	if !ok {
		return SectionSpec{}, fmt.Errorf("missing Units section")
	}
	playerUnits, ok := units.Structs["PlayerUnitsStruct"]
	if !ok {
		return SectionSpec{}, fmt.Errorf("missing PlayerUnitsStruct")
	}
	unitSpec, ok := playerUnits.Structs["UnitStruct"]
	if !ok {
		return SectionSpec{}, fmt.Errorf("missing UnitStruct")
	}
	return unitSpec, nil
}

func triggerChildSpecs(spec *Spec) (SectionSpec, SectionSpec, SectionSpec, error) {
	triggers, ok := spec.section("Triggers")
	if !ok {
		return SectionSpec{}, SectionSpec{}, SectionSpec{}, fmt.Errorf("missing Triggers section")
	}
	triggerSpec, ok := triggers.Structs["TriggerStruct"]
	if !ok {
		return SectionSpec{}, SectionSpec{}, SectionSpec{}, fmt.Errorf("missing TriggerStruct")
	}
	effectSpec, ok := triggerSpec.Structs["EffectStruct"]
	if !ok {
		return SectionSpec{}, SectionSpec{}, SectionSpec{}, fmt.Errorf("missing EffectStruct")
	}
	conditionSpec, ok := triggerSpec.Structs["ConditionStruct"]
	if !ok {
		return SectionSpec{}, SectionSpec{}, SectionSpec{}, fmt.Errorf("missing ConditionStruct")
	}
	return triggerSpec, effectSpec, conditionSpec, nil
}

func indexAnyList(count int) []any {
	out := make([]any, 0, count)
	for i := 0; i < count; i++ {
		out = append(out, i)
	}
	return out
}

func indexUint32List(count int) []uint32 {
	out := make([]uint32, 0, count)
	for i := 0; i < count; i++ {
		out = append(out, uint32(i))
	}
	return out
}

func buildStructRaw(spec SectionSpec, overrides map[string]any) ([]byte, error) {
	var out bytes.Buffer
	for _, field := range spec.Fields {
		value, ok := overrides[field.Name]
		if !ok {
			value = fieldDefault(field)
		}
		if stringsHasStructPrefix(field.Type) {
			chunks, _ := value.([][]byte)
			for _, chunk := range chunks {
				out.Write(chunk)
			}
			continue
		}
		chunk, err := encodeRetrieverValue(field, value)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", field.Name, err)
		}
		out.Write(chunk)
	}
	return out.Bytes(), nil
}

func encodeRetrieverValue(field FieldSpec, value any) ([]byte, error) {
	var out bytes.Buffer
	if values, ok := value.([]any); ok {
		for _, item := range values {
			chunk, err := encodePrimitive(field.Type, item)
			if err != nil {
				return nil, err
			}
			out.Write(chunk)
		}
		return out.Bytes(), nil
	}
	return encodePrimitive(field.Type, value)
}

func fieldDefault(field FieldSpec) any {
	if len(field.Default) == 0 {
		return nil
	}
	var value any
	if err := json.Unmarshal(field.Default, &value); err != nil {
		return nil
	}
	return value
}

func setIntField(node *parsedNode, name, kind string, value int) error {
	field := node.field(name)
	if field == nil {
		return fmt.Errorf("missing %s", name)
	}
	raw, err := encodePrimitive(kind, value)
	if err != nil {
		return err
	}
	field.Raw = raw
	switch kind[0] {
	case 'u':
		field.Value = uint32(value)
	default:
		field.Value = int32(value)
	}
	return nil
}

func setBoolByteField(node *parsedNode, name string, value bool) error {
	intValue := 0
	if value {
		intValue = 1
	}
	return setIntField(node, name, "u8", intValue)
}

func setFloatField(node *parsedNode, name, kind string, value float64) error {
	field := node.field(name)
	if field == nil {
		return fmt.Errorf("missing field %q", name)
	}
	if kind != "f32" {
		return fmt.Errorf("unsupported float field kind %q", kind)
	}
	var raw [4]byte
	binary.LittleEndian.PutUint32(raw[:], math.Float32bits(float32(value)))
	field.Raw = raw[:]
	field.Value = float32(value)
	return nil
}

func setUint32ListField(node *parsedNode, name string, values []uint32) error {
	field := node.field(name)
	if field == nil {
		return fmt.Errorf("missing %s", name)
	}
	var raw []byte
	list := make([]any, 0, len(values))
	for _, value := range values {
		chunk, err := encodePrimitive("u32", value)
		if err != nil {
			return err
		}
		raw = append(raw, chunk...)
		list = append(list, value)
	}
	field.Raw = raw
	field.Value = list
	return nil
}

func setIntListField(node *parsedNode, name, kind string, values []int) error {
	field := node.field(name)
	if field == nil {
		return fmt.Errorf("missing %s", name)
	}
	var raw []byte
	list := make([]any, 0, len(values))
	for _, value := range values {
		chunk, err := encodePrimitive(kind, value)
		if err != nil {
			return err
		}
		raw = append(raw, chunk...)
		list = append(list, int32(value))
	}
	field.Raw = raw
	field.Value = list
	return nil
}

func setStringField(node *parsedNode, name, kind, value string) error {
	field := node.field(name)
	if field == nil {
		return fmt.Errorf("missing %s", name)
	}
	raw, err := encodePrimitive(kind, value)
	if err != nil {
		return err
	}
	field.Raw = raw
	field.Value = value
	return nil
}

func setZeroLengthStringField(node *parsedNode, name, kind string) error {
	field := node.field(name)
	if field == nil {
		return fmt.Errorf("missing %s", name)
	}
	typ, size, err := primitiveType(kind)
	if err != nil {
		return err
	}
	if typ != "str" {
		return fmt.Errorf("%s is %s, want length-prefixed string", name, kind)
	}
	field.Raw = make([]byte, size)
	field.Value = ""
	return nil
}

func setStringListField(node *parsedNode, name, kind string, values []string) error {
	field := node.field(name)
	if field == nil {
		return fmt.Errorf("missing %s", name)
	}
	var raw []byte
	list := make([]any, 0, len(values))
	for _, value := range values {
		chunk, err := encodePrimitive(kind, value)
		if err != nil {
			return err
		}
		raw = append(raw, chunk...)
		list = append(list, value)
	}
	field.Raw = raw
	field.Value = list
	return nil
}

func indexIntList(count int) []int {
	out := make([]int, 0, count)
	for i := 0; i < count; i++ {
		out = append(out, i)
	}
	return out
}

func encodePrimitive(kind string, value any) ([]byte, error) {
	typ, size, err := primitiveType(kind)
	if err != nil {
		return nil, err
	}
	if typ == "str" {
		text, _ := value.(string)
		text = trimAtNUL(text)
		payload := append([]byte(text), 0)
		var out bytes.Buffer
		if err := writeSignedInt(&out, size, len(payload)); err != nil {
			return nil, err
		}
		out.Write(payload)
		return out.Bytes(), nil
	}
	if typ == "c" {
		out := make([]byte, size)
		copy(out, []byte(trimAtNUL(fmt.Sprint(value))))
		return out, nil
	}
	if typ == "data" {
		out := make([]byte, size)
		switch v := value.(type) {
		case string:
			if decoded, err := hex.DecodeString(v); err == nil {
				copy(out, decoded)
			} else {
				copy(out, []byte(v))
			}
		case []byte:
			copy(out, v)
		}
		return out, nil
	}
	out := make([]byte, size)
	switch typ {
	case "s":
		writeSignedBytes(out, int64(asInt(value)))
	case "u":
		writeUnsignedBytes(out, uint64(asInt(value)))
	case "f":
		switch size {
		case 4:
			binary.LittleEndian.PutUint32(out, math.Float32bits(float32(asFloat(value))))
		case 8:
			binary.LittleEndian.PutUint64(out, math.Float64bits(asFloat(value)))
		default:
			return nil, fmt.Errorf("unsupported float size %d", size)
		}
	default:
		return nil, fmt.Errorf("unsupported primitive type %q", typ)
	}
	return out, nil
}

func writeSignedInt(out *bytes.Buffer, size int, value int) error {
	buf := make([]byte, size)
	writeSignedBytes(buf, int64(value))
	_, err := out.Write(buf)
	return err
}

func writeSignedBytes(out []byte, value int64) {
	switch len(out) {
	case 1:
		out[0] = byte(int8(value))
	case 2:
		binary.LittleEndian.PutUint16(out, uint16(int16(value)))
	case 4:
		binary.LittleEndian.PutUint32(out, uint32(int32(value)))
	case 8:
		binary.LittleEndian.PutUint64(out, uint64(value))
	}
}

func writeUnsignedBytes(out []byte, value uint64) {
	switch len(out) {
	case 1:
		out[0] = byte(value)
	case 2:
		binary.LittleEndian.PutUint16(out, uint16(value))
	case 4:
		binary.LittleEndian.PutUint32(out, uint32(value))
	case 8:
		binary.LittleEndian.PutUint64(out, value)
	}
}

func stringsHasStructPrefix(value string) bool {
	return len(value) >= len("struct:") && value[:len("struct:")] == "struct:"
}
