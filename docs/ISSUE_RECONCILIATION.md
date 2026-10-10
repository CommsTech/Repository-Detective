# Issue reconciliation

Repository Detective inspects **already-filed Gitea issues** it created and reconciles them against current scan data.

## Principles

- **Never delete** issues
- **Never close** without evidence (unless `issue_reconciliation_close_verified: true` and fingerprint absent + scanner ran)
- **Never close** just because an issue is old
- **Never close** solely because a later **partial** (PR/diff/changed-files) scan did not observe the fingerprint — see [FINDING_RESOLUTION_SEMANTICS.md](FINDING_RESOLUTION_SEMANTICS.md)
- Label, comment, enrich, suppress — preserve audit history

## Detected states

| Status | Meaning |
|--------|---------|
| `still_present` | Fingerprint in latest scan |
| `already_fixed_verify` | Absent from latest scan; verify before close |
| `duplicate` | Same fingerprint on multiple issues |
| `false_positive` | Finding marked false positive locally |
| `suppressed` | Active suppression rule matches |
| `stale_rule` | Rule is report-only under current profile (e.g. graph noise) |
| `scanner_not_run` | Original scanner did not run — cannot verify fix |
| `needs_enrichment` | Issue lacks current context |
| `needs_human_review` | Critical/high still present or ambiguous |

## API

```text
GET  /api/v1/repos/:id/reconcile-issues/preview
POST /api/v1/repos/:id/reconcile-issues
GET  /api/v1/issues/reconciliation/:run_id
```

## UI

Repository detail → **Reconcile existing issues** → preview table → **Apply reconciliation**.

## Config

```yaml
issue_reconciliation_enabled: true
issue_reconciliation_comment: true
issue_reconciliation_close_verified: false   # default
issue_reconciliation_max_comments_per_issue: 3
```
