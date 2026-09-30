package cba

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"aoe2kit/pkg/aoems"
)

// The archiver exists to win a race we otherwise lose: the match-replay API retains a
// game for roughly two to three weeks, and a laptop that is only on some of the time
// cannot reliably pull inside that window. Run from an always-on host, this turns
// "whatever we happened to catch" into a permanent archive.
//
// It is deliberately unhurried and deliberately bounded. An unattended fetch loop
// against someone else's public API is exactly the thing that becomes abusive by
// accident, so: one request at a time, a real delay between them, a hard ceiling that
// must be stated explicitly, and a circuit breaker that stops on repeated failure.
const (
	// DefaultArchiveDelay paces requests. Slow on purpose -- there is no deadline here
	// beyond the retention window, and politeness costs us nothing.
	DefaultArchiveDelay = 4 * time.Second

	// MaxConsecutiveFailures trips the circuit breaker. Repeated failures mean
	// something changed (auth, rate limit, outage); hammering through them is how a
	// well-behaved client turns into a bad one.
	MaxConsecutiveFailures = 5
)

// QueueItem is one match to archive. Discovery happens elsewhere -- Cloudflare blocks
// non-browser clients from the match-listing site, so ids are produced by a browser
// session and handed here as data.
type QueueItem struct {
	GameID    string `json:"game_id"`
	ProfileID string `json:"profile_id"`
	// Note is free-text provenance from discovery (e.g. which player's history it came
	// from). Carried through to the manifest so an archived game can be traced back.
	Note string `json:"note,omitempty"`
}

// ArchiveRecord is the durable outcome for one queue item. Written for rejects too:
// without them the archiver would re-download every non-CBA game on every run.
type ArchiveRecord struct {
	GameID     string `json:"game_id"`
	ProfileID  string `json:"profile_id"`
	FetchedAt  string `json:"fetched_at"`
	Bytes      int64  `json:"bytes,omitempty"`
	Path       string `json:"path,omitempty"`
	Admitted   bool   `json:"admitted"`
	Token      string `json:"token,omitempty"`
	Triggers   int    `json:"trigger_count,omitempty"`
	Reason     string `json:"reason"`
	Note       string `json:"note,omitempty"`
}

// ArchiveOptions configures a run.
type ArchiveOptions struct {
	// OutDir receives admitted CBA replays.
	OutDir string
	// RejectDir receives everything else. Non-CBA downloads are MOVED here rather than
	// deleted -- the archiver should not be in the business of irreversible cleanup,
	// and a reject pile is easy to inspect or purge by hand.
	RejectDir string
	// Max is a hard ceiling on fetches this run. Required: there is no safe default
	// for "how much of someone else's API should I pull unattended".
	Max int
	// Delay between requests (0 = DefaultArchiveDelay).
	Delay time.Duration
	// Progress is called after each item.
	Progress func(rec ArchiveRecord)
}

// ArchiveResult summarises a run.
type ArchiveResult struct {
	Considered int    `json:"considered"`
	Fetched    int    `json:"fetched"`
	Admitted   int    `json:"admitted"`
	Rejected   int    `json:"rejected"`
	Skipped    int    `json:"skipped"`
	Failed     int    `json:"failed"`
	StoppedBy  string `json:"stopped_by,omitempty"`
}

