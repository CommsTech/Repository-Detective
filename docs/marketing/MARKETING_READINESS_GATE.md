# Marketing readiness gate

**Do not start outbound marketing until all required criteria pass.**

Last updated: 2026-06-12 (external beta stabilization)

## Required before marketing

| # | Criterion | Status |
|---|-----------|--------|
| 1 | Live RC deployed | **ready** (`rc-381667a`) |
| 2 | Product dogfood clean (0 active-present) | **ready** |
| 3 | Findings detail actionable live | **ready** |
| 4 | SBOM UI routes live | **ready** |
| 5 | SBOM artifact download proven | **ready** |
| 6 | Gitea issue target correctness | **ready** (scratch-repo live proof) |
| 7 | GitHub provider honest | **ready** (not release-proven) |
| 8 | UI route crawl | **ready** |
| 9 | Container logs clean | **ready** |
| 10 | 2 non-product beta scans | **ready** (report-only) |
| 11 | Gitea issue templates | **ready** (API verified) |
| 12 | Full `go test ./...` | **ready** (SBOM fix 2026-06-12) |
| 13 | Structured issue body live | **ready** (scratch issue #2 proof) |
| 14 | Wiki populated | **blocked** (HTTP 500) |
| 15 | Screenshots | **ready** (12 pages) |
| 16 | External clean install | **partial** |
| 17 | Store tests stable | **ready** |

## Decision

| Level | Status |
|-------|--------|
| **Marketing ready** | **NO** |
| **Private beta ready** | **YES** |
| **Controlled demo ready** | **YES** |
| **Private beta expansion** | **YES** — small invited cohort; report-only first |

## Private beta expansion (2026-06-11)

- Invited operators may onboard per `docs/beta/PRIVATE_BETA_RC_RELEASE_NOTES.md`
- Testers start with **report-only** scans; one repo initially
- Filing, runner, AI, container scan, Remediation PR stay disabled unless operator approves
- Operator rehearsal recorded; **external tester #1 handoff ready**
- Marketing waits on: wiki + external VM install + named external tester feedback

## Blockers

- Gitea wiki HTTP 500
- Full external VM clean install
- External named tester feedback (operator rehearsal complete)

## Explicit non-goals

- All-repo scanning
- Marketing launch
- AI recommendations enabled by default
- Remediation PR / runner delegation by default
