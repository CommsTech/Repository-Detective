package runner_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"git.commsnet.org/commstech/repository-detective/analyzers"
	"git.commsnet.org/commstech/repository-detective/runner"
	"git.commsnet.org/commstech/repository-detective/scanners"
)

func TestRunnerResultRejectsInvalidEnvelope(t *testing.T) {
	valid := runner.JobResult{
		Version: runner.ContractVersion,
		JobID:   "rj-test", ScanID: "scan-test", Status: runner.JobStatusCompleted,
		StartedAt: time.Now().UTC().Add(-time.Second), FinishedAt: time.Now().UTC(),
	}
	job := runner.JobView{JobID: valid.JobID, ScanID: valid.ScanID}
	cases := []struct {
		name string
		edit func(*runner.JobResult)
	}{
		{"missing version", func(r *runner.JobResult) { r.Version = 0 }},
		{"unsupported version", func(r *runner.JobResult) { r.Version = runner.ContractVersion + 1 }},
		{"missing status", func(r *runner.JobResult) { r.Status = "" }},
		{"invented success status", func(r *runner.JobResult) { r.Status = "verified_safe" }},
		{"negative file count", func(r *runner.JobResult) { r.FilesAnalyzed = -1 }},
		{"negative scanner count", func(r *runner.JobResult) {
			r.ScannerResults = []runner.ScannerResultDTO{{Scanner: "trivy", FindingsCount: -1}}
		}},
		{"reversed timestamps", func(r *runner.JobResult) {
			r.FinishedAt = r.StartedAt.Add(-time.Second)
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := valid
			tc.edit(&got)
			if err := runner.ValidateResultBasic(job, got, 1024*1024); err == nil {
				t.Fatal("invalid worker payload was accepted")
			}
		})
	}
	if err := runner.ValidateResultBasic(job, valid, 1024*1024); err != nil {
		t.Fatalf("valid envelope rejected: %v", err)
	}
}

func TestRunnerScannerEvidenceCountAndErrorSurviveConversion(t *testing.T) {
	result := runner.JobResult{
		Version: runner.ContractVersion,
		ScanID:  "scan-1", JobID: "rj-1", Status: runner.JobStatusCompleted,
		ScannerResults: []runner.ScannerResultDTO{
			{Scanner: "gosec", Status: string(scanners.StatusFound), FindingsCount: 7, Detail: "analyzed Go files"},
			{Scanner: "trivy", Status: string(scanners.StatusClean), Error: "scanner crashed; api_key=supersecretvalue"},
		},
	}
	analysis := result.ToAnalysisResult("o/r", "main", nil)
	if len(analysis.ScannerResults) != 2 {
		t.Fatalf("expected two outcomes, got %d", len(analysis.ScannerResults))
	}
	if analysis.ScannerResults[0].ReportedFindingsCount != 7 || analysis.ScannerResults[0].Status != scanners.StatusFound {
		t.Fatalf("worker findings lost: %+v", analysis.ScannerResults[0])
	}
	failed := analysis.ScannerResults[1]
	if failed.Status != scanners.StatusFailed || !strings.Contains(failed.Detail, "scanner crashed") {
		t.Fatalf("worker scanner error silently passed: %+v", failed)
	}
	if strings.Contains(failed.Detail, "supersecretvalue") {
		t.Fatalf("unredacted scanner credential in audit detail: %q", failed.Detail)
	}
}

func TestCustomRulesExposeStandardScannerStatuses(t *testing.T) {
	workspace := t.TempDir()
	if err := os.MkdirAll(filepath.Join(workspace, "src"), 0700); err != nil {
		t.Fatal(err)
	}
	policy := `{"version":1,"rules":[{"id":"CUSTOM_TODO","title":"Review TODO","category":"quality","severity":"low","path_glob":"src/*.go","pattern":"TODO"}]}`
	if err := os.WriteFile(filepath.Join(workspace, ".repository-detective.rules.json"), []byte(policy), 0600); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(workspace, "src", "app.go")
	for _, tc := range []struct {
		content string
		want    scanners.Status
	}{
		{"package app\n// TODO: investigate\n", scanners.StatusFound},
		{"package app\n// stable implementation\n", scanners.StatusClean},
	} {
		if err := os.WriteFile(path, []byte(tc.content), 0600); err != nil {
			t.Fatal(err)
		}
		spec := runner.JobSpec{
			JobID: "rj-rules", ScanID: "scan-rules",
			AllowedTasks:      []string{"scanners"},
			EffectiveSettings: analyzers.PolicySnapshot{AnalysisDepth: 1},
		}
		got, err := runner.ExecuteWorkspaceScan(context.Background(), spec, runner.ExecuteInput{WorkspaceDir: workspace})
		if err != nil || got.Status != runner.JobStatusCompleted {
			t.Fatalf("custom rule scan failed: %+v err %v", got, err)
		}
		found := false
		for _, sr := range got.ScannerResults {
			if sr.Scanner != "custom_rules" {
				continue
			}
			found = true
			if sr.Status != string(tc.want) || (tc.want == scanners.StatusFound && sr.FindingsCount == 0) {
				t.Fatalf("custom-rule status/count incorrect: %+v want %s", sr, tc.want)
			}
		}
		if !found {
			t.Fatal("missing custom_rules scanner outcome")
		}
	}
}
