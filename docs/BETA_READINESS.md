# Repository Detective — private beta readiness

**Product:** Repository Detective — Inspect. Analyze. Improve.  
**Checkpoint date:** 2026-06-04 (UTC)  
**Branch:** `main` @ `af89214` (synced with `origin/main`)

Private beta **freeze week** is active: no new scanners, features, or Qdrant enablement. Track issues, false positives, scanner failures, reconciliation, scores, UI friction, and scan duration only.

---

## Beta operating mode (recommended)

Use `config/config.yaml.example` as the template. Secrets only in `.env`.

```yaml
scan_profile: beta_standard
ai_startup_test_enabled: false
enable_llm_auditors: false
enable_ai_risk_checks: false
qdrant_enabled: false
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
| Qdrant disabled by default | **Pass** | Redacted/local-only code path; not production-enabled |
| AI startup test disabled by default | **Pass** | `ai_startup_test_enabled: false` |
| Pre-install audit works | **Pass** | On-demand API; RuView dogfood complete |
| Remediation PRs disabled by default | **Pass** | `remediation_pr_enabled: false` |
| Evidence closure close_issues disabled | **Pass** | Comments/verify only |
| Private beta limitations documented | **Pass** | This file + `docs/PRIVACY.md`, `docs/POLICY.md` |
| `go test ./...` | **Pass** | 2026-06-04 |
| `go vet ./...` | **Pass** | 2026-06-04 |
| `staticcheck ./...` | **Pass** | 2026-06-04 |
| Docker build verify script | **Pass** | All targets build; smoke `/health` OK (2026-06-04) |

---

## Known gaps (not beta blockers)

| Gap | Classification |
|-----|----------------|
| `/health` ~4s latency (scanner probes) | nice-to-have |
| Qdrant embedding 1024 + UUID point-id | blocks Qdrant enablement only |
| checkov/grype intermittent timeouts | nice-to-have |
| No multi-user auth/RBAC | blocks SaaS; **not** private beta (API key OK) |
| No tenant isolation | blocks SaaS |
| No billing | blocks SaaS / paid automation |
| GitHub/GitLab connected-repo parity | blocks paid manual audits at scale |
| Broad auto-fixing | intentionally out of scope |
| External pre-install sharing | manual operator workflow |

---

## Remaining blockers by phase

### Blocks private beta

- **None critical** if operator accepts API-key auth and single-tenant SQLite.

### Blocks paid manual audits

- Manual report workflow only (no customer portal).
- GitHub/GitLab integration incomplete for connected repos at scale.
- No RBAC for multi-analyst teams.

### Blocks SaaS

- Auth/RBAC, tenant isolation, billing, Qdrant production path, health latency SLA.

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
