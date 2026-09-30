package scenario

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"os"
	"strings"
)

// Layout is Kit's owned description of byte facts. It deliberately does not
// mirror the legacy field JSON: semantic names, evidence, and preservation
// policy are separate from the byte grammar.
type Layout struct {
	Name         string
	Revisions    []LayoutRevision
	Sections     []LayoutSection
	MirrorGroups []MirrorGroup
}

type LayoutRevision struct {
	Name     string
	Versions []string
	Status   string
}

type LayoutSection struct {
	Name   string
	Fields []LayoutField
}

type LayoutField struct {
	Name     string
	Encoding Encoding
	Count    int
	Opaque   bool
	Evidence Evidence
}

type Encoding string

const (
	EncodingBytes    Encoding = "bytes"
	EncodingCString4 Encoding = "c4"
	EncodingString16 Encoding = "string16"
	EncodingString32 Encoding = "string32"
	EncodingU8       Encoding = "u8"
	EncodingS8       Encoding = "s8"
	EncodingU16      Encoding = "u16"
	EncodingS16      Encoding = "s16"
	EncodingU32      Encoding = "u32"
	EncodingS32      Encoding = "s32"
	EncodingF32      Encoding = "f32"
)

type Evidence struct {
	Tier   string
	Source string
}

// MirrorGroup makes duplicated storage explicit. The writer policy is not
// implied by field names: callers must choose the authoritative copy.
type MirrorGroup struct {
	Name          string
	Authoritative string
	Derived       []string
	Policy        string
}

// OpaqueSpan identifies bytes that the semantic model does not own. Spans are
// retained in their original order and copied verbatim by the preservation
// layer; a dark field is not permission to normalize it.
type OpaqueSpan struct {
	Name   string
	Offset int
	Length int
}

// PreservedDocument is the lossless carrier shared by the codecs. The first
// slice only creates it and reads from it; mutation/writer policies arrive in
// later slices once section framing is migrated.
type PreservedDocument struct {
	Header       []byte
	Body         []byte
	OriginalFile []byte
	Opaque       []OpaqueSpan
}

func NewPreservedDocument(header, body []byte) PreservedDocument {
	return PreservedDocument{Header: append([]byte(nil), header...), Body: append([]byte(nil), body...)}
}

func (d PreservedDocument) RebuildBody() []byte {
	return append([]byte(nil), d.Body...)
}

func (d PreservedDocument) RebuildFile() []byte {
	out := make([]byte, 0, len(d.Header)+len(d.Body))
	out = append(out, d.Header...)
	out = append(out, d.Body...)
	return out
}

// Write writes the original header/body bytes without recompression or
// normalization. It is the safe writer path until a section is explicitly
// marked dirty by a migrated codec.
func (d PreservedDocument) Write(path string) error {
	if len(d.OriginalFile) == 0 {
		return errors.New("preserved document is empty")
	}
	return os.WriteFile(path, append([]byte(nil), d.OriginalFile...), 0o644)
}

// DE158Layout is the first independently owned layout slice. 1.59 is listed
// only because the corpus currently proves these sections are shared; it is
// not an assumption that every later section is unchanged.
var DE158Layout = Layout{
	Name:      "aoe2de-scenario",
	Revisions: []LayoutRevision{{Name: "file-map-v1", Versions: []string{"1.58", "1.59"}, Status: "corpus_verified"}},
	Sections: []LayoutSection{
		{Name: "FileHeader", Fields: []LayoutField{
			{Name: "version", Encoding: EncodingCString4, Evidence: Evidence{Tier: "corpus_verified", Source: "editor-saved headers"}},
			{Name: "header_length", Encoding: EncodingU32, Evidence: Evidence{Tier: "corpus_verified", Source: "editor-saved headers"}},
			{Name: "savable", Encoding: EncodingS32, Evidence: Evidence{Tier: "corpus_verified", Source: "editor-saved headers"}},
			{Name: "timestamp_of_last_save", Encoding: EncodingU32, Evidence: Evidence{Tier: "corpus_verified", Source: "editor-saved headers"}},
			{Name: "scenario_instructions", Encoding: EncodingString32, Evidence: Evidence{Tier: "corpus_verified", Source: "editor-saved headers"}},
			{Name: "player_count", Encoding: EncodingU32, Evidence: Evidence{Tier: "corpus_verified", Source: "editor-saved headers"}},
			{Name: "unknown_value", Encoding: EncodingU32, Opaque: true, Evidence: Evidence{Tier: "dark", Source: "corpus value only"}},
			{Name: "unknown_value_2", Encoding: EncodingU32, Opaque: true, Evidence: Evidence{Tier: "dark", Source: "corpus value only"}},
			{Name: "amount_of_unknown_numbers", Encoding: EncodingU32, Opaque: true, Evidence: Evidence{Tier: "corpus_verified", Source: "count precedes list"}},
			{Name: "unknown_numbers", Encoding: EncodingU32, Count: -1, Opaque: true, Evidence: Evidence{Tier: "corpus_verified", Source: "counted header list"}},
			{Name: "creator_name", Encoding: EncodingString32, Evidence: Evidence{Tier: "corpus_verified", Source: "editor-saved headers"}},
			{Name: "trigger_count", Encoding: EncodingU32, Evidence: Evidence{Tier: "corpus_verified", Source: "header/body agreement"}},
		}},
		{Name: "Map", Fields: []LayoutField{
			{Name: "string_starter_1", Encoding: EncodingBytes, Count: 2, Opaque: true, Evidence: Evidence{Tier: "corpus_verified", Source: "editor-saved map sections"}},
			{Name: "water_definition", Encoding: EncodingString16, Evidence: Evidence{Tier: "corpus_verified", Source: "editor calibration round3"}},
			{Name: "string_starter_2", Encoding: EncodingBytes, Count: 2, Opaque: true, Evidence: Evidence{Tier: "corpus_verified", Source: "editor-saved map sections"}},
			{Name: "map_color_mood", Encoding: EncodingString16, Evidence: Evidence{Tier: "corpus_verified", Source: "editor calibration round3"}},
			{Name: "string_starter_3", Encoding: EncodingBytes, Count: 2, Opaque: true, Evidence: Evidence{Tier: "corpus_verified", Source: "editor-saved map sections"}},
			{Name: "script_name", Encoding: EncodingString16, Evidence: Evidence{Tier: "corpus_verified", Source: "editor-saved map sections"}},
			{Name: "collide_and_correct", Encoding: EncodingU8, Evidence: Evidence{Tier: "corpus_verified", Source: "editor-saved map sections"}},
			{Name: "villager_force_drop", Encoding: EncodingU8, Evidence: Evidence{Tier: "corpus_verified", Source: "editor-saved map sections"}},
			{Name: "initial_player_views", Encoding: EncodingS32, Count: 32, Evidence: Evidence{Tier: "corpus_verified", Source: "fixed 16 x/y pairs"}},
			{Name: "lock_coop_alliances", Encoding: EncodingU8, Evidence: Evidence{Tier: "corpus_verified", Source: "editor-saved map sections"}},
			{Name: "per_player_population_cap", Encoding: EncodingU32, Count: 16, Evidence: Evidence{Tier: "corpus_verified", Source: "fixed player array"}},
			{Name: "secondary_game_modes", Encoding: EncodingU32, Evidence: Evidence{Tier: "engine_verified", Source: "editor calibration round3"}},
			{Name: "unknown_5", Encoding: EncodingBytes, Count: 4, Opaque: true, Evidence: Evidence{Tier: "dark", Source: "corpus bytes only"}},
			{Name: "unknown_4", Encoding: EncodingBytes, Count: 4, Opaque: true, Evidence: Evidence{Tier: "dark", Source: "corpus bytes only"}},
			{Name: "no_waves_on_shore", Encoding: EncodingS8, Evidence: Evidence{Tier: "corpus_verified", Source: "editor-saved map sections"}},
			{Name: "map_width", Encoding: EncodingS32, Evidence: Evidence{Tier: "corpus_verified", Source: "terrain count agreement"}},
			{Name: "map_height", Encoding: EncodingS32, Evidence: Evidence{Tier: "corpus_verified", Source: "terrain count agreement"}},
			{Name: "terrain_data", Encoding: EncodingBytes, Count: -1, Evidence: Evidence{Tier: "corpus_verified", Source: "width x height x 5-byte cells"}},
		}},
	},
	MirrorGroups: []MirrorGroup{
		{Name: "player_diplomacy", Authoritative: "Diplomacy.matrix", Derived: []string{"Players[].diplomacy", "Units[].diplomacy_for_interaction"}, Policy: "authoritative_matrix_projects_to_derived_views"},
		{Name: "allied_victory", Authoritative: "Diplomacy.allied_victory", Derived: []string{"Players[].allied_victory", "Units[].aok_allied_victory"}, Policy: "authoritative_array_projects_to_derived_views"},
	},
}

type byteCursor struct {
	data []byte
	off  int
}

func (c *byteCursor) take(n int) ([]byte, error) {
	if n < 0 || c.off < 0 || n > len(c.data)-c.off {
		return nil, fmt.Errorf("need %d bytes at offset %d, have %d", n, c.off, len(c.data)-c.off)
	}
	b := c.data[c.off : c.off+n]
	c.off += n
	return b, nil
}

