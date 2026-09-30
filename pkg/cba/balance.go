package cba

import "aoe2kit/pkg/replay"

type BalanceReport struct {
	Method       string            `json:"method"`
	Verification string            `json:"verification"`
	Summary      BalanceSummary    `json:"summary"`
	Rows         []BalanceRow      `json:"rows"`
	Evidence     []BalanceEvidence `json:"evidence,omitempty"`
	Missing      []BalanceMissing  `json:"missing,omitempty"`
	Warnings     []string          `json:"warnings,omitempty"`
}

type BalanceSummary struct {
	Rows            int `json:"rows"`
	CivRows         int `json:"civ_rows"`
	VersionRows     int `json:"version_rows"`
	TriggerEvidence int `json:"trigger_evidence"`
	MissingRows     int `json:"missing_rows"`
}

type BalanceRow struct {
	Version          string   `json:"version"`
	CivID            int      `json:"civ_id,omitempty"`
	CivName          string   `json:"civ_name"`
	RazesToVillager  *int     `json:"razes_to_villager,omitempty"`
	CastleKills      *int     `json:"castle_kills,omitempty"`
	ImperialKills    *int     `json:"imperial_kills,omitempty"`
	UnitCount        *int     `json:"unit_count,omitempty"`
	SpawnSeconds     *int     `json:"spawn_seconds,omitempty"`
	StableElephants  *int     `json:"stable_elephants,omitempty"`
	BattleElephants  *int     `json:"battle_elephants,omitempty"`
	Notes            string   `json:"notes,omitempty"`
	Sources          []string `json:"sources"`
	Coverage         string   `json:"coverage"`
	PreferredForV292 bool     `json:"preferred_for_v292,omitempty"`
}

type BalanceEvidence struct {
	Version    string `json:"version"`
	Kind       string `json:"kind"`
	Attribute  int    `json:"attribute,omitempty"`
	Amounts    []int  `json:"amounts,omitempty"`
	Counts     []int  `json:"counts,omitempty"`
	Source     string `json:"source"`
	Confidence string `json:"confidence"`
	Notes      string `json:"notes,omitempty"`
}

type BalanceMissing struct {
	Version string `json:"version"`
	CivID   int    `json:"civ_id,omitempty"`
	CivName string `json:"civ_name,omitempty"`
	Reason  string `json:"reason"`
}

func BuildBalance() *BalanceReport {
	rows := cbaBalanceRows()
	report := &BalanceReport{
		Method:       "source_labeled_cba_requiem_balance_table",
		Verification: "partial_changelog_rows_plus_trigger_graph_global_threshold_evidence",
		Rows:         rows,
		Evidence: []BalanceEvidence{
			{
				Version:    "V292",
				Kind:       "global_trigger_graph_threshold_amounts",
				Attribute:  33,
				Amounts:    []int{1, 5, 10, 30, 40, 50, 70, 80, 100, 150, 200, 250, 300, 350, 400, 450, 500, 550, 600, 650, 700, 750},
				Counts:     []int{40, 16, 16, 16, 8, 16, 16, 8, 16, 16, 16, 16, 48, 16, 32, 16, 64, 16, 48, 16, 16, 8},
				Source:     "V292 embedded trigger graph from latest_cba.aoe2record: 480 accumulate-attribute conditions, all attribute 33",
				Confidence: "trigger_graph_global_amounts_not_per_civ_mapping",
				Notes:      "The replay trigger graph proves the attr-33 threshold vocabulary, but blank trigger names prevent a complete per-civ mapping from this evidence alone.",
			},
		},
		Warnings: []string{
			"coverage is partial; missing civ/version rows must stay unknown and must not default to a shared razes-to-villager value",
			"trigger graph evidence currently proves global attribute-33 threshold amounts, not a full per-civ mapping",
		},
	}
	report.Missing = cbaBalanceMissingRows()
	report.Summary.Rows = len(report.Rows)
	report.Summary.TriggerEvidence = len(report.Evidence)
	report.Summary.MissingRows = len(report.Missing)
	versions := map[string]bool{}
	for _, row := range rows {
		if row.CivID > 0 {
			report.Summary.CivRows++
		}
		versions[row.Version] = true
	}
	report.Summary.VersionRows = len(versions)
	return report
}

func cbaBalanceMissingRows() []BalanceMissing {
	return []BalanceMissing{
		{
			Version: "V292",
			Reason:  "V292 remains partial: new-civ carry-forward rows and the direct Shu fix are covered, but many legacy civs still need a full recent-capture replay/trigger mapping before normalization",
		},
		{
			Version: "2023-04-02_to_2024-11-19",
			Reason:  "neither Discord capture covers this date range; any balance changes in that window are unknown unless recovered from another source",
		},
	}
}

