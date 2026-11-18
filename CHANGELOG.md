## Unreleased — CI executive evidence completeness and operator feedback

- Self-scan CI now always executes on source pushes, even with missing credentials. Configuration failure produces an explicit `EVIDENCE_INCOMPLETE` Markdown/JSON artifact and a failing job rather than skipping report publication.
- Report verifies the scan's actual repository ID/name and forge with an authenticated repository lookup, newly requested scan ID, source commit and report-only acknowledgement. Gitea and GitHub workflows select their own forge and require an uploadable artifact.
- SHA256SUMS now covers both Markdown and JSON evidence. Added fixture tests for wrong repository, wrong forge, mismatched scan ID, malformed evidence, absent dry-run acknowledgement and invalid configuration.
- Scan Details offers a local, privacy-minimal issue-report link prefilled only with the opaque scan ID. Kept README product-focused; updated CI runbook and deprecated release-guide claims.

## Unreleased — commit-specific CI evidence and community feedback

- On source pushes, Gitea and GitHub CI now request a fresh scan of the pushed repository commit, record per-scanner evidence, and upload an executive Markdown/JSON report plus checksum. Missing or mismatched evidence fails the self-scan job; neither a clean scan nor a passing test is described as a safety certification.
- Added a review-first in-app **Report a problem** composer: reproducible steps, optional diagnostics, credential masking, Gitea issue draft, and explicit human submission. No public issue-posting credential or hidden log upload.
- Added independent staging acceptance and fault-injection checklist for a better-equipped test agent. Product README remains task-focused.

# Changelog

## [Unreleased — finding ownership queues, 2026-11-18]

- Added local-user **Assigned to me** and **Unassigned** investigation queues, with owners visible alongside findings and explicit account-bound filters. Scope validation rejects unauthorized/unknown owner views rather than broadening the query.
- Owner filtering now executes in SQLite before ordering and pagination, with an assignee index (migration 31) and a single bounded owner-enrichment query for a page. CSV/JSON downloads preserve the selected scope.
- Added assignment-queue regression tests for access controls, counts, empty/invalid scopes, owner metadata, exports, and pagination.
- Removed duplicate external scanaislop badge from the product README. Forge ownership remains local-only pending identity mapping and permission reviews.


## [Unreleased — finding ownership and export clarity, 2026-11-18]

- Improved Findings CSV/JSON downloads with progress, download errors, and explicit no-cache attachment headers; preserved non-JavaScript link fallback.
- Added local finding-owner assignment, database migration 30, role checks, disabled-user protection, idempotent updates, and assignment history. Does not change external forge issue owners.
- Added the independent scanaislop score badge to README, clearly identified as external code-quality information rather than a security attestation.
- Documented an evidence-first evaluation process for new quality scanners and AI-slop detection candidates; no unvalidated scoring engine is enabled by default.
- Added export/assignment regression tests and a staged acceptance guide; full Go 1.26.11 and browser tests remain release gates.


## [Unreleased — runner callback ingestion consistency, 2026-11-18]

- Require a claimed native/Actions runner job for results. Atomic SQLite state transitions prevent simultaneous callbacks from ingesting one job twice.
- Preserve the signed result JSON artifact and SHA-256 digest **before** marking the job complete. Failed persistence and ingestion are now explicit `ingestion_failed` states rather than false success.
- Exact callback retries after a finalized result are idempotent; conflicting retries fail. Partial ingestion is intentionally not automatically replayed because forge issue and finding side effects are not globally transactional.
- Delegated scan completion now propagates scan recorder, finding-batch and forge-issue-link persistence errors, keeping an incomplete result visible. Added acceptance tests and a release-gate runbook. Full Go 1.26.11 tests and live runner integration remain pending.

## [Unreleased — remediation trust and operator evidence, 2026-11-17]

- Review actions now bind remediation plans to the requested finding; cross-finding submissions are refused. SQLite accepts only one atomic `proposed → approved/rejected` transition, preventing stale tabs from reviving terminal plans.
- PR eligibility now rejects every unsupported validation command (rather than skipping it) and refuses to claim runner-backed verification when the worker path is not connected. The feature remains disabled by default; explicitly selecting local execution does not create a sandbox.
- Added a guard against repeated PR attempts when a prior open/merged PR is recorded. Evidence persistence failures are surfaced as errors, with an instruction to check the forge before retrying. Concurrent cross-process claims and an unrecorded remote PR remain outstanding gates.
- The finding workbench now shows patch-attempt IDs, explicit errors, validation results, and diff summaries in a compact, responsive evidence panel. Root README stays product-oriented; deep operational warnings live in [PR documentation](docs/REMEDIATION_PRS.md).
- Added focused store, eligibility, and UI binding regression tests. Full Go 1.26.11 test execution and authenticated staging/runner validation remain **NOT RUN** in this environment.

