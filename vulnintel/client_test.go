package vulnintel

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func TestExtractCVE(t *testing.T) {
	if got := ExtractCVE("TRIVY-CVE-2026-12345 foo"); got != "CVE-2026-12345" {
		t.Fatalf("got %q", got)
	}
	if got := ExtractCVE("garbage"); got != "" {
		t.Fatalf("unexpected %q", got)
	}
}

func TestLookupAndCache(t *testing.T) {
	var kevCalls, epssCalls atomic.Int32
	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/kev") {
			kevCalls.Add(1)
			_, _ = w.Write([]byte(`{"vulnerabilities":[{"cveID":"CVE-2026-12345","dateAdded":"2026-11-01","dueDate":"2026-11-22"}]}`))
			return
		}
		epssCalls.Add(1)
		if r.URL.Query().Get("cve") != "CVE-2026-12345" {
			t.Errorf("unexpected cve parameter")
		}
		_, _ = w.Write([]byte(`{"data":[{"cve":"CVE-2026-12345","epss":"0.72","percentile":"0.97"}]}`))
	}))
	defer svr.Close()
	client := NewClientWithSources(svr.Client(), svr.URL+"/kev", svr.URL+"/epss")
	for i := 0; i < 2; i++ {
		result, err := client.Lookup(context.Background(), "CVE-2026-12345")
		if err != nil {
			t.Fatal(err)
		}
		if !result.KEVKnown || !result.KEVChecked || result.EPSS == nil || *result.EPSS != .72 {
			t.Fatalf("unexpected result %+v", result)
		}
	}
	if kevCalls.Load() != 1 || epssCalls.Load() != 1 {
		t.Fatalf("cache missed: kev %d epss %d", kevCalls.Load(), epssCalls.Load())
	}
}

func TestPartialFeedFailureIsNotLowRisk(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Error(w, "unavailable", 502) }))
	defer server.Close()
	c := NewClientWithSources(server.Client(), server.URL+"/kev", server.URL+"/epss")
	got, err := c.Lookup(context.Background(), "CVE-2026-4321")
	if err != nil {
		t.Fatal(err)
	}
	if got.KEVChecked || got.EPSSChecked || !strings.Contains(got.Priority, "Unknown") {
		t.Fatalf("fail open: %+v", got)
	}
	if _, err := c.Lookup(context.Background(), "https://evil.example/CVE-2026-4321"); err == nil {
		t.Fatal("accepted non-CVE identifier")
	}
}
