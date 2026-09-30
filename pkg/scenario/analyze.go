package scenario

import (
	"sort"
	"strings"
)

type AnalysisReport struct {
	Path              string              `json:"path,omitempty"`
	Version           string              `json:"version"`
	Verification      string              `json:"verification"`
	DataSet           ScenarioDataSet     `json:"data_set"`
	Scale             ScenarioScale       `json:"scale"`
	DisplayTechniques DisplayTechniques   `json:"display_techniques"`
	EconomySignals    []EconomySignal     `json:"economy_signals,omitempty"`
	Watermarks        []WatermarkSignal   `json:"watermarks,omitempty"`
	EffectTypes       []EffectTypeSummary `json:"effect_types,omitempty"`
}

type ScenarioDataSet struct {
	Status     string `json:"status"`
	Source     string `json:"source"`
	Confidence string `json:"confidence"`
	Note       string `json:"note,omitempty"`
}

type ScenarioScale struct {
	MapWidth       int            `json:"map_width"`
	MapHeight      int            `json:"map_height"`
	TriggerCount   int            `json:"trigger_count"`
	EnabledCount   int            `json:"enabled_count"`
	VariableCount  int            `json:"variable_count"`
	UnitTotal      int            `json:"unit_total"`
	UnitsByPlayer  map[int]int    `json:"units_by_player"`
	EffectCount    int            `json:"effect_count"`
	EffectTypeBins map[string]int `json:"effect_type_bins"`
	PlayerCount    int            `json:"player_count"`
	Sections       []SectionInfo  `json:"sections,omitempty"`
}

type DisplayTechniques struct {
	ColorMarkup             MarkupColorTechnique    `json:"color_markup"`
	VariableSubstitutionHUD VariableHUDTechnique    `json:"variable_substitution_hud"`
	CreateKillDisplayLoops  CreateKillLoopTechnique `json:"create_kill_display_loops"`
	CaptionObjects          CaptionTechnique        `json:"caption_overhead_label_objects"`
}

type MarkupColorTechnique struct {
	Detected bool           `json:"detected"`
	Counts   map[string]int `json:"counts,omitempty"`
}

type VariableHUDTechnique struct {
	Detected bool     `json:"detected"`
	Refs     []string `json:"refs,omitempty"`
	Count    int      `json:"count"`
}

type CreateKillLoopTechnique struct {
	Detected bool                   `json:"detected"`
	Count    int                    `json:"count"`
	Samples  []CreateKillLoopSample `json:"samples,omitempty"`
}

type CreateKillLoopSample struct {
	TriggerIndex  int    `json:"trigger_index"`
	TriggerName   string `json:"trigger_name,omitempty"`
	CreateObjects int    `json:"create_objects"`
	KillObjects   int    `json:"kill_objects"`
}

type CaptionTechnique struct {
	Detected bool `json:"detected"`
	Count    int  `json:"count"`
}

type EconomySignal struct {
	Name       string `json:"name"`
	Confidence string `json:"confidence"`
	Count      int    `json:"count,omitempty"`
	Evidence   string `json:"evidence,omitempty"`
}

type WatermarkSignal struct {
	Kind       string   `json:"kind"`
	Confidence string   `json:"confidence"`
	Terms      []string `json:"terms,omitempty"`
	Triggers   []string `json:"triggers,omitempty"`
}

type EffectTypeSummary struct {
	TypeName string `json:"type_name"`
	Count    int    `json:"count"`
}

func (f *File) Analyze() AnalysisReport {
	stringsReport := f.Strings()
	effects := f.Effects()
	report := AnalysisReport{
		Path:         f.Path,
		Version:      f.Version,
		Verification: "structure_verified_not_engine_verified",
		DataSet: ScenarioDataSet{
			Status:     "vanilla",
			Source:     "scenario_file",
			Confidence: "inferred",
			Note:       "AoE2 scenario files do not carry the replay active data-set block; no active data mod is declared in the scenario structure.",
		},
		Scale:             f.analysisScale(effects),
		DisplayTechniques: f.analysisDisplayTechniques(stringsReport),
		EconomySignals:    f.analysisEconomySignals(stringsReport, effects),
		Watermarks:        f.analysisWatermarks(),
	}
	for _, bucket := range effects.Types {
		report.EffectTypes = append(report.EffectTypes, EffectTypeSummary{TypeName: bucket.TypeName, Count: bucket.Count})
	}
	return report
}

func (f *File) analysisScale(effects EffectsReport) ScenarioScale {
	scale := ScenarioScale{
		UnitsByPlayer:  map[int]int{},
		EffectTypeBins: map[string]int{},
		Sections:       f.Sections,
	}
	if f.Map != nil {
		scale.MapWidth = f.Map.Width
		scale.MapHeight = f.Map.Height
	}
	if f.Triggers != nil {
		scale.TriggerCount = f.Triggers.Count
		scale.VariableCount = f.Triggers.Variables
		for _, trigger := range f.Triggers.Triggers {
			if trigger.Enabled != 0 {
				scale.EnabledCount++
			}
		}
	}
	if f.Units != nil {
		scale.UnitTotal = f.Units.Total
		for _, player := range f.Units.Sections {
			scale.UnitsByPlayer[player.Player] = player.Count
		}
	}
	for _, player := range f.Players {
		if player.Active {
			scale.PlayerCount++
		}
	}
	scale.EffectCount = effects.Total
	for _, bucket := range effects.Types {
		scale.EffectTypeBins[bucket.TypeName] = bucket.Count
	}
	return scale
}

