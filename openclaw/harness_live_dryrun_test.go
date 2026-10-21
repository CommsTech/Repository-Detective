package openclaw_test

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"git.commsnet.org/commstech/repository-detective/ai"
	"git.commsnet.org/commstech/repository-detective/openclaw"
	"git.commsnet.org/commstech/repository-detective/store"
	_ "modernc.org/sqlite"
)

// Target scan/repo constants for the Wifi_Collector harness dry-run.
const (
	harnessTargetScanID = "483e858c63e7e8a2"
	harnessTargetRepoID = int64(10)
)

func TestLiveWifiCollectorHarnessDryRun(t *testing.T) {
	dbPath := "../data/repository-detective.db"
	if _, err := os.Stat(dbPath); err != nil {
		t.Skip("live db not available")
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	// Guard: the target scan must exist and belong to the target repository.
	var repoID int64
	if err := db.QueryRow(`SELECT repository_id FROM scans WHERE id=?`, harnessTargetScanID).Scan(&repoID); err != nil {
		if err == sql.ErrNoRows {
			t.Skipf("target scan %q not present in live db", harnessTargetScanID)
		}
		t.Fatal(err)
	}
	if repoID != harnessTargetRepoID {
		t.Skipf("target scan %q belongs to repository %d, not %d", harnessTargetScanID, repoID, harnessTargetRepoID)
	}

	// Scan-scoped selection: only findings with instances in the target scan,
	// only instances recorded for the target scan. No status='open' filter and
	// no cross-scan instance reuse.
	rows, err := db.Query(`
		SELECT f.id, f.fingerprint, IFNULL(f.category,''), f.source, f.severity, f.confidence,
			IFNULL(f.rule_id,''), f.title, f.file_path, f.line, IFNULL(f.package_name,''), f.repository_id,
			fi.evidence_redacted, fi.scan_id
		FROM findings f
		INNER JOIN finding_instances fi ON fi.finding_id = f.id AND fi.scan_id = ?
		WHERE f.repository_id = ?
		ORDER BY f.severity DESC, f.confidence DESC
		LIMIT 500`, harnessTargetScanID, harnessTargetRepoID)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var findings []store.Finding
	instances := map[int64]store.FindingInstance{}
	scanIDs := map[string]bool{}
	for rows.Next() {
		var f store.Finding
		var ev, instScan string
		if err := rows.Scan(&f.ID, &f.Fingerprint, &f.Category, &f.Source, &f.Severity, &f.Confidence,
			&f.RuleID, &f.Title, &f.FilePath, &f.Line, &f.PackageName, &f.RepositoryID, &ev, &instScan); err != nil {
			t.Fatal(err)
		}
		if instScan != harnessTargetScanID {
			t.Fatalf("contamination: instance scan_id=%q for target scan %q", instScan, harnessTargetScanID)
		}
		if f.RepositoryID != harnessTargetRepoID {
			t.Fatalf("contamination: finding repository_id=%d for target repo %d", f.RepositoryID, harnessTargetRepoID)
		}
		findings = append(findings, f)
		instances[f.ID] = store.FindingInstance{EvidenceRedacted: ev, ScanID: instScan}
		scanIDs[instScan] = true
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if len(scanIDs) > 1 || (len(scanIDs) == 1 && !scanIDs[harnessTargetScanID]) {
		t.Fatalf("packet input not scan-scoped: scans=%v", scanIDs)
	}
	if len(findings) == 0 {
		t.Skipf("no instances recorded for target scan %q", harnessTargetScanID)
	}
	cfg := openclaw.DefaultConfig()
	cfg.MaxTokensPerScan = 4000
	cfg.MaxFindingsPerScan = 12
	cfg.CAH = openclaw.DefaultCAHConfig()
	cfg.CAH.MaxCandidates = 12
	cfg.CAH.TokenBudgetPerScan = 3000
	pkt, err := openclaw.BuildPacket(openclaw.PacketInput{
		ScanID: harnessTargetScanID, Repository: store.Repository{ID: harnessTargetRepoID, FullName: "***/Wifi_Collector"},
		ScanType: openclaw.ScanTypeRepo, Findings: findings, Instances: instances,
	}, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if pkt.ScanID != harnessTargetScanID {
		t.Fatalf("packet ScanID=%q, want %q", pkt.ScanID, harnessTargetScanID)
	}
	if pkt.RepoID != harnessTargetRepoID {
		t.Fatalf("packet RepoID=%d, want %d", pkt.RepoID, harnessTargetRepoID)
	}
	sources := map[string]int{}
	emptyEv := 0
	protected := 0
	for _, f := range pkt.Findings {
		sources[f.Source]++
		if f.EvidenceRedacted == "" {
			emptyEv++
		}
		if f.ProtectedFromDowngrade {
			protected++
		}
	}
	t.Logf("OLD wifi review was 10/10 tech_debt empty evidence")
	t.Logf("NEW harness=%s scan=%s n=%d empty_evidence=%d protected=%d sources=%v",
		pkt.HarnessVersion, harnessTargetScanID, len(pkt.Findings), emptyEv, protected, sources)
	for i, f := range pkt.Findings {
		t.Logf("%d. [%s] %s/%s sev=%s unc=%.2f prot=%v path=%s", i+1, f.CoachLane, f.Source, f.Category, f.Severity, f.UncertaintyScore, f.ProtectedFromDowngrade, f.Path)
	}
	if sources["tech_debt"] == len(pkt.Findings) && len(pkt.Findings) > 0 {
		t.Fatal("packet still entirely tech_debt")
	}
	if protected == 0 {
		t.Fatal("expected some protected coach findings")
	}
	fmt.Fprintf(os.Stderr, "dry-run ok: %d findings sources=%v\n", len(pkt.Findings), sources)
}

// TestHarnessScanScopingRegression proves that scan-scoped harness packet
// loading (the store APIs used by the live harness) cannot be contaminated by
// findings/instances belonging to another scan, or by open-status findings
// that were never observed in the target scan.
func TestHarnessScanScopingRegression(t *testing.T) {
	ctx := context.Background()
	s, err := store.OpenSQLite(filepath.Join(t.TempDir(), "harness-scope.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	now := time.Now().UTC()
	repo, err := s.UpsertRepository(ctx, store.Repository{
		ForgeType: store.ForgeTypeGitea, Owner: "commstech", Name: "Wifi_Collector",
		FullName: "commstech/Wifi_Collector",
	})
	if err != nil {
		t.Fatal(err)
	}
	// Sanity: the live harness pins repository_id 10; the in-memory DB assigns
	// its own id, so record the mapping and use it consistently below.
	t.Logf("in-memory Wifi_Collector repository_id=%d (live dry-run target: %d)", repo.ID, harnessTargetRepoID)

	targetScan := harnessTargetScanID
	otherScan := "aaaaaaaaaaaaaaaa"
	for _, scanID := range []string{targetScan, otherScan} {
		if _, err := s.CreateScan(ctx, store.Scan{ID: scanID, RepositoryID: repo.ID, TriggerType: "web", Status: store.ScanStatusCompleted, StartedAt: now}); err != nil {
			t.Fatal(err)
		}
	}

	issues := []ai.CodeIssue{
		{Severity: "high", Category: "secret", Title: "Possible API key", Description: "KEY=***REDACTED*** in example.env",
			File: "config/example.env", Confidence: 0.99, Source: "gitleaks", RuleID: "generic-api-key", Fingerprint: "fp-target-secret"},
		{Severity: "medium", Category: "security", Title: "JWT hardcoded", Description: "jwt.encode(payload, key='hardcoded')",
			File: "app/auth.py", LineNumber: 42, Confidence: 0.7, Source: "semgrep", RuleID: "jwt-hardcode", Fingerprint: "fp-target-semgrep"},
		{Severity: "low", Category: "tech_debt", Title: "Technical debt marker found in code", Description: "TODO without owner",
			File: "wifi_collector.py", LineNumber: 100, Confidence: 0.95, Source: "tech_debt", RuleID: "HEALTH-TECH-MARKER", Fingerprint: "fp-target-debt"},
		// These three must NEVER reach a target-scan packet:
		{Severity: "critical", Category: "secret", Title: "OTHER SCAN ONLY secret", Description: "KEY=***REDACTED***",
			File: "leak.py", Confidence: 0.99, Source: "gitleaks", RuleID: "generic-api-key", Fingerprint: "fp-other-scan"},
		{Severity: "high", Category: "dependency", Title: "OTHER SCAN ONLY CVE", Description: "GHSA-x affects lib < 1.2.3",
			PackageName: "lib", Confidence: 0.9, Source: "grype", RuleID: "GHSA-x", Fingerprint: "fp-other-cve"},
		{Severity: "medium", Category: "tech_debt", Title: "OPEN STATUS ONLY finding", Description: "old unresolved marker",
			File: "legacy.py", Confidence: 0.9, Source: "tech_debt", RuleID: "HEALTH-TECH-MARKER", Fingerprint: "fp-open-status"},
	}

	// Persist scan-scoped rows for both scans.
	if _, _, err := s.PersistScanFindingsBatch(ctx, repo.ID, targetScan, issues[:3], now); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.PersistScanFindingsBatch(ctx, repo.ID, otherScan, issues[3:5], now); err != nil {
		t.Fatal(err)
	}
	// The "open status" finding exists for the repo (status open via default
	// lifecycle mapping) but has no instance in the target scan.
	if _, err := s.UpsertFinding(ctx, store.Finding{
		RepositoryID: repo.ID, Fingerprint: "fp-open-status", Category: "tech_debt",
		Severity: "medium", Confidence: 0.9, Source: "tech_debt", RuleID: "HEALTH-TECH-MARKER",
		FilePath: "legacy.py", Title: "OPEN STATUS ONLY finding", Status: store.FindingStatusOpen,
		FirstSeenAt: now, LastSeenAt: now,
	}); err != nil {
		t.Fatal(err)
	}

	// Harness loading path: exactly what the scan-scoped harness uses.
	harnessFindings, err := s.ListFindingsForScan(ctx, targetScan, 500)
	if err != nil {
		t.Fatal(err)
	}
	instances, err := s.ListFindingInstancesByScan(ctx, targetScan)
	if err != nil {
		t.Fatal(err)
	}

	byFingerprint := map[string]bool{}
	for _, f := range harnessFindings {
		byFingerprint[f.Fingerprint] = true
		if f.RepositoryID != repo.ID {
			t.Fatalf("contamination: finding %+v belongs to repo %d", f.Fingerprint, f.RepositoryID)
		}
	}
	forbidden := map[string]string{
		"fp-other-scan":  "finding from another scan reached target packet input",
		"fp-other-cve":   "finding from another scan reached target packet input",
		"fp-open-status": "open-status-only finding (no instance in target scan) reached target packet input",
	}
	for fp, why := range forbidden {
		if byFingerprint[fp] {
			t.Fatalf("scan scoping violated: %s (%s)", fp, why)
		}
	}
	for _, want := range []string{"fp-target-secret", "fp-target-semgrep", "fp-target-debt"} {
		if !byFingerprint[want] {
			t.Fatalf("target scan lost expected finding %s; got %v", want, byFingerprint)
		}
	}
	for fid, inst := range instances {
		if inst.ScanID != targetScan {
			t.Fatalf("contamination: instance for finding %d has scan_id=%q", fid, inst.ScanID)
		}
	}
	if len(byFingerprint) != 3 {
		t.Fatalf("expected exactly 3 target-scan findings, got %d: %v", len(byFingerprint), byFingerprint)
	}

	cfg := openclaw.DefaultConfig()
	cfg.MaxTokensPerScan = 4000
	cfg.MaxFindingsPerScan = 12
	cfg.CAH = openclaw.DefaultCAHConfig()
	cfg.CAH.MaxCandidates = 12
	cfg.CAH.TokenBudgetPerScan = 3000
	pkt, err := openclaw.BuildPacket(openclaw.PacketInput{
		ScanID: targetScan, Repository: store.Repository{ID: repo.ID, FullName: repo.FullName},
		ScanType: openclaw.ScanTypeRepo, Findings: harnessFindings, Instances: instances,
	}, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if pkt.ScanID != targetScan {
		t.Fatalf("packet ScanID=%q, want %q", pkt.ScanID, targetScan)
	}
	for _, f := range pkt.Findings {
		switch f.Fingerprint {
		case "fp-other-scan", "fp-other-cve", "fp-open-status":
			t.Fatalf("contaminated packet contains %s", f.Fingerprint)
		}
	}
	foundProtected := false
	for _, f := range pkt.Findings {
		if f.Fingerprint == "fp-target-secret" && f.ProtectedFromDowngrade {
			foundProtected = true
		}
		if f.Fingerprint == "fp-target-debt" && f.ProtectedFromDowngrade {
			t.Fatalf("tech_debt finding must not be protected-from-downgrade: %+v", f)
		}
	}
	if !foundProtected {
		t.Fatal("target secret finding lost protected-from-downgrade prioritization in packet")
	}
	t.Logf("scan-scoped packet ok: n=%d, no cross-scan/open-status contamination", len(pkt.Findings))
}
