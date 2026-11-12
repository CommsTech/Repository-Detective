package closure_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"git.commsnet.org/commstech/repository-detective/closure"
	"git.commsnet.org/commstech/repository-detective/store"
)

func TestRecordDirectRemediationPersistsMergeSHA(t *testing.T) {
	dir := t.TempDir()
	s, err := store.Open(store.Config{Enabled: true, Driver: "sqlite", Path: filepath.Join(dir, "direct.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	ctx := context.Background()
	repo, _ := s.UpsertRepository(ctx, store.Repository{Owner: "o", Name: "r", FullName: "o/r", ConnectedRepo: true})
	finding, _ := s.UpsertFinding(ctx, store.Finding{
		RepositoryID: repo.ID, Fingerprint: "fp-direct", Title: "t", Severity: "medium",
		Source: "checkov", RuleID: "CKV_DOCKER_3", Status: store.FindingStatusOpen,
	})

	eng := &closure.Engine{
		Config: closure.Config{Enabled: true},
		Store:  directRemediationStore{s: s},
	}
	ev, err := eng.RecordDirectRemediation(ctx, closure.DirectRemediationInput{
		FindingID: finding.ID, RepositoryID: repo.ID, Fingerprint: finding.Fingerprint,
		OriginalSource: finding.Source, MergeCommitSHA: "deadbeefcafebabe",
		Reason: "Docker USER fix merged on main",
	})
	if err != nil {
		t.Fatal(err)
	}
	if ev.MergeCommitSHA != "deadbeefcafebabe" {
		t.Fatalf("unexpected merge sha %q", ev.MergeCommitSHA)
	}
	got, err := s.GetLatestClosureEvidenceByFindingID(ctx, finding.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.MergeCommitSHA != "deadbeefcafebabe" {
		t.Fatalf("db merge sha %q", got.MergeCommitSHA)
	}
	if got.OriginalSource != "checkov" {
		t.Fatalf("db original source %q", got.OriginalSource)
	}
	if got.VerificationScanID != "" {
		t.Fatalf("expected empty verification scan id, got %q", got.VerificationScanID)
	}
}

func TestVerifyFindingClosureWithoutPriorEvidence(t *testing.T) {
	dir := t.TempDir()
	s, err := store.Open(store.Config{Enabled: true, Driver: "sqlite", Path: filepath.Join(dir, "verify-direct.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	ctx := context.Background()
	repo, _ := s.UpsertRepository(ctx, store.Repository{Owner: "o", Name: "r", FullName: "o/r", ConnectedRepo: true})
	finding, _ := s.UpsertFinding(ctx, store.Finding{
		RepositoryID: repo.ID, Fingerprint: "fp-absent", Title: "secret", Severity: "high",
		Source: "static", RuleID: "SEC-HARDCODED-SECRET", Status: store.FindingStatusOpen,
	})

	eng := &closure.Engine{
		Config: closure.Config{Enabled: true, RequireScannerSuccess: true},
		Store:  directRemediationStore{s: s},
	}
	scan := closure.ScanContext{
		ScanID: "scan-1", RepositoryID: repo.ID, Owner: "o", Repo: "r",
		FingerprintsSeen: map[string]struct{}{},
		ScannerResults:   map[string]string{"static": "clean"},
	}
	ev, err := eng.VerifyFindingClosure(ctx, finding.ID, scan)
	if err != nil {
		t.Fatal(err)
	}
	if ev.Status != closure.StatusVerified {
		t.Fatalf("expected verified, got %s (%s)", ev.Status, ev.Reason)
	}
	updated, err := s.GetFindingDetail(ctx, finding.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Status != store.FindingStatusResolvedVerified {
		t.Fatalf("finding status %q", updated.Status)
	}
}

type directRemediationStore struct {
	s store.QueryStore
}

func (d directRemediationStore) GetPatchAttemptForClosure(ctx context.Context, attemptID string) (closure.PatchAttemptRow, error) {
	return closure.PatchAttemptRow{}, nil
}

func (d directRemediationStore) ListPatchAttemptsByRepositoryAndStatus(ctx context.Context, repositoryID int64, status string) ([]closure.PatchAttemptRow, error) {
	return nil, nil
}

func (d directRemediationStore) UpdatePatchAttemptMerged(ctx context.Context, attemptID string, mergeSHA string, mergedAt time.Time) error {
	return nil
}

func (d directRemediationStore) GetLatestClosureEvidenceByFindingID(ctx context.Context, findingID int64) (closure.EvidenceRow, error) {
	rec, err := d.s.GetLatestClosureEvidenceByFindingID(ctx, findingID)
	if err != nil {
		return closure.EvidenceRow{}, err
	}
	return closureEvidenceToRow(rec), nil
}

func (d directRemediationStore) SaveClosureEvidence(ctx context.Context, row closure.EvidenceRow) (closure.EvidenceRow, error) {
	rec, err := d.s.SaveClosureEvidence(ctx, closureEvidenceFromRow(row))
	if err != nil {
		return closure.EvidenceRow{}, err
	}
	return closureEvidenceToRow(rec), nil
}

func (d directRemediationStore) UpdateClosureEvidence(ctx context.Context, row closure.EvidenceRow) error {
	return d.s.UpdateClosureEvidence(ctx, closureEvidenceFromRow(row))
}

func (d directRemediationStore) ListClosureEvidenceByRepositoryAndStatus(ctx context.Context, repositoryID int64, status string) ([]closure.EvidenceRow, error) {
	recs, err := d.s.ListClosureEvidenceByRepositoryAndStatus(ctx, repositoryID, status)
	if err != nil {
		return nil, err
	}
	out := make([]closure.EvidenceRow, 0, len(recs))
	for _, rec := range recs {
		out = append(out, closureEvidenceToRow(rec))
	}
	return out, nil
}

func (d directRemediationStore) GetFindingByID(ctx context.Context, findingID int64) (closure.FindingRow, error) {
	detail, err := d.s.GetFindingDetail(ctx, findingID)
	if err != nil {
		return closure.FindingRow{}, err
	}
	return closure.FindingRow{
		ID: detail.ID, RepositoryID: detail.RepositoryID, Fingerprint: detail.Fingerprint, Source: detail.Source, Status: detail.Status,
	}, nil
}

func (d directRemediationStore) UpdateFindingStatus(ctx context.Context, findingID int64, status string) error {
	return d.s.UpdateFindingStatus(ctx, findingID, status)
}

func (d directRemediationStore) AddLifecycleEvent(ctx context.Context, findingID int64, scanID, eventType, message string) error {
	fid := findingID
	return d.s.AddLifecycleEvent(ctx, store.LifecycleEvent{
		FindingID: &fid, ScanID: scanID, EventType: eventType, Message: message,
	})
}

func (d directRemediationStore) ListExternalIssuesByFinding(ctx context.Context, findingID int64) ([]closure.ExternalIssueRow, error) {
	return nil, nil
}

func (d directRemediationStore) GetRepository(ctx context.Context, repositoryID int64) (closure.RepositoryRow, error) {
	repo, err := d.s.GetRepository(ctx, repositoryID)
	if err != nil {
		return closure.RepositoryRow{}, err
	}
	return closure.RepositoryRow{ID: repo.ID, Owner: repo.Owner, Name: repo.Name, FullName: repo.FullName}, nil
}

func closureEvidenceFromRow(row closure.EvidenceRow) store.ClosureEvidenceRecord {
	return store.ClosureEvidenceRecord{
		ID: row.ID, FindingID: row.FindingID, PatchAttemptID: row.PatchAttemptID,
		RepositoryID: row.RepositoryID, Fingerprint: row.Fingerprint,
		MergeCommitSHA: row.MergeCommitSHA, VerificationScanID: row.VerificationScanID,
		OriginalSource: row.OriginalSource, ScannerStatus: row.ScannerStatus,
		FingerprintPresent: row.FingerprintPresent, Status: row.Status, Reason: row.Reason,
		CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
}

func closureEvidenceToRow(rec store.ClosureEvidenceRecord) closure.EvidenceRow {
	return closure.EvidenceRow{
		ID: rec.ID, FindingID: rec.FindingID, PatchAttemptID: rec.PatchAttemptID,
		RepositoryID: rec.RepositoryID, Fingerprint: rec.Fingerprint,
		MergeCommitSHA: rec.MergeCommitSHA, VerificationScanID: rec.VerificationScanID,
		OriginalSource: rec.OriginalSource, ScannerStatus: rec.ScannerStatus,
		FingerprintPresent: rec.FingerprintPresent, Status: rec.Status, Reason: rec.Reason,
		CreatedAt: rec.CreatedAt, UpdatedAt: rec.UpdatedAt,
	}
}
