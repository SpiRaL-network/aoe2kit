package cba

import (
	"fmt"
	"sort"
	"strings"

	"aoe2kit/pkg/replay"
)

type ProgressionReport struct {
	Path         string                     `json:"path,omitempty"`
	Method       string                     `json:"method"`
	Verification string                     `json:"verification"`
	Summary      ProgressionSummary         `json:"summary"`
	Players      []ProgressionPlayerSummary `json:"players,omitempty"`
	Production   []ProductionFirstSeen      `json:"production_first_seen,omitempty"`
	Research     []ResearchFirstSeen        `json:"research_first_seen,omitempty"`
	Warnings     []string                   `json:"warnings,omitempty"`
}

type ProgressionSummary struct {
	ProductionEvents      int `json:"production_events"`
	ResearchEvents        int `json:"research_events"`
	PlayersWithProduction int `json:"players_with_production"`
	PlayersWithResearch   int `json:"players_with_research"`
	FeudalProductionFlags int `json:"feudal_production_flags"`
	ImperialProxySignals  int `json:"imperial_proxy_signals"`
}

type ProgressionPlayerSummary struct {
	PlayerID             int    `json:"player_id"`
	Label                string `json:"label"`
	Name                 string `json:"name,omitempty"`
	CivID                int    `json:"civ_id,omitempty"`
	CivName              string `json:"civ_name,omitempty"`
	FirstProductionTime  string `json:"first_production_time,omitempty"`
	FirstProductionUnit  string `json:"first_production_unit,omitempty"`
	FirstImperialProxy   string `json:"first_imperial_proxy,omitempty"`
	FirstImperialProxyAt string `json:"first_imperial_proxy_at,omitempty"`
	FeudalProductionFlag bool   `json:"feudal_production_flag,omitempty"`
	FeudalProductionUnit string `json:"feudal_production_unit,omitempty"`
	FeudalProductionAt   string `json:"feudal_production_at,omitempty"`
	ProductionUnitTypes  int    `json:"production_unit_types"`
	ResearchTechnologies int    `json:"research_technologies"`
	Confidence           string `json:"confidence"`
}

type ProductionFirstSeen struct {
	PlayerID    int    `json:"player_id"`
	PlayerLabel string `json:"player_label,omitempty"`
	TimeMS      int    `json:"time_ms"`
	Time        string `json:"time"`
	UnitID      int    `json:"unit_id"`
	UnitName    string `json:"unit_name,omitempty"`
	Amount      int    `json:"amount,omitempty"`
	BuildingID  int    `json:"building_id,omitempty"`
	Signal      string `json:"signal,omitempty"`
	Confidence  string `json:"confidence"`
}

type ResearchFirstSeen struct {
	PlayerID     int    `json:"player_id"`
	PlayerLabel  string `json:"player_label,omitempty"`
	TimeMS       int    `json:"time_ms"`
	Time         string `json:"time"`
	TechnologyID int    `json:"technology_id"`
	Confidence   string `json:"confidence"`
}

func BuildProgression(path string) (*ProgressionReport, error) {
	rec, err := replay.Open(path)
	if err != nil {
		return nil, err
	}
	eventReport, err := replay.ExtractEvents(path, replay.EventOptions{IncludeUntypedAction: true})
	if err != nil {
		return nil, err
	}
	production := collectProductionFirstSeen(eventReport.Events)
	research := collectResearchFirstSeen(eventReport.Events)
	players := summarizeProgressionPlayers(rec.Players, production, research)
	report := &ProgressionReport{
		Path:         path,
		Method:       "de_queue_and_research_first_seen",
		Verification: "behavior_inferred_progression_proxy_not_direct_age_state",
		Summary: ProgressionSummary{
			ProductionEvents:      countEventsOfType(eventReport.Events, "de_queue"),
			ResearchEvents:        countEventsOfType(eventReport.Events, "research"),
			PlayersWithProduction: countPlayersWithProduction(production),
			PlayersWithResearch:   countPlayersWithResearch(research),
		},
		Players:    players,
		Production: production,
		Research:   research,
	}
	for _, player := range players {
		if player.FeudalProductionFlag {
			report.Summary.FeudalProductionFlags++
		}
		if player.FirstImperialProxy != "" {
			report.Summary.ImperialProxySignals++
		}
	}
	report.Warnings = append(report.Warnings, "age is not a direct command in this CBA replay; progression is inferred from production/research behavior")
	return report, nil
}

func collectProductionFirstSeen(events []replay.ReplayEvent) []ProductionFirstSeen {
	seen := map[string]ProductionFirstSeen{}
	for _, event := range events {
		if event.Type != "de_queue" || event.PlayerID <= 0 || event.UnitID <= 0 {
			continue
		}
		key := fmt.Sprintf("%d:%d", event.PlayerID, event.UnitID)
		if _, ok := seen[key]; ok {
			continue
		}
		name := replay.UnitDisplayName(event.UnitID)
		seen[key] = ProductionFirstSeen{
			PlayerID:    event.PlayerID,
			PlayerLabel: playerLabel(event.PlayerID),
			TimeMS:      event.TimeMS,
			Time:        replay.FormatTime(event.TimeMS),
			UnitID:      event.UnitID,
			UnitName:    name,
			Amount:      event.Amount,
			BuildingID:  event.BuildingID,
			Signal:      productionSignal(event.UnitID, name),
			Confidence:  "behavior_inferred_from_de_queue",
		}
	}
	out := make([]ProductionFirstSeen, 0, len(seen))
	for _, item := range seen {
		out = append(out, item)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].TimeMS != out[j].TimeMS {
			return out[i].TimeMS < out[j].TimeMS
		}
		if out[i].PlayerID != out[j].PlayerID {
			return out[i].PlayerID < out[j].PlayerID
		}
		return out[i].UnitID < out[j].UnitID
	})
	return out
}