func (c *byteCursor) u8() (uint8, error) {
	b, err := c.take(1)
	if err != nil {
		return 0, err
	}
	return b[0], nil
}
func (c *byteCursor) s8() (int8, error) { v, err := c.u8(); return int8(v), err }
func (c *byteCursor) u16() (uint16, error) {
	b, err := c.take(2)
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint16(b), nil
}
func (c *byteCursor) s16() (int16, error) { v, err := c.u16(); return int16(v), err }
func (c *byteCursor) u32() (uint32, error) {
	b, err := c.take(4)
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint32(b), nil
}
func (c *byteCursor) s32() (int32, error)   { v, err := c.u32(); return int32(v), err }
func (c *byteCursor) f32() (float32, error) { v, err := c.u32(); return math.Float32frombits(v), err }

func (c *byteCursor) stringN(width int) (string, error) {
	var n int
	var err error
	if width == 2 {
		v, e := c.s16()
		n, err = int(v), e
	} else {
		v, e := c.s32()
		n, err = int(v), e
	}
	if err != nil {
		return "", err
	}
	if n < 0 {
		if n == -1 {
			return "", nil
		}
		return "", fmt.Errorf("negative string length %d", n)
	}
	b, err := c.take(n)
	if err != nil {
		return "", err
	}
	if i := bytes.IndexByte(b, 0); i >= 0 {
		b = b[:i]
	}
	return string(b), nil
}

func (c *byteCursor) fixedString(n int) (string, error) {
	b, err := c.take(n)
	if err != nil {
		return "", err
	}
	if i := bytes.IndexByte(b, 0); i >= 0 {
		b = b[:i]
	}
	return string(b), nil
}

// DataHeaderLayout is the owned player/header slice. The player records are
// deliberately typed, while the two fixed opaque tails retain their bytes.
type DataHeaderLayout struct {
	NextUnitID       uint32
	Version          float32
	PlayerCapacity   int32
	GaiaPlayerIndex  int32
	TribeNames       []string
	PlayerNameIDs    []int32
	Players          []PlayerDataLayout
	LockCivilization []uint32
	LockPersonality  []uint32
	Opaque           []byte
	Filename         string
}

type PlayerDataLayout struct {
	Active          uint32
	Human           uint32
	StrSign1        uint16
	Civilization    string
	StrSign2        uint16
	ArchitectureSet string
	CityMode        uint32
}

// DecodeUnitStructNode decodes the stable unit payload. DE 1.59 inserts one
// discriminator byte immediately before caption_string_id; the caller tells
// us which proven layout the enclosing version selected.
func DecodeUnitStructNode(data []byte, has159Byte bool) (*parsedNode, int, error) {
	c := &byteCursor{data: data}
	node := &parsedNode{Name: "UnitStruct", Start: 0}
	read := func(name string, fn func(*byteCursor) (any, error)) error {
		start := c.off
		value, err := fn(c)
		if err != nil {
			return err
		}
		node.Fields = append(node.Fields, &parsedNode{Name: name, Start: start, End: c.off, Raw: append([]byte(nil), data[start:c.off]...), Value: value})
		return nil
	}
	for _, field := range []struct {
		name string
		fn   func(*byteCursor) (any, error)
	}{
		{"x", func(c *byteCursor) (any, error) { return c.f32() }},
		{"y", func(c *byteCursor) (any, error) { return c.f32() }},
		{"z", func(c *byteCursor) (any, error) { return c.f32() }},
		{"reference_id", func(c *byteCursor) (any, error) { return c.s32() }},
		{"unit_const", func(c *byteCursor) (any, error) { return c.u16() }},
		{"status", func(c *byteCursor) (any, error) { return c.u8() }},
		{"rotation", func(c *byteCursor) (any, error) { return c.f32() }},
		{"initial_animation_frame", func(c *byteCursor) (any, error) { return c.u16() }},
		{"garrisoned_in_id", func(c *byteCursor) (any, error) { return c.s32() }},
	} {
		if err := read(field.name, field.fn); err != nil {
			return nil, 0, err
		}
	}
	if has159Byte {
		if err := read("unknown_1_59_before_caption_string_id", func(c *byteCursor) (any, error) { return c.u8() }); err != nil {
			return nil, 0, err
		}
	}
	if err := read("caption_string_id", func(c *byteCursor) (any, error) { return c.s32() }); err != nil {
		return nil, 0, err
	}
	if err := read("caption_string", func(c *byteCursor) (any, error) { return c.stringN(4) }); err != nil {
		return nil, 0, err
	}
	node.End = c.off
	return node, c.off, nil
}

func DecodePlayerDataFourNode(data []byte) (*parsedNode, int, error) {
	c := &byteCursor{data: data}
	node := &parsedNode{Name: "PlayerDataFourStruct"}
	for _, field := range []struct {
		name string
		fn   func(*byteCursor) (any, error)
	}{
		{"food_duplicate", func(c *byteCursor) (any, error) { return c.f32() }},
		{"wood_duplicate", func(c *byteCursor) (any, error) { return c.f32() }},
		{"gold_duplicate", func(c *byteCursor) (any, error) { return c.f32() }},
		{"stone_duplicate", func(c *byteCursor) (any, error) { return c.f32() }},
		{"ore_x_duplicate", func(c *byteCursor) (any, error) { return c.f32() }},
		{"trade_goods_duplicate", func(c *byteCursor) (any, error) { return c.f32() }},
		{"population_limit", func(c *byteCursor) (any, error) { return c.f32() }},
	} {
		start := c.off
		value, err := field.fn(c)
		if err != nil {
			return nil, 0, err
		}
		node.Fields = append(node.Fields, &parsedNode{Name: field.name, Start: start, End: c.off, Raw: append([]byte(nil), data[start:c.off]...), Value: value})
	}
	node.End = c.off
	return node, c.off, nil
}

func DecodePlayerDataThreeNode(data []byte) (*parsedNode, int, error) {
	c := &byteCursor{data: data}
	node := &parsedNode{Name: "PlayerDataThreeStruct"}
	read := func(name string, fn func(*byteCursor) (any, error)) (any, error) {
		start := c.off
		value, err := fn(c)
		if err != nil {
			return nil, err
		}
		node.Fields = append(node.Fields, &parsedNode{Name: name, Start: start, End: c.off, Raw: append([]byte(nil), data[start:c.off]...), Value: value})
		return value, nil
	}
	if _, err := read("constant_name", func(c *byteCursor) (any, error) { return c.stringN(2) }); err != nil {
		return nil, 0, err
	}
	if _, err := read("editor_camera_x", func(c *byteCursor) (any, error) { return c.f32() }); err != nil {
		return nil, 0, err
	}
	if _, err := read("editor_camera_y", func(c *byteCursor) (any, error) { return c.f32() }); err != nil {
		return nil, 0, err
	}
	if _, err := read("initial_camera_x", func(c *byteCursor) (any, error) { return c.s16() }); err != nil {
		return nil, 0, err
	}
	if _, err := read("initial_camera_y", func(c *byteCursor) (any, error) { return c.s16() }); err != nil {
		return nil, 0, err
	}
	if _, err := read("aok_allied_victory", func(c *byteCursor) (any, error) { return c.u8() }); err != nil {
		return nil, 0, err
	}
	v, err := read("player_count_for_diplomacy", func(c *byteCursor) (any, error) { return c.u16() })
	if err != nil {
		return nil, 0, err
	}
	count := int(v.(uint16))
	if count < 0 || count > 64 {
		return nil, 0, fmt.Errorf("invalid diplomacy count %d", count)
	}
	if _, err := read("diplomacy_for_interaction", func(c *byteCursor) (any, error) {
		values := make([]any, count)
		for i := range values {
			x, e := c.u8()
			if e != nil {
				return nil, e
			}
			values[i] = x
		}
		return values, nil
	}); err != nil {
		return nil, 0, err
	}
	if _, err := read("diplomacy_for_ai_system", func(c *byteCursor) (any, error) {
		values := make([]any, count)
		for i := range values {
			x, e := c.u32()
			if e != nil {
				return nil, e
			}
			values[i] = x
		}
		return values, nil
	}); err != nil {
		return nil, 0, err
	}
	if _, err := read("color", func(c *byteCursor) (any, error) { return c.u32() }); err != nil {
		return nil, 0, err
	}
	v, err = read("victory_version", func(c *byteCursor) (any, error) { return c.f32() })
	if err != nil {
		return nil, 0, err
	}
	if _, err := read("unknown", func(c *byteCursor) (any, error) { return c.u16() }); err != nil {
		return nil, 0, err
	}
	countValue := node.Fields[len(node.Fields)-1].Value.(uint16)
	entryCount := int(countValue)
	if entryCount > 64 {
		return nil, 0, fmt.Errorf("invalid victory-condition count %d", entryCount)
	}
	if v.(float32) >= 2 {
		if _, err := read("unknown_2", func(c *byteCursor) (any, error) {
			values := make([]any, 7)
			for i := range values {
				x, e := c.u8()
				if e != nil {
					return nil, e
				}
				values[i] = x
			}
			return values, nil
		}); err != nil {
			return nil, 0, err
		}
	}
	if _, err := read("unknown_structure_3", func(c *byteCursor) (any, error) {
		values := make([]any, entryCount)
		for i := range values {
			value, err := c.take(44)
			if err != nil {
				return nil, err
			}
			values[i] = value
		}
		return values, nil
	}); err != nil {
		return nil, 0, err
	}
	var trailingCount int
	if v.(float32) >= 2 {
		trailing, err := read("unknown_5", func(c *byteCursor) (any, error) { return c.u8() })
		if err != nil {
			return nil, 0, err
		}
		trailingCount = int(trailing.(uint8))
		if trailingCount > 64 {
			return nil, 0, fmt.Errorf("invalid trailing structure count %d", trailingCount)
		}
	}
	if _, err := read("unknown_3", func(c *byteCursor) (any, error) {
		values := make([]any, 7)
		for i := range values {
			x, e := c.u8()
			if e != nil {
				return nil, e
			}
			values[i] = x
		}
		return values, nil
	}); err != nil {
		return nil, 0, err
	}
	if _, err := read("unknown_structure_4", func(c *byteCursor) (any, error) {
		values := make([]any, trailingCount)
		for i := range values {
			value, err := c.take(32)
			if err != nil {
				return nil, err
			}
			values[i] = value
		}
		return values, nil
	}); err != nil {
		return nil, 0, err
	}
	if _, err := read("unknown_4", func(c *byteCursor) (any, error) { return c.s32() }); err != nil {
		return nil, 0, err
	}
	node.End = c.off
	return node, c.off, nil
}

