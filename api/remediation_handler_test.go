package api_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"git.commsnet.org/commstech/repository-detective/api"
	"git.commsnet.org/commstech/repository-detective/remediation"
	"git.commsnet.org/commstech/repository-detective/store"
	"github.com/gin-gonic/gin"
)

type stubRemediation struct{}

func (stubRemediation) GetPlanForFinding(c *gin.Context, findingID int64) (remediation.Plan, error) {
	return remediation.Plan{ID: "rp-1", FindingID: findingID, Status: remediation.StatusProposed}, nil
}

func (stubRemediation) GeneratePlanForFinding(c *gin.Context, findingID int64) (remediation.Plan, error) {
	return remediation.Plan{ID: "rp-new", FindingID: findingID, Status: remediation.StatusProposed, FixStrategy: "fix"}, nil
}

func (stubRemediation) GetPlanByID(c *gin.Context, planID string) (remediation.Plan, error) {
	return remediation.Plan{ID: planID, Status: remediation.StatusProposed}, nil
}

func (stubRemediation) ApprovePlan(c *gin.Context, planID string) error { return nil }
func (stubRemediation) RejectPlan(c *gin.Context, planID string) error  { return nil }

func testRemediationRouter(t *testing.T) *gin.Engine {
	t.Helper()
	dir := t.TempDir()
	s, err := store.Open(store.Config{Enabled: true, Path: filepath.Join(dir, "rem-api.db")})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	ctx := context.Background()
	repo, _ := s.UpsertRepository(ctx, store.Repository{Owner: "o", Name: "r", FullName: "o/r"})
	_, _ = s.UpsertFinding(ctx, store.Finding{RepositoryID: repo.ID, Fingerprint: "x", Title: "t", Severity: "high", Confidence: 0.9})

	gin.SetMode(gin.TestMode)
	r := gin.New()
	g := r.Group("/api/v1")
	api.NewRemediationHandler(s, stubRemediation{}, false).RegisterRoutes(g)
	return r
}

func TestGenerateRemediationPlan(t *testing.T) {
	r := testRemediationRouter(t)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/findings/1/remediation/generate", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status %d body %s", w.Code, w.Body.String())
	}
	if !bytes.Contains(w.Body.Bytes(), []byte("fix")) {
		t.Fatal("expected plan body")
	}
}

func TestApproveRemediationPlan(t *testing.T) {
	r := testRemediationRouter(t)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/remediation/rp-1/approve", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status %d", w.Code)
	}
}
