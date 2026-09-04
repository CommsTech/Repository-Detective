# Development Issues Log

## Open / in progress (2026-09-04) — Phase 6B release alignment

| Priority | Issue | Plan |
|----------|-------|------|
| P0 | Published beta.2 lacked Doctor / current tree | Publish `v0.1.0-beta.3`; prove clean-install + core E2E on exact digest |
| P0 | DOC_TRUTH on GitHub lagged Phase 6A evidence | Snapshot lag root cause; refresh DOC_TRUTH + re-sync |
| P1 | Live-forge POLICY_MET / ACTION_REQUIRED / OBSERVATION_ONLY | Controlled harness scenarios (RD-017C) |
| P1 | Secret auto-close after fix looked incomplete | Document intentional PARTIAL semantics (RD-017D); no naive absence-close |

## Fixed (2026-09-04) — Phase 6A real Gitea E2E (RD-017A / RD-018)

| Priority | Issue | Resolution |
|----------|-------|------------|
| P0 | No disposable real-Gitea acceptance path | `docker-compose.e2e.yml` + `scripts/e2e-gitea-acceptance.sh` (Gitea 1.22.3) |
| P0 | Webhook delivery / FIRST_SCAN not durable | migration 25 `operator_evidence`; Doctor proofs |
| P0 | Required-scanner fail-closed only unit-tested | Live gitleaks stub → EVALUATION_INCOMPLETE E2E |
| P1 | Gitea cold SQLite init exceeded short waits | Readiness polling up to ~7m; healthcheck start_period raised |
| P1 | Safe/secure disclaimer false-failed harness | Context-aware claim check allows product non-assurance wording |
| P1 | Fail-closed stub Permission denied | Install stub as root inside disposable container only |
| P2 | Upgrade E2E | NOT_PROVEN — no prior public-beta baseline |

## Fixed (2026-09-04) — Phase 4 onboarding + doctor (RD-013/014) + RD-008B

| Priority | Issue | Resolution |
|----------|-------|------------|
| P0 | Onboarding ended at env export without proving operation | Connect→Select→Protect→Verify→Ready wizard; shared doctor Verify |
| P0 | No reusable operator diagnostics | `doctor` package + CLI `repository-detective doctor` + `/api/v1/doctor` + `/ui/doctor` |
| P1 | Class-B remediation isolation unclear before Phase 5 | RD-008B Option C — disabled by default; NOT_PROVEN disclosure; startup warn |

## Fixed (2026-09-04) — Phase 2 closure + Phase 3 privacy/security

| Priority | Issue | Resolution |
|----------|-------|------------|
| P0 | PR summary could duplicate on re-scan (RD-006A) | Idempotent upsert by marker; fail-closed on list failure; dedupe RD-owned duplicates |
| P0 | Disabled REQUIRED scanners could shrink evidence set → false POLICY_MET (RD-012A) | Profile-declared required set immutable; SKIPPED_BY_POLICY incomplete for REQUIRED |
| P1 | Privacy modes were badges only | `internal/privacy` LOCAL_ONLY/HYBRID/EXTERNAL_AI; AI + OpenClaw + notify gates |
| P1 | Threat model honesty | `docs/SECURITY_MODEL.md` with PROVEN/PARTIAL/NOT_* |
| P1 | Query API keys / auth migration | Prefer header; reject optional; recommend local session for new installs without silent flip |

## Fixed (2026-09-04) — Phase 1 public-beta blockers (RD-001/002/003/029)

| Priority | Issue | Resolution |
|----------|-------|------------|
| P0 | Public feedback path unclear / incomplete GitHub templates | GitHub Issues confirmed enabled; templates for bug/feature/install/scanner; SECURITY.md + private advisories; docs point one place for public bugs |
| P0 | Multiple install paths / 8080 vs 8081 / AI listed as required | Single **Recommended Installation** (compose pull, :8081); advanced options demoted; AI section optional everywhere |
| P0 | Docs/UI implied AI required despite runtime optional | Defaults (`DefaultGlobalSettings`), configure/onboarding/health copy, AI test-connection → Disabled not 503 |
| P3 | Stale doc claims (host-network default, enable_llm true examples) | Corrected NETWORKING/DOCKER/DEPLOYMENT; DOC_TRUTH_AUDIT.md |

