# Scan policy, branding, and Qdrant verification

Sprint verification after policy correction and brand cleanup.

## Build and tests

| Check | Result |
|-------|--------|
| `go test ./...` | PASS |
| `staticcheck ./...` | PASS |
| `operator-smoke-test.sh` | PASS |
| `make beta-release` | PASS |

## Policy correction

| Item | Verified |
|------|----------|
| `ResolveScanFilingPolicy` resolver | Unit tests pass |
| Manual scan filing when policy allows + dry run unchecked | `TestManualScanFilesWhenDryRunUnchecked` |
| Manual scan 0 issues when dry run checked | `TestManualScanDryRunCheckedSkipsFiling` |
| Pre-install always report-only | `TestResolveScanFilingPolicyPreinstallAlwaysReportOnly` |
| Beta safe mode enforced when issue_policy off | `TestResolveScanFilingPolicyBetaSafeNeverFiles` |

## Manual scan UX

| Item | Verified |
|------|----------|
| Advanced options accordion | `TestManualScanFormShowsAdvancedOptions` |
| Preflight summary | Template + JS `updateScanPreflight` |
| Scan policy mode in UI | Configure + fleet modal |
| Dry-run checkbox unlocked when filing enabled | Fleet `data-report-only=false` + JS |

## Pre-install

| Item | Verified |
|------|----------|
| Low-confidence graph not install blocker | `TestComputeRiskScoreLowConfidenceGraphNotBlocker` |
| Runner never calls issue manager | By design (separate audit store) |

## Qdrant

| Item | Verified |
|------|----------|
| Disabled by default | `TestDefaultConfigUsesCAHFindings` |
| Collection `cah_findings` | DefaultConfig + docs |
| `.env.example` updated | `cah_findings` |

## Branding

| Item | Verified |
|------|----------|
| Layout title | `TestLayoutTitleContainsRepositoryDetective` |
| Legacy API header | operator smoke test |
| Product UI templates | No Bugbot product name in templates |

## Live homelab (pre-redeploy baseline)

| Item | State |
|------|-------|
| `/ui/repos` | 401 without session (service up) |
| Live revision before redeploy | `e7ecd2f` |
| Product repo open issues | 1 (baseline) |
| All-repo scan | Not started |

## Controlled verification notes

- Repo 31 dry-run: verify after container redeploy with operator session
- Controlled filing test: only on explicit policy + non-production test repo
- Docker rebuild: see docker-build-verify output

## Remaining

- Redeploy live container to pick up sprint commits
- Repo 31 manual dry-run + optional controlled filing test post-deploy
