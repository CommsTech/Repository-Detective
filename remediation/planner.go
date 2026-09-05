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
		enriched, err := p.ai.SuggestPlan(fctx, plan)
		if err != nil {
			plan.Advisory = false
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
	_ = ctx
	return plan, nil
}

func needsAIEnrichment(ctx FindingContext) bool {
	category := strings.ToLower(ctx.Category)
	if category == "secret" || category == "dependency" || category == "architecture" {
		return false
	}
	source := strings.ToLower(ctx.Source)
	switch source {
	case "gitleaks", "govulncheck", "grype", "trivy", "gosec", "staticcheck", "hadolint", "checkov", "graph", "test_gap":
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
