package cba

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"

	"aoe2kit/pkg/replay"
)

// The ladder admits exactly one scenario: the community's official CBA Requiem V292.
//
// Identity is the EMBEDDED FILENAME TOKEN plus a trigger-count band -- deliberately not
// the lobby name (hosts rename lobbies freely, so lobby text is worthless as identity)
// and deliberately not the trigger-graph hash (that identifies an EXACT host copy, and
// the same community V292 ships as at least three host copies with different graphs).
// The token is community identity; the band rejects same-named forks and test builds.
const (
	LadderToken       = "CBA_=REQUIEM=_V292"
	RequiemTriggerMin = 2700
	RequiemTriggerMax = 3400
)

// scenarioTokenRe matches the embedded scenario filename, including variant suffixes
// ("CBA_=REQUIEM=_V293 Random Position") so a variant is identified rather than silently
// passing as the base version.
//
// The suffix is matched as up to two WHOLE WORDS of 2+ letters, not as a run of
// "[A-Za-z ]". A character-class run is greedy across spaces and swallows whatever text
// follows the name in the header -- which would corrupt the token, and since admission
// is an equality test against LadderToken, a corrupted V292 token gets rejected as a
// "non-ladder variant". Word-bounded matching stops at the first non-word byte.
var scenarioTokenRe = regexp.MustCompile(`CBA_=REQUIEM=_V\d{3}(?:[ _][A-Za-z]{2,}){0,2}`)

// ScenarioIdentity is why a replay was admitted to, or rejected from, the ladder.
type ScenarioIdentity struct {
	Token        string `json:"token,omitempty"`
	TriggerCount int    `json:"trigger_count,omitempty"`
	Ladder       bool   `json:"ladder"`
	Reason       string `json:"reason"`
}

// IdentifyScenario classifies one replay. The reason field is always populated so a
// rejection can be audited without re-running anything.
func IdentifyScenario(path string) ScenarioIdentity {
	id := ScenarioIdentity{}

	rec, err := replay.Open(path)
	if err != nil {
		id.Reason = "unreadable: " + err.Error()
		return id
	}
	if rec.TriggerGraph != nil {
		id.TriggerCount = rec.TriggerGraph.TriggerCount
	}
	id.Token = scenarioToken(rec.HeaderBytes())

	switch {
	case id.Token == "":
		id.Reason = "no CBA Requiem token in header"
	case id.Token != LadderToken:
		id.Reason = "non-ladder variant: " + id.Token
	case id.TriggerCount == 0:
		id.Reason = "trigger graph unavailable"
	case id.TriggerCount < RequiemTriggerMin || id.TriggerCount > RequiemTriggerMax:
		id.Reason = fmt.Sprintf("trigger count %d outside V292 band %d..%d",
			id.TriggerCount, RequiemTriggerMin, RequiemTriggerMax)
	default:
		id.Ladder = true
		id.Reason = "official V292"
	}
	return id
}

// scenarioToken returns the most frequent token in the header. Frequency rather than
// first-match: the header carries the scenario name in several places, and a stray
// single occurrence (a chat line, a settings blob) must not outvote the real one.
func scenarioToken(header []byte) string {
	if len(header) == 0 {
		return ""
	}
	limit := len(header)
	if limit > 400000 {
		limit = 400000
	}
	counts := map[string]int{}
	for _, m := range scenarioTokenRe.FindAll(header[:limit], -1) {
		counts[strings.TrimSpace(string(m))]++
	}
	best, bestN := "", 0
	for tok, n := range counts {
		// Ties break on the token text so the result is deterministic across runs.
		if n > bestN || (n == bestN && tok < best) {
			best, bestN = tok, n
		}
	}
	return best
}

// RegistryPlayer is one seat in a classified game.
type RegistryPlayer struct {
	Number    int    `json:"number"`
	Name      string `json:"name,omitempty"`
	ProfileID int    `json:"profile_id,omitempty"`
	Team      string `json:"team,omitempty"`
	Civ       int    `json:"civ,omitempty"`
}

