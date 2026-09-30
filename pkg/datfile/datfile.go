package datfile

import (
	"bytes"
	"compress/flate"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"unicode"

	"aoe2kit/pkg/aoe2"
)

const DebugStringMarker uint16 = 0x0A60

const (
	tileTypeCount    = 19
	terrainCount     = 200
	terrainUnitsSize = 30
	taskFormat84     = "hhbhhhhhhhhfffbfbbhhbbbhhhhhhll"
	taskFormat88     = "hhbhhhhhhhhfffbfbbhhbbbhhhhhhllh"
	unitFormat88     = "bhllhhhhhbhfbfffhhhhbbhbhbbhhhhffbbhbhfbbbbbfbllbbbbbbbbbhbbfffll"
	creatable88      = "ffbblhhhhffhhhlbfhllhffffbfffllbh"
)

type Index struct {
	Path                string               `json:"path,omitempty"`
	Compressed          int                  `json:"compressed_bytes"`
	Inflated            int                  `json:"inflated_bytes"`
	Version             string               `json:"version"`
	GraphicsSize        int                  `json:"graphics_size"`
	Graphics            []Graphic            `json:"graphics,omitempty"`
	Effects             []Effect             `json:"effects,omitempty"`
	UnitHeaders         []UnitHeader         `json:"unit_headers,omitempty"`
	Civs                []Civ                `json:"civs,omitempty"`
	Techs               []Tech               `json:"techs,omitempty"`
	TerrainRestrictions []TerrainRestriction `json:"terrain_restrictions,omitempty"`
	Terrains            []Terrain            `json:"terrains,omitempty"`
	PlayerColours       []PlayerColour       `json:"player_colours,omitempty"`
	Sounds              []Sound              `json:"sounds,omitempty"`
	RandomMaps          []RandomMapInfo      `json:"random_maps,omitempty"`
	GameMetrics         Metrics              `json:"game_metrics,omitempty"`
	TechTree            TechTree             `json:"tech_tree,omitempty"`
	Spans               []Span               `json:"spans,omitempty"`
}

type Span struct {
	Name  string `json:"name"`
	Start int    `json:"start"`
	End   int    `json:"end"`
}

func (s Span) Len() int {
	return s.End - s.Start
}

type DebugString struct {
	Value      string `json:"value"`
	Marker     uint16 `json:"marker"`
	ByteLength int    `json:"byte_length"`
	Span       Span   `json:"span"`
	DataSpan   Span   `json:"data_span"`
}

type TerrainRestriction struct {
	Index       int                         `json:"index"`
	TerrainRows []TerrainRestrictionTerrain `json:"terrain_rows,omitempty"`
	Span        Span                        `json:"span"`
}

type TerrainRestrictionTerrain struct {
	Index              int     `json:"index"`
	Passability        float32 `json:"passability"`
	PassGraphicRawSize int     `json:"pass_graphic_raw_size"`
	Span               Span    `json:"span"`
	PassabilitySpan    Span    `json:"passability_span"`
	PassGraphicSpan    Span    `json:"pass_graphic_span"`
}

type PlayerColour struct {
	Index               int   `json:"index"`
	ID                  int32 `json:"id"`
	Base                int32 `json:"base"`
	UnitOutlineColour   int32 `json:"unit_outline_colour"`
	SelectionColour1    int32 `json:"selection_colour_1"`
	SelectionColour2    int32 `json:"selection_colour_2"`
	MinimapColour1      int32 `json:"minimap_colour_1"`
	MinimapColour2      int32 `json:"minimap_colour_2"`
	MinimapColour3      int32 `json:"minimap_colour_3"`
	StatisticsTextColor int32 `json:"statistics_text_color"`
	Span                Span  `json:"span"`
}

type UnitHeader struct {
	Index     int           `json:"index"`
	Exists    bool          `json:"exists"`
	TaskCount int           `json:"task_count"`
	Tasks     []TaskSummary `json:"tasks,omitempty"`
	Span      Span          `json:"span"`
}

type Sound struct {
	Index            int         `json:"index"`
	ID               int16       `json:"id"`
	PlayDelay        int16       `json:"play_delay"`
	CacheTime        int32       `json:"cache_time"`
	TotalProbability int16       `json:"total_probability"`
	Items            []SoundItem `json:"items,omitempty"`
	Span             Span        `json:"span"`
}

type SoundItem struct {
	Index       int         `json:"index"`
	FileName    DebugString `json:"file_name"`
	ResourceID  int32       `json:"resource_id"`
	Probability int16       `json:"probability"`
	Civ         int16       `json:"civ"`
	IconSet     int16       `json:"icon_set"`
	Span        Span        `json:"span"`
}

type Terrain struct {
	Index           int         `json:"index"`
	Name            DebugString `json:"name"`
	Name2           DebugString `json:"name_2"`
	OverlayMaskName DebugString `json:"overlay_mask_name"`
	Span            Span        `json:"span"`
}

type RandomMapInfo struct {
	Pass          int    `json:"pass"`
	Index         int    `json:"index"`
	Lands         uint32 `json:"lands"`
	Terrains      uint32 `json:"terrains"`
	Units         uint32 `json:"units"`
	Elevations    uint32 `json:"elevations"`
	Span          Span   `json:"span"`
	LandSpan      Span   `json:"land_span"`
	TerrainSpan   Span   `json:"terrain_span"`
	UnitSpan      Span   `json:"unit_span"`
	ElevationSpan Span   `json:"elevation_span"`
}

type Graphic struct {
	Index              int                 `json:"index"`
	Present            bool                `json:"present"`
	Name               DebugString         `json:"name"`
	FileName           DebugString         `json:"file_name"`
	ParticleEffectName DebugString         `json:"particle_effect_name"`
	SLP                int32               `json:"slp"`
	IsLoaded           int8                `json:"is_loaded"`
	OldColorFlag       int8                `json:"old_color_flag"`
	Layer              int8                `json:"layer"`
	PlayerColor        int8                `json:"player_color"`
	Rainbow            int8                `json:"rainbow"`
	TransparentSelect  int8                `json:"transparent_selection"`
	Coordinates        [4]int16            `json:"coordinates"`
	SoundID            int16               `json:"sound_id"`
	WwiseSoundID       uint32              `json:"wwise_sound_id"`
	AngleSoundsUsed    int8                `json:"angle_sounds_used"`
	FrameCount         int16               `json:"frame_count"`
	AngleCount         int16               `json:"angle_count"`
	DeltaCount         int16               `json:"delta_count"`
	SpeedMultiplier    float32             `json:"speed_multiplier"`
	FrameDuration      float32             `json:"frame_duration"`
	ReplayDelay        float32             `json:"replay_delay"`
	SequenceType       uint8               `json:"sequence_type"`
	ID                 int16               `json:"id"`
	MirroringMode      int8                `json:"mirroring_mode"`
	EditorFlag         int8                `json:"editor_flag"`
	Deltas             []GraphicDelta      `json:"deltas,omitempty"`
	AngleSounds        []GraphicAngleSound `json:"angle_sounds,omitempty"`
	FieldSpans         GraphicFieldSpans   `json:"field_spans,omitempty"`
	RecordSpan         Span                `json:"record_span"`
}

type GraphicDelta struct {
	Index        int   `json:"index"`
	GraphicID    int16 `json:"graphic_id"`
	Padding1     int16 `json:"padding_1"`
	SpritePtr    int32 `json:"sprite_ptr"`
	OffsetX      int16 `json:"offset_x"`
	OffsetY      int16 `json:"offset_y"`
	DisplayAngle int16 `json:"display_angle"`
	Padding2     int16 `json:"padding_2"`
	Span         Span  `json:"span"`
}

type GraphicDeltaRow struct {
	GraphicID    int16 `json:"graphic_id"`
	Padding1     int16 `json:"padding_1"`
	SpritePtr    int32 `json:"sprite_ptr"`
	OffsetX      int16 `json:"offset_x"`
	OffsetY      int16 `json:"offset_y"`
	DisplayAngle int16 `json:"display_angle"`
	Padding2     int16 `json:"padding_2"`
}

type GraphicAngleSound struct {
	Index         int    `json:"index"`
	FrameNum      int16  `json:"frame_num"`
	SoundID       int16  `json:"sound_id"`
	WwiseSoundID  uint32 `json:"wwise_sound_id"`
	FrameNum2     int16  `json:"frame_num_2"`
	WwiseSoundID2 uint32 `json:"wwise_sound_id_2"`
	SoundID2      int16  `json:"sound_id_2"`
	FrameNum3     int16  `json:"frame_num_3"`
	WwiseSoundID3 uint32 `json:"wwise_sound_id_3"`
	SoundID3      int16  `json:"sound_id_3"`
	Span          Span   `json:"span"`
}

type GraphicAngleSoundRow struct {
	FrameNum      int16  `json:"frame_num"`
	SoundID       int16  `json:"sound_id"`
	WwiseSoundID  uint32 `json:"wwise_sound_id"`
	FrameNum2     int16  `json:"frame_num_2"`
	WwiseSoundID2 uint32 `json:"wwise_sound_id_2"`
	SoundID2      int16  `json:"sound_id_2"`
	FrameNum3     int16  `json:"frame_num_3"`
	WwiseSoundID3 uint32 `json:"wwise_sound_id_3"`
	SoundID3      int16  `json:"sound_id_3"`
}

type GraphicFieldSpans struct {
	SLP               Span    `json:"slp"`
	IsLoaded          Span    `json:"is_loaded"`
	OldColorFlag      Span    `json:"old_color_flag"`
	Layer             Span    `json:"layer"`
	PlayerColor       Span    `json:"player_color"`
	Rainbow           Span    `json:"rainbow"`
	TransparentSelect Span    `json:"transparent_selection"`
	Coordinates       [4]Span `json:"coordinates"`
	DeltaCount        Span    `json:"delta_count"`
	SoundID           Span    `json:"sound_id"`
	WwiseSoundID      Span    `json:"wwise_sound_id"`
	AngleSoundsUsed   Span    `json:"angle_sounds_used"`
	FrameCount        Span    `json:"frame_count"`
	AngleCount        Span    `json:"angle_count"`
	SpeedMultiplier   Span    `json:"speed_multiplier"`
	FrameDuration     Span    `json:"frame_duration"`
	ReplayDelay       Span    `json:"replay_delay"`
	SequenceType      Span    `json:"sequence_type"`
	ID                Span    `json:"id"`
	MirroringMode     Span    `json:"mirroring_mode"`
	EditorFlag        Span    `json:"editor_flag"`
}

type GraphicPatch struct {
	Name               *string                 `json:"name,omitempty"`
	FileName           *string                 `json:"file_name,omitempty"`
	ParticleEffectName *string                 `json:"particle_effect_name,omitempty"`
	SLP                *int32                  `json:"slp,omitempty"`
	IsLoaded           *int8                   `json:"is_loaded,omitempty"`
	OldColorFlag       *int8                   `json:"old_color_flag,omitempty"`
	Layer              *int8                   `json:"layer,omitempty"`
	PlayerColor        *int8                   `json:"player_color,omitempty"`
	Rainbow            *int8                   `json:"rainbow,omitempty"`
	TransparentSelect  *int8                   `json:"transparent_selection,omitempty"`
	Coordinates        []int16                 `json:"coordinates,omitempty"`
	SoundID            *int16                  `json:"sound_id,omitempty"`
	WwiseSoundID       *uint32                 `json:"wwise_sound_id,omitempty"`
	FrameCount         *int16                  `json:"frame_count,omitempty"`
	SpeedMultiplier    *float32                `json:"speed_multiplier,omitempty"`
	FrameDuration      *float32                `json:"frame_duration,omitempty"`
	ReplayDelay        *float32                `json:"replay_delay,omitempty"`
	SequenceType       *uint8                  `json:"sequence_type,omitempty"`
	MirroringMode      *int8                   `json:"mirroring_mode,omitempty"`
	EditorFlag         *int8                   `json:"editor_flag,omitempty"`
	SetDeltas          *[]GraphicDeltaRow      `json:"set_deltas,omitempty"`
	SetAngleSounds     *[]GraphicAngleSoundRow `json:"set_angle_sounds,omitempty"`
}

type PatchReport struct {
	InputCompressedBytes  int                    `json:"input_compressed_bytes"`
	OutputCompressedBytes int                    `json:"output_compressed_bytes"`
	InputInflatedBytes    int                    `json:"input_inflated_bytes"`
	OutputInflatedBytes   int                    `json:"output_inflated_bytes"`
	InflatedLengthDelta   int                    `json:"inflated_length_delta"`
	GraphicID             int                    `json:"graphic_id"`
	Before                GraphicSummary         `json:"before"`
	After                 GraphicSummary         `json:"after"`
	ChangedFields         []string               `json:"changed_fields"`
	Verified              bool                   `json:"verified"`
	Verification          aoe2.VerificationClaim `json:"verification"`
}

type GraphicSummary struct {
	Index              int     `json:"index"`
	Name               string  `json:"name"`
	FileName           string  `json:"file_name"`
	ParticleEffectName string  `json:"particle_effect_name"`
	SLP                int32   `json:"slp"`
	Layer              int8    `json:"layer"`
	PlayerColor        int8    `json:"player_color"`
	SoundID            int16   `json:"sound_id"`
	WwiseSoundID       uint32  `json:"wwise_sound_id"`
	FrameCount         int16   `json:"frame_count"`
	AngleCount         int16   `json:"angle_count"`
	DeltaCount         int16   `json:"delta_count"`
	AngleSoundCount    int     `json:"angle_sound_count"`
	SpeedMultiplier    float32 `json:"speed_multiplier"`
	FrameDuration      float32 `json:"frame_duration"`
	ReplayDelay        float32 `json:"replay_delay"`
	SequenceType       uint8   `json:"sequence_type"`
	ID                 int16   `json:"id"`
	RecordStart        int     `json:"record_start"`
	RecordEnd          int     `json:"record_end"`
}

type GraphicFilter struct {
	NameContains string
	ParticleOnly bool
}

type Civ struct {
	Index         int           `json:"index"`
	Type          uint8         `json:"type"`
	Name          string        `json:"name"`
	ResourcesSize int           `json:"resources_size"`
	TechTreeID    int16         `json:"tech_tree_id"`
	TeamBonusID   int16         `json:"team_bonus_id"`
	Resources     []float32     `json:"resources,omitempty"`
	IconSet       uint8         `json:"icon_set"`
	UnitsSize     int           `json:"units_size"`
	Units         []UnitSummary `json:"units,omitempty"`
	FieldSpans    CivFieldSpans `json:"field_spans,omitempty"`
	Span          Span          `json:"span"`
}

type CivFieldSpans struct {
	Type          Span `json:"type"`
	Name          Span `json:"name"`
	ResourcesSize Span `json:"resources_size"`
	TechTreeID    Span `json:"tech_tree_id"`
	TeamBonusID   Span `json:"team_bonus_id"`
	Resources     Span `json:"resources"`
	IconSet       Span `json:"icon_set"`
	UnitsSize     Span `json:"units_size"`
	UnitPointers  Span `json:"unit_pointers"`
}

type UnitSummary struct {
	CivIndex           int               `json:"civ_index"`
	Index              int               `json:"index"`
	Present            bool              `json:"present"`
	Type               int               `json:"type"`
	ID                 int16             `json:"id"`
	Name               string            `json:"name"`
	Class              int16             `json:"class"`
	ClassName          string            `json:"class_name,omitempty"`
	HitPoints          int16             `json:"hit_points"`
	LineOfSight        float32           `json:"line_of_sight"`
	CollisionSizeX     float32           `json:"collision_size_x"`
	CollisionSizeY     float32           `json:"collision_size_y"`
	CollisionSizeZ     float32           `json:"collision_size_z"`
	ClearanceSizeX     float32           `json:"clearance_size_x"`
	ClearanceSizeY     float32           `json:"clearance_size_y"`
	TerrainTableID     int16             `json:"terrain_table_id"`
	MovementType       uint8             `json:"movement_type"`
	FlyMode            uint8             `json:"fly_mode"`
	ObstructionTypeID  uint8             `json:"obstruction_type_id"`
	ObstructionClassID uint8             `json:"obstruction_class_id"`
	StandingGraphic1   int16             `json:"standing_graphic_1"`
	StandingGraphic2   int16             `json:"standing_graphic_2"`
	DyingGraphic       int16             `json:"dying_graphic"`
	UndeadGraphic      int16             `json:"undead_graphic,omitempty"`
	UndeadFlag         uint8             `json:"undead_flag,omitempty"`
	BloodUnitID        int16             `json:"blood_unit_id"`
	TrainSound         int16             `json:"train_sound,omitempty"`
	DamageSound        int16             `json:"damage_sound,omitempty"`
	DeathSpawn         int16             `json:"death_spawn,omitempty"`
	GarrisonCapacity   uint8             `json:"garrison_capacity,omitempty"`
	IconID             int16             `json:"icon_id"`
	Enabled            uint8             `json:"enabled"`
	MoveSprite         int16             `json:"move_sprite,omitempty"`
	RunSprite          int16             `json:"run_sprite,omitempty"`
	TurnSpeed          float32           `json:"turn_speed,omitempty"`
	SizeClass          uint8             `json:"size_class,omitempty"`
	TrailingUnit       int16             `json:"trailing_unit,omitempty"`
	TrailingOptions    uint8             `json:"trailing_options,omitempty"`
	TrailingSpacing    float32           `json:"trailing_spacing,omitempty"`
	MoveAlgorithm      uint8             `json:"move_algorithm,omitempty"`
	TurnRadius         float32           `json:"turn_radius,omitempty"`
	TurnRadiusSpeed    float32           `json:"turn_radius_speed,omitempty"`
	MaxYawMoving       float32           `json:"maximum_yaw_per_second_moving,omitempty"`
	StationaryYawTime  float32           `json:"stationary_yaw_per_revolution_time,omitempty"`
	MaxYawStationary   float32           `json:"maximum_yaw_per_second_stationary,omitempty"`
	Attributes         []UnitAttribute   `json:"attributes,omitempty"`
	DamageGraphics     []DamageGraphic   `json:"damage_graphics,omitempty"`
	Action             *ActionSummary    `json:"action,omitempty"`
	Type50             *Type50Summary    `json:"type50,omitempty"`
	Creatable          *CreatableSummary `json:"creatable,omitempty"`
	Building           *BuildingSummary  `json:"building,omitempty"`
	FullFields         map[string]any    `json:"full_fields,omitempty"`
	FieldSpans         UnitFieldSpans    `json:"field_spans,omitempty"`
	ListSpans          UnitListSpans     `json:"list_spans,omitempty"`
	RecordStart        int               `json:"record_start"`
	RecordEnd          int               `json:"record_end"`
	RecordLength       int               `json:"record_length"`
}

type UnitFieldSpans struct {
	ID                 Span `json:"id"`
	Class              Span `json:"class"`
	HitPoints          Span `json:"hit_points"`
	LineOfSight        Span `json:"line_of_sight"`
	CollisionSizeX     Span `json:"collision_size_x"`
	CollisionSizeY     Span `json:"collision_size_y"`
	CollisionSizeZ     Span `json:"collision_size_z"`
	ClearanceSizeX     Span `json:"clearance_size_x"`
	ClearanceSizeY     Span `json:"clearance_size_y"`
	TerrainTableID     Span `json:"terrain_table_id"`
	MovementType       Span `json:"movement_type"`
	ObstructionTypeID  Span `json:"obstruction_type_id"`
	ObstructionClassID Span `json:"obstruction_class_id"`
	StandingGraphic1   Span `json:"standing_graphic_1"`
	StandingGraphic2   Span `json:"standing_graphic_2"`
	DyingGraphic       Span `json:"dying_graphic"`
	BloodUnitID        Span `json:"blood_unit_id"`
	IconID             Span `json:"icon_id"`
	Enabled            Span `json:"enabled"`
}

type UnitListSpans struct {
	DamageGraphicCount Span `json:"damage_graphic_count"`
	DamageGraphics     Span `json:"damage_graphics"`
}

type UnitAttribute struct {
	Index         int     `json:"index"`
	AttributeType uint16  `json:"attribute_type"`
	Amount        float32 `json:"amount"`
	Flag          uint8   `json:"flag"`
	Active        bool    `json:"active"`
	Span          Span    `json:"span"`
}

type DamageGraphic struct {
	Index         int    `json:"index"`
	GraphicID     uint16 `json:"graphic_id"`
	DamagePercent uint16 `json:"damage_percent"`
	Flag          uint8  `json:"flag"`
	Span          Span   `json:"span"`
}

type ActionSummary struct {
	DropSites  []int16          `json:"drop_sites,omitempty"`
	Tasks      []TaskSummary    `json:"tasks,omitempty"`
	FieldSpans ActionFieldSpans `json:"field_spans,omitempty"`
}

type ActionFieldSpans struct {
	DropSiteCount Span `json:"drop_site_count"`
	DropSites     Span `json:"drop_sites"`
	TaskCount     Span `json:"task_count"`
	Tasks         Span `json:"tasks"`
}

type TaskSummary struct {
	Index             int     `json:"index"`
	RecordType        int16   `json:"record_type"`
	ID                int16   `json:"id"`
	IsDefault         uint8   `json:"is_default"`
	ActionType        int16   `json:"action_type"`
	ObjectClass       int16   `json:"object_class"`
	ObjectID          int16   `json:"object_id"`
	TerrainID         int16   `json:"terrain_id"`
	AttributeTypes    []int16 `json:"attribute_types,omitempty"`
	WorkValue1        float32 `json:"work_value_1"`
	WorkValue2        float32 `json:"work_value_2"`
	WorkRange         float32 `json:"work_range"`
	AutoSearchTargets uint8   `json:"auto_search_targets"`
	SearchWaitTime    float32 `json:"search_wait_time"`
	EnableTargeting   uint8   `json:"enable_targeting"`
	CombatLevel       uint8   `json:"combat_level"`
	RawTail           []byte  `json:"raw_tail,omitempty"`
	Span              Span    `json:"span"`
}

type TaskRow struct {
	RecordType        int16   `json:"record_type"`
	ID                int16   `json:"id"`
	IsDefault         uint8   `json:"is_default"`
	ActionType        int16   `json:"action_type"`
	ObjectClass       int16   `json:"object_class"`
	ObjectID          int16   `json:"object_id"`
	TerrainID         int16   `json:"terrain_id"`
	AttributeTypes    []int16 `json:"attribute_types"`
	WorkValue1        float32 `json:"work_value_1"`
	WorkValue2        float32 `json:"work_value_2"`
	WorkRange         float32 `json:"work_range"`
	AutoSearchTargets uint8   `json:"auto_search_targets"`
	SearchWaitTime    float32 `json:"search_wait_time"`
	EnableTargeting   uint8   `json:"enable_targeting"`
	CombatLevel       uint8   `json:"combat_level"`
	RawTail           []byte  `json:"raw_tail"`
}

type WeaponInfo struct {
	Index int   `json:"index"`
	Class int16 `json:"class"`
	Value int16 `json:"value"`
	Span  Span  `json:"span"`
}

type Type50Summary struct {
	BaseArmor           int16            `json:"base_armor"`
	DefenseTerrainBonus int16            `json:"defense_terrain_bonus"`
	ProjectileUnitID    int16            `json:"projectile_unit_id"`
	MaxRange            float32          `json:"max_range"`
	BlastWidth          float32          `json:"blast_width"`
	AttackSpeed         float32          `json:"attack_speed"`
	BaseHitChance       int16            `json:"base_hit_chance"`
	BreakOffCombat      uint8            `json:"break_off_combat"`
	FrameDelay          int16            `json:"frame_delay"`
	WeaponOffsetX       float32          `json:"weapon_offset_x"`
	WeaponOffsetY       float32          `json:"weapon_offset_y"`
	WeaponOffsetZ       float32          `json:"weapon_offset_z"`
	BlastAttackLevel    uint8            `json:"blast_attack_level"`
	MinRange            float32          `json:"min_range"`
	MissedMissileSpread float32          `json:"missed_missile_spread"`
	AttackGraphic       int16            `json:"attack_graphic"`
	DisplayedArmor      int16            `json:"displayed_armor"`
	DisplayedAttack     int16            `json:"displayed_attack"`
	DisplayedRange      float32          `json:"displayed_range"`
	DisplayedReloadTime float32          `json:"displayed_reload_time"`
	BlastDamage         float32          `json:"blast_damage"`
	MissileType         uint8            `json:"missile_type"`
	TargetingType       uint8            `json:"targeting_type"`
	MissileHitInfo      uint8            `json:"missile_hit_info"`
	MissileDieInfo      uint8            `json:"missile_die_info"`
	AreaEffectSpecials  uint8            `json:"area_effect_specials"`
	BallisticsRatio     float32          `json:"ballistics_ratio"`
	Attacks             []WeaponInfo     `json:"attacks,omitempty"`
	Armours             []WeaponInfo     `json:"armours,omitempty"`
	FieldSpans          Type50FieldSpans `json:"field_spans,omitempty"`
}

type Type50FieldSpans struct {
	BaseArmor          Span `json:"base_armor"`
	ProjectileUnitID   Span `json:"projectile_unit_id"`
	MaxRange           Span `json:"max_range"`
	BlastWidth         Span `json:"blast_width"`
	AttackGraphic      Span `json:"attack_graphic"`
	BlastDamage        Span `json:"blast_damage"`
	MissileType        Span `json:"missile_type"`
	TargetingType      Span `json:"targeting_type"`
	MissileHitInfo     Span `json:"missile_hit_info"`
	MissileDieInfo     Span `json:"missile_die_info"`
	AreaEffectSpecials Span `json:"area_effect_specials"`
	BallisticsRatio    Span `json:"ballistics_ratio"`
	AttackCount        Span `json:"attack_count"`
	Attacks            Span `json:"attacks"`
	ArmourCount        Span `json:"armour_count"`
	Armours            Span `json:"armours"`
}

type CreatableSummary struct {
	TrainLocationCount int                 `json:"train_location_count"`
	TrainTime0         int16               `json:"train_time_0"`
	TrainUnitID0       int16               `json:"train_unit_id_0"`
	TrainButtonID0     uint8               `json:"train_button_id_0"`
	TrainHotKeyID0     int32               `json:"train_hotkey_id_0"`
	ButtonIconID       int16               `json:"button_icon_id"`
	ButtonHotkeyAction int16               `json:"button_hotkey_action"`
	Costs              []AttributeCost     `json:"costs,omitempty"`
	TrainLocations     []TrainLocation     `json:"train_locations,omitempty"`
	FieldSpans         CreatableFieldSpans `json:"field_spans,omitempty"`
}

type AttributeCost struct {
	Index         int   `json:"index"`
	AttributeType int16 `json:"attribute_type"`
	Amount        int16 `json:"amount"`
	Flag          uint8 `json:"flag"`
	Padding       uint8 `json:"padding"`
	Active        bool  `json:"active"`
	Span          Span  `json:"span"`
}

type TrainLocation struct {
	Index       int   `json:"index"`
	TrainTime   int16 `json:"train_time"`
	TrainUnitID int16 `json:"train_unit_id"`
	TrainButton uint8 `json:"train_button"`
	TrainHotkey int32 `json:"train_hotkey"`
	Span        Span  `json:"span"`
}

type CreatableFieldSpans struct {
	TrainLocationCount Span `json:"train_location_count"`
	TrainLocations     Span `json:"train_locations"`
	TrainTime0         Span `json:"train_time_0"`
	TrainUnitID0       Span `json:"train_unit_id_0"`
	TrainButtonID0     Span `json:"train_button_id_0"`
	TrainHotKeyID0     Span `json:"train_hotkey_id_0"`
	ButtonIconID       Span `json:"button_icon_id"`
	ButtonHotkeyAction Span `json:"button_hotkey_action"`
}

type BuildingSummary struct {
	SpawningGraphic    int16            `json:"spawning_graphic,omitempty"`
	UpgradeGraphic     int16            `json:"upgrade_graphic,omitempty"`
	StackUnit          int16            `json:"stack_unit,omitempty"`
	HeadUnit           int16            `json:"head_unit,omitempty"`
	TransformUnit      int16            `json:"transform_unit,omitempty"`
	PileUnit           int16            `json:"pile_unit,omitempty"`
	ConstructionUnit   int16            `json:"construction_unit,omitempty"`
	SalvageUnit        int16            `json:"salvage_unit,omitempty"`
	TransformSound     int16            `json:"transform_sound,omitempty"`
	ConstructionSound  int16            `json:"construction_sound,omitempty"`
	GarrisonType       uint8            `json:"garrison_type,omitempty"`
	GarrisonHealRate   float32          `json:"garrison_heal_rate,omitempty"`
	GarrisonRepairRate float32          `json:"garrison_repair_rate,omitempty"`
	SalvageAttributes  []uint8          `json:"salvage_attributes,omitempty"`
	LinkedBuildings    []LinkedBuilding `json:"linked_buildings,omitempty"`
}

type LinkedBuilding struct {
	Index  int     `json:"index"`
	UnitID int16   `json:"unit_id"`
	X      float32 `json:"x"`
	Y      float32 `json:"y"`
	Active bool    `json:"active"`
	Span   Span    `json:"span"`
}

type Effect struct {
	Index       int             `json:"index"`
	Name        string          `json:"name"`
	CommandSize int             `json:"command_size"`
	Commands    []EffectCommand `json:"commands,omitempty"`
	Span        Span            `json:"span"`
}

type EffectCommand struct {
	Index    int                    `json:"index"`
	Type     uint8                  `json:"type"`
	A        int16                  `json:"a"`
	B        int16                  `json:"b"`
	C        int16                  `json:"c"`
	D        float32                `json:"d"`
	Semantic *EffectCommandSemantic `json:"semantic,omitempty"`
	Span     Span                   `json:"span"`
}

type EffectCommandSemantic struct {
	TypeName          string                   `json:"type_name"`
	Scope             string                   `json:"scope,omitempty"`
	TargetUnit        int16                    `json:"target_unit"`
	UnitClassID       int16                    `json:"unit_class_id"`
	UnitClassName     string                   `json:"unit_class_name,omitempty"`
	AttributeID       int16                    `json:"attribute_id"`
	AttributeName     string                   `json:"attribute_name,omitempty"`
	PackedTypeID      *int                     `json:"packed_type_id,omitempty"`
	PackedTypeName    string                   `json:"packed_type_name,omitempty"`
	PackedAmount      *int                     `json:"packed_amount,omitempty"`
	StringID          *int                     `json:"string_id,omitempty"`
	ResourceID        *int16                   `json:"resource_id,omitempty"`
	ResourceName      string                   `json:"resource_name,omitempty"`
	OperationID       *int16                   `json:"operation_id,omitempty"`
	OperationName     string                   `json:"operation_name,omitempty"`
	TechAttributeID   *int16                   `json:"tech_attribute_id,omitempty"`
	TechAttributeName string                   `json:"tech_attribute_name,omitempty"`
	Amount            float32                  `json:"amount"`
	ReferenceKind     string                   `json:"reference_kind,omitempty"`
	ReferenceField    string                   `json:"reference_field,omitempty"`
	ReferenceID       *int                     `json:"reference_id,omitempty"`
	References        []EffectCommandReference `json:"references,omitempty"`
	Confidence        string                   `json:"confidence"`
	Source            string                   `json:"source"`
}

type EffectCommandReference struct {
	Kind       string `json:"kind"`
	Field      string `json:"field"`
	ID         int    `json:"id"`
	Confidence string `json:"confidence"`
}

func InterpretEffectCommand(command EffectCommand) *EffectCommandSemantic {
	semantic := &EffectCommandSemantic{
		TypeName:      EffectCommandTypeName(command.Type),
		Scope:         effectCommandScope(command.Type),
		TargetUnit:    command.A,
		UnitClassID:   command.B,
		UnitClassName: EffectUnitClassName(command.B),
		AttributeID:   command.C,
		AttributeName: EffectAttributeName(command.C),
		Amount:        command.D,
		Confidence:    "typed_by_age_effect_command_oracle",
		Source:        "AGE Techs.cpp/GetEffectCmdName and Lists.cpp effect command table; genieutils TechageEffect layout: Type, TargetUnit, UnitClassID, AttributeID, Amount",
	}
	addReference := func(kind, field string, id int, confidence string) {
		semantic.References = append(semantic.References, EffectCommandReference{
			Kind:       kind,
			Field:      field,
			ID:         id,
			Confidence: confidence,
		})
		if semantic.ReferenceID == nil {
			semantic.ReferenceKind = kind
			semantic.ReferenceField = field
			semantic.ReferenceID = &semantic.References[len(semantic.References)-1].ID
			semantic.Confidence = confidence
		}
	}
	i16ptr := func(value int16) *int16 {
		v := value
		return &v
	}
	switch {
	case isLocalBuildingAttributeEffectCommand(command.Type):
		semantic.Scope = "local_building"
		semantic.Confidence = "typed_by_local_building_command_family_and_observed_dat_rows"
		semantic.Source = "Official DE update notes define types 200/201 as local-building set/add attribute commands; observed current-DE DAT rows map 202 to local-building multiply and 204 to local-building additive advanced/packed attribute rows"
		if command.A >= 0 {
			addReference("unit", "local_building_unit", int(command.A), semantic.Confidence)
		}
		if packedType, packedAmount, ok := packedAttackArmorAmount(command); ok {
			semantic.PackedTypeID = &packedType
			semantic.PackedTypeName = EffectPackedAttackArmorClassName(packedType)
			semantic.PackedAmount = &packedAmount
		}
	case isAttributeEffectCommand(command.Type):
		if command.A >= 0 {
			addReference("unit", "target_unit", int(command.A), "typed_by_age_effect_command_oracle")
		}
		if packedType, packedAmount, ok := packedAttackArmorAmount(command); ok {
			semantic.PackedTypeID = &packedType
			semantic.PackedTypeName = EffectPackedAttackArmorClassName(packedType)
			semantic.PackedAmount = &packedAmount
		}
	case isResourceModifierEffectCommand(command.Type):
		semantic.ResourceID = i16ptr(command.A)
		semantic.ResourceName = EffectResourceName(command.A)
		semantic.OperationID = i16ptr(command.B)
		semantic.OperationName = EffectOperationName(command.B)
	case isEnableDisableUnitEffectCommand(command.Type):
		if command.A >= 0 {
			addReference("unit", "target_unit", int(command.A), "typed_by_age_effect_command_oracle")
		}
	case isUpgradeUnitEffectCommand(command.Type):
		if command.A >= 0 {
			addReference("unit", "source_unit", int(command.A), "typed_by_age_effect_command_oracle")
		}
		if command.B >= 0 {
			addReference("unit", "target_unit", int(command.B), "typed_by_age_effect_command_oracle")
		}
	case isResourceMultiplierEffectCommand(command.Type):
		semantic.ResourceID = i16ptr(command.A)
		semantic.ResourceName = EffectResourceName(command.A)
	case isSpawnUnitEffectCommand(command.Type):
		if command.A >= 0 {
			addReference("unit", "spawn_unit", int(command.A), "typed_by_age_effect_command_oracle")
		}
		if command.B >= 0 {
			addReference("unit", "spawn_building", int(command.B), "typed_by_age_effect_command_oracle")
		}
	case isModifyTechEffectCommand(command.Type):
		semantic.TechAttributeID = i16ptr(command.B)
		semantic.TechAttributeName = EffectTechAttributeName(command.B)
		if command.A >= 0 {
			addReference("tech", "target_tech", int(command.A), "typed_by_age_effect_command_oracle")
		}
	case command.Type == 100 || command.Type == 101:
		semantic.ResourceID = i16ptr(command.B)
		semantic.ResourceName = EffectResourceName(command.B)
		if command.A >= 0 {
			addReference("tech", "target_tech", int(command.A), "typed_by_ugc_xs_effect_amount_constants")
		}
	case command.Type == 103:
		semantic.OperationID = i16ptr(command.B)
		semantic.OperationName = EffectOperationName(command.B)
		if command.A >= 0 {
			addReference("tech", "target_tech", int(command.A), "typed_by_ugc_xs_effect_amount_constants")
		}
	case command.Type == 102:
		id := int(command.D)
		if command.D == float32(id) && id >= 0 {
			addReference("tech", "amount", id, "typed_by_upstream_effect_type_and_integral_amount")
		}
	}
	if command.Type == 40 {
		id := int(command.D)
		if command.A >= 0 && command.C == 50 && command.D == float32(id) && id >= 0 {
			semantic.StringID = &id
			semantic.Confidence = "typed_by_observed_de_dat_rename_units_effect"
			semantic.Source = "AGE identifies type 40 as Gaia Set Attribute; observed DE DAT effect named Rename Units uses A as unit id, C as name_id attribute 50, and D as an integral string id"
		}
	}
	return semantic
}

