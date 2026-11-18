// Package doctor provides reusable operator diagnostics for onboarding Verify
// and `repository-detective doctor` (RD-013 / RD-014).
//
// Check states describe system readiness — never repository security.
package doctor

import (
	"strings"
	"time"
)

// Component-level states.
const (
	StatePass          = "PASS"
	StateWarning       = "WARNING"
	StateError         = "ERROR"
	StateNotConfigured = "NOT_CONFIGURED"
	StateNotApplicable = "NOT_APPLICABLE"
	StateNotProven     = "NOT_PROVEN"
)

// Overall report results.
const (
	OverallHealthy  = "HEALTHY"
	OverallDegraded = "DEGRADED"
	OverallNotReady = "NOT_READY"
)

// Onboarding readiness (RD-013 Ready step).
const (
	Ready           = "READY"
	ReadyWithLimits = "READY_WITH_LIMITATIONS"
	NotReady        = "NOT_READY"
)

// Proof levels for individual checks.
const (
	ProofConfigOnly   = "CONFIG_ONLY"
	ProofRuntimeCheck = "RUNTIME_CHECK"
	ProofIntegration  = "INTEGRATION_CHECK"
	ProofE2E          = "E2E_PROVEN"
	ProofNotProven    = "NOT_PROVEN"
)

// Check is one diagnostic finding.
type Check struct {
	ID          string `json:"id"`
	Category    string `json:"category"`
	State       string `json:"state"`
	Summary     string `json:"summary"`
	Detail      string `json:"detail,omitempty"`
	Remediation string `json:"remediation,omitempty"`
	Proof       string `json:"proof,omitempty"`
	Required    bool   `json:"required"`
}

// Report is the machine-readable doctor output.
type Report struct {
	GeneratedAt      time.Time `json:"generated_at"`
	Version          string    `json:"version,omitempty"`
	Commit           string    `json:"commit,omitempty"`
	Edition          string    `json:"edition,omitempty"`
	Overall          string    `json:"overall"`
	OnboardingReady  string    `json:"onboarding_ready,omitempty"`
	Summary          string    `json:"summary"`
	Checks           []Check   `json:"checks"`
	RequiredFailed   int       `json:"required_failed"`
	OptionalWarnings int       `json:"optional_warnings"`
}

// Input configures a doctor run. Secrets may be present for live probes but
// must never appear in Report JSON/text (use RedactReport).
type Input struct {
	Version string
	Commit  string
	Edition string

	// Application
	DatabaseEnabled bool
	DatabaseOK      bool
	DatabaseDetail  string
	SchemaVersion   int
	WorkspaceDir    string
	WorkspaceOK     bool
	WorkspaceDetail string
	ConfigValid     bool
	ConfigDetail    string

	// Auth
	AuthMode                string
	SessionSecretConfigured bool
	RejectQueryStringAPIKey bool
	WarnQueryStringAPIKey   bool
	APIKeyConfigured        bool

	// Privacy / AI
	PrivacyMode          string
	AIEnabled            bool // LLM auditors or advisory invokable
	AIProvider           string
	AIBaseURL            string
	AIModel              string
	AILocality           string // from privacy package
	AIEgressAllowed      bool
	AIEgressReason       string
	NotificationChannels []NotificationChannel

	// Forge
	ForgeURL           string
	ForgeTokenSet      bool
	ForgeLocality      string
	ForgeReachable     bool
	ForgeReachDetail   string
	ForgeAuthOK        bool
	ForgeAuthDetail    string
	ForgeVersion       string
	SkipLiveForgeProbe bool // unit tests

	// Selected repo (optional)
	RepoOwner string
	RepoName  string
	RepoPerms *RepoPermissionResult

	// Webhook
	PublicURL                 string
	WebhookSecretSet          bool
	WebhookRegistered         bool // known from last register / list
	WebhookRegistrationDetail string
	WebhookDeliveryProven     bool
	WebhookLastDelivery       string
	WebhookLastError          string
	FirstScanProven           bool
	FirstScanDetail           string
	WebhookScanProven         bool

	// Policy / profile
	ScanProfile     string
	PolicyMode      string // Observe / Warn / Enforce display
	EnforcementMode string // monitor_only / issue_only / gate_pr

	// Scanners: from operator.CheckTools + roles
	ScannerTools     []ScannerToolInput
	RequiredScanners []string

	// Runners
	RunnerDelegationEnabled bool
	RunnerOnline            bool
	RunnerDetail            string
	RunnerIsolationProven   bool

	// Remediation (Class-B honesty)
	RemediationPlannerEnabled bool
	RemediationPREnabled      bool
	RemediationUseRunner      bool
	ClassBIsolation           string // PROVEN / PARTIAL / NOT_PROVEN
	ClassBExecutionAllowed    bool   // can control-plane run validation
}

// NotificationChannel is a configured notify destination (no secrets).
type NotificationChannel struct {
	Name     string
	URL      string // may be empty for telegram
	Locality string
	Enabled  bool
	Allowed  bool // privacy gate
}

// ScannerToolInput is one scanner probe result for doctor.
type ScannerToolInput struct {
	Name            string
	Role            string // REQUIRED / OPTIONAL / INFORMATIONAL
	EnabledInConfig bool
	Available       bool
	Version         string
	StatusState     string
	Path            string
}

// RepoPermissionResult is a non-secret permission matrix for one repository.
type RepoPermissionResult struct {
	RepositoryRead      string // PASS / ERROR / NOT_GRANTED
	IssuesWrite         string
	CommitStatusWrite   string
	PRCommentWrite      string
	BranchPRRemediation string
	Detail              string
}

// ComputeOverall derives HEALTHY / DEGRADED / NOT_READY from checks.
func ComputeOverall(checks []Check) (overall string, requiredFailed, optionalWarnings int) {
	for _, c := range checks {
		switch c.State {
		case StateError:
			if c.Required {
				requiredFailed++
			} else {
				optionalWarnings++
			}
		case StateWarning, StateNotProven:
			optionalWarnings++
		}
	}
	switch {
	case requiredFailed > 0:
		return OverallNotReady, requiredFailed, optionalWarnings
	case optionalWarnings > 0:
		return OverallDegraded, requiredFailed, optionalWarnings
	default:
		return OverallHealthy, requiredFailed, optionalWarnings
	}
}

// OnboardingState maps overall + optional first-scan evidence to READY*.
func OnboardingState(overall string, firstScanProven bool) string {
	switch overall {
	case OverallHealthy:
		if firstScanProven {
			return Ready
		}
		return Ready // healthy config is READY; first scan is additive evidence
	case OverallDegraded:
		return ReadyWithLimits
	default:
		return NotReady
	}
}

// HasSecretLeak reports whether text appears to contain credential material.
func HasSecretLeak(text string, secrets ...string) bool {
	lower := strings.ToLower(text)
	for _, s := range secrets {
		s = strings.TrimSpace(s)
		if len(s) < 8 {
			continue
		}
		if strings.Contains(text, s) || strings.Contains(lower, strings.ToLower(s)) {
			return true
		}
	}
	return false
}
