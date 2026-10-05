package scenario

import (
	"path/filepath"
	"strings"
	"testing"
)

// Player records, diplomacy rows, and resource rows are zero-based (index 0 is P1);
// labels shown to people must be one-based.
func TestPlayerRecordLabelsAreOneBased(t *testing.T) {
	path := filepath.Join(t.TempDir(), "blank.aoe2scenario")
	report, err := WriteBlankScenarioFile(path, BlankOptions{PlayerCount: 2})
	if err != nil {
		t.Fatalf("WriteBlankScenarioFile: %v", err)
	}
	if !strings.HasPrefix(report.EditorParityNote, "P2 human=false") {
		t.Fatalf("editor parity note names the wrong slot: %q", report.EditorParityNote)
	}

	file, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	plan, err := file.Plan(Recipe{
		Players:   []PlayerRecipe{{Player: 0}},
		Diplomacy: []DiplomacyRecipe{{From: 0, To: 1, Stance: 3}},
		Resources: []ResourceRecipe{{Player: 1}},
	})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	want := map[string]string{"set_player": "P1", "set_diplomacy": "P1->P2=3", "set_resources": "P2"}
	for _, op := range plan.Operations {
		if label, ok := want[op.Op]; ok {
			if op.Name != label {
				t.Errorf("%s labeled %q, want %q", op.Op, op.Name, label)
			}
			delete(want, op.Op)
		}
	}
	for op := range want {
		t.Errorf("plan has no %s operation", op)
	}

	found := false
	for _, issue := range file.Lint().Issues {
		if issue.Code == "active_ai_without_name" {
			found = true
			if !strings.HasPrefix(issue.Message, "P2 ") {
				t.Errorf("active_ai_without_name names the wrong slot: %q", issue.Message)
			}
		}
	}
	if !found {
		t.Error("blank P2 computer slot raised no active_ai_without_name warning")
	}
}