func EffectCommandTypeName(commandType uint8) string {
	switch commandType {
	case 0:
		return "set_attribute"
	case 1:
		return "resource_modifier"
	case 2:
		return "enable_unit"
	case 3:
		return "upgrade_unit"
	case 4:
		return "add_attribute"
	case 5:
		return "attribute_multiplier"
	case 6:
		return "resource_multiplier"
	case 7:
		return "spawn_unit"
	case 8:
		return "modify_tech"
	case 9:
		return "set_player_data"
	case 10:
		return "team_set_attribute"
	case 11:
		return "team_resource_modifier"
	case 12:
		return "team_enable_unit"
	case 13:
		return "team_upgrade_unit"
	case 14:
		return "team_add_attribute"
	case 15:
		return "team_attribute_multiplier"
	case 16:
		return "team_resource_multiplier"
	case 17:
		return "team_spawn_unit"
	case 18:
		return "team_modify_tech"
	case 20:
		return "enemy_set_attribute"
	case 21:
		return "enemy_resource_modifier"
	case 22:
		return "enemy_enable_unit"
	case 23:
		return "enemy_upgrade_unit"
	case 24:
		return "enemy_add_attribute"
	case 25:
		return "enemy_attribute_multiplier"
	case 26:
		return "enemy_resource_multiplier"
	case 27:
		return "enemy_spawn_unit"
	case 28:
		return "enemy_modify_tech"
	case 30:
		return "neutral_set_attribute"
	case 31:
		return "neutral_resource_modifier"
	case 32:
		return "neutral_enable_unit"
	case 33:
		return "neutral_upgrade_unit"
	case 34:
		return "neutral_add_attribute"
	case 35:
		return "neutral_attribute_multiplier"
	case 36:
		return "neutral_resource_multiplier"
	case 37:
		return "neutral_spawn_unit"
	case 38:
		return "neutral_modify_tech"
	case 40:
		return "gaia_set_attribute"
	case 41:
		return "gaia_resource_modifier"
	case 42:
		return "gaia_enable_unit"
	case 43:
		return "gaia_upgrade_unit"
	case 44:
		return "gaia_add_attribute"
	case 45:
		return "gaia_attribute_multiplier"
	case 46:
		return "gaia_resource_multiplier"
	case 47:
		return "gaia_spawn_unit"
	case 48:
		return "gaia_modify_tech"
	case 100:
		return "set_tech_cost"
	case 101:
		return "tech_cost_modifier"
	case 102:
		return "disable_tech"
	case 103:
		return "tech_time_modifier"
	case 200:
		return "set_local_building_attribute"
	case 201:
		return "add_local_building_attribute"
	case 202:
		return "multiply_local_building_attribute"
	case 204:
		return "add_local_building_attribute_advanced"
	case 255:
		return "invalid_or_no_type"
	default:
		return "unknown"
	}
}

func effectCommandScope(commandType uint8) string {
	switch {
	case commandType >= 10 && commandType <= 18:
		return "team"
	case commandType >= 20 && commandType <= 28:
		return "enemy"
	case commandType >= 30 && commandType <= 38:
		return "neutral"
	case commandType >= 40 && commandType <= 48:
		return "gaia"
	default:
		return ""
	}
}

func effectCommandBaseType(commandType uint8) uint8 {
	switch {
	case commandType >= 10 && commandType <= 18:
		return commandType - 10
	case commandType >= 20 && commandType <= 28:
		return commandType - 20
	case commandType >= 30 && commandType <= 38:
		return commandType - 30
	case commandType >= 40 && commandType <= 48:
		return commandType - 40
	default:
		return commandType
	}
}

func isAttributeEffectCommand(commandType uint8) bool {
	base := effectCommandBaseType(commandType)
	return base == 0 || base == 4 || base == 5
}

func isLocalBuildingAttributeEffectCommand(commandType uint8) bool {
	switch commandType {
	case 200, 201, 202, 204:
		return true
	default:
		return false
	}
}

func isResourceModifierEffectCommand(commandType uint8) bool {
	return effectCommandBaseType(commandType) == 1
}

func isEnableDisableUnitEffectCommand(commandType uint8) bool {
	return effectCommandBaseType(commandType) == 2
}

func isUpgradeUnitEffectCommand(commandType uint8) bool {
	return effectCommandBaseType(commandType) == 3
}

func isResourceMultiplierEffectCommand(commandType uint8) bool {
	return effectCommandBaseType(commandType) == 6
}

func isSpawnUnitEffectCommand(commandType uint8) bool {
	return effectCommandBaseType(commandType) == 7
}

func isModifyTechEffectCommand(commandType uint8) bool {
	return effectCommandBaseType(commandType) == 8
}

func packedAttackArmorAmount(command EffectCommand) (int, int, bool) {
	if command.C != 8 && command.C != 9 {
		return 0, 0, false
	}
	raw := int(command.D)
	if command.D != float32(raw) || raw < math.MinInt16 || raw > math.MaxInt16 {
		return 0, 0, false
	}
	if raw >= 0 {
		return int(uint16(raw) >> 8), int(int16(raw) & 0xFF), true
	}
	positive := -raw
	if positive > math.MaxInt16 {
		return 0, 0, false
	}
	return int(uint16(positive) >> 8), -int(int16(positive) & 0xFF), true
}

func EffectPackedAttackArmorClassName(classID int) string {
	switch classID {
	case 3:
		return "pierce"
	case 4:
		return "melee"
	default:
		return ""
	}
}

func EffectPackedAttackArmorClassID(name string) (int, bool) {
	switch normalizedEffectOperandName(name) {
	case "pierce", "pierce_attack", "pierce_armor", "pierce_armour":
		return 3, true
	case "melee", "melee_attack", "melee_armor", "melee_armour":
		return 4, true
	default:
		return 0, false
	}
}

func EffectResourceName(resourceID int16) string {
	switch resourceID {
	case 0:
		return "food"
	case 1:
		return "wood"
	case 2:
		return "stone"
	case 3:
		return "gold"
	case 4:
		return "population_cap"
	case 6:
		return "current_age"
	case 7:
		return "relics"
	case 11:
		return "population"
	case 13:
		return "discovery"
	case 20:
		return "kills"
	case 21:
		return "research_count"
	case 22:
		return "exploration"
	case 37:
		return "civilian_population"
	case 40:
		return "military_population"
	case 41:
		return "conversions"
	case 43:
		return "razings"
	case 44:
		return "kill_ratio"
	case 45:
		return "player_killed"
	case 53:
		return "tribute"
	case 84:
		return "starting_villagers"
	case 91:
		return "starting_food"
	case 92:
		return "starting_wood"
	case 93:
		return "starting_stone"
	case 94:
		return "starting_gold"
	case 152:
		return "value_killed_by_others"
	case 153:
		return "value_razed_by_others"
	case 154:
		return "killed_by_others"
	case 155:
		return "razed_by_others"
	case 164:
		return "value_current_units"
	case 165:
		return "value_current_buildings"
	case 166:
		return "food_total"
	case 167:
		return "wood_total"
	case 168:
		return "stone_total"
	case 169:
		return "gold_total"
	case 170:
		return "total_value_of_kills"
	case 171:
		return "total_tribute_received"
	case 172:
		return "total_value_of_razings"
	case 173:
		return "total_castles_built"
	case 174:
		return "total_wonders_built"
	case 175:
		return "tribute_score"
	case 185:
		return "food_score"
	case 186:
		return "wood_score"
	case 187:
		return "stone_score"
	case 188:
		return "gold_score"
	case 203:
		return "map_reveal"
	case 209:
		return "temporary_map_reveal"
	case 300:
		return "gaia_kills"
	case 301, 302, 303, 304, 305, 306, 307, 308:
		return fmt.Sprintf("player_%d_kills", resourceID-300)
	case 325:
		return "kills_by_gaia"
	case 326, 327, 328, 329, 330, 331, 332, 333:
		return fmt.Sprintf("kills_by_player_%d", resourceID-325)
	case 350:
		return "gaia_razings"
	case 351, 352, 353, 354, 355, 356, 357, 358:
		return fmt.Sprintf("player_%d_razings", resourceID-350)
	case 375:
		return "razings_by_gaia"
	case 376, 377, 378, 379, 380, 381, 382, 383:
		return fmt.Sprintf("razings_by_player_%d", resourceID-375)
	case 400:
		return "gaia_kill_value"
	case 401, 402, 403, 404, 405, 406, 407, 408:
		return fmt.Sprintf("player_%d_kill_value", resourceID-400)
	case 425:
		return "gaia_razing_value"
	case 426, 427, 428, 429, 430, 431, 432, 433:
		return fmt.Sprintf("player_%d_razing_value", resourceID-425)
	case 450:
		return "gaia_tribute"
	case 451, 452, 453, 454, 455, 456, 457, 458:
		return fmt.Sprintf("player_%d_tribute", resourceID-450)
	case 475:
		return "tribute_from_gaia"
	case 476, 477, 478, 479, 480, 481, 482, 483:
		return fmt.Sprintf("tribute_from_player_%d", resourceID-475)
	default:
		return ""
	}
}

func EffectResourceID(name string) (int16, bool) {
	normalized := normalizedEffectOperandName(name)
	switch normalized {
	case "food":
		return 0, true
	case "wood":
		return 1, true
	case "stone":
		return 2, true
	case "gold":
		return 3, true
	case "population_cap":
		return 4, true
	case "current_age":
		return 6, true
	case "relics":
		return 7, true
	case "population":
		return 11, true
	case "discovery":
		return 13, true
	case "kills":
		return 20, true
	case "research_count":
		return 21, true
	case "exploration":
		return 22, true
	case "civilian_population":
		return 37, true
	case "military_population":
		return 40, true
	case "conversions":
		return 41, true
	case "razings":
		return 43, true
	case "kill_ratio":
		return 44, true
	case "player_killed":
		return 45, true
	case "tribute":
		return 53, true
	case "starting_villagers":
		return 84, true
	case "starting_food":
		return 91, true
	case "starting_wood":
		return 92, true
	case "starting_stone":
		return 93, true
	case "starting_gold":
		return 94, true
	case "value_killed_by_others":
		return 152, true
	case "value_razed_by_others":
		return 153, true
	case "killed_by_others":
		return 154, true
	case "razed_by_others":
		return 155, true
	case "value_current_units":
		return 164, true
	case "value_current_buildings":
		return 165, true
	case "food_total":
		return 166, true
	case "wood_total":
		return 167, true
	case "stone_total":
		return 168, true
	case "gold_total":
		return 169, true
	case "total_value_of_kills":
		return 170, true
	case "total_tribute_received":
		return 171, true
	case "total_value_of_razings":
		return 172, true
	case "total_castles_built":
		return 173, true
	case "total_wonders_built":
		return 174, true
	case "tribute_score":
		return 175, true
	case "food_score":
		return 185, true
	case "wood_score":
		return 186, true
	case "stone_score":
		return 187, true
	case "gold_score":
		return 188, true
	case "map_reveal":
		return 203, true
	case "temporary_map_reveal":
		return 209, true
	case "gaia_kills":
		return 300, true
	case "kills_by_gaia":
		return 325, true
	case "gaia_razings":
		return 350, true
	case "razings_by_gaia":
		return 375, true
	case "gaia_kill_value":
		return 400, true
	case "gaia_razing_value":
		return 425, true
	case "gaia_tribute":
		return 450, true
	case "tribute_from_gaia":
		return 475, true
	}
	var player int
	if _, err := fmt.Sscanf(normalized, "player_%d_kills", &player); err == nil && player >= 1 && player <= 8 {
		return int16(300 + player), true
	}
	if _, err := fmt.Sscanf(normalized, "kills_by_player_%d", &player); err == nil && player >= 1 && player <= 8 {
		return int16(325 + player), true
	}
	if _, err := fmt.Sscanf(normalized, "player_%d_razings", &player); err == nil && player >= 1 && player <= 8 {
		return int16(350 + player), true
	}
	if _, err := fmt.Sscanf(normalized, "razings_by_player_%d", &player); err == nil && player >= 1 && player <= 8 {
		return int16(375 + player), true
	}
	if _, err := fmt.Sscanf(normalized, "player_%d_kill_value", &player); err == nil && player >= 1 && player <= 8 {
		return int16(400 + player), true
	}
	if _, err := fmt.Sscanf(normalized, "player_%d_razing_value", &player); err == nil && player >= 1 && player <= 8 {
		return int16(425 + player), true
	}
	if _, err := fmt.Sscanf(normalized, "player_%d_tribute", &player); err == nil && player >= 1 && player <= 8 {
		return int16(450 + player), true
	}
	if _, err := fmt.Sscanf(normalized, "tribute_from_player_%d", &player); err == nil && player >= 1 && player <= 8 {
		return int16(475 + player), true
	}
	return 0, false
}

func EffectOperationName(operationID int16) string {
	switch operationID {
	case 0:
		return "set"
	case 1:
		return "add"
	default:
		return ""
	}
}

func EffectOperationID(name string) (int16, bool) {
	switch normalizedEffectOperandName(name) {
	case "set":
		return 0, true
	case "add":
		return 1, true
	default:
		return 0, false
	}
}

func EffectUnitClassName(classID int16) string {
	switch classID {
	case -1:
		return "all_units"
	case 900:
		return "archer"
	case 901:
		return "artifact"
	case 902:
		return "trade_boat"
	case 903:
		return "building"
	case 904:
		return "villager"
	case 905:
		return "sea_fish"
	case 906:
		return "infantry"
	case 907:
		return "forage_bush"
	case 908:
		return "stone_mine"
	case 909:
		return "prey_animal"
	case 910:
		return "predator_animal"
	case 911:
		return "miscellaneous"
	case 912:
		return "cavalry"
	case 913:
		return "siege_weapon"
	case 914:
		return "terrain"
	case 915:
		return "tree"
	case 916:
		return "tree_stump"
	case 917:
		return "healer"
	case 918:
		return "monk"
	case 919:
		return "trade_cart"
	case 920:
		return "transport_ship"
	case 921:
		return "fishing_boat"
	case 922:
		return "warship"
	case 923:
		return "conquistador"
	case 924:
		return "war_elephant"
	case 925:
		return "hero"
	case 926:
		return "elephant_archer"
	case 927:
		return "wall"
	case 928:
		return "phalanx"
	case 929:
		return "domestic_animal"
	case 930:
		return "flag"
	case 931:
		return "deep_sea_fish"
	case 932:
		return "gold_mine"
	case 933:
		return "shore_fish"
	case 934:
		return "cliff"
	case 935:
		return "petard"
	case 936:
		return "cavalry_archer"
	case 937:
		return "doppelganger"
	case 938:
		return "bird"
	case 939:
		return "gate"
	case 940:
		return "salvage_pile"
	case 941:
		return "resource_pile"
	case 942:
		return "relic"
	case 943:
		return "monk_with_relic"
	case 944:
		return "hand_cannoneer"
	case 945:
		return "two_handed_swordsman"
	case 946:
		return "pikeman"
	case 947:
		return "scout_cavalry"
	case 948:
		return "ore_mine"
	case 949:
		return "farm"
	case 950:
		return "spearman"
	case 951:
		return "packed_unit"
	case 952:
		return "tower"
	case 953:
		return "boarding_ship"
	case 954:
		return "unpacked_siege_unit"
	case 955:
		return "scorpion"
	case 956:
		return "raider"
	case 957:
		return "cavalry_raider"
	case 958:
		return "livestock"
	case 959:
		return "king"
	case 960:
		return "misc_building"
	case 961:
		return "controlled_animal"
	case 963:
		return "gold_fish"
	case 964:
		return "land_mine"
	default:
		return ""
	}
}

func EffectUnitClassID(name string) (int16, bool) {
	normalized := normalizedEffectOperandName(name)
	normalized = strings.TrimSuffix(normalized, "_class")
	switch normalized {
	case "all_units", "all_unit", "all_units_class":
		return -1, true
	case "archer":
		return 900, true
	case "artifact":
		return 901, true
	case "trade_boat":
		return 902, true
	case "building":
		return 903, true
	case "villager":
		return 904, true
	case "sea_fish", "ocean_fish":
		return 905, true
	case "infantry":
		return 906, true
	case "forage_bush":
		return 907, true
	case "stone_mine":
		return 908, true
	case "prey_animal", "deer":
		return 909, true
	case "predator_animal", "boar":
		return 910, true
	case "miscellaneous":
		return 911, true
	case "cavalry":
		return 912, true
	case "siege_weapon":
		return 913, true
	case "terrain":
		return 914, true
	case "tree":
		return 915, true
	case "tree_stump":
		return 916, true
	case "healer":
		return 917, true
	case "monk", "monastery":
		return 918, true
	case "trade_cart":
		return 919, true
	case "transport_ship":
		return 920, true
	case "fishing_boat", "fishing_ship":
		return 921, true
	case "warship":
		return 922, true
	case "conquistador":
		return 923, true
	case "war_elephant":
		return 924, true
	case "hero":
		return 925, true
	case "elephant_archer":
		return 926, true
	case "wall":
		return 927, true
	case "phalanx":
		return 928, true
	case "domestic_animal":
		return 929, true
	case "flag":
		return 930, true
	case "deep_sea_fish":
		return 931, true
	case "gold_mine":
		return 932, true
	case "shore_fish":
		return 933, true
	case "cliff":
		return 934, true
	case "petard":
		return 935, true
	case "cavalry_archer":
		return 936, true
	case "doppelganger":
		return 937, true
	case "bird":
		return 938, true
	case "gate":
		return 939, true
	case "salvage_pile":
		return 940, true
	case "resource_pile":
		return 941, true
	case "relic":
		return 942, true
	case "monk_with_relic":
		return 943, true
	case "hand_cannoneer":
		return 944, true
	case "two_handed_swordsman":
		return 945, true
	case "pikeman":
		return 946, true
	case "scout_cavalry":
		return 947, true
	case "ore_mine":
		return 948, true
	case "farm":
		return 949, true
	case "spearman":
		return 950, true
	case "packed_unit":
		return 951, true
	case "tower":
		return 952, true
	case "boarding_ship":
		return 953, true
	case "unpacked_siege_unit":
		return 954, true
	case "scorpion":
		return 955, true
	case "raider":
		return 956, true
	case "cavalry_raider":
		return 957, true
	case "livestock":
		return 958, true
	case "king":
		return 959, true
	case "misc_building":
		return 960, true
	case "controlled_animal":
		return 961, true
	case "gold_fish":
		return 963, true
	case "land_mine":
		return 964, true
	default:
		return 0, false
	}
}

func EffectTechAttributeName(attributeID int16) string {
	switch attributeID {
	case -1:
		return "set_time"
	case -2:
		return "add_time"
	case -3:
		return "multiply_time"
	case 0:
		return "set_food_cost"
	case 1:
		return "set_wood_cost"
	case 2:
		return "set_stone_cost"
	case 3:
		return "set_gold_cost"
	case 4:
		return "set_location"
	case 5:
		return "set_button"
	case 6:
		return "set_icon"
	case 7:
		return "set_name"
	case 8:
		return "set_description"
	case 9:
		return "set_stacking"
	case 10:
		return "set_stacking_research_cap"
	case 11:
		return "set_hotkey"
	case 12:
		return "set_state"
	case 13:
		return "multiply_food_cost"
	case 14:
		return "multiply_wood_cost"
	case 15:
		return "multiply_stone_cost"
	case 16:
		return "multiply_gold_cost"
	case 17:
		return "multiply_all_costs"
	case 18:
		return "set_effect"
	case 16384:
		return "add_food_cost"
	case 16385:
		return "add_wood_cost"
	case 16386:
		return "add_stone_cost"
	case 16387:
		return "add_gold_cost"
	default:
		return ""
	}
}

func EffectTechAttributeID(name string) (int16, bool) {
	switch normalizedEffectOperandName(name) {
	case "set_time":
		return -1, true
	case "add_time":
		return -2, true
	case "multiply_time", "mul_time":
		return -3, true
	case "set_food_cost":
		return 0, true
	case "set_wood_cost":
		return 1, true
	case "set_stone_cost":
		return 2, true
	case "set_gold_cost":
		return 3, true
	case "set_location":
		return 4, true
	case "set_button":
		return 5, true
	case "set_icon":
		return 6, true
	case "set_name":
		return 7, true
	case "set_description":
		return 8, true
	case "set_stacking":
		return 9, true
	case "set_stacking_research_cap":
		return 10, true
	case "set_hotkey":
		return 11, true
	case "set_state":
		return 12, true
	case "multiply_food_cost", "mul_food_cost":
		return 13, true
	case "multiply_wood_cost", "mul_wood_cost":
		return 14, true
	case "multiply_stone_cost", "mul_stone_cost":
		return 15, true
	case "multiply_gold_cost", "mul_gold_cost":
		return 16, true
	case "multiply_all_costs", "mul_all_costs":
		return 17, true
	case "set_effect":
		return 18, true
	case "add_food_cost":
		return 16384, true
	case "add_wood_cost":
		return 16385, true
	case "add_stone_cost":
		return 16386, true
	case "add_gold_cost":
		return 16387, true
	default:
		return 0, false
	}
}

func normalizedEffectOperandName(name string) string {
	name = camelToSnake(strings.TrimSpace(name))
	name = strings.TrimPrefix(name, "c_attribute_")
	name = strings.TrimPrefix(name, "cattr_")
	name = strings.TrimPrefix(name, "c_attr_")
	name = strings.TrimPrefix(name, "c_")
	name = strings.ReplaceAll(name, "-", "_")
	name = strings.ReplaceAll(name, " ", "_")
	for strings.Contains(name, "__") {
		name = strings.ReplaceAll(name, "__", "_")
	}
	return name
}

func camelToSnake(name string) string {
	var out strings.Builder
	var prevLower bool
	var prevDigit bool
	for _, r := range name {
		switch {
		case unicode.IsUpper(r):
			if prevLower || prevDigit {
				out.WriteByte('_')
			}
			out.WriteRune(unicode.ToLower(r))
			prevLower = false
			prevDigit = false
		case unicode.IsDigit(r):
			if prevLower {
				out.WriteByte('_')
			}
			out.WriteRune(r)
			prevLower = false
			prevDigit = true
		default:
			out.WriteRune(unicode.ToLower(r))
			prevLower = unicode.IsLower(r)
			prevDigit = false
		}
	}
	return out.String()
}

func EffectAttributeName(attributeID int16) string {
	switch attributeID {
	case -1:
		return "invalid"
	case 0:
		return "hit_points"
	case 1:
		return "line_of_sight"
	case 2:
		return "garrison_capacity"
	case 3:
		return "unit_size_x"
	case 4:
		return "unit_size_y"
	case 5:
		return "movement_speed"
	case 6:
		return "rotation_speed"
	case 7:
		return "unused"
	case 8:
		return "armor"
	case 9:
		return "attack"
	case 10:
		return "attack_reload_time"
	case 11:
		return "accuracy_percent"
	case 12:
		return "max_range"
	case 13:
		return "work_rate"
	case 14:
		return "carry_capacity"
	case 15:
		return "base_armor"
	case 16:
		return "projectile_unit"
	case 17:
		return "icon_graphics_angle"
	case 18:
		return "terrain_defense_bonus"
	case 19:
		return "enable_smart_projectiles"
	case 20:
		return "min_range"
	case 21:
		return "main_resource_storage"
	case 22:
		return "blast_width"
	case 23:
		return "search_radius"
	case 50:
		return "name_id"
	case 100:
		return "resource_costs"
	case 101:
		return "train_time"
	case 102:
		return "total_missiles"
	case 103:
		return "food_costs"
	case 104:
		return "wood_costs"
	case 105:
		return "gold_costs"
	case 106:
		return "stone_costs"
	case 107:
		return "max_total_missiles"
	case 108:
		return "garrison_heal_rate"
	case 109:
		return "regeneration_rate"
	default:
		return ""
	}
}

func EffectAttributeID(name string) (int16, bool) {
	switch normalizedEffectOperandName(name) {
	case "invalid":
		return -1, true
	case "hit_points", "hitpoints":
		return 0, true
	case "line_of_sight":
		return 1, true
	case "garrison_capacity":
		return 2, true
	case "unit_size_x":
		return 3, true
	case "unit_size_y":
		return 4, true
	case "movement_speed":
		return 5, true
	case "rotation_speed":
		return 6, true
	case "unused":
		return 7, true
	case "armor":
		return 8, true
	case "attack":
		return 9, true
	case "attack_reload_time":
		return 10, true
	case "accuracy_percent":
		return 11, true
	case "max_range":
		return 12, true
	case "work_rate":
		return 13, true
	case "carry_capacity":
		return 14, true
	case "base_armor":
		return 15, true
	case "projectile_unit":
		return 16, true
	case "icon_graphics_angle":
		return 17, true
	case "terrain_defense_bonus":
		return 18, true
	case "enable_smart_projectiles":
		return 19, true
	case "min_range", "minimum_range":
		return 20, true
	case "main_resource_storage", "amount_first_storage":
		return 21, true
	case "blast_width":
		return 22, true
	case "search_radius":
		return 23, true
	case "resource_costs":
		return 100, true
	case "train_time":
		return 101, true
	case "total_missiles":
		return 102, true
	case "food_costs":
		return 103, true
	case "wood_costs":
		return 104, true
	case "gold_costs":
		return 105, true
	case "stone_costs":
		return 106, true
	case "max_total_missiles":
		return 107, true
	case "garrison_heal_rate":
		return 108, true
	case "regeneration_rate":
		return 109, true
	case "name_id":
		return 50, true
	default:
		return 0, false
	}
}

type Tech struct {
	Index                  int                    `json:"index"`
	RequiredTechs          []int16                `json:"required_techs,omitempty"`
	ResourceCosts          []ResearchResourceCost `json:"resource_costs,omitempty"`
	RequiredTechCount      int16                  `json:"required_tech_count"`
	Civ                    int16                  `json:"civ"`
	FullTechMode           int16                  `json:"full_tech_mode"`
	LanguageDLLName        int32                  `json:"language_dll_name"`
	LanguageDLLDescription int32                  `json:"language_dll_description"`
	EffectID               int16                  `json:"effect_id"`
	Type                   int16                  `json:"type"`
	IconID                 int16                  `json:"icon_id"`
	LanguageDLLHelp        int32                  `json:"language_dll_help"`
	LanguageDLLTechTree    int32                  `json:"language_dll_tech_tree"`
	Name                   string                 `json:"name"`
	Repeatable             uint8                  `json:"repeatable"`
	ResearchLocations      []ResearchLocation     `json:"research_locations,omitempty"`
	Span                   Span                   `json:"span"`
}

type ResearchResourceCost struct {
	Type   int16 `json:"type"`
	Amount int16 `json:"amount"`
	Flag   uint8 `json:"flag"`
}

type ResearchLocation struct {
	LocationID   int16 `json:"location_id"`
	ResearchTime int16 `json:"research_time"`
	ButtonID     uint8 `json:"button_id"`
	HotKeyID     int32 `json:"hotkey_id"`
}

type Metrics struct {
	TimeSlice         int32 `json:"time_slice"`
	UnitKillRate      int32 `json:"unit_kill_rate"`
	UnitKillTotal     int32 `json:"unit_kill_total"`
	UnitHitPointRate  int32 `json:"unit_hit_point_rate"`
	UnitHitPointTotal int32 `json:"unit_hit_point_total"`
	RazingKillRate    int32 `json:"razing_kill_rate"`
	RazingKillTotal   int32 `json:"razing_kill_total"`
	Span              Span  `json:"span"`
}

type TechTree struct {
	AgeCount            int                  `json:"age_count"`
	BuildingCount       int                  `json:"building_count"`
	UnitCount           int                  `json:"unit_count"`
	ResearchCount       int                  `json:"research_count"`
	TotalUnitTechGroups int32                `json:"total_unit_tech_groups"`
	Ages                []TechTreeAge        `json:"ages,omitempty"`
	BuildingConnections []BuildingConnection `json:"building_connections,omitempty"`
	UnitConnections     []UnitConnection     `json:"unit_connections,omitempty"`
	ResearchConnections []ResearchConnection `json:"research_connections,omitempty"`
	Span                Span                 `json:"span"`
}

type TechTreeCommon struct {
	SlotsUsed    int32   `json:"slots_used"`
	UnitResearch []int32 `json:"unit_research,omitempty"`
	Mode         []int32 `json:"mode,omitempty"`
}

type TechTreeAge struct {
	Index              int            `json:"index"`
	ID                 int32          `json:"id"`
	Status             uint8          `json:"status"`
	Buildings          []int32        `json:"buildings,omitempty"`
	Units              []int32        `json:"units,omitempty"`
	Techs              []int32        `json:"techs,omitempty"`
	Common             TechTreeCommon `json:"common"`
	NumBuildingLevels  uint8          `json:"num_building_levels"`
	BuildingsPerZone   []uint8        `json:"buildings_per_zone,omitempty"`
	GroupLengthPerZone []uint8        `json:"group_length_per_zone,omitempty"`
	MaxAgeLength       uint8          `json:"max_age_length"`
	LineMode           int32          `json:"line_mode"`
	Span               Span           `json:"span"`
}

type BuildingConnection struct {
	Index            int            `json:"index"`
	ID               int32          `json:"id"`
	Status           uint8          `json:"status"`
	Buildings        []int32        `json:"buildings,omitempty"`
	Units            []int32        `json:"units,omitempty"`
	Techs            []int32        `json:"techs,omitempty"`
	Common           TechTreeCommon `json:"common"`
	LocationInAge    uint8          `json:"location_in_age"`
	UnitsTechsTotal  []uint8        `json:"units_techs_total,omitempty"`
	UnitsTechsFirst  []uint8        `json:"units_techs_first,omitempty"`
	LineMode         int32          `json:"line_mode"`
	EnablingResearch int32          `json:"enabling_research"`
	Span             Span           `json:"span"`
}

type UnitConnection struct {
	Index            int            `json:"index"`
	ID               int32          `json:"id"`
	Status           uint8          `json:"status"`
	UpperBuilding    int32          `json:"upper_building"`
	Common           TechTreeCommon `json:"common"`
	VerticalLine     int32          `json:"vertical_line"`
	Units            []int32        `json:"units,omitempty"`
	LocationInAge    int32          `json:"location_in_age"`
	RequiredResearch int32          `json:"required_research"`
	LineMode         int32          `json:"line_mode"`
	EnablingResearch int32          `json:"enabling_research"`
	Span             Span           `json:"span"`
}

type ResearchConnection struct {
	Index         int            `json:"index"`
	ID            int32          `json:"id"`
	Status        uint8          `json:"status"`
	UpperBuilding int32          `json:"upper_building"`
	Buildings     []int32        `json:"buildings,omitempty"`
	Units         []int32        `json:"units,omitempty"`
	Techs         []int32        `json:"techs,omitempty"`
	Common        TechTreeCommon `json:"common"`
	VerticalLine  int32          `json:"vertical_line"`
	LocationInAge int32          `json:"location_in_age"`
	LineMode      int32          `json:"line_mode"`
	Span          Span           `json:"span"`
}

type UnitPatch struct {
	Class                       *int16                `json:"class,omitempty"`
	HitPoints                   *int16                `json:"hit_points,omitempty"`
	LineOfSight                 *float32              `json:"line_of_sight,omitempty"`
	MovementType                *uint8                `json:"movement_type,omitempty"`
	StandingGraphic1            *int16                `json:"standing_graphic_1,omitempty"`
	StandingGraphic2            *int16                `json:"standing_graphic_2,omitempty"`
	DyingGraphic                *int16                `json:"dying_graphic,omitempty"`
	BloodUnitID                 *int16                `json:"blood_unit_id,omitempty"`
	IconID                      *int16                `json:"icon_id,omitempty"`
	Enabled                     *uint8                `json:"enabled,omitempty"`
	Attributes                  []UnitAttributePatch  `json:"attributes,omitempty"`
	DamageGraphics              []DamageGraphicPatch  `json:"damage_graphics,omitempty"`
	SetDamageGraphics           *[]DamageGraphicRow   `json:"set_damage_graphics,omitempty"`
	Type50ProjectileUnitID      *int16                `json:"type50_projectile_unit_id,omitempty"`
	Type50MaxRange              *float32              `json:"type50_max_range,omitempty"`
	Type50BlastWidth            *float32              `json:"type50_blast_width,omitempty"`
	Type50AttackGraphic         *int16                `json:"type50_attack_graphic,omitempty"`
	Type50BlastDamage           *float32              `json:"type50_blast_damage,omitempty"`
	Type50Attacks               []WeaponInfoPatch     `json:"type50_attacks,omitempty"`
	Type50Armours               []WeaponInfoPatch     `json:"type50_armours,omitempty"`
	SetType50Attacks            *[]WeaponInfoRow      `json:"set_type50_attacks,omitempty"`
	SetType50Armours            *[]WeaponInfoRow      `json:"set_type50_armours,omitempty"`
	TrainTime0                  *int16                `json:"train_time_0,omitempty"`
	TrainUnitID0                *int16                `json:"train_unit_id_0,omitempty"`
	TrainButtonID0              *uint8                `json:"train_button_id_0,omitempty"`
	TrainHotKeyID0              *int32                `json:"train_hotkey_id_0,omitempty"`
	CreatableButtonIconID       *int16                `json:"creatable_button_icon_id,omitempty"`
	CreatableButtonHotkeyAction *int16                `json:"creatable_button_hotkey_action,omitempty"`
	Costs                       []AttributeCostPatch  `json:"costs,omitempty"`
	TrainLocations              []TrainLocationPatch  `json:"train_locations,omitempty"`
	SetTrainLocations           *[]TrainLocationRow   `json:"set_train_locations,omitempty"`
	SetDropSites                *[]int16              `json:"set_drop_sites,omitempty"`
	LinkedBuildings             []LinkedBuildingPatch `json:"linked_buildings,omitempty"`
	Tasks                       []TaskPatch           `json:"tasks,omitempty"`
	SetTasks                    *[]TaskRow            `json:"set_tasks,omitempty"`
}

type UnitAttributePatch struct {
	Index         int      `json:"index"`
	AttributeType *uint16  `json:"attribute_type,omitempty"`
	Amount        *float32 `json:"amount,omitempty"`
	Flag          *uint8   `json:"flag,omitempty"`
}

type DamageGraphicPatch struct {
	Index         int     `json:"index"`
	GraphicID     *uint16 `json:"graphic_id,omitempty"`
	DamagePercent *uint16 `json:"damage_percent,omitempty"`
	Flag          *uint8  `json:"flag,omitempty"`
}

type DamageGraphicRow struct {
	GraphicID     uint16 `json:"graphic_id"`
	DamagePercent uint16 `json:"damage_percent"`
	Flag          uint8  `json:"flag"`
}

type WeaponInfoPatch struct {
	Index int    `json:"index"`
	Class *int16 `json:"class,omitempty"`
	Value *int16 `json:"value,omitempty"`
}

type WeaponInfoRow struct {
	Class int16 `json:"class"`
	Value int16 `json:"value"`
}

type AttributeCostPatch struct {
	Index         int    `json:"index"`
	AttributeType *int16 `json:"attribute_type,omitempty"`
	Amount        *int16 `json:"amount,omitempty"`
	Flag          *uint8 `json:"flag,omitempty"`
	Padding       *uint8 `json:"padding,omitempty"`
}

type TrainLocationPatch struct {
	Index       int    `json:"index"`
	TrainTime   *int16 `json:"train_time,omitempty"`
	TrainUnitID *int16 `json:"train_unit_id,omitempty"`
	TrainButton *uint8 `json:"train_button,omitempty"`
	TrainHotkey *int32 `json:"train_hotkey,omitempty"`
}

type TrainLocationRow struct {
	TrainTime   int16 `json:"train_time"`
	TrainUnitID int16 `json:"train_unit_id"`
	TrainButton uint8 `json:"train_button"`
	TrainHotkey int32 `json:"train_hotkey"`
}

type LinkedBuildingPatch struct {
	Index  int      `json:"index"`
	UnitID *int16   `json:"unit_id,omitempty"`
	X      *float32 `json:"x,omitempty"`
	Y      *float32 `json:"y,omitempty"`
}

type TaskPatch struct {
	Index             int      `json:"index"`
	RecordType        *int16   `json:"record_type,omitempty"`
	ID                *int16   `json:"id,omitempty"`
	IsDefault         *uint8   `json:"is_default,omitempty"`
	ActionType        *int16   `json:"action_type,omitempty"`
	ObjectClass       *int16   `json:"object_class,omitempty"`
	ObjectID          *int16   `json:"object_id,omitempty"`
	TerrainID         *int16   `json:"terrain_id,omitempty"`
	AttributeTypes    []int16  `json:"attribute_types,omitempty"`
	WorkValue1        *float32 `json:"work_value_1,omitempty"`
	WorkValue2        *float32 `json:"work_value_2,omitempty"`
	WorkRange         *float32 `json:"work_range,omitempty"`
	AutoSearchTargets *uint8   `json:"auto_search_targets,omitempty"`
	SearchWaitTime    *float32 `json:"search_wait_time,omitempty"`
	EnableTargeting   *uint8   `json:"enable_targeting,omitempty"`
	CombatLevel       *uint8   `json:"combat_level,omitempty"`
}

func (patch UnitPatch) Empty() bool {
	return patch.Class == nil &&
		patch.HitPoints == nil &&
		patch.LineOfSight == nil &&
		patch.MovementType == nil &&
		patch.StandingGraphic1 == nil &&
		patch.StandingGraphic2 == nil &&
		patch.DyingGraphic == nil &&
		patch.BloodUnitID == nil &&
		patch.IconID == nil &&
		patch.Enabled == nil &&
		len(patch.Attributes) == 0 &&
		len(patch.DamageGraphics) == 0 &&
		patch.SetDamageGraphics == nil &&
		patch.Type50ProjectileUnitID == nil &&
		patch.Type50MaxRange == nil &&
		patch.Type50BlastWidth == nil &&
		patch.Type50AttackGraphic == nil &&
		patch.Type50BlastDamage == nil &&
		len(patch.Type50Attacks) == 0 &&
		len(patch.Type50Armours) == 0 &&
		patch.SetType50Attacks == nil &&
		patch.SetType50Armours == nil &&
		patch.TrainTime0 == nil &&
		patch.TrainUnitID0 == nil &&
		patch.TrainButtonID0 == nil &&
		patch.TrainHotKeyID0 == nil &&
		patch.CreatableButtonIconID == nil &&
		patch.CreatableButtonHotkeyAction == nil &&
		len(patch.Costs) == 0 &&
		len(patch.TrainLocations) == 0 &&
		patch.SetTrainLocations == nil &&
		patch.SetDropSites == nil &&
		len(patch.LinkedBuildings) == 0 &&
		len(patch.Tasks) == 0 &&
		patch.SetTasks == nil
}

