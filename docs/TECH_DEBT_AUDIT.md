# RD-030 Phase 8A — Architecture Tech-Debt Audit

**Date:** 2026-09-05  
**Scope:** Read-first duplicate-path / overlapping-architecture audit  
**Constraint:** Document only — **do not delete code** in this phase  
**Method:** Static search of Go sources (excluding `vendor/`), route wiring in `main.go` / `handlers/` / `ui/` / `api/`, and related unit tests

## Classification legend

| Class | Meaning |
|-------|---------|
| **ACTIVE_CANONICAL** | Current intended production path; callers wired at runtime |
| **ACTIVE_COMPATIBILITY** | Still reached or intentionally retained for older configs / aliases / wrappers |
| **DEPRECATED** | Documented or commented as legacy; still present; prefer not to extend |
| **DEAD_PROVEN** | Search shows **no non-test production callers** (safe candidate for later removal *after* a dedicated cleanup task) |
| **UNKNOWN** | Ambiguous reachability, feature-flagged, or incomplete wiring — do not treat as dead |

Prefer **UNKNOWN** or **DEPRECATED** over a false **DEAD_PROVEN**.

---

## Executive summary

Repository Detective has **one primary scan pipeline** (`analyzers.Engine` → persist → optional forge issues / PR summary / commit status) with **multiple legitimate entry points** (webhook, manual API, UI Scan Now, scheduler, optional runner delegate). Most “duplicates” are **layered stages** or **thin adapters**, not competing engines.

Highest-value consolidation debt (later phases — not this audit):

1. **Unused `store.PolicyOutcome*`** vs live `gitea.PolicyOutcome*` (delete or single-source).
2. **Multiple redaction implementations** (`internal/security`, `redact`, `openclaw` extras, thin wrappers).
3. **learning ↔ findinglearn** thin wrappers + **calibration** naming collision (matcher vs recommendations).
4. **Scaffold / unused helpers**: `RunAllCandidates`, `BuildWorkflowTrigger`, `handleDefaults`, `RecordIssues`, `ComputeOverallScore`.
5. **GitHub asymmetry**: client LIVE for full-repo/issues; PR analyze, webhook, PR summary, reconcile remain Gitea-bound; **Forgejo** not proven.
6. **“CAH” name collision**: analyzers pipeline vs OpenClaw advisory harness.

---

## 1. Scan triggers

| Path | Role | Class |
|------|------|-------|
| `POST /webhook` → `handlers.WebhookHandler` → `webhookProcessor` | Push / PR forge events | ACTIVE_CANONICAL |
| `POST /api/v1/analyze` (+ `/analyze/all`) → `handleManualAnalysis` / `enqueueManualAnalysis` | Manual / agent / MCP API | ACTIVE_CANONICAL |
| UI Scan Now → `ui.SetScanTrigger` (`main_manual_scan.go` `wireScanTrigger`) → `enqueueManualAnalysis` | Operator UI | ACTIVE_CANONICAL |
| `orch.Scheduler` (`scheduler_enabled`) | Cron / poll scheduled repos | ACTIVE_CANONICAL |
| `tryDelegateScan` → `runner.Dispatcher.CreateScanJob` | Optional off-core runner when policy says delegate | ACTIVE_CANONICAL (when delegation enabled) |

**Convergence:** Scheduled and webhook/manual full-repo paths call `analysisEngine.AnalyzeRepository` (or runner result ingestion). UI and API share `enqueueManualAnalysis`. Delegation short-circuits local analyze when `runner.ShouldDelegate` returns `DecisionDelegate`.

| Field | Notes |
|-------|-------|
| **Old vs newer** | Not old/new — parallel entry points into one engine. Historical note in `issues.md`: webhook auth once lived in `main.go`; consolidated onto `handlers.WebhookHandler`. |
| **Callers** | `main.go` routes; `ui/manual_scan_handlers.go`; `orch/scheduler.go` via `ScanRunner` callback; MCP `rd_analyze` → API. |
| **Reachability** | All five reachable when components ready + feature flags / tokens configured. |
| **Tests** | `handlers/webhook*_test.go`, `ui/handler_test.go` (ScanTrigger), `orch/scheduler_test.go`, `runner/runner_test.go` (`ShouldDelegate`). |
| **Persistence** | Each creates/updates `scans` with `trigger_type` (`push` / `pull_request` / `manual` / `scheduled` / runner-associated). |
| **Safe disposition** | Keep all entry points. Do not collapse into one HTTP route. Optional later: extract shared “start scan” helper to reduce `main.go` duplication between webhook processor and enqueue paths (**UNKNOWN** size — not cleanup-critical). |

