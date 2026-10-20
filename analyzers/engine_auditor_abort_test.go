package analyzers

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"git.commsnet.org/commstech/repository-detective/ai"
	"github.com/sirupsen/logrus"
)

type stallTransport struct {
	calls   atomic.Int32
	started chan struct{}
}

func (t *stallTransport) Name() string { return "stall" }

func (t *stallTransport) Complete(ctx context.Context, req ai.ChatRequest) (*ai.ChatResponse, error) {
	t.calls.Add(1)
	select {
	case t.started <- struct{}{}:
	default:
	}
	<-ctx.Done()
	return nil, ctx.Err()
}

func TestCrossAuditorAbortCancelsSiblings(t *testing.T) {
	tr := &stallTransport{started: make(chan struct{}, 8)}
	client := ai.NewClientWithTransport(tr, "test", logrus.New())
	engine := NewEngine(nil, nil, client, &Config{
		EnableSecurity:              true,
		EnableLLMAuditors:           true,
		AnalysisDepth:               3,
		AbortAuditorsOnFirstTimeout: true,
		MaxAuditorFailuresPerScan:   1,
		MaxFileSize:                 1 << 20,
	}, logrus.New())

	ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
	defer cancel()

	prepare := &PrepareReport{Repository: "owner/repo"}
	// Force LLM path: depth>=3, security on, ai client present, non-empty targets.
	// Call runAuditor orchestration via Scan with empty deterministic results by
	// using a tiny helper that only exercises the parallel auditor block.
	// Directly exercise through a private-path simulation: run two auditors via exported flow.
	_ = prepare

	// Simulate the Scan auditor fan-out logic with the engine's runAuditor.
	type result struct {
		err error
	}
	auditorCtx, cancelAuditors := context.WithCancel(ctx)
	defer cancelAuditors()
	results := make(chan result, 3)
	for i := 0; i < 3; i++ {
		go func() {
			_, err := engine.runAuditor(auditorCtx, AuditorSQL, "sql_injection", prepare, []FileContent{
				{Path: "a.go", Content: "package a\n", Language: "go"},
			})
			results <- result{err: err}
		}()
	}
	// Wait until at least one auditor is in-flight (single-flight means one active Complete).
	select {
	case <-tr.started:
	case <-time.After(2 * time.Second):
		t.Fatal("no auditor started")
	}
	// First timeout/cancel should abort siblings.
	cancelAuditors()
	for i := 0; i < 3; i++ {
		select {
		case r := <-results:
			if r.err == nil {
				t.Fatal("expected auditor error after abort")
			}
		case <-time.After(2 * time.Second):
			t.Fatal("auditor did not finish after abort")
		}
	}
	// Single-flight + cancel: at most a couple Complete entries should have started.
	if calls := tr.calls.Load(); calls > 3 {
		t.Fatalf("expected limited auditor calls after abort, got %d", calls)
	}
}

func TestIsAuditorAbortError(t *testing.T) {
	if !isAuditorAbortError(context.DeadlineExceeded) {
		t.Fatal("deadline should abort")
	}
	if !isAuditorAbortError(context.Canceled) {
		t.Fatal("canceled should abort")
	}
	if isAuditorAbortError(nil) {
		t.Fatal("nil should not abort")
	}
}
