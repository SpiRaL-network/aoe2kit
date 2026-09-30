package datfile

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

type DiffOptions struct {
	Limit    int
	All      bool
	Sections []string
}

type DiffReport struct {
	Base         string        `json:"base"`
	Mod          string        `json:"mod"`
	Verification string        `json:"verification"`
	Same         bool          `json:"same"`
	Summary      DiffSummary   `json:"summary"`
	Sections     []DiffSection `json:"sections,omitempty"`
	Warnings     []string      `json:"warnings,omitempty"`
}

type DiffSummary struct {
	SectionsCompared int  `json:"sections_compared"`
	Added            int  `json:"added"`
	Removed          int  `json:"removed"`
	Changed          int  `json:"changed"`
	Emitted          int  `json:"emitted"`
	Limit            int  `json:"limit"`
	Truncated        bool `json:"truncated"`
}

type DiffSection struct {
	Name      string      `json:"name"`
	BaseCount int         `json:"base_count"`
	ModCount  int         `json:"mod_count"`
	Added     int         `json:"added"`
	Removed   int         `json:"removed"`
	Changed   int         `json:"changed"`
	Entries   []DiffEntry `json:"entries,omitempty"`
	Truncated bool        `json:"truncated,omitempty"`
}

type DiffEntry struct {
	Kind       string `json:"kind"`
	Section    string `json:"section"`
	Key        string `json:"key"`
	Name       string `json:"name,omitempty"`
	BaseSHA256 string `json:"base_sha256,omitempty"`
	ModSHA256  string `json:"mod_sha256,omitempty"`
}

type diffItem struct {
	Key  string
	Name string
	Hash string
}

func DiffFiles(basePath, modPath string, opts DiffOptions) (*DiffReport, error) {
	base, err := Open(basePath)
	if err != nil {
		return nil, err
	}
	mod, err := Open(modPath)
	if err != nil {
		return nil, err
	}
	if opts.Limit == 0 && !opts.All {
		opts.Limit = 200
	}
	report := &DiffReport{
		Base:         basePath,
		Mod:          modPath,
		Verification: "structure_verified_decoded_dat_sections_span_stripped",
		Warnings:     []string{"dat diff opens and decodes both full DAT files; expect high memory/time on current-DE empires*.dat until section-streaming parsers exist"},
	}
	allSections := []struct {
		name  string
		build func(*Index) []diffItem
	}{
		{"graphics", func(idx *Index) []diffItem { return diffGraphics(idx.Graphics) }},
		{"effects", func(idx *Index) []diffItem { return diffEffects(idx.Effects) }},
		{"unit_headers", func(idx *Index) []diffItem { return diffUnitHeaders(idx.UnitHeaders) }},
		{"techs", func(idx *Index) []diffItem { return diffTechs(idx.Techs) }},
		{"civs", func(idx *Index) []diffItem { return diffCivs(idx.Civs) }},
		{"units", func(idx *Index) []diffItem { return diffUnits(idx.Civs) }},
		{"terrain_restrictions", func(idx *Index) []diffItem { return diffTerrainRestrictions(idx.TerrainRestrictions) }},
		{"terrains", func(idx *Index) []diffItem { return diffTerrains(idx.Terrains) }},
		{"sounds", func(idx *Index) []diffItem { return diffSounds(idx.Sounds) }},
		{"player_colours", func(idx *Index) []diffItem { return diffPlayerColours(idx.PlayerColours) }},
		{"random_maps", func(idx *Index) []diffItem { return diffRandomMaps(idx.RandomMaps) }},
	}
	selected := datDiffSelectedSections(opts.Sections)
	if len(opts.Sections) == 0 {
		report.Warnings = append(report.Warnings, "per-civ units are skipped by default because that section is large; pass --section units or --section all when unit-record diffing is intentional")
	}
	for _, section := range allSections {
		if !selected[section.name] {
			continue
		}
		diff := diffSection(section.name, section.build(base), section.build(mod))
		report.Summary.SectionsCompared++
		report.Summary.Added += diff.Added
		report.Summary.Removed += diff.Removed
		report.Summary.Changed += diff.Changed
		if len(diff.Entries) > 0 {
			if !opts.All && len(diff.Entries) > opts.Limit {
				diff.Entries = diff.Entries[:opts.Limit]
				diff.Truncated = true
				report.Summary.Truncated = true
			}
			report.Summary.Emitted += len(diff.Entries)
		}
		report.Sections = append(report.Sections, diff)
	}
	report.Summary.Limit = opts.Limit
	report.Same = report.Summary.Added == 0 && report.Summary.Removed == 0 && report.Summary.Changed == 0
	return report, nil
}

