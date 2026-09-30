package replay

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"sort"

	"aoe2kit/pkg/aoe2"
)

type HeaderAnchorsOptions struct {
	Limit int
}

type HeaderAnchorsReport struct {
	Path         string                 `json:"path,omitempty"`
	Verification aoe2.VerificationClaim `json:"verification"`
	Summary      HeaderAnchorsSummary   `json:"summary"`
	Captions     []HeaderCaptionAnchor  `json:"captions,omitempty"`
	Resources    []PlayerResourceBlock  `json:"player_resource_blocks,omitempty"`
	Warnings     []string               `json:"warnings,omitempty"`
}

type HeaderAnchorsSummary struct {
	InflatedHeaderBytes        int `json:"inflated_header_bytes"`
	CaptionAnchors             int `json:"caption_anchors"`
	PlayerResourceBlocks       int `json:"player_resource_blocks"`
	ShownCaptionAnchors        int `json:"shown_caption_anchors"`
	ShownPlayerResourceBlocks  int `json:"shown_player_resource_blocks"`
	CaptionMedianStrideBytes   int `json:"caption_median_stride_bytes,omitempty"`
	CaptionMinStrideBytes      int `json:"caption_min_stride_bytes,omitempty"`
	CaptionMaxStrideBytes      int `json:"caption_max_stride_bytes,omitempty"`
	DecodedCaptionRecordBytes  int `json:"decoded_caption_record_bytes,omitempty"`
	DecodedResourceRecordBytes int `json:"decoded_resource_record_bytes,omitempty"`
}

type HeaderCaptionAnchor struct {
	MarkerStart int    `json:"marker_start"`
	TextStart   int    `json:"text_start"`
	End         int    `json:"end"`
	Bytes       int    `json:"bytes"`
	Length      int    `json:"length"`
	Text        string `json:"text"`
	HasLeadFF   bool   `json:"has_lead_ff"`
	Confidence  string `json:"confidence"`
}

type PlayerResourceBlock struct {
	Start      int                    `json:"start"`
	End        int                    `json:"end"`
	Bytes      int                    `json:"bytes"`
	Stride     int                    `json:"stride"`
	Players    []PlayerResourceRecord `json:"players"`
	Confidence string                 `json:"confidence"`
	Note       string                 `json:"note,omitempty"`
}

type PlayerResourceRecord struct {
	PlayerIndex int    `json:"player_index"`
	Player      int    `json:"player"`
	Start       int    `json:"start"`
	End         int    `json:"end"`
	Unknown0    uint32 `json:"unknown0"`
	Unknown1    uint32 `json:"unknown1"`
	Gold        uint32 `json:"gold"`
	Wood        uint32 `json:"wood"`
	Food        uint32 `json:"food"`
	Stone       uint32 `json:"stone"`
}

func BuildHeaderAnchors(path string, opts HeaderAnchorsOptions) (*HeaderAnchorsReport, error) {
	data, err := ReadRecordBytes(path)
	if err != nil {
		return nil, err
	}
	rec, err := Parse(data)
	if err != nil {
		return nil, err
	}
	header := rec.HeaderBytes()
	captions := headerCaptionAnchors(header, 0, len(header))
	resources := playerResourceBlocks(header, 0)
	limit := opts.Limit
	if limit < 0 {
		limit = 0
	}
	shownCaptions := captions
	if limit > 0 && len(shownCaptions) > limit {
		shownCaptions = shownCaptions[:limit]
	}
	shownResources := resources
	if limit > 0 && len(shownResources) > limit {
		shownResources = shownResources[:limit]
	}
	summary := HeaderAnchorsSummary{
		InflatedHeaderBytes:       len(header),
		CaptionAnchors:            len(captions),
		PlayerResourceBlocks:      len(resources),
		ShownCaptionAnchors:       len(shownCaptions),
		ShownPlayerResourceBlocks: len(shownResources),
	}
	summary.CaptionMedianStrideBytes, summary.CaptionMinStrideBytes, summary.CaptionMaxStrideBytes = captionStrideStats(captions)
	for _, anchor := range captions {
		summary.DecodedCaptionRecordBytes += anchor.Bytes
	}
	for _, block := range resources {
		if block.Confidence == "engine_verified_v4_resource_probe_anchor" {
			summary.DecodedResourceRecordBytes += block.Bytes
		}
	}
	return &HeaderAnchorsReport{
		Path: path,
		Verification: aoe2.VerificationClaim{
			Label:             "structure_verified_not_engine_verified",
			StructureVerified: true,
			EngineVerified:    false,
			Note:              "Header anchors are decoded from replay byte structure and v4 diagnostic sentinels; only the source diagnostic scenario was engine-verified.",
		},
		Summary:   summary,
		Captions:  shownCaptions,
		Resources: shownResources,
	}, nil
}