---

## 2. Webhook handlers

| Path | Class |
|------|-------|
| `handlers/webhook.go` — HMAC, rate limit, include/exclude, `AnalysisProcessor` | ACTIVE_CANONICAL |
| `main.go` `webhookProcessor` (`ProcessPush` / `ProcessPullRequest`) | ACTIVE_CANONICAL (processor bridge, not a second HTTP handler) |

| Field | Notes |
|-------|-------|
| **Old vs newer** | Old inline `main.go` webhook auth → **replaced** by secure handler (see `issues.md`). Processor remains in `main` to access engine/store. |
| **Callers** | `router.POST("/webhook", … webhookHandler.HandleWebhook)`. |
| **Reachability** | Production forge callback URL `{public_url}/webhook`. |
| **Tests** | `handlers/webhook_test.go`, `webhook_auth_test.go`; E2E webhook delivery proof (RD-017). |
| **Persistence** | Delivery recorder → `operator_evidence` (`webhook.last_delivery`); scan rows on accepted events. |
| **Safe disposition** | Keep. Empty webhook secret still warned (open follow-up in `issues.md`) — policy hardening, not path deletion. |

---

## 3. Finding normalization

Multiple **stages**, not competing normalizers:

| Stage | Location | Class |
|-------|----------|-------|
| Scanner → pipeline candidate | `scanners.Finding.ToCandidateFinding`, `health`/`graph` `ToCandidateFindings` | ACTIVE_CANONICAL |
| Built-in static patterns | `analyzers/static.go` `RunStaticAnalysis*` | ACTIVE_CANONICAL |
| Post-analysis product normalize | `profile.NormalizeIssues` (routing, FP risk, lifecycle, path) | ACTIVE_CANONICAL |
| Persist-time enrich / calibration apply | `findinglearn.*` via `store/findings_persist_sqlite.go` | ACTIVE_CANONICAL |
| Category canonicalization for issues/UI | `issues.NormalizeCategory` | ACTIVE_CANONICAL |
| Learning rule ID collapse | `store.NormalizeLearningRuleID` | ACTIVE_CANONICAL |

| Field | Notes |
|-------|-------|
| **Old vs newer** | `profile.NormalizeIssues` is the product-facing normalizer; `findinglearn` is persist/calibration shaping; `issues.NormalizeCategory` is forge/reporting mapping. |
| **Callers** | `analyzers/engine.go` (~NormalizeIssues); persist path; issue template/manager/fingerprint. |
| **Reachability** | Always on scan completion paths that produce issues. |
| **Tests** | `profile/profile_test.go`, `findinglearn/*_test.go`, `issues/fingerprint_test.go`, `analyzers/static_test.go`. |
| **Persistence** | Affects fingerprints, severities, `structural_hash`, suppressions/calibration application before SQLite write. |
| **Safe disposition** | Do not merge packages blindly. Later: document stage diagram in architecture; avoid adding a fourth normalizer. |

---

## 4. Issue creation / upsert

| Path | Role | Class |
|------|------|-------|
| `issues.Manager.CreateIssuesFromAnalysis` | Create/update forge issues from analysis | ACTIVE_CANONICAL |
| `issuelink` (`BackfillExternalIssueMappings`, `LinkForgeIssue`, `MappedIssue`) | Mapping repair / lookup | ACTIVE_CANONICAL |
| `main_issuelink.go` bridge → `issueManager.SetBackfillRunner` | Wires issuelink into manager | ACTIVE_CANONICAL |
| `reconcile.Engine` | Lifecycle reconcile / comments / calibration annotate | ACTIVE_CANONICAL (separate concern) |
| `store.Recorder.RecordExternalIssues` | Persist `external_issues` after filing | ACTIVE_CANONICAL |
| `store.Recorder.RecordIssues` | Legacy combined persist | **DEAD_PROVEN** (only `store/store_test.go`) |

| Field | Notes |
|-------|-------|
| **Old vs newer** | Manager is filing; issuelink is mapping layer extracted for backfill; reconcile is post-hoc lifecycle. |
| **Callers** | `createIssuesFromResult` in `main.go`; UI/API remediation/reconcile handlers. |
| **Reachability** | Gated by `auto_create_issues` / issue policy / force flags / forge client readiness. |
| **Tests** | `issues/manager_test.go`, `issuelink/backfill_test.go`, `reconcile/engine_test.go`. |
| **Persistence** | `external_issues`, lifecycle events, finding↔issue links. Removing `RecordIssues` later is low risk if tests updated. |
| **Safe disposition** | Keep manager + issuelink. Mark `RecordIssues` for future delete in a cleanup PR. |

---

