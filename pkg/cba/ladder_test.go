package cba

import (
	"strings"
	"testing"
)

// The MS match id is the corpus's identity backbone: the same match arrives as a local
// save and as an API pull under unrelated names, and an unstable id would let one match
// be ingested twice. The regex is ANCHORED for a reason -- local save names are full of
// digit runs (version numbers, dates) that an unanchored search would happily mine.
func TestGameIDForPathPrefersAnchoredMSID(t *testing.T) {
	cases := []struct {
		path string
		want string
		ms   bool
	}{
		{"crawl/492398692.zip", "492398692", true},
		{"crawl/e_484043465/AgeIIDE_Replay_484043465.aoe2record", "484043465", true},
		{"/abs/path/AgeIIDE_Replay_484043465.aoe2record", "484043465", true},

		// Local save names must NOT yield a match id -- "101", "103", "48987" and the
		// date digits are all lurking in here.
		{
			"archive/037__MP Replay v101.103.48987.0 @2026.07.14 210732 (4).aoe2record",
			"037__MP Replay v101.103.48987.0 @2026.07.14 210732 (4)",
			false,
		},
		{
			"MP_Replay_v101.103.48987.0_@2026.07.22_001851__1_.aoe2record",
			"MP_Replay_v101.103.48987.0_@2026.07.22_001851__1_",
			false,
		},
		// Too short / too long to be a match id.
		{"crawl/12345.zip", "12345", false},
		{"crawl/1234567890123.zip", "1234567890123", false},
	}
	for _, c := range cases {
		if got := GameIDForPath(c.path); got != c.want {
			t.Errorf("GameIDForPath(%q) = %q, want %q", c.path, got, c.want)
		}
		if got := HasMSGameID(GameIDForPath(c.path)); got != c.ms {
			t.Errorf("HasMSGameID for %q = %v, want %v", c.path, got, c.ms)
		}
	}
}

// The scenario name appears many times in a header; a stray single occurrence (chat
// line, settings blob) must not outvote the real one.
func TestScenarioTokenUsesFrequencyNotFirstMatch(t *testing.T) {
	header := []byte(
		"chatter CBA_=REQUIEM=_V293 Random Position more bytes " +
			"CBA_=REQUIEM=_V292 padding CBA_=REQUIEM=_V292 padding CBA_=REQUIEM=_V292")
	if got := scenarioToken(header); got != LadderToken {
		t.Fatalf("scenarioToken = %q, want %q", got, LadderToken)
	}
	if got := scenarioToken(nil); got != "" {
		t.Fatalf("scenarioToken(nil) = %q, want empty", got)
	}
}

// A variant must be identified as itself, not silently pass as the base version.
func TestScenarioTokenKeepsVariantSuffix(t *testing.T) {
	got := scenarioToken([]byte("x CBA_=REQUIEM=_V293 Random Position y"))
	if got != "CBA_=REQUIEM=_V293 Random Position" {
		t.Fatalf("variant token = %q", got)
	}
	if got == LadderToken {
		t.Fatal("variant must not be admitted as the ladder scenario")
	}
}

// Dedup identifies a MATCH, not a FILE: same roster, near-same duration, regardless of
// player ordering or a few seconds of POV recording jitter.
func TestDedupSignatureIdentifiesMatchNotFile(t *testing.T) {
	a := []RegistryPlayer{{ProfileID: 3}, {ProfileID: 1}, {ProfileID: 2}, {ProfileID: 4}}
	b := []RegistryPlayer{{ProfileID: 1}, {ProfileID: 2}, {ProfileID: 3}, {ProfileID: 4}}

	if dedupSignature(a, 1803) != dedupSignature(b, 1807) {
		t.Fatal("same roster within the same 10s bucket must share a signature")
	}
	if dedupSignature(a, 1803) == dedupSignature(b, 2400) {
		t.Fatal("clearly different durations must not collide")
	}
	c := []RegistryPlayer{{ProfileID: 1}, {ProfileID: 2}, {ProfileID: 3}, {ProfileID: 9}}
	if dedupSignature(a, 1803) == dedupSignature(c, 1803) {
		t.Fatal("different rosters must not collide")
	}
}

