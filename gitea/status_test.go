package gitea

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestCreateCommitStatusPayload(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody CommitStatus
	var gotAuth string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		if err := json.Unmarshal(body, &gotBody); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	client := NewClient(server.URL, "secret-token", logrus.New())
	err := client.CreateCommitStatus(context.Background(), "owner", "repo", "abc1234", &CommitStatus{
		State:       CommitStatePending,
		TargetURL:   "https://repository-detective.example.com",
		Description: "Repository-Detective scan started",
		Context:     "repository-detective/security-scan",
	})
	if err != nil {
		t.Fatalf("CreateCommitStatus: %v", err)
	}

	if gotMethod != http.MethodPost {
		t.Fatalf("expected POST, got %s", gotMethod)
	}
	if gotPath != "/api/v1/repos/owner/repo/statuses/abc1234" {
		t.Fatalf("unexpected path %s", gotPath)
	}
	if gotAuth != "token secret-token" {
		t.Fatalf("unexpected auth header %q", gotAuth)
	}
	if gotBody.State != CommitStatePending {
		t.Fatalf("expected pending state, got %q", gotBody.State)
	}
	if gotBody.Context != "repository-detective/security-scan" {
		t.Fatalf("unexpected context %q", gotBody.Context)
	}
}

func TestMapGiteaCommitStateWarning(t *testing.T) {
	if got := MapGiteaCommitState(CommitStateWarning); got != CommitStateFailure {
		t.Fatalf("expected warning mapped to failure, got %q", got)
	}
}

func TestStatusReporterPending(t *testing.T) {
	var posted CommitStatus
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &posted)
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	reporter := NewStatusReporter(
		NewClient(server.URL, "token", logrus.New()),
		true,
		ChecksConfig{Context: "repository-detective/security-scan", TargetURL: "https://repository-detective.example.com"},
		logrus.New(),
	)
	reporter.ReportPending(context.Background(), "owner", "repo", "abc1234567890")

	if posted.State != CommitStatePending {
		t.Fatalf("expected pending, got %q", posted.State)
	}
	if posted.Description != "Repository-Detective scan started" {
		t.Fatalf("unexpected description %q", posted.Description)
	}
}

func TestStatusReporterSkipsWithoutSHA(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	logger := logrus.New()
	reporter := NewStatusReporter(
		NewClient(server.URL, "token", logger),
		true,
		ChecksConfig{Context: "repository-detective/security-scan"},
		logger,
	)
	reporter.ReportPending(context.Background(), "owner", "repo", "main")
	if called {
		t.Fatal("expected no status post without commit SHA")
	}
}

func TestStatusReporterNonFatalOnAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer server.Close()

	reporter := NewStatusReporter(
		NewClient(server.URL, "token", logrus.New()),
		true,
		ChecksConfig{Context: "repository-detective/security-scan"},
		logrus.New(),
	)
	reporter.ReportFinal(context.Background(), "owner", "repo", "abc1234567890", nil, nil, false)
}
