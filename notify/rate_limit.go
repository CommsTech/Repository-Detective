package notify

import (
	"sync"
	"time"
)

// RateLimiter enforces per-key cooldown between notifications.
type RateLimiter struct {
	mu       sync.Mutex
	lastSent map[string]time.Time
}

// NewRateLimiter creates an empty rate limiter.
func NewRateLimiter() *RateLimiter {
	return &RateLimiter{lastSent: make(map[string]time.Time)}
}

// Allow reports whether a notification for key may be sent now.
func (r *RateLimiter) Allow(key string, cooldown time.Duration) bool {
	if cooldown <= 0 {
		return true
	}
	now := time.Now().UTC()
	r.mu.Lock()
	defer r.mu.Unlock()
	if last, ok := r.lastSent[key]; ok {
		if now.Sub(last) < cooldown {
			return false
		}
	}
	r.lastSent[key] = now
	return true
}

// Reset clears all cooldown state (for tests).
func (r *RateLimiter) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.lastSent = make(map[string]time.Time)
}

func cooldownKey(repo, eventType string) string {
	return repo + "|" + eventType
}
