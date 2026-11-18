# Issue provider matrix

| Provider | Status | Notes |
|----------|--------|-------|
| **Gitea** | supported | Dry-run proven 2026-06-10; repo-mapping RC regression pending |
| **GitHub** | implemented, not release-proven | Unit test passes; live org token 401; do not claim production-ready |
| **GitLab** | not_implemented | No issue forge adapter |

## Policy

- Pre-install audits: **never** auto-file
- Dry-run: **never** auto-file
- Container scans: issue filing **disabled by default**
- Normal connected repo scans: file/update per `auto_create_issues` and reporting policy
- Issue target must match source repo provider/owner/repo — **no cross-repo filing**

See `docs/beta/ISSUE_FILING_POLICY.md` for operator guidance.
