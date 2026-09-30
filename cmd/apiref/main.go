// apiref generates the AoE2Kit API reference by interrogating the kit binary
// itself: invocation lines come from kit's own usage output, and output
// contracts come from probing read-only commands against real fixtures.
//
// Nothing in the generated reference is hand-asserted. A command that could not
// be probed is labeled as such rather than described from assumption.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"time"
)

type Command struct {
	Path        []string `json:"path"`
	Name        string   `json:"name"`
	Group       string   `json:"group,omitempty"`
	Usage       string   `json:"usage"`
	Flags       []Flag   `json:"flags,omitempty"`
	Positional  []string `json:"positional,omitempty"`
	Subcommands []string `json:"subcommands,omitempty"`
	Mutates     bool     `json:"mutates"`
	Network     bool     `json:"network"`
	Trait       string   `json:"trait,omitempty"`
	Input       string   `json:"input,omitempty"`

	ProbeStatus  string   `json:"probe_status"`
	ProbeArgs    []string `json:"probe_args,omitempty"`
	ExitCode     int      `json:"probe_exit_code,omitempty"`
	OutputKeys   []string `json:"output_top_level_keys,omitempty"`
	ResponseType string   `json:"response_type,omitempty"`
	Verification string   `json:"verification,omitempty"`
	Method       string   `json:"method,omitempty"`
	ProbeNote    string   `json:"probe_note,omitempty"`
}

type Flag struct {
	Name     string `json:"name"`
	Value    string `json:"value_hint,omitempty"`
	Required bool   `json:"required,omitempty"`
}

type Reference struct {
	Tool      string    `json:"tool"`
	Version   string    `json:"version"`
	Generated string    `json:"generated_by"`
	Fixtures  Fixtures  `json:"probe_fixtures"`
	CommandsN int       `json:"command_count"`
	ProbedN   int       `json:"probed_count"`
	Commands  []Command `json:"commands"`
}

type Fixtures struct {
	Scenario string `json:"scenario,omitempty"`
	Replay   string `json:"replay,omitempty"`
	Dat      string `json:"dat,omitempty"`
	Xsdat    string `json:"xsdat,omitempty"`
	Folder   string `json:"folder,omitempty"`
}

// CatalogEntry mirrors one row of the CLI's declared command catalog.
type CatalogEntry struct {
	Name  string `json:"name"`
	Usage string `json:"usage"`
	Trait string `json:"trait"`
	Input string `json:"input"`
}

// loadCatalog asks the binary what its commands are and what they do. This
// replaces the verb lists this generator used to maintain: the CLI is the only
// thing that can know whether a new command writes, so it is the only thing
// that should say so.
func loadCatalog(kitPath string) map[string]CatalogEntry {
	out, err := runCapture(kitPath, []string{"commands", "--json"}, 15*time.Second)
	if err != nil && out == "" {
		return nil
	}
	var payload struct {
		Commands []CatalogEntry `json:"commands"`
	}
	if json.Unmarshal([]byte(out), &payload) != nil {
		return nil
	}
	byName := make(map[string]CatalogEntry, len(payload.Commands))
	for _, entry := range payload.Commands {
		byName[entry.Name] = entry
	}
	return byName
}

var (
	flagRe  = regexp.MustCompile(`^--[a-z][a-z0-9-]*$`)
	identRe = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
	groupRe = regexp.MustCompile(`^<([a-z0-9-]+\|[a-z0-9|-]+)>$`)
)

