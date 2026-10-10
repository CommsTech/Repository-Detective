# Repository Detective — private beta readiness

**Product:** Repository Detective — Inspect. Analyze. Improve.  
**Checkpoint date:** 2026-06-04 (UTC)  
**Branch:** `main` @ `af89214` (synced with `origin/main`)

Private beta **freeze week** is active: **no new features** — only testing, documentation, bug fixes, and release hardening.

**Test/doc hardening phase:** **complete** (2026-06-05)  
**Feature completeness audit:** **complete** — [FEATURE_COMPLETENESS_AUDIT.md](FEATURE_COMPLETENESS_AUDIT.md) · [API_ROUTES.md](API_ROUTES.md)

**Test & docs:** [TEST_MATRIX.md](TEST_MATRIX.md) · [BETA_SMOKE_TEST.md](BETA_SMOKE_TEST.md) · [QUICKSTART.md](QUICKSTART.md) · [RELEASE_NOTES_0.1.0_BETA.md](RELEASE_NOTES_0.1.0_BETA.md)

Run before handoff:

```bash
./scripts/release-test.sh
./scripts/operator-smoke-test.sh
```

---

## Beta operating mode (recommended)

Use `config/config.yaml.example` as the template. Secrets only in `.env`.

```yaml
scan_profile: beta_standard
ai_startup_test_enabled: false
enable_llm_auditors: false
enable_ai_risk_checks: false
remediation_pr_enabled: false
evidence_closure_enabled: true
evidence_closure_close_issues: false
preinstall_audit_enabled: false   # enable on-demand for third-party audits only
notifications_enabled: false      # optional when configured
runner_delegation_enabled: false  # optional when runners configured
```

Per-repo AI policy: use `ai_policy: disabled` unless explicitly testing LLM features.

---

## Readiness checklist

| Item | Status | Notes |
|------|--------|-------|
| Backup/restore drill passed | **Pass** | See `docs/BACKUP_RESTORE.md`; operator drills under `restore-drill-test/` (local) |
| Migration 16 applied | **Pass** | `idx_external_issues_finding_id`; verified on production DB |
| All-in-one image builds | **Pass** | `repository-detective:all-in-one` @ `0b5005a2a2b3` |
| Scanner binaries available | **Pass** | 10/10 on `/health` (git, trivy, grype, gitleaks, semgrep, govulncheck, gosec, staticcheck, hadolint, checkov) |
| Issue reconciliation fast | **Pass** | Beta blocker fixes in `79cae24` |
| Calibration recompute fast | **Pass** | Same |
| Suppression / calibration UI+API | **Pass** | `api/suppressions_handler.go`, dashboard calibration block |
| Scoring fixed (non-zero when findings exist) | **Pass** | `analyzers/scoring.go` |
| Gitleaks parser fixed (8.x report file) | **Pass** | `957421f` |
| Theme persistence fixed | **Pass** | `ui/static/theme.js`, bootstrap in `layout.html`, tests |
| Qdrant | **Removed** | Fingerprint + SQLite forge mappings only |
| AI startup test disabled by default | **Pass** | `ai_startup_test_enabled: false` |
| Pre-install audit works | **Pass** | On-demand API; RuView dogfood complete |
| Remediation PRs disabled by default | **Pass** | `remediation_pr_enabled: false` |
| Evidence closure close_issues disabled | **Pass** | Comments/verify only |
| Private beta limitations documented | **Pass** | This file + `docs/PRIVACY.md`, `docs/POLICY.md` |
| `go test ./...` | **Pass** | 2026-06-04 |
| `go vet ./...` | **Pass** | 2026-06-04 |
| `staticcheck ./...` | **Pass** | 2026-06-04 |
| Docker build verify script | **Pass** | `./scripts/release-test.sh` bundles verify + go tests |
| Operator smoke script | **Pass** | `./scripts/operator-smoke-test.sh` (2026-06-05) |
| `docker-build-verify.sh` | **Pass** | After host disk cleanup; see release-test notes below |
| Disk preflight in verify script | **Pass** | 10 GB minimum; `VERIFY_MIN_DISK_GB` override |

### Release-test infrastructure (2026-06-05)

