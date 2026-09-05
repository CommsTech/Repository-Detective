package doctor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"git.commsnet.org/commstech/repository-detective/internal/privacy"
	"git.commsnet.org/commstech/repository-detective/internal/security"
	"git.commsnet.org/commstech/repository-detective/store"
)

// Run executes the full diagnostic suite from Input (no live network unless
// callers already filled forge/scanner probe fields).
func Run(in Input) Report {
	var checks []Check

	checks = append(checks, checkBuild(in)...)
	checks = append(checks, checkConfig(in)...)
	checks = append(checks, checkAuth(in)...)
	checks = append(checks, checkDatabase(in)...)
	checks = append(checks, checkWorkspace(in)...)
	checks = append(checks, checkPrivacy(in)...)
	checks = append(checks, checkForge(in)...)
	checks = append(checks, checkPermissions(in)...)
	checks = append(checks, checkWebhook(in)...)
	checks = append(checks, checkProofs(in)...)
	checks = append(checks, checkPolicy(in)...)
	checks = append(checks, checkScanners(in)...)
	checks = append(checks, checkAI(in)...)
	checks = append(checks, checkNotifications(in)...)
	checks = append(checks, checkRunners(in)...)
	checks = append(checks, checkRemediation(in)...)

	overall, reqFail, optWarn := ComputeOverall(checks)
	summary := summarize(overall, reqFail, optWarn, in)

	return Report{
		GeneratedAt:      time.Now().UTC(),
		Version:          in.Version,
		Commit:           in.Commit,
		Edition:          firstNonEmpty(in.Edition, "community"),
		Overall:          overall,
		OnboardingReady:  OnboardingState(overall, false),
		Summary:          summary,
		Checks:           checks,
		RequiredFailed:   reqFail,
		OptionalWarnings: optWarn,
	}
}

func summarize(overall string, reqFail, optWarn int, in Input) string {
	switch overall {
	case OverallHealthy:
		return fmt.Sprintf("All requirements for configured mode (profile=%s, policy=%s, privacy=%s) are satisfied.",
			nz(in.ScanProfile, "standard"), nz(in.PolicyMode, "Observe"), privacy.NormalizeMode(in.PrivacyMode))
	case OverallDegraded:
		return fmt.Sprintf("Core operation can work with limitations (%d warning(s)); required failures=%d.", optWarn, reqFail)
	default:
		return fmt.Sprintf("At least one required component cannot function (%d required error(s)).", reqFail)
	}
}

func checkBuild(in Input) []Check {
	return []Check{{
		ID:       "build.version",
		Category: "build",
		State:    StatePass,
		Summary:  "Repository Detective build metadata",
		Detail:   fmt.Sprintf("version=%s commit=%s edition=%s", nz(in.Version, "dev"), nz(in.Commit, "unknown"), nz(in.Edition, "community")),
		Proof:    ProofConfigOnly,
	}}
}

func checkConfig(in Input) []Check {
	if in.ConfigValid {
		return []Check{{
			ID: "config.valid", Category: "configuration", State: StatePass,
			Summary: "Configuration parses and validates", Detail: sanitizeDetail(in.ConfigDetail), Proof: ProofConfigOnly, Required: true,
		}}
	}
	return []Check{{
		ID: "config.valid", Category: "configuration", State: StateError,
		Summary: "Configuration invalid", Detail: sanitizeDetail(in.ConfigDetail),
		Remediation: "Fix configuration errors in .env / config.yaml", Proof: ProofConfigOnly, Required: true,
	}}
}

