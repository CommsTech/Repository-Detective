package gitea_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"git.commsnet.org/commstech/repository-detective/gitea"
	"github.com/sirupsen/logrus"
)

func TestDownloadRepositoryArchiveEnforcesMaxSize(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/zipball/") {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, strings.Repeat("a", 2048))
	}))
	defer server.Close()

	client := gitea.NewClient(server.URL, "token", logrus.New())
	_, cleanup, _, err := client.DownloadRepositoryArchive(context.Background(), "owner", "repo", "main", 1024)
	if cleanup != nil {
		cleanup()
	}
	if err == nil {
		t.Fatal("expected size limit error")
	}
	if !strings.Contains(err.Error(), "size limit") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDownloadRepositoryArchiveSuccess(t *testing.T) {
	payload := "PK\x03\x04fake"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/zipball/") {
			http.NotFound(w, r)
			return
		}
		if auth := r.Header.Get("Authorization"); auth != "token token" {
			t.Fatalf("unexpected auth: %s", auth)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, payload)
	}))
	defer server.Close()

	client := gitea.NewClient(server.URL, "token", logrus.New())
	path, cleanup, written, err := client.DownloadRepositoryArchive(context.Background(), "owner", "repo", "abc1234", 1024*1024)
	if err != nil {
		t.Fatalf("download failed: %v", err)
	}
	defer cleanup()

	if written != int64(len(payload)) {
		t.Fatalf("expected %d bytes, got %d", len(payload), written)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != payload {
		t.Fatalf("unexpected payload: %q", string(data))
	}
}