func intp(v int) *int { return &v }

func perfRow(game string, pid int, team string, prod, produced, lost int) PerformanceRow {
	return PerformanceRow{
		GameID: game, ProfileID: pid, Team: team,
		ProdBuildings: intp(prod),
		UnitsProduced: intp(produced),
		UnitsLost:     intp(lost),
	}
}

// Unresolved/AI seats (profile_id <= 0) must be dropped BEFORE grouping. Filtering them
// later is not equivalent: they would still sit in the team production total and dilute
// every real player's eco share. This is a regression guard -- the first cut of the Go
// port reported a real player at eco 0.41 instead of 0.81 for exactly this reason.
func TestComputeAxesDropsPhantomSeatsBeforeGrouping(t *testing.T) {
	corpus := &Corpus{}
	for _, g := range []string{"g1", "g2", "g3"} {
		corpus.Games = append(corpus.Games, GameRecord{
			GameID:    g,
			DurationS: 1800,
			Rows: []PerformanceRow{
				perfRow(g, 100, "A", 20, 200, 100),
				perfRow(g, 200, "A", 20, 200, 100),
				perfRow(g, -1, "A", 60, 200, 100), // phantom seat, huge production
			},
		})
	}

	axes := ComputeAxes(corpus, AxesOptions{MinGames: 3})
	if len(axes) != 2 {
		t.Fatalf("scored %d players, want 2 (phantom seat must not be a player)", len(axes))
	}
	for _, p := range axes {
		if p.ProfileID <= 0 {
			t.Fatalf("phantom profile id %d present in output", p.ProfileID)
		}
		// Two real players splitting production evenly = an exactly equal share.
		if p.Eco != 1.00 {
			t.Fatalf("player %d eco = %.2f, want 1.00 (phantom row must not dilute the "+
				"team denominator)", p.ProfileID, p.Eco)
		}
	}
}

// Combat must be normalised within the player's OWN TEAM. Normalising against the whole
// game just re-measures winning -- that version credited a known stacker at 1.49.
func TestCombatIsTeamRelativeNotGameRelative(t *testing.T) {
	corpus := &Corpus{}
	for _, g := range []string{"g1", "g2", "g3"} {
		corpus.Games = append(corpus.Games, GameRecord{
			GameID:    g,
			DurationS: 1800,
			Rows: []PerformanceRow{
				// Team A: one player preserves twice as well as their teammate.
				perfRow(g, 100, "A", 20, 400, 100),
				perfRow(g, 200, "A", 20, 400, 200),
				// Team B is getting crushed. If combat were game-relative, team A's
				// pair would both be inflated by team B's losses.
				perfRow(g, 300, "B", 20, 400, 380),
				perfRow(g, 400, "B", 20, 400, 380),
			},
		})
	}

	axes := ComputeAxes(corpus, AxesOptions{MinGames: 3})
	got := map[int]float64{}
	for _, p := range axes {
		got[p.ProfileID] = p.Combat
	}

	// Within team A, mean loss rate is (0.25+0.50)/2 = 0.375.
	// Player 100: 0.375/0.25 = 1.50   Player 200: 0.375/0.50 = 0.75
	if got[100] != 1.50 {
		t.Errorf("player 100 combat = %.2f, want 1.50", got[100])
	}
	if got[200] != 0.75 {
		t.Errorf("player 200 combat = %.2f, want 0.75", got[200])
	}
	// Team B players lose identically to each other, so both sit at parity -- being on
	// the losing side must not by itself move the axis.
	if got[300] != 1.00 || got[400] != 1.00 {
		t.Errorf("team B combat = %.2f/%.2f, want 1.00/1.00 (losing must not itself "+
			"penalise the axis)", got[300], got[400])
	}
}