## 5. PR summary handling

| Path | Role | Class |
|------|------|-------|
| `issues.UpsertPRPolicySummary` + `RenderPRPolicySummary` | Idempotent marker upsert (RD-006A) | ACTIVE_CANONICAL |
| `main_pr_summary.go` `maybePostPRPolicySummary` | Decides when to post; adapts `gitea.Client` | ACTIVE_CANONICAL |
| `gitea` issue comment create/edit/delete APIs | Transport only | ACTIVE_CANONICAL |

| Field | Notes |
|-------|-------|
| **Old vs newer** | Compact single-comment summary replaced ad-hoc multi-comment behavior (RD-006 / 006A). |
| **Callers** | Post-scan path from `main.go` after policy eval (PR number present). |
| **Reachability** | Gitea client required; skipped for observe+empty cases per `maybePostPRPolicySummary`. |
| **Tests** | `issues/pr_summary_test.go`; Gitea E2E PR summary idempotency. |
| **Persistence** | Forge-side comment only (marker `<!-- repository-detective-policy-summary -->`); no separate summary table. |
| **Safe disposition** | Keep. GitHub PR summary path not mirrored here (**UNKNOWN** / incomplete for GitHub PRs). |

---

## 6. Policy status / policy outcomes (`store` vs `gitea`)

| Path | Role | Class |
|------|------|-------|
| `gitea.EvaluatePolicyOutcome` + `gitea/checks.go` | **Computes** POLICY_* + commit-status evaluation | ACTIVE_CANONICAL |
| `gitea/status.go` `MapGiteaCommitState` | Maps logical states → forge API states | ACTIVE_CANONICAL |
| `store/enforcement.go` Observe/Warn/Enforce helpers | Operator-facing policy_level aliases | ACTIVE_CANONICAL |
| `store.PolicyOutcome*` constants in `enforcement.go` | Mirrored string consts intended for API/UI | **DEAD_PROVEN** (defined only; zero `store.PolicyOutcome*` refs — runtime uses `gitea.PolicyOutcome*`) |
| `gitea/checks_policy.go` | Comment-only stub (“logic lives in checks.go”) | DEPRECATED (discoverability stub; no symbols) |
| `store/policy.go` | Effective settings resolution (`policy_level`, issue/AI/runner policies) | ACTIVE_CANONICAL (settings, not outcome engine) |
| `store/scanner_coverage.go` + filing policy helpers | Coverage / filing gates feeding outcomes | ACTIVE_CANONICAL |

| Field | Notes |
|-------|-------|
| **Old vs newer** | Stored DB values remain `monitor_only` / `issue_only` / `gate_pr`; operator labels Observe/Warn/Enforce are newer aliases. Outcome **evaluation** lives only in `gitea`; store outcome constants were a planned mirror that never gained callers. |
| **Callers** | Commit-status posting + PR summary (`main_pr_summary.go` compares `gitea.PolicyOutcome*`); UI/API use `store.Enforcement*`. |
| **Reachability** | Always for Gitea scans that post status; outcomes also land in scan `summary_json` (metrics query both casings). |
| **Tests** | `gitea/checks_test.go`, `store/enforcement_coverage_test.go`, `store/policy_test.go`. |
| **Persistence** | `policy_level` on repo/global settings; outcome strings in scan summary JSON; commit status on forge. |
| **Safe disposition** | Delete unused `store.PolicyOutcome*` in a cleanup PR, or wire them as the sole constant source via a tiny shared package. Keep `checks_policy.go` or replace with a one-line doc pointer. |

---

## 7. Scanner execution

| Path | Role | Class |
|------|------|-------|
| `scanners.Registry.RunAll` / package `RunAll` | External tools (trivy, grype, gitleaks, …) concurrent | ACTIVE_CANONICAL |
| `analyzers` static + `health` + `graph` inside CAH Scan stage | In-process deterministic auditors | ACTIVE_CANONICAL |
| `runner/executor.go` `scanners.RunAll` | Same registry on runner worker | ACTIVE_CANONICAL |
| `preinstall.Runner` `scanners.RunAll` + `RunStaticChecks` | Pre-install audit (separate product surface) | ACTIVE_CANONICAL |
| `scanners.RunAllCandidates` | Thin wrapper returning candidates only | **DEAD_PROVEN** (definition only; no callers) |

| Field | Notes |
|-------|-------|
| **Old vs newer** | External registry is primary for tool findings; static/health/graph are complementary, not legacy replacements. |
| **Callers** | `analyzers/engine.go` Scan stage; runner executor; preinstall. |
| **Reachability** | Profile/required-scanner gates apply; binaries may be missing → coverage incomplete. |
| **Tests** | `scanners/*_test.go`, `analyzers/engine_scan_test.go`, `scanners/status_test.go`. |
| **Persistence** | `scanner_results`, findings, coverage summary. |
| **Safe disposition** | Keep dual in-process + external. `RunAllCandidates` safe delete candidate later. |

