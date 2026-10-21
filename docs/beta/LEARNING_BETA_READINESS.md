# Learning engine — beta readiness

Updated: 2026-06-02 (post-learning beta gate)

## Status

| Capability | Status |
|------------|--------|
| Learning events + schema v20 | Validated (16 events + migration) |
| Operator calibration review | Complete — 3 repo-scoped accepts |
| Benchmark fixture harness | `./benchmark/...` tests PASS |
| Report-only validation | 0 issues / 0 PRs |
| LLM sanity gate | Disabled by default |
| Beta packaging | `make beta-release` PASS |

## Operator review outcome

See [calibration-operator-review-report.md](../dogfood-reports/calibration-operator-review-report.md).

- Accepted: netmapper (2 graph rules), commsnet_optimizer (1 graph rule)
- Rejected: commsnet_optimizer ORPHAN-FUNCTION, nextcloud_scripts (needs more evidence)
- Global accepts: 0

## Recommendation

Private beta ready with learning observability. Deploy new image to expose `/ui/learning` on live instance.
