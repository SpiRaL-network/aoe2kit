package preview

import "fmt"

// CheckSpriteCompleteness ignores intentional exclusions, but not a missing,
// undecodable, unsupported, or invalid-variant sprite in the requested view.
func CheckSpriteCompleteness(r Report) error {
	count := 0
	for _, s := range r.Skipped {
		if s.Reason != "outside crop" && s.Reason != "particle: outside crop" && s.Reason != "objects layer disabled" {
			count++
		}
	}
	if count > 0 {
		return fmt.Errorf("strict sprite preview: %d object(s) could not be rendered; inspect skipped", count)
	}
	return nil
}