| Item | Detail |
|------|--------|
| Initial failure cause | Host root filesystem 100% full |
| Production impact | None — production on 8081 unaffected |
| Fixes | Stale `rd-verify-*` cleanup (`9f958fe`); disk preflight in verify script |
| Cleanup | `docker builder prune`; `docker image prune -af` freed ~35 GB |
| Result | `docker-build-verify.sh` passed after cleanup |
| Test matrix documented | **Pass** | [TEST_MATRIX.md](TEST_MATRIX.md) |
| Beta E2E smoke doc | **Pass** | [BETA_SMOKE_TEST.md](BETA_SMOKE_TEST.md) |
| Operator quickstart | **Pass** | [QUICKSTART.md](QUICKSTART.md) |

---

## Known gaps (not beta blockers)

| Gap | Classification |
|-----|----------------|
| `/health` ~4s latency (scanner probes) | nice-to-have |
| checkov/grype intermittent timeouts | nice-to-have |
| Full multi-user RBAC | blocks SaaS; slice 1 local login optional (`auth_mode=local`); default `api_key_only` |
| No tenant isolation | blocks SaaS |
| No billing | blocks SaaS / paid automation |
| GitHub/GitLab connected-repo parity | blocks paid manual audits at scale |
| Broad auto-fixing | intentionally out of scope |
| External pre-install sharing | manual operator workflow |

---

## Remaining blockers by phase

### Blocks private beta

- **None critical** if operator accepts API-key auth (default) or optional local UI login and single-tenant SQLite.

### Blocks paid manual audits

- Manual report workflow only (no customer portal).
- GitHub/GitLab integration incomplete for connected repos at scale.
- No RBAC for multi-analyst teams.

### Blocks SaaS

- Auth/RBAC, tenant isolation, billing, health latency SLA.

### Nice-to-have

- Health endpoint caching.
- checkov parser hardening.
- Full Docker image rebuild on every release via CI.

---

## Private beta go/no-go

| Decision | Result |
|----------|--------|
| **Go** for limited private beta (single operator, own Gitea, API key) | **YES** |
| **Go** for unsupervised multi-tenant SaaS | **NO** |
| **Go** for automated upstream disclosure | **NO** (manual only) |

---

## Next engineering phase (recommended order)

1. **Beta freeze week** — track issues/FPs/scanner failures only (no feature work)  
2. **Auth/RBAC implementation** — [AUTH_RBAC_PLAN.md](AUTH_RBAC_PLAN.md) in small slices (Commercial unlock)  
3. **Edition / license gates** — [EDITIONS.md](EDITIONS.md) (after beta feedback; not enforced yet)  
4. **Operator docs polish** — onboard another homelab without babysitting  
5. **Paid manual assessment workflow** — pre-install package + human review SOP  

Do **not** expand scanners or RuView dogfood until another operator completes a successful beta week.

---

## Beta week tracking (operator checklist)

During the freeze week, log only:

- New issues created  
- False positives  
- Scanner failures  
- Reconciliation results  
- Score accuracy  
- Operator friction  
- UI bugs  
- Scan duration  

No feature work unless a **blocks-private-beta** defect appears.

---

## Related docs

- [OPERATOR_READINESS.md](OPERATOR_READINESS.md)
- [BACKUP_RESTORE.md](BACKUP_RESTORE.md)
- [POLICY.md](POLICY.md)
- [PRIVACY.md](PRIVACY.md)
- [QDRANT.md](QDRANT.md)
- [AUTH_RBAC_PLAN.md](AUTH_RBAC_PLAN.md)
- [MONETIZATION_READINESS.md](MONETIZATION_READINESS.md)
- [EDITIONS.md](EDITIONS.md)
- [LICENSING_STRATEGY.md](LICENSING_STRATEGY.md)
- [BRANDING_COMPATIBILITY_AUDIT.md](BRANDING_COMPATIBILITY_AUDIT.md)
- [TEST_MATRIX.md](TEST_MATRIX.md)
- [BETA_SMOKE_TEST.md](BETA_SMOKE_TEST.md)
- [QUICKSTART.md](QUICKSTART.md)
- [CONFIGURATION.md](CONFIGURATION.md)
- [DOCS_AUDIT.md](DOCS_AUDIT.md)
- [RELEASE_NOTES_0.1.0_BETA.md](RELEASE_NOTES_0.1.0_BETA.md)