## Fixed (2026-09-01) — Month-end audit: secret scanning, learning loop, disk

| Priority | Issue | Resolution |
|----------|-------|------------|
| CRITICAL | Root filesystem at **99%** (4.7 GB free); same condition that caused the 2026-07-22 scan/API outage at 96% | Pruned 50 dangling images, 4 redundant ~4 GB publish image copies, 86 dangling volumes, 25 stale `data/tmp` scratch dirs and the Go build cache. Now **74% / 63 GB free**. Legacy `bugbot.db` volumes archived to `data/legacy-volume-backups/` first |
| CRITICAL | **Secret scanning silently disabled on most repos.** `gitleaks_config` is the relative path `config/gitleaks.toml`, but gitleaks runs with the scan workspace as cwd, so it looked for the allowlist inside the repo under scan, aborted with "unable to load gitleaks config", exited 1 and wrote no report | Allowlist path resolved against the process working directory; an unusable path is dropped so scans fall back to default rules instead of failing. Hidden from dogfood because this repo *does* ship `config/gitleaks.toml` |
| HIGH | Clean gitleaks scans recorded as `parse_failed` (~1,100). gitleaks writes no report and logs only "no leaks found" to stderr | Output containing no JSON is an empty result; malformed JSON still fails. A command error is only a failure when no report was written, since gitleaks exits non-zero on real findings |
| HIGH | `scanner_failed` was the largest learning event type (9,434 / 14,305 = 66%), so calibration trained on failures that never happened | Root-caused to the two gitleaks bugs above. gitleaks went from 151 failures / 154 runs to 4 clean / 4 runs after the fix |
| HIGH | `gitleaks-history` failed on every private repo (~2,300) with `Authentication failed` | History clone now reuses the forge token like the remediation patcher; `sanitizeHistoryGitError` redacts it |
| HIGH | `sanitizeHistoryGitError` was `_ = out; return fmt.Errorf("git operation failed")`, discarding the diagnosis and leaving ~2,300 failures indistinguishable | Preserves git's message while redacting credentials in URLs and scratch workspace paths |
| HIGH | Calibration auto-apply could downgrade security findings: `IsProtectedFromAutoDowngrade` returns false on an empty category, and generated recommendations almost always have one. `SEC-EVAL`, `SEC-XSS-INNERHTML`, `CKV_SECRET_6` were all eligible | Protection matched on rule ID and source: `SEC-`/`CVE-`/`GHSA-`/`CKV_SECRET` prefixes, semgrep, and any rule naming a secret, credential, password or token |
| MEDIUM | 78 calibration recommendations at confidence 1.0 unapplied since 2026-08-02; 12,510 manual suppressions vs 188 learned rules | `calibration_auto_apply: true`; recompute applied **58** repo-scoped `report_only` rules and held the 5 security-sensitive ones for manual review. Accepted recommendations 22 → 92, active rules 188 → 203 |
| MEDIUM | A month of work uncommitted (22 modified + 7 new files), branch behind origin | Committed as 6 reviewable changesets, rebased onto the upstream CVE patch, pushed to Gitea and mirrored to GitHub |
| INFO | Compose binds `0.0.0.0:8081` for LAN access | Intentional for homelab; the API key is the only control on the LAN |

## Open (2026-08-02) — External architecture review + our audit merge

| Priority | Issue | Plan |
|----------|-------|------|
| HIGH | Live image Go toolchain older than `go.mod` 1.25 | Rebuild all-in-one from current Dockerfile |
| HIGH | Finding backlog ~11k open (crit/high still material) | Calibration / suppressions / focus triage |
| HIGH | Mega `main.go` (~2594 lines) + flat Config | Safe vertical split — see reconciliation report |
| MEDIUM | Was: viper `enable_llm_auditors` default `true` vs beta YAML `false` | **Fixed in tree** — default now `false` |
| INFO | External review praised deterministic-first / isolation / fingerprints | Preserve as invariants during refactor |

