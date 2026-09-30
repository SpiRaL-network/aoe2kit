package xs

import (
	"strings"
	"testing"

	"aoe2kit/pkg/datcache"
)

func TestBuiltinSnapshot(t *testing.T) {
	if len(builtinNames) != 274 {
		t.Fatalf("builtin count: %d (guide commit %s)", len(builtinNames), builtinGuideCommit)
	}
	for name := range builtinNames {
		got := LintSource("test.xs", "int "+name+"=0;")
		if len(got) != 1 || got[0].Code != "xs_builtin_name_collision" || got[0].Severity != "error" {
			t.Fatalf("%s: %+v", name, got)
		}
	}
}

func TestDLCShortBuiltinNames(t *testing.T) {
	for _, name := range []string{"bin", "chr", "fstr", "hex", "ord", "strLen", "bitLsh", "bitRsh"} {
		if !builtinNames[name] {
			t.Fatalf("missing DLC builtin %q from guide commit %s", name, builtinGuideCommit)
		}
		findings := LintSource("test.xs", "int "+name+"=0;")
		if len(findings) != 1 || findings[0].Code != "xs_builtin_name_collision" {
			t.Fatalf("%s: %+v", name, findings)
		}
	}
}

func TestEngineHazardLint(t *testing.T) {
	cases := []struct{ name, source, code, severity string }{
		{"runtime vector", "void f() { vector v=vector(parkX,parkY,-1.0); }", "xs_vector_nonliteral", "error"},
		{"expression vector", "void f() { vector v=vector(2.0+x,0,-1); }", "xs_vector_nonliteral", "error"},
		{"nested vector", "void f() { vector v=vector(xsGetTime(),0,-1); }", "xs_vector_nonliteral", "error"},
		{"builtin loop", "void f() { for (i=0; <3) { int ln=0; } }", "xs_builtin_name_collision", "error"},
		{"builtin function", "float sin(float x=0) { return(x); }", "xs_builtin_name_collision", "error"},
		{"builtin parameter", "void f(int xsGetTime=0) {}", "xs_builtin_name_collision", "error"},
		{"reserved parameter", "void f(string label=\"x\") {}", "xs_reserved_identifier", "error"},
		{"branches", "void f() { if(true) { bool b=false; } else { bool b=true; } }", "xs_local_redeclaration", "error"},
		{"parameter", "void f(string s=\"\") { string s=\"x\"; }", "xs_local_redeclaration", "error"},
		{"rule", "rule tick active { vector v=vector(0,0,0); while(true) { vector v=vector(1,1,1); } }", "xs_local_redeclaration", "error"},
		{"unicode", "// café\nvoid main() { xsChatData(\"café\"); }", "xs_non_ascii_string", "error"},
		{"modulo", "if (e % 2 == 0) {}", "xs_condition_modulo", "error"},
		{"snake c2 mixed modulo", "void updateShell(int ticks=0) { int blink=0; if (ticks <= 120 && gTick % 2 == 0) blink=1; }", "xs_condition_modulo", "error"},
		{"while modulo", "while ((e % 2) == 0) {}", "xs_condition_modulo", "error"},
		{"condition subtraction", "if (now - born >= 12) {}", "xs_condition_arithmetic", "error"},
		{"chain", "if ((a) && (b) || (c)) {}", "xs_condition_boolean_chain", "error"},
		{"logical call", "void f() { if (ready && xsCreateFile(true)) {} }", "xs_logical_operand_call", "error"},
		{"logical type", "int probe() { return (1); } void f() { if (false && probe()) {} }", "xs_logical_operand_type", "error"},
		{"condition type", "int probe() { return (1); } void f() { if (probe()) {} }", "xs_condition_non_bool", "error"},
		{"float bool promotion", "void emit(float value=0.0) {} void f() { emit(xsHasResearchedLocalTechnology(1,22)); }", "xs_float_argument_promotion", "error"},
		{"int bool comparison", "void f() { bool mirror=true; if (xsArrayGetInt(flags,0) != mirror) {} }", "xs_int_bool_comparison", "error"},
		{"unsupported type", "void f() { long n=1; }", "xs_unsupported_type", "error"},
		{"return parentheses", "int f() { return 1+1; }", "xs_return_parentheses", "error"},
		{"single quote", "void f() { string s='x'; }", "xs_single_quote_string", "error"},
		{"call args", "void f() { g(1,2,3,4,5,6,7,8,9,10,11,12,13); }", "xs_call_argument_limit", "error"},
		{"unary negative", "void f(int n=1) { int x=-n; }", "xs_unary_negative", "error"},
		{"mixed", "void f(int n=0) { float x=0.2; float y=n*x; }", "xs_int_first_arithmetic", "warning"},
		{"implicit loop int", "void f() { for (letter=0; <3) { float x=letter*9.0+xsArrayGetFloat(points,letter); } }", "xs_int_first_arithmetic", "warning"},
		{"outline arithmetic", "void f() { for (letter=0; <32) { float across=scale*(letter*9.0+xsArrayGetFloat(points,letter)); } }", "xs_int_first_arithmetic", "warning"},
		{"discarded", "void f() { xsGetPlayerUnitIds(1,906); }", "xs_unit_ids_discarded", "warning"},
		{"no size", "void f() { int a=xsGetPlayerUnitIds(1,906); xsArrayGetInt(a,0); }", "xs_unit_ids_without_size", "warning"},
		{"vector order comparison", "void f(vector a=vector(0,0,0), vector b=vector(1,1,1)) { if (a < b) {} }", "xs_vector_order_comparison", "warning"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			found := false
			for _, f := range LintSource("test.xs", c.source) {
				if f.Code == c.code && f.Severity == c.severity {
					found = true
					if f.File != "test.xs" || f.Line < 1 || f.Evidence == "" {
						t.Fatalf("missing provenance: %+v", f)
					}
					catalogued := strings.Contains(c.code, "int_bool") || strings.Contains(c.code, "call_argument") || strings.Contains(c.code, "unsupported_type") || strings.Contains(c.code, "return_parentheses") || strings.Contains(c.code, "single_quote") || strings.Contains(c.code, "unary_negative")
					if catalogued && !strings.Contains(f.Evidence, "Language Syntax.md") {
						t.Fatalf("missing catalogue citation: %+v", f)
					}
				}
			}
			if !found {
				t.Fatalf("missing %s: %+v", c.code, LintSource("test.xs", c.source))
			}
		})
	}
}

