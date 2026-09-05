# External review reconciliation — 2026-08-02

**External review (truncated paste):** architecture / code quality assessment against live `rc-ui-flow-eval`  
**Our concurrent audit:** accuracy / reliability / UX / docs → live `rc-full-audit7`  
**This doc:** merge both into one prioritized truth set

---

## Snapshot correction

| Claim (external) | Current reality |
|------------------|-----------------|
| Live version `rc-ui-flow-eval` | **`rc-full-audit7`** (healthy, 12/12 tools) |
| Dashboard tools missing implied healthy | Was **misleading historical 10**; now **0** via live probe overlay |
| Open findings ~11,233 (10 crit / 234 high) | Still ~**11.7k** open (10 crit / 238 high) — backlog remains |
| LLM auditors off by default | **Operator `config.yaml`:** `enable_llm_auditors: false`. **Code default:** `viper.SetDefault(..., true)` — mismatch; YAML wins when present, bare installs could enable LLM |
| Mega `main.go` | Still ~**2594** lines — valid critical maintainability finding |
| Docs extensive / wiki | Wiki now **24 published pages** at https://git.commsnet.org/commstech/repository-detective/wiki |

---

## Where the external review is right

Agree and keep as backlog:

1. **Mega `main.go` + flat Config (~180 fields)** — highest maintainability risk; safe decomposition is needed before multi-contributor velocity.
2. **Deterministic-first / concurrent scanners / subprocess isolation / fingerprints / SQLite EffectiveSettings** — correctly praised; preserve these invariants during any refactor.
3. **Security fundamentals** (HMAC compare, API key, rate limit, body limit, redaction) — accurate directionally.
4. **UI theme / charts / a11y / perf indexes** — matches our earlier UI eval work.
5. **Finding backlog volume** — still the main *operator trust* problem for “security product people will get behind,” even when scanners are accurate.

---

## Where our audit already fixed what the external review could not see

| Topic | Status after our audit |
|-------|------------------------|
| ShellCheck 0.10 flat JSON `parse_failed` | **Fixed** → `found` |
| Trivy parse / empty output | **Fixed** (`--output` + cache-dir) → `found` |
| SBOM `sbom_tool_missing` / cyclonedx fail | **Fixed** Syft fallback → `sbom_generated` (~58 pkgs) |
| Grype “malformed DB” during scans | **Fixed** — corrupt DB was under `XDG_CACHE_HOME=/app/data/cache`, not `$HOME/.cache` |
| Dashboard tools-missing vs System Health | **Fixed** live overlay on UI + API |
| Wiki stubs / unpublished | **Fixed** — API publish, full pages |
| OpenAPI `repository_id` / missing repo scan list | **Fixed** in OpenAPI + agent docs |

Evidence: `docs/dogfood-reports/full-application-audit-2026-08-02.md`

---

## Combined priority stack (what to do next)

### P0 — Trust / correctness (product credibility)

1. **Rebuild all-in-one image with Go 1.25** (match `go.mod`) so cyclonedx-gomod / staticcheck / golangci stop failing on toolchain skew.  
2. **Align `enable_llm_auditors` viper default to `false`** (match beta policy + `config.yaml`).  
3. **Continue finding backlog burn-down** (calibration, suppressions, focus list) — 11k opens will sink adoption even if scanners are correct.

### P1 — Maintainability (external review’s #1)

4. **Decompose `main.go` in vertical slices** (no behavior change):
   - `cmd/repository-detective` or keep main thin + `internal/app` bootstrap  
   - `internal/config` (Config struct + viper defaults)  
   - `internal/httpserver` (router / middleware)  
   - `internal/scanorchestrator` (webhook + analyze handlers currently in main)  
   - leave `api/` + `ui/` handlers as-is; only move wiring  
5. Split Config into nested structs (Forge / Scanners / AI / Runner / Preinstall / UI) with mapstructure nesting.

### P2 — Docs / agent surface

6. Expand OpenAPI toward `docs/API_ROUTES.md` (still partial by design).  
7. Keep wiki publisher as API-first (`scripts/publish-gitea-wiki-api.py`).

### P3 — Already deferred / OK for beta

- GitHub forge parity  
- Full LLM prove stage  
- Multi-tenant RBAC  
- HA DB

---

## Note on truncated paste

The external review cut off at:

> **Recommended fix:** Decompose into focused packages:

If the rest of that review (items 2–N under “Needs Improvement”) is pasted, fold them into this file the same way: agree / already fixed / defer.

---

## Recommendation

Do **not** start a big-bang `main.go` rewrite before the **Go 1.25 image rebuild** and the **LLM default fix** — those change operator-visible accuracy. Then decompose `main.go` in small mergeable PRs that keep tests green and the live hotpatch path intact.
