# Feature completeness audit

**Product:** Repository Detective — Inspect. Analyze. Improve.  
**Date:** 2026-06-05  
**Type:** Inventory and gap audit — **not** a feature build

Status key: **implemented** · **partial** · **documented only** · **planned** · **legacy**

---

## Executive summary

| Metric | Count |
|--------|------:|
| Implemented (beta-ready) | ~85% of claimed Community private-beta scope |
| Partial | GitHub scanning, runner delegation, notifications, remediation PRs |
| Planned only | Auth/RBAC, license enforcement, billing, tenant isolation, community intelligence feed |
| Private beta go/no-go | **GO** |

---

## Part A — Feature inventory

### Core

| Feature | Status | Notes |
|---------|--------|-------|
| Gitea connected repo scanning | **implemented** | Webhooks + manual |
| Push webhooks | **implemented** | Gitea only |
| PR webhooks | **implemented** | Gitea only |
| Manual scans | **implemented** | `POST /api/v1/analyze` |
| Scheduled scans | **implemented** | `scheduler_enabled`; cron per repo |
| Full repo / archive workspace | **implemented** | `workspace_mode` |
| Per-repo settings | **implemented** | UI + API |
| Scan profiles | **implemented** | `beta_standard`, etc. |
| Scanner registry | **implemented** | 10+ tools |
| Status / health endpoints | **implemented** | `/health`, `/api/v1/status` |
| Dashboard | **implemented** | UI + API summary |
| SQLite persistence | **implemented** | `data/repository-detective.db` |
| Backup / restore | **implemented** | [BACKUP_RESTORE.md](BACKUP_RESTORE.md) |
| Docker packaging | **implemented** | core / runner / all-in-one |

### Scanners

| Scanner | Impl | Config | Env | Per-repo | Status UI | Docs | Tests | All-in-one binary |
|---------|:----:|:------:|:---:|:--------:|:---------:|:----:|:-----:|:-----------------:|
| trivy | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| grype | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| gitleaks | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ 8.21.2 |
| semgrep | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | partial | ✅ |
| govulncheck | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | partial | ✅ |
| gosec | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | partial | ✅ |
| staticcheck | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | partial | ✅ |
| hadolint | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | partial | ✅ |
| checkov | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | partial | ✅ |
| linters (go/py/sh) | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | partial | ✅ |
| health checks | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | partial | N/A (built-in) |
| graph checks | **partial** | ✅ | — | ✅ | ✅ | ✅ | partial | N/A |

### Analysis / quality

| Feature | Status | Notes |
|---------|--------|-------|
| Repository Map | **implemented** | `/ui/repos/:id/graph` |
| Graph finding overlay | **implemented** | Finding detail links |
| Disconnected code detection | **partial** | Graph orphans; not full IDE parity |
| Health checks | **implemented** | Depth ≥ 2 |
| Tech debt checks | **implemented** | Profile-gated |
| Reliability checks | **implemented** | |
| Maintainability checks | **implemented** | |
| Test gap checks | **implemented** | |
| Performance checks | **implemented** | |
| AI risk checks | **partial** | Off in beta (`enable_ai_risk_checks` / profile) |

### Issue lifecycle

| Feature | Status |
|---------|--------|
| Issue creation (Gitea) | **implemented** |
| GitHub issue creation | **partial** — manual scan repos only |
| Duplicate prevention | **implemented** — fingerprints |
| Labels | **implemented** — `repository-detective/*` default |
| Fingerprinting | **implemented** — `rd-<hex>` values |
| Suppression | **implemented** |
| False positive marking | **implemented** |
| Reconciliation preview/apply | **implemented** |
| Calibration | **implemented** |
| Score calculation | **implemented** |
| Issue enrichment | **partial** — LLM when enabled |

### Remediation