See [docs/dogfood-reports/external-review-reconciliation-2026-08-02.md](docs/dogfood-reports/external-review-reconciliation-2026-08-02.md).

## Fixed (2026-08-02) — Full application audit (accuracy / reliability / docs / wiki)

| Priority | Issue | Resolution |
|----------|-------|------------|
| HIGH | Dashboard `scanner_tools_missing_count` showed historical 10 while health was 12/12 | `ApplyPlatformReadiness` sets missing from live probes; API + System Health apply overlay |
| HIGH | ShellCheck 0.10 flat `[{…}]` JSON always `parse_failed` | Parser accepts flat + nested arrays |
| HIGH | SBOM `sbom_tool_missing` / cyclonedx fail left no artifact | Syft fallback after cyclonedx-gomod; keep SBOM when grype DB broken |
| HIGH | Wiki stubs / “not published” | Full wiki pages + API publish **23/23** |
| MEDIUM | OpenAPI wrong `repository_id`; missing repo scans route | `repo_id`, `GET /repos/{id}/scans`, calibration recompute, partial-spec banner |
| MEDIUM | Trivy trailing progress / empty stdout parse failures | `--output` report file + per-scan `--cache-dir` |
| MEDIUM | Misleading SBOM status when only grype DB failed | Status stays generated; detail notes grype DB |
| MEDIUM | Grype `malformed` during scans while CLI looked fine | Live `XDG_CACHE_HOME=/app/data/cache` held corrupt DB; rebuilt there (not `$HOME/.cache`) |
| LOW | Dogfood script `BASE` typo `8081}}` | Fixed |

## Fixed (2026-08-02) — Full UI eval residual contrast + clean re-pass

| Priority | Issue | Resolution |
|----------|-------|------------|
| MEDIUM | Configure dark mode: “missing” secret values low contrast inside translucent `details` | Secret status uses warn/completed badges; `details` + table headers use `--rd-surface-2` |
| LOW | Eval false-positive contrast on near-transparent rgba backgrounds | Harness ignores alpha &lt; 0.2 when sampling backgrounds |
| INFO | Re-eval after prior Bugbot/theme/print fixes | **36/36 OK** — see `docs/dogfood-reports/ui-flow-eval-2026-08-02.md` |

## Fixed (2026-08-02) — Syft / cyclonedx-gomod missing from live image

| Priority | Issue | Resolution |
|----------|-------|------------|
| HIGH | Health showed syft + cyclonedx-gomod as enabled/missing binary | Layered binaries into `repository-detective:rc-sbom-tools`; redeployed host-network container |
| MEDIUM | Soft `syft install skipped` could hide failures | `install-scanner-tools.sh` now fails the build if syft is absent; Dockerfile verifies both CLIs |

## Fixed (2026-08-02) — Full WebUI flow evaluation (theme / brand / charts)

| Priority | Issue | Resolution |
|----------|-------|------------|
| HIGH | Light-mode heading/KPI text used hardcoded white / translucent parents → unreadable | `--rd-heading` + solid `--rd-surface-2` for KPI/inset/report surfaces |
| HIGH | System Health showed historical `commstech/Bugbot` forge errors | `displayBrand` / `redactHealthText` scrub to Repository-Detective |
| HIGH | Findings showed `bugbot-scan-*` workspace paths | `displayPath` strips scratch prefixes + brand tokens |
| MEDIUM | Charts ignored theme toggles | dashboard/learning/repo-report charts remount on `rd-theme-change` |
| MEDIUM | Duplicate Policies nav; Learning buried | Policies removed; Learning under Intelligence; repo settings nav=`repos` |
| MEDIUM | Project groups showed raw repo IDs | Resolve primary/members to repository full names |
| LOW | Theme toggle did not update `colorScheme` | `theme.js` sets `colorScheme` + page background on apply |

## Fixed (2026-08-02) — Learning page missing Accept on recommendation tiles