func headerCaptionAnchors(header []byte, start int, end int) []HeaderCaptionAnchor {
	if start < 0 {
		start = 0
	}
	if end > len(header) {
		end = len(header)
	}
	marker := []byte{0xff, 0xff, 0xff, 0xff, 0x00, 0x00, 0x00, 0x00, 0xff, 0xff, 0xff, 0xff}
	var out []HeaderCaptionAnchor
	for off := start; off+16 <= end; off++ {
		if !bytes.Equal(header[off:off+12], marker) {
			continue
		}
		length := int(binary.LittleEndian.Uint32(header[off+12:]))
		if length <= 0 || length > 512 || off+16+length > end {
			continue
		}
		textBytes := header[off+16 : off+16+length]
		if !printableHeaderASCII(textBytes) {
			continue
		}
		hasLeadFF := off >= 4 &&
			header[off-4] == 0xff &&
			header[off-3] == 0xff &&
			header[off-2] == 0xff &&
			header[off-1] == 0xff
		confidence := "parsed_ff_0_ff_u32_length_ascii_caption_candidate"
		text := string(textBytes)
		if len(text) >= 5 && text[:5] == "SDSV4" {
			confidence = "engine_verified_v4_caption_probe_anchor"
		}
		out = append(out, HeaderCaptionAnchor{
			MarkerStart: off,
			TextStart:   off + 16,
			End:         off + 16 + length,
			Bytes:       16 + length,
			Length:      length,
			Text:        text,
			HasLeadFF:   hasLeadFF,
			Confidence:  confidence,
		})
		off += 15 + length
	}
	return out
}

func printableHeaderASCII(data []byte) bool {
	if len(data) == 0 {
		return false
	}
	printable := 0
	for _, b := range data {
		if b == 0 {
			return false
		}
		if b >= 32 && b <= 126 {
			printable++
		}
	}
	return printable*100/len(data) >= 95
}

func captionStrideStats(captions []HeaderCaptionAnchor) (median int, min int, max int) {
	if len(captions) < 2 {
		return 0, 0, 0
	}
	deltas := make([]int, 0, len(captions)-1)
	for i := 1; i < len(captions); i++ {
		delta := captions[i].MarkerStart - captions[i-1].MarkerStart
		if delta > 0 && delta < 100000 {
			deltas = append(deltas, delta)
		}
	}
	if len(deltas) == 0 {
		return 0, 0, 0
	}
	sort.Ints(deltas)
	return deltas[len(deltas)/2], deltas[0], deltas[len(deltas)-1]
}

func playerResourceBlocks(header []byte, limit int) []PlayerResourceBlock {
	const stride = 28
	var blocks []PlayerResourceBlock
	for start := 0; start+stride*2 <= len(header); start += 4 {
		records := readPlayerResourceRun(header, start)
		if len(records) < 2 {
			continue
		}
		block := PlayerResourceBlock{
			Start:      start,
			End:        records[len(records)-1].End,
			Bytes:      records[len(records)-1].End - start,
			Stride:     stride,
			Players:    records,
			Confidence: "player_resource_record_stride_candidate",
			Note:       "layout decoded as u32 unknown0, u32 unknown1, u32 zero-based-player-index, then u32 gold/wood/food/stone",
		}
		if resourceRunIsV4Sentinel(records) {
			block.Confidence = "engine_verified_v4_resource_probe_anchor"
		}
		blocks = append(blocks, block)
		start += (len(records) - 1) * stride
	}
	sort.Slice(blocks, func(i, j int) bool {
		ci := resourceConfidenceRank(blocks[i].Confidence)
		cj := resourceConfidenceRank(blocks[j].Confidence)
		if ci != cj {
			return ci > cj
		}
		if len(blocks[i].Players) != len(blocks[j].Players) {
			return len(blocks[i].Players) > len(blocks[j].Players)
		}
		return blocks[i].Start < blocks[j].Start
	})
	if limit > 0 && len(blocks) > limit {
		return blocks[:limit]
	}
	return blocks
}

