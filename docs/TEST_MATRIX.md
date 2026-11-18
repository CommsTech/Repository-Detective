# Test matrix — Repository Detective private beta

**Product:** Repository Detective — Inspect. Analyze. Improve.  
**Purpose:** Regression and operational coverage before handing to another operator.  
**Legend:** ✅ pass (verified) · ⚠️ partial · 🔲 manual · 🤖 automated · 📋 documented procedure

Run automated baseline:

```bash
./scripts/release-test.sh
./scripts/operator-smoke-test.sh   # against running instance
```

End-to-end operator flow: [BETA_SMOKE_TEST.md](BETA_SMOKE_TEST.md)

---

## Core

| Test | Type | Status | Notes |
|------|------|--------|-------|
| Startup — container exits cleanly on bad config | 🤖 | ✅ | Missing forge token fails fast |
| Startup — healthy after components init | 🤖 | ✅ | `/health` → `healthy` |
| Config loading — YAML + env merge | 🤖 | ✅ | `config/config.yaml` + `.env` |
| Env alias — `REPOSITORY_DETECTIVE_*` wins over `REPOSITORY_DETECTIVE_*` | 🤖 | ✅ | `internal/config/envcompat` tests |
| API auth — preferred header | 🤖 | ✅ | `TestRequireAPIKeyAuthAcceptsPreferredAndLegacyHeaders` |
| API auth — legacy `X-Repository-Detective-API-Key` | 🤖 | ✅ | Same test |
| API auth — missing key → 401 | 🤖 | ✅ | `api/security_test.go` |
| DB migrations — fresh install | 🤖 | ✅ | `store` migration tests; schema v16 |
| DB migrations — existing install upgrade | 🔲 | ⚠️ | Operator: backup → pull → start; see [UPGRADE.md](UPGRADE.md) |
| Backup/restore — SQLite file | 🔲 | ✅ | [BACKUP_RESTORE.md](BACKUP_RESTORE.md) drill documented |

---

## Scanning

| Test | Type | Status | Notes |
|------|------|--------|-------|
| Connected repo manual scan (`POST /api/v1/analyze`) | 🔲 | 📋 | [BETA_SMOKE_TEST.md](BETA_SMOKE_TEST.md) step 5 |
| Scheduled scan (cron) | 🔲 | 📋 | [SCHEDULER.md](SCHEDULER.md) |
| Push webhook | 🔲 | 📋 | Requires public URL + Gitea delivery |
| PR webhook | 🔲 | 📋 | Same |
| Scanner status in `/health` / `/api/v1/status` | 🤖 | ✅ | `operator-smoke-test.sh` |
| Scanner missing binary — degraded not crash | 🤖 | ⚠️ | Core image without tools; scan continues |
| Scanner timeout | 🤖 | ⚠️ | Per-scanner timeout config; unit tests partial |
| Scanner parse failure → `parse_failed` not silent pass | 🤖 | ✅ | gitleaks report-file fix; deterministic_result tests |

---

## Scanners (deterministic)

| Scanner | Unit/parser tests | Live binary (all-in-one) |
|---------|-------------------|--------------------------|
| trivy | 🤖 partial | 🔲 `/health` tools_summary |
| grype | 🤖 partial | 🔲 |
| gitleaks | 🤖 ✅ | 🔲 8.21.2 report file |
| semgrep | 🤖 partial | 🔲 |
| govulncheck | 🤖 partial | 🔲 |
| gosec | 🤖 partial | 🔲 |
| staticcheck | 🤖 partial | 🔲 |
| hadolint | 🤖 partial | 🔲 |
| checkov | 🤖 partial | 🔲 intermittent timeouts |
| health checks (repo) | 🤖 | 🔲 preinstall/checks |
| graph checks | 🤖 | 🔲 [CODE_GRAPH.md](CODE_GRAPH.md) |

Verify all binaries in container:

```bash
docker exec repository-detective sh -c 'for t in trivy grype gitleaks semgrep govulncheck gosec staticcheck hadolint checkov; do command -v $t || echo MISSING:$t; done'
```

---

## UI

| Area | Type | Status | Notes |
|------|------|--------|-------|
| Dashboard | 🔲 | 📋 | Repo counts, calibration block |
| Findings list / detail | 🔲 | 📋 | Severity, suppression state |
| Scans history | 🔲 | 📋 | |
| Repos list | 🔲 | 📋 | |
| Repo settings | 🔲 | 📋 | Profile, gates, AI policy |
| Graph / repository map | 🔲 | 📋 | `ui/static/graph.js` — preferred API header |
| Pre-install audit UI | 🔲 | 📋 | On-demand when enabled |
| Remediation plan UI | 🔲 | 📋 | Planner on; PRs off by default |
| Closure evidence UI | 🔲 | 📋 | |
| Theme system/light/dark | 🤖 | ✅ | `ui/theme_static_test.go` |
| Accessibility basics | 🔲 | ⚠️ | [ACCESSIBILITY.md](ACCESSIBILITY.md) checklist |