// RegistryEntry is one classified replay.
type RegistryEntry struct {
	Path      string           `json:"path"`
	GameID    string           `json:"game_id"`
	Scenario  ScenarioIdentity `json:"scenario"`
	DurationS int              `json:"duration_s,omitempty"`
	Players   []RegistryPlayer `json:"players,omitempty"`
	Source    string           `json:"source,omitempty"`
	Admit     bool             `json:"admit"`
	Reason    string           `json:"reason"`
	DuplicateOf string         `json:"duplicate_of,omitempty"`
}

// Registry is the admission decision for a whole replay tree.
type Registry struct {
	Scenario string          `json:"scenario"`
	Summary  RegistrySummary `json:"summary"`
	Entries  []RegistryEntry `json:"entries"`
}

type RegistrySummary struct {
	Scanned    int `json:"scanned"`
	Ladder     int `json:"ladder"`
	Admitted   int `json:"admitted"`
	Duplicates int `json:"duplicates"`
	Rejected   int `json:"rejected"`
	TooFew     int `json:"too_few_players"`
}

// BuildRegistry classifies replays and resolves duplicates.
//
// The same match reaches us from several player POVs and from both the local save
// archive and the crawled MS-API pulls, so admitting on filename alone would multiply
// every game. Two replays are the same match when their profile-id roster and their
// duration (bucketed to 10 s, absorbing POV recording jitter) agree.
func BuildRegistry(paths []string, progress func(path string, id ScenarioIdentity)) *Registry {
	return BuildRegistryWithWorkers(paths, 0, progress)
}

