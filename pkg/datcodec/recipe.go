package datcodec

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"aoe2kit/pkg/aoe2"
	"aoe2kit/pkg/datfile"
)

type Recipe struct {
	CreateAbility       *AbilityCreateRecipe            `json:"create_ability,omitempty"`
	CreateAbilities     []AbilityCreateRecipe           `json:"create_abilities,omitempty"`
	DisableAbility      *int                            `json:"disable_ability,omitempty"`
	DisableAbilities    []int                           `json:"disable_abilities,omitempty"`
	DeleteAbility       *int                            `json:"delete_ability,omitempty"`
	DeleteAbilities     []int                           `json:"delete_abilities,omitempty"`
	CreateEffect        *EffectCreateRecipe             `json:"create_effect,omitempty"`
	CreateEffects       []EffectCreateRecipe            `json:"create_effects,omitempty"`
	DisableEffect       *int                            `json:"disable_effect,omitempty"`
	DisableEffects      []int                           `json:"disable_effects,omitempty"`
	Effects             []EffectPatchRecipe             `json:"effects,omitempty"`
	DeleteEffects       []int                           `json:"delete_effects,omitempty"`
	CreateGraphic       *datfile.GraphicCreateRecipe    `json:"create_graphic,omitempty"`
	CreateGraphics      []datfile.GraphicCreateRecipe   `json:"create_graphics,omitempty"`
	DeleteGraphics      []int                           `json:"delete_graphics,omitempty"`
	Graphics            []datfile.GraphicRecipePatch    `json:"graphics,omitempty"`
	UnitHeaders         []UnitHeaderPatchRecipe         `json:"unit_headers,omitempty"`
	CreateUnit          *datfile.UnitCreateRecipe       `json:"create_unit,omitempty"`
	CreateUnits         []datfile.UnitCreateRecipe      `json:"create_units,omitempty"`
	Units               []datfile.UnitRecipePatch       `json:"units,omitempty"`
	Civs                []CivPatchRecipe                `json:"civs,omitempty"`
	TerrainRestrictions []TerrainRestrictionPatchRecipe `json:"terrain_restrictions,omitempty"`
	Terrains            []TerrainPatchRecipe            `json:"terrains,omitempty"`
	CreateSound         *SoundCreateRecipe              `json:"create_sound,omitempty"`
	CreateSounds        []SoundCreateRecipe             `json:"create_sounds,omitempty"`
	Sounds              []SoundPatchRecipe              `json:"sounds,omitempty"`
	DeleteSounds        []int                           `json:"delete_sounds,omitempty"`
	DeleteUnits         []UnitDeleteRecipe              `json:"delete_units,omitempty"`
	UnitAvailability    []UnitAvailabilityRecipe        `json:"unit_availability,omitempty"`
	CreatePlayerColour  *PlayerColourCreateRecipe       `json:"create_player_colour,omitempty"`
	CreatePlayerColours []PlayerColourCreateRecipe      `json:"create_player_colours,omitempty"`
	PlayerColours       []PlayerColourPatchRecipe       `json:"player_colours,omitempty"`
	DeletePlayerColour  *int                            `json:"delete_player_colour,omitempty"`
	DeletePlayerColours []int                           `json:"delete_player_colours,omitempty"`
	CreateTech          *TechCreateRecipe               `json:"create_tech,omitempty"`
	CreateTechs         []TechCreateRecipe              `json:"create_techs,omitempty"`
	Techs               []TechPatchRecipe               `json:"techs,omitempty"`
	DeleteTechs         []int                           `json:"delete_techs,omitempty"`
	DisconnectTechs     []int                           `json:"disconnect_techs,omitempty"`
	DisconnectUnits     []int                           `json:"disconnect_units,omitempty"`
	TechTree            *TechTreePatchRecipe            `json:"tech_tree,omitempty"`
	ReferenceRewrites   []TechTreeIDRewriteRecipe       `json:"reference_rewrites,omitempty"`
}

func (recipe Recipe) Empty() bool {
	return recipe.CreateAbility == nil && len(recipe.CreateAbilities) == 0 &&
		recipe.DisableAbility == nil && len(recipe.DisableAbilities) == 0 &&
		recipe.DeleteAbility == nil && len(recipe.DeleteAbilities) == 0 &&
		recipe.CreateEffect == nil && len(recipe.CreateEffects) == 0 &&
		recipe.DisableEffect == nil && len(recipe.DisableEffects) == 0 && len(recipe.Effects) == 0 &&
		len(recipe.DeleteEffects) == 0 && recipe.CreateGraphic == nil && len(recipe.CreateGraphics) == 0 &&
		len(recipe.DeleteGraphics) == 0 && len(recipe.Graphics) == 0 &&
		len(recipe.UnitHeaders) == 0 && recipe.CreateUnit == nil && len(recipe.CreateUnits) == 0 && len(recipe.Units) == 0 &&
		len(recipe.Civs) == 0 && len(recipe.TerrainRestrictions) == 0 &&
		len(recipe.Terrains) == 0 && recipe.CreateSound == nil && len(recipe.CreateSounds) == 0 &&
		len(recipe.Sounds) == 0 && len(recipe.DeleteSounds) == 0 && len(recipe.DeleteUnits) == 0 &&
		len(recipe.UnitAvailability) == 0 && recipe.CreatePlayerColour == nil && len(recipe.CreatePlayerColours) == 0 &&
		len(recipe.PlayerColours) == 0 && recipe.DeletePlayerColour == nil && len(recipe.DeletePlayerColours) == 0 &&
		recipe.CreateTech == nil && len(recipe.CreateTechs) == 0 &&
		len(recipe.Techs) == 0 && len(recipe.DeleteTechs) == 0 && recipe.TechTree == nil &&
		len(recipe.DisconnectTechs) == 0 && len(recipe.DisconnectUnits) == 0 && len(recipe.ReferenceRewrites) == 0
}

type AbilityCreateRecipe struct {
	Name                   string                         `json:"name"`
	EffectName             string                         `json:"effect_name,omitempty"`
	FromTech               int                            `json:"from_tech"`
	FromEffect             *int                           `json:"from_effect,omitempty"`
	Commands               []EffectCommandRecipe          `json:"commands,omitempty"`
	AppendCommands         []EffectCommandRecipe          `json:"append_commands,omitempty"`
	RequiredTechs          []int16                        `json:"required_techs,omitempty"`
	ResourceCosts          []datfile.ResearchResourceCost `json:"resource_costs,omitempty"`
	RequiredTechCount      *int16                         `json:"required_tech_count,omitempty"`
	Civ                    *int16                         `json:"civ,omitempty"`
	FullTechMode           *int16                         `json:"full_tech_mode,omitempty"`
	LanguageDLLName        *int32                         `json:"language_dll_name,omitempty"`
	LanguageDLLDescription *int32                         `json:"language_dll_description,omitempty"`
	Type                   *int16                         `json:"type,omitempty"`
	IconID                 *int16                         `json:"icon_id,omitempty"`
	LanguageDLLHelp        *int32                         `json:"language_dll_help,omitempty"`
	LanguageDLLTechTree    *int32                         `json:"language_dll_tech_tree,omitempty"`
	Repeatable             *uint8                         `json:"repeatable,omitempty"`
	ResearchLocations      []datfile.ResearchLocation     `json:"research_locations,omitempty"`
}

type EffectCreateRecipe struct {
	Name           string                `json:"name"`
	FromEffect     *int                  `json:"from_effect,omitempty"`
	Commands       []EffectCommandRecipe `json:"commands,omitempty"`
	RemoveCommands []int                 `json:"remove_commands,omitempty"`
	AppendCommands []EffectCommandRecipe `json:"append_commands,omitempty"`
}

type EffectPatchRecipe struct {
	ID             int                    `json:"id"`
	Name           *string                `json:"name,omitempty"`
	Commands       *[]EffectCommandRecipe `json:"commands,omitempty"`
	RemoveCommands []int                  `json:"remove_commands,omitempty"`
	AppendCommands []EffectCommandRecipe  `json:"append_commands,omitempty"`
}

type EffectCommandRecipe struct {
	Type              uint8    `json:"type"`
	A                 int16    `json:"a"`
	B                 int16    `json:"b"`
	C                 int16    `json:"c"`
	D                 float32  `json:"d"`
	Kind              string   `json:"kind,omitempty"`
	UnitID            *int16   `json:"unit_id,omitempty"`
	ToUnitID          *int16   `json:"to_unit_id,omitempty"`
	BuildingID        *int16   `json:"building_id,omitempty"`
	UnitClassID       *int16   `json:"unit_class_id,omitempty"`
	UnitClass         string   `json:"unit_class,omitempty"`
	UnitClassName     string   `json:"unit_class_name,omitempty"`
	AttributeID       *int16   `json:"attribute_id,omitempty"`
	Attribute         string   `json:"attribute,omitempty"`
	AttributeName     string   `json:"attribute_name,omitempty"`
	PackedTypeID      *int     `json:"packed_type_id,omitempty"`
	PackedType        string   `json:"packed_type,omitempty"`
	PackedTypeName    string   `json:"packed_type_name,omitempty"`
	PackedAmount      *int     `json:"packed_amount,omitempty"`
	ArmorClassID      *int     `json:"armor_class_id,omitempty"`
	ArmorClass        string   `json:"armor_class,omitempty"`
	AttackClassID     *int     `json:"attack_class_id,omitempty"`
	AttackClass       string   `json:"attack_class,omitempty"`
	CombatClassID     *int     `json:"combat_class_id,omitempty"`
	CombatClass       string   `json:"combat_class,omitempty"`
	ResourceID        *int16   `json:"resource_id,omitempty"`
	Resource          string   `json:"resource,omitempty"`
	ResourceName      string   `json:"resource_name,omitempty"`
	OperationID       *int16   `json:"operation_id,omitempty"`
	Operation         string   `json:"operation,omitempty"`
	OperationName     string   `json:"operation_name,omitempty"`
	TechAttrID        *int16   `json:"tech_attribute_id,omitempty"`
	TechAttribute     string   `json:"tech_attribute,omitempty"`
	TechAttributeName string   `json:"tech_attribute_name,omitempty"`
	Amount            *float32 `json:"amount,omitempty"`
	TechID            *int     `json:"tech_id,omitempty"`
}

type UnitHeaderPatchRecipe struct {
	ID       int                 `json:"id"`
	Tasks    []datfile.TaskPatch `json:"tasks,omitempty"`
	SetTasks *[]datfile.TaskRow  `json:"set_tasks,omitempty"`
}

type CivPatchRecipe struct {
	ID          int                `json:"id"`
	Name        *string            `json:"name,omitempty"`
	TechTreeID  *int16             `json:"tech_tree_id,omitempty"`
	TeamBonusID *int16             `json:"team_bonus_id,omitempty"`
	IconSet     *uint8             `json:"icon_set,omitempty"`
	Resources   []CivResourcePatch `json:"resources,omitempty"`
}

type CivResourcePatch struct {
	Index int     `json:"index"`
	Value float32 `json:"value"`
}

type TerrainRestrictionPatchRecipe struct {
	ID   int                       `json:"id"`
	Rows []TerrainPassabilityPatch `json:"rows,omitempty"`
}

type TerrainPassabilityPatch struct {
	TerrainID   int     `json:"terrain_id"`
	Passability float32 `json:"passability"`
}

type TerrainPatchRecipe struct {
	ID              int     `json:"id"`
	Name            *string `json:"name,omitempty"`
	Name2           *string `json:"name_2,omitempty"`
	OverlayMaskName *string `json:"overlay_mask_name,omitempty"`
}

type SoundPatchRecipe struct {
	ID               int                    `json:"id"`
	SoundID          *int16                 `json:"sound_id,omitempty"`
	PlayDelay        *int16                 `json:"play_delay,omitempty"`
	CacheTime        *int32                 `json:"cache_time,omitempty"`
	TotalProbability *int16                 `json:"total_probability,omitempty"`
	RemoveItems      []int                  `json:"remove_items,omitempty"`
	Items            []SoundItemPatchRecipe `json:"items,omitempty"`
}

type SoundCreateRecipe struct {
	From             int                `json:"from"`
	SoundID          *int16             `json:"sound_id,omitempty"`
	PlayDelay        *int16             `json:"play_delay,omitempty"`
	CacheTime        *int32             `json:"cache_time,omitempty"`
	TotalProbability *int16             `json:"total_probability,omitempty"`
	Items            *[]SoundItemRecipe `json:"items,omitempty"`
}

type SoundItemRecipe struct {
	FileName    string `json:"file_name"`
	ResourceID  int32  `json:"resource_id"`
	Probability int16  `json:"probability"`
	Civ         int16  `json:"civ"`
	IconSet     int16  `json:"icon_set"`
}

type SoundItemPatchRecipe struct {
	Index       int     `json:"index"`
	FileName    *string `json:"file_name,omitempty"`
	ResourceID  *int32  `json:"resource_id,omitempty"`
	Probability *int16  `json:"probability,omitempty"`
	Civ         *int16  `json:"civ,omitempty"`
	IconSet     *int16  `json:"icon_set,omitempty"`
}

type PlayerColourPatchRecipe struct {
	ID                  int    `json:"id"`
	ColourID            *int32 `json:"colour_id,omitempty"`
	Base                *int32 `json:"base,omitempty"`
	UnitOutlineColour   *int32 `json:"unit_outline_colour,omitempty"`
	SelectionColour1    *int32 `json:"selection_colour_1,omitempty"`
	SelectionColour2    *int32 `json:"selection_colour_2,omitempty"`
	MinimapColour1      *int32 `json:"minimap_colour_1,omitempty"`
	MinimapColour2      *int32 `json:"minimap_colour_2,omitempty"`
	MinimapColour3      *int32 `json:"minimap_colour_3,omitempty"`
	StatisticsTextColor *int32 `json:"statistics_text_color,omitempty"`
}

type PlayerColourCreateRecipe struct {
	From                int    `json:"from"`
	ColourID            *int32 `json:"colour_id,omitempty"`
	Base                *int32 `json:"base,omitempty"`
	UnitOutlineColour   *int32 `json:"unit_outline_colour,omitempty"`
	SelectionColour1    *int32 `json:"selection_colour_1,omitempty"`
	SelectionColour2    *int32 `json:"selection_colour_2,omitempty"`
	MinimapColour1      *int32 `json:"minimap_colour_1,omitempty"`
	MinimapColour2      *int32 `json:"minimap_colour_2,omitempty"`
	MinimapColour3      *int32 `json:"minimap_colour_3,omitempty"`
	StatisticsTextColor *int32 `json:"statistics_text_color,omitempty"`
}

type UnitDeleteRecipe struct {
	CivID    *int   `json:"civ_id,omitempty"`
	UnitID   int    `json:"unit_id"`
	AllCivs  bool   `json:"all_civs,omitempty"`
	Strategy string `json:"strategy,omitempty"`
}

type UnitAvailabilityRecipe struct {
	CivID   *int  `json:"civ_id,omitempty"`
	CivIDs  []int `json:"civ_ids,omitempty"`
	UnitID  int   `json:"unit_id"`
	AllCivs bool  `json:"all_civs,omitempty"`
	Enabled bool  `json:"enabled"`
}

type TechCreateRecipe struct {
	From                   int                            `json:"from"`
	Name                   *string                        `json:"name,omitempty"`
	RequiredTechs          []int16                        `json:"required_techs,omitempty"`
	ResourceCosts          []datfile.ResearchResourceCost `json:"resource_costs,omitempty"`
	RequiredTechCount      *int16                         `json:"required_tech_count,omitempty"`
	Civ                    *int16                         `json:"civ,omitempty"`
	FullTechMode           *int16                         `json:"full_tech_mode,omitempty"`
	LanguageDLLName        *int32                         `json:"language_dll_name,omitempty"`
	LanguageDLLDescription *int32                         `json:"language_dll_description,omitempty"`
	EffectID               *int16                         `json:"effect_id,omitempty"`
	Type                   *int16                         `json:"type,omitempty"`
	IconID                 *int16                         `json:"icon_id,omitempty"`
	LanguageDLLHelp        *int32                         `json:"language_dll_help,omitempty"`
	LanguageDLLTechTree    *int32                         `json:"language_dll_tech_tree,omitempty"`
	Repeatable             *uint8                         `json:"repeatable,omitempty"`
	ResearchLocations      []datfile.ResearchLocation     `json:"research_locations,omitempty"`
}

type TechPatchRecipe struct {
	ID                     int                            `json:"id"`
	Name                   *string                        `json:"name,omitempty"`
	RequiredTechs          []int16                        `json:"required_techs,omitempty"`
	ResourceCosts          []datfile.ResearchResourceCost `json:"resource_costs,omitempty"`
	RequiredTechCount      *int16                         `json:"required_tech_count,omitempty"`
	Civ                    *int16                         `json:"civ,omitempty"`
	FullTechMode           *int16                         `json:"full_tech_mode,omitempty"`
	LanguageDLLName        *int32                         `json:"language_dll_name,omitempty"`
	LanguageDLLDescription *int32                         `json:"language_dll_description,omitempty"`
	EffectID               *int16                         `json:"effect_id,omitempty"`
	Type                   *int16                         `json:"type,omitempty"`
	IconID                 *int16                         `json:"icon_id,omitempty"`
	LanguageDLLHelp        *int32                         `json:"language_dll_help,omitempty"`
	LanguageDLLTechTree    *int32                         `json:"language_dll_tech_tree,omitempty"`
	Repeatable             *uint8                         `json:"repeatable,omitempty"`
	ResearchLocations      []datfile.ResearchLocation     `json:"research_locations,omitempty"`
}

type TechTreePatchRecipe struct {
	CreateBuildingConnections []TechTreeBuildingConnectionCreate `json:"create_building_connections,omitempty"`
	CreateUnitConnections     []TechTreeUnitConnectionCreate     `json:"create_unit_connections,omitempty"`
	CreateResearchConnections []TechTreeResearchConnectionCreate `json:"create_research_connections,omitempty"`
	DeleteBuildingConnections []int                              `json:"delete_building_connections,omitempty"`
	DeleteUnitConnections     []int                              `json:"delete_unit_connections,omitempty"`
	DeleteResearchConnections []int                              `json:"delete_research_connections,omitempty"`
	BuildingConnections       []TechTreeBuildingConnectionPatch  `json:"building_connections,omitempty"`
	UnitConnections           []TechTreeUnitConnectionPatch      `json:"unit_connections,omitempty"`
	ResearchConnections       []TechTreeResearchConnectionPatch  `json:"research_connections,omitempty"`
	Rewrites                  []TechTreeIDRewriteRecipe          `json:"rewrites,omitempty"`
}

type TechTreeIDRewriteRecipe struct {
	Target string `json:"target"`
	From   int32  `json:"from"`
	To     *int32 `json:"to,omitempty"`
	Remove bool   `json:"remove,omitempty"`
}

type TechTreeBuildingConnectionPatch struct {
	Index            int      `json:"index"`
	ID               *int32   `json:"id,omitempty"`
	Status           *uint8   `json:"status,omitempty"`
	Buildings        *[]int32 `json:"buildings,omitempty"`
	Units            *[]int32 `json:"units,omitempty"`
	Techs            *[]int32 `json:"techs,omitempty"`
	UnitResearch     *[]int32 `json:"unit_research,omitempty"`
	Mode             *[]int32 `json:"mode,omitempty"`
	LocationInAge    *uint8   `json:"location_in_age,omitempty"`
	UnitsTechsTotal  *[]uint8 `json:"units_techs_total,omitempty"`
	UnitsTechsFirst  *[]uint8 `json:"units_techs_first,omitempty"`
	LineMode         *int32   `json:"line_mode,omitempty"`
	EnablingResearch *int32   `json:"enabling_research,omitempty"`
}

type TechTreeBuildingConnectionCreate struct {
	From             int      `json:"from"`
	ID               *int32   `json:"id,omitempty"`
	Status           *uint8   `json:"status,omitempty"`
	Buildings        *[]int32 `json:"buildings,omitempty"`
	Units            *[]int32 `json:"units,omitempty"`
	Techs            *[]int32 `json:"techs,omitempty"`
	UnitResearch     *[]int32 `json:"unit_research,omitempty"`
	Mode             *[]int32 `json:"mode,omitempty"`
	LocationInAge    *uint8   `json:"location_in_age,omitempty"`
	UnitsTechsTotal  *[]uint8 `json:"units_techs_total,omitempty"`
	UnitsTechsFirst  *[]uint8 `json:"units_techs_first,omitempty"`
	LineMode         *int32   `json:"line_mode,omitempty"`
	EnablingResearch *int32   `json:"enabling_research,omitempty"`
}

type TechTreeUnitConnectionPatch struct {
	Index            int      `json:"index"`
	ID               *int32   `json:"id,omitempty"`
	Status           *uint8   `json:"status,omitempty"`
	UpperBuilding    *int32   `json:"upper_building,omitempty"`
	UnitResearch     *[]int32 `json:"unit_research,omitempty"`
	Mode             *[]int32 `json:"mode,omitempty"`
	VerticalLine     *int32   `json:"vertical_line,omitempty"`
	Units            *[]int32 `json:"units,omitempty"`
	LocationInAge    *int32   `json:"location_in_age,omitempty"`
	RequiredResearch *int32   `json:"required_research,omitempty"`
	LineMode         *int32   `json:"line_mode,omitempty"`
	EnablingResearch *int32   `json:"enabling_research,omitempty"`
}

type TechTreeUnitConnectionCreate struct {
	From             int      `json:"from"`
	ID               *int32   `json:"id,omitempty"`
	Status           *uint8   `json:"status,omitempty"`
	UpperBuilding    *int32   `json:"upper_building,omitempty"`
	UnitResearch     *[]int32 `json:"unit_research,omitempty"`
	Mode             *[]int32 `json:"mode,omitempty"`
	VerticalLine     *int32   `json:"vertical_line,omitempty"`
	Units            *[]int32 `json:"units,omitempty"`
	LocationInAge    *int32   `json:"location_in_age,omitempty"`
	RequiredResearch *int32   `json:"required_research,omitempty"`
	LineMode         *int32   `json:"line_mode,omitempty"`
	EnablingResearch *int32   `json:"enabling_research,omitempty"`
}

type TechTreeResearchConnectionPatch struct {
	Index         int      `json:"index"`
	ID            *int32   `json:"id,omitempty"`
	Status        *uint8   `json:"status,omitempty"`
	UpperBuilding *int32   `json:"upper_building,omitempty"`
	Buildings     *[]int32 `json:"buildings,omitempty"`
	Units         *[]int32 `json:"units,omitempty"`
	Techs         *[]int32 `json:"techs,omitempty"`
	UnitResearch  *[]int32 `json:"unit_research,omitempty"`
	Mode          *[]int32 `json:"mode,omitempty"`
	VerticalLine  *int32   `json:"vertical_line,omitempty"`
	LocationInAge *int32   `json:"location_in_age,omitempty"`
	LineMode      *int32   `json:"line_mode,omitempty"`
}

type TechTreeResearchConnectionCreate struct {
	From          int      `json:"from"`
	ID            *int32   `json:"id,omitempty"`
	Status        *uint8   `json:"status,omitempty"`
	UpperBuilding *int32   `json:"upper_building,omitempty"`
	Buildings     *[]int32 `json:"buildings,omitempty"`
	Units         *[]int32 `json:"units,omitempty"`
	Techs         *[]int32 `json:"techs,omitempty"`
	UnitResearch  *[]int32 `json:"unit_research,omitempty"`
	Mode          *[]int32 `json:"mode,omitempty"`
	VerticalLine  *int32   `json:"vertical_line,omitempty"`
	LocationInAge *int32   `json:"location_in_age,omitempty"`
	LineMode      *int32   `json:"line_mode,omitempty"`
}

type PatchReport struct {
	Version                            string                        `json:"version"`
	InputCompressedBytes               int                           `json:"input_compressed_bytes"`
	OutputCompressedBytes              int                           `json:"output_compressed_bytes"`
	InputInflatedBytes                 int                           `json:"input_inflated_bytes"`
	OutputInflatedBytes                int                           `json:"output_inflated_bytes"`
	InflatedLengthDelta                int                           `json:"inflated_length_delta"`
	BeforeEffectCount                  int                           `json:"before_effect_count"`
	AfterEffectCount                   int                           `json:"after_effect_count"`
	BeforeGraphicCount                 int                           `json:"before_graphic_count,omitempty"`
	AfterGraphicCount                  int                           `json:"after_graphic_count,omitempty"`
	BeforeCivCount                     int                           `json:"before_civ_count"`
	AfterCivCount                      int                           `json:"after_civ_count"`
	BeforeTechCount                    int                           `json:"before_tech_count"`
	AfterTechCount                     int                           `json:"after_tech_count"`
	BeforeSoundCount                   int                           `json:"before_sound_count"`
	AfterSoundCount                    int                           `json:"after_sound_count"`
	BeforePlayerColourCount            int                           `json:"before_player_colour_count,omitempty"`
	AfterPlayerColourCount             int                           `json:"after_player_colour_count,omitempty"`
	CreatedAbilities                   []AbilityCreateReport         `json:"created_abilities,omitempty"`
	DisabledAbilities                  []AbilityDeleteReport         `json:"disabled_abilities,omitempty"`
	DeletedAbilities                   []AbilityDeleteReport         `json:"deleted_abilities,omitempty"`
	CreatedEffects                     []int                         `json:"created_effects,omitempty"`
	DisabledEffects                    []int                         `json:"disabled_effects,omitempty"`
	GraphicReports                     []datfile.PatchReport         `json:"graphic_reports,omitempty"`
	CreatedGraphics                    []datfile.GraphicCreateReport `json:"created_graphics,omitempty"`
	DeletedGraphics                    []datfile.GraphicDeleteReport `json:"deleted_graphics,omitempty"`
	CreatedUnits                       []datfile.UnitCreateReport    `json:"created_units,omitempty"`
	UnitReports                        []datfile.UnitPatchReport     `json:"unit_reports,omitempty"`
	CreatedTechs                       []int                         `json:"created_techs,omitempty"`
	CreatedPlayerColours               []int                         `json:"created_player_colours,omitempty"`
	DeletedPlayerColours               []PlayerColourDeleteReport    `json:"deleted_player_colours,omitempty"`
	CreatedSounds                      []int                         `json:"created_sounds,omitempty"`
	DeletedEffects                     []int                         `json:"deleted_effects,omitempty"`
	DeletedTechs                       []int                         `json:"deleted_techs,omitempty"`
	DeletedTechDetails                 []TechDeleteReport            `json:"deleted_tech_details,omitempty"`
	DeletedSounds                      []SoundDeleteReport           `json:"deleted_sounds,omitempty"`
	DeletedUnits                       []UnitDeleteReport            `json:"deleted_units,omitempty"`
	UnitAvailability                   []UnitAvailabilityReport      `json:"unit_availability,omitempty"`
	PatchedEffects                     []int                         `json:"patched_effects,omitempty"`
	PatchedUnitHeaders                 []int                         `json:"patched_unit_headers,omitempty"`
	PatchedCivs                        []int                         `json:"patched_civs,omitempty"`
	PatchedTerrainRestrictions         []int                         `json:"patched_terrain_restrictions,omitempty"`
	PatchedTerrains                    []int                         `json:"patched_terrains,omitempty"`
	PatchedSounds                      []int                         `json:"patched_sounds,omitempty"`
	PatchedPlayerColours               []int                         `json:"patched_player_colours,omitempty"`
	PatchedTechs                       []int                         `json:"patched_techs,omitempty"`
	DisconnectedTechs                  []TechDisconnectReport        `json:"disconnected_techs,omitempty"`
	DisconnectedUnits                  []UnitDisconnectReport        `json:"disconnected_units,omitempty"`
	PatchedTechTree                    bool                          `json:"patched_tech_tree,omitempty"`
	CreatedTechTreeBuildingConnections []int                         `json:"created_tech_tree_building_connections,omitempty"`
	CreatedTechTreeUnitConnections     []int                         `json:"created_tech_tree_unit_connections,omitempty"`
	CreatedTechTreeResearchConnections []int                         `json:"created_tech_tree_research_connections,omitempty"`
	DeletedTechTreeBuildingConnections []int                         `json:"deleted_tech_tree_building_connections,omitempty"`
	DeletedTechTreeUnitConnections     []int                         `json:"deleted_tech_tree_unit_connections,omitempty"`
	DeletedTechTreeResearchConnections []int                         `json:"deleted_tech_tree_research_connections,omitempty"`
	TechTreeRewrites                   []TechTreeRewriteReport       `json:"tech_tree_rewrites,omitempty"`
	ReferenceRewrites                  []ReferenceRewriteReport      `json:"reference_rewrites,omitempty"`
	AuthoringNotes                     []string                      `json:"authoring_notes,omitempty"`
	PayloadRoundTripOK                 bool                          `json:"payload_roundtrip_ok"`
	ReadbackOK                         bool                          `json:"readback_ok"`
	Verified                           bool                          `json:"verified"`
	Verification                       aoe2.VerificationClaim        `json:"verification"`
}

type AbilityCreateReport struct {
	Name     string `json:"name"`
	TechID   int    `json:"tech_id"`
	EffectID int    `json:"effect_id"`
	Strategy string `json:"strategy"`
}

type AbilityDeleteReport struct {
	TechID   int    `json:"tech_id"`
	EffectID int    `json:"effect_id"`
	Strategy string `json:"strategy"`
	Note     string `json:"note,omitempty"`
}

