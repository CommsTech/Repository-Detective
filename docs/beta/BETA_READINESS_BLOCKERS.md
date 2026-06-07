# Beta readiness blockers

Updated: 2026-06-07 (Beta UX + Release Gate sprint)

## Product baseline

| Check | Status |
|-------|--------|
| Open product issues | 1 (#48 operator task) |
| Active-present findings | 0 |
| Backlog-control | active |
| Report-only dry-run | available |
| Limited issue filing | **NOT approved** |
| All-repo scan | **NOT started** |
| Latest commit | see `git log` |

## Blocker inventory

| Blocker | Status | Notes |
|---------|--------|-------|
| Configure action links broken | **fixed** | Capability rows → `/configure#section` |
| Pre-install audit 404 | **fixed** | 200 + disabled banner; configure link |
| Feature flags not fully testable | **documented** | FEATURE_FLAG_TEST_MATRIX + configure page |
| Staticcheck CI | open | Container network; add CI job |
| Ruff gating Python/homelab | **implemented** | RUFF_GATING_POLICY.md |
| Cursor Bugbot benchmark | **planned** | CURSOR_BUGBOT_BENCHMARK_FIXTURE.md |
| Beta package outside CI | **verified** | `make beta-release` |
| Docker rebuild | **pass** | docker-build-verify.sh ~23m |

## Gate decisions (unchanged)

- Limited issue filing: **blocked**
- All-repo scan: **blocked**
- Non-product issue filing: **blocked**