func perfRowCiv(game string, pid int, team, civ string, prod, produced, lost int) PerformanceRow {
	r := perfRow(game, pid, team, prod, produced, lost)
	r.Civ = civ
	return r
}

// CBA spawns armies at wildly different rates per civ (decoded: Malay 20 units/s vs
// Khmer 2 units/s), so raw loss rate scores the CIV, not the player. Two players who each
// perform exactly at their own civ's norm must come out EQUAL -- otherwise the axis just
// reports who drew the friendlier civ.
func TestCombatCivAdjustmentNeutralisesPureCivEffect(t *testing.T) {
	corpus := &Corpus{}
	// 10 games so both civs clear MinCivSamplesForBaseline.
	for i := 0; i < 10; i++ {
		g := "g" + string(rune('a'+i))
		corpus.Games = append(corpus.Games, GameRecord{
			GameID: g, DurationS: 1800,
			Rows: []PerformanceRow{
				perfRowCiv(g, 100, "A", "LossyCiv", 20, 1000, 800), // norm 0.80
				perfRowCiv(g, 200, "A", "TankyCiv", 20, 1000, 400), // norm 0.40
			},
		})
	}

	got := map[int]float64{}
	for _, p := range ComputeAxes(corpus, AxesOptions{MinGames: 3}) {
		got[p.ProfileID] = p.Combat
	}

	// Each player sits exactly at their civ's baseline, so both index to 1.00.
	// WITHOUT the adjustment the raw team mean is 0.60, which would score the tanky-civ
	// player 0.60/0.40 = 1.50 ("specialist (combat)") and the other 0.75 -- a verdict
	// produced entirely by civ assignment.
	if got[100] != 1.00 || got[200] != 1.00 {
		t.Fatalf("civ-adjusted combat = %.2f/%.2f, want 1.00/1.00 (civ effect must cancel)",
			got[100], got[200])
	}
}

// Real skill must still register after civ adjustment -- the correction must not flatten
// genuine differences, only civ-driven ones.
func TestCombatCivAdjustmentPreservesRealDifference(t *testing.T) {
	corpus := &Corpus{}
	for i := 0; i < 10; i++ {
		g := "g" + string(rune('a'+i))
		lost := 800
		if i >= 5 {
			lost = 400 // player 100 plays the same civ far better in half the games
		}
		corpus.Games = append(corpus.Games, GameRecord{
			GameID: g, DurationS: 1800,
			Rows: []PerformanceRow{
				perfRowCiv(g, 100, "A", "SameCiv", 20, 1000, lost),
				perfRowCiv(g, 200, "A", "SameCiv", 20, 1000, 800),
			},
		})
	}
	got := map[int]float64{}
	for _, p := range ComputeAxes(corpus, AxesOptions{MinGames: 3}) {
		got[p.ProfileID] = p.Combat
	}
	if !(got[100] > got[200]) {
		t.Fatalf("same-civ skill difference was flattened: %.2f vs %.2f", got[100], got[200])
	}
}

// A civ with too few samples must pass through UNADJUSTED rather than be corrected by a
// noisy baseline or silently folded into a global average.
func TestCivBaselineWithheldBelowSampleFloor(t *testing.T) {
	var rows []PerformanceRow
	for i := 0; i < MinCivSamplesForBaseline; i++ {
		rows = append(rows, perfRowCiv("g", 1, "A", "CommonCiv", 20, 1000, 500))
	}
	rows = append(rows, perfRowCiv("g", 2, "A", "RareCiv", 20, 1000, 900))

	base := civLossBaselines(rows)
	if _, ok := base["CommonCiv"]; !ok {
		t.Error("civ at the sample floor should have a baseline")
	}
	if _, ok := base["RareCiv"]; ok {
		t.Error("civ below the sample floor must NOT get a baseline")
	}
	// The rare civ's row must still be scoreable, just uncorrected.
	v, ok := lossRateCivAdjusted(rows[len(rows)-1], base)
	if !ok || v != 0.9 {
		t.Fatalf("unadjusted passthrough = %.3f (ok=%v), want raw 0.900", v, ok)
	}
}

