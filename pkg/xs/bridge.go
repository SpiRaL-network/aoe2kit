package xs

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
)

type VerificationClaim struct {
	Label             string `json:"label"`
	StructureVerified bool   `json:"structure_verified"`
	EngineVerified    bool   `json:"engine_verified"`
	Note              string `json:"note"`
}

func StructureVerifiedClaim(note string) VerificationClaim {
	return VerificationClaim{
		Label:             "structure_verified_not_engine_verified",
		StructureVerified: true,
		EngineVerified:    false,
		Note:              note,
	}
}

type BridgeSpec struct {
	Variables     map[string]int `json:"variables"`
	Reads         []string       `json:"reads,omitempty"`
	Writes        []string       `json:"writes,omitempty"`
	XSReads       []string       `json:"xs_reads,omitempty"`
	XSWrites      []string       `json:"xs_writes,omitempty"`
	TriggerReads  []string       `json:"trigger_reads,omitempty"`
	TriggerWrites []string       `json:"trigger_writes,omitempty"`
	ModuleName    string         `json:"module_name,omitempty"`
}

type BridgeVariable struct {
	Name       string `json:"name"`
	ID         int    `json:"id"`
	XSName     string `json:"xs_name"`
	ConstName  string `json:"const_name"`
	GetterName string `json:"getter_name"`
}

type BridgeIssue struct {
	Severity string `json:"severity"`
	Variable string `json:"variable,omitempty"`
	Message  string `json:"message"`
}

type BridgeReport struct {
	Verification        VerificationClaim `json:"verification"`
	ModuleName          string            `json:"module_name"`
	Variables           []BridgeVariable  `json:"variables"`
	XSModule            string            `json:"xs_module"`
	TriggerVariableDefs []BridgeVariable  `json:"trigger_variable_defs"`
	Issues              []BridgeIssue     `json:"issues,omitempty"`
	OK                  bool              `json:"ok"`
}

func LoadBridgeSpec(path string) (BridgeSpec, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return BridgeSpec{}, err
	}
	var spec BridgeSpec
	if err := json.Unmarshal(data, &spec); err == nil && len(spec.Variables) > 0 {
		return spec, nil
	}
	var raw map[string]int
	if err := json.Unmarshal(data, &raw); err != nil {
		return BridgeSpec{}, err
	}
	return BridgeSpec{Variables: raw}, nil
}

func GenerateBridge(spec BridgeSpec) (BridgeReport, error) {
	if len(spec.Variables) == 0 {
		return BridgeReport{}, fmt.Errorf("bridge spec has no variables")
	}
	moduleName := spec.ModuleName
	if moduleName == "" {
		moduleName = "a2k_variables"
	}
	if err := validateIdentifier(moduleName); err != nil {
		return BridgeReport{}, fmt.Errorf("module_name: %w", err)
	}
	names := make([]string, 0, len(spec.Variables))
	ids := map[int]string{}
	for name, id := range spec.Variables {
		if err := validateIdentifier(name); err != nil {
			return BridgeReport{}, fmt.Errorf("variable %q: %w", name, err)
		}
		if id < 0 {
			return BridgeReport{}, fmt.Errorf("variable %q has negative id %d", name, id)
		}
		if prev, ok := ids[id]; ok {
			return BridgeReport{}, fmt.Errorf("variables %q and %q share id %d", prev, name, id)
		}
		ids[id] = name
		names = append(names, name)
	}
	sort.Strings(names)
	variables := make([]BridgeVariable, 0, len(names))
	for _, name := range names {
		id := spec.Variables[name]
		variables = append(variables, BridgeVariable{
			Name:       name,
			ID:         id,
			XSName:     fmt.Sprintf("xsVariable%d", id),
			ConstName:  "A2K_VAR_" + strings.ToUpper(name),
			GetterName: "get_" + name,
		})
	}
	report := BridgeReport{
		Verification:        StructureVerifiedClaim("XS bridge code and trigger-variable definition JSON are generated from declared ids; engine behavior must still be proven in a scenario run."),
		ModuleName:          moduleName,
		Variables:           variables,
		TriggerVariableDefs: variables,
		OK:                  true,
	}
	report.Issues = lintBridge(spec)
	if len(report.Issues) > 0 {
		report.OK = false
	}
	report.XSModule = renderBridgeXS(moduleName, variables)
	return report, nil
}

