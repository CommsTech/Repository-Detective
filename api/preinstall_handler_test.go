package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"git.commsnet.org/commstech/repository-detective/api"
	"git.commsnet.org/commstech/repository-detective/preinstall"
	"git.commsnet.org/commstech/repository-detective/scanners"
	"git.commsnet.org/commstech/repository-detective/store"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func TestPreinstallAPICreateAndGetAudit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	origLookup := preinstall.LookupHostIPsForTests()
	preinstall.SetLookupHostIPsForTests(func(host string) ([]net.IP, error) {
		return []net.IP{net.ParseIP("140.82.121.4")}, nil
	})
	t.Cleanup(func() { preinstall.SetLookupHostIPsForTests(origLookup) })

	s, err := store.OpenSQLite(t.TempDir() + "/api.db")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	runner := preinstall.NewRunner(s, preinstall.Config{Enabled: true, AllowGitClone: false}, scanners.DefaultConfig(), logrus.New())
	h := api.NewPreinstallHandler(s, runner, logrus.New())

	r := gin.New()
	g := r.Group("/api/v1")
	h.RegisterRoutes(g)

	body, _ := json.Marshal(map[string]string{
		"repo_url":    "https://github.com/example/repo",
		"audit_depth": "quick",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/preinstall/audits", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusAccepted {
		t.Fatalf("create status=%d body=%s", w.Code, w.Body.String())
	}

	var created map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	auditID := created["audit_id"]
	if auditID == "" {
		t.Fatal("missing audit_id")
	}

	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/preinstall/audits/"+auditID, nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("get status=%d", w2.Code)
	}

	// audit should exist in queued/running/completed/failed state
	got, err := s.GetAuditRequest(context.Background(), auditID)
	if err != nil {
		t.Fatal(err)
	}
	if got.NormalizedRepoURL != "https://github.com/example/repo" {
		t.Fatalf("unexpected repo: %s", got.NormalizedRepoURL)
	}
}

func TestPreinstallAPIRejectsInvalidURL(t *testing.T) {
	gin.SetMode(gin.TestMode)
	s, err := store.OpenSQLite(t.TempDir() + "/api2.db")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	runner := preinstall.NewRunner(s, preinstall.Config{Enabled: true}, scanners.DefaultConfig(), logrus.New())
	h := api.NewPreinstallHandler(s, runner, logrus.New())
	r := gin.New()
	g := r.Group("/api/v1")
	h.RegisterRoutes(g)

	body, _ := json.Marshal(map[string]string{"repo_url": "file:///tmp/x"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/preinstall/audits", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestPreinstallMarkReportReviewed(t *testing.T) {
	gin.SetMode(gin.TestMode)
	s, err := store.OpenSQLite(t.TempDir() + "/api3.db")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx := context.Background()
	_, _ = s.CreateAuditRequest(ctx, store.AuditRequest{AuditID: "a1", RepoURL: "https://github.com/o/r", NormalizedRepoURL: "https://github.com/o/r", Status: store.AuditStatusCompleted, StartedAt: time.Now().UTC()})
	report, err := s.AddDisclosureReport(ctx, store.DisclosureReport{AuditID: "a1", ReportType: store.ReportTypeInstallRiskSummary, Title: "t", BodyMarkdown: "b"})
	if err != nil {
		t.Fatal(err)
	}

	runner := preinstall.NewRunner(s, preinstall.Config{Enabled: true}, scanners.DefaultConfig(), logrus.New())
	h := api.NewPreinstallHandler(s, runner, logrus.New())
	r := gin.New()
	g := r.Group("/api/v1")
	h.RegisterRoutes(g)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/preinstall/reports/"+strconv.FormatInt(report.ID, 10)+"/mark-reviewed", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	got, _ := s.GetDisclosureReport(ctx, report.ID)
	if !got.ApprovedByUser {
		t.Fatal("expected approved_by_user")
	}
}