---

## 8. AI provider invocation

| Path | Role | Class |
|------|------|-------|
| `ai.Client` / transports (OpenAI-compatible, Anthropic) | CAH LLM auditors when `enable_llm_auditors` + depth ≥ 3 | ACTIVE_CANONICAL (optional) |
| `ai.ResolveConfig` + `LegacyConfig` (OpenWebUI URL/token/model) | Legacy provider fallbacks | ACTIVE_COMPATIBILITY / DEPRECATED |
| `analyzers/llm.go` `LLMEnabled` / policy helpers | Gate only (not a second client) | ACTIVE_CANONICAL |
| `openclaw` review + CAH candidate selection | **Post-scan** AI recommendations (separate from CAH auditors) | ACTIVE_CANONICAL (optional) |
| API aliases `/openclaw/*`, `/ai-review/*` beside `/ai-recommendations/*` | Route aliases | ACTIVE_COMPATIBILITY |
| `remediation.StubAIAdvisor` | Placeholder when remediation_use_ai set | ACTIVE_COMPATIBILITY (non-functional advisor) |

| Field | Notes |
|-------|-------|
| **Old vs newer** | OpenWebUI-specific config → generic `ai_provider`/`ai_base_url`. OpenClaw-named keys → `ai_recommendations_*` with legacy merge (`openclaw/config.go`). |
| **Callers** | `main.go` startup `ai.NewClient` when `needsAIProvider()`; onboarding `test-ai`; openclaw handlers / `maybeEnqueueOpenClawReview`. |
| **Reachability** | Default off; privacy mode can block egress (`privacy.EvaluateAIEgress`). |
| **Tests** | `ai/*_test.go`, `openclaw/config_test.go`, `openclaw/redact_test.go`, `analyzers/scanconfig_test.go`. |
| **Persistence** | AI review tables (`store/ai_review_*.go`); no requirement for deterministic scans. |
| **Safe disposition** | Keep CAH AI and recommendations separate. Deprecate OpenWebUI-only fields gradually; keep aliases until docs/clients migrate. Document that “CAH” in `analyzers` (pipeline) ≠ OpenClaw “CAH harness” (advisory candidate selection). |

---

## 9. Onboarding verification

| Path | Role | Class |
|------|------|-------|
| `handlers/onboarding.go` — UI `/onboard`, test-gitea/ai, repos, webhooks | Connect/select base | ACTIVE_CANONICAL |
| `handlers/onboarding_phase4.go` — permissions, privacy-preview, recommend-profile, **verify** | Protect/verify/ready (RD-013) | ACTIVE_CANONICAL |
| `handleDefaultsExtended` | Defaults JSON for wizard | ACTIVE_CANONICAL |
| `handleDefaults` (older, narrower JSON) | Superseded by extended | **DEAD_PROVEN** (unreferenced; routes use Extended) |

| Field | Notes |
|-------|-------|
| **Old vs newer** | Phase 4 routes registered from `RegisterRoutes` via `registerPhase4Routes`. Verify reuses `doctor.Run`. |
| **Callers** | Wizard static UI + `/api/v1/onboard/*`. |
| **Reachability** | Onboard API group uses API key auth without requiring full component ready (startup-friendly). |
| **Tests** | `handlers/onboarding*_test.go`, `onboarding_auth_test.go`. |
| **Persistence** | Webhook registration on forge; optional fresh-install counters; verify does not replace E2E webhook proof. |
| **Safe disposition** | Keep split files. Delete `handleDefaults` in a later cleanup PR after confirming no external docs reference its shape. |

---

## 10. Doctor checks

| Path | Role | Class |
|------|------|-------|
| `doctor.Run` / `doctor/report.go` | Shared diagnostic engine | ACTIVE_CANONICAL |
| `main_doctor.go` CLI `repository-detective doctor` | CLI | ACTIVE_CANONICAL |
| `GET /api/v1/doctor` (+ `/doctor/bundle`) | API | ACTIVE_CANONICAL |
| `GET /ui/doctor` | UI | ACTIVE_CANONICAL |
| Onboard `POST /api/v1/onboard/verify` | Wizard gate using same engine | ACTIVE_CANONICAL |
| `preinstall` checks / runner | Supply-chain pre-install audit | ACTIVE_CANONICAL (**different product** — not a Doctor duplicate) |