type TechDisconnectReport struct {
	TechID           int              `json:"tech_id"`
	Complete         bool             `json:"complete"`
	ResidualSummary  ReferenceSummary `json:"residual_summary"`
	Strategy         string           `json:"strategy"`
	RecipeRewrites   int              `json:"recipe_rewrites"`
	StructuralCaveat string           `json:"structural_caveat,omitempty"`
}

type TechDeleteReport struct {
	TechID            int    `json:"tech_id"`
	Strategy          string `json:"strategy"`
	MovedTailTechID   *int   `json:"moved_tail_tech_id,omitempty"`
	MovedTailTechName string `json:"moved_tail_tech_name,omitempty"`
	ReferenceRewrites int    `json:"reference_rewrites,omitempty"`
	Note              string `json:"note,omitempty"`
}

func disconnectedTechReports(idx *datfile.Index, ids []int) []TechDisconnectReport {
	if len(ids) == 0 {
		return nil
	}
	reports := make([]TechDisconnectReport, 0, len(ids))
	for _, id := range ids {
		refs := References(idx, DeletePlanRequest{Section: "tech", ID: id})
		item := TechDisconnectReport{
			TechID:          id,
			Complete:        refs.Summary.Total == 0,
			ResidualSummary: refs.Summary,
			Strategy:        "remove_verified_references_without_physical_row_deletion",
			RecipeRewrites:  1,
		}
		if !item.Complete {
			item.StructuralCaveat = "Verified rewrite surfaces were removed, but residual references remain; inspect with kit dat refs before treating the tech as fully inert in-engine."
		}
		reports = append(reports, item)
	}
	return reports
}

type UnitDisconnectReport struct {
	UnitID           int              `json:"unit_id"`
	Complete         bool             `json:"complete"`
	ResidualSummary  ReferenceSummary `json:"residual_summary"`
	Strategy         string           `json:"strategy"`
	RecipeRewrites   int              `json:"recipe_rewrites"`
	StructuralCaveat string           `json:"structural_caveat,omitempty"`
}

func disconnectedUnitReports(idx *datfile.Index, ids []int) []UnitDisconnectReport {
	if len(ids) == 0 {
		return nil
	}
	reports := make([]UnitDisconnectReport, 0, len(ids))
	for _, id := range ids {
		refs := References(idx, DeletePlanRequest{Section: "unit", ID: id})
		item := UnitDisconnectReport{
			UnitID:          id,
			Complete:        refs.Summary.Total == 0,
			ResidualSummary: refs.Summary,
			Strategy:        "remove_verified_unit_references_without_physical_row_deletion",
			RecipeRewrites:  1,
		}
		if !item.Complete {
			item.StructuralCaveat = "Verified unit references were removed across tech-tree, typed effect commands, unit-header tasks, and known unit-record fields; ambiguous effect-command operand sightings may remain and should be inspected with kit dat refs before treating the unit as fully inert in-engine."
		}
		reports = append(reports, item)
	}
	return reports
}

func unitHeaderDisconnectRecipes(idx *datfile.Index, ids []int) []UnitHeaderPatchRecipe {
	targetIDs := make(map[int]bool, len(ids))
	for _, id := range ids {
		targetIDs[id] = true
	}
	out := []UnitHeaderPatchRecipe{}
	for _, header := range idx.UnitHeaders {
		if !header.Exists {
			continue
		}
		patches := []datfile.TaskPatch{}
		for _, task := range header.Tasks {
			if !targetIDs[int(task.ObjectID)] {
				continue
			}
			objectID := int16(-1)
			patches = append(patches, datfile.TaskPatch{
				Index:    task.Index,
				ObjectID: &objectID,
			})
		}
		if len(patches) > 0 {
			out = append(out, UnitHeaderPatchRecipe{ID: header.Index, Tasks: patches})
		}
	}
	return out
}

func mergeUnitHeaderPatchRecipes(base, generated []UnitHeaderPatchRecipe) ([]UnitHeaderPatchRecipe, error) {
	out := append([]UnitHeaderPatchRecipe(nil), base...)
	byID := make(map[int]int, len(out))
	for i, recipe := range out {
		if _, exists := byID[recipe.ID]; exists {
			return nil, fmt.Errorf("unit_headers id=%d appears more than once", recipe.ID)
		}
		byID[recipe.ID] = i
	}
	for _, recipe := range generated {
		i, exists := byID[recipe.ID]
		if !exists {
			byID[recipe.ID] = len(out)
			out = append(out, recipe)
			continue
		}
		if out[i].SetTasks != nil {
			return nil, fmt.Errorf("disconnect_units cannot compose with unit_headers[%d].set_tasks", recipe.ID)
		}
		for _, generatedTask := range recipe.Tasks {
			for _, existingTask := range out[i].Tasks {
				if existingTask.Index == generatedTask.Index && existingTask.ObjectID != nil &&
					generatedTask.ObjectID != nil && *existingTask.ObjectID != *generatedTask.ObjectID {
					return nil, fmt.Errorf("disconnect_units conflicts with unit_headers[%d].task[%d].object_id", recipe.ID, generatedTask.Index)
				}
			}
			out[i].Tasks = append(out[i].Tasks, generatedTask)
		}
	}
	return out, nil
}

func unitRecordDisconnectReplacements(payload []byte, idx *datfile.Index, ids []int) ([]payloadReplacement, error) {
	targetIDs := make(map[int]bool, len(ids))
	for _, id := range ids {
		targetIDs[id] = true
	}
	var replacements []payloadReplacement
	for _, civ := range idx.Civs {
		for _, unit := range civ.Units {
			if !unit.Present {
				continue
			}
			patch := datfile.UnitPatch{}
			if targetIDs[int(unit.BloodUnitID)] {
				patch.BloodUnitID = recipeInt16Ptr(-1)
			}
			if unit.Type50 != nil && targetIDs[int(unit.Type50.ProjectileUnitID)] {
				patch.Type50ProjectileUnitID = recipeInt16Ptr(-1)
			}
			if unit.Action != nil {
				dropSites, changed := filteredI16List(unit.Action.DropSites, targetIDs)
				if changed {
					patch.SetDropSites = &dropSites
				}
				for _, task := range unit.Action.Tasks {
					if !targetIDs[int(task.ObjectID)] {
						continue
					}
					patch.Tasks = append(patch.Tasks, datfile.TaskPatch{
						Index:    task.Index,
						ObjectID: recipeInt16Ptr(-1),
					})
				}
			}
			if unit.Creatable != nil {
				trainLocations, changed := filteredTrainLocations(unit.Creatable.TrainLocations, targetIDs)
				if changed {
					patch.SetTrainLocations = &trainLocations
				}
			}
			if unit.Building != nil {
				for _, row := range unit.Building.LinkedBuildings {
					if !targetIDs[int(row.UnitID)] {
						continue
					}
					patch.LinkedBuildings = append(patch.LinkedBuildings, datfile.LinkedBuildingPatch{
						Index:  row.Index,
						UnitID: recipeInt16Ptr(-1),
					})
				}
			}
			if patch.Empty() {
				continue
			}
			record, _, err := datfile.RebuildUnitRecord(payload, unit, patch)
			if err != nil {
				return nil, fmt.Errorf("disconnect_units unit record civ=%d unit=%d: %w", civ.Index, unit.Index, err)
			}
			replacements = append(replacements, payloadReplacement{
				Start: unit.RecordStart,
				End:   unit.RecordEnd,
				Data:  record,
			})
		}
	}
	return replacements, nil
}

func filteredI16List(values []int16, remove map[int]bool) ([]int16, bool) {
	out := make([]int16, 0, len(values))
	changed := false
	for _, value := range values {
		if remove[int(value)] {
			changed = true
			continue
		}
		out = append(out, value)
	}
	return out, changed
}

func filteredTrainLocations(values []datfile.TrainLocation, remove map[int]bool) ([]datfile.TrainLocationRow, bool) {
	out := make([]datfile.TrainLocationRow, 0, len(values))
	changed := false
	for _, value := range values {
		if remove[int(value.TrainUnitID)] {
			changed = true
			continue
		}
		out = append(out, datfile.TrainLocationRow{
			TrainTime:   value.TrainTime,
			TrainUnitID: value.TrainUnitID,
			TrainButton: value.TrainButton,
			TrainHotkey: value.TrainHotkey,
		})
	}
	return out, changed
}

func recipeInt16Ptr(value int16) *int16 {
	return &value
}

func recipeIntPtr(value int) *int {
	return &value
}

func recipeFloat32Ptr(value float32) *float32 {
	return &value
}

type UnitDeleteReport struct {
	CivID         int    `json:"civ_id"`
	UnitID        int    `json:"unit_id"`
	Strategy      string `json:"strategy"`
	BeforeEnabled uint8  `json:"before_enabled"`
	AfterEnabled  uint8  `json:"after_enabled"`
	Note          string `json:"note,omitempty"`
}

type UnitAvailabilityReport struct {
	CivID         int    `json:"civ_id"`
	UnitID        int    `json:"unit_id"`
	Present       bool   `json:"present"`
	BeforeEnabled uint8  `json:"before_enabled"`
	AfterEnabled  uint8  `json:"after_enabled"`
	Strategy      string `json:"strategy"`
	Note          string `json:"note,omitempty"`
}

type SoundDeleteReport struct {
	SoundID                int    `json:"sound_id"`
	Strategy               string `json:"strategy"`
	BeforeTotalProbability int16  `json:"before_total_probability"`
	AfterTotalProbability  int16  `json:"after_total_probability"`
	ItemCount              int    `json:"item_count"`
	Note                   string `json:"note,omitempty"`
}

type PlayerColourDeleteReport struct {
	ID          int    `json:"id"`
	BeforeCount int    `json:"before_count"`
	AfterCount  int    `json:"after_count"`
	Strategy    string `json:"strategy"`
	Note        string `json:"note,omitempty"`
}

type TechTreeRewriteReport struct {
	Target             string `json:"target"`
	From               int32  `json:"from"`
	To                 *int32 `json:"to,omitempty"`
	Remove             bool   `json:"remove"`
	ReplacedScalars    int    `json:"replaced_scalars"`
	ReplacedListValues int    `json:"replaced_list_values"`
	RemovedListValues  int    `json:"removed_list_values"`
}

type ReferenceRewriteReport struct {
	Target                  string                 `json:"target"`
	From                    int32                  `json:"from"`
	To                      *int32                 `json:"to,omitempty"`
	Remove                  bool                   `json:"remove"`
	TechRequiredRefs        int                    `json:"tech_required_refs"`
	EffectCommandsRewritten int                    `json:"effect_commands_rewritten"`
	EffectCommandsRemoved   int                    `json:"effect_commands_removed"`
	TechTree                *TechTreeRewriteReport `json:"tech_tree,omitempty"`
}

func PatchRecipeFile(inputPath, outputPath string, recipe Recipe) (PatchReport, error) {
	inputAbs, err := filepathAbs(inputPath)
	if err != nil {
		return PatchReport{}, err
	}
	outputAbs, err := filepathAbs(outputPath)
	if err != nil {
		return PatchReport{}, err
	}
	if inputAbs == outputAbs {
		return PatchReport{}, errors.New("refusing in-place dat codec patch; choose a separate output path")
	}
	compressed, err := os.ReadFile(inputPath)
	if err != nil {
		return PatchReport{}, err
	}
	output, report, err := PatchRecipe(compressed, recipe)
	if err != nil {
		return PatchReport{}, err
	}
	if err := os.WriteFile(outputPath, output, 0644); err != nil {
		return PatchReport{}, err
	}
	return report, nil
}

func PlanRecipeFile(inputPath string, recipe Recipe) (PatchReport, error) {
	compressed, err := os.ReadFile(inputPath)
	if err != nil {
		return PatchReport{}, err
	}
	_, report, err := PatchRecipe(compressed, recipe)
	if err != nil {
		return PatchReport{}, err
	}
	return report, nil
}

func PatchRecipe(compressed []byte, recipe Recipe) ([]byte, PatchReport, error) {
	if recipe.Empty() {
		return nil, PatchReport{}, errors.New("recipe has no codec effect/tech operations")
	}
	createGraphics := recipe.CreateGraphics
	if recipe.CreateGraphic != nil {
		createGraphics = append([]datfile.GraphicCreateRecipe{*recipe.CreateGraphic}, createGraphics...)
	}
	createUnits := recipe.CreateUnits
	if recipe.CreateUnit != nil {
		createUnits = append([]datfile.UnitCreateRecipe{*recipe.CreateUnit}, createUnits...)
	}
	originalPayload, err := datfile.Inflate(compressed)
	if err != nil {
		return nil, PatchReport{}, err
	}
	originalIdx, err := datfile.Parse(originalPayload)
	if err != nil {
		return nil, PatchReport{}, fmt.Errorf("parse input: %w", err)
	}
	currentCompressed := append([]byte(nil), compressed...)
	report := PatchReport{
		Version:              Version,
		InputCompressedBytes: len(compressed),
		InputInflatedBytes:   len(originalPayload),
		BeforeGraphicCount:   originalIdx.GraphicsSize,
	}
	for i, id := range recipe.DeleteGraphics {
		output, deleteReport, err := datfile.DeleteGraphic(currentCompressed, id)
		if err != nil {
			return nil, PatchReport{}, fmt.Errorf("delete_graphics[%d] id=%d: %w", i, id, err)
		}
		currentCompressed = output
		report.DeletedGraphics = append(report.DeletedGraphics, deleteReport)
	}
	for i, item := range recipe.Graphics {
		output, graphicReport, err := datfile.PatchGraphic(currentCompressed, item.ID, datfile.GraphicPatch{
			Name:               item.Name,
			FileName:           item.FileName,
			ParticleEffectName: item.ParticleEffectName,
			SLP:                item.SLP,
			IsLoaded:           item.IsLoaded,
			OldColorFlag:       item.OldColorFlag,
			Layer:              item.Layer,
			PlayerColor:        item.PlayerColor,
			Rainbow:            item.Rainbow,
			TransparentSelect:  item.TransparentSelect,
			Coordinates:        item.Coordinates,
			SoundID:            item.SoundID,
			WwiseSoundID:       item.WwiseSoundID,
			FrameCount:         item.FrameCount,
			SpeedMultiplier:    item.SpeedMultiplier,
			FrameDuration:      item.FrameDuration,
			ReplayDelay:        item.ReplayDelay,
			SequenceType:       item.SequenceType,
			MirroringMode:      item.MirroringMode,
			EditorFlag:         item.EditorFlag,
			SetDeltas:          item.SetDeltas,
			SetAngleSounds:     item.SetAngleSounds,
		})
		if err != nil {
			return nil, PatchReport{}, fmt.Errorf("graphics[%d] id=%d: %w", i, item.ID, err)
		}
		currentCompressed = output
		report.GraphicReports = append(report.GraphicReports, graphicReport)
	}
	for i, item := range createGraphics {
		output, createReport, err := datfile.CreateGraphic(currentCompressed, item)
		if err != nil {
			return nil, PatchReport{}, fmt.Errorf("create_graphics[%d] from=%d: %w", i, item.From, err)
		}
		currentCompressed = output
		report.CreatedGraphics = append(report.CreatedGraphics, createReport)
	}
	for i, item := range createUnits {
		output, createReport, err := datfile.CreateUnit(currentCompressed, item)
		if err != nil {
			return nil, PatchReport{}, fmt.Errorf("create_units[%d] from_civ=%d from_unit=%d: %w", i, item.FromCivID, item.FromUnitID, err)
		}
		currentCompressed = output
		report.CreatedUnits = append(report.CreatedUnits, createReport)
	}
	for i, item := range recipe.Units {
		output, unitReport, err := datfile.PatchUnit(currentCompressed, item.CivID, item.UnitID, datfile.UnitPatch{
			Class:                       item.Class,
			HitPoints:                   item.HitPoints,
			LineOfSight:                 item.LineOfSight,
			MovementType:                item.MovementType,
			StandingGraphic1:            item.StandingGraphic1,
			StandingGraphic2:            item.StandingGraphic2,
			DyingGraphic:                item.DyingGraphic,
			BloodUnitID:                 item.BloodUnitID,
			IconID:                      item.IconID,
			Enabled:                     item.Enabled,
			Attributes:                  item.Attributes,
			DamageGraphics:              item.DamageGraphics,
			SetDamageGraphics:           item.SetDamageGraphics,
			Type50ProjectileUnitID:      item.Type50ProjectileUnitID,
			Type50MaxRange:              item.Type50MaxRange,
			Type50BlastWidth:            item.Type50BlastWidth,
			Type50AttackGraphic:         item.Type50AttackGraphic,
			Type50BlastDamage:           item.Type50BlastDamage,
			Type50Attacks:               item.Type50Attacks,
			Type50Armours:               item.Type50Armours,
			SetType50Attacks:            item.SetType50Attacks,
			SetType50Armours:            item.SetType50Armours,
			TrainTime0:                  item.TrainTime0,
			TrainUnitID0:                item.TrainUnitID0,
			TrainButtonID0:              item.TrainButtonID0,
			TrainHotKeyID0:              item.TrainHotKeyID0,
			CreatableButtonIconID:       item.CreatableButtonIconID,
			CreatableButtonHotkeyAction: item.CreatableButtonHotkeyAction,
			Costs:                       item.Costs,
			TrainLocations:              item.TrainLocations,
			SetTrainLocations:           item.SetTrainLocations,
			SetDropSites:                item.SetDropSites,
			LinkedBuildings:             item.LinkedBuildings,
			Tasks:                       item.Tasks,
			SetTasks:                    item.SetTasks,
		})
		if err != nil {
			return nil, PatchReport{}, fmt.Errorf("units[%d] civ=%d unit=%d: %w", i, item.CivID, item.UnitID, err)
		}
		currentCompressed = output
		report.UnitReports = append(report.UnitReports, unitReport)
	}
	payload, err := datfile.Inflate(currentCompressed)
	if err != nil {
		return nil, PatchReport{}, err
	}
	idx, err := datfile.Parse(payload)
	if err != nil {
		return nil, PatchReport{}, fmt.Errorf("parse input: %w", err)
	}
	effects := cloneEffects(idx.Effects)
	techs := cloneTechs(idx.Techs)
	sounds := cloneSounds(idx.Sounds)
	playerColours := clonePlayerColours(idx.PlayerColours)
	report.BeforeEffectCount = len(effects)
	report.BeforeCivCount = len(idx.Civs)
	report.BeforeTechCount = len(techs)
	report.BeforeSoundCount = len(sounds)
	report.BeforePlayerColourCount = len(playerColours)
	if recipe.CreateEffect != nil {
		recipe.CreateEffects = append([]EffectCreateRecipe{*recipe.CreateEffect}, recipe.CreateEffects...)
	}
	if recipe.DisableEffect != nil {
		recipe.DisableEffects = append([]int{*recipe.DisableEffect}, recipe.DisableEffects...)
	}
	if recipe.CreateAbility != nil {
		recipe.CreateAbilities = append([]AbilityCreateRecipe{*recipe.CreateAbility}, recipe.CreateAbilities...)
	}
	if recipe.DisableAbility != nil {
		recipe.DisableAbilities = append([]int{*recipe.DisableAbility}, recipe.DisableAbilities...)
	}
	if recipe.DeleteAbility != nil {
		recipe.DeleteAbilities = append([]int{*recipe.DeleteAbility}, recipe.DeleteAbilities...)
	}
	if recipe.CreateTech != nil {
		recipe.CreateTechs = append([]TechCreateRecipe{*recipe.CreateTech}, recipe.CreateTechs...)
	}
	if recipe.CreateSound != nil {
		recipe.CreateSounds = append([]SoundCreateRecipe{*recipe.CreateSound}, recipe.CreateSounds...)
	}
	if recipe.CreatePlayerColour != nil {
		recipe.CreatePlayerColours = append([]PlayerColourCreateRecipe{*recipe.CreatePlayerColour}, recipe.CreatePlayerColours...)
	}
	if recipe.DeletePlayerColour != nil {
		recipe.DeletePlayerColours = append([]int{*recipe.DeletePlayerColour}, recipe.DeletePlayerColours...)
	}
	referenceRewrites := append([]TechTreeIDRewriteRecipe(nil), recipe.ReferenceRewrites...)
	for i, id := range recipe.DisconnectTechs {
		if id < 0 || id >= len(idx.Techs) {
			return nil, PatchReport{}, fmt.Errorf("disconnect_techs[%d] id=%d outside tech table", i, id)
		}
		referenceRewrites = append([]TechTreeIDRewriteRecipe{{Target: "tech", From: int32(id), Remove: true}}, referenceRewrites...)
	}
	for i, id := range recipe.DisconnectUnits {
		if id < 0 || id > math.MaxInt16 {
			return nil, PatchReport{}, fmt.Errorf("disconnect_units[%d] id=%d outside int16 unit-id range", i, id)
		}
		referenceRewrites = append([]TechTreeIDRewriteRecipe{{Target: "unit", From: int32(id), Remove: true}}, referenceRewrites...)
	}
	unitHeaderRecipes := append([]UnitHeaderPatchRecipe(nil), recipe.UnitHeaders...)
	if len(recipe.DisconnectUnits) > 0 {
		generated := unitHeaderDisconnectRecipes(idx, recipe.DisconnectUnits)
		unitHeaderRecipes, err = mergeUnitHeaderPatchRecipes(unitHeaderRecipes, generated)
		if err != nil {
			return nil, PatchReport{}, err
		}
	}
	for i, item := range recipe.CreateEffects {
		effect, err := buildCreatedEffect(len(effects), item, effects)
		if err != nil {
			return nil, PatchReport{}, fmt.Errorf("create_effects[%d]: %w", i, err)
		}
		effects = append(effects, effect)
		report.CreatedEffects = append(report.CreatedEffects, effect.Index)
	}
	for i, item := range recipe.CreateAbilities {
		if strings.TrimSpace(item.Name) == "" {
			return nil, PatchReport{}, fmt.Errorf("create_abilities[%d] requires name", i)
		}
		if item.FromTech < 0 || item.FromTech >= len(techs) {
			return nil, PatchReport{}, fmt.Errorf("create_abilities[%d] from_tech=%d outside tech table", i, item.FromTech)
		}
		effectName := item.EffectName
		if effectName == "" {
			effectName = item.Name + " Effect"
		}
		commands := append([]EffectCommandRecipe(nil), item.Commands...)
		if len(commands) == 0 {
			fromEffect := int(techs[item.FromTech].EffectID)
			if item.FromEffect != nil {
				fromEffect = *item.FromEffect
			}
			if fromEffect < 0 || fromEffect >= len(effects) {
				return nil, PatchReport{}, fmt.Errorf("create_abilities[%d] source effect_id=%d outside effect table", i, fromEffect)
			}
			commands = effectCommandRecipesFromCommands(effects[fromEffect].Commands)
		}
		commands = append(commands, item.AppendCommands...)
		if len(commands) == 0 {
			return nil, PatchReport{}, fmt.Errorf("create_abilities[%d] requires commands, append_commands, or a source tech/effect with commands", i)
		}
		effect, err := buildCreatedEffect(len(effects), EffectCreateRecipe{
			Name:     effectName,
			Commands: commands,
		}, effects)
		if err != nil {
			return nil, PatchReport{}, fmt.Errorf("create_abilities[%d] effect: %w", i, err)
		}
		effects = append(effects, effect)
		report.CreatedEffects = append(report.CreatedEffects, effect.Index)
		effectID := int16(effect.Index)
		techRecipe := abilityTechCreateRecipe(item, effectID)
		tech, err := buildCreatedTech(len(techs), techs[techRecipe.From], techRecipe)
		if err != nil {
			return nil, PatchReport{}, fmt.Errorf("create_abilities[%d] tech: %w", i, err)
		}
		techs = append(techs, tech)
		report.CreatedTechs = append(report.CreatedTechs, tech.Index)
		report.CreatedAbilities = append(report.CreatedAbilities, AbilityCreateReport{
			Name:     item.Name,
			TechID:   tech.Index,
			EffectID: effect.Index,
			Strategy: "created_effect_then_created_tech_with_effect_id_readback",
		})
	}
	if len(recipe.DisableAbilities) > 0 {
		current := *idx
		current.Effects = effects
		current.Techs = techs
		abilityReports, effectPatches, err := planAbilityRecipeDisables(&current, recipe.DisableAbilities, recipe.Effects)
		if err != nil {
			return nil, PatchReport{}, err
		}
		report.DisabledAbilities = append(report.DisabledAbilities, abilityReports...)
		recipe.Effects = append(effectPatches, recipe.Effects...)
	}
	if len(recipe.DisableEffects) > 0 {
		effectReports, effectPatches, err := planEffectRecipeDisables(len(effects), recipe.DisableEffects, recipe.Effects, recipe.DeleteEffects)
		if err != nil {
			return nil, PatchReport{}, err
		}
		report.DisabledEffects = append(report.DisabledEffects, effectReports...)
		recipe.Effects = append(effectPatches, recipe.Effects...)
	}
	for i, item := range recipe.Effects {
		if item.ID < 0 || item.ID >= len(effects) {
			return nil, PatchReport{}, fmt.Errorf("effects[%d] id=%d outside effects table", i, item.ID)
		}
		effect, err := patchEffect(effects[item.ID], item)
		if err != nil {
			return nil, PatchReport{}, fmt.Errorf("effects[%d]: %w", i, err)
		}
		effects[item.ID] = effect
		report.PatchedEffects = append(report.PatchedEffects, item.ID)
	}
	for i, item := range recipe.CreateTechs {
		if item.From < 0 || item.From >= len(techs) {
			return nil, PatchReport{}, fmt.Errorf("create_techs[%d] from=%d outside tech table", i, item.From)
		}
		tech, err := buildCreatedTech(len(techs), techs[item.From], item)
		if err != nil {
			return nil, PatchReport{}, fmt.Errorf("create_techs[%d]: %w", i, err)
		}
		if tech.EffectID < -1 || int(tech.EffectID) >= len(effects) {
			return nil, PatchReport{}, fmt.Errorf("create_techs[%d] effect_id=%d outside effects table after creates", i, tech.EffectID)
		}
		techs = append(techs, tech)
		report.CreatedTechs = append(report.CreatedTechs, tech.Index)
	}
	for i, item := range recipe.Techs {
		if item.ID < 0 || item.ID >= len(techs) {
			return nil, PatchReport{}, fmt.Errorf("techs[%d] id=%d outside tech table", i, item.ID)
		}
		tech, err := patchTech(techs[item.ID], item)
		if err != nil {
			return nil, PatchReport{}, fmt.Errorf("techs[%d] id=%d: %w", i, item.ID, err)
		}
		if tech.EffectID < -1 || int(tech.EffectID) >= len(effects) {
			return nil, PatchReport{}, fmt.Errorf("techs[%d] effect_id=%d outside effects table", i, tech.EffectID)
		}
		techs[item.ID] = tech
		report.PatchedTechs = append(report.PatchedTechs, item.ID)
	}
	if len(recipe.DeleteAbilities) > 0 {
		current := *idx
		current.Effects = effects
		current.Techs = techs
		abilityReports, techDeletes, effectDeletes, err := planAbilityRecipeDeletes(&current, recipe.DeleteAbilities)
		if err != nil {
			return nil, PatchReport{}, err
		}
		report.DeletedAbilities = append(report.DeletedAbilities, abilityReports...)
		recipe.DeleteTechs = append(techDeletes, recipe.DeleteTechs...)
		recipe.DeleteEffects = append(effectDeletes, recipe.DeleteEffects...)
	}
	tree := cloneTechTree(idx.TechTree)
	techTreeChanged := false
	if len(referenceRewrites) > 0 {
		for i, rewrite := range referenceRewrites {
			rewriteReport, err := applyReferenceRewrite(effects, techs, &tree, rewrite)
			if err != nil {
				return nil, PatchReport{}, fmt.Errorf("reference_rewrites[%d]: %w", i, err)
			}
			effects = rewriteReport.Effects
			techs = rewriteReport.Techs
			report.ReferenceRewrites = append(report.ReferenceRewrites, rewriteReport.Report)
		}
		techTreeChanged = true
	}
	if recipe.TechTree != nil {
		beforeBuildingConnections := len(tree.BuildingConnections)
		beforeUnitConnections := len(tree.UnitConnections)
		beforeResearchConnections := len(tree.ResearchConnections)
		rewriteReports, deleteReports, err := patchTechTree(&tree, *recipe.TechTree)
		if err != nil {
			return nil, PatchReport{}, err
		}
		for id := beforeBuildingConnections; id < len(tree.BuildingConnections); id++ {
			report.CreatedTechTreeBuildingConnections = append(report.CreatedTechTreeBuildingConnections, id)
		}
		for id := beforeUnitConnections; id < len(tree.UnitConnections); id++ {
			report.CreatedTechTreeUnitConnections = append(report.CreatedTechTreeUnitConnections, id)
		}
		for id := beforeResearchConnections; id < len(tree.ResearchConnections); id++ {
			report.CreatedTechTreeResearchConnections = append(report.CreatedTechTreeResearchConnections, id)
		}
		report.DeletedTechTreeBuildingConnections = append(report.DeletedTechTreeBuildingConnections, deleteReports.BuildingConnections...)
		report.DeletedTechTreeUnitConnections = append(report.DeletedTechTreeUnitConnections, deleteReports.UnitConnections...)
		report.DeletedTechTreeResearchConnections = append(report.DeletedTechTreeResearchConnections, deleteReports.ResearchConnections...)
		report.TechTreeRewrites = append(report.TechTreeRewrites, rewriteReports...)
		techTreeChanged = true
	}
	if len(recipe.DeleteTechs) > 0 {
		deleted, details, nextEffects, nextTechs, nextTree, rewriteReports, err := deleteTechs(idx, effects, techs, tree, recipe.DeleteTechs)
		if err != nil {
			return nil, PatchReport{}, err
		}
		effects = nextEffects
		techs = nextTechs
		tree = nextTree
		report.DeletedTechs = deleted
		report.DeletedTechDetails = details
		report.ReferenceRewrites = append(report.ReferenceRewrites, rewriteReports...)
		techTreeChanged = true
	}
	if len(recipe.DeleteEffects) > 0 {
		deleted, nextEffects, nextTechs, err := deleteEffects(effects, techs, recipe.DeleteEffects)
		if err != nil {
			return nil, PatchReport{}, err
		}
		effects = nextEffects
		techs = nextTechs
		report.DeletedEffects = deleted
	}
	var replacements []payloadReplacement
	for i, item := range unitHeaderRecipes {
		if item.ID < 0 || item.ID >= len(idx.UnitHeaders) {
			return nil, PatchReport{}, fmt.Errorf("unit_headers[%d] id=%d outside unit header table", i, item.ID)
		}
		record, err := patchUnitHeaderRecord(payload, idx.UnitHeaders[item.ID], item)
		if err != nil {
			return nil, PatchReport{}, fmt.Errorf("unit_headers[%d] id=%d: %w", i, item.ID, err)
		}
		replacements = append(replacements, payloadReplacement{Start: idx.UnitHeaders[item.ID].Span.Start, End: idx.UnitHeaders[item.ID].Span.End, Data: record})
		report.PatchedUnitHeaders = append(report.PatchedUnitHeaders, item.ID)
	}
	for i, item := range recipe.Civs {
		if item.ID < 0 || item.ID >= len(idx.Civs) {
			return nil, PatchReport{}, fmt.Errorf("civs[%d] id=%d outside civ table", i, item.ID)
		}
		record, err := patchCivRecord(payload, idx.Civs[item.ID], item)
		if err != nil {
			return nil, PatchReport{}, fmt.Errorf("civs[%d] id=%d: %w", i, item.ID, err)
		}
		replacements = append(replacements, payloadReplacement{Start: idx.Civs[item.ID].Span.Start, End: idx.Civs[item.ID].Span.End, Data: record})
		report.PatchedCivs = append(report.PatchedCivs, item.ID)
	}
	for i, item := range recipe.TerrainRestrictions {
		if item.ID < 0 || item.ID >= len(idx.TerrainRestrictions) {
			return nil, PatchReport{}, fmt.Errorf("terrain_restrictions[%d] id=%d outside terrain restriction table", i, item.ID)
		}
		record, err := patchTerrainRestrictionRecord(payload, idx.TerrainRestrictions[item.ID], item)
		if err != nil {
			return nil, PatchReport{}, fmt.Errorf("terrain_restrictions[%d] id=%d: %w", i, item.ID, err)
		}
		replacements = append(replacements, payloadReplacement{Start: idx.TerrainRestrictions[item.ID].Span.Start, End: idx.TerrainRestrictions[item.ID].Span.End, Data: record})
		report.PatchedTerrainRestrictions = append(report.PatchedTerrainRestrictions, item.ID)
	}
	for i, item := range recipe.Terrains {
		if item.ID < 0 || item.ID >= len(idx.Terrains) {
			return nil, PatchReport{}, fmt.Errorf("terrains[%d] id=%d outside terrain table", i, item.ID)
		}
		record, err := patchTerrainRecord(payload, idx.Terrains[item.ID], item)
		if err != nil {
			return nil, PatchReport{}, fmt.Errorf("terrains[%d] id=%d: %w", i, item.ID, err)
		}
		replacements = append(replacements, payloadReplacement{Start: idx.Terrains[item.ID].Span.Start, End: idx.Terrains[item.ID].Span.End, Data: record})
		report.PatchedTerrains = append(report.PatchedTerrains, item.ID)
	}
	for i, item := range recipe.CreateSounds {
		if item.From < 0 || item.From >= len(sounds) {
			return nil, PatchReport{}, fmt.Errorf("create_sounds[%d] from=%d outside sound table", i, item.From)
		}
		sound, err := buildCreatedSound(len(sounds), sounds[item.From], item)
		if err != nil {
			return nil, PatchReport{}, fmt.Errorf("create_sounds[%d]: %w", i, err)
		}
		sounds = append(sounds, sound)
		report.CreatedSounds = append(report.CreatedSounds, sound.Index)
	}
	for i, item := range recipe.Sounds {
		if item.ID < 0 || item.ID >= len(sounds) {
			return nil, PatchReport{}, fmt.Errorf("sounds[%d] id=%d outside sound table", i, item.ID)
		}
		sound, err := patchSound(sounds[item.ID], item)
		if err != nil {
			return nil, PatchReport{}, fmt.Errorf("sounds[%d] id=%d: %w", i, item.ID, err)
		}
		sounds[item.ID] = sound
		report.PatchedSounds = append(report.PatchedSounds, item.ID)
	}
	for i, id := range recipe.DeleteSounds {
		if id < 0 || id >= len(sounds) {
			return nil, PatchReport{}, fmt.Errorf("delete_sounds[%d] id=%d outside sound table", i, id)
		}
		sound, deleteReport := muteSound(sounds[id])
		sounds[id] = sound
		report.DeletedSounds = append(report.DeletedSounds, deleteReport)
	}
	if len(recipe.CreateSounds) > 0 || len(recipe.Sounds) > 0 || len(recipe.DeleteSounds) > 0 {
		soundsStart, soundsEnd, err := sectionBounds(idx, "sounds_header", "sound[", len(idx.Sounds))
		if err != nil {
			return nil, PatchReport{}, err
		}
		soundsBytes, err := encodeSoundsSection(sounds)
		if err != nil {
			return nil, PatchReport{}, err
		}
		replacements = append(replacements, payloadReplacement{Start: soundsStart, End: soundsEnd, Data: soundsBytes})
	}
	for i, item := range recipe.CreatePlayerColours {
		if item.From < 0 || item.From >= len(playerColours) {
			return nil, PatchReport{}, fmt.Errorf("create_player_colours[%d] from=%d outside player colour table", i, item.From)
		}
		colour := buildCreatedPlayerColour(len(playerColours), playerColours[item.From], item)
		playerColours = append(playerColours, colour)
		report.CreatedPlayerColours = append(report.CreatedPlayerColours, colour.Index)
	}
	if len(recipe.CreatePlayerColours) > 0 || len(recipe.DeletePlayerColours) > 0 {
		for i, item := range recipe.PlayerColours {
			if item.ID < 0 || item.ID >= len(playerColours) {
				return nil, PatchReport{}, fmt.Errorf("player_colours[%d] id=%d outside player colour table", i, item.ID)
			}
			colour := playerColours[item.ID]
			applyPlayerColourPatch(&colour, item)
			playerColours[item.ID] = colour
			report.PatchedPlayerColours = append(report.PatchedPlayerColours, item.ID)
		}
		deleteReports, deletedPlayerColours, err := deletePlayerColourRows(idx, playerColours, recipe.DeletePlayerColours)
		if err != nil {
			return nil, PatchReport{}, err
		}
		playerColours = deletedPlayerColours
		report.DeletedPlayerColours = append(report.DeletedPlayerColours, deleteReports...)
		start, end, err := playerColoursSectionBounds(idx)
		if err != nil {
			return nil, PatchReport{}, err
		}
		section, err := encodePlayerColoursSection(playerColours)
		if err != nil {
			return nil, PatchReport{}, err
		}
		replacements = append(replacements, payloadReplacement{Start: start, End: end, Data: section})
	} else {
		for i, item := range recipe.PlayerColours {
			if item.ID < 0 || item.ID >= len(idx.PlayerColours) {
				return nil, PatchReport{}, fmt.Errorf("player_colours[%d] id=%d outside player colour table", i, item.ID)
			}
			record, err := patchPlayerColourRecord(payload, idx.PlayerColours[item.ID], item)
			if err != nil {
				return nil, PatchReport{}, fmt.Errorf("player_colours[%d] id=%d: %w", i, item.ID, err)
			}
			replacements = append(replacements, payloadReplacement{Start: idx.PlayerColours[item.ID].Span.Start, End: idx.PlayerColours[item.ID].Span.End, Data: record})
			report.PatchedPlayerColours = append(report.PatchedPlayerColours, item.ID)
		}
	}
	for i, item := range recipe.DeleteUnits {
		unitReplacements, unitReports, err := deleteUnitRecords(payload, idx, item)
		if err != nil {
			return nil, PatchReport{}, fmt.Errorf("delete_units[%d] unit=%d: %w", i, item.UnitID, err)
		}
		replacements = append(replacements, unitReplacements...)
		report.DeletedUnits = append(report.DeletedUnits, unitReports...)
	}
	if len(recipe.DisconnectUnits) > 0 {
		unitReplacements, err := unitRecordDisconnectReplacements(payload, idx, recipe.DisconnectUnits)
		if err != nil {
			return nil, PatchReport{}, err
		}
		replacements = append(replacements, unitReplacements...)
	}
	for i, item := range recipe.UnitAvailability {
		unitReplacements, unitReports, err := setUnitAvailabilityRecords(payload, idx, item)
		if err != nil {
			return nil, PatchReport{}, fmt.Errorf("unit_availability[%d] unit=%d: %w", i, item.UnitID, err)
		}
		replacements = append(replacements, unitReplacements...)
		report.UnitAvailability = append(report.UnitAvailability, unitReports...)
	}
	if techTreeChanged {
		treeBytes, err := encodeTechTreeSection(tree)
		if err != nil {
			return nil, PatchReport{}, err
		}
		replacements = append(replacements, payloadReplacement{Start: idx.TechTree.Span.Start, End: idx.TechTree.Span.End, Data: treeBytes})
		report.PatchedTechTree = true
	}
	outPayload, err := rebuildCodecSections(payload, idx, effects, techs, replacements)
	if err != nil {
		return nil, PatchReport{}, err
	}
	output, err := datfile.Deflate(outPayload)
	if err != nil {
		return nil, PatchReport{}, err
	}
	roundTripPayload, err := datfile.Inflate(output)
	if err != nil {
		return nil, PatchReport{}, fmt.Errorf("inflate patched output: %w", err)
	}
	report.PayloadRoundTripOK = bytes.Equal(outPayload, roundTripPayload)
	if !report.PayloadRoundTripOK {
		return nil, PatchReport{}, errors.New("patched payload changed after deflate/inflate round trip")
	}
	afterIdx, err := datfile.Parse(roundTripPayload)
	if err != nil {
		return nil, PatchReport{}, fmt.Errorf("re-index patched output: %w", err)
	}
	report.OutputCompressedBytes = len(output)
	report.OutputInflatedBytes = len(outPayload)
	report.InflatedLengthDelta = len(outPayload) - len(originalPayload)
	report.AfterEffectCount = len(afterIdx.Effects)
	report.AfterGraphicCount = afterIdx.GraphicsSize
	report.AfterCivCount = len(afterIdx.Civs)
	report.AfterTechCount = len(afterIdx.Techs)
	report.AfterSoundCount = len(afterIdx.Sounds)
	report.AfterPlayerColourCount = len(afterIdx.PlayerColours)
	report.ReadbackOK = report.AfterEffectCount == len(effects) &&
		report.AfterGraphicCount == originalIdx.GraphicsSize+len(report.CreatedGraphics) &&
		report.AfterCivCount == len(idx.Civs) &&
		report.AfterTechCount == len(techs) &&
		report.AfterSoundCount == len(sounds) &&
		report.AfterPlayerColourCount == len(playerColours)
	if !report.ReadbackOK {
		return nil, PatchReport{}, fmt.Errorf("readback counts effects=%d/%d graphics=%d/%d civs=%d/%d techs=%d/%d sounds=%d/%d player_colours=%d/%d",
			report.AfterEffectCount, len(effects),
			report.AfterGraphicCount, originalIdx.GraphicsSize+len(report.CreatedGraphics),
			report.AfterCivCount, len(idx.Civs),
			report.AfterTechCount, len(techs),
			report.AfterSoundCount, len(sounds),
			report.AfterPlayerColourCount, len(playerColours))
	}
	if err := verifyCivReadback(afterIdx, recipe.Civs); err != nil {
		return nil, PatchReport{}, err
	}
	if err := verifyTerrainRestrictionReadback(afterIdx, recipe.TerrainRestrictions); err != nil {
		return nil, PatchReport{}, err
	}
	if err := verifyTerrainReadback(afterIdx, recipe.Terrains); err != nil {
		return nil, PatchReport{}, err
	}
	if err := verifySoundReadback(afterIdx, recipe.Sounds); err != nil {
		return nil, PatchReport{}, err
	}
	if err := verifyCreatedSoundReadback(afterIdx, recipe.CreateSounds, report.CreatedSounds); err != nil {
		return nil, PatchReport{}, err
	}
	if err := verifySoundDeletesReadback(afterIdx, report.DeletedSounds); err != nil {
		return nil, PatchReport{}, err
	}
	if err := verifyUnitHeaderReadback(afterIdx, unitHeaderRecipes); err != nil {
		return nil, PatchReport{}, err
	}
	if err := verifyPlayerColourReadback(afterIdx, recipe.PlayerColours); err != nil {
		return nil, PatchReport{}, err
	}
	if err := verifyCreatedPlayerColourReadback(afterIdx, recipe.CreatePlayerColours, report.CreatedPlayerColours); err != nil {
		return nil, PatchReport{}, err
	}
	if err := verifyDeletedPlayerColourReadback(afterIdx, report.DeletedPlayerColours); err != nil {
		return nil, PatchReport{}, err
	}
	if err := verifyUnitDeletesReadback(afterIdx, report.DeletedUnits); err != nil {
		return nil, PatchReport{}, err
	}
	if err := verifyUnitAvailabilityReadback(afterIdx, report.UnitAvailability); err != nil {
		return nil, PatchReport{}, err
	}
	if recipe.TechTree != nil {
		if err := verifyTechTreeReadback(afterIdx, *recipe.TechTree); err != nil {
			return nil, PatchReport{}, err
		}
	}
	if err := verifyReferenceRewriteReadback(afterIdx, referenceRewrites); err != nil {
		return nil, PatchReport{}, err
	}
	if err := verifyDisconnectUnitReadback(afterIdx, recipe.DisconnectUnits); err != nil {
		return nil, PatchReport{}, err
	}
	report.DisconnectedTechs = disconnectedTechReports(afterIdx, recipe.DisconnectTechs)
	report.DisconnectedUnits = disconnectedUnitReports(afterIdx, recipe.DisconnectUnits)
	report.Verified = true
	report.Verification = aoe2.StructureVerification(true)
	report.Verification.Note = "codec patch applied requested Graphic/Unit/PlayerColour creates and rebuilt typed Effects/Techs sections plus requested UnitHeader/Civ/Terrain/TerrainRestriction/Sound/PlayerColour/Unit tombstone records, deflate/inflate round-tripped the decompressed payload, and reparsed output for count/readback checks; in-engine behavior remains a separate oracle."
	return output, report, nil
}