func checkAuth(in Input) []Check {
	var out []Check
	mode := strings.ToLower(strings.TrimSpace(in.AuthMode))
	if mode == "" {
		mode = "api_key_only"
	}
	out = append(out, Check{
		ID: "auth.mode", Category: "authentication", State: StatePass,
		Summary: fmt.Sprintf("Auth mode: %s", mode),
		Detail:  "Runtime default api_key_only preserves existing installs; local session recommended for new installs.",
		Proof:   ProofConfigOnly,
	})
	if mode == "local" && !in.SessionSecretConfigured {
		out = append(out, Check{
			ID: "auth.session_secret", Category: "authentication", State: StateError, Required: true,
			Summary: "auth_mode=local requires session_secret", Remediation: "Set REPOSITORY_DETECTIVE_SESSION_SECRET",
			Proof: ProofConfigOnly,
		})
	}
	if !in.APIKeyConfigured {
		out = append(out, Check{
			ID: "auth.api_key", Category: "authentication", State: StateWarning,
			Summary: "API key not configured", Detail: "Automation API will reject requests until set.",
			Remediation: "Set REPOSITORY_DETECTIVE_API_KEY", Proof: ProofConfigOnly,
		})
	} else {
		out = append(out, Check{
			ID: "auth.api_key", Category: "authentication", State: StatePass,
			Summary: "API key configured (value not shown)", Proof: ProofConfigOnly,
		})
	}
	if in.RejectQueryStringAPIKey {
		out = append(out, Check{
			ID: "auth.query_api_key", Category: "authentication", State: StatePass,
			Summary: "Query-string API keys rejected", Proof: ProofConfigOnly,
		})
	} else {
		out = append(out, Check{
			ID: "auth.query_api_key", Category: "authentication", State: StateWarning,
			Summary: "Query-string API keys still accepted (compatibility)",
			Detail:  "Recommended for new installs: REPOSITORY_DETECTIVE_REJECT_QUERY_STRING_API_KEY=true",
			Proof:   ProofConfigOnly,
		})
	}
	return out
}

func checkDatabase(in Input) []Check {
	if !in.DatabaseEnabled {
		return []Check{{
			ID: "database.enabled", Category: "database", State: StateNotConfigured,
			Summary: "Database disabled", Detail: "Store-backed features unavailable", Proof: ProofConfigOnly,
		}}
	}
	if in.DatabaseOK {
		return []Check{{
			ID: "database.connectivity", Category: "database", State: StatePass, Required: true,
			Summary: "Database connectivity OK",
			Detail:  fmt.Sprintf("schema_version=%d", in.SchemaVersion),
			Proof:   ProofRuntimeCheck,
		}}
	}
	return []Check{{
		ID: "database.connectivity", Category: "database", State: StateError, Required: true,
		Summary: "Database unavailable", Detail: sanitizeDetail(in.DatabaseDetail),
		Remediation: "Check REPOSITORY_DETECTIVE_DATABASE_PATH and permissions", Proof: ProofRuntimeCheck,
	}}
}

func checkWorkspace(in Input) []Check {
	dir := strings.TrimSpace(in.WorkspaceDir)
	if dir == "" {
		return []Check{{
			ID: "workspace.path", Category: "storage", State: StateWarning,
			Summary: "Workspace path not configured", Proof: ProofConfigOnly,
		}}
	}
	if in.WorkspaceOK {
		return []Check{{
			ID: "workspace.writable", Category: "storage", State: StatePass, Required: true,
			Summary: "Workspace writable", Detail: dir, Proof: ProofRuntimeCheck,
		}}
	}
	// Probe if caller did not fill WorkspaceOK
	if err := ensureWritableDir(dir); err != nil {
		return []Check{{
			ID: "workspace.writable", Category: "storage", State: StateError, Required: true,
			Summary: "Workspace not writable", Detail: err.Error(),
			Remediation: "Fix permissions on workspace/data directories", Proof: ProofRuntimeCheck,
		}}
	}
	return []Check{{
		ID: "workspace.writable", Category: "storage", State: StatePass, Required: true,
		Summary: "Workspace writable", Detail: dir, Proof: ProofRuntimeCheck,
	}}
}

func ensureWritableDir(dir string) error {
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return err
	}
	probe := filepath.Join(dir, ".rd-doctor-write-probe")
	if err := os.WriteFile(probe, []byte("ok"), 0o600); err != nil {
		return err
	}
	_ = os.Remove(probe)
	return nil
}

