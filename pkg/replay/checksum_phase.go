package replay

import (
	"fmt"
	"sort"
)

type ChecksumPhaseOptions struct {
	WordIndex int
	PlayerID  int
}

type ChecksumPhaseAllReport struct {
	Path         string                  `json:"path,omitempty"`
	Method       string                  `json:"method"`
	Verification string                  `json:"verification"`
	Summary      ChecksumPhaseAllSummary `json:"summary"`
	Words        []*ChecksumPhaseReport  `json:"words"`
}

type ChecksumPhaseAllSummary struct {
	ChecksumSamples int `json:"checksum_samples"`
	Players         int `json:"players"`
	Words           int `json:"words"`
}

type ChecksumPhaseReport struct {
	Path          string                `json:"path,omitempty"`
	Method        string                `json:"method"`
	Verification  string                `json:"verification"`
	Summary       ChecksumPhaseSummary  `json:"summary"`
	Players       []ChecksumPhasePlayer `json:"players,omitempty"`
	Warnings      []string              `json:"warnings,omitempty"`
	WordSemantics map[string]string     `json:"checksum_word_semantics,omitempty"`
}

type ChecksumPhaseSummary struct {
	WordIndex int    `json:"word_index"`
	WordName  string `json:"word_name"`
	Samples   int    `json:"checksum_samples"`
	Players   int    `json:"players"`
}

type ChecksumPhasePlayer struct {
	PlayerID                int                     `json:"player_id"`
	Samples                 int                     `json:"samples"`
	DistinctValues          int                     `json:"distinct_values"`
	KnownStateChanges       int                     `json:"known_state_changes"`
	Transitions             int                     `json:"transitions"`
	DominantTransitions     int                     `json:"dominant_transitions"`
	DominantTransitionShare float64                 `json:"dominant_transition_share"`
	RepeatTransitions       int                     `json:"repeat_transitions"`
	AmbiguousFromValues     int                     `json:"ambiguous_from_values"`
	Ring                    *ChecksumWordRing       `json:"ring,omitempty"`
	Values                  []ChecksumWordValue     `json:"values,omitempty"`
	Successors              []ChecksumWordSuccessor `json:"successors,omitempty"`
}

type ChecksumWordRing struct {
	Detected           bool     `json:"detected"`
	Length             int      `json:"length"`
	Values             []uint32 `json:"values_u32,omitempty"`
	ValuesSigned       []int32  `json:"values_i32,omitempty"`
	ForwardSkips       int      `json:"forward_skips"`
	ReverseTransitions int      `json:"reverse_transitions"`
	OffRingTransitions int      `json:"off_ring_transitions"`
}

type ChecksumWordValue struct {
	U32   uint32 `json:"u32"`
	I32   int32  `json:"i32"`
	Count int    `json:"count"`
}

type ChecksumWordSuccessor struct {
	FromU32     uint32  `json:"from_u32"`
	FromI32     int32   `json:"from_i32"`
	ToU32       uint32  `json:"to_u32"`
	ToI32       int32   `json:"to_i32"`
	Count       int     `json:"count"`
	Dominant    bool    `json:"dominant"`
	ShareOfFrom float64 `json:"share_of_from"`
}

func BuildChecksumPhase(path string, opts ChecksumPhaseOptions) (*ChecksumPhaseReport, error) {
	if opts.WordIndex < 0 || opts.WordIndex > 10 {
		return nil, fmt.Errorf("word index must be 0..10")
	}
	sync, err := BuildSyncStream(path, SyncOptions{ChecksumsOnly: true, RawWords: true})
	if err != nil {
		return nil, err
	}
	report := &ChecksumPhaseReport{
		Path:          path,
		Method:        "checksum_word_successor_relation",
		Verification:  "structure_verified_successor_analysis_not_semantic_decode",
		Warnings:      append([]string{}, sync.Warnings...),
		WordSemantics: sync.WordSemantics,
	}
	report.Summary.WordIndex = opts.WordIndex
	report.Summary.WordName = syncWordName(opts.WordIndex)
	report.Summary.Samples = sync.Summary.ChecksumDE

	byPlayer := map[int][]SyncRawWordRow{}
	for _, row := range sync.RawWords {
		if opts.PlayerID != 0 && row.PlayerID != opts.PlayerID {
			continue
		}
		if len(row.Words) <= opts.WordIndex {
			continue
		}
		if len(row.Words) > 8 && row.Words[8] != uint32(row.PlayerID) {
			continue
		}
		byPlayer[row.PlayerID] = append(byPlayer[row.PlayerID], row)
	}
	players := make([]int, 0, len(byPlayer))
	for playerID := range byPlayer {
		players = append(players, playerID)
	}
	sort.Ints(players)
	for _, playerID := range players {
		report.Players = append(report.Players, buildChecksumPhasePlayer(playerID, byPlayer[playerID], opts.WordIndex))
	}
	report.Summary.Players = len(report.Players)
	return report, nil
}

