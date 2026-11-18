package issues

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"git.commsnet.org/commstech/repository-detective/ai"
	"git.commsnet.org/commstech/repository-detective/gitea"
	"github.com/sirupsen/logrus"
)

// fakeClaimStore is an in-memory durable claim store shared by independent
// Manager "processes" in tests. It mirrors the production SQLite semantics:
// a claim slot is exclusive per (repo, forge, fingerprint); expired claimed
// rows can be taken over; filed rows carry the recovery issue number.
type fakeClaimStore struct {
	mu    sync.Mutex
	rows  map[string]*fakeClaimRow
	clock func() time.Time
}

type fakeClaimRow struct {
	holder      string
	state       string
	issueNumber int
	issueURL    string
	expiresAt   time.Time
}

func newFakeClaimStore() *fakeClaimStore {
	return &fakeClaimStore{rows: map[string]*fakeClaimRow{}, clock: time.Now}
}

func claimKey(repositoryID int64, forgeType, fingerprint string) string {
	return fmt.Sprintf("%d/%s/%s", repositoryID, forgeType, fingerprint)
}

func (f *fakeClaimStore) AcquireIssueFilingClaim(_ context.Context, repositoryID int64, forgeType, fingerprint, holder string, lease time.Duration) (FilingClaim, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	key := claimKey(repositoryID, forgeType, fingerprint)
	now := f.clock()
	row, ok := f.rows[key]
	switch {
	case !ok:
		row = &fakeClaimRow{holder: holder, state: FilingClaimStateClaimed, expiresAt: now.Add(lease)}
		f.rows[key] = row
	case row.holder == holder:
		row.expiresAt = now.Add(lease)
	case row.state == FilingClaimStateFiled:
		// recovery source; never overwritten here
	default:
		if row.state == FilingClaimStateClaimed && !row.expiresAt.After(now) {
			row = &fakeClaimRow{holder: holder, state: FilingClaimStateClaimed, expiresAt: now.Add(lease)}
			f.rows[key] = row
		}
	}
	return FilingClaim{
		Holder: row.holder, State: row.state, IssueNumber: row.issueNumber,
		IssueURL: row.issueURL, ExpiresAt: row.expiresAt,
	}, nil
}

func (f *fakeClaimStore) GetIssueFilingClaim(_ context.Context, repositoryID int64, forgeType, fingerprint string) (FilingClaim, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	row, ok := f.rows[claimKey(repositoryID, forgeType, fingerprint)]
	if !ok {
		return FilingClaim{}, fmt.Errorf("not found")
	}
	return FilingClaim{
		Holder: row.holder, State: row.state, IssueNumber: row.issueNumber,
		IssueURL: row.issueURL, ExpiresAt: row.expiresAt,
	}, nil
}

func (f *fakeClaimStore) MarkIssueFilingClaimFiled(_ context.Context, repositoryID int64, forgeType, fingerprint, holder string, issueNumber int, issueURL string, lease time.Duration) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	row, ok := f.rows[claimKey(repositoryID, forgeType, fingerprint)]
	if !ok || row.holder != holder {
		return fmt.Errorf("claim not held")
	}
	row.state = FilingClaimStateFiled
	row.issueNumber = issueNumber
	row.issueURL = issueURL
	row.expiresAt = f.clock().Add(lease)
	return nil
}

func (f *fakeClaimStore) ReleaseIssueFilingClaim(_ context.Context, repositoryID int64, forgeType, fingerprint, holder string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	key := claimKey(repositoryID, forgeType, fingerprint)
	row, ok := f.rows[key]
	if !ok {
		return false, nil
	}
	if row.holder == holder || row.state == FilingClaimStateFiled {
		delete(f.rows, key)
		return true, nil
	}
	return false, nil
}

// forgeFingerprintServer returns a test forge that records exact create calls.
func forgeFingerprintServer(t *testing.T, creates *atomic.Int32) *httptest.Server {
	t.Helper()
	var mu sync.Mutex
	var issueList []gitea.Issue
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		switch {
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/labels"):
			_, _ = w.Write([]byte("[]"))
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/issues"):
			_ = json.NewEncoder(w).Encode(issueList)
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/issues"):
			creates.Add(1)
			n := len(issueList) + 1
			iss := gitea.Issue{
				Number: n, Title: "dup",
				HTMLURL: fmt.Sprintf("https://git.example.com/owner/repo/issues/%d", n),
				Body:    "## Tracking\n\n- Repository Detective fingerprint: rd-cross\n",
			}
			issueList = append(issueList, iss)
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(iss)
		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/labels"):
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("[]"))
		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/comments"):
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":1}`))
		default:
			http.Error(w, "unexpected "+r.Method+" "+r.URL.Path, http.StatusNotFound)
		}
	}))
}