---

## Issue lifecycle

| Test | Type | Status | Notes |
|------|------|--------|-------|
| Issue creation (Gitea) | 🔲 | 📋 | Token + `auto_create_issues` |
| Duplicate avoidance (fingerprint) | 🤖 | ✅ | `issues/fingerprint_test.go` |
| Suppression create/apply | 🤖 | ✅ | `store/suppressions_test.go` |
| False positive workflow | 🔲 | 📋 | [FALSE_POSITIVES.md](FALSE_POSITIVES.md) |
| Reconciliation preview | 🤖 | ⚠️ | API + engine tests |
| Reconciliation apply | 🔲 | 📋 | UI + API |
| Remediation plan generation | 🔲 | 📋 | `remediation_planner_enabled: true` |
| Safe remediation PR | 🔲 | 📋 | **Off by default** — enable only when ready |
| Merged PR + rescan | 🔲 | 📋 | [EVIDENCE_CLOSURE.md](EVIDENCE_CLOSURE.md) |
| Evidence closure | 🔲 | 📋 | `close_issues: false` default |

---

## Pre-install audit

| Test | Type | Status | Notes |
|------|------|--------|-------|
| Safe HTTPS public repo | 🔲 | 📋 | [PREINSTALL_AUDIT.md](PREINSTALL_AUDIT.md) |
| Invalid URL rejected | 🤖 | ✅ | `preinstall/url_security_test.go` |
| localhost/private IP blocked | 🤖 | ✅ | SSRF tests |
| Large repo limits | 🤖 | ⚠️ | Size/file caps in config |
| Disclosure report generation | 🔲 | 📋 | Markdown drafts |
| Public/private draft separation | 🔲 | 📋 | No secrets in shareable report |
| No auto-submit to upstream | 🤖 | ✅ | Policy — manual share only |

---

## Security / privacy

| Test | Type | Status | Notes |
|------|------|--------|-------|
| No secrets in logs (subprocess env) | 🤖 | ✅ | `scanners/exec_security_test.go` |
| No tokens in API JSON responses | 🤖 | ⚠️ | Spot-check `/api/v1/status`; smoke script greps |
| Qdrant removed | 🤖 | ✅ | no qdrant package/config |
| AI startup test disabled by default | 🤖 | ✅ | `ai_startup_test_enabled: false` |
| Redaction in reports | 🤖 | ✅ | `redact/` package |
| Third-party reports sanitized | 🔲 | ✅ | Dogfood RuView package committed sanitized |

---

## Docker

| Test | Type | Status | Notes |
|------|------|--------|-------|
| core image build | 🤖 | ✅ | `docker-build-verify.sh` |
| runner image build | 🤖 | ✅ | Same |
| all-in-one image build | 🤖 | ✅ | Same |
| HEALTHCHECK script | 🤖 | ✅ | `docker-healthcheck.sh` |
| Non-root user (all-in-one) | 🤖 | ✅ | UID 1001 `repositorydetective` |
| Persistent data volume | 🔲 | 📋 | `./data:/app/data` |
| Config mount read-only | 🔲 | 📋 | `./config:/app/config:ro` |
| No Docker socket mounted | 🔲 | ✅ | Default compose — verify no `/var/run/docker.sock` |
| No privileged mode | 🔲 | ✅ | Default compose |

---

## Automated commands summary

| Command | Covers |
|---------|--------|
| `go test ./...` | Unit + integration tests across packages |
| `go vet ./...` | Static analysis |
| `staticcheck ./...` | Additional lint |
| `./scripts/docker-build-verify.sh` | Disk preflight (10 GB min) + all image targets + smoke `/health` |
| `./scripts/release-test.sh` | All of the above + optional gosec |
| `./scripts/operator-smoke-test.sh` | Live instance API + health |

---

## Gaps before external operator (track during beta week)

- Full webhook E2E on operator's Gitea (manual)
- Scheduled scan fire-on-time (manual)
- Remediation PR merge + closure loop (manual, optional)
- checkov/grype timeout edge cases (nice-to-have)
- `/health` latency ~4s with full scanner probes (documented)

See [KNOWN_LIMITATIONS.md](KNOWN_LIMITATIONS.md) and [BETA_READINESS.md](BETA_READINESS.md).
