# Marketing readiness gate

**Do not start outbound marketing until all required criteria pass.**

Last updated: 2026-06-11 (blocker burn-down sprint)

## Required before marketing

| # | Criterion | Status |
|---|-----------|--------|
| 1 | Live RC deployed | **ready** (`rc-e3e19ec`) |
| 2 | Product dogfood clean (0 active-present) | **ready** (scan `926a5f56a26f03c9`) |
| 3 | Findings detail actionable live | **ready** |
| 4 | SBOM UI routes live | **ready** |
| 5 | SBOM artifact download proven | **ready** (Syft CycloneDX proof) |
| 6 | Gitea issue target correctness | **partial** (dry-run + unit tests) |
| 7 | GitHub provider honest | **ready** (not release-proven) |
| 8 | UI route crawl | **ready** |
| 9 | Container logs clean | **ready** |
| 10 | 2 non-product beta scans | **ready** (report-only) |
| 11 | Wiki populated | **blocked** (HTTP 500) |
| 12 | Screenshots | **ready** (12 pages) |
| 13 | External clean install | **partial** (beta package; full VM pending) |

## Decision

| Level | Status |
|-------|--------|
| **Marketing ready** | **NO** |
| **Private beta ready** | **YES** |
| **Controlled demo ready** | **YES** |

## Blockers

- Gitea wiki HTTP 500
- Live Gitea issue filing regression (optional before wider beta)
- Full external VM clean install
- GitHub issue provider live proof (demoted to not release-proven)

## Explicit non-goals

- All-repo scanning
- Marketing launch
- AI recommendations enabled by default
- Remediation PR / runner delegation by default