func checkPrivacy(in Input) []Check {
	mode := privacy.NormalizeMode(in.PrivacyMode)
	out := []Check{{
		ID: "privacy.mode", Category: "privacy", State: StatePass, Required: true,
		Summary: fmt.Sprintf("Privacy mode: %s", mode),
		Detail:  privacy.LocalOnlyGuarantee,
		Proof:   ProofConfigOnly,
	}}
	forgeClass := in.ForgeLocality
	if forgeClass == "" && in.ForgeURL != "" {
		if c, _, err := privacy.ClassifyURL(in.ForgeURL); err == nil {
			forgeClass = c
		} else {
			forgeClass = privacy.ClassUnknown
		}
	}
	if mode == privacy.ModeLocalOnly && forgeClass == privacy.ClassExternal {
		out = append(out, Check{
			ID: "privacy.forge_egress", Category: "privacy", State: StateWarning,
			Summary: "Forge is EXTERNAL under LOCAL_ONLY",
			Detail:  "Repository Detective blocks external AI/notification egress, but findings and PR summaries may still be sent to the configured external forge.",
			Proof:   ProofRuntimeCheck,
		})
	} else if in.ForgeURL != "" {
		out = append(out, Check{
			ID: "privacy.forge_locality", Category: "privacy", State: StatePass,
			Summary: fmt.Sprintf("Forge locality: %s", nz(forgeClass, privacy.ClassUnknown)),
			Proof:   ProofRuntimeCheck,
		})
	}
	return out
}

func checkForge(in Input) []Check {
	if strings.TrimSpace(in.ForgeURL) == "" || !in.ForgeTokenSet {
		return []Check{{
			ID: "forge.configured", Category: "forge", State: StateError, Required: true,
			Summary:     "Forge URL or token not configured",
			Remediation: "Set REPOSITORY_DETECTIVE_GITEA_URL and REPOSITORY_DETECTIVE_GITEA_TOKEN",
			Proof:       ProofConfigOnly,
		}}
	}
	var out []Check
	out = append(out, Check{
		ID: "forge.url", Category: "forge", State: StatePass,
		Summary: "Forge URL configured", Detail: in.ForgeURL, Proof: ProofConfigOnly, Required: true,
	})
	if in.SkipLiveForgeProbe {
		return out
	}
	if in.ForgeReachable && in.ForgeAuthOK {
		out = append(out, Check{
			ID: "forge.auth", Category: "forge", State: StatePass, Required: true,
			Summary: "Forge reachable and authenticated",
			Detail:  fmt.Sprintf("version=%s %s", nz(in.ForgeVersion, "unknown"), sanitizeDetail(in.ForgeAuthDetail)),
			Proof:   ProofRuntimeCheck,
		})
	} else if !in.ForgeReachable {
		out = append(out, Check{
			ID: "forge.auth", Category: "forge", State: StateError, Required: true,
			Summary: "Forge unreachable", Detail: sanitizeDetail(in.ForgeReachDetail),
			Remediation: "Check network/DNS and Gitea URL", Proof: ProofRuntimeCheck,
		})
	} else {
		out = append(out, Check{
			ID: "forge.auth", Category: "forge", State: StateError, Required: true,
			Summary: "Forge authentication failed", Detail: sanitizeDetail(in.ForgeAuthDetail),
			Remediation: "Verify token scopes and expiry", Proof: ProofRuntimeCheck,
		})
	}
	return out
}

func checkPermissions(in Input) []Check {
	if in.RepoPerms == nil {
		return []Check{{
			ID: "forge.permissions", Category: "forge", State: StateNotConfigured,
			Summary: "Repository permission matrix not probed",
			Detail:  "Select a repository during onboarding or pass owner/repo to doctor.",
			Proof:   ProofNotProven,
		}}
	}
	p := in.RepoPerms
	mk := func(id, label, state string, required bool) Check {
		st := state
		if st == "" {
			st = "NOT_GRANTED"
		}
		checkState := StatePass
		switch st {
		case "PASS":
			checkState = StatePass
		case "NOT_GRANTED":
			if required {
				checkState = StateError
			} else {
				checkState = StateWarning
			}
		default:
			checkState = StateError
		}
		return Check{
			ID: id, Category: "forge", State: checkState, Required: required,
			Summary: fmt.Sprintf("%s: %s", label, st), Detail: sanitizeDetail(p.Detail), Proof: ProofRuntimeCheck,
		}
	}
	// Scanning needs read. Issue filing needs issues write for Warn/Enforce issue_only+.
	policyNeedsIssues := in.EnforcementMode == store.EnforcementWarn || in.EnforcementMode == store.EnforcementEnforce ||
		strings.EqualFold(in.PolicyMode, "Warn") || strings.EqualFold(in.PolicyMode, "Enforce")
	return []Check{
		mk("forge.perm.repository_read", "Repository read", p.RepositoryRead, true),
		mk("forge.perm.issues_write", "Issues write", p.IssuesWrite, policyNeedsIssues),
		mk("forge.perm.commit_status", "Commit status write", p.CommitStatusWrite, false),
		mk("forge.perm.pr_comment", "PR comment write", p.PRCommentWrite, false),
		mk("forge.perm.remediation", "Branch/PR remediation permission", p.BranchPRRemediation, false),
	}
}

