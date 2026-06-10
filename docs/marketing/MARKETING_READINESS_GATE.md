# Marketing readiness gate

**Do not start outbound marketing until all required criteria pass.**

## Required before marketing

| # | Criterion | Status |
|---|-----------|--------|
| 1 | Product dogfood: 0 high/critical, near 0 actionable active | ✅ (0 active-present) |
| 2 | Gitea wiki populated | ❌ HTTP 500 server-side |
| 3 | Quick Start works from clean install | ⚠️ needs external install test |
| 4 | Screenshots captured and current | ⚠️ in progress |
| 5 | Pre-install audit works on 5 known public repos | ⚠️ not verified at scale |
| 6 | Container image scanning: ≥1 successful demo | ⚠️ runner-based; opt-in |
| 7 | Scanner coverage page explains missing tools | ⚠️ partial (/health tools_summary) |
| 8 | Calibration review page exists | ✅ /ui/learning |
| 9 | ≥2 non-product beta scans: useful low-noise reports | ❌ not started |
| 10 | No secrets in docs/screenshots | ✅ |
| 11 | Beta docs linked from UI | ⚠️ partial |
| 12 | Install path tested by non-developer | ❌ |

## Explicit non-goals (pre-marketing)

- All-repo scanning
- Remediation PR enabled by default
- Runner delegation enabled by default
- LLM sanity gate enabled by default

## Decision

**Not ready for marketing.** Continue private beta / controlled demo.

Next gate options: fix Gitea wiki, run 2 non-product report-only scans, complete container scan demo with runner.
