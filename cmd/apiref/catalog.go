package main

// One-time bootstrap of the CLI's command catalog.
//
// Classification used to live here as hand-maintained verb lists, which meant a
// command added to the CLI silently fell out of the documentation until someone
// remembered to update a list in a different program. The catalog moves that
// knowledge next to the dispatch it describes, where a test can enforce it.
//
// This writes the initial file from what the generator already knew. After that
// the catalog is maintained in cmd/kit, and this path exists only to re-bootstrap.

import (
	"fmt"
	"os"
	"sort"
	"strings"
)

func writeCatalog(path string, commands []Command) error {
	var b strings.Builder
	b.WriteString(`package main

// Command traits, declared beside the dispatch they describe.
//
// Bootstrapped once by "apiref --emit-catalog"; maintained here by hand.
// TestCommandCatalogCoversUsage fails if a command in usage() has no entry, so
// adding a command to the CLI forces a decision about what it does.
//
//	trait: read_only writes files/registries, or performs network I/O.
//	input: the kind of file the command consumes, so tooling need not guess.

type CommandSpec struct {
	Name  string ` + "`json:\"name\"`" + `
	Usage string ` + "`json:\"usage\"`" + `
	Trait string ` + "`json:\"trait\"`" + `
	Input string ` + "`json:\"input\"`" + `
}

const (
	TraitReadOnly = "read_only"
	TraitWrites   = "writes"
	TraitNetwork  = "network"
)

var commandCatalog = []CommandSpec{
`)

	sorted := append([]Command{}, commands...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].Name < sorted[j].Name })
	for _, c := range sorted {
		trait := "TraitReadOnly"
		switch {
		case c.Network:
			trait = "TraitNetwork"
		case c.Mutates:
			trait = "TraitWrites"
		}
		fmt.Fprintf(&b, "\t{Name: %q, Usage: %q, Trait: %s, Input: %q},\n",
			c.Name, c.Usage, trait, inputKind(c))
	}
	b.WriteString("}\n")
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

// inputKind names the fixture a command consumes, replacing the generator's
// per-command guesswork with a declared value.
func inputKind(c Command) string {
	usage := c.Usage
	switch {
	case len(c.Subcommands) > 0:
		return "group"
	case strings.Contains(usage, "empires*.dat") || strings.Contains(usage, "<empires") ||
		strings.Contains(usage, "<in.dat>") || strings.Contains(usage, "<base.dat>"):
		return "dat"
	case strings.Contains(usage, ".xsdat"):
		return "xsdat"
	case strings.Contains(usage, ".aoe2scenario"):
		return "scenario"
	case strings.Contains(usage, ".aoe2record") || strings.Contains(usage, "<path>"):
		return "replay"
	case strings.Contains(usage, "[folder]") || strings.Contains(usage, "<folder"):
		return "folder"
	case len(c.Positional) == 0:
		return "none"
	}
	return "other"
}
