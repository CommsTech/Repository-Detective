package runner

import (
	"sync"
	"time"
)

// WorkerHeartbeat records a native runner worker check-in.
type WorkerHeartbeat struct {
	RunnerID     string    `json:"runner_id"`
	Version      string    `json:"version,omitempty"`
	Capabilities []string  `json:"capabilities,omitempty"`
	LastSeenAt   time.Time `json:"last_seen_at"`
}

// Registry tracks recent runner worker heartbeats in memory.
type Registry struct {
	mu      sync.RWMutex
	workers map[string]WorkerHeartbeat
}

// NewRegistry creates an empty runner registry.
func NewRegistry() *Registry {
	return &Registry{workers: make(map[string]WorkerHeartbeat)}
}

// RecordHeartbeat stores or updates a worker heartbeat.
func (r *Registry) RecordHeartbeat(hb WorkerHeartbeat) {
	if r == nil || hb.RunnerID == "" {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if hb.LastSeenAt.IsZero() {
		hb.LastSeenAt = time.Now().UTC()
	}
	r.workers[hb.RunnerID] = hb
}

// ListHeartbeats returns workers seen recently.
func (r *Registry) ListHeartbeats(maxAge time.Duration) []WorkerHeartbeat {
	if r == nil {
		return nil
	}
	cutoff := time.Now().UTC().Add(-maxAge)
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]WorkerHeartbeat, 0, len(r.workers))
	for _, hb := range r.workers {
		if hb.LastSeenAt.After(cutoff) {
			out = append(out, hb)
		}
	}
	return out
}
