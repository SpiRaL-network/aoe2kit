package cba

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
)

// Corpus is the durable store behind the CBA ladder: one GameRecord per ingested
// replay, persisted as JSONL. JSONL rather than SQLite deliberately -- AoE2Kit has
// no external dependencies and that portability is worth more than ad-hoc SQL for a
// corpus this size (hundreds of games, thousands of rows, all loaded and grouped
// in memory anyway).
type Corpus struct {
	Games []GameRecord
}

// GameRecord is one ingested replay: the identity we matched it by, plus the
// extractor's per-player performance rows verbatim. The rows are NOT reshaped on ingest -- the
// extractor's schema is the corpus schema, so a change there flows through without
// a migration.
type GameRecord struct {
	GameID     string           `json:"game_id"`
	Path       string           `json:"path"`
	Scenario   string           `json:"scenario,omitempty"`
	DurationS  int              `json:"duration_s,omitempty"`
	IngestedBy string           `json:"ingested_by,omitempty"`
	Rows       []PerformanceRow `json:"performances"`
	Warnings   []string         `json:"warnings,omitempty"`
}

// LoadCorpus reads a JSONL corpus. A missing file is an empty corpus, not an error --
// ingest is expected to bootstrap it.
func LoadCorpus(path string) (*Corpus, error) {
	c := &Corpus{}
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return c, nil
		}
		return nil, err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 1<<20), 1<<26) // game records carry ~8 fat rows
	line := 0
	for sc.Scan() {
		line++
		raw := strings.TrimSpace(sc.Text())
		if raw == "" {
			continue
		}
		var g GameRecord
		if err := json.Unmarshal([]byte(raw), &g); err != nil {
			return nil, fmt.Errorf("corpus %s line %d: %w", path, line, err)
		}
		c.Games = append(c.Games, g)
	}
	return c, sc.Err()
}

// Save writes the corpus atomically -- a torn corpus file would silently truncate
// the ladder, and the failure mode (fewer games, plausible numbers) is invisible.
func (c *Corpus) Save(path string) error {
	sort.Slice(c.Games, func(i, j int) bool { return c.Games[i].GameID < c.Games[j].GameID })

	tmp := path + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	w := bufio.NewWriter(f)
	enc := json.NewEncoder(w)
	for i := range c.Games {
		if err := enc.Encode(&c.Games[i]); err != nil {
			f.Close()
			os.Remove(tmp)
			return err
		}
	}
	if err := w.Flush(); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, path)
}

// Has reports whether a game is already ingested, so ingest is idempotent and a
// re-run over the whole replay tree costs nothing for games already present.
func (c *Corpus) Has(gameID string) bool {
	for i := range c.Games {
		if c.Games[i].GameID == gameID {
			return true
		}
	}
	return false
}

// Upsert replaces an existing record with the same GameID, else appends.
func (c *Corpus) Upsert(g GameRecord) {
	for i := range c.Games {
		if c.Games[i].GameID == g.GameID {
			c.Games[i] = g
			return
		}
	}
	c.Games = append(c.Games, g)
}

// Rows flattens every performance row across the corpus, stamping GameID onto rows
// that lack it so downstream grouping never has to carry the parent record.
func (c *Corpus) Rows() []PerformanceRow {
	var out []PerformanceRow
	for i := range c.Games {
		for _, r := range c.Games[i].Rows {
			if r.GameID == "" {
				r.GameID = c.Games[i].GameID
			}
			out = append(out, r)
		}
	}
	return out
}

// IngestOptions controls a corpus build.
type IngestOptions struct {
	// Force re-extracts games already present instead of skipping them.
	Force bool
	// Limit caps how many NEW games are ingested this run (0 = no cap).
	Limit int
	// Workers sets extraction concurrency. 0 means "fit it in available memory" --
	// see workersFor. Extraction peaks near 900 MiB per replay, so this is bounded by
	// RAM, not by core count; setting it to NumCPU on a small machine will swap-storm.
	Workers int
	// Progress, if set, is called per replay with the outcome. It may be invoked from
	// multiple goroutines, so implementations must be safe to call concurrently.
	Progress func(path, status string, err error)
}

// IngestResult reports what a run did.
type IngestResult struct {
	Scanned int `json:"scanned"`
	Added   int `json:"added"`
	Updated int `json:"updated"`
	Skipped int `json:"skipped"`
	Failed  int `json:"failed"`
	// Workers records the concurrency actually used. Reported because it is chosen
	// from available memory rather than fixed, and an invisible default is exactly
	// how this job once ate a machine's swap.
	Workers int `json:"workers"`
}