// BuildRegistryWithWorkers classifies concurrently (workers <= 0 means NumCPU).
//
// Classification is parallel because each replay is independent, but DEDUPLICATION is a
// second, sequential pass in stable path order. Resolving duplicates inside the workers
// would make "which copy of this match got admitted" depend on scheduling, so the same
// corpus could produce different registries run to run.
func BuildRegistryWithWorkers(paths []string, workers int, progress func(path string, id ScenarioIdentity)) *Registry {
	reg := &Registry{Scenario: LadderToken + " (embedded token + trigger band)"}

	workers = workersFor(RegistryWorkerMiB, workers)
	if workers > len(paths) {
		workers = len(paths)
	}
	if workers < 1 {
		return reg
	}

	entries := make([]RegistryEntry, len(paths))
	var wg sync.WaitGroup
	var mu sync.Mutex
	idxCh := make(chan int)

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for idx := range idxCh {
				p := paths[idx]
				e := RegistryEntry{Path: p, GameID: GameIDForPath(p), Source: sourceOf(p)}
				e.Scenario = IdentifyScenario(p)

				if e.Scenario.Ladder {
					players, dur, err := rosterOf(p)
					switch {
					case err != nil:
						e.Reason = "roster unavailable: " + err.Error()
					case len(players) < 4:
						e.Players, e.DurationS = players, dur
						e.Reason = fmt.Sprintf("only %d resolved players (need 4+)", len(players))
					default:
						e.Players, e.DurationS = players, dur
					}
				} else {
					e.Reason = e.Scenario.Reason
				}
				entries[idx] = e

				if progress != nil {
					mu.Lock()
					progress(p, e.Scenario)
					mu.Unlock()
				}
			}
		}()
	}
	for i := range paths {
		idxCh <- i
	}
	close(idxCh)
	wg.Wait()

	// Sequential admission pass -- deterministic, and independent of worker scheduling.
	//
	// Order matters: when one match is present as both a local save and an MS-API pull,
	// whichever copy is admitted FIRST becomes canonical and the other is marked its
	// duplicate. Prefer the copy carrying the Microsoft match id, because that id is
	// stable across sources and re-pulls while a local save name is not.
	order := make([]int, len(entries))
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(a, b int) bool {
		ia, ib := entries[order[a]], entries[order[b]]
		ma, mb := HasMSGameID(ia.GameID), HasMSGameID(ib.GameID)
		if ma != mb {
			return ma
		}
		return ia.GameID < ib.GameID
	})

	seen := map[string]string{} // dedup signature -> winning game id
	for _, idx := range order {
		e := entries[idx]
		reg.Summary.Scanned++
		switch {
		case !e.Scenario.Ladder:
			reg.Summary.Rejected++
		case e.Reason != "" && strings.HasPrefix(e.Reason, "roster unavailable"):
			reg.Summary.Rejected++
		case len(e.Players) < 4:
			reg.Summary.TooFew++
		default:
			sig := dedupSignature(e.Players, e.DurationS)
			if first, dup := seen[sig]; dup {
				e.Reason = "duplicate of " + first
				e.DuplicateOf = first
				reg.Summary.Duplicates++
			} else {
				seen[sig] = e.GameID
				e.Admit = true
				e.Reason = "official V292"
				reg.Summary.Admitted++
			}
		}
		reg.Summary.Ladder += boolToInt(e.Scenario.Ladder)
		reg.Entries = append(reg.Entries, e)
	}

	sort.Slice(reg.Entries, func(i, j int) bool { return reg.Entries[i].GameID < reg.Entries[j].GameID })
	return reg
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// rosterOf resolves the human seats and the game duration.
func rosterOf(path string) ([]RegistryPlayer, int, error) {
	rec, err := replay.Open(path)
	if err != nil {
		return nil, 0, err
	}
	var out []RegistryPlayer
	for _, s := range rec.Players {
		// Non-positive profile ids are unresolved or AI seats, not people; admitting
		// them would let a phantom player accumulate games across unrelated matches.
		if s.ProfileID <= 0 {
			continue
		}
		out = append(out, RegistryPlayer{
			Number:    s.Number,
			Name:      s.Name,
			ProfileID: s.ProfileID,
			// Reuse the extractor's own team rule so the registry can never disagree
			// with the performance rows about which side a player was on.
			Team: cbaTeam(s.Number),
			Civ:  s.Civ,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Number < out[j].Number })

	dur := 0
	if profile, err := replay.BuildPlayerProfile(path, replay.PlayerProfileOptions{}); err == nil {
		dur = profile.EventCounts.DurationMS / 1000
	}
	return out, dur, nil
}

// dedupSignature identifies a MATCH rather than a FILE: the roster plus a coarse
// duration. Duration is bucketed because different POV recordings of one match stop a
// few seconds apart.
func dedupSignature(players []RegistryPlayer, durationS int) string {
	ids := make([]string, 0, len(players))
	for _, p := range players {
		ids = append(ids, fmt.Sprintf("%d", p.ProfileID))
	}
	sort.Strings(ids)
	return strings.Join(ids, ",") + "|" + fmt.Sprintf("%d", durationS/10)
}

// sourceOf distinguishes MS-API pulls from local saves. Matched per path SEGMENT
// rather than as a substring: paths are often relative ("crawl/x.zip"), so testing for
// "/crawl/" silently labelled every crawled game as local.
func sourceOf(path string) string {
	for _, seg := range strings.Split(filepath.ToSlash(path), "/") {
		if seg == "crawl" {
			return "crawl"
		}
	}
	return "local"
}

// Admitted returns the paths cleared for ingest.
func (r *Registry) Admitted() []string {
	var out []string
	for _, e := range r.Entries {
		if e.Admit {
			out = append(out, e.Path)
		}
	}
	return out
}

// LoadRegistry reads a registry written by WriteRegistry.
func LoadRegistry(path string) (*Registry, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var reg Registry
	if err := json.Unmarshal(raw, &reg); err != nil {
		return nil, err
	}
	return &reg, nil
}

// WriteRegistry persists the admission decisions, including rejections and their
// reasons -- a registry that recorded only what it admitted could not be audited.
func WriteRegistry(w io.Writer, reg *Registry) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", " ")
	return enc.Encode(reg)
}
