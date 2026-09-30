package replay

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

const headerMetadataWindow = 8 << 20

// OpenMetadata reads bounded windows, not the complete initial-state snapshot.
// Consumers needing snapshot bytes must use ReplayCache.HeaderReader instead.
func OpenMetadata(path string) (*File, error) {
	c, err := OpenReplayCache(path)
	if err != nil {
		return nil, err
	}
	r, err := c.Metadata()
	if err == nil {
		r.Path = path
	}
	return r, err
}

// Metadata inspects bounded prefix/tail windows. Full snapshot bytes remain on
// disk. An unavailable graph is explicit, never a guessed successful parse.
func (c *ReplayCache) Metadata() (*File, error) {
	// v3 invalidates metadata produced before the embedded 1.59 trigger
	// section fallback was added.
	target := filepath.Join(c.Dir, "metadata-v3.json")
	if f, e := os.Open(target); e == nil {
		var rec File
		e = json.NewDecoder(io.LimitReader(f, maxReportBytes)).Decode(&rec)
		f.Close()
		if e == nil {
			return &rec, nil
		}
	}
	f, e := c.HeaderReader()
	if e != nil {
		return nil, e
	}
	defer f.Close()
	n := min(c.Manifest.HeaderBytes, int64(headerMetadataWindow))
	prefix := make([]byte, int(n))
	if _, e = io.ReadFull(f, prefix); e != nil {
		return nil, e
	}
	rec := &File{RecordSHA256: c.Manifest.SHA256, HeaderLength: int(c.Manifest.HeaderLength), InflatedBytes: int(c.Manifest.HeaderBytes)}
	rec.GameVersion, rec.SaveVersion = parseVersion(prefix)
	rec.AILoadout, rec.Players, e = parseDEAIAndPlayers(prefix, rec.SaveVersion)
	if c.Manifest.HeaderBytes <= n {
		rec.AILoadout, rec.Players, e = parseReplayAI(prefix, rec.SaveVersion)
	}
	if e != nil {
		rec.AIParseErr = e.Error()
		rec.Players, _ = parseDEPlayers(prefix, rec.SaveVersion)
	}
	if lobby, e := parseDELobbySettings(prefix, rec.SaveVersion); e == nil {
		rec.LobbySettings = &lobby
	}
	rec.DataSet = parseDataSetIdentity(prefix, rec.SaveVersion)
	meta := parseScenarioMetadata(prefix, rec.SaveVersion)
	rec.ScenarioMeta = &meta
	rec.ScenarioName = meta.Name
	rec.ScenarioDesc = meta.Description
	hm := parseHeaderMetadata(prefix, rec.SaveVersion)
	rec.HeaderMeta = &hm
	// Map terrain and roster reside in the structural prefix in known DE saves.
	if m, e := parseMapInfo(prefix, rec.SaveVersion); e == nil {
		payload := fmt.Sprintf("aoe2kit:fallback:terrain:v1:%dx%d:%s", m.Width, m.Height, m.TerrainSHA256)
		sum := sha256.Sum256([]byte(payload))
		rec.Fallback = &FallbackFingerprint{Tier: "fallback:terrain", SHA256: hex.EncodeToString(sum[:]), Method: "map dimensions + terrain/elevation grid", Map: m}
		rec.FallbackOK = true
	} else {
		rec.FallbackErr = "bounded header map inspection: " + e.Error()
	}
	if c.Manifest.HeaderBytes <= n {
		if fallback, err := fallbackFingerprint(prefix, rec.SaveVersion); err == nil {
			rec.Fallback, rec.FallbackOK, rec.FallbackErr = fallback, true, ""
		}
	}
	graphBytes := prefix
	base := 0
	if c.Manifest.HeaderBytes > n {
		base = int(c.Manifest.HeaderBytes - n)
		graphBytes = make([]byte, int(n))
		if _, e = f.ReadAt(graphBytes, int64(base)); e != nil {
			return nil, e
		}
	}
	rec.TriggerGraph, rec.TriggerRegion, e = parseTriggerGraphWithEmbeddedFallback(graphBytes)
	if e != nil {
		rec.TriggerGraphErr = "bounded header graph inspection: " + e.Error()
		rec.TriggerGraphTier = "unavailable"
	} else {
		rec.TriggerGraphOK = true
		rec.TriggerGraphTier = classifyTriggerGraphTier(rec.TriggerGraph, rec.TriggerRegion)
		if base > 0 {
			rec.TriggerGraphTier = "triggergraph_bounded_candidate"
			rec.TriggerGraph.Warnings = append(rec.TriggerGraph.Warnings, "graph searched within final 8 MiB of header; candidate parsing does not establish complete scenario coverage")
		}
		rec.TriggerGraph.Start += base
		rec.TriggerGraph.End += base
		if rec.TriggerRegion != nil {
			rec.TriggerRegion.Start += base
			rec.TriggerRegion.End += base
		}
	}
	body, e := os.Open(c.Source)
	if e == nil {
		reader := newHealthReader(body, c.Manifest.HeaderLength, c.Manifest.RecordBytes-c.Manifest.HeaderLength)
		rec.LogVersion, _ = readU32(reader)
		body.Close()
	}
	// Atomic derived cache publication; a failed write never poisons readers.
	if e = writeReplayJSON(target, rec); e != nil {
		return nil, e
	}
	return rec, nil
}

func writeReplayJSON(path string, v any) error {
	f, e := os.CreateTemp(filepath.Dir(path), ".json-")
	if e != nil {
		return e
	}
	defer os.Remove(f.Name())
	e = json.NewEncoder(f).Encode(v)
	ce := f.Close()
	if e != nil {
		return e
	}
	if ce != nil {
		return ce
	}
	return os.Rename(f.Name(), path)
}

// LobbyChat scans the disk header in overlapping bounded windows. Offsets are
// absolute and deduplicated across overlaps. No full-header string copy.
func (c *ReplayCache) LobbyChat(ids map[string]int) ([]ChatLine, error) {
	f, e := c.HeaderReader()
	if e != nil {
		return nil, e
	}
	defer f.Close()
	const chunk = 1 << 20
	const overlap = 64 << 10
	buf := make([]byte, chunk+overlap)
	var lines []ChatLine
	lastOffset := -1
	for off := int64(0); off < c.Manifest.HeaderBytes; off += chunk {
		n, e := f.ReadAt(buf, off)
		if e != nil && e != io.EOF {
			return nil, e
		}
		for _, line := range extractHeaderLobbyChat(buf[:n], ids) {
			if line.SourceOffset >= chunk {
				continue
			}
			line.SourceOffset += int(off)
			if line.SourceOffset <= lastOffset {
				continue
			}
			if len(lines) >= 10000 {
				return nil, fmt.Errorf("lobby chat exceeds 10000 row budget")
			}
			lines = append(lines, line)
			lastOffset = line.SourceOffset
		}
	}
	return lines, nil
}
