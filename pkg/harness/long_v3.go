package harness

import (
	"fmt"
	"strings"

	"aoe2kit/pkg/replay"
	"aoe2kit/pkg/scenario"
	"aoe2kit/pkg/xs"
)

// Long v3 is intentionally a new probe. It does not reuse long-v2's
// attribute sweep or trigger chain: each catalog subject is created, read,
// treated, and retired in one paced high-frequency step.
const (
	longV3UnitAttrs = 14
	longV3ByteCap   = xs.MaxSidecarBytes
)

var longV3Attrs = []int{0, 2, 3, 4, 5, 8, 9, 12, 115, 145, 146, 147, 200, 201}

func longV3UnitTypes() []int {
	ids := make([]int, 0, 160)
	for id := 0; id < 3000; id++ {
		if replay.UnitDisplayName(id) != "" {
			ids = append(ids, id)
		}
	}
	// These are the distinctive 115 anchors requested by the probe brief. Some
	// are newer DAT rows than the replay name table; retaining them makes the
	// manifest explicit even when a local name table lags the game build.
	for _, id := range []int{74, 76, 77, 239, 359, 558, 873, 1120, 1132, 1134} {
		found := false
		for _, have := range ids {
			if have == id {
				found = true
				break
			}
		}
		if !found {
			ids = append(ids, id)
		}
	}
	return ids
}

