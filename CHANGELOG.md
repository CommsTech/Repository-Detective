# Changelog

All notable changes to Repository Detective (formerly an internal prototype) are documented here.

## [Unreleased]

### Fixed
- **Gitea #355–#357** — patcher writes use `scanners.WriteWorkspaceBytes` with workspace containment checks (G703 path traversal)
- **Gitea #358** — pre-install audit goroutine inherits detached request context via `context.WithoutCancel`
- Finding detail suppression forms include CSRF tokens (UI verification finding)
- Operator smoke test no longer SIGPIPEs under `pipefail` when truncating JSON output

### Added
- `scanners.WriteWorkspaceBytes` + regression tests
- `/ui/learning` in UI route smoke tests
- `scripts/ui-full-verification.sh` Playwright full-object audit harness

## [v0.1.0-beta.2] — 2026-09-02

### Fixed
- **Secret scanning** — gitleaks config path resolved against process cwd; clean scans no longer recorded as `parse_failed`; private-repo history clones authenticate with the forge token
- **Learning / calibration** — false positives train through `mark-false-positive` + backfill; auto-apply guarded by rule ID (never downgrades SEC-/CVE/secret rules); poisoned pre-fix `scanner_failed` events purged on recompute
- **Operator defaults** — `.env.example` enables the full scanner fleet; LLM auditors off; `min_issue_confidence` 0.75; startup checks on by default

### Added
- Dashboard scan trend chart: remediation plans + auto-remediated findings series
- Gitea-first container publish (`git.commsnet.org/commstech/repository-detective`) with optional GHCR mirror
- `calibration/matcher_test.go` and expanded gitleaks/history-clone regression tests
- Calibration repo pagination (all repos, not only first 50)

### Changed
- Docker image `EXPOSE` documents port **8081** (homelab default)
- `config.yaml.example`: `scan_profile: standard`, calibration section documented

## [Unreleased] — Public community beta

### Added
- Root `LICENSE` (AGPL-3.0) + `NOTICE`, `CONTRIBUTING.md`, `SECURITY.md`
- Public beta guide (`docs/PUBLIC_BETA.md`) and GitHub issue templates
- GitHub mirror sync via deploy key (`scripts/sync-gitea-to-github.sh --github`)

### Changed
- README framed as **public community beta** (Gitea canonical, GitHub mirror)
- Learning Accept installs `report_only` calibration rules only (no rule-wide suppressions)

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