// Archive fetches queued matches, keeps the CBA ones, and records every outcome.
//
// Classification happens AFTER download because scenario identity lives inside the
// replay -- there is no way to know a match is CBA without its bytes. Classification is
// the light path (open + header token + trigger count), not the heavy extraction, so it
// is cheap enough to run on a shared host.
func Archive(queue []QueueItem, seen map[string]bool, client *aoems.Client, opts ArchiveOptions) (ArchiveResult, []ArchiveRecord) {
	var res ArchiveResult
	var records []ArchiveRecord

	if opts.Delay <= 0 {
		opts.Delay = DefaultArchiveDelay
	}
	if opts.RejectDir == "" {
		opts.RejectDir = filepath.Join(opts.OutDir, "rejected")
	}

	consecutiveFailures := 0
	for _, item := range queue {
		res.Considered++

		if seen[item.GameID] {
			res.Skipped++
			continue
		}
		if res.Fetched >= opts.Max {
			res.StoppedBy = fmt.Sprintf("reached --max %d", opts.Max)
			break
		}
		if consecutiveFailures >= MaxConsecutiveFailures {
			res.StoppedBy = fmt.Sprintf("circuit breaker: %d consecutive failures", consecutiveFailures)
			break
		}

		// Pace BEFORE each request rather than after, so an early break never leaves
		// the next run starting with a burst.
		if res.Fetched > 0 {
			time.Sleep(opts.Delay)
		}

		rec := ArchiveRecord{
			GameID:    item.GameID,
			ProfileID: item.ProfileID,
			Note:      item.Note,
			FetchedAt: time.Now().UTC().Format(time.RFC3339),
		}

		report, err := client.FetchReplay(aoems.ReplayFetchOptions{
			GameID:    item.GameID,
			ProfileID: item.ProfileID,
			OutDir:    opts.OutDir,
		})
		if err != nil {
			consecutiveFailures++
			res.Failed++
			rec.Reason = "fetch failed: " + err.Error()
			records = append(records, rec)
			if opts.Progress != nil {
				opts.Progress(rec)
			}
			continue
		}
		consecutiveFailures = 0
		res.Fetched++
		rec.Bytes = report.Bytes
		rec.Path = report.Path

		id := IdentifyScenario(report.Path)
		rec.Token, rec.Triggers, rec.Reason = id.Token, id.TriggerCount, id.Reason
		rec.Admitted = id.Ladder

		if id.Ladder {
			res.Admitted++
		} else {
			res.Rejected++
			if moved, err := moveToReject(report.Path, opts.RejectDir); err == nil {
				rec.Path = moved
			} else {
				rec.Reason += " (reject move failed: " + err.Error() + ")"
			}
		}

		records = append(records, rec)
		if opts.Progress != nil {
			opts.Progress(rec)
		}
	}

	if res.StoppedBy == "" {
		res.StoppedBy = "queue exhausted"
	}
	return res, records
}

// moveToReject relocates a non-CBA download out of the archive. Move, never delete --
// an unattended job should not destroy data it just spent a request acquiring.
func moveToReject(path, rejectDir string) (string, error) {
	if err := os.MkdirAll(rejectDir, 0o755); err != nil {
		return "", err
	}
	dest := filepath.Join(rejectDir, filepath.Base(path))
	if err := os.Rename(path, dest); err != nil {
		return "", err
	}
	return dest, nil
}

// LoadQueue reads discovery output: either a JSON array or one JSON object per line.
func LoadQueue(path string) ([]QueueItem, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" {
		return nil, nil
	}

	if strings.HasPrefix(trimmed, "[") {
		var items []QueueItem
		if err := json.Unmarshal([]byte(trimmed), &items); err != nil {
			return nil, err
		}
		return items, nil
	}

	var items []QueueItem
	for i, line := range strings.Split(trimmed, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var item QueueItem
		if err := json.Unmarshal([]byte(line), &item); err != nil {
			return nil, fmt.Errorf("queue line %d: %w", i+1, err)
		}
		items = append(items, item)
	}
	return items, nil
}

// LoadManifest returns the set of already-processed game ids, so a re-run never
// re-requests a match the API has already answered for -- including rejects.
func LoadManifest(path string) (map[string]bool, error) {
	seen := map[string]bool{}
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return seen, nil
		}
		return nil, err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 1<<16), 1<<22)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var rec ArchiveRecord
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			continue // a corrupt line must not hide the rest of the manifest
		}
		if rec.GameID != "" {
			seen[rec.GameID] = true
		}
	}
	return seen, sc.Err()
}

// AppendManifest appends records. Append-only by design: the manifest is the record of
// what we asked the API for, and rewriting it would lose that history.
func AppendManifest(path string, records []ArchiveRecord) error {
	if len(records) == 0 {
		return nil
	}
	if dir := filepath.Dir(path); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	enc := json.NewEncoder(w)
	for i := range records {
		if err := enc.Encode(&records[i]); err != nil {
			return err
		}
	}
	return w.Flush()
}
