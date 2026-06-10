# Issue provider matrix

| Provider | Status | Notes |
|----------|--------|-------|
| **Gitea** | supported | Tested for product dogfood; repo mapping must match source repo |
| **GitHub** | implemented, RC not fully proven | `ForgeType: github` in issue manager; requires token and testing per org |
| **GitLab** | not_implemented | No issue forge adapter yet |

## Policy

- Pre-install audits: **never** auto-file
- Dry-run: **never** auto-file
- Container scans: issue filing **disabled by default**
- Normal connected repo scans: file/update per `auto_create_issues` and reporting policy
- Issue target must match source repo provider/owner/repo — **no cross-repo filing**

See `docs/beta/ISSUE_FILING_POLICY.md` for operator guidance.
