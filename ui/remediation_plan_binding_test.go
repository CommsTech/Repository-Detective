package ui_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"git.commsnet.org/commstech/repository-detective/internal/security"
	"git.commsnet.org/commstech/repository-detective/patcher"
	"git.commsnet.org/commstech/repository-detective/remediation"
	"git.commsnet.org/commstech/repository-detective/store"
)

type planActionProbe struct{ approvals, rejections, attempts int }

func (p *planActionProbe) GeneratePlan(context.Context, int64) (remediation.Plan, error) {
	return remediation.Plan{}, nil
}
func (p *planActionProbe) ApprovePlan(context.Context, string) error { p.approvals++; return nil }
func (p *planActionProbe) RejectPlan(context.Context, string) error  { p.rejections++; return nil }
func (p *planActionProbe) CheckPREligibility(context.Context, string) (patcher.EligibilityResult, error) {
	return patcher.EligibilityResult{}, nil
}
func (p *planActionProbe) AttemptPR(context.Context, string) (patcher.PatchAttempt, error) {
	p.attempts++
	return patcher.PatchAttempt{}, nil
}
func (p *planActionProbe) ListPatchAttempts(context.Context, string) ([]patcher.PatchAttempt, error) {
	return nil, nil
}

func TestRemediationActionsCannotCrossFindings(t *testing.T) {
	s, err := store.Open(store.Config{Enabled: true, Path: filepath.Join(t.TempDir(), "review-binding.db")})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	ctx := context.Background()
	repo, err := s.UpsertRepository(ctx, store.Repository{Owner: "o", Name: "r", FullName: "o/r"})
	if err != nil {
		t.Fatal(err)
	}
	first, err := s.UpsertFinding(ctx, store.Finding{RepositoryID: repo.ID, Fingerprint: "one", Title: "one", Severity: "low"})
	if err != nil {
		t.Fatal(err)
	}
	second, err := s.UpsertFinding(ctx, store.Finding{RepositoryID: repo.ID, Fingerprint: "two", Title: "two", Severity: "low"})
	if err != nil {
		t.Fatal(err)
	}
	plan, err := s.SaveRemediationPlan(ctx, store.RemediationPlanFromDomain(remediation.Plan{ID: "review-second", FindingID: second.ID, RepositoryID: repo.ID, Status: remediation.StatusProposed}))
	if err != nil {
		t.Fatal(err)
	}
	r, h := testUI(t, s)
	probe := &planActionProbe{}
	h.SetRemediationBackend(true, probe)
	h.SetRemediationPRBackend(true, probe)
	for _, action := range []string{"approve", "reject", "attempt-pr"} {
		form := url.Values{"plan_id": {plan.PlanID}, "csrf_token": {security.CSRFToken("test-secret", "review-test-key")}}
		path := "/ui/findings/" + strconv.FormatInt(first.ID, 10) + "/remediation/" + action
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("X-Repository-Detective-API-Key", "review-test-key")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("%s should reject cross-finding plan: code %d body %s", action, w.Code, w.Body.String())
		}
	}
	if probe.approvals != 0 || probe.rejections != 0 || probe.attempts != 0 {
		t.Fatalf("cross-finding backend action executed: %+v", probe)
	}
}
