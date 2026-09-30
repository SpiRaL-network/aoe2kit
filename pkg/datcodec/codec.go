package datcodec

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"

	"aoe2kit/pkg/aoe2"
	"aoe2kit/pkg/datfile"
)

const Version = "aoe2kit.datcodec.v1"

type Document struct {
	Version       string                 `json:"version"`
	DatVersion    string                 `json:"dat_version"`
	InflatedBytes int                    `json:"inflated_bytes"`
	Sections      []Section              `json:"sections,omitempty"`
	Targets       TypedTargets           `json:"typed_targets"`
	Verification  aoe2.VerificationClaim `json:"verification"`
	payload       []byte
}

type Section struct {
	Name   string `json:"name"`
	Start  int    `json:"start"`
	End    int    `json:"end"`
	Bytes  int    `json:"bytes"`
	Typed  bool   `json:"typed"`
	SHA256 string `json:"sha256"`
}

type TypedTargets struct {
	Graphics            GraphicTarget            `json:"graphics"`
	Effects             EffectTarget             `json:"effects"`
	UnitHeaders         UnitHeaderTarget         `json:"unit_headers"`
	Techs               TechTarget               `json:"techs"`
	Civs                CivTarget                `json:"civs"`
	Units               UnitTarget               `json:"units"`
	TerrainRestrictions TerrainRestrictionTarget `json:"terrain_restrictions"`
	Terrains            TerrainTarget            `json:"terrains"`
	PlayerColours       PlayerColourTarget       `json:"player_colours"`
	Sounds              SoundTarget              `json:"sounds"`
	RandomMaps          RandomMapTarget          `json:"random_maps"`
}

type GraphicTarget struct {
	Count         int      `json:"count"`
	Present       int      `json:"present"`
	Deltas        int      `json:"deltas"`
	AngleSounds   int      `json:"angle_sounds"`
	ParticleBound int      `json:"particle_bound"`
	Editable      []string `json:"editable"`
	DecodedFields []string `json:"decoded_fields"`
}

type EffectTarget struct {
	Count    int               `json:"count"`
	Commands int               `json:"commands"`
	Spans    []TypedRecordSpan `json:"spans,omitempty"`
	Records  []datfile.Effect  `json:"records,omitempty"`
	Editable []string          `json:"editable"`
}

type UnitHeaderTarget struct {
	Count         int      `json:"count"`
	Present       int      `json:"present"`
	Tasks         int      `json:"tasks"`
	Editable      []string `json:"editable"`
	DecodedFields []string `json:"decoded_fields"`
	Note          string   `json:"note,omitempty"`
}

type TechTarget struct {
	Count    int               `json:"count"`
	Spans    []TypedRecordSpan `json:"spans,omitempty"`
	Records  []datfile.Tech    `json:"records,omitempty"`
	Editable []string          `json:"editable"`
}

type CivTarget struct {
	Count        int               `json:"count"`
	Resources    int               `json:"resources"`
	PresentUnits int               `json:"present_units"`
	Spans        []TypedRecordSpan `json:"spans,omitempty"`
	Records      []datfile.Civ     `json:"records,omitempty"`
	Editable     []string          `json:"editable"`
}

type UnitTarget struct {
	Count             int      `json:"count"`
	Type50            int      `json:"type50"`
	Creatable         int      `json:"creatable"`
	Buildings         int      `json:"buildings"`
	StorageAttributes int      `json:"storage_attributes"`
	DamageGraphics    int      `json:"damage_graphics"`
	DropSites         int      `json:"drop_sites"`
	Tasks             int      `json:"tasks"`
	Attacks           int      `json:"attacks"`
	Armours           int      `json:"armours"`
	Costs             int      `json:"costs"`
	TrainLocations    int      `json:"train_locations"`
	LinkedBuildings   int      `json:"linked_buildings"`
	TaskRawTailBytes  int      `json:"task_raw_tail_bytes"`
	Editable          []string `json:"editable"`
	Note              string   `json:"note,omitempty"`
}