func main() {
	kitPath := flag.String("kit", "./kit", "path to the kit binary")
	scenario := flag.String("scenario", "", "scenario fixture for probing")
	replay := flag.String("replay", "", "replay fixture for probing")
	datFile := flag.String("dat", "", "dat fixture for probing")
	xsdat := flag.String("xsdat", "", "xsdat fixture for probing")
	folder := flag.String("folder", ".", "folder fixture for probing")
	outMD := flag.String("out-md", "docs/API_REFERENCE.md", "markdown output path")
	outJSON := flag.String("out-json", "docs/api_reference.json", "json output path")
	outSchemas := flag.String("out-schemas", "docs/api_schemas.json", "full nested response schemas, extracted from source")
	outGoDoc := flag.String("out-godoc", "docs/PACKAGE_API.md", "package API rendered by go doc")
	timeout := flag.Duration("timeout", 25*time.Second, "per-probe timeout")
	emitCatalog := flag.String("emit-catalog", "", "bootstrap the CLI's command catalog to this Go file, then exit")
	probe := flag.Bool("probe", true, "execute read-only commands to capture output contracts")
	reuse := flag.String("reuse-probe", "", "reuse probe results from a previous api_reference.json instead of re-executing")
	flag.Parse()

	usage, err := runCapture(*kitPath, nil, 10*time.Second)
	if err != nil && usage == "" {
		fmt.Fprintf(os.Stderr, "cannot read kit usage: %v\n", err)
		os.Exit(1)
	}
	version, _ := runCapture(*kitPath, []string{"version"}, 10*time.Second)

	fixtures := Fixtures{
		Scenario: *scenario, Replay: *replay, Dat: *datFile,
		Xsdat: *xsdat, Folder: *folder,
	}
	catalog := loadCatalog(*kitPath)
	commands := parseUsage(usage, catalog)

	if *emitCatalog != "" {
		if err := writeCatalog(*emitCatalog, commands); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Printf("catalog written: %s (%d commands)\n", *emitCatalog, len(commands))
		return
	}

	probed := 0
	if *reuse != "" {
		probed = applyPriorProbe(commands, *reuse)
		fmt.Fprintf(os.Stderr, "reused %d probe results from %s\n", probed, *reuse)
		// Commands added since the prior run have no result to reuse. Probe just
		// those, so the reference stays complete as the CLI grows without paying
		// for a full re-run.
		if *probe {
			fresh := 0
			for i := range commands {
				if commands[i].ProbeStatus != "not_probed" {
					continue
				}
				if probeCommand(&commands[i], *kitPath, fixtures, *timeout) {
					probed++
				}
				fresh++
			}
			if fresh > 0 {
				fmt.Fprintf(os.Stderr, "probed %d new command(s) not present in the prior run\n", fresh)
			}
		}
	} else if *probe {
		for i := range commands {
			if probeCommand(&commands[i], *kitPath, fixtures, *timeout) {
				probed++
			}
		}
	}

	ref := Reference{
		Tool:      "aoe2kit",
		Version:   strings.TrimSpace(version),
		Generated: "cmd/apiref (usage parsed from the kit binary; output contracts probed against fixtures)",
		Fixtures:  fixtures,
		CommandsN: len(commands),
		ProbedN:   probed,
		Commands:  commands,
	}

	// Static pass: complete response shapes from source, joined to the probe's
	// observed keys so each command names the type it actually returns.
	schemas, schemaErr := ExtractSchemas("pkg", "cmd")
	if schemaErr != nil {
		fmt.Fprintf(os.Stderr, "schema extraction: %v\n", schemaErr)
	} else {
		for i := range commands {
			if hinted := CommandResponseTypeHint(commands[i].Name, schemas); hinted != "" {
				commands[i].ResponseType = hinted
				continue
			}
			commands[i].ResponseType = MatchResponseType(commands[i].OutputKeys, schemas)
		}
		ref.Commands = commands
		catalog := SchemaCatalog{
			Note:  "Complete nested shape of every exported report type, extracted statically from source. No execution involved; this is the authoritative response schema.",
			Count: len(schemas),
			Types: schemas,
		}
		if blob, err := json.MarshalIndent(catalog, "", "  "); err == nil {
			if err := os.WriteFile(*outSchemas, append(blob, '\n'), 0o644); err != nil {
				fmt.Fprintf(os.Stderr, "write schemas: %v\n", err)
			}
		}
	}
	if err := WriteGoDoc(*outGoDoc); err != nil {
		fmt.Fprintf(os.Stderr, "go doc: %v\n", err)
	}

	sanitize(&ref, fixtures)

	blob, err := json.MarshalIndent(ref, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := os.WriteFile(*outJSON, append(blob, '\n'), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := os.WriteFile(*outMD, []byte(renderMarkdown(ref)), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("commands=%d probed=%d md=%s json=%s\n", len(commands), probed, *outMD, *outJSON)
}

// sanitize replaces local fixture paths with placeholders. The reference is a
// shipped artifact; a generator that bakes the author's directory layout into
// it would be leaking exactly what kit portable-check exists to catch.
func sanitize(ref *Reference, fx Fixtures) {
	replacements := map[string]string{
		fx.Scenario: "<file.aoe2scenario>",
		fx.Replay:   "<file.aoe2record>",
		fx.Dat:      "<empires2_x2_p1.dat>",
		fx.Xsdat:    "<file.xsdat>",
	}
	for i := range ref.Commands {
		for j, arg := range ref.Commands[i].ProbeArgs {
			if placeholder, ok := replacements[arg]; ok && arg != "" {
				ref.Commands[i].ProbeArgs[j] = placeholder
			}
		}
		// Notes quote the command's own output, which can echo the input path.
		for path, placeholder := range replacements {
			if path != "" {
				ref.Commands[i].ProbeNote = strings.ReplaceAll(ref.Commands[i].ProbeNote, path, placeholder)
			}
		}
	}
	ref.Fixtures = Fixtures{
		Scenario: fixtureLabel(fx.Scenario), Replay: fixtureLabel(fx.Replay),
		Dat: fixtureLabel(fx.Dat), Xsdat: fixtureLabel(fx.Xsdat),
	}
}

// fixtureLabel keeps the file's identity without its location.
func fixtureLabel(path string) string {
	if path == "" {
		return ""
	}
	if idx := strings.LastIndexAny(path, "/\\"); idx >= 0 {
		return path[idx+1:]
	}
	return path
}

// applyPriorProbe copies probe results from an earlier run onto freshly parsed
// commands. Probing costs minutes because it executes the real tool against
// real files; regenerating documentation after an edit should not.
func applyPriorProbe(commands []Command, path string) int {
	blob, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "reuse: %v\n", err)
		return 0
	}
	var prior Reference
	if err := json.Unmarshal(blob, &prior); err != nil {
		fmt.Fprintf(os.Stderr, "reuse: %v\n", err)
		return 0
	}
	byName := map[string]Command{}
	for _, c := range prior.Commands {
		byName[c.Name] = c
	}
	reused := 0
	for i := range commands {
		old, ok := byName[commands[i].Name]
		if !ok || old.ProbeStatus == "" || old.ProbeStatus == "not_probed" {
			continue
		}
		commands[i].ProbeStatus = old.ProbeStatus
		commands[i].ProbeArgs = old.ProbeArgs
		commands[i].ExitCode = old.ExitCode
		commands[i].OutputKeys = old.OutputKeys
		commands[i].Verification = old.Verification
		commands[i].Method = old.Method
		commands[i].ProbeNote = old.ProbeNote
		if old.ProbeStatus == "ok" {
			reused++
		}
	}
	return reused
}

func runCapture(bin string, args []string, d time.Duration) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), d)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, args...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// runProbe keeps stdout clean of stderr, because commands that print warnings
// (e.g. loud truncation notices) would otherwise corrupt the JSON payload.
func runProbe(bin string, args []string, d time.Duration) (string, string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), d)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, args...)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	return string(out), stderr.String(), err
}