func checkWebhook(in Input) []Check {
	var out []Check
	if !in.WebhookSecretSet {
		out = append(out, Check{
			ID: "webhook.secret", Category: "webhook", State: StateWarning,
			Summary:     "Webhook secret not configured",
			Remediation: "Set REPOSITORY_DETECTIVE_WEBHOOK_SECRET", Proof: ProofConfigOnly,
		})
	} else {
		out = append(out, Check{
			ID: "webhook.secret", Category: "webhook", State: StatePass,
			Summary: "Webhook secret configured (value not shown)", Proof: ProofConfigOnly,
		})
	}
	cb := strings.TrimSuffix(in.PublicURL, "/") + "/webhook"
	if strings.TrimSpace(in.PublicURL) == "" {
		out = append(out, Check{
			ID: "webhook.callback_url", Category: "webhook", State: StateWarning,
			Summary:     "Public URL not set — webhook callback unknown",
			Remediation: "Set REPOSITORY_DETECTIVE_PUBLIC_URL", Proof: ProofConfigOnly,
		})
	} else {
		out = append(out, Check{
			ID: "webhook.callback_url", Category: "webhook", State: StatePass,
			Summary: "Expected webhook callback URL", Detail: cb, Proof: ProofConfigOnly,
		})
	}
	if in.WebhookRegistered {
		out = append(out, Check{
			ID: "webhook.registration", Category: "webhook", State: StatePass,
			Summary: "Webhook registration: PASS", Detail: sanitizeDetail(in.WebhookRegistrationDetail),
			Proof: ProofIntegration,
		})
	} else {
		out = append(out, Check{
			ID: "webhook.registration", Category: "webhook", State: StateNotProven,
			Summary: "Webhook registration: NOT YET PROVEN",
			Detail:  "Register via onboarding or confirm hooks on the forge.",
			Proof:   ProofNotProven,
		})
	}
	if in.WebhookDeliveryProven {
		out = append(out, Check{
			ID: "webhook.delivery", Category: "webhook", State: StatePass,
			Summary: "Real forge webhook delivery proven",
			Detail:  in.WebhookLastDelivery, Proof: ProofE2E,
		})
	} else {
		detail := "Real Gitea delivery: NOT YET PROVEN — awaiting first repository event"
		if in.WebhookLastError != "" {
			detail += "; last_error=" + sanitizeDetail(in.WebhookLastError)
		}
		out = append(out, Check{
			ID: "webhook.delivery", Category: "webhook", State: StateNotProven,
			Summary: "Webhook delivery not E2E proven", Detail: detail, Proof: ProofNotProven,
		})
	}
	return out
}

func checkProofs(in Input) []Check {
	var out []Check
	if in.FirstScanProven {
		out = append(out, Check{
			ID: "proof.first_scan", Category: "proof", State: StatePass,
			Summary: "FIRST_SCAN_PROVEN", Detail: sanitizeDetail(in.FirstScanDetail), Proof: ProofIntegration,
		})
	} else {
		out = append(out, Check{
			ID: "proof.first_scan", Category: "proof", State: StateNotProven,
			Summary: "FIRST_SCAN_PROVEN not recorded",
			Detail:  "A terminal production-path scan has not yet been persisted as proof.",
			Proof:   ProofNotProven,
		})
	}
	if in.WebhookDeliveryProven {
		out = append(out, Check{
			ID: "proof.webhook_delivery", Category: "proof", State: StatePass,
			Summary: "WEBHOOK_DELIVERY_E2E_PROVEN", Detail: sanitizeDetail(in.WebhookLastDelivery), Proof: ProofE2E,
		})
	}
	return out
}

