# Data retention

Repository Detective does **not** enforce automatic data retention policies. Administrators define how long data is kept.

## What is stored

| Data | Location | Typical retention driver |
|------|----------|-------------------------|
| Findings, scans, instances | SQLite (`database_path`) | Operator policy |
| Workspace clones | Configured work directory | Until scan cleanup / disk policy |
| Gitea issues | Gitea (external) | Forge policy |
| Logs | Container/host logs | Log aggregator policy |

## Recommended practices

1. **Backup before delete** — `sqlite3 … ".backup …"` per [DATABASE.md](DATABASE.md)
2. **Define max age** — e.g. delete findings `closed` &gt; 365 days (manual SQL or future feature)
3. **Workspace cleanup** — ensure failed scans do not fill disk
4. **Export before purge** — use reports/API for audit archives (treat exports as sensitive)

## GDPR-style erasure (administrator responsibility)

To honor a deletion request for a subject whose data appeared in findings:

1. Identify findings/repos containing PII (search UI or SQL)
2. Delete or anonymize rows in `findings` / `finding_instances`
3. Close or redact related Gitea issues manually
4. Remove backups that still contain the data

The product does not provide a one-click “erase subject” API.

## PHI / regulated data

Do not scan PHI systems without legal review. If scanning occurred by mistake, stop the service, delete DB rows and workspaces, and rotate tokens.

## Configuration hooks

- Disable features that increase retention surface: notifications to third parties, LLM
- `evidence_closure_*` — controls issue lifecycle, not DB purge

Future work: configurable retention job (see prepared issue backlog).
