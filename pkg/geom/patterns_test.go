package geom

import "testing"

func TestDefaultPatternsValidate(t *testing.T) {
	if len(Patterns) < 16 {
		t.Fatalf("expected at least 16 patterns, got %d", len(Patterns))
	}
	for _, name := range PatternNames() {
		frames, err := Generate(name)
		if err != nil {
			t.Fatalf("generate %s: %v", name, err)
		}
		validation := Validate(frames)
		if len(validation.Errors) != 0 {
			t.Fatalf("%s failed validation: %v", name, validation.Errors)
		}
	}
}

func TestUnknownPattern(t *testing.T) {
	if _, err := Generate("missing"); err == nil {
		t.Fatal("expected unknown pattern error")
	}
}
