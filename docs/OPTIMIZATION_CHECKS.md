# Optimization checks (Gitea #43)

Repository Detective includes **advisory** optimization signals — not a full profiler.

## Static rules (`analyzers/static.go`)

| Rule ID | Topic |
|---------|--------|
| `OPT-NESTED-LOOP` | Nested loops (algorithmic complexity hint) |
| `OPT-HTTP-CLIENT-PER-CALL` | HTTP client created per call |

Marked `Advisory: true` — verify with profiling before changing production code.

## Health checks

When `enable_performance_checks: true`, health module flags performance footguns (large files, hot paths). See [HEALTH_CHECKS.md](HEALTH_CHECKS.md).

## Not shipped

- Automatic Big-O analysis
- Database N+1 query detector (use language-specific tools)
- Distributed tracing / network chatter analysis

Use findings as triage hints, not automatic defects.
