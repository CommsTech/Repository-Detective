# Prime-time readiness evaluation — 2026-08-02

**Product:** Repository Detective  
**Gitea:** https://git.commsnet.org/commstech/Repository-Detective.git  
**Live version:** `rc-rd-brand-purge`  
**Evaluator:** automated operator smoke + feature matrix + core Go tests + fleet posture

## Verdict

**Conditional GO for private beta / operator dogfood.**  
**Not yet GO for broad public prime-time** until critical/high backlog and scanner parse-failure noise are brought under control.

## Gate results

| Gate | Result | Evidence |
|------|--------|----------|
| Git sync to `Repository-Detective` | **PASS** | `main` @ `15ab07d` (+ follow-up commits below) |
| Live health / ready | **PASS** | healthy, ready=true, scanners 10/10 |
| Operator smoke | **PASS** | health, about, status, dashboard; legacy `X-Bugbot-API-Key` → 401 |
| Feature matrix | **PASS** | 24/24 — [feature-matrix-prime-time-20260802T163910Z.md](feature-matrix-prime-time-20260802T163910Z.md) |
| Manual report-only self-scan | **PASS** | scan `bce46b3e42cf4143` completed |
| Core Go tests | **PASS** | store, ui, api, issues, gitea, envcompat, security |
| Branding (UI/API) | **PASS** | product_name Repository Detective; no Bugbot in `/api/v1/about` |
| Fleet scan health | **PASS** | unhealthy repos = 0; actionable failed = 0 |
| Findings backlog | **BLOCKER for public** | 11,233 open; **244 critical+high** |
| Scanner parse failures (14d) | **WATCH** | 251 parse_failed events |
| Historical DB error text | **INFO** | Health page still shows old `commstech/Bugbot` API errors from prior scans |

## Fleet snapshot

| Metric | Value |
|--------|------:|
| Repositories | 40 |
| Open findings | 11,233 |
| Critical | 10 |
| High | 234 |
| Medium | 3,525 |
| Low | 2,763 |
| Info | 4,701 |
| Suppressed / FP | 4,539 |
| Unhealthy repos (latest failed, non-noise) | 0 |
| Actionable failed scans | 0 |
| Completed scans (24h) | 95 |
| Failed scans (24h) | 9 (restart/noise class historically) |
| Parse failed (14d) | 251 |

## What works well

- Deterministic scanner runtime present (10/10 configured available)
- Dashboard/API/UI routes responsive after DB checkpoint (~1.5–2.5s dashboard)
- Report-only analyze path works end-to-end
- Branding cutover is live (env + API header + about + UI)
- Scan reliability posture is clean (no currently unhealthy repos)

## Must fix before public prime-time

1. **Triage critical/high open findings (244)** — either remediate, suppress with rationale, or scope beta to report-only + backlog-control so operators are not drowned.
2. **Investigate parse_failed volume (251 / 14d)** — especially Trivy/Gitleaks/Hadolint classes; surface root causes and reduce silent coverage gaps.
3. **Rebuild all-in-one image** from current `main` (live is hotpatched `rc-rd-brand-purge` on older image base) so deploys match git without docker cp.
4. **Optional:** re-run product self-scan after rename so health drill-down no longer quotes historical `commstech/Bugbot` forge URLs in stored scanner errors.

## Recommended beta operating mode (until backlog shrinks)

- Keep **report-only / dry-run** for non-product repos
- Keep **issue filing off or tightly gated**
- Prefer **Standard** profile (deterministic); enable **Deep**/LLM only for named dogfood repos
- Treat dashboard critical+high queue as the daily operator KPI

## Commands re-run

```bash
./scripts/operator-smoke-test.sh
./scripts/feature-matrix-smoke.sh
go test -mod=vendor ./store/ ./ui/ ./api/ ./issues/ ./gitea/ ./internal/config/envcompat/ ./internal/security/ -count=1
```