func parseUsage(usage string, catalog map[string]CatalogEntry) []Command {
	seen := map[string]int{}
	var commands []Command
	for _, line := range strings.Split(usage, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "kit ") {
			continue
		}
		tokens := strings.Fields(trimmed)[1:]
		var path []string
		var subs []string
		var positional []string
		var flags []Flag
		pathDone := false
		for i := 0; i < len(tokens); i++ {
			tok := tokens[i]
			switch {
			case !pathDone && identRe.MatchString(tok):
				path = append(path, tok)
			case !pathDone && groupRe.MatchString(tok):
				subs = strings.Split(groupRe.FindStringSubmatch(tok)[1], "|")
				pathDone = true
			case flagRe.MatchString(strings.Trim(tok, "[]")):
				name := strings.Trim(tok, "[]")
				hint := ""
				if i+1 < len(tokens) {
					next := strings.Trim(tokens[i+1], "[]")
					if !strings.HasPrefix(next, "--") && !strings.HasPrefix(next, "<") {
						hint = next
					}
				}
				// A flag not wrapped in [...] is required by the usage line.
				flags = append(flags, Flag{Name: name, Value: hint, Required: !strings.HasPrefix(tok, "[")})
				pathDone = true
			default:
				pathDone = true
				if strings.HasPrefix(tok, "<") || strings.HasPrefix(tok, "[") {
					positional = append(positional, strings.Trim(tok, "[]"))
				}
			}
		}
		if len(path) == 0 {
			continue
		}
		key := strings.Join(path, " ")
		cmd := Command{
			Path: path, Name: key, Usage: trimmed,
			Flags: flags, Positional: positional, Subcommands: subs,
			ProbeStatus: "not_probed",
		}
		if entry, ok := catalog[key]; ok {
			cmd.Trait = entry.Trait
			cmd.Input = entry.Input
			cmd.Mutates = entry.Trait == "writes"
			cmd.Network = entry.Trait == "network"
		} else {
			cmd.Trait = "undeclared"
		}
		if len(path) > 1 {
			cmd.Group = path[0]
		}
		if idx, ok := seen[key]; ok {
			// A later, more specific line wins; merge flags we already found.
			if len(cmd.Flags) >= len(commands[idx].Flags) {
				cmd.Subcommands = mergeStrings(commands[idx].Subcommands, cmd.Subcommands)
				commands[idx] = cmd
			} else {
				commands[idx].Subcommands = mergeStrings(commands[idx].Subcommands, cmd.Subcommands)
			}
			continue
		}
		seen[key] = len(commands)
		commands = append(commands, cmd)
	}
	sort.SliceStable(commands, func(i, j int) bool { return commands[i].Name < commands[j].Name })
	return commands
}