func datDiffSelectedSections(sections []string) map[string]bool {
	defaults := []string{"graphics", "effects", "unit_headers", "techs", "civs", "terrain_restrictions", "terrains", "sounds", "player_colours", "random_maps"}
	all := append(append([]string{}, defaults...), "units")
	selected := map[string]bool{}
	if len(sections) == 0 {
		for _, section := range defaults {
			selected[section] = true
		}
		return selected
	}
	for _, section := range sections {
		name := strings.ToLower(strings.TrimSpace(section))
		name = strings.ReplaceAll(name, "-", "_")
		switch name {
		case "all":
			for _, candidate := range all {
				selected[candidate] = true
			}
		case "graphics", "effects", "unit_headers", "techs", "civs", "units", "terrain_restrictions", "terrains", "sounds", "player_colours", "random_maps":
			selected[name] = true
		}
	}
	return selected
}

func diffSection(name string, baseItems, modItems []diffItem) DiffSection {
	base := diffMap(baseItems)
	mod := diffMap(modItems)
	out := DiffSection{Name: name, BaseCount: len(base), ModCount: len(mod)}
	for key, baseItem := range base {
		modItem, ok := mod[key]
		if !ok {
			out.Removed++
			out.Entries = append(out.Entries, DiffEntry{Kind: "removed", Section: name, Key: key, Name: baseItem.Name, BaseSHA256: baseItem.Hash})
			continue
		}
		if baseItem.Hash != modItem.Hash {
			out.Changed++
			out.Entries = append(out.Entries, DiffEntry{Kind: "changed", Section: name, Key: key, Name: firstNonEmpty(modItem.Name, baseItem.Name), BaseSHA256: baseItem.Hash, ModSHA256: modItem.Hash})
		}
	}
	for key, modItem := range mod {
		if _, ok := base[key]; ok {
			continue
		}
		out.Added++
		out.Entries = append(out.Entries, DiffEntry{Kind: "added", Section: name, Key: key, Name: modItem.Name, ModSHA256: modItem.Hash})
	}
	sort.Slice(out.Entries, func(i, j int) bool {
		if out.Entries[i].Kind == out.Entries[j].Kind {
			return out.Entries[i].Key < out.Entries[j].Key
		}
		return out.Entries[i].Kind < out.Entries[j].Kind
	})
	return out
}

func diffMap(items []diffItem) map[string]diffItem {
	out := make(map[string]diffItem, len(items))
	for _, item := range items {
		out[item.Key] = item
	}
	return out
}

func diffGraphics(items []Graphic) []diffItem {
	out := make([]diffItem, 0, len(items))
	for _, item := range items {
		out = append(out, diffItem{Key: strconv.Itoa(item.Index), Name: item.Name.Value, Hash: diffHash(item)})
	}
	return out
}

func diffEffects(items []Effect) []diffItem {
	out := make([]diffItem, 0, len(items))
	for _, item := range items {
		out = append(out, diffItem{Key: strconv.Itoa(item.Index), Name: item.Name, Hash: diffHash(item)})
	}
	return out
}

func diffUnitHeaders(items []UnitHeader) []diffItem {
	out := make([]diffItem, 0, len(items))
	for _, item := range items {
		out = append(out, diffItem{Key: strconv.Itoa(item.Index), Hash: diffHash(item)})
	}
	return out
}

func diffTechs(items []Tech) []diffItem {
	out := make([]diffItem, 0, len(items))
	for _, item := range items {
		out = append(out, diffItem{Key: strconv.Itoa(item.Index), Name: item.Name, Hash: diffHash(item)})
	}
	return out
}

func diffCivs(items []Civ) []diffItem {
	out := make([]diffItem, 0, len(items))
	for _, item := range items {
		copy := item
		copy.Units = nil
		out = append(out, diffItem{Key: strconv.Itoa(item.Index), Name: item.Name, Hash: diffHash(copy)})
	}
	return out
}

