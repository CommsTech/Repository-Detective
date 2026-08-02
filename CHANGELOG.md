# Changelog

All notable changes to Repository Detective (formerly an internal prototype) are documented here.

## [Unreleased] — Closeout sprint

### Added

- Gitea issue templates under `.gitea/ISSUE_TEMPLATE/` (bug, feature, compliance, accessibility, scanner FP, security triage)
- Issue tracking guide: `docs/ISSUE_TRACKING.md`, prepared backlog `docs/issues/`
- `scripts/gitea-backlog-setup.sh` for labels/milestones (token required)
- Accessibility: skip link, `:focus-visible`, reduced motion, chart text summary on dashboard
- Privacy: `internal/security/redact.go`, scanner log helper `scanners/log_redact.go`
- Evidence sanitization on DB store via `issues.SanitizeSecretEvidence` in `redactSnippet`
- Docs: `ACCESSIBILITY.md`, `ADMIN_HARDENING.md`, `DATA_RETENTION.md`, `SECURITY_MODEL.md`, `RELEASE_READINESS.md`, `COMPLIANCE_READINESS.md`, `KNOWN_LIMITATIONS.md`
- `scripts/dogfood-self-scan.sh` for self-scan workflow

### Changed

- Dashboard: WCAG-oriented chart text summary; scanner coverage table (from prior audit pass)
- Scanner health UX: degraded vs optional/inactive (from prior audit pass)

### Documentation

- Privacy, scanner health, dashboard, wiki publishing guides (prior pass + closeout index)

### Known

- Wiki prepared under `docs/wiki/` — manual push only
- Gitea backlog issues not auto-created (markdown prepared)
- Full WCAG/508/HIPAA/GDPR compliance **not** claimed

## Earlier releases

See git history and [ISSUE_BACKLOG.md](docs/ISSUE_BACKLOG.md) for shipped features (#42 dedup, #44 radar chart, reporting modes, remediation loop, etc.).