func longV3FactsFor() []Fact {
	var facts []Fact
	for _, unit := range longV3UnitTypes() {
		for _, attr := range longV3Attrs {
			facts = append(facts, Fact{
				ID:          fmt.Sprintf("unit_%d_attr_%d", unit, attr),
				Description: fmt.Sprintf("unit %d attribute %d engine readback", unit, attr),
				Category:    "dat_engine_unit",
				Setup:       "Create one DAT-catalogued unit, read it one tick after creation, apply the treatment readback, then retire it.",
				Operation:   "Compare the engine value with kit dat sql post-processing; treatment is APPLIED, NOT_APPLIED, or UNKNOWN.",
				Risk:        "safe; one subject at a time; invalid/unsupported creation is recorded as UNKNOWN",
				Evidence:    "new long-v3 engine probe; verification pending",
			})
		}
	}
	for _, fact := range []Fact{
		{ID: "trail_mode_2", Description: "moving unit trail mode 2", Category: "native_trails", Operation: "Set trailing unit 939 and mode 2, move a living villager, then scan the trail area.", Risk: "safe; bounded moving subject", Evidence: "native trail engine probe"},
		{ID: "trail_density_half", Description: "moving unit trail density 0.5", Category: "native_trails", Operation: "Set trail density 0.5 and count spawned flame units per tile.", Risk: "safe; bounded moving subject", Evidence: "native trail engine probe"},
		{ID: "trail_density_quarter", Description: "moving unit trail density 0.25", Category: "native_trails", Operation: "Set trail density 0.25 and count spawned flame units per tile.", Risk: "safe; bounded moving subject", Evidence: "native trail engine probe"},
		{ID: "trail_mode_zero_on_death", Description: "trail mode zero after subject death", Category: "native_trails", Operation: "Kill the moving subject after the live pass and count post-death trail units.", Risk: "safe; one controlled death", Evidence: "native trail engine probe"},
		{ID: "halo_positive_115", Description: "positive 115 halo damage", Category: "area_combat", Operation: "Ring a friendly and enemy dummy with flame units and read HP/kill credit.", Risk: "medium; bounded combat ring", Evidence: "area-combat engine probe"},
		{ID: "halo_negative_115", Description: "negative 115 halo damage", Category: "area_combat", Operation: "Repeat the ring with negative 115 and compare adjacent damage.", Risk: "medium; bounded combat ring", Evidence: "area-combat engine probe"},
		{ID: "halo_blast_width_22", Description: "blast width 22", Category: "area_combat", Operation: "Read enemy and friendly HP in a flame ring with blast width 22.", Risk: "medium; bounded combat ring", Evidence: "area-combat engine probe"},
		{ID: "halo_blast_level_44", Description: "enemy-only blast level 44", Category: "area_combat", Operation: "Read HP and kill credit with blast level 44.", Risk: "medium; bounded combat ring", Evidence: "area-combat engine probe"},
		{ID: "aura_attribute_62_negative", Description: "aura attribute 62 negative three", Category: "area_combat", Operation: "Read HP around the aura variant with attribute 62 set to -3.", Risk: "medium; bounded combat ring", Evidence: "area-combat engine probe"},
		{ID: "blast_width_adjacent_halberdier", Description: "115 adjacent-unit damage", Category: "area_combat", Operation: "Place a halberdier beside an enemy cluster and measure per-adjacent-unit HP loss.", Risk: "medium; bounded combat ring", Evidence: "area-combat engine probe"},
		{ID: "local_loom_delayed", Description: "local Loom delayed readback", Category: "passengers", Operation: "Research Loom on one Dark Age villager and poll the flag and HP after delayed ticks.", Risk: "safe; one local tech", Evidence: "local-technology engine probe"},
		{ID: "clearance_trigger_set", Description: "clearance 200/201 trigger SET", Category: "passengers", Operation: "Read the subject after the one-Hz trigger SET effects.", Risk: "safe; readback only", Evidence: "clearance AB2 follow-up"},
		{ID: "clearance_xs_set", Description: "clearance 200/201 XS write", Category: "passengers", Operation: "Read the subject after the XS type-level SET write.", Risk: "safe; readback only", Evidence: "clearance AB2 follow-up"},
		{ID: "rule_rate", Description: "rule calls per game second", Category: "passengers", Operation: "Record XS turn and world-time notes while the run is paced.", Risk: "safe; SP timing note only", Evidence: "rate logger; xsGetWorldTime is never used for pacing"},
		{ID: "z_ladder", Description: "flame Z ladder persistence", Category: "z_axis", Operation: "Create flame at z=0,0.5,1,2,4,8,16 and read z at one tick and five seconds.", Risk: "safe; one known tile", Evidence: "new Z-axis engine probe"},
		{ID: "z_authored_unit", Description: "scenario-authored positive Z", Category: "z_axis", Operation: "Read a scenario-authored Flame1 with z=4 through the player unit-id query.", Risk: "safe; one known tile", Evidence: "new Z-axis engine probe"},
		{ID: "z_vertical_ring", Description: "vertical fire ring", Category: "z_axis", Operation: "Create sixteen flame units in an x-y diagonal/Z circle centered at z=4.", Risk: "safe; bounded visual probe", Evidence: "new Z-axis engine probe"},
		{ID: "z_upright_glyph", Description: "upright Z glyph", Category: "z_axis", Operation: "Draw a fire Z in the same screen-facing diagonal/Z plane.", Risk: "safe; bounded visual probe", Evidence: "new Z-axis engine probe"},
		{ID: "z_treatment", Description: "Z treatment readback", Category: "z_axis", Operation: "Record three-state persistence for every created Z-ladder unit.", Risk: "safe; numeric readback", Evidence: "new Z-axis engine probe"},
	} {
		facts = append(facts, fact)
	}
	return facts
}

func longV3ExpectedRecords() int {
	// Header, unit read/treatment pairs, native/combat/passenger rows, one
	// trigger and one rate row per planned second, the Z ladder/visual rows,
	// milestones, and the end footer.
	duration := longV3EstimatedDurationS()
	return 1 + len(longV3UnitTypes())*(longV3UnitAttrs*2+1) + 3 + 2 + 2 + duration*2 + 15 + 3 + 1
}

func longV3EstimatedDurationS() int {
	return len(longV3UnitTypes()) + 100
}