func DecodePlayerUnitsNode(data []byte, has159Byte bool) (*parsedNode, int, error) {
	c := &byteCursor{data: data}
	node := &parsedNode{Name: "PlayerUnitsStruct"}
	start := c.off
	count, err := c.u32()
	if err != nil {
		return nil, 0, err
	}
	node.Fields = append(node.Fields, &parsedNode{Name: "unit_count", Start: start, End: c.off, Raw: append([]byte(nil), data[start:c.off]...), Value: count})
	units := &parsedNode{Name: "units", Start: c.off}
	if count > 1000000 {
		return nil, 0, fmt.Errorf("unit count %d exceeds safety limit", count)
	}
	for i := uint32(0); i < count; i++ {
		unit, consumed, err := DecodeUnitStructNode(data[c.off:], has159Byte)
		if err != nil {
			return nil, 0, fmt.Errorf("unit %d: %w", i, err)
		}
		unit.Start = c.off
		unit.End = c.off + consumed
		c.off += consumed
		units.Elements = append(units.Elements, unit)
	}
	units.End = c.off
	units.Value = units.Elements
	node.Fields = append(node.Fields, units)
	node.End = c.off
	return node, c.off, nil
}

func decodeS32StringNode(data []byte, name string, fields []string) (*parsedNode, int, error) {
	c := &byteCursor{data: data}
	node := &parsedNode{Name: name}
	for _, field := range fields {
		start := c.off
		value, err := c.s32()
		if err != nil {
			return nil, 0, err
		}
		node.Fields = append(node.Fields, &parsedNode{Name: field, Start: start, End: c.off, Raw: append([]byte(nil), data[start:c.off]...), Value: value})
	}
	start := c.off
	value, err := c.stringN(4)
	if err != nil {
		return nil, 0, err
	}
	node.Fields = append(node.Fields, &parsedNode{Name: "xs_function", Start: start, End: c.off, Raw: append([]byte(nil), data[start:c.off]...), Value: value})
	node.End = c.off
	return node, c.off, nil
}

func DecodeConditionNode(data []byte) (*parsedNode, int, error) {
	fields := []string{"condition_type", "static_value_33", "quantity", "attribute", "unit_object", "next_object", "object_list", "source_player", "technology", "timer", "trigger_id", "area_x1", "area_y1", "area_x2", "area_y2", "object_group", "object_type", "ai_signal", "inverted", "unknown_2", "variable", "comparison", "target_player", "unit_ai_action", "unknown_4", "object_state", "timer_id", "victory_timer_type", "include_changeable_weapon_objects", "decision_id", "decision_option", "variable2", "local_technology", "object_group2", "object_type2"}
	c := &byteCursor{data: data}
	node := &parsedNode{Name: "ConditionStruct"}
	marker := int32(0)
	for _, field := range fields {
		start := c.off
		value, err := c.s32()
		if err != nil {
			return nil, 0, err
		}
		if field == "static_value_33" {
			marker = value
		}
		node.Fields = append(node.Fields, &parsedNode{Name: field, Start: start, End: c.off, Raw: append([]byte(nil), data[start:c.off]...), Value: value})
	}
	if marker == 34 {
		start := c.off
		value, err := c.s32()
		if err != nil {
			return nil, 0, err
		}
		node.Fields = append(node.Fields, &parsedNode{Name: "unknown_1_59_editor_before_xs_function", Start: start, End: c.off, Raw: append([]byte(nil), data[start:c.off]...), Value: value})
	} else if marker != 33 {
		return nil, 0, fmt.Errorf("unsupported 1.59 condition layout marker %d", marker)
	}
	start := c.off
	value, err := c.stringN(4)
	if err != nil {
		return nil, 0, err
	}
	node.Fields = append(node.Fields, &parsedNode{Name: "xs_function", Start: start, End: c.off, Raw: append([]byte(nil), data[start:c.off]...), Value: value})
	node.End = c.off
	return node, c.off, nil
}

func DecodeVariableNode(data []byte) (*parsedNode, int, error) {
	c := &byteCursor{data: data}
	node := &parsedNode{Name: "VariableStruct"}
	start := c.off
	id, err := c.u32()
	if err != nil {
		return nil, 0, err
	}
	node.Fields = append(node.Fields, &parsedNode{Name: "variable_id", Start: start, End: c.off, Raw: append([]byte(nil), data[start:c.off]...), Value: id})
	start = c.off
	name, err := c.stringN(4)
	if err != nil {
		return nil, 0, err
	}
	node.Fields = append(node.Fields, &parsedNode{Name: "variable_name", Start: start, End: c.off, Raw: append([]byte(nil), data[start:c.off]...), Value: name})
	node.End = c.off
	return node, c.off, nil
}

func DecodeEffectNode(data []byte) (*parsedNode, int, error) {
	before := []string{"effect_type", "static_value_83", "ai_script_goal", "quantity", "tribute_list", "diplomacy", "number_of_units_selected", "legacy_location_object_reference", "object_list_unit_id", "source_player", "target_player", "technology", "string_id", "unknown_2", "display_time", "trigger_id", "location_x", "location_y", "area_x1", "area_y1", "area_x2", "area_y2", "object_group", "object_type", "instruction_panel_position", "attack_stance", "time_unit", "enabled", "food", "wood", "stone", "gold", "item_id", "flash_object", "force_research_technology", "visibility_state", "scroll", "operation", "object_list_unit_id_2", "button_location", "ai_signal_value", "unknown_3", "object_attributes", "variable", "timer", "facet", "location_object_reference", "play_sound", "player_color", "unknown_4", "color_mood", "reset_timer", "object_state", "action_type", "resource_1", "resource_1_quantity", "resource_2", "resource_2_quantity", "resource_3", "resource_3_quantity", "decision_id", "string_id_option1", "string_id_option2", "variable2", "max_units_affected", "disable_garrison_unload_sound", "hotkey", "train_time", "local_technology", "disable_sound", "object_group2", "object_type2"}
	after := []string{"facet2", "global_sound", "issue_group_command", "queue_action", "mutual_diplomacy", "building_list", "wall_x1", "wall_y1", "wall_x2", "wall_y2", "object_filter", "use_tag_color_for_icon"}
	c := &byteCursor{data: data}
	node := &parsedNode{Name: "EffectStruct"}
	read := func(name string, fn func(*byteCursor) (any, error)) error {
		s := c.off
		v, e := fn(c)
		if e != nil {
			return e
		}
		node.Fields = append(node.Fields, &parsedNode{Name: name, Start: s, End: c.off, Raw: append([]byte(nil), data[s:c.off]...), Value: v})
		return nil
	}
	for _, name := range before {
		if err := read(name, func(c *byteCursor) (any, error) { return c.s32() }); err != nil {
			return nil, 0, err
		}
	}
	if err := read("quantity_float", func(c *byteCursor) (any, error) { return c.f32() }); err != nil {
		return nil, 0, err
	}
	for _, name := range after {
		if err := read(name, func(c *byteCursor) (any, error) { return c.s32() }); err != nil {
			return nil, 0, err
		}
	}
	for _, name := range []string{"message", "sound_name"} {
		if err := read(name, func(c *byteCursor) (any, error) { return c.stringN(4) }); err != nil {
			return nil, 0, err
		}
	}
	selected := int32(0)
	for _, f := range node.Fields {
		if f.Name == "number_of_units_selected" {
			selected = f.Value.(int32)
		}
	}
	if selected < 0 || selected > 1000000 {
		selected = 0
	}
	if err := read("selected_object_ids", func(c *byteCursor) (any, error) {
		values := make([]any, selected)
		for i := range values {
			x, e := c.s32()
			if e != nil {
				return nil, e
			}
			values[i] = x
		}
		return values, nil
	}); err != nil {
		return nil, 0, err
	}
	for _, name := range []string{"message_option1", "message_option2"} {
		if err := read(name, func(c *byteCursor) (any, error) { return c.stringN(4) }); err != nil {
			return nil, 0, err
		}
	}
	node.End = c.off
	return node, c.off, nil
}

func decodeStringSection(data []byte, name string, fields []string) (*parsedNode, int, error) {
	c := &byteCursor{data: data}
	node := &parsedNode{Name: name}
	for _, fieldName := range fields {
		start := c.off
		var value any
		var err error
		if strings.HasPrefix(fieldName, "ascii_") {
			value, err = c.stringN(2)
		} else {
			value, err = c.u32()
		}
		if err != nil {
			return nil, 0, err
		}
		node.Fields = append(node.Fields, &parsedNode{Name: fieldName, Start: start, End: c.off, Raw: append([]byte(nil), data[start:c.off]...), Value: value})
	}
	node.End = c.off
	return node, c.off, nil
}

