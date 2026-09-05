# Issue resolution sprint plan

**Product:** Repository Detective — Inspect. Analyze. Improve.  
**Date:** 2026-06-05  
**Status:** Planning complete — **Batch 1 approved** for implementation  
**Method:** Cursor for deterministic fixes; Repository Detective for verify/close with evidence. **No LLM bulk resolution.**

---

## 1. Source data

| Source | Count | Notes |
|--------|------:|-------|
| Operator-reported open Gitea issues | ~209 | Fleet backlog target |
| `issue-closeout-triage.csv` fingerprint rows | 253 | Exported 2026-06 dogfood closeout |
| Unique Gitea issue numbers in CSV | 130 | Multiple fingerprints per issue |
| `still_in_latest_scan=True` | 157 | Still reproducing |
| `fix_now` disposition | 17 | Critical/high security true positives |

Re-export before each batch:

```bash
# Live preview (repo 1 example)
curl -s -H "X-Repository-Detective-API-Key: $KEY" \
  http://127.0.0.1:8081/api/v1/repos/1/reconcile-issues/preview | jq '.items | length'
```

---

## 2. Grouping summary

### By severity (CSV rows)

| Severity | Count |
|----------|------:|
| medium | 182 |
| high | 52 |
| low | 13 |
| critical | 6 |

### By disposition

| Disposition | Count | Action |
|-------------|------:|--------|
| suppress_false_positive | 142 | Batch 6 — calibration suppressions |
| already_fixed_verify | 44 | Batch 4 — rescan + evidence close |
| defer | 30 | Batch 6 — document noise |
| needs_human_review | 20 | Hold for human triage |
| fix_now | 17 | **Batch 1** |

### By source (scanner)

| Source | Count |
|--------|------:|
| graph | 82 |
| static | 63 |
| ruff | 60 |
| reliability | 14 |
| checkov | 10 |
| maintainability | 8 |
| tech_debt | 8 |
| trivy | 4 |
| hadolint | 3 |
| semgrep | 1 |

### Top noisy rules (all rows)

Graph architecture rules dominate volume (82 rows). Security `fix_now` concentrates on `SEC-HARDCODED-SECRET` (12) and `TRIVY-MIS-DS002` (4).

---

## 3. Batches

### Batch 1 — Critical/high security true positives — **APPROVED**

**Scope:** 17 `fix_now` rows (16 high + 1 critical), 11 unique Gitea issue numbers.

| Issue # | Rule ID | Severity | Count | Root cause (summary) |
|--------:|---------|----------|------:|----------------------|
| 1 | SEC-EVAL | critical | 1 | Static analyzer flags dynamic code execution |
| 1,2,3,4,5,7,205 | SEC-HARDCODED-SECRET | high | 12 | Test fixtures / config examples matching secret heuristics |
| 7,9,207,208 | TRIVY-MIS-DS002 | high | 4 | Dockerfile `USER root` or missing non-root user |

**Proposed fixes:**

| Rule | Fix approach | Files likely changed |
|------|--------------|----------------------|
| SEC-EVAL | Confirm call site; replace `eval`/exec with safe alternative or narrow suppression with evidence if test-only | `main.go`, handlers, or test helpers |
| SEC-HARDCODED-SECRET | Move literals to env/test builders; redact examples; split real secrets vs test tokens | `*_test.go`, `config/*.example`, docs with curl samples |
| TRIVY-MIS-DS002 | Add non-root `USER` in Dockerfiles after install steps | `Dockerfile`, `Dockerfile.offline`, runner images |

**Tests:** `go test ./...`, `./scripts/operator-smoke-test.sh`, targeted rescan of `commstech/Repository-Detective`.

**Risk:** Medium — secret/rule changes need careful review to avoid breaking tests or docs.

**Remediation PR:** One commit/PR for Batch 1 only.

---

### Batch 2 — Build/test failures or scanner failures

**Scope:** Failed scans (13 in dashboard), scanner failures (11), missing tools (12).

| Signal | Action |
|--------|--------|
| `scanner_failures_count` | Fix parser/timeouts; document bypass for optional tools |
| `failed_scans_count` | Replay failed scan IDs; fix OOM/timeout |
| Missing binary in health | Install path or `enable_*: false` + docs |

**Tests:** `go test ./scanners/...`, `docker-build-verify.sh`.

**Risk:** Low–medium.

---

### Batch 3 — Low-risk staticcheck / hadolint / simple code quality

**Scope:** ~60 ruff rows + staticcheck/hadolint findings not security-critical.

| Examples | Fix |
|----------|-----|
| Unused imports | Remove |
| Hadolint DL3018 pin warnings | Pin versions in Dockerfiles |
| Simple lint autofix | `ruff check --fix` where safe |

