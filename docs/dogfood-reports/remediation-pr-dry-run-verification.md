# Remediation PR dry-run verification

Recorded: 2026-06-09

## Goal

Validate plan → approval → verification gate awareness **without** creating a PR.

## Configuration (verified)

| Key | Value |
|-----|-------|
| `remediation_pr_enabled` | `false` |
| `remediation_pr_require_approval` | `true` (default) |
| `remediation_pr_require_tests` | `true` (default) |

Live `/health` confirms `remediation_pr_enabled: false`.

## Flow verification

| Step | Expected | Observed |
|------|----------|----------|
| Remediation planner enabled | yes | `remediation_planner_enabled: true` |
| Operator approves plan | required when PR enabled | N/A while disabled |
| Verification job types | native runner supports `remediation_verify` | worker capabilities include it |
| PR creation | blocked when disabled | API/UI paths require `remediation_pr_enabled` |
| Pre-install audits | never create PRs | unchanged report-only |

## API gate

`POST /api/v1/remediation/:plan_id/attempt-pr` is not mounted for PR service when disabled (returns route not found / service unavailable). UI `AttemptFindingRemediationPR` returns "remediation PR feature disabled".

## Expected operator message when enabling later

When `remediation_pr_enabled: true` but tests fail or approval missing:

- "not eligible: …" from `patcher.CheckPREligibility`
- No branch pushed until gates pass

## PR created

**No** — dry-run only.

## Learning events

No PR lifecycle events recorded during this window (no `remediation_pr_opened` without successful gated attempt).