| Priority | Issue | Resolution |
|----------|-------|------------|
| HIGH | Global calibration tiles showed only “accepts blocked” with no Accept/Reject | Show **Accept for affected repos** + Reject; accept expands into repo-scoped suppressions (never fleet-wide). Security/secret categories stay Reject-only with explanation. |

## Fixed (2026-08-02) — Dashboard / UI responsiveness

| Priority | Issue | Resolution |
|----------|-------|------------|
| HIGH | `/ui`, `/ui/health`, `/ui/reports` ~1.0–1.3s (warm) | Indexes on finding_instances/findings/scans/scanner_results; 2s DashboardSummary cache; windowed scanner rollups |
| HIGH | `/ui/repos` ~1.1s | Rewrote unmapped + active-present counts to use latest scan IDs / EXISTS (was full-table correlated aggregates) |
| MEDIUM | Dashboard chart N+1 category queries | `OpenFindingsByCategoryForRepositories` batch; load 20 repos not 200 |
| LOW | No repeatable page timing harness | `scripts/ui-responsiveness-bench.sh` + dogfood report |

## Fixed (2026-08-02) — External review: scanner reliability + SBOM + triage/export

| Priority | Issue | Resolution |
|----------|-------|------------|
| HIGH | parse_failed from stderr mixed into JSON (staticcheck/hadolint/trivy) | `runCommandStreams` prefers stdout; parsers extract JSON / strip ANSI |
| HIGH | Sequential scanners burn shared analysis timeout | `Registry.RunAll` runs scanners concurrently (ordered results) |
| HIGH | `sbom_tool_missing` / Syft not in image | Syft + cyclonedx-gomod installed in Docker/builder; health probes added |
| MEDIUM | 300s analysis timeout too aggressive under load | Defaults: analysis 900s, scanner 180s; operator `.env` aligned |
| MEDIUM | 11k findings lack prioritization / export | Findings focus list + severity sort; CSV/JSON export UI + API |
| MEDIUM | Parse failures not in dashboard actions | `BuildDashboardActions` links to health when parse_failed > 0 |
| LOW | Flat Config + mega main.go | Deferred — safe decomposition backlog |
| LOW | GitHub forge RC-unproven / LLM deep path | Deferred — beta keeps LLM off by default |

## Fixed (2026-08-02) — Gitea base sanitization + learning accept path

| Priority | Issue | Resolution |
|----------|-------|------------|
| HIGH | Calibration Accept always blocked (`IsProtectedFromAutoDowngrade("high", ...)`) | Accept validates by category/scope only; severity re-checked at scan persist |
| HIGH | Background calibration skipped repo-scoped generation | Job + recompute share `recomputeCalibration` (global + repo) |
| HIGH | Repo-scoped recommendation generation deadlocked under `SetMaxOpenConns(1)` | Collect candidates then insert after closing the SELECT cursor |
| MEDIUM | Learning UI read-only | Accept / Reject / Recompute forms on `/ui/learning` |
| MEDIUM | Operator LAN IPs in tracked reports | Redacted to localhost / example hosts |
| LOW | Setup clone path casing drift | Canonical `Repository-Detective` clone URL + privacy note |

## Fixed (2026-08-02) — Product rename to Repository Detective + Gitea sync

| Priority | Issue | Resolution |
|----------|-------|------------|
| HIGH | Uncommitted work not on Gitea | Committed and pushed to `commstech/Repository-Detective` |
| HIGH | Product still branded Repository-Detective for release | Module/repo/docs/UI renamed; public surfaces say Repository Detective |
| MEDIUM | Live deploy still uses legacy env/DB path | Silent compat kept (`envcompat`, `repository-detective.db`, fingerprint prefix) |

## Fixed (2026-08-02) — Invalid-ref mass failures were forge-outage misclassification