func BuildChecksumPhaseAll(path string, opts ChecksumPhaseOptions) (*ChecksumPhaseAllReport, error) {
	report := &ChecksumPhaseAllReport{
		Path:         path,
		Method:       "checksum_word_successor_relation_all_words",
		Verification: "structure_verified_successor_analysis_not_semantic_decode",
	}
	for word := 0; word <= 10; word++ {
		wordOpts := opts
		wordOpts.WordIndex = word
		wordReport, err := BuildChecksumPhase(path, wordOpts)
		if err != nil {
			return nil, err
		}
		if word == 0 {
			report.Summary.ChecksumSamples = wordReport.Summary.Samples
			report.Summary.Players = wordReport.Summary.Players
		}
		report.Words = append(report.Words, wordReport)
	}
	report.Summary.Words = len(report.Words)
	return report, nil
}

func buildChecksumPhasePlayer(playerID int, rows []SyncRawWordRow, wordIndex int) ChecksumPhasePlayer {
	player := ChecksumPhasePlayer{PlayerID: playerID, Samples: len(rows)}
	if len(rows) == 0 {
		return player
	}
	valueCounts := map[uint32]int{}
	successorCounts := map[uint32]map[uint32]int{}
	fromTotals := map[uint32]int{}
	for i, row := range rows {
		value := row.Words[wordIndex]
		valueCounts[value]++
		if i > 0 {
			prev := rows[i-1]
			if checksumKnownStateChanged(prev.Words, row.Words) {
				player.KnownStateChanges++
			}
			from := prev.Words[wordIndex]
			to := value
			if successorCounts[from] == nil {
				successorCounts[from] = map[uint32]int{}
			}
			successorCounts[from][to]++
			fromTotals[from]++
			player.Transitions++
			if from == to {
				player.RepeatTransitions++
			}
		}
	}
	player.DistinctValues = len(valueCounts)
	player.Values = checksumWordValues(valueCounts)
	var dominant map[uint32]uint32
	player.Successors, player.DominantTransitions, player.AmbiguousFromValues, dominant = checksumWordSuccessors(successorCounts, fromTotals)
	player.Ring = checksumWordRing(valueCounts, successorCounts, dominant)
	if player.Transitions > 0 {
		player.DominantTransitionShare = float64(player.DominantTransitions) / float64(player.Transitions)
	}
	return player
}

func checksumKnownStateChanged(a, b []uint32) bool {
	for _, idx := range []int{1, 2, 3, 4, 6, 7, 10} {
		if len(a) <= idx || len(b) <= idx {
			continue
		}
		if a[idx] != b[idx] {
			return true
		}
	}
	return false
}

func checksumWordValues(counts map[uint32]int) []ChecksumWordValue {
	values := make([]ChecksumWordValue, 0, len(counts))
	for value, count := range counts {
		values = append(values, ChecksumWordValue{U32: value, I32: int32(value), Count: count})
	}
	sort.Slice(values, func(i, j int) bool {
		if values[i].Count != values[j].Count {
			return values[i].Count > values[j].Count
		}
		return values[i].U32 < values[j].U32
	})
	return values
}

