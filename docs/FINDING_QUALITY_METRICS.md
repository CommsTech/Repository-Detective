# Finding quality metrics (RD-024)

Local-only usefulness metrics derived from durable SQLite data. **No external telemetry.**

## Endpoint

`GET /api/v1/analytics/finding-quality?window=7d|30d|all`

Also retained: `GET /api/v1/analytics/scan-quality` (all-time dogfood aggregates).

## Windows

| Value | Meaning |
|-------|---------|
| `7d` | Events/findings with timestamps in the last 7 days |
| `30d` | Last 30 days |
| `all` | All durable rows (default) |

## Metric semantics

| Field | Numerator / meaning | Denominator / notes |
|-------|---------------------|---------------------|
| `findings_opened` | Findings with `first_seen_at` in window | — |
| `findings_resolved` | `status=resolved_verified` (windowed on `last_seen_at`) | — |
| `findings_reopened` | Lifecycle events `still_present` / reopen-like | — |
| `findings_by_severity` / `_scanner` / `_category` | Grouped counts of findings in window | — |
| `new_findings` | `first_seen_scan_id == last_seen_scan_id` | — |
| `recurring_findings` | Seen on more than one scan id | — |
| `false_positive_dispositions` | Findings marked `false_positive` | — |
| `false_positive_disposition_rate` | FP dispositions | FP dispositions + resolved. **Not** “false-positive rate” of all findings; unreviewed opens are excluded. |
| `median_time_to_resolution_hours` | Median `(last_seen_at − first_seen_at)` for resolved findings | Requires usable timestamps |
| `scanner_completion_rate` | Completed-like scanner_results | Completed-like + failed-like |
| `policy_outcomes` | Counts from `scans.summary_json` policy fields when present | — |
| `repeated_findings_suppressed` | Findings with `status=suppressed` | Dedup/suppression outcomes, not scanner “misses” |
| `calibration_*` | Recommendation status counts | Local calibration table only |

Every response includes a `definitions` map. Prefer those strings in UIs over inventing security scores or AI accuracy claims.
