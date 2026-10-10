# Product repo issue burndown plan

**Date:** 2026-06-06  
**Repository:** commstech/Repository-Detective (Gitea)  
**Open issues exported:** **241** (paginated API)

## Summary

| Priority | Count (est.) | Focus |
|----------|--------------|-------|
| P0 | ~15 | CI/release blockers, credential exposure, auth/log redaction |
| P1 | ~120 | Scanner correctness, graph/wiki, reliability noise triage |
| P2 | ~40 | Remediation/evidence closure, reconciliation |
| P3 | ~30 | UI/docs/operator friction |
| P4 | ~36 | Known noise / suppress / defer |

## Severity breakdown (from titles)

| Severity | Open |
|----------|------|
| MEDIUM | 203 |
| LOW | 11 |
| HIGH | 3 |
| untagged title | 24 |

## Category breakdown (from labels)

| Category | Open |
|----------|------|
| reliability | 76 |
| other/unlabeled category | 126 |
| code-quality | 27 |
| security | 9 |
| tech-debt | 3 |

---

## P0 — security exposure / CI blockers / release blockers

**Owner:** Cursor (product repo)

| Theme | Examples | Fix | Tests | Closure path |
|-------|----------|-----|-------|--------------|
| CI workflow red | Gitea Actions run #1835 format failure | Scoped gofmt + Go 1.23 CI (this sprint) | CI green on push | Close after workflow pass |
| Release workflow drift | Go 1.21 vs 1.23 | Release workflow updated | Tag dry-run | Close after tag CI pass |
| API key in URL/logs | `?api_key=` in UI links | Rotated key; HttpOnly cookie redirect; access log redaction | middleware test | Document rotation; no auto-close |
| Access log secret leakage | Gin default logger | `RedactingAccessLogger` | redact test | Verify in deploy |

**Rules:** Do not auto-close P0 until evidence exists (CI green or security verification note).

---

## P1 — scanner correctness / graph / wiki / auth safety

**Owner:** Cursor

| Theme | Count est. | Fix | Tests | Closure |
|-------|------------|-----|-------|---------|
| Graph UI false truncation | 1 scan (`f9ef961c6f9a71c8`) | Missing graph → no truncation banner; settings link when truncated | UI graph missing test | Rescan + manual verify |
| Wiki not on Gitea wiki | docs only | `scripts/publish-gitea-wiki.sh` + `docs/WIKI_PUBLISHING.md` | manual push | Close after wiki push proof |
| staticcheck S1039 durable fixes | 1 E2E class | Patcher + package validation committed | patcher tests | E2E already PASS |
| Reliability: ignored error return | ~76 | Batch fix `_ = err` / handle errors in touched packages | `go test` package scope | Rescan + `resolved-verified` label; **no auto-close** |
| Scanner parse failures (gosec/grype) | recurring in scans | Scanner stderr/timeout fixes (separate batches) | scanner tests | Rescan fingerprint absent |

**Repository Detective:** max **1–2** low-risk mechanical issues per batch (hadolint/staticcheck only).

---

## P2 — remediation / evidence closure / reconciliation

**Owner:** Cursor + RD (limited)

| Theme | Fix | Notes |
|-------|-----|-------|
| Evidence closure UX | Already verified DL3018 + S1039 E2E | Keep `close_issues=false` |
| Issue reconciliation | Wire `already_fixed_verify` → closure on default-branch rescan | Sprint backlog |
| Remediation expansion | **STOPPED** | Two E2Es sufficient for beta |

---

## P3 — UI / docs / operator friction

**Owner:** Cursor

| Theme | Fix |
|-------|-----|
| `?api_key=` in templates | Migrate forms to cookie/session; deprecate query suffix |
| Dashboard/chart polish | After P0/P1 |
| Docs drift | Sync `docs/wiki` → Gitea wiki via publish script |

---

## P4 — known noise / suppress / defer

**Owner:** Operator calibration

| Theme | Action |
|-------|--------|
| Duplicate fingerprints | Suppress or reconcile |
| Test fixture findings | Exclude paths or suppress rules |
| Graph orphan advisory | Manual review only — **no RD auto-delete** |

---

## Batch execution order

```text
Batch 1 (this sprint): P0 CI/release/graph/wiki/security redaction  ← IN PROGRESS
Batch 2: P1 reliability ignored-error returns (packages: handlers, store, scanners) — 20 issues max
Batch 3: P1 code-quality staticcheck/native findings — verify + label
Batch 4: P2 reconciliation wiring
Batch 5: P3 UI cookie auth completion (remove ?api_key= from templates)
Batch 6: P4 suppress/defer after operator review
```

## Per-batch verification

```bash
go test ./...
go vet ./...
staticcheck ./...   # or CI container
./scripts/operator-smoke-test.sh
./scripts/docker-build-verify.sh   # when Docker changes land
```

Then: rescan `commstech/Repository-Detective`, use `POST /api/v1/findings/{id}/verify-closure`, apply `repository-detective/resolved-verified`, keep `evidence_closure_close_issues=false`.

## Issues fixed this sprint (product code, not Gitea close)

| Area | Status |
|------|--------|
| S1039 remediation durable fixes | committed |
| CI/release workflows | updated |
| Graph missing vs truncated UI | fixed |
| Wiki publish script | added |
| API key rotation + log redaction | done locally (.env) |
| Operator flaky tests | fixed |

## Gitea issues closed this sprint

**0** — intentional. Close only after rescan evidence per policy.

## Next recommended batch

**Batch 2:** P1 reliability — ignored error returns in `handlers/` and `store/` (max 20 issues), then rescan + verify.