func diffUnits(civs []Civ) []diffItem {
	var out []diffItem
	for _, civ := range civs {
		for _, unit := range civ.Units {
			if !unit.Present {
				continue
			}
			key := fmt.Sprintf("%d:%d", civ.Index, unit.Index)
			name := fmt.Sprintf("civ=%d unit=%d", civ.Index, unit.Index)
			if unit.Name != "" {
				name += " " + unit.Name
			}
			out = append(out, diffItem{Key: key, Name: name, Hash: diffHash(unitDiffSignature(unit))})
		}
	}
	return out
}

type unitDiffSig struct {
	Present            bool               `json:"present"`
	Type               int                `json:"type"`
	ID                 int16              `json:"id"`
	Name               string             `json:"name"`
	Class              int16              `json:"class"`
	HitPoints          int16              `json:"hit_points"`
	LineOfSight        float32            `json:"line_of_sight"`
	CollisionSizeX     float32            `json:"collision_size_x"`
	CollisionSizeY     float32            `json:"collision_size_y"`
	CollisionSizeZ     float32            `json:"collision_size_z"`
	ClearanceSizeX     float32            `json:"clearance_size_x"`
	ClearanceSizeY     float32            `json:"clearance_size_y"`
	TerrainTableID     int16              `json:"terrain_table_id"`
	MovementType       uint8              `json:"movement_type"`
	ObstructionTypeID  uint8              `json:"obstruction_type_id"`
	ObstructionClassID uint8              `json:"obstruction_class_id"`
	StandingGraphic1   int16              `json:"standing_graphic_1"`
	StandingGraphic2   int16              `json:"standing_graphic_2"`
	DyingGraphic       int16              `json:"dying_graphic"`
	BloodUnitID        int16              `json:"blood_unit_id"`
	IconID             int16              `json:"icon_id"`
	Enabled            uint8              `json:"enabled"`
	Attributes         []unitAttributeSig `json:"attributes,omitempty"`
	DamageGraphics     []damageGraphicSig `json:"damage_graphics,omitempty"`
	Action             *actionSig         `json:"action,omitempty"`
	Type50             *type50Sig         `json:"type50,omitempty"`
	Creatable          *creatableSig      `json:"creatable,omitempty"`
	Building           *buildingSig       `json:"building,omitempty"`
}

type unitAttributeSig struct {
	Index         int     `json:"index"`
	AttributeType uint16  `json:"attribute_type"`
	Amount        float32 `json:"amount"`
	Flag          uint8   `json:"flag"`
	Active        bool    `json:"active"`
}

type damageGraphicSig struct {
	Index         int    `json:"index"`
	GraphicID     uint16 `json:"graphic_id"`
	DamagePercent uint16 `json:"damage_percent"`
	Flag          uint8  `json:"flag"`
}

type actionSig struct {
	DropSites []int16   `json:"drop_sites,omitempty"`
	Tasks     []taskSig `json:"tasks,omitempty"`
}

type taskSig struct {
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
}

type type50Sig struct {
	ProjectileUnitID int16       `json:"projectile_unit_id"`
	MaxRange         float32     `json:"max_range"`
	BlastWidth       float32     `json:"blast_width"`
	AttackGraphic    int16       `json:"attack_graphic"`
	BlastDamage      float32     `json:"blast_damage"`
	Attacks          []weaponSig `json:"attacks,omitempty"`
	Armours          []weaponSig `json:"armours,omitempty"`
}

type weaponSig struct {
	Index int   `json:"index"`
	Class int16 `json:"class"`
	Value int16 `json:"value"`
}

type creatableSig struct {
	TrainLocationCount int                `json:"train_location_count"`
	TrainTime0         int16              `json:"train_time_0"`
	TrainUnitID0       int16              `json:"train_unit_id_0"`
	TrainButtonID0     uint8              `json:"train_button_id_0"`
	TrainHotkeyID0     int32              `json:"train_hotkey_id_0"`
	ButtonIconID       int16              `json:"button_icon_id"`
	ButtonHotkeyAction int16              `json:"button_hotkey_action"`
	Costs              []attributeCostSig `json:"costs,omitempty"`
	TrainLocations     []trainLocationSig `json:"train_locations,omitempty"`
}