func DecodeMessagesNode(data []byte) (*parsedNode, int, error) {
	return decodeStringSection(data, "Messages", []string{
		"instructions", "hints", "victory", "loss", "history", "scouts",
		"ascii_instructions", "ascii_hints", "ascii_victory", "ascii_loss", "ascii_history", "ascii_scouts",
	})
}

func DecodeCinematicsNode(data []byte) (*parsedNode, int, error) {
	return decodeStringSection(data, "Cinematics", []string{"ascii_pregame", "ascii_victory", "ascii_loss"})
}

func DecodeOptionsNode(data []byte) (*parsedNode, int, error) {
	c := &byteCursor{data: data}
	node := &parsedNode{Name: "Options"}
	read := func(name string, count int, fn func(*byteCursor) (any, error)) error {
		start := c.off
		values := make([]any, count)
		for i := range values {
			value, err := fn(c)
			if err != nil {
				return err
			}
			values[i] = value
		}
		var value any = values
		if count == 1 {
			value = values[0]
		}
		node.Fields = append(node.Fields, &parsedNode{Name: name, Start: start, End: c.off, Raw: append([]byte(nil), data[start:c.off]...), Value: value})
		return nil
	}
	readPlayerLists := func(prefix string, counts []uint32) error {
		for player, count := range counts[:8] {
			if err := read(fmt.Sprintf("%s_player_%d", prefix, player+1), int(count), func(c *byteCursor) (any, error) { return c.u32() }); err != nil {
				return err
			}
		}
		return nil
	}
	readCounts := func(name string) ([]uint32, error) {
		if err := read(name, 16, func(c *byteCursor) (any, error) { return c.u32() }); err != nil {
			return nil, err
		}
		values := node.Fields[len(node.Fields)-1].Value.([]any)
		counts := make([]uint32, 16)
		for i, value := range values {
			counts[i] = value.(uint32)
		}
		return counts, nil
	}
	techCounts, err := readCounts("per_player_number_of_disabled_techs")
	if err != nil {
		return nil, 0, err
	}
	if err := readPlayerLists("disabled_tech_ids", techCounts); err != nil {
		return nil, 0, err
	}
	unitCounts, err := readCounts("per_player_number_of_disabled_units")
	if err != nil {
		return nil, 0, err
	}
	if err := readPlayerLists("disabled_unit_ids", unitCounts); err != nil {
		return nil, 0, err
	}
	buildingCounts, err := readCounts("per_player_number_of_disabled_buildings")
	if err != nil {
		return nil, 0, err
	}
	if err := readPlayerLists("disabled_building_ids", buildingCounts); err != nil {
		return nil, 0, err
	}
	for _, name := range []string{"combat_mode", "naval_mode", "all_techs"} {
		if err := read(name, 1, func(c *byteCursor) (any, error) { return c.u32() }); err != nil {
			return nil, 0, err
		}
	}
	if err := read("per_player_starting_age", 16, func(c *byteCursor) (any, error) { return c.u32() }); err != nil {
		return nil, 0, err
	}
	for _, field := range []struct {
		name string
		size int
	}{{"unknown_1", 12}, {"unknown_3", 1}} {
		if err := read(field.name, 1, func(c *byteCursor) (any, error) { b, err := c.take(field.size); return append([]byte(nil), b...), err }); err != nil {
			return nil, 0, err
		}
	}
	if err := read("ai_map_type", 1, func(c *byteCursor) (any, error) { return c.s32() }); err != nil {
		return nil, 0, err
	}
	if err := read("per_player_base_priority", 8, func(c *byteCursor) (any, error) { return c.u8() }); err != nil {
		return nil, 0, err
	}
	if err := read("unknown_2", 1, func(c *byteCursor) (any, error) { b, err := c.take(7); return append([]byte(nil), b...), err }); err != nil {
		return nil, 0, err
	}
	if err := read("number_of_triggers", 1, func(c *byteCursor) (any, error) { return c.u32() }); err != nil {
		return nil, 0, err
	}
	node.End = c.off
	return node, c.off, nil
}

// DecodeBackgroundImageNode owns the image frame. Empty editor images are a
// 16-byte frame; populated images carry one bitmap-info record followed by
// variable color and pixel payloads.
func DecodeBackgroundImageNode(data []byte) (*parsedNode, int, error) {
	c := &byteCursor{data: data}
	node := &parsedNode{Name: "BackgroundImage"}
	read := func(name string, fn func(*byteCursor) (any, error)) error {
		start := c.off
		value, err := fn(c)
		if err != nil {
			return err
		}
		node.Fields = append(node.Fields, &parsedNode{Name: name, Start: start, End: c.off, Raw: append([]byte(nil), data[start:c.off]...), Value: value})
		return nil
	}
	if err := read("ascii_filename", func(c *byteCursor) (any, error) { return c.stringN(2) }); err != nil {
		return nil, 0, err
	}
	if err := read("picture_version", func(c *byteCursor) (any, error) { return c.u32() }); err != nil {
		return nil, 0, err
	}
	if err := read("bitmap_width", func(c *byteCursor) (any, error) { return c.u32() }); err != nil {
		return nil, 0, err
	}
	if err := read("bitmap_height", func(c *byteCursor) (any, error) { return c.s32() }); err != nil {
		return nil, 0, err
	}
	if err := read("picture_orientation", func(c *byteCursor) (any, error) { return c.s16() }); err != nil {
		return nil, 0, err
	}
	width := node.field("bitmap_width").Value.(uint32)
	height := node.field("bitmap_height").Value.(int32)
	if width == 0 || height <= 0 {
		node.End = c.off
		return node, c.off, nil
	}
	bitmap := &parsedNode{Name: "bitmap_info", Start: c.off}
	readBitmap := func(name string, fn func(*byteCursor) (any, error)) error {
		start := c.off
		value, err := fn(c)
		if err != nil {
			return err
		}
		bitmap.Fields = append(bitmap.Fields, &parsedNode{Name: name, Start: start, End: c.off, Raw: append([]byte(nil), data[start:c.off]...), Value: value})
		return nil
	}
	for _, field := range []struct {
		name string
		read func(*byteCursor) (any, error)
	}{
		{"size", func(c *byteCursor) (any, error) { return c.s32() }},
		{"width", func(c *byteCursor) (any, error) { return c.u32() }},
		{"height", func(c *byteCursor) (any, error) { return c.u32() }},
		{"planes", func(c *byteCursor) (any, error) { return c.s16() }},
		{"bit_count", func(c *byteCursor) (any, error) { return c.s16() }},
		{"compression", func(c *byteCursor) (any, error) { return c.u32() }},
		{"image_size", func(c *byteCursor) (any, error) { return c.u32() }},
		{"x_pels", func(c *byteCursor) (any, error) { return c.u32() }},
		{"y_pels", func(c *byteCursor) (any, error) { return c.u32() }},
		{"number_of_colors_used", func(c *byteCursor) (any, error) { return c.u32() }},
		{"important_colors", func(c *byteCursor) (any, error) { return c.u32() }},
	} {
		if err := readBitmap(field.name, field.read); err != nil {
			return nil, 0, err
		}
	}
	colors := bitmap.field("number_of_colors_used").Value.(uint32)
	if colors > uint32(len(data)-c.off)/4 {
		return nil, 0, fmt.Errorf("background image color table too large: %d", colors)
	}
	colorStart := c.off
	colorBytes, err := c.take(int(colors) * 4)
	if err != nil {
		return nil, 0, err
	}
	bitmap.Fields = append(bitmap.Fields, &parsedNode{Name: "colors_used", Start: colorStart, End: c.off, Raw: append([]byte(nil), colorBytes...), Value: append([]byte(nil), colorBytes...)})
	if uint64(uint32(width))*uint64(uint32(height)) > uint64(len(data)-c.off) {
		return nil, 0, fmt.Errorf("background image pixel payload too large: %dx%d", width, height)
	}
	imageStart := c.off
	image, err := c.take(int(uint64(width) * uint64(uint32(height))))
	if err != nil {
		return nil, 0, err
	}
	bitmap.Fields = append(bitmap.Fields, &parsedNode{Name: "image", Start: imageStart, End: c.off, Raw: append([]byte(nil), image...), Value: append([]byte(nil), image...)})
	bitmap.End = c.off
	node.Fields = append(node.Fields, bitmap)
	node.End = c.off
	return node, c.off, nil
}

