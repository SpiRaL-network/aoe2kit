package xs

import (
	"fmt"
	"strconv"
	"strings"

	"aoe2kit/pkg/datcache"
)

// SourceFinding is a conservative source check, not an XS compiler verdict.
type SourceFinding struct {
	Code     string `json:"code"`
	Severity string `json:"severity"`
	File     string `json:"file"`
	Line     int    `json:"line"`
	GameLine int    `json:"game_line,omitempty"`
	Message  string `json:"message"`
	Fix      string `json:"fix"`
	Evidence string `json:"evidence"`
}

type lintToken struct {
	text    string
	line    int
	literal bool
}

// Keep this list aligned with the UGC guide XS lexer. These are tokenizer
// words, not merely builtins; using one as an identifier can make the engine
// reject an otherwise well-formed function signature.
var xsReservedWords = func() map[string]bool {
	out := map[string]bool{}
	for _, word := range strings.Fields("if else while for switch case default break continue include return extern const class then goto label dbg infiniteLoopLimit infiniteRecursionLimit breakpoint static export mutable minInterval maxInterval highFrequency priority active inactive group runImmediately int float bool string vector void rule") {
		out[word] = true
	}
	return out
}()

// SourceLintOptions controls checks that need more context than the historical
// source-only lint. The default LintSource behavior remains compatible for
// callers that lint foreign snippets; Kit generators can opt into strict
// create-result checking.
type SourceLintOptions struct {
	CheckCreateResults bool
	DatPath            string
}

// LintSource checks hazards observed in the September engine fixtures. Comments
// and literals are lexed separately so examples in prose cannot become code.
func LintSource(path, source string) []SourceFinding {
	return LintSourceWithOptions(path, source, SourceLintOptions{})
}

func LintSourceWithOptions(path, source string, opts SourceLintOptions) []SourceFinding {
	tokens := lexLintSource(source)
	var out []SourceFinding
	add := func(code, severity string, t lintToken, message, fix string) {
		out = append(out, SourceFinding{Code: code, Severity: severity, File: path, Line: t.line, Message: message, Fix: fix, Evidence: evidenceFor(code)})
	}
	lintDeclarations(tokens, add)
	lintVectorConstants(tokens, add)
	lintVectorComparisons(tokens, add)
	lintIntBoolComparisons(tokens, add)
	lintCatalogueHazards(tokens, add)
	lintForwardCalls(tokens, add)
	lintLogicalOperandCalls(tokens, add)
	lintLogicalOperandTypes(tokens, add)
	lintFloatArgumentPromotion(tokens, add)
	if opts.CheckCreateResults {
		lintCreateUnitResults(tokens, add)
	}
	if opts.DatPath != "" {
		lintGaiaOnlyCreates(tokens, opts.DatPath, add)
	}
	for i, t := range tokens {
		if !t.literal && i+1 < len(tokens) {
			switch t.text {
			case "int", "float", "bool", "string", "vector", "void":
				name := tokens[i+1]
				if !name.literal && builtinNames[name.text] {
					add("xs_builtin_name_collision", "error", name, "Declaration "+name.text+" collides with an XS builtin.", "Rename "+name.text+" to a distinct variable or function name.")
				}
				if !name.literal && xsReservedWords[name.text] {
					add("xs_reserved_identifier", "error", name, "Declaration "+name.text+" uses an XS reserved word.", "Rename the identifier; reserved words cannot be used as variables or parameters.")
				}
			}
		}
		if t.literal {
			for _, b := range []byte(t.text) {
				if b >= 128 {
					add("xs_non_ascii_string", "error", t, "Non-ASCII string literal can produce 'string not terminated' in DE.", "Use ASCII inside XS string literals; Unicode comments are allowed.")
					break
				}
			}
		}
		if (t.text == "if" || t.text == "while" || t.text == "for") && i+1 < len(tokens) && tokens[i+1].text == "(" {
			end := matchingParen(tokens, i+1)
			if end < 0 {
				continue
			}
			logical := []int{0}
			modulo, arithmetic, chain := false, false, false
			for j, c := range tokens[i+2 : end] {
				if c.literal {
					continue
				}
				switch c.text {
				case "%":
					modulo = true
				case "+", "-", "*", "/":
					if isBinaryArithmetic(tokens, i+2+j) {
						arithmetic = true
					}
				case "(":
					logical = append(logical, 0)
				case ")":
					if len(logical) > 1 {
						logical = logical[:len(logical)-1]
					}
				case "&&", "||":
					logical[len(logical)-1]++
					if logical[len(logical)-1] > 1 {
						chain = true
					}
				}
			}
			if modulo {
				add("xs_condition_modulo", "error", t, "Modulo inside a condition triggers the observed Error 0308 class.", "Compute the remainder in an int assignment before the condition.")
			}
			if arithmetic {
				add("xs_condition_arithmetic", "error", t, "Arithmetic inside a condition triggers the observed Error 0308 class.", "Compute the arithmetic result in a named variable before the condition.")
			}
			if chain {
				add("xs_condition_boolean_chain", "error", t, "More than two ungrouped Boolean terms trigger the observed Error 0308 class.", "Group Boolean terms pairwise with parentheses.")
			}
		}
	}
	// Analyze each function/rule independently. Ambiguous types are left unknown;
	// this intentionally does not pretend to infer arbitrary function results.
	start, depth := 0, 0
	for i, t := range tokens {
		if t.literal {
			continue
		}
		if t.text == "{" {
			depth++
		}
		if t.text == "}" {
			depth--
			if depth == 0 {
				lintLocalRegion(tokens[start:i+1], add)
				start = i + 1
			}
		}
	}
	if start < len(tokens) {
		lintLocalRegion(tokens[start:], add)
	}
	return out
}

