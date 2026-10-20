# WebUI full flow evaluation — 2026-08-02

**Target:** `http://127.0.0.1:8081/ui` (live `repository-detective`, `rc-ui-eval-clean`)  
**Harness:** Playwright Chromium via `scripts/ui-flow-eval.js` (Docker `mcr.microsoft.com/playwright:v1.55.0-jammy`)  
**Themes:** light, dark, system × primary pages + repo detail/report

## Verdict

**PASS — 36/36 OK** (0 warn, 0 fail)

| Metric | Value |
|--------|-------|
| Pages × themes | 36 |
| HTTP failures | 0 |
| Bugbot visible text | 0 |
| Low-contrast findings | 0 |
| Print button (reports) | Present near top (`y≈121`) |
| Dashboard charts | 5 live |
| Learning charts | 2 live |

## Prior failures (first eval) — resolved

| Issue | Fix |
|-------|-----|
| Health showed historical `commstech/Bugbot` errors | Display scrub (`displayBrand` / `redactHealthText`) |
| KPI / inset translucent backgrounds → unreadable text | Solid `--rd-surface-2` surfaces |
| Configure dark “missing” secret contrast | Badge + solid `details` / table header backgrounds |
| Print / Save PDF inert (CSP blocked inline `onclick`) | `data-rd-print` + `app.js`; toolbar at top |
| Charts ignored theme changes | Remount on `rd-theme-change` |

## Coverage

Dashboard, Repositories, Scans, Findings, Reports, Learning, System Health, Pre-install, Project groups, Configure, repo detail, repo report — each in light / dark / system.
