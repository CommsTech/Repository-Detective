# Beta scan policy

Private beta testers receive **report-only by default** via `config/private-beta.example.yaml`:

```yaml
auto_create_issues: false
```

This maps to `issue_policy: off` and scan policy mode `private_beta_safe`.

## First scan (testers)

Use report-only (checkbox checked or API `"report_only_dry_run": true`) until the operator approves issue filing for a repository.

## Moving to controlled issue filing

1. Set `auto_create_issues: true` in config (or per-repo `issue_policy: all` / `fingerprint`)
2. Restart Repository Detective
3. Confirm Configure → **Issue filing policy** shows `production_self_hosted`
4. Run a manual scan with report-only **unchecked** on a controlled test repo first
5. Keep backlog-control enabled during burn-down

## Pre-install audits (beta)

Pre-install mode stays report-only for all deployments — promotional trust builder, not issue filing.

## What does not change in beta

- Remediation PRs remain off by default
- LLM sanity gate remains off by default
- Qdrant removed — issue dedup is fingerprint + SQLite forge mappings only

See also [SCAN_POLICY.md](../SCAN_POLICY.md).
