package scenario

import (
	"bytes"
	"compress/flate"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"strconv"
	"strings"

	"aoe2kit/pkg/triggergraph"
)

const defaultMaxInflatedScenarioBytes = 32 * 1024 * 1024
const maxScenarioFieldRepeat = 1_000_000

type File struct {
	Path            string        `json:"path,omitempty"`
	Version         string        `json:"version"`
	PlayerCount     int           `json:"player_count"`
	HeaderBytes     int           `json:"header_bytes"`
	CompressedBytes int           `json:"compressed_body_bytes"`
	InflatedBytes   int           `json:"inflated_body_bytes"`
	Sections        []SectionInfo `json:"sections,omitempty"`
	Triggers        *TriggerInfo  `json:"triggers,omitempty"`
	Units           *UnitInfo     `json:"units,omitempty"`
	Map             *MapInfo      `json:"map,omitempty"`
	Players         []PlayerInfo  `json:"players,omitempty"`
	AI              []AIFileInfo  `json:"ai,omitempty"`

	header         []byte
	body           []byte
	original       []byte
	originalHeader []byte
	originalBody   []byte
	headerRoot     *parsedNode
	root           *parsedRoot
}

type ParseOptions struct {
	MaxInflatedBytes int
	// StopBeforeSection, when set, halts body parsing when the named section is
	// reached and returns the sections parsed so far. Used to read content
	// (units/terrain) from scenarios whose later sections (e.g. Triggers) use a
	// format kit cannot yet fully decode — Microsoft campaign scenarios.
	StopBeforeSection string
}

type SectionInfo struct {
	Name  string `json:"name"`
	Start int    `json:"start"`
	End   int    `json:"end"`
}

type TriggerInfo struct {
	Version       float64          `json:"version"`
	Count         int              `json:"count"`
	GraphSHA256   string           `json:"graph_sha256,omitempty"`
	DisplayOrder  []uint32         `json:"display_order,omitempty"`
	Variables     int              `json:"variables"`
	RecordStart   int              `json:"record_start"`
	RecordEnd     int              `json:"record_end"`
	Triggers      []TriggerSummary `json:"triggers,omitempty"`
	InvariantOK   bool             `json:"invariant_ok"`
	InvariantNote string           `json:"invariant_note,omitempty"`
}

type TriggerSummary struct {
	Index                         int                `json:"index"`
	Name                          string             `json:"name"`
	Enabled                       uint32             `json:"enabled"`
	Looping                       int8               `json:"looping"`
	ExecuteOnLoad                 int                `json:"execute_on_load"`
	DescriptionStringTableID      int                `json:"description_string_table_id"`
	DisplayAsObjective            int                `json:"display_as_objective"`
	ObjectiveDescriptionOrder     uint32             `json:"objective_description_order"`
	MakeHeader                    int                `json:"make_header"`
	ShortDescriptionStringTableID int                `json:"short_description_string_table_id"`
	DisplayOnScreen               int                `json:"display_on_screen"`
	MuteObjectives                int                `json:"mute_objectives"`
	Effects                       int                `json:"effects"`
	EffectData                    []EffectSummary    `json:"effect_data,omitempty"`
	Conditions                    int                `json:"conditions"`
	ConditionData                 []ConditionSummary `json:"condition_data,omitempty"`
	RecordStart                   int                `json:"record_start"`
	RecordEnd                     int                `json:"record_end"`
}

type UnitInfo struct {
	NumberOfUnitSections int               `json:"number_of_unit_sections"`
	NumberOfPlayers      int               `json:"number_of_players"`
	Sections             []PlayerUnitsInfo `json:"sections"`
	Total                int               `json:"total"`
}

type PlayerUnitsInfo struct {
	Player int           `json:"player"`
	Count  int           `json:"count"`
	Units  []UnitSummary `json:"units,omitempty"`
}

type UnitSummary struct {
	Index           int     `json:"index"`
	ReferenceID     int     `json:"reference_id"`
	UnitConst       int     `json:"unit_const"`
	X               float64 `json:"x"`
	Y               float64 `json:"y"`
	Z               float64 `json:"z"`
	Rotation        float64 `json:"rotation"`
	Status          int     `json:"status"`
	CaptionStringID int     `json:"caption_string_id,omitempty"`
	CaptionString   string  `json:"caption_string,omitempty"`
}

type MapInfo struct {
	Width              int                    `json:"width"`
	Height             int                    `json:"height"`
	TileCount          int                    `json:"tile_count"`
	TerrainCounts      map[int]int            `json:"terrain_counts"`
	SecondaryGameModes SecondaryGameModesInfo `json:"secondary_game_modes"`
}

// SecondaryGameModesInfo is the editor's Map.secondary_game_modes bitfield.
// The names are derived from the measured DE bit assignments, not guessed from
// the raw integer.
type SecondaryGameModesInfo struct {
	Bits  int      `json:"bits"`
	Names []string `json:"names,omitempty"`
}

type PlayerInfo struct {
	Player           int                        `json:"player"`
	PlayerLabel      string                     `json:"player_label,omitempty"`
	SectionIndex     int                        `json:"section_index"`
	IndexBase        string                     `json:"index_base,omitempty"`
	FieldReferences  map[string]PlayerReference `json:"field_references,omitempty"`
	Active           bool                       `json:"active"`
	Human            bool                       `json:"human"`
	TribeName        string                     `json:"tribe_name,omitempty"`
	NameStringID     int                        `json:"name_string_id"`
	Civilization     string                     `json:"civilization,omitempty"`
	Architecture     string                     `json:"architecture,omitempty"`
	LockCivilization bool                       `json:"lock_civilization"`
	LockPersonality  bool                       `json:"lock_personality"`
	StartingAge      int                        `json:"starting_age"`
	StartingAgeName  string                     `json:"starting_age_name,omitempty"`
	Color            int                        `json:"color"`
	ColorName        string                     `json:"color_name,omitempty"`
	BasePriority     int                        `json:"base_priority"`
	PopulationLimit  int                        `json:"population_limit"`
	AIName           string                     `json:"ai_name,omitempty"`
	AIType           int                        `json:"ai_type,omitempty"`
}

// PlayerReference keeps the editor-facing slot separate from the raw array
// index. Scenario sections do not all use the same player indexing convention.
type PlayerReference struct {
	Player       string `json:"player"`
	SectionIndex int    `json:"section_index"`
	IndexBase    string `json:"index_base"`
}

type AIFileInfo struct {
	Index         int    `json:"index"`
	Name          string `json:"name,omitempty"`
	ContentBytes  int    `json:"content_bytes,omitempty"`
	ContentSHA256 string `json:"content_sha256,omitempty"`
}

type DescribeOptions struct {
	Sections []string
	Full     bool
}

type Description struct {
	Path            string        `json:"path,omitempty"`
	Version         string        `json:"version"`
	PlayerCount     int           `json:"player_count"`
	HeaderBytes     int           `json:"header_bytes"`
	CompressedBytes int           `json:"compressed_body_bytes"`
	InflatedBytes   int           `json:"inflated_body_bytes"`
	SectionIndex    []SectionInfo `json:"section_index"`
	Players         []PlayerInfo  `json:"players,omitempty"`
	Triggers        *TriggerInfo  `json:"triggers,omitempty"`
	Units           *UnitInfo     `json:"units,omitempty"`
	Map             *MapInfo      `json:"map,omitempty"`
	AI              []AIFileInfo  `json:"ai,omitempty"`
	Sections        []SectionDump `json:"sections,omitempty"`
}

type SectionDump struct {
	Name   string     `json:"name"`
	Start  int        `json:"start"`
	End    int        `json:"end"`
	Fields []NodeDump `json:"fields,omitempty"`
}

type NodeDump struct {
	Name     string     `json:"name"`
	Start    int        `json:"start"`
	End      int        `json:"end"`
	Value    any        `json:"value,omitempty"`
	Fields   []NodeDump `json:"fields,omitempty"`
	Elements []NodeDump `json:"elements,omitempty"`
}

func Open(path string) (*File, error) {
	return OpenWithOptions(path, ParseOptions{})
}

func InflatedBodyFile(path string) ([]byte, error) {
	file, err := Open(path)
	if err != nil {
		return nil, err
	}
	return append([]byte(nil), file.body...), nil
}

func OpenWithOptions(path string, opts ParseOptions) (*File, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	file, err := ParseWithOptions(data, opts)
	if err != nil {
		return nil, err
	}
	file.Path = path
	return file, nil
}

func Parse(data []byte) (*File, error) {
	return ParseWithOptions(data, ParseOptions{})
}

