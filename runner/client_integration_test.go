package runner_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"git.commsnet.org/commstech/repository-detective/api"
	"git.commsnet.org/commstech/repository-detective/graph"
	"git.commsnet.org/commstech/repository-detective/runner"
	"git.commsnet.org/commstech/repository-detective/store"
	"github.com/gin-gonic/gin"
)

func TestClientSubmitLargeResultHMAC(t *testing.T) {
	gin.SetMode(gin.TestMode)
	secret := "integration-runner-secret"
	dir := t.TempDir()
	s, err := store.Open(store.Config{Enabled: true, Path: filepath.Join(dir, "runner.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	ctx := context.Background()
	repo, _ := s.UpsertRepository(ctx, store.Repository{Owner: "o", Name: "r", FullName: "o/r", CloneURL: "https://example.com/o/r.git"})
	job, _ := s.CreateRunnerJob(ctx, store.RunnerJob{
		JobID: "rj-integ01", RepositoryID: repo.ID, ScanID: "scan1", JobType: runner.JobTypeGraph,
		Status: store.RunnerJobStatusQueued, RunnerMode: runner.ModeNative,
		JobSpecJSON: []byte(`{}`), ResultSummaryJSON: []byte(`{}`),
		CreatedAt: time.Now().UTC(),
	})
	_, _ = s.ClaimNextRunnerJob(ctx, time.Now().UTC())

	cfg := runner.Config{SharedSecret: secret}
	recv := runner.NewReceiver(s, cfg, nil, nil)
	h := api.NewRunnerHandler(s, cfg, recv, nil, runner.NewRegistry(), nil)
	r := gin.New()
	g := r.Group("/api/v1/runner")
	g.Use(func(c *gin.Context) {
		if err := recv.CheckNonce(c.Request.Context(), c.GetHeader(runner.HeaderNonce)); err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "runner nonce rejected"})
			return
		}
		c.Next()
	})
	g.Use(api.RequireRunnerHMAC(secret))
	h.RegisterRunnerRoutes(g)
	srv := httptest.NewServer(r)
	defer srv.Close()

	result := runner.JobResult{
		Version: 1, JobID: job.JobID, ScanID: "scan1", Status: runner.JobStatusCompleted,
		StartedAt: time.Now().UTC(), FinishedAt: time.Now().UTC(), FilesAnalyzed: 10,
		Graph: &graph.Graph{},
	}
	body, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	client := runner.NewClient(srv.URL, secret)
	if err := client.SubmitResultBody(ctx, job.JobID, body); err != nil {
		t.Fatalf("submit large result: %v", err)
	}
	got, err := s.GetRunnerJob(ctx, job.JobID)
	if err != nil || got.Status != store.RunnerJobStatusCompleted {
		t.Fatalf("job status: %v %q", err, got.Status)
	}
	_ = http.MethodPost
}