func readPlayerResourceRun(header []byte, start int) []PlayerResourceRecord {
	const stride = 28
	var records []PlayerResourceRecord
	for i := 0; start+(i+1)*stride <= len(header) && i < 16; i++ {
		off := start + i*stride
		playerIndex := int(binary.LittleEndian.Uint32(header[off+8:]))
		if playerIndex != i {
			break
		}
		rec := PlayerResourceRecord{
			PlayerIndex: playerIndex,
			Player:      playerIndex + 1,
			Start:       off,
			End:         off + stride,
			Unknown0:    binary.LittleEndian.Uint32(header[off:]),
			Unknown1:    binary.LittleEndian.Uint32(header[off+4:]),
			Gold:        binary.LittleEndian.Uint32(header[off+12:]),
			Wood:        binary.LittleEndian.Uint32(header[off+16:]),
			Food:        binary.LittleEndian.Uint32(header[off+20:]),
			Stone:       binary.LittleEndian.Uint32(header[off+24:]),
		}
		if !plausibleResourceRecord(rec) {
			break
		}
		records = append(records, rec)
	}
	return records
}

func plausibleResourceRecord(rec PlayerResourceRecord) bool {
	if rec.Unknown0 > 1000000 || rec.Unknown1 > 1000000 {
		return false
	}
	values := []uint32{rec.Gold, rec.Wood, rec.Food, rec.Stone}
	allZero := true
	for _, value := range values {
		if value > 1000000000 {
			return false
		}
		if value != 0 {
			allZero = false
		}
	}
	return !allZero
}

func resourceRunIsV4Sentinel(records []PlayerResourceRecord) bool {
	if len(records) < 2 {
		return false
	}
	for _, rec := range records {
		if rec.Gold < 90001 || rec.Gold > 99001 {
			return false
		}
		base := uint32(90000 + rec.Player*1000)
		if rec.Gold != base+1 || rec.Wood != base+2 || rec.Food != base+3 || rec.Stone != base+4 {
			return false
		}
	}
	return true
}

func resourceConfidenceRank(confidence string) int {
	switch confidence {
	case "engine_verified_v4_resource_probe_anchor":
		return 2
	default:
		return 1
	}
}

func (r *CoverageReport) coverHeaderBoundedOpaqueCaptionRecords(header []byte) {
	r.carveHeaderBoundedOpaqueIslands(header, func(start int, end int) []ReplayRegion {
		anchors := headerCaptionAnchors(header, start, end)
		regions := make([]ReplayRegion, 0, len(anchors))
		for i, anchor := range anchors {
			if !anchor.HasLeadFF {
				continue
			}
			text := anchor.Text
			if len(text) > 120 {
				text = text[:120]
			}
			regions = append(regions, replayRegion("inflated_header", fmt.Sprintf("caption_record_%04d", i), anchor.MarkerStart, anchor.End, "decoded", anchor.Confidence, map[string]any{
				"text":           text,
				"length":         anchor.Length,
				"has_leading_ff": anchor.HasLeadFF,
				"note":           "decoded ff/0/ff/u32-length ASCII object-caption field; surrounding object record remains opaque",
			}))
		}
		return regions
	})
}

