# Known limitations

Honest constraints as of the closeout sprint. Update when shipping fixes.

## Product

- Single-tenant operator model — no built-in multi-user RBAC in UI
- SQLite default — not HA clustered DB
- Semantic dedup (Qdrant) optional and external
- Remediation PRs off by default

## Scanners

- Optional binaries may be missing — **degraded coverage**, not hard failure
- Version strings may be unparsed (`unknown`) or vendor-specific (`dev`)
- Vendor directories may produce noise — use `suppress_vendor` in profile
- Scanner stderr may contain sensitive paths — log redaction is heuristic and incomplete on some scanners

## Privacy

- Redaction patterns do not catch all secret formats
- Gitea issues already created are not retroactively redacted
- LLM prompts may include code if enabled — administrator must disable
- Reports/exports may contain sensitive snippets — treat as confidential

## Accessibility

- No VPAT; manual testing checklist only
- Charts require supplementary text tables (provided on dashboard)
- No keyboard shortcut layer

## Documentation / wiki

- Wiki not auto-synced to Gitea wiki remote
- Some legacy docs reference “Bugbot” paths

## Integrations

- Gitea-specific; GitHub/GitLab not first-class
- Runner delegation requires separate runner deployment

## Tracking

See [issues/README.md](issues/README.md) and [ISSUE_BACKLOG.md](ISSUE_BACKLOG.md).