func collectResearchFirstSeen(events []replay.ReplayEvent) []ResearchFirstSeen {
	seen := map[string]ResearchFirstSeen{}
	for _, event := range events {
		if event.Type != "research" || event.PlayerID <= 0 || event.TechnologyID <= 0 {
			continue
		}
		key := fmt.Sprintf("%d:%d", event.PlayerID, event.TechnologyID)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = ResearchFirstSeen{
			PlayerID:     event.PlayerID,
			PlayerLabel:  playerLabel(event.PlayerID),
			TimeMS:       event.TimeMS,
			Time:         replay.FormatTime(event.TimeMS),
			TechnologyID: event.TechnologyID,
			Confidence:   "behavior_inferred_from_research_action",
		}
	}
	out := make([]ResearchFirstSeen, 0, len(seen))
	for _, item := range seen {
		out = append(out, item)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].TimeMS != out[j].TimeMS {
			return out[i].TimeMS < out[j].TimeMS
		}
		if out[i].PlayerID != out[j].PlayerID {
			return out[i].PlayerID < out[j].PlayerID
		}
		return out[i].TechnologyID < out[j].TechnologyID
	})
	return out
}

func summarizeProgressionPlayers(players []replay.PlayerSlot, production []ProductionFirstSeen, research []ResearchFirstSeen) []ProgressionPlayerSummary {
	byPlayer := map[int]*ProgressionPlayerSummary{}
	for _, player := range players {
		if !player.Active || player.Number <= 0 {
			continue
		}
		byPlayer[player.Number] = &ProgressionPlayerSummary{
			PlayerID:   player.Number,
			Label:      playerLabel(player.Number),
			Name:       player.Name,
			CivID:      player.Civ,
			CivName:    replay.CivDisplayName(player.Civ),
			Confidence: "behavior_inferred_progression_proxy",
		}
	}
	prodTypes := map[int]map[int]bool{}
	researchTypes := map[int]map[int]bool{}
	for _, item := range production {
		summary := byPlayer[item.PlayerID]
		if summary == nil {
			continue
		}
		if prodTypes[item.PlayerID] == nil {
			prodTypes[item.PlayerID] = map[int]bool{}
		}
		prodTypes[item.PlayerID][item.UnitID] = true
		unit := unitLabel(item.UnitID, item.UnitName)
		if summary.FirstProductionTime == "" {
			summary.FirstProductionTime = item.Time
			summary.FirstProductionUnit = unit
		}
		if item.Signal == "imperial_proxy" && summary.FirstImperialProxy == "" {
			summary.FirstImperialProxy = unit
			summary.FirstImperialProxyAt = item.Time
		}
		if item.Signal == "early_or_feudal_unit" && !summary.FeudalProductionFlag {
			summary.FeudalProductionFlag = true
			summary.FeudalProductionUnit = unit
			summary.FeudalProductionAt = item.Time
		}
	}
	for _, item := range research {
		if researchTypes[item.PlayerID] == nil {
			researchTypes[item.PlayerID] = map[int]bool{}
		}
		researchTypes[item.PlayerID][item.TechnologyID] = true
	}
	out := make([]ProgressionPlayerSummary, 0, len(byPlayer))
	for playerID, summary := range byPlayer {
		summary.ProductionUnitTypes = len(prodTypes[playerID])
		summary.ResearchTechnologies = len(researchTypes[playerID])
		out = append(out, *summary)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].PlayerID < out[j].PlayerID })
	return out
}

func productionSignal(unitID int, name string) string {
	switch unitID {
	case 4, 74, 93:
		return "early_or_feudal_unit"
	case 1901, 1903:
		return "imperial_proxy"
	}
	if strings.HasPrefix(name, "Elite ") || strings.HasPrefix(name, "Imperial ") {
		return "high_tier_named_unit"
	}
	if strings.Contains(name, "Hand Cannoneer") {
		return "gunpowder_unit"
	}
	return ""
}

func unitLabel(unitID int, name string) string {
	if name == "" {
		name = replay.UnitDisplayName(unitID)
	}
	if name == "" {
		return fmt.Sprintf("%d", unitID)
	}
	return fmt.Sprintf("%d/%s", unitID, name)
}

func countEventsOfType(events []replay.ReplayEvent, eventType string) int {
	count := 0
	for _, event := range events {
		if event.Type == eventType {
			count++
		}
	}
	return count
}

func countPlayersWithProduction(production []ProductionFirstSeen) int {
	seen := map[int]bool{}
	for _, item := range production {
		seen[item.PlayerID] = true
	}
	return len(seen)
}

func countPlayersWithResearch(research []ResearchFirstSeen) int {
	seen := map[int]bool{}
	for _, item := range research {
		seen[item.PlayerID] = true
	}
	return len(seen)
}