func cloneEffects(in []datfile.Effect) []datfile.Effect {
	out := make([]datfile.Effect, len(in))
	for i, effect := range in {
		out[i] = effect
		out[i].Commands = append([]datfile.EffectCommand(nil), effect.Commands...)
	}
	return out
}

func cloneTechs(in []datfile.Tech) []datfile.Tech {
	out := make([]datfile.Tech, len(in))
	for i, tech := range in {
		out[i] = tech
		out[i].RequiredTechs = append([]int16(nil), tech.RequiredTechs...)
		out[i].ResourceCosts = append([]datfile.ResearchResourceCost(nil), tech.ResourceCosts...)
		out[i].ResearchLocations = append([]datfile.ResearchLocation(nil), tech.ResearchLocations...)
	}
	return out
}

func cloneTechTree(in datfile.TechTree) datfile.TechTree {
	out := in
	out.Ages = append([]datfile.TechTreeAge(nil), in.Ages...)
	for i := range out.Ages {
		out.Ages[i].Buildings = append([]int32(nil), in.Ages[i].Buildings...)
		out.Ages[i].Units = append([]int32(nil), in.Ages[i].Units...)
		out.Ages[i].Techs = append([]int32(nil), in.Ages[i].Techs...)
		out.Ages[i].Common = cloneTechTreeCommon(in.Ages[i].Common)
		out.Ages[i].BuildingsPerZone = append([]uint8(nil), in.Ages[i].BuildingsPerZone...)
		out.Ages[i].GroupLengthPerZone = append([]uint8(nil), in.Ages[i].GroupLengthPerZone...)
	}
	out.BuildingConnections = append([]datfile.BuildingConnection(nil), in.BuildingConnections...)
	for i := range out.BuildingConnections {
		out.BuildingConnections[i].Buildings = append([]int32(nil), in.BuildingConnections[i].Buildings...)
		out.BuildingConnections[i].Units = append([]int32(nil), in.BuildingConnections[i].Units...)
		out.BuildingConnections[i].Techs = append([]int32(nil), in.BuildingConnections[i].Techs...)
		out.BuildingConnections[i].Common = cloneTechTreeCommon(in.BuildingConnections[i].Common)
		out.BuildingConnections[i].UnitsTechsTotal = append([]uint8(nil), in.BuildingConnections[i].UnitsTechsTotal...)
		out.BuildingConnections[i].UnitsTechsFirst = append([]uint8(nil), in.BuildingConnections[i].UnitsTechsFirst...)
	}
	out.UnitConnections = append([]datfile.UnitConnection(nil), in.UnitConnections...)
	for i := range out.UnitConnections {
		out.UnitConnections[i].Common = cloneTechTreeCommon(in.UnitConnections[i].Common)
		out.UnitConnections[i].Units = append([]int32(nil), in.UnitConnections[i].Units...)
	}
	out.ResearchConnections = append([]datfile.ResearchConnection(nil), in.ResearchConnections...)
	for i := range out.ResearchConnections {
		out.ResearchConnections[i].Buildings = append([]int32(nil), in.ResearchConnections[i].Buildings...)
		out.ResearchConnections[i].Units = append([]int32(nil), in.ResearchConnections[i].Units...)
		out.ResearchConnections[i].Techs = append([]int32(nil), in.ResearchConnections[i].Techs...)
		out.ResearchConnections[i].Common = cloneTechTreeCommon(in.ResearchConnections[i].Common)
	}
	return out
}

func cloneTechTreeCommon(in datfile.TechTreeCommon) datfile.TechTreeCommon {
	return datfile.TechTreeCommon{
		SlotsUsed:    in.SlotsUsed,
		UnitResearch: append([]int32(nil), in.UnitResearch...),
		Mode:         append([]int32(nil), in.Mode...),
	}
}

func cloneSounds(in []datfile.Sound) []datfile.Sound {
	out := make([]datfile.Sound, len(in))
	for i, sound := range in {
		out[i] = sound
		out[i].Items = append([]datfile.SoundItem(nil), sound.Items...)
	}
	return out
}

func clonePlayerColours(in []datfile.PlayerColour) []datfile.PlayerColour {
	return append([]datfile.PlayerColour(nil), in...)
}

func buildCreatedEffect(index int, recipe EffectCreateRecipe, effects []datfile.Effect) (datfile.Effect, error) {
	commandRecipes := append([]EffectCommandRecipe(nil), recipe.Commands...)
	if recipe.FromEffect != nil {
		fromEffect := *recipe.FromEffect
		if fromEffect < 0 || fromEffect >= len(effects) {
			return datfile.Effect{}, fmt.Errorf("from_effect=%d outside effect table", fromEffect)
		}
		if len(commandRecipes) == 0 {
			commandRecipes = effectCommandRecipesFromCommands(effects[fromEffect].Commands)
		}
	}
	commands, err := commandsFromRecipe(commandRecipes)
	if err != nil {
		return datfile.Effect{}, err
	}
	if len(recipe.RemoveCommands) > 0 {
		commands, err = removeEffectCommands(commands, recipe.RemoveCommands)
		if err != nil {
			return datfile.Effect{}, err
		}
	}
	if len(recipe.AppendCommands) > 0 {
		appended, err := commandsFromRecipe(recipe.AppendCommands)
		if err != nil {
			return datfile.Effect{}, fmt.Errorf("append_commands: %w", err)
		}
		commands = append(commands, appended...)
	}
	for i := range commands {
		commands[i].Index = i
	}
	return datfile.Effect{Index: index, Name: recipe.Name, CommandSize: len(commands), Commands: commands}, nil
}

func abilityTechCreateRecipe(recipe AbilityCreateRecipe, effectID int16) TechCreateRecipe {
	name := recipe.Name
	return TechCreateRecipe{
		From:                   recipe.FromTech,
		Name:                   &name,
		RequiredTechs:          append([]int16(nil), recipe.RequiredTechs...),
		ResourceCosts:          append([]datfile.ResearchResourceCost(nil), recipe.ResourceCosts...),
		RequiredTechCount:      recipe.RequiredTechCount,
		Civ:                    recipe.Civ,
		FullTechMode:           recipe.FullTechMode,
		LanguageDLLName:        recipe.LanguageDLLName,
		LanguageDLLDescription: recipe.LanguageDLLDescription,
		EffectID:               &effectID,
		Type:                   recipe.Type,
		IconID:                 recipe.IconID,
		LanguageDLLHelp:        recipe.LanguageDLLHelp,
		LanguageDLLTechTree:    recipe.LanguageDLLTechTree,
		Repeatable:             recipe.Repeatable,
		ResearchLocations:      append([]datfile.ResearchLocation(nil), recipe.ResearchLocations...),
	}
}

func planAbilityRecipeDeletes(idx *datfile.Index, ids []int) ([]AbilityDeleteReport, []int, []int, error) {
	if len(ids) == 0 {
		return nil, nil, nil, nil
	}
	seen := map[int]bool{}
	reports := make([]AbilityDeleteReport, 0, len(ids))
	techDeletes := make([]int, 0, len(ids))
	effectDeletes := make([]int, 0, len(ids))
	for i, id := range ids {
		if seen[id] {
			return nil, nil, nil, fmt.Errorf("delete_abilities[%d] duplicate tech id=%d", i, id)
		}
		seen[id] = true
		plan := DeletePlan(idx, DeletePlanRequest{Section: "ability", ID: id})
		if !plan.Supported || plan.Strategy != "paired_tech_then_effect_delete" {
			if plan.Reason != "" {
				return nil, nil, nil, fmt.Errorf("delete_abilities[%d] tech=%d unsupported: %s", i, id, plan.Reason)
			}
			return nil, nil, nil, fmt.Errorf("delete_abilities[%d] tech=%d unsupported strategy=%s", i, id, plan.Strategy)
		}
		if id < 0 || id >= len(idx.Techs) {
			return nil, nil, nil, fmt.Errorf("delete_abilities[%d] tech=%d outside tech table", i, id)
		}
		effectID := int(idx.Techs[id].EffectID)
		if effectID < 0 || effectID >= len(idx.Effects) {
			return nil, nil, nil, fmt.Errorf("delete_abilities[%d] tech=%d effect_id=%d outside effect table", i, id, effectID)
		}
		reports = append(reports, AbilityDeleteReport{
			TechID:   id,
			EffectID: effectID,
			Strategy: "paired_tech_then_effect_delete",
			Note:     "delete_ability is a guarded shorthand for deleting a tech with the current tech-delete strategy and then deleting its now-unreferenced paired effect; effects above the deleted effect ID are renumbered by the existing delete_effects codec.",
		})
		techDeletes = append(techDeletes, id)
		effectDeletes = append(effectDeletes, effectID)
	}
	return reports, techDeletes, effectDeletes, nil
}

func planAbilityRecipeDisables(idx *datfile.Index, ids []int, existingEffectPatches []EffectPatchRecipe) ([]AbilityDeleteReport, []EffectPatchRecipe, error) {
	if len(ids) == 0 {
		return nil, nil, nil
	}
	patchedEffects := make(map[int]bool, len(existingEffectPatches))
	for _, patch := range existingEffectPatches {
		patchedEffects[patch.ID] = true
	}
	seenTechs := map[int]bool{}
	seenEffects := map[int]bool{}
	reports := make([]AbilityDeleteReport, 0, len(ids))
	patches := make([]EffectPatchRecipe, 0, len(ids))
	for i, id := range ids {
		if seenTechs[id] {
			return nil, nil, fmt.Errorf("disable_abilities[%d] duplicate tech id=%d", i, id)
		}
		seenTechs[id] = true
		if id < 0 || id >= len(idx.Techs) {
			return nil, nil, fmt.Errorf("disable_abilities[%d] tech=%d outside tech table", i, id)
		}
		effectID := int(idx.Techs[id].EffectID)
		if effectID < 0 || effectID >= len(idx.Effects) {
			return nil, nil, fmt.Errorf("disable_abilities[%d] tech=%d effect_id=%d outside effect table", i, id, effectID)
		}
		if seenEffects[effectID] {
			return nil, nil, fmt.Errorf("disable_abilities[%d] tech=%d targets effect_id=%d already disabled by this recipe", i, id, effectID)
		}
		if patchedEffects[effectID] {
			return nil, nil, fmt.Errorf("disable_abilities[%d] tech=%d targets effect_id=%d also patched by effects[]; split the recipe or choose one operation", i, id, effectID)
		}
		if !canSemanticallyClearAbilityEffect(idx, id, effectID) {
			return nil, nil, fmt.Errorf("disable_abilities[%d] tech=%d effect_id=%d is shared by another tech; refusing to clear shared behavior", i, id, effectID)
		}
		seenEffects[effectID] = true
		commands := []EffectCommandRecipe{}
		patches = append(patches, EffectPatchRecipe{ID: effectID, Commands: &commands})
		reports = append(reports, AbilityDeleteReport{
			TechID:   id,
			EffectID: effectID,
			Strategy: "semantic_clear_ability_effect",
			Note:     "disable_ability preserves the tech/effect IDs and clears the paired effect commands; UI, prerequisites, and tech-tree references may remain visible.",
		})
	}
	return reports, patches, nil
}

func planEffectRecipeDisables(effectCount int, ids []int, existingEffectPatches []EffectPatchRecipe, deleteEffects []int) ([]int, []EffectPatchRecipe, error) {
	if len(ids) == 0 {
		return nil, nil, nil
	}
	patchedEffects := make(map[int]bool, len(existingEffectPatches))
	for _, patch := range existingEffectPatches {
		patchedEffects[patch.ID] = true
	}
	deletedEffects := make(map[int]bool, len(deleteEffects))
	for _, id := range deleteEffects {
		deletedEffects[id] = true
	}
	seen := map[int]bool{}
	reports := make([]int, 0, len(ids))
	patches := make([]EffectPatchRecipe, 0, len(ids))
	for i, id := range ids {
		if seen[id] {
			return nil, nil, fmt.Errorf("disable_effects[%d] duplicate effect id=%d", i, id)
		}
		seen[id] = true
		if id < 0 || id >= effectCount {
			return nil, nil, fmt.Errorf("disable_effects[%d] id=%d outside effects table length %d", i, id, effectCount)
		}
		if patchedEffects[id] {
			return nil, nil, fmt.Errorf("disable_effects[%d] id=%d also patched by effects[]; split the recipe or choose one operation", i, id)
		}
		if deletedEffects[id] {
			return nil, nil, fmt.Errorf("disable_effects[%d] id=%d also deleted by delete_effects; split the recipe or choose one operation", i, id)
		}
		commands := []EffectCommandRecipe{}
		patches = append(patches, EffectPatchRecipe{ID: id, Commands: &commands})
		reports = append(reports, id)
	}
	return reports, patches, nil
}

func patchEffect(effect datfile.Effect, recipe EffectPatchRecipe) (datfile.Effect, error) {
	if recipe.Name != nil {
		effect.Name = *recipe.Name
	}
	if recipe.Commands != nil {
		commands, err := commandsFromRecipe(*recipe.Commands)
		if err != nil {
			return datfile.Effect{}, err
		}
		effect.Commands = commands
	}
	if len(recipe.RemoveCommands) > 0 {
		commands, err := removeEffectCommands(effect.Commands, recipe.RemoveCommands)
		if err != nil {
			return datfile.Effect{}, err
		}
		effect.Commands = commands
	}
	if len(recipe.AppendCommands) > 0 {
		commands, err := commandsFromRecipe(recipe.AppendCommands)
		if err != nil {
			return datfile.Effect{}, err
		}
		effect.Commands = append(effect.Commands, commands...)
	}
	for i := range effect.Commands {
		effect.Commands[i].Index = i
	}
	effect.CommandSize = len(effect.Commands)
	return effect, nil
}

func removeEffectCommands(commands []datfile.EffectCommand, indices []int) ([]datfile.EffectCommand, error) {
	remove := make(map[int]bool, len(indices))
	for _, index := range indices {
		if index < 0 || index >= len(commands) {
			return nil, fmt.Errorf("remove_commands index=%d outside command list length %d", index, len(commands))
		}
		if remove[index] {
			return nil, fmt.Errorf("remove_commands index=%d is duplicated", index)
		}
		remove[index] = true
	}
	out := make([]datfile.EffectCommand, 0, len(commands)-len(remove))
	for i, command := range commands {
		if remove[i] {
			continue
		}
		out = append(out, command)
	}
	return out, nil
}