| Priority | Issue | Resolution |
|----------|-------|------------|
| HIGH | ~332 historical `no valid ref` failures polluted actionable metrics | Dashboard actionable = last 14d non-noise; unhealthy = latest-scan-failed repos |
| HIGH | Forge API outages reported as missing git refs | ResolveRef returns `unable to verify refs` when no definitive probe succeeds; classified as `forge_unavailable` |
| MEDIUM | Failure buckets showed lifetime history | Buckets + recent lists scoped to 14-day window |

## Fixed (2026-08-02) — Review follow-ups: stale scan noise + parse failure signal + AI UX

| Priority | Issue | Resolution |
|----------|-------|------------|
| HIGH | Failed-scan UI drowned in restart noise / unclear reasons | Classify `stale_reaped` vs `invalid_ref`; hide stale from primary lists; show failure buckets |
| HIGH | Parser failures not obvious to operators | Dashboard + health parse_failed counts; scan detail highlights parse/timeout/failed rows |
| MEDIUM | AI value prop hard to enable | Configure callout: choose **Deep** for LLM; dashboard CTA to Configure + Pre-install |

## Open / deferred (from external review 2026-08-02)

| Priority | Issue | Notes |
|----------|-------|-------|
| HIGH | Fleet `no valid ref` failures (~332) | **Fixed 2026-08-02** — mostly historical forge outage; metrics windowed + ResolveRef hardened |
| HIGH | Scanner parse_failed / timeouts under load | **Fixed 2026-08-02** — stdout-first parsers, parallel scanners, longer defaults; rebuild image for Syft |
| MEDIUM | Flat Config (~180 fields) + mega `main.go` | Safe decomposition backlog |
| MEDIUM | GitHub forge RC-unproven | Interface exists; parity unproven |
| MEDIUM | Syft / full SBOM coverage | **Fixed in source** — rebuild all-in-one image to pick up Syft/cyclonedx-gomod |
| MEDIUM | Interactive finding explorer + MTTR charts + exports | Focus list + CSV/JSON export shipped; MTTR charts still backlog |
| LOW | RBAC multi-operator | Beyond single API key / local login |
| LOW | Notifications default polish + learning calibration wiring | Ops enablement; learning accept UI shipped |
| LOW | Auto-rescan on remediation merge | evidence_closure exists; trigger polish |

## Fixed (2026-08-02) — System Health versions + failure drill-down + issue prefill

| Priority | Issue | Resolution |
|----------|-------|------------|
| HIGH | Scanner availability showed version `unknown` | Restored parallel version probes (stdout-aware, 12s timeout) with 5m cache |
| HIGH | Scanner run failures were a count only | Health page lists recent failures with Open scan links |
| MEDIUM | No way to file product issues from health problems | Report issue opens prefilled `system_health.md` on Repository-Detective Gitea (edit before submit) + Copy details |

## Fixed (2026-08-02) — Removed Qdrant completely

| Priority | Issue | Resolution |
|----------|-------|------------|
| HIGH | Qdrant integrated but unused (empty collection; semantic path gated off) | Deleted Qdrant package, embedder, config/env, and semantic dedup; rely on fingerprint + SQLite forge mappings |

## Fixed (2026-08-02) — Scan profile names were unintuitive

| Priority | Issue | Resolution |
|----------|-------|------------|
| HIGH | Profiles named `beta_standard`, `issue_only`, `standard_deterministic`, `fast`, etc. | Canonical **Light / Standard / Deep / Custom** with plain-language summaries; legacy IDs normalize automatically |

## Fixed (2026-08-02) — Learning page was a wall of text

| Priority | Issue | Resolution |
|----------|-------|------------|
| MEDIUM | `/ui/learning` showed dense tables instead of dashboard-style visuals | Stat cards + rate meters + Chart.js (events by type, noisiest rules) + recommendation cards |



| Priority | Issue | Resolution |
|----------|-------|------------|
| HIGH | Scan activity graph showed all activity on the newest day | Chart now aggregates completed scans from SQLite by UTC day for the full 14-day window (was only the ~10–50 “recent scans” list) |



