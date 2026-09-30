package main

import (
	"fmt"
	"os"
	"sort"
	"strings"
)

// commandsCommand exposes the CLI's own command catalog: what each command is
// called, what it consumes, and whether running it is safe for a caller that
// only wants to read. Tooling that needs to enumerate or automate the kit reads
// this instead of parsing usage text or maintaining its own list.
func commandsCommand(args []string) {
	text := false
	trait := ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--text":
			text = true
		case "--json":
			text = false
		case "--trait":
			if i+1 < len(args) {
				trait = args[i+1]
				i++
			}
		}
	}

	specs := make([]CommandSpec, 0, len(commandCatalog))
	for _, spec := range commandCatalog {
		if trait != "" && spec.Trait != trait {
			continue
		}
		specs = append(specs, spec)
	}
	sort.SliceStable(specs, func(i, j int) bool { return specs[i].Name < specs[j].Name })

	if !text {
		printJSON(map[string]any{
			"tool":     "aoe2kit",
			"count":    len(specs),
			"traits":   []string{TraitReadOnly, TraitWrites, TraitNetwork},
			"note":     "trait declares whether running a command is side-effect free; input names the file kind it consumes",
			"commands": specs,
		})
		return
	}
	for _, spec := range specs {
		fmt.Printf("%-28s %-10s %-9s %s\n", spec.Name, spec.Trait, spec.Input, spec.Usage)
	}
}

// usageCommandNames extracts every documented command from usage(), so tests can
// hold the catalog and the help text to each other.
func usageCommandNames(usage string) []string {
	var names []string
	seen := map[string]bool{}
	for _, line := range strings.Split(usage, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "kit ") {
			continue
		}
		var path []string
		for _, tok := range strings.Fields(trimmed)[1:] {
			if !isPlainCommandToken(tok) {
				break
			}
			path = append(path, tok)
		}
		if len(path) == 0 {
			continue
		}
		name := strings.Join(path, " ")
		if !seen[name] {
			seen[name] = true
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

func isPlainCommandToken(tok string) bool {
	if tok == "" {
		return false
	}
	// A command name starts with a letter; anything beginning with '-' is a flag
	// and ends the command path.
	if tok[0] < 'a' || tok[0] > 'z' {
		return false
	}
	for _, r := range tok {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '-' {
			return false
		}
	}
	return true
}

// usageText returns the help text as a string for catalog verification.
func usageText() string {
	read, write, err := os.Pipe()
	if err != nil {
		return ""
	}
	saved := os.Stderr
	os.Stderr = write
	usage()
	os.Stderr = saved
	write.Close()

	var b strings.Builder
	buf := make([]byte, 4096)
	for {
		n, readErr := read.Read(buf)
		if n > 0 {
			b.Write(buf[:n])
		}
		if readErr != nil {
			break
		}
	}
	read.Close()
	return b.String()
}
