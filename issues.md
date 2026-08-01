# Development Issues Log

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
| CRITICAL | Project did not compile (`ID::` syntax error, broken imports, circular deps) | Fixed syntax, aligned module path to `git.commsnet.org/commstech/bugbot`, extracted shared types to `models/` |
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
