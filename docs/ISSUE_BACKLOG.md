# Gitea issue backlog (commstech/Bugbot)

Tracks feature and bug issues on Gitea vs implementation status. Update when closing or shipping work.

| Issue | Title | Status | Notes |
|-------|-------|--------|-------|
| #38 | Possible command execution (HIGH) | Open | Scanner finding on repo code — triage separately from feature backlog |
| #39 | Overall scoring and checks improvement | Open | Not implemented |
| #40 | Hardening recommendations | Partial | [SECURITY_HARDENING.md](SECURITY_HARDENING.md); deterministic test hardening ongoing |
| #41 | Additional tools | Partial | govulncheck, gosec, staticcheck, optional Trivy/Grype/Gitleaks in image — see [SBOM.md](SBOM.md) |
| #42 | Dedup merges different categories | **Closed** | Dedup key includes category (`analyzers/engine.go`); `TestDedupKeepsSeparateCategoriesSameLine` |
| #43 | Optimization checks | Open | Not implemented |
| #44 | Radar chart | **Closed** | Category radar on dashboard (`dashboard-charts.js`, Chart.js 4.4.1) |
| #45 | History and repo checks | Partial | Scan history UI, health checks — not full “history + repo checks” spec |
| #46 | Repo Actions/Runners checks | Partial | [RUNNERS.md](RUNNERS.md), runner delegation — not full Actions audit |
| #47 | Review Doc Detective | Open | Not implemented |

## Shipped without dedicated issues

- Repo structure profiling and reporting modes — [REPORTING.md](REPORTING.md), `profile/` package
- Dashboard charts and public static assets — [UI.md](UI.md)
- Gitleaks/gosec JSON extraction — `scanners/json_extract.go`

## Closing issues on Gitea

When work ships on `main`, comment with commit SHA and close via UI or API:

```bash
# Example (requires BUGBOT_GITEA_TOKEN in .env)
curl -X POST -H "Authorization: token $BUGBOT_GITEA_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"body":"Fixed in <sha>: ..."}' \
  "https://git.commsnet.org/api/v1/repos/commstech/Bugbot/issues/42/comments"
```
