package main

import (
	"fmt"
	"os"
	"sort"
	"strings"

	xsauthor "aoe2kit/pkg/xs"
)

func runXSDat(args []string) {
	if len(args) < 1 || args[0] == "--help" || args[0] == "-h" {
		fmt.Fprintln(os.Stderr, "usage: kit xsdat <decode> ...")
		if len(args) > 0 {
			os.Exit(0)
		}
		os.Exit(2)
	}
	switch args[0] {
	case "decode":
		if len(args) > 1 && (args[1] == "--help" || args[1] == "-h") {
			fmt.Fprintln(os.Stderr, "usage: kit xsdat decode <file.xsdat> [--types string,int,...] [--ledger expected.json] [--schema SCHEMA] [--text|--json]")
			os.Exit(0)
		}
		runXSDatDecode(args[1:])
	default:
		fmt.Fprintln(os.Stderr, "usage: kit xsdat <decode> ...")
		os.Exit(2)
	}
}

func runXSDatDecode(args []string) {
	if len(args) > 0 && (args[0] == "--help" || args[0] == "-h") {
		fmt.Fprintln(os.Stderr, "usage: kit xsdat decode <file.xsdat> [--types string,int,...] [--ledger expected.json] [--schema SCHEMA] [--text|--json]")
		os.Exit(0)
	}
	input := ""
	typeSpec := ""
	ledger := ""
	schema := ""
	text := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--types":
			i++
			if i >= len(args) {
				die("kit xsdat decode", fmt.Errorf("--types needs a comma-separated type list"))
			}
			typeSpec = args[i]
		case "--ledger":
			i++
			if i >= len(args) {
				die("kit xsdat decode", fmt.Errorf("--ledger needs an expected ledger JSON path"))
			}
			ledger = args[i]
		case "--schema":
			i++
			if i >= len(args) {
				die("kit xsdat decode", fmt.Errorf("--schema needs a schema name"))
			}
			schema = args[i]
		case "--text":
			text = true
		case "--json":
			text = false
		default:
			if input != "" {
				die("kit xsdat decode", fmt.Errorf("unexpected argument %q", args[i]))
			}
			input = args[i]
		}
	}
	if input == "" {
		fmt.Fprintln(os.Stderr, "usage: kit xsdat decode <file.xsdat> [--types string,int,...] [--ledger expected.json] [--schema rtv12|rtv13|rtv14|rtv141|rtv142|rtv15|rtv16|rtv17|a2ksem2|customstore|engine-harness|long-v3] [--text]")
		os.Exit(2)
	}
	if schema == "" && typeSpec == "" {
		typeSpec = siblingTypesDocument(input)
	}
	if schema != "" {
		report, err := xsauthor.DecodeDataFileSchema(input, xsauthor.DataSchemaDecodeOptions{Schema: schema})
		if err != nil {
			die("kit xsdat decode", err)
		}
		if text {
			printXSDatSchemaDecodeText(report)
		} else {
			printJSON(report)
		}
		if !report.OK {
			os.Exit(1)
		}
		return
	}
	if typeSpec != "" {
		document := typeSpec
		if data, readErr := os.ReadFile(typeSpec); readErr == nil {
			document = string(data)
		}
		if strings.Contains(document, "header =") {
			documentSchema, recognized, documentErr := xsauthor.SchemaFromTypesDocument(document)
			if documentErr != nil {
				die("kit xsdat decode", documentErr)
			}
			if recognized {
				report, decodeErr := xsauthor.DecodeDataFileSchema(input, xsauthor.DataSchemaDecodeOptions{Schema: documentSchema})
				if decodeErr != nil {
					die("kit xsdat decode", decodeErr)
				}
				if text {
					printXSDatSchemaDecodeText(report)
				} else {
					printJSON(report)
				}
				if !report.OK {
					os.Exit(1)
				}
				return
			}
		}
	}
	types, err := xsauthor.ParseDataTypes(typeSpec)
	if err != nil {
		die("kit xsdat decode", err)
	}
	report, err := xsauthor.DecodeDataFile(input, xsauthor.DataDecodeOptions{Types: types, Ledger: ledger})
	if err != nil {
		die("kit xsdat decode", err)
	}
	if text {
		printXSDatDecodeText(report)
	} else {
		printJSON(report)
	}
	if !report.OK {
		os.Exit(1)
	}
}

