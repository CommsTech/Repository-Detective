# Non-product dry-run next gate

Generated: 2026-06-07

## Decision

```text
ready_for_more_dry_runs
```

## Criteria checklist

| Criterion | Status |
|-----------|--------|
| Both dry-run scans completed | ✅ |
| Issue creation stayed 0 | ✅ |
| No duplicate activity | ✅ |
| Reports actionable | ✅ (medium); N/A (small clean) |
| Scanner failures understood | ✅ (grype, ruff, shellcheck) |
| Noisy findings manageable | ⚠️ graph rules need calibration first |
| Product repo remains clean | ✅ (0 active-present, 1 open #48) |
| Operator explicitly approves limited filing | ❌ not requested |

## Options considered

| Option | Verdict |
|--------|---------|
| `blocked` | No — scans succeeded, guardrails work |
| `repeat_report_only` | Possible but same repos less valuable now |
| `ready_for_limited_issue_filing` | **Rejected** — noise + missing operator approval |
| `ready_for_more_dry_runs` | **Selected** |

## Rationale

Report-only infrastructure is **proven**: findings persist, forge filing blocked, remediation skipped. However, netmapper showed **55% graph-noise findings** and container scanner gaps (grype, ruff, shellcheck). Filing issues now would produce a poor signal-to-noise ratio and erode operator trust.

## Recommended next batch

1. **Calibrate graph rules** for small/medium homelab repos (suppress orphan/island below file threshold).
2. **Fix grype JSON parse** and add ruff + shellcheck to scanner image.
3. **Run one more report-only dry run** on a Go or docs-only medium repo (e.g. a utility with `go.mod`, no open issues).
4. **Operator review** netmapper dry-run report — confirm SEC-EVAL and test-gap findings match expectations.
5. Only after steps 1–4: consider `ready_for_limited_issue_filing` with **critical/high only** on a single pilot repo.

## Explicitly not started

- All-repo fleet scan
- Limited issue filing
- PR auto-remediation
- Modification of non-product Gitea issues

## Remaining blockers for limited issue filing

1. Operator explicit approval
2. Graph noise calibration
3. grype / ruff / shellcheck scanner completeness
4. Second medium-repo dry run in different ecosystem