**Tests:** `go test ./...`, `staticcheck ./...`.

**Risk:** Low.

---

### Batch 4 — Reliability findings (deterministic)

**Scope:** 14 `reliability` source rows + `already_fixed_verify` (44) for evidence closure.

| Action | Tool |
|--------|------|
| Fix clear nil/err handling | Code edit |
| Close fixed issues | Rescan + `verify-closure` API |

**Tests:** Package tests + reconciliation preview.

**Risk:** Low–medium.

---

### Batch 5 — Docs / config / test gaps

**Scope:** Stale docs, missing env examples, test matrix gaps.

**Tests:** `scripts/release-test.sh`, docs link check.

**Risk:** Low.

---

### Batch 6 — Suppress / defer known noise

**Scope:** 142 `suppress_false_positive` + 30 `defer` + 82 graph rows (report-only under `beta_standard`).

| Candidate | Rationale |
|-----------|-----------|
| Graph architecture rules | Already report-only in `beta_standard`; suppress legacy Gitea issues |
| Pre-calibration duplicates | `already_fixed_verify` without rescan |
| Test fixture secrets | Mark false positive after Batch 1 confirms |

**Action:** SQL suppressions + calibration recommendations; **no code fix** where policy says report-only.

**Risk:** Low if evidence documented.

---

## 4. Estimated quick wins

| Batch | Est. issues closable | Effort |
|-------|---------------------:|--------|
| 6 (suppress graph noise) | 80+ | 1–2 hours SQL + rescan |
| 4 (evidence close verified) | 44 | Rescan + closure API |
| 3 (lint autofix) | 30–50 | Half day |
| 1 (security fix_now) | 17 | 1 day review |
| 2 (scanner health) | 10–20 | Variable |

**Do not run AI bulk resolution** until Batches 1, 3, and 6 reduce open queue below ~50 actionable items.

---

## 5. High-risk items (human review required)

| Item | Why |
|------|-----|
| SEC-EVAL critical | Could be real code execution path — verify before suppress |
| Live credentials in repo | Rotate if true positive |
| TRIVY-MIS-DS002 | Breaking change if app expects root in container |
| `needs_human_review` (20 rows) | Ambiguous disposition in CSV |
| RuView / disclosure drafts | Never auto-close public security issues |

---

## 6. Suppress / defer candidates

| Rule family | Rows | Recommendation |
|-------------|-----:|----------------|
| Graph `ARCH-*` / `GRAPH-*` | 82 | Suppress at repo or global level; keep dashboard visibility |
| Ruff style in vendor/tests | ~40 | Defer or suppress per path |
| Medium severity maintainability | 8 | Defer post-beta |

Use `docs/dogfood-reports/closeout-suppressions.sql` as template; record in `finding_suppressions` table.

---

## 7. Test plan (per batch)

```bash
go test ./...
go vet ./...
staticcheck ./...          # when available
./scripts/operator-smoke-test.sh
./scripts/docker-build-verify.sh   # after Dockerfile changes
```

After each batch:

1. Run full scan on `commstech/Repository-Detective` (manual or scheduled).
2. `GET /api/v1/repos/1/reconcile-issues/preview` — confirm fingerprints gone.
3. Evidence-close only (`evidence_closure_close_issues: false` default — use verify + manual Gitea close or enable close when ready).

---

## 8. Closure rules

| Rule | Requirement |
|------|-------------|
| Evidence only | Fingerprint absent in latest completed scan **or** documented suppression with reason |
| No AI closure | LLM may advise; operator or deterministic verify closes |
| One batch per commit | Easier revert and audit |
| Fingerprint match | Close Gitea issue only when reconciliation links finding |
| Security | Never commit real secrets; rotate if exposed |

---

## 9. AI evaluation set (after queue &lt; 50)

When Batches 1, 3, and 6 are done, test **advisory AI only** on 5 issues:

| # | Type |
|---|------|
| 2 | Simple true positives |
| 1 | False positive |
| 1 | Remediation planning |
| 1 | Ambiguous human-review |

Measure: classification accuracy, no overclaiming, no unsafe patches, token usage, schema validity, human-review escalation.

---

## 10. Execution order

```text
1. ✅ Auth/RBAC Slice 1 (parallel track)
2. ✅ This sprint plan
3. → Batch 1 security fix_now (APPROVED — implement next)
4. Rescan + evidence-close Batch 1
5. Batch 6 suppress graph noise
6. Batch 3 lint quick wins
7. AI 5-issue evaluation set
```

---

## Related

- [issue-closeout-triage.csv](issue-closeout-triage.csv)
- [repo-issue-closeout-report.md](repo-issue-closeout-report.md)
- [POLICY.md](../POLICY.md)
- [AUTH_LOCAL.md](../AUTH_LOCAL.md)