func (r *CoverageReport) coverHeaderBoundedOpaquePlayerResourceBlocks(header []byte) {
	blocks := playerResourceBlocks(header, 0)
	islands := make([]ReplayRegion, 0, len(blocks))
	for _, block := range blocks {
		if block.Confidence != "engine_verified_v4_resource_probe_anchor" {
			continue
		}
		islands = append(islands, replayRegion("inflated_header", "player_resource_block_v4_probe", block.Start, block.End, "decoded", block.Confidence, map[string]any{
			"players": len(block.Players),
			"stride":  block.Stride,
			"note":    block.Note,
		}))
	}
	r.carveHeaderBoundedOpaquePrecomputedIslands(islands)
}

func (r *CoverageReport) carveHeaderBoundedOpaquePrecomputedIslands(islands []ReplayRegion) {
	if len(islands) == 0 {
		return
	}
	sort.Slice(islands, func(i, j int) bool {
		return islands[i].Start < islands[j].Start
	})
	var out []ReplayRegion
	for _, region := range r.Regions {
		if region.Space != "inflated_header" || region.Status != "bounded_opaque" {
			out = append(out, region)
			continue
		}
		cursor := region.Start
		emitted := 0
		for _, island := range islands {
			if island.End <= region.Start || island.Start >= region.End {
				continue
			}
			if island.Start < cursor || island.End > region.End || island.Start >= island.End {
				continue
			}
			if cursor < island.Start {
				out = append(out, replayRegion("inflated_header", fmt.Sprintf("%s_gap_%04d", region.Name, emitted), cursor, island.Start, region.Status, region.Confidence, map[string]any{
					"source_region": region.Name,
				}))
			}
			if island.Details == nil {
				island.Details = map[string]any{}
			}
			island.Details["source_region"] = region.Name
			if island.Name == "" {
				island.Name = fmt.Sprintf("%s_island_%04d", region.Name, emitted)
			} else {
				island.Name = fmt.Sprintf("%s_%s_%04d", region.Name, island.Name, emitted)
			}
			out = append(out, island)
			cursor = island.End
			emitted++
		}
		if emitted == 0 {
			out = append(out, region)
			continue
		}
		if cursor < region.End {
			out = append(out, replayRegion("inflated_header", fmt.Sprintf("%s_gap_%04d", region.Name, emitted), cursor, region.End, region.Status, region.Confidence, map[string]any{
				"source_region": region.Name,
			}))
		}
	}
	r.Regions = out
}

func (r *CoverageReport) carveHeaderBoundedOpaqueIslands(header []byte, finder func(start int, end int) []ReplayRegion) {
	var out []ReplayRegion
	for _, region := range r.Regions {
		if region.Space != "inflated_header" || region.Status != "bounded_opaque" {
			out = append(out, region)
			continue
		}
		islands := finder(region.Start, region.End)
		if len(islands) == 0 {
			out = append(out, region)
			continue
		}
		sort.Slice(islands, func(i, j int) bool {
			return islands[i].Start < islands[j].Start
		})
		cursor := region.Start
		emitted := 0
		for _, island := range islands {
			if island.Start < cursor || island.End > region.End || island.Start >= island.End {
				continue
			}
			if cursor < island.Start {
				out = append(out, replayRegion("inflated_header", fmt.Sprintf("%s_gap_%04d", region.Name, emitted), cursor, island.Start, region.Status, region.Confidence, map[string]any{
					"source_region": region.Name,
				}))
			}
			if island.Details == nil {
				island.Details = map[string]any{}
			}
			island.Details["source_region"] = region.Name
			if island.Name == "" {
				island.Name = fmt.Sprintf("%s_island_%04d", region.Name, emitted)
			} else {
				island.Name = fmt.Sprintf("%s_%s_%04d", region.Name, island.Name, emitted)
			}
			out = append(out, island)
			cursor = island.End
			emitted++
		}
		if emitted == 0 {
			out = append(out, region)
			continue
		}
		if cursor < region.End {
			out = append(out, replayRegion("inflated_header", fmt.Sprintf("%s_gap_%04d", region.Name, emitted), cursor, region.End, region.Status, region.Confidence, map[string]any{
				"source_region": region.Name,
			}))
		}
	}
	r.Regions = out
}
