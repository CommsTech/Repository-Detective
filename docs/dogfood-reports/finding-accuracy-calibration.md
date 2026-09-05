# Finding accuracy calibration

## What was broken

Graph and template-path findings could appear with high confidence in executive views. Graph disconnected packages could reference test files.

## Changes

- `profile/confidence.go` — additional downgrades for `GRAPH-*` rules and test paths
- `graph/orphans.go` — test files excluded from disconnected package file references
- Executive report separates **actionable** vs **review** counts using confidence gate

## Principles preserved

- No automatic suppression of security findings
- Deterministic scanner evidence remains primary
- Graph rules stay visible but downgraded / report-only via existing profile overrides

## Tests

- `go test ./profile/...`
- `go test ./graph/...`

## Before / after

| Scenario | Before | After |
|----------|--------|-------|
| `foo.go` + `foo_test.go` package | Possible `_test.go` in finding File | Non-test file only |
| GRAPH-ORPHAN-FILE confidence | Base scanner confidence | −12% to −20% heuristic adjustment |
| Executive view | All severities presented equally | Actionable vs review split |

## Calibration recommendations

Existing calibration dashboard unchanged; noisy graph rules remain `ActionReportOnly` under beta profiles via `BetaNoiseRuleIDs`.

## Remaining risks

- ML/AI enrichment not expanded this sprint (intentional)
- Per-rule historical FP rates depend on calibration data volume
