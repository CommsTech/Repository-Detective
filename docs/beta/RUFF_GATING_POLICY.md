# Ruff gating policy (beta)

## Problem

Ruff can emit hundreds of style/import findings on Python homelab repos. All findings remain visible in reports; gating controls severity and issue eligibility.

## Rules

| Ruff prefix | Homelab/infra profile | Other repos |
|-------------|----------------------|-------------|
| S*, B* (security) | Actionable — keep severity | Actionable |
| F821/F822 (undefined) | Medium | Medium/high |
| I*, W*, E501, COM*, Q*, UP* | **info** (report-only) | medium (default) |
| Other F*, E* | info unless security-adjacent | scanner default |

## Scope

- Applied in `profile.CalibrateRuffResults` during scan — **per repo profile**, not global.
- Repo A homelab calibration does not affect repo B.
- Report-only dry runs: findings visible, **0 issues filed** (unchanged policy).

## Issue filing (when approved later)

Only findings above severity/confidence gates and not downgraded to `info` are eligible for Gitea issues.

## Tests

- `profile/ruff_test.go` — style downgraded, security preserved, repo isolation
