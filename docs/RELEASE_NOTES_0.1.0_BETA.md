# Release notes — 0.1.0-beta

**Product:** Repository Detective — Inspect. Analyze. Improve.  
**Tag:** `v0.1.0-beta` (recommended)  
**Date:** 2026-06-04  
**Branch:** `main`

Private beta release for **single-operator Gitea** deployments. Not SaaS-ready.

---

## What works

- **Gitea connected repo scanning** — push/PR webhooks, manual scan API
- **Deterministic scanners** — trivy, grype, gitleaks, semgrep, govulncheck, gosec, staticcheck, hadolint, checkov (all-in-one image)
- **Dashboard UI** — findings, repos, scans, repo settings, graph, calibration
- **Issue lifecycle** — creation, fingerprint dedup, suppressions, reconciliation preview/apply
- **Remediation planner** — plan generation (PR creation optional)
- **Evidence closure** — verify-after-fix workflow (no auto-close by default)
- **Pre-install audit** — on-demand third-party repo assessment (when enabled)
- **Scoring** — non-zero scores when findings exist
- **Theme persistence** — system / light / dark
- **Backup/restore** — SQLite file + documented drill
- **Docker all-in-one** — non-root, healthcheck, persistent volume
- **Legacy compatibility** — `REPOSITORY_DETECTIVE_*` env, `X-Repository-Detective-API-Key`, `repository-detective-*` fingerprints

---

## Intentionally disabled by default (beta-safe)

| Setting | Default | Why |
|---------|---------|-----|
| `remediation_pr_enabled` | `false` | No unsupervised auto-PRs |
| `evidence_closure_close_issues` | `false` | Human stays in loop |
| `qdrant_enabled` | `false` | Not production-ready (embedding/UUID) |
| `ai_startup_test_enabled` | `false` | Avoid paid AI probe on boot |
| `enable_llm_auditors` | `false` | Deterministic-first beta |
| `preinstall_audit_enabled` | `false` | Enable only for manual audits |
| `runner_delegation_enabled` | `false` | Single container default |
| `notifications_enabled` | `false` | Optional |

Recommended profile: **`beta_standard`** — see [BETA_READINESS.md](BETA_READINESS.md).

---

## Known limitations

- **Single API-key auth** — no multi-user RBAC ([AUTH_RBAC_PLAN.md](AUTH_RBAC_PLAN.md))
- **SQLite** — not HA; single tenant
- **`/health` latency** — ~4s when probing all scanners
- **Issue dedup** — fingerprint + SQLite forge mappings (Qdrant removed)
- **GitHub/GitLab** — not first-class connected-repo parity
- **checkov / grype** — occasional timeouts on large repos
- **No billing / license enforcement** — [EDITIONS.md](EDITIONS.md) is planning only

Full list: [KNOWN_LIMITATIONS.md](KNOWN_LIMITATIONS.md)

---

## Upgrade notes

1. Backup `data/repository-detective.db` — [BACKUP_RESTORE.md](BACKUP_RESTORE.md)
2. Pull `main` or checkout tag
3. `docker compose build repository-detective && docker compose up -d --force-recreate repository-detective`
4. Migrations apply automatically on startup (schema v16+)
5. Run `./scripts/operator-smoke-test.sh`

From older Repository-Detective installs: legacy env vars and labels continue to work — [BRANDING_MIGRATION.md](BRANDING_MIGRATION.md).

---

## Safety defaults

- Remediation PRs require explicit enable + approval workflow
- Pre-install: SSRF blocks, no dependency install, sanitized shareable reports
- Subprocesses use minimal env (no forge tokens in scanner children)
- API prefers `X-Repository-Detective-API-Key`; secrets not returned in `/api/v1/status`

See [SECURITY_HARDENING.md](SECURITY_HARDENING.md).

---

## Beta feedback requested

During beta freeze week, please log:

- New issues / false positives
- Scanner failures and parse errors
- Reconciliation accuracy
- Score accuracy
- UI friction and theme bugs
- Scan duration

Use [BETA_SMOKE_TEST.md](BETA_SMOKE_TEST.md) for structured validation.

---

## Edition / licensing

- **Community** (default today) — [COMMUNITY_EDITION.md](COMMUNITY_EDITION.md)
- **Commercial / Enterprise** — planned feature gates; not enforced
- Proposed license: AGPL Community + paid commercial — [LICENSING_STRATEGY.md](LICENSING_STRATEGY.md)

---

## Legacy compatibility

Repository Detective is the product name. Repository-Detective remains supported:

- `REPOSITORY_DETECTIVE_*` environment variables
- `X-Repository-Detective-API-Key` header
- `repository-detective/*` labels (read); `repository-detective/*` (write default)
- `rd-<hex>` fingerprint values
- `data/repository-detective.db` database filename

---

## Verification commands

```bash
./scripts/release-test.sh
export REPOSITORY_DETECTIVE_API_KEY='your-key'
./scripts/operator-smoke-test.sh
```

Test coverage map: [TEST_MATRIX.md](TEST_MATRIX.md)