func mergeStrings(a, b []string) []string {
	if len(a) == 0 {
		return b
	}
	if len(b) == 0 {
		return a
	}
	set := map[string]bool{}
	var out []string
	for _, s := range append(append([]string{}, a...), b...) {
		if !set[s] {
			set[s] = true
			out = append(out, s)
		}
	}
	return out
}

// probeCommand executes a read-only command against fixtures and records the
// shape of what it returns. Returns true when output was captured.
func probeCommand(cmd *Command, kitPath string, fx Fixtures, timeout time.Duration) bool {
	switch {
	case cmd.Trait == "undeclared":
		// The CLI's catalog test should make this unreachable.
		cmd.ProbeStatus = "skipped_undeclared"
		return false
	case cmd.Input == "group":
		cmd.ProbeStatus = "group_entry"
		return false
	case cmd.Mutates:
		cmd.ProbeStatus = "skipped_mutating"
		return false
	case cmd.Network:
		cmd.ProbeStatus = "skipped_network"
		return false
	}
	// Required inputs the generator cannot synthesize (contracts, ledgers,
	// schemas, recipes) mean the command is real but not probeable here.
	for _, f := range cmd.Flags {
		if f.Required && f.Name != "--dat" {
			cmd.ProbeStatus = "requires_additional_input"
			cmd.ProbeNote = "required " + f.Name + " must be supplied by the caller"
			return false
		}
	}

	args := append([]string{}, cmd.Path...)
	arg, ok := fixtureFor(cmd, fx)
	if !ok {
		cmd.ProbeStatus = "skipped_no_fixture"
		return false
	}
	args = append(args, arg...)

	out, errOut, err := runProbe(kitPath, args, timeout)
	cmd.ProbeArgs = args
	if err != nil {
		if exitErr, isExit := err.(*exec.ExitError); isExit {
			cmd.ExitCode = exitErr.ExitCode()
		} else {
			cmd.ProbeStatus = "probe_error"
			cmd.ProbeNote = firstLine(out + errOut)
			return false
		}
	}
	var payload map[string]any
	if jsonErr := json.Unmarshal([]byte(out), &payload); jsonErr != nil {
		// Some commands render text by default; retry once in JSON mode.
		if strings.Contains(cmd.Usage, "--json") {
			retry := append(append([]string{}, args...), "--json")
			if out2, errOut2, err2 := runProbe(kitPath, retry, timeout); err2 == nil || isExitErr(err2) {
				if json.Unmarshal([]byte(out2), &payload) == nil {
					cmd.ProbeArgs = retry
					out, errOut = out2, errOut2
					goto decoded
				}
			}
		}
		cmd.ProbeStatus = "non_json_output"
		cmd.ProbeNote = firstLine(out + errOut)
		return false
	}
decoded:
	keys := make([]string, 0, len(payload))
	for k := range payload {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	cmd.OutputKeys = keys
	cmd.Verification = stringField(payload, "verification")
	cmd.Method = stringField(payload, "method")
	cmd.ProbeStatus = "ok"
	return true
}

// fixtureFor selects a fixture using the input kind the CLI declared, rather
// than guessing from usage text.
func fixtureFor(cmd *Command, fx Fixtures) ([]string, bool) {
	needsDat := strings.Contains(cmd.Usage, "--dat empires") || strings.Contains(cmd.Usage, "--dat <empires")
	switch cmd.Input {
	case "dat":
		if fx.Dat == "" {
			return nil, false
		}
		return withDatArgs(cmd, fx.Dat), true
	case "scenario":
		if fx.Scenario == "" {
			return nil, false
		}
		return []string{fx.Scenario}, true
	case "xsdat":
		if fx.Xsdat == "" {
			return nil, false
		}
		return []string{fx.Xsdat, "--types", "string,int,string,int"}, true
	case "replay":
		if fx.Replay == "" {
			return nil, false
		}
		args := []string{fx.Replay}
		if needsDat && fx.Dat != "" {
			args = append(args, "--dat", fx.Dat)
		}
		return args, true
	case "folder":
		return []string{fx.Folder}, true
	case "none":
		return []string{}, true
	}
	// "other" covers inputs the generator has no stand-in for (asset trees,
	// caller-authored JSON), which is a declaration, not a gap in classification.
	return nil, false
}

func withDatArgs(cmd *Command, dat string) []string {
	args := []string{dat}
	// Singular record lookups need an id; use ids present in every current DAT.
	switch cmd.Path[len(cmd.Path)-1] {
	case "unit":
		args = append(args, "0", "83")
	case "graphic", "effect", "tech", "unit-header", "terrain", "sound", "availability", "ability":
		args = append(args, "1")
	case "refs":
		args = append(args, "tech", "1")
	case "delete-plan":
		args = append(args, "effect", "1")
	}
	return args
}

func stringField(payload map[string]any, key string) string {
	if v, ok := payload[key]; ok {
		if s, isStr := v.(string); isStr {
			return s
		}
		if m, isMap := v.(map[string]any); isMap {
			if s, isStr := m["label"].(string); isStr {
				return s
			}
		}
	}
	return ""
}

func isExitErr(err error) bool {
	_, ok := err.(*exec.ExitError)
	return ok
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if idx := strings.IndexByte(s, '\n'); idx >= 0 {
		s = s[:idx]
	}
	if len(s) > 200 {
		s = s[:200] + "..."
	}
	return s
}

const preamble = `# AoE2Kit API Reference

Generated by ` + "`cmd/apiref`" + ` from the kit binary itself. Invocation lines are the
binary's own usage output; output contracts were captured by executing read-only
commands against real fixtures. Commands that were not executed say so.

Regenerate with:

` + "```sh" + `
go build -o kit ./cmd/kit
go run ./cmd/apiref --kit ./kit \
  --scenario <file.aoe2scenario> --replay <file.aoe2record> \
  --dat <empires2_x2_p1.dat> --xsdat <file.xsdat>
` + "```" + `

## Calling conventions

- **JSON is the default output.** Add ` + "`--text`" + ` for a human-readable rendering.
  Agents should parse JSON and ignore ` + "`--text`" + `.
- **Every report carries a verification label.** ` + "`structure_verified`" + ` means the
  parser/readback proved it. ` + "`engine_verified`" + ` means a live game or editor was the
  oracle. Labels beginning ` + "`behavior_inferred`" + ` or containing ` + "`candidate`" + ` are
  explicitly not proof. Never upgrade a claim past its label.
- **Exit codes.** ` + "`0`" + ` on success. Non-zero on parse failure, missing input, or a
  failed assertion. Assertion-bearing commands (` + "`xsdat decode --ledger`" + `,
  ` + "`verify-run`" + `, ` + "`ci check`" + `, ` + "`replay sidecar-sync --fail-on-mismatch`" + `,
  ` + "`replay coverage --fail-on-opaque`" + `) exit non-zero when the declared expectation
  is not met; that is the intended gate behavior, not a crash.
- **Writes go to a separate output file.** No command edits its input in place.
- **Resource guardrails.** Scenario inflation above 32 MiB is refused by default; raise
  with ` + "`AOE2KIT_MAX_SCENARIO_MB`" + ` or ` + "`AOE2KIT_ALLOW_HUGE_SCENARIO=1`" + `. A 1 GiB soft
  heap limit applies unless ` + "`GOMEMLIMIT`" + ` or ` + "`AOE2KIT_GOMEMLIMIT_MB`" + ` is set.
  Large listings truncate at 200 rows unless ` + "`--limit 0`" + `/` + "`--all`" + ` is passed.
- **No external dependencies or network access**, except ` + "`replay fetch`" + ` and
  ` + "`player stats`" + `, which call Microsoft's public endpoints.

## Probe status meanings

| status | meaning |
| --- | --- |
| ` + "`ok`" + ` | executed during generation; the listed output keys are real |
| ` + "`group_entry`" + ` | dispatch group, not a leaf command |
| ` + "`skipped_mutating`" + ` | writes files/registries; documented from usage only |
| ` + "`skipped_network`" + ` | performs network I/O; documented from usage only |
| ` + "`skipped_no_fixture`" + ` | no fixture of the required type was supplied |
| ` + "`requires_additional_input`" + ` | needs a caller-supplied contract/ledger/schema/recipe |
| ` + "`non_json_output`" + ` | emitted text rather than a JSON object |
| ` + "`probe_error`" + ` | execution failed; see the note |

Flags marked **required** in the tables below appear unbracketed in the tool's own
usage line and must be supplied.

`

func renderMarkdown(ref Reference) string {
	var b strings.Builder
	b.WriteString(preamble)
	fmt.Fprintf(&b, "Tool version `%s` — %d commands, %d probed.\n\n",
		ref.Version, ref.CommandsN, ref.ProbedN)

	groups := map[string][]Command{}
	var order []string
	for _, c := range ref.Commands {
		g := c.Group
		if g == "" {
			g = "(top level)"
		}
		if _, ok := groups[g]; !ok {
			order = append(order, g)
		}
		groups[g] = append(groups[g], c)
	}
	sort.Strings(order)

	b.WriteString("## Command index\n\n")
	for _, g := range order {
		names := make([]string, 0, len(groups[g]))
		for _, c := range groups[g] {
			names = append(names, "`"+c.Name+"`")
		}
		fmt.Fprintf(&b, "- **%s** — %s\n", g, strings.Join(names, ", "))
	}
	b.WriteString("\n")

	for _, g := range order {
		fmt.Fprintf(&b, "## %s\n\n", g)
		for _, c := range groups[g] {
			fmt.Fprintf(&b, "### `%s`\n\n", c.Name)
			fmt.Fprintf(&b, "```\n%s\n```\n\n", c.Usage)
			if len(c.Subcommands) > 0 {
				fmt.Fprintf(&b, "Subcommands: %s\n\n", "`"+strings.Join(c.Subcommands, "`, `")+"`")
			}
			if len(c.Flags) > 0 {
				b.WriteString("| flag | value |\n| --- | --- |\n")
				for _, f := range c.Flags {
					v := f.Value
					if v == "" {
						v = "_(boolean)_"
					} else {
						v = "`" + v + "`"
					}
					if f.Required {
						v += " **(required)**"
					}
					fmt.Fprintf(&b, "| `%s` | %s |\n", f.Name, v)
				}
				b.WriteString("\n")
			}
			var traits []string
			if c.Mutates {
				traits = append(traits, "**writes output files**")
			}
			if c.Network {
				traits = append(traits, "**network I/O**")
			}
			if len(traits) > 0 {
				fmt.Fprintf(&b, "%s\n\n", strings.Join(traits, " · "))
			}
			fmt.Fprintf(&b, "- probe: `%s`\n", c.ProbeStatus)
			if c.Method != "" {
				fmt.Fprintf(&b, "- method: `%s`\n", c.Method)
			}
			if c.Verification != "" {
				fmt.Fprintf(&b, "- verification: `%s`\n", c.Verification)
			}
			if len(c.OutputKeys) > 0 {
				fmt.Fprintf(&b, "- JSON top-level keys: `%s`\n", strings.Join(c.OutputKeys, "`, `"))
			}
			if c.ResponseType != "" {
				fmt.Fprintf(&b, "- response type: `%s` (full nested shape in `api_schemas.json`)\n", c.ResponseType)
			}
			if c.ProbeNote != "" {
				fmt.Fprintf(&b, "- note: %s\n", c.ProbeNote)
			}
			b.WriteString("\n")
		}
	}
	return b.String()
}
