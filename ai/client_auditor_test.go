package ai_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"git.commsnet.org/commstech/repository-detective/ai"
	"github.com/sirupsen/logrus"
)

type blockingTransport struct {
	started chan struct{}
	release chan struct{}
	active  atomic.Int32
	peak    atomic.Int32
}

func (t *blockingTransport) Name() string { return "blocking" }

func (t *blockingTransport) Complete(ctx context.Context, req ai.ChatRequest) (*ai.ChatResponse, error) {
	n := t.active.Add(1)
	defer t.active.Add(-1)
	for {
		old := t.peak.Load()
		if n <= old || t.peak.CompareAndSwap(old, n) {
			break
		}
	}
	select {
	case t.started <- struct{}{}:
	default:
	}
	select {
	case <-t.release:
		return &ai.ChatResponse{Content: `{"findings":[]}`}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func TestRunAuditorRespectsCancel(t *testing.T) {
	tr := &blockingTransport{
		started: make(chan struct{}, 1),
		release: make(chan struct{}),
	}
	client := ai.NewClientWithTransport(tr, "test", logrus.New())
	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		_, err := client.RunAuditor(ctx, &ai.AuditorRequest{
			AuditorType:        "sql",
			VulnerabilityClass: "sql_injection",
		})
		errCh <- err
	}()
	select {
	case <-tr.started:
	case <-time.After(2 * time.Second):
		t.Fatal("auditor did not start")
	}
	cancel()
	select {
	case err := <-errCh:
		if err == nil {
			t.Fatal("expected cancel error")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("RunAuditor did not return after cancel")
	}
}

func TestRunAuditorSingleFlight(t *testing.T) {
	tr := &blockingTransport{
		started: make(chan struct{}, 2),
		release: make(chan struct{}),
	}
	client := ai.NewClientWithTransport(tr, "test", logrus.New())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = client.RunAuditor(ctx, &ai.AuditorRequest{
				AuditorType:        "sql",
				VulnerabilityClass: "sql_injection",
			})
		}()
	}
	select {
	case <-tr.started:
	case <-time.After(2 * time.Second):
		t.Fatal("first auditor did not start")
	}
	time.Sleep(100 * time.Millisecond)
	if peak := tr.peak.Load(); peak != 1 {
		close(tr.release)
		t.Fatalf("expected single-flight peak=1, got %d", peak)
	}
	close(tr.release)
	wg.Wait()
}