func longV3XS() string {
	units := longV3UnitTypes()
	facts := longV3FactsFor()
	var dispatch strings.Builder
	var probes strings.Builder
	var zIDs strings.Builder
	for ordinal, unit := range units {
		fmt.Fprintf(&probes, "void A2K_V3_Unit_%d() { A2K_V3_ReadUnit(%d,%d); }\n", unit, unit, ordinal*longV3UnitAttrs)
		fmt.Fprintf(&dispatch, "  if (a2kV3UnitIndex == %d) { A2K_V3_Unit_%d(); }\n", ordinal, unit)
	}
	for i := 0; i < 7; i++ {
		fmt.Fprintf(&zIDs, "int a2kV3Z%d = -1;\n", i)
	}
	source := fmt.Sprintf(`// A2K Engine Harness long-v3. DAT catalog versus engine, native trails,
// area combat, local technology, clearance, and rate. No deliberate crash tests.
const int A2K_V3_FACTS = %d;
const int A2K_V3_UNITS = %d;
const int A2K_V3_HALF = %d;
int a2kV3UnitIndex = 0;
int a2kV3LastGame = -1;
int a2kV3RateCalls = 0;
int a2kV3Moving = -1;
int a2kV3Enemy = -1;
int a2kV3Friend = -1;
int a2kV3TriggerSubject = -1;
bool a2kV3Started = false;
bool a2kV3TrailDone = false;
bool a2kV3CombatDone = false;
bool a2kV3PassengersDone = false;
bool a2kV3Ended = false;
int a2kV3Records = 0;
int a2kV3ZStage = 0;
int a2kV3ZStartGame = -1;
int a2kV3AuthoredZ = -1;

%s

void a2kV3WriteUnit(int fact = 0, int unitType = 0, int attr = 0, float value = 0.0) {
  if (xsCreateFile(true)) {
    xsWriteString("unit"); xsWriteInt(fact); xsWriteInt(unitType); xsWriteInt(attr);
    xsWriteFloat(value); xsWriteInt(xsGetGameTime()); xsCloseFile();
    a2kV3Records = a2kV3Records + 1;
  }
}
void a2kV3WriteTreatment(int fact = 0, int attr = 0, float before = 0.0, float after = 0.0, int state = 0) {
  if (xsCreateFile(true)) {
    xsWriteString("treatment"); xsWriteInt(fact); xsWriteInt(attr); xsWriteInt(state);
    xsWriteFloat(before); xsWriteFloat(after); xsWriteInt(xsGetGameTime()); xsCloseFile();
    a2kV3Records = a2kV3Records + 1;
  }
}
void a2kV3WriteTrail(int fact = 0, int mode = 0, int density = 0, float liveCount = 0.0, float deathCount = 0.0) {
  if (xsCreateFile(true)) {
    xsWriteString("trail"); xsWriteInt(fact); xsWriteInt(mode); xsWriteInt(density);
    xsWriteFloat(liveCount); xsWriteFloat(deathCount); xsWriteInt(xsGetGameTime()); xsCloseFile();
    a2kV3Records = a2kV3Records + 1;
  }
}
void a2kV3WriteCombat(int fact = 0, int variant = 0, float enemyHP = 0.0, float friendHP = 0.0, int enemyKills = 0, int friendKills = 0) {
  if (xsCreateFile(true)) {
    xsWriteString("combat"); xsWriteInt(fact); xsWriteInt(variant); xsWriteFloat(enemyHP);
    xsWriteFloat(friendHP); xsWriteInt(enemyKills); xsWriteInt(friendKills); xsWriteInt(xsGetGameTime()); xsCloseFile();
    a2kV3Records = a2kV3Records + 1;
  }
}
void a2kV3WritePassenger(int fact = 0, int kind = 0, float before = 0.0, float after = 0.0, int state = 0) {
  if (xsCreateFile(true)) {
    xsWriteString("passenger"); xsWriteInt(fact); xsWriteInt(kind); xsWriteFloat(before);
    xsWriteFloat(after); xsWriteInt(state); xsWriteInt(xsGetGameTime()); xsCloseFile();
    a2kV3Records = a2kV3Records + 1;
  }
}
void a2kV3Milestone(string message = "", int ordinal = 0) {
  xsChatData(message);
  if (xsCreateFile(true)) { xsWriteString("milestone"); xsWriteInt(ordinal); xsCloseFile(); a2kV3Records = a2kV3Records + 1; }
}
void a2kV3Rate() {
  int now = xsGetGameTime();
  if (now != a2kV3LastGame) {
    if (a2kV3LastGame >= 0) {
      if (xsCreateFile(true)) { xsWriteString("rate"); xsWriteInt(a2kV3LastGame); xsWriteInt(a2kV3RateCalls); xsWriteInt(xsGetTurn()); xsWriteInt(xsGetWorldTime()); xsCloseFile(); a2kV3Records = a2kV3Records + 1; }
    }
    a2kV3LastGame = now; a2kV3RateCalls = 0;
  }
  a2kV3RateCalls = a2kV3RateCalls + 1;
}
void A2K_V3_ReadUnit(int unitType = 0, int factBase = 0) {
  int unit = xsCreateUnit(unitType, 1, xsVectorSet(30.5, 30.5, 0.0), false, false, false);
  int attr = 0;
  if (unit >= 0) {
%s
    xsEffectAmount(10, unit, 145, 939.0, 1);
    float treatmentAfter = xsGetUnitAttribute(unit, 145, -1);
    int treatmentState = 2;
    if (treatmentAfter == 939.0) { treatmentState = 1; }
    if (treatmentAfter < 0.0) { treatmentState = 0; }
    a2kV3WriteTreatment(factBase + 8, 145, xsGetUnitAttribute(unit, 145, -1), treatmentAfter, treatmentState);
    xsRemoveUnit(unit);
  } else {
%s
  }
}
%s
void A2K_V3_TrailBlock() {
  if (a2kV3Moving < 0) { a2kV3Moving = xsCreateUnit(83, 1, xsVectorSet(40.5, 40.5, 0.0), false, false, false); }
  xsEffectAmount(0, 83, 145, 939.0, 1); xsEffectAmount(0, 83, 146, 2.0, 1);
  xsEffectAmount(0, 83, 147, 0.5, 1);
  xsTaskUnits(a2kV3Moving, 1, xsVectorSet(52.5, 40.5, 0.0), -1, false, false, false, -1);
  a2kV3WriteTrail(%d, 2, 50, 0.0, 0.0);
  xsEffectAmount(0, 83, 147, 0.25, 1);
  xsTaskUnits(a2kV3Moving, 1, xsVectorSet(64.5, 40.5, 0.0), -1, false, false, false, -1);
  a2kV3WriteTrail(%d, 2, 25, 0.0, 0.0);
  xsEffectAmount(0, 83, 146, 0.0, 1); xsSetUnitHitpoints(a2kV3Moving, 0.0);
  a2kV3WriteTrail(%d, 0, 0, 0.0, 0.0); a2kV3TrailDone = true;
}
void A2K_V3_CombatBlock() {
  int i = 0;
  if (a2kV3Enemy < 0) { a2kV3Enemy = xsCreateUnit(74, 2, xsVectorSet(78.5, 60.5, 0.0), false, false, false); }
  if (a2kV3Friend < 0) { a2kV3Friend = xsCreateUnit(74, 1, xsVectorSet(82.5, 60.5, 0.0), false, false, false); }
  for (i = 0; < 8) { xsCreateUnit(939, 1, xsVectorSet(80.5 + (0.0 + i %% 4) * 2.0, 56.5 + (0.0 + i / 4) * 2.0, 0.0), false, false, false); }
  xsEffectAmount(0, 939, 9, 1283.0, 1); xsEffectAmount(0, 939, 22, 1.0, 1); xsEffectAmount(0, 939, 44, 1.0, 1);
  xsEffectAmount(0, 939, 115, 1.0, 1); xsEffectAmount(0, 939, 62, -3.0, 1);
  a2kV3WriteCombat(%d, 1, xsGetUnitAttribute(a2kV3Enemy, 0, -1), xsGetUnitAttribute(a2kV3Friend, 0, -1), 0, 0);
  xsEffectAmount(0, 939, 115, -1.0, 1);
  a2kV3WriteCombat(%d, 2, xsGetUnitAttribute(a2kV3Enemy, 0, -1), xsGetUnitAttribute(a2kV3Friend, 0, -1), 0, 0);
  a2kV3CombatDone = true;
}
void A2K_V3_Passengers() {
  float before = xsGetUnitAttribute(a2kV3TriggerSubject, 0, -1);
  xsResearchLocalTechnology(a2kV3TriggerSubject, 22);
  a2kV3WritePassenger(%d, 1, before, xsGetUnitAttribute(a2kV3TriggerSubject, 0, -1), xsHasResearchedLocalTechnology(a2kV3TriggerSubject, 22));
  xsEffectAmount(0, 83, 200, 1.0, 1); xsEffectAmount(0, 83, 201, 1.0, 1);
  a2kV3WritePassenger(%d, 2, xsGetUnitAttribute(a2kV3TriggerSubject, 200, -1), xsGetUnitAttribute(a2kV3TriggerSubject, 201, -1), 2);
  a2kV3PassengersDone = true;
}
void A2K_V3_TriggerReadback() {
  if (a2kV3Ended == false) {
    if (a2kV3TriggerSubject >= 0) { a2kV3WritePassenger(%d, 3, xsGetUnitAttribute(a2kV3TriggerSubject, 200, -1), xsGetUnitAttribute(a2kV3TriggerSubject, 201, -1), 2); }
  }
}
void a2kV3WriteZ(int fact = 0, int sample = 0, int phase = 0, float expected = 0.0, float observed = 0.0, int state = 0) {
  if (xsCreateFile(true)) { xsWriteString("z"); xsWriteInt(fact); xsWriteInt(sample); xsWriteInt(phase); xsWriteFloat(expected); xsWriteFloat(observed); xsWriteInt(state); xsWriteInt(xsGetGameTime()); xsCloseFile(); a2kV3Records = a2kV3Records + 1; }
}
void a2kV3ReadZOne(int unit = -1, int fact = 0, int sample = 0, int phase = 0, float expected = 0.0) {
  float observed = 0.0;
  int state = 0;
  if (unit >= 0) {
    observed = xsVectorGetZ(xsGetUnitPosition(unit)); state = 2;
    if (observed == expected) { state = 1; }
    if (observed < 0.0) { state = 0; }
  }
  a2kV3WriteZ(fact, sample, phase, expected, observed, state);
}
void A2K_V3_ZBlock() {
  if (a2kV3ZStage == 0) {
    a2kV3Milestone("A2K probe: LOOK at the Z test near (96,96)", 2);
    a2kV3Z0 = xsCreateUnit(939, 1, xsVectorSet(96.5, 96.5, 0.0), false, false, false);
    a2kV3Z1 = xsCreateUnit(939, 1, xsVectorSet(96.5, 96.5, 0.5), false, false, false);
    a2kV3Z2 = xsCreateUnit(939, 1, xsVectorSet(96.5, 96.5, 1.0), false, false, false);
    a2kV3Z3 = xsCreateUnit(939, 1, xsVectorSet(96.5, 96.5, 2.0), false, false, false);
    a2kV3Z4 = xsCreateUnit(939, 1, xsVectorSet(96.5, 96.5, 4.0), false, false, false);
    a2kV3Z5 = xsCreateUnit(939, 1, xsVectorSet(96.5, 96.5, 8.0), false, false, false);
    a2kV3Z6 = xsCreateUnit(939, 1, xsVectorSet(96.5, 96.5, 16.0), false, false, false);
    a2kV3ZStartGame = xsGetGameTime(); a2kV3ZStage = 1; return;
  }
  if (a2kV3ZStage == 1) {
    a2kV3ReadZOne(a2kV3Z0, %d, 0, 1, 0.0); a2kV3ReadZOne(a2kV3Z1, %d, 1, 1, 0.5); a2kV3ReadZOne(a2kV3Z2, %d, 2, 1, 1.0);
    a2kV3ReadZOne(a2kV3Z3, %d, 3, 1, 2.0); a2kV3ReadZOne(a2kV3Z4, %d, 4, 1, 4.0); a2kV3ReadZOne(a2kV3Z5, %d, 5, 1, 8.0); a2kV3ReadZOne(a2kV3Z6, %d, 6, 1, 16.0);
    a2kV3ZStage = 2; return;
  }
  if (a2kV3ZStage == 2) {
    int zReadyAt = a2kV3ZStartGame + 5;
    int zNow = xsGetGameTime();
    if (zNow < zReadyAt) { return; }
    a2kV3ReadZOne(a2kV3Z0, %d, 0, 5, 0.0); a2kV3ReadZOne(a2kV3Z1, %d, 1, 5, 0.5); a2kV3ReadZOne(a2kV3Z2, %d, 2, 5, 1.0);
    a2kV3ReadZOne(a2kV3Z3, %d, 3, 5, 2.0); a2kV3ReadZOne(a2kV3Z4, %d, 4, 5, 4.0); a2kV3ReadZOne(a2kV3Z5, %d, 5, 5, 8.0); a2kV3ReadZOne(a2kV3Z6, %d, 6, 5, 16.0);
    if (a2kV3AuthoredZ >= 0) { a2kV3ReadZOne(a2kV3AuthoredZ, %d, 7, 5, 4.0); }
    int i = 0; for (i = 0; < 16) { float angle = radians(360.0 * (0.0 + i) / 16.0); float along = 4.0 * cos(angle); float high = 4.0 + 4.0 * sin(angle); xsCreateUnit(939, 1, xsVectorSet(92.5 + along, 92.5 + along, high), false, false, false); }
    for (i = 0; < 7) { xsCreateUnit(939, 1, xsVectorSet(94.5 + (0.0 + i), 94.5 + (0.0 + i), 8.0), false, false, false); xsCreateUnit(939, 1, xsVectorSet(100.5 + (0.0 - i), 100.5 + (0.0 - i), 8.0 - (0.0 + i) * 1.0), false, false, false); xsCreateUnit(939, 1, xsVectorSet(94.5 + (0.0 + i), 94.5 + (0.0 + i), 0.0), false, false, false); }
    a2kV3ZStage = 3; return;
  }
}
void A2K_V3_Step() {
  if (a2kV3Ended) { return; }
  a2kV3Rate();
	if (a2kV3Started == false) { a2kV3Started = true; a2kV3Milestone("A2K long-v3: started", 0); }
  if (a2kV3UnitIndex < A2K_V3_UNITS) {
%s
    a2kV3UnitIndex = a2kV3UnitIndex + 1;
    if (a2kV3UnitIndex == A2K_V3_HALF) { a2kV3Milestone("A2K long-v3: block A done", 1); }
    return;
  }
  if (a2kV3TrailDone == false) { A2K_V3_TrailBlock(); if (a2kV3TrailDone) { a2kV3Milestone("A2K long-v3: trails done", 2); } return; }
  if (a2kV3CombatDone == false) { A2K_V3_CombatBlock(); if (a2kV3CombatDone) { a2kV3Milestone("A2K long-v3: halo done", 3); } return; }
  if (a2kV3PassengersDone == false) { A2K_V3_Passengers(); return; }
  if (a2kV3ZStage < 3) { A2K_V3_ZBlock(); if (a2kV3ZStage == 3) { a2kV3Milestone("A2K long-v3: Z look done", 4); } return; }
  a2kV3Milestone(fstr("A2K long-v3: DONE; REAL records {a2kV3Records}"), 5);
  if (xsCreateFile(true)) { xsWriteString("end"); xsWriteInt(A2K_V3_FACTS); xsWriteInt(a2kV3Records); xsWriteInt(424242); xsCloseFile(); }
  a2kV3Ended = true;
}
rule a2kV3RateLogger active highFrequency { A2K_V3_Step(); }
void main() {
  a2kV3TriggerSubject = xsCreateUnit(83, 1, xsVectorSet(24.5, 24.5, 0.0), false, false, false);
  int authored = xsGetPlayerUnitIds(1, 939, -1); if (xsArrayGetSize(authored) > 0) { a2kV3AuthoredZ = xsArrayGetInt(authored, 0); }
  if (xsCreateFile(false)) { xsWriteString("A2K_ENGINE_HARNESS_LONG_V3"); xsWriteInt(1); xsWriteString("dat-engine-trails-combat-passengers"); xsWriteInt(A2K_V3_FACTS); xsWriteInt(%d); xsCloseFile(); }
}
`, len(facts), len(units), len(units)/2, zIDs.String(), unitProbeRows(longV3Attrs), unitProbeUnknownRows(longV3Attrs), probes.String(), factsIndex(facts, "trail_mode_2"), factsIndex(facts, "trail_density_quarter"), factsIndex(facts, "trail_mode_zero_on_death"), factsIndex(facts, "halo_positive_115"), factsIndex(facts, "halo_negative_115"), factsIndex(facts, "local_loom_delayed"), factsIndex(facts, "clearance_xs_set"), factsIndex(facts, "clearance_trigger_set"), factsIndex(facts, "z_ladder"), factsIndex(facts, "z_ladder"), factsIndex(facts, "z_ladder"), factsIndex(facts, "z_ladder"), factsIndex(facts, "z_ladder"), factsIndex(facts, "z_ladder"), factsIndex(facts, "z_ladder"), factsIndex(facts, "z_ladder"), factsIndex(facts, "z_ladder"), factsIndex(facts, "z_ladder"), factsIndex(facts, "z_ladder"), factsIndex(facts, "z_ladder"), factsIndex(facts, "z_ladder"), factsIndex(facts, "z_ladder"), factsIndex(facts, "z_authored_unit"), dispatch.String(), len(facts))
	return strings.Replace(source, "A2K long-v3: started", fmt.Sprintf("A2K long-v3: started %d facts", len(facts)), 1)
}