## [Unreleased — CI module integrity and staticcheck, 2026-11-17]

- Tidied `go.sum` so `go mod tidy` is idempotent on Go 1.26.11 (was failing Gitea CI Verify module integrity).
- Pinned Gitea CI/release workflows to Go **1.26.11** to match `go.mod` toolchain.
- Cleared staticcheck U1000/ST1013 blockers: removed unused UI helpers, replaced dead first-run auto-calibrate with advisory-only completion evidence.
- Fixed CI container smoke: API key must be ≥16 chars (`ci-smoke-key` was rejected at startup); fail fast if the smoke container exits early.
- Made staticcheck install resilient to flaky `sum.golang.org` IPv6 (`cannot assign requested address`) with retries and a pinned-module GONOSUMDB fallback.

## [Unreleased — calibration FP/FN hardening, 2026-11-17]

- Stopped suppression status from training the false-positive learner (circular quieting); backfill now uses explicit `false_positive` dispositions only and skips protected security rules.
- Expired active repo calibrations and proposed recommendations that target protected families (gosec/CVE/secret/custom_rule) and refuse creating new ones.
- Forge routing no longer applies learned calibrations to protected rules; path-specific known-safe routing remains for reviewed tooling patterns.
- Calibration stats recommend `report_only` only with enough explicit FP evidence and low verified-fix rate (suppressions alone no longer inflate FP rate).
- Known-safe: test-path `errcheck` / ignored-error findings are forge-quiet without silencing production errcheck.
- Rule-ID matching strips AI backtick wrappers; ruff instance suffixes collapse for learning aggregation.

## [Unreleased — product README and remediation status clarity, 2026-11-17]

- Replaced the root README's chronological development diary with a product-oriented overview, capabilities, truthful limitations, an installation and first-scan path, screenshots, and task-based documentation links. Historical implementation details remain in the changelog and dedicated release/QA documents.
- Added an accessible four-stage evidence trail to the finding remediation workbench. It displays detection, plan, patch-attempt and scan-backed closure states without implying that proposals or attempts prove fixes.
- Grouped plan review controls, separated PR eligibility, and aligned secondary action colors with the dashboard design. Existing backend actions and authorization are unchanged.
- Added source-rendering and 24 synthetic-browser-state checks; full Go 1.26.11 tests and authenticated staging workflows are still pending. See [Remediation workbench UX QA](docs/REMEDIATION_WORKBENCH_UX_QA.md).

## [Unreleased — scan evidence & repository settings UX, 2026-11-17]

- Scan detail now presents a pinned-run decision brief and evidence-first review: outcome visualization, scanner result table, AI recommendations as advisory, and collapsible raw diagnostics. Invalid/missing scanner outcomes remain visible, and a scanner error disqualifies a nominally successful row from the completed-outcomes count.
- Repository settings now prioritizes the profile → custom rules → notifications path, retains all existing controls under explicit advanced sections, and improves long-name/mobile layout, labels, save clarity, and keyboard-accessible section links.
- Configure adds direct section anchors and corrects an ID collision on the scan-profile control. Scan soft refresh preserves the last scroll position and open diagnostic disclosures instead of resetting the operator's context.
- Synthetic Go template rendering and 16 Chromium light/dark viewport checks passed. The added Go regression test and live staging acceptance are **NOT RUN**; no change to scanner or AI execution is claimed. [Visual QA record](docs/SCAN_SETTINGS_WORKBENCH_UX_QA.md).

## [Unreleased — finding investigation UX workbench, 2026-11-17]

- Reworked the finding page into an investigation brief with the affected repository, scanner, actionable context, and a prominent remediation action. Separated scanner-match confidence, technical assessment, and explicit unknown exposure into a scannable side panel.
- Refined remediation and closure sections with clear plan status, risk, approval boundaries, supporting evidence links, and well-labeled review controls. Collapsed optional capability inventories and false-positive disposition controls without removing existing authorization or POST handlers.
- Improved long-text wrapping, mobile stacking and responsive severity-accented finding rows. Uses built-in HTML meter, native details and existing CSS instead of additional charting dependencies.
- Added a rendering contract test and a read-only 10-viewport/theme Playwright acceptance script. Standard-library HTML fixtures rendered and static checks passed; real Chromium navigation was denied in this container, and full Go 1.26.11/staging tests remain pending. See [finding workbench QA](docs/FINDING_WORKBENCH_DESIGN_AND_UX_QA.md).