// Ingest walks replay paths, extracts performance rows, and folds them into the
// corpus. Extraction failure on one replay is recorded and skipped -- a single
// unreadable file must never abort a long corpus build.
func Ingest(c *Corpus, paths []string, opts IngestOptions) IngestResult {
	var res IngestResult

	// Decide what to extract BEFORE spawning workers, so the skip/limit rules stay
	// deterministic. Applying a limit inside concurrent workers would make the set of
	// ingested games depend on scheduling order.
	type job struct {
		path   string
		gameID string
		exists bool
	}
	var jobs []job
	newCount := 0
	for _, p := range paths {
		res.Scanned++
		gameID := GameIDForPath(p)
		exists := c.Has(gameID)
		if exists && !opts.Force {
			res.Skipped++
			if opts.Progress != nil {
				opts.Progress(p, "skip", nil)
			}
			continue
		}
		if !exists {
			if opts.Limit > 0 && newCount >= opts.Limit {
				res.Skipped++
				continue
			}
			newCount++
		}
		jobs = append(jobs, job{path: p, gameID: gameID, exists: exists})
	}

	workers := workersFor(ExtractWorkerMiB, opts.Workers)
	if workers > len(jobs) {
		workers = len(jobs)
	}
	if workers < 1 {
		return res
	}
	res.Workers = workers

	type outcome struct {
		job    job
		record GameRecord
		err    error
	}
	jobCh := make(chan job)
	outCh := make(chan outcome)

	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobCh {
				report, err := BuildPerformance(j.path)
				if err != nil {
					outCh <- outcome{job: j, err: err}
					continue
				}
				g := GameRecord{
					GameID:     j.gameID,
					Path:       j.path,
					DurationS:  report.Summary.DurationMS / 1000,
					IngestedBy: report.Method,
					Rows:       report.Rows,
					Warnings:   report.Warnings,
				}
				for i := range g.Rows {
					g.Rows[i].GameID = j.gameID
				}
				outCh <- outcome{job: j, record: g}
			}
		}()
	}
	go func() {
		for _, j := range jobs {
			jobCh <- j
		}
		close(jobCh)
	}()
	go func() {
		wg.Wait()
		close(outCh)
	}()

	// The corpus is mutated only here, on the collecting goroutine, so Upsert needs no
	// lock and the corpus ordering stays independent of worker scheduling.
	for o := range outCh {
		if o.err != nil {
			res.Failed++
			if opts.Progress != nil {
				opts.Progress(o.job.path, "fail", o.err)
			}
			continue
		}
		c.Upsert(o.record)
		if o.job.exists {
			res.Updated++
			if opts.Progress != nil {
				opts.Progress(o.job.path, "update", nil)
			}
		} else {
			res.Added++
			if opts.Progress != nil {
				opts.Progress(o.job.path, "add", nil)
			}
		}
	}
	return res
}

// msGameIDRe matches a filename that is ENTIRELY a match id, optionally behind the
// game's own export prefix: "492398692.zip", "AgeIIDE_Replay_484043465.aoe2record".
// Anchored on purpose -- a local save name like
// "037__MP Replay v101.103.48987.0 @2026.07.14 210732 (4)" is full of digit runs and
// an unanchored search would happily pick one of them as a match id.
var msGameIDRe = regexp.MustCompile(`^(?:AgeIIDE_Replay_)?(\d{6,12})$`)

// GameIDForPath derives a game id from the replay filename, preferring the Microsoft
// match id when the name carries one.
//
// The same match reaches us under completely different filenames -- a local save
// ("037__MP Replay v101...") and an MS-API pull ("492398692.zip") -- so a bare basename
// is NOT a stable identity: re-pulling a match under a new local name would mint a
// second id for a game already in the corpus. The MS id is stable across sources and
// re-pulls, so it wins whenever it is present.
func GameIDForPath(p string) string {
	base := filepath.Base(p)
	base = strings.TrimSuffix(base, filepath.Ext(base))
	if m := msGameIDRe.FindStringSubmatch(base); m != nil {
		return m[1]
	}
	return base
}

// HasMSGameID reports whether an id is a Microsoft match id rather than a local
// filename. Used to pick the canonical copy when one match arrives from both sources.
func HasMSGameID(gameID string) bool {
	return msGameIDRe.MatchString(gameID)
}