| Field | Notes |
|-------|-------|
| **Old vs newer** | Single engine, multiple surfaces (RD-014). |
| **Callers** | CLI argv hook `maybeRunDoctorCLI`; `registerDoctorAPI`; UI handler; onboard verify. |
| **Reachability** | Always available when binary runs; proofs (webhook delivery / first scan) depend on evidence tables. |
| **Tests** | `doctor/report_test.go`. |
| **Persistence** | Reads config + `operator_evidence`; support bundle redacts via `security`/`doctor.RedactReport`. |
| **Safe disposition** | Keep multi-surface. Do not fold preinstall into Doctor. |

---

## 11. Privacy enforcement

| Path | Role | Class |
|------|------|-------|
| `internal/privacy` (`NormalizeMode`, `EvaluateAIEgress`, `ClassifyURL`, …) | Runtime enforcement + classification | ACTIVE_CANONICAL |
| `privacy_mode=standard` alias → hybrid-compatible default | Legacy mode name | ACTIVE_COMPATIBILITY |
| Docs `docs/PRIVACY_MODES.md` / SECURITY_MODEL | Operator contract | ACTIVE_CANONICAL (docs) |
| Older informal “don’t send secrets” only via ad-hoc redaction | Pre–privacy-mode era | DEPRECATED (behavior superseded by modes + gates) |

| Field | Notes |
|-------|-------|
| **Callers** | `main.go` validate/startup AI gate; onboard privacy-preview/verify; doctor privacy checks; status payload. |
| **Reachability** | Always; `local_only` blocks external AI configuration. |
| **Tests** | Privacy exercised via doctor/onboard tests and mode normalization in `internal/privacy`. |
| **Persistence** | Config/env `privacy_mode`; not a separate DB table. |
| **Safe disposition** | Keep `internal/privacy` as sole policy brain. Retain `standard` alias until configs migrate. |

---

## 12. Remediation planning / execution

| Path | Role | Class |
|------|------|-------|
| `remediation.Planner` (+ recipes/risk/renderer) | Plan generation | ACTIVE_CANONICAL |
| `main_remediation.go` bridges → API/UI | Wiring / approve/reject / comment | ACTIVE_CANONICAL |
| `patcher` | PR eligibility + patch attempt execution | ACTIVE_CANONICAL (feature-flagged) |
| `closure` | Evidence / verify / merge check | ACTIVE_CANONICAL (feature-flagged) |
| Class-B sandboxed execution | RD-008B Option C — not expanded | UNKNOWN / NOT_PROVEN (product gate) |

| Field | Notes |
|-------|-------|
| **Old vs newer** | Layered pipeline: plan → patch attempt → closure evidence. Stub AI advisor is intentional Phase-1 placeholder (`remediation/ai.go`). |
| **Callers** | `api/remediation_handler.go`, UI finding detail actions, `main_closure.go` on scan finish. |
| **Reachability** | Gated by `remediation_*` / `evidence_closure` / PR enable flags; Doctor reports planner/PR/class_b states. |
| **Tests** | `remediation/planner_test.go`, `e2e/workflow_test.go` (closure), patcher tests. |
| **Persistence** | `remediation_plans`, `patch_attempts`, `closure_evidence` (and converts in `store`). |
| **Safe disposition** | Keep packages separate. Do not delete StubAIAdvisor until a real advisor is wired. Honor Class-B gate before expanding execution UX. |

---

## 13. Runner dispatch

| Path | Role | Class |
|------|------|-------|
| `runner.ShouldDelegate` + `tryDelegateScan` | Admission to delegate | ACTIVE_CANONICAL |
| `runner.Dispatcher` (`CreateScanJob` / typed / container) | Queue jobs in SQLite | ACTIVE_CANONICAL |
| `runner` receiver/executor/worker | Pull/execute/report results | ACTIVE_CANONICAL |
| `ModeGiteaActions` / `SelectRunnerMode` | Mode selection for job rows | ACTIVE_COMPATIBILITY / partial |
| `runner.BuildWorkflowTrigger` (`gitea_actions_backend.go`) | workflow_dispatch payload builder | **DEAD_PROVEN** (tests only; no production caller) |
| `GiteaActionsBackendConfig` fields on platform settings | Config surface for future/test backend | UNKNOWN (config exists; trigger builder unused) |

