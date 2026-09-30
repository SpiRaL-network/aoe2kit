package jsonout

import (
	"bytes"
	"math"
	"strings"
	"testing"
)

func TestSanitizeNestedNonFiniteValues(t *testing.T) {
	type nested struct {
		Value *float64 `json:"value"`
	}
	nan := math.NaN()
	value := map[string]any{
		"rows": []nested{{Value: &nan}},
		"map":  map[string]float64{"positive": math.Inf(1), "negative": math.Inf(-1)},
	}
	sanitized, count := Sanitize(value)
	if count != 3 {
		t.Fatalf("count=%d, want 3", count)
	}
	var out bytes.Buffer
	if err := Encode(&out, value); err != nil {
		t.Fatal(err)
	}
	text := out.String()
	for _, want := range []string{`"NaN"`, `"Infinity"`, `"-Infinity"`, `"nonfinite_count": 3`} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %s in %s", want, text)
		}
	}
	if sanitized == nil {
		t.Fatal("sanitized value is nil")
	}
}

func TestEncodeFiniteValueMatchesIndentedJSON(t *testing.T) {
	value := map[string]any{"answer": 42, "text": "ok"}
	var out bytes.Buffer
	if err := Encode(&out, value); err != nil {
		t.Fatal(err)
	}
	want := "{\n  \"answer\": 42,\n  \"text\": \"ok\"\n}\n"
	if out.String() != want {
		t.Fatalf("output changed:\n%s", out.String())
	}
}
