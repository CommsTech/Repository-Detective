# Scan policy and brand cleanup — baseline

Recorded at sprint start (parent commit `e7ecd2f`).

## Current commit

- Baseline: `e7ecd2f` — docs(beta): verify repo fleet control layout
- Sprint work builds on this baseline.

## Scan policy behavior (before fix)

| Area | Behavior |
|------|----------|
| Global homelab `config/config.yaml` | `auto_create_issues: true` → issue_policy `all` |
| Private beta example | `auto_create_issues: false` → issue_policy `off` |
| Webhook/scheduled scans | No dry-run flag — filing follows policy |
| Manual scan UI | Checkbox default `checked`; beta note always shown |
| Fleet modal HTML | Hardcoded `checked` on report-only |
| `manual_scan_handlers.go` | Forced `reportOnly=true` when policy off (correct) but UI implied dry-run default for all |
| `repos_control_model.go` | `DefaultReportOnly = !issueFiling` (correct logic, wrong labels) |
| Pre-install audit | Separate runner — no forge issue/PR creation |

## Report-only behavior

- `report_only_dry_run` context → `ApplyReportOnlyDryRunSettings` → monitor_only + issue_policy off
- Scan metadata: `dry_run_report_only: true`, `issue_sync_status: skipped`
- UI tests used `betaTestGlobal()` with `IssuePolicyOff` — masked production filing path

## Repository-Detective references (product-facing audit)

| Location | Classification |
|----------|----------------|
| `ui/templates/*.html` | Clean — uses Repository Detective |
| `ui/layout.html` title | Repository Detective — Inspect. Analyze. Improve. |
| `ui/templates/health.html` | GitHub link → fixed to Gitea |
| `ui/static/graph.js` | `__rdGraph` → `__rdGraph` |
| `ui/configure_model.go` | repository-detective.db display → legacy note |
| `docs/beta/CURSOR_BUGBOT_COMPARISON.md` | Keep — external product comparison |
| `REPOSITORY_DETECTIVE_*` env / `X-Repository-Detective-API-Key` | Keep — legacy compatibility |
| `data/repository-detective.db` path | Keep — migration risk |

## Qdrant

Removed from the product. Historical note: prior builds had optional semantic dedup via Qdrant; current builds use fingerprint + SQLite forge mappings only.

| Setting | Value |
|---------|-------|
| `main.go` default | `qdrant_enabled: false`, `qdrant_collection: cah_findings` |
| `config/config.yaml` | `cah_findings`, disabled |
| `.env.example` (before) | `cah_findings` — corrected to `cah_findings` |
| Required for scans | **No** — optional local learning |

## Manual scan UX gaps (before)

- No advanced options accordion
- No effective policy preflight (severity/confidence/max issues/scanners)
- Report-only checkbox always checked in fleet modal HTML
- Misleading label: "default ON for manual scans" when filing enabled
- No scan policy mode display

## Product repo state (baseline)

- Open operator issues: 1 (#48)
- Active-present findings: 0
- Backlog-control: active (`dogfood_backlog_control_enabled: true`)

## Safety gates confirmed

- No `.env`, `repository-detective` ELF, or `dist/` staged
- All-repo scan not started
- Remediation PRs / LLM sanity gate off by default