## [Unreleased — dashboard visual hierarchy and responsive QA, 2026-11-17]

- Rebuilt the dashboard around priority decisions, concise metrics, meaningful trend/severity/category visuals and prominent investigation links; moved scanner inventories and calibration into accessible expandable evidence panels.
- Removed the dashboard category radar and duplicate backlog/report walls, simplified action colors, wrapped long finding paths and made priority tables scroll inside cards. A visually hidden table header was found to cause mobile document overflow; it was replaced with a visible column label. Theme bootstrap styling is now released after CSS load so switching themes is not overridden by the anti-flash guard.
- Optimized Chart.js labeling and deferred repository-map rendering until the operator opens the advanced panel. Added a ten-viewport dark/light browser QA script and operator design guide.
- Synthetic template rendering, JavaScript/CSS static checks and in-browser viewport checks passed; Go 1.26.11 full suite and live Gitea staging remain pending. See [dashboard QA guide](docs/DASHBOARD_DESIGN_AND_UX_QA.md).

## [Unreleased — delegated scanner evidence fidelity, 2026-11-17]

- Preserved worker-reported scanner finding counts and sanitized scanner errors through core ingestion and scan persistence; scanner `error` evidence now invalidates a nominal `clean` outcome.
- Standardized custom-rule scanner results to `found`/`clean`; passed pinned job commit identity into the persisted analysis result.
- Rejected unsupported runner result versions/statuses and invalid negative counts and timestamp order. Added targeted regression tests.
- Full Go 1.26.11 CI and a disposable Gitea/native-worker E2E **remain pending**. See [runner evidence release gate](docs/release/RUNNER_RESULT_EVIDENCE_INTEGRITY_2026-11-17.md).


## [Unreleased — capability-aware runners, AI result fidelity and safe issue filing, 2026-11-17]

### Fixed / improved
- Native workers now register capabilities before claiming; server-side SQLite claims select only compatible queued jobs, preventing a specialist worker from stealing and failing another worker's job.
- Calibration can no longer auto-quiet gosec G104/G204/G304 rule families. Individual finding dispositions remain available for reviewed scanner operations.
- Malformed AI audits and code reviews now return failures instead of falsely successful empty scans or fabricated 100%-confidence parse-error findings. Attack-surface JSON now correctly decodes the model’s snake_case fields, and missing required arrays fail explicitly.
- Forge issue routing fails closed on missing GitHub credentials; zero-confidence or uncorroborated AI-origin findings are ineligible for automatic filing. Remote deduplication / durable claim failures now prevent duplicate-prone creation.
- Refined the learning page with an evidence-first 3-step operator workflow, mobile layouts and more accurate calibration copy.
- Proposed refreshed runtime/toolchain versions: Go 1.26.11, Alpine 3.24 and a pinned gosec v2.30.0. Docker builds and scanner compatibility require CI validation.

### Verification
- New regression tests cover AI format failures, capability claims, heartbeat gating, calibration protections and filing eligibility. Local Go 1.23.2 cannot download the required toolchain; **tests not yet executed**. See [acceptance matrix](docs/release/DELEGATION_AI_CALIBRATION_ACCEPTANCE_2026-11-17.md).
- No automatic third-party preinstall posting, new Gitea/GitHub Actions backend or verified AI patch executor is claimed.


## [Unreleased — deep-scan report integrity, 2026-11-17]

### Fixed
- Repository audit now recognizes scanner outcomes `clean`, `found`, `passed` and `ok` as completed execution, while keeping `binary_missing`, skipped, failed and error-bearing results visible as coverage limitations. An independent scanner outcome is required before the report can suggest proceeding.
- Report top-risk selection and technical appendix now order by severity **in SQLite before pagination**, preventing older high-risk findings from being obscured by recent low-severity alerts.
- A bounded test-data cleanup batch now reports database close errors rather than ignoring them.