func lintBridge(spec BridgeSpec) []BridgeIssue {
	declared := map[string]bool{}
	for name := range spec.Variables {
		declared[name] = true
	}
	reads := setFromSlices(spec.Reads, spec.XSReads, spec.TriggerReads)
	writes := setFromSlices(spec.Writes, spec.XSWrites, spec.TriggerWrites)
	issues := []BridgeIssue{}
	for name := range reads {
		if !declared[name] {
			issues = append(issues, BridgeIssue{Severity: "error", Variable: name, Message: "variable is read but not declared"})
			continue
		}
		if !writes[name] {
			issues = append(issues, BridgeIssue{Severity: "warning", Variable: name, Message: "variable is read but never written"})
		}
	}
	for name := range writes {
		if !declared[name] {
			issues = append(issues, BridgeIssue{Severity: "error", Variable: name, Message: "variable is written but not declared"})
			continue
		}
		if !reads[name] {
			issues = append(issues, BridgeIssue{Severity: "warning", Variable: name, Message: "variable is written but never read"})
		}
	}
	sort.Slice(issues, func(i, j int) bool {
		if issues[i].Severity == issues[j].Severity {
			return issues[i].Variable < issues[j].Variable
		}
		return issues[i].Severity < issues[j].Severity
	})
	return issues
}

func renderBridgeXS(moduleName string, variables []BridgeVariable) string {
	var b strings.Builder
	b.WriteString("// Generated by AoE2Kit. Do not hand-edit anonymous xsVariableN ids here.\n")
	b.WriteString("// Module: ")
	b.WriteString(moduleName)
	b.WriteString("\n\n")
	for _, variable := range variables {
		fmt.Fprintf(&b, "extern int %s = -1;\n", variable.XSName)
	}
	b.WriteString("\n")
	for _, variable := range variables {
		fmt.Fprintf(&b, "extern const int %s = %d;\n", variable.ConstName, variable.ID)
	}
	b.WriteString("\n")
	for _, variable := range variables {
		fmt.Fprintf(&b, "int %s() {\n", variable.GetterName)
		fmt.Fprintf(&b, "    return(xsTriggerVariable(%s));\n", variable.ConstName)
		b.WriteString("}\n\n")
	}
	return b.String()
}

func WriteBridgeOutputs(report BridgeReport, outDir string) error {
	if outDir == "" {
		return nil
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(outDir+"/"+report.ModuleName+".xs", []byte(report.XSModule), 0o644); err != nil {
		return err
	}
	data, err := json.MarshalIndent(struct {
		Verification VerificationClaim `json:"verification"`
		Variables    []BridgeVariable  `json:"variables"`
	}{
		Verification: report.Verification,
		Variables:    report.TriggerVariableDefs,
	}, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(outDir+"/trigger_variables.json", append(data, '\n'), 0o644); err != nil {
		return err
	}
	lintData, err := json.MarshalIndent(struct {
		OK     bool          `json:"ok"`
		Issues []BridgeIssue `json:"issues,omitempty"`
	}{
		OK:     report.OK,
		Issues: report.Issues,
	}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(outDir+"/lint.json", append(lintData, '\n'), 0o644)
}

func setFromSlices(groups ...[]string) map[string]bool {
	out := map[string]bool{}
	for _, group := range groups {
		for _, item := range group {
			out[item] = true
		}
	}
	return out
}

var identifierRE = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func validateIdentifier(name string) error {
	if name == "" {
		return fmt.Errorf("empty identifier")
	}
	if !identifierRE.MatchString(name) {
		return fmt.Errorf("invalid XS identifier")
	}
	return nil
}

func writeJSONFile(path string, value any) error {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetIndent("", "  ")
	if err := enc.Encode(value); err != nil {
		return err
	}
	return os.WriteFile(path, b.Bytes(), 0o644)
}