type UnitPatchReport struct {
	InputCompressedBytes  int                    `json:"input_compressed_bytes"`
	OutputCompressedBytes int                    `json:"output_compressed_bytes"`
	InputInflatedBytes    int                    `json:"input_inflated_bytes"`
	OutputInflatedBytes   int                    `json:"output_inflated_bytes"`
	InflatedLengthDelta   int                    `json:"inflated_length_delta"`
	CivID                 int                    `json:"civ_id"`
	UnitID                int                    `json:"unit_id"`
	Before                UnitSummary            `json:"before"`
	After                 UnitSummary            `json:"after"`
	ChangedFields         []string               `json:"changed_fields"`
	Verified              bool                   `json:"verified"`
	Verification          aoe2.VerificationClaim `json:"verification"`
}

type Recipe struct {
	Graphics       []GraphicRecipePatch  `json:"graphics,omitempty"`
	CreateGraphic  *GraphicCreateRecipe  `json:"create_graphic,omitempty"`
	CreateGraphics []GraphicCreateRecipe `json:"create_graphics,omitempty"`
	DeleteGraphics []int                 `json:"delete_graphics,omitempty"`
	CreateUnit     *UnitCreateRecipe     `json:"create_unit,omitempty"`
	CreateUnits    []UnitCreateRecipe    `json:"create_units,omitempty"`
	Units          []UnitRecipePatch     `json:"units,omitempty"`
}

func (recipe Recipe) Empty() bool {
	return len(recipe.Graphics) == 0 &&
		recipe.CreateGraphic == nil &&
		len(recipe.CreateGraphics) == 0 &&
		len(recipe.DeleteGraphics) == 0 &&
		recipe.CreateUnit == nil &&
		len(recipe.CreateUnits) == 0 &&
		len(recipe.Units) == 0
}

type GraphicRecipePatch struct {
	ID                 int                     `json:"id"`
	Name               *string                 `json:"name,omitempty"`
	FileName           *string                 `json:"file_name,omitempty"`
	ParticleEffectName *string                 `json:"particle_effect_name,omitempty"`
	SLP                *int32                  `json:"slp,omitempty"`
	IsLoaded           *int8                   `json:"is_loaded,omitempty"`
	OldColorFlag       *int8                   `json:"old_color_flag,omitempty"`
	Layer              *int8                   `json:"layer,omitempty"`
	PlayerColor        *int8                   `json:"player_color,omitempty"`
	Rainbow            *int8                   `json:"rainbow,omitempty"`
	TransparentSelect  *int8                   `json:"transparent_selection,omitempty"`
	Coordinates        []int16                 `json:"coordinates,omitempty"`
	SoundID            *int16                  `json:"sound_id,omitempty"`
	WwiseSoundID       *uint32                 `json:"wwise_sound_id,omitempty"`
	FrameCount         *int16                  `json:"frame_count,omitempty"`
	SpeedMultiplier    *float32                `json:"speed_multiplier,omitempty"`
	FrameDuration      *float32                `json:"frame_duration,omitempty"`
	ReplayDelay        *float32                `json:"replay_delay,omitempty"`
	SequenceType       *uint8                  `json:"sequence_type,omitempty"`
	MirroringMode      *int8                   `json:"mirroring_mode,omitempty"`
	EditorFlag         *int8                   `json:"editor_flag,omitempty"`
	SetDeltas          *[]GraphicDeltaRow      `json:"set_deltas,omitempty"`
	SetAngleSounds     *[]GraphicAngleSoundRow `json:"set_angle_sounds,omitempty"`
}

type GraphicCreateRecipe struct {
	From               int                     `json:"from"`
	Name               *string                 `json:"name,omitempty"`
	FileName           *string                 `json:"file_name,omitempty"`
	ParticleEffectName *string                 `json:"particle_effect_name,omitempty"`
	SLP                *int32                  `json:"slp,omitempty"`
	IsLoaded           *int8                   `json:"is_loaded,omitempty"`
	OldColorFlag       *int8                   `json:"old_color_flag,omitempty"`
	Layer              *int8                   `json:"layer,omitempty"`
	PlayerColor        *int8                   `json:"player_color,omitempty"`
	Rainbow            *int8                   `json:"rainbow,omitempty"`
	TransparentSelect  *int8                   `json:"transparent_selection,omitempty"`
	Coordinates        []int16                 `json:"coordinates,omitempty"`
	SoundID            *int16                  `json:"sound_id,omitempty"`
	WwiseSoundID       *uint32                 `json:"wwise_sound_id,omitempty"`
	FrameCount         *int16                  `json:"frame_count,omitempty"`
	SpeedMultiplier    *float32                `json:"speed_multiplier,omitempty"`
	FrameDuration      *float32                `json:"frame_duration,omitempty"`
	ReplayDelay        *float32                `json:"replay_delay,omitempty"`
	SequenceType       *uint8                  `json:"sequence_type,omitempty"`
	MirroringMode      *int8                   `json:"mirroring_mode,omitempty"`
	EditorFlag         *int8                   `json:"editor_flag,omitempty"`
	SetDeltas          *[]GraphicDeltaRow      `json:"set_deltas,omitempty"`
	SetAngleSounds     *[]GraphicAngleSoundRow `json:"set_angle_sounds,omitempty"`
}

type UnitRecipePatch struct {
	CivID                       int                   `json:"civ_id"`
	UnitID                      int                   `json:"unit_id"`
	HitPoints                   *int16                `json:"hit_points,omitempty"`
	Class                       *int16                `json:"class,omitempty"`
	LineOfSight                 *float32              `json:"line_of_sight,omitempty"`
	MovementType                *uint8                `json:"movement_type,omitempty"`
	StandingGraphic1            *int16                `json:"standing_graphic_1,omitempty"`
	StandingGraphic2            *int16                `json:"standing_graphic_2,omitempty"`
	DyingGraphic                *int16                `json:"dying_graphic,omitempty"`
	BloodUnitID                 *int16                `json:"blood_unit_id,omitempty"`
	IconID                      *int16                `json:"icon_id,omitempty"`
	Enabled                     *uint8                `json:"enabled,omitempty"`
	Attributes                  []UnitAttributePatch  `json:"attributes,omitempty"`
	DamageGraphics              []DamageGraphicPatch  `json:"damage_graphics,omitempty"`
	SetDamageGraphics           *[]DamageGraphicRow   `json:"set_damage_graphics,omitempty"`
	Type50ProjectileUnitID      *int16                `json:"type50_projectile_unit_id,omitempty"`
	Type50MaxRange              *float32              `json:"type50_max_range,omitempty"`
	Type50BlastWidth            *float32              `json:"type50_blast_width,omitempty"`
	Type50AttackGraphic         *int16                `json:"type50_attack_graphic,omitempty"`
	Type50BlastDamage           *float32              `json:"type50_blast_damage,omitempty"`
	Type50Attacks               []WeaponInfoPatch     `json:"type50_attacks,omitempty"`
	Type50Armours               []WeaponInfoPatch     `json:"type50_armours,omitempty"`
	SetType50Attacks            *[]WeaponInfoRow      `json:"set_type50_attacks,omitempty"`
	SetType50Armours            *[]WeaponInfoRow      `json:"set_type50_armours,omitempty"`
	TrainTime0                  *int16                `json:"train_time_0,omitempty"`
	TrainUnitID0                *int16                `json:"train_unit_id_0,omitempty"`
	TrainButtonID0              *uint8                `json:"train_button_id_0,omitempty"`
	TrainHotKeyID0              *int32                `json:"train_hotkey_id_0,omitempty"`
	CreatableButtonIconID       *int16                `json:"creatable_button_icon_id,omitempty"`
	CreatableButtonHotkeyAction *int16                `json:"creatable_button_hotkey_action,omitempty"`
	Costs                       []AttributeCostPatch  `json:"costs,omitempty"`
	TrainLocations              []TrainLocationPatch  `json:"train_locations,omitempty"`
	SetTrainLocations           *[]TrainLocationRow   `json:"set_train_locations,omitempty"`
	SetDropSites                *[]int16              `json:"set_drop_sites,omitempty"`
	LinkedBuildings             []LinkedBuildingPatch `json:"linked_buildings,omitempty"`
	Tasks                       []TaskPatch           `json:"tasks,omitempty"`
	SetTasks                    *[]TaskRow            `json:"set_tasks,omitempty"`
}

type UnitCreateRecipe struct {
	FromCivID                   int                   `json:"from_civ_id"`
	FromUnitID                  int                   `json:"from_unit_id"`
	CivIDs                      []int                 `json:"civ_ids,omitempty"`
	AllCivs                     *bool                 `json:"all_civs,omitempty"`
	HitPoints                   *int16                `json:"hit_points,omitempty"`
	Class                       *int16                `json:"class,omitempty"`
	LineOfSight                 *float32              `json:"line_of_sight,omitempty"`
	MovementType                *uint8                `json:"movement_type,omitempty"`
	StandingGraphic1            *int16                `json:"standing_graphic_1,omitempty"`
	StandingGraphic2            *int16                `json:"standing_graphic_2,omitempty"`
	DyingGraphic                *int16                `json:"dying_graphic,omitempty"`
	BloodUnitID                 *int16                `json:"blood_unit_id,omitempty"`
	IconID                      *int16                `json:"icon_id,omitempty"`
	Enabled                     *uint8                `json:"enabled,omitempty"`
	Attributes                  []UnitAttributePatch  `json:"attributes,omitempty"`
	DamageGraphics              []DamageGraphicPatch  `json:"damage_graphics,omitempty"`
	SetDamageGraphics           *[]DamageGraphicRow   `json:"set_damage_graphics,omitempty"`
	Type50ProjectileUnitID      *int16                `json:"type50_projectile_unit_id,omitempty"`
	Type50MaxRange              *float32              `json:"type50_max_range,omitempty"`
	Type50BlastWidth            *float32              `json:"type50_blast_width,omitempty"`
	Type50AttackGraphic         *int16                `json:"type50_attack_graphic,omitempty"`
	Type50BlastDamage           *float32              `json:"type50_blast_damage,omitempty"`
	Type50Attacks               []WeaponInfoPatch     `json:"type50_attacks,omitempty"`
	Type50Armours               []WeaponInfoPatch     `json:"type50_armours,omitempty"`
	SetType50Attacks            *[]WeaponInfoRow      `json:"set_type50_attacks,omitempty"`
	SetType50Armours            *[]WeaponInfoRow      `json:"set_type50_armours,omitempty"`
	TrainTime0                  *int16                `json:"train_time_0,omitempty"`
	TrainUnitID0                *int16                `json:"train_unit_id_0,omitempty"`
	TrainButtonID0              *uint8                `json:"train_button_id_0,omitempty"`
	TrainHotKeyID0              *int32                `json:"train_hotkey_id_0,omitempty"`
	CreatableButtonIconID       *int16                `json:"creatable_button_icon_id,omitempty"`
	CreatableButtonHotkeyAction *int16                `json:"creatable_button_hotkey_action,omitempty"`
	Costs                       []AttributeCostPatch  `json:"costs,omitempty"`
	TrainLocations              []TrainLocationPatch  `json:"train_locations,omitempty"`
	SetTrainLocations           *[]TrainLocationRow   `json:"set_train_locations,omitempty"`
	SetDropSites                *[]int16              `json:"set_drop_sites,omitempty"`
	LinkedBuildings             []LinkedBuildingPatch `json:"linked_buildings,omitempty"`
	Tasks                       []TaskPatch           `json:"tasks,omitempty"`
	SetTasks                    *[]TaskRow            `json:"set_tasks,omitempty"`
}

type RecipeReport struct {
	InputCompressedBytes  int                    `json:"input_compressed_bytes"`
	OutputCompressedBytes int                    `json:"output_compressed_bytes"`
	InputInflatedBytes    int                    `json:"input_inflated_bytes"`
	OutputInflatedBytes   int                    `json:"output_inflated_bytes"`
	InflatedLengthDelta   int                    `json:"inflated_length_delta"`
	GraphicReports        []PatchReport          `json:"graphic_reports,omitempty"`
	CreatedGraphics       []GraphicCreateReport  `json:"created_graphics,omitempty"`
	DeletedGraphics       []GraphicDeleteReport  `json:"deleted_graphics,omitempty"`
	CreatedUnits          []UnitCreateReport     `json:"created_units,omitempty"`
	UnitReports           []UnitPatchReport      `json:"unit_reports,omitempty"`
	Verified              bool                   `json:"verified"`
	Verification          aoe2.VerificationClaim `json:"verification"`
}

type GraphicCreateReport struct {
	InputCompressedBytes  int                    `json:"input_compressed_bytes"`
	OutputCompressedBytes int                    `json:"output_compressed_bytes"`
	InputInflatedBytes    int                    `json:"input_inflated_bytes"`
	OutputInflatedBytes   int                    `json:"output_inflated_bytes"`
	InflatedLengthDelta   int                    `json:"inflated_length_delta"`
	FromGraphicID         int                    `json:"from_graphic_id"`
	NewGraphicID          int                    `json:"new_graphic_id"`
	BeforeGraphicsSize    int                    `json:"before_graphics_size"`
	AfterGraphicsSize     int                    `json:"after_graphics_size"`
	Template              GraphicSummary         `json:"template"`
	Created               GraphicSummary         `json:"created"`
	ChangedFields         []string               `json:"changed_fields"`
	Verified              bool                   `json:"verified"`
	Verification          aoe2.VerificationClaim `json:"verification"`
}

type GraphicDeleteReport struct {
	InputCompressedBytes  int                    `json:"input_compressed_bytes"`
	OutputCompressedBytes int                    `json:"output_compressed_bytes"`
	InputInflatedBytes    int                    `json:"input_inflated_bytes"`
	OutputInflatedBytes   int                    `json:"output_inflated_bytes"`
	InflatedLengthDelta   int                    `json:"inflated_length_delta"`
	GraphicID             int                    `json:"graphic_id"`
	BeforeGraphicsSize    int                    `json:"before_graphics_size"`
	AfterGraphicsSize     int                    `json:"after_graphics_size"`
	RemovedRecordBytes    int                    `json:"removed_record_bytes"`
	Before                GraphicSummary         `json:"before"`
	Verified              bool                   `json:"verified"`
	Verification          aoe2.VerificationClaim `json:"verification"`
}

type UnitCreateReport struct {
	InputCompressedBytes  int                    `json:"input_compressed_bytes"`
	OutputCompressedBytes int                    `json:"output_compressed_bytes"`
	InputInflatedBytes    int                    `json:"input_inflated_bytes"`
	OutputInflatedBytes   int                    `json:"output_inflated_bytes"`
	InflatedLengthDelta   int                    `json:"inflated_length_delta"`
	FromCivID             int                    `json:"from_civ_id"`
	FromUnitID            int                    `json:"from_unit_id"`
	NewUnitID             int                    `json:"new_unit_id"`
	CivIDs                []int                  `json:"civ_ids"`
	CreatedCount          int                    `json:"created_count"`
	ChangedFields         []string               `json:"changed_fields"`
	Verified              bool                   `json:"verified"`
	Verification          aoe2.VerificationClaim `json:"verification"`
}

func Open(path string) (*Index, error) {
	compressed, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	payload, err := Inflate(compressed)
	if err != nil {
		return nil, fmt.Errorf("inflate %s: %w", path, err)
	}
	idx, err := Parse(payload)
	if err != nil {
		return nil, err
	}
	idx.Path = path
	idx.Compressed = len(compressed)
	return idx, nil
}

func PatchGraphicFile(inputPath, outputPath string, graphicID int, patch GraphicPatch) (PatchReport, error) {
	inputAbs, err := filepath.Abs(inputPath)
	if err != nil {
		return PatchReport{}, err
	}
	outputAbs, err := filepath.Abs(outputPath)
	if err != nil {
		return PatchReport{}, err
	}
	if inputAbs == outputAbs {
		return PatchReport{}, errors.New("refusing in-place dat patch; choose a separate output path")
	}
	compressed, err := os.ReadFile(inputPath)
	if err != nil {
		return PatchReport{}, err
	}
	output, report, err := PatchGraphic(compressed, graphicID, patch)
	if err != nil {
		return PatchReport{}, err
	}
	if err := os.WriteFile(outputPath, output, 0644); err != nil {
		return PatchReport{}, err
	}
	return report, nil
}

func PatchGraphic(compressed []byte, graphicID int, patch GraphicPatch) ([]byte, PatchReport, error) {
	if patch.Empty() {
		return nil, PatchReport{}, errors.New("no graphic patch fields provided")
	}
	payload, err := Inflate(compressed)
	if err != nil {
		return nil, PatchReport{}, err
	}
	beforeIdx, err := Parse(payload)
	if err != nil {
		return nil, PatchReport{}, fmt.Errorf("index input: %w", err)
	}
	before, ok := beforeIdx.Graphic(graphicID)
	if !ok {
		return nil, PatchReport{}, fmt.Errorf("graphic %d is absent or outside graphics table", graphicID)
	}
	originalBefore := before

	patchedPayload := append([]byte(nil), payload...)
	changed := []string{}
	if patch.needsGraphicRecordRebuild() {
		var err error
		patchedPayload, err = spliceGraphicRecord(patchedPayload, before, patch)
		if err != nil {
			return nil, PatchReport{}, err
		}
		if patch.Name != nil {
			changed = append(changed, "name")
		}
		if patch.FileName != nil {
			changed = append(changed, "file_name")
		}
		if patch.ParticleEffectName != nil {
			changed = append(changed, "particle_effect_name")
		}
		if patch.SetDeltas != nil {
			changed = append(changed, "set_deltas")
		}
		if patch.SetAngleSounds != nil {
			changed = append(changed, "set_angle_sounds")
		}
		splicedIdx, err := Parse(patchedPayload)
		if err != nil {
			return nil, PatchReport{}, fmt.Errorf("re-index spliced graphic record: %w", err)
		}
		var ok bool
		before, ok = splicedIdx.Graphic(graphicID)
		if !ok {
			return nil, PatchReport{}, fmt.Errorf("spliced output lost graphic %d", graphicID)
		}
	}
	names, err := applyGraphicScalarPatch(patchedPayload, before, patch)
	if err != nil {
		return nil, PatchReport{}, err
	}
	changed = append(changed, names...)

	output, err := Deflate(patchedPayload)
	if err != nil {
		return nil, PatchReport{}, err
	}
	roundTripPayload, err := Inflate(output)
	if err != nil {
		return nil, PatchReport{}, fmt.Errorf("inflate patched output: %w", err)
	}
	if !bytes.Equal(patchedPayload, roundTripPayload) {
		return nil, PatchReport{}, errors.New("patched payload changed after deflate/inflate round trip")
	}
	afterIdx, err := Parse(roundTripPayload)
	if err != nil {
		return nil, PatchReport{}, fmt.Errorf("re-index patched output: %w", err)
	}
	after, ok := afterIdx.Graphic(graphicID)
	if !ok {
		return nil, PatchReport{}, fmt.Errorf("patched output lost graphic %d", graphicID)
	}
	if beforeIdx.GraphicsSize != afterIdx.GraphicsSize {
		return nil, PatchReport{}, fmt.Errorf("graphics count changed: %d -> %d", beforeIdx.GraphicsSize, afterIdx.GraphicsSize)
	}
	expectedLengthDelta := expectedGraphicPatchLengthDelta(originalBefore, patch)
	actualLengthDelta := len(roundTripPayload) - len(payload)
	if actualLengthDelta != expectedLengthDelta {
		return nil, PatchReport{}, fmt.Errorf("inflated length delta mismatch: got %d want %d", actualLengthDelta, expectedLengthDelta)
	}
	if patch.FileName != nil && after.FileName.Value != *patch.FileName {
		return nil, PatchReport{}, fmt.Errorf("file_name patch did not read back: got %q want %q", after.FileName.Value, *patch.FileName)
	}
	if patch.Name != nil && after.Name.Value != *patch.Name {
		return nil, PatchReport{}, fmt.Errorf("name patch did not read back: got %q want %q", after.Name.Value, *patch.Name)
	}
	if patch.ParticleEffectName != nil && after.ParticleEffectName.Value != *patch.ParticleEffectName {
		return nil, PatchReport{}, fmt.Errorf("particle_effect_name patch did not read back: got %q want %q", after.ParticleEffectName.Value, *patch.ParticleEffectName)
	}
	if err := verifyGraphicPatchReadback(after, patch); err != nil {
		return nil, PatchReport{}, err
	}
	if err := verifyNeighborCanaries(beforeIdx, afterIdx, graphicID); err != nil {
		return nil, PatchReport{}, err
	}
	report := PatchReport{
		InputCompressedBytes:  len(compressed),
		OutputCompressedBytes: len(output),
		InputInflatedBytes:    len(payload),
		OutputInflatedBytes:   len(roundTripPayload),
		InflatedLengthDelta:   len(roundTripPayload) - len(payload),
		GraphicID:             graphicID,
		Before:                originalBefore.Summary(),
		After:                 after.Summary(),
		ChangedFields:         changed,
		Verified:              true,
		Verification:          aoe2.StructureVerification(true),
	}
	return output, report, nil
}

func (patch GraphicPatch) needsGraphicRecordRebuild() bool {
	return patch.Name != nil ||
		patch.FileName != nil ||
		patch.ParticleEffectName != nil ||
		patch.SetDeltas != nil ||
		patch.SetAngleSounds != nil
}

func (patch GraphicPatch) Empty() bool {
	return patch.Name == nil &&
		patch.FileName == nil &&
		patch.ParticleEffectName == nil &&
		patch.SLP == nil &&
		patch.IsLoaded == nil &&
		patch.OldColorFlag == nil &&
		patch.Layer == nil &&
		patch.PlayerColor == nil &&
		patch.Rainbow == nil &&
		patch.TransparentSelect == nil &&
		len(patch.Coordinates) == 0 &&
		patch.SoundID == nil &&
		patch.WwiseSoundID == nil &&
		patch.FrameCount == nil &&
		patch.SpeedMultiplier == nil &&
		patch.FrameDuration == nil &&
		patch.ReplayDelay == nil &&
		patch.SequenceType == nil &&
		patch.MirroringMode == nil &&
		patch.EditorFlag == nil &&
		patch.SetDeltas == nil &&
		patch.SetAngleSounds == nil
}

func applyGraphicScalarPatch(payload []byte, graphic Graphic, patch GraphicPatch) ([]string, error) {
	changed := []string{}
	if patch.SLP != nil {
		putI32(payload, graphic.FieldSpans.SLP, *patch.SLP)
		changed = append(changed, "slp")
	}
	if patch.IsLoaded != nil {
		putU8(payload, graphic.FieldSpans.IsLoaded, uint8(*patch.IsLoaded))
		changed = append(changed, "is_loaded")
	}
	if patch.OldColorFlag != nil {
		putU8(payload, graphic.FieldSpans.OldColorFlag, uint8(*patch.OldColorFlag))
		changed = append(changed, "old_color_flag")
	}
	if patch.Layer != nil {
		putU8(payload, graphic.FieldSpans.Layer, uint8(*patch.Layer))
		changed = append(changed, "layer")
	}
	if patch.PlayerColor != nil {
		putU8(payload, graphic.FieldSpans.PlayerColor, uint8(*patch.PlayerColor))
		changed = append(changed, "player_color")
	}
	if patch.Rainbow != nil {
		putU8(payload, graphic.FieldSpans.Rainbow, uint8(*patch.Rainbow))
		changed = append(changed, "rainbow")
	}
	if patch.TransparentSelect != nil {
		putU8(payload, graphic.FieldSpans.TransparentSelect, uint8(*patch.TransparentSelect))
		changed = append(changed, "transparent_selection")
	}
	if len(patch.Coordinates) > 0 {
		if len(patch.Coordinates) != 4 {
			return changed, fmt.Errorf("coordinates length=%d, want 4", len(patch.Coordinates))
		}
		for i, value := range patch.Coordinates {
			putI16(payload, graphic.FieldSpans.Coordinates[i], value)
		}
		changed = append(changed, "coordinates")
	}
	if patch.SoundID != nil {
		putI16(payload, graphic.FieldSpans.SoundID, *patch.SoundID)
		changed = append(changed, "sound_id")
	}
	if patch.WwiseSoundID != nil {
		putU32(payload, graphic.FieldSpans.WwiseSoundID, *patch.WwiseSoundID)
		changed = append(changed, "wwise_sound_id")
	}
	if patch.FrameCount != nil {
		putI16(payload, graphic.FieldSpans.FrameCount, *patch.FrameCount)
		changed = append(changed, "frame_count")
	}
	if patch.SpeedMultiplier != nil {
		putF32(payload, graphic.FieldSpans.SpeedMultiplier, *patch.SpeedMultiplier)
		changed = append(changed, "speed_multiplier")
	}
	if patch.FrameDuration != nil {
		putF32(payload, graphic.FieldSpans.FrameDuration, *patch.FrameDuration)
		changed = append(changed, "frame_duration")
	}
	if patch.ReplayDelay != nil {
		putF32(payload, graphic.FieldSpans.ReplayDelay, *patch.ReplayDelay)
		changed = append(changed, "replay_delay")
	}
	if patch.SequenceType != nil {
		putU8(payload, graphic.FieldSpans.SequenceType, *patch.SequenceType)
		changed = append(changed, "sequence_type")
	}
	if patch.MirroringMode != nil {
		putU8(payload, graphic.FieldSpans.MirroringMode, uint8(*patch.MirroringMode))
		changed = append(changed, "mirroring_mode")
	}
	if patch.EditorFlag != nil {
		putU8(payload, graphic.FieldSpans.EditorFlag, uint8(*patch.EditorFlag))
		changed = append(changed, "editor_flag")
	}
	return changed, nil
}

func verifyGraphicPatchReadback(after Graphic, patch GraphicPatch) error {
	if patch.SLP != nil && after.SLP != *patch.SLP {
		return fmt.Errorf("slp patch did not read back: got %d want %d", after.SLP, *patch.SLP)
	}
	if patch.IsLoaded != nil && after.IsLoaded != *patch.IsLoaded {
		return fmt.Errorf("is_loaded patch did not read back: got %d want %d", after.IsLoaded, *patch.IsLoaded)
	}
	if patch.OldColorFlag != nil && after.OldColorFlag != *patch.OldColorFlag {
		return fmt.Errorf("old_color_flag patch did not read back: got %d want %d", after.OldColorFlag, *patch.OldColorFlag)
	}
	if patch.Layer != nil && after.Layer != *patch.Layer {
		return fmt.Errorf("layer patch did not read back: got %d want %d", after.Layer, *patch.Layer)
	}
	if patch.PlayerColor != nil && after.PlayerColor != *patch.PlayerColor {
		return fmt.Errorf("player_color patch did not read back: got %d want %d", after.PlayerColor, *patch.PlayerColor)
	}
	if patch.Rainbow != nil && after.Rainbow != *patch.Rainbow {
		return fmt.Errorf("rainbow patch did not read back: got %d want %d", after.Rainbow, *patch.Rainbow)
	}
	if patch.TransparentSelect != nil && after.TransparentSelect != *patch.TransparentSelect {
		return fmt.Errorf("transparent_selection patch did not read back: got %d want %d", after.TransparentSelect, *patch.TransparentSelect)
	}
	if len(patch.Coordinates) > 0 {
		if len(patch.Coordinates) != 4 {
			return fmt.Errorf("coordinates length=%d, want 4", len(patch.Coordinates))
		}
		for i, want := range patch.Coordinates {
			if after.Coordinates[i] != want {
				return fmt.Errorf("coordinates[%d] patch did not read back: got %d want %d", i, after.Coordinates[i], want)
			}
		}
	}
	if patch.SoundID != nil && after.SoundID != *patch.SoundID {
		return fmt.Errorf("sound_id patch did not read back: got %d want %d", after.SoundID, *patch.SoundID)
	}
	if patch.WwiseSoundID != nil && after.WwiseSoundID != *patch.WwiseSoundID {
		return fmt.Errorf("wwise_sound_id patch did not read back: got %d want %d", after.WwiseSoundID, *patch.WwiseSoundID)
	}
	if patch.FrameCount != nil && after.FrameCount != *patch.FrameCount {
		return fmt.Errorf("frame_count patch did not read back: got %d want %d", after.FrameCount, *patch.FrameCount)
	}
	if patch.SpeedMultiplier != nil && after.SpeedMultiplier != *patch.SpeedMultiplier {
		return fmt.Errorf("speed_multiplier patch did not read back: got %g want %g", after.SpeedMultiplier, *patch.SpeedMultiplier)
	}
	if patch.FrameDuration != nil && after.FrameDuration != *patch.FrameDuration {
		return fmt.Errorf("frame_duration patch did not read back: got %g want %g", after.FrameDuration, *patch.FrameDuration)
	}
	if patch.ReplayDelay != nil && after.ReplayDelay != *patch.ReplayDelay {
		return fmt.Errorf("replay_delay patch did not read back: got %g want %g", after.ReplayDelay, *patch.ReplayDelay)
	}
	if patch.SequenceType != nil && after.SequenceType != *patch.SequenceType {
		return fmt.Errorf("sequence_type patch did not read back: got %d want %d", after.SequenceType, *patch.SequenceType)
	}
	if patch.MirroringMode != nil && after.MirroringMode != *patch.MirroringMode {
		return fmt.Errorf("mirroring_mode patch did not read back: got %d want %d", after.MirroringMode, *patch.MirroringMode)
	}
	if patch.EditorFlag != nil && after.EditorFlag != *patch.EditorFlag {
		return fmt.Errorf("editor_flag patch did not read back: got %d want %d", after.EditorFlag, *patch.EditorFlag)
	}
	if patch.SetDeltas != nil {
		if int(after.DeltaCount) != len(*patch.SetDeltas) || len(after.Deltas) != len(*patch.SetDeltas) {
			return fmt.Errorf("set_deltas count did not read back: delta_count=%d rows=%d want %d", after.DeltaCount, len(after.Deltas), len(*patch.SetDeltas))
		}
		for i, want := range *patch.SetDeltas {
			if got := graphicDeltaRow(after.Deltas[i]); got != want {
				return fmt.Errorf("set_deltas[%d] did not read back: got %+v want %+v", i, got, want)
			}
		}
	}
	if patch.SetAngleSounds != nil {
		if len(*patch.SetAngleSounds) == 0 {
			if after.AngleSoundsUsed != 0 || len(after.AngleSounds) != 0 {
				return fmt.Errorf("set_angle_sounds empty did not disable rows: used=%d rows=%d", after.AngleSoundsUsed, len(after.AngleSounds))
			}
		} else {
			if after.AngleSoundsUsed == 0 || len(after.AngleSounds) != len(*patch.SetAngleSounds) {
				return fmt.Errorf("set_angle_sounds count did not read back: used=%d rows=%d want %d", after.AngleSoundsUsed, len(after.AngleSounds), len(*patch.SetAngleSounds))
			}
			for i, want := range *patch.SetAngleSounds {
				if got := graphicAngleSoundRow(after.AngleSounds[i]); got != want {
					return fmt.Errorf("set_angle_sounds[%d] did not read back: got %+v want %+v", i, got, want)
				}
			}
		}
	}
	return nil
}

func graphicDeltaRow(delta GraphicDelta) GraphicDeltaRow {
	return GraphicDeltaRow{
		GraphicID:    delta.GraphicID,
		Padding1:     delta.Padding1,
		SpritePtr:    delta.SpritePtr,
		OffsetX:      delta.OffsetX,
		OffsetY:      delta.OffsetY,
		DisplayAngle: delta.DisplayAngle,
		Padding2:     delta.Padding2,
	}
}

func graphicAngleSoundRow(sound GraphicAngleSound) GraphicAngleSoundRow {
	return GraphicAngleSoundRow{
		FrameNum:      sound.FrameNum,
		SoundID:       sound.SoundID,
		WwiseSoundID:  sound.WwiseSoundID,
		FrameNum2:     sound.FrameNum2,
		WwiseSoundID2: sound.WwiseSoundID2,
		SoundID2:      sound.SoundID2,
		FrameNum3:     sound.FrameNum3,
		WwiseSoundID3: sound.WwiseSoundID3,
		SoundID3:      sound.SoundID3,
	}
}

func PatchUnitFile(inputPath, outputPath string, civID, unitID int, patch UnitPatch) (UnitPatchReport, error) {
	inputAbs, err := filepath.Abs(inputPath)
	if err != nil {
		return UnitPatchReport{}, err
	}
	outputAbs, err := filepath.Abs(outputPath)
	if err != nil {
		return UnitPatchReport{}, err
	}
	if inputAbs == outputAbs {
		return UnitPatchReport{}, errors.New("refusing in-place dat patch; choose a separate output path")
	}
	compressed, err := os.ReadFile(inputPath)
	if err != nil {
		return UnitPatchReport{}, err
	}
	output, report, err := PatchUnit(compressed, civID, unitID, patch)
	if err != nil {
		return UnitPatchReport{}, err
	}
	if err := os.WriteFile(outputPath, output, 0644); err != nil {
		return UnitPatchReport{}, err
	}
	return report, nil
}

func PatchUnit(compressed []byte, civID, unitID int, patch UnitPatch) ([]byte, UnitPatchReport, error) {
	if patch.Empty() {
		return nil, UnitPatchReport{}, errors.New("no unit patch fields provided")
	}
	payload, err := Inflate(compressed)
	if err != nil {
		return nil, UnitPatchReport{}, err
	}
	beforeIdx, err := Parse(payload)
	if err != nil {
		return nil, UnitPatchReport{}, fmt.Errorf("index input: %w", err)
	}
	before, err := unitByID(beforeIdx, civID, unitID)
	if err != nil {
		return nil, UnitPatchReport{}, err
	}
	patchedPayload := append([]byte(nil), payload...)
	var changed []string
	if patch.needsUnitRecordRebuild() {
		record, names, err := rebuildUnitRecord(payload, before, patch)
		if err != nil {
			return nil, UnitPatchReport{}, err
		}
		patchedPayload = spliceUnitRecord(payload, before, record)
		changed = names
	} else {
		var err error
		changed, err = applyUnitPatch(patchedPayload, before, patch)
		if err != nil {
			return nil, UnitPatchReport{}, err
		}
	}
	output, err := Deflate(patchedPayload)
	if err != nil {
		return nil, UnitPatchReport{}, err
	}
	roundTripPayload, err := Inflate(output)
	if err != nil {
		return nil, UnitPatchReport{}, fmt.Errorf("inflate patched output: %w", err)
	}
	if !bytes.Equal(patchedPayload, roundTripPayload) {
		return nil, UnitPatchReport{}, errors.New("patched payload changed after deflate/inflate round trip")
	}
	afterIdx, err := Parse(roundTripPayload)
	if err != nil {
		return nil, UnitPatchReport{}, fmt.Errorf("re-index patched output: %w", err)
	}
	after, err := unitByID(afterIdx, civID, unitID)
	if err != nil {
		return nil, UnitPatchReport{}, fmt.Errorf("patched output: %w", err)
	}
	if !patch.needsUnitRecordRebuild() && len(roundTripPayload) != len(payload) {
		return nil, UnitPatchReport{}, fmt.Errorf("unit fixed-width patch changed inflated length: %d -> %d", len(payload), len(roundTripPayload))
	}
	if len(beforeIdx.Civs) != len(afterIdx.Civs) {
		return nil, UnitPatchReport{}, fmt.Errorf("civ count changed: %d -> %d", len(beforeIdx.Civs), len(afterIdx.Civs))
	}
	if beforeIdx.Civs[civID].UnitsSize != afterIdx.Civs[civID].UnitsSize {
		return nil, UnitPatchReport{}, fmt.Errorf("unit count changed for civ %d: %d -> %d", civID, beforeIdx.Civs[civID].UnitsSize, afterIdx.Civs[civID].UnitsSize)
	}
	if err := verifyUnitPatchReadback(after, patch); err != nil {
		return nil, UnitPatchReport{}, err
	}
	if err := verifyUnitNeighborCanaries(beforeIdx, afterIdx, civID, unitID, patch.needsUnitRecordRebuild()); err != nil {
		return nil, UnitPatchReport{}, err
	}
	report := UnitPatchReport{
		InputCompressedBytes:  len(compressed),
		OutputCompressedBytes: len(output),
		InputInflatedBytes:    len(payload),
		OutputInflatedBytes:   len(roundTripPayload),
		InflatedLengthDelta:   len(roundTripPayload) - len(payload),
		CivID:                 civID,
		UnitID:                unitID,
		Before:                before,
		After:                 after,
		ChangedFields:         changed,
		Verified:              true,
		Verification:          aoe2.StructureVerification(true),
	}
	return output, report, nil
}

