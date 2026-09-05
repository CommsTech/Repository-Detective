# Working tree cleanup report

**Date:** 2026-06-04 (UTC)  
**Branch:** `main`  
**Operator:** Repository Detective beta readiness checkpoint

---

## Summary

Cleaned ~128 modified/untracked paths into committed product artifacts, gitignored local-only items, and excluded private dogfood from the repo. RuView dogfood work was **not** extended; only prior sanitized RuView docs remain on `main`.

---

## Classification

### commit_now (staged and committed)

**Core product**

- Scoring fix: `analyzers/scoring.go`, tests, engine wiring
- Theme persistence: `ui/static/theme.js`, `theme.css`, `layout.html`, `ui/theme_static_test.go`
- Qdrant redaction path: `memory/qdrant/redaction.go`, `payload.go`, `client.go`, tests; `redact/secrets.go`
- Calibration: `calibration/`, `api/calibration_handler.go`, `main_calibration.go`, store/dashboard
- Suppressions + reconcile: `store/suppressions*.go`, `api/suppressions_handler.go`, `api/reconcile_handler.go`, UI reconcile
- Pre-install audit: `preinstall/config.go`, `reports.go`, runner updates
- AI status surface: `ai/status.go`, `api/ai_handler.go`, `main_ai.go`
- Gitleaks deterministic result helper: `scanners/deterministic_result.go`
- Runner telemetry: `operator/runner_telemetry.go`
- Graph detail, closure tests, profile/reporting updates

**Docs**

- `docs/BETA_READINESS.md`
- `docs/BACKUP_RESTORE.md`, `CALIBRATION.md`, `DOCKER.md`, `ISSUE_RECONCILIATION.md`, `UPGRADE.md`
- `docs/AI_TOKEN_EFFICIENCY.md`, operator/policy/UI doc updates
- `docs/examples/docker-compose.yml`

**Scripts (sanitized, portable)**

- `scripts/docker-build-verify.sh`
- `scripts/docker-healthcheck.sh`
- `scripts/install-scanner-tools.sh`
- `scripts/generate-scan-quality-report.sh`

**Config / deploy**

- `.env.example`, `config/config.yaml.example` (private beta defaults)
- `.gitignore` (local artifacts, vendor, private dogfood)
- `deploy.sh`, `docker-compose.yml`, `docker-compose.minimal.yml`

### keep_local_only (not committed)

| Path | Reason |
|------|--------|
| `.env` | Secrets / operator tokens |
| `config/config.yaml` | Homelab URLs and secrets (gitignored) |
| `data/repository-detective.db` | Local SQLite (gitignored) |
| `deployment-backups/` | Pre-image-recreate DB backup (gitignored) |
| `restore-drill-test/` | Operator drill workspace (gitignored) |
| `docs/DOGFOOD_REPORT_FIRST_39_REPOS.md` | Unsanitized private dogfood |
| `vendor/` | Regenerate via `scripts/vendor-deps.sh` when needed |

### delete_generated

None required this pass — no stray binaries or Qdrant dumps in the working tree.

### dogfood_report_sanitized (already on main)

- `docs/dogfood-reports/qdrant-redacted-local-test.md`
- `docs/dogfood-reports/qdrant-redacted-test-result.json`
- `docs/dogfood-reports/ruview-*` (shareable report, triage, compare, diagnosis)

### dogfood_report_private (local / gitignored)

- `docs/DOGFOOD_REPORT_FIRST_39_REPOS.md`
- `docs/dogfood-reports/*` except whitelisted sanitized files (repo closeout JSON/CSV, scan-quality dumps, closeout SQL)

### needs_review

| Item | Decision |
|------|----------|
| Operator closeout Python scripts | **keep_local_only** — added to `.gitignore` |
| Theme/scoring on `origin/main` before this commit | **Missing** — included in this commit |
| `/health` ~4s latency | Documented in `BETA_READINESS.md` as nice-to-have |

---

## Main branch commit verification (pre-push)

| Required commit | SHA | On `origin/main` |
|-----------------|-----|------------------|
| Beta blocker fixes | `79cae24` | Yes |
| Direct remediation closure API | `0732ff6` | Yes |
| Qdrant sanitized docs | `ec1ceb5` | Yes |
| RuView polished docs | `f7687e5` | Yes |
| Gitleaks report-file fix | `957421f` | Yes |
| Theme persistence fix | *(this commit)* | Pending push |
| Scoring fix | *(this commit)* | Pending push |

---

## Quality gates (2026-06-04)

| Command | Result |
|---------|--------|
| `go test ./...` | Pass |
| `go vet ./...` | Pass |
| `staticcheck ./...` | Pass |
| Migration 16 | Applied on production DB |
| `./scripts/docker-build-verify.sh` | Run at operator discretion before release tag |

---

## Private beta go/no-go

**Go** for single-operator homelab private beta after this commit is pushed, with API-key auth and limitations documented in `docs/BETA_READINESS.md`.

**No-go** for multi-tenant SaaS until Auth/RBAC, tenant isolation, and billing.

---

## Next engineering phase

1. Auth/RBAC design → single-admin login → org/team model  
2. Operator onboarding docs polish  
3. Paid manual assessment SOP (pre-install + human review)