type TerrainRestrictionTarget struct {
	Count           int      `json:"count"`
	TerrainRows     int      `json:"terrain_rows"`
	Editable        []string `json:"editable"`
	PassGraphicNote string   `json:"pass_graphic_note,omitempty"`
}

type TerrainTarget struct {
	Count    int      `json:"count"`
	Editable []string `json:"editable"`
	Note     string   `json:"note,omitempty"`
}

type PlayerColourTarget struct {
	Count    int      `json:"count"`
	Editable []string `json:"editable"`
}

type SoundTarget struct {
	Count    int      `json:"count"`
	Items    int      `json:"items"`
	Editable []string `json:"editable"`
}

type RandomMapTarget struct {
	Count      int    `json:"count"`
	Lands      int    `json:"lands"`
	Terrains   int    `json:"terrains"`
	Units      int    `json:"units"`
	Elevations int    `json:"elevations"`
	Note       string `json:"note,omitempty"`
}

type TypedRecordSpan struct {
	Index  int    `json:"index"`
	Name   string `json:"name,omitempty"`
	Start  int    `json:"start"`
	End    int    `json:"end"`
	Bytes  int    `json:"bytes"`
	SHA256 string `json:"sha256"`
}

type RoundTripReport struct {
	Version               string                 `json:"version"`
	Path                  string                 `json:"path"`
	OK                    bool                   `json:"ok"`
	CompressedBytes       int                    `json:"compressed_bytes"`
	InflatedBytes         int                    `json:"inflated_bytes"`
	PayloadIdentical      bool                   `json:"payload_identical"`
	FirstDiffOffset       *int                   `json:"first_diff_offset,omitempty"`
	OriginalPayloadSHA256 string                 `json:"original_payload_sha256"`
	EncodedPayloadSHA256  string                 `json:"encoded_payload_sha256"`
	SectionCount          int                    `json:"section_count"`
	TypedTargets          TypedTargets           `json:"typed_targets"`
	Verification          aoe2.VerificationClaim `json:"verification"`
}

func Decode(compressed []byte) (*Document, error) {
	payload, err := datfile.Inflate(compressed)
	if err != nil {
		return nil, err
	}
	idx, err := datfile.Parse(payload)
	if err != nil {
		return nil, err
	}
	doc := &Document{
		Version:       Version,
		DatVersion:    idx.Version,
		InflatedBytes: len(payload),
		payload:       payload,
		Verification: aoe2.VerificationClaim{
			StructureVerified: true,
			EngineVerified:    false,
			Note:              "datcodec decode frames the current-DE DAT payload into ordered sections and typed target records; in-engine behavior remains a separate oracle.",
		},
	}
	for _, span := range idx.Spans {
		doc.Sections = append(doc.Sections, Section{
			Name:   span.Name,
			Start:  span.Start,
			End:    span.End,
			Bytes:  span.Len(),
			Typed:  isTypedTargetSpan(span.Name),
			SHA256: sha(payload[span.Start:span.End]),
		})
	}
	doc.Targets.Graphics = graphicTarget(idx.Graphics)
	doc.Targets.Effects = effectTarget(idx.Effects, payload)
	doc.Targets.UnitHeaders = unitHeaderTarget(idx.UnitHeaders)
	doc.Targets.Techs = techTarget(idx.Techs, payload)
	doc.Targets.Civs = civTarget(idx.Civs, payload)
	doc.Targets.Units = unitTarget(idx.Civs)
	doc.Targets.TerrainRestrictions = terrainRestrictionTarget(idx.TerrainRestrictions)
	doc.Targets.Terrains = terrainTarget(idx.Terrains)
	doc.Targets.PlayerColours = playerColourTarget(idx.PlayerColours)
	doc.Targets.Sounds = soundTarget(idx.Sounds)
	doc.Targets.RandomMaps = randomMapTarget(idx.RandomMaps)
	return doc, nil
}