func lintCreateUnitResults(ts []lintToken, add func(string, string, lintToken, string, string)) {
	for i, token := range ts {
		if token.literal || token.text != "xsCreateUnit" || !isCallName(ts, i) {
			continue
		}
		end := matchingParen(ts, i+1)
		if end < 0 {
			continue
		}
		assigned := i >= 2 && ts[i-1].text == "=" && !ts[i-2].literal
		if !assigned {
			add("xs_create_unit_unchecked", "error", token,
				"xsCreateUnit returns an object id, but the result is not assigned and checked.",
				"Assign the result to an int and handle id < 0 before using or discarding it.")
			continue
		}
		name := ts[i-2].text
		if !createResultChecked(ts, end+1, name) {
			add("xs_create_unit_unchecked", "error", token,
				"xsCreateUnit result "+name+" has no visible failure check.",
				"Check "+name+" < 0 immediately after creation and record a diagnostic on failure.")
		}
	}
}

func createResultChecked(ts []lintToken, start int, name string) bool {
	limit := start + 40
	if limit > len(ts) {
		limit = len(ts)
	}
	for i := start; i+4 < limit; i++ {
		if ts[i].text != "if" || ts[i+1].text != "(" {
			continue
		}
		end := matchingParen(ts, i+1)
		if end < 0 || end >= limit {
			continue
		}
		for j := i + 2; j+2 < end; j++ {
			if ts[j].literal || ts[j].text != name {
				continue
			}
			op := ts[j+1].text
			if op != "<" && op != "<=" && op != "==" && op != "!=" && op != ">" && op != ">=" {
				continue
			}
			if ts[j+2].text == "0" || ts[j+2].text == "-" {
				return true
			}
		}
	}
	return false
}

