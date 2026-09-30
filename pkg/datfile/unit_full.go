package datfile

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
)

// DecodeUnitFile reads one unit record and projects the fields that the
// current DE parser can prove. It intentionally does not attach full maps to
// every unit in Index, which keeps normal DAT indexing compact.
func DecodeUnitFile(path string, civID, unitID int) (UnitSummary, map[string]any, error) {
	compressed, err := os.ReadFile(path)
	if err != nil {
		return UnitSummary{}, nil, err
	}
	payload, err := Inflate(compressed)
	if err != nil {
		return UnitSummary{}, nil, err
	}
	idx, err := Parse(payload)
	if err != nil {
		return UnitSummary{}, nil, err
	}
	if civID < 0 || civID >= len(idx.Civs) {
		return UnitSummary{}, nil, fmt.Errorf("civ %d is outside civ table", civID)
	}
	if unitID < 0 || unitID >= len(idx.Civs[civID].Units) {
		return UnitSummary{}, nil, fmt.Errorf("unit %d is outside civ %d unit table", unitID, civID)
	}
	unit := idx.Civs[civID].Units[unitID]
	if !unit.Present || unit.RecordStart < 0 || unit.RecordEnd > len(payload) || unit.RecordEnd < unit.RecordStart {
		return UnitSummary{}, nil, fmt.Errorf("unit %d in civ %d is absent", unitID, civID)
	}
	if err := populateSkippedUnitFields(payload[unit.RecordStart:unit.RecordEnd], &unit); err != nil {
		return UnitSummary{}, nil, err
	}
	full, err := DecodeUnitFields(payload[unit.RecordStart:unit.RecordEnd], idx.Version)
	if err != nil {
		return UnitSummary{}, nil, err
	}
	unit.FullFields = full
	return unit, full, nil
}

func populateSkippedUnitFields(record []byte, unit *UnitSummary) error {
	base := unit.RecordStart
	relative := func(offset int) int { return offset - base }
	readI16 := func(offset int) (int16, error) {
		if offset < 0 || offset+2 > len(record) {
			return 0, fmt.Errorf("unit field offset %d is outside record length %d", offset, len(record))
		}
		return int16(binary.LittleEndian.Uint16(record[offset : offset+2])), nil
	}
	readU8 := func(offset int) (uint8, error) {
		if offset < 0 || offset >= len(record) {
			return 0, fmt.Errorf("unit field offset %d is outside record length %d", offset, len(record))
		}
		return record[offset], nil
	}
	var err error
	if unit.UndeadGraphic, err = readI16(relative(unit.FieldSpans.DyingGraphic.End)); err != nil {
		return err
	}
	if unit.UndeadFlag, err = readU8(relative(unit.FieldSpans.DyingGraphic.End) + 2); err != nil {
		return err
	}
	if unit.GarrisonCapacity, err = readU8(relative(unit.FieldSpans.LineOfSight.End)); err != nil {
		return err
	}
	if unit.TrainSound, err = readI16(relative(unit.FieldSpans.CollisionSizeZ.End)); err != nil {
		return err
	}
	if unit.DamageSound, err = readI16(relative(unit.FieldSpans.CollisionSizeZ.End) + 2); err != nil {
		return err
	}
	if unit.DeathSpawn, err = readI16(relative(unit.FieldSpans.CollisionSizeZ.End) + 4); err != nil {
		return err
	}
	return nil
}

