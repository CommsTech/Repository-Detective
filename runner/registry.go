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

// MatchesCapabilities verifies the worker recently registered exactly the same
// capabilities that it is requesting. This is a scheduling constraint, not a
// substitute for per-worker credentials or an isolation boundary.
func (r *Registry) MatchesCapabilities(runnerID string, requested []string, maxAge time.Duration) bool {
	if r == nil || runnerID == "" || len(requested) == 0 || len(requested) > 16 {
		return false
	}
	r.mu.RLock()
	hb, ok := r.workers[runnerID]
	r.mu.RUnlock()
	if !ok || !hb.LastSeenAt.After(time.Now().UTC().Add(-maxAge)) {
		return false
	}
	actual := make(map[string]struct{}, len(hb.Capabilities))
	for _, typ := range hb.Capabilities {
		actual[typ] = struct{}{}
	}
	for _, typ := range requested {
		if _, ok := actual[typ]; !ok {
			return false
		}
	}
	return true
}