func crossProcessIssue(fingerprint string) ai.CodeIssue {
	return ai.CodeIssue{
		Title: "Cross-process finding", Description: "test", Severity: "high",
		Category: "secret", Source: "gitleaks", RuleID: "generic-api-key",
		File: "a.env", LineNumber: 1, Confidence: 0.95,
	}
}

func testManagerConfig() *Config {
	cfg := GetDefaultConfig()
	cfg.GroupSimilarIssues = false // isolate fingerprint filing from summary issues
	return cfg
}

// Two independent Manager instances (separate processes in production) racing
// to file the same fingerprint must converge on exactly one forge issue: the
// durable claim serializes filing, the loser converges on the winner's issue
// via remote search or the filed claim row.
func TestIndependentManagersConvergeOnOneIssue(t *testing.T) {
	var creates atomic.Int32
	server := forgeFingerprintServer(t, &creates)
	defer server.Close()

	store := newFakeClaimStore()
	client := gitea.NewClient(server.URL, "token", logrus.New())

	// Shorten the convergence poll so the test stays fast; preserve semantics.
	origWindow, origInterval := filingPollWindow, filingPollInterval
	filingPollWindow, filingPollInterval = 2*time.Second, 20*time.Millisecond
	t.Cleanup(func() { filingPollWindow, filingPollInterval = origWindow, origInterval })

	buildManager := func() *Manager {
		m := NewManager(client, nil, testManagerConfig(), logrus.New())
		m.SetFilingClaimStore(store)
		return m
	}

	const instances = 4
	var wg sync.WaitGroup
	for i := 0; i < instances; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			issue := crossProcessIssue("rd-cross")
			EnrichIssue("owner/repo", &issue, "scan-cross")
			issue.Fingerprint = "rd-cross"
			_, _ = buildManager().CreateIssuesFromAnalysis(context.Background(), &IssueCreationRequest{
				Owner: "owner", Repository: "repo", RepositoryID: 1, ScanID: "scan-cross",
				AnalysisResult: &ai.CodeAnalysisResult{Issues: []ai.CodeIssue{issue}},
			})
		}()
	}
	wg.Wait()

	if got := creates.Load(); got != 1 {
		t.Fatalf("expected exactly 1 forge create across %d independent managers, got %d", instances, got)
	}
}

// Recovery: a manager that created the remote issue but died before linking it
// locally leaves a filed claim. The next filing attempt for the same
// fingerprint must adopt that remote issue — not create a duplicate.
func TestFiledClaimRecoversRemoteCreatedIssueWithoutDuplicate(t *testing.T) {
	var creates atomic.Int32
	server := forgeFingerprintServer(t, &creates)
	defer server.Close()

	store := newFakeClaimStore()
	client := gitea.NewClient(server.URL, "token", logrus.New())

	// Simulate the crashed predecessor: it acquired the claim, created the
	// remote issue (#7), recorded the filed recovery row, then died before
	// linking the issue locally.
	ctx := context.Background()
	if _, err := store.AcquireIssueFilingClaim(ctx, 1, "gitea", "rd-cross", "crashed/1/abcd", time.Hour); err != nil {
		t.Fatalf("seed claim: %v", err)
	}
	if err := store.MarkIssueFilingClaimFiled(ctx, 1, "gitea", "rd-cross", "crashed/1/abcd", 7, "https://git.example.com/owner/repo/issues/7", time.Hour); err != nil {
		t.Fatalf("seed filed claim: %v", err)
	}

	m := NewManager(client, nil, testManagerConfig(), logrus.New())
	m.SetFilingClaimStore(store)
	issue := crossProcessIssue("rd-cross")
	EnrichIssue("owner/repo", &issue, "scan-cross")
	issue.Fingerprint = "rd-cross"
	_, err := m.CreateIssuesFromAnalysis(ctx, &IssueCreationRequest{
		Owner: "owner", Repository: "repo", RepositoryID: 1, ScanID: "scan-cross",
		AnalysisResult: &ai.CodeAnalysisResult{Issues: []ai.CodeIssue{issue}},
	})
	if err != nil {
		t.Fatalf("CreateIssuesFromAnalysis: %v", err)
	}
	if got := creates.Load(); got != 0 {
		t.Fatalf("recovery must not create a duplicate, got %d creates", got)
	}
}