// DecodeUnitFields decodes the named scalar and nested fields of a DE unit
// record. It is deliberately map-shaped for the SQL/export boundary: the DAT
// record is a tagged family of layouts, not one fixed row for every unit type.
func DecodeUnitFields(record []byte, version string) (map[string]any, error) {
	c := cursor{data: record}
	unit, err := parseUnit(&c, 0, 0, versionStringAtLeast88(version))
	if err != nil {
		return nil, err
	}
	if c.off != len(record) {
		return nil, fmt.Errorf("full unit decode consumed %d of %d bytes", c.off, len(record))
	}
	if err := populateSkippedUnitFields(record, &unit); err != nil {
		return nil, err
	}
	fields := map[string]any{
		"type":                 unit.Type,
		"id":                   unit.ID,
		"name":                 unit.Name,
		"class":                unit.Class,
		"class_name":           unit.ClassName,
		"hit_points":           unit.HitPoints,
		"line_of_sight":        unit.LineOfSight,
		"collision_size_x":     unit.CollisionSizeX,
		"collision_size_y":     unit.CollisionSizeY,
		"collision_size_z":     unit.CollisionSizeZ,
		"clearance_size_x":     unit.ClearanceSizeX,
		"clearance_size_y":     unit.ClearanceSizeY,
		"terrain_table_id":     unit.TerrainTableID,
		"movement_type":        unit.MovementType,
		"fly_mode":             unit.FlyMode,
		"obstruction_type_id":  unit.ObstructionTypeID,
		"obstruction_class_id": unit.ObstructionClassID,
		"standing_graphic_1":   unit.StandingGraphic1,
		"standing_graphic_2":   unit.StandingGraphic2,
		"dying_graphic":        unit.DyingGraphic,
		"undead_graphic":       unit.UndeadGraphic,
		"undead_flag":          unit.UndeadFlag,
		"blood_unit_id":        unit.BloodUnitID,
		"train_sound":          unit.TrainSound,
		"damage_sound":         unit.DamageSound,
		"death_spawn":          unit.DeathSpawn,
		"garrison_capacity":    unit.GarrisonCapacity,
		"icon_id":              unit.IconID,
		"enabled":              unit.Enabled,
	}
	for name, value := range map[string]any{
		"move_sprite": unit.MoveSprite, "run_sprite": unit.RunSprite, "turn_speed": unit.TurnSpeed,
		"size_class": unit.SizeClass, "trailing_unit": unit.TrailingUnit, "trailing_options": unit.TrailingOptions,
		"trailing_spacing": unit.TrailingSpacing, "move_algorithm": unit.MoveAlgorithm,
		"turn_radius": unit.TurnRadius, "turn_radius_speed": unit.TurnRadiusSpeed,
		"maximum_yaw_per_second_moving":      unit.MaxYawMoving,
		"stationary_yaw_per_revolution_time": unit.StationaryYawTime,
		"maximum_yaw_per_second_stationary":  unit.MaxYawStationary,
	} {
		if unit.Type >= 30 && unit.Type != 90 {
			fields[name] = value
		}
	}
	attrs := make([]map[string]any, 0, len(unit.Attributes))
	for _, row := range unit.Attributes {
		attrs = append(attrs, map[string]any{"index": row.Index, "type": row.AttributeType, "amount": row.Amount, "flag": row.Flag, "active": row.Active})
	}
	fields["attributes"] = attrs
	damage := make([]map[string]any, 0, len(unit.DamageGraphics))
	for _, row := range unit.DamageGraphics {
		damage = append(damage, map[string]any{"index": row.Index, "graphic_id": row.GraphicID, "damage_percent": row.DamagePercent, "flag": row.Flag})
	}
	fields["damage_graphics"] = damage
	if unit.Action != nil {
		fields["drop_sites"] = unit.Action.DropSites
		fields["tasks"] = unit.Action.Tasks
	}
	if unit.Type50 != nil {
		for name, value := range map[string]any{
			"base_armor": unit.Type50.BaseArmor, "defense_terrain_bonus": unit.Type50.DefenseTerrainBonus,
			"attack_speed": unit.Type50.AttackSpeed, "base_hit_chance": unit.Type50.BaseHitChance,
			"break_off_combat": unit.Type50.BreakOffCombat, "frame_delay": unit.Type50.FrameDelay,
			"weapon_offset_x": unit.Type50.WeaponOffsetX, "weapon_offset_y": unit.Type50.WeaponOffsetY,
			"weapon_offset_z": unit.Type50.WeaponOffsetZ, "blast_attack_level": unit.Type50.BlastAttackLevel,
			"min_range": unit.Type50.MinRange, "missed_missile_spread": unit.Type50.MissedMissileSpread,
			"displayed_armor": unit.Type50.DisplayedArmor, "displayed_attack": unit.Type50.DisplayedAttack,
			"displayed_range": unit.Type50.DisplayedRange, "displayed_reload_time": unit.Type50.DisplayedReloadTime,
			"missile_type": unit.Type50.MissileType, "targeting_type": unit.Type50.TargetingType,
			"missile_hit_info": unit.Type50.MissileHitInfo, "missile_die_info": unit.Type50.MissileDieInfo,
			"area_effect_specials": unit.Type50.AreaEffectSpecials, "ballistics_ratio": unit.Type50.BallisticsRatio,
		} {
			fields[name] = value
		}
		fields["max_range"] = unit.Type50.MaxRange
		fields["area_effect_range"] = unit.Type50.BlastWidth
		fields["projectile_unit"] = unit.Type50.ProjectileUnitID
		fields["fight_sprite"] = unit.Type50.AttackGraphic
		fields["blast_damage"] = unit.Type50.BlastDamage
		attacks := make([]map[string]any, 0, len(unit.Type50.Attacks))
		for _, row := range unit.Type50.Attacks {
			attacks = append(attacks, map[string]any{"index": row.Index, "class": row.Class, "value": row.Value})
		}
		armours := make([]map[string]any, 0, len(unit.Type50.Armours))
		for _, row := range unit.Type50.Armours {
			armours = append(armours, map[string]any{"index": row.Index, "class": row.Class, "value": row.Value})
		}
		fields["attacks"] = attacks
		fields["armours"] = armours
	}
	if unit.Creatable != nil {
		costs := make([]map[string]any, 0, len(unit.Creatable.Costs))
		for _, row := range unit.Creatable.Costs {
			costs = append(costs, map[string]any{"index": row.Index, "type": row.AttributeType, "amount": row.Amount, "flag": row.Flag})
		}
		fields["costs"] = costs
		locations := make([]map[string]any, 0, len(unit.Creatable.TrainLocations))
		for _, row := range unit.Creatable.TrainLocations {
			locations = append(locations, map[string]any{"index": row.Index, "train_time": row.TrainTime, "train_unit_id": row.TrainUnitID, "train_button": row.TrainButton, "train_hotkey": row.TrainHotkey})
		}
		fields["train_locations"] = locations
		fields["button_icon_id"] = unit.Creatable.ButtonIconID
		fields["button_hotkey_action"] = unit.Creatable.ButtonHotkeyAction
	}
	if unit.Building != nil {
		fields["spawning_graphic"] = unit.Building.SpawningGraphic
		fields["upgrade_graphic"] = unit.Building.UpgradeGraphic
		fields["stack_unit"] = unit.Building.StackUnit
		fields["head_unit"] = unit.Building.HeadUnit
		fields["transform_unit"] = unit.Building.TransformUnit
		fields["pile_unit"] = unit.Building.PileUnit
		fields["construction_unit"] = unit.Building.ConstructionUnit
		fields["salvage_unit"] = unit.Building.SalvageUnit
		fields["transform_sound"] = unit.Building.TransformSound
		fields["construction_sound"] = unit.Building.ConstructionSound
		fields["garrison_type"] = unit.Building.GarrisonType
		fields["garrison_heal_rate"] = unit.Building.GarrisonHealRate
		fields["garrison_repair_rate"] = unit.Building.GarrisonRepairRate
		fields["salvage_attributes"] = unit.Building.SalvageAttributes
		links := make([]map[string]any, 0, len(unit.Building.LinkedBuildings))
		for _, row := range unit.Building.LinkedBuildings {
			links = append(links, map[string]any{"index": row.Index, "unit_id": row.UnitID, "x": row.X, "y": row.Y})
			fields[fmt.Sprintf("annex_unit_%d", row.Index+1)] = row.UnitID
			fields[fmt.Sprintf("annex_unit_%d_offset_x", row.Index+1)] = row.X
			fields[fmt.Sprintf("annex_unit_%d_offset_y", row.Index+1)] = row.Y
		}
		fields["linked_buildings"] = links
	}
	return fields, nil
}

