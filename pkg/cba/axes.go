package cba

import (
	"sort"
)

// CBA skill is multi-axis and the axes are orthogonal -- never collapse them to one
// number. Every axis is scored as CONTRIBUTION SHARE WITHIN A GAME, so a rider on a
// strong team earns little regardless of the result. That is what makes stacking
// self-defeating WITHOUT needing manual stacker tags.
//
// "Genuinely good = multiple positive correlation" (chrae): independent axes AGREE.
//
//	ECO     share of your team's production channels, vs an equal share
//	TEMPO   how early you open the floodgates, vs the others in that game
//	COMBAT  army preservation, normalised within your OWN TEAM
//	CONDUCT quit behaviour, side tendency (descriptive)
const (
	// MinProdForCombat floors the loss-rate denominator. Without it a player who
	// produced a handful of units posts an absurd preservation ratio and dominates
	// the axis on noise.
	MinProdForCombat = 100

	ecoHighThreshold    = 1.10
	combatHighThreshold = 1.05
	ecoLowThreshold     = 0.90
	combatLowThreshold  = 0.95
	riderWinThreshold   = 65
)

// KnownAnchors are players chrae vouches for as genuinely strong, used to sanity-check
// that the model agrees with human ground truth. Ratings are NOT pinned to them here --
// the axes are contribution-based and must stand on their own.
var KnownAnchors = []string{
	"Monkey Boy", "Cengo", "man23457689", "[xCs]Bek", "[xCs]Vangelis", "nW | Sgt.Pepper",
}

// KnownStackers are players chrae has observed stacking lobbies. Stacking is ORTHOGONAL
// to skill (man23457689 is elite and stacks); this tag exists to validate that the
// contribution axes neutralise a stacker's inflated win rate, not to penalise them.
var KnownStackers = []string{"Aster"}

// AxesOptions configures scoring.
type AxesOptions struct {
	// MinGames drops players too thin to score (default 3).
	MinGames int
	// Names maps profile id -> display name.
	Names map[int]string
	// Anchors / Stackers override the package defaults when non-nil.
	Anchors  []string
	Stackers []string
}

// PlayerAxes is one player's multi-axis profile.
type PlayerAxes struct {
	ProfileID int     `json:"profile_id"`
	Name      string  `json:"name,omitempty"`
	Games     int     `json:"games"`
	Eco       float64 `json:"eco,omitempty"`
	HasEco    bool    `json:"-"`
	Tempo     float64 `json:"tempo,omitempty"`
	HasTempo  bool    `json:"-"`
	Combat    float64 `json:"combat,omitempty"`
	HasCombat bool    `json:"-"`
	Verdict   string  `json:"verdict,omitempty"`

	ProdAvg    float64 `json:"prod_avg"`
	APM        float64 `json:"apm,omitempty"`
	WinPct     int     `json:"win_pct"`
	Decided    int     `json:"decided"`
	EarlyQuit  int     `json:"early_quit_pct"`
	EarlySlot  int     `json:"early_slot_pct"`
	IsAnchor   bool    `json:"anchor,omitempty"`
	IsStacker  bool    `json:"stacker,omitempty"`
	EcoSamples int     `json:"eco_samples"`
	CmbSamples int     `json:"combat_samples"`
	// CombatCivAdjusted counts how many combat samples had a trusted civ baseline
	// applied. Reported so a profile never hides that some rows went uncorrected.
	CombatCivAdjusted int `json:"combat_civ_adjusted"`
}

type axisAcc struct {
	eco, tempo, combat []float64
	prod, apm          []float64
	quit               []float64
	slots              []int
	games, won, dec    int
	combatCivAdjusted  int
}