func checksumWordSuccessors(counts map[uint32]map[uint32]int, totals map[uint32]int) ([]ChecksumWordSuccessor, int, int, map[uint32]uint32) {
	var successors []ChecksumWordSuccessor
	dominantTransitions := 0
	ambiguousFromValues := 0
	dominant := map[uint32]uint32{}
	fromValues := make([]uint32, 0, len(counts))
	for from := range counts {
		fromValues = append(fromValues, from)
	}
	sort.Slice(fromValues, func(i, j int) bool { return fromValues[i] < fromValues[j] })
	for _, from := range fromValues {
		toCounts := counts[from]
		if len(toCounts) > 1 {
			ambiguousFromValues++
		}
		bestTo := uint32(0)
		bestCount := -1
		toValues := make([]uint32, 0, len(toCounts))
		for to := range toCounts {
			toValues = append(toValues, to)
		}
		sort.Slice(toValues, func(i, j int) bool { return toValues[i] < toValues[j] })
		for _, to := range toValues {
			count := toCounts[to]
			if count > bestCount {
				bestTo = to
				bestCount = count
			}
		}
		dominantTransitions += bestCount
		dominant[from] = bestTo
		for _, to := range toValues {
			count := toCounts[to]
			successors = append(successors, ChecksumWordSuccessor{
				FromU32:     from,
				FromI32:     int32(from),
				ToU32:       to,
				ToI32:       int32(to),
				Count:       count,
				Dominant:    to == bestTo,
				ShareOfFrom: float64(count) / float64(totals[from]),
			})
		}
	}
	sort.Slice(successors, func(i, j int) bool {
		if successors[i].Dominant != successors[j].Dominant {
			return successors[i].Dominant
		}
		if successors[i].Count != successors[j].Count {
			return successors[i].Count > successors[j].Count
		}
		if successors[i].FromU32 != successors[j].FromU32 {
			return successors[i].FromU32 < successors[j].FromU32
		}
		return successors[i].ToU32 < successors[j].ToU32
	})
	return successors, dominantTransitions, ambiguousFromValues, dominant
}

func checksumWordRing(valueCounts map[uint32]int, successorCounts map[uint32]map[uint32]int, dominant map[uint32]uint32) *ChecksumWordRing {
	if len(valueCounts) <= 1 || len(dominant) != len(valueCounts) {
		return nil
	}
	startSet := false
	start := uint32(0)
	startCount := -1
	for value, count := range valueCounts {
		if !startSet || count > startCount || (count == startCount && value < start) {
			start = value
			startCount = count
			startSet = true
		}
	}
	seen := map[uint32]bool{}
	ring := []uint32{}
	current := start
	for {
		if seen[current] {
			if current != start || len(ring) != len(valueCounts) {
				return nil
			}
			break
		}
		if !valueCountsHas(valueCounts, current) {
			return nil
		}
		seen[current] = true
		ring = append(ring, current)
		next, ok := dominant[current]
		if !ok {
			return nil
		}
		current = next
	}
	if len(ring) != len(valueCounts) {
		return nil
	}
	index := map[uint32]int{}
	signed := make([]int32, len(ring))
	for i, value := range ring {
		index[value] = i
		signed[i] = int32(value)
	}
	result := &ChecksumWordRing{Detected: true, Length: len(ring), Values: ring, ValuesSigned: signed}
	for from, toCounts := range successorCounts {
		fromIndex, ok := index[from]
		if !ok {
			result.OffRingTransitions += sumTransitionCounts(toCounts)
			continue
		}
		for to, count := range toCounts {
			toIndex, ok := index[to]
			if !ok {
				result.OffRingTransitions += count
				continue
			}
			distance := (toIndex - fromIndex + len(ring)) % len(ring)
			switch {
			case distance == 0:
				// Repeats are already reported separately; keep them out of forward-skip counts.
			case distance == 1:
				// Dominant one-step advance.
			case distance > 1:
				result.ForwardSkips += count
			default:
				result.ReverseTransitions += count
			}
		}
	}
	return result
}

func valueCountsHas(counts map[uint32]int, value uint32) bool {
	_, ok := counts[value]
	return ok
}

func sumTransitionCounts(counts map[uint32]int) int {
	total := 0
	for _, count := range counts {
		total += count
	}
	return total
}