| Feature | Status |
|---------|--------|
| Remediation planner | **implemented** — on by default |
| Approval / rejection | **implemented** |
| Safe remediation PRs | **partial** — **off by default** |
| Validation allowlist | **implemented** |
| Evidence closure | **implemented** |
| No auto-merge | **implemented** |
| No auto-close by default | **implemented** |

### Pre-install audit

| Feature | Status |
|---------|--------|
| HTTPS validation | **implemented** |
| SSRF / private IP block | **implemented** |
| Shallow clone | **implemented** |
| Deterministic scanners | **implemented** |
| Risk score | **implemented** |
| Disclosure drafts | **implemented** |
| Public/private report split | **implemented** |
| No auto-submit | **implemented** |
| Report footer / project link | **implemented** |
| Enabled by default | **partial** — **false** in beta config |

### Integrations

| Feature | Status |
|---------|--------|
| Gitea | **implemented** — primary |
| Gitea status checks | **implemented** — optional |
| Gitea issues / PRs | **implemented** |
| GitHub scanning | **partial** — [GITHUB_SCANNING.md](GITHUB_SCANNING.md); no webhooks/PR |
| GitLab | **planned** |
| Notifications | **partial** — implemented; **off by default** |
| Runner delegation | **partial** — implemented; **off by default** |
| Qdrant | **removed** — fingerprint dedup only |
| AI provider status | **implemented** |

### UI

| Area | Status |
|------|--------|
| Dashboard, repos, settings, findings, scans | **implemented** |
| Graph, pre-install, remediation, closure | **implemented** / gated |
| Suppressions, calibration, reconcile | **implemented** |
| Notifications, runner jobs UI | **partial** — when enabled |
| Theme system/light/dark | **implemented** |
| Accessibility baseline | **partial** — [ACCESSIBILITY.md](ACCESSIBILITY.md) |

### Packaging / ops

| Item | Status |
|------|--------|
| Docker core / runner / all-in-one | **implemented** |
| docker-compose, healthcheck, non-root | **implemented** |
| release-test.sh, operator-smoke-test.sh | **implemented** |
| Beta docs (QUICKSTART, TEST_MATRIX, etc.) | **implemented** |

### Product / licensing

| Item | Status |
|------|--------|
| Branding compatibility | **implemented** |
| Edition / licensing docs | **documented only** — no enforcement |
| Auth/RBAC plan | **planned** |
| Monetization readiness | **documented only** |

---

## Part B — Claim verification

| Claim | Verdict | Correction |
|-------|---------|------------|
| Automatically fixes issues | **true with caveat** | Planner + optional PRs; **not** auto-merge; PRs **off** by default |
| Supports GitHub/GitLab | **true with caveat** | GitHub **partial** (manual/bulk); GitLab **planned**; Gitea **full** |
| Uses Qdrant | **false** | Removed |
| Multi-user | **stale/incorrect** | API-key only; Auth/RBAC **planned** |
| Evidence-based closure | **true** | Verify + optional comments; no auto-close default |
| Pre-install audit | **true** | On-demand; **disabled** in beta defaults |
| Safe remediation PRs | **true with caveat** | Implemented; **disabled** by default |
| Community intelligence feed | **planned only** | [PRIVACY.md](PRIVACY.md) — not implemented |
| Multi-tenant SaaS | **planned only** | Documented as not ready |
| License enforcement | **planned only** | Docs only |

**Docs fixed this audit:** README caveats; SCANNERS.md product name + all-in-one binary notes.

---

## Part C — Config matrix (beta-critical keys)

Full keys in `config/config.yaml.example` and `.env.example`. See [CONFIGURATION.md](CONFIGURATION.md).