// siblingTypesDocument lets generated sidecars carry their schema without
// requiring every caller to repeat a fixture-specific --schema flag. Prefer a
// matching basename; otherwise accept a single .types document in the output
// directory and refuse to guess when several candidates exist.
func siblingTypesDocument(input string) string {
	return xsauthor.DiscoverSiblingTypesFile(input)
}

func printXSDatDecodeText(report xsauthor.DataDecodeReport) {
	fmt.Printf("%s: ok=%t bytes=%d mode=%s\n", report.Path, report.OK, report.Decode.SizeBytes, report.Decode.Mode)
	fmt.Printf("verification: %s (%s)\n", report.Verification.Label, report.Verification.Note)
	if report.Ledger != "" {
		fmt.Printf("ledger: %s\n", report.Ledger)
		fmt.Printf("assertions: total=%d passed=%d failed=%d unknown=%d\n", report.Summary.Total, report.Summary.Passed, report.Summary.Failed, report.Summary.Unknown)
	}
	for _, value := range report.Decode.Values {
		fmt.Printf("[%02d] @%04d %-12s size=%d", value.Index, value.Offset, value.Type, value.Size)
		switch value.Type {
		case "string":
			fmt.Printf(" %q", value.String)
		case "int", "int_or_float":
			if value.Int != nil {
				fmt.Printf(" int=%d", *value.Int)
			}
			if value.UInt != nil {
				fmt.Printf(" uint=%d", *value.UInt)
			}
		case "uint":
			if value.UInt != nil {
				fmt.Printf(" uint=%d", *value.UInt)
			}
		case "float":
			if value.Float != nil {
				fmt.Printf(" float=%g", *value.Float)
			}
		case "vector":
			fmt.Printf(" vector=%v", value.Vector)
		case "raw":
			fmt.Printf(" raw_hex=%s", value.RawHex)
		}
		if value.Note != "" {
			fmt.Printf(" note=%q", value.Note)
		}
		fmt.Println()
	}
	if len(report.Assertions) > 0 {
		fmt.Println("assertion_rows:")
		for _, assertion := range report.Assertions {
			message := ""
			if assertion.Message != "" {
				message = " - " + assertion.Message
			}
			fmt.Printf("- [%s] #%02d expected=%q actual=%q%s\n", assertion.Status, assertion.Index, assertion.Expected, assertion.Actual, message)
		}
	}
	if len(report.Errors) > 0 {
		fmt.Println("errors:")
		for _, err := range report.Errors {
			fmt.Printf("- %s\n", err)
		}
	}
	if report.Decode.RemainingBytes > 0 {
		fmt.Printf("remaining @%04d bytes=%d hex=%s\n", report.Decode.RemainingOffset, report.Decode.RemainingBytes, report.Decode.RemainingHex)
	}
}

