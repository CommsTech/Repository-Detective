# Marketing readiness gate

**Do not start outbound marketing until all required criteria pass.**

Last updated: 2026-06-10 (post live RC redeploy `rc-e3e19ec`)

## Required before marketing

| # | Criterion | Status |
|---|-----------|--------|
| 1 | Live RC deployed (`e3e19ec`+) | **ready** |
| 2 | Findings detail actionable live | **ready** (37361 verified) |
| 3 | SBOM UI routes live | **ready** (download pending artifact proof) |
| 4 | AI Recommendations provider-neutral | **ready** |
| 5 | UI route crawl passes | **ready** (15/15 HTTP 200) |
| 6 | Container logs clean | **ready** (0 panics) |
| 7 | Gitea wiki populated | **blocked** (HTTP 500) |
| 8 | External clean install proven | **not tested** |
| 9 | Screenshots current | **partial** |
| 10 | ≥2 non-product beta scans low-noise | **partial** (2 scans run; ansible noisy) |
| 11 | Product dogfood clean | **partial** (`active_present_open` 21 — investigate) |
| 12 | Gitea issue mapping RC-proven | **partial** |
| 13 | GitHub issue filing RC-proven | **not proven** |
| 14 | GitLab honestly marked unsupported | **ready** |

## Decision options

| Level | Status |
|-------|--------|
| Marketing ready | **NO** |
| Private beta ready | **YES** (with known caveats) |
| Controlled demo ready | **YES** (live RC + findings + SBOM UI) |

## Blockers

- Gitea wiki HTTP 500
- External clean install
- Product `active_present_open` regression investigation (2 → 21)
- GitHub live issue filing proof
- Screenshot/visual QA batch
- Ansible beta scan noise calibration

## Explicit non-goals (pre-marketing)

- All-repo scanning
- Remediation PR enabled by default
- Runner delegation enabled by default
- AI Recommendations enabled by default
- Auto-submit disclosures
