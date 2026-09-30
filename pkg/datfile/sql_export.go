package datfile

import (
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"strings"

	_ "modernc.org/sqlite"
)

// ExportSQLite writes a query-oriented view of every present DAT unit. Scalar
// decoded fields become columns; variable-length attacks, armours, attributes,
// and damage graphics are child tables. The source DAT remains untouched.
func ExportSQLite(datPath, outPath string) error {
	compressed, err := os.ReadFile(datPath)
	if err != nil {
		return err
	}
	payload, err := Inflate(compressed)
	if err != nil {
		return err
	}
	idx, err := Parse(payload)
	if err != nil {
		return err
	}
	fields := exportScalarFields()

	tmp := outPath + ".tmp-" + strconv.Itoa(os.Getpid())
	db, err := sql.Open("sqlite", tmp)
	if err != nil {
		return err
	}
	defer db.Close()
	if _, err := db.Exec(`PRAGMA journal_mode=OFF; PRAGMA synchronous=OFF;`); err != nil {
		return err
	}
	if err := createExportSchema(db, fields); err != nil {
		return err
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	rollback := func(e error) error { _ = tx.Rollback(); return e }
	if _, err := tx.Exec(`INSERT INTO metadata(key,value) VALUES(?,?)`, "dat_version", idx.Version); err != nil {
		return rollback(err)
	}
	if _, err := tx.Exec(`INSERT INTO metadata(key,value) VALUES(?,?)`, "source_path", datPath); err != nil {
		return rollback(err)
	}
	if err := insertExportAttributeMap(tx); err != nil {
		return rollback(err)
	}
	coverage := exportAttributeCoverage()
	if err := insertExportCoverageMetadata(tx, coverage); err != nil {
		return rollback(err)
	}

	columns := append([]string{"civ_index", "unit_index", "unit_id", "type", "name", "class", "class_name", "record_start", "record_end"}, fields...)
	quoted := make([]string, len(columns))
	placeholders := make([]string, len(columns))
	for i, column := range columns {
		quoted[i] = quoteIdentifier(column)
		placeholders[i] = "?"
	}
	unitStmt, err := tx.Prepare("INSERT INTO units(" + strings.Join(quoted, ",") + ") VALUES(" + strings.Join(placeholders, ",") + ")")
	if err != nil {
		return rollback(err)
	}
	defer unitStmt.Close()
	fieldStmt, err := tx.Prepare(`INSERT INTO unit_fields(civ_index,unit_index,field_name,field_value) VALUES(?,?,?,?)`)
	if err != nil {
		return rollback(err)
	}
	defer fieldStmt.Close()
	attributeStmt, err := tx.Prepare(`INSERT INTO unit_attributes(civ_index,unit_index,attribute_index,attribute_type,amount,flag,active) VALUES(?,?,?,?,?,?,?)`)
	if err != nil {
		return rollback(err)
	}
	defer attributeStmt.Close()
	attackStmt, err := tx.Prepare(`INSERT INTO unit_attacks(civ_index,unit_index,row_index,class,value) VALUES(?,?,?,?,?)`)
	if err != nil {
		return rollback(err)
	}
	defer attackStmt.Close()
	armourStmt, err := tx.Prepare(`INSERT INTO unit_armours(civ_index,unit_index,row_index,class,value) VALUES(?,?,?,?,?)`)
	if err != nil {
		return rollback(err)
	}
	defer armourStmt.Close()

	for _, civ := range idx.Civs {
		for _, unit := range civ.Units {
			if !unit.Present || unit.RecordStart < 0 || unit.RecordEnd > len(payload) || unit.RecordEnd < unit.RecordStart {
				continue
			}
			record := payload[unit.RecordStart:unit.RecordEnd]
			full, err := DecodeUnitFields(record, idx.Version)
			if err != nil {
				return rollback(fmt.Errorf("civ %d unit %d (%s): %w", civ.Index, unit.Index, unit.Name, err))
			}
			values := []any{civ.Index, unit.Index, unit.ID, unit.Type, unit.Name, unit.Class, unit.ClassName, unit.RecordStart, unit.RecordEnd}
			for _, field := range fields {
				value, ok := full[field]
				if !ok {
					values = append(values, nil)
				} else {
					values = append(values, fmt.Sprint(value))
				}
			}
			if _, err := unitStmt.Exec(values...); err != nil {
				return rollback(err)
			}
			for name, value := range full {
				if isScalarSQLValue(value) {
					if _, err := fieldStmt.Exec(civ.Index, unit.Index, name, fmt.Sprint(value)); err != nil {
						return rollback(err)
					}
				}
			}
			if attrs, ok := full["attributes"].([]map[string]any); ok {
				for _, attr := range attrs {
					if _, err := attributeStmt.Exec(civ.Index, unit.Index, attr["index"], attr["type"], attr["amount"], attr["flag"], attr["active"]); err != nil {
						return rollback(err)
					}
				}
			}
			if err := insertWeaponRows(attackStmt, civ.Index, unit.Index, full["attacks"]); err != nil {
				return rollback(err)
			}
			if err := insertWeaponRows(armourStmt, civ.Index, unit.Index, full["armours"]); err != nil {
				return rollback(err)
			}
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	if err := db.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, outPath)
}

func createExportSchema(db *sql.DB, fields []string) error {
	columns := make([]string, 0, len(fields))
	for _, field := range fields {
		columns = append(columns, quoteIdentifier(field)+" TEXT")
	}
	_, err := db.Exec(`CREATE TABLE metadata(key TEXT PRIMARY KEY,value TEXT NOT NULL);
CREATE TABLE attribute_map(attribute_id INTEGER PRIMARY KEY,attribute_name TEXT NOT NULL,dat_field_name TEXT,confidence TEXT NOT NULL,exported INTEGER NOT NULL);
CREATE TABLE unit_fields(civ_index INTEGER NOT NULL,unit_index INTEGER NOT NULL,field_name TEXT NOT NULL,field_value TEXT,PRIMARY KEY(civ_index,unit_index,field_name));
CREATE TABLE unit_attributes(civ_index INTEGER NOT NULL,unit_index INTEGER NOT NULL,attribute_index INTEGER NOT NULL,attribute_type INTEGER NOT NULL,amount REAL NOT NULL,flag INTEGER NOT NULL,active INTEGER NOT NULL);
CREATE TABLE unit_attacks(civ_index INTEGER NOT NULL,unit_index INTEGER NOT NULL,row_index INTEGER NOT NULL,class INTEGER NOT NULL,value INTEGER NOT NULL);
CREATE TABLE unit_armours(civ_index INTEGER NOT NULL,unit_index INTEGER NOT NULL,row_index INTEGER NOT NULL,class INTEGER NOT NULL,value INTEGER NOT NULL);`)
	if err != nil {
		return err
	}
	createUnits := `CREATE TABLE units(civ_index INTEGER NOT NULL,unit_index INTEGER NOT NULL,unit_id INTEGER NOT NULL,type INTEGER NOT NULL,name TEXT NOT NULL,class INTEGER NOT NULL,class_name TEXT NOT NULL,record_start INTEGER NOT NULL,record_end INTEGER NOT NULL`
	if len(columns) > 0 {
		createUnits += "," + strings.Join(columns, ",")
	}
	_, err = db.Exec(createUnits + ",PRIMARY KEY(civ_index,unit_index));")
	return err
}

func exportScalarFields() []string {
	return []string{
		"area_effect_range", "blast_damage", "blood_unit_id", "button_hotkey_action", "button_icon_id",
		"clearance_size_x", "clearance_size_y", "collision_size_x", "collision_size_y", "collision_size_z",
		"death_spawn", "dying_graphic", "enabled", "fight_sprite", "garrison_capacity", "hit_points", "icon_id", "line_of_sight", "max_range",
		"fly_mode", "movement_type", "obstruction_class_id", "obstruction_type_id", "projectile_unit", "standing_graphic_1",
		"standing_graphic_2", "terrain_table_id", "train_sound", "damage_sound", "undead_graphic", "undead_flag",
	}
}

func insertWeaponRows(stmt *sql.Stmt, civ, unit int, value any) error {
	rows, ok := value.([]map[string]any)
	if !ok {
		return nil
	}
	for _, row := range rows {
		if _, err := stmt.Exec(civ, unit, row["index"], row["class"], row["value"]); err != nil {
			return err
		}
	}
	return nil
}

func isScalarSQLValue(value any) bool {
	switch value.(type) {
	case string, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64, bool:
		return true
	default:
		return false
	}
}

func quoteIdentifier(value string) string {
	return `"` + strings.ReplaceAll(value, `"`, `""`) + `"`
}

func insertExportAttributeMap(tx *sql.Tx) error {
	exported := exportedDATFields()
	for id := 0; id <= 221; id++ {
		name := UGCAttributeName(id)
		field, confidence := UGCAttributeDATField(id)
		isExported := 0
		if exportFieldAvailable(exported, field) {
			isExported = 1
		}
		if _, err := tx.Exec(`INSERT INTO attribute_map(attribute_id,attribute_name,dat_field_name,confidence,exported) VALUES(?,?,?,?,?)`, id, name, field, confidence, isExported); err != nil {
			return err
		}
	}
	return nil
}

func exportedDATFields() map[string]bool {
	fields := make(map[string]bool)
	for _, field := range exportScalarFields() {
		fields[field] = true
	}
	// These are scalar keys emitted by DecodeUnitFields into unit_fields. The
	// SQL schema intentionally stores them sparsely because not every unit type
	// carries every layout block.
	for _, field := range []string{
		"attacks", "armours",
		"hit_points", "line_of_sight", "garrison_capacity", "radius_x", "radius_y", "speed", "turn_speed",
		"displayed_reload_time", "work_rate", "carry_capacity", "base_armor", "min_range", "frame_delay",
		"blast_attack_level", "displayed_attack", "displayed_range", "terrain_restriction_id", "size_class",
		"missed_missile_spread", "ballistics_ratio", "move_sprite", "standing_graphic_1", "standing_graphic_2",
		"dying_graphic", "blood_unit_id", "move_sprite", "run_sprite", "trailing_unit", "trailing_options", "trailing_spacing",
		"move_algorithm", "area_effect_specials", "blast_damage", "attack_speed", "base_hit_chance",
		"break_off_combat", "weapon_offset_x", "weapon_offset_y", "weapon_offset_z", "displayed_armor",
		"maximum_yaw_per_second_moving", "stationary_yaw_per_revolution_time", "maximum_yaw_per_second_stationary",
		"spawning_graphic", "upgrade_graphic", "head_unit", "stack_unit", "transform_unit", "pile_unit",
		"construction_unit", "salvage_unit", "transform_sound", "construction_sound", "garrison_type", "garrison_heal_rate", "garrison_repair_rate",
		"annex_unit_1", "annex_unit_1_offset_x", "annex_unit_1_offset_y", "annex_unit_2", "annex_unit_2_offset_x", "annex_unit_2_offset_y",
		"annex_unit_3", "annex_unit_3_offset_x", "annex_unit_3_offset_y", "annex_unit_4", "annex_unit_4_offset_x", "annex_unit_4_offset_y",
	} {
		fields[field] = true
	}
	return fields
}

func exportFieldAvailable(fields map[string]bool, path string) bool {
	if path == "" {
		return false
	}
	if strings.HasSuffix(path, ".value") {
		return fields[strings.TrimSuffix(path, ".value")]
	}
	return fields[path]
}

type attributeExportCoverage struct {
	Total      int
	Mapped     int
	Exported   int
	Unexported []string
}

func exportAttributeCoverage() attributeExportCoverage {
	fields := exportedDATFields()
	coverage := attributeExportCoverage{}
	for id := 0; id <= 221; id++ {
		coverage.Total++
		field, confidence := UGCAttributeDATField(id)
		if confidence == "unmapped" || field == "" {
			continue
		}
		coverage.Mapped++
		if !exportFieldAvailable(fields, field) {
			coverage.Unexported = append(coverage.Unexported, fmt.Sprintf("%d:%s", id, field))
			continue
		}
		coverage.Exported++
	}
	return coverage
}

func insertExportCoverageMetadata(tx *sql.Tx, coverage attributeExportCoverage) error {
	if _, err := tx.Exec(`INSERT INTO metadata(key,value) VALUES(?,?)`, "attribute_map_total", coverage.Total); err != nil {
		return err
	}
	if _, err := tx.Exec(`INSERT INTO metadata(key,value) VALUES(?,?)`, "attribute_map_mapped", coverage.Mapped); err != nil {
		return err
	}
	if _, err := tx.Exec(`INSERT INTO metadata(key,value) VALUES(?,?)`, "attribute_map_exported", coverage.Exported); err != nil {
		return err
	}
	if _, err := tx.Exec(`INSERT INTO metadata(key,value) VALUES(?,?)`, "attribute_map_unexported", len(coverage.Unexported)); err != nil {
		return err
	}
	if _, err := tx.Exec(`INSERT INTO metadata(key,value) VALUES(?,?)`, "attribute_map_unexported_fields", strings.Join(coverage.Unexported, ",")); err != nil {
		return err
	}
	if len(coverage.Unexported) > 0 {
		fmt.Fprintf(os.Stderr, "warning: DAT SQL export has %d mapped attributes without decoded fields; query attribute_map.exported=1 before using them\n", len(coverage.Unexported))
	}
	return nil
}

// UGCAttributeName is the stable public name used by the UGC attribute page.
func UGCAttributeName(id int) string {
	// The UGC guide assigns these ids to unit attributes. They overlap the
	// player-score names in EffectAttributeName because the engine uses the
	// same numeric namespace with different command contexts.
	switch id {
	case 185:
		return "fly_mode"
	case 186:
		return "can_be_gathered"
	case 187:
		return "hill_mode"
	}
	if name := EffectAttributeName(int16(id)); name != "" {
		return name
	}
	return fmt.Sprintf("attribute_%d", id)
}

func UGCAttributeDATField(id int) (string, string) {
	fields := map[int]string{
		0: "hit_points", 1: "line_of_sight", 2: "garrison_capacity", 3: "radius_x", 4: "radius_y", 5: "speed",
		6: "turn_speed", 8: "attacks.value", 9: "attacks.value", 10: "displayed_reload_time", 12: "max_range",
		13: "work_rate", 14: "carry_capacity", 15: "base_armor", 16: "projectile_unit", 20: "min_range", 22: "area_effect_range",
		23: "search_radius", 32: "radius_z", 33: "can_be_built_on", 34: "on_build_make_tile", 41: "frame_delay",
		42: "create_at_building", 43: "create_button", 44: "blast_attack_level", 45: "area_effect_level", 46: "displayed_attack",
		47: "displayed_range", 48: "armours.value", 49: "armours.value", 50: "string_id", 51: "string_id_2",
		53: "terrain_restriction_id", 54: "unit_level", 56: "attribute_piece", 57: "death_spawn", 58: "hotkey_id",
		64: "missed_missile_spread", 65: "secondary_projectile_unit", 66: "blood_unit_id", 69: "ballistics_ratio", 70: "fight_sprite",
		71: "standing_graphic_1", 72: "standing_graphic_2", 73: "dying_graphic", 74: "undead_sprite", 75: "move_sprite",
		76: "run_sprite", 78: "obstruction_type", 80: "selection_shape", 82: "fight_sprite", 84: "garrison_sprite",
		85: "construction_sprite", 86: "snow_sprite", 91: "damage_graphics", 96: "train_sound", 98: "damage_sound",
		100: "costs", 101: "create_time", 102: "max_attacks_in_volley", 108: "garrison_heal_rate", 109: "regeneration_rate",
		115: "blast_damage", 119: "area_effect_specials", 145: "trailing_unit", 146: "trailing_options",
		147: "trailing_spacing", 162: "charge_target", 163: "size_class", 197: "outline_radius_x", 198: "outline_radius_y",
		185: "fly_mode", 186: "can_be_gathered", 187: "hill_mode",
		183: "spawning_graphic", 184: "upgrade_graphic", 202: "stack_unit", 203: "head_unit", 204: "transform_unit", 205: "pile_unit",
		206: "annex_unit_1", 207: "annex_unit_2", 208: "annex_unit_3", 209: "annex_unit_4",
		210: "annex_unit_1_offset_x", 211: "annex_unit_1_offset_y", 212: "annex_unit_2_offset_x", 213: "annex_unit_2_offset_y",
		214: "annex_unit_3_offset_x", 215: "annex_unit_3_offset_y", 216: "annex_unit_4_offset_x", 217: "annex_unit_4_offset_y",
		199: "outline_radius_z", 200: "clearance_size_x", 201: "clearance_size_y", 218: "move_algorithm",
		220: "can_burn",
	}
	if field, ok := fields[id]; ok {
		return field, "layout_correlated"
	}
	return "", "unmapped"
}
