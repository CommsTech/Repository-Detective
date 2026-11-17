# Dashboard guide

The operator dashboard (`/ui/`) is the command center for backlog, charts, scan health, and scanner coverage.

## Sections

### Hero metrics

- **Critical + high** — open unique findings in those severities
- **Unique backlog** — fingerprint-deduplicated open findings
- **Active / failed scans** — live scan queue signals

### Charts (Chart.js 4.4.1)

| Chart | Data source | Empty state |
|-------|-------------|-------------|
| Severity mix | Open findings by severity | “No open findings by severity” |
| Category radar / bar | Top categories (max 10) | “No category breakdown yet” |
| Scan activity (14d) | Completed scans per day | “No completed scan activity…” |
| Repository risk map | Top repos by open findings | “No repository risk data yet” |

Charts load from embedded JSON (`#rd-dashboard-data`). If Chart.js fails to load, an alert explains that numeric panels below remain valid.

**Note:** Trend chart shows activity only when at least one day has &gt; 0 raw findings from completed scans.

### Executive report strip

Links to fleet report, scan history, and per-repo reports for top risky repositories.

### Platform tables

- **Finding backlog** — deduplicated vs raw detector metrics
- **Scan health** — completed/failed/active and recent failures
- **Scanner coverage** — configured vs installed; **degraded** banner when configured tools are missing from runtime

### Actions panel

Prioritized steps: triage critical/high, failed scans, restore missing scanners, repos needing attention.

## API equivalent

`GET /api/v1/dashboard/summary` returns JSON for integrations (same store queries; no chart bundle).

## Dark theme

Charts use slate grid lines and `#94a3b8` label colors for contrast on the default dark theme ([theme.css](../ui/static/theme.css)).

## Related docs

- [UI.md](UI.md)
- [SCANNER_HEALTH.md](SCANNER_HEALTH.md)
- [REPORTING.md](REPORTING.md)

---

See also [Home](Home).
