# Repository settings UX

## Goal

Testers can understand what each policy does without reading source code.

## Layout

`/ui/repos/:id/settings` now includes:

1. **Guided sections** (read-only reference) — Overview, Issue filing, Report-only / safety, Scanners, Scheduling
2. **Edit form** (existing) — change overrides below the reference tables

Each reference row shows:

| Column | Meaning |
|--------|---------|
| Setting | Human label |
| Effective | Merged value for this repo |
| Source | `profile/default` or `repo override` |
| Notes | Plain-language explanation + beta note |

## Safety badges

| Badge | Meaning |
|-------|---------|
| Safe beta default | No forge side effects |
| Can create issues | Dangerous — filing enabled |
| Scanner | Scanner toggle |
| Advanced | Scheduling / cron |

## Critical settings called out

- `issue_policy` — off / fingerprint / all
- `policy_level` — monitor_only vs issue filing modes
- Report-only dry-run — per-scan API/UI flag
- `remediation_policy` — does not open PRs by itself
- Scanner toggles — effective after profile merge

## Issue filing disabled banner

When effective settings block filing, a banner explains that forge issue counts may be lower than scan findings.
