package runner

import (
	"strings"

	"git.commsnet.org/commstech/repository-detective/store"
)

// DelegationDecision indicates whether a scan should run on core or delegate.
type DelegationDecision int

const (
	DecisionCore DelegationDecision = iota
	DecisionDelegate
	DecisionUnsupported
)

// ShouldDelegate reports whether a scan trigger should create a runner job.
func ShouldDelegate(cfg Config, effective store.EffectiveSettings, triggerType string) DelegationDecision {
	cfg = cfg.Normalized()
	policy := strings.ToLower(strings.TrimSpace(effective.RunnerPolicy))
	// Explicit selection of a nonexistent sandbox must be an error, never an
	// implicit scan of untrusted repository code in the core process.
	if policy == ModeCore {
		return DecisionCore
	}
	if cfg.DelegationEnabled && (cfg.Mode == ModeGiteaActions || policy == ModeGiteaActions) {
		return DecisionUnsupported
	}
	// A repository explicitly requesting an off-core execution policy must
	// never execute its untrusted source in the control plane due to a missing
	// secret, disabled delegation or an unsupported trigger.
	wantsSandbox := policy == ModeNative || policy == ModeAuto || policy == ModeGiteaActions
	if wantsSandbox && (!cfg.DelegationEnabled || cfg.SharedSecret == "" || cfg.Mode == ModeCore) {
		return DecisionUnsupported
	}
	if !cfg.DelegationEnabled || cfg.SharedSecret == "" || cfg.Mode == ModeCore {
		return DecisionCore
	}
	switch triggerType {
	case store.TriggerScheduled, store.TriggerManual:
		// Native delegation currently supports scheduled and manual scans.
	default:
		if wantsSandbox {
			return DecisionUnsupported
		}
		return DecisionCore
	}
	switch strings.ToLower(strings.TrimSpace(effective.RunnerPolicy)) {
	case "core":
		return DecisionCore
	case ModeGiteaActions:
		return DecisionUnsupported
	case ModeAuto:
		return DecisionDelegate // native worker only until Actions integration is complete
	case ModeNative:
		if cfg.Mode == ModeNative || cfg.Mode == ModeAuto {
			return DecisionDelegate
		}
		return DecisionUnsupported
	default:
		return DecisionCore
	}
}

// ShouldFallbackToCore is fail-closed: a requested isolated scan must never
// execute in the control plane just because a runner was unavailable.
func ShouldFallbackToCore(_ store.EffectiveSettings) bool {
	return false
}
