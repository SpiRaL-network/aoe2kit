package xs

import "testing"

func TestAnalyzeFilesIncludeAndCrossFileExternLint(t *testing.T) {
	report, err := AnalyzeFiles(AnalyzeOptions{
		EntryPath: "main.xs",
		Files: []SourceFile{
			{Path: "main.xs", Content: "include \"constants.xs\";\nvoid Boot() { xsSetPlayerAttribute(1, RTV_BAD, RTV_OK); }\n"},
			{Path: "constants.xs", Content: "const int RTV_BAD = 1;\nextern const int RTV_OK = 2;\n"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Includes) != 1 || report.Includes[0].Status != "resolved" {
		t.Fatalf("includes = %+v", report.Includes)
	}
	if len(report.CrossFileUses) != 1 || report.CrossFileUses[0].Name != "RTV_BAD" {
		t.Fatalf("cross-file uses = %+v", report.CrossFileUses)
	}
}

func TestAnalyzeFilesReportsMissingInclude(t *testing.T) {
	report, err := AnalyzeFiles(AnalyzeOptions{
		EntryPath: "main.xs",
		Files:     []SourceFile{{Path: "main.xs", Content: "include \"missing.xs\";\nvoid Boot() {}\n"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Includes) != 1 || report.Includes[0].Status != "missing" {
		t.Fatalf("includes = %+v", report.Includes)
	}
}
