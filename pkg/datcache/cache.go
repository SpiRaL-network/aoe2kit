package datcache

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"aoe2kit/pkg/datfile"

	_ "modernc.org/sqlite"
)

const schemaVersion = 2

type Cache struct {
	db       *sql.DB
	Path     string
	SHA256   string
	Version  string
	Built    bool
	CivCount int
}

type UnitRow struct {
	CivIndex           int
	Index              int
	Type               int
	ID                 int16
	Name               string
	Class              int16
	ClassName          string
	HitPoints          int16
	LineOfSight        float32
	CollisionSizeX     float32
	CollisionSizeY     float32
	CollisionSizeZ     float32
	ClearanceSizeX     float32
	ClearanceSizeY     float32
	TerrainTableID     int16
	MovementType       uint8
	ObstructionTypeID  uint8
	ObstructionClassID uint8
	IconID             int16
	Enabled            uint8
	HasType50          bool
	HasCreatable       bool
	RecordStart        int
	RecordEnd          int
}

type UnitQuery struct {
	Limit        int
	All          bool
	Civ          *int
	ID           *int
	Name         string
	NameContains string
	Class        string
}

func Open(datPath string) (*Cache, error) {
	if os.Getenv("AOE2KIT_DAT_CACHE_DISABLE") == "1" {
		return nil, errors.New("dat cache disabled by AOE2KIT_DAT_CACHE_DISABLE=1")
	}
	hash, err := fileSHA256(datPath)
	if err != nil {
		return nil, err
	}
	dir, err := cacheDir()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	path := filepath.Join(dir, "dat-"+hash+".sqlite")
	built := false
	if _, err := os.Stat(path); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
		if err := build(path, datPath, hash); err != nil {
			return nil, err
		}
		built = true
	}
	db, err := sql.Open("sqlite", path+"?_pragma=query_only(1)")
	if err != nil {
		return nil, err
	}
	cache := &Cache{db: db, Path: path, SHA256: hash, Built: built}
	if err := cache.loadMeta(); err != nil {
		db.Close()
		if !built {
			_ = os.Remove(path)
			if err := build(path, datPath, hash); err != nil {
				return nil, err
			}
			built = true
			db, err = sql.Open("sqlite", path+"?_pragma=query_only(1)")
			if err != nil {
				return nil, err
			}
			cache = &Cache{db: db, Path: path, SHA256: hash, Built: built}
			if err := cache.loadMeta(); err != nil {
				db.Close()
				return nil, err
			}
			return cache, nil
		}
		return nil, err
	}
	return cache, nil
}

func (c *Cache) Close() error {
	if c == nil || c.db == nil {
		return nil
	}
	return c.db.Close()
}

func (c *Cache) Unit(civID, unitID int) (datfile.UnitSummary, bool, error) {
	var blob []byte
	var start int
	err := c.db.QueryRow(`SELECT unit_record, record_start FROM units WHERE civ_index=? AND unit_index=?`, civID, unitID).Scan(&blob, &start)
	if errors.Is(err, sql.ErrNoRows) {
		return datfile.UnitSummary{}, false, nil
	}
	if err != nil {
		return datfile.UnitSummary{}, false, err
	}
	unit, err := datfile.ParseUnitRecord(blob, civID, unitID, c.Version, start)
	if err != nil {
		return datfile.UnitSummary{}, false, err
	}
	return unit, true, nil
}

func (c *Cache) Units() ([]UnitRow, error) {
	rows, _, err := c.QueryUnits(UnitQuery{All: true})
	return rows, err
}

