package store_test

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"git.commsnet.org/commstech/repository-detective/store"
)

func TestListFindingsForScanRanksSeveritySemantically(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	s, err := store.Open(store.Config{Enabled: true, Path: filepath.Join(dir, "ai-review.db")})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer s.Close()

	repo, err := s.UpsertRepository(ctx, store.Repository{Owner: "commstech", Name: "Wifi_Collector", FullName: "commstech/Wifi_Collector"})
	if err != nil {
		t.Fatalf("repo: %v", err)
	}
	scanID := "483e858c63e7e8a2"
	if _, err := s.CreateScan(ctx, store.Scan{
		ID: scanID, RepositoryID: repo.ID, TriggerType: store.TriggerManual,
		Status: store.ScanStatusCompleted, CommitSHA: "bde0ca99a72b11102e77bc449f4e6d221857a44c",
		StartedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("scan: %v", err)
	}

	now := time.Now().UTC()
	seed := []store.Finding{
		{RepositoryID: repo.ID, Fingerprint: "fp-info", Category: "quality", Severity: "info", Confidence: 0.99, Source: "ruff", RuleID: "F401", Title: "unused", Status: store.FindingStatusOpen, FirstSeenScanID: scanID, LastSeenScanID: scanID, FirstSeenAt: now, LastSeenAt: now},
		{RepositoryID: repo.ID, Fingerprint: "fp-low", Category: "tech_debt", Severity: "low", Confidence: 0.95, Source: "tech_debt", RuleID: "HEALTH-TECH-MARKER", Title: "debt", Status: store.FindingStatusOpen, FirstSeenScanID: scanID, LastSeenScanID: scanID, FirstSeenAt: now, LastSeenAt: now},
		{RepositoryID: repo.ID, Fingerprint: "fp-medium", Category: "maintainability", Severity: "medium", Confidence: 0.9, Source: "static", RuleID: "HEALTH-LARGE-FILE", Title: "large", Status: store.FindingStatusOpen, FirstSeenScanID: scanID, LastSeenScanID: scanID, FirstSeenAt: now, LastSeenAt: now},
		{RepositoryID: repo.ID, Fingerprint: "rd-07d2a81ff8aff859", Category: "dependency", Severity: "high", Confidence: 0.98, Source: "trivy", RuleID: "TRIVY-CVE-2024-26130", PackageName: "cryptography", FilePath: "requirements.txt", Title: "cryptography vuln", Status: store.FindingStatusOpen, FirstSeenScanID: scanID, LastSeenScanID: scanID, FirstSeenAt: now, LastSeenAt: now},
	}
	var highID int64
	for _, f := range seed {
		saved, err := s.UpsertFinding(ctx, f)
		if err != nil {
			t.Fatalf("upsert finding %s: %v", f.Fingerprint, err)
		}
		if f.Fingerprint == "rd-07d2a81ff8aff859" {
			highID = saved.ID
		}
		ev := "noise"
		if f.Fingerprint == "rd-07d2a81ff8aff859" {
			ev = "python-cryptography: NULL pointer dereference (redacted cluster evidence)"
		}
		inst := store.FindingInstance{
			FindingID: saved.ID, ScanID: scanID, EvidenceRedacted: ev, CreatedAt: now,
		}
		if f.Fingerprint == "rd-07d2a81ff8aff859" {
			// Non-empty TEXT JSON — modernc must convert string→json.RawMessage on read.
			inst.LocationJSON = json.RawMessage(`{"file":"requirements.txt","line":0}`)
			inst.RawMetadataJSON = json.RawMessage(`{"source":"trivy","rule_id":"TRIVY-CVE-2024-26130"}`)
		}
		if err := s.AddFindingInstance(ctx, inst); err != nil {
			t.Fatalf("instance: %v", err)
		}
	}

	// Flood the scan with extra low findings so a bad ORDER BY + LIMIT would drop high.
	for i := 0; i < 30; i++ {
		fp := fmt.Sprintf("fp-flood-%02d", i)
		saved, err := s.UpsertFinding(ctx, store.Finding{
			RepositoryID: repo.ID, Fingerprint: fp, Category: "tech_debt", Severity: "low",
			Confidence: 0.99, Source: "tech_debt", RuleID: "HEALTH-TECH-MARKER", Title: "flood",
			Status: store.FindingStatusOpen, FirstSeenScanID: scanID, LastSeenScanID: scanID,
			FirstSeenAt: now, LastSeenAt: now,
		})
		if err != nil {
			t.Fatalf("flood upsert: %v", err)
		}
		_ = s.AddFindingInstance(ctx, store.FindingInstance{
			FindingID: saved.ID, ScanID: scanID, EvidenceRedacted: "flood", CreatedAt: now,
		})
	}

	pool, err := s.ListFindingsForScan(ctx, scanID, 8)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(pool) == 0 {
		t.Fatal("expected findings")
	}
	if pool[0].Fingerprint != "rd-07d2a81ff8aff859" || pool[0].Severity != "high" {
		t.Fatalf("expected high cryptography finding first, got %+v", pool[0])
	}

	// A severity-only LIMIT would be filled by high lint and drop the info CVE.
	for i := 0; i < 20; i++ {
		fp := fmt.Sprintf("fp-lint-%02d", i)
		saved, err := s.UpsertFinding(ctx, store.Finding{
			RepositoryID: repo.ID, Fingerprint: fp, Category: "maintainability", Severity: "high",
			Confidence: 0.99, Source: "golangci-lint", RuleID: "LINT-GO-errcheck", Title: "unchecked close",
			Status: store.FindingStatusOpen, FirstSeenScanID: scanID, LastSeenScanID: scanID,
			FirstSeenAt: now, LastSeenAt: now,
		})
		if err != nil {
			t.Fatalf("lint upsert: %v", err)
		}
		_ = s.AddFindingInstance(ctx, store.FindingInstance{
			FindingID: saved.ID, ScanID: scanID, EvidenceRedacted: "lint", CreatedAt: now,
		})
	}
	lister, ok := s.(interface {
		ListFindingsForAIReview(context.Context, string, int) ([]store.Finding, error)
	})
	if !ok {
		t.Fatal("store does not support the AI review candidate pool")
	}
	aiPool, err := lister.ListFindingsForAIReview(ctx, scanID, 4)
	if err != nil {
		t.Fatalf("ai pool: %v", err)
	}
	if len(aiPool) == 0 || aiPool[0].RuleID != "TRIVY-CVE-2024-26130" {
		t.Fatalf("expected vulnerability ahead of lint, got %+v", aiPool)
	}

	instances, err := s.ListFindingInstancesByScan(ctx, scanID)
	if err != nil {
		t.Fatalf("instances: %v", err)
	}
	inst, ok := instances[highID]
	if !ok || inst.ScanID != scanID {
		t.Fatalf("expected scan-scoped instance for high finding, got %#v ok=%v", inst, ok)
	}
	if inst.EvidenceRedacted == "" {
		t.Fatal("expected preserved redacted evidence")
	}
	if string(inst.LocationJSON) != `{"file":"requirements.txt","line":0}` {
		t.Fatalf("location_json round-trip failed: %s", inst.LocationJSON)
	}
	if !json.Valid(inst.LocationJSON) || !json.Valid(inst.RawMetadataJSON) {
		t.Fatalf("expected valid JSON blobs loc=%s meta=%s", inst.LocationJSON, inst.RawMetadataJSON)
	}
}
