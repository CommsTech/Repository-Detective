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
)

// ShouldDelegate reports whether a scan trigger should create a runner job.
func ShouldDelegate(cfg Config, effective store.EffectiveSettings, triggerType string) DelegationDecision {
	cfg = cfg.Normalized()
	if !cfg.DelegationEnabled || cfg.SharedSecret == "" {
		return DecisionCore
	}
	if cfg.Mode == ModeCore {
		return DecisionCore
	}
	switch triggerType {
	case store.TriggerScheduled, store.TriggerManual:
		// Phase 12: scheduled and manual full scans only.
	default:
		return DecisionCore
	}
	switch strings.ToLower(strings.TrimSpace(effective.RunnerPolicy)) {
	case "core":
		return DecisionCore
	case ModeGiteaActions, ModeAuto:
		if cfg.Mode == ModeGiteaActions || cfg.Mode == ModeAuto || cfg.Mode == ModeNative {
			return DecisionDelegate
		}
		return DecisionCore
	case ModeNative:
		if cfg.Mode == ModeNative || cfg.Mode == ModeAuto {
			return DecisionDelegate
		}
		return DecisionCore
	default:
		return DecisionCore
	}
}

// ShouldFallbackToCore reports whether auto policy should fall back when delegation fails.
func ShouldFallbackToCore(effective store.EffectiveSettings) bool {
	return strings.ToLower(strings.TrimSpace(effective.RunnerPolicy)) == ModeAuto
}