// ComputeAxes scores every player in the corpus.
func ComputeAxes(c *Corpus, opts AxesOptions) []PlayerAxes {
	if opts.MinGames <= 0 {
		opts.MinGames = 3
	}
	anchors := toSet(pick(opts.Anchors, KnownAnchors))
	stackers := toSet(pick(opts.Stackers, KnownStackers))

	// Drop unresolved/AI slots ONCE, before grouping. Filtering later is not equivalent:
	// these rows would still sit in the team totals and dilute every real player's
	// contribution share.
	var rows []PerformanceRow
	for _, r := range c.Rows() {
		if r.ProfileID > 0 {
			rows = append(rows, r)
		}
	}

	civBaseline := civLossBaselines(rows)

	durations := map[string]int{}
	for i := range c.Games {
		durations[c.Games[i].GameID] = c.Games[i].DurationS
	}

	// group for share computations
	byGameTeam := map[[2]string][]PerformanceRow{}
	byGame := map[string][]PerformanceRow{}
	for _, r := range rows {
		byGameTeam[[2]string{r.GameID, r.Team}] = append(byGameTeam[[2]string{r.GameID, r.Team}], r)
		byGame[r.GameID] = append(byGame[r.GameID], r)
	}

	acc := map[int]*axisAcc{}
	for _, r := range rows {
		// Non-positive ids are unresolved/AI slots, not people. They must be dropped
		// rather than grouped: every such row collapses into one phantom "player" that
		// accumulates games across unrelated matches and lands at the top of the axes.
		if r.ProfileID <= 0 {
			continue
		}
		a := acc[r.ProfileID]
		if a == nil {
			a = &axisAcc{}
			acc[r.ProfileID] = a
		}
		a.games++
		if r.Won != nil {
			a.dec++
			a.won += *r.Won
		}
		a.slots = append(a.slots, r.Slot)
		a.prod = append(a.prod, float64(deref(r.ProdBuildings)))
		if r.APM != nil {
			a.apm = append(a.apm, *r.APM)
		}

		team := byGameTeam[[2]string{r.GameID, r.Team}]

		// ECO -- share of the team's production, normalised so an equal share is 1.00.
		total := 0
		for _, q := range team {
			total += deref(q.ProdBuildings)
		}
		if total > 0 && len(team) > 1 {
			a.eco = append(a.eco, float64(deref(r.ProdBuildings))/float64(total)*float64(len(team)))
		}

		// TEMPO -- rank of first production build among builders in that game (1.0 = earliest).
		if r.FirstProdBuildS != nil {
			var builders []PerformanceRow
			for _, q := range byGame[r.GameID] {
				if q.FirstProdBuildS != nil {
					builders = append(builders, q)
				}
			}
			if len(builders) > 1 {
				sort.Slice(builders, func(i, j int) bool {
					return *builders[i].FirstProdBuildS < *builders[j].FirstProdBuildS
				})
				for idx, q := range builders {
					if q.ProfileID == r.ProfileID {
						a.tempo = append(a.tempo, 1.0-float64(idx)/float64(len(builders)-1))
						break
					}
				}
			}
		}

		// CONDUCT -- early quit is a resign in the first half of the game.
		if deref(r.Resigned) != 0 {
			dur := durations[r.GameID]
			q := 0.0
			if dur > 0 && deref(r.ResignTimeS) < dur/2 {
				q = 1.0
			}
			a.quit = append(a.quit, q)
		}

		// COMBAT -- army preservation, NOT kills: the extractor explicitly refuses to
		// claim kill attribution. Normalise within the player's OWN TEAM, never the
		// whole game: teammates share the same game state and opponents, so a
		// team-relative comparison strips out the "our side was dominating" confound.
		// Game-normalised preservation just re-measures winning -- it credited a known
		// stacker at 1.49; team-normalised drops him to ~1.05.
		// Use loss RATE (lost/produced, lower is better) so the index cannot go negative.
		// Civ-adjust BEFORE comparing to teammates. CBA is a data mod that spawns armies
		// at wildly different rates per civ -- decoded from the scenario's own triggers,
		// Malay run 120 units every 6 s (20/s) while Khmer run 30 every 15 s (2/s), a 10x
		// design spread. Measured across 42 civs, spawn rate correlates -0.45 with observed
		// loss rate, and civ loss-rate baselines span 0.671..0.906 -- a ~15% spread against
		// a 5% verdict threshold. Comparing a Malay player's raw loss rate to a Khmer
		// teammate's therefore scores the civ, not the player.
		//
		// The baseline is EMPIRICAL rather than derived from the decoded spawn rate on
		// purpose: spawn rate explains only ~20% of the civ variance (r^2), while an
		// observed per-civ baseline absorbs every civ effect at once -- spawn rate, unit
		// durability, and the rest.
		if mine, ok := lossRateCivAdjusted(r, civBaseline); ok {
			var peers []float64
			for _, q := range team {
				if lr, ok := lossRateCivAdjusted(q, civBaseline); ok {
					peers = append(peers, lr)
				}
			}
			if len(peers) > 1 && mine > 0 {
				tm := mean(peers)
				if tm > 0 {
					a.combat = append(a.combat, tm/mine)
					if _, adjusted := civBaseline[r.Civ]; adjusted {
						a.combatCivAdjusted++
					}
				}
			}
		}
	}

	var out []PlayerAxes
	for pid, a := range acc {
		if a.games < opts.MinGames {
			continue
		}
		p := PlayerAxes{
			ProfileID:  pid,
			Name:       opts.Names[pid],
			Games:      a.games,
			Decided:    a.dec,
			ProdAvg:    round(mean(a.prod), 1),
			EcoSamples: len(a.eco),
			CmbSamples: len(a.combat),
			CombatCivAdjusted: a.combatCivAdjusted,
		}
		if len(a.eco) > 0 {
			p.Eco, p.HasEco = round(mean(a.eco), 2), true
		}
		if len(a.tempo) > 0 {
			p.Tempo, p.HasTempo = round(mean(a.tempo), 2), true
		}
		if len(a.combat) > 0 {
			p.Combat, p.HasCombat = round(mean(a.combat), 2), true
		}
		if len(a.apm) > 0 {
			p.APM = round(mean(a.apm), 0)
		}
		if a.dec > 0 {
			p.WinPct = int(round(100*float64(a.won)/float64(a.dec), 0))
		}
		if len(a.quit) > 0 {
			p.EarlyQuit = int(round(100*mean(a.quit), 0))
		}
		if len(a.slots) > 0 {
			early := 0
			for _, s := range a.slots {
				if s <= 4 {
					early++
				}
			}
			p.EarlySlot = int(round(100*float64(early)/float64(len(a.slots)), 0))
		}
		if p.Name != "" {
			p.IsAnchor = anchors[p.Name]
			p.IsStacker = stackers[p.Name]
		}
		p.Verdict = verdictFor(p)
		out = append(out, p)
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].Eco != out[j].Eco {
			return out[i].Eco > out[j].Eco
		}
		return out[i].ProfileID < out[j].ProfileID
	})
	return out
}

