package gitea

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreatePullRequestPayload(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method %s", r.Method)
		}
		var req CreatePullRequestRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatal(err)
		}
		if req.Head == "repository-detective/fix/abc" && req.Base == "main" {
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(PullRequest{Number: 42, HTMLURL: "https://git.example.com/pr/42"})
			return
		}
		http.Error(w, "bad request", http.StatusBadRequest)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "token", nil)
	pr, err := c.CreatePullRequest(context.Background(), "o", "r", &CreatePullRequestRequest{
		Title: "Repository Detective remediation",
		Body:  "test",
		Head:  "repository-detective/fix/abc",
		Base:  "main",
	})
	if err != nil {
		t.Fatal(err)
	}
	if pr.Number != 42 {
		t.Fatalf("unexpected pr number %d", pr.Number)
	}
}