func printXSDatSchemaDecodeText(report xsauthor.DataSchemaDecodeReport) {
	fmt.Printf("%s: ok=%t schema=%s bytes=%d rows=%d footer=%t\n",
		report.Path, report.OK, report.Schema, report.Summary.SizeBytes, report.Summary.Rows, report.Summary.HasFooter)
	fmt.Printf("coverage: consumed=%d/%d bytes cap=%d within_cap=%t rows=%d complete_rows=%d footer=%t\n",
		report.Summary.BytesConsumed, report.Summary.SizeBytes, report.Summary.ByteCap, report.Summary.WithinByteCap, report.Summary.Rows, report.Summary.CompleteRows, report.Summary.HasFooter)
	tags := make([]string, 0, len(report.Summary.TagCounts))
	for tag := range report.Summary.TagCounts {
		tags = append(tags, tag)
	}
	sort.Strings(tags)
	fmt.Print("tags:")
	for _, tag := range tags {
		fmt.Printf(" %s=%d", tag, report.Summary.TagCounts[tag])
	}
	fmt.Println()
	fmt.Printf("verification: %s (%s)\n", report.Verification.Label, report.Verification.Note)
	if len(report.Header) > 0 {
		fmt.Printf("header: magic=%v version=%v fixture=%v\n", report.Header["magic"], report.Header["version"], report.Header["fixture"])
		fmt.Printf("header_fields=%s\n", formatSchemaFields(report.Header))
	}
	if report.Schema == "a2k-snake-v0" {
		for _, warning := range report.Warnings {
			fmt.Printf("warning: %s\n", warning)
		}
		for _, err := range report.Errors {
			fmt.Printf("error: %s\n", err)
		}
		return
	}
	for _, row := range report.Rows {
		f := row.Fields
		if report.Schema != "a2k-z-bouncing-ball" {
			fmt.Printf("  fields=%s\n", formatSchemaFields(f))
		}
		if report.Schema == "rtv13-directed-attribution" {
			fmt.Printf("- row=%02d phase=%d time=%s case=P%v kind=%v p1_kills=%v p1_razes=%v p1_vs_p2=%v/%v p1_vs_p3=%v/%v p1_vs_p4=%v/%v p1_vs_p5=%v/%v p1_vs_p6=%v/%v p1_vs_p7=%v/%v p1_vs_p8=%v/%v counts_p2=%v/%v counts_p8=%v/%v\n",
				row.Index,
				row.PhaseID,
				formatSeconds(row.TimeS),
				f["case_player"],
				f["case_kind"],
				f["p1_kills_attr20"],
				f["p1_razings_attr43"],
				f["p1_player2_kills_attr302"],
				f["p1_player2_razings_attr352"],
				f["p1_player3_kills_attr303"],
				f["p1_player3_razings_attr353"],
				f["p1_player4_kills_attr304"],
				f["p1_player4_razings_attr354"],
				f["p1_player5_kills_attr305"],
				f["p1_player5_razings_attr355"],
				f["p1_player6_kills_attr306"],
				f["p1_player6_razings_attr356"],
				f["p1_player7_kills_attr307"],
				f["p1_player7_razings_attr357"],
				f["p1_player8_kills_attr308"],
				f["p1_player8_razings_attr358"],
				f["p2_militia_74_count"],
				f["p2_barracks_12_count"],
				f["p8_militia_74_count"],
				f["p8_barracks_12_count"],
			)
			continue
		}
		if report.Schema == "rtv14-castle-kill-calibration" {
			fmt.Printf("- row=%02d phase=%d time=%s case_player=%v case_unit=%v spawn=%v expected_p1_kills=%v p1_kills=%v p1_vs_p2=%v p1_vs_p3=%v p1_vs_p4=%v p1_vs_p5=%v p1_vs_p6=%v p1_vs_p7=%v p1_vs_p8=%v counts_p2=%v/%v/%v/%v/%v/%v counts_p8=%v/%v/%v/%v/%v/%v\n",
				row.Index,
				row.PhaseID,
				formatSeconds(row.TimeS),
				f["case_player"],
				f["case_unit"],
				f["spawn_count"],
				f["expected_p1_kills"],
				f["p1_kills_attr20"],
				f["p1_player2_kills_attr302"],
				f["p1_player3_kills_attr303"],
				f["p1_player4_kills_attr304"],
				f["p1_player5_kills_attr305"],
				f["p1_player6_kills_attr306"],
				f["p1_player7_kills_attr307"],
				f["p1_player8_kills_attr308"],
				f["p2_militia_74_count"],
				f["p2_archer_4_count"],
				f["p2_spearman_93_count"],
				f["p2_villager_83_count"],
				f["p2_scout_448_count"],
				f["p2_knight_38_count"],
				f["p8_militia_74_count"],
				f["p8_archer_4_count"],
				f["p8_spearman_93_count"],
				f["p8_villager_83_count"],
				f["p8_scout_448_count"],
				f["p8_knight_38_count"],
			)
			continue
		}
		if report.Schema == "rtv141-castle-kill-calibration" {
			fmt.Printf("- row=%02d phase=%d time=%s case_player=%v case_unit=%v spawn=%v expected_p1_kills=%v p1_kills=%v p1_vs_p2=%v p1_vs_p3=%v p2_killed_by_p1=%v p3_killed_by_p1=%v counts_p2=%v/%v/%v/%v/%v/%v counts_p3=%v/%v/%v/%v/%v/%v\n",
				row.Index,
				row.PhaseID,
				formatSeconds(row.TimeS),
				f["case_player"],
				f["case_unit"],
				f["spawn_count"],
				f["expected_p1_kills"],
				f["p1_kills_attr20"],
				f["p1_player2_kills_attr302"],
				f["p1_player3_kills_attr303"],
				f["p2_kills_by_player1_attr326"],
				f["p3_kills_by_player1_attr326"],
				f["p2_militia_74_count"],
				f["p2_archer_4_count"],
				f["p2_spearman_93_count"],
				f["p2_villager_83_count"],
				f["p2_scout_448_count"],
				f["p2_knight_38_count"],
				f["p3_militia_74_count"],
				f["p3_archer_4_count"],
				f["p3_spearman_93_count"],
				f["p3_villager_83_count"],
				f["p3_scout_448_count"],
				f["p3_knight_38_count"],
			)
			continue
		}
		if report.Schema == "rtv15-castle-kill-ground-truth" {
			fmt.Printf("- row=%02d phase=%d time=%s case_player=%v case_unit=%v spawn=%v expected_p1_kills=%v p1_kills=%v p1_vs_p2=%v p1_vs_p3=%v p1_vs_p4=%v p2_killed_by_p1=%v p3_killed_by_p1=%v p4_killed_by_p1=%v counts_p2=%v/%v/%v/%v/%v/%v/%v counts_p3=%v/%v/%v/%v/%v/%v/%v counts_p4=%v/%v/%v/%v/%v/%v/%v\n",
				row.Index,
				row.PhaseID,
				formatSeconds(row.TimeS),
				f["case_player"],
				f["case_unit"],
				f["spawn_count"],
				f["expected_p1_kills"],
				f["p1_kills_attr20"],
				f["p1_player2_kills_attr302"],
				f["p1_player3_kills_attr303"],
				f["p1_player4_kills_attr304"],
				f["p2_kills_by_player1_attr326"],
				f["p3_kills_by_player1_attr326"],
				f["p4_kills_by_player1_attr326"],
				f["p2_barracks_12_count"],
				f["p2_militia_74_count"],
				f["p2_archer_4_count"],
				f["p2_spearman_93_count"],
				f["p2_villager_83_count"],
				f["p2_scout_448_count"],
				f["p2_knight_38_count"],
				f["p3_barracks_12_count"],
				f["p3_militia_74_count"],
				f["p3_archer_4_count"],
				f["p3_spearman_93_count"],
				f["p3_villager_83_count"],
				f["p3_scout_448_count"],
				f["p3_knight_38_count"],
				f["p4_barracks_12_count"],
				f["p4_militia_74_count"],
				f["p4_archer_4_count"],
				f["p4_spearman_93_count"],
				f["p4_villager_83_count"],
				f["p4_scout_448_count"],
				f["p4_knight_38_count"],
			)
			continue
		}
		if report.Schema == "rtv16-semantic-promotion" {
			fmt.Printf("- row=%02d phase=%d time=%s kind=%v case=P%v expected=%vK/%vR actual_p1=%vK/%vR p1_vs_p2=%vK/%vR p1_vs_p3=%vK/%vR p1_vs_p4=%vK/%vR p2_by_p1=%vK/%vR p3_by_p1=%vK/%vR p4_by_p1=%vK/%vR p1_score_parts=%v/%v/%v/%v p1_explore=%v reveal=%v/%v/%v counts_p2=%v/%v/%v/%v counts_p3=%v/%v/%v/%v counts_p4=%v/%v/%v/%v gaia_militia=%v\n",
				row.Index,
				row.PhaseID,
				formatSeconds(row.TimeS),
				f["case_kind"],
				f["case_player"],
				f["expected_p1_kills"],
				f["expected_p1_razes"],
				f["p1_kills_attr20"],
				f["p1_razings_attr43"],
				f["p1_player2_kills_attr302"],
				f["p1_player2_razings_attr352"],
				f["p1_player3_kills_attr303"],
				f["p1_player3_razings_attr353"],
				f["p1_player4_kills_attr304"],
				f["p1_player4_razings_attr354"],
				f["p2_kills_by_player1_attr326"],
				f["p2_razings_by_player1_attr376"],
				f["p3_kills_by_player1_attr326"],
				f["p3_razings_by_player1_attr376"],
				f["p4_kills_by_player1_attr326"],
				f["p4_razings_by_player1_attr376"],
				f["p1_food_score_attr185"],
				f["p1_wood_score_attr186"],
				f["p1_stone_score_attr187"],
				f["p1_gold_score_attr188"],
				f["p1_exploration_attr22"],
				f["p1_map_reveal_attr203"],
				f["p1_unit_reveal_attr204"],
				f["p1_temporary_map_reveal_attr209"],
				f["p2_barracks_12_count"],
				f["p2_militia_74_count"],
				f["p2_archer_4_count"],
				f["p2_spearman_93_count"],
				f["p3_barracks_12_count"],
				f["p3_militia_74_count"],
				f["p3_archer_4_count"],
				f["p3_spearman_93_count"],
				f["p4_barracks_12_count"],
				f["p4_militia_74_count"],
				f["p4_archer_4_count"],
				f["p4_spearman_93_count"],
				f["gaia_militia_74_count"],
			)
			continue
		}
		if report.Schema == "rtv17-packed-test" {
			fmt.Printf("- row=%02d phase=%d time=%s kind=%v case=P%v expected=%vK/%vD/%vR actual_p1=%vK/%vD/%vR actual_p2=%vK/%vR p1_vs_p2=%vK/%vR p2_vs_p1=%vK/%vR p1_by_p2=%vK/%vR p2_by_p1=%vK/%vR p1_counts=%v/%v/%v/%v/%v/%v p2_castle=%v p2_counts=%v/%v/%v/%v gaia_militia=%v\n",
				row.Index,
				row.PhaseID,
				formatSeconds(row.TimeS),
				f["case_kind"],
				f["case_player"],
				f["expected_p1_kills"],
				f["expected_p1_deaths"],
				f["expected_p1_razes"],
				f["p1_kills_attr20"],
				f["p1_killed_by_others_attr154"],
				f["p1_razings_attr43"],
				f["p2_kills_attr20"],
				f["p2_razings_attr43"],
				f["p1_player2_kills_attr302"],
				f["p1_player2_razings_attr352"],
				f["p2_player1_kills_attr301"],
				f["p2_player1_razings_attr351"],
				f["p1_kills_by_player2_attr327"],
				f["p1_razings_by_player2_attr377"],
				f["p2_kills_by_player1_attr326"],
				f["p2_razings_by_player1_attr376"],
				f["p1_militia_74_count"],
				f["p1_archer_4_count"],
				f["p1_spearman_93_count"],
				f["p1_villager_83_count"],
				f["p1_scout_448_count"],
				f["p1_knight_38_count"],
				f["p2_castle_82_count"],
				f["p2_barracks_12_count"],
				f["p2_militia_74_count"],
				f["p2_archer_4_count"],
				f["p2_spearman_93_count"],
				f["gaia_militia_74_count"],
			)
			continue
		}
		if report.Schema == "a2ksem2-dat-command-semantics" {
			fmt.Printf("- row=%02d phase=%d lane=%v time=%s res(F/W/S/G)=%v/%v/%v/%v pop=%v research=%v counts militia=%v maa=%v villager=%v scout=%v tc=%v barracks=%v stable=%v\n",
				row.Index,
				row.PhaseID,
				f["lane_id"],
				formatSeconds(row.TimeS),
				f["p1_food_attr0"],
				f["p1_wood_attr1"],
				f["p1_stone_attr2"],
				f["p1_gold_attr3"],
				f["p1_population_attr11"],
				f["p1_research_count_attr21"],
				f["p1_militia_74_count"],
				f["p1_man_at_arms_75_count"],
				f["p1_villager_83_count"],
				f["p1_scout_448_count"],
				f["p1_town_center_109_count"],
				f["p1_barracks_12_count"],
				f["p1_stable_101_count"],
			)
			continue
		}
		if report.Schema == "a2k-trig-long" || report.Schema == "a2k-long-v3" {
			fmt.Printf("- row=%02d phase=%d time=%s fields=%v\n", row.Index, row.PhaseID, formatSeconds(row.TimeS), row.Fields)
			continue
		}
		if report.Schema == "a2k-barracks-wall-v2" {
			fmt.Printf("- row=%02d unit=%v owner=%v pos=(%v,%v) attr3=%v attr4=%v attr200=%v attr201=%v\n",
				row.Index, f["unit_id"], f["owner"], f["x"], f["y"], f["attr_3"], f["attr_4"], f["attr_200"], f["attr_201"])
			continue
		}
		if report.Schema == "a2k-z-bouncing-ball" {
			fmt.Printf("- row=%02d tick=%v time=%s fields=%s\n",
				row.Index, f["tick"], formatSeconds(row.TimeS), formatSchemaFields(f))
			continue
		}
		fmt.Printf("- row=%02d phase=%d time=%s p1_kills=%v p1_razes=%v p1_kill_value=%v p1_raze_value=%v p2_killed_by=%v p2_razed_by=%v p3_killed_by=%v p2_militia=%v p2_barracks=%v p3_militia=%v gaia_militia=%v\n",
			row.Index,
			row.PhaseID,
			formatSeconds(row.TimeS),
			f["p1_kills_attr20"],
			f["p1_razings_attr43"],
			f["p1_kill_value_attr170"],
			f["p1_raze_value_attr172"],
			f["p2_killed_by_others_attr154"],
			f["p2_razed_by_others_attr155"],
			f["p3_killed_by_others_attr154"],
			f["p2_militia_74_count"],
			f["p2_barracks_12_count"],
			f["p3_militia_74_count"],
			f["gaia_militia_74_count"],
		)
	}
	if len(report.Footer) > 0 {
		fmt.Printf("footer_fields=%s\n", formatSchemaFields(report.Footer))
		if report.Schema == "a2k-barracks-wall-v2" {
			fmt.Printf("footer: tag=%v integrity=%v\n", report.Footer["tag"], report.Footer["integrity"])
		} else {
			fmt.Printf("footer: tag=%v rows_written=%v append_tag=%v append_probe=%v\n",
				report.Footer["tag"], report.Footer["rows_written"], report.Footer["append_tag"], report.Footer["append_probe"])
		}
	}
	if len(report.Warnings) > 0 {
		fmt.Println("warnings:")
		for _, warning := range report.Warnings {
			fmt.Printf("- %s\n", warning)
		}
	}
	if len(report.Errors) > 0 {
		fmt.Println("errors:")
		for _, err := range report.Errors {
			fmt.Printf("- %s\n", err)
		}
	}
}

func formatSchemaFields(fields map[string]any) string {
	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%v", key, fields[key]))
	}
	return "{" + strings.Join(parts, ", ") + "}"
}

func formatSeconds(seconds int) string {
	if seconds < 0 {
		return ""
	}
	return fmt.Sprintf("%02d:%02d", seconds/60, seconds%60)
}