func commandsFromRecipe(in []EffectCommandRecipe) ([]datfile.EffectCommand, error) {
	out := make([]datfile.EffectCommand, 0, len(in))
	for i, command := range in {
		built, err := commandFromRecipe(command)
		if err != nil {
			return nil, fmt.Errorf("commands[%d]: %w", i, err)
		}
		built.Index = i
		out = append(out, built)
	}
	return out, nil
}

func effectCommandRecipesFromCommands(in []datfile.EffectCommand) []EffectCommandRecipe {
	out := make([]EffectCommandRecipe, 0, len(in))
	for _, command := range in {
		out = append(out, effectCommandRecipeFromCommand(command))
	}
	return out
}

func effectCommandRecipeFromCommand(command datfile.EffectCommand) EffectCommandRecipe {
	amount := command.D
	packedType, packedAmount, hasPacked := datfileEffectPackedAmount(command)
	withPacked := func(recipe EffectCommandRecipe) EffectCommandRecipe {
		if hasPacked {
			recipe.PackedTypeID = recipeIntPtr(packedType)
			recipe.PackedTypeName = datfile.EffectPackedAttackArmorClassName(packedType)
			recipe.PackedAmount = recipeIntPtr(packedAmount)
		}
		return recipe
	}
	raw := EffectCommandRecipe{
		Type: command.Type,
		A:    command.A,
		B:    command.B,
		C:    command.C,
		D:    command.D,
	}
	switch command.Type {
	case 0:
		if command.A >= 0 {
			if command.B == -1 {
				return withPacked(EffectCommandRecipe{Kind: "set_unit_attribute", UnitID: recipeInt16Ptr(command.A), AttributeID: recipeInt16Ptr(command.C), AttributeName: datfile.EffectAttributeName(command.C), Amount: recipeFloat32Ptr(amount)})
			}
			return withPacked(EffectCommandRecipe{Kind: "set_attribute", UnitID: recipeInt16Ptr(command.A), UnitClassID: recipeInt16Ptr(command.B), UnitClassName: datfile.EffectUnitClassName(command.B), AttributeID: recipeInt16Ptr(command.C), AttributeName: datfile.EffectAttributeName(command.C), Amount: recipeFloat32Ptr(amount)})
		}
	case 1:
		if command.C == -1 {
			return EffectCommandRecipe{Kind: "resource_modifier", ResourceID: recipeInt16Ptr(command.A), ResourceName: datfile.EffectResourceName(command.A), OperationID: recipeInt16Ptr(command.B), OperationName: datfile.EffectOperationName(command.B), Amount: recipeFloat32Ptr(amount)}
		}
	case 2:
		if command.A >= 0 {
			return EffectCommandRecipe{Kind: "enable_unit", UnitID: recipeInt16Ptr(command.A)}
		}
	case 3:
		if command.A >= 0 && command.B >= 0 {
			return EffectCommandRecipe{Kind: "upgrade_unit", UnitID: recipeInt16Ptr(command.A), ToUnitID: recipeInt16Ptr(command.B), Amount: recipeFloat32Ptr(amount)}
		}
	case 4:
		if command.A >= 0 {
			if command.B == -1 {
				return withPacked(EffectCommandRecipe{Kind: "add_unit_attribute", UnitID: recipeInt16Ptr(command.A), AttributeID: recipeInt16Ptr(command.C), AttributeName: datfile.EffectAttributeName(command.C), Amount: recipeFloat32Ptr(amount)})
			}
			return withPacked(EffectCommandRecipe{Kind: "add_attribute", UnitID: recipeInt16Ptr(command.A), UnitClassID: recipeInt16Ptr(command.B), UnitClassName: datfile.EffectUnitClassName(command.B), AttributeID: recipeInt16Ptr(command.C), AttributeName: datfile.EffectAttributeName(command.C), Amount: recipeFloat32Ptr(amount)})
		}
	case 5:
		if command.A >= 0 {
			if command.B == -1 {
				return withPacked(EffectCommandRecipe{Kind: "multiply_unit_attribute", UnitID: recipeInt16Ptr(command.A), AttributeID: recipeInt16Ptr(command.C), AttributeName: datfile.EffectAttributeName(command.C), Amount: recipeFloat32Ptr(amount)})
			}
			return withPacked(EffectCommandRecipe{Kind: "multiply_attribute", UnitID: recipeInt16Ptr(command.A), UnitClassID: recipeInt16Ptr(command.B), UnitClassName: datfile.EffectUnitClassName(command.B), AttributeID: recipeInt16Ptr(command.C), AttributeName: datfile.EffectAttributeName(command.C), Amount: recipeFloat32Ptr(amount)})
		}
	case 6:
		if command.C == -1 {
			return EffectCommandRecipe{Kind: "resource_multiplier", ResourceID: recipeInt16Ptr(command.A), ResourceName: datfile.EffectResourceName(command.A), Amount: recipeFloat32Ptr(amount)}
		}
	case 7:
		if command.A >= 0 && command.B >= 0 && command.C == -1 {
			return EffectCommandRecipe{Kind: "spawn_unit", UnitID: recipeInt16Ptr(command.A), BuildingID: recipeInt16Ptr(command.B), Amount: recipeFloat32Ptr(amount)}
		}
	case 8:
		if command.A >= 0 && command.C == -1 {
			techID := int(command.A)
			return EffectCommandRecipe{Kind: "modify_tech", TechID: recipeIntPtr(techID), TechAttrID: recipeInt16Ptr(command.B), TechAttributeName: datfile.EffectTechAttributeName(command.B), Amount: recipeFloat32Ptr(amount)}
		}
	case 100, 101:
		if command.A >= 0 && command.C == -1 {
			techID := int(command.A)
			kind := "set_tech_cost"
			if command.Type == 101 {
				kind = "add_tech_cost"
			}
			return EffectCommandRecipe{Kind: kind, TechID: recipeIntPtr(techID), ResourceID: recipeInt16Ptr(command.B), ResourceName: datfile.EffectResourceName(command.B), Amount: recipeFloat32Ptr(amount)}
		}
	case 102:
		techID := int(command.D)
		if command.A == -1 && command.B == -1 && command.C == -1 && command.D == float32(techID) && techID >= 0 {
			return EffectCommandRecipe{Kind: "disable_tech", TechID: recipeIntPtr(techID)}
		}
	case 103:
		if command.A >= 0 && command.C == -1 {
			techID := int(command.A)
			return EffectCommandRecipe{Kind: "tech_time_modifier", TechID: recipeIntPtr(techID), OperationID: recipeInt16Ptr(command.B), OperationName: datfile.EffectOperationName(command.B), Amount: recipeFloat32Ptr(amount)}
		}
	case 200, 201, 202, 204:
		if command.A >= 0 {
			kind := datfile.EffectCommandTypeName(command.Type)
			return withPacked(EffectCommandRecipe{Kind: kind, UnitID: recipeInt16Ptr(command.A), BuildingID: recipeInt16Ptr(command.A), AttributeID: recipeInt16Ptr(command.C), AttributeName: datfile.EffectAttributeName(command.C), Amount: recipeFloat32Ptr(amount)})
		}
	}
	return raw
}

func datfileEffectPackedAmount(command datfile.EffectCommand) (int, int, bool) {
	semantic := datfile.InterpretEffectCommand(command)
	if semantic == nil || semantic.PackedTypeID == nil || semantic.PackedAmount == nil {
		return 0, 0, false
	}
	return *semantic.PackedTypeID, *semantic.PackedAmount, true
}

func resolveEffectResourceID(kind string, numeric *int16, name string, alias string) (int16, error) {
	if name == "" {
		name = alias
	}
	var named *int16
	if name != "" {
		value, ok := datfile.EffectResourceID(name)
		if !ok {
			return 0, fmt.Errorf("%s unknown resource name %q", kind, name)
		}
		named = &value
	}
	if numeric != nil && named != nil && *numeric != *named {
		return 0, fmt.Errorf("%s resource_id=%d conflicts with resource_name=%q (%d)", kind, *numeric, name, *named)
	}
	if numeric != nil {
		return *numeric, nil
	}
	if named != nil {
		return *named, nil
	}
	return 0, fmt.Errorf("%s needs resource_id or resource_name", kind)
}

func resolveEffectOperationID(kind string, numeric *int16, name string, alias string) (int16, error) {
	if name == "" {
		name = alias
	}
	var named *int16
	if name != "" {
		value, ok := datfile.EffectOperationID(name)
		if !ok {
			return 0, fmt.Errorf("%s unknown operation name %q", kind, name)
		}
		named = &value
	}
	if numeric != nil && named != nil && *numeric != *named {
		return 0, fmt.Errorf("%s operation_id=%d conflicts with operation_name=%q (%d)", kind, *numeric, name, *named)
	}
	if numeric != nil {
		return *numeric, nil
	}
	if named != nil {
		return *named, nil
	}
	return 0, fmt.Errorf("%s needs operation_id or operation_name", kind)
}

func resolveEffectAttributeID(kind string, numeric *int16, name string, alias string) (int16, error) {
	if name == "" {
		name = alias
	}
	var named *int16
	if name != "" {
		value, ok := datfile.EffectAttributeID(name)
		if !ok {
			return 0, fmt.Errorf("%s unknown attribute name %q", kind, name)
		}
		named = &value
	}
	if numeric != nil && named != nil && *numeric != *named {
		return 0, fmt.Errorf("%s attribute_id=%d conflicts with attribute_name=%q (%d)", kind, *numeric, name, *named)
	}
	if numeric != nil {
		return *numeric, nil
	}
	if named != nil {
		return *named, nil
	}
	return 0, fmt.Errorf("%s needs attribute_id or attribute_name", kind)
}

func resolveEffectUnitClassID(kind string, numeric *int16, name string, alias string) (int16, error) {
	if name == "" {
		name = alias
	}
	var named *int16
	if name != "" {
		value, ok := datfile.EffectUnitClassID(name)
		if !ok {
			return 0, fmt.Errorf("%s unknown unit_class_name %q", kind, name)
		}
		named = &value
	}
	if numeric != nil && named != nil && *numeric != *named {
		return 0, fmt.Errorf("%s unit_class_id=%d conflicts with unit_class_name=%q (%d)", kind, *numeric, name, *named)
	}
	if numeric != nil {
		return *numeric, nil
	}
	if named != nil {
		return *named, nil
	}
	return -1, nil
}

func resolveEffectTechAttributeID(kind string, numeric *int16, name string, alias string) (int16, error) {
	if name == "" {
		name = alias
	}
	var named *int16
	if name != "" {
		value, ok := datfile.EffectTechAttributeID(name)
		if !ok {
			return 0, fmt.Errorf("%s unknown tech_attribute_name %q", kind, name)
		}
		named = &value
	}
	if numeric != nil && named != nil && *numeric != *named {
		return 0, fmt.Errorf("%s tech_attribute_id=%d conflicts with tech_attribute_name=%q (%d)", kind, *numeric, name, *named)
	}
	if numeric != nil {
		return *numeric, nil
	}
	if named != nil {
		return *named, nil
	}
	return 0, fmt.Errorf("%s needs tech_attribute_id or tech_attribute_name", kind)
}

func commandFromRecipe(recipe EffectCommandRecipe) (datfile.EffectCommand, error) {
	if recipe.Kind == "" {
		command := datfile.EffectCommand{Type: recipe.Type, A: recipe.A, B: recipe.B, C: recipe.C, D: recipe.D}
		command.Semantic = datfile.InterpretEffectCommand(command)
		return command, nil
	}
	kind := normalizeEffectCommandKind(recipe.Kind)
	switch kind {
	case "resource_modifier", "team_resource_modifier", "enemy_resource_modifier", "neutral_resource_modifier", "gaia_resource_modifier":
		resourceID, err := resolveEffectResourceID(kind, recipe.ResourceID, recipe.ResourceName, recipe.Resource)
		if err != nil {
			return datfile.EffectCommand{}, err
		}
		operationID, err := resolveEffectOperationID(kind, recipe.OperationID, recipe.OperationName, recipe.Operation)
		if err != nil {
			return datfile.EffectCommand{}, err
		}
		if recipe.Amount == nil {
			return datfile.EffectCommand{}, fmt.Errorf("%s needs amount", kind)
		}
		commandType, err := effectResourceModifierCommandType(kind)
		if err != nil {
			return datfile.EffectCommand{}, err
		}
		command := datfile.EffectCommand{Type: commandType, A: resourceID, B: operationID, C: -1, D: *recipe.Amount}
		command.Semantic = datfile.InterpretEffectCommand(command)
		return command, nil
	case "disable_tech":
		if recipe.TechID == nil {
			return datfile.EffectCommand{}, errors.New("disable_tech needs tech_id")
		}
		if *recipe.TechID < 0 {
			return datfile.EffectCommand{}, fmt.Errorf("disable_tech tech_id=%d is negative", *recipe.TechID)
		}
		command := datfile.EffectCommand{Type: 102, A: -1, B: -1, C: -1, D: float32(*recipe.TechID)}
		command.Semantic = datfile.InterpretEffectCommand(command)
		return command, nil
	case "enable_unit", "team_enable_unit", "enemy_enable_unit", "neutral_enable_unit", "gaia_enable_unit":
		if recipe.UnitID == nil {
			return datfile.EffectCommand{}, fmt.Errorf("%s needs unit_id", kind)
		}
		commandType, err := effectEnableUnitCommandType(kind)
		if err != nil {
			return datfile.EffectCommand{}, err
		}
		command := datfile.EffectCommand{Type: commandType, A: *recipe.UnitID, B: -1, C: -1, D: 0}
		command.Semantic = datfile.InterpretEffectCommand(command)
		return command, nil
	case "upgrade_unit", "team_upgrade_unit", "enemy_upgrade_unit", "neutral_upgrade_unit", "gaia_upgrade_unit":
		if recipe.UnitID == nil {
			return datfile.EffectCommand{}, fmt.Errorf("%s needs unit_id", kind)
		}
		if recipe.ToUnitID == nil {
			return datfile.EffectCommand{}, fmt.Errorf("%s needs to_unit_id", kind)
		}
		amount := float32(0)
		if recipe.Amount != nil {
			amount = *recipe.Amount
		}
		commandType, err := effectUpgradeUnitCommandType(kind)
		if err != nil {
			return datfile.EffectCommand{}, err
		}
		command := datfile.EffectCommand{Type: commandType, A: *recipe.UnitID, B: *recipe.ToUnitID, C: -1, D: amount}
		command.Semantic = datfile.InterpretEffectCommand(command)
		return command, nil
	case "resource_multiplier", "team_resource_multiplier", "enemy_resource_multiplier", "neutral_resource_multiplier", "gaia_resource_multiplier":
		resourceID, err := resolveEffectResourceID(kind, recipe.ResourceID, recipe.ResourceName, recipe.Resource)
		if err != nil {
			return datfile.EffectCommand{}, err
		}
		if recipe.Amount == nil {
			return datfile.EffectCommand{}, fmt.Errorf("%s needs amount", kind)
		}
		commandType, err := effectResourceMultiplierCommandType(kind)
		if err != nil {
			return datfile.EffectCommand{}, err
		}
		command := datfile.EffectCommand{Type: commandType, A: resourceID, B: 0, C: -1, D: *recipe.Amount}
		command.Semantic = datfile.InterpretEffectCommand(command)
		return command, nil
	case "spawn_unit", "team_spawn_unit", "enemy_spawn_unit", "neutral_spawn_unit", "gaia_spawn_unit":
		if recipe.UnitID == nil {
			return datfile.EffectCommand{}, fmt.Errorf("%s needs unit_id", kind)
		}
		if recipe.BuildingID == nil {
			return datfile.EffectCommand{}, fmt.Errorf("%s needs building_id", kind)
		}
		if recipe.Amount == nil {
			return datfile.EffectCommand{}, fmt.Errorf("%s needs amount", kind)
		}
		commandType, err := effectSpawnUnitCommandType(kind)
		if err != nil {
			return datfile.EffectCommand{}, err
		}
		command := datfile.EffectCommand{Type: commandType, A: *recipe.UnitID, B: *recipe.BuildingID, C: -1, D: *recipe.Amount}
		command.Semantic = datfile.InterpretEffectCommand(command)
		return command, nil
	case "modify_tech", "team_modify_tech", "enemy_modify_tech", "neutral_modify_tech", "gaia_modify_tech":
		techID, err := checkedEffectCommandTechID(kind, recipe.TechID)
		if err != nil {
			return datfile.EffectCommand{}, err
		}
		techAttrID, err := resolveEffectTechAttributeID(kind, recipe.TechAttrID, recipe.TechAttributeName, recipe.TechAttribute)
		if err != nil {
			return datfile.EffectCommand{}, err
		}
		if recipe.Amount == nil {
			return datfile.EffectCommand{}, fmt.Errorf("%s needs amount", kind)
		}
		commandType, err := effectModifyTechCommandType(kind)
		if err != nil {
			return datfile.EffectCommand{}, err
		}
		command := datfile.EffectCommand{Type: commandType, A: techID, B: techAttrID, C: -1, D: *recipe.Amount}
		command.Semantic = datfile.InterpretEffectCommand(command)
		return command, nil
	case "set_tech_cost", "add_tech_cost":
		techID, err := checkedEffectCommandTechID(kind, recipe.TechID)
		if err != nil {
			return datfile.EffectCommand{}, err
		}
		resourceID, err := resolveEffectResourceID(kind, recipe.ResourceID, recipe.ResourceName, recipe.Resource)
		if err != nil {
			return datfile.EffectCommand{}, err
		}
		if recipe.Amount == nil {
			return datfile.EffectCommand{}, fmt.Errorf("%s needs amount", recipe.Kind)
		}
		commandType := uint8(100)
		if kind == "add_tech_cost" {
			commandType = 101
		}
		command := datfile.EffectCommand{Type: commandType, A: techID, B: resourceID, C: -1, D: *recipe.Amount}
		command.Semantic = datfile.InterpretEffectCommand(command)
		return command, nil
	case "tech_time_modifier":
		techID, err := checkedEffectCommandTechID(kind, recipe.TechID)
		if err != nil {
			return datfile.EffectCommand{}, err
		}
		operationID, err := resolveEffectOperationID(kind, recipe.OperationID, recipe.OperationName, recipe.Operation)
		if err != nil {
			return datfile.EffectCommand{}, err
		}
		if recipe.Amount == nil {
			return datfile.EffectCommand{}, errors.New("tech_time_modifier needs amount")
		}
		command := datfile.EffectCommand{Type: 103, A: techID, B: operationID, C: -1, D: *recipe.Amount}
		command.Semantic = datfile.InterpretEffectCommand(command)
		return command, nil
	case "set_attribute", "add_attribute", "multiply_attribute", "set_unit_attribute", "add_unit_attribute", "multiply_unit_attribute",
		"team_set_attribute", "team_add_attribute", "team_multiply_attribute",
		"enemy_set_attribute", "enemy_add_attribute", "enemy_multiply_attribute",
		"neutral_set_attribute", "neutral_add_attribute", "neutral_multiply_attribute",
		"gaia_set_attribute", "gaia_add_attribute", "gaia_multiply_attribute",
		"set_local_building_attribute", "add_local_building_attribute", "multiply_local_building_attribute", "add_local_building_attribute_advanced",
		"add_unit_armor", "add_unit_armour", "add_unit_attack", "add_local_building_armor", "add_local_building_armour", "add_local_building_attack":
		if recipe.UnitID == nil {
			return datfile.EffectCommand{}, fmt.Errorf("%s needs unit_id", kind)
		}
		resolvedAttributeID, amount, err := resolveAttributeCommandAmount(kind, recipe)
		if err != nil {
			return datfile.EffectCommand{}, err
		}
		unitClassID := int16(-1)
		if kind == "set_attribute" || kind == "add_attribute" || kind == "multiply_attribute" {
			unitClassID, err = resolveEffectUnitClassID(kind, recipe.UnitClassID, recipe.UnitClassName, recipe.UnitClass)
			if err != nil {
				return datfile.EffectCommand{}, err
			}
		}
		commandType, err := effectAttributeCommandType(kind)
		if err != nil {
			return datfile.EffectCommand{}, err
		}
		attributeOperand := resolvedAttributeID
		trailingOperand := int16(-1)
		if strings.Contains(kind, "_unit_") || strings.Contains(kind, "_local_building_") || strings.HasSuffix(kind, "_local_building_attribute") || kind == "add_local_building_attribute_advanced" {
			unitClassID = trailingOperand
		}
		command := datfile.EffectCommand{Type: commandType, A: *recipe.UnitID, B: unitClassID, C: attributeOperand, D: amount}
		command.Semantic = datfile.InterpretEffectCommand(command)
		return command, nil
	default:
		return datfile.EffectCommand{}, fmt.Errorf("unknown effect command kind %q", recipe.Kind)
	}
}

func normalizeEffectCommandKind(kind string) string {
	k := strings.ToLower(strings.TrimSpace(kind))
	k = strings.ReplaceAll(k, "-", "_")
	k = strings.ReplaceAll(k, " ", "_")
	switch k {
	case "multiply_attribute":
		return "multiply_attribute"
	case "team_attribute_multiplier", "team_multiply_attribute":
		return "team_multiply_attribute"
	case "enemy_attribute_multiplier", "enemy_multiply_attribute":
		return "enemy_multiply_attribute"
	case "neutral_attribute_multiplier", "neutral_multiply_attribute":
		return "neutral_multiply_attribute"
	case "gaia_attribute_multiplier", "gaia_multiply_attribute":
		return "gaia_multiply_attribute"
	case "set_local_attribute", "local_building_set_attribute", "local_set_attribute":
		return "set_local_building_attribute"
	case "add_local_attribute", "local_building_add_attribute", "local_add_attribute":
		return "add_local_building_attribute"
	case "multiply_local_attribute", "local_building_multiply_attribute", "local_multiply_attribute":
		return "multiply_local_building_attribute"
	case "local_building_advanced_add_attribute", "local_advanced_add_attribute":
		return "add_local_building_attribute_advanced"
	default:
		return k
	}
}

func effectAttributeCommandType(kind string) (uint8, error) {
	switch kind {
	case "set_attribute", "set_unit_attribute":
		return 0, nil
	case "add_attribute", "add_unit_attribute", "add_unit_armor", "add_unit_armour", "add_unit_attack":
		return 4, nil
	case "multiply_attribute", "multiply_unit_attribute":
		return 5, nil
	case "team_set_attribute":
		return 10, nil
	case "team_add_attribute":
		return 14, nil
	case "team_multiply_attribute":
		return 15, nil
	case "enemy_set_attribute":
		return 20, nil
	case "enemy_add_attribute":
		return 24, nil
	case "enemy_multiply_attribute":
		return 25, nil
	case "neutral_set_attribute":
		return 30, nil
	case "neutral_add_attribute":
		return 34, nil
	case "neutral_multiply_attribute":
		return 35, nil
	case "gaia_set_attribute":
		return 40, nil
	case "gaia_add_attribute":
		return 44, nil
	case "gaia_multiply_attribute":
		return 45, nil
	case "set_local_building_attribute":
		return 200, nil
	case "add_local_building_attribute":
		return 201, nil
	case "multiply_local_building_attribute":
		return 202, nil
	case "add_local_building_attribute_advanced", "add_local_building_armor", "add_local_building_armour", "add_local_building_attack":
		return 204, nil
	default:
		return 0, fmt.Errorf("unsupported attribute command kind %q", kind)
	}
}

func effectResourceModifierCommandType(kind string) (uint8, error) {
	switch kind {
	case "resource_modifier":
		return 1, nil
	case "team_resource_modifier":
		return 11, nil
	case "enemy_resource_modifier":
		return 21, nil
	case "neutral_resource_modifier":
		return 31, nil
	case "gaia_resource_modifier":
		return 41, nil
	default:
		return 0, fmt.Errorf("unsupported resource modifier command kind %q", kind)
	}
}

func effectEnableUnitCommandType(kind string) (uint8, error) {
	switch kind {
	case "enable_unit":
		return 2, nil
	case "team_enable_unit":
		return 12, nil
	case "enemy_enable_unit":
		return 22, nil
	case "neutral_enable_unit":
		return 32, nil
	case "gaia_enable_unit":
		return 42, nil
	default:
		return 0, fmt.Errorf("unsupported enable-unit command kind %q", kind)
	}
}

func effectUpgradeUnitCommandType(kind string) (uint8, error) {
	switch kind {
	case "upgrade_unit":
		return 3, nil
	case "team_upgrade_unit":
		return 13, nil
	case "enemy_upgrade_unit":
		return 23, nil
	case "neutral_upgrade_unit":
		return 33, nil
	case "gaia_upgrade_unit":
		return 43, nil
	default:
		return 0, fmt.Errorf("unsupported upgrade-unit command kind %q", kind)
	}
}

func effectResourceMultiplierCommandType(kind string) (uint8, error) {
	switch kind {
	case "resource_multiplier":
		return 6, nil
	case "team_resource_multiplier":
		return 16, nil
	case "enemy_resource_multiplier":
		return 26, nil
	case "neutral_resource_multiplier":
		return 36, nil
	case "gaia_resource_multiplier":
		return 46, nil
	default:
		return 0, fmt.Errorf("unsupported resource multiplier command kind %q", kind)
	}
}

func effectSpawnUnitCommandType(kind string) (uint8, error) {
	switch kind {
	case "spawn_unit":
		return 7, nil
	case "team_spawn_unit":
		return 17, nil
	case "enemy_spawn_unit":
		return 27, nil
	case "neutral_spawn_unit":
		return 37, nil
	case "gaia_spawn_unit":
		return 47, nil
	default:
		return 0, fmt.Errorf("unsupported spawn-unit command kind %q", kind)
	}
}

func effectModifyTechCommandType(kind string) (uint8, error) {
	switch kind {
	case "modify_tech":
		return 8, nil
	case "team_modify_tech":
		return 18, nil
	case "enemy_modify_tech":
		return 28, nil
	case "neutral_modify_tech":
		return 38, nil
	case "gaia_modify_tech":
		return 48, nil
	default:
		return 0, fmt.Errorf("unsupported modify-tech command kind %q", kind)
	}
}

func resolveAttributeCommandAmount(kind string, recipe EffectCommandRecipe) (int16, float32, error) {
	switch kind {
	case "add_unit_armor", "add_unit_armour", "add_local_building_armor", "add_local_building_armour":
		amount, err := resolvePackedEffectAmount(kind, recipe, 8)
		return 8, amount, err
	case "add_unit_attack", "add_local_building_attack":
		amount, err := resolvePackedEffectAmount(kind, recipe, 9)
		return 9, amount, err
	default:
		resolvedAttributeID, err := resolveEffectAttributeID(kind, recipe.AttributeID, recipe.AttributeName, recipe.Attribute)
		if err != nil {
			return 0, 0, err
		}
		if recipe.Amount == nil && recipe.PackedAmount == nil {
			return 0, 0, fmt.Errorf("%s needs amount", kind)
		}
		if recipe.PackedAmount != nil || recipe.PackedTypeID != nil || recipe.PackedType != "" || recipe.PackedTypeName != "" || recipe.CombatClassID != nil || recipe.CombatClass != "" || recipe.ArmorClassID != nil || recipe.ArmorClass != "" || recipe.AttackClassID != nil || recipe.AttackClass != "" {
			amount, err := resolvePackedEffectAmount(kind, recipe, resolvedAttributeID)
			return resolvedAttributeID, amount, err
		}
		return resolvedAttributeID, *recipe.Amount, nil
	}
}

func resolvePackedEffectAmount(kind string, recipe EffectCommandRecipe, attributeID int16) (float32, error) {
	if attributeID != 8 && attributeID != 9 {
		return 0, fmt.Errorf("%s packed attack/armor amount requires attribute_id 8 armor or 9 attack, got %d", kind, attributeID)
	}
	classID, err := resolvePackedEffectClassID(kind, recipe, attributeID)
	if err != nil {
		return 0, err
	}
	if recipe.PackedAmount == nil {
		if recipe.Amount == nil {
			return 0, fmt.Errorf("%s needs packed_amount or amount", kind)
		}
		amount := int(*recipe.Amount)
		if *recipe.Amount != float32(amount) {
			return 0, fmt.Errorf("%s amount=%g must be integral for packed attack/armor", kind, *recipe.Amount)
		}
		recipe.PackedAmount = &amount
	}
	packed, err := PackEffectAttackArmorAmount(classID, *recipe.PackedAmount)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", kind, err)
	}
	return packed, nil
}

func resolvePackedEffectClassID(kind string, recipe EffectCommandRecipe, attributeID int16) (int, error) {
	var numeric *int
	var name string
	if recipe.PackedTypeID != nil {
		numeric = recipe.PackedTypeID
	}
	if recipe.CombatClassID != nil {
		numeric = recipe.CombatClassID
	}
	if attributeID == 8 && recipe.ArmorClassID != nil {
		numeric = recipe.ArmorClassID
	}
	if attributeID == 9 && recipe.AttackClassID != nil {
		numeric = recipe.AttackClassID
	}
	for _, candidate := range []string{recipe.PackedTypeName, recipe.PackedType, recipe.CombatClass} {
		if candidate != "" {
			name = candidate
			break
		}
	}
	if attributeID == 8 && recipe.ArmorClass != "" {
		name = recipe.ArmorClass
	}
	if attributeID == 9 && recipe.AttackClass != "" {
		name = recipe.AttackClass
	}
	var named *int
	if name != "" {
		value, ok := datfile.EffectPackedAttackArmorClassID(name)
		if !ok {
			return 0, fmt.Errorf("%s unknown packed attack/armor class %q", kind, name)
		}
		named = &value
	}
	if numeric != nil && named != nil && *numeric != *named {
		return 0, fmt.Errorf("%s packed_type_id=%d conflicts with class %q (%d)", kind, *numeric, name, *named)
	}
	if numeric != nil {
		return *numeric, nil
	}
	if named != nil {
		return *named, nil
	}
	return 0, fmt.Errorf("%s needs packed_type_id, combat_class, armor_class, or attack_class", kind)
}

