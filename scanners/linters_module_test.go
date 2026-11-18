package scanners

import (
	"encoding/json"
	"testing"
)

func TestShouldSkipLinterPath(t *testing.T) {
	cases := map[string]bool{
		"testdata/fixtures/foo.go": true,
		"vendor/foo/bar.go":        true,
		"benchmark/fixture/x.go":   true,
		"analyzers/engine.go":      false,
		"cmd/main.go":              false,
	}
	for path, want := range cases {
		if got := shouldSkipLinterPath(path); got != want {
			t.Fatalf("shouldSkipLinterPath(%q) = %v want %v", path, got, want)
		}
	}
}

func TestParseGolangciOutputSkipsFixturePaths(t *testing.T) {
	payload := `{
		"Issues": [
			{
				"Text": "real issue",
				"FromLinter": "govet",
				"Severity": "warning",
				"Pos": {"Filename": "analyzers/engine.go", "Line": 10}
			},
			{
				"Text": "fixture noise",
				"FromLinter": "govet",
				"Severity": "warning",
				"Pos": {"Filename": "testdata/fixtures/foo.go", "Line": 3}
			}
		]
	}`
	findings, err := parseGolangciOutput([]byte(payload), "/workspace", "", Config{LinterMinSeverity: "low"})
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding after skip, got %d", len(findings))
	}
	if findings[0].File != "analyzers/engine.go" {
		t.Fatalf("unexpected file %q", findings[0].File)
	}
}

func TestParseGolangciOutputExtractsEmbeddedJSON(t *testing.T) {
	payload := []byte("level=info msg=\"run\"\n" + `{"Issues":[{"Text":"x","FromLinter":"errcheck","Severity":"warning","Pos":{"Filename":"main.go","Line":1}}],"Error":""}`)
	findings, err := parseGolangciOutput(payload, "/workspace", "", Config{LinterMinSeverity: "low"})
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
}

func TestGolangciReportErrorField(t *testing.T) {
	var report golangciReport
	if err := json.Unmarshal([]byte(`{"Issues":[],"Error":"typechecking failed"}`), &report); err != nil {
		t.Fatal(err)
	}
	if report.Error != "typechecking failed" {
		t.Fatalf("unexpected error field %q", report.Error)
	}
}