func TestCreateUnitResultMustBeChecked(t *testing.T) {
	unchecked := `void f() { int grave = xsCreateUnit(820, 1, vector(40.5,50.5,0.0), false, false, false); }`
	findings := LintSourceWithOptions("generated.xs", unchecked, SourceLintOptions{CheckCreateResults: true})
	if len(findings) == 0 || findings[0].Code != "xs_create_unit_unchecked" {
		t.Fatalf("unchecked xsCreateUnit was accepted: %+v", findings)
	}
	checked := `void f() { int grave = xsCreateUnit(820, 0, vector(40.5,50.5,0.0), false, false, false); if (grave < 0) { xsChatData("failed"); } }`
	if findings := LintSourceWithOptions("generated.xs", checked, SourceLintOptions{CheckCreateResults: true}); len(findings) != 0 {
		t.Fatalf("checked xsCreateUnit was flagged: %+v", findings)
	}
}

func TestGaiaOnlyWrapperChecksConcreteCallSite(t *testing.T) {
	source := `
const int GAIA_ONLY = 820;
int a2kZCreate(int unitType=0, int owner=0, vector position=vector(0,0,0)) {
  return(xsCreateUnit(unitType, owner, position, false, false, false));
}
void good() { a2kZCreate(GAIA_ONLY, 0, vector(1,1,0)); }
void bad() { a2kZCreate(GAIA_ONLY, 1, vector(2,2,0)); }
`
	var findings []SourceFinding
	lintGaiaOnlyCreatesWithLookup(lexLintSource(source), func(unitID int) ([]datcache.UnitRow, error) {
		if unitID != 820 {
			return nil, nil
		}
		return []datcache.UnitRow{
			{ID: 820, CivIndex: 0, Name: "GRAVES"},
		}, nil
	}, func(code, severity string, token lintToken, message, fix string) {
		findings = append(findings, SourceFinding{Code: code, Severity: severity, Line: token.line, Message: message, Fix: fix})
	})
	if len(findings) != 1 || findings[0].Code != "xs_gaia_only_unit_for_player" || findings[0].Line != 7 {
		t.Fatalf("wrapper call-site check: %+v", findings)
	}
}

