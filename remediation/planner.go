package remediation

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

// noopAIAdvisor is used when AI is disabled.
type noopAIAdvisor struct{}

func (noopAIAdvisor) SuggestPlan(_ FindingContext, draft Plan) (Plan, error) {
	return draft, nil
}

// StubAIAdvisor marks plans advisory when AI is enabled but no client is wired.
type StubAIAdvisor struct{}

func (StubAIAdvisor) SuggestPlan(_ FindingContext, draft Plan) (Plan, error) {
	draft.Advisory = true
	draft.Summary = strings.TrimSpace(draft.Summary + " (AI advisory — deterministic recipe applied)")
	draft.RequiresHumanReview = true
	draft.SafeForAutoPR = false
	return draft, nil
}

// Planner generates remediation plans without patches.
type Planner struct {
	cfg   Config
	ai    AIAdvisor
	newID func() string
}

// NewPlanner creates a remediation planner.
func NewPlanner(cfg Config, ai AIAdvisor, newID func() string) *Planner {
	if ai == nil {
		ai = noopAIAdvisor{}
	}
	if newID == nil {
		newID = NewPlanID
	}
	return &Planner{cfg: cfg, ai: ai, newID: newID}
}

// NewPlanID generates a unique remediation plan identifier.
func NewPlanID() string {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("rp-%d", time.Now().UTC().UnixNano())
	}
	return "rp-" + hex.EncodeToString(buf)
}

// ShouldPlan reports whether a finding qualifies for plan generation.
func (p *Planner) ShouldPlan(ctx FindingContext) bool {
	if p == nil || !p.cfg.Enabled {
		return false
	}
	if ctx.AuditID == "" && !ctx.ConnectedRepo {
		return false
	}
	if !passesSeverityGate(ctx.Severity, p.cfg.MinSeverity) {
		if !allowLowSeverityException(ctx) {
			return false
		}
	}
	if ctx.Confidence < p.cfg.MinConfidence {
		return false
	}
	return true
}

func allowLowSeverityException(ctx FindingContext) bool {
	if strings.ToLower(ctx.Severity) != "low" {
		return false
	}
	if ctx.Confidence < 0.90 {
		return false
	}
	category := strings.ToLower(ctx.Category)
	source := strings.ToLower(ctx.Source)
	switch category {
	case "dependency", "misconfiguration":
		return true
	case "code_quality", "maintainability":
		return source == "staticcheck" || source == "hadolint"
	default:
		return source == "staticcheck" || source == "hadolint" || source == "checkov"
	}
}

// Generate builds a remediation plan for a finding context.
func (p *Planner) Generate(ctx context.Context, fctx FindingContext) (Plan, error) {
	if p == nil {
		return Plan{}, fmt.Errorf("planner not configured")
	}
	if !p.ShouldPlan(fctx) {
		return Plan{}, fmt.Errorf("finding not eligible for remediation planning")
	}

	plan := ApplyRecipe(fctx)
	useAI := p.cfg.UseAI && p.cfg.GlobalAIAllowed && !fctx.FromAI && needsAIEnrichment(fctx)

	if useAI {
		// New advisors can honor the scan request cancellation. Preserve support for
		// existing AIAdvisor implementations used by third-party deployments.
		var enriched Plan
		var err error
		if contextual, ok := p.ai.(interface {
			SuggestPlanContext(context.Context, FindingContext, Plan) (Plan, error)
		}); ok {
			enriched, err = contextual.SuggestPlanContext(ctx, fctx, plan)
		} else {
			enriched, err = p.ai.SuggestPlan(fctx, plan)
		}
		if err != nil {
			// Never silently label a deterministic fallback as AI-reviewed.
			plan.Advisory = false
			plan.RequiresHumanReview = true
			plan.SafeForAutoPR = false
			plan.BlockedReasons = append(plan.BlockedReasons, "AI enrichment unavailable — verify provider and regenerate plan")
		} else {
			plan = enriched
			plan.Advisory = true
			plan.RequiresHumanReview = true
			plan.SafeForAutoPR = false
		}
	}

	now := time.Now().UTC()
	plan.ID = p.newID()
	plan.Status = StatusProposed
	plan.CreatedAt = now
	plan.UpdatedAt = now
	return plan, nil
}

func needsAIEnrichment(ctx FindingContext) bool {
	category := strings.ToLower(ctx.Category)
	if category == "secret" || category == "architecture" {
		return false
	}
	// Dependency CVEs need an actionable, version-aware investigation too.
	// Opt-in UseAI controls cost and privacy; AI never authorizes a patch PR.
	if category == "dependency" || category == "vulnerability" || category == "vuln" {
		return true
	}
	source := strings.ToLower(ctx.Source)
	if source == "trivy" && isTrivyMisconfiguration(source, strings.ToLower(ctx.RuleID), category) {
		return false // deterministic low-risk Dockerfile/IaC fix, not a dependency CVE
	}
	switch source {
	case "govulncheck", "grype", "trivy":
		return true
	case "gitleaks", "gosec", "staticcheck", "hadolint", "checkov", "graph", "test_gap":
		return false
	default:
		return ctx.FromAI || category == "security" || category == "unknown"
	}
}

func passesSeverityGate(severity, gate string) bool {
	return severityRank(severity) >= severityRank(gate)
}

func severityRank(sev string) int {
	switch strings.ToLower(strings.TrimSpace(sev)) {
	case "critical":
		return 5
	case "high":
		return 4
	case "medium", "warning", "warn":
		return 3
	case "low":
		return 2
	case "info", "informational", "note":
		return 1
	default:
		return 0
	}
}

// FindingContextFromStore builds planner input from a persisted finding.
func FindingContextFromStore(f FindingContext, connected bool, repoFull string) FindingContext {
	f.ConnectedRepo = connected
	f.RepoFullName = repoFull
	return f
}
