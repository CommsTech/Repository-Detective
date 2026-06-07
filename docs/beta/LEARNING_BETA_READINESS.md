# Learning engine — beta readiness

## Shipped in this sprint

| Capability | Status | Notes |
|------------|--------|-------|
| Learning events data model | Ready | Append-only, idempotent |
| Lifecycle event wiring | Ready | Closure verified, FP, scanner health, dry-run, reconcile duplicates |
| Per-repo calibration recommendations | Ready | Evidence threshold; global accept blocked in beta |
| Safe calibration apply | Ready | Operator accept → repo suppression + expiring rule |
| Structural deduplication | Ready | Deterministic hash at persist time |
| Reachability-informed priority | Ready | Test/docs/vendor path heuristics at persist |
| Optional LLM sanity gate | Ready | Disabled by default (`llm_sanity_gate_*`) |
| Learning health UI | Ready | Dashboard section + `/ui/learning` |
| Diff-aware push/PR scans | Existing | `AnalyzeChangedFiles` for webhooks |

## Not enabled (by policy)

- Limited issue filing
- All-repo scan
- Global auto-calibration without multi-repo review
- LLM required for correctness

## Validation

See [learning-engine-validation-report.md](../dogfood-reports/learning-engine-validation-report.md).

## Differentiators vs Cursor Bugbot

Repository Detective emphasizes self-hosted Gitea, evidence-backed lifecycle, full-repo + scheduled scans, report-only dry-runs, SBOM/scanner transparency, per-repo continuous calibration, graph context, and auditable learning events. Cursor Bugbot remains strong in Cursor/GitHub PR review and Autofix workflows — benchmark comparison requires fixture execution.
