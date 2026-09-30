package kit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type CreditsDocument struct {
	Schema       string        `json:"schema"`
	Project      string        `json:"project"`
	License      string        `json:"license"`
	GeneratedBy  string        `json:"generated_by"`
	Dependencies []CreditEntry `json:"dependencies"`
	References   []CreditEntry `json:"references"`
}

type CreditEntry struct {
	ID              string   `json:"id"`
	Category        string   `json:"category"`
	Name            string   `json:"name"`
	Authors         []string `json:"authors,omitempty"`
	URL             string   `json:"url,omitempty"`
	License         string   `json:"license,omitempty"`
	Permission      string   `json:"permission,omitempty"`
	Gave            string   `json:"gave"`
	Role            string   `json:"role"`
	Relationship    string   `json:"relationship"`
	RungEarned      string   `json:"rung_earned"`
	ThanksStatus    string   `json:"thanks_status"`
	GiveBackStatus  string   `json:"give_back_status,omitempty"`
	CorrectionState string   `json:"correction_state"`
	Credit          string   `json:"credit"`
	Notes           []string `json:"notes,omitempty"`
	CreditClass     string   `json:"credit_class,omitempty"`
}

func Credits(root string) (CreditsDocument, error) {
	if root == "" {
		root = "."
	}
	data, err := os.ReadFile(filepath.Join(root, "data", "credits.json"))
	if err != nil {
		return CreditsDocument{}, err
	}
	var doc CreditsDocument
	if err := json.Unmarshal(data, &doc); err != nil {
		return CreditsDocument{}, fmt.Errorf("decode credits: %w", err)
	}
	return doc, nil
}

func CreditsJSON(root string) ([]byte, error) {
	doc, err := Credits(root)
	if err != nil {
		return nil, err
	}
	return json.MarshalIndent(doc, "", "  ")
}

func CreditsNoticeMarkdown(doc CreditsDocument) []byte {
	var b bytes.Buffer
	fmt.Fprintf(&b, "# Notices and Credits\n\n")
	fmt.Fprintf(&b, "%s is licensed %s.\n\n", doc.Project, doc.License)
	fmt.Fprintf(&b, "This notice is generated from `data/credits.json`, the credit source of truth. ")
	fmt.Fprintf(&b, "If a source, author, project, or community should be credited and is missing, that is a bug.\n\n")
	fmt.Fprintf(&b, "## Credit Principle\n\n")
	fmt.Fprintf(&b, "Attribution is abundant and automatic. Personal thanks is scarce, deliberate, and human. ")
	fmt.Fprintf(&b, "The ledger keeps unpaid gratitude visible instead of pretending it has been paid.\n\n")

	section := func(title, intro, wantClass string) {
		var items []CreditEntry
		for _, e := range doc.References {
			cc := e.CreditClass
			if cc == "" {
				cc = "reference"
			}
			if cc == wantClass {
				items = append(items, e)
			}
		}
		if len(items) == 0 {
			return
		}
		fmt.Fprintf(&b, "## %s\n\n", title)
		fmt.Fprintf(&b, "%s\n\n", intro)
		for _, entry := range items {
			writeCreditEntry(&b, entry)
		}
	}
	section("Built On",
		"Published work studied, built on, or used as a reference. Attribution is abundant and automatic.",
		"reference")
	section("Personal Thanks",
		"Individual people who have directly, personally helped the author. Scarce, deliberate, and human.",
		"personal_thanks")
	return b.Bytes()
}

func writeCreditEntry(b *bytes.Buffer, entry CreditEntry) {
	fmt.Fprintf(b, "### %s\n\n", entry.Name)
	if len(entry.Authors) > 0 {
		fmt.Fprintf(b, "- Authors: %s\n", strings.Join(entry.Authors, ", "))
	}
	if entry.URL != "" {
		fmt.Fprintf(b, "- URL: %s\n", entry.URL)
	}
	if entry.License != "" {
		fmt.Fprintf(b, "- License: %s\n", entry.License)
	}
	if entry.Permission != "" {
		fmt.Fprintf(b, "- Permission: %s\n", entry.Permission)
	}
	fmt.Fprintf(b, "- Gave: %s\n", entry.Gave)
	fmt.Fprintf(b, "- Role: %s\n", entry.Role)
	fmt.Fprintf(b, "- Relationship: %s\n", entry.Relationship)
	fmt.Fprintf(b, "- Rung earned: %s\n", entry.RungEarned)
	fmt.Fprintf(b, "- Thanks status: %s\n", entry.ThanksStatus)
	if entry.GiveBackStatus != "" {
		fmt.Fprintf(b, "- Give-back status: %s\n", entry.GiveBackStatus)
	}
	fmt.Fprintf(b, "- Correction state: %s\n\n", entry.CorrectionState)
	fmt.Fprintf(b, "%s\n\n", entry.Credit)
	if len(entry.Notes) > 0 {
		fmt.Fprintf(b, "Notes:\n")
		for _, note := range entry.Notes {
			fmt.Fprintf(b, "- %s\n", note)
		}
		fmt.Fprintf(b, "\n")
	}
}
