package gitea_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"git.commsnet.org/commstech/repository-detective/gitea"
	"github.com/sirupsen/logrus"
)

func TestResolveTipCommitFromBranchAPI(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/repos/o/r/branches/main" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"name":"main","commit":{"id":"0123456789abcdef0123456789abcdef01234567"}}`))
	}))
	t.Cleanup(srv.Close)

	client := gitea.NewClient(srv.URL, "t", logrus.New())
	got, err := client.ResolveTipCommit(context.Background(), "o", "r", "main")
	if err != nil {
		t.Fatal(err)
	}
	if got != "0123456789abcdef0123456789abcdef01234567" {
		t.Fatalf("got %q", got)
	}
}

func TestResolveTipCommitPassthroughSHA(t *testing.T) {
	client := gitea.NewClient("http://example.invalid", "t", logrus.New())
	sha := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	got, err := client.ResolveTipCommit(context.Background(), "o", "r", sha)
	if err != nil {
		t.Fatal(err)
	}
	if got != sha {
		t.Fatalf("got %q", got)
	}
}