// verdictFor applies "genuinely good = multiple positive correlation": independent
// axes must AGREE. A rider wins a lot while carrying neither axis.
func verdictFor(p PlayerAxes) string {
	if !p.HasEco || !p.HasCombat {
		return ""
	}
	hiE, hiC := p.Eco >= ecoHighThreshold, p.Combat >= combatHighThreshold
	loE, loC := p.Eco < ecoLowThreshold, p.Combat < combatLowThreshold
	switch {
	case hiE && hiC:
		return "CONVERGENT (carries both)"
	case hiE:
		return "specialist (eco)"
	case hiC:
		return "specialist (combat)"
	case loE && loC && p.WinPct >= riderWinThreshold:
		return "RIDER? (wins, carries neither)"
	}
	return "-"
}

// lossRate returns lost/produced when the sample is big enough to mean anything.
func lossRate(r PerformanceRow) (float64, bool) {
	if r.UnitsProduced == nil || r.UnitsLost == nil {
		return 0, false
	}
	if *r.UnitsProduced < MinProdForCombat {
		return 0, false
	}
	return float64(*r.UnitsLost) / float64(*r.UnitsProduced), true
}

// MinCivSamplesForBaseline is how many scoreable rows a civ needs before its baseline is
// trusted. Below this the civ is left UNADJUSTED rather than corrected with a noisy
// number -- and never silently folded into a global average, which would quietly move
// rare-civ players for no measured reason.
const MinCivSamplesForBaseline = 8

// civLossBaselines computes each civ's mean loss rate across the corpus.
//
// Returns only civs that clear MinCivSamplesForBaseline; callers treat a missing civ as
// "no adjustment available", which is honest about coverage instead of pretending to a
// correction we cannot support.
func civLossBaselines(rows []PerformanceRow) map[string]float64 {
	byCiv := map[string][]float64{}
	for _, r := range rows {
		if r.Civ == "" {
			continue
		}
		if lr, ok := lossRate(r); ok {
			byCiv[r.Civ] = append(byCiv[r.Civ], lr)
		}
	}
	out := make(map[string]float64, len(byCiv))
	for civ, xs := range byCiv {
		if len(xs) < MinCivSamplesForBaseline {
			continue
		}
		if m := mean(xs); m > 0 {
			out[civ] = m
		}
	}
	return out
}

// lossRateCivAdjusted divides a player's loss rate by their civ's corpus baseline, so
// 1.0 means "typical for this civ" and the downstream team comparison measures the
// player rather than the civ they drew.
//
// A civ without a trusted baseline passes through UNADJUSTED. That is deliberate: mixing
// adjusted and raw values in one team comparison is imperfect, but it is visible (the
// per-player combat_civ_adjusted count reports it) and strictly better than either
// dropping those rows or inventing a baseline for them.
func lossRateCivAdjusted(r PerformanceRow, baselines map[string]float64) (float64, bool) {
	raw, ok := lossRate(r)
	if !ok {
		return 0, false
	}
	if base, found := baselines[r.Civ]; found && base > 0 {
		return raw / base, true
	}
	return raw, true
}

func deref(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}

func mean(xs []float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	s := 0.0
	for _, x := range xs {
		s += x
	}
	return s / float64(len(xs))
}

func round(v float64, places int) float64 {
	m := 1.0
	for i := 0; i < places; i++ {
		m *= 10
	}
	if v < 0 {
		return float64(int(v*m-0.5)) / m
	}
	return float64(int(v*m+0.5)) / m
}

func toSet(xs []string) map[string]bool {
	m := make(map[string]bool, len(xs))
	for _, x := range xs {
		m[x] = true
	}
	return m
}

func pick(override, def []string) []string {
	if override != nil {
		return override
	}
	return def
}
