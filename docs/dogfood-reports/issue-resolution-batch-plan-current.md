# Issue resolution batch plan (current)

**Date:** 2026-06-05  
**Authority:** Repository Detective evidence closure — not manual Gitea closes  
**Method:** Cursor deterministic fixes → test → rescan → fingerprint absent → evidence close  
**No LLM bulk resolution**

---

## Inventory

| Source | Count | Notes |
|--------|------:|-------|
| Open Gitea-linked issues (`external_issues`) | ~290 | Fleet-wide in local DB |
| Open findings (SQLite) | 4108 | Includes graph noise |
| Triage CSV rows (`issue-closeout-triage.csv`) | 253 | 130 unique issue numbers |
| Operator-reported backlog target | ~209 | Use triage + live reconcile preview |

Re-export before each batch:

```bash
curl -s -H "X-Repository-Detective-API-Key: $KEY" \
  http://127.0.0.1:8081/api/v1/repos/1/reconcile-issues/preview | jq '.items | length'
```

---

## Grouping (fleet)

### By severity (open findings)

| Severity | Approx. count |
|----------|--------------:|
| low | 3215 |
| medium | 834 |
| high | 53 |
| critical | 6 |

### Top rules (open findings)

| Rule | Source | Count |
|------|--------|------:|
| GRAPH-ORPHAN-FILE | graph | 1380 |
| GRAPH-ORPHAN-FUNCTION | graph | 927 |
| QUAL-DEBUG | static | 495 |
| REL-INTERNAL-INFRA-REF | static | 223 |
| GRAPH-SUSPICIOUS-ISLAND | graph | 145 |
| SEC-HARDCODED-SECRET | static | 42 |

### Triage disposition (253 rows)

| Disposition | Count |
|-------------|------:|
| suppress_false_positive | 142 |
| already_fixed_verify | 44 |
| defer | 30 |
| needs_human_review | 20 |
| fix_now | 17 |

---

## Batch 0 — P0 security blockers (Repository Detective itself)

**Status:** Verified fixed/partial — see [current-security-blocker-verification.md](current-security-blocker-verification.md)

| Item | Action |
|------|--------|
| Webhook HMAC | No code change — verified |
| API auth | No code change — verified |
| Log redaction | Backlog: access-log sanitization |
| `rate_limit_per_minute` unused | Backlog: wire or document |

**commstech/Repository-Detective repo:** 0 critical open; 2 high static (SEC-CMD-EXEC, SEC-SQL-CONCAT) — likely rule-definition false positives in `analyzers/static.go`.

**Tests:** `go test ./handlers/... ./main_auth_test.go ./redact/...`  
**Risk:** Low  
**Closure:** Rescan after any fix; evidence only

---

## Batch 1 — Critical/high true positives (fleet)

**Scope:** 17 `fix_now` triage rows (11 Gitea issue numbers)

| Rule | Count | Repos | Proposed action |
|------|------:|-------|-----------------|
| SEC-HARDCODED-SECRET | 12 | House_Grocery_AI, netmon, ansible_playbooks, AMMBER | Fix in **each downstream repo** — not this product repo |
| TRIVY-MIS-DS002 | 4 | optouter, Repository-Detective | Dockerfile USER — **partially fixed** in Repository-Detective (`ab97c40`) |
| SEC-EVAL | 1 | eagle (PyTorch `model.eval()`) | Scanner FP fix in Repository-Detective — **done** (`model.eval()` skip) |

**Product repo only:**

- Rescan after deploy; evidence-close TRIVY-MIS-DS002 if absent
- Review SEC-CMD-EXEC / SEC-SQL-CONCAT in `analyzers/static.go` (self-scan FP)

**Tests:** `./scripts/docker-build-verify.sh`, `go test ./analyzers/...`  
**Risk:** Medium for fleet secret fixes  
**One repo per branch** — do not mix fleet fixes into product repo

---

## Batch 2 — Scanner/config/build blockers

| Signal | Action |
|--------|--------|
| `failed_scans_count: 13` | Replay failed scan IDs from dashboard |
| `scanner_failures_count: 11` | Fix parser/timeouts |
| `scanner_tools_missing_count: 12` | Document optional tools or disable flags |
| Hadolint DL3018 | Pin base image packages in Dockerfile |

**Tests:** `go test ./scanners/...`, operator smoke test  
**Risk:** Low–medium

---

## Batch 3 — Simple staticcheck/hadolint fixes

**Scope:** Low-risk lint in **commstech/Repository-Detective** only first

| Target | Action |
|--------|--------|
| staticcheck warnings | Fix when `staticcheck` available on Go 1.23 |
| Hadolint DL3018 | Pin versions |
| Ruff fleet rows | Defer to per-repo branches |

**Tests:** `go test ./...`, `staticcheck ./...`  
**Risk:** Low

---

## Batch 4 — Reliability findings

**Scope:** 58 `HEALTH-IGNORED-ERROR` in Repository-Detective repo + fleet reliability source

| Action | Where |
|--------|-------|
| Fix clear err handling | Per-file in Repository-Detective first |
| `already_fixed_verify` (44 triage) | Rescan + evidence close |

**Tests:** Package tests + reconciliation preview  
**Risk:** Low–medium

---

## Batch 5 — Docs/config/test gaps

| Item | Action |
|------|--------|
| README/logo/branding | This sprint |
| Go proxy supply-chain docs | This sprint |
| API auth route table | This sprint |
| TEST_MATRIX gaps | Incremental |

**Risk:** Low

---

## Batch 6 — Docs/config/test gaps (duplicate batch label merged into 5 above)

See Batch 5.

---

## Batch 7 — Suppress/defer known noise

**Do not suppress critical/high without human review.**

| Candidate | Rows | Action |
|-----------|-----:|--------|
| GRAPH-ORPHAN-* | 2300+ | Global/repo suppressions; report-only under `beta_standard` |
| QUAL-DEBUG in tests | ~495 fleet | Path-based suppress or defer |
| Triage `suppress_false_positive` | 142 | Apply calibration suppressions with evidence |
| Triage `defer` | 30 | Document in POLICY.md |

**Tests:** Rescan confirms fingerprint absent or suppressed with audit trail  
**Risk:** Low if evidence documented

---

## Evidence closure rules

```text
fix → go test ./... → operator smoke → full scan → fingerprint absent → verify-closure → close Gitea issue
```

- No manual Gitea close without rescan proof
- One batch per commit/PR
- Fleet repos: separate branch per repo
- AI advisory only after queue &lt; ~50 actionable items

---

## Execution order

```text
0. ✅ P0 verification doc
1. Repository-Detective Batch 1 product fixes (SEC-EVAL FP, Dockerfile USER) — shipped
2. Rescan Repository-Detective → evidence-close TRIVY/EVAL FPs
3. Batch 7 graph noise suppressions (SQL + calibration)
4. Batch 4 reliability in Repository-Detective
5. Batch 2 scanner failures
6. Fleet Batch 1 (one repo at a time)
7. AI 5-issue evaluation set (not before queue is small)
```

---

## Related

- [issue-resolution-sprint-plan.md](issue-resolution-sprint-plan.md) (original sprint plan)
- [current-security-blocker-verification.md](current-security-blocker-verification.md)
- [POLICY.md](../POLICY.md)