func (patch UnitPatch) needsUnitRecordRebuild() bool {
	return patch.SetDamageGraphics != nil ||
		patch.SetType50Attacks != nil ||
		patch.SetType50Armours != nil ||
		patch.SetTrainLocations != nil ||
		patch.SetDropSites != nil ||
		patch.SetTasks != nil
}

func rebuildUnitRecord(payload []byte, unit UnitSummary, patch UnitPatch) ([]byte, []string, error) {
	if unit.RecordStart < 0 || unit.RecordEnd < unit.RecordStart || unit.RecordEnd > len(payload) {
		return nil, nil, fmt.Errorf("unit span invalid: %d..%d", unit.RecordStart, unit.RecordEnd)
	}
	return rebuildUnitRecordData(payload[unit.RecordStart:unit.RecordEnd], unit.CivIndex, unit.Index, patch)
}

// RebuildUnitRecord applies a UnitPatch to one parsed unit record and returns the rebuilt record bytes.
func RebuildUnitRecord(payload []byte, unit UnitSummary, patch UnitPatch) ([]byte, []string, error) {
	return rebuildUnitRecord(payload, unit, patch)
}

func rebuildUnitRecordData(record []byte, civIndex, unitIndex int, patch UnitPatch) ([]byte, []string, error) {
	if patch.SetDamageGraphics != nil && len(patch.DamageGraphics) > 0 {
		return nil, nil, errors.New("cannot combine set_damage_graphics with indexed damage_graphics patches")
	}
	if patch.SetType50Attacks != nil && len(patch.Type50Attacks) > 0 {
		return nil, nil, errors.New("cannot combine set_type50_attacks with indexed type50_attacks patches")
	}
	if patch.SetType50Armours != nil && len(patch.Type50Armours) > 0 {
		return nil, nil, errors.New("cannot combine set_type50_armours with indexed type50_armours patches")
	}
	if patch.SetTrainLocations != nil && (len(patch.TrainLocations) > 0 || patch.TrainTime0 != nil || patch.TrainUnitID0 != nil || patch.TrainButtonID0 != nil || patch.TrainHotKeyID0 != nil) {
		return nil, nil, errors.New("cannot combine set_train_locations with indexed train-location or train-location-0 patches")
	}
	if patch.SetTasks != nil && len(patch.Tasks) > 0 {
		return nil, nil, errors.New("cannot combine set_tasks with indexed task patches")
	}
	data := append([]byte(nil), record...)
	parsed, err := parseUnit(&cursor{data: data}, civIndex, unitIndex, true)
	if err != nil {
		return nil, nil, fmt.Errorf("reparse unit record: %w", err)
	}
	changed, err := applyUnitPatch(data, parsed, patch)
	if err != nil {
		return nil, nil, err
	}
	replacements := []recordReplacement{}
	if patch.SetDamageGraphics != nil {
		if len(*patch.SetDamageGraphics) > math.MaxUint8 {
			return nil, nil, fmt.Errorf("set_damage_graphics length=%d exceeds uint8 limit", len(*patch.SetDamageGraphics))
		}
		replacements = append(replacements, recordReplacement{
			Start: parsed.ListSpans.DamageGraphicCount.Start,
			End:   parsed.ListSpans.DamageGraphics.End,
			Data:  encodeDamageGraphicList(*patch.SetDamageGraphics),
		})
		changed = append(changed, "set_damage_graphics")
	}
	if patch.SetType50Attacks != nil || patch.SetType50Armours != nil {
		type50, err := requireType50(parsed)
		if err != nil {
			return nil, nil, err
		}
		if patch.SetType50Attacks != nil {
			if len(*patch.SetType50Attacks) > math.MaxInt16 {
				return nil, nil, fmt.Errorf("set_type50_attacks length=%d exceeds int16 limit", len(*patch.SetType50Attacks))
			}
			replacements = append(replacements, recordReplacement{
				Start: type50.FieldSpans.AttackCount.Start,
				End:   type50.FieldSpans.Attacks.End,
				Data:  encodeWeaponInfoList(*patch.SetType50Attacks),
			})
			changed = append(changed, "set_type50_attacks")
		}
		if patch.SetType50Armours != nil {
			if len(*patch.SetType50Armours) > math.MaxInt16 {
				return nil, nil, fmt.Errorf("set_type50_armours length=%d exceeds int16 limit", len(*patch.SetType50Armours))
			}
			replacements = append(replacements, recordReplacement{
				Start: type50.FieldSpans.ArmourCount.Start,
				End:   type50.FieldSpans.Armours.End,
				Data:  encodeWeaponInfoList(*patch.SetType50Armours),
			})
			changed = append(changed, "set_type50_armours")
		}
	}
	if patch.SetTrainLocations != nil {
		creatable, err := requireCreatable(parsed)
		if err != nil {
			return nil, nil, err
		}
		if len(*patch.SetTrainLocations) > math.MaxInt16 {
			return nil, nil, fmt.Errorf("set_train_locations length=%d exceeds int16 limit", len(*patch.SetTrainLocations))
		}
		replacements = append(replacements, recordReplacement{
			Start: creatable.FieldSpans.TrainLocationCount.Start,
			End:   creatable.FieldSpans.TrainLocations.End,
			Data:  encodeTrainLocationList(*patch.SetTrainLocations),
		})
		changed = append(changed, "set_train_locations")
	}
	if patch.SetDropSites != nil {
		action, err := requireAction(parsed)
		if err != nil {
			return nil, nil, err
		}
		if len(*patch.SetDropSites) > math.MaxInt16 {
			return nil, nil, fmt.Errorf("set_drop_sites length=%d exceeds int16 limit", len(*patch.SetDropSites))
		}
		replacements = append(replacements, recordReplacement{
			Start: action.FieldSpans.DropSiteCount.Start,
			End:   action.FieldSpans.DropSites.End,
			Data:  encodeI16List(*patch.SetDropSites),
		})
		changed = append(changed, "set_drop_sites")
	}
	if patch.SetTasks != nil {
		action, err := requireAction(parsed)
		if err != nil {
			return nil, nil, err
		}
		if len(*patch.SetTasks) > math.MaxInt16 {
			return nil, nil, fmt.Errorf("set_tasks length=%d exceeds int16 limit", len(*patch.SetTasks))
		}
		encoded, err := encodeTaskList(*patch.SetTasks, true)
		if err != nil {
			return nil, nil, err
		}
		replacements = append(replacements, recordReplacement{
			Start: action.FieldSpans.TaskCount.Start,
			End:   action.FieldSpans.Tasks.End,
			Data:  encoded,
		})
		changed = append(changed, "set_tasks")
	}
	data, err = applyRecordReplacements(data, replacements)
	if err != nil {
		return nil, nil, err
	}
	if _, err := parseUnit(&cursor{data: data}, civIndex, unitIndex, true); err != nil {
		return nil, nil, fmt.Errorf("reparse rebuilt unit record: %w", err)
	}
	return data, changed, nil
}

func encodeDamageGraphicList(rows []DamageGraphicRow) []byte {
	out := make([]byte, 1+len(rows)*5)
	out[0] = uint8(len(rows))
	off := 1
	for _, row := range rows {
		binary.LittleEndian.PutUint16(out[off:off+2], row.GraphicID)
		binary.LittleEndian.PutUint16(out[off+2:off+4], row.DamagePercent)
		out[off+4] = row.Flag
		off += 5
	}
	return out
}

func encodeWeaponInfoList(rows []WeaponInfoRow) []byte {
	out := make([]byte, 2+len(rows)*4)
	binary.LittleEndian.PutUint16(out[0:2], uint16(int16(len(rows))))
	off := 2
	for _, row := range rows {
		binary.LittleEndian.PutUint16(out[off:off+2], uint16(row.Class))
		binary.LittleEndian.PutUint16(out[off+2:off+4], uint16(row.Value))
		off += 4
	}
	return out
}

func encodeTrainLocationList(rows []TrainLocationRow) []byte {
	out := make([]byte, 2+len(rows)*9)
	binary.LittleEndian.PutUint16(out[0:2], uint16(int16(len(rows))))
	off := 2
	for _, row := range rows {
		binary.LittleEndian.PutUint16(out[off:off+2], uint16(row.TrainTime))
		binary.LittleEndian.PutUint16(out[off+2:off+4], uint16(row.TrainUnitID))
		out[off+4] = row.TrainButton
		binary.LittleEndian.PutUint32(out[off+5:off+9], uint32(row.TrainHotkey))
		off += 9
	}
	return out
}

func encodeI16List(rows []int16) []byte {
	out := make([]byte, 2, 2+len(rows)*2)
	binary.LittleEndian.PutUint16(out[0:2], uint16(int16(len(rows))))
	for _, row := range rows {
		out = binary.LittleEndian.AppendUint16(out, uint16(row))
	}
	return out
}

func EncodeCurrentDETaskList(rows []TaskRow) ([]byte, error) {
	return encodeTaskList(rows, true)
}

func encodeTaskList(rows []TaskRow, versionAtLeast88 bool) ([]byte, error) {
	taskLen := taskRecordLen(versionAtLeast88)
	rawTailLen := taskLen - 40
	if rawTailLen < 0 {
		return nil, fmt.Errorf("task record length %d is shorter than decoded prefix", taskLen)
	}
	out := make([]byte, 2, 2+len(rows)*taskLen)
	binary.LittleEndian.PutUint16(out[0:2], uint16(int16(len(rows))))
	for i, row := range rows {
		if len(row.AttributeTypes) != 4 {
			return nil, fmt.Errorf("set_tasks[%d] attribute_types length=%d, want 4", i, len(row.AttributeTypes))
		}
		if len(row.RawTail) != rawTailLen {
			return nil, fmt.Errorf("set_tasks[%d] raw_tail length=%d, want %d", i, len(row.RawTail), rawTailLen)
		}
		record := make([]byte, 0, taskLen)
		record = binary.LittleEndian.AppendUint16(record, uint16(row.RecordType))
		record = binary.LittleEndian.AppendUint16(record, uint16(row.ID))
		record = append(record, row.IsDefault)
		record = binary.LittleEndian.AppendUint16(record, uint16(row.ActionType))
		record = binary.LittleEndian.AppendUint16(record, uint16(row.ObjectClass))
		record = binary.LittleEndian.AppendUint16(record, uint16(row.ObjectID))
		record = binary.LittleEndian.AppendUint16(record, uint16(row.TerrainID))
		for _, value := range row.AttributeTypes {
			record = binary.LittleEndian.AppendUint16(record, uint16(value))
		}
		record = binary.LittleEndian.AppendUint32(record, math.Float32bits(row.WorkValue1))
		record = binary.LittleEndian.AppendUint32(record, math.Float32bits(row.WorkValue2))
		record = binary.LittleEndian.AppendUint32(record, math.Float32bits(row.WorkRange))
		record = append(record, row.AutoSearchTargets)
		record = binary.LittleEndian.AppendUint32(record, math.Float32bits(row.SearchWaitTime))
		record = append(record, row.EnableTargeting)
		record = append(record, row.CombatLevel)
		record = append(record, row.RawTail...)
		if len(record) != taskLen {
			return nil, fmt.Errorf("set_tasks[%d] encoded length=%d, want %d", i, len(record), taskLen)
		}
		out = append(out, record...)
	}
	return out, nil
}

func spliceUnitRecord(payload []byte, unit UnitSummary, record []byte) []byte {
	out := make([]byte, 0, len(payload)-unit.RecordLength+len(record))
	out = append(out, payload[:unit.RecordStart]...)
	out = append(out, record...)
	out = append(out, payload[unit.RecordEnd:]...)
	return out
}

type recordReplacement struct {
	Start int
	End   int
	Data  []byte
}

func applyRecordReplacements(record []byte, replacements []recordReplacement) ([]byte, error) {
	sort.Slice(replacements, func(i, j int) bool {
		return replacements[i].Start > replacements[j].Start
	})
	out := append([]byte(nil), record...)
	for i, replacement := range replacements {
		if replacement.Start < 0 || replacement.End < replacement.Start || replacement.End > len(out) {
			return nil, fmt.Errorf("record replacement %d has invalid span %d..%d for record length %d", i, replacement.Start, replacement.End, len(out))
		}
		next := make([]byte, 0, len(out)-replacement.End+replacement.Start+len(replacement.Data))
		next = append(next, out[:replacement.Start]...)
		next = append(next, replacement.Data...)
		next = append(next, out[replacement.End:]...)
		out = next
	}
	return out, nil
}

func PatchUnitGraphicSlotAllCivs(compressed []byte, unitID int, slot string, graphicID int16) ([]byte, []UnitPatchReport, error) {
	if unitID < 0 {
		return nil, nil, errors.New("unit id must be non-negative")
	}
	payload, err := Inflate(compressed)
	if err != nil {
		return nil, nil, err
	}
	beforeIdx, err := Parse(payload)
	if err != nil {
		return nil, nil, fmt.Errorf("index input: %w", err)
	}
	patchedPayload := append([]byte(nil), payload...)
	type pending struct {
		civID  int
		before UnitSummary
		field  string
	}
	var pendingReports []pending
	for _, civ := range beforeIdx.Civs {
		if unitID >= len(civ.Units) || !civ.Units[unitID].Present {
			continue
		}
		unit := civ.Units[unitID]
		span, field, err := unitGraphicSlotSpan(unit, slot)
		if err != nil {
			return nil, nil, fmt.Errorf("unit %d civ %d slot %s: %w", unitID, civ.Index, slot, err)
		}
		putI16(patchedPayload, span, graphicID)
		pendingReports = append(pendingReports, pending{civID: civ.Index, before: unit, field: field})
	}
	if len(pendingReports) == 0 {
		return nil, nil, fmt.Errorf("unit %d was not present in any civ table", unitID)
	}
	output, err := Deflate(patchedPayload)
	if err != nil {
		return nil, nil, err
	}
	roundTripPayload, err := Inflate(output)
	if err != nil {
		return nil, nil, fmt.Errorf("inflate patched output: %w", err)
	}
	if !bytes.Equal(patchedPayload, roundTripPayload) {
		return nil, nil, errors.New("patched payload changed after deflate/inflate round trip")
	}
	afterIdx, err := Parse(roundTripPayload)
	if err != nil {
		return nil, nil, fmt.Errorf("re-index patched output: %w", err)
	}
	if beforeIdx.GraphicsSize != afterIdx.GraphicsSize {
		return nil, nil, fmt.Errorf("graphics count changed: %d -> %d", beforeIdx.GraphicsSize, afterIdx.GraphicsSize)
	}
	var reports []UnitPatchReport
	for _, item := range pendingReports {
		after, err := unitByID(afterIdx, item.civID, unitID)
		if err != nil {
			return nil, nil, err
		}
		got, err := unitGraphicSlotValue(after, slot)
		if err != nil {
			return nil, nil, err
		}
		if got != graphicID {
			return nil, nil, fmt.Errorf("unit %d civ %d slot %s=%d want %d", unitID, item.civID, slot, got, graphicID)
		}
		reports = append(reports, UnitPatchReport{
			InputCompressedBytes:  len(compressed),
			OutputCompressedBytes: len(output),
			InputInflatedBytes:    len(payload),
			OutputInflatedBytes:   len(roundTripPayload),
			InflatedLengthDelta:   len(roundTripPayload) - len(payload),
			CivID:                 item.civID,
			UnitID:                unitID,
			Before:                item.before,
			After:                 after,
			ChangedFields:         []string{item.field},
			Verified:              true,
			Verification:          aoe2.StructureVerification(true),
		})
	}
	return output, reports, nil
}

func unitGraphicSlotSpan(unit UnitSummary, slot string) (Span, string, error) {
	switch slot {
	case "flying", "standing":
		return unit.FieldSpans.StandingGraphic1, "standing_graphic_1", nil
	case "standing2":
		return unit.FieldSpans.StandingGraphic2, "standing_graphic_2", nil
	case "dying":
		return unit.FieldSpans.DyingGraphic, "dying_graphic", nil
	case "attack":
		type50, err := requireType50(unit)
		if err != nil {
			return Span{}, "", err
		}
		return type50.FieldSpans.AttackGraphic, "type50_attack_graphic", nil
	default:
		return Span{}, "", fmt.Errorf("unknown slot %q", slot)
	}
}

func unitGraphicSlotValue(unit UnitSummary, slot string) (int16, error) {
	switch slot {
	case "flying", "standing":
		return unit.StandingGraphic1, nil
	case "standing2":
		return unit.StandingGraphic2, nil
	case "dying":
		return unit.DyingGraphic, nil
	case "attack":
		type50, err := requireType50(unit)
		if err != nil {
			return 0, err
		}
		return type50.AttackGraphic, nil
	default:
		return 0, fmt.Errorf("unknown slot %q", slot)
	}
}

func PatchRecipeFile(inputPath, outputPath string, recipe Recipe) (RecipeReport, error) {
	inputAbs, err := filepath.Abs(inputPath)
	if err != nil {
		return RecipeReport{}, err
	}
	outputAbs, err := filepath.Abs(outputPath)
	if err != nil {
		return RecipeReport{}, err
	}
	if inputAbs == outputAbs {
		return RecipeReport{}, errors.New("refusing in-place dat patch; choose a separate output path")
	}
	compressed, err := os.ReadFile(inputPath)
	if err != nil {
		return RecipeReport{}, err
	}
	output, report, err := PatchRecipe(compressed, recipe)
	if err != nil {
		return RecipeReport{}, err
	}
	if err := os.WriteFile(outputPath, output, 0644); err != nil {
		return RecipeReport{}, err
	}
	return report, nil
}

func PlanRecipeFile(inputPath string, recipe Recipe) (RecipeReport, error) {
	compressed, err := os.ReadFile(inputPath)
	if err != nil {
		return RecipeReport{}, err
	}
	_, report, err := PatchRecipe(compressed, recipe)
	if err != nil {
		return RecipeReport{}, err
	}
	return report, nil
}

