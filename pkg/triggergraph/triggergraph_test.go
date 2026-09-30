package triggergraph

import "testing"

func TestReadTriggerStringTreatsNegativeOneAsEmptySentinel(t *testing.T) {
	got, next, err := readTriggerString([]byte{0xff, 0xff, 0xff, 0xff, 0x7b}, 0)
	if err != nil {
		t.Fatalf("readTriggerString sentinel: %v", err)
	}
	if got != "" {
		t.Fatalf("string = %q, want empty", got)
	}
	if next != 4 {
		t.Fatalf("next = %d, want 4", next)
	}
}
