package limiter

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestConcurrencyLimiterBoundsParallelism(t *testing.T) {
	l := New(2)
	ctx := context.Background()

	var running int32
	var maxRunning int32

	done := make(chan struct{}, 4)
	for i := 0; i < 4; i++ {
		go func() {
			_ = l.Run(ctx, func() {
				current := atomic.AddInt32(&running, 1)
				for {
					prev := atomic.LoadInt32(&maxRunning)
					if current <= prev || atomic.CompareAndSwapInt32(&maxRunning, prev, current) {
						break
					}
				}
				time.Sleep(50 * time.Millisecond)
				atomic.AddInt32(&running, -1)
			})
			done <- struct{}{}
		}()
	}

	for i := 0; i < 4; i++ {
		<-done
	}

	if maxRunning > 2 {
		t.Fatalf("expected at most 2 concurrent jobs, saw %d", maxRunning)
	}
}

func TestConcurrencyLimiterUnlimitedWhenZero(t *testing.T) {
	l := New(0)
	called := false
	if err := l.Run(context.Background(), func() { called = true }); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Fatal("expected fn to run")
	}
}
