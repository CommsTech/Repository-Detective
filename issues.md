# Development Issues Log

## Fixed (2026-08-02) — Product rename to Repository Detective + Gitea sync

| Priority | Issue | Resolution |
|----------|-------|------------|
| HIGH | Uncommitted work not on Gitea | Committed and pushed to `commstech/repository-detective` |
| HIGH | Product still branded Bugbot for release | Module/repo/docs/UI renamed; public surfaces say Repository Detective |
| MEDIUM | Live deploy still uses legacy env/DB path | Silent compat kept (`envcompat`, `bugbot.db`, fingerprint prefix) |

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
| MEDIUM | Flat Config (~180 fields) + mega `main.go` | Safe decomposition backlog |
| MEDIUM | GitHub forge RC-unproven | Interface exists; parity unproven |
| MEDIUM | Syft / full SBOM coverage | Partial CycloneDX gomod fallback today |
| MEDIUM | Interactive finding explorer + MTTR charts + exports | UI expansion |
| LOW | RBAC multi-operator | Beyond single API key / local login |
| LOW | Notifications default polish + learning calibration wiring | Ops enablement |
| LOW | Auto-rescan on remediation merge | evidence_closure exists; trigger polish |

## Fixed (2026-08-02) — System Health versions + failure drill-down + issue prefill

| Priority | Issue | Resolution |
|----------|-------|------------|
| HIGH | Scanner availability showed version `unknown` | Restored parallel version probes (stdout-aware, 12s timeout) with 5m cache |
| HIGH | Scanner run failures were a count only | Health page lists recent failures with Open scan links |
| MEDIUM | No way to file product issues from health problems | Report issue opens prefilled `system_health.md` on Bugbot Gitea (edit before submit) + Copy details |

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
| LOW | Gitea #48 Ops: homelab AI/Qdrant connectivity from Docker | Ops/infra — not a code defect; document runbook or soft-fail already covers |
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
