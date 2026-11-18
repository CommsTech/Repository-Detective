package scanners

import (
	"sync"
)

var (
	deterministicMu           sync.RWMutex
	deterministicSources      = map[string]struct{}{}
	deterministicBootstrapped bool
)

func bootstrapDeterministicSources() {
	deterministicMu.Lock()
	defer deterministicMu.Unlock()
	if deterministicBootstrapped {
		return
	}
	for _, name := range []string{
		"trivy",
		"grype",
		"golangci-lint",
		"ruff",
		"shellcheck",
	} {
		deterministicSources[name] = struct{}{}
	}
	deterministicBootstrapped = true
}

// RegisterDeterministicSource marks an auditor/scanner source name as deterministic.
// Used when registering new scanners so analyzers recognize them without engine changes.
func RegisterDeterministicSource(name string) {
	if name == "" {
		return
	}
	bootstrapDeterministicSources()
	deterministicMu.Lock()
	deterministicSources[name] = struct{}{}
	deterministicMu.Unlock()
}

// IsDeterministicSource reports whether findings from this source skip LLM debate/PoC.
func IsDeterministicSource(name string) bool {
	bootstrapDeterministicSources()
	deterministicMu.RLock()
	_, ok := deterministicSources[name]
	deterministicMu.RUnlock()
	return ok
}

// DeterministicSourceNames returns registered deterministic scanner source names.
func DeterministicSourceNames() []string {
	bootstrapDeterministicSources()
	deterministicMu.RLock()
	defer deterministicMu.RUnlock()
	names := make([]string, 0, len(deterministicSources))
	for name := range deterministicSources {
		names = append(names, name)
	}
	return names
}
