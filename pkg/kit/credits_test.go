package kit

import (
	"bytes"
	"testing"
)

func TestCreditsNameAGE(t *testing.T) {
	doc, err := Credits("../..")
	if err != nil {
		t.Fatal(err)
	}
	if doc.License != "LGPL-3.0-only" {
		t.Fatalf("license = %q, want LGPL-3.0-only", doc.License)
	}
	for _, ref := range doc.References {
		if ref.Name == "Advanced Genie Editor (AGE)" {
			if ref.License != "GPL-3.0" {
				t.Fatalf("AGE license = %q, want GPL-3.0", ref.License)
			}
			if ref.RungEarned != "credited" || ref.ThanksStatus != "owed" {
				t.Fatalf("AGE gratitude state = rung %q thanks %q, want credited/owed", ref.RungEarned, ref.ThanksStatus)
			}
			if ref.Credit == "" {
				t.Fatal("AGE credit must not be empty")
			}
			return
		}
	}
	t.Fatal("credits must name Advanced Genie Editor (AGE)")
}

func TestCreditsNoticeIsGeneratedFromLedger(t *testing.T) {
	doc, err := Credits("../..")
	if err != nil {
		t.Fatal(err)
	}
	notice := CreditsNoticeMarkdown(doc)
	for _, needle := range [][]byte{
		[]byte("This notice is generated from `data/credits.json`"),
		[]byte("Advanced Genie Editor (AGE)"),
		[]byte("Thanks status: owed"),
		[]byte("LGPL-3.0"),
	} {
		if !bytes.Contains(notice, needle) {
			t.Fatalf("generated notice missing %q", needle)
		}
	}
}