// DecodePlayerDataTwoNode owns the fixed player metadata section. AI payloads
// and resource mirrors are decoded structurally, while the eight-byte AI
// prefix remains opaque because its semantics are not established.
func DecodePlayerDataTwoNode(data []byte) (*parsedNode, int, error) {
	c := &byteCursor{data: data}
	node := &parsedNode{Name: "PlayerDataTwo"}
	readList := func(name string, count int, fn func(*byteCursor) (any, error)) error {
		start := c.off
		values := make([]any, count)
		for i := range values {
			value, err := fn(c)
			if err != nil {
				return err
			}
			values[i] = value
		}
		node.Fields = append(node.Fields, &parsedNode{Name: name, Start: start, End: c.off, Raw: append([]byte(nil), data[start:c.off]...), Value: values})
		return nil
	}
	if err := readList("strings", 32, func(c *byteCursor) (any, error) { return c.stringN(2) }); err != nil {
		return nil, 0, err
	}
	if err := readList("ai_names", 16, func(c *byteCursor) (any, error) { return c.stringN(2) }); err != nil {
		return nil, 0, err
	}
	aiFiles := &parsedNode{Name: "ai_files", Start: c.off}
	for i := 0; i < 16; i++ {
		entry := &parsedNode{Name: fmt.Sprintf("ai_file_%d", i+1), Start: c.off}
		unknownStart := c.off
		unknown, err := c.take(8)
		if err != nil {
			return nil, 0, err
		}
		entry.Fields = append(entry.Fields, &parsedNode{Name: "unknown", Start: unknownStart, End: c.off, Raw: append([]byte(nil), unknown...), Value: append([]byte(nil), unknown...)})
		stringStart := c.off
		value, err := c.stringN(4)
		if err != nil {
			return nil, 0, err
		}
		entry.Fields = append(entry.Fields, &parsedNode{Name: "ai_per_file_text", Start: stringStart, End: c.off, Raw: append([]byte(nil), data[stringStart:c.off]...), Value: value})
		entry.End = c.off
		aiFiles.Elements = append(aiFiles.Elements, entry)
	}
	aiFiles.End = c.off
	node.Fields = append(node.Fields, aiFiles)
	if err := readList("ai_type", 16, func(c *byteCursor) (any, error) { return c.u8() }); err != nil {
		return nil, 0, err
	}
	start := c.off
	separator, err := c.u32()
	if err != nil {
		return nil, 0, err
	}
	node.Fields = append(node.Fields, &parsedNode{Name: "separator", Start: start, End: c.off, Raw: append([]byte(nil), data[start:c.off]...), Value: separator})
	resources := &parsedNode{Name: "resources", Start: c.off}
	for i := 0; i < 16; i++ {
		entry := &parsedNode{Name: fmt.Sprintf("resources_%d", i+1), Start: c.off}
		for _, field := range []string{"gold", "wood", "food", "stone", "ore_x_unused", "trade_goods", "player_color"} {
			fieldStart := c.off
			value, err := c.s32()
			if err != nil {
				return nil, 0, err
			}
			entry.Fields = append(entry.Fields, &parsedNode{Name: field, Start: fieldStart, End: c.off, Raw: append([]byte(nil), data[fieldStart:c.off]...), Value: value})
		}
		entry.End = c.off
		resources.Elements = append(resources.Elements, entry)
	}
	resources.End = c.off
	node.Fields = append(node.Fields, resources)
	node.End = c.off
	return node, c.off, nil
}

// DecodeFilesNode owns the script/AI carrier while accepting the editor's
// valid path-only form. Optional tails are parsed only when bytes remain.
func DecodeFilesNode(data []byte) (*parsedNode, int, error) {
	c := &byteCursor{data: data}
	node := &parsedNode{Name: "Files"}
	read := func(name string, fn func(*byteCursor) (any, error)) error {
		start := c.off
		value, err := fn(c)
		if err != nil {
			return err
		}
		node.Fields = append(node.Fields, &parsedNode{Name: name, Start: start, End: c.off, Raw: append([]byte(nil), data[start:c.off]...), Value: value})
		return nil
	}
	if err := read("script_file_path", func(c *byteCursor) (any, error) { return c.stringN(2) }); err != nil {
		return nil, 0, err
	}
	if c.off == len(data) {
		node.End = c.off
		return node, c.off, nil
	}
	if err := read("script_file_content", func(c *byteCursor) (any, error) { return c.stringN(4) }); err != nil {
		return nil, 0, err
	}
	if c.off == len(data) {
		node.End = c.off
		return node, c.off, nil
	}
	if err := read("ai_files_present", func(c *byteCursor) (any, error) { return c.u32() }); err != nil {
		return nil, 0, err
	}
	if err := read("ai_error_present", func(c *byteCursor) (any, error) { return c.u32() }); err != nil {
		return nil, 0, err
	}
	if node.field("ai_error_present").Value.(uint32) != 0 {
		errorNode := &parsedNode{Name: "AIError", Start: c.off}
		for _, field := range []struct {
			name string
			read func(*byteCursor) (any, error)
		}{
			{"ai_file", func(c *byteCursor) (any, error) { return c.fixedString(260) }},
			{"line_number", func(c *byteCursor) (any, error) { return c.s32() }},
			{"message", func(c *byteCursor) (any, error) { return c.fixedString(128) }},
			{"error_code", func(c *byteCursor) (any, error) { return c.s32() }},
		} {
			start := c.off
			value, err := field.read(c)
			if err != nil {
				return nil, 0, err
			}
			errorNode.Fields = append(errorNode.Fields, &parsedNode{Name: field.name, Start: start, End: c.off, Raw: append([]byte(nil), data[start:c.off]...), Value: value})
		}
		errorNode.End = c.off
		node.Fields = append(node.Fields, &parsedNode{Name: "ai_error", Start: errorNode.Start, End: errorNode.End, Elements: []*parsedNode{errorNode}, Value: []*parsedNode{errorNode}})
	}
	var count uint32
	if node.field("ai_files_present").Value.(uint32) != 0 {
		if err := read("number_of_ai_files", func(c *byteCursor) (any, error) { return c.u32() }); err != nil {
			return nil, 0, err
		}
		count = node.field("number_of_ai_files").Value.(uint32)
	}
	if count > uint32(len(data)-c.off)/8 {
		return nil, 0, fmt.Errorf("AI file count %d exceeds remaining bytes", count)
	}
	aiFiles := &parsedNode{Name: "ai_files", Start: c.off}
	for i := uint32(0); i < count; i++ {
		entry := &parsedNode{Name: fmt.Sprintf("ai_file_%d", i+1), Start: c.off}
		for _, field := range []struct {
			name string
			read func(*byteCursor) (any, error)
		}{
			{"ai_file_name", func(c *byteCursor) (any, error) { return c.stringN(4) }},
			{"ai_file", func(c *byteCursor) (any, error) { return c.stringN(4) }},
		} {
			start := c.off
			value, err := field.read(c)
			if err != nil {
				return nil, 0, err
			}
			entry.Fields = append(entry.Fields, &parsedNode{Name: field.name, Start: start, End: c.off, Raw: append([]byte(nil), data[start:c.off]...), Value: value})
		}
		entry.End = c.off
		aiFiles.Elements = append(aiFiles.Elements, entry)
	}
	aiFiles.End = c.off
	node.Fields = append(node.Fields, aiFiles)
	if c.off < len(data) {
		start := c.off
		if _, err := c.take(1); err != nil {
			return nil, 0, err
		}
		node.Fields = append(node.Fields, &parsedNode{Name: "__END_OF_FILE_MARK__", Start: start, End: c.off, Raw: append([]byte(nil), data[start:c.off]...), Value: []byte{data[start]}})
	}
	node.End = c.off
	return node, c.off, nil
}

