# Changelog

All notable changes to Repository Detective (formerly an internal prototype) are documented here.

## [Unreleased — custom rules, runner safety and operator trust, 2026-11-17]

### Implemented (unreleased)
- Added non-executable per-repository `.repository-detective.rules.json` policies with strict JSON/RE2 validation, bounded resource usage, and deterministic finding integration on core and native worker. Added a no-code local JSON-rule builder in repository settings.
- Changed native worker file collection to fail on source-file count/size excess or unsafe symlinks instead of producing misleading clean results. Clone now rejects unsafe transports and credentials and checks the exact commit when provided.
- Removed automatic core fallback for failed delegation, refused unsupported Gitea Actions job claims, added direct dispatcher/backend rejection, and checked job identifiers before workspace deletion.
- Disabled unattended AI-only false-positive calibrations, protected custom rules against automatic suppression, and added regression tests.
- Added evidence-first dashboard guidance, an accessible custom-rule settings flow, additional dark-mode refresh protections, a hardened sample native-worker Compose overlay, and corrected misleading runner documentation. Gitea CI now checks UI JS syntax, a dependency-free rule-builder interaction contract and targeted runner/store/rule tests.

### Verification limitations
- Isolated custom-rule tests, JavaScript syntax, CSS parse, template parse, Go formatting and Compose YAML syntax passed. **Full module Go 1.26.10 tests, Docker, production-like browser, Gitea integration, sandbox penetration tests and verified AI PR workflows remain pending.** See [verification record](docs/release/RELEASE_VALIDATION_2026-11-17.md).

## [Unreleased — operator journey and findings search, 2026-11-16]

### Improved
- Redesigned Findings into clear Priority, All, and Resolved views with a searchable first row and optional advanced filters, while preserving server-side severity ordering, pagination, and exports.
- Added literal substring search by CVE/rule, package, title, path, fingerprint, and repository across the full SQLite inventory. Search is scoped to the selected view, capped at 120 characters and escapes SQL LIKE wildcards; count, page, and export use the same filter.
- Reordered finding detail into an investigation → remediation plan → closure evidence workflow, with direct anchors to underlying scanner evidence and a deliberate confirmation for plan regeneration.
- Added mobile-collapsible navigation (keyboard Escape + focus return, no-JS navigation fallback), improved empty states, and a zero-repository first-run checklist.
- Fixed explicit suppressed/false-positive status selections returning empty queues because suppressed records were filtered a second time. Clarified that accepted risk must not be misreported as a false positive.

### Verification limits
- Server template parse, JavaScript syntax, CSS parse, Go formatting, and patch checks passed. Added database and UI tests, but Go 1.26.10 unit/integration tests, real browser accessibility checks, live Gitea, and Docker E2E remain **NOT RUN** in this environment.
- This is a new **working-tree update**, not a change to the last published beta image or an assertion that unattended CVE remediation is ready.

## [Unreleased — operator triage accuracy, UX and documentation, 2026-11-16]

### Fixed
- Canonical findings now return one queue row even when the finding has multiple linked forge issues; the current open issue is preferred.
- SQL-backed severity ordering and open critical/high focus selection prevent old urgent vulnerabilities from disappearing behind the latest 200 medium/low findings.
- Findings queue now shows true match counts and supports filter-preserving pagination (50 focus, 100 general per page).
- Filtered CSV and JSON exports now traverse multiple DB pages up to a documented 20,000-row safety limit instead of silently truncating at 200 rows. CSV formula-like fields are neutralized; unknown export formats are rejected.
- Dependency details no longer presume easy or low-risk fixes from the mere presence of a scanner-suggested fixed version. Closure guidance explicitly requires redeploy/rescan evidence.

### Added
- Operator-specific, mobile-friendly focus queue help and table scrolling, plus queue and export regression tests.
- Five-minute operator triage runbook and queue verification receipt.
- Release-readiness documentation corrected to Go 1.26.10 and non-vendor builds.

### Verification limits
- JS syntax and diff checks passed; Go integration tests require Go 1.26.10 and are blocked here by unavailable toolchain download. Docker, Gitea and accessibility E2E remain unverified.

## [Unreleased — CVE triage and remediation workflow hardening, 2026-11-16]

### Added
- Opt-in, authenticated, on-demand CISA KEV + FIRST EPSS CVE enrichment with bounded source-specific caching, missing-data states, strict CVE validation, and isolated mock HTTP tests.
- Finding detail exploitation-intelligence panel with accessible status text and mobile layout; explicit offline/privacy behavior.
- AI-assisted dependency/CVE remediation planning when the operator enables an AI provider and `remediation_use_ai`, with reviewer-visible blocking/failure states.
- Operational runbook and staging/production acceptance checklist.

### Fixed
- Repeated scans now preserve existing finding remediation plans rather than overriding approvals or repeatedly invoking AI.
- AI advisor now accepts parent context cancellation, cannot reduce review requirements, and cannot redirect deterministic patcher targets or validation commands.
- Finding detail capabilities reflect actual backend eligibility instead of claiming that a fixed-version hint means the patcher can open a dependency upgrade PR.
- Dashboard critical/high triage link now leads to the combined focus queue.

### Verification limits
- Isolated `vulnintel` and `remediation` package tests passed; JavaScript syntax check passed. Full Go 1.26.10 module tests, Docker/Gitea E2E, live external lookups and full UI/accessibility checks remain NOT_PROVEN in this build environment.

## [v0.1.0-beta.3] — 2026-09-04

### Added
- Doctor commit/build metadata on `/health`, `/about`, and Doctor (`version` + `commit` + `build_date` via image ldflags)
- Disposable Gitea 1.22.3 E2E harness + clean-install proof against published digests (Phase 6A/6B)
- `docs/FINDING_RESOLUTION_SEMANTICS.md` — intentional non-auto-close on partial scans (RD-017D)
- Minimal public validation badges (CI, beta.3, container, license, E2E tested Gitea 1.22.3)
- Sanitized release acceptance evidence under `docs/release/`

### Fixed
- Public installable image alignment: beta.3 includes Doctor and current accepted tree (docs/source vs beta.2 digest drift)
- DOC_TRUTH_AUDIT capability rows updated to Phase 6B evidence (not collapsed “E2E passed”)

### Changed
- Recommended image pin → `v0.1.0-beta.3`
- Secret resolve-after-fix remains **PARTIAL** by design until full-scope reconcile/evidence

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