func PackEffectAttackArmorAmount(classID int, amount int) (float32, error) {
	if classID < 0 || classID > 127 {
		return 0, fmt.Errorf("packed class id %d outside supported range 0..127", classID)
	}
	if amount < -127 || amount > 255 {
		return 0, fmt.Errorf("packed amount %d outside supported range -127..255", amount)
	}
	packed := (classID << 8)
	if amount < 0 {
		packed = -packed + amount
	} else {
		packed += amount
	}
	return float32(packed), nil
}

func checkedEffectCommandTechID(kind string, techID *int) (int16, error) {
	if techID == nil {
		return 0, fmt.Errorf("%s needs tech_id", kind)
	}
	if *techID < 0 || *techID > math.MaxInt16 {
		return 0, fmt.Errorf("%s tech_id=%d outside int16 command operand range", kind, *techID)
	}
	return int16(*techID), nil
}

type techTreeConnectionDeletes struct {
	BuildingConnections []int
	UnitConnections     []int
	ResearchConnections []int
}

func patchTechTree(tree *datfile.TechTree, recipe TechTreePatchRecipe) ([]TechTreeRewriteReport, techTreeConnectionDeletes, error) {
	var reports []TechTreeRewriteReport
	var deletes techTreeConnectionDeletes
	for i, item := range recipe.CreateBuildingConnections {
		if item.From < 0 || item.From >= len(tree.BuildingConnections) {
			return nil, deletes, fmt.Errorf("tech_tree.create_building_connections[%d] from=%d outside building connection table", i, item.From)
		}
		if len(tree.BuildingConnections) >= math.MaxUint8 {
			return nil, deletes, fmt.Errorf("tech_tree.create_building_connections[%d] would exceed uint8 building connection count", i)
		}
		connection, err := buildCreatedBuildingConnection(len(tree.BuildingConnections), tree.BuildingConnections[item.From], item)
		if err != nil {
			return nil, deletes, fmt.Errorf("tech_tree.create_building_connections[%d]: %w", i, err)
		}
		tree.BuildingConnections = append(tree.BuildingConnections, connection)
	}
	for i, item := range recipe.CreateUnitConnections {
		if item.From < 0 || item.From >= len(tree.UnitConnections) {
			return nil, deletes, fmt.Errorf("tech_tree.create_unit_connections[%d] from=%d outside unit connection table", i, item.From)
		}
		connection, err := buildCreatedUnitConnection(len(tree.UnitConnections), tree.UnitConnections[item.From], item)
		if err != nil {
			return nil, deletes, fmt.Errorf("tech_tree.create_unit_connections[%d]: %w", i, err)
		}
		tree.UnitConnections = append(tree.UnitConnections, connection)
	}
	for i, item := range recipe.CreateResearchConnections {
		if item.From < 0 || item.From >= len(tree.ResearchConnections) {
			return nil, deletes, fmt.Errorf("tech_tree.create_research_connections[%d] from=%d outside research connection table", i, item.From)
		}
		if len(tree.ResearchConnections) >= math.MaxUint8 {
			return nil, deletes, fmt.Errorf("tech_tree.create_research_connections[%d] would exceed uint8 research connection count", i)
		}
		connection, err := buildCreatedResearchConnection(len(tree.ResearchConnections), tree.ResearchConnections[item.From], item)
		if err != nil {
			return nil, deletes, fmt.Errorf("tech_tree.create_research_connections[%d]: %w", i, err)
		}
		tree.ResearchConnections = append(tree.ResearchConnections, connection)
	}
	for i, rewrite := range recipe.Rewrites {
		report, err := applyTechTreeRewrite(tree, rewrite)
		if err != nil {
			return nil, deletes, fmt.Errorf("tech_tree.rewrites[%d]: %w", i, err)
		}
		reports = append(reports, report)
	}
	for i, patch := range recipe.BuildingConnections {
		if patch.Index < 0 || patch.Index >= len(tree.BuildingConnections) {
			return nil, deletes, fmt.Errorf("tech_tree.building_connections[%d] index=%d outside table", i, patch.Index)
		}
		connection := &tree.BuildingConnections[patch.Index]
		if err := applyBuildingConnectionPatch(connection, patch); err != nil {
			return nil, deletes, err
		}
	}
	for i, patch := range recipe.UnitConnections {
		if patch.Index < 0 || patch.Index >= len(tree.UnitConnections) {
			return nil, deletes, fmt.Errorf("tech_tree.unit_connections[%d] index=%d outside table", i, patch.Index)
		}
		connection := &tree.UnitConnections[patch.Index]
		if err := applyUnitConnectionPatch(connection, patch); err != nil {
			return nil, deletes, err
		}
	}
	for i, patch := range recipe.ResearchConnections {
		if patch.Index < 0 || patch.Index >= len(tree.ResearchConnections) {
			return nil, deletes, fmt.Errorf("tech_tree.research_connections[%d] index=%d outside table", i, patch.Index)
		}
		connection := &tree.ResearchConnections[patch.Index]
		if err := applyResearchConnectionPatch(connection, patch); err != nil {
			return nil, deletes, err
		}
	}
	deleted, err := deleteTechTreeConnectionRows("tech_tree.delete_building_connections", len(tree.BuildingConnections), recipe.DeleteBuildingConnections)
	if err != nil {
		return nil, deletes, err
	}
	for _, index := range deleted {
		tree.BuildingConnections = append(tree.BuildingConnections[:index], tree.BuildingConnections[index+1:]...)
	}
	deletes.BuildingConnections = deleted
	deleted, err = deleteTechTreeConnectionRows("tech_tree.delete_unit_connections", len(tree.UnitConnections), recipe.DeleteUnitConnections)
	if err != nil {
		return nil, deletes, err
	}
	for _, index := range deleted {
		tree.UnitConnections = append(tree.UnitConnections[:index], tree.UnitConnections[index+1:]...)
	}
	deletes.UnitConnections = deleted
	deleted, err = deleteTechTreeConnectionRows("tech_tree.delete_research_connections", len(tree.ResearchConnections), recipe.DeleteResearchConnections)
	if err != nil {
		return nil, deletes, err
	}
	for _, index := range deleted {
		tree.ResearchConnections = append(tree.ResearchConnections[:index], tree.ResearchConnections[index+1:]...)
	}
	deletes.ResearchConnections = deleted
	return reports, deletes, nil
}

func deleteTechTreeConnectionRows(label string, tableLen int, indexes []int) ([]int, error) {
	if len(indexes) == 0 {
		return nil, nil
	}
	out := append([]int(nil), indexes...)
	sort.Ints(out)
	for i, index := range out {
		if index < 0 || index >= tableLen {
			return nil, fmt.Errorf("%s[%d] index=%d outside table length %d", label, i, index, tableLen)
		}
		if i > 0 && index == out[i-1] {
			return nil, fmt.Errorf("%s contains duplicate index %d", label, index)
		}
	}
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out, nil
}

func buildCreatedBuildingConnection(index int, from datfile.BuildingConnection, recipe TechTreeBuildingConnectionCreate) (datfile.BuildingConnection, error) {
	connection := from
	connection.Index = index
	defaultID := int32(index)
	patch := TechTreeBuildingConnectionPatch{
		Index:            index,
		ID:               &defaultID,
		Status:           recipe.Status,
		Buildings:        recipe.Buildings,
		Units:            recipe.Units,
		Techs:            recipe.Techs,
		UnitResearch:     recipe.UnitResearch,
		Mode:             recipe.Mode,
		LocationInAge:    recipe.LocationInAge,
		UnitsTechsTotal:  recipe.UnitsTechsTotal,
		UnitsTechsFirst:  recipe.UnitsTechsFirst,
		LineMode:         recipe.LineMode,
		EnablingResearch: recipe.EnablingResearch,
	}
	if recipe.ID != nil {
		patch.ID = recipe.ID
	}
	if err := applyBuildingConnectionPatch(&connection, patch); err != nil {
		return datfile.BuildingConnection{}, err
	}
	return connection, nil
}

func applyBuildingConnectionPatch(connection *datfile.BuildingConnection, patch TechTreeBuildingConnectionPatch) error {
	if patch.ID != nil {
		connection.ID = *patch.ID
	}
	if patch.Status != nil {
		connection.Status = *patch.Status
	}
	if patch.Buildings != nil {
		if err := validateCountedI32List("building_connection.buildings", *patch.Buildings); err != nil {
			return err
		}
		connection.Buildings = append([]int32(nil), (*patch.Buildings)...)
	}
	if patch.Units != nil {
		if err := validateCountedI32List("building_connection.units", *patch.Units); err != nil {
			return err
		}
		connection.Units = append([]int32(nil), (*patch.Units)...)
	}
	if patch.Techs != nil {
		if err := validateCountedI32List("building_connection.techs", *patch.Techs); err != nil {
			return err
		}
		connection.Techs = append([]int32(nil), (*patch.Techs)...)
	}
	if patch.UnitResearch != nil {
		if err := validateFixedI32List("building_connection.unit_research", *patch.UnitResearch, 10); err != nil {
			return err
		}
		connection.Common.UnitResearch = append([]int32(nil), (*patch.UnitResearch)...)
	}
	if patch.Mode != nil {
		if err := validateFixedI32List("building_connection.mode", *patch.Mode, 10); err != nil {
			return err
		}
		connection.Common.Mode = append([]int32(nil), (*patch.Mode)...)
	}
	if patch.LocationInAge != nil {
		connection.LocationInAge = *patch.LocationInAge
	}
	if patch.UnitsTechsTotal != nil {
		if err := validateFixedU8List("building_connection.units_techs_total", *patch.UnitsTechsTotal, 5); err != nil {
			return err
		}
		connection.UnitsTechsTotal = append([]uint8(nil), (*patch.UnitsTechsTotal)...)
	}
	if patch.UnitsTechsFirst != nil {
		if err := validateFixedU8List("building_connection.units_techs_first", *patch.UnitsTechsFirst, 5); err != nil {
			return err
		}
		connection.UnitsTechsFirst = append([]uint8(nil), (*patch.UnitsTechsFirst)...)
	}
	if patch.LineMode != nil {
		connection.LineMode = *patch.LineMode
	}
	if patch.EnablingResearch != nil {
		connection.EnablingResearch = *patch.EnablingResearch
	}
	return nil
}

func buildCreatedUnitConnection(index int, from datfile.UnitConnection, recipe TechTreeUnitConnectionCreate) (datfile.UnitConnection, error) {
	connection := from
	connection.Index = index
	defaultID := int32(index)
	patch := TechTreeUnitConnectionPatch{
		Index:            index,
		ID:               &defaultID,
		Status:           recipe.Status,
		UpperBuilding:    recipe.UpperBuilding,
		UnitResearch:     recipe.UnitResearch,
		Mode:             recipe.Mode,
		VerticalLine:     recipe.VerticalLine,
		Units:            recipe.Units,
		LocationInAge:    recipe.LocationInAge,
		RequiredResearch: recipe.RequiredResearch,
		LineMode:         recipe.LineMode,
		EnablingResearch: recipe.EnablingResearch,
	}
	if recipe.ID != nil {
		patch.ID = recipe.ID
	}
	if err := applyUnitConnectionPatch(&connection, patch); err != nil {
		return datfile.UnitConnection{}, err
	}
	return connection, nil
}

func applyUnitConnectionPatch(connection *datfile.UnitConnection, patch TechTreeUnitConnectionPatch) error {
	if patch.ID != nil {
		connection.ID = *patch.ID
	}
	if patch.Status != nil {
		connection.Status = *patch.Status
	}
	if patch.UpperBuilding != nil {
		connection.UpperBuilding = *patch.UpperBuilding
	}
	if patch.UnitResearch != nil {
		if err := validateFixedI32List("unit_connection.unit_research", *patch.UnitResearch, 10); err != nil {
			return err
		}
		connection.Common.UnitResearch = append([]int32(nil), (*patch.UnitResearch)...)
	}
	if patch.Mode != nil {
		if err := validateFixedI32List("unit_connection.mode", *patch.Mode, 10); err != nil {
			return err
		}
		connection.Common.Mode = append([]int32(nil), (*patch.Mode)...)
	}
	if patch.VerticalLine != nil {
		connection.VerticalLine = *patch.VerticalLine
	}
	if patch.Units != nil {
		if err := validateCountedI32List("unit_connection.units", *patch.Units); err != nil {
			return err
		}
		connection.Units = append([]int32(nil), (*patch.Units)...)
	}
	if patch.LocationInAge != nil {
		connection.LocationInAge = *patch.LocationInAge
	}
	if patch.RequiredResearch != nil {
		connection.RequiredResearch = *patch.RequiredResearch
	}
	if patch.LineMode != nil {
		connection.LineMode = *patch.LineMode
	}
	if patch.EnablingResearch != nil {
		connection.EnablingResearch = *patch.EnablingResearch
	}
	return nil
}

func buildCreatedResearchConnection(index int, from datfile.ResearchConnection, recipe TechTreeResearchConnectionCreate) (datfile.ResearchConnection, error) {
	connection := from
	connection.Index = index
	defaultID := int32(index)
	patch := TechTreeResearchConnectionPatch{
		Index:         index,
		ID:            &defaultID,
		Status:        recipe.Status,
		UpperBuilding: recipe.UpperBuilding,
		Buildings:     recipe.Buildings,
		Units:         recipe.Units,
		Techs:         recipe.Techs,
		UnitResearch:  recipe.UnitResearch,
		Mode:          recipe.Mode,
		VerticalLine:  recipe.VerticalLine,
		LocationInAge: recipe.LocationInAge,
		LineMode:      recipe.LineMode,
	}
	if recipe.ID != nil {
		patch.ID = recipe.ID
	}
	if err := applyResearchConnectionPatch(&connection, patch); err != nil {
		return datfile.ResearchConnection{}, err
	}
	return connection, nil
}

func applyResearchConnectionPatch(connection *datfile.ResearchConnection, patch TechTreeResearchConnectionPatch) error {
	if patch.ID != nil {
		connection.ID = *patch.ID
	}
	if patch.Status != nil {
		connection.Status = *patch.Status
	}
	if patch.UpperBuilding != nil {
		connection.UpperBuilding = *patch.UpperBuilding
	}
	if patch.Buildings != nil {
		if err := validateCountedI32List("research_connection.buildings", *patch.Buildings); err != nil {
			return err
		}
		connection.Buildings = append([]int32(nil), (*patch.Buildings)...)
	}
	if patch.Units != nil {
		if err := validateCountedI32List("research_connection.units", *patch.Units); err != nil {
			return err
		}
		connection.Units = append([]int32(nil), (*patch.Units)...)
	}
	if patch.Techs != nil {
		if err := validateCountedI32List("research_connection.techs", *patch.Techs); err != nil {
			return err
		}
		connection.Techs = append([]int32(nil), (*patch.Techs)...)
	}
	if patch.UnitResearch != nil {
		if err := validateFixedI32List("research_connection.unit_research", *patch.UnitResearch, 10); err != nil {
			return err
		}
		connection.Common.UnitResearch = append([]int32(nil), (*patch.UnitResearch)...)
	}
	if patch.Mode != nil {
		if err := validateFixedI32List("research_connection.mode", *patch.Mode, 10); err != nil {
			return err
		}
		connection.Common.Mode = append([]int32(nil), (*patch.Mode)...)
	}
	if patch.VerticalLine != nil {
		connection.VerticalLine = *patch.VerticalLine
	}
	if patch.LocationInAge != nil {
		connection.LocationInAge = *patch.LocationInAge
	}
	if patch.LineMode != nil {
		connection.LineMode = *patch.LineMode
	}
	return nil
}

func applyTechTreeRewrite(tree *datfile.TechTree, recipe TechTreeIDRewriteRecipe) (TechTreeRewriteReport, error) {
	if recipe.From < 0 {
		return TechTreeRewriteReport{}, fmt.Errorf("from=%d is negative", recipe.From)
	}
	switch recipe.Target {
	case "tech", "unit":
	default:
		return TechTreeRewriteReport{}, fmt.Errorf("target=%q, want tech or unit", recipe.Target)
	}
	if recipe.Remove && recipe.To != nil {
		return TechTreeRewriteReport{}, errors.New("remove=true cannot also set to")
	}
	if !recipe.Remove && recipe.To == nil {
		return TechTreeRewriteReport{}, errors.New("rewrite needs to or remove=true")
	}
	to := int32(-1)
	if recipe.To != nil {
		if *recipe.To < 0 {
			return TechTreeRewriteReport{}, fmt.Errorf("to=%d is negative", *recipe.To)
		}
		if *recipe.To == recipe.From {
			return TechTreeRewriteReport{}, fmt.Errorf("to=%d equals from", *recipe.To)
		}
		to = *recipe.To
	}
	report := TechTreeRewriteReport{
		Target: recipe.Target,
		From:   recipe.From,
		To:     recipe.To,
		Remove: recipe.Remove,
	}
	rewriteScalar := func(value *int32) {
		if *value == recipe.From {
			*value = to
			report.ReplacedScalars++
		}
	}
	rewriteFixedList := func(values []int32) {
		for i := range values {
			if values[i] == recipe.From {
				values[i] = to
				report.ReplacedListValues++
			}
		}
	}
	rewriteCountedList := func(values []int32) []int32 {
		if recipe.Remove {
			out := values[:0]
			for _, value := range values {
				if value == recipe.From {
					report.RemovedListValues++
					continue
				}
				out = append(out, value)
			}
			return out
		}
		for i := range values {
			if values[i] == recipe.From {
				values[i] = to
				report.ReplacedListValues++
			}
		}
		return values
	}
	if recipe.Target == "tech" {
		for i := range tree.Ages {
			tree.Ages[i].Techs = rewriteCountedList(tree.Ages[i].Techs)
			rewriteFixedList(tree.Ages[i].Common.UnitResearch)
		}
		for i := range tree.BuildingConnections {
			tree.BuildingConnections[i].Techs = rewriteCountedList(tree.BuildingConnections[i].Techs)
			rewriteFixedList(tree.BuildingConnections[i].Common.UnitResearch)
			rewriteScalar(&tree.BuildingConnections[i].EnablingResearch)
		}
		for i := range tree.UnitConnections {
			rewriteFixedList(tree.UnitConnections[i].Common.UnitResearch)
			rewriteScalar(&tree.UnitConnections[i].RequiredResearch)
			rewriteScalar(&tree.UnitConnections[i].EnablingResearch)
		}
		for i := range tree.ResearchConnections {
			rewriteScalar(&tree.ResearchConnections[i].ID)
			tree.ResearchConnections[i].Techs = rewriteCountedList(tree.ResearchConnections[i].Techs)
			rewriteFixedList(tree.ResearchConnections[i].Common.UnitResearch)
		}
		return report, nil
	}
	for i := range tree.Ages {
		tree.Ages[i].Buildings = rewriteCountedList(tree.Ages[i].Buildings)
		tree.Ages[i].Units = rewriteCountedList(tree.Ages[i].Units)
	}
	for i := range tree.BuildingConnections {
		rewriteScalar(&tree.BuildingConnections[i].ID)
		tree.BuildingConnections[i].Buildings = rewriteCountedList(tree.BuildingConnections[i].Buildings)
		tree.BuildingConnections[i].Units = rewriteCountedList(tree.BuildingConnections[i].Units)
	}
	for i := range tree.UnitConnections {
		rewriteScalar(&tree.UnitConnections[i].ID)
		rewriteScalar(&tree.UnitConnections[i].UpperBuilding)
		tree.UnitConnections[i].Units = rewriteCountedList(tree.UnitConnections[i].Units)
	}
	for i := range tree.ResearchConnections {
		rewriteScalar(&tree.ResearchConnections[i].UpperBuilding)
		tree.ResearchConnections[i].Buildings = rewriteCountedList(tree.ResearchConnections[i].Buildings)
		tree.ResearchConnections[i].Units = rewriteCountedList(tree.ResearchConnections[i].Units)
	}
	return report, nil
}

type referenceRewriteResult struct {
	Effects []datfile.Effect
	Techs   []datfile.Tech
	Report  ReferenceRewriteReport
}

func applyReferenceRewrite(effects []datfile.Effect, techs []datfile.Tech, tree *datfile.TechTree, recipe TechTreeIDRewriteRecipe) (referenceRewriteResult, error) {
	treeReport, err := applyTechTreeRewrite(tree, recipe)
	if err != nil {
		return referenceRewriteResult{}, err
	}
	report := ReferenceRewriteReport{
		Target:   recipe.Target,
		From:     recipe.From,
		To:       recipe.To,
		Remove:   recipe.Remove,
		TechTree: &treeReport,
	}
	to := int32(-1)
	if recipe.To != nil {
		to = *recipe.To
	}
	if recipe.To != nil && recipe.Target == "tech" && to > math.MaxInt16 {
		return referenceRewriteResult{}, fmt.Errorf("tech rewrite to=%d exceeds int16 fields", to)
	}
	if recipe.To != nil && recipe.Target == "unit" && to > math.MaxInt16 {
		return referenceRewriteResult{}, fmt.Errorf("unit rewrite to=%d exceeds int16 effect-command fields", to)
	}
	if recipe.Target == "tech" {
		for i := range techs {
			for j := range techs[i].RequiredTechs {
				if int32(techs[i].RequiredTechs[j]) == recipe.From {
					techs[i].RequiredTechs[j] = int16(to)
					report.TechRequiredRefs++
				}
			}
		}
	}
	for i := range effects {
		nextCommands := effects[i].Commands[:0]
		for _, command := range effects[i].Commands {
			rewrittenCommand, matched, err := rewriteEffectCommandReference(command, recipe.Target, recipe.From, to)
			if err != nil {
				return referenceRewriteResult{}, err
			}
			if !matched {
				nextCommands = append(nextCommands, command)
				continue
			}
			if recipe.Remove {
				report.EffectCommandsRemoved++
				continue
			}
			report.EffectCommandsRewritten++
			nextCommands = append(nextCommands, rewrittenCommand)
		}
		effects[i].Commands = nextCommands
		for j := range effects[i].Commands {
			effects[i].Commands[j].Index = j
		}
		effects[i].CommandSize = len(effects[i].Commands)
	}
	return referenceRewriteResult{Effects: effects, Techs: techs, Report: report}, nil
}

func rewriteEffectCommandReference(command datfile.EffectCommand, target string, from, to int32) (datfile.EffectCommand, bool, error) {
	matched := false
	for _, ref := range effectCommandSemanticReferences(command) {
		if ref.Kind != target || int32(ref.ID) != from {
			continue
		}
		matched = true
		switch target {
		case "tech":
			switch ref.Field {
			case "amount":
				command.D = float32(to)
			case "target_tech":
				command.A = int16(to)
			default:
				return datfile.EffectCommand{}, false, fmt.Errorf("cannot rewrite tech effect-command field %q", ref.Field)
			}
		case "unit":
			switch ref.Field {
			case "source_unit", "spawn_unit":
				command.A = int16(to)
			case "spawn_building":
				command.B = int16(to)
			case "target_unit":
				if command.Type == 3 {
					command.B = int16(to)
				} else {
					command.A = int16(to)
				}
			default:
				return datfile.EffectCommand{}, false, fmt.Errorf("cannot rewrite unit effect-command field %q", ref.Field)
			}
		default:
			return datfile.EffectCommand{}, false, fmt.Errorf("cannot rewrite effect-command target %q", target)
		}
	}
	if !matched {
		return command, false, nil
	}
	command.Semantic = datfile.InterpretEffectCommand(command)
	return command, true, nil
}

func validateCountedI32List(name string, values []int32) error {
	if len(values) > math.MaxUint8 {
		return fmt.Errorf("%s length=%d exceeds u8", name, len(values))
	}
	return nil
}

func validateFixedI32List(name string, values []int32, want int) error {
	if len(values) != want {
		return fmt.Errorf("%s length=%d, want %d", name, len(values), want)
	}
	return nil
}

func validateFixedU8List(name string, values []uint8, want int) error {
	if len(values) != want {
		return fmt.Errorf("%s length=%d, want %d", name, len(values), want)
	}
	return nil
}

func buildCreatedTech(index int, template datfile.Tech, recipe TechCreateRecipe) (datfile.Tech, error) {
	tech := cloneTechs([]datfile.Tech{template})[0]
	tech.Index = index
	if err := applyTechCreatePatch(&tech, recipe); err != nil {
		return datfile.Tech{}, err
	}
	return tech, nil
}

func patchTech(tech datfile.Tech, recipe TechPatchRecipe) (datfile.Tech, error) {
	if recipe.Name != nil {
		tech.Name = *recipe.Name
	}
	if recipe.RequiredTechs != nil {
		if len(recipe.RequiredTechs) != 6 {
			return datfile.Tech{}, fmt.Errorf("required_techs length=%d, want 6", len(recipe.RequiredTechs))
		}
		tech.RequiredTechs = append([]int16(nil), recipe.RequiredTechs...)
	}
	if recipe.ResourceCosts != nil {
		if len(recipe.ResourceCosts) != 3 {
			return datfile.Tech{}, fmt.Errorf("resource_costs length=%d, want 3", len(recipe.ResourceCosts))
		}
		tech.ResourceCosts = append([]datfile.ResearchResourceCost(nil), recipe.ResourceCosts...)
	}
	if recipe.RequiredTechCount != nil {
		tech.RequiredTechCount = *recipe.RequiredTechCount
	}
	if recipe.Civ != nil {
		tech.Civ = *recipe.Civ
	}
	if recipe.FullTechMode != nil {
		tech.FullTechMode = *recipe.FullTechMode
	}
	if recipe.LanguageDLLName != nil {
		tech.LanguageDLLName = *recipe.LanguageDLLName
	}
	if recipe.LanguageDLLDescription != nil {
		tech.LanguageDLLDescription = *recipe.LanguageDLLDescription
	}
	if recipe.EffectID != nil {
		tech.EffectID = *recipe.EffectID
	}
	if recipe.Type != nil {
		tech.Type = *recipe.Type
	}
	if recipe.IconID != nil {
		tech.IconID = *recipe.IconID
	}
	if recipe.LanguageDLLHelp != nil {
		tech.LanguageDLLHelp = *recipe.LanguageDLLHelp
	}
	if recipe.LanguageDLLTechTree != nil {
		tech.LanguageDLLTechTree = *recipe.LanguageDLLTechTree
	}
	if recipe.Repeatable != nil {
		tech.Repeatable = *recipe.Repeatable
	}
	if recipe.ResearchLocations != nil {
		tech.ResearchLocations = append([]datfile.ResearchLocation(nil), recipe.ResearchLocations...)
	}
	return tech, nil
}

func applyTechCreatePatch(tech *datfile.Tech, recipe TechCreateRecipe) error {
	patched, err := patchTech(*tech, TechPatchRecipe{
		Name:                   recipe.Name,
		RequiredTechs:          recipe.RequiredTechs,
		ResourceCosts:          recipe.ResourceCosts,
		RequiredTechCount:      recipe.RequiredTechCount,
		Civ:                    recipe.Civ,
		FullTechMode:           recipe.FullTechMode,
		LanguageDLLName:        recipe.LanguageDLLName,
		LanguageDLLDescription: recipe.LanguageDLLDescription,
		EffectID:               recipe.EffectID,
		Type:                   recipe.Type,
		IconID:                 recipe.IconID,
		LanguageDLLHelp:        recipe.LanguageDLLHelp,
		LanguageDLLTechTree:    recipe.LanguageDLLTechTree,
		Repeatable:             recipe.Repeatable,
		ResearchLocations:      recipe.ResearchLocations,
	})
	if err != nil {
		return err
	}
	*tech = patched
	return nil
}

