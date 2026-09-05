# Remediation Planner (Repository Detective)

Repository Detective — **Inspect. Analyze. Improve.**

The remediation planner produces **structured fix plans only**. It does not generate patches, create branches, open PRs, rotate secrets, or run dependency-changing commands.

## Purpose

For eligible findings, answer:

- Can this finding be safely fixed?
- What would the fix require?
- What tests prove it?
- What is the regression risk?
- Is it safe for a future automated PR?

## Enable

```yaml
remediation_planner_enabled: true
remediation_min_severity: medium
remediation_min_confidence: 0.80
remediation_use_ai: false
remediation_comment_on_issue: false
```

Environment variables (prefer `REPOSITORY_DETECTIVE_*`; legacy `REPOSITORY_DETECTIVE_*` via envcompat):

```text
REPOSITORY_DETECTIVE_REMEDIATION_PLANNER_ENABLED
REPOSITORY_DETECTIVE_REMEDIATION_MIN_SEVERITY
REPOSITORY_DETECTIVE_REMEDIATION_MIN_CONFIDENCE
REPOSITORY_DETECTIVE_REMEDIATION_USE_AI
REPOSITORY_DETECTIVE_REMEDIATION_COMMENT_ON_ISSUE
```

## Deterministic-first recipes

| Finding type | Behavior |
|--------------|----------|
| Secrets (gitleaks) | Human review; no auto-PR; rotate out-of-band |
| Dependencies (govulncheck/grype/trivy) | Version bump + lockfile tests; major upgrades need human approval |
| gosec | Human review; regression tests suggested |
| staticcheck | Small fixes may be future auto-PR candidates |
| hadolint / checkov | Dockerfile/IaC validation commands suggested |
| test_gap / graph | Review and test creation; no auto-PR by default |

## AI policy

When `remediation_use_ai: true` and global AI is configured:

- AI enriches plans only for findings without deterministic recipes
- Output is labeled **advisory**
- `safe_for_auto_pr` remains false for AI-enriched plans in this phase

When AI is disabled, deterministic recipes and conservative generic plans still work.

## Plan fields

Each plan includes: fix strategy, affected files, required tests, validation commands (suggested only — **not executed**), regression risk, complexity, `safe_for_auto_pr`, `requires_human_review`, and blocked reasons.

## Triggers

**Connected repos:** after issue create/update (when finding passes gates), or on demand via API/UI.

**Pre-install audits:** plan-like guidance embedded in disclosure report markdown (no PR path).

Default eligibility: severity medium+ and confidence ≥ 0.80. High-confidence low-severity findings may qualify for simple dependency/container/staticcheck categories.

## API

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/findings/:id/remediation` | Latest plan for finding |
| POST | `/api/v1/findings/:id/remediation/generate` | Generate/regenerate plan |
| GET | `/api/v1/remediation/:plan_id` | Plan by ID |
| POST | `/api/v1/remediation/:plan_id/approve` | Approve (status only) |
| POST | `/api/v1/remediation/:plan_id/reject` | Reject (status only) |

Approve/reject updates plan status only — **no PR is created**.

## UI

Finding detail page shows the remediation section with planning-only warning, plan details, and generate/approve/reject actions.

Dashboard shows remediation candidate counts when planner is enabled.

## Issue comments

When `remediation_comment_on_issue: true`, a concise plan summary is posted to the linked Gitea issue. Default is **false** to avoid noise.

## Safety rules

Plans never recommend committing rotated secrets, public disclosure of secret values, broad rewrites without tests, direct pushes to protected branches, deleting orphaned graph code without review, or exploit instructions as validation steps.

## Future phase

Approved, low-risk plans feed the **Safe Remediation PR** phase when `remediation_pr_enabled: true`. **Evidence-based closure** runs when `evidence_closure_enabled: true`. See [REMEDIATION_PRS.md](REMEDIATION_PRS.md) and [EVIDENCE_CLOSURE.md](EVIDENCE_CLOSURE.md).

## Rollback

1. Set `remediation_planner_enabled: false` and restart.
2. Existing `remediation_plans` rows are harmless historical data.
3. No schema downgrade required (migrations v11+ are additive).
