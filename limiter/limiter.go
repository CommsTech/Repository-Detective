package limiter

import "context"

// ConcurrencyLimiter bounds how many analyses may run at once.
type ConcurrencyLimiter struct {
	sem chan struct{}
}

// New creates a limiter allowing at most maxConcurrent simultaneous jobs.
// A max of zero or less means unlimited.
func New(maxConcurrent int) *ConcurrencyLimiter {
	if maxConcurrent <= 0 {
		return &ConcurrencyLimiter{}
	}
	return &ConcurrencyLimiter{sem: make(chan struct{}, maxConcurrent)}
}

// Run acquires a slot, runs fn, then releases the slot.
func (l *ConcurrencyLimiter) Run(ctx context.Context, fn func()) error {
	if l == nil || l.sem == nil {
		fn()
		return nil
	}

	select {
	case l.sem <- struct{}{}:
		defer func() { <-l.sem }()
		fn()
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// HasCapacity reports whether a slot can be acquired without blocking.
func (l *ConcurrencyLimiter) HasCapacity() bool {
	if l == nil || l.sem == nil {
		return true
	}
	select {
	case l.sem <- struct{}{}:
		<-l.sem
		return true
	default:
		return false
	}
}