// decodeUnitFieldsLegacy is retained temporarily as a comparison aid while
// the export schema grows; it is not used by the production path.
func decodeUnitFieldsLegacy(record []byte, version string) (map[string]any, error) {
	// Keep this boundary decoder on the same implementation as the normal
	// parser. The DE unit format has several versioned opaque regions; a
	// second hand-maintained layout here would silently misalign later fields.
	c := cursor{data: record}
	unit, err := parseUnit(&c, 0, 0, versionStringAtLeast88(version))
	if err != nil {
		return nil, err
	}
	if c.off != len(record) {
		return nil, fmt.Errorf("full unit decode consumed %d of %d bytes", c.off, len(record))
	}

	encoded, err := json.Marshal(unit)
	if err != nil {
		return nil, fmt.Errorf("marshal parsed unit: %w", err)
	}
	fields := make(map[string]any, 32)
	if err := json.Unmarshal(encoded, &fields); err != nil {
		return nil, fmt.Errorf("normalize parsed unit: %w", err)
	}
	// These are the stable, useful names used by the SQL export. Keep the
	// complete parser-shaped object under parsed_unit for callers that need
	// provenance without pretending skipped bytes have semantic names.
	fields["type"] = unit.Type
	fields["id"] = unit.ID
	fields["name"] = unit.Name
	fields["class"] = unit.Class
	fields["hit_points"] = unit.HitPoints
	fields["line_of_sight"] = unit.LineOfSight
	fields["collision_size_x"] = unit.CollisionSizeX
	fields["collision_size_y"] = unit.CollisionSizeY
	fields["collision_size_z"] = unit.CollisionSizeZ
	fields["movement_type"] = unit.MovementType
	fields["enabled"] = unit.Enabled
	fields["terrain_table_id"] = unit.TerrainTableID
	if unit.Type50 != nil {
		attacks := make([]map[string]any, 0, len(unit.Type50.Attacks))
		for _, row := range unit.Type50.Attacks {
			attacks = append(attacks, map[string]any{"index": row.Index, "class": row.Class, "value": row.Value})
		}
		armours := make([]map[string]any, 0, len(unit.Type50.Armours))
		for _, row := range unit.Type50.Armours {
			armours = append(armours, map[string]any{"index": row.Index, "class": row.Class, "value": row.Value})
		}
		fields["attacks"] = attacks
		fields["armours"] = armours
		fields["max_range"] = unit.Type50.MaxRange
		fields["area_effect_range"] = unit.Type50.BlastWidth
		fields["projectile_unit"] = unit.Type50.ProjectileUnitID
		fields["fight_sprite"] = unit.Type50.AttackGraphic
		fields["blast_damage"] = unit.Type50.BlastDamage
	}
	if unit.Creatable != nil {
		costs := make([]map[string]any, 0, len(unit.Creatable.Costs))
		for _, row := range unit.Creatable.Costs {
			costs = append(costs, map[string]any{"index": row.Index, "type": row.AttributeType, "amount": row.Amount, "flag": row.Flag})
		}
		fields["costs"] = costs
		locations := make([]map[string]any, 0, len(unit.Creatable.TrainLocations))
		for _, row := range unit.Creatable.TrainLocations {
			locations = append(locations, map[string]any{"index": row.Index, "train_time": row.TrainTime, "train_unit_id": row.TrainUnitID, "train_button": row.TrainButton, "train_hotkey": row.TrainHotkey})
		}
		fields["train_locations"] = locations
		fields["button_icon_id"] = unit.Creatable.ButtonIconID
		fields["button_hotkey_action"] = unit.Creatable.ButtonHotkeyAction
	}
	if unit.Building != nil {
		links := make([]map[string]any, 0, len(unit.Building.LinkedBuildings))
		for _, row := range unit.Building.LinkedBuildings {
			links = append(links, map[string]any{"index": row.Index, "unit_id": row.UnitID, "x": row.X, "y": row.Y})
		}
		fields["linked_buildings"] = links
	}
	return fields, nil

	/*
	   c := cursor{data: record}
	   fields := make(map[string]any, 96)

	   	putI16 := func(name string) error {
	   		v, err := c.i16()
	   		if err != nil {
	   			return err
	   		}
	   		fields[name] = v
	   		return nil
	   	}

	   	putU8 := func(name string) error {
	   		v, err := c.u8()
	   		if err != nil {
	   			return err
	   		}
	   		fields[name] = v
	   		return nil
	   	}

	   	putI32 := func(name string) error {
	   		v, err := c.i32()
	   		if err != nil {
	   			return err
	   		}
	   		fields[name] = v
	   		return nil
	   	}

	   	putF32 := func(name string) error {
	   		v, err := c.f32()
	   		if err != nil {
	   			return err
	   		}
	   		fields[name] = v
	   		return nil
	   	}

	   put := func(name string, value any) { fields[name] = value }

	   typeByte, err := c.u8()

	   	if err != nil {
	   		return nil, err
	   	}

	   put("type", typeByte)

	   	if err := putI16("name_length"); err != nil {
	   		return nil, err
	   	}

	   	for _, name := range []string{"id", "string_id", "string_id_2", "unit_class", "standing_sprite_1", "standing_sprite_2", "dying_sprite", "undead_sprite"} {
	   		if err := putI16(name); err != nil {
	   			return nil, err
	   		}
	   	}

	   	for _, name := range []string{"undead_flag", "hitpoints"} {
	   		if name == "undead_flag" {
	   			if err := putU8(name); err != nil {
	   				return nil, err
	   			}
	   		} else if err := putI16(name); err != nil {
	   			return nil, err
	   		}
	   	}

	   	for _, name := range []string{"line_of_sight"} {
	   		if err := putF32(name); err != nil {
	   			return nil, err
	   		}
	   	}

	   	if err := putU8("garrison_capacity"); err != nil {
	   		return nil, err
	   	}

	   	for _, name := range []string{"radius_x", "radius_y", "radius_z"} {
	   		if err := putF32(name); err != nil {
	   			return nil, err
	   		}
	   	}

	   	for _, name := range []string{"train_sound", "damage_sound", "death_spawn"} {
	   		if err := putI16(name); err != nil {
	   			return nil, err
	   		}
	   	}

	   	for _, name := range []string{"sort_number", "can_be_built_on"} {
	   		if err := putU8(name); err != nil {
	   			return nil, err
	   		}
	   	}

	   	for _, name := range []string{"button_picture", "portrait"} {
	   		if err := putI16(name); err != nil {
	   			return nil, err
	   		}
	   		if name == "button_picture" {
	   			if err := putU8("hide_in_editor"); err != nil {
	   				return nil, err
	   			}
	   		}
	   	}

	   	for _, name := range []string{"enabled", "disabled"} {
	   		if err := putU8(name); err != nil {
	   			return nil, err
	   		}
	   	}

	   	for _, name := range []string{"tile_req_x", "tile_req_y", "center_tile_req_x", "center_tile_req_y"} {
	   		if err := putI16(name); err != nil {
	   			return nil, err
	   		}
	   	}

	   	for _, name := range []string{"construction_radius_x", "construction_radius_y"} {
	   		if err := putF32(name); err != nil {
	   			return nil, err
	   		}
	   	}

	   	for _, name := range []string{"elevation_flag", "fog_flag"} {
	   		if err := putU8(name); err != nil {
	   			return nil, err
	   		}
	   	}

	   	for _, name := range []string{"terrain_restriction_id"} {
	   		if err := putI16(name); err != nil {
	   			return nil, err
	   		}
	   	}

	   	if err := putU8("movement_type"); err != nil {
	   		return nil, err
	   	}

	   	if err := putI16("attribute_max"); err != nil {
	   		return nil, err
	   	}

	   	if err := putF32("attribute_rotation"); err != nil {
	   		return nil, err
	   	}

	   	for _, name := range []string{"area_effect_level", "combat_level", "select_level", "map_draw_level", "unit_level"} {
	   		if err := putU8(name); err != nil {
	   			return nil, err
	   		}
	   	}

	   	if err := putF32("multiple_attribute_modifier"); err != nil {
	   		return nil, err
	   	}

	   	if err := putU8("map_color"); err != nil {
	   		return nil, err
	   	}

	   	for _, name := range []string{"help_string_id", "help_page_id", "hotkey_id"} {
	   		if err := putI32(name); err != nil {
	   			return nil, err
	   		}
	   	}

	   	for _, name := range []string{"recyclable", "track_as_resource", "create_doppelganger", "resource_group", "occlusion_mask", "obstruction_type", "selection_shape"} {
	   		if err := putU8(name); err != nil {
	   			return nil, err
	   		}
	   	}

	   	if version != "VER 1.0" && version != "VER 1.1" && version != "VER 1.2" {
	   		if err := putI32("object_flags"); err != nil {
	   			return nil, err
	   		}
	   	}

	   	if err := c.skip(22); err != nil {
	   		return nil, err
	   	}

	   attrs := make([]map[string]any, 0, 3)

	   	for i := 0; i < 3; i++ {
	   		kind, err := c.u16()
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
	   		attrs = append(attrs, map[string]any{"index": i, "type": kind, "amount": amount, "flag": flag, "active": kind != 0xffff})
	   	}

	   put("attributes", attrs)
	   damageCount, err := c.u8()

	   	if err != nil {
	   		return nil, err
	   	}

	   damage := make([]map[string]any, 0, int(damageCount))

	   	for i := 0; i < int(damageCount); i++ {
	   		graphic, err := c.u16()
	   		if err != nil {
	   			return nil, err
	   		}
	   		percent, err := c.u16()
	   		if err != nil {
	   			return nil, err
	   		}
	   		flag, err := c.u8()
	   		if err != nil {
	   			return nil, err
	   		}
	   		damage = append(damage, map[string]any{"index": i, "graphic_id": graphic, "damage_percent": percent, "flag": flag})
	   	}

	   put("damage_graphics", damage)

	   	if err := c.skip(22); err != nil {
	   		return nil, err
	   	}

	   name, err := c.debugString("unit_name")

	   	if err != nil {
	   		return nil, err
	   	}

	   put("name", name.Value)

	   	for _, field := range []string{"copy_id", "group"} {
	   		if err := putI16(field); err != nil {
	   			return nil, err
	   		}
	   	}

	   	if typeByte >= 20 && typeByte != 90 {
	   		if err := putF32("speed"); err != nil {
	   			return nil, err
	   		}
	   	}

	   	if typeByte >= 30 && typeByte != 90 {
	   		for _, name := range []string{"move_sprite", "run_sprite"} {
	   			if err := putI16(name); err != nil {
	   				return nil, err
	   			}
	   		}
	   		if err := putF32("turn_speed"); err != nil {
	   			return nil, err
	   		}
	   		for _, name := range []string{"size_class"} {
	   			if err := putU8(name); err != nil {
	   				return nil, err
	   			}
	   		}
	   		if err := putI16("trailing_unit"); err != nil {
	   			return nil, err
	   		}
	   		if err := putU8("trailing_options"); err != nil {
	   			return nil, err
	   		}
	   		if err := putF32("trailing_spacing"); err != nil {
	   			return nil, err
	   		}
	   		if err := putU8("move_algorithm"); err != nil {
	   			return nil, err
	   		}
	   		for _, name := range []string{"turn_radius", "turn_radius_speed", "maximum_yaw_per_second_moving", "stationary_yaw_per_revolution_time", "maximum_yaw_per_second_stationary"} {
	   			if err := putF32(name); err != nil {
	   				return nil, err
	   			}
	   		}
	   		if err := c.skip(4); err != nil {
	   			return nil, err
	   		}
	   	}

	   	if typeByte >= 40 && typeByte != 90 {
	   		if err := putI16("default_task"); err != nil {
	   			return nil, err
	   		}
	   		if err := putF32("search_radius"); err != nil {
	   			return nil, err
	   		}
	   		if err := putF32("work_rate"); err != nil {
	   			return nil, err
	   		}
	   		for _, name := range []string{"drop_site", "backup_drop_site"} {
	   			if err := putI16(name); err != nil {
	   				return nil, err
	   			}
	   		}
	   		if err := putU8("task_by_group"); err != nil {
	   			return nil, err
	   		}
	   		for _, name := range []string{"command_sound", "move_sound"} {
	   			if err := putI16(name); err != nil {
	   				return nil, err
	   			}
	   		}
	   		if err := putU8("run_pattern"); err != nil {
	   			return nil, err
	   		}
	   		if err := c.skip(4); err != nil {
	   			return nil, err
	   		}
	   		taskCount, err := c.i16()
	   		if err != nil {
	   			return nil, err
	   		}
	   		if taskCount < 0 {
	   			return nil, fmt.Errorf("negative task count %d", taskCount)
	   		}
	   		tasks := make([]TaskSummary, 0, int(taskCount))
	   		for i := 0; i < int(taskCount); i++ {
	   			task, err := parseTask(&c, i, versionStringAtLeast88(version))
	   			if err != nil {
	   				return nil, err
	   			}
	   			tasks = append(tasks, task)
	   		}
	   		put("tasks", tasks)
	   	}

	   	if typeByte >= 50 && typeByte != 90 {
	   		if err := putI16("base_armor"); err != nil {
	   			return nil, err
	   		}
	   		weapons, err := parseFullWeapons(&c, "attacks")
	   		if err != nil {
	   			return nil, err
	   		}
	   		put("attacks", weapons)
	   		armours, err := parseFullWeapons(&c, "armours")
	   		if err != nil {
	   			return nil, err
	   		}
	   		put("armours", armours)
	   		for _, name := range []string{"defense_terrain_bonus"} {
	   			if err := putI16(name); err != nil {
	   				return nil, err
	   			}
	   		}
	   		for _, name := range []string{"max_range", "area_effect_range", "attack_speed"} {
	   			if err := putF32(name); err != nil {
	   				return nil, err
	   			}
	   		}
	   		for _, name := range []string{"projectile_unit", "base_hit_chance"} {
	   			if err := putI16(name); err != nil {
	   				return nil, err
	   			}
	   		}
	   		if err := putU8("break_off_combat"); err != nil {
	   			return nil, err
	   		}
	   		if err := putI16("frame_delay"); err != nil {
	   			return nil, err
	   		}
	   		for _, name := range []string{"weapon_offset_x", "weapon_offset_y", "weapon_offset_z"} {
	   			if err := putF32(name); err != nil {
	   				return nil, err
	   			}
	   		}
	   		if err := putU8("blast_attack_level"); err != nil {
	   			return nil, err
	   		}
	   		if err := putF32("min_range"); err != nil {
	   			return nil, err
	   		}
	   		for _, name := range []string{"missed_missile_spread"} {
	   			if err := putF32(name); err != nil {
	   				return nil, err
	   			}
	   		}
	   		for _, name := range []string{"fight_sprite", "displayed_armor", "displayed_attack"} {
	   			if err := putI16(name); err != nil {
	   				return nil, err
	   			}
	   		}
	   		for _, name := range []string{"displayed_range", "displayed_reload_time"} {
	   			if err := putF32(name); err != nil {
	   				return nil, err
	   			}
	   		}
	   	}

	   	if typeByte >= 60 && typeByte != 90 {
	   		for _, name := range []string{"missile_type", "targeting_type", "missile_hit_info", "missile_die_info", "area_effect_specials"} {
	   			if err := putU8(name); err != nil {
	   				return nil, err
	   			}
	   		}
	   		if err := putF32("ballistics_ratio"); err != nil {
	   			return nil, err
	   		}
	   	}

	   	if typeByte >= 70 && typeByte != 90 {
	   		costs := make([]map[string]any, 0, 3)
	   		for i := 0; i < 3; i++ {
	   			kind, err := c.i16()
	   			if err != nil {
	   				return nil, err
	   			}
	   			amount, err := c.i16()
	   			if err != nil {
	   				return nil, err
	   			}
	   			flag, err := c.u8()
	   			if err != nil {
	   				return nil, err
	   			}
	   			if err := c.skip(1); err != nil {
	   				return nil, err
	   			}
	   			costs = append(costs, map[string]any{"index": i, "type": kind, "amount": amount, "flag": flag})
	   		}
	   		put("costs", costs)
	   		trainCount, err := c.i16()
	   		if err != nil {
	   			return nil, err
	   		}
	   		if trainCount < 0 {
	   			return nil, fmt.Errorf("negative train location count %d", trainCount)
	   		}
	   		trainLocations := make([]map[string]any, 0, int(trainCount))
	   		for i := 0; i < int(trainCount); i++ {
	   			trainTime, err := c.i16()
	   			if err != nil {
	   				return nil, err
	   			}
	   			trainUnit, err := c.i16()
	   			if err != nil {
	   				return nil, err
	   			}
	   			trainButton, err := c.u8()
	   			if err != nil {
	   				return nil, err
	   			}
	   			trainHotkey, err := c.i32()
	   			if err != nil {
	   				return nil, err
	   			}
	   			trainLocations = append(trainLocations, map[string]any{"index": i, "train_time": trainTime, "train_unit_id": trainUnit, "train_button": trainButton, "train_hotkey": trainHotkey})
	   		}
	   		put("train_locations", trainLocations)
	   		if err := c.skip(4 + 4 + 1 + 1 + 4 + 2 + 2 + 2 + 2 + 4 + 4 + 2 + 2 + 2 + 4 + 1 + 4); err != nil {
	   			return nil, err
	   		}
	   		if err := putI16("button_icon_id"); err != nil {
	   			return nil, err
	   		}
	   		if err := c.skip(4 + 4); err != nil {
	   			return nil, err
	   		}
	   		if err := putI16("button_hotkey_action"); err != nil {
	   			return nil, err
	   		}
	   		if err := c.skip(4 + 4 + 4 + 4 + 1 + 4 + 4 + 4 + 4 + 4 + 1 + 2); err != nil {
	   			return nil, err
	   		}
	   		for _, name := range []string{"create_time", "create_at_building"} {
	   			if err := putI16(name); err != nil {
	   				return nil, err
	   			}
	   		}
	   		if err := putU8("create_button"); err != nil {
	   			return nil, err
	   		}
	   		for _, name := range []string{"rear_attack_modifier", "flank_attack_modifier"} {
	   			if err := putF32(name); err != nil {
	   				return nil, err
	   			}
	   		}
	   		if err := putU8("tribe_unit_type"); err != nil {
	   			return nil, err
	   		}
	   		if err := putU8("hero_flag"); err != nil {
	   			return nil, err
	   		}
	   		if err := putI32("garrison_sprite"); err != nil {
	   			return nil, err
	   		}
	   		if err := putF32("volley_fire_amount"); err != nil {
	   			return nil, err
	   		}
	   		if err := putU8("max_attacks_in_volley"); err != nil {
	   			return nil, err
	   		}
	   		for _, name := range []string{"volley_spread_x", "volley_spread_y", "volley_start_spread_adjustment"} {
	   			if err := putF32(name); err != nil {
	   				return nil, err
	   			}
	   		}
	   		for _, name := range []string{"volley_missile", "special_attack_sprite"} {
	   			if err := putI32(name); err != nil {
	   				return nil, err
	   			}
	   		}
	   		if err := putU8("special_attack_flag"); err != nil {
	   			return nil, err
	   		}
	   		if err := putI16("displayed_pierce_armor"); err != nil {
	   			return nil, err
	   		}
	   	}

	   	if typeByte == 80 {
	   		for _, name := range []string{"construction_sprite", "snow_sprite"} {
	   			if err := putI16(name); err != nil {
	   				return nil, err
	   			}
	   		}
	   		if err := putU8("connect_flag"); err != nil {
	   			return nil, err
	   		}
	   		if err := putI16("facet"); err != nil {
	   			return nil, err
	   		}
	   		if err := putU8("destroy_on_build"); err != nil {
	   			return nil, err
	   		}
	   		for _, name := range []string{"on_build_make_unit", "on_build_make_tile", "on_build_make_overlay", "on_build_make_tech"} {
	   			if err := putI16(name); err != nil {
	   				return nil, err
	   			}
	   		}
	   		if err := putU8("can_burn"); err != nil {
	   			return nil, err
	   		}
	   		links := make([]map[string]any, 0, 4)
	   		for i := 0; i < 4; i++ {
	   			id, err := c.i16()
	   			if err != nil {
	   				return nil, err
	   			}
	   			x, err := c.f32()
	   			if err != nil {
	   				return nil, err
	   			}
	   			y, err := c.f32()
	   			if err != nil {
	   				return nil, err
	   			}
	   			links = append(links, map[string]any{"index": i, "unit_id": id, "x": x, "y": y})
	   		}
	   		put("linked_buildings", links)
	   		for _, name := range []string{"construction_unit", "transform_unit", "transform_sound", "construction_sound"} {
	   			if err := putI16(name); err != nil {
	   				return nil, err
	   			}
	   		}
	   		if err := putU8("garrison_type"); err != nil {
	   			return nil, err
	   		}
	   		for _, name := range []string{"garrison_heal_rate", "garrison_repair_rate"} {
	   			if err := putF32(name); err != nil {
	   				return nil, err
	   			}
	   		}
	   		if err := putI16("salvage_unit"); err != nil {
	   			return nil, err
	   		}
	   		salvage := make([]int, 6)
	   		for i := range salvage {
	   			v, err := c.u8()
	   			if err != nil {
	   				return nil, err
	   			}
	   			salvage[i] = int(v)
	   		}
	   		put("salvage_attributes", salvage)
	   	}

	   	if c.off != len(record) {
	   		return nil, fmt.Errorf("full unit decode consumed %d of %d bytes", c.off, len(record))
	   	}

	   return fields, nil
	*/
}

func parseFullWeapons(c *cursor, name string) ([]map[string]any, error) {
	count, err := c.i16()
	if err != nil {
		return nil, err
	}
	if count < 0 {
		return nil, fmt.Errorf("negative %s count %d", name, count)
	}
	rows := make([]map[string]any, 0, int(count))
	for i := 0; i < int(count); i++ {
		kind, err := c.i16()
		if err != nil {
			return nil, err
		}
		value, err := c.i16()
		if err != nil {
			return nil, err
		}
		rows = append(rows, map[string]any{"index": i, "class": kind, "value": value})
	}
	return rows, nil
}
