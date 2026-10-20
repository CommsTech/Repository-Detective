# Gitea issue backlog (commstech/Repository-Detective)

Tracks feature and bug issues on Gitea vs implementation status. Update when closing or shipping work.

| Issue | Title | Status | Notes |
|-------|-------|--------|-------|
| #38 | Possible command execution (HIGH) | **Closed** | False positive — `analyzers/static.go` excluded from self-scan |
| #39 | Overall scoring and checks improvement | **Closed** | `ComputeOverallScore`, `docs/SECURITY_CHECK_MATRIX.md` |
| #40 | Hardening recommendations | **Closed** | `deterministic_test.go` hardening applied |
| #41 | Additional tools | **Closed** | SBOM + roadmap in `docs/SBOM.md` |
| #42 | Dedup merges different categories | **Closed** | Dedup key includes category |
| #43 | Optimization checks | **Closed** | Advisory `OPT-*` rules + `docs/OPTIMIZATION_CHECKS.md` |
| #44 | Radar chart | **Closed** | Category radar on dashboard |
| #45 | History and repo checks | **Closed** | `docs/PRE_PUBLISH_CHECKS.md` + static/gitleaks |
| #46 | Repo Actions/Runners checks | **Closed** | `docs/PIPELINE_GOVERNANCE.md` + `GOV-*` rules |
| #47 | Review Doc Detective | **Closed** | `docs/DOC_DETECTIVE_REVIEW.md` |

## Closing issues on Gitea

```bash
./scripts/close-gitea-issues.sh   # requires GITEA_TOKEN in .env
```

Or comment manually with commit SHA via API — see [ISSUE_TRACKING.md](ISSUE_TRACKING.md).