// Small denominators produce absurd preservation ratios; the floor keeps them out.
func TestCombatIgnoresTinyProductionSamples(t *testing.T) {
	corpus := &Corpus{}
	for _, g := range []string{"g1", "g2", "g3"} {
		corpus.Games = append(corpus.Games, GameRecord{
			GameID: g, DurationS: 1800,
			Rows: []PerformanceRow{
				perfRow(g, 100, "A", 20, MinProdForCombat-1, 1),
				perfRow(g, 200, "A", 20, MinProdForCombat-1, 50),
			},
		})
	}
	for _, p := range ComputeAxes(corpus, AxesOptions{MinGames: 3}) {
		if p.HasCombat {
			t.Fatalf("player %d scored combat on a sub-floor sample", p.ProfileID)
		}
	}
}

// "Genuinely good = multiple positive correlation": independent axes must AGREE.
func TestVerdictRequiresAxisAgreement(t *testing.T) {
	cases := []struct {
		name       string
		eco        float64
		combat     float64
		win        int
		wantPrefix string
	}{
		{"carries both", 1.20, 1.08, 40, "CONVERGENT"},
		{"eco only", 1.20, 1.00, 50, "specialist (eco)"},
		{"combat only", 1.00, 1.10, 50, "specialist (combat)"},
		{"wins carrying neither", 0.81, 0.90, 94, "RIDER?"},
		{"wins but not low enough", 0.95, 0.98, 94, "-"},
		{"low but does not win", 0.80, 0.90, 20, "-"},
	}
	for _, c := range cases {
		p := PlayerAxes{Eco: c.eco, HasEco: true, Combat: c.combat, HasCombat: true, WinPct: c.win}
		if got := verdictFor(p); !strings.HasPrefix(got, c.wantPrefix) {
			t.Errorf("%s: verdict = %q, want prefix %q", c.name, got, c.wantPrefix)
		}
	}

	// Without both axes there is no agreement to assess, so no verdict may be claimed.
	if got := verdictFor(PlayerAxes{Eco: 1.5, HasEco: true}); got != "" {
		t.Errorf("verdict without combat = %q, want empty", got)
	}
}

// Registry teams must come from the extractor's own rule, or the registry and the
// performance rows can silently disagree about which side a player was on.
func TestRegistryTeamsMatchExtractorRule(t *testing.T) {
	for slot := 1; slot <= 8; slot++ {
		want := cbaTeam(slot)
		if want == "" {
			t.Fatalf("slot %d has no team", slot)
		}
		if slot <= 4 && want != "A" {
			t.Fatalf("slot %d = %q, want A", slot, want)
		}
		if slot >= 5 && want != "B" {
			t.Fatalf("slot %d = %q, want B", slot, want)
		}
	}
	if cbaTeam(0) != "" || cbaTeam(9) != "" {
		t.Fatal("out-of-range slots must have no team")
	}
}

// Corpus round-trip: a re-ingest of a known game must replace it, not duplicate it.
func TestCorpusUpsertReplacesRatherThanDuplicates(t *testing.T) {
	c := &Corpus{}
	c.Upsert(GameRecord{GameID: "492398692", DurationS: 100})
	c.Upsert(GameRecord{GameID: "492398692", DurationS: 200})
	if len(c.Games) != 1 {
		t.Fatalf("games = %d, want 1", len(c.Games))
	}
	if c.Games[0].DurationS != 200 {
		t.Fatalf("duration = %d, want 200 (upsert must replace)", c.Games[0].DurationS)
	}
	if !c.Has("492398692") || c.Has("nope") {
		t.Fatal("Has() disagrees with corpus contents")
	}
}