func matchingBrace(ts []lintToken, start int) int {
	depth := 0
	for i := start; i < len(ts); i++ {
		if ts[i].literal {
			continue
		}
		switch ts[i].text {
		case "{":
			depth++
		case "}":
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func lintGaiaOnlyCreates(ts []lintToken, datPath string, add func(string, string, lintToken, string, string)) {
	cache, err := datcache.Open(datPath)
	if err != nil {
		return
	}
	defer cache.Close()
	lintGaiaOnlyCreatesWithLookup(ts, func(unitID int) ([]datcache.UnitRow, error) {
		rows, _, err := cache.QueryUnits(datcache.UnitQuery{ID: &unitID, All: true})
		return rows, err
	}, add)
}

func lintGaiaOnlyCreatesWithLookup(ts []lintToken, lookup func(int) ([]datcache.UnitRow, error), add func(string, string, lintToken, string, string)) {
	constants := map[string]int{}
	for i := 0; i+4 < len(ts); i++ {
		if ts[i].text != "const" || ts[i+1].text != "int" || ts[i+3].text != "=" {
			continue
		}
		if value, err := strconv.Atoi(ts[i+4].text); err == nil {
			constants[ts[i+2].text] = value
		}
	}
	type wrapper struct {
		name       string
		unitParam  string
		ownerParam string
		unitIndex  int
		ownerIndex int
	}
	wrappers := make(map[string]wrapper)
	for i := 0; i+3 < len(ts); i++ {
		if ts[i].literal || !isXSReturnType(ts[i].text) || ts[i+2].text != "(" {
			continue
		}
		paramsEnd := matchingParen(ts, i+2)
		if paramsEnd < 0 || paramsEnd+1 >= len(ts) || ts[paramsEnd+1].text != "{" {
			continue
		}
		bodyEnd := matchingBrace(ts, paramsEnd+1)
		if bodyEnd < 0 {
			continue
		}
		params := splitCallArguments(ts[i+3 : paramsEnd])
		paramNames := make([]string, len(params))
		for j, param := range params {
			if len(param) > 1 && !param[1].literal {
				paramNames[j] = param[1].text
			}
		}
		for j := paramsEnd + 1; j < bodyEnd; j++ {
			if ts[j].text != "xsCreateUnit" || !isCallName(ts, j) {
				continue
			}
			callEnd := matchingParen(ts, j+1)
			if callEnd < 0 {
				continue
			}
			args := splitCallArguments(ts[j+2 : callEnd])
			if len(args) < 2 || len(args[0]) != 1 || len(args[1]) != 1 {
				continue
			}
			for paramIndex, paramName := range paramNames {
				if paramName == args[0][0].text && paramIndex < len(paramNames) {
					for ownerIndex, ownerName := range paramNames {
						if ownerName == args[1][0].text {
							wrappers[ts[i+1].text] = wrapper{name: ts[i+1].text, unitParam: paramName, ownerParam: ownerName, unitIndex: paramIndex, ownerIndex: ownerIndex}
						}
					}
				}
			}
		}
	}
	check := func(unitID, owner int, token lintToken) {
		if owner == 0 || unitID <= 0 {
			return
		}
		rows, queryErr := lookup(unitID)
		if queryErr != nil || len(rows) == 0 {
			return
		}
		name := ""
		nonGaia := false
		for _, row := range rows {
			if name == "" {
				name = row.Name
			}
			if row.CivIndex != 0 {
				nonGaia = true
			}
		}
		if !nonGaia {
			add("xs_gaia_only_unit_for_player", "error", token,
				fmt.Sprintf("Gaia-only unit %d %s is created for player %d.", unitID, name, owner),
				"Use owner 0 for Gaia eyecandy, or use a unit present in the target player's civ table.")
		}
	}
	for i, token := range ts {
		if token.literal || token.text != "xsCreateUnit" || !isCallName(ts, i) {
			continue
		}
		end := matchingParen(ts, i+1)
		if end < 0 {
			continue
		}
		args := splitCallArguments(ts[i+2 : end])
		if len(args) < 2 {
			continue
		}
		unitID, unitOK := literalOrConstInt(args[0], constants)
		owner, ownerOK := literalOrConstInt(args[1], constants)
		if !unitOK || !ownerOK {
			// A wrapper's parameters are checked at its call sites below, where
			// the concrete unit and owner constants are still available.
			wrapped := false
			for _, candidate := range wrappers {
				if candidate.unitParam == args[0][0].text && candidate.ownerParam == args[1][0].text {
					wrapped = true
					break
				}
			}
			if !wrapped {
				add("xs_gaia_only_unit_unknown", "warning", token,
					"Cannot verify xsCreateUnit Gaia availability because the unit type or player owner is not constant.",
					"Use a named integer constant or review the creation against the DAT cache.")
			}
			continue
		}
		check(unitID, owner, token)
	}
	for name, candidate := range wrappers {
		for i, token := range ts {
			if token.literal || token.text != name || !isCallName(ts, i) {
				continue
			}
			end := matchingParen(ts, i+1)
			if end < 0 {
				continue
			}
			args := splitCallArguments(ts[i+2 : end])
			if len(args) == 0 {
				continue
			}
			if candidate.unitIndex < len(args) && candidate.ownerIndex < len(args) {
				unitID, unitOK := literalOrConstInt(args[candidate.unitIndex], constants)
				owner, ownerOK := literalOrConstInt(args[candidate.ownerIndex], constants)
				if unitOK && ownerOK {
					check(unitID, owner, token)
				}
			}
		}
	}
}

func literalOrConstInt(ts []lintToken, constants map[string]int) (int, bool) {
	if len(ts) != 1 {
		return 0, false
	}
	if value, err := strconv.Atoi(ts[0].text); err == nil {
		return value, true
	}
	if ts[0].literal {
		return 0, false
	}
	value, ok := constants[ts[0].text]
	return value, ok
}

// lintFloatArgumentPromotion catches the DE parser's refusal to promote an
// int/bool expression at a float parameter boundary. The external xs-check
// calls this NoNumPromo; keeping the rule here makes generated and foreign XS
// agree before an engine run.
func lintFloatArgumentPromotion(ts []lintToken, add func(string, string, lintToken, string, string)) {
	functions := xsFunctionParameters(ts)
	variables := xsVariableTypes(ts)
	for i := range ts {
		if !isCallName(ts, i) {
			continue
		}
		params := functions[ts[i].text]
		if len(params) == 0 {
			continue
		}
		end := matchingParen(ts, i+1)
		if end < 0 {
			continue
		}
		args := splitCallArguments(ts[i+2 : end])
		for index, arg := range args {
			if index >= len(params) || params[index] != "float" {
				continue
			}
			kind, token := xsArgumentType(arg, variables)
			if (kind == "bool" || kind == "int") && token != nil {
				add("xs_float_argument_promotion", "error", *token,
					"A bool or int expression is passed to a float parameter; XS does not promote it reliably.",
					"Assign the expression to an explicit float temporary before passing it.")
			}
		}
	}
}

func xsFunctionParameters(ts []lintToken) map[string][]string {
	out := map[string][]string{}
	for i := 0; i+2 < len(ts); i++ {
		if ts[i].literal || !isXSReturnType(ts[i].text) || ts[i+2].text != "(" {
			continue
		}
		end := matchingParen(ts, i+2)
		if end < 0 || end+1 >= len(ts) || ts[end+1].text != "{" {
			continue
		}
		params := splitCallArguments(ts[i+3 : end])
		for _, param := range params {
			if len(param) > 0 && isXSReturnType(param[0].text) {
				out[ts[i+1].text] = append(out[ts[i+1].text], param[0].text)
			}
		}
	}
	return out
}

func splitCallArguments(ts []lintToken) [][]lintToken {
	if len(ts) == 0 {
		return nil
	}
	var out [][]lintToken
	start, depth := 0, 0
	for i, t := range ts {
		switch t.text {
		case "(":
			depth++
		case ")":
			depth--
		case ",":
			if depth == 0 {
				out = append(out, ts[start:i])
				start = i + 1
			}
		}
	}
	out = append(out, ts[start:])
	return out
}

func xsArgumentType(arg []lintToken, variables map[string]string) (string, *lintToken) {
	if len(arg) == 0 {
		return "", nil
	}
	for _, t := range arg {
		if t.literal {
			continue
		}
		if t.text == "true" || t.text == "false" {
			return "bool", &t
		}
	}
	if len(arg) == 1 {
		if kind := variables[arg[0].text]; kind != "" {
			return kind, &arg[0]
		}
	}
	if len(arg) > 0 && isCallName(arg, 0) {
		if arg[0].text == "xsHasResearchedLocalTechnology" || strings.HasPrefix(arg[0].text, "xsIs") {
			return "bool", &arg[0]
		}
	}
	return "", nil
}

// lintLogicalOperandCalls rejects side-effecting engine calls used as &&/||
// operands. XS runtime short-circuit behavior is not established, and a call
// such as xsCreateFile can leave a resource open even when the logical result
// is false. User-defined calls are left available for deliberate engine probes
// that measure this behavior.
func lintLogicalOperandCalls(ts []lintToken, add func(string, string, lintToken, string, string)) {
	for i, t := range ts {
		if t.literal || (t.text != "&&" && t.text != "||") {
			continue
		}
		for _, call := range []int{logicalOperandCall(ts, i, -1), logicalOperandCall(ts, i, 1)} {
			if call < 0 || !isPotentialSideEffectCall(ts[call].text) {
				continue
			}
			add("xs_logical_operand_call", "error", ts[call],
				"Function calls cannot be used as &&/|| operands because XS short-circuit behavior is not established and the call may leak runtime state.",
				"Evaluate the call in a separate statement, then combine its stored result in the logical condition.")
		}
	}
}

func isPotentialSideEffectCall(name string) bool {
	for _, prefix := range []string{
		"xsCreate", "xsClose", "xsWrite", "xsSet", "xsEffect", "xsResearch",
		"xsRemove", "xsDestroy", "xsTrain", "xsActivate", "xsDeactivate",
		"xsChat",
	} {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

func lintLogicalOperandTypes(ts []lintToken, add func(string, string, lintToken, string, string)) {
	functions := xsFunctionReturnTypes(ts)
	variables := xsVariableTypes(ts)
	for i, t := range ts {
		if t.literal || (t.text != "&&" && t.text != "||") {
			continue
		}
		for _, direction := range []int{-1, 1} {
			j := logicalOperandIndex(ts, i, direction)
			if j < 0 || !isCallName(ts, j) {
				continue
			}
			if direction < 0 && j > 0 && isComparisonOperator(ts[j-1].text) {
				continue
			}
			if direction > 0 && j+1 < len(ts) && isComparisonOperator(ts[j+1].text) {
				continue
			}
			kind := xsOperandType(ts, j, functions, variables)
			if kind != "" && kind != "bool" {
				add("xs_logical_operand_type", "error", ts[j],
					"Logical operands must be bool; XS does not implicitly convert this expression.",
					"Return bool from the helper or compare the numeric result explicitly before using &&/||.")
			}
		}
	}
	for i, t := range ts {
		if t.literal || (t.text != "if" && t.text != "while") || i+1 >= len(ts) || ts[i+1].text != "(" {
			continue
		}
		end := matchingParen(ts, i+1)
		if end < 0 || i+2 >= len(ts) {
			continue
		}
		simple := end == i+3
		call := isCallName(ts, i+2) && i+3 < len(ts) && matchingParen(ts, i+3) == end-1
		if !simple && !call {
			continue
		}
		kind := xsOperandType(ts, i+2, functions, variables)
		if kind != "" && kind != "bool" {
			add("xs_condition_non_bool", "error", ts[i+2],
				"if/while conditions must be bool; XS does not implicitly convert this expression.",
				"Compare the numeric result explicitly or return bool from the helper.")
		}
	}
}

func isComparisonOperator(text string) bool {
	switch text {
	case "=", "==", "!=", "<", ">", "<=", ">=":
		return true
	default:
		return false
	}
}

func logicalOperandIndex(ts []lintToken, logical, direction int) int {
	j := logical + direction
	if direction < 0 {
		for j >= 0 && ts[j].text == ")" {
			open := matchingOpenParen(ts, j)
			if open < 0 {
				return -1
			}
			j = open - 1
		}
	} else {
		for j < len(ts) && ts[j].text == "(" {
			j++
		}
	}
	if j < 0 || j >= len(ts) {
		return -1
	}
	return j
}

func xsFunctionReturnTypes(ts []lintToken) map[string]string {
	out := map[string]string{}
	for i := 0; i+2 < len(ts); i++ {
		if ts[i].literal || ts[i+1].literal || !isXSReturnType(ts[i].text) || ts[i+2].text != "(" {
			continue
		}
		end := matchingParen(ts, i+2)
		if end >= 0 && end+1 < len(ts) && ts[end+1].text == "{" {
			out[ts[i+1].text] = ts[i].text
		}
	}
	return out
}

func xsVariableTypes(ts []lintToken) map[string]string {
	out := map[string]string{}
	for i := 0; i+1 < len(ts); i++ {
		if ts[i].literal || ts[i+1].literal || !isXSReturnType(ts[i].text) {
			continue
		}
		if i+2 < len(ts) && ts[i+2].text == "(" {
			continue
		}
		out[ts[i+1].text] = ts[i].text
	}
	return out
}

func xsOperandType(ts []lintToken, index int, functions, variables map[string]string) string {
	if index < 0 || index >= len(ts) {
		return ""
	}
	text := ts[index].text
	if text == "true" || text == "false" {
		return "bool"
	}
	if _, err := strconv.ParseFloat(text, 64); err == nil {
		return "number"
	}
	if isCallName(ts, index) {
		return functions[text]
	}
	return variables[text]
}

func logicalOperandCall(ts []lintToken, logical, direction int) int {
	j := logical + direction
	if direction > 0 {
		for j < len(ts) && ts[j].text == "(" {
			j++
		}
		if j < len(ts) && isCallName(ts, j) {
			return j
		}
		return -1
	}
	for j >= 0 && ts[j].text == ")" {
		j = matchingOpenParen(ts, j) - 1
	}
	if j >= 0 && isCallName(ts, j) {
		return j
	}
	return -1
}

func matchingOpenParen(ts []lintToken, close int) int {
	depth := 0
	for i := close; i >= 0; i-- {
		switch ts[i].text {
		case ")":
			depth++
		case "(":
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func isBinaryArithmetic(tokens []lintToken, index int) bool {
	if index == 0 || index+1 >= len(tokens) {
		return false
	}
	previous := tokens[index-1].text
	switch previous {
	case "(", ",", "=", "!", "<", ">", "&&", "||", ";":
		return false
	default:
		return true
	}
}

func evidenceFor(code string) string {
	if strings.HasPrefix(code, "xs_") {
		switch code {
		case "xs_int_bool_comparison":
			return "references/ugc-guide/docs/general/xs/bugs/Language Syntax.md#3-function-parameters-and-return-statements-do-not-implicitly-type-cast"
		case "xs_call_argument_limit":
			return "references/ugc-guide/docs/general/xs/bugs/Language Syntax.md#4-limit-on-number-of-params-in-a-function-call"
		case "xs_vector_nonliteral":
			return "references/ugc-guide/docs/general/xs/bugs/Language Syntax.md#5-cannot-use-variables-or-expressions-in-vector-initialisation"
		case "xs_unary_negative":
			return "references/ugc-guide/docs/general/xs/bugs/Language Syntax.md#6-unary-negative-does-not-work"
		case "xs_int_first_arithmetic":
			return "references/ugc-guide/docs/general/xs/bugs/Language Syntax.md#1-outputs-of-operations-are-of-the-wrong-data-type"
		case "xs_unsupported_type":
			return "references/ugc-guide/docs/general/xs/bugs/Language Syntax.md#19-missing-data-types-which-are-documented"
		case "xs_single_quote_string":
			return "references/ugc-guide/docs/general/xs/bugs/Language Syntax.md#21-using-single-quotes-causes-the-could-not-emit-quads-error"
		case "xs_static_global":
			return "references/ugc-guide/docs/general/xs/bugs/Language Syntax.md#12-static-variables-in-global-scope"
		case "xs_global_string":
			return "references/ugc-guide/docs/general/xs/bugs/Language Syntax.md#13-strings-in-global-scope"
		case "xs_return_parentheses":
			return "references/ugc-guide/docs/general/xs/bugs/Language Syntax.md#16-return-statements-do-not-work-as-documented"
		case "xs_loop_self_assignment":
			return "references/ugc-guide/docs/general/xs/bugs/Language Syntax.md#9-assigning-loop-variable-to-itself-does-not-throw-an-error"
		case "xs_integer_literal_limit":
			return "references/ugc-guide/docs/general/xs/bugs/Language Syntax.md#10-integers-softly-limited-to-999999999"
		case "xs_limit_off_by_one":
			return "references/ugc-guide/docs/general/xs/bugs/Language Syntax.md#14-off-by-one-error-with-infinitelooplimit"
		case "xs_forward_function_call":
			return "docs/XS_ENGINE_VERIFIED_2026-09.md"
		case "xs_vector_order_comparison":
			return "https://github.com/Divy1211/xs-check (vector ordering comparison silent crash)"
		case "xs_create_unit_unchecked":
			return "docs/XS_ENGINE_VERIFIED_2026-09.md"
		case "xs_reserved_identifier":
			return "references/ugc-guide/docs/general/xs/beginner.md#variables"
		}
	}
	return "docs/XS_ENGINE_VERIFIED_2026-09.md"
}

// lintVectorComparisons records the external catalogue's silent-crash claim
// without upgrading it to an engine verdict. Only comparisons whose two
// operands are explicitly declared vectors are reported; unknown expressions
// stay available for deliberate probes and future type analysis.
func lintVectorComparisons(ts []lintToken, add func(string, string, lintToken, string, string)) {
	vectors := map[string]bool{}
	for i := 0; i+1 < len(ts); i++ {
		if !ts[i].literal && ts[i].text == "vector" && !ts[i+1].literal && lintIdentifier(ts[i+1].text) {
			vectors[ts[i+1].text] = true
		}
	}
	for i, t := range ts {
		if t.literal || (t.text != "<" && t.text != ">" && t.text != "<=" && t.text != ">=") || i == 0 || i+1 >= len(ts) {
			continue
		}
		left, right := ts[i-1], ts[i+1]
		if left.literal || right.literal || !vectors[left.text] || !vectors[right.text] {
			continue
		}
		add("xs_vector_order_comparison", "warning", t,
			"The external XS quirk catalogue reports a silent crash when vectors are compared with an ordering operator.",
			"Compare vector components explicitly, or keep vector comparisons to == and != until an engine fixture verifies the ordering form.")
	}
}

// lintForwardCalls catches the XS parser's no-forward-declaration rule. It is
// intentionally limited to names that are declared in this source: engine
// operators and external functions cannot be classified safely here.
func lintForwardCalls(ts []lintToken, add func(string, string, lintToken, string, string)) {
	definitions := map[string]int{}
	for i := 0; i+2 < len(ts); i++ {
		if ts[i].literal || !isXSReturnType(ts[i].text) || ts[i+1].literal || ts[i+2].text != "(" {
			continue
		}
		name := ts[i+1].text
		end := matchingParen(ts, i+2)
		if end < 0 || end+1 >= len(ts) || ts[end+1].text != "{" {
			continue
		}
		if _, ok := definitions[name]; !ok {
			definitions[name] = ts[i].line
		}
	}
	for i, t := range ts {
		if !isCallName(ts, i) {
			continue
		}
		if i > 0 && isXSReturnType(ts[i-1].text) {
			continue
		}
		defLine, ok := definitions[t.text]
		if !ok || t.line >= defLine {
			continue
		}
		add("xs_forward_function_call", "error", t,
			"User-defined function "+t.text+" is called before its definition; XS has no forward declarations.",
			"Move the definition above the call, or emit functions in dependency order.")
	}
}

func isXSReturnType(s string) bool {
	switch s {
	case "void", "int", "float", "bool", "string", "vector":
		return true
	default:
		return false
	}
}

// lintCatalogueHazards covers syntax failures documented in the local UGC
// guide. These checks are deliberately lexical and conservative; runtime-only
// hazards remain documentation rather than guessed static diagnostics.
func lintCatalogueHazards(ts []lintToken, add func(string, string, lintToken, string, string)) {
	depth := 0
	for i, t := range ts {
		if t.literal {
			continue
		}
		if t.text == "{" {
			depth++
			continue
		}
		if t.text == "}" {
			depth--
			continue
		}
		if (t.text == "long" || t.text == "double" || t.text == "char") && i+1 < len(ts) {
			add("xs_unsupported_type", "error", t, "This documented XS data type is not implemented by the engine.", "Use int, float, bool, string, vector, or void.")
		}
		if t.text == "static" && depth == 0 {
			add("xs_static_global", "error", t, "Static global variables silently disable XS execution.", "Remove static; globals already have internal linkage in XS.")
		}
		if t.text == "const" && i+1 < len(ts) && (ts[i+1].text == "int" || ts[i+1].text == "float" || ts[i+1].text == "bool" || ts[i+1].text == "string" || ts[i+1].text == "vector") && i > 0 && (ts[i-1].text == "(" || ts[i-1].text == ",") {
			add("xs_const_parameter", "error", t, "const is not a valid XS function parameter type.", "Remove const from the parameter declaration.")
		}
		if t.text == "return" && i+1 < len(ts) && ts[i+1].text != "(" && ts[i+1].text != ";" {
			add("xs_return_parentheses", "error", t, "Return expressions require parentheses in XS.", "Write return (expression);.")
		}
		if t.text == "'" {
			add("xs_single_quote_string", "error", t, "Single-quoted strings cause the XS emitter to fail.", "Use a double-quoted string literal.")
		}
		if t.text == "infiniteLoopLimit" || t.text == "infiniteRecursionLimit" {
			add("xs_limit_off_by_one", "warning", t, "XS execution limits have documented off-by-one or silent-crash behavior.", "Leave headroom and treat the limit as a conservative bound; do not rely on the exact final iteration.")
		}
		if t.text == "for" && i+3 < len(ts) && ts[i+2].text == "=" && ts[i+1].text == ts[i+3].text {
			add("xs_loop_self_assignment", "error", t, "A loop variable assigned to itself can execute unexpectedly.", "Initialize the loop variable from a literal or independent expression.")
		}
		if t.text == "-" && i+1 < len(ts) && i > 0 && isUnaryContext(ts[i-1].text) {
			if _, err := strconv.ParseFloat(ts[i+1].text, 64); err != nil {
				add("xs_unary_negative", "error", t, "Unary negative expressions are not reliable in XS.", "Use 0 - value or a computed assignment with an explicit zero operand.")
			}
		}
		if n, err := strconv.ParseInt(t.text, 10, 64); err == nil && n >= 1000000000 && n <= 2147483647 {
			add("xs_integer_literal_limit", "error", t, "Integer literals above 999999999 are rejected by the XS parser.", "Build the value from smaller arithmetic terms.")
		}
		if isCallName(ts, i) {
			if argc := callArgumentCount(ts, i+1); argc > 12 {
				add("xs_call_argument_limit", "error", t, "XS function calls accept at most 12 arguments.", "Pass a smaller argument set or use an array/struct-like set of globals.")
			}
		}
	}
}

func isUnaryContext(s string) bool {
	return s == "=" || s == "(" || s == "," || s == ":" || s == "return" || s == "{" || s == ";"
}

func isCallName(ts []lintToken, i int) bool {
	if i+1 >= len(ts) || ts[i+1].text != "(" || ts[i].literal {
		return false
	}
	if i > 0 {
		for _, typ := range []string{"int", "float", "bool", "string", "vector", "void"} {
			if ts[i-1].text == typ {
				return false
			}
		}
	}
	return true
}

func callArgumentCount(ts []lintToken, open int) int {
	end := matchingParen(ts, open)
	if end < 0 || end == open+1 {
		return 0
	}
	depth, count := 0, 1
	for _, t := range ts[open+1 : end] {
		switch t.text {
		case "(":
			depth++
		case ")":
			depth--
		case ",":
			if depth == 0 {
				count++
			}
		}
	}
	return count
}

// lintIntBoolComparisons catches the narrow, engine-proven type mismatch that
// caused the font clock to fail at load: an int-returning xsArrayGetInt result
// compared directly with a bool variable. It intentionally does not infer
// arbitrary expression types.
func lintIntBoolComparisons(ts []lintToken, add func(string, string, lintToken, string, string)) {
	boolNames := map[string]bool{}
	for i := 0; i+1 < len(ts); i++ {
		if !ts[i].literal && ts[i].text == "bool" && !ts[i+1].literal {
			boolNames[ts[i+1].text] = true
		}
	}
	for i, t := range ts {
		if t.literal || t.text != "xsArrayGetInt" || i+1 >= len(ts) || ts[i+1].text != "(" {
			continue
		}
		end := matchingParen(ts, i+1)
		if end < 0 || end+3 >= len(ts) || (ts[end+1].text != "!" && ts[end+1].text != "=") || ts[end+2].text != "=" {
			continue
		}
		other := ts[end+3]
		if !other.literal && boolNames[other.text] {
			add("xs_int_bool_comparison", "error", t, "An int-valued xsArrayGetInt result is compared with a bool variable; DE rejects this expression.", "Store the flag as int 0/1, or convert both sides to the same type before comparing.")
		}
	}
}

func lintVectorConstants(ts []lintToken, add func(string, string, lintToken, string, string)) {
	for i, t := range ts {
		if t.literal || t.text != "vector" || i+1 >= len(ts) || ts[i+1].text != "(" {
			continue
		}
		end := matchingParen(ts, i+1)
		if end < 0 {
			continue
		}
		start := i + 2
		valid, count := true, 0
		for j := start; j <= end; j++ {
			if j != end && ts[j].text != "," {
				continue
			}
			part := ts[start:j]
			if len(part) > 0 && (part[0].text == "-" || part[0].text == "+") {
				part = part[1:]
			}
			if len(part) != 1 || part[0].literal {
				valid = false
			} else if _, err := strconv.ParseFloat(part[0].text, 64); err != nil {
				valid = false
			}
			count++
			start = j + 1
		}
		if !valid || count != 3 {
			add("xs_vector_nonliteral", "error", t, "vector(...) requires compile-time coordinates; this check only accepts three numeric literals.", "Use xsVectorSet(x, y, z) for runtime values or expressions. Keep vector(...) for literal constants.")
		}
	}
}

// Local names are checked across nested blocks, but never across functions.
func lintDeclarations(ts []lintToken, add func(string, string, lintToken, string, string)) {
	start := 0
	for i := 0; i < len(ts); i++ {
		if ts[i].literal {
			continue
		}
		if ts[i].text == ";" {
			start = i + 1
		}
		if ts[i].text != "{" {
			continue
		}
		end, depth := i+1, 1
		for ; end < len(ts); end++ {
			if ts[end].literal {
				continue
			}
			if ts[end].text == "{" {
				depth++
			}
			if ts[end].text == "}" {
				depth--
				if depth == 0 {
					break
				}
			}
		}
		if end == len(ts) {
			return
		}
		seen := map[string]bool{}
		for j := start; j+1 < end; j++ {
			if ts[j].literal {
				continue
			}
			switch ts[j].text {
			case "int", "float", "bool", "string", "vector":
			default:
				continue
			}
			name := ts[j+1]
			if name.literal || !lintIdentifier(name.text) || (j+2 < len(ts) && ts[j+2].text == "(") {
				continue
			}
			if seen[name.text] {
				add("xs_local_redeclaration", "error", name, "Local name "+name.text+" is declared more than once in the same function/rule.", "Declare the name once above the loop or branches and assign inside them; use distinct names for distinct values.")
			}
			seen[name.text] = true
		}
		i, start = end, end+1
	}
}

func lintIdentifier(s string) bool {
	if s == "" {
		return false
	}
	for i, c := range s {
		if c == '_' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || i > 0 && c >= '0' && c <= '9' {
			continue
		}
		return false
	}
	return true
}

func lintLocalRegion(ts []lintToken, add func(string, string, lintToken, string, string)) {
	types := map[string]string{}
	for i := 0; i+1 < len(ts); i++ {
		if ts[i].literal || (ts[i].text != "int" && ts[i].text != "float") {
			continue
		}
		name := ts[i+1].text
		if i+2 < len(ts) && ts[i+2].text == "(" {
			continue
		}
		if old, ok := types[name]; ok && old != ts[i].text {
			types[name] = "unknown"
		} else if !ok {
			types[name] = ts[i].text
		}
	}
	// XS loop variables are implicitly declared by the for initializer. Treat
	// an integer initializer as int so mixed arithmetic inside the loop is not
	// silently missed.
	for i := 0; i+4 < len(ts); i++ {
		if ts[i].text == "for" && ts[i+1].text == "(" && ts[i+3].text == "=" {
			if _, err := strconv.ParseInt(ts[i+4].text, 0, 64); err == nil {
				types[ts[i+2].text] = "int"
			}
		}
	}
	kind := func(t lintToken) string {
		if t.literal {
			return ""
		}
		if k, ok := types[t.text]; ok {
			return k
		}
		if _, err := strconv.ParseInt(t.text, 0, 64); err == nil {
			return "int"
		}
		if _, err := strconv.ParseFloat(t.text, 64); err == nil {
			return "float"
		}
		return ""
	}
	for i, t := range ts {
		if i > 0 && i+1 < len(ts) && !t.literal && strings.Contains("+-*/", t.text) && len(t.text) == 1 && kind(ts[i-1]) == "int" && kind(ts[i+1]) == "float" {
			add("xs_int_first_arithmetic", "warning", t, "Int-first mixed arithmetic may trigger FirstOprArith.", "Convert the int operand to float; for commutative operations, put the float first. Preserve subtraction/division order.")
		}
		if t.literal || t.text != "xsGetPlayerUnitIds" || i+1 >= len(ts) || ts[i+1].text != "(" {
			continue
		}
		end := matchingParen(ts, i+1)
		if end < 0 {
			continue
		}
		if i < 2 || ts[i-1].text != "=" {
			// A direct nested consumer is legitimate; only flag discarded results.
			if end+1 < len(ts) && ts[end+1].text == ";" && (i == 0 || ts[i-1].text == ";" || ts[i-1].text == "{") {
				add("xs_unit_ids_discarded", "warning", t, "xsGetPlayerUnitIds returns an array ID; its result is discarded.", "Assign the returned ID and read it with xsArrayGetSize/xsArrayGetInt.")
			}
			continue
		}
		name := ts[i-2].text
		read := false
		for j := end + 1; j+2 < len(ts); j++ {
			if ts[j].text == "xsArrayGetSize" && ts[j+1].text == "(" && ts[j+2].text == name {
				read = true
				break
			}
		}
		if !read {
			add("xs_unit_ids_without_size", "warning", t, "Returned unit-ID array "+name+" has no visible xsArrayGetSize read in this function/rule.", "Read the returned array size before iterating; review manually if a helper consumes the array.")
		}
	}
}

func matchingParen(ts []lintToken, start int) int {
	depth := 0
	for i := start; i < len(ts); i++ {
		if ts[i].literal {
			continue
		}
		switch ts[i].text {
		case "(":
			depth++
		case ")":
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func lexLintSource(s string) []lintToken {
	var out []lintToken
	line := 1
	for i := 0; i < len(s); {
		c := s[i]
		if c == '\n' {
			line++
			i++
			continue
		}
		if c == ' ' || c == '\t' || c == '\r' {
			i++
			continue
		}
		if c == '/' && i+1 < len(s) && s[i+1] == '/' {
			for i < len(s) && s[i] != '\n' {
				i++
			}
			continue
		}
		if c == '/' && i+1 < len(s) && s[i+1] == '*' {
			i += 2
			for i < len(s) {
				if s[i] == '\n' {
					line++
				}
				if s[i] == '*' && i+1 < len(s) && s[i+1] == '/' {
					i += 2
					break
				}
				i++
			}
			continue
		}
		start, at := i, line
		if c == '"' {
			i++
			for i < len(s) {
				if s[i] == '\n' {
					line++
				}
				if s[i] == '\\' && i+1 < len(s) {
					if s[i+1] == '\n' {
						line++
					}
					i += 2
					continue
				}
				if s[i] == '"' {
					i++
					break
				}
				i++
			}
			out = append(out, lintToken{s[start:i], at, true})
			continue
		}
		if c >= '0' && c <= '9' || c == '.' && i+1 < len(s) && s[i+1] >= '0' && s[i+1] <= '9' {
			i++
			for i < len(s) {
				b := s[i]
				if b >= '0' && b <= '9' || b == '.' || b == 'e' || b == 'E' {
					i++
					continue
				}
				if (b == '+' || b == '-') && (s[i-1] == 'e' || s[i-1] == 'E') {
					i++
					continue
				}
				break
			}
		} else if c == '_' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' {
			i++
			for i < len(s) {
				b := s[i]
				if b == '_' || b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' || b >= '0' && b <= '9' {
					i++
				} else {
					break
				}
			}
		} else {
			i++
			if i < len(s) && ((c == '&' && s[i] == '&') || (c == '|' && s[i] == '|')) {
				i++
			}
		}
		out = append(out, lintToken{s[start:i], at, false})
	}
	return out
}