func checkPolicy(in Input) []Check {
	profile := store.NormalizeScanProfile(in.ScanProfile)
	req := in.RequiredScanners
	if len(req) == 0 {
		req = store.ProfileDeclaredRequiredScanners(profile)
	}
	return []Check{
		{
			ID: "policy.profile", Category: "policy", State: StatePass,
			Summary: fmt.Sprintf("Analysis profile: %s", profile), Proof: ProofConfigOnly,
		},
		{
			ID: "policy.mode", Category: "policy", State: StatePass,
			Summary: fmt.Sprintf("Policy mode: %s", nz(in.PolicyMode, "Observe")),
			Detail:  "Policy outcomes describe owner-configured compliance — not that code is safe or secure.",
			Proof:   ProofConfigOnly,
		},
		{
			ID: "policy.required_scanners", Category: "policy", State: StatePass,
			Summary: fmt.Sprintf("Required scanners declared: %d", len(req)),
			Detail:  strings.Join(req, ", "), Proof: ProofConfigOnly,
		},
	}
}

func checkScanners(in Input) []Check {
	if len(in.ScannerTools) == 0 {
		return []Check{{
			ID: "scanner.probes", Category: "scanner", State: StateWarning,
			Summary: "No scanner probes available", Proof: ProofNotProven,
		}}
	}
	requiredSet := map[string]struct{}{}
	for _, r := range in.RequiredScanners {
		requiredSet[strings.ToLower(r)] = struct{}{}
	}
	var out []Check
	requiredOK := 0
	requiredTotal := 0
	for _, t := range in.ScannerTools {
		name := strings.ToLower(t.Name)
		role := t.Role
		if role == "" {
			if _, ok := requiredSet[name]; ok {
				role = store.ScannerRoleRequired
			} else if t.EnabledInConfig {
				role = store.ScannerRoleOptional
			} else {
				role = store.ScannerRoleInformational
			}
		}
		required := role == store.ScannerRoleRequired
		if required {
			requiredTotal++
		}
		st := StatePass
		summary := fmt.Sprintf("%s available", t.Name)
		if !t.Available {
			if required {
				st = StateError
				summary = fmt.Sprintf("%s REQUIRED but unavailable — policy readiness EVALUATION_INCOMPLETE", t.Name)
			} else if t.EnabledInConfig {
				st = StateWarning
				summary = fmt.Sprintf("%s enabled but unavailable", t.Name)
			} else {
				st = StateNotConfigured
				summary = fmt.Sprintf("%s not enabled", t.Name)
			}
		} else if required {
			requiredOK++
		}
		out = append(out, Check{
			ID:       "scanner." + name + ".available",
			Category: "scanner",
			State:    st,
			Required: required,
			Summary:  summary,
			Detail:   fmt.Sprintf("role=%s version=%s status=%s", role, nz(t.Version, "unknown"), t.StatusState),
			Proof:    ProofRuntimeCheck,
			Remediation: func() string {
				if !t.Available && required {
					return "Install the scanner binary in the Repository Detective image/PATH; do not disable REQUIRED scanners to force POLICY_MET"
				}
				return ""
			}(),
		})
	}
	out = append(out, Check{
		ID: "scanner.required_ratio", Category: "scanner",
		State: func() string {
			if requiredTotal == 0 {
				return StateWarning
			}
			if requiredOK == requiredTotal {
				return StatePass
			}
			return StateError
		}(),
		Required: requiredTotal > 0,
		Summary:  fmt.Sprintf("Required scanners available: %d/%d", requiredOK, requiredTotal),
		Proof:    ProofRuntimeCheck,
	})
	return out
}

func checkAI(in Input) []Check {
	if !in.AIEnabled {
		return []Check{{
			ID: "ai.status", Category: "ai", State: StatePass,
			Summary: "AI Analysis: DISABLED — Status: VALID",
			Detail:  "Deterministic scanners operate without an AI provider. This is not a warning.",
			Proof:   ProofConfigOnly,
		}}
	}
	st := StatePass
	if !in.AIEgressAllowed {
		st = StateError
	}
	return []Check{{
		ID: "ai.status", Category: "ai", State: st, Required: true,
		Summary: fmt.Sprintf("AI enabled (%s / %s)", nz(in.AIProvider, "unknown"), nz(in.AIModel, "unset")),
		Detail: fmt.Sprintf("endpoint=%s class=%s allowed=%v — %s",
			nz(in.AIBaseURL, "(default)"), nz(in.AILocality, privacy.ClassUnknown), in.AIEgressAllowed, sanitizeDetail(in.AIEgressReason)),
		Proof: ProofRuntimeCheck,
		Remediation: func() string {
			if !in.AIEgressAllowed {
				return "Use a LOCAL endpoint under LOCAL_ONLY, or change privacy_mode"
			}
			return ""
		}(),
	}}
}

