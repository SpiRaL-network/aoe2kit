package kit

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestManifestCoversCLIUsageChoices(t *testing.T) {
	source := readKitMainSource(t)
	commands := manifestCommands()

	assertManifestCoversChoices(t, source, "kit replay ", commands["kit replay"])
	assertManifestCoversChoices(t, source, "kit scen ", commands["kit scen"])
	assertManifestCoversChoices(t, source, "kit dat ", commands["kit dat"])
	assertManifestCoversChoices(t, source, "kit ai ", commands["kit ai"])

	requireManifestSubcommands(t, "kit", commands["kit"], []string{"verify-run", "cba"})
	requireManifestSubcommands(t, "kit cba", commands["kit cba"], []string{
		"balance", "trigger-razes", "trigger-spawns", "replay", "progression", "razes", "perf",
		"registry", "ingest", "archive", "axes", "export",
	})
}

func readKitMainSource(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "cmd", "kit", "main.go"))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func manifestCommands() map[string]map[string]bool {
	out := map[string]map[string]bool{}
	for _, command := range CurrentManifest().Commands {
		set := map[string]bool{}
		for _, subcommand := range command.Subcommands {
			set[subcommand] = true
		}
		out[command.Name] = set
	}
	return out
}

func assertManifestCoversChoices(t *testing.T, source, usagePrefix string, manifest map[string]bool) {
	t.Helper()
	if manifest == nil {
		t.Fatalf("missing manifest command for %s", strings.TrimSpace(usagePrefix))
	}
	choices := longestUsageChoiceSet(source, usagePrefix)
	for _, choice := range choices {
		if !manifest[choice] {
			t.Fatalf("manifest for %s missing CLI choice %q from usage", strings.TrimSpace(usagePrefix), choice)
		}
	}
}

func longestUsageChoiceSet(source, usagePrefix string) []string {
	re := regexp.MustCompile(regexp.QuoteMeta(usagePrefix) + `<([^>\n]+)>`)
	matches := re.FindAllStringSubmatch(source, -1)
	var best []string
	for _, match := range matches {
		choices := strings.Split(match[1], "|")
		if len(choices) > len(best) {
			best = choices
		}
	}
	return best
}

func requireManifestSubcommands(t *testing.T, name string, manifest map[string]bool, required []string) {
	t.Helper()
	if manifest == nil {
		t.Fatalf("missing manifest command for %s", name)
	}
	for _, subcommand := range required {
		if !manifest[subcommand] {
			t.Fatalf("manifest for %s missing %q", name, subcommand)
		}
	}
}