### Evidence and acceptance
- Added focused report, scan-outcome and priority regression cases. Full Go 1.26.10 CI and live staging report E2E **remain pending** for this patch; the maintainer's previous full test pass applies to the deployed base commit, not this change.
- Historical errcheck/G304/G204 records require same-commit reconciliation and site-by-site review; no blanket `gosec` exclusions were added. See [deep-scan findings reconciliation](docs/release/DEEP_SCAN_FINDINGS_RECONCILIATION_2026-11-17.md).

All notable changes to Repository Detective (formerly an internal prototype) are documented here.

## [Unreleased — report clarity and operator navigation, 2026-11-17]

### Improved
- Fleet and per-repository audit reports now have a clear cover, UTC snapshot timestamp, section jump links, an evidence-first reading order and printer-friendly headings. Replaced repository radar chart with an accessible, printable category summary (no chart runtime on repo reports).
- Report caveats now distinguish enabled scanners from scanner outcomes, first-page findings from full counts, a 200-repository listing cap from fleet totals, and finding-count ordering from actual exploitability risk. Repository reports show actual scanner result records from the newest scan.
- Recommendations now treat no scans, absent scanner records, incomplete latest jobs, scanner errors and stale scans as reasons to seek review rather than to imply a verified clean release. Past failed scans no longer automatically override a newer completed scan.
- Refined shared visual styling, local SVG navigation icons, current-page semantics, and an accessible Ctrl/⌘+K or `/` page switcher based on the existing authorized navigation links.
- Critical report database fetch failures now return an explicit 500 rather than rendering potentially misleading empty metrics.

### Verification limits
- Standard-library template parsing and synthetic fleet/repository page rendering passed; JavaScript syntax, static CSS validation and isolated headless Chromium switcher/dark-mode interaction passed. Full Go 1.26.10 tests and real Gitea/runner/report E2E remain **NOT RUN**. See [operator guide and acceptance](docs/OPERATOR_REPORTS_AND_NAVIGATION.md).

## [Unreleased — runner verification integrity, 2026-11-17]

### Fixed
- `remediation_verify` now actually invokes the external scanner stage, preserves the findings and scanner coverage, and fails when scanners did not run, failed, or still report issues. This is a conservative scan-evidence gate **not** automated PR/fix certification.
- Runner callbacks with `status=completed` and an error list are recorded and summarized as failed rather than silently marked complete.
- Unknown worker job types no longer fall back to a full scan, and empty job-type allowlists no longer allow every job.
- Scan detail no longer visually labels coverage review as a completed proof step; absent scanner evidence and missing binaries are more visible.

### Verification
- Added targeted runner and receiver regression cases; full Go tests and disposable Gitea/native-worker integration remain **NOT RUN** in the available Go 1.23.2 environment (required Go 1.26.10).
- [Detailed safeguards and outstanding acceptance gates](docs/release/RUNNER_VERIFICATION_FAIL_CLOSED_2026-11-17.md).

## [Unreleased — deterministic slop scoring, 2026-11-17]

### Added
- `slopscore` package: bounded, dependency-free quality scoring (0-100, A-F) for patterns generated code tends to leave behind — placeholders, commented-out code, assertions that cannot fail, swallowed errors, debug output, emoji in source and marketing filler in Markdown. Budget exhaustion is an explicit incomplete result, never a clean one.
- Opt-in pipeline wiring via `REPOSITORY_DETECTIVE_SLOP_SCORE_ENABLED` (default off). Emits `slopscore` findings and logs score, grade and top tells. It is a quality signal, not a security verdict, and does not feed policy gates.

## [Unreleased — dependency upgrade review hardening, 2026-11-17]

### Fixed
- Dependency-vulnerability plans (including category aliases) now always require maintainer review and cannot be marked safe for an unattended PR before a verified sandboxed upgrade executor exists.
- Missing scanner-reported fixed versions now produce an explicit blocker instead of implying a patched dependency can be selected.
- Deterministic dependency guidance includes direct/transitive dependency identification, compatibility and lockfile checks, isolated builds, and post-change rescans. Added focused regression tests and an in-context operator checklist.

### Verification limitations
- Twenty-two isolated, dependency-free Go remediation tests passed under the available Go 1.23.2 toolchain (including three new cases). The full project requires Go 1.26.10; full tests and live runner/Gitea remediation are not certified by this change.
- See [Veracode-inspired opportunity assessment](docs/VERACODE_INSPIRED_IMPROVEMENTS.md) for implemented safeguards, missing product capabilities, and release gates.

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