type attributeCostSig struct {
	Index         int   `json:"index"`
	AttributeType int16 `json:"attribute_type"`
	Amount        int16 `json:"amount"`
	Flag          uint8 `json:"flag"`
	Padding       uint8 `json:"padding"`
	Active        bool  `json:"active"`
}

type trainLocationSig struct {
	Index       int   `json:"index"`
	TrainTime   int16 `json:"train_time"`
	TrainUnitID int16 `json:"train_unit_id"`
	TrainButton uint8 `json:"train_button"`
	TrainHotkey int32 `json:"train_hotkey"`
}

type buildingSig struct {
	LinkedBuildings []linkedBuildingSig `json:"linked_buildings,omitempty"`
}

type linkedBuildingSig struct {
	Index  int     `json:"index"`
	UnitID int16   `json:"unit_id"`
	X      float32 `json:"x"`
	Y      float32 `json:"y"`
	Active bool    `json:"active"`
}

func unitDiffSignature(unit UnitSummary) unitDiffSig {
	sig := unitDiffSig{
		Present:            unit.Present,
		Type:               unit.Type,
		ID:                 unit.ID,
		Name:               unit.Name,
		Class:              unit.Class,
		HitPoints:          unit.HitPoints,
		LineOfSight:        unit.LineOfSight,
		CollisionSizeX:     unit.CollisionSizeX,
		CollisionSizeY:     unit.CollisionSizeY,
		CollisionSizeZ:     unit.CollisionSizeZ,
		ClearanceSizeX:     unit.ClearanceSizeX,
		ClearanceSizeY:     unit.ClearanceSizeY,
		TerrainTableID:     unit.TerrainTableID,
		MovementType:       unit.MovementType,
		ObstructionTypeID:  unit.ObstructionTypeID,
		ObstructionClassID: unit.ObstructionClassID,
		StandingGraphic1:   unit.StandingGraphic1,
		StandingGraphic2:   unit.StandingGraphic2,
		DyingGraphic:       unit.DyingGraphic,
		BloodUnitID:        unit.BloodUnitID,
		IconID:             unit.IconID,
		Enabled:            unit.Enabled,
	}
	for _, row := range unit.Attributes {
		sig.Attributes = append(sig.Attributes, unitAttributeSig{Index: row.Index, AttributeType: row.AttributeType, Amount: row.Amount, Flag: row.Flag, Active: row.Active})
	}
	for _, row := range unit.DamageGraphics {
		sig.DamageGraphics = append(sig.DamageGraphics, damageGraphicSig{Index: row.Index, GraphicID: row.GraphicID, DamagePercent: row.DamagePercent, Flag: row.Flag})
	}
	if unit.Action != nil {
		action := &actionSig{DropSites: append([]int16(nil), unit.Action.DropSites...)}
		for _, task := range unit.Action.Tasks {
			action.Tasks = append(action.Tasks, taskSig{
				Index: task.Index, RecordType: task.RecordType, ID: task.ID, IsDefault: task.IsDefault,
				ActionType: task.ActionType, ObjectClass: task.ObjectClass, ObjectID: task.ObjectID,
				TerrainID: task.TerrainID, AttributeTypes: append([]int16(nil), task.AttributeTypes...),
				WorkValue1: task.WorkValue1, WorkValue2: task.WorkValue2, WorkRange: task.WorkRange,
				AutoSearchTargets: task.AutoSearchTargets, SearchWaitTime: task.SearchWaitTime,
				EnableTargeting: task.EnableTargeting, CombatLevel: task.CombatLevel,
				RawTail: append([]byte(nil), task.RawTail...),
			})
		}
		sig.Action = action
	}
	if unit.Type50 != nil {
		t := &type50Sig{
			ProjectileUnitID: unit.Type50.ProjectileUnitID,
			MaxRange:         unit.Type50.MaxRange,
			BlastWidth:       unit.Type50.BlastWidth,
			AttackGraphic:    unit.Type50.AttackGraphic,
			BlastDamage:      unit.Type50.BlastDamage,
		}
		for _, weapon := range unit.Type50.Attacks {
			t.Attacks = append(t.Attacks, weaponSig{Index: weapon.Index, Class: weapon.Class, Value: weapon.Value})
		}
		for _, armour := range unit.Type50.Armours {
			t.Armours = append(t.Armours, weaponSig{Index: armour.Index, Class: armour.Class, Value: armour.Value})
		}
		sig.Type50 = t
	}
	if unit.Creatable != nil {
		c := &creatableSig{
			TrainLocationCount: unit.Creatable.TrainLocationCount,
			TrainTime0:         unit.Creatable.TrainTime0,
			TrainUnitID0:       unit.Creatable.TrainUnitID0,
			TrainButtonID0:     unit.Creatable.TrainButtonID0,
			TrainHotkeyID0:     unit.Creatable.TrainHotKeyID0,
			ButtonIconID:       unit.Creatable.ButtonIconID,
			ButtonHotkeyAction: unit.Creatable.ButtonHotkeyAction,
		}
		for _, cost := range unit.Creatable.Costs {
			c.Costs = append(c.Costs, attributeCostSig{Index: cost.Index, AttributeType: cost.AttributeType, Amount: cost.Amount, Flag: cost.Flag, Padding: cost.Padding, Active: cost.Active})
		}
		for _, loc := range unit.Creatable.TrainLocations {
			c.TrainLocations = append(c.TrainLocations, trainLocationSig{Index: loc.Index, TrainTime: loc.TrainTime, TrainUnitID: loc.TrainUnitID, TrainButton: loc.TrainButton, TrainHotkey: loc.TrainHotkey})
		}
		sig.Creatable = c
	}
	if unit.Building != nil {
		b := &buildingSig{}
		for _, linked := range unit.Building.LinkedBuildings {
			b.LinkedBuildings = append(b.LinkedBuildings, linkedBuildingSig{Index: linked.Index, UnitID: linked.UnitID, X: linked.X, Y: linked.Y, Active: linked.Active})
		}
		sig.Building = b
	}
	return sig
}

