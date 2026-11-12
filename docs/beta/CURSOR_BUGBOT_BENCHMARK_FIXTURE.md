# Cursor Bugbot benchmark fixture plan

Companion to [CURSOR_BUGBOT_COMPARISON.md](CURSOR_BUGBOT_COMPARISON.md).

## Fixture repository (private, controlled)

Create `benchmark-fixture/` (not scanned in production) containing:

| Inject | Purpose |
|--------|---------|
| Hardcoded API key string (fake) | Secret detection recall |
| SQL string concat | Injection recall |
| Unused import + line-too-long | Ruff false-positive rate |
| Outdated dependency in requirements.txt | SBOM/grype recall |
| Orphan module (graph) | Graph finding calibration |
| Intentionally safe homelab internal URL | False-positive rate |

## Same PR scenario

One PR modifying 3 files: introduce bug, fix typo, add dependency.

## Runs

1. **Repository Detective** — report-only dry run on fixture + PR scan path
2. **Cursor Bugbot** — same PR on GitHub mirror (when available)

## Metrics (fill after run)

| Metric | RD | Cursor Bugbot |
|--------|----|---------------|
| True positives (injected bugs found) | TBD | TBD |
| False positives (safe code flagged) | TBD | TBD |
| Duplicate findings (repeat scan) | TBD | TBD |
| Learning calibration impact | TBD | N/A |
| Time to first useful report | TBD | TBD |
| Actionability score (operator 1–5) | TBD | TBD |
| Verified closure after fix commit | TBD | TBD |

## Acceptance

- No marketing claims until table is filled.
- Fixture repo never receives auto issue filing.