| Field | Notes |
|-------|-------|
| **Old vs newer** | Core in-process scan is default; native runner delegation is optional Phase 12 surface; Gitea Actions backend is scaffold-heavy. |
| **Callers** | Scheduled/manual full scans when delegation enabled; container image jobs. |
| **Reachability** | Requires `runner` config + shared secret + worker; push triggers typically stay on core (`ShouldDelegate` policy). |
| **Tests** | `runner/runner_test.go`, `dispatcher_test.go`, `gitea_actions_backend_test.go`, `receiver_test.go`. |
| **Persistence** | `runner_jobs` (+ policy/spec JSON). |
| **Safe disposition** | Keep dispatcher. Treat `BuildWorkflowTrigger` as scaffold — either wire it or remove in a dedicated task after product decision. |

---

## 14. GitHub provider

| Path | Role | Class |
|------|------|-------|
| `github.Client` (`forge.RepoClient`) | Content/list/ref/archive/hooks/issues | ACTIVE_COMPATIBILITY (wired, experimental product claim) |
| `gitea.Client` + `gitea.ForgeClient` adapter | Primary forge | ACTIVE_CANONICAL |
| `forge` interfaces | Shared abstraction | ACTIVE_CANONICAL |
| `AnalyzePullRequest` | Always uses `e.giteaClient.GetPullRequest` / `GetChangedFiles` | ACTIVE_CANONICAL for Gitea; **UNKNOWN/gap** for GitHub PRs |
| `main_reconcile.go` `ForgeFor("gitea")` | Reconcile bridge hardcodes Gitea | ACTIVE_CANONICAL (Gitea-only today) |
| `POST /webhook` | Gitea-shaped payload only | ACTIVE_CANONICAL (no GitHub webhook handler) |
| GitLab / Forgejo dedicated packages | Planned in older architecture docs | UNKNOWN / NOT_IMPLEMENTED |

| Field | Notes |
|-------|-------|
| **Old vs newer** | Gitea-first; GitHub added behind `github_url`/`github_token` and `forge_type=github`. |
| **Callers** | `main.go` constructs client when token set; full-repo analyze via `forge.RepoClient`; `issues.Manager` GitHub forge; bulk analyze orgs. |
| **Reachability** | Full-repo + issue filing can use GitHub when configured. PR analyze, webhooks, PR policy summary, and reconcile remain **Gitea-centric**. Public docs: GitHub issue filing **experimental**. |
| **Tests** | `github` package tests; `issues/forge_client_test.go`. |
| **Persistence** | Same `external_issues` with `forge_type` discriminator. |
| **Safe disposition** | Do not remove the client. Do not advertise parity. Future: either implement GitHub PR/webhook/reconcile paths or quarantine GitHub behind an explicit beta flag documenting the gaps. |

---

## 15. Old setup / config routes

| Path | Role | Class |
|------|------|-------|
| `GET /` → redirect `/onboard/` | Entry | ACTIVE_CANONICAL |
| `/onboard` wizard | Setup | ACTIVE_CANONICAL |
| `/ui/configure` | Platform configuration | ACTIVE_CANONICAL |
| `POST /api/v1/config/reload` | Hot reload | ACTIVE_CANONICAL |
| `/setup` HTTP route | — | **DEAD_PROVEN** / never present in current tree (no route match) |
| `internal/config/envcompat` | Maps `REPOSITORY_DETECTIVE_*` → viper; **ignores** `BUGBOT_*` | ACTIVE_COMPATIBILITY (prefix migration complete; old prefix not applied) |
| UI `brand_display` scrub of Bugbot strings/paths | Display-only legacy brand cleanup | ACTIVE_COMPATIBILITY |
| Query-string API key acceptance | Compat auth | DEPRECATED / ACTIVE_COMPATIBILITY (Doctor warns) |

| Field | Notes |
|-------|-------|
| **Persistence** | Default DB path `./data/repository-detective.db` (no automatic `bugbot.db` rename found in code — operators may still have old files on disk: **UNKNOWN** ops concern). |
| **Safe disposition** | Keep envcompat + onboard/configure. No `/setup` to delete. Document operator DB path migration outside code if needed. |

---

## 16. Redaction packages

| Path | Role | Class |
|------|------|-------|
| `internal/security` (`RedactSecrets`, `RedactLogField`, `SanitizeDiagnostic`, access-log helpers) | **Canonical** ops/UI/diagnostics redaction | ACTIVE_CANONICAL |
| `redact.SecretEvidence` | Smaller pattern set for finding snippets / OpenClaw base | ACTIVE_CANONICAL (narrower domain) |
| `openclaw.RedactText` / `RedactPacket` | Outbound AI packet policy (+ PII options); calls `redact` then extras | ACTIVE_CANONICAL |
| `containers.RedactLogLine`, `runner.RedactLogLine` | Thin wrappers → `internal/security` | ACTIVE_COMPATIBILITY |
| `scanners` log helper using `security.RedactLogField` | Scanner status logs | ACTIVE_CANONICAL |

