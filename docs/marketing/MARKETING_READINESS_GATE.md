# Marketing readiness gate

**Do not start outbound marketing until all required criteria pass.**

Last updated: 2026-06-12 (after external tester #1)

## Required before marketing

| # | Criterion | Status |
|---|-----------|--------|
| 1 | Live RC deployed | **ready** (`rc-381667a`) |
| 2 | Product dogfood clean (0 active-present) | **ready** |
| 3 | Findings detail actionable live | **ready** |
| 4 | SBOM UI routes live | **ready** |
| 5 | SBOM artifact download proven | **ready** |
| 6 | Gitea issue target correctness | **ready** |
| 7 | GitHub provider honest | **ready** (not release-proven) |
| 8 | UI route crawl | **ready** |
| 9 | Container logs clean | **ready** |
| 10 | 2 non-product beta scans | **ready** |
| 11 | Gitea issue templates | **ready** (API verified) |
| 12 | Full `go test ./...` | **ready** |
| 13 | Structured issue body live | **ready** |
| 14 | Wiki populated | **blocked** (HTTP 500) |
| 15 | Screenshots | **ready** (12 pages) |
| 16 | External clean install | **partial** |
| 17 | Store tests stable | **ready** |
| 18 | ≥2 external testers clean | **partial** (1 of 2) |

## Decision

| Level | Status |
|-------|--------|
| **Marketing ready** | **NO** |
| **Private beta ready** | **YES** |
| **Controlled demo ready** | **YES** |
| **Private beta expansion** | **PAUSED for calibration** — then tester #2 |

## External tester #1 outcome (2026-06-12)

- Tester: `ext-operator-jrice`
- Repo: `commstech/Wifi_Collector`
- Scan: `85a8ab62e76da076` (report-only, 0 issues, 0 PRs)
- Feedback: received; false-positive noise elevated
- **Action:** calibration sprint before tester #2 (not a marketing blocker alone)

## Blockers

- Gitea wiki HTTP 500
- Full external VM clean install
- Second external tester with clean feedback
- Optional: logged-in template screenshot

## Explicit non-goals

- All-repo scanning
- Marketing launch
- AI recommendations enabled by default
- Remediation PR / runner delegation by default
