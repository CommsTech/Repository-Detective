# Release candidate acceptance baseline

**Recorded:** 2026-06-02  
**Commit:** `dab8a13` (pre-RC sprint; subsequent RC commits follow on `main`)

## Live revision

| Item | Value |
|------|-------|
| Health | healthy |
| Runner delegation | disabled |
| Remediation PR | disabled |
| AI recommendations | disabled (`max_tokens_per_scan: 0`) |
| Container scanning | opt-in / rolled back post-demo |
| Product active-present (repo 1) | 2 |
| High/critical product findings | 0 (reconciliation) |

## Feature readiness

| Area | Status |
|------|--------|
| Deterministic repo scan | proven (dogfood) |
| Container image scan | proven (`alpine:3.20`, job `rj-fa8317b9a9c7b191`) |
| AI recommendations | implemented; provider-neutral rename in RC sprint |
| SBOM | store + runner; UI routes added in RC sprint |
| Gitea issue filing | partially proven |
| GitHub issue filing | not RC-proven |
| GitLab issue filing | not implemented |
| Wiki | blocked (Gitea HTTP 500) |
| External clean install | incomplete |
| Marketing | **NOT READY** |

## Known blockers

1. Findings detail actionability — RC sprint target
2. Full UI route crawl — RC sprint
3. Container log health from latest deploy — RC sprint
4. SBOM end-to-end UI verification — RC sprint
5. Gitea wiki populate — server-side 500
6. OpenClaw/provider non-JSON response — expected until provider prompt aligned

## Marketing readiness

**NOT READY** — private beta / demo credible after RC acceptance sprint completes green except intentionally disabled features.
