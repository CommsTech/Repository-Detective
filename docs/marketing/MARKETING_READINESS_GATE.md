# Marketing readiness gate

**Do not start outbound marketing until all required criteria pass.**

Last updated: 2026-06-12 (calibration sprint)

## Required before marketing

| # | Criterion | Status |
|---|-----------|--------|
| 1 | Live RC deployed | **ready** |
| 2 | Product dogfood clean (0 active-present) | **ready** |
| 3 | Findings detail actionable live | **ready** |
| 4 | SBOM UI routes live | **ready** |
| 5 | SBOM artifact download proven | **ready** |
| 6 | Gitea issue target correctness | **ready** |
| 7 | GitHub provider honest | **ready** |
| 8 | UI route crawl | **ready** |
| 9 | Container logs clean | **ready** |
| 10 | 2 non-product beta scans | **ready** |
| 11 | Gitea issue templates | **ready** |
| 12 | Full `go test ./...` | **ready** |
| 13 | Structured issue body live | **ready** |
| 14 | Wiki populated | **blocked** (HTTP 500) |
| 15 | Screenshots | **ready** |
| 16 | External clean install | **partial** |
| 17 | Store tests stable | **ready** |
| 18 | ≥2 external testers clean | **partial** (1 of 2 calibrated) |
| 19 | Beta calibration (high FP) | **ready** — rescan 0 high |

## Decision

| Level | Status |
|-------|--------|
| **Marketing ready** | **NO** |
| **Private beta ready** | **YES** |
| **Controlled demo ready** | **YES** |
| **Private beta expansion** | **YES** — proceed to tester #2 |

## External tester #1 calibration

- Before: 123 findings, 1 high (`SEC-HARDCODED-SECRET` FP)
- After: 120 findings, **0 high**, 1 actionable (`HEALTH-TECH-MARKER`)
- Report-only: 0 issues, 0 PRs

## Blockers

- Gitea wiki HTTP 500
- Full external VM clean install
- External tester #2 feedback
- Optional: logged-in template screenshot

## Explicit non-goals

- All-repo scanning
- Marketing launch
- AI / runner / Remediation PR by default