func ParseWithOptions(data []byte, opts ParseOptions) (*File, error) {
	if len(data) < 8 {
		return nil, errors.New("scenario too short")
	}
	headerModel, headerNode, headerLen, err := DecodeScenarioHeaderNode(data)
	if err != nil {
		return nil, fmt.Errorf("decode FileHeader: %w", err)
	}
	version := headerModel.Version
	playerCount := int(headerModel.PlayerCount)
	if !SupportsReadVersion(version) {
		return nil, unsupportedReadVersionError(version)
	}
	bodySpec, err := LoadDESpecForVersion(version)
	if err != nil {
		return nil, err
	}
	header := append([]byte(nil), data[:headerLen]...)
	maxScenarioBytes := opts.MaxInflatedBytes
	if maxScenarioBytes == 0 {
		maxScenarioBytes = maxInflatedScenarioBytes()
	}
	body, err := inflateRawLimited(data[headerLen:], maxScenarioBytes)
	if err != nil {
		return nil, fmt.Errorf("inflate scenario body: %w", err)
	}
	p := parser{
		data:        body,
		sections:    map[string]*parsedSection{},
		stopBefore:  opts.StopBeforeSection,
		strictOwned: true,
		version:     version,
	}
	root, err := p.parse(bodySpec)
	opaqueTriggerVariant := false
	if err != nil && version == "1.59" && opts.StopBeforeSection == "" {
		prefixParser := parser{
			data:        body,
			sections:    map[string]*parsedSection{},
			stopBefore:  "Triggers",
			strictOwned: true,
			version:     version,
		}
		prefixRoot, prefixErr := prefixParser.parse(bodySpec)
		if prefixErr == nil && hasOpaque159TriggerVariant(body, prefixParser.off) {
			opaque := &parsedSection{
				Name:  "Triggers",
				Start: prefixParser.off,
				End:   len(body),
				Raw:   append([]byte(nil), body[prefixParser.off:]...),
			}
			prefixRoot.Sections = append(prefixRoot.Sections, opaque)
			prefixParser.sections[opaque.Name] = opaque
			prefixParser.off = len(body)
			root, p, err = prefixRoot, prefixParser, nil
			opaqueTriggerVariant = true
		}
	}
	if version == "1.59" && opts.StopBeforeSection == "" && !opaqueTriggerVariant {
		// Do not choose a 1.59 layout from parse success alone. The compact
		// schema can consume an editor record while leaving a plausible tree.
		// Parse both complete candidates and use the condition marker as the
		// byte-level discriminator: 33 is compact, 34 is editor output.
		editorSpec, specErr := LoadDESpecForVersion(version)
		if specErr != nil {
			return nil, specErr
		}
		patchDE159EditorConditionSpec(editorSpec)
		editorParser := parser{
			data:        body,
			sections:    map[string]*parsedSection{},
			stopBefore:  opts.StopBeforeSection,
			strictOwned: true,
			version:     version,
		}
		editorRoot, editorParseErr := editorParser.parse(editorSpec)
		compactMatch, compactHasConditions := conditionLayoutMarker(root, 33)
		editorMatch, editorHasConditions := conditionLayoutMarker(editorRoot, 34)
		compactComplete := err == nil && p.off == len(body) && (compactMatch || !compactHasConditions)
		editorComplete := editorParseErr == nil && editorParser.off == len(body) && (editorMatch || !editorHasConditions)
		switch {
		case compactComplete && !editorComplete:
			// Keep the compact parse.
		case editorComplete && !compactComplete:
			root, p, bodySpec, err = editorRoot, editorParser, editorSpec, nil
		case compactComplete && editorComplete:
			if compactHasConditions || editorHasConditions {
				return nil, errors.New("1.59 scenario trigger layout is ambiguous")
			}
			// With no conditions there is no layout-bearing field; compact is
			// canonical and produces the same bytes.
		case err == nil && editorParseErr != nil:
			// Preserve the successful compact parse.
		default:
			return nil, fmt.Errorf("1.59 trigger layout discriminator rejected compact parse (%v) and editor parse (%v)", err, editorParseErr)
		}
	}
	if err != nil {
		return nil, err
	}
	if opts.StopBeforeSection == "" && p.off != len(body) {
		return nil, fmt.Errorf("scenario body parse stopped at %d of %d", p.off, len(body))
	}
	file := &File{
		Version:         version,
		PlayerCount:     playerCount,
		HeaderBytes:     len(header),
		CompressedBytes: len(data) - headerLen,
		InflatedBytes:   len(body),
		Sections:        root.sectionInfos(),
		header:          header,
		body:            body,
		original:        append([]byte(nil), data...),
		originalHeader:  append([]byte(nil), header...),
		originalBody:    append([]byte(nil), body...),
		headerRoot:      headerNode,
		root:            root,
	}
	if root.section("Triggers") != nil && root.section("Triggers").field("number_of_triggers") != nil {
		if triggers, err := root.triggerInfo(); err == nil {
			file.Triggers = triggers
		} else {
			return nil, err
		}
	}
	if root.section("Units") != nil {
		if units, err := root.unitInfo(); err == nil {
			file.Units = units
		} else {
			return nil, err
		}
	}
	if mapInfo, err := root.mapInfo(); err == nil {
		file.Map = mapInfo
	} else {
		return nil, err
	}
	file.Players = root.playerInfo()
	file.AI = root.aiInfo()
	return file, nil
}

func maxInflatedScenarioBytes() int {
	if os.Getenv("AOE2KIT_ALLOW_HUGE_SCENARIO") == "1" {
		return 0
	}
	raw := strings.TrimSpace(os.Getenv("AOE2KIT_MAX_SCENARIO_MB"))
	if raw == "" {
		return defaultMaxInflatedScenarioBytes
	}
	mb, err := strconv.Atoi(raw)
	if err != nil || mb < 0 {
		return defaultMaxInflatedScenarioBytes
	}
	return mb * 1024 * 1024
}

func SupportsReadVersion(version string) bool {
	switch version {
	case "1.58", "1.59":
		return true
	default:
		return false
	}
}

const supportedReadVersions = "1.58, 1.59"

func unsupportedReadVersionError(version string) error {
	return fmt.Errorf("unsupported scenario version %s (supported: %s)", version, supportedReadVersions)
}

// ReadScenarioVersion reads only the FileHeader. It is intentionally cheap so
// corpus commands can classify unsupported versions before attempting a body
// parse.
func ReadScenarioVersion(data []byte) (string, error) {
	if len(data) < 8 {
		return "", errors.New("scenario too short")
	}
	header, _, _, err := DecodeScenarioHeaderNode(data)
	if err != nil {
		return "", fmt.Errorf("decode FileHeader: %w", err)
	}
	if header.Version == "" {
		return "", errors.New("FileHeader has no version")
	}
	return header.Version, nil
}

func ReadScenarioVersionFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return ReadScenarioVersion(data)
}

func SupportsWriteVersion(version string) bool {
	switch version {
	case "1.57", "1.58", "1.59":
		return true
	default:
		return false
	}
}

func (f *File) Describe(opts DescribeOptions) Description {
	desc := Description{
		Path:            f.Path,
		Version:         f.Version,
		PlayerCount:     f.PlayerCount,
		HeaderBytes:     f.HeaderBytes,
		CompressedBytes: f.CompressedBytes,
		InflatedBytes:   f.InflatedBytes,
		SectionIndex:    f.Sections,
		Players:         f.Players,
		Triggers:        f.Triggers,
		Units:           f.Units,
		Map:             f.Map,
		AI:              f.AI,
	}
	if f.root == nil {
		return desc
	}
	selected := sectionSelection(opts.Sections)
	if opts.Full || len(selected) > 0 {
		for _, section := range f.root.Sections {
			if len(selected) > 0 && !selected[section.Name] {
				continue
			}
			desc.Sections = append(desc.Sections, SectionDump{
				Name:   section.Name,
				Start:  section.Start,
				End:    section.End,
				Fields: dumpNodes(section.Fields),
			})
		}
	}
	return desc
}

func (f *File) RebuildBody() ([]byte, error) {
	if f.root == nil {
		return nil, errors.New("scenario has no parsed root")
	}
	return f.root.raw(), nil
}

// rawSection exposes the original bytes of a parsed section to Kit-owned
// codecs. The legacy parser remains the temporary section locator; the new
// codec owns decoding and preservation of the slice it receives.
func (f *File) rawSection(name string) []byte {
	if f.root == nil {
		return nil
	}
	section := f.root.section(name)
	if section == nil {
		return nil
	}
	return append([]byte(nil), section.raw()...)
}

// PreservedDocument returns the lossless carrier used by Kit-owned codecs.
// Section migration can replace the temporary parser locator without changing
// the preservation contract.
func (f *File) PreservedDocument() PreservedDocument {
	doc := NewPreservedDocument(f.originalHeader, f.originalBody)
	doc.OriginalFile = append([]byte(nil), f.original...)
	return doc
}

func (f *File) VerifyRebuild() error {
	rebuilt, err := f.RebuildBody()
	if err != nil {
		return err
	}
	if !bytes.Equal(rebuilt, f.body) {
		return fmt.Errorf("rebuilt body differs: got %d bytes want %d", len(rebuilt), len(f.body))
	}
	compressed, err := DeflateRaw(rebuilt)
	if err != nil {
		return err
	}
	roundTrip, err := InflateRaw(compressed)
	if err != nil {
		return err
	}
	if !bytes.Equal(roundTrip, rebuilt) {
		return errors.New("rebuilt body changed after deflate/inflate")
	}
	return nil
}

func InflateRaw(compressed []byte) ([]byte, error) {
	r := flate.NewReader(bytes.NewReader(compressed))
	defer r.Close()
	return io.ReadAll(r)
}

func inflateRawLimited(compressed []byte, maxBytes int) ([]byte, error) {
	if maxBytes <= 0 {
		return InflateRaw(compressed)
	}
	r := flate.NewReader(bytes.NewReader(compressed))
	defer r.Close()
	body, err := io.ReadAll(io.LimitReader(r, int64(maxBytes)+1))
	if err != nil {
		return nil, err
	}
	if len(body) > maxBytes {
		return nil, fmt.Errorf("inflated scenario body is above safety limit %d bytes; set AOE2KIT_MAX_SCENARIO_MB higher or AOE2KIT_ALLOW_HUGE_SCENARIO=1 for an intentional full parse", maxBytes)
	}
	return body, nil
}