func diffTerrainRestrictions(items []TerrainRestriction) []diffItem {
	out := make([]diffItem, 0, len(items))
	for _, item := range items {
		out = append(out, diffItem{Key: strconv.Itoa(item.Index), Hash: diffHash(item)})
	}
	return out
}

func diffTerrains(items []Terrain) []diffItem {
	out := make([]diffItem, 0, len(items))
	for _, item := range items {
		out = append(out, diffItem{Key: strconv.Itoa(item.Index), Name: item.Name.Value, Hash: diffHash(item)})
	}
	return out
}

func diffSounds(items []Sound) []diffItem {
	out := make([]diffItem, 0, len(items))
	for _, item := range items {
		out = append(out, diffItem{Key: strconv.Itoa(item.Index), Hash: diffHash(item)})
	}
	return out
}

func diffPlayerColours(items []PlayerColour) []diffItem {
	out := make([]diffItem, 0, len(items))
	for _, item := range items {
		out = append(out, diffItem{Key: strconv.Itoa(item.Index), Hash: diffHash(item)})
	}
	return out
}

func diffRandomMaps(items []RandomMapInfo) []diffItem {
	out := make([]diffItem, 0, len(items))
	for _, item := range items {
		out = append(out, diffItem{Key: fmt.Sprintf("%d:%d", item.Pass, item.Index), Hash: diffHash(item)})
	}
	return out
}

func diffHash(value any) string {
	data, _ := json.Marshal(value)
	var normalized any
	if err := json.Unmarshal(data, &normalized); err == nil {
		stripDiffVolatile(normalized)
		data, _ = json.Marshal(normalized)
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func stripDiffVolatile(value any) {
	switch v := value.(type) {
	case map[string]any:
		for key, child := range v {
			switch {
			case key == "span" || key == "field_spans" || key == "list_spans":
				delete(v, key)
				continue
			case key == "record_start" || key == "record_end" || key == "record_length":
				delete(v, key)
				continue
			case strings.HasSuffix(key, "_span") || strings.HasSuffix(key, "_spans"):
				delete(v, key)
				continue
			}
			stripDiffVolatile(child)
		}
	case []any:
		for _, child := range v {
			stripDiffVolatile(child)
		}
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
