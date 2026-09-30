package replay

import (
	"archive/zip"
	"bytes"
	"compress/flate"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"

	"aoe2kit/pkg/aifile"
	"aoe2kit/pkg/scenario"
	"aoe2kit/pkg/triggergraph"
)

type FallbackFingerprint struct {
	Tier     string   `json:"tier"`
	SHA256   string   `json:"sha256"`
	Method   string   `json:"method"`
	Map      *MapInfo `json:"map,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}

type MapInfo struct {
	Width         int       `json:"width"`
	Height        int       `json:"height"`
	TileCount     int       `json:"tile_count"`
	TerrainSHA256 string    `json:"terrain_sha256"`
	Tiles         []MapTile `json:"tiles,omitempty"`
}

type MapTile struct {
	X         int `json:"x"`
	Y         int `json:"y"`
	Terrain   int `json:"terrain"`
	Elevation int `json:"elevation"`
}

type MapInfoOptions struct {
	IncludeTiles bool
}

type LobbySettings struct {
	Source              string   `json:"source"`
	Confidence          string   `json:"confidence"`
	Build               uint32   `json:"build,omitempty"`
	Timestamp           uint32   `json:"timestamp,omitempty"`
	GameType            uint32   `json:"game_type,omitempty"`
	MapDimension        uint32   `json:"map_dimension,omitempty"`
	RMSMapID            uint32   `json:"rms_map_id,omitempty"`
	VictoryType         uint32   `json:"victory_type,omitempty"`
	VictoryValue        uint32   `json:"victory_value,omitempty"`
	StartingResources   uint32   `json:"starting_resources,omitempty"`
	StartingAge         uint32   `json:"starting_age,omitempty"`
	MapReveal           uint32   `json:"map_reveal,omitempty"`
	GameSpeed           uint32   `json:"game_speed,omitempty"`
	TreatyLength        uint32   `json:"treaty_length,omitempty"`
	PopulationLimit     uint32   `json:"population_limit,omitempty"`
	PlayerCount         uint32   `json:"player_count,omitempty"`
	Difficulty          uint8    `json:"difficulty,omitempty"`
	RandomPositions     uint8    `json:"random_positions,omitempty"`
	AllTechnologies     uint8    `json:"all_technologies,omitempty"`
	LockTeams           uint8    `json:"lock_teams,omitempty"`
	LockSpeed           uint8    `json:"lock_speed,omitempty"`
	Multiplayer         uint8    `json:"multiplayer,omitempty"`
	Cheats              uint8    `json:"cheats,omitempty"`
	RecordGame          uint8    `json:"record_game,omitempty"`
	AnimalsEnabled      uint8    `json:"animals_enabled,omitempty"`
	PredatorsEnabled    uint8    `json:"predators_enabled,omitempty"`
	Turbo               uint8    `json:"turbo,omitempty"`
	SharedExploration   uint8    `json:"shared_exploration,omitempty"`
	TeamTogether        uint8    `json:"team_together,omitempty"`
	DiplomacyType       uint8    `json:"diplomacy_type,omitempty"`
	Ranked              uint8    `json:"ranked,omitempty"`
	RawPreDLC           []uint32 `json:"raw_pre_dlc,omitempty"`
	RawPostMapDimension uint32   `json:"raw_post_map_dimension,omitempty"`
	RawPostRMSMapID     uint32   `json:"raw_post_rms_map_id,omitempty"`
	RawPreSpeed         []uint32 `json:"raw_pre_speed,omitempty"`
	RawPreDifficulty    []byte   `json:"raw_pre_difficulty,omitempty"`
	RawTeamSettings     []byte   `json:"raw_team_settings,omitempty"`
	RawPostTeams        []uint32 `json:"raw_post_teams,omitempty"`
	RawTailFlags        []byte   `json:"raw_tail_flags,omitempty"`
}

type PlayerSlot struct {
	Slot           int    `json:"slot"`
	Active         bool   `json:"active"`
	Number         int    `json:"number,omitempty"`
	Name           string `json:"name,omitempty"`
	AIName         string `json:"ai_name,omitempty"`
	DLCID          int    `json:"dlc_id,omitempty"`
	Color          int    `json:"color,omitempty"`
	Team           int    `json:"team,omitempty"`
	SelectedTeamID int    `json:"selected_team_id,omitempty"`
	ResolvedTeamID int    `json:"resolved_team_id,omitempty"`
	Civ            int    `json:"civ,omitempty"`
	ProfileID      int    `json:"profile_id,omitempty"`
	PlayerType     int    `json:"player_type"`
	Human          bool   `json:"human"`
	Kind           string `json:"kind"`
}

type DataSetIdentity struct {
	Status         string   `json:"status"`
	ActiveDataSet  string   `json:"active_data_set,omitempty"`
	ActiveDataSets []string `json:"active_data_sets,omitempty"`
	Checksum       uint32   `json:"checksum,omitempty"`
	WorkshopID     uint32   `json:"workshop_id,omitempty"`
	Source         string   `json:"source"`
	Confidence     string   `json:"confidence"`
	Error          string   `json:"error,omitempty"`
}

type ScenarioMetadata struct {
	Name              string `json:"name,omitempty"`
	Description       string `json:"description,omitempty"`
	NameOffset        int    `json:"name_offset,omitempty"`
	DescriptionOffset int    `json:"description_offset,omitempty"`
	Source            string `json:"source"`
	Confidence        string `json:"confidence"`
	Error             string `json:"error,omitempty"`
}

type HeaderMetadata struct {
	Source         string `json:"source"`
	Confidence     string `json:"confidence"`
	TailGUIDHex    string `json:"tail_guid_hex,omitempty"`
	TailGUIDOffset int    `json:"tail_guid_offset,omitempty"`
	Error          string `json:"error,omitempty"`
}

type deModdedDataSet struct {
	Name       string
	Checksum   uint32
	WorkshopID uint32
}

type File struct {
	Path             string               `json:"path,omitempty"`
	RecordSHA256     string               `json:"record_sha256,omitempty"`
	HeaderLength     int                  `json:"header_length"`
	InflatedBytes    int                  `json:"inflated_header_bytes"`
	GameVersion      string               `json:"game_version"`
	SaveVersion      float64              `json:"save_version"`
	LogVersion       uint32               `json:"log_version"`
	TriggerRegion    *triggergraph.Region `json:"trigger_region,omitempty"`
	TriggerGraph     *triggergraph.Graph  `json:"trigger_graph,omitempty"`
	TriggerGraphTier string               `json:"trigger_graph_decode_tier,omitempty"`
	TriggerGraphOK   bool                 `json:"trigger_graph_ok"`
	TriggerGraphErr  string               `json:"trigger_graph_error,omitempty"`
	Fallback         *FallbackFingerprint `json:"fallback,omitempty"`
	FallbackOK       bool                 `json:"fallback_ok"`
	FallbackErr      string               `json:"fallback_error,omitempty"`
	AILoadout        *aifile.Loadout      `json:"ai_loadout,omitempty"`
	DataSet          DataSetIdentity      `json:"data_set_identity"`
	ScenarioName     string               `json:"scenario_name,omitempty"`
	ScenarioDesc     string               `json:"scenario_description,omitempty"`
	ScenarioMeta     *ScenarioMetadata    `json:"scenario_metadata,omitempty"`
	HeaderMeta       *HeaderMetadata      `json:"header_metadata,omitempty"`
	LobbySettings    *LobbySettings       `json:"lobby_settings,omitempty"`
	Players          []PlayerSlot         `json:"players,omitempty"`
	AIParseErr       string               `json:"ai_parse_error,omitempty"`

	header []byte
}

func Open(path string) (*File, error) {
	data, err := ReadRecordBytes(path)
	if err != nil {
		return nil, err
	}
	file, err := Parse(data)
	if err != nil {
		return nil, err
	}
	file.Path = path
	return file, nil
}

func ReadRecordBytes(path string) ([]byte, error) {
	if strings.EqualFold(filepath.Ext(path), ".zip") {
		return readRecordFromZip(path)
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return readLegacyBounded(f, "recording", 32<<20)
}

func readRecordFromZip(path string) ([]byte, error) {
	zr, err := zip.OpenReader(path)
	if err != nil {
		return nil, err
	}
	defer zr.Close()
	for _, file := range zr.File {
		if !strings.EqualFold(filepath.Ext(file.Name), ".aoe2record") {
			continue
		}
		rc, err := file.Open()
		if err != nil {
			return nil, err
		}
		defer rc.Close()
		return readLegacyBounded(rc, "unzipped recording", 32<<20)
	}
	return nil, fmt.Errorf("%s: zip contains no .aoe2record", path)
}

func Parse(data []byte) (*File, error) {
	if len(data) < 12 {
		return nil, fmt.Errorf("record too short")
	}
	recordSum := sha256.Sum256(data)
	headerLength := int(binary.LittleEndian.Uint32(data[0:4]))
	if headerLength < 8 || headerLength > len(data) {
		return nil, fmt.Errorf("invalid compressed header length %d for file length %d", headerLength, len(data))
	}
	header, err := inflateRaw(data[8:headerLength])
	if err != nil {
		return nil, fmt.Errorf("inflate record header: %w", err)
	}
	file := &File{
		RecordSHA256:  hex.EncodeToString(recordSum[:]),
		HeaderLength:  headerLength,
		InflatedBytes: len(header),
		header:        header,
	}
	file.GameVersion, file.SaveVersion = parseVersion(header)
	if headerLength+4 <= len(data) {
		file.LogVersion = binary.LittleEndian.Uint32(data[headerLength:])
	}
	graph, region, err := parseTriggerGraphWithEmbeddedFallback(header)
	if err != nil {
		file.TriggerGraphErr = err.Error()
		file.TriggerGraphTier = "unavailable"
	} else {
		file.TriggerRegion = region
		file.TriggerGraph = graph
		file.TriggerGraphTier = classifyTriggerGraphTier(graph, region)
		file.TriggerGraphOK = true
	}
	fallback, err := fallbackFingerprint(header, file.SaveVersion)
	if err != nil {
		file.FallbackErr = err.Error()
	} else {
		file.Fallback = fallback
		file.FallbackOK = true
	}
	lobbySettings, lobbyErr := parseDELobbySettings(header, file.SaveVersion)
	if lobbyErr == nil {
		file.LobbySettings = &lobbySettings
	}
	loadout, players, err := parseReplayAI(header, file.SaveVersion)
	if err != nil {
		file.AIParseErr = err.Error()
	}
	if loadout != nil {
		file.AILoadout = loadout
	}
	if len(players) > 0 {
		file.Players = players
	}
	file.DataSet = parseDataSetIdentity(header, file.SaveVersion)
	scenarioMeta := parseScenarioMetadata(header, file.SaveVersion)
	file.ScenarioMeta = &scenarioMeta
	file.ScenarioName = scenarioMeta.Name
	file.ScenarioDesc = scenarioMeta.Description
	headerMeta := parseHeaderMetadata(header, file.SaveVersion)
	file.HeaderMeta = &headerMeta
	return file, nil
}

func parseTriggerGraphWithEmbeddedFallback(header []byte) (*triggergraph.Graph, *triggergraph.Region, error) {
	graph, region, err := triggergraph.ParseLocated(header)
	if err == nil {
		return graph, region, nil
	}
	if embedded, embeddedRegion, embeddedErr := parseEmbedded159TriggerGraph(header); embeddedErr == nil {
		return embedded, embeddedRegion, nil
	}
	return nil, nil, err
}

// parseEmbedded159TriggerGraph handles the scenario section copied into a
// current DE replay. The replay copy is not the older standalone trigger
// graph wire shape: it carries a five-byte trailer between trigger records
// and condition records use marker 34. The versioned scenario parser owns
// those details, so this fallback only locates a bounded section candidate.
func parseEmbedded159TriggerGraph(header []byte) (*triggergraph.Graph, *triggergraph.Region, error) {
	region, err := triggergraph.FindRegion(header)
	if err != nil {
		return nil, nil, err
	}
	strings := triggergraph.ScenarioStrings(header)
	start := region.Start
	if len(strings) > 0 {
		start = strings[len(strings)-1].EndOffset
	}
	end := region.End
	if end <= start || end > len(header) {
		return nil, nil, fmt.Errorf("invalid embedded scenario search bounds %d..%d", start, end)
	}
	const maxCandidates = 256 * 1024
	if end-start > maxCandidates {
		end = start + maxCandidates
	}
	for candidate := start; candidate+13 <= end; candidate++ {
		// Triggers begins with f64 version, s8 instruction marker, and s32
		// trigger count. This cheap shape gate avoids invoking the full parser
		// on the surrounding replay state tail.
		versionBits := binary.LittleEndian.Uint64(header[candidate:])
		version := math.Float64frombits(versionBits)
		if version < 3 || version > 6 || header[candidate+8] > 8 {
			continue
		}
		count := int(int32(binary.LittleEndian.Uint32(header[candidate+9:])))
		if count <= 0 || count > 20000 {
			continue
		}
		graph, consumed, parseErr := scenario.ParseInflatedTriggers("1.59", header[candidate:end])
		if parseErr != nil || graph == nil || graph.TriggerCount != count || graph.EffectCount+graph.ConditionCount == 0 {
			continue
		}
		graph.Start += candidate
		graph.End += candidate
		graph.Warnings = append(graph.Warnings, "trigger graph decoded from embedded DE 1.59 scenario section")
		located := *region
		located.Start = candidate
		located.End = candidate + consumed
		located.Warnings = append(located.Warnings, "embedded DE 1.59 scenario section located by bounded schema probe")
		return graph, &located, nil
	}
	return nil, nil, fmt.Errorf("could not locate embedded DE 1.59 scenario trigger section")
}

func (f *File) HeaderBytes() []byte {
	return append([]byte(nil), f.header...)
}

func inflateRaw(compressed []byte) ([]byte, error) {
	r := flate.NewReader(bytes.NewReader(compressed))
	defer r.Close()
	return readLegacyBounded(r, "inflated header", 32<<20)
}

// Compatibility APIs still return slices. Refuse large inputs until their
// consumers migrate to HeaderReader/WalkOps; GOMEMLIMIT is not a hard cap.
func readLegacyBounded(r io.Reader, label string, limit int64) ([]byte, error) {
	b, err := io.ReadAll(io.LimitReader(r, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(b)) > limit {
		return nil, fmt.Errorf("legacy in-memory %s exceeds %d MiB; use disk-backed replay info, chat, issues, sync or health", label, limit>>20)
	}
	return b, nil
}

func parseVersion(header []byte) (string, float64) {
	nul := bytes.IndexByte(header, 0)
	if nul < 0 || nul > 16 {
		nul = 7
	}
	if nul > len(header) {
		nul = len(header)
	}
	game := strings.TrimRight(string(header[:nul]), "\x00")
	saveOffset := 8
	if len(header) < saveOffset+4 {
		return game, 0
	}
	oldSave := float64(math.Float32frombits(binary.LittleEndian.Uint32(header[saveOffset:])))
	if oldSave == -1 && len(header) >= saveOffset+8 {
		newSave := binary.LittleEndian.Uint32(header[saveOffset+4:])
		if newSave == 37 {
			return game, 37.0
		}
		return game, math.Round((float64(newSave)/(1<<16))*100) / 100
	}
	return game, math.Round(oldSave*100) / 100
}

func fallbackFingerprint(header []byte, saveVersion float64) (*FallbackFingerprint, error) {
	if mapInfo, err := parseMapInfo(header, saveVersion); err == nil {
		payload := fmt.Sprintf("aoe2kit:fallback:terrain:v1:%dx%d:%s", mapInfo.Width, mapInfo.Height, mapInfo.TerrainSHA256)
		sum := sha256.Sum256([]byte(payload))
		return &FallbackFingerprint{
			Tier:   "fallback:terrain",
			SHA256: hex.EncodeToString(sum[:]),
			Method: "map dimensions + terrain/elevation grid",
			Map:    mapInfo,
		}, nil
	}
	if fp, err := headerStringsFingerprint(header); err == nil {
		return fp, nil
	}
	return nil, fmt.Errorf("no fallback fingerprint could be computed")
}

func headerStringsFingerprint(header []byte) (*FallbackFingerprint, error) {
	strings := triggergraph.ScenarioStrings(header)
	var useful []string
	for _, s := range strings {
		if len(s.Text) >= 4 {
			useful = append(useful, s.Text)
		}
	}
	if len(useful) == 0 {
		return nil, fmt.Errorf("no scenario strings available for fallback")
	}
	var buf bytes.Buffer
	buf.WriteString("aoe2kit:fallback:header_strings:v1\n")
	for _, s := range useful {
		buf.WriteString(s)
		buf.WriteByte('\n')
	}
	sum := sha256.Sum256(buf.Bytes())
	return &FallbackFingerprint{
		Tier:   "fallback:header_strings",
		SHA256: hex.EncodeToString(sum[:]),
		Method: "ordered printable scenario strings from inflated replay header",
		Warnings: []string{
			"weaker fallback: scenario strings can miss terrain-only edits",
		},
	}, nil
}

func parseReplayAI(header []byte, saveVersion float64) (*aifile.Loadout, []PlayerSlot, error) {
	loadout, players, err := parseDEAIAndPlayers(header, saveVersion)
	if err == nil {
		return loadout, players, nil
	}
	if partialPlayers, playerErr := parseDEPlayers(header, saveVersion); playerErr == nil {
		players = partialPlayers
	}
	// Fallback keeps the empty/default case available if a future DE header
	// shifts before the structural ai_files field. Trigger strings do not cover
	// DE ai_files, so any non-empty custom AI replay must be proven structurally.
	scenarioStrings := triggergraph.ScenarioStrings(header)
	texts := make([]string, 0, len(scenarioStrings))
	for _, item := range scenarioStrings {
		texts = append(texts, item.Text)
	}
	fallback := aifile.NewLoadout(aifile.ParseReferences(texts))
	return &fallback, players, err
}

func parseDEAIAndPlayers(header []byte, saveVersion float64) (*aifile.Loadout, []PlayerSlot, error) {
	c, err := newDECursor(header, saveVersion, "AI loadout parser")
	if err != nil {
		return nil, nil, err
	}
	players, err := readDEPlayers(c, saveVersion)
	if err != nil {
		return nil, nil, err
	}
	refs, err := readDEAIFiles(c, saveVersion)
	if err != nil {
		return nil, nil, err
	}
	loadout := aifile.NewLoadout(aifile.ParseReferences(refs))
	return &loadout, players, nil
}

func parseDataSetIdentity(header []byte, saveVersion float64) DataSetIdentity {
	identity := DataSetIdentity{
		Status:     "unknown",
		Source:     "embedded_mod_block",
		Confidence: "not_available",
	}
	dataSet, err := readDEModdedDataSet(header, saveVersion)
	if err != nil {
		if name := scanScenarioPathDataSetSuffix(header); name != "" {
			identity.Status = "modded"
			identity.ActiveDataSet = name
			identity.ActiveDataSets = []string{name}
			identity.Source = "early_scenario_metadata_dataset_suffix"
			identity.Confidence = "heuristic_name_only_no_checksum"
			identity.Error = err.Error()
			return identity
		}
		identity.Error = err.Error()
		return identity
	}
	name := strings.TrimSpace(dataSet.Name)
	identity.Confidence = "parsed"
	if name == "" {
		identity.Status = "vanilla"
		return identity
	}
	identity.Status = "modded"
	identity.ActiveDataSet = name
	identity.ActiveDataSets = []string{name}
	identity.Checksum = dataSet.Checksum
	identity.WorkshopID = dataSet.WorkshopID
	return identity
}

func scanScenarioPathDataSetSuffix(header []byte) string {
	text := string(header)
	idx := strings.Index(text, ".aoe2scenario:")
	if idx < 0 {
		return ""
	}
	start := idx + len(".aoe2scenario:")
	rest := text[start:]
	end := strings.IndexByte(rest, ':')
	if end <= 0 {
		return ""
	}
	name := strings.TrimSpace(rest[:end])
	if name == "" || strings.ContainsAny(name, "\x00\r\n/\\") {
		return ""
	}
	if len(name) > 80 {
		return ""
	}
	return name
}

func parseScenarioMetadata(header []byte, saveVersion float64) ScenarioMetadata {
	meta, err := readDEScenarioMetadata(header, saveVersion)
	if err == nil && meta.Name != "" {
		return meta
	}
	fallback := scanEarlyScenarioMetadata(header)
	if fallback.Name != "" {
		if err != nil {
			fallback.Error = err.Error()
		}
		return fallback
	}
	return ScenarioMetadata{
		Source:     "de_lobby_tail",
		Confidence: "not_available",
		Error:      errorString(err),
	}
}

func parseHeaderMetadata(header []byte, saveVersion float64) HeaderMetadata {
	meta, err := readDEHeaderMetadata(header, saveVersion)
	if err == nil {
		return meta
	}
	return HeaderMetadata{
		Source:     "de_tail_after_ai",
		Confidence: "not_available",
		Error:      err.Error(),
	}
}

func readDEHeaderMetadata(header []byte, saveVersion float64) (HeaderMetadata, error) {
	c, err := newDECursor(header, saveVersion, "replay header metadata parser")
	if err != nil {
		return HeaderMetadata{}, err
	}
	if _, err := readDEPlayers(c, saveVersion); err != nil {
		return HeaderMetadata{}, err
	}
	if _, err := readDEAIFiles(c, saveVersion); err != nil {
		return HeaderMetadata{}, err
	}
	c.skip(8)
	guidOffset := c.off
	guid, err := c.bytes(16)
	if err != nil {
		return HeaderMetadata{}, err
	}
	return HeaderMetadata{
		Source:         "de_tail_after_ai",
		Confidence:     "parsed_raw_guid_bytes_before_lobby_name",
		TailGUIDHex:    hex.EncodeToString(guid),
		TailGUIDOffset: guidOffset,
	}, c.err
}

func readDEScenarioMetadata(header []byte, saveVersion float64) (ScenarioMetadata, error) {
	c, err := newDECursor(header, saveVersion, "replay scenario metadata parser")
	if err != nil {
		return ScenarioMetadata{}, err
	}
	if _, err := readDEPlayers(c, saveVersion); err != nil {
		return ScenarioMetadata{}, err
	}
	if _, err := readDEAIFiles(c, saveVersion); err != nil {
		return ScenarioMetadata{}, err
	}
	c.skip(8)
	c.skip(16) // guid
	nameOffset := c.off
	nameBytes, err := c.deString()
	if err != nil {
		return ScenarioMetadata{}, fmt.Errorf("scenario name: %w", err)
	}
	name := strings.TrimSpace(decodeCString(nameBytes))
	return ScenarioMetadata{
		Name:       name,
		NameOffset: nameOffset,
		Source:     "de_lobby_tail",
		Confidence: "parsed_de_lobby_tail_string",
	}, c.err
}

func scanEarlyScenarioMetadata(header []byte) ScenarioMetadata {
	limit := len(header)
	if limit > 8192 {
		limit = 8192
	}
	bestScore := -1
	var best ScenarioMetadata
	for off := 0; off+4 <= limit; off++ {
		if header[off] != 0x60 || header[off+1] != 0x0a {
			continue
		}
		length := int(int16(binary.LittleEndian.Uint16(header[off+2:])))
		if length < 4 || off+4+length > limit {
			continue
		}
		rawText := strings.TrimSpace(decodeCString(header[off+4 : off+4+length]))
		text, source, ok := normalizeScenarioTitleCandidate(rawText)
		if !ok || !plausibleScenarioTitle(text) {
			continue
		}
		score := scenarioTitleScore(text, off)
		if source == "early_de_scenario_path" {
			score += 8
		}
		if score > bestScore {
			bestScore = score
			best = ScenarioMetadata{
				Name:       text,
				NameOffset: off,
				Source:     source,
				Confidence: "heuristic_early_header_string",
			}
		}
		off += 3 + length
	}
	return best
}

func normalizeScenarioTitleCandidate(text string) (string, string, bool) {
	lower := strings.ToLower(text)
	if idx := strings.LastIndex(lower, "scenarios:"); idx >= 0 {
		title := text[idx+len("scenarios:"):]
		if sep := strings.Index(title, "::"); sep >= 0 {
			title = title[:sep]
		}
		title = strings.TrimSuffix(title, ".aoe2scenario")
		title = strings.TrimSuffix(title, ".AOE2SCENARIO")
		title = strings.TrimSpace(strings.ReplaceAll(title, "_", " "))
		if title != "" {
			return title, "early_de_scenario_path", true
		}
	}
	return text, "early_de_string_scan", text != ""
}

func plausibleScenarioTitle(text string) bool {
	if len(text) < 14 || len(text) > 160 {
		return false
	}
	if !plausibleHeaderString(text) {
		return false
	}
	lower := strings.ToLower(text)
	rejects := []string{"/", "\\", ".ai", ".per", ".xs", "promisory", "random-map-scripts:", "lmods:", "resources/_common", "scenario_editor", "none:"}
	for _, reject := range rejects {
		if strings.Contains(lower, reject) {
			return false
		}
	}
	return true
}

func scenarioTitleScore(text string, off int) int {
	score := 0
	if off > 1500 {
		score += 4
	}
	if strings.Contains(text, " ") {
		score += 3
	}
	if strings.Contains(text, "-") {
		score++
	}
	if strings.Contains(strings.ToLower(text), "cba") {
		score++
	}
	return score
}

func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func readDEModdedDataSet(header []byte, saveVersion float64) (deModdedDataSet, error) {
	c, err := newDECursor(header, saveVersion, "replay data-set parser")
	if err != nil {
		return deModdedDataSet{}, err
	}
	if _, err := readDEPlayers(c, saveVersion); err != nil {
		return deModdedDataSet{}, err
	}
	if _, err := readDEAIFiles(c, saveVersion); err != nil {
		return deModdedDataSet{}, err
	}
	c.skip(8)
	c.skip(16) // guid
	if _, err := c.deString(); err != nil {
		return deModdedDataSet{}, fmt.Errorf("lobby name: %w", err)
	}
	c.skip(8)
	mod, err := c.deString()
	if err != nil {
		return deModdedDataSet{}, fmt.Errorf("modded dataset: %w", err)
	}
	workshopID, err := c.u32()
	if err != nil {
		return deModdedDataSet{}, fmt.Errorf("modded dataset workshop id: %w", err)
	}
	c.skip(4) // reserved zero in observed DE v68 mod blocks.
	checksum, err := c.u32()
	if err != nil {
		return deModdedDataSet{}, fmt.Errorf("modded dataset checksum: %w", err)
	}
	if c.err != nil {
		return deModdedDataSet{}, c.err
	}
	return deModdedDataSet{
		Name:       decodeCString(mod),
		Checksum:   checksum,
		WorkshopID: workshopID,
	}, nil
}

func parseDEPlayers(header []byte, saveVersion float64) ([]PlayerSlot, error) {
	c, err := newDECursor(header, saveVersion, "replay player parser")
	if err != nil {
		return nil, err
	}
	return readDEPlayers(c, saveVersion)
}

func parseDELobbySettings(header []byte, saveVersion float64) (LobbySettings, error) {
	c, err := newDECursor(header, saveVersion, "replay lobby settings parser")
	if err != nil {
		return LobbySettings{}, err
	}
	settings, _, err := readDELobbySettingsPrefix(c)
	return settings, err
}

func newDECursor(header []byte, saveVersion float64, label string) (*cursor, error) {
	if !bytes.HasPrefix(header, []byte("VER 9.4\x00")) {
		return nil, fmt.Errorf("unsupported replay header version")
	}
	c := &cursor{data: header}
	c.skip(8)
	oldSave, err := c.float32()
	if err != nil {
		return nil, err
	}
	if oldSave == -1 {
		if _, err := c.u32(); err != nil {
			return nil, err
		}
	}
	if saveVersion < 68 {
		return nil, fmt.Errorf("%s currently targets current DE save_version 68+, got %.2f", label, saveVersion)
	}
	return c, nil
}

func readDEPlayers(c *cursor, save float64) ([]PlayerSlot, error) {
	settings, numPlayers, err := readDELobbySettingsPrefix(c)
	if err != nil {
		return nil, err
	}
	_ = settings
	if numPlayers == 0 || numPlayers > 16 {
		return nil, fmt.Errorf("unreasonable DE player count %d", numPlayers)
	}
	players := make([]PlayerSlot, 0, int(numPlayers))
	slotRecords := int(numPlayers)
	if slotRecords < 8 {
		slotRecords = 8
	}
	for i := 0; i < slotRecords; i++ {
		marker, err := seekDEPlayerMarker(c, i)
		if err != nil {
			if i >= int(numPlayers) {
				break
			}
			return nil, err
		}
		dlcID := int(binary.LittleEndian.Uint32(c.data[marker-8:]))
		color := int(int32(binary.LittleEndian.Uint32(c.data[marker-4:])))
		selectedTeam := int(c.data[marker+1])
		resolvedTeam := int(c.data[marker+2])
		c.off = marker + 3
		c.skip(8) // dat CRC
		c.skip(1) // mp game version
		civ, err := c.u32()
		if err != nil {
			return nil, err
		}
		customCivCount, err := c.u32()
		if err != nil {
			return nil, err
		}
		for j := uint32(0); j < customCivCount && j < 64; j++ {
			if c.remaining() >= 2 && c.data[c.off] == 0x60 && c.data[c.off+1] == 0x0a {
				break
			}
			c.skip(4)
		}
		if _, err := c.deString(); err != nil {
			return nil, fmt.Errorf("player %d data string: %w", i, err)
		}
		c.skip(1)
		aiNameBytes, err := c.deString()
		if err != nil {
			return nil, fmt.Errorf("player %d ai name: %w", i, err)
		}
		if _, err := c.deString(); err != nil {
			return nil, fmt.Errorf("player %d censored name: %w", i, err)
		}
		nameBytes, err := c.deString()
		if err != nil {
			return nil, fmt.Errorf("player %d name: %w", i, err)
		}
		playerType, err := c.u32()
		if err != nil {
			return nil, err
		}
		profileID, err := c.u32()
		if err != nil {
			return nil, err
		}
		c.skip(4)
		number, err := c.u32()
		if err != nil {
			return nil, err
		}
		if i < int(numPlayers) {
			players = append(players, PlayerSlot{
				Slot:           i + 1,
				Active:         true,
				Number:         int(int32(number)),
				Name:           decodeCString(nameBytes),
				AIName:         decodeCString(aiNameBytes),
				DLCID:          dlcID,
				Color:          int(color),
				Team:           normalizedLobbyTeam(resolvedTeam),
				SelectedTeamID: selectedTeam,
				ResolvedTeamID: resolvedTeam,
				Civ:            int(civ),
				ProfileID:      int(int32(profileID)),
				PlayerType:     int(playerType),
				Human:          playerType == 2,
				Kind:           playerTypeKind(playerType),
			})
		}
		c.skip(1) // prefer random
		c.skip(1)
		c.skip(8) // handicap
		c.skip(4) // >=64.3
	}
	if c.err != nil {
		return nil, c.err
	}
	return players, nil
}

func readDELobbySettingsPrefix(c *cursor) (LobbySettings, uint32, error) {
	settings := LobbySettings{
		Source:     "de_header_settings_prefix",
		Confidence: "parsed_existing_readDEPlayers_skip_chain",
	}
	var err error
	if settings.Build, err = c.u32(); err != nil {
		return settings, 0, err
	}
	if settings.Timestamp, err = c.u32(); err != nil {
		return settings, 0, err
	}
	for i := 0; i < 3; i++ {
		v, err := c.u32()
		if err != nil {
			return settings, 0, err
		}
		settings.RawPreDLC = append(settings.RawPreDLC, v)
	}
	dlcCount, err := c.u32()
	if err != nil {
		return settings, 0, err
	}
	c.skip(int(dlcCount) * 4)
	if settings.GameType, err = c.u32(); err != nil {
		return settings, 0, err
	}
	if settings.MapDimension, err = c.u32(); err != nil {
		return settings, 0, err
	}
	if settings.RawPostMapDimension, err = c.u32(); err != nil {
		return settings, 0, err
	}
	if settings.RMSMapID, err = c.u32(); err != nil {
		return settings, 0, err
	}
	if settings.RawPostRMSMapID, err = c.u32(); err != nil {
		return settings, 0, err
	}
	if settings.VictoryType, err = c.u32(); err != nil {
		return settings, 0, err
	}
	if settings.VictoryValue, err = c.u32(); err != nil {
		return settings, 0, err
	}
	if settings.StartingResources, err = c.u32(); err != nil {
		return settings, 0, err
	}
	if settings.StartingAge, err = c.u32(); err != nil {
		return settings, 0, err
	}
	for i := 0; i < 3; i++ {
		v, err := c.u32()
		if err != nil {
			return settings, 0, err
		}
		settings.RawPreSpeed = append(settings.RawPreSpeed, v)
	}
	if len(settings.RawPreSpeed) > 0 {
		settings.MapReveal = settings.RawPreSpeed[0]
	}
	if settings.GameSpeed, err = c.u32(); err != nil {
		return settings, 0, err
	}
	if settings.TreatyLength, err = c.u32(); err != nil {
		return settings, 0, err
	}
	if settings.PopulationLimit, err = c.u32(); err != nil {
		return settings, 0, err
	}
	numPlayers, err := c.u32()
	if err != nil {
		return settings, 0, err
	}
	settings.PlayerCount = numPlayers
	rawPreDifficulty, err := c.bytes(14)
	if err != nil {
		return settings, 0, err
	}
	settings.RawPreDifficulty = append([]byte(nil), rawPreDifficulty...)
	if settings.Difficulty, err = c.u8(); err != nil {
		return settings, 0, err
	}
	if settings.RandomPositions, err = c.u8(); err != nil {
		return settings, 0, err
	}
	if settings.AllTechnologies, err = c.u8(); err != nil {
		return settings, 0, err
	}
	unknown, err := c.u8()
	if err != nil {
		return settings, 0, err
	}
	settings.RawTailFlags = append(settings.RawTailFlags, unknown)
	teamSettings, err := c.bytes(10)
	if err != nil {
		return settings, 0, err
	}
	settings.RawTeamSettings = append([]byte(nil), teamSettings...)
	if len(teamSettings) >= 10 {
		settings.LockTeams = teamSettings[0]
		settings.LockSpeed = teamSettings[1]
		settings.Multiplayer = teamSettings[2]
		settings.Cheats = teamSettings[3]
		settings.RecordGame = teamSettings[4]
		settings.AnimalsEnabled = teamSettings[5]
		settings.PredatorsEnabled = teamSettings[6]
		settings.Turbo = teamSettings[7]
		settings.SharedExploration = teamSettings[8]
		settings.TeamTogether = teamSettings[9]
	}
	for i := 0; i < 3; i++ {
		v, err := c.u32()
		if err != nil {
			return settings, 0, err
		}
		settings.RawPostTeams = append(settings.RawPostTeams, v)
	}
	if settings.DiplomacyType, err = c.u8(); err != nil {
		return settings, 0, err
	}
	if settings.Ranked, err = c.u8(); err != nil {
		return settings, 0, err
	}
	return settings, numPlayers, c.err
}

func seekDEPlayerMarker(c *cursor, slot int) (int, error) {
	if c.err != nil {
		return 0, c.err
	}
	limit := c.off + 512
	if limit > len(c.data)-16 {
		limit = len(c.data) - 16
	}
	for off := c.off + 8; off <= limit; off++ {
		if c.data[off] != 0xff || !plausibleDETeamByte(c.data[off+1]) || !plausibleDETeamByte(c.data[off+2]) {
			continue
		}
		dlcID := binary.LittleEndian.Uint32(c.data[off-8:])
		colorID := binary.LittleEndian.Uint32(c.data[off-4:])
		if dlcID != 15 || (colorID > 15 && colorID != 0xffffffff) {
			continue
		}
		return off, nil
	}
	c.err = fmt.Errorf("player %d marker not found near header offset %d", slot+1, c.off)
	return 0, c.err
}

func plausibleDETeamByte(value byte) bool {
	// The bytes after the ff player marker are the selected and resolved team
	// ids. Older fixtures only exercised no-team/small values, but current DE
	// team games store lobby team ids here (for example 2, 3, 6).
	return value <= 16
}

func normalizedLobbyTeam(resolvedTeam int) int {
	// DE stores 1 as "no team"; mgz summaries only form teams for values > 1.
	if resolvedTeam <= 1 {
		return 0
	}
	return resolvedTeam - 1
}

func readDEAIFiles(c *cursor, save float64) ([]string, error) {
	c.skip(12) // post-player bytes through fog/cheat/colored-chat
	c.skip(4)  // current DE v68 field before separator
	c.skip(4)  // separator
	c.skip(1)  // ranked
	c.skip(1)  // allow_specs
	c.skip(4)  // lobby_visibility
	c.skip(1)  // hidden_civs
	c.skip(1)  // matchmaking
	c.skip(4)  // spec_delay
	c.skip(1)  // scenario_civ
	if err := skipStringBlock(c); err != nil {
		return nil, err
	}
	c.skip(8)
	for i := 0; i < 20; i++ {
		if err := skipStringBlock(c); err != nil {
			return nil, err
		}
	}
	numSN, err := c.u32()
	if err != nil {
		return nil, err
	}
	if numSN > 100000 {
		return nil, fmt.Errorf("unreasonable strategic number count %d", numSN)
	}
	c.skip(int(numSN) * 4)
	numAI, err := c.u64()
	if err != nil {
		return nil, err
	}
	if numAI > 10000 {
		return nil, fmt.Errorf("unreasonable AI file count %d", numAI)
	}
	refs := make([]string, 0, int(numAI))
	for i := uint64(0); i < numAI; i++ {
		c.skip(4)
		name, err := c.deString()
		if err != nil {
			return nil, fmt.Errorf("ai_files[%d]: %w", i, err)
		}
		c.skip(4)
		refs = append(refs, decodeCString(name))
	}
	if c.err != nil {
		return nil, c.err
	}
	return refs, nil
}

func decodeCString(data []byte) string {
	return strings.TrimRight(string(data), "\x00")
}

func playerTypeKind(playerType uint32) string {
	switch playerType {
	case 0:
		return "absent"
	case 1:
		return "closed"
	case 2:
		return "human"
	case 3:
		return "eliminated"
	case 4:
		return "computer"
	case 5:
		return "cyborg"
	case 6:
		return "spectator"
	default:
		return "other"
	}
}

func parseMapInfo(header []byte, saveVersion float64) (*MapInfo, error) {
	return parseMapInfoWithOptions(header, saveVersion, MapInfoOptions{})
}

func parseMapInfoWithOptions(header []byte, saveVersion float64, opts MapInfoOptions) (*MapInfo, error) {
	if !bytes.HasPrefix(header, []byte("VER 9.4\x00")) {
		return nil, fmt.Errorf("unsupported replay header version")
	}
	c := &cursor{data: header}
	c.skip(8)
	oldSave, err := c.float32()
	if err != nil {
		return nil, err
	}
	if oldSave == -1 {
		if _, err := c.u32(); err != nil {
			return nil, err
		}
	}
	if saveVersion < 68 {
		return nil, fmt.Errorf("fallback map parser currently targets current DE save_version 68+, got %.2f", saveVersion)
	}
	if err := skipDE(c, saveVersion); err != nil {
		return nil, fmt.Errorf("skip DE settings: %w", err)
	}
	numPlayers, err := skipMetadata(c, saveVersion)
	if err != nil {
		return nil, fmt.Errorf("skip metadata: %w", err)
	}
	if numPlayers <= 0 || numPlayers > 16 {
		return nil, fmt.Errorf("invalid replay num_players %d", numPlayers)
	}
	return readMapInfoWithOptions(c, saveVersion, opts)
}

func skipDE(c *cursor, save float64) error {
	if _, err := readDEPlayers(c, save); err != nil {
		return err
	}
	if _, err := readDEAIFiles(c, save); err != nil {
		return err
	}
	return skipDETailAfterAI(c)
}

func skipDETailAfterAI(c *cursor) error {
	c.skip(8)
	c.skip(16) // guid
	if _, err := c.deString(); err != nil {
		return err
	}
	c.skip(8)
	if _, err := c.deString(); err != nil {
		return err
	}
	c.skip(33)
	c.skip(1)
	c.skip(8)
	c.skip(21)
	c.skip(4)
	c.skip(8)
	c.skip(3)
	c.skip(8)
	c.skip(1)
	c.skip(5)
	unknownCount, err := c.u32()
	if err != nil {
		return err
	}
	c.skip(12)
	c.skip(int(unknownCount) * 4)
	if _, err := c.deString(); err != nil {
		return err
	}
	c.skip(8)
	c.skip(8) // timestamp + unknown
	c.skip(8) // >=68
	return c.err
}

func skipStringBlock(c *cursor) error {
	for {
		crc, err := c.u32()
		if err != nil {
			return err
		}
		if crc > 0 && crc < 255 {
			return nil
		}
		if _, err := c.deString(); err != nil {
			return err
		}
	}
}

func skipMetadata(c *cursor, save float64) (int, error) {
	ai, err := c.u32()
	if err != nil {
		return 0, err
	}
	if ai > 0 {
		pos := bytes.Index(c.data[c.off:], bytes.Repeat([]byte{0}, 4096))
		if pos < 0 {
			return 0, fmt.Errorf("could not find AI block end")
		}
		c.off += pos + 4096
	}
	c.skip(24)
	c.skip(4) // game speed
	c.skip(17)
	c.skip(2) // owner id
	numPlayers, err := c.byte()
	if err != nil {
		return 0, err
	}
	c.skip(1)
	c.skip(1) // cheats
	c.skip(24 + int(numPlayers)*4)
	return int(numPlayers), c.err
}

func readMapInfo(c *cursor, save float64) (*MapInfo, error) {
	return readMapInfoWithOptions(c, save, MapInfoOptions{})
}

func readMapInfoWithOptions(c *cursor, save float64, opts MapInfoOptions) (*MapInfo, error) {
	c.skip(8)
	sizeX, err := c.u32()
	if err != nil {
		return nil, err
	}
	sizeY, err := c.u32()
	if err != nil {
		return nil, err
	}
	zoneNum, err := c.u32()
	if err != nil {
		return nil, err
	}
	tileNum64 := uint64(sizeX) * uint64(sizeY)
	if sizeX == 0 || sizeY == 0 || tileNum64 > 1024*1024 {
		return nil, fmt.Errorf("unreasonable map dimensions %dx%d", sizeX, sizeY)
	}
	tileNum := int(tileNum64)
	for i := uint32(0); i < zoneNum; i++ {
		c.skip(2048 + tileNum*2)
		numFloats, err := c.u32()
		if err != nil {
			return nil, err
		}
		c.skip(int(numFloats) * 4)
		c.skip(4)
	}
	c.skip(2) // all_visible + fog_of_war
	th := sha256.New()
	var tiles []MapTile
	if opts.IncludeTiles {
		tiles = make([]MapTile, 0, tileNum)
	}
	for i := 0; i < tileNum; i++ {
		terrain, err := c.byte()
		if err != nil {
			return nil, err
		}
		c.skip(2)
		elevation, err := c.byte()
		if err != nil {
			return nil, err
		}
		c.skip(6)
		th.Write([]byte{terrain, elevation})
		if opts.IncludeTiles {
			tiles = append(tiles, MapTile{
				X:         i % int(sizeX),
				Y:         i / int(sizeX),
				Terrain:   int(terrain),
				Elevation: int(elevation),
			})
		}
	}
	numData, err := c.u32()
	if err != nil {
		return nil, err
	}
	if numData > 1000000 {
		return nil, fmt.Errorf("unreasonable map data count %d", numData)
	}
	c.skip(4)
	c.skip(int(numData) * 4)
	for i := uint32(0); i < numData; i++ {
		numObstructions, err := c.u32()
		if err != nil {
			return nil, err
		}
		if numObstructions > 1000000 {
			return nil, fmt.Errorf("unreasonable obstruction count %d", numObstructions)
		}
		c.skip(int(numObstructions) * 8)
	}
	c.skip(8) // repeated map dimensions
	c.skip(tileNum * 4)
	if save >= 61.5 {
		c.skip(tileNum * 4)
	}
	return &MapInfo{
		Width:         int(sizeX),
		Height:        int(sizeY),
		TileCount:     tileNum,
		TerrainSHA256: hex.EncodeToString(th.Sum(nil)),
		Tiles:         tiles,
	}, c.err
}

type cursor struct {
	data []byte
	off  int
	err  error
}

func (c *cursor) remaining() int {
	return len(c.data) - c.off
}

func (c *cursor) skip(n int) {
	if c.err != nil {
		return
	}
	if n < 0 || c.off+n > len(c.data) {
		c.err = fmt.Errorf("need %d bytes at %d, only %d remain", n, c.off, len(c.data)-c.off)
		return
	}
	c.off += n
}

func (c *cursor) seekRel(n int) {
	if c.err != nil {
		return
	}
	next := c.off + n
	if next < 0 || next > len(c.data) {
		c.err = fmt.Errorf("seek relative %d from %d outside 0..%d", n, c.off, len(c.data))
		return
	}
	c.off = next
}

func (c *cursor) byte() (byte, error) {
	if c.err != nil {
		return 0, c.err
	}
	if c.off >= len(c.data) {
		c.err = fmt.Errorf("need byte at %d, no bytes remain", c.off)
		return 0, c.err
	}
	value := c.data[c.off]
	c.off++
	return value, nil
}

func (c *cursor) bytes(n int) ([]byte, error) {
	if c.err != nil {
		return nil, c.err
	}
	if n < 0 || c.off+n > len(c.data) {
		c.err = fmt.Errorf("need %d bytes at %d, only %d remain", n, c.off, len(c.data)-c.off)
		return nil, c.err
	}
	value := c.data[c.off : c.off+n]
	c.off += n
	return value, nil
}

func (c *cursor) u8() (uint8, error) {
	value, err := c.byte()
	return uint8(value), err
}

func (c *cursor) u32() (uint32, error) {
	if c.err != nil {
		return 0, c.err
	}
	if c.off+4 > len(c.data) {
		c.err = fmt.Errorf("need uint32 at %d, only %d remain", c.off, len(c.data)-c.off)
		return 0, c.err
	}
	value := binary.LittleEndian.Uint32(c.data[c.off:])
	c.off += 4
	return value, nil
}

func (c *cursor) u64() (uint64, error) {
	if c.err != nil {
		return 0, c.err
	}
	if c.off+8 > len(c.data) {
		c.err = fmt.Errorf("need uint64 at %d, only %d remain", c.off, len(c.data)-c.off)
		return 0, c.err
	}
	value := binary.LittleEndian.Uint64(c.data[c.off:])
	c.off += 8
	return value, nil
}

func (c *cursor) float32() (float64, error) {
	value, err := c.u32()
	if err != nil {
		return 0, err
	}
	return float64(math.Float32frombits(value)), nil
}

func (c *cursor) deString() ([]byte, error) {
	if c.err != nil {
		return nil, c.err
	}
	if c.off+4 > len(c.data) {
		c.err = fmt.Errorf("need DE string marker at %d", c.off)
		return nil, c.err
	}
	if c.data[c.off] != 0x60 || c.data[c.off+1] != 0x0a {
		c.err = fmt.Errorf("bad DE string marker at %d", c.off)
		return nil, c.err
	}
	c.off += 2
	length := int(int16(binary.LittleEndian.Uint16(c.data[c.off:])))
	c.off += 2
	if length < 0 || c.off+length > len(c.data) {
		c.err = fmt.Errorf("bad DE string length %d at %d", length, c.off-2)
		return nil, c.err
	}
	out := c.data[c.off : c.off+length]
	c.off += length
	return out, nil
}