func graphicTarget(graphics []datfile.Graphic) GraphicTarget {
	target := GraphicTarget{
		Count: len(graphics),
		Editable: []string{
			"name",
			"file_name",
			"particle_effect_name",
			"slp",
			"is_loaded",
			"old_color_flag",
			"layer",
			"player_color",
			"rainbow",
			"transparent_selection",
			"coordinates",
			"sound_id",
			"wwise_sound_id",
			"frame_count",
			"speed_multiplier",
			"frame_duration",
			"replay_delay",
			"sequence_type",
			"mirroring_mode",
			"editor_flag",
		},
		DecodedFields: []string{
			"slp",
			"is_loaded",
			"old_color_flag",
			"layer",
			"player_color",
			"rainbow",
			"transparent_selection",
			"coordinates",
			"sound_id",
			"wwise_sound_id",
			"angle_sounds_used",
			"frame_count",
			"angle_count",
			"speed_multiplier",
			"frame_duration",
			"replay_delay",
			"sequence_type",
			"id",
			"mirroring_mode",
			"editor_flag",
			"deltas",
			"angle_sounds",
		},
	}
	for _, graphic := range graphics {
		if !graphic.Present {
			continue
		}
		target.Present++
		target.Deltas += len(graphic.Deltas)
		target.AngleSounds += len(graphic.AngleSounds)
		if graphic.ParticleEffectName.Value != "" {
			target.ParticleBound++
		}
	}
	return target
}

func (doc *Document) EncodePayload() ([]byte, error) {
	if doc == nil {
		return nil, fmt.Errorf("nil datcodec document")
	}
	if len(doc.Sections) == 0 {
		return nil, fmt.Errorf("document has no sections")
	}
	var out bytes.Buffer
	for i, section := range doc.Sections {
		if section.Start < 0 || section.End < section.Start || section.End > len(doc.payload) {
			return nil, fmt.Errorf("section %d %s has invalid span %d..%d for payload %d", i, section.Name, section.Start, section.End, len(doc.payload))
		}
		if out.Len() != section.Start {
			return nil, fmt.Errorf("section %d %s starts at %d, but encoder is at %d", i, section.Name, section.Start, out.Len())
		}
		out.Write(doc.payload[section.Start:section.End])
	}
	if out.Len() != len(doc.payload) {
		return nil, fmt.Errorf("encoded payload length %d, want %d", out.Len(), len(doc.payload))
	}
	return out.Bytes(), nil
}

func RoundTripFile(path string) (RoundTripReport, error) {
	compressed, err := os.ReadFile(path)
	if err != nil {
		return RoundTripReport{}, err
	}
	doc, err := Decode(compressed)
	if err != nil {
		return RoundTripReport{}, err
	}
	encoded, err := doc.EncodePayload()
	if err != nil {
		return RoundTripReport{}, err
	}
	identical := bytes.Equal(doc.payload, encoded)
	report := RoundTripReport{
		Version:               Version,
		Path:                  path,
		OK:                    identical,
		CompressedBytes:       len(compressed),
		InflatedBytes:         len(doc.payload),
		PayloadIdentical:      identical,
		OriginalPayloadSHA256: sha(doc.payload),
		EncodedPayloadSHA256:  sha(encoded),
		SectionCount:          len(doc.Sections),
		TypedTargets:          doc.Targets,
		Verification: aoe2.VerificationClaim{
			StructureVerified: identical,
			EngineVerified:    false,
			Note:              "roundtrip compares the decompressed DAT payload emitted by the codec; compressed deflate bytes are not expected to be stable across encoders.",
		},
	}
	if !identical {
		diff := firstDiff(doc.payload, encoded)
		report.FirstDiffOffset = &diff
	}
	return report, nil
}

func effectTarget(effects []datfile.Effect, payload []byte) EffectTarget {
	target := EffectTarget{
		Count:    len(effects),
		Editable: []string{"name", "commands"},
	}
	for _, effect := range effects {
		target.Commands += len(effect.Commands)
		target.Spans = append(target.Spans, TypedRecordSpan{
			Index:  effect.Index,
			Name:   effect.Name,
			Start:  effect.Span.Start,
			End:    effect.Span.End,
			Bytes:  effect.Span.Len(),
			SHA256: sha(payload[effect.Span.Start:effect.Span.End]),
		})
	}
	return target
}