func DeflateRaw(payload []byte) ([]byte, error) {
	var out bytes.Buffer
	w, err := flate.NewWriter(&out, flate.BestCompression)
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

type parsedRoot struct {
	Sections []*parsedSection
}

func (r *parsedRoot) raw() []byte {
	var out []byte
	for _, section := range r.Sections {
		out = append(out, section.raw()...)
	}
	return out
}

func (r *parsedRoot) sectionInfos() []SectionInfo {
	out := make([]SectionInfo, 0, len(r.Sections))
	for _, section := range r.Sections {
		out = append(out, SectionInfo{Name: section.Name, Start: section.Start, End: section.End})
	}
	return out
}

func (r *parsedRoot) section(name string) *parsedSection {
	for _, section := range r.Sections {
		if section.Name == name {
			return section
		}
	}
	return nil
}

// hasOpaque159TriggerVariant recognizes the official 1.59 campaign framing
// that Kit has not decoded yet. It requires a valid trigger header, a valid
// first owned record, and the observed five-byte separator immediately after
// that record. The caller preserves the remaining trigger/file tail verbatim.
func hasOpaque159TriggerVariant(data []byte, start int) bool {
	if start < 0 || start+13 > len(data) {
		return false
	}
	count := int32(binary.LittleEndian.Uint32(data[start+9 : start+13]))
	if count <= 0 || count > maxScenarioFieldRepeat {
		return false
	}
	_, used, err := DecodeTriggerNode(data[start+13:])
	if err != nil || start+13+used+5 > len(data) {
		return false
	}
	return allZeroBytes(data[start+13+used:], 5)
}

func allZeroBytes(data []byte, n int) bool {
	if len(data) < n {
		return false
	}
	for _, b := range data[:n] {
		if b != 0 {
			return false
		}
	}
	return true
}

// conditionLayoutMarker checks the field that distinguishes the two known
// 1.59 condition layouts. It returns hasConditions=false for scenarios whose
// trigger section contains no conditions, because those records carry no
// evidence with which to select a layout.
func conditionLayoutMarker(root *parsedRoot, want int) (matches, hasConditions bool) {
	if root == nil {
		return false, false
	}
	triggers := root.section("Triggers")
	if triggers == nil {
		return false, false
	}
	for _, trigger := range triggers.list("trigger_data") {
		for _, condition := range trigger.list("condition_data") {
			hasConditions = true
			marker, ok := condition.intValue("static_value_33")
			if !ok || marker != want {
				return false, true
			}
		}
	}
	return true, hasConditions
}

func (r *parsedRoot) triggerInfo() (*TriggerInfo, error) {
	section := r.section("Triggers")
	if section == nil {
		return nil, errors.New("missing Triggers section")
	}
	version, _ := section.float64("trigger_version")
	count, _ := section.intValue("number_of_triggers")
	display := section.uint32List("trigger_display_order_array")
	variableCount, _ := section.intValue("number_of_variables")
	triggerNodes := section.list("trigger_data")
	graph, graphErr := triggerGraphFromTypedNodes(triggerNodes, triggerDataStart(section), triggerDataEnd(section))
	info := &TriggerInfo{
		Version:      version,
		Count:        count,
		DisplayOrder: display,
		Variables:    variableCount,
		RecordStart:  section.Start,
		RecordEnd:    section.End,
	}
	if graphErr == nil {
		info.GraphSHA256 = graph.SHA256
	}
	for i, trigger := range triggerNodes {
		name, _ := trigger.stringValue("trigger_name")
		enabled, _ := trigger.uint32Value("enabled")
		looping, _ := trigger.int8Value("looping")
		executeOnLoad, _ := trigger.intValue("execute_on_load")
		descriptionSTID, _ := trigger.intValue("description_string_table_id")
		displayAsObjective, _ := trigger.intValue("display_as_objective")
		descriptionOrder, _ := trigger.uint32Value("objective_description_order")
		makeHeader, _ := trigger.intValue("make_header")
		shortDescriptionSTID, _ := trigger.intValue("short_description_string_table_id")
		displayOnScreen, _ := trigger.intValue("display_on_screen")
		muteObjectives, _ := trigger.intValue("mute_objectives")
		effects := trigger.list("effect_data")
		conditions := trigger.list("condition_data")
		effectData := make([]EffectSummary, 0, len(effects))
		for j, effect := range effects {
			summary := summarizeEffect(effect, EffectsOptions{})
			summary.TriggerIndex = i
			summary.TriggerName = name
			summary.EffectIndex = j
			effectData = append(effectData, summary)
		}
		conditionData := make([]ConditionSummary, 0, len(conditions))
		for j, condition := range conditions {
			summary := summarizeConditionData(condition)
			summary.TriggerIndex = i
			summary.TriggerName = name
			summary.ConditionIndex = j
			conditionData = append(conditionData, summary)
		}
		info.Triggers = append(info.Triggers, TriggerSummary{
			Index:                         i,
			Name:                          name,
			Enabled:                       enabled,
			Looping:                       looping,
			ExecuteOnLoad:                 executeOnLoad,
			DescriptionStringTableID:      descriptionSTID,
			DisplayAsObjective:            displayAsObjective,
			ObjectiveDescriptionOrder:     descriptionOrder,
			MakeHeader:                    makeHeader,
			ShortDescriptionStringTableID: shortDescriptionSTID,
			DisplayOnScreen:               displayOnScreen,
			MuteObjectives:                muteObjectives,
			Effects:                       len(effects),
			EffectData:                    effectData,
			Conditions:                    len(conditions),
			ConditionData:                 conditionData,
			RecordStart:                   trigger.Start,
			RecordEnd:                     trigger.End,
		})
	}
	info.InvariantOK, info.InvariantNote = validateTriggerInfo(info, triggerNodes)
	if graphErr != nil {
		info.InvariantOK = false
		if info.InvariantNote != "" {
			info.InvariantNote += "; "
		}
		info.InvariantNote += "trigger graph fingerprint failed: " + graphErr.Error()
	}
	return info, nil
}

func triggerDataStart(section *parsedSection) int {
	if field := section.field("trigger_data"); field != nil {
		return field.Start
	}
	return section.Start
}

func triggerDataEnd(section *parsedSection) int {
	if field := section.field("trigger_data"); field != nil {
		return field.End
	}
	return section.End
}

func triggerGraphFromTypedNodes(triggerNodes []*parsedNode, start, end int) (*triggergraph.Graph, error) {
	triggers := make([]map[string]any, 0, len(triggerNodes))
	for _, trigger := range triggerNodes {
		canonical, err := canonicalTrigger(trigger)
		if err != nil {
			return nil, err
		}
		triggers = append(triggers, canonical)
	}
	return triggergraph.FromTriggers(triggers, start, end, nil)
}

func (f *File) TriggerGraph() (*triggergraph.Graph, error) {
	if f.root == nil {
		return nil, errors.New("scenario body not parsed")
	}
	section := f.root.section("Triggers")
	if section == nil {
		return nil, errors.New("missing Triggers section")
	}
	return triggerGraphFromTypedNodes(section.list("trigger_data"), triggerDataStart(section), triggerDataEnd(section))
}

// ParseInflatedTriggers parses an extracted DE scenario Triggers section.
// Replays can embed this section without the scenario file header; keeping
// this entry point separate lets replay readers reuse the versioned scenario
// schema without pretending the surrounding replay header is a scenario file.
func ParseInflatedTriggers(version string, data []byte) (*triggergraph.Graph, int, error) {
	spec, err := LoadDESpecForVersion(version)
	if err != nil {
		return nil, 0, err
	}
	section, ok := spec.section("Triggers")
	if !ok {
		return nil, 0, errors.New("scenario spec has no Triggers section")
	}
	p := parser{data: data, sections: map[string]*parsedSection{}, allowReplaySentinels: true}
	node := &parsedNode{Name: "Triggers", Start: p.off}
	var triggerSpec SectionSpec
	if p.modernTriggerLayout() {
		for _, field := range []struct {
			name string
			kind string
		}{
			{name: "trigger_version", kind: "f64"},
			{name: "trigger_instruction_start", kind: "s8"},
			{name: "number_of_triggers", kind: "s32"},
		} {
			value, readErr := p.readScalarField(field.name, field.kind)
			if readErr != nil {
				return nil, p.off, readErr
			}
			node.Fields = append(node.Fields, value)
		}
		triggerSpec = section.Structs["TriggerStruct"]
	} else {
		for _, field := range section.Fields {
			if field.Name == "trigger_data" {
				triggerSpec = section.Structs["TriggerStruct"]
				break
			}
			field, parseErr := p.parseCompatibilityField(field, section, node)
			if parseErr != nil {
				return nil, p.off, parseErr
			}
			node.Fields = append(node.Fields, field)
		}
	}
	count := asInt(node.field("number_of_triggers").Value)
	triggerField := &parsedNode{Name: "trigger_data", Start: p.off}
	for i := 0; i < count; i++ {
		trigger, parseErr := p.parseNode("TriggerStruct", triggerSpec, triggerField)
		if parseErr != nil {
			return nil, p.off, fmt.Errorf("TriggerStruct[%d]: %w", i, parseErr)
		}
		triggerField.Elements = append(triggerField.Elements, trigger)
		// The 1.59 replay copy carries a five-byte per-trigger trailer that
		// is absent from the editor scenario body. It sits between records.
		if i+1 < count {
			if p.off+5 > len(p.data) {
				return nil, p.off, errors.New("1.59 replay trigger trailer exceeds data")
			}
			p.off += 5
		}
	}
	triggerField.Value = triggerField.Elements
	triggerField.End = p.off
	node.Fields = append(node.Fields, triggerField)
	node.End = p.off
	graph, err := triggerGraphFromTypedNodes(triggerField.Elements, triggerField.Start, triggerField.End)
	if err != nil {
		return nil, p.off, err
	}
	return graph, p.off, nil
}

func canonicalTrigger(trigger *parsedNode) (map[string]any, error) {
	effects := trigger.list("effect_data")
	conditions := trigger.list("condition_data")
	canonicalEffects := make([]map[string]any, 0, len(effects))
	for _, effect := range effects {
		canonical, err := canonicalEffect(effect)
		if err != nil {
			return nil, err
		}
		canonicalEffects = append(canonicalEffects, canonical)
	}
	canonicalConditions := make([]map[string]any, 0, len(conditions))
	for _, condition := range conditions {
		canonical, err := canonicalCondition(condition)
		if err != nil {
			return nil, err
		}
		canonicalConditions = append(canonicalConditions, canonical)
	}
	enabled, _ := trigger.uint32Value("enabled")
	looping, _ := trigger.intValue("looping")
	executeOnLoad, _ := trigger.intValue("execute_on_load")
	descriptionSTID, _ := trigger.intValue("description_string_table_id")
	displayAsObjective, _ := trigger.intValue("display_as_objective")
	descriptionOrder, _ := trigger.uint32Value("objective_description_order")
	makeHeader, _ := trigger.intValue("make_header")
	shortDescriptionSTID, _ := trigger.intValue("short_description_string_table_id")
	displayOnScreen, _ := trigger.intValue("display_on_screen")
	muteObjectives, _ := trigger.intValue("mute_objectives")
	description, _ := trigger.stringValue("trigger_description")
	name, _ := trigger.stringValue("trigger_name")
	shortDescription, _ := trigger.stringValue("short_description")
	return map[string]any{
		"conditions":             canonicalConditions,
		"condition_order":        trigger.intList("condition_display_order_array"),
		"description":            description,
		"description_order":      int(descriptionOrder),
		"description_stid":       descriptionSTID,
		"display_as_objective":   displayAsObjective,
		"display_on_screen":      displayOnScreen,
		"effect_order":           trigger.intList("effect_display_order_array"),
		"effects":                canonicalEffects,
		"enabled":                int(enabled),
		"execute_on_load":        executeOnLoad,
		"looping":                looping,
		"make_header":            makeHeader,
		"mute_objectives":        muteObjectives,
		"name":                   name,
		"short_description":      shortDescription,
		"short_description_stid": shortDescriptionSTID,
	}, nil
}

func canonicalEffect(effect *parsedNode) (map[string]any, error) {
	raw := effect.raw()
	if len(raw) < 84*4 {
		return nil, fmt.Errorf("effect at %d has %d bytes, need at least %d", effect.Start, len(raw), 84*4)
	}
	values := make([]int, 84)
	for i := range values {
		values[i] = int(int32(binary.LittleEndian.Uint32(raw[i*4:])))
	}
	if values[1] != 83 && values[1] != 81 {
		return nil, fmt.Errorf("bad effect static value at %d: %d", effect.Start, values[1])
	}
	message, _ := effect.stringValue("message")
	selectedObjectIDs := effect.intList("selected_object_ids")
	sum := sha256.Sum256(raw)
	out := map[string]any{
		"fields_prefix": values,
		"message":       message,
		"raw_sha256":    hex.EncodeToString(sum[:]),
		"type":          values[0],
	}
	if values[6] >= 0 {
		out["selected_object_ids"] = selectedObjectIDs
	}
	return out, nil
}

func canonicalCondition(condition *parsedNode) (map[string]any, error) {
	raw := condition.raw()
	if len(raw) < 35*4 {
		return nil, fmt.Errorf("condition at %d has %d bytes, need at least %d", condition.Start, len(raw), 35*4)
	}
	values := make([]int, 35)
	for i := range values {
		values[i] = int(int32(binary.LittleEndian.Uint32(raw[i*4:])))
	}
	// DE 1.59-era replay embeddings use the next condition-record static
	// marker (34) while scenario files still carry the established 33 marker.
	if values[1] != 33 && values[1] != 34 {
		return nil, fmt.Errorf("bad condition static value at %d: %d", condition.Start, values[1])
	}
	xsFunction, _ := condition.stringValue("xs_function")
	return map[string]any{
		"fields":      values,
		"type":        values[0],
		"xs_function": xsFunction,
	}, nil
}

func (r *parsedRoot) unitInfo() (*UnitInfo, error) {
	section := r.section("Units")
	if section == nil {
		return nil, errors.New("missing Units section")
	}
	numberOfUnitSections, _ := section.intValue("number_of_unit_sections")
	numberOfPlayers, _ := section.intValue("number_of_players")
	playerSections := section.list("players_units")
	info := &UnitInfo{
		NumberOfUnitSections: numberOfUnitSections,
		NumberOfPlayers:      numberOfPlayers,
	}
	for i, playerSection := range playerSections {
		count, _ := playerSection.intValue("unit_count")
		unitNodes := playerSection.list("units")
		playerInfo := PlayerUnitsInfo{Player: i, Count: count}
		for j, unit := range unitNodes {
			refID, _ := unit.intValue("reference_id")
			unitConst, _ := unit.intValue("unit_const")
			status, _ := unit.intValue("status")
			captionStringID, _ := unit.intValue("caption_string_id")
			captionString, _ := unit.stringValue("caption_string")
			x, _ := unit.floatValue("x")
			y, _ := unit.floatValue("y")
			z, _ := unit.floatValue("z")
			rotation, _ := unit.floatValue("rotation")
			playerInfo.Units = append(playerInfo.Units, UnitSummary{
				Index:           j,
				ReferenceID:     refID,
				UnitConst:       unitConst,
				X:               x,
				Y:               y,
				Z:               z,
				Rotation:        rotation,
				Status:          status,
				CaptionStringID: captionStringID,
				CaptionString:   captionString,
			})
		}
		info.Sections = append(info.Sections, playerInfo)
		info.Total += len(unitNodes)
	}
	return info, nil
}

func (r *parsedRoot) mapInfo() (*MapInfo, error) {
	section := r.section("Map")
	if section == nil {
		return nil, errors.New("missing Map section")
	}
	width, _ := section.intValue("map_width")
	height, _ := section.intValue("map_height")
	tiles := section.list("terrain_data")
	info := &MapInfo{
		Width:         width,
		Height:        height,
		TileCount:     len(tiles),
		TerrainCounts: map[int]int{},
	}
	if modes, ok := section.intValue("secondary_game_modes"); ok {
		info.SecondaryGameModes = SecondaryGameModesInfo{
			Bits:  modes,
			Names: secondaryGameModeNames(modes),
		}
	}
	for _, tile := range tiles {
		terrainID, _ := tile.intValue("terrain_id")
		info.TerrainCounts[terrainID]++
	}
	return info, nil
}

func (r *parsedRoot) playerInfo() []PlayerInfo {
	dataHeader := r.section("DataHeader")
	playerDataTwo := r.section("PlayerDataTwo")
	options := r.section("Options")
	mapSection := r.section("Map")
	if dataHeader == nil {
		return nil
	}
	tribeNames := dataHeader.stringList("tribe_names")
	nameStringIDs := dataHeader.intList("string_table_player_names")
	lockCiv := dataHeader.intList("per_player_lock_civilization")
	lockPersonality := dataHeader.intList("per_player_lock_personality")
	var startingAges []int
	var basePriorities []int
	if options != nil {
		startingAges = options.intList("per_player_starting_age")
		basePriorities = options.intList("per_player_base_priority")
	}
	populationLimits := populationLimitList(mapSection)
	playerData := dataHeader.list("player_data_1")
	var aiNames []string
	var aiTypes []int
	var playerColors []int
	if playerDataTwo != nil {
		aiNames = playerDataTwo.stringList("ai_names")
		aiTypes = playerDataTwo.intList("ai_type")
		for _, resource := range playerDataTwo.list("resources") {
			color, _ := resource.intValue("player_color")
			playerColors = append(playerColors, color)
		}
	}
	out := make([]PlayerInfo, 0, len(playerData))
	for i, player := range playerData {
		active, _ := player.intValue("active")
		human, _ := player.intValue("human")
		civ, _ := player.stringValue("civilization")
		architecture, _ := player.stringValue("architecture_set")
		info := PlayerInfo{
			Player:           i,
			PlayerLabel:      playerLabel(i),
			SectionIndex:     i,
			IndexBase:        "players_from_1",
			FieldReferences:  playerFieldReferences(i),
			Active:           active != 0,
			Human:            human != 0,
			TribeName:        stringAt(tribeNames, i),
			NameStringID:     intAt(nameStringIDs, i),
			Civilization:     civ,
			Architecture:     architecture,
			LockCivilization: intAt(lockCiv, i) != 0,
			LockPersonality:  intAt(lockPersonality, i) != 0,
			StartingAge:      intAt(startingAges, i),
			StartingAgeName:  startingAgeName(intAt(startingAges, i)),
			Color:            intAt(playerColors, i),
			ColorName:        playerColorName(intAt(playerColors, i)),
			BasePriority:     intAt(basePriorities, i),
			PopulationLimit:  intAt(populationLimits, i),
			AIName:           stringAt(aiNames, i),
			AIType:           intAt(aiTypes, i),
		}
		// The editor writes the first AI name in the zero-based player stream,
		// while the following entry is in the Gaia-at-zero stream. Both entries
		// can therefore describe P1 in one saved scenario; retain both raw refs.
		info.FieldReferences["ai_name"] = aiNameReference(i)
		out = append(out, info)
	}
	return out
}

func playerLabel(sectionIndex int) string {
	return fmt.Sprintf("P%d", sectionIndex+1)
}

func playerFieldReferences(sectionIndex int) map[string]PlayerReference {
	ref := PlayerReference{Player: playerLabel(sectionIndex), SectionIndex: sectionIndex, IndexBase: "players_from_1"}
	refs := map[string]PlayerReference{}
	for _, field := range []string{
		"active", "human", "tribe_name", "name_string_id", "civilization", "architecture",
		"lock_civilization", "lock_personality", "starting_age", "color", "base_priority",
		"population_limit", "ai_type", "resources.gold", "resources.wood", "resources.food",
		"resources.stone", "resources.trade_goods", "allied_victory", "diplomacy",
		"disabled_techs", "disabled_units", "disabled_buildings",
	} {
		refs[field] = ref
	}
	return refs
}

func aiNameReference(sectionIndex int) PlayerReference {
	if sectionIndex == 0 {
		return PlayerReference{Player: "P1", SectionIndex: 0, IndexBase: "players_from_1"}
	}
	if sectionIndex == 1 {
		return PlayerReference{Player: "P1", SectionIndex: 1, IndexBase: "gaia_at_0"}
	}
	return PlayerReference{Player: fmt.Sprintf("P%d", sectionIndex), SectionIndex: sectionIndex, IndexBase: "gaia_at_0"}
}

func (r *parsedRoot) aiInfo() []AIFileInfo {
	files := r.section("Files")
	if files == nil {
		return nil
	}
	aiFiles := files.list("ai_files")
	out := make([]AIFileInfo, 0, len(aiFiles))
	for i, ai := range aiFiles {
		name, _ := ai.stringValue("ai_file_name")
		content, _ := ai.stringValue("ai_file")
		sum := sha256.Sum256([]byte(content))
		info := AIFileInfo{Index: i, Name: name, ContentBytes: len(content)}
		if content != "" {
			info.ContentSHA256 = hex.EncodeToString(sum[:])
		}
		out = append(out, info)
	}
	return out
}

func validateTriggerInfo(info *TriggerInfo, triggers []*parsedNode) (bool, string) {
	if info.Count != len(triggers) {
		return false, fmt.Sprintf("number_of_triggers=%d but trigger_data has %d", info.Count, len(triggers))
	}
	if info.Count != len(info.DisplayOrder) {
		return false, fmt.Sprintf("number_of_triggers=%d but display_order has %d", info.Count, len(info.DisplayOrder))
	}
	seen := map[uint32]bool{}
	for _, id := range info.DisplayOrder {
		if int(id) >= info.Count {
			return false, fmt.Sprintf("display_order contains out-of-range trigger id %d", id)
		}
		if seen[id] {
			return false, fmt.Sprintf("display_order contains duplicate trigger id %d", id)
		}
		seen[id] = true
	}
	for i, trigger := range triggers {
		effectCount, _ := trigger.intValue("number_of_effects")
		effects := trigger.list("effect_data")
		if effectCount != len(effects) {
			return false, fmt.Sprintf("trigger %d number_of_effects=%d but effect_data has %d", i, effectCount, len(effects))
		}
		effectOrder := trigger.intList("effect_display_order_array")
		if ok, note := validateOptionalOrder(effectOrder, len(effects), "effect", i); !ok {
			return false, note
		}
		conditionCount, _ := trigger.intValue("number_of_conditions")
		conditions := trigger.list("condition_data")
		if conditionCount != len(conditions) {
			return false, fmt.Sprintf("trigger %d number_of_conditions=%d but condition_data has %d", i, conditionCount, len(conditions))
		}
		conditionOrder := trigger.intList("condition_display_order_array")
		if ok, note := validateOptionalOrder(conditionOrder, len(conditions), "condition", i); !ok {
			return false, note
		}
		for j, effect := range effects {
			selectedCount, _ := effect.intValue("number_of_units_selected")
			selected := effect.intList("selected_object_ids")
			if selectedCount == -1 && len(selected) == 0 {
				continue
			}
			if selectedCount != len(selected) {
				return false, fmt.Sprintf("trigger %d effect %d selected count=%d but ids has %d", i, j, selectedCount, len(selected))
			}
		}
	}
	return true, ""
}

func validateOptionalOrder(order []int, count int, label string, triggerIndex int) (bool, string) {
	if len(order) == 0 {
		return true, ""
	}
	if len(order) != count {
		return false, fmt.Sprintf("trigger %d %s display order has %d entries for %d %ss", triggerIndex, label, len(order), count, label)
	}
	seen := map[int]bool{}
	for _, id := range order {
		if id < 0 || id >= count {
			return false, fmt.Sprintf("trigger %d %s display order contains out-of-range id %d", triggerIndex, label, id)
		}
		if seen[id] {
			return false, fmt.Sprintf("trigger %d %s display order contains duplicate id %d", triggerIndex, label, id)
		}
		seen[id] = true
	}
	return true, ""
}

type parsedSection = parsedNode

type parsedNode struct {
	Name     string
	Start    int
	End      int
	Fields   []*parsedNode
	Elements []*parsedNode
	Raw      []byte
	Value    any
}

func (n *parsedNode) raw() []byte {
	if len(n.Raw) != 0 {
		return append([]byte(nil), n.Raw...)
	}
	var out []byte
	for _, field := range n.Fields {
		out = append(out, field.raw()...)
	}
	for _, elem := range n.Elements {
		out = append(out, elem.raw()...)
	}
	return out
}

func (n *parsedNode) field(name string) *parsedNode {
	for _, field := range n.Fields {
		if field.Name == name {
			return field
		}
	}
	return nil
}

func (n *parsedNode) list(name string) []*parsedNode {
	field := n.field(name)
	if field == nil {
		return nil
	}
	return field.Elements
}

func (n *parsedNode) stringValue(name string) (string, bool) {
	field := n.field(name)
	if field == nil {
		return "", false
	}
	value, ok := field.Value.(string)
	return value, ok
}

func (n *parsedNode) stringList(name string) []string {
	field := n.field(name)
	if field == nil {
		return nil
	}
	values, ok := field.Value.([]any)
	if !ok {
		if one, ok := field.Value.(string); ok {
			return []string{one}
		}
		return nil
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if v, ok := value.(string); ok {
			out = append(out, v)
		}
	}
	return out
}

func (n *parsedNode) float64(name string) (float64, bool) {
	field := n.field(name)
	if field == nil {
		return 0, false
	}
	value, ok := field.Value.(float64)
	return value, ok
}

func (n *parsedNode) floatValue(name string) (float64, bool) {
	field := n.field(name)
	if field == nil {
		return 0, false
	}
	switch value := field.Value.(type) {
	case float32:
		return float64(value), true
	case float64:
		return value, true
	}
	return 0, false
}

func (n *parsedNode) uint32Value(name string) (uint32, bool) {
	field := n.field(name)
	if field == nil {
		return 0, false
	}
	value, ok := field.Value.(uint32)
	return value, ok
}

func (n *parsedNode) int8Value(name string) (int8, bool) {
	field := n.field(name)
	if field == nil {
		return 0, false
	}
	value, ok := field.Value.(int8)
	return value, ok
}

func (n *parsedNode) intValue(name string) (int, bool) {
	field := n.field(name)
	if field == nil {
		return 0, false
	}
	switch value := field.Value.(type) {
	case int:
		return value, true
	case int8:
		return int(value), true
	case int16:
		return int(value), true
	case int32:
		return int(value), true
	case uint8:
		return int(value), true
	case uint16:
		return int(value), true
	case uint32:
		return int(value), true
	}
	return 0, false
}

func (n *parsedNode) intList(name string) []int {
	field := n.field(name)
	if field == nil {
		return nil
	}
	values, ok := field.Value.([]any)
	if !ok {
		if one, ok := numericListItem(field.Value); ok {
			return []int{one}
		}
		return nil
	}
	out := make([]int, 0, len(values))
	for _, value := range values {
		if v, ok := numericListItem(value); ok {
			out = append(out, v)
		}
	}
	return out
}

func numericListItem(value any) (int, bool) {
	switch v := value.(type) {
	case int:
		return v, true
	case int8:
		return int(v), true
	case int16:
		return int(v), true
	case int32:
		return int(v), true
	case uint8:
		return int(v), true
	case uint16:
		return int(v), true
	case uint32:
		return int(v), true
	}
	return 0, false
}

func sectionSelection(sections []string) map[string]bool {
	if len(sections) == 0 {
		return nil
	}
	selected := map[string]bool{}
	for _, section := range sections {
		for _, name := range strings.Split(section, ",") {
			name = strings.TrimSpace(strings.ToLower(name))
			if name == "" || name == "all" {
				return nil
			}
			for _, canonical := range sectionAliases(name) {
				selected[canonical] = true
			}
		}
	}
	return selected
}

func sectionAliases(name string) []string {
	switch name {
	case "triggers", "trigger":
		return []string{"Triggers"}
	case "units", "unit":
		return []string{"Units"}
	case "map", "terrain":
		return []string{"Map"}
	case "players", "player", "resources":
		return []string{"DataHeader", "PlayerDataTwo", "Diplomacy"}
	case "victory":
		return []string{"GlobalVictory"}
	case "messages", "message":
		return []string{"Messages", "Cinematics"}
	case "disables", "disabled", "options":
		return []string{"Options"}
	case "ai", "files":
		return []string{"Files", "PlayerDataTwo"}
	default:
		return []string{name}
	}
}

func dumpNodes(nodes []*parsedNode) []NodeDump {
	out := make([]NodeDump, 0, len(nodes))
	for _, node := range nodes {
		out = append(out, dumpNode(node))
	}
	return out
}

func dumpNode(node *parsedNode) NodeDump {
	out := NodeDump{
		Name:  node.Name,
		Start: node.Start,
		End:   node.End,
	}
	if node.Value != nil && len(node.Elements) == 0 {
		out.Value = dumpValue(node.Value)
	}
	if len(node.Fields) > 0 {
		out.Fields = dumpNodes(node.Fields)
	}
	if len(node.Elements) > 0 {
		out.Elements = dumpNodes(node.Elements)
	}
	return out
}

func dumpValue(value any) any {
	switch v := value.(type) {
	case []byte:
		return hex.EncodeToString(v)
	case []any:
		out := make([]any, 0, len(v))
		for _, item := range v {
			out = append(out, dumpValue(item))
		}
		return out
	case float32:
		if math.IsNaN(float64(v)) {
			return "NaN"
		}
		if math.IsInf(float64(v), 1) {
			return "+Inf"
		}
		if math.IsInf(float64(v), -1) {
			return "-Inf"
		}
		return v
	case float64:
		if math.IsNaN(v) {
			return "NaN"
		}
		if math.IsInf(v, 1) {
			return "+Inf"
		}
		if math.IsInf(v, -1) {
			return "-Inf"
		}
		return v
	default:
		return v
	}
}

func stringAt(values []string, index int) string {
	if index < 0 || index >= len(values) {
		return ""
	}
	return values[index]
}

func intAt(values []int, index int) int {
	if index < 0 || index >= len(values) {
		return 0
	}
	return values[index]
}

func populationLimitList(mapSection *parsedNode) []int {
	if mapSection == nil {
		return nil
	}
	field := mapSection.field("per_player_population_cap")
	if field == nil {
		return nil
	}
	if values, ok := field.Value.([]any); ok && len(values) > 0 {
		out := make([]int, 0, len(values))
		for _, value := range values {
			out = append(out, asInt(value))
		}
		return out
	}
	sectionRaw := mapSection.raw()
	rel := field.Start - mapSection.Start - 3
	if rel >= 0 && rel+16*4 <= len(sectionRaw) {
		out := make([]int, 0, 16)
		for i := 0; i < 16; i++ {
			out = append(out, int(sectionRaw[rel+i*4+3]))
		}
		return out
	}
	if len(field.Raw)%4 != 0 || len(field.Raw) == 0 {
		return nil
	}
	out := make([]int, 0, len(field.Raw)/4)
	for i := 0; i+4 <= len(field.Raw); i += 4 {
		out = append(out, int(field.Raw[i+3]))
	}
	return out
}

func startingAgeName(raw int) string {
	switch raw {
	case 0, 2:
		return "Dark"
	case 3:
		return "Feudal"
	case 4:
		return "Castle"
	case 5:
		return "Imperial"
	case 6:
		return "Post-Imperial"
	default:
		return ""
	}
}

func playerColorName(raw int) string {
	switch raw {
	case 0:
		return "Blue"
	case 1:
		return "Red"
	case 2:
		return "Green"
	case 3:
		return "Yellow"
	case 4:
		return "Cyan"
	case 5:
		return "Purple"
	case 6:
		return "Gray"
	case 7:
		return "Orange"
	default:
		return ""
	}
}

func (n *parsedNode) uint32List(name string) []uint32 {
	field := n.field(name)
	if field == nil {
		return nil
	}
	values, ok := field.Value.([]any)
	if !ok {
		if one, ok := field.Value.(uint32); ok {
			return []uint32{one}
		}
		return nil
	}
	out := make([]uint32, 0, len(values))
	for _, value := range values {
		if v, ok := value.(uint32); ok {
			out = append(out, v)
		}
	}
	return out
}

type parser struct {
	data                 []byte
	off                  int
	sections             map[string]*parsedSection
	stopBefore           string
	allowReplaySentinels bool
	strictOwned          bool
	version              string
}

func (p *parser) modernTriggerLayout() bool {
	return p.version == "" || p.version == "1.58" || p.version == "1.59"
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func (p *parser) parse(spec *Spec) (*parsedRoot, error) {
	root := &parsedRoot{}
	for _, sectionSpec := range spec.Sections {
		if sectionSpec.Name == "FileHeader" {
			continue
		}
		if p.stopBefore != "" && sectionSpec.Name == p.stopBefore {
			break
		}
		section, err := p.parseNode(sectionSpec.Name, sectionSpec.Spec, nil)
		if err != nil {
			return nil, fmt.Errorf("section %s: %w", sectionSpec.Name, err)
		}
		root.Sections = append(root.Sections, section)
		p.sections[section.Name] = section
	}
	return root, nil
}

func (p *parser) parseNode(name string, spec SectionSpec, parent *parsedNode) (*parsedNode, error) {
	if name == "Units" {
		return p.parseUnitsNode(spec)
	}
	if name == "Triggers" {
		return p.parseTriggersNode(spec)
	}
	if name == "UnitStruct" {
		has159Byte := false
		for _, field := range spec.Fields {
			if field.Name == "unknown_1_59_before_caption_string_id" {
				has159Byte = true
				break
			}
		}
		node, consumed, err := DecodeUnitStructNode(p.data[p.off:], has159Byte)
		if err != nil {
			return nil, err
		}
		node.Start = p.off
		node.End = p.off + consumed
		p.off += consumed
		return node, nil
	}
	if name == "PlayerDataFourStruct" {
		node, consumed, err := DecodePlayerDataFourNode(p.data[p.off:])
		if err != nil {
			return nil, err
		}
		node.Start = p.off
		node.End = p.off + consumed
		p.off += consumed
		return node, nil
	}
	if name == "PlayerDataThreeStruct" {
		node, consumed, err := DecodePlayerDataThreeNode(p.data[p.off:])
		if err != nil {
			return nil, err
		}
		node.Start = p.off
		node.End = p.off + consumed
		p.off += consumed
		return node, nil
	}
	if name == "PlayerUnitsStruct" {
		unitSpec := spec.Structs["UnitStruct"]
		has159Byte := false
		for _, field := range unitSpec.Fields {
			if field.Name == "unknown_1_59_before_caption_string_id" {
				has159Byte = true
				break
			}
		}
		node, consumed, err := DecodePlayerUnitsNode(p.data[p.off:], has159Byte)
		if err != nil {
			return nil, err
		}
		node.Start = p.off
		node.End = p.off + consumed
		p.off += consumed
		return node, nil
	}
	if name == "ConditionStruct" && p.modernTriggerLayout() {
		node, consumed, err := DecodeConditionNode(p.data[p.off:])
		if err != nil {
			return nil, err
		}
		node.Start = p.off
		node.End = p.off + consumed
		p.off += consumed
		return node, nil
	}
	if name == "EffectStruct" && p.modernTriggerLayout() {
		node, consumed, err := DecodeEffectNode(p.data[p.off:])
		if err != nil {
			return nil, err
		}
		node.Start = p.off
		node.End = p.off + consumed
		p.off += consumed
		return node, nil
	}
	if name == "VariableStruct" {
		node, consumed, err := DecodeVariableNode(p.data[p.off:])
		if err != nil {
			return nil, err
		}
		node.Start = p.off
		node.End = p.off + consumed
		p.off += consumed
		return node, nil
	}
	if name == "TriggerStruct" && !p.allowReplaySentinels && p.modernTriggerLayout() {
		decoded, consumed, err := DecodeTriggerNode(p.data[p.off:])
		if err == nil {
			decoded.Start = p.off
			decoded.End = p.off + consumed
			p.off += consumed
			return decoded, nil
		}
		return nil, err
	}
	if name == "DataHeader" {
		_, node, consumed, err := DecodeDataHeaderNode(p.data[p.off:])
		if err != nil {
			return nil, err
		}
		node.Start = p.off
		node.End = p.off + consumed
		p.off += consumed
		return node, nil
	}
	if name == "Map" {
		_, node, consumed, err := DecodeMapLayoutNode(p.data[p.off:])
		if err != nil {
			return nil, err
		}
		node.Start = p.off
		node.End = p.off + consumed
		p.off += consumed
		return node, nil
	}
	if name == "Diplomacy" {
		_, node, consumed, err := DecodeDiplomacyNode(p.data[p.off:])
		if err != nil {
			return nil, err
		}
		node.Start = p.off
		node.End = p.off + consumed
		p.off += consumed
		return node, nil
	}
	if name == "GlobalVictory" {
		_, node, consumed, err := DecodeGlobalVictoryNode(p.data[p.off:])
		if err != nil {
			return nil, err
		}
		node.Start = p.off
		node.End = p.off + consumed
		p.off += consumed
		return node, nil
	}
	if name == "Messages" {
		node, consumed, err := DecodeMessagesNode(p.data[p.off:])
		if err != nil {
			return nil, err
		}
		node.Start = p.off
		node.End = p.off + consumed
		p.off += consumed
		return node, nil
	}
	if name == "Cinematics" {
		node, consumed, err := DecodeCinematicsNode(p.data[p.off:])
		if err != nil {
			return nil, err
		}
		node.Start = p.off
		node.End = p.off + consumed
		p.off += consumed
		return node, nil
	}
	if name == "Options" {
		node, consumed, err := DecodeOptionsNode(p.data[p.off:])
		if err != nil {
			return nil, err
		}
		node.Start = p.off
		node.End = p.off + consumed
		p.off += consumed
		return node, nil
	}
	if name == "BackgroundImage" {
		node, consumed, err := DecodeBackgroundImageNode(p.data[p.off:])
		if err != nil {
			return nil, err
		}
		node.Start = p.off
		node.End = p.off + consumed
		p.off += consumed
		return node, nil
	}
	if name == "PlayerDataTwo" {
		node, consumed, err := DecodePlayerDataTwoNode(p.data[p.off:])
		if err != nil {
			return nil, err
		}
		node.Start = p.off
		node.End = p.off + consumed
		p.off += consumed
		return node, nil
	}
	if name == "Files" {
		node, consumed, err := DecodeFilesNode(p.data[p.off:])
		if err != nil {
			return nil, err
		}
		node.Start = p.off
		node.End = p.off + consumed
		p.off += consumed
		return node, nil
	}
	if name == "TriggerStruct" && p.allowReplaySentinels {
		// Replay trigger copies retain the legacy per-record framing. Keep this
		// narrow compatibility path isolated from editor scenarios, whose
		// current records use DecodeTriggerNode above.
		start := p.off
		node, err := p.parseCompatibilityNode(name, spec, parent)
		if err == nil {
			return node, nil
		}
		compact, ok := withoutField(spec, "unknown")
		if !ok {
			p.off = start
			return nil, err
		}
		p.off = start
		return p.parseCompatibilityNode(name, compact, parent)
	}
	if p.strictOwned && p.modernTriggerLayout() {
		return nil, fmt.Errorf("modern scenario node %q has no owned codec", name)
	}
	// Keep the descriptor walker available for caller-supplied structural
	// probes. All production scenario nodes above use owned codecs; this path
	// is compatibility machinery, not a parser dependency for those codecs.
	return p.parseCompatibilityNode(name, spec, parent)
}

// parseUnitsNode owns the section framing and delegates only the still-dark
// nested records to their versioned field specs. UnitStruct itself is decoded
// by the typed codec above, so this boundary no longer depends on the generic
// struct walker for the unit payload.
func (p *parser) parseUnitsNode(spec SectionSpec) (*parsedNode, error) {
	node := &parsedNode{Name: "Units", Start: p.off}
	unitSections, err := p.readScalarField("number_of_unit_sections", "u32")
	if err != nil {
		return nil, err
	}
	node.Fields = append(node.Fields, unitSections)

	playerData4 := &parsedNode{Name: "player_data_4", Start: p.off}
	for i := 0; i < 8; i++ {
		start := p.off
		child, consumed, err := DecodePlayerDataFourNode(p.data[p.off:])
		if err != nil {
			return nil, fmt.Errorf("player_data_4[%d]: %w", i, err)
		}
		child.Start = start
		child.End = start + consumed
		p.off += consumed
		playerData4.Elements = append(playerData4.Elements, child)
	}
	playerData4.Value = playerData4.Elements
	playerData4.End = p.off
	node.Fields = append(node.Fields, playerData4)

	players, err := p.readScalarField("number_of_players", "u32")
	if err != nil {
		return nil, err
	}
	node.Fields = append(node.Fields, players)

	playerData3 := &parsedNode{Name: "player_data_3", Start: p.off}
	for i := 0; i < 8; i++ {
		start := p.off
		child, consumed, err := DecodePlayerDataThreeNode(p.data[p.off:])
		if err != nil {
			return nil, fmt.Errorf("player_data_3[%d]: %w", i, err)
		}
		child.Start = start
		child.End = start + consumed
		p.off += consumed
		playerData3.Elements = append(playerData3.Elements, child)
	}
	playerData3.Value = playerData3.Elements
	playerData3.End = p.off
	node.Fields = append(node.Fields, playerData3)

	count := asInt(unitSections.Value)
	if count < 0 || count > maxScenarioFieldRepeat {
		return nil, fmt.Errorf("players_units repeat %d exceeds parser safety limit %d", count, maxScenarioFieldRepeat)
	}
	playersUnits := &parsedNode{Name: "players_units", Start: p.off}
	for i := 0; i < count; i++ {
		start := p.off
		child, consumed, err := DecodePlayerUnitsNode(p.data[p.off:], p.unitStructHas159Byte(spec))
		if err != nil {
			return nil, fmt.Errorf("players_units[%d]: %w", i, err)
		}
		child.Start = start
		child.End = start + consumed
		p.off += consumed
		playersUnits.Elements = append(playersUnits.Elements, child)
	}
	playersUnits.Value = playersUnits.Elements
	playersUnits.End = p.off
	node.Fields = append(node.Fields, playersUnits)
	node.End = p.off
	return node, nil
}

func (p *parser) unitStructHas159Byte(spec SectionSpec) bool {
	unitSpec := spec.Structs["PlayerUnitsStruct"].Structs["UnitStruct"]
	for _, field := range unitSpec.Fields {
		if field.Name == "unknown_1_59_before_caption_string_id" {
			return true
		}
	}
	return false
}

func (p *parser) readScalarField(name, kind string) (*parsedNode, error) {
	start := p.off
	value, raw, err := p.readPrimitive(kind)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	return &parsedNode{Name: name, Start: start, End: p.off, Value: value, Raw: raw}, nil
}

// parseTriggersNode owns trigger-section framing. The count-bearing header is
// parsed before any variable-length trigger records, which keeps the section
// boundary explicit for the later typed condition/effect codecs.
func (p *parser) parseTriggersNode(spec SectionSpec) (*parsedNode, error) {
	if !p.modernTriggerLayout() {
		return p.parseCompatibilityNode("Triggers", spec, nil)
	}
	node := &parsedNode{Name: "Triggers", Start: p.off}
	for _, field := range []struct {
		name string
		kind string
	}{
		{name: "trigger_version", kind: "f64"},
		{name: "trigger_instruction_start", kind: "s8"},
		{name: "number_of_triggers", kind: "s32"},
	} {
		value, err := p.readScalarField(field.name, field.kind)
		if err != nil {
			return nil, err
		}
		node.Fields = append(node.Fields, value)
	}

	triggerCount := asInt(node.field("number_of_triggers").Value)
	if err := validateScenarioRepeat("trigger_data", triggerCount); err != nil {
		return nil, err
	}
	triggerField := &parsedNode{Name: "trigger_data", Start: p.off}
	for i := 0; i < triggerCount; i++ {
		start := p.off
		child, consumed, err := DecodeTriggerNode(p.data[p.off:])
		if err != nil {
			return nil, fmt.Errorf("trigger_data[%d]: %w", i, err)
		}
		child.Start = start
		child.End = start + consumed
		p.off += consumed
		triggerField.Elements = append(triggerField.Elements, child)
	}
	triggerField.Value = triggerField.Elements
	triggerField.End = p.off
	node.Fields = append(node.Fields, triggerField)

	order, err := p.readRepeatedField("trigger_display_order_array", "u32", triggerCount)
	if err != nil {
		return nil, err
	}
	node.Fields = append(node.Fields, order)

	reserved, err := p.readRepeatedField("unknown_bytes", "1028", 1)
	if err != nil {
		return nil, err
	}
	node.Fields = append(node.Fields, reserved)

	variables, err := p.readScalarField("number_of_variables", "u32")
	if err != nil {
		return nil, err
	}
	node.Fields = append(node.Fields, variables)
	variableCount := asInt(variables.Value)
	if p.allowReplaySentinels {
		if value, ok := variables.Value.(uint32); ok && value == ^uint32(0) {
			variableCount = 0
		}
	}
	if err := validateScenarioRepeat("variable_data", variableCount); err != nil {
		return nil, err
	}
	variableField := &parsedNode{Name: "variable_data", Start: p.off}
	for i := 0; i < variableCount; i++ {
		start := p.off
		child, consumed, err := DecodeVariableNode(p.data[p.off:])
		if err != nil {
			return nil, fmt.Errorf("variable_data[%d]: %w", i, err)
		}
		child.Start = start
		child.End = start + consumed
		p.off += consumed
		variableField.Elements = append(variableField.Elements, child)
	}
	variableField.Value = variableField.Elements
	variableField.End = p.off
	node.Fields = append(node.Fields, variableField)

	for _, field := range []struct {
		name   string
		kind   string
		repeat int
	}{
		{name: "useless_trigger_data", kind: "9", repeat: 1},
		{name: "unknown_bytes2", kind: "8", repeat: 0},
		{name: "redacted", kind: "16", repeat: 1},
		{name: "legacy_exec_order", kind: "u8", repeat: 0},
	} {
		repeat := field.repeat
		if field.name == "unknown_bytes2" && nodeFloat(node, "trigger_version") >= 3.5 {
			repeat = 1
		}
		if field.name == "legacy_exec_order" && nodeFloat(node, "trigger_version") >= 4.5 {
			repeat = 1
		}
		if repeat == 0 {
			continue
		}
		value, err := p.readRepeatedField(field.name, field.kind, repeat)
		if err != nil {
			return nil, err
		}
		node.Fields = append(node.Fields, value)
	}
	node.End = p.off
	return node, nil
}

func validateScenarioRepeat(name string, repeat int) error {
	if repeat < 0 || repeat > maxScenarioFieldRepeat {
		return fmt.Errorf("%s repeat %d exceeds parser safety limit %d", name, repeat, maxScenarioFieldRepeat)
	}
	return nil
}

func (p *parser) readRepeatedField(name, kind string, repeat int) (*parsedNode, error) {
	if err := validateScenarioRepeat(name, repeat); err != nil {
		return nil, err
	}
	start := p.off
	values := make([]any, 0, repeat)
	var raw []byte
	for i := 0; i < repeat; i++ {
		value, chunk, err := p.readPrimitive(kind)
		if err != nil {
			return nil, fmt.Errorf("%s item %d: %w", name, i, err)
		}
		values = append(values, value)
		raw = append(raw, chunk...)
	}
	var value any = values
	if repeat == 1 {
		value = values[0]
	}
	return &parsedNode{Name: name, Start: start, End: p.off, Value: value, Raw: raw}, nil
}

func nodeFloat(node *parsedNode, name string) float64 {
	if field := node.field(name); field != nil {
		return asFloat(field.Value)
	}
	return 0
}

// parseCompatibilityNode is retained for legacy scenario/replay layouts and
// caller-supplied structural probes. Modern scenario nodes must use owned
// codecs; strictOwned rejects accidental fallback into this path.
func (p *parser) parseCompatibilityNode(name string, spec SectionSpec, parent *parsedNode) (*parsedNode, error) {
	node := &parsedNode{Name: name, Start: p.off}
	for _, fieldSpec := range spec.Fields {
		field, err := p.parseCompatibilityField(fieldSpec, spec, node)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", fieldSpec.Name, err)
		}
		node.Fields = append(node.Fields, field)
	}
	node.End = p.off
	return node, nil
}

func withoutField(spec SectionSpec, name string) (SectionSpec, bool) {
	filtered := make([]FieldSpec, 0, len(spec.Fields))
	removed := false
	for _, field := range spec.Fields {
		if field.Name == name {
			removed = true
			continue
		}
		filtered = append(filtered, field)
	}
	if !removed {
		return SectionSpec{}, false
	}
	spec.Fields = filtered
	return spec, true
}

func (p *parser) parseCompatibilityField(fieldSpec FieldSpec, sectionSpec SectionSpec, node *parsedNode) (*parsedNode, error) {
	repeat, err := p.constructRepeat(fieldSpec, node)
	if err != nil {
		return nil, err
	}
	if fieldSpec.Name == "__END_OF_FILE_MARK__" && p.off == len(p.data) {
		repeat = 0
	}
	if repeat < 0 {
		repeat = 0
	}
	if err := p.validateFieldRepeat(fieldSpec, repeat); err != nil {
		return nil, err
	}
	field := &parsedNode{Name: fieldSpec.Name, Start: p.off}
	if strings.HasPrefix(fieldSpec.Type, "struct:") {
		structName := strings.TrimPrefix(fieldSpec.Type, "struct:")
		structSpec, ok := sectionSpec.Structs[structName]
		if !ok {
			return nil, fmt.Errorf("missing struct spec %s", structName)
		}
		for i := 0; i < repeat; i++ {
			elem, err := p.parseNode(structName, structSpec, field)
			if err != nil {
				return nil, fmt.Errorf("%s[%d]: %w", structName, i, err)
			}
			field.Elements = append(field.Elements, elem)
		}
		field.Value = field.Elements
		field.End = p.off
		return field, nil
	}
	values := make([]any, 0, repeat)
	var raw []byte
	for i := 0; i < repeat; i++ {
		value, chunk, err := p.readPrimitive(fieldSpec.Type)
		if err != nil {
			return nil, fmt.Errorf("item %d: %w", i, err)
		}
		values = append(values, value)
		raw = append(raw, chunk...)
	}
	field.Raw = raw
	field.End = p.off
	if fieldSpec.IsList != nil && !*fieldSpec.IsList && len(values) > 0 {
		field.Value = values[0]
	} else if repeat == 1 && fieldSpec.IsList == nil {
		field.Value = values[0]
	} else {
		field.Value = values
	}
	return field, nil
}

func (p *parser) validateFieldRepeat(field FieldSpec, repeat int) error {
	if repeat > maxScenarioFieldRepeat {
		return fmt.Errorf("%s repeat %d exceeds parser safety limit %d", field.Name, repeat, maxScenarioFieldRepeat)
	}
	if repeat == 0 || strings.HasPrefix(field.Type, "struct:") {
		return nil
	}
	typ, size, err := primitiveType(field.Type)
	if err != nil {
		return err
	}
	if typ == "str" {
		return nil
	}
	remaining := len(p.data) - p.off
	if size > 0 && repeat > remaining/size {
		return fmt.Errorf("%s repeat %d of %s exceeds remaining bytes %d", field.Name, repeat, field.Type, remaining)
	}
	return nil
}

func (p *parser) constructRepeat(field FieldSpec, node *parsedNode) (int, error) {
	repeat := field.Repeat
	if field.RepeatRule != nil {
		value, err := p.evaluateRepeatRule(*field.RepeatRule, node)
		if err != nil {
			return 0, err
		}
		repeat = value
	}
	return repeat, nil
}

func (p *parser) evaluateRepeatRule(rule RepeatRule, node *parsedNode) (int, error) {
	if len(rule.Fields) == 0 {
		return 0, fmt.Errorf("repeat rule has no fields")
	}
	values := make([]any, 0, len(rule.Fields))
	for _, ref := range rule.Fields {
		value, err := p.repeatFieldValue(ref, node)
		if err != nil {
			return 0, err
		}
		values = append(values, value)
	}
	switch rule.Kind {
	case RepeatFromField:
		return asInt(values[0]), nil
	case RepeatFromIndexedField:
		return listIntAt(values[0], rule.Then), nil
	case RepeatProduct:
		product := 1
		for _, value := range values {
			product *= asInt(value)
		}
		return product, nil
	case RepeatSquareRootLength:
		return int(math.Sqrt(float64(valueLen(values[0])))), nil
	case RepeatNonEmpty:
		if valueLen(values[0]) > 0 {
			return rule.Then, nil
		}
		return rule.Else, nil
	case RepeatProductNonEmpty:
		product := 1
		for _, value := range values {
			product *= asInt(value)
		}
		if product != 0 {
			return rule.Then, nil
		}
		return rule.Else, nil
	case RepeatFloatAtLeast:
		if asFloat(values[0]) >= rule.Threshold {
			return rule.Then, nil
		}
		return rule.Else, nil
	case RepeatRoundedFloatAtLeast:
		if math.Round(asFloat(values[0])*100)/100 >= rule.Threshold {
			return rule.Then, nil
		}
		return rule.Else, nil
	case RepeatTypeListOrValue:
		if _, ok := values[0].([]any); ok {
			return rule.Then, nil
		}
		return asInt(values[0]), nil
	default:
		return 0, fmt.Errorf("unknown repeat rule kind %d", rule.Kind)
	}
}

func (p *parser) repeatFieldValue(ref FieldRef, node *parsedNode) (any, error) {
	var owner *parsedNode
	if ref.Section == "self" {
		owner = node
	} else {
		owner = p.sections[ref.Section]
	}
	if owner == nil {
		return nil, fmt.Errorf("unknown repeat field section %s", ref.Section)
	}
	field := owner.field(ref.Name)
	if field == nil {
		return nil, fmt.Errorf("unknown repeat field %s:%s", ref.Section, ref.Name)
	}
	if p.allowReplaySentinels && ref.Name == "number_of_variables" {
		switch value := field.Value.(type) {
		case uint32:
			if value == ^uint32(0) {
				return 0, nil
			}
		case uint64:
			if value == ^uint64(0) {
				return 0, nil
			}
		case int:
			if value < 0 {
				return 0, nil
			}
		}
	}
	return field.Value, nil
}

func asInt(value any) int {
	switch v := value.(type) {
	case int:
		return v
	case int8:
		return int(v)
	case int16:
		return int(v)
	case int32:
		return int(v)
	case uint8:
		return int(v)
	case uint16:
		return int(v)
	case uint32:
		return int(v)
	case float64:
		return int(v)
	case []any:
		if len(v) == 0 {
			return 0
		}
		return asInt(v[0])
	}
	return 0
}

func asFloat(value any) float64 {
	switch v := value.(type) {
	case float32:
		return float64(v)
	case float64:
		return v
	case []any:
		if len(v) == 0 {
			return 0
		}
		return asFloat(v[0])
	}
	return float64(asInt(value))
}

func valueLen(value any) int {
	switch v := value.(type) {
	case []any:
		return len(v)
	case []*parsedNode:
		return len(v)
	case string:
		return len(v)
	}
	return 0
}

func listIntAt(value any, idx int) int {
	switch v := value.(type) {
	case []any:
		if idx >= 0 && idx < len(v) {
			return asInt(v[idx])
		}
	}
	return 0
}

func (p *parser) readPrimitive(kind string) (any, []byte, error) {
	typ, size, err := primitiveType(kind)
	if err != nil {
		return nil, nil, err
	}
	if typ == "str" {
		prefix, err := p.bytes(size)
		if err != nil {
			return nil, nil, err
		}
		n := signedInt(prefix)
		if n < 0 {
			if n == -1 {
				return "", append([]byte(nil), prefix...), nil
			}
			return nil, nil, fmt.Errorf("negative string length %d", n)
		}
		data, err := p.bytes(n)
		if err != nil {
			return nil, nil, err
		}
		raw := append(append([]byte(nil), prefix...), data...)
		return trimAtNUL(string(data)), raw, nil
	}
	raw, err := p.bytes(size)
	if err != nil {
		return nil, nil, err
	}
	switch typ {
	case "data":
		return append([]byte(nil), raw...), raw, nil
	case "c":
		return trimAtNUL(string(raw)), raw, nil
	case "s":
		switch size {
		case 1:
			return int8(raw[0]), raw, nil
		case 2:
			return int16(binary.LittleEndian.Uint16(raw)), raw, nil
		case 4:
			return int32(binary.LittleEndian.Uint32(raw)), raw, nil
		}
	case "u":
		switch size {
		case 1:
			return uint8(raw[0]), raw, nil
		case 2:
			return binary.LittleEndian.Uint16(raw), raw, nil
		case 4:
			return binary.LittleEndian.Uint32(raw), raw, nil
		}
	case "f":
		switch size {
		case 4:
			return math.Float32frombits(binary.LittleEndian.Uint32(raw)), raw, nil
		case 8:
			return math.Float64frombits(binary.LittleEndian.Uint64(raw)), raw, nil
		}
	}
	return nil, nil, fmt.Errorf("unsupported primitive %s", kind)
}

func primitiveType(kind string) (string, int, error) {
	if kind == "" {
		return "data", 0, nil
	}
	i := 0
	for i < len(kind) && (kind[i] < '0' || kind[i] > '9') {
		i++
	}
	prefix := kind[:i]
	num := kind[i:]
	if prefix == "" {
		prefix = "data"
	}
	var bits int
	if _, err := fmt.Sscanf(num, "%d", &bits); err != nil {
		return "", 0, fmt.Errorf("bad primitive kind %q", kind)
	}
	if prefix == "c" || prefix == "data" {
		return prefix, bits, nil
	}
	return prefix, bits / 8, nil
}

func signedInt(raw []byte) int {
	switch len(raw) {
	case 2:
		return int(int16(binary.LittleEndian.Uint16(raw)))
	case 4:
		return int(int32(binary.LittleEndian.Uint32(raw)))
	}
	return 0
}

func trimAtNUL(value string) string {
	if idx := strings.IndexByte(value, 0); idx >= 0 {
		return value[:idx]
	}
	return value
}

func (p *parser) bytes(n int) ([]byte, error) {
	if n < 0 {
		return nil, fmt.Errorf("negative byte count %d", n)
	}
	if p.off+n > len(p.data) {
		return nil, fmt.Errorf("need %d bytes at %d, only %d remain", n, p.off, len(p.data)-p.off)
	}
	out := p.data[p.off : p.off+n]
	p.off += n
	return out, nil
}
