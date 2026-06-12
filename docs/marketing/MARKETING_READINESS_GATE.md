# Marketing readiness gate

**Do not start outbound marketing until all required criteria pass.**

Last updated: 2026-06-12 (first tester cohort scan rehearsal)

## Required before marketing

| # | Criterion | Status |
|---|-----------|--------|
| 1 | Live RC deployed | **ready** (`rc-e3e19ec`) |
| 2 | Product dogfood clean (0 active-present) | **ready** (scan `926a5f56a26f03c9`) |
| 3 | Findings detail actionable live | **ready** |
| 4 | SBOM UI routes live | **ready** |
| 5 | SBOM artifact download proven | **ready** (Syft CycloneDX proof) |
| 6 | Gitea issue target correctness | **ready** (scratch-repo live proof 2026-06-12) |
| 7 | GitHub provider honest | **ready** (not release-proven) |
| 8 | UI route crawl | **ready** |
| 9 | Container logs clean | **ready** |
| 10 | 2 non-product beta scans | **ready** (report-only) |
| 11 | Gitea issue templates | **ready** (API verified) |
| 12 | Wiki populated | **blocked** (HTTP 500) |
| 13 | Screenshots | **ready** (12 pages) |
| 14 | External clean install | **partial** (beta package; full VM pending) |
| 15 | Store tests stable | **ready** (triage report) |

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
- First tester scan rehearsal PASS (`512145e55d4488ea`); operator feedback recorded; external tester pending
- Marketing waits on: wiki + external VM install + external named tester

## Blockers

- Gitea wiki HTTP 500
- Full external VM clean install
- External named tester feedback (operator rehearsal complete)

## Explicit non-goals

- All-repo scanning
- Marketing launch
- AI recommendations enabled by default
- Remediation PR / runner delegation by default