func unitHeaderTarget(headers []datfile.UnitHeader) UnitHeaderTarget {
	target := UnitHeaderTarget{
		Count: len(headers),
		Editable: []string{
			"tasks.record_type",
			"tasks.id",
			"tasks.is_default",
			"tasks.action_type",
			"tasks.object_class",
			"tasks.object_id",
			"tasks.terrain_id",
			"tasks.attribute_types",
			"tasks.work_values",
			"tasks.targeting_flags",
			"tasks.combat_level",
		},
		DecodedFields: []string{
			"exists",
			"task_count",
			"tasks.record_type",
			"tasks.id",
			"tasks.action_type",
			"tasks.object_class",
			"tasks.object_id",
			"tasks.terrain_id",
			"tasks.attribute_types",
			"tasks.work_values",
			"tasks.targeting_flags",
			"tasks.combat_level",
		},
		Note: "unit header task rows support fixed-width field patching; count-changing task list edits still require full record rebuilds.",
	}
	for _, header := range headers {
		if !header.Exists {
			continue
		}
		target.Present++
		target.Tasks += len(header.Tasks)
	}
	return target
}

func techTarget(techs []datfile.Tech, payload []byte) TechTarget {
	target := TechTarget{
		Count:    len(techs),
		Editable: []string{"required_techs", "resource_costs", "effect_id", "language_dll_ids", "name", "repeatable", "research_locations"},
	}
	for _, tech := range techs {
		target.Spans = append(target.Spans, TypedRecordSpan{
			Index:  tech.Index,
			Name:   tech.Name,
			Start:  tech.Span.Start,
			End:    tech.Span.End,
			Bytes:  tech.Span.Len(),
			SHA256: sha(payload[tech.Span.Start:tech.Span.End]),
		})
	}
	return target
}

func civTarget(civs []datfile.Civ, payload []byte) CivTarget {
	target := CivTarget{
		Count:    len(civs),
		Editable: []string{"name", "tech_tree_id", "team_bonus_id", "resources", "icon_set"},
	}
	for _, civ := range civs {
		target.Resources += len(civ.Resources)
		for _, unit := range civ.Units {
			if unit.Present {
				target.PresentUnits++
			}
		}
		target.Spans = append(target.Spans, TypedRecordSpan{
			Index:  civ.Index,
			Name:   civ.Name,
			Start:  civ.Span.Start,
			End:    civ.Span.End,
			Bytes:  civ.Span.Len(),
			SHA256: sha(payload[civ.Span.Start:civ.Span.End]),
		})
	}
	return target
}

func unitTarget(civs []datfile.Civ) UnitTarget {
	target := UnitTarget{
		Editable: []string{
			"class",
			"id",
			"name",
			"hit_points",
			"line_of_sight",
			"movement_type",
			"graphics",
			"enabled",
			"storage_attribute_rows",
			"damage_graphic_rows",
			"set_damage_graphics",
			"type50_scalar_fields",
			"type50_attack_rows",
			"type50_armour_rows",
			"set_type50_attacks",
			"set_type50_armours",
			"first_train_location",
			"creatable_cost_rows",
			"train_location_rows",
			"set_train_locations",
			"creatable_button_fields",
			"set_tasks",
		},
		Note: "unit target parses current-DE unit bodies through attacks/armours, creatable costs, train locations, action task headers, damage graphics, and building annex links; fixed-width row mutation is supported, and decoded damage-graphic/type50/train-location/task lists can be count-changing replaced by rebuilding exactly one unit record. Action task rows expose named leading fields plus raw tail bytes; set_tasks requires each row to preserve the exact raw tail length.",
	}
	for _, civ := range civs {
		for _, unit := range civ.Units {
			if !unit.Present {
				continue
			}
			target.Count++
			target.StorageAttributes += len(unit.Attributes)
			target.DamageGraphics += len(unit.DamageGraphics)
			if unit.Action != nil {
				target.DropSites += len(unit.Action.DropSites)
				target.Tasks += len(unit.Action.Tasks)
				for _, task := range unit.Action.Tasks {
					target.TaskRawTailBytes += len(task.RawTail)
				}
			}
			if unit.Type50 != nil {
				target.Type50++
				target.Attacks += len(unit.Type50.Attacks)
				target.Armours += len(unit.Type50.Armours)
			}
			if unit.Creatable != nil {
				target.Creatable++
				target.Costs += len(unit.Creatable.Costs)
				target.TrainLocations += len(unit.Creatable.TrainLocations)
			}
			if unit.Building != nil {
				target.Buildings++
				target.LinkedBuildings += len(unit.Building.LinkedBuildings)
			}
		}
	}
	return target
}