// DecodeTriggerNode is the owned 1.58/1.59 trigger record codec.  Trigger
// records are framing, not a generic bag of fields: the effect and condition
// counts delimit typed variable-length arrays in the middle of the record.
func DecodeTriggerNode(data []byte) (*parsedNode, int, error) {
	c := &byteCursor{data: data}
	node := &parsedNode{Name: "TriggerStruct"}
	read := func(name string, fn func(*byteCursor) (any, error)) (*parsedNode, error) {
		start := c.off
		value, err := fn(c)
		if err != nil {
			return nil, err
		}
		field := &parsedNode{Name: name, Start: start, End: c.off, Raw: append([]byte(nil), data[start:c.off]...), Value: value}
		node.Fields = append(node.Fields, field)
		return field, nil
	}
	for _, field := range []struct {
		name string
		read func(*byteCursor) (any, error)
	}{
		{"enabled", func(c *byteCursor) (any, error) { return c.u32() }},
		{"looping", func(c *byteCursor) (any, error) { return c.s8() }},
		{"execute_on_load", func(c *byteCursor) (any, error) { return c.s8() }},
		{"description_string_table_id", func(c *byteCursor) (any, error) { return c.s32() }},
		{"display_as_objective", func(c *byteCursor) (any, error) { return c.u8() }},
		{"objective_description_order", func(c *byteCursor) (any, error) { return c.u32() }},
		{"make_header", func(c *byteCursor) (any, error) { return c.u8() }},
		{"short_description_string_table_id", func(c *byteCursor) (any, error) { return c.s32() }},
		{"display_on_screen", func(c *byteCursor) (any, error) { return c.u8() }},
	} {
		if _, err := read(field.name, field.read); err != nil {
			return nil, 0, err
		}
	}
	if _, err := read("unknown", func(c *byteCursor) (any, error) {
		b, err := c.take(5)
		return append([]byte(nil), b...), err
	}); err != nil {
		return nil, 0, err
	}
	for _, field := range []struct {
		name string
		read func(*byteCursor) (any, error)
	}{
		{"mute_objectives", func(c *byteCursor) (any, error) { return c.u8() }},
		{"trigger_description", func(c *byteCursor) (any, error) { return c.stringN(4) }},
		{"trigger_name", func(c *byteCursor) (any, error) { return c.stringN(4) }},
		{"short_description", func(c *byteCursor) (any, error) { return c.stringN(4) }},
	} {
		if _, err := read(field.name, field.read); err != nil {
			return nil, 0, err
		}
	}
	effectCountField, err := read("number_of_effects", func(c *byteCursor) (any, error) { return c.s32() })
	if err != nil {
		return nil, 0, err
	}
	effectCount := int(effectCountField.Value.(int32))
	if effectCount < 0 || effectCount > maxScenarioFieldRepeat {
		return nil, 0, fmt.Errorf("effect count %d exceeds safety limit", effectCount)
	}
	effects := &parsedNode{Name: "effect_data", Start: c.off}
	for i := 0; i < effectCount; i++ {
		effect, consumed, err := DecodeEffectNode(data[c.off:])
		if err != nil {
			return nil, 0, fmt.Errorf("effect %d: %w", i, err)
		}
		effect.Start = c.off
		effect.End = c.off + consumed
		c.off += consumed
		effects.Elements = append(effects.Elements, effect)
	}
	effects.Value = effects.Elements
	effects.End = c.off
	node.Fields = append(node.Fields, effects)
	if _, err := read("effect_display_order_array", func(c *byteCursor) (any, error) {
		values := make([]any, effectCount)
		for i := range values {
			v, err := c.s32()
			if err != nil {
				return nil, err
			}
			values[i] = v
		}
		return values, nil
	}); err != nil {
		return nil, 0, err
	}
	conditionCountField, err := read("number_of_conditions", func(c *byteCursor) (any, error) { return c.s32() })
	if err != nil {
		return nil, 0, err
	}
	conditionCount := int(conditionCountField.Value.(int32))
	if conditionCount < 0 || conditionCount > maxScenarioFieldRepeat {
		return nil, 0, fmt.Errorf("condition count %d exceeds safety limit", conditionCount)
	}
	conditions := &parsedNode{Name: "condition_data", Start: c.off}
	for i := 0; i < conditionCount; i++ {
		condition, consumed, err := DecodeConditionNode(data[c.off:])
		if err != nil {
			return nil, 0, fmt.Errorf("condition %d: %w", i, err)
		}
		condition.Start = c.off
		condition.End = c.off + consumed
		c.off += consumed
		conditions.Elements = append(conditions.Elements, condition)
	}
	conditions.Value = conditions.Elements
	conditions.End = c.off
	node.Fields = append(node.Fields, conditions)
	if _, err := read("condition_display_order_array", func(c *byteCursor) (any, error) {
		values := make([]any, conditionCount)
		for i := range values {
			v, err := c.s32()
			if err != nil {
				return nil, err
			}
			values[i] = v
		}
		return values, nil
	}); err != nil {
		return nil, 0, err
	}
	node.End = c.off
	return node, c.off, nil
}

func DecodeDataHeaderNode(data []byte) (DataHeaderLayout, *parsedNode, int, error) {
	c := &byteCursor{data: data}
	node := &parsedNode{Name: "DataHeader", Start: 0}
	add := func(name string, value any, start, end int) *parsedNode {
		field := &parsedNode{Name: name, Start: start, End: end, Raw: append([]byte(nil), data[start:end]...), Value: value}
		node.Fields = append(node.Fields, field)
		return field
	}
	read := func(name string, fn func(*byteCursor) (any, error)) (any, error) {
		start := c.off
		value, err := fn(c)
		if err != nil {
			return nil, err
		}
		add(name, value, start, c.off)
		return value, nil
	}
	readGroup := func(name string, count int, fn func(*byteCursor) (any, error)) ([]any, error) {
		start := c.off
		values := make([]any, 0, count)
		for i := 0; i < count; i++ {
			value, err := fn(c)
			if err != nil {
				return nil, err
			}
			values = append(values, value)
		}
		add(name, values, start, c.off)
		return values, nil
	}
	var out DataHeaderLayout
	v, err := read("next_unit_id_to_place", func(c *byteCursor) (any, error) { return c.u32() })
	if err != nil {
		return out, nil, 0, err
	}
	out.NextUnitID = v.(uint32)
	v, err = read("version", func(c *byteCursor) (any, error) { return c.f32() })
	if err != nil {
		return out, nil, 0, err
	}
	out.Version = v.(float32)
	v, err = read("player_capacity", func(c *byteCursor) (any, error) { return c.s32() })
	if err != nil {
		return out, nil, 0, err
	}
	out.PlayerCapacity = v.(int32)
	v, err = read("gaia_player_index", func(c *byteCursor) (any, error) { return c.s32() })
	if err != nil {
		return out, nil, 0, err
	}
	out.GaiaPlayerIndex = v.(int32)
	values, err := readGroup("tribe_names", 16, func(c *byteCursor) (any, error) { return c.fixedString(256) })
	if err != nil {
		return out, nil, 0, err
	}
	for _, value := range values {
		out.TribeNames = append(out.TribeNames, value.(string))
	}
	values, err = readGroup("string_table_player_names", 16, func(c *byteCursor) (any, error) { return c.s32() })
	if err != nil {
		return out, nil, 0, err
	}
	for _, value := range values {
		out.PlayerNameIDs = append(out.PlayerNameIDs, value.(int32))
	}
	players := &parsedNode{Name: "player_data_1", Start: c.off}
	for i := 0; i < 16; i++ {
		start := c.off
		player := &parsedNode{Name: "PlayerDataOneStruct", Start: start}
		p := PlayerDataLayout{}
		addPlayer := func(name string, value any, start, end int) {
			player.Fields = append(player.Fields, &parsedNode{Name: name, Start: start, End: end, Raw: append([]byte(nil), data[start:end]...), Value: value})
		}
		readPlayer := func(name string, fn func(*byteCursor) (any, error)) (any, error) {
			s := c.off
			x, e := fn(c)
			if e == nil {
				addPlayer(name, x, s, c.off)
			}
			return x, e
		}
		v, e := readPlayer("active", func(c *byteCursor) (any, error) { return c.u32() })
		if e != nil {
			return out, nil, 0, e
		}
		p.Active = v.(uint32)
		v, e = readPlayer("human", func(c *byteCursor) (any, error) { return c.u32() })
		if e != nil {
			return out, nil, 0, e
		}
		p.Human = v.(uint32)
		v, e = readPlayer("str_sign1", func(c *byteCursor) (any, error) { return c.u16() })
		if e != nil {
			return out, nil, 0, e
		}
		p.StrSign1 = v.(uint16)
		v, e = readPlayer("civilization", func(c *byteCursor) (any, error) { return c.stringN(2) })
		if e != nil {
			return out, nil, 0, e
		}
		p.Civilization = v.(string)
		v, e = readPlayer("str_sign2", func(c *byteCursor) (any, error) { return c.u16() })
		if e != nil {
			return out, nil, 0, e
		}
		p.StrSign2 = v.(uint16)
		v, e = readPlayer("architecture_set", func(c *byteCursor) (any, error) { return c.stringN(2) })
		if e != nil {
			return out, nil, 0, e
		}
		p.ArchitectureSet = v.(string)
		v, e = readPlayer("cty_mode", func(c *byteCursor) (any, error) { return c.u32() })
		if e != nil {
			return out, nil, 0, e
		}
		p.CityMode = v.(uint32)
		player.End = c.off
		player.Value = player.Fields
		players.Elements = append(players.Elements, player)
		out.Players = append(out.Players, p)
	}
	players.End = c.off
	players.Value = players.Elements
	node.Fields = append(node.Fields, players)
	values, err = readGroup("per_player_lock_civilization", 16, func(c *byteCursor) (any, error) { return c.u32() })
	if err != nil {
		return out, nil, 0, err
	}
	for _, value := range values {
		out.LockCivilization = append(out.LockCivilization, value.(uint32))
	}
	values, err = readGroup("per_player_lock_personality", 16, func(c *byteCursor) (any, error) { return c.u32() })
	if err != nil {
		return out, nil, 0, err
	}
	for _, value := range values {
		out.LockPersonality = append(out.LockPersonality, value.(uint32))
	}
	start := c.off
	opaque, err := c.take(9)
	if err != nil {
		return out, nil, 0, err
	}
	out.Opaque = append([]byte(nil), opaque...)
	add("unknown", nil, start, c.off)
	v, err = read("filename", func(c *byteCursor) (any, error) { return c.stringN(2) })
	if err != nil {
		return out, nil, 0, err
	}
	out.Filename = v.(string)
	node.End = c.off
	return out, node, c.off, nil
}

// ScenarioHeader is the first-slice typed view of the uncompressed file header.
// Unknown values remain named opaque rather than being given guessed semantics.
type ScenarioHeader struct {
	Version        string
	HeaderLength   uint32
	Savable        int32
	Timestamp      uint32
	Instructions   string
	PlayerCount    uint32
	UnknownValue   uint32
	UnknownValue2  uint32
	UnknownNumbers []uint32
	CreatorName    string
	TriggerCount   uint32
	Raw            []byte
}

