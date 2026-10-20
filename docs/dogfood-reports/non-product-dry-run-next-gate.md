# Non-product dry-run next gate

Generated: 2026-06-07 (updated post-calibration)

## Decision

```text
ready_for_more_dry_runs
```

## Criteria checklist

| Criterion | Status |
|-----------|--------|
| Product repo clean | ✅ 0 active-present, 1 open #48 |
| Report-only enforcement | ✅ round 2: 0 issues, 0 PRs |
| Graph noise acceptable | ⚠️ improved — downgraded to info; count reduced 48→23 on netmapper |
| grype states understood | ✅ `scanner_unavailable` vs round 1 `parse_failed` |
| Scanner variance documented | ✅ shellcheck pending image rebuild; ruff adds findings needing gating |
| Round 2 dry run completed | ✅ `commstech/commsnet_optimizer` |
| Operator approves limited filing | ❌ not requested |

## Options considered

| Option | Verdict |
|--------|---------|
| `blocked` | No — calibration improved signal; safety controls hold |
| `repeat_report_only` | Valid if operator wants another repo before wider dry runs |
| `ready_for_limited_issue_filing` | **Not recommended** — operator approval required; ruff gating + image rebuild pending |
| `ready_for_more_dry_runs` | **Selected** |

## Technically ready for operator review?

**Partially.** Report-only safety is proven across three repos. Graph calibration and homelab profile improve signal quality. However:

1. **Limited issue filing not approved** by operator.
2. **Full Docker image rebuild** not verified in this session (shellcheck still missing on live container).
3. **Ruff findings** need severity/confidence gating before Python repo issue filing.
4. **grype** reports `scanner_unavailable` until vuln DB is healthy in runtime image.

## Recommended next batch

1. Rebuild and deploy all-in-one image (`docker-build-verify.sh` + compose up).
2. Add ruff finding severity gating for homelab repos (similar to graph calibration).
3. Run one more report-only dry run on a Python medium repo after image rebuild.
4. Operator review round 2 report; explicit approval required before any `ready_for_limited_issue_filing`.

## Explicitly not started

- All-repo fleet scan
- Limited issue filing in non-product repos
- PR auto-remediation
