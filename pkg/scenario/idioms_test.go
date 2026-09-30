package scenario

import "testing"

func TestIdiomsDetectStructuralPatterns(t *testing.T) {
	file := &File{
		Version: "1.58",
		Triggers: &TriggerInfo{Triggers: []TriggerSummary{{
			Index:   7,
			Name:    "Bridge",
			Looping: 0,
			EffectData: []EffectSummary{
				{TypeName: "script_call", Text: "main();"},
				{TypeName: "activate_trigger", TargetTrigger: 8},
				{TypeName: "display_timer", TimerID: 2},
			},
		}, {
			Index: 8,
			Name:  "Display",
			EffectData: []EffectSummary{
				{TypeName: "create_object"},
				{TypeName: "kill_object"},
			},
		}}},
	}
	report := file.Idioms()
	for _, id := range []string{
		"xs_script_call_bridge",
		"trigger_relay_graph",
		"visible_timer_choreography",
		"create_kill_display_loop",
	} {
		if !hasIdiom(report, id) {
			t.Fatalf("missing idiom %q in %+v", id, report.Idioms)
		}
	}
}

func hasIdiom(report IdiomReport, id string) bool {
	for _, idiom := range report.Idioms {
		if idiom.ID == id {
			return true
		}
	}
	return false
}