func terrainRestrictionTarget(restrictions []datfile.TerrainRestriction) TerrainRestrictionTarget {
	target := TerrainRestrictionTarget{
		Count:           len(restrictions),
		Editable:        []string{"passability"},
		PassGraphicNote: "pass-graphic payload is framed as raw 16-byte rows until the DE substructure is independently verified.",
	}
	for _, restriction := range restrictions {
		target.TerrainRows += len(restriction.TerrainRows)
	}
	return target
}

func terrainTarget(terrains []datfile.Terrain) TerrainTarget {
	return TerrainTarget{
		Count:    len(terrains),
		Editable: []string{"name", "name_2", "overlay_mask_name"},
		Note:     "terrain records are currently typed for debug strings and record spans; scalar terrain fields remain raw-preserved.",
	}
}

func playerColourTarget(colours []datfile.PlayerColour) PlayerColourTarget {
	return PlayerColourTarget{
		Count: len(colours),
		Editable: []string{
			"base",
			"unit_outline_colour",
			"selection_colours",
			"minimap_colours",
			"statistics_text_color",
		},
	}
}

func soundTarget(sounds []datfile.Sound) SoundTarget {
	target := SoundTarget{
		Count:    len(sounds),
		Editable: []string{"play_delay", "cache_time", "total_probability", "items"},
	}
	for _, sound := range sounds {
		target.Items += len(sound.Items)
	}
	return target
}

func randomMapTarget(maps []datfile.RandomMapInfo) RandomMapTarget {
	target := RandomMapTarget{
		Count: len(maps),
		Note:  "random map records are currently typed for structural counts/spans; land/terrain/unit/elevation rows remain raw-preserved.",
	}
	for _, info := range maps {
		target.Lands += int(info.Lands)
		target.Terrains += int(info.Terrains)
		target.Units += int(info.Units)
		target.Elevations += int(info.Elevations)
	}
	return target
}

func isTypedTargetSpan(name string) bool {
	return name == "graphics_header" || name == "graphic_pointers" ||
		name == "unit_headers_header" ||
		name == "effects_header" || name == "techs_header" || name == "civs_header" ||
		name == "terrain_header" || name == "terrain_float_pointers" || name == "terrain_pass_graphic_pointers" ||
		name == "terrain_restrictions" || name == "player_colours_header" || name == "player_colours" ||
		name == "sounds_header" || name == "terrain_block" || name == "random_maps" ||
		hasRecordPrefix(name, "graphic[") ||
		hasRecordPrefix(name, "unit_header[") ||
		hasRecordPrefix(name, "sound[") ||
		hasRecordPrefix(name, "effect[") || hasRecordPrefix(name, "tech[") || hasRecordPrefix(name, "civ[")
}

func hasRecordPrefix(name, prefix string) bool {
	return len(name) >= len(prefix) && name[:len(prefix)] == prefix
}

func sha(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func firstDiff(a, b []byte) int {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	for i := 0; i < n; i++ {
		if a[i] != b[i] {
			return i
		}
	}
	return n
}