func deleteTechs(base *datfile.Index, effects []datfile.Effect, techs []datfile.Tech, tree datfile.TechTree, ids []int) ([]int, []TechDeleteReport, []datfile.Effect, []datfile.Tech, datfile.TechTree, []ReferenceRewriteReport, error) {
	deleted := append([]int(nil), ids...)
	sort.Sort(sort.Reverse(sort.IntSlice(deleted)))
	seen := make(map[int]bool, len(deleted))
	details := make([]TechDeleteReport, 0, len(deleted))
	var rewriteReports []ReferenceRewriteReport
	for _, id := range deleted {
		if seen[id] {
			return nil, nil, nil, nil, datfile.TechTree{}, nil, fmt.Errorf("delete_techs duplicate id=%d", id)
		}
		seen[id] = true
		if id < 0 || id >= len(techs) {
			return nil, nil, nil, nil, datfile.TechTree{}, nil, fmt.Errorf("delete_techs id=%d outside tech table length %d", id, len(techs))
		}
		current := *base
		current.Effects = effects
		current.Techs = techs
		current.TechTree = tree
		deleteRefs := techIDReferences(&current, id)
		if len(deleteRefs) > 0 {
			return nil, nil, nil, nil, datfile.TechTree{}, nil, fmt.Errorf("delete_techs id=%d is still referenced by %d decoded refs; remove or retarget references before physical delete", id, len(deleteRefs))
		}
		last := len(techs) - 1
		if id == last {
			techs = techs[:last]
			details = append(details, TechDeleteReport{
				TechID:   id,
				Strategy: "physical_tail_delete",
				Note:     "removed the unreferenced tail tech row; earlier tech IDs are unchanged",
			})
			continue
		}
		tailRefs := techIDReferences(&current, last)
		covered, unsupported := techReferenceRewriteCoverage(tailRefs)
		if unsupported > 0 {
			return nil, nil, nil, nil, datfile.TechTree{}, nil, fmt.Errorf("delete_techs id=%d cannot tail-swap tail tech id=%d: %d tail references are outside the verified rewrite surface", id, last, unsupported)
		}
		if covered > 0 {
			to := int32(id)
			rewrite := TechTreeIDRewriteRecipe{Target: "tech", From: int32(last), To: &to}
			rewriteResult, err := applyReferenceRewrite(effects, techs, &tree, rewrite)
			if err != nil {
				return nil, nil, nil, nil, datfile.TechTree{}, nil, fmt.Errorf("delete_techs id=%d tail migration: %w", id, err)
			}
			effects = rewriteResult.Effects
			techs = rewriteResult.Techs
			rewriteReports = append(rewriteReports, rewriteResult.Report)
		}
		movedName := techs[last].Name
		movedTailID := last
		techs[id] = techs[last]
		techs[id].Index = id
		techs = techs[:last]
		details = append(details, TechDeleteReport{
			TechID:            id,
			Strategy:          "physical_non_tail_tail_swap",
			MovedTailTechID:   &movedTailID,
			MovedTailTechName: movedName,
			ReferenceRewrites: covered,
			Note:              "moved the previous tail tech row into the deleted slot, rewrote verified tail-tech references to the deleted slot, then truncated the tail",
		})
		current = *base
		current.Effects = effects
		current.Techs = techs
		current.TechTree = tree
		if remaining := len(techIDReferences(&current, last)); remaining > 0 {
			return nil, nil, nil, nil, datfile.TechTree{}, nil, fmt.Errorf("delete_techs id=%d left %d refs to truncated tail tech id=%d after migration", id, remaining, last)
		}
	}
	sort.Ints(deleted)
	for i := range techs {
		techs[i].Index = i
	}
	sort.Slice(details, func(i, j int) bool { return details[i].TechID < details[j].TechID })
	return deleted, details, effects, techs, tree, rewriteReports, nil
}

func deleteEffects(effects []datfile.Effect, techs []datfile.Tech, ids []int) ([]int, []datfile.Effect, []datfile.Tech, error) {
	deleted := append([]int(nil), ids...)
	sort.Sort(sort.Reverse(sort.IntSlice(deleted)))
	for _, id := range deleted {
		if id < 0 || id >= len(effects) {
			return nil, nil, nil, fmt.Errorf("delete_effects id=%d outside effects table", id)
		}
		for _, tech := range techs {
			if int(tech.EffectID) == id {
				return nil, nil, nil, fmt.Errorf("delete_effects id=%d is still referenced by tech %d", id, tech.Index)
			}
		}
		effects = append(effects[:id], effects[id+1:]...)
		for i := range techs {
			if int(techs[i].EffectID) > id {
				techs[i].EffectID--
			}
		}
	}
	sort.Ints(deleted)
	for i := range effects {
		effects[i].Index = i
	}
	return deleted, effects, techs, nil
}

type payloadReplacement struct {
	Start int
	End   int
	Data  []byte
}

func rebuildCodecSections(payload []byte, idx *datfile.Index, effects []datfile.Effect, techs []datfile.Tech, replacements []payloadReplacement) ([]byte, error) {
	effectsStart, effectsEnd, err := sectionBounds(idx, "effects_header", "effect[", len(idx.Effects))
	if err != nil {
		return nil, err
	}
	techsStart, techsEnd, err := sectionBounds(idx, "techs_header", "tech[", len(idx.Techs))
	if err != nil {
		return nil, err
	}
	if effectsEnd > techsStart {
		return nil, fmt.Errorf("effects section %d..%d overlaps techs at %d", effectsStart, effectsEnd, techsStart)
	}
	effectsBytes, err := encodeEffectsSection(effects)
	if err != nil {
		return nil, err
	}
	techsBytes, err := encodeTechsSection(techs)
	if err != nil {
		return nil, err
	}
	replacements = append(replacements,
		payloadReplacement{Start: effectsStart, End: effectsEnd, Data: effectsBytes},
		payloadReplacement{Start: techsStart, End: techsEnd, Data: techsBytes},
	)
	return applyPayloadReplacements(payload, replacements)
}

func applyPayloadReplacements(payload []byte, replacements []payloadReplacement) ([]byte, error) {
	sort.Slice(replacements, func(i, j int) bool {
		if replacements[i].Start == replacements[j].Start {
			return replacements[i].End < replacements[j].End
		}
		return replacements[i].Start < replacements[j].Start
	})
	cursor := 0
	out := make([]byte, 0, len(payload))
	for i, replacement := range replacements {
		if replacement.Start < 0 || replacement.End < replacement.Start || replacement.End > len(payload) {
			return nil, fmt.Errorf("replacement %d has invalid span %d..%d for payload %d", i, replacement.Start, replacement.End, len(payload))
		}
		if replacement.Start < cursor {
			return nil, fmt.Errorf("replacement %d overlaps prior replacement: start=%d prior_end=%d", i, replacement.Start, cursor)
		}
		out = append(out, payload[cursor:replacement.Start]...)
		out = append(out, replacement.Data...)
		cursor = replacement.End
	}
	out = append(out, payload[cursor:]...)
	return out, nil
}

func patchUnitHeaderRecord(payload []byte, header datfile.UnitHeader, recipe UnitHeaderPatchRecipe) ([]byte, error) {
	if header.Span.Start < 0 || header.Span.End > len(payload) || header.Span.End < header.Span.Start {
		return nil, fmt.Errorf("invalid unit header span %d..%d", header.Span.Start, header.Span.End)
	}
	if len(recipe.Tasks) == 0 && recipe.SetTasks == nil {
		return nil, errors.New("unit header patch has no task rows")
	}
	if len(recipe.Tasks) > 0 && recipe.SetTasks != nil {
		return nil, errors.New("cannot combine set_tasks with indexed unit-header task patches")
	}
	if !header.Exists {
		return nil, errors.New("unit header is absent; no task rows can be patched")
	}
	record := append([]byte(nil), payload[header.Span.Start:header.Span.End]...)
	if recipe.SetTasks != nil {
		encoded, err := datfile.EncodeCurrentDETaskList(*recipe.SetTasks)
		if err != nil {
			return nil, err
		}
		out := make([]byte, 0, 1+len(encoded))
		out = append(out, record[0])
		out = append(out, encoded...)
		return out, nil
	}
	if _, err := patchTaskRowsInRecord(record, header.Span, header.Tasks, recipe.Tasks); err != nil {
		return nil, err
	}
	return record, nil
}

func patchTaskRowsInRecord(record []byte, recordSpan datfile.Span, rows []datfile.TaskSummary, patches []datfile.TaskPatch) ([]string, error) {
	changed := []string{}
	for _, item := range patches {
		if item.Index < 0 || item.Index >= len(rows) {
			return changed, fmt.Errorf("task index %d outside task count %d", item.Index, len(rows))
		}
		span := rows[item.Index].Span
		if item.RecordType != nil {
			if err := putI16Record(record, recordSpan, datfile.Span{Name: "record_type", Start: span.Start, End: span.Start + 2}, *item.RecordType); err != nil {
				return changed, err
			}
		}
		if item.ID != nil {
			if err := putI16Record(record, recordSpan, datfile.Span{Name: "id", Start: span.Start + 2, End: span.Start + 4}, *item.ID); err != nil {
				return changed, err
			}
		}
		if item.IsDefault != nil {
			if err := putU8Record(record, recordSpan, datfile.Span{Name: "is_default", Start: span.Start + 4, End: span.Start + 5}, *item.IsDefault); err != nil {
				return changed, err
			}
		}
		if item.ActionType != nil {
			if err := putI16Record(record, recordSpan, datfile.Span{Name: "action_type", Start: span.Start + 5, End: span.Start + 7}, *item.ActionType); err != nil {
				return changed, err
			}
		}
		if item.ObjectClass != nil {
			if err := putI16Record(record, recordSpan, datfile.Span{Name: "object_class", Start: span.Start + 7, End: span.Start + 9}, *item.ObjectClass); err != nil {
				return changed, err
			}
		}
		if item.ObjectID != nil {
			if err := putI16Record(record, recordSpan, datfile.Span{Name: "object_id", Start: span.Start + 9, End: span.Start + 11}, *item.ObjectID); err != nil {
				return changed, err
			}
		}
		if item.TerrainID != nil {
			if err := putI16Record(record, recordSpan, datfile.Span{Name: "terrain_id", Start: span.Start + 11, End: span.Start + 13}, *item.TerrainID); err != nil {
				return changed, err
			}
		}
		if item.AttributeTypes != nil {
			if len(item.AttributeTypes) != 4 {
				return changed, fmt.Errorf("task[%d] attribute_types length=%d, want 4", item.Index, len(item.AttributeTypes))
			}
			for i, value := range item.AttributeTypes {
				offset := span.Start + 13 + i*2
				if err := putI16Record(record, recordSpan, datfile.Span{Name: fmt.Sprintf("attribute_types[%d]", i), Start: offset, End: offset + 2}, value); err != nil {
					return changed, err
				}
			}
		}
		if item.WorkValue1 != nil {
			if err := putF32Record(record, recordSpan, datfile.Span{Name: "work_value_1", Start: span.Start + 21, End: span.Start + 25}, *item.WorkValue1); err != nil {
				return changed, err
			}
		}
		if item.WorkValue2 != nil {
			if err := putF32Record(record, recordSpan, datfile.Span{Name: "work_value_2", Start: span.Start + 25, End: span.Start + 29}, *item.WorkValue2); err != nil {
				return changed, err
			}
		}
		if item.WorkRange != nil {
			if err := putF32Record(record, recordSpan, datfile.Span{Name: "work_range", Start: span.Start + 29, End: span.Start + 33}, *item.WorkRange); err != nil {
				return changed, err
			}
		}
		if item.AutoSearchTargets != nil {
			if err := putU8Record(record, recordSpan, datfile.Span{Name: "auto_search_targets", Start: span.Start + 33, End: span.Start + 34}, *item.AutoSearchTargets); err != nil {
				return changed, err
			}
		}
		if item.SearchWaitTime != nil {
			if err := putF32Record(record, recordSpan, datfile.Span{Name: "search_wait_time", Start: span.Start + 34, End: span.Start + 38}, *item.SearchWaitTime); err != nil {
				return changed, err
			}
		}
		if item.EnableTargeting != nil {
			if err := putU8Record(record, recordSpan, datfile.Span{Name: "enable_targeting", Start: span.Start + 38, End: span.Start + 39}, *item.EnableTargeting); err != nil {
				return changed, err
			}
		}
		if item.CombatLevel != nil {
			if err := putU8Record(record, recordSpan, datfile.Span{Name: "combat_level", Start: span.Start + 39, End: span.Start + 40}, *item.CombatLevel); err != nil {
				return changed, err
			}
		}
		changed = append(changed, fmt.Sprintf("task[%d]", item.Index))
	}
	return changed, nil
}

func patchCivRecord(payload []byte, civ datfile.Civ, recipe CivPatchRecipe) ([]byte, error) {
	if civ.Span.Start < 0 || civ.Span.End > len(payload) || civ.Span.End < civ.Span.Start {
		return nil, fmt.Errorf("invalid civ span %d..%d", civ.Span.Start, civ.Span.End)
	}
	record := append([]byte(nil), payload[civ.Span.Start:civ.Span.End]...)
	if recipe.TechTreeID != nil {
		if err := putI16Record(record, civ.Span, civ.FieldSpans.TechTreeID, *recipe.TechTreeID); err != nil {
			return nil, err
		}
	}
	if recipe.TeamBonusID != nil {
		if err := putI16Record(record, civ.Span, civ.FieldSpans.TeamBonusID, *recipe.TeamBonusID); err != nil {
			return nil, err
		}
	}
	if recipe.IconSet != nil {
		if err := putU8Record(record, civ.Span, civ.FieldSpans.IconSet, *recipe.IconSet); err != nil {
			return nil, err
		}
	}
	for _, resource := range recipe.Resources {
		if resource.Index < 0 || resource.Index >= len(civ.Resources) {
			return nil, fmt.Errorf("resource index %d outside civ resource table length %d", resource.Index, len(civ.Resources))
		}
		span := datfile.Span{
			Name:  fmt.Sprintf("resource[%d]", resource.Index),
			Start: civ.FieldSpans.Resources.Start + resource.Index*4,
			End:   civ.FieldSpans.Resources.Start + resource.Index*4 + 4,
		}
		if err := putF32Record(record, civ.Span, span, resource.Value); err != nil {
			return nil, err
		}
	}
	if recipe.Name != nil {
		nameStart := civ.FieldSpans.Name.Start - civ.Span.Start
		nameEnd := civ.FieldSpans.Name.End - civ.Span.Start
		if nameStart < 0 || nameEnd < nameStart || nameEnd > len(record) {
			return nil, fmt.Errorf("invalid civ name span %d..%d within record length %d", nameStart, nameEnd, len(record))
		}
		var out bytes.Buffer
		out.Write(record[:nameStart])
		if err := writeDebugString(&out, *recipe.Name); err != nil {
			return nil, err
		}
		out.Write(record[nameEnd:])
		record = out.Bytes()
	}
	return record, nil
}

func patchTerrainRestrictionRecord(payload []byte, restriction datfile.TerrainRestriction, recipe TerrainRestrictionPatchRecipe) ([]byte, error) {
	if restriction.Span.Start < 0 || restriction.Span.End > len(payload) || restriction.Span.End < restriction.Span.Start {
		return nil, fmt.Errorf("invalid terrain restriction span %d..%d", restriction.Span.Start, restriction.Span.End)
	}
	record := append([]byte(nil), payload[restriction.Span.Start:restriction.Span.End]...)
	for _, row := range recipe.Rows {
		if row.TerrainID < 0 || row.TerrainID >= len(restriction.TerrainRows) {
			return nil, fmt.Errorf("terrain_id=%d outside terrain restriction row table length %d", row.TerrainID, len(restriction.TerrainRows))
		}
		terrainRow := restriction.TerrainRows[row.TerrainID]
		if err := putF32Record(record, restriction.Span, terrainRow.PassabilitySpan, row.Passability); err != nil {
			return nil, err
		}
	}
	return record, nil
}

func patchTerrainRecord(payload []byte, terrain datfile.Terrain, recipe TerrainPatchRecipe) ([]byte, error) {
	if terrain.Span.Start < 0 || terrain.Span.End > len(payload) || terrain.Span.End < terrain.Span.Start {
		return nil, fmt.Errorf("invalid terrain span %d..%d", terrain.Span.Start, terrain.Span.End)
	}
	record := append([]byte(nil), payload[terrain.Span.Start:terrain.Span.End]...)
	type debugReplacement struct {
		field string
		span  datfile.Span
		value *string
	}
	replacements := []debugReplacement{
		{field: "name", span: terrain.Name.Span, value: recipe.Name},
		{field: "name_2", span: terrain.Name2.Span, value: recipe.Name2},
		{field: "overlay_mask_name", span: terrain.OverlayMaskName.Span, value: recipe.OverlayMaskName},
	}
	sort.Slice(replacements, func(i, j int) bool {
		return replacements[i].span.Start > replacements[j].span.Start
	})
	for _, replacement := range replacements {
		if replacement.value == nil {
			continue
		}
		next, err := replaceDebugStringInRecord(record, terrain.Span, replacement.span, *replacement.value)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", replacement.field, err)
		}
		record = next
	}
	return record, nil
}

func buildCreatedSound(index int, template datfile.Sound, recipe SoundCreateRecipe) (datfile.Sound, error) {
	sound := cloneSounds([]datfile.Sound{template})[0]
	sound.Index = index
	sound.ID = int16(index)
	if recipe.SoundID != nil {
		sound.ID = *recipe.SoundID
	}
	if recipe.PlayDelay != nil {
		sound.PlayDelay = *recipe.PlayDelay
	}
	if recipe.CacheTime != nil {
		sound.CacheTime = *recipe.CacheTime
	}
	if recipe.TotalProbability != nil {
		sound.TotalProbability = *recipe.TotalProbability
	}
	if recipe.Items != nil {
		sound.Items = soundItemsFromRecipe(*recipe.Items)
	}
	for i := range sound.Items {
		sound.Items[i].Index = i
	}
	return sound, nil
}

func patchSound(sound datfile.Sound, recipe SoundPatchRecipe) (datfile.Sound, error) {
	patched := sound
	patched.Items = append([]datfile.SoundItem(nil), sound.Items...)
	if recipe.SoundID != nil {
		patched.ID = *recipe.SoundID
	}
	if recipe.PlayDelay != nil {
		patched.PlayDelay = *recipe.PlayDelay
	}
	if recipe.CacheTime != nil {
		patched.CacheTime = *recipe.CacheTime
	}
	if recipe.TotalProbability != nil {
		patched.TotalProbability = *recipe.TotalProbability
	}
	if len(recipe.RemoveItems) > 0 {
		items, err := removeSoundItems(patched.Items, recipe.RemoveItems)
		if err != nil {
			return datfile.Sound{}, err
		}
		patched.Items = items
	}
	for _, itemPatch := range recipe.Items {
		if itemPatch.Index < 0 || itemPatch.Index >= len(patched.Items) {
			return datfile.Sound{}, fmt.Errorf("item index=%d outside sound item table length %d", itemPatch.Index, len(patched.Items))
		}
		item := patched.Items[itemPatch.Index]
		if itemPatch.FileName != nil {
			item.FileName.Value = *itemPatch.FileName
		}
		if itemPatch.ResourceID != nil {
			item.ResourceID = *itemPatch.ResourceID
		}
		if itemPatch.Probability != nil {
			item.Probability = *itemPatch.Probability
		}
		if itemPatch.Civ != nil {
			item.Civ = *itemPatch.Civ
		}
		if itemPatch.IconSet != nil {
			item.IconSet = *itemPatch.IconSet
		}
		patched.Items[itemPatch.Index] = item
	}
	return patched, nil
}

func removeSoundItems(items []datfile.SoundItem, indices []int) ([]datfile.SoundItem, error) {
	remove := make(map[int]bool, len(indices))
	for _, index := range indices {
		if index < 0 || index >= len(items) {
			return nil, fmt.Errorf("remove_items index=%d outside sound item table length %d", index, len(items))
		}
		if remove[index] {
			return nil, fmt.Errorf("remove_items index=%d is duplicated", index)
		}
		remove[index] = true
	}
	out := make([]datfile.SoundItem, 0, len(items)-len(remove))
	for i, item := range items {
		if remove[i] {
			continue
		}
		item.Index = len(out)
		out = append(out, item)
	}
	return out, nil
}

func muteSound(sound datfile.Sound) (datfile.Sound, SoundDeleteReport) {
	patched := sound
	patched.TotalProbability = 0
	patched.Items = append([]datfile.SoundItem(nil), sound.Items...)
	for i := range patched.Items {
		patched.Items[i].Probability = 0
	}
	return patched, SoundDeleteReport{
		SoundID:                sound.Index,
		Strategy:               "semantic_mute",
		BeforeTotalProbability: sound.TotalProbability,
		AfterTotalProbability:  0,
		ItemCount:              len(sound.Items),
		Note:                   "semantic sound delete preserves stable sound IDs and sets total/item probabilities to 0; in-engine silence still needs live verification for each use case.",
	}
}

func patchPlayerColourRecord(payload []byte, colour datfile.PlayerColour, recipe PlayerColourPatchRecipe) ([]byte, error) {
	if colour.Span.Start < 0 || colour.Span.End > len(payload) || colour.Span.End < colour.Span.Start {
		return nil, fmt.Errorf("invalid player colour span %d..%d", colour.Span.Start, colour.Span.End)
	}
	record := append([]byte(nil), payload[colour.Span.Start:colour.Span.End]...)
	if len(record) != 36 {
		return nil, fmt.Errorf("player colour record width=%d, want 36", len(record))
	}
	fields := []struct {
		name  string
		value *int32
		slot  int
	}{
		{name: "colour_id", value: recipe.ColourID, slot: 0},
		{name: "base", value: recipe.Base, slot: 1},
		{name: "unit_outline_colour", value: recipe.UnitOutlineColour, slot: 2},
		{name: "selection_colour_1", value: recipe.SelectionColour1, slot: 3},
		{name: "selection_colour_2", value: recipe.SelectionColour2, slot: 4},
		{name: "minimap_colour_1", value: recipe.MinimapColour1, slot: 5},
		{name: "minimap_colour_2", value: recipe.MinimapColour2, slot: 6},
		{name: "minimap_colour_3", value: recipe.MinimapColour3, slot: 7},
		{name: "statistics_text_color", value: recipe.StatisticsTextColor, slot: 8},
	}
	for _, field := range fields {
		if field.value == nil {
			continue
		}
		offset := field.slot * 4
		if offset+4 > len(record) {
			return nil, fmt.Errorf("%s slot=%d outside player colour record", field.name, field.slot)
		}
		binary.LittleEndian.PutUint32(record[offset:offset+4], uint32(*field.value))
	}
	return record, nil
}

func buildCreatedPlayerColour(index int, from datfile.PlayerColour, recipe PlayerColourCreateRecipe) datfile.PlayerColour {
	colour := from
	colour.Index = index
	defaultID := int32(index)
	patch := PlayerColourPatchRecipe{
		ID:                  index,
		ColourID:            &defaultID,
		Base:                recipe.Base,
		UnitOutlineColour:   recipe.UnitOutlineColour,
		SelectionColour1:    recipe.SelectionColour1,
		SelectionColour2:    recipe.SelectionColour2,
		MinimapColour1:      recipe.MinimapColour1,
		MinimapColour2:      recipe.MinimapColour2,
		MinimapColour3:      recipe.MinimapColour3,
		StatisticsTextColor: recipe.StatisticsTextColor,
	}
	if recipe.ColourID != nil {
		patch.ColourID = recipe.ColourID
	}
	applyPlayerColourPatch(&colour, patch)
	return colour
}

func applyPlayerColourPatch(colour *datfile.PlayerColour, recipe PlayerColourPatchRecipe) {
	if recipe.ColourID != nil {
		colour.ID = *recipe.ColourID
	}
	if recipe.Base != nil {
		colour.Base = *recipe.Base
	}
	if recipe.UnitOutlineColour != nil {
		colour.UnitOutlineColour = *recipe.UnitOutlineColour
	}
	if recipe.SelectionColour1 != nil {
		colour.SelectionColour1 = *recipe.SelectionColour1
	}
	if recipe.SelectionColour2 != nil {
		colour.SelectionColour2 = *recipe.SelectionColour2
	}
	if recipe.MinimapColour1 != nil {
		colour.MinimapColour1 = *recipe.MinimapColour1
	}
	if recipe.MinimapColour2 != nil {
		colour.MinimapColour2 = *recipe.MinimapColour2
	}
	if recipe.MinimapColour3 != nil {
		colour.MinimapColour3 = *recipe.MinimapColour3
	}
	if recipe.StatisticsTextColor != nil {
		colour.StatisticsTextColor = *recipe.StatisticsTextColor
	}
}

func encodePlayerColoursSection(colours []datfile.PlayerColour) ([]byte, error) {
	if len(colours) > math.MaxInt16 {
		return nil, fmt.Errorf("player colour count=%d exceeds int16 header", len(colours))
	}
	out := make([]byte, 2, 2+36*len(colours))
	binary.LittleEndian.PutUint16(out[0:2], uint16(int16(len(colours))))
	for _, colour := range colours {
		out = appendPlayerColourRecord(out, colour)
	}
	return out, nil
}

func appendPlayerColourRecord(out []byte, colour datfile.PlayerColour) []byte {
	values := []int32{
		colour.ID,
		colour.Base,
		colour.UnitOutlineColour,
		colour.SelectionColour1,
		colour.SelectionColour2,
		colour.MinimapColour1,
		colour.MinimapColour2,
		colour.MinimapColour3,
		colour.StatisticsTextColor,
	}
	for _, value := range values {
		out = binary.LittleEndian.AppendUint32(out, uint32(value))
	}
	return out
}

func deletePlayerColourRows(idx *datfile.Index, colours []datfile.PlayerColour, ids []int) ([]PlayerColourDeleteReport, []datfile.PlayerColour, error) {
	if len(ids) == 0 {
		return nil, colours, nil
	}
	deleteIDs := append([]int(nil), ids...)
	sort.Ints(deleteIDs)
	for i, id := range deleteIDs {
		if id < 0 || id >= len(colours) {
			return nil, nil, fmt.Errorf("delete_player_colours[%d] id=%d outside player colour table length %d", i, id, len(colours))
		}
		if i > 0 && id == deleteIDs[i-1] {
			return nil, nil, fmt.Errorf("delete_player_colours contains duplicate id %d", id)
		}
	}
	for i, j := 0, len(deleteIDs)-1; i < j; i, j = i+1, j-1 {
		deleteIDs[i], deleteIDs[j] = deleteIDs[j], deleteIDs[i]
	}
	out := append([]datfile.PlayerColour(nil), colours...)
	reports := make([]PlayerColourDeleteReport, 0, len(deleteIDs))
	for _, id := range deleteIDs {
		if id != len(out)-1 {
			return nil, nil, fmt.Errorf("delete_player_colours id=%d is not current tail id=%d; refusing non-tail palette compaction", id, len(out)-1)
		}
		if len(playerColourIDReferences(idx, id)) > 0 {
			return nil, nil, fmt.Errorf("delete_player_colours id=%d is still referenced by graphics; patch those graphics first", id)
		}
		before := len(out)
		out = out[:len(out)-1]
		reports = append(reports, PlayerColourDeleteReport{
			ID:          id,
			BeforeCount: before,
			AfterCount:  len(out),
			Strategy:    "physical_tail_delete",
			Note:        "tail-only player-colour delete preserves all earlier palette IDs; non-tail palette compaction remains unsupported.",
		})
	}
	return reports, out, nil
}

func replaceDebugStringInRecord(record []byte, recordSpan, fieldSpan datfile.Span, value string) ([]byte, error) {
	start := fieldSpan.Start - recordSpan.Start
	end := fieldSpan.End - recordSpan.Start
	if start < 0 || end < start || end > len(record) {
		return nil, fmt.Errorf("field span %d..%d outside record %d..%d length %d", fieldSpan.Start, fieldSpan.End, recordSpan.Start, recordSpan.End, len(record))
	}
	var encoded bytes.Buffer
	if err := writeDebugString(&encoded, value); err != nil {
		return nil, err
	}
	out := make([]byte, 0, len(record)+encoded.Len()-(end-start))
	out = append(out, record[:start]...)
	out = append(out, encoded.Bytes()...)
	out = append(out, record[end:]...)
	return out, nil
}

func encodeSound(sound datfile.Sound) ([]byte, error) {
	if len(sound.Items) > math.MaxInt16 {
		return nil, fmt.Errorf("sound item count %d exceeds int16", len(sound.Items))
	}
	var out bytes.Buffer
	writeI16(&out, sound.ID)
	writeI16(&out, sound.PlayDelay)
	writeI16(&out, int16(len(sound.Items)))
	writeI32(&out, sound.CacheTime)
	writeI16(&out, sound.TotalProbability)
	for i, item := range sound.Items {
		if err := writeDebugString(&out, item.FileName.Value); err != nil {
			return nil, fmt.Errorf("item[%d] file_name: %w", i, err)
		}
		writeI32(&out, item.ResourceID)
		writeI16(&out, item.Probability)
		writeI16(&out, item.Civ)
		writeI16(&out, item.IconSet)
	}
	return out.Bytes(), nil
}

func encodeSoundsSection(sounds []datfile.Sound) ([]byte, error) {
	if len(sounds) > math.MaxInt16 {
		return nil, fmt.Errorf("sound count %d exceeds int16", len(sounds))
	}
	var out bytes.Buffer
	writeI16(&out, int16(len(sounds)))
	for i, sound := range sounds {
		record, err := encodeSound(sound)
		if err != nil {
			return nil, fmt.Errorf("sound[%d]: %w", i, err)
		}
		out.Write(record)
	}
	return out.Bytes(), nil
}

func soundItemsFromRecipe(in []SoundItemRecipe) []datfile.SoundItem {
	out := make([]datfile.SoundItem, 0, len(in))
	for i, item := range in {
		out = append(out, datfile.SoundItem{
			Index:       i,
			FileName:    datfile.DebugString{Value: item.FileName},
			ResourceID:  item.ResourceID,
			Probability: item.Probability,
			Civ:         item.Civ,
			IconSet:     item.IconSet,
		})
	}
	return out
}

