package runner

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"git.commsnet.org/commstech/repository-detective/analyzers"
	"git.commsnet.org/commstech/repository-detective/scanners"
)

func TestRemediationVerificationRejectsMissingAndFailedScannerEvidence(t *testing.T) {
	tests := []struct {
		name   string
		result JobResult
		want   string
	}{
		{"empty", JobResult{}, "no analyzed source files"},
		{"disabled only", JobResult{FilesAnalyzed: 1, ScannerResults: []ScannerResultDTO{{Scanner: "trivy", Status: string(scanners.StatusDisabled)}}}, "no completed applicable external scanner"},
		{"nonapplicable clean not verification", JobResult{FilesAnalyzed: 1, ScannerResults: []ScannerResultDTO{{Scanner: "govulncheck", Status: string(scanners.StatusClean), Detail: "no Go module or files"}}}, "no completed applicable external scanner"},
		{"scanner unavailable", JobResult{FilesAnalyzed: 1, ScannerResults: []ScannerResultDTO{{Scanner: "trivy", Status: string(scanners.StatusBinaryMissing)}}}, "scanner trivy incomplete"},
		{"finding reported", JobResult{FilesAnalyzed: 1, ScannerResults: []ScannerResultDTO{{Scanner: "semgrep", Status: string(scanners.StatusFound)}}}, "unresolved findings"},
		{"unresolved built-in finding", JobResult{FilesAnalyzed: 1, ScannerResults: []ScannerResultDTO{{Scanner: "trivy", Status: string(scanners.StatusClean)}}, Findings: []FindingResult{{RuleID: "rule1"}}}, "target-specific closure is not supported"},
		{"unknown scanner status", JobResult{FilesAnalyzed: 1, ScannerResults: []ScannerResultDTO{{Scanner: "custom", Status: "unknown"}}}, "scanner custom incomplete"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gaps := remediationVerificationGaps(tt.result)
			if !strings.Contains(strings.Join(gaps, " | "), tt.want) {
				t.Fatalf("want %q in gaps %v", tt.want, gaps)
			}
		})
	}
}

func TestRemediationVerificationCleanEvidenceIsOnlyScanEvidence(t *testing.T) {
	result := JobResult{FilesAnalyzed: 3, ScannerResults: []ScannerResultDTO{{Scanner: "trivy", Status: string(scanners.StatusClean)}, {Scanner: "semgrep", Status: string(scanners.StatusDisabled)}}}
	if gaps := remediationVerificationGaps(result); len(gaps) != 0 {
		t.Fatalf("unexpected gaps: %v", gaps)
	}
}

func TestRemediationVerificationDoesNotSucceedWithScannersDisabled(t *testing.T) {
	workspace := t.TempDir()
	if err := os.WriteFile(filepath.Join(workspace, "main.go"), []byte("package demo\nfunc main() {}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	spec := JobSpec{JobID: "rj-a", ScanID: "scan-a", JobType: JobTypeRemediationVerify, EffectiveSettings: analyzers.PolicySnapshot{AnalysisDepth: 1}}
	result, err := ExecuteJob(context.Background(), spec, JobExecuteInput{WorkspaceDir: workspace, ScannerCfg: scanners.Config{}})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != JobStatusFailed {
		t.Fatalf("scan without scanner coverage must fail closed: %+v", result)
	}
	if len(result.Errors) == 0 {
		t.Fatal("missing operator-visible error")
	}
}

func TestUnknownJobTypeNeverExecutesScan(t *testing.T) {
	result, err := ExecuteJob(context.Background(), JobSpec{JobID: "rj-a", ScanID: "scan-a", JobType: "unexpected_job"}, JobExecuteInput{})
	if err != nil || result.Status != JobStatusFailed || len(result.Errors) == 0 {
		t.Fatalf("unexpected result %v %+v", err, result)
	}
	if !strings.Contains(result.Errors[0], "unsupported runner job type") {
		t.Fatalf("missing failure: %+v", result.Errors)
	}
}

func TestEmptyWorkerAllowlistDeniesAllJobs(t *testing.T) {
	if JobTypeAllowed(nil, JobTypeScanFullRepo) {
		t.Fatal("empty allowlist grants scan authority")
	}
	if JobTypeAllowed([]string{}, JobTypeRemediationVerify) {
		t.Fatal("empty allowlist grants verification authority")
	}
}