| Priority | Issue | Resolution |
|----------|-------|------------|
| HIGH | After Start scan, UI felt frozen / no running-scan view | Modal now navigates straight to `/ui/scans/{id}`; running scans auto-refresh every 5s |
| HIGH | `/health` + UI stalled (~4s+) on scanner version probes; starved scan/UI requests | Presence-only tool checks, 5m cache, singleflight; repos fleet no longer calls full readiness/tools |
| MEDIUM | Scan links embedded cookie API key → extra 303 hop | Cookie sessions get clean scan URLs without `?api_key=` |



| Priority | Issue | Resolution |
|----------|-------|------------|
| HIGH | Configure page timed out (15–45s) because every GET ran full scanner version probes | Configure no longer calls `CheckTools`; tool probe results cached 60s elsewhere |
| HIGH | Browser never POSTed save when page hung | Sticky Save + fast page load; verified POST 303 `?saved=1` in <10ms |
| MEDIUM | Status tables looked unchanged after save | Badges now mirror saved form; green “saved” banner; docs collapsed; explain degraded = missing `.env` secrets |
| MEDIUM | Live apply missed preinstall/notify/remediation off toggles | `SetPreinstallEnabled`, `notify.SetEnabled`, remediation backends toggle off |



| Priority | Issue | Resolution |
|----------|-------|------------|
| HIGH | Gitleaks hits on unit-test secret-shaped samples | Runtime-constructed fixtures; `config/gitleaks.toml` allowlist; wired `gitleaks_config` |
| HIGH | CVE-2025-66471 urllib3 in benchmark fixture | Bumped fixture to urllib3 2.5.0 / requests 2.32.3 |
| HIGH | CVE-2026-39829 x/crypto (stale open finding) | Already on `v0.52.0`; clears on rescan |
| MEDIUM | Semgrep mutable `actions/checkout@v4` / `setup-go@v5` | Pinned to commit SHAs in all `.gitea/workflows` |
| MEDIUM | Ignored errors in openclaw/reconcile/closure/runner/learning | Propagate or handle errors; type-assert GitHub client safely |
| MEDIUM | Empty catch in onboarding `app.js` | Log warning instead of swallowing |
| LOW | Product-repo calibration gaps | Expanded `seed-product-repo-calibration.py`; added `closeout-repo1-findings.py` |

## Fixed (2026-08-01) — open Gitea issues + Go 1.25 toolchain

| Priority | Issue | Resolution |
|----------|-------|------------|
| HIGH | Gitea #352 CVE-2026-39829 in `golang.org/x/crypto` (< 0.52.0) | Bumped to `v0.52.0`; toolchain Go **1.25** (required by crypto); Dockerfile + CI/release workflows + build scripts aligned |
| MEDIUM | `issues/manager.go` non-constant `Errorf` format (vet fail on Go 1.25) | Switched to `logger.Error(errorMsg)` |
| MEDIUM | `TestEnsureScannerTempDir` leaked `TMPDIR` / `XDG_CACHE_HOME` and broke later scanner tests | Restore prior env in `t.Cleanup` |

## Fixed (2026-07-22) — long-running container ops health

| Priority | Issue | Resolution |
|----------|-------|------------|
| CRITICAL | Abandoned `grype-scratch*` / `getter*` filled container `/tmp` (~26GB) and host disk to 96%, correlating with scan/API failures | Entrypoint + startup cleanup; durable `TMPDIR` under `/app/data/tmp`; forward `TMPDIR`/`XDG_CACHE_HOME` in `MinimalSubprocessEnv` |
| HIGH | Grype DB malformed → scanner_unavailable | Startup `WarmGrypeDB`; operator cleanup of bad cache |
| HIGH | Qdrant embedding 400 (`Invalid model`) + 1024 vs 768 dim mismatch with OpenClaw | Auto `openclaw` embedding model; vector size 768; recreate collection on size mismatch |
| MEDIUM | Scheduled scans `no valid ref found` | ResolveRef branch-list fallback + richer errors; scheduled path resolves/persists default branch before analyze |

## Fixed (2026-07-13) — full scanner image never installed tools

