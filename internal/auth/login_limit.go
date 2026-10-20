package auth

import (
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// LoginLimiter rate-limits auth form submissions (login + bootstrap) per client key.
// It is intentionally separate from webhook limiting and does not affect API token auth.
type LoginLimiter struct {
	mu       sync.Mutex
	limiters map[string]*rate.Limiter
	limit    rate.Limit
	burst    int
	maxKeys  int
}

// NewLoginLimiter builds a per-key limiter. Typical values: 1 req/s sustained, burst 5.
func NewLoginLimiter(perSecond float64, burst, maxKeys int) *LoginLimiter {
	if perSecond <= 0 {
		perSecond = 1
	}
	if burst <= 0 {
		burst = 5
	}
	if maxKeys <= 0 {
		maxKeys = 4096
	}
	return &LoginLimiter{
		limiters: make(map[string]*rate.Limiter),
		limit:    rate.Limit(perSecond),
		burst:    burst,
		maxKeys:  maxKeys,
	}
}

// Allow reports whether the client key may proceed.
func (l *LoginLimiter) Allow(key string) bool {
	if l == nil {
		return true
	}
	if key == "" {
		key = "unknown"
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if len(l.limiters) >= l.maxKeys {
		// Drop arbitrary half of entries to bound memory under abuse.
		i := 0
		for k := range l.limiters {
			delete(l.limiters, k)
			i++
			if i >= l.maxKeys/2 {
				break
			}
		}
	}
	lim, ok := l.limiters[key]
	if !ok {
		lim = rate.NewLimiter(l.limit, l.burst)
		l.limiters[key] = lim
	}
	return lim.Allow()
}

// RetryAfter is a conservative hint for clients (not a hard schedule).
func (l *LoginLimiter) RetryAfter() time.Duration {
	return 2 * time.Second
}