func (f *File) analysisDisplayTechniques(stringsReport StringReport) DisplayTechniques {
	loops := f.createKillLoops()
	captions := f.captionObjectCount()
	return DisplayTechniques{
		ColorMarkup: MarkupColorTechnique{
			Detected: len(stringsReport.MarkupSummary.ColorTags) > 0,
			Counts:   stringsReport.MarkupSummary.ColorTags,
		},
		VariableSubstitutionHUD: VariableHUDTechnique{
			Detected: len(stringsReport.MarkupSummary.VariableRefs) > 0,
			Refs:     stringsReport.MarkupSummary.VariableRefs,
			Count:    len(stringsReport.MarkupSummary.VariableRefs),
		},
		CreateKillDisplayLoops: CreateKillLoopTechnique{
			Detected: len(loops) > 0,
			Count:    len(loops),
			Samples:  firstCreateKillLoops(loops, 8),
		},
		CaptionObjects: CaptionTechnique{
			Detected: captions > 0,
			Count:    captions,
		},
	}
}

func (f *File) createKillLoops() []CreateKillLoopSample {
	if f.Triggers == nil {
		return nil
	}
	var out []CreateKillLoopSample
	for _, trigger := range f.Triggers.Triggers {
		creates := 0
		kills := 0
		for _, effect := range trigger.EffectData {
			switch effect.TypeName {
			case "create_object":
				creates++
			case "kill_object", "remove_object":
				kills++
			}
		}
		if creates > 0 && kills > 0 {
			out = append(out, CreateKillLoopSample{
				TriggerIndex:  trigger.Index,
				TriggerName:   trigger.Name,
				CreateObjects: creates,
				KillObjects:   kills,
			})
		}
	}
	return out
}

func firstCreateKillLoops(in []CreateKillLoopSample, n int) []CreateKillLoopSample {
	if len(in) <= n {
		return in
	}
	return in[:n]
}

func (f *File) captionObjectCount() int {
	if f.Units == nil {
		return 0
	}
	count := 0
	for _, player := range f.Units.Sections {
		for _, unit := range player.Units {
			if unit.CaptionString != "" || unit.CaptionStringID > 0 {
				count++
			}
		}
	}
	return count
}

func (f *File) analysisEconomySignals(stringsReport StringReport, effects EffectsReport) []EconomySignal {
	var out []EconomySignal
	if count := effectCountByName(effects, "tribute"); count > 0 {
		out = append(out, EconomySignal{Name: "tribute_resource_flow", Confidence: "structure_verified", Count: count, Evidence: "tribute effects present"})
	}
	if count := effectCountByName(effects, "change_variable") + effectCountByName(effects, "modify_variable"); count > 0 {
		out = append(out, EconomySignal{Name: "scenario_variable_accounting", Confidence: "structure_verified", Count: count, Evidence: "variable effects present"})
	}
	textBlob := strings.ToLower(strings.Join(stringsReport.allText(), "\n"))
	if strings.Contains(textBlob, "coin") {
		out = append(out, EconomySignal{Name: "coin_reward_language", Confidence: "heuristic_text", Count: strings.Count(textBlob, "coin"), Evidence: "scenario text mentions coins"})
	}
	if strings.Contains(textBlob, "kills") || strings.Contains(textBlob, "kill") {
		out = append(out, EconomySignal{Name: "kill_threshold_language", Confidence: "heuristic_text", Count: strings.Count(textBlob, "kill"), Evidence: "scenario text mentions kill thresholds or kill counters"})
	}
	return out
}

func effectCountByName(effects EffectsReport, name string) int {
	for _, bucket := range effects.Types {
		if bucket.TypeName == name {
			return bucket.Count
		}
	}
	return 0
}

func (f *File) analysisWatermarks() []WatermarkSignal {
	if f.Triggers == nil {
		return nil
	}
	terms := []string{"copyright", "do not modify", "unapproved modifications", "forbidden", "spiral", "steamcommunity"}
	foundTerms := map[string]bool{}
	var triggers []string
	for _, trigger := range f.Triggers.Triggers {
		lower := strings.ToLower(trigger.Name)
		for _, term := range terms {
			if strings.Contains(lower, term) {
				foundTerms[term] = true
				if trigger.Name != "" {
					triggers = append(triggers, trigger.Name)
				}
				break
			}
		}
	}
	if len(foundTerms) == 0 {
		return nil
	}
	sort.Strings(triggers)
	outTerms := make([]string, 0, len(foundTerms))
	for term := range foundTerms {
		outTerms = append(outTerms, term)
	}
	sort.Strings(outTerms)
	if len(triggers) > 12 {
		triggers = triggers[:12]
	}
	return []WatermarkSignal{{
		Kind:       "trigger_name_signature_block",
		Confidence: "heuristic_name_match",
		Terms:      outTerms,
		Triggers:   triggers,
	}}
}