func (c *Cache) QueryUnits(query UnitQuery) ([]UnitRow, int, error) {
	where, args := unitWhere(query)
	countSQL := `SELECT count(*) FROM units` + where
	var total int
	if err := c.db.QueryRow(countSQL, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	sqlText := `SELECT civ_index, unit_index, type, id, name, class, class_name, hit_points, line_of_sight,
collision_size_x, collision_size_y, collision_size_z, clearance_size_x, clearance_size_y, terrain_table_id,
movement_type, obstruction_type_id, obstruction_class_id, icon_id, enabled, has_type50, has_creatable,
record_start, record_end FROM units` + where + ` ORDER BY civ_index, unit_index`
	if !query.All {
		limit := query.Limit
		if limit < 1 {
			limit = 200
		}
		sqlText += ` LIMIT ?`
		args = append(args, limit)
	}
	rows, err := c.db.Query(sqlText, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := make([]UnitRow, 0)
	for rows.Next() {
		var row UnitRow
		var hasType50, hasCreatable int
		if err := rows.Scan(&row.CivIndex, &row.Index, &row.Type, &row.ID, &row.Name, &row.Class, &row.ClassName,
			&row.HitPoints, &row.LineOfSight, &row.CollisionSizeX, &row.CollisionSizeY, &row.CollisionSizeZ,
			&row.ClearanceSizeX, &row.ClearanceSizeY, &row.TerrainTableID, &row.MovementType, &row.ObstructionTypeID,
			&row.ObstructionClassID, &row.IconID, &row.Enabled, &hasType50, &hasCreatable, &row.RecordStart, &row.RecordEnd); err != nil {
			return nil, 0, err
		}
		row.HasType50 = hasType50 != 0
		row.HasCreatable = hasCreatable != 0
		out = append(out, row)
	}
	return out, total, rows.Err()
}

func unitWhere(query UnitQuery) (string, []any) {
	var clauses []string
	var args []any
	if query.Civ != nil {
		clauses = append(clauses, "civ_index=?")
		args = append(args, *query.Civ)
	}
	if query.ID != nil {
		clauses = append(clauses, "(id=? OR unit_index=?)")
		args = append(args, *query.ID, *query.ID)
	}
	if query.Name != "" {
		clauses = append(clauses, "lower(name) LIKE ?")
		args = append(args, "%"+strings.ToLower(query.Name)+"%")
	}
	if query.NameContains != "" {
		clauses = append(clauses, "lower(name) LIKE ?")
		args = append(args, "%"+strings.ToLower(query.NameContains)+"%")
	}
	switch query.Class {
	case "building":
		clauses = append(clauses, "type IN (80,90)")
	case "unit":
		clauses = append(clauses, "type NOT IN (80,90)")
	case "creatable":
		clauses = append(clauses, "has_creatable=1")
	}
	if len(clauses) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}

func (c *Cache) loadMeta() error {
	var schema int
	if err := c.db.QueryRow(`SELECT value FROM meta WHERE key='schema_version'`).Scan(&schema); err != nil {
		return err
	}
	if schema != schemaVersion {
		return fmt.Errorf("dat cache schema=%d want %d", schema, schemaVersion)
	}
	if err := c.db.QueryRow(`SELECT value FROM meta WHERE key='version'`).Scan(&c.Version); err != nil {
		return err
	}
	if err := c.db.QueryRow(`SELECT value FROM meta WHERE key='civ_count'`).Scan(&c.CivCount); err != nil {
		return err
	}
	return nil
}

func build(path, datPath, hash string) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	tmp := fmt.Sprintf("%s.%d.tmp", path, os.Getpid())
	_ = os.Remove(tmp)
	db, err := sql.Open("sqlite", tmp)
	if err != nil {
		return err
	}
	defer db.Close()
	if _, err := db.Exec(`PRAGMA journal_mode=OFF; PRAGMA synchronous=OFF; PRAGMA temp_store=MEMORY;`); err != nil {
		return err
	}
	if err := createSchema(db); err != nil {
		return err
	}
	compressed, err := os.ReadFile(datPath)
	if err != nil {
		return err
	}
	payload, err := datfile.Inflate(compressed)
	if err != nil {
		return err
	}
	idx, err := datfile.Parse(payload)
	if err != nil {
		return err
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	if err := insertMeta(tx, hash, idx); err != nil {
		tx.Rollback()
		return err
	}
	if err := insertUnits(tx, idx, payload); err != nil {
		tx.Rollback()
		return err
	}
	if err := insertTechs(tx, idx); err != nil {
		tx.Rollback()
		return err
	}
	if err := insertGraphics(tx, idx); err != nil {
		tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	if err := db.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		if _, statErr := os.Stat(path); statErr == nil {
			_ = os.Remove(tmp)
			return nil
		}
		return err
	}
	return nil
}

func createSchema(db *sql.DB) error {
	_, err := db.Exec(`CREATE TABLE meta(key TEXT PRIMARY KEY, value TEXT NOT NULL);
CREATE TABLE units(
  civ_index INTEGER NOT NULL,
  unit_index INTEGER NOT NULL,
  type INTEGER NOT NULL,
  id INTEGER NOT NULL,
  name TEXT NOT NULL,
  class INTEGER NOT NULL,
  class_name TEXT NOT NULL,
  hit_points INTEGER NOT NULL,
  line_of_sight REAL NOT NULL,
  collision_size_x REAL NOT NULL,
  collision_size_y REAL NOT NULL,
  collision_size_z REAL NOT NULL,
  clearance_size_x REAL NOT NULL,
  clearance_size_y REAL NOT NULL,
  terrain_table_id INTEGER NOT NULL,
  movement_type INTEGER NOT NULL,
  obstruction_type_id INTEGER NOT NULL,
  obstruction_class_id INTEGER NOT NULL,
  icon_id INTEGER NOT NULL,
  enabled INTEGER NOT NULL,
  has_type50 INTEGER NOT NULL,
  has_creatable INTEGER NOT NULL,
  record_start INTEGER NOT NULL,
  record_end INTEGER NOT NULL,
  unit_record BLOB NOT NULL,
  PRIMARY KEY(civ_index, unit_index)
);
CREATE INDEX units_by_id ON units(id);
CREATE INDEX units_by_collision ON units(collision_size_x, collision_size_y, collision_size_z);
CREATE TABLE techs(
  tech_index INTEGER PRIMARY KEY,
  name TEXT NOT NULL,
  civ INTEGER NOT NULL,
  effect_id INTEGER NOT NULL,
  type INTEGER NOT NULL,
  icon_id INTEGER NOT NULL
);
CREATE TABLE graphics(
  graphic_index INTEGER PRIMARY KEY,
  present INTEGER NOT NULL,
  name TEXT NOT NULL,
  slp INTEGER NOT NULL,
  angle_count INTEGER NOT NULL,
  frame_count INTEGER NOT NULL
);`)
	return err
}

func insertMeta(tx *sql.Tx, hash string, idx *datfile.Index) error {
	for _, item := range [][2]string{
		{"schema_version", fmt.Sprintf("%d", schemaVersion)},
		{"dat_sha256", hash},
		{"version", idx.Version},
		{"civ_count", fmt.Sprintf("%d", len(idx.Civs))},
	} {
		if _, err := tx.Exec(`INSERT INTO meta(key,value) VALUES(?,?)`, item[0], item[1]); err != nil {
			return err
		}
	}
	return nil
}

func insertUnits(tx *sql.Tx, idx *datfile.Index, payload []byte) error {
	stmt, err := tx.Prepare(`INSERT INTO units(civ_index, unit_index, type, id, name, class, class_name, hit_points, line_of_sight,
collision_size_x, collision_size_y, collision_size_z, clearance_size_x, clearance_size_y, terrain_table_id,
movement_type, obstruction_type_id, obstruction_class_id, icon_id, enabled, has_type50, has_creatable,
record_start, record_end, unit_record) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for _, civ := range idx.Civs {
		for _, unit := range civ.Units {
			if !unit.Present {
				continue
			}
			if unit.RecordStart < 0 || unit.RecordEnd < unit.RecordStart || unit.RecordEnd > len(payload) {
				return fmt.Errorf("unit civ=%d id=%d has invalid record span %d..%d", unit.CivIndex, unit.Index, unit.RecordStart, unit.RecordEnd)
			}
			record := payload[unit.RecordStart:unit.RecordEnd]
			if _, err := stmt.Exec(unit.CivIndex, unit.Index, unit.Type, unit.ID, unit.Name, unit.Class, unit.ClassName,
				unit.HitPoints, unit.LineOfSight, unit.CollisionSizeX, unit.CollisionSizeY, unit.CollisionSizeZ,
				unit.ClearanceSizeX, unit.ClearanceSizeY, unit.TerrainTableID, unit.MovementType, unit.ObstructionTypeID,
				unit.ObstructionClassID, unit.IconID, unit.Enabled, boolInt(unit.Type50 != nil), boolInt(unit.Creatable != nil),
				unit.RecordStart, unit.RecordEnd, record); err != nil {
				return err
			}
		}
	}
	return nil
}

func insertTechs(tx *sql.Tx, idx *datfile.Index) error {
	stmt, err := tx.Prepare(`INSERT INTO techs(tech_index, name, civ, effect_id, type, icon_id) VALUES(?,?,?,?,?,?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for _, tech := range idx.Techs {
		if _, err := stmt.Exec(tech.Index, tech.Name, tech.Civ, tech.EffectID, tech.Type, tech.IconID); err != nil {
			return err
		}
	}
	return nil
}

func insertGraphics(tx *sql.Tx, idx *datfile.Index) error {
	stmt, err := tx.Prepare(`INSERT INTO graphics(graphic_index, present, name, slp, angle_count, frame_count) VALUES(?,?,?,?,?,?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for _, graphic := range idx.Graphics {
		summary := graphic.Summary()
		if _, err := stmt.Exec(graphic.Index, boolInt(graphic.Present), summary.Name, summary.SLP, summary.AngleCount, summary.FrameCount); err != nil {
			return err
		}
	}
	return nil
}

func cacheDir() (string, error) {
	if dir := strings.TrimSpace(os.Getenv("AOE2KIT_DAT_CACHE_DIR")); dir != "" {
		return dir, nil
	}
	root, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "aoe2kit", "dat-cache"), nil
}

func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
