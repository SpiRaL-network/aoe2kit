package replay

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type TelemetrySchema struct {
	Prefixes []string                        `json:"prefixes,omitempty"`
	Events   map[string]TelemetryEventSchema `json:"events"`
}

type TelemetryEventSchema struct {
	Required []string `json:"required,omitempty"`
	Optional []string `json:"optional,omitempty"`
}

type TelemetryReport struct {
	Path     string           `json:"path"`
	OK       bool             `json:"ok"`
	Events   []TelemetrySeen  `json:"events,omitempty"`
	Issues   []TelemetryIssue `json:"issues,omitempty"`
	Warnings []string         `json:"warnings,omitempty"`
}

type TelemetrySeen struct {
	Time     string            `json:"time,omitempty"`
	PlayerID int               `json:"player_id,omitempty"`
	Prefix   string            `json:"prefix"`
	Name     string            `json:"name,omitempty"`
	Fields   map[string]string `json:"fields,omitempty"`
	Raw      string            `json:"raw"`
}

type TelemetryIssue struct {
	Severity string `json:"severity"`
	Code     string `json:"code"`
	Message  string `json:"message"`
}

func LoadTelemetrySchema(path string) (*TelemetrySchema, error) {
	if path == "" {
		return &TelemetrySchema{}, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var schema TelemetrySchema
	if err := json.Unmarshal(data, &schema); err != nil {
		return nil, fmt.Errorf("%s is not valid telemetry schema JSON: %w", path, err)
	}
	if schema.Events == nil {
		schema.Events = map[string]TelemetryEventSchema{}
	}
	return &schema, nil
}

func ValidateTelemetry(path string, contextPath string, schemaPath string) (*TelemetryReport, error) {
	context, err := LoadContext(contextPath)
	if err != nil {
		return nil, err
	}
	schema, err := LoadTelemetrySchema(schemaPath)
	if err != nil {
		return nil, err
	}
	prefixes := context.TelemetryPrefixes
	if len(schema.Prefixes) > 0 {
		prefixes = schema.Prefixes
	}
	events, err := ExtractEvents(path, EventOptions{TelemetryPrefixes: prefixes})
	if err != nil {
		return nil, err
	}
	report := &TelemetryReport{Path: path, Warnings: events.Warnings}
	for _, event := range events.Events {
		if event.Telemetry == nil {
			continue
		}
		seen := TelemetrySeen{
			Time:     event.Time,
			PlayerID: event.PlayerID,
			Prefix:   event.Telemetry.Prefix,
			Name:     event.Telemetry.Name,
			Fields:   event.Telemetry.Fields,
			Raw:      event.Telemetry.Raw,
		}
		report.Events = append(report.Events, seen)
		if event.Telemetry.Name == "" {
			report.addTelemetryIssue("error", "missing_name", fmt.Sprintf("telemetry marker has no event name: %q", event.Telemetry.Raw))
			continue
		}
		eventSchema, known := schema.Events[event.Telemetry.Name]
		if len(schema.Events) > 0 && !known {
			report.addTelemetryIssue("warning", "unknown_event", fmt.Sprintf("telemetry event %q is not in schema", event.Telemetry.Name))
			continue
		}
		for _, required := range eventSchema.Required {
			if _, ok := event.Telemetry.Fields[required]; !ok {
				report.addTelemetryIssue("error", "missing_required_field", fmt.Sprintf("%s missing required field %q", event.Telemetry.Name, required))
			}
		}
	}
	report.OK = !hasTelemetrySeverity(report.Issues, "error")
	return report, nil
}

func ValidateTelemetryFolder(root string, contextPath string, schemaPath string) ([]TelemetryReport, error) {
	var reports []TelemetryReport
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !isReplayPath(path) {
			return nil
		}
		report, runErr := ValidateTelemetry(path, contextPath, schemaPath)
		if runErr != nil {
			reports = append(reports, TelemetryReport{Path: path, OK: false, Issues: []TelemetryIssue{{Severity: "error", Code: "parse_failed", Message: runErr.Error()}}})
			return nil
		}
		reports = append(reports, *report)
		return nil
	})
	sort.Slice(reports, func(i, j int) bool { return reports[i].Path < reports[j].Path })
	return reports, err
}

func (r *TelemetryReport) addTelemetryIssue(severity, code, message string) {
	r.Issues = append(r.Issues, TelemetryIssue{Severity: severity, Code: code, Message: message})
}

func hasTelemetrySeverity(issues []TelemetryIssue, severity string) bool {
	for _, issue := range issues {
		if strings.EqualFold(issue.Severity, severity) {
			return true
		}
	}
	return false
}