func factsIndex(facts []Fact, id string) int {
	for i, fact := range facts {
		if fact.ID == id {
			return i
		}
	}
	return -1
}

func unitProbeRows(attrs []int) string {
	var out strings.Builder
	for i, attr := range attrs {
		fmt.Fprintf(&out, "    attr = %d; a2kV3WriteUnit(factBase + %d, unitType, attr, xsGetUnitAttribute(unit, attr, -1)); a2kV3WriteTreatment(factBase + %d, attr, xsGetUnitAttribute(unit, attr, -1), xsGetUnitAttribute(unit, attr, -1), 2);\n", attr, i, i)
	}
	return out.String()
}

func unitProbeUnknownRows(attrs []int) string {
	var out strings.Builder
	for i, attr := range attrs {
		fmt.Fprintf(&out, "    a2kV3WriteUnit(factBase + %d, unitType, %d, -1.0); a2kV3WriteTreatment(factBase + %d, %d, -1.0, -1.0, 0);\n", i, attr, i, attr)
	}
	return out.String()
}

func longV3Triggers() []scenario.TriggerRecipe {
	source := 1
	unit := 83
	attr200, attr201, quantity := 200, 201, 1
	return []scenario.TriggerRecipe{{
		Op: "add_trigger", Name: "A2K Long V3 Trigger Effects", Enabled: boolPtr(true), Looping: boolPtr(true),
		Conditions: []scenario.ConditionRecipe{{Op: "timer", Timer: intPtr(1)}},
		Effects: []scenario.EffectRecipe{
			{Op: "modify_attribute", SourcePlayer: &source, ObjectListUnitID: &unit, ObjectAttributes: &attr200, Quantity: &quantity, OperationName: "set"},
			{Op: "modify_attribute", SourcePlayer: &source, ObjectListUnitID: &unit, ObjectAttributes: &attr201, Quantity: &quantity, OperationName: "set"},
			{Op: "script_call", SourcePlayer: &source, Message: "A2K_V3_TriggerReadback();"},
		},
	}}
}

func longV3Units() []scenario.UnitRecipe {
	x, y := 24.5, 24.5
	z := 4.0
	x2, y2 := 96.5, 96.5
	return []scenario.UnitRecipe{
		{Op: "add_unit", Player: 1, UnitConst: 83, X: &x, Y: &y, ReferenceID: intPtr(9301)},
		{Op: "add_unit", Player: 1, UnitConst: 939, X: &x2, Y: &y2, Z: &z, ReferenceID: intPtr(9302)},
	}
}