| Key | Preferred env | Beta default | Safe beta | Docs |
|-----|---------------|--------------|-----------|------|
| `api_key` | `REPOSITORY_DETECTIVE_API_KEY` | (set in `.env`) | required | CONFIGURATION |
| `scan_profile` | `REPOSITORY_DETECTIVE_SCAN_PROFILE` | `beta_standard` | ✅ | SCAN_PROFILES |
| `enable_llm_auditors` | `REPOSITORY_DETECTIVE_ENABLE_LLM_AUDITORS` | `false` | ✅ | POLICY |
| `remediation_pr_enabled` | `REPOSITORY_DETECTIVE_REMEDIATION_PR_ENABLED` | `false` | ✅ | REMEDIATION_PRS |
| `evidence_closure_close_issues` | `REPOSITORY_DETECTIVE_EVIDENCE_CLOSURE_CLOSE_ISSUES` | `false` | ✅ | EVIDENCE_CLOSURE |
| `preinstall_audit_enabled` | `REPOSITORY_DETECTIVE_PREINSTALL_AUDIT_ENABLED` | `false` | ✅ | PREINSTALL_AUDIT |
| `ai_startup_test_enabled` | `REPOSITORY_DETECTIVE_AI_STARTUP_TEST_ENABLED` | `false` | ✅ | AI_TOKEN_EFFICIENCY |
| `notifications_enabled` | `REPOSITORY_DETECTIVE_NOTIFICATIONS_ENABLED` | `false` | ✅ | NOTIFICATIONS |
| `runner_delegation_enabled` | `REPOSITORY_DETECTIVE_RUNNER_DELEGATION_ENABLED` | `false` | ✅ | RUNNERS |
| `enable_trivy` … `enable_checkov` | `REPOSITORY_DETECTIVE_ENABLE_*` | mostly `true` in example | ✅ | SCANNERS |
| `label_compat_mode` | `REPOSITORY_DETECTIVE_LABEL_COMPAT_MODE` | `new_only` | ✅ | NAMING |

Legacy `REPOSITORY_DETECTIVE_*` documented in [BRANDING_MIGRATION.md](BRANDING_MIGRATION.md).

**Config gaps:** Some advanced keys (runner HMAC, notification webhooks) only in `.env.example` — acceptable; listed in CONFIGURATION pointer.

---

## Part D — API routes

See [API_ROUTES.md](API_ROUTES.md) — created this audit.

---

## Part E — Gap list

### Blocks private beta

**None** — with documented caveats (API key, single tenant).

### Blocks public beta

- No self-service onboarding portal
- Operator docs still assume homelab
- No formal VPAT

### Blocks commercial

- Auth/RBAC not implemented
- License enforcement not implemented
- Custom report branding not implemented

### Blocks enterprise

- Tenant isolation
- SSO/OIDC
- Billing
- HA / Postgres
- GitHub/GitLab connected-repo parity at scale

### Nice-to-have

- `/health` latency ~4s
- Community intelligence feed
- checkov timeout hardening
- Edition license gates in code

---

## Part F — Tests (2026-06-05)

| Command | Result |
|---------|--------|
| `go test ./...` | **Pass** |
| `go vet ./...` | **Pass** |
| `staticcheck ./...` | **Pass** |
| `./scripts/docker-build-verify.sh` | **Pass** (after disk cleanup + preflight `e52507e`) |
| `./scripts/operator-smoke-test.sh` | **Pass** |
| `gosec` | Skipped — not installed |

---

## Part G — Private beta go/no-go

| Decision | Result |
|----------|--------|
| **Community private beta** | **GO** |
| **Public beta / SaaS** | **NO** |
| **Commercial sales (self-serve)** | **NO** — manual assessments OK |

**Statement:**

```text
Repository Detective Community private beta is ready.
Commercial/Enterprise roadmap is documented.
SaaS is not ready until Auth/RBAC and tenant isolation.
```

---

## Related docs

- [BETA_READINESS.md](BETA_READINESS.md)
- [TEST_MATRIX.md](TEST_MATRIX.md)
- [DOCS_AUDIT.md](DOCS_AUDIT.md)
- [RELEASE_NOTES_0.1.0_BETA.md](RELEASE_NOTES_0.1.0_BETA.md)
- [API_ROUTES.md](API_ROUTES.md)