| Field | Notes |
|-------|-------|
| **Old vs newer** | Pattern duplication between `internal/security` and `redact` (overlapping regexes). Wrappers exist to avoid import churn. |
| **Callers** | Webhook logs, UI auth, middleware access log, doctor format, preinstall failures (`redact.SecretEvidence`), fingerprints, OpenClaw reviews. |
| **Tests** | `internal/security/redact_corpus_test.go`, `redact/secrets_test.go`, `openclaw/redact_test.go`, wrapper tests. |
| **Persistence** | Redaction is mostly ephemeral; durable diagnostics should use `SanitizeDiagnostic`. |
| **Safe disposition** | Later unify on `internal/security` + keep `openclaw` policy layer; make `redact` a thin alias or delete after call-site migration. **Do not delete in 8A.** |

---

## 17. learning vs findinglearn vs calibration

| Package | Role | Class |
|---------|------|-------|
| `findinglearn` | Structural hash, path classify, actionability, **repo calibration rule apply** | ACTIVE_CANONICAL (core algorithms) |
| `learning` | Event types, emit helpers, accept validation, sanity gate; **wrappers** re-export findinglearn enrich APIs | ACTIVE_CANONICAL |
| `calibration.Matcher` | Suppression matching cache for scan/reconcile | ACTIVE_CANONICAL |
| `store/learning_sqlite.go` + `store/calibration_sqlite.go` | Persistence / recommendation generation | ACTIVE_CANONICAL |
| UI `/ui/learning` + API calibration/suppressions | Operator surfaces | ACTIVE_CANONICAL |

| Field | Notes |
|-------|-------|
| **Old vs newer** | Algorithms live in `findinglearn`; `learning` is the product event/API façade; `calibration` matcher is suppressions (name overlaps “calibration recommendations” in store/UI). |
| **Callers** | Persist path → findinglearn directly; UI/tests often call `learning.*` wrappers; `main_suppressions.go` builds `calibration.Matcher`. |
| **Reachability** | Learning events on scan/lifecycle; recommendations via recompute job / UI. |
| **Tests** | `learning/learning_test.go`, `findinglearn/*_test.go`, `calibration/matcher_test.go`, store learning/calibration tests. |
| **Persistence** | `learning_events`, rule stats, `repo_calibration_rules`, suppressions. |
| **Safe disposition** | Keep three packages short-term. Later rename/docs to reduce “calibration” ambiguity (matcher vs recommendations). Removing `learning` wrappers would force wide call-site churn — low priority. |

---

## Cross-cutting compat inventory (quick)

| Item | Class | Notes |
|------|-------|-------|
| Scan profile legacy IDs (`beta_standard`, `fast`, …) | ACTIVE_COMPATIBILITY | `store.NormalizeScanProfile` |
| Enforcement label vs stored `policy_level` | ACTIVE_COMPATIBILITY | `store.Enforcement*` helpers (live) |
| `store.PolicyOutcome*` | **DEAD_PROVEN** | Unused mirror of `gitea.PolicyOutcome*` |
| OpenClaw / ai-review route aliases | ACTIVE_COMPATIBILITY | `api/openclaw_review_handler.go` |
| Analyzers CAH vs OpenClaw “CAH” harness | ACTIVE_CANONICAL (naming collision) | Different features; document carefully |
| `ai.LegacyConfig` OpenWebUI | DEPRECATED | Still resolved |
| `analyzers.ComputeOverallScore` | **DEAD_PROVEN** | Tests only |
| `gitea/checks_policy.go` | DEPRECATED | Empty stub file |
| Fingerprint prefix / brand scrub | ACTIVE_COMPATIBILITY | Display + labels |

---

## DEAD_PROVEN candidates (evidence)

Only these met the bar (no non-test production callers found):

| Symbol | Evidence |
|--------|----------|
| `scanners.RunAllCandidates` | Sole hit is definition in `scanners/runner.go` |
| `runner.BuildWorkflowTrigger` | Only `gitea_actions_backend_test.go` |
| `OnboardingHandler.handleDefaults` | Only definition; routes use `handleDefaultsExtended` |
| `store.Recorder.RecordIssues` | Only `store/store_test.go` (+ deprecated comment) |
| `analyzers.ComputeOverallScore` | Only `analyzers/scoring_test.go` |
| `store.PolicyOutcome*` constants | Defined in `store/enforcement.go`; zero references as `store.PolicyOutcome*` (callers use `gitea.PolicyOutcome*`) |

**Not** marked dead despite limited use: GitHub client (partial), Gitea Actions **mode** selection on jobs, StubAIAdvisor, envcompat, redaction wrappers, `store.Enforcement*` (live).

