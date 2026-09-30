package main

import (
	"strings"
	"testing"
)

// Regression: the first version of usageCommandNames accepted any token made of
// lowercase letters and dashes, so "kit fx bind --atlas file.png" parsed as the
// command "fx bind --atlas". A flag must end the command path.
func TestUsageCommandNamesStopsAtFlags(t *testing.T) {
	names := usageCommandNames(usageText())
	for _, name := range names {
		if strings.Contains(name, "--") {
			t.Errorf("command name %q contains a flag; the command path must stop at the first flag", name)
		}
	}
	found := false
	for _, name := range names {
		if name == "fx bind" {
			found = true
		}
	}
	if !found {
		t.Error(`expected "fx bind" to be extracted from usage(); a flag-bearing usage line must still yield its command path`)
	}
}

func TestUsageCaptureIsComplete(t *testing.T) {
	text := usageText()
	if !strings.Contains(text, "kit version") || !strings.Contains(text, "kit swatch patterns") {
		t.Error("usage capture is missing lines from the start or end of the help text")
	}
	if got := len(usageCommandNames(text)); got < 100 {
		t.Errorf("extracted only %d commands from usage(); expected the full catalog", got)
	}
}

func TestChecksumProbeUsageAdvertisesSemanticWordFlags(t *testing.T) {
	text := usageText()
	for _, want := range []string{"--state-sum-delta", "--carry-sum-delta", "--digest-delta"} {
		if !strings.Contains(text, want) {
			t.Fatalf("checksum-probe usage missing semantic flag %s", want)
		}
	}
	for _, stale := range []string{"--word3-delta N] [--word4-delta", "--score-candidate-delta"} {
		if strings.Contains(text, stale) {
			t.Fatalf("checksum-probe usage still advertises stale flag wording %q", stale)
		}
	}
}
