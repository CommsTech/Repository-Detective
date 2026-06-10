# Marketing readiness gate

**Do not start outbound marketing until all required criteria pass.**

## Required before marketing

| # | Criterion | Status |
|---|-----------|--------|
| 1 | Product dogfood: 0 high/critical, near 0 actionable active | **ready** (0 active-present) |
| 2 | Gitea wiki populated | **blocked** (HTTP 500 server-side) |
| 3 | Quick Start works from clean install | **not tested** (external install) |
| 4 | Screenshots captured and current | **partially ready** |
| 5 | Pre-install audit works on 5 known public repos | **not tested** |
| 6 | Container image scanning: ≥1 successful demo | **ready** (alpine:3.20 runner scan 2026-06-10) |
| 7 | Scanner coverage page explains missing tools | **partially ready** (/health tools_summary) |
| 8 | Calibration review page exists | **ready** (/ui/learning) |
| 9 | ≥2 non-product beta scans: useful low-noise reports | **not tested** |
| 10 | No secrets in docs/screenshots | **ready** |
| 11 | Beta docs linked from UI | **partially ready** |
| 12 | Install path tested by non-developer | **not tested** |

## Explicit non-goals (pre-marketing)

- All-repo scanning
- Remediation PR enabled by default
- Runner delegation enabled by default
- LLM sanity gate enabled by default

## Decision

**Not ready for marketing.** Container scan demo complete; wiki and external install remain blockers.

Next gate options: fix Gitea wiki server-side, pre-install 5-repo scale test, 2 non-product beta scans, external clean install.