func DecodeScenarioHeader(data []byte) (ScenarioHeader, int, error) {
	c := &byteCursor{data: data}
	start := c.off
	version, err := c.fixedString(4)
	if err != nil {
		return ScenarioHeader{}, 0, err
	}
	headerLength, err := c.u32()
	if err != nil {
		return ScenarioHeader{}, 0, err
	}
	savable, err := c.s32()
	if err != nil {
		return ScenarioHeader{}, 0, err
	}
	timestamp, err := c.u32()
	if err != nil {
		return ScenarioHeader{}, 0, err
	}
	instructions, err := c.stringN(4)
	if err != nil {
		return ScenarioHeader{}, 0, err
	}
	players, err := c.u32()
	if err != nil {
		return ScenarioHeader{}, 0, err
	}
	unknown, err := c.u32()
	if err != nil {
		return ScenarioHeader{}, 0, err
	}
	unknown2, err := c.u32()
	if err != nil {
		return ScenarioHeader{}, 0, err
	}
	count, err := c.u32()
	if err != nil {
		return ScenarioHeader{}, 0, err
	}
	if count > 1024 {
		return ScenarioHeader{}, 0, fmt.Errorf("unknown header list count %d exceeds safety limit", count)
	}
	list := make([]uint32, count)
	for i := range list {
		list[i], err = c.u32()
		if err != nil {
			return ScenarioHeader{}, 0, err
		}
	}
	creator, err := c.stringN(4)
	if err != nil {
		return ScenarioHeader{}, 0, err
	}
	triggerCount, err := c.u32()
	if err != nil {
		return ScenarioHeader{}, 0, err
	}
	return ScenarioHeader{Version: version, HeaderLength: headerLength, Savable: savable, Timestamp: timestamp, Instructions: instructions, PlayerCount: players, UnknownValue: unknown, UnknownValue2: unknown2, UnknownNumbers: list, CreatorName: creator, TriggerCount: triggerCount, Raw: append([]byte(nil), data[start:c.off]...)}, c.off, nil
}

// DecodeScenarioHeaderNode exposes the typed header through the compatibility
// mutation surface used by the writer. The bytes are decoded by the owned
// header codec above; this node is only a mutable projection, not a second
// layout interpreter.
func DecodeScenarioHeaderNode(data []byte) (ScenarioHeader, *parsedNode, int, error) {
	header, consumed, err := DecodeScenarioHeader(data)
	if err != nil {
		return ScenarioHeader{}, nil, 0, err
	}
	c := &byteCursor{data: data}
	node := &parsedNode{Name: "FileHeader", Start: 0, End: consumed}
	add := func(name string, value any, start, end int) {
		node.Fields = append(node.Fields, &parsedNode{Name: name, Start: start, End: end, Raw: append([]byte(nil), data[start:end]...), Value: value})
	}
	read := func(name string, fn func(*byteCursor) (any, error)) error {
		start := c.off
		value, err := fn(c)
		if err != nil {
			return err
		}
		add(name, value, start, c.off)
		return nil
	}
	if err := read("version", func(c *byteCursor) (any, error) { return c.fixedString(4) }); err != nil {
		return ScenarioHeader{}, nil, 0, err
	}
	if err := read("header_length", func(c *byteCursor) (any, error) { return c.u32() }); err != nil {
		return ScenarioHeader{}, nil, 0, err
	}
	if err := read("savable", func(c *byteCursor) (any, error) { return c.s32() }); err != nil {
		return ScenarioHeader{}, nil, 0, err
	}
	if err := read("timestamp_of_last_save", func(c *byteCursor) (any, error) { return c.u32() }); err != nil {
		return ScenarioHeader{}, nil, 0, err
	}
	if err := read("scenario_instructions", func(c *byteCursor) (any, error) { return c.stringN(4) }); err != nil {
		return ScenarioHeader{}, nil, 0, err
	}
	if err := read("player_count", func(c *byteCursor) (any, error) { return c.u32() }); err != nil {
		return ScenarioHeader{}, nil, 0, err
	}
	if err := read("unknown_value", func(c *byteCursor) (any, error) { return c.u32() }); err != nil {
		return ScenarioHeader{}, nil, 0, err
	}
	if err := read("unknown_value_2", func(c *byteCursor) (any, error) { return c.u32() }); err != nil {
		return ScenarioHeader{}, nil, 0, err
	}
	var unknownNumbers []any
	if err := read("amount_of_unknown_numbers", func(c *byteCursor) (any, error) { return c.u32() }); err != nil {
		return ScenarioHeader{}, nil, 0, err
	}
	count := len(header.UnknownNumbers)
	start := c.off
	for i := 0; i < count; i++ {
		value, err := c.u32()
		if err != nil {
			return ScenarioHeader{}, nil, 0, err
		}
		unknownNumbers = append(unknownNumbers, value)
	}
	add("unknown_numbers", unknownNumbers, start, c.off)
	if err := read("creator_name", func(c *byteCursor) (any, error) { return c.stringN(4) }); err != nil {
		return ScenarioHeader{}, nil, 0, err
	}
	if err := read("trigger_count", func(c *byteCursor) (any, error) { return c.u32() }); err != nil {
		return ScenarioHeader{}, nil, 0, err
	}
	return header, node, c.off, nil
}

type TerrainCell struct {
	TerrainID uint8
	Elevation uint8
	Unused    [3]byte
	Layer     int16
}

type MapLayout struct {
	StringStarter1     [2]byte
	WaterDefinition    string
	StringStarter2     [2]byte
	MapColorMood       string
	StringStarter3     [2]byte
	ScriptName         string
	CollideAndCorrect  uint8
	VillagerForceDrop  uint8
	InitialPlayerViews [16][2]int32
	LockCoopAlliances  uint8
	PopulationCaps     [16]uint32
	SecondaryGameModes uint32
	Unknown5           [4]byte
	Unknown4           [4]byte
	NoWavesOnShore     int8
	Width              int32
	Height             int32
	Terrain            []TerrainCell
	Raw                []byte
}

func readString16(c *byteCursor) (string, error) { return c.stringN(2) }

func DecodeMapLayout(data []byte) (MapLayout, int, error) {
	c := &byteCursor{data: data}
	start := c.off
	var out MapLayout
	err := error(nil)
	for _, dst := range []*[2]byte{&out.StringStarter1, &out.StringStarter2, &out.StringStarter3} {
		b, e := c.take(2)
		if e != nil {
			return MapLayout{}, 0, e
		}
		copy(dst[:], b)
		if dst == &out.StringStarter1 {
			out.WaterDefinition, err = readString16(c)
		} else if dst == &out.StringStarter2 {
			out.MapColorMood, err = readString16(c)
		} else {
			out.ScriptName, err = readString16(c)
		}
		if err != nil {
			return MapLayout{}, 0, err
		}
	}
	if out.CollideAndCorrect, err = c.u8(); err != nil {
		return MapLayout{}, 0, err
	}
	if out.VillagerForceDrop, err = c.u8(); err != nil {
		return MapLayout{}, 0, err
	}
	for i := range out.InitialPlayerViews {
		if out.InitialPlayerViews[i][0], err = c.s32(); err != nil {
			return MapLayout{}, 0, err
		}
		if out.InitialPlayerViews[i][1], err = c.s32(); err != nil {
			return MapLayout{}, 0, err
		}
	}
	if out.LockCoopAlliances, err = c.u8(); err != nil {
		return MapLayout{}, 0, err
	}
	for i := range out.PopulationCaps {
		if out.PopulationCaps[i], err = c.u32(); err != nil {
			return MapLayout{}, 0, err
		}
	}
	if out.SecondaryGameModes, err = c.u32(); err != nil {
		return MapLayout{}, 0, err
	}
	if b, e := c.take(4); e != nil {
		return MapLayout{}, 0, e
	} else {
		copy(out.Unknown5[:], b)
	}
	if b, e := c.take(4); e != nil {
		return MapLayout{}, 0, e
	} else {
		copy(out.Unknown4[:], b)
	}
	if out.NoWavesOnShore, err = c.s8(); err != nil {
		return MapLayout{}, 0, err
	}
	if out.Width, err = c.s32(); err != nil {
		return MapLayout{}, 0, err
	}
	if out.Height, err = c.s32(); err != nil {
		return MapLayout{}, 0, err
	}
	if out.Width < 0 || out.Height < 0 || int64(out.Width)*int64(out.Height) > 64*1024*1024 {
		return MapLayout{}, 0, errors.New("map dimensions exceed safety limit")
	}
	out.Terrain = make([]TerrainCell, int(out.Width)*int(out.Height))
	for i := range out.Terrain {
		if out.Terrain[i].TerrainID, err = c.u8(); err != nil {
			return MapLayout{}, 0, err
		}
		if out.Terrain[i].Elevation, err = c.u8(); err != nil {
			return MapLayout{}, 0, err
		}
		if b, e := c.take(3); e != nil {
			return MapLayout{}, 0, e
		} else {
			copy(out.Terrain[i].Unused[:], b)
		}
		if out.Terrain[i].Layer, err = c.s16(); err != nil {
			return MapLayout{}, 0, err
		}
	}
	out.Raw = append([]byte(nil), data[start:c.off]...)
	return out, c.off, nil
}

