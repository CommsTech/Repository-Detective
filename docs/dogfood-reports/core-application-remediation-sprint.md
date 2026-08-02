# Core application remediation sprint

**Date:** 2026-06-05  
**Scope:** Repository Detective product repo (`commstech/Repository-Detective`) only  
**Operating model:** Cursor improves RD core → RD fixes/validates connected repos → humans approve sensitive actions  
**Paused:** Manual fleet repo cleanup, RuView, Qdrant global enablement, new scanners, GitHub/GitLab connected support, Auth/RBAC Slice 2

---

## Goal

Improve Repository Detective so it can accurately resolve findings in **connected repos** through its own workflow:

```text
detect → plan → approved remediation PR → rescan → evidence closure
```

Fleet repos are **test targets**, not manual Cursor fix projects.

---

## Current product inventory (local DB, repo id=1)

| Metric | Count | Notes |
|--------|------:|-------|
| Open findings (all) | ~1642 | Most graph noise suppressed in scoring |
| Open unsuppressed (scored) | ~22 | Actionable product backlog |
| Critical/high open (scored) | 1 | SEC-SQL-CONCAT FP in store layer |
| Resolved verified | 21+ | TRIVY-MIS-DS002 on Dockerfile |
| Remediation plans (safe_for_auto_pr) | Low | Generate per finding from UI/API |

### Scored open findings by rule (product repo)

| Rule | Source | Count | Sprint class |
|------|--------|------:|--------------|
| HEALTH-IGNORED-ERROR | reliability | 13 | Product quality — defer auto-PR |
| DL3018 | hadolint | 2 | **RD self-remediation candidate** |
| OPT-HTTP-CLIENT-PER-CALL | static | 2 | Advisory / defer |
| HEALTH-DEPRECATED | tech_debt | 2 | Defer auto-PR |
| REL-INTERNAL-INFRA-REF | static | 1 | Docs/advisory |
| HEALTH-TECH-PHRASE | tech_debt | 1 | Defer |
| SEC-SQL-CONCAT | static | 1 | **FP — fixed in static.go** |

---

## What blocks safe repo remediation today

| Blocker | Type | Impact |
|---------|------|--------|
| `verify-closure` required prior closure evidence | **Product bug** | Direct fixes / branch rescans could not verify (`sql: no rows`) |
| Fingerprint set built from `last_seen_scan_id` only | **Product bug** | Incorrect verify-closure when instances exist but last_seen stale |
| Branch scans vs default-branch findings | **Design gap** | Auto-close on non-default ref would false-verify — must gate on default branch |
| `grype` / `golangci-lint` timeouts | Scanner ops | Score approximate; closure blocked when `require_scanner_success` and scanner missing |
| `staticcheck` parse_failed on Go 1.23 tree | Scanner ops | Go lint remediation plans blocked until parser fixed |
| No deterministic patcher for secrets/deps/auth | By design | Recipes correctly set `safe_for_auto_pr=false` |
| Access-log `?api_key=` leakage | P1 hardening | Docs only; middleware not implemented |
| Auth/RBAC Slice 2 not staged | Deferred | Slice 1 needs staging smoke before local auth default |

---

## Issue classification

### Product bugs (fix in core — Batch 1)

1. Evidence closure: `POST /api/v1/findings/:id/verify-closure` without prior evidence → **fixed** (`ensureClosureEvidenceRow` + `ListFingerprintsInScan`)
2. SEC-SQL-CONCAT false positive on store `fmt.Sprintf` IN-clause → **fixed**
3. Reconcile `already_fixed_verify` not wired to closure engine on default-branch scan → **Batch 2** (gate on default ref)

### Docs / noise (no code fix required)

- Graph orphan rules (suppressed in beta profile)
- QUAL-DEBUG volume in fleet (report-only)
- REL-INTERNAL-INFRA-REF in homelab docs (advisory)
- P0 security items verified in [current-security-blocker-verification.md](current-security-blocker-verification.md)

### Good RD self-remediation test candidates (after Batch 1 deploy)

| Finding | Rule | Why |
|---------|------|-----|
| Dockerfile pin | DL3018 | hadolint recipe, `safe_for_auto_pr=true`, small diff |
| staticcheck mechanical | SA/ST* | When staticcheck parser fixed |
| Config placeholder cleanup | static secret FP patterns | Only non-secret placeholder moves |

**Not eligible:** secrets, dependency upgrades, gosec/checkov critical, graph orphans, auth logic.

---

## Sprint priorities (ordered)

### 1. Remediation workflow hardening

- [x] Plans generated for valid findings (recipes in `remediation/recipes.go`)
- [x] `safe_for_auto_pr` / blocked reasons on finding detail UI
- [ ] Unsupported rule → explicit recipe `default` with human review
- [ ] Validation commands: scope to repo profile (Go/Python/docker)

### 2. Evidence closure reliability

- [x] Direct verify-closure without prior evidence row
- [x] Fingerprint lookup via `finding_instances` for scan
- [ ] Default-branch-only auto-verify on scan finish (Batch 2)
- [x] PR merge → pending_rescan → verify path (e2e covered)

### 3. Issue reconciliation

- [x] Preview/apply separation (`reconcile/engine.go`)
- [x] Never mass-close (`evidence_closure_close_issues: false`)
- [ ] Wire `already_fixed_verify` → closure on default-branch rescan

### 4. False-positive calibration

