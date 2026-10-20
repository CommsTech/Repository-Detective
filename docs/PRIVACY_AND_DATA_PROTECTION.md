# Privacy and data protection

Repository Detective is designed for **privacy-aware** operation in homelab and team environments. It is **not** certified HIPAA-compliant or GDPR-compliant out of the box. Administrators must configure access controls, retention, network boundaries, and legal basis for processing.

## What the public Gitea repo contains (and does not)

The published source tree is a **clean install base** for any operator. It includes application code, Compose files, examples, and setup docs.

| Included | Not included (local only — gitignored) |
|----------|----------------------------------------|
| Source, templates, scripts, `.env.example` | Your `.env` (API keys, forge tokens, AI keys) |
| `config/*.example.yaml`, scanner configs | Your `config/config.yaml` |
| Docs and issue templates | Your SQLite DB (`data/repository-detective.db`) |
| | Cloned scan workspaces, runner artifacts, local dogfood dumps |

**Never commit** a live database or `.env`. The DB holds private repo names, findings, code snippets, and forge mappings from *your* fleet — publishing it would expose that data to anyone who can clone the Gitea repo.

Fresh installs create an empty local DB on first start. See [SETUP.md](SETUP.md) and [CONFIGURATION.md](CONFIGURATION.md).

## Data collected

| Source | Examples |
|--------|----------|
| Connected repositories | File paths, code snippets in findings, dependency metadata, scan summaries |
| Scanner output | Rule IDs, severities, line numbers, matched lines (may include secrets if present in code) |
| Gitea webhooks | Repo names, commit SHAs, PR metadata |
| Operator actions | Triage status, remediation plans, approval timestamps |
| Optional AI/LLM stages | Prompt context derived from code and findings (when enabled) |

## Data stored

- **SQLite database** (`database_path`): findings, instances, scans, scanner results, remediation plans, lifecycle events, settings, forge issue mappings.
- **Workspace directories**: cloned repository trees during scans (ephemeral; retention depends on deployment).

## Data displayed

- Operator UI (`/ui/*`): finding titles, redacted evidence, repo names, scan status.
- Reports: aggregated counts and per-repo summaries.
- API (`/api/v1/*`): JSON for dashboards and integrations; protect with API keys and network policy.

Evidence in the UI is labeled **redacted**; snippets pass through `SanitizeSecretEvidence` before storage/display where the pipeline applies it. Raw scanner output may still exist in DB fields until reprocessed — treat the DB as sensitive.

## Data sent to external systems

| Destination | When | Content |
|-------------|------|---------|
| **Gitea** | Issues, PRs, commit status, webhooks | Issue bodies use sanitized evidence; templates redact common secret patterns |
| **AI providers** | `enable_llm_auditors` or similar | Code/findings context per provider config — **disable** for strict air-gap |
| **Notification webhooks** | `notifications_enabled` | Redacted titles/summaries via [NOTIFICATIONS.md](NOTIFICATIONS.md) |
| **Runners** | Delegated scans | Scan job payload; runner must be trusted |

Private repository content is **not** sent externally except to configured Gitea, AI, notification, and runner endpoints.

## AI / LLM data handling

- Controlled by `enable_llm_auditors` and provider settings ([AI_PROVIDERS.md](AI_PROVIDERS.md)).
- Deterministic scanners and health checks do not require an LLM.
- To disable: set `enable_llm_auditors: false` and leave AI API keys unset; restart the service.
- Review provider DPAs and data residency before processing regulated data.

## Gitea issue data handling

- Issue bodies are built from findings with `SanitizeSecretEvidence` on snippets ([issues/template.go](../issues/template.go)).
- Fingerprints use hashed redacted evidence ([issues/fingerprint.go](../issues/fingerprint.go)).
- Operators should use private Gitea projects and least-privilege tokens.

## Logs

- Application logs may include repository names, scan IDs, and error text.
- Scanner stderr is not logged verbatim at info level by default; avoid `debug` logging in production with sensitive repos.
- Do not log environment variables containing tokens.

## Secrets redaction

Patterns redacted in issues, notifications, and evidence hashing include API keys, tokens, `AKIA…`, and `Bearer …` prefixes. **Redaction is heuristic** — not guaranteed for all secret formats. Never rely on redaction alone for compliance.

## PHI / PII caution

Findings and logs may contain:

- Email addresses, names, account IDs in source or config files
- Hostnames, internal URLs, customer identifiers
- Health or financial data if present in scanned repositories

**Do not** point Repository Detective at PHI/PII systems without legal review, data minimization, access controls, and retention policies.

## GDPR-style rights and retention

Repository Detective does not implement data-subject portals. Administrators must:

- Define retention (DB backups, workspace cleanup)
- Honor erasure requests by deleting findings/repos in SQLite and related Gitea issues
- Document lawful basis and subprocessors (Gitea, AI, runners)

## Administrator hardening checklist

- [ ] Run UI/API behind TLS and authentication ([SECURITY_HARDENING.md](SECURITY_HARDENING.md))
- [ ] Store secrets in `.env` only; never commit tokens
- [ ] Restrict `gitea_token` scope to required repos
- [ ] Disable LLM for sensitive fleets
- [ ] Encrypt SQLite backups at rest
- [ ] Restrict who can access `/ui` and export reports
- [ ] Review notification webhook destinations

## Known limitations

- Redaction is pattern-based, not ML-based
- Legacy issue bodies in Gitea are not rewritten when redaction improves
- Scanner raw output in DB may predate sanitization on some code paths
- Community/federation features (if enabled) may replicate metadata — review config

## Data sent to scanners

Scanner subprocesses receive a **minimal environment** (PATH, HOME, etc.) without operator API keys. See [internal/security/env.go](../internal/security/env.go).

Scanner logs use heuristic redaction for `detail` fields where implemented (`internal/security/redact.go`).

## Reports and exports

Executive and per-repo reports aggregate finding metadata. Treat exports as **confidential**. Redaction in reports follows issue/preinstall sanitization paths; raw DB queries bypass UI redaction.

## Related docs

- [ADMIN_HARDENING.md](ADMIN_HARDENING.md)
- [DATA_RETENTION.md](DATA_RETENTION.md)
- [COMPLIANCE_READINESS.md](COMPLIANCE_READINESS.md)
- [SECURITY_HARDENING.md](SECURITY_HARDENING.md)
- [DATABASE.md](DATABASE.md)