func PatchRecipe(compressed []byte, recipe Recipe) ([]byte, RecipeReport, error) {
	createGraphics := recipe.CreateGraphics
	if recipe.CreateGraphic != nil {
		createGraphics = append([]GraphicCreateRecipe{*recipe.CreateGraphic}, createGraphics...)
	}
	createUnits := recipe.CreateUnits
	if recipe.CreateUnit != nil {
		createUnits = append([]UnitCreateRecipe{*recipe.CreateUnit}, createUnits...)
	}
	if recipe.Empty() {
		return nil, RecipeReport{}, errors.New("recipe has no patches")
	}
	current := append([]byte(nil), compressed...)
	report := RecipeReport{InputCompressedBytes: len(compressed)}
	for i, id := range recipe.DeleteGraphics {
		output, deleteReport, err := DeleteGraphic(current, id)
		if err != nil {
			return nil, RecipeReport{}, fmt.Errorf("delete_graphics[%d] id=%d: %w", i, id, err)
		}
		current = output
		report.DeletedGraphics = append(report.DeletedGraphics, deleteReport)
	}
	for i, item := range recipe.Graphics {
		output, patchReport, err := PatchGraphic(current, item.ID, GraphicPatch{
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
			return nil, RecipeReport{}, fmt.Errorf("graphics[%d] id=%d: %w", i, item.ID, err)
		}
		current = output
		report.GraphicReports = append(report.GraphicReports, patchReport)
	}
	for i, item := range createGraphics {
		output, createReport, err := CreateGraphic(current, item)
		if err != nil {
			return nil, RecipeReport{}, fmt.Errorf("create_graphics[%d] from=%d: %w", i, item.From, err)
		}
		current = output
		report.CreatedGraphics = append(report.CreatedGraphics, createReport)
	}
	for i, item := range createUnits {
		output, createReport, err := CreateUnit(current, item)
		if err != nil {
			return nil, RecipeReport{}, fmt.Errorf("create_units[%d] from_civ=%d from_unit=%d: %w", i, item.FromCivID, item.FromUnitID, err)
		}
		current = output
		report.CreatedUnits = append(report.CreatedUnits, createReport)
	}
	for i, item := range recipe.Units {
		output, patchReport, err := PatchUnit(current, item.CivID, item.UnitID, UnitPatch{
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
			return nil, RecipeReport{}, fmt.Errorf("units[%d] civ=%d unit=%d: %w", i, item.CivID, item.UnitID, err)
		}
		current = output
		report.UnitReports = append(report.UnitReports, patchReport)
	}
	report.OutputCompressedBytes = len(current)
	if len(compressed) > 0 {
		inputPayload, err := Inflate(compressed)
		if err != nil {
			return nil, RecipeReport{}, err
		}
		outputPayload, err := Inflate(current)
		if err != nil {
			return nil, RecipeReport{}, err
		}
		report.InputInflatedBytes = len(inputPayload)
		report.OutputInflatedBytes = len(outputPayload)
		report.InflatedLengthDelta = len(outputPayload) - len(inputPayload)
	}
	report.Verified = true
	report.Verification = aoe2.StructureVerification(true)
	return current, report, nil
}

func CreateGraphic(compressed []byte, recipe GraphicCreateRecipe) ([]byte, GraphicCreateReport, error) {
	payload, err := Inflate(compressed)
	if err != nil {
		return nil, GraphicCreateReport{}, err
	}
	beforeIdx, err := Parse(payload)
	if err != nil {
		return nil, GraphicCreateReport{}, fmt.Errorf("index input: %w", err)
	}
	if beforeIdx.GraphicsSize >= math.MaxInt16 {
		return nil, GraphicCreateReport{}, fmt.Errorf("graphics table already at int16 limit: %d", beforeIdx.GraphicsSize)
	}
	template, ok := beforeIdx.Graphic(recipe.From)
	if !ok {
		return nil, GraphicCreateReport{}, fmt.Errorf("template graphic %d is absent or outside graphics table", recipe.From)
	}
	graphicsHeader, ok := beforeIdx.spanByName("graphics_header")
	if !ok || graphicsHeader.Len() != 2 {
		return nil, GraphicCreateReport{}, errors.New("graphics_header span unavailable")
	}
	graphicPointers, ok := beforeIdx.spanByName("graphic_pointers")
	if !ok || graphicPointers.Len() != beforeIdx.GraphicsSize*4 {
		return nil, GraphicCreateReport{}, errors.New("graphic_pointers span unavailable or malformed")
	}
	terrainBlock, ok := beforeIdx.spanByName("terrain_block")
	if !ok {
		return nil, GraphicCreateReport{}, errors.New("terrain_block span unavailable")
	}
	if terrainBlock.Start < graphicPointers.End {
		return nil, GraphicCreateReport{}, errors.New("terrain_block starts before graphic_pointers end")
	}
	pointerStart := graphicPointers.Start + recipe.From*4
	if pointerStart < graphicPointers.Start || pointerStart+4 > graphicPointers.End {
		return nil, GraphicCreateReport{}, fmt.Errorf("template graphic pointer %d is outside pointer table", recipe.From)
	}
	pointerBytes := append([]byte(nil), payload[pointerStart:pointerStart+4]...)
	if binary.LittleEndian.Uint32(pointerBytes) == 0 {
		return nil, GraphicCreateReport{}, fmt.Errorf("template graphic %d has an absent pointer", recipe.From)
	}

	newID := beforeIdx.GraphicsSize
	record, err := buildGraphicRecord(payload, template, GraphicPatch{
		Name:               recipe.Name,
		FileName:           recipe.FileName,
		ParticleEffectName: recipe.ParticleEffectName,
		SLP:                recipe.SLP,
		IsLoaded:           recipe.IsLoaded,
		OldColorFlag:       recipe.OldColorFlag,
		Layer:              recipe.Layer,
		PlayerColor:        recipe.PlayerColor,
		Rainbow:            recipe.Rainbow,
		TransparentSelect:  recipe.TransparentSelect,
		Coordinates:        recipe.Coordinates,
		SoundID:            recipe.SoundID,
		WwiseSoundID:       recipe.WwiseSoundID,
		FrameCount:         recipe.FrameCount,
		SpeedMultiplier:    recipe.SpeedMultiplier,
		FrameDuration:      recipe.FrameDuration,
		ReplayDelay:        recipe.ReplayDelay,
		SequenceType:       recipe.SequenceType,
		MirroringMode:      recipe.MirroringMode,
		EditorFlag:         recipe.EditorFlag,
		SetDeltas:          recipe.SetDeltas,
		SetAngleSounds:     recipe.SetAngleSounds,
	}, int16(newID))
	if err != nil {
		return nil, GraphicCreateReport{}, err
	}

	var countBytes [2]byte
	binary.LittleEndian.PutUint16(countBytes[:], uint16(beforeIdx.GraphicsSize+1))
	outPayload := make([]byte, 0, len(payload)+4+len(record))
	outPayload = append(outPayload, payload[:graphicsHeader.Start]...)
	outPayload = append(outPayload, countBytes[:]...)
	outPayload = append(outPayload, payload[graphicsHeader.End:graphicPointers.End]...)
	outPayload = append(outPayload, pointerBytes...)
	outPayload = append(outPayload, payload[graphicPointers.End:terrainBlock.Start]...)
	outPayload = append(outPayload, record...)
	outPayload = append(outPayload, payload[terrainBlock.Start:]...)

	output, err := Deflate(outPayload)
	if err != nil {
		return nil, GraphicCreateReport{}, err
	}
	roundTripPayload, err := Inflate(output)
	if err != nil {
		return nil, GraphicCreateReport{}, fmt.Errorf("inflate patched output: %w", err)
	}
	if !bytes.Equal(outPayload, roundTripPayload) {
		return nil, GraphicCreateReport{}, errors.New("patched payload changed after deflate/inflate round trip")
	}
	afterIdx, err := Parse(roundTripPayload)
	if err != nil {
		return nil, GraphicCreateReport{}, fmt.Errorf("re-index patched output: %w", err)
	}
	created, ok := afterIdx.Graphic(newID)
	if !ok {
		return nil, GraphicCreateReport{}, fmt.Errorf("patched output lost created graphic %d", newID)
	}
	if afterIdx.GraphicsSize != beforeIdx.GraphicsSize+1 {
		return nil, GraphicCreateReport{}, fmt.Errorf("graphics count did not increment by 1: got %d want %d", afterIdx.GraphicsSize, beforeIdx.GraphicsSize+1)
	}
	expectedDelta := 4 + len(record)
	if got := len(roundTripPayload) - len(payload); got != expectedDelta {
		return nil, GraphicCreateReport{}, fmt.Errorf("inflated length delta mismatch: got %d want %d", got, expectedDelta)
	}
	if created.ID != int16(newID) {
		return nil, GraphicCreateReport{}, fmt.Errorf("created graphic internal id did not read back: got %d want %d", created.ID, newID)
	}
	if err := verifyGraphicCreateReadback(created, recipe); err != nil {
		return nil, GraphicCreateReport{}, err
	}
	if err := verifyGraphicAppendCanaries(beforeIdx, afterIdx, recipe.From); err != nil {
		return nil, GraphicCreateReport{}, err
	}
	report := GraphicCreateReport{
		InputCompressedBytes:  len(compressed),
		OutputCompressedBytes: len(output),
		InputInflatedBytes:    len(payload),
		OutputInflatedBytes:   len(roundTripPayload),
		InflatedLengthDelta:   len(roundTripPayload) - len(payload),
		FromGraphicID:         recipe.From,
		NewGraphicID:          newID,
		BeforeGraphicsSize:    beforeIdx.GraphicsSize,
		AfterGraphicsSize:     afterIdx.GraphicsSize,
		Template:              template.Summary(),
		Created:               created.Summary(),
		ChangedFields:         changedGraphicCreateFields(recipe),
		Verified:              true,
		Verification:          aoe2.StructureVerification(true),
	}
	return output, report, nil
}

func DeleteGraphic(compressed []byte, graphicID int) ([]byte, GraphicDeleteReport, error) {
	payload, err := Inflate(compressed)
	if err != nil {
		return nil, GraphicDeleteReport{}, err
	}
	beforeIdx, err := Parse(payload)
	if err != nil {
		return nil, GraphicDeleteReport{}, fmt.Errorf("index input: %w", err)
	}
	if graphicID < 0 || graphicID >= beforeIdx.GraphicsSize {
		return nil, GraphicDeleteReport{}, fmt.Errorf("graphic %d is outside graphics table size %d", graphicID, beforeIdx.GraphicsSize)
	}
	graphic, ok := beforeIdx.Graphic(graphicID)
	if !ok {
		return nil, GraphicDeleteReport{}, fmt.Errorf("graphic %d is already absent", graphicID)
	}
	if graphic.RecordSpan.Len() <= 0 {
		return nil, GraphicDeleteReport{}, fmt.Errorf("graphic %d has invalid record span %d..%d", graphicID, graphic.RecordSpan.Start, graphic.RecordSpan.End)
	}
	graphicPointers, ok := beforeIdx.spanByName("graphic_pointers")
	if !ok || graphicPointers.Len() != beforeIdx.GraphicsSize*4 {
		return nil, GraphicDeleteReport{}, errors.New("graphic_pointers span unavailable or malformed")
	}
	pointerStart := graphicPointers.Start + graphicID*4
	if pointerStart < graphicPointers.Start || pointerStart+4 > graphicPointers.End {
		return nil, GraphicDeleteReport{}, fmt.Errorf("graphic pointer %d is outside pointer table", graphicID)
	}
	if binary.LittleEndian.Uint32(payload[pointerStart:pointerStart+4]) == 0 {
		return nil, GraphicDeleteReport{}, fmt.Errorf("graphic %d has an absent pointer", graphicID)
	}
	if graphic.RecordSpan.Start < graphicPointers.End {
		return nil, GraphicDeleteReport{}, fmt.Errorf("graphic %d record starts before graphic pointer table ends", graphicID)
	}

	outPayload := make([]byte, 0, len(payload)-graphic.RecordSpan.Len())
	outPayload = append(outPayload, payload[:pointerStart]...)
	outPayload = append(outPayload, 0, 0, 0, 0)
	outPayload = append(outPayload, payload[pointerStart+4:graphic.RecordSpan.Start]...)
	outPayload = append(outPayload, payload[graphic.RecordSpan.End:]...)

	output, err := Deflate(outPayload)
	if err != nil {
		return nil, GraphicDeleteReport{}, err
	}
	roundTripPayload, err := Inflate(output)
	if err != nil {
		return nil, GraphicDeleteReport{}, fmt.Errorf("inflate patched output: %w", err)
	}
	if !bytes.Equal(outPayload, roundTripPayload) {
		return nil, GraphicDeleteReport{}, errors.New("patched payload changed after deflate/inflate round trip")
	}
	afterIdx, err := Parse(roundTripPayload)
	if err != nil {
		return nil, GraphicDeleteReport{}, fmt.Errorf("re-index patched output: %w", err)
	}
	if afterIdx.GraphicsSize != beforeIdx.GraphicsSize {
		return nil, GraphicDeleteReport{}, fmt.Errorf("graphics count changed: got %d want %d", afterIdx.GraphicsSize, beforeIdx.GraphicsSize)
	}
	if _, ok := afterIdx.Graphic(graphicID); ok {
		return nil, GraphicDeleteReport{}, fmt.Errorf("graphic %d still present after delete", graphicID)
	}
	expectedDelta := -graphic.RecordSpan.Len()
	if got := len(roundTripPayload) - len(payload); got != expectedDelta {
		return nil, GraphicDeleteReport{}, fmt.Errorf("inflated length delta mismatch: got %d want %d", got, expectedDelta)
	}
	if err := verifyNeighborCanaries(beforeIdx, afterIdx, graphicID); err != nil {
		return nil, GraphicDeleteReport{}, err
	}
	report := GraphicDeleteReport{
		InputCompressedBytes:  len(compressed),
		OutputCompressedBytes: len(output),
		InputInflatedBytes:    len(payload),
		OutputInflatedBytes:   len(roundTripPayload),
		InflatedLengthDelta:   len(roundTripPayload) - len(payload),
		GraphicID:             graphicID,
		BeforeGraphicsSize:    beforeIdx.GraphicsSize,
		AfterGraphicsSize:     afterIdx.GraphicsSize,
		RemovedRecordBytes:    graphic.RecordSpan.Len(),
		Before:                graphic.Summary(),
		Verified:              true,
		Verification:          aoe2.StructureVerification(true),
	}
	report.Verification.Note = "Stable-ID graphic delete zeroes the graphic pointer slot and removes the record bytes; this is structurally verified, not engine-verified."
	return output, report, nil
}

func CreateUnit(compressed []byte, recipe UnitCreateRecipe) ([]byte, UnitCreateReport, error) {
	payload, err := Inflate(compressed)
	if err != nil {
		return nil, UnitCreateReport{}, err
	}
	beforeIdx, err := Parse(payload)
	if err != nil {
		return nil, UnitCreateReport{}, fmt.Errorf("index input: %w", err)
	}
	if recipe.FromCivID < 0 || recipe.FromCivID >= len(beforeIdx.Civs) {
		return nil, UnitCreateReport{}, fmt.Errorf("from_civ_id %d is outside civ table", recipe.FromCivID)
	}
	template, err := unitByID(beforeIdx, recipe.FromCivID, recipe.FromUnitID)
	if err != nil {
		return nil, UnitCreateReport{}, fmt.Errorf("template unit: %w", err)
	}
	civIDs, err := unitCreateCivIDs(beforeIdx, recipe)
	if err != nil {
		return nil, UnitCreateReport{}, err
	}
	if len(civIDs) == 0 {
		return nil, UnitCreateReport{}, errors.New("create_unit selected no civs")
	}
	newID := beforeIdx.Civs[civIDs[0]].UnitsSize
	if newID > math.MaxInt16 {
		return nil, UnitCreateReport{}, fmt.Errorf("unit table already at int16 limit: %d", newID)
	}
	for _, civID := range civIDs {
		if beforeIdx.Civs[civID].UnitsSize != newID {
			return nil, UnitCreateReport{}, fmt.Errorf("selected civs do not share next unit id: civ %d has size %d, want %d", civID, beforeIdx.Civs[civID].UnitsSize, newID)
		}
	}
	patch := unitCreatePatch(recipe)
	outPayload := append([]byte(nil), payload...)
	expectedDelta := 0
	for i := len(civIDs) - 1; i >= 0; i-- {
		civID := civIDs[i]
		civ := beforeIdx.Civs[civID]
		if civ.FieldSpans.UnitsSize.Len() != 2 || civ.FieldSpans.UnitPointers.Len() != civ.UnitsSize*4 {
			return nil, UnitCreateReport{}, fmt.Errorf("civ %d unit spans unavailable or malformed", civID)
		}
		sourceUnit, sourcePointerCivID := template, recipe.FromCivID
		if civID >= 0 && civID < len(beforeIdx.Civs) && recipe.FromUnitID >= 0 && recipe.FromUnitID < len(beforeIdx.Civs[civID].Units) {
			if own := beforeIdx.Civs[civID].Units[recipe.FromUnitID]; own.Present {
				sourceUnit = own
				sourcePointerCivID = civID
			}
		}
		pointerBytes, err := unitPointerBytes(payload, beforeIdx.Civs[sourcePointerCivID], recipe.FromUnitID)
		if err != nil {
			return nil, UnitCreateReport{}, err
		}
		record, err := buildUnitRecord(payload, sourceUnit, patch, int16(newID))
		if err != nil {
			return nil, UnitCreateReport{}, fmt.Errorf("civ %d: %w", civID, err)
		}
		var countBytes [2]byte
		binary.LittleEndian.PutUint16(countBytes[:], uint16(civ.UnitsSize+1))
		next := make([]byte, 0, len(outPayload)+4+len(record))
		next = append(next, outPayload[:civ.FieldSpans.UnitsSize.Start]...)
		next = append(next, countBytes[:]...)
		next = append(next, outPayload[civ.FieldSpans.UnitsSize.End:civ.FieldSpans.UnitPointers.End]...)
		next = append(next, pointerBytes...)
		next = append(next, outPayload[civ.FieldSpans.UnitPointers.End:civ.Span.End]...)
		next = append(next, record...)
		next = append(next, outPayload[civ.Span.End:]...)
		outPayload = next
		expectedDelta += 4 + len(record)
	}
	output, err := Deflate(outPayload)
	if err != nil {
		return nil, UnitCreateReport{}, err
	}
	roundTripPayload, err := Inflate(output)
	if err != nil {
		return nil, UnitCreateReport{}, fmt.Errorf("inflate patched output: %w", err)
	}
	if !bytes.Equal(outPayload, roundTripPayload) {
		return nil, UnitCreateReport{}, errors.New("patched payload changed after deflate/inflate round trip")
	}
	afterIdx, err := Parse(roundTripPayload)
	if err != nil {
		return nil, UnitCreateReport{}, fmt.Errorf("re-index patched output: %w", err)
	}
	if len(afterIdx.Civs) != len(beforeIdx.Civs) {
		return nil, UnitCreateReport{}, fmt.Errorf("civ count changed: %d -> %d", len(beforeIdx.Civs), len(afterIdx.Civs))
	}
	if got := len(roundTripPayload) - len(payload); got != expectedDelta {
		return nil, UnitCreateReport{}, fmt.Errorf("inflated length delta mismatch: got %d want %d", got, expectedDelta)
	}
	for _, civID := range civIDs {
		if afterIdx.Civs[civID].UnitsSize != beforeIdx.Civs[civID].UnitsSize+1 {
			return nil, UnitCreateReport{}, fmt.Errorf("civ %d unit count did not increment by 1: got %d want %d", civID, afterIdx.Civs[civID].UnitsSize, beforeIdx.Civs[civID].UnitsSize+1)
		}
		created, err := unitByID(afterIdx, civID, newID)
		if err != nil {
			return nil, UnitCreateReport{}, fmt.Errorf("created civ %d unit %d: %w", civID, newID, err)
		}
		if created.ID != int16(newID) {
			return nil, UnitCreateReport{}, fmt.Errorf("created civ %d unit internal id did not read back: got %d want %d", civID, created.ID, newID)
		}
		if err := verifyUnitPatchReadback(created, patch); err != nil {
			return nil, UnitCreateReport{}, fmt.Errorf("created civ %d unit %d: %w", civID, newID, err)
		}
	}
	report := UnitCreateReport{
		InputCompressedBytes:  len(compressed),
		OutputCompressedBytes: len(output),
		InputInflatedBytes:    len(payload),
		OutputInflatedBytes:   len(roundTripPayload),
		InflatedLengthDelta:   len(roundTripPayload) - len(payload),
		FromCivID:             recipe.FromCivID,
		FromUnitID:            recipe.FromUnitID,
		NewUnitID:             newID,
		CivIDs:                append([]int(nil), civIDs...),
		CreatedCount:          len(civIDs),
		ChangedFields:         changedUnitCreateFields(recipe),
		Verified:              true,
		Verification:          aoe2.StructureVerification(true),
	}
	return output, report, nil
}

func spliceGraphicRecord(payload []byte, graphic Graphic, patch GraphicPatch) ([]byte, error) {
	record, err := buildGraphicRecord(payload, graphic, patch, graphic.ID)
	if err != nil {
		return nil, err
	}

	out := make([]byte, 0, len(payload)+len(record)-graphic.RecordSpan.Len())
	out = append(out, payload[:graphic.RecordSpan.Start]...)
	out = append(out, record...)
	out = append(out, payload[graphic.RecordSpan.End:]...)
	return out, nil
}

func unitCreateCivIDs(idx *Index, recipe UnitCreateRecipe) ([]int, error) {
	allCivs := true
	if recipe.AllCivs != nil {
		allCivs = *recipe.AllCivs
	}
	if len(recipe.CivIDs) > 0 {
		allCivs = false
	}
	var ids []int
	if allCivs {
		ids = make([]int, 0, len(idx.Civs))
		for i := range idx.Civs {
			ids = append(ids, i)
		}
		return ids, nil
	}
	seen := map[int]bool{}
	for _, id := range recipe.CivIDs {
		if id < 0 || id >= len(idx.Civs) {
			return nil, fmt.Errorf("civ_id %d is outside civ table", id)
		}
		if seen[id] {
			continue
		}
		seen[id] = true
		ids = append(ids, id)
	}
	sort.Ints(ids)
	return ids, nil
}

func unitPointerBytes(payload []byte, civ Civ, unitID int) ([]byte, error) {
	if unitID < 0 || unitID >= civ.UnitsSize {
		return nil, fmt.Errorf("unit pointer %d is outside civ %d pointer table", unitID, civ.Index)
	}
	start := civ.FieldSpans.UnitPointers.Start + unitID*4
	if start < civ.FieldSpans.UnitPointers.Start || start+4 > civ.FieldSpans.UnitPointers.End {
		return nil, fmt.Errorf("unit pointer %d is outside civ %d pointer span", unitID, civ.Index)
	}
	out := append([]byte(nil), payload[start:start+4]...)
	if binary.LittleEndian.Uint32(out) == 0 {
		return nil, fmt.Errorf("unit pointer %d in civ %d is absent", unitID, civ.Index)
	}
	return out, nil
}

func applyUnitPatch(data []byte, unit UnitSummary, patch UnitPatch) ([]string, error) {
	changed := []string{}
	if patch.Class != nil {
		putI16(data, unit.FieldSpans.Class, *patch.Class)
		changed = append(changed, "class")
	}
	if patch.HitPoints != nil {
		putI16(data, unit.FieldSpans.HitPoints, *patch.HitPoints)
		changed = append(changed, "hit_points")
	}
	if patch.LineOfSight != nil {
		putF32(data, unit.FieldSpans.LineOfSight, *patch.LineOfSight)
		changed = append(changed, "line_of_sight")
	}
	if patch.MovementType != nil {
		putU8(data, unit.FieldSpans.MovementType, *patch.MovementType)
		changed = append(changed, "movement_type")
	}
	if patch.StandingGraphic1 != nil {
		putI16(data, unit.FieldSpans.StandingGraphic1, *patch.StandingGraphic1)
		changed = append(changed, "standing_graphic_1")
	}
	if patch.StandingGraphic2 != nil {
		putI16(data, unit.FieldSpans.StandingGraphic2, *patch.StandingGraphic2)
		changed = append(changed, "standing_graphic_2")
	}
	if patch.DyingGraphic != nil {
		putI16(data, unit.FieldSpans.DyingGraphic, *patch.DyingGraphic)
		changed = append(changed, "dying_graphic")
	}
	if patch.BloodUnitID != nil {
		putI16(data, unit.FieldSpans.BloodUnitID, *patch.BloodUnitID)
		changed = append(changed, "blood_unit_id")
	}
	if patch.IconID != nil {
		putI16(data, unit.FieldSpans.IconID, *patch.IconID)
		changed = append(changed, "icon_id")
	}
	if patch.Enabled != nil {
		putU8(data, unit.FieldSpans.Enabled, *patch.Enabled)
		changed = append(changed, "enabled")
	}
	for _, item := range patch.Attributes {
		if item.Index < 0 || item.Index >= len(unit.Attributes) {
			return changed, fmt.Errorf("attribute index %d outside unit attribute list length %d", item.Index, len(unit.Attributes))
		}
		span := unit.Attributes[item.Index].Span
		if item.AttributeType != nil {
			putU16(data, Span{Start: span.Start, End: span.Start + 2}, *item.AttributeType)
		}
		if item.Amount != nil {
			putF32(data, Span{Start: span.Start + 2, End: span.Start + 6}, *item.Amount)
		}
		if item.Flag != nil {
			putU8(data, Span{Start: span.Start + 6, End: span.Start + 7}, *item.Flag)
		}
		changed = append(changed, fmt.Sprintf("attribute[%d]", item.Index))
	}
	for _, item := range patch.DamageGraphics {
		if item.Index < 0 || item.Index >= len(unit.DamageGraphics) {
			return changed, fmt.Errorf("damage_graphic index %d outside damage graphic list length %d", item.Index, len(unit.DamageGraphics))
		}
		span := unit.DamageGraphics[item.Index].Span
		if item.GraphicID != nil {
			putU16(data, Span{Start: span.Start, End: span.Start + 2}, *item.GraphicID)
		}
		if item.DamagePercent != nil {
			putU16(data, Span{Start: span.Start + 2, End: span.Start + 4}, *item.DamagePercent)
		}
		if item.Flag != nil {
			putU8(data, Span{Start: span.Start + 4, End: span.Start + 5}, *item.Flag)
		}
		changed = append(changed, fmt.Sprintf("damage_graphic[%d]", item.Index))
	}
	if patch.Type50ProjectileUnitID != nil || patch.Type50MaxRange != nil || patch.Type50BlastWidth != nil ||
		patch.Type50AttackGraphic != nil || patch.Type50BlastDamage != nil ||
		len(patch.Type50Attacks) > 0 || len(patch.Type50Armours) > 0 {
		type50, err := requireType50(unit)
		if err != nil {
			return changed, err
		}
		if patch.Type50ProjectileUnitID != nil {
			putI16(data, type50.FieldSpans.ProjectileUnitID, *patch.Type50ProjectileUnitID)
			changed = append(changed, "type50_projectile_unit_id")
		}
		if patch.Type50MaxRange != nil {
			putF32(data, type50.FieldSpans.MaxRange, *patch.Type50MaxRange)
			changed = append(changed, "type50_max_range")
		}
		if patch.Type50BlastWidth != nil {
			putF32(data, type50.FieldSpans.BlastWidth, *patch.Type50BlastWidth)
			changed = append(changed, "type50_blast_width")
		}
		if patch.Type50AttackGraphic != nil {
			putI16(data, type50.FieldSpans.AttackGraphic, *patch.Type50AttackGraphic)
			changed = append(changed, "type50_attack_graphic")
		}
		if patch.Type50BlastDamage != nil {
			putF32(data, type50.FieldSpans.BlastDamage, *patch.Type50BlastDamage)
			changed = append(changed, "type50_blast_damage")
		}
		names, err := applyWeaponInfoPatches(data, type50.Attacks, patch.Type50Attacks, "type50_attack")
		if err != nil {
			return changed, err
		}
		changed = append(changed, names...)
		names, err = applyWeaponInfoPatches(data, type50.Armours, patch.Type50Armours, "type50_armour")
		if err != nil {
			return changed, err
		}
		changed = append(changed, names...)
	}
	if patch.TrainTime0 != nil || patch.TrainUnitID0 != nil || patch.TrainButtonID0 != nil ||
		patch.TrainHotKeyID0 != nil || patch.CreatableButtonIconID != nil || patch.CreatableButtonHotkeyAction != nil ||
		len(patch.Costs) > 0 || len(patch.TrainLocations) > 0 {
		creatable, err := requireCreatable(unit)
		if err != nil {
			return changed, err
		}
		if patch.TrainTime0 != nil || patch.TrainUnitID0 != nil || patch.TrainButtonID0 != nil || patch.TrainHotKeyID0 != nil {
			if len(creatable.TrainLocations) == 0 {
				return changed, fmt.Errorf("unit %d in civ %d has no train location 0", unit.Index, unit.CivIndex)
			}
		}
		if patch.TrainTime0 != nil {
			putI16(data, creatable.FieldSpans.TrainTime0, *patch.TrainTime0)
			changed = append(changed, "train_time_0")
		}
		if patch.TrainUnitID0 != nil {
			putI16(data, creatable.FieldSpans.TrainUnitID0, *patch.TrainUnitID0)
			changed = append(changed, "train_unit_id_0")
		}
		if patch.TrainButtonID0 != nil {
			putU8(data, creatable.FieldSpans.TrainButtonID0, *patch.TrainButtonID0)
			changed = append(changed, "train_button_id_0")
		}
		if patch.TrainHotKeyID0 != nil {
			putI32(data, creatable.FieldSpans.TrainHotKeyID0, *patch.TrainHotKeyID0)
			changed = append(changed, "train_hotkey_id_0")
		}
		if patch.CreatableButtonIconID != nil {
			putI16(data, creatable.FieldSpans.ButtonIconID, *patch.CreatableButtonIconID)
			changed = append(changed, "creatable_button_icon_id")
		}
		if patch.CreatableButtonHotkeyAction != nil {
			putI16(data, creatable.FieldSpans.ButtonHotkeyAction, *patch.CreatableButtonHotkeyAction)
			changed = append(changed, "creatable_button_hotkey_action")
		}
		names, err := applyCostPatches(data, creatable.Costs, patch.Costs)
		if err != nil {
			return changed, err
		}
		changed = append(changed, names...)
		names, err = applyTrainLocationPatches(data, creatable.TrainLocations, patch.TrainLocations)
		if err != nil {
			return changed, err
		}
		changed = append(changed, names...)
	}
	for _, item := range patch.LinkedBuildings {
		if unit.Building == nil {
			return changed, fmt.Errorf("unit %d in civ %d has no building subrecord", unit.Index, unit.CivIndex)
		}
		if item.Index < 0 || item.Index >= len(unit.Building.LinkedBuildings) {
			return changed, fmt.Errorf("linked_building index %d outside row count %d", item.Index, len(unit.Building.LinkedBuildings))
		}
		span := unit.Building.LinkedBuildings[item.Index].Span
		if item.UnitID != nil {
			putI16(data, Span{Start: span.Start, End: span.Start + 2}, *item.UnitID)
		}
		if item.X != nil {
			putF32(data, Span{Start: span.Start + 2, End: span.Start + 6}, *item.X)
		}
		if item.Y != nil {
			putF32(data, Span{Start: span.Start + 6, End: span.Start + 10}, *item.Y)
		}
		changed = append(changed, fmt.Sprintf("linked_building[%d]", item.Index))
	}
	if len(patch.Tasks) > 0 {
		if unit.Action == nil {
			return changed, fmt.Errorf("unit %d in civ %d has no action task block", unit.Index, unit.CivIndex)
		}
		names, err := applyTaskPatches(data, unit.Action.Tasks, patch.Tasks)
		if err != nil {
			return changed, err
		}
		changed = append(changed, names...)
	}
	return changed, nil
}

func applyWeaponInfoPatches(data []byte, rows []WeaponInfo, patches []WeaponInfoPatch, name string) ([]string, error) {
	changed := []string{}
	for _, item := range patches {
		if item.Index < 0 || item.Index >= len(rows) {
			return changed, fmt.Errorf("%s index %d outside row count %d", name, item.Index, len(rows))
		}
		span := rows[item.Index].Span
		if item.Class != nil {
			putI16(data, Span{Start: span.Start, End: span.Start + 2}, *item.Class)
		}
		if item.Value != nil {
			putI16(data, Span{Start: span.Start + 2, End: span.Start + 4}, *item.Value)
		}
		changed = append(changed, fmt.Sprintf("%s[%d]", name, item.Index))
	}
	return changed, nil
}

func applyCostPatches(data []byte, rows []AttributeCost, patches []AttributeCostPatch) ([]string, error) {
	changed := []string{}
	for _, item := range patches {
		if item.Index < 0 || item.Index >= len(rows) {
			return changed, fmt.Errorf("cost index %d outside cost count %d", item.Index, len(rows))
		}
		span := rows[item.Index].Span
		if item.AttributeType != nil {
			putI16(data, Span{Start: span.Start, End: span.Start + 2}, *item.AttributeType)
		}
		if item.Amount != nil {
			putI16(data, Span{Start: span.Start + 2, End: span.Start + 4}, *item.Amount)
		}
		if item.Flag != nil {
			putU8(data, Span{Start: span.Start + 4, End: span.Start + 5}, *item.Flag)
		}
		if item.Padding != nil {
			putU8(data, Span{Start: span.Start + 5, End: span.Start + 6}, *item.Padding)
		}
		changed = append(changed, fmt.Sprintf("cost[%d]", item.Index))
	}
	return changed, nil
}

func applyTrainLocationPatches(data []byte, rows []TrainLocation, patches []TrainLocationPatch) ([]string, error) {
	changed := []string{}
	for _, item := range patches {
		if item.Index < 0 || item.Index >= len(rows) {
			return changed, fmt.Errorf("train_location index %d outside train location count %d", item.Index, len(rows))
		}
		span := rows[item.Index].Span
		if item.TrainTime != nil {
			putI16(data, Span{Start: span.Start, End: span.Start + 2}, *item.TrainTime)
		}
		if item.TrainUnitID != nil {
			putI16(data, Span{Start: span.Start + 2, End: span.Start + 4}, *item.TrainUnitID)
		}
		if item.TrainButton != nil {
			putU8(data, Span{Start: span.Start + 4, End: span.Start + 5}, *item.TrainButton)
		}
		if item.TrainHotkey != nil {
			putI32(data, Span{Start: span.Start + 5, End: span.Start + 9}, *item.TrainHotkey)
		}
		changed = append(changed, fmt.Sprintf("train_location[%d]", item.Index))
	}
	return changed, nil
}

func applyTaskPatches(data []byte, rows []TaskSummary, patches []TaskPatch) ([]string, error) {
	changed := []string{}
	for _, item := range patches {
		if item.Index < 0 || item.Index >= len(rows) {
			return changed, fmt.Errorf("task index %d outside task count %d", item.Index, len(rows))
		}
		span := rows[item.Index].Span
		if item.RecordType != nil {
			putI16(data, Span{Start: span.Start, End: span.Start + 2}, *item.RecordType)
		}
		if item.ID != nil {
			putI16(data, Span{Start: span.Start + 2, End: span.Start + 4}, *item.ID)
		}
		if item.IsDefault != nil {
			putU8(data, Span{Start: span.Start + 4, End: span.Start + 5}, *item.IsDefault)
		}
		if item.ActionType != nil {
			putI16(data, Span{Start: span.Start + 5, End: span.Start + 7}, *item.ActionType)
		}
		if item.ObjectClass != nil {
			putI16(data, Span{Start: span.Start + 7, End: span.Start + 9}, *item.ObjectClass)
		}
		if item.ObjectID != nil {
			putI16(data, Span{Start: span.Start + 9, End: span.Start + 11}, *item.ObjectID)
		}
		if item.TerrainID != nil {
			putI16(data, Span{Start: span.Start + 11, End: span.Start + 13}, *item.TerrainID)
		}
		if item.AttributeTypes != nil {
			if len(item.AttributeTypes) != 4 {
				return changed, fmt.Errorf("task[%d] attribute_types length=%d, want 4", item.Index, len(item.AttributeTypes))
			}
			for i, value := range item.AttributeTypes {
				offset := span.Start + 13 + i*2
				putI16(data, Span{Start: offset, End: offset + 2}, value)
			}
		}
		if item.WorkValue1 != nil {
			putF32(data, Span{Start: span.Start + 21, End: span.Start + 25}, *item.WorkValue1)
		}
		if item.WorkValue2 != nil {
			putF32(data, Span{Start: span.Start + 25, End: span.Start + 29}, *item.WorkValue2)
		}
		if item.WorkRange != nil {
			putF32(data, Span{Start: span.Start + 29, End: span.Start + 33}, *item.WorkRange)
		}
		if item.AutoSearchTargets != nil {
			putU8(data, Span{Start: span.Start + 33, End: span.Start + 34}, *item.AutoSearchTargets)
		}
		if item.SearchWaitTime != nil {
			putF32(data, Span{Start: span.Start + 34, End: span.Start + 38}, *item.SearchWaitTime)
		}
		if item.EnableTargeting != nil {
			putU8(data, Span{Start: span.Start + 38, End: span.Start + 39}, *item.EnableTargeting)
		}
		if item.CombatLevel != nil {
			putU8(data, Span{Start: span.Start + 39, End: span.Start + 40}, *item.CombatLevel)
		}
		changed = append(changed, fmt.Sprintf("task[%d]", item.Index))
	}
	return changed, nil
}

func unitCreatePatch(recipe UnitCreateRecipe) UnitPatch {
	return UnitPatch{
		Class:                       recipe.Class,
		HitPoints:                   recipe.HitPoints,
		LineOfSight:                 recipe.LineOfSight,
		MovementType:                recipe.MovementType,
		StandingGraphic1:            recipe.StandingGraphic1,
		StandingGraphic2:            recipe.StandingGraphic2,
		DyingGraphic:                recipe.DyingGraphic,
		BloodUnitID:                 recipe.BloodUnitID,
		IconID:                      recipe.IconID,
		Enabled:                     recipe.Enabled,
		Attributes:                  recipe.Attributes,
		DamageGraphics:              recipe.DamageGraphics,
		SetDamageGraphics:           recipe.SetDamageGraphics,
		Type50ProjectileUnitID:      recipe.Type50ProjectileUnitID,
		Type50MaxRange:              recipe.Type50MaxRange,
		Type50BlastWidth:            recipe.Type50BlastWidth,
		Type50AttackGraphic:         recipe.Type50AttackGraphic,
		Type50BlastDamage:           recipe.Type50BlastDamage,
		Type50Attacks:               recipe.Type50Attacks,
		Type50Armours:               recipe.Type50Armours,
		SetType50Attacks:            recipe.SetType50Attacks,
		SetType50Armours:            recipe.SetType50Armours,
		TrainTime0:                  recipe.TrainTime0,
		TrainUnitID0:                recipe.TrainUnitID0,
		TrainButtonID0:              recipe.TrainButtonID0,
		TrainHotKeyID0:              recipe.TrainHotKeyID0,
		CreatableButtonIconID:       recipe.CreatableButtonIconID,
		CreatableButtonHotkeyAction: recipe.CreatableButtonHotkeyAction,
		Costs:                       recipe.Costs,
		TrainLocations:              recipe.TrainLocations,
		SetTrainLocations:           recipe.SetTrainLocations,
		SetDropSites:                recipe.SetDropSites,
		LinkedBuildings:             recipe.LinkedBuildings,
		Tasks:                       recipe.Tasks,
		SetTasks:                    recipe.SetTasks,
	}
}

func buildUnitRecord(payload []byte, template UnitSummary, patch UnitPatch, id int16) ([]byte, error) {
	if template.RecordStart < 0 || template.RecordEnd < template.RecordStart || template.RecordEnd > len(payload) {
		return nil, fmt.Errorf("template unit span invalid: %d..%d", template.RecordStart, template.RecordEnd)
	}
	data := append([]byte(nil), payload[template.RecordStart:template.RecordEnd]...)
	parsed, err := parseUnit(&cursor{data: data}, template.CivIndex, int(id), true)
	if err != nil {
		return nil, fmt.Errorf("reparse rebuilt unit record: %w", err)
	}
	putI16(data, parsed.FieldSpans.ID, id)
	if patch.needsUnitRecordRebuild() {
		rebuilt, _, err := rebuildUnitRecordData(data, template.CivIndex, int(id), patch)
		if err != nil {
			return nil, err
		}
		return rebuilt, nil
	}
	if _, err := applyUnitPatch(data, parsed, patch); err != nil {
		return nil, err
	}
	return data, nil
}

func buildGraphicRecord(payload []byte, graphic Graphic, patch GraphicPatch, id int16) ([]byte, error) {
	name := graphic.Name.Value
	if patch.Name != nil {
		name = *patch.Name
	}
	fileName := graphic.FileName.Value
	if patch.FileName != nil {
		fileName = *patch.FileName
	}
	particleEffectName := graphic.ParticleEffectName.Value
	if patch.ParticleEffectName != nil {
		particleEffectName = *patch.ParticleEffectName
	}

	var record bytes.Buffer
	if err := writeDebugString(&record, name); err != nil {
		return nil, err
	}
	if err := writeDebugString(&record, fileName); err != nil {
		return nil, err
	}
	if err := writeDebugString(&record, particleEffectName); err != nil {
		return nil, err
	}
	record.Write(payload[graphic.ParticleEffectName.Span.End:graphic.RecordSpan.End])
	data := record.Bytes()
	parsed, err := parseGraphic(&cursor{data: data}, int(id))
	if err != nil {
		return nil, fmt.Errorf("reparse rebuilt graphic record: %w", err)
	}
	putI16(data, parsed.FieldSpans.ID, id)
	if _, err := applyGraphicScalarPatch(data, parsed, patch); err != nil {
		return nil, err
	}
	data, err = applyGraphicListPatch(data, parsed, patch)
	if err != nil {
		return nil, err
	}
	if _, err := parseGraphic(&cursor{data: data}, int(id)); err != nil {
		return nil, fmt.Errorf("reparse rebuilt graphic child lists: %w", err)
	}
	return data, nil
}

func applyGraphicListPatch(record []byte, graphic Graphic, patch GraphicPatch) ([]byte, error) {
	replacements := []recordReplacement{}
	if patch.SetDeltas != nil {
		if len(*patch.SetDeltas) > math.MaxInt16 {
			return nil, fmt.Errorf("set_deltas length=%d exceeds int16 limit", len(*patch.SetDeltas))
		}
		putI16(record, graphic.FieldSpans.DeltaCount, int16(len(*patch.SetDeltas)))
		replacements = append(replacements, recordReplacement{
			Start: graphicDeltasStart(graphic),
			End:   graphicDeltasEnd(graphic),
			Data:  encodeGraphicDeltas(*patch.SetDeltas),
		})
	}
	if patch.SetAngleSounds != nil {
		if len(*patch.SetAngleSounds) == 0 {
			putU8(record, graphic.FieldSpans.AngleSoundsUsed, 0)
		} else {
			if len(*patch.SetAngleSounds) != int(graphic.AngleCount) {
				return nil, fmt.Errorf("set_angle_sounds length=%d, want angle_count %d", len(*patch.SetAngleSounds), graphic.AngleCount)
			}
			putU8(record, graphic.FieldSpans.AngleSoundsUsed, 1)
		}
		replacements = append(replacements, recordReplacement{
			Start: graphicAngleSoundsStart(graphic),
			End:   graphicAngleSoundsEnd(graphic),
			Data:  encodeGraphicAngleSounds(*patch.SetAngleSounds),
		})
	}
	return applyRecordReplacements(record, replacements)
}

func graphicDeltasStart(graphic Graphic) int {
	if len(graphic.Deltas) > 0 {
		return graphic.Deltas[0].Span.Start
	}
	return graphic.FieldSpans.EditorFlag.End
}

func graphicDeltasEnd(graphic Graphic) int {
	if len(graphic.Deltas) > 0 {
		return graphic.Deltas[len(graphic.Deltas)-1].Span.End
	}
	return graphic.FieldSpans.EditorFlag.End
}

func graphicAngleSoundsStart(graphic Graphic) int {
	if len(graphic.AngleSounds) > 0 {
		return graphic.AngleSounds[0].Span.Start
	}
	return graphicDeltasEnd(graphic)
}

func graphicAngleSoundsEnd(graphic Graphic) int {
	if len(graphic.AngleSounds) > 0 {
		return graphic.AngleSounds[len(graphic.AngleSounds)-1].Span.End
	}
	return graphicAngleSoundsStart(graphic)
}

func encodeGraphicDeltas(rows []GraphicDeltaRow) []byte {
	out := make([]byte, 0, len(rows)*16)
	for _, row := range rows {
		out = binary.LittleEndian.AppendUint16(out, uint16(row.GraphicID))
		out = binary.LittleEndian.AppendUint16(out, uint16(row.Padding1))
		out = binary.LittleEndian.AppendUint32(out, uint32(row.SpritePtr))
		out = binary.LittleEndian.AppendUint16(out, uint16(row.OffsetX))
		out = binary.LittleEndian.AppendUint16(out, uint16(row.OffsetY))
		out = binary.LittleEndian.AppendUint16(out, uint16(row.DisplayAngle))
		out = binary.LittleEndian.AppendUint16(out, uint16(row.Padding2))
	}
	return out
}

func encodeGraphicAngleSounds(rows []GraphicAngleSoundRow) []byte {
	out := make([]byte, 0, len(rows)*24)
	for _, row := range rows {
		out = binary.LittleEndian.AppendUint16(out, uint16(row.FrameNum))
		out = binary.LittleEndian.AppendUint16(out, uint16(row.SoundID))
		out = binary.LittleEndian.AppendUint32(out, row.WwiseSoundID)
		out = binary.LittleEndian.AppendUint16(out, uint16(row.FrameNum2))
		out = binary.LittleEndian.AppendUint32(out, row.WwiseSoundID2)
		out = binary.LittleEndian.AppendUint16(out, uint16(row.SoundID2))
		out = binary.LittleEndian.AppendUint16(out, uint16(row.FrameNum3))
		out = binary.LittleEndian.AppendUint32(out, row.WwiseSoundID3)
		out = binary.LittleEndian.AppendUint16(out, uint16(row.SoundID3))
	}
	return out
}

func expectedGraphicPatchLengthDelta(before Graphic, patch GraphicPatch) int {
	delta := 0
	if patch.Name != nil {
		delta += len(*patch.Name) - before.Name.ByteLength
	}
	if patch.FileName != nil {
		delta += len(*patch.FileName) - before.FileName.ByteLength
	}
	if patch.ParticleEffectName != nil {
		delta += len(*patch.ParticleEffectName) - before.ParticleEffectName.ByteLength
	}
	if patch.SetDeltas != nil {
		delta += (len(*patch.SetDeltas) - len(before.Deltas)) * 16
	}
	if patch.SetAngleSounds != nil {
		delta += (len(*patch.SetAngleSounds) - len(before.AngleSounds)) * 24
	}
	return delta
}

func writeDebugString(out *bytes.Buffer, value string) error {
	if len(value) > math.MaxUint16 {
		return fmt.Errorf("debug string too long: %d bytes > %d", len(value), math.MaxUint16)
	}
	var buf [4]byte
	binary.LittleEndian.PutUint16(buf[0:2], DebugStringMarker)
	binary.LittleEndian.PutUint16(buf[2:4], uint16(len(value)))
	out.Write(buf[:])
	out.WriteString(value)
	return nil
}

func unitByID(idx *Index, civID, unitID int) (UnitSummary, error) {
	if civID < 0 || civID >= len(idx.Civs) {
		return UnitSummary{}, fmt.Errorf("civ %d is outside civ table", civID)
	}
	civ := idx.Civs[civID]
	if unitID < 0 || unitID >= len(civ.Units) {
		return UnitSummary{}, fmt.Errorf("unit %d is outside civ %d unit table", unitID, civID)
	}
	unit := civ.Units[unitID]
	if !unit.Present {
		return UnitSummary{}, fmt.Errorf("unit %d in civ %d is absent", unitID, civID)
	}
	return unit, nil
}

func verifyUnitPatchReadback(after UnitSummary, patch UnitPatch) error {
	if patch.Class != nil && after.Class != *patch.Class {
		return fmt.Errorf("class patch did not read back: got %d want %d", after.Class, *patch.Class)
	}
	if patch.HitPoints != nil && after.HitPoints != *patch.HitPoints {
		return fmt.Errorf("hit_points patch did not read back: got %d want %d", after.HitPoints, *patch.HitPoints)
	}
	if patch.LineOfSight != nil && after.LineOfSight != *patch.LineOfSight {
		return fmt.Errorf("line_of_sight patch did not read back: got %f want %f", after.LineOfSight, *patch.LineOfSight)
	}
	if patch.MovementType != nil && after.MovementType != *patch.MovementType {
		return fmt.Errorf("movement_type patch did not read back: got %d want %d", after.MovementType, *patch.MovementType)
	}
	if patch.StandingGraphic1 != nil && after.StandingGraphic1 != *patch.StandingGraphic1 {
		return fmt.Errorf("standing_graphic_1 patch did not read back: got %d want %d", after.StandingGraphic1, *patch.StandingGraphic1)
	}
	if patch.StandingGraphic2 != nil && after.StandingGraphic2 != *patch.StandingGraphic2 {
		return fmt.Errorf("standing_graphic_2 patch did not read back: got %d want %d", after.StandingGraphic2, *patch.StandingGraphic2)
	}
	if patch.DyingGraphic != nil && after.DyingGraphic != *patch.DyingGraphic {
		return fmt.Errorf("dying_graphic patch did not read back: got %d want %d", after.DyingGraphic, *patch.DyingGraphic)
	}
	if patch.BloodUnitID != nil && after.BloodUnitID != *patch.BloodUnitID {
		return fmt.Errorf("blood_unit_id patch did not read back: got %d want %d", after.BloodUnitID, *patch.BloodUnitID)
	}
	if patch.IconID != nil && after.IconID != *patch.IconID {
		return fmt.Errorf("icon_id patch did not read back: got %d want %d", after.IconID, *patch.IconID)
	}
	if patch.Enabled != nil && after.Enabled != *patch.Enabled {
		return fmt.Errorf("enabled patch did not read back: got %d want %d", after.Enabled, *patch.Enabled)
	}
	for _, item := range patch.Attributes {
		if item.Index < 0 || item.Index >= len(after.Attributes) {
			return fmt.Errorf("attribute[%d] patch did not read back: row missing", item.Index)
		}
		row := after.Attributes[item.Index]
		if item.AttributeType != nil && row.AttributeType != *item.AttributeType {
			return fmt.Errorf("attribute[%d].attribute_type patch did not read back: got %d want %d", item.Index, row.AttributeType, *item.AttributeType)
		}
		if item.Amount != nil && row.Amount != *item.Amount {
			return fmt.Errorf("attribute[%d].amount patch did not read back: got %f want %f", item.Index, row.Amount, *item.Amount)
		}
		if item.Flag != nil && row.Flag != *item.Flag {
			return fmt.Errorf("attribute[%d].flag patch did not read back: got %d want %d", item.Index, row.Flag, *item.Flag)
		}
	}
	for _, item := range patch.DamageGraphics {
		if item.Index < 0 || item.Index >= len(after.DamageGraphics) {
			return fmt.Errorf("damage_graphic[%d] patch did not read back: row missing", item.Index)
		}
		row := after.DamageGraphics[item.Index]
		if item.GraphicID != nil && row.GraphicID != *item.GraphicID {
			return fmt.Errorf("damage_graphic[%d].graphic_id patch did not read back: got %d want %d", item.Index, row.GraphicID, *item.GraphicID)
		}
		if item.DamagePercent != nil && row.DamagePercent != *item.DamagePercent {
			return fmt.Errorf("damage_graphic[%d].damage_percent patch did not read back: got %d want %d", item.Index, row.DamagePercent, *item.DamagePercent)
		}
		if item.Flag != nil && row.Flag != *item.Flag {
			return fmt.Errorf("damage_graphic[%d].flag patch did not read back: got %d want %d", item.Index, row.Flag, *item.Flag)
		}
	}
	if patch.SetDamageGraphics != nil {
		if len(after.DamageGraphics) != len(*patch.SetDamageGraphics) {
			return fmt.Errorf("set_damage_graphics readback length got %d want %d", len(after.DamageGraphics), len(*patch.SetDamageGraphics))
		}
		for i, want := range *patch.SetDamageGraphics {
			got := after.DamageGraphics[i]
			if got.GraphicID != want.GraphicID || got.DamagePercent != want.DamagePercent || got.Flag != want.Flag {
				return fmt.Errorf("set_damage_graphics[%d] readback got graphic=%d damage=%d flag=%d want graphic=%d damage=%d flag=%d", i, got.GraphicID, got.DamagePercent, got.Flag, want.GraphicID, want.DamagePercent, want.Flag)
			}
		}
	}
	if patch.Type50ProjectileUnitID != nil {
		type50, err := requireType50(after)
		if err != nil {
			return err
		}
		if type50.ProjectileUnitID != *patch.Type50ProjectileUnitID {
			return fmt.Errorf("type50_projectile_unit_id patch did not read back: got %d want %d", type50.ProjectileUnitID, *patch.Type50ProjectileUnitID)
		}
	}
	if patch.Type50MaxRange != nil {
		type50, err := requireType50(after)
		if err != nil {
			return err
		}
		if type50.MaxRange != *patch.Type50MaxRange {
			return fmt.Errorf("type50_max_range patch did not read back: got %f want %f", type50.MaxRange, *patch.Type50MaxRange)
		}
	}
	if patch.Type50BlastWidth != nil {
		type50, err := requireType50(after)
		if err != nil {
			return err
		}
		if type50.BlastWidth != *patch.Type50BlastWidth {
			return fmt.Errorf("type50_blast_width patch did not read back: got %f want %f", type50.BlastWidth, *patch.Type50BlastWidth)
		}
	}
	if patch.Type50AttackGraphic != nil {
		type50, err := requireType50(after)
		if err != nil {
			return err
		}
		if type50.AttackGraphic != *patch.Type50AttackGraphic {
			return fmt.Errorf("type50_attack_graphic patch did not read back: got %d want %d", type50.AttackGraphic, *patch.Type50AttackGraphic)
		}
	}
	if patch.Type50BlastDamage != nil {
		type50, err := requireType50(after)
		if err != nil {
			return err
		}
		if type50.BlastDamage != *patch.Type50BlastDamage {
			return fmt.Errorf("type50_blast_damage patch did not read back: got %f want %f", type50.BlastDamage, *patch.Type50BlastDamage)
		}
	}
	if err := verifyWeaponInfoPatches(after, patch.Type50Attacks, true); err != nil {
		return err
	}
	if err := verifyWeaponInfoPatches(after, patch.Type50Armours, false); err != nil {
		return err
	}
	if err := verifySetWeaponInfoRows(after, patch.SetType50Attacks, true); err != nil {
		return err
	}
	if err := verifySetWeaponInfoRows(after, patch.SetType50Armours, false); err != nil {
		return err
	}
	if patch.TrainTime0 != nil {
		creatable, err := requireCreatable(after)
		if err != nil {
			return err
		}
		if creatable.TrainTime0 != *patch.TrainTime0 {
			return fmt.Errorf("train_time_0 patch did not read back: got %d want %d", creatable.TrainTime0, *patch.TrainTime0)
		}
	}
	if patch.TrainUnitID0 != nil {
		creatable, err := requireCreatable(after)
		if err != nil {
			return err
		}
		if creatable.TrainUnitID0 != *patch.TrainUnitID0 {
			return fmt.Errorf("train_unit_id_0 patch did not read back: got %d want %d", creatable.TrainUnitID0, *patch.TrainUnitID0)
		}
	}
	if patch.TrainButtonID0 != nil {
		creatable, err := requireCreatable(after)
		if err != nil {
			return err
		}
		if creatable.TrainButtonID0 != *patch.TrainButtonID0 {
			return fmt.Errorf("train_button_id_0 patch did not read back: got %d want %d", creatable.TrainButtonID0, *patch.TrainButtonID0)
		}
	}
	if patch.TrainHotKeyID0 != nil {
		creatable, err := requireCreatable(after)
		if err != nil {
			return err
		}
		if creatable.TrainHotKeyID0 != *patch.TrainHotKeyID0 {
			return fmt.Errorf("train_hotkey_id_0 patch did not read back: got %d want %d", creatable.TrainHotKeyID0, *patch.TrainHotKeyID0)
		}
	}
	if patch.CreatableButtonIconID != nil {
		creatable, err := requireCreatable(after)
		if err != nil {
			return err
		}
		if creatable.ButtonIconID != *patch.CreatableButtonIconID {
			return fmt.Errorf("creatable_button_icon_id patch did not read back: got %d want %d", creatable.ButtonIconID, *patch.CreatableButtonIconID)
		}
	}
	if patch.CreatableButtonHotkeyAction != nil {
		creatable, err := requireCreatable(after)
		if err != nil {
			return err
		}
		if creatable.ButtonHotkeyAction != *patch.CreatableButtonHotkeyAction {
			return fmt.Errorf("creatable_button_hotkey_action patch did not read back: got %d want %d", creatable.ButtonHotkeyAction, *patch.CreatableButtonHotkeyAction)
		}
	}
	if err := verifyCostPatches(after, patch.Costs); err != nil {
		return err
	}
	if err := verifyTrainLocationPatches(after, patch.TrainLocations); err != nil {
		return err
	}
	if patch.SetTrainLocations != nil {
		creatable, err := requireCreatable(after)
		if err != nil {
			return err
		}
		if len(creatable.TrainLocations) != len(*patch.SetTrainLocations) {
			return fmt.Errorf("set_train_locations readback length got %d want %d", len(creatable.TrainLocations), len(*patch.SetTrainLocations))
		}
		for i, want := range *patch.SetTrainLocations {
			got := creatable.TrainLocations[i]
			if got.TrainTime != want.TrainTime || got.TrainUnitID != want.TrainUnitID || got.TrainButton != want.TrainButton || got.TrainHotkey != want.TrainHotkey {
				return fmt.Errorf("set_train_locations[%d] readback got time=%d unit=%d button=%d hotkey=%d want time=%d unit=%d button=%d hotkey=%d", i, got.TrainTime, got.TrainUnitID, got.TrainButton, got.TrainHotkey, want.TrainTime, want.TrainUnitID, want.TrainButton, want.TrainHotkey)
			}
		}
	}
	if patch.SetDropSites != nil {
		action, err := requireAction(after)
		if err != nil {
			return err
		}
		if len(action.DropSites) != len(*patch.SetDropSites) {
			return fmt.Errorf("set_drop_sites readback length got %d want %d", len(action.DropSites), len(*patch.SetDropSites))
		}
		for i, want := range *patch.SetDropSites {
			if action.DropSites[i] != want {
				return fmt.Errorf("set_drop_sites[%d] readback got %d want %d", i, action.DropSites[i], want)
			}
		}
	}
	if err := verifyLinkedBuildingPatches(after, patch.LinkedBuildings); err != nil {
		return err
	}
	if err := verifyTaskPatches(after, patch.Tasks); err != nil {
		return err
	}
	if err := verifySetTaskRows(after, patch.SetTasks); err != nil {
		return err
	}
	return nil
}

func verifyWeaponInfoPatches(after UnitSummary, patches []WeaponInfoPatch, attacks bool) error {
	if len(patches) == 0 {
		return nil
	}
	type50, err := requireType50(after)
	if err != nil {
		return err
	}
	rows := type50.Armours
	name := "type50_armour"
	if attacks {
		rows = type50.Attacks
		name = "type50_attack"
	}
	for _, item := range patches {
		if item.Index < 0 || item.Index >= len(rows) {
			return fmt.Errorf("%s[%d] patch did not read back: row missing", name, item.Index)
		}
		row := rows[item.Index]
		if item.Class != nil && row.Class != *item.Class {
			return fmt.Errorf("%s[%d].class patch did not read back: got %d want %d", name, item.Index, row.Class, *item.Class)
		}
		if item.Value != nil && row.Value != *item.Value {
			return fmt.Errorf("%s[%d].value patch did not read back: got %d want %d", name, item.Index, row.Value, *item.Value)
		}
	}
	return nil
}

func verifySetWeaponInfoRows(after UnitSummary, rows *[]WeaponInfoRow, attacks bool) error {
	if rows == nil {
		return nil
	}
	type50, err := requireType50(after)
	if err != nil {
		return err
	}
	gotRows := type50.Armours
	name := "set_type50_armours"
	if attacks {
		gotRows = type50.Attacks
		name = "set_type50_attacks"
	}
	if len(gotRows) != len(*rows) {
		return fmt.Errorf("%s readback length got %d want %d", name, len(gotRows), len(*rows))
	}
	for i, want := range *rows {
		got := gotRows[i]
		if got.Class != want.Class || got.Value != want.Value {
			return fmt.Errorf("%s[%d] readback got class=%d value=%d want class=%d value=%d", name, i, got.Class, got.Value, want.Class, want.Value)
		}
	}
	return nil
}

func verifyCostPatches(after UnitSummary, patches []AttributeCostPatch) error {
	if len(patches) == 0 {
		return nil
	}
	creatable, err := requireCreatable(after)
	if err != nil {
		return err
	}
	for _, item := range patches {
		if item.Index < 0 || item.Index >= len(creatable.Costs) {
			return fmt.Errorf("cost[%d] patch did not read back: row missing", item.Index)
		}
		row := creatable.Costs[item.Index]
		if item.AttributeType != nil && row.AttributeType != *item.AttributeType {
			return fmt.Errorf("cost[%d].attribute_type patch did not read back: got %d want %d", item.Index, row.AttributeType, *item.AttributeType)
		}
		if item.Amount != nil && row.Amount != *item.Amount {
			return fmt.Errorf("cost[%d].amount patch did not read back: got %d want %d", item.Index, row.Amount, *item.Amount)
		}
		if item.Flag != nil && row.Flag != *item.Flag {
			return fmt.Errorf("cost[%d].flag patch did not read back: got %d want %d", item.Index, row.Flag, *item.Flag)
		}
		if item.Padding != nil && row.Padding != *item.Padding {
			return fmt.Errorf("cost[%d].padding patch did not read back: got %d want %d", item.Index, row.Padding, *item.Padding)
		}
	}
	return nil
}

func verifyTrainLocationPatches(after UnitSummary, patches []TrainLocationPatch) error {
	if len(patches) == 0 {
		return nil
	}
	creatable, err := requireCreatable(after)
	if err != nil {
		return err
	}
	for _, item := range patches {
		if item.Index < 0 || item.Index >= len(creatable.TrainLocations) {
			return fmt.Errorf("train_location[%d] patch did not read back: row missing", item.Index)
		}
		row := creatable.TrainLocations[item.Index]
		if item.TrainTime != nil && row.TrainTime != *item.TrainTime {
			return fmt.Errorf("train_location[%d].train_time patch did not read back: got %d want %d", item.Index, row.TrainTime, *item.TrainTime)
		}
		if item.TrainUnitID != nil && row.TrainUnitID != *item.TrainUnitID {
			return fmt.Errorf("train_location[%d].train_unit_id patch did not read back: got %d want %d", item.Index, row.TrainUnitID, *item.TrainUnitID)
		}
		if item.TrainButton != nil && row.TrainButton != *item.TrainButton {
			return fmt.Errorf("train_location[%d].train_button patch did not read back: got %d want %d", item.Index, row.TrainButton, *item.TrainButton)
		}
		if item.TrainHotkey != nil && row.TrainHotkey != *item.TrainHotkey {
			return fmt.Errorf("train_location[%d].train_hotkey patch did not read back: got %d want %d", item.Index, row.TrainHotkey, *item.TrainHotkey)
		}
	}
	return nil
}

func verifyLinkedBuildingPatches(after UnitSummary, patches []LinkedBuildingPatch) error {
	if len(patches) == 0 {
		return nil
	}
	if after.Building == nil {
		return fmt.Errorf("unit %d in civ %d has no building subrecord", after.Index, after.CivIndex)
	}
	for _, item := range patches {
		if item.Index < 0 || item.Index >= len(after.Building.LinkedBuildings) {
			return fmt.Errorf("linked_building index %d outside row count %d", item.Index, len(after.Building.LinkedBuildings))
		}
		got := after.Building.LinkedBuildings[item.Index]
		if item.UnitID != nil && got.UnitID != *item.UnitID {
			return fmt.Errorf("linked_building[%d].unit_id readback got %d want %d", item.Index, got.UnitID, *item.UnitID)
		}
		if item.X != nil && got.X != *item.X {
			return fmt.Errorf("linked_building[%d].x readback got %f want %f", item.Index, got.X, *item.X)
		}
		if item.Y != nil && got.Y != *item.Y {
			return fmt.Errorf("linked_building[%d].y readback got %f want %f", item.Index, got.Y, *item.Y)
		}
	}
	return nil
}

func verifyTaskPatches(after UnitSummary, patches []TaskPatch) error {
	if len(patches) == 0 {
		return nil
	}
	if after.Action == nil {
		return fmt.Errorf("unit has no action task block after patch")
	}
	for _, item := range patches {
		if item.Index < 0 || item.Index >= len(after.Action.Tasks) {
			return fmt.Errorf("task[%d] patch did not read back: row missing", item.Index)
		}
		row := after.Action.Tasks[item.Index]
		if item.RecordType != nil && row.RecordType != *item.RecordType {
			return fmt.Errorf("task[%d].record_type patch did not read back: got %d want %d", item.Index, row.RecordType, *item.RecordType)
		}
		if item.ID != nil && row.ID != *item.ID {
			return fmt.Errorf("task[%d].id patch did not read back: got %d want %d", item.Index, row.ID, *item.ID)
		}
		if item.IsDefault != nil && row.IsDefault != *item.IsDefault {
			return fmt.Errorf("task[%d].is_default patch did not read back: got %d want %d", item.Index, row.IsDefault, *item.IsDefault)
		}
		if item.ActionType != nil && row.ActionType != *item.ActionType {
			return fmt.Errorf("task[%d].action_type patch did not read back: got %d want %d", item.Index, row.ActionType, *item.ActionType)
		}
		if item.ObjectClass != nil && row.ObjectClass != *item.ObjectClass {
			return fmt.Errorf("task[%d].object_class patch did not read back: got %d want %d", item.Index, row.ObjectClass, *item.ObjectClass)
		}
		if item.ObjectID != nil && row.ObjectID != *item.ObjectID {
			return fmt.Errorf("task[%d].object_id patch did not read back: got %d want %d", item.Index, row.ObjectID, *item.ObjectID)
		}
		if item.TerrainID != nil && row.TerrainID != *item.TerrainID {
			return fmt.Errorf("task[%d].terrain_id patch did not read back: got %d want %d", item.Index, row.TerrainID, *item.TerrainID)
		}
		if item.AttributeTypes != nil {
			if len(row.AttributeTypes) != len(item.AttributeTypes) {
				return fmt.Errorf("task[%d].attribute_types readback length got %d want %d", item.Index, len(row.AttributeTypes), len(item.AttributeTypes))
			}
			for i, want := range item.AttributeTypes {
				if row.AttributeTypes[i] != want {
					return fmt.Errorf("task[%d].attribute_types[%d] patch did not read back: got %d want %d", item.Index, i, row.AttributeTypes[i], want)
				}
			}
		}
		if item.WorkValue1 != nil && row.WorkValue1 != *item.WorkValue1 {
			return fmt.Errorf("task[%d].work_value_1 patch did not read back: got %f want %f", item.Index, row.WorkValue1, *item.WorkValue1)
		}
		if item.WorkValue2 != nil && row.WorkValue2 != *item.WorkValue2 {
			return fmt.Errorf("task[%d].work_value_2 patch did not read back: got %f want %f", item.Index, row.WorkValue2, *item.WorkValue2)
		}
		if item.WorkRange != nil && row.WorkRange != *item.WorkRange {
			return fmt.Errorf("task[%d].work_range patch did not read back: got %f want %f", item.Index, row.WorkRange, *item.WorkRange)
		}
		if item.AutoSearchTargets != nil && row.AutoSearchTargets != *item.AutoSearchTargets {
			return fmt.Errorf("task[%d].auto_search_targets patch did not read back: got %d want %d", item.Index, row.AutoSearchTargets, *item.AutoSearchTargets)
		}
		if item.SearchWaitTime != nil && row.SearchWaitTime != *item.SearchWaitTime {
			return fmt.Errorf("task[%d].search_wait_time patch did not read back: got %f want %f", item.Index, row.SearchWaitTime, *item.SearchWaitTime)
		}
		if item.EnableTargeting != nil && row.EnableTargeting != *item.EnableTargeting {
			return fmt.Errorf("task[%d].enable_targeting patch did not read back: got %d want %d", item.Index, row.EnableTargeting, *item.EnableTargeting)
		}
		if item.CombatLevel != nil && row.CombatLevel != *item.CombatLevel {
			return fmt.Errorf("task[%d].combat_level patch did not read back: got %d want %d", item.Index, row.CombatLevel, *item.CombatLevel)
		}
	}
	return nil
}

func verifySetTaskRows(after UnitSummary, rows *[]TaskRow) error {
	if rows == nil {
		return nil
	}
	if after.Action == nil {
		return fmt.Errorf("unit has no action task block after set_tasks")
	}
	if len(after.Action.Tasks) != len(*rows) {
		return fmt.Errorf("set_tasks readback length got %d want %d", len(after.Action.Tasks), len(*rows))
	}
	for i, want := range *rows {
		got := after.Action.Tasks[i]
		if got.RecordType != want.RecordType ||
			got.ID != want.ID ||
			got.IsDefault != want.IsDefault ||
			got.ActionType != want.ActionType ||
			got.ObjectClass != want.ObjectClass ||
			got.ObjectID != want.ObjectID ||
			got.TerrainID != want.TerrainID ||
			got.WorkValue1 != want.WorkValue1 ||
			got.WorkValue2 != want.WorkValue2 ||
			got.WorkRange != want.WorkRange ||
			got.AutoSearchTargets != want.AutoSearchTargets ||
			got.SearchWaitTime != want.SearchWaitTime ||
			got.EnableTargeting != want.EnableTargeting ||
			got.CombatLevel != want.CombatLevel ||
			!bytes.Equal(got.RawTail, want.RawTail) {
			return fmt.Errorf("set_tasks[%d] readback mismatch", i)
		}
		if len(got.AttributeTypes) != len(want.AttributeTypes) {
			return fmt.Errorf("set_tasks[%d].attribute_types length got %d want %d", i, len(got.AttributeTypes), len(want.AttributeTypes))
		}
		for j, wantAttr := range want.AttributeTypes {
			if got.AttributeTypes[j] != wantAttr {
				return fmt.Errorf("set_tasks[%d].attribute_types[%d] got %d want %d", i, j, got.AttributeTypes[j], wantAttr)
			}
		}
	}
	return nil
}

func requireType50(unit UnitSummary) (*Type50Summary, error) {
	if unit.Type50 == nil {
		return nil, fmt.Errorf("unit %d in civ %d has no Type50 subrecord", unit.Index, unit.CivIndex)
	}
	return unit.Type50, nil
}

func requireAction(unit UnitSummary) (*ActionSummary, error) {
	if unit.Action == nil {
		return nil, fmt.Errorf("unit %d in civ %d has no Action subrecord", unit.Index, unit.CivIndex)
	}
	return unit.Action, nil
}

func requireCreatable(unit UnitSummary) (*CreatableSummary, error) {
	if unit.Creatable == nil {
		return nil, fmt.Errorf("unit %d in civ %d has no Creatable subrecord", unit.Index, unit.CivIndex)
	}
	return unit.Creatable, nil
}

func verifyUnitNeighborCanaries(before, after *Index, civID, unitID int, allowPositionShift bool) error {
	for _, id := range []int{unitID - 1, unitID + 1} {
		if id < 0 || id >= before.Civs[civID].UnitsSize {
			continue
		}
		bu := before.Civs[civID].Units[id]
		au := after.Civs[civID].Units[id]
		if bu.Present != au.Present {
			return fmt.Errorf("neighbor unit %d presence changed in civ %d", id, civID)
		}
		if !bu.Present {
			continue
		}
		if bu.Type != au.Type ||
			bu.ID != au.ID ||
			bu.Name != au.Name ||
			bu.HitPoints != au.HitPoints ||
			bu.StandingGraphic1 != au.StandingGraphic1 ||
			bu.StandingGraphic2 != au.StandingGraphic2 ||
			bu.DyingGraphic != au.DyingGraphic ||
			bu.BloodUnitID != au.BloodUnitID ||
			bu.IconID != au.IconID ||
			bu.Enabled != au.Enabled {
			return fmt.Errorf("neighbor unit %d canary changed in civ %d", id, civID)
		}
		if !allowPositionShift && (bu.RecordStart != au.RecordStart || bu.RecordEnd != au.RecordEnd) {
			return fmt.Errorf("neighbor unit %d position changed in civ %d", id, civID)
		}
	}
	return nil
}

func Inflate(compressed []byte) ([]byte, error) {
	r := flate.NewReader(bytes.NewReader(compressed))
	defer r.Close()
	return io.ReadAll(r)
}

func Deflate(payload []byte) ([]byte, error) {
	var out bytes.Buffer
	w, err := flate.NewWriter(&out, flate.DefaultCompression)
	if err != nil {
		return nil, err
	}
	if _, err := w.Write(payload); err != nil {
		w.Close()
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func RoundTripPayload(payload []byte) error {
	compressed, err := Deflate(payload)
	if err != nil {
		return err
	}
	got, err := Inflate(compressed)
	if err != nil {
		return err
	}
	if !bytes.Equal(payload, got) {
		return errors.New("payload changed after deflate/inflate round trip")
	}
	return nil
}

func Parse(payload []byte) (*Index, error) {
	c := cursor{data: payload}
	idx := &Index{Inflated: len(payload)}

	start := c.off
	versionBytes, err := c.bytes(8)
	if err != nil {
		return nil, err
	}
	idx.Version = string(bytes.TrimRight(versionBytes, "\x00"))
	idx.addSpan("version", start, c.off)

	start = c.off
	terrainRestrictionsSize, err := c.i16()
	if err != nil {
		return nil, err
	}
	terrainsUsed, err := c.i16()
	if err != nil {
		return nil, err
	}
	if terrainRestrictionsSize < 0 || terrainsUsed < 0 {
		return nil, fmt.Errorf("negative terrain counts at %d", start)
	}
	idx.addSpan("terrain_header", start, c.off)

	start = c.off
	if err := c.skip(int(terrainRestrictionsSize) * 4); err != nil {
		return nil, err
	}
	idx.addSpan("terrain_float_pointers", start, c.off)

	start = c.off
	if err := c.skip(int(terrainRestrictionsSize) * 4); err != nil {
		return nil, err
	}
	idx.addSpan("terrain_pass_graphic_pointers", start, c.off)

	start = c.off
	idx.TerrainRestrictions, err = parseTerrainRestrictions(&c, int(terrainRestrictionsSize), int(terrainsUsed))
	if err != nil {
		return nil, err
	}
	idx.addSpan("terrain_restrictions", start, c.off)

	start = c.off
	playerColoursSize, err := c.i16()
	if err != nil {
		return nil, err
	}
	if playerColoursSize < 0 {
		return nil, fmt.Errorf("negative player colour count at %d", start)
	}
	idx.addSpan("player_colours_header", start, c.off)

	start = c.off
	idx.PlayerColours, err = parsePlayerColours(&c, int(playerColoursSize))
	if err != nil {
		return nil, err
	}
	idx.addSpan("player_colours", start, c.off)

	start = c.off
	soundsSize, err := c.i16()
	if err != nil {
		return nil, err
	}
	if soundsSize < 0 {
		return nil, fmt.Errorf("negative sound count at %d", start)
	}
	idx.addSpan("sounds_header", start, c.off)

	for i := 0; i < int(soundsSize); i++ {
		start = c.off
		sound, err := parseSound(&c, i)
		if err != nil {
			return nil, fmt.Errorf("sound[%d]: %w", i, err)
		}
		idx.Sounds = append(idx.Sounds, sound)
		idx.addSpan(fmt.Sprintf("sound[%d]", i), start, c.off)
	}

	start = c.off
	graphicsSize, err := c.i16()
	if err != nil {
		return nil, err
	}
	if graphicsSize < 0 {
		return nil, fmt.Errorf("negative graphic count at %d", start)
	}
	idx.GraphicsSize = int(graphicsSize)
	idx.addSpan("graphics_header", start, c.off)

	start = c.off
	pointers := make([]int32, int(graphicsSize))
	for i := range pointers {
		pointers[i], err = c.i32()
		if err != nil {
			return nil, fmt.Errorf("graphic pointer[%d]: %w", i, err)
		}
	}
	idx.addSpan("graphic_pointers", start, c.off)

	idx.Graphics = make([]Graphic, 0, int(graphicsSize))
	for i, ptr := range pointers {
		if ptr == 0 {
			idx.Graphics = append(idx.Graphics, Graphic{Index: i, Present: false})
			continue
		}
		start = c.off
		graphic, err := parseGraphic(&c, i)
		if err != nil {
			return nil, fmt.Errorf("graphic[%d]: %w", i, err)
		}
		graphic.RecordSpan = Span{Name: fmt.Sprintf("graphic[%d]", i), Start: start, End: c.off}
		idx.Graphics = append(idx.Graphics, graphic)
		idx.addSpan(graphic.RecordSpan.Name, graphic.RecordSpan.Start, graphic.RecordSpan.End)
	}

	start = c.off
	idx.Terrains, err = parseTerrainBlock(&c)
	if err != nil {
		return nil, fmt.Errorf("terrain_block: %w", err)
	}
	idx.addSpan("terrain_block", start, c.off)

	start = c.off
	idx.RandomMaps, err = parseRandomMaps(&c)
	if err != nil {
		return nil, fmt.Errorf("random_maps: %w", err)
	}
	idx.addSpan("random_maps", start, c.off)

	start = c.off
	effectsSize, err := c.i32()
	if err != nil {
		return nil, err
	}
	if effectsSize < 0 {
		return nil, fmt.Errorf("negative effects count at %d", start)
	}
	idx.addSpan("effects_header", start, c.off)
	for i := 0; i < int(effectsSize); i++ {
		start = c.off
		effect, err := parseEffect(&c, i)
		if err != nil {
			return nil, fmt.Errorf("effect[%d]: %w", i, err)
		}
		effect.Span = Span{Name: fmt.Sprintf("effect[%d]", i), Start: start, End: c.off}
		idx.Effects = append(idx.Effects, effect)
		idx.addSpan(effect.Span.Name, effect.Span.Start, effect.Span.End)
	}

	start = c.off
	unitHeadersSize, err := c.i32()
	if err != nil {
		return nil, err
	}
	if unitHeadersSize < 0 {
		return nil, fmt.Errorf("negative unit header count at %d", start)
	}
	idx.addSpan("unit_headers_header", start, c.off)
	idx.UnitHeaders = make([]UnitHeader, 0, int(unitHeadersSize))
	for i := 0; i < int(unitHeadersSize); i++ {
		start = c.off
		header, err := parseUnitHeader(&c, i, idx.versionAtLeast88())
		if err != nil {
			return nil, fmt.Errorf("unit_header[%d]: %w", i, err)
		}
		header.Span = Span{Name: fmt.Sprintf("unit_header[%d]", i), Start: start, End: c.off}
		idx.UnitHeaders = append(idx.UnitHeaders, header)
		idx.addSpan(fmt.Sprintf("unit_header[%d]", i), start, c.off)
	}

	start = c.off
	civsSize, err := c.i16()
	if err != nil {
		return nil, err
	}
	if civsSize < 0 {
		return nil, fmt.Errorf("negative civ count at %d", start)
	}
	idx.addSpan("civs_header", start, c.off)
	idx.Civs = make([]Civ, 0, int(civsSize))
	for i := 0; i < int(civsSize); i++ {
		start = c.off
		civ, err := parseCiv(&c, i, idx.versionAtLeast88())
		if err != nil {
			return nil, fmt.Errorf("civ[%d]: %w", i, err)
		}
		civ.Span = Span{Name: fmt.Sprintf("civ[%d]", i), Start: start, End: c.off}
		idx.Civs = append(idx.Civs, civ)
		idx.addSpan(civ.Span.Name, civ.Span.Start, civ.Span.End)
	}

	start = c.off
	techsSize, err := c.i16()
	if err != nil {
		return nil, err
	}
	if techsSize < 0 {
		return nil, fmt.Errorf("negative tech count at %d", start)
	}
	idx.addSpan("techs_header", start, c.off)
	idx.Techs = make([]Tech, 0, int(techsSize))
	for i := 0; i < int(techsSize); i++ {
		start = c.off
		tech, err := parseTech(&c, i, idx.versionAtLeast88())
		if err != nil {
			return nil, fmt.Errorf("tech[%d]: %w", i, err)
		}
		tech.Span = Span{Name: fmt.Sprintf("tech[%d]", i), Start: start, End: c.off}
		idx.Techs = append(idx.Techs, tech)
		idx.addSpan(tech.Span.Name, tech.Span.Start, tech.Span.End)
	}

	start = c.off
	metrics, err := parseMetrics(&c)
	if err != nil {
		return nil, fmt.Errorf("game metrics: %w", err)
	}
	metrics.Span = Span{Name: "game_metrics", Start: start, End: c.off}
	idx.GameMetrics = metrics
	idx.addSpan(metrics.Span.Name, metrics.Span.Start, metrics.Span.End)

	start = c.off
	techTree, err := parseTechTree(&c, idx.versionAtLeast89())
	if err != nil {
		return nil, fmt.Errorf("tech_tree: %w", err)
	}
	techTree.Span = Span{Name: "tech_tree", Start: start, End: c.off}
	idx.TechTree = techTree
	idx.addSpan(techTree.Span.Name, techTree.Span.Start, techTree.Span.End)

	if c.off != len(payload) {
		return nil, fmt.Errorf("dat parser stopped at %d, payload length %d", c.off, len(payload))
	}
	if err := idx.ValidateCoverage(payload); err != nil {
		return nil, err
	}
	if err := RoundTripPayload(payload); err != nil {
		return nil, fmt.Errorf("raw deflate round trip: %w", err)
	}
	return idx, nil
}

func (idx *Index) Graphic(id int) (Graphic, bool) {
	if id < 0 || id >= len(idx.Graphics) {
		return Graphic{}, false
	}
	g := idx.Graphics[id]
	return g, g.Present
}

func (idx *Index) Effect(id int) (Effect, bool) {
	if id < 0 || id >= len(idx.Effects) {
		return Effect{}, false
	}
	return idx.Effects[id], true
}

func (idx *Index) Tech(id int) (Tech, bool) {
	if id < 0 || id >= len(idx.Techs) {
		return Tech{}, false
	}
	return idx.Techs[id], true
}

func (idx *Index) versionAtLeast88() bool {
	return idx.Version >= "VER 8.8"
}

func (idx *Index) versionAtLeast89() bool {
	return idx.Version >= "VER 8.9"
}

func (idx *Index) PresentGraphics() []Graphic {
	return idx.PresentGraphicsFiltered(GraphicFilter{})
}

func (idx *Index) PresentGraphicsFiltered(filter GraphicFilter) []Graphic {
	var out []Graphic
	for _, g := range idx.Graphics {
		if g.Present && graphicMatchesFilter(g, filter) {
			out = append(out, g)
		}
	}
	return out
}

func (idx *Index) PresentGraphicSummaries() []GraphicSummary {
	return idx.PresentGraphicSummariesFiltered(GraphicFilter{})
}

func (idx *Index) PresentGraphicSummariesFiltered(filter GraphicFilter) []GraphicSummary {
	var out []GraphicSummary
	for _, g := range idx.PresentGraphicsFiltered(filter) {
		out = append(out, g.Summary())
	}
	return out
}

func graphicMatchesFilter(g Graphic, filter GraphicFilter) bool {
	if filter.ParticleOnly && g.ParticleEffectName.Value == "" {
		return false
	}
	if filter.NameContains != "" {
		needle := strings.ToLower(filter.NameContains)
		if !strings.Contains(strings.ToLower(g.Name.Value), needle) &&
			!strings.Contains(strings.ToLower(g.FileName.Value), needle) &&
			!strings.Contains(strings.ToLower(g.ParticleEffectName.Value), needle) {
			return false
		}
	}
	return true
}

func (g Graphic) Summary() GraphicSummary {
	return GraphicSummary{
		Index:              g.Index,
		Name:               g.Name.Value,
		FileName:           g.FileName.Value,
		ParticleEffectName: g.ParticleEffectName.Value,
		SLP:                g.SLP,
		Layer:              g.Layer,
		PlayerColor:        g.PlayerColor,
		SoundID:            g.SoundID,
		WwiseSoundID:       g.WwiseSoundID,
		FrameCount:         g.FrameCount,
		AngleCount:         g.AngleCount,
		DeltaCount:         g.DeltaCount,
		AngleSoundCount:    len(g.AngleSounds),
		SpeedMultiplier:    g.SpeedMultiplier,
		FrameDuration:      g.FrameDuration,
		ReplayDelay:        g.ReplayDelay,
		SequenceType:       g.SequenceType,
		ID:                 g.ID,
		RecordStart:        g.RecordSpan.Start,
		RecordEnd:          g.RecordSpan.End,
	}
}

func verifyNeighborCanaries(before, after *Index, graphicID int) error {
	for _, id := range []int{graphicID - 1, graphicID + 1} {
		if id < 0 || id >= before.GraphicsSize {
			continue
		}
		bg, bok := before.Graphic(id)
		ag, aok := after.Graphic(id)
		if bok != aok {
			return fmt.Errorf("neighbor graphic %d presence changed", id)
		}
		if !bok {
			continue
		}
		if bg.Name.Value != ag.Name.Value ||
			bg.FileName.Value != ag.FileName.Value ||
			bg.ParticleEffectName.Value != ag.ParticleEffectName.Value ||
			bg.SLP != ag.SLP ||
			bg.FrameCount != ag.FrameCount ||
			bg.RecordSpan.Len() != ag.RecordSpan.Len() {
			return fmt.Errorf("neighbor graphic %d canary changed", id)
		}
	}
	return nil
}

func verifyGraphicCreateReadback(created Graphic, recipe GraphicCreateRecipe) error {
	if recipe.Name != nil && created.Name.Value != *recipe.Name {
		return fmt.Errorf("created graphic name did not read back: got %q want %q", created.Name.Value, *recipe.Name)
	}
	if recipe.FileName != nil && created.FileName.Value != *recipe.FileName {
		return fmt.Errorf("created graphic file_name did not read back: got %q want %q", created.FileName.Value, *recipe.FileName)
	}
	if recipe.ParticleEffectName != nil && created.ParticleEffectName.Value != *recipe.ParticleEffectName {
		return fmt.Errorf("created graphic particle_effect_name did not read back: got %q want %q", created.ParticleEffectName.Value, *recipe.ParticleEffectName)
	}
	if recipe.SLP != nil && created.SLP != *recipe.SLP {
		return fmt.Errorf("created graphic slp did not read back: got %d want %d", created.SLP, *recipe.SLP)
	}
	if recipe.FrameCount != nil && created.FrameCount != *recipe.FrameCount {
		return fmt.Errorf("created graphic frame_count did not read back: got %d want %d", created.FrameCount, *recipe.FrameCount)
	}
	if err := verifyGraphicPatchReadback(created, GraphicPatch{
		SLP:               recipe.SLP,
		IsLoaded:          recipe.IsLoaded,
		OldColorFlag:      recipe.OldColorFlag,
		Layer:             recipe.Layer,
		PlayerColor:       recipe.PlayerColor,
		Rainbow:           recipe.Rainbow,
		TransparentSelect: recipe.TransparentSelect,
		Coordinates:       recipe.Coordinates,
		SoundID:           recipe.SoundID,
		WwiseSoundID:      recipe.WwiseSoundID,
		FrameCount:        recipe.FrameCount,
		SpeedMultiplier:   recipe.SpeedMultiplier,
		FrameDuration:     recipe.FrameDuration,
		ReplayDelay:       recipe.ReplayDelay,
		SequenceType:      recipe.SequenceType,
		MirroringMode:     recipe.MirroringMode,
		EditorFlag:        recipe.EditorFlag,
		SetDeltas:         recipe.SetDeltas,
		SetAngleSounds:    recipe.SetAngleSounds,
	}); err != nil {
		return fmt.Errorf("created graphic %w", err)
	}
	return nil
}

func verifyGraphicAppendCanaries(before, after *Index, fromGraphicID int) error {
	if len(before.Graphics)+1 != len(after.Graphics) {
		return fmt.Errorf("graphic table length changed unexpectedly: %d -> %d", len(before.Graphics), len(after.Graphics))
	}
	for _, id := range []int{0, fromGraphicID, before.GraphicsSize - 1} {
		if id < 0 || id >= before.GraphicsSize {
			continue
		}
		bg, bok := before.Graphic(id)
		ag, aok := after.Graphic(id)
		if bok != aok {
			return fmt.Errorf("existing graphic %d presence changed", id)
		}
		if !bok {
			continue
		}
		if bg.Name.Value != ag.Name.Value ||
			bg.FileName.Value != ag.FileName.Value ||
			bg.ParticleEffectName.Value != ag.ParticleEffectName.Value ||
			bg.SLP != ag.SLP ||
			bg.FrameCount != ag.FrameCount ||
			bg.AngleCount != ag.AngleCount ||
			bg.DeltaCount != ag.DeltaCount ||
			bg.ID != ag.ID ||
			bg.RecordSpan.Len() != ag.RecordSpan.Len() {
			return fmt.Errorf("existing graphic %d canary changed", id)
		}
	}
	return nil
}

func changedGraphicCreateFields(recipe GraphicCreateRecipe) []string {
	var changed []string
	if recipe.Name != nil {
		changed = append(changed, "name")
	}
	if recipe.FileName != nil {
		changed = append(changed, "file_name")
	}
	if recipe.ParticleEffectName != nil {
		changed = append(changed, "particle_effect_name")
	}
	if recipe.SLP != nil {
		changed = append(changed, "slp")
	}
	if recipe.FrameCount != nil {
		changed = append(changed, "frame_count")
	}
	if recipe.SetDeltas != nil {
		changed = append(changed, "set_deltas")
	}
	if recipe.SetAngleSounds != nil {
		changed = append(changed, "set_angle_sounds")
	}
	return changed
}

func changedUnitCreateFields(recipe UnitCreateRecipe) []string {
	var changed []string
	if recipe.Class != nil {
		changed = append(changed, "class")
	}
	if recipe.HitPoints != nil {
		changed = append(changed, "hit_points")
	}
	if recipe.LineOfSight != nil {
		changed = append(changed, "line_of_sight")
	}
	if recipe.MovementType != nil {
		changed = append(changed, "movement_type")
	}
	if recipe.StandingGraphic1 != nil {
		changed = append(changed, "standing_graphic_1")
	}
	if recipe.StandingGraphic2 != nil {
		changed = append(changed, "standing_graphic_2")
	}
	if recipe.DyingGraphic != nil {
		changed = append(changed, "dying_graphic")
	}
	if recipe.BloodUnitID != nil {
		changed = append(changed, "blood_unit_id")
	}
	if recipe.IconID != nil {
		changed = append(changed, "icon_id")
	}
	if recipe.Enabled != nil {
		changed = append(changed, "enabled")
	}
	for _, item := range recipe.Attributes {
		changed = append(changed, fmt.Sprintf("attribute[%d]", item.Index))
	}
	for _, item := range recipe.DamageGraphics {
		changed = append(changed, fmt.Sprintf("damage_graphic[%d]", item.Index))
	}
	if recipe.SetDamageGraphics != nil {
		changed = append(changed, "set_damage_graphics")
	}
	if recipe.Type50ProjectileUnitID != nil {
		changed = append(changed, "type50_projectile_unit_id")
	}
	if recipe.Type50MaxRange != nil {
		changed = append(changed, "type50_max_range")
	}
	if recipe.Type50BlastWidth != nil {
		changed = append(changed, "type50_blast_width")
	}
	if recipe.Type50AttackGraphic != nil {
		changed = append(changed, "type50_attack_graphic")
	}
	if recipe.Type50BlastDamage != nil {
		changed = append(changed, "type50_blast_damage")
	}
	for _, item := range recipe.Type50Attacks {
		changed = append(changed, fmt.Sprintf("type50_attack[%d]", item.Index))
	}
	for _, item := range recipe.Type50Armours {
		changed = append(changed, fmt.Sprintf("type50_armour[%d]", item.Index))
	}
	if recipe.SetType50Attacks != nil {
		changed = append(changed, "set_type50_attacks")
	}
	if recipe.SetType50Armours != nil {
		changed = append(changed, "set_type50_armours")
	}
	if recipe.TrainTime0 != nil {
		changed = append(changed, "train_time_0")
	}
	if recipe.TrainUnitID0 != nil {
		changed = append(changed, "train_unit_id_0")
	}
	if recipe.TrainButtonID0 != nil {
		changed = append(changed, "train_button_id_0")
	}
	if recipe.TrainHotKeyID0 != nil {
		changed = append(changed, "train_hotkey_id_0")
	}
	if recipe.CreatableButtonIconID != nil {
		changed = append(changed, "creatable_button_icon_id")
	}
	if recipe.CreatableButtonHotkeyAction != nil {
		changed = append(changed, "creatable_button_hotkey_action")
	}
	for _, item := range recipe.Costs {
		changed = append(changed, fmt.Sprintf("cost[%d]", item.Index))
	}
	for _, item := range recipe.TrainLocations {
		changed = append(changed, fmt.Sprintf("train_location[%d]", item.Index))
	}
	if recipe.SetTrainLocations != nil {
		changed = append(changed, "set_train_locations")
	}
	if recipe.SetDropSites != nil {
		changed = append(changed, "set_drop_sites")
	}
	for _, item := range recipe.LinkedBuildings {
		changed = append(changed, fmt.Sprintf("linked_building[%d]", item.Index))
	}
	for _, item := range recipe.Tasks {
		changed = append(changed, fmt.Sprintf("task[%d]", item.Index))
	}
	if recipe.SetTasks != nil {
		changed = append(changed, "set_tasks")
	}
	return changed
}

func (idx *Index) ValidateCoverage(payload []byte) error {
	if len(idx.Spans) == 0 {
		if len(payload) == 0 {
			return nil
		}
		return errors.New("no spans")
	}
	expected := 0
	for _, span := range idx.Spans {
		if span.Start != expected {
			return fmt.Errorf("span coverage gap/overlap before %s: got start %d, expected %d", span.Name, span.Start, expected)
		}
		if span.End < span.Start {
			return fmt.Errorf("span %s has negative length", span.Name)
		}
		if span.End > len(payload) {
			return fmt.Errorf("span %s ends past payload: %d > %d", span.Name, span.End, len(payload))
		}
		expected = span.End
	}
	if expected != len(payload) {
		return fmt.Errorf("span coverage stops at %d, payload length %d", expected, len(payload))
	}
	return nil
}

func (idx *Index) addSpan(name string, start, end int) {
	if start == end {
		return
	}
	idx.Spans = append(idx.Spans, Span{Name: name, Start: start, End: end})
}

func (idx *Index) spanByName(name string) (Span, bool) {
	for _, span := range idx.Spans {
		if span.Name == name {
			return span, true
		}
	}
	return Span{}, false
}

func skipSound(c *cursor) error {
	if _, err := c.i16(); err != nil {
		return err
	}
	if _, err := c.i16(); err != nil {
		return err
	}
	items, err := c.i16()
	if err != nil {
		return err
	}
	if items < 0 {
		return fmt.Errorf("negative sound item count at %d", c.off-2)
	}
	if _, err := c.i32(); err != nil {
		return err
	}
	if _, err := c.i16(); err != nil {
		return err
	}
	for i := 0; i < int(items); i++ {
		if _, err := c.debugString("sound_item"); err != nil {
			return fmt.Errorf("item[%d] debug string: %w", i, err)
		}
		if err := c.skip(4 + 2 + 2 + 2); err != nil {
			return fmt.Errorf("item[%d] fields: %w", i, err)
		}
	}
	return nil
}

func parseTerrainBlock(c *cursor) ([]Terrain, error) {
	if err := c.skip(24); err != nil {
		return nil, err
	}
	if err := c.skip(tileTypeCount * 6); err != nil {
		return nil, err
	}
	if err := c.skip(2); err != nil {
		return nil, err
	}
	terrains := make([]Terrain, 0, terrainCount)
	for i := 0; i < terrainCount; i++ {
		terrain, err := parseTerrain(c, i)
		if err != nil {
			return nil, fmt.Errorf("terrain[%d]: %w", i, err)
		}
		terrains = append(terrains, terrain)
	}
	if err := c.skip(24 + 28 + 8 + 3); err != nil {
		return nil, err
	}
	return terrains, nil
}

func parseTerrain(c *cursor, index int) (Terrain, error) {
	start := c.off
	terrain := Terrain{Index: index}
	if err := c.skip(4 + 4); err != nil {
		return terrain, err
	}
	var err error
	if terrain.Name, err = c.debugString("terrain_name"); err != nil {
		return terrain, err
	}
	if terrain.Name2, err = c.debugString("terrain_name_2"); err != nil {
		return terrain, err
	}
	if err := c.skip(4 + 4 + 4 + 4 + 4 + 4 + 4); err != nil {
		return terrain, err
	}
	if terrain.OverlayMaskName, err = c.debugString("terrain_overlay_mask_name"); err != nil {
		return terrain, err
	}
	if err := c.skip(3 + 2 + 3 + 2 + 2 + 4 + 4 + 2 + 2 + 4 + 1 + 1); err != nil {
		return terrain, err
	}
	if err := c.skip(tileTypeCount * 6); err != nil {
		return terrain, err
	}
	if err := c.skip(2 + 4 + terrainUnitsSize*2 + terrainUnitsSize*2 + terrainUnitsSize*2 + terrainUnitsSize + 2 + 2); err != nil {
		return terrain, err
	}
	terrain.Span = Span{Name: fmt.Sprintf("terrain[%d]", index), Start: start, End: c.off}
	return terrain, nil
}

func parseRandomMaps(c *cursor) ([]RandomMapInfo, error) {
	count, err := c.u32()
	if err != nil {
		return nil, err
	}
	if err := c.skip(4); err != nil {
		return nil, err
	}
	out := make([]RandomMapInfo, 0, int(count)*2)
	for pass := 0; pass < 2; pass++ {
		for i := 0; i < int(count); i++ {
			info, err := parseMapInfo(c, pass+1, i)
			if err != nil {
				return nil, fmt.Errorf("map_info_%d[%d]: %w", pass+1, i, err)
			}
			out = append(out, info)
		}
	}
	return out, nil
}

func parseMapInfo(c *cursor, pass, index int) (RandomMapInfo, error) {
	start := c.off
	info := RandomMapInfo{Pass: pass, Index: index}
	if err := c.skip(40); err != nil {
		return info, err
	}
	lands, err := c.u32()
	if err != nil {
		return info, err
	}
	info.Lands = lands
	landStart := c.off
	if err := c.skip(4 + int(lands)*44); err != nil {
		return info, err
	}
	info.LandSpan = Span{Name: fmt.Sprintf("random_map[%d][%d].lands", pass, index), Start: landStart, End: c.off}
	terrains, err := c.u32()
	if err != nil {
		return info, err
	}
	info.Terrains = terrains
	terrainStart := c.off
	if err := c.skip(4 + int(terrains)*24); err != nil {
		return info, err
	}
	info.TerrainSpan = Span{Name: fmt.Sprintf("random_map[%d][%d].terrains", pass, index), Start: terrainStart, End: c.off}
	units, err := c.u32()
	if err != nil {
		return info, err
	}
	info.Units = units
	unitStart := c.off
	if err := c.skip(4 + int(units)*44); err != nil {
		return info, err
	}
	info.UnitSpan = Span{Name: fmt.Sprintf("random_map[%d][%d].units", pass, index), Start: unitStart, End: c.off}
	elevations, err := c.u32()
	if err != nil {
		return info, err
	}
	info.Elevations = elevations
	elevationStart := c.off
	if err := c.skip(4 + int(elevations)*24); err != nil {
		return info, err
	}
	info.ElevationSpan = Span{Name: fmt.Sprintf("random_map[%d][%d].elevations", pass, index), Start: elevationStart, End: c.off}
	info.Span = Span{Name: fmt.Sprintf("random_map[%d][%d]", pass, index), Start: start, End: c.off}
	return info, nil
}

func parseEffect(c *cursor, index int) (Effect, error) {
	name, err := c.debugString("effect_name")
	if err != nil {
		return Effect{}, err
	}
	commandCount, err := c.i16()
	if err != nil {
		return Effect{}, err
	}
	if commandCount < 0 {
		return Effect{}, fmt.Errorf("negative effect command count at %d", c.off-2)
	}
	effect := Effect{Index: index, Name: name.Value, CommandSize: int(commandCount), Commands: make([]EffectCommand, 0, int(commandCount))}
	for i := 0; i < int(commandCount); i++ {
		start := c.off
		commandType, err := c.u8()
		if err != nil {
			return Effect{}, fmt.Errorf("command[%d] type: %w", i, err)
		}
		a, err := c.i16()
		if err != nil {
			return Effect{}, fmt.Errorf("command[%d] a: %w", i, err)
		}
		b, err := c.i16()
		if err != nil {
			return Effect{}, fmt.Errorf("command[%d] b: %w", i, err)
		}
		cc, err := c.i16()
		if err != nil {
			return Effect{}, fmt.Errorf("command[%d] c: %w", i, err)
		}
		d, err := c.f32()
		if err != nil {
			return Effect{}, fmt.Errorf("command[%d] d: %w", i, err)
		}
		command := EffectCommand{
			Index: i,
			Type:  commandType,
			A:     a,
			B:     b,
			C:     cc,
			D:     d,
			Span:  Span{Name: fmt.Sprintf("effect[%d].command[%d]", index, i), Start: start, End: c.off},
		}
		command.Semantic = InterpretEffectCommand(command)
		effect.Commands = append(effect.Commands, command)
	}
	return effect, nil
}

func parseUnitHeader(c *cursor, index int, versionAtLeast88 bool) (UnitHeader, error) {
	header := UnitHeader{Index: index}
	exists, err := c.u8()
	if err != nil {
		return header, err
	}
	header.Exists = exists != 0
	if exists == 0 {
		return header, nil
	}
	taskCount, err := c.i16()
	if err != nil {
		return header, err
	}
	if taskCount < 0 {
		return header, fmt.Errorf("negative task count at %d", c.off-2)
	}
	header.TaskCount = int(taskCount)
	header.Tasks = make([]TaskSummary, 0, int(taskCount))
	for i := 0; i < int(taskCount); i++ {
		task, err := parseTask(c, i, versionAtLeast88)
		if err != nil {
			return header, fmt.Errorf("task[%d]: %w", i, err)
		}
		header.Tasks = append(header.Tasks, task)
	}
	return header, nil
}

func parseCiv(c *cursor, index int, versionAtLeast88 bool) (Civ, error) {
	if !versionAtLeast88 {
		return Civ{}, errors.New("unit indexing currently supports VER 8.8+ civ/unit records")
	}
	start := c.off
	civType, err := c.u8()
	if err != nil {
		return Civ{}, err
	}
	typeSpan := Span{Name: "type", Start: start, End: c.off}
	name, err := c.debugString("civ_name")
	if err != nil {
		return Civ{}, err
	}
	start = c.off
	resourcesSize, err := c.i16()
	if err != nil {
		return Civ{}, err
	}
	if resourcesSize < 0 {
		return Civ{}, fmt.Errorf("negative resources count at %d", c.off-2)
	}
	resourcesSizeSpan := Span{Name: "resources_size", Start: start, End: c.off}
	start = c.off
	techTreeID, err := c.i16()
	if err != nil {
		return Civ{}, err
	}
	techTreeIDSpan := Span{Name: "tech_tree_id", Start: start, End: c.off}
	start = c.off
	teamBonusID, err := c.i16()
	if err != nil {
		return Civ{}, err
	}
	teamBonusIDSpan := Span{Name: "team_bonus_id", Start: start, End: c.off}
	start = c.off
	resources := make([]float32, int(resourcesSize))
	for i := range resources {
		resources[i], err = c.f32()
		if err != nil {
			return Civ{}, fmt.Errorf("resource[%d]: %w", i, err)
		}
	}
	resourcesSpan := Span{Name: "resources", Start: start, End: c.off}
	start = c.off
	iconSet, err := c.u8()
	if err != nil {
		return Civ{}, err
	}
	iconSetSpan := Span{Name: "icon_set", Start: start, End: c.off}
	start = c.off
	unitsSize, err := c.i16()
	if err != nil {
		return Civ{}, err
	}
	if unitsSize < 0 {
		return Civ{}, fmt.Errorf("negative units count at %d", c.off-2)
	}
	unitsSizeSpan := Span{Name: "units_size", Start: start, End: c.off}
	start = c.off
	pointers := make([]int32, int(unitsSize))
	for i := range pointers {
		pointers[i], err = c.i32()
		if err != nil {
			return Civ{}, fmt.Errorf("unit pointer[%d]: %w", i, err)
		}
	}
	unitPointersSpan := Span{Name: "unit_pointers", Start: start, End: c.off}
	civ := Civ{
		Index:         index,
		Type:          civType,
		Name:          name.Value,
		ResourcesSize: int(resourcesSize),
		TechTreeID:    techTreeID,
		TeamBonusID:   teamBonusID,
		Resources:     resources,
		IconSet:       iconSet,
		UnitsSize:     int(unitsSize),
		Units:         make([]UnitSummary, 0, int(unitsSize)),
		FieldSpans: CivFieldSpans{
			Type:          typeSpan,
			Name:          name.Span,
			ResourcesSize: resourcesSizeSpan,
			TechTreeID:    techTreeIDSpan,
			TeamBonusID:   teamBonusIDSpan,
			Resources:     resourcesSpan,
			IconSet:       iconSetSpan,
			UnitsSize:     unitsSizeSpan,
			UnitPointers:  unitPointersSpan,
		},
	}
	for i, ptr := range pointers {
		if ptr == 0 {
			civ.Units = append(civ.Units, UnitSummary{CivIndex: index, Index: i, Present: false})
			continue
		}
		start := c.off
		unit, err := parseUnit(c, index, i, versionAtLeast88)
		if err != nil {
			return Civ{}, fmt.Errorf("unit[%d]: %w", i, err)
		}
		unit.RecordStart = start
		unit.RecordEnd = c.off
		unit.RecordLength = unit.RecordEnd - unit.RecordStart
		civ.Units = append(civ.Units, unit)
	}
	return civ, nil
}

func parseTech(c *cursor, index int, versionAtLeast88 bool) (Tech, error) {
	if !versionAtLeast88 {
		return Tech{}, errors.New("tech indexing currently supports VER 8.8+ tech records")
	}
	tech := Tech{Index: index}
	var err error
	tech.RequiredTechs, err = readI16List(c, 6)
	if err != nil {
		return Tech{}, fmt.Errorf("required techs: %w", err)
	}
	tech.ResourceCosts = make([]ResearchResourceCost, 0, 3)
	for i := 0; i < 3; i++ {
		cost, err := parseResearchResourceCost(c)
		if err != nil {
			return Tech{}, fmt.Errorf("resource cost[%d]: %w", i, err)
		}
		tech.ResourceCosts = append(tech.ResourceCosts, cost)
	}
	if tech.RequiredTechCount, err = c.i16(); err != nil {
		return Tech{}, err
	}
	if tech.Civ, err = c.i16(); err != nil {
		return Tech{}, err
	}
	if tech.FullTechMode, err = c.i16(); err != nil {
		return Tech{}, err
	}
	if tech.LanguageDLLName, err = c.i32(); err != nil {
		return Tech{}, err
	}
	if tech.LanguageDLLDescription, err = c.i32(); err != nil {
		return Tech{}, err
	}
	if tech.EffectID, err = c.i16(); err != nil {
		return Tech{}, err
	}
	if tech.Type, err = c.i16(); err != nil {
		return Tech{}, err
	}
	if tech.IconID, err = c.i16(); err != nil {
		return Tech{}, err
	}
	if tech.LanguageDLLHelp, err = c.i32(); err != nil {
		return Tech{}, err
	}
	if tech.LanguageDLLTechTree, err = c.i32(); err != nil {
		return Tech{}, err
	}
	name, err := c.debugString("tech_name")
	if err != nil {
		return Tech{}, fmt.Errorf("name: %w", err)
	}
	tech.Name = name.Value
	if tech.Repeatable, err = c.u8(); err != nil {
		return Tech{}, err
	}
	locationCount, err := c.i16()
	if err != nil {
		return Tech{}, err
	}
	if locationCount < 0 {
		return Tech{}, fmt.Errorf("negative research location count at %d", c.off-2)
	}
	tech.ResearchLocations = make([]ResearchLocation, 0, int(locationCount))
	for i := 0; i < int(locationCount); i++ {
		location, err := parseResearchLocation(c)
		if err != nil {
			return Tech{}, fmt.Errorf("research location[%d]: %w", i, err)
		}
		tech.ResearchLocations = append(tech.ResearchLocations, location)
	}
	return tech, nil
}

func parseResearchResourceCost(c *cursor) (ResearchResourceCost, error) {
	var cost ResearchResourceCost
	var err error
	if cost.Type, err = c.i16(); err != nil {
		return ResearchResourceCost{}, err
	}
	if cost.Amount, err = c.i16(); err != nil {
		return ResearchResourceCost{}, err
	}
	if cost.Flag, err = c.u8(); err != nil {
		return ResearchResourceCost{}, err
	}
	return cost, nil
}

func parseResearchLocation(c *cursor) (ResearchLocation, error) {
	var location ResearchLocation
	var err error
	if location.LocationID, err = c.i16(); err != nil {
		return ResearchLocation{}, err
	}
	if location.ResearchTime, err = c.i16(); err != nil {
		return ResearchLocation{}, err
	}
	if location.ButtonID, err = c.u8(); err != nil {
		return ResearchLocation{}, err
	}
	if location.HotKeyID, err = c.i32(); err != nil {
		return ResearchLocation{}, err
	}
	return location, nil
}

func parseMetrics(c *cursor) (Metrics, error) {
	var metrics Metrics
	var err error
	if metrics.TimeSlice, err = c.i32(); err != nil {
		return Metrics{}, err
	}
	if metrics.UnitKillRate, err = c.i32(); err != nil {
		return Metrics{}, err
	}
	if metrics.UnitKillTotal, err = c.i32(); err != nil {
		return Metrics{}, err
	}
	if metrics.UnitHitPointRate, err = c.i32(); err != nil {
		return Metrics{}, err
	}
	if metrics.UnitHitPointTotal, err = c.i32(); err != nil {
		return Metrics{}, err
	}
	if metrics.RazingKillRate, err = c.i32(); err != nil {
		return Metrics{}, err
	}
	if metrics.RazingKillTotal, err = c.i32(); err != nil {
		return Metrics{}, err
	}
	return metrics, nil
}

func parseTechTree(c *cursor, versionAtLeast89 bool) (TechTree, error) {
	start := c.off
	ageCount, err := c.u8()
	if err != nil {
		return TechTree{}, err
	}
	buildingCount, err := c.u8()
	if err != nil {
		return TechTree{}, err
	}
	unitCount := 0
	if versionAtLeast89 {
		u, err := c.u16()
		if err != nil {
			return TechTree{}, err
		}
		unitCount = int(u)
	} else {
		u, err := c.u8()
		if err != nil {
			return TechTree{}, err
		}
		unitCount = int(u)
	}
	researchCount, err := c.u8()
	if err != nil {
		return TechTree{}, err
	}
	totalGroups, err := c.i32()
	if err != nil {
		return TechTree{}, err
	}
	tree := TechTree{
		AgeCount:            int(ageCount),
		BuildingCount:       int(buildingCount),
		UnitCount:           unitCount,
		ResearchCount:       int(researchCount),
		TotalUnitTechGroups: totalGroups,
	}
	for i := 0; i < tree.AgeCount; i++ {
		age, err := parseTechTreeAge(c, i)
		if err != nil {
			return TechTree{}, fmt.Errorf("age[%d]: %w", i, err)
		}
		tree.Ages = append(tree.Ages, age)
	}
	for i := 0; i < tree.BuildingCount; i++ {
		connection, err := parseBuildingConnection(c, i)
		if err != nil {
			return TechTree{}, fmt.Errorf("building_connection[%d]: %w", i, err)
		}
		tree.BuildingConnections = append(tree.BuildingConnections, connection)
	}
	for i := 0; i < tree.UnitCount; i++ {
		connection, err := parseUnitConnection(c, i)
		if err != nil {
			return TechTree{}, fmt.Errorf("unit_connection[%d]: %w", i, err)
		}
		tree.UnitConnections = append(tree.UnitConnections, connection)
	}
	for i := 0; i < tree.ResearchCount; i++ {
		connection, err := parseResearchConnection(c, i)
		if err != nil {
			return TechTree{}, fmt.Errorf("research_connection[%d]: %w", i, err)
		}
		tree.ResearchConnections = append(tree.ResearchConnections, connection)
	}
	tree.Span = Span{Name: "tech_tree", Start: start, End: c.off}
	return tree, nil
}

func parseTechTreeAge(c *cursor, index int) (TechTreeAge, error) {
	start := c.off
	id, err := c.i32()
	if err != nil {
		return TechTreeAge{}, err
	}
	status, err := c.u8()
	if err != nil {
		return TechTreeAge{}, err
	}
	buildings, err := readCountedI32ListU8(c)
	if err != nil {
		return TechTreeAge{}, fmt.Errorf("buildings: %w", err)
	}
	units, err := readCountedI32ListU8(c)
	if err != nil {
		return TechTreeAge{}, fmt.Errorf("units: %w", err)
	}
	techs, err := readCountedI32ListU8(c)
	if err != nil {
		return TechTreeAge{}, fmt.Errorf("techs: %w", err)
	}
	common, err := parseTechTreeCommon(c)
	if err != nil {
		return TechTreeAge{}, fmt.Errorf("common: %w", err)
	}
	levels, err := c.u8()
	if err != nil {
		return TechTreeAge{}, err
	}
	buildingsPerZone, err := readU8List(c, 10)
	if err != nil {
		return TechTreeAge{}, err
	}
	groupLengthPerZone, err := readU8List(c, 10)
	if err != nil {
		return TechTreeAge{}, err
	}
	maxAgeLength, err := c.u8()
	if err != nil {
		return TechTreeAge{}, err
	}
	lineMode, err := c.i32()
	if err != nil {
		return TechTreeAge{}, err
	}
	return TechTreeAge{
		Index:              index,
		ID:                 id,
		Status:             status,
		Buildings:          buildings,
		Units:              units,
		Techs:              techs,
		Common:             common,
		NumBuildingLevels:  levels,
		BuildingsPerZone:   buildingsPerZone,
		GroupLengthPerZone: groupLengthPerZone,
		MaxAgeLength:       maxAgeLength,
		LineMode:           lineMode,
		Span:               Span{Name: fmt.Sprintf("tech_tree.age[%d]", index), Start: start, End: c.off},
	}, nil
}

func parseBuildingConnection(c *cursor, index int) (BuildingConnection, error) {
	start := c.off
	id, status, err := parseConnectionHeader(c)
	if err != nil {
		return BuildingConnection{}, err
	}
	buildings, err := readCountedI32ListU8(c)
	if err != nil {
		return BuildingConnection{}, fmt.Errorf("buildings: %w", err)
	}
	units, err := readCountedI32ListU8(c)
	if err != nil {
		return BuildingConnection{}, fmt.Errorf("units: %w", err)
	}
	techs, err := readCountedI32ListU8(c)
	if err != nil {
		return BuildingConnection{}, fmt.Errorf("techs: %w", err)
	}
	common, err := parseTechTreeCommon(c)
	if err != nil {
		return BuildingConnection{}, fmt.Errorf("common: %w", err)
	}
	location, err := c.u8()
	if err != nil {
		return BuildingConnection{}, err
	}
	total, err := readU8List(c, 5)
	if err != nil {
		return BuildingConnection{}, err
	}
	first, err := readU8List(c, 5)
	if err != nil {
		return BuildingConnection{}, err
	}
	lineMode, err := c.i32()
	if err != nil {
		return BuildingConnection{}, err
	}
	enablingResearch, err := c.i32()
	if err != nil {
		return BuildingConnection{}, err
	}
	return BuildingConnection{
		Index:            index,
		ID:               id,
		Status:           status,
		Buildings:        buildings,
		Units:            units,
		Techs:            techs,
		Common:           common,
		LocationInAge:    location,
		UnitsTechsTotal:  total,
		UnitsTechsFirst:  first,
		LineMode:         lineMode,
		EnablingResearch: enablingResearch,
		Span:             Span{Name: fmt.Sprintf("tech_tree.building_connection[%d]", index), Start: start, End: c.off},
	}, nil
}

func parseUnitConnection(c *cursor, index int) (UnitConnection, error) {
	start := c.off
	id, status, err := parseConnectionHeader(c)
	if err != nil {
		return UnitConnection{}, err
	}
	upperBuilding, err := c.i32()
	if err != nil {
		return UnitConnection{}, err
	}
	common, err := parseTechTreeCommon(c)
	if err != nil {
		return UnitConnection{}, fmt.Errorf("common: %w", err)
	}
	verticalLine, err := c.i32()
	if err != nil {
		return UnitConnection{}, err
	}
	units, err := readCountedI32ListU8(c)
	if err != nil {
		return UnitConnection{}, fmt.Errorf("units: %w", err)
	}
	locationInAge, err := c.i32()
	if err != nil {
		return UnitConnection{}, err
	}
	requiredResearch, err := c.i32()
	if err != nil {
		return UnitConnection{}, err
	}
	lineMode, err := c.i32()
	if err != nil {
		return UnitConnection{}, err
	}
	enablingResearch, err := c.i32()
	if err != nil {
		return UnitConnection{}, err
	}
	return UnitConnection{
		Index:            index,
		ID:               id,
		Status:           status,
		UpperBuilding:    upperBuilding,
		Common:           common,
		VerticalLine:     verticalLine,
		Units:            units,
		LocationInAge:    locationInAge,
		RequiredResearch: requiredResearch,
		LineMode:         lineMode,
		EnablingResearch: enablingResearch,
		Span:             Span{Name: fmt.Sprintf("tech_tree.unit_connection[%d]", index), Start: start, End: c.off},
	}, nil
}

func parseResearchConnection(c *cursor, index int) (ResearchConnection, error) {
	start := c.off
	id, status, err := parseConnectionHeader(c)
	if err != nil {
		return ResearchConnection{}, err
	}
	upperBuilding, err := c.i32()
	if err != nil {
		return ResearchConnection{}, err
	}
	buildings, err := readCountedI32ListU8(c)
	if err != nil {
		return ResearchConnection{}, fmt.Errorf("buildings: %w", err)
	}
	units, err := readCountedI32ListU8(c)
	if err != nil {
		return ResearchConnection{}, fmt.Errorf("units: %w", err)
	}
	techs, err := readCountedI32ListU8(c)
	if err != nil {
		return ResearchConnection{}, fmt.Errorf("techs: %w", err)
	}
	common, err := parseTechTreeCommon(c)
	if err != nil {
		return ResearchConnection{}, fmt.Errorf("common: %w", err)
	}
	verticalLine, err := c.i32()
	if err != nil {
		return ResearchConnection{}, err
	}
	locationInAge, err := c.i32()
	if err != nil {
		return ResearchConnection{}, err
	}
	lineMode, err := c.i32()
	if err != nil {
		return ResearchConnection{}, err
	}
	return ResearchConnection{
		Index:         index,
		ID:            id,
		Status:        status,
		UpperBuilding: upperBuilding,
		Buildings:     buildings,
		Units:         units,
		Techs:         techs,
		Common:        common,
		VerticalLine:  verticalLine,
		LocationInAge: locationInAge,
		LineMode:      lineMode,
		Span:          Span{Name: fmt.Sprintf("tech_tree.research_connection[%d]", index), Start: start, End: c.off},
	}, nil
}

func parseConnectionHeader(c *cursor) (int32, uint8, error) {
	id, err := c.i32()
	if err != nil {
		return 0, 0, err
	}
	status, err := c.u8()
	if err != nil {
		return 0, 0, err
	}
	return id, status, nil
}

func parseTechTreeCommon(c *cursor) (TechTreeCommon, error) {
	slotsUsed, err := c.i32()
	if err != nil {
		return TechTreeCommon{}, err
	}
	unitResearch, err := readI32List(c, 10)
	if err != nil {
		return TechTreeCommon{}, err
	}
	mode, err := readI32List(c, 10)
	if err != nil {
		return TechTreeCommon{}, err
	}
	return TechTreeCommon{SlotsUsed: slotsUsed, UnitResearch: unitResearch, Mode: mode}, nil
}

func readCountedI32ListU8(c *cursor) ([]int32, error) {
	count, err := c.u8()
	if err != nil {
		return nil, err
	}
	return readI32List(c, int(count))
}

func readI16List(c *cursor, count int) ([]int16, error) {
	out := make([]int16, 0, count)
	for i := 0; i < count; i++ {
		value, err := c.i16()
		if err != nil {
			return nil, err
		}
		out = append(out, value)
	}
	return out, nil
}

func readI32List(c *cursor, count int) ([]int32, error) {
	out := make([]int32, 0, count)
	for i := 0; i < count; i++ {
		value, err := c.i32()
		if err != nil {
			return nil, err
		}
		out = append(out, value)
	}
	return out, nil
}

func readU8List(c *cursor, count int) ([]uint8, error) {
	out := make([]uint8, 0, count)
	for i := 0; i < count; i++ {
		value, err := c.u8()
		if err != nil {
			return nil, err
		}
		out = append(out, value)
	}
	return out, nil
}

func parseTerrainRestrictions(c *cursor, restrictionCount, terrainsUsed int) ([]TerrainRestriction, error) {
	out := make([]TerrainRestriction, 0, restrictionCount)
	for i := 0; i < restrictionCount; i++ {
		start := c.off
		restriction := TerrainRestriction{
			Index:       i,
			TerrainRows: make([]TerrainRestrictionTerrain, 0, terrainsUsed),
		}
		for terrain := 0; terrain < terrainsUsed; terrain++ {
			rowStart := c.off
			passStart := c.off
			passability, err := c.f32()
			if err != nil {
				return nil, err
			}
			passabilitySpan := Span{Name: fmt.Sprintf("terrain_restriction[%d].terrain[%d].passability", i, terrain), Start: passStart, End: c.off}
			graphicStart := c.off
			if err := c.skip(16); err != nil {
				return nil, err
			}
			restriction.TerrainRows = append(restriction.TerrainRows, TerrainRestrictionTerrain{
				Index:              terrain,
				Passability:        passability,
				PassGraphicRawSize: 16,
				Span:               Span{Name: fmt.Sprintf("terrain_restriction[%d].terrain[%d]", i, terrain), Start: rowStart, End: c.off},
				PassabilitySpan:    passabilitySpan,
				PassGraphicSpan:    Span{Name: fmt.Sprintf("terrain_restriction[%d].terrain[%d].pass_graphic", i, terrain), Start: graphicStart, End: c.off},
			})
		}
		restriction.Span = Span{Name: fmt.Sprintf("terrain_restriction[%d]", i), Start: start, End: c.off}
		out = append(out, restriction)
	}
	return out, nil
}

func parsePlayerColours(c *cursor, count int) ([]PlayerColour, error) {
	out := make([]PlayerColour, 0, count)
	for i := 0; i < count; i++ {
		start := c.off
		colour := PlayerColour{Index: i}
		var err error
		if colour.ID, err = c.i32(); err != nil {
			return nil, err
		}
		if colour.Base, err = c.i32(); err != nil {
			return nil, err
		}
		if colour.UnitOutlineColour, err = c.i32(); err != nil {
			return nil, err
		}
		if colour.SelectionColour1, err = c.i32(); err != nil {
			return nil, err
		}
		if colour.SelectionColour2, err = c.i32(); err != nil {
			return nil, err
		}
		if colour.MinimapColour1, err = c.i32(); err != nil {
			return nil, err
		}
		if colour.MinimapColour2, err = c.i32(); err != nil {
			return nil, err
		}
		if colour.MinimapColour3, err = c.i32(); err != nil {
			return nil, err
		}
		if colour.StatisticsTextColor, err = c.i32(); err != nil {
			return nil, err
		}
		colour.Span = Span{Name: fmt.Sprintf("player_colour[%d]", i), Start: start, End: c.off}
		out = append(out, colour)
	}
	return out, nil
}

func parseSound(c *cursor, index int) (Sound, error) {
	start := c.off
	sound := Sound{Index: index}
	var err error
	if sound.ID, err = c.i16(); err != nil {
		return sound, err
	}
	if sound.PlayDelay, err = c.i16(); err != nil {
		return sound, err
	}
	items, err := c.i16()
	if err != nil {
		return sound, err
	}
	if items < 0 {
		return sound, fmt.Errorf("negative sound item count at %d", c.off-2)
	}
	if sound.CacheTime, err = c.i32(); err != nil {
		return sound, err
	}
	if sound.TotalProbability, err = c.i16(); err != nil {
		return sound, err
	}
	sound.Items = make([]SoundItem, 0, int(items))
	for i := 0; i < int(items); i++ {
		item, err := parseSoundItem(c, i)
		if err != nil {
			return sound, fmt.Errorf("item[%d]: %w", i, err)
		}
		sound.Items = append(sound.Items, item)
	}
	sound.Span = Span{Name: fmt.Sprintf("sound[%d]", index), Start: start, End: c.off}
	return sound, nil
}

func parseSoundItem(c *cursor, index int) (SoundItem, error) {
	start := c.off
	item := SoundItem{Index: index}
	var err error
	if item.FileName, err = c.debugString("sound_item"); err != nil {
		return item, fmt.Errorf("debug string: %w", err)
	}
	if item.ResourceID, err = c.i32(); err != nil {
		return item, err
	}
	if item.Probability, err = c.i16(); err != nil {
		return item, err
	}
	if item.Civ, err = c.i16(); err != nil {
		return item, err
	}
	if item.IconSet, err = c.i16(); err != nil {
		return item, err
	}
	item.Span = Span{Name: fmt.Sprintf("sound_item[%d]", index), Start: start, End: c.off}
	return item, nil
}

func parseUnit(c *cursor, civIndex, unitIndex int, versionAtLeast88 bool) (UnitSummary, error) {
	if !versionAtLeast88 {
		return UnitSummary{}, errors.New("unit indexing currently supports VER 8.8+ unit records")
	}
	unit := UnitSummary{CivIndex: civIndex, Index: unitIndex, Present: true}
	typeByte, err := c.u8()
	if err != nil {
		return UnitSummary{}, err
	}
	unit.Type = int(typeByte)
	start := c.off
	unit.ID, err = c.i16()
	if err != nil {
		return UnitSummary{}, err
	}
	unit.FieldSpans.ID = Span{Name: "id", Start: start, End: c.off}
	if err := c.skip(4 + 4); err != nil {
		return UnitSummary{}, err
	}
	start = c.off
	unit.Class, err = c.i16()
	if err != nil {
		return UnitSummary{}, err
	}
	unit.ClassName = EffectUnitClassName(unit.Class)
	unit.FieldSpans.Class = Span{Name: "class", Start: start, End: c.off}
	start = c.off
	unit.StandingGraphic1, err = c.i16()
	if err != nil {
		return UnitSummary{}, err
	}
	unit.FieldSpans.StandingGraphic1 = Span{Name: "standing_graphic_1", Start: start, End: c.off}
	start = c.off
	unit.StandingGraphic2, err = c.i16()
	if err != nil {
		return UnitSummary{}, err
	}
	unit.FieldSpans.StandingGraphic2 = Span{Name: "standing_graphic_2", Start: start, End: c.off}
	start = c.off
	unit.DyingGraphic, err = c.i16()
	if err != nil {
		return UnitSummary{}, err
	}
	unit.FieldSpans.DyingGraphic = Span{Name: "dying_graphic", Start: start, End: c.off}
	if err := c.skip(2 + 1); err != nil {
		return UnitSummary{}, err
	}
	start = c.off
	unit.HitPoints, err = c.i16()
	if err != nil {
		return UnitSummary{}, err
	}
	unit.FieldSpans.HitPoints = Span{Name: "hit_points", Start: start, End: c.off}
	start = c.off
	unit.LineOfSight, err = c.f32()
	if err != nil {
		return UnitSummary{}, err
	}
	unit.FieldSpans.LineOfSight = Span{Name: "line_of_sight", Start: start, End: c.off}
	if err := c.skip(1); err != nil {
		return UnitSummary{}, err
	}
	start = c.off
	unit.CollisionSizeX, err = c.f32()
	if err != nil {
		return UnitSummary{}, err
	}
	unit.FieldSpans.CollisionSizeX = Span{Name: "collision_size_x", Start: start, End: c.off}
	start = c.off
	unit.CollisionSizeY, err = c.f32()
	if err != nil {
		return UnitSummary{}, err
	}
	unit.FieldSpans.CollisionSizeY = Span{Name: "collision_size_y", Start: start, End: c.off}
	start = c.off
	unit.CollisionSizeZ, err = c.f32()
	if err != nil {
		return UnitSummary{}, err
	}
	unit.FieldSpans.CollisionSizeZ = Span{Name: "collision_size_z", Start: start, End: c.off}
	if err := c.skip(2 + 2 + 2); err != nil {
		return UnitSummary{}, err
	}
	start = c.off
	unit.BloodUnitID, err = c.i16()
	if err != nil {
		return UnitSummary{}, err
	}
	unit.FieldSpans.BloodUnitID = Span{Name: "blood_unit_id", Start: start, End: c.off}
	if err := c.skip(1 + 1); err != nil {
		return UnitSummary{}, err
	}
	start = c.off
	unit.IconID, err = c.i16()
	if err != nil {
		return UnitSummary{}, err
	}
	unit.FieldSpans.IconID = Span{Name: "icon_id", Start: start, End: c.off}
	if err := c.skip(1 + 2); err != nil {
		return UnitSummary{}, err
	}
	start = c.off
	unit.Enabled, err = c.u8()
	if err != nil {
		return UnitSummary{}, err
	}
	unit.FieldSpans.Enabled = Span{Name: "enabled", Start: start, End: c.off}
	if err := c.skip(1 + 2 + 2 + 2 + 2); err != nil {
		return UnitSummary{}, err
	}
	start = c.off
	unit.ClearanceSizeX, err = c.f32()
	if err != nil {
		return UnitSummary{}, err
	}
	unit.FieldSpans.ClearanceSizeX = Span{Name: "clearance_size_x", Start: start, End: c.off}
	start = c.off
	unit.ClearanceSizeY, err = c.f32()
	if err != nil {
		return UnitSummary{}, err
	}
	unit.FieldSpans.ClearanceSizeY = Span{Name: "clearance_size_y", Start: start, End: c.off}
	if err := c.skip(1 + 1); err != nil {
		return UnitSummary{}, err
	}
	start = c.off
	unit.TerrainTableID, err = c.i16()
	if err != nil {
		return UnitSummary{}, err
	}
	unit.FieldSpans.TerrainTableID = Span{Name: "terrain_table_id", Start: start, End: c.off}
	start = c.off
	unit.MovementType, err = c.u8()
	if err != nil {
		return UnitSummary{}, err
	}
	unit.FieldSpans.MovementType = Span{Name: "movement_type", Start: start, End: c.off}
	unit.FlyMode = unit.MovementType
	if err := c.skip(2 + 4 + 1 + 1 + 1 + 1 + 1 + 4 + 1 + 4 + 4 + 4 + 1 + 1 + 1 + 1 + 1); err != nil {
		return UnitSummary{}, err
	}
	start = c.off
	unit.ObstructionTypeID, err = c.u8()
	if err != nil {
		return UnitSummary{}, err
	}
	unit.FieldSpans.ObstructionTypeID = Span{Name: "obstruction_type_id", Start: start, End: c.off}
	start = c.off
	unit.ObstructionClassID, err = c.u8()
	if err != nil {
		return UnitSummary{}, err
	}
	unit.FieldSpans.ObstructionClassID = Span{Name: "obstruction_class_id", Start: start, End: c.off}
	if err := c.skip(22); err != nil {
		return UnitSummary{}, err
	}
	unit.Attributes, err = parseUnitAttributes(c)
	if err != nil {
		return UnitSummary{}, err
	}
	unit.DamageGraphics, unit.ListSpans, err = parseDamageGraphics(c)
	if err != nil {
		return UnitSummary{}, err
	}
	if err := c.skip(22); err != nil {
		return UnitSummary{}, err
	}
	name, err := c.debugString("unit_name")
	if err != nil {
		return UnitSummary{}, err
	}
	unit.Name = name.Value
	if err := c.skip(2 + 2); err != nil {
		return UnitSummary{}, err
	}
	unitType := unit.Type
	if unitType != 90 && unitType >= 20 {
		if err := c.skip(4); err != nil {
			return UnitSummary{}, err
		}
		if unitType >= 30 {
			if err := parseUnitMovement(c, &unit); err != nil {
				return UnitSummary{}, err
			}
		}
		if unitType >= 40 {
			action, err := parseAction(c, versionAtLeast88)
			if err != nil {
				return UnitSummary{}, err
			}
			unit.Action = &action
		}
		if unitType >= 50 {
			type50, err := parseType50(c, unitType)
			if err != nil {
				return UnitSummary{}, err
			}
			unit.Type50 = &type50
		}
		if unitType >= 70 {
			creatable, err := parseCreatable(c, versionAtLeast88)
			if err != nil {
				return UnitSummary{}, err
			}
			unit.Creatable = &creatable
		}
		if unitType == 80 {
			building, err := parseBuilding(c)
			if err != nil {
				return UnitSummary{}, err
			}
			unit.Building = &building
		}
	}
	return unit, nil
}

func parseUnitMovement(c *cursor, unit *UnitSummary) error {
	var err error
	if unit.MoveSprite, err = c.i16(); err != nil {
		return err
	}
	if unit.RunSprite, err = c.i16(); err != nil {
		return err
	}
	if unit.TurnSpeed, err = c.f32(); err != nil {
		return err
	}
	if unit.SizeClass, err = c.u8(); err != nil {
		return err
	}
	if unit.TrailingUnit, err = c.i16(); err != nil {
		return err
	}
	if unit.TrailingOptions, err = c.u8(); err != nil {
		return err
	}
	if unit.TrailingSpacing, err = c.f32(); err != nil {
		return err
	}
	if unit.MoveAlgorithm, err = c.u8(); err != nil {
		return err
	}
	if unit.TurnRadius, err = c.f32(); err != nil {
		return err
	}
	if unit.TurnRadiusSpeed, err = c.f32(); err != nil {
		return err
	}
	if unit.MaxYawMoving, err = c.f32(); err != nil {
		return err
	}
	if unit.StationaryYawTime, err = c.f32(); err != nil {
		return err
	}
	if unit.MaxYawStationary, err = c.f32(); err != nil {
		return err
	}
	return c.skip(4)
}

func ParseUnitRecord(record []byte, civIndex, unitIndex int, version string, recordStart int) (UnitSummary, error) {
	c := cursor{data: record}
	unit, err := parseUnit(&c, civIndex, unitIndex, versionStringAtLeast88(version))
	if err != nil {
		return UnitSummary{}, err
	}
	if c.off != len(record) {
		return UnitSummary{}, fmt.Errorf("unit record parse consumed %d of %d bytes", c.off, len(record))
	}
	offsetSpans(reflect.ValueOf(&unit), recordStart)
	unit.RecordStart = recordStart
	unit.RecordEnd = recordStart + len(record)
	unit.RecordLength = len(record)
	return unit, nil
}

func versionStringAtLeast88(version string) bool {
	version = strings.TrimSpace(version)
	if strings.HasPrefix(version, "VER 9") {
		return true
	}
	if strings.HasPrefix(version, "VER 8.8") || strings.HasPrefix(version, "VER 8.9") {
		return true
	}
	return version >= "VER 8.8"
}

func offsetSpans(v reflect.Value, delta int) {
	if delta == 0 || !v.IsValid() {
		return
	}
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return
		}
		offsetSpans(v.Elem(), delta)
		return
	}
	if v.Kind() == reflect.Struct && v.Type() == reflect.TypeOf(Span{}) && v.CanSet() {
		span := v.Interface().(Span)
		span.Start += delta
		span.End += delta
		v.Set(reflect.ValueOf(span))
		return
	}
	switch v.Kind() {
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			if v.Field(i).CanSet() || v.Field(i).Kind() == reflect.Struct || v.Field(i).Kind() == reflect.Slice || v.Field(i).Kind() == reflect.Pointer || v.Field(i).Kind() == reflect.Array {
				offsetSpans(v.Field(i), delta)
			}
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < v.Len(); i++ {
			offsetSpans(v.Index(i), delta)
		}
	}
}

func parseUnitAttributes(c *cursor) ([]UnitAttribute, error) {
	out := make([]UnitAttribute, 0, 3)
	for i := 0; i < 3; i++ {
		start := c.off
		attributeType, err := c.u16()
		if err != nil {
			return nil, err
		}
		amount, err := c.f32()
		if err != nil {
			return nil, err
		}
		flag, err := c.u8()
		if err != nil {
			return nil, err
		}
		out = append(out, UnitAttribute{
			Index:         i,
			AttributeType: attributeType,
			Amount:        amount,
			Flag:          flag,
			Active:        attributeType != 0xFFFF,
			Span:          Span{Name: fmt.Sprintf("unit_attribute[%d]", i), Start: start, End: c.off},
		})
	}
	return out, nil
}

func parseDamageGraphics(c *cursor) ([]DamageGraphic, UnitListSpans, error) {
	var spans UnitListSpans
	start := c.off
	damageGraphicSize, err := c.u8()
	if err != nil {
		return nil, spans, err
	}
	spans.DamageGraphicCount = Span{Name: "damage_graphic_count", Start: start, End: c.off}
	rowsStart := c.off
	if damageGraphicSize == 0 {
		spans.DamageGraphics = Span{Name: "damage_graphics", Start: rowsStart, End: rowsStart}
		return nil, spans, nil
	}
	out := make([]DamageGraphic, 0, int(damageGraphicSize))
	for i := 0; i < int(damageGraphicSize); i++ {
		start := c.off
		graphicID, err := c.u16()
		if err != nil {
			return nil, spans, err
		}
		damagePercent, err := c.u16()
		if err != nil {
			return nil, spans, err
		}
		flag, err := c.u8()
		if err != nil {
			return nil, spans, err
		}
		out = append(out, DamageGraphic{
			Index:         i,
			GraphicID:     graphicID,
			DamagePercent: damagePercent,
			Flag:          flag,
			Span:          Span{Name: fmt.Sprintf("damage_graphic[%d]", i), Start: start, End: c.off},
		})
	}
	spans.DamageGraphics = Span{Name: "damage_graphics", Start: rowsStart, End: c.off}
	return out, spans, nil
}

func parseAction(c *cursor, versionAtLeast88 bool) (ActionSummary, error) {
	var out ActionSummary
	if err := c.skip(2 + 4 + 4); err != nil {
		return out, err
	}
	dropSiteCountStart := c.off
	dropSitesSize, err := c.i16()
	if err != nil {
		return out, err
	}
	out.FieldSpans.DropSiteCount = Span{Name: "drop_site_count", Start: dropSiteCountStart, End: c.off}
	if dropSitesSize < 0 {
		return out, fmt.Errorf("negative drop site count at %d", c.off-2)
	}
	dropSitesStart := c.off
	out.DropSites = make([]int16, 0, int(dropSitesSize))
	for i := 0; i < int(dropSitesSize); i++ {
		id, err := c.i16()
		if err != nil {
			return out, err
		}
		out.DropSites = append(out.DropSites, id)
	}
	out.FieldSpans.DropSites = Span{Name: "drop_sites", Start: dropSitesStart, End: c.off}
	if err := c.skip(1 + 2 + 2 + 4 + 4 + 1); err != nil {
		return out, err
	}
	taskCountStart := c.off
	taskCount, err := c.i16()
	if err != nil {
		return out, err
	}
	out.FieldSpans.TaskCount = Span{Name: "task_count", Start: taskCountStart, End: c.off}
	if taskCount < 0 {
		return out, fmt.Errorf("negative bird task count at %d", c.off-2)
	}
	tasksStart := c.off
	out.Tasks = make([]TaskSummary, 0, int(taskCount))
	for i := 0; i < int(taskCount); i++ {
		task, err := parseTask(c, i, versionAtLeast88)
		if err != nil {
			return out, err
		}
		out.Tasks = append(out.Tasks, task)
	}
	out.FieldSpans.Tasks = Span{Name: "tasks", Start: tasksStart, End: c.off}
	return out, nil
}

func parseTask(c *cursor, index int, versionAtLeast88 bool) (TaskSummary, error) {
	start := c.off
	taskLen := taskRecordLen(versionAtLeast88)
	task := TaskSummary{Index: index, Span: Span{Name: fmt.Sprintf("task[%d]", index), Start: start, End: start + taskLen}}
	var err error
	if task.RecordType, err = c.i16(); err != nil {
		return task, err
	}
	if task.ID, err = c.i16(); err != nil {
		return task, err
	}
	if task.IsDefault, err = c.u8(); err != nil {
		return task, err
	}
	if task.ActionType, err = c.i16(); err != nil {
		return task, err
	}
	if task.ObjectClass, err = c.i16(); err != nil {
		return task, err
	}
	if task.ObjectID, err = c.i16(); err != nil {
		return task, err
	}
	if task.TerrainID, err = c.i16(); err != nil {
		return task, err
	}
	for i := 0; i < 4; i++ {
		value, err := c.i16()
		if err != nil {
			return task, err
		}
		task.AttributeTypes = append(task.AttributeTypes, value)
	}
	if task.WorkValue1, err = c.f32(); err != nil {
		return task, err
	}
	if task.WorkValue2, err = c.f32(); err != nil {
		return task, err
	}
	if task.WorkRange, err = c.f32(); err != nil {
		return task, err
	}
	if task.AutoSearchTargets, err = c.u8(); err != nil {
		return task, err
	}
	if task.SearchWaitTime, err = c.f32(); err != nil {
		return task, err
	}
	if task.EnableTargeting, err = c.u8(); err != nil {
		return task, err
	}
	if task.CombatLevel, err = c.u8(); err != nil {
		return task, err
	}
	tailEnd := start + taskLen
	if tailEnd < c.off || tailEnd > len(c.data) {
		return task, fmt.Errorf("task[%d] tail span invalid: %d..%d", index, c.off, tailEnd)
	}
	task.RawTail = append([]byte(nil), c.data[c.off:tailEnd]...)
	if err := c.skip(start + taskLen - c.off); err != nil {
		return task, err
	}
	task.Span.End = c.off
	return task, nil
}

func parseType50(c *cursor, unitType int) (Type50Summary, error) {
	var out Type50Summary
	var err error
	start := c.off
	if out.BaseArmor, err = c.i16(); err != nil {
		return out, err
	}
	out.FieldSpans.BaseArmor = Span{Name: "type50_base_armor", Start: start, End: c.off}
	attackCount, err := c.i16()
	if err != nil {
		return out, err
	}
	out.FieldSpans.AttackCount = Span{Name: "type50_attack_count", Start: start, End: c.off}
	if attackCount < 0 {
		return out, fmt.Errorf("negative attack count at %d", c.off-2)
	}
	rowsStart := c.off
	out.Attacks = make([]WeaponInfo, 0, int(attackCount))
	for i := 0; i < int(attackCount); i++ {
		weapon, err := parseWeaponInfo(c, i, "attack")
		if err != nil {
			return out, err
		}
		out.Attacks = append(out.Attacks, weapon)
	}
	out.FieldSpans.Attacks = Span{Name: "type50_attacks", Start: rowsStart, End: c.off}
	start = c.off
	armourCount, err := c.i16()
	if err != nil {
		return out, err
	}
	out.FieldSpans.ArmourCount = Span{Name: "type50_armour_count", Start: start, End: c.off}
	if armourCount < 0 {
		return out, fmt.Errorf("negative armour count at %d", c.off-2)
	}
	rowsStart = c.off
	out.Armours = make([]WeaponInfo, 0, int(armourCount))
	for i := 0; i < int(armourCount); i++ {
		armour, err := parseWeaponInfo(c, i, "armour")
		if err != nil {
			return out, err
		}
		out.Armours = append(out.Armours, armour)
	}
	out.FieldSpans.Armours = Span{Name: "type50_armours", Start: rowsStart, End: c.off}
	start = c.off
	if out.DefenseTerrainBonus, err = c.i16(); err != nil {
		return out, err
	}
	if err := c.skip(4); err != nil {
		return out, err
	}
	start = c.off
	out.MaxRange, err = c.f32()
	if err != nil {
		return out, err
	}
	out.FieldSpans.MaxRange = Span{Name: "type50_max_range", Start: start, End: c.off}
	start = c.off
	out.BlastWidth, err = c.f32()
	if err != nil {
		return out, err
	}
	out.FieldSpans.BlastWidth = Span{Name: "type50_blast_width", Start: start, End: c.off}
	if out.AttackSpeed, err = c.f32(); err != nil {
		return out, err
	}
	start = c.off
	out.ProjectileUnitID, err = c.i16()
	if err != nil {
		return out, err
	}
	out.FieldSpans.ProjectileUnitID = Span{Name: "type50_projectile_unit_id", Start: start, End: c.off}
	if out.BaseHitChance, err = c.i16(); err != nil {
		return out, err
	}
	if out.BreakOffCombat, err = c.u8(); err != nil {
		return out, err
	}
	if out.FrameDelay, err = c.i16(); err != nil {
		return out, err
	}
	if out.WeaponOffsetX, err = c.f32(); err != nil {
		return out, err
	}
	if out.WeaponOffsetY, err = c.f32(); err != nil {
		return out, err
	}
	if out.WeaponOffsetZ, err = c.f32(); err != nil {
		return out, err
	}
	if out.BlastAttackLevel, err = c.u8(); err != nil {
		return out, err
	}
	if out.MinRange, err = c.f32(); err != nil {
		return out, err
	}
	if out.MissedMissileSpread, err = c.f32(); err != nil {
		return out, err
	}
	if err != nil {
		return out, err
	}
	start = c.off
	out.AttackGraphic, err = c.i16()
	if err != nil {
		return out, err
	}
	out.FieldSpans.AttackGraphic = Span{Name: "type50_attack_graphic", Start: start, End: c.off}
	if out.DisplayedArmor, err = c.i16(); err != nil {
		return out, err
	}
	if out.DisplayedAttack, err = c.i16(); err != nil {
		return out, err
	}
	if out.DisplayedRange, err = c.f32(); err != nil {
		return out, err
	}
	if out.DisplayedReloadTime, err = c.f32(); err != nil {
		return out, err
	}
	start = c.off
	out.BlastDamage, err = c.f32()
	if err != nil {
		return out, err
	}
	out.FieldSpans.BlastDamage = Span{Name: "type50_blast_damage", Start: start, End: c.off}
	if err := c.skip(4 + 4 + 2 + 4 + 2); err != nil {
		return out, err
	}
	if unitType == 60 {
		start = c.off
		if out.MissileType, err = c.u8(); err != nil {
			return out, err
		}
		if out.TargetingType, err = c.u8(); err != nil {
			return out, err
		}
		if out.MissileHitInfo, err = c.u8(); err != nil {
			return out, err
		}
		if out.MissileDieInfo, err = c.u8(); err != nil {
			return out, err
		}
		if out.AreaEffectSpecials, err = c.u8(); err != nil {
			return out, err
		}
		if out.BallisticsRatio, err = c.f32(); err != nil {
			return out, err
		}
		out.FieldSpans.MissileType = Span{Name: "type60_missile_type", Start: start, End: start + 1}
		out.FieldSpans.TargetingType = Span{Name: "type60_targeting_type", Start: start + 1, End: start + 2}
		out.FieldSpans.MissileHitInfo = Span{Name: "type60_missile_hit_info", Start: start + 2, End: start + 3}
		out.FieldSpans.MissileDieInfo = Span{Name: "type60_missile_die_info", Start: start + 3, End: start + 4}
		out.FieldSpans.AreaEffectSpecials = Span{Name: "type60_area_effect_specials", Start: start + 4, End: start + 5}
		out.FieldSpans.BallisticsRatio = Span{Name: "type60_ballistics_ratio", Start: start + 5, End: c.off}
	}
	return out, nil
}

func parseWeaponInfo(c *cursor, index int, kind string) (WeaponInfo, error) {
	start := c.off
	class, err := c.i16()
	if err != nil {
		return WeaponInfo{}, err
	}
	value, err := c.i16()
	if err != nil {
		return WeaponInfo{}, err
	}
	return WeaponInfo{
		Index: index,
		Class: class,
		Value: value,
		Span:  Span{Name: fmt.Sprintf("%s[%d]", kind, index), Start: start, End: c.off},
	}, nil
}

func parseCreatable(c *cursor, versionAtLeast88 bool) (CreatableSummary, error) {
	var out CreatableSummary
	if !versionAtLeast88 {
		return out, errors.New("creatable indexing currently supports VER 8.8+")
	}
	out.Costs = make([]AttributeCost, 0, 3)
	for i := 0; i < 3; i++ {
		cost, err := parseAttributeCost(c, i)
		if err != nil {
			return out, err
		}
		out.Costs = append(out.Costs, cost)
	}
	start := c.off
	trainLocationCount, err := c.i16()
	if err != nil {
		return out, err
	}
	out.FieldSpans.TrainLocationCount = Span{Name: "train_location_count", Start: start, End: c.off}
	if trainLocationCount < 0 {
		return out, fmt.Errorf("negative train location count at %d", c.off-2)
	}
	out.TrainLocationCount = int(trainLocationCount)
	rowsStart := c.off
	out.TrainLocations = make([]TrainLocation, 0, int(trainLocationCount))
	for i := 0; i < int(trainLocationCount); i++ {
		location, err := parseTrainLocation(c, i)
		if err != nil {
			return out, err
		}
		out.TrainLocations = append(out.TrainLocations, location)
		if i == 0 {
			out.TrainTime0 = location.TrainTime
			out.TrainUnitID0 = location.TrainUnitID
			out.TrainButtonID0 = location.TrainButton
			out.TrainHotKeyID0 = location.TrainHotkey
			out.FieldSpans.TrainTime0 = Span{Name: "train_time_0", Start: location.Span.Start, End: location.Span.Start + 2}
			out.FieldSpans.TrainUnitID0 = Span{Name: "train_unit_id_0", Start: location.Span.Start + 2, End: location.Span.Start + 4}
			out.FieldSpans.TrainButtonID0 = Span{Name: "train_button_id_0", Start: location.Span.Start + 4, End: location.Span.Start + 5}
			out.FieldSpans.TrainHotKeyID0 = Span{Name: "train_hotkey_id_0", Start: location.Span.Start + 5, End: location.Span.End}
		}
	}
	out.FieldSpans.TrainLocations = Span{Name: "train_locations", Start: rowsStart, End: c.off}
	if err := c.skip(4 + 4 + 1 + 1 + 4 + 2 + 2 + 2 + 2 + 4 + 4 + 2 + 2 + 2 + 4 + 1 + 4); err != nil {
		return out, err
	}
	start = c.off
	out.ButtonIconID, err = c.i16()
	if err != nil {
		return out, err
	}
	out.FieldSpans.ButtonIconID = Span{Name: "creatable_button_icon_id", Start: start, End: c.off}
	if err := c.skip(4 + 4); err != nil {
		return out, err
	}
	start = c.off
	out.ButtonHotkeyAction, err = c.i16()
	if err != nil {
		return out, err
	}
	out.FieldSpans.ButtonHotkeyAction = Span{Name: "creatable_button_hotkey_action", Start: start, End: c.off}
	if err := c.skip(4 + 4 + 4 + 4 + 1 + 4 + 4 + 4 + 4 + 4 + 1 + 2); err != nil {
		return out, err
	}
	return out, nil
}

func parseAttributeCost(c *cursor, index int) (AttributeCost, error) {
	start := c.off
	attributeType, err := c.i16()
	if err != nil {
		return AttributeCost{}, err
	}
	amount, err := c.i16()
	if err != nil {
		return AttributeCost{}, err
	}
	flag, err := c.u8()
	if err != nil {
		return AttributeCost{}, err
	}
	padding, err := c.u8()
	if err != nil {
		return AttributeCost{}, err
	}
	return AttributeCost{
		Index:         index,
		AttributeType: attributeType,
		Amount:        amount,
		Flag:          flag,
		Padding:       padding,
		Active:        attributeType >= 0,
		Span:          Span{Name: fmt.Sprintf("cost[%d]", index), Start: start, End: c.off},
	}, nil
}

func parseTrainLocation(c *cursor, index int) (TrainLocation, error) {
	start := c.off
	trainTime, err := c.i16()
	if err != nil {
		return TrainLocation{}, err
	}
	trainUnitID, err := c.i16()
	if err != nil {
		return TrainLocation{}, err
	}
	trainButton, err := c.u8()
	if err != nil {
		return TrainLocation{}, err
	}
	trainHotkey, err := c.i32()
	if err != nil {
		return TrainLocation{}, err
	}
	return TrainLocation{
		Index:       index,
		TrainTime:   trainTime,
		TrainUnitID: trainUnitID,
		TrainButton: trainButton,
		TrainHotkey: trainHotkey,
		Span:        Span{Name: fmt.Sprintf("train_location[%d]", index), Start: start, End: c.off},
	}, nil
}

func parseBuilding(c *cursor) (BuildingSummary, error) {
	var out BuildingSummary
	var err error
	if len(c.data)-c.off < 25 {
		return out, io.ErrUnexpectedEOF
	}
	pre := cursor{data: c.data[c.off : c.off+25]}
	if out.SpawningGraphic, err = pre.i16(); err != nil {
		return out, err
	}
	if out.UpgradeGraphic, err = pre.i16(); err != nil {
		return out, err
	}
	if err := c.skip(25); err != nil {
		return out, err
	}
	out.LinkedBuildings = make([]LinkedBuilding, 0, 4)
	for i := 0; i < 4; i++ {
		start := c.off
		unitID, err := c.i16()
		if err != nil {
			return out, err
		}
		x, err := c.f32()
		if err != nil {
			return out, err
		}
		y, err := c.f32()
		if err != nil {
			return out, err
		}
		out.LinkedBuildings = append(out.LinkedBuildings, LinkedBuilding{
			Index:  i,
			UnitID: unitID,
			X:      x,
			Y:      y,
			Active: unitID != -1,
			Span:   Span{Name: fmt.Sprintf("linked_building[%d]", i), Start: start, End: c.off},
		})
	}
	if len(c.data)-c.off < 33 {
		return out, io.ErrUnexpectedEOF
	}
	post := cursor{data: c.data[c.off : c.off+33]}
	if out.StackUnit, err = post.i16(); err != nil {
		return out, err
	}
	if out.HeadUnit, err = post.i16(); err != nil {
		return out, err
	}
	if out.TransformUnit, err = post.i16(); err != nil {
		return out, err
	}
	if out.PileUnit, err = post.i16(); err != nil {
		return out, err
	}
	if out.ConstructionUnit, err = post.i16(); err != nil {
		return out, err
	}
	// The current layout retains a second legacy transform-unit slot here.
	// Keep it opaque until its distinction from attribute 204 is established.
	if err := post.skip(2); err != nil {
		return out, err
	}
	if out.TransformSound, err = post.i16(); err != nil {
		return out, err
	}
	if out.ConstructionSound, err = post.i16(); err != nil {
		return out, err
	}
	if out.GarrisonType, err = post.u8(); err != nil {
		return out, err
	}
	if out.GarrisonHealRate, err = post.f32(); err != nil {
		return out, err
	}
	if out.GarrisonRepairRate, err = post.f32(); err != nil {
		return out, err
	}
	if out.SalvageUnit, err = post.i16(); err != nil {
		return out, err
	}
	out.SalvageAttributes = make([]uint8, 6)
	for i := range out.SalvageAttributes {
		if out.SalvageAttributes[i], err = post.u8(); err != nil {
			return out, err
		}
	}
	if err := c.skip(33); err != nil {
		return out, err
	}
	return out, nil
}

func taskRecordLen(versionAtLeast88 bool) int {
	if versionAtLeast88 {
		return formatByteLen(taskFormat88)
	}
	return formatByteLen(taskFormat84)
}

func formatByteLen(format string) int {
	total := 0
	for _, ch := range format {
		switch ch {
		case '<':
			continue
		case 'b':
			total += 1
		case 'h':
			total += 2
		case 'l', 'f':
			total += 4
		default:
			panic(fmt.Sprintf("unsupported format token %q", ch))
		}
	}
	return total
}

func parseGraphic(c *cursor, index int) (Graphic, error) {
	g := Graphic{Index: index, Present: true}
	var err error
	if g.Name, err = c.debugString("name"); err != nil {
		return g, err
	}
	if g.FileName, err = c.debugString("file_name"); err != nil {
		return g, err
	}
	if g.ParticleEffectName, err = c.debugString("particle_effect_name"); err != nil {
		return g, err
	}
	start := c.off
	if g.SLP, err = c.i32(); err != nil {
		return g, err
	}
	g.FieldSpans.SLP = Span{Name: "slp", Start: start, End: c.off}
	start = c.off
	if g.IsLoaded, err = c.i8(); err != nil {
		return g, err
	}
	g.FieldSpans.IsLoaded = Span{Name: "is_loaded", Start: start, End: c.off}
	start = c.off
	if g.OldColorFlag, err = c.i8(); err != nil {
		return g, err
	}
	g.FieldSpans.OldColorFlag = Span{Name: "old_color_flag", Start: start, End: c.off}
	start = c.off
	if g.Layer, err = c.i8(); err != nil {
		return g, err
	}
	g.FieldSpans.Layer = Span{Name: "layer", Start: start, End: c.off}
	start = c.off
	if g.PlayerColor, err = c.i8(); err != nil {
		return g, err
	}
	g.FieldSpans.PlayerColor = Span{Name: "player_color", Start: start, End: c.off}
	start = c.off
	if g.Rainbow, err = c.i8(); err != nil {
		return g, err
	}
	g.FieldSpans.Rainbow = Span{Name: "rainbow", Start: start, End: c.off}
	start = c.off
	if g.TransparentSelect, err = c.i8(); err != nil {
		return g, err
	}
	g.FieldSpans.TransparentSelect = Span{Name: "transparent_selection", Start: start, End: c.off}
	for i := range g.Coordinates {
		start = c.off
		if g.Coordinates[i], err = c.i16(); err != nil {
			return g, err
		}
		g.FieldSpans.Coordinates[i] = Span{Name: fmt.Sprintf("coordinates[%d]", i), Start: start, End: c.off}
	}
	start = c.off
	if g.DeltaCount, err = c.i16(); err != nil {
		return g, err
	}
	g.FieldSpans.DeltaCount = Span{Name: "delta_count", Start: start, End: c.off}
	if g.DeltaCount < 0 {
		return g, fmt.Errorf("negative delta count at %d", c.off-2)
	}
	start = c.off
	if g.SoundID, err = c.i16(); err != nil {
		return g, err
	}
	g.FieldSpans.SoundID = Span{Name: "sound_id", Start: start, End: c.off}
	start = c.off
	if g.WwiseSoundID, err = c.u32(); err != nil {
		return g, err
	}
	g.FieldSpans.WwiseSoundID = Span{Name: "wwise_sound_id", Start: start, End: c.off}
	start = c.off
	if g.AngleSoundsUsed, err = c.i8(); err != nil {
		return g, err
	}
	g.FieldSpans.AngleSoundsUsed = Span{Name: "angle_sounds_used", Start: start, End: c.off}
	start = c.off
	if g.FrameCount, err = c.i16(); err != nil {
		return g, err
	}
	g.FieldSpans.FrameCount = Span{Name: "frame_count", Start: start, End: c.off}
	start = c.off
	if g.AngleCount, err = c.i16(); err != nil {
		return g, err
	}
	g.FieldSpans.AngleCount = Span{Name: "angle_count", Start: start, End: c.off}
	if g.AngleCount < 0 {
		return g, fmt.Errorf("negative angle count at %d", c.off-2)
	}
	start = c.off
	if g.SpeedMultiplier, err = c.f32(); err != nil {
		return g, err
	}
	g.FieldSpans.SpeedMultiplier = Span{Name: "speed_multiplier", Start: start, End: c.off}
	start = c.off
	if g.FrameDuration, err = c.f32(); err != nil {
		return g, err
	}
	g.FieldSpans.FrameDuration = Span{Name: "frame_duration", Start: start, End: c.off}
	start = c.off
	if g.ReplayDelay, err = c.f32(); err != nil {
		return g, err
	}
	g.FieldSpans.ReplayDelay = Span{Name: "replay_delay", Start: start, End: c.off}
	start = c.off
	if g.SequenceType, err = c.u8(); err != nil {
		return g, err
	}
	g.FieldSpans.SequenceType = Span{Name: "sequence_type", Start: start, End: c.off}
	start = c.off
	if g.ID, err = c.i16(); err != nil {
		return g, err
	}
	g.FieldSpans.ID = Span{Name: "id", Start: start, End: c.off}
	start = c.off
	if g.MirroringMode, err = c.i8(); err != nil {
		return g, err
	}
	g.FieldSpans.MirroringMode = Span{Name: "mirroring_mode", Start: start, End: c.off}
	start = c.off
	if g.EditorFlag, err = c.i8(); err != nil {
		return g, err
	}
	g.FieldSpans.EditorFlag = Span{Name: "editor_flag", Start: start, End: c.off}
	for i := 0; i < int(g.DeltaCount); i++ {
		delta, err := parseGraphicDelta(c, i)
		if err != nil {
			return g, fmt.Errorf("delta[%d]: %w", i, err)
		}
		g.Deltas = append(g.Deltas, delta)
	}
	if g.AngleSoundsUsed != 0 {
		for i := 0; i < int(g.AngleCount); i++ {
			sound, err := parseGraphicAngleSound(c, i)
			if err != nil {
				return g, fmt.Errorf("angle_sound[%d]: %w", i, err)
			}
			g.AngleSounds = append(g.AngleSounds, sound)
		}
	}
	return g, nil
}

func parseGraphicDelta(c *cursor, index int) (GraphicDelta, error) {
	start := c.off
	delta := GraphicDelta{Index: index}
	var err error
	if delta.GraphicID, err = c.i16(); err != nil {
		return delta, err
	}
	if delta.Padding1, err = c.i16(); err != nil {
		return delta, err
	}
	if delta.SpritePtr, err = c.i32(); err != nil {
		return delta, err
	}
	if delta.OffsetX, err = c.i16(); err != nil {
		return delta, err
	}
	if delta.OffsetY, err = c.i16(); err != nil {
		return delta, err
	}
	if delta.DisplayAngle, err = c.i16(); err != nil {
		return delta, err
	}
	if delta.Padding2, err = c.i16(); err != nil {
		return delta, err
	}
	delta.Span = Span{Name: fmt.Sprintf("graphic_delta[%d]", index), Start: start, End: c.off}
	return delta, nil
}

func parseGraphicAngleSound(c *cursor, index int) (GraphicAngleSound, error) {
	start := c.off
	sound := GraphicAngleSound{Index: index}
	var err error
	if sound.FrameNum, err = c.i16(); err != nil {
		return sound, err
	}
	if sound.SoundID, err = c.i16(); err != nil {
		return sound, err
	}
	if sound.WwiseSoundID, err = c.u32(); err != nil {
		return sound, err
	}
	if sound.FrameNum2, err = c.i16(); err != nil {
		return sound, err
	}
	if sound.WwiseSoundID2, err = c.u32(); err != nil {
		return sound, err
	}
	if sound.SoundID2, err = c.i16(); err != nil {
		return sound, err
	}
	if sound.FrameNum3, err = c.i16(); err != nil {
		return sound, err
	}
	if sound.WwiseSoundID3, err = c.u32(); err != nil {
		return sound, err
	}
	if sound.SoundID3, err = c.i16(); err != nil {
		return sound, err
	}
	sound.Span = Span{Name: fmt.Sprintf("graphic_angle_sound[%d]", index), Start: start, End: c.off}
	return sound, nil
}

func putI16(payload []byte, span Span, value int16) {
	binary.LittleEndian.PutUint16(payload[span.Start:span.End], uint16(value))
}

func putU16(payload []byte, span Span, value uint16) {
	binary.LittleEndian.PutUint16(payload[span.Start:span.End], value)
}

func putU8(payload []byte, span Span, value uint8) {
	payload[span.Start] = value
}

func putI32(payload []byte, span Span, value int32) {
	binary.LittleEndian.PutUint32(payload[span.Start:span.End], uint32(value))
}

func putU32(payload []byte, span Span, value uint32) {
	binary.LittleEndian.PutUint32(payload[span.Start:span.End], value)
}

func putF32(payload []byte, span Span, value float32) {
	binary.LittleEndian.PutUint32(payload[span.Start:span.End], math.Float32bits(value))
}

type cursor struct {
	data []byte
	off  int
}

func (c *cursor) require(n int) error {
	if n < 0 {
		return fmt.Errorf("negative read length %d at %d", n, c.off)
	}
	if c.off+n > len(c.data) {
		return fmt.Errorf("read %d bytes at %d exceeds payload length %d", n, c.off, len(c.data))
	}
	return nil
}

func (c *cursor) skip(n int) error {
	if err := c.require(n); err != nil {
		return err
	}
	c.off += n
	return nil
}

func (c *cursor) bytes(n int) ([]byte, error) {
	if err := c.require(n); err != nil {
		return nil, err
	}
	out := c.data[c.off : c.off+n]
	c.off += n
	return out, nil
}

func (c *cursor) u8() (uint8, error) {
	if err := c.require(1); err != nil {
		return 0, err
	}
	v := c.data[c.off]
	c.off++
	return v, nil
}

func (c *cursor) i8() (int8, error) {
	v, err := c.u8()
	return int8(v), err
}

func (c *cursor) i16() (int16, error) {
	if err := c.require(2); err != nil {
		return 0, err
	}
	v := int16(binary.LittleEndian.Uint16(c.data[c.off:]))
	c.off += 2
	return v, nil
}

func (c *cursor) u16() (uint16, error) {
	if err := c.require(2); err != nil {
		return 0, err
	}
	v := binary.LittleEndian.Uint16(c.data[c.off:])
	c.off += 2
	return v, nil
}

func (c *cursor) u32() (uint32, error) {
	if err := c.require(4); err != nil {
		return 0, err
	}
	v := binary.LittleEndian.Uint32(c.data[c.off:])
	c.off += 4
	return v, nil
}

func (c *cursor) i32() (int32, error) {
	if err := c.require(4); err != nil {
		return 0, err
	}
	v := int32(binary.LittleEndian.Uint32(c.data[c.off:]))
	c.off += 4
	return v, nil
}

func (c *cursor) f32() (float32, error) {
	if err := c.require(4); err != nil {
		return 0, err
	}
	v := math.Float32frombits(binary.LittleEndian.Uint32(c.data[c.off:]))
	c.off += 4
	return v, nil
}

func (c *cursor) debugString(name string) (DebugString, error) {
	start := c.off
	marker, err := c.u16()
	if err != nil {
		return DebugString{}, err
	}
	if marker != DebugStringMarker {
		return DebugString{}, fmt.Errorf("debug string %s at %d has marker 0x%04x, want 0x%04x", name, start, marker, DebugStringMarker)
	}
	length, err := c.u16()
	if err != nil {
		return DebugString{}, err
	}
	dataStart := c.off
	raw, err := c.bytes(int(length))
	if err != nil {
		return DebugString{}, err
	}
	return DebugString{
		Value:      string(raw),
		Marker:     marker,
		ByteLength: int(length),
		Span:       Span{Name: name, Start: start, End: c.off},
		DataSpan:   Span{Name: name + "_data", Start: dataStart, End: c.off},
	}, nil
}
