# Executive report upgrade

## What was broken

Fleet and repository reports led with raw KPIs and technical findings. No executive decision framework, radar chart on repo reports, or print-optimized layout.

## Files changed

- `ui/executive_report.go` — business executive summary builder
- `ui/templates/executive_summary.html` — shared executive section partial
- `ui/templates/reports.html` — fleet executive report
- `ui/templates/repo_report.html` — repo report + radar + print button
- `ui/static/repo-report-charts.js` — Chart.js radar for repo report
- `ui/static/theme.css` — print stylesheet improvements
- `store/sqlite_queries.go` — category + confidence band queries per repo

## Tests

- `ui/executive_report_test.go`
- `go test ./ui/...`

## Before / after

| Feature | Before | After |
|---------|--------|-------|
| Executive structure | KPI dump | Risk posture, business impact, top risks, recommendation, confidence, scope, limitations |
| Repo radar | Missing | Category radar chart on individual repo report |
| Print/PDF | Basic print hidden nav | Print / Save PDF button; `.rd-no-print` hides chrome |
| Low-confidence findings | Mixed with blockers | Separated actionable vs review counts in executive section |

## Manual verification

1. Open `/ui/reports` — executive summary with recommendation badge
2. Open repo report — radar chart + Print / Save PDF
3. Browser print preview — no sidebar/topbar; charts visible

## Remaining risks

- Server-side PDF generation deferred; browser print-to-PDF is the sprint path
- Fleet report has no radar (repo-level only this sprint)
