package scenario

import "fmt"

type IdiomReport struct {
	Path         string         `json:"path,omitempty"`
	Version      string         `json:"version"`
	Verification string         `json:"verification"`
	Count        int            `json:"count"`
	Idioms       []IdiomFinding `json:"idioms,omitempty"`
}

type IdiomFinding struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Confidence string   `json:"confidence"`
	Evidence   string   `json:"evidence"`
	Count      int      `json:"count,omitempty"`
	Samples    []string `json:"samples,omitempty"`
}

func (f *File) Idioms() IdiomReport {
	analysis := f.Analyze()
	report := IdiomReport{
		Path:         f.Path,
		Version:      f.Version,
		Verification: "structure_verified_not_engine_verified",
	}
	if analysis.DisplayTechniques.ColorMarkup.Detected {
		report.add(IdiomFinding{
			ID:         "text_color_markup",
			Name:       "Colored trigger text",
			Confidence: "structure_verified",
			Evidence:   "scenario string/text table contains AoE2DE color markup tags",
			Count:      len(analysis.DisplayTechniques.ColorMarkup.Counts),
		})
	}
	if analysis.DisplayTechniques.VariableSubstitutionHUD.Detected {
		report.add(IdiomFinding{
			ID:         "variable_substitution_hud",
			Name:       "Variable-backed HUD text",
			Confidence: "structure_verified",
			Evidence:   "display text contains variable substitution references",
			Count:      analysis.DisplayTechniques.VariableSubstitutionHUD.Count,
			Samples:    firstStrings(analysis.DisplayTechniques.VariableSubstitutionHUD.Refs, 8),
		})
	}
	if analysis.DisplayTechniques.CreateKillDisplayLoops.Detected {
		samples := make([]string, 0, len(analysis.DisplayTechniques.CreateKillDisplayLoops.Samples))
		for _, sample := range analysis.DisplayTechniques.CreateKillDisplayLoops.Samples {
			samples = append(samples, fmt.Sprintf("#%d %q create=%d kill=%d", sample.TriggerIndex, sample.TriggerName, sample.CreateObjects, sample.KillObjects))
		}
		report.add(IdiomFinding{
			ID:         "create_kill_display_loop",
			Name:       "Create/kill display loop",
			Confidence: "structure_verified",
			Evidence:   "same trigger contains create_object and kill/remove_object effects",
			Count:      analysis.DisplayTechniques.CreateKillDisplayLoops.Count,
			Samples:    samples,
		})
	}
	if analysis.DisplayTechniques.CaptionObjects.Detected {
		report.add(IdiomFinding{
			ID:         "caption_label_objects",
			Name:       "Caption label objects",
			Confidence: "structure_verified",
			Evidence:   "placed units carry caption string ids or literal caption strings",
			Count:      analysis.DisplayTechniques.CaptionObjects.Count,
		})
	}
	if count := f.effectCountByTypeName("script_call"); count > 0 {
		report.add(IdiomFinding{
			ID:         "xs_script_call_bridge",
			Name:       "XS script-call bridge",
			Confidence: "structure_verified",
			Evidence:   "trigger effects call XS functions through script_call",
			Count:      count,
			Samples:    effectTextSamples(f, "script_call", 8),
		})
	}
	relayCount := f.effectCountByTypeName("activate_trigger") + f.effectCountByTypeName("deactivate_trigger")
	if relayCount > 0 {
		report.add(IdiomFinding{
			ID:         "trigger_relay_graph",
			Name:       "Trigger relay graph",
			Confidence: "structure_verified",
			Evidence:   "triggers activate or deactivate other triggers",
			Count:      relayCount,
			Samples:    targetTriggerSamples(f, 8),
		})
	}
	timerCount := f.effectCountByTypeName("display_timer") + f.effectCountByTypeName("clear_timer")
	if timerCount > 0 {
		report.add(IdiomFinding{
			ID:         "visible_timer_choreography",
			Name:       "Visible timer choreography",
			Confidence: "structure_verified",
			Evidence:   "scenario uses display_timer or clear_timer effects",
			Count:      timerCount,
			Samples:    timerSamples(f, 8),
		})
	}
	for _, signal := range analysis.EconomySignals {
		report.add(IdiomFinding{
			ID:         "economy_" + signal.Name,
			Name:       "Economy signal: " + signal.Name,
			Confidence: signal.Confidence,
			Evidence:   signal.Evidence,
			Count:      signal.Count,
		})
	}
	report.Count = len(report.Idioms)
	return report
}

func (f *File) effectCountByTypeName(typeName string) int {
	if f.Triggers == nil {
		return 0
	}
	count := 0
	for _, trigger := range f.Triggers.Triggers {
		for _, effect := range trigger.EffectData {
			if effect.TypeName == typeName {
				count++
			}
		}
	}
	return count
}

func (r *IdiomReport) add(finding IdiomFinding) {
	r.Idioms = append(r.Idioms, finding)
}

func effectTextSamples(f *File, typeName string, limit int) []string {
	if f.Triggers == nil {
		return nil
	}
	var out []string
	for _, trigger := range f.Triggers.Triggers {
		for _, effect := range trigger.EffectData {
			if effect.TypeName != typeName || effect.Text == "" {
				continue
			}
			out = append(out, fmt.Sprintf("#%d %q %s", trigger.Index, trigger.Name, effect.Text))
			if len(out) >= limit {
				return out
			}
		}
	}
	return out
}

func targetTriggerSamples(f *File, limit int) []string {
	if f.Triggers == nil {
		return nil
	}
	var out []string
	for _, trigger := range f.Triggers.Triggers {
		for _, effect := range trigger.EffectData {
			if effect.TypeName != "activate_trigger" && effect.TypeName != "deactivate_trigger" {
				continue
			}
			out = append(out, fmt.Sprintf("#%d %q %s target=%d", trigger.Index, trigger.Name, effect.TypeName, effect.TargetTrigger))
			if len(out) >= limit {
				return out
			}
		}
	}
	return out
}

func timerSamples(f *File, limit int) []string {
	if f.Triggers == nil {
		return nil
	}
	var out []string
	for _, trigger := range f.Triggers.Triggers {
		for _, effect := range trigger.EffectData {
			if effect.TypeName != "display_timer" && effect.TypeName != "clear_timer" {
				continue
			}
			out = append(out, fmt.Sprintf("#%d %q %s timer=%d", trigger.Index, trigger.Name, effect.TypeName, effect.TimerID))
			if len(out) >= limit {
				return out
			}
		}
	}
	return out
}

func firstStrings(values []string, limit int) []string {
	if len(values) <= limit {
		return values
	}
	return values[:limit]
}