func TestBareReturnExpressionRequiresParentheses(t *testing.T) {
	for _, source := range []string{
		"int f() { return 1; }",
		"int f(int value=0) { return value; }",
		"int f(int value=0) { if (value < 0) return 0; return (value); }",
	} {
		findings := LintSource("return.xs", source)
		found := false
		for _, finding := range findings {
			if finding.Code == "xs_return_parentheses" && finding.Severity == "error" {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("missing bare-return finding for %q: %+v", source, findings)
		}
	}
	if findings := LintSource("return.xs", "int f(int value=0) { return (value); }"); len(findings) != 0 {
		t.Fatalf("parenthesized return falsely flagged: %+v", findings)
	}
}

func TestVectorEqualityComparisonIsAllowed(t *testing.T) {
	source := "void f(vector a=vector(0,0,0), vector b=vector(1,1,1)) { if (a == b) {} }"
	for _, finding := range LintSource("vector.xs", source) {
		if finding.Code == "xs_vector_order_comparison" {
			t.Fatalf("equality comparison falsely flagged: %+v", finding)
		}
	}
}

func TestLogicalOperandAllowsUserDefinedProbe(t *testing.T) {
	source := "int probe() { return (1); } void f() { if (false && probe()) {} }"
	for _, finding := range LintSource("test.xs", source) {
		if finding.Code == "xs_logical_operand_call" {
			t.Fatalf("user-defined probe was rejected: %+v", finding)
		}
	}
}

func TestEngineHazardLintSafeForms(t *testing.T) {
	for _, source := range []string{
		"void f() { vector a=vector(-1.0,+0.5,1e-3); vector b=xsVectorSet(x,y,-1.0); }",
		"void f() { string s=\"vector(x,y,z)\"; /* vector(a,b,c) */ }",
		"void f() { vector a=vector(0,0,0); vector b=vector(1,1,1); }",
		"void f() { for(i=0; <3) { int length=0; } while(false) float other=0; }",
		"void f() { float x=ln(2.0); string s=\"int ln=0;\"; /* int sin=0; */ }",
		"int x=0; void f() { int x=1; for(i=0; <3) { x=i; } } void g() { int x=2; }",
		"void f() { string s=\"for(i=0; <3) { int x=0; int x=1; }\"; /* int s=0; */ }",
		"/* café if (a % 2 && b && c) */ // more café\nvoid f() { xsChatData(\"if (a % 2 && b && c)\"); }",
		"void f() { int m=e%2; if ((m==0) && ((b==1)||(c==2))) {} }",
		"void f() { if (old != -1) {} }",
		"void f() { float x=0.2; int n=3; float y=x*n; }",
		"void f() { int a=xsGetPlayerUnitIds(1,4); int n=xsArrayGetSize(a); }", // 4 can mean a unit, not a class.
		"void f() { int n=xsArrayGetSize(xsGetPlayerUnitIds(1,906)); }",
		"void f() { xsChatData(\"escaped \\\" quote\"); }",
	} {
		if got := LintSource("safe.xs", source); len(got) > 0 {
			t.Fatalf("false positive for %q: %+v", source, got)
		}
	}
}

func TestForwardFunctionCall(t *testing.T) {
	source := `int caller() { return (later()); }
int later() { return (1); }`
	findings := LintSource("forward.xs", source)
	for _, finding := range findings {
		if finding.Code == "xs_forward_function_call" && finding.Severity == "error" {
			return
		}
	}
	t.Fatalf("missing forward-call finding: %+v", findings)
}

func TestForwardFunctionCallDoesNotFlagBuiltin(t *testing.T) {
	findings := LintSource("builtin.xs", `int caller() { return (xsGetGameTime()); }`)
	for _, finding := range findings {
		if finding.Code == "xs_forward_function_call" {
			t.Fatalf("builtin falsely flagged: %+v", finding)
		}
	}
}

func TestAnalyzeIncludesLintAllReachableSources(t *testing.T) {
	r, err := AnalyzeFiles(AnalyzeOptions{EntryPath: "main.xs", Files: []SourceFile{{"main.xs", "include \"child.xs\";"}, {"child.xs", "while (x%2==0) {}"}, {"unloaded.xs", "if(a&&b&&c){}"}}})
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Findings) != 1 || r.Findings[0].File != "child.xs" {
		t.Fatalf("findings %+v", r.Findings)
	}
}