func deleteUnitRecords(payload []byte, idx *datfile.Index, recipe UnitDeleteRecipe) ([]payloadReplacement, []UnitDeleteReport, error) {
	strategy := recipe.Strategy
	if strategy == "" {
		strategy = "semantic_disable"
	}
	if strategy != "semantic_disable" && strategy != "disable" {
		return nil, nil, fmt.Errorf("unsupported strategy %q; supported strategy is semantic_disable", recipe.Strategy)
	}
	if recipe.UnitID < 0 {
		return nil, nil, fmt.Errorf("negative unit_id %d", recipe.UnitID)
	}
	if recipe.AllCivs && recipe.CivID != nil {
		return nil, nil, errors.New("set either all_civs or civ_id, not both")
	}
	var targets []datfile.UnitSummary
	if recipe.AllCivs {
		for _, civ := range idx.Civs {
			if recipe.UnitID >= len(civ.Units) {
				continue
			}
			unit := civ.Units[recipe.UnitID]
			if unit.Present {
				targets = append(targets, unit)
			}
		}
	} else {
		if recipe.CivID == nil {
			return nil, nil, errors.New("semantic unit delete needs civ_id or all_civs=true")
		}
		if *recipe.CivID < 0 || *recipe.CivID >= len(idx.Civs) {
			return nil, nil, fmt.Errorf("civ_id=%d outside civ table", *recipe.CivID)
		}
		civ := idx.Civs[*recipe.CivID]
		if recipe.UnitID >= len(civ.Units) {
			return nil, nil, fmt.Errorf("unit_id=%d outside civ %d unit table length %d", recipe.UnitID, *recipe.CivID, len(civ.Units))
		}
		unit := civ.Units[recipe.UnitID]
		if !unit.Present {
			return nil, nil, fmt.Errorf("unit_id=%d is not present for civ %d", recipe.UnitID, *recipe.CivID)
		}
		targets = append(targets, unit)
	}
	if len(targets) == 0 {
		return nil, nil, fmt.Errorf("no present unit records matched unit_id=%d", recipe.UnitID)
	}
	replacements := make([]payloadReplacement, 0, len(targets))
	reports := make([]UnitDeleteReport, 0, len(targets))
	for _, unit := range targets {
		recordSpan := datfile.Span{Name: fmt.Sprintf("civ[%d].unit[%d]", unit.CivIndex, unit.Index), Start: unit.RecordStart, End: unit.RecordEnd}
		if recordSpan.Start < 0 || recordSpan.End > len(payload) || recordSpan.End < recordSpan.Start {
			return nil, nil, fmt.Errorf("invalid unit span civ=%d unit=%d %d..%d", unit.CivIndex, unit.Index, recordSpan.Start, recordSpan.End)
		}
		record := append([]byte(nil), payload[recordSpan.Start:recordSpan.End]...)
		if err := putU8Record(record, recordSpan, unit.FieldSpans.Enabled, 0); err != nil {
			return nil, nil, err
		}
		replacements = append(replacements, payloadReplacement{Start: recordSpan.Start, End: recordSpan.End, Data: record})
		reports = append(reports, UnitDeleteReport{
			CivID:         unit.CivIndex,
			UnitID:        unit.Index,
			Strategy:      "semantic_disable",
			BeforeEnabled: unit.Enabled,
			AfterEnabled:  0,
			Note:          "semantic delete preserves stable unit IDs and sets enabled=0; scenario triggers that create this exact unit ID may still need scenario-side edits.",
		})
	}
	return replacements, reports, nil
}

func setUnitAvailabilityRecords(payload []byte, idx *datfile.Index, recipe UnitAvailabilityRecipe) ([]payloadReplacement, []UnitAvailabilityReport, error) {
	if recipe.UnitID < 0 {
		return nil, nil, fmt.Errorf("negative unit_id %d", recipe.UnitID)
	}
	targets, err := unitAvailabilityTargets(idx, recipe)
	if err != nil {
		return nil, nil, err
	}
	if len(targets) == 0 {
		return nil, nil, fmt.Errorf("no present unit records matched unit_id=%d", recipe.UnitID)
	}
	enabled := uint8(0)
	if recipe.Enabled {
		enabled = 1
	}
	replacements := make([]payloadReplacement, 0, len(targets))
	reports := make([]UnitAvailabilityReport, 0, len(targets))
	for _, unit := range targets {
		recordSpan := datfile.Span{Name: fmt.Sprintf("civ[%d].unit[%d]", unit.CivIndex, unit.Index), Start: unit.RecordStart, End: unit.RecordEnd}
		if recordSpan.Start < 0 || recordSpan.End > len(payload) || recordSpan.End < recordSpan.Start {
			return nil, nil, fmt.Errorf("invalid unit span civ=%d unit=%d %d..%d", unit.CivIndex, unit.Index, recordSpan.Start, recordSpan.End)
		}
		record := append([]byte(nil), payload[recordSpan.Start:recordSpan.End]...)
		if err := putU8Record(record, recordSpan, unit.FieldSpans.Enabled, enabled); err != nil {
			return nil, nil, err
		}
		replacements = append(replacements, payloadReplacement{Start: recordSpan.Start, End: recordSpan.End, Data: record})
		reports = append(reports, UnitAvailabilityReport{
			CivID:         unit.CivIndex,
			UnitID:        unit.Index,
			Present:       true,
			BeforeEnabled: unit.Enabled,
			AfterEnabled:  enabled,
			Strategy:      "set_unit_enabled_flag",
			Note:          "This changes the unit record enabled flag only; trainability can still depend on train locations, tech-tree links, scenario triggers, civ bonuses, and in-engine behavior.",
		})
	}
	return replacements, reports, nil
}

func unitAvailabilityTargets(idx *datfile.Index, recipe UnitAvailabilityRecipe) ([]datfile.UnitSummary, error) {
	selectorCount := 0
	if recipe.AllCivs {
		selectorCount++
	}
	if recipe.CivID != nil {
		selectorCount++
	}
	if len(recipe.CivIDs) > 0 {
		selectorCount++
	}
	if selectorCount != 1 {
		return nil, errors.New("unit_availability needs exactly one of civ_id, civ_ids, or all_civs=true")
	}
	if recipe.AllCivs {
		var targets []datfile.UnitSummary
		for _, civ := range idx.Civs {
			if recipe.UnitID >= len(civ.Units) {
				continue
			}
			unit := civ.Units[recipe.UnitID]
			if unit.Present {
				targets = append(targets, unit)
			}
		}
		return targets, nil
	}
	var civIDs []int
	if recipe.CivID != nil {
		civIDs = []int{*recipe.CivID}
	} else {
		civIDs = append([]int(nil), recipe.CivIDs...)
	}
	targets := make([]datfile.UnitSummary, 0, len(civIDs))
	seen := make(map[int]bool, len(civIDs))
	for _, civID := range civIDs {
		if seen[civID] {
			return nil, fmt.Errorf("duplicate civ_id=%d", civID)
		}
		seen[civID] = true
		if civID < 0 || civID >= len(idx.Civs) {
			return nil, fmt.Errorf("civ_id=%d outside civ table", civID)
		}
		civ := idx.Civs[civID]
		if recipe.UnitID >= len(civ.Units) {
			return nil, fmt.Errorf("unit_id=%d outside civ %d unit table length %d", recipe.UnitID, civID, len(civ.Units))
		}
		unit := civ.Units[recipe.UnitID]
		if !unit.Present {
			return nil, fmt.Errorf("unit_id=%d is not present for civ %d", recipe.UnitID, civID)
		}
		targets = append(targets, unit)
	}
	return targets, nil
}

func verifyUnitHeaderReadback(idx *datfile.Index, recipes []UnitHeaderPatchRecipe) error {
	for i, recipe := range recipes {
		if recipe.ID < 0 || recipe.ID >= len(idx.UnitHeaders) {
			return fmt.Errorf("unit_headers[%d] id=%d outside readback unit header table", i, recipe.ID)
		}
		header := idx.UnitHeaders[recipe.ID]
		if !header.Exists {
			return fmt.Errorf("unit_headers[%d] id=%d is absent after patch", i, recipe.ID)
		}
		if recipe.SetTasks != nil {
			if len(header.Tasks) != len(*recipe.SetTasks) {
				return fmt.Errorf("unit_headers[%d] set_tasks readback length got %d want %d", i, len(header.Tasks), len(*recipe.SetTasks))
			}
			for j, want := range *recipe.SetTasks {
				if err := verifyTaskRowReadback(header.Tasks[j], want, fmt.Sprintf("unit_headers[%d].set_tasks[%d]", i, j)); err != nil {
					return err
				}
			}
		}
		for _, taskPatch := range recipe.Tasks {
			if taskPatch.Index < 0 || taskPatch.Index >= len(header.Tasks) {
				return fmt.Errorf("unit_headers[%d] task[%d] outside readback task count %d", i, taskPatch.Index, len(header.Tasks))
			}
			if err := verifyTaskPatchReadback(header.Tasks[taskPatch.Index], taskPatch, fmt.Sprintf("unit_headers[%d].task[%d]", i, taskPatch.Index)); err != nil {
				return err
			}
		}
	}
	return nil
}

func verifyTaskRowReadback(row datfile.TaskSummary, want datfile.TaskRow, label string) error {
	if row.RecordType != want.RecordType ||
		row.ID != want.ID ||
		row.IsDefault != want.IsDefault ||
		row.ActionType != want.ActionType ||
		row.ObjectClass != want.ObjectClass ||
		row.ObjectID != want.ObjectID ||
		row.TerrainID != want.TerrainID ||
		row.WorkValue1 != want.WorkValue1 ||
		row.WorkValue2 != want.WorkValue2 ||
		row.WorkRange != want.WorkRange ||
		row.AutoSearchTargets != want.AutoSearchTargets ||
		row.SearchWaitTime != want.SearchWaitTime ||
		row.EnableTargeting != want.EnableTargeting ||
		row.CombatLevel != want.CombatLevel {
		return fmt.Errorf("%s readback got %+v want %+v", label, row, want)
	}
	if len(row.AttributeTypes) != len(want.AttributeTypes) {
		return fmt.Errorf("%s.attribute_types readback length got %d want %d", label, len(row.AttributeTypes), len(want.AttributeTypes))
	}
	for i := range want.AttributeTypes {
		if row.AttributeTypes[i] != want.AttributeTypes[i] {
			return fmt.Errorf("%s.attribute_types[%d] readback got %d want %d", label, i, row.AttributeTypes[i], want.AttributeTypes[i])
		}
	}
	if !bytes.Equal(row.RawTail, want.RawTail) {
		return fmt.Errorf("%s.raw_tail readback mismatch", label)
	}
	return nil
}

func verifyTaskPatchReadback(row datfile.TaskSummary, item datfile.TaskPatch, label string) error {
	if item.RecordType != nil && row.RecordType != *item.RecordType {
		return fmt.Errorf("%s.record_type readback got %d want %d", label, row.RecordType, *item.RecordType)
	}
	if item.ID != nil && row.ID != *item.ID {
		return fmt.Errorf("%s.id readback got %d want %d", label, row.ID, *item.ID)
	}
	if item.IsDefault != nil && row.IsDefault != *item.IsDefault {
		return fmt.Errorf("%s.is_default readback got %d want %d", label, row.IsDefault, *item.IsDefault)
	}
	if item.ActionType != nil && row.ActionType != *item.ActionType {
		return fmt.Errorf("%s.action_type readback got %d want %d", label, row.ActionType, *item.ActionType)
	}
	if item.ObjectClass != nil && row.ObjectClass != *item.ObjectClass {
		return fmt.Errorf("%s.object_class readback got %d want %d", label, row.ObjectClass, *item.ObjectClass)
	}
	if item.ObjectID != nil && row.ObjectID != *item.ObjectID {
		return fmt.Errorf("%s.object_id readback got %d want %d", label, row.ObjectID, *item.ObjectID)
	}
	if item.TerrainID != nil && row.TerrainID != *item.TerrainID {
		return fmt.Errorf("%s.terrain_id readback got %d want %d", label, row.TerrainID, *item.TerrainID)
	}
	if item.AttributeTypes != nil {
		if len(item.AttributeTypes) != len(row.AttributeTypes) {
			return fmt.Errorf("%s.attribute_types readback length got %d want %d", label, len(row.AttributeTypes), len(item.AttributeTypes))
		}
		for i, want := range item.AttributeTypes {
			if row.AttributeTypes[i] != want {
				return fmt.Errorf("%s.attribute_types[%d] readback got %d want %d", label, i, row.AttributeTypes[i], want)
			}
		}
	}
	if item.WorkValue1 != nil && row.WorkValue1 != *item.WorkValue1 {
		return fmt.Errorf("%s.work_value_1 readback got %g want %g", label, row.WorkValue1, *item.WorkValue1)
	}
	if item.WorkValue2 != nil && row.WorkValue2 != *item.WorkValue2 {
		return fmt.Errorf("%s.work_value_2 readback got %g want %g", label, row.WorkValue2, *item.WorkValue2)
	}
	if item.WorkRange != nil && row.WorkRange != *item.WorkRange {
		return fmt.Errorf("%s.work_range readback got %g want %g", label, row.WorkRange, *item.WorkRange)
	}
	if item.AutoSearchTargets != nil && row.AutoSearchTargets != *item.AutoSearchTargets {
		return fmt.Errorf("%s.auto_search_targets readback got %d want %d", label, row.AutoSearchTargets, *item.AutoSearchTargets)
	}
	if item.SearchWaitTime != nil && row.SearchWaitTime != *item.SearchWaitTime {
		return fmt.Errorf("%s.search_wait_time readback got %g want %g", label, row.SearchWaitTime, *item.SearchWaitTime)
	}
	if item.EnableTargeting != nil && row.EnableTargeting != *item.EnableTargeting {
		return fmt.Errorf("%s.enable_targeting readback got %d want %d", label, row.EnableTargeting, *item.EnableTargeting)
	}
	if item.CombatLevel != nil && row.CombatLevel != *item.CombatLevel {
		return fmt.Errorf("%s.combat_level readback got %d want %d", label, row.CombatLevel, *item.CombatLevel)
	}
	return nil
}

func verifyCivReadback(idx *datfile.Index, recipes []CivPatchRecipe) error {
	for i, recipe := range recipes {
		if recipe.ID < 0 || recipe.ID >= len(idx.Civs) {
			return fmt.Errorf("civs[%d] id=%d outside readback civ table", i, recipe.ID)
		}
		civ := idx.Civs[recipe.ID]
		if recipe.Name != nil && civ.Name != *recipe.Name {
			return fmt.Errorf("civs[%d] name readback got %q want %q", i, civ.Name, *recipe.Name)
		}
		if recipe.TechTreeID != nil && civ.TechTreeID != *recipe.TechTreeID {
			return fmt.Errorf("civs[%d] tech_tree_id readback got %d want %d", i, civ.TechTreeID, *recipe.TechTreeID)
		}
		if recipe.TeamBonusID != nil && civ.TeamBonusID != *recipe.TeamBonusID {
			return fmt.Errorf("civs[%d] team_bonus_id readback got %d want %d", i, civ.TeamBonusID, *recipe.TeamBonusID)
		}
		if recipe.IconSet != nil && civ.IconSet != *recipe.IconSet {
			return fmt.Errorf("civs[%d] icon_set readback got %d want %d", i, civ.IconSet, *recipe.IconSet)
		}
		for _, resource := range recipe.Resources {
			if resource.Index < 0 || resource.Index >= len(civ.Resources) {
				return fmt.Errorf("civs[%d] resource index %d outside readback resource table length %d", i, resource.Index, len(civ.Resources))
			}
			if civ.Resources[resource.Index] != resource.Value {
				return fmt.Errorf("civs[%d] resource[%d] readback got %g want %g", i, resource.Index, civ.Resources[resource.Index], resource.Value)
			}
		}
	}
	return nil
}

func verifyTerrainRestrictionReadback(idx *datfile.Index, recipes []TerrainRestrictionPatchRecipe) error {
	for i, recipe := range recipes {
		if recipe.ID < 0 || recipe.ID >= len(idx.TerrainRestrictions) {
			return fmt.Errorf("terrain_restrictions[%d] id=%d outside readback terrain restriction table", i, recipe.ID)
		}
		restriction := idx.TerrainRestrictions[recipe.ID]
		for _, row := range recipe.Rows {
			if row.TerrainID < 0 || row.TerrainID >= len(restriction.TerrainRows) {
				return fmt.Errorf("terrain_restrictions[%d] terrain_id=%d outside readback row table length %d", i, row.TerrainID, len(restriction.TerrainRows))
			}
			got := restriction.TerrainRows[row.TerrainID].Passability
			if got != row.Passability {
				return fmt.Errorf("terrain_restrictions[%d] terrain[%d] passability readback got %g want %g", i, row.TerrainID, got, row.Passability)
			}
		}
	}
	return nil
}

func verifyTerrainReadback(idx *datfile.Index, recipes []TerrainPatchRecipe) error {
	for i, recipe := range recipes {
		if recipe.ID < 0 || recipe.ID >= len(idx.Terrains) {
			return fmt.Errorf("terrains[%d] id=%d outside readback terrain table", i, recipe.ID)
		}
		terrain := idx.Terrains[recipe.ID]
		if recipe.Name != nil && terrain.Name.Value != *recipe.Name {
			return fmt.Errorf("terrains[%d] name readback got %q want %q", i, terrain.Name.Value, *recipe.Name)
		}
		if recipe.Name2 != nil && terrain.Name2.Value != *recipe.Name2 {
			return fmt.Errorf("terrains[%d] name_2 readback got %q want %q", i, terrain.Name2.Value, *recipe.Name2)
		}
		if recipe.OverlayMaskName != nil && terrain.OverlayMaskName.Value != *recipe.OverlayMaskName {
			return fmt.Errorf("terrains[%d] overlay_mask_name readback got %q want %q", i, terrain.OverlayMaskName.Value, *recipe.OverlayMaskName)
		}
	}
	return nil
}

func verifySoundReadback(idx *datfile.Index, recipes []SoundPatchRecipe) error {
	for i, recipe := range recipes {
		if recipe.ID < 0 || recipe.ID >= len(idx.Sounds) {
			return fmt.Errorf("sounds[%d] id=%d outside readback sound table", i, recipe.ID)
		}
		sound := idx.Sounds[recipe.ID]
		if recipe.SoundID != nil && sound.ID != *recipe.SoundID {
			return fmt.Errorf("sounds[%d] sound_id readback got %d want %d", i, sound.ID, *recipe.SoundID)
		}
		if recipe.PlayDelay != nil && sound.PlayDelay != *recipe.PlayDelay {
			return fmt.Errorf("sounds[%d] play_delay readback got %d want %d", i, sound.PlayDelay, *recipe.PlayDelay)
		}
		if recipe.CacheTime != nil && sound.CacheTime != *recipe.CacheTime {
			return fmt.Errorf("sounds[%d] cache_time readback got %d want %d", i, sound.CacheTime, *recipe.CacheTime)
		}
		if recipe.TotalProbability != nil && sound.TotalProbability != *recipe.TotalProbability {
			return fmt.Errorf("sounds[%d] total_probability readback got %d want %d", i, sound.TotalProbability, *recipe.TotalProbability)
		}
		for _, itemPatch := range recipe.Items {
			if itemPatch.Index < 0 || itemPatch.Index >= len(sound.Items) {
				return fmt.Errorf("sounds[%d] item index=%d outside readback sound item table length %d", i, itemPatch.Index, len(sound.Items))
			}
			item := sound.Items[itemPatch.Index]
			if itemPatch.FileName != nil && item.FileName.Value != *itemPatch.FileName {
				return fmt.Errorf("sounds[%d].items[%d] file_name readback got %q want %q", i, itemPatch.Index, item.FileName.Value, *itemPatch.FileName)
			}
			if itemPatch.ResourceID != nil && item.ResourceID != *itemPatch.ResourceID {
				return fmt.Errorf("sounds[%d].items[%d] resource_id readback got %d want %d", i, itemPatch.Index, item.ResourceID, *itemPatch.ResourceID)
			}
			if itemPatch.Probability != nil && item.Probability != *itemPatch.Probability {
				return fmt.Errorf("sounds[%d].items[%d] probability readback got %d want %d", i, itemPatch.Index, item.Probability, *itemPatch.Probability)
			}
			if itemPatch.Civ != nil && item.Civ != *itemPatch.Civ {
				return fmt.Errorf("sounds[%d].items[%d] civ readback got %d want %d", i, itemPatch.Index, item.Civ, *itemPatch.Civ)
			}
			if itemPatch.IconSet != nil && item.IconSet != *itemPatch.IconSet {
				return fmt.Errorf("sounds[%d].items[%d] icon_set readback got %d want %d", i, itemPatch.Index, item.IconSet, *itemPatch.IconSet)
			}
		}
	}
	return nil
}

func verifyCreatedSoundReadback(idx *datfile.Index, recipes []SoundCreateRecipe, created []int) error {
	if len(recipes) != len(created) {
		return fmt.Errorf("created sound report length=%d, want recipes length %d", len(created), len(recipes))
	}
	for i, recipe := range recipes {
		id := created[i]
		if id < 0 || id >= len(idx.Sounds) {
			return fmt.Errorf("create_sounds[%d] id=%d outside readback sound table", i, id)
		}
		sound := idx.Sounds[id]
		wantSoundID := int16(id)
		if recipe.SoundID != nil {
			wantSoundID = *recipe.SoundID
		}
		if sound.ID != wantSoundID {
			return fmt.Errorf("create_sounds[%d] sound_id readback got %d want %d", i, sound.ID, wantSoundID)
		}
		if recipe.PlayDelay != nil && sound.PlayDelay != *recipe.PlayDelay {
			return fmt.Errorf("create_sounds[%d] play_delay readback got %d want %d", i, sound.PlayDelay, *recipe.PlayDelay)
		}
		if recipe.CacheTime != nil && sound.CacheTime != *recipe.CacheTime {
			return fmt.Errorf("create_sounds[%d] cache_time readback got %d want %d", i, sound.CacheTime, *recipe.CacheTime)
		}
		if recipe.TotalProbability != nil && sound.TotalProbability != *recipe.TotalProbability {
			return fmt.Errorf("create_sounds[%d] total_probability readback got %d want %d", i, sound.TotalProbability, *recipe.TotalProbability)
		}
		if recipe.Items != nil {
			if len(sound.Items) != len(*recipe.Items) {
				return fmt.Errorf("create_sounds[%d] item count readback got %d want %d", i, len(sound.Items), len(*recipe.Items))
			}
			for j, want := range *recipe.Items {
				got := sound.Items[j]
				if got.Index != j {
					return fmt.Errorf("create_sounds[%d].items[%d] index readback got %d want %d", i, j, got.Index, j)
				}
				if got.FileName.Value != want.FileName ||
					got.ResourceID != want.ResourceID ||
					got.Probability != want.Probability ||
					got.Civ != want.Civ ||
					got.IconSet != want.IconSet {
					return fmt.Errorf("create_sounds[%d].items[%d] readback got %+v want %+v", i, j, got, want)
				}
			}
		}
	}
	return nil
}

func verifySoundDeletesReadback(idx *datfile.Index, reports []SoundDeleteReport) error {
	for i, report := range reports {
		if report.SoundID < 0 || report.SoundID >= len(idx.Sounds) {
			return fmt.Errorf("deleted_sounds[%d] sound_id=%d outside readback sound table", i, report.SoundID)
		}
		sound := idx.Sounds[report.SoundID]
		if sound.TotalProbability != 0 {
			return fmt.Errorf("deleted_sounds[%d] sound_id=%d total_probability readback got %d want 0", i, report.SoundID, sound.TotalProbability)
		}
		for j, item := range sound.Items {
			if item.Probability != 0 {
				return fmt.Errorf("deleted_sounds[%d] sound_id=%d item[%d] probability readback got %d want 0", i, report.SoundID, j, item.Probability)
			}
		}
	}
	return nil
}

func verifyUnitDeletesReadback(idx *datfile.Index, reports []UnitDeleteReport) error {
	for i, report := range reports {
		if report.CivID < 0 || report.CivID >= len(idx.Civs) {
			return fmt.Errorf("deleted_units[%d] civ_id=%d outside readback civ table", i, report.CivID)
		}
		civ := idx.Civs[report.CivID]
		if report.UnitID < 0 || report.UnitID >= len(civ.Units) {
			return fmt.Errorf("deleted_units[%d] unit_id=%d outside readback civ %d unit table", i, report.UnitID, report.CivID)
		}
		unit := civ.Units[report.UnitID]
		if !unit.Present {
			return fmt.Errorf("deleted_units[%d] civ=%d unit=%d no longer present after semantic delete", i, report.CivID, report.UnitID)
		}
		if unit.Enabled != 0 {
			return fmt.Errorf("deleted_units[%d] civ=%d unit=%d enabled readback got %d want 0", i, report.CivID, report.UnitID, unit.Enabled)
		}
	}
	return nil
}

func verifyUnitAvailabilityReadback(idx *datfile.Index, reports []UnitAvailabilityReport) error {
	for i, report := range reports {
		if report.CivID < 0 || report.CivID >= len(idx.Civs) {
			return fmt.Errorf("unit_availability[%d] civ_id=%d outside readback civ table", i, report.CivID)
		}
		civ := idx.Civs[report.CivID]
		if report.UnitID < 0 || report.UnitID >= len(civ.Units) {
			return fmt.Errorf("unit_availability[%d] unit_id=%d outside readback civ %d unit table", i, report.UnitID, report.CivID)
		}
		unit := civ.Units[report.UnitID]
		if !unit.Present {
			return fmt.Errorf("unit_availability[%d] civ=%d unit=%d absent after patch", i, report.CivID, report.UnitID)
		}
		if unit.Enabled != report.AfterEnabled {
			return fmt.Errorf("unit_availability[%d] civ=%d unit=%d enabled readback got %d want %d", i, report.CivID, report.UnitID, unit.Enabled, report.AfterEnabled)
		}
	}
	return nil
}

func verifyTechTreeReadback(idx *datfile.Index, recipe TechTreePatchRecipe) error {
	for i, create := range recipe.CreateBuildingConnections {
		id := len(idx.TechTree.BuildingConnections) - len(recipe.CreateBuildingConnections) + i
		if id < 0 || id >= len(idx.TechTree.BuildingConnections) {
			return fmt.Errorf("tech_tree.create_building_connections[%d] readback id=%d outside table", i, id)
		}
		got := idx.TechTree.BuildingConnections[id]
		wantID := int32(id)
		if create.ID != nil {
			wantID = *create.ID
		}
		if got.ID != wantID {
			return fmt.Errorf("tech_tree.create_building_connections[%d] id readback got %d want %d", i, got.ID, wantID)
		}
		if err := verifyBuildingConnectionPatchReadback("tech_tree.create_building_connections", i, got, TechTreeBuildingConnectionPatch{
			Index:            id,
			ID:               &wantID,
			Status:           create.Status,
			Buildings:        create.Buildings,
			Units:            create.Units,
			Techs:            create.Techs,
			UnitResearch:     create.UnitResearch,
			Mode:             create.Mode,
			LocationInAge:    create.LocationInAge,
			UnitsTechsTotal:  create.UnitsTechsTotal,
			UnitsTechsFirst:  create.UnitsTechsFirst,
			LineMode:         create.LineMode,
			EnablingResearch: create.EnablingResearch,
		}); err != nil {
			return err
		}
	}
	for i, create := range recipe.CreateUnitConnections {
		id := len(idx.TechTree.UnitConnections) - len(recipe.CreateUnitConnections) + i
		if id < 0 || id >= len(idx.TechTree.UnitConnections) {
			return fmt.Errorf("tech_tree.create_unit_connections[%d] readback id=%d outside table", i, id)
		}
		got := idx.TechTree.UnitConnections[id]
		wantID := int32(id)
		if create.ID != nil {
			wantID = *create.ID
		}
		if got.ID != wantID {
			return fmt.Errorf("tech_tree.create_unit_connections[%d] id readback got %d want %d", i, got.ID, wantID)
		}
		if err := verifyUnitConnectionPatchReadback("tech_tree.create_unit_connections", i, got, TechTreeUnitConnectionPatch{
			Index:            id,
			ID:               &wantID,
			Status:           create.Status,
			UpperBuilding:    create.UpperBuilding,
			UnitResearch:     create.UnitResearch,
			Mode:             create.Mode,
			VerticalLine:     create.VerticalLine,
			Units:            create.Units,
			LocationInAge:    create.LocationInAge,
			RequiredResearch: create.RequiredResearch,
			LineMode:         create.LineMode,
			EnablingResearch: create.EnablingResearch,
		}); err != nil {
			return err
		}
	}
	for i, create := range recipe.CreateResearchConnections {
		id := len(idx.TechTree.ResearchConnections) - len(recipe.CreateResearchConnections) + i
		if id < 0 || id >= len(idx.TechTree.ResearchConnections) {
			return fmt.Errorf("tech_tree.create_research_connections[%d] readback id=%d outside table", i, id)
		}
		got := idx.TechTree.ResearchConnections[id]
		wantID := int32(id)
		if create.ID != nil {
			wantID = *create.ID
		}
		if got.ID != wantID {
			return fmt.Errorf("tech_tree.create_research_connections[%d] id readback got %d want %d", i, got.ID, wantID)
		}
		if err := verifyResearchConnectionPatchReadback("tech_tree.create_research_connections", i, got, TechTreeResearchConnectionPatch{
			Index:         id,
			ID:            &wantID,
			Status:        create.Status,
			UpperBuilding: create.UpperBuilding,
			Buildings:     create.Buildings,
			Units:         create.Units,
			Techs:         create.Techs,
			UnitResearch:  create.UnitResearch,
			Mode:          create.Mode,
			VerticalLine:  create.VerticalLine,
			LocationInAge: create.LocationInAge,
			LineMode:      create.LineMode,
		}); err != nil {
			return err
		}
	}
	for i, rewrite := range recipe.Rewrites {
		if rewrite.To != nil && *rewrite.To == rewrite.From {
			return fmt.Errorf("tech_tree.rewrites[%d] to=%d equals from", i, *rewrite.To)
		}
		remaining, err := countTechTreeRewriteTarget(idx.TechTree, rewrite.Target, rewrite.From)
		if err != nil {
			return fmt.Errorf("tech_tree.rewrites[%d]: %w", i, err)
		}
		if remaining != 0 {
			return fmt.Errorf("tech_tree.rewrites[%d] target=%s from=%d remains in %d tech-tree fields after readback", i, rewrite.Target, rewrite.From, remaining)
		}
	}
	for i, patch := range recipe.BuildingConnections {
		if patch.Index < 0 || patch.Index >= len(idx.TechTree.BuildingConnections) {
			return fmt.Errorf("tech_tree.building_connections[%d] index=%d outside readback table", i, patch.Index)
		}
		got := idx.TechTree.BuildingConnections[patch.Index]
		if err := verifyBuildingConnectionPatchReadback("tech_tree.building_connections", patch.Index, got, patch); err != nil {
			return err
		}
	}
	for i, patch := range recipe.UnitConnections {
		if patch.Index < 0 || patch.Index >= len(idx.TechTree.UnitConnections) {
			return fmt.Errorf("tech_tree.unit_connections[%d] index=%d outside readback table", i, patch.Index)
		}
		got := idx.TechTree.UnitConnections[patch.Index]
		if err := verifyUnitConnectionPatchReadback("tech_tree.unit_connections", patch.Index, got, patch); err != nil {
			return err
		}
	}
	for i, patch := range recipe.ResearchConnections {
		if patch.Index < 0 || patch.Index >= len(idx.TechTree.ResearchConnections) {
			return fmt.Errorf("tech_tree.research_connections[%d] index=%d outside readback table", i, patch.Index)
		}
		got := idx.TechTree.ResearchConnections[patch.Index]
		if err := verifyResearchConnectionPatchReadback("tech_tree.research_connections", patch.Index, got, patch); err != nil {
			return err
		}
	}
	return nil
}