---

## Safe disposition principles (for later phases)

1. **No deletions in RD-030 8A** — this file is the inventory.
2. Prefer extracting shared packages (`policyoutcome`, unified redaction) over deleting one side of a mirror.
3. Delete DEAD_PROVEN helpers only with: grep gate + test update + changelog note.
4. Feature scaffolds (`BuildWorkflowTrigger`) need a product decision: wire or remove — do not leave forever as fake support.
5. Preserve all scan **entry points**; consolidate only shared post-admit orchestration inside `main` if needed.

---

## Optional: Forgejo adaptation estimate (investigation only)

**Claim level:** estimate only — **do not claim Forgejo support**. Product docs today: Gitea **1.22.3** E2E-proven; Forgejo **NOT_PROVEN**.

### Recommendation

**Start with Forgejo 15 LTS**, not 16, for the first adaptation spike.

### Why 15 LTS first

- LTS channel matches operator expectations for homelab/enterprise forges (stability over newest API churn).
- Closer behavioral kinship to the already-proven Gitea 1.22.x API surface used by the harness (`docker-compose.e2e.yml`, `scripts/e2e-gitea-acceptance.sh`).
- Forgejo 16 may introduce additional Actions/API deltas; better as a **second** matrix entry after 15 LTS greeps green.

### What would be adapted

| Area | Likely effort | Notes |
|------|---------------|-------|
| Compose image swap + admin bootstrap | **S** (0.5–1 d) | Replace `gitea/gitea:1.22.3` with Forgejo 15 LTS image; re-check root URL, webhook `ALLOWED_HOST_LIST`, install lock |
| Harness script assumptions | **S–M** (1–2 d) | Version assert (`GITEA_VERSION_EXPECTED`), API `/api/v1/version`, user/repo/webhook helpers may need Forgejo response quirks |
| Webhook signatures / events | **S** (0.5–1 d) | Confirm HMAC header names/payload parity with Gitea |
| Commit status + PR comments | **M** (1–3 d) | Core RD-017 scenarios; watch status state enums (`gitea.MapGiteaCommitState`) |
| Issue create/label/search | **M** (1–2 d) | Filing + backfill + idempotent PR summary |
| Actions / runner backend | **L** / defer | Already scaffold on Gitea; do not block Forgejo “API forge” proof on Actions |
| Docs / DOC_TRUTH / badges | **S** | Explicitly mark FORGEJO_15_LTS_E2E_PARTIAL/PROVEN — never imply Gitea proof transfers |

### Rough totals

| Target | Effort band | Outcome |
|--------|-------------|---------|
| **Forgejo 15 LTS** — core harness subset (webhook, first scan, PR summary, policy outcomes, doctor) | **~1–2 engineering weeks** calendar (about 5–8 focused days) including flake chasing | Adaptation spike; support claim only if zero FAIL on agreed scenario set |
| **Forgejo 16** — same subset after 15 | **+3–5 days** | Version matrix expansion; expect extra API/UI drift |
| Full parity marketing (“supported forge”) | **Multi-phase** | Needs soak, upgrade E2E, Actions decision, docs — out of scope for a spike |

### Explicit non-claims

- This audit does **not** add Forgejo to the supported matrix.
- Gitea 1.22.3 E2E proof does **not** transfer to Forgejo.
- GitHub experimental path is unrelated to Forgejo work.

---

## Suggested follow-on tasks (post–8A)

| ID | Task | Depends on |
|----|------|------------|
| 8B | Delete or quarantine DEAD_PROVEN helpers listed above | This audit |
| 8C | Single-source POLICY_* constants | Design for import cycles |
| 8D | Unify redaction → `internal/security` + openclaw policy layer | Corpus tests |
| 8E | Product decision: wire or remove Gitea Actions workflow trigger | Runner roadmap |
| 8F | Optional Forgejo 15 LTS harness spike | E2E maintainers |

---

## Audit metadata

| Field | Value |
|-------|-------|
| Program | RD-001…RD-030 Product Hardening |
| Phase | 8A — read-first duplicate-path audit |
| Code deleted | **None** |
| Primary evidence | Go grep + route wiring + package layout 2026-09-05; cross-checked with explore pass [Audit duplicate code paths](02a68de3-474a-429a-a3b8-37254126a3a3) |
| Related docs | `architecture.md`, `docs/E2E_GITEA_ACCEPTANCE.md`, `docs/DOC_TRUTH_AUDIT.md`, `docs/PR_SUMMARY_IDEMPOTENCY.md`, `docs/PRIVACY_MODES.md` |