| Priority | Issue | Resolution |
|----------|-------|------------|
| CRITICAL | `scripts/apk-retry.sh` ran `apk add` + `exit 0` when sourced, so `install-scanner-tools.sh` exited after an empty `apk add` and never installed trivy/grype/semgrep/etc. Live `/health` stayed at `available_count=4` with six tools missing. | Rewrote `apk-retry.sh` to define `apk_retry()` and only auto-run when executed as a script. Rebuilt `repository-detective:rc-04db228` (~3.87GB); redeployed; `tools_summary` now 10/10. |

## Review Findings (2026-05-30)

### Fixed in this session

| Priority | Issue | Resolution |
|----------|-------|------------|
| CRITICAL | Project did not compile (`ID::` syntax error, broken imports, circular deps) | Fixed syntax, aligned module path to `git.commsnet.org/commstech/repository-detective`, extracted shared types to `models/` |
| CRITICAL | Webhook auth in `main.go` used plain string compare; rate limiting unused | Consolidated on `handlers.WebhookHandler` with HMAC-safe secret check and per-IP rate limiting |
| HIGH | `go.mod` import path mismatch (`github.com/yourusername/...` vs `yourusername/...`) | Unified on Gitea module path |
| HIGH | Missing `go.sum` | Generated via `go mod tidy` |
| HIGH | Gitea file content returned base64-encoded but never decoded | Added base64 decode in `GetFileContent` |
| MEDIUM | `Repository.DefaultBranch` missing — `ListAllFiles` could fail | Added field to struct |
| MEDIUM | API endpoints allowed requests when `api_key` empty | Reject with 503 when unset |
| MEDIUM | Raw string backticks in `createAnalysisPrompt` broke Go parser | Replaced markdown fences with plain delimiters |
| LOW | Duplicate webhook/analysis logic between `main.go` and `handlers/` | `AnalysisProcessor` interface wires engine into secure handler |

### Completed improvements (2026-05-30, session 2)

| Item | Resolution |
|------|------------|
| SCAN stage missing file content | Fetch via `GetFileContent`, include in auditor prompts |
| `enable_security` / `enable_quality` unused | Wired in static scan + LLM gate |
| Repo include/exclude patterns unused | `handlers/repo_filter.go` + webhook filter |
| Issue labels empty | `gitea.ResolveLabelIDs` |
| PoC/file/line missing from issues | Mapped from Prove stage in `analysisResultFromReport` |
| No deterministic pre-scan | `analyzers/static.go` before LLM |
| Docker compose env mismatch | Fixed `docker-compose.minimal.yml` |
| No onboarding UI | `/onboard` wizard + API |
| Documentation outdated | README, QUICK_SETUP, DEPLOYMENT, architecture, status, docs/ONBOARDING |

### Open / follow-up

| Priority | Issue | Notes |
|----------|-------|-------|
| LOW | Gitea #48 Ops: homelab AI/Qdrant connectivity from Docker | **Closed 2026-08-02** — Qdrant removed from product; fingerprint + SQLite dedup only |
| LOW | `WebhookHandler` still allows empty secret (logs warning only) | Consider failing closed in production mode |
| LOW | Full `./scanners` suite can timeout when live `grype db` warmup runs | Unit subset passes; consider skipping network warmup under `testing.Short()` |
| LOW | Integration tests with mocked Gitea/OpenWebUI | Unit tests added for core logic |

## Common Challenges to Watch For

- Gitea plugin API compatibility
- OpenWebUI API integration complexity
- Multi-language code analysis accuracy
- Performance optimization for large repositories
- Error handling and logging

## 2026-08-02 release readiness notes
- Gitleaks RuleID/fingerprint stability fixed (temp workspace paths no longer create duplicate highs).
- AI recommendations remain disabled by default with tight token/CAH budgets when enabled.
- Fleet noise calibrated (docs/archive/example/vendor + report-only categories).
- Remaining high/critical mostly real: SEC-EVAL in AI assistants, dependency/misconfig in apps not yet rescanned clean.
- Rotate credentials that were previously committed in House_Grocery_AI history.