func checkNotifications(in Input) []Check {
	if len(in.NotificationChannels) == 0 {
		return []Check{{
			ID: "notify.channels", Category: "notifications", State: StateNotConfigured,
			Summary: "No notification channels configured", Proof: ProofConfigOnly,
		}}
	}
	var out []Check
	for _, ch := range in.NotificationChannels {
		if !ch.Enabled {
			continue
		}
		st := StatePass
		if !ch.Allowed {
			st = StateError
		}
		out = append(out, Check{
			ID: "notify." + strings.ToLower(ch.Name), Category: "notifications", State: st,
			Summary: fmt.Sprintf("Notification %s locality=%s", ch.Name, nz(ch.Locality, "UNKNOWN")),
			Detail:  "Secret values not shown", Proof: ProofConfigOnly,
		})
	}
	if len(out) == 0 {
		return []Check{{
			ID: "notify.channels", Category: "notifications", State: StateNotConfigured,
			Summary: "Notification channels configured but disabled", Proof: ProofConfigOnly,
		}}
	}
	return out
}

func checkRunners(in Input) []Check {
	if !in.RunnerDelegationEnabled {
		return []Check{{
			ID: "runner.delegation", Category: "runners", State: StateNotConfigured,
			Summary: "Runner delegation not configured", Proof: ProofConfigOnly,
		}}
	}
	st := StatePass
	if !in.RunnerOnline {
		st = StateWarning
	}
	iso := StateNotProven
	isoSummary := "Runner isolation: NOT_PROVEN"
	if in.RunnerIsolationProven {
		iso = StatePass
		isoSummary = "Runner isolation: PROVEN"
	}
	return []Check{
		{
			ID: "runner.online", Category: "runners", State: st,
			Summary: "Runner delegation configured", Detail: sanitizeDetail(in.RunnerDetail), Proof: ProofRuntimeCheck,
		},
		{
			ID: "runner.isolation", Category: "runners", State: iso,
			Summary: isoSummary,
			Detail:  "Online status alone does not prove isolation.", Proof: ProofNotProven,
		},
	}
}

func checkRemediation(in Input) []Check {
	var out []Check
	if in.RemediationPlannerEnabled {
		out = append(out, Check{
			ID: "remediation.planner", Category: "remediation", State: StatePass,
			Summary: "Remediation planning: AVAILABLE", Proof: ProofConfigOnly,
		})
	} else {
		out = append(out, Check{
			ID: "remediation.planner", Category: "remediation", State: StateNotConfigured,
			Summary: "Remediation planning: NOT_CONFIGURED", Proof: ProofConfigOnly,
		})
	}
	if in.RemediationPREnabled {
		exec := "LOCAL (control plane)"
		if in.RemediationUseRunner {
			exec = "RUNNER (requested)"
		}
		out = append(out, Check{
			ID: "remediation.pr", Category: "remediation", State: StateWarning,
			Summary:     "Remediation PR creation: ENABLED",
			Detail:      fmt.Sprintf("validation_execution=%s class_b_isolation=%s", exec, nz(in.ClassBIsolation, "NOT_PROVEN")),
			Proof:       ProofConfigOnly,
			Remediation: "See docs/RD-008B_CLASS_B_EXECUTION.md — Class-B execution is not a proven sandbox on the control plane.",
		})
	} else {
		out = append(out, Check{
			ID: "remediation.pr", Category: "remediation", State: StateNotConfigured,
			Summary: "Remediation PR creation: NOT_CONFIGURED (safe default)", Proof: ProofConfigOnly,
		})
	}
	out = append(out, Check{
		ID: "remediation.class_b", Category: "remediation", State: StateNotProven,
		Summary: fmt.Sprintf("Class-B isolation: %s", nz(in.ClassBIsolation, "NOT_PROVEN")),
		Detail:  "Allowlisted local validation can run on the control plane when remediation PRs are enabled. Not product-enforced sandboxing.",
		Proof:   ProofNotProven,
	})
	return out
}

func sanitizeDetail(s string) string {
	return security.SanitizeDiagnostic(s, 2000)
}

func nz(s, def string) string {
	if strings.TrimSpace(s) == "" {
		return def
	}
	return s
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
