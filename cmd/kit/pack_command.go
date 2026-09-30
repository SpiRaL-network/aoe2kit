package main

import (
	"fmt"
	"os"
	"strings"

	"aoe2kit/pkg/kit"
)

// packCommand builds an archive for a stated audience. Before profiles existed,
// producing a source-only handoff meant copying directories by hand and
// deciding which documents belonged — work the tool should do, and get right
// every time.
func packCommand(args []string) {
	output := "AoE2Kit.zip"
	root := "."
	opts := kit.PackOptions{Profile: kit.ProfileFull}
	positional := 0

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--profile":
			if i+1 >= len(args) {
				die("kit pack", fmt.Errorf("--profile requires a value (full|handoff|sandbox|public)"))
			}
			opts.Profile = kit.PackProfile(args[i+1])
			i++
		case strings.HasPrefix(arg, "--profile="):
			opts.Profile = kit.PackProfile(strings.TrimPrefix(arg, "--profile="))
		case arg == "--exclude":
			if i+1 >= len(args) {
				die("kit pack", fmt.Errorf("--exclude requires a glob"))
			}
			opts.Exclude = append(opts.Exclude, args[i+1])
			i++
		case strings.HasPrefix(arg, "--exclude="):
			opts.Exclude = append(opts.Exclude, strings.TrimPrefix(arg, "--exclude="))
		case arg == "--no-binary":
			no := false
			opts.KeepBinary = &no
		case arg == "--binary":
			yes := true
			opts.KeepBinary = &yes
		case arg == "--sha-sidecar":
			opts.SHASidecar = true
		case arg == "--update-manifest":
			opts.UpdateManifest = true
		case strings.HasPrefix(arg, "-"):
			die("kit pack", fmt.Errorf("unknown flag %q", arg))
		default:
			if positional == 0 {
				output = arg
			} else if positional == 1 {
				root = arg
			}
			positional++
		}
	}

	switch opts.Profile {
	case kit.ProfileFull, kit.ProfileHandoff, kit.ProfileSandbox, kit.ProfilePublic:
	default:
		die("kit pack", fmt.Errorf("unknown profile %q (want full, handoff, sandbox, or public)", opts.Profile))
	}

	report, err := kit.PackWithOptions(root, output, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "kit pack: %v\n", err)
		printJSON(report)
		os.Exit(1)
	}
	printJSON(report)
}