func cbaBalanceRows() []BalanceRow {
	return []BalanceRow{
		{
			Version:         "V3.1",
			CivID:           28,
			CivName:         replay.CivDisplayName(28),
			RazesToVillager: intPtr(4),
			CastleKills:     intPtr(300),
			ImperialKills:   intPtr(700),
			Notes:           "Khmer changed to Elite Battle Elephant; Tusk Swords auto-research at Imperial.",
			Sources:         []string{"announcements_full.txt:82"},
			Coverage:        "direct_changelog_row",
		},
		{
			Version:         "V8b",
			CivID:           42,
			CivName:         replay.CivDisplayName(42),
			RazesToVillager: intPtr(1),
			UnitCount:       intPtr(60),
			Notes:           "Villager requirement changed 2 razes -> 1 raze; pop fixed 80 -> 60.",
			Sources:         []string{"announcements_full.txt:282", "announcements_full.txt:312"},
			Coverage:        "direct_changelog_row",
		},
		{
			Version:         "V8b",
			CivID:           40,
			CivName:         replay.CivDisplayName(40),
			RazesToVillager: intPtr(2),
			Notes:           "Villager requirement changed 3 razes -> 2 razes.",
			Sources:         []string{"announcements_full.txt:314"},
			Coverage:        "direct_changelog_row",
		},
		{
			Version:       "V8b",
			CivID:         23,
			CivName:       replay.CivDisplayName(23),
			ImperialKills: intPtr(600),
			Notes:         "Imperial Age kills changed 500 -> 600.",
			Sources:       []string{"announcements_full.txt:311"},
			Coverage:      "direct_changelog_row",
		},
		{
			Version:       "V8b",
			CivID:         41,
			CivName:       replay.CivDisplayName(41),
			ImperialKills: intPtr(500),
			Notes:         "Imperial Age kills changed 600 -> 500.",
			Sources:       []string{"announcements_full.txt:313"},
			Coverage:      "direct_changelog_row",
		},
		{
			Version:         "V9",
			CivID:           40,
			CivName:         replay.CivDisplayName(40),
			ImperialKills:   intPtr(400),
			SpawnSeconds:    intPtr(6),
			UnitCount:       intPtr(100),
			BattleElephants: intPtr(90),
			Notes:           "Imperial kills 500 -> 400, spawn 8 -> 6 sec, units 80 -> 100, battle elephants 60 -> 90.",
			Sources:         []string{"announcements_full.txt:396"},
			Coverage:        "direct_changelog_row",
		},
		{
			Version:       "V9f",
			CivID:         40,
			CivName:       replay.CivDisplayName(40),
			ImperialKills: intPtr(500),
			Notes:         "Imperial kills reverted 400 -> 500.",
			Sources:       []string{"announcements_full.txt:471"},
			Coverage:      "direct_changelog_row",
		},
		{
			Version:         "V10",
			CivID:           28,
			CivName:         replay.CivDisplayName(28),
			RazesToVillager: intPtr(4),
			CastleKills:     intPtr(250),
			ImperialKills:   intPtr(500),
			StableElephants: intPtr(30),
			Notes:           "Khmer Ballista Elephants imported; kills 300/600 -> 250/500; razes 3 -> 4; stable elephants available.",
			Sources:         []string{"announcements_full.txt:653", "announcements_full.txt:473", "announcements_full.txt:494"},
			Coverage:        "direct_changelog_row",
		},
		{
			Version:       "V10",
			CivID:         14,
			CivName:       replay.CivDisplayName(14),
			ImperialKills: intPtr(700),
			Notes:         "Imperial kills changed 600 -> 700.",
			Sources:       []string{"announcements_full.txt:655"},
			Coverage:      "direct_changelog_row",
		},
		{
			Version:       "legacy_pre_recent",
			CivID:         33,
			CivName:       replay.CivDisplayName(33),
			ImperialKills: intPtr(500),
			UnitCount:     intPtr(60),
			SpawnSeconds:  intPtr(10),
			Notes:         "Tatars have prior V9 unit/spawn changes and later Imperial kills 400 -> 500.",
			Sources:       []string{"announcements_full.txt:400", "announcements_full.txt:572"},
			Coverage:      "old_capture_carry_forward_not_v292_proven",
		},
		{
			Version:         "V286",
			CivID:           58,
			CivName:         replay.CivDisplayName(58),
			RazesToVillager: intPtr(1),
			CastleKills:     intPtr(75),
			ImperialKills:   intPtr(400),
			UnitCount:       intPtr(80),
			SpawnSeconds:    intPtr(10),
			Notes:           "New Last Chieftains civ: Mapuche Standard with Kona cavalry.",
			Sources:         []string{"announcements_recent.txt:528", "announcements_recent.txt:533", "announcements_recent.txt:540"},
			Coverage:        "direct_recent_changelog_row",
		},
		{
			Version:         "V286",
			CivID:           57,
			CivName:         replay.CivDisplayName(57),
			RazesToVillager: intPtr(1),
			CastleKills:     intPtr(200),
			ImperialKills:   intPtr(450),
			UnitCount:       intPtr(80),
			SpawnSeconds:    intPtr(9),
			Notes:           "New Last Chieftains civ: Muisca Elite with Guecha Warrior archer.",
			Sources:         []string{"announcements_recent.txt:528", "announcements_recent.txt:535", "announcements_recent.txt:541"},
			Coverage:        "direct_recent_changelog_row",
		},
		{
			Version:         "V286",
			CivID:           59,
			CivName:         replay.CivDisplayName(59),
			RazesToVillager: intPtr(1),
			CastleKills:     intPtr(100),
			ImperialKills:   intPtr(400),
			UnitCount:       intPtr(80),
			SpawnSeconds:    intPtr(7),
			Notes:           "New Last Chieftains civ: Tupi Elite with Blackwood Archer.",
			Sources:         []string{"announcements_recent.txt:528", "announcements_recent.txt:537", "announcements_recent.txt:542"},
			Coverage:        "direct_recent_changelog_row",
		},
		{
			Version:          "V292",
			CivID:            58,
			CivName:          replay.CivDisplayName(58),
			RazesToVillager:  intPtr(1),
			CastleKills:      intPtr(100),
			ImperialKills:    intPtr(400),
			UnitCount:        intPtr(80),
			SpawnSeconds:     intPtr(10),
			Notes:            "V286 Mapuche carried forward, with V289 castle threshold 75 -> 100k.",
			Sources:          []string{"announcements_recent.txt:540", "announcements_recent.txt:552", "announcements_recent.txt:557"},
			Coverage:         "recent_changelog_carry_forward_for_v292",
			PreferredForV292: true,
		},
		{
			Version:          "V292",
			CivID:            57,
			CivName:          replay.CivDisplayName(57),
			RazesToVillager:  intPtr(1),
			CastleKills:      intPtr(250),
			ImperialKills:    intPtr(500),
			UnitCount:        intPtr(80),
			SpawnSeconds:     intPtr(9),
			Notes:            "V286 Muisca carried forward, with V289 200/450k -> 250/500k.",
			Sources:          []string{"announcements_recent.txt:541", "announcements_recent.txt:554", "announcements_recent.txt:557"},
			Coverage:         "recent_changelog_carry_forward_for_v292",
			PreferredForV292: true,
		},
		{
			Version:          "V292",
			CivID:            59,
			CivName:          replay.CivDisplayName(59),
			RazesToVillager:  intPtr(1),
			CastleKills:      intPtr(100),
			ImperialKills:    intPtr(400),
			UnitCount:        intPtr(80),
			SpawnSeconds:     intPtr(7),
			Notes:            "V286 Tupi row carried forward; no later Tupi change found before V292 in current captures.",
			Sources:          []string{"announcements_recent.txt:542", "announcements_recent.txt:557"},
			Coverage:         "recent_changelog_carry_forward_for_v292",
			PreferredForV292: true,
		},
		{
			Version:          "V292",
			CivID:            49,
			CivName:          replay.CivDisplayName(49),
			Notes:            "Shu unique unit works normally again; kill counter with slowness ability patched. No threshold values are assigned from this line.",
			Sources:          []string{"announcements_recent.txt:557", "announcements_recent.txt:559"},
			Coverage:         "direct_recent_changelog_row_no_numeric_threshold",
			PreferredForV292: true,
		},
		{
			Version:         "V293",
			CivID:           5,
			CivName:         replay.CivDisplayName(5),
			RazesToVillager: intPtr(2),
			Notes:           "Random Position / Team Free only: villager razes changed 3 -> 2.",
			Sources:         []string{"announcements_recent.txt:562", "announcements_recent.txt:565"},
			Coverage:        "direct_recent_changelog_row",
		},
		{
			Version:       "V293",
			CivID:         9,
			CivName:       replay.CivDisplayName(9),
			ImperialKills: intPtr(650),
			Notes:         "Random Position / Team Free only: imperial kills 600 -> 650k.",
			Sources:       []string{"announcements_recent.txt:562", "announcements_recent.txt:567"},
			Coverage:      "direct_recent_changelog_row",
		},
		{
			Version:       "V293",
			CivID:         23,
			CivName:       replay.CivDisplayName(23),
			CastleKills:   intPtr(150),
			ImperialKills: intPtr(400),
			Notes:         "Random Position / Team Free only: 200/500 -> 150/400k.",
			Sources:       []string{"announcements_recent.txt:562", "announcements_recent.txt:569"},
			Coverage:      "direct_recent_changelog_row",
		},
		{
			Version:       "V293",
			CivID:         49,
			CivName:       replay.CivDisplayName(49),
			ImperialKills: intPtr(350),
			Notes:         "Random Position / Team Free only: imperial kills 400 -> 350k.",
			Sources:       []string{"announcements_recent.txt:562", "announcements_recent.txt:571"},
			Coverage:      "direct_recent_changelog_row",
		},
		{
			Version:       "V293",
			CivID:         4,
			CivName:       replay.CivDisplayName(4),
			ImperialKills: intPtr(500),
			Notes:         "Random Position / Team Free only: imperial kills 600 -> 500k.",
			Sources:       []string{"announcements_recent.txt:562", "announcements_recent.txt:573"},
			Coverage:      "direct_recent_changelog_row",
		},
	}
}