// DecodeMapLayoutNode is the mutable projection used while the writer is
// being migrated. Map bytes are consumed by DecodeMapLayout's typed codec;
// this projection preserves field boundaries for existing mutation recipes.
func DecodeMapLayoutNode(data []byte) (MapLayout, *parsedNode, int, error) {
	model, consumed, err := DecodeMapLayout(data)
	if err != nil {
		return MapLayout{}, nil, 0, err
	}
	c := &byteCursor{data: data}
	node := &parsedNode{Name: "Map", Start: 0, End: consumed}
	add := func(name string, value any, start, end int) *parsedNode {
		field := &parsedNode{Name: name, Start: start, End: end, Raw: append([]byte(nil), data[start:end]...), Value: value}
		node.Fields = append(node.Fields, field)
		return field
	}
	read := func(name string, fn func(*byteCursor) (any, error)) (*parsedNode, error) {
		start := c.off
		value, err := fn(c)
		if err != nil {
			return nil, err
		}
		return add(name, value, start, c.off), nil
	}
	readBytes := func(name string, count int) (*parsedNode, error) {
		start := c.off
		value, err := c.take(count)
		if err != nil {
			return nil, err
		}
		return add(name, append([]byte(nil), value...), start, c.off), nil
	}
	if _, err := readBytes("string_starter_1", 2); err != nil {
		return MapLayout{}, nil, 0, err
	}
	if _, err := read("water_definition", func(c *byteCursor) (any, error) { return c.stringN(2) }); err != nil {
		return MapLayout{}, nil, 0, err
	}
	if _, err := readBytes("string_starter_2", 2); err != nil {
		return MapLayout{}, nil, 0, err
	}
	if _, err := read("map_color_mood", func(c *byteCursor) (any, error) { return c.stringN(2) }); err != nil {
		return MapLayout{}, nil, 0, err
	}
	if _, err := readBytes("string_starter_3", 2); err != nil {
		return MapLayout{}, nil, 0, err
	}
	if _, err := read("script_name", func(c *byteCursor) (any, error) { return c.stringN(2) }); err != nil {
		return MapLayout{}, nil, 0, err
	}
	if _, err := read("collide_and_correct", func(c *byteCursor) (any, error) { return c.u8() }); err != nil {
		return MapLayout{}, nil, 0, err
	}
	if _, err := read("villager_force_drop", func(c *byteCursor) (any, error) { return c.u8() }); err != nil {
		return MapLayout{}, nil, 0, err
	}
	views := &parsedNode{Name: "initial_player_views", Start: c.off}
	for i := 0; i < 16; i++ {
		view := &parsedNode{Name: "PlayerView", Start: c.off}
		start := c.off
		x, e := c.s32()
		if e != nil {
			return MapLayout{}, nil, 0, e
		}
		y, e := c.s32()
		if e != nil {
			return MapLayout{}, nil, 0, e
		}
		view.Fields = []*parsedNode{{Name: "location_x", Start: start, End: start + 4, Raw: append([]byte(nil), data[start:start+4]...), Value: x}, {Name: "location_y", Start: start + 4, End: c.off, Raw: append([]byte(nil), data[start+4:c.off]...), Value: y}}
		view.End = c.off
		views.Elements = append(views.Elements, view)
	}
	views.End = c.off
	node.Fields = append(node.Fields, views)
	if _, err := read("lock_coop_alliances", func(c *byteCursor) (any, error) { return c.u8() }); err != nil {
		return MapLayout{}, nil, 0, err
	}
	caps := make([]any, 0, 16)
	start := c.off
	for i := 0; i < 16; i++ {
		value, e := c.u32()
		if e != nil {
			return MapLayout{}, nil, 0, e
		}
		caps = append(caps, value)
	}
	add("per_player_population_cap", caps, start, c.off)
	if _, err := read("secondary_game_modes", func(c *byteCursor) (any, error) { return c.u32() }); err != nil {
		return MapLayout{}, nil, 0, err
	}
	if _, err := readBytes("unknown_5", 4); err != nil {
		return MapLayout{}, nil, 0, err
	}
	if _, err := readBytes("unknown_4", 4); err != nil {
		return MapLayout{}, nil, 0, err
	}
	if _, err := read("no_waves_on_shore", func(c *byteCursor) (any, error) { return c.s8() }); err != nil {
		return MapLayout{}, nil, 0, err
	}
	if _, err := read("map_width", func(c *byteCursor) (any, error) { return c.s32() }); err != nil {
		return MapLayout{}, nil, 0, err
	}
	if _, err := read("map_height", func(c *byteCursor) (any, error) { return c.s32() }); err != nil {
		return MapLayout{}, nil, 0, err
	}
	terrain := &parsedNode{Name: "terrain_data", Start: c.off}
	for i := 0; i < int(model.Width)*int(model.Height); i++ {
		cell := &parsedNode{Name: "TerrainStruct", Start: c.off}
		start := c.off
		terrainID, e := c.u8()
		if e != nil {
			return MapLayout{}, nil, 0, e
		}
		elevation, e := c.u8()
		if e != nil {
			return MapLayout{}, nil, 0, e
		}
		unused, e := c.take(3)
		if e != nil {
			return MapLayout{}, nil, 0, e
		}
		layer, e := c.s16()
		if e != nil {
			return MapLayout{}, nil, 0, e
		}
		cell.Fields = []*parsedNode{{Name: "terrain_id", Start: start, End: start + 1, Raw: append([]byte(nil), data[start:start+1]...), Value: terrainID}, {Name: "elevation", Start: start + 1, End: start + 2, Raw: append([]byte(nil), data[start+1:start+2]...), Value: elevation}, {Name: "unused", Start: start + 2, End: start + 5, Raw: append([]byte(nil), unused...), Value: append([]byte(nil), unused...)}, {Name: "layer", Start: start + 5, End: c.off, Raw: append([]byte(nil), data[start+5:c.off]...), Value: layer}}
		cell.End = c.off
		terrain.Elements = append(terrain.Elements, cell)
	}
	terrain.End = c.off
	node.Fields = append(node.Fields, terrain)
	return model, node, c.off, nil
}

type DiplomacyLayout struct {
	Stances       [][]uint32
	Individual    []byte
	Separator     uint32
	AlliedVictory []uint32
	LockTeams     uint8
	ChooseTeams   uint8
	RandomStarts  uint8
	MaxTeams      uint8
}

func DecodeDiplomacyNode(data []byte) (DiplomacyLayout, *parsedNode, int, error) {
	c := &byteCursor{data: data}
	node := &parsedNode{Name: "Diplomacy"}
	model := DiplomacyLayout{Stances: make([][]uint32, 16), AlliedVictory: make([]uint32, 16)}
	add := func(name string, value any, start, end int) *parsedNode {
		field := &parsedNode{Name: name, Start: start, End: end, Raw: append([]byte(nil), data[start:end]...), Value: value}
		node.Fields = append(node.Fields, field)
		return field
	}
	for i := range model.Stances {
		row := &parsedNode{Name: "PlayerDiplomacyStruct", Start: c.off}
		start := c.off
		values := make([]any, 16)
		model.Stances[i] = make([]uint32, 16)
		for j := range model.Stances[i] {
			value, err := c.u32()
			if err != nil {
				return DiplomacyLayout{}, nil, 0, err
			}
			model.Stances[i][j] = value
			values[j] = value
		}
		row.Fields = []*parsedNode{{Name: "stance_with_each_player", Start: start, End: c.off, Raw: append([]byte(nil), data[start:c.off]...), Value: values}}
		row.End = c.off
		if i == 0 { // The public node stores the repeated struct as one field.
			field := &parsedNode{Name: "per_player_diplomacy", Start: start, End: c.off}
			field.Elements = append(field.Elements, row)
			field.Value = field.Elements
			node.Fields = append(node.Fields, field)
		} else {
			node.field("per_player_diplomacy").Elements = append(node.field("per_player_diplomacy").Elements, row)
			node.field("per_player_diplomacy").End = c.off
		}
	}
	start := c.off
	individual, err := c.take(11520)
	if err != nil {
		return DiplomacyLayout{}, nil, 0, err
	}
	model.Individual = append([]byte(nil), individual...)
	add("individual_victories", model.Individual, start, c.off)
	value, err := c.u32()
	if err != nil {
		return DiplomacyLayout{}, nil, 0, err
	}
	model.Separator = value
	add("separator", value, c.off-4, c.off)
	values := make([]any, 16)
	start = c.off
	for i := range model.AlliedVictory {
		value, err := c.u32()
		if err != nil {
			return DiplomacyLayout{}, nil, 0, err
		}
		model.AlliedVictory[i] = value
		values[i] = value
	}
	add("per_player_allied_victory", values, start, c.off)
	for _, item := range []struct {
		name string
		dst  *uint8
	}{{"lock_teams", &model.LockTeams}, {"allow_players_choose_teams", &model.ChooseTeams}, {"random_start_points", &model.RandomStarts}, {"max_number_of_teams", &model.MaxTeams}} {
		value, err := c.u8()
		if err != nil {
			return DiplomacyLayout{}, nil, 0, err
		}
		*item.dst = value
		add(item.name, value, c.off-1, c.off)
	}
	node.End = c.off
	return model, node, c.off, nil
}

type GlobalVictoryLayout struct {
	Values [11]uint32
}

func DecodeGlobalVictoryNode(data []byte) (GlobalVictoryLayout, *parsedNode, int, error) {
	c := &byteCursor{data: data}
	node := &parsedNode{Name: "GlobalVictory"}
	model := GlobalVictoryLayout{}
	names := [...]string{"separator", "conquest_required", "ruins", "artifacts_required", "discovery", "explored_percent_of_map_required", "gold_required", "all_custom_conditions_required", "mode", "required_score_for_score_victory", "time_for_timed_game_in_10ths_of_a_year"}
	for i, name := range names {
		start := c.off
		value, err := c.u32()
		if err != nil {
			return GlobalVictoryLayout{}, nil, 0, err
		}
		model.Values[i] = value
		node.Fields = append(node.Fields, &parsedNode{Name: name, Start: start, End: c.off, Raw: append([]byte(nil), data[start:c.off]...), Value: value})
	}
	node.End = c.off
	return model, node, c.off, nil
}