- [x] Suppressions + calibration tables
- [x] Do not auto-suppress security (human calibration only)
- [x] SEC-EVAL PyTorch skip, SEC-SQL-CONCAT store IN-clause skip

### 5. Safe remediation PR expansion

- [x] DL3018 patcher (Batch 2) — **E2E PASS 2026-06-06**
- [x] staticcheck S1039-style mechanical patcher (Batch 3) — **E2E PASS 2026-06-06**
- **Never:** secret rotation, dependency bumps, auth rewrites

#### DL3018 self-remediation E2E: PASS

Repository Detective created remediation PR → human merged → main rescanned → finding verified resolved → issue left open with `repository-detective/resolved-verified` label because `close_issues=false`.

| Artifact | Value |
|----------|-------|
| Finding | 9971 (`rd-2e9bfe809e79bcf0`, Dockerfile:100) |
| Plan | `rp-59815d80d8d32abb` |
| Patch attempt | `pa-6cbc72da69690560` |
| PR | [#274](https://git.commsnet.org/commstech/repository-detective/pulls/274) |
| Merge commit | `6f42552233ed15521085b51dca26fb82dfb86d6f` |
| Rescan | `09a44ba983243aab` |
| Report | [rd-self-remediation-dl3018-test.md](rd-self-remediation-dl3018-test.md) |

**First successful product-managed fix** — RD core created the plan, opened the PR, and verified closure after human merge; no manual fleet-repo patching by Cursor.

#### Staticcheck S1039 self-remediation E2E: PASS

Repository Detective created remediation PR → human merged → main rescanned → finding verified resolved. No linked Gitea issue (backfilled finding); `close_issues=false`.

| Artifact | Value |
|----------|-------|
| Finding | 11658 (`rd-c68376af29742113`, `internal/dogfood/staticcheck_e2e_marker.go:8`) |
| Plan | `rp-08270977049e02e8` |
| Patch attempt | `pa-12474c8d554fbbf5` |
| PR | [#288](https://git.commsnet.org/commstech/repository-detective/pulls/288) |
| Merge commit | `a0d32599ff21ab94bbbef905791ebf920d542d84` |
| Rescan | `6bdad6c92f1c8a0c` |
| Report | [rd-self-remediation-staticcheck-e2e-test.md](rd-self-remediation-staticcheck-e2e-test.md) |

**Second successful remediation class** — proves mechanical staticcheck fixes after DL3018. **Do not enable staticcheck auto-PR in beta by default** until Go-in-image, archive workspace, ingest, and package-scoped validation ship.

**Remediation expansion stopped** — two E2Es are enough proof for beta. Next: private beta ops (1 week), then Auth/RBAC Slice 2.

**DB access note:** avoid host-side SQLite against live `repository-detective.db` while RD is scanning/writing (`database is locked`). Prefer API, `docker exec`, backup copy, or read-only with `busy_timeout`.

### 6. Auth/RBAC stabilization

- [x] Slice 1 shipped (`auth_mode=api_key_only` default)
- [ ] Staging smoke: `auth_mode=local` + bootstrap + session CSRF
- [ ] Slice 2 blocked until Slice 1 stable

### 7. Operator UX

- [x] Dashboard remediation insight (`BuildRemediationInsight`)
- [x] Finding detail: eligibility, blocked reasons, validation commands, closure evidence
- [ ] Dashboard badge: “N findings auto-fixable now”

---

## Batch 1 — shipped in this sprint

| Change | File(s) | Effect |
|--------|---------|--------|
| Verify-closure without prior evidence | `closure/engine.go`, `main_closure.go` | API verify works for direct fixes |
| Accurate scan fingerprints for verify | `buildClosureScanContextFromStore`, `ListFingerprintsInScan` | No false still_present from stale last_seen |
| SEC-SQL-CONCAT store FP | `analyzers/static.go` | Removes spurious high on product repo |
| Tests | `closure/engine_direct_test.go`, `analyzers/static_test.go` | Regression coverage |

---

## Validation commands

```bash
go test ./...
go vet ./...
staticcheck ./...    # when available on Go 1.23 toolchain
./scripts/operator-smoke-test.sh
```

After deploy, test fleet workflow on **one** repo:

```bash
# 1. Generate + approve plan on low-risk finding (DL3018)
# 2. Create remediation PR via UI/API
# 3. Merge manually
# 4. Rescan default branch
curl -X POST http://localhost:8081/api/v1/analyze \
  -H "X-Repository-Detective-API-Key: $REPOSITORY_DETECTIVE_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"owner":"commstech","repository":"Repository-Detective","ref":"main"}'
# 5. POST verify-closure OR wait for pending_rescan auto-verify
# 6. Confirm resolved_verified label; issue stays open (close_issues=false)
```

---

## Explicit non-goals

- Manual fleet repo fixes (House_Grocery_AI branch remains for RD workflow test only)
- Broad AI issue resolution
- `evidence_closure_close_issues: true` in production (staging first)
- Auth/RBAC Slice 2, billing, tenant isolation

---

## Next batches

| Batch | Focus |
|-------|--------|
| **2** | ~~Default-branch absent-fingerprint auto-verify; DL3018 patcher; reconcile→closure bridge~~ DL3018 E2E **done**; remaining: default-branch auto-verify, reconcile→closure bridge |
| **3** | staticcheck S1039 second E2E remediation test; parser + mechanical patcher; access-log API key redaction middleware |
| **4** | Auth Slice 1 staging validation; Auth/RBAC Slice 2 or private beta ops (after two passing E2E remediation tests) |