func verifyReferenceRewriteReadback(idx *datfile.Index, recipes []TechTreeIDRewriteRecipe) error {
	for i, recipe := range recipes {
		techTreeRemaining, err := countTechTreeRewriteTarget(idx.TechTree, recipe.Target, recipe.From)
		if err != nil {
			return fmt.Errorf("reference_rewrites[%d]: %w", i, err)
		}
		if techTreeRemaining != 0 {
			return fmt.Errorf("reference_rewrites[%d] target=%s from=%d remains in %d tech-tree fields after readback", i, recipe.Target, recipe.From, techTreeRemaining)
		}
		if recipe.Target == "tech" {
			if remaining := countTechRequiredRefs(idx.Techs, recipe.From); remaining != 0 {
				return fmt.Errorf("reference_rewrites[%d] tech from=%d remains in %d required-tech fields after readback", i, recipe.From, remaining)
			}
		}
		if remaining := countTypedEffectCommandRefs(idx.Effects, recipe.Target, recipe.From); remaining != 0 {
			return fmt.Errorf("reference_rewrites[%d] target=%s from=%d remains in %d typed effect-command refs after readback", i, recipe.Target, recipe.From, remaining)
		}
	}
	return nil
}

func verifyDisconnectUnitReadback(idx *datfile.Index, ids []int) error {
	for i, id := range ids {
		if remaining := countSupportedUnitRefs(idx, id); remaining != 0 {
			return fmt.Errorf("disconnect_units[%d] unit=%d remains in %d supported refs after readback", i, id, remaining)
		}
	}
	return nil
}

func countSupportedUnitRefs(idx *datfile.Index, id int) int {
	count := 0
	for _, ref := range unitIDReferences(idx, id) {
		if unitReferenceRewriteCovers(ref) {
			count++
		}
	}
	return count
}

func countTechRequiredRefs(techs []datfile.Tech, id int32) int {
	count := 0
	for _, tech := range techs {
		for _, value := range tech.RequiredTechs {
			if int32(value) == id {
				count++
			}
		}
	}
	return count
}

func countTypedEffectCommandRefs(effects []datfile.Effect, target string, id int32) int {
	count := 0
	for _, effect := range effects {
		for _, command := range effect.Commands {
			for _, ref := range effectCommandSemanticReferences(command) {
				if ref.Kind == target && int32(ref.ID) == id {
					count++
				}
			}
		}
	}
	return count
}

func countTechTreeRewriteTarget(tree datfile.TechTree, target string, id int32) (int, error) {
	countScalar := func(value int32) int {
		if value == id {
			return 1
		}
		return 0
	}
	countList := func(values []int32) int {
		count := 0
		for _, value := range values {
			if value == id {
				count++
			}
		}
		return count
	}
	countCommon := func(common datfile.TechTreeCommon) int {
		return countList(common.UnitResearch)
	}
	total := 0
	switch target {
	case "tech":
		for _, age := range tree.Ages {
			total += countList(age.Techs)
			total += countCommon(age.Common)
		}
		for _, connection := range tree.BuildingConnections {
			total += countList(connection.Techs)
			total += countCommon(connection.Common)
			total += countScalar(connection.EnablingResearch)
		}
		for _, connection := range tree.UnitConnections {
			total += countCommon(connection.Common)
			total += countScalar(connection.RequiredResearch)
			total += countScalar(connection.EnablingResearch)
		}
		for _, connection := range tree.ResearchConnections {
			total += countScalar(connection.ID)
			total += countList(connection.Techs)
			total += countCommon(connection.Common)
		}
	case "unit":
		for _, age := range tree.Ages {
			total += countList(age.Buildings)
			total += countList(age.Units)
		}
		for _, connection := range tree.BuildingConnections {
			total += countScalar(connection.ID)
			total += countList(connection.Buildings)
			total += countList(connection.Units)
		}
		for _, connection := range tree.UnitConnections {
			total += countScalar(connection.ID)
			total += countScalar(connection.UpperBuilding)
			total += countList(connection.Units)
		}
		for _, connection := range tree.ResearchConnections {
			total += countScalar(connection.UpperBuilding)
			total += countList(connection.Buildings)
			total += countList(connection.Units)
		}
	default:
		return 0, fmt.Errorf("target=%q, want tech or unit", target)
	}
	return total, nil
}

func verifyBuildingConnectionPatchReadback(section string, index int, got datfile.BuildingConnection, patch TechTreeBuildingConnectionPatch) error {
	if err := verifyInt32Ptr(section, index, "id", got.ID, patch.ID); err != nil {
		return err
	}
	if err := verifyU8Ptr(section, index, "status", got.Status, patch.Status); err != nil {
		return err
	}
	if err := verifyInt32SlicePtr(section, index, "buildings", got.Buildings, patch.Buildings); err != nil {
		return err
	}
	if err := verifyInt32SlicePtr(section, index, "units", got.Units, patch.Units); err != nil {
		return err
	}
	if err := verifyInt32SlicePtr(section, index, "techs", got.Techs, patch.Techs); err != nil {
		return err
	}
	if err := verifyInt32SlicePtr(section, index, "unit_research", got.Common.UnitResearch, patch.UnitResearch); err != nil {
		return err
	}
	if err := verifyInt32SlicePtr(section, index, "mode", got.Common.Mode, patch.Mode); err != nil {
		return err
	}
	if err := verifyU8Ptr(section, index, "location_in_age", got.LocationInAge, patch.LocationInAge); err != nil {
		return err
	}
	if err := verifyU8SlicePtr(section, index, "units_techs_total", got.UnitsTechsTotal, patch.UnitsTechsTotal); err != nil {
		return err
	}
	if err := verifyU8SlicePtr(section, index, "units_techs_first", got.UnitsTechsFirst, patch.UnitsTechsFirst); err != nil {
		return err
	}
	if err := verifyInt32Ptr(section, index, "line_mode", got.LineMode, patch.LineMode); err != nil {
		return err
	}
	if err := verifyInt32Ptr(section, index, "enabling_research", got.EnablingResearch, patch.EnablingResearch); err != nil {
		return err
	}
	return nil
}

func verifyUnitConnectionPatchReadback(section string, index int, got datfile.UnitConnection, patch TechTreeUnitConnectionPatch) error {
	if err := verifyInt32Ptr(section, index, "id", got.ID, patch.ID); err != nil {
		return err
	}
	if err := verifyU8Ptr(section, index, "status", got.Status, patch.Status); err != nil {
		return err
	}
	if err := verifyInt32Ptr(section, index, "upper_building", got.UpperBuilding, patch.UpperBuilding); err != nil {
		return err
	}
	if err := verifyInt32SlicePtr(section, index, "unit_research", got.Common.UnitResearch, patch.UnitResearch); err != nil {
		return err
	}
	if err := verifyInt32SlicePtr(section, index, "mode", got.Common.Mode, patch.Mode); err != nil {
		return err
	}
	if err := verifyInt32Ptr(section, index, "vertical_line", got.VerticalLine, patch.VerticalLine); err != nil {
		return err
	}
	if err := verifyInt32SlicePtr(section, index, "units", got.Units, patch.Units); err != nil {
		return err
	}
	if err := verifyInt32Ptr(section, index, "location_in_age", got.LocationInAge, patch.LocationInAge); err != nil {
		return err
	}
	if err := verifyInt32Ptr(section, index, "required_research", got.RequiredResearch, patch.RequiredResearch); err != nil {
		return err
	}
	if err := verifyInt32Ptr(section, index, "line_mode", got.LineMode, patch.LineMode); err != nil {
		return err
	}
	if err := verifyInt32Ptr(section, index, "enabling_research", got.EnablingResearch, patch.EnablingResearch); err != nil {
		return err
	}
	return nil
}

func verifyResearchConnectionPatchReadback(section string, index int, got datfile.ResearchConnection, patch TechTreeResearchConnectionPatch) error {
	if err := verifyInt32Ptr(section, index, "id", got.ID, patch.ID); err != nil {
		return err
	}
	if err := verifyU8Ptr(section, index, "status", got.Status, patch.Status); err != nil {
		return err
	}
	if err := verifyInt32Ptr(section, index, "upper_building", got.UpperBuilding, patch.UpperBuilding); err != nil {
		return err
	}
	if err := verifyInt32SlicePtr(section, index, "buildings", got.Buildings, patch.Buildings); err != nil {
		return err
	}
	if err := verifyInt32SlicePtr(section, index, "units", got.Units, patch.Units); err != nil {
		return err
	}
	if err := verifyInt32SlicePtr(section, index, "techs", got.Techs, patch.Techs); err != nil {
		return err
	}
	if err := verifyInt32SlicePtr(section, index, "unit_research", got.Common.UnitResearch, patch.UnitResearch); err != nil {
		return err
	}
	if err := verifyInt32SlicePtr(section, index, "mode", got.Common.Mode, patch.Mode); err != nil {
		return err
	}
	if err := verifyInt32Ptr(section, index, "vertical_line", got.VerticalLine, patch.VerticalLine); err != nil {
		return err
	}
	if err := verifyInt32Ptr(section, index, "location_in_age", got.LocationInAge, patch.LocationInAge); err != nil {
		return err
	}
	if err := verifyInt32Ptr(section, index, "line_mode", got.LineMode, patch.LineMode); err != nil {
		return err
	}
	return nil
}

func verifyInt32Ptr(section string, index int, field string, got int32, want *int32) error {
	if want != nil && got != *want {
		return fmt.Errorf("%s[%d] %s readback got %d want %d", section, index, field, got, *want)
	}
	return nil
}

func verifyU8Ptr(section string, index int, field string, got uint8, want *uint8) error {
	if want != nil && got != *want {
		return fmt.Errorf("%s[%d] %s readback got %d want %d", section, index, field, got, *want)
	}
	return nil
}

func verifyInt32SlicePtr(section string, index int, field string, got []int32, want *[]int32) error {
	if want == nil {
		return nil
	}
	if len(got) != len(*want) {
		return fmt.Errorf("%s[%d] %s readback length got %d want %d", section, index, field, len(got), len(*want))
	}
	for i := range got {
		if got[i] != (*want)[i] {
			return fmt.Errorf("%s[%d] %s[%d] readback got %d want %d", section, index, field, i, got[i], (*want)[i])
		}
	}
	return nil
}

func verifyU8SlicePtr(section string, index int, field string, got []uint8, want *[]uint8) error {
	if want == nil {
		return nil
	}
	if len(got) != len(*want) {
		return fmt.Errorf("%s[%d] %s readback length got %d want %d", section, index, field, len(got), len(*want))
	}
	for i := range got {
		if got[i] != (*want)[i] {
			return fmt.Errorf("%s[%d] %s[%d] readback got %d want %d", section, index, field, i, got[i], (*want)[i])
		}
	}
	return nil
}

func verifyPlayerColourReadback(idx *datfile.Index, recipes []PlayerColourPatchRecipe) error {
	for i, recipe := range recipes {
		if recipe.ID < 0 || recipe.ID >= len(idx.PlayerColours) {
			return fmt.Errorf("player_colours[%d] id=%d outside readback player colour table", i, recipe.ID)
		}
		colour := idx.PlayerColours[recipe.ID]
		checks := []struct {
			name string
			got  int32
			want *int32
		}{
			{name: "colour_id", got: colour.ID, want: recipe.ColourID},
			{name: "base", got: colour.Base, want: recipe.Base},
			{name: "unit_outline_colour", got: colour.UnitOutlineColour, want: recipe.UnitOutlineColour},
			{name: "selection_colour_1", got: colour.SelectionColour1, want: recipe.SelectionColour1},
			{name: "selection_colour_2", got: colour.SelectionColour2, want: recipe.SelectionColour2},
			{name: "minimap_colour_1", got: colour.MinimapColour1, want: recipe.MinimapColour1},
			{name: "minimap_colour_2", got: colour.MinimapColour2, want: recipe.MinimapColour2},
			{name: "minimap_colour_3", got: colour.MinimapColour3, want: recipe.MinimapColour3},
			{name: "statistics_text_color", got: colour.StatisticsTextColor, want: recipe.StatisticsTextColor},
		}
		for _, check := range checks {
			if check.want != nil && check.got != *check.want {
				return fmt.Errorf("player_colours[%d] %s readback got %d want %d", i, check.name, check.got, *check.want)
			}
		}
	}
	return nil
}

func verifyCreatedPlayerColourReadback(idx *datfile.Index, recipes []PlayerColourCreateRecipe, ids []int) error {
	if len(recipes) != len(ids) {
		return fmt.Errorf("created player colour report length=%d want %d", len(ids), len(recipes))
	}
	for i, id := range ids {
		if id < 0 || id >= len(idx.PlayerColours) {
			return fmt.Errorf("created_player_colours[%d] id=%d outside readback player colour table", i, id)
		}
		colour := idx.PlayerColours[id]
		recipe := recipes[i]
		expectedID := int32(id)
		if recipe.ColourID != nil {
			expectedID = *recipe.ColourID
		}
		checks := []struct {
			name string
			got  int32
			want *int32
		}{
			{name: "colour_id", got: colour.ID, want: &expectedID},
			{name: "base", got: colour.Base, want: recipe.Base},
			{name: "unit_outline_colour", got: colour.UnitOutlineColour, want: recipe.UnitOutlineColour},
			{name: "selection_colour_1", got: colour.SelectionColour1, want: recipe.SelectionColour1},
			{name: "selection_colour_2", got: colour.SelectionColour2, want: recipe.SelectionColour2},
			{name: "minimap_colour_1", got: colour.MinimapColour1, want: recipe.MinimapColour1},
			{name: "minimap_colour_2", got: colour.MinimapColour2, want: recipe.MinimapColour2},
			{name: "minimap_colour_3", got: colour.MinimapColour3, want: recipe.MinimapColour3},
			{name: "statistics_text_color", got: colour.StatisticsTextColor, want: recipe.StatisticsTextColor},
		}
		for _, check := range checks {
			if check.want != nil && check.got != *check.want {
				return fmt.Errorf("created_player_colours[%d] %s readback got %d want %d", i, check.name, check.got, *check.want)
			}
		}
	}
	return nil
}

func verifyDeletedPlayerColourReadback(idx *datfile.Index, reports []PlayerColourDeleteReport) error {
	if len(reports) == 0 {
		return nil
	}
	finalAfterCount := reports[len(reports)-1].AfterCount
	if len(idx.PlayerColours) != finalAfterCount {
		return fmt.Errorf("deleted_player_colours readback count got %d want %d", len(idx.PlayerColours), finalAfterCount)
	}
	for i, report := range reports {
		if report.ID < len(idx.PlayerColours) {
			return fmt.Errorf("deleted_player_colours[%d] id=%d still present in readback table length %d", i, report.ID, len(idx.PlayerColours))
		}
	}
	return nil
}

func putI16Record(record []byte, recordSpan, fieldSpan datfile.Span, value int16) error {
	start, err := recordFieldOffset(record, recordSpan, fieldSpan, 2)
	if err != nil {
		return err
	}
	binary.LittleEndian.PutUint16(record[start:start+2], uint16(value))
	return nil
}

func putU8Record(record []byte, recordSpan, fieldSpan datfile.Span, value uint8) error {
	start, err := recordFieldOffset(record, recordSpan, fieldSpan, 1)
	if err != nil {
		return err
	}
	record[start] = value
	return nil
}

func putF32Record(record []byte, recordSpan, fieldSpan datfile.Span, value float32) error {
	start, err := recordFieldOffset(record, recordSpan, fieldSpan, 4)
	if err != nil {
		return err
	}
	binary.LittleEndian.PutUint32(record[start:start+4], math.Float32bits(value))
	return nil
}

func recordFieldOffset(record []byte, recordSpan, fieldSpan datfile.Span, width int) (int, error) {
	start := fieldSpan.Start - recordSpan.Start
	end := fieldSpan.End - recordSpan.Start
	if start < 0 || end != start+width || end > len(record) {
		return 0, fmt.Errorf("field %s span %d..%d is not width %d inside record %d..%d length %d", fieldSpan.Name, fieldSpan.Start, fieldSpan.End, width, recordSpan.Start, recordSpan.End, len(record))
	}
	return start, nil
}

func sectionBounds(idx *datfile.Index, headerName, recordPrefix string, originalCount int) (int, int, error) {
	header, ok := spanByName(idx.Spans, headerName)
	if !ok {
		return 0, 0, fmt.Errorf("%s span unavailable", headerName)
	}
	end := header.End
	if originalCount > 0 {
		record, ok := spanByName(idx.Spans, fmt.Sprintf("%s%d]", recordPrefix, originalCount-1))
		if !ok {
			return 0, 0, fmt.Errorf("%s last record span unavailable", recordPrefix)
		}
		end = record.End
	}
	return header.Start, end, nil
}

func playerColoursSectionBounds(idx *datfile.Index) (int, int, error) {
	header, ok := spanByName(idx.Spans, "player_colours_header")
	if !ok {
		return 0, 0, fmt.Errorf("player_colours_header span unavailable")
	}
	end := header.End
	if len(idx.PlayerColours) > 0 {
		last := idx.PlayerColours[len(idx.PlayerColours)-1].Span
		if last.Start < header.End || last.End < last.Start {
			return 0, 0, fmt.Errorf("player_colour[%d] invalid span %d..%d", len(idx.PlayerColours)-1, last.Start, last.End)
		}
		end = last.End
	}
	return header.Start, end, nil
}

func spanByName(spans []datfile.Span, name string) (datfile.Span, bool) {
	for _, span := range spans {
		if span.Name == name {
			return span, true
		}
	}
	return datfile.Span{}, false
}

func encodeEffectsSection(effects []datfile.Effect) ([]byte, error) {
	if len(effects) > math.MaxInt32 {
		return nil, fmt.Errorf("effect count %d exceeds int32", len(effects))
	}
	var out bytes.Buffer
	writeI32(&out, int32(len(effects)))
	for i, effect := range effects {
		effect.Index = i
		if err := writeDebugString(&out, effect.Name); err != nil {
			return nil, fmt.Errorf("effect[%d] name: %w", i, err)
		}
		if len(effect.Commands) > math.MaxInt16 {
			return nil, fmt.Errorf("effect[%d] command count %d exceeds int16", i, len(effect.Commands))
		}
		writeI16(&out, int16(len(effect.Commands)))
		for j, command := range effect.Commands {
			out.WriteByte(command.Type)
			writeI16(&out, command.A)
			writeI16(&out, command.B)
			writeI16(&out, command.C)
			writeF32(&out, command.D)
			_ = j
		}
	}
	return out.Bytes(), nil
}

func encodeTechsSection(techs []datfile.Tech) ([]byte, error) {
	if len(techs) > math.MaxInt16 {
		return nil, fmt.Errorf("tech count %d exceeds int16", len(techs))
	}
	var out bytes.Buffer
	writeI16(&out, int16(len(techs)))
	for i, tech := range techs {
		tech.Index = i
		if err := encodeTech(&out, tech); err != nil {
			return nil, fmt.Errorf("tech[%d]: %w", i, err)
		}
	}
	return out.Bytes(), nil
}

func encodeTechTreeSection(tree datfile.TechTree) ([]byte, error) {
	if len(tree.Ages) > math.MaxUint8 || len(tree.BuildingConnections) > math.MaxUint8 || len(tree.ResearchConnections) > math.MaxUint8 {
		return nil, errors.New("tech-tree age/building/research connection count exceeds u8")
	}
	if len(tree.UnitConnections) > math.MaxUint16 {
		return nil, errors.New("tech-tree unit connection count exceeds u16")
	}
	var out bytes.Buffer
	out.WriteByte(byte(len(tree.Ages)))
	out.WriteByte(byte(len(tree.BuildingConnections)))
	writeU16(&out, uint16(len(tree.UnitConnections)))
	out.WriteByte(byte(len(tree.ResearchConnections)))
	writeI32(&out, tree.TotalUnitTechGroups)
	for _, age := range tree.Ages {
		if err := encodeTechTreeAge(&out, age); err != nil {
			return nil, err
		}
	}
	for _, connection := range tree.BuildingConnections {
		if err := encodeBuildingConnection(&out, connection); err != nil {
			return nil, err
		}
	}
	for _, connection := range tree.UnitConnections {
		if err := encodeUnitConnection(&out, connection); err != nil {
			return nil, err
		}
	}
	for _, connection := range tree.ResearchConnections {
		if err := encodeResearchConnection(&out, connection); err != nil {
			return nil, err
		}
	}
	return out.Bytes(), nil
}

func encodeTechTreeAge(out *bytes.Buffer, age datfile.TechTreeAge) error {
	writeI32(out, age.ID)
	out.WriteByte(age.Status)
	if err := writeCountedI32ListU8(out, age.Buildings); err != nil {
		return err
	}
	if err := writeCountedI32ListU8(out, age.Units); err != nil {
		return err
	}
	if err := writeCountedI32ListU8(out, age.Techs); err != nil {
		return err
	}
	if err := encodeTechTreeCommon(out, age.Common); err != nil {
		return err
	}
	out.WriteByte(age.NumBuildingLevels)
	if err := writeFixedU8List(out, age.BuildingsPerZone, 10); err != nil {
		return err
	}
	if err := writeFixedU8List(out, age.GroupLengthPerZone, 10); err != nil {
		return err
	}
	out.WriteByte(age.MaxAgeLength)
	writeI32(out, age.LineMode)
	return nil
}

func encodeBuildingConnection(out *bytes.Buffer, connection datfile.BuildingConnection) error {
	encodeConnectionHeader(out, connection.ID, connection.Status)
	if err := writeCountedI32ListU8(out, connection.Buildings); err != nil {
		return err
	}
	if err := writeCountedI32ListU8(out, connection.Units); err != nil {
		return err
	}
	if err := writeCountedI32ListU8(out, connection.Techs); err != nil {
		return err
	}
	if err := encodeTechTreeCommon(out, connection.Common); err != nil {
		return err
	}
	out.WriteByte(connection.LocationInAge)
	if err := writeFixedU8List(out, connection.UnitsTechsTotal, 5); err != nil {
		return err
	}
	if err := writeFixedU8List(out, connection.UnitsTechsFirst, 5); err != nil {
		return err
	}
	writeI32(out, connection.LineMode)
	writeI32(out, connection.EnablingResearch)
	return nil
}

func encodeUnitConnection(out *bytes.Buffer, connection datfile.UnitConnection) error {
	encodeConnectionHeader(out, connection.ID, connection.Status)
	writeI32(out, connection.UpperBuilding)
	if err := encodeTechTreeCommon(out, connection.Common); err != nil {
		return err
	}
	writeI32(out, connection.VerticalLine)
	if err := writeCountedI32ListU8(out, connection.Units); err != nil {
		return err
	}
	writeI32(out, connection.LocationInAge)
	writeI32(out, connection.RequiredResearch)
	writeI32(out, connection.LineMode)
	writeI32(out, connection.EnablingResearch)
	return nil
}

func encodeResearchConnection(out *bytes.Buffer, connection datfile.ResearchConnection) error {
	encodeConnectionHeader(out, connection.ID, connection.Status)
	writeI32(out, connection.UpperBuilding)
	if err := writeCountedI32ListU8(out, connection.Buildings); err != nil {
		return err
	}
	if err := writeCountedI32ListU8(out, connection.Units); err != nil {
		return err
	}
	if err := writeCountedI32ListU8(out, connection.Techs); err != nil {
		return err
	}
	if err := encodeTechTreeCommon(out, connection.Common); err != nil {
		return err
	}
	writeI32(out, connection.VerticalLine)
	writeI32(out, connection.LocationInAge)
	writeI32(out, connection.LineMode)
	return nil
}

func encodeConnectionHeader(out *bytes.Buffer, id int32, status uint8) {
	writeI32(out, id)
	out.WriteByte(status)
}

func encodeTechTreeCommon(out *bytes.Buffer, common datfile.TechTreeCommon) error {
	writeI32(out, common.SlotsUsed)
	if err := writeFixedI32List(out, common.UnitResearch, 10); err != nil {
		return err
	}
	return writeFixedI32List(out, common.Mode, 10)
}

func writeCountedI32ListU8(out *bytes.Buffer, values []int32) error {
	if len(values) > math.MaxUint8 {
		return fmt.Errorf("i32 list length=%d exceeds u8", len(values))
	}
	out.WriteByte(byte(len(values)))
	for _, value := range values {
		writeI32(out, value)
	}
	return nil
}

func writeFixedI32List(out *bytes.Buffer, values []int32, want int) error {
	if len(values) != want {
		return fmt.Errorf("fixed i32 list length=%d, want %d", len(values), want)
	}
	for _, value := range values {
		writeI32(out, value)
	}
	return nil
}

func writeFixedU8List(out *bytes.Buffer, values []uint8, want int) error {
	if len(values) != want {
		return fmt.Errorf("fixed u8 list length=%d, want %d", len(values), want)
	}
	out.Write(values)
	return nil
}

func encodeTech(out *bytes.Buffer, tech datfile.Tech) error {
	if len(tech.RequiredTechs) != 6 {
		return fmt.Errorf("required_techs length=%d, want 6", len(tech.RequiredTechs))
	}
	if len(tech.ResourceCosts) != 3 {
		return fmt.Errorf("resource_costs length=%d, want 3", len(tech.ResourceCosts))
	}
	for _, value := range tech.RequiredTechs {
		writeI16(out, value)
	}
	for _, cost := range tech.ResourceCosts {
		writeI16(out, cost.Type)
		writeI16(out, cost.Amount)
		out.WriteByte(cost.Flag)
	}
	writeI16(out, tech.RequiredTechCount)
	writeI16(out, tech.Civ)
	writeI16(out, tech.FullTechMode)
	writeI32(out, tech.LanguageDLLName)
	writeI32(out, tech.LanguageDLLDescription)
	writeI16(out, tech.EffectID)
	writeI16(out, tech.Type)
	writeI16(out, tech.IconID)
	writeI32(out, tech.LanguageDLLHelp)
	writeI32(out, tech.LanguageDLLTechTree)
	if err := writeDebugString(out, tech.Name); err != nil {
		return err
	}
	out.WriteByte(tech.Repeatable)
	if len(tech.ResearchLocations) > math.MaxInt16 {
		return fmt.Errorf("research location count %d exceeds int16", len(tech.ResearchLocations))
	}
	writeI16(out, int16(len(tech.ResearchLocations)))
	for _, location := range tech.ResearchLocations {
		writeI16(out, location.LocationID)
		writeI16(out, location.ResearchTime)
		out.WriteByte(location.ButtonID)
		writeI32(out, location.HotKeyID)
	}
	return nil
}

func writeDebugString(out *bytes.Buffer, value string) error {
	if len(value) > math.MaxUint16 {
		return fmt.Errorf("debug string too long: %d bytes > %d", len(value), math.MaxUint16)
	}
	writeU16(out, 0x0A60)
	writeU16(out, uint16(len(value)))
	out.WriteString(value)
	return nil
}

func writeU16(out *bytes.Buffer, value uint16) {
	var buf [2]byte
	binary.LittleEndian.PutUint16(buf[:], value)
	out.Write(buf[:])
}

func writeI16(out *bytes.Buffer, value int16) {
	writeU16(out, uint16(value))
}

func writeI32(out *bytes.Buffer, value int32) {
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], uint32(value))
	out.Write(buf[:])
}

func writeF32(out *bytes.Buffer, value float32) {
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], math.Float32bits(value))
	out.Write(buf[:])
}

func filepathAbs(path string) (string, error) {
	return filepath.Abs(path)
}
